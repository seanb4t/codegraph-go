---
phase: 2
slug: sdk-migration-official-go-sdk-on-the-existing-surface
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-05
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (Go stdlib testing), orchestrated through `Taskfile.yml` |
| **Config file** | `Taskfile.yml` — the single definition of every CI job body (`TestWorkflowRunBodiesInvokeTask` enforces it) |
| **Quick run command** | `task test -- ./internal/mcp/...` |
| **Full suite command** | `task test` |
| **Wire oracle command** | `go test ./test/wireoracle/...` (23 scenarios, ~17s) |
| **Estimated runtime** | quick ~10s · wire oracle ~17s · full suite several minutes |

**Note:** existing infrastructure is mature. This phase adds no test framework —
it re-points existing assertions at a new backend.

---

## Sampling Rate

- **After every task commit:** `task test -- ./internal/mcp/...`
- **After every plan wave:** `task test` plus `go test ./test/wireoracle/...`
- **Before `/gsd-verify-work`:** full suite green AND the D-01 transcript diff read line by line
- **Max feedback latency:** ~30 seconds for the quick + oracle pair

---

## Per-Task Verification Map

*Populated by `gsd-planner`. Every task must map to an automated command or
declare a Wave 0 dependency.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| *pending* | — | — | SDK-01/03/04/05 | — | — | — | — | — | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements. The wire oracle
(`test/wireoracle/`, 23 frozen transcripts) and the `internal/mcp` unit suite
both predate this phase and are the primary instruments.

One structural note carried from `02-RESEARCH.md`: this phase's own gates live
in `internal/mcp/archtest/` and `internal/cli/archtest/`, which were
deliberately written identity-agnostic so they survive the SDK swap. Verify they
still fire against the go-sdk path rather than assuming they do — a guard that
forbade only mark3labs' spelling would pass vacuously after the swap.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The transcript diff review | SDK-01 | This is the phase's bar by decision D-01. A comparator cannot distinguish a cosmetic key-order change from a semantic protocol change; that judgment is the deliverable. | Run the oracle against the pre-swap binary and the post-swap binary. Read the full diff. Every changed line must have a recorded cause. Expected-and-explained per research: `legacy-omitted-version` moves `2025-03-26`→`2025-11-25`; `ttlMs`/`cacheScope` appear on every `tools/list`; `additionalProperties:false` appears on every tool; `properties` key order becomes Go field order; annotation key order changes. NOT expected: `legacy-unsupported-2026-07-28`'s `protocolVersion` value (stays `2025-11-25`). |
| Live client spot-check | SDK-01 | The 8-agent roster's real clients are outside CI. | Rebuild the binary (`bin/codegraph` is a gitignored build artifact — rebuild before checking) and confirm at least one real MCP client still lists tools. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] Every new or re-pointed gate demonstrated RED against a confirmed-applied mutation (standing project rule)
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
