# 03-MUTATION-LOG — Verb Fold

**Phase:** 03-verb-fold
**Date:** 2026-09-16
**Scope:** Three demonstration families across this phase's verb-fold work — (a) VERB-05: a
positive-controlled census of old-verb references (`codegraph query` / `codegraph unlock`),
run BEFORE `internal/cli/` is edited, appended by plan 03-01; (b) VERB-07: a RED demonstration
against a re-visible `query` command (or a missing `--full` flag), showing the generated-docs
drift gate and the flag-accounting test both catch it, appended by plan 03-03; (c) VERB-05 again:
the identical census invocation re-run AFTER the fold, proving zero old-verb references remain
outside the two rename stubs, appended by plan 03-04.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` (or, for
an untracked scratch file, `git status --porcelain <dir>`) is asserted to exit empty/clean. This
proves no pre-existing tracked edit was overwritten by the mutation, and no revert was a
destructive blind checkout of someone else's in-flight work. Every family entry below records this
gate's result at the point it was checked.

---

## Family (a) — VERB-05: positive-controlled census BEFORE the fold

**Test/guard:** the census instrument (used verbatim everywhere in this phase — 03-04 re-runs it
unchanged):

```
rg -nU -w --hidden 'codegraph\s+(query|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .
```

`-w` gives the word boundary D-10 requires (so `internal/query`, a JSON `"query"` key,
`query.json` and `codegraph_search` are never hits — the literal `codegraph` immediately before
the verb is what makes the match); `-U` plus `\s+` is the multiline half (a sentence wrapped
between `codegraph` and the verb is still found); `--hidden` corrects RESEARCH's original
invocation, which lacked it and would silently have skipped `.github/`, `.claude/hooks/` and
`.claude/skills/codegraph/SKILL.md` — three of the paths D-10 explicitly names.

**What are we testing, and why?** Whether the census instrument can find an old-verb reference
planted in a hidden directory, in both single-line and line-wrapped form, before its later "zero
hits" claim (03-04) is trusted — a transcript grep is a claim about the grep, not the product
(standing repo rule; Phase 2 D-14). We are also confirming the 14-line/6-file "before" baseline
RESEARCH already enumerated is exactly what this session's tree produces, so 03-02's fold knows
every consumer it has to retarget.

**Pre-mutation gate:** `git status --porcelain` — clean except pre-existing unrelated
modifications to `.planning/STATE.md` and `.planning/state.json` (orchestrator-owned, outside this
plan's scope); no entry under `.github/`, and no file named `__census_control__.md` anywhere on
disk.

**Mutation applied:** created the UNTRACKED file `.github/__census_control__.md` with exactly
three lines:

```
run codegraph unlock now
and also run codegraph
query now
```

Line 1 is the single-line form; lines 2–3 are the line-wrapped form (`codegraph` at end of line 2,
`query` on line 3). Never `git add`ed. The step ran inside `trap 'rm -f .github/__census_control__.md' EXIT` so the control could not outlive the shell step.

**Observed result** (verbatim, the planted-tree run):

```
./.github/__census_control__.md:1:run codegraph unlock now
./.github/__census_control__.md:2:and also run codegraph
./.github/__census_control__.md:3:query now
./docs/CLI-REFERENCE.md:37:* [codegraph query](#codegraph-query)	 - Search full node records by name/qualifiedName
./docs/CLI-REFERENCE.md:46:* [codegraph unlock](#codegraph-unlock)	 - Clear a stale daemon lock
./docs/CLI-REFERENCE.md:604:## codegraph query
./docs/CLI-REFERENCE.md:609:codegraph query <term> [flags]
./docs/CLI-REFERENCE.md:853:## codegraph unlock
./docs/CLI-REFERENCE.md:858:codegraph unlock [path] [flags]
./internal/daemon/lock.go:208:		return fmt.Errorf("%w: pid=%d — stop it first, or run `codegraph unlock` once it has exited", ErrLockLive, info.PID)
./internal/daemon/lock.go:274:// Unlock implements `codegraph unlock`'s engine (SYNC-05): it removes the
./internal/cli/index_lock_test.go:123:	if !strings.Contains(err.Error(), "codegraph unlock") {
./internal/cli/index_lock_test.go:124:		t.Fatalf("index --force error = %q; want it to name `codegraph unlock` (the verb live today, not `daemon unlock`)", err.Error())
./internal/cli/unlock.go:12:// newUnlockCmd builds the `codegraph unlock` command (D-05, SYNC-05):
./internal/cli/index.go:126:				// remedies. `codegraph unlock` is the verb live today;
./internal/cli/index.go:128:				return fmt.Errorf("%w: another process holds %s open — stop it first with `codegraph daemon stop`, or run `codegraph unlock` once it has exited", priorErr, storeDir)
./internal/cli/query.go:35:// newQueryCmd builds `codegraph query <term>` (QRY-01): full node records
exit=0
```

The control was found on `:1:`, `:2:` and `:3:` exactly as required — the wrapped match spans
lines 2–3 (`codegraph` at end of line 2, `query` matched on line 3). Had the control not been
reported, the plan would have stopped here; it was reported, so the real census below is trusted.

**Revert:** `rm -f .github/__census_control__.md`.

**Byte-clean proof:** `git status --porcelain .github` — empty; `test ! -e
.github/__census_control__.md` — true. The control was never tracked or staged.

**Real census** (verbatim, the clean-tree run — the identical instrument, no control present):

```
./docs/CLI-REFERENCE.md:37:* [codegraph query](#codegraph-query)	 - Search full node records by name/qualifiedName
./docs/CLI-REFERENCE.md:46:* [codegraph unlock](#codegraph-unlock)	 - Clear a stale daemon lock
./docs/CLI-REFERENCE.md:604:## codegraph query
./docs/CLI-REFERENCE.md:609:codegraph query <term> [flags]
./docs/CLI-REFERENCE.md:853:## codegraph unlock
./docs/CLI-REFERENCE.md:858:codegraph unlock [path] [flags]
./internal/daemon/lock.go:208:		return fmt.Errorf("%w: pid=%d — stop it first, or run `codegraph unlock` once it has exited", ErrLockLive, info.PID)
./internal/daemon/lock.go:274:// Unlock implements `codegraph unlock`'s engine (SYNC-05): it removes the
./internal/cli/index_lock_test.go:123:	if !strings.Contains(err.Error(), "codegraph unlock") {
./internal/cli/index_lock_test.go:124:		t.Fatalf("index --force error = %q; want it to name `codegraph unlock` (the verb live today, not `daemon unlock`)", err.Error())
./internal/cli/unlock.go:12:// newUnlockCmd builds the `codegraph unlock` command (D-05, SYNC-05):
./internal/cli/index.go:126:				// remedies. `codegraph unlock` is the verb live today;
./internal/cli/index.go:128:				return fmt.Errorf("%w: another process holds %s open — stop it first with `codegraph daemon stop`, or run `codegraph unlock` once it has exited", priorErr, storeDir)
./internal/cli/query.go:35:// newQueryCmd builds `codegraph query <term>` (QRY-01): full node records
```

`before: 14 lines across 6 files`

Sorted file set: `./docs/CLI-REFERENCE.md ./internal/cli/index.go
./internal/cli/index_lock_test.go ./internal/cli/query.go ./internal/cli/unlock.go
./internal/daemon/lock.go` — exactly the six files RESEARCH enumerated, no hit under
`internal/query/`, `.github/` or `.claude/`.

**Verdict:** The census instrument found the planted control (single-line and wrapped forms,
under a hidden directory) before the real census result was trusted. The `--hidden` correction to
RESEARCH's original invocation changed nothing on today's tree — the real count is still 14 lines
across 6 files, identical to RESEARCH's baseline — but the correction was required for D-10's
scope (`.github/`, `.claude/hooks/`, `.claude/skills/codegraph/SKILL.md`) to be genuinely covered
rather than assumed: without `--hidden`, the planted control under `.github/` would have been
invisible to the same invocation, and RESEARCH's own instrument would have silently missed any
real consumer living in those three paths.

---

## Baseline (pre-fold, captured by 03-01)

Every "after" comparison the later plans make — VERB-01 byte-identity, VERB-05 zero-references,
VERB-06 completions/man, VERB-07 MCP set and wire oracle — is checked against the recorded values
below.

### 1. Pre-fold HEAD

```
$ git rev-parse HEAD
21bb329ef4322c6218ac6afc86b7d244fe6b9ad2
```

`pre-fold HEAD: 21bb329ef4322c6218ac6afc86b7d244fe6b9ad2`

(03-02 Task 3 also derives the pre-fold tree as the `feat!:` commit's parent; this SHA is the
human-readable cross-check.)

### 2. Default `search` output digests (VERB-01 byte-identity)

Binary built at the pre-fold HEAD: `GOTOOLCHAIN=go1.26.6 go build -o /tmp/03-01-prefold-bin ./cmd/codegraph`.

**On the repo's own index** (`-p /Volumes/Code/github.com/seanb4t/codegraph-go`), every hit lives
in `internal/query/` (a directory this phase never touches). Outputs and digests captured
verbatim this session:

| Invocation | sha256(stdout) | Verbatim stdout |
|---|---|---|
| `search matchNodes` | `322f8245496390fe7ef29d52b8b9989ac1e8e8656fb30b116e007ab23e5569b2` | `matchNodes (method) internal/query/search.go:74` |
| `search matchNodes --json` | `8fda99aaa8006a217d23fb4d830fadff1070a86a4f060a288c9cfd981ba64bac` | `[{"name":"matchNodes","kind":"method","filePath":"internal/query/search.go","startLine":74}]` |
| `search lexicalTier --limit 2` | `4a704caea41311038586efd5f08d0349ed9db6c8f63e4fe7f2691fcffcaa70e3` | `lexicalTier (type_alias) internal/query/search.go:26`<br>`lexicalTierExact (constant) internal/query/search.go:29` |
| `search ValidateKind --kind function` | `a08dd35c45bb7d0e9c62aa53e2252446ae9c7ed3a016c8c918507d93626af7d3` | `ValidateKind (function) internal/query/validate.go:209`<br>`TestValidateKind (function) internal/query/engine_test.go:246` |
| `search ValidateKind --json --limit 5` | `b465074721b1a654533537130c9ceac482f259b75f4e3e11a605b815634499f7` | `[{"name":"ValidateKind","kind":"function","filePath":"internal/query/validate.go","startLine":209},{"name":"TestValidateKind","kind":"function","filePath":"internal/query/engine_test.go","startLine":246}]` |

Digest process: run each invocation, pipe stdout to `shasum -a 256`, record the hex digest
verbatim. 03-02's byte-identity proof re-runs the identical invocations against the post-fold
binary and asserts the digests match these.

**On a fresh gofixture** (`cp -R internal/indexer/testdata/gofixture/. <tmp>/ && /tmp/03-01-prefold-bin init <tmp>` — indexed `files=4 nodes=20 edges=22`), full outputs and digests captured verbatim this session:

```
$ /tmp/03-01-prefold-bin search Alpha -p <tmp>
Alpha (function) pkga/pkga.go:13
sha256: bf20b7bacaa9af869653dcea17a8a69415938facaaf4751ddb4d6da6907177a4

$ /tmp/03-01-prefold-bin search Alpha --json -p <tmp>
[{"name":"Alpha","kind":"function","filePath":"pkga/pkga.go","startLine":13}]
sha256: 67c72168dc9ffede4652aac2005a6a1b859d7ba119af0aa3d59544a5ef84e50c

$ /tmp/03-01-prefold-bin search pkga -p <tmp>
pkga (package) :0
pkga/embed.go (file) pkga/embed.go:1
pkga/pkga.go (file) pkga/pkga.go:1
sha256: 96a5106403964172aadc9f38be3c84171c083b2537c00f9880c3580ad71bc652

$ /tmp/03-01-prefold-bin search helper --kind function -p <tmp>
helper (function) pkga/pkga.go:18
sha256: b1eae654b88ecbd6a6cee3f4ec98fb48ea2cdf29b73b695512f6bbd567f7f5e2
```

(Correction on the record: an initial loop-based capture attempt produced two spuriously
identical digests for `Alpha --json` and `helper --kind function` due to a shell quoting bug in
the loop, not a property of the binary; both invocations were re-run individually outside the
loop and produced the distinct digests recorded above, confirmed by `diff` on the raw outputs.)

### 3. Completion listings (VERB-06)

```
$ /tmp/03-01-prefold-bin __complete ""
affected	List test symbols impacted by changes to the given files
callees	List a symbol's forward call targets
callers	List a symbol's reverse callers
completion	Generate the autocompletion script for the specified shell
daemon	List and manage running codegraph daemons
explore	Explore relevant symbols: verbatim source, call paths, blast radius
files	Browse the indexed file structure
githooks	Manage git sync hooks (post-commit/post-merge/post-checkout)
help	Help about any command
impact	Depth-bounded reverse blast radius of a symbol
index	Deterministically rebuild the graph from scratch
init	Create .codegraph/ and build the full graph in one step
install	Configure coding agents to use this codegraph binary as their MCP server
node	Show a symbol's signature, calls, and callers, or a line-numbered file read
query	Search full node records by name/qualifiedName
search	Lexically search symbol names/qualified names (locations only)
serve	Run the codegraph MCP server
status	Report index health and counts
sync	Incrementally update the graph from changed files
telemetry	Print this build's telemetry/network-behavior statement
ui	Run a local, read-only web UI over the repository's own index
uninit	Remove .codegraph/
uninstall	Remove codegraph's configuration from coding agents
unlock	Clear a stale daemon lock
upgrade	Download, verify, and install a new codegraph release
version	Print build version information
:4
Completion ended with directive: ShellCompDirectiveNoFileComp
```

26 entries (excluding the trailing `:4` directive line). `query` and `unlock` are present; the
hidden `man` command is absent — the existing proof that `Hidden: true` keeps a command out of
completions.

```
$ /tmp/03-01-prefold-bin __complete daemon ""
start	Run the shared watch/index server in the foreground
stop	Stop the current-repo daemon, or every running daemon (--all)
:4
Completion ended with directive: ShellCompDirectiveNoFileComp
```

2 entries: `start`, `stop`.

### 4. Man page listing (VERB-06)

```
$ /tmp/03-01-prefold-bin man <tmpdir> && ls <tmpdir> | sort
codegraph-affected.1
codegraph-callees.1
codegraph-callers.1
codegraph-daemon-start.1
codegraph-daemon-stop.1
codegraph-daemon.1
codegraph-explore.1
codegraph-files.1
codegraph-githooks-install.1
codegraph-githooks-remove.1
codegraph-githooks-status.1
codegraph-githooks.1
codegraph-impact.1
codegraph-index.1
codegraph-init.1
codegraph-install.1
codegraph-node.1
codegraph-query.1
codegraph-search.1
codegraph-serve.1
codegraph-status.1
codegraph-sync.1
codegraph-telemetry.1
codegraph-ui.1
codegraph-uninit.1
codegraph-uninstall.1
codegraph-unlock.1
codegraph-upgrade.1
codegraph-version.1
codegraph.1
```

30 pages (`ls | wc -l` = 30). `codegraph-query.1` and `codegraph-unlock.1` are present;
`codegraph-man.1` is absent (man pages are never generated for the `man` command itself);
`codegraph-daemon-unlock.1` is absent (the `daemon unlock` subcommand does not exist until
03-02).

### 5. MCP tool names (VERB-07)

```
$ rg -o 'Name:\s+"codegraph_[a-z]+"' internal/mcp/tools.go | rg -o 'codegraph_[a-z]+' | sort -u
codegraph_callees
codegraph_callers
codegraph_explore
codegraph_files
codegraph_impact
codegraph_node
codegraph_search
codegraph_status
```

Exactly the 8 tool names D-14 names. This set is frozen — no plan in this phase changes it.

### 6. Wire oracle (VERB-07)

```
$ GOTOOLCHAIN=go1.26.6 go test ./test/wireoracle/... -count=1
ok  	github.com/seanb4t/codegraph-go/test/wireoracle	50.404s
?   	github.com/seanb4t/codegraph-go/test/wireoracle/cmd/wireoracle	[no test files]
```

`git status --porcelain testdata/wireoracle` — empty. The 50.404s wall time (well above any
cache-hit floor) confirms this is a real, uncached run (`-count=1`).

### Known, unrelated flake

`internal/daemon`'s `TestDaemonSharedWriter` timed out under full parallel `go test ./...` load
during this planning/execution session and passed in isolation in 0.23s — a pre-existing load
flake, already recorded in STATE.md's deferred_items, unrelated to the verb fold. If it recurs
during 03-02's or later plans' full-suite runs, it should be reported separately rather than
treated as a fold regression.
