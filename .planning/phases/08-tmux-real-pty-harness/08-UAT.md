---
status: diagnosed
phase: 08-tmux-real-pty-harness
source: [08-01-SUMMARY.md, 08-02-SUMMARY.md, 08-03-SUMMARY.md, 08-04-SUMMARY.md, 08-VERIFICATION.md]
started: 2026-09-10T18:08:06.378Z
updated: 2026-09-11T20:37:58Z
---

## Current Test

[testing complete]

## Tests

### 1. TTY-03 tracer assertion is timing-flaky against a cold binary

expected: TestDaemonEmptyRegistryLeaksNoModeQueryBytes passes on every run at HEAD, including the first run on a cold machine, because pollUntilStable cannot converge on a pre-output frame.
result: issue
reported: "Found by Claude re-executing the covering check at HEAD, not by the user: FAILED 3/3 on a cold machine, PASSED ~13/13 once warm. First exec of a fresh binary = 1567 ms in-pane vs stabilityPollInterval = 1000 ms; pollUntilStable converges on the pre-output frame. capture.go's 'can never both land inside that window by construction' claim is false on the cold path."
severity: major
note: |
  Found by re-executing 08-01 D2's covering check at HEAD (5baba883). FAILED 3/3 consecutive
  runs at session start on a cold machine; PASSED ~13/13 afterwards once warm. Failure text:
  'TTY-03: stabilized capture does not contain "no running daemons"' with a capture showing the
  echoed command and no output — pollUntilStable converged on the pre-execution frame.
  Measured in-pane: first exec of a freshly built binary = 1567 ms; subsequent execs = 167-226 ms.
  stabilityPollInterval = 1000 ms (capture.go:29). TestMain builds a fresh 80MB binary per run
  and daemon_empty_test.go is the first test to exec it, so TTY-03 always pays the cold cost.
  capture.go:16-27 claims the interval "exceeds the measured worst case with margin" so samples
  "can never both land inside that window by construction" — that claim is false on the cold path.
  Not reproducible on demand once warm; reported with that limit stated.

### 2. TTY-07 backstop: the executed-count assertion fired on a real CI run

expected: The tmux-e2e job log shows "test:tmux: executed=5 skipped=0 expected=5", the job status is success, and the tmux -V assertion line shows the observed version matched TMUX_EXPECTED_VERSION (or, on the deliberately-unpinned first run, printed the real string for committing per D-11's bootstrap).
result: pass
source: ci-log
evidence: "PR #69, run 34607117422, job 103288150463 (ubuntu-latest, pin commit 3899e6de): log contains 'test:tmux: executed=5 skipped=0 expected=5'; tmux -V observed 'tmux 3.4' = committed TMUX_EXPECTED_VERSION; 0 ::error annotations; step 6 success; all 5 tests pass (TTY-03 2.17s, TTY-04 4.08s, TTY-05 11.1s, TTY-06 7.06s, TTY-01 skip-contract 0s); 'TTY-06: frame-stable across N=5 captures' echoed. Run 1 (34606828356) failed at the version pin BY DESIGN printing 'tmux 3.4' — D-11 bootstrap half 1; pin committed as half 2."
coverage_id: 08-03-D2

### 3. TTY-06 family (d) non-reproduction — human decision on scope

expected: A decision is recorded on whether the D-06-specified v.AltScreen=false mutation's failure to fail TestInstallPickerFrameStableWhileIdle warrants a follow-up settling-transient-observing assertion, or is accepted as a recorded limitation per Phase 7's D-07 precedent.
result: skipped
reason: "Deferred follow-up: defer"
coverage_id: 08-04-D4

### 4. test/tmux package spine: build tag, TestMain resolver, tmux argv wrappers, D-14 stability poll

expected: test/tmux package spine: build tag, TestMain binary resolver, tmux session/send-keys/capture-pane argv wrappers, D-14 stability poll
result: pass
source: automated
coverage_id: 08-01-D1

### 5. pollUntilStable — no fixed sleeps, converges on byte-identical consecutive captures

expected: pollUntilStable — no fixed sleeps, converges on byte-identical consecutive captures, t.Fatalf on non-convergence with both differing captures named
result: pass
source: automated
coverage_id: 08-01-D3

### 6. TestRequireTmuxReportsSkipReasonWhenAbsent — TTY-01 named, always-executed skip-contract test

expected: TestRequireTmuxReportsSkipReasonWhenAbsent — TTY-01's named, always-executed skip-contract test, deterministic executed-count in both environments
result: pass
source: automated
coverage_id: 08-01-D4

### 7. task test:tmux — D-01/D-02/D-03 exact-count anti-vacuity gate

expected: task test:tmux — prints executed+skipped counts on every run; enforces exact equality and D-11's tmux-version pin only under CI
result: pass
source: automated
coverage_id: 08-01-D5

### 8. TTY-04: daemon picker enters alt-screen over a really-seeded running daemon

expected: TTY-04 — alt-screen entry, real daemon start subprocess seeding, "Running daemons" plus the seeded repo basename, main-buffer restoration with no DECRQM residue after q
result: pass
source: automated
coverage_id: 08-02-D1

### 9. TTY-05: checkbox picker glyphs, space toggle, both cancel keys, zero-write proof

expected: TTY-05 — [ ]/[x] glyphs, space toggles, q and esc both cancel, whole-tree sha256 (pure Go) proves cancel writes zero config files
result: pass
source: automated
coverage_id: 08-02-D2

### 10. TTY-06: idle install picker holds byte-identical across 5 captures, N reported

expected: TTY-06 — idle install picker in a 12-row pane holds byte-identical across 5 further captures after its first settled frame, reporting N=5 via t.Logf into the CI job log
result: pass
source: automated
coverage_id: 08-02-D3

### 11. tmux-e2e job lands in ci.yml with matching same-commit inScopeJobs entry

expected: tmux-e2e job in ci.yml (ubuntu-latest) with exactly two bare-task-call run bodies, CI via step-level env, no continue-on-error/if on the job — and inScopeJobs gains its matching entry in the SAME commit, runBodyExceptions and requiredCheckNames byte-unchanged
result: pass
source: automated
coverage_id: 08-03-D1

### 12. Mutation family (a): TTY-03 watched fail against the historical G-07-1 DECRQM leak

expected: Both empty-registry guards mutated together reproduce the historical G-07-1 DECRQM leak against a real binary in a real tmux pane; RED pasted, revert + GREEN pasted
result: pass
source: automated
coverage_id: 08-04-D1

### 13. Mutation family (b): TTY-04 watched fail on v.AltScreen=false

expected: v.AltScreen=false on daemonpicker.go's View() fails the alternateOn signal loudly while the content assertion would have passed — confirming D-13's two-signal design
result: pass
source: automated
coverage_id: 08-04-D2

### 14. Mutation family (c): TTY-05 watched fail on the cancel branch

expected: Mutating agentpicker.go's cancel branch to confirmed=true reproduces a real cancel-writes-config defect — two differing whole-tree sha256 digests and four real .gemini/ config paths
result: pass
source: automated
coverage_id: 08-04-D3

### 15. Phase-wide byte-clean proof: no mutation survived into the phase's final state

expected: git status --porcelain over internal/cli and internal/cli/tui is empty, and git diff against the phase branch point over all three touched production files exits 0
result: pass
source: automated
coverage_id: 08-04-D5

## Summary

total: 15
passed: 13
issues: 1
pending: 0
skipped: 1
blocked: 0

## Deferred Follow-Ups

- test: 3
  idea: "Settling-transient-observing assertion for TTY-06: sample during convergence (or mutate something that renders when idle) so an inline-vs-altscreen defect confined to the settling window is discriminable. Family (d)'s v.AltScreen=false non-reproduction is recorded in 08-MUTATION-LOG.md and .planning/WINDOWS.md; user response verbatim: 'defer'."
  deferred_at: 2026-09-11

## Gaps

- gap_id: G-08-1
  truth: "TestDaemonEmptyRegistryLeaksNoModeQueryBytes passes on every run at HEAD, including the first run on a cold machine"
  status: failed
  reason: "Claude re-executed 08-01 D2's covering check at HEAD 5baba883: FAILED 3/3 at session start (cold machine), PASSED ~13/13 afterwards. In-pane first-exec latency of a freshly built binary = 1567 ms; subsequent = 167-226 ms; stabilityPollInterval = 1000 ms. TestMain builds a fresh binary per run and daemon_empty_test.go is the first test to exec it, so TTY-03 alone pays the cold cost and pollUntilStable converges on the pre-output frame. capture.go:16-27's 'by construction' margin claim does not hold. Not reproducible on demand once warm."
  severity: major
  test: 1
  root_cause: "stabilityPollInterval (1s) is below the cold first-exec latency of the freshly built 80MB binary (1.1-1.6s measured); pollUntilStable's two-sample check converges on the pre-output frame. CI evidence (PR #69 run 34607117422, ubuntu-latest, fresh build): TTY-03 PASSED in 2.17s — the race did NOT fire there (n=1). The defect stands as (a) capture.go's false 'by construction' margin claim and (b) 3/3 observed failures on a cold macOS dev machine; it is not, on this evidence, a CI-breaker."
  artifacts:
    - path: "test/tmux/capture.go"
      issue: "stabilityPollInterval=1s justified on a warm-binary measurement (700-900ms); cold path exceeds it"
    - path: "test/tmux/daemon_empty_test.go"
      issue: "first test to exec the freshly built binary; sole payer of the cold-start cost"
    - path: "test/tmux/main_test.go"
      issue: "TestMain builds a fresh binary per run and never warms it before m.Run()"
  missing:
    - "Either warm the binary once in TestMain before m.Run() (one throwaway exec), or make pollUntilStable require a positive anchor (e.g. the shell prompt returned) before accepting convergence, or raise the interval with a cold-path measurement recorded in the comment"
  debug_session: ".planning/debug/tty03-cold-start-poll-race.md"
