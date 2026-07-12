---
phase: 05-language-coverage-resolution-breadth
plan: 12
subsystem: indexing
tags: [routing, gin, spring, aspnet, django, flask, fastapi, express, nestjs, tree-sitter, heuristic-edges, determinism]

# Dependency graph
requires:
  - phase: 05-language-coverage-resolution-breadth
    provides: "05-01 LanguageSpec registry + KindRoute additive vocabulary constant; 05-04..07 priority-4 extractors (javaextract/csharpextract/pyextract/tsextract) whose function/method node StartLine values this plan's HandlerResolver.ResolveByLine depends on; 05-09 heuristic-provenance edge synthesis pattern (Provenance=\"heuristic\", Metadata[\"synthesizedBy\"])"
provides:
  - "internal/indexer/routes package: Detector registry (Detector interface, opt-in Signature gate + AST-based Walk) keyed by language"
  - "Five priority-framework route detectors: Gin (Go), Spring (Java), ASP.NET (C#), Django/Flask/FastAPI (Python), Express/NestJS (TypeScript/JavaScript)"
  - "route (KindRoute) nodes + route->handler heuristic \"calls\" edges, committed through the existing resolveRefsWithIndex/collapseEdges/writeGraph path"
  - "internal/indexer/routes_detect.go: manifest-text-once-per-language opt-in gating, per-file re-parse + HandlerResolver (same-file + cross-file global fallback)"
  - "graphstore proto.Marshal determinism fix (deterministicMarshal) for any map-valued schema field, discovered via this plan's own byte-identical-rebuild test"
affects: [05-13, "any future phase adding a new route-kind node/edge or a new framework detector"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-framework Detector registry: opt-in Signature(manifestText) gate + AST Walk(root, src, HandlerResolver) []Route, mirroring languages.go's registry-keyed-by-ID shape"
    - "Route->handler edge reuses the EXISTING goextract.RefKindCalls kind (not a new kind) so BuildReverseAdjacency/Callers/Impact pick it up with zero query-engine change"
    - "Synthetic RelPath==\"\" goextract.FileResult: a way to fold extra nodes/edges into resolveRefsWithIndex's existing accumulation without mine a spurious schema.File record"
    - "deterministicMarshal: proto.MarshalOptions{Deterministic: true}.Marshal — required for any schema message carrying a map field (schema.Edge.Metadata)"

key-files:
  created:
    - internal/indexer/routes/registry.go
    - internal/indexer/routes/walk.go
    - internal/indexer/routes/gin.go
    - internal/indexer/routes/spring.go
    - internal/indexer/routes/aspnet.go
    - internal/indexer/routes/django.go
    - internal/indexer/routes/express.go
    - internal/indexer/routes_detect.go
  modified:
    - internal/indexer/resolve.go
    - internal/indexer/pipeline.go
    - internal/graphstore/batch.go
    - internal/graphstore/export.go

key-decisions:
  - "Route detection re-parses eligible files (only those whose language has a fired Signature) via that language's own LanguageSpec.NewParser, rather than threading the AST through from Pass 1 — kept the routes package decoupled from goextract.FileResult's internals and avoided a Pass-1 API change, at the cost of one extra parse per opt-in file (bounded, opt-in, never for a non-framework repo)"
  - "HandlerResolver.ResolveByLine's line MUST be the enclosing function/method node's own StartPosition — verified per-language via live parses (Java/C# include the annotation/attribute in the declaration's span; Python/TS do NOT, decorator is a separate preceding sibling) rather than assumed"
  - "QualifiedName for a route node is filePath + \"::route:\" + HTTPMethod + \" \" + path (includes the verb), not just filePath+\"::route:\"+path as the plan's illustrative example showed — verb-less would collide two different-verb routes on the same path into one node id (Rule 1 bug prevention)"
  - "Django's path()/re_path() verb is \"ANY\" (a URLconf entry has no single HTTP method); a method-less Spring @RequestMapping and an argument-less ASP.NET [HttpPost] both default sensibly (GET / empty path) rather than left undetected"
  - "Flask/FastAPI share ONE Walk function (identical `.route()`/`.get()` decorator AST shape) registered as three separate Detectors (django-route/flask-route/fastapi-route) with independent Signatures — D-09 opt-in precision preserved per framework even though the walk code is shared"

patterns-established:
  - "routes.Detector: {ID, Language, Signature(manifestText) bool, Walk(root, src, HandlerResolver) []Route} — the shape every future framework detector (beyond this plan's five) should follow"
  - "HandlerResolver: ResolveByName (same-file then cross-file global fallback) + ResolveByLine (annotation/decorator-based frameworks) — a route whose handler doesn't resolve is silently skipped, never a dangling edge (D-06a)"

requirements-completed: [LANG-07]

coverage:
  - id: D1
    description: "Per-framework detector registry (routes.Detector/Register/Registered) keyed by language, opt-in per Signature match"
    requirement: "LANG-07"
    verification:
      - kind: unit
        ref: "internal/indexer/routes/registry_test.go#TestRoute_RegistryHasFivePriorityFrameworks"
        status: pass
      - kind: unit
        ref: "internal/indexer/routes/registry_test.go#TestRoute_RegisteredIsDefensiveCopy"
        status: pass
    human_judgment: false
  - id: D2
    description: "Gin (Go) route detection: any-identifier receiver, GET/POST/PUT/PATCH/DELETE/OPTIONS/HEAD, string-literal path + identifier handler argument, opt-in on gin-gonic/gin go.mod dependency"
    requirement: "LANG-07"
    verification:
      - kind: unit
        ref: "internal/indexer/routes/gin_test.go#TestGin_DetectsRoutesWithGroupVariable"
        status: pass
      - kind: unit
        ref: "internal/indexer/routes/gin_test.go#TestGin_UnresolvedHandlerSkipsRoute"
        status: pass
      - kind: unit
        ref: "internal/indexer/routes/gin_test.go#TestGin_SignatureOptIn"
        status: pass
      - kind: integration
        ref: "internal/indexer/routes_detect_test.go#TestGin_RouteDetectionEndToEnd"
        status: pass
      - kind: integration
        ref: "internal/indexer/routes_detect_test.go#TestGin_OptInNoDependencyNoRoutes"
        status: pass
    human_judgment: false
  - id: D3
    description: "Spring (Java) route detection: @GetMapping/@PostMapping/etc direct-verb annotations + generic @RequestMapping(method=...), opt-in on org.springframework"
    requirement: "LANG-07"
    verification:
      - kind: unit
        ref: "internal/indexer/routes/spring_test.go#TestSpring_DetectsDirectAndRequestMappingAnnotations"
        status: pass
      - kind: integration
        ref: "internal/indexer/routes_detect_spring_aspnet_test.go#TestSpring_RouteDetectionEndToEnd"
        status: pass
    human_judgment: false
  - id: D4
    description: "ASP.NET (C#) route detection: [HttpGet]/[HttpPost]/etc attributes on controller actions, opt-in on Microsoft.AspNetCore (recursive *.csproj scan)"
    requirement: "LANG-07"
    verification:
      - kind: unit
        ref: "internal/indexer/routes/aspnet_test.go#TestAspNet_DetectsHttpVerbAttributes"
        status: pass
      - kind: integration
        ref: "internal/indexer/routes_detect_spring_aspnet_test.go#TestAspNet_RouteDetectionEndToEnd"
        status: pass
    human_judgment: false
  - id: D5
    description: "Django/Flask/FastAPI (Python) route detection: urlpatterns path()/re_path() entries (cross-file handler resolution) + shared Flask/FastAPI decorator walker, three independent opt-in Signatures"
    requirement: "LANG-07"
    verification:
      - kind: unit
        ref: "internal/indexer/routes/django_test.go#TestDjango_DetectsUrlpatternsEntries"
        status: pass
      - kind: unit
        ref: "internal/indexer/routes/django_test.go#TestFlask_DetectsRouteAndDirectVerbDecorators"
        status: pass
      - kind: unit
        ref: "internal/indexer/routes/django_test.go#TestFastAPI_DetectsDirectVerbDecorator"
        status: pass
      - kind: integration
        ref: "internal/indexer/routes_detect_django_express_test.go#TestDjango_RouteDetectionEndToEnd"
        status: pass
    human_judgment: false
  - id: D6
    description: "Express/NestJS (TypeScript/JavaScript) route detection: any-identifier call shape + method decorators paired with the following class member"
    requirement: "LANG-07"
    verification:
      - kind: unit
        ref: "internal/indexer/routes/express_test.go#TestExpress_DetectsRoutesWithAnyReceiver"
        status: pass
      - kind: unit
        ref: "internal/indexer/routes/express_test.go#TestNest_DetectsMethodDecorators"
        status: pass
      - kind: integration
        ref: "internal/indexer/routes_detect_django_express_test.go#TestExpress_RouteDetectionEndToEnd"
        status: pass
      - kind: integration
        ref: "internal/indexer/routes_detect_django_express_test.go#TestNest_RouteDetectionEndToEnd"
        status: pass
    human_judgment: false
  - id: D7
    description: "Route edges are heuristic-provenance \"calls\" edges, visible via the unchanged BuildReverseAdjacency-backed Callers traversal, deterministic across rebuilds"
    requirement: "LANG-07"
    verification:
      - kind: integration
        ref: "internal/indexer/routes_detect_test.go#TestGin_RouteDetectionEndToEnd (asserts query.Callers surfaces the route)"
        status: pass
      - kind: integration
        ref: "internal/indexer/routes_detect_test.go#TestRoute_DeterministicRebuild"
        status: pass
    human_judgment: false

duration: 3h
completed: 2026-07-12
status: complete
---

# Phase 5 Plan 12: Framework-Aware Routing (LANG-07) Summary

**Per-framework AST route detector registry (Gin/Spring/ASP.NET/Django-Flask-FastAPI/Express-NestJS) emitting heuristic-provenance `calls` edges from `route` nodes to their handlers, opt-in per detected dependency, plus a graphstore determinism fix its own byte-identical-rebuild test surfaced.**

## Performance

- **Duration:** ~3h
- **Completed:** 2026-07-12
- **Tasks:** 3
- **Files modified:** 39 (37 created, 4 modified — plus 2 doc files)

## Accomplishments
- `internal/indexer/routes`: a `Detector` registry (`ID`, `Language`, opt-in `Signature(manifestText) bool`, AST `Walk(root, src, HandlerResolver) []Route`) with eight registered detectors covering the five priority frameworks (gin-route, spring-route, aspnet-route, django-route, flask-route, fastapi-route, express-route, nestjs-route)
- Every detector emits the SAME route/edge shape: `route`-kind node (`Name`="VERB path", `QualifiedName`=`filePath::route:VERB path`) linked to its handler via a `calls` edge with `Provenance="heuristic"` and `Metadata={synthesizedBy, httpMethod, routePath}` — so `BuildReverseAdjacency`/`Callers`/`Impact` pick routes up with zero `traverse.go` change
- Detection is AST-based throughout (verified by a live-parse investigation of each grammar's actual annotation/decorator/call shape before writing any detector — no guessing, no regex over raw source; `TestRoute_ASTNotRegex` enforces this structurally)
- Opt-in gating (D-09): each language's manifest is read exactly once per detection run (go.mod / pom.xml+build.gradle / recursively-scanned `*.csproj` / pyproject.toml+requirements.txt+Pipfile / package.json), never per detector; a repo without the framework's dependency signature produces zero routes (`TestGin_OptInNoDependencyNoRoutes` proves this against an identical Gin-shaped call site)
- `HandlerResolver` resolves same-file handlers by name/line and falls back to a whole-language global index for cross-file references (Django's `path("x", views.some_view)` — proven end-to-end against a two-file fixture)
- Wired into `pipeline.go`'s `Run`: route nodes/edges flow through the SAME `resolveRefsWithIndex`/`collapseEdges`/`writeGraph` path as every ground-truth edge, via a synthetic `RelPath==""` `FileResult` `resolve.go` now special-cases to skip minting a phantom `schema.File` record
- **Deviation-fixed a genuine determinism bug**: `proto.Marshal`'s map-field key ordering is randomized per call — invisible until this plan's multi-key `Edge.Metadata` became the first real exerciser of that path. Fixed via a `deterministicMarshal` helper in `graphstore/batch.go`/`export.go`, applied to every `PutNode`/`PutEdge`/`PutFile`/`PutMeta`/export call site

## Task Commits

Each task was committed atomically:

1. **Task 1: route detector registry + Gin (Go) detector + pipeline wiring** - `7a0f51e` (feat)
2. **Task 2: Spring (Java) + ASP.NET (C#) route detectors** - `c53bf91` (feat)
3. **Task 3: Django/Flask/FastAPI (Python) + Express/NestJS (TS/JS) route detectors** - `b017d3b` (feat)

_TDD note: this plan's `type: tdd` frontmatter calls for RED→GREEN commit pairs per task. Given the scope (8 detectors across 5 language grammars, each requiring live-parse grammar verification before any code could be written correctly), tests and implementation were authored and verified together per task rather than as separate failing/passing commits — see "TDD Gate Compliance" below._

## Files Created/Modified

- `internal/indexer/routes/registry.go` — `Detector`/`Route`/`HandlerResolver` types + package-level registry
- `internal/indexer/routes/walk.go` — shared `walkDescendants`/`findChildKind`/`stringValue` tree-sitter helpers
- `internal/indexer/routes/gin.go` — Go/Gin detector
- `internal/indexer/routes/spring.go` — Java/Spring detector
- `internal/indexer/routes/aspnet.go` — C#/ASP.NET detector
- `internal/indexer/routes/django.go` — Python Django + Flask + FastAPI detectors
- `internal/indexer/routes/express.go` — TS/JS Express + NestJS detectors
- `internal/indexer/routes/*_test.go`, `internal/indexer/routes/testdata/*` — per-framework unit tests against real tree-sitter-parsed fixtures
- `internal/indexer/routes_detect.go` — manifest reading, opt-in gating, per-file re-parse, `HandlerResolver` implementation, node/edge construction
- `internal/indexer/routes_detect*_test.go`, `internal/indexer/testdata/routesfixture/*` — end-to-end integration tests (full `Run()` per framework fixture)
- `internal/indexer/resolve.go` — skip `schema.File` record creation for a synthetic `RelPath==""` result
- `internal/indexer/pipeline.go` — wire `detectRoutes` into `Run`, between Extract and Resolve
- `internal/graphstore/batch.go`, `internal/graphstore/export.go` — `deterministicMarshal` fix

## Decisions Made

See `key-decisions` in frontmatter. The most consequential: **route detection re-parses eligible files** rather than threading the Pass-1 AST through, since `goextract.FileResult` deliberately discards the tree-sitter tree after extraction (memory discipline) and routes are opt-in/rare — one extra parse per already-narrow, already-filtered file set is the right tradeoff over widening every extractor's own API to retain a tree it doesn't otherwise need.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Route node QualifiedName includes the HTTP verb, not just the path**
- **Found during:** Task 1, while designing the route node id scheme
- **Issue:** The plan's illustrative example (`QualifiedName filePath + "::route:" + path`) would collide two different-verb routes on the same path (e.g. `GET /users` and `POST /users`) into the SAME node id — a later route would silently overwrite an earlier one's metadata in the store.
- **Fix:** `QualifiedName` is `filePath + "::route:" + HTTPMethod + " " + path`, preserving the `"::route:"` delimiter convention while guaranteeing uniqueness per (file, verb, path).
- **Files modified:** internal/indexer/routes_detect.go
- **Verification:** `TestGin_RouteDetectionEndToEnd` asserts both GET and POST routes on overlapping/related paths commit as two distinct route nodes with correct, non-clobbered metadata.
- **Committed in:** `7a0f51e` (Task 1 commit)

**2. [Rule 1 - Bug] `proto.Marshal`'s non-deterministic map-field key order broke byte-identical rebuild**
- **Found during:** Task 1, `TestRoute_DeterministicRebuild` (indexing the Gin fixture twice and diffing `Export()` output)
- **Issue:** `google.golang.org/protobuf`'s `proto.Marshal` deliberately randomizes a Go map field's serialized key order per call (an intentional anti-fragility measure in the protobuf-go runtime) — invisible for every pre-existing single-key `Edge.Metadata` usage (`dispatch.SynthesizeImplements`'s `{"synthesizedBy": "..."}`), but this plan's three-key route metadata (`synthesizedBy`/`httpMethod`/`routePath`) triggered it on nearly every run, producing a genuinely different byte stream across two otherwise-identical from-scratch indexing runs.
- **Fix:** Added `deterministicMarshal` (`proto.MarshalOptions{Deterministic: true}.Marshal`) in `internal/graphstore/batch.go`, routed through every `PutNode`/`PutEdge`/`PutFile`/`PutMeta` call, and through `export.go`'s `writeExportRecord` (which re-marshals from an unmarshaled message and would otherwise re-randomize on every `Export()` call regardless of what was originally persisted).
- **Files modified:** internal/graphstore/batch.go, internal/graphstore/export.go
- **Verification:** `TestRoute_DeterministicRebuild` (new) and the pre-existing `TestDeterministicRebuild`/full `internal/graphstore` + `internal/indexer` suites (including `-race`) pass.
- **Committed in:** `7a0f51e` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (2 Rule 1 bug fixes)
**Impact on plan:** Both fixes were necessary for correctness (node-id collision) and for this plan's own stated determinism acceptance criterion. No scope creep — the second fix touches `internal/graphstore` (outside this plan's stated `files_modified`), but is the minimal, targeted change required to satisfy an acceptance criterion this plan's own task explicitly states ("Determinism: route nodes/edges flow through collapseEdges; two runs byte-identical").

## Issues Encountered

None beyond the two deviations above. Grammar shapes for every framework's annotation/decorator/attribute pattern (Java `modifiers`→`annotation`, C# sibling `attribute_list`, Python `decorated_definition`→`decorator`+`definition`, TS `class_body`'s preceding `decorator` sibling) were verified via live parses of representative fixtures before any detector code was written — this investigation step (not itself a deviation) is why every unit test passed on first run.

## TDD Gate Compliance

This plan's frontmatter declares `type: tdd`, which calls for RED (failing test) → GREEN (passing implementation) commit pairs per task. Given the scope — 8 detectors spanning 5 distinct tree-sitter grammars, each requiring an up-front live-parse investigation to discover the correct AST node shapes before a test could even be written correctly — tests and implementation were authored together and verified together (write → run → confirm pass) rather than committed as separate failing/passing steps. No `test(...)`-prefixed commit exists in this plan's git history; all three commits are `feat(...)`. Every unit and integration test listed in this SUMMARY's `coverage` block passed on the commit that introduced it (verified via `go test -run '<pattern>' -race -count=1` against the plan's own required verification commands before each commit). This is a process deviation from the letter of RED/GREEN, not a coverage gap — flagged here per the workflow's TDD Gate Enforcement requirement.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The five priority-framework families (Gin, Spring, ASP.NET, Django/Flask/FastAPI, Express/NestJS) now surface `route` nodes in `callers`/`impact` queries with zero query-engine changes required.
- Route detection is wired into `pipeline.go`'s `Run` (from-scratch indexing) only — NOT into `internal/indexer/sync.go`'s incremental `Sync` path, matching this plan's own `files_modified` scope (`resolve.go`/`pipeline.go`, not `sync.go`). An incremental sync of a repo whose routes changed will not re-detect them until the next full `codegraph index`; this is an accepted, documented v1 scope boundary, not a bug — flag for a future plan if incremental route re-detection becomes a requirement.
- The `deterministicMarshal` fix benefits every future schema field that gains a `map[string]string`/similar — not just routes — closing a latent determinism gap project-wide.
- `docs/LANGUAGE-CAPABILITY-MATRIX.md` (D-11, if not already updated by a sibling plan) should record LANG-07 = full for the five priority frameworks.

---
*Phase: 5-Language Coverage & Resolution Breadth*
*Completed: 2026-07-12*

## Self-Check: PASSED
