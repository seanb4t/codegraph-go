# Phase 2: Go Indexing Pipeline - Pattern Map

**Mapped:** 2026-07-10
**Files analyzed:** 14 (new packages/files implied by CONTEXT.md/RESEARCH.md)
**Analogs found:** 14 / 14 (all via role/data-flow match against Phase 1 substrate — no external analogs needed)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `internal/indexer/discover.go` | utility | file-I/O | `internal/graphstore/pebble_store.go` (Open/lifecycle shape) | partial |
| `internal/indexer/extract.go` | service | batch (parallel) | `internal/parser/cgo/parser_cgo.go` (backend-owns-lifecycle pattern) | role-match |
| `internal/indexer/goextract/goextract.go` | service | transform (AST→graph records) | `internal/parser/cgo/parser_cgo.go` | role-match |
| `internal/indexer/resolve.go` | service | batch (single-writer CRUD) | `internal/graphstore/store_test.go`'s writer-usage pattern + `store.go` `Writer` iface | exact (consumes GraphStore.Writer directly) |
| `internal/indexer/nodeid.go` | utility | transform | `internal/graphstore/keys.go` (`appendSegment`) | exact (explicitly reused per RESEARCH Pattern 3) |
| `internal/indexer/pipeline.go` | service | event-driven (orchestration) | `internal/graphstore/pebble_store.go` (`Open` + lifecycle/Close discipline) | role-match |
| `internal/cli/root.go` | controller | request-response | none (new Cobra surface) | no analog |
| `internal/cli/init.go` | controller | CRUD (create `.codegraph/`) | `internal/graphstore/pebble_store.go` `Open` (store-open error path) | partial |
| `internal/cli/index.go` | controller | batch | `internal/indexer/pipeline.go` (consumer) | n/a (internal to this phase) |
| `internal/cli/uninit.go` | controller | file-I/O | none (new; simple `os.RemoveAll` + confirm) | no analog |
| `cmd/codegraph/main.go` | config/bootstrap | request-response | none (new; thin `cli.Execute()` wrapper) | no analog |
| `internal/schema/graph.proto` (edit) | model | transform | itself, additive edit — extend existing `Node`/`Edge` messages | exact (edit-in-place) |
| `internal/indexer/discover_test.go`, `extract_test.go`, `resolve_test.go`, `goextract_test.go`, `nodeid_test.go`, `determinism_test.go` | test | request-response | `internal/graphstore/store_test.go`, `internal/parser/parser_test.go` | exact |
| `internal/cli/*_test.go` | test | request-response | `internal/graphstore/store_test.go` (table-driven + `t.TempDir()` idiom) | role-match |

## Pattern Assignments

### `internal/indexer/nodeid.go` (utility, transform)

**Analog:** `internal/graphstore/keys.go`

**Core pattern to copy — length-prefixed segment encoding** (`internal/graphstore/keys.go` lines 26-44):
```go
// appendSegment appends a length-prefixed encoding of seg to buf.
func appendSegment(buf []byte, seg string) []byte {
	var lenBuf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(lenBuf[:], uint64(len(seg)))
	buf = append(buf, lenBuf[:n]...)
	buf = append(buf, seg...)
	return buf
}
```
RESEARCH.md's Pattern 3 already specifies reusing this exact discipline for the node-id hash preimage (not just storage keys) — do not hand-roll a `+`-joined string:
```go
func nodeID(kind, qualifiedName, filePath string) string {
	var buf []byte
	buf = appendSegment(buf, kind)
	buf = appendSegment(buf, qualifiedName)
	buf = appendSegment(buf, filePath)
	sum := sha256.Sum256(buf)
	return kind + ":" + hex.EncodeToString(sum[:])[:32]
}
```
**Doc-comment density:** match `keys.go`'s style — every exported func has a multi-sentence doc comment stating *why* (not just what), citing the relevant decision ID (D-02a) and the threat/rationale it mitigates.

**Error handling:** none needed (pure function, no error return) — matches `keys.go`'s encoder functions, which also return plain `[]byte`/`string` with no error.

---

### `internal/indexer/resolve.go` (service, batch/CRUD via single writer)

**Analog:** `internal/graphstore/store.go` (interface) + usage shape from `internal/graphstore/store_test.go` lines 62-80

**Writer interface to consume** (`store.go` lines 82-112) — resolve.go's Pass 2 binds directly to this, never touching Pebble:
```go
type Writer interface {
	PutNode(n *schema.Node) error
	PutEdge(e *schema.Edge) error
	PutFile(f *schema.File) error
	PutMeta(m *schema.Meta) error
	DeleteFileSubgraph(path string) error
	Commit() error
	Close() error
}
```

**Batched single-writer usage pattern** (`store_test.go` lines 62-80):
```go
w, err := store.NewWriter()
if err != nil {
	t.Errorf("NewWriter: %v", err)
	return
}
for i := 0; i < numNodes; i++ {
	n := &schema.Node{Id: fmt.Sprintf("node-%d", i), Kind: "function", Name: fmt.Sprintf("Fn%d", i)}
	if err := w.PutNode(n); err != nil {
		t.Errorf("PutNode: %v", err)
		return
	}
}
// ... single Commit() at the end — never per-symbol commits (D-04a)
```
**Error handling:** on any staging error, `Close()` the writer (releases the batch without applying it) rather than leaving it dangling — per `store.go`'s Writer.Close doc: "Callers that abandon a Writer without calling Commit ... MUST call Close."

**Determinism requirement (unique to this file, no analog):** aggregate every candidate call site into a slice, sort by `(filePath, line, col)`, take first — see RESEARCH.md Pitfall 1. No existing codebase analog for this; it is new engineering per RESEARCH's explicit guidance.

---

### `internal/indexer/extract.go` (service, parallel/batch)

**Analog:** `internal/parser/cgo/parser_cgo.go` — the per-worker parser-ownership + Close-once discipline

**Backend lifecycle pattern to replicate per worker goroutine** (`parser_cgo.go` lines 27-52, 77-84):
```go
type CGoParser struct {
	inner     *tree_sitter.Parser
	closeOnce sync.Once
}

func NewGoParser() (*CGoParser, error) {
	return newCGoParser(tree_sitter_go.Language())
}

func (p *CGoParser) Close() error {
	p.closeOnce.Do(func() {
		p.inner.Close()
	})
	return nil
}
```
Each `errgroup` worker in `extract.go` must call `cgo.NewGoParser()` once per goroutine (never once per file — parser_cgo.go's CGo allocation cost), and `defer p.Close()` immediately, matching the `closeOnce` guard's intent (safe against double-close on early-return paths).

**Size-ceiling contract to respect** (`internal/parser/parser.go` lines 8-20, and enforced inside `parser_cgo.go` lines 58-61):
```go
if len(source) > parser.MaxSourceBytes {
	return nil, parser.ErrSourceTooLarge
}
```
`extract.go` MUST treat `parser.ErrSourceTooLarge` as a per-file skip-with-warning (RESEARCH Pitfall 4), never a fatal pipeline abort — no existing sentinel-handling analog for "skip and continue" exists yet in this codebase; this is new per-file error containment logic that should be introduced here.

**Tree lifecycle:** `defer tree.Close()` immediately after a successful `Parse()`, per the `Tree.Close()` contract (`parser.go` lines 65-79): "Callers MUST call Close() exactly once when a Tree is discarded... this guard exists as a backstop, not license to skip it."

---

### `internal/indexer/pipeline.go` (service, orchestration)

**Analog:** `internal/graphstore/pebble_store.go` — the `Open`/`Close` lifecycle-guard shape

**Open/lifecycle pattern** (`pebble_store.go` lines 43-50, 72-82):
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
`pipeline.go`'s top-level `Run(dir string) error` (or similar) should mirror this — open the `GraphStore` once, defer `Close()`, and guard against double-invocation the same way (`atomic.Bool` swap-once idiom) if the CLI's `index --force` path could otherwise re-enter.

**Sentinel error style** (`pebble_store.go` lines 14-27):
```go
var ErrNotFound = errors.New("graphstore: not found")
var ErrClosed = errors.New("graphstore: store is closed")
```
Follow the `<package>: <lowercase message>` sentinel convention for any new `indexer.Err*` (e.g. `indexer.ErrAlreadyInitialized`, `indexer.ErrSourceTooLarge` pass-through).

---

### `internal/schema/graph.proto` (edit — additive extension, D-03)

**Analog:** itself (existing `Node`/`Edge` messages, lines 23-62)

**Additive-field pattern to replicate exactly:**
```protobuf
message Node {
  string id = 1;
  ...
  int32 end_col = 10;
  // ADD HERE: signature=11, docstring=12, visibility=13, is_exported=14 (bool), return_type=15
  reserved 50 to 59; // future: embedding vector, community/cluster assignment
}

message Edge {
  string source = 1;
  ...
  int32 col = 5;
  // ADD HERE: provenance=6 (string), metadata=7 (optional)
  reserved 50 to 59;
}
```
**Rule to enforce:** new field numbers MUST be assigned sequentially after the last existing field and MUST stay below 50 — never touch the `reserved 50 to 59` block. `SchemaVersion` (`internal/schema/meta.go` line 12) stays `1` — this is a purely additive change per the doc comment on `NewMeta()`. After editing `.proto`, regenerate `graph.pb.go` (do not hand-edit the generated file).

**Doc-comment convention:** every new field addition should get a one-line comment in the `.proto` explaining what wrote it and why it's additive-safe, matching the existing style (see `File.content_hash`'s comment at lines 69-73 for the density/tone to match).

---

### `internal/cli/*.go` (controller, request-response) — Cobra command tree

**Analog:** none in this codebase (first CLI surface) — follow RESEARCH.md's Code Examples verbatim (Cobra idioms are external-library convention, not project-specific pattern). Match the project's general doc-comment density (every exported `New*Cmd()` constructor gets a doc comment stating the command's contract, e.g. lifted from D-01a's semantics: "init on an existing .codegraph/ errors with guidance rather than silently clobbering").

**Error-sentinel convention to carry over from graphstore** (`pebble_store.go` lines 14-27 style): define `cli.ErrAlreadyInitialized`, `cli.ErrNotInitialized` as package-level sentinels, `errors.New("cli: ...")`, checked via `errors.Is` in tests — matches `ErrNotFound`/`ErrClosed`'s style exactly.

---

### Test files (all new `_test.go` under `internal/indexer/`, `internal/cli/`)

**Analog:** `internal/graphstore/store_test.go`, `internal/parser/parser_test.go`

**Structural conventions to replicate:**
- Package-internal test (`package graphstore`, not `graphstore_test`) — matches both analogs (`package graphstore` in store_test.go, `package parser` in parser_test.go).
- `t.TempDir()` for any on-disk fixture (store_test.go line 25) — reuse for `.codegraph/` test dirs in `internal/cli` and `internal/indexer` tests.
- Stub/fake implementations over mocking frameworks — `parser_test.go`'s `stubParser` (lines 13-34) is the precedent: a minimal hand-written struct implementing the interface under test, not a generated mock. Apply the same for a `stubWriter`/`stubGraphStore` in indexer tests if pipeline.go needs isolation from real Pebble.
- Doc comment on every `TestXxx` explaining *what property* is being proven and *why*, citing the requirement ID (e.g. `store_test.go` lines 11-23 cite INDX-05 and D-04 inline) — indexer tests should cite RES-01/LANG-01/D-04a/D-05 the same way.
- Table-driven tests for the node-kind mapping (`goextract_test.go`) — no direct analog exists yet in this repo for table-driven AST-mapping tests, but the project's general table-driven convention (seen in `internal/graphstore/keyenc_test.go`, not read in full here but consistent with Go stdlib idiom the rest of the repo follows) should be used: one test case per tree-sitter node type per RESEARCH's Architecture Patterns.
- RED→GREEN atomic commits (CONTEXT.md "Established Patterns"): write the failing test commit first, then the implementing commit — applies to every file above, not just tests.

---

## Shared Patterns

### Error Sentinels
**Source:** `internal/graphstore/pebble_store.go` lines 14-27, `internal/parser/parser.go` line 20
**Pattern:** `var ErrX = errors.New("<package>: <lowercase message>")`, checked via `errors.Is`.
**Apply to:** every new package (`internal/indexer`, `internal/cli`) — e.g. `indexer.ErrSourceTooLarge` (pass-through wrap of `parser.ErrSourceTooLarge`), `cli.ErrAlreadyInitialized`.

### Close-once / Idempotent Teardown
**Source:** `internal/parser/cgo/parser_cgo.go` lines 29-30, 79-84; `internal/parser/parser.go` lines 40-44, 73-79; `internal/graphstore/pebble_store.go` lines 40-41, 77-82
**Pattern:** `sync.Once` (or `atomic.Bool.Swap`) guarding a teardown call so repeat/concurrent `Close()` is a safe no-op.
**Apply to:** `pipeline.go`'s top-level orchestrator, any per-worker `Parser` wrapper in `extract.go`, and `cli`'s `.codegraph/` directory handle if one is introduced.

### Additive-Only Schema Discipline
**Source:** `internal/schema/graph.proto` (reserved 50-59 blocks), `internal/schema/meta.go` lines 3-12
**Pattern:** never renumber/reuse a field; new fields go after the last existing number and before any `reserved` range; `SchemaVersion` bump is reserved for genuinely breaking changes only.
**Apply to:** the `graph.proto` edit for D-03.

### Length-Prefixed Segment Encoding (anti-key-injection)
**Source:** `internal/graphstore/keys.go` lines 26-44
**Pattern:** `appendSegment(buf, seg)` — varint length prefix, not a literal delimiter — for any byte-string composed from untrusted/arbitrary input (file paths, qualified names).
**Apply to:** `internal/indexer/nodeid.go`'s hash preimage construction (explicitly called out in RESEARCH.md Pattern 3 and the Security Domain section's Tampering mitigation row).

### Architectural Boundary Enforcement
**Source:** `internal/graphstore/archtest` (referenced in `store.go` doc comment lines 9-12)
**Pattern:** an archtest asserts no package outside `internal/graphstore` imports the Pebble engine directly.
**Apply to:** Phase 2 must NOT add a new archtest exception — `internal/indexer` and `internal/cli` may only import `internal/graphstore`'s exported interface types (`GraphStore`, `Writer`, `Reader`), never `github.com/cockroachdb/pebble/v2` directly. Confirm the existing archtest doesn't need extending; if `internal/indexer` needs to enforce "only `resolve.go` (or the pipeline) calls `GraphStore.NewWriter()`," that would be a new, phase-2-specific archtest rule to consider (no existing analog for this narrower rule — flagged as a planning consideration, not a locked requirement).

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/cli/root.go`, `init.go`, `uninit.go` | controller | request-response | First Cobra CLI surface in this codebase — no prior CLI command exists to copy from. Use RESEARCH.md's Code Examples (Cobra idioms) as the reference instead. |
| `cmd/codegraph/main.go` | bootstrap | request-response | No `cmd/` directory exists yet in this repo. Convention: thin `main()` that calls `cli.Execute()` and returns its exit code — standard Cobra-app shape, not project-specific. |
| Determinism sort/tie-break logic (`resolve.go`'s dedup step) | service | transform | Genuinely new engineering per RESEARCH.md Pitfall 1 — no existing codebase precedent for "sort candidates by (filePath, line, col), take first" exists in Phase 1. |
| `internal/indexer/discover.go` (`go/build.Context.MatchFile` walk) | utility | file-I/O | No existing file-walk code in this repo to copy from; follow RESEARCH.md's Pattern 1 verbatim (stdlib `go/build`, not a codebase pattern). |

## Metadata

**Analog search scope:** `internal/graphstore/`, `internal/parser/`, `internal/parser/cgo/`, `internal/schema/` (all of Phase 1's substrate — the only existing Go source in the repo besides `.planning`/`testdata`)
**Files scanned:** `store.go`, `keys.go`, `pebble_store.go`, `store_test.go`, `parser.go`, `parser_test.go`, `parser_cgo.go`, `graph.proto`, `meta.go` (9 files read directly)
**Pattern extraction date:** 2026-07-10
