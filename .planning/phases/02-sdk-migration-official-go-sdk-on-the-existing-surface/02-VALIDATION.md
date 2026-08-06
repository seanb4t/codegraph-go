---
phase: 2
slug: sdk-migration-official-go-sdk-on-the-existing-surface
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-08-05
plans_mapped: 2026-08-05
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
| 02-01 T0 | 02-01 | 1 | SDK-01 | — | Blocking decision checkpoint: the one-way capture window. No automated verify by nature — the deliverable is a recorded human call. | checkpoint:decision | — (blocking human gate) | n/a | ⬜ pending |
| 02-01 T1 | 02-01 | 1 | SDK-01 | T-02-01, T-02-03, T-02-05, T-02-SC | Confinement untouched; session line still sanitized; stdout stays JSON-RPC-only | tracer / wire integration | `go build ./... && go test ./test/wireoracle/... -count=1 -run 'TestTracerExploreCallSucceeds\|TestToolsListOrderIsDeterministic\|TestToolsListExactSets\|TestEveryRegisteredToolHasASuccessfulCallScenario'` | ✅ `test/wireoracle/oracle_test.go` | ⬜ pending |
| 02-01 T2 | 02-01 | 1 | SDK-01 | T-02-01, T-02-04 | Set-equality and confinement assertions preserved through the client-plumbing port | unit | `go vet ./internal/... && go test ./internal/mcp/... -count=1` | ✅ `internal/mcp/server_test.go` | ⬜ pending |
| 02-02 T1 | 02-02 | 1 | SDK-01 | T-02-07, T-02-08, T-02-09 | Blocking anti-regeneration gate exempts one self-expiring shape and nothing else | unit (table) | `go test ./tools/transcriptfreeze/... -count=1` | ✅ `tools/transcriptfreeze/classify_test.go` | ⬜ pending |
| 02-02 T2 | 02-02 | 1 | SDK-01 | T-02-10 | Guard prose matches the rule it enforces; ci.yml job body untouched | unit + lint | `go test ./tools/transcriptfreeze/... -count=1 && task lint:actions` | ✅ | ⬜ pending |
| 02-03 T1 | 02-03 | 2 | SDK-04 | T-02-11, T-02-12, T-02-13 | Confinement rejection reaches the caller as a tool-visible error; error text pinned | unit (in-memory session) | `go test ./internal/mcp/... -count=1 -run 'TestHandlerErrorIsToolResultNotProtocolError\|TestMissingRequiredArgumentIsToolVisibleError\|TestUnknownArgumentIsRejected\|TestEngineErrorIsToolResult' -v` | ❌ new — `internal/mcp/error_mapping_test.go` created by this task | ⬜ pending |
| 02-03 T2 | 02-03 | 2 | SDK-04 | T-02-13 | The SDK-04 gate demonstrated RED against a `*jsonrpc.Error` mutation | mutation proof | `go test ./internal/mcp/... -count=1 && test -z "$(git status --porcelain internal/mcp/tools.go)"` | ✅ (after T1) | ⬜ pending |
| 02-04 T1 | 02-04 | 2 | SDK-03 | T-02-18 | `testdata/golden`'s byte-identity oracle not weakened; integration tests still spawn the real binary | golden + integration | `task test:golden && task test:integration` | ✅ `testdata/golden/golden_parity_test.go` | ⬜ pending |
| 02-04 T2 | 02-04 | 2 | SDK-03 | T-02-15, T-02-16, T-02-17 | Both archtest non-vacuity self-tests still fire, re-pointed at go-sdk identifiers that resolve | archtest + mutation proof | `go test ./internal/mcp/archtest/... ./internal/cli/archtest/... -count=1 -v` | ✅ | ⬜ pending |
| 02-04 T3 | 02-04 | 2 | SDK-03 | T-02-14, T-02-19 | Post-swap closure re-audited via the existing govulncheck and SBOM paths | build + supply chain | `go build ./... && go vet ./... && go list -m all` (asserted to contain no mark3labs line) plus the repo's govulncheck target | ✅ `Taskfile.yml` | ⬜ pending |
| 02-05 T1 | 02-05 | 3 | SDK-01, SDK-05 | T-02-20, T-02-22 | Human diff review before any regeneration — `PITFALLS.md` Trap C's only real mitigation | checkpoint:human-verify | — (blocking human gate; see Manual-Only Verifications below) | n/a | ⬜ pending |
| 02-05 T2 | 02-05 | 3 | SDK-01, SDK-05 | T-02-20, T-02-23 | 23 transcripts re-frozen only after attribution; divergence record in the commit message | wire integration | `task test:wireoracle` | ✅ `test/wireoracle/oracle_test.go` | ⬜ pending |
| 02-05 T3 | 02-05 | 3 | SDK-01 | T-02-21, T-02-24 | Oracle demonstrated RED against Mutation 1 on the go-sdk backend; harness-unmodified evidenced by diff | mutation proof + full suite | `task test:wireoracle && test -z "$(git status --porcelain internal/mcp/server.go)"`, then `task test` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Sampling continuity check:** no three consecutive tasks lack an automated
verify. The two checkpoints (02-01 T0, 02-05 T1) are each adjacent to tasks
carrying one, and neither is automatable by nature — T0 is a maintainer decision
about an irreversible window, T1 is the phase's stated acceptance mechanism
under D-01, where the judgment a comparator cannot make IS the deliverable.

**Wave 0 status:** the one MISSING reference above is
`internal/mcp/error_mapping_test.go`, created by the task that depends on it
(02-03 T1) — SDK-04's requirement is the test's existence, so there is no
scaffold to build first.

**Expected-red window:** `TestFrozenTranscriptsMatch` is expected RED from
02-01 T1 until 02-05 T2. Plans 02-01, 02-03 and 02-04 therefore scope their wire
verification to the re-capturing tests (`TestTracerExploreCallSucceeds`,
`TestToolsListExactSets`, `TestEveryRegisteredToolHasASuccessfulCallScenario`,
`TestToolsListOrderIsDeterministic`) rather than the byte comparison. Relaxing
the byte comparison to close that window early is a prohibition, not an option.

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
