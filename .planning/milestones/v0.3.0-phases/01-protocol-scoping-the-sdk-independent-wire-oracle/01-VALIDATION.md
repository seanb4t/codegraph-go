---
phase: 1
slug: protocol-scoping-the-sdk-independent-wire-oracle
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-04
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` (`go test`) — no third-party test framework |
| **Config file** | `Taskfile.yml` (`test:unit`, `test:integration`, `test:golden`, `test:daemon`, `test:race`) |
| **Quick run command** | `go test ./internal/mcp/... ./test/wireoracle/...` |
| **Full suite command** | `task test:unit && task test:integration && task test:golden` |
| **Estimated runtime** | ~10s quick (measured 2026-08-04: `go test ./internal/mcp/...` = 10.4s wall cold, 3.3s package time); ~90s full suite (v1.0 baseline for `task test`) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/mcp/... ./test/wireoracle/...`
- **After every plan wave:** Run `task test:unit && task test:integration`
- **Before `/gsd-verify-work`:** Full suite must be green, PLUS VRFY-04's explicit condition — the oracle is run and shown green against the pre-migration `mark3labs`-backed binary as this phase's own acceptance gate, not deferred to Phase 2
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01 | 1 | VRFY-01 | — | N/A | integration | `go test ./test/wireoracle/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `test/wireoracle/` — the standalone capture tool package (D-01); does not exist yet
- [ ] `testdata/wireoracle/fixture/` — dedicated, purpose-built source tree (D-08). MUST NOT reuse `internal/indexer/testdata/gofixture`, which is mutable-by-convention across other tests
- [ ] VRFY-02 archtest package — `go/packages`-based guard forbidding any SDK-owned protocol-version constant reference tree-wide (including `_test.go`), built on the `internal/graphstore/archtest` + `internal/cli/present/archtest` precedents
- [ ] SDK-02 archtest — asserts `internal/cli/serve.go`'s import closure excludes `mark3labs/mcp-go`; new file or extension of the existing `internal/cli` import-graph test

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Which protocol revision each of the 8 roster agent clients negotiates | VRFY-05 | A one-time empirical measurement against real third-party clients; cannot be asserted by `go test` because the clients are external programs, several of which are not installed on the audit machine | Point each installed client at a local `serve --mcp` through the capture shim, record the negotiated revision with its measurement date in `docs/MCP-8-AGENT-AUDIT.md`. Measurable on this machine (verified 2026-08-04): Claude Code 2.1.222, Codex CLI 0.146.0, opencode 1.18.10. Not installed: Cursor, Gemini CLI, Hermes, Kiro → record as structurally-distinct `UNMEASURED` rows with an explicit blocking reason (D-10), never omitted or filled from docs. Antigravity: present in `/Applications` but not confirmed scriptable — spike its shared Gemini-CLI-style MCP config before deciding MEASURED vs UNMEASURED |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
