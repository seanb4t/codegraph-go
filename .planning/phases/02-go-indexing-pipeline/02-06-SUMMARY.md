---
phase: 02-go-indexing-pipeline
plan: 06
subsystem: cli
tags: [cobra, cli, init, index, uninit, indexing-pipeline]

# Dependency graph
requires:
  - phase: 02-go-indexing-pipeline (plan 05)
    provides: internal/indexer.Run(repoRoot, storeDir string, opts Options) (Stats, error) — the single from-scratch pipeline entrypoint this CLI drives
  - phase: 01 (all plans)
    provides: internal/graphstore.GraphStore.Open/Close/Snapshot, internal/schema.Meta (node/edge counts)
provides:
  - "codegraph init/index/uninit Cobra command tree (internal/cli), the user-facing lifecycle surface for INDX-01/INDX-02"
  - "cmd/codegraph/main.go binary entrypoint"
  - "spf13/cobra v1.10.2 as a direct go.mod dependency"
affects: [phase-3-mcp-queries, phase-8-migration-cli, cli-surface]

# Tech tracking
tech-stack:
  added: ["github.com/spf13/cobra v1.10.2"]
  patterns:
    - "CLI commands are pure Cobra glue over indexer.Run — no extraction/resolution logic lives in internal/cli"
    - "Package-level error sentinels (cli.ErrAlreadyInitialized, cli.ErrNotInitialized) in the `errors.New(\"cli: ...\")` style, checked via errors.Is, wrapped with user-facing guidance via fmt.Errorf(\"%w: ...\")"
    - "confirm() reads/writes through cmd.InOrStdin()/OutOrStdout() (not os.Stdin/os.Stdout directly) so tests can drive interactive prompts without a subprocess"
    - "A from-scratch `index --force` rebuild clears the store subdirectory before re-running the pipeline, guaranteeing determinism never depends on prior store state"

key-files:
  created:
    - internal/cli/root.go
    - internal/cli/init.go
    - internal/cli/index.go
    - internal/cli/uninit.go
    - internal/cli/cli_test.go
    - cmd/codegraph/main.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "The Pebble store lives at .codegraph/store/ (D-01b); init also writes .codegraph/.gitignore containing \"*\" (A4) rather than editing the repo's own root .gitignore, so init never risks corrupting a file it doesn't own"
  - "index without --force prompts interactively via a shared confirm() helper (also used by uninit) rather than requiring --force unconditionally, matching D-01a's stated 'without --force on an existing store, prompt/confirm' contract"
  - "cobra was promoted from an indirect to a direct go.mod require via `go get` + manual editing of the require blocks, explicitly avoiding `go mod tidy` per the plan's instruction (it would strip Phase 1's pre-pinned unimported deps)"

requirements-completed: [INDX-01, INDX-02]

coverage:
  - id: D1
    description: "`codegraph init` creates .codegraph/ + store subdir and runs indexer.Run in one step, reporting non-zero files/nodes/edges"
    requirement: INDX-01
    verification:
      - kind: integration
        ref: "internal/cli/cli_test.go#TestInitIndexUninit/init_creates_.codegraph/_and_builds_the_graph_in_one_step"
        status: pass
    human_judgment: false
  - id: D2
    description: "`codegraph init` on an already-initialized directory returns ErrAlreadyInitialized and leaves the existing store's node/edge counts unchanged"
    requirement: INDX-01
    verification:
      - kind: integration
        ref: "internal/cli/cli_test.go#TestInitIndexUninit/init_a_second_time_errors_and_does_not_alter_the_existing_store"
        status: pass
    human_judgment: false
  - id: D3
    description: "`codegraph index --force` is a deterministic from-scratch rebuild — two consecutive runs against the same tree produce identical node/edge counts"
    requirement: INDX-02
    verification:
      - kind: integration
        ref: "internal/cli/cli_test.go#TestInitIndexUninit/index_--force_is_a_deterministic_from-scratch_rebuild"
        status: pass
    human_judgment: false
  - id: D4
    description: "`codegraph index` against an uninitialized directory returns ErrNotInitialized"
    requirement: INDX-01
    verification:
      - kind: integration
        ref: "internal/cli/cli_test.go#TestInitIndexUninit/index_on_an_uninitialized_directory_errors"
        status: pass
    human_judgment: false
  - id: D5
    description: "`codegraph uninit` requires confirmation unless --force: declining leaves .codegraph/ in place, --force removes it cleanly via os.RemoveAll scoped to the resolved target"
    requirement: INDX-01
    verification:
      - kind: integration
        ref: "internal/cli/cli_test.go#TestInitIndexUninit/uninit_without_--force_and_without_confirmation_does_not_remove"
        status: pass
      - kind: integration
        ref: "internal/cli/cli_test.go#TestInitIndexUninit/uninit_--force_removes_.codegraph/_cleanly"
        status: pass
    human_judgment: false
  - id: D6
    description: "--quiet suppresses the end-of-run summary entirely; --verbose emits more output than the default (unresolved/skipped counts)"
    requirement: INDX-02
    verification:
      - kind: integration
        ref: "internal/cli/cli_test.go#TestInitIndexUninit/--quiet_suppresses_the_summary;_--verbose_adds_detail_beyond_default"
        status: pass
    human_judgment: false
  - id: D7
    description: "cmd/codegraph builds into a runnable binary; main() only delegates to cli.Execute()"
    requirement: INDX-01
    verification:
      - kind: unit
        ref: "go build ./cmd/codegraph"
        status: pass
    human_judgment: false

# Metrics
duration: 4min
completed: 2026-07-11
status: complete
---

# Phase 2 Plan 06: CLI Lifecycle (init/index/uninit) Summary

**`codegraph` Cobra command tree (`init`/`index`/`uninit`) wired directly to `indexer.Run`, giving users the one-step "init builds the graph" and deterministic "`index --force` rebuild" lifecycle over the Phase 2 pipeline — INDX-01/INDX-02 shipped and integration-tested over the real command surface.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-07-11T02:35:15Z
- **Completed:** 2026-07-11T02:39:04Z
- **Tasks:** 2
- **Files modified:** 8 (6 created, 2 modified)

## Accomplishments
- `internal/cli/root.go`: `newRootCmd()` builds the `codegraph` root command and attaches `init`/`index`/`uninit`; `Execute()` is the sole entrypoint the binary calls. `ErrAlreadyInitialized`/`ErrNotInitialized` are package sentinels in the `errors.New("cli: ...")` style.
- `internal/cli/init.go`: `init` creates `.codegraph/` + `.codegraph/store/`, writes a self-contained `.codegraph/.gitignore` (`*`), and calls `indexer.Run` — one step (INDX-01). On an existing `.codegraph/` it returns `ErrAlreadyInitialized` wrapped with guidance, never touching the existing store (D-01a).
- `internal/cli/index.go`: `index` requires an existing `.codegraph/` (`ErrNotInitialized` otherwise); `--force` clears the store subdirectory and reruns the pipeline from scratch without prompting (INDX-02 determinism); without `--force` it prompts for confirmation via the shared `confirm()` helper.
- `internal/cli/uninit.go`: `uninit` removes `.codegraph/` via `os.RemoveAll` scoped to the resolved target directory, requiring confirmation unless `--force`; a missing `.codegraph/` exits cleanly with a message. `confirm()` reads/writes through `cmd.InOrStdin()`/`cmd.OutOrStdout()` so tests can drive it without a subprocess.
- `--quiet`/`--verbose`/`--workers` flags map onto `indexer.Options`; `printSummary` prints the concise files/nodes/edges/duration line by default, nothing under `--quiet`, and an extra unresolved/skipped line under `--verbose`.
- `cmd/codegraph/main.go`: thin `main()` delegating to `cli.Execute()`, `os.Exit(1)` on error.
- `internal/cli/cli_test.go` (`TestInitIndexUninit`): drives the real command tree (`newRootCmd` + `SetArgs`/`SetOut`/`SetErr`/`SetIn`) against a copied `internal/indexer/testdata/gofixture` tree — all 7 subtests (create-in-one-step, error-on-reinit without store mutation, `--force` determinism via equal node/edge counts, error-when-uninitialized, uninit decline vs `--force`, quiet/verbose output difference) **passed on first run** against Task 1's already-implemented CLI wiring.
- `spf13/cobra` promoted to a direct `go.mod` require via `go get github.com/spf13/cobra@v1.10.2` plus manual require-block edits — `go mod tidy` was never run, per the plan's explicit instruction to avoid stripping Phase 1's pre-pinned unimported deps.
- `go build ./...`, `go build ./cmd/codegraph`, `go vet ./...`, and `go test ./... -count=1` all pass.

## Task Commits

Each task was committed atomically:

1. **Task 1: Cobra command tree — init/index/uninit with flags + D-01a/D-01b semantics** - `8d12f84` (feat)
2. **Task 2: Binary entrypoint + CLI integration tests (INDX-01, INDX-02 flag behavior)** - `0650557` (test)

**Plan metadata:** (pending, this commit)

_Note: Task 2's acceptance criteria call for a RED `test(02-06):` commit preceding a GREEN `feat(02-06):` commit "for any flag wiring the tests forced." No flag wiring was forced — all `TestInitIndexUninit` subtests passed on first run against Task 1's already-implemented behavior, mirroring the 02-05 precedent for test-only gates against already-shipped code. Committed as a single `test(02-06):` commit rather than manufacturing an artificial RED phase._

## Files Created/Modified
- `internal/cli/root.go` - `newRootCmd`, `Execute`, `ErrAlreadyInitialized`, `ErrNotInitialized`
- `internal/cli/init.go` - `newInitCmd`, `targetRoot`, `writeGitignoreHint`, `printSummary`, `codegraphDirName`/`storeDirName` constants
- `internal/cli/index.go` - `newIndexCmd`
- `internal/cli/uninit.go` - `newUninitCmd`, `confirm`
- `internal/cli/cli_test.go` - `TestInitIndexUninit`, `copyFixture`, `execCmd`/`execCmdWithInput`, `readGraphCounts`
- `cmd/codegraph/main.go` - binary entrypoint (`main`)
- `go.mod` / `go.sum` - `github.com/spf13/cobra v1.10.2` promoted to a direct require

## Decisions Made
- `.codegraph/store/` as the Pebble store subdirectory (D-01b, executor's discretion) plus a self-contained `.codegraph/.gitignore` (`*`) rather than editing the repo's own root `.gitignore` (A4).
- `index` without `--force` prompts interactively (shared `confirm()` helper with `uninit`) rather than hard-requiring `--force`, matching D-01a's "prompt/confirm" wording literally.
- Moved `spf13/cobra`/kept `spf13/pflag`/`inconshreveable/mousetrap` split manually in `go.mod` (cobra to the direct block, its own transitive deps left indirect) instead of running `go mod tidy`, per the plan's explicit constraint.

## Deviations from Plan

None - plan executed exactly as written. `go get github.com/spf13/cobra@v1.10.2` initially added cobra to the `// indirect` block (since nothing imported it yet); this was corrected by manually moving the `cobra` line to the direct `require (...)` block after writing the importing code, without running `go mod tidy` — this is the plan's own instructed path ("add the require line + run `go mod download`/`go get` ... do NOT run `go mod tidy`"), not a deviation from it.

## Issues Encountered

None. All integration tests passed on first run; no flag-wiring gaps were found.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- The full `codegraph init`/`index`/`uninit` lifecycle (INDX-01, INDX-02) is shipped and verified end-to-end over the real command surface, closing out Phase 2's final plan.
- `internal/cli` and `cmd/codegraph` are stable surfaces Phase 3 (MCP query layer) and Phase 8 (migration CLI) can extend without touching the indexing pipeline itself.
- No blockers for Phase 3.

---
*Phase: 02-go-indexing-pipeline*
*Completed: 2026-07-11*

## Self-Check: PASSED

All created files found on disk; both task commits (8d12f84, 0650557) verified present in git log.
