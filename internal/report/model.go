// Package report builds and renders the traceability matrix from a
// fsdtrace SQLite database.
//
// The Report struct is the single in-memory model that every renderer
// (markdown, csv, json, html) consumes. It is built once per CLI
// invocation by Load.
package report

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cax/fsdtrace/internal/db"
)

// Status classifies an FR by the strongest verdict among its matches.
type Status string

// FR-level Status values. An FR is implemented if any match verdict is
// "implements"; otherwise drifts when any verdict is "drifts";
// otherwise missing.
const (
	StatusImplemented Status = "implemented"
	StatusDrifts      Status = "drifts"
	StatusMissing     Status = "missing"
)

// Report is the per-run traceability matrix.
type Report struct {
	RunID            string      `json:"run_id"`
	Generated        time.Time   `json:"generated_at"`
	IncludeCallGraph bool        `json:"include_call_graph"`
	Sections         []Section   `json:"sections"`
	Drift            []DriftRow  `json:"drift"`
	Orphans          []OrphanRow `json:"orphans"`
}

// Options controls optional report enrichment.
type Options struct {
	// IncludeCallGraph expands implemented matches through SCIP-derived
	// relationships and attaches reachable support artifacts.
	IncludeCallGraph bool
	// MaxCallDepth limits relationship traversal. Zero uses the default.
	MaxCallDepth int
}

// Section groups features by their fsd_section value. "(unsectioned)"
// for rows with empty section.
type Section struct {
	Name                string            `json:"name"`
	Features            []FeatureCoverage `json:"features"`
	Total               int               `json:"total"`
	Implemented         int               `json:"implemented"`
	Drifts              int               `json:"drifts"`
	Missing             int               `json:"missing"`
	SupportingArtifacts int               `json:"supporting_artifacts"`
}

// FeatureCoverage is one FR with its matches and overall status.
type FeatureCoverage struct {
	ID                  string     `json:"id"`
	Title               string     `json:"title"`
	Section             string     `json:"section"`
	Status              Status     `json:"status"`
	Matches             []MatchRow `json:"matches"`
	TestedAny           bool       `json:"tested_any"`
	SupportingArtifacts int        `json:"supporting_artifacts"`
}

// MatchRow is one matches-table row joined with its artifact.
type MatchRow struct {
	ArtifactID          int64             `json:"artifact_id"`
	Kind                string            `json:"kind"`
	Identifier          string            `json:"identifier"`
	File                string            `json:"file"`
	StartLine           int               `json:"start_line"`
	EndLine             int               `json:"end_line"`
	Verdict             string            `json:"verdict"`
	Confidence          float64           `json:"confidence"`
	Evidence            []Evidence        `json:"evidence"`
	Notes               string            `json:"notes"`
	Tested              bool              `json:"tested"`
	TestCount           int               `json:"test_count"`
	Model               string            `json:"model"`
	PromptVersion       string            `json:"prompt_version"`
	SupportingArtifacts []SupportArtifact `json:"supporting_artifacts,omitempty"`
}

// SupportArtifact is a code artifact reachable from a directly
// implemented match through SCIP-derived relationships.
type SupportArtifact struct {
	ArtifactID       int64  `json:"artifact_id"`
	Kind             string `json:"kind"`
	Identifier       string `json:"identifier"`
	File             string `json:"file"`
	StartLine        int    `json:"start_line"`
	EndLine          int    `json:"end_line"`
	Depth            int    `json:"depth"`
	RelationshipKind string `json:"relationship_kind"`
}

// Evidence mirrors the JSON shape stored in matches.evidence.
type Evidence struct {
	File  string `json:"file"`
	Start int    `json:"start"`
	End   int    `json:"end"`
	Note  string `json:"note,omitempty"`
}

// DriftRow flattens a drifts verdict for the drift report.
type DriftRow struct {
	FeatureID    string     `json:"feature_id"`
	FeatureTitle string     `json:"feature_title"`
	Section      string     `json:"section"`
	ArtifactID   int64      `json:"artifact_id"`
	Kind         string     `json:"kind"`
	Identifier   string     `json:"identifier"`
	File         string     `json:"file"`
	StartLine    int        `json:"start_line"`
	EndLine      int        `json:"end_line"`
	Confidence   float64    `json:"confidence"`
	Evidence     []Evidence `json:"evidence"`
	Notes        string     `json:"notes"`
}

// OrphanRow is an artifact with no `implements` verdict against any FR.
type OrphanRow struct {
	ArtifactID int64  `json:"artifact_id"`
	Kind       string `json:"kind"`
	Identifier string `json:"identifier"`
	File       string `json:"file"`
	StartLine  int    `json:"start_line"`
	EndLine    int    `json:"end_line"`
}

// Load assembles a Report from the database. If runID is "" the most
// recent run_id from the matches table is used; if there are no matches
// at all the report is still produced (with empty per-FR matches),
// because the orphans + features-by-section view is independently
// useful.
func Load(ctx context.Context, d *db.DB, runID string) (*Report, error) {
	return LoadWithOptions(ctx, d, runID, Options{})
}

// LoadWithOptions assembles a Report with optional enrichment.
func LoadWithOptions(ctx context.Context, d *db.DB, runID string, opts Options) (*Report, error) {
	if runID == "" {
		var ns sql.NullString
		err := d.SQL().QueryRowContext(ctx,
			`SELECT run_id FROM matches GROUP BY run_id
			   ORDER BY MAX(rowid) DESC LIMIT 1`).Scan(&ns)
		switch err {
		case nil:
			runID = ns.String
		case sql.ErrNoRows:
			// no matches yet; leave runID empty
		default:
			return nil, fmt.Errorf("locate latest run: %w", err)
		}
	}

	r := &Report{
		RunID:            runID,
		Generated:        time.Now().UTC(),
		IncludeCallGraph: opts.IncludeCallGraph,
	}

	feats, err := loadFeatures(ctx, d)
	if err != nil {
		return nil, err
	}
	matchesByFeature, err := loadMatches(ctx, d, runID)
	if err != nil {
		return nil, err
	}
	if opts.IncludeCallGraph {
		if err := attachCallGraph(ctx, d, matchesByFeature, opts.MaxCallDepth); err != nil {
			return nil, err
		}
	}

	// Group features by section.
	bySection := map[string][]FeatureCoverage{}
	for _, f := range feats {
		ms := matchesByFeature[f.ID]
		fc := FeatureCoverage{
			ID:      f.ID,
			Title:   f.Title,
			Section: orUnsectioned(f.Section),
			Matches: ms,
			Status:  classify(ms),
		}
		fc.SupportingArtifacts = supportCount(ms)
		for _, m := range ms {
			if m.Tested {
				fc.TestedAny = true
				break
			}
		}
		bySection[fc.Section] = append(bySection[fc.Section], fc)
	}

	for name, list := range bySection {
		s := Section{Name: name, Features: list, Total: len(list)}
		for _, f := range list {
			switch f.Status {
			case StatusImplemented:
				s.Implemented++
			case StatusDrifts:
				s.Drifts++
			case StatusMissing:
				s.Missing++
			}
			s.SupportingArtifacts += f.SupportingArtifacts
		}
		r.Sections = append(r.Sections, s)
	}
	sort.Slice(r.Sections, func(i, j int) bool { return r.Sections[i].Name < r.Sections[j].Name })

	r.Drift, err = loadDrift(ctx, d, runID)
	if err != nil {
		return nil, err
	}
	r.Orphans, err = loadOrphans(ctx, d, runID)
	if err != nil {
		return nil, err
	}
	return r, nil
}

const defaultMaxCallDepth = 3

type featureRow struct {
	ID, Title, Section string
}

func loadFeatures(ctx context.Context, d *db.DB) ([]featureRow, error) {
	rows, err := d.SQL().QueryContext(ctx,
		`SELECT id, title, COALESCE(fsd_section, '') FROM features ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("load features: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []featureRow
	for rows.Next() {
		var f featureRow
		if err := rows.Scan(&f.ID, &f.Title, &f.Section); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func loadMatches(ctx context.Context, d *db.DB, runID string) (map[string][]MatchRow, error) {
	if runID == "" {
		return map[string][]MatchRow{}, nil
	}
	rows, err := d.SQL().QueryContext(ctx,
		`SELECT m.feature_id, m.artifact_id, m.verdict, m.confidence,
		        m.evidence, m.notes, m.model, m.prompt_version,
		        ca.kind, ca.identifier, ca.file, ca.start_line, ca.end_line
		   FROM matches m
		   JOIN code_artifacts ca ON ca.id = m.artifact_id
		  WHERE m.run_id = ?
		  ORDER BY m.feature_id, m.verdict, m.confidence DESC, m.artifact_id`,
		runID)
	if err != nil {
		return nil, fmt.Errorf("load matches: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string][]MatchRow{}
	for rows.Next() {
		var (
			featID, evJSON, notes, model, promptVer string
			m                                       MatchRow
		)
		if err := rows.Scan(&featID, &m.ArtifactID, &m.Verdict, &m.Confidence,
			&evJSON, &notes, &model, &promptVer,
			&m.Kind, &m.Identifier, &m.File, &m.StartLine, &m.EndLine); err != nil {
			return nil, err
		}
		m.Model, m.PromptVersion = model, promptVer
		if evJSON != "" {
			if err := json.Unmarshal([]byte(evJSON), &m.Evidence); err != nil {
				return nil, fmt.Errorf("decode evidence for (%s, %d): %w", featID, m.ArtifactID, err)
			}
		}
		m.Notes, m.Tested, m.TestCount = parseNotesDecorations(notes)
		out[featID] = append(out[featID], m)
	}
	return out, rows.Err()
}

func attachCallGraph(ctx context.Context, d *db.DB, matchesByFeature map[string][]MatchRow, maxDepth int) error {
	if maxDepth <= 0 {
		maxDepth = defaultMaxCallDepth
	}
	edges, err := loadCallGraphEdges(ctx, d)
	if err != nil {
		return err
	}
	if len(edges) == 0 {
		return nil
	}
	for featureID, matches := range matchesByFeature {
		for i := range matches {
			if matches[i].Verdict != "implements" {
				continue
			}
			matches[i].SupportingArtifacts = reachableSupport(matches[i].ArtifactID, edges, maxDepth)
		}
		matchesByFeature[featureID] = matches
	}
	return nil
}

func loadCallGraphEdges(ctx context.Context, d *db.DB) (map[int64][]SupportArtifact, error) {
	rows, err := d.SQL().QueryContext(ctx, `
		SELECT r.from_artifact, r.kind,
		       ca.id, ca.kind, ca.identifier, ca.file, ca.start_line, ca.end_line
		  FROM relationships r
		  JOIN code_artifacts ca ON ca.id = r.to_artifact
		 ORDER BY r.from_artifact, ca.id`)
	if err != nil {
		return nil, fmt.Errorf("load call graph relationships: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[int64][]SupportArtifact{}
	for rows.Next() {
		var (
			from int64
			a    SupportArtifact
		)
		if err := rows.Scan(&from, &a.RelationshipKind, &a.ArtifactID, &a.Kind,
			&a.Identifier, &a.File, &a.StartLine, &a.EndLine); err != nil {
			return nil, err
		}
		out[from] = append(out[from], a)
	}
	return out, rows.Err()
}

func reachableSupport(start int64, edges map[int64][]SupportArtifact, maxDepth int) []SupportArtifact {
	type queued struct {
		id    int64
		depth int
	}
	seen := map[int64]struct{}{start: {}}
	queue := []queued{{id: start}}
	var out []SupportArtifact

	for len(queue) > 0 {
		next := queue[0]
		queue = queue[1:]
		if next.depth >= maxDepth {
			continue
		}
		for _, edge := range edges[next.id] {
			if _, ok := seen[edge.ArtifactID]; ok {
				continue
			}
			seen[edge.ArtifactID] = struct{}{}
			edge.Depth = next.depth + 1
			out = append(out, edge)
			queue = append(queue, queued{id: edge.ArtifactID, depth: edge.Depth})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Depth != out[j].Depth {
			return out[i].Depth < out[j].Depth
		}
		return out[i].ArtifactID < out[j].ArtifactID
	})
	return out
}

func supportCount(ms []MatchRow) int {
	seen := map[int64]struct{}{}
	for _, m := range ms {
		for _, a := range m.SupportingArtifacts {
			seen[a.ArtifactID] = struct{}{}
		}
	}
	return len(seen)
}

func loadDrift(ctx context.Context, d *db.DB, runID string) ([]DriftRow, error) {
	if runID == "" {
		return nil, nil
	}
	rows, err := d.SQL().QueryContext(ctx,
		`SELECT m.feature_id, f.title, COALESCE(f.fsd_section, ''),
		        m.artifact_id, ca.kind, ca.identifier, ca.file,
		        ca.start_line, ca.end_line,
		        m.confidence, m.evidence, m.notes
		   FROM matches m
		   JOIN features f       ON f.id = m.feature_id
		   JOIN code_artifacts ca ON ca.id = m.artifact_id
		  WHERE m.run_id = ? AND m.verdict = 'drifts'
		  ORDER BY m.feature_id, m.confidence DESC, m.artifact_id`,
		runID)
	if err != nil {
		return nil, fmt.Errorf("load drift: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []DriftRow
	for rows.Next() {
		var (
			r             DriftRow
			evJSON, notes string
		)
		if err := rows.Scan(&r.FeatureID, &r.FeatureTitle, &r.Section,
			&r.ArtifactID, &r.Kind, &r.Identifier, &r.File,
			&r.StartLine, &r.EndLine, &r.Confidence, &evJSON, &notes); err != nil {
			return nil, err
		}
		if evJSON != "" {
			if err := json.Unmarshal([]byte(evJSON), &r.Evidence); err != nil {
				return nil, err
			}
		}
		r.Notes, _, _ = parseNotesDecorations(notes)
		out = append(out, r)
	}
	return out, rows.Err()
}

// loadOrphans returns artifacts with no `implements` verdict against
// any feature in this run. Restricted to public-surface kinds — a
// scheduled job or REST endpoint without an FR is meaningful; an
// internal entity row is not.
func loadOrphans(ctx context.Context, d *db.DB, runID string) ([]OrphanRow, error) {
	q := `
		SELECT id, kind, identifier, file, start_line, end_line
		  FROM code_artifacts
		 WHERE kind IN ('rest_endpoint','kafka_listener','rabbit_listener',
		                'jms_listener','event_listener','scheduled_job',
		                'exception_handler')
		   AND id NOT IN (
		       SELECT DISTINCT artifact_id FROM matches
		         WHERE verdict = 'implements' AND run_id = ?
		   )
		 ORDER BY kind, file, start_line
	`
	rows, err := d.SQL().QueryContext(ctx, q, runID)
	if err != nil {
		return nil, fmt.Errorf("load orphans: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []OrphanRow
	for rows.Next() {
		var o OrphanRow
		if err := rows.Scan(&o.ArtifactID, &o.Kind, &o.Identifier,
			&o.File, &o.StartLine, &o.EndLine); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// classify returns the FR-level Status given its matches.
func classify(ms []MatchRow) Status {
	hasDrifts := false
	for _, m := range ms {
		if m.Verdict == "implements" {
			return StatusImplemented
		}
		if m.Verdict == "drifts" {
			hasDrifts = true
		}
	}
	if hasDrifts {
		return StatusDrifts
	}
	return StatusMissing
}

// parseNotesDecorations strips "; tested=X test_count=N" from the
// matcher's notes column and returns the remaining prose plus the
// extracted decoration values.
func parseNotesDecorations(notes string) (cleaned string, tested bool, testCount int) {
	cleaned = notes
	if m := decorationRe.FindStringSubmatch(notes); len(m) == 3 {
		tested = m[1] == "true"
		testCount, _ = strconv.Atoi(m[2])
		cleaned = strings.TrimSpace(decorationRe.ReplaceAllString(notes, ""))
		cleaned = strings.TrimSuffix(cleaned, ";")
		cleaned = strings.TrimSpace(cleaned)
	}
	return
}

var decorationRe = regexp.MustCompile(`;?\s*tested=(true|false)\s+test_count=(\d+)`)

func orUnsectioned(s string) string {
	if s == "" {
		return "(unsectioned)"
	}
	return s
}
