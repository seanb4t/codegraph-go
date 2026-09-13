# Phase 12: CLI Reference & Docs Tail - Pattern Map

**Mapped:** 2026-09-13
**Files analyzed:** 12 (new + modified)
**Analogs found:** 12 / 12

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `tools/clidoc/main.go` (new) | service (Go generator program) | file-I/O (regen-in-place / regen-to-buffer) | `tools/graphcluster/main.go` (shape) + `internal/cli/man.go` (the `cobra/doc` call site) | exact |
| `internal/cli/root.go` `NewRootCmd()` (modify — export) | utility (constructor wrapper) | request-response (called, not served) | itself — `newRootCmd()` (root.go:44-63) and its existing `Execute()` wrapper (root.go:67-69) | exact |
| `internal/cli/cli_reference_test.go` (new) | test (accounting guard) | batch (full-tree walk) | deleted `internal/cli/flag_parity_test.go` (`git show 5139e60c^:...`) for the walk skeleton; `internal/upgrade/taskfile_shape_test.go` for doc/config-shape guard-with-counts convention | exact |
| `internal/cli/testdata/cli-reference-allowlist.txt` (new) | config (committed text fixture) | file-I/O (read-only fixture) | no committed `testdata` allowlist precedent exists in-repo — nearest naming/scoping convention is `internal/graphstore/archtest/stdout_transport_allowlist_selftest_test.go`'s allowlist map (`stdoutTransportWriterAllowlist`), for "scoped narrowly, self-tested for over-suppression" discipline only | role-match (no literal file-format precedent — format is new engineering surface) |
| `Taskfile.yml` `docs:cli` + `docs:cli:drift` (new targets) | config/CI (Taskfile) | batch (regen-to-temp, byte-compare) | `proto:gen`/`proto:drift` (Taskfile.yml:355-440) | exact |
| `.github/workflows/ci.yml` drift job (modify) | config/CI | batch | itself — the `task proto:drift` step (ci.yml ~190-196) | exact |
| `docs/CLI-REFERENCE.md` (new, generated) | config (committed generated doc) | file-I/O (write-only from generator) | `web/src/lib/gen/ui_pb.ts` / `*.pb.go` generated-header banner convention | exact |
| `README.md` docs pointer (modify) | config (docs) | — | itself — the existing `docs/RELEASE.md` link paragraph (README.md ~65-79) | exact |
| `docs/RELEASE.md` brew-trust text (modify) | config (docs) | — | itself — the "untrusted tap" section (~518-533) and blockquote (~570-590) | exact |
| `.planning/todos/pending/2026-08-10-brew-trust-…md` (resolve) | planning artifact | — | Phase 7's todo resolution recipe (`07-04-SUMMARY.md`: `git mv` into `.planning/todos/completed/`) | exact |
| `12-MUTATION-LOG.md` | planning artifact | — | `.planning/phases/11-graph-view-community-clustering/11-MUTATION-LOG.md` | exact |
| `12-SECURITY.md` / `12-VALIDATION.md` | planning artifact | — | `.planning/phases/11-graph-view-community-clustering/11-SECURITY.md` / `11-VALIDATION.md` | exact |

## Pattern Assignments

### `tools/clidoc/main.go` (service, file-I/O — new)

**Analog A (package shape):** `tools/graphcluster/main.go:1-14` (package doc comment convention: state the mechanism, what it never does, exit codes):
```go
// Source: tools/graphcluster/main.go:1-13
// Command graphcluster is GRF-09's measurement instrument: it reads every
// bar it judges against from the committed threshold file — never
// carrying its own defaults — opens the pinned corpus's already-indexed
// store READ-ONLY ... and writes exactly one observation file recording
// the verdict verbatim. It never writes the threshold file itself.
//
// Exit codes: 0 = PASS, 1 = FAIL, 2 = refused or error ...
package main
```
`clidoc/main.go` mirrors this: state the mechanism (walks `cli.NewRootCmd()` via `doc.GenMarkdownCustom` into one buffer, never per-command files — D-02), what it never does (never hand-edits, never emits hidden commands — `cobra/doc` already filters via `IsAvailableCommand()`), and its exit codes (0 = wrote/matched, 1 = write error; no PASS/FAIL verdict concept here, unlike graphcluster — this is a generator, not a measurement harness).

**Analog B (the existing `cobra/doc` call site to extend, not duplicate):** `internal/cli/man.go:47-63` (quoted in full):
```go
// Source: internal/cli/man.go:47-63
func newManCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "man <dir>",
		Short:  "Generate man pages for the full command tree",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return fmt.Errorf("create man page directory %s: %w", dir, err)
			}
			header := &doc.GenManHeader{Title: "CODEGRAPH", Section: "1"}
			if err := doc.GenManTree(newRootCmd(), header, dir); err != nil {
				return fmt.Errorf("generate man pages into %s: %w", dir, err)
			}
			return nil
		},
	}
}
```
`clidoc/main.go` uses the same `github.com/spf13/cobra/doc` import (already a dependency — no `go.mod` change) and the same `newRootCmd()`-equivalent entrypoint, but calls `cli.NewRootCmd()` (the new exported wrapper) and `doc.GenMarkdownCustom(cmd, buf, linkHandler)` recursively per D-02's own walk (not `GenMarkdownTree`, which cobra's own doc package would otherwise use to fan out one file per command — `md_docs.go:125-154`, confirmed this session at `$(go env GOPATH)/pkg/mod/github.com/spf13/cobra@v1.10.x/doc/md_docs.go`). `GenMarkdownCustom` itself already skips unavailable/hidden commands and additional-help-topic commands (`md_docs.go:102`: `if !child.IsAvailableCommand() || child.IsAdditionalHelpTopicCommand() { continue }`) — D-05 is satisfied by the library, not by clidoc filtering anything itself.

**Regen-in-place vs. regen-to-temp-and-compare split**, mirrored from the `proto:gen`/`proto:drift` pair (see Taskfile section below): `clidoc` takes a single `-out <path>` flag (default `docs/CLI-REFERENCE.md`) so `task docs:cli` and `task docs:cli:drift` both invoke the SAME binary, differing only in the path they pass (temp file for drift) — one program, two Taskfile call sites, exactly how `buf generate` is one invocation reused by both `proto:gen` and `proto:drift`.

---

### `internal/cli/root.go` `NewRootCmd()` (utility, request-response — modify)

**Analog:** itself — `newRootCmd()` (root.go:44-63) and `Execute()` (root.go:67-69):
```go
// Source: internal/cli/root.go:44, 67-69
func newRootCmd() *cobra.Command { /* ... */ }

func Execute() error {
	return newRootCmd().Execute()
}
```
Per D-02/Claude's Discretion, add an exported wrapper alongside `Execute()`, same file, same doc-comment convention (state what calls it and why it's exported — for `tools/clidoc`, an out-of-package consumer, unlike every existing call site which is intra-package or `cmd/codegraph`):
```go
// NewRootCmd returns a freshly built root command tree. It exists so
// tools/clidoc (an out-of-package doc generator) and this package's own
// tests can obtain the same command tree Execute() runs, without either
// depending on cobra/doc from inside this package.
func NewRootCmd() *cobra.Command {
	return newRootCmd()
}
```
16 existing call sites of `newRootCmd()` (confirmed this session via count) stay untouched — this is additive, not a rename (Claude's Discretion resolved toward the wrapper per D-02's own parenthetical: "touching ~50 call sites in tests argues for the wrapper").

---

### `internal/cli/cli_reference_test.go` (test — new)

**Analog A (the walk skeleton to resurrect and extend):** deleted `internal/cli/flag_parity_test.go` at `5139e60c^` (quoted in full, read this session):
```go
// Source: internal/cli/flag_parity_test.go (deleted at 5139e60c, retrieved
// via `git show 5139e60c^:internal/cli/flag_parity_test.go`)
func TestFlagParityDocCoversRegisteredFlags(t *testing.T) {
	docBytes, err := os.ReadFile(flagParityDocPath)
	if err != nil {
		t.Fatalf("fail-closed: %s must exist and be readable ...", flagParityDocPath, err)
	}
	docText := string(docBytes)
	root := newRootCmd()
	var missing []string
	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		cmd.Flags().VisitAll(func(f *pflag.Flag) {
			if f.Name == "help" { return }
			if !strings.Contains(docText, f.Name) {
				missing = append(missing, cmd.CommandPath()+" --"+f.Name)
			}
		})
		for _, sub := range cmd.Commands() { walk(sub) }
	}
	walk(root)
	if len(missing) > 0 { /* sort + t.Fatalf with count */ }
}
```
D-07 diverges from this skeleton in three ways the planner's `<action>` must call out explicitly: (1) call `cmd.InitDefaultHelpFlag()` on each command before visiting — the old test's blanket `if f.Name == "help"` skip is replaced by making the walk see exactly what `cobra/doc` sees; (2) visit BOTH `cmd.Flags()` and `cmd.PersistentFlags()`, attributing inherited persistent flags to their declaring command only (no re-count on descendants) — the old test visited `cmd.Flags()` alone; (3) walk hidden commands too (the old test implicitly walked everything since nothing was hidden at 08-06's time) and route non-doc-eligible flags to the allowlist branch instead of a single doc-text substring check.

**Analog B (doc/config-shape guard with reported counts convention):** `internal/upgrade/taskfile_shape_test.go:1-45` (structure: named path constants at package scope, `t.Fatalf` naming the exact missing artifact) — mirrors D-09's "walked ≥ 26 commands, ≥ 50 flags inspected" positive-count assertions and D-07's `read_first` fail-closed discipline on `docs/CLI-REFERENCE.md` and the allowlist file.

---

### `internal/cli/testdata/cli-reference-allowlist.txt` (config, file-I/O — new)

**No literal in-repo precedent** for a committed line-oriented allowlist fixture (`rg -l 'allowlist' -g '*_test.go'` returns only `internal/graphstore/archtest/stdout_transport_allowlist_selftest_test.go`, which is a Go map literal, not a text fixture — no `testdata/*allowlist*` file exists anywhere in the tracked tree). This is genuinely new format surface per D-08: one entry per line, `<full command path> [--flag]<TAB><reason>`, e.g.:
```
codegraph man	D-02: hidden command, generator's own doc-comment covers its purpose
```
Borrow only the *discipline* from `stdout_transport_allowlist_selftest_test.go` — an allowlist entry must be provably scoped, not a blanket exemption, hence D-08's "an entry that no longer matches any registered flag or command is itself a failure." Today's allowlist has exactly the one `codegraph man` line (D-08).

---

### `Taskfile.yml` `docs:cli` + `docs:cli:drift` (new targets)

**Analog:** `proto:gen` / `proto:drift` pair (Taskfile.yml:355-440, quoted from this session's read):
```yaml
# Source: Taskfile.yml proto:drift body (355-440), the exact shape to copy
files=$(git ls-files -- '...')
nfiles=0
if [ -n "${files}" ]; then nfiles=$(printf '%s\n' "${files}" | wc -l | tr -d ' '); fi
echo "proto:drift: compared ${nfiles} generated files"
if [ "${nfiles}" -lt 4 ]; then
  echo "::error::proto:drift: enumerated only ${nfiles} ..."
  exit 1
fi
mkdir -p "${scratch}/bin" "${scratch}/gen"
# ... regenerate into ${scratch}, then byte-compare each committed file
# against its fresh counterpart in ${scratch} ...
```
`docs:cli:drift` mirrors this exactly but with a floor of 1 (D-03: "compared 1 generated file"): run `GOTOOLCHAIN=go1.26.6 go run ./tools/clidoc -out "${scratch}/CLI-REFERENCE.md"`, then byte-compare `${scratch}/CLI-REFERENCE.md` against the committed `docs/CLI-REFERENCE.md`, naming the file on mismatch (`diff` or `cmp` invocation matching `proto:drift`'s per-file comparison loop). `docs:cli` is the regen-in-place sibling (`go run ./tools/clidoc` with no `-out` override, or `-out docs/CLI-REFERENCE.md` explicitly) — the same one-binary-two-invocations shape `buf generate` gives `proto:gen`/`proto:drift`.

**GOTOOLCHAIN prefix (D-14):** every `go run`/`go build` line in both new targets is prefixed `GOTOOLCHAIN=go1.26.6`, matching `proto:drift`'s own `GOWORK=off`-prefixed `go build -modfile=...` lines.

---

### `.github/workflows/ci.yml` drift job (modify)

**Analog:** itself — the `task proto:drift` step (verified this session):
```yaml
# Source: .github/workflows/ci.yml (BLD-04 step, immediately preceding
# "Test subprocess integration harness")
- name: Proto codegen drift guard (BLD-04)
  run: task proto:drift
```
`docs:cli:drift` slots as a new step immediately adjacent to this one (D-14/Claude's Discretion: "adjacent to `task proto:drift`"), same two-line shape (`- name: ... / run: task docs:cli:drift`), no new job — same job `proto:drift` already runs in.

---

### `docs/CLI-REFERENCE.md` (config, generated — new)

**Analog (generated-file banner convention):** the "Code generated ... DO NOT EDIT" header stamped into `internal/uiproto/uiv1/ui.pb.go` and `web/src/lib/gen/ui_pb.ts` by the pinned toolchain (per `proto:drift`'s own doc comment: "a header-only difference IS drift ... the committed file must be exactly what the pinned toolchain emits, header included"). `clidoc`'s banner is hand-written (no third-party toolchain stamps it) but follows the same "first line names this as generated, never hand-edited, and names the regenerating command" convention, e.g. `<!-- Generated by tools/clidoc; DO NOT EDIT. Run \`task docs:cli\` to regenerate. -->`.

---

### `README.md` docs pointer (modify)

**Current text to extend, quoted verbatim** (README.md ~65-79, read this session):
```
### Homebrew (macOS)
...
Installs the binary, shell completions, and man pages in one step — see
[`docs/RELEASE.md`](docs/RELEASE.md) for exactly what the cask guarantees, ...
```
D-06 adds one link to `docs/CLI-REFERENCE.md` in this same paragraph family (the existing pointer style: a bracketed markdown link inline in prose, not a new heading/section) — no restructuring, per D-06's own text.

---

### `docs/RELEASE.md` brew-trust text (modify)

**Current text to replace, quoted verbatim** (docs/RELEASE.md ~518-533):
```
**If `brew install codegraph` refuses with "untrusted tap."** ...
Error: Refusing to load cask seanb4t/tap/codegraph from untrusted tap seanb4t/tap.
Run `brew trust --cask seanb4t/tap/codegraph` or `brew trust seanb4t/tap` to trust it.

Run the command the error names, then re-run `brew install codegraph`:

brew trust --tap seanb4t/tap
brew install codegraph
```
Per D-11, this becomes: recommend `brew trust --cask seanb4t/tap/codegraph` specifically (not "the command the error names," which today resolves to the broader tap-wide form in the example block below the quote), trim the verbatim error quote to its narrow form with an ellipsis, and add one sentence naming the control (arbitrary Ruby at install time via the cask's own post-install hook — the same hook `internal/cli/man.go`'s doc comment already describes generating man pages and checking the installed version). The blockquote at ~570-590 (the BREW-01 verification narrative) is a historical record of what was tested, not instructional text — D-11 does not require rewriting it, only the actionable instructions at ~518-533; confirm with the planner whether the blockquote's own inline mention of `brew trust --tap seanb4t/tap` (~582) needs the same narrowing for consistency, since it currently repeats the broader form as something that was run.

---

### `.planning/todos/pending/2026-08-10-brew-trust-…md` (resolve)

**Analog:** Phase 7's todo-resolution recipe, `07-04-SUMMARY.md` (quoted):
```
- .planning/todos/completed/2026-08-09-post-release-verify-...md (moved from .planning/todos/pending/)
- .planning/todos/completed/2026-08-10-tap-app-secret-distinctness-test-...md (moved from .planning/todos/pending/)
```
Mechanism: `git mv .planning/todos/pending/2026-08-10-brew-trust-instructions-....md .planning/todos/completed/` in the same commit as the README/RELEASE.md wording edit (D-12: "resolved through the todo tool when the edit lands"), leaving `.planning/todos/pending/` with the same "one entry" bookkeeping shape Phase 7 left it in.

---

### `12-MUTATION-LOG.md` / `12-SECURITY.md` / `12-VALIDATION.md` (planning artifacts, house format — values only)

**Analog:** `.planning/phases/11-graph-view-community-clustering/11-MUTATION-LOG.md`, `11-SECURITY.md`, `11-VALIDATION.md` (structure only — do not invent headings; see house rule on tool-owned/generated planning files). Families for `12-MUTATION-LOG.md` per D-10: (a) throwaway hidden flag on `ui` → guard fails naming it; (b) delete a line from committed `docs/CLI-REFERENCE.md` → `docs:cli:drift` fails naming the file; (c) bogus allowlist entry → guard fails on the rotten entry. `12-SECURITY.md` rows per D-13: UF-2 docs-as-security-guidance framing (the brew-trust wording IS the security control being documented — get it wrong and a user opts into the broader grant), hidden-flag accounting blind spot (mitigated by D-07's allowlist), generated-file tampering (caught by `docs:cli:drift`); `threats_open: 0` expected per D-13.

## Shared Patterns

### Regen-in-place vs. regen-to-temp-and-byte-compare (one generator binary, two Taskfile call sites)
**Source:** `Taskfile.yml` `proto:gen`/`proto:drift` pair (355-440); `internal/cli/man.go`'s `doc.GenManTree` call site
**Apply to:** `tools/clidoc/main.go`, `Taskfile.yml` `docs:cli`/`docs:cli:drift`

### Positive-count-floor guard, never a vacuous pass
**Source:** `Taskfile.yml` `proto:drift`'s `nfiles < 4` check (rule `84d1gfpywd`); Phase 11's `check:gonum`/community_test.go run-count assertions
**Apply to:** `docs:cli:drift` (floor 1 per D-03), `cli_reference_test.go` (≥26 commands, ≥50 flags per D-09)

### Fail-closed file read for a repo-relative fixture
**Source:** deleted `flag_parity_test.go`'s `os.ReadFile(flagParityDocPath)` fail-closed `t.Fatalf`; `internal/upgrade/taskfile_shape_test.go`'s package-scope path constants
**Apply to:** `cli_reference_test.go` reading `docs/CLI-REFERENCE.md` and `internal/cli/testdata/cli-reference-allowlist.txt`

### `github.com/spf13/cobra/doc` already a dependency — no new require
**Source:** `internal/cli/man.go`'s `doc.GenManTree` import and call
**Apply to:** `tools/clidoc/main.go`'s `doc.GenMarkdownCustom` call — same package, same import path, zero `go.mod` change

### Wording-only doc change, no test, resolved via the todo tool
**Source:** D-12's own text; Phase 7's `git mv` todo-resolution recipe (`07-04-SUMMARY.md`)
**Apply to:** `README.md`, `docs/RELEASE.md` brew-trust wording + `.planning/todos/pending/2026-08-10-brew-trust-…md`

### House-format planning artifacts (values only, never invented structure)
**Source:** `.planning/phases/11-graph-view-community-clustering/11-MUTATION-LOG.md`, `11-SECURITY.md`, `11-VALIDATION.md`
**Apply to:** `12-MUTATION-LOG.md`, `12-SECURITY.md`, `12-VALIDATION.md`

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/cli/testdata/cli-reference-allowlist.txt` | config (committed text fixture) | file-I/O | No committed line-oriented allowlist file exists anywhere in the tracked tree (`rg -l 'allowlist' -g '*_test.go'` finds only a Go map literal in `stdout_transport_allowlist_selftest_test.go`); format (`<path> [--flag]<TAB><reason>`) is this phase's own new engineering surface per D-08/Claude's Discretion, not copied from an analog. |

## Metadata

**Analog search scope:** `internal/cli/`, `tools/graphcluster/`, `tools/corpora/`, `internal/upgrade/`, `internal/graphstore/archtest/`, `Taskfile.yml`, `.github/workflows/ci.yml`, `README.md`, `docs/RELEASE.md`, `.planning/todos/`, `.planning/phases/07-guards-that-cannot-fire/`, `.planning/phases/11-graph-view-community-clustering/`, `$GOPATH/pkg/mod/github.com/spf13/cobra@v1.10.x/doc/md_docs.go`
**Files scanned:** ~18 (`tools/graphcluster/main.go`, `internal/cli/man.go`, `internal/cli/root.go`, deleted `internal/cli/flag_parity_test.go` via `git show 5139e60c^`, `internal/upgrade/taskfile_shape_test.go`, `internal/graphstore/archtest/stdout_transport_allowlist_selftest_test.go`, `Taskfile.yml` (`proto:gen`/`proto:drift`), `.github/workflows/ci.yml` drift job, `README.md` (~55-90), `docs/RELEASE.md` (~505-595), `07-04-SUMMARY.md`, cobra `doc/md_docs.go`)
**Pattern extraction date:** 2026-09-13

## PATTERNS COMPLETE
