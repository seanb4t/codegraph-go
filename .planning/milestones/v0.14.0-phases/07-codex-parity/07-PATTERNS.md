# Phase 7: Codex Parity - Pattern Map

**Mapped:** 2026-09-19
**Files analyzed:** 23
**Analogs found:** 21 / 23 (2 are pure-fix "analog is itself" cases: `toml.go`'s own state machine, `agentpicker.go`/`daemonpicker.go`'s own one-line defect)

All analog paths below were verified git-tracked (`git ls-files`) before being cited — none are gitignored install/runtime mirrors.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/agents/codex.go` | config/target (installer) | CRUD (file read-modify-write) | `internal/agents/opencode.go` | exact (same dual-scope, dual-instructions, shared-skill-dir shape) |
| `internal/agents/toml.go` | utility (hand-rolled parser/serializer) | transform | itself (extend `findTOMLTableRange`'s state machine) | self — no other hand-rolled scanner of this shape exists; `opencode.go`'s hujson "surgical patch preserving structure" is the closest *idiom* precedent |
| `internal/agents/toml_test.go` | test | transform | itself + `internal/agents/ownership_test.go` (independent-oracle fixture style) | role-match |
| `internal/agents/codex_test.go` | test | CRUD | `internal/agents/opencode_test.go` (dual-scope target test suite) | exact |
| `internal/agents/codex_pretooluse.go` (NEW) | middleware/hook-registration helper | event-driven (config emission, not execution) | `internal/agents/claude_pretooluse.go` | exact |
| `internal/agents/ownership_test.go` | test | CRUD | itself (`ownershipWantSkillDir` oracle switch) | exact |
| `internal/agents/shared.go` (D-11 helper) | utility (shared cross-target helper) | CRUD | `internal/agents/capabilities.go`'s `declaredSkillFallback` (`AllTargets()` iteration + capability-table query) | role-match |
| `docs/AGENT-CAPABILITIES.md` (NEW) | doc | — | `docs/LANGUAGE-CAPABILITY-MATRIX.md` | exact (same doc/test pairing shape) |
| `internal/agents/capability_doc_test.go` (NEW) | test (doc-drift) | transform (parse doc, compare to code) | `internal/indexer/capability/matrix_test.go`'s `TestMatrix_DocMirrorsDescriptor` | exact |
| `internal/cli/install.go` (D-13 fix) | controller (CLI command) | request-response (flag resolution) | itself (case-order swap) | self |
| `internal/cli/uninstall.go` (D-13 fix) | controller (CLI command) | request-response | `internal/cli/install.go` (identical switch shape, sibling file) | exact |
| `internal/cli/install_test.go` (D-13 RED test + any new uninstall cases) | test | request-response | itself (`TestInstall_Yes_ShortCircuitsBeforeInteractiveBranch` / `TestUninstall_Yes_ShortCircuitsBeforeInteractiveBranch` — uninstall's tests already live in this file, there is no separate `uninstall_test.go`) | exact |
| `internal/cli/hook_pretooluse.go` (D-21 Codex envelope) | controller/event-handler | event-driven (stdin JSON in, stdout JSON out) | itself's existing `runHookPreToolUse` Claude envelope | exact (mirror the same function shape for a `--harness codex` branch) |
| `.codex/hooks/hooks.json` (NEW) | config (embedded asset) | — | `.claude/hooks/hooks.json` | exact |
| `.codex/hooks/codegraph-pretooluse.sh` (NEW) | config (embedded shell template) | — | `.claude/hooks/pretooluse-nudge.sh` | exact (adapt for D-22's no-`CLAUDE_PROJECT_DIR` constraint) |
| `codexassets.go` (NEW) | config (root-level `//go:embed` file) | — | `claudeassets.go` | exact |
| `internal/cli/tui/agentpicker.go` (D-24 fix) | component (TUI delegate) | transform (render) | itself (`checkboxDelegate.Render`, drop trailing `\n`) | self |
| `internal/cli/tui/daemonpicker.go` (D-24 identical fix) | component (TUI delegate) | transform (render) | `internal/cli/tui/agentpicker.go` (`checkboxDelegate.Render`'s sibling defect in `daemonDelegate.Render`) | exact |
| `internal/cli/tui/agentpicker_test.go` (D-25 model-level test) | test | transform | itself (existing test file; new `TestAgentPickerFootprintFitsDefaultPane`-shaped case) | exact |
| `test/tmux/install_cancel_test.go` (D-25 re-anchor) | test (real-PTY/tmux) | transform | itself | self |
| `internal/mcp/server.go` (D-29 instructions const) | config (const string) | — | itself (`instructions` const at `:57`) | self |
| `internal/mcp/instructions_contract_test.go` (D-29 WIRE-03 guard) | test | transform | itself | self |
| `testdata/wireoracle/transcripts/*.golden` (D-29 re-freeze) | test fixture | transform | itself (regenerated via the existing `-update`-flag mechanism, never hand-edited) | self |

## Pattern Assignments

### `internal/agents/codex.go` (config/target, CRUD)

**Analog:** `internal/agents/opencode.go`

**Imports pattern** (opencode.go lines 1-11):
```go
package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tailscale/hujson"
)
```
Codex's imports stay close to its current set (`fmt`, `os`, `path/filepath`) — no hujson needed (TOML, not JSONC).

**Core dual-scope Capabilities pattern** (opencode.go lines 45-54):
```go
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
D-09's target shape for `codexTarget.Capabilities()`: `Scopes: []Location{LocationGlobal, LocationLocal}`, `ConfigFormat: ConfigFormatTOML`, `Hooks: HooksCodexJSON`, `MCPConfig: codexConfigPath` (dropping the current `globalOnlyPath(...)` wrapper — see Pattern below), `Instructions: codexInstructionsPath`, `SkillDirs: sharedSkillDirs` (reused verbatim from `skillshared.go:60-69`, imported by `internal/agents/opencode.go:52` the same way).

**Dual-scope PathFunc pattern — the exact shape `codexConfigPath`/`codexInstructionsPath` must become** (opencode.go lines 75-108):
```go
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
	jsonc := filepath.Join(dir, "opencode.jsonc")
	if fileExists(jsonc) {
		return jsonc, nil
	}
	if jsonPath := filepath.Join(dir, "opencode.json"); fileExists(jsonPath) {
		return jsonPath, nil
	}
	return jsonc, nil
}

func opencodeInstructionsPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return "AGENTS.md", nil
	}
	cfgDir, err := resolveOpencodeConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "opencode", "AGENTS.md"), nil
}
```
Codex's config path is simpler — no candidate-file-exists fallback (Codex's config file is always `.codex/config.toml`, never `.json`):
```go
// current codex.go:47-53 (global-only, to be replaced)
func codexConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}
```
becomes a `PathFunc(Location) (string, error)`: `if loc == LocationLocal { return filepath.Join(".codex", "config.toml"), nil }` else the current global body. `codexInstructionsPath` gets the identical branch, but D-04's local target is the REPO ROOT `AGENTS.md` (shared with opencode — see Pattern 3 below), not `.codex/AGENTS.md`:
```go
func codexInstructionsPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return "AGENTS.md", nil // repo-root, shared with opencode (D-11)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "AGENTS.md"), nil
}
```

**Install/Uninstall delegating to shared helpers** (opencode.go lines 297-345 — Install/Uninstall bodies to mirror for the new local-scope branch; the current codex.go:102-167 `if loc != LocationGlobal { return result }` early-return goes away entirely per D-09):
```go
func (t opencodeTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult
	if configPath, err := opencodeConfigPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve opencode config path: %w", err))
	} else {
		fr, err := writeOpencodeEntry(configPath, opts.ExecPath)
		recordFile(&result, configPath, fr, err)
	}
	if instrPath, err := opencodeInstructionsPath(loc); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("resolve opencode instructions path: %w", err))
	} else {
		fr, err := upsertInstructionsEntry(instrPath, codegraphSectionStart, codegraphSectionEnd, instructionsBody())
		recordFile(&result, instrPath, fr, err)
	}
	installDeclaredSkill(&result, t, loc)
	return result
}
```
Codex's `Install` keeps its own TOML-splice call (`spliceTOMLTable`/`codexTableBody`, unchanged in shape) in place of `writeOpencodeEntry`, but gains the `installDeclaredSkill(&result, t, loc)` call (currently absent from codex.go entirely — D-14) and, for local scope only, an appended `result.Notes` trust-reminder line (D-10) and, when opted in, the hooks write (D-18, see `codex_pretooluse.go` below).

**Error handling pattern:** identical everywhere in this package — every resolve step wraps its error with `fmt.Errorf("resolve <target> <thing> path: %w", err)` and appends to `result.Errors`; every file write funnels through `recordFile(&result, path, fr, err)`. No target in this package uses panics or bare `log` calls for a resolution failure.

---

### `internal/agents/toml.go` (utility, transform) — D-07 state-machine rewrite

**Analog:** itself (the fix is a rewrite of `findTOMLTableRange`, not a copy from elsewhere in this repo)

**Current buggy scanner** (toml.go lines 77-103):
```go
func findTOMLTableRange(content, tableName string) (start, end int, found bool) {
	header := "[" + tableName + "]"
	lines := strings.Split(content, "\n")
	offset := 0
	headerLine := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == header {
			headerLine = i
			start = offset
			break
		}
		offset += len(line) + 1
	}
	if headerLine == -1 {
		return 0, 0, false
	}
	scanOffset := start + len(lines[headerLine]) + 1
	for i := headerLine + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "[") {  // <-- BUG: column-0 only
			return start, scanOffset, true
		}
		scanOffset += len(lines[i]) + 1
	}
	return start, len(content), true
}
```
D-07's fix requirements, layered onto this exact function: (1) match `strings.TrimSpace(lines[i])` starting with `[`, not `lines[i]` itself (indented headers); (2) track multi-line `"""`/`'''` string state and unbalanced `[`/`]` array-bracket depth, suppressing header detection while inside either; (3) treat a dotted subtable (`[mcp_servers.codegraph.foo]`) as still inside codegraph's own range by comparing the candidate header's dotted prefix against `tableName+"."`; (4) detect and return an error (not proceed) on an inline `codegraph = {` under `[mcp_servers]` or a bare `mcp_servers.codegraph.foo = ...` dotted-key line outside any header.

**Idiom precedent for "surgical patch preserving everything else":** `internal/agents/opencode.go`'s hujson `Parse -> Patch -> Format -> Pack` round-trip (lines 139-198) is the project's other example of "never re-serialize the whole file, touch exactly the owned span" — useful as the *design philosophy* reference even though the TOML file has no comparable library.

**`tomlString`/`tomlStringArray` escaping pattern to reuse unchanged** (toml.go lines 105-138) — no change needed here; D-07 does not touch table-body rendering.

---

### `internal/agents/toml_test.go` (test) — D-07 fixtures

**Analog:** `internal/agents/ownership_test.go`'s independent-oracle-table style (lines 46-81) — build fixtures as literal TOML strings with the exact real-world shape (2-space-indented `[mcp_servers.codegraph]`, a later column-0 `[memories]` header, per CONTEXT.md's reproduced-2026-09-19 finding), not the existing column-0-only fixtures. Required new cases per D-07/RESEARCH Wave-0-gap list: indentation, CRLF, inline-table-refusal, dotted-key-refusal, subtable-in-range, multi-line-string, multi-line-array, plus a planted-regression mutation (revert to column-0-only end scan) that must turn the new fixture RED.

---

### `internal/agents/codex_pretooluse.go` (NEW — middleware/hook-registration helper, event-driven)

**Analog:** `internal/agents/claude_pretooluse.go` (full file read, 152 lines — copy the shape wholesale)

**Guard-render token pattern** (claude_pretooluse.go lines 20-23, 131-151):
```go
const preToolGuardExecPathToken = "'@codegraph-exec-path@'"

func renderPreToolGuard(execPath string) (string, error) {
	if execPath == "" {
		return "", errors.New("render PreToolUse guard: empty binary path")
	}
	if !filepath.IsAbs(execPath) {
		return "", fmt.Errorf("render PreToolUse guard: binary path %q is not absolute", execPath)
	}
	data, err := claudeassets.PreToolUseGuardTemplate()
	if err != nil {
		return "", fmt.Errorf("render PreToolUse guard: %w", err)
	}
	tmpl := string(data)
	if n := strings.Count(tmpl, preToolGuardExecPathToken); n != 1 {
		return "", fmt.Errorf("render PreToolUse guard: template carries the binary-path token %d times, want 1", n)
	}
	return strings.Replace(tmpl, preToolGuardExecPathToken, shellSingleQuote(execPath), 1), nil
}
```
Codex's guard template needs its OWN embed source (`codexassets.PreToolUseGuardTemplate()` from the new `codexassets.go`) but can reuse the identical token constant and rendering function body (RESEARCH recommends factoring the render body to take `(templateBytes []byte, token string)` if shared — planner's call per D-00's "never invent shared structure for two things that only look similar" caution; a plain duplicate function is equally acceptable and lower-risk).

**Command-string resolution pattern (never bake the binary path into the registered command — D-19)** (claude_pretooluse.go lines 25-49):
```go
func claudePreToolGuardPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".claude", "hooks", "pretooluse-nudge.sh"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "hooks", "pretooluse-nudge.sh"), nil
}

func claudePreToolHookCommand(loc Location) (string, error) {
	if loc == LocationLocal {
		return claudePreToolFragmentCommand, nil
	}
	return claudePreToolGuardPath(LocationGlobal)
}
```
D-20's Codex command form differs from Claude's `${CLAUDE_PROJECT_DIR}/...` (Codex has no such env var): local is `"$(git rev-parse --show-toplevel)/.codex/hooks/codegraph-pretooluse.sh"` (shell-substitution form, not an env-var interpolation), global is the quoted absolute `$CODEX_HOME/hooks/codegraph-pretooluse.sh`.

**Block-building + ownership-identity pattern** (claude_pretooluse.go lines 51-64):
```go
func claudePreToolUseBlocks(loc Location) ([]any, []string, error) {
	ownCommand, err := claudePreToolHookCommand(loc)
	if err != nil {
		return nil, nil, err
	}
	blocks, err := claudeFragmentEventBlocks("PreToolUse", claudePreToolFragmentCommand, ownCommand)
	if err != nil {
		return nil, nil, err
	}
	return blocks, []string{ownCommand}, nil
}
```
The fragment-rewrite helper itself (`claudeFragmentEventBlocks`, `internal/agents/claude.go:288-...`) decode→find→rewrite-command pattern should be DUPLICATED for Codex's own `.codex/hooks/hooks.json` fragment, not shared — per RESEARCH's "State of the Art" table, Codex's schema carries extra fields (`statusMessage`, `additionalContextLimit`) Claude's doesn't use, so forcing one function over both risks D-00's anti-pattern.

**Guard path resolution (both scopes) pattern to mirror**, and the shell-quoting helper reused verbatim:
```go
// shellSingleQuote (claude_pretooluse.go:127-129) — reuse unchanged, no Codex-specific variant needed.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
```

---

### `.codex/hooks/hooks.json` (NEW) — config, embedded asset

**Analog:** `.claude/hooks/hooks.json` (full file, 90 lines)

**Full pattern to adapt** (`.claude/hooks/hooks.json` PreToolUse block, lines 23-34):
```json
{
  "matcher": "Bash",
  "hooks": [
    {
      "type": "command",
      "if": "Bash(grep *)",
      "command": "${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh",
      "timeout": 5
    }
  ]
}
```
D-20's Codex fragment is exactly ONE `PreToolUse` block (Codex has no per-handler `if`, so there is no grep/rg/find/Grep/Glob/Read fan-out the way Claude's fragment has six blocks — D-21 reuses only the shell rows of the corpora inside the Go adapter instead):
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "^Bash$",
        "hooks": [
          {
            "type": "command",
            "command": "$(git rev-parse --show-toplevel)/.codex/hooks/codegraph-pretooluse.sh",
            "timeout": <short, seconds — Rust struct field is `timeout_sec: u64`>
          }
        ]
      }
    ]
  }
}
```
Do not include an `"if"` field (Codex has none) and do not port Claude's SessionStart blocks — CODEX-05 is PreToolUse-only.

---

### `.codex/hooks/codegraph-pretooluse.sh` (NEW) — config, embedded shell template

**Analog:** `.claude/hooks/pretooluse-nudge.sh` (full file, 27 lines)

**Full guard pattern to adapt**:
```sh
#!/bin/sh
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
D-22's Codex local guard has no `CLAUDE_PROJECT_DIR` to read — it derives repo root from its own path two directories up (RESEARCH's illustrative shape, safe to adopt near-verbatim):
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
The global-scope variant checks `[ -d "${PWD}/.codegraph" ]` instead (D-22 — the hook's cwd is the session cwd, no path-derivation possible for a home-installed script). Both variants keep the exact `@codegraph-exec-path@` token and PATH-fallback `case` block byte-identical to Claude's, since `renderPreToolGuard`'s token-count-of-1 check is the same across both templates.

---

### `codexassets.go` (NEW) — config, root-level embed file

**Analog:** `claudeassets.go` (full file, 71 lines) — copy the package doc-comment rationale (must live at repo root, sibling-of-root `//go:embed` rule, `golang/go#46056`) verbatim, only the embedded paths and accessor names change:
```go
// Source: claudeassets.go:36-40, 59-70
//go:embed .claude/skills/codegraph/SKILL.md
//go:embed .claude/hooks/hooks.json
//go:embed .claude/hooks/session-nudge.sh
//go:embed .claude/hooks/pretooluse-nudge.sh
var FS embed.FS

func SkillMarkdown() ([]byte, error) { return FS.ReadFile(SkillMarkdownPath) }
func HooksFragment() ([]byte, error) { return FS.ReadFile(HooksFragmentPath) }
```
`codexassets.go` embeds exactly two files: `.codex/hooks/hooks.json` and `.codex/hooks/codegraph-pretooluse.sh` — Codex has no separate skill package (D-14: shared `.agents/skills` only, no `.codex/skills` write) and no SessionStart script (D-18: PreToolUse only), so this embed set is intentionally smaller than `claudeassets.go`'s four files. Named import will be needed if the package name differs from `codexassets` matching the directory convention — follow `claudeassets`'s exact named-import pattern (`claudeassets "github.com/seanb4t/codegraph-go"`) for consistency, e.g. `codexassets "github.com/seanb4t/codegraph-go"` is NOT possible (one package per directory) — this file lives in the SAME root package as `claudeassets.go`, so its embed directives and accessor functions are simply added to that same `package claudeassets` file set (either appended to `claudeassets.go` itself or a new root-level file in the same package, consistent with the sibling-of-root embed rule). Confirm which the planner prefers before implementation — RESEARCH's "Recommended Project Structure" names it as a separate `codexassets.go` file, which is fine as long as it declares `package claudeassets` (matching `claudeassets.go`'s existing package clause) rather than a new package name.

---

### `internal/cli/hook_pretooluse.go` (MODIFIED — D-21 Codex envelope, event-driven)

**Analog:** itself's existing Claude envelope (full file, 147 lines) — the Codex branch is a sibling function with the identical defensive posture, not a rewrite of the Claude path.

**Envelope shape to mirror** (hook_pretooluse.go lines 94-146):
```go
func runHookPreToolUse(in io.Reader, out io.Writer, getenv func(string) string) {
	data, err := io.ReadAll(io.LimitReader(in, maxHookStdinBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxHookStdinBytes {
		return
	}
	var event claudePreToolUseInput
	if err := json.Unmarshal(data, &event); err != nil {
		return
	}
	if event.HookEventName != "" && event.HookEventName != "PreToolUse" {
		return
	}
	var tool nudge.Tool
	var input string
	switch event.ToolName {
	case "Bash":
		tool, input = nudge.ToolShell, event.ToolInput.Command
	// ...
	default:
		return
	}
	if !hookQualifies(tool, input) {
		return
	}
	session := getenv("CLAUDE_CODE_SESSION_ID")
	if session == "" {
		session = event.SessionID
	}
	key, ok := nudge.SessionKey(session, event.AgentID)
	if !ok {
		return
	}
	if !(nudge.Gate{Dir: nudge.DefaultDir(), Now: hookNow}).Due(key) {
		return
	}
	line, err := json.Marshal(claudeHookOutput{HookSpecificOutput: claudeHookSpecificOutput{
		HookEventName:     "PreToolUse",
		AdditionalContext: nudge.Text,
	}})
	if err != nil {
		return
	}
	_, _ = out.Write(append(line, '\n'))
}
```
D-21's Codex adapter (`runHookPreToolUseCodex` or a `--harness` dispatch inside the same function — planner's naming choice) reuses `nudge.Qualifies`/`nudge.Gate`/`nudge.SessionKey`/`nudge.Text` UNCHANGED (harness-neutral core, per D-00) and needs its own input struct mapping Codex's documented stdin shape (`tool_name`, `tool_input.command` as string OR argv array — take the last argv element, "any doubt means silent" per D-21) and its own output struct (Codex's `hookSpecificOutput.additionalContext`, same field name per RESEARCH's verified schema). The `defer func() { _ = recover() }()` + never-exit-nonzero + never-write-to-stderr posture (hook_pretooluse.go lines 76-82) is inherited verbatim — this IS the V5 threat-model control cited in RESEARCH's Security Domain section.

**Cobra command registration pattern** (hook_pretooluse.go lines 71-83):
```go
func newHookPreToolUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "pretooluse",
		Short:  "Claude Code PreToolUse nudge (reads the hook event on stdin)",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			defer func() { _ = recover() }()
			runHookPreToolUse(cmd.InOrStdin(), cmd.OutOrStdout(), os.Getenv)
			return nil
		},
	}
}
```
D-21's `--harness codex` flag is the simplest way to keep ONE hidden subcommand (`codegraph hook pretooluse --harness codex`) dispatching to a Codex-specific body inside the same RunE, rather than a second hidden subcommand — matches the CONTEXT.md phrasing "the same hidden subcommand."

---

### `internal/agents/shared.go` (D-11 — new cross-target helper, CRUD)

**Analog:** `internal/agents/capabilities.go`'s `declaredSkillFallback` (lines 199-227) — the closest existing precedent for "iterate `AllTargets()`, query another target's `Capabilities()`, compare a resolved path":
```go
func declaredSkillFallback(dir string, loc Location, requester TargetID) ([]TargetID, error) {
	claudeDir, err := claudeSkillDirPath(loc)
	if err != nil {
		return nil, err
	}
	same, err := sameSkillDir(dir, claudeDir)
	if err != nil {
		return nil, err
	}
	if same {
		return []TargetID{Claude}, nil
	}
	return []TargetID{requester}, nil
}
```
D-11's new helper (RESEARCH's illustrative `instructionsStillNeededElsewhere`, exact name is the planner's choice) has the same shape but iterates the FULL registry rather than comparing against one fixed target:
```go
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
`AllTargets()` lives in `registry.go`, same package, already iterated the same way by `declaredSkillFallback`'s siblings — no import cycle risk (RESEARCH confirms this explicitly).

**Existing removal primitive this helper GATES, not replaces** (`removeMarkedSection`, shared.go lines 695-739) — both `codexTarget.Uninstall` and `opencodeTarget.Uninstall` must call `instructionsStillNeededElsewhere` BEFORE calling `removeMarkedSection`, reporting `ActionKept` (the same `FileAction` constant `writeSkillFile`'s foreign-dir branch already uses, `skillshared.go:114-117`, `ActionKeptForeign` — reuse the plain `ActionKept` here since this is not a foreign-content case, just a still-needed one) when it returns true.

**D-30 comment-only update location**: `shared.go:743` currently reads "for the 4 of 8 agent targets that get an instructions file (Claude, Codex, opencode, Gemini)" on `upsertInstructionsEntry`'s doc comment — update the comment text only, the function body is untouched.

---

### `docs/AGENT-CAPABILITIES.md` (NEW) + `internal/agents/capability_doc_test.go` (NEW) — D-27 drift test

**Analog:** `docs/LANGUAGE-CAPABILITY-MATRIX.md` + `internal/indexer/capability/matrix_test.go`'s `TestMatrix_DocMirrorsDescriptor` (full test, lines 108-165)

**Doc header/framing pattern to copy** (`docs/LANGUAGE-CAPABILITY-MATRIX.md` lines 1-11):
```markdown
# Language Capability Matrix

This is the D-11 language capability matrix (...)
— the human-readable half of the coverage contract. The machine-readable
half lives in `internal/indexer/capability/matrix.go`, and
`internal/indexer/capability/matrix_test.go` proves the two stay identical:
every coverage value below matches the Go descriptor exactly (...). If
this table and the Go descriptor ever drift, `go test ./internal/indexer/capability/...`
fails — this document cannot silently overclaim.
```
`docs/AGENT-CAPABILITIES.md` opens the same way, naming `internal/agents/capabilities.go`'s `Capabilities()` literals as the machine-readable half and `internal/agents/capability_doc_test.go` as the drift-proving test.

**Row-regex + comparison pattern to copy** (matrix_test.go lines 103-165):
```go
var mdTableRowPattern = regexp.MustCompile(`^\|\s*` + "`" + `(\w+)` + "`" + `\s*\|\s*(\w+)\s*\|\s*(\w+)\s*\|\s*(\w+)\s*\|\s*(\w+)\s*\|\s*$`)

func TestMatrix_DocMirrorsDescriptor(t *testing.T) {
	docPath := filepath.Join(repoRoot(t), "docs", "LANGUAGE-CAPABILITY-MATRIX.md")
	raw, err := os.ReadFile(docPath)
	// ... parse each `| `id` | col | col | col | col |` row via mdTableRowPattern,
	// compare each column against the Go descriptor's matching entry,
	// t.Errorf per mismatch, then verify every descriptor entry had a doc row
	// and vice versa.
}
```
D-27's table needs 7 code-derived columns per target×scope (scopes, MCP config format, instructions, skill, hooks, nudge — derived from `Capabilities()` for 8 targets × 2 scopes) plus a hand-kept, NOT drift-checked "verified" column (`verified <date> (<evidence file>)` or `[ASSUMED] (<doc URL>, fetched <date>)`) — the row-regex needs enough capture groups for the code-derived columns only; the verification column is read but never compared against anything (it has no code-side source of truth). The planted-mutation proof (`repoRoot(t)` helper reused verbatim, lines 17-28) is the same "flip one value, confirm RED" discipline `toml_test.go`'s planted regression and `agentpicker_test.go`'s D-25 test both need.

**`repoRoot(t)` helper to duplicate verbatim** (matrix_test.go lines 17-28):
```go
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolved repo root %q does not contain go.mod: %v", root, err)
	}
	return root
}
```
Adjust the `..`-count for `internal/agents/capability_doc_test.go`'s shallower package depth (two levels up from `internal/agents`, not three).

---

### `internal/cli/install.go` / `internal/cli/uninstall.go` (D-13 — CRUD, request-response)

**Analog:** itself — the bug and fix are both fully root-caused already (RESEARCH Pitfall 2)

**Current buggy switch order** (install.go, current `case yes:` precedes `case cmd.Flags().Changed("target")`):
```go
switch {
case yes:
	// D-15/Pitfall 6: --yes must short-circuit BEFORE the TTY
	// branch, not merely skip rendering the picker — checked
	// first in the switch so it always wins regardless of
	// stdin/stdout's actual state.
	targets, err = agents.ResolveTargetFlag("auto", loc)
case cmd.Flags().Changed("target"):
	targets, err = agents.ResolveTargetFlag(target, loc)
case interactiveAllowed(cmd):
	targets, err = runAgentPicker(cmd, loc)
default:
	targets, err = agents.ResolveTargetFlag("auto", loc)
}
```
D-13's fix swaps the first two cases — `--target` checked BEFORE `--yes` — in BOTH `install.go` and the structurally identical switch in `uninstall.go` (which uses `"all"` instead of `"auto"` for the `yes`/`default` cases). The `interactiveAllowed`/`default` cases and their comments are otherwise untouched.

---

### `internal/cli/install_test.go` (D-13 RED-first test)

**Analog:** itself — `TestInstall_Yes_ShortCircuitsBeforeInteractiveBranch` (lines 496-513) and `TestUninstall_Yes_ShortCircuitsBeforeInteractiveBranch` (lines 551-...) are the exact test shape to extend — uninstall's tests already live in THIS file (there is no separate `internal/cli/uninstall_test.go`):
```go
func TestInstall_Yes_ShortCircuitsBeforeInteractiveBranch(t *testing.T) {
	home := fakeHome(t)
	withStubbedPicker(t, func(*cobra.Command, agents.Location) ([]agents.AgentTarget, error) {
		t.Fatal("runAgentPicker must never be called when -y is set")
		return nil, nil
	})
	out, _, err := execCmd("install", "-y", "--location", "global")
	// ... assert auto-resolution occurred
}
```
D-13's new test (`TestInstall_TargetAndYes_ExplicitTargetWins` or similar) drives `execCmd("install", "--target", "codex", "-y", "--location", "global")` and asserts EXACTLY Codex was configured (not the auto-detected set) — using the same `fakeHome(t)`/`execCmd`/output-substring-assertion idiom every other test in this file already uses. Mirror for uninstall with `"--target", "codex", "--yes"`.

---

### `internal/cli/tui/agentpicker.go` / `daemonpicker.go` (D-24 — component, transform)

**Analog:** each other (identical defect, identical fix)

**The defect, both places** (agentpicker.go line 64):
```go
func (d *checkboxDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	// ...
	fmt.Fprintf(w, "%s%s %s\n", cursor, box, ai.target.DisplayName())  // <-- trailing \n is the bug
}
```
(daemonpicker.go line 59 has the identical `fmt.Fprintf(w, "%s%s (pid %d, up %s)\n", ...)` shape.) The fix drops the trailing `\n` from the format string in BOTH files — `bubbles/v2@v2.1.1`'s own `populatedView` (list.go:1220-1224) already inserts its own `Spacing()+1` newline separator between items, and `Height()=1`/`Spacing()=0` (agentpicker.go:34-35, daemonpicker.go's equivalent) already assumes the delegate contributes exactly one line.

---

### `internal/cli/tui/agentpicker_test.go` (D-25 — model-level height regression)

**Analog:** itself (existing file; add the new test alongside existing model-level tests) — RESEARCH's illustrative shape:
```go
func TestAgentPickerFootprintFitsDefaultPane(t *testing.T) {
	all := agents.AllTargets() // must be 8 (D-26)
	m := newAgentPickerModel(all, map[agents.TargetID]agents.DetectionResult{})
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m3 := m2.(agentPickerModel)
	view := m3.View()
	h := lipgloss.Height(view.Content)
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
`tea.View.Content` is a plain `string` field (verified `charm.land/bubbletea/v2@v2.0.8/tea.go`); `lipgloss.Height(string) int` is present at the pinned `charm.land/lipgloss/v2@v2.0.5`. `newAgentPickerModel` is the existing unexported constructor (`agentpicker.go:85`) — call it directly the same way `RunAgentPicker` does, no new test seam needed.

---

### `test/tmux/install_cancel_test.go` (D-25 — re-anchor TTY-05)

**Analog:** itself — the assertion currently anchors on the picker's title text ("Select agents to configure") specifically BECAUSE the footer overflowed off-screen before D-24's fix (its own comment says so). Once D-24 lands, re-anchor the assertion on the footer text `space: toggle` (from `agentpicker.go:142`'s `help` string) instead, per D-25.

---

### `internal/mcp/server.go` / `internal/mcp/instructions_contract_test.go` (D-29 — instructions const rewrite)

**Analog:** itself — the current const (server.go:57, 554 bytes) has a Claude-Code-scoped skill sentence ("in Claude Code, codegraph install also adds the codegraph skill"). D-29 makes this sentence harness-neutral and true for the 7 skill-receiving targets (all but Hermes), keeping the whole const ≤600 bytes with the skill sentence inside the first 512 bytes. Re-measure byte offsets AFTER editing (RESEARCH's Assumption A5 flags the pre-rewrite 554/490 figures as stale once this edit lands). The WIRE-03 guard in `instructions_contract_test.go` and the wire transcripts under `testdata/wireoracle/transcripts/*.golden` must be re-frozen in the SAME reviewed diff — reconcile the "38 vs 42 vs 24" transcript-count discrepancy RESEARCH's Open Question 3 flags before re-freezing (`grep -l '"instructions"' testdata/wireoracle/transcripts/*.golden | wc -l` for the actual current count).

---

## Shared Patterns

### File-write primitives (apply to every new/modified file in `internal/agents`)
**Source:** `internal/agents/shared.go`
**Apply to:** `codex.go`, `codex_pretooluse.go`, `shared.go`'s own D-11 helper
- `recordFile(result *WriteResult, path string, fr FileResult, err error)` (shared.go:18) — every file-write call site's result-recording idiom.
- `atomicWriteFile`/`atomicWriteExecutableFile` (shared.go:301-323, 767-768) — every disk write in this package funnels through these; never call `os.WriteFile` directly.
- `writeEmbeddedFile(path, content string, executable bool)` (shared.go:323-363) — the byte-identity-short-circuit idiom for any new embedded artifact (the Codex guard script, once rendered, is written through the SAME executable-artifact path Claude's guard uses).

### Hook JSON ownership (exact-command-identity rule, commit `242ec0a`)
**Source:** `internal/agents/shared.go:184-291` (`blockOwnsAnyCommand`, `commandIsOwned`, `writeHookEntry`), `shared.go:365-479` (`removeHookEntry`)
**Apply to:** `codex_pretooluse.go`'s hooks.json write/remove — reuse `writeHookEntry`/`removeHookEntry` UNCHANGED against `.codex/hooks.json`'s `PreToolUse` array; ownership is determined SOLELY by exact `command` string match, never by `matcher` value or block shape (this is the hardened rule a prior matcher-and-shape heuristic was reverted to reach — do not reintroduce shape-based matching for Codex).
```go
// shared.go:206-218 — the identity primitive both write and remove funnel through
func commandIsOwned(cmd string, ownCommands []string) bool {
	for _, own := range ownCommands {
		if cmd == own {
			return true
		}
	}
	return false
}
```

### Marker-fenced instructions (repo-root AGENTS.md, shared with opencode)
**Source:** `internal/agents/instructions.go` (marker constants + block body), `internal/agents/shared.go:647-754` (`replaceOrAppendMarkedSection`, `removeMarkedSection`, `upsertInstructionsEntry`)
**Apply to:** `codex.go`'s local-scope `Install`/`Uninstall`, and the new D-11 `instructionsStillNeededElsewhere` gate in `shared.go`
```go
// instructions.go:8-11 — the hard, stable marker contract, never altered
const (
	codegraphSectionStart = "<!-- CODEGRAPH_START -->"
	codegraphSectionEnd   = "<!-- CODEGRAPH_END -->"
)
```
`upsertInstructionsEntry`/`removeMarkedSection` are called identically by every instructions-writing target (`codex.go:127-132`, `159-164` today; `opencode.go:307-312`, `335-340`) — the only NEW logic this phase adds is the D-11 pre-check before `removeMarkedSection`'s call in both `codexTarget.Uninstall` and `opencodeTarget.Uninstall`.

### Skill-directory installation (shared `.agents/skills/codegraph`)
**Source:** `internal/agents/capabilities.go:229-275` (`installDeclaredSkill`/`uninstallDeclaredSkill`), `internal/agents/skillshared.go` (full file — the manifest-owned writer)
**Apply to:** `codex.go`'s `Install`/`Uninstall` (currently missing this call entirely — D-14 adds it), `internal/agents/ownership_test.go`'s `ownershipWantSkillDir` oracle (needs a `case Codex, Cursor, Opencode:` — Codex currently falls through to `default: return ""` at ownership_test.go:78-79)
```go
// capabilities.go:238-253 — the one call every skill-writing target's Install makes
func installDeclaredSkill(result *WriteResult, t AgentTarget, loc Location) {
	dir, err := t.Capabilities().WrittenSkillDir(loc)
	// ... declaredSkillFallback, installSkillPackageWithFallback(result, dir, loc, t.ID(), refuseUnmanifested, fallback)
}
```
No `.codex/skills` write is ever added — D-14 is explicit that this would double-list the skill in Codex's own catalog since Codex does not merge same-name skills.

### Notes channel (trust reminders, D-10/D-12/D-18/D-19)
**Source:** `internal/agents/types.go:96-99` (`WriteResult.Notes` field) — Kiro precedent already uses this field for advisory text
**Apply to:** every new Note this phase adds (local-install trust reminder, `AGENTS.override.md` shadow notice, hook-opt-out-on-disabled-feature notice, hook-trust-required notice) — append a plain string to `result.Notes`, exactly like `installSkillPackageWithFallback`'s existing D-17 advisory (`skillshared.go:268-271`):
```go
result.Notes = append(result.Notes, fmt.Sprintf(
	"%s is the same directory as Claude Code's %s — one skill package, one manifest listing every agent that installed it",
	dir, claudeDir))
```

### Nudge core (harness-neutral, UNCHANGED this phase)
**Source:** `internal/nudge/classify.go` (`Qualifies`), `internal/nudge/cooldown.go` (`CooldownWindow`, `SessionKey`, `Gate`), `internal/nudge/text.go` (`Text`)
**Apply to:** `internal/cli/hook_pretooluse.go`'s new Codex branch — call the SAME `nudge.Qualifies`/`nudge.Gate`/`nudge.SessionKey`/`nudge.Text` symbols the Claude envelope already calls (hook_pretooluse.go:107-141); do not fork or wrap this core for Codex.

### V5 input-validation posture (untrusted stdin JSON)
**Source:** `internal/cli/hook_pretooluse.go:14-16, 76-82, 95-98` (`maxHookStdinBytes`, `recover()`-wrapped RunE, size-capped tolerant read)
**Apply to:** the Codex envelope branch in the same file — inherit the identical defenses verbatim: `io.LimitReader(in, maxHookStdinBytes+1)`, silent-return on oversized/empty/malformed input, `defer func() { _ = recover() }()` at the RunE entry point, never write to stderr, never return a non-nil error.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `internal/agents/toml.go`'s state-machine rewrite (D-07) | utility | transform | No other file in the repo hand-rolls a multi-line-string/array-aware line scanner; the fix is internal to this file's own existing function, using RESEARCH's Code Examples section as the design reference instead of a codebase analog |
| `internal/cli/tui/agentpicker.go` / `daemonpicker.go` trailing-`\n` fix (D-24) | component | transform | The two files are each other's only analog (same defect, same one-line fix) — there is no third TUI delegate in this codebase to draw from |

## Metadata

**Analog search scope:** `internal/agents/`, `internal/cli/`, `internal/cli/tui/`, `internal/mcp/`, `internal/indexer/capability/`, `.claude/hooks/`, `docs/`, `test/tmux/` (all read directly this session; RESEARCH.md's own file-by-file citations were independently re-verified against tracked source rather than trusted verbatim)
**Files scanned:** 27 (every file RESEARCH.md's Sources section names as "read in full or in relevant part," re-confirmed tracked via `git ls-files`)
**Pattern extraction date:** 2026-09-19
