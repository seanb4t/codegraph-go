---
phase: 10-index-health-the-coverage-denominator
reviewed: 2026-09-13T05:08:38Z
depth: deep
files_reviewed: 37
files_reviewed_list:
  - internal/graphstore/batch.go
  - internal/graphstore/excludedfile_test.go
  - internal/graphstore/export_test.go
  - internal/graphstore/export.go
  - internal/graphstore/keys.go
  - internal/graphstore/pebble_store.go
  - internal/graphstore/store.go
  - internal/indexer/coverage_fixture_test.go
  - internal/indexer/discover_test.go
  - internal/indexer/discover.go
  - internal/indexer/discoverexclusion_test.go
  - internal/indexer/discoverexclusion.go
  - internal/indexer/pipeline_test.go
  - internal/indexer/pipeline.go
  - internal/indexer/resolve_test.go
  - internal/indexer/resolve.go
  - internal/indexer/sync_coverage_test.go
  - internal/indexer/sync.go
  - internal/indexer/synccoverage_test.go
  - internal/indexer/synccoverage.go
  - internal/query/coverage_test.go
  - internal/query/coverage.go
  - internal/query/errors.go
  - internal/schema/exclusion_test.go
  - internal/schema/exclusion.go
  - internal/schema/graph.proto
  - internal/schema/meta_commit_test.go
  - internal/uiproto/uiv1/ui.proto
  - internal/uiserver/coverage_test.go
  - internal/uiserver/coverage.go
  - internal/uiserver/handlers.go
  - internal/uiserver/readonly_test.go
  - web/src/lib/components/health/CoverageSection.svelte
  - web/src/lib/health-view.ts
  - web/src/routes/health/+page.svelte
  - web/tests/health-page.test.ts
  - web/tests/health-view.test.ts
findings:
  critical: 1
  warning: 0
  info: 3
  total: 4
status: issues_found
---

# Phase 10: Code Review Report

**Reviewed:** 2026-09-13T05:08:38Z
**Depth:** deep
**Files Reviewed:** 37
**Status:** issues_found

## Summary

Final re-review (iteration 3) of `10e9c028`, which replaced the wall-clock, millisecond-resolution `LastSyncUnixMs` generation marker (iteration 2's WR-01 finding) with an additive `Meta.coverage_generation` (field 10, `int64`) monotonic counter, read-modify-written by exactly 1 at each of the three coverage-bearing meta-write sites: `writeGraph`'s from-scratch commit (`internal/indexer/resolve.go:822-841`), and `Sync`'s two mutually-exclusive commit paths (`internal/indexer/sync.go:220`, `:465`).

**Mechanics verified sound in isolation.** `sync.go`'s `meta` (read once via `r0.GetMeta()` at the top of `Sync`, line 75) is never mutated in place before either write site reads `meta.GetCoverageGeneration()`, and `newMeta` is always a fresh `schema.NewMeta()` — so both increments are genuine read-modify-writes of the snapshot value captured at the start of that `Sync` call, not a self-referential read of an already-mutated struct. `writeGraph` performs its own `store.Snapshot()` → `GetMeta()` read strictly before opening its `Writer` (`resolve.go:822-833`), so a from-scratch rewrite delegated from `Sync`'s D-02b backfill path (which opens the SAME already-populated store, never a fresh one) also increments correctly rather than resetting. `CoverageRows`' `ErrAborted` check (`coverage.go:211-213`) keys purely off `meta.GetCoverageGeneration()`, confirmed by `TestCoverageRowsGenerationCheckKeysOffCounterNotClock` (`internal/query/coverage_test.go:749-813`) to be genuinely independent of `LastSyncUnixMs`. An old graph (field unset → 0, `has_coverage` false) still short-circuits to `Known: false` before any token logic runs, so no compat gap. The schema field-number stability test (`internal/schema/meta_commit_test.go:119`) pins field 10. `TestCoverageRowsSurvivesCursorMutationBetweenPages` and `TestCoverageSourceNeverWalksDisk` are present and green; `internal/query/archtest` still forbids importing `internal/indexer`; `internal/uiserver/readonly_test.go`'s `wantUIServiceMethods` fixture is still 16 entries. `go build ./...`, `go vet ./...`, `go test ./internal/query/... ./internal/graphstore/... ./internal/uiserver/... ./internal/indexer/... ./internal/schema/...` all pass. `task web:drift` and `task proto:drift` both PASS (byte-identical). `internal/uiproto/uiv1/ui.proto` is untouched since `725ba1fb`.

**But the fix's own justification — "a plain counter guarantees two distinct writes imply two distinct generations regardless of clock resolution or backward clock steps" — does not hold once the store itself is wiped and rebuilt, which this codebase does routinely via `codegraph index`.** See CR-01 below: this is a real, deterministic aliasing path that the task-provided verification checklist specifically asked this iteration to check for, and it is present. WR-01's closure is therefore **not genuine** — the counter is only monotonic for the lifetime of one on-disk store directory, and a full re-index (`codegraph index`, `os.RemoveAll(storeDir)` then rebuild) resets it back to 1, which is *exactly* the generation value most in-flight page tokens will already be carrying.

IN-01, IN-02, and IN-03 from the prior review are unchanged and carried forward as Info (all pre-existing/low-severity, verified against current source).

## Critical Issues

### CR-01: `Meta.coverage_generation` resets to 1 on every full re-index, defeating the exact aliasing guarantee WR-01 was built to provide

**File:** `internal/indexer/resolve.go:809-841`, `internal/cli/index.go:52-59`, `internal/query/coverage.go:198-213`
**Issue:** `writeGraph`'s read-modify-write of `CoverageGeneration` (`resolve.go:809-819`) is explicitly designed around "a from-scratch rewrite over an EXISTING store" reading the prior generation back via `store.Snapshot()` before opening its `Writer` — and the code comment even cites `internal/cli/index.go`'s `RemoveAll` by name, but only to reason about the (already-handled) `c/`-namespace clear, not about its effect on this counter.

`newIndexCmd` (`internal/cli/index.go:52-59`) is `codegraph index`'s full-rebuild path — the one non-incremental way to (re)build a graph — and it unconditionally does:
```go
if err := os.RemoveAll(storeDir); err != nil { return err }
if err := os.MkdirAll(storeDir, 0o755); err != nil { return err }
```
before calling `indexer.Run`. This deletes the ENTIRE pebble store directory, `Meta` included. When `writeGraph` then runs `store.Snapshot()` → `GetMeta()` against this genuinely empty directory, it gets `ErrNotFound`, so `priorGeneration = 0` and the commit stamps `CoverageGeneration = 1` — every single time `codegraph index` is run, regardless of how high the generation had climbed in the store that was just deleted.

This means the counter is monotonic only *within the lifetime of one store directory*, not globally, and every store's *first* coverage-bearing commit — i.e. any freshly-`index`ed repo that has not yet had a `sync` run against it — is at generation 1. That is an extremely common, not a rare, state (the reasonable UI-viewing sequence "run `codegraph index`, then open `/health`" lands exactly there). Concrete reproduction (mechanically traced, not merely hypothesized):

1. `codegraph index` on a repo with >1000 coverage-relevant rows (extraction failures + excluded files combined — `CoverageDefaultPageSize`/`COVERAGE_PAGE_SIZE` is 1000) → store A, `CoverageGeneration = 1`.
2. A client opens `/health`, `GetCoverage` page 1 returns a token embedding `generation=1`.
3. Before the client fetches page 2, the operator re-runs `codegraph index --force` (a full rebuild — not `codegraph sync`, which is the incremental path this mechanism was actually validated against) because, e.g., files changed and a clean rebuild was chosen over an incremental sync. `RemoveAll` wipes store A; the rebuilt store B's first coverage commit again stamps `CoverageGeneration = 1` (there is no prior `Meta` to read — same code path as step 1).
4. The client's page-2 request replays its stale token, `cursorGeneration=1`. `CoverageRows`' check (`coverage.go:211`) is `cursorGeneration != generation` → `1 != 1` is `false` → the check **passes**, no `ErrAborted`, and `CoverageRows` proceeds to resume the position-based walk (`bytes.Compare(fit.RawKey(), cursorKey)`) against store B — an entirely different graph than the one page 1 was drawn from. The client silently accumulates rows from two unrelated index generations into what it believes is one consistent walk, with no signal that anything went wrong.

This is not the sub-millisecond wall-clock race iteration 2 flagged (which required two commits to land in the same host millisecond); it is a deterministic collision that fires on the very first generation value every rebuilt store ever produces, and it fires specifically on the full-rebuild path this fix's own regression tests never exercise (`TestCoverageGenerationIncrementsByExactlyOnePerCommit` and `TestWriteGraphStampsMonotonicCoverageGeneration` both test increments against a store that is never wiped between calls; nothing in the diff drives `os.RemoveAll` + a second `writeGraph` and asserts the generation is distinct from the first run's).
**Fix:** Persist the generation counter's identity independently of the store directory's own lifetime, or detect the wipe. Two viable approaches:
1. Seed a new store's very first `CoverageGeneration` from something that survives `RemoveAll` — e.g. a random 63-bit high-order salt written once at `codegraph init` time (outside `storeDir`, or into a file `index`'s `RemoveAll` does not touch) OR-ed / added into the counter, so two independently-initialized stores can never coincidentally start at the same value.
2. Simpler and more robust: stop wiping `Meta.coverage_generation`'s effective history at all — have `codegraph index`'s `RemoveAll` step preserve (or `index.go` itself pass through) the prior store's last known `CoverageGeneration` into the rebuild, e.g. by reading it before `RemoveAll` and threading it into `indexer.Run`/`writeGraph` as a floor, mirroring exactly the "read prior, increment" discipline `writeGraph` already applies for the backfill-over-an-existing-store case — just extended to cover the CLI's own wipe-then-rebuild sequence.
Either way, add a regression test that does two full rebuilds of the same `storeDir` (mirroring `codegraph index`'s actual `RemoveAll` + `Run` sequence, not just two `writeGraph` calls against a store that was never wiped) and asserts the second rebuild's `CoverageGeneration` is NOT equal to a value a client could plausibly still be holding a token for.

## Info

### IN-01: `coverageExtractionDetail`'s absolute-path scrub is a plain substring replace, not anchored to a path boundary

**File:** `internal/query/coverage.go:337-343`
**Issue:** Unchanged from prior reviews. `strings.ReplaceAll(detail, repoRoot, ".")` replaces every literal occurrence of `repoRoot`, not just a leading-path occurrence, and depends on `e.repoRoot`'s capitalization/symlink-resolution matching whatever was embedded in the underlying parser error verbatim (e.g. a `/tmp` vs `/private/tmp` class of mismatch on macOS). Carried forward as low-severity: pre-existing pattern, not introduced by this phase.
**Fix:** No action required unless a broader repoRoot-redaction audit is already planned.

### IN-02: `unsupportedExtensionDetail`/exclusion `detail` strings are unbounded on the write path

**File:** `internal/indexer/discoverexclusion.go:78-100`
**Issue:** Unchanged from prior reviews. No explicit byte bound is applied at write time the way `coverageDetailMaxBytes` bounds the read-time `Detail` for extraction failures. Low-severity: filesystem path-component limits already bound this in practice.
**Fix:** No action required now; noted for completeness since coverage rows are now exposed over the wire to a browser.

### IN-03: `fetchAllCoverageRows`'s second (retry) attempt discards any rows it collected before aborting a second time

**File:** `web/src/lib/health-view.ts:335-365`
**Issue:** Unchanged from prior reviews. When the retried walk itself throws `Code.Aborted` partway through (e.g. after collecting several pages, then hitting a third concurrent Sync), the final `.catch` returns a hardcoded `{ known: true, rows: [], incomplete: true }` rather than whatever partial `rows` the second attempt had already accumulated. Not a correctness bug (`incomplete: true` already tells the UI not to trust the list), just a minor, easily-avoidable loss of otherwise-good partial data.
**Fix:** Optional: thread the partially-collected rows out of the second `attempt()` call. Not required before shipping.

---

_Reviewed: 2026-09-13T05:08:38Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
