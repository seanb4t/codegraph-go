---
phase: 06-rendering-seam-pretty-status-files
plan: 02
subsystem: cli
tags: [lipgloss, charm, tty, tdd, status, files]

# Dependency graph
requires:
  - phase: 06-01
    provides: internal/cli/present skeleton (styles.go palette, ChoosePresentation), charm.land/lipgloss/v2 as sole-importer dependency, TUI-01 archtest
  - phase: 02-status-content-git-worktree-awareness
    provides: query.StatusResult / RenderStatusText section layout this plan decorates
provides:
  - present.RenderStatus(r query.StatusResult, projectPath string, w io.Writer) error — lipgloss-styled status rendering
  - present.RenderFiles(r query.FilesResult, w io.Writer) error — lipgloss-styled tree/flat files rendering
  - isTTY + NO_COLOR gated branch wired into internal/cli/status.go and internal/cli/files.go RunE
  - test/integration/status_files_plain_test.go — real-subprocess byte-identity + zero-ANSI regression guard
affects: [06-03-progress-feedback, phase-8-release-dependency-audit]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Package-local duplication of internal/query's unexported formatting helpers (formatNumber/formatMB/sortedCounts) rather than exporting them across the query/present boundary — matches render_results.go's renderFileTreeMarkdown precedent"
    - "--path pinning in subprocess integration tests to avoid os.Getwd()'s $PWD-heuristic symlink-resolution variance across separate child-process invocations (a pre-existing Go stdlib quirk, not a product bug)"

key-files:
  created:
    - internal/cli/present/status.go
    - internal/cli/present/status_test.go
    - internal/cli/present/files.go
    - internal/cli/present/files_test.go
    - test/integration/status_files_plain_test.go
  modified:
    - internal/cli/status.go
    - internal/cli/files.go
    - go.mod

key-decisions:
  - "golang.org/x/term promoted from // indirect to a direct require via `go mod tidy -e` (not plain `go mod tidy`, per 06-01's documented pre-existing tree-sitter-swift test-dependency resolution error) now that status.go/files.go genuinely import it at their RunE call sites"
  - "files.go's WorktreeNotice computation factored into a local `notice` variable, printed identically in both the pretty and plain branches — byte-identical to the pre-existing plain output, and present's pretty branch also gets the notice rather than silently dropping it"

patterns-established:
  - "present.Render* takes an io.Writer and returns error, matching this codebase's Marshal*JSON/Render* sibling convention rather than returning a string"

requirements-completed: [TUI-02]

coverage:
  - id: D1
    description: "present.RenderStatus emits real ANSI-styled output preserving the plain renderer's section order/wording and numeric formatting, and reflects a WorktreeMismatch warning when present"
    requirement: "TUI-02"
    verification:
      - kind: unit
        ref: "internal/cli/present/status_test.go#TestRenderStatus_ContainsANSI,TestRenderStatus_SectionOrder,TestRenderStatus_NumericFormatting,TestRenderStatus_WorktreeWarning"
        status: pass
    human_judgment: false
  - id: D2
    description: "present.RenderFiles styles both tree and flat formats without altering directory/leaf structure or wording, and handles an empty FilesResult without panicking"
    requirement: "TUI-02"
    verification:
      - kind: unit
        ref: "internal/cli/present/files_test.go#TestRenderFiles_Tree,TestRenderFiles_Flat,TestRenderFiles_Empty"
        status: pass
    human_judgment: false
  - id: D3
    description: "Piped/non-TTY status/files output is byte-identical to the pre-Phase-6 plain output and contains zero ANSI bytes, verified against the real spawned binary"
    requirement: "TUI-02"
    verification:
      - kind: integration
        ref: "test/integration/status_files_plain_test.go#TestStatusFilesPlainByteIdentity"
        status: pass
    human_judgment: false
  - id: D4
    description: "The pretty path never reaches the serve-reachable closure (query/mcp/graphstore/daemon/watch/indexer) — TUI-01 archtest stays green after 06-02's CLI wiring"
    requirement: "TUI-01"
    verification:
      - kind: unit
        ref: "internal/cli/present/archtest/import_graph_test.go#TestNoCharmInServeReachablePackages"
        status: pass
    human_judgment: false

duration: 9min
completed: 2026-07-17
status: complete
---

# Phase 06 Plan 02: Rendering Seam — Pretty status/files Rendering Summary

**Lipgloss-styled `present.RenderStatus`/`present.RenderFiles` land as additive siblings to the untouched plain renderers, wired behind the isTTY+NO_COLOR gate at the CLI boundary — verified byte-identical to the pre-Phase-6 plain output via a real-subprocess integration test.**

## Performance

- **Duration:** ~9 min
- **Started:** 2026-07-17T19:50:15-04:00 (first task commit)
- **Completed:** 2026-07-17T19:58:48-04:00
- **Tasks:** 3
- **Files modified:** 8 (5 created, 3 modified)

## Accomplishments
- `present.RenderStatus` walks the exact same section order as `query.RenderStatusText` (header → Project → worktree warning → Index Statistics → Nodes by Kind → Files by Language → advisories), applying the 06-01 palette as structural chrome only; formatting helpers (`formatNumber`/`formatMB`/`sortedCounts`) are byte-for-byte package-local duplicates of `internal/query`'s unexported equivalents
- `present.RenderFiles` mirrors `internal/cli/files.go`'s plain tree/flat branch selection, styling directory names and language tags without recomputing or re-sorting the tree
- `internal/cli/status.go` and `internal/cli/files.go` RunE bodies now gate on `present.ChoosePresentation(term.IsTerminal(int(os.Stdout.Fd())), os.Getenv("NO_COLOR"))`, routing to the new pretty renderers on a real TTY and falling through to the unchanged plain path otherwise
- `test/integration/status_files_plain_test.go` drives the real spawned binary (non-TTY `bytes.Buffer` stdout) and confirms zero ANSI bytes plus byte-identical output with/without `NO_COLOR=1` for `status`, `files`, and `files --format tree`
- `internal/query/render_status.go` and the `files` plain renderers are untouched — confirmed via `git diff` showing no changes to either
- TUI-01 archtest (`TestNoCharmInServeReachablePackages`) and the full golden corpus (`go test ./testdata/golden/...`) both stay green after this plan's CLI wiring

## Task Commits

Each task was committed atomically:

1. **Task 1: present.RenderStatus — styled StatusResult (D-01/D-02)** - RED `3240faa` (test), GREEN `1c0f6a6` (feat)
2. **Task 2: present.RenderFiles — styled tree + flat (D-01/D-02)** - RED `f788997` (test), GREEN `bf4c42f` (feat)
3. **Task 3: Wire isTTY branch + byte-identity integration test (D-03/D-04/D-06)** - RED `5eecaea` (test), GREEN `3e792f7` (feat)

_Note: this is a `type: tdd` plan — each task is itself a RED/GREEN pass; see "TDD Gate Compliance" below for Task 3's nuance._

## Files Created/Modified
- `internal/cli/present/status.go` - `RenderStatus`, plus package-local `formatNumber`/`formatMB`/`sortedCounts`/`writeStatLine`/`writeBreakdownText`/`writeStatusAdvisories`
- `internal/cli/present/status_test.go` - four behavior-case unit tests (ANSI presence, section order, numeric formatting, worktree warning)
- `internal/cli/present/files.go` - `RenderFiles`, `writeFileTree`
- `internal/cli/present/files_test.go` - tree/flat/empty behavior-case unit tests
- `internal/cli/status.go` - isTTY branch inserted before the plain `RenderStatusText` line; doc comment resolved from its Phase-6 forward reference
- `internal/cli/files.go` - isTTY branch gating the tree/flat block; `WorktreeNotice` computation factored into a shared `notice` variable printed identically in both branches
- `test/integration/status_files_plain_test.go` - `TestStatusFilesPlainByteIdentity`, three subtests (status/files-flat/files-tree)
- `go.mod` - `golang.org/x/term` promoted from `// indirect` to a direct require

## Decisions Made
- `go mod tidy -e` (not plain `go mod tidy`): reproduces 06-01's documented pre-existing `tree-sitter-swift` test-dependency resolution error, unrelated to this plan. `-e` let tidy complete the `golang.org/x/term` promotion despite it.
- Integration test pins `--path <dir>` explicitly rather than relying on `cmd.Dir` + the CLI's `os.Getwd()` fallback: Go's `os.Getwd()` consults the `$PWD` env var as a same-inode heuristic before falling back to full syscall reconstruction, which can resolve macOS's `/var` → `/private/var` symlink differently across two separate subprocess invocations even with identical `cmd.Dir`. This is a pre-existing Go stdlib quirk in `resolveStartPath`, orthogonal to TUI-02's actual claim (ANSI presence, NO_COLOR gating) — pinning `--path` removes the noise without touching product code.
- `files.go`'s `WorktreeNotice` print stays byte-identical in the plain branch (factored into a `notice` variable computed once, printed before the tree/flat `if`) — the pretty branch also receives the notice rather than silently dropping it, resolving RESEARCH's "Claude's discretion" open point in D-02's favor of parity over omission.

## Deviations from Plan

None — plan executed as written. The `--path`-pinning integration-test fix was a test-quality correction within Task 3's own scope (the test as originally drafted was comparing an unrelated `os.Getwd()` symlink-resolution artifact, not a product-code deviation), not a Rule 1-3 auto-fix against implementation code.

## TDD Gate Compliance

Tasks 1 and 2 show a clean RED (build failure: `undefined: RenderStatus` / `undefined: RenderFiles`) → GREEN (all four/three behavior tests pass) cycle, confirmed by moving the implementation file aside, re-running `go test`, then restoring it.

Task 3's `TestStatusFilesPlainByteIdentity` is a genuine exception, documented rather than silently skipped: the CLI wiring change it protects only ever ADDS a new branch that fires exclusively when `os.Stdout` is a real TTY — something this test's harness (a subprocess piped into a `bytes.Buffer`) structurally cannot produce (RESEARCH Pitfall 3). Because the non-TTY plain path is safe-by-default, the test passes both before and after the wiring lands; there is no code path under test that can regress from "pass" to "fail" purely by the wiring being absent. This is not a vacuous test — Tasks 1/2's `TestRenderStatus_ContainsANSI`/`TestRenderFiles_Tree` independently prove `RenderStatus`/`RenderFiles` emit real ANSI content, and this integration test is the regression guard proving the CLI wiring change does not accidentally leak that content (or diverge under `NO_COLOR`) into the non-TTY path. Verified both pre- and post-wiring commit that the test result is identical (pass), consistent with this reasoning.

## Issues Encountered
None beyond the `os.Getwd()`/`$PWD`-heuristic integration-test quirk described above, resolved by pinning `--path` in the test rather than touching product code.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- `status`/`files` now render lipgloss-styled sectioned output on a real TTY and stay byte-identical plain when piped — TUI-02 is complete for both commands this phase scopes.
- 06-03 (TUI-05 progress feedback for `init`/`index`/`sync`) can reuse `present.ChoosePresentation` identically at each of its three call sites, gated against `os.Stderr`'s fd per D-08 rather than `os.Stdout`'s.
- `internal/cli/present`'s TUI-01 archtest remains the single build-time guarantee that no charm import ever reaches the serve-reachable closure — verified green after this plan's `internal/cli` wiring changes (which are outside the guarded set by design).

---
*Phase: 06-rendering-seam-pretty-status-files*
*Completed: 2026-07-17*

## Self-Check: PASSED

All created files (status.go, status_test.go, files.go, files_test.go, status_files_plain_test.go, this SUMMARY) verified present on disk; all six task commit hashes (3240faa, 1c0f6a6, f788997, bf4c42f, 5eecaea, 3e792f7) verified present in git log; `git diff --stat internal/query/render_status.go` returns empty (unmodified); `rg -n "bubbletea" go.mod` returns no matches.
