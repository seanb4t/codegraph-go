---
phase: 1
slug: behavioral-parity-explore-node
status: complete
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-14
audited: 2026-07-16
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Sourced from 01-RESEARCH.md `## Validation Architecture` (commit 6de928a).
> Task-ID rows are planner-assigned; the gsd-nyquist-auditor reconciles them
> after PLAN.md exists.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib) + `testdata/golden/` fixture-diff harness |
| **Config file** | none — `go test ./...` |
| **Quick run command** | `go test ./internal/query/... ./testdata/golden/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~20s quick · ~120s full |

**Gotcha (from repo memory `whad9x6gxq`):** `internal/daemon` TestSoak +
TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock are pre-existing flakes
under full-suite parallel load. On a daemon FAIL in `go test ./...`, re-run
`go test ./internal/daemon/ -count=1` before calling it a regression — Phase 1
touches no daemon files.

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/query/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** ~120 seconds (full suite)

---

## Per-Requirement Verification Map

Reconciled against delivered tests by the 2026-07-16 Nyquist audit (all commands re-run green that day).

| Requirement | Behavior | Test Type | Automated Command / Evidence | File Exists | Status |
|-------------|----------|-----------|------------------------------|-------------|--------|
| EXPL-01 | Multi-word query tokenizes, doesn't 0-match | unit | `go test ./internal/query/ -run 'TestExtractSearchTerms\|TestExploreMultiWord' -count=1` | ✅ `tokenize_test.go`, `explore_test.go` | ✅ green |
| EXPL-02 | RWR ranks structurally-connected symbol above lexical-only match | unit + golden | `go test ./internal/query/ -count=1` (rwr/gather/expand/seeding/scoring/explore_gate/tokenize tests — 90+ funcs incl. `TestComputeGraphRelevance_*`, `TestRWRDeterminism_RepeatedRunsIdentical`, `TestExploreStructuralBeatsLexical`) + `go test ./internal/indexer/ ./internal/indexer/{go,java,csharp,py,ts}extract/ -count=1` (D-09 edge-kind emission + resolve) + golden behavioral suite | ✅ 7 query test files + 5 extractor suites | ✅ green |
| EXPL-03 | Weakly-connected `Test*` func doesn't top results | unit + golden | `go test ./internal/query/ -run 'TestFileRelevanceGate\|TestHardTestExclusion\|TestGatherMultiTermReRank\|TestDistinctiveIdentifierExemption\|TestExploreTestNotTop' -count=1` + `TestGoldenBehavioralSyntheticParity/explore-multi` (D-03(c) corpus) | ✅ `explore_gate_test.go`, `scoring_test.go`, `gather_test.go` | ✅ green |
| EXPL-04 | "⚠️ no covering tests" warning fires/doesn't fire correctly | unit | `go test ./internal/query/ -run TestNoCoveringTestsWarning -count=1` | ✅ `render_markdown_test.go:402` | ✅ green |
| EXPL-05 | CLI/MCP byte-identical explore output | golden parity | `go test ./testdata/golden/ -run TestExploreCLIMatchesMCP -count=1` | ✅ `golden_parity_test.go:1504` | ✅ green |
| NODE-01/02 | Multi-def enumeration, header, budget, overflow | unit | `go test ./internal/query/ -run 'TestNodeMultiDef\|TestIsGeneratedFile\|TestRenderNodeMultiDef' -count=1` | ✅ `node_test.go:131`, `render_markdown_test.go` | ✅ green |
| NODE-03 | File/line narrowing never empties set — AND is reachable end-to-end (CLI `--line` + MCP `line`) | unit + parity regression | `go test ./internal/query/ -run 'TestNarrowNeverEmpty\|TestNodeLineHintNarrowsToSingleDef\|TestNodeFileHintNeverEmptiesOnNoMatch' -count=1` + `go test ./testdata/golden/ -run TestNodeLineHintCLIMatchesMCP -count=1` (reachability regression tests from fix e298e26 — the original unit test alone missed the unwired-function gap) | ✅ `node_test.go:204`, `render_markdown_test.go:583,611`, `golden_parity_test.go:1599` | ✅ green |
| NODE-04 | Single-def byte-comparable | unit + golden | `go test ./internal/query/ -run 'TestRenderNodeSingleDefUnchanged\|TestNodeMultiDefWiring' -count=1` + `go test ./testdata/golden/ -run TestNodeCLIMatchesMCP -count=1` | ✅ `render_markdown_test.go`, `golden_parity_test.go:1547` | ✅ green |
| TEST-01 | Harness: CLI+MCP, closes v0.1 blind spot | integration (golden) | `go test ./testdata/golden/... -count=1` (`TestGoldenBehavioralSyntheticParity`, `TestGoldenBehavioralRealCorpora`, `TestGoSideFixturesRegenerated` — real assertions, AD-01..04 divergences logged not swallowed) | ✅ `golden_parity_test.go:1132,1253` | ✅ green |
| D-09 fix | Overloaded methods both synthesize `overrides` edges (no name-only collapse) | unit regression | `go test ./internal/indexer/ -run TestResolveOverrides_OverloadedMethodsBothSynthesized -count=1` (from fix b87b8cf) | ✅ `resolve_test.go:266` | ✅ green |

*Status legend: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `internal/query/tokenize.go` + `tokenize_test.go` — EXPL-01 (plan 01-03)
- [x] `internal/query/rwr.go` + `rwr_test.go` — EXPL-02 core (RWR α=0.25, 25 iters — plan 01-06)
- [x] `internal/query/explore_gate.go` + test — EXPL-03 (5-way relevance gate — plan 01-14)
- [x] Extend `internal/query/node.go` + `node_test.go` — NODE-01/02/03 (plan 01-04; NODE-03 reachability completed by fix e298e26)
- [x] Extend `testdata/golden/capture.sh` — multi-word explore, overloaded `node`, BOTH corpora, BOTH CLI + MCP surfaces (plan 01-01)
- [x] New synthetic fixture corpus per D-03 — `testdata/golden/corpus/synthetic-parity/` (plan 01-01)
- [x] MCP-surface capture path in `capture.sh` (plan 01-01)
- [x] **D-09 foundation (F1–F5):** edge-kind constants → extractor emission → re-index → golden regen (plans 01-02/05/08/09/15; F5 regen in plan 01-17)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live TS 1.3.1 golden capture | TEST-01 | Requires the live TS CLI + a driver for its stdio MCP server; run once while TS 1.3.1 is installed (D-01 time-sensitive) | `cd testdata/golden && ./capture.sh` with TS `codegraph` on PATH; commit the regenerated corpus |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — every requirement row carries a runnable command, re-executed green 2026-07-16
- [x] Sampling continuity: no 3 consecutive tasks without automated verify — 17 plans, each with per-task verify blocks (per SUMMARYs' coverage entries)
- [x] Wave 0 covers all MISSING references — all 8 Wave 0 items delivered
- [x] No watch-mode flags — no build tags or skip-by-default gating on any phase test
- [x] Feedback latency < 120s — query+indexer ~20s, golden ~12s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** auto-validated 2026-07-16 (/gsd-validate-phase audit — zero gaps found)

---

## Validation Audit 2026-07-16

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

Audit method: reconciled the planner's draft map against delivered tests using all 17 SUMMARY coverage blocks + 01-VERIFICATION.md (passed 11/11 on re-verification), confirmed all key test functions exist (114 test funcs across 11 phase test files), and re-ran the covering suites live: `go test ./internal/query/ ./internal/indexer/... -count=1` and `go test ./testdata/golden/... -count=1` — all green. The map now includes the two post-verification fix regressions (NODE-03 reachability from e298e26; D-09 overload-safe overrides from b87b8cf) — the original draft's unit-only NODE-03 row is exactly the shape that missed the unwired-function critical, so the reachability tests are recorded as load-bearing coverage, not extras. The one manual-only item (live TS 1.3.1 golden capture) was performed during execution — the captured fixtures are committed and guarded by `TestGoSideFixturesRegenerated`.
