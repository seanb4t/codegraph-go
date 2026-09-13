# Phase 10: Index Health — The Coverage Denominator - Research

**Researched:** 2026-09-12
**Domain:** In-repo Go storage/wire-layer engineering (no new external libraries; no web research performed — all search providers are disabled in `.planning/config.json` and this phase has no unfamiliar-framework surface). Every finding below is `[VERIFIED: path:line]` from a same-session `Read`, or `[ASSUMED]` where flagged.
**Confidence:** HIGH — every load-bearing claim was confirmed by reading the actual source this session; the two `[ASSUMED]` items are called out explicitly in the Assumptions Log.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

D-01 through D-17, verbatim from `.planning/phases/10-index-health-the-coverage-denominator/10-CONTEXT.md`:

- **D-01:** "Discovered" is pinned once: **every regular file the walker actually visits.** Pruned subtrees (`vendor/`, dot-prefixed directories via `ShouldSkipDir`) are **not** discovered — their contents are never enumerated. The same definition feeds both the denominator and the reason list, so the two can never disagree in the user's face (ROADMAP note). — **Reversibility:** costly — changing the definition later changes every reported number.
- **D-02:** A pruned directory is represented by **one directory-level exclusion record** (`vendor/` → `DIR_VENDOR`; `.github/`, `.codegraph/` … → `DIR_DOTPREFIX`), never by phantom per-file rows. The UI renders these as "directory excluded" rows.
- **D-03:** An unsupported-extension file (`.md`, `.json`, `.yaml`, `.sum` …) gets **one record per file** (path + reason code + the extension in `detail`). The UI groups by reason (and extension) with counts and expands to rows. Aggregate-only counts were rejected because the per-file list would then have to be reconstructed at query time — exactly what HLT-05 forbids.
- **D-04:** Size limit becomes a **stat-based pre-extraction check at discovery**: `SizeBytes > parser.MaxSourceBytes` → reason `SIZE_LIMIT`, bytes never read. The parser's own `ErrSourceTooLarge` stays as the backstop; any file that still trips it remains an extraction failure persisted in `File.errors`, exactly as today (`resolve.go:298`).
- **D-05:** Exclusion records live in a **new graphstore key namespace** — one free single-byte prefix (the planner picks the letter; `m n e f a x` are taken) keyed `<prefix>/<path>` — holding a new `graph.proto` message `ExcludedFile { string path; ExclusionReason reason; string detail; int64 size_bytes; }`. `SchemaVersion` stays `1` (additive only), mirroring how `prefixFileIndex` was added in v0.12.0 Phase 4. `File` records are **not** overloaded with a not-indexed flag. — **Reversibility:** one-way once a release ships the namespace.
- **D-06:** An old graph reports coverage via an additive **`Meta.has_coverage = 9`** (next free field after `commit_sha = 8`), following the `HasFileIndex` precedent: absent/false → coverage **unknown**, never `0/0`, never an error. Presence of any exclusion key is **not** used to infer "known."
- **D-07:** Records are written **in the same commit batch as the `File` records** at the pipeline's commit point, honouring Phase 7's archtest D-01: `query` reads them back through `graphstore` only and never imports the discovery helper. A full index range-deletes the namespace then rewrites; `Sync` upserts per path and prunes paths no longer present. Reasons are **not** written eagerly during the walk.
- **D-08:** The reason vocabulary is a **closed proto enum** `ExclusionReason { UNSPECIFIED = 0; DIR_VENDOR; DIR_DOTPREFIX; UNSUPPORTED_EXTENSION; BUILD_TAG; SIZE_LIMIT; }` plus free-text `detail`. Closed like `EditorLinkAvailability`.
- **D-09:** `GetHealthResponse` is extended **additively** with `Coverage coverage = 16` — `bool known`, `int64 discovered`, `int64 indexed`, `int64 excluded`, `int64 extraction_failed`, `map<string,int64> excluded_by_reason`. **Research correction: field 16 is already spent (`commit_sha`) — use field 17.** — **Reversibility:** one-way once shipped on the wire.
- **D-10:** The per-file reason list is a **new rpc `GetCoverage(GetCoverageRequest{ int32 page_size; string page_token; ExclusionReason reason; }) → GetCoverageResponse{ repeated CoverageRow rows; string next_page_token; }`**, where a row carries path, kind (excluded vs extraction-failed), reason, detail. The name clears all 19 `mutatingVerbs` substrings. It is the **16th** rpc: `wantUIServiceMethods` 15→16 by set-equality in both directions with the count asserted from both sides, and `uiProtoFieldFixture` extended additively (the 09-01 pattern, same commit as the proto change). The full list is **not** inlined in `GetHealthResponse`. — **Reversibility:** one-way once shipped.
- **D-11:** The health page gains a **"Coverage" section**: `N discovered · M indexed · K excluded · J extraction failures`; grouped by reason with counts, each group expanding to file rows; `File.errors` rows rendered as "extraction failed: <error>" and visibly distinct from exclusions; an old graph renders "Coverage unknown — re-index to record it." Remedy is **text only** — no "re-index this file" action (SRV-03). No separate `/coverage` route.
- **D-12:** `codegraph status` and the MCP surface are **out of scope** this phase — recorded under Deferred Ideas.
- **D-13:** Criterion 1's fixture is a **committed fixture module under `internal/indexer/testdata/coverage/`** containing at least one file of each kind: `vendor/x.go` (DIR_VENDOR), `.hidden/y.go` (DIR_DOTPREFIX), `notes.md` (UNSUPPORTED_EXTENSION), `tagged.go` with `//go:build ignore` (BUILD_TAG), an oversize `.go` **generated at test time** above `parser.MaxSourceBytes` (SIZE_LIMIT — never committed), and one file that **fails extraction** so it lands in `File.errors`. The test indexes it and asserts the exact counts **and** the per-file reasons.
- **D-14:** Criterion 2 gets **two guards**: (a) behavioural — index the fixture, mutate the disk **without re-indexing** (strip the build tag, delete `notes.md`) and assert the read-back still reports the original reasons; (b) structural — `internal/query/archtest` still forbids the indexer root, plus a positive-controlled source scan that `internal/query` contains zero `filepath.WalkDir` / `os.ReadDir` calls against the repo root. **Research correction: (b)'s literal package-wide wording is already false today (`status.go` has two pre-existing, legitimate `filepath.WalkDir` calls) — scope the scan to the new coverage-reading file specifically, per the `editordiscovery_test.go` file-scoped-scan precedent.**
- **D-15:** Criterion 3 gets a dedicated **old-graph test**: open a store written without coverage records (`Meta.has_coverage` unset) and assert `GetHealth` reports `coverage.known == false` with no counts — never `0/0`, never an error — following `internal/query/meta_test.go`'s `HasFileIndex` precedent. Mutation family: flip the reader to treat unset as zero and watch it RED.
- **D-16:** Criterion 4: `readonly_test.go` asserts `wantUIServiceMethods` set-equality in **both** directions with the count from both sides, and a table test proves `GetCoverage` clears every `mutatingVerbs` substring while a decoy `GetIndexCoverage` is **rejected**. Mutation family: rename to the decoy and watch RED.
- **D-17:** `10-MUTATION-LOG.md` follows the 07/08/09 house format; `10-SECURITY.md` is required (security enforcement on, ASVS L1) and covers the write path's new trust surface (path strings from the walker persisted and rendered) with every threat row test-or-verdict.

### Claude's Discretion

- The exact key-prefix byte for the new namespace, and whether directory-level records share the file namespace or get a sub-prefix.
- `GetCoverage` default/maximum page size and the page-token encoding (follow whatever the existing paged rpcs use, if any — research finding: none exist).
- Whether `excluded_by_reason` keys are the enum names or numbers on the wire.
- Health-page grouping order and expansion UX details.
- Whether the discovery helper is a new file in `internal/indexer` or lives inside `discover.go` — either satisfies the archtest.

### Deferred Ideas (OUT OF SCOPE)

- `codegraph status --json` additive `coverage` field and any MCP exposure of coverage (D-12) — a later phase.
- Whether `node_modules` (not dot-prefixed, not `vendor`) should be pruned by `ShouldSkipDir` — pre-existing behaviour surfaced by this phase's denominator; a separate decision with watcher implications.
- "Re-index this file" remediation action on the coverage view — out of scope by construction (SRV-03).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|--------------------|
| HLT-04 | Index health shows the coverage denominator — files discovered but not indexed — with a per-file reason, distinguishing extraction failures from pre-extraction exclusions | Architecture Pattern 1 (the 4 discovery decision points with exact `file:line`), Pitfall 1 (the discovered-count unit distinction), Code Example (`File.errors` precedent), D-13's fixture requirements mapped to `internal/indexer/testdata/coverage/`, Validation Architecture's HLT-04 test-map rows |
| HLT-05 | Pre-extraction exclusion reasons recorded at the discovery decision point and persisted additively within SchemaVersion 1, never inferred by a query-time re-walk | Architecture Patterns 1-3 (decision points, one-batch commit, Sync's upsert/prune simplification), Don't Hand-Roll (iterator/key-namespace reuse), Anti-Patterns (the archtest boundary and the corrected D-14b scan scope), Pitfalls 2/3/4 (export/import gap, Meta stamping at all 3 sites, the extraction-failure fixture technique) |
| HLT-06 | The coverage surface extends `GetHealthResponse` additively, or adds an rpc whose name clears every `mutatingVerbs` substring including "Index," with `wantUIServiceMethods` updated by set-equality in both directions | Architecture Pattern 4 (the field-16-is-taken correction, the `mutatingVerbs` verbatim list, `task proto:gen`'s single-invocation scope), Pitfall 5 (the two different, non-interchangeable field-stability fixtures) |

</phase_requirements>

## Summary

Phase 10 is a pure in-repo plumbing phase: no new dependency, no new UI framework, no new proto tooling. The work is (1) teach `internal/indexer.Discover` to emit one record per pre-extraction exclusion instead of silently `return nil`-ing past it, (2) give `internal/graphstore` a seventh key namespace to persist those records in the same commit batch as `File` records, (3) extend `internal/schema/graph.proto`'s `Meta` message with an additive `has_coverage` field, (4) extend the UI wire surface (`internal/uiproto/uiv1/ui.proto`) additively — a `Coverage` summary embedded in `GetHealthResponse`, plus a new paged `GetCoverage` rpc that clears every `mutatingVerbs` substring — and (5) render a new "Coverage" section on `/health`.

The most consequential correction this research makes to the phase's own working notes: **`GetHealthResponse` field 16 is already taken** (`commit_sha = 16`, landed at plan 04-03/09-01's chained fixture — confirmed against the generated descriptor's own comment trail in `readonly_test.go`), so the new `Coverage` field must be **field 17**, not 16 as the phase notes assumed. Field 15 (`stale`) is *not* the last field. Separately, **D-14(b)'s literal wording ("internal/query contains zero `filepath.WalkDir`/`os.ReadDir` calls") is already false today** — `internal/query/status.go` has two production `filepath.WalkDir` calls (`newestSourceMtime`, `dbSizeBytes`) that are pre-existing, legitimate, and unrelated to exclusion reasons. The guard must be scoped to the specific new file/function that reads coverage back, not the whole package, following the file-scoped source-scan precedent `internal/cli/editordiscovery_test.go#TestEditorDiscoverySourceNeverSpawnsAProcess` already establishes in this repo (read one named file's bytes, assert forbidden substrings absent + a positive control present) — not a package-wide grep.

**Primary recommendation:** Add a new `ExcludedFile` proto message to `internal/schema/graph.proto` with a closed `ExclusionReason` enum, a new single-byte graphstore prefix (any byte outside `{m,n,e,f,a,x}`) with a `pebbleFileIterator`-shaped iterator, wire the four exclusion decision points already visible in `Discover`'s `WalkDir` callback (dir-skip, extension-miss, build-tag-miss, plus a new size pre-check) to append to a fourth return value, commit those records in `writeGraph` (`resolve.go`) in the same `Writer` batch as `File`/`Node`/`Edge`, mirror `Meta.has_file_index`'s absent-means-unknown contract for a new `Meta.has_coverage = 9`, and extend `GetHealthResponse` with `Coverage coverage = 17` plus a new `GetCoverage` paged rpc (UIService's 16th method) — all via `task proto:gen`, updating `wantUIServiceMethods` and `uiProtoFieldNumbers` in the same commit, per the established Phase 9 recipe.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Exclusion-reason decision (vendor/dot-dir, ext, build-tag, size) | Indexer (discovery) | — | Must be decided exactly once, at the walk, per D-01/D-05; `internal/indexer/discover.go`'s `WalkDir` callback is the only place that sees every candidate file before extension/build-tag filtering discards it |
| Exclusion-record persistence | Database/Storage (`internal/graphstore`) | Indexer (writer) | New key namespace, written in the same `Writer` batch as `File`/`Node`/`Edge` at `resolve.go`'s `writeGraph` (D-07) |
| Coverage summary + paged read | API/Backend (`internal/query.Engine`) | Database/Storage | Engine reads back through `graphstore.Reader` only — never re-derives by walking disk (archtest-enforced, D-01/07-CONTEXT D-01) |
| Wire projection | API/Backend (`internal/uiserver`) | — | `GetHealthResponse.coverage` + new `GetCoverage` rpc, following `healthToProto`'s mapper convention |
| Rendering | Browser/Client (`web/src/routes/health`) | — | New "Coverage" section, reusing `CountTable`/`DataTable` shell conventions; remedy is text only (SRV-03) |

## Package Legitimacy Audit

**Not applicable this phase.** No new Go module or npm package is introduced — every dependency this phase touches (`google.golang.org/protobuf`, `github.com/cockroachdb/pebble/v2`, `connectrpc.com/connect`, the existing `@bufbuild/protoc-gen-es`-generated client) is already vendored and exercised by prior phases. `gsd_run query package-legitimacy check` was not run because there is nothing to check.

## Standard Stack

No new libraries. The phase reuses the existing stack end to end:

| Component | Already in repo | Role in this phase |
|-----------|-----------------|---------------------|
| `google.golang.org/protobuf` | yes (`internal/schema`, `internal/uiproto`) | New `ExcludedFile`/`ExclusionReason` message (schema proto); new `Coverage`/`GetCoverageRequest`/`GetCoverageResponse`/`CoverageRow` messages (UI proto) |
| `github.com/cockroachdb/pebble/v2` | yes (`internal/graphstore`) | New key-prefix namespace, `NewIter`/`DeleteRange` exactly as `f/`/`x/` already use |
| `connectrpc.com/connect` | yes (`internal/uiserver`) | New `GetCoverage` unary rpc on the existing `UIService` |
| `buf` v1.72.0 + `protoc-gen-go`/`protoc-gen-connect-go`/`protoc-gen-es` | yes (`go.tool-proto.mod`, `web/node_modules/.bin`) | `task proto:gen` regenerates **both** proto surfaces (schema + UI, Go + TS) in one invocation `[VERIFIED: Taskfile.yml:305-347]` |

No `npm install`/`go get` step is expected in this phase's plans.

## Architecture Patterns

### System Architecture Diagram

```
                     ┌───────────────────────────────────────────────┐
                     │        internal/indexer/discover.go            │
                     │        Discover(root) — one WalkDir pass        │
                     │                                                  │
  filesystem  ──────►│  dir entry                                      │
  (real repo)        │    │                                            │
                     │    ├─ ShouldSkipDir(vendor/.dot) ─► SkipDir ────┼──► exclusion: DIR_VENDOR / DIR_DOTPREFIX
                     │    │                                            │        (one record per pruned dir)
                     │  file entry                                     │
                     │    ├─ ext not registered ─────────────────────►┼──► exclusion: UNSUPPORTED_EXTENSION
                     │    ├─ go build.MatchFile() == false ───────────┼──► exclusion: BUILD_TAG
                     │    ├─ stat.SizeBytes > parser.MaxSourceBytes ──┼──► exclusion: SIZE_LIMIT   (NEW, D-04)
                     │    └─ else ──► DiscoveredFile{...} ────────────┼──► (Extract/Resolve as today)
                     └───────────────────────────────────────────────┘
                                          │                    │
                          []DiscoveredFile│         []*schema.ExcludedFile (NEW 4th return)
                                          ▼                    ▼
                     ┌───────────────────────────────────────────────┐
                     │   internal/indexer/resolve.go: writeGraph()    │
                     │   ONE graphstore.Writer batch (D-04a/D-07):    │
                     │     PutNode* / PutFile* / PutEdge*             │
                     │     PutExcludedFile* (NEW)                     │
                     │     PutMeta{ HasFileIndex, HasCoverage=true }  │
                     │     w.Commit()                                 │
                     └───────────────────────────────────────────────┘
                                          │
                                          ▼  (new 'c'-or-other prefix namespace)
                     ┌───────────────────────────────────────────────┐
                     │        internal/graphstore (pebble)            │
                     │  m/ n/ e/ f/ a/ x/  +  <new>/<path> (D-05)     │
                     └───────────────────────────────────────────────┘
                                          │  Snapshot().IterateExcludedFiles()
                                          ▼   (read-only; NEVER a disk re-walk — D-05/archtest)
                     ┌───────────────────────────────────────────────┐
                     │        internal/query.Engine (new methods)     │
                     │  CoverageSummary() / Coverage(pageToken,...)   │
                     └───────────────────────────────────────────────┘
                                          │
                                          ▼
                     ┌───────────────────────────────────────────────┐
                     │  internal/uiserver: healthToProto() extension  │
                     │  + new GetCoverage handler (16th UIService rpc)│
                     └───────────────────────────────────────────────┘
                                          │  Connect/JSON, same-origin
                                          ▼
                     ┌───────────────────────────────────────────────┐
                     │  web/src/routes/health/+page.svelte            │
                     │  new "Coverage" section: N discovered · M      │
                     │  indexed · K excluded · J extraction failures  │
                     └───────────────────────────────────────────────┘
```

The primary use case (a user opens `/health` on a repo with excluded files) traces top-to-bottom: filesystem → `Discover` classifies every visited entry → `writeGraph` commits both `File` and `ExcludedFile` records atomically → `Engine` reads the committed records back (never re-walks) → `uiserver` projects them onto the wire → the SPA renders counts + grouped rows.

### Recommended Project Structure

No new top-level packages. Additions land inside existing directories:

```
internal/indexer/
├── discover.go          # MODIFIED: WalkDir callback appends ExcludedFile records at 4 decision points
├── discoverexclusion.go # NEW (or inline in discover.go — Claude's Discretion, CONTEXT.md) — the exclusion-record builder helper
├── resolve.go            # MODIFIED: writeGraph() stages PutExcludedFile* in the same batch, stamps Meta.HasCoverage
└── sync.go               # MODIFIED: diff current exclusions (from this Sync's own Discover call) vs stored, upsert+prune

internal/graphstore/
├── keys.go               # MODIFIED: new prefix constant + key builder + prefix-scan bound
├── batch.go               # MODIFIED: PutExcludedFile / DeleteExcludedFile on pebbleWriter
├── pebble_store.go        # MODIFIED: IterateExcludedFiles on pebbleReader, new pebbleExcludedFileIterator
└── store.go                # MODIFIED: Reader.IterateExcludedFiles / Writer.PutExcludedFile / DeleteExcludedFile on the interfaces

internal/schema/
└── graph.proto            # MODIFIED: ExcludedFile message, ExclusionReason enum, Meta.has_coverage = 9

internal/query/
├── coverage.go            # NEW — Engine.CoverageSummary()/Coverage(...) reading graphstore.Reader only (no indexer import)
└── coverage_test.go        # NEW, plus a FILE-SCOPED source-scan test (D-14b) targeting coverage.go specifically, not the whole package

internal/uiserver/
├── handlers.go             # MODIFIED: healthToProto() gains Coverage; new GetCoverage handler
├── readonly_test.go         # MODIFIED: wantUIServiceMethods 15→16, uiProtoFieldNumbers +N, uiProtoFieldFixtureLenAtPlan10XX

internal/uiproto/uiv1/
└── ui.proto                # MODIFIED: GetHealthResponse.coverage = 17 (NOT 16); GetCoverage rpc + 4 new messages

web/src/lib/
├── health-view.ts           # MODIFIED: coverage row/group projections (pure, no verdict — same D-04 discipline as today)
└── client.ts                # untouched — uiClient auto-gains getCoverage() from the regenerated GenService

web/src/routes/health/
└── +page.svelte             # MODIFIED: new Coverage section
```

### Pattern 1: Discovery-time decision, never a query-time re-walk

**What:** Every exclusion reason is decided exactly once, inside `Discover`'s single `WalkDir` pass, and persisted in the same write that commits `File` records. Nothing downstream (`Engine`, `uiserver`, the SPA) ever calls `os.Stat`/`filepath.WalkDir` to reconstruct "why is this file missing."

**When to use:** Any "why did X not happen" diagnostic surface backed by a filesystem walk — the walk is the only place with ground truth at the moment the decision is made; a later re-derivation is provably a lie the instant the disk changes between index time and query time (ROADMAP's own framing, `10-CONTEXT.md` Notes).

**Example — the exact 4 decision points, `[VERIFIED: internal/indexer/discover.go:101-156]`:**
```go
walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() {
		if p != root && ShouldSkipDir(d.Name()) {
			return fs.SkipDir   // <-- decision point 1: DIR_VENDOR / DIR_DOTPREFIX
		}
		return nil
	}

	ext := strings.ToLower(filepath.Ext(d.Name()))
	spec, ok := lookupLanguageByExt(ext)
	if !ok {
		return nil               // <-- decision point 2: UNSUPPORTED_EXTENSION
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
			return nil            // <-- decision point 3: BUILD_TAG
		}
	}

	relPath, err := filepath.Rel(root, p)
	// ...
	info, err := d.Info()
	// ...
	pending = append(pending, pendingFile{ /* ... sizeBytes: info.Size() ... */ })
	// <-- decision point 4 (NEW, D-04): insert here, comparing info.Size()
	//     against parser.MaxSourceBytes BEFORE appending to pending
	return nil
})
```
Decision point 1 fires for `d.Name() == "vendor"` (reason `DIR_VENDOR`) or `strings.HasPrefix(d.Name(), ".")` (reason `DIR_DOTPREFIX`) — `ShouldSkipDir` itself `[VERIFIED: internal/indexer/discover.go:53-55]`:
```go
func ShouldSkipDir(name string) bool {
	return name == "vendor" || strings.HasPrefix(name, ".")
}
```
Distinguishing the two reasons at the call site requires checking `d.Name()` against the same two branches `ShouldSkipDir` OR's together — `ShouldSkipDir` itself only returns a bool, so the classification helper must re-inspect `d.Name()` (or `ShouldSkipDir` gains a sibling that returns the reason).

Decision point 4's threshold, `[VERIFIED: internal/parser/parser.go:15]`: `const MaxSourceBytes = 4 * 1024 * 1024 // 4 MiB`.

### Pattern 2: One `graphstore.Writer` batch, no second commit

**What:** `writeGraph` (the from-scratch `Run` path) and `Sync`'s incremental writer already stage every mutation on one `pebble.Batch` and call `Commit()` exactly once. The new `ExcludedFile` records must be staged on the SAME `Writer` instance the `File`/`Node`/`Edge` records use — never a second `NewWriter()`/`Commit()` pair, which would reopen the torn-write window D-04a exists to close.

**Where the from-scratch commit point is**, `[VERIFIED: internal/indexer/resolve.go:723-800]` (elided to the load-bearing lines):
```go
func writeGraph(store graphstore.GraphStore, nodes, packageNodes []*schema.Node, edges []*schema.Edge, files []*schema.File, commitSHA string) error {
	// ... sort nodes, files; collapseEdges(edges, nodeFilePath) ...
	w, err := store.NewWriter()
	if err != nil {
		return err
	}
	for _, n := range allNodes { /* w.PutNode(n) */ }
	for _, f := range sortedFiles { /* w.PutFile(f) */ }
	for _, e := range collapsedEdges { /* w.PutEdge(e, nodeFilePath[e.Source]) */ }

	meta := schema.NewMeta()
	meta.NodeCount = int64(len(allNodes))
	meta.EdgeCount = int64(len(collapsedEdges))
	meta.HasFileIndex = true   // <-- the D-05/D-06 precedent for HasCoverage
	meta.LastSyncUnixMs = time.Now().UnixMilli()
	meta.CommitSha = commitSHA
	if err := w.PutMeta(meta); err != nil { /* ... */ }
	return w.Commit()
}
```
This is the single site to add a sorted loop over the exclusion records (`w.PutExcludedFile(x)`) and to stamp `meta.HasCoverage = true`, mirroring `HasFileIndex`'s unconditional-on-every-writeGraph-call stamping (comment at `resolve.go:770-777` explains why: stamping at only one site would leave graphs built via other paths permanently missing the flag).

**Signature change required**, `[VERIFIED: internal/indexer/resolve.go:654]`: `func Resolve(store graphstore.GraphStore, results []goextract.FileResult, modulePath string, commitSHA string) (int, error)` calls `resolveRefs` then `writeGraph`; both need a `[]*schema.ExcludedFile` parameter threaded through from `Discover`'s new 4th return value. `Discover` itself has exactly 2 in-repo callers `[VERIFIED: internal/indexer/pipeline.go:100, internal/indexer/sync.go:85]` (`grep` confirmed no external package calls `indexer.Discover` — both call sites are inside `internal/indexer` itself), so widening its return tuple is a two-call-site mechanical change, not an external API break.

### Pattern 3: `Sync`'s incremental upsert/prune gets simpler than it looks

**What:** `Sync` (`internal/indexer/sync.go:85`) already calls `Discover(repoRoot)` on **every** invocation — the walk itself is never incremental, only the *resolve* step is (stat-based diff against stored `File` records, `[VERIFIED: internal/indexer/sync.go:96-140]`). This means every `Sync` call already produces a **complete, fresh exclusion list** for the current disk state — Sync does not need to separately re-derive "what changed among exclusions"; it can diff the freshly-discovered exclusion set against the stored `<new-prefix>/` namespace using the identical technique the existing `deleted` computation already uses for `f/` records (`[VERIFIED: internal/indexer/sync.go:141-153]`):
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
The coverage-namespace analogue: build a `map[string]struct{}` of currently-excluded paths from this Sync's own `Discover` call, iterate the stored exclusion namespace, `DeleteExcludedFile` for any stored path no longer excluded (either now-included, or gone from disk), and `PutExcludedFile` (blind upsert — no content-hash comparison needed; a changed reason for the same path is just an overwrite) for every currently-excluded path. Directory-level exclusions (`DIR_VENDOR`/`DIR_DOTPREFIX`) are naturally re-derived on every `Sync` too, since `WalkDir` visits the pruned-directory entry itself (just not its children) on every call.

### Pattern 4: Additive proto extension + same-commit fixture update (the Phase 9 recipe)

**What:** Every prior UI-surface addition in this repo (`GetPermalink`, `GetHealth`, `FileGraph`, `FileSymbols`, `WatchGraph`, `GetEditorLink`) followed one recipe: edit the `.proto`, run `task proto:gen`, update `wantUIServiceMethods` and `uiProtoFieldNumbers`/its length constant in the SAME commit as the proto change (not a follow-up commit) — `[VERIFIED: internal/uiserver/readonly_test.go:71-87, 261-271]`. `.planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-01-SUMMARY.md` documents this as "Every read-only/field-number guard (`wantUIServiceMethods` 14->15, `uiProtoFieldFixtureLenAtPlan0901`, ...) moved in the SAME commit as the proto change."

**Critical correction to the phase's own research questions** — `GetHealthResponse` is NOT capped at field 15. `[VERIFIED: internal/uiproto/uiv1/ui.proto:756-820]`:
```proto
message GetHealthResponse {
  bool initialized = 1;
  string version = 2;
  int64 file_count = 3;
  int64 node_count = 4;
  int64 edge_count = 5;
  int64 db_size_bytes = 6;
  string backend = 7;
  map<string, int64> files_by_language = 8;
  repeated string languages = 9;
  map<string, int64> nodes_by_kind = 10;
  map<string, int64> edges_by_kind = 11;
  PendingChanges pending_changes = 12;
  IndexHealth index_health = 13;
  WorktreeMismatch worktree_mismatch = 14;
  bool stale = 15;
  string commit_sha = 16;
}
```
`readonly_test.go`'s own doc comment at plan 04-03 states this explicitly: `[VERIFIED: internal/uiserver/readonly_test.go:218-222]` — "GetHealthResponse's sixteen" (field count), confirming `commit_sha = 16` was already spent (added after the initial 04-03 freeze, at a later plan). **Field 17 is the next free number for `Coverage coverage = 17`.** Do not reuse 16.

**No existing paged rpc to copy** — `[VERIFIED: repo-wide search]` no `page_token`/`page_size`/`PageToken`/`PageSize` identifier exists anywhere in the Go or proto source outside `.pb.go` generated files. `GetCoverage` is the **first** paged rpc on `UIService`; its page-token shape (opaque string vs. structured offset) is unconstrained by precedent — Claude's Discretion per `10-CONTEXT.md`.

**`mutatingVerbs` already contains "Index"** and 18 other substrings, `[VERIFIED: internal/uiserver/readonly_test.go:132-137]`:
```go
var mutatingVerbs = []string{
	"Create", "Update", "Delete", "Remove", "Set", "Put", "Post",
	"Write", "Add", "Insert", "Mutate", "Patch", "Modify", "Sync",
	"Reindex", "Index", "Clear", "Reset", "Save",
}
```
`GetCoverage` clears all 19 substrings (case-sensitive `strings.Contains`, `[VERIFIED: internal/uiserver/readonly_test.go:148-152]` — the check is `strings.Contains(name, verb)`, exact substring, case-sensitive). A decoy `GetIndexCoverage` fails on `"Index"` — this is the exact trap the phase notes warn about, confirmed live in the fixture, not merely asserted.

`task proto:gen` regenerates **both** proto surfaces (`internal/schema/graph.proto` and `internal/uiproto/uiv1/ui.proto`, Go + TypeScript) in one invocation — `[VERIFIED: Taskfile.yml:305-347]` (doc comment: "Regenerates BOTH proto surfaces this repo maintains ... in TWO languages"). There is no separate schema-only proto:gen target; one `task proto:gen` call covers the `ExcludedFile`/`ExclusionReason` addition and the `GetCoverage`/`Coverage` addition together. `task proto:drift` likewise checks all four generated files (`graph.pb.go`, `ui.pb.go`, `uiv1connect/ui.connect.go`, `web/src/lib/gen/ui_pb.ts`) in one target, `[VERIFIED: Taskfile.yml:369-373]`.

### Anti-Patterns to Avoid

- **Re-deriving exclusion reasons at query time.** `internal/query` may never import `internal/indexer` (the production root) — only its `goextract`/`nodeid` leaves — enforced by a live archtest, `[VERIFIED: internal/query/archtest/import_direction_test.go:1-30, 61-79]`, whose package doc **already anticipates this exact phase**: "Forward consequence for Phase 10 (HLT-05): the discovery-exclusion helper planned there writes reasons from inside the indexer's discovery path; internal/query reads them back through internal/graphstore only. internal/query may not import that helper — doing so would resolve the forbidden internal/indexer root transitively and fail this test." Any helper function containing the exclusion-classification logic (e.g. distinguishing `DIR_VENDOR` from `DIR_DOTPREFIX`, or re-checking `parser.MaxSourceBytes`) must live in `internal/indexer` (or a query-unreachable location), never be imported by `internal/query`.
- **A package-wide "zero WalkDir/ReadDir" scan.** `internal/query/status.go` legitimately contains two production `filepath.WalkDir` calls today — `[VERIFIED: internal/query/status.go:108, 184]` (`newestSourceMtime` for staleness, `dbSizeBytes` for store-size reporting) — both pre-existing, both unrelated to coverage. A guard asserting "zero WalkDir calls in internal/query" would be **false on day one** and either fail spuriously or (if scoped wrong) prove nothing. Scope the guard to the new coverage-reading file specifically, following the `TestEditorDiscoverySourceNeverSpawnsAProcess` pattern (`os.ReadFile("coverage.go")`, assert forbidden substrings absent + a positive control like `graphstore.Reader` or `IterateExcludedFiles` present) `[VERIFIED: internal/cli/editordiscovery_test.go:299-341]`.
- **A second `NewWriter()`/`Commit()` for exclusion records.** Breaks D-04a's one-atomic-batch guarantee and reopens a torn-write window between `File` records committing and exclusion records committing.
- **Reusing field 16 on `GetHealthResponse`.** It is spent (`commit_sha`). Use 17.
- **Inferring the discovered denominator from `len(files)` returned by today's `Discover`.** That slice already excludes extension-misses and build-tag-misses — it is NOT "every file the walker visited" (D-01's definition). The denominator must be `len(files) + len(exclusionRecords accounting for per-file, not per-directory, exclusions)`, i.e., pruned-directory exclusions (one record per directory) do **not** each count as one discovered file — their contents were never visited at all (D-01: "Pruned subtrees ... are not discovered — their contents are never enumerated"). Counting logic must NOT sum "1 discovered file per exclusion record" uniformly; directory-level records are a different unit than file-level records.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| New key-namespace scan | A custom range-scan helper | `pebble.Iterator` + `rangeUpperBound(prefix)`, copying `pebbleFileIterator`'s exact shape (`[VERIFIED: internal/graphstore/pebble_store.go:289-297, 401-436]`) | Five other namespaces (`n/`,`e/`,`f/`,`x/`) already solved this; the length-prefixed-segment encoding (`appendSegment`) is what prevents a crafted path from bleeding into an adjacent key range — re-deriving this is a security regression risk, not a simplification |
| Paged rpc token | A custom cursor/offset scheme invented from scratch | Whatever shape is chosen (Claude's Discretion — no precedent exists), but keep it OPAQUE to the client and stable across a `Sync` in between two pages, mirroring how every other `IterateX` in this codebase treats its key ordering as an implementation detail | No existing paged rpc to copy verbatim (confirmed absence, not merely unfound) — but the underlying `IterateExcludedFiles` iterator already yields a stable, deterministic (path-sorted) order for free, so a page token can be as simple as "last path returned" |
| Enum closedness proof | A `reflect`-based enumeration of every named constant | The `EditorLinkAvailability`-style pattern: exhaustively exercise each defined enum value as a real answer at the RPC boundary (mirroring `TestGetEditorLinkAvailabilityStatesAreAnswersNotErrors`, `[VERIFIED: 09-01-SUMMARY.md coverage table, id D5]`) rather than a dedicated reflection-based set-equality test — no such dedicated enum-reflection test exists today for any of this repo's closed enums, so inventing one would be a new, unprecedented pattern rather than a reused one |

**Key insight:** every mechanism this phase needs (batched atomic writer, prefix-scan iterator, additive-proto-with-fixture-update, answers-not-errors handler shape, closed enum, read-only method-set guard) already has a working, tested precedent somewhere in this codebase from a prior phase. The engineering work is composition and correct field/byte allocation, not invention.

## Runtime State Inventory

Not applicable — this is not a rename/refactor/migration phase. No trigger condition from the execution flow's Step 2.5 applies.

## Common Pitfalls

### Pitfall 1: Off-by-one on "discovered" when directories are pruned
**What goes wrong:** Treating each `ShouldSkipDir` hit as if it excluded exactly one file inflates or deflates the discovered-vs-indexed denominator.
**Why it happens:** D-01 defines "discovered" as every file the walker *visits* — a pruned directory's contents are never visited at all, so they contribute **zero** to "discovered," not one-per-file and not one-per-directory-as-if-it-were-a-file.
**How to avoid:** Keep directory-level exclusion records (`DIR_VENDOR`/`DIR_DOTPREFIX`) structurally separate from file-level exclusion records (`UNSUPPORTED_EXTENSION`/`BUILD_TAG`/`SIZE_LIMIT`) in the summary math — D-02 already mandates this at the persistence level ("one directory-level exclusion record ... never by phantom per-file rows"); the summary/count logic in `internal/query` must honor the same split, never summing both into one "discovered" total.
**Warning signs:** A fixture with one vendored directory containing 3 files reports `discovered: 4` (1 dir record miscounted as a file) or `discovered: 0` for that subtree's contribution — the correct contribution to `discovered` from a pruned directory is 0 files, with exactly one directory-level row in the reason list.

### Pitfall 2: `Export`/`Import` silently drops the new namespace
**What goes wrong:** `internal/graphstore.Export`/`Import` streams `Meta`, `Node`, `Edge`, `File` records via a fixed `exportKindX` tag set — `[VERIFIED: internal/graphstore/export.go:17-25, 190-222]`. Neither the `x/` file-index namespace nor a new exclusion namespace has an `exportKind*` entry (the `x/` namespace is reconstructed for free because `PutNode`/`PutEdge` re-derive it as a side effect of being called during import — `[VERIFIED: internal/graphstore/batch.go:39-53, 58-69]`). A new `ExcludedFile` namespace has **no such free reconstruction path** — nothing else re-derives it, so an `Export` then `Import` round-trip would silently drop every coverage record unless a `exportKindExcludedFile` tag is added to both `Export` and `importRecord`.
**Why it happens:** The export/import framing was built before this phase existed and enumerates a fixed, closed set of record kinds; a new namespace does not automatically participate.
**How to avoid:** If this phase's plans touch `Export`/`Import` (grep for callers — `ARCH-01`'s bulk export/import path), add the new record kind explicitly. If out of scope for this phase, record it as an explicit, named gap (not a silent omission) — the same discipline this research applies to `Meta.has_coverage` absence.
**Warning signs:** A round-trip test (`export.go`'s existing test suite) that only asserts `Node`/`Edge`/`File`/`Meta` counts match pre/post-export would pass green while `ExcludedFile` count silently drops from N to 0 — a false negative unless a coverage-record count is added to that same assertion.

### Pitfall 3: `Sync`'s from-scratch backfill path also needs the new field stamped
**What goes wrong:** `Sync` has a `needsFileIndexBackfill` gate (`[VERIFIED: internal/indexer/sync.go:502]`: `return !meta.GetHasFileIndex(), nil`) that, when true, delegates to the full `run()`/`Resolve()` path — this path already stamps `HasFileIndex`/`HasCoverage` via `writeGraph`. But `Sync`'s **own** two write sites that construct a fresh `schema.NewMeta()` directly (the mtime-refresh-only early return, `[VERIFIED: internal/indexer/sync.go:177]`: `newMeta.HasFileIndex = true`, and the main incremental commit, `[VERIFIED: internal/indexer/sync.go:406]`: same pattern) must ALSO stamp `newMeta.HasCoverage = true` — mirroring exactly how they already both stamp `HasFileIndex = true` independently rather than relying on a shared code path.
**Why it happens:** `schema.NewMeta()` returns a zero-valued struct on every call (`[VERIFIED: internal/schema/meta.go:17-19]`); nothing carries a previous Meta's flags forward automatically except fields the code explicitly copies (`syncCommitSHA` is the one PRESERVING exception, for a different reason — an unresolvable HEAD must not erase a known-good commit).
**How to avoid:** Grep every `schema.NewMeta()` call site in `internal/indexer` (there are at least 3: `resolve.go:773`ish inside `writeGraph`, `sync.go:~172` the mtime-refresh path, `sync.go:~400` the main incremental path) and stamp `HasCoverage = true` at every one, exactly parallel to how `HasFileIndex` is stamped at all three today.
**Warning signs:** A test indexes fresh (works), then runs one incremental `Sync` with no file changes (the mtime-refresh-only early-return path) and finds `Meta.has_coverage` flipped back to false/absent — because that specific write site was missed.

### Pitfall 4: The extraction-failure fixture file can no longer rely on oversize alone
**What goes wrong:** The precedent test for "a file fails extraction" (`TestExtractPool_OversizedFileContained`, `[VERIFIED: internal/indexer/extract_test.go:212-251]`) constructs an oversized `.go` file and feeds it directly to `Extract` — bypassing `Discover` entirely in that test. Under Phase 10's new D-04 pre-check, a REAL oversize file discovered through the normal `Discover(root)` path is now classified `SIZE_LIMIT` at discovery time and never reaches `Extract`/`parser.Parse` at all — so D-13's fixture cannot use "oversize" for both the `SIZE_LIMIT` exclusion AND the "extraction failure" `File.errors` case; those must be two genuinely different files.
**Why it happens:** Prior to this phase, oversize was the ONLY realistic way to make `parser.Parse` fail (`[VERIFIED: internal/indexer/goextract/goextract.go:27-45]`'s own doc comment: "p.Parse owns the parser.MaxSourceBytes ceiling. If Parse fails for ANY reason ... that failure is recorded on FileResult.Err" — tree-sitter itself is error-tolerant and does not fail on malformed syntax, it emits ERROR nodes instead, so a syntactically-broken-but-normal-sized `.go` file will NOT produce a `FileResult.Err`).
**How to avoid:** D-13's genuinely-extraction-failing fixture file needs a distinct failure mode from `SIZE_LIMIT`. The one other per-file failure this codebase's `Extract` pool already records is a `os.ReadFile` error (`[VERIFIED: internal/indexer/extract.go:113-127]`: "A read failure is per-file, not fatal to the batch — recorded the same way goextract.Extract records a parse failure"). A dangling symlink (`ln -s nonexistent target.go`) discovered normally (small `Lstat` size, passes the new size pre-check, matches the `.go` extension) but unreadable at `os.ReadFile` time is one deterministic way to reach this path without depending on size. **This exact technique is `[ASSUMED]`** — no existing test in this repo constructs a broken-symlink fixture; verify `filepath.WalkDir`'s `fs.DirEntry`/`d.Info()` behavior on a dangling symlink (does `Info()` itself fail, landing in Discover's own `return err` path instead of continuing to extraction?) before committing to this approach in a plan.
**Warning signs:** A "committed extraction-failure fixture" that, after this phase's D-04 lands, silently gets reclassified as `SIZE_LIMIT` and never exercises the `File.errors` code path the test intends to cover.

### Pitfall 5: `Meta`'s subset fixture is NOT exact-cardinality — don't assume it blocks a new field
**What goes wrong:** Assuming `internal/schema/meta_commit_test.go`'s `knownMetaFieldNumbers` fixture will fail the moment `has_coverage = 9` is added, the way `internal/uiserver/readonly_test.go`'s UI-proto fixture would.
**Why it happens:** The two fixtures have different contracts. `[VERIFIED: internal/schema/meta_commit_test.go:93-100]`'s own doc comment: "It is deliberately a SUBSET fixture, not an exact-cardinality assertion ... a future additive field 9 must pass this test, not break it." By contrast, `readonly_test.go`'s `uiProtoFieldNumbers` fixture is exact-both-directions (`[VERIFIED: internal/uiserver/readonly_test.go:591-627]` — Direction 2 explicitly fails on any descriptor field the fixture doesn't cover).
**How to avoid:** Adding `Meta.has_coverage = 9` does not strictly require touching `knownMetaFieldNumbers` for the test to keep passing — but the planner should still ADD the entry for documentation/drift-tracking consistency (the fixture's own doc comment frames omission as acceptable, not as preferred). There is currently **no equivalent fixture at all** for `graph.proto`'s `File` message or for a brand-new `ExcludedFile` message — `schema/graph.proto`'s field-stability guard only exists for `Meta`. If `ExcludedFile` is a new message, its field numbers have no pre-existing stability guard to extend or violate; the planner may choose to add one (mirroring `knownMetaFieldNumbers`) or rely on `TestSchemaRoundTripsUnknownFields`-style roundtrip coverage plus ordinary code review — this is a genuine open design choice, not a scripted extension of an existing fixture.
**Warning signs:** A plan task that says "extend `readonly_test.go`'s fixture to cover `ExcludedFile`" — `readonly_test.go` only walks `uiv1.GetStatusRequest{}`'s `ParentFile()` descriptor (`[VERIFIED: internal/uiserver/readonly_test.go:562-563]`), i.e., the **UI proto file only**. `ExcludedFile` lives in the **schema proto** (`internal/schema/graph.proto`) — a structurally different file, walked by a structurally different (and much weaker, subset-only) test. Conflating the two is an easy, costly mistake.

## Code Examples

### Iterator to copy for the new namespace
```go
// Source: internal/graphstore/pebble_store.go:289-297, 401-436 (verbatim structure, renamed)
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
A new `pebbleExcludedFileIterator` is the same shape against a new prefix.

### Key-namespace declaration to extend
```go
// Source: internal/graphstore/keys.go:14-37 (verbatim, current state)
const (
	prefixMeta byte = 'm'
	prefixNode byte = 'n'
	prefixEdge byte = 'e'
	prefixFile byte = 'f'
	prefixAnnotation byte = 'a' // RESERVED for post-v1 embeddings/community assignments
	prefixFileIndex byte = 'x'
)
```
Any byte outside `{m, n, e, f, a, x}` is free. `D-05` leaves the exact letter to Claude's Discretion; `c` (coverage/excluded) is an available, mnemonic choice but is not itself verified as "the" answer — this is a planning decision, not a research finding.

### `File.errors` — the existing precedent this phase must NOT duplicate
```go
// Source: internal/indexer/resolve.go:288-300 (verbatim)
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
A file that fails extraction STILL gets a `File` record (with `Errors` populated) — it counts toward `FileCount`/`file_count` today. The new `extraction_failed` count in the `Coverage` summary must be DERIVED from this existing signal (files with non-empty `Errors`) — never a second, parallel bookkeeping mechanism, and never double-counted against `indexed` (a file with `Errors` is still "indexed" in the `File`-record sense, but the summary must present it as a `coverage` gap per HLT-04's "distinguishing extraction failures ... from pre-extraction exclusions").

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `Discover` silently drops excluded files (`return nil`) | `Discover` records a reason for every exclusion | This phase | The discovered-vs-indexed denominator becomes real instead of implied |
| Size failures surface only as `parser.ErrSourceTooLarge` inside `Extract` | A stat-based pre-check at discovery time short-circuits before `Extract` ever runs | This phase (D-04) | `ErrSourceTooLarge` becomes a backstop for a race (file grows between discovery stat and read), not the primary size-limit signal |

No deprecated/outdated pattern is being replaced from an external ecosystem — this is a from-scratch feature within a stable, already-chosen internal architecture.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | A dangling symlink discovered normally but unreadable at `os.ReadFile` time is a viable, deterministic mechanism to construct D-13's "extraction failure" fixture file without relying on size (since size failures are now pre-empted by the new D-04 discovery-time check) | Common Pitfalls, Pitfall 4 | If `filepath.WalkDir`'s `d.Info()` call on a dangling symlink fails earlier (inside `Discover`'s own error path) rather than surfacing later at `os.ReadFile`, this technique doesn't produce the intended `File.errors` fixture file and the plan needs a different mechanism (e.g., a file made unreadable via permission bits, platform-dependent and CI-fragile) |
| A2 | The byte `'c'` is a reasonable mnemonic choice for the new graphstore key-namespace prefix | Code Examples | Purely cosmetic if wrong — any byte outside `{m,n,e,f,a,x}` works; this is Claude's Discretion per `10-CONTEXT.md`, not a technical constraint |

Every other claim in this document is `[VERIFIED: path:line]` against a same-session `Read` of the actual source, with a verbatim quote alongside the citation.

## Open Questions

1. **Does `internal/graphstore.Export`/`Import` need to carry the new namespace?**
   - What we know: `Export`/`Import`'s fixed `exportKindX` set (`meta`,`node`,`edge`,`file`) has no slot for a new record kind today, and unlike the `x/` file-index namespace (which is free-riding on `PutNode`/`PutEdge`'s side effects), a new `ExcludedFile` namespace has no other write path that would reconstruct it during import.
   - What's unclear: whether any current caller of `Export`/`Import` (grep `ARCH-01`'s consumers) is in this phase's blast radius, or whether it's acceptable to silently not export coverage data for now (an accepted, documented gap) given `ROADMAP.md`'s phase notes don't mention export/import at all.
   - Recommendation: the planner should explicitly decide and record which; either is defensible, but silence (neither implementing nor documenting the gap) is not, per this project's own "positive assertion" discipline.
2. **What key-prefix byte and paged-token shape to use?**
   - What we know: any byte outside `{m,n,e,f,a,x}` is free; no existing paged rpc exists to copy a token shape from.
   - What's unclear: nothing technical — this is explicitly named Claude's Discretion in `10-CONTEXT.md`.
   - Recommendation: pick the byte and token shape at plan-writing time; document the choice in the PLAN.md itself since no precedent exists to point to.
3. **Whether HLT-04's "demonstrated against a fixture repo" needs a live-browser (Playwright) proof or a Go+vitest proof suffices.**
   - What we know: no health-page live-gate script exists today (`web/scripts/` has gates for breadcrumb/graph/live-push, none for `/health`); HLT-04's criterion text says "demonstrated ... with the counts reported rather than implied," which the D-13 Go fixture test already satisfies at the indexer level. Unlike BRW-10/BRW-11, HLT-04's own requirement text does not say "verified in a live browser."
   - What's unclear: whether the planner should still add a Playwright gate for the UI rendering half (grouping/expansion, extraction-failure vs exclusion visual distinction) as an extra-mile verification, or whether a vitest unit test against a mocked `HealthClient`/`getCoverage` stub is sufficient.
   - Recommendation: default to a vitest-level UI test (matching `health-view.test.ts`'s existing convention) unless the plan's own risk assessment calls for a live gate; this keeps the phase's UI verification proportionate to its stated criteria rather than importing BRW-10's heavier bar without a textual mandate to do so.

## Environment Availability

No external tool/service dependency is introduced by this phase beyond what every prior phase already requires (Go toolchain, `buf`+plugins, Node/pnpm for the SPA build). All are already verified present and pinned by the existing Taskfile targets this phase reuses (`proto:gen`, `proto:drift`, `test:unit`, `web:test`). No new row is warranted.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Go framework | `go test` (standard library), invoked via `task test:unit` — `[VERIFIED: Taskfile.yml:117-125]` |
| Go toolchain pin | `go.mod` declares `go 1.26.6` `[VERIFIED: go.mod:3]`; local `go version` reports `go1.27.1` `[VERIFIED: shell]` — **every `go build`/`go test`/`go vet` invocation in this phase's plans MUST be prefixed `GOTOOLCHAIN=go1.26.6`**, exactly as `task test:tmux` already does (`[VERIFIED: Taskfile.yml:239]`) |
| Web framework | `vitest` via `pnpm test` (`"test": "vitest run"`, `[VERIFIED: web/package.json:14]`), wrapped by `task web:test` with a numTotalTests floor check `[VERIFIED: Taskfile.yml:657-680ish]` |
| Live-browser gates | Committed `.mjs` scripts under `web/scripts/` using real Chromium via `@playwright/test`'s `chromium` launcher, `startCodegraphUi` + `pollUntil` conventions (`[VERIFIED: web/scripts/breadcrumb-check.mjs:1-36]`) — no health-page-specific script exists yet; see Open Question 3 |
| Config files | `go.mod` (module root); `web/vitest.config.ts` (not read this session, presumed present per every other phase's convention); no dedicated Taskfile target exists yet for a coverage-specific live gate |
| Quick run command (Go) | `GOTOOLCHAIN=go1.26.6 go test ./internal/indexer/... ./internal/graphstore/... ./internal/query/... ./internal/uiserver/... ./internal/schema/...` |
| Quick run command (web) | `cd web && pnpm test -- health` (vitest name-filter) |
| Full suite command | `task test:unit` (Go, excludes `internal/daemon`), `task test:golden`, `task web:test`, `task proto:drift` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|--------------------|-------------|
| HLT-05 | 4 exclusion reasons recorded at discovery time | unit | `GOTOOLCHAIN=go1.26.6 go test -run TestDiscover ./internal/indexer/...` | ❌ Wave 0 — new test file for `discover.go`'s exclusion-record helper |
| HLT-05 | reasons survive process restart, no re-walk (D-14a behavioural) | integration | `GOTOOLCHAIN=go1.26.6 go test -run TestCoverage.*NoReWalk ./internal/indexer/...` | ❌ Wave 0 — indexes fixture, mutates disk without re-indexing, asserts stale-but-persisted read-back |
| HLT-05 | `internal/query` never walks disk for coverage (D-14b structural) | unit (source-scan) | `GOTOOLCHAIN=go1.26.6 go test -run TestCoverageSourceNeverWalks ./internal/query/...` | ❌ Wave 0 — file-scoped scan of the new `coverage.go`, per Pitfall/Anti-Pattern above |
| HLT-05 | additive within `SchemaVersion 1`, old graph opens with coverage unknown | unit | `GOTOOLCHAIN=go1.26.6 go test -run TestHasCoverage ./internal/query/... ./internal/schema/...` | ❌ Wave 0 — mirrors `TestIndexMetaOnAGraphWithNoMeta`/`TestKnownMetaFieldNumbersAreStable` shape |
| HLT-06 | `GetHealthResponse` additive extension OR new rpc clearing `mutatingVerbs` | unit | `GOTOOLCHAIN=go1.26.6 go test -run TestUIService ./internal/uiserver/...` | ✅ `internal/uiserver/readonly_test.go` exists, extended in-place |
| HLT-04 | discovered/indexed/excluded/extraction-failed counts reported against a real fixture, both kinds distinguished | integration | `GOTOOLCHAIN=go1.26.6 go test -run TestCoverageFixture ./internal/indexer/...` | ❌ Wave 0 — the D-13 fixture module + test |
| HLT-04 | UI renders the distinction | unit (vitest) | `cd web && pnpm test -- health` | ✅ `web/tests/health-page.test.ts`/`health-view.test.ts` exist, extended in-place |

### Sampling Rate
- **Per task commit:** the quick Go command above scoped to touched packages, plus `pnpm test -- health` for any web-touching task
- **Per wave merge:** `task test:unit && task test:golden && task web:test && task proto:drift`
- **Phase gate:** full suite green before `/gsd-verify-work`, plus `go vet ./...` (`task vet`) since this phase adds new production code paths

### Wave 0 Gaps
- [ ] A new `internal/indexer` test file covering the 4 exclusion decision points + the D-13 fixture module under `internal/indexer/testdata/coverage/`
- [ ] A new `internal/query/coverage_test.go` including the file-scoped D-14(b) source-scan test
- [ ] A new `internal/schema` roundtrip/field-stability entry for `Meta.has_coverage` (optional per Pitfall 5, but recommended) and, if a new `ExcludedFile` message field-stability fixture is desired, that's net-new infrastructure (no existing per-message fixture beyond `Meta`'s)
- [ ] Decision on whether a live Playwright gate for `/health`'s Coverage section is in scope (Open Question 3) — if yes, a new `web/scripts/coverage-check.mjs` following `breadcrumb-check.mjs`'s conventions is Wave 0 infrastructure, not reuse

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | no | No auth surface changes; `GetCoverage` rides the existing loopback-only, unauthenticated `UIService` (unchanged trust model from every other rpc) |
| V3 Session Management | no | No session concept in this server |
| V4 Access Control | no | No new access-control decision; read-only surface, same as every other `UIService` rpc |
| V5 Input Validation | yes | `GetCoverageRequest`'s `reason` filter (an `ExclusionReason` enum) is validated by protobuf's own closed-enum decoding — an unrecognized wire value decodes to the zero value (`UNSPECIFIED`) rather than an arbitrary string, so no manual string-validation is needed for that field. `page_token` (opaque, server-defined) must be treated as untrusted input on the way IN (a malformed/tampered token must degrade to "start from the beginning," never panic or index out-of-bounds) |
| V6 Cryptography | no | No new secret, token-signing, or cryptographic operation — the page token (if any) is not a security boundary, merely a resumption cursor over already-public-to-the-loopback-client data |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| Adversarial filenames/directory names in the repo being indexed (arbitrary bytes, control characters, extreme length) flowing into `ExcludedFile.path`/`.detail` | Tampering | **Inherited, not new**: `schema.File.path` already carries the identical exposure today (walker-produced, unsanitized) — the new `ExcludedFile.path`/`.detail` fields inherit the same disposition. The concrete mitigation at the RENDER boundary is Svelte's default text-interpolation escaping (never `{@html}` for a path or detail string) — the same discipline `09-SECURITY.md`'s T-09-06 already documents for a different field (`{@html}` stays confined to `SourcePane.svelte`'s syntax-highlighted source, never fed a raw path) |
| A malformed/tampered `page_token` in `GetCoverageRequest` | Tampering / Denial of Service | Decode defensively; an unparseable or out-of-range token must degrade to "first page," never panic, never construct an out-of-bounds Pebble iterator bound directly from client-controlled bytes without validation |
| A future `ExclusionReason` value the current UI build doesn't recognize (schema/wire skew across an upgrade) | — (a robustness concern, not a STRIDE category) | Closed proto enum with `UNSPECIFIED = 0` as the explicit unknown-default (D-08); the SPA must render `UNSPECIFIED`/unrecognized values as an explicit "unknown reason" state, never crash or silently omit the row — mirroring how `has_coverage` absent must render "unknown," never `0/0` |
| "Re-index this file" style remediation actions | Elevation of Privilege | **Out of scope by construction** (SRV-03, confirmed in `REQUIREMENTS.md`'s Out of Scope table: "'Re-index this file' auto-remediation on the coverage view — Same SRV-03 violation; show reason and remedy as text only"). No plan in this phase may add any button, link, or rpc that triggers an index/sync action from the coverage view. |

No new trust boundary is created — `GetCoverage` registers on the same Connect mux, behind the same `originHostGuard`, that every other `UIService` rpc already uses (the T-09-08 "inherited" disposition pattern from `09-SECURITY.md` applies identically here).

## Sources

### Primary (HIGH confidence — same-session `Read` of actual source)
- `internal/indexer/discover.go` (lines 1-249) — `Discover`, `ShouldSkipDir`, the 4 exclusion decision points, `DiscoveredFile`
- `internal/indexer/pipeline.go` (lines 1-179) — `Run`, `run`, `Discover`'s only production call site outside `sync.go`
- `internal/indexer/sync.go` (lines 1-260, plus 406, 502) — `Sync`, `needsFileIndexBackfill`, the mtime-refresh and incremental commit paths
- `internal/indexer/resolve.go` (lines 250-340, 640-800) — `resolveRefsWithIndex`'s `File.Errors` construction, `Resolve`, `writeGraph`
- `internal/indexer/extract.go` (lines 1-180) — `Extract`, the per-file read-failure and parse-failure recording contract
- `internal/indexer/extract_test.go` (lines 210-251) — `TestExtractPool_OversizedFileContained`, the pre-Phase-10 oversize-fixture precedent
- `internal/indexer/goextract/goextract.go` (lines 1-45) — `p.Parse` owning the size ceiling; tree-sitter's error-tolerance implication
- `internal/parser/parser.go` (line 15) — `MaxSourceBytes = 4 * 1024 * 1024`
- `internal/graphstore/keys.go` (whole file) — the 6 taken prefixes, `appendSegment`, `rangeUpperBound`
- `internal/graphstore/store.go` (whole file) — `GraphStore`/`Reader`/`Writer` interfaces
- `internal/graphstore/batch.go` (whole file) — `pebbleWriter`, `PutFile`, `DeleteFileSubgraph`
- `internal/graphstore/pebble_store.go` (lines 280-450) — `pebbleReader.IterateFiles`, `pebbleFileIterator`
- `internal/graphstore/export.go` (lines 1-60, 180-222) — `exportKind*` tags, `importRecord`, the namespace-drop gap
- `internal/schema/graph.proto` (whole file) — `Node`/`Edge`/`File`/`Meta` field numbers and reserved ranges
- `internal/schema/meta.go` (whole file) — `SchemaVersion`, `NewMeta`, `IndexedCommitSHA`
- `internal/schema/meta_commit_test.go` (whole file) — the subset-fixture contract for `Meta`
- `internal/schema/roundtrip_test.go` (lines 1-40) — the reserved-range forward-compat precedent
- `internal/query/status.go` (whole file) — `StatusResult`, `shouldSkipStaleDir`, the 2 pre-existing `filepath.WalkDir` calls
- `internal/query/meta_test.go` (lines 1-80) — the `HasFileIndex` precedent test shape
- `internal/query/archtest/import_direction_test.go` (whole file) — the GRD-02 archtest, its Phase-10 forward-consequence comment
- `internal/uiproto/uiv1/ui.proto` (lines 756-822, and the `GetHealthResponse` block) — the field-16-is-taken correction
- `internal/uiserver/handlers.go` (lines 900-1050) — `healthToProto`, `GetHealth` handler shape
- `internal/uiserver/readonly_test.go` (whole file) — `wantUIServiceMethods`, `mutatingVerbs`, `uiProtoFieldNumbers`, both directions of the field-stability test
- `internal/cli/editordiscovery_test.go` (lines 299-341) — the file-scoped source-scan precedent
- `web/src/lib/health-view.ts`, `web/src/routes/health/+page.svelte` (whole files) — current `/health` composition
- `web/src/lib/client.ts` (whole file) — the Connect client factory pattern
- `web/scripts/breadcrumb-check.mjs` (lines 1-60) — the live-gate convention
- `Taskfile.yml` (lines 110-180, 295-390, 640-680) — `test:unit`, `proto:gen`, `proto:drift`, `web:test`, `test:tmux`'s `GOTOOLCHAIN` precedent
- `go.mod` (line 3) — `go 1.26.6`
- `.planning/config.json` — `security_enforcement: true`, `security_asvs_level: 1`, `nyquist_validation: true`, all web-search providers `false`
- `.planning/phases/10-index-health-the-coverage-denominator/10-CONTEXT.md` — D-01 through D-17, the full decision set this research serves
- `.planning/phases/07-guards-that-cannot-fire/07-CONTEXT.md` — D-01 through D-04, the archtest boundary Phase 10 must respect
- `.planning/phases/09-source-view-follow-through-breadcrumb-editor-handoff/09-01-SUMMARY.md`, `09-SECURITY.md` — the additive-rpc recipe and house SECURITY.md format
- `.planning/REQUIREMENTS.md` — HLT-04, HLT-05, HLT-06 verbatim, and the "Re-index this file" out-of-scope row

### Secondary (MEDIUM confidence)
- None — no web/Context7 lookups were performed or needed for this phase's domain.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new libraries, every mechanism has a working in-repo precedent read this session
- Architecture: HIGH — every write/read path, key namespace, and wire-field-number claim was confirmed against actual source, not training-data recall
- Pitfalls: HIGH for Pitfalls 1, 2, 3, 5 (each backed by a direct source read); MEDIUM for Pitfall 4's specific symlink technique (flagged `[ASSUMED]` in the Assumptions Log — the underlying problem statement is HIGH confidence, the proposed fixture technique is not yet verified against this repo's actual `Discover`/`Extract` behavior on a dangling symlink)

**Research date:** 2026-09-12
**Valid until:** No external dependency to go stale; valid until the next phase that touches `internal/schema/graph.proto`, `internal/uiproto/uiv1/ui.proto`, or `internal/indexer/discover.go` changes the baseline this research was read against (i.e., effectively until Phase 10 lands, since nothing here depends on an external release cadence).
