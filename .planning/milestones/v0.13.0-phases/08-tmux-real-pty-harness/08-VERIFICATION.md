---
phase: 08-tmux-real-pty-harness
verified: 2026-09-13T22:15:00Z
status: passed
score: 15/15 must-haves verified
behavior_unverified: 0
overrides_applied: 0
covered_files: [".github/workflows/ci.yml",".planning/phases/08-tmux-real-pty-harness/08-01-PLAN.md",".planning/phases/08-tmux-real-pty-harness/08-01-SUMMARY.md",".planning/phases/08-tmux-real-pty-harness/08-02-PLAN.md",".planning/phases/08-tmux-real-pty-harness/08-02-SUMMARY.md",".planning/phases/08-tmux-real-pty-harness/08-03-PLAN.md",".planning/phases/08-tmux-real-pty-harness/08-03-SUMMARY.md",".planning/phases/08-tmux-real-pty-harness/08-04-PLAN.md",".planning/phases/08-tmux-real-pty-harness/08-04-SUMMARY.md",".planning/phases/08-tmux-real-pty-harness/08-05-PLAN.md",".planning/phases/08-tmux-real-pty-harness/08-05-SUMMARY.md",".planning/phases/08-tmux-real-pty-harness/08-CONTEXT.md",".planning/phases/08-tmux-real-pty-harness/08-DISCUSSION-LOG.md",".planning/phases/08-tmux-real-pty-harness/08-PATTERNS.md",".planning/phases/08-tmux-real-pty-harness/08-RESEARCH.md","Taskfile.yml","internal/upgrade/taskfile_shape_test.go","test/tmux/capture.go","test/tmux/confighash.go","test/tmux/daemon_empty_test.go","test/tmux/daemon_picker_test.go","test/tmux/daemon_seed.go","test/tmux/frame_stability_test.go","test/tmux/install_cancel_test.go","test/tmux/main_test.go","test/tmux/poll_contract_test.go","test/tmux/session.go","test/tmux/skip_contract_test.go"]
covered_digest: "v1:sha256:e33aec908208b23a63f94883502cd234ef9ddf1133b5e29edaec1f3016cc1a68"
re_verification:
  previous_status: human_needed
  previous_verified: "2026-09-11T22:30:00Z"
  previous_score: 14/15
  reason: "milestone-close re-pin — canonical status read stale (#4155: prior covered_files listed .planning/REQUIREMENTS.md, which every later phase.complete rewrites, plus 08-REVIEW.md/08-MUTATION-LOG.md/08-SECURITY.md, which the re-verification convention excludes); all 15 must-haves re-executed at HEAD 8c8149de. The prior pass's single open item (TTY-07's live CI backstop at expected=6) was independently closed in the interim by 08-UAT.md (test 21, coverage_id 08-VERIFICATION-human_verification-1: CI run 34658987243, job 103457354620, commit 5ffdc2a0, success) — re-confirmed live in this pass, not merely read from the UAT file."
  gaps_closed:
    - "G-08-1: TestDaemonEmptyRegistryLeaksNoModeQueryBytes (TTY-03) no longer depends on the spawned binary's first exec landing inside a single stabilityPollInterval — pollUntilStable now requires a caller-stated readiness predicate (ready(capture) && capture == predecessor), structurally closing the cold-start race regardless of startup latency short of stabilityPollDeadline. (carried from the 2026-09-11 pass, re-confirmed unregressed at HEAD.)"
    - "TTY-07's live CI backstop (the prior pass's sole open item): a real CI run has now fired against a commit carrying TMUX_EXPECTED_TESTS=6, both mechanism and current code are proven together."
  gaps_remaining: []
  regressions: []
---

# Phase 8: tmux Real-PTY Harness Verification Report

**Phase Goal:** The release binary's interactive TUI is exercised inside a real tmux pane that answers escape queries and actually scrolls — the missing rung between the piped, TTY-blind integration suite and manual human UAT.
**Verified:** 2026-09-13T22:15:00Z
**Status:** passed
**Re-verification:** Yes — milestone-close re-verification (previous: human_needed, 14/15, 2026-09-11T22:30:00Z)

## Goal Achievement

This is a milestone-close re-verification at HEAD (`8c8149de`), triggered because the canonical
`verification.status` query read `stale` for the prior report (it pinned `.planning/REQUIREMENTS.md`,
which every subsequent `phase.complete` rewrites, and included `08-REVIEW.md`/`08-MUTATION-LOG.md`/
`08-SECURITY.md`, excluded from `covered_files` by the current re-verification convention — see
Phase 8's own prior lesson, commit `9849eaf2`). This is not a diff against the 2026-09-11 pass:
every truth was independently re-checked against the current tree, and the one item that pass left
`human_needed` (TTY-07's live CI backstop at `expected=6`) is re-confirmed here with fresh, direct
evidence rather than accepted on the strength of `08-UAT.md`'s narrative alone.

The prior `08-VERIFICATION.md` (2026-09-11, `400b4c1d`-era, score 14/15, `human_needed` solely on
TTY-07's CI backstop) is superseded by this report per the workflow's overwrite instruction.
Nothing between `5ffdc2a0` (the commit `08-UAT.md`'s closing evidence and the merged PR #69 head
both pin) and current HEAD (`8c8149de`, 162 commits later — phases 9 through 12) touched
`test/tmux/**`, the `tmux-e2e` job body, or the `test:tmux` Task target's logic: `git diff --stat
5ffdc2a0..HEAD -- test/tmux Taskfile.yml .github/workflows/ci.yml
internal/upgrade/taskfile_shape_test.go` shows only `Taskfile.yml` (+227, new `docs:cli`,
`docs:cli:drift`, `check:gonum`, `check:no-force-layout` targets — Phase 11/12 additions, none of
which are `test:tmux`) and `.github/workflows/ci.yml` (+7, a `CLI reference drift guard (DOCS-05)`
step inserted into an unrelated job) changed — confirmed by reading both diffs in full, not by
line-count alone. `git diff --quiet 5ffdc2a0..HEAD -- internal/cli/daemon.go
internal/cli/tui/daemonpicker.go internal/cli/tui/agentpicker.go test/tmux/` exits 0.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | TTY-01: harness builds the binary, spawns it in a real tmux pane, sends keys, captures, exits 0; skip contract fires with a reason when tmux is absent | ✓ VERIFIED (regression) | `test/tmux/main_test.go`, `session.go`, `skip_contract_test.go` byte-unchanged since `5ffdc2a0` (confirmed above). Re-executed the full suite at HEAD this pass (`GOTOOLCHAIN=go1.26.6 task test:tmux`): `executed=6 skipped=0 expected=6`, all six named tests PASS including `TestRequireTmuxReportsSkipReasonWhenAbsent`. |
| 2 | TTY-02 (amended by 08-05): `pollUntilStable`'s convergence is `ready(capture) && capture == predecessor`; the readiness predicate is a required parameter (nil -> `t.Fatal`), an empty `paneContains` needle is an immediate `t.Fatal`; proven against a case that races and fails under a single naive capture | ✓ VERIFIED (regression) | `capture.go` byte-unchanged since `5ffdc2a0`. Re-executed `TestPollUntilStableDoesNotConvergeOnPreOutputFrame` directly at HEAD: PASS, `converged at sample 6 after 6.054203834s (ready first held at sample 5)` — past the commanded 4s delay. `go vet -tags tmux ./test/tmux/...` passes clean. |
| 3 | TTY-03 (G-08-1 closed): bare `codegraph daemon` on empty registry leaks zero DECRQM bytes, and now passes on the FIRST exec of a freshly built binary regardless of cold-start latency short of the 10s deadline | ✓ VERIFIED (regression) | Re-ran `TestDaemonEmptyRegistryLeaksNoModeQueryBytes` at HEAD via the full suite run: PASS, `converged at sample 2 after 2.022179375s (ready first held at sample 1)`. `daemon_empty_test.go` byte-unchanged since `5ffdc2a0`; the deterministic self-test (truth #2) carries the structural proof for the genuine cold path, independently reproduced GREEN this pass. |
| 4 | TTY-04: daemon picker enters alt-screen with seeded record content, restores main buffer on quit, zero residual escape bytes; watched FAIL against the historical G-07-2 defect | ✓ VERIFIED (regression) | `daemon_picker_test.go` byte-unchanged since `5ffdc2a0`. Re-ran via `task test:tmux` at HEAD: PASS (`TestDaemonPickerEntersAltScreenAndRestoresMainBuffer`, converged at samples 2 and 1). `internal/cli/tui/daemonpicker.go` and `internal/cli/daemon.go` confirmed byte-identical since `5ffdc2a0`. |
| 5 | TTY-05: checkbox picker renders `[ ]`/`[x]` glyphs, `space` toggles, `q`/`esc` both cancel with zero config writes (whole-tree hash before/after); watched FAIL against a real mutation | ✓ VERIFIED (regression) | `install_cancel_test.go` byte-unchanged since `5ffdc2a0`. Re-ran via `task test:tmux` at HEAD: PASS (`TestInstallPickerCancelWritesNoConfig`, five poll convergences logged). `confighash.go` and `internal/cli/tui/agentpicker.go` confirmed byte-identical since `5ffdc2a0`. |
| 6 | TTY-06: idle checkbox picker holds byte-identical across 5 further captures after its first settled frame; N is reported and echoed into the CI job log | ✓ VERIFIED (regression) | `frame_stability_test.go` byte-unchanged since `5ffdc2a0`. Re-ran at HEAD: `TTY-06: frame-stable across N=5 captures (pane 100x12, install picker idle)` logged and echoed by the Task target's `jq` filter. |
| 7 | TTY-07 design/wiring: `tmux-e2e` CI job installs tmux, asserts `tmux -V` against a committed constant, runs the suite, gates on an exact executed-count equality now at 6 | ✓ VERIFIED (regression) | `.github/workflows/ci.yml`'s `tmux-e2e` job block is untouched since `5ffdc2a0` (the only diff in the whole file since then is an unrelated `docs:cli:drift` step added to a different job, confirmed by reading the full diff). `Taskfile.yml: TMUX_EXPECTED_TESTS: 6` unchanged. `internal/upgrade/taskfile_shape_test.go`'s `inScopeJobs` entry for `{ci.yml, tmux-e2e}` still present; re-ran `go test -count=1 ./internal/upgrade/...` at HEAD: `ok`. |
| 8 | TTY-07 harness-level backstop (08-05's own `verification: backstop` must-have): "the harness's wait primitive can distinguish a settled post-output frame from a settled pre-output frame, rather than accepting whichever settles first" | ✓ VERIFIED | Exactly what `TestPollUntilStableDoesNotConvergeOnPreOutputFrame` proves and what was directly, independently re-executed in truth #2 above (GREEN at HEAD, converged past the commanded delay with readiness first holding mid-poll). |
| 9 | TTY-07 CI-level backstop (carried from 08-01/08-03): the exact-count assertion has actually fired on a real CI run for the code as it exists at HEAD, not merely built correctly | ✓ VERIFIED | **Now closed** (the prior pass's sole open item). Independently re-derived live, not read from `08-UAT.md`: `gh pr view 69 --json headRefOid,state,mergedAt` shows PR #69 MERGED with `headRefOid: 5ffdc2a0` (2026-09-12T13:11:12Z). `gh run view 34658987243 --json conclusion,headSha,status`: `conclusion: success`, `headSha: 5ffdc2a0`. `gh run view --job 103457354620 --log` (job name `tmux e2e (real-pty harness, TTY-01..TTY-07)`, `conclusion: success`, independently resolved via `gh run view 34658987243 --json jobs`): log contains `test:tmux: executed=6 skipped=0 expected=6` and six `--- PASS:` lines (`TestDaemonEmptyRegistryLeaksNoModeQueryBytes` 2.09s, `TestDaemonPickerEntersAltScreenAndRestoresMainBuffer` 4.09s, `TestInstallPickerFrameStableWhileIdle` 7.06s, `TestInstallPickerCancelWritesNoConfig` 11.12s, `TestPollUntilStableDoesNotConvergeOnPreOutputFrame` 6.05s, `TestRequireTmuxReportsSkipReasonWhenAbsent` 0.00s). The three `::error::` hits in the raw log are the Taskfile's own `echo "::error::..."` source lines (grep matches the script body, not a fired GitHub Actions annotation) — confirmed the job conclusion is `success` and the printed `executed=6` line is the real gate output, not a masked failure. `git diff --stat 5ffdc2a0..HEAD -- test/tmux Taskfile.yml .github/workflows/ci.yml` shows neither the harness nor the `tmux-e2e` job/`test:tmux` target changed since that CI run, so this run's evidence still applies unmodified to current HEAD. This closes the truth the prior pass left `human_needed` on. |
| 10 | Byte-clean gap-closure scope: no production file touched by 08-05; `test/tmux/main_test.go` (D-09's resolver contract) untouched | ✓ VERIFIED (regression) | `git diff --quiet 5ffdc2a0..HEAD -- internal/cli/daemon.go internal/cli/tui/daemonpicker.go internal/cli/tui/agentpicker.go test/tmux/` exits 0 at HEAD — independently re-run this pass across the full `test/tmux` directory, not just the four files the prior pass checked. |
| 11 | TTY-05 concurrency edge: unique session names per OS process/test function; teardown tolerates an already-gone session | ✓ VERIFIED (regression) | `session.go` byte-unchanged since `5ffdc2a0` (part of the directory-wide byte-clean proof above). |
| 12 | TTY-07 boundary edge: EXPECTED-1 and EXPECTED+1 both fail the CI gate (equality, not a floor) | ✓ VERIFIED (regression) | `Taskfile.yml`'s `[ "${EXECUTED}" -ne {{.TMUX_EXPECTED_TESTS}} ]` shell logic is unchanged since `5ffdc2a0` — the only `Taskfile.yml` diff in this window is new, unrelated target bodies appended elsewhere in the file (confirmed by reading the full diff). |
| 13 | TTY-04's seeded record comes from a real `codegraph daemon start` subprocess, never a hand-written fixture | ✓ VERIFIED (regression) | `daemon_seed.go` byte-unchanged since `5ffdc2a0` (part of the directory-wide byte-clean proof above). |
| 14 | TTY-03's mutation-log family (a) RED shape is preserved after the anchor change: the doubly-mutated binary still prints the anchor before the leak arrives, so the anchored poll converges past it and the marker assertion still fires as logged | ✓ VERIFIED (regression) | `daemon_empty_test.go` byte-unchanged since `5ffdc2a0` (part of the directory-wide byte-clean proof above); truth #3's fresh re-execution this pass confirms the anchored assertion still fires end-to-end. |
| 15 | Pre/post-mutation cleanliness gates were actually checked, not merely asserted in prose (carried forward) | ✓ VERIFIED | `08-MUTATION-LOG.md` (excluded from this pass's `covered_files` per the re-verification convention, but its subject files remain byte-clean) is unchanged since `5ffdc2a0`; this pass independently re-confirmed the phase-wide byte-clean state at HEAD (truth #10) rather than trusting the log's own final claim alone. |

**Score:** 15/15 truths verified.

### Gap Closure Assessment (carried forward, unregressed)

08-05's readiness-predicate fix for G-08-1 remains structurally and behaviorally verified at
HEAD: `capture.go` is byte-identical to the state the 2026-09-11 pass examined, and this pass
independently re-executed both the deterministic self-test
(`TestPollUntilStableDoesNotConvergeOnPreOutputFrame`, GREEN, converged past the commanded 4s
delay) and `TestDaemonEmptyRegistryLeaksNoModeQueryBytes` (PASS at HEAD). No regression found.

### Live-CI Closure of the Prior Pass's Sole Open Item (TTY-07)

The 2026-09-11 pass left status `human_needed` for exactly one reason: no CI run existed yet for
a commit carrying `TMUX_EXPECTED_TESTS: 6` (HEAD at that time, `400b4c1d`, was 8 commits ahead of
the branch PR #69 tracked). That gap was closed in the interim by pushing and merging PR #69
(`08-UAT.md` test 21, `coverage_id: 08-VERIFICATION-human_verification-1`, evidence: CI run
`34658987243` at commit `5ffdc2a0`). This pass does not take that claim on faith: it independently
re-derived the same facts live via `gh pr view`, `gh run view --json`, and `gh run view --job
... --log`, confirmed the job succeeded and the exact-count line printed a real 6, and confirmed
via `git diff` that nothing relevant has changed between that CI-evidenced commit and current
HEAD. This is what moves the score from the prior pass's 14/15 to 15/15 and the status from
`human_needed` to `passed`.

## Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `test/tmux/capture.go` | Readiness-anchored `pollUntilStable`, `paneContains`, `altScreenOff`, truthful header comment | ✓ VERIFIED | Byte-unchanged since `5ffdc2a0`; content previously read in full and independently reconfirmed unchanged this pass. |
| `test/tmux/poll_contract_test.go` | Deterministic self-test of the wait primitive | ✓ VERIFIED | Byte-unchanged since `5ffdc2a0`; re-executed, PASS. |
| `test/tmux/daemon_empty_test.go` | TTY-03 assertion, poll anchored on `"no running daemons"` | ✓ VERIFIED | Byte-unchanged since `5ffdc2a0`; re-executed, PASS. |
| `test/tmux/daemon_picker_test.go` | TTY-04 assertion, polls anchored | ✓ VERIFIED | Byte-unchanged since `5ffdc2a0`; re-executed via full suite, PASS. |
| `test/tmux/install_cancel_test.go` | TTY-05 assertion, polls anchored | ✓ VERIFIED | Byte-unchanged since `5ffdc2a0`; re-executed via full suite, PASS. |
| `test/tmux/frame_stability_test.go` | TTY-06 assertion, initial settle anchored, post-settle loop untouched | ✓ VERIFIED | Byte-unchanged since `5ffdc2a0`; re-executed, PASS, N=5 reported. |
| `Taskfile.yml` (`test:tmux`, `TMUX_EXPECTED_TESTS: 6`) | Exact-count gate, unmodified logic | ✓ VERIFIED | `test:tmux` task body byte-unchanged since `5ffdc2a0`; only unrelated new targets (`docs:cli`, `docs:cli:drift`, `check:gonum`, `check:no-force-layout`) were appended elsewhere in the file by later phases. |
| `.github/workflows/ci.yml` (`tmux-e2e` job) | Installs tmux, asserts version, runs suite, gates on exact count | ✓ VERIFIED | Job block byte-unchanged since `5ffdc2a0`; the file's only change in this window is a `docs:cli:drift` step in an unrelated job. |
| `.planning/phases/08-tmux-real-pty-harness/08-01-PLAN.md` | Poll truth amended to readiness-anchored contract | ✓ VERIFIED | `rg -n "readiness" 08-01-PLAN.md` shows the amended `must_haves.truths` entry. |
| `internal/cli/daemon.go`, `internal/cli/tui/daemonpicker.go`, `internal/cli/tui/agentpicker.go`, `test/tmux/main_test.go` | Byte-identical to pre-gap-closure state | ✓ VERIFIED | `git diff --quiet 5ffdc2a0..HEAD` exits 0, independently re-run at HEAD (162 commits, phases 9-12, later than the prior pass's `5baba883` baseline). |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| Every `test/tmux/*_test.go` poll call site | `pollUntilStable`'s required `ready` parameter | Go compiler | ✓ WIRED | `go vet -tags tmux ./test/tmux/...` passes clean at HEAD; a call site with no anchor cannot compile. |
| `Taskfile.yml: TMUX_EXPECTED_TESTS` | `test:tmux`'s exact-count gate | shell `-ne` comparison | ✓ WIRED | Re-ran `task test:tmux` at HEAD: `executed=6 skipped=0 expected=6`, exit 0. |
| `.github/workflows/ci.yml: tmux-e2e` | `task test:tmux:install` / `task test:tmux` | bare `run:` lines | ✓ WIRED (design AND live) | Live-fired at commit `5ffdc2a0`, job `103457354620`, conclusion `success` — see truth #9. Job/target both unchanged since that commit. |
| `internal/upgrade/taskfile_shape_test.go: inScopeJobs` | `.github/workflows/ci.yml` jobs on disk | `TestInScopeJobsPopulationMatchesDisk` | ✓ WIRED | `go test -count=1 ./internal/upgrade/...` re-run at HEAD: `ok`. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Full harness suite at HEAD | `GOTOOLCHAIN=go1.26.6 go vet -tags tmux ./test/tmux/...` | exit 0, no output | ✓ PASS |
| Full suite via Task target | `GOTOOLCHAIN=go1.26.6 task test:tmux` | `test:tmux: executed=6 skipped=0 expected=6`, exit 0, 6/6 named tests PASS | ✓ PASS |
| Repo-wide build | `GOTOOLCHAIN=go1.26.6 go build ./...` | exit 0 | ✓ PASS |
| CI-shape guard tests | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/upgrade/...` | `ok` | ✓ PASS |
| Debt-marker scan on phase-touched files | `rg -n "TBD\|FIXME\|XXX" test/tmux/ Taskfile.yml` | no matches (one `mktemp -d ... XXXXXX` placeholder, not a debt marker) | ✓ PASS |
| Byte-clean scope proof since prior CI evidence | `git diff --quiet 5ffdc2a0..HEAD -- internal/cli/daemon.go internal/cli/tui/daemonpicker.go internal/cli/tui/agentpicker.go test/tmux/` | exit 0 | ✓ PASS |
| Byte-clean scope proof, harness/CI job/Task target | `git diff --stat 5ffdc2a0..HEAD -- test/tmux Taskfile.yml .github/workflows/ci.yml internal/upgrade/taskfile_shape_test.go` | Only `Taskfile.yml` (+227, unrelated new targets) and `ci.yml` (+7, unrelated new step) changed; both diffs read in full | ✓ PASS |
| Live CI backstop for TTY-07 at `expected=6` | `gh pr view 69 --json headRefOid,state,mergedAt`; `gh run view 34658987243 --json conclusion,headSha,status`; `gh run view --job 103457354620 --log` | PR #69 MERGED at `5ffdc2a0`; run `conclusion: success`, `headSha: 5ffdc2a0`; job log contains `test:tmux: executed=6 skipped=0 expected=6` and six `--- PASS:` lines | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` files exist for this phase and neither the PLANs nor SUMMARYs
reference any. Step 7c: SKIPPED (no probes declared or discovered).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| TTY-01 | 08-01 | Build-tagged harness, real pane, skip-with-reason when tmux absent | ✓ SATISFIED | Truth #1 |
| TTY-02 | 08-01, amended by 08-05 | Poll-to-convergence before asserting, no fixed sleeps; proven against a genuinely racing case | ✓ SATISFIED | Truths #2, #8 |
| TTY-03 | 08-01, amended by 08-05 | Empty-registry DECRQM leak-free, watched fail; G-08-1 cold-start race closed | ✓ SATISFIED | Truths #3, #14 |
| TTY-04 | 08-02 | Alt-screen entry/exit, watched fail | ✓ SATISFIED | Truth #4 |
| TTY-05 | 08-02 | Checkbox glyphs/toggle/cancel zero-write, watched fail | ✓ SATISFIED | Truth #5 |
| TTY-06 | 08-02 | Idle frame stability, N reported | ✓ SATISFIED | Truth #6 |
| TTY-07 | 08-03, TMUX_EXPECTED_TESTS bumped by 08-05 | CI job design + exact-count gate, live-fired | ✓ SATISFIED (mechanism AND live run both proven at `expected=6`) | Truths #7, #9 |

No orphaned requirements found: `.planning/REQUIREMENTS.md` maps exactly TTY-01…TTY-07 to Phase 8
(spot-checked this pass via `rg -n "TTY-0" .planning/REQUIREMENTS.md`; not included in
`covered_files` per the re-verification convention, since it is rewritten by every subsequent
`phase.complete`).

### Anti-Patterns Found

No new anti-patterns were introduced in the window since the prior pass (`5ffdc2a0`..HEAD touched
only unrelated Taskfile/ci.yml additions from Phases 11-12). The prior pass's findings on
phase-touched files are carried forward unchanged:

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `test/tmux/capture.go` | 190-201 (`paneContains`); doc comment 127-133 | WR-06 (08-REVIEW.md — excluded from `covered_files` this pass but the finding itself still applies to the unchanged code): the anchor-required mechanism only mechanically rejects an *empty* needle, not a needle that also appears in the pre-output echoed command line | ⚠️ Warning | Does not affect any of the 11 current call sites; a real risk for future call sites added without following the documented rule. Non-blocking. |
| `test/tmux/poll_contract_test.go` | 52-54 | IN-05: `start := time.Now()` is captured before `sendKey(t, session, "Enter")`, so `elapsed` slightly over-counts genuine wait time | ℹ️ Info | Conservative direction only; cosmetic. |
| `test/tmux/capture.go` | 113 | IN-01: comment "no blocking pause call" is technically imprecise | ℹ️ Info | Cosmetic. |
| `test/tmux/capture.go` / `frame_stability_test.go` | — | WR-01/WR-02: TTY-06's idle-stability loop necessarily samples on a fixed interval outside `pollUntilStable`, contradicting the package doc comment's absolute "ONLY wait primitive" wording | ⚠️ Warning | Architecturally necessary, doc-wording issue only. |
| `Taskfile.yml` | `test:tmux` cmds block | WR-03 | ⚠️ Warning | Unchanged since `5ffdc2a0`. |
| `test/tmux/daemon_seed.go` | teardown | WR-04 | ⚠️ Warning | Unchanged since `5ffdc2a0`. |
| `test/tmux/confighash.go` | 63-64 | WR-05 | ⚠️ Warning | Unchanged since `5ffdc2a0`. |

No 🛑 Blocker-severity findings and no unresolved debt markers (`TBD`/`FIXME`/`XXX`) in any
phase-touched file at HEAD.

## Gaps Summary

No `gaps_found`-tier defects were identified. All 15 must-haves are `✓ VERIFIED` at HEAD
`8c8149de`: 13 by direct regression confirmation (byte-identical files since the last-evidenced
commit `5ffdc2a0`, re-executed tests still passing), and the two behavior-dependent backstop
truths (TTY-02's harness-level self-test and TTY-07's CI-level exact-count gate) by fresh,
independently re-derived evidence — the harness-level self-test re-run directly in this pass, and
the CI-level backstop re-confirmed live via `gh pr view` / `gh run view --json` / `gh run view
--job ... --log` rather than accepted from `08-UAT.md`'s narrative alone.

This closes the prior pass's sole open item: TTY-07's live CI backstop at `expected=6` was
`human_needed` on 2026-09-11 because no CI run existed yet for the post-08-05 code; PR #69 has
since merged at commit `5ffdc2a0`, CI run `34658987243`/job `103457354620` fired green with
`test:tmux: executed=6 skipped=0 expected=6`, and nothing relevant has changed between that commit
and current HEAD (162 commits later, none touching `test/tmux/**`, the `tmux-e2e` job, or the
`test:tmux` Task target). Status moves from `human_needed` (14/15) to `passed` (15/15).

One advisory-level finding carries forward unchanged: WR-06 (originally `08-REVIEW.md`, now
excluded from `covered_files` per the re-verification convention but still a valid observation
about unchanged code) notes the readiness-anchor mechanism enforces the empty-needle vacuity case
mechanically but relies on doc-comment discipline alone for the "needle present in the pre-output
frame" vacuity case at any *future* call site — non-blocking for this phase (all 11 current call
sites are clean).
