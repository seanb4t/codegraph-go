# `baseline.json` provenance and re-bless instructions

`tools/bench/baseline.json` is the committed PERF-02/INDX-06 regression
baseline `tools/bench/runner -mode regression` gates every CI run
against (via `internal/bench.CheckRegression`).

## Status: valid — recorded on the gate's own runner class

The committed baseline was recorded on `ubuntu-latest` (`linux/amd64`) by
`bench.yml`'s `rebless` job (run 30653247679, `trials: 7`) — the same runner
class the blocking gate spends it on. The gate is green: `main@d4672cf`
measured 11453.30 files/s against an 11279.59 baseline (+1.54%).

It was not always so. The baseline this replaced was recorded on
`darwin/arm64` while the gate ran on `ubuntu-latest`, and
`internal/bench.CheckRegression` compared the two without complaint. That
produced a stable, reproducible, entirely fictitious ~10.6% "throughput
regression" which survived three rounds of triage — twice being written off
as runner noise — before a same-platform control across the same commit
range measured **+0.73%**. There was never a code regression. See
`.planning/debug/perf-gate-throughput-regress.md`.

`CheckRegression` now refuses a `GOOS`/`GOARCH` mismatch outright rather
than producing a number that cannot mean anything:

```
bench: platform mismatch: baseline was measured on darwin/arm64 but this
run is linux/amd64; indexing throughput from this harness is not
comparable across platforms, so this comparison would be meaningless.
```

One consequence, by design: **the gate can no longer be run on a developer
macOS machine.** It refuses rather than misleads. To measure locally, run
the gate in CI or use `bench.yml`'s `rebless` job.

Note that the third bullet under "How the current baseline was produced"
below had warned about exactly this since the baseline was captured. It was
prose. Prose does not fail a build — which is why the requirement is now
executable rather than documented.

## Why the platform matters this much

This harness's throughput is dominated by hardware and OS class, not by
the code under test. Same code, synthetic corpus, one physical Apple
Silicon machine:

| environment | files/s |
|---|---:|
| darwin/arm64 native | 12,816 |
| linux/arm64 container, same machine | ~38,558 |
| GitHub-hosted linux/amd64 runner | ~11,400 |

A ~3x spread between two OSes on identical hardware, with the CI runner
slower than both. On real corpora the head-to-head captures in this
directory measure darwin/arm64 vs linux/amd64 CI **6.7x-148x** apart.

The practical consequence: passing the `GOOS`/`GOARCH` guard is
necessary but **not sufficient**. A baseline reblessed on a maintainer's
own Linux box would satisfy the guard and still be a meaningless
yardstick for the CI runner. The baseline has to be recorded on the
runner class where it is spent.

## The `runner` field

`baseline.json` carries a `runner` key alongside `goos`/`goarch`: a
free-form string recording the CI runner identity a `Metrics` was
measured on (e.g. a Namespace `runs-on` profile label such as
`namespace-profile-linux-amd64-4x8`), populated from the
`CODEGRAPH_BENCH_RUNNER` environment variable or the `-runner` flag
(flag wins if both are set).

This exists because `goos`/`goarch` alone do not identify the
measurement environment. `namespace-profile-linux-amd64-4x8` **is**
`linux/amd64` — so a runner-class migration (GitHub-hosted
`ubuntu-latest` to a Namespace profile, say) can change everything about
the hardware a benchmark runs on while holding `goos`/`goarch` constant,
and `CheckRegression`'s platform guard would accept the comparison
without complaint. `runner` closes that blind spot by recording, in the
baseline itself, the exact identifier a human triaging a suspicious
delta would want to compare.

An empty `runner` means "recorded before this field existed" — not an
error. Measurement never requires it; it is only meaningful in CI, where
`bench.yml` sets `CODEGRAPH_BENCH_RUNNER` to its job's own `runs-on`
label.

**`runner` does not yet participate in `CheckRegression`'s comparison.**
Only `goos`/`goarch` gate a run today. Wiring `runner` into the gate is
a deliberately separate, later change — it only makes sense once a
`runner`-labelled baseline exists to compare against, and doing it
before that baseline is committed would compare a runner-labelled
current run against an unlabelled baseline for no benefit.

## The `scratch_fs` field, and why the regression scratch dir now prefers tmpfs

After the `runner` field closed the runner-class blind spot, a Namespace
A/B investigation (10-04-PLAN) found the runner class itself wasn't the
whole story: `namespace-profile-linux-amd64-4x8` and `ubuntu-latest`
report **identical CPU** (`runtime.NumCPU() == 4` on both — measured, not
assumed) but wildly different storage: a 2GiB sequential-write probe
measured 905 MB/s on Namespace vs 153 MB/s on `ubuntu-latest` (read-back:
1.7 GB/s vs 411 MB/s), and Namespace's workspace root is `overlay` where
`ubuntu-latest`'s is `ext4`.

The corpus itself is small — `tools/bench/gencorpus` writes ~18MB of raw
content for 120,000 files — but `du -sh` on the materialized corpus +
`.codegraph/` Pebble store measures **~773MB on disk**, because 120,000
files each consume at least one filesystem block (~4KB) regardless of
their tiny individual size. At that file count the workload is
**IOPS/metadata-bound, not bandwidth-bound** — exactly the shape where
overlayfs's layer-stack lookups and copy-up-on-write cost more, and vary
more under co-tenancy, than a real block device. This matches the
observed signature precisely: peak RSS was rock-stable (0.56%
session-to-session) while throughput swung 4.36%+ — same work, same
memory, different wall-clock is the textbook sign of storage-latency
jitter, not CPU or memory pressure.

**Fix: `tools/bench/runner`'s regression mode now prefers tmpfs
(`/dev/shm`) for its scratch directory** when `-scratch-dir` isn't set
explicitly (`resolveRegressionScratchDir` / `defaultRegressionScratchDir`,
`tools/bench/runner/main.go`), falling back to the previous
`os.TempDir()` behavior when no tmpfs mount exists (e.g. local macOS
development, which has no `/dev/shm`). This removes storage from the
regression measurement path entirely rather than widening a tolerance
band to paper over storage-latency variance the code doesn't have to
pay for.

**Sizing, measured before wiring the fix in (not assumed):** both CI
runner classes were dispatched with a real seed-42/count-120000 corpus
materialize + `codegraph init` + `index --force`:

| runner | `/dev/shm` available | corpus + store on disk |
|---|---:|---:|
| `ubuntu-latest` | 7.9 GiB | 773 MB (~9.8% of tmpfs) |
| `namespace-profile-linux-amd64-4x8` | 4.0 GiB | 773 MB (~18.9% of tmpfs) |

Comfortable headroom on both — this is not a close call.

**`baseline.json` carries a `scratch_fs` key** (`"tmpfs"` or `"disk"`)
recording which filesystem class the measured trials actually landed on.
This exists for exactly the same reason `runner` does: changing the
scratch filesystem is itself a measurement-frame change, indistinguishable
in kind from changing `GOOS` or the runner class. A baseline recorded on
tmpfs compared against a gate run still measuring on overlayfs (or vice
versa) would silently rebuild the original fictitious-regression failure
in a new location. An empty `scratch_fs` means "recorded before this
field existed," or that an operator supplied an explicit `-scratch-dir`
this runner didn't characterize — not an error.

**`scratch_fs` does NOT yet participate in `CheckRegression`'s
comparison**, matching `runner`'s own precedent: `internal/bench/regression.go`
is untouched by this plan. Wiring a `scratch_fs` mismatch into the gate
as a category error (same treatment as the `GOOS`/`GOARCH` refusal, not
a tolerance question) is explicitly left as a handoff to whichever plan
next touches `CheckRegression` (plan 10-06, which already owns wiring
`runner` into the comparison) — this file documents the requirement so
it isn't lost, not to duplicate 10-06's work here.

**Consequence for `ci.yml`'s perf gate today:** the committed baseline
is currently recorded on `ubuntu-latest` disk (no `scratch_fs` key), and
`ci.yml`'s gate still runs on `ubuntu-latest`. That pairing is
self-consistent and untouched by this section — D-09's ordering still
governs: rebless (now tmpfs-aware) → commit baseline → only then move
the gate. Re-blessing on tmpfs and committing that baseline is a
deliberate, separate act from this documentation change.

## Measurement procedure

Two nested levels of repetition, both deliberate:

- **median-of-5 per command** (`medianRuns`, D-05, not a flag) — each
  measured command runs 5 times inside one session, and each metric takes
  its own median.
- **median-of-3 sessions** (`-trials`, default 3) — the whole
  materialize + init + measure cycle repeats 3 times and each metric
  takes its own median across sessions, each session with a fresh corpus
  and a fresh `init`.

The outer level exists because the inner one does not touch
session-to-session variance. Three CI observations of the same corpus,
each already internally median-of-5, spread 2.4% (11374.57 / 11642.18 /
11361.67 files/s) against a 10.0% budget — enough to oscillate the
verdict, and enough for a single-session `-rebless` to commit a tail draw
as the new normal. The trial count travels with the number as
`median_of_trials`.

## Re-blessing (intentional, human-invoked only)

`baseline.json` is **only ever** overwritten by `-rebless` — never as a
side effect of a normal gating run (`internal/bench.CheckRegression`
itself never mutates the baseline).

**The supported path is the `rebless` job in
`.github/workflows/bench.yml`**, because that is what runs on the same
runner class as the gate:

1. Actions → **bench** → *Run workflow*, with `job: rebless`
   (and `trials: 3`, or higher for a more careful capture).
2. The job records a candidate on `ubuntu-latest`, then takes a **second,
   independent measurement** and gates it against that candidate. If the
   candidate is not reproducible within the PERF-02 bands minutes later
   on the same machine, the job fails — that candidate was a tail draw,
   not a baseline.
3. Read the delta table in the job summary. It shows the committed value,
   the candidate, the percentage change, and flags any platform or corpus
   change explicitly.
4. Download the `baseline-candidate` artifact and commit it as
   `tools/bench/baseline.json` **in its own PR**, isolated from any
   unrelated change, per CONTEXT.md D-05.

The workflow holds no write permission and cannot commit the baseline
itself. A human reads the number before it lands, on purpose.

### Running it locally

Useful for investigation; **not** a source of a committable baseline
unless your machine is the gate's runner class, which it almost
certainly is not:

```sh
go run ./tools/bench/runner -mode regression -rebless \
  -baseline /tmp/candidate.json \
  -seed 42 -count 120000 -trials 3
```

Point `-baseline` somewhere outside the repo. `-trials 1` is available
for quick iteration and produces a single-sample number; it is recorded
as `median_of_trials: 1` so nothing downstream can mistake it for a
gate-quality measurement.

## How the current (darwin) baseline was produced

```sh
go run ./tools/bench/runner -mode regression \
  -seed 42 -count 120000 -rebless -baseline tools/bench/baseline.json
```

- **Seed:** `42` (gencorpus's own default) — encoded in `baseline.json`'s
  `repo` field as `synthetic-seed42-count120000`.
- **Count:** `120000` files — this IS `tools/bench/gencorpus.ProductionFileCount`,
  the full INDX-06 100k+ production size. No smaller stand-in was needed:
  a full 120k-file from-scratch index completed in ~8s per run on the
  machine that captured this baseline (darwin/arm64), so the committed
  baseline already reflects the same corpus size the CI gate (Plan
  08-08) runs — like-for-like, no scale mismatch to account for.
- **Host:** darwin/arm64 (`goos`/`goarch` fields in `baseline.json`).
  The original note here read: *"Re-bless on the actual CI runner
  hardware once it's provisioned if its performance characteristics
  diverge meaningfully from this capture machine —
  `internal/bench.CheckRegression`'s tolerance bands assume a consistent
  measurement host across baseline and gate runs."* That assumption was
  correct and was never enforced; the gate shipped and compared across
  hosts anyway. It is now a hard check, not an assumption.
- **Trials:** single session. This capture predates `-trials`, so
  `median_of_trials` is absent and decodes as 0.
- **Peak RSS at capture:** `842350592` bytes (~803 MiB) for the full
  120k-file corpus — comfortably under the runner's default
  `-ceiling-bytes` (4 GiB), which is itself a documented starting point
  to retune once real CI hardware numbers exist.
