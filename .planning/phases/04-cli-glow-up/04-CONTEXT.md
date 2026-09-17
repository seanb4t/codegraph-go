# Phase 4: CLI Glow-up - Context

**Gathered:** 2026-09-17
**Status:** Ready for planning

<domain>
## Phase Boundary

Every human-output verb renders through a `present` renderer using one shared semantic palette — adaptive to light and dark backgrounds, downsampled to what the terminal can show, overridable by `--color` and the standard environment variables — with grouped, consistently styled help and consistent short flags, while the agent/MCP path, `--json` and piped output stay byte-identical and the archtest that keeps charm out of the serve-reachable closure catches every import path added this milestone. Requirements: CLI-01…CLI-08, GRD-13.

In scope: `internal/cli` and `internal/cli/present` (plus its `archtest`), `go.mod` (promote `colorprofile` to direct; `fang/v2` only on an adopted verdict), `docs/CLI-REFERENCE.md` regeneration via the existing drift gate, and the phase's verdict/mutation-log artifacts. Out of scope: any change to `internal/query`, `internal/mcp`, the wire oracle's transcripts, the 8-tool MCP set, `progress_cli.go`'s stderr-fd resolver (different TTY semantics — not unified with the stdout resolver), and `tools/clidoc`'s `cobra/doc` generator (v0.13.0 Phase 12 ruling).

</domain>

<decisions>
## Implementation Decisions

### Cross-cutting ruling: test our contract, never charm's behaviour or Go's dependency management
- **D-00:** The v0.13.0 Phase 12 maintainer ruling applies to every test this phase adds: **never test a third-party library's behaviour** (lipgloss rendering, colorprofile's env precedence or downsampling, cobra's help/group rendering, fang's chrome) and **never test Go dependency management** (no assertions on `go.mod` contents, module counts, direct-vs-indirect status, or `go.mod`↔denylist sync). Tests assert *our* decisions and *our* registrations: which branch our resolver selects, what environ we hand to `colorprofile.Detect`, which `GroupID` each of our commands carries, that our plain path is byte-identical to its frozen golden, that our import graph excludes charm. Library behaviour we rely on is **cited in a code comment with the pinned version**, not re-verified. "Denylist in the same commit as the `go.mod` change" (GRD-13) is a commit/review discipline recorded in `04-MUTATION-LOG.md`, not a guard. The existing `TestCharmCgoClosure` stays as the guard for the *project* constraint (cgo-free binary); nothing new of that class is added. Where a requirement's wording reads as a library test (CLI-02's four-terminal rendering, CLI-03's "every combination", CLI-06's grouped rendering), it is read as the combinations/assertions **we own** — see D-06, D-09, D-13.

### Fang spike & verdict (CLI-08)
- **D-01:** The `serve --mcp` byte-identical proof is a green `task test:wireoracle` run under the wrapped `Execute()` — the oracle already spawns the real binary via `os/exec` and pins 38 frozen transcripts; the spike records that run in the verdict. No spike-only harness. — **Reversibility:** reversible.
- **D-02:** "SBOM/govulncheck delta acceptable" means: zero new `govulncheck ./...` findings **and** `TestCharmCgoClosure` green over the widened `internal/cli` tree. The list of new modules fang would pull in is recorded in the verdict as information, not a gate (D-00: no module-count assertion).
- **D-03:** Fourth spike criterion (Phase 3 WR-01 lesson): `codegraph query` / `codegraph unlock` must still print their rename message **exactly once** to stderr and exit 1 — `renamed_test.go`'s real-binary test stays green, using fang's error-handler option if one exists. If fang cannot honour the stub contract, it is declined. Under fang, `main.go` must not double-print (fang renders the error; main only exits non-zero).
- **D-04:** The verdict is recorded as `04-FANG-VERDICT.md` in the phase directory (the `02-/03-MUTATION-LOG.md` precedent) and summarised as a PROJECT.md Key Decisions row at phase end. The four criteria are conjunctive: fang is adopted only if all of D-01, D-02, D-03 and the `WithoutManpage()`/`WithoutCompletions()` composition with the existing hidden `man` and cobra `completion` hold; otherwise help is hand-rolled (D-14). If adopted, the wrap lives in `cli.Execute()` (`root.go`) as `fang.Execute(ctx, newRootCmd(), fang.WithoutManpage(), fang.WithoutCompletions(), fang.WithVersion(...))`, leaving `cmd/codegraph/main.go`'s shape and `NewRootCmd()` (clidoc's entry) untouched. **No renderer or help template lands before the verdict commit** (CLI-08). — **Reversibility:** costly once adopted — the `go.mod` require and the `Execute()` boundary change ride the release.
  - **Research correction (2026-09-17, `fang.go` read at v2.0.1):** `fang.WithoutCompletions()` does **not** avoid a fang-owned command — fang has no completion command; it sets `cobra.CompletionOptions.DisableDefaultCmd = true`, which **removes cobra's own `completion` command** that GoReleaser's `generate_completions_from_executable` depends on. The spike therefore tests the candidate call set `fang.Execute(ctx, root, fang.WithoutManpage(), fang.WithVersion(...))` — `WithoutCompletions()` is deliberately **not** called — and the "compose" criterion is read operationally: the hidden `man <dir>` still runs and `codegraph completion bash|zsh|fish` still emit scripts, both checked empirically in the spike and recorded in the verdict. `WithoutManpage()` is confirmed correct (fang only adds its own hidden `man` when `opts.manpages` is true).

### Palette & renderers (CLI-01, CLI-04)
- **D-05:** A `present.Palette` struct with exactly the seven semantic roles — header, label, value, path, count, warning, error — built by `present.NewPalette(dark bool)` using `lipgloss.LightDark(dark)` for every hue. Every `Render*` takes the palette as a parameter; today's three package-level vars (`headerStyle`, `labelStyle`, `sectionStyle`) fold into it. No mutable package-level styles, no `SetDark()` setter.
- **D-06:** Hues are truecolor hex pairs (light/dark) per role, chosen at Claude's discretion and downsampled by colorprofile at the RunE boundary (D-10). Readability on Solarized Light (or macOS light Terminal) and a dark theme is a **human UAT item** — CLI-04 needs eyes; a `human_needed` verification pause at the end of the phase is expected, not a defect. CLI-02's "`TERM=dumb`, 16-colour, 256-colour and truecolor each render correctly" is likewise a **manual check on a real terminal** (`TERM=xterm`, over SSH) recorded as a verification item — colorprofile's downsampling is not unit-tested (D-00). Styled output is **never golden-frozen** (freezing escape bytes = testing lipgloss's encoding).
- **D-07:** `explore` and `node` styled output keeps the **same sections, same order, same wording** as the markdown the plain branch prints (`Engine.Explore`/`Engine.Node`); hue replaces markdown syntax (`#`, `**`, backticks). The styled branch calls `eng.ExploreDetail`/`eng.NodeDetail` — the same plain structs the UI consumes — with zero changes to `internal/query`. No re-layout, so piped vs TTY show the same content.
- **D-08:** One-line verbs (`version`, `uninit`, `githooks`, `upgrade`, `serve`/`ui` banners, `daemon` status lines, `telemetry`) render through a small generic `present.Line`/`present.Lines` helper over the palette roles. Bespoke `Render<Verb>` functions exist only for struct-bearing verbs: explore, node, search (both default and `--full` shapes), callers, callees, impact, affected, install/uninstall (branching inside `printAgentResults` over `agents.WriteResult`), plus the existing status and files renderers.

### Colour resolution, `--color`, short flags (CLI-02, CLI-03, CLI-07)
- **D-09:** One shared `internal/cli` resolver (`colorflag.go`) performs the **single** environment read via `colorprofile.Detect(stdout, environ)`. `--color` is applied by rewriting the environ handed to Detect: `always` ⇒ drop `NO_COLOR`/`CLICOLOR`, add `CLICOLOR_FORCE=1`; `never` ⇒ add `NO_COLOR=1`; `auto` ⇒ pass through untouched. Precedence below `--color` is therefore colorprofile's own (`NO_COLOR` > `CLICOLOR_FORCE` > `CLICOLOR=0` > TTY), **cited in a code comment with the pinned version, not re-tested** (D-00). The `styled` boolean feeds the **unchanged** two-arg `present.ChoosePresentation`. The CLI-03 unit matrix covers only the layer we own — `--color ∈ {auto, always, never, invalid} × TTY ∈ {yes, no}` — asserting (a) the rewritten environ slice, (b) `always` on a pipe → styled and `never` on a TTY with `CLICOLOR_FORCE=1` → plain (positive assertions both ways, rule `84d1gfpywd`), (c) `invalid` → usage error.
  - **Research correction (2026-09-17, `colorprofile/env.go` executed at v0.4.3):** colorprofile gates `NO_COLOR` through `strconv.ParseBool`, so `NO_COLOR=banana` / `NO_COLOR=0` do **not** disable colour there — diverging from CLI-03's "any non-empty value disables" and from this repo's shipped contract (`ChoosePresentation` treats any non-empty `noColor` as off, so `NO_COLOR=banana codegraph status` is plain today). Delegating the raw value would be a **behaviour regression**. Decision: the resolver **normalizes** — if `NO_COLOR` is non-empty in the real environ it is rewritten to `NO_COLOR=1` before the environ reaches `Detect`. This is our input to their function, asserted as one more rewritten-environ case in the D-09 matrix (`NO_COLOR=banana` → environ carries `NO_COLOR=1`), not a test of colorprofile; the code comment cites the ParseBool behaviour at v0.4.3 so nobody removes the normalization as redundant.
- **D-10:** Downsampling is `colorprofile.Writer{Forward: cmd.OutOrStdout(), Profile: p}` wrapped at the RunE boundary. `present` keeps emitting full-fidelity ANSI and never sees the profile — its "must NOT read `os.Getenv` or call `term.IsTerminal`" contract (v1.0 Phase 6 D-03) is unchanged.
- **D-11:** `--color` is a persistent flag on root (the `inheritedFromAncestor` guard in `cli_reference_test.go` already accounts for it), default `auto`; an unknown value is a usage error (exit 1, naming `auto|always|never`). `lipgloss.HasDarkBackground` is queried **once**, in the resolver, only when styled **and** stdout is a TTY — never on a pipe, since the query writes to the terminal.
  - **Research correction (2026-09-17, `lipgloss/v2` `query.go`/`terminal.go` read at v2.0.5):** `HasDarkBackground(in, out)` requires **both** stdin and stdout to be TTYs (it puts stdin into raw mode to read the OSC-11 reply) and blocks up to a hard-coded 2 s when the terminal does not answer, then defaults to dark. The gate is therefore **styled AND stdout is a TTY AND stdin is a TTY**, passing the real `os.Stdin`/`os.Stdout`; with stdin redirected the resolver defaults to dark **without** calling the query. The 2 s worst case on non-answering terminals (some emulators, tmux without `allow-passthrough`, some SSH multiplexers) is inherent to the live query and in scope to *observe*, not fix: the CLI-04/CLI-02 UAT checklist records it as an expected symptom ("a ~2 s pause before styled output over SSH/tmux is the OSC-11 query timing out, not a hang") and the phase's tmux e2e run notes the observed wall time. If UAT finds the pause unacceptable, that is a separate follow-up decision (backlog), not a silent widening here.
- **D-12:** CLI-07 is already satisfied — discovery finding: every `--json`/`--limit`/`--kind`/`--path` on the query verbs already carries `-j`/`-l`/`-k`/`-p` (VERB-02 closed `search`; `node`'s `-l` is `--line` and it has no `--limit`, so there is no conflict; `explore` has none of the four long forms). The deliverable is a **table test pinning the invariant** — walk the cobra tree and, for each of the four long names present on a query verb, assert its short exists (positive control: a planted long-only flag goes RED). No new short flags; `-d`/`-m` are not added.

### Help grouping & guards (CLI-05, CLI-06, GRD-13)
- **D-13:** Group assignments — **Query the graph:** explore, search, node, callers, callees, impact, affected, files, status · **Build the index:** init, index, sync, daemon, githooks, uninit · **Agents & serving:** serve, ui, install, uninstall · **Maintenance:** version, upgrade, telemetry, help, completion (`SetHelpCommandGroupID`/`SetCompletionCommandGroupID`). Hidden `man`/`query`/`unlock` stay groupless. The CLI-06 guard asserts **our data**: every visible command carries one of the four `GroupID`s and none is groupless (a planted ungrouped command goes RED); whether cobra draws the titles is not tested (D-00).
- **D-14:** If fang is declined, help is hand-rolled as `root.SetHelpFunc` → `present.RenderHelp` over cobra's command/group/flag metadata, gated by the same resolver (D-09), so `<verb> --help` inherits root's styling. Unstyled/non-TTY help is cobra's stock grouped template — `tools/clidoc`'s `cobra/doc` walk and the `docs/CLI-REFERENCE.md` drift gate are untouched either way.
- **D-15:** The `present` archtest's `forbiddenImportPaths` becomes **prefix matching** on `charm.land/` and `github.com/charmbracelet/` (both vanity roots), keeping the self-defeat probe as the positive control. It is proven RED by a planted import from a guarded package for **each** newly-required module (`colorprofile`; `fang/v2` if adopted), recorded in `04-MUTATION-LOG.md` in the Phase 2/3 shape. The denylist change lands in the **same commit** as each `go.mod` change — a commit discipline, not a test (D-00).
- **D-16:** CLI-05's regression is a set of **plain-output goldens captured before any renderer lands** (the GRF-09 "bar before measurement" discipline): for every verb that gains a styled branch, run against the existing test corpus with a non-TTY writer and `NO_COLOR=1`, freeze stdout to `testdata/plain/<verb>.golden` as the phase's first test commit. After the glow-up the same test asserts byte-equality **and** zero ESC bytes. The golden oracle, wire oracle and TUI-01 archtest are otherwise untouched.

### Claude's Discretion
- Exact hex values for the seven roles' light/dark pairs (subject to the D-06 UAT).
- File layout of the renderers inside `present/` (one file per verb vs grouped by shape) and the resolver's exact function signature, subject to D-09/D-10.
- The exact `fang` options used in the spike beyond the two `Without*` calls, and how `fang.WithVersion` consumes `versionLine()`.
- The `present.RenderHelp` layout when hand-rolled, as long as it renders the four D-13 groups in that order.
- Plan ordering, subject to the hard sequence: plain goldens (D-16) → fang spike + verdict (D-01…D-04) → archtest prefix + `colorprofile` promotion in one commit (D-15) → resolver (D-09…D-11) → palette + renderers (D-05…D-08) → groups/help (D-13/D-14) → `docs/CLI-REFERENCE.md` regen through the drift gate. TDD mode is on (`workflow.tdd_mode: true`): Go RED evidence is the `test(04-NN):` commit before the implementation plus the pasted `--- FAIL:` transcript in the SUMMARY (rule `x1cjy9vyhq`); never gate a plan on `gsd-tools check tdd-red-evidence`.

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/cli/present` — the sole lipgloss home by archtest (v1.0 Phase 6 D-01); `styles.go` (3 monochrome styles), `status.go`/`files.go` (`RenderStatus`/`RenderFiles` over `query.StatusResult`/`query.FilesResult`), `progress.go` (stderr ticker, `progressStyle.Foreground(lipgloss.Color("212"))` is the one colour precedent), `sanitize.go`, `tty.go` (`ChoosePresentation(isTTY bool, noColor string) bool`, pure).
- Plain-struct seams already in `internal/query`: `NodeDetail`/`ExploreDetail` (`detail.go:250,681`), `Status`, `Files`, `Query` (`[]*schema.Node`) / `Search` (`[]Location`), `Callers`/`Callees`/`Impact`/`Affected` (`traverse.go`). `Engine.Explore`/`Engine.Node` are thin markdown wrappers over the detail structs.
- `install.go:139` `printAgentResults` over `[]agents.WriteResult` — the seam for install/uninstall styling.
- `test/wireoracle` spawns the real binary (`capture.go`, `os/exec`) — reusable as the D-01 proof.
- `internal/cli/renamed.go` + `renamed_test.go` — the stub contract D-03 must preserve (one stderr line via `main.go`, exit 1).
- `cli_reference_test.go` (`TestEveryRegisteredFlagIsAccountedFor`, `inheritedFromAncestor`) and `testdata/cli-reference-allowlist.txt`; `task docs:cli` / `task docs:cli:drift`.
- `internal/cli/present/archtest/import_graph_test.go` — `forbiddenImportPaths` (3 literals), `charmImporterProbePath` self-defeat guard, six `guardedPackages`, `internal/cli` excluded from the closure by design.
- `go.mod`: `charm.land/lipgloss/v2 v2.0.5` direct; `github.com/charmbracelet/colorprofile v0.4.3` and `x/ansi v0.11.7` already **indirect** — promotion adds no new supply-chain surface. `charm.land/fang/v2 v2.0.1` is the candidate (requires cobra ≥ v1.9.1, lipgloss ≥ v2.0.1; repo pins v1.10.2 / v2.0.5).

### Established Patterns
- RunE idiom at three call sites (`status.go:86`, `files.go:72`, `progress_cli.go:31`): `if present.ChoosePresentation(term.IsTerminal(int(os.Stdout.Fd())), os.Getenv("NO_COLOR")) { return present.RenderX(result, out) }` after the `--json` early return — the styled branch is structurally unreachable from `--json` and non-TTY.
- Env/fd reads happen only in `internal/cli` RunE sites; `present` is env-blind (D-03) — `internal/cli/tui/tty.go` mirrors this with package-level vars for test seams.
- Every command sets `SilenceUsage`/`SilenceErrors`; `main.go` prints the returned error once and exits 1 — the only error exit path.
- Guards are positive-controlled (rule `84d1gfpywd`) and proven RED in a per-phase `0N-MUTATION-LOG.md`; generated docs are build artefacts behind a drift gate (Phase 12); never test third-party behaviour (Phase 12, D-00).
- Short flags already consistent: `-p` everywhere; `-j` on search/callers/callees/impact/affected/files/status; `-l` on search/callers/callees; `-k` on search; `-d` on impact/affected; `node` uses `-f`/`-l` for `--file`/`--line`.

### Integration Points
- `root.go` `newRootCmd()` — `AddGroup`, per-command `GroupID`, `SetHelpCommandGroupID`/`SetCompletionCommandGroupID`, persistent `--color`, and (if adopted) the fang wrap in `Execute()`; `NewRootCmd()` stays the clidoc entry.
- Every human-output verb's RunE — resolver call + `colorprofile.Writer` wrap + `present.Render*` branch.
- `present/styles.go` → `Palette`; new `present/help.go`, `present/line.go`, per-verb renderers.
- `present/archtest/import_graph_test.go` — prefix denylist.
- `docs/CLI-REFERENCE.md` regenerated via `task docs:cli` (the `--color` persistent flag and groups flow through automatically).

</code_context>

<specifics>
## Specific Ideas

- Expect two human verification items at phase end and treat them as expected, not gaps: (1) CLI-04 palette readability on Solarized Light / macOS light Terminal and a dark theme; (2) CLI-02 rendering in `TERM=dumb`, a 16-colour `TERM=xterm`, 256-colour and truecolor, ideally one over SSH.
- The fang verdict file records all four criteria with the evidence run for each (wire-oracle output tail, govulncheck output, cgo-closure test, stub real-binary test) so the decision is auditable without re-running the spike.
- Cite colorprofile's precedence (`NO_COLOR` > `CLICOLOR_FORCE` > `CLICOLOR`) in the resolver's comment with its version, noting that Node's `FORCE_COLOR` convention is the opposite so nobody "fixes" it.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. (Adding `-d`/`-m` short flags and testing colorprofile's precedence directly were considered and declined, not deferred.)

</deferred>
