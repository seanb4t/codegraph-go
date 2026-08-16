---
phase: 05-process-ci-in-tree-sweep
plan: 07
subsystem: docs
tags: [ripgrep, multiline-census, comment-hygiene, mcp, cobra, git-hooks]

requires:
  - phase: 05-process-ci-in-tree-sweep
    provides: "05-04/05-05's initial CODE-01 sweep and 05-VERIFICATION.md's gap report identifying 10 residual locations missed by a line-based census"
provides:
  - "Wrap-tolerant (rg -U), positive-controlled census instrument proven live against a synthetic fixture before being trusted"
  - "9 Go files reworded to remove TS-comparison-baseline framing while preserving every technical citation"
  - "Full internal/agents population (30 files) audited, not the prior 10-name sample"
  - "WINDOWS.md ledger entries 4-11 and 14 closed; one new out-of-scope find (entry 17, internal/cli/root.go) logged rather than silently swept"
affects: [ROADMAP Phase 5 verification re-run, future in-tree comment sweeps]

actuals:
  tokens: 2338
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Wrap-tolerant multiline rg -U census with a synthetic positive control proven BEFORE trusting a zero count"
    - "Term-by-term KEEP/FIXED adjudication of every retained citation, never blanket regex deletion"

key-files:
  created: []
  modified:
    - internal/cli/search.go
    - internal/cli/node.go
    - internal/cli/files.go
    - internal/cli/uninit.go
    - internal/cli/serve.go
    - internal/cli/githooks_test.go
    - internal/mcp/tools.go
    - internal/agents/codex.go
    - internal/query/traverse_test.go
    - .planning/WINDOWS.md

key-decisions:
  - "Built and positive-controlled the wrap-tolerant rg -U census against a synthetic wrapped fixture before trusting any zero-count result (multiline=4 matches vs line-based=0 on identical input)"
  - "Every reworded comment preserves its technical citation (WORK-02, D-12, D-13, WR-05, WR-01, D-03, D-06, SURF-02, SURF-04, SURF-06, D-16, D-17, D-05a, CONTEXT D-03) — none were deleted, only the comparison-baseline clause"
  - "opencode.go/claude.go/instructions.go's 4 TS-issue-number citations adjudicated as BUG-PRECEDENT-CITATION (D-02 carve-out) and left unmodified — they cite a historical bug tracker issue, not an ongoing comparison baseline"
  - "internal/cli/root.go:12's 'no TS CodeGraph counterpart' framing found during the internal/cli scope check but left unedited (outside this plan's files_modified) and logged to WINDOWS.md entry 17 per scope discipline"

patterns-established:
  - "A census gate must prove its own instrument (positive control on wrapped synthetic input) before a zero count is trusted as evidence of absence"

requirements-completed: [CODE-01]

coverage:
  - id: D1
    description: "internal/cli/search.go, node.go, files.go reworded — worktree-notice placement rationale and --dir semantics stated on codegraph-go's own terms; --dir help text de-framed"
    requirement: "CODE-01"
    verification:
      - kind: unit
        ref: "go test ./internal/cli/... (full package suite)"
        status: pass
      - kind: other
        ref: "rg -U -o -i '\\bTS\\b' internal/cli/search.go internal/cli/node.go internal/cli/files.go → 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "internal/cli/uninit.go, serve.go, githooks_test.go reworded — D-06 cleanup contract, D-12/D-13 disabled-message contract, D-03 marker-detection note — with every asserted byte (fmt.Fprintf literal, both marker-byte copies) mechanically proven unchanged via git diff comment-only assertion"
    requirement: "CODE-01"
    verification:
      - kind: unit
        ref: "go test ./internal/cli/... ./internal/githooks/..."
        status: pass
      - kind: other
        ref: "git diff -U0 non-comment-line check → 0 non-comment changed lines across the three files"
        status: pass
    human_judgment: false
  - id: D3
    description: "internal/mcp/tools.go, internal/agents/codex.go, internal/query/traverse_test.go reworded; full internal/agents package (30 files) audited — historical bug-precedent citations in opencode.go/claude.go/instructions.go adjudicated KEEP with recorded reasons"
    requirement: "CODE-01"
    verification:
      - kind: unit
        ref: "go test ./internal/mcp/... ./internal/agents/... ./internal/query/..."
        status: pass
      - kind: other
        ref: "rg -U -o -i framing-pattern census over internal/agents/ → 0; historical citation count → 4 (opencode.go x2, claude.go, instructions.go)"
        status: pass
    human_judgment: false

duration: 45min
completed: 2026-08-16
status: complete
---

# Phase 05 Plan 07: CODE-01 Gap Closure — Wrap-Tolerant Sweep Summary

**Reworded 9 Go files' comparison-baseline framing (missed by both prior line-based censuses because the phrase wrapped across a comment line break) using a positive-controlled multiline `rg -U` census, and audited the full 30-file `internal/agents` population instead of the prior 10-name sample.**

## Performance

- **Duration:** 45 min
- **Started:** 2026-08-16T00:57Z
- **Completed:** 2026-08-16T01:41Z
- **Tasks:** 3
- **Files modified:** 10 (9 declared + WINDOWS.md)

## Accomplishments
- Built and positive-controlled a wrap-tolerant `rg -U` census instrument: on a synthetic fixture with the framing phrase split across a comment line break, the multiline patterns found 4 matches while the identical line-based patterns found 0 — demonstrating, not assuming, the defect that let 10 locations survive two prior "green" sweeps.
- Reworded the worktree-notice placement rationale in `search.go`/`node.go` and the `--dir` semantics doc/help-string in `files.go` onto codegraph-go's own terms, preserving WORK-02/D-12/WR-05/SURF-02/CONTEXT D-03 citations.
- Reworded the D-06 cleanup contract (`uninit.go`), the D-12/D-13 disabled-message contract (`serve.go`), and the D-03 marker-detection note (`githooks_test.go`) — mechanically proved every byte contract unchanged (`fmt.Fprintf` format literal, both `markerBegin`/`markerBeginBytes` copies) via a `git diff` non-comment-line assertion.
- Reworded the SURF-06/D-16 markdown-vs-JSON asymmetry record and the D-17 `codegraph_status` exclusion note in `internal/mcp/tools.go`, and `codex.go`'s global-scope-only + hand-rolled-TOML rationale.
- Audited the real `internal/agents` population (30 files: 15 source + 15 test, not 05-05's 10-name sample) and adjudicated all 4 issue-number citations (`opencode.go:39`, `opencode.go:233-234`, `claude.go:22`, `instructions.go:19`) as BUG-PRECEDENT-CITATION (D-02 carve-out) — left unmodified.
- Reworded `traverse_test.go`'s `affectedDepthFixtureNodesEdges` doc comment, dropping the `TS` qualifier while keeping the SURF-04/D-05 pruning property intact.
- Closed WINDOWS.md ledger entries 4-11 and 14 (the 9 CODE-01 gaps this plan fixes) and logged a new out-of-scope find (entry 17: `internal/cli/root.go:12`'s "no TS CodeGraph counterpart" framing, outside this plan's declared scope) rather than silently widening the edit set.

## Task Commits

Each task was committed atomically:

1. **Task 1: Build/positive-control the census, sweep worktree-notice + `--dir` sites** - `f9b8ce5` (docs)
2. **Task 2: Sweep the byte-contract cluster (uninit/serve/githooks_test)** - `1432626` (docs)
3. **Task 3: Sweep tools.go, full internal/agents, traverse_test.go; close WINDOWS ledger** - `8094f15` (docs)

_No plan-metadata commit — STATE.md/ROADMAP.md updates are owned by the orchestrator after the wave completes._

## Files Created/Modified
- `internal/cli/search.go` - worktree-notice placement rationale reworded (comment only)
- `internal/cli/node.go` - worktree-notice placement rationale reworded (comment only)
- `internal/cli/files.go` - `--dir` doc comment and flag help string de-framed
- `internal/cli/uninit.go` - D-06 cleanup contract reworded (comment only)
- `internal/cli/serve.go` - D-12/D-13 disabled-message contract note reworded (comment only; format literal untouched)
- `internal/cli/githooks_test.go` - D-03 marker-detection note reworded (comment only; `markerBeginBytes` value untouched)
- `internal/mcp/tools.go` - SURF-06/D-16 asymmetry record and D-17 exclusion note reworded (comment only)
- `internal/agents/codex.go` - global-scope-only + hand-rolled-TOML rationale reworded (comment only)
- `internal/query/traverse_test.go` - pruning-property doc comment reworded (comment only)
- `.planning/WINDOWS.md` - entries 4-11, 14 marked fixed; entry 17 (root.go) appended

## Decisions Made
- The census instrument's positive control (multiline finds what line-based cannot, on identical wrapped input) is a prerequisite gate, not an afterthought — run and recorded before any zero-count result was trusted for the three task verifications.
- Every technical citation in a reworded comment (WORK-02, D-12, D-13, WR-05, WR-01, D-03, D-06, SURF-02, SURF-04, SURF-06, D-16, D-17, D-05a, CONTEXT D-03) was preserved by construction — deletion of a citation would have failed the task's own verification gate (positive-count assertions), so none were lost.
- The 4 TS-issue-number citations in `opencode.go`/`claude.go`/`instructions.go` were read in full context and adjudicated individually as BUG-PRECEDENT-CITATION (D-02): each cites a specific historical bug in the tracked issue tracker as the reason for present behavior, not an ongoing "we do X because TS does X" baseline claim. Left unmodified.

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written for all 3 tasks; every verification gate passed on the first attempt.

### Out-of-scope find logged, not fixed

**1. `internal/cli/root.go:12` carries CODE-01 framing outside this plan's declared scope**
- **Found during:** Task 1's broader `internal/cli` scope check (searching for any residual `\bTS\b` beyond the three declared files)
- **Issue:** The package doc comment states githooks/man are "documented Go-only surface extensions with no TS CodeGraph counterpart" — comparison-baseline framing (asserting absence of a prior-implementation equivalent)
- **Action:** NOT fixed — `root.go` is not in this plan's `files_modified`. Logged to `.planning/WINDOWS.md` as entry 17 per the plan's scope-discipline instruction rather than silently widening the edit set.
- **Committed in:** `8094f15` (WINDOWS.md update, part of Task 3's commit)

---

**Total deviations:** 0 auto-fixed; 1 out-of-scope find logged (not a plan deviation — scope discipline working as designed)
**Impact on plan:** None. All three tasks' acceptance criteria and verification gates passed unmodified.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- CODE-01's 9-file gap (WINDOWS entries 4-11, 14) is closed. `go build ./...`, `go vet ./...`, and the full test suites for `internal/cli`, `internal/githooks`, `internal/mcp`, `internal/agents`, `internal/query` are all green.
- WINDOWS.md still carries 4 open entries after this plan: #12 (load-sensitive daemon test, pre-existing), #13 (pre-existing docs/RELEASE.md drift), #15 (`testdata/golden/behavioral_test.go` — explicitly plan 05-08's scope, not this plan's), #16 (bench comparison-runner framing — Phase 6 BENCH-02's scope by design), and the new #17 (`internal/cli/root.go` — a future sweep pass's scope).
- Re-running 05-VERIFICATION.md's CODE-01 census with the wrap-tolerant instrument should now find zero residual framing in this plan's 9 declared files.

## Self-Check: PASSED

All 11 claimed files verified present on disk (9 declared source files + WINDOWS.md + this SUMMARY). All 3 task commit hashes (`f9b8ce5`, `1432626`, `8094f15`) verified present in `git log`.

---
*Phase: 05-process-ci-in-tree-sweep*
*Completed: 2026-08-16*
