package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/llm"
	"github.com/cax/fsdtrace/internal/match"
	"github.com/cax/fsdtrace/internal/report"
)

const openAIJudgmentMaxTokens = 12000
const openAIJudgmentBatchSize = 8

// ----- search_features ---------------------------------------------------

type featureHit struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Section    string   `json:"section"`
	Acceptance []string `json:"acceptance"`
	Distance   float64  `json:"distance"`
}

func (s *Server) handleSearchFeatures(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return resultErr(err), nil
	}
	topK := int(req.GetFloat("top_k", 10))
	if topK <= 0 {
		topK = 10
	}

	d, err := s.DB(ctx)
	if err != nil {
		return resultErr(err), nil
	}
	_, emb, err := s.Models(ctx)
	if err != nil {
		return resultErr(err), nil
	}
	cache := embed.NewCache(d)
	v, err := embed.Cached(ctx, emb, cache, query)
	if err != nil {
		return resultErr(fmt.Errorf("embed query: %w", err)), nil
	}

	rows, err := d.SQL().QueryContext(ctx, `
		SELECT f.id, f.title, COALESCE(f.fsd_section, ''),
		       f.acceptance, fv.distance
		  FROM feature_vec fv
		  JOIN features f ON f.id = (
		       SELECT id FROM features WHERE rowid = fv.rowid
		  )
		 WHERE fv.embedding MATCH ? AND fv.k = ?
		 ORDER BY fv.distance`,
		db.PackFloat32(v), topK)
	if err != nil {
		// Fall back to a simpler join: features.id is TEXT and not indexed
		// to vec rowid 1:1; the matcher uses FNV-64. Re-issue without the
		// JOIN so even if the rowid mapping breaks we return *something*.
		rows, err = d.SQL().QueryContext(ctx, `
			SELECT rowid, distance FROM feature_vec
			 WHERE embedding MATCH ? AND k = ?
			 ORDER BY distance`, db.PackFloat32(v), topK)
		if err != nil {
			return resultErr(fmt.Errorf("knn: %w", err)), nil
		}
	}
	defer func() { _ = rows.Close() }()

	hits, err := scanFeatureHitsByRowID(ctx, d, rows)
	if err != nil {
		return resultErr(err), nil
	}
	return mcpgo.NewToolResultStructured(hits, jsonString(hits)), nil
}

// scanFeatureHitsByRowID joins KNN rowids back to feature rows by
// recomputing the FNV-64 hash. Works whether the SQL above used the
// primary join form or the fallback rowid-only form.
func scanFeatureHitsByRowID(ctx context.Context, d *db.DB, rows *sql.Rows) ([]featureHit, error) {
	type rowidHit struct {
		rowid    int64
		distance float64
	}
	var pending []rowidHit
	for rows.Next() {
		// Try 5-column form first; fall through to 2-column form on type
		// mismatch via Scan errors.
		var (
			id, title, section, accJSON string
			rowid                       int64
			dist                        float64
		)
		if err := rows.Scan(&id, &title, &section, &accJSON, &dist); err == nil {
			h := featureHit{ID: id, Title: title, Section: section, Distance: dist}
			if accJSON != "" {
				if err := json.Unmarshal([]byte(accJSON), &h.Acceptance); err != nil {
					return nil, err
				}
			}
			// Append directly; will rebuild fully below.
			pending = append(pending, rowidHit{rowid: -1, distance: dist})
			_ = h // we'll re-collect via second pass when fallback used
			continue
		}
		if err := rows.Scan(&rowid, &dist); err != nil {
			return nil, err
		}
		pending = append(pending, rowidHit{rowid: rowid, distance: dist})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Resolve rowid → feature row by recomputing FNV-64 for each
	// features.id and matching against the KNN rowids we collected.
	if len(pending) == 0 {
		return nil, nil
	}
	idByRow, err := featureIDByRowID(ctx, d)
	if err != nil {
		return nil, err
	}
	out := make([]featureHit, 0, len(pending))
	for _, p := range pending {
		fid, ok := idByRow[p.rowid]
		if !ok {
			continue
		}
		var (
			title, section, accJSON string
		)
		err := d.SQL().QueryRowContext(ctx,
			`SELECT title, COALESCE(fsd_section, ''), acceptance FROM features WHERE id = ?`,
			fid).Scan(&title, &section, &accJSON)
		if err != nil {
			continue
		}
		h := featureHit{ID: fid, Title: title, Section: section, Distance: p.distance}
		if accJSON != "" {
			_ = json.Unmarshal([]byte(accJSON), &h.Acceptance)
		}
		out = append(out, h)
	}
	return out, nil
}

// featureIDByRowID returns a rowid → feature.id map by recomputing the
// FNV-64 hash that the atomizer used. Cheap for thousands of features.
func featureIDByRowID(ctx context.Context, d *db.DB) (map[int64]string, error) {
	rows, err := d.SQL().QueryContext(ctx, `SELECT id FROM features`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[int64]string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[fnv64(id)] = id
	}
	return out, rows.Err()
}

func fnv64(s string) int64 {
	const offset = 1469598103934665603
	const prime = 1099511628211
	h := uint64(offset)
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= prime
	}
	r := int64(h & 0x7fffffffffffffff)
	if r == 0 {
		r = 1
	}
	return r
}

// ----- search_code -------------------------------------------------------

type codeHit struct {
	ID         int64   `json:"id"`
	Kind       string  `json:"kind"`
	Identifier string  `json:"identifier"`
	File       string  `json:"file"`
	StartLine  int     `json:"start_line"`
	EndLine    int     `json:"end_line"`
	Signature  string  `json:"signature"`
	Distance   float64 `json:"distance"`
}

func (s *Server) handleSearchCode(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return resultErr(err), nil
	}
	kind := req.GetString("kind", "")
	topK := int(req.GetFloat("top_k", 10))
	if topK <= 0 {
		topK = 10
	}

	d, err := s.DB(ctx)
	if err != nil {
		return resultErr(err), nil
	}
	_, emb, err := s.Models(ctx)
	if err != nil {
		return resultErr(err), nil
	}
	cache := embed.NewCache(d)
	v, err := embed.Cached(ctx, emb, cache, query)
	if err != nil {
		return resultErr(fmt.Errorf("embed query: %w", err)), nil
	}

	q := `
		SELECT av.rowid, av.distance,
		       ca.kind, ca.identifier, ca.file, ca.start_line, ca.end_line,
		       COALESCE(ca.signature, '')
		  FROM artifact_vec av
		  JOIN code_artifacts ca ON ca.id = av.rowid
		 WHERE av.embedding MATCH ? AND av.k = ?`
	args := []any{db.PackFloat32(v), topK}
	if kind != "" {
		// vec0 doesn't accept extra WHERE on the same statement easily;
		// over-fetch and filter in Go to keep the SQL simple.
		args = []any{db.PackFloat32(v), topK * 3}
	}
	q += ` ORDER BY av.distance`
	rows, err := d.SQL().QueryContext(ctx, q, args...)
	if err != nil {
		return resultErr(fmt.Errorf("knn: %w", err)), nil
	}
	defer func() { _ = rows.Close() }()

	out := make([]codeHit, 0, topK)
	for rows.Next() {
		var h codeHit
		if err := rows.Scan(&h.ID, &h.Distance,
			&h.Kind, &h.Identifier, &h.File, &h.StartLine, &h.EndLine, &h.Signature); err != nil {
			return resultErr(err), nil
		}
		if kind != "" && h.Kind != kind {
			continue
		}
		out = append(out, h)
		if len(out) >= topK {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return resultErr(err), nil
	}
	return mcpgo.NewToolResultStructured(out, jsonString(out)), nil
}

// ----- get_feature -------------------------------------------------------

type featureDetail struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Section        string            `json:"section"`
	Description    string            `json:"description"`
	Acceptance     []string          `json:"acceptance"`
	CoverageStatus string            `json:"coverage_status"`
	RunID          string            `json:"run_id"`
	Matches        []report.MatchRow `json:"matches"`
}

func (s *Server) handleGetFeature(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return resultErr(err), nil
	}
	runID := req.GetString("run_id", "")

	d, err := s.DB(ctx)
	if err != nil {
		return resultErr(err), nil
	}

	var (
		title, section, descr, accJSON string
	)
	err = d.SQL().QueryRowContext(ctx,
		`SELECT title, COALESCE(fsd_section, ''), description, acceptance
		   FROM features WHERE id = ?`, id).Scan(&title, &section, &descr, &accJSON)
	if err != nil {
		return resultErr(fmt.Errorf("feature %s: %w", id, err)), nil
	}
	rep, err := report.Load(ctx, d, runID)
	if err != nil {
		return resultErr(err), nil
	}

	det := featureDetail{ID: id, Title: title, Section: orUnsectioned(section), Description: descr, RunID: rep.RunID}
	if accJSON != "" {
		_ = json.Unmarshal([]byte(accJSON), &det.Acceptance)
	}
	for _, sec := range rep.Sections {
		for _, fc := range sec.Features {
			if fc.ID == id {
				det.CoverageStatus = string(fc.Status)
				det.Matches = fc.Matches
				break
			}
		}
	}
	return mcpgo.NewToolResultStructured(det, jsonString(det)), nil
}

// ----- get_artifact ------------------------------------------------------

type artifactDetail struct {
	ID             int64           `json:"id"`
	Kind           string          `json:"kind"`
	Identifier     string          `json:"identifier"`
	File           string          `json:"file"`
	StartLine      int             `json:"start_line"`
	EndLine        int             `json:"end_line"`
	Signature      string          `json:"signature"`
	Annotations    json.RawMessage `json:"annotations"`
	LinkedFeatures []linkedFR      `json:"linked_features"`
	Tests          []linkedTest    `json:"tests"`
	RunID          string          `json:"run_id"`
}

type linkedFR struct {
	FeatureID  string  `json:"feature_id"`
	Title      string  `json:"title"`
	Verdict    string  `json:"verdict"`
	Confidence float64 `json:"confidence"`
}

type linkedTest struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	TestKind string `json:"test_kind"`
}

func (s *Server) handleGetArtifact(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	idF, err := req.RequireFloat("id")
	if err != nil {
		return resultErr(err), nil
	}
	id := int64(idF)
	runID := req.GetString("run_id", "")

	d, err := s.DB(ctx)
	if err != nil {
		return resultErr(err), nil
	}

	a := artifactDetail{ID: id}
	var (
		sig, ann sql.NullString
	)
	err = d.SQL().QueryRowContext(ctx,
		`SELECT kind, identifier, file, start_line, end_line, signature, annotations
		   FROM code_artifacts WHERE id = ?`, id,
	).Scan(&a.Kind, &a.Identifier, &a.File, &a.StartLine, &a.EndLine, &sig, &ann)
	if err != nil {
		return resultErr(fmt.Errorf("artifact %d: %w", id, err)), nil
	}
	a.Signature = sig.String
	if ann.Valid && ann.String != "" {
		a.Annotations = json.RawMessage(ann.String)
	}

	// Linked features via matches.
	if runID == "" {
		runID = mostRecentMatchRun(ctx, d)
	}
	a.RunID = runID
	if runID != "" {
		rows, err := d.SQL().QueryContext(ctx,
			`SELECT m.feature_id, COALESCE(f.title, ''), m.verdict, m.confidence
			   FROM matches m
			   JOIN features f ON f.id = m.feature_id
			  WHERE m.run_id = ? AND m.artifact_id = ?
			  ORDER BY m.verdict, m.confidence DESC`,
			runID, id)
		if err != nil {
			return resultErr(err), nil
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var l linkedFR
			if err := rows.Scan(&l.FeatureID, &l.Title, &l.Verdict, &l.Confidence); err != nil {
				return resultErr(err), nil
			}
			a.LinkedFeatures = append(a.LinkedFeatures, l)
		}
		if err := rows.Err(); err != nil {
			return resultErr(err), nil
		}
	}

	// Tests targeting this artifact.
	rows, err := d.SQL().QueryContext(ctx,
		`SELECT id, name, file, line, COALESCE(test_kind, '')
		   FROM tests WHERE target_artifact = ?`, id)
	if err != nil {
		return resultErr(err), nil
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var t linkedTest
		if err := rows.Scan(&t.ID, &t.Name, &t.File, &t.Line, &t.TestKind); err != nil {
			return resultErr(err), nil
		}
		a.Tests = append(a.Tests, t)
	}
	if err := rows.Err(); err != nil {
		return resultErr(err), nil
	}
	return mcpgo.NewToolResultStructured(a, jsonString(a)), nil
}

// ----- list_unmatched ----------------------------------------------------

type unmatchedRow struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Section string `json:"section"`
	Status  string `json:"status"`
}

func (s *Server) handleListUnmatched(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	section := req.GetString("section", "")
	runID := req.GetString("run_id", "")

	d, err := s.DB(ctx)
	if err != nil {
		return resultErr(err), nil
	}
	rep, err := report.Load(ctx, d, runID)
	if err != nil {
		return resultErr(err), nil
	}
	out := []unmatchedRow{}
	for _, sec := range rep.Sections {
		if section != "" && !strings.EqualFold(sec.Name, section) {
			continue
		}
		for _, fc := range sec.Features {
			if fc.Status == report.StatusImplemented {
				continue
			}
			out = append(out, unmatchedRow{ID: fc.ID, Title: fc.Title, Section: sec.Name, Status: string(fc.Status)})
		}
	}
	return mcpgo.NewToolResultStructured(out, jsonString(out)), nil
}

// ----- list_orphans ------------------------------------------------------

func (s *Server) handleListOrphans(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	kind := req.GetString("kind", "")
	runID := req.GetString("run_id", "")

	d, err := s.DB(ctx)
	if err != nil {
		return resultErr(err), nil
	}
	rep, err := report.Load(ctx, d, runID)
	if err != nil {
		return resultErr(err), nil
	}
	out := rep.Orphans
	if kind != "" {
		filtered := out[:0]
		for _, o := range rep.Orphans {
			if o.Kind == kind {
				filtered = append(filtered, o)
			}
		}
		out = filtered
	}
	return mcpgo.NewToolResultStructured(out, jsonString(out)), nil
}

// ----- drift_report ------------------------------------------------------

func (s *Server) handleDriftReport(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	runID := req.GetString("run_id", "")
	d, err := s.DB(ctx)
	if err != nil {
		return resultErr(err), nil
	}
	rep, err := report.Load(ctx, d, runID)
	if err != nil {
		return resultErr(err), nil
	}
	return mcpgo.NewToolResultStructured(rep.Drift, jsonString(rep.Drift)), nil
}

// ----- coverage_summary --------------------------------------------------

type sectionSummary struct {
	Name        string `json:"name"`
	Total       int    `json:"total"`
	Implemented int    `json:"implemented"`
	Drifts      int    `json:"drifts"`
	Missing     int    `json:"missing"`
	Untested    int    `json:"untested"`
}

func (s *Server) handleCoverageSummary(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	runID := req.GetString("run_id", "")
	d, err := s.DB(ctx)
	if err != nil {
		return resultErr(err), nil
	}
	rep, err := report.Load(ctx, d, runID)
	if err != nil {
		return resultErr(err), nil
	}
	out := make([]sectionSummary, 0, len(rep.Sections))
	for _, sec := range rep.Sections {
		untested := 0
		for _, fc := range sec.Features {
			if fc.Status == report.StatusImplemented && !fc.TestedAny {
				untested++
			}
		}
		out = append(out, sectionSummary{
			Name: sec.Name, Total: sec.Total,
			Implemented: sec.Implemented, Drifts: sec.Drifts,
			Missing: sec.Missing, Untested: untested,
		})
	}
	return mcpgo.NewToolResultStructured(out, jsonString(out)), nil
}

// ----- rematch_feature ---------------------------------------------------

func (s *Server) handleRematchFeature(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	id, err := req.RequireString("id")
	if err != nil {
		return resultErr(err), nil
	}
	runID := req.GetString("run_id", "")
	if runID == "" {
		runID = fmt.Sprintf("rematch-%d", time.Now().Unix())
	}
	topK := int(req.GetFloat("top_k", float64(match.DefaultTopK)))

	d, err := s.DB(ctx)
	if err != nil {
		return resultErr(err), nil
	}
	generator, emb, err := s.Models(ctx)
	if err != nil {
		return resultErr(err), nil
	}
	jModel := s.cfg.JudgmentModel
	if jModel == "" {
		if providerName(s.cfg.Provider) == ProviderOpenAI {
			jModel = llm.DefaultOpenAIModel
		} else {
			jModel = match.DefaultJudgmentModel
		}
	}
	pipeOpts := []match.PipelineOption{
		match.WithTopK(topK),
		match.WithJudgmentModel(jModel),
	}
	if providerName(s.cfg.Provider) == ProviderOpenAI {
		pipeOpts = append(pipeOpts,
			match.WithJudgmentMaxTokens(openAIJudgmentMaxTokens),
			match.WithJudgmentBatchSize(openAIJudgmentBatchSize),
		)
	}
	pipe := match.NewPipeline(d, generator, emb, pipeOpts...)
	summary, err := pipe.MatchAll(ctx, runID, []string{id})
	if err != nil {
		return resultErr(err), nil
	}
	res := map[string]any{
		"run_id":     summary.RunID,
		"feature_id": id,
		"total":      summary.TotalMatches,
		"implements": summary.Implements,
		"drifts":     summary.Drifts,
		"unrelated":  summary.Unrelated,
		"tested":     summary.Tested,
		"top_k":      topK,
	}
	return mcpgo.NewToolResultStructured(res, jsonString(res)), nil
}

// ----- helpers -----------------------------------------------------------

func mostRecentMatchRun(ctx context.Context, d *db.DB) string {
	var ns sql.NullString
	err := d.SQL().QueryRowContext(ctx,
		`SELECT run_id FROM matches GROUP BY run_id
		   ORDER BY MAX(rowid) DESC LIMIT 1`).Scan(&ns)
	if err != nil {
		return ""
	}
	return ns.String
}

func orUnsectioned(s string) string {
	if s == "" {
		return "(unsectioned)"
	}
	return s
}

func jsonString(v any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
