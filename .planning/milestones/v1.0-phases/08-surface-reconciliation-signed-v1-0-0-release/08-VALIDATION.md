---
phase: 8
slug: surface-reconciliation-signed-v1-0-0-release
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: false
wave_0_complete: true
requirements_total: 9
requirements_automated: 7
requirements_manual_only: 2
gaps_found: 3
gaps_closed: 2
gaps_deferred: 1
created: 2026-07-19
validated: 2026-07-26
---

# Phase 8 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

Reconciled from a seeded draft on 2026-07-26 against the 9 finalized PLAN/SUMMARY
pairs at commit `80c452c`. Every mapped test below was **executed**, not inferred
from filenames — the COVERED bar here is "test exists, targets the behavior, and
runs green."

**Why `nyquist_compliant: false` (PARTIAL, not NOT-VALIDATED):** 7 of 9
requirements have green automated coverage. REL-02 is inherently manual (a tag
push cannot be automated before the release exists). REL-03 retains one
*automatable* gap — `docs/BENCHMARKS.md`'s table is not pinned to its committed
raw run JSONs — consciously deferred to Manual-Only rather than closed. One open
automatable gap means coverage is partial by choice, not complete. Closing
MO-08-03 flips this to `true`.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard `go test` |
| **Quick run command** | `go test ./internal/cli/... ./internal/query/...` |
| **Full suite command** | `go test ./... && go test ./testdata/golden/...` |
| **Estimated runtime** | ~60–120 seconds (golden corpus ~27s of it) |

**Known-flaky, unrelated to this phase:** `internal/daemon` carries four
load-sensitive timing tests (`TestSoak`,
`TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`,
`TestDaemonFlushLockRequeueGivesUpPerEpisode`,
`TestRunWatchdogCancelsRunOnSimulatedReparent`) that fail under full-suite
parallel load and pass in isolation. `git diff 529d818..HEAD -- internal/daemon/`
is empty — no Phase-8 commit touched that package. On a daemon FAIL, re-run
`go test ./internal/daemon/ -count=1` before treating it as a regression.

---

## Sampling Rate

- **After every task commit:** `go test ./internal/cli/... ./internal/query/...`
- **After every plan wave:** `go test ./... && go test ./testdata/golden/...`
- **Before `/gsd-verify-work`:** full suite green
- **Max feedback latency:** 120 seconds

---

## Per-Requirement Verification Map

| Req | Plan(s) | Behavior | Automated Test | Command | Status |
|-----|---------|----------|----------------|---------|--------|
| SURF-01 | 01, 04 | `impact` default depth 5→2 in the shared engine; `MaxDepth=50` ceiling and negative-depth rejection retained | `TestClampDepth`, `TestValidateDepth` (`internal/query/engine_test.go:157,234`) | `go test ./internal/query/` | ✅ green |
| SURF-01 | 01 | MCP tool schemas state the *real* engine defaults (drift barrier — closes GAP-2) | `TestMCPToolSchemaNumericClaimsMatchEngineConstants` (`internal/mcp/tools_schema_drift_test.go`) | `go test ./internal/mcp/` | ✅ green |
| SURF-02 | 02 | `files --dir` prefix filter; prefix-match not glob (D-03), with path-separator boundary | `TestFiles` incl. `dirPrefixMatches` subtests (`internal/query/files_status_test.go:59,248`) | `go test ./internal/query/` | ✅ green |
| SURF-03 | 01, 02, 03, 05, 06 | Short-flag aliases; `upgrade --force` bypasses only the same-version no-op, never signature verification; `install/uninstall -y` | `TestUpgradeRun_ForceStillVerifiesBeforeSwap`, `TestUpgradeRun_ForceReinstallsSameVersion` (`internal/upgrade/upgrade_test.go:220,174`); `TestInstall_Yes_ShortCircuitsBeforeInteractiveBranch`, `TestUninstall_Yes_…` (`internal/cli/install_test.go:418,473`) | `go test ./internal/upgrade/ ./internal/cli/` | ✅ green |
| SURF-04 | 04, 05 | Depth-bounded `Affected` BFS with test-leaf pruning, dangling-edge skip, interface-dispatch composition, deterministic ordering | `TestAffected`, `TestAffectedNegativeDepthRejected`, `TestAffectedEmptyFilesReturnsEmptyResultNoError`, `TestAffectedDepthBFSWithTestLeafPruning`, `TestAffectedSkipsDanglingEdgeInsteadOfFailing`, `TestAffected_DispatchTraversal(_NoImplementsEdgesUnaffected)`, `TestClampAffectedDepth`, `TestImpactCallersAffectedDeterministicAcrossRepeatedCalls` (`internal/query/traverse_test.go`) | `go test ./internal/query/` | ✅ green |
| SURF-04 | 05 | `affected` scripting surface: `--stdin` never hangs, CRLF/blank handling, line + count caps, `--filter`, `--quiet` paths-only | `TestAffectedStdinNeverHangs`, `TestAffectedEmptyStdinNoArgs`, `TestAffectedQuietPathsOnly`, `TestAffectedJSONQuiet`, `TestAffectedStdinCRLFAndBlankLines`, `TestAffectedStdinLineTooLong`, `TestAffectedStdinTooManyFiles`, `TestAffectedFilter`, `TestAffectedCmd` (`internal/cli/affected_test.go`) | `go test ./internal/cli/` | ✅ green |
| SURF-04 | 05 | `--quiet` suppresses paths containing `\n`/`\r` — defense-in-depth for T-08-05-01 (closes GAP-1) | `TestAffectedQuietSkipsControlCharacterPaths` (`internal/cli/affected_newline_test.go`) | `go test ./internal/cli/` | ✅ green |
| SURF-05 | 02, 03, 04, 06 | `docs/FLAG-PARITY.md` covers every registered cobra flag; fail-closed on a missing doc | `TestFlagParityDocCoversRegisteredFlags` (`internal/cli/flag_parity_test.go:40`) | `go test ./internal/cli/` | ✅ green |
| REL-01 | 07 | Zero CGo in the (non-empty) charm.land dependency closure | `TestCharmCgoClosure` (`internal/cli/present/archtest/charm_cgo_test.go:87`) — logs *"charm.land closure audited: 10 packages, 0 with CgoFiles"* | `go test ./internal/cli/present/archtest/` | ✅ green |
| REL-02 | 08, 09 | Signed `v1.0.0` tag → release pipeline → post-release cosign + SLSA verification | — | — | ⬜ manual-only (MO-08-01) |
| REL-03 | 08 | Published head-to-head benchmark numbers | — (harness `bench.yml`; numbers independently re-derived during the 2026-07-26 security audit) | — | ⬜ manual-only (MO-08-02, MO-08-03) |
| REL-04 | 01, 06, 09 | Drop-in gate: behavioral parity harness + flag-parity audit both green before the caveat is retired | `TestGoldenParity_{TSJS,Python,Java,CSharp}`, `TestGoldenFixturesExist` (`testdata/golden/`) + `TestFlagParityDocCoversRegisteredFlags` | `go test ./testdata/golden/... ./internal/cli/` | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements — no Wave 0 test-infrastructure
work was required. `go test` was already the framework; both gaps closed on
2026-07-26 landed as ordinary test files alongside their peers.

---

## Gaps Found and Closed (2026-07-26)

Both closed gaps were verified **non-vacuous**: the guard was broken, the test was
observed to fail, then the guard was restored. A test that passes with *and*
without the behavior it protects is worthless — this codebase already shipped one
guard (CR-01) that read as present and never fired.

### GAP-1 — SURF-04: `--quiet` control-character skip was untested → CLOSED

`internal/cli/affected.go:122-124` skips any `FilePath` containing `\n`/`\r`
before emitting it to `--quiet`'s one-path-per-line machine-readable stream. That
is the defense-in-depth control for HIGH threat T-08-05-01 (line injection into a
downstream shell pipeline), and deleting it failed no test.

Closed by `TestAffectedQuietSkipsControlCharacterPaths`
(`internal/cli/affected_newline_test.go`), which indexes a real
`pkga/ev\nil_test.go` fixture and drives the actual CLI. A `--json` sanity gate
first proves the hostile path reached the graph, so the `--quiet` assertion cannot
pass vacuously, and a clean sibling path proves the guard suppresses *only* the
hostile entry.

**Non-vacuity (guard deleted):**
```
--- FAIL: …/no_emitted_line_contains_a_control_character
    affected --quiet: embedded-newline path leaked into machine-readable output
    as "pkga/ev" (full output "pkga/ev\nil_test.go\npkga/pkga_test.go\n")
```
— the injected extra line T-08-05-01 describes, reproduced verbatim.

*Caveat:* filesystem-capability-dependent. It genuinely exercised the guard on
macOS/APFS; on a filesystem that refuses such a filename it degrades to `t.Skip`
rather than failing. Deliberate — hard-failing there would be a flaky test, not a
stronger one.

### GAP-2 — SURF-01: MCP tool-schema defaults were unpinned → CLOSED

A **demonstrated escape, not a hypothetical.** This phase moved `defaultDepth`
5→2; CLI help and `docs/FLAG-PARITY.md` were both updated, but
`internal/mcp/tools.go` kept advertising `"BFS depth (default 5, max 50)"`, so MCP
agent clients were told the wrong default. It survived the whole phase *and* a
deep code review. `TestFlagParityDocCoversRegisteredFlags` structurally cannot
catch it — it walks the cobra command tree and never inspects MCP schemas.

Closed by `TestMCPToolSchemaNumericClaimsMatchEngineConstants`
(`internal/mcp/tools_schema_drift_test.go`). It extracts claimed values from the
*actually registered* `mcp.Tool` schemas (tool description + every
`InputSchema.Properties[*].description`, matching `(?i)\b(default|max)\s+(\d+)`)
and the expected values from `internal/query/validate.go`'s const block via
`go/parser` — so neither side restates a literal the other checks. Any future
unmapped `default N` / `max N` claim in any registered tool fails the test rather
than sliding through.

**Non-vacuity (independently re-verified by the orchestrator, not only the auditor):**
reintroducing the original `"default 5"` description produced
```
codegraph_impact advertises default 5 but internal/query.defaultDepth is 2 —
MCP clients are being told the wrong value (description: "BFS depth (default 5, max 50)")
```
Restored → clean tree, test green.

Also pinned beyond the required minimum: `codegraph_explore.max_files` "default 5"
verified against `defaultMaxFiles = 5`. `codegraph_files.depth`'s "(0 = unlimited)"
carries no numeric claim, so nothing to pin.

### GAP-3 — REL-03: `BENCHMARKS.md` not pinned to raw runs → DEFERRED

Recorded as MO-08-03 below by explicit decision, not dropped.

---

## Manual-Only Verifications

| ID | Behavior | Requirement | Why Manual | Test Instructions |
|----|----------|-------------|------------|-------------------|
| MO-08-01 | Actual `v1.0.0` tag push → signed release build | REL-02 | Inherently un-automatable pre-release: the tag push is a maintainer action and the artifacts do not exist until it happens. **Not yet performed** — the maintainer withheld the go-ahead (`08-UAT.md` test 1: *"I don't think we're ready to declare 1.0"*); no `v1.0.0` tag exists. | Follow `docs/RELEASE-PROCEDURES.md`, incl. §6 post-release `cosign verify-blob` + `slsa-verifier verify-artifact`. `08-SECURITY.md` records this as the open residual on T-08-09-02/03. |
| MO-08-02 | Published head-to-head benchmark numbers | REL-03 | Requires running the bench harness against a live TS 1.3.1 install; cannot run in a unit test. | Run the `bench.yml` methodology; refresh `docs/BENCHMARKS.md`. Numbers were independently re-derived during the 2026-07-26 security audit (all 12 published cells recompute as median-of-3 from the committed raw runs). |
| MO-08-03 | `docs/BENCHMARKS.md` table matches its committed raw run JSONs | REL-03 | **Automatable but deliberately deferred** (GAP-3). A test could recompute median-of-3 from `tools/bench/headtohead-linux-amd64-ci-20260719-run{1,2,3}.json` and assert every cell of the doc's table. Deferred as the most involved of the three gaps (markdown-table parsing) and the least likely to silently regress. **This is the one open automatable gap keeping `nyquist_compliant: false`.** | Until automated: re-derive by hand as the security audit did — median of the 3 run JSONs per cell vs `docs/BENCHMARKS.md:109-114`. |

---

## Validation Audit 2026-07-26

| Metric | Count |
|--------|-------|
| Requirements total | 9 |
| Automated (COVERED, green) | 7 |
| Manual-only | 2 (REL-02, REL-03) |
| Gaps found | 3 |
| Resolved | 2 (GAP-1, GAP-2) |
| Deferred to manual-only | 1 (GAP-3 → MO-08-03) |
| Escalated | 0 |
| Implementation files modified | 0 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (none required)
- [x] No watch-mode flags
- [x] Feedback latency < 120s
- [x] Every mapped test executed, not inferred — all green
- [x] Both new tests proven non-vacuous by breaking the guard and observing failure
- [ ] `nyquist_compliant: true` — **blocked on MO-08-03**, the one open automatable gap

**Approval:** validated (partial) 2026-07-26
