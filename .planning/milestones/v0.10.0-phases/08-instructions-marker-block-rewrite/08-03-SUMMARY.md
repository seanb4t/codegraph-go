---
phase: 08-instructions-marker-block-rewrite
plan: 03
subsystem: testing
tags: [wire-oracle, mcp, golden-transcripts, wire-contract]

requires:
  - phase: 08-instructions-marker-block-rewrite
    provides: "plan 08-01's rewritten internal/mcp instructions const (server.go), the byte-for-byte value every affected transcript now needed to carry"
provides:
  - "38 re-frozen .golden wire-oracle transcripts carrying the rewritten instructions string byte-identically to what a live client receives"
  - "A verified 38-changed / 4-untouched diff with every changed line attributed to one named cause in the commit message"
  - "Explicit toolslist-repeat.golden inspection ruling out the separately-tracked tools/list arrival-order flake"
affects: [08-instructions-marker-block-rewrite phase completion, any future plan touching internal/mcp/server.go's instructions const]

actuals:
  tokens: 35523
  tasks: 2
  commits: 1

tech-stack:
  added: []
  patterns:
    - "Sequential (non-parallel) capture-CLI re-freeze of a golden transcript corpus, one wireoracle -scenario invocation per file redirected to its .golden path, to avoid seeding a documented arrival-order flake into frozen bytes"

key-files:
  created: []
  modified:
    - testdata/wireoracle/transcripts/call-callees.golden
    - testdata/wireoracle/transcripts/call-callers.golden
    - testdata/wireoracle/transcripts/call-files.golden
    - testdata/wireoracle/transcripts/call-impact.golden
    - testdata/wireoracle/transcripts/call-node.golden
    - testdata/wireoracle/transcripts/call-search.golden
    - testdata/wireoracle/transcripts/call-status.golden
    - testdata/wireoracle/transcripts/error-confinement-reject.golden
    - testdata/wireoracle/transcripts/error-malformed-args.golden
    - testdata/wireoracle/transcripts/error-unknown-method.golden
    - testdata/wireoracle/transcripts/error-unknown-tool.golden
    - testdata/wireoracle/transcripts/handshake-explore.golden
    - testdata/wireoracle/transcripts/index-appears-mid-session.golden
    - testdata/wireoracle/transcripts/legacy-2024-11-05.golden
    - testdata/wireoracle/transcripts/legacy-2025-03-26.golden
    - testdata/wireoracle/transcripts/legacy-2025-06-18.golden
    - testdata/wireoracle/transcripts/legacy-2025-11-25.golden
    - testdata/wireoracle/transcripts/legacy-omitted-version.golden
    - testdata/wireoracle/transcripts/legacy-unsupported-2026-07-28.golden
    - testdata/wireoracle/transcripts/modern-discover-explore.golden
    - testdata/wireoracle/transcripts/resources-list-no-index.golden
    - testdata/wireoracle/transcripts/resources-list.golden
    - testdata/wireoracle/transcripts/resources-read-callees.golden
    - testdata/wireoracle/transcripts/resources-read-callers.golden
    - testdata/wireoracle/transcripts/resources-read-explore.golden
    - testdata/wireoracle/transcripts/resources-read-files.golden
    - testdata/wireoracle/transcripts/resources-read-impact.golden
    - testdata/wireoracle/transcripts/resources-read-index-state.golden
    - testdata/wireoracle/transcripts/resources-read-node.golden
    - testdata/wireoracle/transcripts/resources-read-search.golden
    - testdata/wireoracle/transcripts/resources-read-status.golden
    - testdata/wireoracle/transcripts/resources-read-tools-filter.golden
    - testdata/wireoracle/transcripts/resources-read-unknown.golden
    - testdata/wireoracle/transcripts/toolslist-default.golden
    - testdata/wireoracle/transcripts/toolslist-filter-empty.golden
    - testdata/wireoracle/transcripts/toolslist-narrowed.golden
    - testdata/wireoracle/transcripts/toolslist-no-index.golden
    - testdata/wireoracle/transcripts/toolslist-repeat.golden

key-decisions:
  - "Combined Task 1 (re-capture) and Task 2 (reviewed-diff pass) into a single commit, since both tasks operate on the exact same fileset and the plan's own Task 2 action is to review the diff BEFORE naming the commit message — reviewing pre-commit, then committing once with full attribution, is the natural ordering rather than an artificial two-commit split of one logical change."
  - "Captured all 38 scenarios sequentially via a single shell loop (not parallel) per the plan's explicit instruction, to avoid seeding the documented toolslist-repeat arrival-order flake into the frozen bytes."

requirements-completed: [WIRE-01]

coverage:
  - id: D1
    description: "38 of 42 wire-oracle transcripts re-captured from a freshly built binary via the wireoracle capture CLI, carrying the rewritten instructions string byte-identically; the other 4 (edge-call-before-initialize, modern-listen-catalog-change, modern-meta-invalid-params, modern-meta-unsupported-version) are structurally exempt and untouched"
    requirement: "WIRE-01"
    verification:
      - kind: unit
        ref: "test/wireoracle (TestFrozenTranscriptsMatch via `go test ./test/wireoracle/... -count=1`)"
        status: pass
      - kind: other
        ref: "git status --porcelain testdata/wireoracle/transcripts/ | wc -l == 38"
        status: pass
      - kind: other
        ref: "git status --porcelain testdata/wireoracle/transcripts/ | rg -o '(edge-call-before-initialize|modern-listen-catalog-change|modern-meta-invalid-params|modern-meta-unsupported-version)' | wc -l == 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every added/removed line in the transcript diff is attributed to the single instructions-string cause (38 added, 38 removed, one per file); no unattributed change; toolslist-repeat's diff shows only the instructions line moving, confirming the tools/list arrival-order flake was not absorbed"
    requirement: "WIRE-01"
    verification:
      - kind: other
        ref: "git diff -U0 testdata/wireoracle/transcripts/ | rg '^[+-]' | rg -v '^(\\+\\+\\+|---)' | rg -v 'codegraph indexes this repository' | wc -l == 0"
        status: pass
      - kind: unit
        ref: "go test ./internal/mcp/ ./internal/agents/ -count=1"
        status: pass
    human_judgment: false

duration: ~15min
completed: 2026-08-13
status: complete
---

# Phase 8 Plan 3: Wire-Oracle Transcript Re-Freeze Summary

**Re-captured 38 of 42 frozen wire-oracle transcripts against a freshly built binary so every `initialize`/`server/discover` result carries plan 08-01's rewritten `instructions` string byte-identically, with the reviewed diff attributed to one named cause and the pre-existing `toolslist-repeat` ordering flake explicitly ruled out.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-08-13T21:05:00Z
- **Tasks:** 2
- **Files modified:** 38 (`.golden` transcripts only)

## Accomplishments

- Derived the affected scenario set fresh (never from memory): `rg -l "codegraph indexes this repository" testdata/wireoracle/transcripts/` returned exactly 38 files, and none of the 4 handshake-free scenarios (`edge-call-before-initialize`, `modern-listen-catalog-change`, `modern-meta-invalid-params`, `modern-meta-unsupported-version`) appeared — matching plan 08-CONTEXT.md's Flagged Assumption A-04 exactly.
- Built `codegraph` once (`go build -o <scratch>/codegraph-recapture ./cmd/codegraph`) and re-captured all 38 scenarios sequentially, one `go run ./test/wireoracle/cmd/wireoracle -bin <fresh> -fixture testdata/wireoracle/fixture -scenario <name> > testdata/wireoracle/transcripts/<name>.golden` invocation per scenario, in a single shell loop — no parallelism, per the plan's explicit anti-flake instruction.
- Reviewed the full diff before committing: every added/removed line across all 38 files matched the `instructions`-string cause (`git diff -U0 ... | rg -v 'codegraph indexes this repository'` returned zero unattributed lines); exactly 38 lines added and 38 removed, one per file.
- Gave `toolslist-repeat.golden` explicit scrutiny per Task 2's instruction: its diff shows only the single `initialize` result's `instructions` field changing (line 1 of 3); the two `tools/list` response lines (2 and 3) are byte-identical before and after — **instructions line only, no ordering difference.** The known arrival-order flake was not absorbed.
- `go test ./test/wireoracle/... -count=1` passes cleanly (`TestFrozenTranscriptsMatch` and `len(Scenarios()) == ExpectedScenarioCount == 42`), `go test ./internal/mcp/ ./internal/agents/ -count=1` passes, `go build ./...` succeeds, and `git diff --stat test/wireoracle/` is empty (no harness/scenario source touched).

## Task Commits

Both plan tasks (re-capture + reviewed-diff attribution) landed as one commit, since Task 2's action is to review the Task 1 diff *before* naming the commit message — reviewing pre-commit and committing once with the full attribution is the natural ordering for one logical file-set change, not an artificial two-commit split.

1. **Task 1 + Task 2: Re-capture 38 transcripts and attribute the reviewed diff** - `329bc0c` (test)

## Files Created/Modified

38 `.golden` transcript files under `testdata/wireoracle/transcripts/` — see `key-files.modified` in frontmatter for the complete list. No Go source, `scenarios.go`, or `COVERAGE-BASELINE.md` changed (by design — the scenario set did not move).

## Decisions Made

- Combined Task 1 and Task 2 into a single commit rather than two, since Task 2's action ("read the complete diff ... write the commit message") is itself a pre-commit review step over Task 1's output, not a separate code change.
- Captured sequentially (not in parallel) to avoid seeding the documented `toolslist-repeat` arrival-order flake into the frozen bytes, per the plan's explicit instruction.

## Deviations from Plan

None - plan executed exactly as written for both tasks.

## Issues Encountered

None. The derived scenario count (38) and the exclusion set (the 4 named handshake-free scenarios) matched the plan's Flagged Assumption A-04 exactly, so no stop condition was triggered.

## Toolslist-Repeat Inspection Outcome

**Instructions line only, no ordering difference.** `toolslist-repeat.golden` is a 3-line transcript (one `initialize` result line, two `tools/list` result lines). Only line 1 (the `initialize` result carrying the `instructions` field) changed; lines 2 and 3 (the two `tools/list` responses under test by `TestToolsListOrderIsDeterministic`) are byte-identical to their pre-recapture values. The separately-tracked `tools/list` arrival-order flake (`.planning/todos/2026-08-07-wire-oracle-toolslist-repeat-response-ordering-flake.md`) was not observed and was not silently absorbed into this re-freeze.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The `test/wireoracle` corpus is fully green again (`TestFrozenTranscriptsMatch` passes for all 42 scenarios) and describes exactly what a live client receives, satisfying ROADMAP success criterion 4 for Phase 8.
- No remaining wire-oracle drift is outstanding for this phase; phase-level verification (`go test ./test/wireoracle/...`, `go test ./internal/mcp/ ./internal/agents/...`, `go build ./...`) all pass.

---
*Phase: 08-instructions-marker-block-rewrite*
*Completed: 2026-08-13*
