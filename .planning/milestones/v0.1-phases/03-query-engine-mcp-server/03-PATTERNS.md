# Phase 3: Query Engine & MCP Server - Pattern Map

**Mapped:** 2026-07-11
**Files analyzed:** ~20 (2 modified, ~18 new)
**Analogs found:** 17 / 20 (3 have no in-repo analog — MCP transport, markdown templates, reverse-adjacency builder — grounded in RESEARCH instead)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/graphstore/store.go` (Reader interface +2 methods) | interface/config | CRUD (read) | same file, `IterateEdges` method | exact |
| `internal/graphstore/pebble_store.go` (`IterateNodes`/`IterateFiles` + iterator types) | storage/service | streaming (range-scan) | `pebbleReader.IterateEdges` + `pebbleEdgeIterator` (same file) | exact |
| `internal/graphstore/keys.go` (`nodeKeyPrefix`/`filePrefix` helpers, if needed) | utility | transform | `edgeSrcPrefix`/`fileKey` (same file) | exact |
| `internal/query/engine.go` | service | request-response | `internal/indexer/pipeline.go` (orchestration over a Reader-shaped dependency) | role-match |
| `internal/query/resolve.go` (`resolveCodegraphDir`) | utility | file-I/O | `internal/cli/init.go`'s `targetRoot`/`codegraphDirName` walk | role-match |
| `internal/query/search.go` (lexical matcher) | service | transform | `internal/indexer/symbolindex.go` (in-memory map-index built once, queried by lookup) | role-match |
| `internal/query/traverse.go` (reverse-adjacency + BFS) | service | transform | `internal/indexer/symbolindex.go` (build-once map index) + `pebbleEdgeIterator` consumption pattern | role-match |
| `internal/query/render_json.go` | utility | transform | none in-repo (new); ground in RESEARCH golden JSON shapes | no analog |
| `internal/query/render_markdown.go` | utility | transform | none in-repo (new); ground in RESEARCH `explore.json`/`node.json` templates | no analog |
| `internal/query/engine_test.go` | test | — | `internal/graphstore/store_test.go` (Reader behavior tests against a real Pebble-backed store) | role-match |
| `internal/mcp/server.go` | service/provider | event-driven (stdio) | none in-repo (new); ground in RESEARCH Pattern 3 (mcp-go API) | no analog |
| `internal/mcp/tools.go` | controller | request-response | `internal/mcp/server.go` (sibling, same file's tool-registration idiom) + CLI command RunE bodies as the "parse args, delegate to engine" pattern | role-match |
| `internal/mcp/server_test.go` | test | — | `internal/cli/cli_test.go` (`execCmd`-style black-box exercise of the command tree) | role-match |
| `internal/cli/root.go` (extend `AddCommand`) | config | — | same file | exact |
| `internal/cli/query.go`, `node.go`, `search.go`, `callers.go`, `callees.go`, `impact.go`, `affected.go`, `files.go`, `status.go`, `explore.go` | controller | request-response | `internal/cli/index.go` (thin RunE: `targetRoot` → resolve `.codegraph/` → delegate → print) | exact |
| `internal/cli/serve.go` | controller | event-driven | `internal/cli/init.go` (RunE resolves path/flags, delegates to a package function) | role-match |
| `internal/cli/cli_test.go` (extend) | test | — | same file's `execCmd`/`copyFixture`/`readGraphCounts` helpers | exact |
| `testdata/golden/golden_parity_test.go` (new) | test | batch (fixture diff) | `testdata/golden/golden_test.go` (`findVolatileKeys`, fixture-loading conventions) | exact |
| `go.mod` | config | — | existing `require` block (see `github.com/cockroachdb/pebble/v2`, `github.com/spf13/cobra` entries) | exact |

## Pattern Assignments

### `internal/graphstore/store.go` + `pebble_store.go` — `IterateNodes`/`IterateFiles` (D-03)

**Analog:** `IterateEdges` / `pebbleEdgeIterator`, same files.

**Interface addition pattern** (`store.go` lines 51-58, mirror exactly):
```go
// IterateEdges returns an EdgeIterator over every edge whose source
// is srcPrefix, ordered by (src, kind, dst) — a single contiguous
// range scan (D-03), suitable for callers/callees/impact queries.
IterateEdges(srcPrefix string) (EdgeIterator, error)
```
Add directly below, same doc-comment register (explain the scan contract + D-03/Pitfall-1 full-scan-with-filter caveat for `IterateNodes`):
```go
// IterateNodes returns a NodeIterator over every node record. Phase 3
// v1 kind-filtering, if any, is applied in-memory by the caller/query
// engine — see keys.go's appendSegment doc for why a byte-prefix scan
// cannot cheaply isolate one kind (RESEARCH Pitfall 1).
IterateNodes() (NodeIterator, error)

// IterateFiles returns a FileIterator over every file record under
// the f/ namespace.
IterateFiles() (FileIterator, error)
```
Add matching `NodeIterator`/`FileIterator` interfaces directly below `EdgeIterator` (same four-method shape: `Next() bool`, `Node()`/`File()` accessor, `Err() error`, `Close() error`).

**Implementation pattern** (`pebble_store.go` lines 139-196, mirror `IterateEdges`/`pebbleEdgeIterator` verbatim, substituting the namespace prefix and record type):
```go
func (r *pebbleReader) IterateEdges(srcPrefix string) (EdgeIterator, error) {
	lower := edgeSrcPrefix(srcPrefix)
	upper := rangeUpperBound(lower)
	iter, err := r.snap.NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	return &pebbleEdgeIterator{iter: iter}, nil
}
```
```go
type pebbleEdgeIterator struct {
	iter    *pebble.Iterator
	started bool
	cur     *schema.Edge
	err     error
}

func (it *pebbleEdgeIterator) Next() bool {
	if it.err != nil {
		return false
	}
	var ok bool
	if !it.started {
		it.started = true
		ok = it.iter.First()
	} else {
		ok = it.iter.Next()
	}
	if !ok {
		if err := it.iter.Error(); err != nil {
			it.err = err
		}
		return false
	}
	var e schema.Edge
	if err := proto.Unmarshal(it.iter.Value(), &e); err != nil {
		it.err = err
		return false
	}
	it.cur = &e
	return true
}

func (it *pebbleEdgeIterator) Edge() *schema.Edge { return it.cur }
func (it *pebbleEdgeIterator) Err() error         { return it.err }
func (it *pebbleEdgeIterator) Close() error       { return it.iter.Close() }
```
`IterateNodes`/`IterateFiles` and their iterators are byte-for-byte the same shape: `lower := []byte{prefixNode}` (or reuse `nodeKeyPrefix()`/`filePrefix()` new helpers in `keys.go`, built the same way as `edgeSrcPrefix` at lines 88-93), `upper := rangeUpperBound(lower)`, unmarshal into `schema.Node`/`schema.File` instead of `schema.Edge`.

**`keys.go` helper pattern** (lines 82-93, mirror for the new prefixes):
```go
func edgeSrcPrefix(src string) []byte {
	buf := make([]byte, 0, 1+binary.MaxVarintLen64+len(src))
	buf = append(buf, prefixEdge)
	buf = appendSegment(buf, src)
	return buf
}
```
For a whole-namespace scan (no segment to encode — the caller wants *every* node/file), the prefix is just the single byte: `[]byte{prefixNode}` / `[]byte{prefixFile}` — no `appendSegment` call needed, since there is no additional segment to bound.

**Archtest boundary (critical, applies to every new package):**
`internal/graphstore/archtest/import_graph_test.go` enforces that only `internal/graphstore` (and subpackages) import `github.com/cockroachdb/pebble/v2` (line 18-19, `allowedImporterPrefix`). `internal/query` and `internal/mcp` MUST import only `github.com/seanb4t/codegraph-go/internal/graphstore` (the `GraphStore`/`Reader`/`NodeIterator`/`FileIterator`/`EdgeIterator` interfaces) and `internal/schema` — never `pebble/v2` directly, or `TestNoPackageBypassesGraphStore` fails the build.

---

### `internal/query/engine.go`, `search.go`, `traverse.go` — read-only engine over `Reader`

**Analog:** `internal/indexer/symbolindex.go` (build-once in-memory map index, then query it) — same shape as D-04's reverse-adjacency builder.

**In-memory map-index-then-lookup pattern** (`symbolindex.go` lines 8-49, mirror for `buildReverseAdjacency`):
```go
type symbolIndex struct {
	byImportAndName map[string]map[string]string
}

func newSymbolIndex(results []goextract.FileResult) *symbolIndex {
	idx := &symbolIndex{byImportAndName: make(map[string]map[string]string, len(results))}
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		names := idx.byImportAndName[r.ImportPath]
		if names == nil {
			names = make(map[string]string)
			idx.byImportAndName[r.ImportPath] = names
		}
		for _, en := range r.Nodes {
			names[en.Node.Name] = en.Node.Id
		}
	}
	return idx
}
```
Apply the same "range once in a stable order, populate a `map[string][]T]`, expose narrow lookup methods (`resolveSelector`/`resolveUnqualified`)" shape to `buildReverseAdjacency(r graphstore.Reader) (map[string][]*schema.Edge, error)` — iterate `r.IterateEdges("")` once, key by `e.Target`, append. See RESEARCH Pattern 2 for the exact snippet (no in-repo analog exists for this specific function since it's new query-time logic — RESEARCH's snippet, cross-checked against `store.go`'s `IterateEdges` doc contract, is the concrete source to copy).

**Engine construction / orchestration shape** — model `Engine{reader graphstore.Reader}` with one method per command after `internal/indexer/pipeline.go`'s top-level `Run` orchestration style (multi-pass, single entry point delegating to sub-helpers) — read `pipeline.go` directly if the executor needs the exact orchestration idiom; not excerpted here since Phase 3's engine is read-only and simpler (no goroutine pool/worker orchestration to mirror, just sequential Reader calls).

---

### `internal/query/resolve.go` — `.codegraph/` resolution (D-01a)

**Analog:** `internal/cli/init.go`'s `targetRoot` + `codegraphDirName` constant (lines 14-16, 81-86).

```go
const codegraphDirName = ".codegraph"

func targetRoot(args []string) (string, error) {
	if len(args) > 0 {
		return filepath.Abs(args[0])
	}
	return os.Getwd()
}
```
`resolveCodegraphDir` (new, Pattern 4 in RESEARCH) extends this idiom with an upward walk — reuse `codegraphDirName`/`storeDirName` constants from `internal/cli` (or re-declare in `internal/query` if package placement puts the helper there per CONTEXT's "executor's discretion" note) and the existing `ErrNotInitialized` sentinel from `internal/cli/init.go` line 20-22:
```go
var ErrNotInitialized = errors.New("cli: not initialized")
```

---

### `internal/cli/{query,node,search,callers,callees,impact,affected,files,status,explore}.go` — thin Cobra commands

**Analog:** `internal/cli/index.go` (full file) — the canonical "resolve path → check `.codegraph/` exists → delegate to package function → print result" shape.

**Imports pattern** (lines 1-11):
```go
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)
```
For query commands, swap the last import for `"github.com/seanb4t/codegraph-go/internal/query"`.

**Command skeleton** (lines 20-83, the exact shape every new query command should follow — flags declared with `cmd.Flags().*Var(&x, "name", "n", default, "usage")`, `Args: cobra.MaximumNArgs(1)` for a `[path]`-taking command, `RunE` resolves root/dir then delegates):
```go
func newIndexCmd() *cobra.Command {
	var force, quiet, verbose bool
	var workers int

	cmd := &cobra.Command{
		Use:   "index [path]",
		Short: "Deterministically rebuild the graph from scratch",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(args)
			if err != nil {
				return err
			}

			codegraphDir := filepath.Join(root, codegraphDirName)
			if _, err := os.Stat(codegraphDir); os.IsNotExist(err) {
				return fmt.Errorf("%w: %s does not exist — run `codegraph init` first", ErrNotInitialized, codegraphDir)
			} else if err != nil {
				return err
			}
			// ... delegate to indexer.Run / query.Engine method here ...
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "rebuild without prompting for confirmation")
	return cmd
}
```
Query commands replace `-p`/`--path` as a flag (not positional arg, per D-01a — note the existing `init`/`index` use a positional `[path]` arg; the new query commands use `-p`/`--path` per CONTEXT D-01a, so the flag declaration differs: `cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")`).

**Output pattern** (`printSummary`, lines 101-111) — the `cmd.OutOrStdout()` convention every command must use instead of raw `fmt.Println`, so tests can capture output via `root.SetOut(&buf)`:
```go
out := cmd.OutOrStdout()
fmt.Fprintf(out, "files=%d nodes=%d edges=%d duration=%s\n", ...)
```
`--json` commands: `json.NewEncoder(cmd.OutOrStdout()).Encode(result)` instead.

**Root wiring** (`root.go` line 38, extend the existing `AddCommand` call):
```go
root.AddCommand(newInitCmd(), newIndexCmd(), newUninitCmd())
```
becomes
```go
root.AddCommand(newInitCmd(), newIndexCmd(), newUninitCmd(),
	newQueryCmd(), newNodeCmd(), newSearchCmd(), newCallersCmd(), newCalleesCmd(),
	newImpactCmd(), newAffectedCmd(), newFilesCmd(), newStatusCmd(), newExploreCmd(), newServeCmd())
```

**Confirm/interactive prompt idiom** (only relevant if `serve`/query commands ever need confirmation — unlikely, but note the pattern exists): `internal/cli/uninit.go` lines 69-78, `confirm(cmd, prompt)` using `cmd.InOrStdin()`/`cmd.OutOrStdout()`.

---

### `internal/cli/cli_test.go` — test scaffolding to extend

**Analog:** same file, existing helpers (lines 18-96) — reuse directly, do not reinvent.

```go
func copyFixture(t *testing.T) string { /* copies internal/indexer/testdata/gofixture into t.TempDir() */ }

func execCmd(args ...string) (stdout, stderr string, err error) {
	return execCmdWithInput("", args...)
}

func execCmdWithInput(input string, args ...string) (stdout, stderr string, err error) {
	root := newRootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetIn(strings.NewReader(input))
	root.SetArgs(args)
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}
```
Query-command tests should `copyFixture(t)` then `execCmd("init", dir)` to build a real index, then `execCmd("query", "main", "-p", dir, "--json")` and assert on stdout — exactly the `TestInitIndexUninit` subtest shape (lines 98-207), one `t.Run` per command/flag combination.

---

### `internal/mcp/server.go`, `tools.go` — stdio MCP server (D-08)

**No in-repo analog** — ground directly in RESEARCH Pattern 3 (Context7-verified `mcp-go` v0.56.0 API):
```go
func buildServer(hasIndex bool, allowlist map[string]bool) *server.MCPServer {
	s := server.NewMCPServer("codegraph", version, server.WithToolCapabilities(true))
	if !hasIndex {
		return s // MCP-03: zero tools when no .codegraph/ resolves
	}
	s.AddTool(exploreTool(), exploreHandler)
	for _, name := range []string{"node", "search", "callers", "callees", "impact", "files", "status"} {
		if allowlist[name] {
			s.AddTool(companionTool(name), companionHandler(name))
		}
	}
	return s
}
```
Tool handlers MUST call the same `internal/query.Engine` methods/formatters as the CLI commands (D-08b) — see `internal/cli/index.go`'s RunE body as the template for "resolve args → call engine → format" even though the transport differs (`req.RequireString("query")` instead of Cobra flags).

**Fresh-snapshot-per-call discipline (Pitfall 2):** each tool handler must call `store.Snapshot()` itself, not close over a snapshot captured at server startup — mirror `internal/graphstore/store_test.go`'s per-test `store.Snapshot()` / `defer r.Close()` pattern (see `cli_test.go`'s `readGraphCounts`, lines 76-96, for the exact `Open` → `Snapshot()` → `defer r.Close()` → `defer store.Close()` idiom to replicate inside each tool handler).

---

### `testdata/golden/golden_parity_test.go` — parity harness

**Analog:** `testdata/golden/golden_test.go` (full file read) — reuse `isVolatileKey`/`findVolatileKeys` directly rather than reimplementing volatile-field stripping:
```go
var volatileKeys = map[string]bool{
	"score":       true,
	"lastIndexed": true,
	"dbSizeBytes": true,
}

func isVolatileKey(key string) bool {
	if volatileKeys[key] {
		return true
	}
	if strings.HasSuffix(key, "_at") || strings.HasSuffix(key, "At") {
		return true
	}
	return false
}
```
The new parity test loads `testdata/golden/corpus/weft-go/{query,callers,callees,impact,status,explore,node}.json`, runs the equivalent `codegraph` command against a locally-indexed copy of the `weft-go` fixture (or the existing `internal/indexer/testdata/gofixture`, per D-05a/b's exact-shape requirement — confirm which fixture corpus the planner selects), decodes both sides, strips volatile keys via `findVolatileKeys`, and diffs remaining structure per D-05's normalization rules (ids excluded from comparison, `isAsync`/`isStatic`/`isAbstract` rendered `false`, etc.).

---

## Shared Patterns

### Path resolution / `.codegraph/` existence check
**Source:** `internal/cli/init.go` lines 79-86 (`targetRoot`), `index.go` lines 34-39 (existence check + `ErrNotInitialized`).
**Apply to:** Every new CLI command (`query`, `node`, `search`, `callers`, `callees`, `impact`, `affected`, `files`, `status`, `explore`) and the MCP server's own path-resolution logic (D-01a explicitly requires matching behavior).

### Cobra thin-command delegation
**Source:** `internal/cli/index.go` (whole file) — `RunE` parses flags/paths only, all logic lives in a delegated package call (`indexer.Run` → `query.Engine.<Method>`).
**Apply to:** All ten new CLI command files.

### Iterator-over-Pebble-range-scan
**Source:** `internal/graphstore/pebble_store.go` lines 139-196 (`IterateEdges`/`pebbleEdgeIterator`).
**Apply to:** `IterateNodes`/`IterateFiles` in the same file, and any `internal/query` code consuming them (`for it.Next() { ... }; if err := it.Err(); err != nil { ... }; it.Close()`).

### Build-once in-memory map index
**Source:** `internal/indexer/symbolindex.go` (whole file).
**Apply to:** `internal/query/traverse.go`'s `buildReverseAdjacency` (D-04) — same "populate a map in one pass, expose narrow query methods" shape.

### Archtest interface-boundary discipline
**Source:** `internal/graphstore/archtest/import_graph_test.go` (whole file).
**Apply to:** `internal/query` and `internal/mcp` package design — never import `github.com/cockroachdb/pebble/v2`; import only `internal/graphstore`'s exported interfaces and `internal/schema`.

### Test scaffolding (`execCmd`/`copyFixture`)
**Source:** `internal/cli/cli_test.go` lines 18-96.
**Apply to:** Every new CLI command's tests, and `internal/mcp/server_test.go` (black-box exercise via the same fixture-copy convention, adapted to drive the MCP tool-call path instead of Cobra `RunE`).

### `cmd.OutOrStdout()` / `cmd.SetOut` output discipline
**Source:** `internal/cli/index.go` line 105 (`printSummary`), `cli_test.go` lines 63-64 (`root.SetOut(&outBuf)`).
**Apply to:** Every new CLI command — never write to `os.Stdout` directly, always via the Cobra-provided writer, so tests can capture output.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `internal/mcp/server.go` | provider | event-driven (stdio) | No existing MCP/JSON-RPC server in this codebase; use RESEARCH Pattern 3 (mark3labs/mcp-go v0.56.0 Context7-verified API) as the authoritative source instead |
| `internal/query/render_markdown.go` | utility | transform | No markdown-templating code exists yet; the golden fixture strings (`testdata/golden/corpus/weft-go/explore.json`, `node.json`) are themselves the literal template spec — copy the exact interpolation structure documented in RESEARCH's "Code Examples" section (D-05a/D-05b), not an in-repo analog |
| `internal/query/traverse.go`'s `buildReverseAdjacency`/BFS | service | transform | New query-time logic with no prior in-repo implementation; RESEARCH Pattern 2's snippet (cross-checked against `store.go`'s `IterateEdges` contract) is the concrete source |

## Metadata

**Analog search scope:** `internal/graphstore/`, `internal/cli/`, `internal/indexer/`, `internal/schema/`, `testdata/golden/`
**Files scanned:** `store.go`, `pebble_store.go`, `keys.go`, `archtest/import_graph_test.go`, `root.go`, `init.go`, `index.go`, `uninit.go`, `cli_test.go`, `symbolindex.go`, `golden_test.go`, `graph.pb.go` (Node/Edge/File/Meta shapes)
**Pattern extraction date:** 2026-07-11
