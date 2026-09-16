---
phase: 03-verb-fold
reviewed: 2026-09-16T22:44:06Z
depth: deep
files_reviewed: 13
files_reviewed_list:
  - internal/cli/search.go
  - internal/cli/renamed.go
  - internal/cli/daemon.go
  - internal/cli/index.go
  - internal/cli/root.go
  - internal/daemon/lock.go
  - internal/cli/query_cli_test.go
  - internal/cli/renamed_test.go
  - internal/cli/daemon_test.go
  - internal/cli/index_lock_test.go
  - internal/cli/notice_test.go
  - internal/cli/testdata/cli-reference-allowlist.txt
  - docs/CLI-REFERENCE.md
findings:
  critical: 0
  warning: 1
  info: 1
  total: 2
status: issues_found
---

# Phase 03: Code Review Report

**Reviewed:** 2026-09-16T22:44:06Z
**Depth:** deep
**Files Reviewed:** 13
**Status:** issues_found

## Summary

Reviewed the Phase 3 "Verb Fold" changes: `search.go`'s merge of the old `query` verb's full-node-record branch behind `--full` (byte-for-byte carried over from the deleted `internal/cli/query.go` at `21bb329e`), the `daemon unlock` move (`newDaemonUnlockCmd` in `daemon.go` is a verbatim copy of the deleted `internal/cli/unlock.go`'s `RunE` body), the two hidden rename stubs in `renamed.go`, `root.go`'s registration list, and the two D-08 message-text updates in `internal/daemon/lock.go` and `internal/cli/index.go`.

Verification performed beyond reading: built the binary (`go build ./cmd/codegraph`), ran it directly to observe real stderr/exit-code behavior for `query`/`unlock`, ran `go vet` and the full `internal/cli`/`internal/daemon` test suites (all green except one unrelated, pre-existing timing-sensitive test — `TestDaemonSharedWriter` in `internal/daemon`, which failed under load on unrelated sync-retry logic untouched by this diff and outside the file scope), diffed every file against `21bb329e` to confirm each moved body is verbatim, grepped the whole tree for lingering `codegraph query`/`codegraph unlock` references (only the allowlist reason strings and this phase's own doc comments remain, as intended), and confirmed `docs/CLI-REFERENCE.md` and the allowlist match the source exactly (no hand-edit drift).

One real, reproducible behavioral gap was found by exercising the compiled binary directly (something `execCmd`-based unit tests structurally cannot catch, since they call `root.Execute()` and never go through `cmd/codegraph/main.go`): the rename stubs' documented "exactly two lines to stderr" (D-06) is not what a real invocation prints — `main.go` appends the returned error as a third line, partially duplicating the first. No blocking defects were found; the fold's logic, flag wiring, message-text updates, and generated-surface changes are all correct against the locked decisions in `03-CONTEXT.md`.

## Warnings

### WR-01: Rename stubs print a redundant third stderr line in the real binary, untested by the unit suite

**File:** `internal/cli/renamed.go:61-65` (query stub), `internal/cli/renamed.go:75-79` (unlock stub); interacts with `cmd/codegraph/main.go:14-17`

**Issue:** D-06 specifies the stub's stderr output is exactly two lines (the rename notice plus the removal-lifetime notice), and `renamed_test.go`'s `wantStderr` asserts exactly those two lines. That assertion is only true through `execCmd`, which calls `root.Execute()` directly and never exercises `cmd/codegraph/main.go`. `main.go` unconditionally prints any non-nil error `cli.Execute()` returns:

```go
func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

Since each stub's `RunE` returns a non-nil `fmt.Errorf` (by design, D-05) *in addition to* writing its own two lines to `cmd.ErrOrStderr()`, the real compiled binary prints three lines, confirmed by building and running it directly:

```
$ codegraph query main
"query" has been renamed to "search --full" — run: codegraph search --full <term>
the "query" stub is removed in the next minor release (v0.15.0)
codegraph: "query" has been renamed to "search --full"
```

The third line largely repeats the first line's content (minus the actionable `run: codegraph search --full <term>` part) and reads as an unintentional echo rather than a deliberate three-line message. Every other command in the tree gets exactly one line of stderr output for a plain returned error (they don't also write their own explanatory text to `cmd.ErrOrStderr()`), so this doubling is specific to the two new stubs' design — the pattern of writing an explanatory message to stderr *and* returning an error whose text also displays is new to this phase and not exercised end-to-end by any test, since `renamed_test.go` and `query_cli_test.go` only assert against `execCmd`'s captured buffers.

**Fix:** Either (a) drop the two explicit `fmt.Fprintln` calls and let the single returned error carry the full guidance text (matching every other command's one-line-per-error convention), or (b) keep the two explicit lines and return a bare/opaque sentinel-style error whose `Error()` text is not printed a second time (e.g. return `errRenamed` and have the stub's own two `Fprintln`s be the sole user-visible text — main.go still exits 1 either way). Whichever is chosen, add a test that exercises `cmd/codegraph/main.go`'s error-printing path (or documents why it's structurally untestable) so the two-line contract in D-06 is actually verified against what a user sees, not just against `root.Execute()`'s return value.

## Info

### IN-01: `search` has no `Long` description explaining `--full`'s two-line/JSON-envelope behavior

**File:** `internal/cli/search.go:67-70`

**Issue:** `newSearchCmd` sets only `Short` (`"Lexically search symbol names/qualified names"`) and no `Long` field. The old `query` command's replacement behavior — `--full`'s two-line human render, the `query.MarshalQueryJSON` envelope shape it selects under `--json --full` versus the plain `Location` array for default `--json` — is documented only in code comments and this phase's `03-CONTEXT.md`, not in anything a user sees via `codegraph search --help` or `docs/CLI-REFERENCE.md`. Comparable commands in the tree (`daemon`, `ui`) carry a multi-line `Long` describing exactly this kind of mode-dependent behavior. This is not a defect in the generated doc (which correctly reflects the empty `Long`) — it's a gap in the generator's input.

**Fix:** Consider adding a short `Long` to `newSearchCmd` describing the `--full` flag's two-line human output and its distinct `--json` envelope shape, so `docs/CLI-REFERENCE.md` and `--help` carry the same level of detail the old `query` verb's doc comment did.

---

_Reviewed: 2026-09-16T22:44:06Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
