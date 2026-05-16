package code

import (
	"context"
	"strings"
	"testing"
)

func TestHarvestSpringFixtureCoversAllKinds(t *testing.T) {
	arts, tests, err := HarvestSpring(context.Background(), "../../testdata/sample-spring-app")
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) == 0 {
		t.Fatal("no artifacts harvested")
	}
	if len(tests) != 1 {
		t.Errorf("expected 1 test, got %d", len(tests))
	}
	if tests[0].TestKind != "WebMvcTest" {
		t.Errorf("expected WebMvcTest, got %q", tests[0].TestKind)
	}
	if len(tests[0].MockPaths) == 0 || tests[0].MockPaths[0] != "GET /api/v1/notes/1" {
		t.Errorf("expected GET /api/v1/notes/1, got %v", tests[0].MockPaths)
	}

	wantKinds := []string{
		"rest_endpoint",
		"kafka_listener",
		"scheduled_job",
		"security_rule",
		"entity",
		"config_props",
		"exception_handler",
	}
	have := map[string]int{}
	for _, a := range arts {
		have[a.Kind]++
	}
	for _, k := range wantKinds {
		if have[k] == 0 {
			t.Errorf("missing kind %q in harvested artifacts; got %v", k, have)
		}
	}

	// Spot-check identifiers.
	wantIDs := map[string]bool{
		"GET /api/v1/notes/{id}":                             false,
		"POST /api/v1/notes":                                 false,
		"DELETE /api/v1/notes/{id}":                          false,
		"kafka topics=notes-events groupId=notebook-indexer": false,
		"cron=0 0 2 * * *":                                   false,
	}
	for _, a := range arts {
		if _, ok := wantIDs[a.Identifier]; ok {
			wantIDs[a.Identifier] = true
		}
	}
	for id, seen := range wantIDs {
		if !seen {
			t.Errorf("identifier %q not produced; got identifiers = %v",
				id, identifiersOf(arts))
		}
	}
}

func TestHarvestSpringSecurityRuleSeparateFromRest(t *testing.T) {
	arts, _, err := HarvestSpring(context.Background(), "../../testdata/sample-spring-app")
	if err != nil {
		t.Fatal(err)
	}
	var (
		rest, sec int
	)
	for _, a := range arts {
		switch a.Kind {
		case "rest_endpoint":
			rest++
		case "security_rule":
			sec++
		}
	}
	// Three @{Get,Post,Delete}Mapping each carry @PreAuthorize ⇒ 3 of each.
	if rest != 3 {
		t.Errorf("expected 3 rest_endpoint rows, got %d", rest)
	}
	if sec != 3 {
		t.Errorf("expected 3 security_rule rows, got %d", sec)
	}
}

func TestHarvestSpringEntityCarriesTableName(t *testing.T) {
	arts, _, err := HarvestSpring(context.Background(), "../../testdata/sample-spring-app")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range arts {
		if a.Kind == "entity" {
			if !strings.Contains(a.Identifier, "table=notes") {
				t.Errorf("entity identifier missing table=notes: %q", a.Identifier)
			}
			return
		}
	}
	t.Fatal("no entity artifact found")
}

func identifiersOf(arts []Artifact) []string {
	out := make([]string, 0, len(arts))
	for _, a := range arts {
		out = append(out, a.Kind+": "+a.Identifier)
	}
	return out
}
