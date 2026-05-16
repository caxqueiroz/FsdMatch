// Command fsdtrace is the CLI + MCP server entrypoint.
// All subcommands operate on a single SQLite file (--db).
package main

import (
	"fmt"
	"os"

	"github.com/cax/fsdtrace/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
