---
phase: 02-status-content-git-worktree-awareness
fixed_at: 2026-07-16T01:58:51Z
review_path: .planning/phases/02-status-content-git-worktree-awareness/02-REVIEW-2.md
iteration: 2
findings_in_scope: 5
fixed: 5
skipped: 0
status: all_fixed
---

# Phase 2: Code Review Fix Report (iteration 2)

**Fixed at:** 2026-07-16T01:58:51Z
**Source review:** .planning/phases/02-status-content-git-worktree-awareness/02-REVIEW-2.md
**Iteration:** 2

**Summary:**
- Findings in scope: 5 (1 Critical, 4 Warnings)
- Fixed: 5
- Skipped: 0

Every fix below is committed atomically and was verified against the
exact mutation the reviewer used to prove the original bug — each new/
changed test was confirmed to FAIL when the corresponding fix is reverted,
and to PASS with the fix applied. `go build ./...`, `go vet ./...`,
`go test ./...` (all non-daemon packages + `internal/daemon` isolated),
and `go test ./testdata/golden/...` are all green. `go.mod`/`go.sum` are
unchanged. No `Marshal*JSON` body was touched. `internal/cli/query_cli_test.go`
is untouched and green (`--json` stays notice-free on all 9 read commands).

## Fixed Issues

### BL-01 (Critical): cancelled MCP tool call permanently disabling the worktree notice

**Files modified:** `internal/gitmeta/cache.go`, `internal/gitmeta/cache_test.go`, `internal/mcp/markdown_test.go`, `internal/query/engine.go`
**Commit:** `d5bf263`
**Applied fix:** `CachingDetector.Detect` now checks `ctx.Err()` after computing (but
before caching) a verdict: if the context was cancelled, the degraded
per-call verdict is still returned (WORK-03's "never block/error a read"
contract is preserved) but is **not** written into the cache, so the next
healthy call re-probes instead of inheriting a permanent false "clean"
verdict. Added a unit-level regression test
(`TestCachingDetectorCancelledContextNotCached`) plus the review's
required acceptance-bar test — a real `BuildServer` → `CallTool` probe
(`TestCancelledCallDoesNotPoisonNoticeForSubsequentCalls`) that issues one
call with an already-cancelled context and asserts a subsequent healthy
call on the *same* server still emits the notice. Both tests were verified
to fail under the exact reverted mutation, reproducing the reviewer's
proof (`main` content returned with no worktree warning). A doc-comment
note was added to `Engine.WorktreeMismatch` explaining why the parallel
per-Engine `mismatchOnce` latch is safe (MCP builds a fresh `Engine` per
call) and why the CLI is unaffected (`cmd.Context()` is always
`context.Background()` today, per IN-01).

### WR-01: `deriveServeRepoPath` did not close the gap it claimed to close

**Files modified:** `internal/cli/serve.go`, `internal/cli/serve_test.go`, `internal/mcp/markdown_test.go` (doc-comment correction only, folded into the WR-02 commit)
**Commit:** `ac3b16a` (WR-02's doc-comment correction to `deriveServeRepoPath` landed in `0351901`)
**Applied fix:** Extracted `serveServerPaths(start)` from `newServeCmd`'s
`RunE` — the exact `repoPath`/`hasIndex` derivation CR-01 fixes — as its
own package-level function, and updated `RunE` to call it. Added
`internal/cli/serve_test.go` with
`TestServeKeepsStartPathDistinctFromConfinementRoot`, which calls
`serveServerPaths` directly (using the existing
`statusWorktreeMismatchFixture`) and asserts `repoPath != start` and
`repoPath == main`. This is the first test that observes serve.go's own
wiring rather than a hand-built replica; verified to fail under the exact
root-cause mutation the reviewer used
(`BuildServer(..., repoPath, repoPath)`-equivalent collapse). Also
corrected `deriveServeRepoPath`'s doc comment in `internal/mcp`, which had
overclaimed that routing fixtures through it closed CR-01's test gap — it
now documents that it is a same-package fixture-input replica only, and
points at `serve_test.go` for the real proof.

### WR-02: confinement guard could not distinguish which of CR-01's two params it anchors on

**Files modified:** `internal/mcp/server_test.go`, `internal/mcp/markdown_test.go`
**Commit:** `0351901`
**Applied fix:** Added `TestConfinementAnchoredOnRepoRootNotStartPath`,
which builds the server in the linked-worktree shape production actually
produces (`startPath != repoPath`, via `mcpWorktreeMismatchFixture` +
`deriveServeRepoPath`) and probes a path that is *inside* `repoPath` but
*outside* `startPath` — a request that only resolves correctly if
`confineToRepoRoot` anchors on `repoPath`. Verified to fail under the
exact "anchor on `startPath`" mutation
(`confineToRepoRoot(path, defaultPath)`), which the prior
`TestOpenEnginePathConfinedToRepoRoot` (degenerate `startPath == repoPath`
configuration) could not catch.

### WR-03 / CR-02-REGRESSION: golden fixture copies inherited pollution from the source tree's own `.codegraph/`

**Files modified:** `testdata/golden/golden_parity_test.go`
**Commit:** `8afd885`
**Applied fix:** `copyDir` now skips any `.codegraph` directory entirely
(`fs.SkipDir`) rather than copying it verbatim, so `buildIndexedFixture`'s
fresh index can never merge into an inherited store — this was not
hypothetical: `testdata/golden/corpus/synthetic-parity/src/.codegraph`
exists on this checkout today (untracked, 28 KB, a stray local `codegraph
init`). Added `TestBuildIndexedFixtureIgnoresInheritedStore`, which plants
a populated, unrelated store at a 1-file source tree's `.codegraph/store`
and asserts the built fixture reports `fileCount == 1`; verified to fail
under the exact "copy `.codegraph` verbatim" mutation (reproducing the
reviewer's `fileCount=69, nodeCount=762` proof shape).

### WR-04: `BuildServer`'s internal call order inverted relative to its own signature

**Files modified:** `internal/mcp/server.go`, `internal/mcp/tools.go`
**Commit:** `5359d41`
**Applied fix:** Took the "at minimum" remediation from the review rather
than the full `ServerPaths` struct option, to keep the fix's blast radius
inside `internal/mcp` (no `BuildServer` call-site changes across its 17
callers): `exploreHandler` and `companionHandler` now both take
`(repoPath, defaultPath, ...)`, matching `BuildServer`'s own
`(repoPath, startPath)` declared parameter order, so both call sites in
`BuildServer` read the same way the signature does. No behavior change —
`confineToRepoRoot` and `openEngine` bodies were not touched, only the
two wrapper functions' parameter order and doc comments. Full `internal/mcp`
and golden suites re-verified green after the reorder (including WR-02's
new confinement test and BL-01's cancelled-call probe, both of which would
catch a real swap here).

### GOLDEN-01 (raised inside WR-01's finding text): `go test ./...` never runs the golden suite

**Files modified:** `.github/workflows/ci.yml`, `testdata/golden/README.md`
**Commit:** `157319a`
**Applied fix:** Confirmed empirically (`go list ./...` returns zero
packages under `testdata/golden`) that every "full suite green" signal in
this phase's history silently excluded the golden parity suite. Added an
explicit `go test ./testdata/golden/...` step to `ci.yml`'s `test` job
(separate from the `go list ./...`-driven step, so this suite's coverage
cannot silently regress back to zero), and documented the exclusion
prominently at the top of `testdata/golden/README.md`.

## Skipped Issues

None — all 5 in-scope findings (plus the GOLDEN-01 sub-item) were fixed
and verified.

## Verification

```
go build ./...                                    # clean
go vet ./...                                       # clean
go test $(go list ./... | grep -v /internal/daemon) # all green
go test ./internal/daemon/ -count=1                 # green (isolated, per CI convention)
go test ./testdata/golden/...                       # green (explicitly invoked — see GOLDEN-01)
git diff --stat <base> -- go.mod go.sum             # empty
```

Each of the 6 fix commits was individually verified to make its
corresponding new/changed test FAIL when reverted (mutation-tested against
the reviewer's own reproduction shapes) before being restored and
committed.

---

_Fixed: 2026-07-16T01:58:51Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
