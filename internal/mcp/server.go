// Package mcp exposes the fsdtrace database as an MCP server so Claude
// Code can query coverage, drift, and orphan endpoints directly.
//
// Hard constraint (SPEC §10): startup must stay under ~200ms cold.
// Therefore the DB and model clients are opened lazily — the first
// request that needs them pays the cost, not Init.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/cax/fsdtrace/internal/db"
	"github.com/cax/fsdtrace/internal/embed"
	"github.com/cax/fsdtrace/internal/llm"
)

// Version is the fsdtrace server version reported via the MCP
// initialize handshake.
const Version = "1.0.0"

// EnvBedrockBaseURL mirrors the CLI env name; tools using provider=bedrock
// read it on first use.
const EnvBedrockBaseURL = "BEDROCK_BASE_URL"

const (
	// ProviderBedrock keeps the original KrakenD -> Bedrock behavior.
	ProviderBedrock = "bedrock"
	// ProviderOpenAI uses OpenAI directly for generation and embeddings.
	ProviderOpenAI = "openai"
	// EnvOpenAIAPIKey is the direct OpenAI API key used by provider=openai.
	EnvOpenAIAPIKey = "OPENAI_API_KEY" // #nosec G101 -- env var name, not a credential.
)

// Config is the per-server configuration passed to NewServer.
type Config struct {
	// DBPath is the SQLite file the server reads from. Required.
	DBPath string
	// Provider selects bedrock or openai. Empty defaults to bedrock.
	Provider string
	// BedrockBaseURL overrides the env value when non-empty.
	BedrockBaseURL string
	// OpenAIBaseURL overrides the OpenAI API URL when non-empty.
	OpenAIBaseURL string
	// EmbeddingModel overrides the default embedding model when non-empty.
	EmbeddingModel string
	// JudgmentModel overrides the default judgment model when non-empty.
	JudgmentModel string
	// HTTPClient is used by model providers; tests inject a recorder.
	HTTPClient embed.HTTPDoer
}

// Server holds shared, lazy-initialised dependencies for every handler.
type Server struct {
	cfg Config

	openDBOnce  sync.Once
	openDBErr   error
	openDB      *db.DB
	openDBClose func()

	openModelsOnce sync.Once
	openModelsErr  error
	openGenerator  llm.Generator
	openEmbedder   embed.Embedder
}

// NewServer builds the MCP server, registers tools and resources, and
// returns the underlying *server.MCPServer ready to be served on stdio
// or SSE. Note: this does NOT touch the DB; that happens on first
// request.
func NewServer(cfg Config) (*server.MCPServer, *Server) {
	// Note: empty cfg.DBPath is intentional. Validation is deferred to
	// the first DB use so the MCP initialise handshake works even when
	// the operator hasn't pointed at a database yet.
	mcpSrv := server.NewMCPServer(
		"fsdtrace", Version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, true),
		server.WithRecovery(),
		server.WithInputSchemaValidation(),
	)
	s := &Server{cfg: cfg}
	registerTools(mcpSrv, s)
	registerResources(mcpSrv, s)
	return mcpSrv, s
}

// ServeStdio runs the JSON-RPC stdio loop. Blocks until ctx is done.
func ServeStdio(ctx context.Context, srv *server.MCPServer) error {
	// server.ServeStdio doesn't take a context; rely on stdin EOF or
	// ctx cancellation via stdin close. We close on ctx done.
	doneCh := make(chan error, 1)
	go func() { doneCh <- server.ServeStdio(srv) }()
	select {
	case <-ctx.Done():
		// Returning closes os.Stdin's reader effectively when the
		// process exits; for MCP stdio the client controls lifetime.
		return nil
	case err := <-doneCh:
		return err
	}
}

// ServeSSE runs the MCP server over Server-Sent Events on addr (e.g.
// "127.0.0.1:8765"). Blocks until ctx is done; shutdown is graceful
// up to 5 seconds.
func ServeSSE(ctx context.Context, srv *server.MCPServer, addr string) error {
	sse := server.NewSSEServer(srv)
	doneCh := make(chan error, 1)
	go func() { doneCh <- sse.Start(addr) }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return sse.Shutdown(shutdownCtx)
	case err := <-doneCh:
		return err
	}
}

// DB returns the lazily-opened *db.DB (and remembers it for the rest
// of the process lifetime). All handlers call this; tests can pre-open
// a DB and inject it via WithDB.
func (s *Server) DB(ctx context.Context) (*db.DB, error) {
	s.openDBOnce.Do(func() {
		if s.openDB != nil {
			return // injected via WithDB
		}
		if s.cfg.DBPath == "" {
			s.openDBErr = errors.New("mcp: no --db path configured")
			return
		}
		d, err := db.Open(ctx, s.cfg.DBPath)
		if err != nil {
			s.openDBErr = fmt.Errorf("mcp: open db %s: %w", s.cfg.DBPath, err)
			return
		}
		if err := d.ApplySchema(ctx); err != nil {
			_ = d.Close()
			s.openDBErr = fmt.Errorf("mcp: apply schema: %w", err)
			return
		}
		s.openDB = d
		s.openDBClose = func() { _ = d.Close() }
	})
	return s.openDB, s.openDBErr
}

// WithDB pre-installs a DB so the lazy opener is a no-op. Tests use
// this to share a fixture DB.
func (s *Server) WithDB(d *db.DB) {
	s.openDB = d
	s.openDBOnce.Do(func() {})
}

// Models returns the lazily-built generator + query embedder. Returns
// an explanatory error when the selected provider is not configured.
func (s *Server) Models(_ context.Context) (llm.Generator, embed.Embedder, error) {
	s.openModelsOnce.Do(func() {
		switch providerName(s.cfg.Provider) {
		case ProviderOpenAI:
			s.openGenerator, s.openEmbedder, s.openModelsErr = s.openOpenAIModels()
		default:
			s.openGenerator, s.openEmbedder, s.openModelsErr = s.openBedrockModels()
		}
	})
	return s.openGenerator, s.openEmbedder, s.openModelsErr
}

// WithBedrock pre-installs Bedrock dependencies (used by tests).
func (s *Server) WithBedrock(c *embed.BedrockClient, e embed.Embedder) {
	s.openGenerator = llm.NewBedrockGenerator(c)
	s.openEmbedder = e
	s.openModelsOnce.Do(func() {})
}

// WithModels pre-installs provider-neutral model dependencies.
func (s *Server) WithModels(g llm.Generator, e embed.Embedder) {
	s.openGenerator = g
	s.openEmbedder = e
	s.openModelsOnce.Do(func() {})
}

// Close releases the lazy DB if one was opened. Safe to call multiple
// times.
func (s *Server) Close() {
	if s.openDBClose != nil {
		s.openDBClose()
		s.openDBClose = nil
	}
}

// resultErr is a thin helper that wraps an error into an MCP tool
// result with the error flag set.
func resultErr(err error) *mcp.CallToolResult {
	return mcp.NewToolResultErrorf("%s", err.Error())
}

func (s *Server) openBedrockModels() (llm.Generator, embed.Embedder, error) {
	baseURL := s.cfg.BedrockBaseURL
	if baseURL == "" {
		baseURL = os.Getenv(EnvBedrockBaseURL)
	}
	if baseURL == "" {
		return nil, nil, fmt.Errorf("mcp: %s not set; set it to the KrakenD route to enable embedding-based tools", EnvBedrockBaseURL)
	}
	opts := []embed.ClientOption{}
	if s.cfg.HTTPClient != nil {
		opts = append(opts, embed.WithHTTPClient(s.cfg.HTTPClient))
	} else {
		opts = append(opts, embed.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}))
	}
	c, err := embed.NewClient(baseURL, opts...)
	if err != nil {
		return nil, nil, err
	}
	return llm.NewBedrockGenerator(c), embed.NewBedrockEmbedder(c, s.cfg.EmbeddingModel, embed.PurposeQuery), nil
}

func (s *Server) openOpenAIModels() (llm.Generator, embed.Embedder, error) {
	apiKey := os.Getenv(EnvOpenAIAPIKey)
	if apiKey == "" {
		return nil, nil, fmt.Errorf("mcp: %s not set; set it to use provider=openai", EnvOpenAIAPIKey)
	}
	genOpts := []llm.OpenAIOption{}
	embOpts := []embed.OpenAIEmbedderOption{}
	if s.cfg.OpenAIBaseURL != "" {
		genOpts = append(genOpts, llm.WithOpenAIBaseURL(s.cfg.OpenAIBaseURL))
		embOpts = append(embOpts, embed.WithOpenAIEmbedderBaseURL(s.cfg.OpenAIBaseURL))
	}
	if s.cfg.HTTPClient != nil {
		genOpts = append(genOpts, llm.WithOpenAIHTTPClient(s.cfg.HTTPClient))
		embOpts = append(embOpts, embed.WithOpenAIEmbedderHTTPClient(s.cfg.HTTPClient))
	}
	g, err := llm.NewOpenAIClient(apiKey, genOpts...)
	if err != nil {
		return nil, nil, err
	}
	e, err := embed.NewOpenAIEmbedder(apiKey, s.cfg.EmbeddingModel, embed.PurposeQuery, embOpts...)
	if err != nil {
		return nil, nil, err
	}
	return g, e, nil
}

func providerName(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case ProviderOpenAI:
		return ProviderOpenAI
	default:
		return ProviderBedrock
	}
}
