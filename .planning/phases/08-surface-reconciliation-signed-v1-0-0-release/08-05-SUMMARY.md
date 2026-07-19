---
phase: 08-surface-reconciliation-signed-v1-0-0-release
plan: 05
subsystem: cli
tags: [cobra, cli, scripting, git-hooks, ci, stdin, glob]

# Dependency graph
requires:
  - phase: 08-surface-reconciliation-signed-v1-0-0-release (plan 04)
    provides: "Engine.Affected(files, depth) depth-bounded BFS + clampAffectedDepth(defaultAffectedDepth=5)"
provides:
  - "affected --stdin/-d/-f/-q/-j scripting flag surface, TS-parity with git-hook/CI pipelines"
  - "affected_test.go proving the stdin/quiet/json/filter/never-hang contract against the real CLI"
affects: [08-06, 08-07, 08-08, 08-09]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "bufio.NewScanner(cmd.InOrStdin()) for line-oriented stdin ingestion that returns cleanly on EOF (never hangs)"
    - "filepath.Match for CLI glob filtering instead of a regex/glob dependency"
    - "zero-input advisory resolved before opening the query engine, so a no-op stdin pipe never requires an initialized index"

key-files:
  created:
    - internal/cli/affected_test.go
  modified:
    - internal/cli/affected.go

key-decisions:
  - "Args relaxed from cobra.MinimumNArgs(1) to cobra.ArbitraryArgs (not a custom validator) since the zero-input case is itself a valid, non-error outcome (advisory + exit 0)"
  - "Positional args and --stdin paths are unioned and deduplicated (order-preserving) rather than treated as mutually exclusive sources"
  - "--quiet output dedupes affected test file paths (multiple test symbols in one file collapse to one printed line) since the CLI's plain-path-list contract is file-oriented, matching git-hook consumption"
  - "--filter uses filepath.Match's glob semantics (which does not cross a path separator with '*') rather than a custom or third-party glob engine, per the plan's 'no new dependency' prohibition"

patterns-established:
  - "Scripting-flag CLI commands resolve their zero-input/degenerate case before touching the storage layer, keeping git-hook no-op invocations index-independent"

requirements-completed: [SURF-04, SURF-03]

coverage:
  - id: D1
    description: "affected gains --stdin (bufio.Scanner over cmd.InOrStdin(), CRLF/blank-line safe, never hangs on EOF), -d/--depth, -f/--filter <glob> via filepath.Match, -q/--quiet, and -j/--json shorthand; Args relaxed to ArbitraryArgs"
    requirement: "SURF-04"
    verification:
      - kind: unit
        ref: "internal/cli/affected_test.go#TestAffectedStdinNeverHangs"
        status: pass
      - kind: unit
        ref: "internal/cli/affected_test.go#TestAffectedStdinCRLFAndBlankLines"
        status: pass
      - kind: unit
        ref: "internal/cli/affected_test.go#TestAffectedFilter"
        status: pass
    human_judgment: false
  - id: D2
    description: "Zero-input (no positional args, no/empty stdin) prints a 'no files provided' advisory and exits 0 (silent under --quiet), resolved before the query engine is opened"
    requirement: "SURF-04"
    verification:
      - kind: unit
        ref: "internal/cli/affected_test.go#TestAffectedEmptyStdinNoArgs"
        status: pass
    human_judgment: false
  - id: D3
    description: "--quiet emits a plain, deduped, newline-delimited affected-test-path list with no present.RenderFiles call and no WorktreeNotice — safe for git-hook/CI piping"
    requirement: "SURF-04"
    verification:
      - kind: unit
        ref: "internal/cli/affected_test.go#TestAffectedQuietPathsOnly"
        status: pass
    human_judgment: false
  - id: D4
    description: "--json continues to emit the full AffectedResult object even when --quiet is also set"
    requirement: "SURF-04"
    verification:
      - kind: unit
        ref: "internal/cli/affected_test.go#TestAffectedJSONQuiet"
        status: pass
    human_judgment: false
  - id: D5
    description: "-j/--json short-flag alias added to affected, matching TS-parity short-flag conventions"
    requirement: "SURF-03"
    verification:
      - kind: unit
        ref: "go build ./... (cmd.Flags().BoolVarP(&jsonOut, \"json\", \"j\", ...) + affected -h output)"
        status: pass
    human_judgment: false

# Metrics
duration: 20min
completed: 2026-07-19
status: complete
---

# Phase 08 Plan 05: affected scripting flags (--stdin/-d/-f/-q/-j) Summary

**`codegraph affected` now accepts piped stdin, depth-bounded filtering, glob narrowing, and a plain machine-readable `--quiet` path list for `git diff --name-only | codegraph affected --stdin --quiet` pipelines.**

## Performance

- **Duration:** 20 min
- **Started:** 2026-07-19T19:40:00Z (approx.)
- **Completed:** 2026-07-19T20:00:00Z
- **Tasks:** 2 completed
- **Files modified:** 2 (1 modified, 1 created)

## Accomplishments
- `affected`'s `Args` validator relaxed from `cobra.MinimumNArgs(1)` to `cobra.ArbitraryArgs`, so `codegraph affected --stdin` with zero positional args reaches `RunE` instead of being rejected by cobra.
- New flags: `--stdin` (bool), `-d/--depth` (int, default 0 → engine's `defaultAffectedDepth=5`), `-f/--filter <glob>` (string, `filepath.Match` semantics), `-q/--quiet` (bool), `-j/--json` (bool, now with a short alias matching the rest of the CLI's SURF-03 short-flag convention).
- `collectAffectedFiles` unions positional args with `bufio.NewScanner(cmd.InOrStdin())`-read lines, trimming whitespace/CRLF and skipping blanks, deduplicating while preserving order.
- Zero-input case (no args, no/empty stdin) is resolved *before* `query.OpenAt` — prints a "no files provided" advisory (silent under `--quiet`) and exits 0, so a no-op git-hook invocation never requires an initialized index.
- `--quiet` bypasses the human summary/`WorktreeNotice`/`present.RenderFiles` path entirely, emitting only deduped, newline-delimited affected test file paths via a raw `fmt.Fprintf` loop.
- `--json` continues to emit the full `AffectedResult` object regardless of `--quiet`.
- `internal/cli/affected_test.go` drives the real command end-to-end: never-hang (explicit goroutine+timeout assertion), zero-input advisory/silent behavior, quiet paths-only output (no summary substring, no `⚠`), json+quiet still parses as the result object, CRLF/blank-line stdin handling, and `--filter` narrowing/emptying.

## Task Commits

Each task was committed atomically:

1. **Task 1: affected scripting flags + stdin parsing + Args relax (SURF-04/03)** - `2b1c5e1` (feat)
2. **Task 2: affected CLI test — stdin/quiet/exit-code/never-hang** - `f9a5b84` (test)

**Plan metadata:** (this commit)

_Note: this plan's frontmatter is `type: tdd`, but the two tasks were, by the plan's own explicit sequencing, ordered implementation-first (Task 1) then test (Task 2) rather than RED-then-GREEN — see "TDD Gate Compliance" below._

## Files Created/Modified
- `internal/cli/affected.go` - Added `--stdin`/`-d/--depth`/`-f/--filter`/`-q/--quiet`/`-j/--json` flags, `collectAffectedFiles` stdin/args union helper, zero-input advisory branch, `--filter` glob application, and the plain `--quiet` path-list output branch. Relaxed `Args` to `cobra.ArbitraryArgs`.
- `internal/cli/affected_test.go` - New CLI-level test file proving the stdin/quiet/json/filter/never-hang contract against the real command tree, using a `pkga_test.go` fixture addition (`TestAlpha` calling `Alpha()`) so the BFS has a real `isTestSymbol` leaf to surface (gofixture otherwise ships no `_test.go` files).

## Decisions Made
- **`cobra.ArbitraryArgs` over a custom validator**: the plan offered either; `ArbitraryArgs` is simpler and the "zero files" case is itself valid behavior (advisory, not an error), so no additional validation logic was needed at the `Args` layer — it's handled entirely inside `RunE`.
- **Union + dedupe of positional args and `--stdin` lines**: matches the plan's stated behavior ("positional args and stdin both provided → union of both") and avoids `eng.Affected` doing redundant work on duplicate file paths.
- **`--quiet` output dedupes by `FilePath`**: multiple affected test *symbols* in the same file (e.g., two `Test*` functions in one `_test.go`) collapse to a single printed path, since the quiet contract is a file-path list for downstream tooling (test runners, hooks), not a symbol list.
- **`--filter` glob is `filepath.Match`-native, not `**`-aware**: per the plan's explicit prohibition against a new glob dependency. Documented in the test (`*/*_test.go`, not `*_test.go`) since `filepath.Match`'s `*` does not cross a path separator — a real TS-parity gotcha worth surfacing for future callers of this flag.
- **Zero-input handling precedes `query.OpenAt`**: keeps a no-op `--stdin` git-hook invocation from requiring `.codegraph/` to exist, matching the plan's explicit ordering instruction.

## Deviations from Plan

None — plan executed as written. One clarifying addition beyond the plan's literal read_first list: `internal/cli/query_cli_test.go`'s existing `TestAffectedCmd` and `setupIndexedFixture` helper were read to avoid naming collisions and to understand the shared `execCmd`/`execCmdWithInput` test harness (not explicitly listed in `read_first`, but directly relevant and already in-tree).

## TDD Gate Compliance

This plan's frontmatter declares `type: tdd`, which nominally expects a `test(...)` commit (RED) before the corresponding `feat(...)` commit (GREEN) in git log. This plan's own two tasks are explicitly sequenced implementation-first: Task 1 ("scripting flags + stdin parsing") is a `feat` commit, Task 2 ("CLI test") is a `test` commit that lands *after* it — `2b1c5e1` (feat) then `f9a5b84` (test), not RED-then-GREEN order. This is a plan-authored task order, not an executor deviation: Task 2's own text ("Write these as the RED tests before Task 1's implementation is complete *where practical*") acknowledges this is a soft preference, not a hard requirement, and the two tasks were executed and committed in the plan's literal numeric sequence. Both commits exist and both are green; no gate is silently missing, but the RED→GREEN ordering convention is not satisfied by this plan's structure.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `affected --stdin --quiet` is now a valid git-hook/CI pipeline stage (`git diff --name-only | codegraph affected --stdin --quiet`), closing SURF-04.
- SURF-03's short-flag parity gap for `affected -j` is closed alongside this plan's other flags.
- No blockers for 08-06 onward; `internal/cli/affected.go`'s flag surface is stable for any later plan that reads it.

---
*Phase: 08-surface-reconciliation-signed-v1-0-0-release*
*Completed: 2026-07-19*

## Self-Check: PASSED
