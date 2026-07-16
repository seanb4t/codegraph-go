---
phase: 02-status-content-git-worktree-awareness
reviewed: 2026-07-16T01:41:35Z
depth: deep
iteration: 2
review_of: .planning/phases/02-status-content-git-worktree-awareness/02-REVIEW-FIX.md
diff_base: d31d48b
files_reviewed: 26
files_reviewed_list:
  - internal/cli/serve.go
  - internal/cli/query.go
  - internal/cli/affected.go
  - internal/cli/explore.go
  - internal/cli/node.go
  - internal/cli/status.go
  - internal/cli/search.go
  - internal/cli/callers.go
  - internal/cli/callees.go
  - internal/cli/impact.go
  - internal/cli/files.go
  - internal/cli/root.go
  - internal/cli/notice_test.go
  - internal/mcp/server.go
  - internal/mcp/tools.go
  - internal/mcp/markdown_test.go
  - internal/mcp/server_test.go
  - internal/mcp/reconnect_test.go
  - internal/query/engine.go
  - internal/query/status.go
  - internal/query/worktree_notice.go
  - internal/query/engine_worktree_test.go
  - internal/query/files_status_test.go
  - internal/query/status_staleness_test.go
  - internal/gitmeta/detect.go
  - internal/gitmeta/cache.go
  - internal/gitmeta/worktree.go
  - internal/gitmeta/cache_test.go
  - testdata/golden/golden_parity_test.go
findings:
  critical: 1
  warning: 4
  info: 3
  total: 8
status: issues_found
---

# Phase 2: Code Review Report (iteration 2 — re-review of the fix commits)

**Reviewed:** 2026-07-16T01:41:35Z
**Depth:** deep (cross-file: call chains, trust-boundary data flow, mutation testing of the new guards)
**Files Reviewed:** 26 (7 fix commits, `dc6ddd5..HEAD`)
**Status:** issues_found

## Summary

**Six of the seven fixes are correct. One introduced a new Critical defect, and the two
regression guards the fix set added do not guard what they claim to.**

The headline concern the brief raised — *did CR-01 weaken path confinement?* — is **answered
no, conclusively**. I traced `req → resolvePath → confineToRepoRoot → OpenAt → node.go`'s
source-read confinement by hand. `openEngine`'s confinement anchor is `repoPath`, which
`serve.go:122` sources from `query.ResolveCodegraphDir(start)` — an **operator**-configured
value (the `-p` flag or the server's cwd), never a client-supplied one. The client's
`args.path` reaches only `resolvePath`'s *default-override* slot and is checked against
`repoPath` before `OpenAt` ever runs. Source reads stay anchored on `Engine.repoRoot`
(`node.go:34-64`), which is `ResolveCodegraphDir(confined)` and therefore always inside
`repoPath`. `ResolveCodegraphDir` uses `filepath.Abs` only (no `EvalSymlinks`), so the
default `startPath` is always a textual descendant of `repoPath` and confinement of the
default can never spuriously reject — the structural claim in `BuildServer`'s doc comment is
accurate. **CR-01's blast radius is also clean:** I checked all 9 `openEngine` call sites, both
handler constructors, and all 17 `BuildServer` call sites individually — **no argument swap
anywhere**. And a swap would *tighten* confinement (`startPath` is at-or-below `repoPath`), not
weaken it.

WR-03's "intentional divergence from TS" **checks out against the real TS source**. I read
`sync/worktree.js:147-151` directly: `if (worktreeCommon && indexCommon && worktreeCommon !== indexCommon) return null;`
— TS genuinely falls through to *reporting* a mismatch whenever either `gitCommonDir` call
fails. Our fail-closed inversion does not over-correct: gates 1 and 3 already proved `git
rev-parse` works in both directories, so both `--git-common-dir` calls succeed in every
realistic layout, and the linked-worktree fixture (which requires gate 4's *positive* path)
still passes. WR-02's bound, negative caching, and nil-receiver safety are all correct.
WR-04/WR-05 are correct and their new tests are honest. The additive constraint held (`git
diff` confirms **no** `Marshal*JSON` body was touched), glyph byte-parity survives (`xxd`:
`22 e2 9a a0 22` in both files — bare U+26A0, closing quote immediately after), `--json` is
gated on all 9 read commands including the newly-added `query`/`affected`, and no assertion in
any touched test was weakened.

**But the fixes were validated by a green suite, and the green suite is lying in three places
— all three proven by execution, not inspection:**

**BL-01 (Critical)** — WR-01's context threading combined with WR-02's server-scoped cache
creates a **permanent cache poisoning**: one cancelled tool call (mcp-go v0.56 implements
`notifications/cancelled` by cancelling exactly the ctx WR-01 now threads) makes
`DetectIndexMismatch` return `nil`, and `CachingDetector` stores that `nil` forever. **The
worktree notice then silently disappears for the entire life of the MCP server.** I proved this
through the real `BuildServer`/`CallTool` path. Before WR-01, `context.Background()` made this
impossible — **the fix strictly traded a bounded 20s cost for the permanent, silent loss of the
phase's core deliverable.**

**WR-01 (Warning)** — I reintroduced CR-01 at its **root cause** (`serve.go`,
`BuildServer(…, repoPath, repoPath)`) and ran the **entire** suite including the golden corpus:
**zero failures.** `deriveServeRepoPath`'s doc comment claims routing fixtures through it
"closes that gap (CR-01's 'required test')". It does not. It *replicates* serve.go's derivation
inside the test; nothing asserts serve.go *performs* it. This would be the **third** recurrence
of this exact pattern in this project.

**WR-02 (Warning)** — I re-anchored confinement on the start path
(`confineToRepoRoot(path, defaultPath)`) and the entire suite stayed green. The sole confinement
guard, `TestOpenEnginePathConfinedToRepoRoot`, runs only in the degenerate `startPath == repoPath`
configuration, so it cannot distinguish which of CR-01's two new same-typed string params
confinement anchors on.

**WR-03 (Warning)** — CR-02's fix routed the golden `status` subtest through
`buildIndexedFixture`, whose `copyDir` copies the source tree **verbatim including its own
`.codegraph/store`**, and `indexer.Run` **merges into** rather than replaces that store. Proven:
a fixture with **exactly 1 Go file** reports `fileCount=69, nodeCount=762, languages=[go javascript python]`.
CR-02 fixed "measures the wrong directory" but moved the same invisible pollution *inside the
store under test*.

`go vet ./...` is clean; `go test ./... ./testdata/golden/` is green. (Note: `go test ./...`
alone **never runs the golden suite** — the go tool ignores `testdata/` directories.)

---

## Critical Issues

### CR-01 (BL-01): One cancelled MCP tool call permanently disables the worktree notice for the whole server — WR-01's fix poisons WR-02's shared cache

**File:** `internal/gitmeta/cache.go:80-91` (the cache store), root cause at `internal/query/engine.go:123` + `internal/gitmeta/detect.go:34-37`, triggered from `internal/mcp/tools.go:114,237,259,280,301,322,344,361`

**Issue:**

WR-01 replaced `context.Background()` with the caller's real, cancelable context. WR-02 made the
detector's cache server-scoped and long-lived. Neither change is wrong alone; **together they
are a silent-correctness bug of exactly the class this phase exists to eliminate.**

`DetectIndexMismatch` collapses *every* git failure into gate 1's `""` and returns `nil` — by
design (WORK-03). It cannot distinguish **"checked, clean"** from **"the git spawn was aborted
because the caller's context was cancelled"**. `CachingDetector.Detect` then caches that `nil`
permanently:

```go
// cache.go:80-91
v := DetectIndexMismatch(ctx, startPath, indexRoot)   // ctx cancelled ⇒ nil, indistinguishable from "clean"
d.mu.Lock()
if len(d.cache) >= maxCacheEntries { d.cache = make(map[string]*Mismatch) }
d.cache[key] = v                                       // ★ nil is now the permanent verdict
d.mu.Unlock()
```

The window is **not** narrow. No handler checks `ctx.Err()`, so a cancellation arriving at any
point during `openEngine` (Pebble open) or the query itself (`eng.Explore` reads source files
fresh from disk) leaves an **already-cancelled** ctx at the `eng.WorktreeMismatch(ctx)` call on
the very next line. `exec.CommandContext.Start` returns `ctx.Err()` immediately on a cancelled
context, so detection fails instantly and deterministically. **The entire handler duration is
the exposure window.**

This is reachable in production, not theoretical. mcp-go v0.56.0 (`server/request_handler.go:83-93`)
implements the MCP spec's `notifications/cancelled` by cancelling precisely this ctx:

```go
// Wrap context with cancel for in-flight request cancellation (MCP spec: notifications/cancelled)
ctx, cancel := context.WithCancel(ctx)
...
s.inflightCancels.Store(key, cancel)
```

Claude Code cancels in-flight tool calls on user interrupt and on client-side timeout — routine
events, and the exact scenario WR-01 was written to handle.

**Proof (real `BuildServer` → `CallTool` handler path; probe written, run, and removed):**

```
sanity: fresh server DOES emit the notice
call 1 issued with a cancelled context
POISONED: after ONE cancelled tool call, this server no longer emits the worktree notice on healthy calls.
got: "**Callers of `helper`** — 1 caller\n\n| Name | Kind | Location |\n|---|---|---|\n| `Alpha` | function | `pkga/pkga.go:13` |\n"
--- FAIL: TestProbeCancelledCallPoisonsServer
```

Reproduced independently at the unit level:

```
call 1 (cancelled ctx): <nil>
call 2 (live ctx):      <nil>     <-- healthy request, real linked-worktree fixture, notice gone forever
```

**Severity rationale:** a user in a borrowed worktree presses Esc once, then gets main-branch
results with **no warning** for the rest of the session. That is the precise silent-correctness
bug WORK-01/WORK-02 exist to prevent, now triggerable by a keystroke. Before WR-01 this was
impossible. The fix is strictly worse than the defect it addressed.

**Fix:** never cache a verdict computed under a cancelled context — a cancelled probe is
"unknown", not "clean":

```go
// internal/gitmeta/cache.go
v := DetectIndexMismatch(ctx, startPath, indexRoot)

// BL-01: a verdict computed under a cancelled context is NOT a verdict.
// DetectIndexMismatch collapses the aborted git spawn into gate 1's "" and
// returns nil — indistinguishable from "checked, no mismatch". Caching that
// would make one cancelled tool call permanently disable the notice for this
// (startPath, indexRoot) pair for the rest of this long-lived server's life.
// Return the degraded verdict for THIS call (WORK-03: never fail a read on
// git), but leave the cache untouched so the next healthy call re-probes.
if ctx.Err() != nil {
	return v
}

d.mu.Lock()
if len(d.cache) >= maxCacheEntries {
	d.cache = make(map[string]*Mismatch)
}
d.cache[key] = v
d.mu.Unlock()
return v
```

`Engine.mismatchOnce` (engine.go:119) latches the same way, but each MCP Engine is per-call and
discarded, and the CLI's ctx is always `context.Background()` (see IN-01), so no change is
needed there. Consider a short comment saying so, since the asymmetry is non-obvious.

**Required test (the one that is missing):** two successive tool calls against **one**
`BuildServer` where the first carries a cancelled context and the second does not; assert the
second still starts with `worktreeNoticeText`. `TestWorktreeNoticeConsistentAcrossCalls`
(markdown_test.go:389) is the natural sibling — it already pins "the notice does not flap
between calls" but only on the happy path, which is why it missed this.

---

## Warnings

### WR-01: `deriveServeRepoPath` does not close the gap it says it closes — reintroducing CR-01 at its root cause leaves the entire suite green

**File:** `internal/mcp/markdown_test.go:130-150` (helper + its claim), gap at `internal/cli/serve.go:122`

**Issue:** The helper's doc comment asserts:

> "A test that calls BuildServer with a literal it chose itself cannot prove this feature is
> reachable through the real entry point; **routing every fixture-driven BuildServer call through
> this helper closes that gap (CR-01's 'required test')**."

It does not. The helper *replicates* serve.go's derivation **inside the test package**; no test
observes serve.go's actual `BuildServer` call. The tests catch a regression in `internal/mcp`
(if `openEngine` defaulted to `repoPath`, `TestWorktreeNoticeOnMismatch` would fail — verified) —
but they are blind to the **root cause the first review identified**, which lives in
`internal/cli/serve.go`.

**Proof (mutation test — the exact CR-01 bug reintroduced at its root cause, then reverted):**

```diff
- s := mcp.BuildServer(hasIndex, allowlist, repoPath, start)
+ s := mcp.BuildServer(hasIndex, allowlist, repoPath, repoPath)
```
```
$ go test ./... ./testdata/golden/
=== FAIL count: 0 ===        # entire suite, golden corpus included, 100% green
```

The three-line comment block at `serve.go:114-121` warning future readers not to collapse the
params is currently the *only* thing protecting this. Note also that `go test ./...` **never
runs the golden suite at all** — the go tool ignores `testdata/` directories — so CI must invoke
`./testdata/golden/` explicitly or that oracle is dead weight.

Given this phase's own history (Phase-1 CR-02 → Phase-2 CR-01 is the same pattern twice), an
unguarded root cause is the single most likely place for a third recurrence.

**Fix:** assert on serve.go's own wiring, not a replica of it. Extract the derivation so both
production and test call the same function, and let the test observe what serve.go passes:

```go
// internal/cli/serve.go — one exported-to-package seam, no behavior change
// serveServerPaths returns BuildServer's two DELIBERATELY DISTINCT arguments
// (CR-01): the confinement root (the resolved index root) and the caller's
// actual start path. Extracted so a test can pin the derivation itself,
// rather than replicating it and proving nothing (WR-01).
func serveServerPaths(start string) (repoPath string, hasIndex bool, err error) {
	repoPath = start
	if dir, err := query.ResolveCodegraphDir(start); err == nil {
		return dir, true, nil
	} else if !errors.Is(err, query.ErrNotInitialized) {
		return "", false, err
	}
	return repoPath, false, nil
}
```

```go
// internal/cli/serve_test.go — fails the moment serve.go collapses the two again
func TestServeKeepsStartPathDistinctFromConfinementRoot(t *testing.T) {
	wt, main := statusWorktreeMismatchFixture(t)   // already exists in this package
	repoPath, hasIndex, err := serveServerPaths(wt)
	if err != nil || !hasIndex {
		t.Fatalf("serveServerPaths(%s) = %q, %v, %v", wt, repoPath, hasIndex, err)
	}
	if repoPath == wt {
		t.Fatal("repoPath must be the RESOLVED index root, not the start path")
	}
	if repoPath != main {
		t.Fatalf("repoPath = %q, want the main checkout %q", repoPath, main)
	}
	// The pair serve.go hands BuildServer must remain distinct (CR-01).
}
```

### WR-02: the confinement guard cannot tell which of CR-01's two new params it anchors on — re-anchoring it on the start path is not caught by any test

**File:** `internal/mcp/server_test.go:224-266` (`TestOpenEnginePathConfinedToRepoRoot`), subject at `internal/mcp/tools.go:65`

**Issue:** CR-01 split a security-adjacent parameter into two adjacent, same-typed strings. The
only test pinning the trust boundary builds its server as `BuildServer(true, {...}, dir, dir)` —
`startPath == repoPath`. In that degenerate configuration the two anchors are **indistinguishable**,
so the test passes identically whichever one `openEngine` uses.

**Proof (mutation test, then reverted):**

```diff
- confined, err := confineToRepoRoot(path, repoPath)
+ confined, err := confineToRepoRoot(path, defaultPath)   // anchor on the START path instead
```
```
$ go test ./internal/... ./testdata/golden/
(no failures — NOT caught)
```

To be precise about the risk: this mutation is **not** currently a vulnerability — `defaultPath`
is `serve.go`'s operator-controlled `start`, and confining to it is *tighter* than confining to
`repoPath`. The finding is that **the trust boundary is now untested in the only configuration
production actually uses** (`startPath != repoPath`, i.e. a linked worktree). A future refactor
that makes `defaultPath` client-influenced — for example "remember the last `path` argument as
the new default", an entirely plausible convenience change — would turn this into a real
confinement bypass with the guard still green.

**Fix:** run the existing confinement assertion in the configuration production uses, so it pins
the anchor rather than a coincidence:

```go
// internal/mcp/server_test.go — add alongside TestOpenEnginePathConfinedToRepoRoot
// WR-02: with startPath != repoPath (the linked-worktree shape serve.go
// actually produces), assert confinement anchors on repoPath — a path that is
// OUTSIDE repoPath but would be accepted if the anchor were startPath, and
// vice versa, is what makes this test able to fail.
func TestConfinementAnchoredOnRepoRootNotStartPath(t *testing.T) {
	wt, main := mcpWorktreeMismatchFixture(t)      // startPath=wt, repoPath=main
	s := BuildServer(true, map[string]bool{"status": true}, deriveServeRepoPath(t, wt), wt)

	// A sibling of wt is INSIDE repoPath (main) but OUTSIDE startPath (wt).
	// Anchored on repoPath (correct) this resolves; anchored on startPath it
	// would be rejected — so this call distinguishes the two anchors.
	sibling := filepath.Join(main, "pkga")
	result := callTool(t, s, "codegraph_status", map[string]any{"path": sibling})
	if result.IsError {
		t.Fatalf("path inside repoPath was rejected — confinement is anchored on startPath, not repoPath: %+v", result)
	}

	outside := copyFixture(t)
	indexFixture(t, outside)
	result = callTool(t, s, "codegraph_status", map[string]any{"path": outside})
	if !result.IsError {
		t.Fatal("path outside repoPath was accepted — confinement is not enforced")
	}
}
```

### WR-03: CR-02 moved the invisible filesystem pollution *inside* the store under test — the golden fixture inherits the corpus checkout's own `.codegraph/store`

**File:** `testdata/golden/golden_parity_test.go:208-218` (`buildEngineAt`), `:280-304` (`copyDir`), `:315-329` (`buildIndexedFixture`)

**Issue:** CR-02's fix routes `buildEngineAt` — and therefore the golden `status` subtest's
`fileCount`/`nodeCount`/`nodesByKind`/`languages`/`dbSizeBytes` assertions — through
`buildIndexedFixture` for the first time. Two properties combine badly:

1. `copyDir` (`:283`) walks the source tree with **no exclusions** — it copies `.git/` *and*
   `.codegraph/` verbatim, including a live Pebble store's `MANIFEST`/`CURRENT`/`*.sst`/`*.log`.
2. `buildIndexedFixture` (`:321-325`) only `os.MkdirAll`s the store dir before `indexer.Run`.
   It mirrors `internal/cli/init.go`'s `MkdirAll` but **not** `internal/cli/index.go:57`'s
   `os.RemoveAll(storeDir)` — and `indexer.Run` does not clear the store itself (the caller
   owns that, `pipeline.go:76-116`). So the fresh index **merges into** the copied one.

`../weft/.codegraph/store` exists on this machine (314 KB, an old developer `codegraph init`) —
the *same* pollution CR-02 was written to eliminate. It is now copied straight into the
directory the assertions measure.

**Proof (fixture containing exactly ONE Go file with one function; probe run, then removed):**

```
copyDir would copy 313955 bytes of pre-existing store into the fixture
RESULT fileCount=69 nodeCount=762 languages=[go javascript python] dbSizeBytes=299149
       (this fixture has exactly 1 source file)
```

`Status()` reports **69 files and 762 nodes for a 1-file tree** — every one of them resurrected
from the copied store.

**Honest scoping — this is latent, not currently producing a wrong result.** I ran the real
golden status subtest with and without the pollution:

```
WITH    ../weft/.codegraph/store present:  fileCount=68 nodeCount=760 languages=[go javascript python]  PASS
WITHOUT (store stashed; what CI sees):     fileCount=68 nodeCount=760 languages=[go javascript python]  PASS
```

Identical — but only because this machine's store happens to be current for the pinned commit.
`resolveWeftCorpus` pins the corpus to commit `f89ae3e`; nothing pins the *store*, which a
developer may have built at any HEAD or any older schema version. The moment those diverge, the
oracle silently reports the stale store's graph. That is precisely the "passes for everyone who
has ever run `codegraph init` in `../weft`" failure mode CR-02 was raised to kill.

**Fix:** make the fixture's store honest by construction — mirror `codegraph index`'s full
rebuild rather than `codegraph init`'s create:

```go
// testdata/golden/golden_parity_test.go
func buildIndexedFixture(t *testing.T, src string) string {
	t.Helper()

	dst := t.TempDir()
	copyDir(t, src, dst)

	// WR-03: copyDir copies the SOURCE tree verbatim, including any
	// .codegraph/ a developer's own `codegraph init` left in the corpus
	// checkout — and indexer.Run MERGES into an existing store rather than
	// replacing it (internal/cli/index.go owns the wipe, not the pipeline).
	// Without this RemoveAll the fixture inherits the corpus's stale graph and
	// every count/language assertion below measures it instead of what this
	// test indexed. Mirrors `codegraph index`'s full-rebuild semantics.
	storeDir := filepath.Join(dst, ".codegraph", "store")
	if err := os.RemoveAll(storeDir); err != nil {
		t.Fatalf("clear inherited store dir: %v", err)
	}
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store dir: %v", err)
	}
	if _, err := indexer.Run(dst, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture at %s: %v", dst, err)
	}
	return dst
}
```

Better still, have `copyDir` skip `.codegraph/` outright, so no fixture can ever inherit one.

### WR-04: `BuildServer`'s parameter order is inverted relative to its own internal handler calls — a silent-swap footgun on two adjacent same-typed strings

**File:** `internal/mcp/server.go:94` vs `:111,114`

**Issue:** The signature takes `(…, repoPath, startPath)` but every internal call inverts it:

```go
func BuildServer(hasIndex bool, allowlist map[string]bool, repoPath, startPath string) *server.MCPServer {
	...
	s.AddTool(exploreTool(), exploreHandler(startPath, repoPath, detector))          // ← inverted
	s.AddTool(companionTool(name), companionHandler(name, startPath, repoPath, detector))  // ← inverted
}
```

Both are `string`. Both are paths. Nothing type-checks the distinction, and the order flips
across a single function boundary. A reader reconciling `BuildServer(…, repoPath, startPath)`
against `exploreHandler(startPath, repoPath, …)` has only the doc comments to go on. Combined
with WR-02 (no test distinguishes the two anchors) and WR-01 (no test pins serve.go's call), a
swap at any of these three layers compiles, passes CI, and silently inverts detection.

I verified **no swap exists today** — all 17 `BuildServer` call sites and all 9 `openEngine`
call sites are correct. This is about keeping it that way.

**Fix:** make the compiler carry the distinction that the doc comments currently carry alone:

```go
// ServerPaths carries BuildServer's two DELIBERATELY DISTINCT roots (CR-01)
// as named fields, so they cannot be transposed at a call site: both are
// plain strings, and a positional swap would compile silently while
// inverting both confinement and worktree detection.
type ServerPaths struct {
	// RepoPath is the confinement root: the RESOLVED index root
	// (query.ResolveCodegraphDir's output) every handler's
	// confineToRepoRoot check anchors against.
	RepoPath string
	// StartPath is the caller's actual starting directory (serve.go's
	// `start`), each handler's default "path" and the value that must reach
	// query.OpenAt for WorktreeMismatch to have anything to compare.
	StartPath string
}

func BuildServer(hasIndex bool, allowlist map[string]bool, paths ServerPaths) *server.MCPServer
```

At minimum, align the internal call order with the signature so the two read the same way.

---

## Info

### IN-01: WR-01's CLI half is inert — `cmd.Context()` is always `context.Background()`

**File:** `internal/cli/root.go:57-59`, consumed at `internal/cli/{query,affected,explore,node,search,callers,callees,impact,files,status}.go`

**Issue:** `Execute()` calls `newRootCmd().Execute()`, not `ExecuteContext(...)`, and no signal
handling is wired. Cobra therefore sets `c.ctx = context.Background()`, so every
`cmd.Context()` the fix threaded through the 9 CLI commands is an uncancellable Background
context. `Engine.WorktreeMismatch`'s doc comment ("every MCP handler and CLI command already
receives a real, cancelable context (the handler's ctx / `cmd.Context()`)") is true for MCP and
false for the CLI.

Harmless today — a one-shot CLI process has nothing to cancel, and this is also why BL-01
cannot bite the CLI. But the plumbing is dead until `Execute` wires a signal-cancelled context,
and the comment reads as if it already does.

**Fix:** either wire it and make the claim true —

```go
func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return newRootCmd().ExecuteContext(ctx)
}
```

— or amend `WorktreeMismatch`'s comment to say the CLI passes `cmd.Context()` for uniformity
and future signal handling, and that it is Background today.

### IN-02: golden `WorktreeMismatch` assertion is no longer a tautology, but its message is still wrong

**File:** `testdata/golden/golden_parity_test.go:658-660`

**Issue:** Carried from iteration 1's IN-01 (Info, out of the fixer's scope) — but CR-02 changed
its meaning and nobody updated it. `buildEngineAt` now opens via `OpenAt`, so `startPath` **is**
set and the assertion genuinely runs the 4-gate cascade (it passes because the fixture temp dir
resolves gate 2 to equality). It is now a real test with a stale message: `"(Phase-4 sync
placeholder)"` describes neither the field (live since 02-04) nor what is being pinned.

**Fix:** `t.Errorf("status.WorktreeMismatch = %v, want nil — the fixture's start path and index root are the same tree (gate 2)", *got.WorktreeMismatch)`

### IN-03: iteration-1 Info findings remain open (correctly out of the fixer's scope)

**Files:** `internal/cli/notice_test.go:178`, `internal/cli/status.go:61`, `internal/query/render_results.go:52,146`

**Issue:** Recorded so they are not lost, not as a criticism of the fix set — the fixer's brief
scoped Info out:

- **IN-03 (iter 1):** `TestNoticeOnWorktreeMismatch` still asserts `strings.HasPrefix(out, glyph)`
  on a single rune; any output starting with `⚠` passes. The MCP sibling pins the full lead
  sentence (`markdown_test.go:55`) and is the model to copy.
- **IN-02 (iter 1):** `Project:` renders the raw `-p` value while `Running in:` is absolute+resolved.
- **IN-04 (iter 1):** markdown table cells interpolate names/paths without escaping `|`.

---

## Verified Correct (no action — re-verified this iteration)

- **CR-01's confinement is NOT weakened.** `repoPath` (the anchor) is `ResolveCodegraphDir(start)`
  from `serve.go:53` — operator-configured, never client-supplied. The client's `args.path`
  reaches only `resolvePath`'s default-override and is checked by `confineToRepoRoot` **before**
  `OpenAt`. Source reads stay anchored on `Engine.repoRoot` (`node.go:34-64`), always inside
  `repoPath`. `ResolveCodegraphDir` uses `filepath.Abs` only, so no `EvalSymlinks` mismatch can
  make the default path spuriously fail confinement (checked specifically because macOS
  `/var`→`/private/var` would expose it).
- **CR-01 blast radius: no swapped call site.** All 17 `BuildServer`, 9 `openEngine`, and both
  handler-constructor call sites checked individually. `serve.go:122` passes `(repoPath, start)`
  correctly. A swap would tighten, not weaken, confinement.
- **WR-03's TS claim is accurate.** Read `sync/worktree.js:147-151` verbatim: TS's gate 4 is
  `if (worktreeCommon && indexCommon && worktreeCommon !== indexCommon) return null;` — it does
  fall through to reporting on a degraded git. Our fail-closed inversion does not over-correct:
  gates 1/3 already proved git works in both dirs, and the linked-worktree fixture (which needs
  gate 4's positive path) still passes. Gate 4's counterintuitive polarity (differing common
  dirs ⇒ SUPPRESS) is preserved.
- **WR-02's cache mechanics.** Bound is correct (`len >= max` reset ⇒ never exceeds 1024, no
  off-by-one); negative caching intact (two-value map form preserved — `nil` still means
  "checked, clean", distinguishable from absent); nil-receiver path safe; reset under lock. The
  `os.Stat` short-circuit is behavior-preserving (gate 1 would return `""` for a nonexistent
  path anyway) and carries no TOCTOU risk (it gates caching, not access). The nil-detector path
  skips the stat but reaches the same verdict. *Its interaction with WR-01 is BL-01.*
- **WR-04/WR-05 are correct and honestly tested.** `query`/`affected` print strictly after their
  `if jsonOut { …; return }`; `explore`/`node` print after the engine call and the notice is
  **not** dropped on success (`TestNoticeOnWorktreeMismatch` covers both rows).
  `TestNoticeNotEmittedOnQueryFailure` drives genuinely failing calls. `noticeCommandCases` is 9
  rows matching `root.go:45-50`'s 9 registered read commands.
- **Additive constraint held.** `git diff dc6ddd5~1..HEAD -- internal/query/` touches **no**
  `Marshal*JSON` body; `status.go`'s diff is the `ctx` parameter and doc comment only.
- **Glyph byte-parity survives.** `xxd` on both `warnGlyph` constants: `22 e2 9a a0 22` — bare
  U+26A0, closing quote immediately after, no U+FE0F. `containsBareNoticeGlyph`'s
  variation-selector lookahead still closes the byte-prefix trap.
- **`--json` never carries the notice on any of the 9 read commands**, including the two WR-04
  added. `TestNoticeSuppressedInJSON` covers all 7 `--json`-capable commands + status;
  `explore`/`node` have no `--json` mode.
- **No weakened assertions.** `server_test.go`, `reconnect_test.go`, `files_status_test.go`,
  `status_staleness_test.go`, `engine_worktree_test.go` diffs are mechanical signature updates
  only (`dir` → `dir, dir`; `Status()` → `Status(context.Background())`).
- **CR-02's core claim holds.** `dbSizeBytes` now measures the store `buildEngineAt` actually
  built; the `> 0` assertion cannot false-pass, since the fresh index alone guarantees it. (The
  *contamination* of the sibling count/language assertions is WR-03.)

---

_Reviewed: 2026-07-16T01:41:35Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep — iteration 2 (re-review of fixes)_
_All probe/mutation artifacts removed; working tree clean, no source modified._
</content>
</invoke>
