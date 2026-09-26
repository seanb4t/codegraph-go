# 03-MUTATION-LOG — Rename-Stub Removal

**Phase:** 03-rename-stub-removal
**Date:** 2026-09-26
**Scope:** Evidence for VERB-09's removal of the hidden `query` and `unlock` rename stubs,
organized in the sections below:
- the RED-first and GREEN transcripts (Task 1);
- the regression gates and Families (a) and (b): the guard against a revived verb, and the
  allowlist against re-added entries (Task 2);
- Families (c) and (d): the census before and after the removal, plus the commit-history audit
  (Task 3).

All mutations happened in disposable, detached scratch git worktrees — never the main working
tree — created and removed within their own task's execution.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` (or, for
the directory-scoped checks, `git status --porcelain -- <dir>`) is asserted to exit empty/clean.
This proves no pre-existing tracked edit was overwritten by the mutation, and no revert was a
destructive blind checkout of someone else's in-flight work. Every family entry below records this
gate's result at the point it was checked.

## RED-first transcript (Task 1)

Command: `GOTOOLCHAIN=go1.26.6 go test ./test/integration/ -run '^TestRemovedVerbsAreUnknownCommands$' -count=1 -v 2>&1`

Run against the tree with `test/integration/removed_verbs_test.go` rewritten as the removal's
guard, while `internal/cli/renamed.go` still registers the stubs (before the `feat(cli)!:`
commit). All four subtests fail on the stderr assertion, with the stub's rename text as the got
value — an assertion failure, not a build failure. The executed-count line still reads 4 of 4.

```
    removed_verbs_test.go:61: [query main]: stderr = "\"query\" has been renamed to \"search --full\" — run: codegraph search --full <term>\nthe \"query\" stub is removed in the next minor release (v0.15.0)\n", want exactly "unknown command \"query\" for \"codegraph\"\n" (cobra's own unknown-command line, VERB-09/D-01)
    removed_verbs_test.go:61: [query --json main]: stderr = "\"query\" has been renamed to \"search --full\" — run: codegraph search --full <term>\nthe \"query\" stub is removed in the next minor release (v0.15.0)\n", want exactly "unknown command \"query\" for \"codegraph\"\n" (cobra's own unknown-command line, VERB-09/D-01)
    removed_verbs_test.go:61: [unlock -p /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/TestRemovedVerbsAreUnknownCommands3880884257/001]: stderr = "\"unlock\" has been renamed to \"daemon unlock\" — run: codegraph daemon unlock [path]\nthe \"unlock\" stub is removed in the next minor release (v0.15.0)\n", want exactly "unknown command \"unlock\" for \"codegraph\"\n" (cobra's own unknown-command line, VERB-09/D-01)
    removed_verbs_test.go:61: [unlock -p /nonexistent]: stderr = "\"unlock\" has been renamed to \"daemon unlock\" — run: codegraph daemon unlock [path]\nthe \"unlock\" stub is removed in the next minor release (v0.15.0)\n", want exactly "unknown command \"unlock\" for \"codegraph\"\n" (cobra's own unknown-command line, VERB-09/D-01)
    removed_verbs_test.go:74: removed-verb guard: executed 4 of 4 cases
--- FAIL: TestRemovedVerbsAreUnknownCommands (0.64s)
    --- FAIL: TestRemovedVerbsAreUnknownCommands/query_bare (0.01s)
    --- FAIL: TestRemovedVerbsAreUnknownCommands/query_with_--json_flag (0.01s)
    --- FAIL: TestRemovedVerbsAreUnknownCommands/unlock_bare (0.01s)
    --- FAIL: TestRemovedVerbsAreUnknownCommands/unlock_with_nonexistent_path_arg (0.01s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/integration	5.668s
FAIL
```

`gsd_run check tdd-red-evidence` is not used to certify this RED: it is TAP-only and returns
`INVALID_RED` for every Go test (rule x1cjy9vyhq, 03-RESEARCH.md Pitfall 1). This pasted `--- FAIL:`
transcript, showing the target test failing on an assertion for the planned behavior, is the RED
evidence.

## GREEN after the removal (Task 1)

After the `feat(cli)!:` commit deletes `internal/cli/renamed.go` and `internal/cli/renamed_test.go`,
drops `newQueryCmd()`/`newUnlockCmd()` from `root.go`'s `AddCommand` list, updates the stale
comments, and removes the two allowlist entries, the same guard command passes 4 of 4:

Command: `GOTOOLCHAIN=go1.26.6 go test ./test/integration/ -run '^TestRemovedVerbsAreUnknownCommands$' -count=1 -v 2>&1`

```
removed_verbs_test.go:74: removed-verb guard: executed 4 of 4 cases
--- PASS: TestRemovedVerbsAreUnknownCommands (0.67s)
    --- PASS: TestRemovedVerbsAreUnknownCommands/query_bare (0.02s)
    --- PASS: TestRemovedVerbsAreUnknownCommands/query_with_--json_flag (0.01s)
    --- PASS: TestRemovedVerbsAreUnknownCommands/unlock_bare (0.01s)
    --- PASS: TestRemovedVerbsAreUnknownCommands/unlock_with_nonexistent_path_arg (0.01s)
ok  	github.com/seanb4t/codegraph-go/test/integration	4.679s
```

The allowlist and flag-accounting guards are also green, with their `walked N commands … inspected
M flags` line recorded as observed (not asserted from memory):

Command: `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -run '^(TestEveryRegisteredFlagIsAccountedFor|TestEveryCommandHasGroupID)$' -count=1 -v`

```
cli_reference_test.go:254: walked 37 commands (hidden included), inspected 117 flags: 113 accepted via ../../docs/CLI-REFERENCE.md, 4 accepted via testdata/cli-reference-allowlist.txt
--- PASS: TestEveryRegisteredFlagIsAccountedFor (0.02s)
    cli_reference_test.go:378: inspected 24 visible commands across 4 groups
--- PASS: TestEveryCommandHasGroupID (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/cli	0.462s
```

`gofmt -l internal/cli test/integration` printed nothing, `GOTOOLCHAIN=go1.26.6 go build ./...` and
`GOTOOLCHAIN=go1.26.6 go vet ./internal/cli/ ./test/integration/` were clean, and the allowlist now
holds its original 6 header lines plus exactly 3 entries (`man`, `hook`, `hook pretooluse`). The
floor constants `cliReferenceMinCommands`, `cliReferenceMinFlags` and `cliReferenceMinVisibleCommands`
in `internal/cli/cli_reference_test.go` were not touched, and the 37/117/24 counts clear all three
floors comfortably.
