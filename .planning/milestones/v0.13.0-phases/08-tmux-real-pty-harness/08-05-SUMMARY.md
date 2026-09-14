---
phase: 08-tmux-real-pty-harness
plan: 05
subsystem: testing
tags: [tmux, go-test, tdd, gap-closure, real-pty-harness]

# Dependency graph
requires:
  - phase: 08-tmux-real-pty-harness
    provides: "08-01's pollUntilStable primitive, 08-02's daemon-seeding helper, 08-04's mutation-log discipline and TMUX_EXPECTED_TESTS contract"
provides:
  - "pollUntilStable(t, session, ready) with a required readiness predicate — convergence is ready(capture) && capture == predecessor"
  - "paneContains and altScreenOff anchor constructors"
  - "A deterministic, on-demand self-test (TestPollUntilStableDoesNotConvergeOnPreOutputFrame) that reproduces G-08-1's race without depending on machine coldness"
  - "TMUX_EXPECTED_TESTS = 6"
  - "A truthful capture.go header comment recording cold vs warm exec measurements"
  - "08-01's poll-truth wording amended to the readiness-anchored contract"
affects: [08-tmux-real-pty-harness, any-future-tmux-harness-test]

gap_ids: [G-08-1]

actuals:
  tokens: 5411
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Wait primitives that can converge on a not-yet-ready stable frame must take a caller-stated readiness predicate, not rely on interval tuning alone."
    - "Deterministic self-tests of a harness's own timing primitive (commanding a fixed shell delay) are preferable to relying on a product binary's incidental cold-start latency to reproduce a timing race."

key-files:
  created:
    - test/tmux/poll_contract_test.go
  modified:
    - test/tmux/capture.go
    - test/tmux/daemon_empty_test.go
    - test/tmux/daemon_picker_test.go
    - test/tmux/install_cancel_test.go
    - test/tmux/frame_stability_test.go
    - Taskfile.yml
    - .planning/phases/08-tmux-real-pty-harness/08-01-PLAN.md

key-decisions:
  - "Chose fix direction 2 (readiness predicate) over direction 1 (warm the binary in TestMain) and direction 3 (raise stabilityPollInterval) — see 'Why fix directions 1 and 3 were not taken' below."
  - "Cold-arm evidence runs all showed K<=1 (machine was warm from repeated test invocations during this session) — per the plan's own interpretation rule, did not re-run chasing K>=2; the deterministic self-test's RED/GREEN transcripts carry the proof instead."

patterns-established:
  - "pollUntilStable's readiness predicate pattern: paneContains for content a launch/keystroke adds, altScreenOff for a quit key whose only guaranteed change is leaving alt-mode."

requirements-completed: [TTY-02, TTY-03]

coverage:
  - id: D1
    description: "pollUntilStable takes a required readiness predicate (ready func(capture string) bool); convergence is ready(capture) && capture == predecessor; nil predicate is an immediate t.Fatal; empty paneContains needle is an immediate t.Fatal."
    requirement: TTY-02
    verification:
      - kind: unit
        ref: "test/tmux/poll_contract_test.go#TestPollUntilStableDoesNotConvergeOnPreOutputFrame"
        status: pass
    human_judgment: false
  - id: D2
    description: "TestDaemonEmptyRegistryLeaksNoModeQueryBytes (TTY-03) passes on the first exec of a freshly built binary because the poll anchors on \"no running daemons\" and cannot converge on the pre-output frame."
    requirement: TTY-03
    verification:
      - kind: e2e
        ref: "test/tmux/daemon_empty_test.go#TestDaemonEmptyRegistryLeaksNoModeQueryBytes"
        status: pass
    human_judgment: false
  - id: D3
    description: "TMUX_EXPECTED_TESTS bumped 5 -> 6 in the same commit as the new self-test; task test:tmux reports executed=6 skipped=0 expected=6."
    verification:
      - kind: other
        ref: "Taskfile.yml#test:tmux"
        status: pass
    human_judgment: false
  - id: D4
    description: "capture.go's header comment and pollUntilStable's doc comment state only what is actually guaranteed, record the cold (1.1-1.6s in-pane / 739-1230ms shell-timed) vs warm (167-226ms, 08-01's 700-900ms trace) measurements, and delete the false 'exceeds the measured worst case by construction' claim."
    verification: []
    human_judgment: true
    rationale: "Doc-comment content accuracy is a prose/judgment call not mechanically verifiable beyond the numeric facts already covered by D1/D2's passing tests; a human should confirm the wording reads as intended."
  - id: D5
    description: "No production file (internal/cli/daemon.go, internal/cli/tui/daemonpicker.go, internal/cli/tui/agentpicker.go) or test/tmux/main_test.go changed; 08-01-PLAN.md's poll truth amended, nothing else in that file touched."
    verification:
      - kind: other
        ref: "git diff --quiet 5baba883..HEAD -- internal/cli/daemon.go internal/cli/tui/daemonpicker.go internal/cli/tui/agentpicker.go test/tmux/main_test.go"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-09-11
status: complete
---

# Phase 8 Plan 5: Close G-08-1 — anchor pollUntilStable on a readiness predicate Summary

**Gave `pollUntilStable` a required readiness predicate so a pre-output frame — stable to any byte-equality check — can never be accepted as converged; closes G-08-1's cold-start TTY-03 flake with a deterministic self-test watched RED then GREEN.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-09-11T21:32:00Z (approx)
- **Completed:** 2026-09-11T21:57:06Z
- **Tasks:** 3
- **Files modified:** 8 (1 created, 7 modified)

## Accomplishments

- `pollUntilStable(t, session, ready)` now requires a caller-stated readiness predicate; convergence is `ready(capture) && capture == predecessor`. A nil predicate is an immediate `t.Fatal`.
- Two anchor constructors added: `paneContains(t, needle)` (t.Fatal on empty needle) and `altScreenOff(t, session)` (re-queries `alternateOn` fresh every sample).
- All ten pre-existing poll call sites anchored, every downstream assertion left byte-unchanged.
- `TestPollUntilStableDoesNotConvergeOnPreOutputFrame` — a deterministic, on-demand reproduction of G-08-1's race — was watched FAIL against the pre-fix primitive and PASS after the fix.
- `capture.go`'s header comment rewritten to state only what is actually guaranteed, with the cold vs warm exec measurements recorded and the false "exceeds the measured worst case by construction" claim removed.
- `TMUX_EXPECTED_TESTS` bumped 5 → 6; `task test:tmux` reports `executed=6 skipped=0 expected=6`.
- 08-01-PLAN.md's stability-poll `must_haves.truths` entry amended to match the readiness-anchored contract.

## Task Commits

Each task was committed atomically:

1. **Task 1: A deterministic self-test watched FAIL against the pre-fix primitive** - `e994b0a5` (test)
2. **Task 2: Give pollUntilStable a required readiness predicate, anchor all call sites** - `b9dfc849` (fix)
3. **Task 3: Cold-arm evidence, byte-clean proofs, 08-01's poll truth amended** - `21063a2e` (docs)

**Plan metadata:** committed together with STATE.md/ROADMAP.md/REQUIREMENTS.md after this SUMMARY.

## Files Created/Modified

- `test/tmux/poll_contract_test.go` (created) — `TestPollUntilStableDoesNotConvergeOnPreOutputFrame`, the deterministic self-test
- `test/tmux/capture.go` — `pollUntilStable` gains a required `ready` predicate; `paneContains`/`altScreenOff` added; header and doc comments rewritten to the amended D-14 contract
- `test/tmux/daemon_empty_test.go` — poll call anchored on `"no running daemons"`, doc comment extended
- `test/tmux/daemon_picker_test.go` — poll calls anchored on `"Running daemons"` and `altScreenOff`
- `test/tmux/install_cancel_test.go` — poll calls anchored on `"[ ]"`, `"[x]"`, `altScreenOff` (both cancel paths)
- `test/tmux/frame_stability_test.go` — poll call anchored on `"[ ]"`; post-settle loop left byte-unchanged
- `Taskfile.yml` — `TMUX_EXPECTED_TESTS: 5` → `6`
- `.planning/phases/08-tmux-real-pty-harness/08-01-PLAN.md` — third `must_haves.truths` entry amended (one line, nothing else touched)

## Decisions Made

- Chose fix direction 2 (readiness predicate) — see "Why fix directions 1 and 3 were not taken" below.
- On seeing K<=1 on all three cold-arm runs (machine was warm from this session's own repeated test invocations), did not re-run chasing K>=2 — followed the plan's own interpretation rule and relied on the deterministic self-test's RED/GREEN transcripts as the proof instead.

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written for all three tasks; every action, doc-comment content, and call-site anchor matches the plan's `<action>` blocks verbatim.

### Gate-tooling note (not a plan deviation, reported per the gate-authoring-gotchas instruction)

`gsd_run check tdd-red-evidence` (the generic TDD RED-evidence classifier) parses TAP-format
output (`# tests N`, `# pass N`, `# fail N`, `ok/not ok N - <name>` lines) — the shape Node's
`node --test` runner emits. `go test -v` output is not TAP-formatted, so feeding it to this
classifier would always report `zero_tests_discovered` / `INVALID_RED` regardless of whether
the target Go test genuinely failed on its intended assertion. This is a tool/language gap in
gsd-core's generic TDD gate, not a defect in this plan or this codebase, and it lives outside
this project's repo (`$HOME/.claude/gsd-core`), so it was not "fixed" here. Per the gate-
authoring-gotchas guidance, the BEHAVIOR the gate exists to prove — an intentional RED against
the target test, followed by a GREEN — was satisfied instead via the plan's own Go-native
`<verify>` blocks (exact `rg` matches against `go test -v` output for the FAIL/PASS lines and
the specific failure/success messages), which is precisely the mechanism this plan's authors
designed and required verbatim transcripts for. Both transcripts are pasted below.

---

**Total deviations:** 0 auto-fixed. One gate-tooling gap reported (not fixed, not blocking — the plan's own verification mechanism substituted correctly and was itself run to completion).
**Impact on plan:** None on scope or correctness — every plan-authored `<verify>` command was executed and passed.

## Issues Encountered

None.

## RED Transcript (Task 1 — against the pre-fix `pollUntilStable`)

Command: `GOTOOLCHAIN=go1.26.6 go test -tags tmux -count=1 -run 'TestPollUntilStableDoesNotConvergeOnPreOutputFrame' -v ./test/tmux/...`

```
poll_contract_test.go:60: pollUntilStable converged on a frame without the anchor "cgtmux-ready" after 2.03400625s — the pre-output frame was accepted as stable:
        sleep 4; echo cgtmux-''ready
        sean@denver tmux % sleep 4; echo cgtmux-''ready




        (blank rows — no output line, no new prompt)




--- FAIL: TestPollUntilStableDoesNotConvergeOnPreOutputFrame (2.88s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/tmux	8.144s
FAIL
```

The poll converged at ~2.03s — inside the 4s commanded delay — on a frame showing only the
echoed `sleep 4; echo cgtmux-''ready` command line, with no `cgtmux-ready` output beneath it.
This is exactly the pre-output convergence G-08-1 describes, reproduced deterministically.

## GREEN Transcript (Task 2 — against the post-fix `pollUntilStable`)

Command: same as above, after Task 2's implementation.

```
    poll_contract_test.go:56: pollUntilStable: converged at sample 5 after 5.059206542s (ready first held at sample 4)
--- PASS: TestPollUntilStableDoesNotConvergeOnPreOutputFrame (5.81s)
ok  	github.com/seanb4t/codegraph-go/test/tmux	10.640s
```

Converged at sample 5 after 5.06s — past the 4s commanded delay — with readiness first holding
at sample 4 (K=4, satisfying the acceptance criterion "K at least 4 and D at least 4s").

## Cold-Arm Evidence (Task 3 — three consecutive fresh-build runs, `CODEGRAPH_TEST_BIN` unset)

Command (run three times, each its own process so `TestMain` builds a fresh binary into a
fresh temp dir each time): `GOTOOLCHAIN=go1.26.6 go test -tags tmux -count=1 -run 'TestDaemonEmptyRegistryLeaksNoModeQueryBytes' -v ./test/tmux/...`

**Run 1:**
```
    daemon_empty_test.go:48: pollUntilStable: converged at sample 2 after 2.017310166s (ready first held at sample 1)
--- PASS: TestDaemonEmptyRegistryLeaksNoModeQueryBytes (2.77s)
ok  	github.com/seanb4t/codegraph-go/test/tmux	6.309s
```

**Run 2:**
```
    daemon_empty_test.go:48: pollUntilStable: converged at sample 2 after 2.023342375s (ready first held at sample 1)
--- PASS: TestDaemonEmptyRegistryLeaksNoModeQueryBytes (2.94s)
ok  	github.com/seanb4t/codegraph-go/test/tmux	6.570s
```

**Run 3:**
```
    daemon_empty_test.go:48: pollUntilStable: converged at sample 2 after 2.014830459s (ready first held at sample 1)
--- PASS: TestDaemonEmptyRegistryLeaksNoModeQueryBytes (2.77s)
ok  	github.com/seanb4t/codegraph-go/test/tmux	6.529s
```

**K interpretation (honest, per the plan's rule):** all three runs show `ready first held at
sample 1` — K=1. Per the plan's interpretation rule, K<=1 means the first exec was fast enough
on this run that the pre-fix race would not have fired even without the anchor fix — this
development machine was warm during this evidence-gathering session (it had already run dozens
of `go test -tags tmux` invocations building fresh binaries in the preceding minutes, keeping
disk/page caches hot for the build output). **None of these three runs exercised the genuine
cold path**, and none is presented as if it had. Per the plan's explicit instruction, no further
re-runs were attempted chasing a K>=2 result. The deterministic self-test above — watched FAIL
at 2.03s (before the anchor fix) and PASS at 5.06s with K=4 (after it) — carries the actual
proof that the fix closes the race; that reproduction does not depend on machine coldness at
all, which is exactly why it was built deterministically in Task 1 rather than relying on the
codegraph binary's own incidental cold-start latency.

## Byte-Clean Proofs

```
$ git diff --quiet 5baba883..HEAD -- internal/cli/daemon.go internal/cli/tui/daemonpicker.go internal/cli/tui/agentpicker.go test/tmux/main_test.go
$ echo $?
0
```

```
$ git status --porcelain internal/cli test/tmux/main_test.go
(empty)
```

Both confirm: no production file and no part of `TestMain` changed by this gap-closure plan —
it is harness-only, and D-09's binary-resolution contract is untouched.

## Why Fix Directions 1 and 3 Were Not Taken

The debug session (`.planning/debug/tty03-cold-start-poll-race.md`) recorded three candidate fix
directions. This plan took direction 2 (readiness predicate) and deliberately rejected the other
two:

- **Direction 1 — warm the binary once in `TestMain` before `m.Run()`.** Rejected because it
  would hide the structural defect rather than fix it, AND it would remove the only genuine cold
  first exec the suite has: every future run would become the debug session's own invalidated
  experiment (a shell probe that warmed the binary before the real test ran, proving nothing).
  `TestMain` was deliberately left untouched — proven by the byte-clean check above.
- **Direction 3 — raise `stabilityPollInterval`.** Rejected because it is numeric, not
  structural: it moves the cliff to a longer latency rather than removing the race. A future
  slower cold exec (a bigger binary, a slower CI runner, a loaded dev machine) would reproduce
  the identical failure mode at a new threshold. The constants stay 1s/10s per the plan; the
  comment explicitly states retuning the interval is not the fix.

Direction 2 — the readiness predicate — is structural: it makes "settled" mean "settled AND the
caller's stated content/state is present," which no interval value can substitute for. This is
the amendment to D-14 that this plan makes explicit in both `capture.go`'s header comment and
`pollUntilStable`'s own doc comment.

## Invalidated-Experiment Pitfall (restated, so it is not repeated)

Recorded originally in `.planning/debug/tty03-cold-start-poll-race.md`: one arm of the original
debug session timed the first exec with a shell probe and THEN ran the real test against the
SAME already-warmed binary — the probe had warmed it, so that arm proved nothing about the cold
path. **In this plan and in any future work on this harness:** never exec the binary `TestMain`
builds before the test under evaluation does; never set `CODEGRAPH_TEST_BIN` for a cold-arm run;
never add a `--version` or `daemon` warm-up call to `TestMain`. The only process permitted to
touch a freshly built binary before the first test execs it is the `go build` inside `TestMain`
itself. This plan's Task 3 cold-arm runs followed this protocol exactly (see the byte-clean
proof that `test/tmux/main_test.go` is untouched).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- G-08-1 is closed: `TestDaemonEmptyRegistryLeaksNoModeQueryBytes` (TTY-03) no longer depends on
  the spawned binary's first exec landing inside a single `stabilityPollInterval`; the readiness
  predicate makes convergence on a pre-output frame structurally impossible regardless of
  startup latency, short of `stabilityPollDeadline`.
- `pollUntilStable` remains the package's only wait primitive; every call site (all ten
  pre-existing plus the new self-test) states its own anchor.
- No blockers for phase completion. This was the phase's only open gap per `08-UAT.md`; the
  deferred TTY-06 settling-transient follow-up was explicitly out of scope for this plan.

---
*Phase: 08-tmux-real-pty-harness*
*Completed: 2026-09-11*

## Self-Check: PASSED

- All 8 key-files (1 created, 7 modified) confirmed present on disk.
- All 3 task commits (`e994b0a5`, `b9dfc849`, `21063a2e`) confirmed present in `git log`.
- All plan-level `<verify>` commands re-run and PASSED: `go vet -tags tmux`, three fresh-build
  cold-arm TTY-03 runs, `task test:tmux` (`executed=6 skipped=0 expected=6`), byte-clean diff
  against `5baba883`, repo-wide `go build ./...` and `go vet ./...`.
- Plan commit ledger: `plan_head_before: a4590956acea21682ed682b66f729105a1b04806`,
  `commits: 3` (measured via `git rev-list --count`).
