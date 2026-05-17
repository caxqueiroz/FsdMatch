package match

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/cax/fsdtrace/internal/db"
)

// DefaultTopK matches SPEC §7.4.
const DefaultTopK = 15

// Retriever combines anchor-based filtering with vec0 KNN to assemble
// candidate artifacts for one FR.
type Retriever struct {
	d *db.DB
}

// NewRetriever builds a Retriever bound to the given DB.
func NewRetriever(d *db.DB) *Retriever { return &Retriever{d: d} }

// Retrieve returns up to topK candidates for queryVec, optionally
// pre-seeded with artifacts that match any anchor. Anchored candidates
// always appear first, then vec0 KNN fills the remaining slots.
func (r *Retriever) Retrieve(ctx context.Context, queryVec []float32, anchors []Anchor, topK int) ([]ArtifactCandidate, error) {
	return r.RetrieveRanked(ctx, queryVec, anchors, "", topK)
}

// RetrieveRanked returns up to topK candidates after collecting a larger
// internal candidate pool and applying deterministic reranking signals.
func (r *Retriever) RetrieveRanked(
	ctx context.Context,
	queryVec []float32,
	anchors []Anchor,
	queryText string,
	topK int,
) ([]ArtifactCandidate, error) {
	if topK <= 0 {
		topK = DefaultTopK
	}

	cands := make([]ArtifactCandidate, 0, candidatePoolSize(topK))
	seen := map[int64]struct{}{}

	// 1. Anchor-matched artifacts enter the pool first.
	if len(anchors) > 0 {
		anchored, err := r.byAnchor(ctx, anchors)
		if err != nil {
			return nil, err
		}
		for _, c := range anchored {
			if _, ok := seen[c.ID]; ok {
				continue
			}
			c.Anchored = true
			seen[c.ID] = struct{}{}
			cands = append(cands, c)
		}
	}

	// 2. Fill the pool via vec0 KNN.
	poolK := candidatePoolSize(topK)
	if len(cands) < poolK {
		need := poolK - len(cands)
		// Over-fetch slightly so dedup against anchored doesn't shrink us.
		knn, err := r.byKNN(ctx, queryVec, need+len(seen))
		if err != nil {
			return nil, err
		}
		for _, c := range knn {
			if len(cands) >= topK {
				break
			}
			if _, ok := seen[c.ID]; ok {
				continue
			}
			seen[c.ID] = struct{}{}
			cands = append(cands, c)
		}
	}
	return rerankCandidates(cands, anchors, queryText, topK), nil
}

// byAnchor returns artifacts matching any anchor. Implementation is
// deliberately simple: load artifacts of plausible kinds, then filter
// in Go via MatchesArtifactIdentifier. The fixture is small; a more
// targeted SQL approach can wait.
func (r *Retriever) byAnchor(ctx context.Context, anchors []Anchor) ([]ArtifactCandidate, error) {
	kinds := map[string]struct{}{}
	for _, a := range anchors {
		for _, k := range plausibleKindsFor(a.Kind) {
			kinds[k] = struct{}{}
		}
	}
	if len(kinds) == 0 {
		return nil, nil
	}
	kindList := make([]string, 0, len(kinds))
	for k := range kinds {
		kindList = append(kindList, k)
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(kindList)), ",")
	args := make([]any, 0, len(kindList))
	for _, k := range kindList {
		args = append(args, k)
	}
	q := fmt.Sprintf(`
		SELECT id, kind, identifier, package, class, method,
		       file, start_line, end_line, signature, source
		  FROM code_artifacts
		 WHERE kind IN (%s)
	`, placeholders)
	rows, err := r.d.SQL().QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("retrieve by anchor: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []ArtifactCandidate
	for rows.Next() {
		var c ArtifactCandidate
		var pkg, cls, mth, sig, src sql.NullString
		if err := rows.Scan(&c.ID, &c.Kind, &c.Identifier,
			&pkg, &cls, &mth, &c.File, &c.StartLine, &c.EndLine, &sig, &src); err != nil {
			return nil, err
		}
		c.Package, c.Class, c.Method = pkg.String, cls.String, mth.String
		c.Signature, c.Source = sig.String, src.String
		// Keep only artifacts that an anchor genuinely matches.
		if !anyAnchorMatches(anchors, c.Kind, c.Identifier) {
			continue
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// byKNN runs a vec0 KNN over artifact_vec and hydrates each rowid into
// an ArtifactCandidate.
func (r *Retriever) byKNN(ctx context.Context, queryVec []float32, k int) ([]ArtifactCandidate, error) {
	if k <= 0 || len(queryVec) == 0 {
		return nil, nil
	}
	q := `
		SELECT av.rowid, av.distance,
		       ca.kind, ca.identifier, ca.package, ca.class, ca.method,
		       ca.file, ca.start_line, ca.end_line, ca.signature, ca.source
		  FROM artifact_vec av
		  JOIN code_artifacts ca ON ca.id = av.rowid
		 WHERE av.embedding MATCH ? AND av.k = ?
		 ORDER BY av.distance
	`
	rows, err := r.d.SQL().QueryContext(ctx, q, db.PackFloat32(queryVec), k)
	if err != nil {
		return nil, fmt.Errorf("retrieve by knn: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []ArtifactCandidate
	for rows.Next() {
		var c ArtifactCandidate
		var pkg, cls, mth, sig, src sql.NullString
		if err := rows.Scan(&c.ID, &c.Distance,
			&c.Kind, &c.Identifier,
			&pkg, &cls, &mth, &c.File, &c.StartLine, &c.EndLine, &sig, &src); err != nil {
			return nil, err
		}
		c.Package, c.Class, c.Method = pkg.String, cls.String, mth.String
		c.Signature, c.Source = sig.String, src.String
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func plausibleKindsFor(a AnchorKind) []string {
	switch a {
	case AnchorRESTPath:
		return []string{"rest_endpoint"}
	case AnchorTopic:
		return []string{"kafka_listener", "rabbit_listener", "jms_listener", "event_listener"}
	case AnchorRole:
		return []string{"security_rule"}
	case AnchorScheduled:
		return []string{"scheduled_job"}
	}
	return nil
}

func anyAnchorMatches(anchors []Anchor, kind, identifier string) bool {
	for _, a := range anchors {
		if MatchesArtifactIdentifier(a, kind, identifier) {
			return true
		}
	}
	return false
}

func candidatePoolSize(topK int) int {
	if topK <= 0 {
		topK = DefaultTopK
	}
	n := topK * 3
	if n < topK+20 {
		n = topK + 20
	}
	if n > 60 {
		n = 60
	}
	return n
}

type rankedCandidate struct {
	c     ArtifactCandidate
	score float64
	order int
}

func rerankCandidates(
	cands []ArtifactCandidate,
	anchors []Anchor,
	queryText string,
	topK int,
) []ArtifactCandidate {
	if topK <= 0 {
		topK = DefaultTopK
	}
	queryTerms := tokenize(queryText)
	ranked := make([]rankedCandidate, 0, len(cands))
	for i, c := range cands {
		score := 0.0
		if anyAnchorMatches(anchors, c.Kind, c.Identifier) {
			c.Anchored = true
		}
		if c.Anchored {
			score += 1000
		}
		for _, a := range anchors {
			if MatchesArtifactIdentifier(a, c.Kind, c.Identifier) {
				score += 250
			}
		}
		score += weightedOverlap(queryTerms, tokenize(c.Identifier)) * 12
		score += weightedOverlap(queryTerms, tokenize(c.Class+" "+c.Method+" "+c.Signature)) * 6
		score += weightedOverlap(queryTerms, tokenize(c.File)) * 3
		score += weightedOverlap(queryTerms, tokenize(c.Source)) * 2
		if c.Distance > 0 {
			score -= math.Min(c.Distance, 10)
		}
		ranked = append(ranked, rankedCandidate{c: c, score: score, order: i})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		if ranked[i].c.Anchored != ranked[j].c.Anchored {
			return ranked[i].c.Anchored
		}
		if ranked[i].c.Distance != ranked[j].c.Distance {
			return ranked[i].c.Distance < ranked[j].c.Distance
		}
		return ranked[i].order < ranked[j].order
	})
	if len(ranked) > topK {
		ranked = ranked[:topK]
	}
	out := make([]ArtifactCandidate, 0, len(ranked))
	for _, r := range ranked {
		out = append(out, r.c)
	}
	return out
}

func weightedOverlap(queryTerms, candidateTerms map[string]int) float64 {
	if len(queryTerms) == 0 || len(candidateTerms) == 0 {
		return 0
	}
	score := 0.0
	for term, qn := range queryTerms {
		if cn := candidateTerms[term]; cn > 0 {
			score += float64(min(qn, cn))
		}
	}
	return score
}

func tokenize(s string) map[string]int {
	terms := map[string]int{}
	for _, raw := range tokenRe.FindAllString(splitCamel(s), -1) {
		tok := strings.ToLower(raw)
		if len(tok) < 3 || stopTerms[tok] {
			continue
		}
		terms[tok]++
	}
	return terms
}

func splitCamel(s string) string {
	return camelRe.ReplaceAllString(s, "${1} ${2}")
}

var (
	tokenRe = regexp.MustCompile(`[A-Za-z0-9]+`)
	camelRe = regexp.MustCompile(`([a-z0-9])([A-Z])`)

	stopTerms = map[string]bool{
		"and": true, "are": true, "for": true, "from": true,
		"get": true, "post": true, "put": true, "the": true,
		"this": true, "that": true, "via": true, "with": true,
	}
)
