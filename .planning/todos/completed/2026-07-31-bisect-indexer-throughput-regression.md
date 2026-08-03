---
created: 2026-07-31T16:54:13.986Z
title: Bisect the ~10.6% indexer throughput regression
area: perf
severity: major
status: refuted
resolved_at: 2026-07-31T00:00:00.000Z
resolved_by_debug: perf-gate-throughput-regress
files:
  - tools/bench/baseline.json
  - tools/bench/runner
  - .github/workflows/ci.yml (perf regression gate job, PERF-02/INDX-06)
---

## Resolution (2026-07-31) — REFUTED. Do not run the bisect.

**There is no code regression.** The premise of this todo is disproved, and
the bisect it asks for must not be run: it would search a range that
contains nothing.

**The control that settles it.** Two self-contained clones at `e7aa091`
(the commit that recorded `baseline.json`) and `dcf8580` (main at the time
of writing), built and measured natively inside one `golang:1.26-bookworm`
linux/arm64 container — identical environment, identical corpus, only the
checked-out commit varied, three trials each:

| commit | files/s (3 trials) | median |
|---|---|---:|
| `e7aa091` | 38504.11 · 38558.08 · 38970.68 | 38558.08 |
| `dcf8580` | 37738.67 · 38837.64 · 38894.07 | 38837.64 |

Current main is **+0.73% FASTER**, in the opposite direction from a
regression and well inside the 1.2–3.0% inter-trial spread seen at both
commits. A ~10.6% slowdown would be unmissable here. It is not there.

**What the gate was actually measuring.** `baseline.json` records
`goos: darwin, goarch: arm64`; the gate job runs on `ubuntu-latest`. Every
measurement the gate ever made was compared against an Apple Silicon
baseline, and `internal/bench.CheckRegression` never read `GOOS`/`GOARCH`,
so it accepted that silently. On this harness the platform gap is not a
tolerance question — the repo's own head-to-head captures measure
darwin/arm64 vs linux/amd64 CI **6.7x–148x** apart on identical code. The
entire −10.6% was a measurement-frame artifact.

**Where this todo's reasoning went wrong.** The argument was: two REDs
agreeing to within 13 files/s (0.1%) on different runners days apart cannot
be noise, therefore a real regression. *The first half is correct.* The
second half does not follow. A fixed systematic offset from a
wrong-platform baseline reproduces to 0.1% exactly as faithfully as a code
regression does — arguably more faithfully, since it has no dependence on
the code at all. What the three points proved was "stable, not noise."
What was then assumed, without a test, was "therefore the code."

**Extending this todo's own methodological lesson** (kept below, and still
correct as far as it goes): three points do reveal a distribution where two
looked like a coin flip, and that was the right upgrade. But the *tightness*
of a distribution only tells you the signal is systematic — it does not
tell you which system produced it. Stability discriminates signal from
noise; it does not discriminate between competing explanations of the
signal. Both "the code got slower" and "the yardstick is from a different
machine" predict a tight cluster at −10.6%, so no amount of resampling the
same comparison could ever separate them. The move that separates them is a
**control**: vary only the thing you are accusing (the commit) and hold
everything else fixed. That took three container runs per commit and
answered in one pass what three CI observations could not answer at all.

Generalised: before bisecting for a cause, confirm that the *measurement*
is capable of detecting that cause. Ask what else, other than the accused,
could produce this exact number — and go and rule it out, rather than
ruling it out by argument.

**What shipped instead** (branch `gsd/v1.0-drop-in-parity-human-ux`):

1. `internal/bench.CheckRegression` now refuses a `GOOS`/`GOARCH` mismatch
   outright, with tests covering both single-field neighbours and the
   matching case (verified by mutation).
2. `.github/workflows/bench.yml` gained a dispatchable `rebless` job that
   records a candidate baseline on `ubuntu-latest` — the gate's own runner
   class — self-checks it with a second independent measurement, and
   publishes it for a human to review and commit. `-rebless` remains the
   sole writer of `baseline.json` (D-05).
3. The gate is now median-of-3 independent measurement sessions
   (`-trials`, default 3) on both the gating and rebless paths.
4. This todo, refuted.

**The HARD "do not re-baseline" constraint below was honoured and still
stands.** `baseline.json` is untouched. It will move only via the new CI
rebless path, in its own reviewed PR — and when it does, it will be
because the cause is now understood, which is exactly the condition the
constraint set.

**Note on this todo's prediction** that "a median-of-3 would most likely
still fail, just consistently rather than 2-runs-in-3": correct, and now
moot. The gate does still fail — but on the platform mismatch, which is
raised before any throughput comparison is attempted, so it now fails for
the true reason instead of a fictitious one.

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
