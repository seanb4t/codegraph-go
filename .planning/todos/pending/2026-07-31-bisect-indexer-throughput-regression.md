---
created: 2026-07-31T16:54:13.986Z
title: Bisect the ~10.6% indexer throughput regression
area: perf
severity: major
files:
  - tools/bench/baseline.json
  - tools/bench/runner
  - .github/workflows/ci.yml (perf regression gate job, PERF-02/INDX-06)
---

## Problem

The CI perf-regression gate (`PERF-02, INDX-06`) reports indexer throughput ~10.6% below `tools/bench/baseline.json`. **This is a real, stable regression — not runner noise.** That distinction was got wrong twice before it was settled, so the evidence is recorded here in full.

Three measurements, all linux/amd64, corpus `synthetic-seed42-count120000`, baseline `12816.38` files/s, budget 10.0% (threshold `11534.74`):

| commit | files/s | vs baseline | verdict |
|---|---|---|---|
| `a1c298f` | 11374.57 | −11.25% | RED |
| `3738acc` | 11642.18 | −9.16% | GREEN |
| `dcf8580` | 11361.67 | −11.35% | RED |

Mean = `11459.47` = **−10.59%**. Band = 11361.67…11642.18 (spread ~2.4%).

**Why this is a real regression and not noise:** the two RED measurements agree to within **13 files/s (0.1%)**, on different runners days apart. Independent noise scatters; it does not reproduce to 0.1%. The distribution is stable and centered at −10.6%, and the 10.0% threshold cuts straight through it. So the *regression* is real and consistent; only the *pass/fail verdict* oscillates. The single green run was the favourable tail, not the truth emerging.

Earlier readings that were **wrong** and should not be repeated:
- `9za63pt543` hypothesised "runner generation drift" (marginal 1.2pp over).
- `gnews1m4pb` then upgraded that to a conclusion ("runner drift CORROBORATED") on the strength of a red→green transition — an inference made on **two** data points. The third point refuted it. Superseded by `n9mmwpshjb`.

Methodological lesson worth keeping: two points that disagree look like noise; three points reveal a distribution. Never conclude "flaky" from a single red→green transition on a continuous metric — take a third sample and look at the **spread**. Tight cluster near a threshold ⇒ the metric is stable and the threshold is the problem. Wide scatter ⇒ the metric is noisy.

**Scope of the cause:** the regression predates the entire 09-07 triage session — `a1c298f` already showed −11.25%, before the grpc bump, the daemon fix, and the test-fixture changes. None of those are plausible causes. The cause lies somewhere between whenever `baseline.json` was recorded and `a1c298f`.

Other metrics from the `dcf8580` run, for comparison against whatever the baseline run recorded:
`peak_rss_bytes` 909582336 · `query_latency_median_ms` 280.63 · `cold_start_ms` 13.261

## Solution

Bisect indexer throughput across the range from the `baseline.json` recording commit to `a1c298f`, using `go run ./tools/bench/runner -mode regression -baseline tools/bench/baseline.json -ceiling-bytes 4294967296`.

**HARD CONSTRAINT — do NOT re-baseline.** Moving `tools/bench/baseline.json` would encode a real, unexplained ~10.6% regression as the new normal and permanently destroy the signal. The baseline may move only *after* someone understands what got slower and accepts it as a deliberate trade. This constraint was also enforced during the 09-07 triage: `tools/bench/baseline.json` was added to the forbidden-files list of the govulncheck quick task specifically so the perf gate could not be "fixed" under cover of an unrelated change.

**Separate, do not conflate:** the regression gate is **single-sample**, while PERF-01 was ratified as **median-of-3** (see `8sa948y0g4`) precisely because single-sample throughput is too noisy to compare. Two perf mechanisms in the same repo with different rigor. Converting the gate to median-of-N would stop the verdict oscillating — but it does **not** explain the regression, and at a −10.6% mean against a 10.0% budget a median-of-3 would most likely still fail, just consistently rather than 2-runs-in-3. Fix the measurement methodology and find the regression as two separate pieces of work; do not let the first be mistaken for the second.

Refs: engram `n9mmwpshjb` (supersedes the drift conclusion), `8sa948y0g4` (PERF-01 median-of-3), ci run 30646401249.
