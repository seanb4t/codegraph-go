# Pitfalls Research

**Domain:** CLI colour/TTY glow-up + hard verb rename + multi-harness agent-reach (skills/instructions/nudge hooks) + Codex parity + defect burn-down, added to an existing shipped Go CLI/MCP/UI product (codegraph-go v0.13.0 → v0.14.0)
**Researched:** 2026-09-14
**Confidence:** HIGH for repo-internal findings (read from source); MEDIUM for lipgloss v2 / colorprofile / fang API claims (official docs + pkg.go.dev, not exercised in this repo yet); MEDIUM for Claude Code hooks JSON schema (official docs, cross-checked against a 2026 gist and two blog summaries); LOW-MEDIUM for Codex CLI's current skills/AGENTS.md/config surface (community docs, changed during 2026, no official reference doc read end-to-end) — flagged `[ASSUMED]` per WINDOWS.md #35's own precedent for this repo.

## Critical Pitfalls

### Pitfall 1: lipgloss v2's renderer removal is a silent trap for a "TTY-gated" mental model

**What goes wrong:**
`internal/cli/present/styles.go`'s own doc comment already states the truth: "lipgloss v2 removed the renderer-construction API present in v1 — `Style.Render()` always emits full-fidelity ANSI." Today that is safe only because `present` never renders unless `ChoosePresentation(isTTY, noColor)` is already true (binary on/off gating at three call sites — `status.go:86`, `files.go:72`, `progress_cli.go:31`). The v0.14.0 glow-up plan is to move from "3 monochrome styles, 3 of 24 verbs" to "a real palette, colour not just bold/faint/underline, across every human-output verb." The trap: adding real 24-bit/256-colour styles to `headerStyle`/`labelStyle`/`sectionStyle`-equivalents while keeping the *same* binary TTY gate produces full-fidelity truecolor ANSI on any terminal that passes `term.IsTerminal()` — including a 16-colour xterm, a `TERM=dumb`-but-somehow-still-a-tty session, or a terminal that supports ANSI but not truecolor. lipgloss v2 does not downsample for you; that is `charmbracelet/colorprofile`'s job (`colorprofile.Detect`, `colorprofile.Writer`), and it is a *separate* package this repo does not import today (confirmed: only `charm.land/lipgloss/v2` appears in `internal/cli/present`). Skipping it means "TTY-gated with NO_COLOR honoured" ships colour-capability-blind.

**Why it happens:**
The repo's existing mental model — "isTTY && NO_COLOR=="" ⇒ pretty branch" — was sufficient when the palette was 3 monochrome attributes (bold/faint/underline render identically on every ANSI terminal). It stops being sufficient the moment real colour enters, because colour support is a *spectrum* (NoTTY/Ascii/ANSI/ANSI256/TrueColor), not a binary, and lipgloss v2 deliberately moved that responsibility out of the style layer.

**How to avoid:**
Route every `present` render through `colorprofile.Writer` (or `lipgloss.Println`/`Fprintln`/`Sprint`, which wrap it) at the same RunE call sites that already own `os.Getenv`/`term.IsTerminal` per D-03 — never inside `present` itself, preserving `present`'s existing "must NOT read os.Getenv or call term.IsTerminal" contract. `colorprofile.Detect` already implements the NO_COLOR/CLICOLOR_FORCE/TERM=dumb precedence correctly (see Pitfall 2) — do not hand-roll a second decision alongside `ChoosePresentation`; either replace `ChoosePresentation`'s pretty/plain binary with the detected profile becoming the single source of truth for both "should I style at all" and "at what fidelity," or make `ChoosePresentation` explicitly answer only "is a lipgloss render attempted" while the profile answers "at what fidelity" — but not both answered inconsistently by two independent env reads.

**Warning signs:**
A style renders visually broken (garbled escape codes, wrong colours, "quoted-looking" boxes) on a 16-colour terminal or over SSH with a constrained `TERM`; a manual test in `TERM=xterm` (no `-256color` suffix) or a CI log viewer that only understands 16 ANSI colours; styled output going through a pipe that *is* a tty (e.g. `less -R`) but degrades to a narrower profile than the outer terminal.

**Phase to address:**
CLI glow-up phase, as the first task before any palette expansion — establish the `colorprofile` seam once, then build every subsequent style on top of it.

---

### Pitfall 2: NO_COLOR/CLICOLOR_FORCE precedence — codegraph-go currently supports NEITHER of CLICOLOR/CLICOLOR_FORCE

**What goes wrong:**
A repo-wide search (`rg -n "CLICOLOR|NO_COLOR"`) finds `NO_COLOR` read at exactly 4 call sites (`progress_cli.go`, `status.go`, `files.go`, `tui/tty.go`) and zero occurrences of `CLICOLOR`/`CLICOLOR_FORCE` anywhere in the tree. If the glow-up phase adds `CLICOLOR_FORCE` support (reasonable, since it lets a user force colour into a non-tty pipe for a human-piped-through-`less -R` workflow) without checking precedence against the existing `NO_COLOR` seam, the two most common wrong orderings are: (a) `CLICOLOR_FORCE` checked before/instead of `NO_COLOR`, so `NO_COLOR=1 CLICOLOR_FORCE=1` incorrectly shows colour — violating no-color.org's own stated contract that other libraries (Node.js `FORCE_COLOR`, but also Rust/Go ports) get backwards in practice; or (b) `CLICOLOR_FORCE` is wired in a second independent `os.Getenv` call that disagrees with `colorprofile.Detect`'s own opinion (see Pitfall 1) if that library is adopted — two decision-makers for one question.

**Why it happens:**
The existing `ChoosePresentation(isTTY, noColor)` signature only has room for the property it was built for (v1.0-era binary styled/plain). Adding a new environment knob under time pressure is naturally done by adding a third bool parameter and an `||`/`&&` at the call site rather than re-deriving the whole decision from one authoritative source.

**How to avoid:**
Verified via web search (2026-09-14): `charmbracelet/colorprofile`'s own `Detect` implements the correct order — NO_COLOR beats CLICOLOR/CLICOLOR_FORCE, and CLICOLOR_FORCE beats CLICOLOR, matching the no-color.org / CMake-documented spec (community implementations diverge, e.g. Node.js does it backwards — cite this explicitly in code comments so a future maintainer does not "fix" it to match Node's convention). If `colorprofile.Detect` is adopted (Pitfall 1's remedy), CLICOLOR_FORCE precedence is free and correct; if it is not adopted and a hand-rolled check is kept instead, the precedence order (`NO_COLOR` set-to-anything ⇒ no colour, full stop, regardless of CLICOLOR_FORCE) needs its own explicit unit test with all four combinations (`NO_COLOR` set/unset × `CLICOLOR_FORCE` set/unset) — not just the two that exist today.

**Warning signs:**
A CI log (non-tty) that sets `CLICOLOR_FORCE=1` but also inherits `NO_COLOR=1` from a shared env and gets colour anyway; grep for `CLICOLOR` finding zero call sites in the shipped diff even though the milestone's spec talks about "NO_COLOR honoured" without naming CLICOLOR_FORCE at all — if CLICOLOR_FORCE is genuinely out of scope for this milestone, say so explicitly rather than leaving it silently absent.

**Phase to address:**
CLI glow-up phase, same task as Pitfall 1 (one settled env-precedence policy, one test matrix).

---

### Pitfall 3: agent/MCP-path ANSI leakage is a fail-closed archtest today — but its denylist is a fixed list of three literal import paths that a new dependency (fang, x/ansi, colorprofile) will not automatically join

**What goes wrong:**
`internal/cli/present/archtest/import_graph_test.go`'s `TestNoCharmInServeReachablePackages` is the ANSI-isolation guard this milestone's context explicitly calls "fail-closed." Reading it: `forbiddenImportPaths` is a **hardcoded 3-entry slice** — `charm.land/lipgloss/v2`, `charm.land/bubbletea/v2`, `charm.land/bubbles/v2` — checked against the transitive import closure of exactly six `guardedPackages` (mcp, graphstore, daemon, watch, indexer, query). If the glow-up phase adds `charmbracelet/fang` (which itself imports lipgloss v2 transitively, confirmed via pkg.go.dev search) or `charmbracelet/x/ansi` / `charmbracelet/colorprofile` for downsampling, and any of THOSE new packages leak into one of the six guarded packages' import graph, the archtest stays silent — not because the property holds, but because the literal string being checked for is not the literal string that would actually indicate the violation. This is exactly the failure shape the milestone's own context calls out ("a guard that cannot fire," rule `84d1gfpywd`): the test still runs, still reports PASS, and is not testing what a reader of its name believes it tests.

**Why it happens:**
`forbiddenImportPaths` was written when lipgloss v1.0's `internal/cli/present` had exactly one dependency. A dependency added later (fang wraps cobra + lipgloss; colorprofile is lipgloss's own downsampling sibling) is a natural, innocent addition that nobody thinks to cross-reference against a denylist buried in an `archtest` subpackage three call-sites away from where `go.mod get` was run.

**How to avoid:**
Before adding any new Charm-family or ANSI-adjacent dependency this milestone (fang, colorprofile, x/ansi, or any bubbles/bubbletea companion), add its exact vanity import path to `forbiddenImportPaths` in the SAME commit that adds it to `go.mod`, and re-run `TestNoCharmInServeReachablePackages` to confirm it can still fail (temporarily import the new package from one of the six guarded packages, confirm RED, revert — the repo's own mutation-log discipline). Better: invert the check from an enumerated denylist to a prefix match on `charm.land/` (the vanity domain every Charm v2 package shares per the repo's own Finding-1 comment about `/v2`-suffix aliasing) plus a small explicit allowlist for anything under `charm.land/` legitimately reachable from a guarded package (none exist today) — a prefix match cannot silently miss a new sibling package the way an enumerated list can. If a prefix match is judged too broad, at minimum add a comment at the top of `forbiddenImportPaths` cross-referencing every Charm-family dependency added anywhere in `go.mod`, so a `go.mod` diff review has something concrete to check the list against.

**Warning signs:**
`go.mod` grows a `charmbracelet/fang` or `charmbracelet/colorprofile` require and `forbiddenImportPaths` is untouched in the same diff; `TestNoCharmInServeReachablePackages` passes on a PR that also touches `internal/mcp` or `internal/daemon` imports.

**Phase to address:**
CLI glow-up phase — specifically, the task that evaluates `fang` per the milestone's own stated plan ("`charmbracelet/fang` evaluated for help/error/version styling before any help template is hand-rolled") must update this archtest as part of adopting or rejecting fang, not as an afterthought.

---

### Pitfall 4: `query` → `search --full` rename touches far more files than `internal/cli/query.go` — and `search.go`'s own JSON marshal path is NOT byte-identical to `query.go`'s today

**What goes wrong:**
`query.go` and `search.go` both call `eng.matchNodes` transitively (`Engine.Query` and `Engine.Search` in `internal/query/search.go`), but `query.go`'s `--json` path calls `query.MarshalQueryJSON(nodes)` (a dedicated helper producing the "golden query.json envelope shape") while `search.go`'s `--json` path calls raw `json.Marshal(locs)` on `[]query.Location` — a genuinely different, narrower shape (no source body, no signature, matching MCP's `search` companion handler, D-08b). The milestone's plan describes the rename as "the flag selects record-vs-location shape and the human `--full` branch finally shows what it discards today" — meaning `search --full` must dispatch to the `query.go` code path (full node records via `MarshalQueryJSON`), while bare `search` keeps today's `search.go` path (`Location` records). A naive rename that just renames the `query` command's `Use:` string to `search --full` without actually merging the two RunE bodies leaves TWO commands both claiming the `search` verb (a cobra registration collision) or silently drops one code path's behavior. Flag inconsistency compounds this: `query` has `-j/-l/-k` shorthands registered (`cmd.Flags().StringVarP`, `IntVarP`, `BoolVarP`), `search` does not (`StringVar`/`IntVar`/`BoolVar`, no `P` suffix, no shorthand) — confirmed by reading both files. If the merge keeps `search`'s flag registration verbatim and just adds a `--full` bool, users who type `-j`/`-l`/`-k` (habituated from `query`) get "unknown shorthand flag" errors post-rename, which is exactly the kind of silent breakage this milestone is trying to prevent for the *hard* verbs (`unlock`) while introducing it accidentally for the *merged* one.

**Why it happens:**
The two commands look nearly identical (same `resolveStartPath`, same `query.OpenAt`, same worktree-notice placement) and evolved with genuinely different flag/JSON conventions at different times (search.go's own comment: "no dedicated MarshalSearchJSON helper exists"), making a "just rename" instinct plausible when the real work is "merge two RunE bodies and reconcile two independently-evolved flag registrations."

**How to avoid:**
Treat this as a merge, not a rename: keep one `matchNodes`-backed command, gate the full-vs-location JSON/`-l`ine shape behind `--full`, and reconcile flags by keeping the superset (`-j/-l/-k` shorthands) rather than the narrower set — a shorthand flag disappearing is a worse regression than a shorthand flag appearing on a verb that didn't have it before. Write the merged command's flag test FIRST (assert `-j`, `-l`, `-k` all parse on both `search` and `search --full`) before touching `root.go` registration, so the merge is driven by a red test rather than eyeballing two files.

**Warning signs:**
`cobra.Command.Flags()` for the shipped `search` command missing `-j`/`-l`/`-k` shorthands that `query` had; `--json` output shape under `search --full` matching `search`'s narrow `Location` shape instead of the full node-record envelope; two `newXCmd()` functions both registering `Use: "search ..."` at root-command build time (a cobra panic or silent last-registration-wins).

**Phase to address:**
Verb-fold phase (the CLI glow-up milestone's hard-rename work), with its own dedicated plan separate from the colour work — this is a behavioral merge, not a styling change, and mixing the two in one plan risks the colour work masking a functional regression in review.

---

### Pitfall 5: `unlock` → `daemon unlock` and stub-exit-nonzero for removed verbs — every consumer of the OLD verb name must be found, not just `internal/cli/`

**What goes wrong:**
`internal/cli/unlock.go` registers a *top-level* `codegraph unlock [path]` command today. Moving it under `daemon` (a cobra parent that already exists — `daemon.go` has `start`/`stop` subcommands) is mechanically simple in Go, but the milestone's own required-reading list flags exactly this class of miss: "which files reference verbs by name — SKILL.md, MCP instructions, resources/*.md, README, CLI-REFERENCE.md, hooks scripts, Taskfile, CI." `docs/CLI-REFERENCE.md` is **generated** (via `tools/clidoc`) and drift-gated (`docs:cli:drift` in Taskfile.yml), so it will regenerate correctly IF `task docs:cli` is re-run — but the generator only fixes the reference doc, not prose that mentions `codegraph unlock` or `codegraph query` in free text (README code fences, the SKILL.md decision procedure, MCP `resources/*.md` reference content, `docs/RELEASE.md`, any Taskfile comment, any shell script in `.github/workflows/` that shells out to `codegraph query` for a smoke test). This repo has already burned real WINDOWS.md entries on exactly this shape of miss (ids 4–11, 14–19: the CODE-01 census repeatedly found comparison-framing prose living in files a plan's declared `files_modified` scope excluded) — the same "grep, don't trust scope-declared file lists" lesson applies here for verb-name prose.

**Why it happens:**
A rename inside `internal/cli/*.go` is easy to scope tightly (cobra registration, the RunE body, its `_test.go`), but a verb name is also **prose** scattered across docs, embedded assets (`claudeassets.SkillMarkdown()`, `claudeassets.HooksFragment()`), and CI scripts — none of which a Go compiler or `go vet` will catch, and none of which live in `internal/cli/`.

**How to avoid:**
Run a literal, whole-repo, multiline-aware census for `codegraph query`, `codegraph unlock`, ` query <`, ` unlock ` (word-boundary, not substring — `query` also appears as a Go identifier `query.Engine`/`internal/query` package name throughout, which must NOT be touched) BEFORE editing `internal/cli/`, covering `docs/`, `README.md`, `.claude/` (this repo's own dogfooded skill install), embedded skill/hook assets under the module root, `.github/workflows/*.yml`, `Taskfile.yml`, and any `testdata/golden/*` fixture that encodes command lines. Treat MCP-side naming as a SEPARATE, already-decided constraint: the milestone context is explicit that "MCP tool names are frozen — the wire oracle's transcripts pin them" — the CLI verb rename must not be allowed to bleed into `internal/mcp/tools.go`'s tool names (`codegraph_explore`, etc., which are unrelated strings already isolated from CLI `Use:` strings). Ship the stub commands (`query`, `unlock` at root) that exit non-zero and print "renamed to `search --full`"/"renamed to `daemon unlock`" for exactly one release — verify the stub is registered in `root.go` and appears (or is deliberately allowlisted as hidden, per the CLI-REFERENCE.md allowlist mechanism already in the repo) so `docs:cli:drift` and the flag-accounting guard (`cli_reference_test.go`) do not choke on a command with a message-only RunE and zero flags.

**Warning signs:**
`grep -rn "codegraph query\|codegraph unlock"` (word-boundary aware) returning hits outside `internal/cli/` and `internal/cli/*_test.go` after the rename lands; the flag-accounting test's positive floors (`cliReferenceMinCommands = 26`, `cliReferenceMinFlags = 50`) silently absorbing a net change in command count without anyone checking whether it moved in the expected direction; a stubbed verb missing from shell completion output (`codegraph completion bash` — cobra regenerates this from the live tree automatically, so a stub registered correctly is covered for free, but a stub that's NOT registered as a real `*cobra.Command` — e.g. just a `root.go` special-case in `Execute()` — silently skips completions and `man`).

**Phase to address:**
Verb-fold phase — the census must run before ANY file is edited, not as a post-hoc cleanup pass (this repo's CODE-01 census precedent shows a post-hoc pass catches most but not all instances, and each miss becomes its own WINDOWS.md entry).

---

### Pitfall 6: conventional-commit `feat!:` under `bump-minor-pre-major` — the rename IS a breaking change but the release-please config will NOT bump a major

**What goes wrong:**
The milestone context states "release-please-config.json sets `bump-minor-pre-major`, so the `feat!:` verb rename still cuts a minor, not a major" — this is stated as fact, not a risk, but it is exactly the kind of thing a contributor unfamiliar with this repo's `release-please-config.json` would get wrong: writing `feat:` (no bang) for the rename commit because "it doesn't feel like a new feature," which would (a) still bump minor under `bump-minor-pre-major` — masking the real issue, which is that release-please's generated CHANGELOG entry needs the `!` to correctly categorize the change as BREAKING CHANGE in the notes even though the version bump itself is unaffected pre-1.0. Getting the commit type wrong doesn't break the build, but it silently produces a changelog that undersells a hard, user-visible breaking rename as an ordinary feature — which matters because this is precisely the kind of change (verb removed, old script breaks) users need loudly flagged.

**Why it happens:**
`bump-minor-pre-major` decouples "is this breaking" from "what version number results," which is correct release-please behavior pre-1.0 but is counter-intuitive: most engineers' mental model of `feat!:` is "this triggers a major bump," and when it visibly doesn't, the temptation is to conclude the `!` was pointless and drop it.

**How to avoid:**
Use `feat!:` (or `BREAKING CHANGE:` footer) on the verb-rename commit regardless of the version-number outcome — the marker's job here is changelog categorization, not version arithmetic. Write this into the phase's PLAN.md commit-message guidance explicitly so no executor has to re-derive it from `release-please-config.json`.

**Warning signs:**
The generated `CHANGELOG.md` entry for the rename reads as an ordinary `Features` bullet instead of a highlighted `BREAKING CHANGES` section.

**Phase to address:**
Verb-fold phase, commit-message step.

---

### Pitfall 7: the golden/wire oracle needs a deliberate re-freeze with a RED demonstration — verb renames WILL touch frozen fixtures that were never meant to encode CLI verb names

**What goes wrong:**
`testdata/golden/` fixtures and the MCP wire oracle's frozen transcripts are this repo's proof instrument (the v0.3.0/v0.11.0 history is full of "N transcripts re-frozen in one reviewed-diff pass" language). If any golden fixture encodes a literal `codegraph query ...` or `codegraph unlock ...` invocation (e.g. as a documented example inside `testdata/golden/README.md`, which v0.11.0's Key Decisions explicitly says "records what shipped" and was "deliberately NOT edited" in a prior sweep), a rename either (a) breaks the fixture (good, loud, expected) or (b) the fixture is untouched because nobody re-ran the golden-freeze step, and a stale example ships describing a verb that no longer exists as documented — the doc equivalent of `docs/RELEASE.md`'s already-open WINDOWS.md #13 staleness.

**Why it happens:**
Golden/frozen fixtures are, by design, meant to stay untouched under normal development (that is their whole value — catching accidental drift) so there is a learned habit of NOT touching them. A rename is the rare case where touching them is correct and necessary, and that correctness is easy to miss precisely because the muscle memory says "don't."

**How to avoid:**
Before merging the verb-fold phase, run a literal-string census over `testdata/golden/` for the old verb names, re-freeze anything found, and — per this repo's own standing rule ("A gate is not trusted until demonstrated RED against a confirmed-applied mutation") — prove the re-frozen fixture can still fail: temporarily reintroduce the old verb name into whatever fixture-comparison logic covers it, confirm the comparison test goes RED, then revert. Do not treat "the fixture still passes" as evidence the fixture was checked — a fixture with no old-verb-name content in it will trivially pass whether or not the rename happened correctly, which is a vacuous-guard shape this repo has hit repeatedly (rule `84d1gfpywd`).

**Warning signs:**
`testdata/golden/README.md` or any `*.golden`/`*.json` fixture under `testdata/golden/` containing the string `"query"` or `"unlock"` as a command-line token (not as a JSON field name or package identifier) after the rename ships.

**Phase to address:**
Verb-fold phase, final task before merge.

---

### Pitfall 8: a Claude Code PreToolUse nudge hook that "redirects" instead of "adds context" reintroduces the exact friction GUARD-HOOK-01/02 was deferred to avoid

**What goes wrong:**
The milestone context is explicit that GUARD-HOOK-01/02 is "reframed from 'redirect' to 'add context': on grep/find/Read in an indexed repo it points at `codegraph_explore` and never denies." The Claude Code hooks JSON schema (confirmed via official docs search, 2026) supports `hookSpecificOutput.permissionDecision` with values `"allow"`, `"deny"`, `"ask"`, `"defer"`, plus a separate `additionalContext` field, AND a wholly separate exit-code channel where **exit code 2 blocks unconditionally regardless of any JSON emitted** ("even a JSON permissionDecision of 'allow' can't override it"). A nudge implementation that (a) exits non-zero for any reason (a bug, an unhandled error path, a missing binary on PATH) will silently become a hard block instead of a soft nudge, and (b) a nudge that sets `permissionDecision: "ask"` (rather than omitting `permissionDecision` and using only `additionalContext`) reintroduces an interactive prompt on every matching tool call — which is functionally a redirect/friction mechanism wearing "nudge" branding. The distinction between "prints context that Claude sees" (soft, `additionalContext` only, no `permissionDecision`, exit 0) and "makes a permission decision" (hard, blocks or prompts) is the entire point of this reframing, and the two are one JSON-field typo apart.

**Why it happens:**
The PreToolUse hook schema conflates two genuinely different mechanisms (permission gating and context injection) in one output shape, and it's natural to reach for `permissionDecision` because it's the more prominently documented field in most third-party hook-authoring guides (several of the search results above lead with `permissionDecision` before mentioning `additionalContext` at all).

**How to avoid:**
Write the hook script to emit ONLY `{"hookSpecificOutput": {"hookEventName": "PreToolUse", "additionalContext": "..."}}` with no `permissionDecision` key at all, and exit 0 unconditionally from the script's own logic (wrap the matching logic in error handling that falls through to a no-op exit 0 on any internal failure, never propagating a non-zero exit for "I couldn't determine whether to nudge" — fail open, not closed, since this is advisory). Add an explicit test (can be a small integration test invoking the hook script directly with representative stdin JSON) asserting the script's exit code is always 0 and its stdout JSON never contains `"permissionDecision"`.

**Warning signs:**
Any code path in the nudge script that calls `os.Exit(1)` or `os.Exit(2)` (if implemented in Go) or `exit 1`/`exit 2` (if a shell script, matching the existing `session-nudge.sh` pattern); the JSON schema test asserting the shape does not exist yet.

**Phase to address:**
Agent-reach phase, Claude Code nudge-hook task — write the "never blocks" test before the matching logic.

---

### Pitfall 9: a nudge firing on every Bash/Read/Grep call becomes noise that trains the agent (and the user) to ignore it — the matcher needs to be an indexed-repo AND a codegraph-shaped-query gate, not a blanket tool-name match

**What goes wrong:**
PreToolUse hooks match on `tool_name` (and optionally a `matcher` regex/glob on tool input) at the hook registration level, firing BEFORE every single invocation of the matched tool(s). If the codegraph nudge is registered against `Bash`, `Grep`, `Read`, and `Glob` unconditionally, it fires on unrelated Bash commands (`git status`, `npm test`), unrelated greps (searching a `.md` file for prose), and reads of files that have nothing to do with "where is X defined" — the exact class of question `codegraph_explore` answers. A nudge that fires constantly gets mentally filtered out by both the agent (repeated identical `additionalContext` becomes low-signal) and, if visible in transcripts, the human reviewing sessions.

**Why it happens:**
The cheapest implementation matches on tool name alone (`"matcher": "Bash|Grep|Read"`) because content-based matching (does this grep/read/bash command look like a code-navigation question?) requires either a heuristic (keyword match on the tool input) or accepting some false-negative rate — and under time pressure, "always fire, let content stay unfiltered" ships faster than a real heuristic.

**How to avoid:**
Gate on two things, not one: (1) an indexed repo exists (`.codegraph/` present — cheap, already a pattern this repo uses via `fileExists`-shaped checks elsewhere in `internal/agents`), checked first so the hook is a true no-op outside any codegraph-managed repo; (2) some minimal content heuristic on the tool input for `Grep`/`Read`/`Glob`/`Bash` — e.g. only nudge on a `Grep` whose pattern looks like an identifier search (word-boundary regex, no prose-shaped query), or a `Bash` command matching `rg`/`grep`/`find` invocations specifically (not arbitrary Bash). Given this is explicitly a v2 evidence-gathering exercise per the milestone's phrasing ("evidence that doesn't exist yet"), plan to measure nudge-fire frequency in the fresh-session verification pass (the "genuinely fresh live session" standard this repo already uses for skill verification) and treat a high fire-rate with low uptake as a signal to narrow the matcher further, not to ship broader.

**Warning signs:**
The hook's `matcher` is a bare tool name with no content-based sub-filter; a live-session verification transcript shows the nudge firing on more than a small fraction of matched-tool calls, or firing identically multiple times in one session without the agent ever acting on it.

**Phase to address:**
Agent-reach phase, Claude Code nudge-hook task.

---

### Pitfall 10: shared-array-entry ownership by shape/position is a REVERTED vulnerability in THIS repo — every new harness's hook/skill install must use exact-identity matching from day one

**What goes wrong:**
This is not a hypothetical: commit `242ec0a` (v0.10.0 Phase 7) introduced a "recovery" heuristic that re-claimed ownership of a JSON hooks array entry by matcher name + shape rather than exact command-string match, was caught by an independent security review, confirmed RED against the vulnerable code, and reverted same-day — recorded in `.planning/PROJECT.md`'s Key Decisions as a durable rule: "Ownership of a shared-array-entry (JSON hooks, config blocks) must be exact-identity, never shape/position." `internal/agents/claude.go`'s `writeHookEntry`/`removeHookEntry` (via `internal/agents/shared.go`) already implement the corrected pattern for Claude — matching on exact command string. Extending nudge/hook mechanisms to Cursor, Gemini CLI, opencode, Kiro, Antigravity, and Hermes (each with its own JSON/TOML/YAML config shape) creates six NEW places this exact bug can be reintroduced independently, because each harness's hook-registration array has a different schema and a developer implementing harness #4 or #5 late in the milestone, working from a different harness's file as a template, can easily "simplify" by matching on array position or matcher-name-only if the target's config shape doesn't obviously support an exact-command-string key the way Claude's does.

**Why it happens:**
Under schedule pressure, six harnesses' worth of hook-splicing code invites copy-paste-and-adapt, and the adapted version is exactly where a subtly-weaker identity check creeps back in — especially for a harness whose own config format doesn't have as clean an "exact command string" field as Claude's `hooks.SessionStart[].hooks[].command`.

**How to avoid:**
For every harness that gains a hook/nudge mechanism this milestone, write the ownership-identity test FIRST, styled after whatever `TestHookRegistrationMatchesFragmentAndScript`-equivalent already exists for Claude (referenced in `claudeHookCommand`'s own doc comment) — specifically a test that plants an unrelated, hand-authored entry occupying the same array slot/shape as codegraph's own entry, then asserts install/uninstall touches ONLY the entry with codegraph's exact identity marker and leaves the hand-authored one byte-for-byte untouched. Do not accept "the shape is different for this harness so the old pattern doesn't transfer" as a reason to weaken the invariant — find the exact-identity equivalent for that harness's shape (a distinguishing field, a marker comment, a fixed literal string) before writing the splice logic.

**Warning signs:**
A new harness's hook-install function that iterates an array and matches by index, by "does this look like our matcher," or by overwriting the first/last entry of a given type rather than searching for an exact previously-written identity string.

**Phase to address:**
Agent-reach phase — one plan per harness gaining a hook mechanism, each carrying its own ownership-identity test, reviewed against the `242ec0a` incident explicitly (cite the commit in the plan's review checklist).

---

### Pitfall 11: assuming per-harness config paths/mechanisms from memory instead of verifying against current 2026 docs — Codex's own config surface changed during 2026 and this repo's `codex.go` already encodes a now-outdated assumption

**What goes wrong:**
`internal/agents/codex.go`'s doc comment states, as fact: "Codex CLI has no per-project config concept, so `SupportsLocation(LocationLocal)` is false." Current (2026-09-14) web research shows this is stale: Codex CLI now discovers project configuration "by walking up from the working directory until it reaches a project root" (a directory containing `.git`), reads AGENTS.md at the project root AND in subdirectories (with subdirectory files taking precedence for conflicts), and supports skills via `[[skills.config]]` in `config.toml` discovered from multiple scoped locations including `$REPO_ROOT/.agents/skills/` — i.e., Codex DOES have a project-local concept today, contradicting the comment's "no per-project config concept" claim the milestone explicitly flags for re-verification ("re-verifying the v1.0-era 'Codex has no per-project config' claim in `codex.go`"). Shipping the Codex-parity phase without re-verifying this against Codex's *own* current documentation (not this search's summary, which is itself community-sourced and should be spot-checked against `openai/codex`'s official docs/CHANGELOG before implementation) risks either (a) leaving `SupportsLocation(LocationLocal)` as `false` when it should now be `true`, missing the entire point of the parity phase, or (b) implementing project-local support against a wrong assumed schema (e.g. wrong `.agents/skills/` vs `.codex/skills/` path, wrong AGENTS.md-vs-AGENTS.override.md precedence) because a community blog post was trusted over the authoritative source.

**Why it happens:**
`codex.go`'s comment was accurate when v0.10.0/v1.0-era work happened; Codex CLI, per multiple 2026 sources, went through visible config-surface changes across the year, and nothing in this repo re-checks external tool documentation on a cadence — a comment stating an external fact silently becomes stale the moment the external tool ships a change, with no compiler or test to catch it.

**How to avoid:**
Before writing any Codex-parity code, read `openai/codex`'s own current docs/CHANGELOG (not a third-party summary) for: (1) whether project-local (repo-root, `.git`-discovered) config is real and what file(s) it lives in; (2) the exact skill-discovery path precedence order; (3) whether a PreToolUse-equivalent hook mechanism exists at all (the milestone's own phrasing — "Codex's nudge/hook mechanism if one exists" — signals this is genuinely unconfirmed, not assumed-yes); (4) AGENTS.md nesting/precedence rules and any size limit. Update `codex.go`'s doc comment in the SAME commit that changes `SupportsLocation`, so the comment cannot silently drift from the code again. Mark anything not verifiable against an authoritative primary source with the same `[ASSUMED]` convention WINDOWS.md #35 already established for the Cursor/JetBrains editor-link templates — this repo already has a working precedent for "ship it, but mark it visibly unverified" rather than blocking on documentation that may not exist in sufficiently precise form.

**Warning signs:**
Any Codex-parity code review or plan that cites "Codex has no per-project config" without a fresh citation dated to this milestone; a `[[skills.config]]` path or AGENTS.md precedence rule hard-coded without a comment linking to the source docs checked.

**Phase to address:**
Codex-parity phase, first task (research/verification), blocking the implementation tasks.

---

### Pitfall 12: "verified in a genuinely fresh live session" is easy to fake for a headless/CLI harness — Codex and opencode don't give you a browser UAT rung, and this repo's own v0.12.0 lesson ("enumerating what exists cannot surface an omission") applies directly

**What goes wrong:**
The milestone requires "each harness verified in a genuinely fresh live session, the v0.10.0 evidence standard" and specifically "verifying in a real Codex session (what 'fresh session' evidence looks like for Codex)." For Claude Code, "fresh session" has a known playbook from v0.10.0 (open a new session, do nothing, observe whether the skill/nudge surfaces unprompted). For a CLI-only harness like Codex, opencode, or Hermes, there's no browser-driven UAT equivalent and no `agent-browser`-skill-shaped tool to drive it (`agent-browser` targets browser windows/webviews, not a terminal-only CLI's internal model reasoning) — so "fresh session" evidence risks degrading to "I ran `codex exec` once and it printed something that looked plausible," which is exactly the shape of self-satisfying verification this repo's own v0.12.0 Key Decisions warn against ("Enumerating what exists cannot surface an omission... a positive control does not help, because the instrument was not broken" — the root-Status-route defect that survived 5/5 verification specifically because the test enumerated what it expected to see rather than checking for what should have been there but wasn't).

**Why it happens:**
CLI harnesses genuinely don't offer the same observability a browser does (no DOM to screenshot, no console to read errors from), so "did the agent actually reach for the skill unprompted, for the right reason, at the right moment" is harder to demonstrate mechanically and easier to accept on weak evidence — especially across 6+ new harnesses where thoroughness per-harness naturally declines as the list gets longer.

**How to avoid:**
For each CLI-only harness, define what "fresh session" evidence means BEFORE running it, in writing, as this milestone's context already implicitly asks for Codex specifically. Minimum bar: a transcript (not a summary) of a genuinely new process invocation, a task phrased the way a real user would phrase it (not "use codegraph to find X" — that's testing whether the tool works, not whether the agent reaches for it unprompted), and explicit before/after evidence that the skill/instructions content was actually loaded into context (e.g. Codex's own `--verbose`/debug output showing the skill file was read, or the agent's own response referencing codegraph-specific vocabulary it could only have gotten from the skill). Apply the same rigor to opencode/Kiro/Antigravity/Hermes — do not let Claude Code's stronger, established playbook be the only harness that gets a real transcript while the rest get "installed cleanly, assumed working."

**Warning signs:**
A verification note that says "skill installs correctly" without a transcript of the skill actually being invoked; per-harness verification depth visibly declining across the 6 non-Claude harnesses in the same PR; no negative-space check (did the agent use grep/find instead of codegraph_explore when it clearly should have reached for the latter) — positive-only enumeration is the exact v0.12.0 trap.

**Phase to address:**
Agent-reach phase, verification task for each harness — budget it as real per-harness work, not a checkbox.

---

### Pitfall 13: `web:drift`'s `find`-vs-`git ls-files` fix (WINDOWS #29) can be "fixed" in a way that still cannot fail

**What goes wrong:**
WINDOWS.md #29 already documents the root cause precisely: the SOURCE half of `web:drift` uses `git ls-files` (git-tree-aware) at `Taskfile.yml:42`, the OUTPUT half uses `find web/build -type f` (filesystem-aware) at `Taskfile.yml:60`, so a build output file that exists on disk but isn't staged is hashed as if committed — passing PASS while the git tree is actually missing files. The suggested fix recorded in the ledger is either "enumerate the output half with `git ls-files web/build`" or "add a paired assertion that the find-derived set equals the git-ls-files-derived set with a non-zero floor on both." The trap: switching the OUTPUT half to `git ls-files web/build` ALONE reintroduces a different vacuity — now BOTH halves are git-tree-aware, so the gate can no longer detect the exact failure mode it exists for (an incompletely-staged bundle), because an unstaged file is now invisible to both sides equally, and they'll always "agree." This would look like a complete fix (the two enumeration methods now unify) while actually removing the gate's only teeth.

**Why it happens:**
"Make both halves use the same method" is the intuitive fix for "the two halves disagree," but the two halves were using DIFFERENT methods on purpose (one measuring intent — what's committed — the other measuring reality — what's on disk) and the bug is that neither side currently cross-checks the other, not that the methods themselves are wrong.

**How to avoid:**
Implement the ledger's second suggested fix, not the first: keep `find web/build -type f` measuring what actually exists on disk (this is the OUTPUT half's job — verify the build produced what it should) AND add `git ls-files web/build` as a SEPARATE check with a paired, non-vacuous assertion that the filesystem set and the git-tracked set are equal (both directions) — this is what actually catches "a file exists on disk but was never staged." Before trusting the fix, run this repo's own standing mutation-log discipline: reproduce the original failure mode (commit 98cd41dd's shape — build 9 new files, stage a bare `web/build` pathspec, confirm some files are silently un-staged) and confirm the fixed gate goes RED, not just that it reads more robust on inspection.

**Warning signs:**
The fixed Taskfile target has only one enumeration method total (both halves now identical); no mutation-log entry demonstrating the fix catches the original real incident (commit 98cd41dd) when replayed.

**Phase to address:**
Guards+flakes burn-down phase (bucket 2), with a mutation-log entry required before closing WINDOWS #29.

---

### Pitfall 14: fixing the daemon watchdog full-suite flake (#12) by widening the timeout masks the real cause — WINDOWS.md #12 already diagnosed it as load-sensitivity, not a deadline that's merely "a bit too tight"

**What goes wrong:**
WINDOWS.md #12 records `TestRunWatchdogCancelsRunOnSimulatedReparent` passing isolated (1.4s), passing as a lone package (64.7s, matching the 65.7s pre-merge baseline), but FAILING at 250s timeout when run inside `go test ./...` alongside ~49 parallel packages — explicitly NOT caused by the phase that surfaced it (zero diff in `internal/daemon`). The obvious "fix" — bump the 250s timeout to 400s or 500s — treats the symptom (a specific number was exceeded) without addressing the cause (the test asserts a real wall-clock watchdog deadline under CPU contention the full suite creates, so it is fundamentally non-deterministic under load, not merely under-provisioned). Any fixed timeout, however generous, remains a flake under sufficiently high parallel load — the fix doesn't converge, it just moves the failure threshold, and the next time CI runners get busier or the test suite grows, it reappears identical.

**Why it happens:**
Widening a timeout is a one-line change with immediate, visible payoff (the flake stops reproducing locally), while root-causing a wall-clock-under-load flake genuinely requires either mocking/injecting the watchdog's time source (so the test controls simulated time rather than racing real wall-clock time) or restructuring the test to not depend on a real deadline being met under contention — both are more invasive changes to `internal/daemon/watchdog.go`'s design, not just the test.

**How to avoid:**
Follow the pattern this repo already used successfully for an analogous class of flake — D-14 (v0.13.0 Phase 8, closing G-08-1): "A wait primitive must take a caller-stated readiness predicate, never rely on interval tuning alone." The watchdog test's fix should be structurally the same idea: make the test assert on a controllable/injectable clock or an explicit synchronization signal (e.g. inject a fake `time.Now`/`time.After` the test can advance deterministically) rather than a real wall-clock race against `getppid` polling (see Pitfall 15) under whatever CPU contention the runner happens to have. If a full clock-injection refactor is out of scope for this milestone, the acceptable minimal fix is: run this specific test with `t.Parallel()` disabled AND `GOMAXPROCS`-isolated (its own `go test` invocation in CI, not inside the shared `go test ./...` run) — a scoping fix, not a timeout-widening one — with a comment explaining why, so a future maintainer doesn't "helpfully" fold it back into the shared run.

**Warning signs:**
A diff that touches only a numeric timeout constant in `watchdog.go` or `daemon_test.go` with no change to how the test observes time; the fix commit's message doesn't reference WINDOWS.md #12's specific finding (load-sensitivity, not tightness).

**Phase to address:**
Guards+flakes burn-down phase (bucket 2).

---

### Pitfall 15: the `getppid` test seam (`var getppid = os.Getppid`) is a package-level mutable global — GH #17's data race is inherent to the pattern, not incidental

**What goes wrong:**
`internal/daemon/watchdog.go` defines `var getppid = os.Getppid` specifically so tests can swap in a fake — a standard Go test-seam pattern. But it's a **package-level `var`**, not per-test or per-instance state. Any test that reassigns `getppid` (to simulate reparenting) while ANOTHER test in the same package is concurrently reading it (either via `t.Parallel()` or via the watchdog goroutine itself still running from a previous subtest that didn't fully tear down) is racing on a plain, unsynchronized global function variable — precisely the shape `go test -race` is built to catch, and precisely why GH #17 exists. Fixing the symptom (making the specific failing test not run in parallel) does not fix the underlying design defect: the seam itself is unsafe for ANY future test author who doesn't know to serialize against it, and a new test added later (plausible during this very burn-down milestone, since other daemon-adjacent flakes are also in scope) can reintroduce the race independently.

**Why it happens:**
`var fn = os.SomeFunc` is the simplest, most common Go test-seam idiom, and it's genuinely fine as long as exactly one goroutine/test touches it at a time — a property that's easy to hold when the package has one test file and breaks silently as more tests accumulate around the same seam without anyone re-auditing concurrency assumptions.

**How to avoid:**
Either (a) make the seam per-instance rather than package-global — thread a `getppid func() int` field through whatever struct owns the watchdog goroutine, set once at construction, never mutated after — which makes the race structurally impossible rather than merely avoided by test discipline; or (b) if a package-global seam must be kept for minimal-diff reasons, add explicit synchronization (a `sync.Mutex` guarding reads/writes, or restrict all tests touching it to run serially via a shared `TestMain`-level lock) and a comment at the `var` declaration warning future test authors. Prefer (a) — it's this repo's own stated preference pattern ("remove the code path rather than guard it," D-02, Phase 4 of v0.5.0) applied to a race instead of a force-flag.

**Warning signs:**
`go test -race ./internal/daemon/...` flagging the exact `getppid` var; a new test added to `internal/daemon` during this same burn-down milestone that reassigns `getppid` without first checking whether any other test in the package does the same.

**Phase to address:**
Guards+flakes burn-down phase (bucket 2), same plan as the watchdog timeout fix — they're adjacent code and likely the same root investigation.

---

### Pitfall 16: `CheckRegression` never comparing `Metrics.Repo` (GH #16) — the naive fix ("just add a Repo equality check") will go red on legitimate corpus renames unless it's structured like the existing GOOS/Runner/ScratchFS checks

**What goes wrong:**
Confirmed by reading `internal/bench/regression.go`: `CheckRegression` already has three "category error, not tolerance" guards — GOOS/GOARCH, Runner, ScratchFS — each following the identical pattern (mismatch on a non-empty-vs-non-empty pair is refused with a message pointing at the correct re-bless mechanism; an empty value on either side is treated as "never recorded," not a wildcard). `Metrics.Repo` (confirmed present as a field in `internal/bench/metrics.go`) is comparable across baseline/current with zero such guard today, meaning a baseline measured against one corpus and a current run measured against a differently-named (or entirely different) corpus can pass or fail the throughput/RSS deltas as if they were the same measurement — the exact "fictitious regression/false pass across an incomparable measurement frame" class this file's own doc comments say this repo already got burned by once (the GOOS/GOARCH cross-platform incident, "a stable, reproducible, entirely fictitious ~10.6% regression... survived three rounds of triage"). The naive fix — `if baseline.Repo != current.Repo { return error }` — would ALSO fire on a corpus that's the same underlying repo but was re-cloned to a differently-named directory, or renamed/relocated between baseline and current measurement (e.g. `google/guava` vs `guava` vs an absolute path vs a relative path), producing a false-positive refusal on a legitimate comparison and training people to route around the check rather than trust it.

**Why it happens:**
`Repo` looks, at a glance, like it should just be a strict string-equality field the same shape as `Runner`/`ScratchFS` — the existing three checks are a strong nearby template that biases toward copy-paste without asking whether `Repo`'s semantics (identifying WHICH CORPUS was measured, for comparability of the *workload*, not the *environment*) need path-normalization or corpus-identity-by-content-hash rather than raw string equality the way an environment-class string can be compared literally.

**How to avoid:**
Before adding the check, determine what `Repo` is actually populated with today (a path? a URL? a short name?) by reading every write site, then decide: if it's already normalized to a stable corpus identifier at write time, a direct equality check following the existing three-guard template (including the "empty means never recorded, not wildcard" rule) is correct and sufficient. If it can vary in representation for the same underlying corpus (path vs URL vs relative/absolute), either normalize at write time (preferred — keeps the comparison simple and matches this repo's stated preference for fixing causes over guarding symptoms) or compare a derived stable identifier instead of the raw field. Either way, add the SAME message-quality bar the other three checks have: point at the correct re-bless mechanism, not just "mismatch."

**Warning signs:**
A GH #16 fix PR that adds `if baseline.Repo != current.Repo` with no investigation into what values `Repo` actually holds across the write sites that populate `Metrics`; a subsequent CI failure on an otherwise-legitimate regression run right after the fix lands, attributable to corpus path representation differing between the baseline-recording job and the comparison job.

**Phase to address:**
Burn-down bucket 4 (GH #16), with a short research/read-the-write-sites step before the fix, mirroring how the existing three checks each cite the specific incident (`tools/bench/BASELINE.md`) that justified their exact shape.

---

### Pitfall 17: the `pull_request_target` heredoc fix (GH #15) must not just change the delimiter — the vulnerability class is GitHub Actions multi-line-output injection, and a "safer" fixed delimiter is still guessable

**What goes wrong:**
Confirmed by reading `.github/workflows/require-issue-link.yml` and `pr-template-format.yml`: both use `echo "list<<PRFILES_EOF"` / `echo "files<<PRFILES_EOF"` — the GitHub Actions multi-line `GITHUB_OUTPUT` heredoc idiom — with a FIXED, PREDICTABLE delimiter (`PRFILES_EOF`) to write a list of changed file paths sourced from `pull_request_target`-triggered context (fork-controlled: the PR's changed files, which an attacker fully controls by naming a file `PRFILES_EOF` or including that string in a file's content, depending on exactly how `CHANGED_FILES` is populated upstream). The naive fix — rename the delimiter to something else fixed, e.g. `PRFILES_END_MARKER_V2` — does not fix the vulnerability class, it just requires the attacker to guess (or read from this now-public source file) the new literal string, which they trivially can since the workflow file itself is public. The actual fix needs either (a) a delimiter that is NOT a fixed literal (e.g. a random/UUID-based delimiter generated per-run, `EOF_$(uuidgen)` or `EOF_${{ github.run_id }}_${{ github.run_attempt }}`), or (b) avoiding the heredoc-multi-line pattern entirely for fork-controlled content (write to a temp file and reference it, or use `actions/github-script` to set outputs via the JS API rather than shell string interpolation, which sidesteps the delimiter-collision class altogether).

**Why it happens:**
The heredoc-with-fixed-delimiter pattern is copy-pasted verbatim from GitHub's own official multi-line-output documentation examples, which use a fixed delimiter (`EOF`) for illustration without flagging that a fixed delimiter is unsafe specifically when the content between the delimiters can be influenced by an untrusted party (a `pull_request_target` file list is exactly that case) — the docs' example is fine for trusted content and silently wrong for this specific trigger type.

**How to avoid:**
Generate the delimiter per-run from an unpredictable-to-the-attacker source (job/run id combined with a random component is sufficient and simple; a full UUID is safer and just as easy: `DELIM="EOF_$(openssl rand -hex 16)"`), and — separately — sanity-check that the list content itself can't otherwise smuggle a `GITHUB_OUTPUT`-poisoning line even with an unguessable delimiter (e.g. if a file path could itself contain a literal newline, some encodings could still misbehave; prefer writing paths null-separated or one-per-`printf` rather than relying purely on the delimiter uniqueness). Also fix `pr-template-format.yml`'s IDENTICAL pattern in the SAME change — both files share the exact vulnerable shape, and fixing only the one GH #15 named while leaving its sibling untouched is exactly the kind of miss this repo's census discipline exists to catch.

**Warning signs:**
A fix PR touching only `require-issue-link.yml` and not `pr-template-format.yml`; the new delimiter is still a fixed string literal (even a longer/uglier one) rather than something generated per-run.

**Phase to address:**
Burn-down bucket 4 (GH #15), and explicitly include `pr-template-format.yml` in the same fix's scope.

---

### Pitfall 18: the stock Svelte favicon + CSP fix (#30) — widening `img-src` to include `data:` "fixes" the symptom but contradicts the repo's own deliberate, tested CSP posture

**What goes wrong:**
WINDOWS.md #30 is unusually explicit that the correct fix is NOT to widen CSP: "Adding `img-src data:` to the CSP would also work but widens the policy for a cosmetic asset," and separately, "spa_test.go:439 asserting default-src is exactly ['self'] is correct and deliberate (T-02-02-06); the asset choice is the defect." A CSP-widening fix is the path of least resistance (one line, `img-src 'self' data:;`, in `spa.go`) and would make the visible symptom (broken favicon) disappear — but it also (a) contradicts this repo's own recorded threat-model rationale for the tight default-src policy, (b) will make `spa_test.go:439`'s existing assertion either need editing (a deliberate, tested security posture being loosened to fix a cosmetic bug) or start failing for the wrong reason if left unedited, and (c) leaves the STOCK SVELTE LOGO shipping in a tool called codegraph even after the CSP violation stops appearing in the console — the ledger entry names TWO independent defects ("two independent defects that happen to cancel into 'no visible logo'"), and a CSP-only fix addresses neither the branding embarrassment nor the actual root asset problem.

**Why it happens:**
CSP-widening is a one-line, easily-verified-working fix (open the browser console, confirm the error is gone) that doesn't require touching build tooling (Vite's asset-inlining threshold) or design (a real codegraph mark), which is a much larger effort surface for what looks, superficially, like a small cosmetic bug.

**How to avoid:**
Implement WINDOWS.md #30's own recorded fix path: replace the asset with an actual codegraph mark, AND ship it as a static file under `web/static/` (served as `'self'`, no CSP change needed) rather than relying on Vite's default small-asset inlining (which produces the `data:` URI that trips CSP in the first place) — OR raise Vite's inline-asset threshold specifically for this one asset so it's emitted as a real file. Either path fixes BOTH defects (branding AND CSP-compliant loading) with zero CSP policy change, preserving `spa_test.go:439`'s existing assertion untouched. If a CSP change is genuinely chosen instead (contradicting the ledger's own recommendation), that decision needs to be an explicit, reasoned override recorded in the phase's design docs, not a silent one-liner — this repo's practice (see the `bump-minor-pre-major` and D-04 migrate-removal precedents) is that overriding a previously-recorded rationale must be recorded, not silently done.

**Warning signs:**
A fix PR that touches `spa.go`'s CSP string and does NOT touch `web/src/lib/assets/favicon.svg` (fixes the console error, leaves the Svelte logo); `spa_test.go:439` edited without a design-doc note explaining why the previously-"correct and deliberate" assertion changed.

**Phase to address:**
User-facing-defects burn-down (bucket 1), WINDOWS #30.

---

### Pitfall 19: bubbles v2 list picker footer overflow (#32) — the "fix" needs to be measured against the real 100x30 pane height budget, not guessed and re-tested only at default terminal size

**What goes wrong:**
WINDOWS.md #32 already root-caused this precisely: "the footer never renders in the default 100x30 pane with all 8 registered agent targets (bubbles v2 list pagination padding overflows its allocated height before the footer is appended)." The obvious "fixes" — reduce padding, hide pagination dots, shrink the title — are all plausible but each interacts with `SetHeight`/`SetShowPagination`/`SetShowHelp` (the actual bubbles v2 list APIs, confirmed via search) differently, and none of them is provably correct without measuring against the SAME test harness that found the bug: the tmux real-PTY e2e harness (v0.13.0 Phase 8) at the SAME 100x30 pane dimensions with the SAME 8-target list (this milestone adds MORE harnesses via AGENT-04…07, so if agent-reach work lands first and adds a 9th target to the install picker before this fix ships, the height budget shifts again and a fix measured against 8 targets may not hold against 9). WINDOWS.md #32 is filed against `test/tmux/install_cancel_test.go`'s assertion currently checking the picker TITLE instead of the footer text specifically BECAUSE the footer doesn't render — a naive fix might restore the footer-text assertion without re-confirming the footer actually renders reliably across the now-larger (post-agent-reach) target count, silently re-introducing the same overflow one target later.

**Why it happens:**
Bubbles v2's list height/pagination/help interaction is a layout-budget problem (item rows + pagination row + help row must all fit inside `SetHeight`'s value), and layout budget bugs are notoriously easy to "fix" for the specific input that was tested (8 items) while remaining broken for a slightly different input (9 items) — exactly the shape of fix this milestone risks given agent-reach work is scoped in the SAME milestone and could grow the target list.

**How to avoid:**
Sequence the fix AFTER (or re-verify it after) any target-list growth from AGENT-04…07/Codex-parity work lands, so the picker is tested against its FINAL target count for this milestone, not today's 8. Fix via `SetHeight` sized to actually accommodate title + N items + pagination + help rows (compute the budget explicitly rather than trial-and-error against one observed count), and restore `install_cancel_test.go`'s assertion to check the actual help footer text (not just the title) as the positive control — the current title-only assertion is itself a weaker guard than the milestone's own "burn down every known... vacuous guard" goal implies it should tolerate keeping. Re-run the tmux harness at 100x30 (this repo's own established pane size) as the acceptance check, not a wider/taller pane that would make the bug disappear without fixing the real-world default size.

**Warning signs:**
A fix verified only against today's 8-target list, landing before or without re-verification against the agent-reach phase's possibly-larger target count; `install_cancel_test.go` still asserting only the title after the "fix."

**Phase to address:**
User-facing-defects burn-down (bucket 1) — but sequence it to run its final verification AFTER the agent-reach phase's target-count changes are known, even if the code fix itself lands earlier.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|------------------|
| Hand-roll a second `NO_COLOR`/`CLICOLOR_FORCE` env check instead of adopting `colorprofile.Detect` | No new dependency, small diff | Precedence bugs (Pitfall 2), no real colour downsampling (Pitfall 1), a second source of truth to keep in sync with any future profile-aware rendering | Never, once real colour (not just bold/faint/underline) is added — the milestone's own colour-palette goal requires downsampling, which only `colorprofile` (or equivalent) provides correctly |
| Ship the verb-rename stubs as a `root.go` special-case string match in `Execute()` rather than real `*cobra.Command` registrations | Faster to write, no flag/help machinery to wire up | Invisible to `docs:cli:drift`, `cli_reference_test.go`'s flag-accounting floors, shell completion, and `man` generation — silently un-discoverable by every mechanism that walks the real cobra tree | Never — register real (hidden if desired) commands so every existing generator/guard covers them for free |
| Widen a CI timeout or CSP directive to make a burn-down item's test go green | Fast, low-risk-looking, doesn't touch product code | Masks the real defect (Pitfalls 14, 18); the next milestone re-discovers the same root cause under a new symptom | Never for this milestone's explicit "burn down... never mask" goal — acceptable ONLY as a documented, deliberately-recorded interim measure with the real fix tracked, matching this repo's own "accepted limitation" pattern (e.g. the daemon extreme-load timeout tail already carried as accepted, not silently widened without a note) |
| Copy Claude's hook-splice code to a new harness verbatim, adapting only the file path | Fast harness coverage, consistent code shape | Reintroduces the exact-identity-ownership bug class per-harness if the new harness's config shape doesn't have as clean an identity field (Pitfall 10) | Never without writing that harness's own ownership-identity test first |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|-----------------|-------------------|
| Claude Code PreToolUse hooks | Using `permissionDecision: "ask"` or any non-zero exit to implement a "nudge," reintroducing the friction GUARD-HOOK-01/02 was reframed to avoid | `additionalContext` only, no `permissionDecision` key, unconditional exit 0 (Pitfall 8) |
| Codex CLI config/skills surface | Trusting this repo's own existing `codex.go` doc comment ("no per-project config") without re-checking Codex's current (2026) official docs | Re-verify against `openai/codex`'s own docs/CHANGELOG before implementing; update the doc comment in the same commit as any code change (Pitfall 11) |
| Six new harnesses' hook/skill config formats (Cursor, Gemini CLI, opencode, Kiro, Antigravity, Hermes) | Assuming each harness's config shape/discovery path from memory or from one already-implemented harness as a template | Verify per-harness against that harness's own current docs; do not extrapolate Claude's JSON shape onto a TOML/YAML target uncritically |
| lipgloss v2 + `colorprofile` | Calling `Style.Render()` directly and printing the result, assuming it downsamples like v1 did | Route every render through `colorprofile.Writer`/`lipgloss.Println`/`Fprintln`/`Sprint` at the same call sites that already own env/tty reads (Pitfall 1) |
| `charmbracelet/fang` (if adopted) | Wiring `fang.Execute` in without updating the `present/archtest` denylist, since fang itself imports lipgloss v2 transitively | Add fang's exact import path to `forbiddenImportPaths` (or convert to a prefix match) in the same commit fang is adopted (Pitfall 3) |
| `pull_request_target` + `GITHUB_OUTPUT` multi-line heredoc | Fixed, guessable delimiter over fork-controlled content | Per-run generated/random delimiter, applied to every workflow sharing the pattern, not just the one GH issue named (Pitfall 17) |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| Daemon watchdog wall-clock deadline under CI load | `TestRunWatchdogCancelsRunOnSimulatedReparent` flakes only inside the full `go test ./...` run, never in isolation | Inject a controllable time source instead of racing real wall-clock time under contention (Pitfall 14) | Any time the full-suite parallelism or runner class changes — a timeout bump only postpones recurrence |
| `getppid` package-global test seam | `go test -race` flags concurrent reads/writes of `var getppid` | Make the seam per-instance, not package-global (Pitfall 15) | As soon as a second test in the package touches the seam concurrently — plausible this very milestone given other daemon-adjacent fixes are in scope |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Fixed/guessable `GITHUB_OUTPUT` heredoc delimiter over `pull_request_target` (fork-controlled) content | A malicious PR crafts a changed-file path/content that closes the heredoc early and injects arbitrary additional `GITHUB_OUTPUT` key/value pairs, potentially influencing downstream steps that trust that output in a privileged (`pull_request_target`-scoped, secrets-bearing) context | Per-run unpredictable delimiter (or avoid the shell-heredoc pattern entirely for untrusted content); fix in BOTH `require-issue-link.yml` and `pr-template-format.yml`, which share the identical vulnerable shape (Pitfall 17) |
| CSP-widening as the "fix" for the favicon defect | Loosens a deliberately tight, tested `default-src 'self'` policy (T-02-02-06) for a cosmetic reason, setting a precedent that CSP gets widened whenever it's inconvenient | Fix the asset (static file under `web/static/`, served as `'self'`), not the policy (Pitfall 18) |
| Shape/position-based ownership of a shared hook/config array entry, reintroduced per-harness | A hand-edited entry occupying the same slot/shape as codegraph's own gets silently overwritten — the exact vulnerability class reverted in commit `242ec0a` | Exact-identity matching (marker string/field), verified by a planted-foreign-entry test, for every NEW harness gaining a hook mechanism this milestone (Pitfall 10) |
| PreToolUse hook exiting non-zero on an internal error path | A "nudge" silently becomes a hard block (exit 2 blocks unconditionally, JSON or not), degrading UX for reasons invisible to the user and contradicting the milestone's explicit "never denies" design goal | Fail-open: any internal hook error falls through to exit 0 with no `additionalContext`, never a non-zero exit (Pitfall 8) |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-------------------|
| Verb rename without a census of prose references (docs, embedded skill/nudge assets, CI scripts) | A user follows a README or skill instruction referencing `codegraph query`/`codegraph unlock`, hits the one-release stub's non-zero exit, and has to guess the new verb from an error message rather than the doc they were reading being correct in the first place | Whole-repo literal census before editing `internal/cli/`, covering embedded assets and prose, not just Go source (Pitfall 5) |
| Nudge hook firing on every unrelated Bash/Grep/Read call | Constant low-signal `additionalContext` injections train the agent (and any human reading transcripts) to ignore the nudge entirely, defeating its purpose | Gate on indexed-repo presence AND a content heuristic, not tool-name alone (Pitfall 9) |
| `search --full` silently keeping `search`'s narrower flag set (no `-j/-l/-k`) | A user who habitually typed `codegraph query -j term` gets an "unknown shorthand flag" error after upgrading, with no clear migration path from the error alone | Keep the flag superset across the merge; test both shorthand and long forms on both `search` and `search --full` (Pitfall 4) |
| Install picker footer silently missing help text at the terminal's actual default size | A user driving the real picker in a real 100x30 terminal never sees keybinding help, discoverable only by accident or by reading source | Fix measured against the real pane size and the FINAL (post-agent-reach) target count, not a synthetic wider pane or today's smaller target list (Pitfall 19) |

## "Looks Done But Isn't" Checklist

- [ ] **CLI colour glow-up:** Often missing real downsampling — verify manually in a `TERM=xterm` (non-256color) session and over an SSH connection with a constrained `TERM`, not just the developer's own modern terminal emulator.
- [ ] **`present/archtest` ANSI-isolation guard:** Often missing coverage of NEW Charm-family dependencies (fang, colorprofile, x/ansi) — verify `forbiddenImportPaths` (or its prefix-match replacement) was updated in the same diff as any new `go.mod` Charm require, and that the guard was proven RED against a deliberate mutation importing the new package from a guarded package.
- [ ] **Verb rename (`query`→`search --full`, `unlock`→`daemon unlock`):** Often missing prose references outside `internal/cli/` — verify a whole-repo grep for the old verb names (word-boundary, excluding the unrelated `internal/query` package identifier) returns zero hits outside the one-release stub and its own tests.
- [ ] **Claude Code nudge hook:** Often missing the "never blocks" guarantee — verify by forcing every internal error path in the hook script and confirming exit code is always 0 with no `permissionDecision` key ever emitted.
- [ ] **Per-harness skill/nudge install (all 8 targets):** Often missing a genuine fresh-session transcript — verify each of the 6 non-Claude harnesses has a real transcript (not a summary) showing the skill/instructions were actually loaded, matching the depth of evidence Claude Code's own v0.10.0 verification set.
- [ ] **Codex parity:** Often missing re-verification of `codex.go`'s stale "no per-project config" claim against Codex's own current docs — verify the doc comment and `SupportsLocation(LocationLocal)` were changed together, with a dated citation.
- [ ] **`web:drift` gate fix:** Often missing a mutation-log RED demonstration — verify the fix is proven to catch the exact original incident (commit 98cd41dd's shape: new files built but not staged) by deliberately reproducing it against the fixed gate.
- [ ] **Favicon/CSP fix (#30):** Often missing the actual asset replacement — verify the shipped fix touches `web/src/lib/assets/favicon.svg` (or its static-file replacement), not only `spa.go`'s CSP string.
- [ ] **`CheckRegression` Metrics.Repo comparison (GH #16):** Often missing investigation of what values `Repo` actually holds across write sites — verify the fix doesn't fire false-positive on a legitimate corpus path/representation difference between baseline and current runs.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|-----------------|-------------------|
| `present/archtest` denylist misses a new Charm dependency | LOW | Add the missing import path (or convert to prefix match), re-run the archtest, confirm RED-then-fixed via a temporary deliberate-violation mutation, land as a follow-up patch |
| Verb-rename prose census misses a file (a WINDOWS.md-shaped deviation, not a blocker) | LOW–MEDIUM | Log a WINDOWS.md entry per this repo's own established discipline (see ids 4–11, 14–19 for the precedent), fix in a fast-follow, close the entry with a resolved_at timestamp |
| A nudge hook ships with `permissionDecision` wired in error | MEDIUM | Revert the hook's Claude Code registration (uninstall + reinstall via the existing `writeHookEntry`/`removeHookEntry` exact-identity mechanism) rather than hand-editing `settings.json`; re-ship with `additionalContext`-only |
| `web:drift` "fix" turns out to still be vacuous (both halves unified onto git-tree-only enumeration) | MEDIUM | Re-open WINDOWS #29, do not mark it fixed until the paired-assertion form (filesystem set == git-tracked set, both directions) is in place and mutation-log-proven |
| Codex-parity work implemented against a wrong assumed config schema | MEDIUM–HIGH | Re-verify against Codex's primary docs, patch the schema assumption, re-run the exact-identity ownership test (Pitfall 10) since any TOML-splice change risks the same class of regression `spliceTOMLTable`/`stripTOMLTable` already guard against |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|-------------------|----------------|
| 1. lipgloss v2 renderer removal / no auto-downsampling | CLI glow-up | Manual render check in a non-256-colour `TERM`; automated test asserting `colorprofile`-derived fidelity, not just isTTY binary |
| 2. NO_COLOR/CLICOLOR_FORCE precedence | CLI glow-up | Unit test matrix: all 4 combinations of NO_COLOR/CLICOLOR_FORCE set/unset |
| 3. `present/archtest` denylist rot on new Charm deps | CLI glow-up | RED-then-fixed mutation log entry for any newly adopted Charm-family package |
| 4. `query`→`search --full` merge losing flags/shape | Verb-fold | Flag-parse test on both `search` and `search --full` covering `-j/-l/-k`; JSON-shape test distinguishing full-record vs location-only |
| 5. Verb-rename prose census gaps | Verb-fold | Whole-repo word-boundary grep for old verb names returning zero hits outside the stub, run BEFORE and AFTER the edit |
| 6. `feat!:` vs `bump-minor-pre-major` confusion | Verb-fold | PR description/commit message review checklist item; CHANGELOG.md entry inspected post-merge for correct BREAKING CHANGES categorization |
| 7. Golden/wire oracle re-freeze needing RED demonstration | Verb-fold | Mutation-log entry proving the re-frozen fixture still fails against a reintroduced old-verb-name |
| 8. Nudge hook accidentally blocking | Agent reach (Claude Code) | Test asserting exit 0 always, no `permissionDecision` key ever emitted |
| 9. Nudge firing too broadly (noise) | Agent reach (Claude Code) | Fresh-session transcript measuring fire-rate against matched-tool-call count |
| 10. Shape-based ownership reintroduced per-harness | Agent reach (all 6 non-Claude harnesses) | Planted-foreign-entry test per harness, citing commit `242ec0a` in review |
| 11. Codex config-surface assumptions stale | Codex parity | Doc-comment + code changed together, dated citation to Codex's own current docs |
| 12. Weak "fresh session" evidence for CLI-only harnesses | Agent reach + Codex parity | Real transcript per harness, not a summary; negative-space check (did it reach for grep instead) |
| 13. `web:drift` fix still vacuous | Guards+flakes burn-down | Mutation-log replay of commit 98cd41dd's exact incident against the fixed gate |
| 14. Watchdog flake fixed by widening timeout | Guards+flakes burn-down | Fix touches time-source injection or test isolation, not a bare constant; comment cites WINDOWS #12 |
| 15. `getppid` global-var data race | Guards+flakes burn-down | `go test -race ./internal/daemon/...` clean; seam made per-instance or explicitly synchronized |
| 16. `CheckRegression` Metrics.Repo naive equality check | Docs+CI wiring / GH #16 burn-down | Write-site investigation documented; false-positive check against a legitimate path-representation difference |
| 17. `pull_request_target` heredoc delimiter fix incomplete | GH #15 burn-down | Fix applied to BOTH `require-issue-link.yml` and `pr-template-format.yml`; delimiter is per-run generated, not a longer fixed literal |
| 18. Favicon/CSP fix widens policy instead of fixing the asset | User-facing defects burn-down (#30) | `spa_test.go:439` unchanged; new asset served as static `'self'` file |
| 19. Bubbles v2 footer fix measured against stale target count | User-facing defects burn-down (#32) | Final verification sequenced after agent-reach target-count changes are known; tmux harness re-run at 100x30 |

## Sources

- `.planning/PROJECT.md` (repo-internal, HIGH confidence) — Current Milestone v0.14.0 scope, Key Decisions table (rules `84d1gfpywd`, D-04, D-06R, the `242ec0a` shared-array-ownership revert, D-14/G-08-1 wait-primitive fix, the `bump-minor-pre-major` config)
- `.planning/WINDOWS.md` (repo-internal, HIGH confidence) — ids 12, 13(paired with #17 via daemon), 26, 28, 29, 30, 31, 32, 36 read in full
- `internal/cli/present/archtest/import_graph_test.go`, `internal/cli/present/styles.go`, `internal/cli/present/tty.go` (repo-internal, HIGH confidence, read directly) — ANSI-isolation guard shape, `forbiddenImportPaths` literal list, `ChoosePresentation` binary gate
- `internal/cli/query.go`, `internal/cli/search.go`, `internal/cli/unlock.go`, `internal/query/search.go` (repo-internal, HIGH confidence, read directly) — `matchNodes`/`Query`/`Search` shared-vs-divergent shape, flag shorthand asymmetry
- `internal/cli/cli_reference_test.go` (repo-internal, HIGH confidence, read directly) — flag-accounting allowlist mechanism and positive floors (`cliReferenceMinCommands=26`, `cliReferenceMinFlags=50`)
- `internal/agents/claude.go`, `internal/agents/codex.go`, `internal/agents/toml.go` (repo-internal, HIGH confidence, read directly) — exact-identity hook ownership mechanism, Codex's current (in-repo, possibly-stale) global-only assumption, hand-rolled TOML splice
- `internal/bench/regression.go`, `internal/bench/metrics.go` (repo-internal, HIGH confidence, read directly) — `CheckRegression`'s existing GOOS/Runner/ScratchFS guard pattern and confirmed absence of a `Repo` comparison
- `internal/daemon/watchdog.go` (repo-internal, HIGH confidence, read directly) — `var getppid = os.Getppid` package-global test seam
- `.github/workflows/require-issue-link.yml`, `.github/workflows/pr-template-format.yml` (repo-internal, HIGH confidence, read directly) — fixed-delimiter `GITHUB_OUTPUT` heredoc pattern in both files
- [lipgloss `UPGRADE_GUIDE_V2.md`](https://github.com/charmbracelet/lipgloss/blob/main/UPGRADE_GUIDE_V2.md) (official, MEDIUM confidence, web search 2026-09-14) — renderer removal, `Style.Render()` always full-fidelity, downsampling moved to print-time via `colorprofile`
- [`charm.land/lipgloss/v2` on pkg.go.dev](https://pkg.go.dev/charm.land/lipgloss/v2) (official, MEDIUM confidence)
- [`charmbracelet/colorprofile` GitHub + pkg.go.dev](https://github.com/charmbracelet/colorprofile) (official, MEDIUM confidence, web search 2026-09-14) — `Detect`, `Profile` enum (NoTTY/Ascii/ANSI/ANSI256/TrueColor), NO_COLOR-beats-CLICOLOR_FORCE precedence, `TERM=dumb` → NoTTY unless `CLICOLOR_FORCE=1`
- [CMake `NO_COLOR`/`CLICOLOR_FORCE` envvar docs](https://cmake.org/cmake/help/latest/envvar/CLICOLOR_FORCE.html) (third-party but spec-citing, MEDIUM confidence) — cross-check for NO_COLOR > CLICOLOR_FORCE > CLICOLOR precedence and the note that other ecosystems (Node.js) implement it backwards
- [`no-color.org`](https://no-color.org/) (referenced by the above sources; not independently fetched this session — cite directly before implementation)
- [Claude Code Hooks reference](https://code.claude.com/docs/en/hooks) (official, MEDIUM confidence, web search 2026-09-14) — `hookSpecificOutput.permissionDecision` (allow/deny/ask/defer), `additionalContext`, exit-code-2-blocks-unconditionally semantics
- [`charmbracelet/fang` on pkg.go.dev](https://pkg.go.dev/github.com/charmbracelet/fang) and [`charm.land/fang/v2`](https://pkg.go.dev/charm.land/fang/v2) (official, MEDIUM confidence, web search 2026-09-14) — cobra help/error/version styling wrapper, built on lipgloss v2
- [`charm.land/bubbles/v2` list component on pkg.go.dev](https://pkg.go.dev/charm.land/bubbles/v2) (official, MEDIUM confidence, web search 2026-09-14) — `SetHeight`, `SetShowPagination`, `SetShowHelp` APIs relevant to the footer-overflow fix
- Codex CLI config/AGENTS.md/skills sources (community-sourced, **LOW-MEDIUM confidence, `[ASSUMED]` per this repo's own WINDOWS.md #35 convention** — re-verify against `openai/codex`'s own primary docs before implementation): [OpenAI Codex "Advanced Configuration"](https://developers.openai.com/codex/config-advanced), [Codex CLI Cheatsheet (Shipyard)](https://shipyard.build/blog/codex-cli-cheat-sheet/), [Codex CLI Customisation Stack](https://codex.danielvaughan.com/2026/04/12/codex-cli-customisation-stack-unified-system/), [Codex CLI skills storage (Loadout)](https://loadout.migsilva.dev/guides/where-are-codex-cli-skills-stored/), [Codex CLI Skills & AGENTS.md Setup Guide 2026](https://www.agensi.io/learn/codex-cli-agents-md-complete-guide)
