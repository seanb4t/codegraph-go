---
phase: 01-protocol-scoping-the-sdk-independent-wire-oracle
plan: 06
subsystem: testing
tags: [ci, github-actions, taskfile, anti-regeneration, wire-oracle, d-03]

# Dependency graph
requires:
  - phase: 01-protocol-scoping-the-sdk-independent-wire-oracle plan 01
    provides: "testdata/wireoracle/transcripts/handshake-explore.golden — the first frozen transcript this guard protects"
provides:
  - "tools/transcriptfreeze — pure, unit-tested D-03 cross-change classifier (Classify/ParseChangedList/MCPDependencyTouched) plus its CI entrypoint"
  - "task check:transcript-freeze — the single definition of the guard's body"
  - ".github/workflows/ci.yml transcript-freeze job — pull-request-only, blocking, merge-base-diff, fetch-depth 0"
  - "internal/upgrade/taskfile_shape_test.go inScopeJobs registration for transcript-freeze"
affects: [02-sdk-migration, 03-04-05-scenario-expansion, 07-mutation-matrix]

# Actuals (#2632)
actuals:
  tokens: 7525
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "package main co-locating a pure, I/O-free classifier (classify.go) with its thin CLI entrypoint (main.go) in one tools/-resident directory, following the tools/bench/runner precedent — no separate importable library package"
    - "ErrNoInput sentinel reserved for main.go's file-read failure only; ParseChangedList itself never conflates a clean empty diff with a malformed-but-read one — the two failure modes are kept categorically distinct end to end"
    - "Verdict.Reason states the trigger set's residual-risk disclosure inline in every violation message, not just in docs — the floor statement travels with the failure a contributor actually sees"

key-files:
  created:
    - tools/transcriptfreeze/classify.go
    - tools/transcriptfreeze/classify_test.go
    - tools/transcriptfreeze/main.go
  modified:
    - Taskfile.yml
    - .github/workflows/ci.yml
    - internal/upgrade/taskfile_shape_test.go

key-decisions:
  - "tools/transcriptfreeze is package main (not an importable library) — Task 1's classify.go and Task 2's main.go co-locate in one directory per the plan's artifact table (`tools/transcriptfreeze.Classify` etc.), following the tools/bench/runner precedent for a tools/-resident, Taskfile-invoked binary. To keep `go build ./...` and `go vet ./...` green after Task 1 alone (before main.go existed), classify.go carried a temporary `func main() {}` stub, removed the moment Task 2 added the real main.go in the same PR-equivalent sequence."
  - "ErrNoInput is reserved exclusively for main.go's own file-read failure (the input file is missing or unreadable) — ParseChangedList never returns it. An empty or blank-lines-only changed list returns ([]string{}, nil): a clean diff, not a failure. A non-empty but unparseable record (no tab, or an unrecognized git diff --name-status status letter) returns a distinct, dynamically-built error naming the offending line. This directly fixes the review concern the plan named: the pre-review draft errored on both empty and unreadable input, which would have made a clean local run against an identical HEAD and base fail instead of report clean."
  - "The mutation demonstration used a disposable, detached `git worktree add <tmp> HEAD --detach` rather than a scratch branch on the developer's checkout, per the plan's explicit instruction — removed with `git worktree remove --force` immediately after, confirmed via `git worktree list` and `git status --porcelain`."

patterns-established:
  - "Classify(changed []string, goModDiff string) Verdict — the two-input pure classifier shape any future D-03-style cross-change rule in this repo should follow: collect matching paths from a changed-file list, compute a separate dependency-diff-touched boolean, AND them together, and build a Reason that discloses the check's own limits rather than implying completeness."

requirements-completed: [VRFY-04]

coverage:
  - id: D1
    description: "CI fails when a pull request's merge-base diff touches both a frozen transcript and either internal/mcp/*.go or the MCP dependency line in go.mod; a pull request touching only one side passes; verified against the real wired task check:transcript-freeze in a disposable worktree, not just the classifier in isolation"
    requirement: "VRFY-04"
    verification:
      - kind: unit
        ref: "go test ./tools/transcriptfreeze/... -run TestClassifyFlagsTranscriptPlusServerChange"
        status: pass
      - kind: unit
        ref: "go test ./tools/transcriptfreeze/... -run TestClassifyReproducesTheWiredGuardsDemonstratedRedGreenPair"
        status: pass
      - kind: other
        ref: "task check:transcript-freeze run against a confirmed-applied cross-change commit in a disposable detached worktree (recorded verbatim below)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The guard's path predicates are proven to match the real tree (not stale), and every declared-firing rule in the taskfile-shape guard actually fires — the D-01 single-definition property extended to the new job with no runBodyExceptions entry needed"
    requirement: "VRFY-04"
    verification:
      - kind: unit
        ref: "go test ./tools/transcriptfreeze/... -run TestGuardPatternsMatchRealTree"
        status: pass
      - kind: unit
        ref: "go test ./internal/upgrade/... -run TestWorkflowRunBodiesInvokeTask"
        status: pass
    human_judgment: false
  - id: D3
    description: "The trigger set is disclosed as a floor, not a proof of innocence, consistent with docs/MCP-2026-07-28-SCOPING.md's own wording — naming internal/query and internal/indexer as recorded residual risk in every violation Reason"
    requirement: "VRFY-04"
    verification:
      - kind: unit
        ref: "go test ./tools/transcriptfreeze/... -run TestClassifyFlagsTranscriptPlusServerChange (asserts Reason contains 'floor', 'internal/query', 'internal/indexer')"
        status: pass
    human_judgment: false

duration: ~15min
completed: 2026-08-05
status: complete
---

# Phase 1 Plan 6: D-03 Anti-Regeneration CI Guard Summary

**A pure, unit-tested cross-change classifier (`tools/transcriptfreeze`) wired into `Taskfile.yml` and a pull-request-only, blocking CI job that fails when a frozen wire-oracle transcript changes together with `internal/mcp/*.go` or the MCP dependency line in `go.mod` — demonstrated red against a confirmed-applied mutation in a disposable git worktree, then encoded as a permanent regression test.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-08-05
- **Tasks:** 3
- **Files modified:** 6 (3 created, 3 modified)

## Accomplishments

- Built `tools/transcriptfreeze` (package `main`, following the `tools/bench/runner` precedent): a pure, I/O-free classifier (`Classify`, `ParseChangedList`, `MCPDependencyTouched`, `Verdict`) plus a thin CI entrypoint (`main.go`) with three distinct exit codes (0 clean, 1 violation, 2 unusable input).
- The classifier's narrow, locked trigger set: violation iff at least one frozen-transcript path AND (at least one `internal/mcp/*.go` path OR the MCP dependency line in `go.mod` was touched). Transcript-only and `internal/mcp`-only changes are both clean — the sanctioned Phase 3 regeneration path is not blocked.
- `ParseChangedList` keeps three outcomes categorically distinct: a clean empty/blank diff (`[]string{}, nil`), a malformed-but-read record (a dynamic error naming the offending line), and an unreadable file (`main.go`'s `ErrNoInput`, never returned by `ParseChangedList` itself) — directly closing the review concern that the pre-review draft conflated empty and unreadable input.
- `Verdict.Reason` states, in every violation message, that the trigger set is a deliberate floor, not a proof of innocence, naming `internal/query` and `internal/indexer` as recorded residual risk — worded consistently with `docs/MCP-2026-07-28-SCOPING.md`'s own "trigger set is a floor" section (plan 02), never contradicting or duplicating it.
- `git diff --name-status -M -C` (not `--name-only`) is what the Taskfile body captures, so both sides of a renamed transcript or a renamed transcripts directory reach the classifier — proven by `TestClassifyDetectsRenamedTranscript`.
- Wired `task check:transcript-freeze` (requires `TRANSCRIPT_FREEZE_BASE`, fails loud when unset, computes the merge-base diff for pull-request/squash-merge-safe granularity) and a `transcript-freeze` CI job (`pull_request`-only, `fetch-depth: 0`, base ref passed via step-level `env:`, no `${{ github.* }}` in the shell body) into `.github/workflows/ci.yml`, registered in `internal/upgrade/taskfile_shape_test.go`'s `inScopeJobs` with no `runBodyExceptions` entry needed.
- **Demonstrated the wired guard RED against a confirmed-applied cross-change** in a disposable, detached `git worktree` (never a scratch branch on the primary checkout) — see "Task 3 Demonstration" below — then encoded the exact file pair as a permanent regression test, `TestClassifyReproducesTheWiredGuardsDemonstratedRedGreenPair`.

## Task Commits

Each task was committed atomically:

1. **Task 1: The cross-change classifier — a pure function with a planted-defect companion** - `f00a8f1` (test) — `classify.go`/`classify_test.go`, all 8 named behavior tests plus `TestGuardPatternsMatchRealTree`; a temporary `func main() {}` stub kept `go build ./...` green until Task 2.
2. **Task 2: Wire the guard into Taskfile and CI at pull-request granularity** - `20f9054` (feat) — `main.go` replaces the Task 1 stub; `Taskfile.yml`'s `check:transcript-freeze`; `.github/workflows/ci.yml`'s `transcript-freeze` job; `inScopeJobs` registration.
3. **Task 3: Demonstrate the guard red against a confirmed-applied cross-change, then revert** - `210ff93` (test) — disposable worktree mutation demonstration (recorded below), permanent regression test added.

**Plan metadata:** (this commit, pending)

## Files Created/Modified

- `tools/transcriptfreeze/classify.go` - pure classifier: `Classify`, `Verdict`, `ParseChangedList`, `MCPDependencyTouched`, `ErrNoInput`, `mcpSDKModulePrefixes`, `transcriptDirPrefix`, `serverDirPrefix`
- `tools/transcriptfreeze/classify_test.go` - full behavior coverage, fail-loudly table, real-tree pattern check, and the Task 3 demonstration's permanent regression test
- `tools/transcriptfreeze/main.go` - CI entrypoint; exit 0/1/2 per the plan's exact contract
- `Taskfile.yml` - `check:transcript-freeze` target
- `.github/workflows/ci.yml` - `transcript-freeze` job, `pull_request`-only, blocking
- `internal/upgrade/taskfile_shape_test.go` - `inScopeJobs` gains `{Workflow: "ci.yml", JobID: "transcript-freeze"}`

## Task 3 Demonstration (recorded per acceptance criteria)

Disposable, detached worktree created with `git worktree add <tmp>/transcript-freeze-proof <base> --detach`, where `<base>` = `20f905447fa9add679be3efb9f1e04ae794220b5` (the commit after Task 2).

**1. Cross-change (expect RED):** appended a trailing newline to `testdata/wireoracle/transcripts/handshake-explore.golden` AND a comment line to `internal/mcp/server.go`, in one commit. Confirmed both paths landed before trusting the result:

```
$ git diff --name-status -M -C 20f9054 HEAD
M	internal/mcp/server.go
M	testdata/wireoracle/transcripts/handshake-explore.golden
```

`TRANSCRIPT_FREEZE_BASE=20f9054 task check:transcript-freeze` exited **1**, printing (verbatim):

```
D-03 anti-regeneration violation: this pull request changes frozen transcript(s) [testdata/wireoracle/transcripts/handshake-explore.golden] together with internal/mcp source file(s) [internal/mcp/server.go]. Split this into two pull requests: one for the protocol/server change, and a separate one that regenerates the frozen transcript(s) afterward. This trigger set is a deliberate floor, not a proof of innocence: transcript bytes also legitimately depend on internal/query and internal/indexer (and the tree-sitter grammars), which this guard does not watch — a clean pass means this PR did not commit the one shape the guard can detect mechanically, not that this regeneration was safe.
```

**2. Single-sided (transcript-only, expect GREEN):** reset to `<base>`, re-applied only the transcript edit, confirmed via `git diff --name-status -M -C` that only `testdata/wireoracle/transcripts/handshake-explore.golden` changed. `TRANSCRIPT_FREEZE_BASE=20f9054 task check:transcript-freeze` exited **0**.

**3. Empty diff (expect GREEN, not "unusable"):** reset to `<base>` with no further edits (merge-base diff touches nothing). `TRANSCRIPT_FREEZE_BASE=20f9054 task check:transcript-freeze` exited **0** — the regression `ParseChangedList`'s empty-versus-unreadable split exists to prevent.

**Cleanup:** `git worktree remove --force <tmp>/transcript-freeze-proof`. Afterward, `git worktree list` showed only the primary checkouts (no stranded proof worktree), and `git status --porcelain` in this agent's own worktree was empty.

**Permanent encoding:** `TestClassifyReproducesTheWiredGuardsDemonstratedRedGreenPair` in `classify_test.go` drives `Classify` through the exact same two file lists (cross-change → violation, transcript-only → clean), so the proof survives as a regression test rather than only as this summary paragraph.

## Decisions Made

See `key-decisions` in frontmatter — summarized:
- `tools/transcriptfreeze` is `package main`, co-locating the pure classifier and its CLI entrypoint in one directory (the artifact table's dotted notation names the directory, not an importable library path).
- `ErrNoInput` is reserved exclusively for `main.go`'s file-read failure; `ParseChangedList` never returns it, keeping "clean empty," "malformed but read," and "unreadable" as three permanently distinct outcomes.
- The mutation demonstration used a disposable detached worktree, never a scratch branch on this agent's own checkout.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Temporary `func main() {}` stub in `classify.go` during Task 1**
- **Found during:** Task 1 implementation
- **Issue:** The plan splits `classify.go` (Task 1) and `main.go` (Task 2) across two commits in the same `tools/transcriptfreeze` directory, which the artifact table establishes as `package main` (a standalone Go program, following the `tools/bench/runner` precedent). Without `main.go`, a `package main` directory with no `func main()` fails to link (`go build ./tools/transcriptfreeze/...` errors `function main is undeclared in the main package`), which would have broken Task 1's own acceptance criterion (`go build ./... && go vet ./... exit 0`).
- **Fix:** Added a minimal, explicitly-commented `func main() {}` stub to the end of `classify.go` in Task 1, documented as temporary. Removed it in Task 2's commit the moment the real `main.go` landed — no functional code was lost, and the stub never shipped alongside the real entrypoint.
- **Files modified:** `tools/transcriptfreeze/classify.go` (added in Task 1's commit `f00a8f1`, removed in Task 2's commit `20f9054`)
- **Verification:** `go build ./... && go vet ./...` passed after both Task 1 and Task 2's commits individually.
- **Committed in:** `f00a8f1` (added), `20f9054` (removed)

---

**Total deviations:** 1 auto-fixed (1 blocking — a build-config gap between two sequential same-directory commits)
**Impact on plan:** Necessary to keep every task's own acceptance criteria (`go build ./...` green) individually satisfiable. No scope creep; the stub carried zero production behavior and was removed in the very next commit.

## Issues Encountered

- `go build ./tools/transcriptfreeze/...` (run without `-o`) writes a `transcriptfreeze` binary into the current working directory as a side effect. Removed it before each commit (`rm -f transcriptfreeze`) so it never got staged; it is not part of the repository and requires no `.gitignore` entry beyond what's already conventional for stray local builds.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The guard is live from this commit forward: any future pull request that both changes a frozen transcript and touches `internal/mcp/*.go` or the MCP dependency line in `go.mod` will fail CI. Phase 2's SDK migration (plan depends_on: `01-01`) must regenerate transcripts, if needed, in its own separate pull request from the protocol/server change — the guard enforces exactly that split.
- `mcpSDKModulePrefixes` already forward-declares `github.com/modelcontextprotocol/go-sdk` alongside today's `github.com/mark3labs/mcp-go`, so Phase 2's dependency swap will be detected by this guard without any change to `tools/transcriptfreeze`.
- Plan 07's mutation matrix is the independent proof that the bootstrap transcript (frozen in plan 01, in the same change as its own `internal/mcp` seam — recorded there as an unavoidable, permission-granted exception) is not vacuous; this guard does not and cannot retroactively cover that bootstrap commit, by design (the guard did not exist yet).
- No blockers.

---
*Phase: 01-protocol-scoping-the-sdk-independent-wire-oracle*
*Completed: 2026-08-05*

## Self-Check: PASSED

All 3 created files (`tools/transcriptfreeze/classify.go`, `classify_test.go`,
`main.go`) verified present via `git ls-files --error-unmatch`. All 3 task
commits (`f00a8f1`, `20f9054`, `210ff93`) verified present in
`git log --oneline --all`.
