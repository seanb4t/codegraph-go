---
phase: 05-language-coverage-resolution-breadth
verified: 2026-07-12T16:38:09Z
gap_closed: 2026-07-12
status: gaps_closed
score: 10/10 must-haves verified
behavior_unverified: 0
overrides_applied: 0
gaps: []
gaps_closed:
  - truth: "Java/C#/Python/TS-JS full extraction + cross-file resolution validated on real repos (ROADMAP SC1 / D-12)"
    status: closed
    reason: >
      Original finding: Python + TS/JS were genuinely validated end-to-end against
      real repos in Phase 5 (litellm/types 168 files: 2099 nodes/2068 edges/58
      calls+81 embeds resolved per 05-06-SUMMARY.md; ccstatusline 13,464 files:
      83,184 nodes/123,682 edges/48,809 calls+7,205 embeds resolved per
      05-07-SUMMARY.md), but Java's TestGoldenParity_Java was only smoke-tested
      against a synthetic 3-file tree and C#'s TestGoldenParity_CSharp was never
      run against any corpus at all.

      Closed by a gap-closure session (see 05-GAP-SUMMARY.md): ran the pre-existing,
      unmodified TestGoldenParity_Java against a real local checkout of
      temporalio/sdk-java (1,223 .java files, commit f919926a2d67c10c34fee4b19eed1c605d4223a4)
      — files=1271 nodes=14405 edges=19037, nodeKinds={file:1223 interface:500
      method:9539 struct:2046}, edgeKinds={calls:6141 contains:12219 embeds:267
      implements:410}, deterministic rebuild confirmed. Ran TestGoldenParity_CSharp
      against a shallow clone of serilog/serilog (Apache-2.0, commit
      07d39cfb2928076ecd902a61d295f90d74fe1fa5, 216 .cs files) — files=216
      nodes=1818 edges=1633, nodeKinds={file:216 interface:15 method:969
      struct:242}, edgeKinds={calls:387 contains:1227 embeds:17 implements:2},
      deterministic rebuild confirmed. No extraction/resolution bugs were found
      on either real corpus — no source changes were required. Full repo suite
      (go build ./... && go vet ./... && go test ./... ./testdata/golden/...
      -count=1, 27 targets) remains green.
    artifacts:
      - path: "testdata/golden/parity_java_test.go"
        issue: "RESOLVED — exercised against a real 1,223-file Java repo (temporalio/sdk-java), all shape + determinism checks pass"
      - path: "testdata/golden/parity_csharp_test.go"
        issue: "RESOLVED — exercised against a real 216-file C# repo (serilog), all shape + determinism checks pass"
    evidence: ".planning/phases/05-language-coverage-resolution-breadth/05-GAP-SUMMARY.md"
human_verification: []
---

# Phase 5: Language Coverage & Resolution Breadth Verification Report

**Phase Goal:** Parity language support with cross-file resolution, dynamic-dispatch synthesis carrying provenance, and framework-aware routing for the priority stack.
**Verified:** 2026-07-12T16:38:09Z
**Gap closed:** 2026-07-12 (see `05-GAP-SUMMARY.md`)
**Status:** gaps_closed
**Re-verification:** No — initial verification, gap closed in a follow-up gap-closure session

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Priority-4 (Java/C#/Python/TS-JS) get full structural extraction + cross-file resolution (imports/calls/inheritance) | ✓ VERIFIED | `internal/indexer/javaextract`, `csharpextract`, `pyextract`, `tsextract` all exist, registered in `internal/indexer/languages_*.go`; each has table-driven extraction tests plus dedicated cross-file resolution tests (e.g. `TestResolve_FullyQualifiedCrossNamespaceCall`/`Inheritance` for C#); all pass in `go test ./...` |
| 2 | Priority-4 languages validated on real repos (D-12 acceptance gate) | ✓ VERIFIED (gap closed 2026-07-12) | Python: real 168-file `litellm/types` corpus — 2099 nodes/2068 edges, 58 calls + 81 embeds resolved, deterministic rebuild (05-06-SUMMARY.md). TS/JS: real 13,464-file `ccstatusline` corpus — 83,184 nodes/123,682 edges, 48,809 calls + 7,205 embeds, deterministic rebuild (05-07-SUMMARY.md). Java: real 1,223-file `temporalio/sdk-java` corpus — 14,405 nodes/19,037 edges, 6,141 calls + 267 embeds + 410 implements, deterministic rebuild (05-GAP-SUMMARY.md). C#: real 216-file `serilog` corpus — 1,818 nodes/1,633 edges, 387 calls + 17 embeds + 2 implements, deterministic rebuild (05-GAP-SUMMARY.md). All four `TestGoldenParity_*` harnesses remain self-skipping in the committed repo (corpora are user-provided, not committed) — same precedent as Python/TS-JS |
| 3 | Mainstream tier (Rust/Ruby/PHP/C/C++/Swift/Kotlin) at full or documented-partial coverage | ✓ VERIFIED | All 7 packages exist under `internal/indexer/mainstream/`, registered via `languages_*.go`; `docs/LANGUAGE-CAPABILITY-MATRIX.md` + `internal/indexer/capability/matrix.go` document Extraction=full, Resolution/Dispatch/Routing=partial-or-none with named gaps per language; `matrix_test.go`'s `TestMatrix_PartialEntriesNameGaps` enforces every partial cell names ≥1 gap |
| 4 | Interface→implementation dispatch edges synthesized (Go structural, Java/C# declared) so callers/impact traverse dynamic dispatch | ✓ VERIFIED | `internal/indexer/dispatch/implements.go` (Go structural method-set match, bounded via inverted method-name index per D-06); `resolve.go` Pattern-2 declared-implements promotion (lines ~116-150) for Java/C#; `internal/query/traverse.go`'s `BuildImplementsIndex` + `dispatchSiblingIDs` composed into Callers/Impact; proven by passing `TestCallers_DispatchTraversal`, `TestCallers_DispatchTraversal_NoImplementsEdgesUnaffected`, `TestImpact_DispatchTraversal` |
| 5 | Every synthesized/heuristic edge carries `provenance: heuristic` + source location, distinct from `ast` edges | ✓ VERIFIED | `dispatch/implements.go`: `Provenance: "heuristic"`, `Metadata: {"synthesizedBy": "go-structural-methodset"}` (no Line/Col — structural synthesis has no single source location, an accepted design per the package doc). `resolve.go` declared-implements: `Line`/`Col` from the resolved ref + `Provenance: "heuristic"` + `Metadata: {"synthesizedBy": "declared-implements"}`. Route edges: heuristic-provenance `calls` edges per `routes/registry.go` doc comment. Ground-truth edges (`calls`, `embeds`, `contains`) keep `Provenance: "ast"` |
| 6 | No schema-version bump | ✓ VERIFIED | `internal/schema/meta.go`: `SchemaVersion uint32 = 1`, unchanged |
| 7 | Framework-aware routing for Gin/Spring/ASP.NET/Django-Flask-FastAPI/Express-NestJS emits route nodes linked to handlers | ✓ VERIFIED | `internal/indexer/routes/{gin,spring,aspnet,django,express}.go` all exist; `TestRoute_RegistryHasFivePriorityFrameworks` passes; each framework has dedicated detection + opt-in signature tests (all passing); `TestRoute_ASTNotRegex` confirms AST-based (not regex) detection; wired into `pipeline.go` via `detectRoutes` |
| 8 | D-05 deferred Go resolution fixes land in this phase (WR-01 collision handling, WR-02 selector mis-resolution, call-as-argument gap) | ✓ VERIFIED | `TestSymbolIndex_WR01_CollisionDeterministicOrderIndependent`, `TestExtract_SelectorNonIdentifierOperandNeverAliasQualified`, `TestExtract_CallAsArgument` all exist and pass |
| 9 | Determinism preserved; Go's own golden-parity fixture unregressed by the multi-language generalization | ✓ VERIFIED | `go test ./testdata/golden/... -run TestGoldenParity -v`: all 7 subtests pass (status/query/callers/callees/impact/explore/node) against the pinned `weft` corpus fixture |
| 10 | D-11 capability matrix (human + machine readable) committed and consistency-checked | ✓ VERIFIED | `docs/LANGUAGE-CAPABILITY-MATRIX.md` (14 rows) + `internal/indexer/capability/matrix.go`; `TestMatrix_CoversRegisteredLanguages`, `TestMatrix_CellsValid`, `TestMatrix_DocMirrorsDescriptor`, `TestMatrix_FullPriority4EntriesHaveGoldenTest`, `TestMatrix_PartialEntriesNameGaps` all pass |

**Score:** 10/10 truths verified (0 present-behavior-unverified) — gap closed 2026-07-12

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/indexer/languages.go` + `languages_*.go` (14 files) | Registry keyed by language ID | ✓ VERIFIED | 14 `LanguageSpec` registrations (go, java, csharp, python, typescript, tsx, javascript, rust, ruby, php, c, cpp, swift, kotlin) |
| `internal/indexer/javaextract/`, `csharpextract/`, `pyextract/`, `tsextract/` | Priority-4 extractors | ✓ VERIFIED | All exist with types.go, main extract file, table-driven tests — all passing |
| `internal/indexer/mainstream/{rustextract,rubyextract,phpextract,cextract,swiftextract,kotlinextract}/` | Mainstream-6 extractors | ✓ VERIFIED | All exist, registered, tested |
| `internal/indexer/dispatch/implements.go` | RES-02 synthesis | ✓ VERIFIED | Bounded structural synthesis, tested |
| `internal/indexer/routes/{registry,gin,spring,aspnet,django,express}.go` | LANG-07 routing | ✓ VERIFIED | All 5 framework families present |
| `internal/indexer/capability/matrix.go` + `docs/LANGUAGE-CAPABILITY-MATRIX.md` | D-11 matrix | ✓ VERIFIED | Consistency-checked, all tests pass |
| `testdata/golden/parity_java_test.go`, `parity_csharp_test.go`, `parity_python_test.go`, `parity_tsjs_test.go` | D-12 validation harnesses | ✓ VERIFIED | All 4 exist as well-built self-skipping harnesses; all four now have documented real-repo evidence (Python/TS-JS: 05-06/05-07-SUMMARY.md; Java/C#: 05-GAP-SUMMARY.md) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `pipeline.go` `Run` | `detectRoutes` | direct call at pipeline.go:108 | ✓ WIRED | Route nodes/edges folded into the same resolve pass |
| `resolve.go` `RefKindEmbeds` handling | `dispatch.EdgeKindImplements` promotion | Pattern-2 inline promotion when target is `KindInterface` | ✓ WIRED | Sets `Provenance: "heuristic"`, `Line`/`Col`, `Metadata` |
| `internal/query/traverse.go` `Callers`/`Impact` | `BuildImplementsIndex` | composed via `dispatchSiblingIDs` | ✓ WIRED | Proven by passing dispatch-traversal tests |
| `internal/indexer/routes/registry.go` | route→handler edges | `goextract.RefKindCalls` + `Provenance: "heuristic"` | ✓ WIRED | Reuses existing `calls` edge kind so `BuildReverseAdjacency` picks them up unchanged |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| LANG-02 | 05-01,02,03,04 | Java full extraction+resolution | ✓ SATISFIED | Real 1,223-file `temporalio/sdk-java` corpus validated (05-GAP-SUMMARY.md) |
| LANG-03 | 05-01,02,03,05 | C# full extraction+resolution | ✓ SATISFIED | Real 216-file `serilog` corpus validated (05-GAP-SUMMARY.md) |
| LANG-04 | 05-01,02,03,06 | Python full extraction+resolution | ✓ SATISFIED | Real 168-file corpus validated |
| LANG-05 | 05-01,02,03,07 | TS/JS full extraction+resolution | ✓ SATISFIED | Real 13,464-file corpus validated |
| LANG-06 | 05-08,10,11,13 | Mainstream-6 full/documented-partial | ✓ SATISFIED | Matrix + gap-naming tests pass |
| LANG-07 | 05-12 | Framework routing, 5 families | ✓ SATISFIED | All 5 detectors present, tested, wired |
| RES-02 | 05-09 | Dispatch synthesis | ✓ SATISFIED | Bounded synthesis + query traversal tests |
| RES-03 | 05-09 | Provenance tagging | ✓ SATISFIED | `heuristic` vs `ast` provenance confirmed, no schema bump |

No orphaned requirements — all 8 phase requirement IDs (LANG-02..07, RES-02, RES-03) appear in at least one plan's `requirements` frontmatter and are traced above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/indexer/resolve.go` | ~262-264 | `gofmt -l` violation (pre-existing var-decl comment misalignment, unrelated function) | ℹ️ Info | Confirmed pre-existing per `deferred-items.md` (verified via `git stash` before 05-12's own edit); not a Phase-5 regression, cosmetic only |

No `TBD`/`FIXME`/`XXX` debt markers found in any Phase-5-touched file. No stub returns, empty handlers, or hardcoded-empty data flows found — "placeholder" occurrences are all legitimate architectural doc comments describing the discovery-time path-based `ModuleKey` fallback concept, not incomplete code.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full test suite (single run, non-race, matches guidance) | `go test ./...` | All 26 packages `ok` | ✓ PASS |
| `go build ./...` | exit 0 | ✓ PASS |
| `go vet ./...` | exit 0, no output | ✓ PASS |
| Go golden-parity fixture unregressed | `go test ./testdata/golden/... -run TestGoldenParity -v` | 7/7 subtests pass | ✓ PASS |
| D-12 priority-4 harnesses currently runnable state | same command, `TestGoldenParity_{Java,CSharp,Python,TSJS}` | All 4 self-skip cleanly (no corpus committed) | ? SKIP (expected — corpora are user-provided, not committed); all 4 confirmed passing with real corpora configured (see 05-GAP-SUMMARY.md for Java/C#, 05-06/05-07-SUMMARY.md for Python/TS-JS) |
| D-11 capability matrix consistency | `go test ./internal/indexer/capability/... -v` | 7/7 subtests pass | ✓ PASS |
| Dispatch traversal | tests included in full suite run above | `TestCallers_DispatchTraversal`, `TestImpact_DispatchTraversal` pass | ✓ PASS |
| Route detection AST-not-regex + 5-framework registry | tests included in full suite run above | `TestRoute_RegistryHasFivePriorityFrameworks`, `TestRoute_ASTNotRegex` pass | ✓ PASS |

Note: the guidance's known pre-existing `internal/daemon.TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock` race flake was not triggered because the full suite was run non-race (per guidance instruction), matching the sanctioned verification method for this phase.

### Human Verification Required

None. All observable truths are objectively verifiable from the codebase and SUMMARY/gap-closure evidence.

### Gaps Summary

**RESOLVED 2026-07-12.** Phase 5 delivers a solid, well-tested, honestly-documented multi-language extraction/resolution/dispatch/routing architecture, and all ten observable truths are now fully verified against the actual codebase (not just SUMMARY narrative): all 14 languages are registered and extract correctly, dispatch synthesis and provenance tagging work exactly as RES-02/RES-03 require with no schema bump, all five framework route detectors exist and are wired, the D-11 capability matrix is committed and self-consistency-checked, the three D-05 Go resolution fixes (WR-01/WR-02/call-as-argument) all landed with passing tests, and all four priority-4 languages now have real-repo D-12 validation evidence.

The original finding was ROADMAP Success Criterion 1's explicit "validated on real repos" bar for the four priority-4 languages. Python and TypeScript/JavaScript met this bar within Phase 5 itself (litellm/types 168 files, ccstatusline 13,464 files — 05-06/05-07-SUMMARY.md). Java (only a synthetic 3-file smoke test) and C# (no smoke test evidence at all) did not, at initial verification time.

A gap-closure session (2026-07-12, see `05-GAP-SUMMARY.md`) closed this: `TestGoldenParity_Java` was run against a real local checkout of `temporalio/sdk-java` (1,223 files, commit `f919926a2d67c10c34fee4b19eed1c605d4223a4`) — 14,405 nodes/19,037 edges, 6,141 calls + 267 embeds + 410 implements resolved, deterministic rebuild confirmed. `TestGoldenParity_CSharp` was run against a real shallow clone of `serilog/serilog` (Apache-2.0, 216 files, commit `07d39cfb2928076ecd902a61d295f90d74fe1fa5`) — 1,818 nodes/1,633 edges, 387 calls + 17 embeds + 2 implements resolved, deterministic rebuild confirmed. No extraction or resolution bugs were found on either real corpus — the pre-existing `javaextract`/`csharpextract` implementations (05-04/05-05) worked correctly against real, unmodified, third-party source without any code changes. The full repo suite (`go build ./... && go vet ./... && go test ./... ./testdata/golden/... -count=1`, 27 targets) remains green.

All four priority-4 languages (Java, C#, Python, TS/JS) now share the same real-repo D-12 evidence bar. No further action is required against this phase's success criteria.

---

*Verified: 2026-07-12T16:38:09Z*
*Verifier: Claude (gsd-verifier)*
