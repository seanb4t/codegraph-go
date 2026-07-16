# Phase 3: Watcher-on-MCP Default - Research

**Researched:** 2026-07-16
**Domain:** Go CLI/MCP-server watcher lifecycle, TS-parity policy porting, subprocess integration testing
**Confidence:** HIGH — every load-bearing claim below is `[VERIFIED: <path read>]` against either the installed TS 1.3.1 dist or this repo's own Go source, per the phase's explicit "verify by reading source, never infer" mandate. No web search or Context7 lookups were needed or used — this phase has zero new external dependencies; all research is white-box source reading.

## Summary

This phase ports TS 1.3.1's `watch-policy.js` verbatim into a new `internal/watch/policy.go`, flips `serve --mcp`'s watcher to default-on behind that policy, moves ALL watcher startup off the MCP handshake path into a background goroutine, replaces the daemon's current defer-once behavior with a defer-and-retry loop for concurrent-session convergence, and adds a subprocess integration harness (`test/integration/`) that spawns the real release binary and drives it over real argv/stdio.

The two riskiest ambiguities the CONTEXT.md left as "Claude's Discretion" both have a single clean answer once the source is read carefully, and this research pins both:

1. **D-16 (the explicit "must confirm" blocker):** `internal/daemon/lock.go`'s `acquire()` **self-heals a stale lock on every call** — it is not a one-shot check performed once at startup. Any retry loop that simply calls `acquire()` (via `Daemon.Run`) again after `ErrLockLive` will, on its next attempt, independently re-read the lockfile and re-run `isStale()`. A crashed holder is detected and cleared the very next retry; a live holder is never touched. **No wedge risk exists** — the defer-and-retry design (D-14) requires no new staleness machinery, just a loop around the existing `Run`.
2. **The flag/env reason-string question (not explicitly resolved by any D-decision):** TS's `--no-watch` flag routes through `process.env.CODEGRAPH_NO_WATCH = '1'` (bin/codegraph.js ~1573-1576) *before* `watchDisabledReason` ever runs, so TS's own disabled-stderr message is **byte-identical** whether the env var or the flag triggered it — TS cannot tell the difference internally. Our port (D-05: never mutate env, pass explicit values instead) must therefore treat `--no-watch` flag and `CODEGRAPH_NO_WATCH=1` env as two inputs to the SAME precedence check, both producing the SAME reason string `"CODEGRAPH_NO_WATCH=1 is set"` — this is not an assumption, it is what reading bin/codegraph.js proves TS actually does.

**Primary recommendation:** Port `watch-policy.js` as a single `internal/watch/policy.go` function taking explicit flag/env/WSL-probe inputs (never touching `os.Setenv`); enforce it as the very first action inside `Daemon.Run` (before `acquire()`, so a policy-disabled watcher never even touches the lockfile); return a distinguishable sentinel error so `serve --mcp`'s goroutine can print the verbatim TS stderr message while `codegraph daemon` gets it "for free" through the same shared `Run` — zero additional wiring in `internal/cli/daemon.go`. Implement the D-14 retry loop as a **reusable, goleak-testable helper inside `internal/daemon`** (not ad hoc inside `serve.go`) so it inherits `internal/daemon/soak_test.go`'s existing `TestMain(goleak.VerifyTestMain)` coverage instead of requiring a new goleak harness in `internal/cli`.

## Architectural Responsibility Map

This project is a CLI/MCP-server Go binary, not a web app — the standard Browser/SSR/API/CDN/DB tiers don't apply. Substituting this project's own established layer boundaries (per `.claude/CLAUDE.md`'s architecture and the CONTEXT.md's `<code_context>` integration points):

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Flag surface & precedence (`--no-watch`/`--watch`) | `internal/cli` (serve.go) | — | Cobra flag definitions + mutual-exclusivity are a command-layer concern; `[VERIFIED: cobra v1.10.2 ships MarkFlagsMutuallyExclusive in flag_groups.go]` |
| Watch-policy decision (WATCH-03) | `internal/watch` (new policy.go) | — | D-09: lives with the watcher so both `serve --mcp` and standalone `daemon` share one source of truth; mirrors TS's `sync/watch-policy.js` centralization |
| Policy enforcement + lock/retry (WATCH-02/04) | `internal/daemon` (Run) | `internal/cli` (goroutine wiring) | D-11: `daemon.Run` is the shared seam BOTH callers drive through; `internal/cli` only wires the goroutine and prints the human message |
| Off-handshake-path watcher startup (WATCH-02) | `internal/cli` (serve.go RunE) | `internal/daemon`/`internal/watch` | The *structural* guarantee (no watcher code before `server.ServeStdio`) is provable only by reading serve.go's RunE body; the *work* it defers lives one layer down |
| Subprocess integration harness (TEST-04) | `test/integration/` (new package) | `internal/mcp` (client transport reused), `cmd/codegraph` (binary under test) | D-17: must be a normal Go package (never `testdata/`) so `go test ./...` and the new explicit CI step both reach it |

## Standard Stack

No new dependencies this phase. Every library the plan needs is already in `go.mod`:

| Library | Version (verified) | Purpose | Why no new dep needed |
|---------|---------|---------|--------------|
| `github.com/mark3labs/mcp-go` | v0.56.0 `[VERIFIED: go.mod line 10]` | TEST-04's stdio JSON-RPC client | `client.NewStdioMCPClient(command, env, args...)` exists in this exact pinned version — `[VERIFIED: $GOMODCACHE/github.com/mark3labs/mcp-go@v0.56.0/client/stdio.go]` — launches a subprocess, wires stdin/stdout pipes, auto-starts the transport |
| `github.com/spf13/cobra` | v1.10.2 `[VERIFIED: go.mod line 12]` | `--no-watch`/`--watch` mutual exclusivity | `Command.MarkFlagsMutuallyExclusive` exists in this version — `[VERIFIED: $GOMODCACHE/github.com/spf13/cobra@v1.10.2/flag_groups.go]` — gives D-04's "hard flag error" for free, no hand-rolled validation |
| `github.com/fsnotify/fsnotify` | pinned (unchanged) | Underlying watcher | Untouched by this phase — WATCH-01 changes *when* `watch.Open` runs, not *how* |
| `go.uber.org/goleak` | v1.3.0 `[VERIFIED: go.mod line 27]` | D-15 goleak soak extension | Already wired into `internal/daemon`'s `TestMain` (`internal/daemon/soak_test.go`) and `internal/watch`'s |

### Alternatives Considered

None — the phase's own CONTEXT.md (`<code_context>` "Reusable Assets") already states "mcp-go (existing dep) ships a client package: the harness's stdio JSON-RPC side needs no new dependency. Zero new deps expected this phase," and source-reading confirms it.

**Installation:** none required.

## Package Legitimacy Audit

Not applicable — zero new external packages this phase. No `npm view`/`pip index`/`cargo search` gate needed.

## Architecture Patterns

### System Architecture Diagram

```
                     serve --mcp (process start)
                              │
                              ▼
                 resolveStartPath / serveServerPaths
                 (repoPath, hasIndex — CR-01 seam, unchanged)
                              │
                              ▼
        hasIndex? ──no──► skip reconcile, skip watcher (MCP-03, unchanged)
              │yes
              ▼
     indexer.Sync (D-07: STAYS synchronous, pre-handshake)
              │
              ▼
   ┌──────────────────────────────────────────────────────┐
   │  spawn ONE goroutine (D-06: ALL of the below is here) │
   │                                                        │
   │   noWatch flag/env? ──yes──► never enter this block   │
   │         │no                                            │
   │         ▼                                              │
   │   daemon.New(repoPath, opts)   [cheap: Abs + Stat]     │
   │         │                                              │
   │         ▼                                              │
   │   loop: d.Run(watchCtx)  ◄────────────────┐            │
   │         │                                  │            │
   │    ┌────┴─────┐                            │            │
   │    ▼          ▼                            │            │
   │  policy    acquire()                       │            │
   │  check     (may self-heal a stale lock)    │            │
   │  FIRST     │                                │            │
   │    │       ├─ok──► watch.Open (recursive   │            │
   │    │       │        fsnotify walk) → blocks │            │
   │    │       │        on ctx.Done()           │            │
   │    │       │                                │            │
   │    │       └─ErrLockLive──► log ONCE,       │            │
   │    │                        sleep ~30s±jit ─┘  (retry,   │
   │    │                                            D-14)    │
   │    └─disabled──► return ErrWatchDisabled          │      │
   │              (no lock ever touched; terminal,      │      │
   │               no retry — policy doesn't change      │     │
   │               mid-session)                           │    │
   └──────────────────────────────────────────────────────┘
              │ (goroutine spawn returns IMMEDIATELY — no blocking above
              │  this line ever executes before the line below, D-06/08)
              ▼
   mcp.BuildServer(hasIndex, allowlist, repoPath, start)  [CR-01: distinct args]
              │
              ▼
   server.ServeStdio(s)   ◄── WATCH-02's guarantee: zero watcher code
                               has executed above this line by construction
```

### Recommended Project Structure

```
internal/
├── cli/
│   ├── serve.go            # flags (--no-watch new, --watch repurposed force-on),
│   │                        # extracted watch-start seam (D-08), goroutine wiring only
│   ├── serve_test.go        # existing WR-01 tests + new WATCH-02 seam test
│   └── daemon.go            # UNCHANGED — inherits policy gate for free via Run()
├── watch/
│   ├── policy.go            # NEW: WatchDisabledReason + DetectWSL (D-09/D-10)
│   ├── policy_test.go       # NEW: precedence table + WSL/env-injection tests
│   ├── watcher.go            # unchanged (Open/Run/Close)
│   └── soak_test.go          # existing goleak TestMain
└── daemon/
    ├── daemon.go             # Run gains policy-check-first + ErrWatchDisabled;
    │                          # RunWithRetry-shaped helper for D-14 (recommended
    │                          # location — see "Don't Hand-Roll" below)
    ├── lock.go               # UNCHANGED — acquire() already self-heals (D-16 answer)
    └── soak_test.go           # existing goleak TestMain, extended for two-session
                                 # convergence (D-15)
test/
└── integration/              # NEW (D-17): normal Go package, TestMain builds the
    ├── main_test.go            # release binary once (D-18)
    ├── worktree_notice_test.go # D-20 CR-01 anchor case
    └── watch_default_test.go   # D-21 WATCH default-on + NO_WATCH cases
```

### Pattern 1: Explicit-input policy function (never mutate process env)

**What:** Port `watchDisabledReason` as a pure function taking explicit flag/env/WSL-probe inputs, matching D-05's "never mutate the process env" rule while preserving TS's exact precedence and message behavior.

**When to use:** `internal/watch/policy.go`, called from `Daemon.Run`.

**TS source (verbatim precedence, `[VERIFIED: sync/watch-policy.js]`):**
```javascript
// Source: /opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/
//   @colbymchenry/codegraph-darwin-arm64/lib/dist/sync/watch-policy.js lines 105-118
function watchDisabledReason(projectRoot, probe = {}) {
    const env = probe.env ?? process.env;
    if (env.CODEGRAPH_NO_WATCH === '1') {
        return 'CODEGRAPH_NO_WATCH=1 is set';
    }
    if (env.CODEGRAPH_FORCE_WATCH === '1') {
        return null;
    }
    const isWsl = probe.isWsl ?? detectWsl();
    if (isWsl && isWindowsDriveMount(projectRoot)) {
        return 'project is on a WSL2 /mnt/ drive, where recursive fs.watch is too slow to be reliable';
    }
    return null;
}
```

**WSL detection (verbatim, `[VERIFIED: sync/watch-policy.js lines 65-93]`):**
```javascript
function detectWsl() {
    if (wslChecked) return wslValue;       // cached (sync.Once-shaped in Go)
    wslChecked = true;
    if (process.platform !== 'linux') { wslValue = false; return wslValue; }
    if (process.env.WSL_DISTRO_NAME || process.env.WSL_INTEROP) { wslValue = true; return wslValue; }
    try {
        const version = fs.readFileSync('/proc/version', 'utf8').toLowerCase();
        wslValue = version.includes('microsoft') || version.includes('wsl');
    } catch { wslValue = false; }
    return wslValue;
}
function isWindowsDriveMount(projectRoot) {
    // Deliberately single-letter only — excludes /mnt/wsl/...
    return /^\/mnt\/[a-z](\/|$)/i.test(normalizePath(projectRoot));
}
// normalizePath (utils.js line 242-244): filePath.replace(/\\/g, '/')
// — this is EXACTLY what Go's filepath.ToSlash already does elsewhere in
// this codebase (`[VERIFIED: internal/indexer/discover.go, internal/migrate/reader.go,
// internal/query/files.go all use filepath.ToSlash for this exact purpose]`).
```

**Recommended Go port shape** (structure only — exact naming is plan-time detail, but the FIELD SET below is load-bearing since D-05 forbids env mutation):
```go
// internal/watch/policy.go
type Probe struct {
    Env        func(string) string // default os.Getenv, injectable for tests
    IsWSL      func() bool         // default DetectWSL (cached), injectable
    NoWatch    bool                // --no-watch flag (D-01/D-02)
    ForceWatch bool                // --watch flag, repurposed force-on (D-03)
}

// WatchDisabledReason returns "" when watching should run, or a short
// human-readable reason (verbatim TS strings, D-12/D-13) when it should not.
// Precedence, first match wins (D-04):
//  1. NoWatch flag OR Env("CODEGRAPH_NO_WATCH")=="1" -> off
//  2. ForceWatch flag OR Env("CODEGRAPH_FORCE_WATCH")=="1" -> on (beats auto-detect)
//  3. WSL2 + /mnt/[a-z] drive -> off
//  4. default -> on
func WatchDisabledReason(projectRoot string, p Probe) string
```

**Critical wording finding (`[VERIFIED: bin/codegraph.js lines 1570-1576]`):**
```javascript
.option('--no-watch', 'Disable the file watcher (no auto-sync; useful on slow filesystems like WSL2 /mnt drives)')
.action(async (options) => {
    // Commander sets watch=false when --no-watch is passed. Route it through
    // the same env-var chokepoint the watcher and MCP server already honor.
    if (options.watch === false) {
        process.env.CODEGRAPH_NO_WATCH = '1';
    }
```
TS's `--no-watch` flag is ALWAYS indistinguishable from `CODEGRAPH_NO_WATCH=1` by the time `watchDisabledReason` runs — it mutates env first. This means TS's own disabled-stderr message reads `"...disabled — CODEGRAPH_NO_WATCH=1 is set..."` **even when a human typed `--no-watch` on the command line, never set an env var**. For byte-for-byte behavioral parity (not just message-text parity), our port's tier-1 check must be `p.NoWatch || p.Env("CODEGRAPH_NO_WATCH") == "1"` returning the SAME string `"CODEGRAPH_NO_WATCH=1 is set"` regardless of which triggered it — this is a proven fact from reading TS's source, not a divergence to document.

### Pattern 2: Enforcement inside `Daemon.Run`, sentinel error distinguishes disabled vs live-lock

**What:** D-11 requires ONE enforcement point shared by `serve --mcp`'s in-process watcher and the standalone `codegraph daemon` command. Reading both callers (`[VERIFIED: internal/cli/serve.go lines 95-129]`, `[VERIFIED: internal/cli/daemon.go lines 39-51]`) confirms both already funnel through `daemon.New` + `d.Run(ctx)` — the ONLY shared code path. `codegraph daemon`'s RunE (`internal/cli/daemon.go`) needs **zero code changes** to inherit the policy gate once `Run` itself enforces it.

**TS's own enforcement point for comparison (`[VERIFIED: sync/watcher.js lines 290-305]`):**
```javascript
start() {
    if (this.recursiveWatcher || this.dirWatchers.size > 0 || this.inert)
        return true;
    // ...
    const disabledReason = watchDisabledReason(this.projectRoot);
    if (disabledReason) {
        logDebug('File watcher disabled', { reason: disabledReason, projectRoot: this.projectRoot });
        return false;
    }
```
TS enforces INSIDE `watcher.start()` — the direct analog of our `watch.Open`. D-11's choice of `daemon.Run` (one level up, wrapping `watch.Open`) is the correct Go equivalent because `daemon.Run` is our actual shared seam (TS's engine.js separately re-checks the policy for its own stderr message at `startWatching()`, ~line 254 — a belt-and-suspenders duplication our single-enforcement-point design deliberately avoids per D-11's text).

**Recommended shape:**
```go
// internal/daemon (or internal/watch, exported so cli can errors.Is against it)
var ErrWatchDisabled = errors.New("daemon: watching is disabled by policy")

func (d *Daemon) Run(ctx context.Context) error {
    if reason := watch.WatchDisabledReason(d.repoRoot, d.probe); reason != "" {
        return fmt.Errorf("%w: %s", ErrWatchDisabled, reason)
    }
    if err := acquire(d.codegraphDir); err != nil {
        return err // unchanged — ErrLockLive path
    }
    // ... unchanged watch.Open / debounce loop
}
```
`serve --mcp`'s goroutine distinguishes the two outcomes:
```go
if errors.Is(runErr, watch.ErrWatchDisabled) {
    // D-12: verbatim TS stderr message, D-13: verbatim reason strings
    fmt.Fprintf(stderr, "[CodeGraph MCP] File watcher disabled — %s. "+
        "The graph will not auto-update; run `codegraph sync` "+
        "(or install the git sync hooks via `codegraph init`) to refresh.\n", reason)
    return // terminal — do not retry; policy doesn't change mid-session
} else if errors.Is(runErr, daemon.ErrLockLive) {
    // D-14: log once, then retry on a jittered cadence
}
```

**`codegraph daemon`'s RunE inheritance — flag for the planner:** with this design, `codegraph daemon` run on a disabled-policy project (e.g. WSL2 `/mnt/c`) will have `Run` return a non-nil `ErrWatchDisabled`-wrapped error, which cobra prints and exits non-zero. This is arguably CORRECT (an idle standalone daemon serves no purpose — better to fail fast with an actionable message than hang) but is a genuine, not-explicitly-resolved-by-CONTEXT.md design point since DMON-01..04 lifecycle UX is explicitly out of this phase's scope. Recommend documenting this behavior explicitly in the plan rather than leaving it as an accidental side effect.

### Pattern 3: Retry loop lives in `internal/daemon`, not `internal/cli` (goleak reuse)

**What:** CONTEXT.md's "Claude's Discretion" leaves "goroutine/channel structure inside serve.go's watcher block" open. Reading D-15's requirement ("extend the EXISTING goleak soak pattern... internal/daemon/soak_test.go, internal/watch/soak_test.go") against the fact that `internal/cli` currently has **no** `TestMain`/goleak wiring at all (`[VERIFIED: no TestMain/goleak reference found in internal/cli via search]`) strongly favors extracting the retry loop as a reusable, directly-testable function living in `internal/daemon` (which already has `TestMain(goleak.VerifyTestMain)` per `internal/daemon/soak_test.go` lines 19-21) rather than writing raw goroutine/ticker code inline inside `serve.go`.

**D-16's answer makes this loop trivial** — `acquire()` self-heals on every independent call (see Common Pitfalls below), so the loop is just:
```go
// internal/daemon — new, goleak-testable via the existing TestMain
func RunWithRetry(ctx context.Context, d *Daemon, interval time.Duration, onDeferred func()) error {
    for {
        err := d.Run(ctx)
        if err == nil || !errors.Is(err, ErrLockLive) {
            return err // clean shutdown, ErrWatchDisabled, or a genuine error
        }
        onDeferred() // caller logs the "deferring to it" line ONCE per D-14 —
                      // caller can no-op this after the first call
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(jitter(interval)):
        }
    }
}
```
This single function is directly soak-testable: spin up two `Daemon` instances sharing one `.codegraph/` lockfile, cancel the winner mid-run, assert the loser's `RunWithRetry` converges and becomes the sole writer, all inside `internal/daemon`'s existing goleak-gated `TestMain` — no new goleak harness needed anywhere. `serve.go`'s goroutine becomes a thin wiring call: `go func() { defer close(done); runErr <- daemon.RunWithRetry(watchCtx, d, retryInterval, logOnce) }()`.

### Pattern 4: WATCH-02's mutation-testable seam (D-08)

**What:** D-08 explicitly requires a structural test proving serve.go's RunE genuinely defers ALL watcher work to a goroutine — reusing the exact WR-01 precedent (`serveServerPaths` + `serve_test.go`'s `TestServeKeepsStartPathDistinctFromConfinementRoot`, `[VERIFIED: internal/cli/serve_test.go lines 17-33]`, which tests the REAL function RunE calls, not a hand-built replica — the doc comment explicitly names this as the fix for a prior reviewer-caught regression).

**Recommended pattern** (same shape as WR-01): extract a `serveWatchStart`-named function that:
1. Takes `(ctx, repoPath, hasIndex, noWatch, forceWatch, opts)` and returns `(cancel func(), done <-chan struct{})` — or a no-op pair when `!hasIndex || noWatch`.
2. Spawns exactly one goroutine internally and **returns before that goroutine's body executes any of `daemon.New`/policy-check/`acquire`/`watch.Open`** — provable via a test-only synchronization hook (mirrors `Daemon.onSyncStart`'s existing precedent, `[VERIFIED: internal/daemon/daemon.go lines 67-75]`) that signals the moment real work starts, letting a test assert `serveWatchStart` returned strictly BEFORE that signal fires.
3. RunE calls `serveWatchStart(...)` then IMMEDIATELY calls `server.ServeStdio(s)` on the next line — the structural guarantee a code reviewer (and TEST-04's handshake-latency case) can verify by reading top-to-bottom.

A mutation of moving any of the deferred work (e.g. `daemon.New`) back above `serveWatchStart`'s goroutine boundary must turn this test red — satisfying D-08(a)'s explicit requirement.

### Anti-Patterns to Avoid

- **Duplicating the policy check in both `serve.go` and `daemon.Run`:** TS itself does this (once in `engine.js`'s `startWatching()` for its own message, once in `watcher.js`'s `start()`) — D-11 explicitly chooses a SINGLE enforcement point for our port. Don't replicate TS's duplication; it would create two sources of truth that can drift.
- **Mutating `os.Setenv` to route `--no-watch`/`--watch` through the policy function** the way TS does (`process.env.CODEGRAPH_NO_WATCH = '1'`): D-05 explicitly forbids this. The in-process watcher never spawns a subprocess that needs to inherit the env var — pass explicit booleans instead.
- **A retry loop with no jitter:** multiple concurrent `serve --mcp` sessions retrying on the exact same fixed interval will thunder-herd against the lockfile every cycle. Even a small jitter avoids synchronized retries (Claude's Discretion covers the exact shape, but SOME jitter is a correctness-adjacent choice, not purely cosmetic).
- **Testing the retry loop only via the TEST-04 subprocess harness:** subprocess tests are slow and non-deterministic for timing-sensitive convergence (~30s retry cadence). The soak-test-in-`internal/daemon` approach (Pattern 3) should use an injectable/short retry interval (mirroring `CODEGRAPH_DEBOUNCE_MS`'s existing test-override precedent, `[VERIFIED: internal/watch/debounce.go lines 15-19, used by soak_test.go line 29]`) so convergence tests run in milliseconds, not 30 seconds.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| `--no-watch --watch` contradiction detection | A manual `if noWatch && forceWatch { return err }` check in RunE | `cmd.MarkFlagsMutuallyExclusive("no-watch", "watch")` | Already available in the pinned cobra v1.10.2 (`[VERIFIED]`); cobra validates BEFORE RunE runs, producing a consistent flag-error UX matching every other cobra mutual-exclusivity error in the ecosystem |
| Stale-lock detection for the retry loop | A NEW staleness-check-on-retry mechanism | The EXISTING `acquire()` — it already re-checks staleness (PID liveness + start-time corroboration) on every independent call | D-16's answer: `acquire()` is not a one-shot startup check; it is fully self-contained and self-healing per invocation. Building a parallel staleness mechanism would duplicate `isStale`'s PID-reuse corroboration logic (`internal/daemon/lock.go` lines 68-113) for zero benefit |
| Subprocess stdio JSON-RPC driving for TEST-04 | A hand-rolled `exec.Command` + manual JSON-RPC framing over stdin/stdout | `mcp-go`'s `client.NewStdioMCPClient(command, env, args...)` | Already a pinned dependency (`[VERIFIED]`); handles process spawn, pipe wiring, `Initialize`/`ListTools`/`CallTool` request/response correlation — reimplementing this is the exact kind of "deceptively complex" protocol-framing work the project's Don't-Hand-Roll philosophy exists to avoid |
| WSL detection | Any new library or syscall-based approach | `runtime.GOOS == "linux"` + env var check + `/proc/version` read, exactly as TS does it | No existing Go library does this better than 6 lines of stdlib; TS's own implementation is already minimal and directly portable — `[VERIFIED: no existing WSL-detection code anywhere in this repo]`, so this is genuinely new code, but it should stay this small |

**Key insight:** This phase's most valuable "don't hand-roll" finding is D-16 itself — the temptation, faced with "must confirm stale-lock semantics before finalizing the retry design," is to assume new staleness-tracking state is needed for the retry loop. Reading `lock.go` proves the opposite: the retry loop can be a trivial `for { err := d.Run(ctx); if !ErrLockLive { return err }; sleep; }` wrapper, because staleness detection is already a property of `acquire()` itself, not of when it's called.

## Common Pitfalls

### Pitfall 1: Assuming the retry loop needs a "is the crashed daemon dead yet" poll separate from `acquire()`

**What goes wrong:** Building a parallel liveness-poll mechanism (e.g., a ticker that calls `isProcessLive(pid)` directly) that duplicates or races against `acquire()`'s own staleness check.
**Why it happens:** D-16's "must confirm before finalizing" framing reads as if the answer might require new machinery.
**How to avoid:** Re-read `[VERIFIED: internal/daemon/lock.go lines 136-186]` — `acquire()`'s stale-lock recovery path (read lock → `isStale` → remove if stale → recreate) runs FRESH on every call, including calls made from inside a retry loop. The loop needs nothing more than "call `Run` again."
**Warning signs:** Any new exported function in `internal/daemon` whose name contains "poll", "watch for exit", or "wait for pid" is a signal the design has drifted from the source-verified answer.

### Pitfall 2: `--no-watch` flag silently skipping the disabled-message entirely

**What goes wrong:** Treating `--no-watch` as "the CLI layer's own opt-out, never touches `WatchDisabledReason`" — which seems clean but silently diverges from TS's actual (message-producing) behavior when a human explicitly disables the watcher, since a future dashboard/log-scraper expecting the standard `"File watcher disabled — ..."` line on ANY disablement (env, flag, or WSL auto-detect) would see it missing for the flag case only.
**Why it happens:** It's tempting to treat an explicit user opt-out as "no message needed, they asked for this" — reasonable UX instinct, wrong TS-parity answer.
**How to avoid:** `[VERIFIED: bin/codegraph.js lines 1573-1576]` proves TS DOES print the message for the flag case (because the flag is internally indistinguishable from the env var by the time the check runs). Route `--no-watch` through the SAME `Probe.NoWatch` input to `WatchDisabledReason`, not a separate early-return in `serve.go`.
**Warning signs:** A code path in `serve.go` that checks `if noWatch { return nil /* skip everything */ }` before ever calling the watch-start goroutine at all — this loses the stderr message.

### Pitfall 3: Reading `.d.ts` stub files instead of the platform sub-package's real `.js`

**What goes wrong:** `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/dist/` contains ONLY `.d.ts` type stubs (no implementation) — reading there produces type signatures with zero behavioral detail, silently producing an incomplete or wrong port.
**Why it happens:** The top-level `dist/` looks like the obvious place to look; the real implementation is one level down inside the platform-specific optional dependency (`@colbymchenry/codegraph-darwin-arm64`).
**How to avoid:** CONTEXT.md already flags this exact trap in `<canonical_refs>` — this research confirms it by having successfully read all four cited files at their given absolute paths under `.../codegraph-darwin-arm64/lib/dist/...` and finding real, executable JS with full logic (not `.d.ts` stubs) at every one.
**Warning signs:** A cited TS file that's suspiciously short, contains only type declarations, or has a `.d.ts` extension.

### Pitfall 4: Golden/CI silent-skip recurrence (GOLDEN-01-shaped risk for `test/integration/`)

**What goes wrong:** Placing the new harness under any directory named `testdata/` (or otherwise excluded from `go list ./...`'s expansion) would silently repeat GOLDEN-01 — a `go test ./...` CI step reporting green while never running these tests at all.
**Why it happens:** `testdata/` "feels" like the natural home for test fixtures.
**How to avoid:** D-17 already mandates `test/integration/` (NOT `testdata/`) as a normal Go package, PLUS an explicit named CI step `go test ./test/integration/...` alongside the existing explicit `go test ./testdata/golden/...` step. `[VERIFIED: .github/workflows/ci.yml]` already contains this exact belt-and-braces pattern for the golden suite (with an inline comment citing GOLDEN-01) — the new step should be added the same way, not folded into the "Test (excluding internal/daemon)" step that does `go list ./...` (which DOES reach `test/integration/` fine since it's not a `testdata` dir, but an explicit step is still required per D-17's "belt and braces" mandate so a future refactor of that filtered `go list` line cannot silently drop it).
**Warning signs:** No new named step appears in ci.yml's `test` job; the harness only runs implicitly via the existing filtered `go list ./... | grep -v daemon` line.

### Pitfall 5: Bare-glyph vs. emoji-presentation U+26A0 false positive/negative in TEST-04 assertions

**What goes wrong:** A naive `strings.Contains(payload, "⚠")` in the new subprocess harness will ALSO match the pre-existing EXPL-04 "⚠️ no covering tests found" warning (U+26A0 + U+FE0F variation selector), since the bare glyph's UTF-8 bytes are a byte-prefix of the emoji-presentation variant's bytes.
**Why it happens:** The two warnings share their first codepoint; only a variation-selector-aware check distinguishes them.
**How to avoid:** Reuse the EXISTING helper `[VERIFIED: internal/cli/notice_test.go lines 89-110, containsBareNoticeGlyph]` rather than reimplementing — TEST-04's harness lives in a different package (`test/integration`) so this helper must be copied (Go test helpers aren't importable across packages, per the same file's own doc comment about `runGitC` being a "third, package-local copy") — this is the established, deliberate pattern in this codebase (three existing copies: `internal/query/engine_worktree_test.go`'s `runGitW`, `internal/mcp/markdown_test.go`'s `runGitM`, `internal/cli/notice_test.go`'s `runGitC`), not an oversight to fix.
**Warning signs:** A new assertion helper in `test/integration` that doesn't mention U+FE0F/variation-selector handling.

## Code Examples

### Real git worktree fixture pattern (D-15/D-20 precedent to reuse)

```go
// Source: internal/cli/notice_test.go lines 20-72 (statusWorktreeMismatchFixture)
// — the exact pattern TEST-04's D-20 anchor case must mirror, adapted to
// drive the SUBPROCESS binary (via execCmd-shaped exec.Command calls)
// instead of the in-process CLI command tree.
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
// main := copyFixture(t); runGitC(t, main, "init"); runGitC(t, main, "add", "-A");
// runGitC(t, main, "commit", "-m", "init")
// wt := filepath.Join(main, ".claude", "worktrees", "probe")
// runGitC(t, main, "worktree", "add", "-b", "probe", wt)
```

### mcp-go stdio client, the shape TEST-04 needs (D-19)

```go
// Source: $GOMODCACHE/github.com/mark3labs/mcp-go@v0.56.0/client/stdio.go
// and client.go (verified method list: Initialize, ListTools, CallTool, Close)
c, err := client.NewStdioMCPClient(binPath, env, "serve", "--mcp", "-p", repoPath)
if err != nil { /* ... */ }
defer c.Close()

_, err = c.Initialize(ctx, mcp.InitializeRequest{ /* protocol version, clientInfo */ })
// then:
result, err := c.CallTool(ctx, mcp.CallToolRequest{
    Params: mcp.CallToolParams{Name: "codegraph_explore", Arguments: map[string]any{"query": "Alpha"}},
})
// result.Content[0].(mcp.TextContent).Text carries the payload to assert
// containsBareNoticeGlyph against.
```
`env` here is where `CODEGRAPH_MCP_TOOLS` must be set for any companion-tool case (D-21 note); `codegraph_explore` needs no allowlist entry — `[VERIFIED: internal/mcp/tools.go, internal/mcp/server.go ParseAllowlist/WarnUnknownToolsTo, internal/mcp/server_test.go]` confirm the allowlist only gates the 7 companion tools, not the always-visible `codegraph_explore`.

### Existing goleak soak precedent to extend (D-15)

```go
// Source: internal/daemon/soak_test.go lines 14-21 — the EXISTING TestMain
// this phase's retry-loop convergence test should run under (Pattern 3
// above), not a new TestMain in a different package.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `serve --mcp` watches only with explicit `--watch` (opt-in, v0.1) | Default-on watching, opt-out via `--no-watch`, matching TS 1.3.1 | This phase (WATCH-01) | Restores the zero-config live-sync experience `install` already advertises via its byte-identical agent config |
| `daemon.Run` returns `ErrLockLive` once, session gives up forever | Defer-and-retry on a jittered cadence until a surviving session becomes the sole writer | This phase (WATCH-04) | Fixes "zero watchers after the lock holder exits" — today's code leaves every concurrent session permanently unwatched once the first holder dies |
| Green `go test ./...` + green in-process `BuildServer`→`CallTool` tests treated as sufficient proof of reachability | Real subprocess spawn + real stdio JSON-RPC session required for any claim about `serve --mcp`'s production wiring | This phase (TEST-04), motivated by Phase-2's CR-01/CR-02/BL-01 | Closes the exact class of bug where correctly-implemented, correctly-unit-tested code was dead in production because the in-process test harness bypassed the real cwd/argv→handler wiring seam |

**Deprecated/outdated:** none — this phase has no library deprecations to track; it is a pure behavior/wiring change against code already in this repo.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Exact retry interval/jitter shape (~30s ± jitter, per D-14's own text) is left to plan-time discretion, no verified TS equivalent exists (TS has no analogous retry loop — its multi-session model is pool/proxy sharing, DMON-FUT-01, explicitly out of scope) | Pattern 3 / Don't Hand-Roll | Low — CONTEXT.md's own `<decisions>` explicitly places this under "Claude's Discretion"; no TS parity claim is being made here, so there is nothing to get wrong relative to a spec |
| A2 | `codegraph daemon`'s RunE returning a non-zero exit on `ErrWatchDisabled` (inherited "for free" from the shared `Run`) is the CORRECT behavior, not an accidental regression, since an idle standalone daemon has no other useful mode | Pattern 2 | Medium — if a future Phase 7 DMON-01..04 lifecycle expects `codegraph daemon` to stay running (idle) rather than exit on a disabled policy, this phase's `Run` change could need revisiting; flagged explicitly for the planner rather than silently baked in |

**If empty:** N/A — table has 2 entries above requiring planner/user awareness, though both are low-to-medium risk and explicitly scoped as discretion by CONTEXT.md itself.

## Open Questions

1. **Should `codegraph daemon`'s RunE special-case `ErrWatchDisabled` to exit 0 with just a log line, rather than propagating it as a command error?**
   - What we know: `Run` returning a non-nil error today always means "daemon failed to do its job," and cobra will print it + exit non-zero for ANY RunE error — this is the existing, unmodified behavior of `internal/cli/daemon.go`.
   - What's unclear: whether a WSL2-detected, policy-disabled `codegraph daemon` invocation should be treated as a normal command failure (arguably correct — nothing useful for it to do) or should log-and-exit-0 (arguably friendlier for a `daemon.lock`-adjacent supervisor script that doesn't expect an idle daemon invocation to be an "error").
   - Recommendation: treat as command-failure (exit non-zero) for this phase — it is the minimal-change, most defensible interpretation, and DMON-01..04's lifecycle UX (Phase 7) is the natural place to revisit this if it proves wrong in practice. Document the decision explicitly in the plan so it isn't an accidental side effect.

2. **Does the WATCH-02 structural seam test need a real synchronization hook (like `Daemon.onSyncStart`), or is a simpler timing-based assertion (`serveWatchStart` must return within N milliseconds) sufficient?**
   - What we know: the WR-01 precedent (`serveServerPaths`) is a PURE function with no goroutines, so its test is trivially deterministic; `serveWatchStart` is fundamentally different (it must spawn a goroutine and return before that goroutine's real work executes) — timing-based assertions are inherently a little flaky under CI load, while a synchronization-hook approach (mirroring `Daemon.onSyncStart`'s existing test-only-field precedent) is deterministic but adds a small amount of test-only surface area to production code.
   - Recommendation: use the synchronization-hook approach — the codebase already has direct precedent for exactly this shape (`internal/daemon/daemon.go` lines 67-75's `onSyncStart` field, explicitly documented as "a test-only control seam... lets daemon_test.go deterministically hold a flush 'in flight'"). The plan should propose an analogous unexported hook rather than a sleep/timeout race.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Everything | ✓ | go1.26.5 darwin/arm64 `[VERIFIED]` | — |
| git CLI | TEST-04 fixtures, WATCH-04 lock tests | ✓ | 2.55.0 `[VERIFIED]` | Fixture helpers already `t.Skip` (never `t.Fatal`) on any git failure, per the existing `runGitC`/`runGitW`/`runGitM` pattern — no fallback needed, degrade gracefully |
| `mark3labs/mcp-go` client package | TEST-04 stdio harness | ✓ | v0.56.0, `NewStdioMCPClient` confirmed present `[VERIFIED]` | — |
| WSL2 environment for manual repro | Validating WATCH-03's auto-off in the real failure mode | ✗ (macOS host) | — | Table-driven unit tests with injectable `Probe.Env`/`Probe.IsWSL` cover this deterministically without needing a real WSL2 machine — STATE.md itself flags "needs a real WSL2 reproduction to validate" as an open blocker/concern carried from planning; the plan should treat the automated policy tests as sufficient for phase completion and note real-WSL2 manual validation as a follow-up, not a blocker |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** WSL2 real-hardware validation (covered by injectable-probe unit tests instead).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (no third-party test framework anywhere in this repo) |
| Config file | none — plain `go test` |
| Quick run command | `go test ./internal/watch/... ./internal/daemon/... ./internal/cli/...` |
| Full suite command | `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| WATCH-01 | `serve --mcp` watches by default; `--no-watch` opts out | unit + subprocess | `go test ./internal/watch/... -run TestWatchDisabledReason` / `go test ./test/integration/... -run TestDefaultWatch` | ❌ Wave 0 (both new) |
| WATCH-02 | Watcher startup never delays the handshake | unit (structural seam) + subprocess (latency) | `go test ./internal/cli/... -run TestServeWatchStartDeferred` / `go test ./test/integration/... -run TestHandshakeNotDelayed` | ❌ Wave 0 (both new) |
| WATCH-03 | WSL2/`/mnt` auto-off, env precedence | unit (table-driven, injectable probes) | `go test ./internal/watch/... -run TestWatchDisabledReason` | ❌ Wave 0 (new `policy_test.go`) |
| WATCH-04 | Concurrent sessions converge, goleak-clean | soak (extends existing goleak `TestMain`) | `go test ./internal/daemon/ -count=1 -run TestSoak` (extended) | ⚠️ Wave 0 (extend existing `soak_test.go`, don't create new file) |
| TEST-04 | Subprocess harness itself, CR-01 anchor | integration (new package) | `go test ./test/integration/...` | ❌ Wave 0 (whole package new) |

### Sampling Rate
- **Per task commit:** `go test ./internal/watch/... ./internal/daemon/... ./internal/cli/...` (fast, no subprocess spawn)
- **Per wave merge:** full suite command above, including the new `test/integration/...` step
- **Phase gate:** full suite green (including the two new explicit CI steps: golden — already exists — and integration — new) before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/watch/policy.go` + `internal/watch/policy_test.go` — new, covers WATCH-03
- [ ] `internal/daemon` gains `ErrWatchDisabled` + policy-check-first in `Run` + `RunWithRetry` helper — covers WATCH-02 (partial)/WATCH-04
- [ ] `internal/cli/serve.go`'s extracted watch-start seam + `internal/cli/serve_test.go`'s new structural test — covers WATCH-02
- [ ] `test/integration/` whole package (TestMain binary build, D-20 anchor fixture, D-21 default-on/NO_WATCH cases) — covers TEST-04
- [ ] `.github/workflows/ci.yml` gains an explicit `go test ./test/integration/...` step beside the existing golden step — no code gap, a CI-config gap

## Security Domain

### Applicable ASVS Categories

This phase has no auth/session/web surface — it is a local CLI/MCP-server process lifecycle change. Most ASVS categories are N/A; the relevant ones:

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | not applicable — no auth surface |
| V3 Session Management | no | not applicable |
| V4 Access Control | no | not applicable — single-user local process |
| V5 Input Validation | yes | Env var values are checked with TS's exact `=== '1'` strictness (D-10: "TS checks `=== '1'` exactly — port that strictness (only the string '1' triggers), don't invent truthiness") — this is itself an input-validation control: a permissive truthy-string check (`"true"`, `"yes"`, `"1"`, non-empty) would silently change behavior for values TS treats as falsy, a parity bug, not a security bug per se, but the same strict-equality discipline also prevents accidental activation from unrelated env noise |
| V12 File and Resources | yes | The daemon lockfile's `createLockExclusive` (unchanged this phase) already uses atomic temp-file-then-`Link` to avoid partial-write races (`[VERIFIED: internal/daemon/lock.go lines 187-227]`) — this phase's retry loop must not introduce a new code path that bypasses this exclusivity guarantee; `RunWithRetry` (Pattern 3) calls the SAME `d.Run`→`acquire()` path on every iteration, so no new lock-acquisition code is introduced |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Subprocess argv/env injection in TEST-04's harness | Tampering | `test/integration/`'s fixtures are entirely test-authored, hermetic `t.TempDir()` content — no user-controlled input reaches the spawned binary's argv/env in this harness; standard Go `exec.Command` (no shell interpolation) is already the established pattern (`execCmd` in `internal/cli/cli_test.go`) |
| A retry loop that never backs off, enabling a local resource-exhaustion loop against a contended lockfile | Denial of Service (self-inflicted, low severity) | D-14's jittered ~30s cadence (not a tight spin loop) already addresses this; Pattern 3's `time.After(jitter(interval))` with a `ctx.Done()` select is the standard non-spinning backoff shape already used elsewhere in this codebase (fsnotify's debounce timer) |
| A crashed daemon's stale lockfile blocking all future watchers indefinitely | Denial of Service | Already mitigated by the EXISTING `isStale`/PID-liveness + start-time corroboration logic in `lock.go` — this phase reuses it via D-16's confirmed self-healing behavior, introduces no new state |

## Sources

### Primary (HIGH confidence — direct source reads this session)
- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/sync/watch-policy.js` — complete `detectWsl`/`isWindowsDriveMount`/`watchDisabledReason` implementation, precedence, exact reason strings, `'1'`-strict env checks
- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/mcp/engine.js` (lines 175-270) — `startWatching()` ordering, verbatim disabled stderr message, `catchUpSync`, debounce-env handling (deferred item)
- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/sync/watcher.js` (lines 280-330) — policy enforced inside `watcher.start()`
- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/bin/codegraph.js` (lines 1555-1585) — `--no-watch` flag definition and the env-routing trick (D-05's "don't port this" target)
- `/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/node_modules/@colbymchenry/codegraph-darwin-arm64/lib/dist/utils.js` (lines 242-244) — `normalizePath`, confirmed equivalent to Go's `filepath.ToSlash`
- `internal/cli/serve.go` (whole file) — current wiring, CR-01 comment, reconcile-Sync placement, `--watch` block
- `internal/daemon/lock.go` (whole file) — `acquire`/`isStale`/`createLockExclusive` — the D-16 answer
- `internal/daemon/daemon.go` (whole file) — `New`/`Run`/`flush` teardown invariants
- `internal/watch/watcher.go` (whole file) — `Open`/`Run`/`Close`, no existing policy hook
- `internal/cli/daemon.go` (whole file) — confirms the shared-seam claim requires zero changes
- `internal/cli/serve_test.go`, `internal/cli/notice_test.go` — WR-01 precedent, real-git fixture pattern, bare-glyph helper
- `internal/mcp/markdown_test.go` (lines 1-90) — in-process client pattern precedent (contrast with TEST-04's subprocess requirement)
- `internal/mcp/server.go`, `internal/mcp/tools.go`, `internal/mcp/server_test.go` — `CODEGRAPH_MCP_TOOLS` allowlist mechanics
- `internal/daemon/soak_test.go` — existing goleak `TestMain` + `TestSoak` pattern
- `.github/workflows/ci.yml` — existing explicit golden-suite CI step precedent (GOLDEN-01)
- `$GOMODCACHE/github.com/mark3labs/mcp-go@v0.56.0/client/{stdio,client}.go` — confirmed `NewStdioMCPClient`/`Initialize`/`ListTools`/`CallTool` API surface for the EXACT pinned version
- `$GOMODCACHE/github.com/spf13/cobra@v1.10.2/flag_groups.go` — confirmed `MarkFlagsMutuallyExclusive` exists in the pinned version
- `go.mod` — version pins for mcp-go, cobra, goleak, Go toolchain

### Secondary (MEDIUM confidence)
- none used — this phase required no web search or Context7 lookups; all claims trace to direct file reads above.

### Tertiary (LOW confidence)
- none.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies, every API surface used is directly verified against the pinned `go.mod` versions in the local module cache
- Architecture: HIGH — every design pattern above is derived from reading BOTH the TS reference implementation AND the current Go extension points named in CONTEXT.md's `<canonical_refs>`, not inferred
- Pitfalls: HIGH — Pitfall 1 (D-16) and the flag/env wording finding are both proven by direct source reads, not assumed; Pitfalls 3-5 are directly observed codebase precedents

**Research date:** 2026-07-16
**Valid until:** 30 days (stable — this research is grounded in pinned dependency versions and a frozen TS 1.3.1 reference capture, not a fast-moving external API)
