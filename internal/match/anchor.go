// Package match runs the per-FR matching pipeline: anchor extraction →
// vec0 retrieval → model judgment → test cross-check, with the
// hard rule from SPEC §7.4 that any verdict missing file:line evidence
// is downgraded to "unrelated".
package match

import (
	"regexp"
	"sort"
	"strings"
)

// AnchorKind classifies what an anchor extracted from FR text refers to.
type AnchorKind string

const (
	// AnchorRESTPath captures "VERB /path/segment" pairs.
	AnchorRESTPath AnchorKind = "rest_path"
	// AnchorTopic captures explicit Kafka/Rabbit/JMS topic names.
	AnchorTopic AnchorKind = "topic"
	// AnchorRole captures Spring Security role names like ROLE_USER, USER, ADMIN.
	AnchorRole AnchorKind = "role"
	// AnchorScheduled flags FRs that talk about scheduled / nightly / cron jobs.
	AnchorScheduled AnchorKind = "scheduled"
)

// Anchor is one (kind, value) tuple extracted from an FR.
type Anchor struct {
	Kind  AnchorKind
	Value string
}

// FeatureLike is the slice of feature data the anchor extractor needs.
// Defined here as a small interface so the matcher can stay decoupled
// from internal/fsd's concrete Feature type.
type FeatureLike interface {
	Title() string
	Description() string
	Acceptance() []string
}

// ExtractAnchors returns a deduplicated list of anchors for FR text.
// Sorted by (kind, value) for deterministic output.
func ExtractAnchors(fr FeatureLike) []Anchor {
	text := fr.Title() + "\n" + fr.Description()
	for _, a := range fr.Acceptance() {
		text += "\n" + a
	}

	seen := map[Anchor]struct{}{}
	add := func(k AnchorKind, v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		seen[Anchor{Kind: k, Value: v}] = struct{}{}
	}

	for _, m := range restPathRe.FindAllStringSubmatch(text, -1) {
		verb := strings.ToUpper(m[1])
		path := strings.TrimRight(m[2], ".,;:)")
		add(AnchorRESTPath, verb+" "+path)
	}
	for _, m := range topicRe.FindAllStringSubmatch(text, -1) {
		// Any of the alternation groups may have matched.
		for _, g := range m[1:] {
			if g != "" {
				add(AnchorTopic, g)
			}
		}
	}
	for _, m := range roleRe.FindAllStringSubmatch(text, -1) {
		for _, g := range m[1:] {
			if g != "" {
				add(AnchorRole, strings.ToUpper(g))
			}
		}
	}
	if scheduledRe.MatchString(text) {
		add(AnchorScheduled, "scheduled")
	}

	out := make([]Anchor, 0, len(seen))
	for a := range seen {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Value < out[j].Value
	})
	return out
}

// MatchesArtifactIdentifier reports whether an anchor and an artifact
// row's `kind`/`identifier` look like the same surface area. Used by
// the retrieve layer to filter candidates before vec0 KNN.
func MatchesArtifactIdentifier(a Anchor, artifactKind, identifier string) bool {
	switch a.Kind {
	case AnchorRESTPath:
		if artifactKind != "rest_endpoint" {
			return false
		}
		// "POST /api/v1/notes" matches "POST /api/v1/notes" or
		// "POST /api/v1/notes/{id}" (path prefix); we match either way
		// to handle FRs that name a base path or a templated path.
		return restPathOverlap(a.Value, identifier)
	case AnchorTopic:
		if !strings.HasSuffix(artifactKind, "_listener") {
			return false
		}
		return strings.Contains(identifier, "topics="+a.Value) ||
			strings.Contains(identifier, "topics={"+a.Value) ||
			strings.Contains(identifier, "destination="+a.Value) ||
			strings.Contains(identifier, "queues="+a.Value)
	case AnchorRole:
		if artifactKind != "security_rule" {
			return false
		}
		return strings.Contains(strings.ToUpper(identifier), a.Value)
	case AnchorScheduled:
		return artifactKind == "scheduled_job"
	}
	return false
}

// restPathOverlap is true when one of (anchor, identifier) is a path
// prefix of the other, after normalising templated segments {x} → "*".
func restPathOverlap(anchor, identifier string) bool {
	a := strings.SplitN(anchor, " ", 2)
	i := strings.SplitN(identifier, " ", 2)
	if len(a) != 2 || len(i) != 2 {
		return false
	}
	if a[0] != i[0] && a[0] != "ANY" && i[0] != "ANY" {
		return false
	}
	aSegs := splitPath(a[1])
	iSegs := splitPath(i[1])
	short, long := aSegs, iSegs
	if len(short) > len(long) {
		short, long = long, short
	}
	for k := range short {
		if short[k] == long[k] {
			continue
		}
		// Templated segment matches anything.
		if isTemplated(short[k]) || isTemplated(long[k]) {
			continue
		}
		return false
	}
	return true
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func isTemplated(seg string) bool {
	return strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}")
}

var (
	// restPathRe captures "VERB /path/segments" with optional surrounding
	// punctuation/quotes. Stops at whitespace, ?, comma, etc.
	restPathRe = regexp.MustCompile(`(?i)\b(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\s+` +
		"`?(/[A-Za-z0-9_/{}.\\-]+)`?")

	// topicRe captures "topic <name>" or "to/on the <name> Kafka topic"
	// patterns; keeps it simple by matching `<name>-events`,
	// `<name>-topic`, or quoted strings adjacent to "topic".
	topicRe = regexp.MustCompile("`([a-zA-Z0-9_.-]+(?:-events|-topic|\\.events))`" +
		`|topic\s+` + "`([a-zA-Z0-9_.-]+)`" +
		`|"([a-zA-Z0-9_.-]+(?:-events|-topic))"`)

	// roleRe captures Spring Security role hints. Simple list of common
	// upper-case identifiers next to "role".
	roleRe = regexp.MustCompile(`(?i)\brole[s]?\s+([A-Z][A-Z0-9_]{1,})\b` +
		`|hasRole\(['"]?([A-Z][A-Z0-9_]+)['"]?\)`)

	scheduledRe = regexp.MustCompile(`(?i)\b(scheduled|nightly|hourly|daily|cron|at\s+\d{2}:\d{2})\b`)
)
