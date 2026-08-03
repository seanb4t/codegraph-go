---
phase: 01-behavioral-parity-explore-node
reviewed: 2026-07-15T16:34:00Z
depth: deep
files_reviewed: 24
files_reviewed_list:
  - internal/cli/explore.go
  - internal/indexer/resolve.go
  - internal/indexer/goextract/goextract.go
  - internal/indexer/goextract/types.go
  - internal/indexer/javaextract/javaextract.go
  - internal/indexer/csharpextract/csharpextract.go
  - internal/indexer/pyextract/pyextract.go
  - internal/indexer/tsextract/tsextract.go
  - internal/query/rwr.go
  - internal/query/explore_gate.go
  - internal/query/expand.go
  - internal/query/gather.go
  - internal/query/scoring.go
  - internal/query/seeding.go
  - internal/query/tokenize.go
  - internal/query/node.go
  - internal/query/explore.go
  - internal/query/render_markdown.go
  - internal/cli/node.go
  - internal/mcp/tools.go
  - internal/indexer/goextract/goextract_test.go
  - internal/indexer/resolve_test.go
  - internal/query/node_test.go
  - internal/query/rwr_test.go
findings:
  critical: 2
  warning: 3
  info: 2
  total: 7
status: issues_found
---

# Phase 01: Code Review Report — Behavioral Parity (explore & node)

**Reviewed:** 2026-07-15T16:34:00Z
**Depth:** deep
**Files Reviewed:** 24 (of 37 changed; test-only fixture/golden files and the 4 `resolution_test.go`/`d09_test.go` per-language test files were skimmed but not separately enumerated in `files_reviewed_list`)
**Status:** issues_found

## Summary

This is a large (~9,600 line) port of TS CodeGraph 1.3.1's `explore`/`node` heuristic pipeline. The load-bearing algorithmic core — `rwr.go` (RWR power iteration), `explore_gate.go` (the 5-way relevance gate / 5-tier sort / central-file selection), `expand.go` (H10-H12 subgraph bounding + DoS caps), `gather.go`/`scoring.go`/`seeding.go` (H3-H16), and `tokenize.go` (WR-05 empty-query guard) — is well-constructed: determinism is handled correctly throughout (sorted map iteration before every walk, score-desc/Id-asc tie-breaks, fixed 25-iteration RWR with no early exit, score rounding to 1e-9 before comparison), and the DoS caps (`ExpandMaxNodes=200`, `GlueNodeCap=60`, NODE-02's `HARD_CAP=16`/`BODY_BUDGET=12000`) are genuinely wired into `Explore()`'s real call path, not just present as unused constants. Path-traversal defenses in `node.go` (`resolveSourcePath`) correctly re-verify confinement after `EvalSymlinks`, closing the TOCTOU-style symlink escape a naive string-only check would miss.

Two real defects were found by tracing behavior rather than reading for style, both matching the pattern this review was primed to look for (a green TDD suite validating an isolated unit while missing an integration/logic gap):

1. A genuine **correctness bug** in the new D-09 `overrides` edge synthesis (`resolve.go`): method name→id maps used to find a supertype override collapse Java/C#-style method overloads (same name, different arity) to a single arbitrary candidate, silently dropping or misattributing override edges — and `overrides` is one of the exact `RankEdges` kinds RWR walks, so this can skew ranking exactly as the review brief warned.
2. NODE-03 (file/line narrowing for `node`'s multi-def enumeration) is implemented and unit-tested as a pure function, but was **never wired into any reachable surface** — `internal/cli/node.go` has no `--line` flag and `internal/query/node.go`'s only file-disambiguation path (`resolveNodeForDetail`) does exact `FilePath` equality, not TS's `endsWith`/`includes` substring match. `narrowNodeMatches` is dead code from every entry point except its own unit test, despite the phase's `01-04-SUMMARY.md` listing `requirements-completed: [..., NODE-03, ...]`.

## Critical Issues

### CR-01: D-09 `overrides` synthesis collapses overloaded methods, silently dropping/misattributing RANK_EDGES `overrides` edges

**File:** `internal/indexer/resolve.go:464-526` (bug at line 479, consumed at lines 496-514)

**Issue:** `synthesizeOverrides` builds `typeMethods[typeID] map[string]string` — methodName -> methodID — by iterating every `"contains"` edge and unconditionally overwriting on name collision:

```go
// line 470-479
case "contains":
    if nodeKindByID[e.Target] != goextract.KindMethod {
        continue
    }
    methods := typeMethods[e.Source]
    if methods == nil {
        methods = make(map[string]string)
        typeMethods[e.Source] = methods
    }
    methods[nodeNameByID[e.Target]] = e.Target
```

For a Java or C# type with two overloaded methods sharing a name (e.g. `void Foo(int x)` and `void Foo(String s)` — both extractors emit two distinct `"contains"` edges type→method for these, since overloading is normal in both languages), this map keeps only ONE of the two method ids per name — whichever happens to be processed last in `edges`' append order. The discarded overload:

- is never considered as a candidate for its own override match against a same-named/same-arity supertype method (a real override edge silently never gets synthesized), **and**
- if the *surviving* overload doesn't actually match the supertype's arity, no edge is synthesized at all even though the *other* overload (now invisible) would have matched.

The same collapsed map is also consulted on the **supertype side** (`typeMethods[cur][name]`, line ~509), so a supertype's own overloaded methods are equally at risk of being reduced to one arbitrary candidate — an override can be missed even when both types individually have only the "wrong" overload survive the collision.

This is a genuine correctness regression in a newly-added heuristic, not a pre-existing limitation being inherited: `EdgeKindOverrides` is one of the exact 9 kinds in `query/rwr.go`'s `RankEdges` set (rwr.go:21-31), so a dropped or misattributed `overrides` edge changes RWR's adjacency and therefore its relevance scores — precisely the failure mode flagged as highest-risk in this review's brief ("Any wrong edge could skew RWR ranking").

No test exercises this: `internal/indexer/javaextract/d09_test.go` and `internal/indexer/csharpextract/d09_test.go` contain no "overload" test, and `resolve_test.go`'s two `TestResolveOverrides_*` cases each use a type with exactly one method per name (`TestResolveOverrides_ArityMismatchNoEdge` tests a *cross-type* name+arity mismatch with a single method on each side — it does not exercise same-type name collision).

**Fix:** Track a slice of `(methodID, arity)` per name instead of a single `methodID`, and check every same-named candidate on both the "self" and "supertype" side against every arity, e.g.:

```go
typeMethods := make(map[string]map[string][]string) // typeID -> methodName -> []methodID (all overloads)
...
methods[nodeNameByID[e.Target]] = append(methods[nodeNameByID[e.Target]], e.Target)
...
// when walking supertypes, check every candidate id in typeMethods[cur][name] against arity,
// not just typeMethods[cur][name] as a lone string.
```

---

### CR-02: NODE-03 (file/line narrowing) is implemented and unit-tested but unreachable from any CLI/MCP entry point

**File:** `internal/query/node.go:192-257` (`narrowNodeMatches`, unused outside its own test); `internal/cli/node.go:18-55`; `internal/mcp/tools.go:107-115`

**Issue:** `narrowNodeMatches` — the RESEARCH §9-cited, never-empty file/line narrowing function for `node`'s multi-def enumeration (`mcp/tools.js:3603-3620` in the TS original) — exists, is documented in detail, and is unit-tested (`node_test.go:TestNarrowNeverEmpty`). But it is called from nowhere in production code:

```
$ rg -n "narrowNodeMatches" internal/query internal/cli internal/mcp
internal/query/node.go:177/192/204   (definition + doc comments)
internal/query/node_test.go:214-253  (only caller)
```

`internal/cli/node.go`'s `--file`/`-f` flag and `internal/mcp/tools.go`'s `codegraph_node` tool schema both expose only `symbol` and `file` (exact-match) parameters — no `--line`/`line` parameter exists anywhere in the CLI or MCP surface. `Engine.Node`'s only file-based disambiguation path is `resolveNodeForDetail` (node.go:267-294), which does an **exact** `n.FilePath == file` string match — not TS's `endsWith(fh) || includes(fh)` substring semantics `narrowNodeMatches` actually implements. So even the "file" half of NODE-03 (not just the "line" half) is not reachable: a user who passes a partial/relative path that isn't byte-identical to the stored `FilePath` gets "not found" instead of TS's substring-narrowed match.

This was transparently documented as deferred in `01-04-SUMMARY.md` ("narrowNodeMatches (NODE-03) is implemented and independently tested but not reachable through the current public Engine.Node(symbol, file string) signature... A future plan adding a line/refined file flag can call it directly"). However, no later plan in this phase (`01-05` through `01-17`, confirmed via `rg` across the phase's CLI/MCP files) ever added that wiring, and `01-04-SUMMARY.md`'s own frontmatter still lists `requirements-completed: [NODE-01, NODE-02, NODE-03, NODE-04]` — a reader trusting that summary (or the phase's requirements tracker) would believe NODE-03 ships end-to-end when it does not. Given this phase's explicit charter is "Behavioral Parity — explore & node," and NODE-03 is one of its four named `node` requirements, this is a real parity gap, not a cosmetic one.

**Fix:** Either (a) add a `--line`/`line` parameter to `internal/cli/node.go` and `internal/mcp/tools.go`'s `codegraph_node` schema, thread it through `Engine.Node`, and switch `resolveNodeForDetail`'s file match (or a NODE-01 multi-match path) to call `narrowNodeMatches` instead of/in addition to the exact match — or (b) if intentionally deferred to a later phase, correct `01-04-SUMMARY.md`'s `requirements-completed` list and any downstream ROADMAP/requirements tracker to reflect that NODE-03 is only function-level-complete, not user-reachable, so this phase isn't marked as having shipped full `node` behavioral parity.

## Warnings

### WR-01: `enumerateSymbolDefs` / NODE-01 multi-def enumeration has no upper bound on the initial scan or matched set before the render-time HARD_CAP applies

**File:** `internal/query/node.go:149-175`

**Issue:** `enumerateSymbolDefs` does a full unbounded `IterateNodes()` scan and appends every node whose `Name == symbol` to `matches`, with no cap. For a very common symbol name in a large monorepo (e.g. a name like `New`, `Validate`, `Handler` that could legitimately appear hundreds of times), `matches` could grow arbitrarily large before `RenderNodeMultiDef`'s `HARD_CAP`/`BODY_BUDGET` bound the *rendered* output. The listing tail (`nodeMultiDefListCap=20` + "+K more") does bound the final markdown size, but the intermediate `[]*schema.Node` slice, the `sort.SliceStable` over it, and (in the multi-def render path) the `**Other definitions**` list construction all pay O(matches) cost with no ceiling — a difference in degree from the deliberate DoS-bounding discipline applied everywhere else in this phase (`ExpandMaxNodes`, `GlueNodeCap`, `groupMatchesByFile`'s `maxFiles` cap, NODE-02's own HARD_CAP/BODY_BUDGET). This is lower severity than CR-01/CR-02 since it does not corrupt output correctness and the actual rendered output is still bounded — but it's a gap in an otherwise very consistently-applied bounding discipline, worth a documented cap (e.g. same pattern as `groupMatchesByFile`: stop scanning once some generous ceiling like a few hundred matches is hit) rather than an unbounded full-graph scan.

**Fix:** Consider capping the scan (e.g. break out of `IterateNodes()` once `len(matches)` exceeds a documented ceiling well above `nodeMultiDefHardCap`, mirroring the "cap, don't truncate silently" discipline used elsewhere in this phase), or explicitly document why an unbounded scan here is acceptable (e.g. "index size is already bounded upstream by X").

### WR-02: `narrowNodeMatches`'s final "never assign empty" guard is unreachable dead code

**File:** `internal/query/node.go:253-256`

**Issue:**

```go
if len(narrowed) > 0 {
    return narrowed // final guard: never assign an empty working set
}
return matches
```

By construction, every branch above this guarantees `narrowed` is non-empty by the time this is reached: the `fileHint` branch only reassigns `narrowed = byFile` when `len(byFile) > 0`; the `lineHint` branch always leaves `narrowed` as either a non-empty `containing` slice or a single-element `[]*schema.Node{nearest}`. So `len(narrowed) > 0` is always true here and the `return matches` fallback is unreachable. Harmless (defensive redundancy, not a bug), but worth trimming or converting to an invariant-asserting comment so a future edit to the branches above doesn't silently rely on dead code as a safety net that no longer fires.

**Fix:** Either remove the redundant guard (trusting the invariant already established above), or add a comment/assertion making explicit that this is deliberately-defensive dead code, not a load-bearing fallback.

### WR-03: `emit ... goextract.RefKindInstantiates` D-09 reference-capture walks are a substantial new source of unresolved-ref volume with no batch/scan cap

**File:** `internal/indexer/goextract/goextract.go:784-1040` (`collectReferencesAndInstantiates`, `captureExprRead`, and language-equivalent walks in `javaextract.go`/`csharpextract.go`/`pyextract.go`/`tsextract.go`)

**Issue:** The new D-09 `references`/`instantiates` capture walks emit an `UnresolvedRef` for essentially every identifier/selector read reachable from a bounded-but-still-broad allow-list of read positions (call arguments, return values, RHS of assignment/declaration, composite-literal elements, recursively through common expression wrappers) across every function/method body in the indexed corpus. This is a deliberate, well-documented scope decision (each extractor's doc comment explains the allow-list rationale), and it's Pass-1 extraction volume rather than a query-time DoS surface (the query-side caps reviewed above — `ExpandMaxNodes`, `GlueNodeCap` — do bound what RWR/explore ever walks at query time). Still, this is a meaningful, unbounded-per-file increase in `UnresolvedRef`/edge volume with no corpus-size safety valve at index time, which could materially increase index build time/storage on a very large monorepo. Not a correctness bug, but worth flagging since it's a new, broad surface introduced in this phase across 5 language extractors simultaneously.

**Fix:** No action required for phase correctness; recommend a follow-up profiling pass against a large real-world monorepo to confirm index-time cost/graph size growth from the new `references`/`instantiates`/`returns`/`type_of` capture is acceptable before this ships broadly, given it multiplies edge volume across every supported language at once.

## Info

### IN-01: `resolveNodeForDetail`'s exact-match `file` semantics diverge from TS's substring semantics without being flagged as a divergence at the call site

**File:** `internal/query/node.go:267-294`

**Issue:** This was a deliberate, documented decision (`01-04-SUMMARY.md`: "Kept resolveNodeForDetail's file != "" behavior completely untouched... NODE-04 explicitly requires 'keep the existing single-def resolveNodeForDetail path intact'"), and is reasonable given NODE-04's explicit constraint. Flagging only so it's visible alongside CR-02: the combination of "exact match here" + "substring match implemented but unwired in narrowNodeMatches" means TS's actual `-f`/fileHint behavior is not available anywhere in this codebase yet, which is easy to lose track of once CR-02 above is eventually fixed piecemeal.

### IN-02: `channel1BaseScore`/`channel3BaseScore` (10.0) are documented as "not verbatim, chosen for cross-channel consistency" — worth a tracking note

**File:** `internal/query/gather.go:47-55, 75-79`

**Issue:** Both constants are explicitly flagged in their own doc comments as *not* cited verbatim from the (currently unreadable) TS dist source — they're this port's own documented default, chosen only for "cross-channel scale consistency." This is transparently disclosed (not a hidden assumption) and is consistent with this phase's broader "document every D-02 divergence" discipline, so it's not a defect. Noting it here only so it's tracked as a candidate to re-verify if/when the TS 1.3.1 dist source becomes readable again, since an incorrect base score changes every downstream ranking decision this phase's tests golden-pin against a *self-consistent* (not necessarily TS-byte-identical) baseline.

---

_Reviewed: 2026-07-15T16:34:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
