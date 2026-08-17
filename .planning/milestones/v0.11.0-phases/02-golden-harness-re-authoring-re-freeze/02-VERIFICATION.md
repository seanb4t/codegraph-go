---
phase: 02-golden-harness-re-authoring-re-freeze
verified: 2026-08-14T21:40:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
requirements:
  - CODE-02: VERIFIED
  - FIXT-04: VERIFIED
  - FIXT-05: VERIFIED
  - FIXT-06: VERIFIED
---

# Phase 2: Golden Harness Re-authoring & Re-freeze — Verification Report

**Phase Goal:** The golden suite reads as codegraph-go's own behavioral regression suite — its files, tests and fixtures named for what they assert, its goldens frozen from codegraph-go's own output against the locked corpora, and the origin-driving capture path gone.

**Verified:** 2026-08-14T21:40:00Z
**Status:** PASSED
**Score:** 5/5 must-haves verified

## Goal Achievement

### Success Criteria Verdicts

| # | Criterion | Status | Evidence |
|---|-----------|--------|----------|
| 1 | No comparison framing in golden harness names; `go test ./...` + `go test ./testdata/golden/...` pass (CODE-02) | VERIFIED | `rg "parity" testdata/golden/` returns no hits. Files named `behavioral_*_test.go`. No `TestGoldenParity*` or `TestParity` functions found. `go build ./...` (exit 0), `go test ./testdata/golden/...` (exit 0), `go test ./...` (exit 0) all pass. |
| 2 | Rename and re-freeze are separate diffs: rename changes no golden byte; re-freeze changes no identifier (CODE-02, FIXT-06) | VERIFIED | Commit `6833979` (rename) changes 10 files, 64+64 lines — renames only: `parity_*.go` to `behavioral_*.go`, `TestGoldenParity*` to `TestCorpusBehavior*`, doc strings. No golden corpus file touched. Commits `72b640e`+`cfda0c1`+`5eb5138` (cleanup+move) also change no golden bytes. Commit `f63598f` (re-freeze) changes 28 files — 24 locked + 2 behavioral golden files regenerated, plus `gocapture/main.go:92` one-line `ILogger`->`LogEvent` deviation (user-approved) and `golden_test.go` guard entries. No identifier renames. |
| 3 | `capture.sh`, `mcp-capture.mjs`, `weft-go`/`colbymchenry-codegraph` corpora absent from tree with no references (FIXT-04) | VERIFIED | `capture.sh` and `mcp-capture.mjs` deleted. `testdata/golden/corpus/weft-go/` (14 fixtures) and `colbymchenry-codegraph/` (14 fixtures) deleted. `rg -i "capture\.sh" .` returns no hits. `rg -i "weft|colbymchenry" testdata/golden/` returns only historical data in `ts-schema.dump.sql` (D-08 boundary — preserved TS schema reference, not a corpus reference). |
| 4 | Every behavioral case (overloaded same-named symbols, multi-word queries, weakly-connected cluster, structural-beats-lexical) exercised by named test under `corpus/behavioral/` with case map intact (FIXT-05) | VERIFIED | `corpus/behavioral/CASES.json` contains 4 cases (a-d) with all required cases. `corpus/behavioral/src/` contains `accounts/`, `orders/`, `recovery/`, `ledger/` subdirectories with source files. `TestCorpusBehaviorSynthetic` in `behavioral_test.go` reads CASES.json via `loadBehavioralCases()` and exercises all 4 assertions. Test passes with all 4 subtests. |
| 5 | Every golden produced by Go-side capture path (`testdata/golden/gocapture`) from codegraph-go's output against locked corpora (FIXT-06) | VERIFIED | 24 locked golden files under `testdata/golden/corpus/{hugo,guava,serilog,requests}/` and 2 behavioral goldens under `corpus/behavioral/`. All 26 verified by `TestReFrozenGoldensValid` (26/26) which checks non-empty, 1st-byte `{` marker, parseable `goldenCapture` envelope, non-empty `Output`. Guard enumerates expected set — never globs — and includes non-vacuity check (`expectedTotal == 0` fatal). Golden envelope commands use `gocapture` format (`explore "..." -p hugo`). |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `corpus/behavioral/CASES.json` | Case map with 4 targeted cases | VERIFIED | `a:overloaded-same-named-symbols`, `b:multi-word-query`, `c:test-heavy-weakly-connected-cluster`, `d:structural-beats-lexical` |
| `corpus/behavioral/src/` | Source files for behavioral cases | VERIFIED | Contains `accounts/`, `orders/`, `recovery/`, `ledger/` subdirs with source |
| `testdata/golden/behavioral_test.go` | Renamed behavioral test with CASES.json-driven assertions | VERIFIED | `TestCorpusBehaviorSynthetic` data-drives over loadBehavioralCases() |
| `testdata/golden/golden_test.go` | Byte-identity guard for re-frozen goldens | VERIFIED | `TestReFrozenGoldensValid` enumerates 26 expected goldens, non-vacuous |
| `testdata/golden/gocapture/main.go` | Extended gocapture with locked-corpus + behavioral + MCP capture | VERIFIED | Captures all corpora through temp-then-move; fail-closed mandatory source resolution |
| `testdata/golden/corpus/{hugo,guava,serilog,requests}/` | 24 locked golden files | VERIFIED | 6 per corpus (explore, node, explore-multi, node-multi, explore-mcp, node-mcp) |
| `corpus/behavioral/go-*.json` | 2 behavioral golden files | VERIFIED | go-explore-multi.json, go-node-multi.json |
| `testdata/golden/behavioral_{java,tsjs,csharp,python}_test.go` | Renamed per-language behavior tests | VERIFIED | No parity or comparison framing in names |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `TestCorpusBehaviorSynthetic` | `corpus/behavioral/CASES.json` | `loadBehavioralCases()` reads filepath `../../corpus/behavioral/CASES.json` | WIRED | Test loads JSON, iterates cases, switches on assertion mode |
| `TestReFrozenGoldensValid` | All 26 golden files | `expectedGoCaptures` table enumerates slugs + filenames | WIRED | Not glob-based — explicit enumeration with non-vacuity check |
| Locked corpus tests | `internal/corpora` | `lockedCorpusDir(t, lang)` helper → `Entry.Dir(CorpusRoot())` | WIRED | Fail-loud on missing corpus; never skips |
| `task golden:regen` | `testdata/golden/gocapture` | `go run ./testdata/golden/gocapture` after `corpora:fetch` + `corpora:assert` | WIRED | Single-entrypoint re-freeze with temp-then-move |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Guard enumerates and verifies all 26 goldens | `go test -count=1 ./testdata/golden/... -run TestReFrozenGoldensValid` | 26/26 goldens verified, PASS | PASS |
| All 4 behavioral cases exercise live engine output | `go test -count=1 ./testdata/golden/... -run TestCorpusBehaviorSynthetic` | 4/4 subtests pass | PASS |
| CLI==MCP byte identity for all corpora | `go test -count=1 ./testdata/golden/... -run 'TestExploreCLIMatchesMCP\|TestNodeCLIMatchesMCP'` | All PASS | PASS |
| Locked corpus resolution works for all 5 priority languages | `go test -count=1 ./testdata/golden/... -run TestPriorityLanguagesResolveToLockedCorpus` | PASS | PASS |
| Capability matrix gate passes | `go test -count=1 ./internal/indexer/capability/...` | PASS | PASS |
| All corpora fetch+assert integrity | `task corpora:assert` | 4/4 locked corpora OK | PASS |
| `go build ./...` | Build check | Exit 0 | PASS |
| `go test ./...` | Full test suite | Exit 0 (all packages pass) | PASS |
| `go vet ./...` | Vet check | Exit 0 | PASS |

### Requirements Coverage

| Req | Description | Status | Evidence |
|-----|-------------|--------|----------|
| CODE-02 | Test/fixture identifiers no longer encode comparison framing; `go test ./...` + `go test ./testdata/golden/...` pass | VERIFIED | No `parity_*_test.go`, no `TestGoldenParity*`. All tests pass. |
| FIXT-04 | Capture.sh, mcp-capture.mjs, weft-go/colbymchenry-codegraph removed | VERIFIED | Files deleted from tree. No references in golden harness scope. |
| FIXT-05 | Behavioral corpus survives as `corpus/behavioral/` with CASES.json and all 4 cases | VERIFIED | CASES.json has 4 cases. Source files present. Tests exercise all. |
| FIXT-06 | Every golden re-frozen from codegraph-go's own output | VERIFIED | 26 goldens produced by gocapture against locked corpora + behavioral corpus. Guard verifies all. |

### Anti-Patterns Found

None. No `TBD`, `FIXME`, or `XXX` markers in any Phase 2 source file. No `placeholder` or `coming soon` patterns in golden test code (the "Phase-4 sync placeholder" comments in `behavioral_test.go` are documented gaps referenced by test assertion messages for fields `WorktreeMismatch`/`PendingChanges` which are correctly tested for nil/zero — not stubs in the code under test).

### Deferred Items

None identified. All Phase 2 requirements are satisfied in full.

## Gaps Summary

No gaps found. All 5 success criteria are verified against the codebase.

The serilog deviation (`ILogger` -> `LogEvent` in `gocapture/main.go`) is the only non-golden, non-test-bookkeeping change in the re-freeze diff. It was surfaced at a human-verify checkpoint, user-approved, and is documented in the SUMMARY as a Rule 2 deviation — the C# tree-sitter parser does not resolve `ILogger` as a named symbol, requiring a different multi-symbol. This is an acceptable, documented divergence.

---

_Verified: 2026-08-14T21:40:00Z_
_Verifier: Claude (gsd-verifier)_