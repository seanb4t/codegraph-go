---
phase: 6
slug: rendering-seam-pretty-status-files
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-17
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (stdlib testing) |
| **Config file** | none — Go modules; archtests use `golang.org/x/tools/go/packages` (already a test dep) |
| **Quick run command** | `go test ./internal/cli/... ./internal/query/...` |
| **Full suite command** | `go test ./... && go test ./testdata/golden/...` |
| **Estimated runtime** | ~30–90 seconds |

*Note: `go test ./...` silently skips `testdata/` — the golden-corpus byte-identity assertions (TUI-02 SC4) MUST be run explicitly via `go test ./testdata/golden/...`, per the Phase-3/TEST-04 lesson.*

---

## Sampling Rate

- **After every task commit:** Run the quick run command
- **After every plan wave:** Run the full suite command (including `./testdata/golden/...`)
- **Before `/gsd-verify-work`:** Full suite must be green AND the TUI-01 archtest green
- **Max feedback latency:** ~90 seconds

---

## Per-Task Verification Map

> Seeded by plan-phase; the planner/validate-phase fills exact task IDs and commands. Derived from RESEARCH.md §"Validation Architecture".

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 6-01-xx | 01 | 1 | TUI-01 | T-06-01 / — | no `charm.land/*/v2` import reachable from serve-side packages | archtest | `go test ./internal/cli/present/archtest/...` (fail-closed) | ❌ W0 | ⬜ pending |
| 6-02-xx | 02 | 2 | TUI-02 | — | `status`/`files` byte-identical when non-TTY / piped | golden + unit | `go test ./testdata/golden/... ./internal/cli/present/...` | ❌ W0 | ⬜ pending |
| 6-03-xx | 03 | 2 | TUI-05 | — | init/index/sync progress on stderr, TTY-gated, absent when piped | unit | `go test ./internal/cli/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] TUI-01 archtest (`internal/cli/present/archtest/import_graph_test.go`) — mirrors `internal/graphstore/archtest/import_graph_test.go`; forbid `charm.land/lipgloss/v2`, `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`; self-defeat guard. Lands FIRST, fail-closed until lipgloss is added.
- [ ] TUI-02 byte-identity fixtures — assert piped/non-TTY `status`/`files` output equals current golden bytes.
- [ ] TUI-05 progress tests — assert stderr-only, no ANSI when non-TTY.

*Existing infrastructure (`go test`, `golang.org/x/tools/go/packages`) covers the archtest and unit dimensions; no framework install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Colorized output actually renders on a real terminal | TUI-02 | Visual ANSI perception on a real TTY isn't asserted by golden bytes | Run `codegraph status` and `codegraph files` in an interactive terminal; confirm sectioned color output |
| Spinner animation is visible during a real index | TUI-05 | Frame animation timing on a real TTY | Run `codegraph index` in a terminal; confirm spinner/progress on stderr, gone when `2>/dev/null`/piped |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
