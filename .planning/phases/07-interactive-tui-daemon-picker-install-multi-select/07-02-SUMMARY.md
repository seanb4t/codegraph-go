---
phase: 07-interactive-tui-daemon-picker-install-multi-select
plan: 02
subsystem: infra
tags: [go, daemon, registry, fsatomic, tdd]

requires:
  - phase: 05-git-sync-hooks
    provides: internal/fsatomic.WriteFile (atomic crash-safe file write primitive)
  - phase: 04-output-hygiene
    provides: internal/daemon/lock.go's isProcessLive/isStale self-heal discipline (Phase 4 D-16)
provides:
  - "internal/daemon/registry.go — Record{PID,StartedAt,RepoRoot}; Register/Deregister via fsatomic; List() with self-heal"
  - "A global ~/.codegraph/daemons/ registry that a picker (07-07) or daemon stop --all (07-04) can list/prune across projects"
affects: [07-04-daemon-start-stop, 07-05-watchdog-registry-wiring, 07-07-daemon-picker-tui]

tech-stack:
  added: []
  patterns:
    - "Registry self-heal-on-read: List() applies lock.go's existing isStale/isProcessLive (same package, unexported, no rival liveness probe) and prunes dead records in place — no background reaper, generalizing acquire()'s single-lockfile discipline to many record files"
    - "Test seam via package-level func var: registryDir is an overridable var (mirrors daemon.go's onSync*/onWatchOpen convention) so tests point it at t.TempDir() instead of the real home directory"

key-files:
  created:
    - internal/daemon/registry.go
    - internal/daemon/registry_test.go
  modified: []

key-decisions:
  - "registryDir is a package-level func var (not a parameter threaded through every call), matching the established onSync*/onWatchOpen test-seam pattern in this same package"
  - "Register relies on fsatomic.WriteFile's own MkdirAll rather than duplicating an explicit os.MkdirAll call — kept the implementation to the minimal correct form"

patterns-established:
  - "Pattern 3 from 07-RESEARCH.md (charm-free registry self-heal) implemented verbatim: Record/registryDir/Register/List generalizing lock.go's single-file self-heal to a directory of per-pid files"

requirements-completed: [DMON-04]

coverage:
  - id: D1
    description: "Register/Deregister atomically write and remove ~/.codegraph/daemons/<pid>.json records via fsatomic.WriteFile; two daemons sharing a repoRoot get distinct pid-keyed files; Deregister of a missing pid is a nil no-op"
    requirement: "DMON-04"
    verification:
      - kind: unit
        ref: "internal/daemon/registry_test.go#TestRegistryRegisterDeregister"
        status: pass
      - kind: unit
        ref: "internal/daemon/registry_test.go#TestRegistrySameRepoRootDistinctFiles"
        status: pass
    human_judgment: false
  - id: D2
    description: "List() self-heals on every independent read by reusing lock.go's isStale/isProcessLive: prunes and excludes dead-pid records, keeps live ones, skips malformed records without erroring; absent/empty registry dir returns (nil, nil)"
    requirement: "DMON-04"
    verification:
      - kind: unit
        ref: "internal/daemon/registry_test.go#TestRegistryListPrunesStale"
        status: pass
      - kind: unit
        ref: "internal/daemon/registry_test.go#TestRegistryListMissingDir"
        status: pass
      - kind: unit
        ref: "internal/daemon/registry_test.go#TestRegistryListEmptyDir"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-07-18
status: complete
---

# Phase 7 Plan 2: Charm-Free Global Daemon Registry Summary

**Global `~/.codegraph/daemons/` registry (Register/Deregister/List) that self-heals stale records by reusing `lock.go`'s isStale/isProcessLive, with no rival liveness implementation.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-07-18T15:19:38-04:00
- **Completed:** 2026-07-18T15:24:53-04:00
- **Tasks:** 2 (both TDD: RED → GREEN)
- **Files modified:** 2 (both new)

## Accomplishments
- `internal/daemon/registry.go`: `Record{PID, StartedAt, RepoRoot}`, injectable `registryDir()` defaulting to `~/.codegraph/daemons/` via `os.UserHomeDir()`, `Register`/`Deregister` writing/removing atomic per-pid JSON records via `fsatomic.WriteFile`
- `List()` self-heals on every call: prunes dead-pid records from disk (calling `lock.go`'s `isStale`/`isProcessLive` directly, same package, zero rival liveness logic), returns only live records, tolerates missing dir / empty dir / malformed record / vanished-between-scan race
- Full test coverage of the DMON-04 edges named in the plan's `must_haves`: round trip, missing-pid deregister, same-repoRoot adjacency (two distinct files, never merged), stale-pruning, empty/missing-dir, malformed-record-skip

## Task Commits

Each task followed RED → GREEN (TDD):

1. **Task 1: Record type + Register/Deregister via fsatomic**
   - `e6b33d3` test(07-02): add failing test for registry Register/Deregister round trip (RED)
   - `08e0947` feat(07-02): add daemon registry Record/Register/Deregister via fsatomic (GREEN)
2. **Task 2: List() with self-heal via lock.go isStale**
   - `7cd8602` test(07-02): add failing test for registry List self-heal (RED)
   - `2a9ff5e` feat(07-02): add registry.List() with self-heal via lock.go isStale (GREEN)

No REFACTOR commits — both implementations landed minimal and clean on GREEN; no cleanup was needed.

## Files Created/Modified
- `internal/daemon/registry.go` - `Record`, `registryDir` (test seam), `recordPath`, `Register`, `Deregister`, `List`
- `internal/daemon/registry_test.go` - `withRegistryDir` test seam helper, round-trip/adjacency/self-heal/empty-edge/malformed-skip tests

## Decisions Made
- `registryDir` implemented as a package-level func var (not threaded as a parameter) to match the existing `onSync*`/`onWatchOpen` test-seam convention already established in `daemon.go`/`daemon_test.go` in this same package.
- Skipped an explicit `os.MkdirAll` in `Register` since `fsatomic.WriteFile` already creates the target directory — avoided duplicating logic the plan's action text described but the underlying primitive already provides.

## Deviations from Plan

None — plan executed exactly as written, including the RED-first TDD ordering and the exact `Record`/`registryDir`/`Register`/`Deregister`/`List` shapes from 07-RESEARCH.md's Pattern 3.

## Issues Encountered

None. `go build ./...`, `go vet ./internal/daemon/...`, `go test ./internal/daemon/... -race` (goleak-clean via the package's existing `TestMain`), and `go test ./internal/cli/present/archtest/...` (`TestNoCharmInServeReachablePackages`) all pass — confirming `internal/daemon` stays charm-free and no rival liveness probe was introduced (grep-confirmed: `registry.go` only calls into `lock.go`'s `isStale`).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `registry.Register`/`Deregister`/`List` are ready for 07-05 to wire into `daemon.Run`'s start/shutdown path (D-06) and for 07-07's picker / 07-04's `daemon stop --all` to consume as the cross-project data source.
- No blockers. The registry is fully independent of the not-yet-built watchdog/picker/CLI-tree work in this phase's other plans.

---
*Phase: 07-interactive-tui-daemon-picker-install-multi-select*
*Completed: 2026-07-18*

## Self-Check: PASSED

- FOUND: internal/daemon/registry.go
- FOUND: internal/daemon/registry_test.go
- FOUND: .planning/phases/07-interactive-tui-daemon-picker-install-multi-select/07-02-SUMMARY.md
- FOUND: e6b33d3 (test RED, Task 1)
- FOUND: 08e0947 (feat GREEN, Task 1)
- FOUND: 7cd8602 (test RED, Task 2)
- FOUND: 2a9ff5e (feat GREEN, Task 2)
