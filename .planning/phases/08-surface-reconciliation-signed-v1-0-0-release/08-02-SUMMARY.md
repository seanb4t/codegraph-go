---
phase: 08-surface-reconciliation-signed-v1-0-0-release
plan: 02
subsystem: cli
tags: [cobra, query-engine, files, parity, tdd]

requires:
  - phase: 08-surface-reconciliation-signed-v1-0-0-release
    provides: "08-CONTEXT.md D-03 (add-alongside decision) and 08-RESEARCH.md's confirmed TS files --filter <dir> prefix semantics"
provides:
  - "query.FilesOptions.Dir: a plain path-prefix filter matching TS 1.3.1's files --filter <dir> exactly"
  - "files --dir <prefix> CLI flag, orthogonal to and composing (AND) with the retained language --filter"
  - "files -j short alias for --json (SURF-03)"
affects: [08-05-audit-flag-parity, files-cli, query-engine]

tech-stack:
  added: []
  patterns:
    - "dirPrefixMatches(path, dir) helper: strings.HasPrefix only, no glob/regex — TS parity + anti-ReDoS by construction"

key-files:
  created: []
  modified:
    - internal/query/files.go
    - internal/query/files_status_test.go
    - internal/cli/files.go

key-decisions:
  - "D-03 add-alongside: --dir (directory-path prefix) and --filter (language) are orthogonal dimensions that coexist and compose AND, not a rename/replace of --filter"
  - "--dir matches via strings.HasPrefix only (path or './'+path prefix) — never filepath.Match/doublestar/regexp — mirroring TS's own bin/codegraph.js implementation exactly rather than treating the <dir> placeholder as a glob spec"

patterns-established:
  - "Prefix-match filters (as opposed to glob Pattern) get their own small pure predicate function (dirPrefixMatches) unit-tested independently of the fixture, so edge cases like the './'-prefix branch don't require constructing matching on-disk paths"

requirements-completed: [SURF-02, SURF-03]

coverage:
  - id: D1
    description: "FilesOptions.Dir prefix filter in the query engine, composing AND with the language Filter"
    requirement: SURF-02
    verification:
      - kind: unit
        ref: "internal/query/files_status_test.go#TestFiles/dir_narrows_by_path_prefix"
        status: pass
      - kind: unit
        ref: "internal/query/files_status_test.go#TestFiles/dir_with_zero_matches_returns_empty_result,_not_an_error"
        status: pass
      - kind: unit
        ref: "internal/query/files_status_test.go#TestFiles/dir_composes_with_the_language_filter_(AND)"
        status: pass
      - kind: unit
        ref: "internal/query/files_status_test.go#TestFiles/dirPrefixMatches:_plain_prefix_semantics,_not_a_glob"
        status: pass
    human_judgment: false
  - id: D2
    description: "files --dir CLI flag wired to FilesOptions.Dir; -j short alias for --json"
    requirement: SURF-03
    verification:
      - kind: unit
        ref: "go build ./... && go test ./internal/cli/... -count=1"
        status: pass
      - kind: manual_procedural
        ref: "codegraph files -h shows --dir and -j, --json with no shorthand collision"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-07-19
status: complete
---

# Phase 08 Plan 02: files --dir prefix filter + -j alias Summary

**Added `files --dir <prefix>` as a new plain-string-prefix directory filter (matching TS 1.3.1's `files --filter <dir>` exactly via `strings.HasPrefix`) alongside the existing untouched language `--filter`, plus a `-j` short alias for `--json`.**

## Performance

- **Duration:** 6 min
- **Tasks:** 2 completed
- **Files modified:** 3

## Accomplishments
- `query.FilesOptions.Dir` filters by path prefix (`strings.HasPrefix(path, dir)` or `"./"+dir`), composing AND with the existing language `Filter` — no glob/regex/doublestar introduced
- `files --dir <prefix>` and `files -j`/`--json` wired end-to-end in the CLI; the existing `--filter` (language), `--pattern`, `--depth`, and `--format` are untouched
- TDD RED→GREEN cycle completed for the engine change: a failing compile (missing `Dir` field/`dirPrefixMatches`) committed first, then the minimal implementation to turn it green

## Task Commits

Each task was committed atomically:

1. **Task 1: RED→GREEN — FilesOptions.Dir prefix filter in the engine (SURF-02)**
   - `374fe1d` (test) — RED: dir-narrows/zero-match/compose-with-filter subtests + `dirPrefixMatches` table test; fails to compile
   - `178cfbd` (feat) — GREEN: `FilesOptions.Dir` field, `dirPrefixMatches` helper, wired into `Files()`
2. **Task 2: files --dir + -j flags wired to FilesOptions (SURF-02/03)** - `80c062d` (feat)

_Note: Task 1 is a plan-level `type: tdd` task; both RED and GREEN commits exist in git log as required._

## Files Created/Modified
- `internal/query/files.go` - Added `FilesOptions.Dir` field, `dirPrefixMatches` prefix-match helper, wired into the `Files()` filter pass and doc comments
- `internal/query/files_status_test.go` - Added `TestFiles` subtests: dir-narrows-by-prefix, dir-zero-match, dir-composes-with-language-filter, and a table test for `dirPrefixMatches` itself (including the `./`-prefix branch, which the fixture's own paths don't naturally exercise)
- `internal/cli/files.go` - Added `--dir` `StringVar` flag wired to `query.FilesOptions.Dir`; changed `--json` from `BoolVar` to `BoolVarP` with `-j` short alias; updated doc comment

## Decisions Made
- Followed CONTEXT D-03's locked "add-alongside" decision precisely: `--dir` is a new, orthogonal flag; `--filter` (language) was not renamed, repointed, or altered in any way
- Extracted the prefix-match logic into a standalone `dirPrefixMatches(path, dir string) bool` function rather than inlining it in `Files()`'s filter loop, so the `./`-prefix branch (which none of the `gofixture` test fixture's real paths naturally trigger, since none carry a leading `./`) could be unit-tested directly and deterministically rather than skipped or faked through fixture setup

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

`files --dir` and `-j` are fully wired and tested, ready for 08-05's flag-parity audit to confirm this documented divergence (language `--filter` retained + new `--dir` added) is recorded correctly. No blockers for subsequent phase-08 plans.

---
*Phase: 08-surface-reconciliation-signed-v1-0-0-release*
*Completed: 2026-07-19*

## Self-Check: PASSED
All commits (374fe1d, 178cfbd, 80c062d) and all modified/created files verified present.
