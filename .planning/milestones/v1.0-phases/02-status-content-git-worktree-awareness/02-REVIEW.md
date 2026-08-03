---
phase: 02-status-content-git-worktree-awareness
reviewed: 2026-07-16T00:52:57Z
depth: deep
files_reviewed: 21
files_reviewed_list:
  - internal/gitmeta/worktree.go
  - internal/gitmeta/detect.go
  - internal/gitmeta/notice.go
  - internal/gitmeta/cache.go
  - internal/gitmeta/fixtures_test.go
  - internal/gitmeta/detect_test.go
  - internal/gitmeta/notice_test.go
  - internal/gitmeta/cache_test.go
  - internal/query/engine.go
  - internal/query/status.go
  - internal/query/worktree_notice.go
  - internal/query/render_results.go
  - internal/query/render_status.go
  - internal/query/engine_worktree_test.go
  - internal/query/files_status_test.go
  - internal/query/render_results_test.go
  - internal/query/render_status_test.go
  - internal/mcp/tools.go
  - internal/mcp/server.go
  - internal/mcp/markdown_test.go
  - internal/cli/status.go
  - internal/cli/explore.go
  - internal/cli/node.go
  - internal/cli/search.go
  - internal/cli/callers.go
  - internal/cli/callees.go
  - internal/cli/impact.go
  - internal/cli/files.go
  - internal/cli/notice_test.go
  - internal/cli/status_cli_test.go
  - testdata/golden/golden_parity_test.go
findings:
  critical: 2
  warning: 5
  info: 4
  total: 11
status: issues_found
---

# Phase 2: Code Review Report

**Reviewed:** 2026-07-16T00:52:57Z
**Depth:** deep (cross-file: import graph, call chains, entry-point reachability)
**Files Reviewed:** 21 source + 10 test
**Status:** issues_found

## Summary

The `internal/gitmeta` 4-gate cascade itself is a **faithful, correct port**. I traced all seven
fixture layouts by hand against the implementation: gate 4's counterintuitive polarity
(both common dirs non-empty AND differing ⇒ SUPPRESS) is preserved exactly, gate 3 correctly
kills the monorepo-subdir/plain-ancestor/non-git false positives, gate ordering has no early
short-circuit, and no gate is inverted. Glyph byte-parity is verified: both `warnGlyph`
constants hexdump to `e2 9a a0` with the closing quote (`22`) immediately following — no
U+FE0F. `notice_test.go`'s `b[3] == 0xef` check genuinely closes the byte-prefix trap, and
`internal/cli/notice_test.go`'s `containsBareNoticeGlyph` correctly distinguishes bare U+26A0
from the emoji variant. The additive constraint holds: `rg` over all non-test call sites
confirms every `Marshal*JSON` has exactly one caller (CLI) and every `Render*Markdown` exactly
one (MCP), with no `Marshal*JSON` body mutated. The `--json` notice gate is airtight on all 7
read commands (explore/node have no `--json` at all; the other 5 + status print the notice
strictly after the `if jsonOut { …; return }` early return).

**Two Critical defects, both proven by execution, both invisible to the green suite.**

The headline finding (CR-01) is a **literal recurrence of the Phase-1 CR-02 pattern that
`internal/mcp/markdown_test.go`'s own header comment claims to have closed**: the entire
WORK-01/WORK-02 worktree notice is **unreachable through the real `codegraph serve --mcp`
entry point**. `internal/cli/serve.go` reassigns `repoPath = ResolveCodegraphDir(start)`,
discarding the caller's actual cwd; every handler then defaults `path` to that index root, so
`startPath == repoRoot` and gate 2 short-circuits to nil on every production call. The MCP
tests pass only because they hand `BuildServer` a `repoPath` value production never produces.
`markdown_test.go` states "this fixture does not work around confinement, it is deliberately
structured to satisfy it" — it does satisfy confinement, but in doing so it constructs the one
server shape that makes detection possible, and no test drives serve.go's actual derivation.

CR-02 is a second reachability/validity failure in `testdata/golden/golden_parity_test.go`:
its new `dbSizeBytes > 0` assertion measures a directory unrelated to the store the test built,
and passes only because the reviewer's machine has a stale developer-created `../weft/.codegraph/store`.
On a fresh clone at the pinned commit it **fails** (reproduced below).

`go vet` is clean; `go test ./internal/{gitmeta,query,mcp,cli}/` is green. That is not evidence
of correctness — it is exactly the state the phase brief warned about.

---

## Critical Issues

### CR-01: The MCP worktree notice is unreachable through `codegraph serve --mcp` — WORK-01/WORK-02 is dead code on the MCP surface

**File:** `internal/mcp/server.go:81` (`BuildServer` repoPath param) + `internal/mcp/tools.go:20-22,58-70` (`resolvePath`/`openEngine`), root cause at `internal/cli/serve.go:51-55`

**Issue:**
`BuildServer(hasIndex, allowlist, repoPath)` conflates two distinct concepts into one
parameter: the **confinement root** and the **default start path** every handler passes to
`query.OpenAt`. `internal/cli/serve.go` — the only production caller — supplies the *resolved
index root*, not where the user stood:

```go
// internal/cli/serve.go:47-55
start, err := resolveStartPath(path)   // the user's cwd — e.g. the linked worktree
repoPath := start
if dir, err := query.ResolveCodegraphDir(start); err == nil {
    hasIndex = true
    repoPath = dir                     // ★ the worktree is DISCARDED here
}
s := mcp.BuildServer(hasIndex, allowlist, repoPath)
```

Then, for every tool call where the client omits `path` (the normal case — Claude Code does
not send it):

- `resolvePath(req, defaultPath)` → returns `defaultPath` == the index root
- `openEngine` → `query.OpenAt(indexRoot)` → `repoRoot = indexRoot`, `startPath = indexRoot`
- `Engine.WorktreeMismatch()` → `Detect(indexRoot, indexRoot)` → **gate 2** (`worktreeRoot == resolvedIndexRoot`) → `nil`

The whole reason `Engine` carries `startPath` separately from `repoRoot` (engine.go:26-37 spells
this out precisely) is defeated before detection is ever called: serve.go collapses the two
into the same value. Every one of the 7 compact-notice call sites and status's blockquote is
structurally unreachable in production. `internal/query/worktree_notice.go`,
`WorktreeWarningBlockquote`, and the `gitmeta.CachingDetector` wired through `BuildServer` are
all dead on this surface.

**Why the suite is green:** `internal/mcp/markdown_test.go:293` (and 344, 367) calls
`BuildServer(true, allowlist, wt)` with the **worktree** as repoPath — a value serve.go can
never produce, since it always overwrites it with `ResolveCodegraphDir`'s output. The fixture
comment at markdown_test.go:134-145 reasons carefully about confineToRepoRoot but never asks
whether serve.go produces that repoPath.

**Proof (probe test replicating serve.go's derivation verbatim, then removed):**

```
user's cwd (start)      = …/001/.claude/worktrees/probe
BuildServer(repoPath)   = …/001                        <-- serve.go passes THIS
repoPath == main?       = true
notice present via REAL serve.go derivation? false
Output head: "**Search: `Alpha`** — 1 result\n\n| Name | Kind | Location |…"
```

Contrast: the same fixture with `BuildServer(true, allowlist, wt)` (the committed test) passes.

**Fix:** Give `BuildServer`/`openEngine` the caller's start path, distinct from the confinement
root. Minimal shape:

```go
// internal/mcp/server.go
func BuildServer(hasIndex bool, allowlist map[string]bool, repoPath, startPath string) *server.MCPServer

// internal/mcp/tools.go — default to where the caller stood, still confined to repoPath
func openEngine(req mcp.CallToolRequest, defaultPath, repoPath string, detector *gitmeta.CachingDetector) (*query.Engine, func() error, error) {
	path := resolvePath(req, defaultPath)          // defaultPath == startPath now
	confined, err := confineToRepoRoot(path, repoPath)
	…
}

// internal/cli/serve.go — stop discarding `start`
s := mcp.BuildServer(hasIndex, allowlist, repoPath /* index root: confinement */, start /* caller's cwd */)
```

(Passing `start` as *both* args also works and tightens confinement to the worktree; either way
`start` must survive to `OpenAt`.)

**Required test (the one that is missing):** a test that derives repoPath the way serve.go does
— `repoPath := start; if dir, err := query.ResolveCodegraphDir(start); err == nil { repoPath = dir }`
— rather than hand-picking `wt`. Any test that calls `BuildServer` with a literal it chose
itself cannot prove reachability of this feature.

---

### CR-02: `TestGoldenParity/status`'s new `dbSizeBytes` assertion measures the wrong directory and fails on any clean checkout

**File:** `testdata/golden/golden_parity_test.go:706-716` (assertion), `testdata/golden/golden_parity_test.go:190-211` (`buildEngineAt`), `internal/query/status.go:280-285` (`Status`'s store-dir derivation)

**Issue:** Two coupled defects.

1. **The test measures a directory that has nothing to do with the index it built.**
   `buildEngineAt` indexes into a **fresh `t.TempDir()`** but constructs
   `query.NewWithRoot(reader, sourceDir)`. `Status()` then derives the store location purely
   from `repoRoot`:

   ```go
   // status.go:282
   if size, err := dbSizeBytes(filepath.Join(e.repoRoot, codegraphDirName, storeSubdir)); err == nil {
   ```

   i.e. `<weftDir>/.codegraph/store` — **not** the temp store the test just wrote. The
   assertion is therefore not measuring the engine under test at all.

2. **It passes only because of local filesystem pollution.** `../weft/.codegraph/store` exists
   on this machine (320 KB, a stale developer `codegraph init`), and `.codegraph/store` is
   untracked in weft (only `.codegraph/.gitignore` is tracked). A CI runner or any contributor
   cloning weft at the pinned commit gets no `store/` → `dbSizeBytes` errors → `Status()`
   swallows it → `DbSizeBytes = 0` → `t.Fatalf`.

**Proof (fresh `--local` clone of weft at the pinned commit `f89ae3e`):**

```
$ ls -a <fresh-clone>/.codegraph
.  ..  .gitignore                       # no store/

$ CODEGRAPH_WEFT_CORPUS=<fresh-clone> go test ./testdata/golden/ -run 'TestGoldenParity/status'
    golden_parity_test.go:715: dbSizeBytes = 0, want a positive integer
--- FAIL: TestGoldenParity/status (0.00s)
FAIL
```

This is a hard CI break gated behind an env var / sibling-checkout convention, so it will
surface the first time anyone wires the corpus into CI — and it silently "passes" for everyone
who has ever run `codegraph init` in `../weft`.

**Fix:** Make the engine's store location honest rather than inferred, then assert against it.
Either (a) have `buildEngineAt` index into `<sourceDir>/.codegraph/store` and open via
`OpenAt` (matching `buildIndexedFixture`, which already does exactly this and would make the
assertion meaningful), or (b) give `Engine` the actual store dir instead of re-deriving it:

```go
// Option (a) — smallest change, reuses the existing correct helper:
func buildEngineAt(t *testing.T, sourceDir string) *query.Engine {
	t.Helper()
	dst := buildIndexedFixture(t, sourceDir)   // copies + indexes at <dst>/.codegraph/store
	eng, closer, err := query.OpenAt(dst)
	if err != nil {
		t.Fatalf("OpenAt(%s): %v", dst, err)
	}
	t.Cleanup(func() { _ = closer.Close() })
	return eng
}
```

Separately, `Status()`'s `repoRoot + ".codegraph/store"` inference is wrong for any Engine
whose store is not at that path (`New`/`NewWithRoot` engines). It should read the store dir the
Engine was actually opened with, not reconstruct it by convention.

---

## Warnings

### WR-01: `Engine.WorktreeMismatch()` hardcodes `context.Background()`, so client cancellation cannot abort up to 20s of git subprocesses

**File:** `internal/query/engine.go:115`

**Issue:** Every `gitmeta` entry point takes a `ctx` and honours it — but the single production
call site discards the caller's context:

```go
e.mismatchCache = e.detector.Detect(context.Background(), e.startPath, e.repoRoot)
```

All six MCP handlers receive `ctx context.Context` (tools.go:90, 204, 230, 252, 273, 294, 315,
337) and **none of them use it**. A client that disconnects or times out still leaves the
handler spawning up to four `git` subprocesses at 5s each (worst case ~20s) with no way to
cancel. This is the exact "context not passed" defect class the daemon's 60s liveness watchdog
rationale in `worktree.go:22-27` exists to guard against — the timeout bounds each call, but
nothing bounds a cancelled request.

**Fix:** Thread the context through:

```go
func (e *Engine) WorktreeMismatch(ctx context.Context) *gitmeta.Mismatch {
	e.mismatchOnce.Do(func() {
		if e.startPath == "" || e.repoRoot == "" {
			return
		}
		e.mismatchCache = e.detector.Detect(ctx, e.startPath, e.repoRoot)
	})
	return e.mismatchCache
}
```

and pass the handler's `ctx` at each MCP call site (CLI passes `cmd.Context()`). Note `Status()`
also calls `WorktreeMismatch()` and would need the same plumbing.

---

### WR-02: `CachingDetector`'s map grows without bound on a client-controlled key

**File:** `internal/gitmeta/cache.go:21-24,45`

**Issue:** The cache is keyed `startPath + "\x00" + indexRoot` and **never evicts**. On the
long-lived MCP server, `startPath` is derived from the client-supplied `path` argument. I
confirmed `query.OpenAt` **succeeds for paths that do not exist** — `ResolveCodegraphDir`
stats `<dir>/.codegraph`, fails, and walks up until it finds the real root:

```
OpenAt SUCCEEDED for nonexistent path.
  startPath="…/001/does-not-exist-12345"   repoRoot="…/001"
```

`confineToRepoRoot` only rejects paths *outside* the root — it never requires existence. So a
looping or malicious client can issue `{"path": "<root>/x1"}`, `{"path": "<root>/x2"}`, … and
mint an unbounded number of distinct, permanently-retained cache entries, each also costing one
`git` spawn (defeating the cache's entire purpose). The mutex and two-value-map presence check
are both correct; the growth bound is what is missing.

Practical exploitation is throttled by the Pebble open per call, so this is a slow-burn
resource leak rather than an immediate DoS — but it is unbounded, and the entries are pure
waste (a nonexistent startPath always caches `nil`).

**Fix:** Bound the cache and/or reject non-directory start paths before caching:

```go
const maxCacheEntries = 1024

func (d *CachingDetector) Detect(ctx context.Context, startPath, indexRoot string) *Mismatch {
	if d == nil {
		return DetectIndexMismatch(ctx, startPath, indexRoot)
	}
	// A start path that is not an existing directory can never be inside a
	// working tree — don't spawn git for it and don't retain a key for it.
	if fi, err := os.Stat(startPath); err != nil || !fi.IsDir() {
		return nil
	}
	key := startPath + "\x00" + indexRoot
	d.mu.Lock()
	if v, ok := d.cache[key]; ok {
		d.mu.Unlock()
		return v
	}
	d.mu.Unlock()

	v := DetectIndexMismatch(ctx, startPath, indexRoot)

	d.mu.Lock()
	if len(d.cache) >= maxCacheEntries {
		d.cache = make(map[string]*Mismatch) // simplest bounded policy; LRU if churn matters
	}
	d.cache[key] = v
	d.mu.Unlock()
	return v
}
```

---

### WR-03: Gate 4 fails **open** on a degraded git — a transient error produces a false-positive warning

**File:** `internal/gitmeta/detect.go:78-84`

**Issue:**

```go
worktreeCommon := CommonDir(ctx, worktreeRoot)
indexCommon := CommonDir(ctx, resolvedIndexRoot)
if worktreeCommon != "" && indexCommon != "" && worktreeCommon != indexCommon {
	return nil
}
return &Mismatch{…}
```

`CommonDir` collapses *every* failure — timeout, transient fork failure, `safe.directory`
rejection — into `""`. When either side returns `""`, the suppression guard is skipped and the
function **reports a mismatch**. So the submodule / embedded-clone suppression silently stops
working whenever git hiccups on the 3rd or 4th subprocess, producing exactly the "nags users
constantly" false positive the phase brief flags as the worse failure mode. This inverts
`worktree.go:8-11`'s stated contract ("Every function here degrades to a safe zero value on ANY
failure… report 'no signal' rather than an error").

Gates 1 and 3 already proved git works in both directories, so this is a narrow window — but it
is a real fail-open, and it is trivially closed:

**Fix:** Treat an unavailable common dir as "no signal", consistent with gates 1-3:

```go
worktreeCommon := CommonDir(ctx, worktreeRoot)
indexCommon := CommonDir(ctx, resolvedIndexRoot)
// A missing common dir means git failed here — degrade to "no signal" (WORK-03),
// never to "warn anyway". Only a SHARED common dir is positive evidence of a
// genuine borrowed worktree; differing dirs mean a distinct repo (suppress).
if worktreeCommon == "" || indexCommon == "" || worktreeCommon != indexCommon {
	return nil
}
return &Mismatch{WorktreeRoot: worktreeRoot, IndexRoot: resolvedIndexRoot}
```

If TS's original genuinely fails open here, keep the behaviour but add a fixture pinning it, so
the choice is deliberate rather than incidental.

---

### WR-04: `codegraph query` and `codegraph affected` are read commands with no worktree notice

**File:** `internal/cli/query.go:40-79`, `internal/cli/affected.go:44`

**Issue:** `newQueryCmd` and `newAffectedCmd` are registered in `root.go:46-47`, both call
`query.OpenAt(start)`, both render human output from the resolved index — and neither prints
`query.WorktreeNotice(eng.WorktreeMismatch())`. WORK-02 was applied to 7 of the 9 read
commands. A user standing in a borrowed worktree running `codegraph query Foo` gets main-branch
results with no warning: the precise silent-correctness bug this phase exists to eliminate,
left live on two commands.

`internal/cli/notice_test.go`'s `noticeCommandCases` table encodes the same 7-command blind spot,
so no test catches it.

**Fix:** Add the same gated print used by `search.go:52-57` to both commands' human branches
(strictly after the `if jsonOut { … return }` early return), and extend
`noticeCommandCases` to 9 rows. If the omission is deliberate, say so in each command's doc
comment — right now it reads as an oversight.

---

### WR-05: CLI `explore`/`node` print the notice before the query runs, so failures emit a notice to stdout

**File:** `internal/cli/explore.go:57-63`, `internal/cli/node.go:56-62`

**Issue:** Both commands print the notice *before* calling the engine:

```go
fmt.Fprint(cmd.OutOrStdout(), query.WorktreeNotice(eng.WorktreeMismatch()))
out, err := eng.Explore(exploreQuery, maxFiles)
if err != nil {
	return err                  // ← notice already on stdout; cobra now prints an error
}
```

When the query fails (unknown symbol, unreadable source), stdout carries a bare worktree notice
followed by an error on stderr. The MCP surface's documented contract is the opposite —
`tools.go:196-199` explains that "every branch's failure paths return through
`mcp.NewToolResultError` BEFORE reaching that point, so 'no-op on isError results' holds
structurally". The five other CLI commands match MCP (notice after the engine call, inside the
success branch); explore/node are the two that diverge.

**Fix:** Move the print below the engine call, matching search/callers/callees/impact/files:

```go
out, err := eng.Explore(exploreQuery, maxFiles)
if err != nil {
	return err
}
fmt.Fprint(cmd.OutOrStdout(), query.WorktreeNotice(eng.WorktreeMismatch()))
fmt.Fprint(cmd.OutOrStdout(), out)
```

---

## Info

### IN-01: `golden_parity_test.go`'s `WorktreeMismatch != nil` assertion is now a vestigial tautology with a stale message

**File:** `testdata/golden/golden_parity_test.go:651-653`

**Issue:** The message still reads `"(Phase-4 sync placeholder)"` — untrue since this phase made
the field live. Worse, the assertion can no longer fail: `buildEngineAt` uses `NewWithRoot`,
which never sets `startPath`, so `WorktreeMismatch()` returns nil by construction
(engine.go:112). It tests the degradation path while claiming to test a placeholder.

**Fix:** Either delete it, or retitle it to what it actually pins:
`"want nil — NewWithRoot engines carry no startPath and must degrade safely (D-14)"`.

### IN-02: `Project:` line renders the raw `-p` value while the mismatch paths are absolute+resolved

**File:** `internal/cli/status.go:61`, `internal/query/render_status.go:173`

**Issue:** `RenderStatusText(result, start)` receives `resolveStartPath`'s output, which returns
the `-p` flag **verbatim** (`internal/cli/query.go:18-19`). `codegraph status -p ../foo` prints
`Project: ../foo` directly above `Running in: /abs/resolved/path` — two different path
conventions in one 6-line block.

**Fix:** `fmt.Fprintf(&b, "Project: %s\n", projectPath)` should receive an absolutized path;
absolutize in `newStatusCmd` before the call (`OpenAt` already does this internally for
`startPath`).

### IN-03: `TestNoticeOnWorktreeMismatch` asserts only a single-character prefix

**File:** `internal/cli/notice_test.go:169`

**Issue:** `strings.HasPrefix(out, glyph)` where `glyph` is one rune. Any output beginning with
`⚠` passes — a truncated, garbled, or wrong-wording notice would not be caught. The MCP sibling
does this correctly (`markdown_test.go:53` pins the full lead sentence
`"⚠ CodeGraph results below come from a different git worktree"`).

**Fix:** Assert against the full lead text, sourced from `gitmeta` rather than pasted:
`wantPrefix := (&gitmeta.Mismatch{WorktreeRoot: "/a", IndexRoot: "/b"}).Notice()[:len("⚠ CodeGraph results below come from a different git worktree")]`
— or simply `strings.HasPrefix(out, "⚠ CodeGraph results below come from a different git worktree")`.

### IN-04: Markdown table cells interpolate names/paths without escaping `|`

**File:** `internal/query/render_results.go:52,146`

**Issue:** `fmt.Fprintf(&b, "| \`%s\` | %s | \`%s:%d\` |\n", l.Name, l.Kind, l.FilePath, l.StartLine)`
— a `|` in `FilePath` (legal on POSIX) or a backtick in a symbol name silently corrupts the
table for the consuming model. Low impact (the consumer is an LLM, not a parser), but it is a
free fix.

**Fix:** `strings.ReplaceAll(s, "|", "\\|")` on interpolated cell values in
`renderLocationTable` and `RenderFilesMarkdown`'s flat branch.

---

## Verified Correct (no action)

Recorded so a future reader does not re-litigate these:

- **Gate cascade fidelity** — all 7 fixture layouts traced by hand; gate 4 polarity, gate 3's
  false-positive kill, and gate ordering are all faithful. No inversion, no early short-circuit.
- **Glyph byte-parity** — `xxd` confirms `e2 9a a0` + `0x22` in both `internal/gitmeta/notice.go:11`
  and `internal/query/worktree_notice.go:14`; no U+FE0F. `notice_test.go:43`'s `b[3] == 0xef`
  check and `containsBareNoticeGlyph`'s variation-selector lookahead both genuinely close the
  byte-prefix trap.
- **Additive constraint** — no `Marshal*JSON` body changed; call-graph confirms 1 CLI caller per
  `Marshal*JSON` and 1 MCP caller per `Render*Markdown`; no `Render*` leaked onto a `--json` path.
- **`--json` notice gate** — airtight on all 7 read commands + status. explore/node have no
  `--json` mode; the rest print strictly after the early return.
- **`nil` → `null` marshalling** — `*gitmeta.Mismatch` nil marshals to the literal `null` token,
  pinned by `TestStatusWorktreeMismatchJSONShape`. No import cycle: `query → gitmeta` is a clean
  new leaf edge (gitmeta imports only stdlib).
- **Context leak** — `WorktreeRoot`/`CommonDir` both `defer cancel()` correctly.
- **`--git-common-dir` relative resolution** — correctly joined against `dir` before `realpath`
  (worktree.go:79-83).
- **Subprocess safety** — no attacker-controlled argv (paths go to `cmd.Dir`, never argv);
  `cmd.Stdin = nil` prevents interactive blocking; `cmd.Output()` bounds output; git absent ⇒ `""`.
- **`CachingDetector` correctness** — mutex held correctly on both read and write; two-value map
  form correctly distinguishes cached-nil from absent; nil-receiver safe.
- **`dbSizeBytes` / `newestSourceMtime` traversal** — `filepath.WalkDir` does not follow symlinks,
  so no loop or escape; root-vs-deeper error split behaves as documented; `Status()` swallows the
  error and leaves 0.
- **`internal/mcp/markdown_test.go`'s SURF-06 assertions** — the dual `json.Unmarshal` must-fail +
  markdown-marker check is genuinely non-defeatable, exactly as its header claims. It is a good
  test; it just does not cover reachability (CR-01).

---

_Reviewed: 2026-07-16T00:52:57Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
