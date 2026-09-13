# Phase 12: CLI Reference & Docs Tail - Context

**Gathered:** 2026-09-13
**Status:** Ready for planning

<domain>
## Phase Boundary

The docs tail of v0.13.0: a committed `docs/CLI-REFERENCE.md` generated from the live Cobra tree and kept current by a regenerate-and-diff gate (DOCS-05); a small walk-based guard so no hidden or deprecated flag is ever silently unaccounted for (DOCS-06); and the brew-trust instructions rewritten to recommend the narrow `--cask` grant with one sentence of security framing (DOCS-07). Nothing in this phase changes runtime behaviour of the binary beyond exporting the root-command constructor for the generator and, where a command's help text is thin, improving its `Long`. The ROADMAP Phase 12 goal, criteria 1–3 and Notes, and REQUIREMENTS DOCS-05/06 were reworded at discuss time (2026-09-13, maintainer) to this shape; the pre-amendment wording (hand-authored reference, docs-grep guard for DOCS-07) is superseded.

</domain>

<decisions>
## Implementation Decisions

### CLI reference generation (DOCS-05)
- **D-01:** `docs/CLI-REFERENCE.md` is ONE committed file generated from `newRootCmd()` by `github.com/spf13/cobra/doc` — already a dependency (`codegraph man` uses `doc.GenManTree`), so no new require. It is never hand-edited; every edit to it comes from regeneration. Hand-maintaining ~94 flag rows is exactly the toil a generator exists to remove (maintainer, 2026-09-13).
- **D-02:** The generator is a small Go program `tools/clidoc/main.go` (sibling of `tools/graphcluster`) that walks the tree in Cobra's own order, calls `doc.GenMarkdownCustom(cmd, w, linkHandler)` per available command into one buffer (single-file output; `GenMarkdownTree`'s one-file-per-command layout is rejected — 30+ files for a reference nobody browses that way), and writes `docs/CLI-REFERENCE.md` with a short generated-file banner. `internal/cli` exports the constructor for it (`NewRootCmd()` wrapper around `newRootCmd()`, or the rename — planner's call); the `tools/` package may import `internal/*` within this module (precedent `tools/graphcluster`).
- **D-03:** `task docs:cli` regenerates in place; `task docs:cli:drift` regenerates into a TEMPORARY file and byte-compares it against the committed one (the `proto:drift` shape: reports `compared 1 generated file` before comparing, names the file on mismatch, non-zero exit) and is wired into `ci.yml` in the same job that runs `task proto:drift` (ci.yml:196). A stale committed reference is the one real failure mode this phase guards against, and it fires on every flag addition.
- **D-04:** Command prose is carried by `cmd.Short`/`cmd.Long` in the Cobra tree — the single source of truth — never by a separate hand-written file stitched in. Nine commands carry a `Long` today; the planner adds a `Long` only where a command's current `--help` does not convey its argument semantics or precedence rules (e.g. `ui` already explains `--editor-url` / `CODEGRAPH_EDITOR_URL` / `--no-editor-url` precedence in its `Long`, so the reference documents this milestone's new flag automatically). Do not gold-plate: `Short` alone is acceptable for a command whose flags are self-explanatory.
- **D-05:** Hidden commands (`man`, `Hidden: true` per v0.5.0 D-02 — it exists for the Homebrew cask post-install hook) stay OUT of the generated reference, as `cobra/doc` already does (`md_docs.go:135` skips `!IsAvailableCommand()`); hidden flags likewise. Hidden means hidden from user docs; the guard (below) is what keeps them from being *silently* hidden.
- **D-06:** README.md gains one link to `docs/CLI-REFERENCE.md` in its existing docs pointers (the paragraph that already points at `docs/RELEASE.md`); no other README restructuring.

### Accounting guard (DOCS-06)
- **D-07:** `internal/cli/cli_reference_test.go` → `TestEveryRegisteredFlagIsAccountedFor` (runs in `task test:unit`, therefore in CI with no extra wiring). It builds the root command, calls `InitDefaultHelpFlag()` on each command exactly as `cobra/doc` does (`md_docs.go:59`) so the walk and the generator see the same flag set, walks EVERY command recursively — hidden commands included — and visits `cmd.Flags()` and `cmd.PersistentFlags()` with `VisitAll` (which visits hidden and deprecated flags), de-duplicating by command path + flag name. Inherited persistent flags are attributed to the command that declares them, not re-counted on every descendant.
- **D-08:** One rule, no exemption list: a flag on an available (non-hidden) command that is itself neither hidden nor deprecated must appear as `--name` in the generated `docs/CLI-REFERENCE.md`; every other flag — hidden, deprecated, or belonging to a hidden command — must appear in the committed allowlist `internal/cli/testdata/cli-reference-allowlist.txt` (one entry per line: `<full command path> [--flag]<TAB><reason>`; a whole hidden command may be listed once by path to cover all its flags). An entry in the allowlist that no longer matches any registered flag or command is itself a failure (allowlists rot upward too). Today the allowlist has exactly one entry: `codegraph man` with D-02's reason.
- **D-09:** Positive assertions (rule `84d1gfpywd`): the test `t.Logf`s and asserts commands walked ≥ 26, flags inspected ≥ 50, and reports how many were accepted via the reference vs the allowlist; it fails closed if `docs/CLI-REFERENCE.md` or the allowlist is unreadable. It does NOT scope-match flags to sections (the drift gate already guarantees the doc's content is exactly the generator's) and it does NOT test `cobra/doc`'s behaviour — only this project's artefacts.
- **D-10:** RED families for `12-MUTATION-LOG.md`, each applied for real in the working tree and reverted byte-clean: (a) register a throwaway hidden flag on `ui` (`Bool("zz-throwaway", …)` + `MarkHidden`) → the guard fails naming `codegraph ui --zz-throwaway` as unaccounted; (b) delete one line from the committed `docs/CLI-REFERENCE.md` → `task docs:cli:drift` fails naming the file; (c) add a bogus allowlist entry for a flag that does not exist → the guard fails on the rotten entry. No family exercises `cobra/doc` itself.

### brew-trust instructions (DOCS-07)
- **D-11:** `docs/RELEASE.md` (the "untrusted tap" instructions at ~518–533 and the blockquote at ~577–586) recommends — *corrected at plan time 2026-09-13: research finding 3 established that `README.md` carries no `brew trust` text at all; its Homebrew paragraph (~71–83) only points readers at `docs/RELEASE.md`, so README needs no DOCS-07 edit beyond keeping that pointer* — `brew trust --cask seanb4t/tap/codegraph` as THE command to run, followed by one sentence naming the control: Homebrew refuses because a third-party cask runs arbitrary Ruby on your machine at install time — this cask's own post-install hook does (it generates man pages and checks the installed version) — so `brew trust` is a security control you are opting out of, and the `--cask` form limits that opt-out to this one cask. The tap-wide grant is mentioned as existing and not recommended by this project, without spelling its command (nothing to copy-paste). The verbatim Homebrew error quote is trimmed to the narrow form with an ellipsis so the docs never carry the broad form as an instruction.
- **D-12:** No guard, no test, no docs-grep for DOCS-07 — it is a wording change verified by reading the diff (ROADMAP.md:121 already says so; criterion 3 was reworded to match). The pending todo `2026-08-10-brew-trust-instructions-…` is resolved through the todo tool when the edit lands.

### Verification and house artefacts
- **D-13:** `12-VALIDATION.md` (plan-phase seeds it), `12-MUTATION-LOG.md` (families a–c), `12-SECURITY.md` (rows: UF-2 docs-as-security-guidance framing; hidden-flag accounting blind spot; generated-file tampering caught by drift; `threats_open: 0` expected). Small phase: 2–3 plans, sequential (`use_worktrees=false`).
- **D-14:** Every `go` invocation prefixed `GOTOOLCHAIN=go1.26.6`; positive-count guards (`rg -o … | wc -l`, never `-c`); the drift target reports its compared-file count BEFORE comparing (proto:drift precedent, Taskfile.yml:436).

### Claude's Discretion
- Whether `NewRootCmd` is an exported wrapper or a rename of `newRootCmd` (touching ~50 call sites in tests argues for the wrapper).
- Exact `linkHandler` / banner text and whether `GenMarkdownCustom`'s per-command `### SEE ALSO` blocks are kept (they are harmless in a single file; keep unless they double the file).
- Which thin `Short`-only commands get a `Long`, if any.
- Allowlist file location (`internal/cli/testdata/…` preferred so the test's `read_first` is local) and line format details.
- Where in `ci.yml`'s drift job `docs:cli:drift` slots (adjacent to `task proto:drift`).

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `github.com/spf13/cobra/doc` already imported by `internal/cli/man.go` (`doc.GenManTree`) — `GenMarkdownCustom` is in the same package; no new dependency.
- `internal/cli/root.go:44` `newRootCmd()` builds the whole tree (26 top-level commands incl. the hidden `man`; nested `completion bash|zsh|fish|powershell`, `daemon …`, `githooks install|uninstall`, …; ~94 visible flags; no persistent, hidden or deprecated flags today).
- The deleted 08-06 guard (`internal/cli/flag_parity_test.go` at `5139e60c^`) — recursive `newRootCmd()` walk with `Flags().VisitAll`, fail-closed on a missing doc — is the skeleton for D-07; drop its `help` exemption (D-07 calls `InitDefaultHelpFlag` instead) and add the allowlist branch + counts.
- `tools/graphcluster/` (Phase 11) — the shape for a `tools/` Go program importing `internal/*` with its own `_test.go`.
- `Taskfile.yml` `proto:drift` (lines 369–440) — the regenerate-into-temp-then-byte-compare gate with a reported file count; `ci.yml:196` runs it.

### Established Patterns
- Guards carry a positive assertion that they did work (rule `84d1gfpywd`) and are demonstrated RED against an applied, byte-cleanly-reverted mutation before they are trusted (every v0.13.0 phase's MUTATION-LOG).
- Generated files are committed and drift-checked, never regenerated in CI as a side effect (`proto:drift`, `web:drift`).
- Docs pointers live in README's install section paragraph (already points at `docs/RELEASE.md`).

### Integration Points
- `docs/CLI-REFERENCE.md` (new, generated) · `tools/clidoc/main.go` (new) · `internal/cli/root.go` (exported constructor) · `internal/cli/cli_reference_test.go` + `internal/cli/testdata/cli-reference-allowlist.txt` (new) · `Taskfile.yml` (`docs:cli`, `docs:cli:drift`) · `.github/workflows/ci.yml` (drift job) · `README.md`, `docs/RELEASE.md` (brew-trust wording, reference link) · `.planning/todos/pending/2026-08-10-brew-trust-…md` (resolved).

</code_context>

<specifics>
## Specific Ideas

- "Are we reimplementing cobra doc? If so, why?" — the maintainer's framing that reset the phase: use the generator for the document, guard only the generator's one blind spot, and never write a test whose subject is a third-party library's behaviour.
- "What are we actually testing here and why?" — DOCS-07 is a wording change; no test.

</specifics>

<deferred>
## Deferred Ideas

- Wiring Phase 11's `check:gonum` and `check:no-force-layout` Taskfile targets into `ci.yml` — recorded in 11-SECURITY.md and STATE.md as a follow-up; not docs-tail work.
- Reconciling the four stale rows in STATE.md's Pending Todos table against `.planning/todos/pending/` — noted since Phase 7; a deliberate pass, not this phase.

</deferred>
