package code

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
)

// Indexer turns harvested artifacts + tests into rows in code_artifacts,
// tests, relationships, and artifact_vec. SCIP merge is optional.
type Indexer struct {
	d        *db.DB
	embedder embed.Embedder
	cache    *embed.Cache
	logger   *slog.Logger
}

// NewIndexer constructs an Indexer.
func NewIndexer(d *db.DB, embedder embed.Embedder) *Indexer {
	return &Indexer{
		d:        d,
		embedder: embedder,
		cache:    embed.NewCache(d),
		logger:   slog.Default(),
	}
}

// IndexResult is the per-run summary returned by Index.
type IndexResult struct {
	RunID         string
	ArtifactCount int
	TestCount     int
	RelationCount int
	ScipMerged    int
}

// Index runs the full code-side pipeline: harvest, optional SCIP merge,
// embed, write.
func (ix *Indexer) Index(ctx context.Context, repoRoot, scipIndexPath, runID string) (*IndexResult, error) {
	if runID == "" {
		runID = fmt.Sprintf("index-%d", time.Now().Unix())
	}

	arts, tests, err := HarvestSpring(ctx, repoRoot)
	if err != nil {
		return nil, fmt.Errorf("harvest: %w", err)
	}

	var (
		digest      *ScipDigest
		scipMerged  int
		callEdgeMap map[int][]string
	)
	if scipIndexPath != "" {
		digest, err = ParseScipIndex(ctx, scipIndexPath)
		if err != nil {
			return nil, err
		}
		scipMerged = digest.MergeIntoArtifacts(arts)
		callEdgeMap = digest.CalledArtifacts(arts, repoRoot)
	}

	// Wipe prior rows for this repo's path so re-runs stay idempotent
	// without leaking deletions across unrelated runs.
	if err := ix.purgePreviousRows(ctx, repoRoot); err != nil {
		return nil, err
	}

	rowIDs, err := ix.writeArtifacts(ctx, arts, runID)
	if err != nil {
		return nil, err
	}
	if err := ix.embedArtifacts(ctx, arts, rowIDs); err != nil {
		return nil, err
	}
	if err := ix.writeTests(ctx, tests, arts, rowIDs, runID); err != nil {
		return nil, err
	}

	relations := 0
	if callEdgeMap != nil {
		relations, err = ix.writeRelationships(ctx, arts, rowIDs, callEdgeMap, digest)
		if err != nil {
			return nil, err
		}
	}

	return &IndexResult{
		RunID:         runID,
		ArtifactCount: len(arts),
		TestCount:     len(tests),
		RelationCount: relations,
		ScipMerged:    scipMerged,
	}, nil
}

// purgePreviousRows wipes prior rows whose `file` lives under the
// indexed repo root, plus any tests and relationships that referenced
// them. Lets re-runs against the same tree replace rather than duplicate.
func (ix *Indexer) purgePreviousRows(ctx context.Context, repoRoot string) error {
	prefix := strings.TrimSuffix(repoRoot, "/") + "/"
	return ix.d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		// 1. Drop tests pointing into this tree first (foreign keys would block otherwise).
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM tests WHERE target_artifact IN
			   (SELECT id FROM code_artifacts WHERE file LIKE ?)`,
			prefix+"%"); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM tests WHERE file LIKE ?`, prefix+"%"); err != nil {
			return err
		}
		// 2. Drop relationships referencing artifacts in this tree.
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM relationships WHERE
			   from_artifact IN (SELECT id FROM code_artifacts WHERE file LIKE ?) OR
			   to_artifact   IN (SELECT id FROM code_artifacts WHERE file LIKE ?)`,
			prefix+"%", prefix+"%"); err != nil {
			return err
		}
		// 3. Drop artifact_vec rows for those artifacts.
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM artifact_vec WHERE rowid IN
			   (SELECT id FROM code_artifacts WHERE file LIKE ?)`,
			prefix+"%"); err != nil {
			return err
		}
		// 4. Finally drop the artifacts themselves.
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM code_artifacts WHERE file LIKE ?`,
			prefix+"%"); err != nil {
			return err
		}
		return nil
	})
}

// writeArtifacts inserts all artifacts in one tx and returns the
// allocated rowids in input order.
func (ix *Indexer) writeArtifacts(ctx context.Context, arts []Artifact, runID string) ([]int64, error) {
	rowIDs := make([]int64, len(arts))
	err := ix.d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO code_artifacts
			  (kind, identifier, scip_symbol, package, class, method,
			   file, start_line, end_line, signature, annotations, source, run_id)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
		`)
		if err != nil {
			return err
		}
		defer func() { _ = stmt.Close() }()

		for i, a := range arts {
			ann, err := json.Marshal(a.Annotations)
			if err != nil {
				return err
			}
			res, err := stmt.ExecContext(ctx,
				a.Kind, a.Identifier, a.ScipSymbol,
				a.Package, a.Class, a.Method,
				a.File, a.StartLine, a.EndLine,
				a.Signature, string(ann), a.Source, runID,
			)
			if err != nil {
				return err
			}
			id, err := res.LastInsertId()
			if err != nil {
				return err
			}
			rowIDs[i] = id
		}
		return nil
	})
	return rowIDs, err
}

// embedArtifacts computes one vector per artifact via the cache and
// inserts into artifact_vec.
func (ix *Indexer) embedArtifacts(ctx context.Context, arts []Artifact, rowIDs []int64) error {
	for i, a := range arts {
		text := EmbeddingText(a)
		v, err := embed.Cached(ctx, ix.embedder, ix.cache, text)
		if err != nil {
			return fmt.Errorf("embed artifact %s: %w", a.Identifier, err)
		}
		blob := db.PackFloat32(v)
		rowID := rowIDs[i]
		if err := ix.d.Writer().Submit(ctx, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx,
				`DELETE FROM artifact_vec WHERE rowid = ?`, rowID); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx,
				`INSERT INTO artifact_vec(rowid, embedding) VALUES (?, ?)`, rowID, blob)
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

// EmbeddingText is the canonical text fed to the embedding model for an artifact.
func EmbeddingText(a Artifact) string {
	var b strings.Builder
	b.WriteString(a.Kind)
	b.WriteString(": ")
	b.WriteString(a.Identifier)
	if a.Signature != "" {
		b.WriteString("\n")
		b.WriteString(a.Signature)
	}
	if a.Source != "" {
		b.WriteString("\n")
		b.WriteString(a.Source)
	}
	return b.String()
}

func (ix *Indexer) writeTests(ctx context.Context, tests []TestCase, arts []Artifact, rowIDs []int64, runID string) error {
	if len(tests) == 0 {
		return nil
	}
	// Build a path → artifactID map for MockMvc → rest_endpoint linking.
	pathToRowID := map[string]int64{}
	for i, a := range arts {
		if a.Kind == "rest_endpoint" {
			pathToRowID[a.Identifier] = rowIDs[i]
		}
	}

	return ix.d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO tests(name, file, line, test_kind, target_artifact, asserts, run_id)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		defer func() { _ = stmt.Close() }()

		for _, t := range tests {
			var target *int64
			for _, p := range t.MockPaths {
				if id, ok := pathToRowID[normalizeMockPath(p)]; ok {
					v := id
					target = &v
					break
				}
			}
			asserts, err := json.Marshal(orEmpty(t.Asserts))
			if err != nil {
				return err
			}
			if _, err := stmt.ExecContext(ctx,
				t.Name, t.File, t.Line, t.TestKind, target, string(asserts), runID,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

// normalizeMockPath converts a concrete MockMvc path like
// "GET /api/v1/notes/1" into the @{Get,Post,...}Mapping form
// "GET /api/v1/notes/{id}" by replacing each segment composed entirely
// of digits with "{id}". Best-effort; richer matching can come later.
func normalizeMockPath(p string) string {
	parts := strings.SplitN(p, " ", 2)
	if len(parts) != 2 {
		return p
	}
	verb, path := parts[0], parts[1]
	segs := strings.Split(path, "/")
	for i, s := range segs {
		if s == "" {
			continue
		}
		isDigits := true
		for _, r := range s {
			if r < '0' || r > '9' {
				isDigits = false
				break
			}
		}
		if isDigits {
			segs[i] = "{id}"
		}
	}
	return verb + " " + strings.Join(segs, "/")
}

func (ix *Indexer) writeRelationships(ctx context.Context, arts []Artifact, rowIDs []int64, edges map[int][]string, _ *ScipDigest) (int, error) {
	if len(edges) == 0 {
		return 0, nil
	}
	// SCIP symbol → artifact rowid, built from rows whose ScipSymbol
	// was filled in by MergeIntoArtifacts.
	symbolToRow := make(map[string]int64, len(arts))
	for i, a := range arts {
		if a.ScipSymbol != "" {
			symbolToRow[a.ScipSymbol] = rowIDs[i]
		}
	}

	count := 0
	err := ix.d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		for fromIdx, callees := range edges {
			fromID := rowIDs[fromIdx]
			for _, sym := range callees {
				toID, ok := symbolToRow[sym]
				if !ok {
					continue
				}
				if _, err := tx.ExecContext(ctx,
					`INSERT OR IGNORE INTO relationships(from_artifact, to_artifact, kind)
					 VALUES (?, ?, 'calls')`, fromID, toID); err != nil {
					return err
				}
				count++
			}
		}
		return nil
	})
	return count, err
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
