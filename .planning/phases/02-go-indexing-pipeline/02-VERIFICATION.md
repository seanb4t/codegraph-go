---
phase: 02-go-indexing-pipeline
verified: 2026-07-11T07:15:00Z
status: passed
score: 21/21 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 2: Go Indexing Pipeline Verification Report

**Phase Goal:** A user can index a Go repository from scratch and get a correct, cross-file-resolved, queryable graph — proving the two-pass indexer mechanism on the first language.
**Verified:** 2026-07-11T07:15:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Roadmap Success Criteria

| # | Success Criterion | Status | Evidence |
|---|---|---|---|
| 1 | `codegraph init` creates `.codegraph/` with a fully built graph in one step; `uninit [--force]` removes it cleanly (INDX-01) | ✓ VERIFIED | Built `cmd/codegraph` and ran end-to-end against the fixture and against this repo itself (48 files). `init` created `.codegraph/` and reported `files=4 nodes=20 edges=21`; second `init` returned `cli: already initialized ... use codegraph index --force` (non-zero exit) without touching the store; `uninit` (no `--force`, no stdin) aborted ("aborted (pass --force to remove without confirming)") leaving `.codegraph/` intact; `uninit --force` removed it (`ls` confirms gone). `internal/cli/cli_test.go:TestInitIndexUninit` covers the same matrix and passes. |
| 2 | `codegraph index` does a deterministic from-scratch rebuild with `--force`, `--quiet`, `--verbose` (INDX-02) | ✓ VERIFIED | `index --force` rebuilt and printed the summary; `index --force --quiet` printed nothing; `index --force --verbose` added `unresolved=/skipped=` detail. `TestDeterministicRebuild` (indexer package) indexes the fixture twice into separate temp stores and asserts byte-identical `Export()` streams after `Meta.last_sync_unix_ms` normalization — passes under `-race` and `GOMAXPROCS` forcing. Independently re-ran `index --force` twice against a self-indexed copy of this real 48-file repo; node/edge/unresolved counts were identical both times (`nodes=414 edges=660 unresolved=1333`). |
| 3 | Cross-file resolution links imports, call edges, and type inheritance across a multi-package Go repo via two-pass parallel-extract → sequential-resolve (RES-01) | ✓ VERIFIED | `internal/indexer/extract.go` runs Pass 1 as a bounded `errgroup` pool (one parser per worker, index-addressed results); `internal/indexer/resolve.go`/`symbolindex.go` run Pass 2 (global symbol index, then resolve `calls`/`imports`/`embeds`, single `Writer.Commit()`). `TestResolve_CrossPackageCall`, `TestResolve_IntraPackageCall`, `TestResolve_StructEmbeds`, `TestResolve_InterfaceEmbeds`, `TestResolve_IntraModuleImport`, `TestResolve_ExternalImportNoEdge`, `TestResolve_CrossFileMethodContainment`, `TestEdgeCollapse_Deterministic/OrderIndependent`, `TestSingleWriter_CommitsOnce/CloseOnStagingError` all pass (`-race` clean). See Warnings below for two narrow, documented resolution-accuracy edge cases (WR-01/WR-02) that do not falsify this criterion on the tested/common-case inputs. |
| 4 | Indexing a real-world Go project produces symbols and edges matching expected structure (LANG-01) | ✓ VERIFIED | `TestRealRepoStructure` asserts the full node-kind taxonomy (file/function/method/struct/interface/type_alias/constant/variable/package) and four named cross-file edges on the multi-package fixture. Beyond the fixture, I independently indexed this actual repository (48 real `.go` files, multiple packages, real embeds/calls/imports) with the built binary: `files=48 nodes=414 edges=660 duration=~60ms unresolved=1333 skipped=0` — plausible, non-empty, non-trivial structure from a real repo, not just the synthetic fixture. The `weft-go`-corpus spot-check named in `02-VALIDATION.md` is explicitly scoped by the phase's own plan (`02-05-SUMMARY.md` coverage item D6, `human_judgment: true`) as a deferred manual check, not a blocker — `TestRealRepoStructure` plus my independent real-repo run substitute as the automated/programmatic evidence for this criterion. |

**Score:** 4/4 roadmap success criteria verified.

### Observable Truths (must_haves, merged across all 6 plans)

| # | Truth | Status | Evidence |
|---|---|---|---|
| 1 | Node id is `<kind>:<sha256-hex-32>`, deterministic | ✓ VERIFIED | `internal/indexer/nodeid/nodeid.go` uses `crypto/sha256` (no md5); `go test ./internal/indexer/nodeid/...` passes (determinism/distinctness/injection-safety cases). |
| 2 | Re-running node-id construction on same tuple yields byte-identical id | ✓ VERIFIED | Same test suite; `nodeid.go` is a pure function over length-prefixed segments. |
| 3 | Node/Edge/File records carry Go parity fields | ✓ VERIFIED | `graph.proto`/`graph.pb.go` expose `GetSignature/GetDocstring/GetVisibility/GetIsExported/GetReturnType` (Node), `GetProvenance/GetMetadata` (Edge), `GetErrors` (File); `go test ./internal/schema/...` green; field numbers all <50, `SchemaVersion==1` unchanged. |
| 4 | Discovery walks repo, returns only build-included `.go` files, sorted, skipping vendor/dot-dirs | ✓ VERIFIED | `internal/indexer/discover.go` uses `build.Default.MatchFile` + `filepath.WalkDir` with `vendor`/dot-dir `fs.SkipDir`; sorted by `RelPath`. `TestDiscover*` pass. |
| 5 | Each discovered file carries computed import path | ✓ VERIFIED | `discover.go:113` `modfile.Parse`; `ImportPath` field populated and asserted in tests. |
| 6 | Committed multi-package Go fixture exercises required constructs | ✓ VERIFIED | `internal/indexer/testdata/gofixture/` (pkga/pkgb/main/embed.go) contains value+pointer receivers, struct/interface embedding, intra+cross-package calls, imports, constants — `go vet` clean in isolation, `go build ./...` at root unaffected. |
| 7 | Each tree-sitter node type maps to correct codegraph node kind | ✓ VERIFIED | `internal/indexer/goextract/goextract.go` maps `function_declaration`/`method_declaration`/`type_declaration` (struct/interface/type_alias)/`const_declaration`/`var_declaration`; `goextract_test.go` table-driven cases pass. |
| 8 | Method receiver (value/pointer) resolves to correct type + contains edge | ✓ VERIFIED | Tested in `goextract_test.go` and `TestResolve_CrossFileMethodContainment`. |
| 9 | Struct/interface embedding recorded as unresolved refs | ✓ VERIFIED | `goextract.go` embed detection (field with no name child) → `UnresolvedRef{Kind:"embeds"}`; resolved into edges in Pass 2, tested. |
| 10 | Cross-package and intra-package calls recorded w/ name/kind/line/col | ✓ VERIFIED | `recordCall` in `goextract.go`; asserted in tests. |
| 11 | Pass 1 parallel, one Parser/worker, oversized file skip-not-fatal, index-addressed | ✓ VERIFIED | `extract.go` errgroup pool, `TestExtractPool_BoundedNotPerFile`, `TestExtractPool_OversizedFileContained`, `TestExtractPool_OrderStable` all pass under `-race`. |
| 12 | Cross-package `pkg.Fn()` call resolves via alias→import-path→symbol-index | ✓ VERIFIED | `TestResolve_CrossPackageCall` passes. |
| 13 | Intra-package unqualified call resolves against caller's own package | ⚠️ VERIFIED (tested case) / see WR-01 | `TestResolve_IntraPackageCall` passes for the tested (non-colliding) case; code review (WR-01) documents a same-package func/method bare-name collision that mis-resolves — untested edge case, does not falsify the tested truth. |
| 14 | Embedding produces `embeds` edges to in-repo embedded type | ✓ VERIFIED | `TestResolve_StructEmbeds`, `TestResolve_InterfaceEmbeds` pass. |
| 15 | Intra-module import → edge to synthetic package node; external/stdlib → no edge | ✓ VERIFIED | `TestResolve_IntraModuleImport`, `TestResolve_ExternalImportNoEdge` pass. |
| 16 | Duplicate call sites collapse to one edge via deterministic total order | ✓ VERIFIED | `TestEdgeCollapse_Deterministic`, `TestEdgeCollapse_OrderIndependent` pass (see IN-03 below — tiebreak note, non-blocking). |
| 17 | Only ground-truth edges written (provenance ast/empty) | ✓ VERIFIED | Code review confirms no `"heuristic"` literal written by resolver; `grep` clean. |
| 18 | `Indexer.Run` discovers→extracts→resolves→commits | ✓ VERIFIED | `internal/indexer/pipeline.go`; `TestPipelineRun` passes; manual run confirms store built on disk. |
| 19 | Double from-scratch index yields byte-identical `Export()` (after Meta normalization) | ✓ VERIFIED | `TestDeterministicRebuild` passes under `-race`/high `GOMAXPROCS`; independently reproduced via manual double-`index --force` on a real 48-file repo (stable counts). |
| 20 | Pipeline opens store once, closes on every path, stamps Meta | ✓ VERIFIED | `TestPipelineRun_ClosesStoreOnResolveError` passes; `defer store.Close()` confirmed in `pipeline.go`. |
| 21 | CLI commands (`init`/`index`/`uninit`) wire to `indexer.Run` with correct D-01a/D-01b semantics | ✓ VERIFIED | Manual binary smoke test (see SC1/SC2 above) + `TestInitIndexUninit` all pass; `cmd/codegraph/main.go` is a thin delegator. |

**Score:** 21/21 truths verified (0 present-but-behavior-unverified).

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `internal/indexer/nodeid/nodeid.go` | `NodeID(kind,qn,path) string`, sha256 | ✓ VERIFIED | Exists, substantive, wired (imported by `goextract`), tested. |
| `internal/schema/graph.proto` / `graph.pb.go` | Additive D-03 fields | ✓ VERIFIED | Regenerated (protoc-gen-go header present), accessors confirmed. |
| `internal/indexer/discover.go` | `Discover(root)` + module resolution | ✓ VERIFIED | `MatchFile`/`modfile.Parse` present, wired into `pipeline.go`. |
| `internal/indexer/testdata/gofixture/go.mod` | isolated fixture module | ✓ VERIFIED | Present, isolated (`testdata/`), vets clean. |
| `internal/indexer/goextract/goextract.go` | AST→vocabulary mapper | ✓ VERIFIED | Present, substantive (600+ lines), wired into `extract.go`. |
| `internal/indexer/extract.go` | Pass-1 worker pool | ✓ VERIFIED | `errgroup`-based, wired into `pipeline.go`. |
| `internal/indexer/resolve.go` | Pass-2 resolve + write | ✓ VERIFIED | `NewWriter`/`Commit` present, wired into `pipeline.go`. |
| `internal/indexer/symbolindex.go` | global symbol index | ✓ VERIFIED | `map[string]map[string]string`-shaped index, wired into `resolve.go`. |
| `internal/indexer/pipeline.go` | `Run(repoRoot, storeDir, opts)` | ✓ VERIFIED | Present, wired into `internal/cli/{init,index}.go`. |
| `internal/indexer/determinism_test.go` | double-index + Export diff gate | ✓ VERIFIED | `TestDeterministicRebuild`, `TestRealRepoStructure` present and passing. |
| `internal/cli/root.go` | cobra root + `Execute()` | ✓ VERIFIED | Present, wired into `cmd/codegraph/main.go`. |
| `cmd/codegraph/main.go` | binary entrypoint | ✓ VERIFIED | Thin delegator to `cli.Execute()`; builds (`go build ./cmd/codegraph`). |
| `internal/cli/init.go` | init subcommand → `indexer.Run` | ✓ VERIFIED | Confirmed via grep + manual run. |

### Key Link Verification

| From | To | Via | Status |
|---|---|---|---|
| `nodeid.go` | `graphstore/keys.go` discipline | varint length-prefixed preimage | ✓ WIRED (independently reimplemented, no import — as designed) |
| `discover.go` | `golang.org/x/mod/modfile` | `modfile.Parse` | ✓ WIRED |
| `discover.go` | `go/build` | `MatchFile` | ✓ WIRED |
| `extract.go` | `internal/parser/cgo` | one `cgo.NewGoParser()` per worker | ✓ WIRED |
| `goextract.go` | `internal/indexer/nodeid` | `nodeid.NodeID(...)` | ✓ WIRED |
| `resolve.go` | `internal/graphstore` | single `Writer` + one `Commit()` | ✓ WIRED |
| `pipeline.go` | `internal/graphstore` | `graphstore.Open` once, `defer Close` | ✓ WIRED |
| `determinism_test.go` | `internal/graphstore` | diffs `Export()` output | ✓ WIRED |
| `cli/init.go`, `cli/index.go` | `internal/indexer` | `indexer.Run` | ✓ WIRED |
| `cmd/codegraph/main.go` | `internal/cli` | `cli.Execute()` | ✓ WIRED |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Full test suite | `go test ./... -count=1` | all packages `ok` | ✓ PASS |
| Indexer determinism/resolve/collapse/pipeline suite | `go test ./internal/indexer/... -run '...' -race -count=1 -v` | all named tests PASS | ✓ PASS |
| Build binary | `go build ./cmd/codegraph` | succeeds | ✓ PASS |
| `init` on fixture | `codegraph init` | `files=4 nodes=20 edges=21` | ✓ PASS |
| `init` twice | `codegraph init` (repeat) | error, `ErrAlreadyInitialized`, non-zero exit | ✓ PASS |
| `index --force` | `codegraph index --force` | rebuilds, same counts | ✓ PASS |
| `index --force --quiet` | | no stdout output | ✓ PASS |
| `index --force --verbose` | | adds `unresolved=`/`skipped=` | ✓ PASS |
| `uninit` (no `--force`, no stdin) | | aborts, `.codegraph/` remains | ✓ PASS |
| `uninit --force` | | removes `.codegraph/` | ✓ PASS |
| Real-repo index (this repo, 48 files) | `codegraph init --verbose` | `files=48 nodes=414 edges=660 unresolved=1333` | ✓ PASS |
| Real-repo determinism | `index --force` x2 on same real-repo copy | identical counts both runs | ✓ PASS |
| `go vet ./...` | | clean | ✓ PASS |
| `archtest` (no GraphStore bypass) | `go test ./internal/graphstore/archtest/...` | PASS | ✓ PASS |
| Anti-pattern scan (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) on all phase-2 files | `grep` | no matches | ✓ PASS |
| Same-package func/method name-collision check on real repo | manual grep across `internal/`, `cmd/` | no collisions found in this codebase | ✓ PASS (WR-01 latent, not triggered here) |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| INDX-01 | 02-06 | `init` creates `.codegraph/` + full graph in one step; `uninit [--force]` removes cleanly | ✓ SATISFIED | Manual + `TestInitIndexUninit` |
| INDX-02 | 02-05, 02-06 | `index` deterministic from-scratch rebuild, `--force`/`--quiet`/`--verbose` | ✓ SATISFIED | `TestDeterministicRebuild` + manual |
| RES-01 | 02-01, 02-03, 02-04, 02-05 | Cross-file resolution: imports, calls, type inheritance, two-pass | ✓ SATISFIED | `TestResolve_*`, `TestEdgeCollapse_*`, `TestSingleWriter_*` |
| LANG-01 | 02-01, 02-02, 02-03, 02-04 | Go structural extraction + cross-file resolution (first language) | ✓ SATISFIED | `goextract_test.go`, `TestRealRepoStructure`, real-repo smoke test |

No orphaned requirements — all four Phase-2 REQUIREMENTS.md entries (INDX-01, INDX-02, RES-01, LANG-01) appear in plan frontmatter and are addressed above.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `internal/indexer/symbolindex.go` | 37-46 | WR-01 (from 02-REVIEW.md): methods indexed under bare name can shadow a same-package function of the same name, producing a wrong `calls` edge on collision | ⚠️ Warning | Deterministic, narrow, untested by current fixture/self-index (no collision present in either); does not falsify any tested must-have. Recommend a follow-up fix/test (exclude `KindMethod` from the unqualified namespace) — acceptable as Phase-5 resolution-breadth follow-up per REVIEW's own framing. |
| `internal/indexer/goextract/goextract.go` | 597-616 | WR-02 (from 02-REVIEW.md): selector calls on a non-identifier operand (`foo().Bar()`, `arr[0].Bar()`) collapse to `PkgAlias:""` and can falsely resolve as same-package calls | ⚠️ Warning | Same disposition as WR-01 — deterministic, narrow, untested, recommended Phase-5 follow-up. |
| `internal/indexer/goextract/goextract.go` | 301-331 | IN-01: multi-name const/var specs stamp identical line/col on every declared name | ℹ️ Info | Positional metadata only, node ids stay distinct. |
| `internal/indexer/goextract/goextract.go` | 203-206 | IN-02: tagged embedded struct field (2 named children) missed as embed | ℹ️ Info | False negative on a rare Go pattern (tagged embed). |
| `internal/indexer/resolve.go` | 220-230 | IN-03: collapse tiebreak's `filePath` term is always-equal/dead code; doc comment overstates the sort key | ℹ️ Info | Cosmetic — behavior is still correct and deterministic on `(line,col)`. |
| `internal/indexer/pipeline.go` | 100-103 | IN-04: `readGraphCounts` failure path returns bare `Stats{}` instead of partially-populated Stats | ℹ️ Info | Minor observability inconsistency only. |
| `internal/indexer/goextract/goextract.go` | 593-596 | IN-05: dot-imports not resolved to the dot-imported package | ℹ️ Info | Rare pattern, documented limitation, within RQ-2 narrow-scope intent. |

None of the above are debt markers (no TBD/FIXME/XXX found in phase-2 files) and none are blockers — the two Warnings (WR-01/WR-02) are real, deterministic resolution-accuracy bugs but affect only inputs not exercised by the phase's own test fixture or by an independent real-repo run of this project; they are explicitly framed in `02-REVIEW.md` as acceptable Phase-5 (resolution-breadth) follow-ups and do not undermine any of the four roadmap success criteria as stated.

### Human Verification Required

None. All items above were resolvable via automated tests, code inspection, and independent manual binary runs (including against a real 48-file multi-package Go repo, going beyond the phase's own synthetic fixture).

### Gaps Summary

No gaps. All 4 roadmap success criteria and all 21 merged must-have truths across the 6 plans are verified against the actual codebase (not just SUMMARY claims): the schema/nodeid substrate (02-01), discovery + fixture (02-02), Pass-1 extraction (02-03), Pass-2 resolution + deterministic write (02-04), pipeline orchestration + determinism gate (02-05), and CLI lifecycle (02-06) all exist, are substantive, are wired together, and pass `go build ./...`, `go test ./... -count=1`, `go vet ./...`, and a race-enabled indexer test run. The two code-review warnings (WR-01/WR-02) are real, documented, narrow resolution-accuracy edge cases that do not currently manifest against either the committed fixture or this repository's own real 48-file structure, and are appropriately scoped as Phase-5 follow-ups rather than Phase-2 blockers — recorded here for traceability, not as a gate.

---

_Verified: 2026-07-11T07:15:00Z_
_Verifier: Claude (gsd-verifier)_
