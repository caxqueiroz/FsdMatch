package code

import (
	"regexp"
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// TestCase is one row destined for the tests table.
type TestCase struct {
	Name      string
	File      string
	Line      int
	TestKind  string   // WebMvcTest | SpringBootTest | DataJpaTest | unit
	MockPaths []string // any HTTP paths reached via MockMvc inside this test
	Asserts   []string // best-effort summary of assertion calls
}

// harvestTestMethod returns a TestCase if m is a JUnit @Test method.
func harvestTestMethod(ctx classCtx, m *sitter.Node) (TestCase, bool) {
	ann := collectAnnotations(ctx.file.Source, m)
	if _, ok := ann["Test"]; !ok {
		// Some legacy code uses @org.junit.Test or @org.junit.jupiter.api.Test;
		// annotationName already strips the FQN so plain "Test" is enough.
		return TestCase{}, false
	}
	name := ""
	if n := m.ChildByFieldName("name"); n != nil {
		name = nodeText(ctx.file.Source, n)
	}
	body := m.ChildByFieldName("body")
	bodyText := nodeText(ctx.file.Source, body)

	tc := TestCase{
		Name:      name,
		File:      ctx.file.Path,
		Line:      startLine(m),
		TestKind:  classTestKind(ctx.annotations),
		MockPaths: extractMockMvcPaths(bodyText),
		Asserts:   extractAsserts(bodyText),
	}
	return tc, true
}

func classTestKind(ann map[string]string) string {
	switch {
	case has(ann, "WebMvcTest"):
		return "WebMvcTest"
	case has(ann, "DataJpaTest"):
		return "DataJpaTest"
	case has(ann, "SpringBootTest"):
		return "SpringBootTest"
	default:
		return "unit"
	}
}

func has(m map[string]string, k string) bool { _, ok := m[k]; return ok }

// mockMvcCallRe matches expressions like `get("/api/v1/x")`,
// `post("/api/v1/y")`, `mockMvc.perform(get("/foo"))` etc. It captures
// the verb in group 1 and the unquoted path in group 2.
var mockMvcCallRe = regexp.MustCompile(`\b(get|post|put|delete|patch|head|options)\(\s*"([^"]+)"`)

func extractMockMvcPaths(body string) []string {
	matches := mockMvcCallRe.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		v := strings.ToUpper(m[1]) + " " + m[2]
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// assertCallRe captures the leaf method name on chains like
// `andExpect(status().isOk())`, `assertThat(x).isEqualTo(y)`,
// `assertEquals(...)`. We pull out the suffix calls for the asserts
// JSON column.
var assertCallRe = regexp.MustCompile(`\.(is[A-Z]\w*|hasSize|isEqualTo|hasValue)\(`)

func extractAsserts(body string) []string {
	matches := assertCallRe.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if _, ok := seen[m[1]]; ok {
			continue
		}
		seen[m[1]] = struct{}{}
		out = append(out, m[1])
	}
	sort.Strings(out)
	return out
}

func sortTests(t []TestCase) {
	sort.SliceStable(t, func(i, j int) bool {
		if t[i].File != t[j].File {
			return t[i].File < t[j].File
		}
		return t[i].Line < t[j].Line
	})
}
