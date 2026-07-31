---
created: 2026-07-31T00:00:00.000Z
title: Record and commit a linux/amd64 perf baseline via the CI rebless job
area: perf
severity: major
blocks: perf regression gate (PERF-02, INDX-06) is RED until this lands
files:
  - tools/bench/baseline.json
  - .github/workflows/bench.yml (rebless job)
  - tools/bench/BASELINE.md
---

## Problem

The perf regression gate fails on every run, on purpose, and will keep
failing until this todo is done.

`tools/bench/baseline.json` was recorded on `darwin/arm64`. The gate runs
on `ubuntu-latest` (`linux/amd64`). `internal/bench.CheckRegression` used
to compare those two silently, which produced a stable, reproducible,
entirely fictitious ~10.6% "throughput regression" that survived three
rounds of triage before a same-platform control showed the real
commit-to-commit delta is **+0.73%** (see
`.planning/debug/perf-gate-throughput-regress.md` and the refuted todo
`2026-07-31-bisect-indexer-throughput-regression.md`).

`CheckRegression` now refuses that comparison outright:

```
bench: platform mismatch: baseline was measured on darwin/arm64 but this
run is linux/amd64; ...
```

Failing loud is the correct behaviour and was chosen deliberately — it is
strictly better than gating on a number that cannot mean anything. But the
red gate is an accepted interim state, not the end state, and it should not
be left to rot into background noise that people learn to ignore. That is
what this todo exists to prevent.

## Solution

1. Actions → **bench** → *Run workflow* on the branch carrying the fix,
   with `job: rebless` and `trials: 3`.
2. The job records a candidate on `ubuntu-latest`, then takes a second
   independent measurement and gates it against the candidate. If that
   check fails, do **not** commit the candidate — it was a tail draw. Re-run,
   or re-run with `trials: 5`.
3. Read the delta table in the job summary. Expect `platform` to show
   **CHANGED** (darwin/arm64 → linux/amd64) and `files_per_sec` to drop
   roughly 10-11% (12,816 → ~11,400). **That drop is the platform change,
   not a regression** — it is the whole point of this exercise. Confirm
   `corpus` is unchanged (`synthetic-seed42-count120000`).
4. Download the `baseline-candidate` artifact, commit it as
   `tools/bench/baseline.json` in its **own PR**, isolated from unrelated
   changes, per CONTEXT.md D-05.
5. Confirm the gate goes green on that PR, then update the "Status" section
   at the top of `tools/bench/BASELINE.md`, which currently documents the
   committed baseline as invalid.

The workflow holds no write permission and cannot commit the baseline
itself; step 4 is deliberately a human action.

**Do not shortcut this by running `-rebless` locally and committing the
result.** Throughput here is dominated by hardware class, not just
GOOS/GOARCH: one Apple Silicon machine measures 12,816 files/s on darwin
native and ~38,558 in a linux/arm64 container, while the GitHub runner
measures ~11,400 — a ~3x spread on identical code. A locally-reblessed
Linux baseline satisfies the new `GOOS`/`GOARCH` guard and is still the
wrong yardstick, which would reintroduce this exact bug in a form the
guard cannot catch.

**Do not "fix" the red gate by widening `DefaultThroughputTolerance` or by
hand-editing `baseline.json`.** Both re-encode the unexplained delta as the
new normal. `-rebless` is the sole writer by design (D-05).
