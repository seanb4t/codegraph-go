---
phase: 5
slug: mcp-resources-capability-claims-drift-guard
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-12
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` package |
| **Config file** | none — `go test ./...` via `Taskfile.yml`'s `test`/`test:wireoracle` tasks |
| **Quick run command** | `go test ./internal/mcp/...` |
| **Full suite command** | `go test ./...` and `task test:wireoracle` (`go test ./test/wireoracle/...`) |
| **Estimated runtime** | ~30 seconds (unit) / ~60 seconds (full incl. wire-oracle) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/mcp/...`
- **After every plan wave:** Run `go test ./... && task test:wireoracle`
- **Before `/gsd-verify-work`:** Full suite must be green, including a demonstrated-red-then-reverted mutation proof appended to `test/wireoracle/MUTATION-PROOF.md` for GUARD-02 (success criterion 3)
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 05-01-T1 | 01 | 1 | RSRC-01, RSRC-02, RSRC-03 | T-05-02, T-05-03 | Unregistered URI never reaches the filesystem; capability advertised explicitly, never inferred | unit | `go test ./internal/mcp/... -run 'TestResources' -count=1 -v` | ❌ W0 | ⬜ pending |
| 05-01-T2 | 01 | 1 | RSRC-01 | T-05-03 | 25 transcripts re-frozen under one named cause; 4 provably unaffected | wire | `go test ./test/wireoracle/... -count=1` | ✅ exists | ⬜ pending |
| 05-01-T3 | 01 | 1 | RSRC-01, RSRC-02 | T-05-03 | `cacheScope: private` anchored on both resource responses | wire | `go test ./test/wireoracle/... -count=1` | ❌ W0 | ⬜ pending |
| 05-02-T1 | 02 | 2 | RSRC-01, RSRC-02 | T-05-01 | No host facts in node/search/callers/callees fact-sheets | unit | `go test ./internal/mcp/... -run 'TestResources' -count=1 -v` | ✅ after 05-01 | ⬜ pending |
| 05-02-T2 | 02 | 2 | RSRC-01, RSRC-02 | T-05-01 | Numeric claims in impact/explore agree with engine constants | unit | `go test ./internal/mcp/... -run 'TestResources' -count=1 -v` | ✅ after 05-01 | ⬜ pending |
| 05-02-T3 | 02 | 2 | RSRC-01, RSRC-02 | T-05-01, T-05-04 | Filter/index-state docs derived from `server.go` only | unit + wire | `go test ./... -count=1 && go test ./test/wireoracle/... -count=1` | ✅ after 05-01 | ⬜ pending |
| 05-03-T1 | 03 | 3 | GUARD-02 | T-05-05, T-05-06 | Tool add/remove/rename fails bidirectionally; checker proven non-vacuous | unit | `go test ./internal/mcp/... -run 'TestResourceFileSetMatchesToolNames\|TestResourceStemSetDiffIsNotVacuous' -count=1 -v` | ❌ W0 | ⬜ pending |
| 05-03-T2 | 03 | 3 | GUARD-01 | T-05-01, T-05-05, T-05-06 | Numeric/count/env-var claims pinned; no host facts in any resource file | unit | `go test ./internal/mcp/... -run 'TestMCPResourceNumericClaimsMatchToolSchemas\|TestResourceCount\|TestResourceEnvVarNamesAreReal\|TestResourceContentCarriesNoHostFacts\|TestResourceHostFactCheckerIsNotVacuous' -count=1 -v` | ❌ W0 | ⬜ pending |
| 05-03-T3 | 03 | 3 | GUARD-01, GUARD-02 | T-05-05 | Mutations 6/7/8 observed RED, reverted byte-clean | unit (mutation proof) | `git status --porcelain -- internal/ && go test ./... -count=1` | ✅ after 05-03-T2 | ⬜ pending |
| 05-04-T1 | 04 | 4 | RSRC-02, RSRC-03 | — | 9 per-URI reads served with no index present | wire | `go test ./test/wireoracle/... -count=1` | ✅ after 05-01 | ⬜ pending |
| 05-04-T2 | 04 | 4 | RSRC-01, RSRC-02, RSRC-03 | T-05-02 | Unknown URI returns `-32602` disclosing no server configuration | wire | `go test ./test/wireoracle/... -count=1 -v -run 'TestEveryAdvertisedResourceURIHasASuccessfulReadScenario\|TestScenarioCountIsExact\|TestTranscriptSetMatchesScenarioSet\|TestSpecAnchorsHold'` | ❌ W0 | ⬜ pending |
| 05-04-T3 | 04 | 4 | RSRC-03 | T-05-03 | Mutations 9/10 observed RED, reverted byte-clean; oracle re-proved non-vacuous | wire (mutation proof) | `git status --porcelain -- internal/ && go test ./... -count=1 && go test ./test/wireoracle/... -count=1` | ✅ after 05-04-T2 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Sampling continuity:** every task above carries an `<automated>` verify. No three consecutive
tasks run without one. Max feedback latency is bounded by the slowest command, `go test
./test/wireoracle/... -count=1`, which grows from ~29 to ~42 subprocess-spawning scenarios in this
phase — 11 of the 13 additions skip `codegraph init` (`Index: false`), so the added cost is
handshake-and-one-request per scenario, not indexing. Re-measure at 05-04-T3 and record the real
number in the SUMMARY; if it exceeds the 60s budget below, that is a finding to record, not a
reason to drop scenarios.

**Two deliberate red windows.** 05-01-T1 leaves `go test ./test/wireoracle/...` RED (turning
`capabilities.resources` on moves 25 frozen transcripts), closed by 05-01-T2. 05-02-T1 and
05-02-T2 leave it RED (`resources-list` advertises more entries than the tracer froze), closed by
05-02-T3. Both are expected and stated in the owning plans' acceptance criteria. Neither crosses a
plan boundary, and neither crosses the phase boundary — criterion 5 requires main green there.

---

## Wave 0 Requirements

Every MISSING test reference above is created inside this phase, by the plan named:

- [ ] `internal/mcp/resources_test.go` — unit list/read coverage, reusing `server_test.go`'s existing `newTestSession` → **05-01 Task 1** (created BEFORE the code it tests, per the task's `<behavior>` block)
- [ ] `internal/mcp/resources.go` — `go:embed` directive, `resourceURIFor` map, `registerResources(s)` → **05-01 Task 1**
- [ ] `internal/mcp/resources/*.md` — the 10 fact-sheet/behavior-doc files (D-01 … D-10) → **05-01 Task 1** (`explore.md`) and **05-02 Tasks 1-3** (the other 9)
- [ ] `internal/mcp/resources_schema_drift_test.go` — GUARD-01/GUARD-02 → **05-03 Tasks 1-2**
- [ ] `test/wireoracle/scenarios.go` request helpers + `Scenario` entries + `ExpectedScenarioCount` → **05-01 Task 3** (helpers, 29→31) and **05-04 Tasks 1-2** (31→40→42)
- [ ] `test/wireoracle/oracle_test.go` — `resourceURIsFromCapture`, `TestEveryAdvertisedResourceURIHasASuccessfulReadScenario` → **05-04 Task 2**
- [ ] `test/wireoracle/anchors.go` — `assertResourceCacheControl` → **05-01 Task 3**; unknown-URI `-32602` anchor → **05-04 Task 2**

**Corrected blast radius for the transcript re-freeze.** 05-RESEARCH.md Pitfall 2 predicted that
all 29 frozen transcripts would move when `capabilities.resources` went live, and instructed that
"fewer than 29" be treated as evidence the capability had not reached every code path. That is
wrong and would raise a false alarm. Measured against the pre-change tree, **25** transcripts carry
a `capabilities` object; the other four produce neither an `initialize` nor a `server/discover`
result and must come back byte-identical: `edge-call-before-initialize`,
`modern-listen-catalog-change`, `modern-meta-invalid-params`, `modern-meta-unsupported-version`.
05-01 Task 2 carries this correction and asserts the count of 25 directly.

**Async-dispatch constraint, upgraded from assumed to verified.** 05-RESEARCH.md recorded as
`[ASSUMED]` (A2) that `resources/read` shares `tools/call`'s async worker-pool ordering race. It
does: go-sdk routes every call except `initialize` through `jsonrpc2.Async(ctx)`
(`$(go env GOMODCACHE)/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go:1910-1915`).
Every new scenario therefore carries at most one async request, placed last — which is why
per-URI wire coverage costs 10 scenarios rather than one batched scenario, and why 05-RESEARCH.md
Open Question 3 resolves to full per-URI coverage.

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — 12/12 tasks carry one
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references — every one is mapped to an owning plan and task above
- [x] No watch-mode flags — every command is `-count=1` or a one-shot `go test`
- [ ] Feedback latency < 60s — re-measure `go test ./test/wireoracle/...` at 42 scenarios in 05-04-T3 and record the real number in that SUMMARY
- [ ] `nyquist_compliant: true` set in frontmatter — set by `/gsd-validate-phase` after execution, not by the planner

**Approval:** pending
