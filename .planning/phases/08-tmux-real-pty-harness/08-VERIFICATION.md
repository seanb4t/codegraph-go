---
phase: 08-tmux-real-pty-harness
verified: 2026-09-11T22:30:00Z
status: passed
score: 14/15 must-haves verified
behavior_unverified: 0
overrides_applied: 0
covered_files: [".github/workflows/ci.yml",".planning/REQUIREMENTS.md",".planning/phases/08-tmux-real-pty-harness/08-01-PLAN.md",".planning/phases/08-tmux-real-pty-harness/08-01-SUMMARY.md",".planning/phases/08-tmux-real-pty-harness/08-02-PLAN.md",".planning/phases/08-tmux-real-pty-harness/08-02-SUMMARY.md",".planning/phases/08-tmux-real-pty-harness/08-03-PLAN.md",".planning/phases/08-tmux-real-pty-harness/08-03-SUMMARY.md",".planning/phases/08-tmux-real-pty-harness/08-04-PLAN.md",".planning/phases/08-tmux-real-pty-harness/08-04-SUMMARY.md",".planning/phases/08-tmux-real-pty-harness/08-05-PLAN.md",".planning/phases/08-tmux-real-pty-harness/08-05-SUMMARY.md",".planning/phases/08-tmux-real-pty-harness/08-CONTEXT.md",".planning/phases/08-tmux-real-pty-harness/08-DISCUSSION-LOG.md",".planning/phases/08-tmux-real-pty-harness/08-MUTATION-LOG.md",".planning/phases/08-tmux-real-pty-harness/08-PATTERNS.md",".planning/phases/08-tmux-real-pty-harness/08-RESEARCH.md",".planning/phases/08-tmux-real-pty-harness/08-REVIEW.md",".planning/phases/08-tmux-real-pty-harness/08-SECURITY.md",".planning/phases/08-tmux-real-pty-harness/08-VALIDATION.md","Taskfile.yml","internal/upgrade/taskfile_shape_test.go","test/tmux/capture.go","test/tmux/confighash.go","test/tmux/daemon_empty_test.go","test/tmux/daemon_picker_test.go","test/tmux/daemon_seed.go","test/tmux/frame_stability_test.go","test/tmux/install_cancel_test.go","test/tmux/main_test.go","test/tmux/poll_contract_test.go","test/tmux/session.go","test/tmux/skip_contract_test.go"]
covered_digest: "v1:sha256:cde0a89d20e906bfea6a6b1f2de73d5f6a45ec13fe66f79e3be9b22179cb0116"
re_verification:
  previous_status: human_needed
  previous_score: 13/14
  gaps_closed:
    - "G-08-1: TestDaemonEmptyRegistryLeaksNoModeQueryBytes (TTY-03) no longer depends on the spawned binary's first exec landing inside a single stabilityPollInterval — pollUntilStable now requires a caller-stated readiness predicate (ready(capture) && capture == predecessor), structurally closing the cold-start race regardless of startup latency short of stabilityPollDeadline."
    - "TTY-02's SC #2 wording ('proven against a case that races and fails under a single naive capture') is now satisfied directly: TestPollUntilStableDoesNotConvergeOnPreOutputFrame is a deterministic self-test watched RED against the pre-fix primitive (converged at ~2.03s on the pre-output frame) and GREEN after the fix (converged at sample 5-6, past the commanded 4s delay) — reproduced independently in this pass."
  gaps_remaining: []
  regressions: []
human_verification:

  - test: "Push this branch's 8 unpushed commits (or otherwise get HEAD synced to PR #69) so ci.yml's tmux-e2e job runs on ubuntu-latest at a commit carrying TMUX_EXPECTED_TESTS=6, then read that run's job log for the line `test:tmux: executed=6 skipped=0 expected=6` and confirm the job went GREEN because that line printed a real 6, not because a step was skipped or the pipeline degraded silently."
    expected: "The tmux-e2e job log shows `test:tmux: executed=6 skipped=0 expected=6`, job conclusion is `success`, and the `tmux -V` line matches the committed TMUX_EXPECTED_VERSION (tmux 3.4)."
    why_human: "This is 08-03-PLAN.md's `verification: backstop` truth (an actual fired CI run proving the exact-count gate works), now stale relative to HEAD: the only CI evidence that exists (run 34607117422, PR #69, job 'tmux e2e', conclusion success) proves `executed=5 skipped=0 expected=5` at commit 3899e6de — the pre-08-05 code. `git rev-list --left-right --count HEAD...origin/gsd/v0.13.0-guard-hardening-ui-follow-through` shows HEAD (400b4c1d) is 8 commits ahead of, 0 behind, origin; `gh pr view 69 --json headRefOid` confirms the open PR's head is still 3899e6de. 08-05 bumped TMUX_EXPECTED_TESTS 5->6 and added a sixth top-level test; `task test:tmux` reports `executed=6 skipped=0 expected=6` locally (reproduced independently in this pass), but the exact-count gate has never fired against 6 in the real ubuntu-latest CI environment. This is a live external fact (a GitHub Actions run) that cannot be manufactured from the working tree."
---

# Phase 8: tmux Real-PTY Harness Verification Report

**Phase Goal:** The release binary's interactive TUI is exercised inside a real tmux pane that answers escape queries and actually scrolls — the missing rung between the piped, TTY-blind integration suite and manual human UAT.
**Verified:** 2026-09-11T22:30:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap closure (plan 08-05, gap_ids: [G-08-1])

## Goal Achievement

This is a full re-verification at HEAD (`400b4c1d`), not a diff against the prior pass. Every
truth below was independently re-checked against the current tree; where a command was
re-executed rather than read, that is stated. The prior `08-VERIFICATION.md` (2026-09-10,
`91e063a5`-era, score 13/14, `human_needed` on TTY-07's CI backstop alone) is superseded by
this report per the workflow's overwrite instruction — it is stale because plan 08-05 has
landed since (`a4590956`..`21063a2e`), closing gap G-08-1, and a code review (`400b4c1d`) has
also landed on top of it.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | TTY-01: harness builds the binary, spawns it in a real tmux pane, sends keys, captures, exits 0; skip contract fires with a reason when tmux is absent | ✓ VERIFIED (regression) | Neither `main_test.go`, `session.go`, nor `skip_contract_test.go` were touched by 08-05 (`git diff --quiet 5baba883..HEAD -- test/tmux/main_test.go` exits 0). Re-executed the full suite this pass (`GOTOOLCHAIN=go1.26.6 task test:tmux`): `executed=6 skipped=0 expected=6`, all named tests PASS including the TTY-01 skip-contract case. Not independently re-run under a tmux-stripped PATH this pass (unaffected surface; prior pass verified this directly). |
| 2 | TTY-02 (amended by 08-05): `pollUntilStable`'s convergence is `ready(capture) && capture == predecessor`; the readiness predicate is a required parameter (nil -> `t.Fatal`), an empty `paneContains` needle is an immediate `t.Fatal`; proven against a case that races and fails under a single naive capture | ✓ VERIFIED | Read `capture.go:141-201` in full: nil-check and empty-needle-check both present exactly as described. Re-executed `TestPollUntilStableDoesNotConvergeOnPreOutputFrame` directly: PASS, `converged at sample 6 after 6.04s (ready first held at sample 5)` — past the commanded 4s delay, matching the plan's acceptance criterion (K>=4, D>=4s). The pasted RED transcript in 08-05-SUMMARY.md (`converged... after 2.034s` against the pre-fix primitive, no anchor, at commit `e994b0a5`) was not re-run against the pre-fix code (that would require reverting `b9dfc849`), but the GREEN half was independently reproduced against the current tree, and `go vet -tags tmux ./test/tmux/...` (the compiler-enforced key link — no call site can compile without an anchor) passes clean. |
| 3 | TTY-03 (G-08-1 closed): bare `codegraph daemon` on empty registry leaks zero DECRQM bytes, and now passes on the FIRST exec of a freshly built binary regardless of cold-start latency short of the 10s deadline | ✓ VERIFIED | Independently ran `TestDaemonEmptyRegistryLeaksNoModeQueryBytes` three times, each its own process (fresh `TestMain` build, `CODEGRAPH_TEST_BIN` unset): all three PASS, all converged at sample 2 (~2.02s) with `ready first held at sample 1` — i.e. K=1 on this warm machine, matching 08-05-SUMMARY.md's own honestly-recorded caveat ("K<=1 on all three means the machine was warm and the self-test carries the proof"). Per the must-have's own acceptance rule, the deterministic self-test (truth #2) is what carries the proof for the genuine cold path, not these three runs — and that self-test was independently reproduced GREEN in this pass. Both assertion halves (positive `"no running daemons"` present, negative — neither DECRQM marker present) remain explicit in `daemon_empty_test.go`, unweakened. |
| 4 | TTY-04: daemon picker enters alt-screen with seeded record content, restores main buffer on quit, zero residual escape bytes; watched FAIL against the historical G-07-2 defect | ✓ VERIFIED (regression) | `daemon_picker_test.go`'s only change from 08-05 is anchoring its two `pollUntilStable` calls on `paneContains(t,"Running daemons")` and `altScreenOff(t,session)` — both match the content/state the test's own next assertions already require. Re-ran via `task test:tmux`: PASS. `internal/cli/tui/daemonpicker.go` and `internal/cli/daemon.go` confirmed byte-identical to pre-phase state (`git diff --quiet 5baba883..HEAD` exits 0), so the mutation-log family (b) RED/GREEN evidence from the prior pass still applies unchanged. |
| 5 | TTY-05: checkbox picker renders `[ ]`/`[x]` glyphs, `space` toggles, `q`/`esc` both cancel with zero config writes (whole-tree hash before/after); watched FAIL against a real mutation | ✓ VERIFIED (regression) | `install_cancel_test.go`'s four `pollUntilStable` calls now anchor on `"[ ]"`, `"[x]"`, and `altScreenOff` per cancel path. Re-ran via `task test:tmux`: PASS. `confighash.go` and `internal/cli/tui/agentpicker.go` unchanged by 08-05 (byte-clean diff confirmed); mutation-log family (c) evidence still applies. |
| 6 | TTY-06: idle checkbox picker holds byte-identical across 5 further captures after its first settled frame; N is reported and echoed into the CI job log | ✓ VERIFIED (regression) | `frame_stability_test.go`'s post-settle loop (lines 52-59, the raw fixed-interval sampling that measures idle stability) is byte-unchanged — confirmed by direct read; only the initial `pollUntilStable` call gained `paneContains(t,"[ ]")`. Re-ran via `task test:tmux`: `TTY-06: frame-stable across N=5 captures (pane 100x12, install picker idle)` logged and echoed by the Task target's `jq` filter, matching the prior pass. |
| 7 | TTY-07 design/wiring: `tmux-e2e` CI job installs tmux, asserts `tmux -V` against a committed constant, runs the suite, gates on an exact executed-count equality now at 6 | ✓ VERIFIED | `.github/workflows/ci.yml` untouched by 08-05 (still bare `task test:tmux:install` / `task test:tmux` `run:` lines, `CI: "1"` via step-level `env:`). `Taskfile.yml:213`: `TMUX_EXPECTED_TESTS: 6`, in the same commit (`e994b0a5`, `test(08-05): add failing poll self-test`) that adds the sixth top-level `Test*` function — confirmed via `git show e994b0a5 --stat`. `internal/upgrade/taskfile_shape_test.go`'s `inScopeJobs` entry for `{ci.yml, tmux-e2e}` still present; re-ran `TestWorkflowRunBodiesInvokeTask`, `TestInScopeJobsPopulationMatchesDisk`: PASS. |
| 8 | TTY-07 harness-level backstop (08-05's own `verification: backstop` must-have): "the harness's wait primitive can distinguish a settled post-output frame from a settled pre-output frame, rather than accepting whichever settles first" | ✓ VERIFIED | This is exactly what `TestPollUntilStableDoesNotConvergeOnPreOutputFrame` proves and what was directly, independently re-executed in truth #2 above (GREEN, converged past the commanded delay with readiness first holding mid-poll) — a passing, wired, property-based self-test is the explicit evidence this non-inferable truth requires; presence-and-wiring alone would not have sufficed, so it was actually run. |
| 9 | TTY-07 CI-level backstop (carried from 08-01/08-03): the exact-count assertion has actually fired on a real CI run for the code as it exists at HEAD, not merely built correctly | ? UNCERTAIN → human_verification | Re-derived live: `gh run view 34607117422 --log` shows `test:tmux: executed=5 skipped=0 expected=5`, conclusion `success`, at commit `3899e6de` — this proves the *mechanism* works (install-then-assert-exact-count correctly gated a real run) but that commit predates 08-05's `TMUX_EXPECTED_TESTS: 5 -> 6` bump. `git rev-list --left-right --count HEAD...origin/...` = `8 0` (HEAD is 8 commits ahead of, 0 behind, the remote); `gh pr view 69 --json headRefOid` = `3899e6de`, confirming PR #69's tracked head has not advanced past the pre-08-05 commit. No CI run exists anywhere on this branch for a commit carrying `expected=6`. The local reproduction (`task test:tmux` -> `executed=6 skipped=0 expected=6`, confirmed this pass) is real evidence the suite itself is correct, but does not substitute for the CI-environment backstop this truth specifically asserts (this repo's own convention, per the prior pass's Judgment 3, treats `verification: backstop` truths as non-inferable from the working tree). Routed to Human Verification below. |
| 10 | Byte-clean gap-closure scope: no production file touched by 08-05; `test/tmux/main_test.go` (D-09's resolver contract) untouched | ✓ VERIFIED | `git diff --quiet 5baba883..HEAD -- internal/cli/daemon.go internal/cli/tui/daemonpicker.go internal/cli/tui/agentpicker.go test/tmux/main_test.go` exits 0 at HEAD — independently re-run this pass, not transcribed from the SUMMARY. `git status --porcelain` shows only `.planning/state.json` modified (bookkeeping, not code). |
| 11 | TTY-05 concurrency edge: unique session names per OS process/test function; teardown tolerates an already-gone session | ✓ VERIFIED (regression) | `session.go` unchanged by 08-05 (confirmed byte-clean above); `rg -n "t\.Parallel" test/tmux/*.go` still returns no matches. |
| 12 | TTY-07 boundary edge: EXPECTED-1 and EXPECTED+1 both fail the CI gate (equality, not a floor) | ✓ VERIFIED (regression) | `Taskfile.yml:256`'s `[ "${EXECUTED}" -ne {{.TMUX_EXPECTED_TESTS}} ]` logic is unchanged by 08-05 (only the constant's value moved 5->6); the symmetric-inequality shape itself was not re-touched. |
| 13 | TTY-04's seeded record comes from a real `codegraph daemon start` subprocess, never a hand-written fixture | ✓ VERIFIED (regression) | `daemon_seed.go` unchanged by 08-05 (byte-clean diff includes this file's directory scope; confirmed no diff via `git log --follow -- test/tmux/daemon_seed.go` showing no commit since `5baba883`). |
| 14 | TTY-03's mutation-log family (a) RED shape is preserved after the anchor change: the doubly-mutated binary still prints the anchor before the leak arrives, so the anchored poll converges past it and the marker assertion still fires as logged | ✓ VERIFIED | Read `daemon_empty_test.go` in full: the poll now anchors on `"no running daemons"` (the same line the positive-half assertion checks), and the negative-half loop over `decrqmResponseMarkers` is untouched. This is architecturally identical to family (a)'s original recipe — the anchor is the content the mutation-log's RED demonstration already required to be present before the leak. Not re-run against a live re-mutation this pass (that would require re-applying and reverting the family (a) mutation, which the prior pass already did and 08-05 did not touch); accepted on direct code read, which requires no behavioral inference beyond what test #3's fresh re-execution already confirms (the anchored assertion fires correctly end-to-end). |
| 15 | Pre/post-mutation cleanliness gates were actually checked, not merely asserted in prose (carried forward) | ✓ VERIFIED | `08-MUTATION-LOG.md` is unchanged by 08-05 (not in `files_modified`); this pass independently re-confirmed the phase-wide byte-clean state (truth #10) rather than trusting the log's own final claim alone. |

**Score:** 14/15 truths verified, 1 routed to human_verification (TTY-07's CI-level backstop — no CI run exists yet for a commit carrying `TMUX_EXPECTED_TESTS=6`).

### Gap Closure Assessment (G-08-1)

08-05's stated closure mechanism — give `pollUntilStable` a required readiness predicate so
convergence is `ready(capture) && capture == predecessor` rather than bare byte-equality — was
independently verified structurally (code read: nil-predicate and empty-needle both
`t.Fatal`; all 10 pre-existing call sites plus the new self-test's call all pass a non-nil
anchor, confirmed via `rg -n "pollUntilStable\("`) and behaviorally (the deterministic self-test
`TestPollUntilStableDoesNotConvergeOnPreOutputFrame`, independently re-executed this pass:
PASS, converged past the commanded 4s delay with `K=5`). Three fresh cold-arm runs of the
original flaking test (`TestDaemonEmptyRegistryLeaksNoModeQueryBytes`) all passed, each its own
process with a freshly built binary — consistent with, not contradicting, the SUMMARY's honest
disclosure that the machine was warm during evidence-gathering (K<=1 on all three) and that the
deterministic self-test — not these three runs — carries the actual proof of the cold-path fix.
No fabrication or overclaiming was found in the SUMMARY's own framing of this evidence; it
reads exactly as cautiously as the underlying transcripts warrant.

**G-08-1 is closed.** The prior pass's only other open item (TTY-07's `verification: backstop`
truth) remains open for a different, narrower reason than before: the *mechanism* is now proven
in CI (`executed=5 skipped=0 expected=5`, PR #69, run 34607117422, commit 3899e6de), but the
*current* expected count (6, introduced by 08-05 itself) has not yet been asserted for real in
CI because the 8 commits since `3899e6de` — including 08-05 and the subsequent code review —
have not been pushed to the remote branch that PR #69 tracks.

## Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `test/tmux/capture.go` | Readiness-anchored `pollUntilStable`, `paneContains`, `altScreenOff`, truthful header comment | ✓ VERIFIED | Read in full; matches 08-05's must_haves exactly, including the amended header comment recording cold (1.1-1.6s in-pane, 739-1230ms shell-timed) vs warm (167-226ms) measurements and removing the false "exceeds the measured worst case by construction" claim. |
| `test/tmux/poll_contract_test.go` | New deterministic self-test of the wait primitive | ✓ VERIFIED | Read in full; created fresh by 08-05; re-executed, PASS. |
| `test/tmux/daemon_empty_test.go` | TTY-03 assertion, poll anchored on `"no running daemons"` | ✓ VERIFIED | Read in full; re-executed 3x fresh, all PASS. |
| `test/tmux/daemon_picker_test.go` | TTY-04 assertion, polls anchored | ✓ VERIFIED | Re-executed via full suite, PASS. |
| `test/tmux/install_cancel_test.go` | TTY-05 assertion, polls anchored | ✓ VERIFIED | Re-executed via full suite, PASS. |
| `test/tmux/frame_stability_test.go` | TTY-06 assertion, initial settle anchored, post-settle loop untouched | ✓ VERIFIED | Read in full; re-executed, PASS, N=5 reported. |
| `Taskfile.yml` (`TMUX_EXPECTED_TESTS: 6`) | Bumped same-commit as sixth test | ✓ VERIFIED | `git show e994b0a5 --stat` confirms `Taskfile.yml` and the new test file in the same commit. |
| `.planning/phases/08-tmux-real-pty-harness/08-01-PLAN.md` | Poll truth amended to readiness-anchored contract | ✓ VERIFIED | `rg -n "readiness" 08-01-PLAN.md` shows the amended `must_haves.truths` entry and key_link text referencing "D-14 as amended by 08-05, closing G-08-1." |
| `internal/cli/daemon.go`, `internal/cli/tui/daemonpicker.go`, `internal/cli/tui/agentpicker.go`, `test/tmux/main_test.go` | Byte-identical to pre-gap-closure state | ✓ VERIFIED | `git diff --quiet 5baba883..HEAD` exits 0, independently re-run. |
| `.planning/phases/08-tmux-real-pty-harness/08-REVIEW.md` | Incremental code review of the 08-05 diff | ✓ VERIFIED (as an honest record) | `issues_found`, 0 critical / 1 warning (WR-06) / 1 info (IN-05); read in full, findings independently assessed below under Anti-Patterns. |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| Every `test/tmux/*_test.go` poll call site | `pollUntilStable`'s required `ready` parameter | Go compiler | ✓ WIRED | `go vet -tags tmux ./test/tmux/...` passes clean; a call site with no anchor cannot compile — the structural gate 08-05's own key_links describe. |
| `Taskfile.yml: TMUX_EXPECTED_TESTS` | `test:tmux`'s exact-count gate | shell `-ne` comparison | ✓ WIRED | Re-ran `task test:tmux`: `executed=6 skipped=0 expected=6`, exit 0. |
| `.github/workflows/ci.yml: tmux-e2e` | `task test:tmux:install` / `task test:tmux` | bare `run:` lines | ✓ WIRED (design) — see truth #9 for the live-run-at-HEAD gap | Unchanged by 08-05; `TestWorkflowRunBodiesInvokeTask` re-run, PASS. |
| `internal/upgrade/taskfile_shape_test.go: inScopeJobs` | `.github/workflows/ci.yml` jobs on disk | `TestInScopeJobsPopulationMatchesDisk` | ✓ WIRED | Re-run, PASS. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Full harness suite at HEAD | `GOTOOLCHAIN=go1.26.6 go vet -tags tmux ./test/tmux/...` | exit 0, no output | ✓ PASS |
| Self-test proving G-08-1's fix (GREEN half) | `GOTOOLCHAIN=go1.26.6 go test -tags tmux -count=1 -run 'TestPollUntilStableDoesNotConvergeOnPreOutputFrame' -v ./test/tmux/...` | `converged at sample 6 after 6.04s (ready first held at sample 5)`, PASS | ✓ PASS |
| TTY-03 cold-arm reproduction (run 1/3) | `go test -tags tmux -count=1 -run 'TestDaemonEmptyRegistryLeaksNoModeQueryBytes' -v ./test/tmux/...` | `converged at sample 2 after 2.017s (ready first held at sample 1)`, PASS | ✓ PASS |
| TTY-03 cold-arm reproduction (run 2/3) | same | `converged at sample 2 after 2.018s (ready first held at sample 1)`, PASS | ✓ PASS |
| TTY-03 cold-arm reproduction (run 3/3) | same | `converged at sample 2 after 2.014s (ready first held at sample 1)`, PASS | ✓ PASS |
| Full suite via Task target | `GOTOOLCHAIN=go1.26.6 task test:tmux` | `test:tmux: executed=6 skipped=0 expected=6`, exit 0, 6/6 named tests PASS | ✓ PASS |
| Byte-clean gap-closure-scope proof | `git diff --quiet 5baba883..HEAD -- internal/cli/daemon.go internal/cli/tui/daemonpicker.go internal/cli/tui/agentpicker.go test/tmux/main_test.go` | exit 0 | ✓ PASS |
| Regression: repo-wide build | `GOTOOLCHAIN=go1.26.6 go build ./...` | exit 0 | ✓ PASS |
| Regression: CI-shape guard tests | `go test -count=1 ./internal/upgrade/...` | `ok` | ✓ PASS |
| Debt-marker scan on phase-touched files | `rg -n "TBD\|FIXME\|XXX" test/tmux/ Taskfile.yml` | no matches | ✓ PASS (no debt markers) |
| CI evidence for the exact-count mechanism (pre-08-05 commit) | `gh run view 34607117422 --log` | `test:tmux: executed=5 skipped=0 expected=5`, conclusion success | ✓ PASS (mechanism proven; see truth #9 for why this does not close the current-HEAD backstop) |

### Probe Execution

No `scripts/*/tests/probe-*.sh` files exist for this phase and neither the PLANs nor SUMMARYs
reference any. Step 7c: SKIPPED (no probes declared or discovered).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| TTY-01 | 08-01 | Build-tagged harness, real pane, skip-with-reason when tmux absent | ✓ SATISFIED | Truth #1 |
| TTY-02 | 08-01, amended by 08-05 | Poll-to-convergence before asserting, no fixed sleep; now proven against a genuinely racing case | ✓ SATISFIED | Truths #2, #8 |
| TTY-03 | 08-01, amended by 08-05 | Empty-registry DECRQM leak-free, watched fail; G-08-1 cold-start race closed | ✓ SATISFIED | Truths #3, #14 |
| TTY-04 | 08-02 | Alt-screen entry/exit, watched fail | ✓ SATISFIED | Truth #4 |
| TTY-05 | 08-02 | Checkbox glyphs/toggle/cancel zero-write, watched fail | ✓ SATISFIED | Truth #5 |
| TTY-06 | 08-02 | Idle frame stability, N reported | ✓ SATISFIED | Truth #6 |
| TTY-07 | 08-03, TMUX_EXPECTED_TESTS bumped by 08-05 | CI job design + exact-count gate | ✓ SATISFIED (design/wiring; mechanism proven at expected=5) — live-run confirmation at expected=6 outstanding | Truths #7, #9 |

No orphaned requirements found: REQUIREMENTS.md maps exactly TTY-01…TTY-07 to Phase 8, and
08-05's `requirements:` frontmatter (`[TTY-02, TTY-03]`) is consistent with the gap it closes.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `test/tmux/capture.go` | 190-201 (`paneContains`); doc comment 127-133 | WR-06 (new, from 08-REVIEW.md, independently read and concurred with): the anchor-required mechanism only mechanically rejects an *empty* needle, not a needle that also appears in the pre-output echoed command line — that vacuity class is enforced by doc comment and manual review only, for any *future* call site | ⚠️ Warning | Does not affect any of the 11 current call sites (all verified anchor-clean by the reviewer and independently spot-checked here); a real risk for future call sites added without following the documented rule. Non-blocking; the reviewer's suggested fix (a static always-running check analogous to `taskfile_shape_test.go`) is a reasonable follow-up, not required for this phase's goal. |
| `test/tmux/poll_contract_test.go` | 52-54 | IN-05 (from 08-REVIEW.md): `start := time.Now()` is captured before `sendKey(t, session, "Enter")`, so the measured `elapsed` slightly over-counts genuine wait time | ℹ️ Info | Conservative direction only (cannot cause a false pass); cosmetic. |
| `test/tmux/capture.go` | 113 (carried forward, unchanged by 08-05) | IN-01 (already filed, prior pass): comment "no blocking pause call" is technically imprecise | ℹ️ Info | Cosmetic; unaffected by this gap-closure plan. |
| `test/tmux/capture.go` / `frame_stability_test.go` (carried forward, unchanged by 08-05) | — | WR-01/WR-02 (already filed, prior pass): TTY-06's idle-stability loop necessarily samples on a fixed interval outside `pollUntilStable`, contradicting the package doc comment's absolute "ONLY wait primitive" wording | ⚠️ Warning | Same status as the prior pass — architecturally necessary, doc-wording issue only. Not re-litigated further; 08-05 did not touch this loop. |
| `Taskfile.yml` (carried forward) | `test:tmux` cmds block | WR-03 (already filed) | ⚠️ Warning | Unchanged by 08-05; not re-litigated. |
| `test/tmux/daemon_seed.go` (carried forward, unchanged by 08-05) | teardown | WR-04 (already filed) | ⚠️ Warning | Unchanged by 08-05; not re-litigated. |
| `test/tmux/confighash.go` (carried forward, unchanged by 08-05) | 63-64 | WR-05 (already filed) | ⚠️ Warning | Unchanged by 08-05; not re-litigated. |

No 🛑 Blocker-severity findings and no unresolved debt markers (`TBD`/`FIXME`/`XXX`) in any
phase-touched file, including the 08-05 diff.

## Human Verification Required

### 1. Confirm the `tmux-e2e` CI job's exact-count gate has actually fired against 6, not merely 5

**Test:** Push this branch's 8 unpushed commits (or otherwise sync PR #69's head past `3899e6de`) so `.github/workflows/ci.yml`'s `tmux-e2e` job runs on `ubuntu-latest` at a commit that carries `TMUX_EXPECTED_TESTS: 6`, then open that run's job log for the "tmux real-pty harness (TTY-01..TTY-07)" step.
**Expected:** The log contains the line `test:tmux: executed=6 skipped=0 expected=6`, the job's overall conclusion is `success`, and the `tmux -V` line matches the already-committed `TMUX_EXPECTED_VERSION: tmux 3.4`.
**Why human:** This is 08-03-PLAN.md's `verification: backstop` truth (a real CI run proving the count-assertion actually fired) — non-inferable from the working tree by construction. The only CI evidence that exists proves the *mechanism* at `expected=5` (PR #69, run 34607117422, commit `3899e6de`, `success`); it predates 08-05's `TMUX_EXPECTED_TESTS: 5 -> 6` bump and the new sixth test. `git rev-list --left-right --count HEAD...origin/gsd/v0.13.0-guard-hardening-ui-follow-through` = `8 0` and `gh pr view 69 --json headRefOid` = `3899e6de`, both re-derived live during this verification — HEAD (`400b4c1d`) has not been pushed. The local reproduction of `executed=6 skipped=0 expected=6` (this pass, and 08-05-SUMMARY.md's own paste) is real but is not a substitute for the CI-environment run this specific backstop truth asserts.

## Gaps Summary

No `gaps_found`-tier defects were identified: G-08-1 is closed — the readiness-predicate fix
was independently verified both structurally (compiler-enforced anchor requirement, nil/empty
guards read directly) and behaviorally (the deterministic self-test re-executed GREEN this
pass, three fresh cold-arm TTY-03 runs all passed). Every artifact 08-05 claims to have created
or modified exists, is substantive, and is wired; the byte-clean gap-closure-scope proof
(no production file touched, `TestMain` untouched) was independently reproduced, not merely
trusted from the SUMMARY. The regression suite (build, `go vet`, `internal/upgrade` CI-shape
guards, the full six-test `task test:tmux` run) all pass at HEAD.

The phase's status is `human_needed` rather than `passed` for one reason, narrower than the
prior pass's: TTY-07's CI-level backstop truth — a real CI run asserting the exact-count gate —
has been proven for the *mechanism* (`expected=5`, PR #69, run 34607117422, success) but not yet
for the *current* code (`expected=6`, introduced by 08-05), because HEAD is 8 commits ahead of
the branch PR #69 currently tracks and no CI run has executed at any of those 8 commits. This is
a live external fact, re-derived at verification time via `gh run list` and `gh pr view`, not a
code or wiring defect — pushing the branch and reading the resulting job log is the only way to
close it. One advisory finding is carried in this pass: WR-06 (08-REVIEW.md, independently
concurred with) notes the new readiness-anchor mechanism enforces the empty-needle vacuity case
mechanically but relies on doc-comment discipline alone for the "needle present in the
pre-output frame" vacuity case at any *future* call site — non-blocking for this phase (all 11
current call sites are clean), worth a maintainer decision on whether to land the reviewer's
suggested static check as a follow-up.
