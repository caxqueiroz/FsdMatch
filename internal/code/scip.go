package code

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	scip "github.com/scip-code/scip/bindings/go/scip"
)

// ErrScipJavaMissing is returned by RunScipJava when the binary is not on PATH.
var ErrScipJavaMissing = errors.New("scip-java not found on PATH. Install: https://sourcegraph.github.io/scip-java/docs/getting-started.html")

// RunScipJava shells out to scip-java in repoRoot and writes index.scip
// to outPath. Honours ctx cancellation.
func RunScipJava(ctx context.Context, repoRoot, outPath, scipJavaBin string) error {
	if scipJavaBin == "" {
		scipJavaBin = "scip-java"
	}
	if _, err := exec.LookPath(scipJavaBin); err != nil {
		return ErrScipJavaMissing
	}
	cmd := exec.CommandContext(ctx, scipJavaBin, "index", "--output", outPath) // #nosec G204 -- args are operator-controlled
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("scip-java in %s: %w", repoRoot, err)
	}
	return nil
}

// SymbolHit is one definition harvested from a SCIP document.
type SymbolHit struct {
	// Symbol is the canonical SCIP symbol string ("scip-java maven ...").
	Symbol string
	// FQN is the parsed fully-qualified name ("com.example.notebook.controller.NoteController#getNote").
	FQN string
	// File is the document-relative path from the SCIP index.
	File string
	// Line is 1-based.
	Line int
}

// Reference is one (caller, callee) pair derived from a Reference occurrence.
type Reference struct {
	FromFile string
	FromLine int
	ToSymbol string
}

// ScipDigest summarises a parsed scip index.
type ScipDigest struct {
	// Definitions, keyed by FQN for fast joining against harvester output.
	Definitions map[string]SymbolHit
	// References from any caller site to a callee symbol.
	References []Reference
	// ProjectRoot from the index Metadata, or "" when absent.
	ProjectRoot string
}

// ParseScipIndex reads a SCIP protobuf file streamingly and returns a
// digest suitable for merging against harvester output.
func ParseScipIndex(ctx context.Context, path string) (*ScipDigest, error) {
	f, err := os.Open(path) // #nosec G304 -- caller-supplied path
	if err != nil {
		return nil, fmt.Errorf("open scip %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	d := &ScipDigest{Definitions: map[string]SymbolHit{}}
	visitor := scip.IndexVisitor{
		VisitMetadata: func(_ context.Context, m *scip.Metadata) error {
			if m != nil {
				d.ProjectRoot = m.GetProjectRoot()
			}
			return nil
		},
		VisitDocument: func(_ context.Context, doc *scip.Document) error {
			absorbDocument(d, doc)
			return nil
		},
	}
	if err := visitor.ParseStreaming(ctx, f); err != nil {
		return nil, fmt.Errorf("parse scip %s: %w", path, err)
	}
	return d, nil
}

func absorbDocument(d *ScipDigest, doc *scip.Document) {
	relPath := doc.GetRelativePath()
	for _, occ := range doc.GetOccurrences() {
		role := occ.GetSymbolRoles()
		// Definition occurrences populate Definitions.
		if role&int32(scip.SymbolRole_Definition) != 0 {
			fqn := fqnFromSymbol(occ.GetSymbol())
			if fqn == "" {
				continue
			}
			d.Definitions[fqn] = SymbolHit{
				Symbol: occ.GetSymbol(),
				FQN:    fqn,
				File:   relPath,
				Line:   firstLineOfRange(occ.GetRange()),
			}
			continue
		}
		// Pure references are call-graph edges.
		d.References = append(d.References, Reference{
			FromFile: relPath,
			FromLine: firstLineOfRange(occ.GetRange()),
			ToSymbol: occ.GetSymbol(),
		})
	}
}

func firstLineOfRange(r []int32) int {
	if len(r) == 0 {
		return 0
	}
	return int(r[0]) + 1
}

// fqnFromSymbol parses a SCIP symbol string to a Java FQN.
//
// Example input  : "scip-java maven . . com/example/notebook/controller/NoteController#getNote()."
// Example output : "com.example.notebook.controller.NoteController.getNote"
//
// Best-effort: anything we can't parse cleanly returns "".
func fqnFromSymbol(sym string) string {
	if sym == "" {
		return ""
	}
	// Last whitespace-separated token is the descriptor.
	tail := sym
	if i := strings.LastIndex(sym, " "); i >= 0 {
		tail = sym[i+1:]
	}
	// Strip leading scheme-specific dots used as separators.
	tail = strings.TrimSpace(tail)
	if tail == "" {
		return ""
	}
	// Splitting on "/" reveals the package + class. The trailing "."
	// (Sourcegraph descriptor terminator) and "#method()." form for
	// methods are unwound below.
	parts := strings.Split(tail, "/")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	parts = parts[:len(parts)-1]

	// last looks like "Class#method()." or "Class#field." or "Class#" or "Class.".
	className := last
	method := ""
	if i := strings.Index(last, "#"); i >= 0 {
		className = last[:i]
		rest := strings.TrimSuffix(last[i+1:], ".")
		// drop "()" and signature trailers.
		if j := strings.Index(rest, "("); j >= 0 {
			rest = rest[:j]
		}
		method = rest
	} else {
		className = strings.TrimSuffix(className, ".")
	}
	full := strings.Join(append(parts, className), ".")
	if method != "" {
		full += "." + method
	}
	return full
}

// MergeIntoArtifacts fills Artifact.ScipSymbol on entries whose FQN
// matches a SCIP definition. Returns the number of artifacts updated.
func (d *ScipDigest) MergeIntoArtifacts(arts []Artifact) int {
	if d == nil || len(d.Definitions) == 0 {
		return 0
	}
	updated := 0
	for i := range arts {
		fqn := arts[i].FQN()
		if hit, ok := d.Definitions[fqn]; ok {
			arts[i].ScipSymbol = hit.Symbol
			updated++
		}
	}
	return updated
}

// CalledArtifacts returns reference edges grouped per caller artifact.
// Each (caller, callee) pair is unique. Callers are matched by (file,
// line range) overlap; callees are SCIP symbol strings, leaving the
// row-id resolution to the indexer.
func (d *ScipDigest) CalledArtifacts(arts []Artifact, indexRoot string) map[int][]string {
	if d == nil || len(d.References) == 0 {
		return nil
	}
	byPath := groupArtifactsByFile(arts, indexRoot)
	out := map[int][]string{}
	for _, ref := range d.References {
		owners := byPath[ref.FromFile]
		for _, idx := range owners {
			a := arts[idx]
			if ref.FromLine < a.StartLine || ref.FromLine > a.EndLine {
				continue
			}
			out[idx] = appendUnique(out[idx], ref.ToSymbol)
		}
	}
	return out
}

// groupArtifactsByFile keys artifact slice indices by SCIP-relative
// path. SCIP records doc paths relative to the project root; harvester
// paths are absolute, so we strip indexRoot before keying.
func groupArtifactsByFile(arts []Artifact, indexRoot string) map[string][]int {
	out := map[string][]int{}
	indexRoot = filepath.Clean(indexRoot)
	for i, a := range arts {
		rel := a.File
		if abs, err := filepath.Abs(a.File); err == nil {
			if r, err := filepath.Rel(indexRoot, abs); err == nil {
				rel = filepath.ToSlash(r)
			}
		}
		out[rel] = append(out[rel], i)
	}
	return out
}

func appendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}
