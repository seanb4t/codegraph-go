# Phase 7: Interactive TUI — Daemon Picker & Install Multi-Select - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-18
**Phase:** 7-Interactive TUI — Daemon Picker & Install Multi-Select
**Mode:** --auto --chain (Claude selected the recommended option for every gray area; no interactive prompts)
**Areas discussed:** Daemon command shape & lifecycle, Global daemon registry, PPID watchdog, Charm isolation & interactive seam, Non-interactive fallbacks, install/uninstall multi-select, TEST-03 harness

---

## Daemon command shape & lifecycle (DMON-01, DMON-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Cobra sub-tree: bare=picker, `start`/`stop`/`stop --all` | Current foreground RunE moves to `daemon start`; bare `daemon` = picker. Resolves TS name collision. | ✓ |
| Keep bare `daemon` foreground, add sibling `daemon-picker` | Avoids restructuring but keeps the TS collision unresolved. | |

| Option | Description | Selected |
|--------|-------------|----------|
| Foreground blocking `start` (reuse existing lockfile Run) | No detached fork; explicit lifecycle only. Detached daemon = DMON-FUT-01. | ✓ |
| Detached double-fork background daemon | True daemonization; larger design, explicitly deferred to DMON-FUT-01. | |

**Choice:** Sub-tree with foreground `start`; **no detached fork** (DMON-FUT-01 deferred). No silent auto-spawn (`serve --mcp` in-process watcher stays the zero-config path).
**Notes:** Locked by PROJECT.md/REQUIREMENTS Out-of-Scope ("no daemon auto-spawn") and the DMON-FUT-01 deferral.

---

## Global daemon registry (DMON-04)

| Option | Description | Selected |
|--------|-------------|----------|
| `~/.codegraph/daemons/` dir, one atomic file per daemon | No global write-lock; mirrors per-project lockfile model; atomic via fsatomic. | ✓ |
| Single shared `~/.codegraph/daemons.json` | One file, but needs a global write-lock for concurrent registrations. | |

| Option | Description | Selected |
|--------|-------------|----------|
| Self-heal on scan (reuse lock.go isProcessLive/isStale) | Prune dead/stale records during any read; no background reaper. | ✓ |
| Dedicated background reaper goroutine | Continuous cleanup; extra moving part, unneeded. | |

**Choice:** Directory of atomic per-daemon record files; liveness-prune on every scan reusing the existing lock staleness predicate.
**Notes:** Record fields ≥ {pid, startedAt, repoRoot}; register on Run start, best-effort remove on shutdown (like `release()`).

---

## PPID watchdog (DMON-03)

| Option | Description | Selected |
|--------|-------------|----------|
| Poll parent liveness ~1–2s; reparent=death; build-split files | POSIX ppid-reparent + Windows liveness poll; mirrors procstart_*.go split. | ✓ |
| Signal/pipe-based parent-death notification | More precise but non-portable / more plumbing. | |

**Choice:** Polling watchdog goroutine, `watchdog_posix.go` / `watchdog_windows.go`, cancels the daemon/watcher ctx on parent death; wired into both `daemon start`/Run AND `serve --mcp`'s in-process watcher; lives in charm-free `internal/daemon`.
**Notes:** Reuses the ctx-cancel teardown that already releases the lock + prunes the registry record.

---

## Charm isolation & interactive seam (TUI-01, TUI-03, TUI-04)

| Option | Description | Selected |
|--------|-------------|----------|
| bubbletea/bubbles ONLY in `internal/cli`; data layers charm-free | Picker/multi-select views in cli consume plain structs from daemon-registry + agents. Archtest already forbids charm from the guarded set incl. `internal/daemon`. | ✓ |
| Put picker UI in `internal/daemon` | Would violate the TUI-01 archtest (daemon is a guarded package) — build fails. | |

| Option | Description | Selected |
|--------|-------------|----------|
| TTY-gate via `ChoosePresentation`/`term.IsTerminal` before `tea.NewProgram()` | Same seam as Phase 6; non-TTY → fallback, never blocks stdin. | ✓ |
| Rely on bubbletea's own non-TTY handling | Not guaranteed byte-safe/non-hanging; library heuristic. | |

**Choice:** Confine bubbletea + bubbles to the cli presentation layer; TTY-gate every interactive entry before `tea.NewProgram()`; add `charm.land/bubbletea/v2` + `charm.land/bubbles/v2` (pinned).
**Notes:** The Phase-6 archtest keeps this honest by construction (green as long as charm stays in cli).

---

## Non-interactive fallbacks (TUI-04)

| Option | Description | Selected |
|--------|-------------|----------|
| daemon no-TTY: print plain running-daemon list, exit 0 | Read-only, script-safe; never opens a picker. | ✓ |
| daemon no-TTY: print usage/error, exit non-zero | Less useful; surprises scripts. | |

| Option | Description | Selected |
|--------|-------------|----------|
| install no-TTY: resolve to `auto` (existing behavior) | Never prompts; `-y`/`--yes` forces it even on TTY. | ✓ |

**Choice:** daemon no-args off-TTY = plain list; install/uninstall off-TTY = `auto`.
**Notes:** install already degrades EOF/non-TTY to auto today — extend the same never-block guarantee to the daemon picker.

---

## install / uninstall multi-select (TUI-03)

| Option | Description | Selected |
|--------|-------------|----------|
| bubbles checkbox multi-select on TTY (pre-check detected) | Replaces the plain numbered-line prompt; auto fallback off-TTY. | ✓ |
| Keep the plain numbered-line prompt | Works but is not the "bubbles multi-select" the requirement names. | |

| Option | Description | Selected |
|--------|-------------|----------|
| Add `-y`/`--yes` to install AND uninstall | Skip picker, use auto set — for scripts/CI; matches TS. | ✓ |

**Choice:** bubbles multi-select on TTY for both install and uninstall; add `-y`/`--yes`. Keep `--target`/`--location`/`--auto-allow`.
**Notes:** UI in cli layer keeps `internal/agents` charm-free.

---

## TEST-03 (byte-invariance + piped never-hang)

| Option | Description | Selected |
|--------|-------------|----------|
| githooks unit byte-invariance + integration piped never-hang | install→edit→remove == original (unit); daemon/install piped-stdin under timeout (integration harness). | ✓ |
| One combined new harness | Reinvents what `test/integration/` already provides. | |

**Choice:** Focused `internal/githooks` byte-invariance test + piped never-hang assertions in the existing `test/integration/` subprocess harness (TEST-04's home).
**Notes:** Adding bubbletea must not regress the piped-stream exit behavior.

---

## Claude's Discretion

- bubbles list styling / key bindings / checkbox glyphs; picker column layout.
- Watchdog poll interval within ~1–2s; exact reparent predicate (ppid==1 vs ≠original).
- Registry record filename scheme + fields beyond {pid, startedAt, repoRoot}.
- Whether to expose an explicit `daemon list` alias.
- Stop signal policy — recommended graceful SIGTERM only (SIGKILL escalation only if needed).

## Deferred Ideas

- DMON-FUT-01 — true detached / double-forked per-project daemons + unix-socket sharing (later milestone).
- Explicit `daemon list` subcommand alias (if not adopted under D-12).
- SIGKILL grace-timeout escalation for `stop`.
- Charm dependency-closure audit (REL-01) → Phase 8.
- `--color`/`--no-color` explicit flag (carried from Phase 6).

### Reviewed Todos (not folded)

- **"Document release procedures (maintainer runbook)"** (score 0.4) — reviewed, NOT folded. Release-process runbook belonging to Phase 8 (REL-02); matched only on generic keywords. Same disposition as Phase 6.
