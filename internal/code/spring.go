package code

import (
	"context"
	"fmt"
	"sort"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// Artifact is one row destined for code_artifacts. Population happens
// here; SCIP merge fills scip_symbol later.
type Artifact struct {
	Kind        string
	Identifier  string
	ScipSymbol  string
	Package     string
	Class       string
	Method      string
	File        string
	StartLine   int
	EndLine     int
	Signature   string
	Annotations map[string]string // annotation name → raw arguments
	Source      string
}

// FQN returns "package.Class" or "package.Class.method".
func (a Artifact) FQN() string {
	parts := []string{}
	if a.Package != "" {
		parts = append(parts, a.Package)
	}
	if a.Class != "" {
		parts = append(parts, a.Class)
	}
	fqn := strings.Join(parts, ".")
	if a.Method != "" {
		fqn += "." + a.Method
	}
	return fqn
}

// HarvestSpring scans every .java file under root and returns one
// Artifact per recognised Spring annotation, plus extracted JUnit
// tests. Both lists are stable-sorted by (file, start_line).
func HarvestSpring(ctx context.Context, root string) ([]Artifact, []TestCase, error) {
	var (
		artifacts []Artifact
		tests     []TestCase
	)
	err := WalkSourceTree(ctx, root, func(_ context.Context, f *File) error {
		fileArts, fileTests := harvestFile(f)
		artifacts = append(artifacts, fileArts...)
		tests = append(tests, fileTests...)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sortArtifacts(artifacts)
	sortTests(tests)
	return artifacts, tests, nil
}

func sortArtifacts(a []Artifact) {
	sort.SliceStable(a, func(i, j int) bool {
		if a[i].File != a[j].File {
			return a[i].File < a[j].File
		}
		if a[i].StartLine != a[j].StartLine {
			return a[i].StartLine < a[j].StartLine
		}
		return a[i].Kind < a[j].Kind
	})
}

func harvestFile(f *File) ([]Artifact, []TestCase) {
	var (
		arts  []Artifact
		tests []TestCase
	)
	for _, classNode := range findClassDeclarations(f.Tree.RootNode()) {
		ctx := classContext(f, classNode)

		// Class-level rows.
		for _, kind := range classLevelKinds(ctx.annotations) {
			arts = append(arts, makeClassArtifact(ctx, kind))
		}

		// Method-level rows.
		body := firstChildOfType(classNode, "class_body")
		if body == nil {
			body = firstChildOfType(classNode, "interface_body")
		}
		if body == nil {
			body = firstChildOfType(classNode, "record_body")
		}
		if body == nil {
			continue
		}
		for _, m := range methodDeclarations(body) {
			arts = append(arts, harvestMethod(ctx, m)...)
			if t, ok := harvestTestMethod(ctx, m); ok {
				tests = append(tests, t)
			}
		}
	}
	return arts, tests
}

// classCtx is the per-class harvest state shared across method
// processing.
type classCtx struct {
	file        *File
	classNode   *sitter.Node
	className   string
	annotations map[string]string
}

func classContext(f *File, classNode *sitter.Node) classCtx {
	name := ""
	if n := classNode.ChildByFieldName("name"); n != nil {
		name = nodeText(f.Source, n)
	}
	return classCtx{
		file:        f,
		classNode:   classNode,
		className:   name,
		annotations: collectAnnotations(f.Source, classNode),
	}
}

// methodDeclarations returns every method_declaration child of body
// (skipping nested classes' methods, which are handled by their own
// classContext).
func methodDeclarations(body *sitter.Node) []*sitter.Node {
	var out []*sitter.Node
	for i := 0; i < int(body.NamedChildCount()); i++ {
		c := body.NamedChild(i)
		switch c.Type() {
		case "method_declaration", "constructor_declaration":
			out = append(out, c)
		}
	}
	return out
}

// collectAnnotations returns name → raw-argument-text for every
// annotation directly attached to decl. Modifiers in the Java AST
// hold both modifiers (public/static) and annotations.
func collectAnnotations(src []byte, decl *sitter.Node) map[string]string {
	out := map[string]string{}
	mods := firstChildOfType(decl, "modifiers")
	if mods == nil {
		return out
	}
	for i := 0; i < int(mods.NamedChildCount()); i++ {
		c := mods.NamedChild(i)
		switch c.Type() {
		case "marker_annotation":
			if name := annotationName(src, c); name != "" {
				out[name] = ""
			}
		case "annotation":
			if name := annotationName(src, c); name != "" {
				args := c.ChildByFieldName("arguments")
				out[name] = strings.TrimSpace(nodeText(src, args))
			}
		}
	}
	return out
}

func annotationName(src []byte, ann *sitter.Node) string {
	n := ann.ChildByFieldName("name")
	if n == nil {
		return ""
	}
	// Strip a leading package prefix like "org.springframework.web.bind.annotation.GetMapping".
	full := nodeText(src, n)
	if i := strings.LastIndex(full, "."); i >= 0 {
		return full[i+1:]
	}
	return full
}

// classLevelKinds reports which class-level rows to emit for the given
// class annotations. Class-only kinds: entity, config_props.
// Marker kinds (RestController, RestControllerAdvice, Service,
// Component, Repository) do NOT emit a row of their own; they only
// gate method-level emissions or join with a child @ExceptionHandler.
func classLevelKinds(ann map[string]string) []string {
	var kinds []string
	if _, ok := ann["Entity"]; ok {
		kinds = append(kinds, "entity")
	}
	if _, ok := ann["ConfigurationProperties"]; ok {
		kinds = append(kinds, "config_props")
	}
	return kinds
}

func makeClassArtifact(ctx classCtx, kind string) Artifact {
	id := classIdentifier(ctx, kind)
	return Artifact{
		Kind:        kind,
		Identifier:  id,
		Package:     ctx.file.Pkg,
		Class:       ctx.className,
		File:        ctx.file.Path,
		StartLine:   startLine(ctx.classNode),
		EndLine:     endLine(ctx.classNode),
		Signature:   "class " + ctx.className,
		Annotations: copyMap(ctx.annotations),
		Source:      sourceSlice(ctx.file.Source, ctx.classNode, 1024),
	}
}

func classIdentifier(ctx classCtx, kind string) string {
	switch kind {
	case "entity":
		table := stringAttr(ctx.annotations["Table"], "name", ctx.className)
		return fmt.Sprintf("%s.%s (table=%s)", ctx.file.Pkg, ctx.className, table)
	case "config_props":
		prefix := stringAttr(ctx.annotations["ConfigurationProperties"], "prefix",
			singleStringArg(ctx.annotations["ConfigurationProperties"]))
		if prefix == "" {
			prefix = "(none)"
		}
		return fmt.Sprintf("%s.* (%s)", prefix, ctx.className)
	default:
		return ctx.file.Pkg + "." + ctx.className
	}
}

// harvestMethod yields zero or more Artifacts for a single method.
// One row per primary annotation kind; @Transactional + conditional
// annotations stay as attributes only.
func harvestMethod(ctx classCtx, m *sitter.Node) []Artifact {
	ann := collectAnnotations(ctx.file.Source, m)
	if len(ann) == 0 {
		return nil
	}
	methodName := ""
	if n := m.ChildByFieldName("name"); n != nil {
		methodName = nodeText(ctx.file.Source, n)
	}

	type row struct {
		kind, identifier string
	}
	var rows []row
	for ann1, args := range ann {
		switch ann1 {
		case "GetMapping", "PostMapping", "PutMapping", "DeleteMapping", "PatchMapping", "RequestMapping":
			rows = append(rows, row{
				kind:       "rest_endpoint",
				identifier: restIdentifier(ctx.annotations, ann1, args),
			})
		case "KafkaListener":
			rows = append(rows, row{
				kind:       "kafka_listener",
				identifier: listenerIdentifier("kafka", args),
			})
		case "RabbitListener":
			rows = append(rows, row{kind: "rabbit_listener", identifier: listenerIdentifier("rabbit", args)})
		case "JmsListener":
			rows = append(rows, row{kind: "jms_listener", identifier: listenerIdentifier("jms", args)})
		case "EventListener":
			rows = append(rows, row{kind: "event_listener", identifier: listenerIdentifier("event", args)})
		case "Scheduled":
			rows = append(rows, row{kind: "scheduled_job", identifier: scheduledIdentifier(args)})
		case "PreAuthorize", "PostAuthorize", "Secured", "RolesAllowed":
			rows = append(rows, row{
				kind:       "security_rule",
				identifier: fmt.Sprintf("@%s %s", ann1, normaliseArgs(args)),
			})
		case "ExceptionHandler":
			rows = append(rows, row{
				kind:       "exception_handler",
				identifier: exceptionHandlerIdentifier(ctx.className, methodName, args),
			})
		}
	}

	if len(rows) == 0 {
		return nil
	}
	out := make([]Artifact, 0, len(rows))
	for _, r := range rows {
		out = append(out, Artifact{
			Kind:        r.kind,
			Identifier:  r.identifier,
			Package:     ctx.file.Pkg,
			Class:       ctx.className,
			Method:      methodName,
			File:        ctx.file.Path,
			StartLine:   startLine(m),
			EndLine:     endLine(m),
			Signature:   methodSignature(ctx.file.Source, m),
			Annotations: ann,
			Source:      sourceSlice(ctx.file.Source, m, 2048),
		})
	}
	return out
}

func methodSignature(src []byte, m *sitter.Node) string {
	// Compose "ReturnType name(parameters)" from labelled fields.
	var b strings.Builder
	if t := m.ChildByFieldName("type"); t != nil {
		b.WriteString(nodeText(src, t))
		b.WriteString(" ")
	}
	if n := m.ChildByFieldName("name"); n != nil {
		b.WriteString(nodeText(src, n))
	}
	if p := m.ChildByFieldName("parameters"); p != nil {
		b.WriteString(nodeText(src, p))
	}
	return strings.TrimSpace(b.String())
}

// restIdentifier composes "VERB /class-prefix/method-path".
func restIdentifier(classAnn map[string]string, methodAnn, methodArgs string) string {
	verb := verbForMapping(methodAnn)
	classPath := pathFromAnnotation(classAnn["RequestMapping"])
	methodPath := pathFromAnnotation(methodArgs)
	combined := joinPath(classPath, methodPath)
	if combined == "" {
		combined = "/"
	}
	if verb == "" {
		verb = "ANY"
	}
	return verb + " " + combined
}

func verbForMapping(name string) string {
	switch name {
	case "GetMapping":
		return "GET"
	case "PostMapping":
		return "POST"
	case "PutMapping":
		return "PUT"
	case "DeleteMapping":
		return "DELETE"
	case "PatchMapping":
		return "PATCH"
	default:
		return ""
	}
}

// pathFromAnnotation extracts the path string from a mapping annotation's
// arguments. Handles both `("/x")` and `(value = "/x", ...)` forms,
// plus arrays `({"/x","/y"})` taking the first element.
func pathFromAnnotation(args string) string {
	args = strings.TrimSpace(args)
	if args == "" || args == "()" {
		return ""
	}
	args = strings.TrimPrefix(args, "(")
	args = strings.TrimSuffix(args, ")")

	if v := stringAttr("("+args+")", "value", ""); v != "" {
		return firstArrayElem(v)
	}
	if v := stringAttr("("+args+")", "path", ""); v != "" {
		return firstArrayElem(v)
	}
	// Bare positional string.
	return firstArrayElem(unquote(args))
}

func joinPath(a, b string) string {
	a = strings.TrimRight(a, "/")
	b = strings.TrimLeft(b, "/")
	switch {
	case a == "" && b == "":
		return ""
	case a == "":
		return "/" + b
	case b == "":
		return a
	default:
		return a + "/" + b
	}
}

// listenerIdentifier composes a stable identifier for any listener kind
// from the most useful args.
func listenerIdentifier(kind, args string) string {
	topics := stringAttr(args, "topics", "")
	if topics == "" {
		topics = stringAttr(args, "destination", "")
	}
	if topics == "" {
		topics = stringAttr(args, "queues", "")
	}
	if topics == "" {
		topics = stringAttr(args, "value", "")
	}
	groupID := stringAttr(args, "groupId", "")
	if topics == "" && groupID == "" {
		return kind + "-listener"
	}
	out := kind + " topics=" + topics
	if groupID != "" {
		out += " groupId=" + groupID
	}
	return out
}

func scheduledIdentifier(args string) string {
	if v := stringAttr(args, "cron", ""); v != "" {
		return "cron=" + v
	}
	if v := stringAttr(args, "fixedRate", ""); v != "" {
		return "fixedRate=" + v
	}
	if v := stringAttr(args, "fixedDelay", ""); v != "" {
		return "fixedDelay=" + v
	}
	return "scheduled"
}

func exceptionHandlerIdentifier(class, method, args string) string {
	target := strings.TrimSpace(args)
	target = strings.TrimPrefix(target, "(")
	target = strings.TrimSuffix(target, ")")
	target = strings.TrimSuffix(strings.TrimSpace(target), ".class")
	if target == "" {
		target = "Throwable"
	}
	return target + " → " + class + "." + method
}

// stringAttr finds `key = "value"` (or `key = ClassName.class`) inside
// args and returns the unquoted value, or fallback when absent.
func stringAttr(args, key, fallback string) string {
	if args == "" {
		return fallback
	}
	rest := args
	if i := strings.Index(rest, key+" ="); i >= 0 {
		rest = rest[i+len(key)+2:]
	} else if i := strings.Index(rest, key+"="); i >= 0 {
		rest = rest[i+len(key)+1:]
	} else {
		return fallback
	}
	rest = strings.TrimLeft(rest, " ")
	if strings.HasPrefix(rest, `"`) {
		end := strings.Index(rest[1:], `"`)
		if end < 0 {
			return fallback
		}
		return rest[1 : 1+end]
	}
	if strings.HasPrefix(rest, "{") {
		end := strings.Index(rest, "}")
		if end < 0 {
			return fallback
		}
		// First element of a string-array literal.
		inner := strings.TrimSpace(rest[1:end])
		return firstArrayElem(inner)
	}
	// Unquoted token (number, enum, ClassName.class).
	cut := len(rest)
	for i, r := range rest {
		if r == ',' || r == ')' {
			cut = i
			break
		}
	}
	return strings.TrimSpace(rest[:cut])
}

// singleStringArg pulls the string out of an annotation that takes a
// single positional string argument like @ConfigurationProperties("prefix").
func singleStringArg(args string) string {
	args = strings.TrimSpace(args)
	args = strings.TrimPrefix(args, "(")
	args = strings.TrimSuffix(args, ")")
	args = strings.TrimSpace(args)
	return unquote(args)
}

// firstArrayElem returns the first element of a curly-braced string
// list, or s itself when it's not an array.
func firstArrayElem(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "{") {
		return unquote(s)
	}
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	if i := strings.Index(s, ","); i >= 0 {
		s = s[:i]
	}
	return unquote(strings.TrimSpace(s))
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// normaliseArgs collapses inner whitespace to a single space so two
// formattings of the same argument list compare equal.
func normaliseArgs(args string) string {
	return strings.Join(strings.Fields(args), " ")
}

// sourceSlice returns up to maxBytes of n's source, suffixing "..." on
// truncation. Used to populate code_artifacts.source for downstream
// matcher prompts.
func sourceSlice(src []byte, n *sitter.Node, maxBytes int) string {
	s := nodeText(src, n)
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes] + "..."
}

func copyMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
