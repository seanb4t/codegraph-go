---
phase: 06-benchmark-de-coupling-memory-sweep
plan: 01
subsystem: infra
tags: [go, benchmarking, tools-bench, ci-tooling]

requires: []
provides:
  - "tools/bench/runner -mode publish measuring only the Go binary over a two-entry realcorpus"
  - "tools/bench/publishcheck, a pure-Go verifier for publish-mode JSON (reused by 06-04 with -emit-rows)"
  - "tools/bench/realcorpus.Corpora() reduced to weft-go + cockroachdb-pebble"
affects: [06-02, 06-04]

actuals:
  tokens: 12933
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Positive-controlled multiline framing census (rg -U -o against a planted wrapped-phrase fixture, proven to beat a line-based rg before any zero is trusted)"
    - "Pure-Go verifier binary (publishcheck) replacing a would-be Node one-liner for benchmark-output verification"

key-files:
  created:
    - tools/bench/publishcheck/main.go
    - tools/bench/publishcheck/main_test.go
    - .planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PREPLAN-SHA.txt
  modified:
    - tools/bench/runner/main.go
    - tools/bench/runner/main_test.go
    - tools/bench/realcorpus/manifest.go
    - tools/bench/realcorpus/manifest_test.go

key-decisions:
  - "Kept Metrics.Subject (always \"go\" in publish mode) rather than removing it — shared json tag with the committed baseline.json and the regression gate (assumption_delta_decision in the plan)."
  - "Extended the prose sweep beyond the plan's explicit line list: TestCorporaProvenanceComplete's doc comment, the Entry struct doc comment, the unmeasured-warmup comment, and copyTree's doc comment all still named the retired head-to-head/TS-comparison framing and were re-authored in Task 2's census pass."

patterns-established:
  - "Census exclusion companions: when a single file must legitimately retain one instance of a retired literal (e.g. a removal-proof flag-name test), the census excludes that file by full repo-relative glob and pins exactly what it contains with 3 companion counts, never a bare unanchored exclusion."

requirements-completed: [BENCH-02]

coverage:
  - id: D1
    description: "tools/bench/runner has exactly two modes (publish, regression); the two-subject comparison architecture, its -ts-binary flag, resolveTSBinary/macOSHomebrewTSBinary, and runHeadToHead are gone"
    requirement: BENCH-02
    verification:
      - kind: unit
        ref: "tools/bench/runner/main_test.go#TestParseFlags_RejectsRetiredComparisonBinaryFlag"
        status: pass
      - kind: unit
        ref: "tools/bench/runner/main_test.go#TestParseFlags_PublishOverridesApply"
        status: pass
    human_judgment: false
  - id: D2
    description: "realcorpus.Corpora() carries exactly two entries (weft-go, cockroachdb-pebble); both CommitSHA pins byte-unchanged"
    requirement: BENCH-02
    verification:
      - kind: unit
        ref: "tools/bench/realcorpus/manifest_test.go#TestCorporaHasExactlyTwoEntries"
        status: pass
      - kind: unit
        ref: "tools/bench/realcorpus/manifest_test.go#TestReferencesExistingGoldenCorpora"
        status: pass
    human_judgment: false
  - id: D3
    description: "A real -mode publish run over both corpus entries emits two subject:\"go\" records with positive metrics, verified end-to-end by publishcheck (no Node dependency in the verification path)"
    requirement: BENCH-02
    verification:
      - kind: integration
        ref: "go run ./tools/bench/runner -mode publish | go run ./tools/bench/publishcheck -want-records 2 -want-repos weft-go,cockroachdb-pebble"
        status: pass
      - kind: unit
        ref: "tools/bench/publishcheck/main_test.go#TestPublishCheck (7 subtests: pass, short-count, wrong-subject, non-positive-metric, wrong-repo-set, empty-array, SHA256-identity)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Six committed comparison-era headtohead-*.json capture files removed from tracking; git history preserves them"
    requirement: BENCH-02
    verification:
      - kind: other
        ref: "git ls-files 'tools/bench/headtohead-*.json' | wc -l -> 0; git log --oneline -1 -- tools/bench/headtohead-darwin-arm64-20260713-run1.json -> resolves at 2572a67"
        status: pass
    human_judgment: false
  - id: D5
    description: "Surviving prose in tools/bench/realcorpus and tools/bench/runner describes absolute single-subject measurement, citing no second implementation"
    requirement: BENCH-02
    verification:
      - kind: other
        ref: "Positive-controlled multiline census over tools/bench/realcorpus/ and tools/bench/runner/ (excluding the pre-authorised tools/bench/runner/main_test.go) -> BENCH_TRACER_SURFACE_TOTAL=0"
        status: pass
    human_judgment: false

duration: 40min
completed: 2026-08-16
status: complete
---

# Phase 6 Plan 1: Absolute-Only Publish Path Summary

**`tools/bench/runner -mode publish` replaces the two-subject head-to-head comparison with a single-subject Go-only measurement over a two-entry realcorpus (weft-go, cockroachdb-pebble), verified end-to-end by a new pure-Go `tools/bench/publishcheck` binary.**

## Performance

- **Duration:** ~40 min
- **Completed:** 2026-08-16
- **Tasks:** 2
- **Files modified:** 13 (2 created new: publishcheck/main.go, publishcheck/main_test.go; 4 modified: runner/main.go, runner/main_test.go, realcorpus/manifest.go, realcorpus/manifest_test.go; 6 deleted: headtohead-*.json capture files; 1 new phase-record: 06-01-PREPLAN-SHA.txt)

## Accomplishments

- Deleted the entire two-subject comparison architecture from `tools/bench/runner`: `runHeadToHead`, `resolveTSBinary`, `macOSHomebrewTSBinary`, the `config.tsBinary` field, the `-ts-binary` flag, and the `headtohead` mode string are all gone. `runPublish` replaces `runHeadToHead`, calling `measureSubject("go", cfg.goBinary, entry, ...)` once per corpus entry.
- Dropped the `colbymchenry-codegraph` realcorpus entry (its `CODEGRAPH_TSCG_CORPUS` env var, `codegraph-ts` sibling dir, and the test's `tscgPinnedCommit` const) — `Corpora()` now returns exactly `weft-go` and `cockroachdb-pebble`, both `CommitSHA` pins byte-unchanged since the pre-plan SHA.
- Added `tools/bench/publishcheck`, a pure-Go verifier for publish-mode JSON (closes review finding 06-01:246 — no Node runtime anywhere in this plan's verification path). `TestPublishCheck` is the single top-level test symbol, carrying all 7 required positive-control cases (pass, short record count, wrong subject, non-positive metric, wrong repo set, empty array, SHA-256 byte-identity) as subtests.
- Removed the six committed `tools/bench/headtohead-*.json` capture files via `git rm` (D-01) — history preserves them.
- Closed the framing census: a positive-controlled multiline `rg -U` sweep over `tools/bench/realcorpus/` and `tools/bench/runner/`, with the one pre-authorised exclusion (`tools/bench/runner/main_test.go`, which must carry the literal `ts-binary` for the flag-removal test), reports a verbatim `BENCH_TRACER_SURFACE_TOTAL=0`.
- Ran a real end-to-end `-mode publish` invocation: two `subject:"go"` records, all metrics positive, verified byte-for-byte by `publishcheck`.

## Task Commits

1. **Task 1: End-to-end absolute-only publish path — corpus, runner mode, one real run** - `16cfb4b` (feat)
2. **Task 2: Delete the committed comparison-era captures and confirm the swept prose by count** - `29a8a31` (fix)

## Files Created/Modified

- `tools/bench/publishcheck/main.go` - Pure-Go publish-mode JSON verifier: parses `-file` into `[]internal/bench.Metrics`, asserts record count/repo-set/subject/median-of-trials/positivity, prints `PUBLISH_RECORDS=... SHA256=...`, optional `-emit-rows` for 06-04
- `tools/bench/publishcheck/main_test.go` - `TestPublishCheck` (single top-level symbol) with 7 subtests proving the verifier both passes good input and rejects each bad-input class
- `tools/bench/runner/main.go` - `runHeadToHead` -> `runPublish`; `-ts-binary`/`resolveTSBinary`/`macOSHomebrewTSBinary` deleted; mode switch is now `publish`/`regression`; prose re-authored for single-subject measurement (package doc, `measureSubject`, `resolveOrClone`, `copyTree`, the unmeasured-warmup comment, the `defaultTrials` rationale)
- `tools/bench/runner/main_test.go` - `TestParseFlags_OverridesApply` -> `TestParseFlags_PublishOverridesApply`; new `TestParseFlags_RejectsRetiredComparisonBinaryFlag` (the positive removal proof, naming the retired `-ts-binary` flag exactly once); `TestResolveTSBinary_FindsOnPath`/`TestResolveTSBinary_EmptyWhenNotFound` deleted
- `tools/bench/realcorpus/manifest.go` - `colbymchenry-codegraph` entry removed; package/`Entry`/`Corpora`/`CommitSHA` doc comments and the `cockroachdb-pebble` entry's own comment re-authored for absolute single-subject measurement and D-11's redistribution-question framing
- `tools/bench/realcorpus/manifest_test.go` - New `TestCorporaHasExactlyTwoEntries`; `TestReferencesExistingGoldenCorpora` narrowed to the surviving `weft-go` pin only; `TestCorporaProvenanceComplete`'s doc comment re-authored
- `tools/bench/headtohead-darwin-arm64-20260713-run{1,2,3}.json`, `tools/bench/headtohead-linux-amd64-ci-20260719-run{1,2,3}.json` - Deleted (D-01; recoverable via `git show 2572a67`)
- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PREPLAN-SHA.txt` - Phase-record: the pre-plan commit SHA every tamper guard in this plan compares against

## Decisions Made

- Kept `internal/bench.Metrics.Subject` (json tag `subject`, always `"go"` in publish mode) rather than removing it, per the plan's own `assumption_delta_decision`: it's a shared field with the committed `baseline.json` and the regression gate, and removing it would be an out-of-scope schema change.
- Did not touch `internal/bench` or `tools/bench/baseline.json` in this plan (verified via `git diff --quiet <pre-plan-SHA> -- internal/bench/ tools/bench/baseline.json`).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Task 2's own real-census run found retired framing beyond the plan's explicit line list**
- **Found during:** Task 2 (framing census)
- **Issue:** The bounded multiline census over `tools/bench/realcorpus/` and `tools/bench/runner/` returned 4 unexpected hits after Task 1's edits: `TestCorporaProvenanceComplete`'s doc comment ("PERF-01 head-to-head benchmark"), the `Entry` struct's doc comment ("a head-to-head number is always traceable"), the unmeasured-warmup comment in `measureSubject` ("both CLIs' `init`... the TS CLI's `init`"), and `copyTree`'s doc comment ("the TS binary's SQLite store"). None of these lines were in the plan's `<action>` line-number list, but all were genuine retired-framing prose the phase's own D-04 discipline requires swept, not innocent text to bound around.
- **Fix:** Re-authored all four comments to describe the single-subject `publish`-mode reality (no second implementation named), matching the discipline Task 1 already applied elsewhere in the same files.
- **Files modified:** `tools/bench/realcorpus/manifest_test.go`, `tools/bench/realcorpus/manifest.go`, `tools/bench/runner/main.go`
- **Verification:** Re-ran the census after each fix; final run reports `BENCH_TRACER_SURFACE_TOTAL=0`.
- **Committed in:** `29a8a31` (Task 2 commit)

**2. [Rule 1 - Bug] My own doc comment for the removal-proof test repeated the exact test name, making its own census gate unsatisfiable**
- **Found during:** Task 1, running the `RETIRED_RUNNER_SYMBOLS_TOTAL` acceptance gate
- **Issue:** I initially wrote `TestParseFlags_RejectsRetiredComparisonBinaryFlag`'s doc comment starting with its own name (idiomatic Go convention), which made the literal test name occur twice in `main_test.go` (once in the comment, once in the `func` line). Task 1's own acceptance criterion asserts the file must carry that exact name **exactly once** (`REJECTION_TEST_PRESENT=1`).
- **Fix:** Reworded the doc comment to start with "This is the positive removal assertion..." instead of repeating the function name, dropping the count to 1.
- **Files modified:** `tools/bench/runner/main_test.go`
- **Verification:** `rg -o -F 'TestParseFlags_RejectsRetiredComparisonBinaryFlag' tools/bench/runner/main_test.go | wc -l` -> `1`.
- **Committed in:** `16cfb4b` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — bugs in my own first-pass prose that the plan's own gates caught)
**Impact on plan:** No scope creep; both fixes were required to satisfy gates the plan itself specifies. No architectural change.

## Issues Encountered

- A stray `publishcheck` build artifact (Mach-O binary) landed at the repo root from an earlier `go build ./tools/bench/...` invocation without an explicit `-o`; removed before staging, never committed.
- Precondition (`git ls-remote --exit-code --heads https://github.com/cockroachdb/pebble`) passed; no sibling checkouts existed for `weft`/`pebble` next to the worktree, so the end-to-end publish run performed real shallow clones over the network — both completed within the verify timeout.

## Verbatim Verification Totals

- `RETIRED_RUNNER_SYMBOLS_TOTAL=0` (companions: `EXCLUDED_FILE_TS_BINARY_OCCURRENCES=1 REJECTION_TEST_PRESENT=1 EXCLUDED_FILE_OTHER_RETIRED_SYMBOLS=0`)
- `RETIRED_CORPUS_REFS_TOTAL=0`
- `SURVIVING_PIN_DIFF_LINES=0 SURVIVING_PINS_PRESENT=2`
- `GATE_INPUTS_UNCHANGED_SINCE=575e610b0c03db21120c4a139ab1681899a0b681 BASELINE_PRESENT=true`
- Test-surface delta: `TESTS_AFTER=55 REMOVED=3 ADDED=4` — removed set `{TestParseFlags_OverridesApply, TestResolveTSBinary_EmptyWhenNotFound, TestResolveTSBinary_FindsOnPath}`; added set `{TestCorporaHasExactlyTwoEntries, TestParseFlags_PublishOverridesApply, TestParseFlags_RejectsRetiredComparisonBinaryFlag, TestPublishCheck}` — both exact-matched the plan's literal lists.
- `TRACKED_CAPTURES=0 ON_DISK_CAPTURES=0`
- Positive control: `MULTILINE=3 LINEBASED=1` (multiline strictly beats line-based on the planted wrapped-phrase fixture)
- `BENCH_TRACER_SURFACE_TOTAL=0` (companions: `EXCLUDED_FILE_TS_BINARY_OCCURRENCES=1 REJECTION_TEST_PRESENT=1 EXCLUDED_FILE_OTHER_RETIRED_TERMS=0`)
- `NOTICE_PRESENT=true NOTICE_COPYRIGHT_LINES=1 NOTICE_UNCHANGED_SINCE=575e610b0c03db21120c4a139ab1681899a0b681`
- End-to-end run: `PUBLISH_RECORDS=2 REPOS=cockroachdb-pebble,weft-go ALL_SUBJECT_GO=true ALL_METRICS_POSITIVE=true SHA256=84d2ebf8701163c657e6e48c3de28f7642ca64ad95ec20943176d1497994146f`

## Next Phase Readiness

- `tools/bench/runner -mode publish` and `tools/bench/publishcheck` are ready for 06-02 (CI wiring: `.github/workflows/bench.yml`'s `publish` job) and 06-04 (`docs/BENCHMARKS.md`, which reuses `publishcheck -emit-rows` as its single source of published numbers).
- No blockers. `internal/bench` and `tools/bench/baseline.json` are untouched, so 06-03 (regression-mode prose sweep) starts from an unmodified base.

## Self-Check: PASSED

- FOUND: tools/bench/publishcheck/main.go
- FOUND: tools/bench/publishcheck/main_test.go
- FOUND: .planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-PREPLAN-SHA.txt
- FOUND: .planning/phases/06-benchmark-de-coupling-memory-sweep/06-01-SUMMARY.md
- CONFIRMED-DELETED: tools/bench/headtohead-darwin-arm64-20260713-run1.json
- FOUND commit: 16cfb4b
- FOUND commit: 29a8a31
- FOUND commit: 0e92384

---
*Phase: 06-benchmark-de-coupling-memory-sweep*
*Completed: 2026-08-16*
