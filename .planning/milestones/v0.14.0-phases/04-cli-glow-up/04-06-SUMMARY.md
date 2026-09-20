---
phase: 04-cli-glow-up
plan: 06
subsystem: cli
tags: [lipgloss, colorprofile, tdd, present, palette, one-line-verbs]

# Dependency graph
requires:
  - phase: 04-03
    provides: "resolveColor(cmd) resolver, colorMode.Writer, present.Palette/NewPalette, and the shared present.stripANSI test helper (ansistrip_test.go) — the tracer this plan expands"
provides:
  - "internal/cli/present/line.go: Line, Lines, KV, NewLineWriter — the generic one-line/one-writer helper D-08 names, consumed directly by plan 07's serve/upgrade stderr wrappers"
  - "init.go/index.go/sync.go's shared printSummary/printSummaryMode/printSyncSummary, printWatchFallbackAdvisory, uninit.go, version.go, telemetry.go each carrying a styled branch through present.Line/Lines and the palette"
affects: [04-07, 04-08]

# Actuals (#2632)
actuals:
  tokens: 5159
  tasks: 2
  commits: 2
  plan_head_before: 4285e87e86129a19860e2f8acd101d5cc50ac838

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Generic one-line helper (present.Line/Lines/KV) over the palette roles: sanitizeControl(text) then pal.Style(role).Render(...) + \"\\n\" — the same idiom present/status.go's writeStatLine already used, generalized into an exported, package-external helper so one-line verbs never hand-roll their own style+sanitize composition"
    - "NewLineWriter: an io.Writer wrapping a Palette+Role, splitting on \"\\n\", styling every complete line and passing an unbuffered partial tail through unstyled — built for plan 07's stderr banner/pass-through consumers, not consumed by this plan's own verbs"
    - "printSummary delegates to an unexported printSummaryMode(cmd, mode, ...) taking an already-resolved colorMode, so printSyncSummary — which needs a SECOND styled line right after printSummary's own — resolves colour exactly ONCE per RunE (D-11) instead of the naive double-resolveColor call a straight reuse of printSummary would have produced"
    - "codegraphDir (uninit.go) sanitized via a package-local sanitizePathForDisplay before styling — a duplicate of present's unexported sanitizeControl across the cli/present boundary, matching this codebase's established convention (e.g. present/status.go's kindCount/formatNumber duplicating internal/query's unexported equivalents) rather than exporting internal-only logic out of present"

key-files:
  created:
    - internal/cli/present/line.go
    - internal/cli/present/line_test.go
  modified:
    - internal/cli/init.go
    - internal/cli/sync.go
    - internal/cli/uninit.go
    - internal/cli/version.go
    - internal/cli/telemetry.go

key-decisions:
  - "printSyncSummary does NOT call the public printSummary — it resolves colour itself once and calls the shared printSummaryMode directly, so calling `codegraph sync` in styled mode never triggers lipgloss.HasDarkBackground's OSC-11 query twice in the same invocation (D-11's 'queried at most once per RunE' contract, RESEARCH Pitfall 2's ~2s-per-query cost)."
  - "uninit.go's codegraphDir is sanitized before pal.Path.Render (Rule 2 — missing critical, security) even though the plan's own action text for uninit.go did not spell out sanitizeControl explicitly; codegraphDir is the same class of value (resolved --path/root argument) present/status.go's own doc comment already calls out as adversarial and sanitizes (CR-01). Implemented as a package-local duplicate (sanitizePathForDisplay) since present/sanitize.go is outside this plan's files_modified scope."
  - "Line(w, pal, RoleValue, \"\")'s output is NOT byte-identical to a bare \"\\n\" — lipgloss.Style.Render(\"\") still wraps empty content in its own prefix/reset ESC codes. stripANSI(out) == \"\\n\" holds (matching fmt.Fprintln(out, \"\")'s plain-path byte shape after stripping), which is the property every other assertion in this plan's test suite checks; line_test.go's own initial literal-equality assertion was wrong and was corrected to a stripANSI comparison before GREEN, not committed as an incorrect RED expectation."

requirements-completed: [CLI-01]

coverage:
  - id: D1
    description: "present.Line/Lines/KV/NewLineWriter implemented per <behavior>: sanitize-then-style, KV's label/value split (label never sanitized), NewLineWriter's split-on-newline partial-tail contract, all pinned by three named tests with a positive ESC-presence control"
    requirement: "CLI-01"
    verification:
      - kind: unit
        ref: "internal/cli/present/line_test.go#TestLineSanitizesAndStyles"
        status: pass
      - kind: unit
        ref: "internal/cli/present/line_test.go#TestLinesAndKV"
        status: pass
      - kind: unit
        ref: "internal/cli/present/line_test.go#TestLineWriterStylesEachLine"
        status: pass
    human_judgment: false
  - id: D2
    description: "init/index/sync's shared printSummary(+printSummaryMode)/printSyncSummary, init's printWatchFallbackAdvisory, uninit, version and telemetry each render styled under --color=always and strip back to byte-identical plain output; version --json stays unstyled; TestPlainGolden/TestNoColorNonTTYRegression stay 29/29"
    requirement: "CLI-01"
    verification:
      - kind: integration
        ref: "internal/cli TestPlainGolden / TestNoColorNonTTYRegression (29/29 subtests)"
        status: pass
      - kind: other
        ref: "real-binary loop: version/telemetry/init/sync/uninit under --color=always (ESC present, stripped==plain via cmp/diff) and version --json --color=always (ESC-free)"
        status: pass
    human_judgment: true
    rationale: "Hue legibility on light/dark terminals (D-06) is a human UAT item harvested at end of phase alongside CLI-04, not verified here — the content/byte contract above is fully automated."

# Metrics
duration: 42min
completed: 2026-09-17
status: complete
---

# Phase 4 Plan 06: One-Line Renderer Helper + Six Lifecycle Verbs Summary

**`present.Line`/`Lines`/`KV`/`NewLineWriter` ship as the generic one-line helper D-08 names, and init/index/sync's shared summary printer, the watch-fallback advisory, uninit, version and telemetry all gain a styled branch through it — plain output stays byte-identical across 29/29 frozen goldens.**

## Performance

- **Duration:** 42 min
- **Started:** 2026-09-17T22:45:00Z
- **Completed:** 2026-09-17T23:27:00Z
- **Tasks:** 2
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments

- `internal/cli/present/line.go` implements `Line`, `Lines`, `KV` and `NewLineWriter` exactly per D-08: every helper routes text through `sanitizeControl` before styling; `Lines` delegates to `Line` per entry; `KV` sanitizes only its value (label is a fixed literal); `NewLineWriter`'s `Write` splits on `"\n"`, styles every complete line, and passes an unbuffered, sanitized partial tail through unstyled.
- `line_test.go`'s three named tests (`TestLineSanitizesAndStyles`, `TestLinesAndKV`, `TestLineWriterStylesEachLine`) pin the sanitize+style contract (including an embedded ESC byte and a mid-string newline both being dropped, verified via `stripANSI`), the KV label/value split, and the two-`Write` partial-tail fixture the plan's own acceptance criteria names — plus a positive ESC-presence control (rule `84d1gfpywd`).
- `init.go`'s `printSummary` now delegates to an unexported `printSummaryMode(cmd, mode, ...)` taking an already-resolved `colorMode` — `printSyncSummary` (sync.go) resolves colour itself exactly once and calls `printSummaryMode` directly for BOTH its lines, so `codegraph sync` in styled mode never fires `lipgloss.HasDarkBackground`'s OSC-11 query twice in one invocation (D-11).
- `printWatchFallbackAdvisory` (init.go) gains a styled branch rendering all four possible lines (warning + frozen-index notice + exactly one of the three guidance lines) via `present.Line`, inserted strictly before the untouched plain branch — the plain branch's `fmt.Fprintf`/`Fprintln` calls are byte-for-byte unchanged.
- `uninit.go` resolves `mode` once at the top of `RunE` and branches per output line (`does not exist`, `aborted`, `Removed git ... hook(s)`, `removed <dir>`); `codegraphDir` is sanitized via a new package-local `sanitizePathForDisplay` before `pal.Path.Render` (CR-01, see Deviations).
- `version.go` gains a styled branch after the `--json` early return, composing `pal.Header`/`pal.Label`/`pal.Value` fragments that strip back to the exact plain `"codegraph %s (commit %s, built %s) %s %s/%s\n"` line.
- `telemetry.go` renders `telemetryStatement` styled via `present.Lines(w, pal, present.RoleValue, strings.Split(telemetryStatement, "\n")...)` — the blank-line paragraph breaks in the const become bare-newline `Line("")` calls, stripping back to byte-identical output including the trailing newline `fmt.Fprintln` would add.
- `internal/cli/...` stays green (5/5 packages) with `-skip 'TestEveryRegisteredFlagIsAccountedFor$'`; `TestPlainGolden`/`TestNoColorNonTTYRegression` stay 29/29; real-binary loop over `version`/`telemetry`/`init`/`sync`/`uninit` confirms ESC bytes under `--color=always`, zero ESC bytes on a bare pipe, `perl`-stripped styled output byte-identical to plain via `cmp`, and `version --json --color=always` emits zero ESC bytes.

## Task Commits

Each task committed atomically (TDD: RED test commit precedes the GREEN feat commit):

1. **Task 1 (RED): present.Line/Lines/KV/NewLineWriter failing tests** — `631d4201` (test)
2. **Task 1+2 (GREEN): line.go implementation; six lifecycle verbs wired** — `864566dc` (feat)

**Plan metadata:** committed alongside SUMMARY.md/STATE.md/ROADMAP.md/REQUIREMENTS.md in the metadata commit that follows this SUMMARY.

## RED Evidence (pasted verbatim)

`GOTOOLCHAIN=go1.26.6 go test ./internal/cli/present/ -count=1 -run 'TestLineSanitizesAndStyles$|TestLinesAndKV$|TestLineWriterStylesEachLine$' -v`, against the compiling placeholder `line.go` (four functions declared, `Line`/`Lines`/`KV` no-ops returning nil, `NewLineWriter` returning `io.Discard`):

```
line_test.go:29: stripANSI(out) = "", want "a[31mbc\n"
line_test.go:32: PositiveControl: styled output "" does not contain an ESC byte
line_test.go:42: Line(..., "") = "", want "\n" (matching fmt.Fprintln(out, ""))
--- FAIL: TestLineSanitizesAndStyles (0.00s)
    --- FAIL: TestLineSanitizesAndStyles/sanitizes_controls_and_styles (0.00s)
    --- FAIL: TestLineSanitizesAndStyles/empty_text_writes_a_bare_newline (0.00s)
line_test.go:70: stripANSI(out) = "", want "x\ny\n"
line_test.go:81: stripANSI(out) = "", want "Files: 4\n"
line_test.go:92: stripANSI(out) = "", want "Label: ab\n"
--- FAIL: TestLinesAndKV (0.00s)
    --- PASS: TestLinesAndKV/Lines_with_zero_texts_writes_nothing (0.00s)
    --- FAIL: TestLinesAndKV/Lines_writes_one_styled_line_per_text (0.00s)
    --- FAIL: TestLinesAndKV/KV_composes_label_and_sanitized_value (0.00s)
    --- FAIL: TestLinesAndKV/KV_sanitizes_only_the_value (0.00s)
line_test.go:130: stripANSI(out) = "", want "one\ntwo\npar"
line_test.go:133: partial tail "par" was not written unstyled/verbatim at the end of ""
line_test.go:136: expected at least 2 styled (ESC-containing) lines, got 0 in ""
--- FAIL: TestLineWriterStylesEachLine (0.00s)
    --- FAIL: TestLineWriterStylesEachLine/two_complete_lines_then_a_partial_tail (0.00s)
    --- PASS: TestLineWriterStylesEachLine/empty_Write_writes_nothing_and_returns_0,_nil (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli/present	0.170s
FAIL
```

Every content-comparison failure is `got: ""` vs a non-empty `want:` (the placeholder wrote nothing via no-op `Line`/`Lines`/`KV` and `io.Discard`-backed `NewLineWriter`) — the two `PASS` subtests (zero-texts `Lines`, empty `Write`) pass trivially against the no-op placeholder, which is expected for those particular fixtures. Never gated on `check tdd-red-evidence`, per project rule.

## GREEN Result

```
--- PASS: TestLineSanitizesAndStyles (0.00s)
    --- PASS: TestLineSanitizesAndStyles/sanitizes_controls_and_styles (0.00s)
    --- PASS: TestLineSanitizesAndStyles/empty_text_writes_a_bare_newline (0.00s)
--- PASS: TestLinesAndKV (0.00s)
    --- PASS: TestLinesAndKV/Lines_with_zero_texts_writes_nothing (0.00s)
    --- PASS: TestLinesAndKV/Lines_writes_one_styled_line_per_text (0.00s)
    --- PASS: TestLinesAndKV/KV_composes_label_and_sanitized_value (0.00s)
    --- PASS: TestLinesAndKV/KV_sanitizes_only_the_value (0.00s)
--- PASS: TestLineWriterStylesEachLine (0.00s)
    --- PASS: TestLineWriterStylesEachLine/two_complete_lines_then_a_partial_tail (0.00s)
    --- PASS: TestLineWriterStylesEachLine/empty_Write_writes_nothing_and_returns_0,_nil (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/cli/present	0.173s
```

## Real-Binary Result

```
OK: version (ESC under --color=always, stripped==plain, ESC-free bare pipe)
OK: telemetry (ESC under --color=always, stripped==plain, ESC-free bare pipe)
init summary OK      (stripped "files=4 nodes=20 edges=22 duration=<N>ms")
init advisory OK     (stripped "Live file watching is disabled here — CODEGRAPH_NO_WATCH=1 is set.")
sync OK              (stripped "reparsed=0 pruned=0 nodesRemoved=0 edgesRemoved=0 dependentsRecomputed=0")
uninit OK            (stripped "removed <FIXTURE>/.codegraph")
version --json unstyled OK (zero ESC bytes under --color=always)
```

## Files Created/Modified

- `internal/cli/present/line.go` — `Line`, `Lines`, `KV`, `NewLineWriter`, `lineWriter`
- `internal/cli/present/line_test.go` — `TestLineSanitizesAndStyles`, `TestLinesAndKV`, `TestLineWriterStylesEachLine`
- `internal/cli/init.go` — `printSummary`/`printSummaryMode` split, `printWatchFallbackAdvisory` styled branch
- `internal/cli/sync.go` — `printSyncSummary` resolves colour once, delegates to `printSummaryMode`, styled second line
- `internal/cli/uninit.go` — styled branch per output line, `sanitizePathForDisplay` helper
- `internal/cli/version.go` — styled branch after `--json` early return
- `internal/cli/telemetry.go` — styled branch via `present.Lines`

## Decisions Made

See `key-decisions` in frontmatter for full rationale. Summary: `printSummary`/`printSummaryMode` split to give `printSyncSummary` a single-resolve path (D-11); `uninit.go`'s `codegraphDir` sanitized via a package-local duplicate of `sanitizeControl` (Rule 2, CR-01 precedent); corrected `line_test.go`'s own empty-text assertion from a literal byte-equality check to a `stripANSI`-based one before GREEN, since `lipgloss.Style.Render("")` legitimately wraps even empty content in real ESC codes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `printSyncSummary` calling the public `printSummary` would double-query the dark background**
- **Found during:** Task 2, implementing `sync.go`'s styled second line
- **Issue:** A straightforward `printSummary(cmd, ...)` followed by `resolveColor(cmd)` again for the second (`reparsed=...`) line would call `resolveColor` — and therefore `lipgloss.HasDarkBackground`'s OSC-11 query, when styled and both stdin/stdout are real terminals — TWICE in one `codegraph sync` invocation, violating D-11's "queried at most once per RunE" contract and doubling the worst-case ~2s stall RESEARCH Pitfall 2 documents.
- **Fix:** Split `printSummary` into a thin `if quiet { return }` + `resolveColor(cmd)` wrapper around a new unexported `printSummaryMode(cmd, mode, ...)`. `printSyncSummary` resolves colour itself once and calls `printSummaryMode` directly for its own first line, then reuses the same `mode` for its second line — `resolveColor(cmd)` is now called exactly once per `sync` invocation, and the literal text still appears inside both `printSummary` (for init/index) and `printSyncSummary` (for sync), satisfying the plan's own acceptance-criteria grep.
- **Files modified:** `internal/cli/init.go`, `internal/cli/sync.go`
- **Verification:** `internal/cli` package green; real-binary `sync` invocation confirmed single styled line pair, byte-identical to plain when stripped.
- **Committed in:** `864566dc` (Task 2 GREEN)

**2. [Rule 2 - Missing critical, security] `uninit.go`'s `codegraphDir` was not sanitized before styling**
- **Found during:** Task 2, implementing `uninit.go`'s styled branches
- **Issue:** The plan's own action text composes `pal.Path.Render(codegraphDir)` directly with no sanitization step. `codegraphDir` derives from the caller's `--path`/positional root argument — the SAME class of value `present/status.go`'s own doc comment calls out as adversarial and sanitizes via `sanitizeControl` before styling (CR-01). Skipping it here would leave a real terminal-escape-injection gap the rest of this phase's threat model explicitly closes everywhere else a resolved path reaches a styled render.
- **Fix:** Added a package-local `sanitizePathForDisplay` (an unexported duplicate of `present.sanitizeControl`'s logic, since `present/sanitize.go` is outside this plan's `files_modified`) and applied it to both `codegraphDir` styling sites (`does not exist` and `removed`).
- **Files modified:** `internal/cli/uninit.go`
- **Verification:** `internal/cli` package green; real-binary `uninit --force --color=always` output correct.
- **Committed in:** `864566dc` (Task 2 GREEN)

**3. [Rule 1 - Bug] `line_test.go`'s own empty-text assertion was byte-literal instead of stripped**
- **Found during:** Task 1, first GREEN run
- **Issue:** The initial test asserted `Line(&b, pal, RoleValue, "")`'s raw output equals a bare `"\n"` exactly — but `lipgloss.Style.Render("")` legitimately wraps even empty content in its own prefix/reset ESC codes (verified empirically), so the raw bytes are never a bare `"\n"` on the styled path.
- **Fix:** Changed the assertion to `stripANSI(b.String()) == "\n"`, consistent with every other assertion in this test file and in the phase's established renderer-contract test style.
- **Files modified:** `internal/cli/present/line_test.go`
- **Verification:** `TestLineSanitizesAndStyles` passes.
- **Committed in:** `864566dc` (Task 2 GREEN). The RED commit (`631d4201`) carries the pre-fix (byte-literal) assertion — the RED transcript above was captured against it, and it failed for the intended reason (empty placeholder output) regardless of which form of the assertion was used, so RED validity is unaffected. The fix was applied afterward, before the real GREEN implementation was written.

---

**Total deviations:** 3 auto-fixed (1 bug — double dark-background query risk; 1 missing-critical/security — path sanitization; 1 bug — a test's own incorrect literal-equality assertion).
**Impact on plan:** All three necessary for correctness/security or for the test suite itself to assert the right property. No scope creep — every fix stays within this plan's `files_modified`.

## Issues Encountered

None beyond the deviations above, all resolved during execution.

## Authentication Gates

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `present.Line`/`Lines`/`KV`/`NewLineWriter` are available for plan 07's serve/upgrade stderr wrappers, which are the intended next consumers of `NewLineWriter` specifically (not exercised by this plan's own six verbs).
- CLI-01 is now declared complete by this plan (it was also declared by 04-04/04-05, both already `Complete` — this plan is the last to finish among CLI-01's declaring plans in this phase's requirements, so `requirements.ready-ids` should mark it complete on this run).
- `internal/query`, `internal/mcp`, `testdata/golden`, `testdata/wireoracle` and `go.mod` are untouched, confirmed via `git diff --name-only`.
- Known scheduled RED (not this plan's): `TestEveryRegisteredFlagIsAccountedFor` flags `codegraph --color` as undocumented until plan 08 regenerates `docs/CLI-REFERENCE.md` — ran with `-skip 'TestEveryRegisteredFlagIsAccountedFor$'` throughout, per project rules.
- No blockers for 04-07/04-08.

---
*Phase: 04-cli-glow-up*
*Completed: 2026-09-17*

## Self-Check: PASSED

- FOUND: internal/cli/present/line.go
- FOUND: internal/cli/present/line_test.go
- FOUND: commit 631d4201 (git log --oneline --all)
- FOUND: commit 864566dc (git log --oneline --all)
- Re-ran plan `<verification>`:
  - `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -count=1 -skip 'TestEveryRegisteredFlagIsAccountedFor$'` — all 5 packages `ok`
  - `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestPlainGolden$'` — 29/29 PASS
  - Real-binary loop (version/telemetry/init/sync/uninit under `--color=always`, `version --json --color=always`) — all assertions passed
- `git diff --diff-filter=D --name-only HEAD~1 HEAD` — empty (no unexpected deletions).
- `git status --porcelain` — clean.
