# Phase 5: Git Sync Hooks - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-16
**Phase:** 5-git-sync-hooks
**Mode:** --auto --all --chain (all gray areas auto-selected; recommended option chosen per question, no interactive prompts)
**Areas discussed:** Splice semantics & TS fidelity, Marker block bytes, Command surface, HOOK-03 surfacing point, uninit integration, fsatomic extraction scope, Git-probe placement, Test coverage shape

---

## Splice semantics & TS fidelity

| Option | Description | Selected |
|--------|-------------|----------|
| Port TS git-hooks.js exactly | Strip prior block + re-append at end; delete hook file when effectively empty on remove; `#!/bin/sh` seed; chmod 0755 best-effort; trimmed-line marker matching | ✓ |
| Reuse agents' replaceOrAppendMarkedSection | True in-place replacement, no file deletion, no shebang/chmod handling | |
| Hybrid | In-place replace but TS's delete-when-empty | |

**Auto-selected:** TS-exact port (recommended). TS fidelity is the project's parity bar; requirement's "replace-in-place" read as idempotent-replacement-not-duplication; TEST-03's later byte-invariance is defined against these semantics. Writes routed through atomic write (Go improvement, identical output bytes).

---

## Marker block bytes

| Option | Description | Selected |
|--------|-------------|----------|
| Verbatim TS bytes | Identical markers + all 7 inner lines incl. "remove with `codegraph uninit`" | ✓ |
| Go-adapted text | Mention `codegraph githooks remove` in the block comment | |

**Auto-selected:** Verbatim TS (recommended). Byte-identical markers make TS-installed hooks manageable by the Go binary (drop-in win); uninit cleanup (D-06) keeps the comment truthful.

---

## Command surface

| Option | Description | Selected |
|--------|-------------|----------|
| `githooks install/remove/status [path]`, fixed hook trio | Matches requirement text; targetRoot-consistent; no selection flags in v1.0 | ✓ |
| Add --hooks selection flag | Choose which of the three hooks to install | |
| Fold into init/uninit only (TS-literal) | No dedicated command — TS has none | |

**Auto-selected:** Dedicated `githooks` command tree (locked by HOOK-01/02 requirement text); documented Go-only extension for Phase 8 SURF-05.

---

## HOOK-03 surfacing point

| Option | Description | Selected |
|--------|-------------|----------|
| Non-interactive advisory in `init` success path | Port offerWatchFallback gates as plain text; point at `codegraph githooks install`; prompt UI deferred to Phase 7 | ✓ |
| Auto-install on init when watcher disabled | TS's opts.yes path made default | |
| No init integration this phase | Ship command only; surface nothing | |

**Auto-selected:** Non-interactive advisory (recommended). Preserves TS's narrower trigger (fires only when WatchDisabledReason non-empty); no auto-install without explicit user action; interactive select is Phase-7 bubbletea territory. Shipped Phase-3 D-12 serve message stays byte-untouched (locked parity string); its init-rerun residual recorded for Phase 8.

---

## uninit integration

| Option | Description | Selected |
|--------|-------------|----------|
| TS-parity best-effort cleanup in uninit | Strip marker blocks after removing .codegraph/, non-fatal | ✓ |
| Leave uninit untouched | Hooks removed only via `githooks remove` | |

**Auto-selected:** TS-parity cleanup (recommended) — matches bin/codegraph.js ~629-636 and keeps the marker block's own advice truthful.

---

## fsatomic extraction scope

| Option | Description | Selected |
|--------|-------------|----------|
| Extract atomicWriteFile only | Splice logic stays per-package (semantics genuinely differ) | ✓ |
| Extract atomic write + generic marker splice | Shared abstraction per ROADMAP note's literal reading | |

**Auto-selected:** Atomic-write-only (recommended). Agents' in-place `<!-- -->` splice vs hooks' strip/re-append/delete-when-empty/shebang/chmod `#` splice would distort under one abstraction. ROADMAP note deliberately narrowed; documented in the package comment.

---

## Git-probe placement

| Option | Description | Selected |
|--------|-------------|----------|
| Add IsGitRepo + HooksDir to internal/gitmeta | Phase 2 D-04 designed gitmeta for this reuse; one git-exec seam | ✓ |
| Put probes in internal/githooks | Self-contained package, duplicated exec contract | |

**Auto-selected:** gitmeta (recommended) — follows the established 5s-timeout CommandContext contract verbatim.

---

## Test coverage shape

| Option | Description | Selected |
|--------|-------------|----------|
| Real-git t.TempDir fixtures + mutation-proof CLI wiring; no execution e2e | Content-level assertions incl. TS-block compatibility fixture; TEST-03 formal harness stays Phase 7 | ✓ |
| Include hook-execution e2e (git commit fires sync) | Flaky by construction (backgrounded, silenced) | |

**Auto-selected:** Content-level + reachability (recommended). Optional `testing.Short()`-gated execution smoke left to planner discretion.

---

## Claude's Discretion

File layout inside `internal/githooks`; result struct shape; exact `githooks status` output lines; D-07 step-5 pointer wording; whether `sh -n` validation runs in tests.

## Deferred Ideas

- Interactive hook-offer select in `init` (bubbletea) — Phase 7
- TEST-03 formal byte-invariance + piped-stream harness — Phase 7
- D-12 serve-message wording residual — Phase 8 SURF-05 divergence table
- `affected` git-hook scripting flags (SURF-04) — Phase 8
- Release runbook todo — reviewed, not folded (5th consecutive review); belongs with Phase 8
