---
phase: 02
plan: 01
title: Rename comparison framing to behavioral vocabulary
agents: gsd-executor (worktree isolation)
status: complete
---

# Plan 02-01 — Rename (CODE-02, Diff A part 1)

## What was done

Renamed the comparison-framing out of the golden harness in one atomic commit (`916bdd9`), changing no golden byte:

- `parity_{java,tsjs,csharp,python}_test.go` → `behavioral_{java,tsjs,csharp,python}_test.go`
- `golden_parity_test.go` → `behavioral_test.go`; `TestGoldenParity*` / `TestGoldenBehavioral*` identifiers → behavioral vocabulary
- Carried the capability-matrix gate in the same commit: `goldenTestFuncsByLanguage` (matrix_test.go), the `matrix.go` doc strings, `docs/LANGUAGE-CAPABILITY-MATRIX.md`, and stale refs in `internal/corpora/coverage_test.go` and `testdata/golden/README.md`.

10 files changed, 64 insertions, 64 deletions, **zero golden bytes moved.**

## Contract points honored

- The capability-matrix gate (`TestMatrix_FullPriority4EntriesHaveGoldenTest`) moves in the same commit — it would otherwise have failed `go test ./...`.
- `golden_test.go`'s corpus-path reference (`corpus/synthetic-parity`) is deliberately NOT updated here — the corpus moves in 02-02; the rename diff stays byte-clean.
- No change to `tools/bench/realcorpus`, `bench.yml`, `headtohead-*.json`, `docs/BENCHMARKS.md`, README attribution, or `NOTICE` (D-08).

## Verification

- `go build ./...` — clean
- `go test -count=1 ./internal/indexer/capability/...` — green (matrix gate)
- `go test -count=1 ./testdata/golden/...` — green (renamed package)

## Deviations

The executor was interrupted (upstream idle timeout) before committing; the rename work was complete on the worktree but uncommitted. The orchestrator verified build + matrix gate + golden package all green, committed the result, and wrote this summary.
