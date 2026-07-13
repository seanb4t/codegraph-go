# `baseline.json` provenance and re-bless instructions

`tools/bench/baseline.json` is the committed PERF-02/INDX-06 regression
baseline `tools/bench/runner -mode regression` gates every CI run
against (via `internal/bench.CheckRegression`).

## How it was produced (real, not fabricated)

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
  Re-bless on the actual CI runner hardware once it's provisioned if its
  performance characteristics diverge meaningfully from this capture
  machine — `internal/bench.CheckRegression`'s tolerance bands
  (`DefaultThroughputTolerance` 10%, `DefaultRSSTolerance` 15%) assume a
  consistent measurement host across baseline and gate runs.
- **Peak RSS at capture:** `842350592` bytes (~803 MiB) for the full
  120k-file corpus — comfortably under the runner's default
  `-ceiling-bytes` (4 GiB), which is itself a documented starting point
  to retune once real CI hardware numbers exist.

## Re-blessing (intentional, human-invoked only)

`baseline.json` is **only ever** overwritten by `-rebless` — never as a
side effect of a normal gating run (`internal/bench.CheckRegression`
itself never mutates the baseline). To intentionally accept a new
normal (e.g. after a real, expected performance change):

```sh
go run ./tools/bench/runner -mode regression \
  -seed 42 -count 120000 -rebless -baseline tools/bench/baseline.json
```

Review the resulting `baseline.json` diff in its own PR, isolated from
any unrelated code change, per CONTEXT.md D-05.
