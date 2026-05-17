package match

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/llm"
)

type fakeEmbedder struct{ calls atomic.Int64 }

func (f *fakeEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	f.calls.Add(1)
	v := make([]float32, db.EmbeddingDim)
	v[0] = float32(len(text))
	return v, nil
}
func (f *fakeEmbedder) Model() string { return "fake-embed" }

type batchGenerator struct {
	calls     [][]int64
	failAbove int
}

func (g *batchGenerator) Generate(_ context.Context, req llm.GenerateRequest) (string, error) {
	ids := extractCandidateIDs(req.User)
	g.calls = append(g.calls, ids)
	if g.failAbove > 0 && len(ids) > g.failAbove {
		return "", &llm.IncompleteError{Provider: "test", Reason: "max_output_tokens"}
	}
	entries := make([]judgmentEntry, 0, len(ids))
	for _, id := range ids {
		if id%2 == 0 {
			continue // omitted candidates are treated as unrelated.
		}
		entries = append(entries, judgmentEntry{
			ArtifactID: id,
			Verdict:    VerdictImplements,
			Confidence: 0.9,
			Evidence:   []Evidence{{File: "Candidate.java", Start: 1, End: 2}},
		})
	}
	out, _ := json.Marshal(entries)
	return string(out), nil
}

type concurrentGenerator struct {
	active   atomic.Int64
	max      atomic.Int64
	calls    atomic.Int64
	delay    time.Duration
	response string
}

func (g *concurrentGenerator) Generate(ctx context.Context, _ llm.GenerateRequest) (string, error) {
	current := g.active.Add(1)
	defer g.active.Add(-1)
	g.calls.Add(1)
	for {
		maxActive := g.max.Load()
		if current <= maxActive || g.max.CompareAndSwap(maxActive, current) {
			break
		}
	}
	timer := time.NewTimer(g.delay)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
		return "", ctx.Err()
	}
	if g.response != "" {
		return g.response, nil
	}
	return "[]", nil
}

type failingGenerator struct{ calls atomic.Int64 }

func (g *failingGenerator) Generate(_ context.Context, _ llm.GenerateRequest) (string, error) {
	g.calls.Add(1)
	return "", fmt.Errorf("unexpected judge call")
}

func extractCandidateIDs(prompt string) []int64 {
	re := regexp.MustCompile(`\[(\d+)\] kind=`)
	matches := re.FindAllStringSubmatch(prompt, -1)
	ids := make([]int64, 0, len(matches))
	for _, m := range matches {
		ids = append(ids, atoi64(m[1]))
	}
	return ids
}

func setupPipelineDB(t *testing.T) *db.DB {
	t.Helper()
	ctx := context.Background()
	dir := t.TempDir()
	d, err := db.Open(ctx, filepath.Join(dir, "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := d.ApplySchema(ctx); err != nil {
		t.Fatal(err)
	}
	return d
}

func seedFeature(t *testing.T, d *db.DB, id, title, desc string, accept []string) {
	t.Helper()
	ctx := context.Background()
	acc, _ := json.Marshal(accept)
	err := d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO features(id, title, description, acceptance, fsd_section, fsd_anchor, run_id, created_at)
			 VALUES (?,?,?,?,?,?,?,strftime('%s','now'))`,
			id, title, desc, string(acc), "Notes", id, "test-run")
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	v := make([]float32, db.EmbeddingDim)
	v[0] = 1
	err = d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO feature_vec(rowid, embedding) VALUES (?, ?)`,
			fnv64(id), db.PackFloat32(v))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func seedArtifact(t *testing.T, d *db.DB, kind, identifier, file string, start, end int) int64 {
	t.Helper()
	ctx := context.Background()
	var id int64
	err := d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO code_artifacts(kind, identifier, file, start_line, end_line, run_id)
			 VALUES (?,?,?,?,?,'test-run')`,
			kind, identifier, file, start, end)
		if err != nil {
			return err
		}
		id, err = res.LastInsertId()
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	v := make([]float32, db.EmbeddingDim)
	v[0] = 1
	err = d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO artifact_vec(rowid, embedding) VALUES (?, ?)`,
			id, db.PackFloat32(v))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func seedTest(t *testing.T, d *db.DB, name, file string, line int, target int64) {
	t.Helper()
	ctx := context.Background()
	err := d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO tests(name, file, line, test_kind, target_artifact, run_id)
			 VALUES (?,?,?,?,?,'test-run')`,
			name, file, line, "WebMvcTest", target)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// fakeJudgment server inspects the request body for "[1] kind=X"
// markers and returns one judgment per candidate. REST endpoints whose
// identifier contains "/api/v1/notes" become "implements"; everything
// else "unrelated". Always emits real evidence so the downgrade rule
// doesn't fire spuriously.
func fakeJudgmentServer(t *testing.T, calls *atomic.Int64) *httptest.Server {
	candidateRe := regexp.MustCompile(`\[(\d+)\] kind=(\w+) identifier="([^"]+)" file=([^:]+):(\d+)-(\d+)`)
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		calls.Add(1)
		// Drop one layer of JSON escaping by extracting the user message.
		var env struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(raw, &env); err != nil {
			t.Errorf("decode envelope: %v", err)
		}
		var userText string
		if len(env.Messages) > 0 {
			userText = env.Messages[0].Content
		}
		matches := candidateRe.FindAllStringSubmatch(userText, -1)
		entries := make([]judgmentEntry, 0, len(matches))
		for _, m := range matches {
			id := atoi64(m[1])
			kind := m[2]
			ident := m[3]
			file := m[4]
			start := atoi(m[5])
			end := atoi(m[6])

			verdict := VerdictUnrelated
			conf := 0.0
			var ev []Evidence
			if kind == "rest_endpoint" && strings.Contains(ident, "/api/v1/notes") {
				verdict = VerdictImplements
				conf = 0.9
				ev = []Evidence{{File: file, Start: start, End: end, Note: "matches POST /api/v1/notes"}}
			}
			entries = append(entries, judgmentEntry{
				ArtifactID: id, Verdict: verdict, Confidence: conf, Evidence: ev,
			})
		}
		out, _ := json.Marshal(entries)
		resp, _ := json.Marshal(map[string]any{
			"content":     []map[string]any{{"type": "text", "text": string(out)}},
			"stop_reason": "end_turn",
		})
		_, _ = w.Write(resp)
	}))
}

func TestPipelineJudgeCandidatesBatchesAndTreatsOmissionsAsUnrelated(t *testing.T) {
	gen := &batchGenerator{}
	pipe := NewPipeline(nil, gen, &fakeEmbedder{},
		WithJudgmentModel("test-model"),
		WithJudgmentBatchSize(2),
	)
	candidates := []ArtifactCandidate{
		{ID: 1, Kind: "rest_endpoint", Identifier: "GET /a", File: "A.java", StartLine: 1, EndLine: 2},
		{ID: 2, Kind: "rest_endpoint", Identifier: "GET /b", File: "B.java", StartLine: 1, EndLine: 2},
		{ID: 3, Kind: "rest_endpoint", Identifier: "GET /c", File: "C.java", StartLine: 1, EndLine: 2},
		{ID: 4, Kind: "rest_endpoint", Identifier: "GET /d", File: "D.java", StartLine: 1, EndLine: 2},
		{ID: 5, Kind: "rest_endpoint", Identifier: "GET /e", File: "E.java", StartLine: 1, EndLine: 2},
	}

	matches, err := pipe.judgeCandidates(context.Background(), FRSnapshot{ID: "FR-1"}, nil, candidates)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(gen.calls), 3; got != want {
		t.Fatalf("judge calls = %d, want %d", got, want)
	}
	if got := len(gen.calls[0]); got != 2 {
		t.Fatalf("first batch size = %d, want 2", got)
	}
	if len(matches) != len(candidates) {
		t.Fatalf("matches = %d, want %d", len(matches), len(candidates))
	}
	for _, m := range matches {
		want := VerdictImplements
		if m.ArtifactID%2 == 0 {
			want = VerdictUnrelated
		}
		if m.Verdict != want {
			t.Fatalf("artifact %d verdict = %q, want %q", m.ArtifactID, m.Verdict, want)
		}
	}
}

func TestPipelineJudgeCandidatesRetriesIncompleteWithSmallerBatches(t *testing.T) {
	gen := &batchGenerator{failAbove: 2}
	pipe := NewPipeline(nil, gen, &fakeEmbedder{},
		WithJudgmentModel("test-model"),
		WithJudgmentBatchSize(4),
	)
	candidates := []ArtifactCandidate{
		{ID: 1, Kind: "rest_endpoint", Identifier: "GET /a", File: "A.java", StartLine: 1, EndLine: 2},
		{ID: 2, Kind: "rest_endpoint", Identifier: "GET /b", File: "B.java", StartLine: 1, EndLine: 2},
		{ID: 3, Kind: "rest_endpoint", Identifier: "GET /c", File: "C.java", StartLine: 1, EndLine: 2},
		{ID: 4, Kind: "rest_endpoint", Identifier: "GET /d", File: "D.java", StartLine: 1, EndLine: 2},
	}

	matches, err := pipe.judgeCandidates(context.Background(), FRSnapshot{ID: "FR-1"}, nil, candidates)
	if err != nil {
		t.Fatal(err)
	}
	gotSizes := make([]int, 0, len(gen.calls))
	for _, call := range gen.calls {
		gotSizes = append(gotSizes, len(call))
	}
	wantSizes := []int{4, 2, 2}
	if !intSlicesEqual(gotSizes, wantSizes) {
		t.Fatalf("batch sizes = %v, want %v", gotSizes, wantSizes)
	}
	if len(matches) != len(candidates) {
		t.Fatalf("matches = %d, want %d", len(matches), len(candidates))
	}
}

func TestPipelineMatchAllHonorsConcurrency(t *testing.T) {
	ctx := context.Background()
	d := setupPipelineDB(t)
	for i := 1; i <= 4; i++ {
		id := fmt.Sprintf("FR-%03d", i)
		seedFeature(t, d, id, "List owners", "User can list owners", []string{"GET /owners"})
	}
	_ = seedArtifact(t, d, "rest_endpoint", "GET /owners", "OwnerController.java", 10, 20)

	gen := &concurrentGenerator{delay: 50 * time.Millisecond}
	pipe := NewPipeline(d, gen, &fakeEmbedder{},
		WithTopK(1),
		WithMatchConcurrency(2),
	)

	summary, err := pipe.MatchAll(ctx, "match-concurrent", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := gen.calls.Load(); got != 4 {
		t.Fatalf("judge calls = %d, want 4", got)
	}
	if got := gen.max.Load(); got < 2 {
		t.Fatalf("max concurrent judge calls = %d, want at least 2", got)
	}
	if got := gen.max.Load(); got > 2 {
		t.Fatalf("max concurrent judge calls = %d, want no more than 2", got)
	}
	if got, want := summary.TotalMatches, 4; got != want {
		t.Fatalf("total matches = %d, want %d", got, want)
	}
}

func TestPipelineResumeSkipsFeaturesWithExistingMatches(t *testing.T) {
	ctx := context.Background()
	d := setupPipelineDB(t)
	seedFeature(t, d, "FR-010", "Create owner", "User creates an owner", []string{"POST /owners"})
	_ = seedArtifact(t, d, "rest_endpoint", "POST /owners", "OwnerController.java", 10, 20)

	gen := &batchGenerator{}
	pipe := NewPipeline(d, gen, &fakeEmbedder{}, WithTopK(1))
	if _, err := pipe.MatchAll(ctx, "resume-match", nil); err != nil {
		t.Fatal(err)
	}
	if len(gen.calls) != 1 {
		t.Fatalf("initial judge calls = %d, want 1", len(gen.calls))
	}

	failing := &failingGenerator{}
	progress := make([]int, 0, 1)
	resumePipe := NewPipeline(d, failing, &fakeEmbedder{},
		WithTopK(1),
		WithResume(true),
		WithProgress(func(done, total int) {
			if total != 1 {
				t.Errorf("progress total = %d, want 1", total)
			}
			progress = append(progress, done)
		}),
	)
	summary, err := resumePipe.MatchAll(ctx, "resume-match", nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.SkippedFeatures != 1 || summary.TotalMatches != 0 {
		t.Fatalf("resume summary = %+v, want one skipped feature and no new matches", summary)
	}
	if got := failing.calls.Load(); got != 0 {
		t.Fatalf("resume made %d judge calls", got)
	}
	if len(progress) != 1 || progress[0] != 1 {
		t.Fatalf("progress callbacks = %v, want [1]", progress)
	}
}

func TestPipelineMatchAll_PopulatesMatchesWithEvidenceAndTestDecoration(t *testing.T) {
	ctx := context.Background()
	d := setupPipelineDB(t)

	// Seed: one FR + three artifacts, one of which is a tested REST endpoint.
	seedFeature(t, d, "FR-010",
		"Create a note",
		"User creates a note via POST /api/v1/notes",
		[]string{"POST `/api/v1/notes` with {title,body} returns 201"},
	)
	postID := seedArtifact(t, d, "rest_endpoint", "POST /api/v1/notes", "NoteController.java", 23, 30)
	getID := seedArtifact(t, d, "rest_endpoint", "GET /api/v1/notes/{id}", "NoteController.java", 16, 22)
	_ = seedArtifact(t, d, "kafka_listener", "kafka topics=notes-events groupId=x", "Listener.java", 8, 12)
	seedTest(t, d, "createNoteOk", "NoteControllerTest.java", 14, postID)

	var calls atomic.Int64
	ts := fakeJudgmentServer(t, &calls)
	t.Cleanup(ts.Close)
	bedrock, err := embed.NewClient(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	pipe := NewPipeline(d, llm.NewBedrockGenerator(bedrock), &fakeEmbedder{}, WithTopK(10))

	summary, err := pipe.MatchAll(ctx, "match-test", nil)
	if err != nil {
		t.Fatalf("MatchAll: %v", err)
	}
	if summary.TotalMatches == 0 {
		t.Fatal("no matches written")
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 judge call (one FR), got %d", calls.Load())
	}

	// Verify a row exists for the POST endpoint with verdict=implements
	// and that tested/test_count made it into notes.
	var (
		verdict, notes, ev string
		conf               float64
	)
	err = d.SQL().QueryRowContext(ctx,
		`SELECT verdict, confidence, evidence, notes FROM matches
		   WHERE feature_id = ? AND artifact_id = ?`, "FR-010", postID,
	).Scan(&verdict, &conf, &ev, &notes)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != VerdictImplements {
		t.Errorf("verdict = %q", verdict)
	}
	if conf < 0.5 {
		t.Errorf("confidence too low: %f", conf)
	}
	if !strings.Contains(notes, "tested=true") || !strings.Contains(notes, "test_count=1") {
		t.Errorf("notes missing test decoration: %q", notes)
	}
	if !strings.Contains(ev, "NoteController.java") {
		t.Errorf("evidence missing source file: %q", ev)
	}

	// Verify the GET endpoint got a row too (likely "unrelated").
	var verdictGet string
	err = d.SQL().QueryRowContext(ctx,
		`SELECT verdict FROM matches WHERE feature_id = ? AND artifact_id = ?`,
		"FR-010", getID).Scan(&verdictGet)
	if err != nil {
		t.Errorf("expected a row for GET endpoint: %v", err)
	}

	// Re-run idempotency.
	if _, err := pipe.MatchAll(ctx, "match-test", nil); err != nil {
		t.Fatal(err)
	}
	var n int
	err = d.SQL().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM matches WHERE run_id = 'match-test'`).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != summary.TotalMatches {
		t.Errorf("re-run grew matches: %d → %d", summary.TotalMatches, n)
	}
}

// TestRejudgeDrifts_PromotesAndPersists exercises the second-pass
// rejudge against drifts rows already in the DB.
func TestRejudgeDrifts_PromotesAndPersists(t *testing.T) {
	ctx := context.Background()
	d := setupPipelineDB(t)
	seedFeature(t, d, "FR-010", "Create note",
		"User creates a note via POST /api/v1/notes",
		[]string{"POST `/api/v1/notes`"})
	postID := seedArtifact(t, d, "rest_endpoint", "POST /api/v1/notes",
		"NoteController.java", 23, 30)

	// Seed a pre-existing drifts row so RejudgeDrifts has something to lift.
	insertMatch := func(verdict string) {
		err := d.Writer().Submit(ctx, func(tx *sql.Tx) error {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO matches(run_id,feature_id,artifact_id,verdict,confidence,
				                    evidence,notes,model,prompt_version)
				VALUES('rj-run','FR-010',?,?,0.5,
				       '[{"file":"NoteController.java","start":23,"end":30,"note":"first pass"}]',
				       'tested=false test_count=0','sonnet','match-v1')`,
				postID, verdict)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	insertMatch("drifts")

	// Opus mock: any rest_endpoint with a /notes path becomes "implements".
	opusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var env struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(raw, &env)
		user := ""
		if len(env.Messages) > 0 {
			user = env.Messages[0].Content
		}
		idRe := regexp.MustCompile(`\[(\d+)\]`)
		fileRe := regexp.MustCompile(`file=([^:]+):(\d+)-(\d+)`)
		ids := idRe.FindAllStringSubmatch(user, -1)
		entries := make([]judgmentEntry, 0, len(ids))
		for _, m := range ids {
			id := atoi64(m[1])
			fm := fileRe.FindStringSubmatch(user)
			file, start, end := "x.java", 1, 2
			if len(fm) == 4 {
				file = fm[1]
				start = atoi(fm[2])
				end = atoi(fm[3])
			}
			entries = append(entries, judgmentEntry{
				ArtifactID: id, Verdict: VerdictImplements, Confidence: 0.95,
				Evidence: []Evidence{{File: file, Start: start, End: end, Note: "Opus says yes"}},
				Notes:    "rejudged",
			})
		}
		body, _ := json.Marshal(entries)
		resp, _ := json.Marshal(map[string]any{
			"content":     []map[string]any{{"type": "text", "text": string(body)}},
			"stop_reason": "end_turn",
		})
		_, _ = w.Write(resp)
	}))
	t.Cleanup(opusServer.Close)

	bedrock, err := embed.NewClient(opusServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	pipe := NewPipeline(d, llm.NewBedrockGenerator(bedrock), &fakeEmbedder{})

	rj, err := pipe.RejudgeDrifts(ctx, "rj-run", "anthropic.claude-opus-4-v2:0")
	if err != nil {
		t.Fatal(err)
	}
	if rj.Total != 1 || rj.PromotedToImplements != 1 {
		t.Errorf("expected 1 promoted to implements; got %+v", rj)
	}

	// Row updated in place: same run_id + feature_id + artifact_id.
	var verdict, model, notes string
	var conf float64
	err = d.SQL().QueryRowContext(ctx,
		`SELECT verdict, confidence, model, notes FROM matches
		   WHERE run_id='rj-run' AND feature_id='FR-010' AND artifact_id=?`,
		postID).Scan(&verdict, &conf, &model, &notes)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != VerdictImplements {
		t.Errorf("verdict = %q (want implements)", verdict)
	}
	if model != "anthropic.claude-opus-4-v2:0" {
		t.Errorf("model not updated: %q", model)
	}
	if !strings.Contains(notes, "rejudged") {
		t.Errorf("notes = %q (want 'rejudged' substring)", notes)
	}
	if conf < 0.9 {
		t.Errorf("confidence not raised: %f", conf)
	}
}

func atoi64(s string) int64 {
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int64(r-'0')
	}
	return n
}

func atoi(s string) int { return int(atoi64(s)) }

func intSlicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func fnv64(s string) int64 {
	const offset = 1469598103934665603
	const prime = 1099511628211
	h := uint64(offset)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	return int64(h & 0x7fffffffffffffff)
}
