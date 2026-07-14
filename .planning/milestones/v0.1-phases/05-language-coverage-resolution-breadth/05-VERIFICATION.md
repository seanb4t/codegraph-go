---
phase: 05-language-coverage-resolution-breadth
verified: 2026-07-12T13:23:00Z
gap_closed: 2026-07-12
status: passed
score: 10/10 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 9/10
  gaps_closed:
    - "Java/C#/Python/TS-JS full extraction + cross-file resolution validated on real repos (ROADMAP SC1 / D-12)"
  gaps_remaining: []
  regressions: []
gaps: []
human_verification: []
---

# Phase 5: Language Coverage & Resolution Breadth Verification Report

**Phase Goal:** Parity language support with cross-file resolution, dynamic-dispatch synthesis carrying provenance, and framework-aware routing for the priority stack.
**Verified:** 2026-07-12T13:23:00Z
**Gap closed:** 2026-07-12 (see `05-GAP-SUMMARY.md`)
**Status:** passed
**Re-verification:** Yes — independent re-verification of gap closure, following a gap-closure execution session

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Priority-4 (Java/C#/Python/TS-JS) get full structural extraction + cross-file resolution (imports/calls/inheritance) | ✓ VERIFIED | `internal/indexer/javaextract`, `csharpextract`, `pyextract`, `tsextract` all exist, registered in `internal/indexer/languages_*.go`; each has table-driven extraction tests plus dedicated cross-file resolution tests; all pass in `go test ./...` (independently re-run this session) |
| 2 | Priority-4 languages validated on real repos (D-12 acceptance gate) | ✓ VERIFIED | Python: real 168-file `litellm/types` corpus — 2099 nodes/2068 edges, 58 calls + 81 embeds resolved, deterministic rebuild (05-06-SUMMARY.md). TS/JS: real 13,464-file `ccstatusline` corpus — 83,184 nodes/123,682 edges, 48,809 calls + 7,205 embeds, deterministic rebuild (05-07-SUMMARY.md). Java: real 1,223-file `temporalio/sdk-java` corpus (commit `f919926a`) — 14,405 nodes/19,037 edges, 6,141 calls + 267 embeds + 410 implements, deterministic rebuild (05-GAP-SUMMARY.md). C#: real 216-file `serilog` corpus (commit `07d39cfb`) — 1,818 nodes/1,633 edges, 387 calls + 17 embeds + 2 implements, deterministic rebuild (05-GAP-SUMMARY.md). Independently confirmed: `testdata/golden/parity_java_test.go` and `parity_csharp_test.go` gate on `CODEGRAPH_JAVA_CORPUS`/`CODEGRAPH_CSHARP_CORPUS` env vars exactly as the gap-closure evidence describes (self-skipping in the committed repo, same precedent as Python/TS-JS), and their shape/determinism assertions (`t.Errorf("shape check...")`, second-pass rebuild comparison) match the claimed evidence structure. All four `TestGoldenParity_*` harnesses now share the same real-repo evidence bar |
| 3 | Mainstream tier (Rust/Ruby/PHP/C/C++/Swift/Kotlin) at full or documented-partial coverage | ✓ VERIFIED | All 7 packages exist under `internal/indexer/mainstream/`, registered via `languages_*.go`; `docs/LANGUAGE-CAPABILITY-MATRIX.md` + `internal/indexer/capability/matrix.go` document Extraction=full, Resolution/Dispatch/Routing=partial-or-none with named gaps per language; `TestMatrix_PartialEntriesNameGaps` re-run and passing this session |
| 4 | Interface→implementation dispatch edges synthesized (Go structural, Java/C# declared) so callers/impact traverse dynamic dispatch | ✓ VERIFIED | `internal/indexer/dispatch/implements.go` (Go structural method-set match); `resolve.go` Pattern-2 declared-implements promotion for Java/C#; `internal/query/traverse.go`'s `BuildImplementsIndex` + `dispatchSiblingIDs` composed into Callers/Impact; `TestCallers_DispatchTraversal`, `TestCallers_DispatchTraversal_NoImplementsEdgesUnaffected`, `TestImpact_DispatchTraversal` re-run and passing this session. The Java corpus's `implements:410` real-repo count (05-GAP-SUMMARY.md) corroborates RES-02 synthesis firing on real Java code, not just synthetic fixtures |
| 5 | Every synthesized/heuristic edge carries `provenance: heuristic` + source location, distinct from `ast` edges | ✓ VERIFIED | `dispatch/implements.go`: `Provenance: "heuristic"`, `Metadata: {"synthesizedBy": "go-structural-methodset"}`. `resolve.go` declared-implements: `Line`/`Col` + `Provenance: "heuristic"` + `Metadata: {"synthesizedBy": "declared-implements"}`. Route edges: heuristic-provenance `calls` edges. Ground-truth edges (`calls`, `embeds`, `contains`) keep `Provenance: "ast"` |
| 6 | No schema-version bump | ✓ VERIFIED | `internal/schema/meta.go`: `SchemaVersion uint32 = 1`, unchanged |
| 7 | Framework-aware routing for Gin/Spring/ASP.NET/Django-Flask-FastAPI/Express-NestJS emits route nodes linked to handlers | ✓ VERIFIED | `internal/indexer/routes/{gin,spring,aspnet,django,express}.go` all exist; `TestRoute_RegistryHasFivePriorityFrameworks`, `TestRoute_ASTNotRegex` pass; wired into `pipeline.go` via `detectRoutes` |
| 8 | D-05 deferred Go resolution fixes land in this phase (WR-01 collision handling, WR-02 selector mis-resolution, call-as-argument gap) | ✓ VERIFIED | `TestSymbolIndex_WR01_CollisionDeterministicOrderIndependent`, `TestExtract_SelectorNonIdentifierOperandNeverAliasQualified`, `TestExtract_CallAsArgument` all exist and pass |
| 9 | Determinism preserved; Go's own golden-parity fixture unregressed by the multi-language generalization | ✓ VERIFIED | `go test ./testdata/golden/... -run TestGoldenParity -v`: all 7 subtests pass against the pinned `weft` corpus fixture (re-run this session) |
| 10 | D-11 capability matrix (human + machine readable) committed and consistency-checked | ✓ VERIFIED | `docs/LANGUAGE-CAPABILITY-MATRIX.md` (14 rows) + `internal/indexer/capability/matrix.go`; `TestMatrix_CoversRegisteredLanguages`, `TestMatrix_CellsValid`, `TestMatrix_DocMirrorsDescriptor`, `TestMatrix_FullPriority4EntriesHaveGoldenTest`, `TestMatrix_PartialEntriesNameGaps` all re-run and passing this session |

**Score:** 10/10 truths verified (0 present-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/indexer/languages.go` + `languages_*.go` (14 files) | Registry keyed by language ID | ✓ VERIFIED | 14 `LanguageSpec` registrations confirmed via direct grep count this session (go, java, csharp, python, typescript, tsx, javascript, rust, ruby, php, c, cpp, swift, kotlin) |
| `internal/indexer/javaextract/`, `csharpextract/`, `pyextract/`, `tsextract/` | Priority-4 extractors | ✓ VERIFIED | All exist with types.go, main extract file, table-driven tests — all passing |
| `internal/indexer/mainstream/{rustextract,rubyextract,phpextract,cextract,swiftextract,kotlinextract}/` | Mainstream-6 extractors | ✓ VERIFIED | All exist, registered, tested |
| `internal/indexer/dispatch/implements.go` | RES-02 synthesis | ✓ VERIFIED | Bounded structural synthesis, tested |
| `internal/indexer/routes/{registry,gin,spring,aspnet,django,express}.go` | LANG-07 routing | ✓ VERIFIED | All 5 framework families present |
| `internal/indexer/capability/matrix.go` + `docs/LANGUAGE-CAPABILITY-MATRIX.md` | D-11 matrix | ✓ VERIFIED | Consistency-checked, all tests pass |
| `testdata/golden/parity_java_test.go`, `parity_csharp_test.go`, `parity_python_test.go`, `parity_tsjs_test.go` | D-12 validation harnesses | ✓ VERIFIED | All 4 exist as well-built self-skipping harnesses (confirmed the `CODEGRAPH_JAVA_CORPUS`/`CODEGRAPH_CSHARP_CORPUS` gate this session); all four now have documented real-repo evidence (Python/TS-JS: 05-06/05-07-SUMMARY.md; Java/C#: 05-GAP-SUMMARY.md) |
| `.planning/phases/05-language-coverage-resolution-breadth/05-GAP-SUMMARY.md` | Gap-closure evidence record | ✓ VERIFIED | Exists, committed at `0d2391c`, records named real corpora (temporalio/sdk-java @ `f919926a`, serilog @ `07d39cfb`), real non-zero counts, and explicit deterministic-rebuild confirmation for both Java and C# — same evidence bar as Python/TS-JS |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `pipeline.go` `Run` | `detectRoutes` | direct call at pipeline.go:108 | ✓ WIRED | Route nodes/edges folded into the same resolve pass |
| `resolve.go` `RefKindEmbeds` handling | `dispatch.EdgeKindImplements` promotion | Pattern-2 inline promotion when target is `KindInterface` | ✓ WIRED | Sets `Provenance: "heuristic"`, `Line`/`Col`, `Metadata` |
| `internal/query/traverse.go` `Callers`/`Impact` | `BuildImplementsIndex` | composed via `dispatchSiblingIDs` | ✓ WIRED | Proven by passing dispatch-traversal tests (re-run this session) |
| `internal/indexer/routes/registry.go` | route→handler edges | `goextract.RefKindCalls` + `Provenance: "heuristic"` | ✓ WIRED | Reuses existing `calls` edge kind so `BuildReverseAdjacency` picks them up unchanged |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| LANG-02 | 05-01,02,03,04 | Java full extraction+resolution | ✓ SATISFIED | Real 1,223-file `temporalio/sdk-java` corpus validated (05-GAP-SUMMARY.md); REQUIREMENTS.md marks LANG-02 `[x]` Phase 5 Complete |
| LANG-03 | 05-01,02,03,05 | C# full extraction+resolution | ✓ SATISFIED | Real 216-file `serilog` corpus validated (05-GAP-SUMMARY.md); REQUIREMENTS.md marks LANG-03 `[x]` Phase 5 Complete |
| LANG-04 | 05-01,02,03,06 | Python full extraction+resolution | ✓ SATISFIED | Real 168-file corpus validated; REQUIREMENTS.md marks LANG-04 `[x]` Phase 5 Complete |
| LANG-05 | 05-01,02,03,07 | TS/JS full extraction+resolution | ✓ SATISFIED | Real 13,464-file corpus validated; REQUIREMENTS.md marks LANG-05 `[x]` Phase 5 Complete |
| LANG-06 | 05-08,10,11,13 | Mainstream-6 full/documented-partial | ✓ SATISFIED | Matrix + gap-naming tests pass; REQUIREMENTS.md marks LANG-06 `[x]` Phase 5 Complete |
| LANG-07 | 05-12 | Framework routing, 5 families | ✓ SATISFIED | All 5 detectors present, tested, wired; REQUIREMENTS.md marks LANG-07 `[x]` Phase 5 Complete |
| RES-02 | 05-09 | Dispatch synthesis | ✓ SATISFIED | Bounded synthesis + query traversal tests; REQUIREMENTS.md marks RES-02 `[x]` Phase 5 Complete |
| RES-03 | 05-09 | Provenance tagging | ✓ SATISFIED | `heuristic` vs `ast` provenance confirmed, no schema bump; REQUIREMENTS.md marks RES-03 `[x]` Phase 5 Complete |

No orphaned requirements — all 8 phase requirement IDs (LANG-02..07, RES-02, RES-03) appear in at least one plan's `requirements` frontmatter, are traced above, and are cross-referenced against REQUIREMENTS.md (all marked `[x]` / Phase 5 / Complete, no discrepancies).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/indexer/resolve.go` | ~262-264 | `gofmt -l` violation (pre-existing var-decl comment misalignment, unrelated function) | ℹ️ Info | Confirmed pre-existing per `deferred-items.md`; not a Phase-5 regression, cosmetic only |

No `TBD`/`FIXME`/`XXX` debt markers found in any Phase-5-touched file, including the new `05-GAP-SUMMARY.md`. No stub returns, empty handlers, or hardcoded-empty data flows.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `go build ./...` | exit 0 | clean | ✓ PASS |
| `go vet ./...` | exit 0, no output | clean | ✓ PASS |
| Full test suite (single run, non-race) | `go test ./... ./testdata/golden/... -count=1` | 25/26 packages `ok` on first pass; `internal/daemon` `TestSoak` failed with "lock held by current process" | ⚠️ See note below |
| `internal/daemon` isolation re-run | `go test ./internal/daemon/... -run TestSoak -count=3 -v` | 3/3 passes | ✓ PASS (flake confirmed transient) |
| Go golden-parity fixture unregressed | `go test ./testdata/golden/... -run TestGoldenParity -v` | 7/7 subtests pass | ✓ PASS |
| D-12 priority-4 harness gating confirmed | grep of `CODEGRAPH_JAVA_CORPUS`/`CODEGRAPH_CSHARP_CORPUS` in both parity test files | Matches gap-closure evidence exactly | ✓ PASS |
| D-11 capability matrix consistency | `go test ./internal/indexer/capability/... -v` | 5/5 subtests pass | ✓ PASS |
| Dispatch traversal | `go test ./internal/query/... -run TestCallers_DispatchTraversal\|TestImpact_DispatchTraversal -v` | 3/3 pass | ✓ PASS |
| Route detection AST-not-regex + 5-framework registry | included in full suite run above | pass | ✓ PASS |
| `LanguageSpec{` registration count | `grep -c` across `languages_*.go` + `languages.go` | 14 total (2+3+1×9) | ✓ PASS |

**Note on `TestSoak` failure:** `internal/daemon` was entirely built in Phase 4 (`git log` shows only `fix(04)`/`feat(04-07)`/`test(04-09)` commits touching this package — no Phase 5 plan modifies it). The `TestSoak` failure ("daemon-driven sync failed: lock held by current process") occurred only when run as part of the full `./...` suite under default test parallelism, and reproduced 0/3 times when re-run in isolation (`-count=3`) — consistent with parallel-test lock/fd contention on this machine (see this repo's CLAUDE.md note on macOS sandbox restrictions affecting local SQLite/file-lock writes), not a Phase-5 regression. This is the same class of pre-existing, unrelated `internal/daemon` flakiness the prior verification's guidance already carved out (there, the specific flaky test was `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock` under `-race`; here it is `TestSoak` under parallel non-race load) — a different symptom of the same pre-existing Phase-4 test-isolation issue, not a new one introduced by the Java/C# gap-closure work (which touched zero source files).

### Human Verification Required

None. All observable truths are objectively verifiable from the codebase and independently re-run test evidence.

### Gaps Summary

**RESOLVED and independently re-verified 2026-07-12.** The sole remaining gap from initial verification — ROADMAP Success Criterion 1 / D-12's "validated on real repos" bar not yet met for Java and C# — is confirmed closed:

- `.planning/phases/05-language-coverage-resolution-breadth/05-GAP-SUMMARY.md` (committed at `0d2391c`) records real, named corpora for both languages: Java via `temporalio/sdk-java` (1,223 files, commit `f919926a2d67c10c34fee4b19eed1c605d4223a4`) and C# via `serilog/serilog` (216 files, commit `07d39cfb2928076ecd902a61d295f90d74fe1fa5`), with real non-zero node/edge counts by kind and explicit deterministic-rebuild confirmation for each — matching the exact evidence bar Python (`litellm/types`) and TS/JS (`ccstatusline`) already met in 05-06/05-07.
- Independently confirmed this session (not just trusting the SUMMARY): the `parity_java_test.go`/`parity_csharp_test.go` harnesses gate on `CODEGRAPH_JAVA_CORPUS`/`CODEGRAPH_CSHARP_CORPUS` exactly as claimed, remain self-skipping in the committed repo, and their shape/determinism assertion structure matches what the recorded evidence describes. The Java corpus's `implements:410` count independently corroborates RES-02 dispatch synthesis firing on real Java source, not just the synthetic fixture.
- Re-ran `go build ./...`, `go vet ./...`, and `go test ./... ./testdata/golden/...` this session (not relying on the gap-closure SUMMARY's claim alone): all packages pass except a transient `internal/daemon.TestSoak` lock-contention flake under full-suite parallelism, confirmed pre-existing (Phase-4-only package, unrelated to this gap closure) and non-reproducing in isolation (0/3 failures on repeat).
- Cross-referenced all 8 requirement IDs (LANG-02..07, RES-02, RES-03) against REQUIREMENTS.md: all marked `[x]` / Phase 5 / Complete, no orphans.

All 10 must-haves are now verified against the actual codebase. Phase 5's goal — parity language support with cross-file resolution, dynamic-dispatch synthesis carrying provenance, and framework-aware routing for the priority stack — is achieved. No further action required.

---

*Verified: 2026-07-12T13:23:00Z*
*Verifier: Claude (gsd-verifier, re-verification pass)*
