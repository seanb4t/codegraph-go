---
phase: 04-cli-glow-up
plan: 07
subsystem: cli
tags: [lipgloss, colorprofile, present, palette, githooks, daemon, serve, upgrade, install]

# Dependency graph
requires:
  - phase: 04-03
    provides: "resolveColor(cmd)/resolveColorStderr(cmd), colorMode.Writer, present.Palette/NewPalette — the resolver and palette every verb in this plan wires against"
  - phase: 04-06
    provides: "present.Line/Lines/KV/NewLineWriter (the generic one-line helper) and the printSummaryMode single-resolve precedent for avoiding a double dark-background query per RunE"
provides:
  - "githooks install/remove/status, daemon's printDaemonList/printStoppedDaemons/unlock, ui's URL line, serve's watcher stderr banners, and upgrade's CLI-owned lines all render through resolveColor(Stderr)?(cmd) + present.Line/KV/NewLineWriter"
  - "install/uninstall's shared printAgentResults gains one styled branch (headline, unsupported, per-file action+path, notes, errors) over agents.WriteResult, with errors.Join(errs...) and every plain format literal unchanged"
  - "TestInstall_StyledOutputStripsToPlain — an in-process --color=always vs plain regression pin covering both a normal multi-target install and the write-failure/non-zero-exit shape"
affects: [04-08]

# Actuals (#2632)
actuals:
  tokens: 5695
  tasks: 2
  commits: 2
  plan_head_before: cfdc0c3d

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Tab-separated fields (daemon's pid\\trepo\\tstarted header/rows) are composed with pal.<Role>.Render() calls joined by literal \"\\t\", never passed through present.Line/KV — sanitizeControl treats tab as a control rune and would silently corrupt the column structure if the whole line went through the generic helper"
    - "A `mode := resolveColor(cmd)` resolved ONCE per RunE is threaded into a shared print helper as a parameter (printStoppedDaemons(mode, out, stopped)) rather than re-resolved inside it, extending 04-06's printSummaryMode precedent to daemon stop's two call sites so a styled `daemon stop` never fires the OSC-11 dark-background query twice"
    - "Package-local sanitizePathForDisplay (introduced in uninit.go, 04-06) is reused verbatim across githooks.go, daemon.go and install.go — same package, no re-declaration — for every filesystem-derived string (hooks dir, daemon repo roots, install/uninstall file paths, error text) reaching a styled Path/Error render, per this plan's own T-04-24 threat-register line"

key-files:
  created: []
  modified:
    - internal/cli/githooks.go
    - internal/cli/daemon.go
    - internal/cli/ui.go
    - internal/cli/serve.go
    - internal/cli/upgrade.go
    - internal/cli/install.go
    - internal/cli/install_test.go

key-decisions:
  - "daemon.go's printStoppedDaemons signature changed to printStoppedDaemons(mode colorMode, out io.Writer, stopped []daemon.Record) rather than taking cmd — the plan named this as one of two acceptable shapes ('or receives the palette'); passing the already-resolved mode lets newDaemonStopCmd's RunE reuse ONE resolveColor(cmd) call for both the stopped-daemon lines and the trailing no-running-daemon(s) notice, avoiding the double-query D-11 forbids."
  - "githooks status's `hooks dir: <path>` and per-hook `<name>: <state>` lines are composed manually (pal.Label.Render(...) + \" \" + pal.Path/Value/Warning.Render(...) + \"\\n\") rather than through present.KV, because KV hardcodes the value role to RoleValue and the must_haves specify Path for the hooks-dir value and a state-dependent Warning/Value split for the per-hook line — neither fits KV's fixed two-role shape."
  - "install.go's per-file action role predicate treats agents.ActionUnchanged/ActionKept/ActionNotFound as the no-op set (Label) and everything else (created/updated/removed) as mutating (Warning) — read from internal/agents/types.go's FileAction vocabulary per the plan's read_first instruction, rather than assumed."
  - "TestInstall_StyledOutputStripsToPlain compares plain and styled runs against TWO independent fakeHome() temp directories (never the same one twice, since a second install to an already-configured home reports \"unchanged\" instead of \"created\") and normalizes each run's own absolute home path to a literal <HOME> placeholder before comparing — the same normalizeDir idiom TestPlainGolden's install-local/uninstall-local cases already use."

requirements-completed: [CLI-01]

coverage:
  - id: D1
    description: "githooks install/remove/status, daemon's printDaemonList/printStoppedDaemons/unlock, ui's URL line, and upgrade's CLI-owned lines each render through resolveColor(cmd) + present.Line/Lines/KV/manual Palette composition, byte-identical to the 29 frozen plain goldens when NO_COLOR is set or output is non-TTY"
    requirement: "CLI-01"
    verification:
      - kind: unit
        ref: "internal/cli/plain_golden_test.go#TestPlainGolden (29/29 subtests)"
        status: pass
      - kind: unit
        ref: "internal/cli/plain_golden_test.go#TestNoColorNonTTYRegression (29/29 subtests)"
        status: pass
      - kind: other
        ref: "real-binary: HOME=<fake> codegraph daemon --color=always (ESC present) vs bare pipe (ESC-free), stripped == plain"
        status: pass
    human_judgment: false
  - id: D2
    description: "serve's watcher stderr banners route through resolveColorStderr(cmd) + present.NewLineWriter into serveWatchStart's stderr parameter ONLY — serve --mcp's stdout (the JSON-RPC stream) is never wrapped, and the wire oracle's 38 frozen transcripts stay green"
    requirement: "CLI-05"
    verification:
      - kind: integration
        ref: "GOTOOLCHAIN=go1.26.6 go test ./test/wireoracle/... -count=1 (ok, testdata/wireoracle clean)"
        status: pass
      - kind: other
        ref: "real-binary: serve --mcp -p <fixture> --color=always emits ZERO bytes to stdout; stderr contains an ESC byte and its stripped banner matches the plain [CodeGraph MCP] File watcher disabled line exactly; the plain (no --color) invocation's stderr carries zero ESC bytes"
        status: pass
    human_judgment: false
  - id: D3
    description: "install/uninstall's shared printAgentResults gains a styled branch over agents.WriteResult (headline, unsupported, per-file action+path, notes, errors) with errors.Join(errs...) and every plain format literal unchanged; uninstall.go itself is untouched, reaching the styled branch through the shared function"
    requirement: "CLI-01"
    verification:
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_StyledOutputStripsToPlain"
        status: pass
      - kind: unit
        ref: "internal/cli/install_test.go#TestInstall_WriteFailure_ReportsErrorAndNonZeroExit"
        status: pass
      - kind: other
        ref: "real-binary: install --target claude --location local --color=always vs plain, stripped+home-normalized equality; uninstall.go diff since 04-06's feat commit is empty"
        status: pass
    human_judgment: false
  - id: D4
    description: "On a real terminal the install/uninstall report reads with agent headlines, path lines and errors visibly distinct, and serve --mcp run interactively shows a hued watcher banner on stderr"
    verification: []
    human_judgment: true
    rationale: "D-06/CLI-04's backstop truth explicitly calls this a human UAT item, harvested at end-of-phase per workflow.human_verify_mode=end-of-phase — colour legibility is not unit-testable per D-00 (never test lipgloss's rendering)."

# Metrics
duration: 37min
completed: 2026-09-17
status: complete
---

# Phase 4 Plan 7: CLI-01 verb-list completion — githooks/daemon/ui/serve/upgrade/install Summary

**CLI-01's remaining one-line and report verbs — `githooks`, `daemon`, `ui`, `serve`'s non-MCP stderr banners, `upgrade`, and `install`/`uninstall`'s shared `printAgentResults` — now render through the plan-03/06 resolver, palette and line helpers, with `serve --mcp`'s stdout untouched and the wire oracle, 29 plain goldens, and a new install strip-test all green.**

## Performance

- **Duration:** 37 min
- **Started:** 2026-09-17T22:58:00Z
- **Completed:** 2026-09-17T23:35:00Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments

- `githooks install/remove/status` each resolve colour once per RunE and route their stdout lines through `present.KV` (`Skipped:`), `present.Line` (warnings/values), and a manual `Label + Path` composition for `hooks dir:` and each `<hook>: <state>` line (state gets `Warning` only for "installed but not executable"). `printHookErrors`'s stderr warnings are untouched.
- `daemon`'s `printDaemonList` gains a styled branch: the empty notice in `Label`, the `pid\trepo\tstarted` header in `Header`, and each record's pid/repo/timestamp in `Count`/`Path`/`Value` — tabs are literal separators between independently-styled fields, never passed through `present.Line` (which would strip them as control runes). `printStoppedDaemons` now takes an already-resolved `colorMode` so `daemon stop`'s two call sites (its own line plus the trailing "no running daemon(s)" notice) share ONE `resolveColor(cmd)` call, never double-querying the dark background. `daemon unlock`'s message line styles via `present.Line(RoleValue)`.
- `ui`'s URL line styles via `present.Line(..., present.RolePath, srv.URL())` when styled, still printed before `srv.Serve` blocks.
- `serve`'s watcher stderr banners (the three lines `serveWatchStart` prints) are wrapped in `present.NewLineWriter` under `resolveColorStderr(cmd)` — a stderr-only resolver that never queries the terminal — while `serve --mcp`'s stdout (the JSON-RPC stream) stays completely untouched.
- `upgrade`'s RunE wraps `refreshInstalledSkillsFunc`'s writer in a `present.NewLineWriter(RoleValue)` when styled (the function body itself is untouched — writer-agnostic by design) and its own two-line refresh-failure warning renders as two `present.Line(RoleWarning)` calls; `upgrade.Options.Out` stays the raw, unwrapped `cmd.OutOrStdout()`.
- `install`/`uninstall`'s shared `printAgentResults` gained one styled branch: headline in `Header`+`Value`, the unsupported-location line in `Value`+`Warning`, each per-file `action: path` line in `Warning`-or-`Label` (mutating vs no-op actions, read from `agents.FileAction`'s real vocabulary) + sanitized `Path`, notes in `Label`+`Value`, and errors in `Error` (sanitized). `errors.Join(errs...)` and every plain format literal are unchanged; `uninstall.go` itself has a byte-identical diff since 04-06, reaching the styled branch entirely through the shared function.
- Added `TestInstall_StyledOutputStripsToPlain`: an in-process `--color=always` vs plain comparison across a two-target install and the write-failure/non-zero-exit shape, normalizing each run's own fakeHome path before asserting stripped equality.
- `TestPlainGolden`/`TestNoColorNonTTYRegression` stayed 29/29 throughout both tasks; the wire oracle (`go test ./test/wireoracle/...`) stayed green with `testdata/wireoracle` clean; real-binary checks confirmed `daemon`/`install` strip to their plain output byte-for-byte and `serve --mcp --color=always` writes zero bytes to stdout with a styled stderr banner.

## Task Commits

1. **Task 1: githooks, daemon, ui, serve (stderr banners), upgrade through the helpers** — `e185beea` (feat)
2. **Task 2: install/uninstall styled branch inside printAgentResults, pinned by an in-process strip test** — `f1bb4aad` (feat)

**Plan metadata:** committed alongside SUMMARY.md/STATE.md/ROADMAP.md/REQUIREMENTS.md in the metadata commit that follows this SUMMARY.

## Files Created/Modified

- `internal/cli/githooks.go` — styled branches for install/remove/status subcommands
- `internal/cli/daemon.go` — `printDaemonList`, `printStoppedDaemons` (signature change), `daemon unlock` styled branches
- `internal/cli/ui.go` — styled URL line
- `internal/cli/serve.go` — watcher stderr banner wrapping via `resolveColorStderr`/`present.NewLineWriter`
- `internal/cli/upgrade.go` — `refreshInstalledSkillsFunc` writer wrap + two-line warning styling
- `internal/cli/install.go` — `printAgentResults`'s styled branch
- `internal/cli/install_test.go` — `TestInstall_StyledOutputStripsToPlain`

## Decisions Made

See `key-decisions` in frontmatter for full rationale. Summary: `printStoppedDaemons` takes an already-resolved `colorMode` parameter (one of the plan's two named options) to keep `daemon stop` at one `resolveColor` call per RunE; `githooks status`'s hooks-dir/per-hook lines are composed manually rather than through `present.KV` because KV's value role is fixed to `RoleValue` and these lines need `Path`/state-dependent `Warning`; the install per-file action-role predicate is read from `agents.FileAction`'s real constants (`ActionUnchanged`/`ActionKept`/`ActionNotFound` are the no-op set); and the new install strip-test normalizes each run's own fakeHome path before comparing, following the existing `install-local`/`uninstall-local` golden convention.

## Deviations from Plan

None — plan executed exactly as written. Both tasks' acceptance criteria and automated `<verify>` blocks passed without requiring a Rule 1-4 deviation; the only design choices made (printStoppedDaemons's exact signature, the manual hooks-dir/per-hook composition, and the action-role predicate) were explicitly left to Claude's discretion by the plan's own action text ("pick one signature", "read the real constants before choosing").

## Issues Encountered

The plan's own Task 2 `<verify>` real-binary example used `--target claude-code`, which is not a registered target id in this codebase (the correct id is `claude`, as `install_test.go`'s own `execCmd(...)` calls already use). This was a pre-existing documentation slip in the plan text, not a code defect — verification was carried out with the correct `claude` id instead, and every other assertion in the verify chain passed as written.

## Authentication Gates

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- CLI-01's full verb list is now complete: `explore`/`node`/`search`/`callers`/`callees`/`impact`/`affected`/`files`/`status` (04-03/04-04/04-05), the six lifecycle/one-line verbs (04-06), and now `githooks`/`daemon`/`ui`/`serve`/`upgrade`/`install`/`uninstall` (this plan) all render through the shared resolver, palette and line helpers.
- `internal/query`, `internal/mcp`, `testdata/golden`, `testdata/wireoracle` and `go.mod` are untouched — confirmed via `git diff --name-only` against this plan's two commits.
- **Carried-forward, scheduled RED (not this plan's, not a blocker):** `TestEveryRegisteredFlagIsAccountedFor` and `task docs:cli:drift` still flag `codegraph --color` as undocumented. Per 04-CONTEXT.md's hard sequence and the 03/06 precedent, this closes when plan 08 regenerates `docs/CLI-REFERENCE.md` — do not hand-edit the reference before then.
- **Deferred to end-of-phase human UAT (D4 above, expected per D-06/CLI-04):** eyeballing the install/uninstall report and `serve --mcp`'s interactive stderr banner on a real terminal.
- No blockers for 04-08.

---
*Phase: 04-cli-glow-up*
*Completed: 2026-09-17*

## Self-Check: PASSED

- FOUND: internal/cli/githooks.go (modified)
- FOUND: internal/cli/daemon.go (modified)
- FOUND: internal/cli/ui.go (modified)
- FOUND: internal/cli/serve.go (modified)
- FOUND: internal/cli/upgrade.go (modified)
- FOUND: internal/cli/install.go (modified)
- FOUND: internal/cli/install_test.go (modified)
- FOUND: commit e185beea (git log --oneline --all)
- FOUND: commit f1bb4aad (git log --oneline --all)
- Re-ran plan `<verification>`:
  - `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -count=1` — green except the one scheduled `TestEveryRegisteredFlagIsAccountedFor` RED (recorded above, carried from 04-03/04-06)
  - `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestPlainGolden$|TestNoColorNonTTYRegression$'` — 29/29 PASS
  - `GOTOOLCHAIN=go1.26.6 go test ./test/wireoracle/... -count=1` — ok, `git status --porcelain testdata/wireoracle` empty
  - Real-binary: `daemon --color=always` (ESC), bare pipe (no ESC), stripped == plain; `serve --mcp --color=always` stdout 0 bytes, stderr ESC present and stripped == plain `[CodeGraph MCP]` banner, plain invocation's stderr ESC-free; `install --target claude --location local --color=always` vs plain, stripped+home-normalized equality
- `git diff --diff-filter=D --name-only` across both commits — empty (no unexpected deletions).
- `git status --porcelain` — clean.
