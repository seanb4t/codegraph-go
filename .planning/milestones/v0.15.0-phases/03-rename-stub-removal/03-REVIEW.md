---
phase: 03-rename-stub-removal
reviewed: 2026-09-26T21:06:13Z
depth: deep
files_reviewed: 4
files_reviewed_list:
  - .changes/unreleased/Breaking-20260926-165849.599611000.yaml
  - internal/cli/root.go
  - internal/cli/testdata/cli-reference-allowlist.txt
  - test/integration/removed_verbs_test.go
findings:
  critical: 0
  warning: 0
  info: 1
  total: 1
status: clean
---

# Phase 03: Code Review Report

**Reviewed:** 2026-09-26T21:06:13Z
**Depth:** deep
**Files Reviewed:** 4
**Status:** clean

## Summary

This phase removes the one-release `query`/`unlock` rename stubs (VERB-09): `internal/cli/renamed.go` and `internal/cli/renamed_test.go` are deleted outright, `root.go`'s `AddCommand` list and doc comments are updated to stop referencing `newQueryCmd()`/`newUnlockCmd()`, the two now-orphaned entries are dropped from `testdata/cli-reference-allowlist.txt`, and `test/integration/renamed_stubs_test.go` is rewritten (via `git mv`, confirmed in `git log`) into `removed_verbs_test.go`, which asserts the real spawned binary now answers `query`/`unlock` with cobra's stock unknown-command error and exit code 1.

Verification performed beyond static reading:
- Built the binary and ran `query`, `query --json main`, `unlock -p <dir>`, `unlock -p /nonexistent` directly — all four produced exactly `unknown command "<verb>" for "codegraph"` on stderr, empty stdout, and exit 1, with no suggestion/hint text (cobra's `legacyArgs`/`stripFlags` mechanics confirmed empirically, not just by reading the test).
- Ran `TestRemovedVerbsAreUnknownCommands`, `TestEveryRegisteredFlagIsAccountedFor`, and `TestEveryCommandHasGroupID` — all pass.
- `go build ./...`, `go vet ./...`, and `gofmt -l` on the two changed `.go` files are all clean.
- Grepped the full tree (excluding `.planning/`, `CHANGELOG.md`, `.changes/`) for `newQueryCmd`, `newUnlockCmd`, `codegraph query`, `codegraph unlock`, `SuggestFor`, and `Aliases:` — no dangling references, no revival path for either name (satisfies the D-01 "no alias, no SuggestFor, no hidden command, no hint text" invariant named in the test's own doc comment).
- Cross-checked the changelog fragment's replacement guidance (`search --full <term>`, `daemon unlock [path]`) against the actual flag/usage strings in `search.go` and `daemon.go` — both match exactly.
- Confirmed via `git log --diff-filter=A` that `Breaking-20260926-165849.599611000.yaml`'s filename matches `.changie.yaml`'s `fragmentFileFormat` template and its `time:` field, and that it is the only fragment of its kind (no duplicate/conflicting Breaking entry for this change).

No incorrect behavior, security issue, or maintainability defect was found. One minor test-duplication note is recorded below at Info severity.

## Info

### IN-01: Redundant `verb` field duplicates `args[0]` in the test table

**File:** `test/integration/removed_verbs_test.go:34-43`
**Issue:** Each table case carries both `args` (whose first element is always the verb being tested) and a separate `verb` field used only to build `wantStderr`:
```go
{"query bare", []string{"query", "main"}, "query"},
{"query with --json flag", []string{"query", "--json", "main"}, "query"},
{"unlock bare", []string{"unlock", "-p", dir}, "unlock"},
{"unlock with nonexistent path arg", []string{"unlock", "-p", "/nonexistent"}, "unlock"},
```
If a future edit changes `args[0]` (e.g. adding a case for a third removed verb) without updating the trailing `verb` field, the test would silently assert against the wrong expected error text instead of failing loudly or catching a real regression — undermining the guard's own stated purpose (a guard that asserts the wrong thing is worse than one that doesn't run).
**Fix:** Derive the expected verb from `tc.args[0]` directly and drop the separate field:
```go
cases := []struct {
	name string
	args []string
}{
	{"query bare", []string{"query", "main"}},
	{"query with --json flag", []string{"query", "--json", "main"}},
	{"unlock bare", []string{"unlock", "-p", dir}},
	{"unlock with nonexistent path arg", []string{"unlock", "-p", "/nonexistent"}},
}
...
wantStderr := fmt.Sprintf("unknown command %q for %q\n", tc.args[0], "codegraph")
```

---

_Reviewed: 2026-09-26T21:06:13Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
