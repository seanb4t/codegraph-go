# Phase 3: Watcher-on-MCP Default - Pattern Map

**Mapped:** 2026-07-16
**Files analyzed:** 8 (2 new, 4 modified, 1 new package, 1 modified config)
**Analogs found:** 8 / 8

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|---------------|
| `internal/watch/policy.go` (new) | utility (pure decision function) | request-response (input→string) | `internal/daemon/lock.go` (`isStale`/`isProcessLive`, pure predicate style) + `internal/watch/debounce.go` (`DebounceDuration`, env-parse-with-fallback) | role-match (composite) |
| `internal/watch/policy_test.go` (new) | test | table-driven | `internal/daemon/lock_test.go` style (not read but same package convention) — use `internal/watch/debounce_test.go` for env-injection idiom | role-match |
| `internal/cli/serve.go` (modified — watcher block rework) | controller (CLI RunE) | event-driven (goroutine-deferred startup) | itself (existing `--watch` block, lines 95-129) + `serveServerPaths`'s WR-01 extraction precedent | exact (self, extend) |
| `internal/cli/serve_test.go` (extended) | test (structural seam) | request-response | `TestServeKeepsStartPathDistinctFromConfinementRoot` (lines 17-33) | exact |
| `internal/daemon/daemon.go` (modified — policy gate + `RunWithRetry`) | service (long-lived process driver) | event-driven | itself (`Run`, lines 118-156; `onSyncStart` test seam, lines 67-75) | exact (self, extend) |
| `test/integration/` (new package) | test (subprocess/integration) | request-response over stdio JSON-RPC | `internal/cli/notice_test.go` (`runGitC`, `statusWorktreeMismatchFixture`) + `internal/mcp/markdown_test.go` (in-process MCP client contrast) | role-match |
| `internal/daemon/soak_test.go` (extended) | test (goleak soak) | event-driven | itself (`TestSoak`, lines 36-109) | exact |
| `.github/workflows/ci.yml` (modified) | config (CI) | batch | itself (golden-suite explicit step, lines 68-76) | exact |

## Pattern Assignments

### `internal/watch/policy.go` (new)

**Analog 1 — pure-predicate-with-cached-detection shape:** `internal/daemon/lock.go`

**Pattern to copy** (lines 82-92, `isStale`):
```go
func isStale(info lockInfo) bool {
	if !isProcessLive(info.PID) {
		return true
	}
	if actualStart, ok := processStartTime(info.PID); ok {
		if !startTimesCorroborate(info.StartedAt, actualStart) {
			return true
		}
	}
	return false
}
```
Shape to reuse: a top-level pure function taking a struct of already-resolved
inputs, first-match-wins branching, no I/O inside the branches themselves
(I/O happens in helper functions called once and cached/injected).

**Analog 2 — env-var-with-fallback + package doc precedent:** `internal/watch/debounce.go` lines 1-25

```go
// defaultDebounceMs is the debounce window used when CODEGRAPH_DEBOUNCE_MS
// is unset or invalid (D-04).
const defaultDebounceMs = 2000

func DebounceDuration() time.Duration {
	if v := os.Getenv("CODEGRAPH_DEBOUNCE_MS"); v != "" {
		if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return defaultDebounceMs * time.Millisecond
}
```
Copy: doc-comment style citing the D-number, `os.Getenv` default pattern —
but D-10 requires **strict `== "1"`** comparison (not `Atoi`/truthy), so
adapt the shape, not the exact parse logic.

**Required shape from RESEARCH.md** (already fully specified, use verbatim as
the target structure):
```go
type Probe struct {
    Env        func(string) string // default os.Getenv, injectable for tests
    IsWSL      func() bool         // default DetectWSL (cached), injectable
    NoWatch    bool                // --no-watch flag (D-01/D-02)
    ForceWatch bool                // --watch flag, repurposed force-on (D-03)
}

func WatchDisabledReason(projectRoot string, p Probe) string
```

**Cached-detection precedent (sync.Once-shaped):** no existing sync.Once
pattern in this codebase for this exact case — model on `debounce.go`'s
package-level constant/var pattern but implement `DetectWSL` with a
`sync.Once` guarding a package-level `wslCached bool; wslValue bool` pair,
plus an unexported reset hook for tests (mirrors `onSyncStart`'s "test-only
control seam, no exported setter" convention below).

**Verbatim reason strings (D-12/D-13, MUST match exactly):**
```go
"CODEGRAPH_NO_WATCH=1 is set"
"project is on a WSL2 /mnt/ drive, where recursive file watching is too slow to be reliable"
```
(Note: `fs.watch` → `file watching`, per D-13's documented allowed divergence.)

---

### `internal/cli/serve.go` (modified)

**Analog:** itself — the existing `--watch` block (lines 95-129) is the
direct predecessor; extend it in place following the `serveServerPaths`
extraction precedent (lines 24-46) for the new `serveWatchStart`-shaped seam.

**Imports pattern** (lines 1-17, unchanged, extend as needed):
```go
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
```

**Extraction-for-testability pattern (WR-01 precedent, D-08 reuses this exact
shape)** (lines 24-46):
```go
// serveServerPaths computes BuildServer's two DELIBERATELY DISTINCT
// arguments (CR-01) from start, the caller's actual starting directory:
// ...
// WR-01 (02-REVIEW-2.md): extracted so a test can pin THIS function's
// actual output — the derivation newServeCmd's RunE really performs —
// rather than a hand-built replica living only inside a test file, which
// proves nothing about whether serve.go itself still passes the caller's
// start path through to BuildServer.
func serveServerPaths(start string) (repoPath string, hasIndex bool, err error) {
	...
}
```
Copy this doc-comment shape (cite the WR-01/D-08 precedent explicitly) for
the new `serveWatchStart`-equivalent function that RunE calls.

**Current watcher block to rework** (lines 102-129) — becomes the goroutine
body that must move ALL of: `daemon.New`, the policy check, lock acquisition,
and `watch.Open`'s walk inside the goroutine (D-06):
```go
if watchMode && hasIndex {
	watchCtx, cancelWatch := context.WithCancel(context.Background())
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
```
D-06/D-08 requirement: `daemon.New(repoPath, ...)` itself must move INSIDE
the goroutine (today it runs on-path before the goroutine spawns) — this is
the exact mutation the seam test must catch.

**Error handling pattern to extend for D-14 retry + D-11/D-12 disabled
message** — branch on `errors.Is` exactly as today, add a third branch:
```go
if errors.Is(runErr, watch.ErrWatchDisabled) {
    fmt.Fprintf(cmd.ErrOrStderr(), "[CodeGraph MCP] File watcher disabled — %s. "+
        "The graph will not auto-update; run `codegraph sync` "+
        "(or install the git sync hooks via `codegraph init`) to refresh.\n", reason)
} else if errors.Is(runErr, daemon.ErrLockLive) {
    // D-14: retry path now lives in daemon.RunWithRetry, not here
}
```

**Flag definitions to add/repurpose** (lines 147-149 today):
```go
cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
cmd.Flags().BoolVar(&mcpMode, "mcp", false, "run the stdio MCP server")
cmd.Flags().BoolVar(&watchMode, "watch", false, "...")
```
Add `--no-watch` bool + `cmd.MarkFlagsMutuallyExclusive("no-watch", "watch")`
(cobra v1.10.2, confirmed available — see RESEARCH.md "Don't Hand-Roll").

---

### `internal/cli/serve_test.go` (extended)

**Analog:** `TestServeKeepsStartPathDistinctFromConfinementRoot` (lines 17-33)
— the exact WR-01 seam-test shape to replicate for WATCH-02's structural
guarantee.

```go
// TestServeKeepsStartPathDistinctFromConfinementRoot is WR-01's required
// test (02-REVIEW-2.md): it exercises serveServerPaths, the EXACT function
// newServeCmd's RunE calls to derive BuildServer's repoPath argument — not
// a hand-built replica living only in a test file...
//
// Reproduces the reviewer's mutation directly: if serveServerPaths ever
// collapsed to `return start, hasIndex, nil` unconditionally (the literal
// CR-01 regression...), this test would fail.
func TestServeKeepsStartPathDistinctFromConfinementRoot(t *testing.T) {
	wt, main := statusWorktreeMismatchFixture(t)
	repoPath, hasIndex, err := serveServerPaths(wt)
	...
}
```
**Copy the doc-comment discipline exactly**: name the prior regression class
(CR-01/WR-01 style), state what mutation must turn the test red — the new
test (D-08a) must state "moving daemon.New/policy-check/acquire/watch.Open
back above the goroutine boundary must turn this test red."

**New synchronization-hook precedent to model the seam on** (per
RESEARCH.md's Pattern 4/Open-Question-2 recommendation) — `daemon.go` lines
67-75:
```go
// onSyncStart, when non-nil, is invoked at the very start of flush,
// before touchPending or indexer.Sync run. It is a test-only
// control seam (mirrors onSync) that lets daemon_test.go
// deterministically hold a flush "in flight" — CR-01's exact
// untracked-goroutine window — long enough to prove Run's shutdown
// path genuinely waits for it... Production callers leave it nil.
onSyncStart func()
```
Copy this exact "unexported field, no exported setter, doc comment names
what test hazard it closes" shape for `serveWatchStart`'s own
start-of-real-work hook.

---

### `internal/daemon/daemon.go` (modified — policy gate + `RunWithRetry`)

**Analog:** itself — `Run` (lines 118-156).

**Policy-gate insertion point** (before `acquire`, per RESEARCH.md Pattern 2
— so a policy-disabled watcher never touches the lockfile):
```go
func (d *Daemon) Run(ctx context.Context) error {
	if err := acquire(d.codegraphDir); err != nil {
		return err
	}
	defer func() {
		if err := release(d.codegraphDir); err != nil {
			log.Printf("daemon: releasing lock: %v", err)
		}
	}()
	...
```
becomes (policy check first):
```go
func (d *Daemon) Run(ctx context.Context) error {
	if reason := watch.WatchDisabledReason(d.repoRoot, d.probe); reason != "" {
		return fmt.Errorf("%w: %s", ErrWatchDisabled, reason)
	}
	if err := acquire(d.codegraphDir); err != nil {
		return err
	}
	...
```

**Sentinel-error pattern to copy** — `lock.go` line 30:
```go
var ErrLockLive = errors.New("daemon: lock is held by a live process")
```
Model `ErrWatchDisabled` on this exact shape, in `daemon.go` or `watch`
package (RESEARCH suggests `internal/daemon`, exported so `cli` can
`errors.Is` against it):
```go
var ErrWatchDisabled = errors.New("daemon: watching is disabled by policy")
```

**`RunWithRetry` helper (D-14) — new function, model on `Run`'s doc-comment
density and the `wg`/`ctx.Done()` shutdown discipline already used** (lines
136-154):
```go
var wg sync.WaitGroup
wg.Add(1)
go func() {
	defer wg.Done()
	w.Run(ctx, deb)
}()

<-ctx.Done()
wg.Wait()
```
Copy this ctx.Done()-then-join discipline for `RunWithRetry`'s own
`select { case <-ctx.Done(): ...; case <-time.After(jitter(interval)): }`
loop (RESEARCH.md Pattern 3 gives the exact target shape).

---

### `internal/daemon/soak_test.go` (extended, D-15)

**Analog:** itself — `TestSoak` (lines 36-109) + the package's `TestMain`.

```go
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
```
New convergence test(s) run in this SAME package/TestMain (do not create a
new `TestMain` — RESEARCH.md explicitly flags this as the "don't hand-roll a
new goleak harness" trap). Reuse the `t.Setenv("CODEGRAPH_DEBOUNCE_MS", "20")`
short-interval-for-tests idiom (line 37) for `RunWithRetry`'s own interval —
inject a short retry interval directly as a parameter rather than an env var,
per RESEARCH.md Pattern 3's signature `RunWithRetry(ctx, d, interval, onDeferred)`.

**Cancel-then-join-then-goleak-assert discipline** (lines 83-90, copy
verbatim shape):
```go
// Cancel and join BEFORE any goleak assertion runs (TestMain's
// VerifyTestMain fires after the whole package's tests complete)...
cancel()
wg.Wait()
```

---

### `test/integration/` (new package, TEST-04)

**Analog 1 — real-git fixture + hermetic-skip helper:**
`internal/cli/notice_test.go` lines 20-72 (`runGitC`, `statusWorktreeMismatchFixture`)

```go
func runGitC(t *testing.T, dir string, args ...string) string {
	t.Helper()
	base := []string{
		"-c", "init.defaultBranch=main",
		"-c", "user.name=codegraph-test",
		"-c", "user.email=test@example.invalid",
		"-c", "commit.gpgsign=false",
		"-c", "protocol.file.allow=always",
	}
	full := append(append([]string{}, base...), args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Skipf("git %v failed (git missing or unsupported here): %v: %s", args, err, string(out))
	}
	return string(out)
}
```
**Must be copied verbatim as a FOURTH package-local copy** (the codebase's
established, deliberate pattern per `notice_test.go`'s own doc comment: three
existing copies — `runGitW`, `runGitM`, `runGitC` — Go test helpers aren't
importable across packages). Name it `runGitI` or similar in `test/integration`.

```go
func statusWorktreeMismatchFixture(t *testing.T) (worktreeStart, mainRoot string) {
	t.Helper()
	main := copyFixture(t)
	runGitC(t, main, "init")
	runGitC(t, main, "add", "-A")
	runGitC(t, main, "commit", "-m", "init")
	wt := filepath.Join(main, ".claude", "worktrees", "probe")
	runGitC(t, main, "worktree", "add", "-b", "probe", wt)
	if _, _, err := execCmd("init", main); err != nil { ... }
	...
}
```
Adapt: replace `execCmd("init", main)` (in-process CLI call) with the
subprocess binary invocation (`exec.Command(binPath, "init", "-p", main)`) —
this is precisely the D-19 seam distinction (drive the REAL spawned binary,
not the in-process command tree).

**Bare-glyph helper — copy verbatim as a FOURTH package-local copy** (lines
89-110 of `notice_test.go`):
```go
func containsBareNoticeGlyph(s, glyph string) bool {
	const variationSelector = "️"
	idx := 0
	for {
		i := strings.Index(s[idx:], glyph)
		if i < 0 {
			return false
		}
		pos := idx + i
		rest := s[pos+len(glyph):]
		if !strings.HasPrefix(rest, variationSelector) {
			return true
		}
		idx = pos + len(glyph)
	}
}
```
And `noticeGlyph(t)` (lines 78-87) sourcing the glyph from
`gitmeta.Mismatch.Notice()` rather than a pasted literal.

**Analog 2 — mcp-go stdio client shape (D-19), from RESEARCH.md's verified API surface:**
```go
c, err := client.NewStdioMCPClient(binPath, env, "serve", "--mcp", "-p", repoPath)
defer c.Close()
_, err = c.Initialize(ctx, mcp.InitializeRequest{...})
result, err := c.CallTool(ctx, mcp.CallToolRequest{
    Params: mcp.CallToolParams{Name: "codegraph_explore", Arguments: map[string]any{"query": "Alpha"}},
})
```
Remember `CODEGRAPH_MCP_TOOLS` allowlist env for any companion-tool case
(D-21) — `codegraph_explore` needs no allowlist entry.

**TestMain binary-build pattern (D-18)** — no existing exact analog in this
repo (`internal/cli`'s tests use in-process `execCmd`, not a built binary);
model on Go's standard `TestMain` + `go build -o <path> .` idiom, storing the
binary path in a package-level var for all tests in the package to share
(one build per `go test` run, per D-18).

---

## Shared Patterns

### Sentinel errors + `errors.Is` branching
**Source:** `internal/daemon/lock.go` line 30 (`ErrLockLive`), extended in
this phase with `ErrWatchDisabled` in `internal/daemon/daemon.go`.
**Apply to:** `internal/daemon/daemon.go` (`Run`), `internal/cli/serve.go`
(goroutine branch), `test/integration/` (asserting disabled-message stderr).
```go
var ErrWatchDisabled = errors.New("daemon: watching is disabled by policy")
// ...
if errors.Is(runErr, watch.ErrWatchDisabled) { ... }
```

### Test-only control seam (unexported field, no exported setter)
**Source:** `internal/daemon/daemon.go` lines 67-75 (`onSyncStart`).
**Apply to:** the new `serveWatchStart` seam in `internal/cli/serve.go`
(WATCH-02's D-08(a) synchronization hook), so `serve_test.go` can
deterministically observe "goroutine's real work has started" without a
sleep/timeout race.

### Env-var strict-equality parsing
**Source:** D-10's requirement, contrasted with `internal/watch/debounce.go`'s
looser `Atoi`-with-fallback style (lines 19-24) — do NOT reuse that
truthiness-tolerant shape for `policy.go`; TS checks `=== '1'` exactly, so
`internal/watch/policy.go` must use `p.Env("CODEGRAPH_NO_WATCH") == "1"`,
never `strconv.ParseBool` or non-empty-string truthiness.

### Cancel-then-join-then-assert goroutine teardown
**Source:** `internal/daemon/daemon.go` `Run` (lines 143-154, `wg.Wait()` +
`deb.Wait()`) and `internal/daemon/soak_test.go` `TestSoak` (lines 83-90).
**Apply to:** `RunWithRetry`'s own shutdown path and its soak test —
`ctx.Done()` must be honored inside the retry `select`, and the calling
test must `cancel(); wg.Wait()` before any goleak assertion fires.

### Real-git fixture + `t.Skip` (never `t.Fatal`) on git absence
**Source:** `internal/cli/notice_test.go` (`runGitC`), matched by
`internal/query/engine_worktree_test.go`'s `runGitW` and
`internal/mcp/markdown_test.go`'s `runGitM`.
**Apply to:** `test/integration/`'s own git-driving helper (a deliberate
fourth package-local copy, not a shared import — Go test helpers aren't
importable across packages, an established convention in this codebase).

### Explicit named CI step for anything `go list ./...`/testdata can skip
**Source:** `.github/workflows/ci.yml` lines 68-76 (the GOLDEN-01 fix,
`go test ./testdata/golden/...`).
```yaml
- name: Test golden parity suite (testdata/golden, NOT covered by go list ./...)
  run: go test ./testdata/golden/...
```
**Apply to:** add a sibling step for `test/integration/` (D-17 belt-and-braces,
even though `go list ./...` DOES already reach it since it's not under
`testdata/` — the explicit step guards against a future refactor of the
filtered `go list` line silently dropping it):
```yaml
- name: Test subprocess integration harness (test/integration)
  run: go test ./test/integration/...
```

## No Analog Found

None — every file in scope has at least a role-match analog already read and
excerpted above. The one genuinely novel piece of code (WSL `/proc/version`
detection + `sync.Once`-cached probe) has no existing in-repo precedent per
RESEARCH.md's own "Don't Hand-Roll" table ("no existing WSL-detection code
anywhere in this repo") — it is new code by necessity, kept deliberately
small (~6 lines) per RESEARCH.md's guidance, ported directly from the cited
TS source rather than invented.

## Metadata

**Analog search scope:** `internal/cli/`, `internal/daemon/`, `internal/watch/`,
`internal/mcp/` (contrast only), `.github/workflows/`
**Files read this pass:** `internal/cli/serve.go`, `internal/cli/serve_test.go`,
`internal/cli/notice_test.go`, `internal/daemon/daemon.go`,
`internal/daemon/lock.go`, `internal/daemon/soak_test.go`,
`internal/watch/debounce.go` (partial), `.github/workflows/ci.yml`
**Pattern extraction date:** 2026-07-16
