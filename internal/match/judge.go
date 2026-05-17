package match

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/cax/fsdtrace/internal/llm"
)

// Judge calls a configured model with one FR and its candidate artifacts
// and returns the per-candidate verdicts. Enforces the SPEC §7.4 hard
// rule: any verdict missing concrete evidence is downgraded to
// "unrelated" before the result reaches callers.
type Judge struct {
	generator llm.Generator
	model     string
	maxTok    int
}

// JudgeOption configures a Judge.
type JudgeOption func(*Judge)

// WithJudgeMaxTokens overrides the per-call max output tokens.
func WithJudgeMaxTokens(n int) JudgeOption {
	return func(j *Judge) {
		if n > 0 {
			j.maxTok = n
		}
	}
}

// NewJudge builds a Judge.
func NewJudge(generator llm.Generator, model string, opts ...JudgeOption) *Judge {
	if model == "" {
		model = DefaultJudgmentModel
	}
	j := &Judge{generator: generator, model: model, maxTok: 4096}
	for _, o := range opts {
		o(j)
	}
	return j
}

// Model returns the model id this judge will invoke.
func (j *Judge) Model() string { return j.model }

// JudgeFeature returns one Match per candidate. Candidates not present
// in the model's response default to verdict=unrelated with no evidence.
func (j *Judge) JudgeFeature(ctx context.Context, fr FRSnapshot, anchors []Anchor, candidates []ArtifactCandidate) ([]Match, error) {
	if len(candidates) == 0 {
		return nil, nil
	}

	text, err := j.generator.Generate(ctx, llm.GenerateRequest{
		Model:     j.model,
		System:    JudgmentSystem,
		User:      BuildJudgmentUser(fr, anchors, candidates),
		MaxTokens: j.maxTok,
	})
	if err != nil {
		return nil, fmt.Errorf("judge invoke: %w", err)
	}
	if text == "" {
		return nil, errors.New("judge: empty response")
	}

	var raw []judgmentEntry
	if err := json.Unmarshal([]byte(stripFence(text)), &raw); err != nil {
		return nil, fmt.Errorf("judge: parse response: %w; body=%q",
			err, truncate(text, 256))
	}

	byID := map[int64]judgmentEntry{}
	for _, e := range raw {
		byID[e.ArtifactID] = e
	}

	out := make([]Match, 0, len(candidates))
	for _, c := range candidates {
		m := Match{
			FeatureID:     fr.ID,
			ArtifactID:    c.ID,
			Verdict:       VerdictUnrelated,
			Confidence:    0,
			Evidence:      nil,
			Notes:         "",
			Model:         j.model,
			PromptVersion: PromptVersion,
		}
		if e, ok := byID[c.ID]; ok {
			m = applyJudgment(m, e)
		} else {
			m.Notes = "no judgment returned by model"
		}
		out = append(out, enforceEvidenceRule(m))
	}
	return out, nil
}

// applyJudgment copies validated fields from the LLM response. Invalid
// verdicts default to "unrelated".
func applyJudgment(m Match, e judgmentEntry) Match {
	if IsValidVerdict(e.Verdict) {
		m.Verdict = e.Verdict
	}
	if e.Confidence < 0 {
		e.Confidence = 0
	}
	if e.Confidence > 1 {
		e.Confidence = 1
	}
	m.Confidence = e.Confidence
	for _, ev := range e.Evidence {
		if ev.File == "" || ev.Start <= 0 || ev.End < ev.Start {
			continue
		}
		m.Evidence = append(m.Evidence, ev)
	}
	m.Notes = strings.TrimSpace(e.Notes)
	return m
}

// enforceEvidenceRule downgrades any non-unrelated verdict that has no
// usable evidence. SPEC §7.4 hard rule.
func enforceEvidenceRule(m Match) Match {
	if m.Verdict == VerdictUnrelated {
		return m
	}
	if len(m.Evidence) == 0 {
		downgradeNote := fmt.Sprintf("downgraded from %q: no evidence", m.Verdict)
		m.Verdict = VerdictUnrelated
		m.Confidence = 0
		if m.Notes != "" {
			m.Notes += "; "
		}
		m.Notes += downgradeNote
	}
	return m
}

type judgmentEntry struct {
	ArtifactID int64      `json:"artifact_id"`
	Verdict    string     `json:"verdict"`
	Confidence float64    `json:"confidence"`
	Evidence   []Evidence `json:"evidence"`
	Notes      string     `json:"notes"`
}

func stripFence(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
