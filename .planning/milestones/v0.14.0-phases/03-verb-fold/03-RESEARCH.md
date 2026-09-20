# Phase 3: Verb Fold - Research

**Researched:** 2026-09-16
**Domain:** Cobra CLI command surface refactor (merge, move, deprecate) in a Go monorepo with a generated-docs drift gate and a golden/wire-oracle test suite
**Confidence:** HIGH

## Summary

This phase is a pure `internal/cli` refactor with three independent moving parts, all already fully scoped by `03-CONTEXT.md`'s locked decisions: (1) merge `query.go`'s RunE body into `search.go` behind a new `--full` bool, reconciling two independently-evolved flag registrations and two different JSON shapes; (2) move `unlock.go`'s body verbatim under `daemon.go` as `daemon unlock`, updating two live error-message call sites that name the old verb; (3) register `query` and `unlock` as `Hidden: true` stub commands that print a rename notice to stderr and exit non-zero. Every piece of this was verified directly against the live source this session — `internal/cli/query.go`, `search.go`, `unlock.go`, `daemon.go`, `root.go`, `man.go`, `internal/query/search.go`, `internal/daemon/lock.go`, `internal/cli/index.go`, `cli_reference_test.go`, `query_cli_test.go`, `notice_test.go`, `tools/clidoc/main.go`, `Taskfile.yml`'s `docs:cli`/`docs:cli:drift` targets, `release-please-config.json`, and `.github/workflows/pr-title.yml`.

The whole-repo word-boundary, positive-controlled census (VERB-05, D-10) was run this session and found **exactly** the six-category baseline `03-CONTEXT.md` predicted: `docs/CLI-REFERENCE.md` (6 generated hits), `internal/daemon/lock.go` (2), `internal/cli/index.go` (2), `internal/cli/index_lock_test.go` (2), and one comment/doc-comment hit each in `query.go` and `unlock.go` — zero hits anywhere else in the repo (README, `docs/`, `SKILL.md`, `internal/mcp/resources/*.md`, hooks, `Taskfile.yml`, `.github/workflows/`, `testdata/golden/`, `testdata/wireoracle/`). The positive control (a planted `codegraph unlock` string in a scratch file) was confirmed caught by the same `rg` invocation before trusting the zero-hit result on the real tree, satisfying D-10's discipline.

Two non-obvous, verified findings materially affect how the plan should be written: (1) **shell completions and man pages are not committed artifacts in this repo** — they are generated live from the installed binary's Cobra tree at Homebrew-cask install time (`generate_completions_from_executable`, `codegraph man <dir>`), and Cobra's own `IsAvailableCommand()` gate (which both the bash-completion generator and `doc.GenManTree`'s predicate honor) excludes `Hidden: true` commands automatically — so "completions and man pages reflect the new surface" (VERB-06) is satisfied *for free* by registering the stubs as `Hidden: true`, with nothing to regenerate or commit; (2) this repo's own `.github/workflows/pr-title.yml` Conventional-Commits regex requires the breaking-change bang **after** the scope — `type(scope)!: subject` — not `type!(scope): subject`. `03-CONTEXT.md` D-15's literal example string (`feat!(cli): fold query…`) does not match that regex and would fail this repo's own PR-title gate; the commit must be written `feat(cli)!: …`.

**Primary recommendation:** Treat this as three sequenced, independently-testable sub-changes (search merge → daemon-unlock move + message updates → stub registration), each landing behind its own commit, with the single `feat(cli)!:` commit carrying only the user-facing surface change (search merge, daemon-unlock move, stub registration, message updates) per D-15 — run the VERB-05 census as a shell step before touching `internal/cli/` and again after, and let `task docs:cli` + `task docs:cli:drift` + `TestEveryRegisteredFlagIsAccountedFor` be the only gates for the generated-reference half of VERB-06 (completions/man need no separate step, per the finding above).

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| VERB-01 | `search --full` returns full node records — `MarshalQueryJSON` envelope under `--json`, human branch shows signature + qualified name; default `search` unchanged | Verified exact source of both `--json` shapes (`internal/query/search.go:259-264`, `MarshalQueryJSON`; `internal/cli/search.go:44-49`, raw `json.Marshal(locs)`) and both engine methods (`Engine.Query`/`Engine.Search`, `internal/query/search.go:119-186`), both backed by the same `matchNodes` (`internal/query/search.go:74-108`) |
| VERB-02 | `search` accepts `query`'s flag superset (`-j/-l/-k/-p`) with a flag-parse test covering both forms | Verified `query.go`'s `StringVarP/IntVarP/BoolVarP` shorthand registrations (lines 88-91) vs. `search.go`'s current no-shorthand `StringVar/IntVar/BoolVar` (lines 65-68); no existing test exercises short-flag forms on either command |
| VERB-03 | `codegraph query …` hidden stub, prints rename to stderr, non-nil error, no `Deprecated` | Verified `root.go`'s `SilenceUsage`/`SilenceErrors` (lines 52-53) route every command error through `cmd/codegraph/main.go:14-17`'s single `os.Exit(1)` — confirmed the sole `os.Exit` call in the `codegraph` binary's own tree (all other `os.Exit` calls are in unrelated `tools/`/`test/`/`scripts/` binaries); verified `cobra.Command.DisableFlagParsing` (cobra v1.10.2 `command.go:243`) as the mechanism that keeps the stub's RunE reachable regardless of what flags/args a caller passes |
| VERB-04 | `unlock` → `daemon unlock`, identical flags/behavior; `codegraph unlock` is the stub | Verified `unlock.go` registers **zero flags** (only cobra's implicit `-h/--help`) and resolves its path via positional `targetRoot(args)`, not a `-p` flag; verified `daemon.go`'s own `-p` flag is registered via `cmd.Flags()` (not `cmd.PersistentFlags()`), so it does not propagate to `daemon start`/`daemon stop`/would-be `daemon unlock` — zero collision risk |
| VERB-05 | Positive-controlled, word-boundary, multiline census finds zero references outside stubs, run before and after `internal/cli/` edits | Census run this session with the exact `rg` invocation below; confirmed zero hits outside the 6-category, 14-line baseline `03-CONTEXT.md` predicted; positive control confirmed |
| VERB-06 | `docs/CLI-REFERENCE.md` regenerated with stubs allowlisted; `task docs:cli:drift` + `TestEveryRegisteredFlagIsAccountedFor` green; completions/man reflect new surface | Verified `tools/clidoc/main.go`'s `documented()` predicate == cobra's `IsAvailableCommand()` == `cli_reference_test.go`'s `documentedByReference()`; verified completions/man are generated at Homebrew-install time from the live binary (`.goreleaser.yaml`'s `generate_completions_from_executable`, `man.go`'s cask post-install hook), never committed; verified cobra's bash-completion generator itself skips `!c.IsAvailableCommand()` commands (`bash_completions.go:450,654`) |
| VERB-07 | Goldens re-frozen in reviewed diff with RED demo; 8-tool MCP set + wire-oracle transcripts unchanged | Verified zero occurrences of `codegraph query`/`codegraph unlock` in `testdata/golden/` or `testdata/wireoracle/transcripts/` this session; verified the 8 MCP tool names (`internal/mcp/tools.go`) are `codegraph_explore/node/search/callers/callees/impact/files/status` and `codegraph_search`'s handler calls `eng.Search` only (`internal/mcp/tools.go:447`), never `eng.Query` — `search --full` cannot touch it |
| VERB-08 | `feat!:` commit, stub removal tracked in next minor via release notes + `docs/CLI-REFERENCE.md` | Verified `release-please-config.json`'s `bump-minor-pre-major: true` + current manifest version `0.13.0`; verified `.github/workflows/pr-title.yml`'s regex requires `type(scope)!:` ordering, not `type!(scope):` |

## User Constraints

<user_constraints>
### Locked Decisions

D-01 through D-15 in `03-CONTEXT.md` are LOCKED and reproduced here for the planner (do not re-derive or explore alternatives — see the full file for rationale on each):

- **D-01:** `search --full` human branch is two lines per hit: line 1 byte-identical to default `search`'s line; line 2 is four-space indent + `QualifiedName  Signature` (two spaces between; when Signature is empty, line 2 is just the qualified name). `search --full … | rg '^\S'` reproduces default output.
- **D-02:** `search --full`'s human branch prints the same WORK-02 worktree notice, in the same position, as default `search`/today's `query`. JSON branches never print it.
- **D-03:** `search --full --json` emits `query.MarshalQueryJSON(nodes)` (old `query --json` envelope); default `search --json` stays `json.Marshal(locs)`. Same engine path: `eng.Query` for `--full`, `eng.Search` otherwise.
- **D-04:** `search` gains `query`'s short flags: `-k`, `-l`, `-j`; `-p` already exists on both. One flag-parse test covers long+short on `search` and `search --full`.
- **D-05:** `query`/`unlock` become hidden dedicated stubs (`Hidden: true`): execute nothing, print to stderr, return non-nil error → exit 1 (the tree's only error exit). No new exit code, no `os.Exit` in the stub, `Deprecated` is not used.
- **D-06:** Stderr text is the rename + full replacement invocation + lifetime:
  - `"query" has been renamed to "search --full" — run: codegraph search --full <term>` then `the "query" stub is removed in the next minor release (v0.15.0)`
  - `"unlock" has been renamed to "daemon unlock" — run: codegraph daemon unlock [path]` then `the "unlock" stub is removed in the next minor release (v0.15.0)`
  - Args/flags passed to a stub are ignored (nothing forwarded, nothing runs).
- **D-07:** `daemon unlock` is `unlock.go`'s body moved verbatim under `daemon` (`Use: "unlock [path]"`, `MaximumNArgs(1)`, identical flags/behaviour). `codegraph unlock` becomes the stub.
- **D-08:** The two live error messages naming `codegraph unlock` (`internal/daemon/lock.go:208`, `internal/cli/index.go:128`, plus the `index.go:126` comment) change to `codegraph daemon unlock` in the same commit as the move, with `index_lock_test.go` updated (RED first). VERB-05 census must find zero old-verb references outside the stubs.
- **D-09:** Stub removal tracked twice: a ROADMAP `## Backlog` `999.x` row via the roadmap tool verb, and the `feat!:` commit's `BREAKING CHANGE:` footer naming the removal minor. `docs/CLI-REFERENCE.md` records the same lifetime via the allowlist reason.
- **D-10:** VERB-05 census is positive-controlled, word-boundary, multiline (`rg -U -w`) for `codegraph query`/`codegraph unlock`, excluding `.planning/`, `CHANGELOG.md`, `web/build/`, the `internal/query` package identifier, JSON `"query"` keys/`query.json` fixture names. Run before `internal/cli/` is edited and again after.
- **D-11:** Re-freeze is a reviewed diff, never a blanket regenerate: `docs/CLI-REFERENCE.md` via `task docs:cli`, completions/man via existing generators, any touched CLI golden/test. RED demonstration reintroduces a visible `query` command (or removes `--full`) and shows `task docs:cli:drift` + `TestEveryRegisteredFlagIsAccountedFor` go red, then reverts byte-clean — recorded in `03-MUTATION-LOG.md` in the `02-MUTATION-LOG.md` shape.
- **D-12:** The two stubs each get one command-level allowlist line in `testdata/cli-reference-allowlist.txt` (actual path: `internal/cli/testdata/cli-reference-allowlist.txt`) whose reason carries the rename target and removal minor.
- **D-13:** Executor reviews the re-frozen diff and commits when gates are green; per-file diff summary in SUMMARY; maintainer reviews at end of phase. No mid-plan checkpoint.
- **D-14:** The 8-tool MCP set and wire-oracle transcripts are not touched; `task test:wireoracle` must pass byte-identically. `search --full` does not change `codegraph_search`'s MCP shape.
- **D-15:** One `feat!:` commit carries the surface change as `feat!(cli): fold query into search --full and unlock into daemon unlock` — **see Common Pitfalls below: this literal string does not match the repo's own `pr-title.yml` gate; use `feat(cli)!: …` instead** — with a `BREAKING CHANGE:` footer stating removed verbs, replacements, and the v0.15.0 stub removal. Census/reference/completions/goldens/backlog/docs are ordinary `test:`/`docs:`/`chore:` commits. Under `bump-minor-pre-major` this cuts a minor; CHANGELOG's BREAKING CHANGES entry is inspected after release-please runs.

### Claude's Discretion

- Exact file layout for the stubs (a `renamed.go` holding both, or one stub per old file) and the flag-parse test's location (`query_cli_test.go` → renamed to match `search`, or a new `search_cli_test.go`).
- The census script's exact `rg` invocation and exclusion list (subject to D-10's bars); whether it lives inline in the plan's `<verify>` or as a small `scripts/` file — a one-time census does not need a Taskfile target.
- How completions and man pages are regenerated and whether they are committed artefacts or generated at release (follow whatever `Taskfile.yml`/`release.yml` already do; do not introduce a new committed artefact). **Research finding: they are generated at Homebrew-install time, never committed — see Architecture Patterns.**
- The `999.x` backlog row's number and wording (via the roadmap verb). **Research finding: `999.5` is the next available id — see Code Examples.**
- Ordering of the ordinary commits around the single `feat!:` commit, as long as every commit leaves `task docs:cli:drift` and `go test ./internal/cli/...` green.

### Deferred Ideas (OUT OF SCOPE)

- Making `codegraph_search` (MCP) return full records under a flag — out of scope: the 8-tool set and its shapes are frozen for this phase.
- A distinct "renamed" exit code (e.g. 2/EX_USAGE) for stubs — declined for a one-release stub; revisit only if a caller needs to distinguish rename from failure.
- Removing the stubs (v0.15.0) — tracked as the backlog row D-09 creates.
- `tools/bench/runner pinnedAt() validates a checkout by git rev-parse HEAD alone` — unrelated bench-runner integrity issue, stays pending for a later phase.
</user_constraints>

## Project Constraints (from CLAUDE.md)

- Project instructions direct GSD workflow enforcement: file-changing tools must be used through a GSD command (`/gsd-execute-phase` for planned phase work) — this phase's plan(s) execute under that umbrella.
- `internal/cli` package doc comment (`root.go:1-16`) already documents the intended post-fold surface narrative — no change needed there beyond what the fold requires.
- Standing repo rules that bind this phase (from STATE.md "Standing decisions"): a gate is not trusted until demonstrated RED against a confirmed-applied mutation; a guard must carry a positive assertion it did its work; `go test -run PATTERN` exits 0 on zero matches — report a count, never bare green; a transcript grep is a claim about the grep, not the product — prove the census finds a planted control before trusting a zero-hit result (already done this session, see below).
- User's global instruction: use `rg` for text search (not `grep`), atomic conventional commits, no `--no-verify`/`--no-gpg-sign` unless requested, never amend/rebase unless asked.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CLI verb registration/dispatch (`search --full`, stubs, `daemon unlock`) | CLI (`internal/cli`) | — | Cobra command tree lives entirely in this package; no server/API tier involved |
| Ranking/record retrieval (`Engine.Query`/`Engine.Search`/`matchNodes`) | Query engine (`internal/query`) | — | Already exists, unchanged by this phase — both CLI commands call into it, neither CLI command duplicates ranking logic |
| Generated docs/completions/man (drift-gated) | Build artifact / CI gate | CLI (source of truth) | `docs/CLI-REFERENCE.md` and `TestEveryRegisteredFlagIsAccountedFor` are downstream of the live Cobra tree — the CLI tier is authoritative, the doc is generated, completions/man are generated even later (Homebrew install time) |
| MCP tool surface (`codegraph_search`, 8-tool set) | MCP server (`internal/mcp`) | — | Explicitly frozen this phase (D-14); the CLI fold must not reach into it — verified `codegraph_search`'s handler calls `eng.Search` only, unaffected by `--full` |
| Release/versioning (`feat!:` commit, CHANGELOG) | CI/release tooling (release-please) | — | Version and changelog derivation is entirely release-please's responsibility; the phase's only job is emitting a correctly-shaped commit |

## Standard Stack

No new external dependencies. This phase edits existing Go source using already-vendored libraries.

### Core (existing, unchanged versions)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/spf13/cobra` | v1.10.2 (verified: `go.mod:17`, module cache) | CLI command tree, flag parsing, hidden-command semantics | Already the project's CLI framework; `Hidden`/`DisableFlagParsing` fields used by this phase are stock cobra, no upgrade needed |
| `github.com/spf13/pflag` | (cobra's transitive dep) | Flag registration/short-forms (`-j/-l/-k/-p`) | Already used by every existing command; `-p/--path` shorthand pattern to replicate is `search.go`'s own existing `StringVarP` call |
| `github.com/spf13/cobra/doc` | v1.10.2 | `docs/CLI-REFERENCE.md` generation (`tools/clidoc`), man pages (`man.go`) | Already the project's doc-generation mechanism; `documented()`/`IsAvailableCommand()` predicate is what makes `Hidden: true` sufficient for VERB-06 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `Hidden: true` stub + `DisableFlagParsing: true` | `cobra.Command.Deprecated` | Explicitly rejected by D-05/D-06 and the milestone's Out-of-Scope table: `Deprecated` prints a warning but still executes the command — opposite of "exits non-zero, nothing executed" |
| `Hidden: true` stub | A hidden alias silently forwarding to `search --full` | Explicitly rejected by the milestone's Out-of-Scope table: changes default output shape for anyone still typing `query` and gives scripts no signal |

**Installation:** none — no `go get`/`go mod` changes required for this phase.

**Version verification:** `go.mod:17` pins `github.com/spf13/cobra v1.10.2` — verified present in the local module cache this session (`go env GOMODCACHE`). No version bump needed or recommended.

## Package Legitimacy Audit

Not applicable — this phase installs no new external packages. No `go get`, no `npm install`, no new `go.mod` requires.

## Architecture Patterns

### System Architecture Diagram

```
                     codegraph <verb> [args] [flags]
                              │
                              ▼
                    root.go: newRootCmd()
                    (SilenceUsage/SilenceErrors)
                              │
              ┌───────────────┼────────────────────────┬──────────────┐
              ▼               ▼                         ▼              ▼
     newSearchCmd()   newQueryStubCmd()          newDaemonCmd()   (other verbs,
     (merged query+       Hidden:true            ├─ start           unchanged)
      search, --full   DisableFlagParsing:true    ├─ stop
      gate)             RunE: print rename        └─ unlock (moved from
              │           to stderr, return err       top-level unlock.go,
              │                    │                  body unchanged)
              ▼                    ▼                         │
      if --full:            main.go: os.Exit(1)              ▼
        eng.Query()          (the ONLY exit-path       daemon.Unlock(codegraphDir)
      else:                   in the codegraph          (internal/daemon/lock.go —
        eng.Search()          binary's own tree)         message text updated to
              │                                            say "daemon unlock")
              ▼
     if --full && --json:
       query.MarshalQueryJSON(nodes)   ← old query.json envelope
     elif --json:
       json.Marshal(locs)              ← unchanged search shape
     else (human):
       print WORK-02 notice, then
       1 line (search) or 2 lines (--full) per match
              │
              ▼
     (unaffected: internal/mcp/tools.go's codegraph_search
      handler calls eng.Search only — never sees --full)
```

### Recommended Project Structure

No new directories. Files touched, per Claude's Discretion on exact layout:
```
internal/cli/
├── search.go          # gains --full flag, -k/-l/-j shorthands, dual RunE branch
├── query.go            # RunE body extracted into search.go; file becomes the
│                        # stub registration (newQueryCmd → hidden stub), OR
│                        # deleted with stub moved to a new renamed.go — Claude's discretion
├── unlock.go            # RunE body moves into daemon.go's newDaemonUnlockCmd;
│                        # file becomes the stub registration, OR deleted — Claude's discretion
├── daemon.go            # gains newDaemonUnlockCmd(), registered in newDaemonCmd()
├── root.go              # AddCommand list: query/unlock stay registered (now as
│                        # stub commands), daemon gains its unlock child
├── index.go             # index.go:126 comment, :128 error string updated
├── query_cli_test.go    # TestQueryCmd's assertions retarget the stub's behavior;
│                        # flag-parse test for -j/-l/-k/-p added (search + search --full)
├── notice_test.go       # noticeCommandCases' "query" row becomes a "search --full" row
├── daemon_test.go       # gains daemon-unlock dispatch tests
├── index_lock_test.go   # lines 123-127: assertion flips to expect "daemon unlock"
│                        # and no longer forbids it
└── testdata/
    └── cli-reference-allowlist.txt   # +2 lines (query, unlock) with reason+lifetime
internal/daemon/lock.go  # line 208 message text updated
```

### Pattern 1: Merging two RunE bodies behind a bool flag, not renaming

**What:** `search --full` must dispatch to `query.go`'s current logic (`eng.Query` + `MarshalQueryJSON` + two-line human render); bare `search` keeps `search.go`'s current logic (`eng.Search` + `json.Marshal(locs)` + one-line render). This is a **merge**, not a rename of `query`'s `Use:` string — both commands' code paths must survive inside one `newSearchCmd()`.

**When to use:** Whenever two Cobra commands share enough machinery (same path resolution, same engine open, same notice placement) that a flag can select between their two RunE tails without duplicating the shared prefix.

**Example (based on verified current source, `internal/cli/query.go` + `internal/cli/search.go`):**
```go
// Source: internal/cli/query.go:49-84 (eng.Query/MarshalQueryJSON branch, D-01/D-03),
// internal/cli/search.go:27-61 (eng.Search/json.Marshal branch, unchanged default)
func newSearchCmd() *cobra.Command {
	var path, kind string
	var limit int
	var jsonOut, full bool

	cmd := &cobra.Command{
		Use:   "search <term>",
		Short: "Lexically search symbol names/qualified names",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := resolveStartPath(path)
			if err != nil {
				return err
			}
			eng, closer, err := query.OpenAt(start)
			if err != nil {
				return err
			}
			defer closer.Close()

			if full {
				nodes, err := eng.Query(args[0], kind, limit)
				if err != nil {
					return err
				}
				if jsonOut {
					data, err := query.MarshalQueryJSON(nodes)
					if err != nil {
						return err
					}
					return writeJSONLine(cmd, data)
				}
				out := cmd.OutOrStdout()
				fmt.Fprint(out, query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))
				for _, n := range nodes {
					fmt.Fprintf(out, "%s (%s) %s:%d\n", n.Name, n.Kind, n.FilePath, n.StartLine)
					fmt.Fprintf(out, "    %s  %s\n", n.QualifiedName, n.Signature) // D-01: empty Signature → just qualified name
				}
				return nil
			}

			locs, err := eng.Search(args[0], kind, limit)
			if err != nil {
				return err
			}
			if jsonOut {
				data, err := json.Marshal(locs)
				if err != nil {
					return err
				}
				return writeJSONLine(cmd, data)
			}
			out := cmd.OutOrStdout()
			fmt.Fprint(out, query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))
			for _, l := range locs {
				fmt.Fprintf(out, "%s (%s) %s:%d\n", l.Name, l.Kind, l.FilePath, l.StartLine)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().StringVarP(&kind, "kind", "k", "", "restrict to one node kind")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "cap on results returned")
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "emit JSON output")
	cmd.Flags().BoolVar(&full, "full", false, "return full node records (signature, qualified name) instead of locations")

	return cmd
}
```
This is illustrative — D-01's exact line-2 formatting (`n.QualifiedName` then two spaces then `n.Signature`, empty-Signature case just the qualified name) must be written as its own small helper the planner's task can unit-test directly, per D-04's "write the flag-parse test first" guidance from PITFALLS.md Pitfall 4.

### Pattern 2: Hidden stub that always fires its message, regardless of what the caller typed

**What:** D-06 requires "any args/flags passed to a stub are ignored (nothing is forwarded, nothing runs)" — but if the stub command has no registered flags, a caller typing `codegraph query --json main` gets pflag's own `unknown flag: --json` error instead of the D-06 redirect message, because cobra's flag-parsing step runs and fails *before* `RunE` is ever reached. **Verified fix:** set `DisableFlagParsing: true` on the stub commands. This bypasses pflag entirely — cobra passes every raw arg (including anything that looks like a flag) straight to `RunE`'s `args []string`, so the redirect message fires unconditionally.

**When to use:** Any Cobra command whose contract is "always run this exact behavior no matter what the user typed after the verb" — exactly the stub's contract here.

**Verified mechanism (cobra v1.10.2 `command.go`):**
```go
// DisableFlagParsing field: command.go:243
// Checked at three points in Command.execute()/ValidateArgs()/ValidateRequiredFlags(): command.go:964, 1181, 1869
// Default Args (nil) already resolves to ArbitraryArgs (command.go:1172-1176) — no Args
// field needs to be set on the stub; any number of positional args is already accepted.
```

**Example:**
```go
// Source: pattern derived from man.go's existing Hidden:true precedent
// (internal/cli/man.go:46-51) plus the verified DisableFlagParsing mechanism above.
func newQueryStubCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "query",
		Hidden: true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), `"query" has been renamed to "search --full" — run: codegraph search --full <term>`)
			fmt.Fprintln(cmd.ErrOrStderr(), `the "query" stub is removed in the next minor release (v0.15.0)`)
			return fmt.Errorf("codegraph: \"query\" has been renamed to \"search --full\"")
		},
	}
}
```
Note: `cli_reference_test.go`'s walk (`cliReferenceTree()` → `cmd.InitDefaultHelpFlag()`) registers `--help` on this command **regardless** of `DisableFlagParsing` (verified: `InitDefaultHelpFlag()` is called unconditionally at `command.go:916`, independent of the parsing-disable flag) — this is exactly why D-12's allowlist line is required per stub.

### Pattern 3: Moving `daemon.AddCommand` without a flag collision

**What:** `unlock.go` registers **zero flags** today (verified: no `cmd.Flags()` calls in the file at all — only cobra's implicit `-h/--help`; path resolution is via positional `targetRoot(args)`, matching `init`/`index`/`uninit`'s convention, not the `-p` flag convention `query`/`search`/`daemon` use). `daemon.go`'s own `-p/--path` flag on the bare `daemon` command is registered via `cmd.Flags()` (non-persistent — verified `internal/cli/daemon.go:90`), not `cmd.PersistentFlags()`, so it does **not** propagate to child commands. Moving `unlock`'s body verbatim under `daemon` therefore introduces zero flag collisions.

**Example:**
```go
// Source: internal/cli/daemon.go:91 (existing AddCommand call, extended)
cmd.AddCommand(newDaemonStartCmd(), newDaemonStopCmd(), newDaemonUnlockCmd())

// newDaemonUnlockCmd is unlock.go's newUnlockCmd() body, moved verbatim (D-07):
// Source: internal/cli/unlock.go:21-43, unchanged except the enclosing func name
func newDaemonUnlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock [path]",
		Short: "Clear a stale daemon lock",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := targetRoot(args)
			if err != nil {
				return err
			}
			codegraphDir := filepath.Join(root, codegraphDirName)
			msg, err := daemon.Unlock(codegraphDir)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), msg)
			return nil
		},
	}
	return cmd
}
```

### Pattern 4: Generated docs/completions/man need no separate regeneration step for hidden commands

**What:** `tools/clidoc/main.go`'s `documented()` predicate (`cmd.IsAvailableCommand() && !cmd.IsAdditionalHelpTopicCommand()`) is byte-identical to `cli_reference_test.go`'s `documentedByReference()` and to cobra's own bash-completion generator's skip condition (`!c.IsAvailableCommand()`, verified `bash_completions.go:450,654`). `doc.GenManTree` (used by `man.go`'s cask post-install hook) also respects `IsAvailableCommand()`. Since `Hidden: true` makes `IsAvailableCommand()` return `false` (verified `command.go:1607-1621`), registering the stubs as `Hidden: true` automatically excludes them from `docs/CLI-REFERENCE.md`, `codegraph completion {bash,zsh,fish}`, and `codegraph man <dir>` output — no completions/man regeneration step exists to run because **neither is a committed artifact**: completions are generated live from the *installed* binary at Homebrew-cask install time (`generate_completions_from_executable` in `.goreleaser.yaml`'s cask stanza) and man pages are generated live by `codegraph man <dir>` (the cask's own post-install hook invoking the freshly-installed binary). The only "regenerate" step in this phase is `task docs:cli` for `docs/CLI-REFERENCE.md`, since that IS a committed, drift-gated file.

**When to use:** Whenever adding/hiding a command and needing to reason about what "the generated surface" actually requires touching.

### Anti-Patterns to Avoid

- **Using `cobra.Command.Deprecated` for the stubs:** Explicitly rejected (D-05, milestone Out-of-Scope) — it prints a warning but still executes the command, the opposite of "nothing runs."
- **A silent alias forwarding `query` → `search --full`:** Explicitly rejected (milestone Out-of-Scope) — changes default output shape for anyone still typing `query`, gives scripts no failure signal.
- **Renaming `query`'s `Use:` string to `search --full` instead of merging RunE bodies:** Produces a cobra registration collision (two commands both claiming `search`) or silently drops one code path — PITFALLS.md Pitfall 4's exact warning.
- **Trusting a "zero references" census result without a positive control:** This repo's own standing rule ("a transcript grep is a claim about the grep, not the product") — the planted-control step is mandatory, not optional, and was performed this session (see census section below).
- **Regenerating shell completions or man pages as a phase deliverable:** They are not committed; there is nothing to regenerate or commit for them — see Pattern 4.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| "Redirect old verb to new verb, exit non-zero" | A custom `os.Exit` call inside the stub's RunE, or a hand-rolled arg/flag scanner | `Hidden: true` + `DisableFlagParsing: true` + returning a non-nil error from `RunE` (main.go's existing single `os.Exit(1)`) | Reuses the tree's one existing, tested error-exit path; a stub-local `os.Exit` would bypass `SilenceUsage`/`SilenceErrors` and introduce a second exit mechanism `TestEveryRegisteredFlagIsAccountedFor` and the golden suite never anticipated |
| "Hide a command from generated docs/completions/man" | A bespoke exclusion list in `tools/clidoc` or a new allowlist mechanism for completions | `Hidden: true` (cobra's own `IsAvailableCommand()` gate) | Every consumer (`clidoc`, `cli_reference_test.go`, cobra's bash-completion generator, `doc.GenManTree`) already keys off this exact field — a parallel exclusion mechanism would need to be kept in sync with all four by hand |

**Key insight:** Every mechanism this phase needs (hide from docs/completions, force RunE regardless of flags, single error-exit path) already exists as a stock cobra field or an established repo convention (`man.go`'s `Hidden: true` precedent). There is no place in this phase where a custom mechanism is justified.

## Common Pitfalls

### Pitfall 1: `feat!(cli):` does not match this repo's own PR-title gate

**What goes wrong:** `03-CONTEXT.md` D-15's literal commit-message example is `feat!(cli): fold query into search --full and unlock into daemon unlock`. `.github/workflows/pr-title.yml`'s regex is:
```
^(feat|fix|perf|refactor|docs|chore|ci|test|build|revert)(\([a-z0-9_-]+\))?!?: .+
```
verified this session by reading the workflow file directly. This regex requires the optional `(scope)` group **before** the optional `!`, i.e. `type(scope)!: subject`. A title/commit-message of `feat!(cli): …` fails to match (the `!` consumes the position right after `feat`, then `(cli)` is left unconsumed before the required `: `), because under this repo's squash-merge model the **PR title** is what release-please parses as the commit message.

**Why it happens:** Conventional Commits spec places `!` after the scope; `feat!(cli):` is an easy, plausible-looking but non-conformant ordering that a hand-typed commit message can slip into.

**How to avoid:** Write the commit/PR title as `feat(cli)!: fold query into search --full and unlock into daemon unlock`.

**Warning signs:** `pr-title.yml`'s CI job failing on the phase's feature PR with `PR title is not Conventional-Commits-shaped`.

### Pitfall 2: Registering the stub with default flag parsing silently changes its behavior depending on what the user typed

**What goes wrong:** Without `DisableFlagParsing: true`, `codegraph query --json main` fails at cobra's flag-parsing step with pflag's own `unknown flag: --json` error — never reaching the D-06 redirect message — while `codegraph query main` (no flags) does reach it. This produces an inconsistent, confusing UX exactly opposite to D-06's "any args/flags passed to a stub are ignored" requirement.

**Why it happens:** Cobra parses flags before invoking `RunE`; a command with no registered flags rejects any `--flag`-shaped token by default.

**How to avoid:** Set `DisableFlagParsing: true` on both stub commands (verified field: cobra v1.10.2 `command.go:243`) — this bypasses pflag and guarantees `RunE` always runs regardless of arguments.

**Warning signs:** A stub test that only exercises `codegraph query <term>` (no flags) passes, but `codegraph query --json <term>` produces a different error message — a genuine, easy-to-miss gap in test coverage for VERB-03/VERB-04.

### Pitfall 3 (from PITFALLS.md Pitfall 4): the merge, not a rename, is the real work

Reproduced from `.planning/research/PITFALLS.md` (verified present, read this session): `query.go`'s `--json` path calls `query.MarshalQueryJSON` (full-node envelope) while `search.go`'s calls raw `json.Marshal(locs)` (locations only) — genuinely different shapes. `query` has `-j/-l/-k` shorthands (`StringVarP`/`IntVarP`/`BoolVarP`); `search` does not (`StringVar`/`IntVar`/`BoolVar`, verified this session at `search.go:65-68`). A naive `Use:` rename either collides two commands both claiming `search` or silently drops one code path. Write the flag-parse test first (assert `-j`,`-l`,`-k` parse on both `search` and `search --full`) before touching `root.go` registration.

### Pitfall 4 (from PITFALLS.md Pitfall 5): the census must run BEFORE editing `internal/cli/`, not after

Reproduced and independently re-verified this session (see census output below): prose consumers of the old verb names live outside `internal/cli/` (docs, error messages in `internal/daemon`/`internal/cli/index.go`) and none of them are caught by `go vet` or the compiler. Running the census only after editing risks missing an already-stale reference that the edit itself made harder to find (e.g., if `index.go`'s comment were edited first without checking `lock.go`'s matching string).

### Pitfall 5 (from PITFALLS.md Pitfall 6): `bump-minor-pre-major` decouples "is this breaking" from "what version results" — write `!` anyway

Verified: `release-please-config.json` sets `"bump-minor-pre-major": true` and the manifest's current version is `0.13.0`. A `feat(cli)!:` commit on this config bumps `0.13.0 → 0.14.0` (a minor, not a major) — this is correct and expected; the `!`/`BREAKING CHANGE:` footer's job here is CHANGELOG categorization, not version arithmetic. Do not drop the `!` because "the version didn't change as much as I expected."

### Pitfall 6 (from PITFALLS.md Pitfall 7): a golden/wire-oracle fixture that doesn't mention the old verb will vacuously pass either way

Verified this session: zero occurrences of `codegraph query`/`codegraph unlock` exist in `testdata/golden/` or `testdata/wireoracle/transcripts/` today, so VERB-07's RED demonstration must be staged deliberately (temporarily reintroduce the old verb into whatever comparison the phase's own new/touched test covers, confirm RED, then revert) rather than relying on an existing fixture accidentally containing the string.

## Code Examples

### The verified census invocation (VERB-05, D-10)

```bash
# Source: this session's live run against the repo at HEAD (44550da4)
rg -nU -w "codegraph query|codegraph unlock" \
  --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .
```
Verified output (14 lines, matching `03-CONTEXT.md`'s predicted 6-category baseline exactly):
```
./docs/CLI-REFERENCE.md:37,46,604,609,853,858    (6 lines — generated reference)
./internal/cli/index_lock_test.go:123,124        (2 lines — test assertion)
./internal/cli/index.go:126,128                  (2 lines — comment + error string)
./internal/cli/query.go:35                       (1 line — doc comment)
./internal/cli/unlock.go:12                      (1 line — doc comment)
./internal/daemon/lock.go:208,274                (2 lines — error string + doc comment)
```
Positive control verified: a scratch file containing `run codegraph unlock now` outside the repo tree was found by the identical invocation before trusting the zero-hit result on any other path.

### The ROADMAP backlog verb (D-09)

```bash
# Verified: `phase next-decimal 999` is a read-only query against the live ROADMAP.md
node <gsd-core>/bin/gsd-tools.cjs phase next-decimal 999
# → {"found": false, "base_phase": "999", "next": "999.5", "existing": ["999.2", "999.4"]}

# The sanctioned write verb (verified router source, phase-command-router.cjs):
node <gsd-core>/bin/gsd-tools.cjs phase add "remove the query/unlock rename stubs — v0.15.0" --id 999.5
```
`999.5` is the next available backlog id (verified: only `999.2` and `999.4` currently exist in `.planning/ROADMAP.md`'s `## Backlog` section).

### The message-site updates required by D-08 (verbatim current text, to be changed)

```
// internal/daemon/lock.go:208 (verified verbatim)
return fmt.Errorf("%w: pid=%d — stop it first, or run `codegraph unlock` once it has exited", ErrLockLive, info.PID)

// internal/cli/index.go:126 (verified verbatim, comment)
// remedies. `codegraph unlock` is the verb live today;
// internal/cli/index.go:128 (verified verbatim, error string)
return fmt.Errorf("%w: another process holds %s open — stop it first with `codegraph daemon stop`, or run `codegraph unlock` once it has exited", priorErr, storeDir)

// internal/cli/index_lock_test.go:123-127 (verified verbatim, assertion to flip)
if !strings.Contains(err.Error(), "codegraph unlock") {
    t.Fatalf("index --force error = %q; want it to name `codegraph unlock` (the verb live today, not `daemon unlock`)", err.Error())
}
if strings.Contains(err.Error(), "daemon unlock") {
    t.Fatalf("index --force error = %q; must NOT name `daemon unlock` — that verb does not exist until Phase 3", err.Error())
}
```
All four occurrences of the string `codegraph unlock` here must become `codegraph daemon unlock` in the same commit; `index_lock_test.go`'s two assertions must swap (the first now expects `daemon unlock`, the second's "must NOT" becomes obsolete and should be removed or inverted since `daemon unlock` now legitimately exists).

### The WORK-02 notice test table needs its `query` row retargeted

Verified `internal/cli/notice_test.go:130-142`'s `noticeCommandCases` table includes a `{"query", []string{"query", "Alpha", "-p", path}}` row, exercised by three tests (`TestNoticeOnWorktreeMismatch`, `TestNoticeAbsentOnCleanTree`, `TestNoticeSuppressedInJSON`). After the fold, `query` is a stub that never opens an engine or prints a notice — this row must become a `search --full` row (D-02 requires the same notice behavior on `search --full`'s human branch) or the tests will spuriously pass against a stub that prints nothing rather than testing the notice at all.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `codegraph query <term>` (full records) + `codegraph search <term>` (locations, separate command) | `codegraph search <term> [--full]` (single command, flag selects shape) | This phase | Fewer top-level verbs; existing scripts using `query` get a loud, actionable stub for one release |
| `codegraph unlock [path]` (top-level) | `codegraph daemon unlock [path]` (nested under `daemon`) | This phase | Groups lock-clearing with the rest of the daemon lifecycle surface; existing scripts using `unlock` get the same stub treatment |

**Deprecated/outdated:** None — no third-party dependency version changes in this phase.

## Assumptions Log

No claims in this research are tagged `[ASSUMED]`. Every factual claim about source code, test files, generated-doc tooling, cobra semantics, and CI gate regexes was verified this session by reading the actual file or invoking the actual command against the live repository at HEAD (`44550da4`). The two locked decisions quoted verbatim from `03-CONTEXT.md` (D-06's message text, D-15's commit-message template) are `[CITED: 03-CONTEXT.md]` — they are the user's own locked wording, not this research's claim, except where explicitly flagged as non-conformant (Pitfall 1).

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed beyond the Pitfall 1 correction (which does not change any locked decision's substance, only the literal string ordering).

## Open Questions

1. **Exact two-line human-render formatting for `search --full` when `Signature` is non-empty vs. empty**
   - What we know: D-01 specifies "four spaces of indent then `QualifiedName  Signature` (two spaces between); when `Signature` is empty — packages, files — line 2 is just the qualified name."
   - What's unclear: Whether "line 2 is just the qualified name" means `    QualifiedName` (still four-space indented, no trailing spaces) or some other exact whitespace shape. This affects a byte-exact test assertion.
   - Recommendation: The plan should specify this as a literal string in its own task (e.g., a small `renderFullLine(n *schema.Node) string` helper with its own unit test), so the ambiguity is resolved by a test assertion rather than left to executor interpretation.

2. **Where the flag-parse test (VERB-02) and the two-line-format test should physically live**
   - What we know: `03-CONTEXT.md` explicitly leaves this to Claude's Discretion (`query_cli_test.go` renamed, or a new `search_cli_test.go`).
   - What's unclear: Nothing blocking — purely a file-organization choice.
   - Recommendation: Given `query_cli_test.go` currently holds `TestQueryCmd` AND `TestSearchCmd` (verified, same file), the least-churn choice is renaming this file to `search_cli_test.go` and retargeting `TestQueryCmd`'s body to test the stub, with `TestSearchCmd` gaining the `--full` subtests.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building/testing the CLI | ✓ | 1.27.1 installed; repo pins `go 1.26.6` in `go.mod` and forces `GOTOOLCHAIN=go1.26.6` for every `docs:cli`/`docs:cli:drift` invocation | GOTOOLCHAIN env var auto-downloads/uses the pinned 1.26.6 toolchain transparently — no separate install needed |
| `rg` (ripgrep) | VERB-05 census | ✓ | 15.2.0 | — |
| `git` | Census file enumeration, `docs:cli:drift`'s `git ls-files` | ✓ | 2.54.0 | — |
| `task` (go-task) | `task docs:cli`, `task docs:cli:drift`, `task test:wireoracle` | ✓ | 3.52.0 | — |
| `gsd-tools` (`phase next-decimal`, `phase add`) | D-09's backlog row | ✓ | verified `phase next-decimal 999` returns `999.5` this session | — |

No missing dependencies.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go's stdlib `testing` package, `go test` |
| Config file | none — plain `go test ./internal/cli/...` |
| Quick run command | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -run 'TestSearch\|TestQuery\|TestDaemon\|TestNotice'` |
| Full suite command | `GOTOOLCHAIN=go1.26.6 go test ./... && task docs:cli:drift && task test:wireoracle` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| VERB-01 | `search --full --json` emits `MarshalQueryJSON` envelope; default `search --json` unchanged | unit | `go test ./internal/cli/... -run TestSearchCmd -v` | ✅ existing (`query_cli_test.go`), extend |
| VERB-01 | Human `--full` branch shows signature + qualified name as a two-line superset | unit | `go test ./internal/cli/... -run TestSearchFullHuman -v` | ❌ Wave 0 — new test |
| VERB-02 | `-j/-l/-k/-p` all parse on `search` and `search --full` | unit | `go test ./internal/cli/... -run TestSearchFlagShortForms -v` | ❌ Wave 0 — new test |
| VERB-03 | `codegraph query` exits non-zero, prints rename message, executes nothing | unit | `go test ./internal/cli/... -run TestQueryStub -v` | ❌ Wave 0 — new test |
| VERB-04 | `daemon unlock` behaves identically to old `unlock`; `codegraph unlock` is the stub | unit | `go test ./internal/cli/... -run TestDaemonUnlockCmd -v` | ❌ Wave 0 — new test (no prior CLI-level `unlock` test existed — verified via `rg` this session) |
| VERB-04 | `internal/daemon`/`internal/cli/index.go` messages say `daemon unlock` | unit | `go test ./internal/cli/... -run TestIndexLock -v` | ✅ existing (`index_lock_test.go`), flip assertion |
| VERB-05 | Zero old-verb references outside stubs, before and after | shell census | `rg -nU -w "codegraph query|codegraph unlock" --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' .` | manual/CI shell step, not a `go test` |
| VERB-06 | `docs/CLI-REFERENCE.md` current; allowlist covers both stubs; flag accounting green | drift gate + unit | `task docs:cli:drift && go test ./internal/cli/... -run TestEveryRegisteredFlagIsAccountedFor -v` | ✅ existing gates |
| VERB-07 | Goldens/wire-oracle unaffected, RED demonstrated once | integration + manual RED | `task test:wireoracle` + `03-MUTATION-LOG.md`'s recorded RED/revert cycle | ✅ existing suite; RED demo is a one-time manual step per plan |
| VERB-08 | Commit message conforms to `pr-title.yml`'s regex | shell check | `echo "feat(cli)!: fold query into search --full and unlock into daemon unlock" | grep -qE '^(feat|fix|perf|refactor|docs|chore|ci|test|build|revert)(\([a-z0-9_-]+\))?!?: .+'` | manual pre-commit check |

### Sampling Rate
- **Per task commit:** `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... ./internal/daemon/...`
- **Per wave merge:** `GOTOOLCHAIN=go1.26.6 go test ./... && task docs:cli:drift`
- **Phase gate:** Full suite green (`go test ./...`, `task docs:cli:drift`, `task test:wireoracle`) before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] New unit tests for the stub commands (`TestQueryStub`, `TestUnlockStub`) — VERB-03
- [ ] New unit test for `search --full`'s two-line human render (`TestSearchFullHuman`) — VERB-01
- [ ] New flag-parse test covering `-j/-l/-k/-p` short and long forms on `search` and `search --full` — VERB-02
- [ ] New unit test for `daemon unlock`'s dispatch (`TestDaemonUnlockCmd`) — VERB-04 (no prior CLI-level `unlock` test existed to migrate)
- [ ] `03-MUTATION-LOG.md` — the RED/revert demonstration record for VERB-07, in the `02-MUTATION-LOG.md` shape

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth surface touched — local CLI only |
| V3 Session Management | no | No session concept in this phase |
| V4 Access Control | no | No access-control surface touched |
| V5 Input Validation | yes | Already handled by `Engine.Query`/`Engine.Search`'s existing `ValidateKind`/`validateLimit`/empty-term checks (`internal/query/search.go:119-161`, unchanged by this phase) |
| V6 Cryptography | no | No cryptographic material involved |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| A stub command silently executing partial logic before erroring (info leak or partial side effect) | Tampering / Information Disclosure | D-05's explicit "executes nothing" contract — verified achievable via `Hidden: true` + `DisableFlagParsing: true` with RunE doing nothing but printing and returning an error; no engine open, no file I/O in the stub |
| A stale error message pointing users at a removed verb, training them to type something that now silently no-ops or errors confusingly | (not STRIDE — a UX/support-burden risk, not a security one) | D-08's requirement that both live message call sites are updated in the same commit as the move, verified with exact line numbers above |

No new attack surface is introduced by this phase — it is a pure command-tree reshuffle over already-validated engine calls.

## Sources

### Primary (HIGH confidence — read directly this session)
- `internal/cli/query.go`, `search.go`, `unlock.go`, `daemon.go`, `root.go`, `man.go`, `index.go` — full file reads
- `internal/query/search.go` — full file read (`Engine.Query`, `Engine.Search`, `matchNodes`, `MarshalQueryJSON`, `Location`)
- `internal/schema/graph.pb.go` — `Node` struct field list (lines 104-138)
- `internal/daemon/lock.go` (lines 190-220) — `acquire`'s live error message
- `internal/cli/index.go` (lines 100-140) — `priorCoverageGeneration` switch and its message
- `internal/cli/query_cli_test.go`, `notice_test.go`, `cli_reference_test.go`, `index_lock_test.go` — full/partial reads
- `internal/cli/testdata/cli-reference-allowlist.txt` — full read (confirmed actual path, one existing `man` entry)
- `tools/clidoc/main.go` — full read
- `Taskfile.yml` lines 481-536 (`docs:cli`, `docs:cli:drift`) — read directly
- `.github/workflows/pr-title.yml` — full read (regex verified)
- `release-please-config.json`, `.release-please-manifest.json` — full read
- `internal/mcp/tools.go` (relevant sections) — 8 tool names, `codegraph_search`'s `eng.Search` call site
- cobra v1.10.2 source (`command.go`, `bash_completions.go`) — `IsAvailableCommand`, `DisableFlagParsing`, `ValidateArgs` default, `InitDefaultHelpFlag` call sites, hidden-command completion skip logic
- `.planning/research/PITFALLS.md` (Pitfalls 4-7) — full read
- `.planning/ROADMAP.md` `## Backlog` section — full read (existing `999.2`/`999.4` rows)
- `gsd-tools phase next-decimal 999` — live invocation, returned `999.5`
- `gsd-tools/bin/lib/phase-command-router.cjs` — `phase add` argument parsing (`--id`)
- Live `rg` census invocation against the repo at HEAD `44550da4`

### Secondary (MEDIUM confidence)
- `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md` — read for shape/format precedent only, not for factual claims about this phase

### Tertiary (LOW confidence)
None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, all mechanisms verified against the actual vendored cobra version in this repo's module cache
- Architecture: HIGH — every pattern is grounded in a direct read of the current source it modifies
- Pitfalls: HIGH — 4 of 6 pitfalls are re-verified restatements of the project's own pre-existing PITFALLS.md research; the other 2 (PR-title regex mismatch, DisableFlagParsing requirement) are new findings verified this session against live workflow/library source

**Research date:** 2026-09-16
**Valid until:** Stable — this research is scoped entirely to this phase's own source tree at a single commit (`44550da4`); it does not depend on external library churn (cobra v1.10.2 is pinned) and should not go stale before the phase executes. Re-verify the census if the phase's own edits substantially precede plan execution.
