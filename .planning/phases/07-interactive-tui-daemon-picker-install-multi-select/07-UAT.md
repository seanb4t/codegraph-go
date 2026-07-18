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
result: [pending]

### 2. Install/uninstall checkbox multi-select visual rendering on a real TTY
expected: `codegraph install` (no `--target`, no `-y`) shows a checkbox list pre-checked for already-installed agents; space toggles, enter confirms, q/esc cancels — legible glyphs, correct pre-check state. Same for `codegraph uninstall`. (Off-TTY / `-y` auto path is already automated.)
result: [pending]

### 3. Windows daemon stop termination + PPID watchdog on a real Windows host
expected: On Windows, `daemon start` then `daemon stop` from another shell terminates the daemon (hard-kill via TerminateProcess) and clears its registry record; the PPID watchdog shuts a daemon/watcher down when its supervising process dies. (No Windows CI runner exists; cross-compile + `GOOS=windows go vet` via mingw-w64 pass, but real termination/liveness semantics need a Windows host.)
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps
