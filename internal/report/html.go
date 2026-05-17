package report

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

// WriteHTML writes a single index.html under outDir. Static, no JS,
// uses <details> for collapsible sections.
func WriteHTML(r *Report, outDir string) error {
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", outDir, err)
	}
	path := filepath.Join(outDir, "index.html")
	f, err := os.Create(path) // #nosec G304 -- outDir from CLI flag
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	return htmlTemplate.Execute(f, r)
}

var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"upper": func(s Status) string {
		switch s {
		case StatusImplemented:
			return "IMPLEMENTED"
		case StatusDrifts:
			return "DRIFTS"
		case StatusMissing:
			return "MISSING"
		}
		return string(s)
	},
	"scipGraph": renderSCIPGraph,
}).Parse(`<!doctype html>
<html lang="en"><head>
<meta charset="utf-8">
<title>fsdtrace report — {{.RunID}}</title>
<style>
 body { font: 14px/1.4 system-ui, sans-serif; margin: 2rem auto; max-width: 64rem; padding: 0 1rem; color:#222; }
 h1 { border-bottom: 1px solid #ccc; padding-bottom: .3rem; }
 table { border-collapse: collapse; width: 100%; margin: .8rem 0; }
 th, td { border: 1px solid #ddd; padding: .25rem .5rem; vertical-align: top; }
 th { background: #f7f7f7; text-align: left; }
 td.num { text-align: right; }
 .badge { padding: 2px 6px; border-radius: 4px; font-weight: 600; font-size: 12px; }
 .b-implemented { background:#d4edda; color:#155724; }
 .b-drifts { background:#fff3cd; color:#856404; }
 .b-missing { background:#f8d7da; color:#721c24; }
 details { margin: .4rem 0; }
 summary { cursor: pointer; font-weight: 600; }
 code { background:#f4f4f4; padding:1px 4px; border-radius:3px; }
 .ev { font-size: 12px; color:#555; margin-left: 1.2rem; }
 .graph-cell { background:#fbfbfb; padding:.6rem; }
 .scip-graph { display:block; max-width:100%; border:1px solid #ddd; background:#fff; border-radius:6px; }
 .scip-edge { stroke:#888; stroke-width:1.4; }
 .scip-node rect { fill:#f8f9fa; stroke:#bbb; stroke-width:1; }
 .scip-node-root rect { fill:#eef6ff; stroke:#7aa7d9; }
 .scip-node-kind { font-size:10px; fill:#666; font-weight:700; text-transform:uppercase; }
 .scip-node-label { font-size:11px; fill:#222; }
 .scip-depth { font-size:10px; fill:#777; }
</style>
</head><body>
<h1>Coverage — run <code>{{.RunID}}</code></h1>
<p>Generated {{.Generated.Format "2006-01-02 15:04:05"}} UTC.</p>
{{if .IncludeCallGraph}}<p>SCIP call graph support is included for implemented matches.</p>{{end}}

<h2>Roll-up</h2>
<table>
 <tr><th>Section</th><th class="num">Total</th><th class="num">Implemented</th><th class="num">Drifts</th><th class="num">Missing</th>{{if .IncludeCallGraph}}<th class="num">SCIP Support</th>{{end}}</tr>
{{range .Sections}}
 <tr><td>{{.Name}}</td><td class="num">{{.Total}}</td><td class="num">{{.Implemented}}</td><td class="num">{{.Drifts}}</td><td class="num">{{.Missing}}</td>{{if $.IncludeCallGraph}}<td class="num">{{.SupportingArtifacts}}</td>{{end}}</tr>
{{end}}
</table>

{{range .Sections}}
<h2>Section: {{.Name}}</h2>
<p>Total {{.Total}} · implemented {{.Implemented}} · drifts {{.Drifts}} · missing {{.Missing}}</p>
{{range .Features}}
<details>
 <summary>{{.ID}} — {{.Title}}
  <span class="badge b-{{.Status}}">{{upper .Status}}</span>
  {{if .TestedAny}}<span class="badge" title="At least one matched artifact has tests">tested</span>{{end}}
  {{if and $.IncludeCallGraph .SupportingArtifacts}}<span class="badge" title="SCIP support artifacts">{{.SupportingArtifacts}} support</span>{{end}}
 </summary>
 {{if .Matches}}
 <table>
  <tr><th>Verdict</th><th>Confidence</th><th>Tested</th>{{if $.IncludeCallGraph}}<th>SCIP Support</th>{{end}}<th>Artifact</th><th>Location</th></tr>
  {{range .Matches}}
  <tr>
   <td><span class="badge b-{{.Verdict}}">{{.Verdict}}</span></td>
   <td class="num">{{printf "%.2f" .Confidence}}</td>
   <td class="num">{{if .Tested}}✓ ({{.TestCount}}){{end}}</td>
   {{if $.IncludeCallGraph}}<td class="num">{{len .SupportingArtifacts}}</td>{{end}}
   <td><code>{{.Kind}}</code> {{.Identifier}}</td>
   <td><code>{{.File}}:{{.StartLine}}-{{.EndLine}}</code></td>
  </tr>
  {{range .Evidence}}
  <tr><td colspan="{{if $.IncludeCallGraph}}6{{else}}5{{end}}" class="ev">↳ <code>{{.File}}:{{.Start}}-{{.End}}</code> {{if .Note}}— {{.Note}}{{end}}</td></tr>
  {{end}}
  {{if $.IncludeCallGraph}}
  {{if .SupportingArtifacts}}
  <tr><td colspan="6" class="graph-cell">{{scipGraph .}}</td></tr>
  {{end}}
  {{range .SupportingArtifacts}}
  <tr><td colspan="6" class="ev">↳ SCIP depth {{.Depth}} <code>{{.Kind}}</code> {{.Identifier}} via <code>{{.RelationshipKind}}</code> at <code>{{.File}}:{{.StartLine}}-{{.EndLine}}</code></td></tr>
  {{end}}
  {{end}}
  {{end}}
 </table>
 {{else}}
 <p><em>No matching artifacts.</em></p>
 {{end}}
</details>
{{end}}
{{end}}

<h1>Drift</h1>
{{if .Drift}}
<table>
 <tr><th>FR</th><th>Section</th><th>Artifact</th><th>Location</th><th>Confidence</th><th>Notes</th></tr>
 {{range .Drift}}
 <tr>
  <td>{{.FeatureID}} — {{.FeatureTitle}}</td>
  <td>{{.Section}}</td>
  <td><code>{{.Kind}}</code> {{.Identifier}}</td>
  <td><code>{{.File}}:{{.StartLine}}-{{.EndLine}}</code></td>
  <td class="num">{{printf "%.2f" .Confidence}}</td>
  <td>{{.Notes}}</td>
 </tr>
 {{end}}
</table>
{{else}}
<p><em>No drift detected.</em></p>
{{end}}

<h1>Orphans</h1>
{{if .Orphans}}
<table>
 <tr><th>Kind</th><th>Identifier</th><th>Location</th></tr>
 {{range .Orphans}}
 <tr><td><code>{{.Kind}}</code></td><td>{{.Identifier}}</td><td><code>{{.File}}:{{.StartLine}}-{{.EndLine}}</code></td></tr>
 {{end}}
</table>
{{else}}
<p><em>No orphan public-surface artifacts.</em></p>
{{end}}

</body></html>
`))

const (
	scipGraphMargin = 24
	scipGraphNodeW  = 190
	scipGraphNodeH  = 42
	scipGraphColGap = 240
	scipGraphRowGap = 70
)

type graphPosition struct {
	x int
	y int
}

func renderSCIPGraph(m MatchRow) template.HTML {
	if len(m.SupportingArtifacts) == 0 {
		return ""
	}
	counts := map[int]int{0: 1}
	maxDepth := 0
	for _, a := range m.SupportingArtifacts {
		counts[a.Depth]++
		if a.Depth > maxDepth {
			maxDepth = a.Depth
		}
	}
	maxRows := 1
	for _, n := range counts {
		if n > maxRows {
			maxRows = n
		}
	}
	width := scipGraphMargin*2 + scipGraphNodeW + scipGraphColGap*maxDepth
	height := scipGraphMargin*2 + scipGraphNodeH + scipGraphRowGap*(maxRows-1)
	if height < 120 {
		height = 120
	}

	positions := map[int64]graphPosition{
		m.ArtifactID: {
			x: scipGraphMargin,
			y: height/2 - scipGraphNodeH/2,
		},
	}
	seenAtDepth := map[int]int{}
	for _, a := range m.SupportingArtifacts {
		idx := seenAtDepth[a.Depth]
		seenAtDepth[a.Depth]++
		columnRows := counts[a.Depth]
		yOffset := (maxRows - columnRows) * scipGraphRowGap / 2
		positions[a.ArtifactID] = graphPosition{
			x: scipGraphMargin + scipGraphColGap*a.Depth,
			y: scipGraphMargin + yOffset + idx*scipGraphRowGap,
		}
	}

	var b strings.Builder
	title := "SCIP call graph support for " + compactText(m.Identifier)
	_, _ = fmt.Fprintf(&b,
		`<svg class="scip-graph" role="img" aria-label="%s" viewBox="0 0 %d %d" width="100%%" height="%d">`,
		template.HTMLEscapeString(title), width, height, height)
	_, _ = fmt.Fprintf(&b, `<title>%s</title>`, template.HTMLEscapeString(title))
	for _, a := range m.SupportingArtifacts {
		from := positions[m.ArtifactID]
		if pos, ok := positions[a.FromArtifactID]; ok {
			from = pos
		}
		to := positions[a.ArtifactID]
		_, _ = fmt.Fprintf(&b,
			`<line class="scip-edge" data-from-artifact-id="%d" data-to-artifact-id="%d" x1="%d" y1="%d" x2="%d" y2="%d"/>`,
			a.FromArtifactID, a.ArtifactID,
			from.x+scipGraphNodeW, from.y+scipGraphNodeH/2,
			to.x, to.y+scipGraphNodeH/2)
	}
	writeGraphNode(&b, positions[m.ArtifactID], m.ArtifactID, m.Kind, m.Identifier, 0, true)
	for _, a := range m.SupportingArtifacts {
		writeGraphNode(&b, positions[a.ArtifactID], a.ArtifactID, a.Kind, a.Identifier, a.Depth, false)
	}
	b.WriteString(`</svg>`)
	// #nosec G203 -- dynamic SVG labels are HTML-escaped before writing.
	return template.HTML(b.String())
}

func writeGraphNode(
	b *strings.Builder,
	pos graphPosition,
	artifactID int64,
	kind string,
	identifier string,
	depth int,
	root bool,
) {
	class := "scip-node"
	if root {
		class += " scip-node-root"
	}
	_, _ = fmt.Fprintf(b,
		`<g class="%s" data-artifact-id="%d" transform="translate(%d %d)">`,
		class, artifactID, pos.x, pos.y)
	_, _ = fmt.Fprintf(b, `<rect width="%d" height="%d" rx="6" ry="6"/>`, scipGraphNodeW, scipGraphNodeH)
	nodeKind := compactText(kind)
	if root {
		nodeKind = "matched " + nodeKind
	}
	_, _ = fmt.Fprintf(b, `<text class="scip-node-kind" x="10" y="16">%s</text>`, template.HTMLEscapeString(shortText(nodeKind, 28)))
	_, _ = fmt.Fprintf(b, `<text class="scip-node-label" x="10" y="32">%s</text>`, template.HTMLEscapeString(shortText(compactText(identifier), 34)))
	if depth > 0 {
		_, _ = fmt.Fprintf(b, `<text class="scip-depth" x="%d" y="16">d%d</text>`, scipGraphNodeW-24, depth)
	}
	b.WriteString(`</g>`)
}

func compactText(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func shortText(s string, maxLen int) string {
	if maxLen <= 3 || len([]rune(s)) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen-3]) + "..."
}
