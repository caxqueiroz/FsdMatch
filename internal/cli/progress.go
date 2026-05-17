package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const progressWidth = 24

type progressBar struct {
	mu       sync.Mutex
	w        io.Writer
	label    string
	enabled  bool
	rendered bool
	lastLen  int
}

func newProgress(w io.Writer, label string) *progressBar {
	return newProgressBar(w, label, isTerminalWriter(w))
}

func newProgressBar(w io.Writer, label string, enabled bool) *progressBar {
	return &progressBar{w: w, label: label, enabled: enabled}
}

func (p *progressBar) Advance(done, total int) {
	if p == nil || !p.enabled || p.w == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	line := p.render(done, total)
	pad := ""
	if p.lastLen > len(line) {
		pad = strings.Repeat(" ", p.lastLen-len(line))
	}
	_, _ = fmt.Fprintf(p.w, "\r%s%s", line, pad)
	p.lastLen = len(line)
	p.rendered = true
}

func (p *progressBar) Finish() {
	if p == nil || !p.enabled || p.w == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.rendered {
		_, _ = fmt.Fprintln(p.w)
		p.rendered = false
	}
}

func (p *progressBar) render(done, total int) string {
	if total <= 0 {
		return fmt.Sprintf("%s %d", p.label, done)
	}
	if done < 0 {
		done = 0
	}
	if done > total {
		done = total
	}
	filled := done * progressWidth / total
	bar := strings.Repeat("=", filled) + strings.Repeat(" ", progressWidth-filled)
	percent := done * 100 / total
	return fmt.Sprintf("%s [%s] %d/%d %3d%%", p.label, bar, done, total, percent)
}

func isTerminalWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
