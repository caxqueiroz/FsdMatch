package mcp

import (
	"bytes"
	"context"
	"encoding/json"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/cax/fsdtrace/internal/report"
)

// registerResources wires the three SPEC §7.6 resources. They render
// the live report (latest match run) on every read.
func registerResources(mcpSrv *server.MCPServer, s *Server) {
	mcpSrv.AddResource(
		mcpgo.NewResource("fsd://coverage",
			"Coverage report",
			mcpgo.WithMIMEType("text/markdown"),
			mcpgo.WithResourceDescription("Per-FSD-section coverage matrix from the most recent match run, in markdown."),
		),
		s.handleCoverageResource,
	)
	mcpSrv.AddResource(
		mcpgo.NewResource("fsd://drift",
			"Drift report",
			mcpgo.WithMIMEType("text/markdown"),
			mcpgo.WithResourceDescription("All `drifts` verdicts in the most recent match run, with file:line evidence, as markdown."),
		),
		s.handleDriftResource,
	)
	mcpSrv.AddResource(
		mcpgo.NewResource("fsd://features",
			"Feature catalog",
			mcpgo.WithMIMEType("application/json"),
			mcpgo.WithResourceDescription("All FRs in the database with title, section, and acceptance criteria, as JSON."),
		),
		s.handleFeaturesResource,
	)
}

func (s *Server) handleCoverageResource(ctx context.Context, req mcpgo.ReadResourceRequest) ([]mcpgo.ResourceContents, error) {
	d, err := s.DB(ctx)
	if err != nil {
		return nil, err
	}
	rep, err := report.Load(ctx, d, "")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := report.RenderMarkdownCoverage(&buf, rep); err != nil {
		return nil, err
	}
	return []mcpgo.ResourceContents{mcpgo.TextResourceContents{
		URI:      req.Params.URI,
		MIMEType: "text/markdown",
		Text:     buf.String(),
	}}, nil
}

func (s *Server) handleDriftResource(ctx context.Context, req mcpgo.ReadResourceRequest) ([]mcpgo.ResourceContents, error) {
	d, err := s.DB(ctx)
	if err != nil {
		return nil, err
	}
	rep, err := report.Load(ctx, d, "")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := report.RenderMarkdownDrift(&buf, rep); err != nil {
		return nil, err
	}
	return []mcpgo.ResourceContents{mcpgo.TextResourceContents{
		URI:      req.Params.URI,
		MIMEType: "text/markdown",
		Text:     buf.String(),
	}}, nil
}

func (s *Server) handleFeaturesResource(ctx context.Context, req mcpgo.ReadResourceRequest) ([]mcpgo.ResourceContents, error) {
	d, err := s.DB(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := d.SQL().QueryContext(ctx,
		`SELECT id, title, COALESCE(fsd_section, ''), description, acceptance
		   FROM features ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	type feat struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Section     string   `json:"section"`
		Description string   `json:"description"`
		Acceptance  []string `json:"acceptance"`
	}
	var out []feat
	for rows.Next() {
		var (
			f       feat
			accJSON string
		)
		if err := rows.Scan(&f.ID, &f.Title, &f.Section, &f.Description, &accJSON); err != nil {
			return nil, err
		}
		if accJSON != "" {
			_ = json.Unmarshal([]byte(accJSON), &f.Acceptance)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	body, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return []mcpgo.ResourceContents{mcpgo.TextResourceContents{
		URI:      req.Params.URI,
		MIMEType: "application/json",
		Text:     string(body),
	}}, nil
}
