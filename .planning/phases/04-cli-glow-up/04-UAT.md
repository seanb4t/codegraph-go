---
status: testing
phase: 04-cli-glow-up
source: [04-VERIFICATION.md]
started: 2026-09-18T01:22:11Z
updated: 2026-09-18T01:22:11Z
---

## Current Test

number: 1
name: CLI-04 palette readability on a light theme and a dark theme
expected: |
  Header, label, value, path, count, warning and error hues are visibly distinct and legible on
  (a) Solarized Light or macOS light Terminal and (b) a dark theme — no low-contrast pair.
awaiting: user response

## Tests

### 1. CLI-04 palette readability on a light theme and a dark theme
expected: Run `codegraph status`, `codegraph explore <term>`, `codegraph node <symbol>`, `codegraph search <term> --full`, `codegraph install --target claude --location local` (against a scratch HOME) and `codegraph --help` on (a) Solarized Light or macOS light Terminal and (b) a dark theme. Header, label, value, path, count, warning and error hues are visibly distinct and legible in both. (Note: the target id is `claude`, not `claude-code`.)
result: [pending]

### 2. CLI-02 rendering under TERM=dumb / 16-colour / 256-colour / truecolor, incl. SSH and tmux
expected: Run a styled verb (e.g. `codegraph status --color=auto`) under `TERM=dumb`, `TERM=xterm`, `TERM=xterm-256color` and `COLORTERM=truecolor`; ideally one run over SSH and one inside tmux. TERM=dumb is plain with no garbled escapes; 16/256/truecolor degrade sensibly. A ~2 s pause before styled output over SSH/tmux is the OSC-11 background query timing out (D-11 correction), not a hang — note the observed wall time.
result: [pending]

### 3. `--help` and `<verb> --help` styling consistency
expected: On a real TTY, `codegraph --help`, `codegraph status --help` and `codegraph explore --help` render with the same palette and the four group sections (Query the graph / Build the index / Agents & serving / Maintenance). Piped, help is cobra's stock template with no ANSI.
result: [pending]

### 4. `--color=always | less -R` shows colour; `--color=never` on a TTY is plain
expected: `codegraph status --color=always | less -R` displays ANSI colour; `codegraph status --color=never` run directly in a terminal is plain.
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
