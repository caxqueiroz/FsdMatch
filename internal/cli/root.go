// Package cli wires the cobra command tree. Subcommands live in this
// package as separate files for readability.
package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/cax/fsdtrace/internal/mcp"
)

// globalOpts are populated by RootCmd persistent flags.
type globalOpts struct {
	dbPath   string
	cfgPath  string
	logLevel string
	runID    string
}

var opts globalOpts

// NewRootCmd returns the top-level fsdtrace command.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "fsdtrace",
		Short:         "Validate an FSD against a Java/Spring Boot codebase",
		Long:          "fsdtrace atomizes a Functional Specification Document, indexes a Spring Boot codebase, and produces a traceability matrix with file:line evidence.",
		Version:       mcp.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(*cobra.Command, []string) error {
			return setupLogger(opts.logLevel)
		},
	}

	root.PersistentFlags().StringVar(&opts.dbPath, "db", "./fsdtrace.db", "SQLite file path")
	root.PersistentFlags().StringVar(&opts.cfgPath, "config", "", "config file (default: search ./fsdtrace.yaml, ~/.fsdtrace.yaml)")
	root.PersistentFlags().StringVar(&opts.logLevel, "log-level", "info", "log level: debug|info|warn|error")
	root.PersistentFlags().StringVar(&opts.runID, "run-id", "", "stable run identifier (default: generated)")

	root.AddCommand(
		newInitCmd(),
		newIngestCmd(),
		newIndexCmd(),
		newEmbedCmd(),
		newMatchCmd(),
		newReportCmd(),
		newTraceCmd(),
		newMCPCmd(),
		newInstallClaudeCodeCmd(),
		newStatusCmd(),
	)
	return root
}

// signalCtx returns a context cancelled on SIGINT/SIGTERM.
func signalCtx() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
}

func setupLogger(level string) error {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info", "":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		return fmt.Errorf("unknown log level %q", level)
	}
	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: lvl})
	slog.SetDefault(slog.New(h))
	return nil
}
