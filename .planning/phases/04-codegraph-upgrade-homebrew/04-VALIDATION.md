---
phase: 4
slug: codegraph-upgrade-homebrew
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-11
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib `testing`) |
| **Config file** | none — `Taskfile.yml` defines every CI job body (`TestWorkflowRunBodiesInvokeTask` enforces it) |
| **Quick run command** | `go test ./internal/upgrade/...` |
| **Full suite command** | `task test` |
| **Estimated runtime** | ~5s package / ~90s full suite |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/upgrade/...`
- **After every plan wave:** Run `task test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

*Populated by the planner from PLAN.md task IDs — do not hand-author task rows here before plans exist.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| {N}-01-01 | 01 | 1 | REQ-{XX} | T-{N}-01 / — | {expected secure behavior or "N/A"} | unit | `{command}` | ✅ / ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/upgrade/brew_test.go` — table-driven detection tests for UPGR-02, including the
      mandatory executing false-positive row (a non-brew binary under a path containing
      `Caskroom`/`Cellar` with no `INSTALL_RECEIPT.json` above it)
- [ ] Constructed-tree fixtures under `t.TempDir()` for `/opt/homebrew`, `/usr/local`, a custom
      prefix, and linuxbrew — **both Caskroom and Cellar shapes at the linuxbrew prefix**
      (D-04R, 2026-08-11)
- [ ] Seam assertion in `internal/upgrade/upgrade_test.go` proving `Options.download` and
      `Options.swap` are never invoked under a detected brew install (D-08)

*Existing `go test` infrastructure covers the framework itself — no framework install needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The refusal is observed against a **real** `brew tap seanb4t/tap` + `brew install codegraph` from the Phase-3 tap | UPGR-01 | The ROADMAP's Depends-on clause states a synthetic layout can only prove the mechanism, not that it matches what actually ships. Requires a real published cask and a real Homebrew install on the host. | `brew trust seanb4t/tap` (per `khb8v67c48` — the published contract does not work without it), `brew install codegraph`, then `codegraph upgrade` and `codegraph upgrade --check`; record both outputs and exit codes verbatim. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
