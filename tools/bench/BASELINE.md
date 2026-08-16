# `baseline.json` provenance and re-bless instructions

`tools/bench/baseline.json` is the committed PERF-02/INDX-06 regression
baseline `tools/bench/runner -mode regression` gates every CI run
against (via `internal/bench.CheckRegression`).

## Status: valid — recorded on the gate's own runner class, reblessed 2026-08-02 after a staleness finding

The committed baseline was recorded on `ubuntu-latest` (`linux/amd64`,
`scratch_fs: disk`) by `bench.yml`'s `rebless` job (run 30758999046,
`trials: 7`) — the same runner class the blocking gate spends it on, and
the same scratch-filesystem class the gate itself uses. Both frame
descriptors (`runner`, `scratch_fs`) are populated for the first time in
this baseline.

**`files_per_sec: 17090.88` — up 51.5% from the baseline this replaced
(`11279.59`), which had been reblessed only two days earlier.** That is
not a frame change: same runner class, same `scratch_fs: disk`, same
`-seed 42 -count 120000 -mode regression`. See "10-04-PLAN: the staleness
finding" below — this is likely the single most consequential thing this
plan produced, and the cause of the jump is **not established**. Read
that section before trusting any percentage this file reports.

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
slower than both. On real corpora the platform spread measured wider
still: darwin/arm64 vs linux/amd64 CI measured **6.7x-148x** apart.

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

**`runner` now participates in `CheckRegression`'s comparison** (plan
10-06): a `runner` mismatch is refused as a category error, the same
treatment as `goos`/`goarch`, and fires before any numeric tolerance
check. An empty `runner` on either side is refused too — it means "never
recorded", not a wildcard — so a baseline predating this field (any
`baseline.json` committed before plan 10-04) must be re-blessed via
`bench.yml`'s `rebless` job before it can gate a `runner`-attributed run.
Both sides empty still passes, so unit tests and any other caller that
never attributes `Metrics.Runner` remain unaffected.

## The `scratch_fs` field

`baseline.json` carries a `scratch_fs` key (`"tmpfs"` or `"disk"`)
recording which filesystem class the measured trials actually landed on.
This exists for exactly the same reason `runner` does: changing the
scratch filesystem is itself a measurement-frame change, indistinguishable
in kind from changing `GOOS` or the runner class. A baseline recorded on
tmpfs compared against a gate run still measuring on disk (or vice versa)
would silently rebuild the original fictitious-regression failure in a
new location. An empty `scratch_fs` means "recorded before this field
existed," or that an operator supplied an explicit `-scratch-dir` this
runner didn't characterize — not an error.

**`scratch_fs` now participates in `CheckRegression`'s comparison too**
(plan 10-06, same commit that wired `runner` in) — same category-error
treatment, same empty-means-never-recorded handling, same
before-any-numeric-check ordering. A baseline predating this field must
be re-blessed via `bench.yml`'s `rebless` job before it can gate a
`scratch_fs`-attributed run.

**The scratch filesystem class defaults to `disk`** (`-scratch-fs disk`,
`tools/bench/runner`'s flag default). It is pinnable via `-scratch-fs`
(`disk`, `tmpfs`, or `auto` — prefer tmpfs, fall back to disk) for
exactly this kind of investigation. See the next section for why disk,
not tmpfs, won.

## 10-04-PLAN: the Namespace investigation, the tmpfs refutation, and the staleness finding

This is the full record of a multi-round investigation, kept complete on
purpose — a future reader needs to know tmpfs was tried and made things
**worse**, or someone will spend a day re-discovering that.

### Round 1 — Namespace migration, an unexplained variance

10-04-PLAN set out to move `bench.yml`'s `rebless` job (and, eventually,
`ci.yml`'s perf gate) from `ubuntu-latest` to a Namespace runner profile
(`namespace-profile-linux-amd64-4x8`), per D-06. A same-runner-class
7-trial rebless on Namespace showed a real problem: two independent
7-trial sessions disagreed by **4.36%** (only **2.30x** headroom against
the 10% `DefaultThroughputTolerance` budget), against peak RSS that
stayed rock-stable at **0.56%** session-to-session. Same work, same
memory, very different wall-clock — a real, reproducible instability,
not sampling noise (going from 3 to 7 trials barely moved the number).

### Round 2 — hypothesis: CPU oversubscription. Refuted.

First hypothesis: `tools/bench/runner` never passes `--workers` to
`codegraph index`, so the extraction pool defaults to `runtime.NumCPU()`
— if a runner's cgroup CPU quota is smaller than its host-visible CPU
count, Go would oversubscribe the container. **Measured, not assumed:**
`nproc`, `runtime.NumCPU()`, `runtime.GOMAXPROCS(0)`, and the cgroup v2
`cpu.max` quota all agreed at exactly 4, unlimited quota, on Namespace.
Oversubscription was not the story. (See `Taskfile.yml`'s `diag:cpu`
target and `tools/bench/cpudiag` — kept as permanent diagnostic tooling.)

### Round 3 — A/B, and a second hypothesis: I/O-bound variance on overlayfs. Also refuted, but instructively.

A same-day CPU/memory/storage A/B between `ubuntu-latest` and Namespace
found **identical CPU count** (`runtime.NumCPU() == 4` on both) but a
large sequential-I/O gap: a 2GiB `dd` write/read-back probe measured 905
MB/s / 1.7 GB/s on Namespace vs 153 MB/s / 411 MB/s on `ubuntu-latest`,
and Namespace's workspace root is `overlay` where `ubuntu-latest`'s is
`ext4`. The regression corpus is small in raw bytes (`tools/bench/gencorpus`
writes ~18MB of content for 120,000 files) but `du -sh` on the
materialized corpus + `.codegraph/` Pebble store measured **~773MB on
disk**, because 120,000 files each consume at least one filesystem block
(~4KB) regardless of their tiny individual size — this workload is
**IOPS/metadata-bound, not bandwidth-bound**, exactly the shape where
overlayfs's layer-stack lookups and copy-up-on-write were hypothesized to
cost more and vary more under co-tenancy than a real block device.

**The fix this hypothesis implied — point the regression scratch
directory at tmpfs (`/dev/shm`) — was implemented, measured, and
refuted.** A same-day, same-methodology control across all four
`(runner, scratch_fs)` combinations found tmpfs made variance **worse on
both runner classes**, not better:

| config | session-to-session disagreement | headroom (10% tolerance) |
|---|---:|---:|
| `ubuntu-latest` + disk | **0.35%** | **28.6x** |
| `ubuntu-latest` + tmpfs | 5.75% | 1.74x |
| `namespace-profile-linux-amd64-4x8` + disk | 4.36% | 2.30x |
| `namespace-profile-linux-amd64-4x8` + tmpfs | 12.46% | 0.80x |

The sequential-`dd` evidence was real but was circumstantial for this
workload: `dd` measures sequential bandwidth, and this benchmark's actual
bottleneck is 120,000 discrete small-file metadata operations, a
different access pattern `dd` never exercised directly. With storage
genuinely controlled for (same tmpfs on both runners), Namespace's
variance got **worse**, not better — evidence against I/O-bound variance
being the (sole) driver, and consistent instead with host-placement
variance: each Namespace session in this investigation ran on a
**different ephemeral VM instance** (`nsc-runner-js3chg16gsvfu` vs
`nsc-runner-95o3fs4b3avu6`), while `ubuntu-latest`'s sessions, also on
different instances (`GitHub Actions 1000017094` / `...095`), stayed
tight regardless. GitHub's larger, more homogeneous fleet appears to
absorb whatever this is; Namespace's does not, at least not without
further work (see "Deferred: Namespace cache volumes" below).

### Round 4 — the missing control, and the actual answer

Before concluding "tmpfs harmed `ubuntu-latest` too," the investigation
caught its own gap: the only historical `ubuntu-latest + disk` number on
record (0.65% disagreement, ~15x headroom) was measured **weeks earlier**
in Phase 9, under different tooling, on a fleet that could itself have
drifted. Comparing a fresh Namespace/tmpfs measurement against a stale
`ubuntu-latest` reference would have repeated, in a new location, the
exact mistake that produced this repo's original fictitious 10.6%
"regression" (see "It was not always so," above).

**The control:** `ubuntu-latest` + disk, measured TODAY, same
methodology as every other number in this table (`-scratch-fs disk`,
added as a first-class flag specifically so this control didn't require
reverting the tmpfs code path to run). Result: **0.35% disagreement,
28.6x headroom** — tighter than the historical reference, not stale or
lucky. This is branch 1 of the decision this investigation was designed
to distinguish: the incumbent frame (`ubuntu-latest`, disk-backed
scratch) is genuinely, reproducibly the most stable configuration
measured, by a wide margin, and it costs nothing extra on a public repo.

One more re-scaling worth recording: the "+79.5%" Namespace-over-
`ubuntu-latest` throughput advantage that motivated Round 1 was measured
against the STALE `ubuntu-latest` baseline (11279.59 files/s). Against
the `ubuntu-latest` + disk control's average (`(16330.41 + 16387.47) / 2
≈ 16358.94` files/s) and Namespace's own disk-backed 7-trial measurement
from earlier in this investigation (`19733.69` files/s, pre-tmpfs-fix),
Namespace + disk is **+20.6%** over `ubuntu-latest` + disk — modest, and
consistent with somewhat faster underlying storage rather than a
dramatic win. The original "+79.5%" headline was mostly baseline
staleness, not a real Namespace throughput advantage of that magnitude.

**Resolution:** `-scratch-fs`'s default flipped from `auto` (prefer
tmpfs) to `disk`. `bench.yml`'s `rebless` job returned to `ubuntu-latest`
(matching `ci.yml`'s still-unmoved perf gate — nothing needed to move on
the gate side, since the gate was never migrated). `publish`
(PERF-01, non-gating, publish-only) deliberately **stays** on Namespace —
it never needs to match `rebless`'s runner class, and Namespace measures
meaningfully faster on this corpus, so leaving a non-gated job there
costs nothing. `DefaultThroughputTolerance` was **not** changed: the existing 10% budget
is justified and the adopt-and-widen path considered mid-investigation was
abandoned entirely. But see "Which headroom number applies to the gate"
below before sizing any future tolerance off the 28.6x figure — it
describes within-dispatch session agreement, which is NOT the variance the
gate experiences.

### Which headroom number applies to the gate

Two different quantities get called "headroom" in this document, and only
one of them describes how the gate is actually used.

**Within-dispatch session agreement** — two 7-trial sessions run
back-to-back inside a single CI job. Measured 0.35% (control run) and
0.62% (rebless run) on `ubuntu-latest` + disk, i.e. 28.6x and 16.1x
against the 10% tolerance. This is what most of the comparisons in this
document report, because it is what a single dispatch can measure about
itself.

**Across-dispatch variance** — the spread between separate CI dispatches
hours apart. All four `ubuntu-latest` + disk session medians recorded in
this investigation:

| dispatch | session medians (files/s) |
|---|---|
| disk-scratch control | `16330.41`, `16387.47` |
| rebless (committed)  | `17090.88`, `16984.54` |

Full spread `16330.41` -> `17090.88` = **4.66%**, i.e. roughly **2.15x**
headroom against the 10% tolerance.

**The gate experiences the second number, not the first.** `CheckRegression`
takes ONE fresh measurement in a NEW CI dispatch and compares it against
this committed baseline; it never observes within-dispatch session
agreement. Sizing a tolerance off 28.6x would assume ~13x more margin than
the gate actually has.

2.15x is still workable — a >10% regression trips it while ~4.7% of
observed drift does not — and it is far better than Namespace's
sub-1.0x. **The runner decision is unaffected:** Namespace + tmpfs
disagreed by 12.46% within a single dispatch, before across-dispatch
variance is even counted.

**The ~4.7% across-dispatch component is NOT explained.** Two candidates,
neither verified: genuine drift between dispatches on the same runner
class, or a difference in how the read-only control job and the `-rebless`
job measure (phase structure, corpus regeneration, binary build). If a
future change depends on this number, establish which it is first rather
than inheriting the assumption — treating the two jobs as comparable is
itself an unverified premise.
The `-scratch-fs`/`ScratchFS`/`resolveScratchDirForClass` machinery all
stays — first-class capabilities, not reverted, because they are what
made every control and comparison in this investigation cheap to run,
and what the cache-volume follow-up below will need.

### The staleness finding itself — separate from, and larger than, the runner-migration question

Reblessing on `ubuntu-latest` + disk today, to close out this
investigation, surfaced a second, independent finding: **the baseline
this replaced (`files_per_sec: 11279.59`, reblessed only two days
earlier on 2026-07-31) was itself stale by 51.5%.** Today's `ubuntu-latest`
+ disk measurement is `17090.88` files/s — same runner class, same
`scratch_fs: disk`, same `-seed 42 -count 120000 -mode regression`. This
is not a frame change on either side.

**Consequence:** at the old baseline, `CheckRegression`'s 10% tolerance
only tripped below `11279.59 * 0.9 ≈ 10152` files/s — a **37.8% drop
from today's actual performance** would still have gated green. The gate
had quietly lost most of its detection power in two days, not because
the tolerance band was wrong, but because the number it was measured
against had gone stale.

**The cause of the 51.5% jump is NOT established.** It is either a
genuine code speedup somewhere in the two-day window, or a GitHub-hosted
fleet hardware change — the CPU/storage A/B in this investigation
recorded `ubuntu-latest`'s CPU as `AMD EPYC 9V74 80-Core Processor`
(family 25, stepping 1), and GitHub is known to roll fleet hardware
without advance notice. **Do not assert a cause that has not been
measured.** If this recurs, the next re-bless's delta table (job summary,
`bench.yml`'s `rebless` job) is the place to start looking, and a `git
bisect` against the two-day commit range is the only way to distinguish
"code got faster" from "hardware changed" with any confidence.

### Deferred: Namespace cache volumes

Namespace offers cache volumes (local NVMe block storage, distinct from
the overlayfs root filesystem this investigation measured) as a
candidate faster-and-more-stable-than-tmpfs storage backend. **Untested.**
Filed as a follow-up, not pursued in this plan. Caveat worth stating
explicitly: a cache volume would only address the storage-latency
hypothesis (Round 3), which the tmpfs experiment already partially
refuted — it would **not** address host-placement variance (the
different-VM-instance observation in Round 3), which stays the more
likely explanation for Namespace's excess session-to-session variance
after storage is controlled for. Anyone picking this up should expect
cache volumes to help some, but should not expect them to close the gap
to `ubuntu-latest`'s 28.6x headroom on their own.

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

## How the CURRENT baseline was produced (2026-08-02, ubuntu-latest, disk)

```sh
go run ./tools/bench/runner -mode regression -rebless \
  -baseline tools/bench/baseline.json \
  -seed 42 -count 120000 -trials 7 \
  -runner ubuntu-latest -scratch-fs disk
```

Dispatched via `bench.yml`'s `rebless` job (run 30758999046), `job:
rebless`, `trials: 7`. Downloaded the `baseline-candidate` artifact and
committed it byte-for-byte (verified via SHA-256 before commit — no
reformatting, no rounding, no key reordering).

- **Seed/count:** `42` / `120000`, same as always — `repo:
  "synthetic-seed42-count120000"`, `tools/bench/gencorpus.ProductionFileCount`.
- **Host:** `ubuntu-latest` (`linux`/`amd64`), `runner: "ubuntu-latest"`,
  `scratch_fs: "disk"` — both frame descriptors populated for the first
  time in a committed baseline. See "10-04-PLAN: the Namespace
  investigation" above for why this is the runner/scratch-fs pairing
  that won.
- **Trials:** `median_of_trials: 7`. Record-phase 7 trials: `17090.88,
  17142.65, 16909.77, 16973.85, 16841.61, 17109.98, 17138.99` files/s
  (median `17090.88`, matches the committed value). The job's own
  independent reproducibility check (a second 7-trial session gated
  against this candidate) passed: verify-phase median `16984.54`, a
  0.62% session-to-session disagreement — 16.1x headroom against the
  10% tolerance, consistent with the 28.6x control measurement that
  motivated staying on this runner/scratch-fs pairing. **Both figures are
  within-dispatch.** Against the control dispatch's sessions the spread is
  4.66% (~2.15x), which is the number the gate actually experiences — see
  "Which headroom number applies to the gate" above.
- **Peak RSS at capture:** `845950976` bytes (~807 MiB) — consistent
  with every other measurement in this investigation (peak RSS stayed
  in a ~840-880 MiB band throughout, never the variable quantity).
- **Staleness relative to the immediately prior baseline:** `+51.5%`
  (`11279.59` → `17090.88`). See "The staleness finding itself" above —
  the cause is not established, do not assert one.

## Historical: how the ORIGINAL darwin baseline was produced (superseded)

Kept for provenance history only — this baseline was replaced by the
`ubuntu-latest`-recorded one described in "Status" above before this
plan even started, and is two staleness-relevant generations behind the
current committed value.

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
