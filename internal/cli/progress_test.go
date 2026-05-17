package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestProgressBarRendersProgressAndFinishesLine(t *testing.T) {
	var b bytes.Buffer
	p := newProgressBar(&b, "match", true)

	p.Advance(1, 4)
	p.Finish()

	out := b.String()
	if !strings.Contains(out, "match [") || !strings.Contains(out, "1/4") || !strings.HasSuffix(out, "\n") {
		t.Fatalf("progress output = %q", out)
	}
}
