package match

import (
	"fmt"
	"strings"
)

// PromptVersion is the version stamp written into matches.prompt_version.
// HARD CONSTRAINT (CLAUDE.md): bump whenever the prompt template changes.
const PromptVersion = "match-v2"

// DefaultJudgmentModel is used when provider=bedrock.
const DefaultJudgmentModel = "anthropic.claude-sonnet-4-v2:0"

// JudgmentSystem is the system prompt for the judge call.
const JudgmentSystem = `You are a senior reviewer judging whether each candidate code artifact implements a Functional Requirement (FR).

Return EXACTLY one JSON array — no prose, no markdown fence — with one object only for candidates that are "implements" or "drifts", using the candidate's "id" field. Omit unrelated candidates; omitted candidates are treated as "unrelated" by the caller:

[
  {
    "artifact_id": int,                    // matches one of the candidate ids
    "verdict":     "implements" | "drifts",
    "confidence":  number,                 // 0..1
    "evidence":    [                       // REQUIRED for implements/drifts
      {"file": string, "start": int, "end": int, "note": string}
    ],
    "notes":       string                  // optional short prose
  }
]

Verdict rules:
- "implements" — the artifact concretely satisfies an acceptance criterion of the FR.
- "drifts"     — the artifact does something similar (same surface area, different behaviour) and likely needs to converge with the FR.
- Omit artifacts with no meaningful overlap. Do not emit "unrelated" objects.

Evidence rules (HARD):
- Every "implements" or "drifts" verdict MUST include at least one evidence object whose file/start/end refer to the actual lines you cite.
- If you cannot ground a verdict in concrete lines, omit that artifact.`

// BuildJudgmentUser composes the user-side prompt for one FR.
func BuildJudgmentUser(fr FRSnapshot, anchors []Anchor, candidates []ArtifactCandidate) string {
	var b strings.Builder
	fmt.Fprintf(&b, "FR: %s — %s\n", fr.ID, fr.Title)
	if fr.Section != "" {
		fmt.Fprintf(&b, "Section: %s\n", fr.Section)
	}
	fmt.Fprintf(&b, "\nDescription:\n%s\n", fr.Description)
	if len(fr.Acceptance) > 0 {
		b.WriteString("\nAcceptance criteria:\n")
		for _, a := range fr.Acceptance {
			fmt.Fprintf(&b, "  - %s\n", a)
		}
	}
	if len(anchors) > 0 {
		b.WriteString("\nDeterministic anchors extracted from the FR:\n")
		for _, a := range anchors {
			fmt.Fprintf(&b, "  - %s = %s\n", a.Kind, a.Value)
		}
	}
	b.WriteString("\nCandidate artifacts:\n")
	for _, c := range candidates {
		fmt.Fprintf(&b, "\n[%d] kind=%s identifier=%q file=%s:%d-%d anchored=%v\n",
			c.ID, c.Kind, c.Identifier, c.File, c.StartLine, c.EndLine, c.Anchored)
		if c.Signature != "" {
			fmt.Fprintf(&b, "    signature: %s\n", c.Signature)
		}
		if c.Source != "" {
			fmt.Fprintf(&b, "    source:\n%s\n", indent(c.Source, "    | "))
		}
	}
	b.WriteString("\nReturn the JSON array now.")
	return b.String()
}

// FRSnapshot is the per-FR input to the judgment prompt. Defined here
// so internal/match doesn't need to import internal/fsd.
type FRSnapshot struct {
	ID          string
	Title       string
	Description string
	Acceptance  []string
	Section     string
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
