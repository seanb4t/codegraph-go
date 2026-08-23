---
phase: 01-engine-seam-wire-protocol-secure-transport
plan: 05
subsystem: api
tags: [go, refactor, seam-extraction, explore-result, typed-errors, mutation-testing, byte-identity]

# Dependency graph
requires:
  - phase: 01-engine-seam-wire-protocol-secure-transport (plan 01-02)
    provides: "TestGoldensMatchLiveEngineOutput — the byte-identity oracle over all 26 frozen golden pairs, the safety net both this plan's extraction and its D-04 mutation-proof are measured against"
  - phase: 01-engine-seam-wire-protocol-secure-transport (plan 01-04)
    provides: "internal/query/detail.go's NodeDetail sum-type seam, buildReverseAdjacency seam, and the buildNodeDetail/buildSingleDefDetail/buildMultiDefDetail builders this plan's ExploreResult seam is built alongside and whose single-def builder this plan's D-04 mutation-proof targets"
provides:
  - "internal/query.ExploreFileGroup / internal/query.ExploreBlast — the former exploreFileGroup/exploreBlast, exported package-wide so internal/uiserver can name them"
  - "internal/query.ExploreResult — Query/Empty/Stale/Groups/SymbolCount/Blasts/Sources/SkeletonFiles, the ENG-02 twin of NodeDetail"
  - "(*Engine).ExploreDetail(query string, maxFiles int) (ExploreResult, error) — the exported ENG-02 seam, a direct call to buildExploreResult"
  - "internal/query/errors.go — ErrNotFound, ErrInvalidArgument, the message-preserving classifiedError carrier, and the notFoundf/invalidArgumentf constructors"
  - "20 caller-reachable error sites across internal/query converted to classified errors with byte-identical messages"
  - "A recorded D-04 mutation-proof over both the NodeDetail (01-04) and ExploreResult (01-05) extracted builders, with counts and named failing sets"
affects: ["01-09 and later UI-view plans that map ExploreResult to uiv1 wire messages and classify Engine errors onto Connect codes via errors.Is"]

# Actuals (#2632)
actuals:
  tokens: 18800
  tasks: 3
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ExploreResult mirrors NodeDetail's D-02 sum-type-seam pattern but with a boolean Empty mode flag instead of a three-way enum, since Explore() has exactly two render shapes (empty vs populated) rather than Node()'s three"
    - "classifiedError: an unexported error carrier storing {class error, msg string}, implementing Error() by returning msg verbatim and Is(target) by comparing target against class — classification without ever changing what a caller reads, and without ever wrapping with %w"
    - "Invariant-proof subtests for structurally-unreachable defensive branches: rather than forcing an unreachable code path open (which would require weakening unrelated pipeline invariants), the test drives the DOWNSTREAM functions directly with adversarial inputs and asserts they can never produce the state the branch guards against"

key-files:
  created:
    - internal/query/errors.go
    - internal/query/errors_test.go
    - internal/query/detail_external_test.go
  modified:
    - internal/query/detail.go
    - internal/query/detail_test.go
    - internal/query/explore.go
    - internal/query/render_markdown.go
    - internal/query/render_markdown_test.go
    - internal/query/node.go
    - internal/query/files.go
    - internal/query/search.go
    - internal/query/traverse.go
    - internal/query/validate.go

key-decisions:
  - "buildExploreResult was placed in internal/query/detail.go, not internal/query/explore.go, mirroring buildNodeDetail's placement (01-04): detail.go holds the seam's TYPES and BUILDERS for both Node and Explore; explore.go and node.go keep only the thin public-API wrappers plus their own genuinely domain-local helpers (groupMatchesByFile, buildBlastEntry, exploreZeroResult, resolveSourcePath, etc). This is a larger diff to explore.go/detail.go than a same-file extraction would have been, but keeps the two seams structurally parallel."
  - "ExploreDetail (not ExploreResult) is the method name, for the identical reason 01-04 chose NodeDetail over NodeResult: the method name and the type name must never collide at a call site in another package."
  - "The 'engine has no repo root configured for source reads' error in resolveSourcePath was deliberately left UNCLASSIFIED. It is an internal engine-misconfiguration guard (repoRoot never set), not a caller-input mistake — classifying it as ErrInvalidArgument would misrepresent what a Connect client should do with it (retry with different arguments, which cannot fix a missing repoRoot)."
  - "internal/query.ValidateAffectedFiles was deliberately left UNCLASSIFIED. It is exported specifically so internal/cli/affected.go can prefix it with its own \"affected: \" text before Engine.Affected is ever called — it is not reached through any internal/query Engine entry point, so it falls outside this task's 'caller-reachable through the Engine' scope (see the task's own read_first: 'internal/cli/serve.go's own stated reason for classifying by sentinel')."
  - "resolveNodeForDetail's \"query: symbol %q not found in file %q\" error (node.go) was NOT converted: when it fires, buildNodeDetail silently swallows it and falls through to the general enumerateSymbolDefs path (`if node, err := e.resolveNodeForDetail(symbol, file); err == nil { ... }` — the err branch is never taken). It is not caller-reachable."

patterns-established:
  - "Pattern: when a task's action text and files_modified were written against an assumption an earlier plan in the SAME wave already invalidated (here: task 2's plan text assumed the argument/not-found errors still lived in node.go, but 01-04 had already moved them into detail.go), convert the site anyway at its ACTUAL current location and record the file-list drift as a Rule 3 finding rather than skip the conversion because the named file doesn't contain it."
  - "Pattern: proving a defensive early-return branch dead, not reachable, is itself a valid and required finding for a D-02 'no missed branch' claim — traced via direct code-path analysis of the downstream functions' own invariants (fileRelevanceGate's 'never prunes below 2' fallback, fiveTierFileSort's stable-sort-never-drops property, groupMatchesByFile's maxFiles>=1 floor), then locked in place by a test that exercises those functions directly with adversarial inputs and asserts the invariant, rather than smoothing over the branch's own count as 'reached' when it provably cannot be."

requirements-completed: [ENG-02, ENG-01]

coverage:
  - id: D1
    description: "ExploreFileGroup/ExploreBlast exported package-wide; ExploreResult extracted covering the populated case and all five zero-match early returns; Explore() reduced to a thin wrapper; ExploreDetail exported as the ENG-02 seam; all 26 frozen goldens still match byte-for-byte"
    requirement: "ENG-02"
    verification:
      - kind: unit
        ref: "internal/query/detail_test.go#TestExploreResultZeroMatchModeCoversEveryBranch"
        status: pass
      - kind: unit
        ref: "internal/query/detail_test.go#TestExploreResultMaxFilesBoundary"
        status: pass
      - kind: unit
        ref: "internal/query/detail_test.go#TestExploreResultCarriesRenderInputsUnchanged"
        status: pass
      - kind: unit
        ref: "internal/query/detail_external_test.go#TestExploreResultComponentTypesAreExported"
        status: pass
      - kind: integration
        ref: "testdata/golden/byte_identity_test.go#TestGoldensMatchLiveEngineOutput"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every caller-reachable argument rejection and not-found error in internal/query is classifiable via errors.Is against ErrInvalidArgument/ErrNotFound, with byte-identical messages; the classification table is counted so it cannot silently shrink"
    requirement: "ENG-01"
    verification:
      - kind: unit
        ref: "internal/query/errors_test.go#TestClassifiedErrorsPreserveTheirMessages"
        status: pass
      - kind: unit
        ref: "internal/query/errors_test.go#TestEveryReachableErrorIsClassified"
        status: pass
    human_judgment: false
  - id: D3
    description: "Both extracted seams (01-04's NodeDetail single-def builder, 01-05's ExploreResult builder) proven safe by a deliberate mutation watched fail against the byte-identity oracle, with counts and named failing sets recorded and both mutations reverted via exact reverse patch without collateral"
    verification:
      - kind: manual_procedural
        ref: "01-05-SUMMARY.md#task-3-d-04-mutation-proof-full-transcript (this document, below)"
        status: pass
    human_judgment: true
    rationale: "This is D-04's own scoped exception to the pass/fail conjunction (see 01-CONTEXT.md and this plan's Gate conventions): the deliberate-mutation runs are EXPECTED to exit non-zero and are scored by --- FAIL/--- PASS counts and a named failing set, not by a go test status. There is no single automated pass/fail signal that captures 'the mutation-proof procedure was carried out correctly and both reverts left the tree clean' — that is a review-time judgment over the recorded transcript below, which is why this deliverable is form A3 (observation) per the plan's own sixth-prohibition amendment, not a countable PASS-line leg."

# Metrics
duration: ~90min
completed: 2026-08-23
status: complete
---

# Phase 1 Plan 5: ExploreResult Seam, Typed Errors & D-04 Mutation-Proof Summary

**Extracted `ExploreResult` (ENG-02's twin of plan 01-04's `NodeDetail`), exported every reachable component type, converted 20 caller-reachable errors to classifiable sentinels with unchanged message bytes, and proved both extracted seams safe by watching the byte-identity oracle fail against a deliberate break in each — all 26 frozen goldens stay byte-identical throughout.**

## Performance

- **Duration:** ~90 min
- **Completed:** 2026-08-23
- **Tasks:** 3
- **Files modified:** 13 (3 created, 10 modified)

## Accomplishments

- Renamed `exploreFileGroup`/`exploreBlast` to `ExploreFileGroup`/`ExploreBlast` package-wide (explore.go, render_markdown.go, render_markdown_test.go) — nameable from `internal/uiserver` from Phase 3 onward, proven by an external-package compile test.
- Extracted `buildExploreResult` (internal/query/detail.go) from `Explore()`'s ~350-line pipeline, funneling all FIVE early "no results" returns through the same `ExploreResult{Empty: true}` representation instead of a pre-rendered string escaping the builder — enumerated, tested, and recorded below.
- Exported `(*Engine).ExploreDetail(query string, maxFiles int) (ExploreResult, error)` as the direct ENG-02 seam and reduced `Explore()` to a thin wrapper (`buildExploreResult` → `exploreZeroResult` or `RenderExplore`).
- Built `internal/query/errors.go`: `ErrNotFound`/`ErrInvalidArgument` sentinels and the message-preserving `classifiedError` carrier, then converted 20 caller-reachable error-returning statements across `detail.go`, `node.go`, `traverse.go`, `validate.go`, `files.go` and `search.go` — every message byte captured live from the pre-conversion tree and asserted unchanged.
- Performed the D-04 mutation-proof over BOTH extracted seams: `buildSingleDefDetail` (01-04, drop `CalledBy`) and `buildExploreResult` (01-05, drop the blasts assembly) — each deliberately broken, the oracle watched go RED with counts and named failing sets recorded, both reverted via an exact `git apply -R` reverse patch (never `git checkout --`), and the oracle re-confirmed at 26/26/26.
- Re-ran the byte-identity oracle after every task: `attempted=26 completed=26 matched=26` throughout — before Task 1, after Task 1, after Task 2, and after both Task 3 restores.

## Task Commits

1. **Task 1: Export the component types and extract `ExploreResult`, including every zero-match branch** - `a17c847` (feat)
2. **Task 2: Classifiable errors with byte-identical messages** - `81d4cfe` (test)
3. **Task 3: D-04 mutation-proof over both extracted seams, with the counts and failing sets recorded** - no commit (see "Task 3 commit note" below)

**Plan metadata:** (this commit)

### Task 3 commit note

Task 3's deliverable is a recorded observation, not a code change: both deliberate mutations were reverted via exact reverse patch before this task ended, and `git status --porcelain internal/query/` was empty at every checkpoint. There is nothing to `git add` for this task — its evidence lives entirely in this SUMMARY's "Task 3: D-04 mutation-proof full transcript" section below, which the final docs commit persists.

## Files Created/Modified

- `internal/query/errors.go` (created) - `ErrNotFound`, `ErrInvalidArgument`, `classifiedError`, `notFoundf`, `invalidArgumentf`
- `internal/query/errors_test.go` (created) - `TestClassifiedErrorsPreserveTheirMessages` (19 rows), `TestEveryReachableErrorIsClassified` (15 rows)
- `internal/query/detail_external_test.go` (created) - `TestExploreResultComponentTypesAreExported` (package `query_test`)
- `internal/query/detail.go` - `ExploreResult`, `buildExploreResult`, `(*Engine).ExploreDetail`; converted `buildNodeDetail`'s argument/not-found errors and `buildExploreResult`'s empty-query error to classified errors
- `internal/query/detail_test.go` - `TestExploreResultZeroMatchModeCoversEveryBranch`, `TestExploreResultMaxFilesBoundary`, `TestExploreResultCarriesRenderInputsUnchanged`, plus the `zeroMetaFakeReader` fixture
- `internal/query/explore.go` - `ExploreFileGroup`/`ExploreBlast` exported; `Explore()` reduced to a thin wrapper over `buildExploreResult`
- `internal/query/render_markdown.go` - `RenderExplore`/`computeSkeletonFiles`/`renderBlastBullet` signatures updated to the exported types
- `internal/query/render_markdown_test.go` - compile-break fix from the type rename (Rule 3)
- `internal/query/node.go` - `resolveSourcePath`'s four caller-facing rejections converted to `invalidArgumentf`
- `internal/query/traverse.go` - `resolveSymbolNode`'s not-found error converted to `notFoundf`
- `internal/query/validate.go` - `validateLimit`/`validateMaxFiles`/`validateDepth`/`ValidateKind` converted to `invalidArgumentf`
- `internal/query/files.go` - `validateFilesDepth` and `Files`' format rejection converted to `invalidArgumentf`
- `internal/query/search.go` - `Query`/`Search`'s empty-term rejections converted to `invalidArgumentf`

## Decisions Made

See `key-decisions` in frontmatter for the four load-bearing ones (buildExploreResult's placement, the `ExploreDetail` naming, and the two deliberately-unclassified error sites). Additionally:

- **Branches 4 and 5 of `TestExploreResultZeroMatchModeCoversEveryBranch` are invariant-proof subtests, not Explore()-driven byte-identity assertions.** During this task's execution I traced, by direct code reading and empirical probing (documented in full below), that `buildExploreResult`'s branches at `len(fileOrder)==0` (detail.go:540) and `len(groups)==0` (detail.go:626) are **structurally unreachable** given the current implementations of `fileRelevanceGate`/`fiveTierFileSort` (explore_gate.go) and `groupMatchesByFile`/`clampMaxFiles`. Forcing them open would require weakening one of those functions' own documented invariants — an architectural change outside this mechanical extraction's scope (Rule 4 territory, not attempted). Rather than silently accept a subtest that doesn't actually exercise the branch, I wrote the two subtests to assert the INVARIANT directly (`fileRelevanceGate` never empties a non-empty input; `groupMatchesByFile` never empties a non-empty `ranked` when `maxFiles>=1`), and documented the finding explicitly instead of smoothing it over — matching this plan's own stated culture for D-04's node-mutation failing-set comparison ("a difference is a finding to explain, not to smooth over").

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed render_markdown_test.go's compile break from the `exploreFileGroup`/`exploreBlast` rename**
- **Found during:** Task 1
- **Issue:** `render_markdown_test.go` referenced the unexported `exploreFileGroup{...}` and `exploreBlast{...}` composite literals directly (`TestNoCoveringTestsWarning`, `TestSkeletonization`) — 6 use sites. Renaming the types broke compilation.
- **Fix:** Updated all 6 references to `ExploreFileGroup{...}` / `ExploreBlast{...}`.
- **Files modified:** internal/query/render_markdown_test.go (not originally in Task 1's `<files>` list, but required for the package to compile — no other wave-1 plan touches this file)
- **Verification:** `go build ./...` clean; `go vet ./internal/query/...` clean.
- **Committed in:** `a17c847` (Task 1 commit)

**2. [Rule 3 - Blocking] Converted `buildNodeDetail`'s and `buildExploreResult`'s errors in `internal/query/detail.go`, not `node.go`/`explore.go`, per this task's own file list**
- **Found during:** Task 2
- **Issue:** Task 2's action text and `<files>` list (`node.go`, `explore.go`, `search.go`, `traverse.go`, `files.go`, `validate.go`) was written assuming the argument error and not-found error for `Node`/`NodeDetail` still lived in `node.go`, and `Explore`'s empty-query error still lived in `explore.go`. Both had already been moved into `internal/query/detail.go` by 01-04 (`buildNodeDetail`) and by THIS plan's own Task 1 (`buildExploreResult`) before Task 2 ran.
- **Fix:** Converted the sites at their actual current location (`detail.go`) rather than skip them because the named file didn't contain them, since the task's explicit intent ("at minimum the argument error and the not-found error in Node") clearly names these exact errors regardless of which file currently holds them.
- **Files modified:** internal/query/detail.go (added to Task 2's actual touched-file set; not itself declared in Task 2's `<files>`)
- **Verification:** `TestClassifiedErrorsPreserveTheirMessages`/`TestEveryReachableErrorIsClassified` both exercise these exact sites and pass; golden oracle unaffected.
- **Committed in:** `81d4cfe` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 3 — blocking compile/scope-drift fixes required for correctness, not new functionality)
**Impact on plan:** Both were necessary; neither changed any production behavior or message byte. No scope creep.

## Error Classification Site Enumeration (Task 2)

**Total: 20 caller-reachable sites converted.** Every message literal below was captured LIVE from the pre-conversion tree via a temporary scratch test (`TestScratchCaptureErrorLiterals`, run once, then deleted before any site was converted) — not transcribed from this plan.

| # | File:approx-line (pre-conversion) | Function | Class | Message |
|---|---|---|---|---|
| 1 | node.go:38 | `resolveSourcePath` | InvalidArgument | `query: empty file path` |
| 2 | node.go:41 | `resolveSourcePath` | InvalidArgument | `query: absolute path %q is not allowed` |
| 3 | node.go:46 | `resolveSourcePath` (string-level Clean check) | InvalidArgument | `query: path %q escapes the repo root` |
| 4 | node.go:60 | `resolveSourcePath` (Rel check) | InvalidArgument | `query: path %q escapes the repo root` |
| 5 | node.go:75 | `resolveSourcePath` (symlink re-verify) | InvalidArgument | `query: path %q escapes the repo root` |
| 6 | detail.go:194 | `buildNodeDetail` | InvalidArgument | `query: node requires a symbol name or a file path` |
| 7 | detail.go:221 | `buildNodeDetail` | NotFound | `query: symbol %q not found` |
| 8 | detail.go:309 | `buildExploreResult` | InvalidArgument | `query: explore query must not be empty` |
| 9 | traverse.go:226 | `resolveSymbolNode` | NotFound | `query: symbol %q not found` |
| 10 | validate.go:109 | `validateLimit` (negative) | InvalidArgument | `query: limit %d must be non-negative` |
| 11 | validate.go:112 | `validateLimit` (exceeds max) | InvalidArgument | `query: limit %d exceeds maximum %d` |
| 12 | validate.go:121 | `validateMaxFiles` (negative) | InvalidArgument | `query: max-files %d must be non-negative` |
| 13 | validate.go:124 | `validateMaxFiles` (exceeds max) | InvalidArgument | `query: max-files %d exceeds maximum %d` |
| 14 | validate.go:139 | `validateDepth` (negative) | InvalidArgument | `query: depth %d must be non-negative` |
| 15 | validate.go:221 | `ValidateKind` | InvalidArgument | `query: unknown kind %q — allowed kinds: ...` |
| 16 | files.go:88 | `validateFilesDepth` (negative) | InvalidArgument | `query: depth %d must be non-negative` |
| 17 | files.go:91 | `validateFilesDepth` (exceeds max) | InvalidArgument | `query: depth %d exceeds maximum %d` |
| 18 | files.go:145 | `Files` (format rejection) | InvalidArgument | `query: unknown files format %q — allowed: flat, tree` |
| 19 | search.go:122 | `Query` (empty term) | InvalidArgument | `query: search term must not be empty` |
| 20 | search.go:155 | `Search` (empty term) | InvalidArgument | `query: search term must not be empty` |

**Deliberately NOT classified** (see Decisions Made): `resolveSourcePath`'s "engine has no repo root configured for source reads" (internal misconfiguration, not caller input); `ValidateAffectedFiles` (not reached through any Engine entry point); `resolveNodeForDetail`'s "not found in file" error (not caller-reachable — silently swallowed by `buildNodeDetail`'s fallthrough).

**Live-capture transcript** (recorded verbatim, before any conversion):
```
validateLimit(-1) = "query: limit -1 must be non-negative"
validateLimit(MaxLimit+1) = "query: limit 1001 exceeds maximum 1000"
validateMaxFiles(-1) = "query: max-files -1 must be non-negative"
validateMaxFiles(MaxFiles+1) = "query: max-files 1001 exceeds maximum 1000"
validateDepth(-1) = "query: depth -1 must be non-negative"
validateFilesDepth(-1) = "query: depth -1 must be non-negative"
validateFilesDepth(MaxDepth+1) = "query: depth 51 exceeds maximum 50"
ValidateKind(banana) = "query: unknown kind \"banana\" — allowed kinds: constant, file, function, interface, method, package, struct, type_alias, variable"
Files(bogus format) = "query: unknown files format \"bogus\" — allowed: flat, tree"
Query(empty) = "query: search term must not be empty"
Search(empty) = "query: search term must not be empty"
Explore(empty query) = "query: explore query must not be empty"
Callers(not found) = "query: symbol \"nosuchsymbolxyz\" not found"
resolveSourcePath(empty) = "query: empty file path"
resolveSourcePath(absolute) = "query: absolute path \"/etc/passwd\" is not allowed"
resolveSourcePath(escapes) = "query: path \"../outside.txt\" escapes the repo root"
NodeDetail(empty args) = "query: node requires a symbol name or a file path"
NodeDetail(not found) = "query: symbol \"nosuchsymbolxyz\" not found"
```

## ExploreResult Zero-Match Branch Enumeration (Task 1)

**Total: 5 branches**, enumerated top-to-bottom in `buildExploreResult` (`internal/query/detail.go`, line numbers as of the final committed state):

| # | Line | Condition | Named subtest |
|---|---|---|---|
| 1 | 360 | `len(candidates) == 0 && len(seeds.SeedIDs) == 0` | `no-candidates-no-seeds` |
| 2 | 424 | `len(finalNodeIDs) == 0` | `bfs-fully-pruned-no-final-nodes` |
| 3 | 477 | `len(fileScores) == 0` (post H15 hard exclusion) | `hard-test-exclusion-empties-file-scores` |
| 4 | 540 | `len(fileOrder) == 0` (post H17 relevance gate + H18 sort) | `relevance-gate-never-empties-a-non-empty-input` (invariant proof — see below) |
| 5 | 626 | `len(groups) == 0` (post `groupMatchesByFile`) | `group-assembly-never-empties-a-non-empty-ranked-set` (invariant proof — see below) |

A branch count of exactly 5 (never 0 or 1) confirms the function demonstrably has several, per the plan's own review-fail condition.

### Branches 4 and 5: reachability finding

During test authoring I attempted, empirically and by direct code tracing, to reach branches 4 and 5 through `Explore()`'s real public API (real corpora, the local behavioral fixture, and hand-crafted fake-reader graphs). I could not, and traced why:

- **Branch 4 (`len(fileOrder)==0`).** `fileOrder := fiveTierFileSort(gated, ...)` where `gated := fileRelevanceGate(candidatePaths, ...)`. `fileRelevanceGate`'s own documented "never prunes below 2 files" guard (`explore_gate.go`, `fileRelevanceGateMinFiles`) means it returns EITHER the filtered subset (only when that subset has >= 2 members) OR the original `candidatePaths` completely unchanged as a fallback — it can never return fewer than 2 elements from a non-empty input in a way that discards everything, and for `len(candidatePaths) < 2` the same fallback still returns `candidatePaths` verbatim. `fiveTierFileSort` is a pure in-place stable SORT (`out := append([]string(nil), paths...)`; sorts; returns `out`) — it always returns exactly `len(input)` elements. Composed: any non-empty `candidatePaths` produces a non-empty `fileOrder`. And `candidatePaths` is built directly from `fileScores`'s own keys — the SAME `fileScores` branch 3 already required to be non-empty. So branch 4 can only be reached if branch 3 already fired, which is a contradiction (branch 3 returns before this code is ever reached).
- **Branch 5 (`len(groups)==0`).** `groups, _ := groupMatchesByFile(ranked, maxFiles)`. `groupMatchesByFile` only skips a NEW file once `len(groups) >= maxFiles`; since `maxFiles` is floored at 1 by `clampMaxFiles`/`clampExploreBudget` by the time it reaches this call (there is no live code path that leaves it at 0), the FIRST ranked node with a non-empty `FilePath` always seeds a group. And `ranked` is non-empty whenever `fileOrder` is non-empty: for every `f` in `fileOrder`, `nodesByFile[f]` is guaranteed non-empty (both `fileScores` and `nodesByFile` are built from the SAME `finalNodeIDs`/`GetNode` calls with identical skip conditions — `computeFileScoreTiers` cannot score a file that `nodesByFile` doesn't also have an entry for), so the per-file fallback `nodesByFile[f][:1]` always supplies at least one entry to `ranked` even when `matchedByFile[f]` is empty.

**Both branches are therefore dead code today, by construction of `fileRelevanceGate`/`fiveTierFileSort`/`groupMatchesByFile`/`clampMaxFiles`'s own current invariants — not by assumption.** The extraction still funnels them through `ExploreResult{Empty: true}` identically to the other three (mechanical completeness, D-02), and their corresponding subtests assert the INVARIANT that makes them unreachable rather than driving `Explore()` through a state nothing can reach. This is recorded here rather than smoothed over, matching the plan's own stated standard for D-04's failing-set comparison.

## `maxFiles` Boundary (Task 1)

`TestExploreResultMaxFilesBoundary` used the local behavioral corpus, query `"account balance"`, which produces >= 2 file groups (verified: this query is the same one `TestExploreStructuralBeatsLexical`, explore_test.go, already exercises for its RWR-structural-bridge property).

- `exactly-at-match-count`: `maxFiles == n` (the full untruncated group count) includes every group.
- `one-below`: `maxFiles == n-1` drops exactly one group.
- `zero-is-unchanged`: `maxFiles == 0` was compared against an INDEPENDENTLY-COMPUTED value — `clampExploreBudget(getExploreOutputBudget(e.countIndexedFiles()))`, the exact H21 formula `buildExploreResult` itself calls — rather than a hardcoded constant, so the "behaves exactly as it does today" claim has a failing input if the formula's wiring ever drifts.

## Task 3: D-04 Mutation-Proof Full Transcript

Both mutations below were applied to `internal/query/detail.go` in an already-clean, already-committed working tree (Tasks 1 and 2 were committed before Task 3 began). Each mutation was captured as an exact reverse patch via `git diff > <patch>` IMMEDIATELY after applying it and BEFORE running anything, then reverted with `git apply -R <patch>` — never `git checkout --`. No isolated `git worktree` was used (an exact reverse patch was sufficient and simpler for a single-file, single-hunk mutation).

### Mutation 1 — Node's `buildSingleDefDetail` (targets 01-04's extracted builder)

**Mutation applied:** `buildSingleDefDetail`'s return statement changed from
`return &DefinitionDetail{Node: node, Calls: calls, CalledBy: calledBy}, nil`
to
`_ = calledBy; return &DefinitionDetail{Node: node, Calls: calls}, nil`
— dropping the `CalledBy` field, the exact mutation D-04 names.

**Applied-confirmation:** `git diff --stat internal/query/detail.go` reported `1 file changed, 2 insertions(+), 1 deletion(-)` before running anything.

**Result:** `go test -v -count=1 -run 'TestGoldensMatchLiveEngineOutput$' ./testdata/golden/` exited 1 (non-zero, the expected evidence for this deliberate-mutation observation). `attempted=26 completed=26 matched=24 of 26`. **`--- FAIL` count: 2. `--- PASS` count: 24.**

**Named failing set (2 of 26):**
- `requests/go-node.json` (single-def "Session" — CalledBy line changed from `session (src/requests/sessions.py:908)` to empty)
- `requests/go-node-mcp.json` (same symbol via MCP)

**Comparison against plan 01-02's recorded pre-extraction failing set** (4 pairs: `serilog/go-node-multi.json`, `requests/go-node.json`, `requests/go-node-multi.json`, `requests/go-node-mcp.json`, from mutating BOTH `RenderNode` and `renderNodeSection`): **the sets differ — mine (2) is a proper subset of 01-02's (4).** This is explained, not smoothed over: 01-02's mutation broke BOTH the single-def render path (`RenderNode`) AND the multi-def render path (`renderNodeSection`), while this task's mutation targets ONLY `buildSingleDefDetail` — the single-def GATHER builder. `serilog/go-node-multi.json` and `requests/go-node-multi.json` render through the multi-def path (`buildMultiDefDetail`, a completely separate builder this mutation never touches), so they are correctly unaffected. This is the expected, narrower blast radius of a mutation scoped to exactly one of the two builders 01-04 extracted, and matches the plan's own framing ("the same behavior is being broken through a different code path" — here, through half of that code, by design).

**Restore method:** exact reverse patch (`git diff internal/query/detail.go > mutation1-node-calledby.patch`, then `git apply -R mutation1-node-calledby.patch`).

**Post-restore confirmation:** `test -d internal/query && git status --porcelain internal/query/` printed nothing (clean). Re-running the oracle returned `attempted=26 completed=26 matched=26 of 26`, PASS, 27 `--- PASS` lines.

### Mutation 2 — Explore's `buildExploreResult` (targets THIS plan's own extracted builder)

**Mutation applied:** the blasts-assembly loop's `blasts = append(blasts, bl)` call was removed (the loop still calls `e.buildBlastEntry` for its error-propagation side effect, but never appends the result), so `blasts` stays permanently empty regardless of how many groups/symbols are found — the exact "drop the blasts assembly" mutation the plan names.

**Applied-confirmation:** `git diff --stat internal/query/detail.go` reported `1 file changed, 1 insertion(+), 2 deletions(-)` before running anything.

**Result:** `go test -v -count=1 -run 'TestGoldensMatchLiveEngineOutput$' ./testdata/golden/` exited 1. `attempted=26 completed=26 matched=13 of 26`. **`--- FAIL` count: 13. `--- PASS` count: 13.**

**Named failing set (13 of 26 — every explore-bearing pair, and ONLY explore-bearing pairs):**
`behavioral/go-explore-multi.json`, `hugo/go-explore.json`, `hugo/go-explore-multi.json`, `hugo/go-explore-mcp.json`, `guava/go-explore.json`, `guava/go-explore-multi.json`, `guava/go-explore-mcp.json`, `serilog/go-explore.json`, `serilog/go-explore-multi.json`, `serilog/go-explore-mcp.json`, `requests/go-explore.json`, `requests/go-explore-multi.json`, `requests/go-explore-mcp.json`.

**Measured against the plan's expectation** ("the expected failing set here is the explore-bearing pairs"): **exact match, zero survivors.** The 26-pair corpus contains exactly 13 explore-bearing pairs (4 corpora × 3 explore slugs = 12, plus `behavioral/go-explore-multi.json` = 13) and every one of them failed; all 13 node-bearing pairs passed unaffected. No explore-bearing survivor exists, so the plan's "for any explore-bearing pair that survives, open its golden and confirm it contains no blast-radius section" clause does not apply — there is nothing to check.

**Restore method:** exact reverse patch (`git diff internal/query/detail.go > mutation2-explore-blasts.patch`, then `git apply -R mutation2-explore-blasts.patch`).

**Post-restore confirmation:** `test -d internal/query && git status --porcelain internal/query/` printed nothing (clean). Re-running the oracle returned `attempted=26 completed=26 matched=26 of 26`, PASS.

### Final restored-tree gate (this task's own `<automated>` verify)

`test -d internal/query && test -z "$(git status --porcelain internal/query/)"` passed (existence-guarded, non-vacuous), then `go test -v -count=1 -run 'TestGoldensMatchLiveEngineOutput$' ./testdata/golden/` exited **0** with **27** `--- PASS` lines (1 parent + 26 per-pair subtests) — meeting the floor exactly, as designed (zero headroom: a dropped or unrestored pair fails this gate). `task test:unit` and `task test:golden` both pass across the whole repo.

Both `--- FAIL` counts (2 and 13) are greater than zero — neither observation is vacuous; the STOP CONDITION named in the plan's acceptance criteria does not apply to either.

## Issues Encountered

None beyond the branches-4/5 unreachability investigation and the two Rule-3 deviations, both documented above and both resolved during execution.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 01-09 now has a real `ExploreResult` seam (mirroring 01-04's `NodeDetail`) to plan the UI RPC layer's explore-detail mapping against — `(*Engine).ExploreDetail` and its exported `ExploreFileGroup`/`ExploreBlast` component types are the exact shapes a wire consumer maps from.
- Plan 01-09's Connect error mapping now has real sentinels (`query.ErrNotFound`, `query.ErrInvalidArgument`) to classify against via `errors.Is` — the RPC layer never needs to match on message text.
- Both extracted seams (01-04's NodeDetail, 01-05's ExploreResult) are proven able to fail under deliberate mutation, with counts and named failing sets recorded — the "goldens stay byte-identical" claim behind ROADMAP success criterion 1 is a demonstrated property, not an assumed one.
- No blockers. All 26 frozen goldens remain byte-identical; `codegraph node`/`codegraph explore`'s CLI and MCP output is unchanged.

---
*Phase: 01-engine-seam-wire-protocol-secure-transport*
*Completed: 2026-08-23*

## Self-Check: PASSED

- FOUND: internal/query/errors.go
- FOUND: internal/query/errors_test.go
- FOUND: internal/query/detail_external_test.go
- FOUND: commit a17c847 (Task 1)
- FOUND: commit 81d4cfe (Task 2)
- Task 3: no commit (nothing to commit — working tree clean, evidence recorded in this SUMMARY)
