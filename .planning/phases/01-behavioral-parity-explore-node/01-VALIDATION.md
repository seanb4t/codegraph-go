---
phase: 1
slug: behavioral-parity-explore-node
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-14
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

Task IDs assigned by the planner; commands lifted from RESEARCH.md's test map.

| Requirement | Behavior | Test Type | Automated Command | File Exists |
|-------------|----------|-----------|-------------------|-------------|
| EXPL-01 | Multi-word query tokenizes, doesn't 0-match | unit | `go test ./internal/query/... -run TestTokenize -v` | ❌ W0 (new tokenize.go + test) |
| EXPL-02 | RWR ranks structurally-connected symbol above lexical-only match | unit + golden | `go test ./internal/query/... -run TestComputeGraphRelevance -v` + golden diff | ❌ W0 (new rwr.go + test; extend capture.sh) |
| EXPL-03 | Weakly-connected `Test*` func doesn't top results | golden | fixture diff against synthetic D-03(c) corpus | ❌ W0 (new synthetic fixture) |
| EXPL-04 | "⚠️ no covering tests" warning fires/doesn't fire correctly | unit | `go test ./internal/query/... -run TestNoCoveringTestsWarning -v` | ❌ W0 |
| EXPL-05 | CLI/MCP byte-identical explore output | golden parity | `go test ./testdata/golden/... -run TestGoldenParity -v` (extend for MCP) | ✅ extend `golden_parity_test.go` |
| NODE-01/02 | Multi-def enumeration, header, budget, overflow | unit + golden | `go test ./internal/query/... -run TestNodeMultiDef -v` | ❌ W0 (new overloaded-symbol fixture) |
| NODE-03 | File/line narrowing never empties set | unit | `go test ./internal/query/... -run TestNarrowNeverEmpty -v` | ❌ W0 |
| NODE-04 | Single-def byte-comparable | golden | `go test ./testdata/golden/... -run TestGoldenFixturesExist -v` | ✅ covered by existing node.json + RenderNode |
| TEST-01 | Harness: CLI+MCP, closes v0.1 blind spot | integration | `go test ./testdata/golden/... -v` | ✅ extend capture.sh + add synthetic corpus (D-03) |

*Status legend: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/query/tokenize.go` + `tokenize_test.go` — EXPL-01
- [ ] `internal/query/rwr.go` + `rwr_test.go` — EXPL-02 core (RWR α=0.25, 25 iters)
- [ ] `internal/query/explore_gate.go` + test — EXPL-03 (5-way relevance gate, `0.06`)
- [ ] Extend `internal/query/node.go` + `node_test.go` — NODE-01/02/03
- [ ] Extend `testdata/golden/capture.sh` — multi-word explore (no `--max-files 1`), overloaded `node` (no `-f`) on BOTH corpora, BOTH CLI + MCP surfaces
- [ ] New synthetic fixture corpus per D-03 — `testdata/golden/corpus/synthetic-parity/` (overloaded symbols, multi-word query, `Test*`-heavy weakly-connected, structural-beats-lexical)
- [ ] MCP-surface capture path in `capture.sh` (currently CLI-only) — drive TS's stdio `codegraph_explore`/`codegraph_node` tools programmatically
- [ ] **D-09 foundation (F1–F5):** new edge-kind constants → extractor emission → `index --force` re-index → golden regen. F5 stales committed `explore.json` fixtures — plan the regen explicitly.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Live TS 1.3.1 golden capture | TEST-01 | Requires the live TS CLI + a driver for its stdio MCP server; run once while TS 1.3.1 is installed (D-01 time-sensitive) | `cd testdata/golden && ./capture.sh` with TS `codegraph` on PATH; commit the regenerated corpus |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 120s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
