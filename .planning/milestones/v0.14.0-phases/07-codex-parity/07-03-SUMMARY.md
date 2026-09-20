---
phase: 07-codex-parity
plan: 03
subsystem: tui
tags: [bubbletea, bubbles-v2, lipgloss, tui, tdd, mutation-testing, tmux]

# Dependency graph
requires:
  - phase: 07-codex-parity
    provides: "install/uninstall --target-vs---yes resolution order (07-02) — untouched by this plan"
provides:
  - "checkboxDelegate.Render / daemonDelegate.Render write exactly one line per row (D-24)"
  - "TestAgentPickerFootprintFitsDefaultPane, TestAgentPickerFootprint_HeightBoundaries, TestAgentPickerFootprint_EmptyRoster, TestDaemonPickerFootprintFitsDefaultPane"
  - "TTY-05 re-anchored on the footer text `space: toggle` + all 8 display names (D-25)"
  - "07-MUTATION-LOG.md Family (c): c1 (agentpicker.go), c2 (daemonpicker.go), c3 (not run — maintainer decision)"
affects: [07-09]

# Actuals (#2632)
actuals:
  tokens: 5589
  tasks: 3
  commits: 5
plan_head_before: 6c4cc4e7f0d310937cd1a39030b9211fd29e5758

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "bubbles v2's list.populatedView already inserts a row separator — a delegate must render exactly one line and never append its own trailing newline, or every row costs 2 lines against the Height() budget"

key-files:
  created: []
  modified:
    - internal/cli/tui/agentpicker.go
    - internal/cli/tui/daemonpicker.go
    - internal/cli/tui/agentpicker_test.go
    - internal/cli/tui/daemonpicker_test.go
    - test/tmux/install_cancel_test.go
    - .planning/phases/07-codex-parity/07-MUTATION-LOG.md

key-decisions:
  - "Family (c3) NOT performed as planned: the orchestrator did not install tmux or run `task test:tmux`. Maintainer decision (2026-09-19, verbatim): \"tmux was replaced with herdr, not kubectx. skip the tmux evidence, open a GH issue to consider a move from tmux to herdr, or if the e2e that we use tmux for is still needed (or if there are other options)\". Recorded in 07-MUTATION-LOG.md; follow-up filed at https://github.com/seanb4t/codegraph-go/issues/75. CI's tmux-e2e job (ubuntu-latest, tmux 3.4) remains the only real-PTY run of the re-anchored TTY-05 assertion; 07-09's post-scope-flip tmux re-run is skipped under the same decision."
  - "Pre-existing gofmt drift in test/tmux/poll_contract_test.go (both local go1.27.1 gofmt and the pinned go1.26.6 gofmt) is out of scope — untouched by this plan since v0.13.0 (#69, 3da59354) — and deferred rather than fixed; see Deviations."

patterns-established: []

requirements-completed: []  # FIX-03 is shared with 07-09 (post-scope-flip TTY-05 re-run) — not fully closed until that plan lands; see Next Phase Readiness.

coverage:
  - id: D1
    description: "checkboxDelegate.Render and daemonDelegate.Render write exactly one line per row (no trailing newline); four model-level footprint tests prove the agent/daemon pickers fit a 100x30 pane with all 8 targets/records and show their help footer, including height-boundary (29/30/31, 1/2) and empty-roster subtests"
    requirement: "FIX-03"
    verification:
      - kind: unit
        ref: "internal/cli/tui/agentpicker_test.go#TestAgentPickerFootprintFitsDefaultPane"
        status: pass
      - kind: unit
        ref: "internal/cli/tui/agentpicker_test.go#TestAgentPickerFootprint_HeightBoundaries"
        status: pass
      - kind: unit
        ref: "internal/cli/tui/agentpicker_test.go#TestAgentPickerFootprint_EmptyRoster"
        status: pass
      - kind: unit
        ref: "internal/cli/tui/daemonpicker_test.go#TestDaemonPickerFootprintFitsDefaultPane"
        status: pass
    human_judgment: false
  - id: D2
    description: "07-MUTATION-LOG.md Family (c1)/(c2) prove the model-level footprint guards fail against the byte-exact pre-fix delegate (re-planted trailing newline), each reverted byte-clean with a green control run after"
    verification:
      - kind: other
        ref: "perl-planted re-shadowing mutation and revert session, transcripts pasted into 07-MUTATION-LOG.md Family (c1)/(c2); git diff --quiet gates before/after each plant and revert"
        status: pass
    human_judgment: false
  - id: D3
    description: "Family (c3): real-PTY tmux RED/GREEN evidence for the re-anchored TTY-05 assertion, on local tmux"
    verification: []
    human_judgment: true
    rationale: "Maintainer explicitly decided (2026-09-19) to skip local tmux evidence rather than install tmux, deferring to GH issue #75 and to the CI tmux-e2e job as the only remaining real-PTY run. Whether that CI job passes for the re-anchored assertion cannot be asserted here; it must be judged when that job next runs."

duration: ~15min (Tasks 1-2 per prior executor's own accounting; this continuation covered SUMMARY/state only)
completed: 2026-09-19
status: complete
---

# Phase 7 Plan 3: Picker Footer Overflow (FIX-03) Summary

**Fixed the agent/daemon picker footer overflow at its cause — `checkboxDelegate.Render`/`daemonDelegate.Render` were writing a trailing newline on top of bubbles v2's own row separator, doubling every row's line cost and hiding the help footer at the default 100x30 pane — proven by a failing model-level test first, then by re-planting and reverting the exact pre-fix byte pattern (Family c1/c2); the real-PTY tmux confirmation (Family c3) was explicitly waived by the maintainer in favor of CI-only coverage and a follow-up issue.**

## Performance

- **Duration:** ~15min (Tasks 1-2, per the plan's own prior-executor accounting); this continuation session wrote only the SUMMARY and state updates
- **Started:** 2026-09-19 (commits span 12:33–~16:00 local)
- **Completed:** 2026-09-19
- **Tasks:** 3 (Task 3 resolved by maintainer decision rather than performed as planned)
- **Files modified:** 6

## Accomplishments

- `checkboxDelegate.Render` (agentpicker.go) and `daemonDelegate.Render` (daemonpicker.go) now write exactly `fmt.Fprintf(w, "%s%s %s", cursor, box, ai.target.DisplayName())` / `fmt.Fprintf(w, "%s%s (pid %d, up %s)", cursor, filepath.Base(di.record.RepoRoot), di.record.PID, age)` — no trailing `\n` — since `bubbles/v2/list`'s `populatedView` already supplies the row separator. `helpFooterLines`, list sizing, and pagination are untouched, per D-24's own prohibition.
- Four new model-level tests (`TestAgentPickerFootprintFitsDefaultPane`, `TestAgentPickerFootprint_HeightBoundaries`, `TestAgentPickerFootprint_EmptyRoster`, `TestDaemonPickerFootprintFitsDefaultPane`) prove the fix at a 100x30 pane with the real registry's 8 targets, at height boundaries 29/30/31 (footer present) and 1/2 (no panic), and against an empty roster.
- `test/tmux/install_cancel_test.go`'s TTY-05 assertion is re-anchored on the footer text `space: toggle` plus all 8 real display names, replacing the prior title-text workaround (08-04's documented positive control); `TMUX_EXPECTED_TESTS` stays 6 — no top-level test was added.
- `07-MUTATION-LOG.md` Family (c) proves the guards are not vacuous: (c1) re-plants the pre-fix newline in `agentpicker.go`, reproducing the exact pre-fix height (37 > 30) and turning `TestAgentPickerFootprintFitsDefaultPane` RED; (c2) does the same for `daemonpicker.go` against `TestDaemonPickerFootprintFitsDefaultPane`; both revert byte-clean with a green control run after.
- Family (c3) — the real-PTY tmux RED/GREEN run — was **not performed**. The maintainer decided to skip local tmux evidence entirely (tmux has been replaced by herdr on the maintainer's machines) rather than have the orchestrator install it; a follow-up issue (#75) tracks whether to move the harness to herdr, a pure-Go PTY, or teatest, or retire it. CI's `tmux-e2e` job (ubuntu-latest, tmux 3.4) remains the only real-PTY confirmation of the re-anchored assertion, at PR time.

## Task Commits

Each task was committed atomically:

1. **Task 1 RED: failing picker footprint tests + TTY-05 re-anchor** — `53efa074` (test)
2. **Task 1 GREEN: one-line delegate fix** — `28cf3eb0` (fix)
3. **Task 2: Family (c1)/(c2) mutation log** — `54bf2b6a` (docs)
4. **Task 3 (orchestrator, resolved by maintainer decision): Family (c3) recorded as skipped** — `2085a47e` (docs)

**Plan metadata:** committed separately after this SUMMARY (see below).

## Files Created/Modified

- `internal/cli/tui/agentpicker.go` — `checkboxDelegate.Render` drops its trailing newline; doc comment updated to say the list supplies the row separator
- `internal/cli/tui/daemonpicker.go` — `daemonDelegate.Render` gets the identical fix and comment update
- `internal/cli/tui/agentpicker_test.go` — `TestAgentPickerFootprintFitsDefaultPane`, `TestAgentPickerFootprint_HeightBoundaries` (subtests h1/h2/h29/h30/h31), `TestAgentPickerFootprint_EmptyRoster`
- `internal/cli/tui/daemonpicker_test.go` — `TestDaemonPickerFootprintFitsDefaultPane`
- `test/tmux/install_cancel_test.go` — TTY-05 positive control switched from the picker title to the footer text (`space: toggle` + all 8 display names); explanatory comment rewritten to describe the footer as the positive control since FIX-03
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — Family (c1)/(c2) mutation entries with verbatim RED transcripts and revert proofs; Family (c3) recorded as not run (maintainer decision), with the follow-up issue link

## RED Transcript

### Task 1 RED — model-level footprint tests (pre-fix delegate, `test(07-03)` commit `53efa074`)

The pre-fix delegate reproduces the same defect Family (c1) later re-plants and captures verbatim (07-MUTATION-LOG.md lines 460-507):

```
    agentpicker_test.go:212: lipgloss.Height(view.Content) = 37, want <= 30 at 100x30 with 8 targets:
           Select agents to configure
        ...
        space: toggle  enter: confirm  q/esc: cancel
--- FAIL: TestAgentPickerFootprintFitsDefaultPane (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli/tui	0.348s
FAIL
exit=1
```

(daemon picker equivalent, also height 37, in Family (c2): `--- FAIL: TestDaemonPickerFootprintFitsDefaultPane`, 07-MUTATION-LOG.md lines 553-600.)

**Measured height before the fix: 37 lines** (8 rows x 2 lines/row + 4 title/blank/footer lines + trailing pagination blanks), 7 over the 30-line budget — the footer never renders.

### Post-fix GREEN (`fix(07-03)` commit `28cf3eb0`, re-verified at this HEAD)

```
ok  	github.com/seanb4t/codegraph-go/internal/cli/tui	0.317s
```

All four footprint tests, both height-boundary subtest groups, and the full `internal/cli/tui` package pass at HEAD. **Measured height after the fix: <= 30 lines** at 100x30 with 8 targets/records, footer text present in both pickers.

## Decisions Made

- Family (c3) is recorded as **not run**, per explicit maintainer instruction, rather than the orchestrator installing tmux locally as the plan's Task 3 originally specified. See Deviations below.
- The gofmt drift in `test/tmux/poll_contract_test.go` is left untouched and deferred rather than fixed inline, since it predates this plan (since v0.13.0, #69, commit `3da59354`) and fixing it (gofmt's doc-comment rewrite would corrupt the literal `''` in the documented `cgtmux-''ready"` example) is out of this plan's scope.

## Deviations from Plan

### Auto-fixed Issues

None — Tasks 1 and 2 executed exactly as written; both fixes (Fprintf newline removal) and the mutation-log entries (c1)/(c2) match the plan's pinned mutation sites and verify gates byte for byte.

### Maintainer-directed deviation (not a Rule 1-4 auto-fix)

**Task 3 (checkpoint:human-action, gate=blocking-human) was resolved by explicit maintainer decision instead of being performed as planned.**
- **Planned:** the orchestrator installs tmux locally (`brew install tmux`), runs `task test:tmux` GREEN at HEAD (executed=6), re-plants Family (c1)'s exact mutation to show TTY-05 RED, reverts byte-clean, and records tmux version + both transcripts in Family (c3).
- **What happened instead:** the maintainer responded (2026-09-19, verbatim): *"tmux was replaced with herdr, not kubectx. skip the tmux evidence, open a GH issue to consider a move from tmux to herdr, or if the e2e that we use tmux for is still needed (or if there are other options)."* No tmux was installed; `task test:tmux` was never run locally. The (c3) placeholder was replaced with a "not run" record documenting the decision (commit `2085a47e`), and a follow-up issue was opened: https://github.com/seanb4t/codegraph-go/issues/75.
- **Consequence:** the local guard for FIX-03 is the model-level footprint test (Families c1/c2 above — RED on the pre-fix delegate, GREEN at HEAD). The re-anchored TTY-05 assertion compiles (`go vet -tags tmux ./test/tmux/`, confirmed clean at this HEAD) but its real-PTY run has not been exercised on this machine; it is left to the CI `tmux-e2e` job (ubuntu-latest, tmux 3.4). 07-09's planned post-scope-flip tmux re-run is skipped under the same decision.
- **Files affected:** `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` only (Family (c3) section).

### Deferred (out of scope)

**Pre-existing gofmt drift in `test/tmux/poll_contract_test.go`.** Both the local `go1.27.1` gofmt and the pinned `GOTOOLCHAIN=go1.26.6` gofmt flag this file — untouched since v0.13.0 (#69, commit `3da59354`). gofmt's doc-comment rewrite would turn the literal `''` in the file's documented `cgtmux-''ready` example into a typographic quote, corrupting the example. Not fixed here; recorded as deferred. All files this plan actually touched (`internal/cli/tui/agentpicker.go`, `internal/cli/tui/daemonpicker.go`, `internal/cli/tui/agentpicker_test.go`, `internal/cli/tui/daemonpicker_test.go`, `test/tmux/install_cancel_test.go`) are gofmt-clean.

---

**Total deviations:** 1 maintainer-directed (Task 3 resolution), 1 deferred pre-existing issue (out of scope, not fixed).
**Impact on plan:** FIX-03's cause is fixed and proven at the model level; the real-PTY confirmation is deferred to CI per maintainer decision rather than lost — no scope creep, no silently-skipped verification.

## Issues Encountered

None beyond the Task 3 resolution documented above.

## User Setup Required

None — no external service configuration required. (tmux installation was explicitly waived by the maintainer, not deferred to the user.)

## Next Phase Readiness

- FIX-03 is **not yet fully closed**: this plan fixes the delegate and proves it at the model level (Family c1/c2), but 07-09's frontmatter (`requirements: [CODEX-05, CODEX-06, FIX-03]`) explicitly re-runs the TTY-05 assertion after the Codex scope flip (line 27: "The picker's tmux TTY-05 assertion is re-run after the scope flip and passes with executed=6 (D-25, FIX-03)"). REQUIREMENTS.md's FIX-03 row stays unchecked until that re-run lands — or, per the maintainer's decision recorded here, until issue #75 resolves how (or whether) that re-run happens without local tmux.
- The re-anchored TTY-05 assertion (`space: toggle` + all 8 display names) is in place and compiles under `go vet -tags tmux`; it has not been exercised against a real PTY on this machine. 07-09 should re-check the maintainer's decision on #75 before assuming a local tmux run is expected there.
- Both non-test files touched by this plan (`agentpicker.go`, `daemonpicker.go`) are byte-clean at rest (`git diff --quiet -- internal/cli/tui/` exits 0), matching the plan's acceptance criteria.
- 07-VALIDATION row `07-FIX03` is satisfiable at the model level today (`GOTOOLCHAIN=go1.26.6 go test ./internal/cli/tui/ -count=1` green); its real-PTY leg is pending CI or a future decision on #75.

## Self-Check: PASSED

- `internal/cli/tui/agentpicker.go` — one-line Fprintf fix present — FOUND (`fmt.Fprintf(w, "%s%s %s", cursor, box, ai.target.DisplayName())`)
- `internal/cli/tui/daemonpicker.go` — one-line Fprintf fix present — FOUND (`fmt.Fprintf(w, "%s%s (pid %d, up %s)", cursor, filepath.Base(di.record.RepoRoot), di.record.PID, age)`)
- `internal/cli/tui/agentpicker_test.go` — `TestAgentPickerFootprintFitsDefaultPane`, `TestAgentPickerFootprint_HeightBoundaries`, `TestAgentPickerFootprint_EmptyRoster` — FOUND
- `internal/cli/tui/daemonpicker_test.go` — `TestDaemonPickerFootprintFitsDefaultPane` — FOUND
- `test/tmux/install_cancel_test.go` — `space: toggle` anchor present, no top-level test added (`^func Test` count = 1) — FOUND
- `.planning/phases/07-codex-parity/07-MUTATION-LOG.md` — Family (c1)/(c2)/(c3) — FOUND
- Commit `53efa074` (test) — FOUND in `git log --oneline`
- Commit `28cf3eb0` (fix) — FOUND in `git log --oneline`
- Commit `54bf2b6a` (docs) — FOUND in `git log --oneline`
- Commit `2085a47e` (docs) — FOUND in `git log --oneline`
- `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/tui/ -count=1` → `ok` — re-verified at this HEAD
- `GOTOOLCHAIN=go1.26.6 go vet -tags tmux ./test/tmux/` → clean — re-verified at this HEAD
- `git diff --quiet -- internal/cli/tui/ test/tmux/` — clean at rest — re-verified at this HEAD

---
*Phase: 07-codex-parity*
*Completed: 2026-09-19*
