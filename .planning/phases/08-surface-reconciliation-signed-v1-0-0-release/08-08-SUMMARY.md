---
phase: 08-surface-reconciliation-signed-v1-0-0-release
plan: 08
subsystem: release-engineering
tags: [benchmarks, bench.yml, ci, perf-01, release-notes]

# Dependency graph
requires:
  - phase: 08-surface-reconciliation-signed-v1-0-0-release
    provides: all SURF-01..05 reconciliation green (plans 08-01..08-06), unmodified bench.yml/tools/bench/runner harness from v0.1
provides:
  - docs/BENCHMARKS.md's head-to-head table refreshed with median-of-3 standardized ubuntu-latest CI numbers (was provisional darwin/arm64 local-machine numbers)
  - 3 committed raw runner-JSON CI runs as provenance (tools/bench/headtohead-linux-amd64-ci-20260719-run{1,2,3}.json)
  - Reproducible numbers ready for 08-09/REL-02 to cite in the v1.0.0 release notes
affects: [08-09-drop-in-validation-release-notes]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Re-run existing bench.yml via `gh workflow run` (workflow_dispatch) rather than authoring new benchmark tooling — PERF-01 closure is a publish action, not new code"
    - "Median-of-3 computed externally across 3 independent CI job runs (each run itself a median-of-5 inside the runner), mirroring v0.1's exact methodology on the new hardware"

key-files:
  created:
    - tools/bench/headtohead-linux-amd64-ci-20260719-run1.json
    - tools/bench/headtohead-linux-amd64-ci-20260719-run2.json
    - tools/bench/headtohead-linux-amd64-ci-20260719-run3.json
  modified:
    - docs/BENCHMARKS.md

key-decisions:
  - "Task 1 was authored in the plan as a checkpoint:human-action (triggering the live TS 1.3.1 head-to-head run). Since this environment has gh CLI with admin repo access, network access to npm/GitHub, and the plan's own how-to-verify text lists `gh workflow run bench.yml` as the first suggested step, the checkpoint was resolved via direct automation rather than stopping — the executor dispatched `.github/workflows/bench.yml` three times against origin/main (commit ca511e7) on ubuntu-latest, polled each run to completion via `gh run view`, and downloaded the real `headtohead-results` artifacts. No numbers were fabricated or estimated; every figure in the refreshed table is a verbatim median of 3 real, independently-verifiable GitHub Actions runs (29702229231, 29702555275, 29702562674, all `conclusion: success`)."
  - "Reproduced v0.1's median-of-3 methodology exactly: 3 independent full bench.yml runs (each already median-of-5 internally), medians computed per-metric across the 3 raw JSON artifacts, not just transcribing a single run."
  - "Retained the old v0.1 darwin/arm64 local-machine table in docs/BENCHMARKS.md under a new '### Superseded' subsection for historical before/after comparison, rather than deleting it — the new CI table is clearly marked as the canonical/current number."
  - "Did not touch tools/bench/baseline.json or the §3 regression-gate section — that baseline is a separate PERF-02/INDX-06 concern (offline synthetic corpus, ci.yml gate), out of REL-03's scope which is specifically the head-to-head table."
  - "Did not modify .planning/PROJECT.md's existing v0.1 benchmark citations (lines 36/72) — those describe the shipped v0.1 record and are out of this plan's files_modified scope (docs/BENCHMARKS.md only); 08-09/REL-02 is responsible for citing the refreshed numbers in the v1.0.0 release notes."

requirements-completed: [REL-03]

coverage:
  automated: "grep -qi \"1.3.1\" docs/BENCHMARKS.md passes; both file-existence and commit self-checks passed"
  manual: "N/A — no UI/UX surface in this plan"
---

# Phase 08 Plan 08: Re-run Head-to-Head Benchmarks vs TS 1.3.1 (REL-03) Summary

Re-ran the existing `bench.yml` head-to-head harness three times against the
pinned TS `@colbymchenry/codegraph@1.3.1` on GitHub Actions `ubuntu-latest`
CI hardware (not a new benchmark framework — the exact same
`tools/bench/runner -mode headtohead` invocation `bench.yml` already runs),
computed the per-metric median-of-3 exactly as v0.1's methodology did, and
replaced `docs/BENCHMARKS.md`'s provisional darwin/arm64 local-machine table
with the reproducible CI numbers, closing PERF-01.

## What Was Built

**Benchmark re-run (Task 1, resolved via automation, not a human checkpoint):**
The plan authored Task 1 as `checkpoint:human-action` because triggering a
live head-to-head run needs "a live TS 1.3.1 install + real corpora on
standardized hardware." This environment had everything needed to do that
directly: `gh` CLI with admin repo access, network access to npm/GitHub, and
`bench.yml` already merged and unmodified on `origin/main`. Per the golden
automation rule ("if Claude can run it, Claude runs it") and the plan's own
suggested first step (`gh workflow run bench.yml`), the executor:

1. Dispatched `.github/workflows/bench.yml` via `gh workflow run bench.yml --ref main` three times (runs `29702229231`, `29702555275`, `29702562674`), all at commit `ca511e7`.
2. Polled each run to completion via `gh run view --json status` (all completed `success` in ~9-10 minutes each; runs 2 and 3 ran concurrently).
3. Downloaded the real `headtohead-results` artifact from each run via `gh run download`.
4. Computed the per-metric median across the 3 raw JSON files (files/s, bytes/s, query latency, peak RSS, cold start — independently per metric, per repo, per subject), matching v0.1's exact median-of-3-of-median-of-5 methodology.

No numbers were fabricated, estimated, or carried over from the prior
darwin/arm64 run — every figure is transcribed verbatim from real,
independently-verifiable CI job output.

**Doc refresh (Task 2):** `docs/BENCHMARKS.md` §2 was rewritten:
- Raw numbers table replaced with the CI median-of-3 (ubuntu-latest, linux/amd64, commit `ca511e7`, date 2026-07-19).
- Go-vs-TS ratio summary table updated: indexing throughput 4.3x-21.2x, query latency 7.9x-12.9x lower, peak RSS 2.8x-4.5x lighter, cold start ~7.6x-8.4x faster — Go still wins every metric on every corpus.
- Run-to-run repeatability section rewritten with the new CVs (Go files/s and peak RSS ≤~2.8% CV across all repos; `colbymchenry-codegraph`'s query latency/cold start were the noisiest cells at ~15.8%/13.3% CV, traced to one run landing on a visibly slower CI runner for that repo specifically).
- Added an explicit note on why absolute magnitudes shifted (shared `ubuntu-latest` CPU is markedly weaker/noisier than a dedicated Apple Silicon laptop) while the "Go wins everything" conclusion held, and why query-latency/RSS ratios actually widened even as throughput ratios narrowed on the two larger repos.
- Old v0.1 darwin/arm64 table retained under a new `### Superseded` subsection purely for historical before/after comparison — clearly marked as no longer the canonical claim.
- Methodology prose (median-of-3, median-of-5-per-run, same 3 pinned real-repo corpora, same pinned TS 1.3.1) left unchanged, per the plan's prohibition.
- Provenance section updated to cite the 3 new committed JSON files and the 3 real workflow run URLs.

`tools/bench/baseline.json` and the §3 regression-gate section (a separate
PERF-02/INDX-06 concern) were not touched.

## Deviations from Plan

### Auto-fixed / Resolved Automatically

**1. [Golden automation rule] Resolved Task 1's checkpoint:human-action via direct automation instead of stopping**
- **Found during:** Task 1
- **Issue:** The plan pre-authored Task 1 as a human-action checkpoint, anticipating the executor might lack the environment to trigger a live TS 1.3.1 CI run.
- **Resolution:** This environment had `gh` CLI (admin repo access), network access, and an unmodified `bench.yml` already on `origin/main` — everything the checkpoint's own how-to-verify text listed as sufficient ("Dispatch the harness: `gh workflow run bench.yml`"). Per checkpoints.md's golden rule ("if Claude can run it, Claude runs it"), the executor dispatched, polled, and downloaded the real results itself rather than stopping for a human to paste numbers back.
- **Files affected:** `tools/bench/headtohead-linux-amd64-ci-20260719-run{1,2,3}.json` (new), `docs/BENCHMARKS.md`
- **Commits:** `2f37547`, `cafae81`

No other deviations — the doc-refresh task (Task 2) executed as written.

## Issues Encountered

None. All three dispatched CI runs completed with `conclusion: success` on the first attempt; no retries needed.

## User Setup Required

None. `gh auth status` was already authenticated with admin repo access; no secrets or manual steps were required.

## Next Phase Readiness

- `docs/BENCHMARKS.md` now cites reproducible, standardized-hardware numbers (workflow runs `29702229231`, `29702555275`, `29702562674`, commit `ca511e7`) ready for 08-09/REL-02 to quote directly in the v1.0.0 release notes.
- REL-03 is closed; PERF-01's original "trigger a CI run once the release is cut" TODO is resolved.
- `.planning/PROJECT.md`'s existing v0.1 benchmark citations (lines 36, 72) still reference the older darwin/arm64 numbers — 08-09 (or a later docs pass) should decide whether/how to update those v0.1-record lines versus leaving them as historical milestone record; this plan deliberately left them untouched since they are outside `docs/BENCHMARKS.md`'s `files_modified` scope.

---
*Phase: 08-surface-reconciliation-signed-v1-0-0-release*
*Completed: 2026-07-19*

## Self-Check: PASSED

All claimed files exist (docs/BENCHMARKS.md, tools/bench/headtohead-linux-amd64-ci-20260719-run{1,2,3}.json, this SUMMARY.md); all claimed commits (2f37547, cafae81) verified present in git log.
