---
status: testing
phase: 07-interactive-tui-daemon-picker-install-multi-select
source: [07-VERIFICATION.md]
started: 2026-07-18T00:00:00Z
updated: 2026-07-18T00:00:00Z
---

## Current Test

number: 1
name: Bubbletea daemon picker visual rendering on a real TTY
expected: |
  `codegraph daemon` (with ≥1 running daemon) opens a legible single-select
  list — current repo's daemon(s) shown first — and stop-one / stop-all /
  cancel all work and render correctly (colors, layout, selection highlight).
awaiting: user response

## Tests

### 1. Bubbletea daemon picker visual rendering on a real TTY
expected: `codegraph daemon` with ≥1 running daemon opens a legible single-select list, current repo's daemon(s) first; stop-one/stop-all/cancel work and render correctly. (Off-TTY / piped is already automated: plain list, exit 0, never hangs.)
result: issue
reported: "escape issues in the UI — `codegraph daemon` with no running daemons on a real TTY printed `no running daemons` followed by leaked terminal control sequences `^[[?2026;2$y^[[?2027;0$y`"
severity: minor
root_cause: "daemon.go bare RunE opened a bubbletea Program (RunDaemonPicker) whenever stdout was a TTY, even with an empty registry. daemonPickerModel.Init() returns tea.Quit immediately for the empty set, so the Program started — emitting DECRQM capability probes (\\e[?2026$p synchronized-output, \\e[?2027$p grapheme) — then quit before its event loop consumed the terminal's replies, leaking `;2$y`/`;0$y` to stdout. Automated tests run piped (non-TTY) so the terminal never replied and the leak was invisible."
fixed_by: inline (this session)

### 2. Install/uninstall checkbox multi-select visual rendering on a real TTY
expected: `codegraph install` (no `--target`, no `-y`) shows a checkbox list pre-checked for already-installed agents; space toggles, enter confirms, q/esc cancels — legible glyphs, correct pre-check state. Same for `codegraph uninstall`. (Off-TTY / `-y` auto path is already automated.)
result: [pending]

### 3. Windows daemon stop termination + PPID watchdog on a real Windows host
expected: On Windows, `daemon start` then `daemon stop` from another shell terminates the daemon (hard-kill via TerminateProcess) and clears its registry record; the PPID watchdog shuts a daemon/watcher down when its supervising process dies. (No Windows CI runner exists; cross-compile + `GOOS=windows go vet` via mingw-w64 pass, but real termination/liveness semantics need a Windows host.)
result: [pending]

## Summary

total: 3
passed: 0
issues: 1
pending: 2
skipped: 0
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
