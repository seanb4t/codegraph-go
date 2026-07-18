---
phase: 07-interactive-tui-daemon-picker-install-multi-select
plan: 01
subsystem: cli
tags: [bubbletea, bubbles, tui, tty, cobra]

requires: []
provides:
  - "internal/cli/tui — the new sibling package that is the sole importer of charm.land/bubbletea/v2 and charm.land/bubbles/v2"
  - "InteractiveAllowed(cmd *cobra.Command) bool — the shared dual-TTY gate every later tea.NewProgram() call site checks first"
  - "charm.land/bubbletea/v2 v2.0.8 + charm.land/bubbles/v2 v2.1.1 pinned in go.mod"
affects: [07-06-daemon-picker, 07-07-install-multi-select]

tech-stack:
  added: ["charm.land/bubbletea/v2 v2.0.8", "charm.land/bubbles/v2 v2.1.1"]
  patterns:
    - "internal/cli/tui sibling package confines all bubbletea/bubbles imports outside the TUI-01 archtest's guarded closure (internal/cli is excluded by construction)"
    - "Injectable package-level func vars (stdinIsInteractive/stdoutIsTTY/noColor) for pty-free unit testing, mirroring install.go's installStdinIsInteractive"
    - "Anchor blank-import (doc.go) to pin a dependency in go.mod ahead of its real usage in a later plan"

key-files:
  created:
    - internal/cli/tui/doc.go
    - internal/cli/tui/tty.go
    - internal/cli/tui/tty_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Re-verified charm.land/bubbletea/v2@v2.0.8 and charm.land/bubbles/v2@v2.1.1 as current via go list -m -versions before pinning (RESEARCH's versions still current)"
  - "Added internal/cli/tui/doc.go (not in the plan's stated files_modified) with two blank anchor imports of the new deps — without a real importer, go mod tidy -e prunes unused requires, which would have made Task 1's own acceptance criterion (go.mod contains the require lines) unachievable until 07-06/07-07 land their real Models"

requirements-completed: [TUI-04]

coverage:
  - id: D1
    description: "charm.land/bubbletea/v2 + charm.land/bubbles/v2 pinned in go.mod; TUI-01 archtest confirmed GREEN with the new deps present"
    requirement: TUI-04
    verification:
      - kind: unit
        ref: "go test ./internal/cli/present/archtest/ -run TestNoCharmInServeReachablePackages"
        status: pass
    human_judgment: false
  - id: D2
    description: "InteractiveAllowed(cmd) gates on stdin-TTY AND stdout-TTY AND NO_COLOR unset, all four branches unit-proven via injectable seams"
    requirement: TUI-04
    verification:
      - kind: unit
        ref: "internal/cli/tui/tty_test.go#TestInteractiveAllowed"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-07-18
status: complete
---

# Phase 7 Plan 1: Interactive-Layer Deps & Dual-TTY Gate Summary

**Pinned charm.land/bubbletea/v2 + bubbles/v2 in go.mod and shipped `internal/cli/tui.InteractiveAllowed`, the shared stdin+stdout+NO_COLOR gate every later `tea.NewProgram()` call site must pass first — with the Phase-6 ANSI-isolation archtest confirmed GREEN.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 2 completed
- **Files modified:** 5 (go.mod, go.sum, internal/cli/tui/doc.go, tty.go, tty_test.go)

## Accomplishments

- `go.mod` now requires `charm.land/bubbletea/v2 v2.0.8` and `charm.land/bubbles/v2 v2.1.1` (re-verified current via `go list -m -versions`, not stale RESEARCH values)
- `internal/cli/present/archtest`'s `TestNoCharmInServeReachablePackages` stays GREEN with zero edits — `internal/daemon` and the other five guarded packages remain charm-free by construction
- New `internal/cli/tui` package hosts `InteractiveAllowed(cmd *cobra.Command) bool`, composing the stdin-char-device probe (copied from `install.go`'s `installStdinIsInteractive` shape) with `present.ChoosePresentation`'s existing stdout+NO_COLOR seam (D-10)
- All three probes (`stdinIsInteractive`, `stdoutIsTTY`, `noColor`) are injectable package-level func vars; `tty_test.go` drives all four behavior rows (both-TTY→true, each single-piped→false, NO_COLOR set→false) without a real pty

## Task Commits

1. **Task 1: Add bubbletea/v2 + bubbles/v2 deps and confirm the ANSI-isolation archtest stays green** - `d83761e` (feat)
2. **Task 2: Create internal/cli/tui with the InteractiveAllowed dual-TTY gate**
   - RED: `ee28544` (test) — failing compile, `InteractiveAllowed`/seams undefined
   - GREEN: `845b0a0` (feat) — implementation + doc.go anchor

## Files Created/Modified

- `go.mod` / `go.sum` — `charm.land/bubbletea/v2 v2.0.8`, `charm.land/bubbles/v2 v2.1.1` pinned direct requires; `golang.org/x/sys` deliberately left `// indirect` (promotion owned by 07-03)
- `internal/cli/tui/tty.go` — `InteractiveAllowed`, `stdinIsInteractive`, `stdoutIsTTY`, `noColor`
- `internal/cli/tui/tty_test.go` — `TestInteractiveAllowed`, four-row table test via injected seams
- `internal/cli/tui/doc.go` — package doc + the two anchor blank imports (see Deviations)

## Decisions & Deviations

**None — followed the plan's version pinning and gate design as specified**, except one auto-fixed blocking issue:

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added `internal/cli/tui/doc.go` anchor imports to keep `go mod tidy -e` from pruning the new deps**
- **Found during:** Task 1 (`go mod tidy -e` after `go get`)
- **Issue:** Neither task in this plan actually imports `bubbletea`/`bubbles` in real code — `tty.go`'s gate only needs `cobra`/`term`/`present`. Running `go mod tidy -e` as the task instructs stripped both new require lines as unused, directly contradicting Task 1's own acceptance criterion ("go.mod contains ... require lines at pinned versions").
- **Fix:** Added `internal/cli/tui/doc.go` with two documented blank imports (`_ "charm.land/bubbles/v2"`, `_ "charm.land/bubbletea/v2"`) as a temporary anchor, explicitly commented to be removed once 07-06/07-07 land the real `daemonpicker.go`/`agentpicker.go` Models that import these packages for real. `internal/cli/tui` is excluded from the TUI-01 archtest's guarded closure (the whole `internal/cli` prefix is excluded), so this blank import introduces zero risk to the ANSI-isolation guarantee.
- **Files modified:** `internal/cli/tui/doc.go` (new)
- **Verification:** `go build ./...`, `go test ./internal/cli/present/archtest/`, and `go test ./internal/cli/tui/` all pass with the anchor present; `git diff go.sum` shows only expected charmbracelet-org transitive additions (`bubbles`, `bubbletea`, `ultraviolet` upgrade, `go-runewidth` upgrade, `go-udiff`, `x/exp/golden`) — no CGo surface introduced.
- **Committed in:** `845b0a0` (Task 2 GREEN commit, alongside `tty.go`)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary to make Task 1's literal acceptance criteria achievable given Task 2 (this plan) doesn't yet construct a real `tea.Program`. No scope creep — the anchor is temporary and self-documents its own removal condition.

## Issues Encountered

None beyond the deviation above.

## Next Phase Readiness

- `internal/cli/tui.InteractiveAllowed` is ready for 07-06 (daemon picker) and 07-07 (install/uninstall multi-select) to call before every `tea.NewProgram()`.
- `internal/cli/tui/doc.go`'s anchor imports should be deleted once those plans land real bubbletea/bubbles usage in the package.
- `golang.org/x/sys` remains `// indirect`; its promotion to direct is explicitly deferred to 07-03 (Windows watchdog file).

---
*Phase: 07-interactive-tui-daemon-picker-install-multi-select*
*Completed: 2026-07-18*

## Self-Check: PASSED

All created files verified present; all task commit hashes (`d83761e`, `ee28544`, `845b0a0`) verified in git log.
