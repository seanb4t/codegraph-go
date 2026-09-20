---
phase: 04-cli-glow-up
plan: 05
subsystem: cli
tags: [lipgloss, colorprofile, tdd, present, palette, query, markdown-contract]

# Dependency graph
requires:
  - phase: 04-03
    provides: "resolveColor(cmd) resolver, colorMode.Writer, present.Palette/NewPalette, and the shared present.stripANSI test helper (ansistrip_test.go) — the tracer this plan expands"
provides:
  - "internal/cli/present/explore.go: RenderExplore(r query.ExploreResult, pal Palette, w io.Writer) error, writeBlastBullet, joinSymbolKindList, writeNumberedSource, writeNumberedSourceRange, writeSkeleton, pluralize, sourceDisclaimerText"
  - "internal/cli/present/node.go: RenderNode(d query.NodeDetail, pal Palette, w io.Writer) error, writeSingleDef, writeNodeRefs, writeNodeSection, writeMultiDef, nodeMultiDefHardCap/nodeMultiDefBodyBudget/nodeMultiDefListCap (16/12000/20, duplicated from internal/query/render_markdown.go), strippedLen"
  - "explore.go/node.go RunE styled branches calling eng.ExploreDetail/eng.NodeDetail before the frozen eng.Explore/eng.Node markdown call"
affects: [04-06, 04-07, 04-08]

# Actuals (#2632)
actuals:
  tokens: 9217
  tasks: 2
  commits: 3
  plan_head_before: c765226e3984b4566604099b98b42596c8af98e8

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Markdown-syntax-only contract test: markdownToPlainContract(md) strips only **, backticks, a leading \"> \", and ```go/``` fence lines, then asserts stripANSI(styled) == markdownToPlainContract(the EXPORTED query renderer's real output) — anchors the D-07 content contract to real plain output, never a hand-typed string"
    - "Lazy multi-def budget loop reproduced verbatim in present with package-local constant duplicates (16/12000/20, citing internal/query/render_markdown.go — the status.go/formatNumber precedent), measuring the BODY_BUDGET decision via a local ANSI-stripping regexp (strippedLen) so production code never imports the test-only ansistrip_test.go helper"
    - "Styled-branch insertion point unchanged from plan 04's shape: mode := resolveColor(cmd); if mode.Styled { ... return present.RenderX(...) } inserted strictly before the frozen eng.Explore/eng.Node markdown call — a styled run never computes markdown and a plain run never builds the detail struct"

key-files:
  created:
    - internal/cli/present/explore_test.go
    - internal/cli/present/node_test.go
  modified:
    - internal/cli/present/explore.go
    - internal/cli/present/node.go
    - internal/cli/explore.go
    - internal/cli/node.go

key-decisions:
  - "Test fixtures for the MultiDef/File contract tests deliberately avoid embedding literal tabs in synthetic source bodies used purely for HARD_CAP/BODY_BUDGET/LIST_CAP arithmetic — sanitizeControl (CR-01, established in plan 03) strips control bytes including tabs on the styled path only (the plain path is frozen and unsanitized, TUI-02), so a tab-bearing fixture legitimately differs in CONTENT between plain and stripped-styled, not just markdown syntax. The one fixture that deliberately exercises tabs (\"File\") and the dedicated control-byte test build their expectation from sanitizeLines(content) — the same per-line sanitization the styled renderer actually applies — rather than raw plain markdown."
  - "The plan's Task 2 <verify> literal '`**` missing from plain output' check does not hold for the `node -f <file>` (File mode) invocation: File mode's plain markdown is a bare fenced code block (```go ... ```) with no bold markdown anywhere by construction (renderNumberedSource never emits **) — confirmed pre-existing and unrelated to this plan's changes (TestPlainGolden's node-file case, unchanged, still passes 29/29). Verified File mode's plain output retains its fence markers instead, which the plan's own explore/node header checks (Exploration:/Location: lines) do not exercise for this shape."

requirements-completed: [CLI-05]

coverage:
  - id: D1
    description: "explore and node gain a styled branch calling eng.ExploreDetail/eng.NodeDetail and rendering via present.RenderExplore/present.RenderNode — the plain branch (eng.Explore/eng.Node) stays byte-identical; internal/query is zero-diff"
    requirement: "CLI-01"
    verification:
      - kind: unit
        ref: "internal/cli/present/explore_test.go#TestRenderExploreStrippedEqualsMarkdownContract (6/6 subtests)"
        status: pass
      - kind: unit
        ref: "internal/cli/present/node_test.go#TestRenderNodeStrippedEqualsMarkdownContract (8/8 subtests)"
        status: pass
      - kind: integration
        ref: "internal/cli TestPlainGolden (29/29), task test:golden"
        status: pass
    human_judgment: false
  - id: D2
    description: "Styled content keeps the same sections/order/wording as the markdown across every explore/node shape (empty, populated, stale, skeleton, File/SingleDef/MultiDef, and every NODE-02 budget boundary: HARD_CAP=16, BODY_BUDGET=12000, LIST_CAP=20)"
    requirement: "CLI-01"
    verification:
      - kind: unit
        ref: "internal/cli/present/node_test.go#TestRenderNodeStrippedEqualsMarkdownContract/MultiDef-HardCap-17, /MultiDef-BodyBudget, /MultiDef-ListCap-40"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every repo/user-derived string passes sanitizeControl before styling (CR-01); no disk reads inside present (T-04-16); internal/query/internal/mcp/go.mod stay at zero diff"
    requirement: "CLI-01"
    verification:
      - kind: other
        ref: "grep gates: sanitizeControl call-site count (26, floor 8), no os.ReadFile|os.Open in explore.go/node.go, git diff --name-only against internal/query internal/mcp go.mod (empty)"
        status: pass
    human_judgment: false
  - id: D4
    description: "Real-terminal styled output for explore/node is visibly sectioned and reads identically to the markdown minus its syntax"
    verification: []
    human_judgment: true
    rationale: "D-06 explicitly calls hue legibility and 'reads identically' a human UAT item, harvested at end of phase alongside CLI-04 per workflow.human_verify_mode=end-of-phase — not verified here beyond the fully-automated byte-content contract above."

# Metrics
duration: 38min
completed: 2026-09-17
status: complete
---

# Phase 4 Plan 05: Explore/Node Styled Renderers Summary

**`explore` and `node` gain a styled branch consuming `ExploreDetail`/`NodeDetail` — the same sections, order and wording as their markdown, hue replacing `**`/backticks/fences/`> ` — pinned by a syntax-only contract against the exported `query.RenderExplore`/`RenderNode`/`RenderNodeMultiDef`, including all three NODE-02 multi-def budget boundaries (HARD_CAP=16, BODY_BUDGET=12000, LIST_CAP=20).**

## Performance

- **Duration:** 38 min
- **Started:** 2026-09-17T22:03:00Z
- **Completed:** 2026-09-17T22:41:00Z
- **Tasks:** 2
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments

- `internal/cli/present/explore.go` implements `RenderExplore(r query.ExploreResult, pal Palette, w io.Writer) error` covering both the `Empty` zero-result shape and the populated shape (stale banner, symbol/file counts, blast-radius bullets with all three caller-count/test-coverage conditions, the source disclaimer, per-file numbered source or H20 skeleton).
- `internal/cli/present/node.go` implements `RenderNode(d query.NodeDetail, pal Palette, w io.Writer) error` switching on `NodeDetailMode`: File (numbered source, no fences), SingleDef (name/kind, Location, Signature, the fixed Trail line, Calls →/Called by ←, always rendered even when empty), and MultiDef — reproducing `RenderNodeMultiDef`'s exact lazy loop with package-local `nodeMultiDefHardCap`/`nodeMultiDefBodyBudget`/`nodeMultiDefListCap` (16/12000/20) constants, fetching each candidate via `d.Multi.Definition(n)` lazily, in match order, stopping at HARD_CAP.
- `explore_test.go`'s `TestRenderExploreStrippedEqualsMarkdownContract` (6 subtests: empty, empty+stale, populated, populated+stale, populated+skeleton, plus a positive ANSI-presence control) and `node_test.go`'s `TestRenderNodeStrippedEqualsMarkdownContract` (8 subtests: File, SingleDef with/without calls+calledBy, MultiDef at 2/17-HARD_CAP/2-huge-BODY_BUDGET/40-huge-LIST_CAP, plus a control-byte-drop check) both assert `stripANSI(styled) == markdownToPlainContract(plain)`, where `plain` comes from the EXPORTED `query.RenderExplore`/`RenderNode`/`RenderNodeMultiDef` — never a hand-typed string.
- `markdownToPlainContract` (and its own `TestMarkdownToPlainContract` unit test) strips exactly the four syntax elements D-07 names: `**`, backticks, a leading `"> "`, and `` ```go ``/`` ``` `` fence lines — nothing else.
- `explore.go`/`node.go` RunE bodies gained the styled branch (`mode := resolveColor(cmd); if mode.Styled { ... return present.RenderExplore/RenderNode(...) }`) inserted strictly before the existing `eng.Explore`/`eng.Node` markdown call; the plain path (worktree notice + markdown `fmt.Fprint`) is untouched below it.
- Real-binary verification: `explore Alpha`, `node Alpha` and `node -f pkga/pkga.go` all show ANSI escapes under `--color=always`, stay ESC-free on a plain pipe, and the stripped styled output has no `**`/fences while the plain output keeps its markdown syntax (fence markers for File mode, `**` for the other two).
- `TestPlainGolden` (29/29) and `task test:golden` stayed green throughout — the styled branch insertion never touched the plain path's bytes.

## Task Commits

Each task committed atomically (TDD: RED test commit precedes the GREEN feat commit):

1. **Task 1 (RED): markdown-contract tests + placeholder renderers** — `573ae83b` (test)
2. **Task 1 (GREEN): RenderExplore/RenderNode implementations** — `95e96e0f` (feat)
3. **Task 2 (GREEN): explore.go/node.go styled-branch wiring** — `8332f659` (feat)

**Plan metadata:** committed alongside SUMMARY.md/STATE.md/ROADMAP.md/REQUIREMENTS.md in the metadata commit that follows this SUMMARY.

## RED Evidence (pasted verbatim)

`GOTOOLCHAIN=go1.26.6 go test ./internal/cli/present/ -count=1 -run 'TestRenderExploreStrippedEqualsMarkdownContract$|TestRenderNodeStrippedEqualsMarkdownContract$|TestMarkdownToPlainContract$' -v`, against the compiling placeholder `explore.go`/`node.go` (signatures + budget consts only, writing nothing):

```
--- PASS: TestMarkdownToPlainContract (0.00s)
--- FAIL: TestRenderExploreStrippedEqualsMarkdownContract (0.00s)
    --- FAIL: TestRenderExploreStrippedEqualsMarkdownContract/empty (0.00s)
    --- FAIL: TestRenderExploreStrippedEqualsMarkdownContract/empty-stale (0.00s)
    --- FAIL: TestRenderExploreStrippedEqualsMarkdownContract/populated (0.00s)
    --- FAIL: TestRenderExploreStrippedEqualsMarkdownContract/populated-stale (0.00s)
    --- FAIL: TestRenderExploreStrippedEqualsMarkdownContract/populated-skeleton (0.00s)
    --- FAIL: TestRenderExploreStrippedEqualsMarkdownContract/PositiveControl_StyledOutputContainsANSI (0.00s)
--- FAIL: TestRenderNodeStrippedEqualsMarkdownContract (0.00s)
    --- FAIL: TestRenderNodeStrippedEqualsMarkdownContract/File (0.00s)
    --- FAIL: TestRenderNodeStrippedEqualsMarkdownContract/SingleDef (0.00s)
    --- FAIL: TestRenderNodeStrippedEqualsMarkdownContract/SingleDef-nil-refs (0.00s)
    --- FAIL: TestRenderNodeStrippedEqualsMarkdownContract/MultiDef-2 (0.00s)
    --- FAIL: TestRenderNodeStrippedEqualsMarkdownContract/MultiDef-HardCap-17 (0.00s)
    --- FAIL: TestRenderNodeStrippedEqualsMarkdownContract/MultiDef-BodyBudget (0.00s)
    --- FAIL: TestRenderNodeStrippedEqualsMarkdownContract/MultiDef-ListCap-40 (0.00s)
    --- PASS: TestRenderNodeStrippedEqualsMarkdownContract/ControlBytesStrippedFromStyled (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli/present	0.192s
```

Every content-comparison failure is `got: ""` vs a non-empty `want:` (the placeholder renderers wrote nothing) — `ControlBytesStrippedFromStyled` passed trivially against the empty placeholder output (an empty string legitimately contains no control byte) and continued to pass unchanged after GREEN. `TestMarkdownToPlainContract` passed immediately since it exercises the test's own helper function, not the renderers under test. Never gated on `check tdd-red-evidence`, per project rule.

## GREEN Result

`GOTOOLCHAIN=go1.26.6 go test ./internal/cli/present/ -count=1 -run 'TestRenderExploreStrippedEqualsMarkdownContract$|TestRenderNodeStrippedEqualsMarkdownContract$|TestMarkdownToPlainContract$' -v`:

```
--- PASS: TestMarkdownToPlainContract (0.00s)
--- PASS: TestRenderExploreStrippedEqualsMarkdownContract (0.00s)  (6/6 subtests)
--- PASS: TestRenderNodeStrippedEqualsMarkdownContract (0.01s)  (8/8 subtests)
PASS
ok  	github.com/seanb4t/codegraph-go/internal/cli/present	0.171s
```

## Real-Binary Result

```
OK: explore Alpha
OK: node Alpha
OK: node -f pkga/pkga.go
explore-header: OK   (stripped "Exploration: Alpha")
node-location: OK    (stripped "Location: pkga/pkga.go:13")
frozen: OK           (git diff --name-only -- internal/query internal/mcp testdata/golden is empty)
```

## Files Created/Modified

- `internal/cli/present/explore.go` — `RenderExplore`, `writeBlastBullet`, `joinSymbolKindList`, `writeNumberedSource`, `writeNumberedSourceRange`, `writeSkeleton`, `pluralize`, `sourceDisclaimerText`
- `internal/cli/present/node.go` — `RenderNode`, `writeSingleDef`, `writeNodeRefs`, `writeNodeSection`, `writeMultiDef`, the three `nodeMultiDef*` budget constants, `strippedLen`/`nodeSectionAnsi`
- `internal/cli/present/explore_test.go` — `markdownToPlainContract`, `TestMarkdownToPlainContract`, `explorePlainMarkdown`, `TestRenderExploreStrippedEqualsMarkdownContract`
- `internal/cli/present/node_test.go` — `plainNumbered`, `sanitizeLines`, `synthNode`, `multiDefFixture`, `TestRenderNodeStrippedEqualsMarkdownContract`
- `internal/cli/explore.go` — styled branch (`resolveColor` → `eng.ExploreDetail` → `present.RenderExplore`) before the plain `eng.Explore` call
- `internal/cli/node.go` — styled branch (`resolveColor` → `eng.NodeDetail` → `present.RenderNode`) before the plain `eng.Node` call

## Decisions Made

See `key-decisions` in frontmatter for full rationale. Summary: kept the MultiDef budget-boundary fixtures free of literal tabs (sanitizeControl legitimately strips them on the styled path only, per plan 03's established CR-01 contract — a tab-bearing fixture there would test a real, intentional content divergence, not a bug); the one fixture that deliberately carries a tab (File mode) and the dedicated control-byte test build their plain expectation from `sanitizeLines(content)` instead of raw source. Documented that the plan's own literal Task 2 `<verify>` check for `**` in plain output does not apply to the `node -f <file>` invocation, since File mode's plain markdown is a bare fenced block with no bold markdown by construction — verified via its fence marker instead.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test fixtures embedding literal tabs in synthetic source bodies broke the content contract for reasons unrelated to markdown syntax**
- **Found during:** Task 1, first GREEN test run
- **Issue:** `synthNode`'s body-line generator and the "File" fixture's source both embedded a literal tab character. `sanitizeControl` (CR-01, plan 03) strips control bytes — including tabs — on the styled path only; the plain path (`query.RenderNode`/`RenderNodeMultiDef`) is frozen and unsanitized. Comparing raw plain output against sanitized styled output for tab-bearing content therefore failed on a real, intentional CONTENT difference, not a markdown-syntax drift the D-07 contract is meant to catch.
- **Fix:** Removed the tab from `synthNode`'s filler lines (budget-arithmetic fixtures don't need it). Kept the deliberate tab in the "File" fixture (per the plan's own `<behavior>` spec) but built its plain expectation via a new `sanitizeLines(content)` helper — the same per-line `sanitizeControl` transform the styled renderer actually applies — instead of the raw source.
- **Files modified:** `internal/cli/present/node_test.go`
- **Verification:** All 8 `TestRenderNodeStrippedEqualsMarkdownContract` subtests pass.
- **Committed in:** `95e96e0f` (Task 1 GREEN)

**2. [Rule 1 - Bug] `gofmt` flagged `explore_test.go`'s comment alignment after edits**
- **Found during:** Task 1 verify, before the GREEN commit
- **Issue:** Trailing-comment column alignment in the `blasts` slice literal drifted after minor edits, failing `gofmt -l`.
- **Fix:** `gofmt -w internal/cli/present/explore_test.go`.
- **Files modified:** `internal/cli/present/explore_test.go`
- **Verification:** `gofmt -l internal/cli/present/` reports nothing.
- **Committed in:** `95e96e0f` (Task 1 GREEN)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — test-fixture and formatting fixes; zero production renderer logic changed beyond what the plan specified).
**Impact on plan:** No scope creep. Both fixes are test-file-only or formatting-only.

## Issues Encountered

- The plan's Task 2 literal `<verify>` automated script asserts `rg -q '\*\*' plain.txt` uniformly across all three example invocations (`explore Alpha`, `node Alpha`, `node -f pkga/pkga.go`). `node -f pkga/pkga.go`'s plain output (File mode) is `renderNumberedSource`'s bare fenced code block — it has never contained `**` (confirmed via `git show` on the pre-plan tree and via `TestPlainGolden`'s unchanged `node-file` golden). This is a pre-existing structural fact of File mode, not a regression from this plan's changes. Verified the equivalent invariant for that one case via its fence marker (`` ```go ``) instead of `**`; all other assertions in the verify chain (ESC presence/absence, no leaked `**`/fences in stripped styled output, header/location content) ran and passed unmodified for all three invocations.

## Authentication Gates

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `present.RenderExplore`/`present.RenderNode` and their `writeNumberedSource`/`writeNumberedSourceRange` helpers are available to any later plan needing numbered-source rendering (none currently planned to reuse them beyond this plan).
- CLI-01 stays `Pending` in REQUIREMENTS.md — it is a phase-wide requirement shared with plan 06 (which owns `present/line.go`, `init.go`, `index.go`, `sync.go`, `uninit.go`, `version.go`, `telemetry.go`) and plan 07/08; it will flip to `Complete` once every declaring plan finishes (`requirements.ready-ids` confirmed CLI-01 blocked, CLI-05 ready — CLI-05 marked complete this plan).
- `internal/query`, `internal/mcp`, `testdata/golden` and `go.mod` are untouched, confirmed via `git diff --name-only`.
- No blockers for 04-06.

---
*Phase: 04-cli-glow-up*
*Completed: 2026-09-17*

## Self-Check: PASSED

- FOUND: internal/cli/present/explore.go
- FOUND: internal/cli/present/node.go
- FOUND: internal/cli/present/explore_test.go
- FOUND: internal/cli/present/node_test.go
- FOUND: internal/cli/explore.go
- FOUND: internal/cli/node.go
- FOUND: commit 573ae83b (git log --oneline --all)
- FOUND: commit 95e96e0f (git log --oneline --all)
- FOUND: commit 8332f659 (git log --oneline --all)
- Re-ran plan `<verification>`:
  - `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -count=1 -skip 'TestEveryRegisteredFlagIsAccountedFor$'` — all 5 packages `ok`
  - `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1 -run 'TestPlainGolden$' -v` — 29/29 PASS
  - `GOTOOLCHAIN=go1.26.6 task test:golden` — PASS
  - `git diff --name-only -- internal/query internal/mcp` — empty
- `git status --porcelain` — clean.
