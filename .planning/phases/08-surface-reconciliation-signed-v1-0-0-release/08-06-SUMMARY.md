---
phase: 08-surface-reconciliation-signed-v1-0-0-release
plan: 06
subsystem: docs
tags: [cobra, cli, flag-parity, drift-test, docs]

# Dependency graph
requires:
  - phase: 08-surface-reconciliation-signed-v1-0-0-release
    provides: "08-01..08-05's finalized flag surface (impact -d/-j, files --dir/-j, status/query/callers/callees/install/uninstall shorts, upgrade --force, affected --stdin/--depth/--filter/--quiet/-j)"
provides:
  - "docs/FLAG-PARITY.md — the complete per-command TS 1.3.1 <-> Go flag/default/status matrix, the single artifact REL-04's drop-in gate reads"
  - "internal/cli/flag_parity_test.go — a self-verifying drift guard walking newRootCmd() and failing if any registered flag is undocumented"
affects: [08-07, 08-08, 08-09, REL-04-drop-in-gate]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Doc-as-oracle drift test: read a markdown doc once, walk the real cobra command tree, assert every registered flag name is a substring of the doc text — fails closed if the doc is missing, reports every gap in one failure message"

key-files:
  created:
    - docs/FLAG-PARITY.md
    - internal/cli/flag_parity_test.go
  modified:
    - go.mod

key-decisions:
  - "spf13/pflag promoted go.mod indirect->direct since the drift test imports pflag.Flag directly to walk cmd.Flags().VisitAll — same precedent as Phase 6's x/term promotion"
  - "Genuine TDD RED was not naturally achievable for this task: docs/FLAG-PARITY.md (Task 1) already fully covers the current flag surface, so the test passes immediately on first write. The self-defeat guard (temporarily stripping 'auto-allow' from the doc, confirming a real failure, then reverting) was performed manually during execution as the RED-equivalent proof instead of forcing an artificial doc gap into a separate commit — matching the plan's own acceptance-criteria wording ('verified once during execution, then reverted')"
  - "query/callers/callees --limit's default-value gap (Go 0=uncapped vs TS 10/20) is recorded as an accepted divergence rather than fixed — SURF-05 is audit-only per the plan's explicit prohibition against behavior changes"

patterns-established:
  - "SURF-05-style parity docs pair a static markdown matrix with a lightweight tree-walk test so the doc can never silently drift from the real flag surface it claims to describe"

requirements-completed: [SURF-05]

coverage:
  - id: D1
    description: "docs/FLAG-PARITY.md is a per-command matrix covering every command registered in newRootCmd(), including flag-less commands (unlock, telemetry, every githooks subcommand)"
    requirement: "SURF-05"
    verification:
      - kind: unit
        ref: "internal/cli/flag_parity_test.go#TestFlagParityDocCoversRegisteredFlags"
        status: pass
      - kind: manual_procedural
        ref: "grep -q -- \"--dir\"/\"--stdin\"/\"auto-allow\" docs/FLAG-PARITY.md (plan's automated verify command)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every divergence named in CONTEXT/RESEARCH is recorded: dual --filter/--dir, short-flag divergences, install --auto-allow default-off, files --format default, files --depth vs --max-depth naming, node missing file-mode, upgrade --force (now present), search/migrate/githooks as Go-only/accepted"
    requirement: "SURF-05"
    verification:
      - kind: unit
        ref: "manual review of docs/FLAG-PARITY.md's 'Summary of every recorded divergence' section against 08-RESEARCH.md's SURF-03 table and Common Pitfalls 1-6"
        status: pass
    human_judgment: false
  - id: D3
    description: "The drift test walks newRootCmd().Commands() recursively and Flags().VisitAll, asserting every registered long flag appears in the doc; fails closed if the doc is absent; a flag-less command contributes no assertion and does not fail"
    requirement: "SURF-05"
    verification:
      - kind: unit
        ref: "go test ./internal/cli/... -run FlagParity -count=1"
        status: pass
      - kind: manual_procedural
        ref: "self-defeat guard: stripping 'auto-allow' from docs/FLAG-PARITY.md reproduced a real test failure (codegraph install --auto-allow reported missing), reverted afterward with zero diff"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-07-19
status: complete
---

# Phase 8 Plan 6: FLAG-PARITY.md matrix + drift test Summary

**Produced `docs/FLAG-PARITY.md`, the complete per-command TS 1.3.1 <-> Go flag/default/status matrix covering every `newRootCmd()` command, backed by a self-verifying `internal/cli/flag_parity_test.go` drift guard that fails the build if any registered flag goes undocumented.**

## Performance

- **Duration:** ~25 min
- **Completed:** 2026-07-19
- **Tasks:** 2 completed
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments
- `docs/FLAG-PARITY.md` documents all 24 top-level commands plus `daemon start/stop` and `githooks install/remove/status` subcommands — every command registered in `internal/cli/root.go`'s `newRootCmd()`, including flag-less commands (`unlock`, `telemetry`, every `githooks` subcommand).
- Every divergence named in CONTEXT D-06/RESEARCH's Common Pitfalls is recorded verbatim: the dual `--filter`(language, kept)/`--dir`(directory, new) split; every short-flag divergence (`init`/`index -f` semantic mismatch); `install --auto-allow` as a deliberate security-conservative behavioral divergence; `files --format` default mismatch (flat vs tree); `files --depth` vs TS's `--max-depth` naming; `node`'s missing TS file-mode (`--offset`/`--limit`/`--symbols-only`); `upgrade --force`'s now-complete parity (added in 08-03); and `search`/`migrate`/`githooks` as Go-only extensions/accepted divergences.
- Additionally surfaced and documented a divergence not spelled out verbatim in CONTEXT but confirmed by reading `internal/query/validate.go`/`search.go`/`traverse.go`: `query`/`callers`/`callees --limit`'s default is `0` (meaning "uncapped", per `validateLimit`'s doc comment) in Go vs TS's explicit 10/20 defaults — recorded as an accepted, pre-existing divergence, not fixed (SURF-05 is audit-only).
- `internal/cli/flag_parity_test.go` reads the doc once via `os.ReadFile("../../docs/FLAG-PARITY.md")` (fails loudly if absent), walks `newRootCmd()` and every subcommand recursively, and for each `pflag.Flag` (skipping the auto-generated `help` flag) asserts its long name is a literal substring of the doc text — reporting every missing flag in one failure message, not just the first.
- Verified the assertion is real (not vacuous) via a manual self-defeat check: temporarily replaced every occurrence of `auto-allow` in the doc with a decoy string, re-ran the test, confirmed `TestFlagParityDocCoversRegisteredFlags` failed with `codegraph install --auto-allow` reported missing, then reverted the doc to a byte-identical clean state (`git diff docs/FLAG-PARITY.md` empty afterward).

## Task Commits

Each task was committed atomically:

1. **Task 1: docs/FLAG-PARITY.md per-command matrix (SURF-05, D-06)** - `4056c7e` (docs)
2. **Task 2: flag-parity drift test (tree-walk over newRootCmd)** - `6ccd16c` (test)

**Plan metadata:** (this commit, docs: complete plan)

## Files Created/Modified
- `docs/FLAG-PARITY.md` - New per-command TS<->Go flag/default/status matrix, one `##` section per command, plus a "Summary of every recorded divergence" rollup and a "How this doc is enforced" note pointing at the drift test.
- `internal/cli/flag_parity_test.go` - New `TestFlagParityDocCoversRegisteredFlags`: fail-closed doc read, recursive `newRootCmd()` walk, `pflag.Flag` substring assertion, single aggregated failure message.
- `go.mod` - `github.com/spf13/pflag` promoted from `// indirect` to a direct requirement (the test file imports `pflag.Flag` directly for `VisitAll`'s callback signature).

## Decisions Made
- Followed the plan's literal per-command action instructions precisely: reflected the flags AS FINALIZED by 08-01..08-05 (verified against the actual current `internal/cli/*.go` source via `rg -n 'Flags\(\)\.'` across all 26 command files, not just the RESEARCH table, since 08-01..08-05 landed after RESEARCH was written).
- TDD RED was not naturally achievable for Task 2 as a separate failing-then-passing commit: Task 1's doc already fully covers the flag surface it describes, so the drift test passes on first write. Rather than manufacture an artificial gap into a committed RED state (which would require either reverting Task 1's doc or writing an intentionally-wrong test), the self-defeat guard was performed as a manual, uncommitted verification step during execution — exactly matching the plan's own acceptance-criteria wording ("verified once during execution, then reverted"). Both commits (`4056c7e` docs, `6ccd16c` test) exist and are green; no gate is silently missing.
- `spf13/pflag`'s indirect->direct promotion in `go.mod` was accepted as the correct, minimal `go mod tidy -e` byproduct of the test file's direct import — same precedent as Phase 6's `x/term` promotion (STATE.md decision log). `go.sum` was unaffected (already contained the correct checksum lines).

## Deviations from Plan

None — plan executed as written. The `query`/`callers`/`callees --limit` default-value divergence is an additional finding surfaced by reading the engine source during doc authoring (not itself a code change, no Rule 1/2/3 fix applied — SURF-05's own prohibition forbids behavior changes in this plan), documented as an accepted divergence per the doc's own "present or a documented divergence" contract.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- SURF-01..05 are now all complete and green (CONTEXT D-01's sequencing gate). `docs/FLAG-PARITY.md` is the artifact REL-04's drop-in gate reads; `internal/cli/flag_parity_test.go` keeps it honest going forward.
- The REL block (08-07 onward: Charm-closure CGo/vuln/SBOM audit, signed v1.0.0 cut, benchmark re-run, drop-in validation + caveat retirement) is unblocked per D-01.
- No blockers. `go build ./...`, `go vet ./internal/cli/...`, and `go test ./internal/cli/... -count=1` all green.

---
*Phase: 08-surface-reconciliation-signed-v1-0-0-release*
*Completed: 2026-07-19*

## Self-Check: PASSED

All claimed files exist (docs/FLAG-PARITY.md, internal/cli/flag_parity_test.go, this SUMMARY.md); all claimed commits (4056c7e, 6ccd16c) verified present in git log.
