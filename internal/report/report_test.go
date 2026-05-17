package report

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cax/fsdtrace/internal/db"
)

func setupReportDB(t *testing.T) *db.DB {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.Open(ctx, filepath.Join(dir, "r.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.ApplySchema(ctx); err != nil {
		t.Fatal(err)
	}
	return d
}

// seedReportFixture builds a small but realistic snapshot: two FRs in
// "Notes", one in "Auth"; one rest_endpoint that implements FR-010
// (tested), one drift, one orphan listener.
func seedReportFixture(t *testing.T, d *db.DB) {
	t.Helper()
	ctx := context.Background()
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
	      VALUES('FR-001','Login','desc','["a"]','Auth','FR-001','x',1)`)
	exec(`INSERT INTO features(id,title,description,acceptance,fsd_section,fsd_anchor,run_id,created_at)
	      VALUES('FR-010','Create note','desc','["a"]','Notes','FR-010','x',1)`)
	exec(`INSERT INTO features(id,title,description,acceptance,fsd_section,fsd_anchor,run_id,created_at)
	      VALUES('FR-011','List notes','desc','["a"]','Notes','FR-011','x',1)`)

	exec(`INSERT INTO code_artifacts(id,kind,identifier,file,start_line,end_line,run_id)
	      VALUES(101,'rest_endpoint','POST /api/v1/notes','NoteController.java',23,30,'x')`)
	exec(`INSERT INTO code_artifacts(id,kind,identifier,file,start_line,end_line,run_id)
	      VALUES(102,'rest_endpoint','GET /api/v1/notes/{id}','NoteController.java',16,22,'x')`)
	exec(`INSERT INTO code_artifacts(id,kind,identifier,file,start_line,end_line,run_id)
	      VALUES(103,'kafka_listener','kafka topics=other-events','Listener.java',8,12,'x')`)
	exec(`INSERT INTO code_artifacts(id,kind,identifier,file,start_line,end_line,run_id)
	      VALUES(104,'service_method','com.example.NoteService.create','NoteService.java',9,18,'x')`)
	exec(`INSERT INTO relationships(from_artifact,to_artifact,kind)
	      VALUES(101,104,'calls')`)

	exec(`INSERT INTO tests(name,file,line,test_kind,target_artifact,run_id)
	      VALUES('createNoteOk','NoteControllerTest.java',14,'WebMvcTest',101,'x')`)

	// FR-010 implements via 101 (with evidence + tested), drifts via 102.
	exec(`INSERT INTO matches(run_id,feature_id,artifact_id,verdict,confidence,evidence,notes,model,prompt_version)
	      VALUES('run-1','FR-010',101,'implements',0.9,
	             '[{"file":"NoteController.java","start":23,"end":30,"note":"matches POST"}]',
	             'looks good; tested=true test_count=1','m','match-v1')`)
	exec(`INSERT INTO matches(run_id,feature_id,artifact_id,verdict,confidence,evidence,notes,model,prompt_version)
	      VALUES('run-1','FR-010',102,'drifts',0.5,
	             '[{"file":"NoteController.java","start":16,"end":22,"note":"path mismatch"}]',
	             'similar shape; tested=false test_count=0','m','match-v1')`)
	// FR-001 has no matches → missing.
	// FR-011 has only unrelated → missing.
	exec(`INSERT INTO matches(run_id,feature_id,artifact_id,verdict,confidence,evidence,notes,model,prompt_version)
	      VALUES('run-1','FR-011',103,'unrelated',0.0,'[]',
	             'tested=false test_count=0','m','match-v1')`)
}

func TestLoad_BuildsExpectedReportShape(t *testing.T) {
	ctx := context.Background()
	d := setupReportDB(t)
	seedReportFixture(t, d)

	r, err := Load(ctx, d, "")
	if err != nil {
		t.Fatal(err)
	}
	if r.RunID != "run-1" {
		t.Errorf("RunID = %q", r.RunID)
	}
	if len(r.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d (%+v)", len(r.Sections), r.Sections)
	}

	// Section roll-ups.
	for _, s := range r.Sections {
		switch s.Name {
		case "Auth":
			if s.Total != 1 || s.Missing != 1 || s.Implemented != 0 || s.Drifts != 0 {
				t.Errorf("Auth roll-up wrong: %+v", s)
			}
		case "Notes":
			if s.Total != 2 || s.Implemented != 1 || s.Missing != 1 {
				// FR-010 → implemented; FR-011 → missing (only unrelated)
				t.Errorf("Notes roll-up wrong: %+v", s)
			}
		default:
			t.Errorf("unexpected section %q", s.Name)
		}
	}

	// Drift list.
	if len(r.Drift) != 1 {
		t.Errorf("drift list = %d, want 1", len(r.Drift))
	}
	if len(r.Drift) > 0 && r.Drift[0].FeatureID != "FR-010" {
		t.Errorf("drift FR = %s", r.Drift[0].FeatureID)
	}

	// Orphans: 102 (no implements) and 103 (only unrelated). 101 is implemented so excluded.
	orphanIDs := map[int64]bool{}
	for _, o := range r.Orphans {
		orphanIDs[o.ArtifactID] = true
	}
	if !orphanIDs[102] || !orphanIDs[103] || orphanIDs[101] {
		t.Errorf("orphans wrong: %+v", r.Orphans)
	}

	// Tested decoration round-trips.
	for _, s := range r.Sections {
		for _, fc := range s.Features {
			if fc.ID != "FR-010" {
				continue
			}
			if !fc.TestedAny {
				t.Error("FR-010 should be TestedAny=true")
			}
			for _, m := range fc.Matches {
				if m.ArtifactID == 101 {
					if !m.Tested || m.TestCount != 1 {
						t.Errorf("101 tested decoration not parsed: %+v", m)
					}
					if !strings.Contains(m.Notes, "looks good") || strings.Contains(m.Notes, "tested=") {
						t.Errorf("notes not cleaned: %q", m.Notes)
					}
				}
			}
		}
	}
}

func TestLoadWithOptions_IncludesCallGraphSupportOnlyWhenRequested(t *testing.T) {
	ctx := context.Background()
	d := setupReportDB(t)
	seedReportFixture(t, d)

	surfaceOnly, err := Load(ctx, d, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if got := findMatch(t, surfaceOnly, "FR-010", 101).SupportingArtifacts; len(got) != 0 {
		t.Fatalf("default Load included call graph support: %+v", got)
	}

	withGraph, err := LoadWithOptions(ctx, d, "run-1", Options{IncludeCallGraph: true})
	if err != nil {
		t.Fatal(err)
	}
	if !withGraph.IncludeCallGraph {
		t.Fatal("IncludeCallGraph flag not reflected in report")
	}
	fc := findFeature(t, withGraph, "FR-010")
	if fc.SupportingArtifacts != 1 {
		t.Fatalf("FR-010 SupportingArtifacts = %d, want 1", fc.SupportingArtifacts)
	}
	m := findMatch(t, withGraph, "FR-010", 101)
	if len(m.SupportingArtifacts) != 1 {
		t.Fatalf("supporting artifacts = %+v, want one", m.SupportingArtifacts)
	}
	support := m.SupportingArtifacts[0]
	if support.ArtifactID != 104 || support.Kind != "service_method" ||
		support.Identifier != "com.example.NoteService.create" || support.Depth != 1 {
		t.Fatalf("support artifact mismatch: %+v", support)
	}
}

func TestWriteMarkdown_ProducesThreeAcceptanceFiles(t *testing.T) {
	ctx := context.Background()
	d := setupReportDB(t)
	seedReportFixture(t, d)

	r, err := Load(ctx, d, "")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WriteMarkdown(r, out); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"coverage.md", "drift.md", "orphans.md"} {
		path := filepath.Join(out, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Errorf("%s is empty", name)
		}
	}

	cov := mustReadFile(t, filepath.Join(out, "coverage.md"))
	if !strings.Contains(cov, "FR-010") || !strings.Contains(cov, "IMPLEMENTED") {
		t.Errorf("coverage.md missing FR-010 IMPLEMENTED row:\n%s", cov)
	}
	if !strings.Contains(cov, "Roll-up") {
		t.Error("coverage.md missing roll-up table")
	}

	drift := mustReadFile(t, filepath.Join(out, "drift.md"))
	if !strings.Contains(drift, "FR-010") || !strings.Contains(drift, "GET /api/v1/notes/{id}") {
		t.Errorf("drift.md content wrong:\n%s", drift)
	}

	orph := mustReadFile(t, filepath.Join(out, "orphans.md"))
	if !strings.Contains(orph, "kafka topics=other-events") {
		t.Errorf("orphans.md missing kafka listener:\n%s", orph)
	}
}

func TestWriteMarkdown_IncludesCallGraphSupportWhenRequested(t *testing.T) {
	ctx := context.Background()
	d := setupReportDB(t)
	seedReportFixture(t, d)

	r, err := LoadWithOptions(ctx, d, "run-1", Options{IncludeCallGraph: true})
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WriteMarkdown(r, out); err != nil {
		t.Fatal(err)
	}
	cov := mustReadFile(t, filepath.Join(out, "coverage.md"))
	for _, want := range []string{
		"SCIP Support",
		"SCIP support for `POST /api/v1/notes`",
		"com.example.NoteService.create",
	} {
		if !strings.Contains(cov, want) {
			t.Errorf("coverage.md missing %q:\n%s", want, cov)
		}
	}
}

func TestWriteCSV_ProducesParseableMatchesCSV(t *testing.T) {
	ctx := context.Background()
	d := setupReportDB(t)
	seedReportFixture(t, d)

	r, err := Load(ctx, d, "")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WriteCSV(r, out); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(filepath.Join(out, "matches.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) < 4 {
		t.Errorf("expected header + ≥3 match rows, got %d", len(rows))
	}
	if rows[0][0] != "run_id" {
		t.Errorf("header[0] = %q", rows[0][0])
	}
}

func TestWriteJSON_RoundTrips(t *testing.T) {
	ctx := context.Background()
	d := setupReportDB(t)
	seedReportFixture(t, d)

	r, err := Load(ctx, d, "")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WriteJSON(r, out); err != nil {
		t.Fatal(err)
	}
	raw := mustReadFile(t, filepath.Join(out, "report.json"))
	var got Report
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	if got.RunID != r.RunID || len(got.Sections) != len(r.Sections) {
		t.Errorf("round-trip mismatch: %+v", got)
	}
}

func TestWriteHTML_EmitsCollapsibleSections(t *testing.T) {
	ctx := context.Background()
	d := setupReportDB(t)
	seedReportFixture(t, d)
	r, err := Load(ctx, d, "")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WriteHTML(r, out); err != nil {
		t.Fatal(err)
	}
	html := mustReadFile(t, filepath.Join(out, "index.html"))
	for _, want := range []string{
		"<title>fsdtrace report",
		"<details>", "<summary>",
		"FR-010", "POST /api/v1/notes",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("html missing %q", want)
		}
	}
}

func TestWriteHTML_IncludesCallGraphSVGWhenRequested(t *testing.T) {
	ctx := context.Background()
	d := setupReportDB(t)
	seedReportFixture(t, d)
	r, err := LoadWithOptions(ctx, d, "run-1", Options{IncludeCallGraph: true})
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := WriteHTML(r, out); err != nil {
		t.Fatal(err)
	}
	html := mustReadFile(t, filepath.Join(out, "index.html"))
	for _, want := range []string{
		`<svg class="scip-graph"`,
		`SCIP call graph support for POST /api/v1/notes`,
		`data-artifact-id="101"`,
		`data-from-artifact-id="101" data-to-artifact-id="104"`,
		`com.example.NoteService.create`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("html missing %q", want)
		}
	}
}

func TestParseNotesDecorations(t *testing.T) {
	cases := []struct {
		in    string
		want  string
		ok    bool
		count int
	}{
		{"foo; tested=true test_count=2", "foo", true, 2},
		{"tested=false test_count=0", "", false, 0},
		{"only prose", "only prose", false, 0},
		{"a; b; tested=true test_count=5", "a; b", true, 5},
	}
	for _, c := range cases {
		got, ok, n := parseNotesDecorations(c.in)
		if got != c.want || ok != c.ok || n != c.count {
			t.Errorf("parseNotesDecorations(%q) = (%q, %v, %d); want (%q, %v, %d)",
				c.in, got, ok, n, c.want, c.ok, c.count)
		}
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func findFeature(t *testing.T, r *Report, featureID string) FeatureCoverage {
	t.Helper()
	for _, s := range r.Sections {
		for _, f := range s.Features {
			if f.ID == featureID {
				return f
			}
		}
	}
	t.Fatalf("feature %s not found", featureID)
	return FeatureCoverage{}
}

func findMatch(t *testing.T, r *Report, featureID string, artifactID int64) MatchRow {
	t.Helper()
	f := findFeature(t, r, featureID)
	for _, m := range f.Matches {
		if m.ArtifactID == artifactID {
			return m
		}
	}
	t.Fatalf("match %s/%d not found", featureID, artifactID)
	return MatchRow{}
}
