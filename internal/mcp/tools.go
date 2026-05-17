package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerTools wires every SPEC §7.6 tool into mcpSrv. Each tool's
// description is action-oriented, lists input keys with examples, and
// notes side effects.
func registerTools(mcpSrv *server.MCPServer, s *Server) {
	mcpSrv.AddTool(mcp.NewTool(
		"fsd_search_features",
		mcp.WithDescription("Semantic search over Functional Requirement (FR) rows using configured embeddings. Returns id, title, section, acceptance, and KNN distance. Example: query='POST /api/v1/orders'."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("query", mcp.Description("Free-text search prompt; embedded with the configured Bedrock embedding model."), mcp.Required()),
		mcp.WithNumber("top_k", mcp.Description("How many results to return (default 10).")),
	), s.handleSearchFeatures)

	mcpSrv.AddTool(mcp.NewTool(
		"fsd_search_code",
		mcp.WithDescription("Semantic search over indexed code artifacts. Returns id, kind, identifier, file:line, signature, and distance. Optionally filter by kind (e.g. rest_endpoint, kafka_listener)."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("query", mcp.Description("Free-text search prompt; embedded with the configured Bedrock embedding model."), mcp.Required()),
		mcp.WithString("kind", mcp.Description("Restrict to one artifact kind, e.g. rest_endpoint, kafka_listener, scheduled_job, security_rule, entity, config_props, exception_handler.")),
		mcp.WithNumber("top_k", mcp.Description("How many results to return (default 10).")),
	), s.handleSearchCode)

	mcpSrv.AddTool(mcp.NewTool(
		"fsd_get_feature",
		mcp.WithDescription("Get one FR with its matched artifacts and test coverage from the latest match run (or --run-id). Returns the feature row, the per-artifact verdict list with evidence, and a coverage_status."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("id", mcp.Description("FR identifier, e.g. FR-042."), mcp.Required()),
		mcp.WithString("run_id", mcp.Description("Optional match run id; defaults to the most recent.")),
	), s.handleGetFeature)

	mcpSrv.AddTool(mcp.NewTool(
		"fsd_get_artifact",
		mcp.WithDescription("Get one code artifact with its linked FRs and tests. Returns the full artifact row, every match against it, and any test that targets it."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithNumber("id", mcp.Description("Numeric artifact id (code_artifacts.id)."), mcp.Required()),
		mcp.WithString("run_id", mcp.Description("Optional match run id; defaults to the most recent.")),
	), s.handleGetArtifact)

	mcpSrv.AddTool(mcp.NewTool(
		"fsd_list_unmatched",
		mcp.WithDescription("List FRs that have no `implements` verdict in the latest match run. Optionally filter by FSD section."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("section", mcp.Description("FSD section name to filter by, e.g. 'Authentication'.")),
		mcp.WithString("run_id", mcp.Description("Optional match run id; defaults to the most recent.")),
	), s.handleListUnmatched)

	mcpSrv.AddTool(mcp.NewTool(
		"fsd_list_orphans",
		mcp.WithDescription("List public-surface code artifacts (REST endpoints, listeners, scheduled jobs, exception handlers) with no FR mapping. Optionally filter by kind."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("kind", mcp.Description("Restrict to one kind: rest_endpoint, kafka_listener, rabbit_listener, jms_listener, event_listener, scheduled_job, exception_handler.")),
		mcp.WithString("run_id", mcp.Description("Optional match run id; defaults to the most recent.")),
	), s.handleListOrphans)

	mcpSrv.AddTool(mcp.NewTool(
		"fsd_drift_report",
		mcp.WithDescription("All `drifts` verdicts for the latest match run, with file:line evidence."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("run_id", mcp.Description("Optional match run id; defaults to the most recent.")),
	), s.handleDriftReport)

	mcpSrv.AddTool(mcp.NewTool(
		"fsd_coverage_summary",
		mcp.WithDescription("Per-FSD-section roll-up: total / implemented / drifts / missing / untested counts."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithString("run_id", mcp.Description("Optional match run id; defaults to the most recent.")),
	), s.handleCoverageSummary)

	mcpSrv.AddTool(mcp.NewTool(
		"fsd_rematch_feature",
		mcp.WithDescription("Re-run the matcher pipeline for one FR and persist the result. NOT read-only — costs one Bedrock judgment call. Requires BEDROCK_BASE_URL."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithString("id", mcp.Description("FR identifier, e.g. FR-042."), mcp.Required()),
		mcp.WithString("run_id", mcp.Description("Run id to upsert the matches under (default: rematch-<unix-ts>).")),
		mcp.WithNumber("top_k", mcp.Description("KNN candidate cap (default 15).")),
	), s.handleRematchFeature)
}
