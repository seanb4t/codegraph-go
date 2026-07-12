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
