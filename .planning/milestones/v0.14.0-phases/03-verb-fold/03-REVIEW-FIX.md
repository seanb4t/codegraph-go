---
phase: 03-verb-fold
fixed_at: 2026-09-16T23:20:00Z
review_path: .planning/phases/03-verb-fold/03-REVIEW.md
iteration: 1
findings_in_scope: 1
fixed: 1
skipped: 0
status: all_fixed
---

# Phase 03: Code Review Fix Report

**Fixed at:** 2026-09-16T23:20:00Z
**Source review:** .planning/phases/03-verb-fold/03-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 1 (WR-01; IN-01 is Info and out of the `critical_warning` fix scope — see "Out of Scope" below)
- Fixed: 1
- Skipped: 0

**Verification environment:** main checkout (`/Volumes/Code/github.com/seanb4t/codegraph-go`, branch `gsd/v0.14.0-milestone`) — `workflow.use_worktrees` is `false` for this project, so no isolated worktree was created; all edits, builds, and test runs happened directly in the checkout the user already had open.

## Fixed Issues

### WR-01: Rename stubs print a redundant third stderr line in the real binary, untested by the unit suite

**Files modified:** `internal/cli/renamed.go`, `internal/cli/renamed_test.go`, `test/integration/renamed_stubs_test.go`
**Commit:** `fa81672c`

**Applied fix:** Option (a) from REVIEW.md's Fix section, shaped per the orchestrator's decision so the user still sees exactly the two D-06 lines:

- `internal/cli/renamed.go`: both stubs' `RunE` no longer call `fmt.Fprintln(cmd.ErrOrStderr(), ...)` themselves. Each now returns a single `fmt.Errorf` whose `Error()` text IS the whole two-line D-06 message (the rename-notice line and the removal-lifetime line, joined by one `\n`, no `codegraph: ` prefix, no trailing newline). `cmd/codegraph/main.go`'s existing `fmt.Fprintln(os.Stderr, err)` then prints exactly those two lines once — the prior code path printed them via `Fprintln` *and* had `main.go` print a third, partially-duplicate line derived from the old short error text. `Hidden`, `DisableFlagParsing`, the `Short` strings, and the file's doc-comment discipline (no new spellings of the retired `codegraph query`/`codegraph unlock` invocations) were preserved; the doc comment's paragraph describing the old two-`Fprintln`-calls design was reworded to describe the new single-error design.
- `internal/cli/renamed_test.go`: updated `TestQueryStub`/`TestUnlockStub` to assert `cmd.ErrOrStderr()` (`errOut`) is now empty and `err.Error()` equals the two-line text exactly (not a substring match as before). Ran RED-first against the pre-fix `renamed.go` (confirmed `--- FAIL`, both tests failing on the now-empty-expected stderr) before restoring the fix and confirming GREEN. The now-unused `strings` import was removed.
- `test/integration/renamed_stubs_test.go` (new): `TestRenamedStubsPrintExactlyOnce` spawns the real built binary via the package's existing `runBinary` harness (`query main`, `query --json main`, `unlock -p <dir>`, `unlock -p /nonexistent`) and asserts stdout is empty, stderr is exactly the two D-06 lines plus one trailing newline, and the exit code is 1 — the assertion WR-01 identified as missing, since it exercises `main.go`'s print path that `execCmd`-based unit tests structurally cannot reach. Confirmed RED by temporarily stashing the `renamed.go` fix and re-running: failed with the duplicated third line (`codegraph: "query" has been renamed to "search --full"` appended). Restored the fix and confirmed GREEN.

**Gates run before committing (all green):**
- `GOTOOLCHAIN=go1.26.6 go build ./...`
- `GOTOOLCHAIN=go1.26.6 go vet ./internal/cli/ ./test/integration/`
- `gofmt -l internal/cli test/integration` — empty output
- `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1` — full package green, including `TestQueryStub`/`TestUnlockStub` and `TestEveryRegisteredFlagIsAccountedFor` (logged `walked 37 commands (hidden included), inspected 113 flags: 110 accepted via ../../docs/CLI-REFERENCE.md, 3 accepted via testdata/cli-reference-allowlist.txt`)
- `GOTOOLCHAIN=go1.26.6 go test ./test/integration/ -count=1 -run 'TestRenamedStub' -v` — green
- `GOTOOLCHAIN=go1.26.6 task docs:cli:drift` — `docs/CLI-REFERENCE.md byte-identical to a fresh regeneration` (unaffected, as expected — stub is `Hidden`, `Short` unchanged)
- Manual binary check: `GOTOOLCHAIN=go1.26.6 go build -o /tmp/03-fix-bin ./cmd/codegraph && /tmp/03-fix-bin query main; echo exit=$?` — printed exactly the two D-06 lines and `exit=1`; same manual check repeated for `unlock` with identical result. Temp binary removed after verification.

## Out of Scope

### IN-01: `search` has no `Long` description explaining `--full`'s two-line/JSON-envelope behavior

**File:** `internal/cli/search.go:67-70`
**Reason:** Info-severity finding; outside this run's `fix_scope` (`critical_warning`), per the orchestrator's explicit scoping decision. Left untouched. See REVIEW.md's IN-01 section for the finding and suggested fix if a future pass wants to pick it up.

---

_Fixed: 2026-09-16T23:20:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
