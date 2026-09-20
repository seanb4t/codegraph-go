# Phase 5: Agent Reach — Capability Model & Skill in Every Harness - Pattern Map

**Mapped:** 2026-09-18
**Files analyzed:** 15 (8 target files modified + 5 new files + 2 test fakes modified)
**Analogs found:** 15 / 15

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/agents/types.go` (interface grows `Capabilities()`) | interface/model | transform | itself (existing `AgentTarget` interface + enums) | exact — additive edit |
| `internal/agents/claude.go` (add `Capabilities()` literal) | config/service | CRUD (file writes) | `internal/agents/codex.go`'s `SupportsLocation`/path-func shape | exact — one-literal-per-file pattern already established |
| `internal/agents/cursor.go` (add `Capabilities()` literal) | config/service | CRUD | `internal/agents/codex.go` | exact |
| `internal/agents/codex.go` (add `Capabilities()` literal) | config/service | CRUD | itself (`SupportsLocation`, path funcs) | exact — template target for the pattern |
| `internal/agents/opencode.go` (add `Capabilities()` literal + skill-dir wiring, AGENT-06) | config/service | CRUD | `internal/agents/claude.go`'s skill-install block (lines 405-503) for the *skill* half; itself for MCP/instructions | exact (skill half is role-match, adapted from Claude's shared-writer-caller shape) |
| `internal/agents/hermes.go` (add `Capabilities()` literal, no behavior change) | config/service | CRUD (YAML splice) | itself | exact |
| `internal/agents/gemini.go` (add `Capabilities()` literal + harness-specific `.gemini/skills/` writer, AGENT-10) | config/service | CRUD | `internal/agents/claude.go`'s skill-install block for the skill write; itself for MCP/instructions | exact |
| `internal/agents/antigravity.go` (add `Capabilities()` literal, no instructions/skill change per Finding #3) | config/service | CRUD | itself | exact |
| `internal/agents/kiro.go` (add `Capabilities()` literal + NEW `kiroInstructionsPath` write, AGENT-11, + harness-specific `.kiro/skills/` writer, D-06) | config/service | CRUD | `internal/agents/codex.go` (adding an instructions path function + `upsertInstructionsEntry` call is exactly Codex's existing shape); `internal/agents/claude.go`'s skill-install block for the skill write | exact |
| `internal/agents/skillshared.go` (NEW — shared `.agents/skills/` writer, D-05/D-07/D-08) | service (shared writer) | CRUD, idempotent file-I/O | `internal/agents/claude.go` lines 405-503 (`Install`'s skill-package block: `writeEmbeddedFile` + `skillManifest` build + `writeManifest`) and lines 538-567 (`Uninstall`'s mirror-image removal) | exact — same primitives, generalized to a caller-supplied `requester TargetID` and multi-target `targets []TargetID` manifest field |
| `internal/agents/manifest.go` (`skillManifest` grows `Targets []TargetID`, D-07) | model | transform | itself (`writeManifest`/`readManifest`) | exact — additive field |
| `internal/agents/capabilities_test.go` (NEW — D-03 table-equality guard) | test | transform (table-driven assertion) | `internal/agents/registry_test.go`'s table-driven `fakeTarget` style; `internal/agents/manifest_test.go` for the writer/reader round-trip assertion shape | role-match |
| `internal/agents/skillshared_test.go` (NEW — AGENT-09) | test | file-I/O, idempotency | `internal/agents/claude_skillpackage_test.go` (`TestClaude_Install_WritesSkillPackage_EndToEnd`, full file — the tracer-test shape: `fakeHome(t)`, install, assert file bytes/mode, assert `WriteResult.Files`) | exact |
| `internal/agents/ownership_test.go` (NEW — D-13 planted-foreign-entry table, RED-first) | test | event-driven (ownership guard) | `git show 242ec0a`'s `TestClaude_Install_NeverClaimsOwnershipOfUnrelatedHookUnderSameMatcher` (quoted in RESEARCH.md Code Examples); `internal/agents/shared_test.go` for `writeHookEntry`/`removeMcpEntry` exact-identity assertions | exact |
| `internal/agents/registry_test.go`'s `fakeTarget` (add `Capabilities()` stub) | test fixture | transform | itself | exact — mechanical addition |
| `internal/cli/tui/agentpicker_test.go`'s `fakeAgentTarget` (add `Capabilities()` stub) | test fixture | transform | itself | exact — mechanical addition |
| `internal/cli/install.go` (add `--print-config-style` flag + render func, D-04) | CLI command | request-response (read-only report) | `internal/cli/install.go`'s own `printAgentResults` (lines 141-209) for the styled/plain dual-branch shape; `internal/cli/present/line.go`'s `KV`/`Line` helpers | exact — same file, same styled/plain branching pattern already in use |
| `docs/CLI-REFERENCE.md` (regenerated) | doc/generated artifact | batch (codegen) | itself, via `task docs:cli` | exact — tool-owned, do not hand-edit |

## Pattern Assignments

### `internal/agents/types.go` — `Capabilities()` addition to `AgentTarget`

**Analog:** itself (existing interface + `DescribePaths`/`SupportsLocation` doc-comment conventions, lines 124-160)

**Interface method doc-comment pattern** (lines 156-159, the shape to mirror for the new method):
```go
// DescribePaths returns every config/instructions file path this
// target reads or writes at loc, for --print-config-style reporting
// and test assertions.
DescribePaths(loc Location) []string
```
Add `Capabilities() Capabilities` immediately after `DescribePaths` with a comment naming D-01/D-02/D-08's derivation contract ("SupportsLocation, DescribePaths, and Detect's path inputs are derivations of this table — see D-02").

**New enum pattern** — mirror the existing `FileAction`/`Location` const-block shape (lines 17-24, 59-79) for the new `HookMechanism` and `ConfigFormat` enums:
```go
// Location is the install/uninstall scope: a per-user (global) config or a
// per-project (local) config.
type Location string

const (
	LocationGlobal Location = "global"
	LocationLocal  Location = "local"
)
```

---

### `internal/agents/codex.go` — template for every target's `Capabilities()` literal

**Analog:** itself — every other target's literal follows this exact shape

**Existing one-liner-per-property pattern to extend** (lines 26-28):
```go
func (codexTarget) ID() TargetID                       { return Codex }
func (codexTarget) DisplayName() string                { return "Codex CLI" }
func (codexTarget) SupportsLocation(loc Location) bool { return loc == LocationGlobal }
```

**Path functions to call from the literal, never restate** (lines 30-44):
```go
func codexConfigPath() (string, error) { ... filepath.Join(home, ".codex", "config.toml") ... }
func codexInstructionsPath() (string, error) { ... filepath.Join(home, ".codex", "AGENTS.md") ... }
```

**Capabilities() literal shape** (RESEARCH.md's illustrative example, lines 229-238 — field names are Claude's Discretion per D-02, but the *shape* — one struct literal, same file, calling existing path funcs — is load-bearing):
```go
func (codexTarget) Capabilities() Capabilities {
	return Capabilities{
		Scopes:       []Location{LocationGlobal},
		ConfigFormat: ConfigFormatTOML,
		Hooks:        HooksNone,
		// MCPConfig/Instructions/SkillDirs: call codexConfigPath()/
		// codexInstructionsPath() per-scope, never re-literal the string.
	}
}
```

**Anti-pattern to avoid** (RESEARCH.md "Anti-Patterns to Avoid"): restating `"~/.codex/config.toml"` as a literal string inside `Capabilities()` instead of calling `codexConfigPath()` — this is exactly what D-03's planted-divergence guard exists to catch.

---

### `internal/agents/claude.go` — the shared-writer caller shape (for opencode.go/gemini.go/kiro.go's new skill-dir wiring)

**Analog:** `internal/agents/claude.go` lines 405-503 (`Install`'s skill-package block) and lines 538-567 (`Uninstall`'s mirror)

**Install-side pattern to copy for calling the new shared writer** (lines 430-444 — the `writeEmbeddedFile` + `recordFile` + captured-content-for-manifest shape):
```go
if skillFilePath, err := claudeSkillFilePath(loc); err != nil {
	result.Errors = append(result.Errors, fmt.Errorf("resolve claude skill file path: %w", err))
} else {
	content, rerr := claudeassets.SkillMarkdown()
	if rerr != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", skillFilePath, rerr))
	} else {
		fr, werr := writeEmbeddedFile(skillFilePath, string(content), false)
		recordFile(&result, skillFilePath, fr, werr)
		if werr == nil {
			skillMDContent = content
			haveSkillMDContent = true
		}
	}
}
```
For opencode/gemini/antigravity's *shared*-path install, the equivalent call is `installSharedSkillPackage(loc, Opencode)` (or `Gemini`/`Antigravity`) from the new `skillshared.go` — see that file's pattern below — funnelled through `recordFile` exactly the same way.

**Uninstall-side pattern to copy** (lines 555-567 — remove-then-`removeSkillDirIfEmpty`):
```go
if skillFilePath, err := claudeSkillFilePath(loc); err != nil {
	result.Errors = append(result.Errors, fmt.Errorf("resolve claude skill file path: %w", err))
} else {
	fr, rerr := removeEmbeddedFile(skillFilePath)
	recordFile(&result, skillFilePath, fr, rerr)
	if rerr == nil {
		if skillDir, derr := claudeSkillDirPath(loc); derr == nil {
			if cerr := removeSkillDirIfEmpty(skillDir); cerr != nil {
				result.Errors = append(result.Errors, fmt.Errorf("%s: %w", skillDir, cerr))
			}
		}
	}
}
```
D-08's twist for the shared writer: uninstall must remove *only the calling target's ID* from the manifest's `targets` list, deleting the package only when `targets` becomes empty — this is new logic in `skillshared.go`, not a verbatim copy of Claude's single-owner removal.

**Manifest-build pattern** (lines 478-503 — build only after every prerequisite content resolved, hash via `hashContent`):
```go
if manifestPath, err := claudeManifestPath(loc); err != nil {
	result.Errors = append(result.Errors, fmt.Errorf("resolve claude manifest path: %w", err))
} else if haveSkillMDContent && haveScriptContent && haveSessionStart {
	m := skillManifest{
		SchemaVersion:    manifestSchemaVersion,
		CodegraphVersion: version.Info().Version,
		Location:         string(loc),
		Files: map[string]string{ manifestKeySkillMD: hashContent(skillMDContent), ... },
	}
	fr, werr := writeManifest(manifestPath, m)
	recordFile(&result, manifestPath, fr, werr)
}
```

**Error-handling pattern (repo-wide convention)**: every path-resolution error is wrapped `fmt.Errorf("resolve <target> <thing> path: %w", err)` and appended to `result.Errors`; every write outcome funnels through `recordFile(&result, path, fr, err)` — never a bare `if err != nil { return }` that would swallow one step's failure and skip the rest silently (CR-01, `internal/agents/shared.go` lines 12-24).

---

### `internal/agents/skillshared.go` (NEW) — the shared `.agents/skills/` writer

**Analog:** `internal/agents/claude.go`'s skill-install/-uninstall blocks (above) generalized; `internal/agents/manifest.go`'s `writeManifest`/`readManifest`/`stringMapEqual` reused verbatim, not reimplemented.

**Core pattern** (RESEARCH.md Code Examples, illustrative shape — reuse `writeEmbeddedFile`/`readManifest`/`writeManifest` verbatim, read the manifest fresh on every call, never memoize "already wrote this run"):
```go
func installSharedSkillPackage(loc Location, requester TargetID) (WriteResult, error) {
	dir := sharedSkillDirPath(loc) // .agents/skills/codegraph or ~/.agents/skills/codegraph
	skillPath := filepath.Join(dir, "SKILL.md")
	content, err := claudeassets.SkillMarkdown() // same embed, no new asset
	if err != nil {
		return WriteResult{}, err
	}
	var result WriteResult
	fr, werr := writeEmbeddedFile(skillPath, string(content), false)
	recordFile(&result, skillPath, fr, werr)

	manifestPath := filepath.Join(dir, ".codegraph-manifest.json")
	existing, present, _ := readManifest(manifestPath)
	targets := existing.Targets // read fresh, not memoed — D-08 needs every requester
	if present && !containsTarget(targets, requester) {
		targets = append(targets, requester)
	} else if !present {
		targets = []TargetID{requester}
	}
	// ... writeManifest with schema_version bumped, targets: targets ...
	return result, nil
}
```

**D-17's symlink guard (new, no direct analog — write it fresh)**: both this writer and `claude.go`'s existing writer must compare their target directories with `filepath.EvalSymlinks` before writing; if `claudeSkillDirPath(loc)` and `sharedSkillDirPath(loc)` resolve to the same physical directory, treat it as one package with one manifest and record Claude's target ID in `targets` like any other requester (D-08's last-requester-removal rule applies, not a delete-out-from-under-others). Pin with a symlinked `fakeHome` test layout, RED-first.

**Anti-pattern to avoid** (RESEARCH.md "Anti-Patterns to Avoid"): a package-level "already wrote the shared package this run" boolean — breaks D-08's per-target `targets` accumulation for the second+ target in one run. `writeEmbeddedFile`'s own byte-comparison already gives the SKILL.md write its idempotency; only the manifest's `targets` field needs the "read fresh every call" discipline.

**Manifest schema-upgrade pattern**: `internal/agents/manifest.go` lines 47-53 (`skillManifest` struct) — add `Targets []TargetID` additively; an older manifest read without the field decodes to a nil/empty slice, which the writer's "upgrade in place" logic (D-07) must populate with `[requester]` on the next write, not treat as an error.

---

### `internal/agents/manifest.go` — `Targets` field addition

**Analog:** itself, lines 47-53 (`skillManifest` struct) and lines 119-164 (`writeManifest`'s no-needless-rewrite idempotency)

**Struct to extend**:
```go
type skillManifest struct {
	SchemaVersion    int               `json:"schema_version"`
	CodegraphVersion string            `json:"codegraph_version"`
	InstalledAt      string            `json:"installed_at"`
	Location         string            `json:"location"`
	Files            map[string]string `json:"files"`
	// NEW: Targets []TargetID `json:"targets"` (D-07) — additive.
}
```

**Idempotency comparison to extend** (lines 143-149) — `writeManifest`'s equality check must also compare `Targets` (order-independent, since D-08 appends per-run in registry order) before short-circuiting to `ActionUnchanged`:
```go
if existedBefore &&
	existing.SchemaVersion == m.SchemaVersion &&
	existing.CodegraphVersion == m.CodegraphVersion &&
	existing.Location == m.Location &&
	stringMapEqual(existing.Files, m.Files) {
	return FileResult{Path: path, Action: ActionUnchanged}, nil
}
```

---

### `internal/agents/kiro.go` — new `kiroInstructionsPath` + skill-dir wiring (AGENT-11)

**Analog:** `internal/agents/codex.go` (adding an instructions-file write to a target that currently has none is exactly Codex's existing shape, lines 38-44 + 110-115 + 142-147)

**New path function pattern** (mirror `codexInstructionsPath`, lines 38-44):
```go
func kiroInstructionsPath(loc Location) (string, error) {
	if loc == LocationLocal {
		return "AGENTS.md", nil // Finding #4: same literal path opencode already uses locally
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".kiro", "steering", "AGENTS.md"), nil
}
```

**Install-side call pattern** (mirror `codex.go` lines 110-115):
```go
if instrPath, err := kiroInstructionsPath(loc); err != nil {
	result.Errors = append(result.Errors, fmt.Errorf("resolve kiro instructions path: %w", err))
} else {
	fr, err := upsertInstructionsEntry(instrPath, codegraphSectionStart, codegraphSectionEnd, instructionsBody())
	recordFile(&result, instrPath, fr, err)
}
```

**Uninstall-side call pattern** (mirror `codex.go` lines 142-147):
```go
if instrPath, err := kiroInstructionsPath(loc); err != nil {
	result.Errors = append(result.Errors, fmt.Errorf("resolve kiro instructions path: %w", err))
} else {
	action, err := removeMarkedSection(instrPath, codegraphSectionStart, codegraphSectionEnd)
	recordAction(&result, instrPath, action, err)
}
```
No new collision-handling code needed at local scope (Finding #4): `upsertInstructionsEntry`/`removeMarkedSection` are already idempotent, marker-fenced upserts — a second target writing the identical block to the same `AGENTS.md` is a no-op collision.

**Kiro/Gemini's harness-specific skill-dir writer** — same `writeEmbeddedFile` call as `claude.go`'s skill block, targeting `.kiro/skills/codegraph/SKILL.md` / `.gemini/skills/codegraph/SKILL.md` respectively, called *in addition to* the shared writer (D-06).

---

### `internal/agents/registry_test.go`'s `fakeTarget` and `internal/cli/tui/agentpicker_test.go`'s `fakeAgentTarget` — `Capabilities()` stub

**Analog:** each fake's own existing method set (Pitfall 1 — both MUST be updated in the same commit as the interface change, or the build breaks in a package the plan may not have touched)

**Pattern** (`registry_test.go` lines 18-26 — add one more one-liner in the same style):
```go
func (f fakeTarget) ID() TargetID                   { return f.id }
func (f fakeTarget) DisplayName() string            { return string(f.id) }
func (f fakeTarget) SupportsLocation(Location) bool { return true }
// NEW: func (f fakeTarget) Capabilities() Capabilities { return Capabilities{} }
func (f fakeTarget) Detect(Location) DetectionResult { return DetectionResult{Installed: f.installed} }
func (f fakeTarget) Install(Location, InstallOptions) WriteResult { return WriteResult{} }
func (f fakeTarget) Uninstall(Location) WriteResult               { return WriteResult{} }
func (f fakeTarget) DescribePaths(Location) []string              { return nil }
```
Same shape, `agents.Capabilities{}` qualifier, for `agentpicker_test.go`'s `fakeAgentTarget` (lines 21-33).

---

### `internal/agents/capabilities_test.go` (NEW) — D-03 table-equality guard

**Analog:** `internal/agents/registry_test.go`'s table-driven-over-`AllTargetIDs()` style (loops every registered target); `internal/agents/manifest_test.go` for the "build expected, compare to actual" round-trip assertion shape.

**Core pattern**: for every target in `agents.AllTargets()` × both `Location` values, assert `t.DescribePaths(loc)` equals the path set flattened out of `t.Capabilities()` (MCPConfig, Instructions if non-empty, SkillDirs[0] if present) — order-independent set comparison, not slice-equality, since D-02 doesn't mandate an ordering between `DescribePaths` and `Capabilities`'s derivation. Also assert `--print-config-style`'s rendered output (captured via `cmd.SetOut(&buf)`, mirroring `internal/cli/install_test.go`'s existing capture pattern — not read this session, but the project's `cmd.SetOut`/`bytes.Buffer` idiom is repo-standard for cobra command tests) matches the table exactly.

**Mutation-guard pattern (D-03)**: plant a divergence — add one path to a target's `DescribePaths` that its `Capabilities()` literal does not declare (or the reverse) — and assert the equality check goes RED; revert byte-clean, log in `05-MUTATION-LOG.md` (Family (a), per the Phase 2/3/4 shape already established in earlier phases' own mutation logs).

---

### `internal/agents/skillshared_test.go` (NEW) — AGENT-09

**Analog:** `internal/agents/claude_skillpackage_test.go`'s `TestClaude_Install_WritesSkillPackage_EndToEnd` (full file, lines 1-120+ read this session)

**Structure to mirror** (the tracer-test shape: `fakeHome(t)` isolation → call the writer → assert file bytes match the embed → assert `WriteResult.Files` names every touched path):
```go
func TestSharedSkillPackage_WritesOnceAndAccumulatesTargets(t *testing.T) {
	home := fakeHome(t)

	r1, err := installSharedSkillPackage(LocationGlobal, Opencode)
	// assert SKILL.md bytes == claudeassets.SkillMarkdown(), manifest.targets == ["opencode"]

	r2, err := installSharedSkillPackage(LocationGlobal, Gemini)
	// assert SKILL.md write reports ActionUnchanged (2nd target, same run)
	// assert manifest.targets == ["opencode", "gemini"] — NOT just ["gemini"]
}
```
Also cover: uninstalling one requester removes only that ID from `targets` (D-08); the package (SKILL.md + manifest) is deleted only once `targets` is empty; the D-17 symlinked-`fakeHome` case (Claude's dir and the shared dir resolve to one physical directory).

---

### `internal/agents/ownership_test.go` (NEW, RED-first) — D-13/AGENT-13

**Analog:** `git show 242ec0a`'s `TestClaude_Install_NeverClaimsOwnershipOfUnrelatedHookUnderSameMatcher` (quoted structure, RESEARCH.md Code Examples) — the literal template D-15 requires citing by SHA.

**Precondition/assertion shape to reuse** (RESEARCH.md lines 366-382):
```go
first := c.Install(LocationGlobal, opts)                 // 1. establish a manifest at this location
// plant a foreign entry that shares codegraph's own matcher/key/directory-name —
// NOT merely an unrelated entry somewhere else (Pitfall 5: the weaker precondition
// would not have caught the historical vulnerability)
writeFile(t, settingsPath, `{"hooks":{"SessionStart":[{"matcher":"startup","hooks":[
  {"type":"command","command":"/opt/some-other-tool/on-startup.sh"}]}]}}`)
second := c.Install(LocationGlobal, opts)                  // 3. re-install
// 4. assert: the unrelated command survives byte-for-byte;
//    codegraph's own blocks are appended alongside it, never merged in
```
Extend this same shape across all 8 targets × both locations, and additionally plant: a foreign `.agents/skills/other/` directory, a foreign `.agents/skills/codegraph/` directory *without* a codegraph manifest (must be `kept (foreign)`, never overwritten — D-14), and a foreign marker-less section in every instructions file. Doc-comment must cite `242ec0a` by SHA and name the differential it closed (D-15).

**RED-first discipline (tdd_mode)**: write this test before the ownership-guard code exists (it should already pass against current single-target behavior for the 7 non-Claude targets since they have no shared-skill-dir concept yet); the NEW RED case is specifically the shared-writer's foreign-directory-without-manifest scenario, which must go RED against a naive first implementation before `skillshared.go`'s D-14 check is added.

---

### `internal/cli/install.go` — `--print-config-style` flag (D-04)

**Analog:** itself — `printAgentResults` (lines 141-209), the styled/plain dual-branch convention already established in this exact file.

**Flag registration pattern** (lines 107-110 — add alongside the existing flags):
```go
cmd.Flags().StringVarP(&target, "target", "t", "auto", "which agents to configure: auto|all|none|<comma-separated ids>")
cmd.Flags().StringVarP(&location, "location", "l", string(agents.LocationGlobal), "config scope: global|local")
cmd.Flags().BoolVar(&autoAllow, "auto-allow", false, "also add mcp__codegraph__* to Claude Code's permissions.allow list")
// NEW: cmd.Flags().BoolVar(&printConfigStyle, "print-config-style", false, "print each target's capability table and exit (no writes)")
```

**Read-only-report render pattern** — reuse `internal/cli/present/line.go`'s `KV`/`Line` behind `resolveColor(cmd)`, mirroring `printAgentResults`'s own styled/plain branch (lines 143-149, 171-175):
```go
mode := resolveColor(cmd)
var pal present.Palette
var w io.Writer
if mode.Styled {
	pal = present.NewPalette(mode.Dark)
	w = mode.Writer(out)
}
// per target: mode.Styled ? present.Line/KV through pal : fmt.Fprintf(out, "%s: scopes=%s mcp=%s ...\n", ...)
```
Route BEFORE `printAgentResults` is called in `RunE` — this flag opens no picker and writes nothing (D-04), so it must short-circuit the picker/auto-resolve branch entirely, the same early-return discipline `yes` already uses in the existing `switch` (lines 80-95).

**Sanitization requirement**: any filesystem path printed through this flag must go through `sanitizePathForDisplay` exactly like every path `printAgentResults` already prints (line 187, 201) — no new sanitization primitive.

**Golden-frozen plain output**: the plain (non-styled) rendering is byte-stable and covered by `internal/cli/plain_golden_test.go`'s existing harness (Phase 4) — add this flag's output as a new golden fixture there, not a new ad hoc byte-comparison test.

## Shared Patterns

### Exact-identity ownership (never shape/position)
**Source:** `internal/agents/shared.go` lines 174-269 (`writeHookEntry`'s doc comment and `isOwned` closure) — the `242ec0a` revert's restored invariant.
**Apply to:** the new `skillshared.go` writer's foreign-directory detection (D-14), and `ownership_test.go`'s planted-foreign-entry table (D-13) for all 8 targets.
```go
// Ownership of a block is determined SOLELY by exact command-string match
// within the block's own hooks[] sub-array against ownCommands, never by
// the block's matcher value or shape...
```
For skill directories, the equivalent identity signal is D-14: "our sidecar manifest present" — a `codegraph/` directory without one is foreign, reported `kept (foreign)`, never overwritten or removed.

### recordFile/recordAction — the single write-outcome funnel
**Source:** `internal/agents/shared.go` lines 12-35.
**Apply to:** every new `Install`/`Uninstall` step this phase adds (Kiro's new instructions write, every target's skill-dir wiring) — never a bare `if err != nil { return }` that skips remaining steps silently (CR-01).
```go
func recordFile(result *WriteResult, path string, fr FileResult, err error) {
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("%s: %w", path, err))
		return
	}
	result.Files = append(result.Files, fr)
}
```

### writeEmbeddedFile — raw-byte idempotent artifact writes
**Source:** `internal/agents/shared.go` lines 286-341.
**Apply to:** `skillshared.go`'s shared-package SKILL.md write, and Gemini/Kiro's harness-specific SKILL.md writes — reuse verbatim, do not build a second byte-comparison helper (RESEARCH.md "Don't Hand-Roll").

### removeSkillDirIfEmpty — never a recursive delete
**Source:** `internal/agents/shared.go` lines 479-500.
**Apply to:** every skill-directory uninstall path this phase adds (shared `.agents/skills/codegraph/`, `.gemini/skills/codegraph/`, `.kiro/skills/codegraph/`) — reused verbatim; never delete a directory that still has entries (a user file, or another target's still-live manifest).

### upsertInstructionsEntry / removeMarkedSection — marker-fenced instructions upsert
**Source:** `internal/agents/shared.go` lines 670-683 (`upsertInstructionsEntry`) and 616-668 (`removeMarkedSection`); the marker constants in `internal/agents/instructions.go` lines 8-11.
**Apply to:** Kiro's new `kiroInstructionsPath` write (AGENT-11) — no Kiro-specific instructions writer needed, this is a location-path change only.

### writeManifest / readManifest — drift signal, never tamper-detection
**Source:** `internal/agents/manifest.go` lines 90-164, and its doc comment lines 9-15.
**Apply to:** `skillshared.go`'s manifest read/write for the shared package, and D-16's "hand-edited own file keeps the existing manifest response" posture — a hash mismatch is recorded, never treated as a security event, for the new `targets`-bearing manifest exactly as for the existing one.

### The styled/plain dual-branch CLI report
**Source:** `internal/cli/install.go`'s `printAgentResults` (lines 141-209), `internal/cli/present/line.go`'s `KV`/`Line`, `internal/cli/colorflag.go`'s `resolveColor`.
**Apply to:** `--print-config-style`'s new render function (D-04) — same `mode := resolveColor(cmd)` → `if mode.Styled { pal := present.NewPalette(mode.Dark); ... } else { fmt.Fprintf(...) }` shape, same `sanitizePathForDisplay` call on every printed path.

## No Analog Found

None. Every file this phase touches is either an additive edit to an existing target file (all 8 targets already implement the interface this phase extends) or a new file whose closest analog is `internal/agents/claude.go`'s existing skill-install/-uninstall block plus `internal/agents/manifest.go`'s existing manifest read/write — both read in full this session and quoted above. The two genuinely novel pieces of logic (D-17's symlink-collision check, and the "shared package accumulates `targets` across a single install run without a memo") have no prior in-repo analog to copy verbatim; RESEARCH.md's "Architecture Patterns → Pattern 2" and "Anti-Patterns to Avoid" sections (quoted above under `skillshared.go`) are the design to follow instead of a copy-paste source.

## Metadata

**Analog search scope:** `internal/agents/*.go` (all 8 target files + `types.go`, `registry.go`, `shared.go`, `manifest.go`, `instructions.go`), `claudeassets.go`, `internal/cli/{install,uninstall}.go`, `internal/cli/present/line.go`, `internal/cli/colorflag.go`, `internal/agents/{registry_test,claude_skillpackage_test,testhelpers_test}.go`, `internal/cli/tui/agentpicker_test.go`, `git show 242ec0a` — every path named above was read directly this session (no Glob/Grep-only inference), matching RESEARCH.md's own read list.
**Files scanned:** 20 source files, ~9,465 lines total across `internal/agents` + `internal/cli` per `wc -l`.
**Pattern extraction date:** 2026-09-18
**Tracked-source gate:** `git ls-files` confirmed all 20 analog paths above are tracked source (no gitignored mirror paths in this repository).
