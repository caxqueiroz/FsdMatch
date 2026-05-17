package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/cax/fsdtrace/internal/mcp"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server for Claude Code integration",
	}
	var (
		transport      string
		addr           string
		bedrockBaseURL string
		embeddingModel string
		judgmentModel  string
	)
	serve := &cobra.Command{
		Use:   "serve",
		Short: "Start the MCP server (stdio or sse)",
		Long: `Runs the SPEC §7.6 tool catalog over the chosen transport. The DB and
Bedrock client open lazily on first request so cold start stays under
~200ms (SPEC §10).

Transports:
  stdio (default) — Claude Code launches the binary as a subprocess.
  sse             — listens on --addr and serves the MCP protocol over
                    Server-Sent Events. Use this for remote setups
                    (e.g. claude.ai/code over a tunnel).
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signalCtx()
			defer cancel()

			cfg, err := loadAppConfig(opts.cfgPath, nil)
			if err != nil {
				return err
			}
			baseURL := cfg.bedrockURL()
			if cmd.Flags().Changed("bedrock-base-url") && bedrockBaseURL != "" {
				baseURL = bedrockBaseURL
			}
			embeddingModel := cfg.model(modelEmbedding, embeddingModel, cmd.Flags().Changed("embedding-model"))
			judgmentModel := cfg.model(modelJudgment, judgmentModel, cmd.Flags().Changed("judgment-model"))

			srv, state := mcp.NewServer(mcp.Config{
				DBPath:         opts.dbPath,
				BedrockBaseURL: baseURL,
				EmbeddingModel: embeddingModel,
				JudgmentModel:  judgmentModel,
			})
			defer state.Close()
			switch transport {
			case "stdio":
				return mcp.ServeStdio(ctx, srv)
			case "sse":
				return mcp.ServeSSE(ctx, srv, addr)
			default:
				return fmt.Errorf("unknown transport %q (want stdio|sse)", transport)
			}
		},
	}
	serve.Flags().StringVar(&transport, "transport", "stdio", "transport: stdio|sse")
	serve.Flags().StringVar(&addr, "addr", "127.0.0.1:8765", "address to bind for sse transport")
	serve.Flags().StringVar(&bedrockBaseURL, "bedrock-base-url", "", "Bedrock route (flag > env > config)")
	serve.Flags().StringVar(&embeddingModel, "embedding-model", "", "Bedrock embedding model id (flag > env > config > default)")
	serve.Flags().StringVar(&judgmentModel, "judgment-model", "", "Bedrock Claude model id for fsd_rematch_feature (flag > env > config > default)")
	cmd.AddCommand(serve)
	return cmd
}

// ----- install-claude-code -----------------------------------------------

func newInstallClaudeCodeCmd() *cobra.Command {
	var (
		configPath string
		serverName string
		dryRun     bool
	)
	cmd := &cobra.Command{
		Use:   "install-claude-code",
		Short: "Add fsdtrace to a Claude Code MCP config (~/.claude.json by default)",
		Long: `Idempotently inserts an entry into the mcpServers map of a Claude Code
config file. Existing keys outside mcpServers.<name> are preserved.

Defaults to ~/.claude.json. Use --config to target a project-local config.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			binPath, err := absoluteSelfPath()
			if err != nil {
				return err
			}
			dbPath, err := filepath.Abs(opts.dbPath)
			if err != nil {
				return err
			}

			path, err := resolveClaudeConfigPath(configPath)
			if err != nil {
				return err
			}

			cfg, err := readJSONObject(path)
			if err != nil {
				return err
			}
			servers, _ := cfg["mcpServers"].(map[string]any)
			if servers == nil {
				servers = map[string]any{}
			}
			servers[serverName] = map[string]any{
				"command": binPath,
				"args":    []any{"--db", dbPath, "mcp", "serve"},
			}
			cfg["mcpServers"] = servers

			if dryRun {
				body, err := json.MarshalIndent(cfg, "", "  ")
				if err != nil {
					return err
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(body))
				return nil
			}
			if err := writeJSONObject(path, cfg); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(),
				"installed mcpServers.%s = {%s --db %s mcp serve} into %s\n",
				serverName, binPath, dbPath, path)
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "Claude Code config path (default ~/.claude.json)")
	cmd.Flags().StringVar(&serverName, "name", "fsdtrace", "MCP server name to write")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the merged config to stdout instead of writing")
	return cmd
}

func resolveClaudeConfigPath(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating home dir: %w", err)
	}
	return filepath.Join(home, ".claude.json"), nil
}

func readJSONObject(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path) // #nosec G304 -- operator-supplied path
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return out, nil
}

func writeJSONObject(path string, v map[string]any) error {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o600) // #nosec G306 -- user config
}

func absoluteSelfPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return "", err
	}
	return abs, nil
}
