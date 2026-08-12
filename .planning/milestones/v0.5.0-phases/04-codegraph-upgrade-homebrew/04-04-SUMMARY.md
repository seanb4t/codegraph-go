---
phase: 04-codegraph-upgrade-homebrew
plan: 04
subsystem: cli
tags: [cobra, help-text, docs, homebrew, upgrade]

# Dependency graph
requires:
  - phase: 04-codegraph-upgrade-homebrew (plan 04-01)
    provides: detectBrewManaged and brewPointerMessage in internal/upgrade/brew.go, and the offline detection branch in upgrade.Run() that this plan documents
provides:
  - "codegraph upgrade --help documents the Homebrew refusal, the brew upgrade codegraph pointer, and both exit behaviours (non-zero for bare refusal, zero for --check)"
  - "README.md and docs/RELEASE.md describe the shipped brew-refusal mechanism instead of announcing it as future work"
affects: [phase-04-ship-readiness, docs]

# Actuals (#2632)
actuals:
  tokens: 1592
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns: ["positive-then-negative substring assertions on cobra Long text, per repo rule 84d1gfpywd"]

key-files:
  created: []
  modified:
    - internal/cli/upgrade.go
    - internal/cli/upgrade_test.go
    - README.md
    - docs/RELEASE.md

key-decisions:
  - "Long text states no override/bypass exists by simply never mentioning one (per plan action: 'naming a non-existent one is worse than silence'), rather than saying 'there is no override' - keeps the absence assertion (no 'override'/'--force'/'bypass' substrings) meaningful rather than self-contradicting"
  - "Both README.md and docs/RELEASE.md keep 'brew upgrade codegraph' on a single unwrapped line so the literal substring check (which does not span line breaks) matches; the earlier draft wrapped it across two lines and failed the plan's own automated verify"

patterns-established: []

requirements-completed: [UPGR-01, UPGR-03]

coverage:
  - id: D1
    description: "codegraph upgrade --help names the Homebrew refusal, the brew upgrade codegraph pointer, and both exit behaviours (D-07, D-10)"
    requirement: "UPGR-01"
    verification:
      - kind: unit
        ref: "internal/cli/upgrade_test.go#TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes"
        status: pass
    human_judgment: false
  - id: D2
    description: "README.md and docs/RELEASE.md describe the shipped Homebrew-refusal mechanism instead of announcing it as future work, and preserve Phase-3's measured verification evidence"
    requirement: "UPGR-03"
    verification:
      - kind: other
        ref: "rg --no-heading -c -e \"next phase's work\" -e 'not yet shipped' -e 'undefined interaction' README.md docs/RELEASE.md (0 matches) paired with rg -c -e 'brew upgrade codegraph' matching both files"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-08-11
status: complete
---

# Phase 4 Plan 04: Document Homebrew Upgrade Refusal Summary

**`codegraph upgrade --help`, README.md, and docs/RELEASE.md now all describe the brew-refusal mechanism plan 04-01 shipped — non-zero exit on bare refusal, zero exit on `--check`, no override — replacing three stale "not yet shipped" claims.**

## Performance

- **Duration:** 25 min
- **Started:** 2026-08-11T17:04:00Z
- **Completed:** 2026-08-11T17:29:22Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Extended `newUpgradeCmd()`'s `Long` with a paragraph naming Homebrew detection, the `brew upgrade codegraph` pointer (D-07), the bare-refusal non-zero exit (D-05), and the `--check` zero exit (D-09/D-10)
- Added `TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes`, asserting 4 required positive substrings and 3 forbidden override-wording substrings (`--force`, `override`, `bypass`); observed RED (all 4 positive assertions failing) when the paragraph was temporarily removed, then reverted byte-clean
- Rewrote README.md's Homebrew block and docs/RELEASE.md's `**Upgrading.**` paragraph to state the shipped detection mechanism (symlink-resolved binary path against Homebrew's own `INSTALL_RECEIPT.json`), both exit behaviours, and the deliberate absence of an override (`brew uninstall --cask codegraph` is the honest path)
- Added a dated `Amended 2026-08-11 (phase 4, UPGR-01/UPGR-02/UPGR-03)` note to docs/RELEASE.md; left the `Verified 2026-08-10 (plan 03-05)` block and `brew trust` instructions untouched

## Task Commits

Each task was committed atomically:

1. **Task 1: `--help` states the refusal and both exit behaviours, pinned by a test** - `316da9d` (feat)
2. **Task 2: Correct the published Homebrew-upgrade story in README.md and docs/RELEASE.md** - `c5bcd14` (docs)

_Note: this SUMMARY and STATE.md/ROADMAP.md updates are committed separately by the orchestrator after all wave agents complete._

## Files Created/Modified
- `internal/cli/upgrade.go` - `Long`/`Example` extended with the brew-refusal paragraph and a `--check` brew example line
- `internal/cli/upgrade_test.go` - added `TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes`
- `README.md` - Homebrew block's closing sentence rewritten to state shipped behaviour
- `docs/RELEASE.md` - `**Upgrading.**` paragraph rewritten with a dated amendment note

## Decisions Made
- Kept the `Long` text free of the words "override"/"--force"/"bypass" entirely, per the plan's instruction not to describe a non-existent override — this made the test's absence assertion meaningful rather than self-defeating (an earlier draft said "there is no flag to bypass the refusal," which contained "bypass" and would have failed its own forbidden-substring check)
- Reflowed the `brew upgrade codegraph` phrase onto a single unwrapped line in both README.md and docs/RELEASE.md after the plan's own `rg` verify command (which does not match across a line break) initially failed on README.md's wrapped Markdown paragraph

## Deviations from Plan

None - plan executed exactly as written. The line-wrapping fix above was a mechanical correction to satisfy the plan's own literal `rg` verification command, not a deviation from the plan's intent — the semantic content (naming `brew upgrade codegraph`) was correct from the first edit; only its Markdown line-wrap position needed to change for the substring match.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Verification Evidence

- `go test ./internal/cli/ -run '^TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes$' -v` → exactly one `--- PASS: TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes` line (confirmed via `rg -c`)
- RED proof: temporarily removing the brew paragraph produced 4 failing assertions (`Long missing required substring...` for `brew upgrade codegraph`, `Homebrew-managed install`, `exits\nnon-zero`, `exits\nzero`); reverted byte-clean via a scratchpad backup, confirmed with `git diff internal/cli/upgrade.go` showing the intended final diff only
- `go test ./internal/cli/...` → all 5 sub-packages `ok`
- `go build ./...` and `go vet ./internal/cli/` → both exit 0
- `task test` (full suite, including `test:race` over `internal/cli`) → exit 0, all packages `ok`
- `git diff --exit-code go.mod go.sum` → exit 0 (no dependency changes)
- All plan `<acceptance_criteria>` `rg` checks for Task 2 re-run individually and confirmed passing: stale-phrase absence (0 matches), `brew upgrade codegraph` present in README.md, docs/RELEASE.md, internal/cli/upgrade.go, and internal/upgrade/brew.go (all ≥1); both exit behaviours named in both docs; `brew uninstall --cask codegraph` present in docs/RELEASE.md; `Verified 2026-08-10 (plan 03-05)` exactly once; `brew trust` ≥2 times; `bash-completion` present in README.md; amendment note naming UPGR-01/UPGR-02/UPGR-03 present

## Next Phase Readiness
- Every published surface (`--help`, README.md, docs/RELEASE.md) now agrees with the shipped `internal/upgrade` mechanism from plan 04-01; no outstanding "not yet shipped" claims remain in this documentation surface
- No blockers for the remaining phase-4 work

---
*Phase: 04-codegraph-upgrade-homebrew*
*Completed: 2026-08-11*
