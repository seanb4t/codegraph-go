---
phase: 01-behavioral-parity-explore-node
fixed_at: 2026-07-15T13:30:00Z
review_path: .planning/phases/01-behavioral-parity-explore-node/01-REVIEW.md
verification_path: .planning/phases/01-behavioral-parity-explore-node/01-VERIFICATION.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 1: Behavioral Parity — explore & node Fix Summary

**Fixed at:** 2026-07-15T13:30:00Z
**Source review:** `.planning/phases/01-behavioral-parity-explore-node/01-REVIEW.md` (CR-01, CR-02)
**Source verification:** `.planning/phases/01-behavioral-parity-explore-node/01-VERIFICATION.md` (9/11, `gaps_found`)
**Iteration:** 1

**Summary:** both phase-goal-blocking gaps from the code review and independent verification are fixed, each with a regression test confirmed to FAIL against the pre-fix code and PASS after. `go build`, `go vet`, and `go test` are green across `internal/query`, `internal/indexer`, `internal/cli`, `internal/mcp`, and `testdata/golden`.

## Fixed Issues

### CR-01: D-09 `overrides` synthesis collapsed overloaded methods

**Files modified:** `internal/indexer/resolve.go`, `internal/indexer/resolve_test.go`
**Commit:** `b87b8cf`

**Applied fix:** `synthesizeOverrides`'s `typeMethods` map changed from `map[string]map[string]string` (methodName -> ONE methodID, overwritten on collision) to `map[string]map[string][]string` (methodName -> ALL overload methodIDs). Every overload on the "self" side is now checked against every same-named candidate's arity on the "supertype" side (previously only one arbitrary collapsed candidate was checked on both sides). Determinism preserved: type ids, method names, AND each name's overload-id slice are all sorted before walking, so the emitted edge order never depends on Go map iteration or `edges`' append order.

**Regression test:** `TestResolveOverrides_OverloadedMethodsBothSynthesized` calls `synthesizeOverrides` directly with hand-built edges (a `Base` type declaring two `Foo` overloads, arity 1 and 2, and a `Derived` type overriding both) — Go itself can't produce overloaded methods, so this can't be driven through the Go extractor the way `TestResolveOverrides_StructMethodOverridesSupertypeMethod` is. Edge order is deliberately arranged so the pre-fix collapsing map keeps a *mismatched* arity pair on each side. Confirmed **FAIL before** (0 edges synthesized instead of 2 — the bug isn't just "misses one edge," it can silently drop all of them) and **PASS after**.

### CR-02: NODE-03 (file/line narrowing) wired end-to-end (CLI + MCP)

**Files modified:** `internal/query/node.go`, `internal/cli/node.go`, `internal/mcp/tools.go`, `internal/query/node_test.go`, `internal/query/render_markdown_test.go`, `testdata/golden/golden_parity_test.go`, `testdata/golden/gocapture/main.go`
**Commit:** `e298e26`

**Applied fix:**
- `Engine.Node(symbol, file string, line *int)` gained an optional `line` narrowing hint. A `nil` line with a non-empty `file` still tries the pre-fix exact-match `resolveNodeForDetail` path *first* and returns immediately on success — every existing exact-match caller (golden fixtures, NODE-04) gets byte-identical output. Otherwise the full `enumerateSymbolDefs` set is narrowed via the existing (previously orphaned) `narrowNodeMatches` — substring file match + `[StartLine,EndLine]` line containment, never emptying the set — before rendering single- or multi-def.
- `internal/cli/node.go` gained a `--line`/`-l` int flag (0 = unset, matching narrowNodeMatches' own "0 means unset" convention already used for `EndLine`).
- `internal/mcp/tools.go`'s `codegraph_node` tool schema gained a `line` number parameter, delegating to the same `Engine.Node` (EXPL-05/NODE-04 CLI==MCP identity preserved for the new parameter, not just the pre-existing one).
- D-07 preserved: narrowing remains a pure in-memory filter over already-resolved nodes; no new disk read keyed on the raw hint.
- Every existing call site of `Engine.Node` (test files across `internal/query`, `testdata/golden`) updated to pass a third `nil`/pointer argument; no behavioral changes to those callers.

**Regression tests:**
- `TestNodeLineHintNarrowsToSingleDef` / `TestNodeFileHintNeverEmptiesOnNoMatch` (`internal/query/render_markdown_test.go`) — prove `Engine.Node`'s public surface now reaches `narrowNodeMatches`: a line hint containing exactly one of three candidates narrows to a single-def render; a file hint matching none of three candidates falls back to the full 3-def set (never empty, never errors).
- `TestNodeLineHintCLIMatchesMCP` (`testdata/golden/golden_parity_test.go`) — builds a real indexed fixture with two same-named defs at distinct lines, drives both the CLI-facing `Engine.Node` call and the MCP-facing `codegraph_node` call with the same `line` hint, and asserts (a) the hint narrowed to a single def and (b) CLI and MCP output are byte-identical.
- All three confirmed **FAIL** against a temporarily-reverted pre-fix `Engine.Node` body (line ignored, file exact-match-only — reproducing "narrowNodeMatches has zero production callers") and **PASS** after restoring the fix.

**Live smoke test** (`go run ./cmd/codegraph node <symbol> --line <n>` / `--file <hint>` against this repo's own index):
- `node Run` (baseline): 6 definitions, 3 shown.
- `node Run --line 92`: narrows to 2 definitions (both containing line 92) — reachable, never empty.
- `node Run --line 999999` (no def contains it): falls back to the single *nearest* def — never empty, never errors.
- `node Run --file nope/nowhere.go` (matches nothing): falls back to the full 6-definition set — never empty, never errors.

## Verification

- `go build ./...` — clean.
- `go vet ./internal/query/... ./internal/indexer/... ./internal/cli/... ./internal/mcp/...` — clean.
- `go test ./internal/query/... ./internal/indexer/... ./internal/cli/... ./internal/mcp/... ./testdata/golden/... -count=1` — all green.
- `gofmt -l` on every modified file — clean.

## Deferred (non-blocking, from 01-REVIEW.md)

Per the fix scope, WR-01/WR-02/WR-03 are **not** addressed in this pass — none of them block the phase goal (they're a follow-up DoS-bounding hardening item, a harmless dead-code trim, and a perf/scale profiling recommendation, respectively):

- **WR-01** (`internal/query/node.go:149-175`): `enumerateSymbolDefs`'s `IterateNodes()` scan has no upper bound before NODE-02's render-time `HARD_CAP` applies. Follow-up: cap the scan at a documented ceiling, mirroring `groupMatchesByFile`'s pattern.
- **WR-02** (`internal/query/node.go:253-256`): `narrowNodeMatches`'s final "never assign empty" guard is unreachable dead code (every branch above it already guarantees non-empty). Harmless; follow-up: trim or convert to an invariant comment.
- **WR-03** (`internal/indexer/goextract/goextract.go:784-1040` + language-equivalent walks): the new D-09 `references`/`instantiates` capture walks have no index-time batch/scan cap across a large monorepo. Follow-up: profile index-time cost/graph-size growth on a large real-world corpus.

---

_Fixed: 2026-07-15T13:30:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
