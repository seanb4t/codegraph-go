package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/daemon"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/mcp"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// codegraphMCPToolsEnv is the operator allowlist env var (D-08a, MCP-02):
// a comma-separated list of companion tool names to register alongside
// the always-visible codegraph_explore.
const codegraphMCPToolsEnv = "CODEGRAPH_MCP_TOOLS"

// serveServerPaths computes BuildServer's two DELIBERATELY DISTINCT
// arguments (CR-01) from start, the caller's actual starting directory:
// repoPath, the confinement root (start itself, overwritten with the
// RESOLVED index root only when query.ResolveCodegraphDir finds one at or
// above start), and hasIndex, whether an index was found at all (MCP-03).
//
// WR-01 (02-REVIEW-2.md): extracted so a test can pin THIS function's
// actual output — the derivation newServeCmd's RunE really performs —
// rather than a hand-built replica living only inside a test file, which
// proves nothing about whether serve.go itself still passes the caller's
// start path through to BuildServer. The re-review reintroduced CR-01 at
// its root cause (BuildServer(hasIndex, allowlist, repoPath, repoPath))
// and the entire suite, golden corpus included, stayed green — because
// nothing exercised serve.go's own wiring. See serve_test.go.
func serveServerPaths(start string) (repoPath string, hasIndex bool, err error) {
	repoPath = start
	if dir, dirErr := query.ResolveCodegraphDir(start); dirErr == nil {
		return dir, true, nil
	} else if !errors.Is(dirErr, query.ErrNotInitialized) {
		return "", false, dirErr
	}
	return repoPath, false, nil
}

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
	var watchMode bool

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

			repoPath, hasIndex, err := serveServerPaths(start)
			if err != nil {
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

			// D-05/SYNC-04 in-process watcher fallback: where a separate
			// `codegraph daemon` process is undesired, --watch runs the same
			// watch/debounce/Sync loop in-process, under the SAME lockfile
			// internal/daemon uses. That shared lock makes an in-process
			// watcher and a standalone daemon mutually exclusive (T-04-08-01)
			// — if a live daemon already holds it, Run returns ErrLockLive
			// and serve simply defers to that daemon rather than failing.
			if watchMode && hasIndex {
				watchCtx, cancelWatch := context.WithCancel(context.Background())
				// WR-04: the in-process fallback has no CLI flags of its own
				// for daemon-side sync customization — Quiet mirrors the
				// reconcile Sync call above, since this watcher's flushes are
				// only ever logged (internal/daemon.flush), never printed to
				// this command's stdout.
				d, err := daemon.New(repoPath, indexer.Options{Quiet: true})
				if err != nil {
					cancelWatch()
					return err
				}
				watchDone := make(chan struct{})
				go func() {
					defer close(watchDone)
					if runErr := d.Run(watchCtx); runErr != nil {
						if errors.Is(runErr, daemon.ErrLockLive) {
							fmt.Fprintln(cmd.ErrOrStderr(), "codegraph serve: --watch: a daemon is already running, deferring to it")
						} else {
							fmt.Fprintf(cmd.ErrOrStderr(), "codegraph serve: --watch: %v\n", runErr)
						}
					}
				}()
				defer func() {
					cancelWatch()
					<-watchDone
				}()
			}

			allowlist, unknown := mcp.ParseAllowlist(os.Getenv(codegraphMCPToolsEnv))
			mcp.WarnUnknownToolsTo(cmd.ErrOrStderr(), unknown)

			// CR-01: repoPath (the RESOLVED index root) is the confinement
			// root; start (the caller's actual cwd, captured above BEFORE
			// repoPath overwrote it for storeDir/daemon.New's purposes) must
			// ALSO survive to BuildServer as the handlers' default path — a
			// literal recurrence of Phase-1 CR-02 otherwise, since passing
			// repoPath for both collapses startPath == repoRoot and every
			// worktree-mismatch check silently short-circuits to nil on
			// every production call. See mcp.BuildServer's doc comment.
			s := mcp.BuildServer(hasIndex, allowlist, repoPath, start)
			return server.ServeStdio(s)
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().BoolVar(&mcpMode, "mcp", false, "run the stdio MCP server")
	cmd.Flags().BoolVar(&watchMode, "watch", false, "run an in-process watcher alongside the MCP server, under the same lockfile a standalone `codegraph daemon` uses (mutually exclusive with one)")

	return cmd
}
