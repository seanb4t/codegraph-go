# Phase 1: Behavioral Parity — explore & node - Pattern Map

**Mapped:** 2026-07-14
**Files analyzed:** 14 (new + modified)
**Analogs found:** 14 / 14

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `internal/query/tokenize.go` (new) | utility | transform | `internal/query/search.go` (`lexicalMatchTier`) | role-match |
| `internal/query/rwr.go` (new) | service | batch/transform | `internal/query/traverse.go` (`BuildReverseAdjacency`, `Impact`'s BFS) | role-match |
| `internal/query/explore_gate.go` (new) | service | transform | `internal/query/explore.go` (`groupMatchesByFile`) | role-match |
| `internal/query/node.go` (extend) | service | CRUD (read) | itself — `resolveNodeForDetail`/`Node` (existing) | exact |
| `internal/query/render_markdown.go` (extend) | utility (renderer) | transform | itself — `RenderNode`/`RenderExplore` (existing) | exact |
| `internal/query/explore.go` (extend) | service | request-response | itself — `Explore` (existing) | exact |
| `internal/cli/explore.go` (modify: `cobra.ExactArgs(1)` → variadic) | controller (CLI) | request-response | `internal/cli/node.go` (`cobra.MaximumNArgs(1)` variadic-arg pattern) | exact |
| `internal/indexer/goextract/types.go` (extend: 6 new `RefKind*`/edge-kind consts) | model/config | CRUD | itself — existing `RefKind*`/`EdgeKindImplements` block | exact |
| `internal/indexer/resolve.go` (extend: new `case` arms + Pass-2 synthesis) | service (extraction pipeline) | batch/event-driven | itself — `RefKindEmbeds`→`implements` promotion branch (lines 116-150), `synthesizeGoImplements` (lines 326-381) | exact |
| `internal/indexer/{javaextract,csharpextract,pyextract,tsextract}/*.go` (extend) | service (extraction pipeline) | event-driven | `internal/indexer/goextract/*.go` — the per-language mirror-of-goextract's-shape convention | role-match |
| `internal/indexer/routes/*.go` (reference only, no edit) | service (synthesis) | event-driven | n/a — cited as the synthesized-edge/heuristic-provenance precedent | exact |
| `internal/mcp/tools.go` (no change expected) | controller (MCP) | request-response | itself — `exploreHandler`/`companionHandler["node"]` already delegate to `Engine.Explore`/`Engine.Node` | exact |
| `testdata/golden/capture.sh` (extend) | test (fixture capture) | batch | itself — `capture_repo` function | exact |
| `testdata/golden/golden_test.go` / `golden_parity_test.go` (extend) | test | request-response (diff) | itself — existing diff harness | exact |
| `testdata/golden/corpus/synthetic-parity/` (new fixture set) | test fixture | batch | `testdata/golden/corpus/weft-go/{explore,node}.json` | role-match |

## Pattern Assignments

### `internal/query/tokenize.go` (new) — utility, transform

**Analog:** `internal/query/search.go` (`lexicalMatchTier`, `matchNodes`) for file/package conventions; TS source (RESEARCH §1-2) for the actual algorithm to port.

**Imports pattern** (mirror `search.go:1-10`):
```go
package query

import (
	"regexp"
	"strings"
	"unicode"
)
```
No `schema` import needed — tokenizers operate on plain strings only.

**Core pattern — two distinct pure functions, do not conflate** (RESEARCH Pitfall "Conflating the two tokenizers"):
```go
// extractSymbolsFromQuery mirrors search.go's exported-boundary-per-func
// convention (small, single-purpose funcs) — port TS's identifier
// extraction (CamelCase/snake/SCREAMING/acronym/dot-notation/plain),
// filtered by the ~90-word commonWords set (RESEARCH §1).
func extractSymbolsFromQuery(query string) []string { ... }

// extractSearchTerms mirrors STOP_WORDS filtering (RESEARCH §2) — EXPL-01's
// literal "stopword-filtered" target.
func extractSearchTerms(query string) []string { ... }
```

**Determinism note:** both funcs must return in a stable, reproducible order (D-04) — use an ordered slice build (append in scan order), not map iteration, exactly like `matchNodes`'s `sort.SliceStable` discipline in `search.go:98-106`.

---

### `internal/query/rwr.go` (new) — service, batch/transform

**Analog:** `internal/query/traverse.go` — `BuildReverseAdjacency` (fresh-per-call scan discipline) + `Impact`'s bounded BFS (deterministic frontier expansion, WR-04 dangling-edge skip).

**Imports pattern** (mirror `traverse.go:1-13`):
```go
package query

import (
	"errors"
	"sort"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)
```

**Fresh-per-call, no cache — copy `BuildReverseAdjacency`'s doc-comment discipline verbatim** (`traverse.go:15-51`):
```go
// computeGraphRelevance builds an in-memory undirected adjacency from
// edges whose Kind is in RankEdges, then runs FIXED 25-iteration
// power-iteration RWR (α=0.25, D-04) restarting to a uniform
// distribution over seedIDs present in the candidate set. Built fresh
// every call — no package-level cache (mirrors BuildReverseAdjacency's
// RESEARCH Pitfall 2 discipline: a long-lived MCP process must never
// serve a stale point-in-time graph view).
func computeGraphRelevance(nodeIDs []string, edges []*schema.Edge, seedIDs map[string]bool) map[string]float64 { ... }
```

**WR-04 dangling-edge tolerance** — mirror `BuildReverseAdjacency`'s pattern of skipping edges pointing to unknown nodes rather than failing the whole call:
```go
// from traverse.go:41-46, the pattern to mirror inside adjacency build:
for it.Next() {
	e := it.Edge()
	if !RankEdges[e.Kind] {
		continue
	}
	// map e.Source/e.Target through idx; skip (don't error) if either id
	// is absent from the candidate set — RESEARCH's WR-04 convention.
}
```

**RANK_EDGES set — 9 kinds after D-09** (RESEARCH §C.1):
```go
// RankEdges is the Go RANK_EDGES-equivalent set — undirected, UNWEIGHTED
// membership only (mirrors goextract's "one definition not three copies"
// discipline — reference goextract.RefKind*/EdgeKindImplements consts,
// do not redeclare the literals here).
var RankEdges = map[string]bool{
	goextract.RefKindCalls:        true,
	RefKindReferences:             true, // new
	RefKindExtends:                true, // new
	goextract.EdgeKindImplements:  true,
	RefKindOverrides:              true, // new
	RefKindInstantiates:           true, // new
	RefKindReturns:                true, // new
	RefKindTypeOf:                 true, // new
	goextract.RefKindImports:      true,
}
```

**Deterministic tie-break** — reuse the codebase-wide lowest-`Id` convention (`traverse.go:194-218`, `resolveSymbolNode`) for any RWR score tie, per D-04.

---

### `internal/query/explore_gate.go` (new) — service, transform

**Analog:** `internal/query/explore.go` (`groupMatchesByFile`'s maxFiles-cap pattern) for file-scoped aggregation shape; `traverse.go`'s `isTestSymbol` for the test-file heuristic to extend into TS's path-based `isTestFile`.

**Core pattern — 5-way OR gate, never a bare threshold** (RESEARCH §4/Pitfall "Assuming the gate is a bare threshold"):
```go
// fileRelevanceGate ports EXPL-03's 5-way OR (RESEARCH Code Examples §4):
// graph-mass >= 0.06*max, OR central, OR entry/named file, OR
// change-surface-rescued, OR >=2 distinct term hits — guarded so it never
// prunes below 2 files (mirrors groupMatchesByFile's cap-not-truncate
// discipline in explore.go:40-64).
func fileRelevanceGate(files []fileCandidate, maxGraphMass float64) []fileCandidate { ... }
```

**isTestFile (path-based, NOT `isTestSymbol`'s name-based heuristic)** — extend, do not reuse, `traverse.go:467-476`'s `isTestSymbol`:
```go
// traverse.go:467-476 — existing SYMBOL-name heuristic, reused by
// EXPL-04's warning but NOT sufficient for EXPL-03's file-level gate,
// which needs TS's broader isTestFile path predicate (RESEARCH §5:
// test_*/*_test.*/*.test.*/*-spec.*/*Test.ext, /tests?//__tests__//specs?/,
// plus non-production dirs). Port isTestFile as a SEPARATE function in
// explore_gate.go; do not widen isTestSymbol's existing contract.
func isTestFile(filePath string) bool { ... }
```

---

### `internal/query/node.go` (extend) — service, CRUD (read)

**Analog:** itself — `resolveNodeForDetail` (existing single-def path, `node.go:91-126`) is the base NODE-01 multi-def enumeration extends.

**Existing narrowing pattern to extend, not replace** (`node.go:99-126`):
```go
func (e *Engine) resolveNodeForDetail(symbol, file string) (*schema.Node, error) {
	if file == "" {
		return e.resolveSymbolNode(symbol)   // <- traverse.go:194-218, D-03 full-scan + lowest-Id
	}
	// ... existing single-candidate scan; NODE-01 extends this to return
	// ALL matches (not just the lowest-Id one) when file/line hints are
	// absent, applying isGeneratedFile-then-lowest-Id sort (D-07) instead
	// of a single winner.
}
```

**Generated-files-last sort** (RESEARCH §7, D-07) — new `isGeneratedFile` predicate (verbatim regex list), PRIMARY sort key, with the existing lowest-`Id` convention as secondary tie-break (documented divergence per RESEARCH Pattern 2):
```go
sort.SliceStable(matches, func(i, j int) bool {
	gi, gj := isGeneratedFile(matches[i].FilePath), isGeneratedFile(matches[j].FilePath)
	if gi != gj {
		return !gi // non-generated first
	}
	return false // stable: preserve scan order; true secondary tie-break
	// (lowest-Id) applied only when scan order isn't already deterministic —
	// mirror resolveSymbolNode's `n.Id < best.Id` comparison, node.go:207.
})
```

**Never-empty narrowing guard** (RESEARCH §9, NODE-03) — mirror the existing `resolveNodeForDetail`'s "only replaces if non-empty" discipline already present at `node.go:120` (`if len(candidates) == 0 { return nil, fmt.Errorf(...) }`) — extend to a filter-then-fallback chain that never assigns an empty slice to the working set, exactly as TS's `narrowed.length > 0` guard (§9) requires.

**WR-04 / edge-read pattern already established** — `node.go:149-171` (`Calls`) and `node.go:176-191` (`CalledBy`, via `BuildReverseAdjacency`) both use the `errors.Is(err, graphstore.ErrNotFound) { continue }` skip — reuse verbatim for any new edge-kind reads the multi-def path needs.

---

### `internal/query/render_markdown.go` (extend) — utility renderer, transform

**Analog:** itself — `RenderNode` (`render_markdown.go:87-96`) and `RenderExplore` (`render_markdown.go:138-156`).

**Two-line multi-def header — exact structure, RESEARCH Pitfall 4** (do NOT combine into one sentence):
```go
// Mirrors RenderNode's fmt.Fprintf-per-line, strings.Builder accumulation
// style (render_markdown.go:87-96). New RenderNodeMultiDef (or an
// extended RenderNode signature) must emit TWO separate lines joined by
// a blank line, per RESEARCH §8:
fmt.Fprintf(&b, "**%d definitions named %q**\n\n", len(matches), symbol)
fmt.Fprintf(&b, "Returning %d in full%s — pick the one you need (no Read required).\n\n",
	len(rendered), overflowClause)
```

**Plain-text constraint** — `render_markdown.go` has zero ANSI/styling today (confirmed: only `**bold**`/backtick markdown, no color codes) — keep it that way (Phase 6 archtest will enforce later, per CONTEXT hard constraint).

**"No covering tests" warning — extend `renderBlastBullet`** (`render_markdown.go:105-117`):
```go
// Existing pattern already has the "; tests: ..." clause (lines 109-115)
// for WHEN test files exist. EXPL-04 adds the else-branch: when
// bl.CallerCount > 0 AND bl.TestFiles is empty, append the EXACT string
// "; ⚠️ no covering tests found" (RESEARCH §5 — verbatim, note "found").
// Trigger is CallerCount > 0 — a root with zero callers gets NEITHER
// clause (mirrors TS's early-continue before this block).
if bl.CallerCount > 0 && len(bl.TestFiles) == 0 {
	s += "; ⚠️ no covering tests found"
} else if len(bl.TestFiles) > 0 {
	// existing tests: clause
}
```

---

### `internal/query/explore.go` (extend) — service, request-response

**Analog:** itself — `Explore` (`explore.go:105-159`) is the orchestrator the RWR pipeline plugs into; `groupMatchesByFile`/`buildBlastEntry`/`readSourceFile` are KEPT, not replaced.

**Swap point — NOT a rerank of `matchNodes`'s output** (RESEARCH Pitfall 1, confirmed via `explore.go:114` `ranked, err := e.matchNodes(query, "")`): the new hybrid-gather + RWR pipeline replaces this call site's INPUT construction, but `groupMatchesByFile`/`buildBlastEntry` downstream stay as-is per D-05's discretion note ("extend, don't replace" in the recommended project structure).

**WR-05 empty-query guard** — reuse verbatim (`explore.go:106-108`):
```go
if strings.TrimSpace(query) == "" {
	return "", fmt.Errorf("query: explore query must not be empty")
}
```

**maxFiles validate+clamp** — reuse verbatim (`explore.go:109-112`).

---

### `internal/cli/explore.go` (modify)

**Analog:** `internal/cli/node.go` (`node.go:24`, `cobra.MaximumNArgs(1)`) — the variadic-arg pattern already exists in the sibling command.

**Exact fix** (RESEARCH §10, confirmed bug):
```go
// Before (explore.go:24): Args:  cobra.ExactArgs(1),
// After (EXPL-01):
Args: cobra.MinimumNArgs(1),
// ... inside RunE, before eng.Explore call:
query := strings.Join(args, " ")
out, err := eng.Explore(query, maxFiles)
```
Add `"strings"` to the existing import block (`explore.go:3-9`).

---

### `internal/indexer/goextract/types.go` (extend) — model/config, CRUD

**Analog:** itself — the existing `RefKind*`/`EdgeKindImplements` const block (`types.go:32-50`).

**Exact pattern to mirror — "one definition shared by all extractors"** (per F1, RESEARCH §A):
```go
// New RefKind* constants for D-09's 6 missing edge kinds (RESEARCH §C.1;
// extends/overrides are Pass-2 synthesis, so they may not need a
// UnresolvedRef.Kind case — only Pass-1-captured kinds do):
const (
	RefKindReferences   = "references"   // new Pass-1 capture
	RefKindInstantiates = "instantiates" // new Pass-1 capture
	RefKindReturns      = "returns"      // new Pass-1 capture (reuses parsed ReturnType)
	RefKindTypeOf       = "type_of"      // new Pass-1 capture
)
// EdgeKindExtends is Pass-2 SYNTHESIS ONLY (split from the existing
// RefKindEmbeds→resolve.go promotion branch) — mirrors EdgeKindImplements's
// doc-comment discipline exactly (types.go:39-50): named here so
// resolve.go's promotion check and query/rwr.go's RankEdges share ONE
// definition.
const EdgeKindExtends = "extends"
// EdgeKindOverrides is Pass-2 SYNTHESIS ONLY (derived from
// contains+extends/implements/embeds, no new UnresolvedRef.Kind).
const EdgeKindOverrides = "overrides"
```

**Update `UnresolvedRef.Kind` doc comment** (`types.go:99-101`) to list all new Pass-1 kinds, mirroring the existing enumeration style.

---

### `internal/indexer/resolve.go` (extend) — service, batch/event-driven

**Analog:** itself — the `RefKindEmbeds` promotion branch (`resolve.go:116-150`) for the `extends` split; `synthesizeGoImplements` (`resolve.go:326-381`) for the `overrides` Pass-2 synthesis pattern.

**`extends` — split the existing embeds branch** (LOW difficulty per RESEARCH §B):
```go
// resolve.go:137 today: kind := "embeds" (unconditional fallback).
// D-09 split: when target is NOT an interface (the branch that today
// stays "embeds"), further distinguish class/struct-extends-class/struct
// as EdgeKindExtends vs. genuine same-kind embedding — exact split rule
// TBD by executor from RESEARCH §B's guidance (interface target →
// implements, unchanged; class/struct target → NEW extends). Mirrors the
// existing if/else structure at resolve.go:140-144 exactly, just adds one
// more branch.
```

**`references`/`instantiates`/`type_of` — new Pass-1 emit + Pass-2 `case` arms**, mirroring the existing `RefKindCalls` case shape (`resolve.go:100-114`) exactly:
```go
case goextract.RefKindReferences:
	targetID, ok := resolveNameRef(idx, r, ref.PkgAlias, ref.Name)
	if !ok {
		unresolvedCount++
		continue
	}
	edges = append(edges, &schema.Edge{
		Source: ref.FromID, Target: targetID, Kind: goextract.RefKindReferences,
		Line: ref.Line, Col: ref.Col, Provenance: "ast",
	})
	resolvedForFile++
// instantiates/type_of/returns follow the identical shape — resolve a
// name to a target id, filtered per RESEARCH §B's per-kind Kind-check
// disambiguation (e.g. instantiates target must be a type-Kind node).
```

**`overrides` — Pass-2 synthesis, mirror `synthesizeGoImplements`'s composition-from-already-resolved-edges pattern** (`resolve.go:326-381`, `326-338` doc comment is the template): walk `contains` (type→method) + `extends`/`implements`/`embeds` (type→supertype) edges already built, find same-named methods on supertypes, emit `overrides` edges. Reuses `retryConformanceCalls`'s supertype-walk helper shape (`resolve.go:249-324`, `walkSupertypesForMethod`) rather than writing a new BFS.

**Provenance discipline** — new Pass-2-synthesized edges (`extends` split's reclassified case, `overrides`) get `Provenance: "heuristic"` + a `Metadata: map[string]string{"synthesizedBy": "..."}` tag, exactly mirroring the existing `implements` promotion's metadata stamp (`resolve.go:143`, `"synthesizedBy": "declared-implements"`).

---

### Per-language extractors (`javaextract`/`csharpextract`/`pyextract`/`tsextract`)

**Analog:** `internal/indexer/goextract/*.go` — every sibling package's own doc comments already state "mirrors goextract's shape" (confirmed via RESEARCH §11's citation of `languages_*.go` doc comments). New Pass-1 emit sites for `references`/`instantiates`/`type_of` in each language extractor should locate the equivalent tree-sitter node kinds per RESEARCH §B's table (e.g. Go `composite_literal`/Java `object_creation_expression`/C# `object_creation_expression`/Python `call` with class-Kind target/TS `new_expression` for `instantiates`) and emit an `UnresolvedRef` with the new `Kind` exactly like the existing `RefKindCalls`/`RefKindEmbeds` emit sites in each package.

---

## Shared Patterns

### WR-04 dangling-edge tolerance
**Source:** `internal/query/traverse.go` (`BuildReverseAdjacency`, `Callers`, `Impact` — every `e.reader.GetNode` call after an edge scan)
**Apply to:** `rwr.go`'s adjacency build, `node.go`'s multi-def edge reads, `explore_gate.go`'s per-file mass aggregation.
```go
target, err := e.reader.GetNode(edge.Target)
if err != nil {
	if errors.Is(err, graphstore.ErrNotFound) {
		continue // skip a dangling edge rather than aborting
	}
	return "", err
}
```

### Deterministic lowest-`Id` tie-break (D-04)
**Source:** `internal/query/traverse.go:194-218` (`resolveSymbolNode`)
**Apply to:** RWR score ties (rwr.go), node multi-def secondary sort (node.go), any new ranking code.
```go
if best == nil || n.Id < best.Id {
	best = n
}
```

### Fresh-per-call, no package-level cache (RESEARCH Pitfall 2 / T-03-04-Stale)
**Source:** `internal/query/traverse.go:15-31` (`BuildReverseAdjacency` doc comment)
**Apply to:** `rwr.go`'s graph build, `explore_gate.go`'s gate computation — a long-lived MCP process must never serve a stale point-in-time view.

### One shared edge-kind definition, not three copies
**Source:** `internal/indexer/goextract/types.go:39-50` (`EdgeKindImplements` doc comment: "so dispatch.go, resolve.go's promotion check, and query/traverse.go's dispatch-index filter share ONE definition rather than three copies")
**Apply to:** every new `RefKind*`/`EdgeKind*` constant (F1) — declare once in `goextract/types.go`, import everywhere else (resolve.go, rwr.go's `RankEdges`), never redeclare the string literal.

### Plain-text renderer discipline (hard constraint)
**Source:** `internal/query/render_markdown.go` (zero ANSI/color codes anywhere today)
**Apply to:** all `RenderExplore`/`RenderNode` extensions — markdown-only (`**bold**`, backticks), never a color/style escape sequence.

### Shared-Engine CLI/MCP parity (EXPL-05/NODE-04 — already structurally correct)
**Source:** `internal/mcp/tools.go` `exploreHandler` (`tools.go:81-101`) and `companionHandler["node"]` (`tools.go:173-189`)
**Apply to:** confirms NO new plumbing is needed in `internal/mcp` — both handlers already call `eng.Explore(q, maxFiles)` / `eng.Node(symbol, file)` directly and return the markdown text as-is. Any algorithm change inside `Engine.Explore`/`Engine.Node` is automatically visible on both surfaces. The only MCP-surface work this phase needs is in `capture.sh` (a new invocation path to exercise the TS MCP server for fixture capture) — not in `internal/mcp/tools.go` itself.

### Golden fixture capture — `capture_repo` extension pattern
**Source:** `testdata/golden/capture.sh:84-103` (`capture_repo` function)
**Apply to:** new behavioral fixture invocations. The existing function hardcodes `--max-files 1` (line 97) and `-f "$symbol_file"` (line 99) — RESEARCH Pitfall 3 confirms these must be extended with NEW invocations (not edits to the existing ones, which stay as the template-parity baseline): a multi-word `explore` call WITHOUT `--max-files 1`, a `node <name>` call WITHOUT `-f` on a symbol with 2+ defs, plus new `wrap_text`-wrapped JSON envelopes exactly like the existing `explore.json`/`node.json` shape (`wrap_text` helper, `capture.sh:76-82`).

---

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| MCP-surface capture mechanism in `capture.sh` (driving TS's stdio MCP server programmatically) | test infra | request-response | No existing Go or shell code in this repo drives an MCP server as a test client — `capture.sh` today is CLI-only. RESEARCH suggests `mcp-go`'s test client or a small Node harness script; no in-repo precedent exists to copy from. |
| `internal/query/rwr.go`'s power-iteration loop itself | algorithm | batch | No existing Go code in this repo implements iterative graph-mass propagation (BFS in `traverse.go`'s `Impact` is the closest structural cousin — bounded frontier expansion — but RWR's fixed-iteration mass-redistribution math has no in-repo precedent). Port directly from the verbatim TS source (RESEARCH Code Examples §3), not from a repo analog. |

## Metadata

**Analog search scope:** `internal/query/`, `internal/cli/`, `internal/mcp/`, `internal/indexer/` (goextract + resolve.go + language extractors), `testdata/golden/`.
**Files scanned:** ~20 (explore.go, node.go, traverse.go, search.go, render_markdown.go, engine.go references, cli/explore.go, cli/node.go, mcp/tools.go, goextract/types.go, resolve.go, capture.sh, golden_test.go/golden_parity_test.go headers).
**Pattern extraction date:** 2026-07-14
