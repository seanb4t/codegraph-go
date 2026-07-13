# Phase 4: Incremental Sync & File Watcher - Pattern Map

**Mapped:** 2026-07-11
**Files analyzed:** 16
**Analogs found:** 16 / 16

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `internal/graphstore/keys.go` (extend, `x/` namespace builders) | storage/key-encoder | CRUD (index maintenance) | same file, `fileKey`/`fileSubgraphPrefix`/`edgeKey` builders | exact (same file, add alongside) |
| `internal/graphstore/store.go` (extend `Reader`/`Writer` interfaces) | storage interface | CRUD | same file, existing `IterateNodes`/`DeleteFileSubgraph` method docs | exact |
| `internal/graphstore/batch.go` (extend `pebbleWriter`) | storage/service | CRUD | same file, `PutNode`/`PutEdge`/`DeleteFileSubgraph` | exact |
| `internal/graphstore/pebble_store.go` (extend `pebbleReader`, new iterator) | storage/service | CRUD (streaming iteration) | same file, `pebbleNodeIterator`/`IterateNodes` | exact |
| `internal/graphstore/keyenc_test.go` (extend) | test | transform (collision-safety) | same file, `TestKeyEncodingRejectsDelimiterInjection` | exact |
| `internal/schema/graph.proto` + `graph.pb.go` (additive fields) | model/schema | CRUD | same file, `File.content_hash`/`Meta.last_sync_unix_ms` additive fields | exact |
| `internal/indexer/sync.go` (NEW) | service/orchestrator | batch (incremental ETL) | `internal/indexer/pipeline.go` `Run`/`run`/`Stats`/`Options` | exact (structural twin) |
| `internal/indexer/resolve.go` (extract `resolveRefs`'s idx param) | service | batch | same file, `resolveRefs`/`writeGraph`/`collapseEdges` | exact |
| `internal/indexer/symbolindex.go` (new `newSymbolIndexFromStore`) | service | transform (store→index) | same file, `newSymbolIndex` | exact |
| `internal/indexer/discover.go` (export `shouldSkipDir`) | utility | file-I/O | same file, inline skip predicate in `Discover`'s `WalkDir` callback | exact |
| `internal/watch/watcher.go`, `internal/watch/debounce.go` (NEW package) | service (event-driven) | event-driven | no direct analog; nearest lifecycle shape: `internal/graphstore/pebble_store.go` `Open`/`Close` (open-then-Close-once discipline) | no analog (new concern) |
| `internal/daemon/daemon.go`, `internal/daemon/lock.go` (NEW package) | service/CLI-support | event-driven + file-I/O | `internal/cli/init.go`/`uninit.go` (`.codegraph/` dir lifecycle, `os.Stat`/`os.RemoveAll` guard pattern) | role-match |
| `internal/cli/sync.go` (NEW) | controller/CLI command | request-response (batch trigger) | `internal/cli/index.go` (`newIndexCmd`) | exact |
| `internal/cli/daemon.go` (NEW) | controller/CLI command | event-driven (long-running) | `internal/cli/serve.go` (`newServeCmd`, long-running server command) | role-match |
| `internal/cli/unlock.go` (NEW) | controller/CLI command | CRUD (lockfile) | `internal/cli/uninit.go` (`newUninitCmd`, guarded destructive op + `--force`) | exact |
| `internal/cli/status.go` (modify: staleness field) | controller/CLI command | request-response | same file (already exists, extend) | exact |
| `internal/cli/root.go` (wire new commands) | config/wiring | — | same file, `root.AddCommand(...)` | exact |
| `internal/query/status.go` (staleness signal live) | service/formatter | transform | same file, `StatusResult` (already has inert placeholder fields per doc comment) | exact |
| `internal/query/render_markdown.go` (banner prepend) | service/formatter | transform | not read this pass — locate `RenderExplore`; same package/file convention as `status.go` | role-match |
| `internal/query/traverse.go` (export `buildReverseAdjacency`) | service | graph traversal | same file, existing unexported function — just export | exact |
| `internal/mcp/*` + `internal/cli/serve.go` (D-06 reconcile wiring) | controller/wiring | request-response | `internal/cli/serve.go` `newServeCmd` (`RunE`, `query.ResolveCodegraphDir` idiom) | exact |

## Pattern Assignments

### `internal/graphstore/keys.go` (extend — `x/` file-index namespace)

**Analog:** same file, `fileKey`/`fileSubgraphPrefix`/`edgeKey`/`appendSegment`/`rangeUpperBound` (`internal/graphstore/keys.go:11-148`)

**Namespace-prefix discipline** (lines 11-24):
```go
const (
	prefixMeta byte = 'm'
	prefixNode byte = 'n'
	prefixEdge byte = 'e'
	prefixFile byte = 'f'
	prefixAnnotation byte = 'a' // RESERVED — do not collide with this
)
```
Add `prefixFileIndex byte = 'x'` alongside these, with a doc comment mirroring `prefixAnnotation`'s "reserved namespace" framing — RESEARCH's Pattern 2 gives the exact byte and key-builder shapes (`fileIndexNodeKey`/`fileIndexEdgeKey`/`fileIndexPrefix`) to copy in.

**Segment-safety discipline to copy exactly** (lines 26-44, `appendSegment`):
```go
func appendSegment(buf []byte, seg string) []byte {
	var lenBuf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(lenBuf[:], uint64(len(seg)))
	buf = append(buf, lenBuf[:n]...)
	buf = append(buf, seg...)
	return buf
}
```
Every new `x/` key builder MUST reuse this exact primitive (length-prefixed segments), never raw string concatenation — this is the collision-safety property `keyenc_test.go` gates.

**Range-delete prefix/bound pairing to mirror** (lines 104-120, `fileSubgraphPrefix` + doc comment naming this *"the Phase 4 hook"*):
```go
func fileSubgraphPrefix(path string) []byte {
	return fileKey(path)
}
```
New `fileIndexPrefix(path)` follows the identical shape (prefix byte + `appendSegment(path)`), paired with the existing `rangeUpperBound` helper (lines 131-148) unchanged — no new range-math needed.

---

### `internal/graphstore/store.go` (extend `Reader`/`Writer` interfaces)

**Analog:** same file — `Writer.DeleteFileSubgraph` doc (lines 148-151), `Reader.IterateNodes` doc (lines 56-62)

**Interface method doc-comment convention to copy:**
```go
// DeleteFileSubgraph stages a single range-delete over path's own
// file record (D-03) — the mechanism Phase 4's rename/delete pruning
// binds to.
DeleteFileSubgraph(path string) error
```
New `Writer.DeleteNode(id string) error` / `Writer.DeleteEdge(source, kind, target string) error` and `Reader.IterateFileIndex(path string) (FileIndexIterator, error)` should each get a doc comment in this same voice: state the storage mechanism, cite the decision id (D-02), and state the caller/motivation. Follow `EdgeIterator`/`NodeIterator`/`FileIterator`'s exact three-method shape (`Next() bool` / accessor / `Err() error` / `Close() error`) for the new `FileIndexIterator` (lines 72-127).

---

### `internal/graphstore/batch.go` (extend `pebbleWriter`)

**Analog:** same file, `pebbleWriter.PutNode`/`PutEdge`/`DeleteFileSubgraph`/`Commit` (`internal/graphstore/batch.go:20-85`)

**Put pattern to mirror for any new staged mutation:**
```go
func (w *pebbleWriter) PutNode(n *schema.Node) error {
	data, err := proto.Marshal(n)
	if err != nil {
		return err
	}
	return w.batch.Set(nodeKey(n.GetId()), data, nil)
}
```
`PutNode`/`PutEdge` should each grow one extra `w.batch.Set(fileIndexNodeKey(...), ...)` / `w.batch.Set(fileIndexEdgeKey(...), ...)` call staged alongside the existing one (RESEARCH Pattern 2's "PutEdge gains ownerPath param" — thread `ownerPath` through from `resolve.go`'s `writeGraph`, which already computes `nodeFilePath[e.Source]`).

**Range-delete pattern to extend (not replace):**
```go
func (w *pebbleWriter) DeleteFileSubgraph(path string) error {
	start := fileSubgraphPrefix(path)
	end := rangeUpperBound(start)
	return w.batch.DeleteRange(start, end, nil)
}
```
Add a second `w.batch.DeleteRange(fileIndexPrefix(path), rangeUpperBound(fileIndexPrefix(path)), nil)` call inside this same method — still one logical Writer call from the caller's perspective (RESEARCH Pattern 2's explicit instruction).

**New point-delete methods (no existing analog in this file — model on `PutNode`'s marshal-then-batch-op shape, but `Delete` not `Set`):**
```go
func (w *pebbleWriter) DeleteNode(id string) error {
	return w.batch.Delete(nodeKey(id), nil)
}
func (w *pebbleWriter) DeleteEdge(source, kind, target string) error {
	return w.batch.Delete(edgeKey(source, kind, target), nil)
}
```

**Error/Close discipline to copy exactly** (lines 62-85, `Commit`/`Close`) — unchanged, no new pattern needed; new methods stage onto the same `w.batch`.

---

### `internal/graphstore/pebble_store.go` (extend `pebbleReader`, new `pebbleFileIndexIterator`)

**Analog:** same file, `pebbleReader.IterateNodes` (lines 162-171) + `pebbleNodeIterator` (lines 231-268)

**Iterate-open pattern to mirror:**
```go
func (r *pebbleReader) IterateNodes() (NodeIterator, error) {
	iter, err := r.snap.NewIter(&pebble.IterOptions{
		LowerBound: []byte{prefixNode},
		UpperBound: rangeUpperBound([]byte{prefixNode}),
	})
	...
}
```
(exact body not re-read this pass — same construction idiom visible at `IterateEdges`/`IterateFiles`, lines 139-184: `r.snap.NewIter` bounded by `[prefix, rangeUpperBound(prefix))`.) New `IterateFileIndex(path)` bounds its iterator to `[fileIndexPrefix(path), rangeUpperBound(fileIndexPrefix(path)))` — same idiom, narrower bound.

**Iterator adapter pattern to mirror exactly** (lines 231-268, `pebbleNodeIterator`):
```go
type pebbleNodeIterator struct {
	iter    *pebble.Iterator
	started bool
	cur     *schema.Node
	err     error
}

func (it *pebbleNodeIterator) Next() bool {
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
	var n schema.Node
	if err := proto.Unmarshal(it.iter.Value(), &n); err != nil {
		it.err = err
		return false
	}
	it.cur = &n
	return true
}
```
New `pebbleFileIndexIterator` follows this identically, but its `Next()` decodes a typed entry (marker byte 0x01 node / 0x02 edge → `IsNode bool; NodeID string; Source, Kind, Target string`) from the key bytes rather than unmarshaling a proto value — the `x/` index stores no payload, only the key encodes the reference (RESEARCH Pattern 2's `fileIndexKindNode`/`fileIndexKindEdge` marker bytes).

---

### `internal/graphstore/keyenc_test.go` (extend — collision-safety for `x/`)

**Analog:** same file, `TestKeyEncodingRejectsDelimiterInjection` (`internal/graphstore/keyenc_test.go:26-70`)

Reuse `adversarialSegments` (lines 12-24) directly; add a `t.Run("distinct file-index node/edge entries never collide", ...)` subtest following the exact same shape as the "distinct node ids never collide" subtest (lines 32-42) and the "no crafted key falls inside another namespace's range" subtest (lines 58+), substituting `fileIndexNodeKey`/`fileIndexEdgeKey` for `nodeKey`.

---

### `internal/schema/graph.proto` + `graph.pb.go` (additive fields)

**Analog:** same file — `File.content_hash` (field 2) and `Meta.last_sync_unix_ms` (field 4) doc comments (`internal/schema/graph.proto:99-132`), plus Node's `reserved 50 to 59` pattern (line 65)

**Additive-field doc-comment convention to copy exactly:**
```protobuf
message File {
  string path = 1;
  string content_hash = 2;
  string language = 3;
  int64 node_count = 4;
  int64 edge_count = 5;
  repeated string errors = 6;
}
```
Add `int64 mtime_unix_ns = 7;` and `int64 size_bytes = 8;` — next unused field numbers, below `File`'s (currently absent, but Node's precedent at line 65 shows the convention) reserved range; give each a short doc comment stating decision id (D-01a) and writer (`index`+`sync`), matching `errors`'s comment style (lines 113-117).

Add `bool has_file_index = 7;` to `Meta` (next number after `health_message = 6`) for D-02b's backfill-detection flag, with a doc comment in `last_sync_unix_ms`'s voice (line 129 area).

**`SchemaVersion` non-bump rule to cite** (`internal/schema/meta.go:6-11`):
```go
// Additive-only discipline (D-02a): within a single SchemaVersion, fields
// on Node, Edge, File, and Meta are NEVER renumbered or reused.
```
No change to `meta.go` itself — cite this comment as the authority for "SchemaVersion stays 1" in the sync.go/keys.go doc comments (D-02's own text says exactly this).

---

### `internal/indexer/sync.go` (NEW — `Sync()` entry point)

**Analog:** `internal/indexer/pipeline.go` — `Run`/`run`/`Options`/`Stats`/`readGraphCounts` (`internal/indexer/pipeline.go:1-131`)

**Signature/lifecycle pattern to mirror exactly:**
```go
func Run(repoRoot, storeDir string, opts Options) (Stats, error) {
	return run(repoRoot, storeDir, opts, Resolve)
}

func run(repoRoot, storeDir string, opts Options, resolve resolveFunc) (Stats, error) {
	start := time.Now()
	files, modulePath, err := Discover(repoRoot)
	...
	store, err := graphstore.Open(storeDir)
	if err != nil { ... }
	defer store.Close()
	...
}
```
`Sync(repoRoot, storeDir string, opts Options) (Stats, error)` should follow this identical shape: `time.Now()` start, `Discover` for the cheap walk, `graphstore.Open` once with `defer store.Close()` on every path, ending with a `Stats` struct literal — but internally replacing the single `resolve(store, results, modulePath)` call with the multi-step algorithm RESEARCH Pattern 1 lays out (stat pre-filter → hash confirm → prune via `x/` index → `query.BuildReverseAdjacency` dependent scan → `Extract` the bounded batch → `newSymbolIndexFromStore` → resolve → one `Commit`).

**`Stats` field-extension pattern** (lines 26-38): add `FilesPruned`, `NodesRemoved`, `EdgesRemoved`, `DependentsRecomputed int` fields the same additive way `Skipped`/`Unresolved` were added alongside `Files`/`Nodes`/`Edges` — a flat struct, no nesting, each field a plain summary count (D-01b's summary line).

**`readGraphCounts`-style post-commit re-derivation to reuse verbatim** (lines 115-131) — `Sync()` can call the exact same unexported `readGraphCounts(store)` helper `Run` uses; no new counting logic needed.

---

### `internal/indexer/resolve.go` (extract store-seeded index injection point)

**Analog:** same file — `resolveRefs`, `writeGraph`, `collapseEdges` (`internal/indexer/resolve.go:39-134, 236-292`)

**Symbol-index-building line to change (the seam):**
```go
func resolveRefs(results []goextract.FileResult, modulePath string) (nodes, packageNodes []*schema.Node, edges []*schema.Edge, files []*schema.File, unresolvedCount int) {
	idx := newSymbolIndex(results)   // <-- Sync() needs to inject a different idx here
	...
```
Refactor so `idx` is a parameter (or extract the per-ref `switch ref.Kind` resolution loop, lines 52-117, into a helper taking an externally supplied `*symbolIndex`) — `Resolve()` (full, from-scratch) keeps calling `newSymbolIndex(results)` unchanged; `Sync()` calls the new `newSymbolIndexFromStore(...)` and passes the merged index in. This is a **surgical signature change**, not a rewrite — every other line of `resolveRefs`/`writeGraph`/`collapseEdges` is reused as-is by `Sync()`.

**`writeGraph`'s single-writer commit discipline to reuse unchanged** (lines 242-292): `Sync()`'s final write step should call this exact function (or a narrow variant threading `ownerPath` into `PutEdge`, per the batch.go change above) — one `store.NewWriter()`, staged Puts, one `w.Commit()`. Never per-symbol commits (D-01b).

---

### `internal/indexer/symbolindex.go` (new `newSymbolIndexFromStore`)

**Analog:** same file, `newSymbolIndex` (`internal/indexer/symbolindex.go:19-49`)

**Exact skeleton to build from (already drafted in RESEARCH, cite it):**
```go
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
			if en.Node.Kind == goextract.KindFile {
				continue
			}
			names[en.Node.Name] = en.Node.Id
		}
	}
	return idx
}
```
`newSymbolIndexFromStore(r graphstore.Reader, modulePath string, exclude map[string]bool) (*symbolIndex, error)` mirrors this shape exactly but iterates `r.IterateNodes()` instead of `results`, skips `goextract.KindFile` AND `kindPackage` (the resolve.go pseudo-node kind, `internal/indexer/resolve.go:23`), skips any node whose `FilePath` is in `exclude` (the reparse batch, about to be superseded), and computes `importPath` via `importPathFor(modulePath, n.FilePath)` instead of reading `r.ImportPath` off a `FileResult`. `resolveSelector`/`resolveUnqualified` (lines 51-83) need **zero changes** — they operate on `byImportAndName` regardless of how it was populated.

---

### `internal/indexer/discover.go` (export `shouldSkipDir`)

**Analog:** same file, inline predicate inside `Discover`'s `WalkDir` callback (`internal/indexer/discover.go:54-63`)

```go
walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
		if p != root && (d.Name() == "vendor" || strings.HasPrefix(d.Name(), ".")) {
			return fs.SkipDir
		}
		return nil
	}
	...
```
Extract the inner boolean condition into `func shouldSkipDir(name string) bool { return name == "vendor" || strings.HasPrefix(name, ".") }`, call it from this same site (`if p != root && shouldSkipDir(d.Name())`), and export/share it so `internal/watch`'s `addRecursive` walker calls the identical predicate (RESEARCH Pattern 3's explicit "watcher and discoverer must never silently diverge" requirement). This directory already dot-prefix-excludes `.codegraph/` — no special-case needed for it.

---

### `internal/watch/watcher.go` + `internal/watch/debounce.go` (NEW package)

**No direct analog** — this is a genuinely new concern (native OS event consumption). Nearest structural precedent for open/close lifecycle discipline:

**Open-then-Close-once pattern to borrow the *shape* of** (`internal/graphstore/pebble_store.go:43-82`):
```go
func Open(dir string) (GraphStore, error) {
	db, err := pebble.Open(dir, &pebble.Options{})
	if err != nil {
		return nil, err
	}
	return &pebbleStore{db: db}, nil
}

func (s *pebbleStore) Close() error {
	if s.closed.Swap(true) {
		return nil
	}
	return s.db.Close()
}
```
Apply the same "idempotent Close via `atomic.Bool` swap" idiom to the watcher's lifecycle type, since `fsnotify.Watcher.Close()` closing twice can also misbehave — guard it the same way `pebbleStore.Close` guards `s.db.Close()`.

RESEARCH's own Pattern 3 code block (`addRecursive`/`watchLoop`/`debounceDuration`) is the concrete skeleton to implement directly — it is this research's synthesis (no upstream copy source exists in-repo), verified against fsnotify's official docs. Use `internal/indexer/discover.go`'s exported `shouldSkipDir` (see above) as the `skip` predicate passed into `addRecursive`, not a re-derived one.

---

### `internal/daemon/daemon.go` + `internal/daemon/lock.go` (NEW package)

**Analog:** `internal/cli/init.go` / `internal/cli/uninit.go` — `.codegraph/` directory lifecycle + guarded destructive-op pattern (`internal/cli/init.go:44-57`, `internal/cli/uninit.go:32-53`)

**Stat-then-act existence-check pattern to mirror:**
```go
codegraphDir := filepath.Join(root, codegraphDirName)
if _, err := os.Stat(codegraphDir); err == nil {
	return fmt.Errorf("%w: %s already exists — use ...", ErrAlreadyInitialized, codegraphDir)
} else if !os.IsNotExist(err) {
	return err
}
```
`lock.go`'s stale-lock check follows the identical `os.Stat`-then-branch shape: stat the lockfile, if absent proceed to acquire; if present, decode it and check `isStale` (RESEARCH Pattern 6's `isStale` skeleton, `os.FindProcess`/`proc.Signal(syscall.Signal(0))`) before deciding to error or clear.

**Guarded-destructive-op + `--force` pattern to mirror for `unlock`** (`internal/cli/uninit.go:40-49`, `confirm` helper lines 64-78):
```go
if !force {
	ok, err := confirm(cmd, fmt.Sprintf("Remove %s?", codegraphDir))
	...
}
if err := os.RemoveAll(codegraphDir); err != nil {
	return err
}
```
`codegraph unlock` should reuse the existing `confirm` helper (`internal/cli/uninit.go:69-78`) verbatim when the lock is live-but-user-insists (though per D-05, unlock should refuse outright on a **live** lock, only silently clearing a **stale** one — no confirm prompt needed for the stale case, matching D-05's "only removes if stale" language).

---

### `internal/cli/sync.go` (NEW — thin Cobra command)

**Analog:** `internal/cli/index.go` — `newIndexCmd` (`internal/cli/index.go:20-83`)

**Thin-delegation pattern to mirror exactly:**
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

			storeDir := filepath.Join(codegraphDir, storeDirName)
			stats, err := indexer.Run(root, storeDir, indexer.Options{
				Workers: workers, Verbose: verbose, Quiet: quiet,
			})
			if err != nil {
				return err
			}
			printSummary(cmd, stats, quiet, verbose)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress progress and summary output")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "emit per-file/per-pass detail")
	cmd.Flags().IntVar(&workers, "workers", 0, "bound the extraction worker pool (default: number of CPUs)")
	return cmd
}
```
`newSyncCmd` follows this identically: same `ErrNotInitialized` guard (index's `.codegraph/`-must-exist check, not init's must-NOT-exist check), same `targetRoot(args)`/`storeDir := filepath.Join(codegraphDir, storeDirName)` resolution, same `--quiet`/`--verbose` flags (D-01b), but calls `indexer.Sync(root, storeDir, indexer.Options{...})` instead of `indexer.Run`. Reuse `printSummary` (`internal/cli/init.go:101-111`) or extend it additively for the new `Stats` fields (files pruned, dependents recomputed) — do not fork a second summary printer.

---

### `internal/cli/daemon.go` (NEW — thin Cobra command)

**Analog:** `internal/cli/serve.go` — `newServeCmd` (`internal/cli/serve.go:28-67`)

**Long-running-server command pattern to mirror:**
```go
func newServeCmd() *cobra.Command {
	var path string
	var mcpMode bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the codegraph MCP server",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := resolveStartPath(path)
			if err != nil {
				return err
			}
			repoPath := start
			hasIndex := false
			if dir, err := query.ResolveCodegraphDir(start); err == nil {
				hasIndex = true
				repoPath = dir
			} else if !errors.Is(err, query.ErrNotInitialized) {
				return err
			}
			...
			return server.ServeStdio(s)
		},
	}
	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	return cmd
}
```
`newDaemonCmd` mirrors the `resolveStartPath`/`query.ResolveCodegraphDir` path-resolution idiom, then blocks on the daemon's own run loop (`daemon.Run(ctx)`, analogous to `server.ServeStdio(s)` blocking) instead of the MCP server. `--path` flag reused verbatim.

---

### `internal/cli/unlock.go` (NEW — thin Cobra command)

**Analog:** `internal/cli/uninit.go` — `newUninitCmd` (`internal/cli/uninit.go:19-62`)

**Guarded-existence-check + `--force`-optional pattern to mirror:**
```go
codegraphDir := filepath.Join(root, codegraphDirName)
if _, err := os.Stat(codegraphDir); os.IsNotExist(err) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s does not exist — nothing to do\n", codegraphDir)
	return nil
} else if err != nil {
	return err
}
```
`newUnlockCmd` follows this "absent lock → clean no-op message, not an error" idiom for a missing lockfile, then calls into `daemon.Unlock(lockPath)` (D-05's "only removes if stale — never blindly deletes... errors clearly if the lock is live"), printing a `"removed stale lock (pid=%d)"` / `"daemon still running, pid=%d — stop it first"` message in the same `fmt.Fprintf(cmd.OutOrStdout(), ...)` voice `uninit.go` uses at line 54.

---

### `internal/cli/status.go` (modify — staleness field)

**Analog:** same file, existing `newStatusCmd` (`internal/cli/status.go:15-60`)

```go
result, err := eng.Status()
if err != nil {
	return err
}
if jsonOut {
	data, err := query.MarshalStatusJSON(result)
	...
}
out := cmd.OutOrStdout()
fmt.Fprintf(out, "backend=%s files=%d nodes=%d edges=%d state=%s reindexRecommended=%t\n",
	result.Backend, result.FileCount, result.NodeCount, result.EdgeCount,
	result.Index.State, result.Index.ReindexRecommended)
```
Add `result.Stale`/`result.PendingSync` (whatever field name D-04a's `StatusResult` extension lands on — see `internal/query/status.go` below) to this same `Fprintf` line and to `MarshalStatusJSON`'s output — additive field on the existing struct/format line, not a new output mode.

---

### `internal/cli/root.go` (wire new commands)

**Analog:** same file, `root.AddCommand(...)` (`internal/cli/root.go:38-41`)

```go
root.AddCommand(newInitCmd(), newIndexCmd(), newUninitCmd(),
	newQueryCmd(), newSearchCmd(), newCallersCmd(), newCalleesCmd(),
	newImpactCmd(), newAffectedCmd(), newFilesCmd(), newStatusCmd(),
	newNodeCmd(), newExploreCmd(), newServeCmd())
```
Add `newSyncCmd(), newDaemonCmd(), newUnlockCmd()` to this same call — no other change to `root.go` needed. Package doc comment (lines 1-7) should get one added sentence naming `sync`/`daemon`/`unlock` alongside `init`/`index`/`uninit`.

---

### `internal/query/status.go` (staleness signal live)

**Analog:** same file — `StatusResult`'s already-documented "Phase-4 sync concept, present-but-inert placeholder" fields (`PendingChanges{Added,Modified,Removed}`, `WorktreeMismatch *string`, per RESEARCH Pattern 4's citation of this file's own doc comment)

Read `internal/query/status.go` directly before implementing — RESEARCH already names the exact fields to make live. Follow the existing `Status()` method's construction pattern (build a `StatusResult` struct literal from `eng.reader` scans) and add the mtime-vs-`last_sync_unix_ms` comparison RESEARCH Pattern 4 specifies as the no-daemon fallback signal.

---

### `internal/query/traverse.go` (export `buildReverseAdjacency`)

**Analog:** same file, the function itself (`internal/query/traverse.go:14-46`)

```go
func buildReverseAdjacency(r graphstore.Reader) (map[string][]*schema.Edge, error) {
	it, err := r.IterateEdges("")
	if err != nil {
		return nil, err
	}
	defer it.Close()

	rev := make(map[string][]*schema.Edge)
	for it.Next() {
		e := it.Edge()
		if e.Kind != goextract.RefKindCalls {
			continue
		}
		rev[e.Target] = append(rev[e.Target], e)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	return rev, nil
}
```
Rename to `BuildReverseAdjacency` (capital B — the only change needed to make it importable), update every in-package call site (`Callers`/`Impact`/`Affected`, unread this pass but named in the doc comment) to the new name. Doc comment already states "built fresh inside every caller — no package-level cache" (lines 22-26); `Sync()` must follow this exact same discipline: call it fresh, once, per `Sync()` invocation against its own snapshot `r0` (RESEARCH Pattern 1 step 7) — never cache it across syncs.

---

### `internal/mcp/*` + `internal/cli/serve.go` (D-06 reconnect reconcile)

**Analog:** `internal/cli/serve.go`, `newServeCmd`'s `RunE` (`internal/cli/serve.go:36-59`)

```go
start, err := resolveStartPath(path)
if err != nil {
	return err
}
repoPath := start
hasIndex := false
if dir, err := query.ResolveCodegraphDir(start); err == nil {
	hasIndex = true
	repoPath = dir
} else if !errors.Is(err, query.ErrNotInitialized) {
	return err
}
...
s := mcp.BuildServer(hasIndex, allowlist, repoPath)
return server.ServeStdio(s)
```
Insert the `indexer.Sync(repoPath, storeDir, opts)` call (RESEARCH Pattern 5) between the `hasIndex`/`repoPath` resolution block and `mcp.BuildServer(...)` — only when `hasIndex` is true (an uninitialized repo has nothing to reconcile, matching this function's existing "MCP-03: absent .codegraph/ is not an error" branch at lines 47-53). No new code path in `internal/mcp` itself — the reconcile is a `serve.go`-level call, same `Sync()` entry `sync`/`daemon` use (D-06's explicit "no second reconciliation code path").

## Shared Patterns

### Storage-boundary discipline (archtest)
**Source:** `internal/graphstore` package doc comment, `internal/graphstore/store.go:9-12`
**Apply to:** `internal/indexer/sync.go`, `internal/watch/*`, `internal/daemon/*`, `internal/cli/sync.go`/`daemon.go`/`unlock.go`
```go
// GraphStore is the only door onto the embedded key-value engine (D-04).
// No package outside internal/graphstore (and its own subpackages) may
// import the engine directly — archtest.TestNoPackageBypassesGraphStore
// enforces this boundary at test time (D-04a).
```
Every new package (watch, daemon, sync.go) MUST depend only on `graphstore.GraphStore`/`Reader`/`Writer` interfaces — never `github.com/cockroachdb/pebble/v2` directly. The `x/` secondary index lives entirely inside `internal/graphstore` (D-02).

### Single-writer commit discipline
**Source:** `internal/indexer/resolve.go:236-292` (`writeGraph`), `internal/graphstore/store.go:129-131` (`Writer` doc)
**Apply to:** `internal/indexer/sync.go`, `internal/daemon/daemon.go`
```go
w, err := store.NewWriter()
if err != nil {
	return err
}
for _, n := range allNodes {
	if err := w.PutNode(n); err != nil {
		w.Close()
		return err
	}
}
...
return w.Commit()
```
Never per-symbol commits (D-01b/D-04a). `Sync()`'s prune-then-write steps land in ONE `Writer`/one `Commit()` (RESEARCH Pattern 1 step 12). The daemon holds exactly one `GraphStore.Writer` at a time across the whole process (INDX-05, D-05).

### Thin CLI → engine delegation
**Source:** `internal/cli/index.go` (`newIndexCmd`), package doc comment `internal/cli/root.go:1-6`
**Apply to:** `internal/cli/sync.go`, `daemon.go`, `unlock.go`
```go
// Package cli implements the codegraph command-line interface ...
// Commands contain no extraction/resolution logic of their own — they
// resolve paths, manage the .codegraph/ directory layout (D-01b), and
// delegate all indexing work to indexer.Run.
```
No logic in the Cobra `RunE` beyond: resolve path/flags, check `.codegraph/` existence with the `ErrNotInitialized`/`ErrAlreadyInitialized` sentinel idiom, delegate to the engine package (`indexer.Sync`, `daemon.Run`, `daemon.Unlock`), print via a shared summary helper.

### Error-sentinel + `%w`-wrapped guidance messages
**Source:** `internal/cli/root.go:15-22` (`ErrAlreadyInitialized`/`ErrNotInitialized`), used at `internal/cli/index.go:36` and `internal/cli/init.go:46`
**Apply to:** `internal/cli/sync.go`, `internal/daemon/lock.go`
```go
return fmt.Errorf("%w: %s does not exist — run `codegraph init` first", ErrNotInitialized, codegraphDir)
```
`sync`/`daemon` reuse the existing `ErrNotInitialized` sentinel (no new error type needed — same "not initialized" condition). `unlock`'s "daemon still running" case should follow the identical `%w`-wrapped, actionable-guidance-in-the-message convention (introduce a `daemon.ErrLockLive` sentinel analogous in spirit to `cli.ErrNotInitialized`).

### Additive-only schema evolution
**Source:** `internal/schema/meta.go:6-11`, `internal/schema/graph.proto:1-9` (package doc comment)
**Apply to:** `internal/schema/graph.proto` (`File.mtime_unix_ns`/`size_bytes`, `Meta.has_file_index`)
```go
// Additive-only discipline (D-02a): within a single SchemaVersion, fields
// on Node, Edge, File, and Meta are NEVER renumbered or reused.
```
Next available field numbers only; `SchemaVersion` stays `1`; no `reserved` range disturbed.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/watch/watcher.go` | service | event-driven | No fsnotify/native-watch code exists anywhere in the codebase yet — genuinely new concern (SYNC-01). Use RESEARCH Pattern 3's `addRecursive`/`watchLoop` skeleton directly; it is this research's own synthesis against the verified fsnotify API, not derived from an in-repo analog. |
| `internal/watch/debounce.go` | utility | event-driven | Same — no existing timer/debounce code in the codebase. RESEARCH Pattern 3's `debounceDuration`/`time.AfterFunc` skeleton is the concrete starting point. |
| `internal/daemon/lock.go` (pid-liveness check) | utility | file-I/O | No pidfile/process-liveness code exists in-repo. RESEARCH Pattern 6's `isStale`/`os.FindProcess`/`syscall.Signal(0)` skeleton (stdlib-only) is the concrete starting point; document the PID-namespace-reuse caveat RESEARCH flags. |

## Metadata

**Analog search scope:** `internal/graphstore/`, `internal/indexer/`, `internal/cli/`, `internal/query/`, `internal/schema/`, `internal/mcp/` (existing Phase 1-3 packages)
**Files scanned:** `keys.go`, `store.go`, `batch.go`, `pebble_store.go`, `keyenc_test.go`, `graph.proto`, `meta.go`, `pipeline.go`, `resolve.go`, `symbolindex.go`, `discover.go`, `index.go`, `init.go`, `uninit.go`, `serve.go`, `status.go`, `root.go`, `traverse.go` (18 files read directly)
**Pattern extraction date:** 2026-07-11
