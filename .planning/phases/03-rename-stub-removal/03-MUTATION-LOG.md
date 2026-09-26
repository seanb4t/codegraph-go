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

## Regression gates (Task 2)

All run in the main checkout, on HEAD after Task 1's `feat(cli)!:` commit (D-03, VERB-09 SC2).

**`task docs:cli` then `git diff --exit-code -- docs/CLI-REFERENCE.md`:**

```
$ task docs:cli
task: [docs:cli] set -euo pipefail
GOTOOLCHAIN=go1.26.6 go run ./tools/clidoc -out docs/CLI-REFERENCE.md
echo "docs:cli: regenerated docs/CLI-REFERENCE.md from the live Cobra tree via tools/clidoc"

docs:cli: regenerated docs/CLI-REFERENCE.md from the live Cobra tree via tools/clidoc

$ git diff --exit-code -- docs/CLI-REFERENCE.md
$ echo "exit=$?"
exit=0
```

No diff. The generated reference was already correct — hidden commands (including the now-removed
`query`/`unlock`) were already excluded from it before this phase, so removing them changes
nothing in the generated output.

**`task docs:cli:drift`:**

```
$ task docs:cli:drift
task: [docs:cli:drift] set -euo pipefail
scratch=$(mktemp -d)
trap 'rm -rf "${scratch}"' EXIT

files=$(git ls-files -- 'docs/CLI-REFERENCE.md')
ndocs=0
if [ -n "${files}" ]; then
  ndocs=$(printf '%s\n' "${files}" | wc -l | tr -d ' ')
fi
echo "docs:cli:drift: compared ${ndocs} generated file"
if [ "${ndocs}" -lt 1 ]; then
  echo "::error::docs:cli:drift: enumerated only ${ndocs} committed generated file at docs/CLI-REFERENCE.md (expected exactly 1) — a broken enumeration must fail loud, never read as a clean pass"
  exit 1
fi

GOTOOLCHAIN=go1.26.6 go run ./tools/clidoc -out "${scratch}/CLI-REFERENCE.md"

if ! cmp -s docs/CLI-REFERENCE.md "${scratch}/CLI-REFERENCE.md"; then
  echo "::error::docs:cli:drift: docs/CLI-REFERENCE.md differs from a fresh regeneration by the pinned toolchain — run \`task docs:cli\` and commit the result"
  diff -u docs/CLI-REFERENCE.md "${scratch}/CLI-REFERENCE.md" | head -40 || true
  exit 1
fi

echo "docs:cli:drift: docs/CLI-REFERENCE.md byte-identical to a fresh regeneration (temporary file only — source tree untouched)"

docs:cli:drift: compared 1 generated file
docs:cli:drift: docs/CLI-REFERENCE.md byte-identical to a fresh regeneration (temporary file only — source tree untouched)
```

**`GOTOOLCHAIN=go1.26.6 task test:unit` then `GOTOOLCHAIN=go1.26.6 task test:integration`:**

Both exited 0 on the first attempt — no load flake occurred, so no rerun-alone/rerun-whole cycle
was needed. Full package list, `task test:unit`:

```
?   	github.com/seanb4t/codegraph-go/cmd/codegraph	[no test files]
ok  	github.com/seanb4t/codegraph-go/internal/agents	34.398s
ok  	github.com/seanb4t/codegraph-go/internal/bench	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/cli	58.173s
ok  	github.com/seanb4t/codegraph-go/internal/cli/archtest	10.526s
ok  	github.com/seanb4t/codegraph-go/internal/cli/present	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/cli/present/archtest	3.035s
ok  	github.com/seanb4t/codegraph-go/internal/cli/tui	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/corpora	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/fsatomic	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/githooks	19.677s
ok  	github.com/seanb4t/codegraph-go/internal/gitmeta	48.671s
ok  	github.com/seanb4t/codegraph-go/internal/goldenspec	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/graphstore	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/graphstore/archtest	21.457s
ok  	github.com/seanb4t/codegraph-go/internal/indexer	32.638s
ok  	github.com/seanb4t/codegraph-go/internal/indexer/capability	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/indexer/csharpextract	2.178s
ok  	github.com/seanb4t/codegraph-go/internal/indexer/dispatch	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/indexer/goextract	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/indexer/javaextract	2.178s
ok  	github.com/seanb4t/codegraph-go/internal/indexer/mainstream/cextract	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/indexer/mainstream/kotlinextract	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/indexer/mainstream/phpextract	1.757s
ok  	github.com/seanb4t/codegraph-go/internal/indexer/mainstream/rubyextract	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/indexer/mainstream/rustextract	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/indexer/mainstream/swiftextract	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/indexer/nodeid	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/indexer/pyextract	1.726s
ok  	github.com/seanb4t/codegraph-go/internal/indexer/routes	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/indexer/tsextract	2.175s
ok  	github.com/seanb4t/codegraph-go/internal/mcp	24.066s
ok  	github.com/seanb4t/codegraph-go/internal/mcp/archtest	15.954s
ok  	github.com/seanb4t/codegraph-go/internal/nudge	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/parser	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/parser/cgo	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/query	35.955s
ok  	github.com/seanb4t/codegraph-go/internal/query/archtest	0.780s
ok  	github.com/seanb4t/codegraph-go/internal/schema	(cached)
ok  	github.com/seanb4t/codegraph-go/internal/textutil	(cached)
?   	github.com/seanb4t/codegraph-go/internal/uiproto/uiv1	[no test files]
?   	github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect	[no test files]
ok  	github.com/seanb4t/codegraph-go/internal/uiserver	55.289s
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	3.582s
?   	github.com/seanb4t/codegraph-go/internal/version	[no test files]
ok  	github.com/seanb4t/codegraph-go/internal/watch	(cached)
ok  	github.com/seanb4t/codegraph-go/test/integration	36.350s
ok  	github.com/seanb4t/codegraph-go/test/wireoracle	(cached)
?   	github.com/seanb4t/codegraph-go/test/wireoracle/cmd/wireoracle	[no test files]
?   	github.com/seanb4t/codegraph-go/tools/bench/cpudiag	[no test files]
ok  	github.com/seanb4t/codegraph-go/tools/bench/gencorpus	(cached)
ok  	github.com/seanb4t/codegraph-go/tools/bench/publishcheck	(cached)
ok  	github.com/seanb4t/codegraph-go/tools/bench/realcorpus	(cached)
ok  	github.com/seanb4t/codegraph-go/tools/bench/runner	(cached)
?   	github.com/seanb4t/codegraph-go/tools/clidoc	[no test files]
ok  	github.com/seanb4t/codegraph-go/tools/corpora	(cached)
ok  	github.com/seanb4t/codegraph-go/tools/graphcluster	1.550s
ok  	github.com/seanb4t/codegraph-go/tools/mcpaudit	2.118s
ok  	github.com/seanb4t/codegraph-go/tools/transcriptfreeze	(cached)
ok  	github.com/seanb4t/codegraph-go/web	(cached)
```

`task test:integration`:

```
task: [test:integration] go test ./test/integration/...
ok  	github.com/seanb4t/codegraph-go/test/integration	(cached)
```

## Family (a) — VERB-09: the removed-verb guard catches a revived verb

**Test/guard:** `TestRemovedVerbsAreUnknownCommands` (`test/integration/removed_verbs_test.go`),
the spawned-binary guard rewritten in Task 1.

**What and why:** D-01 forbids reviving either removed verb in any form — an alias, a hint, a
hidden command. This family proves the guard actually catches both forms, run in sequence in a
disposable scratch worktree (the main checkout is never mutated).

**Setup:**

```
$ S="$(mktemp -d)/p03-scratch-wt"
$ git worktree add --detach "$S" HEAD
Preparing worktree (detached HEAD 4e8220d4)
HEAD is now at 4e8220d4 docs(03-01): record the RED-first and GREEN transcripts for the removed-verb guard

$ git worktree list
/Volumes/Code/github.com/seanb4t/codegraph-go                        4e8220d4 [gsd/v0.15.0-milestone]
/private/var/.../tmp.8lYZcn1INe/p03-scratch-wt                       4e8220d4 (detached HEAD)
```

No commits were made in the worktree.

### a1 — an alias

**Pre-mutation gate:** `git -C "$S" diff --quiet -- internal/cli/search.go` — clean.

**Mutation applied:** added `Aliases: []string{"query"},` to `newSearchCmd`'s `cobra.Command`
literal, on the line after `Use:`.

**Confirmed applied:** `git -C "$S" diff --numstat -- internal/cli/search.go` → `1	0	internal/cli/search.go`
(exactly 1 insertion); `rg -n -F 'Aliases' internal/cli/search.go` → `70:		Aliases: []string{"query"},`.

**Observed result** (`GOTOOLCHAIN=go1.26.6 go test ./test/integration/ -run '^TestRemovedVerbsAreUnknownCommands$' -count=1 -v`):

```
removed_verbs_test.go:58: [query main]: expected empty stdout, got "main (function) main.go:6\nmain.go (file) main.go:1\n"
    removed_verbs_test.go:58: [query --json main]: expected empty stdout, got "[{\"name\":\"main\",\"kind\":\"function\",\"filePath\":\"main.go\",\"startLine\":6},{\"name\":\"main.go\",\"kind\":\"file\",\"filePath\":\"main.go\",\"startLine\":1}]\n"
    removed_verbs_test.go:74: removed-verb guard: executed 4 of 4 cases
--- FAIL: TestRemovedVerbsAreUnknownCommands (0.85s)
    --- FAIL: TestRemovedVerbsAreUnknownCommands/query_bare (0.09s)
    --- FAIL: TestRemovedVerbsAreUnknownCommands/query_with_--json_flag (0.07s)
    --- PASS: TestRemovedVerbsAreUnknownCommands/unlock_bare (0.01s)
    --- PASS: TestRemovedVerbsAreUnknownCommands/unlock_with_nonexistent_path_arg (0.01s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/integration	7.615s
FAIL
```

The two `query_` subtests fail (the alias routes to `search`, which produces real output instead of
the empty stdout the guard expects); the two `unlock_` subtests pass; the executed 4 of 4 line is
present.

**Revert:** `git -C "$S" checkout -- internal/cli/search.go`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — clean.

### a2 — a hint

**Pre-mutation gate:** `git -C "$S" diff --quiet -- internal/cli/search.go` — clean.

**Mutation applied:** added `SuggestFor: []string{"query"},` at the same place.

**Confirmed applied:** `git -C "$S" diff --numstat -- internal/cli/search.go` → `1	0	internal/cli/search.go`;
`rg -n -F 'SuggestFor' internal/cli/search.go` → `70:		SuggestFor: []string{"query"},`.

**Observed result:**

```
removed_verbs_test.go:61: [query main]: stderr = "unknown command \"query\" for \"codegraph\"\n\nDid you mean this?\n\tsearch\n\n", want exactly "unknown command \"query\" for \"codegraph\"\n" (cobra's own unknown-command line, VERB-09/D-01)
    removed_verbs_test.go:61: [query --json main]: stderr = "unknown command \"query\" for \"codegraph\"\n\nDid you mean this?\n\tsearch\n\n", want exactly "unknown command \"query\" for \"codegraph\"\n" (cobra's own unknown-command line, VERB-09/D-01)
    removed_verbs_test.go:74: removed-verb guard: executed 4 of 4 cases
--- FAIL: TestRemovedVerbsAreUnknownCommands (1.05s)
    --- FAIL: TestRemovedVerbsAreUnknownCommands/query_bare (0.09s)
    --- FAIL: TestRemovedVerbsAreUnknownCommands/query_with_--json_flag (0.06s)
    --- PASS: TestRemovedVerbsAreUnknownCommands/unlock_bare (0.04s)
    --- PASS: TestRemovedVerbsAreUnknownCommands/unlock_with_nonexistent_path_arg (0.05s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/integration	53.023s
FAIL
```

The two `query_` subtests fail because stderr gains cobra's `Did you mean this?` suggestion naming
`search`.

**Revert:** `git -C "$S" checkout -- internal/cli/search.go`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — clean.

**Verdict:** D-01 forbids both forms (an alias and a hint), and the guard rejects both.

## Family (b) — VERB-09: the allowlist's fail-closed grammar catches re-added entries

**Test/guard:** `TestEveryRegisteredFlagIsAccountedFor` (`internal/cli/cli_reference_test.go`),
reading `internal/cli/testdata/cli-reference-allowlist.txt`. This is D-03's built-in second guard.

**Resolved commit:** `T=$(git log --format='%H %s' 84778a829d54fdfa94a69c2da9c5c2774f9acffe..HEAD | rg '^[0-9a-f]{40} test\(03-01\): ' | cut -d' ' -f1)` → `c9df8e626acbfc428795f607bcfe98d0b3635f1d`. At that commit the allowlist still held both `codegraph query`/`codegraph unlock` entries (the stubs had not yet been removed).

**Pre-mutation gate:** `git -C "$S" diff --quiet -- internal/cli/testdata/cli-reference-allowlist.txt` — clean.

**Mutation applied:** `git -C "$S" show "$T":internal/cli/testdata/cli-reference-allowlist.txt > "$S/internal/cli/testdata/cli-reference-allowlist.txt"`.

**Confirmed applied:** `git -C "$S" diff --numstat -- internal/cli/testdata/cli-reference-allowlist.txt` → `2	0	internal/cli/testdata/cli-reference-allowlist.txt` (exactly 2 insertions, 0 deletions).

**Observed result** (`GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -run '^TestEveryRegisteredFlagIsAccountedFor$' -count=1 -v`):

```
cli_reference_test.go:254: walked 37 commands (hidden included), inspected 117 flags: 113 accepted via ../../docs/CLI-REFERENCE.md, 4 accepted via testdata/cli-reference-allowlist.txt
    cli_reference_test.go:274: 2 problem(s):
        stale allowlist entry: codegraph query (matches no registered command or flag)
        stale allowlist entry: codegraph unlock (matches no registered command or flag)
--- FAIL: TestEveryRegisteredFlagIsAccountedFor (0.17s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.765s
FAIL
```

Fails with exactly two `stale allowlist entry:` lines, one for each removed command path.

**Revert:** `git -C "$S" checkout -- internal/cli/testdata/cli-reference-allowlist.txt`.

**Byte-clean proof:** `git -C "$S" diff --quiet` — clean.

## Teardown (Task 2)

```
$ git worktree remove --force "$S"
$ git worktree list
/Volumes/Code/github.com/seanb4t/codegraph-go                        4e8220d4 [gsd/v0.15.0-milestone]
```

(the `p03-scratch-wt` entry is gone.)

```
$ git status --porcelain -- internal cmd test docs tools
```
(empty — the main checkout was never mutated.)
