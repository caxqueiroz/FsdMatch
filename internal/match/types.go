package match

// ArtifactCandidate is one row from code_artifacts hydrated for the
// matcher prompt. The matcher does not need every column.
type ArtifactCandidate struct {
	ID         int64
	Kind       string
	Identifier string
	Package    string
	Class      string
	Method     string
	File       string
	StartLine  int
	EndLine    int
	Signature  string
	Source     string
	// Anchored is true when this candidate was added because at least
	// one anchor matched. Surfaces in the judgment prompt as a hint.
	Anchored bool
	// Distance is the vec0 KNN distance for vec-retrieved candidates;
	// 0 for anchor-only candidates.
	Distance float64
}

// Evidence is one file:start-end span justifying a verdict.
type Evidence struct {
	File  string `json:"file"`
	Start int    `json:"start"`
	End   int    `json:"end"`
	Note  string `json:"note,omitempty"`
}

// Match is the per-(FR, artifact) verdict. Mirrors the matches table
// columns plus in-memory decorations from the test cross-check.
type Match struct {
	FeatureID     string
	ArtifactID    int64
	Verdict       string
	Confidence    float64
	Evidence      []Evidence
	Notes         string
	Model         string
	PromptVersion string

	// Decorations not stored in matches; computed at write/read time
	// from the tests table.
	Tested    bool
	TestCount int
}

// Verdict values.
const (
	VerdictImplements = "implements"
	VerdictDrifts     = "drifts"
	VerdictUnrelated  = "unrelated"
)

// IsValidVerdict reports whether v is one of the three SPEC verdicts.
func IsValidVerdict(v string) bool {
	switch v {
	case VerdictImplements, VerdictDrifts, VerdictUnrelated:
		return true
	}
	return false
}
