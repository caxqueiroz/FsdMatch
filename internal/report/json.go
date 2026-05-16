package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteJSON serialises the full Report as report.json under outDir.
// This is the lossless dump for downstream tooling.
func WriteJSON(r *Report, outDir string) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	path := filepath.Join(outDir, "report.json")
	f, err := os.Create(path) // #nosec G304 -- outDir from CLI flag
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return fmt.Errorf("encode report: %w", err)
	}
	return nil
}
