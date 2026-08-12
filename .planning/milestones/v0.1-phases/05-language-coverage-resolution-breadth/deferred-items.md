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
  status: resolved
  resolved_at: 2026-08-10
  resolution: "Closed by maintainer decision during /gsd-audit-uat — not fixed,
  deliberately stopped tracking. Logged three times across two milestones
  (05-06, 05-07, 05-12) and never reproduced outside full-parallel-suite
  contention; passes in isolation every time. Current signal: `task test`
  on 2026-08-10 ran `test:daemon` (64.9s) and `test:race` over
  `./internal/daemon/...` (73.5s), both green — one clean run, not proof of
  non-flakiness. Reopen with a fresh entry if it returns."

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
  status: resolved
  resolved_at: 2026-08-10
  resolution: "Closed with the 05-06 entry above — same finding, second
  sighting. Maintainer decision: stop tracking; reopen if it returns."

## From 05-12 (framework-aware routing, LANG-07)

- **`internal/indexer/resolve.go` has a pre-existing `gofmt -l` violation**
  (`retryConformanceCalls`'s three `map[...]` var-decl comment alignment,
  lines ~262-264) that predates this plan's edit to the same file — verified
  via `git stash`/`gofmt -l` before this plan's own change was applied. This
  plan's own edit (the `r.RelPath == ""` synthetic-result skip in the
  file-record loop) is itself gofmt-clean; the pre-existing misalignment is
  in an unrelated function this plan never touches. Out of scope per the
  executor's scope-boundary rule — left unfixed for whichever plan next
  touches `resolve.go`.
  status: resolved
  resolved_at: 2026-08-10
  resolution: "FIXED — verified 2026-08-10 during /gsd-audit-uat.
  `gofmt -l internal/indexer/resolve.go` returns empty, and
  `retryConformanceCalls` still exists (resolve.go:353), so the file was
  genuinely reformatted rather than the cited function being deleted out
  from under this entry. Stale, not waived."
- **`internal/daemon`'s `TestSoak` flaked again** on a full non-required
  `go test ./... -count=1` sweep (a goroutine panic trace pointing at
  `watch.(*Debouncer).fire`); the plan's own required verification (`go
  test ./internal/indexer/routes/ ./internal/indexer/ -run ... -race
  -count=1` for each task) passed cleanly, and `go test
  ./internal/daemon/... -count=1` in isolation immediately after also
  passed. Same pre-existing timing sensitivity already logged twice above
  (05-06, 05-07) and called out explicitly in this plan's own
  `critical_constraints` ("the pre-existing internal/daemon race flake is
  unrelated (do NOT fix)") — not fixed, out of this plan's file scope.
  status: resolved
  resolved_at: 2026-08-10
  resolution: "Closed with the 05-06 entry above — same finding, third
  sighting. Maintainer decision: stop tracking; reopen if it returns."
