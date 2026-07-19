---
phase: 08-surface-reconciliation-signed-v1-0-0-release
plan: 01
subsystem: api
tags: [cobra, cli, query-engine, tdd, flag-parity]

# Dependency graph
requires:
  - phase: 07-interactive-tui-daemon-picker-install-multi-select
    provides: stable internal/cli command surface and internal/query.Engine shape to extend
provides:
  - Shared engine impact-depth default now 2 (was 5), inherited by CLI + MCP codegraph_impact together
  - impact command -d/-j short-flag aliases matching TS 1.3.1
  - Confirmation that the golden/behavioral corpus already encodes the depth-2 default (no regen needed)
affects: [08-02-affected-depth-BFS, 08-05-flag-parity-audit, 08-09-drop-in-validation]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Shared-engine default (CLI==MCP), single point of change — internal/query/validate.go's defaultDepth constant, changed once, both surfaces inherit"
    - "RED test hardcodes a literal expected value (not the symbol under test) so a constant-only change produces a genuine failing test before the fix"

key-files:
  created: []
  modified:
    - internal/query/validate.go
    - internal/query/engine_test.go
    - internal/cli/impact.go

key-decisions:
  - "TestClampDepth's default-value cases were rewritten to hardcode a literal 2 (not reference the defaultDepth symbol) so the RED phase actually fails against the pre-change constant — the plan's literal test already used the symbolic constant, which would have made RED impossible without this rewrite"
  - "No golden fixture regeneration needed — testdata/golden/corpus/{weft-go,colbymchenry-codegraph}/impact.json already encoded depth:2 (the TS 1.3.1 oracle default), so the engine change makes Go's no-flag output match rather than diverge"

patterns-established:
  - "Depth-default changes in internal/query land as a single-constant edit with a re-verified grep guard (rg -n \"clampDepth\\(\" internal/query/*.go) confirming no other consumer silently inherits the new default"

requirements-completed: [SURF-01, SURF-03]

coverage:
  - id: D1
    description: "impact's engine-level default depth changed from 5 to 2 (shared internal/query.Engine, so CLI + MCP codegraph_impact move together); MaxDepth=50 ceiling and negative-depth rejection unchanged"
    requirement: "SURF-01"
    verification:
      - kind: unit
        ref: "internal/query/engine_test.go#TestClampDepth"
        status: pass
      - kind: unit
        ref: "internal/query/traverse_test.go#TestImpact"
        status: pass
    human_judgment: false
  - id: D2
    description: "impact CLI command registers -d (alias of --depth) and -j (alias of --json) short flags; --depth help text updated from stale 'default 5' to 'default 2'"
    requirement: "SURF-03"
    verification:
      - kind: unit
        ref: "internal/cli test suite (go test ./internal/cli/... -count=1)"
        status: pass
      - kind: manual_procedural
        ref: "go run ./cmd/codegraph impact -h — confirmed '-d, --depth' and '-j, --json' with no cobra shorthand collision"
        status: pass
    human_judgment: false
  - id: D3
    description: "Golden/behavioral corpus (testdata/golden/corpus/{weft-go,colbymchenry-codegraph}/impact.json) confirmed to already encode depth:2 — no regeneration needed; parity suite green post-change"
    requirement: "SURF-01"
    verification:
      - kind: integration
        ref: "go test ./testdata/golden/... -count=1"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-07-19
status: complete
---

# Phase 8 Plan 1: Engine Impact-Depth Default & Short Flags Summary

**Changed `impact`'s shared-engine default BFS depth from 5 to 2 (matching TS CodeGraph 1.3.1) in a single `internal/query` constant edit, added `-d`/`-j` short-flag aliases to the CLI command, and confirmed the golden corpus already encoded the new default — zero fixture regeneration required.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-19
- **Tasks:** 3 (2 produced commits; Task 3 was verification-only, no diff)
- **Files modified:** 3

## Accomplishments
- `internal/query/validate.go`'s `defaultDepth` constant changed 5→2 (SURF-01/D-02): both the CLI `impact` command and the `codegraph_impact` MCP tool inherit the new default from the same shared engine seam, with `MaxDepth=50` and `validateDepth`'s negative-rejection left untouched.
- `internal/cli/impact.go` now registers `-d`/`-j` as short aliases for `--depth`/`--json` (SURF-03), with the `--depth` help text corrected from a stale "default 5" to "default 2, max 50".
- Verified via `go test ./testdata/golden/... -count=1` that the existing `impact.json` golden fixtures (weft-go, colbymchenry-codegraph) already assert `"depth": 2` — the TS 1.3.1 oracle default — so this change makes Go's no-flag `impact` output match the golden corpus rather than diverge from it.

## Task Commits

Each task was committed atomically:

1. **Task 1a: RED — clampDepth zero-input expects depth-2** - `15aaac3` (test)
2. **Task 1b: GREEN — engine defaultDepth 5->2** - `6fae70f` (feat)
3. **Task 2: impact.go -d/-j short flags + stale help text** - `3c316e4` (feat)
4. **Task 3: golden-corpus parity guard** - no commit (verification-only; `go test ./testdata/golden/... -count=1` confirmed green with zero fixture changes — see Deviations)

**Plan metadata:** (this commit, docs: complete plan)

_Note: Task 1 is a TDD task with a RED→GREEN pair per plan frontmatter `tdd="true"`._

## Files Created/Modified
- `internal/query/validate.go` - `defaultDepth` constant 5→2; doc comment updated to cite SURF-01/D-02
- `internal/query/engine_test.go` - `TestClampDepth`'s default-value cases hardcoded to literal `2` (was symbolic `defaultDepth`); added explicit boundary cases (depth=1, MaxDepth, MaxDepth+1)
- `internal/cli/impact.go` - `--depth`/`--json` flags changed from `IntVar`/`BoolVar` to `IntVarP`/`BoolVarP` registering `-d`/`-j`; `--depth` help text updated

## Decisions Made
- Rewrote `TestClampDepth`'s zero/negative-input expectations to a literal `2` instead of the `defaultDepth` symbol, because the plan's literal RED-test description ("fails against the current `defaultDepth = 5`") is only achievable if the test doesn't reference the constant it's testing — a test that asserts `clampDepth(0) == defaultDepth` passes trivially regardless of the constant's value and can never RED. This is a faithful-to-intent execution of the plan's TDD instruction, not a deviation from it.
- No golden fixture regeneration: Task 3's read-first step and the plan's own acceptance criteria anticipated this outcome ("no regen is expected... fixtures already encode depth:2"), confirmed by running the suite.

## Deviations from Plan

None — plan executed exactly as written, including the anticipated "no regen needed" outcome for Task 3. Task 3 produced no file changes and thus no commit; this matches the plan's `<action>` which frames regeneration as conditional ("If the suite reports an impact mismatch...") and the acceptance criteria's implicit expectation that the fixtures stay untouched.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- SURF-01 (impact depth default) and the `impact`-command slice of SURF-03 (short flags) are complete and green.
- `docs/FLAG-PARITY.md` (SURF-05, a later plan) can record `impact`'s `-d`/`-j` additions and the depth-2 default as "present/matches TS".
- 08-02 (or wherever `affected`'s SURF-04 work lands) must NOT reuse `defaultDepth`/`clampDepth` unmodified — TS's own `affected` default is 5, not 2 (per 08-RESEARCH.md's documented pitfall); a sibling constant is required there.

---
*Phase: 08-surface-reconciliation-signed-v1-0-0-release*
*Completed: 2026-07-19*

## Self-Check: PASSED

All claimed files exist (internal/query/validate.go, internal/query/engine_test.go, internal/cli/impact.go, this SUMMARY.md); all claimed commits (15aaac3, 6fae70f, 3c316e4) verified present in git log.
