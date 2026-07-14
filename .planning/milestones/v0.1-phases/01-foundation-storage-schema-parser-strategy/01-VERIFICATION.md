---
phase: 01-foundation-storage-schema-parser-strategy
verified: 2026-07-10T21:42:29Z
status: passed
score: 4/4 must-haves verified
behavior_unverified: 0
overrides_applied: 0
---

# Phase 1: Foundation — Storage, Schema & Parser Strategy Verification Report

**Phase Goal:** A schema-versioned graph store with a clean interface boundary and a chosen, benchmarked parser approach — the substrate every extractor, query, and migration builds on.
**Verified:** 2026-07-10T21:42:29Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | All graph reads/writes go through a `GraphStore` interface; concurrent lock-free readers alongside a single coordinated writer, no package bypasses (INDX-05) | ✓ VERIFIED | `CGO_ENABLED=1 go test ./internal/graphstore/... -race -run TestConcurrentReadersSingleWriter -v` → PASS, no race. `go test ./internal/graphstore/archtest/... -run TestNoPackageBypassesGraphStore -v` → PASS. Independent grep confirms `cockroachdb/pebble` is imported ONLY by `internal/graphstore/{pebble_store,batch,export}.go` and the archtest file itself (which references the string, not the import). |
| 2 | On-disk format is schema-versioned and round-trips future annotation fields AND bulk export without a format break (ARCH-01) | ✓ VERIFIED | `go test ./internal/schema/... -run TestSchemaRoundTripsUnknownFields -v` → PASS. `go test ./internal/graphstore/... -run TestBulkExportReimportsLosslessly -v` → PASS. `internal/schema/graph.proto` contains `reserved 50 to 59;` on both Node and Edge with an explicit future-annotation comment. `internal/schema/meta.go` exports `SchemaVersion uint32 = 1` with additive-only discipline documented. |
| 3 | Parser strategy (CGo vs wazero) selected from a head-to-head spike with parse throughput + static-build impact documented | ✓ VERIFIED | `PARSER-DECISION.md` (repo root) exists, is substantive (262 lines), and records: measured full-parse throughput (CGo ~1.77x faster than wazero on both Go and Python), incremental-reparse numbers with an honestly-reported caveat (vendored wazero WASM binary lacks `ts_tree_edit`/`ts_tree_get_changed_ranges`, making its "incremental" numbers non-comparable), `CGO_ENABLED=0` static-build result (CGo arm fails to build, wazero arm succeeds), crash-isolation results (no crash observed in either arm under truncated/garbage/deep-nesting/oversized adversarial input, bounded by the shared 4 MiB `MaxSourceBytes` ceiling), and a ratified decision (Option A / CGo) dated 2026-07-10. Per task instructions, the wazero spike arm (`internal/parser/wazero/`) and `tools/spike/` were deliberately deleted after ratification — confirmed absent from the working tree — and this is documented in PARSER-DECISION.md's "Disposition of Spike Artifacts" section, not treated as a gap. |
| 4 | Golden-output corpus + TS `.codegraph/` SQLite DDL captured from live TS v1.3.1 | ✓ VERIFIED | `testdata/golden/ts-schema.sql` non-empty, contains `CREATE TABLE` DDL. `testdata/golden/ts-version.txt` records `codegraph_version=1.3.1` + capture date. `testdata/golden/corpus/{weft-go,colbymchenry-codegraph}/` each contain 7 JSON fixtures (explore/query/callees/callers/impact/node/status). `go test ./testdata/golden/... -run TestGoldenFixturesExist -v` → PASS (all 4 subtests green, including the determinism-stripping guard). Independent grep for `"score"`/`_at"`/`At"` keys across all corpus JSON returned zero matches — determinism-stripping confirmed directly, not just via the test's own logic. |

**Score:** 4/4 truths verified (0 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `go.mod` | pebble/v2 (not v1), all Phase 1 deps pinned | ✓ VERIFIED | `github.com/cockroachdb/pebble/v2 v2.1.6` present; bare `pebble` path absent. `wazero` require removed post-ratification (expected). |
| `internal/graphstore/store.go` | `GraphStore`/`Reader`/`Writer` interfaces | ✓ VERIFIED | Declares `GraphStore` with `Snapshot`, `NewWriter`, `Export`; 106 lines, substantive. |
| `internal/graphstore/pebble_store.go` | sole `*pebble.DB` holder | ✓ VERIFIED | Imports `cockroachdb/pebble/v2`; 165 lines. |
| `internal/graphstore/keys.go` | typed key encoders, delimiter-safe | ✓ VERIFIED | 148 lines; prefix constants for meta/node/edge/file/annotation namespaces present. |
| `internal/graphstore/export.go` | bulk export/import | ✓ VERIFIED | 181 lines; `Export`/import round-trip test passes. |
| `internal/graphstore/archtest/import_graph_test.go` | boundary enforcement via go/packages | ✓ VERIFIED | Uses `golang.org/x/tools/go/packages`; passes. |
| `internal/schema/graph.proto` / `graph.pb.go` / `meta.go` | schema-versioned records | ✓ VERIFIED | `reserved` ranges present; `Node`/`Edge`/`File`/`Meta` structs generated; 505-line generated file. |
| `internal/parser/parser.go` | narrow backend-agnostic interface + size ceiling | ✓ VERIFIED | 90 lines; `Parser` interface, `MaxSourceBytes`, `ErrSourceTooLarge`, documented crash-isolation contract. |
| `internal/parser/cgo/parser_cgo.go` | CGo tree-sitter backend, Go+Python | ✓ VERIFIED | 76 lines; `NewGoParser`/`NewPythonParser`, ceiling enforced pre-parse, `Close()` implemented idempotently. |
| `testdata/golden/*` | DDL + golden JSON corpus + smoke test | ✓ VERIFIED | All fixtures present and non-empty; smoke test passes. |
| `PARSER-DECISION.md` | decision record with measured numbers | ✓ VERIFIED | Present at repo root, substantive, ratified. |
| `internal/parser/wazero/`, `tools/spike/` | (spike-only, expected deleted per ratified decision) | N/A — intentionally absent | Confirmed absent from working tree; consistent with PARSER-DECISION.md disposition note. Not flagged as a gap per task instructions. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `go.mod` | `github.com/cockroachdb/pebble/v2` | require directive | ✓ WIRED | `go list -m github.com/cockroachdb/pebble/v2` resolves; bare path does not (confirmed by build success + grep). |
| `internal/graphstore/pebble_store.go` | `internal/schema` | marshals Node/Edge/File/Meta as pebble values | ✓ WIRED | Confirmed by passing `TestBulkExportReimportsLosslessly` (protocmp-compared round trip). |
| `internal/graphstore/pebble_store.go` | `internal/graphstore/keys.go` | all keys built via typed encoders | ✓ WIRED | No raw `[]byte` key construction found outside `keys.go`; store/batch/export use encoder functions. |
| `internal/graphstore/archtest/import_graph_test.go` | `golang.org/x/tools/go/packages` | import-graph load + assertion | ✓ WIRED | Test passes; only `internal/graphstore` (and subpackages) import pebble/v2 per direct grep across the whole module. |
| `tools/spike/bench_test.go` (pre-ratification) | `internal/parser` | both backends benchmarked through shared interface | ✓ WIRED (historical) | Evidenced by PARSER-DECISION.md's measured numbers; harness itself deliberately deleted post-ratification per plan/decision. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full module builds | `CGO_ENABLED=1 go build ./...` | exit 0, no output | ✓ PASS |
| Full module tests (race) | `CGO_ENABLED=1 go test ./... -race -count=1` | 5/5 packages ok (graphstore, archtest, parser, parser/cgo, schema) | ✓ PASS |
| Concurrency test | `go test ./internal/graphstore/... -race -run TestConcurrentReadersSingleWriter -v` | PASS | ✓ PASS |
| Import-graph boundary test | `go test ./internal/graphstore/archtest/... -run TestNoPackageBypassesGraphStore -v` | PASS | ✓ PASS |
| Schema round-trip test | `go test ./internal/schema/... -run TestSchemaRoundTripsUnknownFields -v` | PASS | ✓ PASS |
| Bulk export round-trip test | `go test ./internal/graphstore/... -run TestBulkExportReimportsLosslessly -v` | PASS | ✓ PASS |
| Key-injection + range tests | `go test ./internal/graphstore/... -run 'TestKeyEncodingRejectsDelimiterInjection|TestEdgeKeyRangeScan|TestFileSubgraphRangeDelete' -v` | 5/5 subtests + 2 more tests PASS | ✓ PASS |
| Golden-fixtures smoke test | `go test ./testdata/golden/... -run TestGoldenFixturesExist -v` | 4/4 subtests PASS (not part of `./...` — Go tooling excludes `testdata/` dirs from `./...` by convention; ran explicitly) | ✓ PASS |
| Parser size-ceiling guard, no CGo | `CGO_ENABLED=0 go test ./internal/parser/ -run TestParserRejectsOversizeInput -v` | PASS | ✓ PASS |
| `go vet ./...` | `CGO_ENABLED=1 go vet ./...` | exit 0, no output | ✓ PASS |
| Only graphstore imports pebble | `grep -rl "cockroachdb/pebble" --include="*.go" .` | 4 files, all under `internal/graphstore/` (incl. its own archtest subpackage) | ✓ PASS |
| Golden corpus determinism | `grep -l -E '"score"\|_at"\|At"' testdata/golden/corpus/*/*.json` | no matches | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| INDX-05 | 01-01, 01-05, 01-06 | Concurrent lock-free readers + single coordinated writer behind `GraphStore` | ✓ SATISFIED | `TestConcurrentReadersSingleWriter` (-race clean), `TestNoPackageBypassesGraphStore`, key encoders with injection guard. REQUIREMENTS.md traceability already marks INDX-05 → Phase 1 → Complete, consistent with evidence. |
| ARCH-01 | 01-01, 01-02, 01-06 | Schema-versioned format, round-trips future annotations, bulk export without format break | ✓ SATISFIED | `TestSchemaRoundTripsUnknownFields`, `TestBulkExportReimportsLosslessly`, `reserved` field ranges in `graph.proto`. REQUIREMENTS.md traceability marks ARCH-01 → Phase 1 → Complete, consistent. |

**Note (informational, not a gap):** Plans 01-03/01-07 declare `requirements: [DIST-05]` and Plan 01-04 declares `requirements: [MCP-04]`. REQUIREMENTS.md's traceability table assigns full ownership of DIST-05 to Phase 8 and MCP-04 to Phase 3 (both already checked `[x]` and marked "Complete" in REQUIREMENTS.md, ahead of those phases executing). This phase's plans lay groundwork feeding those later phases (CGo-exception documentation for DIST-05; golden corpus as the parity oracle for MCP-04) rather than fully closing them. This is consistent with the task's explicit scope ("Phase requirement IDs: INDX-05, ARCH-01") and is not treated as an orphaned or blocking requirement for this phase's verification — flagged here only for visibility since REQUIREMENTS.md's status column reads "Complete" for requirements whose owning phases (3, 8) have not yet run.

### Anti-Patterns Found

None. Scanned all Go/proto/shell/markdown files touched by this phase's commits (internal/, testdata/golden/, PARSER-DECISION.md, go.mod) for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` — zero matches. No stub returns, no empty handler bodies, no hardcoded-empty data flowing to output. All artifacts are substantive (76–505 lines per file, no trivial stubs).

### Human Verification Required

None. All four success criteria are backed by automated, independently-re-run tests and direct filesystem/grep evidence. The one criterion (parser spike) that the plan itself marks as requiring human ratification (`checkpoint:human-verify`, Plan 01-07 Task 4) was already ratified per PARSER-DECISION.md's own record ("human-ratified 2026-07-10"), and the decision's reasoning was independently re-checked against the measured numbers in this verification pass (not merely re-read from the SUMMARY).

### Gaps Summary

No gaps. All four ROADMAP success criteria for Phase 1 are independently verified against the actual codebase: the `GraphStore` interface enforces the concurrency and boundary contract (proven, not just present), the schema is versioned and forward-compatible with a lossless bulk-export round trip, the parser strategy was measured head-to-head and ratified with CGo selected (spike artifacts intentionally removed per that same decision), and the TS golden corpus + DDL are captured, committed, and determinism-stripped. `CGO_ENABLED=1 go build ./...` and `go test ./... -race -count=1` are both green.

---

_Verified: 2026-07-10T21:42:29Z_
_Verifier: Claude (gsd-verifier)_
