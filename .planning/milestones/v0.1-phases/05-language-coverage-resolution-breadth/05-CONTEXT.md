# Phase 5: Language Coverage & Resolution Breadth - Context

**Gathered:** 2026-07-11
**Status:** Ready for planning

> Captured in `--auto` mode: all gray areas auto-selected, recommended default
> chosen for each. Decisions below are locked for research/planning; the planner
> may refine the *how* but not re-open *whether*.

<domain>
## Phase Boundary

Extend CodeGraph Go from a Go-only indexer to a multi-language one, at TS
CodeGraph v1.3.x parity, delivering exactly five things (ROADMAP success
criteria, requirements LANG-02..07, RES-02, RES-03):

1. **Full extraction + cross-file resolution** for the priority-4 languages —
   Java (LANG-02), C# (LANG-03), Python (LANG-04), TypeScript/JavaScript
   (LANG-05) — each validated on a real repo in that language.
2. **Mainstream-tier support** (LANG-06) for Rust, Ruby, PHP, C/C++, Swift,
   Kotlin at *full or explicitly documented-partial* coverage.
3. **Interface→implementation dispatch synthesis** (RES-02) — Go implicit
   interfaces, Java/C# declared implementations — so callers/impact traverse
   dynamic dispatch.
4. **Provenance tagging** (RES-03) — every synthesized (heuristic) edge carries
   `provenance: heuristic` + source location, distinct from ground-truth `ast`
   edges.
5. **Framework-aware routing** (LANG-07) for Gin, Spring, ASP.NET,
   Django/Flask/FastAPI, Express/NestJS — `route` nodes linked to handlers.

**Not in this phase (scope anchors — belong elsewhere):**
- Agent installers / CLI lifecycle / self-upgrade → Phase 6.
- Persisted reverse-edge index → Phase 8 (dispatch traversal reuses Phase-3
  in-memory query-time reverse adjacency, NOT a new persisted index).
- Embeddings, community detection, graph-viz — post-v1 milestones.
- New capabilities beyond TS v1.3.x parity — parity + performance is the bar.

</domain>

<decisions>
## Implementation Decisions

### Extractor & discovery architecture
- **D-01:** Generalize the Go-hardwired `internal/indexer/goextract` into a
  **shared `LanguageExtractor` interface + per-language extractor packages**
  (e.g. `internal/indexer/<lang>extract`) selected through a **registry keyed
  by language ID**. Reuse the existing shared vocabulary — `FileResult`,
  `ExtractedNode`, `IntraEdge`, `UnresolvedRef`, the `Kind*` node constants,
  and the `RefKind*` reference constants — extending it **additively** for
  language-specific kinds (e.g. `route`, and any new node kinds a language
  needs). The parser layer is already being generalized the same way
  (`cgo.NewGoParser()`, `cgo.NewPythonParser()` exist today) — the extractor
  layer follows that per-language-constructor pattern.
- **D-02:** Where a grammar ships `queries/tags.scm`, those tree-sitter tag
  queries MAY be used to cut per-language extraction boilerplate, but
  resolution logic and the codegraph-vocabulary mapping stay in Go for control
  and determinism. Query-driven extraction is an implementation lever, not an
  architecture — do NOT replace the imperative extractor seam with a pure-`.scm`
  generic engine.
- **D-03:** File discovery generalizes from the Go-only `Discover`
  (`go.mod` + `go/build.MatchFile` + `.go` filter) to an **extension→language
  registry driving a generic recursive walker** that reuses `ShouldSkipDir`.
  Per-language **project-descriptor hooks** resolve module/namespace identity
  (go.mod, package.json/tsconfig, pom.xml/build.gradle, `*.csproj`,
  pyproject/setup.py, Cargo.toml, composer.json, …). Go's existing go.mod
  path resolution becomes the first implementation of that hook. A file whose
  language has no descriptor still gets extracted with path-based identity —
  discovery never silently drops a supported extension.

### Cross-file resolution (per-language, tiered)
- **D-04:** Cross-file resolution becomes **per-language behind a shared
  resolver seam** (today `resolve.go`/`symbolindex.go` import `goextract`
  directly and assume Go package-alias semantics). Fidelity is **tiered to the
  success criteria**: priority-4 (Java, C#, Python, TS/JS) get full import +
  call + inheritance resolution validated on real repos; mainstream-6 get
  extraction + best-effort resolution with gaps explicitly documented (D-11).
- **D-05:** **Fold in the three deferred Go resolution items** parked at
  "Phase 5" by prior decisions — they define the resolution patterns the new
  languages inherit and are in-scope for "Resolution Breadth":
  - **WR-01** — same-package func/method name collision overwriting in
    `symbolindex` (from `esdnwn12gg`).
  - **WR-02** — selector calls on non-identifier operands mis-resolved as
    same-package (from `esdnwn12gg`).
  - **Call-as-argument extraction gap** — a call passed as an argument to
    another call is not resolved into a `calls` edge (from `252e2sav94`;
    a Phase-2 extraction gap, not a resolution one). The deliberate
    `RefKindCalls`-only `callees` scoping (excluding non-call references)
    stays as-is — that is architectural, not a bug.
  - *Planner note:* confirm these three fit the phase budget; if the phase is
    already large, they may split into their own plan(s) but stay within
    Phase 5.

### Dynamic-dispatch synthesis & provenance (RES-02 / RES-03)
- **D-06:** Synthesize **`implements` edges** (concrete type → interface) at
  resolve time — Go via structural method-set match (name + arity bounded to
  avoid quadratic blowup on wide interfaces), Java/C# via declared
  `implements`/`: Interface`. Callers/impact/affected **traverse dynamic
  dispatch by following `implements` at query time** (reuse the Phase-3
  in-memory query-time reverse adjacency), rather than materializing an
  O(callers × implementations) cartesian call→impl edge explosion. This keeps
  the graph size linear and defers any persisted reverse index to Phase 8.
- **D-07:** **No schema-version bump.** The `schema.Edge` record already
  reserves everything RES-03 needs (verified in `internal/schema/graph.pb.go`):
  set `Edge.Provenance = "heuristic"` on every synthesized edge, carry source
  location in `Edge.Line`/`Edge.Col`, and use the open `Edge.Metadata` bag for
  edge-kind-specific detail (which heuristic fired, HTTP method, etc.).
  Ground-truth AST edges keep `Provenance = "ast"`. All additive within
  `SchemaVersion 1` (D-02a additive-only discipline). The Edge `provenance`
  field comment literally reserves `"heuristic"` as "Phase 5's addition."

### Framework-aware routing (LANG-07)
- **D-08:** A **per-framework detector registry** keyed by (language,
  framework signature) — Gin, Spring, ASP.NET, Django/Flask/FastAPI,
  Express/NestJS. Each detector emits a new **`route` kind node** (route path +
  HTTP method stored in `Edge.Metadata`/node fields) linked to its handler
  symbol via a heuristic-provenance `handles`/`route` edge.
- **D-09:** Framework detectors are **opt-in per detected dependency** (fire
  only when the framework's dependency/import signature is present), not
  always-on scanning — keeps cost proportional and avoids false-positive routes
  in repos that don't use the framework.

### Coverage policy & documentation
- **D-11:** Ship a **language capability matrix** — both a committed
  human-readable doc and a machine-readable capability descriptor per language
  (extraction / resolution / dispatch / routing: full | partial | none).
  Priority-4 = full across the board; mainstream-6 = extraction + best-effort
  resolution, every gap named. "Documented-partial" (LANG-06) means the gap is
  written down in the matrix, not silently missing.

### Validation & parity method
- **D-12:** **Reuse the Phase-3 TS-CodeGraph golden-parity harness.** For each
  priority-4 language, run TS CodeGraph v1.3.x on a curated real repo, capture
  golden output, and diff **shape (not byte)** against our output — the same
  drop-in-parity bar Phase 3 used against the pinned `weft` checkout.
  Mainstream-6 languages get lighter self-consistency + spot-check validation,
  with coverage recorded in the D-11 matrix.

### Claude's Discretion
- Exact package layout/naming for per-language extractors and resolvers.
- Which specific real-world repos serve as each language's validation corpus
  (researcher/planner selects; must be representative and license-clean).
- Whether the three Go fixes (D-05) land in one plan or split across plans,
  provided all stay inside Phase 5.
- Precise `route`/`implements`/`handles` edge-kind string names and metadata
  key names (subject to TS parity check in research).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` §"Phase 5: Language Coverage & Resolution Breadth"
  (lines ~173-186) — goal, depends-on (Phase 2), requirement list, the 5
  success criteria that gate completion.
- `.planning/REQUIREMENTS.md` — LANG-02..07 (lines 64-70), RES-02/RES-03
  (lines 19-20); the coverage map (lines 136-137, 164-169).
- `.planning/PROJECT.md` — parity bar, language priority order, out-of-scope
  fence (no capabilities beyond TS v1.3.x parity).

### Parser & extraction seam (what this phase generalizes)
- `internal/parser/parser.go` — the backend-neutral `parser.Parser` seam,
  `MaxSourceBytes` (4 MiB) DoS ceiling, and the crash-isolation contract
  (a grammar's external-scanner segfault is NOT recover()-able — matters for
  Python INDENT/DEDENT and other external scanners the new languages add).
- `internal/parser/cgo/parser_cgo.go` — per-language parser constructors
  (`NewGoParser`, `NewPythonParser` already present); the pattern new grammars
  follow.
- `internal/indexer/goextract/` — the reference Go extractor: `goextract.go`
  (imperative tree-sitter walk → codegraph vocabulary), `types.go` (the
  `Kind*` and `RefKind*` vocabulary every language maps onto), `goextract_test.go`
  (table-driven node-kind mapping test — the shape each new language's tests
  mirror).
- `internal/indexer/discover.go` — Go-only discovery (`go.mod` + `go/build`);
  `ShouldSkipDir` is the shared skip predicate the generic walker must reuse.
- `internal/indexer/resolve.go`, `internal/indexer/symbolindex.go` — the
  Go-specific two-pass resolver that D-04 generalizes; also where WR-01/WR-02
  (D-05) live.
- `internal/indexer/pipeline.go` — hardwired to `goextract.Extract` today;
  the dispatch point that must select an extractor by language.

### Schema (RES-02/RES-03 representation — already prepared)
- `internal/schema/graph.pb.go` — `Edge` type (line ~214): `Provenance`,
  `Line`, `Col`, and `Metadata` fields; the `Provenance` comment explicitly
  reserves `"heuristic"` for Phase 5. `Node` carries `Language` + start/end
  line/col.
- `internal/schema/meta.go` — `SchemaVersion = 1` and the additive-only
  discipline (D-02a) that keeps this phase from bumping the version.

### Prior-phase decisions this phase depends on
- `.planning/phases/02-go-indexing-pipeline/02-CONTEXT.md`,
  `.../02-RESEARCH.md` — the node-id/edge-key model
  (`<kind>:<sha256(kind+qualified_name+file_path)>`; one edge per (src,kind,dst)),
  the two-pass extract→resolve pipeline, the ratified "no `field` node" skip.
- `.planning/phases/03-query-engine-mcp-server/03-CONTEXT.md` — the query-time
  reverse-adjacency (D-04) that dispatch traversal (D-06) reuses; the golden
  TS-parity harness (D-12) originates here.
- `.planning/PARSER-DECISION.md` — Option A (CGo tree-sitter) is the ratified
  parser strategy; new grammars are per-language CGo modules; wazero WASM stays
  a monitored future option behind the `parser.Parser` seam.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`parser.Parser` seam + per-language constructors** — `cgo.NewGoParser`,
  `cgo.NewPythonParser` already exist; adding a grammar is a bounded
  `New<Lang>Parser` + `go.mod` require, no seam change (`internal/parser/cgo`).
- **Shared extraction vocabulary** — `FileResult`/`ExtractedNode`/`IntraEdge`/
  `UnresolvedRef` + `Kind*`/`RefKind*` constants are language-agnostic enough
  to reuse; extend additively (`internal/indexer/goextract/types.go`).
- **`ShouldSkipDir`** — the one skip predicate discovery and the Phase-4
  watcher both call; the generic multi-language walker must reuse it, not fork
  it (`internal/indexer/discover.go`).
- **Query-time reverse adjacency** — Phase-3 traversal that dispatch synthesis
  rides on instead of a persisted reverse index.
- **`Edge.Provenance` / `Edge.Line` / `Edge.Col` / `Edge.Metadata`** — reserved
  in Phase 2 precisely for this phase's heuristic edges; no schema change.
- **Golden TS-parity harness** — the Phase-3 corpus-diff machinery, reused per
  priority language.

### Established Patterns
- **Additive-only within `SchemaVersion 1`** (D-02a) — new node/edge kinds and
  metadata keys are allowed; field deletion/repurposing and version bumps are
  not, unless a wire-incompatible change is truly forced.
- **Determinism is load-bearing** — stable sort by RelPath at discovery, one
  collapsed edge per (src,kind,dst), byte-identical rebuild. Every new language
  extractor and every synthesized edge MUST preserve deterministic ordering
  and idempotent output.
- **Ground-truth vs heuristic separation** — `provenance` distinguishes AST
  facts from inferred edges; this phase is the first to write `"heuristic"`.

### Integration Points
- `pipeline.go` `Extract()` — must dispatch to a language-selected extractor
  instead of calling `goextract.Extract` directly.
- `discover.go` `Discover()` — must walk by an extension→language registry and
  invoke per-language project-descriptor hooks for module identity.
- `resolve.go` / `symbolindex.go` — must select a per-language resolver and is
  where the deferred Go fixes (WR-01/WR-02) land.
- The new dispatch/route synthesis runs in Pass 2 (resolve), where the global
  cross-file symbol index exists.

</code_context>

<specifics>
## Specific Ideas

- Language priority order is fixed by PROJECT.md: Go (done) → Java/C# → Python
  → TS/JS → mainstream remainder. Java/C# lead because Sean's work team is
  Java/C#-heavy and open-source release is day-one; parity gaps are felt
  immediately since Sean runs TS CodeGraph daily.
- Parity is measured **shape, not byte**, against TS CodeGraph v1.3.x — the
  same standard Phases 2-3 used.
- Crash-isolation caveat is real for the new grammars: Python's INDENT/DEDENT
  and other languages' external C scanners run in-process with no recover()
  safety net — the `MaxSourceBytes` ceiling is the front-line mitigation.

</specifics>

<deferred>
## Deferred Ideas

- **Persisted reverse-edge index** — dispatch traversal deliberately uses
  Phase-3 in-memory query-time reverse adjacency; a persisted reverse index
  stays Phase 8 (per `9t8ss4d3vs`).
- **wazero WASM parser backend** — remains a monitored future option behind the
  `parser.Parser` seam if CGo grammar crash-isolation proves painful in
  practice; not opened in this phase.
- **Non-call reference edges** (constants, interface-type usage as references)
  — TS surfaces these; our `RefKindCalls`-only `callees` scoping deliberately
  excludes them. Architectural, not a bug; revisit only if a future
  requirement demands it (per `252e2sav94`).

None of the above are re-opened here.

</deferred>

---

*Phase: 5-Language Coverage & Resolution Breadth*
*Context gathered: 2026-07-11*
