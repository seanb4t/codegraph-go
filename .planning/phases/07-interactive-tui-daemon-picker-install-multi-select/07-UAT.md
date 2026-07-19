---
status: complete
phase: 07-interactive-tui-daemon-picker-install-multi-select
source: [07-VERIFICATION.md]
started: 2026-07-18T00:00:00Z
updated: 2026-07-19T00:00:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Bubbletea daemon picker visual rendering on a real TTY
expected: `codegraph daemon` with ≥1 running daemon opens a legible single-select list, current repo's daemon(s) first; stop-one/stop-all/cancel work and render correctly. (Off-TTY / piped is already automated: plain list, exit 0, never hangs.)
result: pass
note: "Passed after two rendering bugs found during this UAT were fixed and re-verified: G-07-1 (empty-registry escape-sequence leak — fixed 3e43b25) and G-07-2 (picker flicker + blank list, missing bubbletea-v2 alt-screen — fixed ad6e9cb). Final re-test: alt-screen picker renders 'Running daemons' + the current-repo record, no flicker; enter stopped pid 74320 with a 'stopped pid …' confirmation line (WR-01) and the alt-screen restored cleanly."

### 2. Install/uninstall checkbox multi-select visual rendering on a real TTY
expected: `codegraph install` (no `--target`, no `-y`) shows a checkbox list pre-checked for already-installed agents; space toggles, enter confirms, q/esc cancels — legible glyphs, correct pre-check state. Same for `codegraph uninstall`. (Off-TTY / `-y` auto path is already automated.)
result: pass
note: "Alt-screen checkbox picker (post G-07-2 fix) renders 'Select agents to configure' + all 8 agents pre-checked [x] (detected installed), '>' cursor, no flicker."

### 3. Windows daemon stop termination + PPID watchdog on a real Windows host
expected: On Windows, `daemon start` then `daemon stop` from another shell terminates the daemon (hard-kill via TerminateProcess) and clears its registry record; the PPID watchdog shuts a daemon/watcher down when its supervising process dies. (No Windows CI runner exists; cross-compile + `GOOS=windows go vet` via mingw-w64 pass, but real termination/liveness semantics need a Windows host.)
result: skipped
reason: "Windows platform gap — tester is on macOS, no Windows host available. Windows-tagged code is cross-compiled + go-vet-typechecked via mingw-w64 in CI; only real runtime termination/liveness semantics remain unverified. Accepted platform gap, tracked for a future Windows verification pass."

## Summary

total: 3
passed: 2
issues: 0
pending: 0
skipped: 1
blocked: 0

## Gaps

- gap_id: G-07-1
  truth: "`codegraph daemon` on a real TTY with no running daemons prints only `no running daemons` — no leaked terminal control sequences"
  status: resolved
  reason: "User reported leaked DECRQM responses (^[[?2026;2$y^[[?2027;0$y) after 'no running daemons' on a TTY"
  severity: minor
  test: 1
  root_cause: "bare `daemon` opened a bubbletea Program for the empty registry even on a TTY; the immediate tea.Quit leaked the terminal's capability-probe responses"
  fix: "gate the picker on len(records) > 0 in daemon.go RunE (empty falls through to the plain 'no running daemons' notice) + defense-in-depth empty-guard in tui.RunDaemonPicker before tea.NewProgram + regression test"
  resolved_at: 2026-07-19

- gap_id: G-07-2
  truth: "the daemon picker (and install/uninstall checkbox picker) render cleanly on a real TTY/tmux — no flicker, title + rows visible"
  status: resolved
  reason: "User re-test (partial pass): escape leak gone, but the picker flickered heavily and rendered blank (only the footer visible, no title/rows)"
  severity: minor
  test: 1
  root_cause: "both RunDaemonPicker and RunAgentPicker ran the bubbletea Program WITHOUT alt-screen, so it rendered inline below the prompt; a full-height list that didn't fit the space scrolled the main buffer every frame (flicker) and pushed the title/rows out of view. Compounded by View() appending a 2-line help footer while Update sized the list to the FULL window height (2-line overflow)."
  fix: "bubbletea v2 makes alt-screen a per-View field (no WithAltScreen option) — set View.AltScreen = true in both pickers' View(); reserve helpFooterLines (2) when sizing the list in WindowSizeMsg so list + footer fit exactly"
  resolved_at: 2026-07-19
