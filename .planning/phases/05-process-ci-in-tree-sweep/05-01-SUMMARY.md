---
phase: 05-process-ci-in-tree-sweep
plan: 01
subsystem: cli-storage
tags: [migrate, cli, graphstore, golden-fixtures, go-mod, cobra, pebble]

requires:
  - phase: 03-corpus-selection-golden-re-freeze
    provides: the current golden harness / TestGoldenFixturesExist scaffold this plan edits
provides:
  - "codegraph migrate command removed entirely from the Cobra tree (no registration, no help/man entry)"
  - "internal/migrate/ package deleted (4,018 lines incl. archtest, migratetest fixture)"
  - "modernc.org/sqlite and its libc/mathutil/memory indirect deps removed from go.mod/go.sum"
  - "graphstore migration-progress cursor API (GetMigration/PutMigration/migrationRecordName) deleted, including 7 test-fake stubs and the 173-line migration_record_test.go"
  - "testdata/golden/ts-{schema.sql,schema.dump.sql,version.txt} fixtures and their two TestGoldenFixturesExist subtests deleted"
  - "README.md, testdata/golden/README.md, internal/indexer/nodeid/nodeid.go doc references to the removed migrate capability/fixtures reworded"
affects: [05-02, 05-03, 05-04, 05-05, 05-06]

actuals:
  tokens: 70747
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Whole-capability removal in one coherent multi-commit diff: command -> package -> dependency -> downstream API -> fixtures -> docs, each layer its own atomic commit but no layer left half-done"

key-files:
  created: []
  modified:
    - internal/cli/root.go
    - internal/graphstore/store.go
    - internal/graphstore/pebble_store.go
    - internal/graphstore/batch.go
    - internal/graphstore/keys.go
    - internal/indexer/resolve_test.go
    - internal/query/expand_test.go
    - internal/query/gather_test.go
    - internal/query/scoring_test.go
    - internal/query/search_test.go
    - internal/query/seeding_test.go
    - internal/query/traverse_test.go
    - testdata/golden/golden_test.go
    - testdata/golden/README.md
    - internal/indexer/nodeid/nodeid.go
    - README.md
    - go.mod
    - go.sum

key-decisions:
  - "Followed maintainer ruling D-04 (2026-08-15, CODE-03 amended): codegraph migrate is removed entirely, not reframed — reverses the milestone's standing 'sweep removes framing, never capability' rule for this one capability, as directed."
  - "go mod tidy required -e: a pre-existing, unrelated tree-sitter-swift test-dependency resolution error (documented across 6+ prior SUMMARYs in this repo's history, e.g. 02-04-SUMMARY.md, 06-01-SUMMARY.md) blocks plain go mod tidy. -e let tidy complete the modernc.org prune correctly; confirmed the resulting go.mod diff only touches expected direct/indirect bookkeeping."
  - "internal/daemon's two test failures (TestRunWatchdogCancelsRunOnSimulatedReparent, TestDaemonSharedWriter) in the full go test -count=1 ./... run are the pre-existing, documented 'daemon extreme-load timeout tail' accepted limitation (STATE.md/PROJECT.md: '52/52 real CI runs clean — the flake was always local'; maintainer ruled CI load the governing standard for MAINT-02). internal/daemon has zero relationship to migrate/graphstore-cursor/ts-fixtures; a separate go test -count=1 run over every OTHER package (49/50, explicitly excluding internal/daemon) passed clean, and testdata/golden was verified green explicitly and separately. Not a regression introduced by this plan."

patterns-established:
  - "Layered capability-removal diff: task 1 cuts command+package+dependency, task 2 cuts the downstream storage-layer API the package alone consumed, task 3 cuts the fixtures/docs that described the whole thing — each task is its own atomic, individually-verified commit, reused by later plans in this phase (05-02..05-06) that will remove other in-tree surfaces."

requirements-completed: [CODE-03]

coverage:
  - id: D1
    description: "codegraph migrate command, internal/migrate package (4,018 lines), and internal/cli/migrate.go + migrate_test.go (258 lines) deleted; root.go registration and doc comments reworded; modernc.org/sqlite + libc/mathutil/memory pruned from go.mod/go.sum via go mod tidy -e"
    requirement: CODE-03
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./... (exit 0)"
        status: pass
      - kind: unit
        ref: "rg -c modernc go.mod go.sum -> 0; rg -ln internal/migrate internal/ --glob '*.go' -> 0 (after task 2 lands, see Deviations)"
        status: pass
      - kind: e2e
        ref: "codegraph help / codegraph --help contain no migrate subcommand entry"
        status: pass
    human_judgment: false
  - id: D2
    description: "graphstore migration-progress cursor API (Reader.GetMigration, Writer.PutMigration, pebble/batch impls, migrationRecordName meta-key const) deleted along with migration_record_test.go and the 7 test-fake stubs across internal/indexer and internal/query"
    requirement: CODE-03
    verification:
      - kind: unit
        ref: "rg -c 'GetMigration|PutMigration|migrationRecordName' internal/ --glob '*.go' -> 0"
        status: pass
      - kind: unit
        ref: "go test -count=1 ./internal/graphstore/... ./internal/indexer/... ./internal/query/..."
        status: pass
    human_judgment: false
  - id: D3
    description: "testdata/golden/ts-{schema.sql,schema.dump.sql,version.txt} fixtures deleted with the two TestGoldenFixturesExist subtests that hard-failed on their absence; testdata/golden/README.md, internal/indexer/nodeid/nodeid.go, and README.md's Commands table reworded to drop references to the removed fixtures/command"
    requirement: CODE-03
    verification:
      - kind: unit
        ref: "go test -count=1 ./testdata/golden/... (explicit run, not hidden behind ./...)"
        status: pass
      - kind: unit
        ref: "test ! -e testdata/golden/ts-*; rg -n 'migrate' README.md -> 0"
        status: pass
    human_judgment: false

duration: ~25min
completed: 2026-08-15
status: complete
---

# Phase 5 Plan 1: Remove codegraph migrate End-to-End Summary

**Deleted the entire `codegraph migrate` capability — command, `internal/migrate` package, `modernc.org/sqlite` dependency, graphstore migration-cursor API, and the ts-* golden fixtures — in three atomic, individually-verified commits (plus one staging-bug fixup), per maintainer ruling D-04.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-08-15 (session start)
- **Completed:** 2026-08-16T00:03:28Z
- **Tasks:** 3/3 completed
- **Files modified:** 42 (21 deleted, 21 edited)

## Accomplishments

- `codegraph migrate` no longer exists as a Cobra command: `internal/cli/root.go`'s `AddCommand` list and both doc comments no longer mention it; `internal/cli/migrate.go`/`migrate_test.go` and the whole `internal/migrate/` package (migrate.go, reader.go, translate.go, swap.go, validate.go, progress.go, batchwriter_test.go, all `_test.go`, `migratetest/` fixture, `archtest/modernc_confinement_test.go`) are deleted — 4,276 lines total.
- `modernc.org/sqlite` and its indirect closure (`modernc.org/{libc,mathutil,memory}`) are gone from `go.mod`/`go.sum` — the sole reason that dependency existed. `go mod tidy -e` was required due to a pre-existing, unrelated `tree-sitter-swift` test-dependency resolution error (documented repo history); the resulting diff is exactly the expected direct/indirect require bookkeeping.
- The graphstore migration-progress cursor API (`Reader.GetMigration`, `Writer.PutMigration`, the pebble/batch impls, the `migrationRecordName = "migration"` meta-key const) is deleted along with the 173-line `migration_record_test.go` and all 7 test-fake stubs (1 `PutMigration` in `internal/indexer/resolve_test.go`, 6 `GetMigration` across `internal/query/{expand,gather,scoring,search,seeding,traverse}_test.go`).
- The three `testdata/golden/ts-*` legacy-schema fixtures are deleted along with the two `TestGoldenFixturesExist` subtests that hard-failed on their absence; the third corpus-JSON subtest is untouched. `go test -count=1 ./testdata/golden/...` (the explicit run that would catch a fixture/subtest mismatch — `go test ./...` alone excludes `testdata/`) is green.
- Doc references to the removed capability/fixtures are reworded on their own terms: `testdata/golden/README.md`'s ts-fixture bullet and its TS-project historical-bug note (edge-dedup rationale kept, TS-specific framing dropped), `internal/indexer/nodeid/nodeid.go`'s id-shape comment (dropped the `ts-schema.dump.sql` citation), and `README.md`'s Commands table (`migrate` row removed).

## Task Commits

Each task was committed atomically:

1. **Task 1 (tracer): Cut the command, package, dep — codegraph migrate and internal/migrate and modernc go** — `d512cef` (feat)
2. **Task 2: Cut the graphstore migration cursor — Reader/Writer methods, impls, const, test, and the 7 fakes** — `4affd9d` (feat)
3. **Task 3: Delete the ts-* fixtures and their golden-suite subtests, golden/README, nodeid comment, README row** — `15b195d` (feat) + `736a8e9` (feat, staging-bug fixup — see Deviations)

**Plan metadata:** this commit (docs: complete plan)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - broken verify gate] Task 1's `<automated>` verify block contained an unescaped `$2` inside a nested double-quoted `awk` script**
- **Found during:** Task 1 verification
- **Issue:** `rg -c "modernc" go.mod go.sum | awk -F: "{s+=$2} END {print s+0}"` — the `$2` inside the double-quoted awk script is prematurely expanded by the outer shell as a positional parameter (empty), producing the invalid awk program `{s+=} END {print s+0}` and an `awk: syntax error`. This is the same class of never-green shell-quoting defect the cycle-3 convergence review found and fixed in 05-03/05-04, but this instance in 05-01 was not caught (the review's "also verified clean" note for 05-01 checked for stray apostrophes, not `$`-in-double-quotes).
- **Fix:** Ran the corrected equivalent (`\$2` to pass the field variable through literally) to actually perform the intended verification. Did not edit PLAN.md (tool-owned generated artifact — the fix is in how the check is invoked, not the file).
- **Verified:** the corrected command ran cleanly and reported `modernc references: 0`.
- **Files modified:** none (verification-only).
- **Commit:** N/A (no file change).

**2. [Rule 1 - plan-level verify scope mismatch] Task 1's own `<verify>` block, and the plan's overall `<verification>` section ("post-task-1"), both assert `rg -ln internal/migrate internal/` is empty immediately after task 1 alone — but graphstore's comments referencing `internal/migrate` (in `store.go`, `pebble_store.go`, `batch.go`, `keys.go`) are explicitly task 2's scope, not task 1's.**
- **Found during:** Task 1 verification (the corrected verify command above reported 4 remaining importers, not 0)
- **Issue:** `rg -ln "internal/migrate" internal/ --glob "*.go"` after task 1 alone returns 5 hits, all in graphstore comments describing the (still-present-at-that-point) migration cursor API, which task 2 explicitly removes in the same change as its comments. The plan's task split makes task 1's own zero-hits claim structurally impossible to satisfy before task 2 runs.
- **Fix:** Verified task 1's real intent instead — "no remaining Go import of internal/migrate" (confirmed via `go build ./...` succeeding, which a broken import would fail) and confirmed the residual grep hits were exclusively in files task 2 explicitly scopes. Re-ran the same grep after task 2 completed: 0 hits, confirming the plan's overall `<verification>` claim becomes true only once both tasks 1 and 2 have landed.
- **Verified:** `rg -ln "internal/migrate" internal/ --glob "*.go" | wc -l` returned `0` after task 2's commit.
- **Files modified:** none (verification-only; task 2's own file changes already covered the fix).
- **Commit:** N/A (no file change beyond what task 2 already committed).

**3. [Rule 1 - bug] `git add` staging failure split task 3 into two commits**
- **Found during:** Task 3 commit
- **Issue:** `git add README.md internal/indexer/nodeid/nodeid.go testdata/golden/README.md testdata/golden/golden_test.go testdata/golden/ts-schema.dump.sql ...` — the last three pathspecs were already `git rm`-staged deletions, and one of them (a `git rm`'d path re-listed in a plain `git add`) triggered `fatal: pathspec ... did not match any files`, which aborted the whole `git add` invocation before it staged the four still-unstaged edits. The resulting commit (`15b195d`) landed with only the 3 fixture deletions; the edits to `golden_test.go`, `testdata/golden/README.md`, `nodeid.go`, and `README.md` were still unstaged.
- **Fix:** Re-ran `git add` scoped only to the still-modified files and created a follow-up commit (`736a8e9`) completing task 3.
- **Verified:** `git status --short` clean after the second commit; `git diff --stat 524fe61 HEAD` matches the plan's artifact table exactly (7 files: README.md, nodeid.go, testdata/golden/README.md, testdata/golden/golden_test.go, and the 3 deleted ts-* fixtures).
- **Files modified:** README.md, internal/indexer/nodeid/nodeid.go, testdata/golden/README.md, testdata/golden/golden_test.go.
- **Commit:** `736a8e9`.

### Pre-existing, unrelated failure (not a deviation caused by this plan)

`go test -count=1 ./...` reports `FAIL github.com/seanb4t/codegraph-go/internal/daemon` (`TestRunWatchdogCancelsRunOnSimulatedReparent`, `TestDaemonSharedWriter`) — this is the repo's documented "daemon extreme-load timeout tail," accepted as a non-blocking, load-dependent local-environment characteristic (PROJECT.md: "52/52 real CI runs clean — the flake was always local"; MAINT-02 governing standard is CI load, not local). `internal/daemon` has zero import relationship to `internal/migrate`, the graphstore migration cursor, or the ts-* fixtures this plan touches. A separate `go test -count=1` run over every other package (49/50, `internal/daemon` explicitly excluded) passed clean, and `go test -count=1 ./testdata/golden/...` was independently confirmed green. Not fixed (pre-existing, out of this task's scope per the deviation-rule scope boundary; a fix would require the accepted daemon watchdog/lock-contention behavior to be revisited, which is unrelated to CODE-03).

## Self-Check

- `internal/migrate/` — MISSING (deleted, as expected): confirmed via `test -d internal/migrate` → false
- `internal/cli/migrate.go` — MISSING (deleted, as expected)
- `d512cef` — FOUND: `git log --oneline --all | grep d512cef`
- `4affd9d` — FOUND: `git log --oneline --all | grep 4affd9d`
- `15b195d` — FOUND: `git log --oneline --all | grep 15b195d`
- `736a8e9` — FOUND: `git log --oneline --all | grep 736a8e9`
- `testdata/golden/ts-schema.sql`, `ts-schema.dump.sql`, `ts-version.txt` — MISSING (deleted, as expected)
- `rg -c "GetMigration|PutMigration|migrationRecordName" internal/ --glob '*.go'` — 0
- `rg -c "modernc" go.mod go.sum` — 0

## Self-Check: PASSED
