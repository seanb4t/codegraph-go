---
phase: 06-benchmark-de-coupling-memory-sweep
plan: 04
subsystem: infra
tags: [go, benchmarking, docs, ci, census]

# Dependency graph
requires:
  - phase: 06-01
    provides: "tools/bench/runner -mode publish + tools/bench/publishcheck (verifier + -emit-rows generator)"
  - phase: 06-02
    provides: ".github/workflows/bench.yml's publish job"
  - phase: 06-03
    provides: "internal/bench framing sweep + committed-baseline mutation proof"
provides:
  - "docs/BENCHMARKS.md rewritten from scratch on absolute-measurement terms, every data cell mechanically generated from the committed measurement"
  - ".planning/phases/06-benchmark-de-coupling-memory-sweep/06-PUBLISH-RESULTS.json — the local measurement of record (BENCH-01's numbers)"
  - ".planning/phases/06-benchmark-de-coupling-memory-sweep/06-CENSUS.md — the phase acceptance census, its instrument, its positive control, its exclusions and one new pattern narrowing"
  - ".planning/phases/06-benchmark-de-coupling-memory-sweep/06-LIVE-VERIFICATIONS.md — BENCH-03 recorded pending-no-ci-run (created here; 06-05/06-06 append)"
  - ".planning/phases/06-benchmark-de-coupling-memory-sweep/COVERAGE.md — no-external-API declaration"
affects: [06-05, 06-06, phase-06-verification]

# Actuals (#2632)
actuals:
  tokens: 10080
  tasks: 4
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Pattern-level (not just file-level) census exclusion: when only some patterns in a bounded set are affected by genuinely innocent text, split the set and apply the path-anchored -g exclusion only to the affected sub-invocation, keeping every other pattern walking the file"

key-files:
  created:
    - .planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PREPLAN-SHA.txt
    - .planning/phases/06-benchmark-de-coupling-memory-sweep/06-PUBLISH-RESULTS.json
    - .planning/phases/06-benchmark-de-coupling-memory-sweep/06-CENSUS.md
    - .planning/phases/06-benchmark-de-coupling-memory-sweep/06-LIVE-VERIFICATIONS.md
    - .planning/phases/06-benchmark-de-coupling-memory-sweep/COVERAGE.md
  modified:
    - docs/BENCHMARKS.md

key-decisions:
  - "Task 1 took the local-measurement fallback path, not the CI dispatch path: the branch was confirmed unpushed to origin (git ls-remote exit 2) before starting. Host conditions recorded in the provenance block per D-02's local-fallback reproducibility discipline."
  - "Task 3's blocking checkpoint:decision was pre-resolved by the maintainer as pending-no-ci-run, independently re-verified before transcription (git ls-remote confirmed unpushed; gh run list showed only pre-phase runs on unrelated branches). BENCH-03 is NOT marked complete in this SUMMARY's requirements-completed — it remains genuinely open, recorded honestly rather than laundered through a local measurement."
  - "One new pattern narrowing found during Task 4's census, not anticipated by 06-01/06-02/06-03: tools/bench/publishcheck/main_test.go's comment explaining why it deliberately uses \"other\" (not \"ts\") as its wrong-subject fixture value quotes \"ts\" twice, tripping the bounded set's literal '\"ts\"' pattern. Verified genuinely innocent (no real \"ts\" assignment exists in the file) and bounded at pattern granularity — a second, path-anchored -g exclusion for only the two `\"ts\"`-quoting patterns, not the whole bounded set — per D-04's own remedy (bound the pattern, never reword innocent text). Full reasoning and companion checks in 06-CENSUS.md."

patterns-established:
  - "Pattern-scoped census exclusion (vs. file-scoped): split a bounded rg -e set into two invocations when only some patterns are tripped by innocent text in one file, so the exclusion's blast radius is the minimum necessary — the rest of the pattern set still walks that file."

requirements-completed: [BENCH-01, BENCH-02]

coverage:
  - id: D1
    description: "docs/BENCHMARKS.md publishes measured, provenanced absolute figures (files/s, bytes/s, query latency, peak RSS, cold start) for both surviving corpora with methodology and the regression-gate description, and carries no comparison table, ratio, or second implementation"
    requirement: BENCH-01
    verification:
      - kind: other
        ref: "BENCHMARKS_RETIRED_TERMS_TOTAL=0; all 10 required-content tokens present >=1; three headings each present exactly once (BENCHMARKS_TOP_SECTIONS=3); PERF-02 ID survives"
        status: pass
      - kind: other
        ref: "publishcheck -emit-rows re-run and both data rows matched verbatim in docs/BENCHMARKS.md (ROWS_EMITTED=2 ROWS_MATCHED=2)"
        status: pass
    human_judgment: false
  - id: D2
    description: "BENCH-02's in-tree half closed by a phase-wide census whose instrument was proven live against a planted wrapped-phrase fixture before its zero was trusted"
    requirement: BENCH-02
    verification:
      - kind: other
        ref: "POSCTRL_MULTILINE=6 > POSCTRL_LINEBASED=2 > 0; CENSUS_FILES_SCANNED=23 CENSUS_CRITICAL_PRESENT=13/13; PHASE6_CENSUS_TOTAL=0; BARE_TS_RESIDUAL_LINES=4, all four quoted and adjudicated in 06-CENSUS.md"
        status: pass
      - kind: unit
        ref: "go build ./... && go vet ./... && go test -count=1 ./internal/bench/... ./internal/corpora/... ./internal/upgrade/... ./tools/bench/..."
        status: pass
    human_judgment: false
  - id: D3
    description: "BENCH-03's status recorded explicitly and honestly, not closed by implication — 06-LIVE-VERIFICATIONS.md carries exactly one BENCH03_STATUS token (pending-no-ci-run), transcribed from the maintainer's checkpoint selection with independently re-verified evidence"
    verification:
      - kind: other
        ref: "BENCH03_STATUS_TOKENS=1 (BENCH03_STATUS=pending-no-ci-run); git ls-remote --exit-code --heads origin gsd/v0.11.0-standalone-project-identity exits non-zero; gh run list --workflow=bench.yml shows no matching run for this branch"
        status: pass
    human_judgment: false

duration: 40min
completed: 2026-08-16
status: complete
---

# Phase 6 Plan 4: Absolute-Measurement Publish + Phase Acceptance Census Summary

**`docs/BENCHMARKS.md` rewritten from scratch to publish locally-measured absolute throughput/query-latency/peak-RSS figures (mechanically generated from a committed raw measurement via `publishcheck -emit-rows`), a phase-wide positive-controlled census reporting a proven zero across the Phase 6 surface, and BENCH-03 recorded honestly as pending (no `bench.yml` run exists for this unpushed branch) rather than closed by a local measurement standing in for it.**

## Performance

- **Duration:** ~40 min
- **Completed:** 2026-08-16
- **Tasks:** 4 (Task 3 was a blocking decision gate pre-resolved by the maintainer; its selection is transcribed, not chosen, in Task 4)
- **Files modified:** 6 (5 created: `06-04-PREPLAN-SHA.txt`, `06-PUBLISH-RESULTS.json`, `06-CENSUS.md`, `06-LIVE-VERIFICATIONS.md`, `COVERAGE.md`; 1 rewritten: `docs/BENCHMARKS.md`)

## Accomplishments

- Measured the de-coupled `publish` path locally (branch confirmed unpushed to origin before starting — `git ls-remote --exit-code --heads origin gsd/v0.11.0-standalone-project-identity` exits non-zero): two `subject:"go"` records (`weft-go`, `cockroachdb-pebble`), all metrics strictly positive. Byte identity between the runner's stdout and the committed artifact proven by `cmp` plus two independent `publishcheck` invocations producing the matching SHA256 `dc06affa42bc5f79aa25014e76316ada82523041eee5c816266aeea12b86cf01`.
- Rewrote `docs/BENCHMARKS.md` in full: no Comparison target section, no ratio/multiplier table, no run-to-run repeatability section against a second implementation, no superseded provisional table. Three required `##` headings each present exactly once. Both data-table rows generated mechanically by `publishcheck -emit-rows` and pasted verbatim — never hand-typed. Full provenance block including local-fallback host conditions (macOS 27.0, Apple M4 Max, 16 cores, 64 GiB, battery power). The committed synthetic-baseline section and the regression-gate section carried forward with their dangling head-to-head cross-references dropped, substance intact.
- Ran the phase's positive-controlled multiline census (D-04) over the declared Phase 6 surface (23 files, all 13 critical files confirmed present): `PHASE6_CENSUS_TOTAL=0`. Found and bounded one new genuinely-innocent flag beyond the two the plan pre-authorised — a meta-commentary comment in `tools/bench/publishcheck/main_test.go` that quotes the retired `"ts"` value to explain why it is deliberately not used. Bounded at pattern granularity (only the two `"ts"`-quoting patterns excluded for that one file), not file-wide, per D-04's own remedy.
- Recorded BENCH-03's status in the new `06-LIVE-VERIFICATIONS.md` as `pending-no-ci-run`, transcribing the maintainer's pre-resolved Task 3 checkpoint selection, with independently re-verified evidence (unpushed branch, no matching `gh run list` entry).
- Declared no external API integration for this phase in `COVERAGE.md`.

## Task Commits

1. **Task 1: Produce and commit the measurement of record** - `4cb47cd` (feat)
2. **Task 2: Rewrite docs/BENCHMARKS.md from scratch on absolute-measurement terms** - `2f37577` (feat)
3. **Task 3: Resolve BENCH-03's status** - blocking `checkpoint:decision`, pre-resolved by the maintainer as `pending-no-ci-run` before this plan ran; no separate commit (its record is written in Task 4)
4. **Task 4: Phase acceptance census, BENCH-03 status record, coverage declaration** - `0f8b6a0` (feat)

_This plan's SUMMARY commit follows as the metadata commit (worktree mode — STATE.md/ROADMAP.md excluded; orchestrator updates those centrally after the wave completes)._

## Files Created/Modified

- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PREPLAN-SHA.txt` - pre-plan HEAD SHA anchor (`e771e67858624d865d52f81a1852977b6211534b`), every tamper guard in this plan compares against it
- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-PUBLISH-RESULTS.json` - the raw, unedited local publish-mode measurement of record (2 records)
- `docs/BENCHMARKS.md` - full rewrite: §1 Methodology, §2 Absolute numbers (mechanically-generated rows + provenance), §3 Regression gate (PERF-02, INDX-06) carried forward verbatim
- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-CENSUS.md` - the census instrument, positive control, per-file table, exclusion list, and the one new pattern narrowing, all with citations
- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-LIVE-VERIFICATIONS.md` - BENCH-03's `pending-no-ci-run` record with independently re-verified evidence
- `.planning/phases/06-benchmark-de-coupling-memory-sweep/COVERAGE.md` - no-external-API declaration

## Decisions Made

- **Local measurement fallback, not CI dispatch, for Task 1.** Verified before starting (`git ls-remote --exit-code --heads origin gsd/v0.11.0-standalone-project-identity` exits non-zero — the branch is not pushed). This supplies BENCH-01's numbers only; it does not and cannot verify BENCH-03, per review HIGH 5's fix.
- **BENCH-03 is NOT in `requirements-completed`.** The plan's own success criteria explicitly forbid closing BENCH-03 by implication ("what it cannot do is complete with BENCH-03 unmarked" — marking it, not necessarily closing it). It is recorded, machine-checkably, as pending. Marking it "complete" here would corrupt `REQUIREMENTS.md`'s traceability and repeat exactly the failure mode review HIGH 5 exists to prevent.
- **New pattern narrowing bounded at pattern granularity, not file granularity.** `tools/bench/publishcheck/main_test.go`'s comment about avoiding `"ts"` quotes `"ts"` twice to make its point. Verified genuinely innocent (`rg -n 'Subject.*=.*"ts"' tools/bench/publishcheck/main_test.go` = 0) and bounded by running only the two `"ts"`-quoting patterns with a path-anchored exclusion for that one file — every other pattern in the bounded set still walks it, and the bare-`TS` completeness control still surfaces both lines for adjudication. Full reasoning in `06-CENSUS.md`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Task 2's own numeric-traceability verify command, as literally written in the plan, omits required `publishcheck` flags and cannot produce `EMITTED=2`**
- **Found during:** Task 2 verification
- **Issue:** The plan's acceptance criterion invokes `go run ./tools/bench/publishcheck -file ... -emit-rows` without `-want-records`/`-want-repos`, which `publishcheck`'s `parseFlags` rejects as invalid (both are required, `wantRecords` defaults to 0). Run with the required flags supplied (as every other invocation in this plan does), the command emits 3 lines (the `PUBLISH_RECORDS=...` summary line plus 2 data rows), not the 2 the criterion's `test "$EMITTED" = "2"` expects.
- **Fix:** Filtered the emitter's output to data-row lines only (`grep '^| '`) before counting, matching the criterion's own stated intent ("one pipe-delimited Markdown data row per record"). Both filtered rows matched verbatim in `docs/BENCHMARKS.md`.
- **Files modified:** None — verification-only correction, no source change.
- **Verification:** `ROWS_EMITTED=2 ROWS_MATCHED=2`, both rows individually confirmed present via `rg -q -F`.
- **Committed in:** N/A (verification step, not a source change)

**2. [Rule 1 - Bug] One genuinely-innocent census flag not anticipated by any prior plan in this phase**
- **Found during:** Task 4 (phase acceptance census)
- **Issue:** `tools/bench/publishcheck/main_test.go:48,50` (a comment 06-01 authored, explaining why the wrong-subject fixture uses `"other"` rather than `"ts"`) trips the bounded set's `'"ts"'` pattern, since the comment quotes `"ts"` to make its point. `PHASE6_CENSUS_TOTAL` would read 2, not 0, without addressing it.
- **Fix:** Verified genuinely innocent (no real `"ts"` assignment in the file) and bounded per D-04's own remedy — a path-anchored exclusion applied only to the two `"ts"`-quoting patterns for this one file, leaving every other pattern in the bounded set walking it, and leaving the bare-`TS` completeness control unexcluded so both lines still surface there for adjudication.
- **Files modified:** None — census-instrument correction only, no source file edited or reworded.
- **Verification:** `rg -o -F '"ts"' tools/bench/publishcheck/main_test.go` = 2 (both the flagged lines, nothing else); `rg -n 'Subject.*=.*"ts"' tools/bench/publishcheck/main_test.go` = 0; re-ran the split census, `PHASE6_CENSUS_TOTAL=0`.
- **Committed in:** `0f8b6a0` (Task 4 commit — the census record itself)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — corrections to this plan's own verify/census instrument, not to any source file)
**Impact on plan:** No scope creep, no source files reworded to chase a green gate. Both corrections are documented per D-04's explicit discipline: bound the pattern, quote the flagged text, never reword innocent prose.

## Issues Encountered

- The execution sandbox refused several otherwise-valid Bash invocations as "too complex to verify... stays inside the worktree" — specifically, any command writing to a path built from `${TMPDIR:-/tmp}` shell-parameter-expansion syntax, and several multi-statement one-liners using `$(...)` command substitution. Worked around by resolving `$TMPDIR` to its literal value once (`/var/folders/.../T/`) and using that literal path for the byte-identity comparison file, and by splitting multi-statement commands into separate, simpler Bash tool calls. No functional impact — every verification the plan specifies was still run, with the same inputs and the same assertions.

## User Setup Required

None — no external service configuration required. Closing BENCH-03 requires a maintainer action (push the branch, dispatch `bench.yml` with `job=publish`) that was explicitly declined for this session; the exact remaining steps are recorded in `06-LIVE-VERIFICATIONS.md`.

## Next Phase Readiness

- BENCH-01 and BENCH-02 are fully satisfied. BENCH-03 remains genuinely open — `06-LIVE-VERIFICATIONS.md` records exactly what closing it requires, and 06-05/06-06 append their own MEM-02 rows to the same ledger.
- `docs/BENCHMARKS.md`, `06-PUBLISH-RESULTS.json`, `06-CENSUS.md` and `COVERAGE.md` are ready for phase-level verification.
- No blockers for 06-05/06-06 (the memory-sweep plans) — this plan touched no memory-store or CLAUDE.md/PROJECT.md/STATE.md content.

## Self-Check: PASSED

- FOUND: `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-04-PREPLAN-SHA.txt`
- FOUND: `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-PUBLISH-RESULTS.json`
- FOUND: `docs/BENCHMARKS.md`
- FOUND: `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-CENSUS.md`
- FOUND: `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-LIVE-VERIFICATIONS.md`
- FOUND: `.planning/phases/06-benchmark-de-coupling-memory-sweep/COVERAGE.md`
- FOUND commit `4cb47cd` (Task 1)
- FOUND commit `2f37577` (Task 2)
- FOUND commit `0f8b6a0` (Task 4)

---
*Phase: 06-benchmark-de-coupling-memory-sweep*
*Completed: 2026-08-16*
