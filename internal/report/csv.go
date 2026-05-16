package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// WriteCSV writes matches.csv, drift.csv, and orphans.csv under outDir.
// One row per match; cells with embedded newlines or commas are
// CSV-escaped via encoding/csv.
func WriteCSV(r *Report, outDir string) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	if err := writeMatchesCSV(r, filepath.Join(outDir, "matches.csv")); err != nil {
		return err
	}
	if err := writeDriftCSV(r, filepath.Join(outDir, "drift.csv")); err != nil {
		return err
	}
	return writeOrphansCSV(r, filepath.Join(outDir, "orphans.csv"))
}

func writeMatchesCSV(r *Report, path string) error {
	f, err := os.Create(path) // #nosec G304 -- outDir from CLI flag
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	w := csv.NewWriter(f)
	defer w.Flush()

	if err := w.Write([]string{
		"run_id", "feature_id", "feature_title", "section",
		"artifact_id", "kind", "identifier", "file", "start_line", "end_line",
		"verdict", "confidence", "tested", "test_count",
		"evidence_files", "notes",
	}); err != nil {
		return err
	}

	for _, s := range r.Sections {
		for _, fc := range s.Features {
			for _, m := range fc.Matches {
				row := []string{
					r.RunID, fc.ID, fc.Title, s.Name,
					strconv.FormatInt(m.ArtifactID, 10), m.Kind, m.Identifier,
					m.File, strconv.Itoa(m.StartLine), strconv.Itoa(m.EndLine),
					m.Verdict, fmt.Sprintf("%.4f", m.Confidence),
					strconv.FormatBool(m.Tested), strconv.Itoa(m.TestCount),
					evidenceFilesString(m.Evidence), m.Notes,
				}
				if err := w.Write(row); err != nil {
					return err
				}
			}
		}
	}
	return w.Error()
}

func writeDriftCSV(r *Report, path string) error {
	f, err := os.Create(path) // #nosec G304 -- outDir from CLI flag
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{
		"feature_id", "feature_title", "section",
		"artifact_id", "kind", "identifier", "file", "start_line", "end_line",
		"confidence", "evidence_files", "notes",
	}); err != nil {
		return err
	}
	for _, d := range r.Drift {
		row := []string{
			d.FeatureID, d.FeatureTitle, d.Section,
			strconv.FormatInt(d.ArtifactID, 10), d.Kind, d.Identifier,
			d.File, strconv.Itoa(d.StartLine), strconv.Itoa(d.EndLine),
			fmt.Sprintf("%.4f", d.Confidence),
			evidenceFilesString(d.Evidence), d.Notes,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return w.Error()
}

func writeOrphansCSV(r *Report, path string) error {
	f, err := os.Create(path) // #nosec G304 -- outDir from CLI flag
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"artifact_id", "kind", "identifier", "file", "start_line", "end_line"}); err != nil {
		return err
	}
	for _, o := range r.Orphans {
		row := []string{
			strconv.FormatInt(o.ArtifactID, 10), o.Kind, o.Identifier,
			o.File, strconv.Itoa(o.StartLine), strconv.Itoa(o.EndLine),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return w.Error()
}

func evidenceFilesString(ev []Evidence) string {
	if len(ev) == 0 {
		return ""
	}
	parts := make([]string, 0, len(ev))
	for _, e := range ev {
		parts = append(parts, fmt.Sprintf("%s:%d-%d", e.File, e.Start, e.End))
	}
	return strings.Join(parts, ";")
}
