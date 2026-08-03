---
phase: 07-interactive-tui-daemon-picker-install-multi-select
plan: 06
subsystem: cli
tags: [bubbletea, bubbles, charm, tui, install, uninstall, tdd]

# Dependency graph
requires:
  - phase: 07-interactive-tui-daemon-picker-install-multi-select
    provides: internal/cli/tui.InteractiveAllowed (07-01's dual-TTY gate) + the go.mod bubbletea/v2+bubbles/v2 pins
provides:
  - internal/cli/tui.RunAgentPicker — a bubbles/v2/list.Model + custom checkbox ItemDelegate multi-select
  - -y/--yes on both install and uninstall, short-circuiting to the non-interactive default FIRST in the target-resolution switch
  - uninstall gains the interactive picker it never had (install-only before this plan)
affects: [07-07 (or whichever later plan adds the daemon picker Model), phase 8's charm dependency-closure audit]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hand-rolled checkbox ItemDelegate over bubbles/v2/list.Model (list.Model has no built-in multi-select) — toggle logic lives in the delegate's OWN Update, never the outer Model's, to avoid colliding with list.Model's internal KeyMap dispatch"
    - "Outer bubbletea Model intercepts enter/quit keys itself, before ever forwarding to list.Model.Update, so 'confirm' and 'cancel' are unambiguous to the caller"
    - "Injectable func-var seam (interactiveAllowed/runAgentPicker) for testing a cobra RunE's interactive branch without a real pty or a real tea.Program"

key-files:
  created:
    - internal/cli/tui/agentpicker.go
    - internal/cli/tui/agentpicker_test.go
  modified:
    - internal/cli/install.go
    - internal/cli/uninstall.go
    - internal/cli/install_test.go
    - go.mod
    - go.sum

key-decisions:
  - "selectByIndices is duplicated (not imported) into internal/cli/tui — internal/cli imports internal/cli/tui (RunAgentPicker's caller), so the reverse import would be a cycle; kept byte-identical dedup+ascending-order semantics"
  - "list.Model's own Quit/ForceQuit keybindings are irrelevant by construction: the outer Update's switch on enter/q/esc/ctrl+c returns before ever calling list.Model.Update for those keys, so there's no keymap collision to resolve"
  - "uninstall's off-TTY/-y non-interactive default stays 'all' (its historical no-flag behavior), not 'auto' like install — the interactive picker is new for uninstall but the non-interactive fallback is unchanged from before this plan"
  - "interactiveAllowed/runAgentPicker are cli-package func vars defaulting to tui.InteractiveAllowed/tui.RunAgentPicker — lets install_test.go/uninstall_test.go force the interactive branch and stub the picker without ever constructing a real tea.Program in a test process"

patterns-established:
  - "Checkbox multi-select TUI: agentItem/checkboxDelegate/agentPickerModel in internal/cli/tui, consuming plain agents.AgentTarget/DetectAll data — the exact producers-plain/tui-renders seam Phase 6 established, extended to a stateful interactive component"

requirements-completed: [TUI-03, TUI-04]

coverage:
  - id: D1
    description: "RunAgentPicker + checkboxDelegate: bubbles checkbox multi-select pre-checked from DetectAll(loc), space toggles in the delegate's own Update, Enter resolves through selectByIndices (dedup + ascending order), quit yields empty"
    requirement: "TUI-03"
    verification:
      - kind: unit
        ref: "internal/cli/tui/agentpicker_test.go#TestAgentPickerModel_PreChecksDetectedTargets"
        status: pass
      - kind: unit
        ref: "internal/cli/tui/agentpicker_test.go#TestAgentPickerModel_SpaceTogglesFocusedRow"
        status: pass
      - kind: unit
        ref: "internal/cli/tui/agentpicker_test.go#TestAgentPickerModel_EnterResolvesCheckedSetInAscendingOrder"
        status: pass
      - kind: unit
        ref: "internal/cli/tui/agentpicker_test.go#TestAgentPickerModel_QuitYieldsEmptySelection"
        status: pass
    human_judgment: false
  - id: D2
    description: "-y/--yes on install and uninstall short-circuits to the non-interactive default FIRST in the switch, even with the TTY branch forced allowed, and never invokes the picker"
    requirement: "TUI-03"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_Yes_ShortCircuitsBeforeInteractiveBranch"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestUninstall_Yes_ShortCircuitsBeforeInteractiveBranch"
        status: pass
    human_judgment: false
  - id: D3
    description: "The interactive branch (tui.InteractiveAllowed true, no -y, no --target) does call RunAgentPicker and uses its resolved targets, for both install and uninstall"
    requirement: "TUI-03"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_InteractiveAllowed_CallsRunAgentPicker"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestUninstall_InteractiveAllowed_CallsRunAgentPicker"
        status: pass
    human_judgment: false
  - id: D4
    description: "Real interactive checkbox picker rendering in an actual terminal (glyphs, cursor movement, help line legibility)"
    verification: []
    human_judgment: true
    rationale: "No real pty/terminal available in this execution environment to visually confirm the rendered checkbox list; the Model's state-transition logic is fully unit-tested (D1) but the on-screen render was not eyeballed in a live terminal."

duration: 13min
completed: 2026-07-18
status: complete
---

# Phase 7 Plan 06: Bubbles Checkbox Multi-Select for Install/Uninstall Summary

**Replaced install's plain numbered-line prompt with a hand-rolled bubbles/v2/list checkbox multi-select (pre-checked from `agents.DetectAll`), gave uninstall the same picker it never had, and added `-y`/`--yes` to both commands as a hard short-circuit before the TTY branch.**

## Performance

- **Duration:** ~13 min
- **Tasks:** 2 (Task 1 TDD: RED then GREEN; Task 2: execute)
- **Files modified:** 6 (2 new, 4 modified) + go.mod/go.sum (transitive deps only)

## Accomplishments
- `internal/cli/tui/agentpicker.go`: `agentItem`/`checkboxDelegate`/`agentPickerModel`/`RunAgentPicker` — a bubbles/v2/list.Model with a custom checkbox `ItemDelegate` (bubbles has no built-in multi-select). The delegate toggles `checked[index]` in its own `Update` on a space key press (never the outer Model's `Update`), avoiding any collision with `list.Model`'s internal KeyMap dispatch. The outer Model intercepts `enter`/`q`/`esc`/`ctrl+c` itself, before ever forwarding those keys to `list.Model.Update`, so "confirm" (resolve the checked set through `selectByIndices`) and "cancel" (empty selection) are unambiguous.
- `-y`/`--yes` added to both `install` and `uninstall` (`BoolVarP`), checked **first** in each command's target-resolution `switch` — it always short-circuits to the non-interactive default (`auto` for install, `all` for uninstall) regardless of what the TTY branch would otherwise decide, per D-15/RESEARCH Pitfall 6.
- `uninstall` gained the interactive picker path it never had before this plan: on an allowed TTY with no `--target`/`-y`, it now presents the same checkbox multi-select (pre-checked from `DetectAll`) that install uses.
- Replaced `installStdinIsInteractive`/`promptAgentMultiSelect` with `interactiveAllowed`/`runAgentPicker` — cli-package func vars defaulting to `tui.InteractiveAllowed`/`tui.RunAgentPicker` — so tests can force the interactive branch and stub the picker without a real pty or a real `tea.Program`.
- `selectByIndices`/`printAgentResults`/`installStatus`/`uninstallStatus` and `--target`/`--location`/`--auto-allow` are byte-for-byte unchanged; every pre-existing install/uninstall test passes unmodified.

## Task Commits

1. **Task 1: bubbles checkbox multi-select Model** — TDD (RED then GREEN):
   - `9899955` test(07-06): add failing agent checkbox picker Model tests (RED)
   - `06fece7` feat(07-06): implement agent checkbox picker Model (GREEN)
   - `5902cfc` refactor(07-06): drop redundant DisableQuitKeybindings in agentPickerModel (cleanup, same task — the outer Update's own enter/quit interception already made it dead code)
2. **Task 2: Wire -y/--yes + the picker into install.go and uninstall.go** — `f893992` feat(07-06): wire -y/--yes and the bubbles picker into install/uninstall

**Plan metadata:** this commit (docs: complete 07-06 plan)

## Files Created/Modified
- `internal/cli/tui/agentpicker.go` - `agentItem`, `checkboxDelegate`, `agentPickerModel`, `newAgentPickerModel`, `selectByIndices`, `RunAgentPicker`
- `internal/cli/tui/agentpicker_test.go` - Model unit tests driven by synthetic `tea.KeyPressMsg` (no real pty)
- `internal/cli/install.go` - `-y`/`--yes` flag, yes-first switch, `interactiveAllowed`/`runAgentPicker` seams replacing `installStdinIsInteractive`/`promptAgentMultiSelect`
- `internal/cli/uninstall.go` - `-y`/`--yes` flag, same picker/auto(all) switch install now has
- `internal/cli/install_test.go` - regression tests for the -y short-circuit and the interactive-branch wiring, for both install and uninstall
- `go.mod`/`go.sum` - `go mod tidy -e` pulled in `bubbles/v2/list`'s transitive `github.com/sahilm/fuzzy` (filter) and `github.com/atotto/clipboard` (textinput) — additive only, no direct new require

## Decisions Made
- `selectByIndices` is duplicated in `internal/cli/tui` rather than imported from `internal/cli`, since `internal/cli` imports `internal/cli/tui` (the reverse direction would be a cycle). Kept byte-identical dedup + ascending-index-order semantics to the legacy `install.go` version it replaces.
- `list.Model`'s own `Quit`/`ForceQuit` keybindings never actually fire in practice: `agentPickerModel.Update`'s own switch on `enter`/`q`/`esc`/`ctrl+c` returns immediately, before ever calling `list.Model.Update` for those keys — an initial `DisableQuitKeybindings()` call was added defensively then removed as dead weight once this was confirmed (see the Task 1 refactor commit).
- `uninstall`'s non-interactive default stays `"all"` (its pre-existing no-flag behavior) rather than switching to `"auto"` like install — only the *interactive* path is new for uninstall; the off-TTY/`-y` fallback is unchanged from before this plan.

## Deviations from Plan

None — plan executed exactly as written. The one self-initiated cleanup (dropping `DisableQuitKeybindings()`) was a same-task refactor, not a correction of a bug or gap; it left behavior identical and is documented above.

## Issues Encountered

None. `go mod tidy -e`'s pre-existing tree-sitter-swift resolution error (documented in STATE.md from Phase 6) reproduced as expected and was ignored via the `-e` flag; `go build ./...` succeeded afterward with no other issues.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `internal/cli/tui` now has a real bubbletea Model (`agentPickerModel`) alongside `tty.go`'s TTY-gate — any subsequent daemon-picker plan in this phase can follow the same producers-plain/tui-renders + injectable-seam pattern established here.
- `internal/cli/tui/doc.go`'s anchor blank-imports for `charm.land/bubbles/v2`/`charm.land/bubbletea/v2` were deliberately left untouched (out of this plan's file scope) even though `agentpicker.go` now imports both packages directly — a future daemon-picker plan should remove the anchor once its own Model lands, per that file's own comment.
- `internal/cli/tui/agentpicker.go`'s Model rendering (glyphs, help line, cursor) has not been visually verified in a real terminal (see Known Stubs/coverage D4) — a manual spot-check on a real TTY is a reasonable follow-up before shipping, though the state-transition logic itself is fully unit-tested.

---
*Phase: 07-interactive-tui-daemon-picker-install-multi-select*
*Completed: 2026-07-18*

## Self-Check: PASSED

All claimed files exist on disk and all claimed commit hashes (9899955, 06fece7, f893992, 5902cfc) are present in git history.
