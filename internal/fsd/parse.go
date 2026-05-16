// Package fsd parses an FSD document into atomic Functional Requirement
// chunks and then into structured Feature rows via the Bedrock atomizer.
//
// Phase 2 supports markdown only. PDF and DOCX are deferred to a later
// phase per SPEC §7.1.
package fsd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// DefaultAnchorPattern matches "FR-001", "FR-042", etc. Users can
// override via the --anchor-pattern flag.
const DefaultAnchorPattern = `FR-\d+`

// Chunk is one slice of the FSD, headed by an anchor that appears
// somewhere on the first line.
type Chunk struct {
	// Anchor is the matched anchor text, e.g. "FR-042".
	Anchor string
	// Section is the most recent H1/H2 heading above this chunk.
	Section string
	// Text is the full chunk content including the heading line.
	Text string
	// Line is the 1-based line number where the chunk starts.
	Line int
}

var headingRe = regexp.MustCompile(`^(#{1,3})\s+(.+?)\s*$`)

// ParseMarkdown splits raw markdown into anchor-keyed chunks. A chunk
// runs from one anchor's heading up to the line preceding the next
// anchor. Content before the first anchor is discarded.
//
// pattern must compile as a Go regular expression; if empty,
// DefaultAnchorPattern is used.
func ParseMarkdown(raw, pattern string) ([]Chunk, error) {
	if pattern == "" {
		pattern = DefaultAnchorPattern
	}
	anchorRe, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("anchor pattern %q: %w", pattern, err)
	}

	type startIdx struct {
		line    int
		anchor  string
		section string
	}

	var (
		section string
		starts  []startIdx
		lines   []string
	)

	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	for lineNum := 1; scanner.Scan(); lineNum++ {
		l := scanner.Text()
		lines = append(lines, l)
		if m := headingRe.FindStringSubmatch(l); m != nil {
			// Treat headings without anchors as section markers.
			if !anchorRe.MatchString(l) {
				section = m[2]
				continue
			}
		}
		if a := anchorRe.FindString(l); a != "" {
			starts = append(starts, startIdx{line: lineNum, anchor: a, section: section})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning markdown: %w", err)
	}

	out := make([]Chunk, 0, len(starts))
	for i, s := range starts {
		end := len(lines)
		if i+1 < len(starts) {
			end = starts[i+1].line - 1
		}
		body := strings.Join(lines[s.line-1:end], "\n")
		out = append(out, Chunk{
			Anchor:  s.anchor,
			Section: s.section,
			Text:    strings.TrimRight(body, "\n"),
			Line:    s.line,
		})
	}
	return out, nil
}

// ParseFile is a convenience wrapper that reads path and calls
// ParseMarkdown. Caller is responsible for ensuring path is a markdown
// file.
func ParseFile(path, pattern string) ([]Chunk, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- CLI user-supplied path
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return ParseMarkdown(string(raw), pattern)
}
