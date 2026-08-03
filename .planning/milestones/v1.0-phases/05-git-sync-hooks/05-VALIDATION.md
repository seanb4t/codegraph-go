---
phase: 5
slug: git-sync-hooks
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
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
| 05-01-T1 fsatomic.WriteFile | 05-01 | 1 | HOOK-01/02 | T-05-04 | Atomic temp+rename; existing-mode preservation, no partial file | unit | `go test ./internal/fsatomic/... -v` | ✅ | ✅ green |
| 05-01-T2 agents rewire | 05-01 | 1 | HOOK-01/02 | T-05-04 | Byte-invariance regression gate stays green | regression | `go test ./internal/agents/... && go build ./...` | ✅ existing | ✅ green |
| 05-02-T1 gitmeta.IsGitRepo | 05-02 | 1 | HOOK-01/02/03 | T-05-02b | 5s timeout, Stdin nil, degrade-to-false | unit | `go test ./internal/gitmeta/... -run TestIsGitRepo -v` | ✅ | ✅ green |
| 05-02-T2 gitmeta.HooksDir | 05-02 | 1 | HOOK-01/02/03 | T-05-02 | --git-path resolution; no hand-joined .git/hooks; degrade-to-"" | unit | `go test ./internal/gitmeta/... -run 'TestHooksDir' -v` | ✅ | ✅ green |
| 05-03-T1 markers/strip primitives | 05-03 | 2 | HOOK-01/02 | T-05-01 | Verbatim constant block, zero interpolation | unit | `go test ./internal/githooks/... -run 'TestMarkerBlock|TestStripMarkerBlock|TestIsEffectivelyEmpty' -v` | ✅ | ✅ green |
| 05-03-T2 Install strip-then-append | 05-03 | 2 | HOOK-01 | T-05-04, T-05-05 | Atomic write, mode 0755, idempotent, no in-place divergence | unit | `go test ./internal/githooks/... -run 'TestInstall' -v` | ✅ | ✅ green |
| 05-03-T3 Remove/Status + TS-block interop | 05-03 | 2 | HOOK-02 | T-05-05 | Delete-when-empty, user-content preserved, TS-block detect/remove | unit | `go test ./internal/githooks/... -run 'TestRemove|TestStatus' -v` | ✅ | ✅ green |
| 05-04-T1 githooks command tree + root wiring | 05-04 | 3 | HOOK-01/02 | T-05-03 | targetRoot filepath.Abs; fixed hook trio | build | `go build ./... && go vet ./internal/cli/...` | ✅ | ✅ green |
| 05-04-T2 CLI reachability tests | 05-04 | 3 | HOOK-01/02 | T-05-03 | Real cobra tree; non-repo skip exit 0 | CLI reachability | `go test ./internal/cli/... -run 'TestGithooks' -v` | ✅ | ✅ green |
| 05-05-T1 init D-07 advisory | 05-05 | 3 | HOOK-03 | T-05-06 | Injectable Probe gate; never blocks init; success-path only | build | `go build ./... && go vet ./internal/cli/...` | ✅ | ✅ green |
| 05-05-T2 uninit D-06 cleanup | 05-05 | 3 | HOOK-03 | T-05-06 | Best-effort, never fails uninit | build | `go build ./... && go vet ./internal/cli/...` | ✅ | ✅ green |
| 05-05-T3 advisory/cleanup reachability | 05-05 | 3 | HOOK-03 | T-05-06 | Mutation-proof: revert wiring turns red; D-12 stderr untouched | CLI reachability | `go test ./internal/cli/... -run 'TestInitAdvisory|TestUninit_RemovesGitHooks' -v` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] Existing infrastructure covers all phase requirements — `go test` + real-git `t.TempDir()` fixture pattern (Phase 2 D-15 precedent in `internal/gitmeta`) already established; no new framework needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Hook fires on a real `git commit` and backgrounds `codegraph sync` without blocking | HOOK-01 | Backgrounded, silenced subshell is inherently racy to assert in CI (D-14: no execution e2e in v1.0) | In a scratch repo with hooks installed: `git commit --allow-empty -m x` returns immediately; `.codegraph/` mtime advances within a few seconds |

*All other phase behaviors have automated verification (content-level fixtures).*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none — no MISSING gaps)
- [x] No watch-mode flags
- [x] Feedback latency < 90s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-07-17 — all 12 tasks COVERED by green automated tests; 1 manual-only (D-14 backgrounded-hook e2e, out of v1.0 scope).

---

## Validation Audit 2026-07-17

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

Audited State-A seed (`status: draft`) against executed reality. All 12 tasks map to named, existing tests; ran each mapped `-run` pattern with `-count=1` (guarding the no-tests-ran false-green): fsatomic 5, IsGitRepo 2, HooksDir 4, marker/strip 9, Install 11, Remove/Status 16, CLI reachability 8, init-advisory/uninit 5 — all pass. Build, `go vet`, agents byte-invariance regression, and the `testdata/golden` suite (skipped by `go test ./...`, GOLDEN-01) all green. No auditor spawn needed — zero gaps to fill.
