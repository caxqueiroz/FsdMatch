package match

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/llm"
)

// Pipeline runs the per-FR matching pipeline end-to-end and persists
// rows into the matches table.
type Pipeline struct {
	d                 *db.DB
	generator         llm.Generator
	embedder          embed.Embedder
	cache             *embed.Cache
	retriever         *Retriever
	judge             *Judge
	judgmentModel     string
	judgmentMaxToken  int
	judgmentBatchSize int
	matchConcurrency  int
	topK              int
	logger            *slog.Logger
	resume            bool
	progress          func(done, total int)
}

// PipelineOption configures a Pipeline.
type PipelineOption func(*Pipeline)

// WithTopK overrides the default per-FR candidate cap.
func WithTopK(k int) PipelineOption { return func(p *Pipeline) { p.topK = k } }

// WithJudgmentModel overrides the judgment model.
func WithJudgmentModel(m string) PipelineOption {
	return func(p *Pipeline) {
		p.judgmentModel = m
		p.resetJudge()
	}
}

// WithJudgmentMaxTokens overrides the per-call judgment output budget.
func WithJudgmentMaxTokens(n int) PipelineOption {
	return func(p *Pipeline) {
		if n > 0 {
			p.judgmentMaxToken = n
			p.resetJudge()
		}
	}
}

// WithJudgmentBatchSize splits candidate judgment into smaller model
// calls. A value <= 0 keeps the whole candidate set in one call.
func WithJudgmentBatchSize(n int) PipelineOption {
	return func(p *Pipeline) {
		if n > 0 {
			p.judgmentBatchSize = n
		}
	}
}

// WithMatchConcurrency controls how many FRs may be matched in parallel.
// Values <= 1 keep the historical serial behavior.
func WithMatchConcurrency(n int) PipelineOption {
	return func(p *Pipeline) {
		if n > 1 {
			p.matchConcurrency = n
		} else {
			p.matchConcurrency = 1
		}
	}
}

// WithResume skips features that already have match rows for the run.
func WithResume(resume bool) PipelineOption {
	return func(p *Pipeline) { p.resume = resume }
}

// WithProgress receives progress updates after a feature is skipped or
// successfully matched.
func WithProgress(fn func(done, total int)) PipelineOption {
	return func(p *Pipeline) { p.progress = fn }
}

// WithLogger overrides the default logger.
func WithLogger(l *slog.Logger) PipelineOption {
	return func(p *Pipeline) { p.logger = l }
}

// NewPipeline constructs a Pipeline using shared infrastructure.
func NewPipeline(d *db.DB, generator llm.Generator, embedder embed.Embedder, opts ...PipelineOption) *Pipeline {
	p := &Pipeline{
		d:                d,
		generator:        generator,
		embedder:         embedder,
		cache:            embed.NewCache(d),
		retriever:        NewRetriever(d),
		judgmentMaxToken: 4096,
		matchConcurrency: 1,
		topK:             DefaultTopK,
		logger:           slog.Default(),
	}
	p.resetJudge()
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *Pipeline) resetJudge() {
	p.judge = NewJudge(p.generator, p.judgmentModel, WithJudgeMaxTokens(p.judgmentMaxToken))
}

// FeatureRow is a hydrated features row used by the pipeline.
type FeatureRow struct {
	ID          string
	Title       string
	Description string
	Acceptance  []string
	Section     string
}

// RejudgeDrifts re-evaluates every "drifts" verdict produced by an
// earlier run using a stronger judgment model. Each drifts
// (feature, artifact) pair is re-judged in a fresh model call
// and the matches row is updated in place under the same run_id.
//
// SPEC §7.4 wording: "drifts verdicts may be re-judged by Opus in a
// second pass (gated by --rejudge-drifts)".
func (p *Pipeline) RejudgeDrifts(ctx context.Context, runID, opusModel string) (*RejudgeSummary, error) {
	if runID == "" {
		return nil, fmt.Errorf("rejudge: run id required")
	}
	if opusModel == "" {
		opusModel = p.judge.Model()
	}
	rejudge := NewJudge(p.generator, opusModel, WithJudgeMaxTokens(p.judgmentMaxToken))

	// 1. Collect all drift candidates for the run, grouped by feature.
	pairs, err := p.loadDriftCandidates(ctx, runID)
	if err != nil {
		return nil, err
	}
	summary := &RejudgeSummary{RunID: runID, Model: opusModel}
	if len(pairs) == 0 {
		return summary, nil
	}

	// 2. For each feature, re-fetch its row, replay the prompt against
	//    the stronger judge with only its drift candidates, and persist.
	for featureID, candidates := range pairs {
		fr, err := p.loadOneFeature(ctx, featureID)
		if err != nil {
			return summary, fmt.Errorf("rejudge load %s: %w", featureID, err)
		}
		matches, err := p.judgeCandidatesWith(ctx, rejudge, FRSnapshot(fr), nil, candidates)
		if err != nil {
			return summary, fmt.Errorf("rejudge call %s: %w", featureID, err)
		}
		if err := p.decorateTests(ctx, matches); err != nil {
			return summary, err
		}
		if err := p.writeMatches(ctx, runID, matches); err != nil {
			return summary, err
		}
		for _, m := range matches {
			summary.Total++
			switch m.Verdict {
			case VerdictImplements:
				summary.PromotedToImplements++
			case VerdictDrifts:
				summary.StillDrifts++
			case VerdictUnrelated:
				summary.DowngradedToUnrelated++
			}
		}
		p.logger.InfoContext(ctx, "rejudged drifts for FR",
			"id", featureID, "candidates", len(matches),
			"model", opusModel)
	}
	return summary, nil
}

// RejudgeSummary aggregates the second-pass outcome.
type RejudgeSummary struct {
	RunID                 string
	Model                 string
	Total                 int
	PromotedToImplements  int
	StillDrifts           int
	DowngradedToUnrelated int
}

// loadDriftCandidates returns featureID → candidate slice for every
// drifts row in the run. The artifact row is hydrated so the rejudge
// prompt is identical in shape to the first pass.
func (p *Pipeline) loadDriftCandidates(ctx context.Context, runID string) (map[string][]ArtifactCandidate, error) {
	rows, err := p.d.SQL().QueryContext(ctx, `
		SELECT m.feature_id,
		       ca.id, ca.kind, ca.identifier,
		       COALESCE(ca.package, ''), COALESCE(ca.class, ''), COALESCE(ca.method, ''),
		       ca.file, ca.start_line, ca.end_line,
		       COALESCE(ca.signature, ''), COALESCE(ca.source, '')
		  FROM matches m
		  JOIN code_artifacts ca ON ca.id = m.artifact_id
		 WHERE m.run_id = ? AND m.verdict = 'drifts'
		 ORDER BY m.feature_id, m.artifact_id`, runID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := map[string][]ArtifactCandidate{}
	for rows.Next() {
		var (
			featID string
			c      ArtifactCandidate
		)
		if err := rows.Scan(&featID,
			&c.ID, &c.Kind, &c.Identifier,
			&c.Package, &c.Class, &c.Method,
			&c.File, &c.StartLine, &c.EndLine,
			&c.Signature, &c.Source,
		); err != nil {
			return nil, err
		}
		out[featID] = append(out[featID], c)
	}
	return out, rows.Err()
}

func (p *Pipeline) loadOneFeature(ctx context.Context, id string) (FeatureRow, error) {
	var (
		r       FeatureRow
		accJSON string
	)
	err := p.d.SQL().QueryRowContext(ctx,
		`SELECT id, title, description, acceptance, COALESCE(fsd_section, '') FROM features WHERE id = ?`,
		id).Scan(&r.ID, &r.Title, &r.Description, &accJSON, &r.Section)
	if err != nil {
		return r, err
	}
	if accJSON != "" {
		if err := json.Unmarshal([]byte(accJSON), &r.Acceptance); err != nil {
			return r, err
		}
	}
	return r, nil
}

// MatchAll runs the pipeline over every feature in the DB (or only
// featureIDs when non-empty) and writes the results to matches.
// Re-runs are idempotent: prior rows for (run_id, feature_id, *) are
// replaced via INSERT-OR-REPLACE.
func (p *Pipeline) MatchAll(ctx context.Context, runID string, featureIDs []string) (*RunSummary, error) {
	if runID == "" {
		runID = fmt.Sprintf("match-%d", time.Now().Unix())
	}
	frs, err := p.loadFeatures(ctx, featureIDs)
	if err != nil {
		return nil, err
	}
	summary := &RunSummary{RunID: runID}
	total := len(frs)
	done := 0
	if p.resume {
		completed, err := p.featuresWithMatches(ctx, runID)
		if err != nil {
			return nil, err
		}
		todo := frs[:0]
		for _, fr := range frs {
			if _, ok := completed[fr.ID]; ok {
				summary.SkippedFeatures++
				done++
				p.reportProgress(done, total)
				continue
			}
			todo = append(todo, fr)
		}
		frs = todo
	}
	if p.matchConcurrency > 1 && len(frs) > 1 {
		return p.matchAllConcurrent(ctx, runID, frs, summary, done, total)
	}
	for _, fr := range frs {
		matches, err := p.MatchFeature(ctx, fr)
		if err != nil {
			return summary, fmt.Errorf("match %s: %w", fr.ID, err)
		}
		if err := p.persistFeatureMatches(ctx, runID, fr, matches, summary); err != nil {
			return summary, err
		}
		done++
		p.reportProgress(done, total)
	}
	return summary, nil
}

type featureMatchResult struct {
	feature FeatureRow
	matches []Match
	err     error
}

func (p *Pipeline) matchAllConcurrent(
	ctx context.Context,
	runID string,
	frs []FeatureRow,
	summary *RunSummary,
	done int,
	total int,
) (*RunSummary, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	workers := p.matchConcurrency
	if workers > len(frs) {
		workers = len(frs)
	}
	jobs := make(chan FeatureRow)
	results := make(chan featureMatchResult)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for fr := range jobs {
				if ctx.Err() != nil {
					return
				}
				matches, err := p.MatchFeature(ctx, fr)
				select {
				case results <- featureMatchResult{feature: fr, matches: matches, err: err}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, fr := range frs {
			select {
			case jobs <- fr:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	var firstErr error
	for res := range results {
		if res.err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("match %s: %w", res.feature.ID, res.err)
				cancel()
			}
			continue
		}
		if firstErr != nil {
			continue
		}
		if err := p.persistFeatureMatches(ctx, runID, res.feature, res.matches, summary); err != nil {
			if firstErr == nil {
				firstErr = err
				cancel()
			}
		}
		done++
		p.reportProgress(done, total)
	}
	if firstErr != nil {
		return summary, firstErr
	}
	return summary, nil
}

func (p *Pipeline) reportProgress(done, total int) {
	if p.progress != nil {
		p.progress(done, total)
	}
}

func (p *Pipeline) persistFeatureMatches(
	ctx context.Context,
	runID string,
	fr FeatureRow,
	matches []Match,
	summary *RunSummary,
) error {
	if err := p.writeMatches(ctx, runID, matches); err != nil {
		return fmt.Errorf("write %s: %w", fr.ID, err)
	}
	summary.absorb(matches)
	p.logger.InfoContext(ctx, "matched FR",
		"id", fr.ID, "candidates", len(matches),
		"implements", countVerdict(matches, VerdictImplements),
		"drifts", countVerdict(matches, VerdictDrifts),
		"unrelated", countVerdict(matches, VerdictUnrelated),
	)
	return nil
}

// MatchFeature runs the pipeline for one FR and returns the per-candidate
// matches without persisting them.
func (p *Pipeline) MatchFeature(ctx context.Context, fr FeatureRow) ([]Match, error) {
	snap := FRSnapshot(fr)
	stub := frStubForAnchors{snap}
	anchors := ExtractAnchors(stub)

	queryVec, err := embed.Cached(ctx, p.embedder, p.cache, embeddingTextFor(snap))
	if err != nil {
		return nil, fmt.Errorf("embed FR: %w", err)
	}

	candidates, err := p.retriever.RetrieveRanked(ctx, queryVec, anchors, embeddingTextFor(snap), p.topK)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	matches, err := p.judgeCandidates(ctx, snap, anchors, candidates)
	if err != nil {
		return nil, err
	}
	if err := p.decorateTests(ctx, matches); err != nil {
		return nil, err
	}
	return matches, nil
}

func (p *Pipeline) judgeCandidates(
	ctx context.Context,
	fr FRSnapshot,
	anchors []Anchor,
	candidates []ArtifactCandidate,
) ([]Match, error) {
	return p.judgeCandidatesWith(ctx, p.judge, fr, anchors, candidates)
}

func (p *Pipeline) judgeCandidatesWith(
	ctx context.Context,
	judge *Judge,
	fr FRSnapshot,
	anchors []Anchor,
	candidates []ArtifactCandidate,
) ([]Match, error) {
	if len(candidates) == 0 {
		return nil, nil
	}
	batchSize := p.judgmentBatchSize
	if batchSize <= 0 || batchSize >= len(candidates) {
		return p.judgeBatchWithRetry(ctx, judge, fr, anchors, candidates)
	}
	out := make([]Match, 0, len(candidates))
	for start := 0; start < len(candidates); start += batchSize {
		end := start + batchSize
		if end > len(candidates) {
			end = len(candidates)
		}
		matches, err := p.judgeBatchWithRetry(ctx, judge, fr, anchors, candidates[start:end])
		if err != nil {
			return nil, err
		}
		out = append(out, matches...)
	}
	return out, nil
}

func (p *Pipeline) judgeBatchWithRetry(
	ctx context.Context,
	judge *Judge,
	fr FRSnapshot,
	anchors []Anchor,
	candidates []ArtifactCandidate,
) ([]Match, error) {
	matches, err := judge.JudgeFeature(ctx, fr, anchors, candidates)
	if err == nil {
		return matches, nil
	}
	if !llm.IsIncomplete(err) || len(candidates) == 1 {
		return nil, err
	}
	mid := len(candidates) / 2
	left, err := p.judgeBatchWithRetry(ctx, judge, fr, anchors, candidates[:mid])
	if err != nil {
		return nil, err
	}
	right, err := p.judgeBatchWithRetry(ctx, judge, fr, anchors, candidates[mid:])
	if err != nil {
		return nil, err
	}
	return append(left, right...), nil
}

func embeddingTextFor(s FRSnapshot) string {
	var b strings.Builder
	b.WriteString(s.Title)
	b.WriteString("\n")
	b.WriteString(s.Description)
	for _, c := range s.Acceptance {
		b.WriteString("\n- ")
		b.WriteString(c)
	}
	return b.String()
}

func (p *Pipeline) loadFeatures(ctx context.Context, only []string) ([]FeatureRow, error) {
	q := `SELECT id, title, description, acceptance, COALESCE(fsd_section, '') FROM features`
	args := []any{}
	if len(only) > 0 {
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(only)), ",")
		q += " WHERE id IN (" + placeholders + ")"
		for _, id := range only {
			args = append(args, id)
		}
	}
	q += " ORDER BY id"
	rows, err := p.d.SQL().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []FeatureRow
	for rows.Next() {
		var r FeatureRow
		var accJSON string
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &accJSON, &r.Section); err != nil {
			return nil, err
		}
		if accJSON != "" {
			if err := json.Unmarshal([]byte(accJSON), &r.Acceptance); err != nil {
				return nil, fmt.Errorf("decode acceptance for %s: %w", r.ID, err)
			}
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (p *Pipeline) featuresWithMatches(ctx context.Context, runID string) (map[string]struct{}, error) {
	rows, err := p.d.SQL().QueryContext(ctx,
		`SELECT DISTINCT feature_id FROM matches WHERE run_id = ?`, runID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = struct{}{}
	}
	return out, rows.Err()
}

// decorateTests fills Match.Tested/TestCount by counting test rows
// pointing at each artifact id in the result set.
func (p *Pipeline) decorateTests(ctx context.Context, matches []Match) error {
	if len(matches) == 0 {
		return nil
	}
	ids := make([]any, 0, len(matches))
	for _, m := range matches {
		ids = append(ids, m.ArtifactID)
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	rows, err := p.d.SQL().QueryContext(ctx,
		`SELECT target_artifact, COUNT(*) FROM tests
		   WHERE target_artifact IN (`+placeholders+`)
		   GROUP BY target_artifact`, ids...)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	counts := map[int64]int{}
	for rows.Next() {
		var artID int64
		var n int
		if err := rows.Scan(&artID, &n); err != nil {
			return err
		}
		counts[artID] = n
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i := range matches {
		matches[i].TestCount = counts[matches[i].ArtifactID]
		matches[i].Tested = matches[i].TestCount > 0
	}
	return nil
}

func (p *Pipeline) writeMatches(ctx context.Context, runID string, matches []Match) error {
	if len(matches) == 0 {
		return nil
	}
	return p.d.Writer().Submit(ctx, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO matches
			  (run_id, feature_id, artifact_id, verdict, confidence,
			   evidence, notes, model, prompt_version)
			VALUES (?,?,?,?,?,?,?,?,?)
			ON CONFLICT(run_id, feature_id, artifact_id) DO UPDATE SET
			  verdict        = excluded.verdict,
			  confidence     = excluded.confidence,
			  evidence       = excluded.evidence,
			  notes          = excluded.notes,
			  model          = excluded.model,
			  prompt_version = excluded.prompt_version
		`)
		if err != nil {
			return err
		}
		defer func() { _ = stmt.Close() }()

		for _, m := range matches {
			ev := m.Evidence
			if ev == nil {
				ev = []Evidence{}
			}
			evJSON, err := json.Marshal(ev)
			if err != nil {
				return err
			}
			notes := m.Notes
			// Persist tested/test_count in notes since the table has no
			// dedicated columns. Reporter parses this back out.
			if notes != "" {
				notes += "; "
			}
			notes += fmt.Sprintf("tested=%v test_count=%d", m.Tested, m.TestCount)

			if _, err := stmt.ExecContext(ctx,
				runID, m.FeatureID, m.ArtifactID,
				m.Verdict, m.Confidence,
				string(evJSON), notes, m.Model, m.PromptVersion,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

// RunSummary aggregates per-verdict counts for one match run.
type RunSummary struct {
	RunID           string
	TotalMatches    int
	Implements      int
	Drifts          int
	Unrelated       int
	Tested          int
	SkippedFeatures int
}

func (s *RunSummary) absorb(matches []Match) {
	for _, m := range matches {
		s.TotalMatches++
		switch m.Verdict {
		case VerdictImplements:
			s.Implements++
		case VerdictDrifts:
			s.Drifts++
		case VerdictUnrelated:
			s.Unrelated++
		}
		if m.Tested {
			s.Tested++
		}
	}
}

func countVerdict(matches []Match, v string) int {
	n := 0
	for _, m := range matches {
		if m.Verdict == v {
			n++
		}
	}
	return n
}

// frStubForAnchors lets internal/match consume an FRSnapshot via the
// FeatureLike interface in anchor.go without exposing a wider type.
type frStubForAnchors struct{ s FRSnapshot }

func (f frStubForAnchors) Title() string        { return f.s.Title }
func (f frStubForAnchors) Description() string  { return f.s.Description }
func (f frStubForAnchors) Acceptance() []string { return f.s.Acceptance }
