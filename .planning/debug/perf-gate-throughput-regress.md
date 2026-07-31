---
status: awaiting_human_verify
trigger: "The CI perf regression gate (PERF-02, INDX-06) has failed on main for multiple commits, reporting ~10.6% indexer throughput below tools/bench/baseline.json. A pending todo asserts this is a real, stable code regression and asks for a bisect from e7aa091 to a1c298f. Corrected premise gathered before starting: baseline.json records darwin/arm64 but the gate runs on ubuntu-latest (linux/amd64), and internal/bench.CheckRegression never validates GOOS/GOARCH."
created: 2026-07-31
updated: 2026-07-31
slug: perf-gate-throughput-regress
---

# Debug: perf regression gate reports ~10.6% throughput regression

## Symptoms

**Expected behavior**
The CI job `perf regression gate (PERF-02, INDX-06)` passes: measured indexer
throughput on the deterministic synthetic corpus stays within the 10.0%
`DefaultThroughputTolerance` budget of `tools/bench/baseline.json`.

**Actual behavior**
The gate fails. Measured throughput sits ~10.6% below the recorded baseline,
with the pass/fail verdict oscillating because the true value straddles the
threshold.

Three measurements, all on `ubuntu-latest` (linux/amd64), corpus
`synthetic-seed42-count120000`, baseline `12816.38` files/s, budget 10.0%
(threshold `11534.74`):

| commit | files/s | vs baseline | verdict |
|---|---|---|---|
| `a1c298f` | 11374.57 | -11.25% | RED |
| `3738acc` | 11642.18 | -9.16% | GREEN |
| `dcf8580` | 11361.67 | -11.35% | RED |

Mean `11459.47` = -10.59%. Band 11361.67..11642.18 (spread ~2.4%).
The two REDs agree to within 13 files/s (0.1%) on different runners days apart.

**Error message**
From `internal/bench/regression.go:43`:

```
bench: throughput regressed 11.4% (budget: 10.0%): baseline=12816.38 files/s current=11361.67 files/s
```

**Timeline**
`tools/bench/baseline.json` has been written exactly once, ever, by commit
`e7aa091` ("feat(08-07): establish and commit the initial PERF-02/INDX-06
baseline"). The regression predates the entire 09-07 triage session -
`a1c298f` already showed -11.25%, before the grpc bump, the daemon
single-writer fix, and the test-fixture changes. None of those are plausible
causes.

**Reproduction**
The synthetic corpus is deterministic (seed 42) and network-free, so it can be
generated locally:

```
go run ./tools/bench/runner -mode regression -baseline tools/bench/baseline.json -ceiling-bytes 4294967296
```

Latest failing CI run: 30646401249 (main@dcf8580), sole non-passing job.

## Corrected Premise (evidence gathered before the session opened)

The pending todo `.planning/todos/pending/2026-07-31-bisect-indexer-throughput-regression.md`
asserts a real code regression and asks for a bisect. That assertion rests on an
argument that is sound but incomplete, and on a comparison that was never
like-for-like. Both are recorded here so the investigation does not inherit the
error.

**1. The baseline was recorded on a different platform than the gate runs on.**

`tools/bench/baseline.json`:

```json
{
  "subject": "go",
  "repo": "synthetic-seed42-count120000",
  "goos": "darwin",
  "goarch": "arm64",
  "files_per_sec": 12816.383131473423,
  "bytes_per_sec": 1941331.3590571666,
  "query_latency_median_ms": 137.572,
  "peak_rss_bytes": 842350592,
  "cold_start_ms": 11.147
}
```

The gate job is `runs-on: ubuntu-latest` (`.github/workflows/ci.yml:255`), i.e.
linux/amd64. So every measurement the gate has ever made was compared against an
Apple Silicon baseline.

**2. The check silently permits that.**

`tools/bench/runner/main.go:502-503` faithfully records
`GOOS: runtime.GOOS, GOARCH: runtime.GOARCH` into the `Metrics` struct.
`internal/bench/regression.go:33` `CheckRegression` compares only `FilesPerSec`
and `PeakRSSBytes` (plus the absolute RSS ceiling). It never reads `GOOS` or
`GOARCH`. A cross-platform baseline is accepted without warning.

**3. This harness is known to be wildly platform-sensitive.**

The repo contains direct same-code, same-corpus measurements on both platforms
(`tools/bench/headtohead-darwin-arm64-20260713-run*.json` vs
`tools/bench/headtohead-linux-amd64-ci-20260719-run*.json`), subject `go`,
median of 3 runs:

| corpus | darwin/arm64 | linux/amd64 CI | ratio |
|---|---:|---:|---:|
| weft-go | 152,457 | 1,027 | 148x |
| colbymchenry-codegraph | 2,066.7 | 226.1 | 9.1x |
| cockroachdb-pebble | 2,566.3 | 380.6 | 6.7x |

Those are real corpora (cloned) rather than the synthetic one, so the magnitude
does not transfer directly - but it establishes that cross-platform comparison
from this harness is categorically invalid.

**4. Why the todo's reasoning does not settle the question.**

The todo argues: two REDs agreeing to within 13 files/s (0.1%) cannot be noise,
therefore a real regression. The first half is correct - that is not noise. The
second half does not follow. A fixed systematic offset from a wrong-platform
baseline reproduces to 0.1% exactly as faithfully as a code regression does.
"Stable, not noise" was proven; "therefore a code regression" was not, because
there has never been a same-platform control.

Note the synthetic corpus is evidently far less platform-sensitive than the real
ones (10.6% apart, not 6.7x), so a genuine code regression could still be
present underneath the platform offset. Both explanations remain live.

## Current Focus

hypothesis: CONFIRMED. The ~10.6% gap is wholly a darwin/arm64 -> linux baseline
  measurement-frame mismatch, not a code regression. Root cause has two parts:
  (1) no code-level throughput regression exists between e7aa091 and dcf8580 —
  proven by a same-platform control; (2) CheckRegression never validates
  GOOS/GOARCH, so the gate silently compares across platforms.
test: TDD - write a failing test proving CheckRegression accepts a GOOS/GOARCH
  mismatch it should reject, confirm RED, checkpoint, then apply the minimal
  fix (refuse the mismatch with a clear error) and confirm GREEN.
expecting: red test demonstrates the defect independent of the platform-control
  finding; green test after fix proves it's closed.
next_action: ALL FOUR APPROVED PARTS DELIVERED (see Resolution). The one
  remaining action is NOT this session's to take: dispatch bench.yml's new
  `rebless` job on ubuntu-latest and commit the resulting candidate baseline
  in its own PR. Tracked by
  `.planning/todos/pending/2026-07-31-rebless-perf-baseline-on-ubuntu-latest.md`.
  Until that lands the CI gate is RED by design — it now refuses the
  darwin-vs-linux comparison instead of reporting a fictitious regression.
  This session did NOT archive itself to resolved/ for exactly that reason:
  the diagnosis is settled but the end state is not reached. Archive after
  the rebless PR turns the gate green.
  baseline.json was NOT touched in this session (verified: `git status
  --porcelain tools/bench/baseline.json` empty at every commit).

reasoning_checkpoint:
  hypothesis: "The -10.6% gate failure has two independent contributing
    causes and requires both to be true simultaneously (AND-gate): (a) no
    code-level throughput regression exists between e7aa091 and dcf8580 -
    the linux-to-linux same-platform control is flat (+0.73%, opposite
    direction, inside noise); and (b) CheckRegression accepts a baseline
    and current Metrics with different GOOS/GOARCH without complaint,
    letting a darwin/arm64 baseline gate a linux/amd64 CI run."
  confirming_evidence:
    - "Same-platform control: e7aa091 median 38558.08 files/s, dcf8580
      median 38837.64 files/s (linux/arm64, Docker Desktop VM, identical
      environment, only commit varied, median-of-3, 3 trials each) - direct
      observation, not inference."
    - "internal/bench/regression.go:33-65 CheckRegression source read in
      full: only FilesPerSec, PeakRSSBytes, and ceilingBytes are compared;
      no GOOS/GOARCH field is ever read from either Metrics argument -
      direct observation of the function body."
    - "New regression_test.go subtest with baseline{darwin/arm64} vs
      current{linux/amd64}, IDENTICAL FilesPerSec/PeakRSSBytes on both
      sides, currently returns nil (no error) - isolates the platform gap
      from the numeric-tolerance logic and proves it by execution, not
      reading."
  falsification_test: "If the same-platform control had shown dcf8580
    materially slower than e7aa091 (beyond the ~1-3% observed inter-trial
    noise band), hypothesis (a) would be refuted and a residual code
    regression would need its own bisect across e7aa091..dcf8580. It did
    not - both trial sets overlap and the medians differ by +0.73% in the
    faster direction."
  fix_rationale: "The fix must make an invalid comparison fail loudly
    instead of failing silently-wrong. Rejecting a GOOS/GOARCH mismatch in
    CheckRegression addresses cause (b) directly: it is the exact place a
    cross-platform comparison is currently accepted. It does not touch
    cause (a) - no code changes are needed there because there is no code
    regression to fix. Sliding DefaultThroughputTolerance or rewriting
    baseline.json would treat the SYMPTOM (gate is red) without touching
    either root cause, and would also violate the HARD constraint against
    re-baselining outside -rebless."
  blind_spots: "Have not tested GOARCH matching while GOOS differs (e.g.
    linux/amd64 vs linux/arm64) as its own case - the new test only checks
    the fully-mismatched pair. Have not verified whether any CALLER (CI
    workflow, other code) constructs a Metrics value with a deliberately
    empty GOOS/GOARCH expecting CheckRegression to skip the check for
    legacy baselines - reading tools/bench/runner/main.go confirms the
    real baseline.json always has both fields populated, and this is the
    only currently-committed baseline, but a schema-migration edge case
    for a future empty-field baseline was not tested."
  candidate_causes:
    - "data/config: baseline.json recorded on a different platform
      (darwin/arm64) than the CI gate runs on (linux/amd64) - a data
      provenance problem, not a code defect on its own."
    - "code: internal/bench/regression.go CheckRegression has no
      GOOS/GOARCH validation - a code defect that makes the config
      mismatch silently exploitable instead of surfaced."
  and_gate: "yes - both conditions are jointly necessary to produce the
    observed CI failure. If the baseline had been recorded on linux/amd64
    (config cause absent), CheckRegression's missing validation would be a
    latent defect with no visible symptom today. If CheckRegression
    validated platforms but the config cause were absent, there would
    still be nothing to reject. The visible failure required both: a
    real cross-platform baseline AND a check that lets it through."

tdd_checkpoint:
  test_file: "internal/bench/regression_test.go"
  test_name: "TestCheckRegression/GOOS/GOARCH_mismatch_between_baseline_and_current_fails_even_when_metrics_are_identical"
  status: "red"
  failure_output: |
    === RUN   TestCheckRegression/GOOS/GOARCH_mismatch_between_baseline_and_current_fails_even_when_metrics_are_identical
        regression_test.go:154: CheckRegression() = nil, want error
    --- FAIL: TestCheckRegression/GOOS/GOARCH_mismatch_between_baseline_and_current_fails_even_when_metrics_are_identical (0.00s)
    FAIL
    (all 8 pre-existing subtests still PASS)

## Evidence

- timestamp: 2026-07-31
  observation: baseline.json records goos=darwin goarch=arm64; the gate job runs
    on ubuntu-latest. Confirmed by reading tools/bench/baseline.json and
    .github/workflows/ci.yml:254-266.
  source: file read

- timestamp: 2026-07-31
  observation: internal/bench/regression.go:33 CheckRegression compares only
    FilesPerSec and PeakRSSBytes; GOOS/GOARCH are recorded into Metrics by
    tools/bench/runner/main.go:502-503 but never validated by the check.
  source: file read

- timestamp: 2026-07-31
  observation: git log --follow shows tools/bench/baseline.json was written by
    exactly one commit ever: e7aa091 (plan 08-07).
  source: git

- timestamp: 2026-07-31
  observation: repo head-to-head data shows the same code differs 6.7x-148x
    between darwin/arm64 and linux/amd64 CI on real corpora, establishing that
    cross-platform comparison from this harness is invalid.
  source: tools/bench/headtohead-*.json

- timestamp: 2026-07-31
  observation: CI run 30646401249 on main@dcf8580 has exactly one non-passing
    job: "perf regression gate (PERF-02, INDX-06)".
  source: gh run view

- timestamp: 2026-07-31
  observation: Same-platform control built and run. Two self-contained git
    clones (not worktrees - worktree .git-file indirection broke inside
    Docker) checked out at e7aa091 and dcf8580, built and measured natively
    (CGO_ENABLED=1, no cross-compile) inside golang:1.26-bookworm on an
    arm64 Linux container (Docker Desktop's native VM arch on this arm64
    Mac host - go version go1.26.5 linux/arm64, matching go.mod). Ran
    `go run ./tools/bench/runner -mode regression -rebless` 3x per commit
    against the same deterministic seed-42/count-120000 corpus, identical
    environment both times, only the checked-out commit differed.

    e7aa091 files/s: 38504.11, 38558.08, 38970.68 -> median 38558.08
    dcf8580 files/s: 37738.67, 38837.64, 38894.07 -> median 38837.64
    delta: dcf8580 is +0.73% FASTER than e7aa091, well inside the ~1-3%
    inter-trial spread observed at both commits.

    Both measured goos=linux goarch=arm64 (confirmed in each output JSON).
  source: command execution (docker run golang:1.26-bookworm), raw JSON
    outputs in scratchpad/results/{e7aa091,dcf8580}-trial{1,2,3}.json

- timestamp: 2026-07-31
  observation: Interpretation of the control. There is no code-level
    throughput regression between the baseline commit and current main -
    the same-platform ratio is flat (+0.7%, opposite direction of a
    regression, and inside noise). This directly refutes the todo's
    "real regression, needs bisect" premise. The entire -10.6% the CI gate
    reports is explained by comparing a linux current run against a
    darwin/arm64 baseline value (12816.38 files/s) that was never
    established on the gate's own platform - a measurement-frame artifact,
    not a code defect. (Absolute files/s in this control, ~38.5-38.9k, is
    much higher than either the darwin baseline, 12816, or CI's own linux
    readings, ~11.4-11.6k - that's expected and irrelevant: Docker
    Desktop's VM on this host has different CPU/tmpfs characteristics than
    both a bare-metal Mac and a GitHub-hosted ubuntu-latest runner. Only
    the e7aa091-vs-dcf8580 RATIO within this one environment is
    load-bearing, per the same reasoning that makes an emulated-amd64
    control valid despite absolute slowness.)
  source: reasoning over the above measurement

## Eliminated

- hypothesis: The failure is runner-generation drift / noise.
  why: Refuted by the third measurement. Two REDs agree to within 13 files/s
    (0.1%) on different runners days apart; independent noise does not reproduce
    to 0.1%. Recorded in engram n9mmwpshjb, which supersedes the drift
    conclusion in gnews1m4pb and the hypothesis in 9za63pt543.

- hypothesis: The grpc bump, the daemon single-writer fix, or the test-fixture
    changes from the 09-07 triage caused the regression.
  why: The regression predates all of them - a1c298f already measured -11.25%
    before any of those landed.

- hypothesis: A real code-level indexer throughput regression exists between
    e7aa091 (baseline commit) and dcf8580 (current main), of a size warranting
    a bisect per the pending todo.
  why: Refuted by the same-platform control (see Evidence). Measured
    linux-to-linux, identical environment, only the commit varies: dcf8580 is
    +0.73% FASTER than e7aa091, not slower, and well inside the ~1-3%
    inter-trial noise band observed at both commits. If a regression existed
    at the magnitude the gate reports (~10-11%), it would show up as a clear,
    reproducible slowdown here - it does not. No bisect is warranted; the
    pending todo's premise does not hold.

## Constraints

- HARD: do NOT re-baseline. Do NOT modify tools/bench/baseline.json to make the
  gate pass. Moving it would encode an unexplained delta as the new normal and
  permanently destroy the signal. It may move only after the cause is understood
  and accepted as a deliberate trade - and, if the cause is the platform
  mismatch, only as part of a fix that also makes the comparison valid.
- Do NOT conflate two separate pieces of work: (a) explaining the gap, and (b)
  the gate being single-sample while PERF-01 was ratified median-of-3 (engram
  8sa948y0g4). Converting the gate to median-of-N would stop the verdict
  oscillating but would not explain the gap.
- The rebless path (`-rebless`) is the ONLY writer of baseline.json by design
  (D-05). Do not add another.

## Pending Decision — RESOLVED 2026-07-31 (maintainer chose full scope)

**Resolution: option (a) extended to full scope, plus (d).** The maintainer was
shown the options below plus a third defect not in the original list (the gate
is single-sample on BOTH sides, so a rebless from one noisy CI run would bake a
tail value into the new baseline and reproduce the very oscillation that made
this failure ambiguous for three rounds of triage). Chosen scope:

1. Land the `CheckRegression` GOOS/GOARCH guard (closes cause 2).
2. Add a dispatchable rebless path that records the baseline on the same
   `ubuntu-latest` runner class the gate executes on (closes cause 1).
3. Convert the gate to median-of-N so neither side is a single sample.
4. Resolve `.planning/todos/pending/2026-07-31-bisect-indexer-throughput-regression.md`
   as REFUTED — the same-platform control disproved its premise; the bisect it
   asks for must NOT be run.

### Necessary but not sufficient — why the guard alone does not make the gate valid

A `GOOS`/`GOARCH` guard makes cross-*platform* comparison impossible, but it
does not make same-platform comparison *meaningful*. Throughput on this harness
is dominated by hardware class, not just OS/arch. Measured on one physical
Apple Silicon machine, synthetic corpus:

| environment | files/s |
|---|---:|
| darwin/arm64 native | 12,816 (the committed baseline) |
| linux/arm64 container, same machine | ~38,558 |
| GitHub-hosted linux/amd64 runner | ~11,400 |

Same code. A ~3x spread between two OSes on identical hardware, and the
GitHub runner slower than both. So a baseline reblessed on a developer's Linux
box would pass the `GOOS`/`GOARCH` guard and still be a meaningless yardstick
for a shared CI runner. This is why part (2) above specifies the runner class,
not merely the platform — the rebless must happen in CI, on `ubuntu-latest`.

Note `.github/workflows/bench.yml` is already `workflow_dispatch`-able and runs
on `ubuntu-latest`, but it only invokes `-mode headtohead`. There is currently
NO path to record a regression baseline on a CI runner; part (2) is building
that.

---

### Original options as surfaced (retained for the record)

Surfaced at the TDD red checkpoint, unresolved. The code fix (reject a
GOOS/GOARCH mismatch in `CheckRegression`) closes cause (2) and is in scope for
this session. But it has a consequence that must be a conscious choice, not a
surprise:

Once the mismatch check lands, the CI gate will hard-fail on EVERY run — the
committed baseline is darwin/arm64 and the gate runs linux/amd64, so the check
will correctly reject the comparison — until someone records a platform-matched
baseline by running `-rebless` on ubuntu-latest (or equivalent linux/amd64).
This session must NOT do that: `-rebless` is the sole writer of baseline.json by
design (D-05), and the HARD constraint above forbids touching the file here.

Failing loud is the correct behavior (it is strictly better than silently gating
on an invalid comparison), but the interim red gate needs to be expected and
owned. Options to put to the user:
  a. Land the fix now + file a follow-up todo for a deliberate linux/amd64
     `-rebless`. Gate is visibly red until that lands.
  b. Land the fix only; track the rebless outside this session.
  c. Hold the fix until a linux baseline exists, so the gate never enters a
     guaranteed-fail state.
  d. Any of the above, plus resolve the pending bisect todo
     `.planning/todos/pending/2026-07-31-bisect-indexer-throughput-regression.md`
     as REFUTED — the same-platform control disproved its premise, so the bisect
     it asks for should not be run.

## Resolution

root_cause: Two contributing causes (AND-gate: both required to produce the
  observed failure - see checkpoint below for full RCA):
  (1) data/config cause - tools/bench/baseline.json was recorded on
  darwin/arm64 (12816.38 files/s) but the CI gate runs on ubuntu-latest
  (linux/amd64); a same-platform control (linux e7aa091 vs linux dcf8580,
  median-of-3, identical environment) shows +0.73% - no code regression
  exists.
  (2) code cause - internal/bench/regression.go:33 CheckRegression never
  validates that baseline.GOOS/GOARCH match current.GOOS/GOARCH before
  comparing FilesPerSec, so it silently accepts and gates on a cross-platform
  comparison that is categorically invalid for this harness (repo's own
  headtohead data shows 6.7x-148x platform deltas on real corpora).

oracle_type: specified — the assertion is not "no crash" but a stated
  contract ("a baseline recorded on one platform must never gate a run on
  another"), asserted with metrics held IDENTICAL on both sides so the case
  can only fail on the platform check, never on the tolerance arithmetic.

fix: |
  Four parts, all four approved at the resolved Pending Decision, all four
  delivered. Commits on gsd/v1.0-drop-in-parity-human-ux:

  1. 0c4d550 fix(bench): CheckRegression now rejects a GOOS/GOARCH mismatch
     before any numeric comparison, since platform validity precedes numeric
     validity — comparing across platforms is a category error, not a
     tolerance question. Unattributed platform on BOTH sides still matches,
     so callers that build Metrics without platform fields are unaffected.
     Closes root cause (2).

  1b. 1e07032 test(bench): closed the two recorded blind spots. The original
     red case used a fully-mismatched pair, which a guard reading only ONE
     field would also satisfy. Added single-field neighbours in both
     directions, the "no more" case (a matching pair must still pass, so an
     always-reject guard cannot masquerade as correct), and both
     unattributed-platform cases. 9 subtests -> 14.

  2. e35bf7e ci(bench): .github/workflows/bench.yml gained a dispatchable
     `rebless` job on ubuntu-latest — the gate's own runner class. It
     invokes -rebless (still the sole writer of baseline.json, D-05) against
     the runner's throwaway working copy, takes a SECOND independent
     measurement and gates it against the fresh candidate (a baseline nobody
     ever ran the gate against is what caused this bug), publishes a
     committed-vs-candidate delta table to the job summary flagging platform
     and corpus changes explicitly, and uploads the candidate as an
     artifact. The job holds no write permission and cannot commit anything:
     a human reads the number and commits the artifact in its own PR.
     bench.yml gained a `job` selector so dispatching rebless does not also
     run the expensive headtohead publish; the weekly cron still runs
     headtohead only and can never trigger a rebless. Addresses root cause
     (1) by building the missing mechanism — the act of recording is
     deliberately left to a human (see next_action).

  3. ebdc95d feat(bench): -trials N (default 3) on regression mode. Each
     trial is a full materialize+init+measure session with a fresh corpus
     and fresh init; each metric takes its own median across sessions.
     Applied on BOTH the gating and rebless paths, so neither side is a
     single sample. Metrics gained median_of_trials so provenance travels
     with the number. ci.yml states -trials 3 at the call site.

  4. (this commit) .planning/todos/pending/2026-07-31-bisect-indexer-
     throughput-regression.md resolved in place as `status: refuted`,
     following the repo's verified convention (b5d6745 modified the runbook
     todo in pending/ rather than moving it; completed/ is empty and has
     never been used). Its methodological lesson is preserved verbatim and
     extended rather than deleted. The bisect it asks for must not be run.

  Also: tools/bench/BASELINE.md rewritten. Its own third bullet had warned
  since capture that "CheckRegression's tolerance bands assume a consistent
  measurement host across baseline and gate runs" — correct, prose, never
  enforced. That line is now quoted back in the doc as the thing that
  should have been a check.

chosen_N: 3, and it is enforced by a test (TestDefaultTrialsIsAtLeastThree)
  rather than left as a constant anyone can quietly lower.
  - It is the repo's already-ratified figure for its other perf mechanism
    (PERF-01 head-to-head is median-of-3, engram 8sa948y0g4). This closes the
    recorded inconsistency with ONE number for both, rather than inventing a
    second.
  - It is the smallest N with a true median. At N=2 medianFloat64 averages
    the pair, so an outlier still moves the result — below 3 the defect is
    silently reopened.
  - Sufficiency: the measured session-to-session spread is ~1-3% (container
    control) and 2.4% (CI's own three observations) against a 10.0% budget.
    Rejecting a single outlier session is enough at that variance; N=5 costs
    ~1.7x the CI time for marginal precision.
  - Cost, stated not hidden: the ci.yml gate job now runs ~3x longer. It is
    a parallel job and blocks nothing else.
  What is NOT gated: median_of_trials is recorded and printed but is not a
  hard check, unlike GOOS/GOARCH. A differing N is a precision difference,
  not a measurement-frame difference — median-of-3 and median-of-5 estimate
  the same population median — so refusing to compare them would enforce a
  rule that is not a validity rule.

verification: |
  guardrail_verdict: accepted

  1. Original defect reproduced then closed (red -> green):
     RED  (before fix): TestCheckRegression/GOOS/GOARCH_mismatch... failed
       with "CheckRegression() = nil, want error"; other 8 subtests passed.
     GREEN (after fix): all 9 passed, then all 14 after the blind-spot
       cases were added. Command: CGO_ENABLED=1 go test ./internal/bench/...
     The red test was NOT weakened or edited to make it pass.

  2. Mutation-tested (the guardrail signal that matters most here — a
     platform guard is exactly the kind of code that can look right and
     check nothing). Three mutants applied to the fix site, all killed:
       - compare GOOS only  -> kills "GOARCH differs while GOOS matches"
       - compare GOARCH only -> kills "GOOS differs while GOARCH matches"
       - always reject       -> kills 8 cases incl. "matching ... passes"
     None of these would have been caught by the original single red case;
     that is why the blind-spot commit exists.

  3. End-to-end through the real runner, not just the unit under test:
     - gate run against a linux/amd64 baseline while on darwin/arm64 ->
       "runner: regression gate failed: bench: platform mismatch: baseline
       was measured on linux/amd64 but this run is darwin/arm64 ...",
       exit status 1. The wiring from runner -> CheckRegression works.
     - gate run against a same-platform baseline -> compares normally.

  4. The median-of-N change demonstrated its own value on first contact.
     Smoke test, count=3000, darwin/arm64, 3 trials:
       12836.75, 12537.13, 8313.85 files/s
     The third is a -35% tail draw. Single-sample -rebless would have
     committed it verbatim as the new baseline. The median discarded it
     (result: 12537.13). Separately, a `-trials 1` gate run against a
     3-trial baseline reported a spurious -13.2% "regression", while
     `-trials 3` against the same baseline passed — the oscillation this
     bug is made of, reproduced and then removed, in miniature, locally.

  5. Regression / adjacent surface:
     CGO_ENABLED=1 go build ./...                          -> ok
     CGO_ENABLED=1 go vet ./internal/bench/... ./tools/bench/... -> ok
     CGO_ENABLED=1 go test ./internal/bench/... ./tools/bench/... -> ok
       (internal/bench, tools/bench/gencorpus, tools/bench/realcorpus,
        tools/bench/runner all pass)
     actionlint v1.7.12 over .github/workflows/*.yml       -> clean
       (shellcheck present locally, so the shell linting CI performs
        actually ran rather than being silently skipped)
     jq summary expression executed locally against the real committed
     baseline plus a synthetic linux candidate; renders the intended table
     and correctly flags platform **CHANGED**.

  6. HARD constraint honoured: tools/bench/baseline.json untouched.
     `git status --porcelain tools/bench/baseline.json` empty; the file
     does not appear in any of this session's commits. All smoke tests
     wrote to scratch paths via -baseline.

  NOT verified, and cannot be from this session: that the gate goes green
  on ubuntu-latest. That requires the rebless job to run in CI and a human
  to commit the candidate — deliberately out of scope here, tracked by the
  follow-up todo. Until then the gate is RED by design.

files_changed:
  - internal/bench/regression.go: platform-mismatch guard + platformString
    helper; doc comment records why platform validity precedes numeric
    validity.
  - internal/bench/regression_test.go: the red case plus 5 blind-spot /
    boundary-neighbour cases (9 -> 14 subtests).
  - internal/bench/metrics.go: added MedianOfTrials (json median_of_trials)
    as recorded provenance, documented as deliberately not a gate.
  - tools/bench/runner/main.go: -trials flag (default defaultTrials=3,
    validated >= 1); runRegression restructured into an N-trial loop;
    measureRegressionTrial and medianMetrics extracted; headtohead records
    MedianOfTrials=1.
  - tools/bench/runner/main_test.go: medianMetrics tests (per-metric median,
    outlier rejection, identity/platform carry-through, empty, single),
    -trials flag defaults/validation, and defaultTrials >= 3.
  - .github/workflows/bench.yml: `job`/`trials` dispatch inputs; new
    ubuntu-latest `rebless` job (record -> summary -> artifact -> verify).
  - .github/workflows/ci.yml: gate passes -trials 3 explicitly; comment
    records the ~3x cost and the runner-class requirement.
  - tools/bench/BASELINE.md: rewritten — current baseline documented as
    invalid for the gate, platform sensitivity table, two-level measurement
    procedure, CI rebless as the supported path.
  - .planning/todos/pending/2026-07-31-bisect-indexer-throughput-regression.md:
    resolved in place as refuted.
  - .planning/todos/pending/2026-07-31-rebless-perf-baseline-on-ubuntu-latest.md:
    new follow-up (option (a)'s own required companion).

## Prevention (blameless postmortem)

why_not_caught: |
  No gate existed for this class, and the one artifact that could have
  raised the alarm was prose.

  Walking it back, per contributing cause:

  (2) the missing check. Every OTHER gate in this repo asserts something
  about the code under test. This one asserted something about a
  COMPARISON — and nobody wrote a test for the comparison's own
  preconditions. internal/bench had 8 subtests, all of which asked "given
  two numbers, is the tolerance arithmetic right?". None asked "are these
  two numbers even about the same thing?". Unit tests written against the
  happy shape of the data cannot discover that the data's provenance is
  unchecked; you have to think about provenance to test for it. Typecheck,
  lint, vet and review could none of them have caught it: the code was
  well-formed, well-commented, and did exactly what it was written to do.

  (1) the wrong-platform baseline. tools/bench/BASELINE.md documented the
  requirement at capture time, in the same commit that captured the
  baseline: "CheckRegression's tolerance bands assume a consistent
  measurement host across baseline and gate runs." That sentence is
  correct, was written by someone who understood the risk, and stopped
  nothing — because there was no mechanism to record a baseline on the CI
  runner class even if you wanted to (bench.yml was dispatchable on
  ubuntu-latest but only ran headtohead). The requirement was documented
  and simultaneously made impossible to satisfy.

  Then the failure mode compounded: the gate did not fail, it LIED
  quietly, producing a plausible number (-10.6%) with a plausible story
  (something got slower). It cost three rounds of triage, two superseded
  engram conclusions (9za63pt543, gnews1m4pb), and a filed bisect todo
  before anyone questioned the yardstick instead of the measurement. A
  gate that fails loudly wastes minutes; a gate that reports a confident
  wrong number wastes days and trains people to distrust the signal.

  The reasoning error is worth naming precisely, because it was a good
  error: three tight observations correctly established "systematic, not
  noise", and that conclusion was then silently upgraded to "systematic,
  therefore the code". Tightness discriminates signal from noise; it does
  not discriminate between competing explanations of the signal. Both
  hypotheses predicted the same tight cluster. Only a control — vary the
  accused, hold everything else fixed — could separate them, and it did,
  in one pass.

recurrence_guard: |
  - Regression test, mutation-verified:
    internal/bench/regression_test.go:TestCheckRegression — 6 platform
    cases covering the fully-mismatched pair, each single-field neighbour,
    the matching pair (so an always-reject guard cannot pass), and both
    unattributed cases. Three mutants of the guard were applied and all
    three died.
  - Executable check replacing prose: internal/bench.CheckRegression's
    platform guard turns BASELINE.md's documented-but-unenforced host
    assumption into a hard failure.
  - Mechanism closing the config cause: bench.yml's `rebless` job makes
    "record the baseline on the runner class that spends it" possible for
    the first time, and its self-verification step means a candidate that
    is not reproducible on that runner cannot be published as green.
  - Sampling floor: tools/bench/runner/main_test.go:
    TestDefaultTrialsIsAtLeastThree pins N >= 3 so the single-sample defect
    cannot be silently reopened by editing a constant; TestParseFlags_Defaults
    pins the DEFAULT to 3 so a caller that forgets the flag gets the safe
    procedure.
  - Provenance surfaced: Metrics.median_of_trials travels in baseline.json
    and in gate output, and the rebless job's summary table flags platform
    and corpus changes explicitly, so the next reviewer sees the frame,
    not just the number.
  - KB pattern recorded in .planning/debug/knowledge-base.md.

known_gap_not_fixed: |
  CheckRegression still does not compare Metrics.Repo, which encodes the
  corpus identity (e.g. synthetic-seed42-count120000). A baseline
  reblessed with -count 1000 would gate a -count 120000 run and produce
  exactly the same class of fictitious result this session just spent
  itself on — same bug, different field. It is guarded for now only by
  convention (both workflows pass -seed 42 -count 120000 explicitly) and
  surfaced by the rebless summary table, which flags a corpus change.
  Deliberately NOT fixed here: it is a fifth defect, outside the four-part
  scope the maintainer approved, and silently expanding scope is its own
  failure mode. Recommended as the next follow-up.
