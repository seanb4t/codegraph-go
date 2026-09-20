---
phase: 04-cli-glow-up
plan: 03
subsystem: cli
tags: [colorprofile, lipgloss, cobra, tdd, resolver, palette, tty]

# Dependency graph
requires:
  - phase: 04-01
    provides: "TestPlainGolden/TestNoColorNonTTYRegression/TestShortFlagsConsistent — the live verify gate this plan's own tracer re-confirmed stayed green (28/28 goldens, byte-identical)"
  - phase: 04-02
    provides: "fang declined (help stays hand-rolled per D-14); present archtest prefix-widened to both charm vanity roots; colorprofile promoted to a direct go.mod requirement"
provides:
  - "internal/cli/colorflag.go: the D-09 shared colour resolver — colorChoice (pflag.Value, auto|always|never), addColorFlag (persistent root flag), colorChoiceOf, rewriteEnviron (environ rewrite + amendments A1/A2), colorMode + (colorMode).Writer, fdIsTerminal/queryDarkBackground seams, resolveColorFrom/resolveColor/resolveColorStderr"
  - "internal/cli/present/palette.go: Role (7 constants), paletteHex (7 light/dark truecolor pairs), Palette (7 lipgloss.Style fields), NewPalette(dark bool), (Palette).Style(Role) — no mutable package-level style"
  - "codegraph status and codegraph files restyled end to end through the resolver + colorprofile.Writer + Palette, plain path byte-identical to the 04-01 goldens"
  - "internal/cli/present/ansistrip_test.go: the one shared stripANSI/sgrSequence helper every later renderer contract test in this package reuses"
affects: [04-04, 04-05, 04-06, 04-07, 04-08]

# Actuals (#2632)
actuals:
  tokens: 13466
  tasks: 2
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single-environment-read resolver: colorflag.go is the ONE place internal/cli reads os.Environ()/--color; every RunE call site wraps its writer in colorprofile.Writer and calls present.NewPalette(mode.Dark) — present itself never reads env/fd"
    - "Injectable func-var seams for terminal probes (fdIsTerminal/queryDarkBackground), mirroring install.go's interactiveAllowed/tui/tty.go's stdinIsInteractive idiom, so the at-most-once dark-background query is unit-testable without a real pty"
    - "One shared package-test ANSI stripper (ansistrip_test.go) instead of each renderer test file defining its own regex — a duplicate-declaration risk this plan pre-empted for the parallel 04/05/06 renderer plans"

key-files:
  created:
    - internal/cli/colorflag.go
    - internal/cli/colorflag_test.go
    - internal/cli/present/palette.go
    - internal/cli/present/palette_test.go
    - internal/cli/present/ansistrip_test.go
  modified:
    - internal/cli/root.go
    - internal/cli/status.go
    - internal/cli/files.go
    - internal/cli/present/styles.go
    - internal/cli/present/tty.go
    - internal/cli/present/status.go
    - internal/cli/present/files.go
    - internal/cli/present/status_test.go
    - internal/cli/present/files_test.go
    - test/integration/status_files_plain_test.go

key-decisions:
  - "present/tty.go and present/styles.go doc comments reworded to drop the literal substrings os.Getenv/term.IsTerminal (kept the same meaning in different words) — the D-03 env-blind grep gate (rg 'os\\.Getenv|os\\.Environ|term\\.IsTerminal|colorprofile' internal/cli/present/*.go) was matching pre-existing PROSE describing the constraint, not code violating it; this is the same recurring substring-proxy gate shape already documented in 04-02 (sha256sum) and 08-01/08-02/08-03 (time.Sleep, continue-on-error) — the established fix in this project is to reword the comment, not the gate."
  - "status_test.go's own ansiRE/stripANSI duplicate removed in favor of the new ansistrip_test.go — Task 2's action text calls this 'the ONE shared ANSI stripper'; leaving both would be a duplicate-declaration compile error."
  - "present/tty.go therefore is NOT byte-identical to its pre-plan state, contradicting one literal Task-1 acceptance-criteria bullet ('internal/cli/present/tty.go is unchanged'). ChoosePresentation's body and two-arg signature are unchanged — only the doc comment above it changed, and only because Task 1's OWN other acceptance criterion (the env-blind grep gate) required it. Both criteria cannot be satisfied literally simultaneously given the file's pre-existing comment text; the resolver-shape criterion (behavioral) was treated as authoritative over the byte-diff criterion (textual)."
  - "Task 2's literal automated <verify> command cannot pass its own '! rg -q ^(--- FAIL|FAIL)' assertion end to end: go test ./internal/cli/... -count=1 legitimately contains one FAIL line (TestEveryRegisteredFlagIsAccountedFor flagging --color as undocumented) — the SAME pre-existing, plan-acknowledged condition Task 1 already introduced and the plan explicitly schedules for plan 08, not this plan. Task 1's own verify script excluded this test by name via -run; Task 2's whole-package run does not. Every OTHER assertion in Task 2's verify chain was run and confirmed individually (see Deviations)."

requirements-completed: [CLI-01, CLI-02, CLI-03, CLI-04, CLI-05]

coverage:
  - id: D1
    description: "Tracer: codegraph status renders through the resolver + colorprofile.Writer end to end; --color=always emits ANSI on a pipe, --color=never/NO_COLOR=1/bare pipe emit none and are byte-identical to the 04-01 golden, --color=bogus exits 1"
    requirement: "CLI-01"
    verification:
      - kind: unit
        ref: "internal/cli/plain_golden_test.go#TestPlainGolden"
        status: pass
      - kind: unit
        ref: "internal/cli/plain_golden_test.go#TestNoColorNonTTYRegression"
        status: pass
      - kind: integration
        ref: "test/integration/status_files_plain_test.go#TestStatusColorFlagRealBinary"
        status: pass
    human_judgment: false
  - id: D2
    description: "--color is a persistent root flag backed by a closed pflag.Value enum (auto|always|never); an unknown or empty value is a usage error naming all three, exit 1 through main.go's single error path"
    requirement: "CLI-03"
    verification:
      - kind: unit
        ref: "internal/cli/colorflag_test.go#TestColorFlagInvalidValue"
        status: pass
      - kind: integration
        ref: "test/integration/status_files_plain_test.go#TestStatusColorFlagRealBinary/--color=bogus_is_a_usage_error_naming_all_three_values"
        status: pass
    human_judgment: false
  - id: D3
    description: "rewriteEnviron applies the D-09 environ rewrite (always drops NO_COLOR/CLICOLOR + forces CLICOLOR_FORCE=1; never drops CLICOLOR_FORCE/CLICOLOR (A1) + sets NO_COLOR=1; auto normalizes a non-empty NO_COLOR to 1 and a ParseBool-false CLICOLOR to +NO_COLOR=1 unless CLICOLOR_FORCE is truthy (A2)), input slice never mutated"
    requirement: "CLI-02"
    verification:
      - kind: unit
        ref: "internal/cli/colorflag_test.go#TestRewriteEnviron"
        status: pass
      - kind: unit
        ref: "internal/cli/colorflag_test.go#TestResolveColorMatrix"
        status: pass
    human_judgment: false
  - id: D4
    description: "lipgloss.HasDarkBackground is queried at most once, only when Styled AND both stdout and stdin are real terminal *os.File values — never on a pipe, never on redirected stdin, never when the plain branch was chosen"
    requirement: "CLI-04"
    verification:
      - kind: unit
        ref: "internal/cli/colorflag_test.go#TestDarkBackgroundQueryGate"
        status: pass
    human_judgment: false
  - id: D5
    description: "present.Palette (7 lipgloss.Style fields) built by NewPalette(dark bool) from one lipgloss.LightDark closure folds the old headerStyle/labelStyle/sectionStyle package vars; RenderStatus/RenderFiles take Palette as an explicit parameter; no mutable package-level style survives"
    requirement: "CLI-04"
    verification:
      - kind: unit
        ref: "internal/cli/present/palette_test.go#TestNewPaletteRolesFollowDark"
        status: pass
      - kind: unit
        ref: "internal/cli/present/status_test.go (all TestRenderStatus_* cases, updated call sites)"
        status: pass
      - kind: unit
        ref: "internal/cli/present/files_test.go (all TestRenderFiles_* cases, updated call sites)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Palette readability on Solarized Light/macOS light Terminal and a dark theme, and correct rendering under TERM=dumb/16-colour/256-colour/truecolor (ideally over SSH/tmux, where a ~2s pause before styled output is the OSC-11 query timing out, not a hang)"
    verification: []
    human_judgment: true
    rationale: "D-06/CLI-04 explicitly call this a human UAT item — colour legibility and downsampling fidelity are not unit-testable per D-00 (never test lipgloss's rendering or colorprofile's downsampling); harvested at end of phase per workflow.human_verify_mode=end-of-phase from this task's <verify><human-check> block."

# Metrics
duration: 32min
completed: 2026-09-17
status: complete
---

# Phase 4 Plan 3: Shared Colour Resolver, --color Flag and Palette Summary

**The phase's tracer verb (`codegraph status`) and `codegraph files` now render through one shared `colorflag.go` resolver — environ rewrite, single `colorprofile.Detect`, at-most-once dual-TTY background query — and a seven-role `present.Palette`, with the plain path proven byte-identical to the 04-01 goldens and `--color` validated against the real binary.**

## Performance

- **Duration:** 32 min
- **Started:** 2026-09-17T21:03:00Z
- **Completed:** 2026-09-17T21:35:12Z
- **Tasks:** 2
- **Files modified:** 15 (5 created, 10 modified)

## Accomplishments

- Built `internal/cli/colorflag.go`: `colorChoice` (a closed `pflag.Value` enum), `addColorFlag` (persistent root flag, default `auto`), `rewriteEnviron` (the D-09 environ rewrite plus amendments A1 — `never` also drops `CLICOLOR_FORCE`/`CLICOLOR` — and A2 — `auto` normalizes a falsy `CLICOLOR` to `NO_COLOR=1` unless `CLICOLOR_FORCE` is truthy), `colorMode`/`resolveColorFrom`/`resolveColor`/`resolveColorStderr`, and the `fdIsTerminal`/`queryDarkBackground` test seams.
- `codegraph status` now resolves colour through `resolveColor(cmd)` and wraps stdout in `mode.Writer(...)` (a `colorprofile.Writer`) before calling `present.RenderStatus` — the `--json` early return and the plain `RenderStatusText` fallback are structurally untouched.
- `codegraph files` gained the same resolver + `colorprofile.Writer` + palette wiring (previously an inline `ChoosePresentation` check), keeping the worktree-notice print before the styled branch in its original position.
- Built `internal/cli/present/palette.go`: `Role` (7 constants), `paletteHex` (7 light/dark truecolor pairs, chosen for Solarized Light/dark legibility — subject to end-of-phase human UAT), `Palette` (7 `lipgloss.Style` fields), `NewPalette(dark bool)` built from one `lipgloss.LightDark(dark)` closure.
- Folded the three package-level `headerStyle`/`labelStyle`/`sectionStyle` vars into `Palette`; `RenderStatus`/`RenderFiles` now take a `Palette` parameter, with section headings via `pal.Header`, labels via `pal.Label`, breakdown/stat numbers via `pal.Count`, advisory labels via `pal.Warning`, and file/directory names/paths via `pal.Path`.
- Added `internal/cli/present/ansistrip_test.go` as the package's one shared `stripANSI`/`sgrSequence` helper, removing the pre-existing duplicate from `status_test.go`.
- Real-binary proof (`test/integration/status_files_plain_test.go#TestStatusColorFlagRealBinary`): `--color=always` emits ANSI on a plain `os/exec` pipe; `--color=never` emits none and is byte-identical to the bare-pipe invocation; `--color=bogus` exits 1 with `auto`/`always`/`never` named in stderr.
- `TestPlainGolden`/`TestNoColorNonTTYRegression` stayed 28/28 green throughout — the resolver/palette wiring never touched the plain path's bytes.
- Both TDD gates followed RED → GREEN discipline: `test(04-03): add failing resolver matrix, environ-rewrite, invalid-flag and background-gate tests` (all four named tests observed `--- FAIL` on assertions, not compile errors) then `feat(04-03): shared colour resolver, --color flag; status restyled...`; `test(04-03): add failing palette role/hue tests` (`--- FAIL: TestNewPaletteRolesFollowDark`) then `feat(04-03): shared colour resolver, --color flag and seven-role palette...`.

## RED Evidence (pasted verbatim)

**Task 1** (`GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestRewriteEnviron$|TestResolveColorMatrix$|TestColorFlagInvalidValue$|TestDarkBackgroundQueryGate$' -v`, against the compiling placeholder `colorflag.go`):

```
--- FAIL: TestRewriteEnviron (0.00s)
    --- FAIL: TestRewriteEnviron/always_drops_NO_COLOR/CLICOLOR,_forces_CLICOLOR_FORCE (0.00s)
    --- FAIL: TestRewriteEnviron/never_drops_CLICOLOR_FORCE/CLICOLOR_(A1),_sets_NO_COLOR (0.00s)
    --- FAIL: TestRewriteEnviron/auto_normalizes_a_non-empty_non-boolean_NO_COLOR_to_1 (0.00s)
    --- PASS: TestRewriteEnviron/auto_leaves_an_empty_NO_COLOR_alone (0.00s)
    --- FAIL: TestRewriteEnviron/auto_appends_NO_COLOR=1_for_a_ParseBool-false_CLICOLOR_(A2) (0.00s)
    --- PASS: TestRewriteEnviron/auto's_A2_normalization_is_skipped_when_CLICOLOR_FORCE_is_truthy (0.00s)
    --- PASS: TestRewriteEnviron/auto_leaves_a_truthy_CLICOLOR_untouched (0.00s)
--- FAIL: TestResolveColorMatrix (0.00s)
    --- FAIL: TestResolveColorMatrix/always/pipe/empty (0.00s)
    --- FAIL: TestResolveColorMatrix/always/pipe/NO_COLOR=1 (0.00s)
    --- FAIL: TestResolveColorMatrix/always/pipe/TERM=dumb (0.00s)
    --- PASS: TestResolveColorMatrix/never/tty/CLICOLOR_FORCE (0.00s)
    --- PASS: TestResolveColorMatrix/never/pipe/CLICOLOR_FORCE (0.00s)
    --- FAIL: TestResolveColorMatrix/auto/pipe/TERM=xterm-256color (0.00s)
    --- FAIL: TestResolveColorMatrix/auto/tty/TERM=xterm-256color (0.00s)
    --- FAIL: TestResolveColorMatrix/auto/tty/TERM=xterm (0.00s)
    --- PASS: TestResolveColorMatrix/auto/tty/TERM=dumb (0.00s)
    --- PASS: TestResolveColorMatrix/auto/tty/NO_COLOR=banana (0.00s)
    --- FAIL: TestResolveColorMatrix/auto/tty/NO_COLOR=empty (0.00s)
    --- PASS: TestResolveColorMatrix/auto/tty/CLICOLOR=0 (0.00s)
    --- FAIL: TestResolveColorMatrix/auto/tty/CLICOLOR=0/CLICOLOR_FORCE=1 (0.00s)
--- FAIL: TestColorFlagInvalidValue (0.10s)
    colorflag_test.go:191: execCmd([status -p ... --color=bogus]): error/stderr missing "auto": "unknown flag: --color "
    colorflag_test.go:202: execCmd(status -p ... --color=never): unexpected error: unknown flag: --color
--- FAIL: TestDarkBackgroundQueryGate (0.00s)
    --- FAIL: TestDarkBackgroundQueryGate/both_stdin_and_stdout_are_terminals:_queried_exactly_once (0.00s)
    --- FAIL: TestDarkBackgroundQueryGate/stdout_is_a_terminal_but_stdin_is_not:_never_queried,_defaults_dark (0.00s)
    --- FAIL: TestDarkBackgroundQueryGate/a_bytes.Buffer_stdout_is_never_a_terminal:_never_queried,_defaults_dark (0.00s)
    --- PASS: TestDarkBackgroundQueryGate/styled_false_(never):_never_queried (0.00s)
FAIL
```

**Task 2** (`GOTOOLCHAIN=go1.26.6 go test ./internal/cli/present/ -count=1 -run 'TestNewPaletteRolesFollowDark$' -v`, against the compiling placeholder `palette.go` with an empty `paletteHex`):

```
palette_test.go:34: paletteHex has no entry for role 0
--- FAIL: TestNewPaletteRolesFollowDark (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli/present	0.211s
```

## Task Commits

1. **Task 1 RED: resolver matrix, environ-rewrite, invalid-flag and background-gate tests** — `fb99e950`
2. **Task 1 GREEN: shared colour resolver, --color flag; status restyled** — `8a80625`
3. **Task 2 RED: palette role/hue tests** — `a396acd8`
4. **Task 2 GREEN: seven-role palette; status and files restyled** — `1054efef`

**Plan metadata:** committed alongside SUMMARY.md/STATE.md/ROADMAP.md/REQUIREMENTS.md in the metadata commit that follows this SUMMARY.

## Files Created/Modified

- `internal/cli/colorflag.go` — the D-09 resolver's whole surface
- `internal/cli/colorflag_test.go` — `TestRewriteEnviron`, `TestResolveColorMatrix`, `TestColorFlagInvalidValue`, `TestDarkBackgroundQueryGate`
- `internal/cli/present/palette.go` — `Role`, `paletteHex`, `Palette`, `NewPalette`, `(Palette).Style`
- `internal/cli/present/palette_test.go` — `TestNewPaletteRolesFollowDark`
- `internal/cli/present/ansistrip_test.go` — the shared `stripANSI`/`sgrSequence`
- `internal/cli/root.go` — `addColorFlag(root)` in `newRootCmd`
- `internal/cli/status.go` — `resolveColor(cmd)` + `mode.Writer(...)` + `present.NewPalette(mode.Dark)`; dropped now-unused `os`/`golang.org/x/term` imports
- `internal/cli/files.go` — same resolver/Writer/palette wiring; dropped the same now-unused imports
- `internal/cli/present/styles.go` — the three style vars removed; doc comment points at `NewPalette`
- `internal/cli/present/tty.go` — doc comment reworded only (see Decisions); `ChoosePresentation`'s body/signature unchanged
- `internal/cli/present/status.go` — `RenderStatus`/`writeStatLine`/`writeBreakdownText`/`writeStatusAdvisories` all take `pal Palette`
- `internal/cli/present/files.go` — `RenderFiles`/`writeFileTree` take `pal Palette`
- `internal/cli/present/status_test.go` — call sites pass `NewPalette(true)`; duplicate `stripANSI` removed
- `internal/cli/present/files_test.go` — call sites pass `NewPalette(true)`
- `test/integration/status_files_plain_test.go` — `TestStatusColorFlagRealBinary`

## Decisions Made

See `key-decisions` in frontmatter for full rationale. Summary: reworded `present/tty.go` and `present/styles.go` doc comments to stop tripping the D-03 env-blind grep gate on descriptive prose (the third instance of this project's own recurring substring-proxy gate defect); removed `status_test.go`'s duplicate `stripANSI` in favor of the new shared `ansistrip_test.go`; accepted that `present/tty.go` is not byte-identical to its pre-plan state (comment-only change, function behavior unchanged) because Task 1's OWN env-blind gate required the edit; and recorded that Task 2's literal whole-package `<verify>` cannot show zero `FAIL` lines because of the plan's own pre-announced `--color`-undocumented RED window (scheduled for plan 08), which Task 1's narrower `-run`-filtered verify script does not hit.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `present/tty.go`/`present/styles.go` doc comments tripped the D-03 env-blind grep gate on prose, not code**
- **Found during:** Task 1, running the tracer's full automated `<verify>` script for the first time
- **Issue:** `test -z "$(rg -l 'os\.Getenv|os\.Environ|term\.IsTerminal|colorprofile' internal/cli/present/*.go | rg -v '_test\.go')"` matched `tty.go` and `styles.go` — both had pre-existing doc comments literally containing the phrase "os.Getenv or call term.IsTerminal" to DESCRIBE the constraint, not violate it. This predates the plan (confirmed via `git show <pre-plan HEAD>:internal/cli/present/tty.go`).
- **Fix:** Reworded both comments to convey the same meaning ("must NOT read the process environment or probe terminal state itself") without the literal matched substrings. No function body or signature changed.
- **Files modified:** `internal/cli/present/tty.go`, `internal/cli/present/styles.go` (styles.go's comment was further revised in Task 2 alongside the var removal)
- **Verification:** `test -z "$(rg -l '...' internal/cli/present/*.go | rg -v '_test\.go')"` now passes; `ChoosePresentation`'s behavior is unchanged (`TestChoosePresentation` still green).
- **Committed in:** `8a80625` (Task 1 GREEN)

**2. [Rule 1 - Bug] `status_test.go`'s own `ansiRE`/`stripANSI` collides with the new shared `ansistrip_test.go`**
- **Found during:** Task 2, implementing the shared ANSI stripper the plan's action text calls for
- **Issue:** `status_test.go` already declared a package-level `stripANSI`/`ansiRE`. Adding `ansistrip_test.go`'s own `stripANSI`/`sgrSequence` without removing the duplicate would be a compile-time redeclaration error.
- **Fix:** Removed `ansiRE`/`stripANSI` from `status_test.go`, replaced with a one-line comment pointing at `ansistrip_test.go`. Behavior identical (same regex).
- **Files modified:** `internal/cli/present/status_test.go`
- **Verification:** `go build`/`go vet` clean; all `TestRenderStatus_*` cases pass.
- **Committed in:** `1054efef` (Task 2 GREEN)

**3. [Rule 2 - Missing critical, applied conservatively] `headerStyle`/`labelStyle`/`sectionStyle` literal-name grep gate widened to whole-package prose**
- **Found during:** Task 2, verifying `rg -n 'headerStyle|labelStyle|sectionStyle' internal/cli/present/` matches nothing outside the SUMMARY (Task 2's own acceptance criterion)
- **Issue:** Two DOC-COMMENT prose references remained after the var removal — one in `status_test.go` explaining why `TestRenderStatus_SanitizesControlChars` tolerates unrelated ESC sequences, one in `styles.go`'s own new doc comment describing what folded into what.
- **Fix:** Reworded both to describe the same facts without the literal old identifier names.
- **Files modified:** `internal/cli/present/status_test.go`, `internal/cli/present/styles.go`
- **Verification:** `rg -n 'headerStyle|labelStyle|sectionStyle' internal/cli/present/` now matches nothing.
- **Committed in:** `1054efef` (Task 2 GREEN)

---

**Total deviations:** 3 auto-fixed (all Rule 1 — comment/duplicate-declaration fixes required to satisfy the plan's own acceptance gates; zero behavioral changes to any renderer or the resolver).
**Impact on plan:** No scope creep — every fix is a doc-comment reword or a test-only duplicate removal, made necessary by other acceptance criteria in the SAME two tasks. No production rendering or resolution logic changed beyond what the plan specified.

## Issues Encountered

**Task 1's "`present/tty.go` is unchanged" acceptance criterion and Task 1's own env-blind grep gate are in tension** given the file's pre-existing comment text — satisfying one literally breaks the other. Resolved by treating the behavioral criterion (pure function, unchanged signature, present stays env-blind) as authoritative over the textual byte-diff criterion; documented rather than silently picked.

**Task 2's literal automated `<verify>` cannot show zero `FAIL` lines** across the whole `./internal/cli/...` package, because `TestEveryRegisteredFlagIsAccountedFor` legitimately (and, per 04-CONTEXT.md, expectedly) flags the new `--color` flag as undocumented until plan 08 regenerates `docs/CLI-REFERENCE.md`. This is the same condition Task 1 introduced and is explicitly scheduled, not a regression from Task 2's work. Every other assertion in Task 2's verify chain (package-`ok` count ≥ 4, palette test PASS, role/signature greps, `-race` clean of `DATA RACE`, RED-then-GREEN commit ordering) was run and confirmed individually.

## Authentication Gates

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The colour-resolver architecture (`colorflag.go`) and the seven-role `Palette` are proven end to end on `status` and `files` — every later plan wiring `explore`/`node`/`search`/`callers`/`callees`/`impact`/`affected`/one-line verbs consumes the SAME `resolveColor`/`NewPalette`/`colorprofile.Writer` shape, no new architecture needed.
- `TestPlainGolden`/`TestNoColorNonTTYRegression` (28/28) and `TestNoCharmInServeReachablePackages`/`TestCharmCgoClosure` all stayed green — nothing in this plan widened the serve-reachable charm-import closure or touched the plain path's bytes.
- **Carried-forward, scheduled RED (not a blocker):** `TestEveryRegisteredFlagIsAccountedFor` and `task docs:cli:drift` both flag the new `--color` persistent flag as undocumented. Per 04-CONTEXT.md's hard sequence and the 03-01 precedent, this closes when plan 08 regenerates `docs/CLI-REFERENCE.md` — do not add an allowlist entry or hand-edit the reference before then.
- **Deferred to end-of-phase human UAT (D6 above, expected per D-06/CLI-04):** palette readability on light/dark terminals and correct rendering across `TERM=dumb`/16-colour/256-colour/truecolor, ideally including one SSH/tmux run to observe the OSC-11 query's worst-case ~2s pause (D-11 correction) as expected behavior, not a hang.
- No blockers for 04-04.

---
*Phase: 04-cli-glow-up*
*Completed: 2026-09-17*

## Self-Check: PASSED

- FOUND: internal/cli/colorflag.go
- FOUND: internal/cli/colorflag_test.go
- FOUND: internal/cli/present/palette.go
- FOUND: internal/cli/present/palette_test.go
- FOUND: internal/cli/present/ansistrip_test.go
- FOUND: commit fb99e950 (git log --oneline --all)
- FOUND: commit 8a80625 (git log --oneline --all)
- FOUND: commit a396acd8 (git log --oneline --all)
- FOUND: commit 1054efef (git log --oneline --all)
- Re-ran plan `<verification>`:
  - `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -count=1` — green except the one scheduled `TestEveryRegisteredFlagIsAccountedFor` RED (recorded above)
  - `GOTOOLCHAIN=go1.26.6 go test ./test/integration/ -run 'TestStatusColorFlagRealBinary|TestStatusFilesPlainByteIdentity' -count=1` — PASS (7/7 subtests)
  - `GOTOOLCHAIN=go1.26.6 go test -race -count=1 ./internal/cli/ ./internal/cli/present/` — no DATA RACE (the one FAIL line is the same scheduled `--color` doc-drift condition)
  - `rg -n 'os\.Getenv|os\.Environ|term\.IsTerminal|colorprofile' internal/cli/present/*.go` — no non-test hits
  - `task docs:cli:drift` — RED as expected (scheduled window)
- `git status --porcelain` — clean.
