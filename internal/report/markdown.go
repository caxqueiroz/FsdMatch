package report

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// WriteMarkdown writes coverage.md, drift.md and orphans.md under outDir.
// Phase 5 acceptance hits these three filenames specifically.
func WriteMarkdown(r *Report, outDir string) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	for _, x := range []struct {
		name string
		fn   func(io.Writer, *Report) error
	}{
		{"coverage.md", renderCoverageMD},
		{"drift.md", renderDriftMD},
		{"orphans.md", renderOrphansMD},
	} {
		path := filepath.Join(outDir, x.name)
		f, err := os.Create(path) // #nosec G304 -- outDir from CLI flag
		if err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}
		if err := x.fn(f, r); err != nil {
			_ = f.Close()
			return fmt.Errorf("render %s: %w", x.name, err)
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

// RenderMarkdownCoverage renders the coverage matrix to w. Exposed so
// non-file callers (e.g. the MCP coverage resource) can stream the same
// output without writing to disk.
func RenderMarkdownCoverage(w io.Writer, r *Report) error { return renderCoverageMD(w, r) }

// RenderMarkdownDrift renders the drift list to w.
func RenderMarkdownDrift(w io.Writer, r *Report) error { return renderDriftMD(w, r) }

// RenderMarkdownOrphans renders the orphan list to w.
func RenderMarkdownOrphans(w io.Writer, r *Report) error { return renderOrphansMD(w, r) }

func renderCoverageMD(w io.Writer, r *Report) error {
	header(w, r, "Coverage")
	if len(r.Sections) == 0 {
		_, err := fmt.Fprintln(w, "_No features in the database._")
		return err
	}
	if err := renderRollup(w, r); err != nil {
		return err
	}
	for _, s := range r.Sections {
		if _, err := fmt.Fprintf(w, "## Section: %s\n\n", s.Name); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w,
			"Total %d · implemented %d · drifts %d · missing %d\n\n",
			s.Total, s.Implemented, s.Drifts, s.Missing); err != nil {
			return err
		}
		for _, f := range s.Features {
			if err := renderFeatureMD(w, f); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderRollup(w io.Writer, r *Report) error {
	if _, err := fmt.Fprintln(w, "## Roll-up"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| Section | Total | Implemented | Drifts | Missing |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "|---|---:|---:|---:|---:|"); err != nil {
		return err
	}
	var t, i, d, m int
	for _, s := range r.Sections {
		if _, err := fmt.Fprintf(w, "| %s | %d | %d | %d | %d |\n",
			escapePipes(s.Name), s.Total, s.Implemented, s.Drifts, s.Missing); err != nil {
			return err
		}
		t += s.Total
		i += s.Implemented
		d += s.Drifts
		m += s.Missing
	}
	_, err := fmt.Fprintf(w, "| **Total** | **%d** | **%d** | **%d** | **%d** |\n\n", t, i, d, m)
	return err
}

func renderFeatureMD(w io.Writer, f FeatureCoverage) error {
	badge := strings.ToUpper(string(f.Status))
	if _, err := fmt.Fprintf(w, "### %s — %s — %s\n\n", f.ID, escapePipes(f.Title), badge); err != nil {
		return err
	}
	if len(f.Matches) == 0 {
		_, err := fmt.Fprintln(w, "_No matching artifacts._")
		return err
	}
	if _, err := fmt.Fprintln(w, "| Verdict | Confidence | Tested | Artifact | Location |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "|---|---:|:---:|---|---|"); err != nil {
		return err
	}
	for _, m := range f.Matches {
		tested := ""
		if m.Tested {
			tested = fmt.Sprintf("✓ (%d)", m.TestCount)
		}
		if _, err := fmt.Fprintf(w, "| %s | %.2f | %s | `%s` %s | `%s:%d-%d` |\n",
			m.Verdict, m.Confidence, tested,
			m.Kind, escapePipes(m.Identifier), shortPath(m.File), m.StartLine, m.EndLine); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	// Evidence snippets, if any, for non-unrelated rows.
	for _, m := range f.Matches {
		if m.Verdict == "unrelated" || len(m.Evidence) == 0 {
			continue
		}
		if _, err := fmt.Fprintf(w, "**Evidence for `%s`:**\n", m.Identifier); err != nil {
			return err
		}
		for _, e := range m.Evidence {
			if _, err := fmt.Fprintf(w, "- `%s:%d-%d`", shortPath(e.File), e.Start, e.End); err != nil {
				return err
			}
			if e.Note != "" {
				if _, err := fmt.Fprintf(w, " — %s", e.Note); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		if m.Notes != "" {
			if _, err := fmt.Fprintf(w, "\n_Notes:_ %s\n", m.Notes); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

func renderDriftMD(w io.Writer, r *Report) error {
	header(w, r, "Drift")
	if len(r.Drift) == 0 {
		_, err := fmt.Fprintln(w, "_No drift detected._")
		return err
	}
	if _, err := fmt.Fprintf(w, "%d drift verdicts.\n\n", len(r.Drift)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| FR | Section | Artifact | Location | Confidence | Notes |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "|---|---|---|---|---:|---|"); err != nil {
		return err
	}
	for _, d := range r.Drift {
		if _, err := fmt.Fprintf(w, "| %s — %s | %s | `%s` %s | `%s:%d-%d` | %.2f | %s |\n",
			d.FeatureID, escapePipes(d.FeatureTitle), escapePipes(d.Section),
			d.Kind, escapePipes(d.Identifier),
			shortPath(d.File), d.StartLine, d.EndLine, d.Confidence,
			escapePipes(d.Notes)); err != nil {
			return err
		}
	}
	return nil
}

func renderOrphansMD(w io.Writer, r *Report) error {
	header(w, r, "Orphans")
	if len(r.Orphans) == 0 {
		_, err := fmt.Fprintln(w, "_No orphan public-surface artifacts._")
		return err
	}
	if _, err := fmt.Fprintf(w, "%d artifacts with no `implements` mapping.\n\n", len(r.Orphans)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "| Kind | Identifier | Location |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "|---|---|---|"); err != nil {
		return err
	}
	for _, o := range r.Orphans {
		if _, err := fmt.Fprintf(w, "| `%s` | %s | `%s:%d-%d` |\n",
			o.Kind, escapePipes(o.Identifier),
			shortPath(o.File), o.StartLine, o.EndLine); err != nil {
			return err
		}
	}
	return nil
}

func header(w io.Writer, r *Report, name string) {
	_, _ = fmt.Fprintf(w, "# %s — fsdtrace\n\n", name)
	_, _ = fmt.Fprintf(w, "_Run `%s` · generated %s UTC_\n\n",
		fallback(r.RunID, "(no matches)"),
		r.Generated.Format("2006-01-02 15:04:05"))
}

func fallback(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}

func escapePipes(s string) string {
	return strings.ReplaceAll(s, "|", `\|`)
}

// shortPath strips the working directory prefix from absolute paths so
// the output stays readable. Best-effort: errors fall through to the
// raw path.
func shortPath(p string) string {
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, p); err == nil && !strings.HasPrefix(rel, "..") {
			return rel
		}
	}
	return p
}
