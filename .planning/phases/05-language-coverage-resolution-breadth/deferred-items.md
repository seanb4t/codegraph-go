# Deferred Items — Phase 5

Out-of-scope discoveries logged during plan execution, per the executor's
scope-boundary rule (only auto-fix issues directly caused by the current
task's changes).

## From 05-06 (Python extraction + resolution)

- **`internal/daemon`'s `TestSoak` is flaky under `-race` + full-parallel-suite
  contention.** Observed failing once during a full `go test -race ./...`
  run, then passing 3/3 in isolation and on a full-suite retry. Not caused
  by this plan's changes — `internal/daemon` has no dependency on
  `internal/indexer/pyextract` or `languages_python.go`. Pre-existing
  timing sensitivity, out of this plan's file scope (`internal/daemon`
  is not in `files_modified`). Left unfixed; flag for whichever plan next
  touches `internal/daemon`'s test suite.

## From 05-07 (TypeScript/TSX/JavaScript extraction + resolution)

- **`internal/daemon`'s `TestSoak` flaked again**, this time under a plain
  (non-`-race`) full-suite `go test ./... ./testdata/golden/... -count=1`
  run; the plan's own required verification (`go test -race
  ./internal/indexer/tsextract/ ./internal/indexer/ ./testdata/golden/
  ...` and the full `-race ./...` sweep) both passed cleanly — this was
  observed only on a broader, non-required ad-hoc full-suite check.
  `internal/daemon` has no dependency on `internal/indexer/tsextract` or
  `languages_typescript.go`; re-ran `go test ./internal/daemon/... -run
  TestSoak -count=1 -v` in isolation and it passed. Same pre-existing
  timing sensitivity already logged above from 05-06 — not fixed, out of
  this plan's file scope.
