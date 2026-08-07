---
phase: 03-2026-07-28-spec-compliance
plan: 02
subsystem: testing
tags: [ci, anti-regeneration-guard, wire-oracle, go]

# Dependency graph
requires:
  - phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface
    provides: "The D-03 anti-regeneration guard and its self-expiring sdkSwapExemption (02-CONTEXT.md D-01/D-02/D-03), which this plan builds on and whose remedy this plan found structurally unavailable for Phase 3."
provides:
  - "The D-03 anti-regeneration guard's advisory (not blocking) posture for every future regeneration, ending the per-phase exemption treadmill 02-02 started."
  - "run()'s first direct test coverage, via a testable runCLI(args, stderr) seam."
  - "A recorded maintainer decision (option-advisory) that later phases (Phase 5 / SPEC-09) do not need to re-litigate."
affects: [05-subscriptions-list-changed, transcriptfreeze, ci-workflows]

actuals:
  tokens: 4174
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Testable CLI entrypoint via a fresh flag.FlagSet + io.Writer seam (runCLI(args, stderr) int), rather than the global flag.CommandLine/os.Stderr — lets run()'s exit-code and report contract be asserted directly without spawning the binary."
    - "Mutation-then-revert as the standing proof that a report-only (no longer blocking) guard still detects: flip the collision predicate, observe the new run() test go RED, revert, confirm a byte-clean diff."

key-files:
  created:
    - tools/transcriptfreeze/main_test.go
  modified:
    - tools/transcriptfreeze/classify.go
    - tools/transcriptfreeze/main.go
    - Taskfile.yml
    - .github/workflows/ci.yml

key-decisions:
  - "The D-03 anti-regeneration guard becomes advisory rather than blocking. Its own prescribed remedy — split into two pull requests — is structurally unavailable: PR-1 (server change, no transcripts) fails TestFrozenTranscriptsMatch, which is a required PR leg, so it can never merge to create PR-2's base. Phase 2 answered this with a one-time exemption keyed to a go.mod diff shape that occurs exactly once in this repository's history; Phase 3 has no equivalent shape and Phase 5 would need a third waiver. Doing nothing was not available either — if Phase 3's PR merges from a base still containing Phase 2's go.mod swap, the guard exits 0 for the wrong reason and misattributes Phase 3's regeneration to SDK-01's dependency transition. Advisory is merge-timing-independent, ends the exemption treadmill, keeps every byte of the detection and the report a reviewer actually reads, and mirrors this milestone's own VULN-03 principle that a job's blocking-versus-advisory stance must be stated explicitly so an advisory job is never mistaken for a gate. The real Trap C mitigation is unchanged and remains mandatory under D-06: capture before and after, read every changed line, name every cause in the commit message — the pass that caught Phase 2's ninth, unpredicted semantic regression. Accepted by the maintainer at 03-02's Task 1 checkpoint."
  - "requirements-completed intentionally left empty for this plan. 03-02's frontmatter lists SPEC-01, SPEC-02, SPEC-05, SPEC-07, but this plan's actual deliverable is the anti-regeneration guard's blocking-vs-advisory stance — it implements or verifies none of those SPEC behaviors (discover capabilities, per-request _meta validation, live catalog re-check, the instructions field). Marking them complete here would misrepresent REQUIREMENTS.md's traceability table; the requirement IDs likely reached this plan's frontmatter because they name the surfaces whose transcripts this guard's decision governs, not because this plan builds them. Left as Pending for the plans that actually implement them (03-01, 03-03, 03-04, 03-05)."

patterns-established:
  - "A CI guard's blocking-vs-advisory stance is stated in both the Taskfile task description's first clause and the CI step name, never left implicit in behavior alone."

requirements-completed: []

coverage:
  - id: D1
    description: "A detected D-03 collision (transcript + internal/mcp source change together) reports both offending sides in full but no longer fails the build — implemented in run()/buildReason and proven via runCLI's direct test coverage."
    verification:
      - kind: unit
        ref: "tools/transcriptfreeze/main_test.go#TestRunReportsCollisionButDoesNotFail"
        status: pass
      - kind: unit
        ref: "tools/transcriptfreeze/classify_test.go#TestClassifyFlagsTranscriptPlusServerChange"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every pre-existing classify_test.go assertion (11 tests) still passes unweakened — Classify's predicate and Verdict.Violation's meaning are untouched by the advisory change."
    verification:
      - kind: unit
        ref: "go test ./tools/transcriptfreeze/... -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "The blocking-vs-advisory stance is stated unambiguously in both Taskfile.yml's check:transcript-freeze description and .github/workflows/ci.yml's step name, so an advisory job cannot be mistaken for a gate."
    verification:
      - kind: other
        ref: "rg -n 'check:transcript-freeze' -A 3 Taskfile.yml; rg -n 'Anti-regeneration guard' .github/workflows/ci.yml"
        status: pass
    human_judgment: false
  - id: D4
    description: "The detection survives the advisory change — demonstrated RED-equivalent against a confirmed-applied mutation of Classify's collision predicate, then reverted to a byte-clean diff."
    verification:
      - kind: other
        ref: "manual mutation test, quoted below in Deviations/Verification section"
        status: pass
    human_judgment: false

duration: 2min (task-execution wall clock across 3 commits; excludes reading/research time)
completed: 2026-08-06
status: complete
---

# Phase 3 Plan 02: D-03 Anti-Regeneration Guard Goes Advisory Summary

**The D-03 anti-regeneration guard stops blocking pull requests and starts reporting: `run()` still detects and names both sides of every collision, but exits 0 instead of 1, because its own "split into two PRs" remedy is structurally unavailable in this repository.**

## Performance

- **Duration:** ~2 min of task-commit wall clock (3 commits, 11:27:23–11:28:35 local)
- **Started:** 2026-08-06T15:27:23Z (first commit)
- **Completed:** 2026-08-06T15:28:35Z (last commit)
- **Tasks:** 2 (Task 1: checkpoint:decision, pre-answered by the maintainer; Task 2: implementation)
- **Files modified:** 4 modified, 1 created

## Accomplishments
- Implemented the maintainer's Task 1 decision (`option-advisory`): a detected D-03 collision no longer fails the build, but the full `Verdict.Reason` — naming both offending sides — is still printed to stderr.
- Gave `run()` its first direct test coverage via a new, narrow `runCLI(args []string, stderr io.Writer) int` seam, added through a proper RED→GREEN commit sequence (the new `main_test.go` was committed first, referencing the not-yet-existing `runCLI` and failing to build; the implementation commit that followed turned it green).
- Rewrote `buildReason`'s closing sentence to stop instructing the author to do something impossible (split the PR) and instead name the reviewed-diff pass (D-06) as the control that applies — while leaving the "floor, not a proof of innocence" disclosure (Phase 1 D-03's deliberate mitigation) untouched.
- Stated the new advisory stance explicitly in both `Taskfile.yml`'s `check:transcript-freeze` description and `.github/workflows/ci.yml`'s step name, changing only those two strings.
- Proved detection survives the change: mutated `Classify`'s collision predicate (`len(transcripts) > 1` instead of `> 0`), observed six tests go RED (five pre-existing classifier tests plus the new `TestRunReportsCollisionButDoesNotFail`), reverted, and confirmed a byte-clean `git diff` plus a green suite.

## Task Commits

Each task was committed atomically, with the plan-level TDD gate honored via a real RED→GREEN sequence (not just a passing-test-after-the-fact commit):

1. **Task 2 (RED)**: `test(03-02): add failing run() tests for advisory D-03 guard behavior` — `2e0df42`. `main_test.go` created against the *original* `main.go`/`classify.go` (temporarily reverted via `git checkout --`); `go build ./tools/transcriptfreeze/...` failed with `undefined: runCLI`, confirmed and quoted below.
2. **Task 2 (GREEN)**: `feat(03-02): make D-03 anti-regeneration guard advisory, not blocking` — `51f47af`. Re-applied the `main.go`/`classify.go` changes (the `runCLI` seam, the exit-code change, `buildReason`'s rewritten closing sentence); all tests pass.
3. **Task 2 (stance)**: `docs(03-02): state the D-03 guard's advisory stance in Taskfile and CI` — `855b8e2`. `Taskfile.yml`'s `desc:` and `ci.yml`'s step `name:` only.

**Task 1** (`checkpoint:decision`) required no commit of its own — it was pre-answered by the maintainer in the execution context (`option-advisory`) and is recorded verbatim in Decisions above.

## Files Created/Modified
- `tools/transcriptfreeze/main_test.go` — new: `TestRunReportsCollisionButDoesNotFail`, `TestRunTranscriptOnlyIsClean`, `TestRunUnreadableInputIsExitTwo`, `TestRunMalformedRecordIsExitTwo`, plus a `writeTemp` test helper
- `tools/transcriptfreeze/main.go` — `run()` now delegates to a testable `runCLI(args []string, stderr io.Writer) int`; the `Violation` branch prints `Reason` but returns 0 instead of 1; file doc comment's exit-code table rewritten for the new 0/2 (no longer 0/1/2) contract
- `tools/transcriptfreeze/classify.go` — package doc comment and `buildReason` updated to state the advisory stance and point to the reviewed-diff pass instead of the now-impossible two-PR split; `Classify`'s predicate, `Verdict`, `sdkSwapExemption`, and `buildSwapExemptionNotice` all byte-unchanged
- `Taskfile.yml` — `check:transcript-freeze`'s `desc:` rewritten; `cmds:` body byte-unchanged
- `.github/workflows/ci.yml` — the `transcript-freeze` job's step `name:` renamed from "...blocking)" to "...advisory since 03-02)"; `if:`, runner, `TRANSCRIPT_FREEZE_BASE` env indirection, and `run:` body all byte-unchanged (confirmed via `git diff` grep in Verification below)

## Decisions Made

See `key-decisions` in frontmatter for the verbatim Task 1 decision record (`option-advisory`) and the reasoning for leaving `requirements-completed` empty on this plan despite its frontmatter listing SPEC-01/02/05/07.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - missing critical documentation accuracy] Package doc comment's stale "still belongs in its own separate, reviewable pull request" clause**

- **Found during:** Task 2, while implementing `buildReason`'s rewrite
- **Issue:** `classify.go`'s top-of-file package doc comment stated "every other regeneration, including Phase 3's, still belongs in its own separate, reviewable pull request" — a sentence that becomes false the moment the advisory decision lands, and the plan's `<action>` text only explicitly named `buildReason`'s closing sentence for rewriting, not this comment.
- **Fix:** Rewrote the package doc comment's closing paragraph to state the advisory stance and its rationale (03-CONTEXT.md D-06's structural unavailability of the two-PR split), matching what `buildReason` now says, so the file's own top-level documentation does not contradict its own runtime behavior.
- **Files modified:** `tools/transcriptfreeze/classify.go`
- **Verification:** `rg -c 'Split this into two pull requests' tools/transcriptfreeze/classify.go` = 0; reviewed by re-reading the full updated comment.
- **Committed in:** `51f47af` (Task 2 GREEN commit)

---

**Total deviations:** 1 auto-fixed (Rule 2 — documentation accuracy, not a behavior change)
**Impact on plan:** No scope creep; this brought the file's own top-of-file documentation in line with the plan's explicit decision. No prohibited item was touched (`Classify`'s predicate, `sdkSwapExemption`, `Verdict`, `ErrNoInput`, path constants — all untouched).

## Issues Encountered

- `go build ./...` (a general sanity check, not part of the plan's `<verify>`) writes multi-main-package output binaries into the current directory rather than a build cache when run across packages including `tools/transcriptfreeze`. A stray `transcriptfreeze` binary appeared twice in `git status --short` and was removed both times before committing — it is not part of any commit.
- The RED→GREEN commit split required temporarily reverting `main.go`/`classify.go` to `HEAD` via `git checkout --` (the sanctioned single-file revert, not a blanket reset), confirming the build failure, committing the test file, then reapplying the exact same edits. This is a deliberate, git-log-visible RED gate — not a shortcut around the plan's `tdd="true"` requirement.

## Verification — quoted per acceptance criteria

**Test suite (all 15 tests: 11 pre-existing + 4 new):**
```
$ go test ./tools/transcriptfreeze/... -count=1
ok  	github.com/seanb4t/codegraph-go/tools/transcriptfreeze	0.119s
```

**RED confirmation (before the GREEN commit, `main.go`/`classify.go` reverted to `HEAD`):**
```
$ go build ./tools/transcriptfreeze/...
# github.com/seanb4t/codegraph-go/tools/transcriptfreeze [github.com/seanb4t/codegraph-go/tools/transcriptfreeze.test]
tools/transcriptfreeze/main_test.go:35:9: undefined: runCLI
tools/transcriptfreeze/main_test.go:62:9: undefined: runCLI
tools/transcriptfreeze/main_test.go:80:9: undefined: runCLI
tools/transcriptfreeze/main_test.go:99:9: undefined: runCLI
FAIL	github.com/seanb4t/codegraph-go/tools/transcriptfreeze [build failed]
```

**Mutation-then-revert detection proof** (`Classify`'s collision predicate mutated from `len(transcripts) > 0` to `len(transcripts) > 1`):
```
$ go test ./tools/transcriptfreeze/... -count=1
--- FAIL: TestClassifyFlagsTranscriptPlusServerChange (0.00s)
--- FAIL: TestClassify_TranscriptPlusGoModMCPBumpIsError/added_MCP_SDK_line_is_a_violation (0.00s)
--- FAIL: TestClassifySDKSwapExemption (0.00s)
    --- FAIL: .../full_swap_plus_transcript+internal/mcp_change_is_exempted
    --- FAIL: .../go-sdk_added_without_mark3labs_removed_is_still_Violation:_true
    --- FAIL: .../mark3labs_removed_without_go-sdk_added_is_still_Violation:_true
    --- FAIL: .../empty_go.mod_diff_is_still_Violation:_true...
--- FAIL: TestClassifyReproducesTheWiredGuardsDemonstratedRedGreenPair (0.00s)
--- FAIL: TestRunReportsCollisionButDoesNotFail (0.00s)
    main_test.go:42: runCLI(...) wrote nothing to stderr, want a non-empty collision report
FAIL
```
Reverted (`len(transcripts) > 0` restored):
```
$ go test ./tools/transcriptfreeze/... -count=1
ok  	github.com/seanb4t/codegraph-go/tools/transcriptfreeze	0.067s

$ git diff -- tools/transcriptfreeze/classify.go
(only the intentional package-doc-comment / buildReason wording changes remain — the predicate line itself is byte-identical to HEAD)
```

**`go vet`, `actionlint`, and the Taskfile-owns-every-job-body test:**
```
$ go vet ./tools/transcriptfreeze/...
(clean)
$ go tool -modfile=go.tool-lint.mod actionlint .github/workflows/ci.yml
(exit 0)
$ go test ./internal/... -count=1 -run TestWorkflowRunBodiesInvokeTask
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.143s
```

**The wired `task check:transcript-freeze`, run against the actual current tree (exit code and full stderr, per the last acceptance criterion):**
```
$ TRANSCRIPT_FREEZE_BASE=origin/main task check:transcript-freeze
D-03 anti-regeneration EXEMPTED (SDK-01 swap): this pull request changes frozen transcript(s)
[testdata/wireoracle/transcripts/call-callees.golden, ... 23 transcripts total ...] together with
internal/mcp source file(s) [internal/mcp/archtest/protocol_version_selftest_test.go, ...
internal/mcp/tools_schema_drift_test.go] and the MCP dependency line in go.mod. This is SDK-01's
one-time github.com/mark3labs/mcp-go -> github.com/modelcontextprotocol/go-sdk transition.
02-CONTEXT.md D-01/D-02/D-03 replaced this guard's byte-identity bar with semantic equivalence
plus one human diff read, so these transcripts moving is expected, not a violation. This exemption
is self-expiring: once github.com/mark3labs/mcp-go is absent from go.mod, no future diff can
reproduce a removal line for it, so this exact waiver can fire at most once in this repository's
history.
EXIT_CODE=0
```
This is exactly the outcome 03-RESEARCH.md's Pitfall 4 predicted: `origin/main`'s merge-base still predates Phase 2's `go.mod` swap, so `sdkSwapExemption` fires first and the advisory branch is never reached in this particular run. **What the same command will do once Phase 2's `go.mod` swap is no longer inside the merge-base diff** (i.e., once `main` has advanced past that swap): `sdkSwapExemption` will no longer fire (no `-mark3labs`/`+go-sdk` lines exist in that future diff), so `Classify` will report `Violation: true` instead, and `run()` will print the same 23-transcript-plus-13-file collision list under the new `"D-03 anti-regeneration collision (advisory, not blocking, since 03-02)"` heading — but still **exit 0**, because the advisory branch this plan added also returns 0. The exit code is identical either way; only the report's heading and its stated rationale change.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The D-03 guard's Phase 3/5 behavior is now settled and merge-timing-independent — 03-05 (which regenerates the frozen corpus per D-06) and Phase 5 (SPEC-09) both inherit an advisory guard with no further exemption work needed.
- `internal/mcp/`, `test/wireoracle/`, and `testdata/wireoracle/transcripts/` were not touched — 03-01 (parallel wave-1 plan) and the later transcript-regenerating plans (03-05) have a clean, non-conflicting base.
- No blockers. The one open item is informational, not actionable: `requirements-completed` is empty on this plan by design (see Decisions) — SPEC-01/02/05/07 remain `Pending` in REQUIREMENTS.md until the plans that actually implement them complete.

## Self-Check: PASSED

All claimed files verified present: `tools/transcriptfreeze/main_test.go`, `tools/transcriptfreeze/main.go`, `tools/transcriptfreeze/classify.go`, `Taskfile.yml`, `.github/workflows/ci.yml`. All three commit hashes (`2e0df42`, `51f47af`, `855b8e2`) confirmed present in `git log --oneline -5`.

---
*Phase: 03-2026-07-28-spec-compliance*
*Completed: 2026-08-06*
