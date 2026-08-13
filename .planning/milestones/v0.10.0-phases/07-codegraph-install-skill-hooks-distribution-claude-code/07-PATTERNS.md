# Phase 7: `codegraph install` Skill + Hooks Distribution (Claude Code) - Pattern Map

**Mapped:** 2026-08-13
**Files analyzed:** 8 (2 new, 6 modified)
**Analogs found:** 8 / 8

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `claudeassets.go` (NEW, repo root) | config (embed source) | file-I/O | `internal/mcp/resources.go` | exact — same `//go:embed` + `embed.FS` shape, first and only other embed use in repo |
| `internal/agents/claude.go` (extend `Install`/`Uninstall`/`DescribePaths`) | controller (AgentTarget impl) | CRUD (file writes) | itself — extend in place | exact — same file, same function bodies |
| `internal/agents/shared.go` (new: `writeHookEntry`/`removeHookEntry`, strict JSON reader, manifest read/write/hash) | utility (shared write/merge helpers) | CRUD + transform | `writeMcpEntry`/`removeMcpEntry` in same file | exact — explicitly named precedent in CONTEXT.md D-05/D-06 |
| `internal/agents/shared.go` (new strict reader, e.g. `readJSONFileStrict`) | utility (validation/read) | file-I/O | `internal/githooks/githooks.go` `Install`'s read-switch (lines 256-297) | exact — CONTEXT.md/RESEARCH.md name this as the correct precedent, not `readJSONFile` |
| `internal/agents/shared.go` or new file (manifest hash helpers) | utility (content hashing) | transform | `internal/upgrade/upgrade.go:194` (`sha256.Sum256`) | exact — same hash algorithm/idiom |
| `internal/cli/upgrade.go` (extend `RunE` post-swap refresh) | controller (CLI command) | event-driven (post-swap trigger) | itself, plus `internal/cli/upgrade_test.go`'s `upgradeRunFunc` injectable-seam pattern | role-match — same file, new step appended after existing delegation |
| `internal/agents/claude_test.go` (new test cases) | test | request-response (assert file state) | `TestClaude_GlobalRoundTrip_ByteInvariantWithSibling`, `TestClaude_Uninstall_PreservesUnrelatedAllowEntries`, `TestClaude_Install_ReRunIsByteIdempotent` (same file) | exact |
| `internal/cli/upgrade_test.go` (new `TestUpgradeCommand_RefreshesConfiguredTargets`-shaped test) | test | request-response | `TestUpgradeCommand_DelegatesWithCheckAndVersion` (same file) | exact |

## Pattern Assignments

### `claudeassets.go` (NEW, repository root) — config/embed source

**Analog:** `internal/mcp/resources.go`

**Embed pattern** (lines 1-19):
```go
package mcp

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// resourcesFS embeds every fact-sheet/behavior-doc markdown file this
// server serves over resources/list and resources/read (D-07). This is
// the first //go:embed use in this repository.
//
//go:embed resources/*.md
var resourcesFS embed.FS
```

**What to copy:** the `//go:embed` directive + package-level `var FS embed.FS` shape. Per RESEARCH.md Pattern 1/Pitfall 2, the new file MUST live at the repo root (not `internal/agents/`, which cannot reach `.claude/` via `..`), and MUST name exact files, never a bare directory glob:
```go
// claudeassets.go — repository root
package claudeassets

import "embed"

//go:embed .claude/skills/codegraph/SKILL.md
//go:embed .claude/hooks/hooks.json
//go:embed .claude/hooks/session-nudge.sh
var FS embed.FS
```
`internal/agents` imports this new root package and reads via `FS.ReadFile(...)`, mirroring how `internal/mcp` reads `resourcesFS.ReadFile` elsewhere in `resources.go`.

---

### `internal/agents/claude.go` — extend `Install`/`Uninstall`/`DescribePaths`

**Analog:** itself (same file, same functions) — this is an in-place extension, not a new file.

**Location-aware path resolution pattern** (lines 76-117, to mirror for `claudeSkillDirPath(loc)`/`claudeHooksScriptPath(loc)`/`claudeManifestPath(loc)`):
```go
func claudeConfigPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return ".mcp.json", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude.json"), nil
}

func claudeSettingsPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return filepath.Join(".claude", "settings.json"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}
```
New helpers follow this exact global/local branch shape — e.g. skill dir is `.claude/skills/codegraph` (local) vs `~/.claude/skills/codegraph` (global); manifest path is `<skillDir>/.codegraph-manifest.json`.

**Install step-sequencing pattern** (lines 212-252) — every new step (skill dir write, script write+chmod, hooks merge, manifest write) slots in as one more `if path, err := resolvePath(loc); err != nil { ...append to result.Errors... } else { fr, err := doTheWrite(...); recordFile(&result, path, fr, err) }` block, in the same function body, after the existing MCP-entry/instructions/AutoAllow steps:
```go
func (claudeTarget) Install(loc Location, opts InstallOptions) WriteResult {
	var result WriteResult
	// ... existing MCP entry step ...
	// ... existing instructions step ...
	if opts.AutoAllow {
		if settingsPath, err := claudeSettingsPath(loc); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("resolve claude settings path: %w", err))
		} else {
			fr, err := addClaudeAllowPermission(settingsPath)
			recordFile(&result, settingsPath, fr, err)
		}
	}
	// NEW: skill dir write, script write+chmod, hooks merge, manifest write
	// go here, same shape.
	return result
}
```

**Uninstall mirror pattern** (lines 254-285) — same shape, using `recordFile`/`recordAction` for every removal step, in reverse-appropriate order.

**DescribePaths pattern** (lines 287-299) — append each new path the same way:
```go
func (claudeTarget) DescribePaths(loc Location) []string {
	var paths []string
	if p, err := claudeConfigPath(loc); err == nil {
		paths = append(paths, p)
	}
	// ... append skill dir, script path, manifest path the same way
	return paths
}
```

---

### `internal/agents/shared.go` — new `writeHookEntry`/`removeHookEntry` (array-scoped merge)

**Analog:** `writeMcpEntry`/`removeMcpEntry` (lines 130-193) — explicitly named precedent in CONTEXT.md D-05/D-06 and RESEARCH.md Pattern 2.

**Core CRUD/idempotency pattern to extend** (lines 130-165):
```go
func writeMcpEntry(path string, buildEntry func() any) (FileResult, error) {
	existing, err := readJSONFile(path)
	if err != nil {
		return FileResult{}, err
	}
	mcpServers, _ := existing["mcpServers"].(map[string]any)
	if mcpServers == nil {
		mcpServers = map[string]any{}
	}
	before := mcpServers["codegraph"]

	after, err := normalizeJSON(buildEntry())
	if err != nil {
		return FileResult{}, err
	}

	if jsonDeepEqual(before, after) {
		return FileResult{Path: path, Action: ActionUnchanged}, nil
	}

	action := ActionCreated
	if before != nil {
		action = ActionUpdated
	}
	mcpServers["codegraph"] = after
	existing["mcpServers"] = mcpServers
	if err := writeJSONFile(path, existing); err != nil {
		return FileResult{}, err
	}
	return FileResult{Path: path, Action: action}, nil
}
```
**Divergence required (per RESEARCH.md Pattern 2/Pitfall 1):** `mcpServers.codegraph` is a single named map key; `hooks.<Event>` is an *array* of `{matcher, hooks[]}` blocks where multiple unrelated blocks can share the same matcher value. `writeHookEntry` must locate "codegraph's own" block by scanning the array for a block whose `hooks[]` sub-array contains an entry with `"command"` equal to codegraph's own resolved command string — never by matcher value alone. Use `jsonDeepEqual`/`normalizeJSON` (already in this file, lines 80-128) for the idempotency comparison exactly as `writeMcpEntry` does.

**Keep-clean removal pattern** (lines 167-193, `removeMcpEntry`) — extend the same way: delete the matched block from `hooks[]`; if that empties the event's array, delete the event key; if that empties `hooks`, delete the top-level `hooks` key.

---

### `internal/agents/shared.go` — new strict JSON reader (settings.json hooks-merge + manifest)

**Analog:** `internal/githooks/githooks.go`'s `Install` read-switch (lines 256-297) — NOT `readJSONFile` (lines 45-67 of the same file), per RESEARCH.md Pattern 3/Critical Finding 3.

**Correct precedent — three-way outcome switch, skip-and-accumulate on error, never silently treat malformed as empty:**
```go
// internal/githooks/githooks.go:256-297
existing, err := os.ReadFile(file)
switch {
case err == nil:
	base := string(existing)
	stripped, ok := stripMarkerBlock(base)
	if !ok {
		// A malformed marker block means the strip can't be trusted (CR-01).
		// ... Skip this hook entirely and leave the file byte-for-byte
		// untouched — a skipped hook is recoverable, silently eaten user
		// content is not.
		errs = append(errs, fmt.Errorf("%s: hook file has a malformed codegraph marker block — please fix or remove it manually", hook))
		continue
	}
	// ... normal path
case errors.Is(err, fs.ErrNotExist):
	content = "#!/bin/sh\n" + block + "\n"
default:
	// CR-02: any other read error ... Skip this hook, accumulate the
	// error, and leave whatever is on disk untouched.
	errs = append(errs, fmt.Errorf("%s: could not read existing hook file: %w", hook, err))
	continue
}
```

**WRONG precedent to avoid copying — `readJSONFile`'s permissive fallback** (`internal/agents/shared.go:45-67`, doc comment lines 37-44):
```go
// readJSONFile parses path as a JSON object and returns it as a generic
// map. A missing file, an empty file, or unparseable content all fall
// back to an empty map rather than erroring...
func readJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]any{}, nil
		}
		return map[string]any{}, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		// Malformed existing config — defensive empty-map fallback per V5
		return map[string]any{}, nil
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}
```
This is correct for the *existing* MCP-entry/CLAUDE.md paths it already governs, but wrong for this phase's own success criterion. The new strict reader must distinguish: file absent (proceed as empty), genuine read/parse error (return error, caller must not write), success — matching the githooks three-way switch above, not `readJSONFile`'s two-way collapse.

---

### `internal/agents/shared.go` (or new file) — manifest hash helpers

**Analog:** `internal/upgrade/upgrade.go:194` — binary-integrity hashing idiom.

```go
// internal/upgrade/upgrade.go:194
digest := sha256.Sum256(binary)
return verifyRelease(b, trustedMaterial, "sha256", digest[:], releaseWorkflowRefPattern)
```
**What to copy:** `crypto/sha256` + hex-encode (`encoding/hex`) for content hashing — same algorithm/idiom already established in this codebase, avoiding a second hash convention. Per RESEARCH.md Pattern 4/Pitfall 5, hash only the codegraph-owned fragment of `settings.json` (re-marshaled deterministically via `normalizeJSON`), never the whole shared file.

---

### `internal/cli/upgrade.go` — extend `RunE` with post-swap refresh (D-06/D-07)

**Analog:** itself (same file) — injectable-seam pattern already established for testability.

**Current delegation shape** (lines 1-70):
```go
var upgradeRunFunc = upgrade.Run

func newUpgradeCmd() *cobra.Command {
	var check bool
	var force bool

	cmd := &cobra.Command{
		Use: "upgrade [version]",
		// ...
		RunE: func(cmd *cobra.Command, args []string) error {
			var pinned string
			if len(args) > 0 {
				pinned = args[0]
			}
			target, err := os.Executable()
			if err != nil {
				return fmt.Errorf("codegraph upgrade: resolve running binary path: %w", err)
			}
			return upgradeRunFunc(version.Info().Version, target, upgrade.Options{
				Check:   check,
				Version: pinned,
				Force:   force,
				Out:     cmd.OutOrStdout(),
			})
		},
	}
	// ...
	return cmd
}
```
**What to add (D-06/D-07):** after `upgradeRunFunc(...)` returns nil (swap succeeded, and not under `--check`), probe the 2 fixed manifest candidate paths (`~/.claude/skills/codegraph/.codegraph-manifest.json`, `./.claude/skills/codegraph/.codegraph-manifest.json`), and for each found, call `agents`'s Claude target `Install(loc, opts)` again with the new binary's embedded content. A refresh failure must be printed as a separate warning naming `codegraph install` — it must NOT change `upgrade`'s own returned error/exit code (D-07). Follow the same package-level-var injectable-seam idiom (`upgradeRunFunc`) for the new refresh call so `upgrade_test.go` can fake it without touching the filesystem/network.

---

### `internal/agents/claude_test.go` — new AGENT-01/02/03 test cases

**Analog:** `TestClaude_GlobalRoundTrip_ByteInvariantWithSibling` (lines 182-225), `TestClaude_Uninstall_PreservesUnrelatedAllowEntries` (lines 227-246), `TestClaude_Install_ReRunIsByteIdempotent` (lines 275+, same file).

**Byte-invariant round-trip pattern** (lines 182-225):
```go
func TestClaude_GlobalRoundTrip_ByteInvariantWithSibling(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".claude.json")
	pre := `{
  "mcpServers": {
    "other-server": {
      "command": "other-binary",
      "args": ["run"]
    }
  },
  "someUnrelatedKey": true
}
`
	writeFile(t, configPath, pre)

	c := claudeTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph", AutoAllow: true})
	c.Uninstall(LocationGlobal)

	got := readFile(t, configPath)
	var gotObj, wantObj map[string]any
	if err := json.Unmarshal([]byte(got), &gotObj); err != nil {
		t.Fatalf("post-round-trip config not valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(pre), &wantObj); err != nil {
		t.Fatalf("bad fixture: %v", err)
	}
	if !jsonDeepEqual(gotObj, wantObj) {
		t.Fatalf("round trip not byte-invariant:\ngot=%s\nwant=%s", got, pre)
	}
	// ... asserts settings.json / CLAUDE.md also cleaned up
}
```
**Preserves-unrelated-entries pattern** (lines 227-246) — write a fixture settings.json with an unrelated `hooks.SessionStart` matcher-block, `Uninstall`, assert codegraph's block is gone but the unrelated block/matcher survives (extend this exact shape for AGENT-02's hooks fixture per RESEARCH.md Wave 0 gaps).

**Idempotency pattern** (`TestClaude_Install_ReRunIsByteIdempotent`, starting line 275) — call `Install` twice, assert file contents identical byte-for-byte between the two runs; extend to also cover the new skill dir/script/hooks/manifest files.

**New test fixtures needed** (per RESEARCH.md Wave 0 Gaps): a `settings.json` with an unrelated `SessionStart` matcher-block AND an unrelated event key; a malformed `settings.json` fixture for the read-error matrix; a mode assertion (`os.Stat(scriptPath).Mode()&0o111 != 0`) after `Install()` for Pitfall 3 (executable script regression).

---

### `internal/cli/upgrade_test.go` — new `TestUpgradeCommand_RefreshesConfiguredTargets`-shaped test

**Analog:** `TestUpgradeCommand_DelegatesWithCheckAndVersion` (lines 11-46).

```go
func TestUpgradeCommand_DelegatesWithCheckAndVersion(t *testing.T) {
	var gotCurrent, gotTarget string
	var gotOpts upgrade.Options

	orig := upgradeRunFunc
	upgradeRunFunc = func(currentVersion, targetPath string, opts upgrade.Options) error {
		gotCurrent = currentVersion
		gotTarget = targetPath
		gotOpts = opts
		return nil
	}
	t.Cleanup(func() { upgradeRunFunc = orig })

	if _, _, err := execCmd("upgrade", "--check", "v1.4.0"); err != nil {
		t.Fatalf("upgrade --check v1.4.0: %v", err)
	}
	// ... assertions on gotOpts/gotCurrent/gotTarget
}
```
**What to copy:** the package-level-var swap + `t.Cleanup` restore idiom for faking `upgradeRunFunc` without touching the network. The new test needs an analogous injectable seam for whatever function performs the D-06 refresh (e.g. a package-level `agentsInstallFunc` var), so the test can assert `Install()` was re-invoked per discovered manifest without real filesystem/agents side effects, and separately assert a refresh failure surfaces as a warning without altering `upgrade`'s own success/exit code (D-07).

## Shared Patterns

### Mandatory error funnel (CR-01)
**Source:** `internal/agents/shared.go:12-35` (`recordFile`/`recordAction`)
**Apply to:** every new write/read call site this phase adds (skill write, script write, hooks merge, manifest write/read) — must route through `recordFile`/`recordAction`, never a bare `if err == nil { append }`.
```go
func recordFile(result *WriteResult, path string, fr FileResult, err error) {
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", path, err))
		return
	}
	result.Files = append(result.Files, fr)
}
```

### Atomic write primitive
**Source:** `internal/fsatomic/fsatomic.go:32-65` (`WriteFile`)
**Apply to:** every new file this phase writes (SKILL.md copy, session-nudge.sh, settings.json, manifest). Note the mode-preservation caveat: a brand-new file gets `0644`, so the script write must follow with an explicit `os.Chmod(scriptPath, 0o755)` (Pitfall 3) — `fsatomic.WriteFile` itself is not modified.
```go
func WriteFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// ... temp file + chmod-to-existing-mode + os.Rename
}
```

### JSON structural equality / normalization
**Source:** `internal/agents/shared.go:80-128` (`normalizeJSON`, `jsonDeepEqual`)
**Apply to:** the new hooks-merge idempotency comparison and manifest content hashing (re-marshal deterministically before hashing) — both should reuse these existing helpers unchanged rather than reimplementing structural comparison.

### FileAction/FileResult vocabulary (D-07/D-08 invariants)
**Source:** `internal/agents/types.go:57-108` (`FileAction`, `FileResult`, `WriteResult`)
**Apply to:** every new artifact type must report `ActionCreated`/`ActionUpdated`/`ActionUnchanged`/`ActionRemoved`/`ActionNotFound`/`ActionKept` using the existing enum — no new action values needed; re-running install twice must be byte-level `ActionUnchanged`, and uninstall on a never-installed target must be `ActionNotFound`, never an error.

## No Analog Found

None — every file this phase creates or modifies has a strong, explicitly-named in-repo analog (confirmed both by CONTEXT.md's own discretion notes and RESEARCH.md's Architecture Patterns section).

## Metadata

**Analog search scope:** `internal/agents/`, `internal/fsatomic/`, `internal/githooks/`, `internal/mcp/`, `internal/cli/`, `internal/upgrade/`
**Files scanned:** `claude.go`, `shared.go`, `types.go`, `fsatomic.go`, `githooks.go` (lines 230-300), `resources.go` (lines 1-30), `claude_test.go` (lines 175-284), `upgrade.go` (cli), `upgrade_test.go` (lines 11-47), `upgrade.go` (internal/upgrade, sha256 usage)
**Pattern extraction date:** 2026-08-13
