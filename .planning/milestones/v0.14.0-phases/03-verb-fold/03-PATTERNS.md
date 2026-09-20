# Phase 3: Verb Fold - Pattern Map

**Mapped:** 2026-09-16
**Files analyzed:** 12 (created/modified) — all git-tracked source, no gitignored mirrors involved (single-repo project, no `.gsd/capabilities` mirrors in play)
**Analogs found:** 12 / 12

All analogs below were verified present via `git ls-files -- <path>` before citation.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|-----------------|---------------|
| `internal/cli/search.go` (gains `--full`, `-k/-l/-j` shorthands, dual RunE branch) | controller (CLI command) | request-response | `internal/cli/query.go` (source of the branch being merged in) | exact — same package, same shared helpers (`resolveStartPath`, `writeJSONLine`) |
| `internal/cli/query.go` → becomes hidden stub (or new `renamed.go`) | controller (CLI command, stub) | request-response | `internal/cli/man.go` (the one existing `Hidden: true` precedent) | role-match — `man` is hidden but not a "renamed" stub; closest available registered-command shape |
| `internal/cli/unlock.go` → becomes hidden stub | controller (CLI command, stub) | request-response | `internal/cli/man.go` | role-match — same as above |
| `internal/cli/daemon.go` (gains `newDaemonUnlockCmd`, `AddCommand` extended) | controller (CLI command, subcommand registration) | request-response | `internal/cli/daemon.go` itself — `newDaemonStartCmd`/`newDaemonStopCmd` registration shape | exact — same file, same parent-child idiom already used twice |
| `internal/cli/root.go` (`AddCommand` list, no structural change — stubs stay registered by name) | route (root command tree) | request-response | `internal/cli/root.go` itself (current `AddCommand` call) | exact — edit in place, not a new file |
| `internal/cli/index.go` (lines 126 comment, 128 error string) | controller (CLI command, error text) | request-response | `internal/daemon/lock.go:208` (sibling message needing the identical text change) | exact — both are the two D-08 message sites, same fix applied in the same commit |
| `internal/daemon/lock.go` (line 208 error string) | service (lock acquisition) | request-response | `internal/cli/index.go:126-128` | exact — paired sibling edit |
| `internal/cli/query_cli_test.go` (or renamed `search_cli_test.go`) | test | request-response | `internal/cli/query_cli_test.go` itself (`TestQueryCmd`, `TestSearchCmd`) | exact — extend/retarget in place |
| `internal/cli/notice_test.go` (`noticeCommandCases`' `"query"` row) | test | request-response | `internal/cli/notice_test.go` itself | exact — edit one table row |
| `internal/cli/index_lock_test.go` (lines 123-127 assertions) | test | request-response | `internal/cli/index_lock_test.go` itself | exact — flip two assertions |
| `internal/cli/daemon_test.go` (new `TestDaemonUnlockCmd`) | test | request-response | `internal/cli/daemon_test.go`'s `TestDaemonStopCmd_DispatchesToStopMatching` (dispatch-style test) | role-match — no prior CLI-level `unlock` test exists; closest sibling dispatch test in the same file |
| `internal/cli/testdata/cli-reference-allowlist.txt` (+2 lines) | config (generated-docs allowlist) | batch | `internal/cli/testdata/cli-reference-allowlist.txt` itself (existing `codegraph man` line) | exact — same file, same one-line-per-command grammar |
| `.planning/ROADMAP.md` `## Backlog` (999.x row, via roadmap tool verb) | config (planning artifact) | batch | N/A — written exclusively through `gsd-tools phase add`, never hand-edited (see `planning-artifacts.md`) | tool-owned, no source analog applicable |

## Pattern Assignments

### `internal/cli/search.go` (controller, request-response) — gains `--full`

**Analog:** `internal/cli/query.go` (the RunE body being merged in) + `internal/cli/search.go` (the file being edited, current default branch)

**Imports pattern** (`query.go` lines 1-10, current `search.go` lines 1-10 — merge, add nothing new):
```go
package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/query"
)
```
No new import is required for the `--full` branch — `query.MarshalQueryJSON` and `query.OpenAt` are already reachable via the existing `internal/query` import; `encoding/json` already imported by `search.go` for the default `--json` branch.

**Shared helpers already in package `cli`, reuse verbatim** (`internal/cli/query.go:12-33`):
```go
// resolveStartPath resolves the -p/--path flag every query/serve command
// accepts...
func resolveStartPath(p string) (string, error) {
	if p != "" {
		return p, nil
	}
	return os.Getwd()
}

// writeJSONLine writes already-marshaled JSON data to cmd's configured
// stdout followed by a trailing newline...
func writeJSONLine(cmd *cobra.Command, data []byte) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}
```
These live in `query.go` today. If `query.go` is deleted (Claude's Discretion on stub layout), these two helpers MUST move to `search.go` or a shared file first — every other CLI command file depends on them.

**Core dual-branch RunE pattern** — merge `query.go`'s `--full` tail (lines 49-84) into `search.go`'s existing RunE (lines 27-62), gated on a new `full bool`:

`query.go`'s branch to move in verbatim (lines 61-84, D-03's `--full` path):
```go
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
}
return nil
```
`search.go`'s branch that stays as the default (lines 39-61, unchanged):
```go
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
```
Per D-01, the `--full` human branch's per-hit render is NOT `query.go`'s original one-liner — it gains a second line. Write it as its own small helper (RESEARCH's Open Question 1 recommendation) so the exact whitespace (`    ` + `QualifiedName` + two spaces + `Signature`, or just `    QualifiedName` when `Signature` is empty) is pinned by a unit test, not left inline:
```go
// renderFullLine renders the D-01 second line for a --full hit: four-space
// indent, QualifiedName, then (if Signature is non-empty) two spaces and
// Signature. Line 1 (the default search line, byte-identical) is rendered
// separately by the existing "%s (%s) %s:%d\n" format — this helper only
// ever emits line 2.
func renderFullLine(n *schema.Node) string {
	if n.Signature == "" {
		return "    " + n.QualifiedName
	}
	return "    " + n.QualifiedName + "  " + n.Signature
}
```

**Auth/Guard pattern:** none — no auth surface in this CLI (verified by RESEARCH's Security Domain section: V2/V3/V4 all "no").

**Flag registration — merge query's shorthands into search (D-04):**

`query.go`'s shorthand registration (lines 88-91, the pattern to adopt):
```go
cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
cmd.Flags().StringVarP(&kind, "kind", "k", "", "restrict to one node kind")
cmd.Flags().IntVarP(&limit, "limit", "l", 0, "cap on results returned")
cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "emit JSON output")
```
`search.go`'s current no-shorthand registration (lines 65-68, being replaced):
```go
cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
cmd.Flags().StringVar(&kind, "kind", "", "restrict to one node kind")
cmd.Flags().IntVar(&limit, "limit", 0, "cap on results returned")
cmd.Flags().BoolVar(&jsonOut, "json", "j", false, "emit JSON output")
```
Result: `search.go` adopts `query.go`'s four lines verbatim, plus a fifth new line:
```go
cmd.Flags().BoolVar(&full, "full", false, "return full node records (signature, qualified name) instead of locations")
```

**Error handling pattern:** every error from `resolveStartPath`, `query.OpenAt`, `eng.Query`/`eng.Search`, `json.Marshal`/`query.MarshalQueryJSON` is returned bare (`return err`) — no wrapping, no local error type. `SilenceUsage`/`SilenceErrors` on the command tree (`root.go:52-53`) means `cmd/codegraph/main.go`'s single `os.Exit(1)` is what turns this into a process exit; nothing in `search.go` prints or exits directly.

**WORK-02 notice placement (D-02):** in BOTH branches, `fmt.Fprint(out, query.WorktreeNotice(...))` fires only in the human branch, immediately after the `--json` early return — verified identical placement already exists in both `query.go:80` and `search.go:57`; the merge must preserve this exact position in each branch, not hoist it above the `if jsonOut` block.

---

### `internal/cli/query.go` → hidden rename stub (VERB-03)

**Analog:** `internal/cli/man.go` (the only existing `Hidden: true` precedent in the codebase)

**Imports pattern** — the stub needs less than `man.go` (no `os`, no `doc`):
```go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)
```

**`Hidden: true` precedent** (`internal/cli/man.go:46-51`):
```go
func newManCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "man <dir>",
		Short:  "Generate man pages for the full command tree",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
```
Apply the same `Hidden: true` field; doc-comment convention (decision-ID citation, rationale for hiding) also follows `man.go`'s own comment block style (lines 11-45).

**Core stub pattern (new — no existing "stub" analog in this codebase; synthesized from man.go's Hidden precedent + RESEARCH's verified `DisableFlagParsing` mechanism):**
```go
// newQueryCmd (retargeted, VERB-03/D-05/D-06): "query" is renamed to
// "search --full". This hidden stub executes nothing — it prints the
// rename plus the exact replacement invocation plus the stub's lifetime
// to stderr, then returns a non-nil error so main.go's single os.Exit(1)
// fires. DisableFlagParsing:true (cobra command.go:243) is required so
// ANY args/flags typed after "query" reach RunE unconditionally instead
// of failing earlier at pflag's unknown-flag check (D-06: "any args/flags
// passed to a stub are ignored"). Deprecated is deliberately NOT used —
// it would still execute the command (rejected, D-05).
func newQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "query",
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), `"query" has been renamed to "search --full" — run: codegraph search --full <term>`)
			fmt.Fprintln(cmd.ErrOrStderr(), `the "query" stub is removed in the next minor release (v0.15.0)`)
			return fmt.Errorf(`codegraph: "query" has been renamed to "search --full"`)
		},
	}
}
```
The `unlock` stub is byte-for-byte the same shape, swapping the two message strings for D-06's `unlock` text and `Use: "unlock"`.

**Error handling pattern:** the stub's `RunE` return value is the ONLY thing that drives the non-zero exit (verified: `cmd/codegraph/main.go`'s single `os.Exit(1)` path, `root.go`'s `SilenceUsage`/`SilenceErrors`) — no `os.Exit` call belongs inside the stub itself (would bypass `SilenceUsage`/`SilenceErrors` and introduce a second exit mechanism, per RESEARCH's Don't-Hand-Roll table).

**Testing pattern** — flag-parse/dispatch test analog, `internal/cli/query_cli_test.go:24-51` (`TestQueryCmd`'s existing `--json`/default/`--kind`/uninitialized subtests) is the file/table shape to retarget into a stub test:
```go
func TestQueryStub(t *testing.T) {
	// analog shape: execCmd(...) from internal/cli/cli_test.go:55
	_, stderr, err := execCmd("query", "main")
	if err == nil {
		t.Fatal("codegraph query: expected a non-nil error (the rename stub), got nil")
	}
	if !strings.Contains(stderr, `renamed to "search --full"`) {
		t.Fatalf("codegraph query: stderr = %q, want the rename message", stderr)
	}
}
```

---

### `internal/cli/unlock.go` → hidden rename stub (VERB-04/D-05/D-06)

**Analog:** same as `query.go`'s stub above (`internal/cli/man.go`'s `Hidden: true` precedent + the `DisableFlagParsing` mechanism)

Body to move OUT of this file first (D-07 — moves verbatim to `daemon.go`, see below), then this file's `newUnlockCmd` becomes the stub with `unlock`'s own D-06 message text:
```go
fmt.Fprintln(cmd.ErrOrStderr(), `"unlock" has been renamed to "daemon unlock" — run: codegraph daemon unlock [path]`)
fmt.Fprintln(cmd.ErrOrStderr(), `the "unlock" stub is removed in the next minor release (v0.15.0)`)
return fmt.Errorf(`codegraph: "unlock" has been renamed to "daemon unlock"`)
```

---

### `internal/cli/daemon.go` (controller, request-response) — gains `newDaemonUnlockCmd`

**Analog:** `internal/cli/daemon.go` itself — `newDaemonStartCmd`/`newDaemonStopCmd`'s own registration idiom (this is a same-file, same-pattern addition, the strongest possible match)

**Imports pattern** (current `daemon.go` lines 1-18) — gains one import the moved body needs:
```go
import (
	"errors"
	"fmt"
	"io"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/seanb4t/codegraph-go/internal/cli/tui"
	"github.com/seanb4t/codegraph-go/internal/daemon"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/watch"
)
```
`unlock.go`'s body needs nothing new beyond what `daemon.go` already imports (`path/filepath`, `fmt`, `github.com/seanb4t/codegraph-go/internal/daemon` — all already present).

**Registration pattern** (`daemon.go:91`, extend the existing `AddCommand` call):
```go
cmd.AddCommand(newDaemonStartCmd(), newDaemonStopCmd())
```
becomes:
```go
cmd.AddCommand(newDaemonStartCmd(), newDaemonStopCmd(), newDaemonUnlockCmd())
```

**Core pattern — `unlock.go`'s body moved verbatim (D-07)**, source `internal/cli/unlock.go:21-43`:
```go
// newDaemonUnlockCmd builds `codegraph daemon unlock [path]` (D-07, moved
// verbatim from the old top-level newUnlockCmd): clears a stale daemon
// lockfile left behind by a crash. Mirrors newUninitCmd's guarded shape
// (targetRoot(args) positional-arg resolution)... [comment carries over
// unchanged from unlock.go:12-20]
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
Zero flag collision risk verified: `unlock.go` registers zero flags of its own (only cobra's implicit `-h/--help`); `daemon.go`'s own `-p/--path` on the bare `daemon` command is registered via `cmd.Flags()` (non-persistent, `daemon.go:90`), not `cmd.PersistentFlags()`, so it does not propagate to `daemon unlock`.

**Testing pattern** — analog `internal/cli/daemon_test.go`'s dispatch-style tests (`TestDaemonStopCmd_DispatchesToStopMatching`, line 241) is the shape for the new `TestDaemonUnlockCmd`; no prior CLI-level `unlock` test exists to migrate (verified via `rg`), so this is a net-new test using the same file's existing `execCmd`/fixture conventions (`setupIndexedFixture`, `internal/cli/query_cli_test.go:14-22`).

---

### `internal/cli/root.go` (route, request-response) — registration unchanged in shape

**Analog:** `internal/cli/root.go` itself, current `AddCommand` list (lines 55-61):
```go
root.AddCommand(newInitCmd(), newIndexCmd(), newUninitCmd(),
	newQueryCmd(), newSearchCmd(), newCallersCmd(), newCalleesCmd(),
	newImpactCmd(), newAffectedCmd(), newFilesCmd(), newStatusCmd(),
	newNodeCmd(), newExploreCmd(), newServeCmd(), newSyncCmd(),
	newDaemonCmd(), newUnlockCmd(), newVersionCmd(), newTelemetryCmd(),
	newUpgradeCmd(), newInstallCmd(), newUninstallCmd(),
	newGithooksCmd(), newManCmd(), newUiCmd())
```
No structural change: `newQueryCmd()` and `newUnlockCmd()` stay in this exact list, at the exact same call sites — they now return hidden stub commands instead of live commands. Only the package doc comment (lines 1-16, 33-43) needs prose updated to describe the new surface (`query`/`unlock` are now rename stubs; `search` gained `--full`; `daemon unlock` is the live verb) — no `AddCommand` line is added, removed, or reordered.

---

### `internal/cli/index.go` (controller, error text) + `internal/daemon/lock.go` (service, error text) — D-08

**Analog for both:** each other — these are the two paired sibling sites the same commit must fix identically.

**`internal/cli/index.go`, current text (lines 126, 128 — verified verbatim):**
```go
// Never indexed, or a store with no Meta record yet
// (D-11): silent, floor stays 0 — the only branch that
// preserves pre-FIX-06 behavior.
case errors.Is(priorErr, graphstore.ErrStoreLocked):
	// Another process holds the store open (D-10): refuse
	// BEFORE RemoveAll, naming the holder and the two live
	// remedies. `codegraph unlock` is the verb live today;
	// `daemon unlock` does not exist until Phase 3.
	return fmt.Errorf("%w: another process holds %s open — stop it first with `codegraph daemon stop`, or run `codegraph unlock` once it has exited", priorErr, storeDir)
```
Change: `codegraph unlock` → `codegraph daemon unlock` in the error string (line 128) and drop/update the now-stale comment (line 126-127, "Phase 3" framing is now past-tense).

**`internal/daemon/lock.go:208` (verified verbatim):**
```go
} else if !isStale(info) {
	return fmt.Errorf("%w: pid=%d — stop it first, or run `codegraph unlock` once it has exited", ErrLockLive, info.PID)
```
Change: `codegraph unlock` → `codegraph daemon unlock`.

**Testing pattern** — `internal/cli/index_lock_test.go:120-128` (verified verbatim, the RED-first assertions to flip):
```go
if !strings.Contains(err.Error(), "codegraph daemon stop") {
	t.Fatalf("index --force error = %q; want it to name `codegraph daemon stop`", err.Error())
}
if !strings.Contains(err.Error(), "codegraph unlock") {
	t.Fatalf("index --force error = %q; want it to name `codegraph unlock` (the verb live today, not `daemon unlock`)", err.Error())
}
if strings.Contains(err.Error(), "daemon unlock") {
	t.Fatalf("index --force error = %q; must NOT name `daemon unlock` — that verb does not exist until Phase 3", err.Error())
}
```
Post-fold shape: the second assertion's `Contains(..., "codegraph unlock")` must become `Contains(..., "codegraph daemon unlock")`, and the third assertion (the "must NOT" guard) is now obsolete — invert it or delete it, since `daemon unlock` is now the correct, expected text. Keep the `codegraph daemon stop` assertion unchanged (unaffected by this phase).

---

### Generated-docs allowlist — `internal/cli/testdata/cli-reference-allowlist.txt` (D-12)

**Analog:** the file's own existing `man` line (verified verbatim, full file contents above):
```
codegraph man	hidden by v0.5.0 D-02 — the Homebrew cask post-install hook's man-page generator, never an interactive command; its only flag is cobra's --help (12-CONTEXT D-05)
```

**Grammar** (from the file's own header comment, lines 1-6): `<full command path>[ --flag]<TAB><reason>`; a bare `<command path>` (no `--flag`) covers a hidden command's flags entirely.

**Lines to add (D-12's exact reason shape — rename target + removal minor):**
```
codegraph query	hidden rename stub for `search --full` (VERB-03); removed in v0.15.0
codegraph unlock	hidden rename stub for `daemon unlock` (VERB-04); removed in v0.15.0
```

**Consumer/gate pattern** — `internal/cli/cli_reference_test.go:147-187`'s `TestEveryRegisteredFlagIsAccountedFor` walks every registered command (`cmd.InitDefaultHelpFlag()` called unconditionally, `cli_reference_test.go:178`, mirroring `doc.GenMarkdownCustom`) and requires every flag be accounted for either by appearing in `docs/CLI-REFERENCE.md` or by an allowlist line — this is why each stub needs its own line even though it registers no flags beyond cobra's implicit `--help`.

**`documented()` predicate that makes `Hidden: true` sufficient for VERB-06 for free** (`tools/clidoc/main.go:61-63`, byte-identical to `cli_reference_test.go`'s `documentedByReference()`):
```go
func documented(cmd *cobra.Command) bool {
	return cmd.IsAvailableCommand() && !cmd.IsAdditionalHelpTopicCommand()
}
```
No separate completions/man regeneration step exists to run — neither is a committed artifact (verified: generated live at Homebrew-cask install time). The only regeneration command for this phase is `task docs:cli` (for `docs/CLI-REFERENCE.md`).

---

## Shared Patterns

### Silence + single-exit-path (applies to every touched command)
**Source:** `internal/cli/root.go:52-53`, `cmd/codegraph/main.go` (single `os.Exit(1)`)
```go
root := &cobra.Command{
	...
	SilenceUsage:  true,
	SilenceErrors: true,
}
```
**Apply to:** `search.go` (merged command), both stubs, `daemon unlock`. No command in this tree calls `os.Exit` itself; every error return propagates to `main.go`'s one exit call.

### WORK-02 worktree notice placement (applies to `search.go`'s both branches)
**Source:** `internal/cli/query.go:74-83`, `internal/cli/search.go:52-60` (identical placement in both today)
```go
out := cmd.OutOrStdout()
fmt.Fprint(out, query.WorktreeNotice(eng.WorktreeMismatch(cmd.Context())))
for _, x := range results {
	fmt.Fprintf(out, "%s (%s) %s:%d\n", x.Name, x.Kind, x.FilePath, x.StartLine)
}
```
**Apply to:** `search.go`'s merged `--full` branch (D-02: same notice, same position, human branch only, never JSON) — and its `noticeCommandCases` test row (`internal/cli/notice_test.go:130-141`) must be retargeted from `{"query", []string{"query", ...}}` to a `{"search --full", []string{"search", "Alpha", "--full", "-p", path}}`-shaped row, since the stub `query` no longer opens an engine or prints anything.

### `Hidden: true` — the one mechanism covering docs, completions, man, and allowlist gating uniformly
**Source:** `internal/cli/man.go:46-51`; consumed identically by `tools/clidoc/main.go:61-63`, `internal/cli/cli_reference_test.go`'s `documentedByReference()`, and cobra's own bash-completion generator (`bash_completions.go:450,654`, verified in RESEARCH)
**Apply to:** both rename stubs (`query`, `unlock`). Do not invent a parallel exclusion list for completions/man — every consumer already keys off this one field.

### `DisableFlagParsing: true` — required companion to `Hidden: true` for stubs specifically
**Source:** cobra v1.10.2 `command.go:243` (verified by RESEARCH this session; no in-repo precedent exists yet — this is the first `DisableFlagParsing` use in the codebase)
**Apply to:** both rename stubs only. Without it, `codegraph query --json main` fails at pflag's flag-parse step with `unknown flag: --json` before `RunE` ever runs, violating D-06's "any args/flags passed to a stub are ignored."

### Message-site pairing (D-08) — both live `codegraph unlock` references change together
**Source:** `internal/daemon/lock.go:208`, `internal/cli/index.go:126-128` — the two sites, edited in the same commit, with `internal/cli/index_lock_test.go`'s RED-first assertion flip
**Apply to:** any future site the VERB-05 census turns up beyond the four the research already enumerated (docs/CLI-REFERENCE.md's 6 generated hits regenerate automatically via `task docs:cli`, not by hand).

### Census script shape (VERB-05, D-10) — positive-controlled, word-boundary, multiline
**Source:** RESEARCH's verified invocation (Code Examples section), reproduced here as the pattern to copy into the plan's `<verify>` step or a `scripts/` file:
```bash
rg -nU -w "codegraph query|codegraph unlock" \
  --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .
```
Must be run once BEFORE any `internal/cli/` edit and once AFTER, with a planted-control check (a scratch string containing `codegraph unlock` proven caught by the same invocation) preceding trust in a zero-hit result — the standing repo rule ("a transcript grep is a claim about the grep, not the product").

### Mutation-log shape (VERB-07's RED demonstration)
**Source:** `.planning/phases/02-guards-ci-wiring-docs-burn-down/02-MUTATION-LOG.md` — read for shape/format precedent only (not read in full this session; already verified by RESEARCH as the template `03-MUTATION-LOG.md` must follow). The planner's plan for VERB-07 should instruct the executor to reproduce that file's own section structure (mutation description, RED command + output, revert command + confirmation) rather than inventing a new log shape.

## No Analog Found

None — every file in scope has at least a role-match analog in the existing `internal/cli` package. The two rename stubs have no exact "renamed stub" precedent (only `man.go`'s `Hidden: true` precedent, which is not itself a redirect/stub), so their core RunE pattern above is synthesized from `man.go`'s field usage plus RESEARCH's independently-verified `DisableFlagParsing` mechanism rather than copied from a second in-repo example — flagged as role-match, not exact, in the classification table.

## Metadata

**Analog search scope:** `internal/cli/` (all 20+ command files, all 12+ test files), `internal/daemon/lock.go`, `tools/clidoc/main.go`, `internal/cli/testdata/`
**Files scanned:** 15 read in full or by targeted line range this session (`query.go`, `search.go`, `unlock.go`, `daemon.go`, `daemon_test.go` symbol list, `man.go`, `root.go`, `index.go` lines 95-139, `lock.go` lines 190-220, `query_cli_test.go` full, `index_lock_test.go` lines 100-149, `notice_test.go` lines 100-160, `cli-reference-allowlist.txt` full, `tools/clidoc/main.go` lines 55-84, `cli_reference_test.go` lines 147-187) — all confirmed git-tracked via `git ls-files` before citation
**Pattern extraction date:** 2026-09-16
