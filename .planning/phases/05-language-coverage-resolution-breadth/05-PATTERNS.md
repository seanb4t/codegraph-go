# Phase 5: Language Coverage & Resolution Breadth - Pattern Map

**Mapped:** 2026-07-11
**Files analyzed:** 26 (new packages/files + modified seams)
**Analogs found:** 24 / 26 (2 have no direct analog — new architecture)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/indexer/languages.go` (NEW — `LanguageSpec`/registry) | config/registry | request-response (lookup by lang ID) | `internal/parser/cgo/parser_cgo.go` (per-lang constructors) + `internal/indexer/pipeline.go` (dispatch caller) | role-match |
| `internal/indexer/javaextract/javaextract.go` (NEW) | service (extractor) | transform (AST → FileResult) | `internal/indexer/goextract/goextract.go` | exact |
| `internal/indexer/javaextract/types.go` (NEW) | model/vocabulary | transform | `internal/indexer/goextract/types.go` | exact |
| `internal/indexer/javaextract/javaextract_test.go` (NEW) | test | transform | `internal/indexer/goextract/goextract_test.go` | exact |
| `internal/indexer/csharpextract/csharpextract.go` (NEW) | service (extractor) | transform | `internal/indexer/goextract/goextract.go` | exact |
| `internal/indexer/csharpextract/types.go` (NEW) | model/vocabulary | transform | `internal/indexer/goextract/types.go` | exact |
| `internal/indexer/pyextract/pyextract.go` (NEW) | service (extractor) | transform | `internal/indexer/goextract/goextract.go` | exact |
| `internal/indexer/tsextract/tsextract.go` (NEW — TS+TSX+JS shared) | service (extractor) | transform | `internal/indexer/goextract/goextract.go` | exact |
| `internal/indexer/mainstream/{rust,ruby,php,c,swift,kotlin}extract/*.go` (NEW, 6 pkgs) | service (extractor) | transform | `internal/indexer/goextract/goextract.go` | role-match (lighter — extraction + best-effort only, per D-04 tiering) |
| `internal/parser/cgo/parser_cgo.go` (MODIFIED — add `NewJavaParser`, `NewCSharpParser`, `NewJavaScriptParser`, `NewTypeScriptParser`, `NewTSXParser`, `NewRustParser`, etc.) | service (parser constructors) | transform | itself — `NewGoParser`/`NewPythonParser` (lines 32-42) | exact |
| `internal/indexer/discover.go` (MODIFIED — generalize to ext→lang registry + walker + descriptor hooks) | service (discovery) | file-I/O | itself — `Discover`/`ShouldSkipDir`/`readModulePath`/`importPathFor` | exact (self-analog for the generalization) |
| `internal/indexer/extract.go` (MODIFIED — fix Pitfall 1: per-file parser+extractor selection, not per-worker-lifetime) | service (worker pool) | transform, event-driven (worker fan-out) | itself — `extractWithFactory` (lines 65-140) | exact (self-analog) |
| `internal/indexer/resolve.go` (MODIFIED — per-language resolver dispatch, WR-01/WR-02 fixes, call-as-argument gap) | service (resolver) | transform, CRUD (symbol index build/lookup) | itself — `resolveRefsWithIndex`/`resolveNameRef` (lines 40-179) | exact (self-analog) |
| `internal/indexer/symbolindex.go` (MODIFIED — generalize `byImportAndName` → per-language `ModuleKey` hook) | model/service (index) | CRUD | itself — `symbolIndex`/`newSymbolIndex`/`overlay` | exact (self-analog) |
| `internal/indexer/dispatch/implements.go` (NEW) | service (resolve-time synthesis pass) | transform, batch | `internal/indexer/resolve.go` (`resolveRefsWithIndex`'s edge-synthesis shape + `collapseEdges`'s bounded/deterministic pattern) | role-match |
| `internal/indexer/dispatch/implements_test.go` (NEW) | test | transform | `internal/indexer/goextract/goextract_test.go` (table-driven) | role-match |
| `internal/indexer/routes/registry.go` (NEW) | service (detector registry) | event-driven (opt-in per detected dependency) | `internal/indexer/languages.go` (registry-keyed-by-ID shape, same file, new) — secondary: `symbolindex.go`'s map-based registry pattern | role-match |
| `internal/indexer/routes/{gin,spring,aspnet,django,express}.go` (NEW, 5 files) | service (per-framework detector) | transform (AST → route node/edge) | `internal/indexer/goextract/goextract.go` (AST-walk → schema.Node/Edge emission pattern) | role-match |
| `internal/indexer/routes/*_test.go` (NEW) | test | transform | `internal/indexer/goextract/goextract_test.go` | role-match |
| `internal/query/traverse.go` (MODIFIED — add `implements`-edge traversal step to Callers/Impact) | service (query traversal) | streaming (iterator scan) | itself — `BuildReverseAdjacency` (lines 31-50) | exact (self-analog) |
| `internal/schema/graph.pb.go` (NO CHANGE — verify only) | model | — | itself — `Edge` struct (line 214: `Provenance`, `Line`, `Col`, `Metadata`) | exact (already has fields) |
| `testdata/golden/corpus/<lang>/` (NEW fixtures, per priority-4 lang) | test fixture | file-I/O | `testdata/golden/` weft-go corpus + `golden_parity_test.go` pattern | exact |
| `testdata/golden/golden_parity_test.go` (MODIFIED — add per-language test funcs) | test | transform | itself — existing `resolveWeftCorpus` pinned-commit-skip pattern | exact (self-analog) |
| `docs/LANGUAGE-CAPABILITY-MATRIX.md` (NEW, D-11) | config/doc | — | none — no analog | no analog |
| `internal/indexer/capability/matrix.go` (NEW, D-11 machine-readable descriptor) | model/config | — | `internal/indexer/languages.go` (registry-of-structs shape, same new file) | role-match |

## Pattern Assignments

### `internal/indexer/javaextract/javaextract.go` (and csharpextract/pyextract/tsextract — same shape)

**Analog:** `internal/indexer/goextract/goextract.go`

**Imports pattern** (lines 1-17):
```go
package goextract

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/schema"
)
```
Per-language extractor swaps only the parse-tree node-kind strings consulted; the import block (nodeid, parser, schema, tree_sitter) is identical across every extractor package.

**Entry-point signature + skip/error contract** (lines 19-51):
```go
func Extract(p parser.Parser, importPath, relPath string, src []byte) (FileResult, error) {
	sum := sha256.Sum256(src)
	result := FileResult{
		ImportPath:  importPath,
		RelPath:     relPath,
		Language:    "go",
		ContentHash: hex.EncodeToString(sum[:]),
		Imports:     make(map[string]string),
	}

	tree, err := p.Parse(src, nil)
	if err != nil {
		result.Err = err
		return result, nil   // per-file failure never aborts the batch
	}
	defer tree.Close()

	native, ok := tree.Inner().(*tree_sitter.Tree)
	if !ok || native == nil {
		result.Err = fmt.Errorf("goextract: parser returned an unexpected tree type for %s", relPath)
		return result, nil
	}
	root := native.RootNode()
	...
```
Every new extractor's `Extract(p parser.Parser, moduleKey, relPath string, src []byte) (FileResult, error)` MUST reproduce this exact skip contract: a parse/tree failure sets `FileResult.Err` and returns a **nil** function error — one bad file never aborts the batch (RESEARCH Pitfall 4). `importPath` generalizes to `moduleKey` per D-01/Pitfall 2 (Java package name, C# namespace, Python dotted module path, TS/JS resolved module specifier) — same field name and position, different semantic source.

**File node + extractor struct pattern** (lines 58-97):
```go
	fileID := nodeid.NodeID(KindFile, relPath, relPath)
	result.Nodes = append(result.Nodes, ExtractedNode{Node: &schema.Node{
		Id: fileID, Kind: KindFile, Name: relPath, QualifiedName: relPath,
		FilePath: relPath, Language: "go", StartLine: 1,
		EndLine: int32(root.EndPosition().Row) + 1,
	}})

	ex := &extractor{
		src: src, relPath: relPath, fileID: fileID, result: &result,
		typeNodesByName: make(map[string]string),
	}

	ex.collectTypes(root)
	ex.collectConstsVars(root)
	ex.collectImports(root)
	ex.collectFuncsAndMethods(root)

	return result, nil
```
Reuse the `extractor` struct-carrying-state pattern (fields threaded through helper methods walking the tree) and the ordered collect-phase calls (types first, so same-file method receiver-type lookups always succeed regardless of declaration order — same ordering discipline applies to Java/C# class-before-method, Python class-before-def).

### `internal/indexer/goextract/types.go` (shared vocabulary — extend additively, do not fork)

**Analog:** `internal/indexer/goextract/types.go` itself (lines 1-114)

```go
const (
	KindFile      = "file"
	KindFunction  = "function"
	KindMethod    = "method"
	KindStruct    = "struct"
	KindInterface = "interface"
	KindTypeAlias = "type_alias"
	KindConstant  = "constant"
	KindVariable  = "variable"
)

const (
	RefKindCalls    = "calls"
	RefKindImports  = "imports"
	RefKindEmbeds   = "embeds"
	RefKindContains = "contains"
)

type ExtractedNode struct{ Node *schema.Node }
type IntraEdge struct{ Edge *schema.Edge }
type UnresolvedRef struct {
	FromID   string
	Name     string
	PkgAlias string
	Kind     string
	Line, Col int32
}
type FileResult struct {
	ImportPath, RelPath, Language string
	Nodes      []ExtractedNode
	IntraEdges []IntraEdge
	Unresolved []UnresolvedRef
	Imports    map[string]string
	ContentHash string
	MtimeUnixNs int64
	SizeBytes   int64
	Err error
}
```
D-01 mandates reusing `FileResult`/`ExtractedNode`/`IntraEdge`/`UnresolvedRef` and the `Kind*`/`RefKind*` constants **unchanged, extended additively** (new node kind: `KindRoute = "route"`; new ref kinds only if a language needs one beyond calls/imports/embeds/contains — e.g. Java/C#/TS declared-`extends` reuses `RefKindEmbeds`-shaped unresolved refs promoted to `implements` at resolve time per Pattern 2 in RESEARCH, not a new ref kind). Every per-language extractor package imports these types from `goextract`, it does NOT redefine its own `FileResult` shape.

### `internal/indexer/languages.go` (NEW — LanguageSpec registry, D-01)

**Analog:** `internal/parser/cgo/parser_cgo.go` (per-language constructor pattern) + `internal/indexer/pipeline.go` (today's single-language call site to generalize)

**Per-language constructor pattern to mirror** (`internal/parser/cgo/parser_cgo.go` lines 32-42):
```go
// NewGoParser returns a CGoParser configured for the Go grammar.
func NewGoParser() (*CGoParser, error) {
	return newCGoParser(tree_sitter_go.Language())
}

// NewPythonParser returns a CGoParser configured for the Python grammar.
func NewPythonParser() (*CGoParser, error) {
	return newCGoParser(tree_sitter_python.Language())
}
```
New grammars (`NewJavaParser`, `NewCSharpParser`, `NewJavaScriptParser`, `NewTypeScriptParser`, `NewTSXParser`, `NewRustParser`, `NewRubyParser`, `NewPHPParser`, `NewCParser`, `NewCppParser`, `NewSwiftParser`, `NewKotlinParser`) follow this exact one-liner-per-grammar shape — `newCGoParser(tree_sitter_<lang>.Language())` — no seam change to `parser.Parser`.

**Dispatch point to generalize** — today `internal/indexer/extract.go`'s `defaultParserFactory` hardcodes `cgo.NewGoParser()` (lines 22-24) and `goextract.Extract(p, ...)` is called directly (line 112). The `languages.go` registry becomes the single source of truth both `extract.go` (parser+extractor selection per file's `DiscoveredFile.Language`) and `discover.go` (extension→language mapping) consult — see RESEARCH's illustrative `LanguageSpec` shape (already vetted against this codebase's actual seams, reuse verbatim as starting point):
```go
type LanguageSpec struct {
	ID         string
	Extensions []string
	NewParser  func() (parser.Parser, error)
	Extract    func(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error)
	ModuleKey  func(descriptor ProjectDescriptor, relPath string) string
	Descriptor func(root string) (ProjectDescriptor, error)
}
```

### `internal/indexer/discover.go` (MODIFIED — generalize ext→lang walk + descriptor hooks, D-03)

**Analog:** itself — `Discover`/`ShouldSkipDir`/`readModulePath`/`importPathFor` (full file, 158 lines)

**Skip predicate to reuse unchanged** (lines 43-45):
```go
func ShouldSkipDir(name string) bool {
	return name == "vendor" || strings.HasPrefix(name, ".")
}
```
Both the generalized walker and the Phase-4 watcher's recursive-add loop MUST keep calling this exact predicate — do not fork a second skip list.

**Walk + stable-sort pattern to generalize** (lines 63-128): the `filepath.WalkDir` callback with `ShouldSkipDir`-driven `fs.SkipDir`, ext filter (today hardcoded `.go`, becomes the extension→language registry lookup), and the closing `sort.Slice(files, ... RelPath ...)` determinism guarantee — this exact ordering guarantee must survive the generalization untouched (D-01a determinism is load-bearing project-wide).

**Project-descriptor hook to generalize** (lines 130-158, `readModulePath` + `importPathFor`): Go's go.mod-based module path resolution becomes the FIRST implementation of the new `Descriptor(root string) (ProjectDescriptor, error)` / `ModuleKey(...)` hook (Pitfall 2) — read this pair as the concrete shape every new language's descriptor hook (pom.xml, *.csproj, pyproject.toml, package.json+tsconfig.json) must match: parse-once-per-repo-root, return an error if the manifest is malformed/missing (a file whose language has no descriptor still gets extracted with path-based identity per D-03 — this is a NEW behavior `readModulePath`'s current all-or-nothing error return does not have, so the modified version must add a "descriptor absent, fall back to path identity" branch rather than hard-failing `Discover` entirely).

### `internal/indexer/extract.go` (MODIFIED — fix Pitfall 1, per-file parser+extractor selection)

**Analog:** itself — `extractWithFactory` (lines 65-140)

**Current one-parser-per-worker-lifetime shape (the bug to fix):**
```go
func extractWithFactory(files []DiscoveredFile, limit int, newParser parserFactory) ([]goextract.FileResult, error) {
	...
	g := new(errgroup.Group)
	for w := 0; w < limit; w++ {
		g.Go(func() error {
			p, err := newParser()   // ONE parser, whole worker lifetime — breaks multi-language
			if err != nil { return fmt.Errorf(...) }
			defer p.Close()

			for {
				i := int(next.Add(1)) - 1
				if i >= len(files) { return nil }
				f := files[i]
				src, err := os.ReadFile(f.AbsPath)
				...
				r, err := goextract.Extract(p, f.ImportPath, f.RelPath, src)  // hardcoded goextract
				...
				results[i] = r
			}
		})
	}
	...
}
```
**Required fix shape (per RESEARCH Pitfall 1 + Open Question 1's recommendation):** replace the single `p` with a per-worker `map[string]parser.Parser` cache keyed by `f.Language`, lazily constructed via the `languages.go` registry's `NewParser` on first encounter of that language, and `Close()`'d for every cached entry at worker exit (not just one `defer p.Close()`). Replace the hardcoded `goextract.Extract(p, ...)` call with a `languages.go` registry lookup by `f.Language` returning the right `Extract` func. Preserve everything else verbatim: the atomic-counter work-stealing loop, the disjoint pre-allocated `results[i]` write (never append), and the per-file-error-vs-batch-fatal-error distinction (lines 96-121).

### `internal/indexer/resolve.go` / `symbolindex.go` (MODIFIED — per-language resolver + ModuleKey, WR-01/WR-02, call-as-argument gap)

**Analog:** itself — `resolveRefsWithIndex` (lines 40-179), `symbolIndex` (whole file)

**Symbol index structure to generalize (Pitfall 2)** — `symbolindex.go` lines 8-20:
```go
type symbolIndex struct {
	// byImportAndName is keyed importPath -> declaredName -> nodeID.
	byImportAndName map[string]map[string]string
}
```
The `map[moduleKey]map[symbolName]nodeID` STRUCTURE stays; only the string computed as the outer key changes from Go's `ImportPath` to each language's `ModuleKey` (RESEARCH Pitfall 2's exact recommendation — do not redesign the two-level map).

**WR-01 site** — `overlay` (lines 40-61), specifically `names[en.Node.Name] = en.Node.Id` (line 58) unconditionally overwrites on a same-module-key name collision (e.g. two files in the same Go package both declaring a same-named func/method) — this is the WR-01 bug; the fix must detect/handle the collision (documented in D-05) rather than silently last-write-wins.

**WR-02 site + resolveSelector's narrowest-safe-set boundary (already correct, keep this shape)** — lines 105-124:
```go
func (idx *symbolIndex) resolveSelector(callerImports map[string]string, pkgAlias, name string) (string, bool) {
	importPath, ok := callerImports[pkgAlias]
	if !ok {
		return "", false   // correct: non-import-alias operand never resolves here
	}
	...
}
```
WR-02 (selector calls on non-identifier operands mis-resolved as same-package) is a different call site in `goextract.go`'s call-collection walk (not shown above — the extractor must not emit a qualified `UnresolvedRef` with `PkgAlias` set for a non-identifier operand in the first place), NOT a bug in `resolveSelector` itself — this function's alias-membership check is already the correct narrowest-safe-set boundary and should be preserved as the pattern every new language's selector-resolution follows.

**Two-pass conformance-retry shape (Pitfall 3, new requirement for Java/C#/TS)** — `resolveRefsWithIndex`'s switch-on-`ref.Kind` structure (lines 65-128) is the Pass-2-resolution shape to extend with a SECOND retry pass: resolve everything resolvable without inheritance info first (as today), synthesize `extends`/`implements` edges, then retry any call that failed pass 1 whose origin type has an `extends`/`implements` edge, walking the supertype chain. Implement as a second loop over `r.Unresolved` after the `implements`-synthesis pass (Wave C dependency), not interleaved into the existing single loop.

**Deterministic edge collapse (reuse unchanged)** — `collapseEdges` (lines 210-248) is the canonical determinism pattern (`edgeTriple{source, kind, target}` grouping + `(filePath, line, col)` total-order tie-break) — the `implements`-edge synthesis pass (dispatch/implements.go) and route detectors (routes/*.go) both emit edges that flow through this SAME function; do not write a second collapse function.

### `internal/indexer/dispatch/implements.go` (NEW — RES-02/RES-03)

**Analog:** `internal/indexer/resolve.go`'s edge-synthesis shape (the `schema.Edge{...}` literal construction pattern, lines 72-127) + RESEARCH's directly-provided illustrative code (already vetted against `schema.Edge`'s real field names)

**Edge construction pattern to copy** (mirroring `resolve.go` line 72-75's shape, adapted per RESEARCH's Code Examples section):
```go
edges = append(edges, &schema.Edge{
	Source: ref.FromID, Target: calleeID, Kind: "calls",
	Line: ref.Line, Col: ref.Col, Provenance: "ast",
})
```
becomes, for synthesized dispatch edges:
```go
edges = append(edges, &schema.Edge{
	Source: structID, Target: ifaceID, Kind: "implements",
	Provenance: "heuristic",
	Metadata: map[string]string{"synthesizedBy": "go-structural-methodset"},
})
```
`Provenance: "heuristic"` (vs. `"ast"` for every existing edge in `resolve.go`) is the ONE required field difference (D-07/RES-03) — no schema change, `schema.Edge` (`internal/schema/graph.pb.go` line 214) already carries `Provenance string`, `Line int32`, `Col int32`, `Metadata map[string]string` verified present.

**Bounded matching (avoid quadratic blowup, D-06)** — build an inverted index (`methodName -> []interfaceID`) BEFORE the struct×interface comparison loop, exactly as RESEARCH's Code Examples section specifies; do not write a naive nested-loop-over-all-pairs matcher.

### `internal/indexer/routes/*.go` (NEW — LANG-07 framework detectors)

**Analog:** `internal/indexer/goextract/goextract.go`'s AST-walk-emits-node/edge pattern; no direct analog exists for the registry-of-detectors shape, closest is `languages.go`'s registry-keyed-by-ID pattern (same new file, see above)

**Node/edge shape (per D-08, vocabulary confirmed against parity target in RESEARCH):**
```go
// Node: Kind = "route", Name = "GET /users/:id",
//       QualifiedName = filePath + "::route:" + path
// Edge: route -> handler, Kind = "calls" (reuses EXISTING RefKindCalls-filtered
//       reverse adjacency in traverse.go — zero query-engine changes needed
//       for THIS edge kind), Provenance = "heuristic",
//       Metadata = {"synthesizedBy": "gin-route", "httpMethod": "GET", "routePath": "/users/:id"}
```
Use `Kind: goextract.RefKindCalls` (the existing `"calls"` constant, not a new `"handles"` kind) for the route→handler edge specifically so `internal/query/traverse.go`'s `BuildReverseAdjacency` (which filters `e.Kind != goextract.RefKindCalls`, line 41) picks these up with zero traversal-code changes — this is RESEARCH's explicit primary recommendation. Detect via the already-parsed AST (walk `call_expression`-equivalent nodes already visited during extraction, filter by HTTP-verb-method-name + string-literal-path + handler-arg shape), NOT a second regex pass over raw source (RESEARCH Pattern 4 explicitly prefers AST over the parity target's own regex approach, citing ReDoS avoidance).

**Opt-in gating (D-09)** — each detector's `detect()` must check for the framework's dependency/import signature (e.g. `gin-gonic/gin` in go.mod, `org.springframework` in pom.xml) BEFORE running its route-emission walk — mirror the `ProjectDescriptor`-hook pattern from `discover.go`'s generalization (manifest already parsed once per repo root, reuse that parse, don't re-read the manifest file per detector).

### `internal/query/traverse.go` (MODIFIED — implements-edge dispatch traversal, RES-02)

**Analog:** itself — `BuildReverseAdjacency` (lines 31-50)

```go
func BuildReverseAdjacency(r graphstore.Reader) (map[string][]*schema.Edge, error) {
	it, err := r.IterateEdges("")
	if err != nil { return nil, err }
	defer it.Close()

	rev := make(map[string][]*schema.Edge)
	for it.Next() {
		e := it.Edge()
		if e.Kind != goextract.RefKindCalls {
			continue
		}
		rev[e.Target] = append(rev[e.Target], e)
	}
	if err := it.Err(); err != nil { return nil, err }
	return rev, nil
}
```
D-06 requires `Callers`/`Impact` to traverse `implements` edges at query time (interface method → concrete impls), which is a SEPARATE traversal shape from this call-graph-only reverse adjacency (name-joined method lookup across an `implements` edge, not identity-followed like `calls`) — RESEARCH explicitly flags this needs NEW code in `query.Callers`/`query.Impact`, not a filter-relaxation on this existing function. Do not widen `BuildReverseAdjacency`'s `RefKindCalls`-only filter to include `implements` — build a second, purpose-built adjacency step (e.g. `BuildImplementsIndex` mirroring this function's shape: one full `IterateEdges("")` scan, filtered to `Kind == "implements"`, keyed by `Target` i.e. the interface) and compose it into `Callers`/`Impact` at the call sites that currently consult `BuildReverseAdjacency` alone.

### `testdata/golden/golden_parity_test.go` (MODIFIED — per-language golden fixtures, D-12)

**Analog:** itself — the existing `resolveWeftCorpus` pinned-commit-skip pattern (not fully read this session — file exists per RESEARCH's Sources section as "already-committed, already-verified code"; planner/implementer should read this specific function's skip-if-corpus-absent shape before writing `TestGoldenParity_Java`/`_CSharp`/`_Python`/`_TSJS`, mirroring its self-skip discipline so CI doesn't hard-fail when a corpus isn't checked out).

## Shared Patterns

### Per-file skip/error contract (never abort the batch)
**Source:** `internal/indexer/goextract/goextract.go` lines 40-51 (`Extract`'s parse-failure branch) and `internal/indexer/extract.go` lines 96-121 (`extractWithFactory`'s read-failure branch)
**Apply to:** every new `<lang>extract.Extract` function and the worker-pool's per-file dispatch loop
```go
tree, err := p.Parse(src, nil)
if err != nil {
	result.Err = err
	return result, nil   // nil function error — this file is SKIPPED, not fatal
}
```

### Deterministic sort-then-collapse
**Source:** `internal/indexer/discover.go` line 126 (`sort.Slice(files, ...RelPath...)`) and `internal/indexer/resolve.go` lines 210-248 (`collapseEdges`)
**Apply to:** every new extractor's file enumeration, and every new edge-synthesis pass (dispatch/implements.go, routes/*.go) — all synthesized edges MUST flow through the existing `collapseEdges` before commit, never a parallel ad hoc dedup.

### Provenance tagging
**Source:** `internal/schema/graph.pb.go` line 214 (`Edge.Provenance`, `Edge.Line`, `Edge.Col`, `Edge.Metadata` — already present, no schema change) and `internal/indexer/resolve.go` line 74 (`Provenance: "ast"` on every existing ground-truth edge)
**Apply to:** every edge `dispatch/implements.go` and `routes/*.go` synthesize — set `Provenance: "heuristic"` (never `"ast"`, which is reserved for extraction-time ground truth), always populate `Line`/`Col` when a source location exists, and use `Metadata["synthesizedBy"]` to name which heuristic fired.

### Registry-keyed-by-ID
**Source:** `internal/parser/cgo/parser_cgo.go`'s per-language constructor functions (`NewGoParser`, `NewPythonParser`) called through a lookup keyed by language, and `internal/indexer/symbolindex.go`'s two-level `map[string]map[string]string` keying pattern
**Apply to:** `languages.go` (`map[string]LanguageSpec`), `routes/registry.go` (`map[(language, framework)]Detector`), `internal/indexer/capability/matrix.go` (`map[string]CapabilityEntry`) — all three new registries should follow the same "package-level `var registry = map[string]T{...}`, looked up by a stable string ID, never re-built per call" shape already established by `symbolIndex`.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `docs/LANGUAGE-CAPABILITY-MATRIX.md` | doc | — | D-11's human-readable matrix is a wholly new artifact type for this project; no existing doc of this shape to pattern-match against. Use RESEARCH's Validation Architecture test map (LANG-02..07/RES-02/RES-03 rows) as the row structure. |
| `internal/indexer/routes/registry.go` (the registry itself, not the per-framework detectors) | service (registry) | event-driven | No prior per-(language,framework)-keyed registry exists in this codebase; closest available shape is `languages.go`'s new registry (also being created this phase) — treat both as siblings designed together, not one derived from a pre-existing pattern. |

## Metadata

**Analog search scope:** `internal/indexer/`, `internal/parser/`, `internal/query/`, `internal/schema/`, `testdata/golden/`
**Files scanned:** 12 read directly (goextract.go, types.go, goextract_test.go, discover.go, resolve.go, symbolindex.go, extract.go, pipeline.go, parser_cgo.go, parser.go [referenced], traverse.go, graph.pb.go)
**Pattern extraction date:** 2026-07-11
