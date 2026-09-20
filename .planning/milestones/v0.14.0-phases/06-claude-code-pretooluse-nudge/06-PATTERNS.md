# Phase 6: Claude Code PreToolUse Nudge - Pattern Map

**Mapped:** 2026-09-19
**Files analyzed:** 21 (new + modified)
**Analogs found:** 17 / 21 (4 in "No Analog Found" — genuinely new mechanisms per RESEARCH.md; use its Architecture Patterns/Code Examples instead)

All analog paths below were verified git-tracked via `git ls-files` before being cited.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/nudge/classify.go` (new) | utility (pure transform) | transform | *(none — see No Analog Found)* | none |
| `internal/nudge/classify_test.go` (new) | test | transform | `internal/mcp/skill_claims_drift_test.go` (table-driven string-classification tests) | role-match |
| `internal/nudge/testdata/corpus.json` (new) | config (test fixture) | batch | *(none — new fixture shape)* | none |
| `internal/nudge/cooldown.go` (new) | utility (stateful file I/O) | file-I/O | `internal/daemon/lock.go` | exact (same problem shape: sentinel/lock file, staleness check, race-safe create) |
| `internal/nudge/cooldown_test.go` (new) | test | file-I/O | `internal/daemon/lock.go`'s own test suite pattern (not read; infer from lock.go's injectable-clock-free design) + `internal/agents/hookpackage_test.go` (`os.Chtimes`-style planted state, no `sleep`) | role-match |
| `internal/nudge/text.go` (new) | config (pinned constant) | — | `internal/agents/hookpackage_test.go:43` (`nudgeLine` constant) | exact |
| `internal/cli/hook.go` (new) — `newHookCmd()` | route (hidden CLI command, parent) | request-response | `internal/cli/man.go` | exact |
| `internal/cli/hook_pretooluse.go` (new) — `newHookPreToolUseCmd()` + RunE adapter | controller (stdin/stdout envelope adapter) | request-response | `internal/cli/renamed.go` (`newQueryCmd`/`newUnlockCmd` — hidden child command, `RunE` never surfaces to cobra's error path) + `internal/cli/affected.go:212-239` (stdin read pattern) + `internal/upgrade/upgrade.go:238-258` (`io.LimitReader` bounded read) | role-match (composite) |
| `internal/cli/hook_test.go` (new, implied by D-16) | test | request-response | `internal/agents/hookpackage_test.go` (`runSessionNudge` exec-harness shape, adapted to `exec.Command` on the built binary or direct `RunE` invocation) | role-match |
| `.claude/hooks/pretooluse-nudge.sh.tmpl` (new) | config (embedded shell template) | request-response | `.claude/hooks/session-nudge.sh` | role-match (new: templated, not literal) |
| `claudeassets.go` (modify — new `//go:embed` line + accessor) | config (embed accessor) | file-I/O | `claudeassets.go:33-58` (own prior lines for `hooks.json`/`session-nudge.sh`) | exact |
| `internal/agents/claude.go` (modify — `claudePreToolUseBlocks(loc)`, `claudeHooksGuardScriptPath(loc)`/reuse `claudeHooksScriptPath`, template-render helper) | service (install-time config builder) | CRUD | `internal/agents/claude.go:253-322` (`claudeHookCommand` + `claudeSessionStartBlocks`) | exact |
| `internal/agents/claude.go` (modify — `Install`/`Uninstall` wiring) | service | CRUD | `internal/agents/claude.go:436-583` (`Install`) and `:585-660` (`Uninstall`) | exact |
| `internal/agents/manifest.go` (modify — 2 new key consts + `PreToolNudgeConfigured(loc)`) | service (manifest reader/writer) | CRUD | `internal/agents/manifest.go:34-49` (key consts) + `:277-302` (`ConfiguredSkillLocations`) | exact |
| `internal/agents/types.go` (modify — `InstallOptions.PreToolNudge` field) | model (options struct) | — | `internal/agents/types.go:110-122` (`InstallOptions`) | exact |
| `internal/cli/install.go` (modify — `--pretool-nudge` flag) | route (CLI flag wiring) | request-response | `internal/cli/install.go:124-136` (`--auto-allow` flag + `InstallOptions{...}` construction) | exact |
| `internal/cli/upgrade.go` (modify — `refreshInstalledSkills` carries opt-in) | service | CRUD | `internal/cli/upgrade.go:30-73` (`refreshInstalledSkills`, `AutoAllow:false` comment) | exact |
| `internal/cli/testdata/cli-reference-allowlist.txt` (modify — 1 new line) | config (allowlist data) | batch | `internal/cli/testdata/cli-reference-allowlist.txt:7` (`codegraph man` line) | exact |
| `internal/agents/hookpackage_test.go` (modify — extend `TestHookRegistrationMatchesFragmentAndScript`, add `TestPreToolUseGuard*`) | test | file-I/O | `internal/agents/hookpackage_test.go:65-104` (`runSessionNudge`) and `:371-416` (`TestHookRegistrationMatchesFragmentAndScript`) | exact |
| `internal/agents/ownership_test.go` (modify — extend `TestOwnershipExactIdentity`, `reproduce242ec0aPrecondition`, `assertOwnEntriesGoneAfterUninstall`) | test | CRUD | `internal/agents/ownership_test.go:272-288` and `:379-459` | exact |
| `internal/mcp/skill_claims_drift_test.go` (modify — new sibling tests for the PreToolUse nudge constant) | test | transform | `internal/mcp/skill_claims_drift_test.go:105-113,801-860` (`nudgeScriptPath`-based tests) | role-match (source differs: file vs. Go constant — see Pitfall 4 note below) |

## Pattern Assignments

### `internal/nudge/cooldown.go` (utility, file-I/O)

**Analog:** `internal/daemon/lock.go` (git-tracked)

This is the strongest analog in the repo for D-08's sentinel mechanics: both are a per-key, filesystem-backed, race-aware state file with a staleness/age check and a "some failure states are normal, not errors" posture. Adapt, don't copy verbatim — `lock.go` guards process liveness via PID; `cooldown.go` guards elapsed wall-clock time via mtime/content-timestamp, and additionally needs the D-08 symlink/ownership refusal `lock.go` does not need (its lock lives under the project's own `.codegraph/`, not a shared `os.TempDir()`).

**Imports pattern** (`internal/daemon/lock.go:14-22`):
```go
import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)
```

**Race-safe file creation pattern** (`internal/daemon/lock.go:222-261`, `createLockExclusive`): stage a full payload into a uniquely-named temp file in the same directory, then `os.Link` it into place — `Link` is atomic w.r.t. both existence (EEXIST if destination exists) and content (never observable half-written). This is the mechanism to reuse for `codegraph-nudge-<uid>/` directory creation if a similar exclusive-create-without-partial-read guarantee is wanted; **however**, RESEARCH.md's Pattern 3 recommends a different concrete primitive for the sentinel's own record (`os.OpenFile(path, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)` + write to the already-open fd, `Truncate(0)` first) — D-08's symlink-refusal need is closer to that RESEARCH.md recipe than to `lock.go`'s `Link`-based approach, since `lock.go` never has to worry about an adversarial symlink swap (its directory is project-owned, not a shared multi-user tempdir). Use `lock.go`'s "read → check a staleness/age predicate → decide" control flow shape (`readLock` → `isStale` → act) as the analog for cooldown's own "read sentinel → check `time.Since(modTime) < 60s` → decide fire-or-silent" shape:
```go
// Source: internal/daemon/lock.go:51-66,82-92 (adapted shape)
func readLock(codegraphDir string) (info lockInfo, ok bool, err error) {
	data, err := os.ReadFile(lockPath(codegraphDir))
	if err != nil {
		if os.IsNotExist(err) {
			return lockInfo{}, false, nil
		}
		return lockInfo{}, false, err
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return lockInfo{}, false, fmt.Errorf("daemon: decoding lockfile %s: %w", lockPath(codegraphDir), err)
	}
	return info, true, nil
}

func isStale(info lockInfo) bool {
	if !isProcessLive(info.PID) {
		return true
	}
	// ... corroboration check
	return false
}
```

**Error handling pattern:** every I/O failure in `lock.go` (`readLock`, `acquire`, `Unlock`) is either folded into `ok=false, err=nil` for "absent, nothing to do" (D-08's "any failure means silent, exit 0" is the same posture, taken further — even a genuine I/O error must resolve to silence upstream, not a returned error that reaches the hook's stdout), or returned as a wrapped `error` the caller decides how to react to. `cooldown.go`'s caller (the hidden subcommand's core) must map every error path to "silent" per D-08 — do not propagate an error to `hook_pretooluse.go`'s stdout path.

**Testing pattern:** `lock.go`'s own test suite (not read directly, but implied by its doc comments referencing `TestAcquireConcurrentRaceOnlyOneWinner`) exercises concurrent racers against the same lock; D-17 requires `cooldown_test.go` to use an injected clock and `os.Chtimes`-planted mtimes instead of `sleep` — mirror `internal/agents/hookpackage_test.go`'s "concurrency and statelessness" subtest shape (`hookpackage_test.go:243-300`, `sync.WaitGroup` over N goroutines, capture results per-goroutine rather than calling `t.Fatalf` from a spawned goroutine) for the "parallel runs" case in D-16.

---

### `internal/nudge/classify.go` (utility, transform) — NO STRONG ANALOG

See "No Analog Found" below. Use RESEARCH.md's own Architecture Diagram and D-02/D-03 heuristic text as the spec; there is no existing pure-string-classifier function in this codebase to copy structurally. The closest *structural* precedent for "a pure function with no I/O, driven by a table test over `testdata/*.json` rows" is `internal/mcp/skill_claims_drift_test.go`'s checker functions (e.g. `countSkillWorkedExamples`, `docNamesCompanionsWithoutTheFilter`) — copy their shape (pure `func(doc string) T`, table-driven test with a `{name, body, want}` struct slice) even though their domain (markdown claim-checking) is unrelated:

**Core pattern** (`internal/mcp/skill_claims_drift_test.go:133-150` — shape only, not the domain logic):
```go
func countSkillWorkedExamples(body string) int {
	lines := strings.Split(body, "\n")
	start := -1
	for i, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), "## worked example") {
			start = i
			// ...
		}
	}
	// ...
}
```

**Table-test pattern** (`internal/mcp/skill_claims_drift_test.go:780-798`):
```go
cases := []struct {
	name string
	body string
	want int
}{
	{"heading outside the worked-examples section is not counted", "...", 1},
	{"a #### heading is not counted as ###", "...", 1},
}
for _, tc := range cases {
	t.Run(tc.name, func(t *testing.T) {
		got := countSkillWorkedExamples(tc.body)
		if got != tc.want {
			t.Errorf("countSkillWorkedExamples(%q) = %d, want %d", tc.body, got, tc.want)
		}
	})
}
```
D-15 asks for `testdata/corpus.json` rows (`{tool, input, want}`) instead of an inline Go slice — use `encoding/json` to decode the fixture, keeping the `t.Run(tc.name, ...)` per-row loop identical to the pattern above so failures name the offending row.

---

### `internal/nudge/text.go` (config, pinned constant)

**Analog:** `internal/agents/hookpackage_test.go:34-43` (`nudgeLine`), and `.claude/hooks/session-nudge.sh:13` (the SessionStart nudge's own pinned line, as prose inside the shell script rather than a Go constant — D-14 is explicit that the new text lives in Go, not a file, unlike this analog).

**Core pattern** (`internal/agents/hookpackage_test.go:43`):
```go
const nudgeLine = "This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y questions."
```
D-14's new constant follows the identical shape (one factual sentence, backtick-quoted CLI fallback, em dash, no imperative wording) but lives in `internal/nudge/text.go` as production code (not a test file), since the hidden subcommand emits it live. Name it clearly (e.g. `PreToolUseNudgeText`) and export it so `internal/cli/hook_pretooluse.go` and `internal/mcp/skill_claims_drift_test.go`'s new sibling tests can both reference the same symbol — avoids a second hand-typed copy of the string anywhere.

---

### `internal/cli/hook.go` + `internal/cli/hook_pretooluse.go` (route + controller, request-response)

**Analog:** `internal/cli/man.go` (hidden single-level command) and `internal/cli/renamed.go` (hidden two-level-shaped `RunE` that never lets an error surface unexpectedly — though `renamed.go`'s stubs deliberately DO return an error; D-01a's hidden command must do the opposite and NEVER return one).

**Imports pattern** (`internal/cli/man.go:1-9`):
```go
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)
```

**Hidden-command registration pattern** (`internal/cli/man.go:46-67`):
```go
func newManCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "man <dir>",
		Short:  "Generate man pages for the full command tree",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// ...
			return nil
		},
	}
}
```
Adapt to a **two-level** hidden tree (`hook` parent + `pretooluse` child, both `Hidden: true` — see RESEARCH.md Pitfall 6, no exact two-level hidden precedent exists in this repo).

**Registration site** (`internal/cli/root.go:124-130`):
```go
root.AddCommand(newInitCmd(), newIndexCmd(), newUninitCmd(),
	newQueryCmd(), newSearchCmd(), newCallersCmd(), newCalleesCmd(),
	newImpactCmd(), newAffectedCmd(), newFilesCmd(), newStatusCmd(),
	newNodeCmd(), newExploreCmd(), newServeCmd(), newSyncCmd(),
	newDaemonCmd(), newUnlockCmd(), newVersionCmd(), newTelemetryCmd(),
	newUpgradeCmd(), newInstallCmd(), newUninstallCmd(),
	newGithooksCmd(), newManCmd(), newUiCmd())
```
Add `newHookCmd()` to this list. It must stay absent from `commandGroups` (`root.go:63-89`), following the documented "man, query and unlock are deliberately absent" precedent (`root.go:59-61` comment) — hidden commands stay groupless.

**Error-never-surfaces pattern** (`internal/cli/renamed.go:59-87`, shape only — invert the outcome):
```go
func newQueryCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "query",
		Hidden:             true,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf(...)
		},
	}
}
```
`hook pretooluse`'s `RunE` must do the structural opposite of this: always `return nil`, with a `defer recover()` wrapping the entire body (RESEARCH.md's own Code Examples section already gives the exact skeleton — reuse it verbatim, it is this phase's own worked example, not a separate codebase analog).

**Stdin-read pattern** (`internal/cli/affected.go:212-239`, adapt from newline-delimited to JSON):
```go
scanner := bufio.NewScanner(cmd.InOrStdin())
// scanner.Buffer(...) caps the per-line/per-read size (WR-06 precedent)
```
Combine with the bounded-read precedent from `internal/upgrade/upgrade.go:251-258`:
```go
limited := io.LimitReader(resp.Body, maxReleaseAssetBytes+1)
data, err := io.ReadAll(limited)
if err != nil { /* ... */ }
if len(data) > maxReleaseAssetBytes {
	return nil, fmt.Errorf("download %s: exceeds %d byte limit", url, maxReleaseAssetBytes)
}
```
Adapt: `io.LimitReader(cmd.InOrStdin(), maxHookStdinBytes+1)` feeding an `encoding/json.Decoder`, so D-16's "oversized input" forced-error case has a concrete, already-proven-in-this-repo mechanism instead of a hand-rolled one (matches this repo's own "don't hand-roll" posture from RESEARCH.md's Don't Hand-Roll table).

**JSON strict-decode pattern** (`internal/agents/shared.go:156-172`, `readJSONFileStrict` — same "distinguish malformed from absent" posture, adapt from file to stdin reader):
```go
func readJSONFileStrict(path string) (map[string]any, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, false, nil
		}
		return nil, false, fmt.Errorf("could not read existing file: %w", err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, true, fmt.Errorf("%s: existing file is not valid JSON — fix or remove it manually: %w", path, err)
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, true, nil
}
```
D-01a's posture is stricter still: malformed/absent/oversized stdin must never even return an error the caller could mis-route — the hidden subcommand's adapter turns any of these into "stay silent, print nothing, `RunE` returns nil," never into a returned Go `error` that reaches `cmd/codegraph/main.go`.

---

### `.claude/hooks/pretooluse-nudge.sh.tmpl` (config, embedded shell template)

**Analog:** `.claude/hooks/session-nudge.sh` (git-tracked)

**Full existing script** (`.claude/hooks/session-nudge.sh`):
```sh
#!/bin/sh
if [ -d "${CLAUDE_PROJECT_DIR:-.}/.codegraph" ]; then
  printf '%s\n' 'This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y questions.'
fi
exit 0
```
This is the **stateless, literal** precedent — no per-machine value, never invokes the binary. The new guard diverges precisely at the two points D-01/D-01b call out: it must (a) branch on a `[ -x "$EXECPATH" ]` check for the templated `ExecPath` before (b) `exec`-ing the binary with stdin passed through, then (c) unconditionally `exit 0` regardless of the exec's own exit code — none of which the existing script needs, since it never launches a subprocess. Follow RESEARCH.md's Architecture Diagram (System Architecture Diagram section) for the exact 4-step guard body; there is no existing "guard that execs the codegraph binary" script in this repo to copy line-for-line, only the "one directory check, `exit 0`" opening this one already establishes.

---

### `claudeassets.go` (config, embed accessor)

**Analog:** `claudeassets.go:33-58` (this repository's own prior art, extended in place — not a separate file)

**Full existing embed block:**
```go
//go:embed .claude/skills/codegraph/SKILL.md
//go:embed .claude/hooks/hooks.json
//go:embed .claude/hooks/session-nudge.sh
var FS embed.FS

const (
	SkillMarkdownPath = ".claude/skills/codegraph/SKILL.md"
	HooksFragmentPath = ".claude/hooks/hooks.json"
	SessionNudgeScriptPath = ".claude/hooks/session-nudge.sh"
)

func SkillMarkdown() ([]byte, error) { return FS.ReadFile(SkillMarkdownPath) }
func HooksFragment() ([]byte, error) { return FS.ReadFile(HooksFragmentPath) }
func SessionNudgeScript() ([]byte, error) { return FS.ReadFile(SessionNudgeScriptPath) }
```
Add a fourth `//go:embed .claude/hooks/pretooluse-nudge.sh.tmpl` line plus a matching `PreToolUseGuardTemplatePath` const and `PreToolUseGuardTemplate() ([]byte, error)` accessor, following the exact three-part shape (embed directive, path const, accessor func) already used for all three existing files. Note the package doc comment's warning (`claudeassets.go:7-17`) about `//go:embed` patterns never containing `..` — the new template file must live under `.claude/hooks/` exactly like its siblings, for the same root-package-placement reason.

Also update `.claude/hooks/hooks.json` (the embedded fragment `claudeSessionStartBlocks` derives `SessionStart` from) with a parallel `PreToolUse` key if the plan follows the "derive from the embedded fragment" pattern for PreToolUse too (mirroring `claudeSessionStartBlocks`'s own derivation) — see the `internal/agents/claude.go` pattern assignment below for the exact function to mirror.

---

### `internal/agents/claude.go` (service, CRUD — helper functions)

**Analog:** `internal/agents/claude.go:190-322` (own prior art in the same file)

**Guard script path pattern** (`claude.go:190-202`, `claudeHooksScriptPath`):
```go
func claudeHooksScriptPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".claude", "hooks", "session-nudge.sh"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "hooks", "session-nudge.sh"), nil
}
```
A `claudeHooksGuardScriptPath(loc)` following this identical local/global branch shape resolves the new guard's install path (`.claude/hooks/pretooluse-nudge.sh` local, `~/.claude/hooks/pretooluse-nudge.sh` global) — reuse `claudeHooksScriptPath` itself if the plan chooses to keep both scripts in the same directory (recommended; no new directory-resolution logic needed).

**Command-string resolution pattern** (`claude.go:242-258`, `claudeHookCommand`):
```go
func claudeHookCommand(loc Location) (string, error) {
	if loc == LocationLocal {
		return claudeFragmentCommand, nil
	}
	return claudeHooksScriptPath(LocationGlobal)
}
```
D-01b requires the registered command to be the **script path**, never the binary's `ExecPath` — this existing function's shape (local: fixed literal; global: fully-resolved absolute path) is exactly right to copy for the PreToolUse guard's own command string, with its own fixed local literal (e.g. `${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh`) mirroring `claudeFragmentCommand` (`claude.go:151`).

**Fragment-derivation pattern** (`claude.go:260-322`, `claudeSessionStartBlocks` → mirror as `claudePreToolUseBlocks`):
```go
func claudeSessionStartBlocks(loc Location) ([]any, []string, error) {
	data, err := claudeassets.HooksFragment()
	// decode decoded.Hooks.SessionStart
	ownCommand, err := claudeHookCommand(loc)
	// rewrite every "command" field equal to claudeFragmentCommand into ownCommand
	return blocks, []string{ownCommand}, nil
}
```
Copy this shape exactly for `claudePreToolUseBlocks(loc)`, decoding a `PreToolUse` key from the same embedded `hooks.json` fragment instead of `SessionStart`, and rewriting each block's own literal guard-command placeholder into the resolved, per-location command. D-12 requires **one handler per matcher/`if` rule** (three Bash variants + Grep + Glob + Read) — the fragment's `PreToolUse` array will have more entries than `SessionStart`'s two, but the decode/rewrite loop shape is unchanged.

**Install wiring pattern** (`claude.go:525-555`, the SessionStart script-write + hook-entry steps inside `Install`):
```go
if scriptPath, err := claudeHooksScriptPath(loc); err != nil {
	// ...
} else {
	content, rerr := claudeassets.SessionNudgeScript()
	// ...
	fr, werr := writeEmbeddedFile(scriptPath, string(content), true)
	recordFile(&result, scriptPath, fr, werr)
}

if settingsPath, err := claudeSettingsPath(loc); err != nil {
	// ...
} else {
	blocks, ownCommands, berr := claudeSessionStartBlocks(loc)
	// ...
	fr, werr := writeHookEntry(settingsPath, "SessionStart", blocks, ownCommands)
	recordFile(&result, settingsPath, fr, werr)
}
```
The new PreToolUse steps mirror this pair exactly, but (a) gate both behind `opts.PreToolNudge` (mirroring the `if opts.AutoAllow { ... }` gate at `claude.go:466-473`), and (b) the script-write step must call the new `renderPreToolUseGuard(opts.ExecPath)` template-render helper (RESEARCH.md Pattern 1) before `writeEmbeddedFile`, since this is the first artifact whose content is not the raw embedded bytes.

**Uninstall wiring pattern** (`claude.go:640-657`):
```go
if scriptPath, err := claudeHooksScriptPath(loc); err != nil {
	// ...
} else {
	fr, rerr := removeEmbeddedFile(scriptPath)
	recordFile(&result, scriptPath, fr, rerr)
}

if settingsPath, err := claudeSettingsPath(loc); err != nil {
	// ...
} else {
	_, ownCommands, berr := claudeSessionStartBlocks(loc)
	// ...
	fr, werr := removeHookEntry(settingsPath, "SessionStart", ownCommands)
	recordFile(&result, settingsPath, fr, werr)
}
```
D-11 requires `uninstall` to **always** attempt these two removals regardless of whether the opt-in was ever set (reporting `ActionNotFound` when absent, which `removeHookEntry`/`removeEmbeddedFile` already do natively — no new gate needed here, unlike Install's gate).

---

### `internal/agents/manifest.go` (service, CRUD)

**Analog:** `internal/agents/manifest.go:34-49` (key consts) and `:277-302` (`ConfiguredSkillLocations`)

**Manifest key const pattern** (`manifest.go:34-49`):
```go
const (
	manifestKeySkillMD   = "skills/codegraph/SKILL.md"
	manifestKeyScript    = "hooks/session-nudge.sh"
	manifestKeyHooksFrag = "settings.json#hooks.SessionStart"
	manifestSchemaVersion = 2
)
```
Add two new keys following the identical naming convention (artifact identity, not absolute path) — e.g. `manifestKeyPreToolGuard = "hooks/pretooluse-nudge.sh"` and `manifestKeyPreToolFrag = "settings.json#hooks.PreToolUse"`. **No schema bump** — `Files` is already `map[string]string` (`manifest.go:62`); RESEARCH.md's Pattern 2 explicitly rejects a new struct field for this reason.

**Presence-check reader pattern** (`manifest.go:277-302`, `ConfiguredSkillLocations` — adapt from "any Files key" to "this specific Files key"):
```go
func ConfiguredSkillLocations(id TargetID) []Location {
	if id != Claude {
		return nil
	}
	var locs []Location
	for _, loc := range []Location{LocationGlobal, LocationLocal} {
		path, err := claudeManifestPath(loc)
		if err != nil {
			continue
		}
		m, present, rerr := readManifest(path)
		if rerr != nil {
			locs = append(locs, loc)
			continue
		}
		if !present {
			continue
		}
		if containsTarget(manifestRequesters(m, present, rerr, []TargetID{Claude}), id) {
			locs = append(locs, loc)
		}
	}
	return locs
}
```
`PreToolNudgeConfigured(loc Location) bool` is a narrower, single-location, single-key version of this same read-manifest-and-check shape (RESEARCH.md Pattern 2 gives the exact function body — reuse it verbatim, it is this phase's own worked design, grounded directly in this analog).

---

### `internal/agents/types.go` (model)

**Analog:** `internal/agents/types.go:110-122` (`InstallOptions`)

```go
type InstallOptions struct {
	AutoAllow bool
	ExecPath  string
}
```
Add `PreToolNudge bool` following the identical doc-comment style (state the Claude-only scope and the no-op-for-other-targets behavior, exactly as `AutoAllow`'s own comment does).

---

### `internal/cli/install.go` (route, request-response)

**Analog:** `internal/cli/install.go:124-136`

```go
opts := agents.InstallOptions{AutoAllow: autoAllow, ExecPath: execPath}
// ...
cmd.Flags().BoolVar(&autoAllow, "auto-allow", false, "also add mcp__codegraph__* to Claude Code's permissions.allow list")
```
Add a `pretoolNudge bool` local var, a matching `cmd.Flags().BoolVar(&pretoolNudge, "pretool-nudge", false, "...")` line immediately following the `--auto-allow` line (same flag-declaration block), and thread it into `InstallOptions{AutoAllow: autoAllow, ExecPath: execPath, PreToolNudge: pretoolNudge}`. D-09 additionally requires a `note:` print when the flag is given but Claude is not among the resolved `targets` — no existing precedent in `install.go` prints a flag-specific note today (this is new logic), but `printAgentResults`'s own `Notes` handling (`install.go:154-...`, prints `"  note: %s\n"` per `result.Notes` entry) is the exact analog for **how** a note reaches the user; the "flag given but target not resolved" check itself is new control flow at the call site, not inside `printAgentResults`.

---

### `internal/cli/upgrade.go` (service, CRUD)

**Analog:** `internal/cli/upgrade.go:30-73` (`refreshInstalledSkills`)

```go
func refreshInstalledSkills(execPath string, out io.Writer) error {
	locs := agents.ConfiguredSkillLocations(agents.Claude)
	if len(locs) == 0 {
		return nil
	}
	var errs []error
	for _, loc := range locs {
		targets, err := agents.ResolveTargetFlag(string(agents.Claude), loc)
		// ...
		for _, t := range targets {
			result := t.Install(loc, agents.InstallOptions{ExecPath: execPath, AutoAllow: false})
			// ...
		}
	}
	return errors.Join(errs...)
}
```
D-10 requires this call to change from a hardcoded `AutoAllow: false` to `agents.InstallOptions{ExecPath: execPath, AutoAllow: false, PreToolNudge: agents.PreToolNudgeConfigured(loc)}` — the **loop shape is unchanged**, only the options-construction line changes, reading the new manifest helper per-location instead of a constant. RESEARCH.md's Pitfall 5 explicitly warns against gating this on `len(locs) == 0` alone (that only proves Claude was configured *at all*, not that the PreToolUse opt-in specifically was set) — `PreToolNudgeConfigured(loc)` must be called per-location inside the loop, not hoisted above it.

---

### `internal/cli/testdata/cli-reference-allowlist.txt` (config, batch)

**Analog:** `internal/cli/testdata/cli-reference-allowlist.txt:7` (the `codegraph man` line)

```
codegraph man	hidden by v0.5.0 D-02 — the Homebrew cask post-install hook's man-page generator, never an interactive command; its only flag is cobra's --help (12-CONTEXT D-05)
```
Add one new line, same tab-separated `<command path>\t<reason>` shape (per the file's own header comment, lines 1-6), naming the full two-segment path per RESEARCH.md's Code Examples section:
```
codegraph hook pretooluse	hidden Claude Code PreToolUse nudge subcommand (NUDGE-03..06); reached only through the embedded sh guard, never invoked directly by a human
```

---

### `internal/agents/hookpackage_test.go` (test, file-I/O)

**Analog:** own file, `hookpackage_test.go:65-104` (`runSessionNudge`) and `:371-416` (`TestHookRegistrationMatchesFragmentAndScript`)

**Exec-harness pattern to mirror for a new `runPreToolUseGuard` helper** (`hookpackage_test.go:65-104`):
```go
func runSessionNudge(t *testing.T, dir string, useEnv bool) (stdout, stderr string, exitCode int, err error) {
	t.Helper()
	scriptPath, absErr := filepath.Abs(sessionNudgeScriptPath)
	// ...
	cmd := exec.Command(scriptPath)
	// env filtering, cmd.Dir vs env var branch
	// ...
	runErr := cmd.Run()
	// distinguish *exec.ExitError from a harness-level failure
}
```
The new guard's exec harness needs additional inputs this one does not: an `EXECPATH` env/template value, stdin content to pipe through (`cmd.Stdin`), and a way to substitute a fake/crashing "codegraph" binary for D-16's "binary exiting non-zero or crashing (the guard still exits 0)" case — extend this shape with a `cmd.Stdin = strings.NewReader(stdinJSON)` line and a `binaryPath` parameter pointing at a test-built stub executable.

**Registration-fragment-equality pattern to extend** (`hookpackage_test.go:340-416`, `sessionStartBlock` + `TestHookRegistrationMatchesFragmentAndScript`):
```go
func sessionStartBlock(t *testing.T, path string) any {
	// decode path's JSON, return hooks.SessionStart
}

func TestHookRegistrationMatchesFragmentAndScript(t *testing.T) {
	settingsBlock := sessionStartBlock(t, claudeSettingsFilePath)
	fragmentBlock := sessionStartBlock(t, claudeHooksFragmentPath)
	if !reflect.DeepEqual(settingsBlock, fragmentBlock) { /* ... */ }
	// walk entries, assert each command path resolves + is executable
}
```
D-13 requires this test to extend to `PreToolUse` — generalize `sessionStartBlock` to a `hookBlock(t, path, event string) any` (parameterize the hardcoded `"SessionStart"` key at `hookpackage_test.go:359`) and add a second assertion pass for `"PreToolUse"`, reusing the identical executable-file-exists loop.

---

### `internal/agents/ownership_test.go` (test, CRUD)

**Analog:** own file, `:272-288` (`reproduce242ec0aPrecondition`) and `:379-459` (`assertOwnEntriesGoneAfterUninstall`)

```go
func reproduce242ec0aPrecondition(t *testing.T, target AgentTarget, loc Location, opts InstallOptions) {
	pre := target.Install(loc, opts)
	// ...
	writeFile(t, settingsPath, `{"hooks":{"SessionStart":[{"matcher":"startup","hooks":[{"type":"command","command":"`+ownershipUnrelatedHookCommand+`"}]}]}}`)
}
```
D-11's requirement ("plants an unrelated PreToolUse block under the same matcher, which must survive byte-identical") needs a sibling precondition writing an unrelated `PreToolUse` block (same `ownershipUnrelatedHookCommand` constant, `ownership_test.go:181`, reused verbatim) — either extend this function to also plant a PreToolUse block, or add a twin function following its exact shape.

`assertOwnEntriesGoneAfterUninstall` (`:379-459`) currently checks only `"SessionStart"` (`:438-439`, `hooks["SessionStart"].([]any)`) — extend with an equivalent block for `"PreToolUse"`, checking codegraph's own PreToolUse command strings are gone while the planted unrelated block survives, mirroring lines 438-458's loop structure exactly.

Note `runOwnershipLeaf` (`:463-...`) currently constructs `opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}` (`:472`) with no `PreToolNudge` field — the opt-in must be explicitly set `true` on the Claude leaves that need to exercise the new hook (RESEARCH.md's Established Patterns note: "The ownership table ... uses `InstallOptions{ExecPath}` only" — this is the one place that must change to add `PreToolNudge: true` for Claude-target leaves).

---

### `internal/mcp/skill_claims_drift_test.go` (test, transform)

**Analog:** own file, `:105-113` (`nudgeScriptPath` const) and `:801-860` (`TestNudgeTextNamesOnlyRealTools`, `TestNudgeTextCarriesNoUnpinnedFacts`)

```go
const nudgeScriptPath = "../../.claude/hooks/session-nudge.sh"

func TestNudgeTextNamesOnlyRealTools(t *testing.T) {
	data, err := os.ReadFile(nudgeScriptPath)
	// ...
	doc := string(data)
	matches := toolNameTokenRe.FindAllString(doc, -1)
	// ... check every match against allToolNames()
	if err := docNamesCompanionsWithoutTheFilter(doc); err != nil { /* ... */ }
}

func TestNudgeTextCarriesNoUnpinnedFacts(t *testing.T) {
	data, err := os.ReadFile(nudgeScriptPath)
	doc := string(data)
	if found := hostFactsIn(doc); len(found) > 0 { /* ... */ }
	if claims := numericClaimsMultiset(doc); len(claims) != 0 { /* ... */ }
	if claims := countClaimsIn(doc); len(claims) != 0 { /* ... */ }
	for _, m := range envVarTokenRe.FindAllString(doc, -1) { /* ... */ }
}
```
**Pitfall (RESEARCH.md Pitfall 4, already flagged):** do NOT point a second `nudgeScriptPath`-style const at a new file — the PreToolUse nudge text is a Go string constant (`internal/nudge/text.go`, above), not a file. Write sibling functions (e.g. `TestPreToolUseNudgeTextNamesOnlyRealTools`) that call the exact same helpers (`toolNameTokenRe`, `hostFactsIn`, `numericClaimsMultiset`, `countClaimsIn`, `docNamesCompanionsWithoutTheFilter`, `envVarTokenRe`) directly against `nudge.PreToolUseNudgeText` (or whatever the constant is named) with **no `os.ReadFile` call at all** — every helper already takes a plain `string doc` parameter (confirmed at the call sites above), so this is a drop-in substitution of the `doc` variable's source, not a new helper.

## Shared Patterns

### Exact-identity hook ownership
**Source:** `internal/agents/shared.go:174-269` (`writeHookEntry`) and `:343-462` (`removeHookEntry`)
**Apply to:** `internal/agents/claude.go`'s new `Install`/`Uninstall` PreToolUse steps — call these two functions completely unmodified, passing `"PreToolUse"` as the `event` argument and the PreToolUse-specific `ownCommands` slice. No new hook-registration primitive is needed; this is the one mechanism NUDGE-06 and D-11 both depend on, already proven for SessionStart.
```go
fr, werr := writeHookEntry(settingsPath, "PreToolUse", blocks, ownCommands)
// ...
fr, werr := removeHookEntry(settingsPath, "PreToolUse", ownCommands)
```

### Embedded-file idempotent write with self-healing executable bit
**Source:** `internal/agents/shared.go:286-341` (`writeEmbeddedFile`)
**Apply to:** the new guard script's Install step. Confirmed via direct signature read that `writeEmbeddedFile(path, content string, executable bool) (FileResult, error)` takes a plain `content string` — a `text/template`-rendered string is a drop-in argument, no change to this function required.

### Manifest-recorded ownership (Files map, no schema bump)
**Source:** `internal/agents/manifest.go:34-49`, `:213-241` (`writeManifest`)
**Apply to:** `internal/agents/claude.go`'s `recordSkillManifest` call inside `Install` (`claude.go:569-580`) — add the two new keys (`manifestKeyPreToolGuard`, `manifestKeyPreToolFrag`) to the `map[string]string` literal passed there, gated the same way the script/hooks-frag keys already are (only recorded when the corresponding write actually succeeded — the `haveScriptContent`/`haveSessionStart`-style boolean-gate pattern at `claude.go:492-580`).

### Fail-loud strict JSON reads
**Source:** `internal/agents/shared.go:142-172` (`readJSONFileStrict`)
**Apply to:** any new settings.json read path this phase touches — already inherited for free by reusing `writeHookEntry`/`removeHookEntry` verbatim (both call `readJSONFileStrict` internally); no new call site needed unless `hook_pretooluse.go`'s stdin-JSON decode chooses to mirror this three-outcome (absent/malformed/present) shape for its own decoding, which RESEARCH.md's own recommendation (`encoding/json.Decoder` + `io.LimitReader`) already covers with a simpler two-outcome (malformed vs. valid) shape appropriate for a one-shot stdin read rather than a persisted file.

### Sentinel/lock file staleness check
**Source:** `internal/daemon/lock.go:51-92` (`readLock`, `isStale`)
**Apply to:** `internal/nudge/cooldown.go`'s core age check. See the dedicated Pattern Assignment above for the adaptation notes (mtime/mtime-vs-PID, `O_NOFOLLOW` vs `Link`-based exclusive create).

### Concurrency-safe goroutine test result collection
**Source:** `internal/agents/hookpackage_test.go:243-300` (`TestSessionNudgeOutputIsPinnedAndStateless`, "concurrency and statelessness" subtest)
**Apply to:** `internal/nudge/cooldown_test.go`'s D-16 "parallel runs" case and `internal/agents/hookpackage_test.go`'s new guard-level parallel-run coverage. Capture each goroutine's result into a pre-sized slice and report failures from the main test goroutine after `wg.Wait()` — never call `t.Fatalf` from inside a spawned goroutine (testing package's documented contract, cited explicitly in this analog's own doc comment at lines 58-64).

## No Analog Found

Files with no close structural match in this codebase (planner should use RESEARCH.md's Architecture Patterns / Code Examples sections instead, which already give concrete skeletons grounded in this session's own direct source reads):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/nudge/classify.go` | utility | transform | No existing pure string/JSON-input classifier function in this codebase — every prior "classify a string" precedent (`internal/mcp/skill_claims_drift_test.go`'s checkers) is test-only code checking documentation claims, not production classification logic. RESEARCH.md's Architecture Diagram + D-02/D-03 text is the spec; `skill_claims_drift_test.go`'s checker *shape* (pure `func(string) T`) is copyable structurally, cited above. |
| `internal/nudge/testdata/corpus.json` | config (fixture) | batch | No existing `testdata/*.json` corpus of `{tool, input, want}` rows in this repo — every existing `testdata/` directory in `internal/cli` holds CLI-reference golden text or the allowlist, not classification fixtures. New shape; follow D-15's own field names exactly. |
| `.claude/hooks/pretooluse-nudge.sh.tmpl` | config | request-response | First guard script requiring per-machine template substitution (RESEARCH.md Pattern 1's own framing: "no existing precedent to copy verbatim"). `session-nudge.sh` is the closest sibling for the *stateless directory-check* half only; the *exec-the-binary-with-templated-path* half is genuinely new — use RESEARCH.md's System Architecture Diagram for the 4-step guard body. |
| `internal/nudge/classify_test.go` `testdata`-driven harness wiring | test | transform | Table-driven-over-a-JSON-fixture (rather than an inline Go slice) test wiring has no exact precedent; nearest structural cousin is `skill_claims_drift_test.go`'s inline-slice table tests (cited above) — the JSON-fixture-loading step itself is new, but trivial (`encoding/json.Unmarshal` into a `[]struct{Tool, Input, Want string/bool}`). |

## Metadata

**Analog search scope:** `internal/agents/`, `internal/cli/`, `internal/daemon/`, `internal/mcp/`, `internal/upgrade/`, repository-root `claudeassets.go`, `.claude/hooks/`, `.claude/settings.json`
**Files scanned:** 19 read directly this session (all confirmed git-tracked via `git ls-files`), plus RESEARCH.md's own already-verified line citations reused where re-reading would have duplicated a range already recorded there
**Pattern extraction date:** 2026-09-19
