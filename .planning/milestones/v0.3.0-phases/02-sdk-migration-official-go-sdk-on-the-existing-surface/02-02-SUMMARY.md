---
phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface
plan: 02
subsystem: infra
tags: [ci, go, anti-regeneration-guard, transcriptfreeze, taskfile]

# Dependency graph
requires:
  - phase: 01-protocol-scoping-the-sdk-independent-wire-oracle
    provides: "D-03's two-mechanism anti-regeneration guard (tools/transcriptfreeze + the frozen transcript corpus) and the CI cross-change rule that blocks a PR touching both a frozen transcript and internal/mcp/*.go or go.mod's MCP dependency line"
provides:
  - "A self-expiring SwapExemption predicate in tools/transcriptfreeze/classify.go, keyed to a go.mod diff that both removes github.com/mark3labs/mcp-go and adds github.com/modelcontextprotocol/go-sdk"
  - "Verdict.SwapExemption field and buildSwapExemptionNotice(), wired into main.go to print the exemption notice to stderr and return 0 when it fires"
  - "Taskfile.yml's check:transcript-freeze description and classify.go's package doc corrected to describe the exemption instead of the superseded 'Phase 2 = frozen, no exceptions' clause"
  - "Confirmed (both via go test and a real `task check:transcript-freeze` run) that the guard still exits non-zero on this branch today, and via a synthetic binary invocation that it exits 0 and prints the notice for the exact swap shape"
affects: ["02-04 (removes mark3labs from go.mod — the moment this exemption becomes reachable on the real branch diff)", "02-05 (re-runs check:transcript-freeze and records the actual pass)"]

actuals:
  tokens: 3312
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Self-expiring exemption keyed to a diff SHAPE (not env/flag/file) — the same escape-hatch shape internal/mcp/archtest/protocol_version_test.go already documents for VRFY-02"

key-files:
  created: []
  modified:
    - tools/transcriptfreeze/classify.go
    - tools/transcriptfreeze/classify_test.go
    - tools/transcriptfreeze/main.go
    - Taskfile.yml

key-decisions:
  - "Reused mcpSDKModulePrefixes[0]/[1] by index inside sdkSwapExemption rather than retyping the module path strings, so the exemption structurally cannot name a module MCPDependencyTouched doesn't already track."
  - "SwapExemption is computed only inside the existing collision branch (transcripts present AND (server changes present OR dep touched)) — a diff with no transcript path never touches the exemption path at all, leaving the already-clean case provably unaffected."
  - "main.go checks SwapExemption before Violation and always prints the notice to stderr on that path, even though it returns 0 — an exempted run must be visible in the CI log, never silent."

requirements-completed: [SDK-01]

coverage:
  - id: D1
    description: "A go.mod diff removing mark3labs and adding go-sdk, alongside a transcript+internal/mcp collision, is exempted (Violation=false, SwapExemption=true, notice printed to stderr)"
    requirement: SDK-01
    verification:
      - kind: unit
        ref: "tools/transcriptfreeze/classify_test.go#TestClassifySDKSwapExemption/full_swap_plus_transcript+internal/mcp_change_is_exempted"
        status: pass
      - kind: integration
        ref: "go run ./tools/transcriptfreeze against synthetic -changed-list/-gomod-diff files carrying the swap shape, observed exit 0 with the exemption notice on stderr"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every neighbouring diff shape (go-sdk added without mark3labs removed; mark3labs removed without go-sdk added; empty go.mod diff; both module paths as context-only lines; no transcript path present) still reports Violation=true or leaves the already-clean case untouched — the exemption is not vacuous"
    requirement: SDK-01
    verification:
      - kind: unit
        ref: "tools/transcriptfreeze/classify_test.go#TestClassifySDKSwapExemption (4 sub-cases) and #TestSDKSwapExemptionRequiresActualDiffLines"
        status: pass
      - kind: other
        ref: "Demonstrated RED: sdkSwapExemption inverted to `return true` unconditionally, git diff confirmed the mutation landed, `go test ./tools/transcriptfreeze/... -count=1` failed on the three non-vacuity Violation:true cases (plus pre-existing tests that also depend on the guard firing), then reverted and confirmed green"
        status: pass
    human_judgment: false
  - id: D3
    description: "Taskfile.yml's check:transcript-freeze description and classify.go's package doc no longer assert the superseded 'Phase 2 is frozen with no exceptions' clause; ci.yml is untouched; the real branch diff against main still exits non-zero today (mark3labs not yet removed)"
    requirement: SDK-01
    verification:
      - kind: other
        ref: "rg -n 'Phase 2 is frozen with no exceptions' Taskfile.yml tools/transcriptfreeze/classify.go returns nothing; git diff --stat -- .github/workflows/ci.yml is empty; TRANSCRIPT_FREEZE_BASE=main task check:transcript-freeze exits non-zero (underlying binary: exit status 1) naming all 23 transcripts and eight internal/mcp files"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-08-05
status: complete
---

# Phase 2 Plan 02: Self-expiring SDK-01 swap exemption for the transcript-freeze guard Summary

**A go.mod-diff-shaped, self-expiring exemption lets the SDK-01 mark3labs→go-sdk swap through the blocking D-03 anti-regeneration guard, announces itself loudly on stderr, and has been demonstrated RED against every neighbouring shape by mutation.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 2/2 completed
- **Files modified:** 4 (classify.go, classify_test.go, main.go, Taskfile.yml)

## Accomplishments

- Added `sdkSwapExemption(goModDiff string) bool` to `tools/transcriptfreeze/classify.go`: fires only when the go.mod diff contains BOTH a removal line naming `mcpSDKModulePrefixes[0]` (`github.com/mark3labs/mcp-go`) and an addition line naming `mcpSDKModulePrefixes[1]` (`github.com/modelcontextprotocol/go-sdk`). Reuses the existing slice by index rather than retyping the module paths.
- Added `Verdict.SwapExemption bool`. `Classify` now checks the exemption predicate inside the existing collision branch: when it fires, `Violation` stays `false`, `SwapExemption` is set `true`, and `Reason` carries a new `buildSwapExemptionNotice` message naming both waived sides plus the three facts a reviewer needs (this is SDK-01's one-time transition; D-01/D-02/D-03 replaced byte-identity with semantic equivalence; the exemption is self-expiring because mark3labs will be absent from `go.mod` afterward).
- Wired `main.go` to print the exemption notice to stderr and return 0 when `SwapExemption` is true — an exempted CI run is never a silent pass.
- Extended `classify_test.go` with `TestClassifySDKSwapExemption` (5 table cases covering the exempted shape and four non-exempted neighbours) and `TestSDKSwapExemptionRequiresActualDiffLines` (context-line-only proof).
- Demonstrated RED: inverted `sdkSwapExemption` to `return true` unconditionally, confirmed the mutation landed via `git diff`, ran `go test ./tools/transcriptfreeze/... -count=1` and observed the three non-vacuity `Violation: true` cases fail (along with several pre-existing tests that also depend on the guard firing), then reverted and confirmed green.
- Retired the superseded "Phase 2 is frozen with no exceptions" claim from `Taskfile.yml`'s `check:transcript-freeze` description and added a corresponding note to `classify.go`'s package doc comment, without touching `buildReason`'s "floor, not a proof of innocence" disclosure or `.github/workflows/ci.yml`.
- Verified with the real, wired task: `TRANSCRIPT_FREEZE_BASE=main task check:transcript-freeze` still exits non-zero on this branch (underlying `go run ./tools/transcriptfreeze` reports `exit status 1`), naming all 23 transcripts and eight `internal/mcp` files — exactly as expected, since mark3labs has not yet been removed from `go.mod` (that lands in plan 02-04) and the exemption correctly does not apply yet.
- Additionally sanity-checked the wiring end-to-end (beyond the plan's stated `<verify>` command) with synthetic `-changed-list`/`-gomod-diff` files carrying the swap shape: the compiled binary printed the exemption notice on stderr and exited 0; the add-only variant printed the violation reason and exited 1.

## Task Commits

Each task was committed atomically:

1. **Task 1: A self-expiring exemption for the one-time SDK swap, red both ways** - `b9f8912` (feat)
2. **Task 2: Retire the superseded "Phase 2 is frozen with no exceptions" claim, and prove the real branch diff now passes** - `d6bcbf2` (docs)

## Files Created/Modified

- `tools/transcriptfreeze/classify.go` - Added `Verdict.SwapExemption`, `sdkSwapExemption`, `buildSwapExemptionNotice`, wired into `Classify`; updated package doc.
- `tools/transcriptfreeze/classify_test.go` - Added `TestClassifySDKSwapExemption` and `TestSDKSwapExemptionRequiresActualDiffLines`.
- `tools/transcriptfreeze/main.go` - Print the exemption notice to stderr and return 0 when `SwapExemption` is true; updated exit-code doc comment.
- `Taskfile.yml` - Rewrote `check:transcript-freeze`'s description to state the exemption instead of the superseded "no exceptions" clause.

## Decisions Made

- Reused `mcpSDKModulePrefixes` by index (`[0]` = mark3labs, `[1]` = go-sdk) inside the new predicate rather than retyping the module path strings — the exemption structurally cannot name a module the guard doesn't already treat as an MCP SDK dependency.
- `SwapExemption` is computed only inside the existing collision `if` branch. A diff with no transcript path never reaches the exemption logic at all, which is the cheapest possible proof that the already-clean, non-colliding case is unaffected — no separate early-return branch was needed.
- `main.go` checks `SwapExemption` before `Violation` and always writes the notice to stderr on that path even though the process exits 0, matching the plan's explicit requirement that an exemption "is never a silent pass."

## Deviations from Plan

None — plan executed as written. One informational note: Task 2's acceptance criterion `rg -c 'floor, not a proof of innocence' tools/transcriptfreeze/classify.go returns 1` does not match the actual count, which is 3 (package doc, `buildReason`'s own doc comment, and `buildReason`'s string) — and was already 3 in the pre-Phase-2 baseline (`git show HEAD~2:tools/transcriptfreeze/classify.go` also returns 3). This task did not touch any of those three sites, so the count is unchanged before and after; the substantive property the criterion is checking for — "the disclosure survives, unmodified" — holds. The literal expected count in the plan text appears to have been miscounted at authoring time against a file the planner had already read in full. Not treated as a deviation requiring a fix, since fixing it would mean altering `buildReason`'s disclosure, which the plan explicitly forbids ("Leave buildReason's ... disclosure exactly as written").

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Phase 2's blocking `check:transcript-freeze` PR gate now has a narrow, self-expiring exemption for the SDK-01 swap. `tools/transcriptfreeze` is otherwise unchanged in behavior for every other diff shape (proven by table cases and a real mutation demonstration). `test/wireoracle/` and `testdata/wireoracle/` were not touched, and `.github/workflows/ci.yml` was not touched.

This unblocks the rest of Wave 1/2/3 plans in Phase 2: once 02-04 removes `github.com/mark3labs/mcp-go` from `go.mod` alongside adding `github.com/modelcontextprotocol/go-sdk`, the same PR's frozen-transcript and `internal/mcp` changes will hit `Classify`'s exemption branch instead of `Violation`. Because the exemption's key is a removal line for a module that will then be permanently absent from `go.mod`, it cannot fire again — 02-05 (and every subsequent PR) is checked against the fully-restored, non-exempted guard.

---
*Phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface*
*Completed: 2026-08-05*

## Self-Check: PASSED

- FOUND: tools/transcriptfreeze/classify.go
- FOUND: tools/transcriptfreeze/classify_test.go
- FOUND: tools/transcriptfreeze/main.go
- FOUND: Taskfile.yml
- FOUND commit b9f8912 (git log --oneline -5)
- FOUND commit d6bcbf2 (git log --oneline -5)
