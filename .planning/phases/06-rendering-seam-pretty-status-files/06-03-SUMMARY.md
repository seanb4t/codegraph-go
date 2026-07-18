---
phase: 06-rendering-seam-pretty-status-files
plan: 03
subsystem: cli
tags: [lipgloss, charm, tty, tdd, progress, spinner, stderr]

# Dependency graph
requires:
  - phase: 06-01
    provides: internal/cli/present skeleton (styles.go palette, ChoosePresentation), charm.land/lipgloss/v2 as sole-importer dependency, TUI-01 archtest
  - phase: 06-02
    provides: the isTTY+NO_COLOR CLI-boundary wiring pattern (status.go/files.go) this plan reuses for stderr instead of stdout
provides:
  - present.NewProgress(w io.Writer) *Progress — hand-rolled, stderr-only, lipgloss-styled spinner with deterministic Stop() teardown
  - TTY-gated spinner wiring in internal/cli/init.go, index.go, sync.go (gated on os.Stderr's fd + --quiet)
  - internal/cli/present/progress_test.go — stderr-only + no-goroutine-leak + ANSI-content unit coverage
  - internal/cli/progress_cli_test.go — non-TTY CLI reachability + --quiet regression guard
affects: [phase-8-release-dependency-audit]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hand-rolled ticker spinner (stdlib time.Ticker + lipgloss.Style.Render(), \\r redraw, \\r\\x1b[K clear-on-stop) — no charm.land/bubbles/bubbletea anywhere (D-09/D-13)"
    - "Progress.Stop() blocks on a doneCh close signal before returning — makes goroutine teardown synchronously provable in a unit test without goleak"

key-files:
  created:
    - internal/cli/present/progress.go
    - internal/cli/present/progress_test.go
    - internal/cli/progress_cli_test.go
  modified:
    - internal/cli/init.go
    - internal/cli/index.go
    - internal/cli/sync.go

key-decisions:
  - "progressTickInterval set to 80ms (not the RESEARCH sketch's ~100ms) — Claude's discretion per the plan; keeps the animation snappy and shaves a small amount off each progress_test.go sleep-based assertion without materially changing the visual effect"
  - "Progress.Start()/Stop() are both idempotent no-ops on a non-running instance rather than panicking — matches the plan's explicit 'Stop() is idempotent-safe' behavior-case requirement (Task 1, Test 4)"
  - "TestProgressCLI* tests document (rather than paper over) that this in-process, non-TTY execCmd harness structurally cannot exercise the real spinner-render path — same RESEARCH Pitfall 3 constraint 06-02 already hit for status/files' pretty branch"

patterns-established:
  - "Every progress-writer call site (init/index/sync) evaluates ChoosePresentation against os.Stderr's fd, not os.Stdout's — the one TUI-05-specific divergence from 06-02's status/files stdout-fd convention"

requirements-completed: [TUI-05]

coverage:
  - id: D1
    description: "Progress renders lipgloss-styled frames containing the label on a stdlib time.Ticker to exactly the injected io.Writer, never os.Stdout"
    requirement: "TUI-05"
    verification:
      - kind: unit
        ref: "internal/cli/present/progress_test.go#TestProgress_FramesContainLabelAndANSI,TestProgress_StderrOnly"
        status: pass
    human_judgment: false
  - id: D2
    description: "Stop() terminates the ticker goroutine deterministically (blocks until the goroutine confirms exit) and clears the line; idempotent on repeat/never-started calls"
    requirement: "TUI-05"
    verification:
      - kind: unit
        ref: "internal/cli/present/progress_test.go#TestProgress_NoGoroutineLeak,TestProgress_StopClearsLine,TestProgress_StopIdempotent"
        status: pass
    human_judgment: false
  - id: D3
    description: "init/index/sync's non-TTY path stays byte-identical to pre-Phase-6 behavior: zero ANSI on stderr, unchanged stdout summary, --quiet still suppresses output"
    requirement: "TUI-05"
    verification:
      - kind: integration
        ref: "internal/cli/progress_cli_test.go#TestProgressCLINonTTYReachability,TestProgressCLIQuietSuppressesSummary,TestProgressWiringDoesNotBreakErrorPaths"
        status: pass
    human_judgment: false
  - id: D4
    description: "No charm.land/bubbles or charm.land/bubbletea import anywhere in the progress path or go.mod; TUI-01 archtest and stdout/MCP purity stay green"
    requirement: "TUI-05"
    verification:
      - kind: unit
        ref: "internal/cli/present/archtest/import_graph_test.go#TestNoCharmInServeReachablePackages"
        status: pass
      - kind: integration
        ref: "test/integration -run 'Sync|Purity'"
        status: pass
    human_judgment: false

duration: 4min
completed: 2026-07-18
status: complete
---

# Phase 06 Plan 03: Rendering Seam — TTY-Gated Progress Feedback (TUI-05) Summary

**A hand-rolled, stderr-only lipgloss spinner (stdlib `time.Ticker`, deterministic `Stop()` teardown) lands in `internal/cli/present` and is wired behind the same `ChoosePresentation` gate as status/files — evaluated against `os.Stderr`'s fd this time — into `init`/`index`/`sync`, with zero `charm.land/bubbles`/`bubbletea` surface anywhere.**

## Performance

- **Duration:** ~4 min
- **Started:** 2026-07-17T20:04:16-04:00 (first task commit)
- **Completed:** 2026-07-17T20:07:44-04:00
- **Tasks:** 2
- **Files modified:** 6 (3 created, 3 modified)

## Accomplishments
- `present.Progress` (`NewProgress(w io.Writer)`, `Start(label string)`, `Stop()`) renders one lipgloss-styled Braille-spinner frame per 80ms tick to exactly the writer it was constructed with — `progress_test.go`'s `TestProgress_StderrOnly` proves this by redirecting the *real* `os.Stdout` to a pipe and confirming zero bytes land there while frames land on a separate target buffer
- `Stop()` closes a stop channel and blocks on a `doneCh` close signal before returning, so by the time `Stop()` returns the ticker goroutine is provably gone (`TestProgress_NoGoroutineLeak`) and the line-clear sequence (`\r\x1b[K`) is already written (`TestProgress_StopClearsLine`); both `Start`/`Stop` are safe no-ops when called out of order or repeatedly (`TestProgress_StopIdempotent`)
- `internal/cli/init.go`, `index.go`, `sync.go` each wrap their `indexer.Run`/`indexer.Sync` call with `present.NewProgress(os.Stderr).Start(...)` behind `!quiet && present.ChoosePresentation(term.IsTerminal(int(os.Stderr.Fd())), os.Getenv("NO_COLOR"))`, with `defer prog.Stop()` immediately after `Start` so teardown runs even if the wrapped call errors
- `internal/cli/progress_cli_test.go` drives the real `init`/`index`/`sync`/`--quiet` commands through the in-process cobra tree and confirms the non-TTY path (the only path this harness can produce, per RESEARCH Pitfall 3) is byte-unchanged: zero ANSI on stderr, unchanged `files=…` / `reparsed=…` stdout summaries, `--quiet` still suppresses summary output entirely
- `go test ./...`, `go test ./testdata/golden/...`, and `go test ./test/integration/... -run 'Sync|Purity'` all stay green; the TUI-01 archtest (`TestNoCharmInServeReachablePackages`) stays green; `rg 'bubbletea|bubbles' go.mod` returns nothing

## Task Commits

Each task was committed atomically:

1. **Task 1: Hand-rolled stderr progress writer (D-08/D-09)** - RED `412c9c4` (test), GREEN `5016d5c` (feat)
2. **Task 2: Wire TTY-gated progress into init/index/sync (D-07)** - `b2aff19` (feat)

_Note: this is a `type: tdd` plan — see "TDD Gate Compliance" below for Task 2's documented exception, matching 06-02's precedent._

## Files Created/Modified
- `internal/cli/present/progress.go` - `Progress` type: `NewProgress`, `Start`, `Stop`, the Braille spinner-frame table, `progressTickInterval` (80ms), `clearLineSeq`
- `internal/cli/present/progress_test.go` - five behavior-case unit tests: frame/label/ANSI content, stderr-exclusivity (real `os.Stdout` pipe redirect), clean-stop line clear, idempotent Stop, no-goroutine-leak
- `internal/cli/init.go` - spinner wiring around `indexer.Run`, new imports (`golang.org/x/term`, `internal/cli/present`)
- `internal/cli/index.go` - spinner wiring around `indexer.Run`, same new imports
- `internal/cli/sync.go` - spinner wiring around `indexer.Sync`, same new imports
- `internal/cli/progress_cli_test.go` - three CLI-reachability tests: non-TTY zero-ANSI/unchanged-summary (init/index/sync), `--quiet` suppression, pre-existing error-path regression guard

## Decisions Made
- `progressTickInterval` set to 80ms rather than the RESEARCH sketch's illustrative ~100ms — Claude's explicit discretion per the plan's action text ("cadence Claude's discretion, ~100ms"); keeps the animation visually snappy and slightly speeds up `progress_test.go`'s sleep-based assertions.
- Both `Start` and `Stop` are idempotent no-ops rather than panicking on out-of-order/repeat calls (`Start` on an already-running `Progress`, `Stop` on a never-started or already-stopped one) — directly satisfies Task 1's Test 4 behavior case ("Stop() is idempotent-safe") and makes the CLI wiring's `defer prog.Stop()` safe by construction even if a future edit accidentally called `Stop` twice.
- `TestProgressCLI*`'s doc comments explicitly document (rather than silently accept) that the in-process, non-TTY `execCmd` harness cannot exercise the real spinner-render path — `os.Stderr` under `go test` is never a real terminal, so `ChoosePresentation`'s `isTTY` branch never fires in this test tier. This is the same RESEARCH Pitfall 3 constraint 06-02 already documented for status/files' pretty-vs-plain branch; `Progress`'s own render/leak/stop behavior is separately and directly proven at the unit level in `progress_test.go`, which needs no real terminal since `Progress` takes an arbitrary `io.Writer`.

## Deviations from Plan

None — plan executed as written.

## TDD Gate Compliance

Task 1 shows a clean RED (build failure: `undefined: NewProgress`, `undefined: progressTickInterval`, `undefined: clearLineSeq` — confirmed by moving `progress.go` aside and re-running `go test`, then restoring it) → GREEN (all five `progress_test.go` cases pass) cycle.

Task 2 is a documented exception, following 06-02's own precedent for the identical structural reason: `progress_cli_test.go`'s reachability assertions ("no ANSI on stderr", "unchanged stdout summary") were already true of the codebase *before* this task's wiring landed, because the wiring only ever ADDS a branch that fires exclusively on a real TTY — something the in-process `execCmd` harness (a `bytes.Buffer`-backed, non-TTY `os.Stderr`) structurally cannot produce. There is no code path under test that can regress from "pass" to "fail" purely by the wiring being present vs. absent at this test tier; verified by running the full `progress_cli_test.go` suite both before and after staging the `init.go`/`index.go`/`sync.go` changes — identical (pass) result both times. This is not a vacuous test: `progress_test.go`'s `TestProgress_FramesContainLabelAndANSI`/`TestProgress_StderrOnly` independently prove the underlying `Progress` type renders real content and stays writer-exclusive, and `progress_cli_test.go` is the regression guard proving the CLI wiring change doesn't accidentally leak spinner output (or diverge under `--quiet`) into the non-TTY path that every `go test` run and every piped/scripted invocation actually takes.

## Issues Encountered
`CGO_ENABLED=0 go vet ./internal/cli/present/...` fails on an unrelated, pre-existing `tree_sitter.Node`/CGo build error in `internal/indexer/goextract` — reproduced identically via `git stash` on the pre-Task-2 tree, confirming it is triggered by the TUI-01 archtest's `packages.Load` call over its `guardedPackages` list (which includes `internal/indexer`), not by anything this plan introduced. Same class of pre-existing, out-of-scope issue 06-01/06-02 already documented for `go mod tidy`/whole-module `CGO_ENABLED=0 go build`.

## User Setup Required
None — no external service configuration required.

## Next Phase Readiness
- TUI-05 is complete: `init`/`index`/`sync` show animated stderr progress on a real terminal and stay byte-identical to today's plain behavior when piped, non-TTY, or `--quiet`.
- Phase 6's full requirement set (TUI-01, TUI-02, TUI-05) is now delivered across 06-01/06-02/06-03; the only carried-forward blocker is the pre-existing Phase-8 item: audit the full `charm.land/...` transitive closure for CGo/govulncheck/SBOM before the v1.0.0 release (already tracked in STATE.md's Blockers/Concerns).
- No new `charm.land/bubbles`/`bubbletea` dependency exists anywhere in `go.mod` — the TUI-01 archtest continues to be the sole, permanent build-time enforcement of that boundary.

---
*Phase: 06-rendering-seam-pretty-status-files*
*Completed: 2026-07-18*

## Self-Check: PASSED

All created files (progress.go, progress_test.go, progress_cli_test.go, this SUMMARY) verified present on disk; all three task commit hashes (412c9c4, 5016d5c, b2aff19) verified present in git log; `go test ./... && go test ./testdata/golden/...` green; `rg 'bubbletea|bubbles' go.mod` returns no matches.
