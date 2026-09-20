# Phase 6: Claude Code PreToolUse Nudge - Research

**Researched:** 2026-09-19
**Domain:** Go CLI hook subcommand wiring, embedded-template rendering, sidecar-manifest opt-in persistence, Unix sentinel-file safety
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Carried forward (binding)**
- **D-00 (from Phase 4/5):** tests assert only what this repo owns: the guard's and the subcommand's bytes and behaviour, the settings entries we write, and ownership. They never assert Claude Code's matcher or delivery behaviour, and never Go dependency management. That Claude Code actually matches a call and delivers the context is shown by the live session (D-18), not by a unit test. Every new guard is positive-controlled in `06-MUTATION-LOG.md`. Go RED evidence is the `test(06-NN):` commit plus a pasted `--- FAIL:` transcript.
- **Contract (2026-09-14 reframe; carried verbatim into CODEX-05):** the hook emits only `hookSpecificOutput.additionalContext`. It exits 0 on every path, never emits `permissionDecision`, `decision` or `continue`, and never exits 2. State the contract harness-neutrally, so Phase 7 reuses it rather than re-deriving it.
- **Ownership:** hooks are owned by exact command string, never by matcher (`242ec0a`). A hand-edited own entry is duplicated, not overwritten. An unrelated `PreToolUse` entry under the same event stays byte-identical.

**A. Hook form & trigger heuristic**
- **D-01 (maintainer: option 1, "Go subcommand plus a sh guard"):** the logic lives in a hidden Go subcommand, `codegraph hook pretooluse`, reached through a tiny embedded POSIX `sh` guard.
  - **Guard:** dogfooded under `.claude/hooks/`, embedded via `claudeassets` beside `session-nudge.sh`. It is the registered hook command, so it carries the owned identity.
  - **Guard job:** (a) the D-04 directory check, exiting 0 at once when un-indexed (no binary started); (b) exit 0 silently when the codegraph binary is missing or not executable; (c) run the binary with stdin passed through, then `exit 0` whatever the binary did.
  - **Rejected alternatives:** Python via `uv` (interpreter start, downloads, macOS stub dialog); pure `sh` (fragile JSON, BSD/GNU `find -mmin` mismatch).
- **D-01a:** The subcommand is hidden, registered with one bare allowlist line in `internal/cli/testdata/cli-reference-allowlist.txt` (the `codegraph man` precedent). It must not open the index, the store or the daemon. It never returns an error to cobra. It recovers panics, and writes nothing to stdout except the one pinned JSON object when it fires. The core is a pure harness-neutral function; the Claude stdin/stdout envelope is a thin adapter.
- **D-01b:** The binary's location must not be part of the registered command's identity. The recommended approach is to render the absolute `ExecPath` into the guard script from an embedded template at install time. The settings entry stays the stable script path; install and upgrade refresh the script as own content. Leaning on `PATH` alone is rejected.
- **D-02:** Bash detection has two layers: `if` rules pre-filter (`Bash(grep *)`, `Bash(rg *)`, `Bash(find *)`); the Go core confirms the command's first word (after stripping `VAR=val` assignments) is `grep`/`egrep`/`fgrep`/`rg`/`find`. On any parse doubt, stay silent.
- **D-03:** `Grep` and `Glob` always qualify. `Read` qualifies unless `file_path` is an obvious non-code file (`*.md`, `*.json`, `*.yaml`/`*.yml`, `*.toml`, `*.txt`, `*.lock`).
- **D-04:** Indexed = the guard's `[ -d "${CLAUDE_PROJECT_DIR:-.}/.codegraph" ]` holds, run before any process starts or stdin is read.

**B. Cooldown & scope**
- **D-05 (maintainer: "C, once per minute"):** fire on the first matched call, then at most once every 60 seconds per key. The 60s value is one named Go constant.
- **D-06:** subagents are included, each with its own cooldown. Key = `(session, agent)`: `session_id` plus `agent_id`, or the literal `main` when absent.
- **D-07:** Session id is `$CLAUDE_CODE_SESSION_ID`, falling back to `session_id` from stdin. With neither, stay silent, never fire unkeyed.
- **D-08:** Sentinel is per key, under `os.TempDir()` (honours `TMPDIR`), in `codegraph-nudge-<uid>/`. Directory created mode 0700, checked via `Lstat`; stay silent if symlinked or not owned by current uid. Recording a fire must never write through a symlink — refuse a symlinked sentinel, never truncate someone else's file. Age = `time.Since(modTime)` against the D-05 constant; clock injectable for tests. A rare double fire on parallel races is accepted. Any failure means silent, exit 0.

**C. Opt-in & lifecycle**
- **D-09:** Opt-in is a Claude-only bool flag on `install`, shaped like `--auto-allow` (working name `--pretool-nudge`). Goes through the CLI-reference drift gate and flag accounting. No picker row. Print a `note:` when given but Claude isn't among resolved targets.
- **D-10:** The opt-in is sticky, recorded in the manifest as new Files keys for the guard script and `settings.json#hooks.PreToolUse`. `install` and `upgrade` refresh it while the manifest records it. Only an explicit `--pretool-nudge=false` (via cobra `Changed`) or `uninstall` removes it. `upgrade`'s refresh (`upgrade.go:48-73`) must carry the recorded opt-in; today it passes only `AutoAllow:false`.
- **D-11:** `uninstall` always attempts `removeHookEntry("PreToolUse", own)` and guard-script removal, reporting `not-found` when never opted in. The ownership table runs the Claude leaves with the opt-in on and plants an unrelated `PreToolUse` block under the same matcher, which must survive byte-identical.
- **D-12:** Registration mirrors SessionStart: shell form, `${CLAUDE_PROJECT_DIR}/.claude/hooks/<guard>` local / absolute guard path global; the command never contains the binary path (D-01b); one handler per matcher/`if` rule; an explicit short `"timeout"`; no `statusMessage`.
- **D-13:** `HookFiles` for `claude-json` names the new guard script. `--print-config-style` keeps `hooks=claude-json` unchanged. This repo's `.claude/settings.json` registers the hook. `TestHookRegistrationMatchesFragmentAndScript` extends to PreToolUse.

**D. Text & validation**
- **D-14:** Nudge text is a new factual one-liner, a byte-exact pinned Go constant, naming only `codegraph_explore` and the `codegraph explore` CLI fallback. Existing nudge drift guards extend to cover it. No new env token outside the drift guard's allowlist.
- **D-15:** Corpora are harness-neutral `testdata` rows `{tool, input, want}` (true positives: where-is-X; false positives: legitimate grep use). A Go table test drives the pure core. CODEX-05 reuses the rows.
- **D-16:** Forced-error coverage at subcommand level (no stdin, malformed JSON, oversized input; both session-id sources absent; unwritable/foreign-owned/symlinked sentinel; in/out of cooldown; parallel runs; recovered panic) and guard level (`.codegraph` missing/is-a-file; binary missing/not-executable; binary non-zero/crash; `CLAUDE_PROJECT_DIR` unset) — both asserting exit 0, stdout empty or exactly the pinned JSON, no `permissionDecision`/`decision`/`continue`. Both positive-controlled by planted mutations.
- **D-17:** Cooldown tested with an injected clock and planted sentinel mtimes (`os.Chtimes`), never `sleep`.
- **D-18 (live check; pass bar locked):** Herdr-driven fresh `claude --debug-file` sessions in a scratch indexed repo (opt-in on) plus an un-indexed negative control. PASS requires: (1) first matched call in a fresh main-thread session fires exactly once, visible in transcript/debug log; (2) no two fires share a key within 60s; (3) after a ≥60s gap, the next matched call fires again; (4) a subagent's first matched call fires once regardless of the main thread's cooldown; (5) after `/clear`, the next matched call fires; (6) the un-indexed control has 0 fires; (7) 0 "hook error" notices, no permission prompt/deny/block attributable to the hook. Recorded-not-gated: matched-call count, fire rate, true-positive share, uptake, wall time per run.

### Claude's Discretion
- The guard script name, the exact flag name (working: `--pretool-nudge`), the subcommand's package placement, the timeout value, and the nudge wording within D-14.
- The sentinel mechanics within D-08 (mtime vs content timestamp; `O_NOFOLLOW` vs an `Lstat` check before writing).
- How the binary path reaches the guard, within D-01b (templated guard recommended).

### Deferred Ideas (OUT OF SCOPE)
- Exec-form (`args`) or quoting for the *existing* SessionStart entry.
- `once: true` via skill-frontmatter hooks (declined: Claude-only key in portable SKILL.md).
- A SessionEnd sentinel-cleanup hook (OS reaps `$TMPDIR`; cooldown makes it unnecessary).
- A `nudge` field/column in the capability table (Phase 7, AGENT-14).
- The Codex nudge (CODEX-05, Phase 7) and other harness nudges (v2), both reusing the D-15 corpus.
- Pre-existing: `install --yes` discards an explicit `--target` (open todo); affects whether the new flag's Claude selection is honoured.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| NUDGE-03 | PreToolUse hook returns only `additionalContext`, exits 0 unconditionally, never emits `permissionDecision` | §Architecture Patterns (subcommand wiring, exit-0 guarantee via `main.go`/guard double-net), §Common Pitfalls 1/6, verified `main.go`/`root.go` have no code path that returns a non-nil error from the hidden subcommand's own RunE unless it chooses to |
| NUDGE-04 | Fires first-call-then-60s-cooldown per (session, agent), silent with zero overhead when un-indexed | §Common Pitfalls 2 (startup cost measured), §Code Examples (sentinel open-with-O_NOFOLLOW pattern), §Environment Availability (`syscall.O_NOFOLLOW` confirmed both GOOS) |
| NUDGE-05 | Validated against false/true-positive corpora; fire rate measured live | §Validation Architecture (table test + D-18 mapping) |
| NUDGE-06 | `install`/`uninstall` register/remove via `writeHookEntry`/`removeHookEntry` as opt-in; hand-edited entry duplicates | §Architecture Patterns (manifest Files-key opt-in, `claudePreToolUseBlocks` mirror), §Common Pitfalls 3/4/5 |
</phase_requirements>

## Summary

This phase adds a second Claude Code hook (`PreToolUse`) to a codebase whose only existing hook (`SessionStart`/`session-nudge.sh`) is deliberately stateless and never invokes the `codegraph` binary. Three things make this phase genuinely new relative to that precedent, all confirmed by reading the actual source rather than by extrapolating from CONTEXT.md's prose: (1) this is the **first** guard script that must have a per-machine value (`ExecPath`) baked into its own bytes, which the existing `writeEmbeddedFile` byte-identity idempotency check already tolerates correctly as long as the *rendered* bytes are compared, not the literal embedded template; (2) this is the **first** sticky, manifest-persisted CLI opt-in — `AutoAllow` is deliberately non-sticky (`upgrade.go` always passes `AutoAllow:false` with a comment explaining why), so the opt-in-persistence mechanism for `--pretool-nudge` has no code to copy, only a pattern to extend (infer opt-in from **presence of a Files key**, not a new manifest field — this needs no schema bump); (3) the hidden subcommand executes on **every** matched tool call in an indexed repo (the guard has no cooldown logic itself — cooldown lives inside the Go binary), so its process-startup cost is paid up to once per grep/find/Grep/Glob/Read, not once per minute — measured this session at ~9ms warm on this repository's own build, ~6-7ms of which is CGo/tree-sitter-registration overhead over a bare Go binary's ~2-3ms.

Root.go/main.go were read in full: there is no `PersistentPreRun`/`PersistentPreRunE`/`cobra.OnInitialize` anywhere in `internal/cli`, and `main.go`'s only behavior is `cli.Execute()` → print+exit(1) on a non-nil error. This means D-01a's "never returns an error to cobra" constraint is sufficient on its own to guarantee the hidden subcommand can never trigger `main.go`'s exit-1 path — there is no other global initialization hook to audit or defeat. The package-level `init()` functions that register all 12+ tree-sitter language grammars (`internal/indexer/languages_*.go`) DO run unconditionally at process start for every `codegraph` invocation, because `internal/cli` transitively imports `internal/indexer` (via `daemon.go`, `serve.go`, `index.go`, `init.go`, `sync.go`) and Go runs every imported package's `init()` regardless of which subcommand executes — but each `init()` is a cheap map insert (`registerLanguage` stores a struct with an unevaluated closure); the actual CGo `tree_sitter_<lang>.Language()` call is never invoked at init, only lazily inside a parser constructor closure nothing in the hook path calls.

**Primary recommendation:** implement the guard-script ExecPath templating as a `text/template`-rendered variant of the existing embedded-script pattern (new `//go:embed` source, a `text/template` execute step at install time, then the *rendered* result piped through the unmodified `writeEmbeddedFile`), implement the sticky opt-in as a manifest Files-key presence check (zero schema bump), and implement the sentinel via `os.OpenFile(path, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)` after an `Lstat` ownership check, writing a fresh timestamp as file *content* rather than calling `os.Chtimes(path, …)` after the fact — `os.Chtimes` re-resolves the path by name and therefore reintroduces exactly the TOCTOU-through-symlink race D-08 exists to close.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Repo-indexed check (`.codegraph` dir exists) | Guard (sh) | — | Must happen before any process starts (D-04); zero-cost path for the un-indexed case is the whole point of a shell pre-filter |
| Bash-first-word / matched-call classification | Go subcommand (core) | Claude's own `if` handlers | `if` rules are best-effort pre-filters (docs' own word); the Go core is the authoritative, testable classifier (D-02) |
| Cooldown / sentinel state | Go subcommand (core) | OS temp filesystem | Stateful logic belongs in a real language with real tests (D-05/D-08), not shell; the filesystem is the only durable state a stateless per-call process can use |
| Nudge text | Go subcommand (core, pinned const) | — | A single pinned byte-exact string is the whole "don't hand-roll templating" answer here — no templating of the *nudge text itself*, only of the *guard script's ExecPath* |
| Registration / ownership / removal | `internal/agents` (CLI/install layer) | Claude Code's own settings.json | Exact-identity ownership (`242ec0a`) is this repo's own invariant, enforced entirely in `internal/agents`; Claude Code only stores whatever JSON this repo wrote |
| Opt-in persistence across `upgrade` | `internal/agents` manifest (Files map) | `internal/cli/upgrade.go` | The manifest is the only durable record of "what did the user ask for" this repo has; `upgrade.go`'s refresh step must read it, not assume a flag default |

## Standard Stack

No new external dependency is introduced by this phase. Every mechanism below is Go's standard library plus code already vendored in this repository (`spf13/cobra`, `spf13/pflag` via cobra, `encoding/json`).

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `text/template` (stdlib) | Go 1.26.6 (pinned, `go.mod`) | Render `ExecPath` into the embedded guard-script template at install time | Stdlib, zero new dependency; the only templating need is one variable substitution, which is exactly `text/template`'s minimum use case — a hand-rolled `strings.Replace` is a viable, even simpler, alternative discussed under Don't-Hand-Roll below |
| `syscall` (stdlib) | Go 1.26.6 | `O_NOFOLLOW` open flag, `Stat_t.Uid` ownership check | `[VERIFIED: go doc syscall O_NOFOLLOW, GOOS=darwin → 0x100; GOOS=linux,GOARCH=amd64 → 0x20000; go1.26.6 toolchain, this session]` — both target GOOS values define the constant; darwin's `Stat_t` and linux's `Stat_t` both carry a `Uid uint32` field `[VERIFIED: go doc syscall.Stat_t, both GOOS, this session]` |
| `os` (stdlib) | Go 1.26.6 | `os.TempDir()`, `os.Lstat`, `os.OpenFile`, `os.MkdirAll` (mode 0700) | `os.TempDir()` honours `$TMPDIR` on Unix `[CITED: pkg.go.dev/os#TempDir]`; `os.Lstat` does not follow the final symlink component, the exact property D-08 needs |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `encoding/json` (stdlib) | Go 1.26.6 | Decode Claude's stdin JSON envelope, encode the pinned `additionalContext` response | Already the pattern `readJSONFileStrict`/`writeJSONFile` use elsewhere in this repo |
| `spf13/cobra` (existing dep) | pinned in `go.mod` | Register the hidden `hook pretooluse` subcommand | Already the CLI framework; `man.go`/`renamed.go` are the direct hidden-command precedent |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `text/template` for ExecPath substitution | `strings.Replace(tmpl, "{{EXECPATH}}", opts.ExecPath, 1)` | Simpler, no new import surface (still stdlib either way), but loses `text/template`'s automatic escaping and any future need for a second substitution (e.g. a timeout value) would require hand-rolling a second placeholder convention. Either is legitimate; `text/template` is recommended only because this repo has zero existing string-templating precedent to diverge from, and a `{{.ExecPath}}` placeholder is self-documenting inside a checked-in `.sh` file read by a human. |
| A new manifest struct field (e.g. `PreToolNudge bool`) | Infer opt-in from `Files` key presence | A new field repeats the exact "costly" precedent `Targets` set at schema 2 (self-heal logic needed for older binaries reading/rewriting the manifest) for zero benefit — `Files` is already `map[string]string`, and D-10 already asks for "new Files keys," not a new field. |

**Installation:** none — no `go.mod` change required.

## Package Legitimacy Audit

No external packages are installed by this phase. All new code uses `syscall`, `os`, `text/template`, `encoding/json`, and the already-vendored `spf13/cobra` — every one already a direct or stdlib dependency of this repository. This section is intentionally empty of rows.

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
Claude Code tool-call pipeline (PreToolUse event)
        │
        ▼
 ┌──────────────────────────┐   registered exactly like SessionStart:
 │ .claude/settings.json    │   shell command = guard script path,
 │  hooks.PreToolUse[]      │   one block per matcher (Bash/Grep/Glob/Read),
 │  { matcher, if, hooks[] }│   short explicit "timeout"
 └────────────┬─────────────┘
              │ Claude Code spawns the guard, pipes event JSON to stdin
              ▼
 ┌───────────────────────────────────────────┐
 │ embedded guard (sh), ExecPath templated in │
 │  1. [ -d "$CLAUDE_PROJECT_DIR/.codegraph" ]│──▶ not indexed: exit 0, NO process started (D-04)
 │  2. [ -x "$EXECPATH" ]                     │──▶ missing/not-exec: exit 0 silently
 │  3. exec "$EXECPATH" hook pretooluse       │
 │     (stdin passed through)                 │
 │  4. exit 0  (always, regardless of $?)     │──▶ NUDGE-03's "never non-zero" guarantee
 └────────────────────┬────────────────────────┘
                       ▼
 ┌─────────────────────────────────────────────────────────┐
 │ codegraph hook pretooluse  (hidden cobra subcommand)     │
 │  - recover() wraps everything (D-01a)                    │
 │  - decode stdin JSON (session_id, agent_id, tool_name,   │
 │    tool_input) — malformed/oversized/absent → silent     │
 │  - pure core: classify(tool_name, tool_input) → fire?    │
 │    (D-02/D-03 heuristics; D-15 corpus drives this)        │
 │  - if fire: sentinel key = session_id[+":"+agent_id]      │
 │    else "main"; check/refresh mtime under                │
 │    os.TempDir()/codegraph-nudge-<uid>/<key>              │
 │    (Lstat ownership+symlink guard, O_NOFOLLOW open)      │
 │  - emit {"hookSpecificOutput":{"hookEventName":           │
 │    "PreToolUse","additionalContext":"<pinned text>"}}     │
 │    or nothing at all                                      │
 │  - RunE always returns nil (never reaches cobra's error   │
 │    path in main.go)                                       │
 └─────────────────────────────────────────────────────────┘

Install-time (codegraph install --pretool-nudge, Claude target only):
 ┌────────────────────────┐   ┌──────────────────────────┐   ┌─────────────────────────┐
 │ claudeassets (embedded) │──▶│ render ExecPath template  │──▶│ writeEmbeddedFile(path,  │
 │ pretooluse-guard.sh.tmpl│   │ (text/template, one field)│   │ rendered, executable)   │
 └────────────────────────┘   └──────────────────────────┘   └───────────┬─────────────┘
                                                                          ▼
 ┌────────────────────────────┐   ┌───────────────────────────┐  ┌───────────────────────┐
 │ claudePreToolUseBlocks(loc) │──▶│ writeHookEntry(settings,   │  │ recordSkillManifest:   │
 │ (mirrors                    │   │ "PreToolUse", blocks,      │  │ Files[guard-script]=  │
 │  claudeSessionStartBlocks)  │   │ ownCommands)               │  │ Files[hooks-frag]=... │
 └────────────────────────────┘   └───────────────────────────┘  └───────────────────────┘
```

### Recommended Project Structure
```
internal/cli/
├── hook.go              # newHookCmd() (Hidden, groupless) + newHookPreToolUseCmd()
├── hook_pretooluse.go    # the RunE adapter: stdin decode, sentinel I/O, JSON emit
internal/nudge/           # NEW — the harness-neutral pure core CODEX-05 also imports
├── classify.go           # classify(toolName, toolInput string) (fire bool)
├── classify_test.go      # D-15 table test driving testdata/ corpus rows
├── testdata/
│   └── corpus.json        # {tool, input, want}[] — true/false positive rows
├── cooldown.go           # sentinel key derivation + mtime check/refresh, injectable clock
├── cooldown_test.go       # D-17: os.Chtimes-planted mtimes, no sleep
└── text.go               # the pinned nudge-text Go constant (D-14)
.claude/hooks/
└── pretooluse-nudge.sh.tmpl  # NEW embedded template, {{.ExecPath}} placeholder (claudeassets.go gains a 4th //go:embed line)
internal/agents/
├── claude.go             # claudePreToolUseBlocks(loc), claudeHooksGuardScriptPath(loc) — mirrors existing SessionStart helpers
├── manifest.go           # new manifestKeyGuardScript / manifestKeyPreToolUseFrag const keys (Files map — no schema bump)
```

### Pattern 1: Templated embedded guard script (new — no existing precedent to copy verbatim)

**What:** `claudeassets.go` currently embeds three *literal* files with no per-machine substitution — `session-nudge.sh` is byte-identical on every machine. The PreToolUse guard is the first artifact that must differ per machine (it names the absolute `ExecPath`).

**When to use:** Any embedded script that must reference the running binary's own path, per D-01b's rejection of relying on `$PATH`.

**Example:**
```go
// Source: this repository, claudeassets.go:33-58 (existing pattern, extended)
//go:embed .claude/hooks/pretooluse-nudge.sh.tmpl
var FS embed.FS // (additional //go:embed line alongside the existing three)

// internal/agents/claude.go — new helper mirroring claudeSessionStartBlocks'
// "decode the embedded fragment, then rewrite" shape, but templating a
// script body instead of rewriting a JSON "command" field.
func renderPreToolUseGuard(execPath string) (string, error) {
    tmplBytes, err := claudeassets.PreToolUseGuardTemplate() // new accessor
    if err != nil {
        return "", err
    }
    tmpl, err := template.New("guard").Parse(string(tmplBytes))
    if err != nil {
        return "", err
    }
    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, struct{ ExecPath string }{execPath}); err != nil {
        return "", err
    }
    return buf.String(), nil
}
```
The *rendered* result is what `writeEmbeddedFile(path, rendered, true)` compares against the on-disk bytes — `writeEmbeddedFile` itself (`internal/agents/shared.go:301-341`) needs no change; it already does a plain byte-identity check regardless of where the `content` string came from. This is confirmed by reading its signature: `func writeEmbeddedFile(path, content string, executable bool) (FileResult, error)` `[VERIFIED: internal/agents/shared.go:301]` — it takes a `content string`, not an embed.FS reference, so a rendered template is a drop-in argument.

### Pattern 2: Sticky opt-in via manifest Files-key presence (new mechanism; `AutoAllow` is explicitly NOT this)

**What:** `InstallOptions.AutoAllow` is deliberately per-invocation and non-sticky — `internal/cli/upgrade.go:41-47` states this explicitly: `[VERIFIED: internal/cli/upgrade.go:41-47]` "AutoAllow is always passed as false. --auto-allow is a per-invocation choice the user made at install time and is not recorded in the manifest, so re-asserting it on every upgrade would silently re-add a permission the user may have deliberately removed afterward." D-10 requires the opposite behavior for `--pretool-nudge`: it must be sticky and survive `upgrade`.

**When to use:** Any future opt-in flag that must persist across `upgrade` without a new manifest schema version.

**Example:**
```go
// internal/agents/manifest.go — additive consts, Files stays map[string]string
const (
    manifestKeyGuardScript    = "hooks/pretooluse-nudge.sh"
    manifestKeyPreToolUseFrag = "settings.json#hooks.PreToolUse"
)

// New exported helper internal/cli/upgrade.go needs (does not exist today):
// reads the manifest's Files map directly rather than inferring from
// ConfiguredSkillLocations alone (which only proves a manifest EXISTS,
// not which Files keys it carries — internal/agents/manifest.go:277-302
// `[VERIFIED: internal/agents/manifest.go:277-302]` — ConfiguredSkillLocations
// returns []Location with no Files detail).
func PreToolNudgeConfigured(loc Location) bool {
    path, err := claudeManifestPath(loc)
    if err != nil {
        return false
    }
    m, present, err := readManifest(path)
    if err != nil || !present {
        return false
    }
    _, ok := m.Files[manifestKeyGuardScript]
    return ok
}
```
`upgrade.go`'s `refreshInstalledSkills` (currently `internal/cli/upgrade.go:48-73`) then reads this per-location before calling `Install`, passing `InstallOptions{ExecPath: execPath, AutoAllow: false, PreToolNudge: PreToolNudgeConfigured(loc)}` instead of a hardcoded `false`.

**No schema bump required.** `skillManifest.Files` is already declared `map[string]string` `[VERIFIED: internal/agents/manifest.go:62]` (`Files map[string]string \`json:"files"\``) — adding new keys to an already-generic map is data, not shape. This is a materially different case from the schema-1→2 bump, which added a whole new **struct field** (`Targets []TargetID`) `[VERIFIED: internal/agents/manifest.go:69]` and required D-07's self-heal handling for older binaries. A new struct field changes the manifest's *shape*; a new map key does not. The one caveat (see Common Pitfalls) is that an **older binary** running `Install()` after this phase ships would write a smaller `Files` map lacking the two new keys, silently dropping the sticky record on that machine — this is the same class of forward/backward tension `Targets`' self-heal already manages for a different field, and is out of this phase's scope to defend against (no downgrade path exists today).

### Pattern 3: Sentinel write without a symlink-following gap (D-08)

**What:** D-08 explicitly forbids writing through a symlink and forbids truncating a file the process does not own. Two Go primitives interact here in a way worth flagging: `os.Lstat` does not follow a symlink, but `os.Chtimes(path, atime, mtime)` **does** — it operates by path name via `utimes`/`utimensat`, which by default follows the final symlink component. A naive "Lstat to check, then Chtimes to record" sequence has a TOCTOU gap: nothing prevents a symlink from being swapped in between the two calls.

**When to use:** Any sentinel/lockfile write in a shared, world-writable-adjacent temp directory (`os.TempDir()` is often `/tmp`, shared across users on multi-user Unix hosts).

**Example:**
```go
// Avoids the TOCTOU gap entirely: open with O_NOFOLLOW (fails outright if
// the final component is a symlink, atomically with the open itself), and
// write fresh content to the ALREADY-OPEN fd rather than calling
// os.Chtimes(path, ...) afterward (which re-resolves the path by name).
func recordFire(path string, now time.Time) error {
    info, err := os.Lstat(path)
    if err == nil {
        if info.Mode()&os.ModeSymlink != 0 {
            return errSentinelIsSymlink // D-08: refuse, stay silent upstream
        }
        if st, ok := info.Sys().(*syscall.Stat_t); ok && st.Uid != uint32(os.Getuid()) {
            return errSentinelForeignOwner
        }
    }
    f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
    if err != nil {
        return err // includes ELOOP if a symlink appeared between Lstat and Open
    }
    defer f.Close()
    if err := f.Truncate(0); err != nil {
        return err
    }
    _, err = f.WriteString(now.UTC().Format(time.RFC3339Nano))
    return err
}
```
`syscall.O_NOFOLLOW` is defined for both target platforms: `0x100` on darwin, `0x20000` on linux/amd64 `[VERIFIED: go doc syscall.O_NOFOLLOW, GOOS=darwin and GOOS=linux GOARCH=amd64, go1.26.6 toolchain, this session]`. Combining it with the portable `os.O_RDWR|os.O_CREATE` flags works on Unix because Go's `os` package flag constants are bitwise-distinct from and safely OR-able with raw `syscall` flags on unix build targets (a widely-used idiom; darwin and linux are the only two GOOS values this phase's scope requires per the additional_context note).

### Anti-Patterns to Avoid
- **`os.Chtimes(path, …)` as the sole "record a fire" mechanism:** re-resolves `path` by name, following a symlink if one has been substituted since the last check — reintroduces the exact race D-08 exists to close. Prefer writing to an already-opened, `O_NOFOLLOW`-protected file descriptor.
- **A new manifest struct field for opt-in state:** repeats the `Targets`/schema-2 cost for no reason — `Files` map-key presence already expresses "on/off" with zero schema change.
- **Calling `cgo.NewGoParser()`-style constructors from the hidden subcommand's own init path:** none of this phase's code needs to; flagged here only because `internal/indexer`'s `init()` functions run unconditionally in this binary (see Common Pitfalls #2) and a future contributor could mistakenly believe indexer language registration is "free" to invoke eagerly — it registers cheaply, but constructing an actual parser does real CGo work and must stay behind the interfaces this phase never touches.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ExecPath substitution into a script | A custom `{{PLACEHOLDER}}` string-scanning function | `text/template` (stdlib) or a single `strings.Replace` call | Either is fine; a hand-rolled multi-placeholder scanner would be over-engineering for exactly one substitution, and `text/template` is stdlib so "don't hand-roll" here means "don't write your own templating engine," not "you must use `text/template`" |
| Symlink-safe file write | A custom "check-then-write" wrapper without `O_NOFOLLOW` | `os.OpenFile(..., syscall.O_NOFOLLOW, ...)` | The check-then-write shape is exactly the TOCTOU class `O_NOFOLLOW` exists to close atomically at the syscall level; a Go-level "check first" retains the gap |
| JSON stdin decoding for the hook envelope | A hand-rolled line scanner over Claude's stdin JSON | `encoding/json` `Decoder` with `io.LimitReader` for the oversized-input case (D-16) | Already this repo's own established pattern (`readJSONFileStrict`) |

**Key insight:** this phase's genuine novelty is not in any one primitive (every primitive above is stdlib) — it's in composing three already-proven-elsewhere patterns (embedded-file idempotency, exact-identity hook ownership, manifest-recorded state) into one new artifact that needs a machine-specific rendered value inside a byte-identity-compared file, which no existing test or helper in this repo has had to reconcile before.

## Common Pitfalls

### Pitfall 1: Assuming a global `PersistentPreRun` could break the "never non-zero" contract
**What goes wrong:** A future refactor could add a `PersistentPreRunE` on root that does something like opening the index or checking for updates, and — because cobra runs a parent's `PersistentPreRunE` before every descendant's `RunE`, including a hidden one — that could make the hidden `hook pretooluse` command fail even though its own `RunE` never returns an error.
**Why it happens:** Cobra's execution model runs ancestor `PersistentPreRun(E)` hooks unconditionally unless a child overrides them.
**How to avoid:** Confirmed this session there is currently no `PersistentPreRun`/`PersistentPreRunE`/`cobra.OnInitialize` anywhere in `internal/cli` `[VERIFIED: rg search over internal/cli/*.go, zero matches, this session]`, so today nothing threatens D-01a. The plan should add a regression test asserting the hidden `hook pretooluse` command's tree has no inherited `PersistentPreRunE` that could return an error, so a future addition elsewhere in the tree cannot silently break this guarantee.
**Warning signs:** Any new `cmd.PersistentPreRunE` added to `root.go` or any ancestor of the `hook` command.

### Pitfall 2: Treating the guard's exec of the Go binary as "free" because of the 60s cooldown
**What goes wrong:** The 60-second cooldown lives inside the Go binary (D-05/D-08), not in the shell guard. In an indexed repo, the shell guard execs the Go binary on **every** matched Bash/Grep/Glob/Read call, cooldown or not — the binary is what decides to stay silent. If someone assumes the guard itself gates on time and therefore the binary "only runs once a minute," the real per-call cost is invisible until measured.
**Why it happens:** The cooldown language in D-05 ("fires at most once a minute") describes the *nudge emission*, not the *process invocation count*.
**How to avoid:** Measured this session on this repository's own build (`GOTOOLCHAIN=go1.26.6 go build ./cmd/codegraph`, scratch dir only): a warm-cache `codegraph --version` invocation takes ~9ms via bash's `time` builtin (15-run median), versus ~2-3ms for a bare `package main; func main(){}` Go binary built with the same toolchain `[VERIFIED: local build + `time` builtin measurement, this session, /private/tmp scratch dir]`. The ~6-7ms delta is attributable to CGo/tree-sitter-grammar package linkage and the cgo runtime's own startup cost (confirmed the actual grammar `Language()` calls are never invoked at `init()` — see Pitfall 3 — so the delta is link/runtime overhead, not grammar loading). At an assumed worst case of continuous grepping (D-05's own "~60 fires/hour of continuous searching" estimate implies far more matched calls than fires), a ~9ms-per-call tax is unlikely to be perceptible to a human but is a real, non-zero cost the D-18 "wall time per run" recording should capture precisely rather than estimating.
**Warning signs:** A future report of "Claude feels slower on grep-heavy tasks in indexed repos" should check this cost first, before suspecting the classify/sentinel logic.

### Pitfall 3: Believing package `init()` cost includes grammar loading
**What goes wrong:** It would be reasonable to assume the ~9ms startup cost above is dominated by loading 12+ tree-sitter grammars (a plausible, expensive-sounding operation), and therefore that trimming the indexer import graph out of the hook's build would meaningfully help.
**Why it happens:** `internal/indexer/languages_go.go`'s `init()` (and its 11 siblings) does look, at a glance, like it "sets up Go language support."
**How to avoid:** Read the actual code: `func init() { registerLanguage(LanguageSpec{ID: "go", ..., NewParser: func() (parser.Parser, error) { return cgo.NewGoParser() }, ...}) }` `[VERIFIED: internal/indexer/languages_go.go:22-38]`, and `registerLanguage`'s body is `func registerLanguage(spec LanguageSpec) { registry[spec.ID] = spec; for _, ext := range spec.Extensions { extToLang[ext] = spec.ID } }` `[VERIFIED: internal/indexer/languages.go:71-76]` — a plain map insert. `NewParser` is stored as an unevaluated closure; the actual `tree_sitter_go.Language()` CGo call inside `cgo.NewGoParser()` `[VERIFIED: internal/parser/cgo/parser_cgo.go:45-46]` never runs unless something calls the closure, which nothing on the hook's code path does. The measured ~6-7ms delta (Pitfall 2) is therefore CGo/cgo-runtime linkage overhead (thread pool setup, etc.), not grammar loading, and cannot be reduced by touching the language-registration code.
**Warning signs:** A profiling session that finds most of the ~9ms inside Go's cgo runtime setup rather than inside any `languages_*.go` file confirms this reading; if profiling instead shows time inside a specific `tree_sitter_*` call, that would falsify this finding and should be reported back to research, not silently "fixed."

### Pitfall 4: Extending the wrong existing test for the new nudge text (D-14)
**What goes wrong:** `TestNudgeTextNamesOnlyRealTools` and `TestNudgeTextCarriesNoUnpinnedFacts` both open `nudgeScriptPath = "../../.claude/hooks/session-nudge.sh"` as a **file** via `os.ReadFile` `[VERIFIED: internal/mcp/skill_claims_drift_test.go:113,807-815]` — that constant names the *SessionStart* script specifically. The new PreToolUse nudge text is a pinned **Go string constant** emitted as JSON by the hidden subcommand, not a file on disk anywhere Claude Code reads directly. Extending these tests by pointing `nudgeScriptPath` at a second file path would be wrong — the new text doesn't live in a file at all.
**Why it happens:** The two existing tests are named generically ("nudge text") but are hard-wired to one file.
**How to avoid:** Add sibling test functions (e.g. `TestPreToolUseNudgeTextNamesOnlyRealTools`) that call the same underlying helpers (`toolNameTokenRe`, `hostFactsIn`, `numericClaimsMultiset`, `docNamesCompanionsWithoutTheFilter` — all of which already take a `string doc` parameter, confirmed by their call sites reading `doc := string(data)` then passing `doc` `[VERIFIED: internal/mcp/skill_claims_drift_test.go:807-846]`) against the new Go constant's *value* directly, with no file read at all.
**Warning signs:** A test that "passes" but never actually exercises the new PreToolUse constant — check the mutation log's positive control specifically flips a character inside the new Go constant, not inside `session-nudge.sh`.

### Pitfall 5: Assuming `ConfiguredSkillLocations` tells you whether the opt-in is on
**What goes wrong:** `agents.ConfiguredSkillLocations(agents.Claude)` `[VERIFIED: internal/agents/manifest.go:277-302]` answers "does a manifest exist here naming Claude as a requester" — it says nothing about which `Files` keys that manifest carries. Code that gates the PreToolUse refresh on `len(locs) == 0` alone (mirroring `refreshInstalledSkills`'s existing shape) would refresh the guard script at every previously-installed location regardless of whether that location ever opted into the nudge.
**Why it happens:** `refreshInstalledSkills`'s existing loop `[VERIFIED: internal/cli/upgrade.go:48-73]` already has exactly this shape for the SessionStart/skill refresh, which is unconditional (every Claude install always gets SessionStart) — copying that shape verbatim for a conditional (opt-in) artifact is the natural but wrong generalization.
**How to avoid:** A new manifest-reading helper is needed (see Architecture Pattern 2) that checks `Files[manifestKeyGuardScript]` presence per location, not just manifest existence.
**Warning signs:** A live test where a location with `--pretool-nudge` never set nonetheless gains a PreToolUse hook entry after `codegraph upgrade`.

### Pitfall 6: Two-level hidden command registration gaps
**What goes wrong:** `codegraph hook pretooluse` is a two-level command (`hook` parent, `pretooluse` child). If only the child is marked `Hidden: true` and the parent `hook` command is left visible (or vice versa), the command could show up in `--help`'s command list even though its own entry is hidden, or the CLI-reference allowlist's ancestor-walk (`documentedByReference` walks "up to" every ancestor `[VERIFIED: internal/cli/cli_reference_test.go:50-57 header comment]`) could disagree with what `cobra --help` actually renders.
**Why it happens:** `man.go`'s and `renamed.go`'s hidden-command precedents are both single-level (`newManCmd()`, `newQueryCmd()` attach directly to root); this phase's `hook pretooluse` shape has no exact precedent in this codebase to copy.
**How to avoid:** Mark **both** the `hook` parent command and the `pretooluse` child `Hidden: true`; add exactly one allowlist line `codegraph hook pretooluse\t<reason>` (a bare two-segment path, following the `codegraph man`/`codegraph query` single-segment precedent's shape scaled up one level) `[CITED: internal/cli/testdata/cli-reference-allowlist.txt:1-9, this session]`.
**Warning signs:** `codegraph --help` or `codegraph hook --help` showing anything about the pretooluse nudge; `TestEveryRegisteredFlagIsAccountedFor`-style drift failing on an unaccounted flag under `hook`.

### Pitfall 7: Cross-ecosystem confusion — N/A this phase, noted for completeness
This phase adds no new package dependency in any ecosystem, so the Package Legitimacy Gate's cross-ecosystem-confusion concern does not apply. Recorded here only to make the omission of the Package Legitimacy Audit's rows an explicit, checked decision rather than a silent gap.

## Code Examples

### Hidden two-level subcommand registration (mirrors `man.go`/`renamed.go`)
```go
// Source: this repository's own internal/cli/man.go:46-67 and renamed.go:59-87 pattern, adapted
func newHookCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:    "hook",
        Hidden: true, // D-01a: never in --help, man pages, or completions
    }
    cmd.AddCommand(newHookPreToolUseCmd())
    return cmd
}

func newHookPreToolUseCmd() *cobra.Command {
    return &cobra.Command{
        Use:    "pretooluse",
        Hidden: true,
        Args:   cobra.NoArgs,
        RunE: func(cmd *cobra.Command, args []string) error {
            defer func() {
                _ = recover() // D-01a: never let a panic escape to main.go's exit-1 path
            }()
            runHookPreToolUse(cmd.InOrStdin(), cmd.OutOrStdout())
            return nil // D-01a: NEVER return a non-nil error to cobra
        },
    }
}
```
Register in `root.go`'s existing `root.AddCommand(...)` list `[VERIFIED: internal/cli/root.go:124-130]` alongside `newManCmd()`; `hook` is deliberately absent from `commandGroups` (like `man`, `query`, `unlock` — `[VERIFIED: internal/cli/root.go:63-89 comment, "man, query and unlock are deliberately absent"]`).

### One allowlist line (D-01a)
```
# Source: internal/cli/testdata/cli-reference-allowlist.txt existing format
codegraph hook pretooluse	hidden Claude Code PreToolUse nudge subcommand (NUDGE-03..06); reached only through the embedded sh guard, never invoked directly by a human
```

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `text/template` (vs. a plain `strings.Replace`) is the recommended mechanism for ExecPath substitution | Architecture Patterns, Pattern 1 | Low — this is Claude's Discretion per CONTEXT.md; either choice is stdlib-only and functionally equivalent for one substitution. Flagged only so the planner doesn't treat it as a locked decision. |
| A2 | Recommending `os.OpenFile(..., O_NOFOLLOW, ...)` + content-write over `os.Lstat`-then-`os.Chtimes` for sentinel recording | Architecture Patterns, Pattern 3; Common Pitfalls | Medium — this is a security-relevant recommendation (TOCTOU-through-symlink) grounded in `os.Chtimes`'s documented path-based (not fd-based) semantics `[CITED: pkg.go.dev/os#Chtimes]`, but the actual race window and its practical exploitability in `os.TempDir()`'s per-uid-scoped `codegraph-nudge-<uid>/` directory has not been demonstrated by a live exploit attempt this session — D-08 already asks for "refuse a symlinked sentinel," so this recommendation is the mechanism that satisfies that ask, not a new requirement. |
| A3 | An older (pre-Phase-6) binary's `Install()` call would silently drop the new manifest Files keys on that machine if it ever re-runs `install`/`upgrade` after this phase ships | Architecture Patterns, Pattern 2 | Low — this is a downgrade scenario (running an older binary against a newer manifest) not raised by any locked decision or requirement; flagged for completeness, not as a blocking finding. |
| A4 | Naming the new guard script `pretooluse-nudge.sh` and its manifest key `hooks/pretooluse-nudge.sh` | Recommended Project Structure, Pattern 2 | None — explicitly Claude's Discretion per CONTEXT.md ("The guard script name... within D-08... How the binary path reaches the guard"). |

**If this table is empty:** N/A — see rows above; none of them contest a locked D-xx decision, all are either explicitly-delegated discretion points or low-risk completeness notes.

## Open Questions

1. **Does `hookSpecificOutput.additionalContext`'s 10,000-character cap ever matter for a one-line pinned constant?**
   - What we know: the docs state the cap `[CITED: code.claude.com/docs/en/hooks, per CONTEXT.md's already-completed docs pass, 2026-09-19]`; D-14's nudge is "a new factual one-liner."
   - What's unclear: nothing — a one-liner cannot plausibly approach 10,000 characters. Recorded only because D-18's live-check protocol should not need to test this dimension; the plan should not add a test for it.
   - Recommendation: no action needed; do not add a "text under 10k chars" test, it would assert nothing meaningful.

2. **Exact byte contract between the guard's `exec` and the Go binary's stdin consumption when the binary panics mid-read**
   - What we know: D-16 requires "the binary exiting non-zero or crashing (the guard still exits 0)" to be guard-level tested; the Go core recovers panics (D-01a).
   - What's unclear: whether a panic during JSON decode (before `recover()`'s deferred call would run, if placed incorrectly) could still propagate as a process crash with a non-zero exit — this depends entirely on where the plan places the `defer recover()` relative to the stdin-read step, not on anything external to this repo.
   - Recommendation: the plan's task-level acceptance criteria should require the `defer recover()` to wrap the entire `RunE` body, including stdin decode, not just the classify/emit logic — the code example above already reflects this ordering.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain 1.26.6 | Building/testing this phase's code | ✓ | `go1.26.6` via `GOTOOLCHAIN=go1.26.6` `[VERIFIED: successful build this session]` | — |
| `syscall.O_NOFOLLOW` on darwin | Sentinel write safety (D-08) | ✓ | `0x100` `[VERIFIED: go doc, this session]` | — |
| `syscall.O_NOFOLLOW` on linux/amd64 | Sentinel write safety (D-08), cross-compiled | ✓ | `0x20000` `[VERIFIED: go doc, this session]` | — |
| CGo C toolchain | Building `codegraph` at all (pre-existing project constraint, not new to this phase) | ✓ | build succeeded this session | — |
| Herdr | D-18's live-check orchestration | Not probed this session (orchestrator-level tool, outside this research's build/test scope) | — | — |

**Missing dependencies with no fallback:** none identified.
**Missing dependencies with fallback:** none identified; Herdr availability is the orchestrator's concern at D-18 execution time, not a phase-planning blocker.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib), `go test` |
| Config file | none — `go.mod` pins the toolchain (`GOTOOLCHAIN=go1.26.6` required per project-wide carry-over note) |
| Quick run command | `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/... ./internal/cli/... ./internal/nudge/... ./internal/mcp/...` |
| Full suite command | `task test:unit` (excludes `internal/daemon`, run separately per project-wide rule — this phase touches none of `internal/daemon`) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NUDGE-03 | Emits only `additionalContext`, exits 0, never `permissionDecision` | unit (subcommand-level, D-16) | `go test ./internal/cli/... -run TestHookPreToolUse` | ❌ Wave 0 |
| NUDGE-03 | Guard never exits non-zero even on binary crash | unit (guard-level exec harness, D-16, mirrors `runSessionNudge`) | `go test ./internal/agents/... -run TestPreToolUseGuard` | ❌ Wave 0 (mirrors existing `runSessionNudge` at `internal/agents/hookpackage_test.go:64-110`) |
| NUDGE-04 | Fires first call, then ≤1/60s per (session,agent) key | unit (D-17, injected clock, `os.Chtimes`-planted mtimes) | `go test ./internal/nudge/... -run TestCooldown` | ❌ Wave 0 |
| NUDGE-04 | Silent, zero-overhead when un-indexed | unit (guard-level, D-04) | `go test ./internal/agents/... -run TestPreToolUseGuard/no_codegraph_dir` | ❌ Wave 0 |
| NUDGE-05 | True/false-positive corpus classification | unit (D-15 table test) | `go test ./internal/nudge/... -run TestClassify` | ❌ Wave 0 |
| NUDGE-05 | Fire rate measured in a genuinely fresh live session | manual-only (D-18) | Herdr-driven `claude --debug-file` transcript, human-reviewed against D-18's 7-point PASS bar | N/A — manual by design |
| NUDGE-06 | install/uninstall register/remove via exact-identity `writeHookEntry`/`removeHookEntry`; hand-edit duplicates | unit (extends `TestOwnershipExactIdentity`, D-11) | `go test ./internal/agents/... -run TestOwnershipExactIdentity` | ✅ exists (extend, don't replace) |
| NUDGE-06 | Sticky opt-in survives `upgrade`; explicit `--pretool-nudge=false` removes it | unit (new, mirrors `upgrade_test.go`'s existing `refreshInstalledSkills` coverage shape) | `go test ./internal/cli/... -run TestUpgradeCarriesPreToolNudge` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** targeted `go test ./internal/nudge/... ./internal/agents/... ./internal/cli/...` (quick run command above)
- **Per wave merge:** `task test:unit`
- **Phase gate:** Full suite green before `/gsd-verify-work`; D-18's live-check is a separate, human-executed gate per its own locked pass bar — it is not part of the automated suite and cannot be.

### Wave 0 Gaps
- [ ] `internal/nudge/classify.go` + `classify_test.go` + `testdata/corpus.json` — the new harness-neutral pure core (D-15)
- [ ] `internal/nudge/cooldown.go` + `cooldown_test.go` — sentinel/cooldown logic with injectable clock (D-05/D-08/D-17)
- [ ] `internal/nudge/text.go` — the pinned nudge-text constant (D-14), plus its sibling drift-guard tests in `internal/mcp/skill_claims_drift_test.go`
- [ ] `internal/cli/hook.go` + `hook_pretooluse.go` + tests — the hidden subcommand and its stdin/stdout envelope adapter
- [ ] `.claude/hooks/pretooluse-nudge.sh.tmpl` + `claudeassets.go`'s new `//go:embed` line + accessor
- [ ] `internal/agents/claude.go`'s `claudePreToolUseBlocks(loc)`, `claudeHooksGuardScriptPath(loc)`, template-render helper
- [ ] `internal/agents/manifest.go`'s two new key consts + a `PreToolNudgeConfigured(loc)`-shaped exported helper
- [ ] `internal/cli/testdata/cli-reference-allowlist.txt`'s one new line
- [ ] `06-MUTATION-LOG.md` positive controls for every new guard (D-00, D-16)

*(Framework install: none — `go test` is already fully configured in this repository.)*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | This hook has no auth surface — it is a local, unauthenticated child process of the user's own Claude Code session |
| V3 Session Management | No | The "session" concept here is Claude Code's own `session_id`, treated purely as an opaque cache key, never as an authentication or authorization token |
| V4 Access Control | Partial | Sentinel-file ownership check (`Stat_t.Uid` match) is the one access-control-adjacent control — it prevents one local user's hook process from reading/clobbering another local user's sentinel in a shared `os.TempDir()` |
| V5 Input Validation | Yes | Claude's stdin JSON is untrusted input from the process's perspective (D-16's "malformed JSON, oversized input" cases); `encoding/json.Decoder` over an `io.LimitReader`-bounded stream is the standard control, matching this repo's existing `readJSONFileStrict` posture of "malformed input never proceeds as if it were valid" |
| V6 Cryptography | No | No cryptographic operation in this phase — the manifest's sha256 hash (pre-existing, `hashContent`) is a drift signal only, explicitly documented as "NOT a tamper-detection or authenticity control" `[VERIFIED: internal/agents/manifest.go:9-15 package doc comment]`, and is unchanged by this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Symlink race on a shared-tempdir sentinel file (a local attacker on a multi-user host pre-creates `codegraph-nudge-<uid>/<key>` as a symlink to a victim file before the real uid-owned directory exists) | Tampering | `os.Lstat` + uid check before any write; `O_NOFOLLOW` on the actual open call so the open itself fails atomically if a symlink is substituted mid-race (D-08; see Architecture Pattern 3) |
| Oversized/malformed stdin as a local resource-exhaustion vector (Claude Code itself is trusted, but D-16 explicitly requires testing "oversized input" as a forced-error case) | Denial of Service | `io.LimitReader` bounding the JSON decode; malformed JSON treated as "stay silent, exit 0" rather than retried or logged verbosely |
| A hand-edited `settings.json` hook entry that shares this repo's exact command string, attempting to make an attacker-controlled block "adopt" ownership | Spoofing / Tampering | Exact-identity ownership (`242ec0a`) already defends this — ownership is proven by exact command-string match, never matcher/shape, and this phase's `writeHookEntry`/`removeHookEntry` calls inherit that property unmodified from `internal/agents/shared.go` |
| Guard script executing an `ExecPath` that has been swapped for a malicious binary between install-time and hook-fire-time (TOCTOU on the binary itself, not the sentinel) | Tampering | Out of this phase's threat model — this is the same trust boundary every other MCP/CLI entry this repo writes already accepts (`stdioMcpEntry(opts.ExecPath, ...)` has identical exposure for the MCP server entry, pre-existing and unchanged) |

## Sources

### Primary (HIGH confidence — code read directly, this session)
- `cmd/codegraph/main.go` (full file) — no PersistentPreRun, single error→exit1 path
- `internal/cli/root.go` (full file) — no PersistentPreRun/OnInitialize; hidden-command grouping convention
- `internal/cli/man.go`, `internal/cli/renamed.go` (full files) — hidden single-level command precedent
- `internal/indexer/languages_go.go:1-40`, `internal/indexer/languages.go:60-90` — init()/registerLanguage cost
- `internal/parser/cgo/parser_cgo.go:1-140` (grep) — lazy `Language()` construction
- `claudeassets.go` (full file), `.claude/hooks/session-nudge.sh` (full file) — existing embedded-asset pattern
- `internal/agents/shared.go:140-489` — `writeHookEntry`, `writeEmbeddedFile`, `removeHookEntry`
- `internal/agents/manifest.go` (full file) — schema history, `Files map[string]string`, `hashOwnedHookBlocks`
- `internal/agents/claude.go:1-660` — `claudeFragmentCommand`, `claudeHookCommand`, `claudeSessionStartBlocks`, `Install`, `Uninstall`
- `internal/agents/types.go:90-140` — `InstallOptions`
- `internal/cli/install.go:1-150` — `--auto-allow` flag wiring, non-sticky pattern
- `internal/cli/upgrade.go:1-80` — `refreshInstalledSkills`, explicit non-sticky `AutoAllow` comment
- `internal/agents/ownership_test.go:260-455` — `reproduce242ec0aPrecondition`, `assertOwnEntriesGoneAfterUninstall`
- `internal/mcp/skill_claims_drift_test.go:95-130,790-850` — existing nudge-text drift guards
- `internal/cli/testdata/cli-reference-allowlist.txt` (full file) — allowlist format precedent
- `go doc syscall.O_NOFOLLOW` / `go doc syscall.Stat_t` under `GOOS=darwin` (native) and `GOOS=linux GOARCH=amd64` (cross), `go1.26.6` toolchain — this session
- `go doc os.Chtimes` — this session
- Local build + timing measurement (`GOTOOLCHAIN=go1.26.6 go build ./cmd/codegraph`, bash `time` builtin, 15-run comparison against a bare `package main` binary) — this session, scratch dir only

### Secondary (MEDIUM confidence)
- `code.claude.com/docs/en/hooks` and related pages — already fetched and recorded verbatim in `06-CONTEXT.md`'s "Current hooks reference" block (2026-09-19, local `claude` 2.1.277); this research did not re-fetch, per the orchestrator's instruction to re-verify only what the plan depends on and CONTEXT does not already settle

### Tertiary (LOW confidence)
- None used — every claim in this document is either grounded in code read this session, a locked CONTEXT.md decision, or an empirical measurement taken this session.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependency; every mechanism is stdlib, confirmed present in the pinned Go toolchain this session
- Architecture: HIGH — every claim about existing mechanisms (`writeHookEntry`, `writeEmbeddedFile`, manifest schema, hidden-command precedent) is a direct code read with line citations, not an inference from CONTEXT.md's prose
- Pitfalls: HIGH for the startup-cost and init()-cost findings (both independently measured/read this session); MEDIUM for the sentinel-TOCTOU recommendation (grounded in documented stdlib semantics, not a demonstrated live exploit)

**Research date:** 2026-09-19
**Valid until:** 30 days (stable — no external API surface involved beyond Claude Code's already-pinned hooks reference in CONTEXT.md, which itself carries its own currency window)
