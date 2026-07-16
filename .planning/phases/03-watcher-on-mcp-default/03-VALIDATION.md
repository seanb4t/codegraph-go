---
phase: 3
slug: watcher-on-mcp-default
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-16
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (stdlib) + goleak soaks + testdata/golden parity suite + test/integration subprocess harness (new this phase) |
| **Config file** | none — go.mod toolchain |
| **Quick run command** | `go test ./internal/... ./cmd/...` |
| **Full suite command** | `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...` |
| **Estimated runtime** | ~90 seconds (quick ~30s; golden + integration add ~60s) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... ./cmd/...`
- **After every plan wave:** Run `go test ./... && go test ./testdata/golden/... && go test ./test/integration/...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 120 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (filled by planner — every WATCH-01..04 + TEST-04 task maps here) | — | — | WATCH-01..04, TEST-04 | — | watcher never blocks handshake; env opt-out honored | unit / integration | `go test ./...` + explicit golden/integration steps | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `test/integration/` — TestMain that builds the real binary once (`go build -o <tmp>/codegraph .`) — TEST-04 substrate
- [ ] Real-git worktree fixture helper (reuse Phase-2 D-15 pattern) reachable from `test/integration/`
- [ ] `internal/watch/policy_test.go` — stubs for WATCH-03 precedence table

*Existing infrastructure covers the rest: goleak TestMain in `internal/daemon`, golden harness in `testdata/golden/`.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| WSL2 `/mnt/*` auto-off on a real WSL2 host | WATCH-03 | No WSL2 environment in CI or on darwin dev machine; detection is unit-tested via injected probes | On a WSL2 box: `cd /mnt/c/<repo> && codegraph serve --mcp` → stderr shows the verbatim disabled message; `CODEGRAPH_FORCE_WATCH=1` re-enables |

*All other phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
