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
const ownership242ec0aSHA = "242ec0a418703c6a4dab45188242149960cda77d"

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
	case Cursor, Opencode:
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
		return filepath.Join(home, ".gemini", "antigravity-cli", "skills", "codegraph")
	default:
		return ""
	}
}

// newSkillDirs lists every skill directory this PHASE introduces (D-05,
// D-06) at loc: the shared package every target eventually funnels onto,
// plus the two harness-specific directories (Gemini, Kiro) and, at global
// scope only, Antigravity's two documented skill roots (05-CONTEXT.md
// D-06's research correction). These are the ownership guard's planting
// sites for the foreign-codegraph-dir variant, and the sites that must
// never accumulate a manifest for a target that does not write there.
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
	writeFile(t, settingsPath, `{"hooks":{"SessionStart":[{"matcher":"startup","hooks":[{"type":"command","command":"`+ownershipUnrelatedHookCommand+`"}]}]}}`)
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
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}
	plant := plantForeignFixtures(t, caps, loc, home, variant == "foreign-codegraph-dir")

	if target.ID() == Claude && caps.Supports(loc) {
		reproduce242ec0aPrecondition(t, target, loc, opts)
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
