package match

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
)

type fakeEmbedder struct{ calls atomic.Int64 }

func (f *fakeEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	f.calls.Add(1)
	v := make([]float32, db.EmbeddingDim)
	v[0] = float32(len(text))
	return v, nil
}
func (f *fakeEmbedder) Model() string { return "fake-embed" }

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
	pipe := NewPipeline(d, bedrock, &fakeEmbedder{}, WithTopK(10))

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
	pipe := NewPipeline(d, bedrock, &fakeEmbedder{})

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
