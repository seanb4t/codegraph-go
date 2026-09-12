---
phase: 08-tmux-real-pty-harness
plan: 03
subsystem: testing
tags: [ci, github-actions, tmux, taskfile-shape-guards, anti-vacuity]

requires:
  - phase: 08-tmux-real-pty-harness (plan 01)
    provides: "test/tmux harness spine, task test:tmux target (TMUX_EXPECTED_TESTS=5, TMUX_EXPECTED_VERSION sentinel)"
  - phase: 08-tmux-real-pty-harness (plan 02)
    provides: "The three remaining assertion classes (TTY-04/05/06); test/tmux holds exactly 5 top-level Test* functions"
provides:
  - "tmux-e2e job in .github/workflows/ci.yml (ubuntu-latest, D-10) — two bare task-call run: bodies, CI carried via step-level env:"
  - "inScopeJobs entry for tmux-e2e in internal/upgrade/taskfile_shape_test.go, added in the same commit as the job"
  - "Explicit, recorded bootstrap-deferred state for TMUX_EXPECTED_VERSION — sentinel intentionally left in place, no value guessed"
affects: [08-04-mutation-log]

actuals:
  tokens: 1086
  tasks: 2
  commits: 1

tech-stack:
  added: []
  patterns:
    - "Job-level env: CI: \"1\" carrying a strict-mode trigger through a bare `task <target>` run: body, matching transcript-freeze's TRANSCRIPT_FREEZE_BASE precedent — never interpolated into the run: line, per taskCallLineRe's exact-match constraint"
    - "Install-then-assert CI-version pinning (D-11) deliberately bootstrapped in TWO commits: this one lands the job with the sentinel still in place; a later commit (after a real ci.yml run exists) commits the observed tmux -V string"

key-files:
  created: []
  modified:
    - .github/workflows/ci.yml
    - internal/upgrade/taskfile_shape_test.go

key-decisions:
  - "Task 2 deferred branch taken, not the real-version branch — verified via `gh run list --workflow ci.yml --branch gsd/v0.13.0-guard-hardening-ui-follow-through --limit 5 --json databaseId,conclusion,headSha,status`, which returned `[]`: no ci.yml run exists for this branch (ci.yml triggers only on pull_request and push to main; no PR is open). Taskfile.yml's TMUX_EXPECTED_VERSION sentinel (UNPINNED-BOOTSTRAP) is left byte-unchanged, exactly as Task 2's own <action> specifies for this case — this is the plan's designed, legitimate completion path, not a workaround."
  - "Own explanatory-comment substring-proxy self-correction: the job's header comment originally used the literal phrase \"No continue-on-error and no if:\" to document what the job deliberately omits — this tripped the plan's own verify gate (`rg -o 'continue-on-error' ci.yml | wc -l` must equal 2, the two pre-existing occurrences). This is the same substring-proxy shape flagged from 08-01 (time.Sleep) and 08-02 (sha256sum): a legitimate check for a real property (no soft-fail escape hatch on this job) tripped by prose, not by an actual YAML directive. Fixed by rewording the comment to describe the same property without the literal substring, never by weakening the gate."

requirements-completed: [TTY-07]

coverage:
  - id: D1
    description: "tmux-e2e job lands in ci.yml (ubuntu-latest, D-10) with exactly two bare-task-call run: bodies (task test:tmux:install, task test:tmux), CI carried via step-level env:, no continue-on-error/if: anywhere on the job — and inScopeJobs gains its matching entry in the SAME commit, with runBodyExceptions and requiredCheckNames byte-unchanged"
    requirement: TTY-07
    verification:
      - kind: integration
        ref: "internal/upgrade/taskfile_shape_test.go#TestWorkflowRunBodiesInvokeTask"
        status: pass
      - kind: integration
        ref: "internal/upgrade/taskfile_shape_test.go#TestInScopeJobsPopulationMatchesDisk"
        status: pass
      - kind: integration
        ref: "internal/upgrade/taskfile_shape_test.go#TestRequiredCheckNamesPreserved"
        status: pass
      - kind: integration
        ref: "internal/upgrade/taskfile_shape_test.go#TestWorkflowFilePopulationMatchesDisk"
        status: pass
      - kind: other
        ref: "GOTOOLCHAIN=go1.26.6 task lint:actions"
        status: pass
    human_judgment: false
  - id: D2
    description: "TMUX_EXPECTED_VERSION bootstrap: verified no real ci.yml run exists yet for this branch, so the UNPINNED-BOOTSTRAP sentinel is left in place rather than guessed — the deliberate first half of D-11's two-step bootstrap"
    requirement: TTY-07
    verification: []
    human_judgment: true
    rationale: "This deliverable's other half — reading the tmux-e2e job's first real CI log, confirming the version assertion and the executed-count=5 line actually fired, and (if still on the sentinel) committing the real observed tmux -V string — cannot be completed until a PR triggers a real ci.yml run. The task's own <human-check> names this explicitly; it is unrun by design, not skipped."

duration: ~20min
completed: 2026-09-10
status: complete
---

# Phase 8 Plan 3: CI Wiring for the tmux Real-PTY Harness Summary

**`tmux-e2e` job lands in `ci.yml` on `ubuntu-latest` with two bare `task <target>` run bodies and a same-commit `inScopeJobs` fixture entry, turning `test/tmux`'s 5 assertions from a build-tagged package nothing ever runs into an actual, un-bypassable PR/push gate — with `TMUX_EXPECTED_VERSION`'s bootstrap deliberately left unresolved pending the first real CI run.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-09-10T14:53:48Z (approx., prior plan's close)
- **Completed:** 2026-09-10T15:13:26Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- `tmux-e2e` (`tmux e2e (real-pty harness, TTY-01..TTY-07)`) added to `.github/workflows/ci.yml`, placed immediately after `perf-regression` (both `ubuntu-latest`, D-10). Five steps: Checkout and Set up Go with the exact pinned SHAs copied verbatim from `perf-regression`, `go-version-file: go.mod` (never `stable` — the same lesson this workflow already paid for once in the `govulncheck` job), Install Task, `Install tmux` (`run: task test:tmux:install`), and `tmux real-pty harness (TTY-01..TTY-07)` (step-level `env: CI: "1"`, `run: task test:tmux`). No `continue-on-error`, no `if:` — the executed-count equality inside `task test:tmux` is what actually gates the job, not its exit code in isolation.
- `internal/upgrade/taskfile_shape_test.go`'s `inScopeJobs` gained `{Workflow: "ci.yml", JobID: "tmux-e2e"}` in the same commit — `TestInScopeJobsPopulationMatchesDisk` never observed the job on disk without a matching fixture entry. `runBodyExceptions` and `requiredCheckNames` are byte-unchanged; both this workflow-shape guard and `task lint:actions` pass green.
- Task 2's precondition (a real `ci.yml` run of `tmux-e2e` on this branch) was evaluated and confirmed unmet — `gh run list --workflow ci.yml --branch gsd/v0.13.0-guard-hardening-ui-follow-through --limit 5 --json databaseId,conclusion,headSha,status` returned `[]`. Per the task's own design this is a legitimate completion state, not a blocker: `Taskfile.yml`'s `TMUX_EXPECTED_VERSION` sentinel (`UNPINNED-BOOTSTRAP`) is left byte-unchanged, and the deferral is recorded here rather than a value being guessed from this dev machine's tmux 3.7c or a web search.

## Task Commits

1. **Task 1: The `tmux-e2e` CI job and its same-commit `inScopeJobs` entry** - `f4584c2f` (feat)
2. **Task 2: Bootstrap the pinned tmux version from a real runner, or record the deferral explicitly** - no commit (deferred branch taken; `Taskfile.yml` byte-unchanged, confirmed via `git diff --quiet -- Taskfile.yml`)

## Files Created/Modified

- `.github/workflows/ci.yml` - added the `tmux-e2e` job (5 steps: Checkout, Set up Go, Install Task, Install tmux, tmux real-pty harness)
- `internal/upgrade/taskfile_shape_test.go` - `inScopeJobs` gained one literal entry, `{Workflow: "ci.yml", JobID: "tmux-e2e"}`

## Decisions Made

- **Task 2's deferred branch taken, not the real-version branch.** Verified via `gh run list` returning `[]` for this branch — no PR is open, and `ci.yml` triggers only on `pull_request`/`push:main`, so no run can exist yet. `Taskfile.yml`'s `TMUX_EXPECTED_VERSION` sentinel stays `UNPINNED-BOOTSTRAP`, exactly as the task's own `<action>` specifies for this case: "do NOT invent a value... Leave the sentinel in place, and record the deferral explicitly."
- **Own explanatory-comment substring-proxy self-correction, before any commit.** The job's own header comment initially wrote "No continue-on-error and no if:" to document what the job deliberately omits — this tripped the plan's own Task 1 verify gate (`rg -o 'continue-on-error' ci.yml | wc -l` must equal exactly 2, the file's two pre-existing occurrences at the header comment line 18 and the reproducibility arm64 leg line 384). This is the identical substring-proxy shape already flagged twice in this phase (08-01's `time.Sleep`, 08-02's `sha256sum`): a legitimate check for a real property tripped by explanatory prose, not by an actual YAML directive. Fixed by rewording the comment to state the same property ("Nothing on this job or any of its steps can turn a failure soft or make a step conditional") without the literal substring — never by weakening the gate or adding a `runBodyExceptions`-style carve-out.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Own new comment tripped the plan's own `continue-on-error` substring-count verify gate**
- **Found during:** Task 1, first run of the plan's own `<verify>` block
- **Issue:** The `tmux-e2e` job's header comment, written to explain why the job carries no `continue-on-error` or `if:` condition, used the literal substring `continue-on-error` in prose — pushing the file's total occurrence count from 2 (the pre-existing baseline) to 3, which the plan's own verify command (`test "$(rg -o 'continue-on-error' ci.yml | wc -l)" = "2"`) treats as "the new job was allowed to fail soft," even though no actual `continue-on-error:` YAML key was ever added.
- **Fix:** Reworded the comment to describe the identical property ("Nothing on this job or any of its steps can turn a failure soft or make a step conditional") without the literal token — same meaning, no substring trip. The real property (no soft-fail escape hatch on this job) was true throughout; only the prose describing it needed to change.
- **Files modified:** .github/workflows/ci.yml
- **Verification:** `test "$(rg -o 'continue-on-error' .github/workflows/ci.yml | wc -l | tr -d ' ')" = "2"` passes; `task lint:actions` and all four `internal/upgrade` workflow-shape tests still pass.
- **Commit:** f4584c2f (Task 1's single commit — fixed before committing, no separate corrective commit needed)

---

**Total deviations:** 1 auto-fixed (1 bug — own comment tripped a legitimate gate, fixed honestly per the phase's own standing rule against contorting code/prose to dodge a grep without weakening the check).
**Impact on plan:** Confined entirely to comment prose in the new job; no functional change, no gate weakened, no scope creep beyond the plan's own declared `files_modified`.

## Issues Encountered

None beyond the one auto-fixed deviation documented above, resolved before Task 1's single commit.

## User Setup Required

**External repository-settings action required — NOT completed, and NOT self-approved.** This plan's `user_setup` block names a step no agent can perform: adding the required-status-check context `tmux e2e (real-pty harness, TTY-01..TTY-07)` to GitHub ruleset `20157557` (GitHub → repo Settings → Rules → Rulesets → 20157557 → Require status checks to pass). Per the checkpoint protocol, this is treated as a `blocking-human` gate — it is NOT marked done and the fixture was NOT edited as though it were. Concretely:

- `internal/upgrade/taskfile_shape_test.go`'s `requiredCheckNames` was deliberately left byte-unchanged (per the plan's explicit prohibition) — it must not name `tmux e2e (real-pty harness, TTY-01..TTY-07)` until the live ruleset actually requires it.
- **Two follow-up actions remain, both external to this plan and this repo checkout:**
  1. Add the required-status-check context to ruleset `20157557` (GitHub UI, human-only action).
  2. After that, re-verify with `gh api repos/seanb4t/codegraph-go/rulesets/20157557` and add the same string to `requiredCheckNames` in `internal/upgrade/taskfile_shape_test.go` in a follow-up commit, so the in-repo fixture stops under-reporting the real gate set.

## Next Phase Readiness

- `test/tmux`'s 5 assertions are now wired into `ci.yml` as a real, un-bypassable PR/push gate (`tmux-e2e`), not just a build-tagged package nothing runs — closing the "documented intent, never executed" gap this phase exists to close.
- **`TMUX_EXPECTED_VERSION` remains `UNPINNED-BOOTSTRAP`.** The first real `ci.yml` run of `tmux-e2e` (once a PR opens on this branch) is EXPECTED to fail on the version assertion — this is the designed bootstrap, not a defect. The next agent/session to touch this must: read that failing run's log, extract the observed `tmux -V` string from the mismatch line, commit it as `TMUX_EXPECTED_VERSION` in `Taskfile.yml`, and confirm the following run passes the version assertion with an executed count of 5. Both the mismatch line and the follow-up pass-count line should be pasted into whatever artifact tracks that follow-up (this plan does not — the branch taken here was the deferred one).
- **The ruleset `user_setup` item is still open** (see above) — `requiredCheckNames` is intentionally out of sync with the live ruleset until a human completes it. This is the single item this SUMMARY explicitly reports as blocking-human, not silently assumed.
- Both of `08-VALIDATION.md`'s Manual-Only rows (the version-constant match and the executed-count assertion actually firing) remain unanswered pending that first real CI run — they are the same deferred item as `TMUX_EXPECTED_VERSION`'s bootstrap, not a separate gap.
- `git diff --quiet -- internal/cli internal/cli/tui` exits 0 — this plan touched no product code, matching its own `<verification>` requirement.
- Ready for plan 08-04 (the four-family mutation log) once this phase's CI wiring is confirmed live.

## Self-Check: PASSED

`.github/workflows/ci.yml` and `internal/upgrade/taskfile_shape_test.go` confirmed modified on disk; commit `f4584c2f` confirmed present in `git log --oneline --all`. Task 2's no-commit outcome confirmed intentional: `git diff --quiet -- Taskfile.yml` exits 0 (sentinel byte-unchanged, as designed).

---
*Phase: 08-tmux-real-pty-harness*
*Completed: 2026-09-10*
