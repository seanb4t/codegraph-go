---
status: complete
phase: 04-cli-glow-up
source: [04-VERIFICATION.md]
started: 2026-09-18T01:22:11Z
updated: 2026-09-18T01:50:38Z
---

## Current Test

number: 4
name: all items validated
expected: |
  n/a — all four items recorded below
awaiting: none

## Tests

### 1. CLI-04 palette readability on a light theme and a dark theme
expected: Run `codegraph status`, `codegraph explore <term>`, `codegraph node <symbol>`, `codegraph search <term> --full`, `codegraph install --target claude --location local` (against a scratch HOME) and `codegraph --help` on (a) Solarized Light or macOS light Terminal and (b) a dark theme. Header, label, value, path, count, warning and error hues are visibly distinct and legible in both. (Note: the target id is `claude`, not `claude-code`.)
result: pass — validated by the agent in a Herdr PTY (w1K:p2, xterm-256color/truecolor) at the maintainer's direction. (a) WCAG contrast of every role's hex vs Solarized Light #fdf6e3 / white and Solarized Dark #002b36 / #1e1e1e: all ≥ 4.5 (AA) except light Warning on Solarized Light = 4.37, which renders bold (large-text threshold 3.0 → pass); closest same-mode pairs differ by hue, not luminance. (b) Live switch proven: with the pane background dark, `status` emitted #5fd7ff/#93a1a1/#ffd75f (dark Header/Label/Count); after OSC 11 set the background to #fdf6e3 the SAME command emitted #005f87/#586e75/#875f00 (light Header/Label/Count) and `explore Alpha` used six distinct light roles (#586e75 #073642 #5f5faf #875f00 #005f87 #af5f00); background reset via OSC 111 afterwards. Residual: no human eye applied; the `install --target claude --location local` sub-check rests on 04-07's real-binary ESC/stripped==plain evidence and TestInstall_StyledOutputStripsToPlain rather than a pane view.

### 2. CLI-02 rendering under TERM=dumb / 16-colour / 256-colour / truecolor, incl. SSH and tmux
expected: Run a styled verb (e.g. `codegraph status --color=auto`) under `TERM=dumb`, `TERM=xterm`, `TERM=xterm-256color` and `COLORTERM=truecolor`; ideally one run over SSH and one inside tmux. TERM=dumb is plain with no garbled escapes; 16/256/truecolor degrade sensibly. A ~2 s pause before styled output over SSH/tmux is the OSC-11 background query timing out (D-11 correction), not a hang — note the observed wall time.
result: pass (four profiles) — stdout on the real PTY: TERM=dumb → 0 ESC bytes, plain; TERM=xterm (no COLORTERM) → 38;5;{6,9,12} (basic-palette indices ≤ 15, colorprofile's ANSI encoding); TERM=xterm-256color → 38;5;{81,109,221}; COLORTERM=truecolor → 38;2;r;g;b. Wall time 0.05–0.16 s per run — Herdr's terminal answers OSC 11, so no pause. NOT RUN: tmux (not installed locally; CI installs it for the e2e job) and SSH — the D-11 ~2 s OSC-11 timeout on a non-answering terminal was therefore not observed; expected symptom stays documented, not verified.

### 3. `--help` and `<verb> --help` styling consistency
expected: On a real TTY, `codegraph --help`, `codegraph status --help` and `codegraph explore --help` render with the same palette and the four group sections (Query the graph / Build the index / Agents & serving / Maintenance). Piped, help is cobra's stock template with no ANSI.
result: pass — on the TTY, `codegraph --help`, `status --help` and `explore --help` all use Header #5fd7ff bold for Usage:/Flags:/Global Flags:/group titles, Value #eee8d5 for names, Label #93a1a1 dim for descriptions; the four D-13 groups appear in order. Piped: ESC-on-pipe root=0 status=0 explore=0 and the stock cobra `Query the graph:` line is present as plain text.

### 4. `--color=always | less -R` shows colour; `--color=never` on a TTY is plain
expected: `codegraph status --color=always | less -R` displays ANSI colour; `codegraph status --color=never` run directly in a terminal is plain.
result: pass — `status --color=never` on the TTY: 0 ESC bytes; `status --color=auto` on the same TTY: styled (3 hues), and its stripped bytes are identical to the --color=never output; `status --color=always | less -R` rendered colour inside the pager (counts #ffd75f, headings bold #5fd7ff, labels dim) with less in the foreground.

## Summary

total: 4
passed: 4
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

None. Residuals recorded inside results 1 and 2 (no human eye on colour; tmux/SSH sub-checks not run).
