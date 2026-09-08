---
phase: 04-query-workbench-index-health
plan: 02
subsystem: query
tags: [go, glob, doublestar, engine-files, mcp, cli, ui]

# Dependency graph
requires: []
provides:
  - "internal/query.FilesOptions.Pattern matched via github.com/bmatcuk/doublestar/v4 — ** crosses directory separators and matches zero-or-more path segments, {a,b} brace alternation supported"
  - "TestFilesPatternRecursiveGlob (internal/query/files_status_test.go) — the two-direction regression test 04-06 and future Files-pattern work must keep green"
  - "The escaped_metacharacter_is_literal matcher-level contract (doublestar.Match with a backslash-escaped metacharacter) that 04-06's escapeGlobLiteral is written against"
affects: [04-06-file-picker]

# Actuals (#2632)
actuals:
  tokens: 5443
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: ["github.com/bmatcuk/doublestar/v4 v4.10.0"]
  patterns:
    - "Independent-oracle test design: expectedGlobMatches(t, raw, pattern) filters the RAW indexed path set through doublestar.Match directly, so glob-pattern-set assertions are portable across build-tag/platform differences (e.g. skip_linux.go) instead of hardcoding a literal expected-path list."
    - "Instrumented-reader convention (search_test.go's searchFakeReader) reused for globRefusingReader — proves a sanity-check refusal precedes the store scan by making a scan call itself fail the test."

key-files:
  created: []
  modified:
    - internal/query/files.go
    - internal/query/files_status_test.go
    - go.mod
    - go.sum
    - web/src/lib/search.ts
    - .planning/todos/completed/2026-08-29-files-rpc-pattern-glob-cannot-cross-directory-boundaries-for-live-file-search.md

key-decisions:
  - "D-14 (04-CONTEXT.md): fixed the glob matcher itself (doublestar.Match replacing filepath.Match) rather than adding a second Substring-style option to FilesOptions — one mechanism, not two overlapping ones."
  - "go.mod change kept to the single doublestar line: go mod tidy could not run cleanly (pre-existing, unrelated github.com/tree-sitter/tree-sitter-swift/bindings/go module-resolution failure, confirmed identical on a clean checkout via git stash) and go mod tidy -e additionally reclassified three unrelated packages from indirect to direct (also reproduced on a clean checkout) — both are pre-existing repo conditions, out of this plan's scope; the doublestar require was added and promoted from indirect to direct by hand instead, verified by a full go build."
  - "web/src/lib/search.ts's own Files pattern construction (*term*, single-star) is left unchanged by design — Task 3's scope is comment-only. The recursive **/*term* client is 04-06's new file-search.ts, not a retrofit onto this RPC's existing dispatch."

patterns-established:
  - "Fixture-corpus augmentation: copyFixture(t) -> write extra files into the copied temp tree -> indexFixture(t, dir), used when the shared gofixture's depth/language coverage doesn't reach a test's needs."

requirements-completed: [WRK-02]

coverage:
  - id: D1
    description: "Engine.Files with Pattern \"**/*term*\" returns both a nested match (internal/deep/termnested.go) and the pre-existing root-level match (termroot.go) in one call — the root-cause glob bug is closed for all three callers (CLI, MCP, UI) at once."
    requirement: "WRK-02"
    verification:
      - kind: unit
        ref: "internal/query/files_status_test.go#TestFilesPatternRecursiveGlob/nested"
        status: pass
      - kind: unit
        ref: "internal/query/files_status_test.go#TestFilesPatternRecursiveGlob/root_level"
        status: pass
    human_judgment: false
  - id: D2
    description: "Existing non-recursive patterns (*term*) and brace alternation (**/*.{go,ts}) behave identically/as newly documented; malformed-pattern refusal still precedes the store scan; a backslash-escaped glob metacharacter matches literally (the contract 04-06's escapeGlobLiteral depends on)."
    requirement: "WRK-02"
    verification:
      - kind: unit
        ref: "internal/query/files_status_test.go#TestFilesPatternRecursiveGlob/non_recursive_unchanged"
        status: pass
      - kind: unit
        ref: "internal/query/files_status_test.go#TestFilesPatternRecursiveGlob/brace_alternation"
        status: pass
      - kind: unit
        ref: "internal/query/files_status_test.go#TestFilesPatternRecursiveGlob/malformed_refused"
        status: pass
      - kind: unit
        ref: "internal/query/files_status_test.go#TestFilesPatternRecursiveGlob/refusal_precedes_scan"
        status: pass
      - kind: unit
        ref: "internal/query/files_status_test.go#TestFilesPatternRecursiveGlob/escaped_metacharacter_is_literal"
        status: pass
    human_judgment: false
  - id: D3
    description: "No frozen CLI golden or MCP wire-oracle transcript changed as a result of the matcher swap (T-03-14) — the full backend suite, including test/wireoracle, is green with a clean git status."
    verification:
      - kind: unit
        ref: "task test:unit (50/50 packages ok, including test/wireoracle)"
        status: pass
      - kind: other
        ref: "git status --porcelain shows no change under any golden/testdata/transcript path after the full plan"
        status: pass
    human_judgment: false
  - id: D4
    description: "The folded todo is closed with a resolution record (git mv, history preserved) and web/src/lib/search.ts no longer documents the limitation as live."
    verification:
      - kind: unit
        ref: "web/tests/search.test.ts (13/13 pass, byte-unchanged behavior)"
        status: pass
      - kind: other
        ref: "git log --follow -- .planning/todos/completed/2026-08-29-...md shows pre-move history"
        status: pass
    human_judgment: false

# Metrics
duration: ~30min
completed: 2026-08-29
status: complete
---

# Phase 4 Plan 2: Recursive Glob Fix for Engine.Files Summary

**Swapped `internal/query.FilesOptions.Pattern`'s matcher from `path/filepath.Match` to `github.com/bmatcuk/doublestar/v4`, closing the bug where live file search returned nothing for any nested path, for all three `Engine.Files` callers (CLI, MCP, UI) at once.**

## Performance

- **Duration:** ~30 min (commit span 18:22–18:28 local; exploration/verification preceded it — exact session start was not captured)
- **Tasks:** 3
- **Files modified:** 6 (go.mod, go.sum, internal/query/files.go, internal/query/files_status_test.go, web/src/lib/search.ts, one todo file relocated)

## Accomplishments
- `internal/query/files.go`'s two `filepath.Match` call sites (pre-scan sanity check and per-entry match) now both call `doublestar.Match`, so `**` crosses directory boundaries and matches zero-or-more path segments — the same pattern shape now finds both nested and root-level matches
- A committed, two-direction regression test (`TestFilesPatternRecursiveGlob`, 7 subtests) proves the fix and pins the properties it must never regress: non-recursive patterns unchanged, refusal-precedes-scan, and the escape-literal contract 04-06 depends on
- The folded todo describing this exact bug is closed with a resolution record; `web/src/lib/search.ts`'s comment no longer claims the limitation is live

## Task Commits

Each task was committed atomically:

1. **Task 1: Add doublestar, then RED** - `f679b2e1` (test) — vetted the package legitimacy row, added `doublestar/v4 v4.10.0` as a main-module dependency, wrote the two-direction RED test against the untouched `filepath.Match` matcher
2. **Task 2: GREEN — swap in doublestar/v4** - `cad025ea` (feat) — replaced both call sites, updated `FilesOptions.Pattern`'s doc comment, confirmed the full `internal/query`/`internal/cli`/`internal/mcp` suites and the frozen `test/wireoracle` transcripts are unaffected
3. **Task 3: Close the folded todo** - `069d4f1` (docs) — `git mv`'d the todo with a resolution record, rewrote `search.ts`'s stale comment (comment-only diff)

_TDD gate sequence confirmed: `test(04-02)` RED commit precedes `feat(04-02)` GREEN commit._

## Files Created/Modified
- `internal/query/files.go` - both glob-match call sites now use `doublestar.Match`; doc comment updated to state `**`/`{a,b}` semantics
- `internal/query/files_status_test.go` - `TestFilesPatternRecursiveGlob` (7 subtests) plus its `globRefusingReader` instrumented reader and `expectedGlobMatches`/`filesGlobFixture` helpers
- `go.mod` / `go.sum` - `github.com/bmatcuk/doublestar/v4 v4.10.0` added as a direct main-module dependency
- `web/src/lib/search.ts` - comment rewritten to describe the closed root cause; no executable line changed
- `.planning/todos/completed/2026-08-29-files-rpc-pattern-glob-cannot-cross-directory-boundaries-for-live-file-search.md` - relocated from `pending/` (git mv, history preserved) with a resolution record

## Decisions Made
- Fixed the matcher itself (doublestar) rather than adding a second `Substring` option to `FilesOptions`, per 04-CONTEXT.md D-14 — one mechanism, not two overlapping ones.
- Kept `go.mod`'s diff to exactly the one `doublestar` line by hand-promoting it from indirect to direct, rather than running `go mod tidy` to completion — `go mod tidy` fails on a pre-existing, unrelated `tree-sitter-swift` module-resolution error (confirmed identical on a clean checkout via `git stash`), and `go mod tidy -e` additionally reclassifies three unrelated packages (`connectrpc.com/connect`, `github.com/pkg/browser`, `github.com/spf13/pflag`) from indirect to direct (also reproduced on a clean checkout) — both are pre-existing repo conditions out of this plan's scope. Verified correct via a full `go build ./...` and the complete `task test:unit` suite.
- Left `web/src/lib/search.ts`'s own pattern construction (`*term*`, single-star) unchanged by design — Task 3 is a comment-only fix. The recursive `**/*term*` client is 04-06's new `file-search.ts`, not a retrofit onto this RPC's existing dispatch.
- Reused the existing `internal/query.containsPath` helper (`explore_gate_test.go`) rather than redeclaring an identical one — discovered via a build-failure RED-gate run (redeclared in this block), fixed inline (Rule 1).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed a duplicate `containsPath` helper**
- **Found during:** Task 1, first RED-gate run
- **Issue:** The new test file declared its own `containsPath(paths []string, path string) bool`, which collides with an identical helper already declared in `internal/query/explore_gate_test.go` (same package) — a build failure (`containsPath redeclared in this block`), not the intended assertion-level RED.
- **Fix:** Removed the duplicate declaration and reused the existing one.
- **Files modified:** internal/query/files_status_test.go
- **Verification:** `go build ./...` clean; re-ran the RED gate, which then failed on assertions as intended (8 subtest lines, 0 build-failure markers).
- **Committed in:** f679b2e1 (Task 1 commit)

**2. [Rule 3 - Blocking, documentation-only] `go mod tidy` could not complete; go.mod updated by hand**
- **Found during:** Task 1 step (b) and Task 2 step (a)
- **Issue:** `GOTOOLCHAIN=go1.26.5 go mod tidy` fails with `go: finding module for package github.com/tree-sitter/tree-sitter-swift/bindings/go ... module ... does not contain package`. Confirmed via `git stash` that this failure is byte-identical on a clean checkout of this branch, unrelated to doublestar. `go mod tidy -e` "succeeds" but additionally promotes `connectrpc.com/connect`, `github.com/pkg/browser`, and `github.com/spf13/pflag` from indirect to direct — also reproduced on the clean checkout, so this is pre-existing go.mod/go.sum drift, not something this plan introduced or should absorb.
- **Fix:** Ran `go get github.com/bmatcuk/doublestar/v4@v4.10.0` (adds it as `// indirect`), then hand-edited `go.mod` to move that single line into the direct `require` block, alphabetically ordered, leaving every other line untouched. `go.sum` needed no further edit — `go get` had already written correct hash entries.
- **Files modified:** go.mod
- **Verification:** `git diff go.mod` shows exactly one added line; `go build ./...` and the full `task test:unit` (50/50 packages) are green.
- **Committed in:** f679b2e1 (Task 1 commit)

**3. [Documented, no fix — out of scope] `rg -c 'bmatcuk' go.tool-*.mod` reports a pre-existing match**
- **Found during:** Task 1/2 acceptance-criteria verification
- **Issue:** `go.tool-lint.mod` already carries `github.com/bmatcuk/doublestar/v4 v4.10.0 // indirect` (golangci-lint's own transitive dependency), predating this plan — confirmed via `git log --oneline -1 -- go.tool-lint.mod` (commit `82ffd608`, the tool-modfile-isolation commit, long before Phase 4). This plan did not touch `go.tool-lint.mod`; the literal `rg -c` count is non-zero for a reason unrelated to this plan's dependency addition. `TestToolModfilesRemainIsolated`/the isolation invariant is unaffected since nothing in this plan wrote to that file.
- **Fix:** None — pre-existing, out of this plan's file-modification scope (`files_modified` in the plan frontmatter does not include `go.tool-lint.mod`).
- **Files modified:** none
- **Verification:** `git diff --stat go.tool-lint.mod` across all three commits is empty.

**4. [Documented, no fix — out of scope] `CGO_ENABLED=0 go build ./...` fails repo-wide, unrelated to doublestar**
- **Found during:** Task 2 acceptance-criteria verification
- **Issue:** `CGO_ENABLED=0 GOTOOLCHAIN=go1.26.5 go build ./...` fails with `undefined: tree_sitter.Node` in `internal/indexer/goextract` and `internal/indexer/routes` — the project's own justified CGo exception for `tree-sitter/go-tree-sitter` (CLAUDE.md's Parser Decision) means the whole repo, including `internal/query` (which transitively imports `internal/indexer`), has never built with `CGO_ENABLED=0`. Confirmed identical on a clean checkout via `git stash`.
- **Fix:** None — this is a structural, pre-existing property of the repo, not something this plan's change affects. `doublestar` itself was independently confirmed CGo-free per 04-RESEARCH.md's Package Legitimacy Audit (zero transitive deps, `go mod graph` clean) and by this session's own `govulncheck ./...` run finding no doublestar-attributable vulnerability.
- **Files modified:** none

**5. [Documented, empirical correction — no test-design fix needed] The RED-phase split observed 3 failures, not the plan's predicted 2**
- **Found during:** Task 1 step (c), recording the RED output
- **Issue:** The plan's `<behavior>` text states `root_level` ("`**/*term*` ALSO returns `termroot.go`") is "already passes today." Empirically verified (both via a standalone Go program and the actual RED run) that `filepath.Match("**/*term*", "termroot.go")` returns `false` under the pre-fix matcher — the pattern's literal `/` between the two `**` segments and `*term*` cannot be satisfied by a slash-less root-level filename, so this direction fails too, not just `nested`. The observed RED split was: `nested`, `root_level`, `brace_alternation` FAIL; `non_recursive_unchanged`, `malformed_refused`, `refusal_precedes_scan`, `escaped_metacharacter_is_literal` PASS (4 pass / 3 fail, not the plan's predicted 5 pass / 2 fail).
- **Fix:** None to the test itself — `root_level`'s query and assertion are exactly what `<behavior>` specifies, and the automated `<verify>` gate (which only checks that `nested` and `brace_alternation` are the exactly-two subtests matching its named-FAIL regex, not that `root_level` is green) passed unaffected: `subtests=8 build-markers=0 assertion-reds=2 literal-green=1 rc=1`. This is documented here as a correction to the plan's stated assumption, and is itself stronger evidence for the bug: the recursive-glob pattern shape didn't just fail to cross into nested directories under the old matcher, it matched nothing at all, root or nested. After Task 2's fix, all 7 subtests including `root_level` pass, satisfying the plan's `>= 7 --- PASS` GREEN-gate requirement regardless.
- **Files modified:** none (documentation-only correction)

---

**Total deviations:** 5 (1 Rule 1 auto-fix, 1 Rule 3 auto-fix, 3 documented-only findings — 2 pre-existing/out-of-scope, 1 empirical correction to the plan's own stated assumption)
**Impact on plan:** No scope creep. All auto-fixes were mechanical (duplicate-symbol removal, hand-editing one go.mod line). The three documented-only findings are pre-existing repo conditions or a corrected assumption, none of which weakened the plan's actual verification — every automated `<verify>` gate specified in Task 1/2/3 passed as literally written.

## Issues Encountered
None beyond the deviations above — all resolved inline without blocking.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `Engine.Files` now supports recursive glob and brace alternation for every caller (CLI, MCP, UI), unblocking 04-06's multi-file picker, which depends on both the matcher fix and the `escaped_metacharacter_is_literal` contract this plan proved.
- 04-06 should build its `**/*term*`-shaped pattern (with `escapeGlobLiteral`) in a new `web/src/lib/file-search.ts`, not by editing `search.ts`'s existing `*term*` dispatch (left untouched here by design).
- No blockers. `go mod tidy`'s pre-existing `tree-sitter-swift` failure and the repo-wide `CGO_ENABLED=0` build gap are unrelated to this plan and were not introduced or worsened by it.

## Self-Check: PASSED

- `internal/query/files.go` FOUND, contains 2 `doublestar.Match(` calls, 0 remaining `filepath.Match(` calls.
- `internal/query/files_status_test.go` FOUND, contains `TestFilesPatternRecursiveGlob` (exactly 1 declaration).
- `.planning/todos/completed/2026-08-29-files-rpc-pattern-glob-cannot-cross-directory-boundaries-for-live-file-search.md` FOUND; `.planning/todos/pending/2026-08-29-files-rpc-pattern-glob-cannot-cross-directory-boundaries-for-live-file-search.md` confirmed absent.
- Commit `f679b2e1` FOUND in `git log --oneline`.
- Commit `cad025ea` FOUND in `git log --oneline`.
- Commit `069d4f1d` FOUND in `git log --oneline`.
- `go.mod` contains exactly 1 occurrence of `bmatcuk/doublestar/v4`.
- Full `task test:unit` re-run at end of plan: 50/50 packages `ok`, zero `FAIL`.
- `git status --porcelain`: clean.

---
*Phase: 04-query-workbench-index-health*
*Completed: 2026-08-29*
