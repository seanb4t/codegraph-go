---
phase: 02-status-content-git-worktree-awareness
plan: 07
subsystem: cli
tags: [cli, status, worktree, gitmeta, tdd, reachability]

requires:
  - phase: 02-status-content-git-worktree-awareness
    provides: "02-04's Engine.WorktreeMismatch()/query.WorktreeNotice/query.WorktreeWarningBlockquote — this plan's CLI call sites for both"
  - phase: 02-status-content-git-worktree-awareness
    provides: "02-05's query.RenderStatusText(r, projectPath) — this plan's sole caller"
provides:
  - "internal/cli/status.go: newStatusCmd's human branch renders D-09's sectioned layout via query.RenderStatusText, replacing the terse backend=/files=/stale= one-liner; --json unchanged"
  - "internal/cli/{explore,node,search,callers,callees,impact,files}.go: all 7 non-status read commands print the compact worktree notice on a borrowed-index mismatch, nothing on a clean tree, never in --json output"
  - "internal/cli/status_cli_test.go + internal/cli/notice_test.go: the phase's CLI-side reachability proof — real command invocations, not renderer-in-isolation tests"
affects: []

tech-stack:
  added: []
  patterns:
    - "Notice placement asymmetry: explore/node (no --json mode) print unconditionally right after OpenAt succeeds; the 5 --json-flagged commands print strictly inside the human-output branch, after the `if jsonOut { …; return }` early return — never before it"
    - "containsBareNoticeGlyph (test-only): a byte-aware containment check distinguishing the bare U+26A0 D-11 notice glyph from Phase 1's U+26A0+U+FE0F emoji-presentation warning glyph, whose bytes are otherwise a false-positive prefix match under naive strings.Contains"

key-files:
  created:
    - internal/cli/status_cli_test.go
    - internal/cli/notice_test.go
  modified:
    - internal/cli/status.go
    - internal/cli/explore.go
    - internal/cli/node.go
    - internal/cli/search.go
    - internal/cli/callers.go
    - internal/cli/callees.go
    - internal/cli/impact.go
    - internal/cli/files.go

key-decisions:
  - "Doc-comment rewording in status.go was restructured to preserve the original '// --json emits query.MarshalStatusJSON's StatusResult shape.' line byte-for-byte at its own position, appending new commentary as additional lines instead of rewriting it in place — the plan's literal `git diff | grep -c '^-.*MarshalStatusJSON'` returns-0 acceptance gate would otherwise false-fail on a pure doc-comment reword that happens to touch a line containing that substring"
  - "explore.go's local `query := strings.Join(args, \" \")` variable was renamed to `exploreQuery` to keep the new `query.WorktreeNotice(eng.WorktreeMismatch())` call (inserted before that declaration) unambiguous, even though Go's scoping already made the original shadow safe at that insertion point"
  - "Test 8/9's glyph-containment check cannot use naive strings.Contains(out, glyph): Phase 1's EXPL-04 \"⚠️ no covering tests found\" warning uses U+26A0+U+FE0F, whose UTF-8 bytes have the bare D-11 glyph's bytes as a literal prefix, so a naive check on `explore`'s clean-tree output false-positives. containsBareNoticeGlyph walks all occurrences and rejects any immediately followed by the U+FE0F variation-selector bytes (verified via a python3 byte dump: ef b8 8f)"
  - "The plan's Task 2 acceptance criterion \"go test ./internal/cli/... fully passes\" is only literally true after Task 3 also lands — Task 1's RED commit added tests for BOTH Task 2's (status) and Task 3's (notice) scope in one file set, so the full package is legitimately still red between Task 2 and Task 3, matching standard TDD RED→GREEN→GREEN sequencing rather than a deviation"

requirements-completed: [STAT-01, STAT-02, STAT-03, WORK-02]

coverage:
  - id: D1
    description: "codegraph status prints the sectioned layout (Index Statistics:, Nodes by Kind:, Files by Language:, DB Size as a well-formed two-decimal MB value) instead of the terse one-liner, driven through the real command"
    requirement: STAT-01
    verification:
      - kind: integration
        ref: "internal/cli/status_cli_test.go#TestStatusCmdSections/Test_1"
        status: pass
      - kind: integration
        ref: "internal/cli/status_cli_test.go#TestStatusCmdSections/Test_4"
        status: pass
    human_judgment: false
  - id: D2
    description: "Nodes-by-Kind and Files-by-Language breakdowns are reachable through the real status command"
    requirement: STAT-02
    verification:
      - kind: integration
        ref: "internal/cli/status_cli_test.go#TestStatusCmdSections/Test_2"
        status: pass
    human_judgment: false
  - id: D3
    description: "The live staleness/reindex advisory is reachable through the real status command, not just internal/query"
    requirement: STAT-03
    verification:
      - kind: integration
        ref: "internal/cli/status_cli_test.go#TestStatusCmdSections/Test_3"
        status: pass
    human_judgment: false
  - id: D4
    description: "status --json still emits valid JSON containing a positive dbSizeBytes; MarshalStatusJSON's body is untouched"
    requirement: STAT-01
    verification:
      - kind: integration
        ref: "internal/cli/status_cli_test.go#TestStatusCmdSections/Test_5"
        status: pass
      - kind: unit
        ref: "git diff internal/cli/status.go | grep -c '^-.*MarshalStatusJSON' returns 0"
        status: pass
    human_judgment: false
  - id: D5
    description: "status prints the verbose worktree warning (opening line, Running in:, Index from:) from a real borrowed-index worktree fixture"
    requirement: WORK-02
    verification:
      - kind: integration
        ref: "internal/cli/status_cli_test.go#TestStatusCmdSections/Test_6"
        status: pass
    human_judgment: false
  - id: D6
    description: "All 7 non-status read commands (explore/node/search/callers/callees/impact/files) print the compact worktree notice as the first thing in their output, driven through a real .claude/worktrees/ fixture, including one row using a RELATIVE --path (Pitfall-3 absolutization guard)"
    requirement: WORK-02
    verification:
      - kind: integration
        ref: "internal/cli/notice_test.go#TestNoticeOnWorktreeMismatch"
        status: pass
    human_judgment: false
  - id: D7
    description: "None of the 8 commands emit the notice glyph against a clean, non-borrowed fixture"
    requirement: WORK-02
    verification:
      - kind: integration
        ref: "internal/cli/notice_test.go#TestNoticeAbsentOnCleanTree"
        status: pass
    human_judgment: false
  - id: D8
    description: "The notice never appears in --json output on any command, even against a real worktree-mismatch fixture where the human branch shows it"
    requirement: WORK-02
    verification:
      - kind: integration
        ref: "internal/cli/notice_test.go#TestNoticeSuppressedInJSON"
        status: pass
    human_judgment: false
  - id: D9
    description: "The SURF-06 --json regression guard (TestSearchCmd/TestCallersCalleesCmd/TestImpactCmd/TestFilesCmd) stays green with query_cli_test.go byte-for-byte untouched"
    requirement: STAT-01
    verification:
      - kind: integration
        ref: "internal/cli/query_cli_test.go#TestSearchCmd,TestCallersCalleesCmd,TestImpactCmd,TestFilesCmd"
        status: pass
      - kind: unit
        ref: "git diff --stat internal/cli/query_cli_test.go (empty)"
        status: pass
    human_judgment: false

duration: 25min
completed: 2026-07-15
status: complete
---

# Phase 2 Plan 7: CLI Wiring — Sectioned Status + Compact Worktree Notice Summary

**`codegraph status` now prints TS's real sectioned layout (DB size, node/file breakdowns, live staleness advisory) instead of a terse one-liner, and the 7 other read commands prefix a compact borrowed-index notice — closing STAT-01/02/03 and WORK-02's CLI reachability gap, the final plan of Phase 2.**

## Performance

- **Duration:** 25 min
- **Tasks:** 3 (TDD RED / GREEN / GREEN)
- **Files created:** 2
- **Files modified:** 8

## Accomplishments

- `codegraph status` (human mode) now renders `query.RenderStatusText(result, start)`: `CodeGraph Status` header, `Project: <path>`, the verbose worktree warning when the index is borrowed, `Index Statistics:` (Files/Nodes/Edges/DB Size/Backend), `Nodes by Kind:` and `Files by Language:` breakdowns, and the live staleness/reindex advisory — replacing the old `backend=… files=… stale=…` one-liner (D-09). `--json` is byte-for-byte unchanged: `MarshalStatusJSON`'s body was never touched.
- `explore`, `node`, `search`, `callers`, `callees`, `impact`, and `files` all now print `query.WorktreeNotice(eng.WorktreeMismatch())` as the first thing in their human-mode output on a borrowed-index mismatch, and nothing on a clean tree (WORK-02, D-12). For the 5 commands with a `--json` flag, the print lives strictly inside the human-output branch, after the `if jsonOut {...; return}` early return — the notice can never leak into `--json` output.
- ★ **Reachability closed** (the Phase 1 CR-02 lesson this plan exists to prevent recurring): every one of STAT-01/02/03 and WORK-02 is now proven through the real `codegraph status`/`explore`/`node`/`search`/`callers`/`callees`/`impact`/`files` commands, not a renderer or detector tested in isolation. `status_cli_test.go` and `notice_test.go` drive the actual cobra command tree end-to-end (`execCmd`), including a real `git worktree add`-built `.claude/worktrees/probe` fixture.
- Manually verified end-to-end against a real, disposable `.claude/worktrees/` fixture outside this repo's own checkout: `status` prints the verbose warning with `Running in:`/`Index from:` rows, and `callers` prints the compact notice as its first line.

## Task Commits

Each task was committed atomically (TDD RED/GREEN/GREEN):

1. **Task 1: RED — sectioned status output + notice reachability through real CLI commands** — `a40fd37` (test)
2. **Task 2: GREEN — the sectioned status layout** — `3fd3f38` (feat)
3. **Task 3: GREEN — the compact notice on the 7 other read commands** — `4cd4f1e` (feat)

No REFACTOR commit was needed.

## Files Created/Modified

- `internal/cli/status_cli_test.go` (new) — `TestStatusCmdSections`, 6 subtests: DB Size MB formatting, section headers, live stale/reindex advisory reachability, absence of the terse one-liner tokens, `--json` `dbSizeBytes`, and the verbose worktree warning from a real fixture.
- `internal/cli/notice_test.go` (new) — `TestNoticeOnWorktreeMismatch` (table over the 7 non-status commands, one row using a relative `--path`), `TestNoticeAbsentOnCleanTree` (all 8 commands), `TestNoticeSuppressedInJSON` (all 6 `--json`-capable commands). Includes local `runGitC`/`statusWorktreeMismatchFixture` helpers (a third package-local copy of the D-15 fixture pattern, since Go test helpers aren't importable across packages) and `noticeGlyph`/`containsBareNoticeGlyph` for glyph-sourced, byte-aware assertions.
- `internal/cli/status.go` — human branch now calls `query.RenderStatusText`; doc comment updated to describe the new layout while preserving the original `--json` line verbatim.
- `internal/cli/explore.go` — notice print inserted after `OpenAt`; carries the "no TS precedent" rationale comment (the one place this comment lives, referenced by the other 6 files).
- `internal/cli/node.go`, `search.go`, `callers.go`, `callees.go`, `impact.go`, `files.go` — notice print inserted at the correct placement for each command's shape (unconditional for node; after the `--json` early return for the other 5).

## Decisions Made

See `key-decisions` in frontmatter for the full list. Highlights:
- Preserved the exact original `--json emits query.MarshalStatusJSON's StatusResult shape.` doc-comment line at its own position rather than rewriting it in place, so the plan's literal `git diff | grep '^-.*MarshalStatusJSON'` gate (checking the `--json` branch is untouched) doesn't false-fail on a pure comment reword.
- Built a glyph-aware `containsBareNoticeGlyph` test helper rather than a naive `strings.Contains`, after discovering Phase 1's `⚠️` (U+26A0+U+FE0F) "no covering tests" warning shares the D-11 notice's base byte sequence as a literal prefix — a real false-positive caught during RED verification, not a hypothetical.

## Real `codegraph status` Output (against this repo)

```
CodeGraph Status

Project: /Volumes/Code/github.com/seanb4t/codegraph-go

Index Statistics:
  Files:     308
  Nodes:     3,038
  Edges:     7,367
  DB Size:   1.18 MB
  Backend:   pebble
Nodes by Kind:
  function        1,610
  method          591
  file            308
  constant        208
  struct          185
  variable        69
  package         31
  type_alias      17
  interface       15
  route           4
Files by Language:
  go              290
  python          8
  typescript      4
  java            3
  csharp          2
  javascript      1

Pending Changes: a sync is recommended — this index may be stale. Run "codegraph sync" to update.
```

`codegraph status --json` continues to emit `dbSizeBytes: 1237973` alongside the full `StatusResult` shape, confirming both surfaces are live simultaneously.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test 8/9's clean-tree glyph check false-positived against Phase 1's EXPL-04 warning**
- **Found during:** Task 1 (RED verification)
- **Issue:** `TestNoticeAbsentOnCleanTree/explore` failed even though no notice wiring existed yet: the naive `strings.Contains(out, glyph)` matched inside `explore`'s pre-existing `⚠️ no covering tests found` warning, because the bare D-11 glyph (U+26A0) is a literal byte-prefix of the emoji-presentation variant (U+26A0+U+FE0F) that warning uses.
- **Fix:** Added `containsBareNoticeGlyph`, which walks every occurrence of the bare glyph and rejects any immediately followed by the U+FE0F variation-selector bytes (`ef b8 8f`, confirmed via a python3 byte dump). Both Test 8 and Test 9 now use this instead of `strings.Contains`.
- **Files modified:** `internal/cli/notice_test.go`
- **Verification:** `TestNoticeAbsentOnCleanTree/explore` passes both before notice wiring existed (correctly showing no false positive) and after (correctly showing no true positive either, since a clean tree has no mismatch).
- **Committed in:** `a40fd37` (Task 1 RED commit — fixed before RED was finalized, so no separate commit was needed)

---

**Total deviations:** 1 auto-fixed (test-infrastructure only — no product code affected; a genuine bug in the RED test's own assertion, caught by RED verification working as intended).
**Impact on plan:** No scope creep. The fix sharpened the test's precision without weakening what it proves.

## Issues Encountered

The plan's Task 2 acceptance criterion "`go test ./internal/cli/...` fully passes" is only literally achievable after Task 3 also lands, since Task 1's single RED commit added tests for both Task 2's and Task 3's scope in one file set. This is standard TDD RED→GREEN→GREEN sequencing (mirroring the precedent 02-05-SUMMARY.md documented for an analogous whole-package-compilation constraint), not a defect — verified by re-running the full acceptance-criteria checklist, including the full suite, after Task 3 completed.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 2 (status Content & Git/Worktree Awareness) is now fully delivered on both surfaces: MCP (02-06) and CLI (this plan). All 8 phase requirements (STAT-01/02/03, WORK-01/02/03, TEST-02, SURF-06) are reachable through real entry points, closing the phase's own CR-02 recurrence risk.
- `go test ./...` is fully green; `go.mod`/`go.sum` are byte-identical to before this plan.
- `git diff --stat internal/cli/query_cli_test.go internal/query/ go.mod go.sum` is empty — the SURF-06 `--json` regression guard, `internal/query`, and dependencies are all untouched by this plan.
- No `lipgloss`/`charm.land` imports introduced anywhere in `internal/cli` — Phase 6 (TUI-02) still owns colorization with a clean plain-text foundation to build on.
- No blockers for Phase 3 or later phases.

---
*Phase: 02-status-content-git-worktree-awareness*
*Completed: 2026-07-15*
