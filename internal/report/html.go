package report

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
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
