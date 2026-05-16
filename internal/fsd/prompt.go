package fsd

import (
	"fmt"
	"strings"
)

// AtomizerPromptVersion is the version stamp written to
// features metadata so historical runs remain attributable.
// HARD CONSTRAINT (CLAUDE.md): bump this whenever the prompt changes.
const AtomizerPromptVersion = "fsd-atomize-v1"

// DefaultAtomizerModel is the Bedrock Anthropic model used for FSD
// atomization. Override via config.
const DefaultAtomizerModel = "anthropic.claude-sonnet-4-v2:0"

// BedrockAnthropicVersion is the required `anthropic_version` field for
// the Bedrock-hosted Claude Messages API.
const BedrockAnthropicVersion = "bedrock-2023-05-31"

// AtomizerSystem describes the task to Claude. Kept short, declarative,
// and grounded in the schema columns.
const AtomizerSystem = `You are an analyst extracting one Functional Requirement (FR) from a slice of a Functional Specification Document.

Return EXACTLY one JSON object — no prose, no markdown fence — with these keys:

{
  "id":            string,        // the FR identifier, e.g. "FR-042"
  "title":         string,        // single-sentence summary
  "description":   string,        // paragraph describing the requirement
  "acceptance":    string[],      // array of testable criteria; empty array if none stated
  "actor":         string|null,   // primary actor or system; null if unstated
  "inputs":        string[],      // input data/events; empty array if none
  "outputs":       string[],      // outputs or post-conditions; empty array if none
  "side_effects":  string[],      // notable side-effects; empty array if none
  "non_functional": string[]      // perf/security/UX constraints; empty array if none
}

Rules:
- Quote the FR id exactly as it appears in the source.
- Do not invent details not present in the source.
- Prefer brevity over completeness.`

// BuildAtomizerUserMessage assembles the user-side payload for one chunk.
// It echoes the anchor so the model can pin down the id even if the
// chunk omits an explicit "FR-NNN" line.
func BuildAtomizerUserMessage(anchor, section, body string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Anchor: %s\n", anchor)
	if section != "" {
		fmt.Fprintf(&b, "Section: %s\n", section)
	}
	b.WriteString("\nFSD slice:\n---\n")
	b.WriteString(body)
	b.WriteString("\n---\n")
	return b.String()
}
