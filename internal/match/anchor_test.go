package match

import (
	"reflect"
	"sort"
	"testing"
)

type frStub struct {
	title  string
	desc   string
	accept []string
}

func (f frStub) Title() string        { return f.title }
func (f frStub) Description() string  { return f.desc }
func (f frStub) Acceptance() []string { return f.accept }

func TestExtractAnchors_Notes(t *testing.T) {
	fr := frStub{
		title: "Create a note",
		desc:  "An authenticated user creates a note via POST /api/v1/notes.",
		accept: []string{
			"POST `/api/v1/notes` with {title,body} returns 201",
			"Side effects: emits notes.created to `notes-events` Kafka topic.",
			"Requires role USER",
		},
	}
	got := ExtractAnchors(fr)
	want := []Anchor{
		{Kind: AnchorRESTPath, Value: "POST /api/v1/notes"},
		{Kind: AnchorRole, Value: "USER"},
		{Kind: AnchorTopic, Value: "notes-events"},
	}
	sortAnchors(got)
	sortAnchors(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("anchors:\n  got  %+v\n  want %+v", got, want)
	}
}

func TestExtractAnchors_ScheduledFlag(t *testing.T) {
	fr := frStub{
		desc:   "A scheduled job runs nightly to compact deleted rows.",
		accept: []string{"Runs at 02:00 UTC every day."},
	}
	got := ExtractAnchors(fr)
	for _, a := range got {
		if a.Kind == AnchorScheduled {
			return
		}
	}
	t.Errorf("expected AnchorScheduled in %+v", got)
}

func TestMatchesArtifactIdentifier_RESTPath(t *testing.T) {
	cases := []struct {
		name   string
		anchor Anchor
		kind   string
		ident  string
		want   bool
	}{
		{"exact match", Anchor{AnchorRESTPath, "POST /api/v1/notes"}, "rest_endpoint", "POST /api/v1/notes", true},
		{"path-templated longer", Anchor{AnchorRESTPath, "POST /api/v1/notes"}, "rest_endpoint", "POST /api/v1/notes/{id}", true},
		{"path-templated shorter", Anchor{AnchorRESTPath, "GET /api/v1/notes/{id}"}, "rest_endpoint", "GET /api/v1/notes", true},
		{"verb mismatch", Anchor{AnchorRESTPath, "POST /api/v1/notes"}, "rest_endpoint", "DELETE /api/v1/notes", false},
		{"path mismatch", Anchor{AnchorRESTPath, "POST /api/v1/notes"}, "rest_endpoint", "POST /api/v1/auth", false},
		{"wrong kind", Anchor{AnchorRESTPath, "POST /api/v1/notes"}, "kafka_listener", "topics=x", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MatchesArtifactIdentifier(c.anchor, c.kind, c.ident); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestMatchesArtifactIdentifier_Topic(t *testing.T) {
	a := Anchor{Kind: AnchorTopic, Value: "notes-events"}
	if !MatchesArtifactIdentifier(a, "kafka_listener",
		"kafka topics=notes-events groupId=notebook-indexer") {
		t.Error("expected topic match")
	}
	if MatchesArtifactIdentifier(a, "kafka_listener",
		"kafka topics=other-events groupId=x") {
		t.Error("unexpected topic match")
	}
}

func sortAnchors(a []Anchor) {
	sort.Slice(a, func(i, j int) bool {
		if a[i].Kind != a[j].Kind {
			return a[i].Kind < a[j].Kind
		}
		return a[i].Value < a[j].Value
	})
}
