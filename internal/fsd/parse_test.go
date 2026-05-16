package fsd

import (
	"strings"
	"testing"
)

func TestParseMarkdownSplitsByAnchor(t *testing.T) {
	chunks, err := ParseFile("../../testdata/fsd-sample.md", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 5 {
		t.Fatalf("expected 5 chunks, got %d", len(chunks))
	}
	wantAnchors := []string{"FR-001", "FR-002", "FR-010", "FR-011", "FR-020"}
	for i, c := range chunks {
		if c.Anchor != wantAnchors[i] {
			t.Errorf("chunk %d: anchor %q want %q", i, c.Anchor, wantAnchors[i])
		}
		if !strings.Contains(c.Text, c.Anchor) {
			t.Errorf("chunk %d text missing its own anchor: %q", i, c.Text[:min(80, len(c.Text))])
		}
	}
	if chunks[0].Section != "Authentication" {
		t.Errorf("chunk 0 section = %q", chunks[0].Section)
	}
	if chunks[4].Section != "Scheduled jobs" {
		t.Errorf("chunk 4 section = %q", chunks[4].Section)
	}
}

func TestParseMarkdownCustomPattern(t *testing.T) {
	src := `# top
## sec
US-1: do a thing
US-2: do another`
	chunks, err := ParseMarkdown(src, `US-\d+`)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 2 {
		t.Fatalf("got %d chunks", len(chunks))
	}
}

func TestParseMarkdownInvalidPattern(t *testing.T) {
	if _, err := ParseMarkdown("", "FR-[a-"); err == nil {
		t.Fatal("expected regex error")
	}
}
