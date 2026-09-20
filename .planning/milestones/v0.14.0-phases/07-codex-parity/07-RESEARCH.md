# Phase 7: Codex Parity - Research

**Researched:** 2026-09-19
**Domain:** Go CLI agent-installer internals (hand-rolled TOML splice, bubbles/v2 TUI, Codex CLI hooks/skills/config mechanics)
**Confidence:** HIGH for in-repo code paths (all read this session); MEDIUM-HIGH for Codex CLI mechanics (Context7 `/openai/codex` + `/llmstxt/learn_chatgpt_llms-full_txt`, cross-checked against 6 live `codex` CLI probes on codex-cli 0.155.0, 2026-09-19); LOW/ASSUMED only where flagged.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

Carried forward (binding): D-00 (tests assert only what this repo owns; every new guard positive-controlled in `07-MUTATION-LOG.md`; Go RED evidence is a `test(07-NN):` commit + pasted `--- FAIL:`); the nudge contract carried verbatim from Phase 6 (`hookSpecificOutput.additionalContext` only, exit 0 always, first-match-then-60s-cooldown-per-session-and-subagent, hidden `codegraph hook pretooluse` behind a POSIX `sh` guard with ExecPath rendered into the guard).

Plan order (D-01): (1) TOML fix (D-07, D-08); (2) FIX-03 (D-24, D-25), before CODEX-02; (3) CODEX-01 live verification before any `codex.go` change, correcting the "no per-project config" comment in the same commit as the first `codex.go` change; (4) scope flip + companions (CODEX-02/03/04); (5) nudge (CODEX-05); (6) CODEX-06 live uptake + re-run picker tmux assertion; (7) AGENT-14 docs, last.

A. Live verification (CODEX-01, CODEX-06): D-02 (scratch `HOME`/`CODEX_HOME`, symlinked `auth.json`, real-HOME checksums unchanged); D-03 (model-free evidence tiers: `codex mcp list --json`, `codex debug prompt-input`, `codex features list`; model sessions only for CODEX-06/CODEX-05); D-04 (`codex exec --json -C <repo>` + session JSONL, backed by one interactive Herdr-pane TUI session for the real trust prompt); D-05 (untrusted-first negative control, then TUI-trusted; `-c 'projects."<path>".trust_level="trusted"'` override recorded separately); D-06 (L1–L7 pass bar, locked before any session).

B. Project-local scope, TOML splice, shared AGENTS.md (CODEX-02, CODEX-04): D-07 (fix TOML splice at cause, RED-first, ANY-indentation header scan, never inside multi-line strings/arrays, CRLF preserved, own subtables in-range, inline/dotted `mcp_servers.codegraph.*` refused with an error, planted-regression mutation); D-08 (released binaries carry the data-loss bug, no patch release now — maintainer must not run `install`/`uninstall --target codex|all` until fixed); D-09 (scope flip through the capability table: `Scopes {global, local}`, per-location `MCPConfig`/`Instructions`, `SkillDirs = sharedSkillDirs`, `Hooks = codex-json` with a `HookFiles` case); D-10 (every local Codex install adds a `WriteResult.Notes` trust reminder); D-11 (shared `AGENTS.md` marker removed only when no OTHER target still reports `Detect(loc).AlreadyConfigured`); D-12 (`AGENTS.override.md` shadows for Codex; install adds a Note, override never written); D-13 (fix `install --yes` discarding `--target`, RED-first, uninstall too).

C. Skill path (CODEX-03): D-14 (Codex writes shared `.agents/skills/codegraph` local / `~/.agents/skills/codegraph` global via `sharedSkillDirs`; no `.codex/skills` write — no merge, so a second copy would double-list); D-15 (`.codex/skills`/`$CODEX_HOME/skills` listed as read-only `SkillDirs[1:]` only if live-confirmed); D-16 (whether project `.agents/skills` is trust-gated settled live); D-17 (Codex's description truncation recorded as advisory only).

D. Codex nudge (CODEX-05): D-18 (`--pretool-nudge` widens to Codex; skip with a Note if `[features] hooks = false`/`codex_hooks = false`); D-19 (Codex skips new/changed hooks until trusted in `/hooks`; trust hash covers normalized definition; ExecPath rendered into the guard, never the command; never tell users to use `--dangerously-bypass-hook-trust`); D-20 (command quoted from day one: local `"$(git rev-parse --show-toplevel)/.codex/hooks/codegraph-pretooluse.sh"`, global a quoted absolute path; one group, `matcher: "^Bash$"`, one handler, explicit short timeout, both scopes); D-21 (adapter is a Codex envelope on the same hidden subcommand, e.g. `codegraph hook pretooluse --harness codex`; maps `tool_name` `Bash` and defensively `exec_command`/`shell`; accepts `tool_input.command` as string or argv array, doubt means silent; keys on stdin `session_id` + `agent_id`, else `main`; reuses only shell rows of D-15 corpora); D-22 (no `CLAUDE_PROJECT_DIR` — local guard derives repo root from its own path two dirs up from `.codex/hooks/`, no process spawn; global guard checks `$PWD`; Go core re-checks stdin `cwd`; no `if` pre-filter, every Bash call runs the guard); D-23 (stickiness evidence is the exact-identity group in `hooks.json`, appended LAST).

E. FIX-03 and AGENT-14: D-24 (fix at cause: `checkboxDelegate.Render`'s trailing newline doubles per-row cost against `Height()=1`; drop it; fix `daemonDelegate` identically; this is a code-reading hypothesis until a failing test proves it); D-25 (guard is a model-level `lipgloss.Height(View()) <= 30` test at 100×30 with all 8 target names, plus the tmux TTY-05 assertion re-anchored on the footer text `space: toggle`, both shown RED first, re-run after CODEX-02); D-26 (picker keeps all 8 rows; CODEX-02 changes only Codex's local pre-check; ROADMAP's "scope flip changes target count" premise was corrected 2026-09-19); D-27 (capability table in new `docs/AGENT-CAPABILITIES.md`, linked from README's agent section; Go drift test mirroring `TestMatrix_DocMirrorsDescriptor`, code-derived columns vs `Capabilities()` for 8×2, planted mutation proves RED; hand-kept verification column with `verified <date> (<evidence file>)` or `[ASSUMED] (<doc URL>, fetched <date>)`; Cursor/Gemini CLI/Kiro stay `[ASSUMED]`); D-28 (nudge column derived from `Hooks` in the drift test, no new `Capabilities` field, no new `--print-config-style` field); D-29 (MCP `instructions` skill sentence becomes harness-neutral, true for the 7 skill-receiving targets, whole const ≤600 bytes with skill sentence inside first 512 bytes; WIRE-03 guard updated; wire transcripts re-frozen in one reviewed diff); D-30 (the "4 of 8" comments updated as comments only at `instructions.go:13-28`, `shared.go:743`, `codex.go:13-19`; installed marker-block text unchanged).

### Claude's Discretion
The Codex guard filename, the adapter flag spelling, the timeout value, and the exact Note wording within D-10/D-12/D-18/D-19. How the drift test parses `docs/AGENT-CAPABILITIES.md` (a table-row format of the planner's choosing). Test fixture shapes for D-07.

### Deferred Ideas (OUT OF SCOPE)
Exec-form/quoting for Claude's existing SessionStart/PreToolUse commands (AR-06-08). A Codex SessionStart nudge. Codex skill metadata `agents/openai.yaml` and Codex plugin packaging (v2). `codegraph upgrade` refreshing non-Claude skill packages/the Codex guard, unless D-23's evidence needs it. Uninstall leaving an empty parent skills directory. Kiro reading the codegraph block twice through opencode/Codex-written `AGENTS.md`. Live verification for Cursor, Gemini CLI, Kiro. Filtering picker rows per location. A patch release carrying the TOML fix ahead of the milestone.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CODEX-01 | Live-verify project config/skill-path/hooks before any `codex.go` change; correct the stale comment | See "Live Codex CLI probes" and "Codex mechanics reference" below — model-free probes already run this session confirm `hooks=stable,true`, `mcp list --json` scope-less shape, and the skill/AGENTS.md chain via `debug prompt-input` |
| CODEX-02 | `SupportsLocation(LocationLocal)`; local install writes `.codex/config.toml` via the fixed splice; trust Note; picker/`--target auto` reflect the flip | See "TOML splice fix" and "Scope flip" sections |
| CODEX-03 | Skill package to verified path(s) at both scopes, idempotent, byte-invariant | See "Skill path" section — `sharedSkillDirs`/`installDeclaredSkill` already generic; live-confirmed `.agents/skills` read by Codex this session |
| CODEX-04 | Local install writes marker block into repo-root `AGENTS.md`; uninstall byte-identical restore, sharing-aware | See "Shared repo-root AGENTS.md (D-11)" section |
| CODEX-05 | Codex PreToolUse nudge, opt-in only, additionalContext-only, trust-aware | See "Codex hooks.json schema" and "Nudge adapter for Codex" sections — includes a flagged gap: PreToolUse's documented stdin fields carry no `agent_id` |
| CODEX-06 | Fresh Codex session reaches for codegraph unprompted; `mcp list` shows both scopes | Live-verification protocol only; no code changes — see "Live verification protocol notes" |
| FIX-03 | Picker footer renders correctly in 100×30 with 8 targets | See "FIX-03 root cause, confirmed with line-level evidence" — bug fully reproduced from `bubbles/v2@v2.1.1` source, not just hypothesized |
| AGENT-14 | Published, drift-tested per-harness capability table | See "AGENT-14 doc-drift test" section, modeled on `TestMatrix_DocMirrorsDescriptor` |

</phase_requirements>

## Summary

This phase is almost entirely a "read the existing generic machinery, then extend one target's literal" exercise — Phase 5's capability-table refactor already made `Detect`, `DescribePaths`, `SupportsLocation`, and `--print-config-style` pure derivations of `Capabilities()`. Flipping Codex from global-only to `{global, local}` is mechanically the same kind of change opencode already embodies (dual-scope MCP config + dual-scope instructions + shared skill dir), so `opencode.go` is the load-bearing reference implementation for `codex.go`'s new shape, not a from-scratch design.

Two defects block everything else and must land first per D-01: the TOML splice bug (`findTOMLTableRange` in `internal/agents/toml.go:77-103` ends a table at the next **column-0** `[` line — verified this session against the maintainer's real, 2-space-indented `[mcp_servers.codegraph]` block — so any indented user table between codegraph's block and the true next header would be silently deleted), and the picker footer overflow (root-caused this session, not hypothesized: `checkboxDelegate.Render` in `internal/cli/tui/agentpicker.go:64` writes its own trailing `\n`, and `bubbles/v2@v2.1.1`'s `populatedView` in `list.go:1220-1224` *also* inserts a `\n`-separator between items, so N items cost `2N-1` lines against a `Height()=1` budget of `N` — an exact, provable N-1 overflow, confirmed against the recorded 35-vs-28 figure at 8 targets).

The Codex-specific mechanics (hooks.json schema, trust model, skill-path precedence, `tool_input.command` shape) are now verified from **primary sources** this session — the official `openai/codex` GitHub repo via Context7 (Rust source: `codex-rs/hooks/`, `codex-rs/config/src/hooks_tests.rs`, `codex-rs/features/`) and `learn.chatgpt.com/docs/hooks` — plus six live, model-free `codex` CLI invocations against codex-cli 0.155.0 in an isolated scratch `HOME`/`CODEX_HOME`. One material finding narrows a locked decision's scope: Codex's documented **PreToolUse** stdin schema carries no `agent_id` field at all (that field is documented only for `SubagentStart`/`SubagentStop`); D-21's "keys on stdin session_id + agent_id" may not be satisfiable for PreToolUse specifically on today's Codex — flagged as an Open Question requiring the live verification pass, not silently downgraded.

**Primary recommendation:** Fix TOML splice and FIX-03 first exactly as D-01/D-24/D-25 specify (both are provable, mechanical bugs with line-level root causes already found — no further design work needed, only RED-first test-and-fix); model the Codex scope flip byte-for-byte on `opencode.go`'s dual-scope shape; build the Codex hooks.json fragment and guard script as new sibling files to the existing `.claude/hooks/` pair, embedded via a new root-level `codexassets.go` file (same `//go:embed` sibling-of-root-file rule `claudeassets.go` already documents); and treat the missing PreToolUse `agent_id` field as a live-verification checkpoint, not an assumption to code around silently.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| TOML table splice/strip (`.codex/config.toml`) | CLI / installer (Go, `internal/agents`) | — | Local file mutation, no network/server tier involved |
| Codex scope/config/skill/instructions resolution | CLI / installer (Go, `internal/agents`) | — | Pure capability-table derivation, same tier as every other target |
| PreToolUse hook dispatch and trust | External process (Codex CLI, Rust) | CLI / installer (Go) writes the config Codex reads | codegraph never runs inside Codex's process; it only writes `hooks.json` and a guard script Codex's own hook engine invokes |
| Nudge decision logic (`nudge.Qualifies`, cooldown) | CLI / installer (Go, `internal/nudge`) | — | Harness-neutral core, already built Phase 6; Codex reuses it unchanged behind a new envelope adapter |
| Agent picker TUI rendering | CLI / installer (Go, `internal/cli/tui`, bubbles/v2) | — | Pure terminal rendering; no persistence, no network |
| Capability-table doc drift check | CLI / installer (Go test) + docs (Markdown) | — | Same tier pairing as the existing `LANGUAGE-CAPABILITY-MATRIX.md` / `matrix_test.go` precedent |

## Standard Stack

### Core
No new third-party libraries are added by this phase — every capability (TOML splice, JSONC patch, hooks.json write, embedded template rendering) is built on dependencies already in `go.mod` [VERIFIED: go.mod:1-40, read this session]:

| Library | Version | Purpose | Why Standard (in this repo) |
|---------|---------|---------|------------------------------|
| `charm.land/bubbles/v2` | v2.1.1 [VERIFIED: go.mod] | Picker list widget (FIX-03 root cause lives here) | Already the project's TUI dependency; the fix is a one-line change to this repo's own delegate, not to the library |
| `charm.land/bubbletea/v2` | v2.0.8 [VERIFIED: go.mod] | `tea.Model`/`tea.View` for the picker | Same |
| `charm.land/lipgloss/v2` | v2.0.5 [VERIFIED: go.mod] | `lipgloss.Height(string) int` for D-25's model-level test | `Height` confirmed present at `/Users/sean/go/pkg/mod/charm.land/lipgloss/v2@v2.0.5/size.go:29` [VERIFIED: read this session] |
| `github.com/tailscale/hujson` | v0.0.0-20260302212456 [VERIFIED: go.mod] | opencode's JSONC patch (reference pattern only — Codex uses TOML, not this) | Precedent for "surgical patch preserving comments," not directly reused by Codex |
| stdlib `embed`, `encoding/json`, `crypto/sha256` | Go 1.26.6 [VERIFIED: go.mod:3] | Embedding new Codex templates, hooks.json shape, manifest hashing | Matches every existing target's implementation |

**No TOML library exists in `go.mod` and none should be added.** `internal/agents/toml.go`'s own doc comment states this is deliberate (D-05a: "a full TOML parser/serializer dependency is unjustified for editing exactly one dotted-key table") [VERIFIED: internal/agents/toml.go:1-12, read this session]. **A parse-validation step without a new dependency is feasible**: the same hand-rolled tokenizer that must be extended to recognize headers at any indentation and skip multi-line strings/arrays (D-07) can double as a post-write sanity check — e.g. count `^\s*\[` header lines before and after the splice and assert the count only changed by the expected delta, or re-run `findTOMLTableRange` after the write and assert it still finds exactly one occurrence of codegraph's table. This is validation-by-construction (reusing the fixed scanner), not a second parser.

### Package Legitimacy Audit

**Not applicable — no external packages are installed by this phase.** Every mechanism (TOML splice, hooks.json write, embedded shell script) is hand-rolled Go using stdlib and libraries already vetted in prior phases. Skip the Package Legitimacy Gate.

## Architecture Patterns

### System Architecture Diagram

```
                    codegraph install --target codex --location local
                                    │
                                    ▼
                    internal/cli/install.go (RunE)
                    [D-13 fix: check --target Changed() BEFORE --yes]
                                    │
                                    ▼
                    internal/agents.codexTarget.Install(loc, opts)
                                    │
                    ┌───────────────┼────────────────┬─────────────────────┐
                    ▼               ▼                ▼                     ▼
        MCPConfig(loc)      Instructions(loc)   installDeclaredSkill  (opt-in) hooks
     .codex/config.toml   repo-root AGENTS.md    .agents/skills/       .codex/hooks.json
     (global: ~/.codex/…) (global: ~/.codex/…)    codegraph/           + guard script
            │                     │                      │                    │
            ▼                     ▼                      ▼                    ▼
     spliceTOMLTable()    upsertInstructionsEntry() installSkillPackage  writeHookEntry()
     [D-07: any-indent    [D-11: shared with          WithFallback()      [reuses Phase 6
      header scan,         opencode; "kept" if       [already generic,    machinery verbatim,
      CRLF-safe,           opencode still            reused unchanged]    D-23: group appended
      subtable-aware]      configured there]                              LAST]
                                                                                │
                                                                                ▼
                                                              Codex CLI's own hook engine
                                                              (external process, Rust) —
                                                              trust-gates via `/hooks`,
                                                              runs codegraph-pretooluse.sh
                                                              on every matching PreToolUse
                                                                                │
                                                                                ▼
                                                              guard script (D-22: derives
                                                              repo root from its own path,
                                                              no CLAUDE_PROJECT_DIR) execs
                                                              `codegraph hook pretooluse
                                                               --harness codex` <<< stdin
                                                                                │
                                                                                ▼
                                                              internal/cli/hook_pretooluse.go
                                                              Codex envelope branch ->
                                                              internal/nudge.Qualifies/Gate
                                                              (harness-neutral, UNCHANGED)
```

### Recommended Project Structure (new/changed files only)
```
.codex/
├── hooks/
│   ├── hooks.json               # NEW — embedded PreToolUse-only fragment (mirrors .claude/hooks/hooks.json)
│   └── codegraph-pretooluse.sh  # NEW — embedded guard template (mirrors .claude/hooks/pretooluse-nudge.sh)
codexassets.go                   # NEW — repo-root embed file, sibling of claudeassets.go (embed sibling-of-root rule)
internal/agents/
├── codex.go                     # MODIFIED — scope flip, D-09 capability literal
├── toml.go                      # MODIFIED — D-07 header scanner fix
├── toml_test.go                 # MODIFIED — indentation/CRLF/inline/subtable/multi-line-array fixtures
├── codex_test.go                # MODIFIED — local-unsupported tests become local-supported tests
├── codex_pretooluse.go          # NEW — Codex hook-block/guard-render helpers (mirrors claude_pretooluse.go)
├── ownership_test.go            # MODIFIED — ownershipWantSkillDir gains a Codex case
├── shared.go                    # MODIFIED — D-11 shared-instructions "still configured elsewhere" check
docs/
└── AGENT-CAPABILITIES.md        # NEW — D-27 published capability table
internal/agents/capability_doc_test.go  # NEW — D-27 drift test (naming: planner's choice)
internal/cli/
├── install.go                   # MODIFIED — D-13 ordering fix
├── uninstall.go                 # MODIFIED — D-13 ordering fix
├── hook_pretooluse.go           # MODIFIED — Codex envelope branch (D-21)
└── tui/
    └── agentpicker.go           # MODIFIED — D-24 drop trailing \n; daemonpicker.go identically
test/tmux/
└── install_cancel_test.go       # MODIFIED — D-25 footer-text re-anchor (after CODEX-02 lands)
```

### Pattern 1: Capability-table-derived target (the shape every target already follows)
**What:** A target's `SupportsLocation`, `Detect`, `DescribePaths` are one-line derivations of `Capabilities()`; only `Install`/`Uninstall` contain target-specific logic, and even those increasingly delegate to shared helpers (`installDeclaredSkill`, `upsertInstructionsEntry`, `writeHookEntry`).
**When to use:** Every change to `codex.go` in this phase.
**Example (opencode.go — the closest existing dual-scope, dual-instructions, shared-skill-dir target):**
```go
// Source: internal/agents/opencode.go:45-54 (read this session)
func (opencodeTarget) Capabilities() Capabilities {
	return Capabilities{
		Scopes:       []Location{LocationGlobal, LocationLocal},
		ConfigFormat: ConfigFormatJSONC,
		Hooks:        HooksNone,
		MCPConfig:    opencodeConfigPath,
		Instructions: opencodeInstructionsPath,
		SkillDirs:    sharedSkillDirs,
	}
}
```
The Codex equivalent (D-09) is structurally identical, substituting `ConfigFormatTOML`, `Hooks: HooksCodexJSON`, and Codex's own path functions:
```go
// Illustrative target shape for codex.go — verbatim values not yet committed to code
func (codexTarget) Capabilities() Capabilities {
	return Capabilities{
		Scopes:       []Location{LocationGlobal, LocationLocal},
		ConfigFormat: ConfigFormatTOML,
		Hooks:        HooksCodexJSON,
		MCPConfig:    codexConfigPath,      // must gain a Location branch, see Pattern 2
		Instructions: codexInstructionsPath, // local: "AGENTS.md" (repo root); global: unchanged ~/.codex/AGENTS.md
		SkillDirs:    sharedSkillDirs,
	}
}
```

### Pattern 2: Dual-scope path function (opencode's exact shape to copy)
**What:** A `PathFunc` that branches on `loc == LocationLocal` for a bare relative path vs. a `$HOME`-joined absolute path.
**Example:**
```go
// Source: internal/agents/opencode.go:72-95 (read this session) — codexConfigPath needs the
// identical local/global branch; codex.go:47-53 today is global-only (globalOnlyPath wrapper).
func opencodeConfigPath(loc Location) (string, error) {
	var dir string
	if loc == LocationLocal {
		dir = "."
	} else {
		cfgDir, err := resolveOpencodeConfigDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(cfgDir, "opencode")
	}
	// ... resolve jsonc/json candidate inside dir
}
```
`codexConfigPath` must drop its current `globalOnlyPath(func() (string, error))` wrapper (`codex.go:42`) and become a real `PathFunc(Location) (string, error)` returning `.codex/config.toml` (relative) for local and `filepath.Join(home, ".codex", "config.toml")` for global — mirroring the shape above exactly, not opencode's candidate-file-exists logic (Codex's config file is always `.codex/config.toml`, no `.json` fallback).

### Pattern 3: Shared instructions file with cross-target "still configured elsewhere" check (D-11 — genuinely new code)
**What:** D-11 requires a helper, callable from both `codexTarget.Uninstall` and `opencodeTarget.Uninstall`, that checks whether any OTHER registered target still declares the same instructions path at the same location and reports `Detect(loc).AlreadyConfigured` before removing the shared marker block.
**No such helper exists yet** — `removeMarkedSection` (shared.go:695-739) unconditionally removes the span. This is the one genuinely new piece of cross-target logic in this phase (everything else in D-07..D-30 extends an existing single-target pattern).
**Illustrative shape** (exact naming is the planner's choice):
```go
// New helper in shared.go — illustrative, not yet written
func instructionsStillNeededElsewhere(path string, loc Location, self TargetID) bool {
	for _, t := range AllTargets() {
		if t.ID() == self {
			continue
		}
		caps := t.Capabilities()
		if !caps.Supports(loc) {
			continue
		}
		p, err := caps.InstructionsPath(loc)
		if err != nil || p == "" || p != path {
			continue
		}
		if t.Detect(loc).AlreadyConfigured {
			return true
		}
	}
	return false
}
```
Both `codexTarget.Uninstall` and `opencodeTarget.Uninstall` (the two local-scope targets sharing repo-root `AGENTS.md`, per `instructions.go:13-19`'s own comment "4 of 8" list of Claude/Codex/opencode/Gemini) call this before `removeMarkedSection`, reporting `ActionKept` when it returns true. **No cycle risk**: `AllTargets()` lives in `registry.go`, same package `agents`, already called by `codex.go`/`opencode.go`'s siblings via `installDeclaredSkill`'s own `declaredSkillFallback` pattern (capabilities.go:214-227) which iterates a comparison target the same way.

### Pattern 4: Embedded, path-rendered shell guard (Claude's pretooluse-nudge.sh, to be mirrored for Codex)
**What:** A template embedded via `//go:embed`, with exactly one occurrence of a placeholder token replaced at install time with the POSIX-single-quoted absolute binary path — never the binary path baked into the *registered command string* (D-19).
**Example — the exact rendering mechanism to reuse unchanged for Codex:**
```go
// Source: internal/agents/claude_pretooluse.go:131-151 (read this session)
const preToolGuardExecPathToken = "'@codegraph-exec-path@'"

func renderPreToolGuard(execPath string) (string, error) {
	if execPath == "" { return "", errors.New("render PreToolUse guard: empty binary path") }
	if !filepath.IsAbs(execPath) { return "", fmt.Errorf("... not absolute") }
	data, err := claudeassets.PreToolUseGuardTemplate()
	tmpl := string(data)
	if n := strings.Count(tmpl, preToolGuardExecPathToken); n != 1 {
		return "", fmt.Errorf("template carries the binary-path token %d times, want 1", n)
	}
	return strings.Replace(tmpl, preToolGuardExecPathToken, shellSingleQuote(execPath), 1), nil
}
```
The Codex guard template needs its own token (or reuse the same literal constant if factored into a shared function taking the template bytes and token as parameters — recommended, since the rendering logic is byte-identical, only the embedded template source differs).

**Claude's current guard, verbatim (`.claude/hooks/pretooluse-nudge.sh`, read this session) — the pattern the Codex guard must adapt for D-22's no-`CLAUDE_PROJECT_DIR` constraint:**
```sh
[ -d "${CLAUDE_PROJECT_DIR:-.}/.codegraph" ] || exit 0
codegraph_bin='@codegraph-exec-path@'
case $codegraph_bin in
@*@) codegraph_bin=$(command -v codegraph 2>/dev/null) || exit 0 ;;
esac
if [ ! -f "$codegraph_bin" ] || [ ! -x "$codegraph_bin" ]; then
  exit 0
fi
"$codegraph_bin" hook pretooluse
exit 0
```
Illustrative Codex local-scope guard (D-20, D-22 — the script itself, installed at `.codex/hooks/codegraph-pretooluse.sh`, derives repo root two directories up from its own path with no process spawn, since it always lives at `<repo>/.codex/hooks/`):
```sh
#!/bin/sh
script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(dirname -- "$(dirname -- "$script_dir")")
[ -d "$repo_root/.codegraph" ] || exit 0
codegraph_bin='@codegraph-exec-path@'
case $codegraph_bin in
@*@) codegraph_bin=$(command -v codegraph 2>/dev/null) || exit 0 ;;
esac
[ -f "$codegraph_bin" ] && [ -x "$codegraph_bin" ] || exit 0
"$codegraph_bin" hook pretooluse --harness codex
exit 0
```
The global-scope guard (installed at `$CODEX_HOME/hooks/codegraph-pretooluse.sh`) instead checks `[ -d "${PWD}/.codegraph" ]` per D-22 ("the global guard checks `$PWD`").

### Pattern 5: hooks.json fragment shape (verified against Codex's own schema, not guessed)
**What:** Claude's embedded `.claude/hooks/hooks.json` fragment (read this session, `.claude/hooks/hooks.json:1-89`) and Codex's documented schema are structurally the same top-level shape (`{"hooks": {"<Event>": [{"matcher": ..., "hooks": [{"type": "command", "command": ..., "timeout": N}]}]}}`), so the same `claudeFragmentEventBlocks`-style JSON-rewrite helper (`internal/agents/claude.go:288-...`, decode → find `hooks.PreToolUse` → rewrite each block's `command` field) can be duplicated (not shared — the two fragments differ enough in shape that forcing one function over both risks the D-00 rule against inventing shared structure for two things that happen to look similar today but are independently versioned) for a new `.codex/hooks/hooks.json`.
**Verified Codex hooks.json schema** [VERIFIED: Context7 `/openai/codex`, source `codex-rs/config/src/hooks_tests.rs`, fetched 2026-09-19]:
```json
{
  "description": "Optional description",
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "^Bash$",
        "hooks": [
          {
            "type": "command",
            "command": "python3 /tmp/pre.py",
            "timeout": 10,
            "statusMessage": "checking",
            "additionalContextLimit": 4096
          }
        ]
      }
    ]
  }
}
```
D-20's Codex fragment is exactly one `PreToolUse` block with `matcher: "^Bash$"`, one handler, `type: "command"`, `command` = the placeholder, and a short `timeout` in **seconds** (confirmed: the Rust `HookMetadata` struct field is `timeout_sec: u64` [VERIFIED: Context7 `/openai/codex`, source `codex-rs/app-server-protocol/src/protocol/v2/plugin.rs`]) — do not port Claude's `5` assuming it is milliseconds or any other unit; it is the same unit, seconds, in both hook systems, but confirm this is stated explicitly wherever the timeout value is chosen.

### Anti-Patterns to Avoid
- **Adding a `.codex/skills` write "just in case":** D-14 is explicit that Codex writes ONLY the shared `.agents/skills/codegraph` path. Codex does not merge same-name skills across roots [CITED: `learn.chatgpt.com/docs/customization/overview`, "Same-name skills are NOT merged" per CONTEXT.md's already-verified finding], so writing a second copy at `.codex/skills/codegraph` would make the skill appear twice in Codex's own listing.
- **Assuming the trust-hash covers only the `command` string:** the Rust struct carries `current_hash: String` on the whole `HookMetadata` (matcher, timeout, handler all included) [VERIFIED: Context7 `/openai/codex`, `plugin.rs`], consistent with D-19's "the trust hash covers the normalized definition" — do not build a guard/test that assumes only the command text is hashed.
- **Treating `codex mcp list --json`'s absence of a scope column as "codegraph can't tell which scope an entry came from":** confirmed this session (`codex mcp list --json` returns `[]` on an empty scratch `CODEX_HOME`, and the Rust source shows the JSON schema has no scope field at all: `name`, `enabled`, `disabled_reason`, `transport`, `startup_timeout_sec`, `tool_timeout_sec`, `auth_status` [VERIFIED: Context7 `/openai/codex`, `codex-rs/cli/src/mcp_cmd.rs`, fetched 2026-09-19]) — CODEX-06's "both scopes" pass bar (L1) must be evidenced by running `codex mcp list --json` from two different cwds/HOME configurations, or by reading the two config files directly, never by a `scope` field in the JSON that does not exist.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Detecting which target still needs a shared `AGENTS.md` block | A new registry/index of "who wrote this file" | `AllTargets()` iteration + `Capabilities().InstructionsPath(loc)` + `Detect(loc).AlreadyConfigured` (Pattern 3 above) | The registry (`registry.go`) and capability table already carry every fact needed; a parallel index would be a second source of truth that can drift |
| Parsing/validating the spliced TOML | A TOML parser dependency | The same header-scanning tokenizer `findTOMLTableRange` already implements, extended for D-07 and reused as its own sanity check | D-05a's project-constraint reasoning (minimal deps) already rejected a TOML library for editing; validation should reuse the fixed scanner, not add a second dependency to check the first hand-rolled implementation |
| Codex hook JSON rewriting | A generic JSON-patch library for hooks.json | The same decode-into-`map[string]any`-then-rebuild pattern `writeHookEntry`/`removeHookEntry` (shared.go:220-479) already implement and that Codex's hooks.json shares the top-level shape with | Codex's hooks.json is a plain JSON file (not JSONC like opencode's config), so the existing `readJSONFileStrict`/`writeJSONFile` primitives apply directly with zero new tooling |

**Key insight:** every "new" mechanism this phase needs (dual-scope path resolution, shared-file ownership counting, JSON hook-block ownership) already exists somewhere else in this package for a different target. The work is almost entirely "generalize an existing single-instance pattern to a second caller," which is exactly the shape `installDeclaredSkill`/`uninstallDeclaredSkill` (capabilities.go:229-275) already did for the skill-directory write path in Phase 5.

## Common Pitfalls

### Pitfall 1: The TOML splice bug is reproducible today against the real config
**What goes wrong:** `findTOMLTableRange` (toml.go:96-98) ends a table scan at the next line with `strings.HasPrefix(lines[i], "[")` — literally column 0 only. The maintainer's real `~/.codex/config.toml` has `[mcp_servers.codegraph]` at 2-space indentation with the next column-0 header 70 lines later (per CONTEXT.md's already-recorded finding, confirmed structurally consistent with this scanner's exact logic).
**Why it happens:** The scanner was written assuming every top-level table header starts at column 0, which is true for a file codegraph wrote from scratch but false for a hand-edited file with any indentation style (many TOML editors/linters indent nested-table bodies, and some indent headers themselves for visual grouping).
**How to avoid:** D-07's exact fix: any-indentation header recognition (`strings.TrimSpace(line)[0] == '['`), while tracking multi-line basic/literal string state (`"""..."""`, `'''...'''`) and multi-line array bracket-depth state so a `[` inside either is never mistaken for a header. This is a real per-character/per-line state machine, not a regex tweak.
**Warning signs:** Any fixture where the codegraph table is NOT the last table in the file and has ANY leading whitespace — the existing `toml_test.go` fixtures (read this session) are all column-0, unindented, so they do not currently exercise this bug at all. Confirmed: `toml_test.go`'s `tomlUnrelatedTable` const is column-0. **A RED-first fixture reproducing the real bug must use indentation.**

### Pitfall 2: `install --yes` bug already has a confirmed root cause and exact fix
**What goes wrong:** In both `internal/cli/install.go:111-119` and `internal/cli/uninstall.go:50-56`, the `switch` statement's `case yes:` branch is checked BEFORE `case cmd.Flags().Changed("target"):`, so `--target claude,cursor --yes` silently resolves to `auto`/`all` instead of the explicit list.
**Why it happens:** The comment on `case yes:` documents the INTENT ("must short-circuit before the TTY branch") but the switch's case ORDER accidentally also shadows the explicit-target case, which was never the intent (confirmed: `.planning/todos/pending/2026-09-18-install-yes-discards-explicit-target.md`, read this session, already root-causes this and names the exact fix).
**How to avoid:** Swap the two `case` clauses' order in both files — `case cmd.Flags().Changed("target"):` first, `case yes:` second. This is a 2-line diff per file with a RED-first test (`--target codex --yes` on a fresh scratch HOME must configure exactly Codex, not the auto-detected set).
**Warning signs:** Any existing test asserting `--yes`'s resolution to `auto`/`all` without ALSO covering `--target X --yes` — check for this gap; if none exists, the RED fixture is new.

### Pitfall 3: FIX-03's overflow formula is exact, not approximate
**What goes wrong:** `bubbles/v2@v2.1.1`'s `populatedView` (`list.go:1220-1224`, read this session) writes `strings.Repeat("\n", m.delegate.Spacing()+1)` between every pair of items (`Spacing()` is 0, so this is exactly one `\n`) — AND `checkboxDelegate.Render` (`agentpicker.go:64`) independently writes its own trailing `\n` on every row via `fmt.Fprintf(w, "%s%s %s\n", ...)`. This means every rendered item (except conceptually the very last) contributes 2 newline characters where the list's own height accounting (`Height()=1`, `Spacing()=0`) expects 1.
**Why it happens:** `checkboxDelegate.Render`'s doc comment does not mention that `bubbles/v2/list` already adds its own inter-item separator — the delegate was written as if `Render` alone controls every byte of vertical space per item, which was true for `Height()=1, Spacing()=0` in a naive reading but is contradicted by `populatedView`'s own separator logic.
**How to avoid:** D-24's fix — drop the delegate's own trailing `\n` (end the `Fprintf` format string without `\n`; `populatedView`'s separator supplies the line break between items, and the final item's line is still terminated by the enclosing `lipgloss.NewStyle().Height(availHeight).Render(...)` call in `list.go:1072`). Apply identically to `daemonDelegate.Render` (`daemonpicker.go:59`), which has the exact same trailing-`\n` defect.
**Warning signs:** The overflow is exactly `N-1` lines for `N` items — 8 targets → 7 extra blank lines → 28 (intended) vs 35 (actual), matching the CONTEXT.md-recorded figures exactly. Any fix that doesn't produce this exact before/after delta on the model-level `lipgloss.Height` test has not addressed the real cause.

### Pitfall 4: Codex's PreToolUse stdin schema may not carry a subagent-distinguishing field
**What goes wrong:** D-21 specifies the adapter "keys on stdin `session_id` (the parent's for subagents) plus `agent_id`, else `main`" — mirroring Claude's own PreToolUse shape. But the documented Codex `PreToolUse` input fields [VERIFIED: Context7 `/llmstxt/learn_chatgpt_llms-full_txt`, `learn.chatgpt.com/docs/hooks`, fetched 2026-09-19] are only: `turn_id`, `tool_name`, `tool_use_id`, `tool_input` (plus the Common input fields: `session_id`, `transcript_path`, `cwd`, `hook_event_name`, `model`). **`agent_id` is documented only for `SubagentStart`/`SubagentStop`**, not `PreToolUse`. The Common input fields section explicitly notes "Subagent hooks use the parent session id" — implying a PreToolUse call fired during a subagent's turn carries the SAME `session_id` as the primary agent's turn, with no separate field to tell them apart.
**Why it happens:** Codex's hook system was documented incrementally across event types; PreToolUse's field table may simply be incomplete, or Codex genuinely does not expose per-subagent identity to PreToolUse hooks (unlike Claude Code, where `agent_id` is present on PreToolUse).
**How to avoid:** Do not silently code the adapter as if `agent_id` will always be present and empty-string-fallback to `"main"` without flagging the consequence: if Codex truly never sends `agent_id` on PreToolUse, EVERY subagent nudge call collapses onto the SAME cooldown key as the primary agent's calls (since `nudge.SessionKey("", agentID="")` → key = `sessionID + "\x00main"` for all of them), which is a materially different cooldown granularity than Claude gets. This must be confirmed or refuted live (CODEX-05's L6 check) — a probe should specifically drive a PreToolUse-triggering Bash call from within a Codex subagent and inspect the raw stdin JSON for an `agent_id` key.
**Warning signs:** If the live check finds no `agent_id` key on a real PreToolUse stdin payload even during a subagent turn, D-21's "separately per subagent" nudge behavior for Codex degrades to "per session only" — this is a decision the maintainer should confirm is acceptable (or trigger a scope reduction on that one sub-clause of D-21) rather than something the code should silently paper over.

### Pitfall 5: `.codex/skills` may be a real, additionally-documented path, not just a future maybe
**What goes wrong:** D-15 treats `.codex/skills` as "listed as read-only `SkillDirs[1:]` entries only if the live check shows Codex reading them" — implicitly treating this as unconfirmed. A Codex changelog entry [CITED: `learn.chatgpt.com/docs/changelog`, "2025-12-19 > Agent skills in Codex", fetched 2026-09-19 via Context7] states explicitly: "You can install skills for just yourself in `~/.codex/skills`, or for everyone on a project by checking them into `.codex/skills` in the repository" — this is an affirmative documentation claim, not silence, and it names BOTH `~/.codex/skills` (global) AND `.codex/skills` (project) as supported skill install locations, separate from `.agents/skills`.
**Why it happens:** CONTEXT.md's "Not confirmed" list was compiled before this specific changelog entry was located this session.
**How to avoid:** Elevate this from "maybe" to a specific live-verification target: place a uniquely-named skill in `.codex/skills/<name>/SKILL.md` in the scratch repo and confirm via `codex debug prompt-input` whether it appears in the skill catalog alongside `.agents/skills`. This session's own live probe (below) shows `debug prompt-input` DOES enumerate skill roots by label (`r0`, `r1`, ...) — a `.codex/skills` root would show up as an additional `rN` entry if Codex reads it, making this a cheap, decisive, already-available check.
**Warning signs:** If `.codex/skills` IS read, D-15's SkillDirs entry list needs a second read-only path; if it is read only when the project is trusted (unlike `.agents/skills`, which this session's live probe found readable even without explicit trust registration — see below), that asymmetry needs its own Note wording.

## Code Examples

### TOML header-scanner extension shape (D-07)
```go
// Source: internal/agents/toml.go:77-103 (read this session) — the function to extend.
// findTOMLTableRange today (THE BUG):
func findTOMLTableRange(content, tableName string) (start, end int, found bool) {
	header := "[" + tableName + "]"
	lines := strings.Split(content, "\n")
	// ...
	for i := headerLine + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "[") {  // <-- column-0 only; the bug
			return start, scanOffset, true
		}
		scanOffset += len(lines[i]) + 1
	}
	return start, len(content), true
}
```
The D-07 fix must: (1) match `strings.TrimSpace(lines[i])` starting with `[` rather than `lines[i]` itself, to catch indented headers; (2) track whether the scanner is inside a `"""`/`'''` multi-line string or an unbalanced `[`/`]` multi-line array, and skip header-detection entirely while inside either; (3) treat `[mcp_servers.codegraph.foo]` (a dotted subtable of codegraph's own table) as still inside codegraph's range, not a new top-level header ending the scan — this requires comparing the candidate header's dotted-prefix against `tableName+"."` in addition to the exact-match check `spliceTOMLTable`/`stripTOMLTable` already do; (4) detect and refuse (return an error, not proceed) when scanning finds `codegraph = {` inline under `[mcp_servers]` or a bare `mcp_servers.codegraph.foo = ...` dotted-key line outside any header — per D-07's explicit "refused with an error, never answered with a duplicate key" requirement. This is a genuine state-machine rewrite of `findTOMLTableRange`, not a one-line patch; budget planning time accordingly.

### FIX-03 model-level regression test shape (D-25)
```go
// Illustrative — exact test name/location is the planner's choice, in internal/cli/tui
func TestAgentPickerFootprintFitsDefaultPane(t *testing.T) {
	all := agents.AllTargets() // must be 8 (D-26)
	m := newAgentPickerModel(all, map[agents.TargetID]agents.DetectionResult{})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m3 := m2.(agentPickerModel)
	view := m3.View()
	h := lipgloss.Height(view.Content) // tea.View.Content is the plain string field
	if h > 30 {
		t.Fatalf("picker view is %d lines tall in a 100x30 pane, want <= 30", h)
	}
	for _, target := range all {
		if !strings.Contains(view.Content, target.DisplayName()) {
			t.Errorf("picker view does not contain target %q", target.DisplayName())
		}
	}
}
```
`tea.View.Content` is confirmed a plain `string` field [VERIFIED: `/Users/sean/go/pkg/mod/charm.land/bubbletea/v2@v2.0.8/tea.go:84-96`, read this session]. `lipgloss.Height(str string) int` is confirmed present at the pinned `v2.0.5` [VERIFIED: `/Users/sean/go/pkg/mod/charm.land/lipgloss/v2@v2.0.5/size.go:29`, read this session].

### `install --yes`/`--target` ordering fix (D-13)
```go
// Source: internal/cli/install.go:110-126 (read this session) — swap these two cases' order:
var targets []agents.AgentTarget
switch {
case cmd.Flags().Changed("target"):                // MOVED FIRST
	targets, err = agents.ResolveTargetFlag(target, loc)
case yes:                                           // MOVED SECOND
	targets, err = agents.ResolveTargetFlag("auto", loc)
case interactiveAllowed(cmd):
	targets, err = runAgentPicker(cmd, loc)
default:
	targets, err = agents.ResolveTargetFlag("auto", loc)
}
```
Identical swap in `internal/cli/uninstall.go:50-56` (targets `"all"` instead of `"auto"` for the `yes` case there).

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Codex hooks described as "experimental, disabled by default" | Stable, `default_enabled: true` [VERIFIED: Context7 `/openai/codex`, `codex-rs/features/src/lib.rs`, and live `codex features list` output this session: `hooks stable true`] | Confirmed against codex-cli 0.155.0, 2026-09-19 | CODEX-05's roadmap wording was already corrected 2026-09-19 per CONTEXT.md; this research independently reconfirms the premise with a live probe, not just a doc read |
| `codex_hooks` config key | Deprecated legacy alias for `[features] hooks`, with an explicit CLI doctor warning [VERIFIED: Context7 `/openai/codex`, `codex-rs/cli/src/doctor.rs`, `codex-rs/features/src/legacy.rs`] | Ongoing | D-18's "explicitly set `[features] hooks = false` (or the deprecated `codex_hooks = false`)" check must read BOTH keys via the hand-rolled TOML helpers, since either can disable the feature |
| Assuming Codex CLI hooks are Claude-hook-JSON-identical | Structurally very similar top-level shape (`hooks.<Event>[].matcher` / `.hooks[].{type,command,timeout}`) but with additional fields Claude's fragment doesn't use (`statusMessage`, `additionalContextLimit`) and a documented `updatedInput`/`permissionDecision` output vocabulary Claude does not expose the same way | Confirmed via Context7 this session | The Codex fragment/adapter should NOT literally reuse Claude's `claudeFragmentEventBlocks` function untouched — the schemas are close enough to justify copying the PATTERN, not the code, per D-00's "never invent shared structure for two things that only look similar" caution generalized here |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Codex's PreToolUse hook handler's `command` string is executed through a POSIX shell such that `$(git rev-parse --show-toplevel)` in D-20's local command form is expanded before exec, the same way Claude Code's own hook command strings are | "Pattern 5" / D-20 | If Codex instead execs the literal string as an argv[0] without shell expansion, the local command form silently fails to resolve to a real path and the hook never runs — a live check (does the guard actually execute?) must precede committing to this exact command-string form |
| A2 | `codex debug prompt-input`'s skill/AGENTS.md rendering in an untrusted, unregistered project (this session's live probe) reflects the SAME trust gating a real interactive/`codex exec` session applies to those two mechanisms | "Live Codex CLI probes" | If `debug prompt-input` is a debug-only bypass of trust gating (plausible — it is a `debug` subcommand), CODEX-01/CODEX-06's real pass bar (L1-L7) could show DIFFERENT behavior (e.g. AGENTS.md/skills genuinely gated by trust in a real session) than this research's live probe suggests; the orchestrator's own live-verification plan must not treat this session's `debug prompt-input` output as a substitute for L3/L5's real-session check |
| A3 | Claude Code's own hook `timeout` field (`.claude/hooks/hooks.json`'s `"timeout": 5`) is also in seconds, making it directly comparable to Codex's confirmed `timeout_sec` unit | "Pattern 5" | If Claude's timeout field turns out to be a different unit, choosing "an explicit short timeout" for Codex by eyeballing Claude's `5` would be miscalibrated; low risk since both values are small and any reasonable single-digit-to-low-double-digit-seconds value is safe for a lightweight guard script |
| A4 | Codex's PreToolUse hook truly never carries an `agent_id`/subagent-identity field, based on the documented field tables not listing one | "Pitfall 4" | If Codex DOES send an undocumented `agent_id`-equivalent field on PreToolUse (docs are sometimes incomplete), D-21's per-subagent cooldown degrades unnecessarily; the live check (CODEX-05, in a real subagent-spawning session) is the only way to resolve this, and it must specifically inspect raw stdin JSON, not just observe fire/no-fire behavior |
| A5 | The exact byte offset/count claims about `internal/mcp/server.go`'s `instructions` const (554 bytes, skill sentence at byte 490) will still hold after D-29's rewrite makes the skill sentence harness-neutral | "AGENT-14 doc-drift test" section below | These are PRE-rewrite measurements, cited to ground the ≤600-byte/≤512-byte budget math for the planner, not a claim about the POST-rewrite string; the planner must re-measure after editing the sentence |

**A2 and A4 are the two assumptions that most directly affect whether CODEX-05/CODEX-06's live pass bar can be met as currently worded** — both require the orchestrator's own live session (not this research pass) to resolve.

## Open Questions

1. **Does Codex's PreToolUse hook input carry any field that distinguishes a subagent's tool calls from the primary agent's?**
   - What we know: The documented Common input fields and PreToolUse-specific field table list no such field; `agent_id`/`agent_type` are documented only for `SubagentStart`/`SubagentStop` [VERIFIED: Context7 `/llmstxt/learn_chatgpt_llms-full_txt`, fetched 2026-09-19].
   - What's unclear: Whether this is a genuine product limitation or a documentation gap.
   - Recommendation: Resolve during CODEX-05's live verification by spawning a subagent in a real Codex session and inspecting the exact PreToolUse stdin JSON for any subagent-identifying key before writing the adapter's key-derivation logic. If none exists, key on `session_id` alone for Codex and record the narrower cooldown granularity as an accepted, documented difference from Claude — do not silently invent a field.

2. **Is Codex's shell-invoked hook `command` string expanded by a shell, or exec'd as a literal argv?**
   - What we know: D-20 specifies `"$(git rev-parse --show-toplevel)/.codex/hooks/codegraph-pretooluse.sh"` for the local command string, which requires shell expansion to resolve to a usable path.
   - What's unclear: The exact invocation mechanism (`sh -c "<command>"` vs. splitting on whitespace and exec'ing argv[0] directly) is not something this research session's Context7/live-probe pass confirmed at the byte level.
   - Recommendation: The CODEX-01 live verification step (already planned, D-01 step 3) should include a minimal smoke test — install the guard, register it, trust it, and confirm via a side-effect (e.g. the guard writes to a marker file on invocation) that the `$(...)` command substitution actually resolved before relying on it in production.

3. **Does the "38 wire transcripts" figure in D-29 match the current on-disk count?**
   - What we know: `testdata/wireoracle/transcripts/` currently holds 42 `.golden` files [VERIFIED: `find testdata/wireoracle/transcripts -type f | wc -l`, run this session]; a doc comment elsewhere in `internal/mcp/instructions_contract_test.go` (read this session) separately states "38 committed wire-oracle transcripts" in one place and "24 frozen wire-oracle transcripts" in another comment in the same file.
   - What's unclear: Whether these numbers reflect different subsets (e.g. transcripts that embed the `instructions` string specifically, vs. the total golden-file count) or are simply stale comments.
   - Recommendation: Before D-29's re-freeze, run `grep -l '"instructions"' testdata/wireoracle/transcripts/*.golden | wc -l` (or the equivalent) to get the actual current count of transcripts embedding the instructions string, and reconcile against whichever figure (38, 42, or something else) that produces — do not assume 38 is still accurate without checking.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `codex` CLI | CODEX-01/02/03/04/05/06 live verification | ✓ [VERIFIED: `codex --version` run this session] | codex-cli 0.155.0 | — |
| `tmux` | FIX-03's re-run TTY-05 assertion (D-25), FIX-08 tmux harness generally | ✗ [VERIFIED: `command -v tmux` → not found, run this session] | — | The `tmux-e2e` CI job (`.github/workflows/ci.yml`) is the only environment that runs `//go:build tmux` tests today (`task test:tmux`, `TMUX_EXPECTED_VERSION: tmux 3.4` per Taskfile.yml); on this machine the orchestrator must either request `brew install tmux` explicitly, or defer the TTY-05 RED/GREEN evidence to a CI run and cite the run URL/SHA rather than a local execution |
| Go toolchain matching `go.mod`'s pin | Building/testing `internal/agents`, `internal/cli/tui` | Unconfirmed this session (not probed) — `go.mod` pins `go 1.26.6`; `test/tmux`'s own doc comment (read this session) warns a newer toolchain fails inside `cockroachdb/swiss` with `undefined: hashFn`/`undefined: fastrand64` | — | Prefix any local tmux-package build/test with `GOTOOLCHAIN=go1.26.6` per the package's own doc comment |

**Missing dependencies with no fallback:** none — tmux has a documented CI-based fallback.

**Missing dependencies with fallback:**
- `tmux` — use the `tmux-e2e` CI job, or request a local `brew install tmux` if the orchestrator needs local RED/GREEN evidence before opening a PR.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go's stdlib `testing` package (`go test`), plus a dedicated `//go:build tmux` real-PTY suite |
| Config file | none — flag-driven (`-update-plain-goldens`, `-update-wireoracle`-equivalent human redirect, `TMUX_EXPECTED_TESTS`/`TMUX_EXPECTED_VERSION` in `Taskfile.yml`) |
| Quick run command | `go test ./internal/agents/... ./internal/cli/... -run '<Pattern>' -v` |
| Full suite command | `go test ./...` (excludes `//go:build tmux`); `task test:tmux` (tmux-tagged, requires tmux) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CODEX-01 | Comment correction + live evidence recorded | manual/live (not automatable — the "before any codex.go change" live session is inherently a human/orchestrator-driven artifact) | n/a (evidence file, not a test) | n/a |
| CODEX-02 | TOML splice preserves indented/CRLF/subtable content; refuses inline/dotted conflicts | unit | `go test ./internal/agents/ -run 'TestSpliceTOMLTable|TestStripTOMLTable|TestCodex_'` | ✅ existing file, ❌ new fixtures (Wave 0 gap) |
| CODEX-03 | Skill installed at shared path, both scopes, idempotent | unit | `go test ./internal/agents/ -run 'TestOwnershipExactIdentity|TestCodex_'` | ✅ `ownership_test.go` exists; ❌ needs a Codex case in `ownershipWantSkillDir` |
| CODEX-04 | Repo-root AGENTS.md marker install/uninstall, sharing-aware | unit | `go test ./internal/agents/ -run 'TestCodex_|TestOpencode_'` (new sharing test name TBD) | ❌ Wave 0 gap — no test for D-11's sharing check exists yet |
| CODEX-05 | Hooks.json write, guard render, trust Note, opt-out on `[features] hooks=false` | unit | `go test ./internal/agents/ -run 'TestCodex_'`; adapter: `go test ./internal/cli/ -run 'TestRunHookPreToolUse'` (adjust to actual name) | ❌ Wave 0 gap — no Codex hook tests exist yet |
| CODEX-06 | Fresh session reaches for codegraph | manual/live | n/a | n/a |
| FIX-03 | Footer fits 100×30 with 8 targets | unit (model-level) + tmux (real-PTY) | `go test ./internal/cli/tui/ -run 'TestAgentPicker'`; `task test:tmux` (requires tmux; see Environment Availability) | ❌ Wave 0 gap — no model-level height test exists yet; tmux test exists but currently pinned to title-not-footer per its own comment |
| AGENT-14 | Doc mirrors code-derived capability table | unit | `go test ./internal/agents/ -run 'TestCapabilityDoc'` (new name TBD) | ❌ Wave 0 gap — new file needed, modeled on `internal/indexer/capability/matrix_test.go` |

### Sampling Rate
- **Per task commit:** the narrowest `-run` pattern covering the touched target/file.
- **Per wave merge:** `go test ./internal/agents/... ./internal/cli/...` (excludes tmux).
- **Phase gate:** `go test ./...` full suite green, plus a `tmux-e2e` CI run (or explicit `brew install tmux` local run) for the re-anchored TTY-05 assertion, before `/gsd-verify-work`.

### Wave 0 Gaps
- [ ] `internal/agents/toml_test.go` — indentation, CRLF, inline-table-refusal, dotted-key-refusal, subtable-in-range, multi-line-string, multi-line-array fixtures (D-07)
- [ ] `internal/agents/codex_test.go` — flip `TestCodex_Install_Local_IsUnsupportedNoWrite`/`TestCodex_DescribePaths_LocalEmpty` to their local-supported equivalents; add hooks/skill/AGENTS.md-sharing tests
- [ ] `internal/agents/ownership_test.go` — add a `case Codex:` to `ownershipWantSkillDir` (currently falls through to `default: return ""`)
- [ ] `internal/cli/tui/agentpicker_test.go` (or a new file) — the model-level `lipgloss.Height` regression test (D-25)
- [ ] A new `internal/agents/*_test.go` for AGENT-14's doc-drift check, modeled line-for-line on `internal/indexer/capability/matrix_test.go`'s `TestMatrix_DocMirrorsDescriptor`
- [ ] `internal/cli/install_test.go`/`uninstall_test.go` — RED-first test for `--target X --yes` resolving to exactly X (D-13)
- [ ] Framework install: none — everything needed is already in `go.mod`

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | This phase never touches Codex's own auth (`auth.json` is only symlinked read-only into the scratch HOME for live verification, never copied or modified) |
| V3 Session Management | no | N/A — no session state introduced |
| V4 Access Control | yes (narrowly) | Trust-gating of project-scoped `.codex/` config/hooks is Codex's own mechanism; codegraph's role is limited to (a) never writing a trust entry itself (D-10: "codegraph never writes the trust entry itself") and (b) never advising users to bypass trust in production docs (D-19: "users are never told to use `--dangerously-bypass-hook-trust`") |
| V5 Input Validation | yes | PreToolUse stdin JSON from Codex is untrusted input to `codegraph hook pretooluse --harness codex` — the existing Claude adapter's pattern (`hook_pretooluse.go:94-146`, read this session) already treats every field as adversarial: size-capped read (`maxHookStdinBytes = 1<<20`), tolerant JSON decode, `recover()`-wrapped RunE, never writes to stderr, never exits non-zero. The Codex branch must inherit this exact posture — including D-21's explicit "any doubt means silent" rule for `tool_input.command`'s string-vs-array ambiguity |
| V6 Cryptography | yes (narrowly) | Manifest content hashing (`hashContent`, manifest.go:140-146) uses `sha256` already in the codebase — no new crypto primitive needed for anything in this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| A hand-edited `.codex/config.toml` with a malicious/malformed `[mcp_servers.codegraph]` table that the splice logic misparses, silently corrupting or deleting unrelated user tables | Tampering (of user data, by codegraph's own bug) | D-07's fix + the RED-first fixture reproducing the exact real-world shape (indented header, later column-0 header) + the planted-regression mutation (column-0-only end scan) proving the guard actually catches the bug class |
| A foreign, non-codegraph PreToolUse hook block in `hooks.json` being silently overwritten or absorbed by codegraph's exact-identity ownership check | Tampering (of a user's unrelated hook) | Reuse `blockOwnsAnyCommand`/`commandIsOwned` (shared.go:174-218) UNCHANGED — these already implement exact-command-string identity (the hardened rule from commit `242ec0a`, cited throughout this codebase's history) and generalize to Codex's hooks.json with zero new logic, since the top-level shape is the same `hooks.<Event>[].hooks[].command` array |
| A malicious or malformed PreToolUse stdin payload from Codex (e.g. an oversized `tool_input.command`, or a `tool_input.command` that is neither string nor array) crashing or hanging the hidden `codegraph hook pretooluse` subcommand, which Codex's hook engine then treats as a failed hook | Denial of Service | Inherit the existing Claude adapter's defenses verbatim: `maxHookStdinBytes` cap, `recover()`-wrapped body, tolerant decode-and-return-silently on any parse failure — D-21's "any doubt means silent" is exactly this posture extended to the argv-vs-string ambiguity |
| The registered hook command embedding the running binary's absolute filesystem path directly in `hooks.json` (rather than only in the guard script), leaking a local path into a file that might be committed to a shared/team `.codex/hooks.json` | Information Disclosure | D-19's explicit requirement: "the registered definition stays byte-stable across upgrades: the ExecPath is rendered into the guard, never into the command" — the `hooks.json` entry's `command` field is always the guard SCRIPT path (project-relative for local scope, matching D-20's `$(git rev-parse --show-toplevel)/...` form), never the codegraph binary's own path |

## Live Codex CLI probes (this session, 2026-09-19, codex-cli 0.155.0, isolated scratch `HOME`/`CODEX_HOME`, no auth, no model session)

All probes below were run with `HOME`/`CODEX_HOME` pointed at a fresh scratch directory under the session's own scratchpad — never `~/.codex`, never `~/.agents`, never `~/.claude`. No `codegraph install`/`uninstall` was run in any form. No model session was invoked.

- `codex --version` → `codex-cli 0.155.0` [VERIFIED, matches CONTEXT.md's cited local version]
- `codex features list` → includes the line `hooks                                    stable             true` [VERIFIED — confirms CONTEXT.md's "Stable and on by default" claim with a fresh, independent probe]
- `codex mcp list --json` on an empty scratch `CODEX_HOME` (no config.toml at all) → `[]` [VERIFIED — empty JSON array, no error]
- `codex exec --help` confirms the presence of: `--dangerously-bypass-hook-trust`, `-C, --cd <DIR>`, `--skip-git-repo-check`, `--ephemeral`, `--ignore-user-config`, `--json`, `--ignore-rules` [VERIFIED — all flags exist exactly as CONTEXT.md's D-04 describes]
- `codex mcp add --help` confirms no `--scope`/project-targeting flag exists on the CLI's own `mcp add` subcommand — it always writes to the global `$CODEX_HOME/config.toml` [VERIFIED via `--help` output and cross-checked against Context7's `codex-rs/cli/src/mcp_cmd.rs` excerpt showing `run_add` calling `load_global_mcp_servers`]. Irrelevant to codegraph's own architecture, which writes `.codex/config.toml` directly rather than shelling out to `codex mcp add`.
- `codex debug prompt-input` run inside a throwaway git repo (initialized fresh under the scratch dir, containing a project `AGENTS.md` and a project `.agents/skills/testskill/SKILL.md`, with NO trust registration anywhere in the scratch `CODEX_HOME`'s config) produced a JSON array whose first `developer` message included a `<skills_instructions>` block enumerating:
  - `r0` = `$CODEX_HOME/skills/.system` (built-in system skills: `imagegen`, `openai-docs`, `plugin-creator`, `skill-creator`, `skill-installer`)
  - `r1` = the project's `.agents/skills` directory, listing `testskill` by name and description exactly as authored
  and a separate `user`-role message headed `"# AGENTS.md instructions for <project path>"` wrapping the project `AGENTS.md`'s exact content in `<INSTRUCTIONS>` tags.
  **This is a live, direct confirmation that Codex reads project `.agents/skills` and repo-root `AGENTS.md`** [VERIFIED] — but see Assumption A2: this ran without ANY explicit trust registration for the project, which may mean `debug prompt-input` does not apply the same trust gate a real interactive/`exec` session would; this nuance must be resolved by the orchestrator's own live-verification session, not assumed resolved by this observation alone.

## Sources

### Primary (HIGH confidence)
- `internal/agents/toml.go`, `toml_test.go`, `codex.go`, `codex_test.go`, `capabilities.go`, `skillshared.go`, `shared.go`, `manifest.go`, `opencode.go`, `types.go`, `instructions.go`, `claude_pretooluse.go`, `ownership_test.go`, `claude.go` (Capabilities excerpt) — all read in full or in relevant part this session
- `internal/cli/install.go`, `uninstall.go`, `hook_pretooluse.go` — read in full this session
- `internal/cli/tui/agentpicker.go`, `daemonpicker.go` — read in full this session
- `internal/nudge/classify.go`, `cooldown.go`, `text.go` — read in full this session
- `internal/mcp/server.go` (instructions const, byte-measured with a script this session), `instructions_contract_test.go` — read in full this session
- `internal/indexer/capability/matrix_test.go` — read in full this session (AGENT-14's structural precedent)
- `test/tmux/install_cancel_test.go`, `main_test.go`, `frame_stability_test.go` — read in full this session
- `.claude/hooks/hooks.json`, `.claude/hooks/pretooluse-nudge.sh`, `claudeassets.go` — read in full this session
- `charm.land/bubbles/v2@v2.1.1/list/list.go` (module cache, `populatedView`) — read this session, root-causes FIX-03 with line numbers
- `charm.land/bubbletea/v2@v2.0.8/tea.go` (`View` struct) — read this session
- `charm.land/lipgloss/v2@v2.0.5/size.go` (`Height` function) — read this session
- `go.mod` — read this session (dependency versions, absence of a TOML library)
- `.planning/todos/pending/2026-09-18-install-yes-discards-explicit-target.md` — read this session (D-13's exact root cause and fix)
- Context7 `/openai/codex` (official `openai/codex` GitHub repo, High source reputation) — `codex-rs/hooks/src/engine/dispatcher.rs`, `codex-rs/config/src/hooks_tests.rs`, `codex-rs/app-server-protocol/src/protocol/v2/plugin.rs`, `codex-rs/features/src/lib.rs`, `codex-rs/features/src/legacy.rs`, `codex-rs/cli/src/doctor.rs`, `codex-rs/cli/src/mcp_cmd.rs`, `codex-rs/cli/src/main.rs`, `codex-rs/codex-home/src/instructions/mod.rs` — fetched 2026-09-19
- Context7 `/llmstxt/learn_chatgpt_llms-full_txt` (mirrors `learn.chatgpt.com/docs/*`, High source reputation) — `docs/hooks`, `docs/llms-full.txt`, `docs/config-file/config-reference`, `docs/config-file/config-sample`, `docs/customization/overview`, `docs/changelog`, `docs/agent-approvals-security`, `docs/non-interactive-mode` — fetched 2026-09-19
- Live `codex` CLI invocations (codex-cli 0.155.0), isolated scratch `HOME`/`CODEX_HOME`, run this session — see "Live Codex CLI probes" section

### Secondary (MEDIUM confidence)
- `.planning/phases/07-codex-parity/07-CONTEXT.md`'s own "Current Codex reference" block (already dated/verified 2026-09-19 by the orchestrator's prior research pass) — used as a starting point, then independently re-verified against Context7 and live probes rather than trusted verbatim

### Tertiary (LOW confidence)
- None used without independent verification in this pass

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, every existing one confirmed against `go.mod` directly
- Architecture: HIGH — every pattern cited traces to a specific file/line read this session
- Codex CLI mechanics: MEDIUM-HIGH — Context7-sourced from the official repo and its docs mirror, cross-checked against 6 independent live probes this session; the two flagged gaps (A2, A4) are explicitly named rather than silently assumed
- Pitfalls: HIGH for TOML/FIX-03/`--yes` (all three have exact line-level root causes found this session); MEDIUM for the Codex subagent-id gap (documentation-based, not yet live-session-confirmed)

**Research date:** 2026-09-19
**Valid until:** 14 days for the Codex-CLI-mechanics sections (Codex CLI ships frequently — `codex-cli 0.155.0` observed this session may not match the version at execution time; re-run `codex --version` and `codex features list` before relying on any Codex-specific claim above); 30 days for the in-repo code-path sections (stable until the next unrelated refactor touches these files)
