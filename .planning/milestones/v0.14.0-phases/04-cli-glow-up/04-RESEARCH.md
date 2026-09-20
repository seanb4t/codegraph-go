# Phase 4: CLI Glow-up - Research

**Researched:** 2026-09-17
**Domain:** Terminal-color/CLI-styling glow-up on an existing Go/Cobra CLI (lipgloss v2 + colorprofile + optional fang wrapper), grouped cobra help, short-flag consistency
**Confidence:** HIGH — every API claim below was verified this session either by reading the pinned module's source directly from the local module cache (`/Users/sean/go/pkg/mod`) or by running real Go code against it; no claim rests on training-data recall of these APIs.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Cross-cutting ruling: test our contract, never charm's behaviour or Go's dependency management**
- **D-00:** The v0.13.0 Phase 12 maintainer ruling applies to every test this phase adds: **never test a third-party library's behaviour** (lipgloss rendering, colorprofile's env precedence or downsampling, cobra's help/group rendering, fang's chrome) and **never test Go dependency management** (no assertions on `go.mod` contents, module counts, direct-vs-indirect status, or `go.mod`↔denylist sync). Tests assert *our* decisions and *our* registrations: which branch our resolver selects, what environ we hand to `colorprofile.Detect`, which `GroupID` each of our commands carries, that our plain path is byte-identical to its frozen golden, that our import graph excludes charm. Library behaviour we rely on is **cited in a code comment with the pinned version**, not re-verified. "Denylist in the same commit as the `go.mod` change" (GRD-13) is a commit/review discipline recorded in `04-MUTATION-LOG.md`, not a guard. The existing `TestCharmCgoClosure` stays as the guard for the *project* constraint (cgo-free binary); nothing new of that class is added. Where a requirement's wording reads as a library test (CLI-02's four-terminal rendering, CLI-03's "every combination", CLI-06's grouped rendering), it is read as the combinations/assertions **we own** — see D-06, D-09, D-13.

**Fang spike & verdict (CLI-08)**
- **D-01:** The `serve --mcp` byte-identical proof is a green `task test:wireoracle` run under the wrapped `Execute()` — the oracle already spawns the real binary via `os/exec` and pins 38 frozen transcripts; the spike records that run in the verdict. No spike-only harness. — **Reversibility:** reversible.
- **D-02:** "SBOM/govulncheck delta acceptable" means: zero new `govulncheck ./...` findings **and** `TestCharmCgoClosure` green over the widened `internal/cli` tree. The list of new modules fang would pull in is recorded in the verdict as information, not a gate (D-00: no module-count assertion).
- **D-03:** Fourth spike criterion (Phase 3 WR-01 lesson): `codegraph query` / `codegraph unlock` must still print their rename message **exactly once** to stderr and exit 1 — `renamed_test.go`'s real-binary test stays green, using fang's error-handler option if one exists. If fang cannot honour the stub contract, it is declined. Under fang, `main.go` must not double-print (fang renders the error; main only exits non-zero).
- **D-04:** The verdict is recorded as `04-FANG-VERDICT.md` in the phase directory (the `02-/03-MUTATION-LOG.md` precedent) and summarised as a PROJECT.md Key Decisions row at phase end. The four criteria are conjunctive: fang is adopted only if all of D-01, D-02, D-03 and the `WithoutManpage()`/`WithoutCompletions()` composition with the existing hidden `man` and cobra `completion` hold; otherwise help is hand-rolled (D-14). If adopted, the wrap lives in `cli.Execute()` (`root.go`) as `fang.Execute(ctx, newRootCmd(), fang.WithoutManpage(), fang.WithoutCompletions(), fang.WithVersion(...))`, leaving `cmd/codegraph/main.go`'s shape and `NewRootCmd()` (clidoc's entry) untouched. **No renderer or help template lands before the verdict commit** (CLI-08). — **Reversibility:** costly once adopted — the `go.mod` require and the `Execute()` boundary change ride the release.

  > **Research correction (verified this session against `charm.land/fang/v2@v2.0.1`'s actual source — see Common Pitfalls #3 below):** fang does **not** ship its own competing `completion` subcommand. `WithoutCompletions()` sets `root.CompletionOptions.DisableDefaultCmd = true`, which **removes cobra's own default `completion` command** — the exact command GoReleaser's `generate_completions_from_executable` relies on. Calling `WithoutCompletions()` therefore does not "compose with the existing cobra completions" as D-04's literal option list implies — it deletes them. The verdict spike must resolve this before landing: either the correct call is `fang.WithoutManpage()` alone (and cobra's stock `completion` command is left untouched, which is what "composes with" should mean operationally), or `WithoutCompletions()` is deliberately kept off the call list. `WithoutManpage()` is verified correct as-is: fang only adds its own hidden `man` command when `opts.manpages` is true, so `WithoutManpage()` cleanly avoids the real collision with this repo's own hidden `man <dir>` command.

**Palette & renderers (CLI-01, CLI-04)**
- **D-05:** A `present.Palette` struct with exactly the seven semantic roles — header, label, value, path, count, warning, error — built by `present.NewPalette(dark bool)` using `lipgloss.LightDark(dark)` for every hue. Every `Render*` takes the palette as a parameter; today's three package-level vars (`headerStyle`, `labelStyle`, `sectionStyle`) fold into it. No mutable package-level styles, no `SetDark()` setter.
- **D-06:** Hues are truecolor hex pairs (light/dark) per role, chosen at Claude's discretion and downsampled by colorprofile at the RunE boundary (D-10). Readability on Solarized Light (or macOS light Terminal) and a dark theme is a **human UAT item** — CLI-04 needs eyes; a `human_needed` verification pause at the end of the phase is expected, not a defect. CLI-02's "`TERM=dumb`, 16-colour, 256-colour and truecolor each render correctly" is likewise a **manual check on a real terminal** (`TERM=xterm`, over SSH) recorded as a verification item — colorprofile's downsampling is not unit-tested (D-00). Styled output is **never golden-frozen** (freezing escape bytes = testing lipgloss's encoding).
- **D-07:** `explore` and `node` styled output keeps the **same sections, same order, same wording** as the markdown the plain branch prints (`Engine.Explore`/`Engine.Node`); hue replaces markdown syntax (`#`, `**`, backticks). The styled branch calls `eng.ExploreDetail`/`eng.NodeDetail` — the same plain structs the UI consumes — with zero changes to `internal/query`. No re-layout, so piped vs TTY show the same content.
- **D-08:** One-line verbs (`version`, `uninit`, `githooks`, `upgrade`, `serve`/`ui` banners, `daemon` status lines, `telemetry`) render through a small generic `present.Line`/`present.Lines` helper over the palette roles. Bespoke `Render<Verb>` functions exist only for struct-bearing verbs: explore, node, search (both default and `--full` shapes), callers, callees, impact, affected, install/uninstall (branching inside `printAgentResults` over `agents.WriteResult`), plus the existing status and files renderers.

**Colour resolution, `--color`, short flags (CLI-02, CLI-03, CLI-07)**
- **D-09:** One shared `internal/cli` resolver (`colorflag.go`) performs the **single** environment read via `colorprofile.Detect(stdout, environ)`. `--color` is applied by rewriting the environ handed to Detect: `always` ⇒ drop `NO_COLOR`/`CLICOLOR`, add `CLICOLOR_FORCE=1`; `never` ⇒ add `NO_COLOR=1`; `auto` ⇒ pass through untouched. Precedence below `--color` is therefore colorprofile's own (`NO_COLOR` > `CLICOLOR_FORCE` > `CLICOLOR=0` > TTY), **cited in a code comment with the pinned version, not re-tested** (D-00). The `styled` boolean feeds the **unchanged** two-arg `present.ChoosePresentation`. The CLI-03 unit matrix covers only the layer we own — `--color ∈ {auto, always, never, invalid} × TTY ∈ {yes, no}` — asserting (a) the rewritten environ slice, (b) `always` on a pipe → styled and `never` on a TTY with `CLICOLOR_FORCE=1` → plain (positive assertions both ways, rule `84d1gfpywd`), (c) `invalid` → usage error.

  > **Research finding, positively verified this session (see Common Pitfalls #1):** `colorprofile@v0.4.3`'s own `NO_COLOR` check is `strconv.ParseBool(env.get("NO_COLOR"))`, not "any non-empty value." `NO_COLOR=banana` does **not** disable colour via `colorprofile.Detect`/`Env` — only a `strconv.ParseBool`-truthy value (`1`, `true`, `t`, `T`, `TRUE`, `True`) does. This diverges from CLI-03's literal wording ("NO_COLOR (any non-empty value disables)") and from the no-color.org spec's plain-English framing. Since `--color=never`'s rewrite writes the canonical `NO_COLOR=1` (ParseBool-true), the resolver itself is unaffected — this only matters for a user's own raw `NO_COLOR=<arbitrary-string>` in their environment. Flagged in Open Questions; D-00 forbids re-testing colorprofile's own parsing, so this is a decision (accept the divergence, document it) not a bug to fix in `present`.

- **D-10:** Downsampling is `colorprofile.Writer{Forward: cmd.OutOrStdout(), Profile: p}` wrapped at the RunE boundary. `present` keeps emitting full-fidelity ANSI and never sees the profile — its "must NOT read `os.Getenv` or call `term.IsTerminal`" contract (v1.0 Phase 6 D-03) is unchanged.
- **D-11:** `--color` is a persistent flag on root, default `auto`; an unknown value is a usage error (exit 1, naming `auto|always|never`). `lipgloss.HasDarkBackground` is queried **once**, in the resolver, only when styled **and** stdout is a TTY — never on a pipe, since the query writes to the terminal.

  > **Research finding, verified this session against `lipgloss/v2@v2.0.5`'s actual source (see Common Pitfalls #2):** `HasDarkBackground(in, out term.File) bool` requires **both** `in` and `out` to be a terminal (`BackgroundColor` returns an error on Unix if either is not a TTY) and blocks for up to a hard-coded 2-second timeout (`defaultQueryTimeout = 2 * time.Second`) waiting for the terminal's OSC-11 response before falling back to `true` (dark) on any error/timeout. "Only when styled and stdout is a TTY" (D-11's literal wording) is therefore an *incomplete* gate: if stdout is a TTY but stdin is redirected (`echo x | codegraph status`, common in scripts and some CI harnesses), the call still fires, still blocks (briefly — it errors fast on a non-TTY stdin, not the full 2s), and defaults to dark. The **worse** case — a real interactive terminal that doesn't answer OSC 11 (some emulators, some tmux/SSH configurations without passthrough) — pays the full 2-second stall on **every styled invocation**. This is exactly Pitfall 1's "over SSH" manual-check item in CONTEXT.md's Notes, now with the precise mechanism and duration.

- **D-12:** CLI-07 is already satisfied — discovery finding: every `--json`/`--limit`/`--kind`/`--path` on the query verbs already carries `-j`/`-l`/`-k`/`-p` (VERB-02 closed `search`; `node`'s `-l` is `--line` and it has no `--limit`, so there is no conflict; `explore` has none of the four long forms). The deliverable is a **table test pinning the invariant** — walk the cobra tree and, for each of the four long names present on a query verb, assert its short exists (positive control: a planted long-only flag goes RED). No new short flags; `-d`/`-m` are not added.

**Help grouping & guards (CLI-05, CLI-06, GRD-13)**
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

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope. (Adding `-d`/`-m` short flags and testing colorprofile's precedence directly were considered and declined, not deferred.)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CLI-01 | Every human-output verb renders through `present` with 7 semantic hues, consuming existing plain-struct seams, zero `internal/query`/`internal/mcp` changes | Verb-by-verb seam table already in `ARCHITECTURE.md` Part 1 confirmed live: `search.go`, `node.go`, `explore.go`, `install.go` read directly this session (see Code Examples) — `explore`/`node` currently have **no** styled branch and **no** `--json` at all, confirming they are pure markdown-string call sites today that must switch to `ExploreDetail`/`NodeDetail` |
| CLI-02 | `colorprofile` downsampling at the RunE boundary; `TERM=dumb`/16/256/truecolor render correctly | `colorprofile@v0.4.3`'s `Detect`/`Env`/`Writer` read directly from source (Common Pitfalls #1, Code Examples) |
| CLI-03 | `--color=auto\|always\|never` with the stated precedence, unit-matrix tested | `colorprofile@v0.4.3`'s actual precedence order verified by running real Go code (Common Pitfalls #1) — diverges from CLI-03's own "any non-empty value" wording for raw `NO_COLOR` |
| CLI-04 | Adaptive palette, `HasDarkBackground` read once | `lipgloss/v2@v2.0.5`'s `HasDarkBackground`/`BackgroundColor`/`queryBackgroundColor` read directly from source — dual-TTY requirement and 2s timeout newly documented (Common Pitfalls #2) |
| CLI-05 | Byte-identical agent/MCP/`--json`/piped path; `NO_COLOR`+non-TTY regression | `test/wireoracle` (real-binary harness, `task test:wireoracle`) and `test/integration/renamed_stubs_test.go` (real-binary stub-contract test) both read directly — exact D-01/D-03 evidence mechanisms confirmed |
| CLI-06 | `cobra.Group`-based grouped help, `help`/`completion` filed, styled `<verb> --help` | `cobra@v1.10.2`'s `Group`, `AddGroup`, `SetHelpFunc`, `SetHelpCommandGroupID`, `SetCompletionCommandGroupID` all read directly from `command.go` (Code Examples) |
| CLI-07 | `-j`/`-l`/`-k`/`-p` present wherever the long form exists | `search.go`/`node.go` read directly — confirms D-12's claim that this requirement is *already* satisfied; only a pinning test is new work |
| CLI-08 | Fang spiked with a recorded verdict before any renderer lands | `charm.land/fang/v2@v2.0.1`'s `fang.go`/`help.go` read directly from source — corrects D-04's `WithoutCompletions()` assumption (see User Constraints correction above and Common Pitfalls #3) |
| GRD-13 | `present` archtest catches every charm-family import added this milestone | `import_graph_test.go` and `charm_cgo_test.go` both read directly — confirms D-15's prefix-widening is necessary (colorprofile/x-ansi live at `github.com/charmbracelet/...`, not `charm.land/...`) and that `TestCharmCgoClosure`'s own `charm.land`-only prefix scope is a **separate, narrower** boundary that D-02 does not ask to widen |
</phase_requirements>

## Summary

This phase's technical shape was already locked down in exhaustive detail by `/gsd-discuss-phase` (see `04-CONTEXT.md`'s D-00 through D-16) and by the milestone-level `STACK.md`/`PITFALLS.md`/`ARCHITECTURE.md`. This research's job — per its own brief — was to verify the *exact current APIs* those decisions depend on, against the actually-pinned module versions, rather than trust training-data recall or the milestone research's necessarily-lighter (Context7/WebFetch-sourced, MEDIUM-confidence) verification pass. All four target modules (`charm.land/lipgloss/v2@v2.0.5`, `github.com/charmbracelet/colorprofile@v0.4.3`, `github.com/spf13/cobra@v1.10.2`, `charm.land/fang/v2@v2.0.1`) were read directly from the local Go module cache this session, and two behavioral claims were confirmed by running real Go code against the pinned `colorprofile` version.

Three findings correct or sharpen the locked decisions and must reach whoever executes the D-01…D-04 fang spike and the D-09/D-11 resolver:

1. **`colorprofile@v0.4.3`'s `NO_COLOR` check is `strconv.ParseBool`-gated, not "any non-empty value."** `NO_COLOR=banana` does not disable colour. Verified by running real code against the pinned version (Common Pitfalls #1).
2. **`lipgloss.HasDarkBackground` requires both stdin AND stdout to be a TTY, and can block up to 2 seconds** waiting for an OSC-11 terminal response before defaulting to dark. This is a real latency risk on terminals that don't answer the query (some emulators, tmux/SSH without passthrough) — every styled invocation pays the cost, not just a cold start (Common Pitfalls #2).
3. **`fang.WithoutCompletions()` does not avoid a collision with a fang-owned completion command — fang has none.** It instead **disables cobra's own default `completion` command**, the one GoReleaser's `generate_completions_from_executable` relies on. D-04's literal option list (`WithoutManpage()` + `WithoutCompletions()`) needs re-examination during the spike: the correct call for "compose with the existing cobra completions" is very likely `WithoutManpage()` alone (Common Pitfalls #3).

Everything else CONTEXT.md already decided is confirmed accurate against the pinned source: `cobra.Group`/`AddGroup`/`SetHelpCommandGroupID`/`SetCompletionCommandGroupID`/`SetHelpFunc` all exist exactly as described; `fang.Execute`'s error-handler behavior matches D-03's double-print concern precisely (it prints `err.Error()` once, so `main.go` must stop printing under fang); `search.go`/`node.go` already carry the short flags D-12 claims; `explore.go`/`node.go` today call only the markdown-string `Engine.Explore`/`Engine.Node` methods with no styled branch and no `--json` at all, confirming the "third consumer of an existing seam" plan for D-07.

**Primary recommendation:** proceed exactly per CONTEXT.md's locked sequence (plain goldens → fang spike+verdict → archtest+colorprofile promotion → resolver → palette/renderers → groups/help → docs regen), but before writing `04-FANG-VERDICT.md`, re-derive the correct fang option set from `fang.go`'s actual `settings`/`Execute` logic (this document's Code Examples section has it verbatim) rather than from the milestone-level STACK.md's Context7-sourced summary, which is medium-confidence and contains the `WithoutCompletions()` misconception this research corrects.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Colour/TTY/`NO_COLOR`/`--color` resolution | CLI (`internal/cli`, RunE boundary) | — | D-03/D-09/D-10's existing contract: `present` must stay env-blind; only `internal/cli` reads `os.Getenv`/`term.IsTerminal`/`colorprofile.Detect` |
| Styled rendering (palette, `Render<Verb>`) | CLI (`internal/cli/present`) | — | Sole lipgloss-import home by archtest (D-01, v1.0 Phase 6); consumes plain structs, never recomputes them |
| Plain-struct data assembly (`StatusResult`, `ExploreResult`, `NodeDetail`, …) | Query Engine (`internal/query`) | — | Unchanged this phase (CLI-01's explicit "zero changes to `internal/query`") — CLI/present is strictly a third consumer |
| Agent/MCP JSON-RPC output | MCP (`internal/mcp`) | — | Untouched; never imports charm (TUI-01 archtest); this phase adds zero new import paths reaching it |
| Cobra command tree / grouping / help dispatch | CLI (`internal/cli/root.go`) | — | `cobra.Group`/`AddGroup`/`SetHelpFunc` are cobra APIs operated entirely from `internal/cli`; no server/storage tier involved |
| Process-level help/error/version chrome (if fang adopted) | CLI (`cmd/codegraph/main.go` / `internal/cli.Execute()`) | — | `fang.Execute` wraps the whole `Execute()` call, one level above every RunE — a process-boundary concern, not a per-verb one |
| Docs generation (`docs/CLI-REFERENCE.md`) | Build/Docs tooling (`tools/clidoc`) | CLI | Reads the live cobra tree's metadata; fang/groups changes flow through automatically, no `tools/clidoc` code change needed |

## Standard Stack

### Core
| Library | Version (verified) | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `charm.land/lipgloss/v2` | v2.0.5 (already pinned in `go.mod`; confirmed present at `/Users/sean/go/pkg/mod/charm.land/lipgloss/v2@v2.0.5`) | Style palette (`Style.Render`, `LightDark`, `HasDarkBackground`) | Already the project's sole styling package by archtest (D-01); no version change needed [VERIFIED: go.mod + local module cache, read this session] |
| `github.com/charmbracelet/colorprofile` | v0.4.3 (currently `// indirect` in `go.mod`; promote to direct per D-15) | `Detect`/`Env`/`Writer` — NO_COLOR/CLICOLOR/CLICOLOR_FORCE/TERM=dumb detection and ANSI downsampling | Lipgloss v2's own documented downsampling sibling; already transitively resolved, zero new supply-chain surface [VERIFIED: go.mod + `env.go`/`writer.go` read this session, and behavior confirmed by executing real Go code against v0.4.3 — see Common Pitfalls #1] |
| `github.com/spf13/cobra` | v1.10.2 (already pinned) | `Group`/`AddGroup`/`SetHelpCommandGroupID`/`SetCompletionCommandGroupID`/`SetHelpFunc` | Core cobra API, present in this exact form well before the pinned version [VERIFIED: `command.go` read directly at lines 45-48 (`Group`), 77 (`GroupID` field), 333 (`SetHelpFunc`), 343 (`SetHelpCommandGroupID`), 352 (`SetCompletionCommandGroupID`), 1396 (`AddGroup`)] |
| `charm.land/fang/v2` | v2.0.1 (candidate — NOT yet in `go.mod`; adopt only per D-01…D-04 verdict) | Styled `--help`/error/version chrome wrapping `cobra.Command.Execute` | Resolved as the true current-latest via a real `go get charm.land/fang/v2@latest` against the live module proxy this session; `fang.go`/`help.go` read directly from the resolved module cache [VERIFIED: `go list -m charm.land/fang/v2` → `v2.0.1`, executed this session; source read from `/Users/sean/go/pkg/mod/charm.land/fang/v2@v2.0.1/`] |

**Installation (if/when each is adopted — no installs happen until their respective plan step, per D-15's "same commit as go.mod" discipline):**
```bash
# Promote colorprofile from indirect to direct (already in go.sum at v0.4.3)
go get github.com/charmbracelet/colorprofile@v0.4.3

# ONLY if the D-01...D-04 fang verdict is "adopt"
go get charm.land/fang/v2@v2.0.1
```

**Version verification performed this session:**
```
$ go list -m charm.land/fang/v2   # after `go get charm.land/fang/v2@latest` in a scratch module
charm.land/fang/v2 v2.0.1
```
This confirms v2.0.1 is genuinely the current latest (not a stale training-data guess) and that the module resolves cleanly against the live Go module proxy at research time (2026-09-17).

### Supporting (fang's transitive closure — pulled in only if fang is adopted)
| Library | Purpose | Note |
|---------|---------|------|
| `github.com/muesli/mango-cobra`, `github.com/muesli/mango`, `github.com/muesli/roff` | Man-page generation fang uses internally | Only exercised if `opts.manpages == true`; `WithoutManpage()` means these compile in but their man-page-building code path (`mango.NewManPage`) never runs [VERIFIED: `fang.go`'s `if opts.manpages { root.AddCommand(...) }` block, read this session] |
| `github.com/charmbracelet/x/term`, `x/termios`, `x/windows` | Terminal primitives fang/lipgloss/colorprofile depend on | Already-familiar charm-ecosystem packages; `x/term` already a direct dep |
| `github.com/lucasb-eyer/go-colorful`, `github.com/mattn/go-runewidth`, `github.com/rivo/uniseg`, `github.com/clipperhouse/{displaywidth,uax29}` | Color math / Unicode width | Pure Go, pulled in transitively; `go get charm.land/fang/v2@latest` in a scratch module this session listed the full set with no CGo-suggestive package names [VERIFIED: `go get` output, this session] |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `colorprofile.Detect`/`lipgloss.LightDark`/`HasDarkBackground` | `charm.land/lipgloss/v2/compat` (`compat.AdaptiveColor`, `compat.HasDarkBackground`) | The compat package is lipgloss's v1→v2 migration shim for existing v1-era code — not applicable here since the palette is freshly authored (D-05 already made this call correctly) |
| `cobra.AddGroup` + hand-rolled help template (D-14) | `charm.land/fang/v2` full adoption | Fang gives styled error/version/help chrome for free but wraps the whole `Execute()` boundary — exactly the tradeoff the D-01…D-04 spike exists to evaluate |

## Package Legitimacy Audit

> The `gsd-tools query package-legitimacy check` seam only supports `--ecosystem npm|pypi|crates`; it has no Go-ecosystem mode (confirmed by invoking it this session — it prints its own usage error for `--ecosystem go`). Go module legitimacy was therefore verified manually, by source: every listed package's origin, maintainer, and behavior was read directly from the local module cache after resolving through the real `proxy.golang.org` (via `go get`), not from a summary or a search result.

| Package | Registry | Age / Provenance | Source Repo | Verdict | Disposition |
|---------|----------|------|-------------|---------|-------------|
| `charm.land/lipgloss/v2` | Go module proxy | Already a direct dependency of this repo since v1.0/Phase 6; charmbracelet org | `github.com/charmbracelet/lipgloss` (aliased at `charm.land/lipgloss`) | OK | Approved — already in use |
| `github.com/charmbracelet/colorprofile` | Go module proxy | Already resolved transitively (indirect) via bubbletea v2; charmbracelet org | `github.com/charmbracelet/colorprofile` | OK | Approved — promote to direct only |
| `charm.land/fang/v2` | Go module proxy | v2.0.1, resolved live via `go get ...@latest` this session; charmbracelet org, same org/vanity-domain as lipgloss/bubbletea/bubbles already trusted by this repo | `github.com/charmbracelet/fang` (aliased at `charm.land/fang`) | OK | Candidate only — gated behind the D-01…D-04 spike verdict, never installed speculatively |
| `github.com/spf13/cobra` | Go module proxy | Already a direct dependency; v1.10.2 pinned | `github.com/spf13/cobra` | OK | Approved — already in use, no version change |

**Packages removed due to SLOP verdict:** none.
**Packages flagged as suspicious [SUS]:** none — all four packages are either already in this repo's dependency graph or from the same well-established `charmbracelet` GitHub org this repo already depends on for its TUI surface (bubbletea/bubbles/lipgloss), verified by reading their actual source, not by trusting a name match.

*fang's transitive closure (mango-cobra/mango/roff, x/term family, go-colorful, go-runewidth, uniseg, clipperhouse/displaywidth+uax29) was enumerated via a real `go get` this session — no CGo-suggestive import paths or `CgoFiles` were observed among fang's own package tree, consistent with `TestCharmCgoClosure`'s existing "charm.land closure has 0 CgoFiles" invariant.*

## Architecture Patterns

### System Architecture Diagram

```
                     stdin/stdout/stderr, --color flag, NO_COLOR/CLICOLOR* env
                                          │
                                          ▼
                          cmd/codegraph/main.go → cli.Execute()
                                          │
                    ┌─────────────────────┴─────────────────────┐
                    │ if fang adopted (D-01..D-04 verdict):      │
                    │   fang.Execute(ctx, root,                  │
                    │     WithoutManpage(), [maybe skip          │
                    │     WithoutCompletions() — see correction],│
                    │     WithVersion(...))                      │
                    │   — wraps help/error/version chrome for    │
                    │   the WHOLE command tree, one boundary     │
                    │   above every RunE                         │
                    └─────────────────────┬─────────────────────┘
                                          ▼
                      newRootCmd() — cobra.Group / GroupID / --color
                      persistent flag / AddGroup / SetHelpFunc (D-14
                      hand-rolled path, only if fang declined)
                                          │
                           ┌──────────────┴───────────────┐
                           ▼                               ▼
                  every verb's RunE                 hidden stubs (query/
                  (24 human-output verbs)            unlock), man, help,
                           │                          completion — groupless
        ┌──────────────────┼──────────────────┐       or Maintenance group
        ▼                  ▼                   ▼
  --json early     internal/cli resolver   plain fmt.Fprint
  return (D-02,     (colorflag.go, D-09):   fallback (non-TTY,
  byte-identical    colorprofile.Detect(    NO_COLOR set, or
  to today)         rewritten environ)      --color=never)
        │                  │
        │                  ▼
        │         styled?  → colorprofile.Writer{Forward, Profile}
        │                    wraps cmd.OutOrStdout() (D-10, downsampling
        │                    ONLY here, never inside present)
        │                  │
        │                  ▼
        │         lipgloss.HasDarkBackground(stdin, stdout) — ONCE,
        │         only if styled && stdout is a TTY (D-11; NOTE: also
        │         requires stdin to be a TTY per lipgloss's own
        │         BackgroundColor contract — see Common Pitfalls #2)
        │                  │
        │                  ▼
        │         present.NewPalette(dark) → present.Render<Verb>(
        │           eng.ExploreDetail/NodeDetail/StatusResult/...,
        │           palette, w)
        │                  │
        └──────────────────┴──────────────────┐
                                                ▼
                          internal/query.Engine (UNCHANGED this phase)
                          — sole read-only plain-struct source, already
                          shared by CLI/MCP/UI (v0.12.0 Phase 1 seam)
```

The critical property this diagram must make visible: **`--json` and the non-TTY/`NO_COLOR` plain path never pass through the resolver's styled branch at all** — they return before the resolver call is even reached (matching every existing RunE's structure, confirmed in `status.go`/`search.go` this session), which is exactly what keeps CLI-05's byte-identity guarantee mechanically true rather than merely tested.

### Recommended Project Structure
```
internal/cli/
├── root.go              # cobra.Group / AddGroup / --color persistent flag / (fang wrap if adopted)
├── colorflag.go          # NEW — the D-09 resolver: --color × TTY × NO_COLOR/CLICOLOR* → (styled bool, profile)
├── search.go, node.go,   # each gains: resolver call → colorprofile.Writer wrap →
│   explore.go, ...       #   present.Render<Verb>(eng.<Detail-or-existing-call>(...), palette, w)
└── present/
    ├── styles.go          # today's 3 vars fold into present.Palette (D-05)
    ├── palette.go          # NEW — present.Palette struct + NewPalette(dark bool)
    ├── line.go             # NEW — present.Line/Lines generic one-liner helper (D-08)
    ├── help.go             # NEW, only if fang declined (D-14) — present.RenderHelp
    ├── explore.go, node.go,# NEW per-verb Render* funcs, called with ExploreDetail/NodeDetail (D-07)
    │   search.go, ...
    └── archtest/
        ├── import_graph_test.go  # forbiddenImportPaths → prefix match (D-15)
        └── charm_cgo_test.go     # UNCHANGED — charm.land-prefix scope is a separate, narrower guard (see GRD-13 note)
```

### Pattern 1: The three-call-site RunE idiom, extended
**What:** Every RunE that gains styling follows the exact shape `status.go`/`search.go` already use today (verified this session): resolve start path → open engine → **`--json` early return** → styled/plain branch. The glow-up only changes the third step's *content* (resolver + colorprofile.Writer + present.Render*), never the first two, and never moves the `--json` return.
**When to use:** Every one of the 24 verbs listed in CLI-01.
**Example (verified current shape, `status.go`, lines 78-96):**
```go
// Source: internal/cli/status.go, read directly this session — this is
// the exact idiom every styled verb already follows and the glow-up
// extends, not replaces.
if jsonOut {
    data, err := query.MarshalStatusJSON(result)
    if err != nil {
        return err
    }
    return writeJSONLine(cmd, data)
}

if present.ChoosePresentation(term.IsTerminal(int(os.Stdout.Fd())), os.Getenv("NO_COLOR")) {
    return present.RenderStatus(result, start, cmd.OutOrStdout())
}

fmt.Fprint(cmd.OutOrStdout(), query.RenderStatusText(result, start))
return nil
```
The D-09 resolver replaces the inline `term.IsTerminal(...)`/`os.Getenv(...)` pair with a call to `colorflag.Resolve(cmd)` (or similar), and the styled branch additionally wraps `cmd.OutOrStdout()` in a `colorprofile.Writer` before calling `present.RenderStatus`. `present.RenderStatus`'s own signature and body do not need to change for this — it already takes an `io.Writer` and returns `error`.

### Pattern 2: `explore`/`node` gain a styled branch that does not exist today
**What:** Confirmed by reading `explore.go`/`node.go` directly this session: **neither command has `--json`, a styled branch, or any TTY/NO_COLOR check today** — both call only `eng.Explore(query, maxFiles) (string, error)` / `eng.Node(symbol, file, lineHint) (string, error)` and print the returned markdown string verbatim. Per D-07, the new styled branch must call `eng.ExploreDetail`/`eng.NodeDetail` (confirmed present at `internal/query/detail.go:681`/`:250`) — never regex/string-manipulate the markdown `Explore`/`Node` already return.
**When to use:** `explore`, `node` only (the two verbs whose plain path is markdown text rather than a struct-driven loop).
**Example:**
```go
// Source: internal/query/detail.go, read directly this session —
// confirmed signatures the styled branch must call.
func (e *Engine) ExploreDetail(query string, maxFiles int) (ExploreResult, error) // detail.go:681
func (e *Engine) NodeDetail(symbol, file string, line *int) (NodeDetail, error)  // detail.go:250

// internal/query/explore.go:231 and internal/query/node.go:337 confirm
// Explore/Node are thin wrappers over these — the plain markdown path
// stays untouched; the styled branch becomes present's third consumer.
```

### Pattern 3: The resolver as the single environment read (D-09)
**What:** `colorflag.go`'s resolver is the only place `os.Environ()`/`--color` is consulted; it produces both the `styled bool` (feeding the unchanged `present.ChoosePresentation`) and the `colorprofile.Profile` (feeding the `colorprofile.Writer` at D-10).
**Verified signatures to build against:**
```go
// Source: github.com/charmbracelet/colorprofile@v0.4.3/env.go, read this session
func Detect(output io.Writer, env []string) Profile
func Env(env []string) (p Profile)

// Source: github.com/charmbracelet/colorprofile@v0.4.3/writer.go, read this session
func NewWriter(w io.Writer, environ []string) *Writer
type Writer struct {
    Forward io.Writer
    Profile Profile
}
func (w *Writer) Write(p []byte) (int, error)

// Source: charm.land/lipgloss/v2@v2.0.5/color.go, read this session
type LightDarkFunc func(light, dark color.Color) color.Color
func LightDark(isDark bool) LightDarkFunc

// Source: charm.land/lipgloss/v2@v2.0.5/query.go, read this session
func HasDarkBackground(in term.File, out term.File) bool
```

### Pattern 4: cobra grouping (D-13/D-14, if fang declined)
**Verified signatures:**
```go
// Source: github.com/spf13/cobra@v1.10.2/command.go, read this session
type Group struct {
    ID    string
    Title string
}
// Command struct field, command.go:77
GroupID string

func (c *Command) AddGroup(groups ...*Group)                 // command.go:1396
func (c *Command) SetHelpFunc(f func(*Command, []string))    // command.go:333
func (c *Command) SetHelpCommandGroupID(groupID string)      // command.go:343
func (c *Command) SetCompletionCommandGroupID(groupID string) // command.go:352
```
Usage shape for D-13's four groups:
```go
root.AddGroup(
    &cobra.Group{ID: "query", Title: "Query the graph:"},
    &cobra.Group{ID: "build", Title: "Build the index:"},
    &cobra.Group{ID: "agents", Title: "Agents & serving:"},
    &cobra.Group{ID: "maintenance", Title: "Maintenance:"},
)
root.SetHelpCommandGroupID("maintenance")
root.SetCompletionCommandGroupID("maintenance")
// each subcommand: cmd.GroupID = "query" (etc.) at construction
```

### Pattern 5: fang's actual `Execute` mechanics (verified verbatim from source)
```go
// Source: charm.land/fang/v2@v2.0.1/fang.go, read directly this session
// (abridged to the parts load-bearing for D-01...D-04)
func Execute(ctx context.Context, root *cobra.Command, options ...Option) error {
    opts := settings{
        manpages:    true,   // WithoutManpage() sets this false
        completions: true,   // WithoutCompletions() sets this false
        colorscheme: DefaultColorScheme,
        errHandler:  DefaultErrorHandler,
    }
    for _, option := range options {
        option(&opts)
    }
    root.SilenceUsage = true   // already true in this repo (root.go) — no-op
    root.SilenceErrors = true  // already true in this repo (root.go) — no-op
    if !opts.skipVersion {
        root.Version = buildVersion(opts) // WithVersion(v) sets opts.version = v
    }
    root.SetHelpFunc(helpFunc) // fang OWNS help styling unconditionally when adopted

    if opts.manpages {
        root.AddCommand(&cobra.Command{Use: "man", Hidden: true, ...}) // ONLY added if manpages==true
    }
    if !opts.completions {
        root.CompletionOptions.DisableDefaultCmd = true // removes COBRA'S OWN completion cmd
    }

    if err := root.ExecuteContext(ctx); err != nil {
        w := colorprofile.NewWriter(root.ErrOrStderr(), os.Environ())
        opts.errHandler(w, makeStyles(mustColorscheme(opts.colorscheme)), err) // prints err.Error() EXACTLY ONCE
        return err
    }
    return nil
}
```
```go
// Source: charm.land/fang/v2@v2.0.1/help.go, read directly this session
func DefaultErrorHandler(w io.Writer, styles Styles, err error) {
    if w, ok := w.(term.File); ok {
        if !term.IsTerminal(w.Fd()) {
            _, _ = fmt.Fprintln(w, err.Error()) // non-TTY: bare err.Error(), one line — matches
            return                              // this repo's current main.go print exactly
        }
    }
    // TTY: styled box, still exactly one rendering of err.Error()
    ...
}
```
**Direct implication for D-03:** if fang is adopted, `cmd/codegraph/main.go` must stop calling `fmt.Fprintln(os.Stderr, err)` itself (its current, sole behavior, confirmed by reading `main.go` this session — see below) — otherwise the D-03 stub contract (`renamed_test.go`'s real-binary test, `test/integration/renamed_stubs_test.go`, asserting stderr equals the two-line message **exactly once**) breaks by double-printing.
```go
// Source: cmd/codegraph/main.go, read verbatim this session — the
// ENTIRE current file body:
func main() {
    if err := cli.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Terminal color-capability detection (NO_COLOR/CLICOLOR/CLICOLOR_FORCE/TERM=dumb/COLORTERM) | A second, hand-rolled env-precedence decision tree alongside `ChoosePresentation` | `colorprofile.Detect(w, env)` | Already implements the correct precedence (verified this session); a second hand-rolled decision-maker is exactly Pitfall 2's "two decision-makers for one question" risk PITFALLS.md already flagged |
| ANSI downsampling to a terminal's actual palette (truecolor→256→16→none) | Regex/string manipulation of lipgloss's rendered ANSI, or a bespoke SGR-sequence rewriter | `colorprofile.Writer{Forward, Profile}` | `Writer.downsample` already parses SGR sequences via `ansi.DecodeSequence` and rewrites them per-profile (verified this session, `writer.go`) — reimplementing this is exactly the "silent trap" Pitfall 1 describes |
| Adaptive light/dark color selection | A hand-rolled `if dark { ... } else { ... }` scattered through every `Render*` | `lipgloss.LightDark(isDark)` returning a `LightDarkFunc` closure, called once per palette construction | One `LightDarkFunc` closure per role keeps `present.NewPalette(dark bool)` the single place the branch exists (D-05) |
| Background-color detection | Manually issuing the OSC-11 query and parsing the response | `lipgloss.HasDarkBackground(in, out)` | Already handles raw-mode toggling, timeout, and ANSI response parsing (verified this session, `terminal.go`) — but see Common Pitfalls #2 for its real cost, which a hand-rolled version would not avoid either |
| Grouped, titled `--help` output | A fully custom help template built from scratch | `cobra.Group`/`AddGroup` (+ `present.RenderHelp` only for styling, if fang declined) | Cobra's own grouping machinery already renders titled sections; only the *styling* of an otherwise-standard layout is new work (D-14) |

**Key insight:** every piece of genuinely hard terminal-capability logic this phase touches (color detection, downsampling, background-color querying) already ships in the two pinned charm.land-family libraries this repo already partially depends on — the phase's real work is entirely in *wiring* (one shared resolver, one palette struct, ~20 RunE call sites), not in reimplementing terminal protocol handling.

## Common Pitfalls

### Pitfall 1: `colorprofile`'s `NO_COLOR` check is boolean-gated, not "any non-empty value" — verified by running real code
**What goes wrong:** CLI-03's own requirement text reads "NO_COLOR (any non-empty value disables)," matching the plain-English framing of no-color.org's spec. But `colorprofile@v0.4.3`'s actual implementation (`env.go`) is:
```go
func envNoColor(env environ) bool {
    noColor, _ := strconv.ParseBool(env.get("NO_COLOR"))
    return noColor
}
```
`strconv.ParseBool` only recognizes `1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False` — anything else (including the empty string, but also including a value like `NO_COLOR=yes` that a user might reasonably type) returns an error, and the zero value `false` is used. This was verified by executing real Go code against the pinned v0.4.3 module this session:
```
NO_COLOR="" -> ANSI256    (unset: expected)
NO_COLOR="1" -> Ascii     (disables: expected)
NO_COLOR="true" -> Ascii  (disables: expected)
NO_COLOR="banana" -> ANSI256   ← does NOT disable, contradicting "any non-empty value"
NO_COLOR="0" -> ANSI256        ← does NOT disable (ParseBool("0") is FALSE, not "non-empty")
NO_COLOR="false" -> ANSI256    ← does NOT disable
```
**Why it happens:** `colorprofile`'s author chose a strict boolean parse rather than the no-color.org spec's literal "presence and non-empty" test, likely to avoid `NO_COLOR=0`/`NO_COLOR=false` being misread as "enabled" by a less careful implementation — a defensible but spec-diverging choice.
**How to avoid:** This is a decision, not a bug fixable inside `present` (D-00 forbids re-testing colorprofile's own parsing). Two options for the resolver plan step: (a) accept `colorprofile`'s behavior as the standard and note the divergence from CLI-03's literal wording in the resolver's code comment (cheapest, matches D-09's "cited... not re-tested" instruction) — the `--color=never` rewrite already writes the canonical `NO_COLOR=1`, so this only affects a user's own raw env var; or (b) pre-normalize: if `os.Getenv("NO_COLOR") != ""`, rewrite it to `"1"` before handing the environ to `Detect`, restoring the literal "any non-empty value" semantics at the cost of one extra line of `internal/cli`-owned logic (not a colorprofile re-test — it's normalizing our own input to their function). Recorded as an Open Question below; both are reasonable, and the choice should be made explicitly rather than discovered as a support ticket.
**Warning signs:** A user reports `NO_COLOR=yes codegraph status` still shows color; a CLI-03 test case asserting `NO_COLOR=<arbitrary-truthy-looking-string>` disables colour fails unexpectedly.
**Phase to address:** The resolver plan step (D-09/D-11).

### Pitfall 2: `lipgloss.HasDarkBackground` needs both stdin AND stdout as a TTY, and can block up to 2 seconds
**What goes wrong:** Verified by reading `lipgloss/v2@v2.0.5`'s `query.go`/`terminal.go` directly this session. `HasDarkBackground(in, out term.File) bool` calls `BackgroundColor(in, out)`, which on Unix requires **both** `in` and `out` to pass `term.IsTerminal` — if either is not a terminal, it returns an error immediately and `HasDarkBackground` falls back to `true` (dark). If both ARE terminals, it puts `in` into raw mode and sends an OSC-11 background-color query, then blocks on `queryTerminal`'s read loop for up to `defaultQueryTimeout = 2 * time.Second` before giving up and (via the caller's own error handling) defaulting to dark. D-11's literal wording — "queried once... only when styled and stdout is a TTY" — checks only stdout; a real invocation with `stdout` a TTY but `stdin` redirected (e.g. `echo x | codegraph status`, or some CI/wrapper harnesses that pipe stdin even in an otherwise-interactive shell) will still attempt the query, fail fast on the stdin check, and silently default to dark — not a crash, but a silent behavior gap from the literal wording. Worse: a *genuinely* interactive terminal that simply doesn't answer OSC 11 (some terminal emulators, tmux without `set -g allow-passthrough on`, some SSH-multiplexed sessions) pays the **full 2-second stall on every single styled invocation** — not a one-time cost.
**Why it happens:** `HasDarkBackground` is designed for "ask the real terminal," which inherently requires a full-duplex TTY on both ends and a network-like round trip with an unpredictable-response terminal at the other end.
**How to avoid:** D-11's gate should explicitly become "styled AND stdout is a TTY AND stdin is a TTY" to match what `HasDarkBackground` actually needs (cheap, no library change) — this closes the "stdin redirected" gap. The 2-second worst-case latency is inherent to the library and cannot be eliminated without abandoning the live query entirely (out of scope per the locked decisions); it should be called out explicitly in the D-06/CLI-04 human-UAT checklist ("try over SSH and inside tmux — if a command visibly pauses for ~2s before printing, that is this query timing out, not a hang") so a human tester doesn't mistake it for a defect requiring a code fix.
**Warning signs:** `codegraph status` (or any styled verb) pauses for a beat before printing when run interactively over certain SSH/tmux configurations; a styled command run with `stdin < /dev/null` or in a shell with redirected stdin renders in the "wrong" (always-dark) theme with no visible error.
**Phase to address:** Resolver plan step (D-11), and the CLI-04/CLI-02 human-verification checklist items CONTEXT.md already anticipates as `human_needed`.

### Pitfall 3: `fang.WithoutCompletions()` disables cobra's OWN completion command — it does not avoid a fang-owned collision, because fang has no completion command to collide
**What goes wrong:** The milestone-level STACK.md (Context7/WebFetch-sourced, MEDIUM confidence) states fang "ships its own hidden `man` subcommand and its own `completion` subcommand by default," and D-04 accordingly plans to call both `WithoutManpage()` and `WithoutCompletions()`. Reading `fang.go` directly this session shows this is only half right:
```go
if opts.manpages {
    root.AddCommand(&cobra.Command{Use: "man", ...})  // fang DOES add its own competing "man" command
}
if !opts.completions {
    root.CompletionOptions.DisableDefaultCmd = true    // fang does NOT add a competing command —
}                                                        // it just toggles cobra's OWN flag that
                                                          // controls whether COBRA creates its
                                                          // default completion command
```
`cobra.CompletionOptions.DisableDefaultCmd`'s doc comment (read directly from `cobra@v1.10.2/completions.go` this session) is unambiguous: "prevents Cobra from creating a default 'completion' command." There is no separate fang-owned completion command anywhere in `fang.go`/`help.go`. Calling `fang.WithoutCompletions()` therefore does not "avoid a collision with cobra's own completion command" — it **deletes cobra's own completion command outright**, the same command GoReleaser's `generate_completions_from_executable` step depends on (per STACK.md's own "What NOT to Use" table, which correctly identifies the GoReleaser dependency but incorrectly attributes the fix).
**Why it happens:** The name `WithoutCompletions` reads as "fang's completions feature," inviting the assumption fang owns a completions command the way it owns the `man` command; the milestone-level research's Context7/WebFetch pass evidently did not read `fang.go`'s actual `Execute` body closely enough to catch the asymmetry between the two `Without*` options' real mechanics.
**How to avoid:** During the D-01…D-04 spike, verify empirically (not just by reading source) that `codegraph completion bash` still works when fang is wired in **without** calling `WithoutCompletions()` — the correct call set is very likely `fang.Execute(ctx, root, fang.WithoutManpage(), fang.WithVersion(...))`, omitting `WithoutCompletions()` entirely, since leaving `opts.completions` at its default `true` means fang does nothing to cobra's completion machinery at all. If the spike's own criterion 1 ("`WithoutManpage()`/`WithoutCompletions()` compose with the existing hidden `man` and cobra completions") is read as "the shipped call set must include both," that criterion is now understood to be internally contradictory for `WithoutCompletions()` specifically — worth a maintainer note in `04-FANG-VERDICT.md` regardless of which way the verdict ultimately lands, since it changes what "compose" means for this one option.
**Warning signs:** `codegraph completion bash` (or zsh/fish/powershell) stops working, or GoReleaser's completion-generation step fails, immediately after a fang adoption that calls `WithoutCompletions()`.
**Phase to address:** The D-01…D-04 fang spike, before `04-FANG-VERDICT.md` is written.

### Pitfall 4 (inherited from PITFALLS.md, re-confirmed with exact mechanism): the archtest's fixed-list denylist is real and already correctly scoped for widening
**What goes wrong:** `import_graph_test.go`'s `forbiddenImportPaths` (`charm.land/lipgloss/v2`, `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`) is a 3-entry literal list, confirmed by reading the file this session. `colorprofile` lives at `github.com/charmbracelet/colorprofile` — a **different vanity root** than `charm.land/...` — so D-15's plan to prefix-match on **both** `charm.land/` and `github.com/charmbracelet/` is necessary, not optional: a prefix match on `charm.land/` alone would still miss `colorprofile`/`x/ansi` entirely.
**Separately, and worth distinguishing clearly:** `charm_cgo_test.go`'s `TestCharmCgoClosure` scopes its own scan to `charmClosurePathPrefix = "charm.land"` only (confirmed by reading the file this session) — this is a **different, narrower guard** for a different property (no new CGo in the charm closure), and D-02 only requires it stay **green**, not that its own prefix be widened to match D-15's `import_graph_test.go` change. Since `colorprofile`/`x/ansi` are pure Go (confirmed via the `go get` executed this session — no CGo-suggestive packages in fang's own transitive closure either), `TestCharmCgoClosure` staying green needs no code change; conflating the two guards' scopes during implementation would be a scope-creep risk worth flagging in review.
**How to avoid:** Implement D-15's prefix-widening in `import_graph_test.go` only; leave `charm_cgo_test.go`'s `charmClosurePathPrefix` untouched unless a future dependency genuinely introduces CGo outside the `charm.land` prefix (not the case for anything this phase adds).
**Phase to address:** The archtest+colorprofile-promotion commit (D-15).

## Code Examples

### `colorprofile.Detect`'s actual precedence order (verbatim from source, v0.4.3)
```go
// Source: github.com/charmbracelet/colorprofile@v0.4.3/env.go — read directly
// this session. Cite this comment (with the pinned version) in colorflag.go
// per D-09/D-00; do not re-derive or re-test this logic.
//
// Detect's documented rules:
//   - TERM=dumb is always treated as NoTTY unless CLICOLOR_FORCE=1 is set.
//   - If COLORTERM=truecolor, and the profile is not NoTTY, it gets upgraded to TrueColor.
//   - TERM=xterm-256color -> ANSI256; TERM=xterm-color -> ANSI.
//   - CLICOLOR=1 without TERM defined is treated as ANSI if output is a terminal.
//   - NO_COLOR takes precedence over CLICOLOR/CLICOLOR_FORCE (colour disabled,
//     but text decoration like bold/faint/underline is NOT disabled).
//   - NO_COLOR itself is strconv.ParseBool-gated — see Common Pitfalls #1.
func Detect(output io.Writer, env []string) Profile
```

### The `--color` environ-rewrite shape (D-09), built against verified signatures
```go
// Illustrative — the exact function name/signature is Claude's Discretion
// (D-09's own note), but the shape below is what the verified APIs require.
func resolveColorProfile(cmd *cobra.Command, colorFlag string) colorprofile.Profile {
    out := cmd.OutOrStdout()
    environ := os.Environ()
    switch colorFlag {
    case "always":
        environ = withoutEnv(environ, "NO_COLOR", "CLICOLOR")
        environ = append(environ, "CLICOLOR_FORCE=1")
    case "never":
        environ = append(environ, "NO_COLOR=1") // canonical ParseBool-true value
    case "auto":
        // pass through untouched
    }
    return colorprofile.Detect(out, environ) // single environment read, D-09
}
```

### The fang double-print risk, concretely (D-03)
```go
// If fang is adopted, cmd/codegraph/main.go's CURRENT body (verified
// verbatim this session) must change from:
func main() {
    if err := cli.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err) // duplicates fang's own DefaultErrorHandler print
        os.Exit(1)
    }
}
// to:
func main() {
    if err := cli.Execute(); err != nil { // cli.Execute() now calls fang.Execute internally,
        os.Exit(1)                         // which already printed err.Error() exactly once
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| lipgloss v1's `Renderer`/`termenv`-based automatic profile detection baked into `Style.Render()` | lipgloss v2: `Style.Render()` always emits full-fidelity ANSI; downsampling is an explicit, separate `colorprofile` step | lipgloss v2 (already the pinned version in this repo) | Already fully accounted for in D-05/D-10/Pitfall 1 of PITFALLS.md; this research only adds the exact mechanism, not a new finding |
| `charmbracelet/fang` (pre-v2, `github.com/charmbracelet/fang`) | `charm.land/fang/v2` — same charmbracelet org, moved to the `charm.land` vanity domain, v2.0.1 current | Some time before this research date; confirmed current via live `go get` this session | Matches this repo's existing lipgloss/bubbletea/bubbles `charm.land/.../v2` vanity-import convention already noted in `import_graph_test.go`'s own "Finding 1" comment |

**Deprecated/outdated:**
- `fang.WithTheme(theme)` is explicitly marked `// Deprecated: use [WithColorSchemeFunc] instead` in the v2.0.1 source read this session — if fang is adopted and any theme customization beyond the default is wanted, use `WithColorSchemeFunc(func(lipgloss.LightDarkFunc) ColorScheme) Option`, not `WithTheme`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The exact seven-role hex palette proposed for `present.NewPalette` is Claude's Discretion per D-06 and is explicitly gated on human UAT (Solarized Light + a dark theme) — no specific hex values are asserted as correct in this document, by design | Architecture Patterns / D-06 | None — this is deliberately deferred to the implementation plan + human checkpoint, not a research gap |
| A2 | `colorprofile.Detect`'s and `lipgloss.HasDarkBackground`'s behavior is stable across the pinned versions verified this session and will not change if a patch bump occurs mid-phase | Common Pitfalls #1, #2 | Low — both are point-released, mature libraries; re-verify if `go.mod` bumps either during the phase |
| A3 | The correct fang option set for the D-04 verdict, once corrected per Common Pitfalls #3, is `WithoutManpage()` + `WithVersion(...)` with `WithoutCompletions()` omitted — this is a strong inference from reading `fang.go`'s source, not yet confirmed by actually running `codegraph completion bash` under a fang-wrapped binary | Common Pitfalls #3, User Constraints correction | Medium — if wrong, the spike would surface it immediately (completions either work or don't); flagged explicitly so the spike verifies this empirically rather than trusting the source-reading inference alone |

**If this table is empty:** N/A — see above; none of these need user confirmation before planning proceeds (A1 is an intentional deferral already designed into D-06, A2/A3 are self-verifying during phase execution).

## Open Questions

1. **Should the resolver normalize `NO_COLOR` to restore "any non-empty value disables" semantics, or accept `colorprofile`'s ParseBool-gated behavior as-is?**
   - What we know: `colorprofile@v0.4.3`'s actual behavior (verified by execution this session) only treats a `strconv.ParseBool`-truthy `NO_COLOR` value as disabling colour; CLI-03's own wording says "any non-empty value."
   - What's unclear: whether the discuss-phase/planning process wants this documented-and-accepted (cheapest, D-00-aligned) or normalized in the resolver (one extra line, restores literal spec compliance for a user's raw env var — not a re-test of colorprofile, since it normalizes *our* input before handing it to their function).
   - Recommendation: default to "document and accept" (matches D-09's "cited... not re-tested" instruction most directly) unless the planner or a reviewer judges strict no-color.org compliance for arbitrary `NO_COLOR` values as load-bearing; either way, put one sentence in the resolver's code comment citing this research so the choice is deliberate, not accidental.

2. **Does the D-01…D-04 fang verdict's criterion 1 need its wording corrected before `04-FANG-VERDICT.md` is written?**
   - What we know: `WithoutCompletions()` does not avoid a fang-owned collision (fang has none); it disables cobra's own completion command, which this repo needs.
   - What's unclear: whether the spike should therefore test "fang adopted with `WithoutManpage()` only" as the actual candidate configuration, rather than literally attempting `WithoutManpage()` + `WithoutCompletions()` and discovering completions break.
   - Recommendation: the plan for the fang-spike task should explicitly test the corrected option set (`WithoutManpage()` alone, or with `WithVersion`) and record in `04-FANG-VERDICT.md` that `WithoutCompletions()` was deliberately not called and why — citing this research.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go's stdlib `testing` package (`go test`) — no third-party test framework anywhere in `internal/cli`/`internal/cli/present` |
| Config file | none — plain `go test ./...` per package; `test/wireoracle` and `test/integration` are separate real-binary harnesses invoked via dedicated Taskfile targets |
| Quick run command | `go test ./internal/cli/... ./internal/cli/present/...` |
| Full suite command | `go test ./...` (unit) plus `task test:wireoracle` (real-binary MCP transcript oracle) plus `go test ./test/integration/...` (real-binary CLI harness, includes `renamed_stubs_test.go`) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CLI-01 | Every styled verb's plain path is byte-identical to its pre-glow-up golden | unit (golden fixture compare) | `go test ./internal/cli/... -run TestPlainGolden` | ❌ Wave 0 — `testdata/plain/<verb>.golden` fixtures + the comparison test are D-16's first commit, new this phase |
| CLI-02/CLI-03 | `--color` × TTY × NO_COLOR/CLICOLOR* resolver matrix | unit | `go test ./internal/cli/... -run TestResolveColor` | ❌ Wave 0 — `colorflag.go` + `colorflag_test.go` are new this phase |
| CLI-05 | `serve --mcp` byte-identical transcript under any adopted wrapper | real-binary (spawns built binary) | `task test:wireoracle` | ✅ — 38 frozen transcripts already exist; the fang spike (D-01) reuses this run unmodified |
| CLI-05 | `NO_COLOR`+non-TTY plain-output regression (the gh #13335 lesson) | unit | `go test ./internal/cli/... -run TestNoColorNonTTYRegression` | ❌ Wave 0 — new test, folds into the D-16 golden-freeze work |
| CLI-06 | Every visible command carries a `GroupID`, none groupless | unit | `go test ./internal/cli/... -run TestEveryCommandHasGroupID` | ❌ Wave 0 — new guard, mirrors `cli_reference_test.go`'s existing walk-the-tree pattern |
| CLI-07 | `-j`/`-l`/`-k`/`-p` present wherever the long form exists | unit (table test) | `go test ./internal/cli/... -run TestShortFlagsConsistent` | ❌ Wave 0 — pins an already-true invariant (D-12), new pinning test |
| CLI-08/D-03 | Rename-stub stderr contract holds under any adopted wrapper | real-binary | `go test ./test/integration/... -run TestRenamedStubsPrintExactlyOnce` | ✅ — already exists (`test/integration/renamed_stubs_test.go`), reused unmodified as the D-03 spike criterion |
| GRD-13 | `present` archtest catches every charm-family import path added | unit (import-graph walk) | `go test ./internal/cli/present/archtest/... -run TestNoCharmInServeReachablePackages` | ✅ — exists, needs `forbiddenImportPaths` widened to prefix-match (D-15); RED-proof via a planted import, not a new test file |

### Sampling Rate
- **Per task commit:** `go test ./internal/cli/... ./internal/cli/present/...`
- **Per wave merge:** `go test ./... && task test:wireoracle`
- **Phase gate:** Full suite green (including `test/integration` and `test:wireoracle`) before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/cli/testdata/plain/*.golden` — plain-output goldens for every verb gaining a styled branch (D-16), captured BEFORE any renderer lands
- [ ] `internal/cli/colorflag_test.go` — the D-09/CLI-03 resolver matrix
- [ ] `internal/cli/cli_reference_test.go`-adjacent new test for `GroupID` coverage (CLI-06) — likely extends the existing file rather than a new one, given its established walk-the-tree pattern
- [ ] `internal/cli/short_flags_test.go` (or similar) — the CLI-07 table-test pinning `-j`/`-l`/`-k`/`-p`

*(No framework install needed — `go test` is already the toolchain-native framework in use throughout this repo.)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | This phase touches only local CLI rendering/help; no auth surface |
| V3 Session Management | No | N/A |
| V4 Access Control | No | N/A |
| V5 Input Validation | Yes | `--color` flag value validated against a closed enum (`auto\|always\|never`) with a usage error on anything else (D-11) — reuses cobra's own flag-validation idiom, no new parser |
| V6 Cryptography | No | N/A |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Adversarial terminal-escape injection via user-controlled path/name strings reaching a styled render | Tampering | Already mitigated and unchanged this phase: `present/sanitize.go`'s `sanitizeControl` (confirmed in use at `status.go`'s `RenderStatus`, read this session — strips control characters from `projectPath`/`WorktreeMismatch` fields before they reach the terminal); the glow-up's new `Render<Verb>` functions must follow the same discipline for any user-supplied string (symbol names, file paths, search terms) that flows into a styled line, per the existing `CR-01` precedent already established in `present/status.go` |
| Fang's `errHandler`/help chrome rendering an attacker-influenced error string | Tampering / Information Disclosure | Not a new risk this phase introduces — `err.Error()` strings already flow to stderr today via `main.go`'s own print; fang's `DefaultErrorHandler` renders the identical string, just through lipgloss styling on a TTY. No new attacker-controlled data path is created. |
| A malicious/compromised terminal emulator answering the OSC-11 background-color query with crafted data | Tampering | Out of this repo's control surface — `lipgloss.HasDarkBackground`'s parsing of the terminal's response is third-party library behavior (D-00: not re-tested); the worst case is a wrong light/dark choice (cosmetic), not a security boundary violation, since the result only selects which of two pre-defined hex colors to use, never executes or evaluates the response as code |

## Sources

### Primary (HIGH confidence — read directly from the pinned module's source this session)
- `/Users/sean/go/pkg/mod/github.com/charmbracelet/colorprofile@v0.4.3/env.go`, `writer.go` — `Detect`/`Env`/`NewWriter`/`Writer.Write`/`downsample`, exact NO_COLOR/CLICOLOR/CLICOLOR_FORCE precedence; behavior additionally confirmed by executing real Go code against this exact pinned version this session
- `/Users/sean/go/pkg/mod/charm.land/lipgloss/v2@v2.0.5/color.go`, `query.go`, `terminal.go` — `LightDark`/`LightDarkFunc`, `HasDarkBackground`/`BackgroundColor`/`queryBackgroundColor`, the 2-second `defaultQueryTimeout` and dual-TTY requirement
- `/Users/sean/go/pkg/mod/github.com/spf13/cobra@v1.10.2/command.go` — `Group`, `GroupID` field, `AddGroup`, `SetHelpFunc`, `SetHelpCommandGroupID`, `SetCompletionCommandGroupID`
- `/Users/sean/go/pkg/mod/github.com/spf13/cobra@v1.10.2/completions.go` — `CompletionOptions.DisableDefaultCmd`'s doc comment ("prevents Cobra from creating a default 'completion' command")
- `/Users/sean/go/pkg/mod/charm.land/fang/v2@v2.0.1/fang.go`, `help.go` — full `Execute`/`settings`/`Option` surface, `DefaultErrorHandler`, `WithoutManpage`/`WithoutCompletions`/`WithVersion`/`WithColorSchemeFunc`/`WithTheme` (deprecated)/`WithErrorHandler`/`WithoutVersion`/`WithCommit`/`WithNotifySignal`
- This repo, read directly this session: `internal/cli/{root,status,search,node,explore,progress_cli,renamed,cli_test}.go`, `internal/cli/install.go` (`printAgentResults`), `internal/cli/cli_reference_test.go` (`inheritedFromAncestor`, allowlist parsing), `internal/cli/testdata/cli-reference-allowlist.txt`, `internal/cli/present/{styles,tty}.go`, `internal/cli/present/archtest/{import_graph_test,charm_cgo_test}.go`, `internal/query/detail.go` (`ExploreDetail`/`NodeDetail` signatures), `internal/query/{explore,node}.go` (confirming `Explore`/`Node` are the markdown-only methods), `cmd/codegraph/main.go`, `test/integration/renamed_stubs_test.go`, `Taskfile.yml` (`test:wireoracle`, `docs:cli`, `docs:cli:drift`, `govulncheck`/`GOTOOLCHAIN` conventions), `.planning/phases/03-verb-fold/03-MUTATION-LOG.md` (mutation-log shape/convention), `go.mod`
- Live command execution this session: `go get charm.land/fang/v2@latest` + `go list -m charm.land/fang/v2` (resolved `v2.0.1` against the real module proxy); a scratch Go program run against the pinned `colorprofile@v0.4.3` confirming its exact `NO_COLOR` parsing behavior; `GOTOOLCHAIN=go1.26.6 govulncheck ./...` (baseline: 0 vulnerabilities found in code that is actually called — 3 in imported-but-uncalled packages, pre-existing and unrelated to this phase)

### Secondary (MEDIUM confidence)
- `.planning/research/{STACK,PITFALLS,ARCHITECTURE}.md` (milestone-level research, dated 2026-09-14) — comprehensive verb-by-verb seam table and pitfall catalogue, Context7/WebFetch-sourced; this document's Common Pitfalls #3 and the User Constraints correction identify one place (fang's `WithoutCompletions()` mechanics) where that pass's medium-confidence sourcing needed a source-level correction

### Tertiary (LOW confidence)
- None used in this document — every API claim above was either read from the pinned module's source directly or confirmed by executing real code against it this session.

## Metadata

**Confidence breakdown:**
- Standard stack (exact API signatures, versions): HIGH — every signature read directly from the module cache or confirmed via live command execution
- Architecture (RunE idiom, resolver placement, verb-by-verb seams): HIGH for the parts re-verified this session (search/node/explore/install/status/progress_cli/root/renamed all read directly); the milestone-level ARCHITECTURE.md's broader verb table is inherited at its own (also HIGH, source-read) confidence
- Pitfalls: HIGH for the three newly-verified findings (NO_COLOR ParseBool gating, HasDarkBackground dual-TTY+2s-timeout, fang WithoutCompletions mechanics) — each confirmed by reading source and/or executing real code, not inferred

**Research date:** 2026-09-17
**Valid until:** 30 days for the cobra/lipgloss/colorprofile findings (stable, already-pinned libraries); re-verify the fang findings specifically if the phase's own fang-spike task observes different behavior than this document predicts, since fang is a new, not-yet-integrated dependency for this repo and its v2.0.1 pin could move before the phase executes
