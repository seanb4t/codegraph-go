---
phase: 6
slug: rendering-seam-pretty-status-files
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-17
validated: 2026-07-17
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
| 6-01 | 01 | 1 | TUI-01 | T-06-01/02/03 | no `charm.land/*/v2` import reachable from serve-side packages; self-defeat guard | archtest | `go test ./internal/cli/present/archtest/... -run TestNoCharmInServeReachablePackages` | ✅ | ✅ green |
| 6-02 | 02 | 2 | TUI-02 | T-06-05/06 | `status`/`files` byte-identical when non-TTY/piped; styled on TTY; no ANSI leak | golden + integration + unit | `go test ./testdata/golden/... ./test/integration/ -run StatusFilesPlainByteIdentity ./internal/cli/present/ -run 'RenderStatus\|RenderFiles\|ChoosePresentation\|SanitizeControl'` | ✅ | ✅ green |
| 6-03 | 03 | 2 | TUI-05 | T-06-07/08/09 | init/index/sync progress on stderr, TTY-gated, absent when piped; no goroutine leak | unit | `go test ./internal/cli/present/ -run Progress ./internal/cli/ -run Progress` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Mapped tests actually run 2026-07-17 (ran, not inferred — count > 0 per package):**
- TUI-01 (1 test): `TestNoCharmInServeReachablePackages` — PASS
- TUI-02 (12): `TestRenderFiles_{Tree,Flat,Empty}`, `TestRenderStatus_{ContainsANSI,SectionOrder,NumericFormatting,WorktreeWarning}`, `TestChoosePresentation`, `TestSanitizeControl(+_CleanStringIdentity)`, `TestStatusFilesPlainByteIdentity`, golden corpus — PASS
- TUI-05 (8): `TestProgress_{FramesContainLabelAndANSI,StderrOnly,StopClearsLine,StopIdempotent,NoGoroutineLeak}`, `TestProgressCLINonTTYReachability`, `TestProgressCLIQuietSuppressesSummary`, `TestProgressWiringDoesNotBreakErrorPaths` — PASS

---

## Wave 0 Requirements

- [x] TUI-01 archtest (`internal/cli/present/archtest/import_graph_test.go`) — forbidden-set closure walk; forbids `charm.land/lipgloss/v2`, `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`; self-defeat guard (`assertCharmImporterExists`) + guarded-package count guard. Landed RED-first, now green.
- [x] TUI-02 byte-identity — `TestStatusFilesPlainByteIdentity` (subprocess non-TTY buffer) + golden corpus assert piped/non-TTY output unchanged.
- [x] TUI-05 progress tests — `TestProgress_StderrOnly` + `TestProgressCLINonTTYReachability` assert stderr-only, no ANSI when non-TTY; `TestProgress_NoGoroutineLeak` (`-race` clean).

*Existing infrastructure (`go test`, `golang.org/x/tools/go/packages`) covered the archtest and unit dimensions; no framework install was needed. Wave 0 complete.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Colorized output actually renders on a real terminal | TUI-02 | Visual ANSI perception on a real TTY isn't asserted by golden bytes | Run `codegraph status` and `codegraph files` in an interactive terminal; confirm sectioned color output |
| Spinner animation is visible during a real index | TUI-05 | Frame animation timing on a real TTY | Run `codegraph index` in a terminal; confirm spinner/progress on stderr, gone when `2>/dev/null`/piped |

---

## Validation Audit 2026-07-17

| Metric | Count |
|--------|-------|
| Requirements (TUI-01/02/05) | 3 |
| COVERED | 3 |
| PARTIAL | 0 |
| MISSING | 0 |
| Gaps found | 0 |
| Mapped tests run (all green) | 21 |

State A audit: `06-VALIDATION.md` existed as `draft` (seeded at plan time). No gaps — every requirement maps to ≥1 automated test that was **run** this audit (not inferred; each package reported >0 tests). No auditor spawn required. `nyquist_compliant: true`. The 2 manual-only items (real-terminal color rendering, live spinner animation) are UX-observation only and do not affect automated compliance.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none — 0 gaps)
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-07-17
