# Architecture Research — v0.14.0 Polish & Agent Reach

**Domain:** Brownfield integration — CLI styling glow-up, verb-surface fold, multi-harness agent reach, Codex parity, on an existing shipped Go codebase.
**Researched:** 2026-09-14
**Confidence:** HIGH for CLI/present/MCP/agents mechanics (read from source with line evidence); MEDIUM for Codex CLI precedence claims and `charm.land/fang/v2` runtime behavior (web-sourced, not yet run against this repo).

## Standard Architecture (today, before this milestone)

```
┌───────────────────────────────────────────────────────────────────────┐
│ cmd/codegraph/main.go — cli.Execute() → os.Exit(1) on error            │
├───────────────────────────────────────────────────────────────────────┤
│ internal/cli  (24 visible verbs, one file each, cobra RunE)            │
│   RunE reads TTY/NO_COLOR/flags → chooses plain vs styled branch       │
│   ┌─────────────┐   ┌──────────────────┐   ┌────────────────────┐     │
│   │ present/     │   │ tui/ (bubbletea) │   │ agents/ (8 targets) │     │
│   │ lipgloss v2  │   │ agentpicker      │   │ install/uninstall   │     │
│   │ D-01 sole    │   │ daemonpicker     │   │ AgentTarget iface   │     │
│   │ lipgloss home│   │                  │   │                    │     │
│   └──────┬───────┘   └──────────────────┘   └─────────┬──────────┘     │
├──────────┼──────────────────────────────────────────────┼──────────────┤
│          ▼ consumes plain structs, never recomputes      ▼ writes ext. │
│ internal/query.Engine — SOLE read-only surface (Query/Search/Node/     │
│   Explore/Callers/Callees/Impact/Affected/Files/Status/FileGraph)      │
│   ExploreDetail/NodeDetail = plain-struct seams (v0.12 Phase 1, "third │
│   consumer never a second gather path") already shared by CLI/MCP/UI   │
├───────────────────────────────────────────────────────────────────────┤
│ internal/mcp (stdio) ── internal/uiserver (ConnectRPC) — never import  │
│ charm; TUI-01 archtest (present/archtest) proves the 6-package closure │
│ {mcp, graphstore, daemon, watch, indexer, query} is charm-free         │
└───────────────────────────────────────────────────────────────────────┘
```

`internal/cli/present` today has exactly 3 style vars (`headerStyle`, `labelStyle`, `sectionStyle`, `present/styles.go:19-28`) and is consumed by 3 call sites: `status.go:86-88`, `files.go:72-73`, `progress_cli.go:31-34`. Every other RunE prints via bare `fmt.Fprintf` loops over already-computed structs.

## Component Responsibilities (this milestone's new/changed pieces)

| Component | Responsibility | New or Modified |
|-----------|-----------------|------------------|
| `internal/cli/present/*.go` (new `Render<Verb>` funcs) | Styled rendering of every verb's plain struct, mirroring `RenderStatus`/`RenderFiles` | **New** ~15 files, same package |
| `internal/cli/colorflag.go` (proposed) | Resolves `--color`/TTY/`NO_COLOR` into `present.ChoosePresentation`'s two args, once, shared by every RunE | **New** |
| `internal/cli/search.go` (existing, extended) | `--full` flag selects `Engine.Query` vs `Engine.Search` shape | **Modified** |
| `internal/cli/query.go` | Becomes a `Hidden` non-zero-exit stub naming `search --full` | **Modified**, behavior-breaking |
| `internal/cli/unlock.go` | Becomes a `Hidden` non-zero-exit stub naming `daemon unlock`; new `daemon unlock` subcommand added to `daemon.go` | **Modified + new subcommand** |
| `internal/agents/types.go` `AgentTarget` | Grows capability-describing methods (skill dir, hook mechanism) so 8 targets stop needing 8 bespoke re-implementations | **Modified interface**, additive |
| `internal/agents/codex.go` | Grows project-local scope, skill-dir install, `AGENTS.md` repo-root block (already has global `AGENTS.md`), possible Codex hook mechanism | **Modified** |
| `.claude/hooks/hooks.json` + `claudeassets.go` | New `PreToolUse` block alongside existing `SessionStart` block | **Modified** (additive embed) |
| `internal/mcp/server.go` `instructions` const | Unaffected by the verb fold (no CLI verb names in it today) but should gain "Codex has a skill too" parity language if the Claude-only carve-out (`instructions.go:20-27`) is revisited | **Possibly modified, low risk** |

## Part 1 — The CLI glow-up

### Where colour enters (D-03 rule, already precedented)

The rule the milestone states — "chosen at the RunE call site from TTY + `NO_COLOR` + `--color`, passed in rather than read from env inside `present`" — is not a new invention, it is the **existing, enforced pattern**. `present.ChoosePresentation(isTTY bool, noColor string) bool` (`present/tty.go:10-12`) is already pure and already documented as forbidden from reading `os.Getenv`/`term.IsTerminal` itself (`present/styles.go:7-9`). The 3 existing call sites each do:

```go
// status.go:86, files.go:72, progress_cli.go:31 — same 2-line idiom, 3x duplicated
if present.ChoosePresentation(term.IsTerminal(int(os.Stdout.Fd())), os.Getenv("NO_COLOR")) {
    return present.RenderStatus(result, start, cmd.OutOrStdout())
}
```

**Integration point for `--color`:** do not change `ChoosePresentation`'s signature (it is a stable, tested, pure 2-arg function used identically by 3 call sites today — widening it fans out to every caller and to `progress_cli.go`'s stderr-fd variant, which has different TTY semantics). Instead add one shared resolver in `internal/cli` (new `colorflag.go`, or folded into `root.go`) that takes `(cmd *cobra.Command, fd uintptr) (isTTY bool, noColor string)` and pre-empts the two inputs when `--color` is explicit:

```go
// proposed, internal/cli — NOT in present (present stays env-blind, D-03)
func resolveColor(cmd *cobra.Command, fd uintptr) (isTTY bool, noColor string) {
    isTTY = term.IsTerminal(int(fd))
    noColor = os.Getenv("NO_COLOR")
    switch colorFlag { // persistent string flag on root, "auto"|"always"|"never"
    case "always":
        return true, ""
    case "never":
        return false, "1"
    }
    return isTTY, noColor
}
```

This keeps `present` at zero new inputs and zero new tests to re-verify D-03/D-04 (`present/tty_test.go`); only the 20+ RunE call sites change their 2-line preamble to `resolveColor(cmd, os.Stdout.Fd())`. `--color` should be a **persistent flag on root** (`root.go:44-54`, alongside `Version`/`SilenceUsage`) so every subcommand inherits it without per-verb flag registration — this also means it becomes one of the flags `TestEveryRegisteredFlagIsAccountedFor` (`cli_reference_test.go`) walks once via `PersistentFlags().VisitAll` at root and never re-attributes to children (the existing `inheritedFromAncestor` guard at `cli_reference_test.go:66-79` already handles this correctly for a new persistent flag with zero changes needed to the guard itself).

### Palette, not just chrome

Today's 3 styles (`Bold`, `Faint`, `Bold+Underline`) are monochrome — the milestone wants real colour. Add the palette as more `lipgloss.NewStyle()` vars in `present/styles.go` (e.g. `successStyle`, `warnStyle`, `errStyle`, `accentStyle` with `.Foreground(lipgloss.Color(...))`, matching `progress.go:13`'s existing `progressStyle.Foreground(lipgloss.Color("212"))` precedent) — this is additive to an already-established file, not a new architectural seam.

### How 20+ plain-fmt verbs get a renderer each, without touching `internal/query` or MCP

This is the milestone's central technical risk, and the codebase already answers it via a pattern proven twice (`RenderStatus` over `query.StatusResult`, `RenderFiles` over `query.FilesResult`): **every verb's plain branch already consumes a plain Go struct that `present` can also consume — the struct boundary already exists, `present` just needs to grow one `Render<X>` function per struct.**

Verb-by-verb evidence of what already exists to render against:

| Verb | Engine call already made in RunE | Struct/string to style | Evidence |
|------|-----------------------------------|--------------------------|----------|
| `status` | `eng.Status(ctx)` | `query.StatusResult` | already has `present.RenderStatus` |
| `files` | `eng.Files(opts)` | `query.FilesResult` | already has `present.RenderFiles` |
| `search`/`query` (post-fold) | `eng.Query`/`eng.Search` | `[]*schema.Node` / `[]query.Location` | `query.go:61`, `search.go:39` |
| `callers` | `eng.Callers(symbol, limit)` | `query.CallersResult` | `callers.go:52-55` loops `.Callers` |
| `callees` | `eng.Callees(symbol, limit)` | `query.CalleesResult` | `callees.go:53-56` |
| `impact` | `eng.Impact(symbol, depth)` | `query.ImpactResult` | `impact.go:54-58` |
| `affected` | `eng.Affected(files, depth)` | `query.AffectedResult` | `affected.go:136-143` |
| `explore` | **currently** `eng.Explore(q, maxFiles)` → markdown `string` | **available instead:** `eng.ExploreDetail(q, maxFiles)` → `query.ExploreResult` (`explore.go:231-238` shows `Explore` is a thin wrapper: `buildExploreResult` → `RenderExplore(...)`) | `query/detail.go:306,681` |
| `node` | **currently** `eng.Node(symbol,file,line)` → markdown `string` | **available instead:** `eng.NodeDetail(...)` → `query.NodeDetail` (mode-switched: File/SingleDef/MultiDef) (`node.go:337-357` shows the same thin-wrapper shape) | `query/detail.go:190,250` |
| `init`/`index`/`sync`/`daemon start` | progress + final summary line | already TTY-gated via `present.NewProgress` (`progress_cli.go`) | styling is header/summary text only, no new struct |
| `install`/`uninstall` | `agents.WriteResult` per target | new `present.RenderAgentResults` over `[]agents.WriteResult` | `install.go:139-165` (`printAgentResults`) is the exact seam to branch inside |

**The critical finding:** `explore` and `node` do **not** need a new `internal/query` surface. `Engine.Explore`/`Engine.Node` were already refactored in v0.12.0 Phase 1 (Key Decision: *"A new consumer gets a seam, never a second gather path"*, PROJECT.md) into thin wrappers over `ExploreDetail`/`NodeDetail`, specifically so the UI (`internal/uiserver`) could consume the same plain structs `Explore`/`Node`'s markdown renderers consume. The CLI glow-up's `present.RenderExplore`/`present.RenderNode` should call `eng.ExploreDetail`/`eng.NodeDetail` directly in the styled branch — exactly the same shape `status.go` already uses (`eng.Status()` once, then branch between `present.RenderStatus` and `query.RenderStatusText` on the same result). **Zero changes to `internal/query`, zero changes to `internal/mcp`** — this is a third consumer of an already-built seam, not a fourth gather path.

For `install`/`uninstall`, the natural boundary is `printAgentResults` (`install.go:139`) — thread a `render func(agents.WriteResult) string` (or branch inside the loop) the same way `status.go` branches, keeping `agents.WriteResult` as the shared plain struct both plain and styled paths consume.

### Golden oracle and CLI-REFERENCE drift-gate constraints

- **Wire oracle** (`test/wireoracle`, 38 frozen transcripts) freezes MCP JSON-RPC traffic — tool names, descriptions, resource content. The glow-up touches **only** `internal/cli`; MCP never imports charm (TUI-01 archtest, `present/archtest/import_graph_test.go:39-46`), so the glow-up cannot touch a single wire-oracle transcript **as long as no styled output leaks into a code path the MCP server also executes** (see the fang risk below — this is the one place that boundary could be breached).
- **CLI golden fixtures**: no literal per-verb `query.json`/`search.json`/etc. golden files were found under `testdata/golden` or `internal/query/testdata` in this pass — the "golden" referenced by `query.go`'s doc comment (`query.MarshalQueryJSON`) is the `--json` envelope shape, asserted by unit tests inline rather than a frozen fixture tree. **This must be re-verified at plan time** (`rg -l MarshalQueryJSON` across `_test.go` files) before assuming no fixture needs updating, but the styling work itself never touches `--json` output — every `present.Render*` call in every verb is gated **after** the `if jsonOut { ... }` early return (the existing pattern in every RunE, e.g. `query.go:66-72` before `status.go:86`), so `--json` and the agent/MCP-adjacent non-TTY plain path are structurally unreachable from the new styled branch.
- **`docs/CLI-REFERENCE.md` drift gate** (`tools/clidoc/main.go`, `cli_reference_test.go`): generated from `cobra/doc` output over the live tree; adding a persistent `--color` flag, `search --full`, and `daemon unlock` all flow through automatically on the next `task docs:cli` regen. The one manual step: `TestEveryRegisteredFlagIsAccountedFor`'s allowlist (`testdata/cli-reference-allowlist.txt`, D-08 format `<command path>\t<reason>`) needs one new command-level entry per hidden renamed-verb stub (`query` and `unlock`), because `documentedByReference` (`cli_reference_test.go:57-64`) returns `false` for a `Hidden: true` command, and every hidden command still gets a `help` flag via `cmd.InitDefaultHelpFlag()` (`cli_reference_test.go:178`) that must land somewhere in the accounting.

### Fang: where it can live, and the risk that needs proving

`charm.land/fang/v2` exists (matches this repo's already-adopted `charm.land/.../v2` vanity-import family for `lipgloss`/`bubbletea`/`bubbles`, unlike the plain `github.com/charmbracelet/fang` path — confirm the `charm.land` variant is current before pinning, per the same Finding-1 caution the archtest's own comment records for lipgloss, `present/archtest/import_graph_test.go:9-13`). Fang is a **process-level** wrapper: `fang.Execute(ctx, root, ...)` replaces `cobra.Command.Execute()` itself, styling `--help`, error output, and `--version`, and typically owns `os.Exit`.

This changes the integration point from "one more `present` call inside a RunE" to **`cmd/codegraph/main.go`'s single call site** (`main.go:12-16`, currently `cli.Execute()` → manual `fmt.Fprintln(os.Stderr, err); os.Exit(1)`), or `internal/cli.Execute()` itself (`root.go:68-70`). Two consequences the archtest boundary does **not** currently cover:

1. **`internal/cli` is excluded from the TUI-01 closure scan by design** (`present/archtest/import_graph_test.go:73-75`, comment: *"it is also the package tree ... that legitimately owns the sole charm import ... it must not be treated as part of the serve-reachable closure"*). A fang import in `root.go`/`main.go` is therefore **invisible** to `TestNoCharmInServeReachablePackages` today — not a violation, but not a proof of safety either, since that test was never designed to reason about the *entry point* `serve --mcp` shares with every other verb.
2. **`serve --mcp` and `daemon start` dispatch through the exact same `Execute()`/`fang.Execute()` call** as every styled verb. Fang's documented behavior only intercepts help/usage/error rendering, not a command's own `RunE` stdout — but this is a claim from web documentation (MEDIUM confidence), not yet verified against this binary. Before adopting fang at the `Execute()` boundary, the phase that does so must add a positive proof (a wire-oracle-style transcript re-capture of `serve --mcp`'s stdout before/after the fang wrap, byte-identical) — this is a **new guard**, not an amendment to the existing archtest, because the existing archtest's scope (a static import-closure walk) cannot express "and no runtime output changed on the happy path."

**Recommendation:** either (a) confine fang to `--help`/error rendering only via its narrower API surface (styling functions, not `fang.Execute` replacing the whole dispatch) if such an API exists, keeping `main.go`'s existing error-handling contract (`SilenceUsage`/`SilenceErrors`, `main.go:12-16`) intact, or (b) adopt `fang.Execute` at the `main.go` boundary and add the byte-identical `serve --mcp` transcript proof as a first-class task in the same plan. Do not treat "fang wraps Execute" as a drop-in with no verification burden — this is the one place in the glow-up where the CLI/MCP boundary genuinely widens.

### The hidden `man` verb under fang / `cobra/doc`

`man` is already `Hidden: true` (`man.go:50`) and calls `doc.GenManTree(newRootCmd(), ...)` (`man.go:61`) — it builds its own fresh, **un-executed** root tree, so it never passes through `cli.Execute()`/`main.go` at all; it is invoked directly by the Homebrew cask's post-install hook as a subcommand RunE. Fang wrapping `Execute()` therefore has **no interaction with `man`** — `man`'s own `RunE` still runs cobra's normal dispatch either way, and `cobra/doc`'s `GenManTree`/`GenMarkdownCustom` (also used by `tools/clidoc`) read the command tree's metadata (`Use`, `Short`, `Long`, flags) which fang does not rewrite — fang only wraps *rendering at execution time*, not the static command metadata `cobra/doc` walks. `documented()`'s `IsAvailableCommand()` filter (`clidoc/main.go:61-63`) already excludes `Hidden` commands from both `man` and the CLI-REFERENCE walk identically, so the renamed-verb stubs (also `Hidden: true`) get the same free exclusion `man` already enjoys.

## Part 2 — The verb fold

### `query` → `search --full`

`Engine.Query(term, kind, limit) []*schema.Node` and `Engine.Search(term, kind, limit) []Location` already share `matchNodes` (`query/search.go:74,119,152` — confirmed: both call `e.matchNodes(term, kind)` and differ only in what they project from the ranked result). The fold is mechanical:

- `search.go` (`search.go:18-71`) gains a `--full` bool flag. When set, RunE calls `eng.Query(...)` instead of `eng.Search(...)`, and the human branch prints the "what it discards today" fields the milestone calls out (full node records vs. locations-only) — `query.go`'s current human loop already prints `n.Name, n.Kind, n.FilePath, n.StartLine` (`query.go:81-83`), identical to `search.go`'s (`search.go:58-60`); `--full`'s value-add is printing the *additional* fields `query.Node` carries that `Location` does not (signature/body preview) — the milestone's own framing ("finally shows what it discards today") implies the human branch should get richer, not just re-labeled.
- `--json` under `--full` should emit `query.MarshalQueryJSON` (the exact function `query.go:67` already calls) instead of the bare `json.Marshal(locs)` `search.go:45` uses today — this is the one place the two verbs' `--json` envelopes genuinely differ in shape, and that difference must be preserved post-fold (`search --full --json` = today's `query --json`; `search --json` unchanged = today's `search --json`).
- **`internal/cli/query.go`** itself is deleted as a real command and replaced with a `Hidden: true` stub: `Use: "query"`, `RunE` returns a non-zero error naming `search --full`, e.g. `fmt.Errorf("codegraph query has been renamed to \"codegraph search --full\"")`. Cobra's own error-return path already causes non-zero exit (`SilenceErrors: true` at root, `root.go:52-53`, means the caller — `main.go`'s `fmt.Fprintln(os.Stderr, err); os.Exit(1)` — prints and exits 1, satisfying "exits non-zero with 'renamed to X'" with **zero new error-printing code**).
- `root.go:56-61`'s `AddCommand(...)` list keeps `newQueryCmd()` registered (now returning the stub), so `--target`-style discovery, shell completion, and `cobra/doc`'s tree walk all still see it exists (as hidden) — this matters for the "for one release, then vanish" plan: the stub's removal in a *later* milestone is a one-line deletion from `AddCommand`, not a re-architecture.

### `unlock` → `daemon unlock`

`daemon.go` already has the `daemon`/`daemon start`/`daemon stop` sub-tree (`daemon.go:47-94,113-181,201-249`) built with `cmd.AddCommand(newDaemonStartCmd(), newDaemonStopCmd())` (`daemon.go:91`). Adding `unlock` as a third child is mechanical: `cmd.AddCommand(newDaemonStartCmd(), newDaemonStopCmd(), newDaemonUnlockCmd())`, where `newDaemonUnlockCmd()` is **`unlock.go`'s existing `newUnlockCmd()` body moved verbatim** (`unlock.go:21-43`, identical shape to how `daemon start` was itself "moved verbatim" from the old bare `daemon` RunE per `daemon.go:113-118`'s own doc comment — this milestone repeats a pattern the codebase already executed once). `internal/cli/unlock.go`'s top-level `newUnlockCmd()` becomes the same `Hidden`+non-zero-exit stub shape as `query`, naming `daemon unlock`.

### Registering the "renamed" stub consistently across every guard

Both `query` and `unlock` stubs need the identical five-way consistency the milestone calls out:

1. **`cobra/doc`/CLI-REFERENCE**: automatic — `Hidden: true` excludes them from `documented()` (`clidoc/main.go:61-63`), so they never appear in `docs/CLI-REFERENCE.md`.
2. **Flag-accounting guard** (`cli_reference_test.go`): the stub's `help` flag (added by every command via `InitDefaultHelpFlag`) needs a command-level allowlist line — `query\t<reason: renamed to "search --full">` and `unlock\t<reason: renamed to "daemon unlock">` in `testdata/cli-reference-allowlist.txt`. If the stub takes zero flags of its own (recommended — no `--path`/`-p` on a stub that only errors), this is the *only* line needed per stub.
3. **Goldens**: no CLI golden fixture keys off `query`/`unlock` by literal command name was found in this pass (see Part 1's caveat) — verify at plan time, but the wire oracle (MCP) is untouched since MCP tool names are frozen and neither `query` nor `unlock` ever had an MCP tool (the 8 MCP tools map to `explore/node/search/callers/callees/impact/files/status` — `tools.go:174,310-346` — `query` and `unlock` were CLI-only, so the fold has **zero MCP wire surface to preserve or break**).
4. **`root.go`'s command count** (`newRootCmd`, `root.go:44-63`): unchanged count of top-level `AddCommand` entries (stub replaces real command 1:1); `daemon`'s subcommand count grows by one.
5. **SKILL.md / `internal/mcp/resources`**: per the read-through of `search.md` (`internal/mcp/resources/search.md`) and `SKILL.md` (`.claude/skills/codegraph/SKILL.md:10-16,46-55`), **neither `query` nor `unlock` is named by CLI-verb-form anywhere in the skill or resource docs today** — only `explore` and `impact` get an explicit `codegraph <verb> "<query>"` CLI-fallback mention (`SKILL.md:36,40`), and those verbs are untouched by the fold. **No skill/resource text needs to change for the fold itself.** (It *would* be worth a documentation pass adding a `codegraph search --full <term>` fallback mention alongside `codegraph_search`'s row for consistency, but that is a docs-polish addition, not a fold-driven correction.)

### Verb fold vs. glow-up ordering inside Part 1/2's shared surface

Because `search.go`/`daemon.go` are exactly the files Part 1's `present.RenderSearch`/`present.RenderQuery` would also touch, doing the fold **first** means the styled renderer is written once against the final `search --full` flag shape, never against a `query` struct that gets deleted out from under it a phase later.

## Part 3 — Agent reach

### Growing `AgentTarget` without 8 re-implementations

The interface (`types.go:127-160`) is already minimal and correctly factored: `ID/DisplayName/SupportsLocation/Detect/Install/Uninstall/DescribePaths`. Today, **Claude alone** gets skill+hook (`claude.go:405-503` — `claudeSkillFilePath`, `claudeHooksScriptPath`, `claudeSessionStartBlocks`), while Codex/opencode/Gemini get only the shared marker-fenced instructions block (`instructions.go`, `codegraphInstructionsBlock`) and the other 4 targets (Cursor, Hermes, Antigravity, Kiro) get neither. `install.go`'s loop (`printAgentResults`, `install.go:139-165`) never branches on capability — it calls `t.Install(loc, opts)` uniformly and prints whatever `WriteResult` comes back. **This is already the right shape for "grows capabilities without re-implementing install"**: the interface itself does not need new methods to add skill+hook to more targets, because `Install`/`Uninstall` are already per-target black boxes. What *would* need a new interface method is a **capability query** the milestone's reach matrix needs to report ("which of the 8 targets has a skill dir / instructions path / hook mechanism") without every caller re-deriving it from `DescribePaths()`'s flat path list.

Recommended addition — additive, does not break the 8 existing implementations if given a default:

```go
// Capability describes what mechanisms a target actually supports —
// distinct from DescribePaths (which paths exist) because a target with
// no hook mechanism should never be asked to write one, and
// "supports X" must be knowable without attempting a write.
type Capability struct {
    Skill        bool // has a skill-directory mechanism (SKILL.md-shaped or equivalent)
    Instructions bool // has a marker-fenced instructions file (existing 4-target set)
    Hook         bool // has a session/tool-event hook mechanism to register into
}

// AgentTarget grows:
Capabilities() Capability
```

Each of the 8 target files implements `Capabilities()` as a one-line literal (mirroring how `SupportsLocation` is already a one-line literal per target, e.g. `codex.go:28`, `kiro.go:27`) — no shared logic duplicated, no behavior change to `Install`/`Uninstall`, and `install.go`'s picker/summary code can now render a capability column without probing `DescribePaths()` heuristically. This is additive to the interface (every existing implementer must add one method — a compile-time-enforced, mechanical change across 8 files, not a design risk).

### Where the Claude Code PreToolUse nudge hook lives

The embedded hook mechanism is already **event-generic**, not `SessionStart`-specific, at the writer level: `writeHookEntry(path, event string, ownBlocks []any, ownCommands []string)` (`shared.go:202-269`) takes `event` as a plain string and does exact-command-string ownership matching (`shared.go:180-183,214-236` — the v0.10.0 security lesson: *"exact command-string match ... never by the block's matcher value or shape"*). Registering a `PreToolUse` block requires **zero changes to `writeHookEntry` itself** — only:

1. A new script, e.g. `.claude/hooks/pretooluse-nudge.sh`, embedded alongside the existing `session-nudge.sh` via a new `//go:embed` line in `claudeassets.go:33-36` and a new accessor (`PreToolUseNudgeScript()`, mirroring `SessionNudgeScript()`, `claudeassets.go:58`).
2. A new `PreToolUse` array added to the **same** `.claude/hooks/hooks.json` fragment (`hooks.json`'s top-level `hooks` object already supports multiple event keys — the JSON shape is `{"hooks": {"SessionStart": [...], "PreToolUse": [...]}}`), with a `matcher` targeting the tool names the milestone specifies (grep/find/Read) — Claude Code's own hook-matcher syntax needs a docs check at plan time (verify the current matcher grammar against `docs.anthropic.com/.../hooks` before hand-authoring the matcher string — flagged `[ASSUMED]` here since not fetched in this pass).
3. `claude.go` grows a `claudePreToolUseBlocks(loc)` function mirroring `claudeSessionStartBlocks(loc)` (`claude.go:207-260`) exactly — decode the embedded fragment's `PreToolUse` key instead of `SessionStart`, rewrite the command path the same way, and call `writeHookEntry(settingsPath, "PreToolUse", blocks, ownCommands)` alongside (not instead of) the existing `writeHookEntry(settingsPath, "SessionStart", ...)` call in `Install` (`claude.go:462-476`).
4. **Uninstall reversal** is symmetric and already generic: `removeHookEntry(settingsPath, "PreToolUse", ownCommands)` alongside the existing `removeHookEntry(settingsPath, "SessionStart", ownCommands)` call (`claude.go:576-586`) — same exact-identity matching, same idempotency guarantee, same "never touches a user's unrelated hook under the same event" property `writeHookEntry`'s doc comment already establishes.

### Runtime cost — the `.codegraph/` presence check

The milestone's own precedent is the answer: `session-nudge.sh` (`.claude/hooks/session-nudge.sh:12-15`) does exactly one `[ -d "${CLAUDE_PROJECT_DIR:-.}/.codegraph" ]` check, no binary invocation, no index read — a `PreToolUse` script must be at least as cheap since it runs on **every** matched tool call, not once per session. The nudge script's job is: check `.codegraph/` exists, and if the matched tool is grep/find/Read, emit additive context (never block — the milestone is explicit: "add context... never denies", i.e. `PreToolUse`'s hook output must use the non-blocking/advisory response shape, not an exit code or JSON field that denies the tool call — verify the exact non-blocking response contract against current Claude Code hooks docs at plan time, since a `PreToolUse` hook *can* block by design and this feature must deliberately opt out of that capability).

## Part 4 — Codex parity

### Current state (`codex.go`)

- Global-only (`SupportsLocation` returns `loc == LocationGlobal`, `codex.go:28`) — `Install`/`Uninstall`/`DescribePaths` all short-circuit to a no-op at `LocationLocal` (`codex.go:87-89,122-124,153-155`).
- MCP entry: hand-rolled TOML splice of exactly one table, `[mcp_servers.codegraph]`, via `spliceTOMLTable`/`stripTOMLTable` (`toml.go:19-69`) against `~/.codex/config.toml` (`codexConfigPath`, `codex.go:30-36`).
- Instructions: a shared marker-fenced block at `~/.codex/AGENTS.md` (`codexInstructionsPath`, `codex.go:38-44`) — **global**, not repo-root.
- No skill directory, no hook/nudge mechanism today.

### What must grow

1. **Re-verify the "Codex has no per-project config" premise.** Current web evidence (MEDIUM confidence, not yet cross-checked against `openai/codex`'s own repo docs) says Codex resolves config through a precedence chain **"`.codex/config.toml` (closest to cwd wins) → `~/.codex/config.toml` → `/etc/codex/config.toml` → defaults"**, and that `.codex/config.toml` in a repo only loads when the project is marked **trusted** — an untrusted project silently falls back to user/system config with no error. If accurate, `codexTarget.SupportsLocation(LocationLocal)` should flip to `true`, `codexConfigPath()` should take a `loc Location` parameter (mirroring `claudeConfigPath(loc)`, `claude.go:82-91`) and resolve to `./.codex/config.toml` for local, and `codexTableBody`/`spliceTOMLTable` need no changes — the splice function already operates on arbitrary file content, so it is location-agnostic by construction (`toml.go:19-41` takes `content string`, never a path). **This is the single highest-risk unverified claim in this research — a live Codex CLI session (`codex --version`, then a real repo with `.codex/config.toml` written and a fresh session probing whether it took effect) must confirm project-trust gating before this ships**, because a silent-fallback config write is exactly the "looks configured, does nothing" failure class this codebase's own guard-discipline (rule `84d1gfpywd`, PROJECT.md Key Decisions) exists to catch.
2. **Skill directory.** Codex's skill mechanism (per the same web pass) is `.agents/skills/<name>/` at project scope — **not** `.claude/skills/`. This needs a new `codexSkillDirPath(loc)` and a new install/uninstall step in `codex.go` mirroring `claudeSkillFilePath`/`claudeSkillDirPath` (`claude.go:129-164`) but writing into `.agents/skills/codegraph/SKILL.md` (global equivalent under `~/.agents/skills/` or Codex's documented global skill root — verify exact path at plan time). The **content** should be the same embedded `SKILL.md` (`claudeassets.SkillMarkdown()`) — no reason to author a second skill file, since the skill's content is harness-agnostic Markdown, not Claude-specific syntax (the file's own frontmatter is just `name:`/`description:`, `.claude/skills/codegraph/SKILL.md:1-4`).
3. **Repo-root `AGENTS.md` block.** Distinct from the existing **global** `~/.codex/AGENTS.md` block — a **project-local** `./AGENTS.md` at repo root, using the exact same `codegraphSectionStart`/`codegraphSectionEnd` marker fence and `upsertInstructionsEntry`/`removeMarkedSection` helpers already shared by every target (`instructions.go:9-11`, `shared.go` — `claude.go:389-394` and `codex.go:110-115` both already call these verbatim). This is a **second, location-scoped instructions file** for Codex, not a replacement of the global one — both must coexist, the same way Claude already writes both a global `~/.claude/CLAUDE.md` block and a local `./.claude/CLAUDE.md` block depending on `loc` (`claudeInstructionsPath(loc)`, `claude.go:101-110`).
4. **Codex's own hook/nudge mechanism, if one exists.** Not confirmed in this research pass — Codex CLI's customization stack (per the web search) lists AGENTS.md, skills, MCP, and subagents as its four composition layers; no lifecycle-hook-equivalent to Claude's `SessionStart`/`PreToolUse` was surfaced. **Flagged `[ASSUMED: none exists]`** — if a plan-time doc check confirms no hook mechanism, Codex parity for the "nudge" theme is satisfied by the `AGENTS.md` block and skill file alone (both of which Codex already reads at session start per the web search's "Codex walks up from cwd toward project root and includes AGENTS.md guidance in the first turn" finding) — no PreToolUse-equivalent to build.
5. **TOML splice coping with a second table/file.** `spliceTOMLTable`/`stripTOMLTable` (`toml.go`) operate on a single named table within one file's content string; a **second file** (`.codex/config.toml` vs `~/.codex/config.toml`) needs no new code in `toml.go` at all — it is just a second `content, err := os.ReadFile(newLocalConfigPath)` / `spliceTOMLTable(...)` / `atomicWriteFile(...)` call sequence, parameterized by `loc` exactly like `claudeConfigPath(loc)` already parameterizes Claude's two files. **A second table within one file** is also already handled: `findTOMLTableRange` (`toml.go:77-103`) scopes its replace/append purely to `[mcp_servers.codegraph]`'s own header-to-next-header byte range, so an unrelated `[some_other_table]` in the same `config.toml` is preserved untouched by construction (the same "preserving every other byte of content verbatim" guarantee documented at `toml.go:23`).

### `SupportsLocation(LocationLocal)` flipping to `true` — downstream effects

- **`ResolveTargetFlag("auto", loc)`** (`registry.go:79-93`): iterates `AllTargetIDs()` and calls `Detect(loc).Installed` per target — Codex's `Detect` (`codex.go:63-83`) already branches on `loc != LocationGlobal` and returns an empty `DetectionResult{}` for local; flipping `SupportsLocation` **without** also updating `Detect` to actually check `.codex/` presence at the *local* path would silently make `--target auto --location local` treat Codex as "not installed" forever even though it now supports local scope — this is a **required companion change**, not optional: `Detect(LocationLocal)` must check for `.codex/config.toml` or `.codex/` existing in the repo, mirroring `claudeTarget.Detect`'s own local-vs-global path branch (`claude.go:348-364`).
- **`agentpicker` (`internal/cli/tui/agentpicker.go`)**: the picker renders `DisplayName()` + a `SupportsLocation(loc)`-gated selectability per target for whatever `loc` the interactive `install` run is targeting (`install.go` → `runAgentPicker(cmd, loc)`). Today Codex is presumably shown greyed-out/unselectable (or absent) under `--location local`; once `SupportsLocation(LocationLocal)` is `true`, Codex becomes selectable in the local picker for the first time — this is a **UI-visible behavior change** the picker's own test suite (`agentpicker_test.go`) will need a new fixture/assertion for, and is exactly the kind of change WINDOWS #32 ("picker footer overflow ... with all 8 targets") is already tracking — worth sequencing the Codex-local flip **after** #32's footer fix lands, not before, so the newly-selectable 8th/9th row doesn't make the pre-existing overflow bug worse mid-milestone.
- **`DescribePaths(loc)`** (`codex.go:152-164`) must add the new local paths (`.codex/config.toml`, `.agents/skills/codegraph/SKILL.md`, `./AGENTS.md`) to its local-scope return, or `--print-config-style` reporting and any test asserting `DescribePaths` completeness will silently under-report.

## Part 5 — Suggested build order

The four themes have a real dependency graph, not just a priority order:

```
                    ┌─────────────────────────┐
                    │ Bug/window burn-down     │  independent, parallelizable
                    │ (WINDOWS #12,13,26,28,   │  any time, any engineer —
                    │  29,30,31,32,36; GH      │  touches unrelated files
                    │  #13-20)                 │  (web/, ci.yml, docs/RELEASE.md)
                    └─────────────────────────┘

1. Verb fold FIRST                     2. Reach capability model FIRST
   (search --full, daemon unlock,         (AgentTarget.Capabilities(),
    hidden stubs, allowlist entries)      per-harness doc verification)
        │                                        │
        ▼                                        ▼
3. CLI glow-up SECOND                  4. Codex parity SECOND
   (present.Render* written once          (first real consumer of the
    against the FINAL verb surface;       capability model; project-
    --color flag; fang evaluation)        local scope verified live)
        │                                        │
        └──────────────┬─────────────────────────┘
                        ▼
              5. Cross-cutting close-out
                 (SKILL.md/resources docs pass naming search --full
                  where explore/impact are already named; re-freeze
                  CLI-REFERENCE + allowlist; live-session verification
                  per harness, the v0.10.0 evidence standard)
```

**Why verb fold before glow-up:** the milestone context states this directly and the codebase confirms it — writing `present.RenderQuery`/`present.RenderSearch` against `search.go`'s pre-fold two-flag shape means deleting and rewriting that renderer one phase later when `--full` lands. `search.go` and `daemon.go` are exactly the files both themes touch (Part 2's fold and Part 1's per-verb renderer), so sequencing avoids a guaranteed rework.

**Why the reach capability model before Codex parity:** Codex is explicitly named as "the first consumer" reaching for the general mechanism (skill dir capability, hook capability) the other 7 targets either already have (Claude) or will get in the same phase (AGENT-04…07 un-deferred). Building `Capability{}` and wiring it into `install.go`'s reporting *before* writing Codex's specific skill-dir/AGENTS.md/local-scope code means Codex's implementation is the first real exercise of the new interface method, not a special case bolted on afterward — the same "seam before second consumer" discipline this codebase already applies elsewhere (v0.12.0 Phase 1's Key Decision).

**Why glow-up and Codex parity are each "second" within their track, not "last":** glow-up has no dependency on agent-reach work (disjoint packages: `internal/cli/present` + `internal/cli/*.go` RunE bodies vs. `internal/agents/*.go`), so the two tracks can run as parallel workstreams once their own track's first step lands — only the *internal* ordering within each track (fold-before-style, capability-before-Codex) is a hard dependency.

**Why bug/window burn-down is fully independent:** every item in the four burn-down buckets (favicon/CSP, picker footer, `/graph` console errors, `priorCoverageGeneration`, `web:drift`, the vendored `button.svelte`, the daemon watchdog flake, docs/CI wiring, GH #15/#16/#20) lives in files disjoint from `internal/cli/present`, `internal/agents`, and the verb surface — `web/`, `.github/workflows/`, `docs/RELEASE.md`, `SECURITY.md`, `internal/uiserver` (cytoscape-elk, coverage). The one soft coupling: WINDOWS #32 (picker footer overflow) should land **before** the Codex-local-scope flip (Part 4) makes the picker's selectable-target count grow, per the note in Part 4 above — everything else in the burn-down can run on any timeline relative to the other three themes.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Widening `present.ChoosePresentation`'s signature for `--color`

**What people do:** add a third parameter (or an enum) to `ChoosePresentation` to thread `--color` through.
**Why it's wrong:** breaks the D-03/D-04 contract that made `present` trivially unit-testable and env-blind (`present/tty_test.go`); fans out to every one of the 20+ new call sites for a concern (`--color` flag parsing) that belongs entirely in `internal/cli`, not `internal/cli/present`.
**Do this instead:** resolve `--color` + TTY + `NO_COLOR` into the existing two-arg shape at a single shared `internal/cli` helper, called from every RunE — `present` never changes.

### Anti-Pattern 2: Reading `internal/query`'s markdown strings back into styled output

**What people do:** call `eng.Explore(...)`/`eng.Node(...)` (the markdown-string methods) in the styled branch and try to lipgloss-colorize the resulting markdown text with regex/string manipulation.
**Why it's wrong:** re-derives formatting from a rendered string instead of the plain struct underneath it — exactly the anti-pattern `present`'s own doc comments repeatedly warn against ("present decorates, never recomputes", `present/status.go:27-28,55-56`), and it is fragile against any future change to `RenderExplore`'s markdown shape.
**Do this instead:** call `eng.ExploreDetail`/`eng.NodeDetail` directly in the styled branch — the same plain structs the UI already consumes.

## Sources

- `/Volumes/Code/github.com/seanb4t/codegraph-go` — read directly: `internal/cli/{present,tui}/*.go`, `internal/cli/{query,search,root,unlock,daemon,status,install,man,explore,node,callers,callees,impact,affected}.go`, `internal/query/{engine,search,detail,explore,node}.go`, `internal/agents/{types,registry,claude,codex,toml,instructions,shared,claudeassets}.go`, `internal/mcp/{server,resources,tools}.go`, `tools/clidoc/main.go`, `internal/cli/cli_reference_test.go`, `.claude/skills/codegraph/SKILL.md`, `.claude/hooks/{hooks.json,session-nudge.sh}`, `claudeassets.go`, `.planning/PROJECT.md` (Key Decisions, Current Milestone) — HIGH confidence, primary source.
- `pkg.go.dev/github.com/charmbracelet/fang`, `pkg.go.dev/charm.land/fang/v2`, `github.com/charmbracelet/fang` (web, MEDIUM confidence) — fang's styled help/error/version feature set confirmed; runtime interaction with a long-lived `RunE` (e.g. `serve --mcp`) not independently verified against this binary.
- Codex CLI config precedence, `.codex/config.toml` project-trust gating, `.agents/skills/` layout, `AGENTS.md` discovery (web, MEDIUM confidence, not cross-checked against `openai/codex`'s own repository docs in this pass) — `developers.openai.com/codex/config-basic`, `codex.danielvaughan.com` (Codex Knowledge Base, third-party but detailed and internally consistent across 3 separate articles), `inventivehq.com/knowledge-base/openai/where-configuration-files-are-stored`. **Flagged for live-session re-verification per the milestone's own "verified against current docs rather than assumed" instruction** — this is the single highest-risk unverified claim in this document.
- Claude Code `PreToolUse` hook matcher grammar and non-blocking/advisory response contract — **not fetched in this pass**; flagged `[ASSUMED]`, required reading before Part 3's hook script is authored (`docs.anthropic.com` hooks reference).

---
*Architecture research for: CodeGraph Go v0.14.0 Polish & Agent Reach*
*Researched: 2026-09-14*
