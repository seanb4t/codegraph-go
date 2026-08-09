package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/daemon"
	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/mcp"
	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/watch"
)

// codegraphMCPToolsEnv is the operator NARROWING filter (D-08a, MCP-02): a
// comma-separated list of companion tool names. All eight tools register by
// default; setting this variable narrows the surface to the companions it
// names, plus the always-visible codegraph_explore. Setting it to the empty
// string narrows to codegraph_explore alone.
//
// Read with os.LookupEnv, never os.Getenv — see mcp.ResolveCompanions for
// why "unset" and "set to empty" must answer differently here.
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

// watchRetryInterval is the base cadence (before jitter) serveWatchStart's
// background goroutine waits between daemon.RunWithRetry's ErrLockLive
// retries (D-14/WATCH-04) — a slow-enough cadence to avoid needlessly
// hammering a live lock holder, fast enough that a session converges to
// sole-writer status well within a typical human interaction window once the
// holder exits. Jitter is applied inside daemon.RunWithRetry itself.
const watchRetryInterval = 30 * time.Second

// serveWatchStart is WATCH-02's off-handshake-path seam (D-06/D-08),
// mirroring serveServerPaths's WR-01 precedent above: extracted so a test
// can pin THIS function's actual behavior — the watcher startup newServeCmd's
// RunE genuinely defers — rather than a hand-built replica living only in a
// test file. It spawns exactly one goroutine and returns (cancel, done)
// BEFORE that goroutine performs any of daemon.New, the watch-policy check
// (inside daemon.Run), lock acquisition, or watch.Open's recursive fsnotify
// walk. Moving any of that work back above this function's goroutine
// boundary — the literal WATCH-02 regression — is the exact mutation
// TestServeWatchStartDeferred (serve_test.go) must catch.
//
// A no-op (cancel, already-closed done) pair is returned when !hasIndex
// (MCP-03: no watcher, but RunE's control flow stays uniform — the caller
// always defers cancel()/<-done unconditionally). hasIndex is a
// startup-time snapshot (IN-09, deliberate): an index created mid-session
// (`codegraph init` while serve is running) is served live by per-call
// query resolution but does NOT retroactively start the watcher —
// auto-sync begins on the next serve --mcp session, and the D-04a
// stale/mtime fallback keeps staleness observable meanwhile.
//
// onWatchWorkStart, when non-nil, is invoked at the very start of the
// goroutine's real work, before daemon.New runs — a test-only control seam
// mirroring daemon.Daemon's onSyncStart convention (internal/daemon/daemon.go
// lines 67-75), letting a test deterministically observe "the goroutine's
// real work has started" without a sleep/timeout race. Production callers
// pass nil.
func serveWatchStart(
	repoPath string,
	hasIndex bool,
	noWatch bool,
	forceWatch bool,
	opts indexer.Options,
	stderr io.Writer,
	onWatchWorkStart func(),
) (cancel func(), done <-chan struct{}) {
	if !hasIndex {
		noop := make(chan struct{})
		close(noop)
		return func() {}, noop
	}

	watchCtx, cancelWatch := context.WithCancel(context.Background())
	watchDone := make(chan struct{})

	go func() {
		defer close(watchDone)

		if onWatchWorkStart != nil {
			onWatchWorkStart()
		}

		probe := watch.Probe{NoWatch: noWatch, ForceWatch: forceWatch}

		d, err := daemon.New(repoPath, opts, daemon.WithProbe(probe))
		if err != nil {
			fmt.Fprintf(stderr, "codegraph serve: watcher: %v\n", err)
			return
		}

		logOnce := sync.OnceFunc(func() {
			fmt.Fprintln(stderr, "codegraph serve: a daemon is already running, deferring to it")
		})

		runErr := daemon.RunWithRetry(watchCtx, d, watchRetryInterval, logOnce)
		var de *watch.DisabledError
		switch {
		case runErr == nil, errors.Is(runErr, context.Canceled):
			// Clean shutdown (RunE returned and cancelled watchCtx) — either
			// mid-watch (d.Run returns nil on ctx.Done()) or mid-retry-sleep
			// (RunWithRetry returns ctx.Err()). Neither is an error worth
			// surfacing.
		case errors.As(runErr, &de):
			// D-12/D-13: verbatim TS disabled message, stderr-only
			// (model-invisible). Terminal — no retry: policy doesn't change
			// mid-session (Pitfall 2: --no-watch must still print this).
			// IN-05: the reason is extracted from the typed error
			// daemon.Run returned — the exact string its own policy gate
			// saw — never re-derived with a fresh WatchDisabledReason call
			// that could desynchronize.
			// IN-02 (round 5): the case itself is gated on errors.As, not
			// errors.Is(watch.ErrWatchDisabled) — symmetric with
			// cli/daemon.go — so a bare or fmt.Errorf-wrapped sentinel from
			// some future producer falls through to default (which renders
			// DisabledError.Error()'s reason-embedding text via %v) instead
			// of printing a malformed "disabled — ." line with an empty
			// reason. The malformed shape is unreachable by construction.
			fmt.Fprintf(stderr, "[CodeGraph MCP] File watcher disabled — %s. "+
				"The graph will not auto-update; run `codegraph sync` "+
				"(or install the git sync hooks via `codegraph init`) to refresh.\n", de.Reason)
		default:
			fmt.Fprintf(stderr, "codegraph serve: watcher: %v\n", runErr)
		}
	}()

	return cancelWatch, watchDone
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
	var forceWatch bool
	var noWatch bool

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
					// CR-01 scenario 3 (03-REVIEW.md): a lock-held failure
					// here means another codegraph process (a sibling
					// session's flush or reconcile) is actively syncing the
					// SAME store right now — which is exactly the "graph
					// will be fresh" case. Killing this session's MCP server
					// over it is strictly worse than starting against a
					// possibly seconds-stale graph the stale banner already
					// covers (D-04a), so degrade to a stderr warning and
					// continue. Every non-lock error stays fatal: a corrupt
					// or unreadable store is not something serving through
					// papers over. errors.Is on the ErrStoreLocked sentinel
					// (classified inside graphstore.Open, the only seam with
					// unambiguous provenance — 03-REVIEW-2.md WR-01) means a
					// permission error anywhere in Sync's chain can never
					// masquerade as lock contention here.
					if !errors.Is(err, graphstore.ErrStoreLocked) {
						return err
					}
					fmt.Fprintf(cmd.ErrOrStderr(),
						"codegraph serve: startup reconcile skipped — the graph store is locked by another codegraph process (likely an in-flight sync; the graph will refresh shortly): %v\n", err)
				}
			}

			// D-01/D-06/D-08: watcher startup is now default-ON whenever an
			// index exists — no `if watchMode` gate here. serveWatchStart
			// spawns the background goroutine and returns immediately;
			// hasIndex is the ONLY gate (MCP-03: no index, no watcher), and
			// it lives inside serveWatchStart itself so this call site stays
			// unconditional. WR-04's Quiet mirrors the reconcile Sync call
			// above.
			cancelWatchStart, watchStartDone := serveWatchStart(
				repoPath, hasIndex, noWatch, forceWatch,
				indexer.Options{Quiet: true}, cmd.ErrOrStderr(), nil,
			)
			defer func() {
				cancelWatchStart()
				<-watchStartDone
			}()

			// os.LookupEnv, not os.Getenv: an UNSET variable means "register
			// all eight tools" while a variable SET to the empty string means
			// "register codegraph_explore alone". os.Getenv returns "" for
			// both, so reading it here would silently reinstate the
			// pre-inversion explore-only default for every user on earth.
			rawFilter, filterSet := os.LookupEnv(codegraphMCPToolsEnv)
			companions, unknown := mcp.ResolveCompanions(rawFilter, filterSet)
			mcp.WarnToolFilterTo(cmd.ErrOrStderr(), unknown, companions)

			// CR-01: repoPath (the RESOLVED index root) is the confinement
			// root; start (the caller's actual cwd, captured above BEFORE
			// repoPath overwrote it for storeDir/daemon.New's purposes) must
			// ALSO survive to BuildServer as the handlers' default path — a
			// literal recurrence of Phase-1 CR-02 otherwise, since passing
			// repoPath for both collapses startPath == repoRoot and every
			// worktree-mismatch check silently short-circuits to nil on
			// every production call. See mcp.BuildServer's doc comment.
			//
			// SDK-02: bootstraps entirely through the mcp.Server seam —
			// this file names no MCP SDK package. cmd.ErrOrStderr() is
			// already this file's stderr idiom (mcp.WarnToolFilterTo
			// above uses it too); VRFY-03's always-on session line rides
			// the same writer.
			s := mcp.NewStdioServer(hasIndex, companions, repoPath, start, cmd.ErrOrStderr())
			return s.ServeStdio()
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().BoolVar(&mcpMode, "mcp", false, "run the stdio MCP server")
	cmd.Flags().BoolVar(&forceWatch, "watch", false, "Force the file watcher on, overriding the WSL2/slow-filesystem auto-off (the CLI twin of CODEGRAPH_FORCE_WATCH=1)")
	cmd.Flags().BoolVar(&noWatch, "no-watch", false, "Disable the file watcher (no auto-sync; useful on slow filesystems like WSL2 /mnt drives)")
	cmd.MarkFlagsMutuallyExclusive("no-watch", "watch")

	return cmd
}
