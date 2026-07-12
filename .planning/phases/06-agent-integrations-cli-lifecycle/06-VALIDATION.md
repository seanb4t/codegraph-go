---
phase: 6
slug: agent-integrations-cli-lifecycle
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-12
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (stdlib testing) |
| **Config file** | none — Go modules, no separate test config |
| **Quick run command** | `go test ./internal/agents/... ./internal/cli/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~30–90 seconds (full suite; agents/cli packages ~5–15s) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./<touched-package>/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green (`go test ./... -race` for the installer round-trip + upgrade swap tests)
- **Max feedback latency:** 90 seconds

---

## Per-Task Verification Map

> Populated by the planner from PLAN.md tasks. Each installer/CLI task maps to a `go test` package + the requirement it satisfies. The load-bearing invariants below are pre-seeded from RESEARCH.md's Validation Architecture.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 06-XX-XX | XX | 1 | AGNT-01/02 | — | install→uninstall round-trip returns config file to pre-install bytes (per-agent, all 8) | unit (golden/table) | `go test ./internal/agents/...` | ❌ W0 | ⬜ pending |
| 06-XX-XX | XX | 1 | AGNT-01 | — | re-running install twice is byte-idempotent | unit | `go test ./internal/agents/...` | ❌ W0 | ⬜ pending |
| 06-XX-XX | XX | 1 | AGNT-03 | — | Cursor entry carries `--path` (local=abs cwd / global=`${workspaceFolder}`) | unit | `go test ./internal/agents/...` | ❌ W0 | ⬜ pending |
| 06-XX-XX | XX | 2 | CLI-02 | T-06-upgrade | upgrade REJECTS an unsigned/tampered binary before swap (fixture) | unit | `go test ./internal/upgrade/...` | ❌ W0 | ⬜ pending |
| 06-XX-XX | XX | 2 | CLI-02 | — | atomic self-replace leaves no partial binary on interrupted swap | unit | `go test ./internal/upgrade/...` | ❌ W0 | ⬜ pending |
| 06-XX-XX | XX | 2 | CLI-01 | — | `version` prints ldflags-injected semver/commit/date; `version --json` parses | unit | `go test ./internal/cli/...` | ❌ W0 | ⬜ pending |
| 06-XX-XX | XX | 2 | CLI-03 | — | `telemetry` output states zero passive phone-home + names upgrade as sole network path | unit | `go test ./internal/cli/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/agents/*_test.go` — per-agent install/uninstall/round-trip table stubs for AGNT-01/02/03 (mirror TS `installer-targets.test.ts`)
- [ ] `internal/agents/testdata/` — pre-install config fixtures per agent + format (JSON/JSONC/TOML) for the round-trip byte-invariant
- [ ] `internal/upgrade/*_test.go` — signature-verification stubs (CLI-02) with a hermetic signed/tampered fixture pair (Open Question 1 — resolve the offline sigstore trust-chain fixture)
- [ ] `internal/cli` — `version`/`telemetry` command test stubs (CLI-01/03)

*Existing infrastructure (`go test`) covers the runner; per-package test files are the Wave 0 gap.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real end-to-end MCP handshake after `install` in a live Claude Code / Cursor session | AGNT-01/03 | Requires the actual agent runtime + a live MCP connection; not reproducible in `go test` | After `codegraph install`, open the agent, confirm `codegraph_explore` tool is listed and returns results |
| `upgrade` against a real signed GitHub Release | CLI-02 | Production cosign identity + release assets do not exist until Phase 8 (DIST-02) | Deferred to Phase 8 integration; Phase 6 verifies the verify-then-swap path against fixtures only |

*Fixture-based automation covers the signature-verification logic; the live-release path is a documented Phase-8 integration follow-up.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 90s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
