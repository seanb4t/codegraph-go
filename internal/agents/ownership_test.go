package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	claudeassets "github.com/seanb4t/codegraph-go"
	"github.com/tailscale/hujson"
)

// TestOwnershipExactIdentity (D-13) is the planted-foreign-entry table over
// every registered target x {global, local} x {foreign-codegraph-dir,
// clean} — 32 leaves. It extends the exact-identity ownership discipline
// commit 242ec0a418703c6a4dab45188242149960cda77d ("fix(07-03): revert
// hook-block ownership recovery, close authorization-differential")
// restored for Claude's SessionStart hooks to EVERY skill/config/
// instructions write this package makes.
//
// 242ec0a reverted a matcher-and-shape recovery path in writeHookEntry that
// let codegraph silently claim and overwrite an unrelated, user-authored
// SessionStart hook block whenever that block happened to share codegraph's
// own matcher name ("startup") AND a codegraph manifest already existed at
// that install location — ownership had been granted by (a manifest
// present at this location) + (matcher name and shape), not by the block's
// own command-string identity. The claude/* leaves below reproduce that
// exact precondition (install once so a manifest exists, hand-edit the
// SessionStart matcher slot to hold an unrelated command, re-install,
// uninstall) rather than merely planting an unrelated command somewhere
// else in the file (RESEARCH.md Pitfall 5: the WEAKER precondition would
// not have caught the historical vulnerability).
//
// Every skill-directory write this phase adds (Cursor, opencode — and,
// pending 05-05, Gemini/Kiro/Antigravity) uses the same narrow identity
// 242ec0a's revert restored: a skill directory is codegraph's own only if
// codegraph's OWN manifest file is present in it (D-14), never because its
// name is "codegraph" or it happens to contain a SKILL.md. Family (b) in
// 05-MUTATION-LOG.md demonstrates this guard going RED against both shapes
// of that historical vulnerability (shape-based skill-dir ownership, and
// the literal matcher-based hook recovery reintroduced).
// The hardening commit this table guards is 242ec0a418703c6a4dab45188242149960cda77d.

// ownershipWantSkillDir is an INDEPENDENT oracle for the skill directory a
// target writes at loc — built by hand from this phase's own decisions
// (D-05/D-06), never by calling Capabilities().WrittenSkillDir, so the
// table under test and this oracle cannot drift together. "" means the
// target declares no skill directory yet at this plan.
func ownershipWantSkillDir(id TargetID, loc Location, home string) string {
	switch id {
	case Claude:
		if loc == LocationLocal {
			return filepath.Join(".claude", "skills", "codegraph")
		}
		return filepath.Join(home, ".claude", "skills", "codegraph")
	case Cursor, Opencode, Codex:
		if loc == LocationLocal {
			return filepath.Join(".agents", "skills", "codegraph")
		}
		return filepath.Join(home, ".agents", "skills", "codegraph")
	case Gemini:
		if loc == LocationLocal {
			return filepath.Join(".gemini", "skills", "codegraph")
		}
		return filepath.Join(home, ".gemini", "skills", "codegraph")
	case Kiro:
		if loc == LocationLocal {
			return filepath.Join(".kiro", "skills", "codegraph")
		}
		return filepath.Join(home, ".kiro", "skills", "codegraph")
	case Antigravity:
		if loc == LocationLocal {
			return ""
		}
		return filepath.Join(home, ".gemini", "config", "skills", "codegraph")
	default:
		return ""
	}
}

// newSkillDirs lists every skill directory this PHASE introduces (D-05,
// D-06) at loc: the shared package every target eventually funnels onto,
// plus the two harness-specific directories (Gemini, Kiro) and, at global
// scope only, two Antigravity roots: the config dir it writes (maintainer
// decision 1A, the one agy is live-proven to read) and the former
// antigravity-cli path, proven unread and no longer written — a foreign
// codegraph/ dir planted there must stay untouched. These are the
// ownership guard's planting sites for the foreign-codegraph-dir variant,
// and the sites that must never accumulate a manifest for a target that
// does not write there.
func newSkillDirs(home string, loc Location) []string {
	if loc == LocationLocal {
		return []string{
			filepath.Join(".agents", "skills", "codegraph"),
			filepath.Join(".gemini", "skills", "codegraph"),
			filepath.Join(".kiro", "skills", "codegraph"),
		}
	}
	return []string{
		filepath.Join(home, ".agents", "skills", "codegraph"),
		filepath.Join(home, ".gemini", "skills", "codegraph"),
		filepath.Join(home, ".kiro", "skills", "codegraph"),
		filepath.Join(home, ".gemini", "antigravity-cli", "skills", "codegraph"),
		filepath.Join(home, ".gemini", "config", "skills", "codegraph"),
	}
}

// containsPath reports whether needle is present in haystack.
func containsPath(haystack []string, needle string) bool {
	for _, p := range haystack {
		if p == needle {
			return true
		}
	}
	return false
}

// foreignMCPContent renders a foreign (non-codegraph) MCP server entry in
// format's canonical on-disk shape, so a round trip through install/
// uninstall can be asserted byte-identical rather than merely
// content-equal.
func foreignMCPContent(t *testing.T, format ConfigFormat) string {
	t.Helper()
	switch format {
	case ConfigFormatJSON:
		v := map[string]any{
			"mcpServers": map[string]any{
				"other-server": map[string]any{
					"args":    []string{"serve"},
					"command": "/opt/other/bin/tool",
				},
			},
		}
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatalf("marshal foreign json fixture: %v", err)
		}
		return string(data) + "\n"
	case ConfigFormatJSONC:
		src := "{\n" +
			"  \"$schema\": \"https://opencode.ai/config.json\",\n" +
			"  \"mcp\": {\n" +
			"    \"other-server\": {\n" +
			"      \"type\": \"local\",\n" +
			"      \"command\": [\"/opt/other/bin/tool\", \"serve\"],\n" +
			"      \"enabled\": true\n" +
			"    }\n" +
			"  }\n" +
			"}\n"
		v, err := hujson.Parse([]byte(src))
		if err != nil {
			t.Fatalf("parse foreign jsonc fixture: %v", err)
		}
		v.Format()
		return string(v.Pack())
	case ConfigFormatTOML:
		return tomlUnrelatedTable
	case ConfigFormatYAML:
		return hermesPyYAMLDefaultFixture
	default:
		t.Fatalf("foreignMCPContent: unhandled ConfigFormat %q", format)
		return ""
	}
}

// agentsSkillsOtherDir is a foreign sibling skill directory planted next to
// the shared codegraph package at every leaf — codegraph must never even
// look at it, let alone touch it.
func agentsSkillsOtherDir(home string, loc Location) string {
	if loc == LocationLocal {
		return filepath.Join(".agents", "skills", "other")
	}
	return filepath.Join(home, ".agents", "skills", "other")
}

const ownershipForeignSkillContent = "# someone else's codegraph skill\n"
const ownershipForeignInstructionsContent = "# My own notes\n\nKeep this paragraph exactly.\n"
const ownershipForeignSiblingSkillContent = "# an unrelated skill\n"
const ownershipUnrelatedHookCommand = "/opt/some-other-tool/on-startup.sh"

// ownershipUnrelatedPreToolUseBlock is an unrelated PreToolUse block
// planted under the SAME "Bash" matcher codegraph's own opt-in block uses
// (D-11). Ownership is the exact command string, never the matcher
// (242ec0a), so it must survive install and uninstall byte-identical.
const ownershipUnrelatedPreToolUseBlock = `{"matcher":"Bash","hooks":[{"type":"command","command":"` + ownershipUnrelatedHookCommand + `"}]}`

// ownershipCodexForeignBashGroup (07-08, D-23) is an unrelated PreToolUse
// group planted under Codex's OWN matcher, "^Bash$" — the 242ec0a shape for
// Codex's hooks.json: ownership must be the exact command string, never the
// matcher, so this group must survive install and uninstall byte-identical.
const ownershipCodexForeignBashGroup = `{"matcher":"^Bash$","hooks":[{"type":"command","command":"` + ownershipUnrelatedHookCommand + `"}]}`

// plantForeignCodexHooksGroup seeds loc's Codex hooks.json with
// ownershipCodexForeignBashGroup before Install — exercising exact-identity
// ownership (D-23) the same way reproduce242ec0aPrecondition does for
// Claude's SessionStart matcher, but without needing a pre-install call
// first: Codex's stickiness evidence is the hooks.json group itself, never
// a manifest, so there is no manifest precondition to establish first.
func plantForeignCodexHooksGroup(t *testing.T, loc Location) {
	t.Helper()
	hooksPath, err := codexHooksJSONPath(loc)
	if err != nil {
		t.Fatalf("codexHooksJSONPath(%s): %v", loc, err)
	}
	writeFile(t, hooksPath, `{"hooks":{"PreToolUse":[`+ownershipCodexForeignBashGroup+`]}}`)
}

// foreignPlant records exactly what was planted at one ownership leaf, so
// the post-uninstall assertions compare against what was ACTUALLY written
// rather than re-deriving it a second time.
type foreignPlant struct {
	mcpPath        string
	mcpContent     string
	hasMCP         bool
	instrPath      string
	instrContent   string
	hasInstr       bool
	siblingPath    string
	siblingContent string
	skillRootFiles map[string]string
}

// plantForeignFixtures seeds every foreign artifact a D-13 leaf needs
// BEFORE any Install call: a foreign MCP entry (when loc is supported), a
// foreign instructions section (when the target declares one), a foreign
// sibling skill directory, and — for the foreign-codegraph-dir variant — a
// manifest-less SKILL.md in every newSkillDirs root.
func plantForeignFixtures(t *testing.T, caps Capabilities, loc Location, home string, plantForeignSkillDirs bool) foreignPlant {
	t.Helper()
	var p foreignPlant

	if caps.Supports(loc) && caps.MCPConfig != nil {
		path, err := caps.MCPConfig(loc)
		if err != nil {
			t.Fatalf("resolve MCP config path: %v", err)
		}
		content := foreignMCPContent(t, caps.ConfigFormat)
		writeFile(t, path, content)
		p.mcpPath, p.mcpContent, p.hasMCP = path, content, true
	}

	if caps.Supports(loc) {
		instrPath, err := caps.InstructionsPath(loc)
		if err != nil {
			t.Fatalf("resolve instructions path: %v", err)
		}
		if instrPath != "" {
			writeFile(t, instrPath, ownershipForeignInstructionsContent)
			p.instrPath, p.instrContent, p.hasInstr = instrPath, ownershipForeignInstructionsContent, true
		}
	}

	siblingPath := filepath.Join(agentsSkillsOtherDir(home, loc), "SKILL.md")
	writeFile(t, siblingPath, ownershipForeignSiblingSkillContent)
	p.siblingPath, p.siblingContent = siblingPath, ownershipForeignSiblingSkillContent

	if plantForeignSkillDirs {
		p.skillRootFiles = make(map[string]string)
		for _, dir := range newSkillDirs(home, loc) {
			path := filepath.Join(dir, skillFileName)
			writeFile(t, path, ownershipForeignSkillContent)
			p.skillRootFiles[path] = ownershipForeignSkillContent
		}
	}

	return p
}

// snapshotTree returns a sorted, newline-joined listing of every path under
// root — used to assert an unsupported-location Install/Uninstall call
// changes NOTHING on disk.
func snapshotTree(t *testing.T, roots ...string) string {
	t.Helper()
	var lines []string
	seen := make(map[string]bool)
	for _, root := range roots {
		if seen[root] {
			continue
		}
		seen[root] = true
		_ = filepath.Walk(root, func(path string, _ os.FileInfo, err error) error {
			if err != nil {
				return nil //nolint:nilerr // best-effort snapshot; a walk error just stops descending that branch
			}
			rel, rerr := filepath.Rel(root, path)
			if rerr != nil {
				rel = path
			}
			lines = append(lines, root+"::"+rel)
			return nil
		})
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

// reproduce242ec0aPrecondition is the D-15 requirement: the claude/* leaves
// reproduce the EXACT precondition the reverted recovery path required
// (manifest present at this location) before overwriting the SessionStart
// "startup" matcher slot with an unrelated command — never merely an
// unrelated command placed elsewhere (RESEARCH Pitfall 5).
func reproduce242ec0aPrecondition(t *testing.T, target AgentTarget, loc Location, opts InstallOptions) {
	t.Helper()
	pre := target.Install(loc, opts)
	if len(pre.Errors) != 0 {
		t.Fatalf("pre-install to establish claude's own manifest: %v", pre.Errors)
	}
	settingsPath, err := claudeSettingsPath(loc)
	if err != nil {
		t.Fatalf("claudeSettingsPath: %v", err)
	}
	// D-11: next to the unrelated SessionStart "startup" block, an
	// unrelated PreToolUse block under the same "Bash" matcher as
	// codegraph's own — the 242ec0a shape for the opt-in event.
	writeFile(t, settingsPath, `{"hooks":{"SessionStart":[{"matcher":"startup","hooks":[{"type":"command","command":"`+ownershipUnrelatedHookCommand+`"}]}],"PreToolUse":[{"matcher":"Bash","hooks":[{"type":"command","command":"`+ownershipUnrelatedHookCommand+`"}]}]}}`)
}

// assertForeignVariantAfterInstall checks the foreign-codegraph-dir
// variant's Install-side invariants: the oracle dir, if it is one of
// newSkillDirs, must be reported kept (foreign); no manifest may appear at
// ANY newSkillDirs root; and every planted foreign SKILL.md must be
// byte-identical (D-14 — ownership is manifest presence only).
func assertForeignVariantAfterInstall(t *testing.T, result WriteResult, oracle string, skillRoots []string) {
	t.Helper()
	if oracle != "" && containsPath(skillRoots, oracle) {
		found := false
		for _, f := range result.Files {
			if f.Path == oracle && f.Action == ActionKeptForeign {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected {%s, %q} in Install result, got %+v", oracle, ActionKeptForeign, result.Files)
		}
	}
	for _, dir := range skillRoots {
		if fileExists(skillManifestPath(dir)) {
			t.Fatalf("a manifest was created at foreign dir %s — ownership claimed by shape, not manifest presence (D-14)", dir)
		}
		skillPath := filepath.Join(dir, skillFileName)
		if got := readFile(t, skillPath); got != ownershipForeignSkillContent {
			t.Fatalf("foreign SKILL.md at %s was modified: got=%q want=%q", skillPath, got, ownershipForeignSkillContent)
		}
	}
}

// assertCleanVariantAfterInstall checks the clean variant's Install-side
// invariants: a target with an oracle dir wrote the embedded SKILL.md and
// recorded itself in that manifest's targets (the positive "did its work"
// check); a target with no oracle dir (or whose oracle dir is not one of
// newSkillDirs) left no manifest anywhere under newSkillDirs.
func assertCleanVariantAfterInstall(t *testing.T, id TargetID, oracle string, skillRoots []string) {
	t.Helper()
	if oracle != "" {
		skillPath := filepath.Join(oracle, skillFileName)
		want, err := claudeassets.SkillMarkdown()
		if err != nil {
			t.Fatalf("claudeassets.SkillMarkdown: %v", err)
		}
		if got := readFile(t, skillPath); got != string(want) {
			t.Fatalf("oracle SKILL.md at %s does not match the embedded content", skillPath)
		}
		m, present, err := readManifest(skillManifestPath(oracle))
		if err != nil || !present {
			t.Fatalf("expected a manifest at oracle dir %s (present=%v err=%v)", oracle, present, err)
		}
		if !containsTarget(m.Targets, id) {
			t.Fatalf("manifest targets %v at %s does not include %s", m.Targets, oracle, id)
		}
	}
	if oracle == "" || !containsPath(skillRoots, oracle) {
		for _, dir := range skillRoots {
			if fileExists(skillManifestPath(dir)) {
				t.Fatalf("%s does not declare a skill dir at %s, but a manifest was created there", id, dir)
			}
		}
	}
}

// assertForeignBytesUnchangedAfterUninstall is D-13's reversal half: every
// foreign artifact planted before Install must be byte-identical to its
// planted bytes after Install THEN Uninstall — the full round trip, not
// the (necessarily different, since codegraph's own entries are present)
// mid-install state.
func assertForeignBytesUnchangedAfterUninstall(t *testing.T, p foreignPlant) {
	t.Helper()
	if p.hasMCP {
		if got := readFile(t, p.mcpPath); got != p.mcpContent {
			t.Fatalf("foreign MCP config at %s not byte-identical after uninstall:\ngot=%q\nwant=%q", p.mcpPath, got, p.mcpContent)
		}
	}
	if p.hasInstr {
		if got := readFile(t, p.instrPath); got != p.instrContent {
			t.Fatalf("foreign instructions at %s not byte-identical after uninstall:\ngot=%q\nwant=%q", p.instrPath, got, p.instrContent)
		}
	}
	if got := readFile(t, p.siblingPath); got != p.siblingContent {
		t.Fatalf("foreign sibling skill at %s not byte-identical after uninstall:\ngot=%q\nwant=%q", p.siblingPath, got, p.siblingContent)
	}
	for path, content := range p.skillRootFiles {
		if got := readFile(t, path); got != content {
			t.Fatalf("foreign skill-dir file %s not byte-identical after uninstall:\ngot=%q\nwant=%q", path, got, content)
		}
	}
}

// assertOwnEntriesGoneAfterUninstall is D-13's other reversal half: every
// entry CODEGRAPH itself wrote must be gone after Uninstall, format-by-
// format, plus (for Claude specifically) the D-15 precondition's own
// after-the-fact check — the unrelated "startup" hook survives, and no
// SessionStart entry names claudeHookCommand(loc).
func assertOwnEntriesGoneAfterUninstall(t *testing.T, id TargetID, caps Capabilities, loc Location) {
	t.Helper()

	if caps.Supports(loc) && caps.MCPConfig != nil {
		if path, err := caps.MCPConfig(loc); err == nil {
			switch caps.ConfigFormat {
			case ConfigFormatJSON:
				if mcpEntryPresent(path) {
					t.Fatalf("mcpServers.codegraph still present at %s after uninstall", path)
				}
			case ConfigFormatJSONC:
				if opencodeMcpEntryPresent(path) {
					t.Fatalf("mcp.codegraph still present at %s after uninstall", path)
				}
			case ConfigFormatTOML:
				if _, _, found := findTOMLTableRange(readFileOrEmpty(path), codexTOMLTable); found {
					t.Fatalf("%s table still present at %s after uninstall", codexTOMLTable, path)
				}
			case ConfigFormatYAML:
				if hermesConfigured(readFileOrEmpty(path)) {
					t.Fatalf("mcp_servers.codegraph block still present at %s after uninstall", path)
				}
			}
		}
	}

	if caps.Supports(loc) {
		if instrPath, err := caps.InstructionsPath(loc); err == nil && instrPath != "" {
			if strings.Contains(readFileOrEmpty(instrPath), codegraphSectionStart) {
				t.Fatalf("instructions file %s still has codegraph's marker block after uninstall", instrPath)
			}
		}
	}

	if id != Claude || !caps.Supports(loc) {
		return
	}

	settingsPath, err := claudeSettingsPath(loc)
	if err != nil {
		t.Fatalf("claudeSettingsPath: %v", err)
	}
	content := readFileOrEmpty(settingsPath)
	if content == "" {
		t.Fatalf("settings.json missing after uninstall — the unrelated startup hook should have survived")
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("unmarshal post-uninstall settings.json: %v", err)
	}
	wantCmd, err := claudeHookCommand(loc)
	if err != nil {
		t.Fatalf("claudeHookCommand: %v", err)
	}
	hooks, _ := decoded["hooks"].(map[string]any)
	sessionStart, _ := hooks["SessionStart"].([]any)
	sawUnrelated := false
	for _, e := range sessionStart {
		entry, _ := e.(map[string]any)
		matcher, _ := entry["matcher"].(string)
		entries, _ := entry["hooks"].([]any)
		for _, h := range entries {
			hObj, _ := h.(map[string]any)
			cmd, _ := hObj["command"].(string)
			if cmd == wantCmd {
				t.Fatalf("codegraph's own SessionStart command %q still present after uninstall", wantCmd)
			}
			if matcher == "startup" && cmd == ownershipUnrelatedHookCommand {
				sawUnrelated = true
			}
		}
	}
	if !sawUnrelated {
		t.Fatalf("unrelated startup hook missing after uninstall: %#v", sessionStart)
	}

	// D-11 / 242ec0a: the leaf installed with the PreToolUse opt-in on, so
	// codegraph's own PreToolUse handlers and guard must be gone, while the
	// unrelated block planted under the same "Bash" matcher survives
	// deep-equal to what was planted.
	wantPreCmd, err := claudePreToolHookCommand(loc)
	if err != nil {
		t.Fatalf("claudePreToolHookCommand: %v", err)
	}
	guardPath, err := claudePreToolGuardPath(loc)
	if err != nil {
		t.Fatalf("claudePreToolGuardPath: %v", err)
	}
	if _, err := os.Lstat(guardPath); !os.IsNotExist(err) {
		t.Fatalf("PreToolUse guard %s still present after uninstall (Lstat err %v)", guardPath, err)
	}
	var planted any
	if err := json.Unmarshal([]byte(ownershipUnrelatedPreToolUseBlock), &planted); err != nil {
		t.Fatalf("unmarshal planted PreToolUse block: %v", err)
	}
	preToolUse, _ := hooks["PreToolUse"].([]any)
	sawUnrelatedPre := false
	for _, e := range preToolUse {
		if jsonDeepEqual(e, planted) {
			sawUnrelatedPre = true
		}
		entry, _ := e.(map[string]any)
		entries, _ := entry["hooks"].([]any)
		for _, h := range entries {
			hObj, _ := h.(map[string]any)
			if cmd, _ := hObj["command"].(string); cmd == wantPreCmd {
				t.Fatalf("codegraph's own PreToolUse command %q still present after uninstall", wantPreCmd)
			}
		}
	}
	if !sawUnrelatedPre {
		t.Fatalf("unrelated same-matcher PreToolUse block missing or changed after uninstall: %#v", preToolUse)
	}
}

// assertCodexOwnPreToolUseEntriesGoneAfterUninstall (07-08, D-23/242ec0a) is
// assertOwnEntriesGoneAfterUninstall's Codex analog: the leaf planted
// ownershipCodexForeignBashGroup under Codex's own "^Bash$" matcher before
// Install, so after Uninstall that foreign group must survive
// deep-equal to what was planted, no group may carry codegraph's own
// command, and the guard must be gone.
func assertCodexOwnPreToolUseEntriesGoneAfterUninstall(t *testing.T, loc Location) {
	t.Helper()

	guardPath, err := codexPreToolGuardPath(loc)
	if err != nil {
		t.Fatalf("codexPreToolGuardPath: %v", err)
	}
	if _, err := os.Lstat(guardPath); !os.IsNotExist(err) {
		t.Fatalf("Codex PreToolUse guard %s still present after uninstall (Lstat err %v)", guardPath, err)
	}

	hooksPath, err := codexHooksJSONPath(loc)
	if err != nil {
		t.Fatalf("codexHooksJSONPath: %v", err)
	}
	content := readFileOrEmpty(hooksPath)
	if content == "" {
		t.Fatalf("%s missing after uninstall — the unrelated ^Bash$ group should have survived", hooksPath)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(content), &decoded); err != nil {
		t.Fatalf("unmarshal post-uninstall %s: %v", hooksPath, err)
	}
	var planted any
	if err := json.Unmarshal([]byte(ownershipCodexForeignBashGroup), &planted); err != nil {
		t.Fatalf("unmarshal planted codex group: %v", err)
	}
	_, ownCommands, err := codexPreToolUseBlocks(loc)
	if err != nil {
		t.Fatalf("codexPreToolUseBlocks: %v", err)
	}
	hooks, _ := decoded["hooks"].(map[string]any)
	preToolUse, _ := hooks["PreToolUse"].([]any)
	sawForeign := false
	for _, e := range preToolUse {
		if jsonDeepEqual(e, planted) {
			sawForeign = true
		}
		entry, _ := e.(map[string]any)
		entries, _ := entry["hooks"].([]any)
		for _, h := range entries {
			hObj, _ := h.(map[string]any)
			if cmd, _ := hObj["command"].(string); commandIsOwned(cmd, ownCommands) {
				t.Fatalf("codegraph's own Codex PreToolUse command %q still present after uninstall", cmd)
			}
		}
	}
	if !sawForeign {
		t.Fatalf("foreign ^Bash$ group missing or changed after uninstall: %#v", preToolUse)
	}
}

// runOwnershipLeaf executes one <target>/<loc>/<variant> leaf of
// TestOwnershipExactIdentity.
func runOwnershipLeaf(t *testing.T, target AgentTarget, loc Location, variant string) {
	t.Helper()
	home := fakeHome(t)
	if loc == LocationLocal {
		dir := t.TempDir()
		t.Chdir(dir)
	}

	caps := target.Capabilities()
	// D-11: every leaf installs with the PreToolUse opt-in on (a no-op for
	// every target but Claude), so the Claude leaves exercise the opt-in
	// event's exact-identity ownership (242ec0a) as well as SessionStart's.
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph", PreToolNudge: PreToolNudgeOn}
	plant := plantForeignFixtures(t, caps, loc, home, variant == "foreign-codegraph-dir")

	if target.ID() == Claude && caps.Supports(loc) {
		reproduce242ec0aPrecondition(t, target, loc, opts)
	}
	if target.ID() == Codex && caps.Supports(loc) {
		plantForeignCodexHooksGroup(t, loc)
	}

	if !caps.Supports(loc) {
		cwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Getwd: %v", err)
		}
		before := snapshotTree(t, home, cwd)
		first := target.Install(loc, opts)
		if len(first.Files) != 0 || len(first.Errors) != 0 {
			t.Fatalf("unsupported-location Install returned files=%v errors=%v, want both empty", first.Files, first.Errors)
		}
		second := target.Uninstall(loc)
		if len(second.Files) != 0 || len(second.Errors) != 0 {
			t.Fatalf("unsupported-location Uninstall returned files=%v errors=%v, want both empty", second.Files, second.Errors)
		}
		after := snapshotTree(t, home, cwd)
		if before != after {
			t.Fatalf("unsupported-location Install/Uninstall changed the filesystem:\nbefore=%s\nafter=%s", before, after)
		}
		return
	}

	first := target.Install(loc, opts)
	if len(first.Errors) != 0 {
		t.Fatalf("Install returned errors: %v", first.Errors)
	}

	oracle := ownershipWantSkillDir(target.ID(), loc, home)
	skillRoots := newSkillDirs(home, loc)
	if variant == "foreign-codegraph-dir" {
		assertForeignVariantAfterInstall(t, first, oracle, skillRoots)
	} else {
		assertCleanVariantAfterInstall(t, target.ID(), oracle, skillRoots)
	}

	second := target.Uninstall(loc)
	if len(second.Errors) != 0 {
		t.Fatalf("Uninstall returned errors: %v", second.Errors)
	}

	assertForeignBytesUnchangedAfterUninstall(t, plant)
	assertOwnEntriesGoneAfterUninstall(t, target.ID(), caps, loc)
	if target.ID() == Codex {
		// D-23/242ec0a: this leaf planted ownershipCodexForeignBashGroup
		// before Install (above), so the foreign group must survive here —
		// unlike assertOwnEntriesGoneAfterUninstall's generic callers (e.g.
		// TestOwnershipSharedInstructions), which never plant it.
		assertCodexOwnPreToolUseEntriesGoneAfterUninstall(t, loc)
	}

	for _, dir := range skillRoots {
		if fileExists(skillManifestPath(dir)) {
			t.Fatalf("manifest still present at %s after uninstall", dir)
		}
	}
	if oracle != "" {
		if fileExists(skillManifestPath(oracle)) {
			t.Fatalf("manifest still present at oracle dir %s after uninstall", oracle)
		}
	}
}

// TestOwnershipExactIdentity is the D-13 32-leaf table:
// AllTargets() (8) x {global, local} x {foreign-codegraph-dir, clean}.
func TestOwnershipExactIdentity(t *testing.T) {
	targets := AllTargets()
	variants := []string{"clean", "foreign-codegraph-dir"}
	executed := 0
	for _, target := range targets {
		id := target.ID()
		for _, loc := range []Location{LocationGlobal, LocationLocal} {
			for _, variant := range variants {
				t.Run(string(id)+"/"+string(loc)+"/"+variant, func(t *testing.T) {
					executed++
					runOwnershipLeaf(t, target, loc, variant)
				})
			}
		}
	}
	if executed != 32 {
		t.Fatalf("executed %d ownership leaves, want 32", executed)
	}
}

// TestOwnershipSharedInstructions (D-13, D-11, 07-06) extends the
// commit-242ec0a418703c6a4dab45188242149960cda77d exact-identity ownership
// discipline documented above the guard's earlier tests in this file to
// the shared repo-root AGENTS.md: this is a positive-controlled GUARD, not
// a novel assertion — it passes against 07-06's Task 1 implementation
// today, and Family (e2) in 07-MUTATION-LOG.md demonstrates it going RED
// when opencodeTarget.Uninstall's instructionsRequestedElsewhere gate is
// removed (the same "guard must carry a positive assertion that it did
// its work" discipline 242ec0a's revert established, generalized to a
// second sharer of one file rather than a second copy of one hook block).
//
// Plants a foreign MCP entry in BOTH .codex/config.toml and
// opencode.jsonc, a foreign marker-less section in the shared AGENTS.md,
// a foreign sibling skill directory, and — for the foreign-codegraph-dir
// variant — a manifest-less SKILL.md in every newSkillDirs root (planted
// once via codex's plantForeignFixtures call; opencode's call skips
// re-planting the skill roots to avoid a redundant double-write of
// identical bytes). Installs codex and opencode at local scope, uninstalls
// in the named order, and asserts every foreign byte survives untouched,
// no codegraph MCP table/entry remains in either config, no marker block
// remains in AGENTS.md, and no manifest remains at any skill root.
func TestOwnershipSharedInstructions(t *testing.T) {
	orders := []string{"codex_then_opencode", "opencode_then_codex", "target_all"}
	variants := []string{"clean", "foreign-codegraph-dir"}
	executed := 0
	for _, order := range orders {
		order := order
		for _, variant := range variants {
			variant := variant
			t.Run(order+"/"+variant, func(t *testing.T) {
				executed++
				home := fakeHome(t)
				dir := t.TempDir()
				t.Chdir(dir)

				codex := codexTarget{}
				opencode := opencodeTarget{}
				opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

				foreignSkillDirs := variant == "foreign-codegraph-dir"
				codexPlant := plantForeignFixtures(t, codex.Capabilities(), LocationLocal, home, foreignSkillDirs)
				// opencode shares the SAME AGENTS.md and sibling-skill paths
				// at local scope (D-11) — plant only opencode's own MCP
				// config here, never re-plant the shared skill roots a
				// second time.
				opencodePlant := plantForeignFixtures(t, opencode.Capabilities(), LocationLocal, home, false)

				if r := codex.Install(LocationLocal, opts); len(r.Errors) != 0 {
					t.Fatalf("codex install: %v", r.Errors)
				}
				if r := opencode.Install(LocationLocal, opts); len(r.Errors) != 0 {
					t.Fatalf("opencode install: %v", r.Errors)
				}

				switch order {
				case "codex_then_opencode":
					codex.Uninstall(LocationLocal)
					opencode.Uninstall(LocationLocal)
				case "opencode_then_codex":
					opencode.Uninstall(LocationLocal)
					codex.Uninstall(LocationLocal)
				case "target_all":
					for _, target := range AllTargets() {
						target.Uninstall(LocationLocal)
					}
				}

				assertForeignBytesUnchangedAfterUninstall(t, codexPlant)
				assertForeignBytesUnchangedAfterUninstall(t, opencodePlant)
				assertOwnEntriesGoneAfterUninstall(t, Codex, codex.Capabilities(), LocationLocal)
				assertOwnEntriesGoneAfterUninstall(t, Opencode, opencode.Capabilities(), LocationLocal)

				for _, skillDir := range newSkillDirs(home, LocationLocal) {
					if fileExists(skillManifestPath(skillDir)) {
						t.Fatalf("manifest still present at %s after uninstall", skillDir)
					}
				}
			})
		}
	}
	if executed != 6 {
		t.Fatalf("executed %d shared-instructions leaves, want 6", executed)
	}
}

// TestOwnershipExactIdentity_CrossCheckWrittenSkillDir cross-checks
// Capabilities().WrittenSkillDir against the INDEPENDENT ownershipWantSkillDir
// oracle for every target x supported location — the table and the oracle
// cannot drift apart without this failing.
func TestOwnershipExactIdentity_CrossCheckWrittenSkillDir(t *testing.T) {
	for _, target := range AllTargets() {
		id := target.ID()
		for _, loc := range []Location{LocationGlobal, LocationLocal} {
			t.Run(string(id)+"/"+string(loc), func(t *testing.T) {
				home := fakeHome(t)
				if loc == LocationLocal {
					dir := t.TempDir()
					t.Chdir(dir)
				}
				if !target.SupportsLocation(loc) {
					return
				}
				got, err := target.Capabilities().WrittenSkillDir(loc)
				if err != nil {
					t.Fatalf("WrittenSkillDir(%s): %v", loc, err)
				}
				want := ownershipWantSkillDir(id, loc, home)
				if got != want {
					t.Fatalf("WrittenSkillDir(%s) = %q, want oracle %q", loc, got, want)
				}
			})
		}
	}
}
