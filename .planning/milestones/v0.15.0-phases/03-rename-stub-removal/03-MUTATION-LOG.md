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

## Family (c) — VERB-09: positive-controlled census BEFORE the removal

**Test/guard:** the v0.14.0 census instrument, reused byte for byte (D-06):

```
rg -nU -w --hidden 'codegraph\s+(query|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .
```

**What are we testing, and why?** Whether the census can still find a planted reference in a
hidden directory before its "4 hits" baseline is trusted, run on the tree at the `test(03-01)`
commit — the guard is already rewritten there, but the stubs and their allowlist entries still
exist. This proves the rewritten guard's own text does not itself trip the census (D-06's
consequence for new text), and pins the exact "before" count this phase's removal must zero out.

**Setup:** `T=c9df8e626acbfc428795f607bcfe98d0b3635f1d` (the `test(03-01):` commit from Task 1).
`C="$(mktemp -d)/p03-census-wt"`, `git worktree add --detach "$C" "$T"`.

```
$ git worktree add --detach "$C" "$T"
Preparing worktree (detached HEAD c9df8e62)
HEAD is now at c9df8e62 test(03-01): expect the removed query and unlock verbs to be unknown commands (VERB-09)
```

**Pre-mutation gate:** `git -C "$C" status --porcelain .github` — empty; the control does not exist.

**Mutation applied:** the untracked `.github/__census_control__.md`, three lines, inside a
subshell with `trap 'rm -f .github/__census_control__.md' EXIT`, never `git add`ed:

```
run codegraph unlock now
and also run codegraph
query now
```

**Observed result** (verbatim, planted-tree run from `$C`):

```
./.github/__census_control__.md:1:run codegraph unlock now
./.github/__census_control__.md:2:and also run codegraph
./.github/__census_control__.md:3:query now
./internal/cli/renamed.go:10:// this phase folds away (Phase 3 VERB-03/VERB-04): `codegraph query <term>`
./internal/cli/renamed.go:11:// into `codegraph search --full <term>`, and `codegraph unlock [path]` into
./internal/cli/testdata/cli-reference-allowlist.txt:8:codegraph query	hidden rename stub for `search --full` (VERB-03); removed in v0.15.0
./internal/cli/testdata/cli-reference-allowlist.txt:9:codegraph unlock	hidden rename stub for `daemon unlock` (VERB-04); removed in v0.15.0
exit: 0
```

The control was found on `:1:` and `:3:` exactly as required.

**Revert:** through the trap (`rm -f .github/__census_control__.md`).

**Byte-clean proof:** `git -C "$C" status --porcelain .github` — empty; the control is absent.

**Real census** (verbatim, clean-tree run from `$C`):

```
./internal/cli/testdata/cli-reference-allowlist.txt:8:codegraph query	hidden rename stub for `search --full` (VERB-03); removed in v0.15.0
./internal/cli/testdata/cli-reference-allowlist.txt:9:codegraph unlock	hidden rename stub for `daemon unlock` (VERB-04); removed in v0.15.0
./internal/cli/renamed.go:10:// this phase folds away (Phase 3 VERB-03/VERB-04): `codegraph query <term>`
./internal/cli/renamed.go:11:// into `codegraph search --full <term>`, and `codegraph unlock [path]` into
```

`before: 4 lines across 2 files`. No line from `test/integration` appears — the rewritten guard's
own text obeys D-06.

**Teardown:** `git worktree remove --force "$C"`.

## Family (d) — VERB-09: positive-controlled census AFTER the removal

**Test/guard:** the identical census instrument, byte for byte, run in the main checkout at HEAD
(after both Task 1 commits and Task 2's regression-gate work):

```
rg -nU -w --hidden 'codegraph\s+(query|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .
```

**Pre-mutation gate:** `git status --porcelain .github` — empty; the control does not exist.

**Mutation applied:** the same untracked `.github/__census_control__.md`, three lines, inside the
same trapped subshell, never `git add`ed.

**Observed result** (verbatim):

```
./.github/__census_control__.md:1:run codegraph unlock now
./.github/__census_control__.md:2:and also run codegraph
./.github/__census_control__.md:3:query now
exit: 0
```

Only the control lines are reported, on `:1:` and `:3:` — no other line.

**Revert:** through the trap.

**Byte-clean proof:** `git status --porcelain .github` — empty; the control is absent.

**Real census** (verbatim, clean-tree run at HEAD):

```
(no output)
exit: 1
```

`after: 0 lines across 0 files`. `rg` exits 1 (no match), recorded with its exit status. At this
point `.changes/unreleased/` holds only `.gitkeep` — the capability fragment, and the one expected
D-07 hit, only appear at the phase close.

## Commit-history audit (Task 3)

```
$ git log --format=%s 84778a829d54fdfa94a69c2da9c5c2774f9acffe..HEAD
docs(03-01): record regression gates and the guard mutation families (VERB-09)
docs(03-01): record the RED-first and GREEN transcripts for the removed-verb guard
feat(cli)!: remove the query and unlock rename stubs
test(03-01): expect the removed query and unlock verbs to be unknown commands (VERB-09)
docs(03): begin phase execution
docs(03): sync state.json with planned phase
docs(03): create phase plan
docs(03): create phase plan
```

```
$ F=90dc419a1778dc8ebea77d5de8b5acd8ec6a8e17
$ git log -1 --format=%B "$F"
feat(cli)!: remove the query and unlock rename stubs

v0.14.0 kept hidden query and unlock stubs for one minor release after
folding them into search --full and daemon unlock. This commit removes
the stubs, their unit tests, their registration and their two allowlist
entries, and either name now fails with cobra's own unknown-command
error and exits 1.

BREAKING CHANGE: the query and unlock commands are removed and are now unknown commands (exit 1); use search --full <term> instead of query <term>, and daemon unlock [path] instead of unlock [path].
Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>
Claude-Session: https://claude.ai/code/session_01VJZpxvJro4s7kAk5tQeuP8
```

**commit audit:** every subject in the range matches `pr-title.yml`'s
`^(feat|fix|perf|refactor|docs|chore|ci|test|build|revert)(\([a-z0-9_-]+\))?!?: .+` grammar; there
is exactly one `feat(` subject and exactly one `!:`, both on the same commit (`90dc419a`); its
`BREAKING CHANGE:` footer names `search --full` and `daemon unlock`; the `test(03-01):` commit
(`c9df8e62`) precedes it; no `ci skip`/`skip ci` marker appears anywhere in the range.

## Verdict (Task 3)

Before: 4 lines across 2 files. After: 0. The instrument is byte-identical between Families (c)
and (d), and identical to the v0.14.0 copy in
`.planning/milestones/v0.14.0-phases/03-verb-fold/03-MUTATION-LOG.md`, so the two counts answer
the same question. Every before hit belonged to the removed stub's own declaration
(`internal/cli/renamed.go:10-11`) or its two allowlist entries
(`internal/cli/testdata/cli-reference-allowlist.txt:8-9`) — both now deleted. At the phase close, a
hit confined to the capability-written `.changes/unreleased/` fragment is expected (D-07); the
verifier records it and the instrument is not changed.
