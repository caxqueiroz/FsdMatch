package match

import "testing"

func TestRerankCandidatesPromotesTermRichCandidate(t *testing.T) {
	query := stringsJoinForTest(
		"Update pet",
		"POST /owners/{ownerId}/pets/{petId}/edit validates and persists pet updates",
		"pet name birthDate type owner",
	)
	cands := []ArtifactCandidate{
		{
			ID:         1,
			Kind:       "rest_endpoint",
			Identifier: "GET /vets.html",
			Class:      "VetController",
			Method:     "showVetList",
			Source:     "return findPaginated(page);",
			Distance:   0.01,
		},
		{
			ID:         2,
			Kind:       "rest_endpoint",
			Identifier: "POST /owners/{ownerId}/pets/{petId}/edit",
			Class:      "PetController",
			Method:     "processUpdateForm",
			Source:     "validate pet birthDate type owner and save owner",
			Distance:   0.42,
		},
	}

	got := rerankCandidates(cands, nil, query, 1)
	if len(got) != 1 {
		t.Fatalf("got %d candidates", len(got))
	}
	if got[0].ID != 2 {
		t.Fatalf("top candidate ID = %d, want 2", got[0].ID)
	}
}

func TestRerankCandidatesPreservesAnchorPriority(t *testing.T) {
	anchors := []Anchor{{Kind: AnchorRESTPath, Value: "POST /owners/{ownerId}/pets/{petId}/edit"}}
	cands := []ArtifactCandidate{
		{
			ID:         1,
			Kind:       "rest_endpoint",
			Identifier: "GET /owners/{ownerId}",
			Source:     "owner details with pets and visits",
			Distance:   0.01,
		},
		{
			ID:         2,
			Kind:       "rest_endpoint",
			Identifier: "POST /owners/{ownerId}/pets/{petId}/edit",
			Source:     "save updated pet",
			Distance:   0.9,
		},
	}

	got := rerankCandidates(cands, anchors, "pet update", 1)
	if len(got) != 1 {
		t.Fatalf("got %d candidates", len(got))
	}
	if got[0].ID != 2 {
		t.Fatalf("top candidate ID = %d, want anchored candidate 2", got[0].ID)
	}
	if !got[0].Anchored {
		t.Fatal("expected reranker to mark the anchor match")
	}
}

func stringsJoinForTest(parts ...string) string {
	out := ""
	for _, p := range parts {
		if out != "" {
			out += "\n"
		}
		out += p
	}
	return out
}
