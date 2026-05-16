package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"

	"github.com/cax/fsdtrace/internal/db"
)

func setupServer(t *testing.T) (*Server, *db.DB) {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.Open(ctx, filepath.Join(dir, "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.ApplySchema(ctx); err != nil {
		t.Fatal(err)
	}

	exec := func(q string, args ...any) {
		t.Helper()
		err := d.Writer().Submit(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, q, args...)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO features(id,title,description,acceptance,fsd_section,fsd_anchor,run_id,created_at)
	      VALUES('FR-010','Create note','desc','["a","b"]','Notes','FR-010','x',1)`)
	exec(`INSERT INTO features(id,title,description,acceptance,fsd_section,fsd_anchor,run_id,created_at)
	      VALUES('FR-001','Login','desc','["a"]','Auth','FR-001','x',1)`)

	exec(`INSERT INTO code_artifacts(id,kind,identifier,file,start_line,end_line,run_id)
	      VALUES(101,'rest_endpoint','POST /api/v1/notes','NoteController.java',23,30,'x')`)
	exec(`INSERT INTO code_artifacts(id,kind,identifier,file,start_line,end_line,run_id)
	      VALUES(102,'kafka_listener','kafka topics=other-events','Listener.java',8,12,'x')`)

	exec(`INSERT INTO tests(name,file,line,test_kind,target_artifact,run_id)
	      VALUES('createOk','NCTest.java',14,'WebMvcTest',101,'x')`)

	exec(`INSERT INTO matches(run_id,feature_id,artifact_id,verdict,confidence,evidence,notes,model,prompt_version)
	      VALUES('run-1','FR-010',101,'implements',0.9,
	             '[{"file":"NoteController.java","start":23,"end":30,"note":"matches POST"}]',
	             'tested=true test_count=1','m','match-v1')`)
	exec(`INSERT INTO matches(run_id,feature_id,artifact_id,verdict,confidence,evidence,notes,model,prompt_version)
	      VALUES('run-1','FR-001',102,'unrelated',0.0,'[]',
	             'tested=false test_count=0','m','match-v1')`)

	srv, state := NewServer(Config{}) // empty cfg; we inject DB via WithDB
	state.WithDB(d)
	t.Cleanup(state.Close)
	_ = srv // not exercised here; we hit handlers directly
	return state, d
}

func callTool(name string, args map[string]any) mcpgo.CallToolRequest {
	return mcpgo.CallToolRequest{
		Params: mcpgo.CallToolParams{Name: name, Arguments: args},
	}
}

func decodeStructured(t *testing.T, res *mcpgo.CallToolResult, out any) {
	t.Helper()
	if res.IsError {
		t.Fatalf("tool result is error: %+v", res.Content)
	}
	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode structured: %v; raw=%s", err, raw)
	}
}

func TestHandlers_GetFeature_ReturnsCoverageStatusAndMatches(t *testing.T) {
	s, _ := setupServer(t)
	res, err := s.handleGetFeature(context.Background(), callTool("fsd_get_feature",
		map[string]any{"id": "FR-010"}))
	if err != nil {
		t.Fatal(err)
	}
	var det featureDetail
	decodeStructured(t, res, &det)

	if det.ID != "FR-010" || det.CoverageStatus != "implemented" {
		t.Errorf("got id=%q status=%q", det.ID, det.CoverageStatus)
	}
	if len(det.Matches) != 1 || det.Matches[0].ArtifactID != 101 {
		t.Errorf("matches = %+v", det.Matches)
	}
	if !det.Matches[0].Tested || det.Matches[0].TestCount != 1 {
		t.Errorf("tested decoration not parsed: %+v", det.Matches[0])
	}
}

func TestHandlers_GetArtifact_LinksFeatureAndTests(t *testing.T) {
	s, _ := setupServer(t)
	res, err := s.handleGetArtifact(context.Background(), callTool("fsd_get_artifact",
		map[string]any{"id": float64(101)}))
	if err != nil {
		t.Fatal(err)
	}
	var det artifactDetail
	decodeStructured(t, res, &det)

	if det.ID != 101 || det.Identifier != "POST /api/v1/notes" {
		t.Errorf("artifact wrong: %+v", det)
	}
	if len(det.LinkedFeatures) != 1 || det.LinkedFeatures[0].FeatureID != "FR-010" {
		t.Errorf("linked features wrong: %+v", det.LinkedFeatures)
	}
	if len(det.Tests) != 1 || det.Tests[0].Name != "createOk" {
		t.Errorf("tests wrong: %+v", det.Tests)
	}
}

func TestHandlers_ListUnmatched_FiltersBySection(t *testing.T) {
	s, _ := setupServer(t)
	res, err := s.handleListUnmatched(context.Background(), callTool("fsd_list_unmatched",
		map[string]any{"section": "Auth"}))
	if err != nil {
		t.Fatal(err)
	}
	var rows []unmatchedRow
	decodeStructured(t, res, &rows)
	if len(rows) != 1 || rows[0].ID != "FR-001" {
		t.Errorf("expected FR-001 only; got %+v", rows)
	}
}

func TestHandlers_ListOrphans_FiltersByKind(t *testing.T) {
	s, _ := setupServer(t)
	res, err := s.handleListOrphans(context.Background(), callTool("fsd_list_orphans",
		map[string]any{"kind": "kafka_listener"}))
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	decodeStructured(t, res, &rows)
	if len(rows) != 1 || rows[0]["kind"] != "kafka_listener" {
		t.Errorf("expected one kafka listener orphan; got %+v", rows)
	}
}

func TestHandlers_CoverageSummary(t *testing.T) {
	s, _ := setupServer(t)
	res, err := s.handleCoverageSummary(context.Background(), callTool("fsd_coverage_summary",
		map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	var rows []sectionSummary
	decodeStructured(t, res, &rows)
	if len(rows) != 2 {
		t.Errorf("expected 2 sections; got %+v", rows)
	}
	for _, r := range rows {
		switch r.Name {
		case "Notes":
			if r.Implemented != 1 || r.Total != 1 {
				t.Errorf("Notes wrong: %+v", r)
			}
		case "Auth":
			if r.Missing != 1 || r.Total != 1 {
				t.Errorf("Auth wrong: %+v", r)
			}
		}
	}
}

func TestHandlers_DriftReport_EmptyOnFixture(t *testing.T) {
	s, _ := setupServer(t)
	res, err := s.handleDriftReport(context.Background(), callTool("fsd_drift_report",
		map[string]any{}))
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("drift report error: %+v", res)
	}
	// No drifts in fixture — structured content should be null/empty array.
	if res.StructuredContent != nil {
		raw, _ := json.Marshal(res.StructuredContent)
		if string(raw) != "null" && string(raw) != "[]" {
			t.Errorf("expected null or empty drift list; got %s", raw)
		}
	}
}

func TestHandlers_BedrockToolWithoutEnvFailsCleanly(t *testing.T) {
	s, _ := setupServer(t)
	t.Setenv(EnvBedrockBaseURL, "")
	// search_features needs Bedrock; with no env and no override it
	// must return a tool-error rather than panic.
	res, err := s.handleSearchFeatures(context.Background(), callTool("fsd_search_features",
		map[string]any{"query": "anything"}))
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatalf("expected error result; got %+v", res)
	}
	// First text content carries the explanatory message.
	var msg string
	for _, c := range res.Content {
		if tc, ok := c.(mcpgo.TextContent); ok {
			msg = tc.Text
			break
		}
	}
	if !strings.Contains(msg, "BEDROCK_BASE_URL") {
		t.Errorf("expected BEDROCK_BASE_URL hint; got %q", msg)
	}
}
