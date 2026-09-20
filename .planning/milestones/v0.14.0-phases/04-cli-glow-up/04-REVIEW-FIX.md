---
phase: 04-cli-glow-up
fixed_at: 2026-09-17T00:00:00Z
review_path: .planning/phases/04-cli-glow-up/04-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 04: Code Review Fix Report

**Fixed at:** 2026-09-17
**Source review:** .planning/phases/04-cli-glow-up/04-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope (critical_warning): 4
- Fixed: 4
- Skipped: 0
- Out of scope (Info, not attempted): 1 (IN-01)

Verification ran directly in the main checkout on `gsd/v0.14.0-milestone` (no worktree), per this task's explicit override of the standard worktree-isolation step — the orchestrator instructed working on the main tree on the current branch directly.

## Fixed Issues

### CR-01: `codegraph init` resolves colour twice in one RunE

**Files modified:** `internal/cli/init.go`, `internal/cli/init_advisory_test.go`
**Commit:** `35852c5e`
**Applied fix:** Resolved `colorMode` exactly once at the top of `newInitCmd`'s `RunE` and threaded it into both `printSummaryMode` and `printWatchFallbackAdvisory` (the latter's signature now takes `mode colorMode` as a parameter instead of calling `resolveColor(cmd)` internally), mirroring `sync.go`'s `printSummaryMode` / `daemon.go`'s `printStoppedDaemons(mode, …)` pattern. Added `TestInitAdvisory_ColorResolvedOnce`, a seam-based positive assertion (following `colorflag_test.go`'s `fdIsTerminal`/`queryDarkBackground` stub pattern) that drives the real `init` RunE end-to-end with the watcher disabled and asserts `queryDarkBackground` fires exactly once — this required real `*os.File` stdout/stdin (a temp file and `/dev/null`), since `execCmd`'s `bytes.Buffer` harness can never exercise this gate (the dark-background query requires a genuine `*os.File` on both ends).

### CR-02: `sanitizeControl` strips tabs from real source code

**Files modified:** `internal/cli/present/sanitize.go`, `internal/cli/present/sanitize_test.go`, `internal/cli/present/node_test.go`, `internal/cli/present/status.go`
**Commit:** `fad9879b`
**Applied fix:** Narrowed `sanitizeControl`'s strip predicate to `unicode.IsControl(r) && r != '\t'`, exactly as specified — ESC, CSI/OSC introducers, CR, backspace, and every other C0/C1 control are still stripped; only the literal tab survives. Updated `sanitize_test.go` in both directions (a tab now survives; CR/ESC/DEL are still stripped) and updated `node_test.go`'s `File` subtest comment/fixture.

One nuance surfaced during verification that the review's fix text did not anticipate: narrowing `sanitizeControl` alone is not sufficient to make the styled path byte-identical to plain's raw tab. `pal.Value.Render()` (lipgloss's own `maybeConvertTabs`, `charm.land/lipgloss/v2@v2.0.5`) independently expands any surviving literal tab to 4 spaces before the terminal ever sees it — a lipgloss rendering convention entirely outside `sanitizeControl`'s control. I verified this empirically (the naive fix produced a test failure showing `"func    Façade()"` — 4 spaces — instead of the raw-tab `"func\tFaçade()"` the review's fix comment implied). Rather than widening the change to disable lipgloss's tab conversion (which the task's constraints explicitly ruled out — "Do not widen the change beyond `sanitizeControl`"), I updated `node_test.go`'s comparison helper (`sanitizeLines`) to model this lipgloss-owned tab expansion when building the test's expected value, and documented the distinction in both the fixture comment and `sanitizeControl`'s own doc comment. The net effect for a real user: styled `explore`/`node` output on tab-indented source now shows correct indentation as 4 spaces (previously: indentation was deleted entirely) — a clear improvement, though not literally byte-identical to plain's raw tab. Also corrected a now-stale doc comment in `status.go` claiming `sanitizeControl` strips `\t`.

### WR-01: `files.go` never styles the worktree notice

**Files modified:** `internal/cli/files.go`
**Commit:** `df8f718a`
**Applied fix:** Moved the worktree-mismatch notice print after `resolveColor` and branched it exactly like the six sibling verbs (`explore`, `node`, `search`, `callers`, `callees`, `impact`, `affected`): styled branch calls `present.RenderNotice` then `present.RenderFiles` through the shared `mode.Writer`/`pal`; plain branch is untouched in content and position, so `files-flat.golden`/`files-tree.golden` remain byte-identical.

### WR-02: `lineWriter.Write` returns `(0, err)` on partial failure

**Files modified:** `internal/cli/present/line.go`, `internal/cli/present/line_test.go`
**Commit:** `8d65f2f3`
**Applied fix:** `Write` now accumulates the actual number of bytes of `p` consumed by successfully-written segments (each complete segment's length plus its `\n` separator, or the final segment's length) and returns that count alongside the error on a partial failure, satisfying `io.Writer`'s general contract. Added `failAfterWriter`, a stub that lets the first N `Write` calls through and fails every one after, plus `TestLineWriterPartialFailureReturnsBytesWritten`, which asserts the returned count equals exactly the bytes of the one successfully-written line.

## Skipped Issues (out of scope)

### IN-01: `RenderStatus`'s "Project:" value is sanitized but never styled

**File:** `internal/cli/present/status.go:163`
**Reason:** Info-level finding; `fix_scope` for this run is `critical_warning`, so IN-01 was not attempted. Not a defect that blocks anything — left for a future pass or explicit request.

## Verification

Ran after each fix and again before finishing, all in the main checkout (no worktree):

- `GOTOOLCHAIN=go1.26.6 go build ./...` — clean
- `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/... ./internal/cli/present/...` — all pass
- `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run 'TestPlainGolden$'` — all 29 subtests pass
- `gofmt -l internal/cli/` — no output (clean)
- `GOTOOLCHAIN=go1.26.6 go vet ./internal/cli/...` — clean
- `task docs:cli:drift` — `docs/CLI-REFERENCE.md` byte-identical to a fresh regeneration
- `git diff --stat 51025160..HEAD -- internal/query internal/mcp testdata/golden testdata/wireoracle go.mod` — empty (zero diff, as required)
- Working tree clean apart from this report file (per `git status --short`)

---

_Fixed: 2026-09-17_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
