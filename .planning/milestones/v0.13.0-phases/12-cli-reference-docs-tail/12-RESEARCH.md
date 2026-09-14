# Phase 12: CLI Reference & Docs Tail - Research

**Researched:** 2026-09-13
**Domain:** Cobra CLI documentation generation (`spf13/cobra/doc`), a small accounting guard over the live command tree, and a two-file wording edit
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- D-01: `docs/CLI-REFERENCE.md` is ONE committed file generated from `newRootCmd()` by `github.com/spf13/cobra/doc` — already a dependency (`codegraph man` uses `doc.GenManTree`), so no new require. It is never hand-edited; every edit to it comes from regeneration. Hand-maintaining ~94 flag rows is exactly the toil a generator exists to remove (maintainer, 2026-09-13).
- D-02: The generator is a small Go program `tools/clidoc/main.go` (sibling of `tools/graphcluster`) that walks the tree in Cobra's own order, calls `doc.GenMarkdownCustom(cmd, w, linkHandler)` per available command into one buffer (single-file output; `GenMarkdownTree`'s one-file-per-command layout is rejected — 30+ files for a reference nobody browses that way), and writes `docs/CLI-REFERENCE.md` with a short generated-file banner. `internal/cli` exports the constructor for it (`NewRootCmd()` wrapper around `newRootCmd()`, or the rename — planner's call); the `tools/` package may import `internal/*` within this module (precedent `tools/graphcluster`).
- D-03: `task docs:cli` regenerates in place; `task docs:cli:drift` regenerates into a TEMPORARY file and byte-compares it against the committed one (the `proto:drift` shape: reports `compared 1 generated file` before comparing, names the file on mismatch, non-zero exit) and is wired into `ci.yml` in the same job that runs `task proto:drift` (ci.yml:196). A stale committed reference is the one real failure mode this phase guards against, and it fires on every flag addition.
- D-04: Command prose is carried by `cmd.Short`/`cmd.Long` in the Cobra tree — the single source of truth — never by a separate hand-written file stitched in. Nine commands carry a `Long` today; the planner adds a `Long` only where a command's current `--help` does not convey its argument semantics or precedence rules (e.g. `ui` already explains `--editor-url` / `CODEGRAPH_EDITOR_URL` / `--no-editor-url` precedence in its `Long`, so the reference documents this milestone's new flag automatically). Do not gold-plate: `Short` alone is acceptable for a command whose flags are self-explanatory.
- D-05: Hidden commands (`man`, `Hidden: true` per v0.5.0 D-02 — it exists for the Homebrew cask post-install hook) stay OUT of the generated reference, as `cobra/doc` already does (`md_docs.go:135` skips `!IsAvailableCommand()`); hidden flags likewise. Hidden means hidden from user docs; the guard (below) is what keeps them from being *silently* hidden.
- D-06: README.md gains one link to `docs/CLI-REFERENCE.md` in its existing docs pointers (the paragraph that already points at `docs/RELEASE.md`); no other README restructuring.
- D-07: `internal/cli/cli_reference_test.go` → `TestEveryRegisteredFlagIsAccountedFor` (runs in `task test:unit`, therefore in CI with no extra wiring). It builds the root command, calls `InitDefaultHelpFlag()` on each command exactly as `cobra/doc` does (`md_docs.go:59`) so the walk and the generator see the same flag set, walks EVERY command recursively — hidden commands included — and visits `cmd.Flags()` and `cmd.PersistentFlags()` with `VisitAll` (which visits hidden and deprecated flags), de-duplicating by command path + flag name. Inherited persistent flags are attributed to the command that declares them, not re-counted on every descendant.
- D-08: One rule, no exemption list: a flag on an available (non-hidden) command that is itself neither hidden nor deprecated must appear as `--name` in the generated `docs/CLI-REFERENCE.md`; every other flag — hidden, deprecated, or belonging to a hidden command — must appear in the committed allowlist `internal/cli/testdata/cli-reference-allowlist.txt` (one entry per line: `<full command path> [--flag]<TAB><reason>`; a whole hidden command may be listed once by path to cover all its flags). An entry in the allowlist that no longer matches any registered flag or command is itself a failure (allowlists rot upward too). Today the allowlist has exactly one entry: `codegraph man` with D-02's reason.
- D-09: Positive assertions (rule `84d1gfpywd`): the test `t.Logf`s and asserts commands walked ≥ 26, flags inspected ≥ 50, and reports how many were accepted via the reference vs the allowlist; it fails closed if `docs/CLI-REFERENCE.md` or the allowlist is unreadable. It does NOT scope-match flags to sections (the drift gate already guarantees the doc's content is exactly the generator's) and it does NOT test `cobra/doc`'s behaviour — only this project's artefacts.
- D-10: RED families for `12-MUTATION-LOG.md`, each applied for real in the working tree and reverted byte-clean: (a) register a throwaway hidden flag on `ui` (`Bool("zz-throwaway", …)` + `MarkHidden`) → the guard fails naming `codegraph ui --zz-throwaway` as unaccounted; (b) delete one line from the committed `docs/CLI-REFERENCE.md` → `task docs:cli:drift` fails naming the file; (c) add a bogus allowlist entry for a flag that does not exist → the guard fails on the rotten entry. No family exercises `cobra/doc` itself.
- D-11: `README.md` (the Homebrew paragraph, ~line 71–83) and `docs/RELEASE.md` (the "untrusted tap" instructions at ~518–533 and the blockquote at ~579–585) recommend `brew trust --cask seanb4t/tap/codegraph` as THE command to run, followed by one sentence naming the control: Homebrew refuses because a third-party cask runs arbitrary Ruby on your machine at install time — this cask's own post-install hook does (it generates man pages and checks the installed version) — so `brew trust` is a security control you are opting out of, and the `--cask` form limits that opt-out to this one cask. The tap-wide grant is mentioned as existing and not recommended by this project, without spelling its command (nothing to copy-paste). The verbatim Homebrew error quote is trimmed to the narrow form with an ellipsis so the docs never carry the broad form as an instruction. **Research correction: README.md contains no `brew trust` text today (verified this session) — the substantive DOCS-07 edit is entirely inside `docs/RELEASE.md`; see Sources.**
- D-12: No guard, no test, no docs-grep for DOCS-07 — it is a wording change verified by reading the diff (ROADMAP.md:121 already says so; criterion 3 was reworded to match). The pending todo `2026-08-10-brew-trust-instructions-…` is resolved through the todo tool when the edit lands.
- D-13: `12-VALIDATION.md` (plan-phase seeds it), `12-MUTATION-LOG.md` (families a–c), `12-SECURITY.md` (rows: UF-2 docs-as-security-guidance framing; hidden-flag accounting blind spot; generated-file tampering caught by drift; `threats_open: 0` expected). Small phase: 2–3 plans, sequential (`use_worktrees=false`).
- D-14: Every `go` invocation prefixed `GOTOOLCHAIN=go1.26.6`; positive-count guards (`rg -o … | wc -l`, never `-c`); the drift target reports its compared-file count BEFORE comparing (proto:drift precedent, Taskfile.yml:436).

### Claude's Discretion

- Whether `NewRootCmd` is an exported wrapper or a rename of `newRootCmd` (touching ~50 call sites in tests argues for the wrapper). **Research finding: actual call-site count is much smaller — only 3 real (non-comment) test call sites (`internal/cli/cli_test.go:62`, `internal/cli/man_test.go:102`, `internal/cli/telemetry_test.go:49`) plus `root.go`'s own definition/`Execute()`/`man.go:61` — the wrapper-vs-rename choice is low-cost either way; this research recommends the wrapper (`func NewRootCmd() *cobra.Command { return newRootCmd() }`) since it needs zero call-site changes.**
- Exact `linkHandler` / banner text and whether `GenMarkdownCustom`'s per-command `### SEE ALSO` blocks are kept (they are harmless in a single file; keep unless they double the file).
- Which thin `Short`-only commands get a `Long`, if any.
- Allowlist file location (`internal/cli/testdata/…` preferred so the test's `read_first` is local) and line format details.
- Where in `ci.yml`'s drift job `docs:cli:drift` slots (adjacent to `task proto:drift`).

### Deferred Ideas (OUT OF SCOPE)

- Wiring Phase 11's `check:gonum` and `check:no-force-layout` Taskfile targets into `ci.yml` — recorded in 11-SECURITY.md and STATE.md as a follow-up; not docs-tail work.
- Reconciling the four stale rows in STATE.md's Pending Todos table against `.planning/todos/pending/` — noted since Phase 7; a deliberate pass, not this phase.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| DOCS-05 | A committed `docs/CLI-REFERENCE.md`, generated from the live Cobra tree by `cobra/doc` and kept current by a regenerate-and-diff gate, documents every non-hidden command and flag, replacing what `docs/FLAG-PARITY.md` used to carry | Architecture Patterns (Patterns 1-3), Code Examples (`tools/clidoc/main.go` skeleton), Common Pitfalls 1-4, Validation Architecture (`task docs:cli:drift`) |
| DOCS-06 | A guard walks the live Cobra tree — including hidden, inherited persistent and deprecated flags — and fails when any registered flag is unaccounted for (in the generated reference, or in a committed hidden/deprecated allowlist with a reason); demonstrated RED by registering a throwaway hidden flag | Architecture Patterns (Pattern 3), Common Pitfalls 3 and 5, Code Examples (`cli_reference_test.go` skeleton), Security Domain (allowlist rot row) |
| DOCS-07 | The brew-trust instructions recommend the narrow grant with security framing rather than the broader `--tap` grant | Sources (README.md/docs/RELEASE.md exact line reads — corrects the CONTEXT.md file list), Security Domain (UF-2 row) |
</phase_requirements>

## Summary

This phase generates `docs/CLI-REFERENCE.md` from the live `internal/cli` Cobra tree using `github.com/spf13/cobra/doc` (already a direct dependency — `internal/cli/man.go` imports it for `doc.GenManTree`), guards the generator's one blind spot (hidden/deprecated flags are invisible by design) with a small walk-based accounting test, and rewords two `brew trust` instructions in `docs/RELEASE.md`. All three requirements are narrow and the maintainer has already resolved the architectural question ("are we reimplementing cobra doc?") by ruling that generation IS the source of truth — this research verifies the exact API contract needed to call it correctly, not cobra's own behavior.

Two verified findings materially change what the planner must do beyond the CONTEXT.md text. **First**, `cobra.Command.DisableAutoGenTag` is a plain bool field with no automatic parent inheritance except through `GenMarkdownCustom`'s own internal `VisitParents` walk (which fires because `hasSeeAlso(cmd)` is true for every non-root command) — setting it `true` on the root command object before generation is sufficient to suppress the date-stamped `###### Auto generated by spf13/cobra on <date>` line on every command in the tree, and this step is **mandatory** for a byte-stable committed file (confirmed by reading `md_docs.go` directly, not assumed). **Second**, and more consequential: the `completion` command (with its `bash`/`zsh`/`fish`/`powershell` children, each carrying a `--no-descriptions` flag) is **not present on a freshly built `newRootCmd()` tree at all** — Cobra only registers it inside `Command.ExecuteC()` via `InitDefaultCompletionCmd()`, which a plain tree-walking generator or guard test never triggers. Verified live via a temporary probe test this session: a bare `newRootCmd()` walk finds 26 commands; calling `root.InitDefaultCompletionCmd()` first finds 36 (10 more: `completion` + 4 shells were the only real change — no phantom `__complete` command ever appears outside `ExecuteC()`, since that command is added by an unexported `initCompleteCmd` cobra never calls from a doc-generation path). Both `tools/clidoc/main.go` and `internal/cli/cli_reference_test.go` **must** call `root.InitDefaultCompletionCmd()` once, immediately after building the tree, or the committed reference and the guard will silently agree on an incomplete tree — exactly the vacuous-guard shape this milestone exists to eliminate.

**Primary recommendation:** Build `tools/clidoc/main.go` as a small program that constructs `internal_cli.NewRootCmd()`, calls `root.InitDefaultCompletionCmd()` once, walks `cmd.Commands()` recursively (Cobra's own alphabetical order — `EnableCommandSorting` defaults true and is never overridden in `root.go`), skips non-available commands via `IsAvailableCommand()`, calls `doc.GenMarkdownCustom(cmd, &buf, linkHandler)` per available command into one buffer with `linkHandler` rewriting cobra's `name.md` targets into single-file GFM anchors (`#name-with-hyphens`), and sets `root.DisableAutoGenTag = true` before generating anything. Mirror the exact same tree construction (`NewRootCmd()` + `InitDefaultCompletionCmd()` + `InitDefaultHelpFlag()` per node) in `internal/cli/cli_reference_test.go`'s guard so both walks see an identical command/flag universe.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CLI reference generation (DOCS-05) | Build tooling (`tools/clidoc`, a `main` package under the module) | — | A one-shot Go program invoked via `task docs:cli` / `task docs:cli:drift`, never part of the shipped binary |
| Command-tree constructor export (D-02) | `internal/cli` (library package) | — | `NewRootCmd()` is a pure library API; `tools/clidoc` and `cmd/codegraph/main.go` both consume it |
| Accounting guard (DOCS-06) | `internal/cli` test suite | Build tooling (CI, via `task test:unit`) | The guard is a `_test.go` file in the same package as the tree it inspects — no wire-layer or CLI-runtime involvement |
| Drift gate (`task docs:cli:drift`) | Build tooling (Taskfile + CI job) | — | Same tier as `proto:drift`/`web:drift` — regenerate-into-temp, byte-compare, never mutate the working tree |
| brew-trust wording (DOCS-07) | Documentation (`README.md`, `docs/RELEASE.md`) | — | Prose-only change; no runtime tier touched |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/spf13/cobra` | v1.10.2 (pinned in `go.mod:17`) `[VERIFIED: go.mod:17]` | CLI framework already in use for every `codegraph` command | Already the whole CLI's foundation; no alternative considered |
| `github.com/spf13/cobra/doc` | same module, v1.10.2 | Markdown reference generation (`GenMarkdownCustom`) | Already imported by `internal/cli/man.go:8` for `doc.GenManTree` (D-01) — zero new `go.mod` entries |
| `github.com/spf13/pflag` | v1.0.10 (resolved via MVS; cobra's own `go.mod` requires v1.0.9) `[VERIFIED: go.sum, cobra@v1.10.2/go.mod]` | Flag registration/rendering that `cobra/doc` and the guard both read (`VisitAll`, `Hidden`, `Deprecated` fields) | Transitive dependency of cobra; no direct action needed |

No new `go.mod` requires for this phase (D-01). `tools/clidoc` is a plain `main` package in the existing module — same pattern as `tools/graphcluster`, `tools/corpora`, `tools/bench/*` (all already excluded from nothing: `go list ./...` reaches every `tools/*` directory, confirmed by running it this session — only `internal/daemon` is filtered out of `task test:unit`, and `tools/` packages are neither excluded from `task test:unit` nor from `go vet ./...`/`task lint:go`).

### Supporting
None — this phase adds no supporting libraries.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `doc.GenMarkdownCustom` walked manually into one buffer | `doc.GenMarkdownTree`/`GenMarkdownTreeCustom` | Rejected explicitly (D-02): one file per command (30+ files) for a reference nobody browses that way; also loses control over the single-file anchor rewrite |
| Hand-maintained `docs/CLI-REFERENCE.md` | Cobra-generated reference | Rejected at Phase 12 discuss time — this is the whole point of the phase reset; hand-maintaining ~94–114 flag rows is exactly the toil a generator removes |
| A hand-rolled accounting scheme scoping flags to doc sections | The existing drift gate (`task docs:cli:drift`) already guarantees content matches the generator | D-09 explicitly rejects section-scoped matching — the guard only proves nothing is *silently* missing, not that formatting is correct |

**Installation:**
No new packages. `tools/clidoc/main.go` imports `github.com/spf13/cobra`, `github.com/spf13/cobra/doc`, and `github.com/seanb4t/codegraph-go/internal/cli` — all already resolved in `go.mod`.

**Version verification:** `github.com/spf13/cobra v1.10.2` confirmed present in `go.mod:17` (read directly). `github.com/spf13/pflag` is transitive at v1.0.10 (`go.sum`), one minor above cobra's own declared `v1.0.9` floor (`cobra@v1.10.2/go.mod`) — both read directly from the module cache this session, not assumed from training data.

## Package Legitimacy Audit

Not applicable — this phase installs zero new external packages. `github.com/spf13/cobra` and its `doc` subpackage are an existing, already-audited direct dependency (present since the project's `man` command shipped); `github.com/spf13/pflag` is an existing transitive dependency. No `go.mod`/`go.sum` change is expected from this phase's work.

## Architecture Patterns

### System Architecture Diagram

```
                    ┌─────────────────────────┐
                    │  internal/cli/root.go    │
                    │  NewRootCmd() (exported) │──────► cmd/codegraph/main.go
                    │  wraps newRootCmd()      │        (real binary execution,
                    └───────────┬───────────────┘        unchanged behavior)
                                │ builds full Cobra tree
                                │ (25 registered cmds + root;
                                │  +5 completion family after
                                │  InitDefaultCompletionCmd())
                 ┌──────────────┴───────────────┐
                 │                               │
                 ▼                               ▼
   ┌─────────────────────────┐     ┌──────────────────────────────┐
   │ tools/clidoc/main.go     │     │ internal/cli/                │
   │ (generator, `go run`)    │     │   cli_reference_test.go       │
   │                          │     │ (accounting guard, `go test`) │
   │ 1. NewRootCmd()          │     │ 1. NewRootCmd()                │
   │ 2. InitDefaultCompletion │     │ 2. InitDefaultCompletionCmd()  │
   │    Cmd()                 │     │ 3. recursive walk: every cmd   │
   │ 3. root.DisableAutoGen   │     │    (hidden included),          │
   │    Tag = true            │     │    InitDefaultHelpFlag() each  │
   │ 4. walk cmd.Commands()   │     │ 4. Flags()+PersistentFlags()   │
   │    (skip !IsAvailable    │     │    .VisitAll per node          │
   │    Command())            │     │ 5. classify each flag:         │
   │ 5. doc.GenMarkdownCustom │     │    available+non-hidden+non-   │
   │    (cmd, buf, linkHandler)│    │    deprecated → must be in     │
   │ 6. write docs/           │     │    generated reference;        │
   │    CLI-REFERENCE.md      │     │    else → must be in allowlist │
   └───────────┬──────────────┘     └───────────────┬────────────────┘
               │ commits generated file              │ asserts counts (≥26 cmds,
               ▼                                      │ ≥50 flags — D-09) and
   docs/CLI-REFERENCE.md (committed, never            │ fails on any unaccounted
   hand-edited)                                       │ flag or rotten allowlist
               │                                       │ entry
               ▼                                       ▼
   task docs:cli:drift (Taskfile + ci.yml, the    internal/cli/testdata/
   test job right after `task proto:drift`,       cli-reference-allowlist.txt
   proto:drift's regen-into-temp + byte-compare   (committed; today: one line
   shape) — fails the build if the committed      for `codegraph man`, D-02's
   file is stale relative to a fresh regen        reason)
```

### Recommended Project Structure
```
docs/
└── CLI-REFERENCE.md          # generated, never hand-edited (D-01)
internal/cli/
├── root.go                   # add NewRootCmd() exported wrapper
├── cli_reference_test.go     # new: DOCS-06 accounting guard
└── testdata/
    └── cli-reference-allowlist.txt   # new: hidden/deprecated flag allowlist
tools/
└── clidoc/
    ├── main.go                # new: the generator (sibling of tools/graphcluster)
    └── main_test.go           # optional: smoke test, following graphcluster's own _test.go precedent
```

### Pattern 1: Single-file assembly with anchor-rewriting linkHandler
**What:** Walk the live tree in Cobra's own order, call `doc.GenMarkdownCustom` per available command into ONE shared buffer, with a `linkHandler` that turns cobra's default `command_name.md` cross-reference target into a same-file GitHub-flavored-markdown anchor (`#command-name`) instead of a separate-file link.
**When to use:** Any time `GenMarkdownTree`'s one-file-per-command layout is undesirable (D-02's explicit rejection reason: 30+ files for a reference nobody browses that way).
**Example:**
```go
// Source: github.com/spf13/cobra/doc md_docs.go v1.10.2 (GenMarkdownCustom, GenMarkdownTreeCustom)
// read directly from the module cache this session — not GenMarkdownTree's own
// linkHandler (which cobra ships as `func(s string) string { return "" }` for the
// zero-arg convenience wrapper); this project needs a real transform.
func linkHandler(link string) string {
    name := strings.TrimSuffix(link, ".md")
    anchor := strings.ReplaceAll(name, "_", "-") // cobra's own name.md already
                                                   // replaced spaces with "_"
                                                   // (md_docs.go: link = strings.
                                                   // ReplaceAll(link, " ", "_"))
    return "#" + anchor
}
```
`cmd.CommandPath()` values in this tree (e.g. `"codegraph ui"`) are already lowercase, so no extra case-folding is needed to match GFM's auto-generated heading anchors.

### Pattern 2: Recursive walk that mirrors `IsAvailableCommand()`/`IsAdditionalHelpTopicCommand()`
**What:** Both the generator and the guard must skip exactly what `cobra/doc`'s own `GenMarkdownTreeCustom` skips when descending: `if !c.IsAvailableCommand() || c.IsAdditionalHelpTopicCommand() { continue }` (`md_docs.go:135`, read directly). `IsAvailableCommand()` returns `false` for `Hidden`, `Deprecated`, or the auto-injected `help` subcommand; `IsAdditionalHelpTopicCommand()` further filters non-runnable pure-help-topic commands.
**When to use:** Anywhere the tree is walked to decide what's "in" the generated doc, so the generator's inclusion set and the guard's "must appear in the doc" set are provably the same predicate.
**Example:**
```go
// Source: github.com/spf13/cobra command.go v1.10.2 (IsAvailableCommand,
// IsAdditionalHelpTopicCommand) — read directly, not paraphrased.
func isDocumented(c *cobra.Command) bool {
    return c.IsAvailableCommand() && !c.IsAdditionalHelpTopicCommand()
}
```

### Pattern 3: Completing the tree before walking it
**What:** `newRootCmd()` alone does NOT register the `completion` command family. `Command.ExecuteC()` calls `c.InitDefaultCompletionCmd(args...)` (`command.go:1113`, read directly) as one of its own setup steps — a plain, unexecuted tree never gets this call. Both the generator and the guard must call it explicitly.
**When to use:** Any tool that inspects a Cobra tree without ever calling `Execute()`.
**Example:**
```go
// Source: github.com/spf13/cobra command.go v1.10.2 (ExecuteC, InitDefaultCompletionCmd)
// — verified this session by a temporary probe test (see Pitfall 2 below): a bare
// newRootCmd() walk found 26 commands; adding this one call found 36 (root +
// completion + bash/zsh/fish/powershell, no phantom __complete command).
root := cli.NewRootCmd()
root.InitDefaultCompletionCmd() // args are irrelevant here — codegraph has 24
                                  // real registered subcommands, so
                                  // InitDefaultCompletionCmd's own "only add
                                  // completion if it's the sole subcommand and
                                  // being invoked" special case never triggers
                                  // (completions.go: hasSubCommands == true)
root.DisableAutoGenTag = true
```

### Anti-Patterns to Avoid
- **Walking `cmd.Commands()` before completing the tree:** produces a reference and a guard that both silently omit `completion bash|zsh|fish|powershell` — a real user-facing command family invisible to both artifacts. This is precisely the "guard that cannot fire" shape v0.13.0 exists to eliminate; it must not be reintroduced here.
- **Forgetting `root.DisableAutoGenTag = true`:** every regeneration embeds today's date (`###### Auto generated by spf13/cobra on <date>`), so `task docs:cli:drift` would fail on every calendar day even with zero flag changes — a permanently-red gate that trains developers to ignore it.
- **Re-scoping the guard to check doc *sections*, not just flag presence:** D-09 explicitly rejects this — the drift gate already guarantees exact content match; the guard's only job is "is every registered flag accounted for somewhere," never formatting.
- **Treating a deprecated-but-visible flag's appearance in the generated Markdown as sufficient:** pflag's `FlagUsagesWrapped` does NOT skip `Deprecated` flags (only `Hidden` ones) — a deprecated flag DOES render in the doc text with a `(DEPRECATED: ...)` suffix. D-08's rule is independent of this: deprecated flags belong in the allowlist regardless of whether cobra also prints them, so the guard must not use "found deprecated flag's name in the doc text" as a pass condition for a deprecated flag — only the allowlist counts.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| CLI reference document | A hand-maintained `docs/CLI-REFERENCE.md` (the pre-2026-09-13 plan) | `doc.GenMarkdownCustom` walked per-command | Maintainer ruling at Phase 12 discuss: hand-maintaining ~94–114 flag rows is exactly the toil a generator exists to remove |
| Flag-usage text rendering (defaults, shorthand, wrapping) | A custom flag-formatter | `pflag.FlagSet.FlagUsagesWrapped` (invoked transitively via `cmd.Flags().PrintDefaults()` inside `printOptions`) | Already handles shorthand, `NoOptDefVal`, default-value quoting, and `(DEPRECATED: ...)` suffixes correctly; reimplementing it duplicates a well-tested library |
| Command-tree traversal order | A custom sort | `cmd.Commands()` (Cobra sorts alphabetically by default; `EnableCommandSorting` is never disabled in `root.go`) | Guarantees the generator's command order matches what `codegraph --help` itself shows, for free |

**Key insight:** Every piece of rendering logic this phase would otherwise hand-roll (flag formatting, command ordering, hidden/deprecated suppression rules) already exists correctly in `cobra`/`pflag` — the only work this project owns is (a) the single-file assembly/anchor rewrite, (b) the completion-cmd initialization step neither library does automatically outside `Execute()`, and (c) the accounting guard for cobra/doc's one deliberate blind spot (hidden flags).

## Common Pitfalls

### Pitfall 1: Non-deterministic auto-gen-tag date breaks the drift gate
**What goes wrong:** `task docs:cli:drift` fails every single day, even with zero real changes, because `GenMarkdownCustom` appends `###### Auto generated by spf13/cobra on <today's date>` unless `cmd.DisableAutoGenTag` is true (`md_docs.go:111-113`, read directly).
**Why it happens:** The flag defaults to `false` and is NOT set anywhere in `internal/cli` today (confirmed by search — zero matches for `DisableAutoGenTag` in the tree before this phase).
**How to avoid:** Set `root.DisableAutoGenTag = true` once, before generating any command's markdown. This is sufficient for the WHOLE tree: `GenMarkdownCustom`'s own `hasSeeAlso(cmd)` branch (true whenever `cmd.HasParent()`, per `util.go:24-27`) walks `cmd.VisitParents` and copies `DisableAutoGenTag` down from any ancestor that has it set (`md_docs.go:91-93`) — so only the root needs the explicit assignment; every descendant inherits it via that internal propagation at generation time, not via any field default.
**Warning signs:** `task docs:cli:drift` failing on a PR that touched no CLI code, with the reported diff being exactly one date line per command section.

### Pitfall 2: The `completion` command family is invisible unless explicitly initialized
**What goes wrong:** A generator or guard built against a bare `newRootCmd()`/`NewRootCmd()` never sees `codegraph completion`, `completion bash|zsh|fish|powershell`, or their `--no-descriptions` flags — even though a real user running `codegraph completion --help` sees all of it, because `cmd/codegraph/main.go`'s call path goes through `Execute()` → `ExecuteC()`, which calls `InitDefaultCompletionCmd()` as one of its own setup steps (`command.go:1113`).
**Why it happens:** `InitDefaultCompletionCmd` and the hidden `__complete`/`__completeNoDesc` debug command (`initCompleteCmd`, unexported, only reachable from `ExecuteC()`) are both lazy, Execute-time-only mutations of the tree, not construction-time additions from `newRootCmd()`.
**How to avoid:** Call `root.InitDefaultCompletionCmd()` (no args needed — see Pattern 3) immediately after building the tree, in BOTH `tools/clidoc/main.go` and `internal/cli/cli_reference_test.go`, before any walk begins.
**Warning signs:** Verified this session with a temporary probe test (`internal/cli`, removed after use, tree left clean): `newRootCmd()` alone → 26 commands walked; `+ root.InitDefaultCompletionCmd()` → 36 commands walked, adding exactly `completion` + `bash`/`zsh`/`fish`/`powershell` (each contributing one `--no-descriptions` flag) and nothing else — no `__complete` command ever appeared, confirming it truly requires the unexported `initCompleteCmd`/`ExecuteC()` path and is a non-issue for this phase.

### Pitfall 3: `cmd.Flags()` on an unexecuted tree does not include the auto `--help` flag until `InitDefaultHelpFlag()` runs
**What goes wrong:** A guard that visits `cmd.Flags().VisitAll(...)` on a freshly built tree without calling `cmd.InitDefaultHelpFlag()` per command first will see zero `--help` flags, while the generator (which calls `InitDefaultHelpFlag()` internally at the top of `GenMarkdownCustom`, `md_docs.go:59`) DOES print `-h, --help` for every available command — a spurious mismatch that has nothing to do with a real accounting gap.
**Why it happens:** `InitDefaultHelpFlag()` lazily adds the flag to `c.Flags()` only when called (`command.go`, read directly): `if c.Flags().Lookup(helpFlagName) == nil { c.Flags().BoolP(helpFlagName, "h", false, usage) }`.
**How to avoid:** Call `cmd.InitDefaultHelpFlag()` on every node in the guard's own walk, exactly mirroring what `GenMarkdownCustom` does per command (D-07's own citation of `md_docs.go:59` is the correct fix, confirmed by reading the source).
**Warning signs:** The guard reporting `--help` as "unaccounted" on every single command the moment it's written naively.

### Pitfall 4: Assuming flag/command output order is nondeterministic
**What goes wrong:** A developer might assume Go map iteration randomization could make the generated Markdown byte-unstable across runs, and over-engineer explicit sorting into `tools/clidoc`.
**Why it happens:** pflag's internal flag storage is a map, and naive intuition says map order is random.
**How to avoid:** No extra sorting is needed. `pflag.NewFlagSet` sets `SortFlags: true` by default (`flag.go:1272`, read directly), so `VisitAll`/`FlagUsagesWrapped` always iterate flags in alphabetical name order regardless of registration order. Command order is likewise handled by Cobra itself: `EnableCommandSorting` defaults `true` and `root.go` never sets it `false`, so `cmd.Commands()` sorts alphabetically by name on every call (cached after the first sort per `command.go`'s `commandsAreSorted` flag). The committed reference's byte-stability rests entirely on (a) this default sort behavior and (b) Pitfall 1's fix — not on any custom sorting this phase would otherwise be tempted to add.
**Warning signs:** None expected if the two upstream defaults are left untouched; a regression here would only appear if a future change explicitly sets `SortFlags = false` on a `FlagSet` or `cobra.EnableCommandSorting = false` globally — neither of which this phase should ever do.

### Pitfall 5: `mergePersistentFlags` side effects when using `NonInheritedFlags()`/`InheritedFlags()` for accounting instead of raw `Flags()`/`PersistentFlags()`
**What goes wrong:** Calling `cmd.NonInheritedFlags()`/`cmd.InheritedFlags()` (what `printOptions` uses internally) triggers `mergePersistentFlags()`, which folds ALL ancestor persistent flags into `cmd.Flags()` as a side effect (`command.go: mergePersistentFlags`, read directly) — if the guard used this merged view for its own recursive accounting instead of `cmd.Flags()` + `cmd.PersistentFlags()` per node, a persistent flag would be double-counted once at its declaring command and again at every descendant.
**Why it happens:** `Flags()`'s doc comment ("the complete FlagSet that applies to this command") is easy to misread as "this command's own flags."
**How to avoid:** Per D-07's own design, visit `cmd.Flags()` (local) and `cmd.PersistentFlags()` (local persistent) directly at each node in the recursive walk — never `InheritedFlags()`/`NonInheritedFlags()` — so each flag is attributed exactly once, to the command that declares it. This repo currently has zero `PersistentFlags()` calls anywhere in `internal/cli` (confirmed by search), so this distinction is currently moot in practice but is exactly the mechanism the guard's design already correctly avoids per D-07's phrasing ("attributed to the command that declares them, not re-counted on every descendant").
**Warning signs:** A future `PersistentFlags()` addition making the guard's reported flag count jump by more than the number of new flags actually added.

## Code Examples

### tools/clidoc/main.go skeleton
```go
// Source: pattern synthesized from github.com/spf13/cobra/doc md_docs.go
// (GenMarkdownCustom, GenMarkdownTreeCustom's own walk-and-skip idiom) plus
// tools/graphcluster/main.go's doc-comment convention (Phase 11 precedent,
// read directly) — read this session, not reused verbatim from either.
//
// Command clidoc is DOCS-05's generator: it builds the full codegraph Cobra
// tree via internal/cli.NewRootCmd(), completes it with
// InitDefaultCompletionCmd() (the completion command family is otherwise
// invisible outside Command.ExecuteC()), walks every available command in
// Cobra's own alphabetical order, and writes one assembled
// docs/CLI-REFERENCE.md via doc.GenMarkdownCustom per command. It never
// writes more than one file and never touches anything outside the -out
// path.
package main

import (
    "bytes"
    "flag"
    "fmt"
    "os"
    "strings"

    "github.com/spf13/cobra"
    "github.com/spf13/cobra/doc"

    internalcli "github.com/seanb4t/codegraph-go/internal/cli"
)

func main() {
    out := flag.String("out", "docs/CLI-REFERENCE.md", "output path")
    flag.Parse()

    root := internalcli.NewRootCmd()
    root.InitDefaultCompletionCmd() // Pitfall 2 — mandatory
    root.DisableAutoGenTag = true   // Pitfall 1 — mandatory

    var buf bytes.Buffer
    buf.WriteString("<!-- generated by tools/clidoc — do not hand-edit; run `task docs:cli` -->\n\n")

    var walk func(cmd *cobra.Command) error
    walk = func(cmd *cobra.Command) error {
        if cmd.IsAvailableCommand() && !cmd.IsAdditionalHelpTopicCommand() {
            if err := doc.GenMarkdownCustom(cmd, &buf, linkHandler); err != nil {
                return fmt.Errorf("generate markdown for %s: %w", cmd.CommandPath(), err)
            }
        }
        for _, sub := range cmd.Commands() {
            if err := walk(sub); err != nil {
                return err
            }
        }
        return nil
    }
    if err := walk(root); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }

    if err := os.WriteFile(*out, buf.Bytes(), 0o644); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func linkHandler(link string) string {
    name := strings.TrimSuffix(link, ".md")
    return "#" + strings.ReplaceAll(name, "_", "-")
}
```
Note: `walk` calls `GenMarkdownCustom` on `root` itself too (root passes `IsAvailableCommand()` when it has runnable/available subcommands) — this matches what a real `codegraph --help` invocation implies exists at the top of the tree, and is consistent with `GenMarkdownTreeCustom`'s own recursive shape (it always generates a page for the command it's called on, in addition to recursing into children).

### internal/cli/cli_reference_test.go — accounting shape (skeleton, not exhaustive)
```go
// Source: pattern combining the deleted internal/cli/flag_parity_test.go
// walk shape (git show 5139e60c^:internal/cli/flag_parity_test.go, read
// directly this session) with D-07's InitDefaultHelpFlag/
// InitDefaultCompletionCmd additions and D-08's allowlist branch.
package cli

const cliReferenceDocPath = "../../docs/CLI-REFERENCE.md"
const cliReferenceAllowlistPath = "testdata/cli-reference-allowlist.txt"

func TestEveryRegisteredFlagIsAccountedFor(t *testing.T) {
    docBytes, err := os.ReadFile(cliReferenceDocPath)
    if err != nil {
        t.Fatalf("fail-closed: %s must exist and be readable: %v", cliReferenceDocPath, err)
    }
    allowBytes, err := os.ReadFile(cliReferenceAllowlistPath)
    if err != nil {
        t.Fatalf("fail-closed: %s must exist and be readable: %v", cliReferenceAllowlistPath, err)
    }
    // ... parse allowBytes into a set of "<command path> [--flag]" entries ...

    root := newRootCmd()
    root.InitDefaultCompletionCmd() // must match tools/clidoc exactly

    var cmdCount, flagCount int
    var walk func(cmd *cobra.Command)
    walk = func(cmd *cobra.Command) {
        cmd.InitDefaultHelpFlag() // must match doc.GenMarkdownCustom exactly
        cmdCount++
        visit := func(f *pflag.Flag) {
            flagCount++
            // classify: hidden(cmd) || hidden(f) || deprecated(f) → must be
            // in allowlist; else → must appear as "--"+f.Name in docBytes
        }
        cmd.Flags().VisitAll(visit)
        cmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
            if cmd.Flags().Lookup(f.Name) == nil { // avoid double count
                visit(f)
            }
        })
        for _, sub := range cmd.Commands() {
            walk(sub)
        }
    }
    walk(root)

    t.Logf("walked %d commands, inspected %d flags", cmdCount, flagCount)
    if cmdCount < 26 {
        t.Fatalf("walked only %d commands — expected at least 26 (D-09 floor)", cmdCount)
    }
    if flagCount < 50 {
        t.Fatalf("inspected only %d flags — expected at least 50 (D-09 floor)", flagCount)
    }
    // ... report accepted-via-reference vs accepted-via-allowlist counts,
    // fail on any unaccounted flag, fail on any allowlist entry matching
    // nothing currently registered ...
}
```
Verified counts this session (temporary probe, `internal/cli`, tree left clean afterward): with `InitDefaultCompletionCmd()` called, the real tree walks to **36 commands** and **114 flags** (including one `--help` per command and the four completion `--no-descriptions` flags) — both floors (`≥26`, `≥50`) clear comfortably regardless of exactly how the final guard counts persistent-vs-local. `hiddenCmds=1` (`man`), `hiddenFlags=0`, `deprecatedFlags=0` — matching D-02/root.go's existing state exactly.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hand-authored `docs/FLAG-PARITY.md` + `internal/cli/flag_parity_test.go` substring-match drift guard | `docs/CLI-REFERENCE.md` generated by `cobra/doc` + a hidden/deprecated-only accounting guard | `docs/FLAG-PARITY.md` deleted at v0.11.0 (commit `5139e60c`, DOCS-02); this phase (v0.13.0 Phase 12) is DOCS-05, the deferred replacement | The old guard could only assert "flag name appears as a substring somewhere in the doc" — it did nothing for content generation itself. The new shape removes hand-authoring entirely and narrows the guard to the one thing generation can't cover (hidden/deprecated visibility) |

**Deprecated/outdated:** None — `cobra/doc`'s `GenMarkdownCustom`/`GenMarkdownTree` API is the current, actively maintained surface at cobra v1.10.2; no newer generator API exists in this version.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The planner's exact `linkHandler` transform (`strings.TrimSuffix` + `strings.ReplaceAll("_", "-")` + `"#"` prefix) will produce anchors that GitHub's Markdown renderer (and any Markdown viewer used to browse this file) actually resolves — this assumes standard GFM auto-heading-anchor rules (lowercase, spaces→hyphens, no other punctuation in `cmd.CommandPath()` values) | Architecture Patterns → Pattern 1, Code Examples | Low risk: every `cmd.CommandPath()` value observed in this tree is already lowercase ASCII with only spaces as separators (verified via the probe's printed command paths), so the transform is exercised against the real command set, not a hypothetical one. Worst case is a cosmetic broken in-file link, not a functional failure |
| A2 | `tools/clidoc` is discretionary in whether it also emits a page for `root` itself the same way `GenMarkdownTreeCustom` does (this research recommends yes, for `codegraph --help`-parity) | Code Examples → tools/clidoc/main.go skeleton | Low risk: an explicit planner decision either way is cheap to verify against the committed file's own root section once generated |

**If this table is empty:** N/A — two low-risk assumptions recorded above; everything else in this document is `[VERIFIED]` via direct source reads or a live probe run this session, or `[CITED]` from the deleted-test git history and Taskfile/ci.yml as committed.

## Open Questions (RESOLVED)

1. **Should `tools/clidoc` also generate a page for the `root` command itself?** — RESOLVED at plan time (12-01-PLAN.md Task 1: `walk(root)` — the root section is generated, matching `codegraph --help` parity).
   - What we know: `GenMarkdownTreeCustom` always generates the command it's called on plus recurses into children — so cobra's own convention includes root. `GenMarkdownCustom(root, ...)` prints root's own `Short`/`Long`/usage/`### Options` (none today — root has no local flags, only the auto `--help`)/`### SEE ALSO` (root has no parent link, but lists every available top-level child).
   - What's unclear: whether a root-level section adds value in a single-file reference where the top-level command list is already implicit from the section headers below it.
   - Recommendation: Include it (as the skeleton above does) — it's what a real `codegraph --help` shows, it's zero extra code (the same `walk` function handles it), and omitting it would be an arbitrary special case to maintain.

2. **Exact allowlist line format details (D-08 leaves this to planner discretion).** — RESOLVED at plan time (12-01-PLAN.md Task 2: a literal tab character delimits `<command path>[ --flag]` from `<reason>`; `#`-prefixed comment lines and blank lines are ignored).
   - What we know: D-08 specifies the semantic content (`<full command path> [--flag]<TAB><reason>`, one whole-hidden-command line covers all its flags) and the one committed example (`codegraph man` for D-02's reason).
   - What's unclear: whether to use a literal tab character or a documented delimiter convention, and whether comment lines (`#`-prefixed) should be supported for readability.
   - Recommendation: A literal tab is fine and matches D-08's own notation; a plan can add `#`-comment support for zero extra guard complexity if useful, but it is not required by any of D-07..D-10.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | `tools/clidoc`, guard test, `task docs:cli`/`docs:cli:drift` | ✓ | `go 1.26.6` pinned in `go.mod:3`; every invocation prefixed `GOTOOLCHAIN=go1.26.6` per D-14 | — |
| `github.com/spf13/cobra/doc` | Generator | ✓ (already vendored via `go.sum`, v1.10.2) | v1.10.2 | — |
| `git` | `task docs:cli:drift` (byte-compare against committed file), Taskfile precondition pattern shared with `proto:drift` | ✓ | — | — |
| CI runner (`ci.yml` `test` job) | `docs:cli:drift` slotting after `task proto:drift` (line 196) | ✓ — same job already has Go/Task set up; no Node/pnpm needed since this is Go-only | — | — |

No missing dependencies. This phase needs nothing beyond what `test` job in `ci.yml` already provisions.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (`go test`), same as the rest of `internal/cli` |
| Config file | none — plain `go test ./internal/cli/...` |
| Quick run command | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -run TestEveryRegisteredFlagIsAccountedFor -v` |
| Full suite command | `GOTOOLCHAIN=go1.26.6 task test:unit` (reaches `internal/cli` — confirmed: `test:unit`'s `go list ./...` excludes only `/internal/daemon$`, and `internal/cli` is not that) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DOCS-05 | `docs/CLI-REFERENCE.md` is generated from the live tree and stays current | drift/integration | `GOTOOLCHAIN=go1.26.6 task docs:cli:drift` | ❌ Wave 0 — new Taskfile target |
| DOCS-06 | Every registered flag is accounted for (reference or allowlist), RED-demonstrated against a throwaway hidden flag | unit | `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -run TestEveryRegisteredFlagIsAccountedFor -v` | ❌ Wave 0 — new `internal/cli/cli_reference_test.go` + `internal/cli/testdata/cli-reference-allowlist.txt` |
| DOCS-07 | brew-trust wording recommends the narrow `--cask` grant with security framing | manual: diff review, no test (D-12; ROADMAP.md:121, criterion 3) | — (verified by reading the diff) | N/A by decision |

### Sampling Rate
- **Per task commit:** `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/... -run TestEveryRegisteredFlagIsAccountedFor -v`
- **Per wave merge:** `GOTOOLCHAIN=go1.26.6 task test:unit` and `GOTOOLCHAIN=go1.26.6 task docs:cli:drift`
- **Phase gate:** Full suite green (`task test:unit` including the new guard) plus `task docs:cli:drift` green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/cli/cli_reference_test.go` — covers DOCS-06
- [ ] `internal/cli/testdata/cli-reference-allowlist.txt` — the committed allowlist DOCS-06's guard reads (starts with exactly one entry: `codegraph man`)
- [ ] `tools/clidoc/main.go` — the generator DOCS-05 depends on (no test framework needed to create it, but its own correctness is what `docs:cli:drift` exercises transitively)
- [ ] `Taskfile.yml` — `docs:cli` and `docs:cli:drift` targets (neither exists today; confirmed via search)
- [ ] `.github/workflows/ci.yml` — new step for `docs:cli:drift` in the `test` job, immediately after the existing `task proto:drift` step (currently `ci.yml:195-196`)

Framework install: none — Go's standard `testing` package and the already-present Task runner cover everything; no new test framework dependency.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Not applicable — no auth surface touched |
| V3 Session Management | no | Not applicable |
| V4 Access Control | no | Not applicable |
| V5 Input Validation | no | No user input is parsed by this phase's code — the generator reads the in-process Cobra tree, the guard reads two committed files by fixed relative path |
| V6 Cryptography | no | Not applicable |
| V12 Files and Resources (informal — no direct ASVS bucket in the template above) | yes | `tools/clidoc` writes only to `-out` (default `docs/CLI-REFERENCE.md`), never elsewhere; the guard reads only `docs/CLI-REFERENCE.md` and `internal/cli/testdata/cli-reference-allowlist.txt`, both fixed relative paths, fail-closed if either is unreadable (D-07, mirrors the deleted `flag_parity_test.go`'s fail-closed pattern) |

### Known Threat Patterns for this phase's stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| UF-2 — docs-as-security-guidance framing (this phase's DOCS-07 work directly touches this class: telling users to run a command that silences a real Homebrew security control) | Repudiation / Information Disclosure (of what the control actually protects against) | Recommend the narrowest working grant (`--cask`, not `--tap`) and name the actual control being bypassed (arbitrary Ruby execution at install time via this cask's own post-install hook) in one sentence — D-11's exact fix shape. This IS the mitigation; there is no code-level control to add |
| Hidden/deprecated flag accounting blind spot (DOCS-06's whole reason for existing) | Information Disclosure avoidance failure — a flag silently omitted from BOTH the generated reference and any tracking mechanism | The walk-based guard with a committed, reason-carrying allowlist (D-07/D-08); RED-demonstrated by registering a throwaway hidden flag and confirming the guard fails naming it (D-10 family a) |
| Generated-file tampering (a developer or a compromised dependency hand-edits `docs/CLI-REFERENCE.md` after generation, e.g. to hide a flag or insert misleading text) | Tampering | Caught by `task docs:cli:drift`'s byte-compare against a fresh regeneration into a temp directory — any hand-edit, however small, differs from what the pinned toolchain and current tree would produce, and fails the gate (same shape as `proto:drift`/`web:drift`) |
| Allowlist rot (an entry lingers after the flag/command it covered is removed, silently widening the set of "flags nobody has to document") | Tampering / a stale exemption becoming an unnoticed backdoor for future undocumented flags | D-08's own rule: "An entry in the allowlist that no longer matches any registered flag or command is itself a failure" — the guard must actively check allowlist entries against the live tree, not just check the live tree against the allowlist. Expected `threats_open: 0` per D-13 |

`threats_open: 0` is the expected outcome recorded in `12-SECURITY.md` per D-13 — every threat row above carries either a test (DOCS-06's guard, drift gate) or a verdict (DOCS-07 is a wording fix with no code-level mitigation possible or needed).

## Sources

### Primary (HIGH confidence)
- `github.com/spf13/cobra@v1.10.2/doc/md_docs.go` — read directly from the module cache (`GenMarkdownCustom`, `GenMarkdownTreeCustom`, `printOptions`, `DisableAutoGenTag` propagation, exact line numbers cited)
- `github.com/spf13/cobra@v1.10.2/doc/util.go` — read directly (`hasSeeAlso`)
- `github.com/spf13/cobra@v1.10.2/command.go` — read directly (`Flags`, `PersistentFlags`, `LocalFlags`, `InheritedFlags`, `mergePersistentFlags`, `InitDefaultHelpFlag`, `InitDefaultHelpCmd`, `IsAvailableCommand`, `IsAdditionalHelpTopicCommand`, `ExecuteC`, `InitDefaultCompletionCmd` call site, `Commands`, `EnableCommandSorting`)
- `github.com/spf13/cobra@v1.10.2/completions.go` — read directly (`InitDefaultCompletionCmd`, `initCompleteCmd`, the `--no-descriptions` flag on each shell subcommand)
- `github.com/spf13/pflag@v1.0.10/flag.go` — read directly (`FlagUsagesWrapped`, `PrintDefaults`, `HasAvailableFlags`, `SortFlags` default)
- `internal/cli/root.go`, `internal/cli/man.go` — read directly (existing tree shape, `Hidden: true` on `man`, no `PersistentFlags`/`Deprecated`/other `Hidden` anywhere today)
- Temporary probe test run this session against the real `internal/cli` package (added, run, deleted, tree confirmed clean via `git status --porcelain`) — 36 commands / 114 flags / 1 hidden command / 0 hidden or deprecated flags with `InitDefaultCompletionCmd()` called; 26 commands without it
- `git show 5139e60c^:internal/cli/flag_parity_test.go` — the deleted DOCS-06 precedent, read directly
- `Taskfile.yml` lines 369-479 (`proto:drift`) and 744-776 (`web:build` marker/digest shape) — read directly
- `.github/workflows/ci.yml` lines 46-244 (`test` job, including exact `proto:drift` step at line 196) — read directly
- `/Users/sean/.claude/gsd-core/bin/lib/commands.cjs` lines 2684-2719 (`cmdTodoComplete`) — read directly, confirms `gsd_run todo complete <basename>` moves `.planning/todos/pending/<file>` → `.planning/todos/completed/<file>` with stamped `completed`/`status` frontmatter fields
- `README.md` lines 65-86, `docs/RELEASE.md` lines 505-540 and 565-590 — read directly; confirms **no `brew trust` text exists in README.md today** (only a pointer to `docs/RELEASE.md`) — the DOCS-07 wording fix is entirely inside `docs/RELEASE.md`, not README.md
- `.planning/config.json` — read directly, confirms `workflow.nyquist_validation: true` and `workflow.security_enforcement: true`

### Secondary (MEDIUM confidence)
- None used beyond primary sources — every claim in this document traces to a direct source read or a live probe run this session.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependency; existing `cobra`/`pflag` versions confirmed directly from `go.mod`/`go.sum`/module cache
- Architecture: HIGH — every API call recommended (`InitDefaultCompletionCmd`, `DisableAutoGenTag`, `IsAvailableCommand`, flag visitation) is read directly from the pinned cobra/pflag source, not inferred
- Pitfalls: HIGH — Pitfalls 1 and 2 are both independently confirmed via a live probe run against this exact codebase this session, not just read from documentation

**Research date:** 2026-09-13
**Valid until:** Stable for the life of this `cobra v1.10.2` pin — re-verify `DisableAutoGenTag`/`InitDefaultCompletionCmd` behavior only if `go.mod`'s cobra version bumps
