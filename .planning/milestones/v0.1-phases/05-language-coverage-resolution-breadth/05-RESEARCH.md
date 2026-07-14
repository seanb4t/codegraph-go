# Phase 5: Language Coverage & Resolution Breadth - Research

**Researched:** 2026-07-11
**Domain:** Multi-language tree-sitter extraction, per-language cross-file resolution, dynamic-dispatch synthesis, framework-aware route detection
**Confidence:** MEDIUM — grammar/version facts and TS CodeGraph parity-target behavior are VERIFIED against authoritative sources (Go module proxy, live GitHub source); per-language resolution *semantics* (Java/C#/Python/TS-JS import models) are ASSUMED (well-established language knowledge, not verified against a live compiler this session).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Extractor & discovery architecture**
- **D-01:** Generalize the Go-hardwired `internal/indexer/goextract` into a shared `LanguageExtractor` interface + per-language extractor packages (e.g. `internal/indexer/<lang>extract`) selected through a registry keyed by language ID. Reuse the existing shared vocabulary — `FileResult`, `ExtractedNode`, `IntraEdge`, `UnresolvedRef`, the `Kind*` node constants, and the `RefKind*` reference constants — extending it additively for language-specific kinds (e.g. `route`, and any new node kinds a language needs). The parser layer is already being generalized the same way (`cgo.NewGoParser()`, `cgo.NewPythonParser()` exist today) — the extractor layer follows that per-language-constructor pattern.
- **D-02:** Where a grammar ships `queries/tags.scm`, those tree-sitter tag queries MAY be used to cut per-language extraction boilerplate, but resolution logic and the codegraph-vocabulary mapping stay in Go for control and determinism. Query-driven extraction is an implementation lever, not an architecture — do NOT replace the imperative extractor seam with a pure-`.scm` generic engine.
- **D-03:** File discovery generalizes from the Go-only `Discover` (`go.mod` + `go/build.MatchFile` + `.go` filter) to an extension→language registry driving a generic recursive walker that reuses `ShouldSkipDir`. Per-language project-descriptor hooks (go.mod, package.json/tsconfig, pom.xml/build.gradle, `*.csproj`, pyproject/setup.py, Cargo.toml, composer.json, …) resolve module/namespace identity. Go's existing go.mod path resolution becomes the first implementation of that hook. A file whose language has no descriptor still gets extracted with path-based identity — discovery never silently drops a supported extension.

**Cross-file resolution (per-language, tiered)**
- **D-04:** Cross-file resolution becomes per-language behind a shared resolver seam (today `resolve.go`/`symbolindex.go` import `goextract` directly and assume Go package-alias semantics). Fidelity is tiered to the success criteria: priority-4 (Java, C#, Python, TS/JS) get full import + call + inheritance resolution validated on real repos; mainstream-6 get extraction + best-effort resolution with gaps explicitly documented (D-11).
- **D-05:** Fold in the three deferred Go resolution items parked at "Phase 5" by prior decisions — WR-01 (same-package func/method name collision overwriting in `symbolindex`), WR-02 (selector calls on non-identifier operands mis-resolved as same-package), and the call-as-argument extraction gap (a call passed as an argument to another call is not resolved into a `calls` edge). The deliberate `RefKindCalls`-only `callees` scoping (excluding non-call references) stays as-is — that is architectural, not a bug. Planner note: confirm these three fit the phase budget; if the phase is already large, they may split into their own plan(s) but stay within Phase 5.

**Dynamic-dispatch synthesis & provenance (RES-02 / RES-03)**
- **D-06:** Synthesize `implements` edges (concrete type → interface) at resolve time — Go via structural method-set match (name + arity bounded to avoid quadratic blowup on wide interfaces), Java/C# via declared `implements`/`: Interface`. Callers/impact/affected traverse dynamic dispatch by following `implements` at query time (reuse the Phase-3 in-memory query-time reverse adjacency), rather than materializing an O(callers × implementations) cartesian call→impl edge explosion. This keeps the graph size linear and defers any persisted reverse index to Phase 8.
- **D-07:** No schema-version bump. The `schema.Edge` record already reserves everything RES-03 needs: set `Edge.Provenance = "heuristic"` on every synthesized edge, carry source location in `Edge.Line`/`Edge.Col`, and use the open `Edge.Metadata` bag for edge-kind-specific detail (which heuristic fired, HTTP method, etc.). Ground-truth AST edges keep `Provenance = "ast"`. All additive within `SchemaVersion 1` (D-02a additive-only discipline).

**Framework-aware routing (LANG-07)**
- **D-08:** A per-framework detector registry keyed by (language, framework signature) — Gin, Spring, ASP.NET, Django/Flask/FastAPI, Express/NestJS. Each detector emits a new `route` kind node (route path + HTTP method stored in `Edge.Metadata`/node fields) linked to its handler symbol via a heuristic-provenance `handles`/`route` edge.
- **D-09:** Framework detectors are opt-in per detected dependency (fire only when the framework's dependency/import signature is present), not always-on scanning — keeps cost proportional and avoids false-positive routes in repos that don't use the framework.

**Coverage policy & documentation**
- **D-11:** Ship a language capability matrix — both a committed human-readable doc and a machine-readable capability descriptor per language (extraction / resolution / dispatch / routing: full | partial | none). Priority-4 = full across the board; mainstream-6 = extraction + best-effort resolution, every gap named. "Documented-partial" (LANG-06) means the gap is written down in the matrix, not silently missing.

**Validation & parity method**
- **D-12:** Reuse the Phase-3 TS-CodeGraph golden-parity harness. For each priority-4 language, run TS CodeGraph v1.3.x on a curated real repo, capture golden output, and diff shape (not byte) against our output — the same drop-in-parity bar Phase 3 used against the pinned `weft` checkout. Mainstream-6 languages get lighter self-consistency + spot-check validation, with coverage recorded in the D-11 matrix.

### Claude's Discretion
- Exact package layout/naming for per-language extractors and resolvers.
- Which specific real-world repos serve as each language's validation corpus (researcher/planner selects; must be representative and license-clean).
- Whether the three Go fixes (D-05) land in one plan or split across plans, provided all stay inside Phase 5.
- Precise `route`/`implements`/`handles` edge-kind string names and metadata key names (subject to TS parity check in research).

### Deferred Ideas (OUT OF SCOPE)
- **Persisted reverse-edge index** — dispatch traversal deliberately uses Phase-3 in-memory query-time reverse adjacency; a persisted reverse index stays Phase 8.
- **wazero WASM parser backend** — remains a monitored future option behind the `parser.Parser` seam if CGo grammar crash-isolation proves painful in practice; not opened in this phase.
- **Non-call reference edges** (constants, interface-type usage as references) — TS surfaces these; our `RefKindCalls`-only `callees` scoping deliberately excludes them. Architectural, not a bug; revisit only if a future requirement demands it.

None of the above are re-opened here.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| LANG-02 | Java — full extraction + resolution | Grammar pin (`tree-sitter-java` v0.23.5, no external scanner); Architecture Pattern 2 (extends→implements promotion); Pitfall 3 (inheritance-chain conformance retry); Wave B plan-structure recommendation; Validation Architecture test map |
| LANG-03 | C# — full extraction + resolution | Grammar pin (`tree-sitter-c-sharp` v0.23.5, has external scanner); Architecture Pattern 2; Pitfall 5 (partial-class node-identity open question, flagged for discuss-phase); Wave B |
| LANG-04 | Python — full extraction + resolution | Grammar already pinned (Phase 1); Don't Hand-Roll (module-path resolution semantics, `A1`); Pitfall 2 (module-key generalization); Wave B |
| LANG-05 | TypeScript/JavaScript — full extraction + resolution | Grammar pins (`tree-sitter-javascript` v0.25.0, `tree-sitter-typescript` v0.23.2 — two grammars, two constructors); Don't Hand-Roll (tsconfig-aware resolution, `A1`); Wave B |
| LANG-06 | Mainstream tier — Rust, Ruby, PHP, C/C++, Swift, Kotlin at full/documented-partial | Grammar pins for all six (Package Legitimacy Audit flags Swift/Kotlin `[SUS]`); Wave E plan-structure recommendation; D-11 capability matrix requirement carried into Validation Architecture |
| LANG-07 | Framework-aware routing (Gin, Spring, ASP.NET, Django/Flask/FastAPI, Express/NestJS) | Architecture Pattern 4 (AST-based route detection recommendation vs. parity target's regex approach); Code Examples (route node/edge shape); Wave D; Security Domain (ReDoS caveat if porting regexes verbatim) |
| RES-02 | Interface→implementation dispatch edges synthesized, traversed at query time | Architecture Pattern 3 (Go structural method-set match, directly verified against parity target's #584 changelog entry); Code Examples (bounded synthesis); Wave C; integration point `internal/query/traverse.go` |
| RES-03 | Every heuristic edge carries `provenance: heuristic` + source location | Confirmed zero schema changes needed (`schema.Edge` already has `Provenance`/`Line`/`Col`/`Metadata`); Code Examples; Validation Architecture (RES-03 test asserts embedded in RES-02's test suite) |
</phase_requirements>

## Summary

This phase generalizes a Go-only, hand-written extractor/discoverer/resolver into a multi-language pipeline while reusing every seam Phases 1-3 already built for exactly this purpose. The parser layer (`internal/parser.Parser` + `internal/parser/cgo`) is already backend-neutral and multi-constructor (`NewGoParser`, `NewPythonParser`) — adding a language is a `New<Lang>Parser` + one `go get`. The schema (`internal/schema/graph.pb.go`) already reserves `Provenance="heuristic"`, `Line`, `Col`, and an open `Metadata` bag on every `Edge` — RES-02/RES-03 need zero schema changes. The query engine's reverse-adjacency traversal (`internal/query/traverse.go`) already exists for dispatch synthesis to ride on.

The two places that are **not** yet ready, and that this phase's real engineering weight falls on, are: (1) `internal/indexer/extract.go`'s worker pool, which constructs **exactly one parser per worker for its entire lifetime** — a design that silently breaks the moment a single indexing run must parse files in more than one language, since a worker pulling files off the shared counter has no way to swap grammars; and (2) `internal/indexer/symbolindex.go`'s `byImportAndName` index, which is addressed by Go's own `importPath` concept — every other target language addresses cross-file symbols by a structurally different key (Java package, C# namespace, Python dotted module path, TS/JS resolved module specifier), so the "global symbol index" itself needs a per-language key-computation hook, not just a per-language resolver.

Direct inspection of the parity target's live source (`colbymchenry/codegraph`, TypeScript, the same repo pinned as this project's golden-fixture oracle) gives near-exact answers for every "how should we name this" question the locked decisions left open: `route` is a literal node kind, `implements` is a literal edge kind distinct from `extends`, route→handler dispatch is synthesized as a `calls` edge with `provenance:'heuristic'` and `metadata.synthesizedBy`, and Go's implicit-interface `implements` edges are synthesized structurally (method-set coverage) and then made query-time-reachable exactly the way D-06 already specifies. This phase should follow that vocabulary, since it is both the parity target and a working precedent for exactly this problem.

**Primary recommendation:** Fix the worker-pool per-language-parser gap and generalize `symbolindex.go`'s key scheme FIRST (Wave A), before touching any new language — every other wave depends on both. Use `calls` (not a new `handles` kind) for route→handler dispatch edges so they fall inside the EXISTING `RefKindCalls`-only reverse-adjacency filter with zero traversal-code changes; use a genuinely new `implements` edge kind for RES-02 since that traversal is semantically different (name-joined, not identity-followed) and needs new code in `query.Callers`/`query.Impact` regardless.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Grammar parsing (per language) | Indexing Core (`internal/parser/cgo`) | — | CGo tree-sitter backend, one constructor per language, already the ratified pattern |
| Per-language symbol extraction | Indexing Core (`internal/indexer/<lang>extract`) | — | New per-language packages behind a language-ID registry (D-01) |
| File discovery / language detection | Indexing Core (`internal/indexer/discover.go`) | — | Extension→language registry + generic walker + project-descriptor hooks (D-03) |
| Cross-file symbol resolution | Indexing Core (`internal/indexer/resolve.go`, `symbolindex.go`) | — | Per-language resolver behind shared seam (D-04); this is where WR-01/WR-02 live |
| Dispatch synthesis (implements) | Indexing Core (resolve-time, Pass 2) | Query Engine (query-time traversal) | Edge synthesis at index time, traversal at query time (D-06) — split responsibility is deliberate, not accidental |
| Framework route detection | Indexing Core (extraction or a dedicated resolve-time pass) | — | Per-framework detector registry (D-08), opt-in per detected dependency (D-09) |
| Dispatch/route traversal | Query Engine (`internal/query/traverse.go`) | MCP Server (`codegraph_explore`, `callers`, `impact`) | Reuses Phase-3 in-memory reverse adjacency; MCP surfaces it unchanged |
| Provenance/heuristic tagging | Storage/Schema (`schema.Edge`) | — | Already-reserved fields, no schema change (D-07) |

## Package Legitimacy Audit

This phase's only external dependencies are Go modules — tree-sitter grammar bindings. The `gsd-tools query package-legitimacy check` seam covers npm/PyPI/crates, not Go modules; Go-ecosystem verification was instead done directly against the authoritative Go module proxy (`proxy.golang.org`) via `go list -m -versions` and raw `.mod` fetches — an equivalent authoritative-source check, tagged `[VERIFIED: proxy.golang.org]` below.

| Package | Registry | Maintainer | External C scanner | Verdict | Disposition |
|---|---|---|---|---|---|
| `github.com/tree-sitter/tree-sitter-java` v0.23.5 | Go proxy | `tree-sitter` org | No | OK | Approved |
| `github.com/tree-sitter/tree-sitter-c-sharp` v0.23.5 | Go proxy | `tree-sitter` org | **Yes** | OK | Approved — crash-isolation caveat applies |
| `github.com/tree-sitter/tree-sitter-python` v0.25.0 | Go proxy | `tree-sitter` org | **Yes** (already pinned Phase 1) | OK | Already in `go.mod` |
| `github.com/tree-sitter/tree-sitter-javascript` v0.25.0 | Go proxy | `tree-sitter` org | **Yes** | OK | Approved |
| `github.com/tree-sitter/tree-sitter-typescript` v0.23.2 | Go proxy | `tree-sitter` org | **Yes** (both `typescript/` and `tsx/` grammars) | OK | Approved — two grammars, two constructors |
| `github.com/tree-sitter/tree-sitter-rust` v0.24.2 | Go proxy | `tree-sitter` org | **Yes** | OK | Approved |
| `github.com/tree-sitter/tree-sitter-ruby` v0.23.1 | Go proxy | `tree-sitter` org | **Yes** | OK | Approved |
| `github.com/tree-sitter/tree-sitter-php` v0.24.2 | Go proxy | `tree-sitter` org | **Yes** (`php/src`, not `php_only/src`) | OK | Approved |
| `github.com/tree-sitter/tree-sitter-c` v0.24.2 | Go proxy | `tree-sitter` org | No | OK | Approved |
| `github.com/tree-sitter/tree-sitter-cpp` v0.23.4 | Go proxy | `tree-sitter` org | **Yes** | OK | Approved |
| `github.com/alex-pinkus/tree-sitter-swift` (pseudo-version, e.g. `v0.0.0-20260704222518-28fe3a8a8558`) | Go proxy | Individual maintainer, **not** `tree-sitter` org | **Yes** | SUS | Flagged — no semver tags (pseudo-version pin only), single-maintainer repo; planner should add `checkpoint:human-verify` before pinning an exact commit |
| `github.com/fwcd/tree-sitter-kotlin/bindings/go` (pseudo-version, e.g. `v0.0.0-20260207053055-6b9788578ae2`) | Go proxy | Individual maintainer, **not** `tree-sitter` org | **Yes** | SUS | Flagged — same reasons as Swift; root repo does carry a `v0.3.2` semver tag but its `go.mod` has no `require` line, so the real importable package is the unversioned `bindings/go` subpath |

**Packages removed due to `[SLOP]` verdict:** none.
**Packages flagged as suspicious `[SUS]`:** `alex-pinkus/tree-sitter-swift`, `fwcd/tree-sitter-kotlin` — both are community (non-`tree-sitter`-org) grammars with no proper semver Go releases, pinned by commit pseudo-version. This is a real, if modest, supply-chain quality step down from the `tree-sitter`-org grammars, consistent with these two languages sitting in the "documented-partial acceptable" mainstream tier (D-11) rather than the priority-4 full-parity tier. Planner must add `checkpoint:human-verify` before `go get`-ing either, and pin the exact commit hash (not `@latest`) in `go.mod` per DIST-05's audited-dependency discipline.

All grammar modules independently `require github.com/tree-sitter/go-tree-sitter v0.24.0`–`v0.25.0` in their own `go.mod` (verified via proxy fetch, not assumed) — Go's MVS selects the already-pinned `v0.25.0` project-wide, so no version conflict exists across any of the 10 new grammars `[VERIFIED: proxy.golang.org]`.

**Installation (priority-4):**
```bash
go get github.com/tree-sitter/tree-sitter-java@v0.23.5
go get github.com/tree-sitter/tree-sitter-c-sharp@v0.23.5
go get github.com/tree-sitter/tree-sitter-javascript@v0.25.0
go get github.com/tree-sitter/tree-sitter-typescript@v0.23.2
# tree-sitter-python@v0.25.0 already pinned (Phase 1)
```

**Installation (mainstream-6):**
```bash
go get github.com/tree-sitter/tree-sitter-rust@v0.24.2
go get github.com/tree-sitter/tree-sitter-ruby@v0.23.1
go get github.com/tree-sitter/tree-sitter-php@v0.24.2
go get github.com/tree-sitter/tree-sitter-c@v0.24.2
go get github.com/tree-sitter/tree-sitter-cpp@v0.23.4
go get github.com/alex-pinkus/tree-sitter-swift@<pinned-commit>   # checkpoint:human-verify first
go get github.com/fwcd/tree-sitter-kotlin/bindings/go@<pinned-commit>  # checkpoint:human-verify first
```

Per D-02a additive-only discipline, none of these require a `go.mod` `go` directive bump beyond what each grammar's own `go.mod` states (`go 1.22`/`go 1.23`) — well below this project's pinned toolchain.

## Standard Stack

### Core (parser layer — extends the existing `internal/parser/cgo` pattern)

| Grammar module | Version | Language(s) | External scanner (crash-isolation caveat) | Why standard |
|---|---|---|---|---|
| `tree-sitter/tree-sitter-java` | v0.23.5 | Java | No | Official `tree-sitter` org module `[VERIFIED: proxy.golang.org]` |
| `tree-sitter/tree-sitter-c-sharp` | v0.23.5 | C# | Yes | Official `tree-sitter` org module `[VERIFIED: proxy.golang.org]` |
| `tree-sitter/tree-sitter-python` | v0.25.0 | Python | Yes (INDENT/DEDENT) | Already pinned (Phase 1 spike partner) |
| `tree-sitter/tree-sitter-javascript` | v0.25.0 | JavaScript | Yes | Official `tree-sitter` org module |
| `tree-sitter/tree-sitter-typescript` | v0.23.2 | TypeScript + TSX (two grammars, one repo, `typescript/` and `tsx/` subdirs — two `New<Lang>Parser` constructors needed) | Yes (both) | Official `tree-sitter` org module |

### Supporting (mainstream-6, LANG-06)

| Grammar module | Version | Language | External scanner | When to use |
|---|---|---|---|---|
| `tree-sitter/tree-sitter-rust` | v0.24.2 | Rust | Yes | Extraction + best-effort resolution only (D-04) |
| `tree-sitter/tree-sitter-ruby` | v0.23.1 | Ruby | Yes | Same |
| `tree-sitter/tree-sitter-php` | v0.24.2 | PHP (`php/src`, not `php_only`) | Yes | Same |
| `tree-sitter/tree-sitter-c` | v0.24.2 | C | No | Same |
| `tree-sitter/tree-sitter-cpp` | v0.23.4 | C++ | Yes | Same |
| `alex-pinkus/tree-sitter-swift` | pseudo-version (pin exact commit) | Swift | Yes | `[SUS]` — `checkpoint:human-verify` first |
| `fwcd/tree-sitter-kotlin/bindings/go` | pseudo-version (pin exact commit) | Kotlin | Yes | `[SUS]` — `checkpoint:human-verify` first |

### Alternatives Considered

| Instead of | Could use | Tradeoff |
|---|---|---|
| Two `tree-sitter-typescript` constructors (typescript + tsx) | Skip TSX, extract `.tsx` as plain TypeScript | Loses JSX-specific node types (component detection for LANG-07's Express/NestJS-adjacent React work is out of scope anyway per PROJECT.md — but any `.tsx` file's JSX elements would parse as errors under the plain TS grammar). Not recommended; the two-constructor cost is small. |
| Official `tree-sitter/tree-sitter-c-sharp` | Community C# grammars | None found with better standing; the official module is the only real option and is `OK`-verdict. |
| Pinning Swift/Kotlin grammars by commit | Skip Swift/Kotlin from mainstream-6, document as `none` coverage | D-11 requires the matrix to name every mainstream language; "documented-partial" is the ratified floor, not zero coverage. `checkpoint:human-verify` is the correct mitigation, not exclusion. |

## Architecture Patterns

### System Architecture Diagram

```
                         ┌─────────────────────────────┐
                         │  internal/indexer/discover.go │
   repo files ──────────▶│  ext→language registry        │
                         │  + generic recursive walker    │
                         │  (reuses ShouldSkipDir)         │
                         │  + per-lang project-descriptor  │
                         │    hook (go.mod, pom.xml, ...)  │
                         └───────────────┬─────────────────┘
                                         │ []DiscoveredFile{..., Language}
                                         ▼
                         ┌─────────────────────────────────┐
                         │  internal/indexer/extract.go      │
                         │  worker pool — MUST select parser  │
                         │  + extractor PER FILE'S LANGUAGE   │
                         │  (not per-worker-lifetime anymore) │
                         └───────┬─────────────────┬──────────┘
                                 │                  │
                    per language │                  │ per language
                                 ▼                  ▼
                  parser.Parser (cgo.New<Lang>Parser)   LanguageExtractor registry
                                 │                  │  (goextract, javaextract, ...)
                                 └────────┬─────────┘
                                          ▼
                              []goextract.FileResult
                              (shared FileResult/Node/Edge
                               vocabulary, extended additively)
                                          │
                                          ▼
                    ┌───────────────────────────────────────────┐
                    │ internal/indexer/resolve.go + symbolindex.go │
                    │  per-language resolver behind shared seam    │
                    │  - global symbol index keyed by a PER-LANG    │
                    │    module/namespace/package identity, not     │
                    │    Go's importPath concept                    │
                    │  - Pass 2: calls/imports/embeds/contains       │
                    │  - NEW: implements synthesis (structural for   │
                    │    Go, declared-promotion for Java/C#/TS)      │
                    │  - NEW: route detector registry (opt-in per    │
                    │    detected framework dependency)               │
                    └───────────────────┬───────────────────────────┘
                                        │ nodes/edges (Provenance=ast|heuristic)
                                        ▼
                              GraphStore.Writer (unchanged, Phase 1)
                                        │
                                        ▼
                    ┌───────────────────────────────────────────┐
                    │ internal/query/traverse.go                  │
                    │  BuildReverseAdjacency (RefKindCalls only)   │
                    │  Callers/Callees/Impact                      │
                    │  NEW: implements-edge traversal step for      │
                    │  dispatch (interface method → concrete impls) │
                    └───────────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/indexer/
├── discover.go              # generalized: ext→language registry + generic walker
├── extract.go                # generalized: per-file parser+extractor selection
├── resolve.go                # generalized: per-language resolver dispatch
├── symbolindex.go            # generalized: per-language module-key computation
├── languages.go              # NEW: language registry (extension, descriptor hook,
│                              #      extractor ctor, resolver ctor, parser ctor)
├── goextract/                 # unchanged reference implementation + vocabulary
│   ├── goextract.go
│   └── types.go               # extended additively: RouteKind, ImplementsRefKind, etc.
├── javaextract/                # NEW, mirrors goextract's shape
├── csharpextract/               # NEW
├── pyextract/                    # NEW
├── tsextract/                     # NEW (shared TS/JS/TSX package; JSX handled via
│                                  #      the tsx grammar constructor, same extractor)
├── mainstream/                     # NEW, one subpackage per mainstream-6 language
│   ├── rustextract/
│   ├── rubyextract/
│   ├── phpextract/
│   ├── cextract/                    # shared C/C++ extractor (two grammars, one package)
│   ├── swiftextract/
│   └── kotlinextract/
├── dispatch/                          # NEW: implements-edge synthesis (RES-02/RES-03)
│   └── implements.go
└── routes/                             # NEW: framework route-detector registry (LANG-07)
    ├── registry.go
    ├── gin.go
    ├── spring.go
    ├── aspnet.go
    ├── django.go       # + flask.go, fastapi.go, or one file covering all 3
    └── express.go       # + nestjs.go, or combined
```

### Pattern 1: Generic tree-walker + per-language declarative config (implementation lever, not required)

**What:** The parity target (`colbymchenry/codegraph`) does NOT implement N fully-independent imperative extractors. It implements ONE large generic tree-walker (`src/extraction/tree-sitter.ts`, ~6600 lines) driven by small per-language declarative config objects (`src/extraction/languages/go.ts`, `java.ts`, etc. — each maps tree-sitter node-type strings to codegraph node kinds/fields) plus a handful of per-language special-case functions (e.g. `extractGoInterfaceMethods`).

**When to use:** D-01 requires per-language extractor *packages* behind a *registry* — it does not mandate that each package be a fully independent imperative walker. A shared generic walker parameterized by a per-language config table is a legitimate way to satisfy D-01 while cutting the ~10x boilerplate duplication that 10+ fully-independent `goextract.go`-style packages would otherwise incur — and it is NOT the "pure-.scm generic engine" D-02 forbids (D-02 is about tags.scm-as-the-architecture, not about a shared imperative Go walker). Recommend the planner evaluate this tradeoff explicitly in Wave A rather than defaulting to N independent copies of `goextract.go`'s ~650 lines.

**Example (TS CodeGraph's shape, for reference — not literal code to port):**
```typescript
// Source: colbymchenry/codegraph src/extraction/languages/go.ts (verified via GitHub)
// A per-language config object the shared walker consults for node-type mapping
export const goExtractor: LanguageExtractorConfig = {
  functionTypes: ['function_declaration'],
  methodTypes: ['method_declaration'],
  interfaceTypes: ['type_declaration'],  // + a type-guard on the inner type
  // ...
};
```

### Pattern 2: Extends→implements promotion at resolve time (Java/C#/TS declared implements)

**What:** At extraction time, do NOT try to distinguish "class extends class" from "class implements interface" — both are syntactically an unresolved supertype reference. Emit a single unresolved `extends`-shaped reference for both. At resolve time, once the target node's `Kind` is known, promote the edge to `implements` if-and-only-if the target is an `interface` node and the source is not itself an `interface`.

**When to use:** Java (`implements`/`extends` both appear in `super_interfaces`/`superclass` grammar nodes and are easy to conflate at parse time without a symbol table), C# (`: IFoo, BaseClass` is a single comma-separated list with no syntactic marker distinguishing base class from interface), TypeScript (`class X extends Y implements Z`). This exact pattern is what the parity target does — verified in its live source.

**Example:**
```typescript
// Source: colbymchenry/codegraph src/resolution/index.ts (verified via GitHub, ~line 957)
// Promote "extends" to "implements" when a class/struct targets an interface
if (kind === 'extends') {
  const targetNode = this.queries.getNodeById(ref.targetNodeId);
  if (targetNode && (targetNode.kind === 'interface' || targetNode.kind === 'protocol')) {
    const sourceNode = this.queries.getNodeById(ref.original.fromNodeId);
    if (sourceNode && sourceNode.kind !== 'interface' && sourceNode.kind !== 'protocol') {
      kind = 'implements';
    }
  }
}
```

### Pattern 3: Structural (implicit) interface satisfaction for Go — extract interface method specs as nodes, match method sets at resolve time

**What:** Go has no `implements` keyword. The parity target extracts each interface's method specs (`method_elem`/`method_spec` tree-sitter nodes) as first-class `method` nodes *contained by the interface node* (not just the concrete struct's methods) — verified in `src/extraction/tree-sitter.ts` around `extractGoInterfaceMethods`. A whole-graph resolve-time pass then compares each struct's declared method names (bounded by name+arity, per D-06's quadratic-blowup guard) against each interface's method-spec set; a struct whose methods are a superset of an interface's method-spec names is synthesized as `implements` that interface.

**When to use:** Go only, among the priority-4/mainstream-10 set. This is exactly D-06's Go branch.

**Verified precedent (parity target's own changelog, `[VERIFIED: github.com/colbymchenry/codegraph CHANGELOG.md]`):**
> "Go interfaces now connect to their implementations. Go has no `implements` keyword — a type satisfies an interface just by having the right methods — so CodeGraph now infers that link: a struct whose methods cover an interface's method set is treated as implementing it, and a call through the interface (`API.Marshal(...)`) reaches every concrete implementation." (#584)

### Pattern 4: Route detection as regex-over-comment-stripped-source, not full AST query

**What:** The parity target's framework route extractors (`src/resolution/frameworks/go.ts`, `python.ts`, etc.) do NOT walk the tree-sitter AST for route detection — they run targeted regexes over comment-stripped source text (`stripCommentsForRegex`) looking for framework-specific call shapes (`\w+\.(GET|POST|...)\("...")`for Gin/Echo/Chi, `path\(...\)`/`re_path\(...\)` for Django). This is deliberately lighter-weight than full-grammar extraction.

**When to use:** This project already parses every file into a full tree-sitter AST for symbol extraction (unlike the parity target's per-framework-only regex pass) — so route detection here has a *more* precise option available (walk `call_expression` nodes already visited during `collectCalls`-equivalent traversal, filter by the same verb/string-literal shape) at similar implementation cost to regex, with fewer false positives (a regex can match inside a string literal or a shadowed identifier; an AST-node-kind check cannot). **Recommend using the already-parsed AST, not a second regex pass over raw source**, while keeping the exact same detection *signatures* (HTTP verb + string path literal + handler argument) the parity target validated. This is a deliberate improvement over the parity target's implementation, not a deviation from its behavior.

**Example (Go/Gin route pattern the AST-based detector should recognize — same signature as the parity target's regex, applied via AST instead):**
```typescript
// Source: colbymchenry/codegraph src/resolution/frameworks/go.ts (verified via GitHub)
// <anyVar>.METHOD("/path", handler) — Gin (GET/POST/...), Chi (Get/Post/...), net/http
const routeRegex = /\b\w+\.(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD|Get|Post|Put|Patch|Delete|Handle|HandleFunc)\s*\(\s*"([^"]+)"\s*,\s*([^)]+)\)/g;
```
The receiver is deliberately **any identifier**, not a fixed name list (`router`/`r`/`app`) — real Gin apps route on group variables (`v1.GET`, `userRouter.POST`). Verb + string-literal path + handler-argument is the precision gate, not the receiver name.

### Anti-Patterns to Avoid
- **One parser per worker for the whole worker lifetime, unchanged for multi-language:** `internal/indexer/extract.go`'s current design constructs exactly one `parser.Parser` per worker goroutine and reuses it for every file that worker's shared-counter loop claims. This is a **hard architectural bug** the moment files of more than one language appear in a single indexing run (Java+Kotlin, C+C++, or simply any repo containing both `.go` and `.md`/`.json` if those were ever added). Each worker must instead own a small per-language parser cache (map[language]parser.Parser, lazily constructed, closed at worker exit) OR files must be partitioned by language before worker-pool dispatch. This is the single most important architectural finding in this research — see Common Pitfalls #1.
- **A single `byImportAndName`-shaped symbol index for all languages:** Go's import-path addressing does not generalize. Do not bolt Java package names or Python module paths into the same string-keyed map without first defining what the per-language "module key" actually means for that language (see Common Pitfalls #2).
- **Regex-only route/framework detection when a full AST is already available:** the parity target's regex approach is a reasonable choice for a language it does NOT already fully parse into an AST; this project always has a full AST by the time resolve-time framework detection would run, so prefer AST-node matching (Pattern 4).
- **Materializing a cartesian call→impl edge for every dispatch site:** explicitly rejected by D-06; synthesize `implements` once per (concrete-type, interface) pair, and traverse it at query time.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---|---|---|---|
| Tree-sitter grammar for a language | A hand-rolled recursive-descent parser | The pinned `tree-sitter/tree-sitter-<lang>` (or vetted community) grammar module | Grammar correctness for 10+ languages is a multi-year community effort; hand-rolling is exactly the "huge effort, immature" risk `PARSER-DECISION.md` Option C already rejected for v1 |
| Java/C#/TS import resolution | Bespoke string-splitting heuristics with no grounding in the language's actual module system | Java: package-name + classpath-relative resolution; C#: namespace matching (no file-path coupling); Python: `sys.path`/package-root-relative dotted paths; TS/JS: `tsconfig.json` `paths`/`baseUrl` + relative-specifier + `package.json` main/exports resolution order | Each ecosystem's own resolution algorithm is well-specified and has known edge cases (Python relative-import dots, TS path-mapping, C# no-file-path-coupling) that a naive string-match heuristic reliably gets wrong on real repos |
| Interface/dispatch synthesis | A fresh design for "how do I know type X implements interface Y" | Pattern 2 (declared-implements promotion) + Pattern 3 (Go structural method-set match) — both directly verified against the parity target's live, ship-validated implementation | The parity target already solved this exact problem for the exact same requirement (RES-02) at production scale; re-deriving it from scratch risks re-discovering the same edge cases (quadratic blowup on wide interfaces, ambiguous multi-candidate matches) the hard way |
| Framework route detection | A fully general "detect any web framework" engine | A per-framework detector registry, opt-in per detected manifest/import signature (D-08/D-09) | TS CodeGraph ships 25+ per-framework detectors, not one general engine — the problem does not generalize well and the registry pattern is the proven shape |

**Key insight:** for every open naming/shape question this phase's locked decisions deliberately left to "Claude's Discretion" (route/implements/handles edge-kind strings, node-kind names, metadata keys), the parity target's live source is a directly authoritative answer, not just a design inspiration — since D-12 measures parity against exactly this project's output shape.

## Runtime State Inventory

Not applicable — this is a greenfield extension of the extraction/resolution pipeline, not a rename/refactor/migration phase. No existing runtime state (databases, live service config, OS-registered state, secrets, build artifacts) is renamed or restructured by this phase; new node/edge kinds and new Go packages are additive.

## Common Pitfalls

### Pitfall 1: The extract.go worker pool's "one parser per worker" design does not survive multi-language files
**What goes wrong:** `internal/indexer/extract.go`'s `extractWithFactory` constructs exactly one `parser.Parser` via `newParser()` per worker goroutine, then that SAME parser instance is reused for every file the worker pulls off the shared atomic counter for its entire lifetime. This is correct today because every file is Go. The instant `DiscoveredFile` carries a `Language` field (required by D-01/D-03) and a single indexing run can contain files of more than one language (which is the entire point of this phase, and true even within priority-4 alone — e.g. a Spring Boot repo has `.java` AND `.yml`/`.properties` config, a TS/JS repo has `.ts`/`.tsx`/`.js`/`.jsx`), a worker that grabs a Java file then a TypeScript file has no mechanism to swap grammars.
**Why it happens:** the original design (Phase 2) never needed to consider more than one language, so "one parser per worker, for the worker's whole life" and "one parser per worker, for the CURRENT file's language" were indistinguishable requirements.
**How to avoid:** give each worker a small `map[string]parser.Parser` cache keyed by language, lazily constructing (and eventually `Close()`-ing, at worker exit) a parser the first time that worker encounters that language — OR partition `files` by language before dispatching worker pools (simpler, but loses cross-language work-stealing balance within one `Extract()` call; likely fine given per-language batches are usually large). Either fix must preserve the existing "N parsers total, not N-per-file" bound (RESEARCH note this project's own Phase 2 RESEARCH already called out as a determinism/cost pitfall) — a per-file `New<Lang>Parser()` call would reintroduce exactly the cost problem Phase 2 avoided.
**Warning signs:** a multi-language golden-parity fixture repo (e.g. a Spring Boot service with YAML config, or the `colbymchenry/codegraph` corpus itself — TS/JS/Python/Astro/YAML) produces wrong or missing extraction for every file whose language differs from whatever the worker's first-claimed file was.

### Pitfall 2: `symbolindex.go`'s import-path-keyed global index does not generalize across languages
**What goes wrong:** `byImportAndName map[string]map[string]string` is keyed by Go's `importPath` — a concept computed from `go.mod` + directory structure. Java addresses cross-file symbols by package name (declared in-source, `package com.foo.bar;`, independent of directory layout by convention but not by grammar rule). C# addresses by namespace (also declared in-source, `namespace Foo.Bar;`, with zero required relationship to file path or project structure). Python addresses by dotted module path (directory-structure-derived, but rooted at whatever `sys.path` entry applies — ambiguous without a project descriptor). TypeScript/JavaScript addresses by resolved module specifier (relative paths, `tsconfig.json` `paths`/`baseUrl` remapping, or `node_modules` package resolution) — the LEAST directory-structure-stable of the four. Naively keying all five languages' symbols into one `importPath`-shaped map either collides unrelated symbols or silently fails to resolve real cross-file references.
**Why it happens:** the abstraction that worked for one language (Go's import path is uniquely both the resolution key AND a directory-derivable value) accidentally conflated "the resolution key" with "a directory-derivable string," which is only true for Go among these five languages.
**How to avoid:** define a per-language `ModuleKey(file DiscoveredFile) string` function as part of each language's project-descriptor hook (D-03) — Go's `importPathFor` becomes the FIRST implementation of this hook, not a special case. `symbolindex.go`'s core structure (`map[moduleKey]map[symbolName]nodeID`) can stay the same; only the computation of `moduleKey` needs to become a per-language plugin point.
**Warning signs:** cross-file calls resolve correctly within a single Java/C#/Python/TS file's own package/namespace/module but silently fail to resolve calls into a DIFFERENT file that logically shares the same package/namespace/module (the exact "same-package name collision" failure class WR-01 already describes for Go — this generalizes to every new language unless the key-computation is done correctly per language).

### Pitfall 3: Package-qualified vs. bare-name call resolution ambiguity multiplies with declared-import languages
**What goes wrong:** Go's `resolveSelector` only fires for an operand that is a real key in the file's own `Imports` map (RQ-2's narrowest-safe-set boundary) — this correctly excludes local-variable-receiver calls (`w.Describe()`). Java/C#/TS have a structurally different problem: a bare method call inside a class body (`describe()`) implicitly means `this.describe()`, and an inherited method call has NO local declaration at all — it must walk the `extends`/`implements` chain (once those edges exist) to find where the method is actually declared. This is exactly the "conformance pass" fallback pattern the parity target implements (deferred-ref retry after `implements`/`extends` edges exist) — WITHOUT it, priority-4 languages will show a much higher `unresolvedCount` than Go ever did, purely from inherited-method calls.
**Why it happens:** Go has no class inheritance to walk; Java/C#/TS/Python (via MRO) all do.
**How to avoid:** resolve calls/implements edges in TWO passes within Pass 2 itself: (1) resolve everything resolvable without inheritance info, synthesize `extends`/`implements` edges; (2) retry any call that failed pass 1 AND originates from a class/struct with an `extends`/`implements` edge, this time walking the supertype chain for the method. This two-pass-within-Pass-2 shape is directly analogous to what the parity target calls `deferredChainRefs` / the "conformance pass" (`[VERIFIED: colbymchenry/codegraph src/resolution/index.ts]`).
**Warning signs:** a Java/C# golden-parity fixture shows correct extraction (right node counts) but abnormally poor `callers`/`impact` results specifically for methods only ever invoked via inheritance (an overridden method called through a base-class reference).

### Pitfall 4: External-scanner grammars run in-process with no crash isolation — this now applies to 8 of 10 new languages, not just Python
**What goes wrong:** `internal/parser/parser.go`'s documented crash-isolation contract (a C-level segfault in an external scanner is NOT `recover()`-able) was written and accepted for ONE grammar (Python's INDENT/DEDENT scanner) in Phase 1. This phase adds C#, JavaScript, TypeScript/TSX, Rust, Ruby, PHP, C++, Swift, and Kotlin — 9 of the 10 new grammars carry an external `scanner.c` (verified directly against each grammar's GitHub `src/` directory listing; only Java and C do not). The accepted-risk surface is now nearly the entire language matrix, not a single spike-partner language.
**Why it happens:** external scanners exist precisely for constructs a context-free grammar can't express cleanly — heredocs (Ruby), string interpolation (Swift), template literals/regex (JS/TS), raw strings (Rust/C++), tag-switching (PHP `<?php ?>`), significant whitespace (Python) — which is most "interesting" modern language syntax.
**How to avoid:** no code fix exists within this phase's scope (subprocess isolation is explicitly deferred, per `parser.go`'s own doc comment, to "Phase 2+" if the risk proves unacceptable — and D-07/wazero WASM stays a monitored future option, not opened here). The mitigation IS already in place: `parser.MaxSourceBytes` (4 MiB) bounds the pathological-input surface before any backend-specific parsing runs, for every language uniformly. Document this explicitly in the language capability matrix (D-11) as a shared caveat, not a per-language footnote.
**Warning signs:** a fuzzed or adversarial-input test causes the whole `codegraph` process to crash rather than return a per-file extraction error — expected/accepted behavior per the existing contract, but must not be mistaken for a NEW bug introduced by this phase.

### Pitfall 5: Java/C# multi-file compilation units and partial classes break the "one node ID scheme" assumption subtly
**What goes wrong:** Go's `nodeid.NodeID(kind, qualifiedName, filePath)` scheme assumes a symbol has exactly one declaring file. C#'s `partial class` keyword allows ONE logical class to be declared across multiple files (extremely common in ASP.NET scaffolded/generated code, e.g. `Controller.cs` + `Controller.Designer.cs`). A naive port of Go's node-id scheme would either collide two partial-class fragments into one overwriting node (losing one file's methods) or create two separate nodes for what is logically one type (breaking `contains`/`implements` edge attachment).
**Why it happens:** the node-id formula's `filePath` component was designed for a language where "one type, one file" is nearly universal (Go allows multiple files per package but not multiple files per TYPE).
**How to avoid:** either (a) document `partial class` as an explicit, named gap in the C# capability matrix entry (methods from each partial fragment become separate FILE-scoped nodes containing that fragment's methods, with no single "the class" node — acceptable for priority-4 "full" if documented, or treat as the mainstream-tier bar), or (b) if full parity requires it, key the type-declaration node by `(qualifiedName, namespace)` only (no filePath) and treat each partial fragment's methods as `contains`-edged into that shared node, with the node's own `FilePath`/`StartLine`/`EndLine` pointing at the FIRST fragment encountered (deterministic tie-break by file path sort order, mirroring D-05's edge-collapse tie-break pattern). Flag this as an explicit open question for the planner/discuss-phase to resolve — do not silently pick (a) or (b) without documenting the choice.
**Warning signs:** a real ASP.NET repo (a likely C# validation corpus candidate) with generated `.Designer.cs` partial-class scaffolding shows a class with fewer methods than it actually has, or two colliding same-ID nodes silently overwriting each other in the store.

## Code Examples

### Language registry shape (D-01, extends the existing per-constructor parser pattern)
```go
// internal/indexer/languages.go (new file, illustrative shape)
type LanguageSpec struct {
	ID         string   // "go", "java", "csharp", "python", "typescript", ...
	Extensions []string // [".go"], [".java"], [".cs"], [".py"], [".ts", ".tsx"], ...
	NewParser  func() (parser.Parser, error)
	Extract    func(p parser.Parser, moduleKey, relPath string, src []byte) (goextract.FileResult, error)
	// ModuleKey computes this language's cross-file symbol-index key for a
	// discovered file — Go's importPathFor becomes the first implementation.
	ModuleKey  func(descriptor ProjectDescriptor, relPath string) string
	// ProjectDescriptor, when non-nil, parses this language's manifest
	// (go.mod, pom.xml, *.csproj, package.json+tsconfig.json, pyproject.toml)
	// to resolve module/namespace identity for the whole repo.
	Descriptor func(root string) (ProjectDescriptor, error)
}

var registry = map[string]LanguageSpec{
	"go": {ID: "go", Extensions: []string{".go"}, NewParser: func() (parser.Parser, error) { return cgo.NewGoParser() }, /* ... */},
	// "java", "csharp", "python", "typescript" added by this phase
}
```

### Structural implements-edge synthesis (Go), bounded to avoid quadratic blowup (D-06)
```go
// internal/indexer/dispatch/implements.go (new file, illustrative shape)
// interfaceMethods: interfaceNodeID -> sorted set of method names the interface declares
// structMethods: structNodeID -> sorted set of method names the struct declares
// Bound: only compare a struct against interfaces whose method-name set is a
// SUBSET-CANDIDATE by a cheap pre-filter (e.g. shares at least one method name
// via an inverted index method-name -> []interfaceID), never O(structs × interfaces).
func synthesizeGoImplements(structMethods, interfaceMethods map[string]map[string]struct{}) []*schema.Edge {
	methodNameIndex := invertMethodIndex(interfaceMethods) // methodName -> []interfaceID
	var edges []*schema.Edge
	for structID, methods := range structMethods {
		candidates := candidateInterfaces(methods, methodNameIndex) // pre-filtered, not all interfaces
		for _, ifaceID := range candidates {
			if isSuperset(methods, interfaceMethods[ifaceID]) {
				edges = append(edges, &schema.Edge{
					Source: structID, Target: ifaceID, Kind: "implements",
					Provenance: "heuristic",
					Metadata: map[string]string{"synthesizedBy": "go-structural-methodset"},
				})
			}
		}
	}
	return edges
}
```

### Route node + heuristic dispatch edge (LANG-07), matching the parity target's vocabulary
```go
// Node: Kind = "route", Name = "GET /users/:id", QualifiedName = filePath + "::route:" + path
// Edge: route -> handler, Kind = "calls" (reuses the EXISTING RefKindCalls-filtered
// reverse-adjacency traversal with zero query-engine changes), Provenance = "heuristic",
// Metadata = {"synthesizedBy": "gin-route", "httpMethod": "GET", "routePath": "/users/:id"}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|---|---|---|---|
| `smacker/go-tree-sitter` community fork bundling grammars | `tree-sitter/go-tree-sitter` official bindings + per-language grammar modules | Aug 2024 (org takeover) | Already the ratified project decision (`PARSER-DECISION.md`); this phase's grammar pins all target the official pattern |
| Cartesian call→impl edge materialization for dynamic dispatch | Synthesize the relationship once (`implements`), traverse at query time | This is the current, deliberate v1 design (D-06) — not a change from an older approach within this project, but worth noting as the more scalable of the two options the parity target itself also converged on (structural `implements` edges + query-time-reachable, not per-call-site fanout) |

**Deprecated/outdated:** none directly applicable — this is new capability, not a migration off a deprecated approach.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Java package-name resolution, C# namespace resolution, Python dotted-module resolution, and TS/JS `tsconfig`-aware module resolution follow the semantics described in "Don't Hand-Roll" and Pitfall 2/3 | Don't Hand-Roll, Pitfalls 2-3 | If a language's real resolution algorithm has an edge case not captured here (e.g. Python's PEP 420 implicit namespace packages, C#'s `global using` C# 10+ feature, TS's `exports` map conditional resolution), the per-language resolver may under- or mis-resolve on real repos until the golden-parity harness (D-12) surfaces the gap |
| A2 | Framework `detect()` heuristics (manifest-string-contains + annotation/decorator scan) generalize cleanly from the parity target's Go/Java/C#/Python examples to Spring/ASP.NET/Django/Flask/FastAPI/Express/NestJS without further tuning | Architecture Pattern 4, Don't Hand-Roll | A false-negative detect() silently produces zero routes for a real framework user (not a crash, just missing coverage) — mitigated by D-12's validation requiring routes to show up on a real repo in each target framework |
| A3 | C#'s `partial class` handling (Pitfall 5) is a real, not hypothetical, parity gap for a representative ASP.NET validation corpus | Pitfall 5 | If the chosen C# validation repo doesn't happen to use partial classes/generated Designer files, this gap goes undetected until a real user's repo hits it post-ship — recommend the planner deliberately pick a corpus that DOES exercise `partial class` (common in older ASP.NET MVC scaffolding) to surface this during D-12 validation rather than after |
| A4 | The `extends`→`implements` promotion pattern (Pattern 2) and the two-pass "conformance retry" (Pitfall 3) are sufficient, without additional per-language special-casing, to reach "full resolution" for priority-4 | Architecture Pattern 2, Pitfall 3 | Under-resolution would show up as an inflated `unresolvedCount` in the golden-parity diff — self-detecting via D-12's own validation method, low risk of shipping silently broken |

**If this table is empty:** N/A — see rows above.

## Open Questions

1. **Worker-pool per-language-parser fix: cache-per-worker vs. partition-by-language?**
   - What we know: the current one-parser-per-worker-lifetime design (Pitfall 1) must change; both a per-worker language-keyed parser cache and a pre-partition-by-language approach are viable.
   - What's unclear: which has better real-world throughput on a genuinely mixed-language repo (e.g. a monorepo with Go services AND a TS/JS frontend) — the pre-partition approach loses cross-language load-balancing within one `Extract()` call if one language's file count vastly outnumbers another's.
   - Recommendation: Wave A should implement per-worker language-keyed caching (simpler to reason about, preserves the existing single-`Extract()`-call API and worker-count semantics) unless a quick benchmark on a real mixed-language corpus shows a meaningful throughput gap.

2. **C# `partial class` node-identity scheme (Pitfall 5)** — resolved above as an explicit planner decision point, not resolved here. Recommend surfacing this in `/gsd-discuss-phase` if not already implicitly covered by D-11's "documented-partial" allowance, since C# is priority-4 (full parity required, not documented-partial).

3. **Shared generic walker (Pattern 1) vs. N independent imperative extractors — a real implementation-cost decision, not just a style preference.**
   - What we know: the parity target uses a shared generic walker; this project's existing `goextract.go` is a fully independent imperative walker (~650 lines for ONE language).
   - What's unclear: whether investing in a shared generic walker + per-language config tables pays for itself within this phase's scope (10 new languages) versus the simpler-to-review, easier-to-parallelize-across-plan-waves cost of N independent packages.
   - Recommendation: given this phase's plan-structure likely parallelizes priority-4 languages across separate waves/plans (see Plan Structure Recommendation below), N independent packages following `goextract.go`'s exact shape is probably the lower-risk choice for THIS phase even if it produces more total code — a shared walker becomes more attractive as a Phase 5.x refactor once 10+ language packages exist and duplication cost is empirically visible. Do not block Wave A on designing a shared walker.

## Plan Structure Recommendation

This phase is large (8 requirements, 10+ languages, dispatch synthesis, framework routing) and should split into multiple plans across ordered waves — later waves depend on Wave A's generalized seams.

**Wave A — Foundation (blocks everything else):**
- Generalize `discover.go` (extension→language registry + generic walker + project-descriptor hook interface; go.mod becomes the first hook implementation)
- Generalize `extract.go` (fix Pitfall 1 — per-file parser+extractor selection)
- Generalize `resolve.go`/`symbolindex.go` (per-language `ModuleKey` hook; fix Pitfall 2)
- Fold in WR-01 (same-package name collision), WR-02 (selector-on-non-identifier), and the call-as-argument extraction gap (D-05) — these reshape `symbolindex.go`/`goextract.go` and every subsequent language inherits whatever shape they land in, so fixing them AFTER new languages exist means fixing N places instead of 1
- Add a `LanguageExtractor` interface + registry (D-01) with Go as its first (refactored, not rewritten) implementation, proving the seam before any new language uses it

**Wave B — Priority-4 languages (can parallelize across plans once Wave A lands; each is independently validated against D-12's golden-parity harness):**
- Java: parser constructor + `javaextract` package + resolver (package/import model) + golden-parity validation on a real Java repo
- C#: parser constructor + `csharpextract` package + resolver (namespace/using model, Pitfall 5 decision) + golden-parity validation
- Python: `pyextract` package + resolver (Python parser already exists from Phase 1; only the extractor/resolver are new) + golden-parity validation
- TypeScript/JavaScript: two parser constructors (TS + TSX) + `tsextract` package + resolver (tsconfig-aware module resolution) + golden-parity validation

**Wave C — Dispatch synthesis (RES-02/RES-03, depends on Wave A + at least Go being generalized; benefits from Wave B's Java/C# landing since declared-implements is easier to validate with real Java/C# fixtures):**
- Structural `implements` synthesis for Go (Pattern 3)
- Declared-implements promotion for Java/C#/TS (Pattern 2) + the two-pass conformance retry (Pitfall 3)
- Query-time dispatch traversal extension in `internal/query/traverse.go` (Callers/Impact walk `implements` edges)
- Provenance/metadata conformance (D-07 — verify no schema changes needed, just correct field population)

**Wave D — Framework routing (LANG-07, depends on Wave B for the languages each framework targets):**
- Route detector registry (D-08/D-09)
- Gin (Go), Spring (Java), ASP.NET (C#), Django/Flask/FastAPI (Python), Express/NestJS (TS/JS) — likely one plan per framework or grouped 2-3 per plan given the regex/AST-matching pattern is highly repetitive per Pattern 4

**Wave E — Mainstream tier (LANG-06, can start once Wave A lands; independent of Waves B-D):**
- Rust, Ruby, PHP, C/C++ extraction + best-effort resolution (no dedicated resolver depth required — extraction + same-file/same-module resolution only, per D-04's tiering)
- Swift, Kotlin extraction + best-effort resolution (with the `checkpoint:human-verify` gate on their `[SUS]`-flagged grammar pins)

**Wave F — Coverage documentation + validation closeout (depends on all prior waves):**
- Language capability matrix (D-11) — both human-readable doc and machine-readable descriptor
- Mainstream-tier self-consistency + spot-check validation (D-12)
- Final cross-language regression pass (ensure Wave A's generalization didn't regress Go's own golden-parity fixture)

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|---|---|---|---|---|
| Go toolchain | All waves | ✓ | project-pinned (1.24/1.25 line) | — |
| C toolchain (CGo) | Grammar compilation | ✓ (already required by Phase 1's CGo tree-sitter decision) | — | — |
| Real validation repos (Java, C#, Python, TS/JS, per-framework) | D-12 golden-parity + LANG-07 validation | Not yet selected | — | Researcher/planner selects representative, license-clean real repos per "Claude's Discretion" |
| Live TS CodeGraph v1.3.1 CLI | Re-running `capture.sh` to capture NEW golden fixtures for non-Go languages | Unknown — `testdata/golden/README.md` explicitly warns this is "a time-sensitive, one-shot capture" that may no longer be runnable | v1.3.1 (if still installed) | If unavailable, D-12's priority-4 validation must fall back to a lighter shape-consistency check against this project's own OWN prior runs (self-consistency), OR read the parity target's SOURCE (as this research did) as a specification rather than a live golden-output oracle. Flag this to the planner explicitly — it changes what "validated on real repos" (D-12) can mean in practice for anything beyond Go. |

**Missing dependencies with no fallback:** none blocking — the TS CLI availability question has a documented fallback above.
**Missing dependencies with fallback:** live TS CodeGraph CLI for capturing NEW per-language golden fixtures (fallback: source-as-specification + self-consistency, as this research itself demonstrates is viable).

## Validation Architecture

### Test Framework
| Property | Value |
|---|---|
| Framework | Go `testing` package, `go test ./...` |
| Config file | none — standard `go test` |
| Quick run command | `go test ./internal/indexer/... -count=1` |
| Full suite command | `go test ./... -count=1` (includes `testdata/golden` parity tests, which self-skip when their corpus isn't checked out) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|---|---|---|---|---|
| LANG-02 | Java full extraction + resolution | golden-parity (shape) | `go test ./testdata/golden/... -run TestGoldenParity_Java -count=1` | ❌ Wave B/F — new fixture + test needed, mirroring `golden_parity_test.go`'s weft-go pattern |
| LANG-03 | C# full extraction + resolution | golden-parity (shape) | `go test ./testdata/golden/... -run TestGoldenParity_CSharp -count=1` | ❌ Wave B/F |
| LANG-04 | Python full extraction + resolution | golden-parity (shape) | `go test ./testdata/golden/... -run TestGoldenParity_Python -count=1` | ❌ Wave B/F |
| LANG-05 | TS/JS full extraction + resolution | golden-parity (shape) | `go test ./testdata/golden/... -run TestGoldenParity_TSJS -count=1` | ❌ Wave B/F |
| LANG-06 | Mainstream-6 documented-partial | self-consistency + matrix | `go test ./internal/indexer/mainstream/... -count=1` | ❌ Wave E |
| LANG-07 | Framework route detection | fixture-based unit test per framework | `go test ./internal/indexer/routes/... -count=1` | ❌ Wave D |
| RES-02 | Implements/dispatch synthesis | unit test (bounded structural matcher) + fixture-based dispatch-traversal test | `go test ./internal/indexer/dispatch/... ./internal/query/... -count=1` | ❌ Wave C |
| RES-03 | Provenance tagging on every heuristic edge | assertion embedded in RES-02's tests (every synthesized edge has `Provenance=="heuristic"` + non-zero `Line`) | same as RES-02 | ❌ Wave C |

### Sampling Rate
- **Per task commit:** `go test ./internal/indexer/... -count=1` (fast subset covering the package under active work)
- **Per wave merge:** `go test ./... -count=1` (full suite, including golden-parity tests that self-skip absent their corpus)
- **Phase gate:** Full suite green before `/gsd-verify-work`, PLUS the D-11 capability matrix committed and cross-checked against every requirement's actual test coverage (a "full" entry in the matrix must have a corresponding green golden-parity test; a "partial" entry must name its specific gap)

### Wave 0 Gaps
- [ ] `testdata/golden/corpus/<lang>/` — new golden fixtures for Java, C#, Python, TS/JS validation repos, mirroring `weft-go`'s capture pattern (contingent on live TS CLI availability — see Environment Availability)
- [ ] `testdata/golden/golden_parity_test.go`-style per-language test files — mirror the existing `resolveWeftCorpus` pinned-commit-skip pattern for each new corpus
- [ ] `internal/indexer/<lang>extract/<lang>extract_test.go` — table-driven node-kind mapping tests per new language, mirroring `goextract_test.go`'s shape
- [ ] `internal/indexer/dispatch/implements_test.go` — bounded structural-matcher unit tests (including a "wide interface" stress case proving the quadratic-blowup guard actually bounds cost)
- [ ] `internal/indexer/routes/*_test.go` — per-framework fixture tests (a small real-shaped source snippet per framework, not a live repo)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---|---|---|
| V2 Authentication | No | This phase parses source code and synthesizes graph edges; no authentication surface |
| V3 Session Management | No | N/A |
| V4 Access Control | No | N/A |
| V5 Input Validation | Yes | `parser.MaxSourceBytes` (already enforced, Phase 1) bounds every new grammar's input the same way it bounds Go/Python today — no new validation code needed, but EVERY new `New<Lang>Parser` constructor must route through the SAME `Parse()` seam, never a bypass |
| V6 Cryptography | No | N/A |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---|---|---|
| Pathological/adversarial source input crashing an external-scanner grammar in-process (Pitfall 4) | Denial of Service | `parser.MaxSourceBytes` ceiling (existing, Phase 1) — accepted-risk-with-mitigation for the 9 of 10 new grammars carrying external scanners; not eliminated, bounded |
| A malicious `.csproj`/`pom.xml`/`package.json`/`requirements.txt` manifest string crafted to false-positive-trigger a framework detector (D-08/D-09's `detect()` reads these files) | Spoofing (of framework presence) | Low severity — a false-positive framework detection produces spurious `route` nodes, not a code-execution or data-exposure risk; no special mitigation needed beyond D-09's existing opt-in-per-signature design already limiting blast radius |
| Regex-based (or AST-based) route/framework detection running on attacker-controlled source (any indexed repo is, by this project's own threat model, arbitrary third-party code) causing catastrophic regex backtracking (ReDoS) if Pattern 4's route regexes are naively ported | Denial of Service | If porting the parity target's regexes verbatim (Pattern 4's example), audit each for catastrophic-backtracking shapes before use; preferring the AST-based approach this research recommends over raw regex avoids the ReDoS class entirely, since tree-sitter's own parse is already bounded by `MaxSourceBytes` |

## Sources

### Primary (HIGH confidence — VERIFIED against authoritative source)
- `proxy.golang.org` module proxy — direct `go list -m -versions` and raw `.mod` fetches for all 10 new grammar modules (existence, version, `go-tree-sitter` compatibility requirement)
- GitHub API (`api.github.com/repos/tree-sitter/...`) — direct `src/` directory listings confirming external-scanner (`scanner.c`) presence per grammar, and `queries/` listings confirming `tags.scm` availability
- `colbymchenry/codegraph` live GitHub source (the parity target itself) — `src/types.ts` (NodeKind/EdgeKind unions), `src/resolution/index.ts` (extends→implements promotion, conformance-pass deferral), `src/extraction/tree-sitter.ts` (Go interface-method extraction), `src/resolution/frameworks/{go,java,csharp,python}.ts` (per-framework detect/extract patterns), `CHANGELOG.md` (#584 — Go implicit-interface dispatch feature description)

### Secondary (MEDIUM confidence)
- This project's own already-committed, already-verified code: `internal/parser/parser.go`, `internal/parser/cgo/parser_cgo.go`, `internal/indexer/{discover,extract,resolve,symbolindex,pipeline}.go`, `internal/indexer/goextract/{goextract,types}.go`, `internal/schema/graph.pb.go`, `internal/query/traverse.go` — read directly this session, ground truth for this project's current state

### Tertiary (LOW confidence — training knowledge, flagged in Assumptions Log)
- Java/C#/Python/TS-JS import/namespace/module resolution semantics beyond what was directly observed in the parity target's source (A1)
- Whether the parity target's framework `detect()` heuristics generalize cleanly to the specific frameworks this project targets (A2) — the parity target's OWN detect() code for Spring/ASP.NET/Django was read directly (making those specific claims MEDIUM), but the generalization claim itself (that this pattern works for all listed frameworks without further tuning) is LOW/A2

## Metadata

**Confidence breakdown:**
- Standard stack (grammar versions/pins): HIGH — verified directly against `proxy.golang.org`
- Architecture (extraction/resolution generalization patterns): MEDIUM-HIGH — grounded in this project's own read code plus the parity target's live source, but the specific refactor shape (Wave A) is a recommendation, not a verified fact
- Dispatch synthesis / route detection vocabulary (node/edge kind names): HIGH — directly read from parity target's `types.ts`/`CHANGELOG.md`/framework resolver source
- Per-language resolution semantics (Java/C#/Python/TS import models): LOW-MEDIUM — training knowledge, not independently verified this session (see Assumptions Log A1)
- Pitfalls (worker-pool, symbol-index key generalization): HIGH — directly derived from reading this project's own current code, not speculative

**Research date:** 2026-07-11
**Valid until:** 30 days for grammar version pins (tree-sitter grammars release moderately frequently); indefinite for the architectural findings (Pitfalls 1-2, parity-target vocabulary) barring a parity-target rewrite
