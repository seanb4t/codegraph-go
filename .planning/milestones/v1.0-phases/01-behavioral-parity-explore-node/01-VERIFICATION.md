---
phase: 01-behavioral-parity-explore-node
verified: 2026-07-15T13:05:00Z
status: passed
score: 11/11 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 9/11
  gaps_closed:
    - "NODE-03 file/line narrowing wired end-to-end (Engine.Node line param + CLI --line/-l + MCP line schema), reachable from every surface, never-empty — fixed in e298e26"
    - "D-09 overrides synthesis no longer collapses overloaded methods (name->[]methodID with per-overload arity check) — fixed in b87b8cf"
  gaps_remaining: []
  regressions: []
gaps: []
deferred: []
human_verification: []
---

# Phase 1: Behavioral Parity — explore & node Verification Report

**Phase Goal:** Agents and users get TS-identical `explore` and `node` results — graph-relevance-ranked exploration and full multi-definition disambiguation — proven equivalent on both the CLI and the `codegraph_explore` MCP surface by a behavioral fixture harness that lands with the algorithm.
**Verified:** 2026-07-15T13:05:00Z
**Status:** passed
**Re-verification:** Yes — after gap closure (previous: gaps_found, 9/11)

## Re-Verification Summary

The two blocking gaps from the initial verification were fixed with FAIL→PASS regression tests and independently re-verified here:

- **Gap 1 (NODE-03 unreachable) → CLOSED** by commit `e298e26`. `Engine.Node` now takes `(symbol, file string, line *int)` (`internal/query/node.go:314`), threaded through a new `--line`/`-l` CLI flag (`internal/cli/node.go:25,37-39`) and a new `line` MCP schema param + handler (`internal/mcp/tools.go:114,182-194`). `narrowNodeMatches` is now called from the real `Engine.Node` path (`node.go:349`) — no longer dead code. NODE-04's exact single-def byte-parity is preserved: `file != "" && line == nil` still hits the pre-existing `resolveNodeForDetail` exact-match fast path first (`node.go:329-333`). Verified live: `node Run --line 92` narrows 6→2 defs; `node Run --line 999999` falls back to nearest (single def, exit 0); `node Run -f nonexistent/nowhere.go` falls back to the full 6-def set (exit 0) — never empty, never errors. Regression tests `TestNodeLineHintNarrowsToSingleDef`, `TestNodeFileHintNeverEmptiesOnNoMatch`, `TestNodeLineHintCLIMatchesMCP` all pass.
- **Gap 2 (overrides overload collapse) → CLOSED** by commit `b87b8cf`. `synthesizeOverrides` (`internal/indexer/resolve.go:464-543`) now keys the type→method map as `map[string]map[string][]string` (typeID → methodName → ALL methodIDs) and checks every same-named candidate's arity independently on both the self side (`node.go:519-520`) and the supertype side (the `walkSupertypes` closure loops all candidates, `resolve.go:527-532`). Confirmed by direct read that `Foo(int)` and `Foo(String)` are no longer collapsed. Regression test `TestResolveOverrides_OverloadedMethodsBothSynthesized` asserts 0→2 distinct override edges with no cross-arity misattribution — passes.

Full suite `go test ./... -count=1` is entirely green — all 34 test packages including `internal/daemon` (no flake this run) and `testdata/golden` (the behavioral harness + all CLI==MCP parity tests), confirming no regression in the 9 previously-passing must-haves. Fixes documented in `01-FIX-SUMMARY.md` (`f710ca3`).

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | EXPL-01: `explore` accepts a multi-word variadic `<query...>`, tokenized and stopword-filtered | ✓ VERIFIED | `codegraph explore --help` shows `Usage: codegraph explore <query...>`; `internal/query/tokenize.go` implements `extractSymbolsFromQuery`/`extractSearchTerms`; live smoke test `go run ./cmd/codegraph explore ReconcileLedger balance` returns ranked results, not a 0-match |
| 2 | EXPL-02: `explore` ranks by graph relevance (RWR, α=0.25, 25 iters, 9 edge kinds) instead of lexical match | ✓ VERIFIED | `internal/query/rwr.go`: `rwrAlpha=0.25`, `rwrIterations=25` (fixed, no early-exit), `RankEdges` map has exactly 9 kinds (calls/references/extends/implements/overrides/instantiates/returns/type_of/imports); `TestGoldenBehavioralSyntheticParity` passes including the structural-beats-lexical case; live smoke test shows `ReconcileLedger` (structurally central) ranked ahead of the lexically-matching but graph-isolated `AccountBalanceHelper`. The `overrides` RankEdges input is now correct for overloaded methods (gap 2 fix, b87b8cf). |
| 3 | EXPL-03: file-relevance gate stops weakly-connected `Test*` funcs from topping results | ✓ VERIFIED | `internal/query/explore_gate.go` implements the 5-way OR gate exactly as D-08 requires (graph-mass≥6%, central, entry/named, rescued, ≥2 term hits), never pruning below 2 files; `TestGoldenBehavioralSyntheticParity/explore-multi` (the D-03(c) Test*-heavy synthetic case) passes |
| 4 | EXPL-04: per-root "⚠️ no covering tests found" warning fires correctly | ✓ VERIFIED | `render_markdown.go:283` emits the exact verbatim string; live smoke test on `ReconcileLedger` (2 callers, no test file) fires the warning; `reconcileCounts`/`Run` (which DO have `_test.go` coverage) correctly do NOT fire it |
| 5 | EXPL-05: `explore` output is byte-identical across CLI and MCP | ✓ VERIFIED | `internal/cli/explore.go` and `internal/mcp/tools.go` both call `eng.Explore(query, maxFiles)` on the shared `Engine`; `TestExploreCLIMatchesMCP` (drives `query.OpenAt` + a real in-process MCP server against the same on-disk index) passes for synthetic-parity and weft-go |
| 6 | NODE-01: `node` enumerates ALL exact-name defs, generated-files-last | ✓ VERIFIED | `enumerateSymbolDefs` (node.go:149-175) does a full `IterateNodes()` scan matching by Name, sorted `isGeneratedFile` (D-07 verbatim TS pattern list) then lowest-Id; live smoke test `node Run` on this repo returns "6 definitions named "Run"" |
| 7 | NODE-02: multi-def header + budget (≤16/12000) + overflow list | ✓ VERIFIED | `render_markdown.go`: `nodeMultiDefHardCap=16`, `nodeMultiDefBodyBudget=12000`; live smoke test shows "Returning 3 in full; 3 more listed below" for the 6-def `Run` case |
| 8 | NODE-03: optional file/line narrowing never empties the result set | ✓ VERIFIED (gap closed by e298e26) | `narrowNodeMatches` is now wired into `Engine.Node` (node.go:349) via a `line *int` param, exposed as CLI `--line`/`-l` and the MCP `codegraph_node` `line` param; live smoke tests confirm narrowing + never-empty fallback on bad line/file (exit 0, no error); `TestNodeLineHintNarrowsToSingleDef`, `TestNodeFileHintNeverEmptiesOnNoMatch`, `TestNodeLineHintCLIMatchesMCP` pass |
| 9 | NODE-04: single-def `node` output byte-comparable to TS (CLI+MCP) | ✓ VERIFIED | `TestNodeCLIMatchesMCP` passes for synthetic-parity (`Validate`, `AuditEntry`) and weft-go (`Run`); the `file != "" && line == nil` fast path preserves the untouched `resolveNodeForDetail` exact-match output for existing callers (node.go:329-333) |
| 10 | TEST-01: behavioral fixture harness diffs Go vs TS 1.3.1 goldens (D-02 oracle), CLI+MCP, and does NOT trivially always-pass | ✓ VERIFIED | `TestGoldenBehavioralSyntheticParity`/`TestGoldenBehavioralRealCorpora` run real assertions (header text, def-set equality, core-file membership) and log 4 explicitly documented divergences (AD-01..04) rather than silently passing; full suite `go test ./... && go test ./testdata/golden/...` green |
| 11 | D-09: overrides edge synthesis correctly attributes overridden methods, including overloaded methods (Java/C#) | ✓ VERIFIED (gap closed by b87b8cf) | `synthesizeOverrides` (`resolve.go:464-543`) now uses `map[string]map[string][]string` and checks each overload's arity independently on both self + supertype sides; `TestResolveOverrides_OverloadedMethodsBothSynthesized` asserts 0→2 distinct edges (no collapse, no cross-arity misattribution) — passes |

**Score:** 11/11 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/query/tokenize.go` | H1/H2 tokenizers | ✓ VERIFIED | present, wired into `Explore()`, unit-tested |
| `internal/query/rwr.go` | RWR core, 9-kind RankEdges | ✓ VERIFIED | present, wired, `rwrAlpha=0.25`/`rwrIterations=25` confirmed by reading the constants |
| `internal/query/gather.go` | H3-H9 gather channels + rerankers | ✓ VERIFIED | present, wired into `Explore()` orchestration |
| `internal/query/expand.go` | H10-H12 subgraph expansion | ✓ VERIFIED | present, wired |
| `internal/query/seeding.go` | H13 named-symbol seeding | ✓ VERIFIED | present, wired |
| `internal/query/scoring.go` | H14-H16 per-file scoring | ✓ VERIFIED | present, wired |
| `internal/query/explore_gate.go` | H17-H19 gate/central-file/5-tier sort | ✓ VERIFIED | present, wired; 5-way OR confirmed by direct read |
| `internal/query/node.go` | NODE-01/02/03 multi-def + narrowing | ✓ VERIFIED | multi-def enumeration/budget wired; `narrowNodeMatches` (NODE-03) now called from `Engine.Node` (node.go:349) — no longer orphaned |
| `internal/cli/node.go` | NODE-03 `--line`/`-l` flag | ✓ VERIFIED | `--line`/`-l` flag added (line 25), threaded to `Engine.Node` as `*int` lineHint (line 37-39) |
| `internal/mcp/tools.go` | NODE-03 `line` schema param | ✓ VERIFIED | `codegraph_node` schema exposes `line` (line 114); handler threads it as `lineHint` (line 182-194) |
| `internal/query/render_markdown.go` | EXPL-04 warning, H20/H21, multi-def render | ✓ VERIFIED | present, wired; verbatim warning string confirmed |
| `internal/indexer/resolve.go` | D-09 Pass-2 extends/overrides synthesis | ✓ VERIFIED | `extends` synthesis correct; `overrides` synthesis now overload-safe (gap 2 fix) |
| `internal/indexer/{go,java,csharp,py,ts}extract/` | D-09 Pass-1 emission (references/instantiates/returns/type_of) | ✓ VERIFIED | all 5 priority-4 language extractors emit all 4 new kinds (confirmed via grep + passing extractor unit tests, e.g. `TestExtract_TypeOf`) |
| `testdata/golden/gocapture/main.go` + `go-*.json` fixtures | F5 Go-side golden regeneration | ✓ VERIFIED | present; `TestGoSideFixturesRegenerated` passes |
| `testdata/golden/golden_parity_test.go` | D-02 behavioral harness + CLI==MCP | ✓ VERIFIED | present, all subtests pass (incl. `TestNodeLineHintCLIMatchesMCP`), divergences logged not swallowed |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `internal/cli/explore.go` | `Engine.Explore` | `eng.Explore(query, maxFiles)` | ✓ WIRED | confirmed by direct read |
| `internal/mcp/tools.go` (exploreHandler) | `Engine.Explore` | `eng.Explore(q, maxFiles)` | ✓ WIRED | confirmed by direct read |
| `internal/cli/node.go` | `Engine.Node` | `eng.Node(symbol, file, lineHint)` | ✓ WIRED | confirmed by direct read (now passes lineHint) |
| `internal/mcp/tools.go` (node case) | `Engine.Node` | `eng.Node(symbol, file, lineHint)` | ✓ WIRED | confirmed by direct read (now passes lineHint) |
| goextract/javaextract/csharpextract/pyextract/tsextract Pass-1 refs | `internal/indexer/resolve.go` Pass-2 switch | `UnresolvedRef{Kind}` -> `schema.Edge{Kind}` | ✓ WIRED | confirmed via grep across all 5 extractors + resolve.go case arms |
| `internal/query/node.go` `narrowNodeMatches` | `Engine.Node` → CLI `--line` / MCP `line` | NODE-03 wiring | ✓ WIRED | called from `Engine.Node` (node.go:349); reachable from both CLI and MCP surfaces |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| EXPL-01/02/04: multi-word explore, RWR ranking, no-covering-tests warning | `go run ./cmd/codegraph explore ReconcileLedger balance` (synthetic-parity corpus) | Returns 8 symbols/4 files; `ReconcileLedger` (structurally central, 2 callers) surfaces with `⚠️ no covering tests found`; ranking is not purely lexical | ✓ PASS |
| NODE-01/02: multi-def enumeration + budget/overflow | `go run ./cmd/codegraph node Run` (this repo's own index) | "6 definitions named "Run"" / "Returning 3 in full; 3 more listed below" | ✓ PASS |
| NODE-03: line narrowing + never-empty fallback | `node Run --line 92` / `node Run --line 999999` / `node Run -f nonexistent/nowhere.go` | 92 narrows 6→2; bad line falls back to nearest single def (exit 0); bad file substring falls back to full 6-def set (exit 0) — never empty, never errors | ✓ PASS |
| EXPL-05/NODE-04: CLI==MCP | `go test ./testdata/golden/... -run 'TestExploreCLIMatchesMCP|TestNodeCLIMatchesMCP|TestNodeLineHintCLIMatchesMCP' -v` | All subtests PASS | ✓ PASS |
| D-09 overrides: overloaded methods both synthesized | `go test ./internal/indexer/ -run TestResolveOverrides_OverloadedMethodsBothSynthesized -count=1` | PASS (0→2 distinct override edges, no misattribution) | ✓ PASS |
| TEST-01: harness genuinely diffs, doesn't always-pass | `go test ./testdata/golden/... -run TestGoldenBehavioral -v` | Real assertions pass; 4 divergences (AD-01..04) logged via `t.Logf`, not swallowed | ✓ PASS |
| D-09: all 9 RankEdges kinds present in a real graph | Custom read-only scan of this repo's `.codegraph/store` + `colbymchenry-codegraph` (per 01-15-SUMMARY.md, independently re-verified) | This repo: 7 of 9 kinds present; `overrides`/`type_of` legitimately zero (idiomatic all-Go source: no package-level typed vars, no override chains); `colbymchenry-codegraph` (multi-language) independently shows all 9 kinds nonzero | ✓ PASS |
| Full suite green | `go test ./... -count=1` | All 34 test packages pass, including `internal/daemon` (no flake this run) | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` convention or PLAN/SUMMARY-declared probes exist for this phase. SKIPPED (no runnable probe scripts — this phase's verification surface is `go test`, covered above).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| EXPL-01 | 01-03, 01-16 | Multi-word variadic query, tokenized | ✓ SATISFIED | see truth #1 |
| EXPL-02 | 01-02,05,06,07,08,09,10,11,12,13,14 | RWR graph-relevance ranking, 9 edge kinds | ✓ SATISFIED | see truth #2 (overrides input now correct, b87b8cf) |
| EXPL-03 | 01-10, 01-13, 01-14 | File-relevance gate | ✓ SATISFIED | see truth #3 |
| EXPL-04 | 01-16 | "no covering tests" warning | ✓ SATISFIED | see truth #4 |
| EXPL-05 | 01-17 | CLI==MCP byte-identity | ✓ SATISFIED | see truth #5 |
| NODE-01 | 01-04 | Multi-def enumeration | ✓ SATISFIED | see truth #6 |
| NODE-02 | 01-04 | Header + budget + overflow | ✓ SATISFIED | see truth #7 |
| NODE-03 | 01-04, 01-fix (e298e26) | File/line narrowing, never-empty | ✓ SATISFIED | see truth #8 — now reachable end-to-end from CLI + MCP |
| NODE-04 | 01-17 | Single-def byte-comparable | ✓ SATISFIED | see truth #9 |
| TEST-01 | 01-01, 01-17 | Behavioral fixture harness | ✓ SATISFIED | see truth #10 |

No orphaned requirements — REQUIREMENTS.md's Phase 1 mapping (EXPL-01..05, NODE-01..04, TEST-01) matches every PLAN's declared `requirements` field exactly. NODE-03 is now genuinely reachable, so REQUIREMENTS.md's `[x]` Complete marking is accurate.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/query/node.go` | 149-175 | `enumerateSymbolDefs` unbounded `IterateNodes()` scan, no ceiling before `HARD_CAP` render-time bound applies | ⚠️ Warning (deferred, WR-01) | Inconsistent with the phase's otherwise-consistent DoS-bounding discipline; not a correctness bug, worth a follow-up cap. Tracked as a non-blocking deferred item in `01-FIX-SUMMARY.md`. |
| `internal/query/node.go` | narrowNodeMatches tail | Dead/unreachable "never assign empty" fallback branch | ℹ️ Info (deferred, WR-02) | Harmless defensive redundancy. Tracked in `01-FIX-SUMMARY.md`. |
| `internal/indexer/*extract/` | D-09 reference-capture walks | Broad new `references`/`instantiates` emission with no index-time corpus cap | ℹ️ Info (deferred, WR-03) | Not a correctness bug; recommended follow-up profiling on a large monorepo. Tracked in `01-FIX-SUMMARY.md`. |
| `internal/query/gather.go` | 47-55, 75-79 | `channel1BaseScore`/`channel3BaseScore` documented as non-verbatim TS constants | ℹ️ Info | Transparently disclosed; candidate to re-verify if TS dist source becomes available again |

The two 🛑 Blocker anti-patterns from the initial verification (overload-collapsing `synthesizeOverrides` map, orphaned `narrowNodeMatches`) are both RESOLVED by the fix commits. No unresolved `TBD`/`FIXME`/`XXX` debt markers in any file the phase or its fixes modified. No `TODO`/`HACK`/`PLACEHOLDER` markers either.

### Accepted / Future-Work Divergences (non-blocking)

The 4 documented harness divergences (AD-01..04) surfaced by the TEST-01 behavioral harness remain accepted and tracked as future work, exactly as in the initial verification — they were always logged (not silently swallowed) and were never blockers:

- **AD-01** — stemming-precision divergence vs TS's undocumented `getStemVariants()` (colbymchenry-codegraph).
- **AD-02** — TS/JS object-literal method-shorthand extraction gap in `internal/indexer/tsextract` (the most concrete/actionable follow-up).
- **AD-03** — a 1-definition count delta for weft-go's `Run`.
- **AD-04** — file-selection breadth / blast-radius bullet-scope difference on synthetic-parity.

These live inline in `golden_parity_test.go` (AD-01..04) and in `01-17-SUMMARY.md` / `01-FIX-SUMMARY.md` for a future plan to pick up.

### Human Verification Required

None. Every must-have was verifiable programmatically (code reads, live CLI smoke tests, and the full green automated test suite).

### Gaps Summary

No gaps remain. Both blocking gaps from the initial verification (gaps_found, 9/11) are genuinely closed:

1. **NODE-03 is now shippable.** `narrowNodeMatches` is wired into `Engine.Node` and reachable via the new CLI `--line`/`-l` flag and MCP `codegraph_node` `line` param; live smoke tests confirm narrowing and never-empty fallback; NODE-04 byte-parity preserved via the exact-match fast path. Fixed in `e298e26`, regression-tested by `TestNodeLineHintNarrowsToSingleDef`/`TestNodeFileHintNeverEmptiesOnNoMatch`/`TestNodeLineHintCLIMatchesMCP`.
2. **D-09 `overrides` synthesis is overload-safe.** `synthesizeOverrides` no longer collapses same-name/different-arity overloads; each overload's arity is checked independently on both sides of the match. Fixed in `b87b8cf`, regression-tested by `TestResolveOverrides_OverloadedMethodsBothSynthesized` (0→2 edges).

Full suite `go test ./... -count=1` is green across all 34 test packages (including `internal/daemon` and `testdata/golden`), confirming no regression in the 9 previously-passing must-haves. The phase goal — TS-identical `explore` and `node` results, RWR-ranked exploration and full multi-definition disambiguation, proven equivalent on both CLI and MCP by a behavioral fixture harness that landed with the algorithm — is achieved.

---

*Verified: 2026-07-15T13:05:00Z (re-verification after gap closure)*
*Verifier: Claude (gsd-verifier)*
