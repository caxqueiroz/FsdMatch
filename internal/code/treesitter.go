// Package code indexes a Java/Spring Boot source tree into the
// code_artifacts and tests tables.
//
// Three layers, all optional individually:
//  1. tree-sitter walker (this file): produces (kind, location) tuples
//     by scanning .java files for Spring annotations.
//  2. SCIP merge (scip.go): when scip-java has produced an index.scip
//     file, fills in scip_symbol on existing rows and inserts
//     relationships from reference occurrences.
//  3. Test extractor (tests.go): pulls out @Test methods and any
//     MockMvc paths they exercise.
//
// SPEC §7.2 hard constraint: tree-sitter is the AST. Switching
// requires bumping the SPEC.
package code

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/java"
)

// File holds the parsed tree-sitter view of one .java source file. The
// tree is owned by the file and freed by Close().
type File struct {
	Path   string
	Pkg    string
	Source []byte
	Tree   *sitter.Tree
	parser *sitter.Parser
}

// Close releases tree-sitter resources.
func (f *File) Close() {
	if f.Tree != nil {
		f.Tree.Close()
		f.Tree = nil
	}
	if f.parser != nil {
		f.parser.Close()
		f.parser = nil
	}
}

// ParseFile parses a single Java source file.
func ParseFile(ctx context.Context, path string) (*File, error) {
	src, err := os.ReadFile(path) // #nosec G304 -- caller-supplied indexer root
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	parser := sitter.NewParser()
	parser.SetLanguage(java.GetLanguage())
	tree, err := parser.ParseCtx(ctx, nil, src)
	if err != nil {
		parser.Close()
		return nil, fmt.Errorf("tree-sitter parse %s: %w", path, err)
	}
	pkg := extractPackage(tree.RootNode(), src)
	return &File{Path: path, Pkg: pkg, Source: src, Tree: tree, parser: parser}, nil
}

// WalkSourceTree finds every .java file under root and yields it to fn.
// Any walk-time error short-circuits the traversal. Files are closed
// after fn returns.
func WalkSourceTree(ctx context.Context, root string, fn func(ctx context.Context, f *File) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip vendored / build output directories, common in JVM repos.
			name := d.Name()
			if name == "target" || name == "build" || name == ".git" || name == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".java") {
			return nil
		}
		f, err := ParseFile(ctx, path)
		if err != nil {
			return err
		}
		defer f.Close()
		return fn(ctx, f)
	})
}

// nodeText returns the source slice for n.
func nodeText(src []byte, n *sitter.Node) string {
	if n == nil {
		return ""
	}
	return string(src[n.StartByte():n.EndByte()])
}

// startLine returns the 1-based starting line number of n.
func startLine(n *sitter.Node) int {
	if n == nil {
		return 0
	}
	return int(n.StartPoint().Row) + 1
}

// endLine returns the 1-based ending line number of n (inclusive).
func endLine(n *sitter.Node) int {
	if n == nil {
		return 0
	}
	return int(n.EndPoint().Row) + 1
}

// extractPackage finds the package_declaration and returns its
// identifier, or "" when none is present.
func extractPackage(root *sitter.Node, src []byte) string {
	for i := 0; i < int(root.NamedChildCount()); i++ {
		c := root.NamedChild(i)
		if c.Type() == "package_declaration" {
			if id := firstChildOfType(c, "scoped_identifier"); id != nil {
				return nodeText(src, id)
			}
			if id := firstChildOfType(c, "identifier"); id != nil {
				return nodeText(src, id)
			}
		}
	}
	return ""
}

// firstChildOfType returns the first named child of n with the given
// type, or nil if absent.
func firstChildOfType(n *sitter.Node, typeName string) *sitter.Node {
	if n == nil {
		return nil
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		c := n.NamedChild(i)
		if c.Type() == typeName {
			return c
		}
	}
	return nil
}

// findClassDeclarations returns every class_declaration,
// interface_declaration, and record_declaration in the file.
func findClassDeclarations(root *sitter.Node) []*sitter.Node {
	var out []*sitter.Node
	walk(root, func(n *sitter.Node) bool {
		switch n.Type() {
		case "class_declaration", "interface_declaration", "record_declaration":
			out = append(out, n)
		}
		return true
	})
	return out
}

// walk depth-first invokes fn on every named node. Return false from
// fn to stop descending into the subtree.
func walk(n *sitter.Node, fn func(*sitter.Node) bool) {
	if n == nil {
		return
	}
	if !fn(n) {
		return
	}
	for i := 0; i < int(n.NamedChildCount()); i++ {
		walk(n.NamedChild(i), fn)
	}
}
