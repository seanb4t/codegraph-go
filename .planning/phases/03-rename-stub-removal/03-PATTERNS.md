# Phase 3: Rename-Stub Removal - Pattern Map

**Mapped:** 2026-09-26
**Files analyzed:** 6 (2 deletions, 2 edits, 1 rewrite, 1 new evidence file)
**Analogs found:** 6 / 6

All analog paths below were verified git-tracked via `git ls-files --error-unmatch <path>` before being named here (tracked-source gate #3645). None are gitignored install/runtime mirrors.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `internal/cli/renamed.go` (DELETE) | controller (cobra command constructors) | request-response | itself — deletion has no "write" analog; see Shared Patterns for what remains after deletion | n/a (deletion) |
| `internal/cli/renamed_test.go` (DELETE) | test (unit, in-process) | request-response | itself — deletion, no rewrite | n/a (deletion) |
| `internal/cli/root.go` (EDIT: `AddCommand` list + comments at ~14-15, ~46, ~99-100) | controller / route registration | request-response | itself (edit in place) — no external analog needed, exact target lines identified below | exact (self) |
| `internal/cli/testdata/cli-reference-allowlist.txt` (EDIT: remove lines 8-9) | config (test fixture / allowlist data) | CRUD (static table) | itself (edit in place) | exact (self) |
| `test/integration/renamed_stubs_test.go` → rewritten as `test/integration/removed_verbs_test.go` (or kept name, Claude's discretion) | test (integration, spawned-binary) | request-response (subprocess exec) | `internal/cli/hook_pretooluse_codex_test.go` (positive-assertion executed-case-count pattern) + `test/integration/main_test.go` (spawned-binary helpers `copyFixture`/`runBinary`) | exact (role+flow) for the harness; role-match for the count-assertion idiom |
| `.planning/phases/03-rename-stub-removal/03-MUTATION-LOG.md` (NEW) | test evidence / migration-log document | batch (static analysis transcript) | `.planning/milestones/v0.14.0-phases/03-verb-fold/03-MUTATION-LOG.md` (Family (a) and (c)) and `.planning/phases/01-changie-baseline/01-MUTATION-LOG.md` | exact |

## Pattern Assignments

### `internal/cli/renamed.go` (DELETE) and `internal/cli/renamed_test.go` (DELETE)

**Analog:** N/A — pure deletion per D-01/D-02. No replacement file inherits their content. The only obligation is that nothing else in the tree still references `newQueryCmd`/`newUnlockCmd` or imports `renamed.go`'s content after deletion (verified by `root.go`'s edit below and by the census).

**Current content being removed** (`internal/cli/renamed.go:59-87`, both constructors):
```go
func newQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "query",
		Short:              `Renamed to "search --full" — stub removed in v0.15.0`,
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("%s\n%s",
				`"query" has been renamed to "search --full" — run: codegraph search --full <term>`,
				`the "query" stub is removed in the next minor release (v0.15.0)`,
			)
		},
	}
}

func newUnlockCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "unlock",
		Short:              `Renamed to "daemon unlock" — stub removed in v0.15.0`,
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("%s\n%s",
				`"unlock" has been renamed to "daemon unlock" — run: codegraph daemon unlock [path]`,
				`the "unlock" stub is removed in the next minor release (v0.15.0)`,
			)
		},
	}
}
```
Both files are removed wholesale (`git rm internal/cli/renamed.go internal/cli/renamed_test.go`), no salvage.

---

### `internal/cli/root.go` (EDIT)

**Analog:** self — the file already contains everything the executor needs; this is a surgical edit, not a copy-from-elsewhere job.

**Exact edit targets** (verified this session, current line numbers):

1. **`AddCommand` call, lines 126-132** — remove `newQueryCmd(),` and `newUnlockCmd(),` from the argument list:
```go
	root.AddCommand(newInitCmd(), newIndexCmd(), newUninitCmd(),
		newQueryCmd(), newSearchCmd(), newCallersCmd(), newCalleesCmd(),
		newImpactCmd(), newAffectedCmd(), newFilesCmd(), newStatusCmd(),
		newNodeCmd(), newExploreCmd(), newServeCmd(), newSyncCmd(),
		newDaemonCmd(), newUnlockCmd(), newVersionCmd(), newTelemetryCmd(),
		newUpgradeCmd(), newInstallCmd(), newUninstallCmd(),
		newGithooksCmd(), newManCmd(), newHookCmd(), newUiCmd())
```
`newQueryCmd()` sits in the first continuation line, `newUnlockCmd()` in the fourth. The `commandGroups` map (lines 63-89) and the `AddGroup`/grouping loop (134-144) need **no** edit — `query`/`unlock` were never keys in `commandGroups` (confirmed: absent from the map), consistent with D-05's note that "the group loop after it is unaffected, because the stubs never carried a `GroupID`."

2. **Package doc comment, lines 14-15** (the "hidden rename stubs (renamed.go)" sentence inside the `Package cli` doc block, lines 1-22):
```go
// `query` and `unlock` remain registered only as
// hidden rename stubs (renamed.go) that print the rename and exit 1,
// removed entirely in v0.15.0.
```
Rewrite or delete this sentence — D-05 requires stale stub prose updated in the same commit, and D-06's consequence-for-new-text rule forbids the two-token phrase `codegraph query`/`codegraph unlock` appearing in any tracked file outside `.planning/` (this comment currently uses only bare `query`/`unlock`, which is fine — keep that convention, don't upgrade it to `codegraph query`).

3. **`newRootCmd` doc comment, line 46** and **lines 99-100** (inside the comment block spanning roughly 91-113):
```go
// `query` and `unlock` stay registered as hidden rename stubs
// (renamed.go) for one release (D-05/D-09).
```
Same treatment: remove or rewrite to past tense / removal note, staying within the "refer to the verbs as `query`/`unlock`" convention (D-06).

**Everything else in `root.go` is untouched** — `ErrAlreadyInitialized`/`ErrNotInitialized` (32-39), `commandGroups` (48-89), `installHelpFunc` (162-172), `Execute`/`NewRootCmd` (174-189) all stay byte-identical.

---

### `internal/cli/testdata/cli-reference-allowlist.txt` (EDIT)

**Analog:** self. Current full content (5 entries, lines 1-11):
```
# DOCS-06 allowlist: flags docs/CLI-REFERENCE.md cannot show by design
# (hidden commands, hidden flags, deprecated flags). One entry per line:
# <full command path>[ --flag]<TAB><reason>. An entry matching nothing FAILS the guard.
# A bare "<command path>" entry (no --flag) covers a HIDDEN command's flags
# only. A flag on a documented (visible) command always requires its own
# per-flag entry: "<command path> --<flag>"<TAB><reason>.
codegraph man	hidden by v0.5.0 D-02 — the Homebrew cask post-install hook's man-page generator, never an interactive command; its only flag is cobra's --help (12-CONTEXT D-05)
codegraph query	hidden rename stub for `search --full` (VERB-03); removed in v0.15.0
codegraph unlock	hidden rename stub for `daemon unlock` (VERB-04); removed in v0.15.0
codegraph hook	hidden parent of the PreToolUse nudge entry point (Phase 6 D-01a); its only flag is cobra's --help
codegraph hook pretooluse	hidden; reached only through the installed guard script, never typed by a human (Phase 6 D-01/D-01a)
```
**Edit:** delete exactly lines 8-9 (the `codegraph query` / `codegraph unlock` bare-command-path entries). Lines 1-7 (header comment + `codegraph man` entry) and lines 10-11 (`codegraph hook` entries) are untouched. The allowlist's own grammar — "An entry matching nothing FAILS the guard" — is why leaving these two lines in place after the commands are removed would itself go RED in `TestEveryRegisteredFlagIsAccountedFor` (`internal/cli/cli_reference_test.go:250`, `stale allowlist entry: ... (matches no registered command or flag)`); this is the removal's built-in second guard per D-03.

**Guard that reads this file (context, unchanged):** `internal/cli/cli_reference_test.go`:
- `cliReferenceAllowlistPath = "testdata/cli-reference-allowlist.txt"` (line 23)
- `cliReferenceMinCommands = 26`, `cliReferenceMinFlags = 50` (lines 33-34) — floors, do NOT edit (Pitfall 3: 37 commands / 117 flags after removal, comfortably above)
- `cliReferenceMinVisibleCommands = 24` (line 300) — same, do not edit
- `parseCLIReferenceAllowlist` (line 92), `hasAllowEntry` (line 278), stale-entry detection (line 250)

---

### `test/integration/renamed_stubs_test.go` → rewritten as the removal's guard

**Primary analog (harness):** `test/integration/main_test.go` — `copyFixture` (lines 219-250) and `runBinary` (lines 257-270), already imported and used unchanged by the current file; reuse verbatim, no edit needed to the helpers themselves.

**Imports pattern** (current file, lines 1-6):
```go
package integration

import (
	"os/exec"
	"testing"
)
```

**Current core pattern being replaced** (`test/integration/renamed_stubs_test.go:23-65`, full file) — this is the file to overwrite/rewrite in place:
```go
func TestRenamedStubsPrintExactlyOnce(t *testing.T) {
	dir := copyFixture(t)
	if _, stderr, err := runBinary(t, dir, nil, "init", dir); err != nil {
		t.Fatalf("init fixture via subprocess binary: %v: %s", err, stderr)
	}

	const queryWant = "\"query\" has been renamed to \"search --full\" — run: codegraph search --full <term>\n" +
		"the \"query\" stub is removed in the next minor release (v0.15.0)\n"
	const unlockWant = "\"unlock\" has been renamed to \"daemon unlock\" — run: codegraph daemon unlock [path]\n" +
		"the \"unlock\" stub is removed in the next minor release (v0.15.0)\n"

	cases := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{"query bare", []string{"query", "main"}, queryWant},
		{"query with --json flag", []string{"query", "--json", "main"}, queryWant},
		{"unlock bare", []string{"unlock", "-p", dir}, unlockWant},
		{"unlock with nonexistent path arg", []string{"unlock", "-p", "/nonexistent"}, unlockWant},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runBinary(t, dir, nil, tc.args...)

			if stdout != "" {
				t.Fatalf("%v: expected empty stdout, got %q", tc.args, stdout)
			}
			if stderr != tc.wantStderr {
				t.Fatalf("%v: stderr = %q, want exactly %q (the message printed exactly once by main.go, WR-01)", tc.args, stderr, tc.wantStderr)
			}

			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("%v: err = %v (%T), want *exec.ExitError", tc.args, err, err)
			}
			if exitErr.ExitCode() != 1 {
				t.Fatalf("%v: exit code = %d, want 1", tc.args, exitErr.ExitCode())
			}
		})
	}
}
```
**What changes per D-01/D-02:** keep the four-case table shape (`query bare`, `query --json`, `unlock bare`, `unlock with path`) and the `copyFixture`/`runBinary` harness verbatim. Change:
- `wantStderr` values: both become the single shared unknown-command line, not the two-line rename message. D-06's ban on the two-token phrase `codegraph query`/`codegraph unlock` in tracked files outside `.planning/` means the constant must be built without literally writing that phrase, e.g. reference the verbs as `query`/`unlock` in comments and build the expected string as `fmt.Sprintf("unknown command %q for \"codegraph\"\n", "query")` or an equivalent that never concatenates the literal two-token phrase.
- The `init` subprocess call before the table may be unnecessary now since neither guard invocation depends on the fixture being indexed (cobra never reaches command logic) — D-02 doesn't require removing it, but the guard's own comment/doc should stop citing WR-01 (that ticket is closed) and instead cite D-01/D-02/VERB-09.
- **Add the positive-assertion executed-case-count check** (rule `84d1gfpywd`) immediately after the `cases := []struct{...}{...}` literal, mirroring `internal/cli/hook_pretooluse_codex_test.go:169-171`:
```go
	if len(cases) != 4 {
		t.Fatalf("test table has %d cases, want 4", len(cases))
	}
```
This is the concrete, load-bearing pattern for D-02's "asserts an executed-case count of 4, and fails on 0" requirement — copy this idiom exactly (`len(cases) != N` guard, `t.Fatalf` with the actual and wanted count), not a generic comment.

**Secondary analog (idiom-only, count assertion):** `internal/cli/hook_pretooluse_codex_test.go:169-171`:
```go
	if len(cases) != 24 {
		t.Fatalf("test table has %d cases, want 24", len(cases))
	}
```
This confirms the idiom already exists twice in the codebase (`plain_golden_test.go:465-466` uses a `<` floor variant instead of `!=` exact-count; for this guard, D-02's "count of 4" is an exact fixed number, so the `!= 4` / `hook_pretooluse_codex_test.go` form is the correct one to copy, not the floor form).

**Rewritten file's package/location:** stays in `package integration`; `git mv test/integration/renamed_stubs_test.go test/integration/removed_verbs_test.go` is one reasonable discretion choice (preserves blame) — either filename satisfies D-02.

**Error handling / exit-code pattern (unchanged, keep verbatim):**
```go
			exitErr, ok := err.(*exec.ExitError)
			if !ok {
				t.Fatalf("%v: err = %v (%T), want *exec.ExitError", tc.args, err, err)
			}
			if exitErr.ExitCode() != 1 {
				t.Fatalf("%v: exit code = %d, want 1", tc.args, exitErr.ExitCode())
			}
```

**Context for what the guard now proves:** `cmd/codegraph/main.go:13-18` (full file), the single error-print/exit seam both the old and new guard exercise:
```go
func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```
This is unchanged by the phase — it's the seam the guard proves fires with cobra's own unknown-command error instead of the stub's custom error, per the RESEARCH.md architecture diagram.

---

### `.planning/phases/03-rename-stub-removal/03-MUTATION-LOG.md` (NEW)

**Analog:** `.planning/milestones/v0.14.0-phases/03-verb-fold/03-MUTATION-LOG.md` Family (a) (lines 23-129) and Family (c) (lines 703-847); secondary structural analog `.planning/phases/01-changie-baseline/01-MUTATION-LOG.md` (lines 1-50) for the document-header/worktree-setup shape.

**Document header pattern** (`01-MUTATION-LOG.md:1-16`):
```markdown
# 01-MUTATION-LOG — Changie Baseline

**Phase:** 01-changie-baseline
**Date:** 2026-09-25
**Scope:** Four demonstration families, all appended by 01-02-PLAN.md Task 3, proving the
`check:changie` live-tool guard ...

All mutations happened in a disposable, detached scratch git worktree — never the main working
tree — created and removed within this single Task 3 execution.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>` (or, for
the directory-scoped checks, `git status --porcelain -- <dir>`) is asserted to exit empty/clean.
This proves no pre-existing tracked edit was overwritten by the mutation, and no revert was a
destructive blind checkout of someone else's in-flight work. Every family entry below records this
gate's result at the point it was checked.
```
Adapt to: `# 03-MUTATION-LOG — Rename-Stub Removal`, `**Phase:** 03-rename-stub-removal`, date 2026-09-26, scope naming this phase's plan IDs and the two census runs (before-removal and after-removal, per D-06/D-07).

**Family section pattern — the exact structure to reproduce twice (before-removal and after-removal), copied from `03-verb-fold/03-MUTATION-LOG.md` Family (a) lines 23-129 and Family (c) lines 703-847:**

Required subsections per family, in this order (verbatim structure, not verbatim content):
1. `## Family (X) — <requirement>: positive-controlled census <BEFORE|AFTER> the <removal>`
2. `**Test/guard:**` — the exact `rg` invocation (D-06's byte-for-byte command)
3. `**What are we testing, and why?**` — prose justifying the positive control
4. `**Pre-mutation gate:**` — `git status --porcelain .github` result
5. `**Mutation applied:**` — the untracked `.github/__census_control__.md` creation, exact 3-line content (from CONTEXT.md D-06 / RESEARCH.md Code Examples: `run codegraph unlock now` / `and also run codegraph` / `query now`), noting it is inside `trap 'rm -f .github/__census_control__.md' EXIT` and never `git add`ed
6. `**Observed result**` — verbatim `rg` output including the control hits
7. `**Revert:**` — `rm -f .github/__census_control__.md`
8. `**Byte-clean proof:**` — `git status --porcelain .github` empty + `test ! -e .github/__census_control__.md`
9. `**Real census**` — verbatim clean-tree `rg` output (today: 4 hits / `renamed.go:10-11` + allowlist `:8-9`; expected after: 0 hits, or exactly 1 hit confined to the capability-written `.changes/unreleased/*.yaml` fragment per D-07)
10. `**Verdict:**` — comparison sentence

**Exact `rg` invocation to embed (D-06, byte-for-byte, do not alter):**
```
rg -nU -w --hidden 'codegraph\s+(query|unlock)' --glob '!.planning/**' --glob '!CHANGELOG.md' --glob '!web/build/**' --glob '!.git/**' .
```

**Exact positive-control content to embed (verbatim, from CONTEXT.md D-06 and RESEARCH.md):**
```
run codegraph unlock now
and also run codegraph
query now
```

**Family (c)'s "after" observed-result shape to mirror** (`03-verb-fold/03-MUTATION-LOG.md:758-767`), as the template for this phase's own after-removal census entry:
```
./internal/cli/renamed.go:10:// this phase folds away (Phase 3 VERB-03/VERB-04): `codegraph query <term>`
./internal/cli/renamed.go:11:// into `codegraph search --full <term>`, and `codegraph unlock [path]` into
./internal/cli/testdata/cli-reference-allowlist.txt:8:codegraph query	hidden rename stub for `search --full` (VERB-03); removed in v0.15.0
./internal/cli/testdata/cli-reference-allowlist.txt:9:codegraph unlock	hidden rename stub for `daemon unlock` (VERB-04); removed in v0.15.0
```
`after: 4 lines across 2 files (renamed.go=2, allowlist=2)` — this is this phase's own **before-removal** baseline (matching CONTEXT.md D-06's "today it reports exactly four"); the phase's own **after-removal** census should report **zero** hits (or the single capability-fragment hit per D-07), which is the new, not-yet-precedented number this phase's log must record.

**Commit-history audit pattern to reuse** (`03-verb-fold/03-MUTATION-LOG.md:780-802`):
```
$ F=$(git log --format=%H --grep='^feat(cli)!: ...$' -1)
$ echo "$F"
<sha>
$ git log --format=%s "$F^..HEAD"
...
```
Adapt the grep pattern to whatever exact `feat(cli)!:` subject line the plan settles on for D-04's commit; verify exactly one `feat(` subject, exactly one `!:`, a `BREAKING CHANGE:` footer naming `search --full` / `daemon unlock`, no `ci skip`/`skip ci` anywhere in range.

**Final-gate pattern to reuse** (`03-verb-fold/03-MUTATION-LOG.md:804-827`), adapted to `GOTOOLCHAIN=go1.26.6` prefix per this phase's environment finding:
```
$ GOTOOLCHAIN=go1.26.6 go build ./...
(clean, no output)

$ GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -count=1
ok  	...

$ task docs:cli:drift
docs:cli:drift: compared 1 generated file
docs:cli:drift: docs/CLI-REFERENCE.md byte-identical to a fresh regeneration
```

## Shared Patterns

### Positive-assertion executed-case-count guard (rule `84d1gfpywd`)
**Source:** `internal/cli/hook_pretooluse_codex_test.go:169-171`
**Apply to:** the rewritten `test/integration/*_test.go` guard (D-02's mandatory "asserts an executed-case count of 4, and fails on 0")
```go
	if len(cases) != 24 {
		t.Fatalf("test table has %d cases, want 24", len(cases))
	}
```
Substitute `4` for `24`. This must run immediately after the `cases := []struct{...}{...}` literal and before the `for _, tc := range cases` loop, so a silently-emptied table fails loudly instead of the loop vacuously passing on zero iterations.

### Single error-print/exit seam
**Source:** `cmd/codegraph/main.go:13-18`
**Apply to:** understanding what both the old (deleted) stub test and the new guard exercise; no code change needed here, but the guard's assertions (exit 1, stderr exact match, empty stdout) map directly onto this seam.
```go
func main() {
	if err := cli.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

### Allowlist fail-closed grammar
**Source:** `internal/cli/testdata/cli-reference-allowlist.txt:1-6` (header comment) + `internal/cli/cli_reference_test.go:250` (stale-entry detection)
**Apply to:** confirming the allowlist edit is mandatory, not optional cleanup — "An entry matching nothing FAILS the guard" is the allowlist's own stated contract; leaving `codegraph query`/`codegraph unlock` lines in place after their commands are removed triggers this failure automatically (D-03's "built-in second guard").

### Mutation-log family template (RED/positive-control evidence)
**Source:** `.planning/milestones/v0.14.0-phases/03-verb-fold/03-MUTATION-LOG.md` Family (a) (lines 23-129) and Family (c) (lines 703-847); `.planning/phases/01-changie-baseline/01-MUTATION-LOG.md` (lines 1-50)
**Apply to:** the new `03-MUTATION-LOG.md` this phase must create — reuse the ten-part family structure listed above verbatim in shape; only the `rg` output content and commit SHAs are new.

### GOTOOLCHAIN pin for every go/task invocation
**Source:** `Taskfile.yml` `docs:cli`/`docs:cli:drift` targets (already-established pattern, cited in RESEARCH.md; not re-excerpted here since RESEARCH.md's Environment Availability table and Pitfall 2 already give the exact prefix and rationale)
**Apply to:** every `go build`/`go test`/`task build`/`task test:unit`/`task test:integration` command this phase's plan or executor runs on this machine: prefix with `GOTOOLCHAIN=go1.26.6`.

## No Analog Found

None. All six files/edits in scope have a concrete, git-tracked analog (either themselves, for in-place edits, or a sibling file for new/rewritten content).

## Metadata

**Analog search scope:** `internal/cli/`, `test/integration/`, `cmd/codegraph/`, `.planning/milestones/v0.14.0-phases/03-verb-fold/`, `.planning/phases/01-changie-baseline/`
**Files scanned:** `internal/cli/renamed.go`, `internal/cli/renamed_test.go`, `internal/cli/root.go`, `internal/cli/testdata/cli-reference-allowlist.txt`, `internal/cli/cli_reference_test.go`, `internal/cli/hook_pretooluse_codex_test.go`, `internal/cli/plain_golden_test.go`, `test/integration/renamed_stubs_test.go`, `test/integration/main_test.go`, `cmd/codegraph/main.go`, `.planning/milestones/v0.14.0-phases/03-verb-fold/03-MUTATION-LOG.md`, `.planning/phases/01-changie-baseline/01-MUTATION-LOG.md`
**Pattern extraction date:** 2026-09-26
