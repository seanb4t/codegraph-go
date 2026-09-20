---
created: 2026-09-18T00:00:00.000Z
title: install --yes silently discards an explicit --target and resolves to auto
area: cli
severity: minor
files:

  - internal/cli/install.go:99-109

completed: 2026-09-19
status: completed
---

## Problem

`install`'s `RunE` switch checks `case yes:` before `case cmd.Flags().Changed("target"):`
(`internal/cli/install.go:99-109`), so `codegraph install --target claude,cursor --yes`
ignores the explicit target list and configures the auto-detected set instead. Reproduced
live on 2026-09-18 during v0.14.0 Phase 5 plan 05-06 (a scratch dir configured only
`opencode`, the machine's auto default). Introduced in f8939922 (2026-07-18, v1.0 Phase 7).

The comment on `case yes:` states the intent: `--yes` must short-circuit before the TTY
picker branch. It was not meant to override an explicit `--target`.

## Solution

Check `cmd.Flags().Changed("target")` before `yes`: an explicit target wins, and `--yes`
still skips the picker and falls back to `auto` only when no target was given. Check
`uninstall` for the same ordering. Add a test for `--target X --yes` → exactly X,
proven RED on the current ordering first.
