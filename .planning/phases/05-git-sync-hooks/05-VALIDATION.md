---
phase: 5
slug: git-sync-hooks
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-16
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — stdlib testing, per-package TestMain where needed |
| **Quick run command** | `go test ./internal/githooks/ ./internal/fsatomic/ ./internal/gitmeta/ ./internal/cli/` |
| **Full suite command** | `go test ./... && go test ./testdata/golden/... && go vet ./...` |
| **Estimated runtime** | ~60 seconds (quick ~10s) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/githooks/ ./internal/fsatomic/ ./internal/gitmeta/ ./internal/cli/`
- **After every plan wave:** Run `go test ./... && go test ./testdata/golden/... && go vet ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (filled by planner — one row per PLAN.md task) | | | HOOK-01/02/03 | | | unit/integration | `go test ./...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Existing infrastructure covers all phase requirements — `go test` + real-git `t.TempDir()` fixture pattern (Phase 2 D-15 precedent in `internal/gitmeta`) already established; no new framework needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Hook fires on a real `git commit` and backgrounds `codegraph sync` without blocking | HOOK-01 | Backgrounded, silenced subshell is inherently racy to assert in CI (D-14: no execution e2e in v1.0) | In a scratch repo with hooks installed: `git commit --allow-empty -m x` returns immediately; `.codegraph/` mtime advances within a few seconds |

*All other phase behaviors have automated verification (content-level fixtures).*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
