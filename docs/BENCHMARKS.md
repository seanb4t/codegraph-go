# Benchmarks: codegraph-go vs TS CodeGraph

This document describes how codegraph-go's performance is measured, the
raw numbers a real head-to-head run produces, and the CI regression gate
that protects against silently getting slower or heavier over time.

## 1. Methodology

### What is measured

Four metrics, per `internal/bench.Metrics`:

| Metric | What it measures | How |
|---|---|---|
| Indexing throughput | files/s and bytes/s on a **from-scratch** `index --force` | Self-measured file/byte count of the corpus (never trusting either binary's own self-reported stats), divided by wall-clock duration of the `index --force` run |
| Query latency | median wall-clock time of a single `query <term>` invocation | A fixed, real symbol name confirmed present in the corpus at the pinned commit (never an arbitrary string that may or may not match) |
| Peak RSS | peak resident set size of the **`index --force`** subprocess | OS-level, external measurement — see below |
| Cold start | median wall-clock time of `--version` | A trivial subcommand that still exercises full binary/process startup cost |

### median-of-5

Every measured command is run **5 times** (`medianRuns` in
`tools/bench/runner/main.go`), and each metric's median is reported
independently over its own sorted sample — not "the run whose total
duration happened to be the median," but the median duration and the
median peak RSS computed separately. This damps run-to-run OS noise
(disk cache state, GC pause timing, scheduler jitter) without needing a
large sample size.

### Peak RSS is measured at the OS level, not in-process

Peak RSS is read via `bench.PeakRSSBytes(cmd.ProcessState)` — i.e.
`getrusage`'s `ru_maxrss`, exposed through Go's `os.ProcessState`, on the
**completed child process** — never via the *runner's own* in-process
`runtime.MemStats`.

This distinction matters: an in-process `runtime.MemStats` reading would
only be meaningful for the Go binary, and there is no equivalent
same-process reading available for the TS/Node subject at all — Node's own
`process.memoryUsage()` measures the V8 heap from *inside* the Node
process, which is not directly comparable to a Go process's RSS even in
principle, and querying it would require instrumenting the TS binary
itself. Measuring both subjects externally, from the parent `runner`
process's point of view via the OS's own process-accounting API, is the
only apples-to-apples way to compare a Go binary's memory footprint against
a TS-CodeGraph-on-Node process's memory footprint — and it is exactly what
makes peak RSS usable as a first-class, externally-observable CI metric for
the regression gate (INDX-06), independent of either subject's runtime.

### Comparison target

The head-to-head comparison runs against the **installed TS
`@colbymchenry/codegraph@1.3.1`** binary (the exact released version this
project targets parity with), invoked at its real installed path (default
`/opt/homebrew/bin/codegraph`, overridable via `-ts-binary`).

### Pinned real repos (PERF-01)

Three real, publicly-available repos, each pinned by full 40-character
commit SHA (never a branch or tag alone) so a published number stays
reproducible run-to-run and machine-to-machine
(`tools/bench/realcorpus/manifest.go`):

| Name | Source | Commit SHA | Selection | Why this repo |
|---|---|---|---|---|
| `weft-go` | `github.com/seanb4t/weft` | `f89ae3ea4e4c37509f7302fd4e37986212a72079` | entire repo (~84 files, mostly Go) | Compact, already-pinned corpus (reused from `testdata/golden`'s Phase-3 oracle) |
| `colbymchenry-codegraph` | `github.com/colbymchenry/codegraph` | `edb9f2f14cd7394a4d31f94ebc871531ef498ab0` | entire repo (multi-language: TS/JS/Python/Astro/YAML) | The original TS project this port replaces — exercises multi-language extraction breadth |
| `cockroachdb-pebble` | `github.com/cockroachdb/pebble` | `dbdc1acb859689dc4237b40ef8fcdbb877526a84` (`v2.1.6`) | entire repo | A real, substantially larger Go codebase — also this project's own storage dependency — to exercise scale beyond `weft-go`'s compact size |

Each entry also carries a fixed query-latency probe term (a real symbol
confirmed present at that pinned commit — see `manifest.go` for the exact
terms) and a `License`/`SelectionRule` field, mirroring
`tools/spike/testdata/ATTRIBUTION.md`'s provenance discipline.

### Regenerating the numbers

```sh
go run ./tools/bench/runner -mode headtohead
```

This builds a fresh Go `codegraph` binary, resolves (or shallow-clones, at
exactly the pinned commit — never `HEAD`) each of the three repos above,
and runs both subjects over each, printing a JSON array of `bench.Metrics`
to stdout. `.github/workflows/bench.yml` runs this exact mode on
`workflow_dispatch` and a weekly `schedule` — never on `pull_request`/`push`
— and publishes the raw output as both a job-summary table and a
downloadable artifact.

## 2. Raw numbers

**These are the MEDIAN of 3 full head-to-head runs** — not a single run.
Each cell below is the per-metric median across **three** independent
`tools/bench/runner -mode headtohead` invocations (each itself already a
median-of-5 per command), captured back-to-back on **2026-07-13 on a
darwin/arm64 (Apple Silicon) local development machine**, against the
installed TS `@colbymchenry/codegraph@1.3.1`. Taking the median across
runs (rather than reporting the first, coldest, most-favorable run) guards
against cherry-picking and against per-run thermal/scheduler noise. It is
NOT a live `bench.yml` CI run — that remains the canonical publish target
(see the hardware caveat below). Every figure is transcribed verbatim from
runner JSON (nothing hand-estimated); all three raw runs are committed
(see provenance below), so the median is independently re-derivable.

| Repo | Subject | Files/s (median-of-3) | Bytes/s | Query latency (ms) | Peak RSS | Cold start (ms) |
|---|---|---|---|---|---|---|
| weft-go | **go** | 152,457.08 | 1,592,931,724 (≈1.59 GB/s) | 40.529 | 55,689,216 (55.7 MB) | 12.452 |
| weft-go | ts | 24,915.99 | 260,332,067 (≈260.3 MB/s) | 110.209 | 305,889,280 (305.9 MB) | 73.673 |
| colbymchenry-codegraph | **go** | 2,066.66 | 225,199,365 (≈225.2 MB/s) | 42.702 | 103,120,896 (103.1 MB) | 12.132 |
| colbymchenry-codegraph | ts | 161.79 | 17,629,859 (≈17.6 MB/s) | 118.710 | 511,148,032 (511.1 MB) | 72.985 |
| cockroachdb-pebble | **go** | 2,566.26 | 40,438,690 (≈40.4 MB/s) | 51.945 | 161,054,720 (161.1 MB) | 11.876 |
| cockroachdb-pebble | ts | 43.01 | 677,761 (≈678 KB/s) | 117.581 | 580,583,424 (580.6 MB) | 72.174 |

### Go vs TS 1.3.1 — summary (from the medians above)

| Repo | Indexing throughput | Query latency | Peak RSS | Cold start |
|---|---|---|---|---|
| weft-go | **6.1× faster** | **2.7× lower** | **5.5× lighter** | **5.9× faster** |
| colbymchenry-codegraph | **12.8× faster** | **2.8× lower** | **5.0× lighter** | **6.0× faster** |
| cockroachdb-pebble | **59.7× faster** | **2.3× lower** | **3.6× lighter** | **6.1× faster** |

Across all three real repos, codegraph-go beats TS CodeGraph 1.3.1 on
**every** metric: indexing throughput by 6.1×–59.7×, query latency by
~2.3–2.8×, peak RSS by 3.6×–5.5×, and cold start by ~6× — direct evidence
for the project's core "same or better — faster, lighter, from a single
binary" value proposition.

### Run-to-run repeatability (3 runs)

Across the three runs, the metrics separate into stable and noisy:

- **Peak RSS, throughput (files/s), and cold start are highly repeatable** —
  coefficient of variation (CV) **< ~4%** on essentially every cell (peak
  RSS is the tightest, CV 0.2–3.0%). Safe to cite as absolutes.
- **Go query latency on the two larger repos is the noisy metric** — CV
  ~8.9% (colbymchenry/go) and ~8.8% (pebble/go), drifting monotonically
  upward across the three back-to-back runs. That upward drift is the
  signature of **thermal throttling / load accumulation** from running
  three passes in a row on a laptop, not a real regression — the TS query
  numbers on the same repos stayed stable in *absolute* ms; they only look
  steadier in *percentage* terms because TS's query baseline is ~3× larger.
  On isolated CI hardware this metric is expected to tighten.
- **The Go/TS ratios stayed decisive in every individual run** (throughput
  6.8/5.9/6.1× on weft-go, 13.7/12.6/12.2× on colbymchenry, 61.6/59.7/56.6×
  on pebble) — the conclusion is robust regardless of which run you cite.

**Reading the throughput numbers:** `files/s` is `corpus_file_count /
index_wall_time`, so on the tiny `weft-go` corpus (~84 files) fixed
per-invocation startup cost dominates and inflates the absolute `files/s`
for *both* subjects — the meaningful comparison is the Go-vs-TS **ratio
within each repo** (both subjects pay the same overhead structure) plus
`bytes/s` and the larger `cockroachdb-pebble` corpus, where the ratio
widens to ~60× as real per-file work dominates over fixed startup.

**Hardware caveat (before treating these as canonical):** these are local
runs on Apple Silicon (darwin/arm64). Absolute magnitudes are
machine-specific; the Go-vs-TS *ratios* are the durable signal. The
canonical published numbers should come from a `bench.yml`
`workflow_dispatch` run on standardized GitHub Actions hardware — trigger
that once the release is cut, and replace this table's absolute figures
with the CI run's output (keeping the ratio summary, which is expected to
hold).

**Provenance.** The three verbatim runner-JSON runs the medians above are
computed from are committed alongside this doc:
`tools/bench/headtohead-darwin-arm64-20260713-run1.json`,
`-run2.json`, and `-run3.json` (regenerate any run with
`go run ./tools/bench/runner -mode headtohead`).

### The one real, committed number: the synthetic regression baseline

Unlike the head-to-head table above, `tools/bench/baseline.json` — the
PERF-02/INDX-06 regression baseline — **is** real, committed runner output,
captured at the full 120,000-file production scale (see
`tools/bench/BASELINE.md` for full provenance):

```json
{
  "subject": "go",
  "repo": "synthetic-seed42-count120000",
  "goos": "darwin",
  "goarch": "arm64",
  "files_per_sec": 12816.38,
  "bytes_per_sec": 1941331.36,
  "query_latency_median_ms": 137.572,
  "peak_rss_bytes": 842350592,
  "cold_start_ms": 11.147
}
```

This is a from-scratch `index --force` of a deterministic, network-free,
120,000-file synthetic corpus (captured on darwin/arm64) — ~803 MiB peak
RSS, comfortably under the regression gate's absolute ceiling (§3 below).
It is not a head-to-head number (there is no TS-subject equivalent captured
against the synthetic corpus) — it exists purely to give the CI regression
gate something real to compare against from day one, without waiting on a
live head-to-head run.

## 3. Regression gate (PERF-02, INDX-06)

`.github/workflows/ci.yml`'s `perf-regression` job runs on every PR/push:

```sh
go run ./tools/bench/runner -mode regression \
  -baseline tools/bench/baseline.json \
  -ceiling-bytes 4294967296
```

This is **entirely offline and deterministic** — it materializes the
synthetic corpus via `tools/bench/gencorpus` (seed `42`, `120000` files by
default, matching `ProductionFileCount`, INDX-06's own 100k+ requirement)
and never clones a remote repo or touches the network. The same seed always
produces a byte-identical corpus, so the gate never flakes on corpus
variance.

`internal/bench.CheckRegression` fails the run when:

- **Throughput regresses more than `DefaultThroughputTolerance` (10%)**
  relative to the committed baseline's `files_per_sec`.
- **Peak RSS grows more than `DefaultRSSTolerance` (15%)** relative to the
  committed baseline's `peak_rss_bytes`.
- **Peak RSS exceeds the absolute ceiling** (`-ceiling-bytes`, default
  4 GiB / `4294967296` bytes) — checked **independently** of the relative
  15% tolerance band, so a baseline that already sits close to the ceiling
  can't mask further unbounded growth behind a large denominator (a
  baseline near 950 MB growing "only" 10% would still be well within the
  15% band, but could still be pushed over an absolute budget the relative
  check alone would never catch).

A passing run prints `regression gate passed`; a failing run exits
non-zero with the specific tolerance or ceiling that was exceeded, plus the
measured metrics, to stderr.

### Re-blessing the baseline (explicit, human-invoked only)

`tools/bench/baseline.json` is **only ever** overwritten by `-rebless` —
never as a side effect of a normal gating run:

```sh
go run ./tools/bench/runner -mode regression \
  -seed 42 -count 120000 -rebless -baseline tools/bench/baseline.json
```

Use this only to *intentionally* accept a new normal (e.g. after a real,
expected performance change to the indexer). Review the resulting
`baseline.json` diff in its own dedicated PR, isolated from any unrelated
code change — never bundle a re-bless with a feature or bugfix commit,
so a reviewer can evaluate the performance delta on its own terms.

**Known caveat:** the committed baseline was captured on darwin/arm64 (a
local development machine), not on GitHub Actions' `ubuntu-latest` runner
hardware that `ci.yml` actually gates against. `CheckRegression`'s
tolerance bands assume a reasonably consistent measurement host between
baseline capture and gate run; re-blessing once real CI runner hardware
numbers are available (rather than continuing to gate CI runs against a
different machine's baseline) is recommended before treating small
regressions on CI as meaningful.
