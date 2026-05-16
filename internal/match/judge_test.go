package match

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cax/fsdtrace/internal/embed"
)

// makeBedrockServer returns an httptest server that responds to any
// /invoke call with the given list of judgment entries serialised
// inside the Anthropic content envelope.
func makeBedrockServer(t *testing.T, entries []judgmentEntry) *httptest.Server {
	t.Helper()
	body, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	resp := map[string]any{
		"content":     []map[string]any{{"type": "text", "text": string(body)}},
		"stop_reason": "end_turn",
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		_, _ = w.Write(payload)
	}))
}

func TestJudge_HappyPath(t *testing.T) {
	ts := makeBedrockServer(t, []judgmentEntry{
		{
			ArtifactID: 7,
			Verdict:    VerdictImplements,
			Confidence: 0.91,
			Evidence:   []Evidence{{File: "x.java", Start: 10, End: 20, Note: "matches POST"}},
			Notes:      "looks good",
		},
	})
	t.Cleanup(ts.Close)

	bedrock, _ := embed.NewClient(ts.URL)
	j := NewJudge(bedrock, "fake-model")

	got, err := j.JudgeFeature(context.Background(),
		FRSnapshot{ID: "FR-1", Title: "x", Description: "y"},
		nil,
		[]ArtifactCandidate{{ID: 7, Kind: "rest_endpoint", Identifier: "POST /x", File: "x.java", StartLine: 10, EndLine: 20}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d matches", len(got))
	}
	m := got[0]
	if m.Verdict != VerdictImplements {
		t.Errorf("verdict = %q", m.Verdict)
	}
	if m.Confidence != 0.91 {
		t.Errorf("confidence = %f", m.Confidence)
	}
	if m.Model != "fake-model" || m.PromptVersion != PromptVersion {
		t.Errorf("model=%q version=%q", m.Model, m.PromptVersion)
	}
	if len(m.Evidence) != 1 || m.Evidence[0].File != "x.java" {
		t.Errorf("evidence = %+v", m.Evidence)
	}
}

func TestJudge_DowngradesNoEvidence(t *testing.T) {
	// Hard rule from SPEC §7.4.
	ts := makeBedrockServer(t, []judgmentEntry{
		{ArtifactID: 1, Verdict: VerdictImplements, Confidence: 0.99, Evidence: nil, Notes: "trust me"},
		{ArtifactID: 2, Verdict: VerdictDrifts, Confidence: 0.7, Evidence: []Evidence{{File: "", Start: 0, End: 0}}, Notes: "blank ev"},
	})
	t.Cleanup(ts.Close)

	bedrock, _ := embed.NewClient(ts.URL)
	j := NewJudge(bedrock, "fake-model")

	got, err := j.JudgeFeature(context.Background(),
		FRSnapshot{ID: "FR-2"},
		nil,
		[]ArtifactCandidate{{ID: 1, File: "a.java"}, {ID: 2, File: "b.java"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range got {
		if m.Verdict != VerdictUnrelated {
			t.Errorf("artifact %d should be unrelated; got %q", m.ArtifactID, m.Verdict)
		}
		if !strings.Contains(m.Notes, "no evidence") {
			t.Errorf("expected downgrade note on artifact %d; got %q", m.ArtifactID, m.Notes)
		}
		if m.Confidence != 0 {
			t.Errorf("downgrade should zero confidence; got %f", m.Confidence)
		}
	}
}

func TestJudge_AbsentArtifactDefaultsToUnrelated(t *testing.T) {
	ts := makeBedrockServer(t, []judgmentEntry{}) // model returns nothing
	t.Cleanup(ts.Close)
	bedrock, _ := embed.NewClient(ts.URL)
	j := NewJudge(bedrock, "fake-model")

	got, err := j.JudgeFeature(context.Background(),
		FRSnapshot{ID: "FR-3"},
		nil,
		[]ArtifactCandidate{{ID: 99}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Verdict != VerdictUnrelated {
		t.Errorf("expected single unrelated match; got %+v", got)
	}
	if !strings.Contains(got[0].Notes, "no judgment returned") {
		t.Errorf("expected explanatory note; got %q", got[0].Notes)
	}
}

func TestJudge_RejectsInvalidVerdict(t *testing.T) {
	ts := makeBedrockServer(t, []judgmentEntry{
		{ArtifactID: 1, Verdict: "supersedes", Confidence: 0.5,
			Evidence: []Evidence{{File: "x", Start: 1, End: 1}}},
	})
	t.Cleanup(ts.Close)
	bedrock, _ := embed.NewClient(ts.URL)
	j := NewJudge(bedrock, "fake-model")

	got, err := j.JudgeFeature(context.Background(),
		FRSnapshot{ID: "FR-4"},
		nil,
		[]ArtifactCandidate{{ID: 1}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Verdict != VerdictUnrelated {
		t.Errorf("invalid verdict should fall back to unrelated; got %q", got[0].Verdict)
	}
}
