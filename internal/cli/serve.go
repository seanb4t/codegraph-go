package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/mcp"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// codegraphMCPToolsEnv is the operator allowlist env var (D-08a, MCP-02):
// a comma-separated list of companion tool names to register alongside
// the always-visible codegraph_explore.
const codegraphMCPToolsEnv = "CODEGRAPH_MCP_TOOLS"

// newServeCmd builds `codegraph serve --mcp` (MCP-01 command surface,
// D-08a): runs the stdio MCP server built in 03-07 (internal/mcp).
// --mcp is required — stdio is the only transport v1 ships (HTTP/SSE is
// v2, SERVER-01) but the flag makes the transport selection explicit and
// future-proof. Unlike every other query command, an absent .codegraph/
// is NOT an error here (MCP-03): the server still starts and completes
// MCP init, advertising zero tools, so agents fall back to built-ins
// gracefully instead of the connection being refused.
func newServeCmd() *cobra.Command {
	var path string
	var mcpMode bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the codegraph MCP server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !mcpMode {
				return fmt.Errorf("codegraph serve: --mcp is required (stdio is the only supported transport)")
			}

			start, err := resolveStartPath(path)
			if err != nil {
				return err
			}

			repoPath := start
			hasIndex := false
			if dir, err := query.ResolveCodegraphDir(start); err == nil {
				hasIndex = true
				repoPath = dir
			} else if !errors.Is(err, query.ErrNotInitialized) {
				return err
			}

			// D-06/SYNC-03: on (re)connect, reconcile any offline changes
			// (made while the watcher/daemon was down) via the same
			// indexer.Sync entry `codegraph sync` and the daemon use —
			// before the first tool is served, so the first
			// codegraph_explore reads a current graph. A no-op when
			// nothing changed (the stat pre-filter makes this cheap by
			// construction). MCP-03's absent-index case is unaffected:
			// hasIndex false skips reconcile entirely.
			if hasIndex {
				storeDir := filepath.Join(repoPath, codegraphDirName, storeDirName)
				if _, err := indexer.Sync(repoPath, storeDir, indexer.Options{Quiet: true}); err != nil {
					return err
				}
			}

			allowlist, unknown := mcp.ParseAllowlist(os.Getenv(codegraphMCPToolsEnv))
			mcp.WarnUnknownToolsTo(cmd.ErrOrStderr(), unknown)

			s := mcp.BuildServer(hasIndex, allowlist, repoPath)
			return server.ServeStdio(s)
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().BoolVar(&mcpMode, "mcp", false, "run the stdio MCP server")

	return cmd
}
