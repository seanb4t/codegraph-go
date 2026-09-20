---
phase: 04-cli-glow-up
plan: 04
subsystem: cli
tags: [lipgloss, colorprofile, cobra, tdd, present, palette, query]

# Dependency graph
requires:
  - phase: 04-03
    provides: "resolveColor(cmd) resolver, colorMode.Writer, present.Palette/NewPalette, and the shared present.stripANSI test helper (ansistrip_test.go) — the tracer this plan expands"
provides:
  - "internal/cli/present/results.go: RenderSearch, RenderSearchFull, RenderCallers, RenderCallees, RenderImpact, RenderAffected, RenderNotice, plus writeLocationLine/writeNodeLine helpers"
  - "search.go (--full and default), callers.go, callees.go, impact.go, affected.go each wired with a styled branch after their --json (and, for affected.go, --quiet) early return"
affects: [04-05, 04-06, 04-07, 04-08]

# Actuals (#2632)
actuals:
  tokens: 6993
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Six-verb list renderer: one writeLocationLine helper (unstyled separators \" (\", \") \", \":\", \"\\n\" preserved verbatim) shared by all six renderers so the ANSI-stripped bytes equal the plain %s (%s) %s:%d format by construction, never by re-deriving it per renderer"
    - "Styled-branch insertion point: mode := resolveColor(cmd); if mode.Styled { ... return present.RenderX(...) } inserted strictly after every --json (and affected.go's --quiet) early return and before the frozen plain fmt.Fprintf loop — the plain path is never touched, only additively wrapped"

key-files:
  created:
    - internal/cli/present/results.go
    - internal/cli/present/results_test.go
  modified:
    - internal/cli/search.go
    - internal/cli/callers.go
    - internal/cli/callees.go
    - internal/cli/impact.go
    - internal/cli/affected.go

key-decisions:
  - "RenderNotice styles the worktree notice line-by-line (splitting on \\n, styling each non-empty line, preserving the trailing newline) so a multi-line notice strips back to the exact plain string, matching query.WorktreeNotice's own byte shape rather than assuming a single line"
  - "Task 1's GREEN commit was folded into a single feat(04-04) commit alongside Task 2's wiring (plan explicitly permits either split), keeping the plan at 2 commits total: one RED test(04-04), one GREEN feat(04-04)"

requirements-completed: [CLI-01, CLI-05]

coverage:
  - id: D1
    description: "Six list renderers (RenderSearch, RenderSearchFull, RenderCallers, RenderCallees, RenderImpact, RenderAffected) plus RenderNotice, each proven to strip back to the plain bytes over empty/single/three-unordered/duplicate/unicode/control-byte fixtures"
    requirement: "CLI-01"
    verification:
      - kind: unit
        ref: "internal/cli/present/results_test.go#TestRenderResultsStrippedEqualsPlain"
        status: pass
    human_judgment: false
  - id: D2
    description: "search (--full and default), callers, callees, impact and affected render through the palette on --color=always and stay byte-identical to their plain/golden output otherwise"
    requirement: "CLI-05"
    verification:
      - kind: integration
        ref: "internal/cli TestPlainGolden / TestNoColorNonTTYRegression (29/29 subtests)"
        status: pass
      - kind: other
        ref: "real-binary loop: search/search --full/callers/callees/impact/affected under --color=always (ESC present) vs plain pipe (ESC-free), perl-stripped styled == plain via cmp; affected --quiet --color=always ESC-free"
        status: pass
    human_judgment: true
    rationale: "Hue legibility on light/dark terminals (D-06) is a human UAT item harvested at end of phase alongside CLI-04, not verified here — the content/byte contract above is fully automated, but 'looks right' is not."

# Metrics
duration: 15min
completed: 2026-09-17
status: complete
---

# Phase 4 Plan 04: Six List Renderers (search/callers/callees/impact/affected) Summary

**Six bespoke `present` renderers for the list-shaped query verbs, pinned by a stripped-styled-equals-plain content contract and wired into every RunE after its `--json`/`--quiet` early return.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-09-17T21:38:00Z
- **Completed:** 2026-09-17T21:53:38Z
- **Tasks:** 2
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments
- `internal/cli/present/results.go` implements `RenderSearch`, `RenderSearchFull`, `RenderCallers`, `RenderCallees`, `RenderImpact`, `RenderAffected` and `RenderNotice`, sharing one `writeLocationLine` helper so every renderer's ANSI-stripped output equals the RunE's own `"%s (%s) %s:%d\n"` plain format byte-for-byte
- `results_test.go`'s `TestRenderResultsStrippedEqualsPlain` covers all seven renderers across empty/single/three-unordered/duplicate/unicode/control-byte fixtures (36 leaf subtests + 1 positive-control subtest), reusing plan 03's shared `stripANSI`
- `search.go` (both `--full` and default branches), `callers.go`, `callees.go`, `impact.go` and `affected.go` each gained a styled branch — `resolveColor(cmd)` → `colorprofile.Writer` → `present.NewPalette(mode.Dark)` → `RenderNotice` + `RenderX` — inserted strictly after the `--json` (and, in `affected.go`, `--quiet`) early return and before the untouched plain path
- Real-binary verification: all six invocations (`search`, `search --full`, `callers`, `callees`, `impact`, `affected`) show ANSI escapes under `--color=always`, stay ESC-free on a plain pipe, and their `perl`-stripped styled output is byte-identical to the plain output via `cmp`; `affected --quiet --color=always` stays ESC-free

## Task Commits

Each task was committed atomically (TDD: RED test commit precedes the GREEN feat commit):

1. **Task 1 (RED): Six list renderers + RenderNotice, pinned by stripped-styled == plain** - `5a47768a` (test) — `results_test.go` + placeholder `results.go` (all seven funcs write nothing, return nil)
2. **Task 1+2 (GREEN): Implement the six renderers and wire the five RunE styled branches** - `7aa2bbb3` (feat) — real `results.go` implementation + `search.go`/`callers.go`/`callees.go`/`impact.go`/`affected.go` styled branches

**Plan metadata:** committed separately per `git_commit_metadata` step (see below).

_Note: this is a `tdd="true"` task pair, not a full `type: tdd` plan — 2 commits total (RED test, GREEN feat), matching the plan's own "one feat commit is fine" allowance for folding Task 1's GREEN into Task 2's commit._

## RED Phase Evidence

Target test intentionally failed on the byte comparison (never a crash/syntax error/zero-test-discovery — `INVALID_RED` per #3770 would have been a hard stop):

```
--- FAIL: TestRenderResultsStrippedEqualsPlain (0.00s)
    --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderSearch (0.00s)
        --- PASS: TestRenderResultsStrippedEqualsPlain/RenderSearch/empty (0.00s)
        --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderSearch/single (0.00s)
        --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderSearch/three-unordered (0.00s)
        --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderSearch/duplicate (0.00s)
        --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderSearch/unicode (0.00s)
        --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderSearch/control-bytes (0.00s)
    --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderSearchFull (0.00s)
        (... single/three-unordered/duplicate/unicode/control-bytes all FAIL, empty PASSes ...)
    --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderCallers (0.00s)
        (all 6 fixtures FAIL — header line always non-empty)
    --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderCallees (0.00s)
    --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderImpact (0.00s)
    --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderAffected (0.00s)
    --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderNotice (0.00s)
        --- PASS: TestRenderResultsStrippedEqualsPlain/RenderNotice/empty (0.00s)
        --- FAIL: TestRenderResultsStrippedEqualsPlain/RenderNotice/non-empty (0.00s)
    --- FAIL: TestRenderResultsStrippedEqualsPlain/PositiveControl_StyledOutputContainsANSI (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli/present	0.224s
```

Every failure is `got: ""` vs a non-empty `want:` — the placeholder renderers wrote nothing, so the assertion failed for exactly the intended reason (never `check tdd-red-evidence`, per project rules).

## Real-Binary 6/6 Result

```
6/6 invocations OK, n=6, quiet ESC-free
```

Ran against a fresh `codegraph init` over `internal/indexer/testdata/gofixture`: `search Alpha`, `search Alpha --full`, `callers Alpha`, `callees Alpha`, `impact Alpha`, `affected pkga/pkga.go` — each compared `--color=always` (ESC present) against the plain pipe (ESC-free, non-empty), and `perl -pe 's/\e\[[0-9;]*m//g'` on the styled output matched the plain output byte-for-byte via `cmp`. `affected pkga/pkga.go --quiet --color=always` produced zero ESC bytes.

## Files Created/Modified
- `internal/cli/present/results.go` - Seven renderers (`RenderSearch`, `RenderSearchFull`, `RenderCallers`, `RenderCallees`, `RenderImpact`, `RenderAffected`, `RenderNotice`) plus `writeLocationLine`/`writeNodeLine`
- `internal/cli/present/results_test.go` - `TestRenderResultsStrippedEqualsPlain` contract table + plain-expectation helpers mirroring the RunE formats verbatim
- `internal/cli/search.go` - styled branch in both `--full` and default human paths
- `internal/cli/callers.go` - styled branch after `--json`
- `internal/cli/callees.go` - styled branch after `--json`
- `internal/cli/impact.go` - styled branch after `--json`
- `internal/cli/affected.go` - styled branch after `--json` and `--quiet` (quiet stays plain/unstyled)

## Decisions Made
- `RenderNotice` splits a multi-line notice on `\n`, styles each non-empty line with `Warning`, and rejoins preserving the original trailing-newline structure — this is the "simplest faithful form" the plan called for, verified against both an empty notice (no output) and a real single-line WORK-02 notice string in the contract test.
- Folded Task 1's GREEN implementation into the same commit as Task 2's RunE wiring (one `feat(04-04):` commit) rather than a separate GREEN-only commit — the plan explicitly allows either split ("Commit GREEN as part of this plan's `feat(04-04):` (after Task 2, or now — one feat commit per plan is fine)").

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- `TestEveryRegisteredFlagIsAccountedFor` fails on the pre-existing `codegraph --color` flag (registered in plan 04-03, documented once `docs/CLI-REFERENCE.md` is regenerated in plan 04-08). This is the project's own documented "known scheduled RED" — not caused by this plan's changes. Ran the full `internal/cli/...` suite with `-skip 'TestEveryRegisteredFlagIsAccountedFor$'` to confirm everything else (including `TestPlainGolden`/`TestNoColorNonTTYRegression`, 29/29) is green.
- The plan's Task 2 acceptance-criteria grep command (`git log --format=%H -E --grep='^test\(04-04\): ' | tail -1`) matches an unrelated, much older commit from a previous milestone that reused the same phase/plan number ("04-04") — a known repo-wide caveat documented in this project's own `update_codebase_map` workflow step (#4459: "a phase number is unique within a MILESTONE, not a repository"). Substituted the correct anchor (this plan's own RED commit, `5a47768a`, found via `git log --oneline -5`) to verify the plain-format/notice grep counts were unchanged before/after Task 2's wiring — confirmed unchanged for all five files.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `present/results.go`'s renderers and the shared `writeLocationLine`/`writeNodeLine` helpers are available for plans 04-05/04-06 (which own `present/explore.go`, `present/node.go`, `present/line.go` and their respective RunE files) to reference as the established list-rendering pattern, though this plan does not modify those files.
- No blockers. `internal/query`, `internal/mcp`, `testdata/golden` and `testdata/wireoracle` are untouched, confirmed via `git diff --name-only`.

## Self-Check: PASSED

All created files verified present on disk; both commit hashes (`5a47768a` RED, `7aa2bbb3` GREEN) verified present in `git log --oneline --all`.

---
*Phase: 04-cli-glow-up*
*Completed: 2026-09-17*
