---
phase: 07-migration-tool
plan: 07
subsystem: cli
tags: [cobra, cli-wiring, migration, non-destructive-guard]

# Dependency graph
requires:
  - phase: 07-migration-tool (plan 06)
    provides: "internal/migrate.Run(from, to, Options) (Result, error) — the single-call MIGR-01/MIGR-02 orchestration entry point, plus Result.Report and Result.HealthMessage (D-01 first-sync note)"
provides:
  - "codegraph migrate [--from] [--to] [--force] [--drop-dangling] — the user-facing single-command entry point for MIGR-01"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "CLI-level non-destructive confirmation gate simplified to 'non-empty target -> confirm unless --force', deliberately not replicating migrate.Run's checkTargetOverwrite recognized-prior-migration exemption — avoids duplicating store-open logic at the CLI layer (which would itself create an empty store/ subdirectory as an inspection side effect) while still satisfying D-08's refuse-without---force requirement as a conservative superset; migrate.Run's own checkTargetOverwrite remains the authoritative guard regardless of what the CLI does"
    - "Result.HealthMessage (migrate.go's healthMessage()) is printed verbatim rather than re-derived — the D-01 first-sync/index full-reindex note lives in exactly one place (internal/migrate), the CLI is a thin pass-through"

key-files:
  created:
    - internal/cli/migrate.go
    - internal/cli/migrate_test.go
  modified:
    - internal/cli/root.go

key-decisions:
  - "--from and --to both default to the same path (cwd/.codegraph/) — matching D-08's 'in place, via temp->swap' intent: the common case reads the existing TS index and atomically swaps a new-format store into the SAME directory, relying on migrate.Run keeping the source SQLite file handle open (already-open-file survives unlink/rename on POSIX) until after the swap completes"
  - "The CLI's pre-Run confirmation check only tests 'is the target directory non-empty', not 'is it recognizably a prior healthy migration' — recognizing a prior migration requires opening <target>/store via graphstore.Open, which creates the directory as a side effect if absent; duplicating that check at the CLI layer would risk polluting the target during a mere confirmation probe. migrate.Run's own checkTargetOverwrite already handles the finer-grained recognized-vs-unrecognized distinction once --force is set (interactively or via flag)"
  - "On user confirmation (interactive 'y'), the CLI sets force=true before calling migrate.Run, so Run's own checkTargetOverwrite does not immediately re-refuse the just-approved overwrite"
  - "ErrNotATSSource is rewrapped with a clearer, path-specific message ('%s does not look like a TypeScript CodeGraph index...') via errors.Is + fmt.Errorf(%w) rather than replaced, so callers can still errors.Is(err, migrate.ErrNotATSSource)"

requirements-completed: [MIGR-01, MIGR-02]

coverage:
  - id: D1
    description: "codegraph migrate is a registered subcommand parsing --from/--to/--force/--drop-dangling, delegating to migrate.Run, and printing a reconciliation report"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "internal/cli/migrate_test.go#TestMigrateCmdRegisteredAndFlags"
        status: pass
      - kind: unit
        ref: "internal/cli/migrate_test.go#TestMigrateCmdEndToEnd"
        status: pass
    human_judgment: false
  - id: D2
    description: "A non-TS source fails loud with a clear, path-specific message and a non-zero exit; no target directory is created"
    requirement: MIGR-02
    verification:
      - kind: unit
        ref: "internal/cli/migrate_test.go#TestMigrateCmdFailsLoudOnNonTSSource"
        status: pass
    human_judgment: false
  - id: D3
    description: "A non-empty, unrecognized target refuses without --force (confirm prompt); declining leaves the target's existing contents untouched and writes nothing"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "internal/cli/migrate_test.go#TestMigrateCmdRefusesUnrecognizedTargetWithoutForce"
        status: pass
    human_judgment: false
  - id: D4
    description: "On success the command prints per-table source/migrated count reconciliation and the D-01 first-sync/index full-reindex note; the migrated store opens healthy with HasFileIndex=true"
    requirement: MIGR-01
    verification:
      - kind: unit
        ref: "internal/cli/migrate_test.go#TestMigrateCmdEndToEnd"
        status: pass
    human_judgment: false
  - id: D5
    description: "Registration is additive only — no existing root subcommand reordered or dropped; codegraph migrate --help is reachable and the binary builds"
    requirement: MIGR-01
    verification:
      - kind: manual
        ref: "go run ./cmd/codegraph migrate --help; go run ./cmd/codegraph --help | grep migrate; go build ./...; go test ./... -count=1"
        status: pass
    human_judgment: false

duration: 45min
completed: 2026-07-13
status: complete
---

# Phase 7 Plan 7: CLI Wiring (codegraph migrate) Summary

**`codegraph migrate [--from] [--to] [--force] [--drop-dangling]` is a registered cobra subcommand that resolves D-08's cwd-`.codegraph/`-in-place defaults, gates a non-empty target behind an interactive confirm (or `--force`), delegates entirely to `migrate.Run`, and prints the D-09 count reconciliation plus the D-01 first-sync/index full-reindex note — the single-command surface that closes out MIGR-01.**

## Performance

- **Duration:** ~45 min
- **Completed:** 2026-07-13
- **Tasks:** 2
- **Files modified:** 3 (migrate.go, migrate_test.go — both new; root.go — additive registration)

## Accomplishments
- `newMigrateCmd()` (`internal/cli/migrate.go`): `Use: "migrate"`, `Args: cobra.NoArgs`, four flags (`--from`, `--to`, `--force`/`-f`, `--drop-dangling`), Long text documenting the D-01 first-sync-full-reindex behavior up front
- `--from`/`--to` both default to `targetRoot(nil)`-resolved `cwd/.codegraph` — the common in-place conversion case; either flag can be pointed elsewhere for an out-of-place migration
- Non-destructive-to-target policy: `dirNonEmpty(resolvedTo)` gates an interactive `confirm()` prompt (mirroring `index.go`/`uninit.go`'s established pattern) whenever the target is non-empty and `--force` wasn't passed; declining aborts cleanly with no `migrate.Run` call and no write; confirming sets `force=true` before calling `Run` so its own `checkTargetOverwrite` doesn't immediately re-refuse
- `ErrNotATSSource` is caught via `errors.Is` and rewrapped with a clearer, source-path-specific message while remaining `errors.Is`-matchable
- `printMigrateReport` prints per-table migrated/source counts, a dropped-dangling-count line when `Report.Dropped > 0`, a resumed-migration note when `Result.Resumed`, and `Result.HealthMessage` verbatim (already carries the D-01 note from `internal/migrate`'s `healthMessage()`)
- `root.go`: `newMigrateCmd()` added as a single additive entry to the existing `root.AddCommand(...)` call; doc comment updated to mention `migrate` (D-08)
- Four new tests in `migrate_test.go`: registration+flags, an end-to-end run against a `migratetest.BuildTSIndex(VariantHappy)` fixture producing a healthy new-format store with `HasFileIndex=true`, fail-loud on a hand-built non-TS SQLite file (no target created), and refuse-without-`--force` on a non-empty unrecognized target (declined confirm leaves existing content untouched, no `store/` written)

## Task Commits

Each task was committed atomically:

1. **Task 1: newMigrateCmd — flags, path resolution, non-destructive policy, report** - `5054a0c` (feat)
2. **Task 2: Register migrate on the root command** - `6779f2d` (feat)

## Files Created/Modified
- `internal/cli/migrate.go` - `newMigrateCmd`, `dirNonEmpty`, `printMigrateReport`
- `internal/cli/migrate_test.go` - `TestMigrateCmdRegisteredAndFlags`, `TestMigrateCmdEndToEnd`, `TestMigrateCmdFailsLoudOnNonTSSource`, `TestMigrateCmdRefusesUnrecognizedTargetWithoutForce`
- `internal/cli/root.go` - `newMigrateCmd()` added to `root.AddCommand(...)`; doc comment mentions migrate

## Decisions Made
- The CLI-level pre-`Run` confirmation check tests only "is the target non-empty", not "is it recognizably a prior healthy migration" — replicating the latter would mean opening `<target>/store` via `graphstore.Open` just to inspect it, which creates the store directory as a side effect if it doesn't already exist (Pebble's `Open` creates-if-missing). That would pollute a target directory purely from a confirmation probe, even on the path where the user later declines. The simpler "non-empty -> confirm" check is a conservative superset of D-08's "refuse... unrecognized target" requirement (it also asks for confirmation on a recognized prior migration, which is a minor UX cost, not a correctness gap) and `migrate.Run`'s own `checkTargetOverwrite` remains the authoritative, non-duplicated recognized/unrecognized distinction once `--force` (interactive or flag) is set.
- Confirming the prompt sets the local `force` variable to `true` before calling `migrate.Run`, so a user's interactive "yes" isn't immediately re-refused by `Run`'s own force-gated guard.
- `--from`/`--to` resolve to the same default path (`cwd/.codegraph`) rather than distinct defaults, matching D-08's explicit "in place, via temp→swap" framing — this was confirmed against `migrate.Run`'s architecture (source SQLite handle stays open across the final `atomicSwapDir` call, relying on POSIX unlink/rename-survives-open-handle semantics) rather than assumed.

## Deviations from Plan
None - plan executed exactly as written.

## Issues Encountered
None. `golangci-lint run ./internal/cli/...` reports 16 pre-existing `errcheck` findings across the package (unrelated files: affected.go, callees.go, callers.go, explore.go, index.go, install.go, node.go, cli_test.go); the 2 new findings in `migrate_test.go` (`defer store.Close()` / `defer r.Close()`) exactly match the pre-existing pattern already established in `cli_test.go`'s identical lines — out of scope to change per the scope-boundary rule (only fix issues directly caused by this plan's changes).

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- MIGR-01 and MIGR-02 are both fully closed: `codegraph migrate` is reachable, non-destructive to the source, refuses unrecognized targets without `--force`, and reports the D-01/D-09 reconciliation on success.
- This closes out the migration-tool phase's command surface. No downstream plan currently depends on 07-07's artifacts (`affects: []`).
- No blockers.

---
*Phase: 07-migration-tool*
*Completed: 2026-07-13*

## Self-Check: PASSED

All created files and task commit hashes verified present on disk / in git log (see below).
