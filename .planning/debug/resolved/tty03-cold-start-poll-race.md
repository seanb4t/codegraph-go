---
status: root_cause_found
gap: G-08-1
phase: 08-tmux-real-pty-harness
created: 2026-09-10
head: 5baba883
---

# TTY-03 cold-start poll race

## Symptom

`TestDaemonEmptyRegistryLeaksNoModeQueryBytes` (test/tmux/daemon_empty_test.go) fails with:

```
TTY-03: stabilized capture does not contain "no running daemons" — the harness may not have observed the real command run:
<capture: the echoed command line, then blank rows — no output, no new prompt>
```

Observed 3/3 consecutive runs at session start on a cold macOS dev machine (tmux 3.7c), at HEAD
5baba883, with `TestMain` building the binary. Not observed again in ~13 later runs once the
machine was warm. Passed on ubuntu-latest in CI (PR #69 run 34607117422) in 2.17s.

## Root cause

`pollUntilStable` (test/tmux/capture.go:102) samples the pane at t≈0 and again after
`stabilityPollInterval = 1s`, returning the first sample that byte-equals its predecessor.
If the spawned `codegraph daemon` process has not yet written its first stdout line by the
second sample, both samples are the identical pre-output frame and the poll converges on it.

The first exec of a freshly built 80 MB binary exceeds 1 s on this machine; every later exec
is well under it. `TestMain` (test/tmux/main_test.go:118) builds a fresh binary per run into a
new temp dir and never execs it before `m.Run()`; `daemon_empty_test.go` is the first test to
exec it, so TTY-03 alone pays the cold cost.

capture.go:16-27 justifies the 1 s interval on a measured "700-900 ms" latency and claims it
"exceeds the measured worst case with margin, so consecutive samples can never both land inside
that window by construction." That measurement was of a warm binary; the claim is false on the
cold path.

## Evidence

| Experiment | Result |
|---|---|
| TTY-03, fresh TestMain build, cold machine (session start) | FAIL 3/3 |
| `codegraph daemon` direct, empty registry | prints `no running daemons`, exit 0 — product OK |
| First exec of a fresh build, shell-timed | 1447 ms; then 69-190 ms warm |
| Six fresh builds, first exec each | 739, 1109, 1124, 1119, 1120, 1230 ms (5/6 > 1000) |
| In-pane (tmux send-keys → "no running daemons" visible) | first exec 1567 ms; next four 167-226 ms |
| TTY-03 via `CODEGRAPH_TEST_BIN` pointing at a pre-warmed binary | PASS 2/2 |
| TTY-03, fresh builds, machine now warm | PASS 4/4, then 3/3 cold-arm + 3/3 warm-arm |
| CI ubuntu-latest, fresh build, first exec | PASS, 2.17s elapsed |

Invalidated experiment (recorded so it is not repeated): one arm timed the first exec with a
shell probe and THEN ran the test against the same binary — the probe had warmed it, so that
arm proved nothing. Re-run properly as the cold-arm / warm-arm pair above.

## Fix directions (non-binding — the binding payload is the root cause above)

1. Warm the binary once in `TestMain` before `m.Run()` (a single throwaway exec, e.g.
   `codegraph --version`), so no test pays the cold cost. Smallest change; keeps
   `pollUntilStable`'s contract untouched.
2. Give `pollUntilStable` a positive anchor: convergence is accepted only once the capture
   also contains the shell prompt returning after the command (the command has finished), so a
   pre-output frame can never be "stable". Stronger, but changes the primitive's contract and
   every caller's assumptions.
3. Raise `stabilityPollInterval` with a cold-path measurement recorded in the comment. Weakest —
   it moves the cliff rather than removing it.

Whichever is chosen, capture.go:16-27's "by construction" paragraph must be rewritten to state
what is actually guaranteed.
