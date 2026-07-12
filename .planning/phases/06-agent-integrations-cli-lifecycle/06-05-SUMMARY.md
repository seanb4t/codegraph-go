---
phase: 06-agent-integrations-cli-lifecycle
plan: 05
subsystem: cli
tags: [cobra, version, ldflags, telemetry, cli-ergonomics]

# Dependency graph
requires: []
provides:
  - internal/version package with ldflags-injected Version/Commit/Date (dev/unknown defaults) and Info() combining them with runtime Go version/os/arch
  - codegraph version subcommand (+ --json)
  - codegraph --version root flag
  - codegraph telemetry command with the honest zero-passive-telemetry statement
affects: [06-06 (upgrade --check consumes internal/version for comparison)]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "ldflags injection seam: internal/version package vars (Version/Commit/Date) set via -X at build time, dev/unknown defaults for go run/go test — main.go itself needs no changes"
    - "static-statement command: RunE is a single Fprintln of a package-level const (telemetry.go), mirroring uninit.go's no-op-message shape"

key-files:
  created:
    - internal/version/version.go
    - internal/cli/version.go
    - internal/cli/version_test.go
    - internal/cli/telemetry.go
    - internal/cli/telemetry_test.go
  modified:
    - internal/cli/root.go

key-decisions:
  - "Named the JSON-tagged struct VersionInfo (not Info) with an Info() accessor function — Go forbids a func and type sharing one identifier at package scope, so the plan's literal 'Info() struct' wording was resolved this way while keeping the version.Info() call-site shape the plan's key_links describe."
  - "Telemetry statement avoids literal 'net/http' or 'net.' substrings in its own text (originally referenced them as verification hints) after the acceptance-criteria grep 'net/http\\|net\\.' matched the const string itself — reworded to 'audit it for outbound connections outside the upgrade command's package' to satisfy the negative-grep check without weakening the honesty claim."

patterns-established:
  - "Version metadata: internal/version.Info() is the single source of truth for build identity; downstream commands (upgrade --check, 06-06) call it rather than reading ldflags vars directly."

requirements-completed: [CLI-01, CLI-03]

coverage:
  - id: D1
    description: "codegraph version / version --json prints ldflags-injected build identity with dev/unknown defaults plus runtime Go version and os/arch"
    requirement: "CLI-01"
    verification:
      - kind: unit
        ref: "internal/cli/version_test.go#TestVersionInfoDefaults"
        status: pass
      - kind: unit
        ref: "internal/cli/version_test.go#TestVersionCommandPlain"
        status: pass
      - kind: unit
        ref: "internal/cli/version_test.go#TestVersionCommandJSON"
        status: pass
    human_judgment: false
  - id: D2
    description: "codegraph --version (root flag) prints a version line"
    requirement: "CLI-01"
    verification:
      - kind: unit
        ref: "internal/cli/version_test.go#TestRootVersionFlag"
        status: pass
    human_judgment: false
  - id: D3
    description: "codegraph help [command] gives command-specific usage for every registered command (Cobra built-in, Short/Long/Example present)"
    requirement: "CLI-01"
    verification:
      - kind: unit
        ref: "internal/cli/telemetry_test.go#TestHelpEveryCommand"
        status: pass
      - kind: unit
        ref: "internal/cli/telemetry_test.go#TestHelpVersionCommand"
        status: pass
    human_judgment: false
  - id: D4
    description: "codegraph telemetry prints the honest zero-passive-telemetry statement naming codegraph upgrade as the sole intentional network path, with no network imports in telemetry.go"
    requirement: "CLI-03"
    verification:
      - kind: unit
        ref: "internal/cli/telemetry_test.go#TestTelemetryStatement"
        status: pass
      - kind: unit
        ref: "internal/cli/telemetry_test.go#TestTelemetryStatementIsConst"
        status: pass
      - kind: other
        ref: "rg -n 'net/http|net\\.' internal/cli/telemetry.go (no matches)"
        status: pass
    human_judgment: false

duration: 12min
completed: 2026-07-12
status: complete
---

# Phase 6 Plan 5: CLI Lifecycle — version, --version, telemetry Summary

**Greenfield `internal/version` package (ldflags-injected build identity) wired into `codegraph version`/`--version`, plus the honest `codegraph telemetry` zero-passive-telemetry statement that names `upgrade` as the sole network path.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-12T18:38:00Z
- **Completed:** 2026-07-12T18:41:42Z
- **Tasks:** 2
- **Files modified:** 6 (5 created, 1 modified)

## Accomplishments
- `internal/version` package: `Version`/`Commit`/`Date` vars (dev/unknown defaults) as the `-ldflags -X` injection target, `VersionInfo` struct, `Info()` accessor combining ldflags vars with `runtime.Version()`/`GOOS`/`GOARCH`
- `codegraph version` (+ `--json`) and `codegraph --version` root flag, both formatted from `version.Info()`
- `codegraph telemetry`: static, auditable statement asserting zero passive/background telemetry while honestly disclosing `codegraph upgrade` as the one intentional, user-initiated network path — no network imports in the file
- Confirmed every existing command already carries a non-empty `Short` (D-10 help ergonomics was already satisfied; no changes needed beyond the two new commands)

## Task Commits

Each task was committed atomically (RED → GREEN):

1. **Task 1: internal/version package + version command + root --version**
   - `0328f83` test(06-05): add failing tests for version package + version command
   - `c745e99` feat(06-05): add internal/version package + version command + root --version
2. **Task 2: telemetry command + help ergonomics check**
   - `fce793d` test(06-05): add failing tests for telemetry command + help ergonomics
   - `158ec0b` feat(06-05): add telemetry command, register in root (D-15)

**Plan metadata:** (this commit)

## Files Created/Modified
- `internal/version/version.go` - ldflags-injected `Version`/`Commit`/`Date` vars, `VersionInfo` struct, `Info()` accessor
- `internal/cli/version.go` - `newVersionCmd()` (+ `--json`), `versionLine()` helper for root.Version
- `internal/cli/version_test.go` - defaults, plain output, JSON output, `--version` root flag tests
- `internal/cli/telemetry.go` - `telemetryStatement` const + `newTelemetryCmd()`
- `internal/cli/telemetry_test.go` - statement content, verbatim-const pinning, help-ergonomics smoke tests
- `internal/cli/root.go` - registered `newVersionCmd()`/`newTelemetryCmd()`, set `root.Version = versionLine()`

## Decisions Made
- **`VersionInfo` type name (not `Info`):** the plan's literal wording ("`Info()` returning an exported `Info` struct") is a Go identifier collision — a func and type cannot share a package-scope name. Resolved by naming the struct `VersionInfo` and keeping the accessor function named `Info()`, preserving the `version.Info()` call-site shape the plan's `key_links` pattern (`version\.Info`) actually requires.
- **Reworded telemetry statement to avoid the literal strings `net/http`/`net.`:** the plan's own acceptance criterion (`grep -n 'net/http\|net\.' internal/cli/telemetry.go` returns no matches) would have failed against the statement's own text if it named those import paths as verification hints. Reworded to "audit it for outbound connections outside the upgrade command's package" — same honesty and verifiability, no self-defeating grep match.

## Deviations from Plan

None beyond the two decisions above (both are Rule 1-class corrections needed to make the plan's own literal acceptance criteria satisfiable — not scope changes).

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required. (Actual `-ldflags -X` wiring into the release build process is Phase 8/DIST scope, per D-14; this plan only builds the package the ldflags target.)

## Next Phase Readiness
- `internal/version.Info()` is ready for 06-06's `upgrade --check` version comparison, as noted in the plan's wave rationale.
- `codegraph version`/`--version`/`telemetry`/`help [command]` all verified via `go run ./cmd/codegraph` manual smoke checks in addition to the automated test suite.

---
*Phase: 06-agent-integrations-cli-lifecycle*
*Completed: 2026-07-12*
