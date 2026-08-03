---
phase: 06-rendering-seam-pretty-status-files
plan: 01
subsystem: cli
tags: [lipgloss, charm, archtest, go-packages, tty, tdd]

# Dependency graph
requires:
  - phase: 04-output-hygiene
    provides: the six-package serve-reachable guarded-set precedent (stdout_confinement_test.go) this archtest mirrors
  - phase: 02-status-content-git-worktree-awareness
    provides: query.StatusResult / render_status.go section layout the pretty renderer will decorate in 06-02
provides:
  - internal/cli/present package skeleton — the sole home for charm.land/lipgloss/v2 styling (D-01)
  - internal/cli/present/archtest — TUI-01 build-time ANSI-isolation guarantee, permanently green
  - present.ChoosePresentation(isTTY, noColor) — the one shared, unit-tested TTY/NO_COLOR branch selector
  - charm.land/lipgloss/v2 as a direct go.mod dependency (no bubbles/bubbletea)
affects: [06-02-status-files-rendering, 06-03-progress-feedback, phase-8-release-dependency-audit]

# Tech tracking
tech-stack:
  added: [charm.land/lipgloss/v2 v2.0.5]
  patterns:
    - "Build-enforced import-graph archtest (go/packages, guarded-set + self-defeat guard) — mirrors internal/graphstore/archtest twice over, now proven for a third boundary (charm isolation)"
    - "Pure branch-selector function (ChoosePresentation) with the real TTY/env read deferred to the RunE call site — makes TTY-gated logic unit-testable without a real pty"

key-files:
  created:
    - internal/cli/present/archtest/import_graph_test.go
    - internal/cli/present/styles.go
    - internal/cli/present/tty.go
    - internal/cli/present/tty_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "go mod tidy was run with -e (ignore-errors) because the whole-module dependency graph resolution hits a pre-existing, unrelated tree-sitter-swift test-dependency error (confirmed present at HEAD before this plan's changes too, via a scratch checkout) — -e let tidy still complete the direct/indirect require bookkeeping for lipgloss/x-term without being blocked by that pre-existing issue"
  - "golang.org/x/term was NOT promoted to a direct require in this plan — nothing in 06-01's file scope imports it directly (tty.go is explicitly forbidden from calling term.IsTerminal itself per D-03/Task 3's own acceptance criteria); the promotion happens naturally in 06-02 when internal/cli/status.go and files.go RunE bodies call term.IsTerminal(os.Stdout.Fd()) directly"

patterns-established:
  - "TUI-01 archtest: guardedPackages (six serve-reachable packages) + forbiddenImportPaths (the three /v2-suffixed charm paths) + a whole-module self-defeat guard in one TestNoCharmInServeReachablePackages, following D-10/D-11/D-12"

requirements-completed: [TUI-01]

coverage:
  - id: D1
    description: "TUI-01 archtest fails closed (self-defeat guard) before any charm dependency exists, then goes green once internal/cli/present is the sole charm.land/lipgloss/v2 importer and none of the six guarded packages import any /v2 charm path"
    requirement: "TUI-01"
    verification:
      - kind: unit
        ref: "internal/cli/present/archtest/import_graph_test.go#TestNoCharmInServeReachablePackages"
        status: pass
    human_judgment: false
  - id: D2
    description: "ChoosePresentation is a pure, unit-tested TTY/NO_COLOR branch selector matching the D-04/D-05 truth table"
    requirement: "TUI-01"
    verification:
      - kind: unit
        ref: "internal/cli/present/tty_test.go#TestChoosePresentation"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-07-17
status: complete
---

# Phase 06 Plan 01: Rendering Seam Foundation — TUI-01 Archtest + Charm Skeleton Summary

**Build-enforced charm.land/lipgloss/v2 isolation via a fail-closed TUI-01 archtest, landed RED-first then flipped GREEN by the sole `internal/cli/present` charm importer, plus the pure `ChoosePresentation` TTY/NO_COLOR selector.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-07-17T23:41:52Z
- **Completed:** 2026-07-17T23:48:45Z
- **Tasks:** 3
- **Files modified:** 6 (4 created, 2 modified)

## Accomplishments
- `internal/cli/present/archtest/import_graph_test.go` mirrors `internal/graphstore/archtest/stdout_confinement_test.go`'s guarded-set/closure-walk mechanism plus `import_graph_test.go`'s self-defeat guard, corrected to the `/v2`-suffixed charm forbidden paths (RESEARCH Finding 1)
- Confirmed the archtest is genuinely fail-closed: it FAILED via the self-defeat guard before any charm dependency existed (RED), then went GREEN once `internal/cli/present` became a real charm importer
- `charm.land/lipgloss/v2` landed as the sole new direct dependency — no `bubbles`/`bubbletea` anywhere in `go.mod`
- `ChoosePresentation(isTTY, noColor) bool` is a pure, four-case unit-tested selector with zero `os`/`golang.org/x/term` imports of its own

## Task Commits

Each task was committed atomically:

1. **Task 1: Land the fail-closed TUI-01 archtest (RED-first)** - `5f1e527` (test)
2. **Task 2: Add charm.land/lipgloss/v2 + present skeleton — archtest goes GREEN** - `f5f399b` (feat)
3. **Task 3: ChoosePresentation pure selector (D-04/D-05)** - `97ff459` (feat)

**Plan metadata:** (this commit)

_Note: this is a `type: tdd` plan — each task is itself a RED/GREEN pass; Task 1's own commit lands after confirming RED, Tasks 2/3 land after confirming GREEN._

## Files Created/Modified
- `internal/cli/present/archtest/import_graph_test.go` - TUI-01 guarded-set + self-defeat-guard archtest
- `internal/cli/present/styles.go` - sole charm.land/lipgloss/v2 importer; shared header/label/section style palette
- `internal/cli/present/tty.go` - `ChoosePresentation(isTTY, noColor) bool`, pure
- `internal/cli/present/tty_test.go` - four-case TTY/NO_COLOR truth-matrix unit test
- `go.mod` / `go.sum` - `charm.land/lipgloss/v2` direct require + its pure-Go transitive closure (colorprofile, ultraviolet, x/ansi, x/term (charmbracelet's), x/termios, x/windows, clipperhouse/displaywidth, clipperhouse/uax29/v2, go-colorful, cancelreader, xo/terminfo); go-runewidth upgraded, uniseg added

## Decisions Made
- `go mod tidy -e`: the plain (no `-e`) `go mod tidy` fails on a pre-existing, unrelated `tree-sitter-swift` test-dependency resolution error. Verified via a scratch checkout that this failure exists at HEAD independent of this plan's changes. `-e` lets tidy finish its direct/indirect bookkeeping despite that unrelated error.
- `golang.org/x/term` stays `// indirect` after this plan. RESEARCH's installation note ("go mod tidy promotes golang.org/x/term ... since present will import it downstream") assumed a direct import that Task 3's own acceptance criteria explicitly forbid inside `tty.go` (D-03: real TTY/env reads happen only at the RunE call sites). Nothing in 06-01's file scope imports `golang.org/x/term` directly, so `go mod tidy` correctly leaves it indirect; it becomes a genuine direct import (and gets promoted) in 06-02 when `internal/cli/status.go`/`files.go` call `term.IsTerminal(os.Stdout.Fd())`.
- `CGO_ENABLED=0 go build ./...` fails for the whole module (tree-sitter grammar packages require CGo — the project's own documented, justified CGo exception). Verified this is pre-existing and unrelated to this plan (same failure at HEAD before lipgloss was added). Scoped the CGo-free verification to what this plan actually touches: `CGO_ENABLED=0 go vet ./internal/cli/present/...` passes clean, confirming no new CGo entered the build via this plan's changes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed literal "Renderer"/"NewRenderer"/"DefaultRenderer" substrings from a styles.go doc comment**
- **Found during:** Task 2 verification
- **Issue:** The acceptance criterion "No occurrence of Renderer/NewRenderer/DefaultRenderer in internal/cli/present" is a literal string check; the original doc comment explained lipgloss v2's removed API by name, which would have failed a naive grep even though no such symbol was actually used.
- **Fix:** Reworded the comment to describe the removal without repeating the removed symbol names verbatim.
- **Files modified:** internal/cli/present/styles.go
- **Verification:** `rg -n "Renderer|NewRenderer|DefaultRenderer" internal/cli/present/` now returns no matches (exit 1); package still builds and vets clean.
- **Committed in:** f5f399b (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug/wording)
**Impact on plan:** Cosmetic only — no behavior change. No scope creep.

## Issues Encountered
- `go mod tidy` (no `-e`) errors on an unrelated, pre-existing `github.com/tree-sitter/tree-sitter-swift/bindings/go` test-dependency resolution problem in `internal/parser/cgo`'s test imports. Confirmed pre-existing (reproduces identically on a scratch checkout of HEAD with none of this plan's changes applied). Worked around with `go mod tidy -e`, which completes the direct/indirect require updates this plan needed. Not fixed as part of this plan (out of scope — a Swift-grammar test-dependency issue, unrelated to the rendering seam).

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The rendering seam is live and permanently build-enforced: `internal/cli/present` is the sole charm importer, `TestNoCharmInServeReachablePackages` will fail the build the moment any of the six guarded packages (or anything they transitively import) reaches for `charm.land/lipgloss/v2`, `charm.land/bubbletea/v2`, or `charm.land/bubbles/v2`.
- `present.ChoosePresentation` is ready for 06-02 (`status`/`files` TUI-02 wiring) and 06-03 (progress feedback TUI-05) to call identically at each `RunE` boundary.
- `internal/cli/present/styles.go`'s header/label/section palette is available for 06-02's `RenderStatus`/`RenderFiles` to consume directly.
- Blocker/concern carried into 06-02: `golang.org/x/term` will need to be genuinely imported at the CLI RunE call sites (status.go/files.go) — expect `go mod tidy -e` (not plain `go mod tidy`, due to the pre-existing swift issue) to be needed again there to promote it to a direct require.

---
*Phase: 06-rendering-seam-pretty-status-files*
*Completed: 2026-07-17*

## Self-Check: PASSED

All created files (archtest, styles.go, tty.go, tty_test.go, this SUMMARY) verified present on disk; all four commit hashes (5f1e527, f5f399b, 97ff459, 6556c4c) verified present in git log.
