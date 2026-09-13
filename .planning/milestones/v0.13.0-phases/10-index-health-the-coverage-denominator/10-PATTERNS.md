# Phase 10: Index Health — The Coverage Denominator - Pattern Map

**Mapped:** 2026-09-12
**Files analyzed:** 15 (13 modified, 2 new modules with sub-files)
**Analogs found:** 15 / 15 (all files have an in-repo precedent; RESEARCH.md already did the heavy verification — this document translates that into planner-consumable analog assignments)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/schema/graph.proto` (`ExcludedFile`, `ExclusionReason`, `Meta.has_coverage=9`) | model/schema | CRUD (additive schema evolution) | itself — `Meta.has_file_index=7`/`commit_sha=8` additions | exact |
| `internal/graphstore/keys.go` (new prefix byte + key builders) | model/storage | CRUD (key encode/decode) | `prefixFileIndex`/`fileIndexKey*` addition | exact |
| `internal/graphstore/batch.go` (`PutExcludedFile`/`DeleteExcludedFile`) | service (writer) | CRUD (batched write) | `pebbleWriter.PutFile` / `PutNode` | exact |
| `internal/graphstore/pebble_store.go` (`IterateExcludedFiles`, `pebbleExcludedFileIterator`) | service (reader) | streaming (prefix-scan iterator) | `pebbleReader.IterateFiles` / `pebbleFileIterator` | exact |
| `internal/graphstore/store.go` (`Reader.IterateExcludedFiles`, `Writer.PutExcludedFile`/`DeleteExcludedFile` interface methods) | model/interface | CRUD | `Reader.IterateFiles`/`GetFile`, `Writer.PutFile` interface decls | exact |
| `internal/graphstore/export.go` (open question: add `exportKindExcludedFile`) | service (batch/file-I/O) | batch (streaming export/import) | existing `exportKind*` tags + `importRecord` switch | role-match (explicit gap — see below) |
| `internal/indexer/discover.go` (4 decision-point exclusion capture + size pre-check) | service (discovery walker) | event-driven (WalkDir callback) | itself — `Discover`, `ShouldSkipDir`, `pendingFile` | exact |
| `internal/indexer/pipeline.go` (thread `[]*schema.ExcludedFile` through `Discover`→`Resolve`) | controller (pipeline orchestration) | request-response (call-site wiring) | itself — existing `Discover`/`Resolve` call at `pipeline.go:~100` | exact |
| `internal/indexer/resolve.go` (`writeGraph` stages `PutExcludedFile`, stamps `Meta.HasCoverage`) | service (commit/write path) | CRUD (one atomic batch commit) | itself — `writeGraph`'s `File`/`Meta.HasFileIndex` staging | exact |
| `internal/indexer/sync.go` (diff current exclusions vs stored; upsert+prune; stamp `HasCoverage` at both `NewMeta()` sites) | service (incremental sync) | CRUD (upsert/prune diff) | itself — the `deleted` file diff (`sync.go:141-153`) and the two `newMeta.HasFileIndex = true` stamp sites | exact |
| `internal/query/coverage.go` (new — `Engine.CoverageSummary()`/`Coverage(...)`) | service (query engine) | request-response (read-only aggregation + paged read) | `internal/query/status.go`'s `Status`/`IndexHealth` derivation | exact |
| `internal/query/coverage_test.go` (+ file-scoped source-scan test) | test | — | `internal/cli/editordiscovery_test.go#TestEditorDiscoverySourceNeverSpawnsAProcess` | exact (for the scan half); `internal/query/meta_test.go#TestIndexMetaOnAGraphWithNoMeta` (for the unknown-state half) |
| `internal/uiproto/uiv1/ui.proto` (`Coverage coverage = 17`, `GetCoverage` rpc + 4 messages) | model/wire schema | CRUD (additive wire schema) | Phase 9's `GetEditorLink` addition | exact |
| `internal/uiserver/handlers.go` (`healthToProto` extension + new `GetCoverage` handler) | controller | request-response | `healthToProto`/`GetHealth`; `editorlink.go`'s answers-not-errors handler shape | exact |
| `internal/uiserver/readonly_test.go` (`wantUIServiceMethods` 15→16, `mutatingVerbs` decoy check, `uiProtoFieldNumbers` extension) | test | — | itself — existing fixtures | exact |
| `web/src/lib/health-view.ts` (coverage row/group projections) | utility (pure view derivation) | transform | itself — `toCountRows`, `hasWorktreeMismatch` | exact |
| `web/src/routes/health/+page.svelte` (new Coverage section) | component | request-response (poll + render) | itself — existing count-table / worktree-mismatch sections | exact |
| `web/tests/health-page.test.ts` / `health-view.test.ts` | test | — | itself — existing test structure | exact |
| `internal/indexer/testdata/coverage/` fixture module | fixture (test data) | file-I/O | `internal/indexer/testdata/prunefixture/`, `gofixture/` | exact |
| `web/scripts/coverage-check.mjs` (optional) | test (live-browser gate) | event-driven (Playwright) | `web/scripts/breadcrumb-check.mjs` | role-match |

## Pattern Assignments

### `internal/schema/graph.proto` (model, CRUD)

**Analog:** itself — additive field history on `Meta`

**Pattern to copy** — additive-only field numbering, comment convention (`internal/schema/graph.proto`, `Meta` message; confirmed via RESEARCH.md's read of the whole file and `internal/schema/meta_commit_test.go`):
```proto
message Meta {
  // ... fields 1-6 ...
  bool has_file_index = 7; // additive: Phase 4 D-02, absent/false = pre-phase graph, "unknown", never inferred true
  string commit_sha = 8;   // additive: absent = "", never an error
  bool has_coverage = 9;   // NEW (D-06): absent/false = coverage UNKNOWN, never 0/0, never inferred from key presence
}

// NEW message + enum (D-05/D-08), sibling to File, not a File field:
message ExcludedFile {
  string path = 1;
  ExclusionReason reason = 2;
  string detail = 3;      // extension, byte size, or pruned dir name — free text
  int64 size_bytes = 4;
}

enum ExclusionReason {
  EXCLUSION_REASON_UNSPECIFIED = 0;
  EXCLUSION_REASON_DIR_VENDOR = 1;
  EXCLUSION_REASON_DIR_DOTPREFIX = 2;
  EXCLUSION_REASON_UNSUPPORTED_EXTENSION = 3;
  EXCLUSION_REASON_BUILD_TAG = 4;
  EXCLUSION_REASON_SIZE_LIMIT = 5;
}
```
`SchemaVersion` in `internal/schema/meta.go` stays `1` — this is an additive field/message, not a format break (mirrors the `has_file_index`/`commit_sha` precedent exactly).

**Field-stability guard note (Pitfall 5):** `internal/schema/meta_commit_test.go`'s `knownMetaFieldNumbers` fixture is a documented SUBSET fixture (its own comment: "a future additive field 9 must pass this test, not break it") — add the `has_coverage = 9` entry for drift-tracking but it is not strictly required for the test to pass. There is currently NO equivalent field-stability fixture for `File` or for a brand-new `ExcludedFile` message; adding one is a genuine open design choice, not a scripted extension.

---

### `internal/graphstore/keys.go` (model/storage, CRUD)

**Analog:** `prefixFileIndex byte = 'x'` and its key builders (this file, lines 14-193)

**Imports/const pattern** (`internal/graphstore/keys.go:14-37`):
```go
const (
	prefixMeta byte = 'm'
	prefixNode byte = 'n'
	prefixEdge byte = 'e'
	prefixFile byte = 'f'
	prefixAnnotation byte = 'a' // RESERVED for post-v1
	prefixFileIndex byte = 'x'
	// NEW: prefixExcludedFile byte = '?' — any byte outside {m,n,e,f,a,x} (Claude's Discretion; 'c' is mnemonic and unverified-but-free per RESEARCH A2)
)
```

**Key-builder pattern to copy verbatim, renamed** (`internal/graphstore/keys.go:118-125`, the `fileKey` shape — length-prefixed single-segment key):
```go
func fileKey(path string) []byte {
	buf := make([]byte, 0, 1+binary.MaxVarintLen64+len(path))
	buf = append(buf, prefixFile)
	buf = appendSegment(buf, path)
	return buf
}
```
Use `appendSegment` (keys.go:61-67) for the same tamper-safety reason documented there: length-prefixed segments, never raw `"prefix/" + path` string concatenation — a crafted path containing `/`, NUL, or 0xFF must not bleed into an adjacent key range (Security Domain V5 / T-01-02, this file's own doc comment).

Use `rangeUpperBound(prefix)` (keys.go:226-243) unchanged for the new namespace's prefix-scan/range-delete upper bound — do not reimplement.

---

### `internal/graphstore/batch.go` (service/writer, CRUD)

**Analog:** `pebbleWriter.PutFile`/`PutNode` (this file, lines 39-97+)

**Core pattern** (`internal/graphstore/batch.go:39-53`, `PutNode` shape — marshal via `deterministicMarshal`, `batch.Set`):
```go
func (w *pebbleWriter) PutNode(n *schema.Node) error {
	data, err := deterministicMarshal(n)
	if err != nil {
		return err
	}
	if err := w.batch.Set(nodeKey(n.GetId()), data, nil); err != nil {
		return err
	}
	// ... side-effect secondary-index write, if any ...
	return nil
}
```
`PutExcludedFile(x *schema.ExcludedFile) error` copies this exact shape: `deterministicMarshal(x)` (never raw `proto.Marshal` — this file's own comment explains why: deterministic map-field ordering, though `ExcludedFile` has no map field today, route through the shared helper for consistency and future-proofing), then `w.batch.Set(excludedFileKey(x.GetPath()), data, nil)`. Add `DeleteExcludedFile(path string) error` mirroring `DeleteFileSubgraph`'s single-key delete style (`batch.go:125`).

---

### `internal/graphstore/pebble_store.go` (service/reader, streaming)

**Analog:** `pebbleReader.IterateFiles` + `pebbleFileIterator` (verified in RESEARCH.md at `internal/graphstore/pebble_store.go:289-297, 401-436`)

**Core pattern to copy verbatim, renamed:**
```go
func (r *pebbleReader) IterateFiles() (FileIterator, error) {
	lower := []byte{prefixFile}
	upper := rangeUpperBound(lower)
	iter, err := r.snap.NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	return &pebbleFileIterator{iter: iter}, nil
}

type pebbleFileIterator struct {
	iter    *pebble.Iterator
	started bool
	cur     *schema.File
	err     error
}

func (it *pebbleFileIterator) Next() bool {
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
	var f schema.File
	if err := proto.Unmarshal(it.iter.Value(), &f); err != nil {
		it.err = err
		return false
	}
	it.cur = &f
	return true
}
```
`IterateExcludedFiles() (ExcludedFileIterator, error)` and `pebbleExcludedFileIterator` are the identical shape against the new prefix, unmarshaling `schema.ExcludedFile` instead of `schema.File`. This is the pattern explicitly named "Don't Hand-Roll" in RESEARCH.md — do not invent a new scan mechanism.

---

### `internal/graphstore/store.go` (model/interface, CRUD)

**Analog:** existing `Reader`/`Writer` interface method docs (this file, lines 1-80)

**Pattern:** add to `Reader`:
```go
// IterateExcludedFiles returns an ExcludedFileIterator over every
// exclusion record under the new namespace — a single contiguous range
// scan (mirrors IterateFiles).
IterateExcludedFiles() (ExcludedFileIterator, error)
```
and to `Writer`: `PutExcludedFile(x *schema.ExcludedFile) error`, `DeleteExcludedFile(path string) error` — doc-comment style copied from `PutFile`/`GetFile`'s existing entries in this same file.

---

### `internal/graphstore/export.go` (service, batch — EXPLICIT GAP, Pitfall 2)

**Analog:** existing `exportKind*` tag set + `importRecord` switch (`internal/graphstore/export.go:1-60, 190-222`, verified in RESEARCH.md)

**Decision required, not a silent omission (RESEARCH Open Question 1):** `Export`/`Import`'s fixed `exportKindX` set (`meta`, `node`, `edge`, `file`) has no slot for a new record kind, and — unlike `prefixFileIndex`, which free-rides on `PutNode`/`PutEdge`'s side effects during import — a new `ExcludedFile` namespace has NO other write path that reconstructs it. The planner must either (a) add `exportKindExcludedFile` to both the `Export` switch and `importRecord`'s switch, following the existing `file` case's shape exactly, or (b) explicitly document the round-trip gap (a documented gap, per this project's "positive assertion" discipline — silence is not acceptable per RESEARCH.md).

---

### `internal/indexer/discover.go` (service, event-driven)

**Analog:** itself — `Discover`'s `WalkDir` callback, `ShouldSkipDir` (verified `[internal/indexer/discover.go:53-55, 101-156]` per RESEARCH.md, quoted there verbatim)

**The 4 decision points (exact excerpt, RESEARCH.md Pattern 1):**
```go
walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
		if p != root && ShouldSkipDir(d.Name()) {
			return fs.SkipDir   // decision point 1: DIR_VENDOR / DIR_DOTPREFIX
		}
		return nil
	}

	ext := strings.ToLower(filepath.Ext(d.Name()))
	spec, ok := lookupLanguageByExt(ext)
	if !ok {
		return nil               // decision point 2: UNSUPPORTED_EXTENSION
	}

	abs, err := filepath.Abs(p)
	if err != nil {
		return err
	}

	if spec.ID == "go" {
		match, err := ctx.MatchFile(filepath.Dir(abs), filepath.Base(abs))
		if err != nil {
			return err
		}
		if !match {
			return nil            // decision point 3: BUILD_TAG
		}
	}

	relPath, err := filepath.Rel(root, p)
	// ...
	info, err := d.Info()
	// ...
	pending = append(pending, pendingFile{ /* ... sizeBytes: info.Size() ... */ })
	// decision point 4 (NEW, D-04): insert here, comparing info.Size()
	//     against parser.MaxSourceBytes BEFORE appending to pending
	return nil
})
```
```go
func ShouldSkipDir(name string) bool {
	return name == "vendor" || strings.HasPrefix(name, ".")
}
```
Distinguishing `DIR_VENDOR` from `DIR_DOTPREFIX` requires re-inspecting `d.Name()` at the call site (`ShouldSkipDir` itself only returns bool) — either give it a sibling that returns the reason, or inline the same two-branch check. Size threshold: `internal/parser/parser.go:15` — `const MaxSourceBytes = 4 * 1024 * 1024`.

`Discover`'s signature widens to a 4th return value `[]*schema.ExcludedFile` (or an equivalent slice type) — its only 2 callers are `internal/indexer/pipeline.go:~100` and `internal/indexer/sync.go:~85`, both inside `internal/indexer` itself (confirmed no external caller, RESEARCH.md).

---

### `internal/indexer/resolve.go` (service, CRUD — one atomic batch)

**Analog:** itself — `writeGraph`'s existing `Meta.HasFileIndex` stamping (`internal/indexer/resolve.go:257-277`, quoted verbatim in RESEARCH.md Pattern 2)

```go
func writeGraph(store graphstore.GraphStore, nodes, packageNodes []*schema.Node, edges []*schema.Edge, files []*schema.File, commitSHA string) error {
	w, err := store.NewWriter()
	if err != nil {
		return err
	}
	for _, n := range allNodes { /* w.PutNode(n) */ }
	for _, f := range sortedFiles { /* w.PutFile(f) */ }
	for _, e := range collapsedEdges { /* w.PutEdge(e, nodeFilePath[e.Source]) */ }
	// NEW: for _, x := range sortedExcluded { w.PutExcludedFile(x) }

	meta := schema.NewMeta()
	meta.NodeCount = int64(len(allNodes))
	meta.EdgeCount = int64(len(collapsedEdges))
	meta.HasFileIndex = true
	// NEW: meta.HasCoverage = true — same unconditional-every-call stamping as HasFileIndex
	meta.LastSyncUnixMs = time.Now().UnixMilli()
	meta.CommitSha = commitSHA
	if err := w.PutMeta(meta); err != nil { /* ... */ }
	return w.Commit()
}
```
CRITICAL: exclusion records go on the SAME `w` — never a second `store.NewWriter()`/`Commit()` pair (Anti-Pattern in RESEARCH.md — reopens the torn-write window D-04a exists to close).

**`File.errors` precedent — the source for `extraction_failed`, NOT to duplicate** (`internal/indexer/resolve.go:288-300`):
```go
f := &schema.File{
	Path:        r.RelPath,
	ContentHash: r.ContentHash,
	Language:    r.Language,
	NodeCount:   int64(len(r.Nodes)),
	EdgeCount:   int64(len(r.IntraEdges) + resolvedForFile),
	MtimeUnixNs: r.MtimeUnixNs,
	SizeBytes:   r.SizeBytes,
}
if r.Err != nil {
	f.Errors = []string{r.Err.Error()}
}
files = append(files, f)
```
The coverage summary's `extraction_failed` count is DERIVED from `File.Errors` non-empty — never a second, parallel bookkeeping mechanism.

---

### `internal/indexer/sync.go` (service, upsert/prune diff)

**Analog:** itself — the existing `deleted` file diff (`internal/indexer/sync.go:141-153`, quoted verbatim in RESEARCH.md Pattern 3):
```go
var deleted []string
fit, err := r0.IterateFiles()
// ...
for fit.Next() {
	p := fit.File().GetPath()
	if _, ok := discovered[p]; !ok {
		deleted = append(deleted, p)
	}
}
```
Coverage-namespace analogue: build `map[string]struct{}` from this Sync's own fresh `Discover` exclusion list, iterate the stored exclusion namespace via `IterateExcludedFiles`, `DeleteExcludedFile` for any stored path no longer excluded, `PutExcludedFile` (blind upsert, no hash comparison) for every currently-excluded path.

**Pitfall 3 — stamp `HasCoverage` at ALL THREE `schema.NewMeta()` call sites**, mirroring how `HasFileIndex` is stamped independently at each: `resolve.go`'s `writeGraph` (above), and `sync.go`'s two direct-construction sites — the mtime-refresh-only early return (`sync.go:~177`: `newMeta.HasFileIndex = true`) and the main incremental commit (`sync.go:~406`: same pattern). `schema.NewMeta()` returns a zero-valued struct every call (`internal/schema/meta.go:17-19`) — nothing carries a previous Meta's flags forward automatically.

---

### `internal/query/coverage.go` (NEW — service, request-response)

**Analog:** `internal/query/status.go`'s `Status`/`IndexHealth` derivation (this file, lines 43-108, 184, 360 confirmed present)

**Pattern:** an `Engine` method that opens a `Reader` (`store.Snapshot()`), calls `reader.GetMeta()` to check `meta.GetHasCoverage()` FIRST (unknown-state short-circuit, mirrors `internal/query/meta_test.go#TestIndexMetaOnAGraphWithNoMeta`'s `HasFileIndex` precedent), then — only if known — iterates `reader.IterateFiles()` (for `extraction_failed`, deriving from non-empty `Errors`) and `reader.IterateExcludedFiles()` (for `excluded`/`excluded_by_reason`), aggregating counts in-memory. NEVER `filepath.WalkDir`/`os.ReadDir` on the repo — `status.go`'s own two `WalkDir` calls (`newestSourceMtime`, `dbSizeBytes`, lines 108, 184) are a DIFFERENT concern (mtime/db-size) and must not be treated as license to walk for coverage.

**Unknown-state contract to copy** (mirrors `HasFileIndex`'s absent/false → unknown, never 0/0):
```go
meta, err := eng.IndexMeta()
if err != nil { return CoverageSummary{}, err }
if !meta.GetHasCoverage() {
	return CoverageSummary{Known: false}, nil // never {Known:false, Discovered:0, ...} read as "0/0"
}
```

---

### `internal/query/coverage_test.go` (test)

**Analog A (file-scoped source scan, D-14b):** `internal/cli/editordiscovery_test.go#TestEditorDiscoverySourceNeverSpawnsAProcess` (lines 299-341, quoted in full above under tool output) — copy this EXACT shape:
```go
func TestCoverageSourceNeverWalksDisk(t *testing.T) {
	src, err := os.ReadFile("coverage.go")
	if err != nil { t.Fatalf("read coverage.go: %v", err) }
	text := string(src)

	forbidden := []string{"filepath.WalkDir(", "os.ReadDir("}
	for _, f := range forbidden {
		if strings.Contains(text, f) {
			t.Fatalf("coverage.go contains %q — HLT-05 forbids reconstructing exclusion reasons by re-walking disk", f)
		}
	}
	// Positive control — proves this scan inspected real content, not an empty/renamed file.
	required := []string{"graphstore.Reader", "IterateExcludedFiles"}
	for _, r := range required {
		if !strings.Contains(text, r) {
			t.Fatalf("coverage.go does not contain %q — positive control failed", r)
		}
	}
}
```
Scope strictly to the new file, NOT package-wide — `internal/query/status.go` has 2 legitimate pre-existing `WalkDir` calls that would make a package-wide zero-guard false on day one (Anti-Pattern, RESEARCH.md).

**Analog B (old-graph unknown-state):** `internal/query/meta_test.go#TestIndexMetaOnAGraphWithNoMeta` (line 70) — open a store with `Meta.has_coverage` unset, assert `coverage.known == false` with no counts, never an error.

---

### `internal/uiproto/uiv1/ui.proto` (model/wire schema, CRUD)

**Analog:** Phase 9's `GetEditorLink` addition; `GetHealthResponse`'s existing field history (`internal/uiproto/uiv1/ui.proto:756-822`, quoted verbatim in RESEARCH.md — CRITICAL correction: field 16 is `commit_sha`, already spent)

```proto
message GetHealthResponse {
  // ... fields 1-15 ...
  string commit_sha = 16;
  Coverage coverage = 17;   // NEW — do NOT reuse 16
}

message Coverage {
  bool known = 1;
  int64 discovered = 2;
  int64 indexed = 3;
  int64 excluded = 4;
  int64 extraction_failed = 5;
  map<string, int64> excluded_by_reason = 6;
}

// New rpc — first paged rpc on UIService, no precedent to copy a token shape from (RESEARCH: confirmed absence of any page_token/page_size identifier repo-wide)
message GetCoverageRequest {
  int32 page_size = 1;
  string page_token = 2;
  codegraph.schema.v1.ExclusionReason reason = 3; // optional filter
}
message GetCoverageResponse {
  repeated CoverageRow rows = 1;
  string next_page_token = 2;
}
message CoverageRow {
  string path = 1;
  string kind = 2;    // "excluded" | "extraction_failed"
  codegraph.schema.v1.ExclusionReason reason = 3;
  string detail = 4;
}

service UIService {
  // ... existing 15 rpcs ...
  rpc GetCoverage(GetCoverageRequest) returns (GetCoverageResponse);
}
```
Run `task proto:gen` once — it regenerates BOTH proto surfaces (schema + UI, Go + TS) in one invocation (`Taskfile.yml:305-347`).

---

### `internal/uiserver/handlers.go` (controller, request-response)

**Analog A (additive response field):** `healthToProto` (`internal/uiserver/handlers.go:900-931`, quoted in full above):
```go
func healthToProto(result query.StatusResult, commitSHA string) *uiv1.GetHealthResponse {
	return &uiv1.GetHealthResponse{
		Initialized: result.Initialized,
		// ... existing fields ...
		Stale:     result.Stale,
		CommitSha: commitSHA,
		Coverage:  coverageToProto(result.Coverage), // NEW mapper, same convention
	}
}
```
Add a named `coverageToProto` mapper — never an inline literal at the call site (this file's established convention, per `worktreeMismatchToProto`'s doc comment on nil handling).

**Analog B (answers-not-errors handler shape):** `internal/uiserver/editorlink.go` — every configuration/unknown state is a SUCCESSFUL response with a populated reason, never an error. Apply identically: `Coverage.known == false` is a normal, successful `GetHealthResponse`, not an error. `GetCoverage`'s handler follows `GetHealth`'s ordinary `withEngine` shape (`internal/uiserver/handlers.go:968` region) — not `GetStatus`'s degrade-and-answer exception.

---

### `internal/uiserver/readonly_test.go` (test)

**Analog:** itself — `wantUIServiceMethods` (line 71), `mutatingVerbs` (line 134), `uiProtoFieldNumbers` (line 318)

```go
var mutatingVerbs = []string{
	"Create", "Update", "Delete", "Remove", "Set", "Put", "Post",
	"Write", "Add", "Insert", "Mutate", "Patch", "Modify", "Sync",
	"Reindex", "Index", "Clear", "Reset", "Save",
}
```
`GetCoverage` clears all 19 substrings (case-sensitive `strings.Contains`, line ~148). Add `"GetCoverage": {}` to `wantUIServiceMethods` (15→16, set-equality both directions per `TestUIServiceMethodSetIsExactlyTheReadSet`, line 98). Add a decoy-rejection subtest: `GetIndexCoverage` must be rejected because it contains `"Index"` — the exact trap RESEARCH.md flags. Extend `uiProtoFieldNumbers` (line 318) additively with the new `Coverage`/`GetCoverageRequest`/`GetCoverageResponse`/`CoverageRow` message field entries, and bump `uiProtoFieldFixtureLenAtPlan0901`-equivalent length constant — all in the SAME commit as the proto change (Phase 9 recipe, `09-01-SUMMARY.md`).

---

### `web/src/lib/health-view.ts` (utility, transform)

**Analog:** itself — `toCountRows`, `hasWorktreeMismatch` (this file, lines 1-80, quoted above)

```ts
export interface CountRow {
	key: string;
	count: number;
}

export function toCountRows(map: { [key: string]: bigint } | undefined): CountRow[] {
	if (!map) return [];
	return Object.entries(map)
		.map(([key, count]) => ({ key, count: Number(count) }))
		.sort((a, b) => b.count - a.count || a.key.localeCompare(b.key));
}
```
A new `toCoverageView(response)`-shaped pure function follows this exact convention: no verdict computed here (this module's own D-04 discipline, stated in its header comment — "This module computes NO verdict of its own"), `Coverage.known === false` maps to an explicit `{ known: false }` view object the component renders as "Coverage unknown — re-index to record it," never an empty table. `excluded_by_reason` (a bigint-valued map, same shape as `FilesByLanguage`) reuses `toCountRows` directly for the reason-grouped counts.

---

### `web/src/routes/health/+page.svelte` (component, request-response)

**Analog:** itself — existing count-table (`CountTable.svelte`) / worktree-mismatch section composition

Add a new "Coverage" section following the same `CountTable`/`DataTable` shell reuse (D-06's "one generic DataTable shell" discipline, per `health-view.ts`'s header comment) — text-only remedy copy for the unknown state and for `File.errors` rows ("extraction failed: `<error>`"), never a "re-index this file" action or button (SRV-03, out of scope by construction).

---

### `internal/indexer/testdata/coverage/` (fixture, file-I/O)

**Analog:** `internal/indexer/testdata/prunefixture/` (a `go.mod` + `pkg/` fixture already used to exercise `ShouldSkipDir`-adjacent pruning) and `internal/indexer/testdata/gofixture/` (a plain `go.mod` + multi-package fixture)

**Structure to copy:** a committed `go.mod` at the fixture root (mirrors `gofixture/go.mod`, `prunefixture/go.mod`), then per D-13:
```
internal/indexer/testdata/coverage/
├── go.mod
├── vendor/x.go              # DIR_VENDOR
├── .hidden/y.go             # DIR_DOTPREFIX
├── notes.md                 # UNSUPPORTED_EXTENSION
├── tagged.go                # //go:build ignore -> BUILD_TAG
└── <extraction-failure file, per Pitfall 4 — NOT oversize, since D-04 now
     pre-empts oversize at discovery; RESEARCH proposes (ASSUMED, verify
     first) a dangling symlink discovered normally but unreadable at
     os.ReadFile time>
```
The oversize `.go` file for `SIZE_LIMIT` is GENERATED AT TEST TIME (never committed, per D-13) — follow `internal/indexer/extract_test.go#TestExtractPool_OversizedFileContained`'s (lines 212-251) technique for constructing an oversized file programmatically in the test, but note per Pitfall 4 that this precedent test bypasses `Discover` entirely (feeds `Extract` directly) — the new coverage fixture test must go through the REAL `Discover(root)` path so the file is classified `SIZE_LIMIT` at discovery, not fed straight to `Extract`.

---

### `web/scripts/coverage-check.mjs` (optional, live-browser gate)

**Analog:** `web/scripts/breadcrumb-check.mjs` (lines 1-60, quoted above)

Copy conventions: real Chromium via `@playwright/test`'s `chromium` launcher, `startCodegraphUi`-equivalent child-process lifecycle owned by the script itself, `pollUntil` (poll, never sleep), a diagnostic JSON record on every exit path. Per RESEARCH.md Open Question 3, this is genuinely optional — HLT-04's requirement text does not mandate a live-browser proof the way BRW-10/BRW-11 did; default to a vitest-level test (`health-view.test.ts` convention) unless the plan's own risk assessment calls for the heavier live gate.

## Shared Patterns

### Additive-only schema/wire evolution
**Source:** `internal/schema/graph.proto`'s `Meta.has_file_index=7`/`commit_sha=8` history; `internal/uiproto/uiv1/ui.proto`'s `GetHealthResponse` field history
**Apply to:** `graph.proto`, `ui.proto` — never renumber or repurpose an existing field; absent/unset always means "unknown," never zero, never an error.

### Absent-means-unknown, never zero/error
**Source:** `Meta.HasFileIndex` precedent (`internal/indexer/resolve.go:270`), `internal/query/meta_test.go#TestIndexMetaOnAGraphWithNoMeta`
**Apply to:** `internal/query/coverage.go`'s `CoverageSummary`, `internal/uiserver/handlers.go`'s `healthToProto`/`coverageToProto`, `web/src/lib/health-view.ts`'s coverage view.

### One atomic writer batch per commit point
**Source:** `internal/indexer/resolve.go`'s `writeGraph` (single `store.NewWriter()`/`w.Commit()`)
**Apply to:** `internal/indexer/resolve.go`, `internal/indexer/sync.go` — exclusion records stage on the SAME `Writer` as `File`/`Node`/`Edge`/`Meta`, never a second writer.

### Length-prefixed segment key encoding (never raw concatenation)
**Source:** `internal/graphstore/keys.go`'s `appendSegment`
**Apply to:** the new `excludedFileKey` builder — tamper-safety against crafted paths (Security Domain V5/T-01-02).

### Answers-not-errors for closed-enum/config states
**Source:** `internal/uiserver/editorlink.go` (`EditorLinkAvailability`)
**Apply to:** `GetCoverage`/`GetHealth`'s `Coverage.known == false`, and every `ExclusionReason` value — always a successful response, never an error.

### File-scoped structural source-scan guard (never package-wide)
**Source:** `internal/cli/editordiscovery_test.go#TestEditorDiscoverySourceNeverSpawnsAProcess`
**Apply to:** `internal/query/coverage_test.go`'s D-14b structural guard — scope to `coverage.go` specifically; `internal/query/status.go`'s 2 legitimate `WalkDir` calls make a package-wide guard false on day one.

### Additive rpc + same-commit fixture update (the Phase 9 recipe)
**Source:** `09-01-SUMMARY.md`, `internal/uiserver/readonly_test.go`
**Apply to:** `ui.proto`'s `GetCoverage` addition, `wantUIServiceMethods`, `uiProtoFieldNumbers` — proto edit + `task proto:gen` + both fixture updates in ONE commit.

## No Analog Found

None. Every file this phase touches has at least a role-match analog in the codebase; RESEARCH.md's own exhaustive verification (every claim `[VERIFIED: path:line]`) left no unmatched file. The one genuinely novel piece — `GetCoverage`'s page-token shape — has no existing paged-rpc precedent to copy (confirmed absence, not merely unfound); it is Claude's Discretion per `10-CONTEXT.md`, and RESEARCH.md's "Don't Hand-Roll" table gives the constraint (opaque to the client, stable ordering off `IterateExcludedFiles`'s already-sorted iteration) rather than a literal analog.

## Metadata

**Analog search scope:** `internal/schema/`, `internal/graphstore/`, `internal/indexer/`, `internal/query/`, `internal/uiserver/`, `internal/uiproto/uiv1/`, `internal/cli/`, `web/src/lib/`, `web/src/routes/health/`, `web/scripts/`, `internal/indexer/testdata/`
**Files scanned:** ~30 (via RESEARCH.md's prior verification plus this session's targeted confirmatory reads of `status.go`, `handlers.go`, `editorlink.go`, `readonly_test.go`, `keys.go`, `batch.go`, `store.go`, `health-view.ts`, `editordiscovery_test.go`, `breadcrumb-check.mjs`, `testdata/` layout)
**Pattern extraction date:** 2026-09-12
