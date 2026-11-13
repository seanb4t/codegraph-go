package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/realpath/realpathtest"
)

const symlinkTestExec = "/usr/local/bin/codegraph"

var symlinkTestOpts = InstallOptions{ExecPath: symlinkTestExec}

const otherServerJSON = `{"mcpServers": {"other-server": {"command": "other-binary"}}}`

// --- Global scope -----------------------------------------------------

func TestInstall_GlobalConfigSymlinkIsWrittenThrough(t *testing.T) {
	home := realHome(t)
	target := filepath.Join(home, "dotfiles", "claude.json")
	link := filepath.Join(home, ".claude.json")
	writeFile(t, target, otherServerJSON)
	realpathtest.Symlink(t, target, link)

	result := claudeTarget{}.Install(LocationGlobal, symlinkTestOpts)

	assertNoErrors(t, result)
	assertSymlinkTo(t, link, target)
	got := readFile(t, target)
	if !strings.Contains(got, `"codegraph"`) || !strings.Contains(got, "other-server") {
		t.Fatalf("the target does not hold both servers: %s", got)
	}
	if fileAction(result, link) == "" {
		t.Fatalf("result.Files = %v, want an entry under the configured path %s", result.Files, link)
	}
}

func TestInstall_GlobalConfigSymlinkChain(t *testing.T) {
	home := realHome(t)
	target := filepath.Join(home, "dotfiles", "gemini.json")
	link := filepath.Join(home, ".gemini", "settings.json")
	link2 := filepath.Join(home, ".gemini", "link2")
	writeFile(t, target, `{}`)
	realpathtest.Symlink(t, target, link2)
	realpathtest.Symlink(t, link2, link)

	result := geminiTarget{}.Install(LocationGlobal, symlinkTestOpts)

	assertNoErrors(t, result)
	assertSymlinkTo(t, link, link2)
	assertSymlinkTo(t, link2, target)
	if got := readFile(t, target); !strings.Contains(got, `"codegraph"`) {
		t.Fatalf("the target does not hold the codegraph entry: %s", got)
	}
}

func TestInstall_GlobalConfigSymlinkToMissingTarget(t *testing.T) {
	home := realHome(t)
	target := filepath.Join(home, "dotfiles", "claude.json")
	link := filepath.Join(home, ".claude.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	realpathtest.Symlink(t, target, link)

	result := claudeTarget{}.Install(LocationGlobal, symlinkTestOpts)

	assertNoErrors(t, result)
	assertSymlinkTo(t, link, target)
	if got := readFile(t, target); !strings.Contains(got, `"codegraph"`) {
		t.Fatalf("the target does not hold the codegraph entry: %s", got)
	}
}

func TestInstall_GlobalRelativeLinkUnderLinkedDirectory(t *testing.T) {
	home := realHome(t)
	dotClaude := filepath.Join(home, "dotfiles", "claude")
	if err := os.MkdirAll(dotClaude, 0o755); err != nil {
		t.Fatal(err)
	}
	realpathtest.Symlink(t, dotClaude, filepath.Join(home, ".claude"))
	relTarget := filepath.Join("..", "shared", "CLAUDE.md")
	realpathtest.Symlink(t, relTarget, filepath.Join(dotClaude, "CLAUDE.md"))

	result := claudeTarget{}.Install(LocationGlobal, symlinkTestOpts)

	assertNoErrors(t, result)
	if got := readFile(t, filepath.Join(home, "dotfiles", "shared", "CLAUDE.md")); !strings.Contains(got, codegraphSectionStart) {
		t.Fatalf("dotfiles/shared/CLAUDE.md does not hold the section: %s", got)
	}
	assertMissing(t, filepath.Join(home, "shared", "CLAUDE.md"))
	assertSymlinkTo(t, filepath.Join(dotClaude, "CLAUDE.md"), relTarget)
}

func TestInstall_GlobalConfigSymlinkKeepsTargetMode(t *testing.T) {
	home := realHome(t)
	target := filepath.Join(home, "dotfiles", "claude.json")
	link := filepath.Join(home, ".claude.json")
	writeFile(t, target, `{}`)
	if err := os.Chmod(target, 0o640); err != nil {
		t.Fatal(err)
	}
	realpathtest.Symlink(t, target, link)

	result := claudeTarget{}.Install(LocationGlobal, symlinkTestOpts)

	assertNoErrors(t, result)
	if got := readFile(t, target); !strings.Contains(got, `"codegraph"`) {
		t.Fatalf("the target does not hold the codegraph entry: %s", got)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("the target has mode %o, want 640", info.Mode().Perm())
	}
}

func TestUninstall_GlobalConfigSymlinkIsKept(t *testing.T) {
	home := realHome(t)
	target := filepath.Join(home, "dotfiles", "claude.json")
	link := filepath.Join(home, ".claude.json")
	writeFile(t, target, otherServerJSON)
	realpathtest.Symlink(t, target, link)
	assertNoErrors(t, claudeTarget{}.Install(LocationGlobal, symlinkTestOpts))

	result := claudeTarget{}.Uninstall(LocationGlobal)

	assertNoErrors(t, result)
	assertSymlinkTo(t, link, target)
	got := readFile(t, target)
	if strings.Contains(got, `"codegraph"`) || !strings.Contains(got, "other-server") {
		t.Fatalf("the target does not hold only the other server: %s", got)
	}
}

func TestUninstall_GlobalInstructionSymlinkWithOnlyTheSectionKeepsLink(t *testing.T) {
	home := realHome(t)
	target := filepath.Join(home, "dotfiles", "CLAUDE.md")
	link := filepath.Join(home, ".claude", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	realpathtest.Symlink(t, target, link)
	assertNoErrors(t, claudeTarget{}.Install(LocationGlobal, symlinkTestOpts))
	if got := readFile(t, target); !strings.Contains(got, codegraphSectionStart) {
		t.Fatalf("the install did not write the section through the link: %s", got)
	}

	result := claudeTarget{}.Uninstall(LocationGlobal)

	assertNoErrors(t, result)
	assertMissing(t, target)
	assertSymlinkTo(t, link, target)
}

func TestUninstall_GlobalSettingsSymlinkWithOnlyCodegraphHooksKeepsLink(t *testing.T) {
	home := realHome(t)
	target := filepath.Join(home, "dotfiles", "settings.json")
	link := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	realpathtest.Symlink(t, target, link)
	assertNoErrors(t, claudeTarget{}.Install(LocationGlobal, symlinkTestOpts))
	if got := readFile(t, target); !strings.Contains(got, "SessionStart") {
		t.Fatalf("the install did not write the hook through the link: %s", got)
	}

	result := claudeTarget{}.Uninstall(LocationGlobal)

	assertNoErrors(t, result)
	assertMissing(t, target)
	assertSymlinkTo(t, link, target)
}

func TestUninstall_GlobalCodexConfigSymlinkWithOnlyTheTableKeepsLink(t *testing.T) {
	home := realHome(t)
	target := filepath.Join(home, "dotfiles", "config.toml")
	link := filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	realpathtest.Symlink(t, target, link)
	assertNoErrors(t, codexTarget{}.Install(LocationGlobal, symlinkTestOpts))
	if got := readFile(t, target); !strings.Contains(got, "codegraph") {
		t.Fatalf("the install did not write the table through the link: %s", got)
	}

	result := codexTarget{}.Uninstall(LocationGlobal)

	assertNoErrors(t, result)
	assertMissing(t, target)
	assertSymlinkTo(t, link, target)
}

// --- Local scope ------------------------------------------------------

func TestInstall_LocalInstructionSymlinkInsideProjectIsWrittenThrough(t *testing.T) {
	projectAndOutside(t)
	realpathtest.Symlink(t, "AGENTS.md", "GEMINI.md")

	result := geminiTarget{}.Install(LocationLocal, symlinkTestOpts)

	assertNoErrors(t, result)
	assertSymlinkTo(t, "GEMINI.md", "AGENTS.md")
	if got := readFile(t, "AGENTS.md"); !strings.Contains(got, codegraphSectionStart) {
		t.Fatalf("AGENTS.md does not hold the section: %s", got)
	}
}

func TestInstall_LocalSymlinkToDotDotPrefixedNameIsWrittenThrough(t *testing.T) {
	projectAndOutside(t)
	realpathtest.Symlink(t, "..notes.md", "AGENTS.md")

	result := codexTarget{}.Install(LocationLocal, symlinkTestOpts)

	assertNoErrors(t, result)
	assertSymlinkTo(t, "AGENTS.md", "..notes.md")
	if got := readFile(t, "..notes.md"); !strings.Contains(got, codegraphSectionStart) {
		t.Fatalf("..notes.md does not hold the section: %s", got)
	}
}

func TestInstall_LocalInstructionSymlinkOutsideProjectIsRefused(t *testing.T) {
	_, outside := projectAndOutside(t)
	target := filepath.Join(outside, ".zshrc")
	const content = "export PATH=/bin\n"
	writeFile(t, target, content)
	realpathtest.Symlink(t, target, "AGENTS.md")

	result := codexTarget{}.Install(LocationLocal, symlinkTestOpts)

	if !errorNaming(result, "AGENTS.md", target) {
		t.Fatalf("errors %v do not name AGENTS.md and %s", result.Errors, target)
	}
	if got := readFile(t, target); got != content {
		t.Fatalf("the outside file changed: %q", got)
	}
	assertSymlinkTo(t, "AGENTS.md", target)
}

func TestInstall_LocalConfigSymlinkOutsideProjectIsRefused(t *testing.T) {
	_, outside := projectAndOutside(t)
	target := filepath.Join(outside, ".claude.json")
	writeFile(t, target, otherServerJSON)
	realpathtest.Symlink(t, target, ".mcp.json")

	result := claudeTarget{}.Install(LocationLocal, symlinkTestOpts)

	if !errorNaming(result, ".mcp.json", target) {
		t.Fatalf("errors %v do not name .mcp.json and %s", result.Errors, target)
	}
	if got := readFile(t, target); got != otherServerJSON {
		t.Fatalf("the outside file changed: %q", got)
	}
	assertSymlinkTo(t, ".mcp.json", target)
}

func TestInstall_LocalConfigSymlinkOutsideWithTheEntryIsRefused(t *testing.T) {
	_, outside := projectAndOutside(t)
	target := filepath.Join(outside, ".claude.json")
	if _, err := writeMcpEntry(target, func() any { return stdioMcpEntry(symlinkTestExec, "serve", "--mcp") }); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, target)
	realpathtest.Symlink(t, target, ".mcp.json")

	result := claudeTarget{}.Install(LocationLocal, symlinkTestOpts)

	if !errorNaming(result, ".mcp.json", target) {
		t.Fatalf("errors %v do not name .mcp.json and %s", result.Errors, target)
	}
	if got := readFile(t, target); got != before {
		t.Fatalf("the outside file changed: %q", got)
	}
	assertSymlinkTo(t, ".mcp.json", target)
}

func TestInstall_LocalSymlinkOutsideThroughLinkedDirectoryIsRefused(t *testing.T) {
	_, outside := projectAndOutside(t)
	notes := filepath.Join(outside, "notes.md")
	const content = "my notes\n"
	writeFile(t, notes, content)
	realpathtest.Symlink(t, outside, "docs")
	linkTarget := filepath.Join("docs", "notes.md")
	realpathtest.Symlink(t, linkTarget, "AGENTS.md")

	result := codexTarget{}.Install(LocationLocal, symlinkTestOpts)

	if !errorNaming(result, "AGENTS.md", notes) {
		t.Fatalf("errors %v do not name AGENTS.md and %s", result.Errors, notes)
	}
	if got := readFile(t, notes); got != content {
		t.Fatalf("the outside file changed: %q", got)
	}
	assertSymlinkTo(t, "AGENTS.md", linkTarget)
}

func TestInstall_LocalSymlinkThatClimbsOutOfALinkedDirectoryIsRefused(t *testing.T) {
	project, outside := projectAndOutside(t)
	if err := os.Mkdir(filepath.Join(outside, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	realpathtest.Symlink(t, filepath.Join(outside, "a"), "sub")
	realpathtest.Symlink(t, "sub/../x.md", "AGENTS.md")

	result := codexTarget{}.Install(LocationLocal, symlinkTestOpts)

	if !errorNaming(result, "AGENTS.md", filepath.Join(outside, "x.md")) {
		t.Fatalf("errors %v do not name AGENTS.md and the real location", result.Errors)
	}
	assertMissing(t, filepath.Join(outside, "x.md"))
	assertMissing(t, filepath.Join(project, "x.md"))
	assertSymlinkTo(t, "AGENTS.md", "sub/../x.md")
}

func TestInstall_LocalUserFilesUnderDirectoryLinkedOutsideAreRefused(t *testing.T) {
	_, outside := projectAndOutside(t)
	realpathtest.Symlink(t, outside, ".claude")

	result := claudeTarget{}.Install(LocationLocal, symlinkTestOpts)

	for _, path := range []string{filepath.Join(".claude", "settings.json"), filepath.Join(".claude", "CLAUDE.md")} {
		if !errorNaming(result, path, outside) {
			t.Errorf("errors %v do not name %s and its real location", result.Errors, path)
		}
	}
	assertMissing(t, filepath.Join(outside, "settings.json"))
	assertMissing(t, filepath.Join(outside, "CLAUDE.md"))
	assertSymlinkTo(t, ".claude", outside)
}

func TestUninstall_LocalSymlinkOutsideProjectIsKeptWithANote(t *testing.T) {
	_, outside := projectAndOutside(t)
	target := filepath.Join(outside, "agents.md")
	if _, err := upsertInstructionsEntry(target, codegraphSectionStart, codegraphSectionEnd, instructionsBody()); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, target)
	realpathtest.Symlink(t, target, "AGENTS.md")

	result := codexTarget{}.Uninstall(LocationLocal)

	assertNoErrors(t, result)
	if got := readFile(t, target); got != before {
		t.Fatalf("the outside file changed: %q", got)
	}
	assertSymlinkTo(t, "AGENTS.md", target)
	assertFileResult(t, result.Files, "AGENTS.md", ActionKept)
	if !noteNaming(result, "AGENTS.md") {
		t.Fatalf("notes %v do not name AGENTS.md", result.Notes)
	}
}

func TestUninstall_LocalConfigSymlinkOutsideProjectIsKeptWithANote(t *testing.T) {
	_, outside := projectAndOutside(t)
	target := filepath.Join(outside, ".claude.json")
	if _, err := writeMcpEntry(target, func() any { return stdioMcpEntry(symlinkTestExec, "serve", "--mcp") }); err != nil {
		t.Fatal(err)
	}
	before := readFile(t, target)
	realpathtest.Symlink(t, target, ".mcp.json")

	result := claudeTarget{}.Uninstall(LocationLocal)

	assertNoErrors(t, result)
	if got := readFile(t, target); got != before {
		t.Fatalf("the outside file changed: %q", got)
	}
	assertSymlinkTo(t, ".mcp.json", target)
	assertFileResult(t, result.Files, ".mcp.json", ActionKept)
	if !noteNaming(result, ".mcp.json") {
		t.Fatalf("notes %v do not name .mcp.json", result.Notes)
	}
}

// --- Two agents that share one file ----------------------------------

func TestUninstall_Gemini_LinkedInstructionFileKeepsTheSectionCodexUses(t *testing.T) {
	projectAndOutside(t)
	realpathtest.Symlink(t, "AGENTS.md", "GEMINI.md")
	assertNoErrors(t, codexTarget{}.Install(LocationLocal, symlinkTestOpts))
	assertNoErrors(t, geminiTarget{}.Install(LocationLocal, symlinkTestOpts))

	result := geminiTarget{}.Uninstall(LocationLocal)

	assertNoErrors(t, result)
	if got := readFile(t, "AGENTS.md"); !strings.Contains(got, codegraphSectionStart) {
		t.Fatalf("AGENTS.md lost the section that codex still uses: %s", got)
	}
	assertFileResult(t, result.Files, "GEMINI.md", ActionKept)
	assertSymlinkTo(t, "GEMINI.md", "AGENTS.md")
}

func TestUninstall_Claude_LinkedInstructionFileKeepsTheSectionCodexUses(t *testing.T) {
	project, _ := projectAndOutside(t)
	agents := filepath.Join(project, "AGENTS.md")
	instr := filepath.Join(".claude", "CLAUDE.md")
	realpathtest.Symlink(t, agents, instr)
	assertNoErrors(t, codexTarget{}.Install(LocationLocal, symlinkTestOpts))
	assertNoErrors(t, claudeTarget{}.Install(LocationLocal, symlinkTestOpts))

	result := claudeTarget{}.Uninstall(LocationLocal)

	assertNoErrors(t, result)
	if got := readFile(t, agents); !strings.Contains(got, codegraphSectionStart) {
		t.Fatalf("AGENTS.md lost the section that codex still uses: %s", got)
	}
	assertFileResult(t, result.Files, instr, ActionKept)
	assertSymlinkTo(t, instr, agents)
}

// --- Loops ------------------------------------------------------------

// loop makes two links that point at each other and returns the second.
func loop(t *testing.T, path string) string {
	t.Helper()
	other := filepath.Join(filepath.Dir(path), "a")
	realpathtest.Symlink(t, other, path)
	realpathtest.Symlink(t, path, other)
	return other
}

func TestInstall_SymlinkLoopReportsAnErrorAndWritesNothing(t *testing.T) {
	home := realHome(t)
	path := filepath.Join(home, ".claude.json")
	other := loop(t, path)

	result := claudeTarget{}.Install(LocationGlobal, symlinkTestOpts)

	if !errorNaming(result, path) {
		t.Fatalf("errors %v do not name %s", result.Errors, path)
	}
	assertSymlinkTo(t, path, other)
	assertSymlinkTo(t, other, path)
}

func TestUninstall_SymlinkLoopReportsAnError(t *testing.T) {
	home := realHome(t)
	path := filepath.Join(home, ".codex", "config.toml")
	other := loop(t, path)

	result := codexTarget{}.Uninstall(LocationGlobal)

	if !errorNaming(result, path) {
		t.Fatalf("errors %v do not name %s", result.Errors, path)
	}
	assertSymlinkTo(t, path, other)
	assertSymlinkTo(t, other, path)
}

func TestUninstall_Hermes_SymlinkLoopReportsAnError(t *testing.T) {
	home := realHome(t)
	path := filepath.Join(home, ".hermes", "config.yaml")
	other := loop(t, path)

	result := hermesTarget{}.Uninstall(LocationGlobal)

	if !errorNaming(result, path) {
		t.Fatalf("errors %v do not name %s", result.Errors, path)
	}
	assertSymlinkTo(t, path, other)
	assertSymlinkTo(t, other, path)
}

// --- A file that does not parse --------------------------------------

func TestInstall_GlobalConfigSymlinkToUnparseableTargetIsUntouched(t *testing.T) {
	home := realHome(t)
	target := filepath.Join(home, "dotfiles", "claude.json")
	link := filepath.Join(home, ".claude.json")
	const content = `{"mcpServers": {"other-server": {"command": "other-binary"}},}`
	writeFile(t, target, content)
	realpathtest.Symlink(t, target, link)

	result := claudeTarget{}.Install(LocationGlobal, symlinkTestOpts)

	if !errorNaming(result, target) {
		t.Fatalf("errors %v do not name the real file %s", result.Errors, target)
	}
	if got := readFile(t, target); got != content {
		t.Fatalf("the target changed: %q", got)
	}
	assertSymlinkTo(t, link, target)
}

func TestOpencode_EmptyAndWhitespaceConfigIsWritten(t *testing.T) {
	for name, content := range map[string]string{"empty": "", "whitespace": " \n\t\n"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "opencode.jsonc")
			writeFile(t, path, content)

			if _, err := writeOpencodeEntry(path, symlinkTestExec); err != nil {
				t.Fatalf("writeOpencodeEntry: %v", err)
			}
			if got := readFile(t, path); !strings.Contains(got, "codegraph") {
				t.Fatalf("the file does not hold the codegraph entry: %q", got)
			}
		})
	}
}

// --- The Antigravity migration ---------------------------------------

const legacyAntigravityJSON = `{"mcpServers": {"codegraph": {"command": "old-codegraph"}, "other-server": {"command": "other-binary"}}}`

func TestAntigravity_MigrationKeepsADanglingLinkAtTheUnifiedPath(t *testing.T) {
	home := realHome(t)
	unified := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	target := filepath.Join(home, "dotfiles", "mcp_config.json")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	realpathtest.Symlink(t, target, unified)
	writeFile(t, filepath.Join(home, ".gemini", "antigravity", "mcp_config.json"), legacyAntigravityJSON)

	result := antigravityTarget{}.Install(LocationGlobal, symlinkTestOpts)

	assertNoErrors(t, result)
	assertSymlinkTo(t, unified, target)
	if got := readFile(t, target); !strings.Contains(got, symlinkTestExec) {
		t.Fatalf("the file behind the unified link does not hold the current entry: %s", got)
	}
}

func TestAntigravity_MigrationKeepsALinkAtTheLegacyPath(t *testing.T) {
	home := realHome(t)
	legacy := filepath.Join(home, ".gemini", "antigravity", "mcp_config.json")
	target := filepath.Join(home, "dotfiles", "legacy.json")
	writeFile(t, target, legacyAntigravityJSON)
	realpathtest.Symlink(t, target, legacy)

	result := antigravityTarget{}.Install(LocationGlobal, symlinkTestOpts)

	assertNoErrors(t, result)
	assertSymlinkTo(t, legacy, target)
	got := readFile(t, target)
	if strings.Contains(got, "codegraph") || !strings.Contains(got, "other-server") {
		t.Fatalf("the file behind the legacy link does not hold only the other server: %s", got)
	}
	assertFileResult(t, result.Files, legacy, ActionRemoved)
	unified := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	if got := readFile(t, unified); !strings.Contains(got, symlinkTestExec) {
		t.Fatalf("the unified file does not hold the current entry: %s", got)
	}
}

func TestAntigravity_MigrationSymlinkLoopAtTheUnifiedPathReportsAnError(t *testing.T) {
	home := realHome(t)
	unified := filepath.Join(home, ".gemini", "config", "mcp_config.json")
	other := loop(t, unified)

	result := antigravityTarget{}.Install(LocationGlobal, symlinkTestOpts)

	if !errorNaming(result, unified) {
		t.Fatalf("errors %v do not name %s", result.Errors, unified)
	}
	assertSymlinkTo(t, unified, other)
	assertMissing(t, filepath.Join(home, ".gemini", "config", ".migrated"))
}

// --- Review findings ----------------------------------------------------

// A stale APPDATA path can be a link to the live OpenCode file. The sweep
// must not remove the entry that the same install just wrote there.
func TestOpencode_AppDataSweep_SkipsALinkToTheLiveConfig(t *testing.T) {
	home := realHome(t)
	live := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	appData := filepath.Join(home, "AppData")
	t.Setenv("APPDATA", appData)
	stale := filepath.Join(appData, "opencode", "opencode.jsonc")
	realpathtest.Symlink(t, live, stale)

	result := opencodeTarget{}.Install(LocationGlobal, symlinkTestOpts)

	assertNoErrors(t, result)
	if got := readFile(t, live); !strings.Contains(got, `"codegraph"`) {
		t.Fatalf("the sweep removed the entry from the live file: %s", got)
	}
	assertSymlinkTo(t, stale, live)
}

// An install that turns the Codex nudge off edits hooks.json. It is an
// install, so a local hooks.json outside the project is an error.
func TestInstall_Codex_NudgeOffRefusesHooksFileOutsideProject(t *testing.T) {
	_, outside := projectAndOutside(t)
	target := filepath.Join(outside, "hooks.json")
	const content = `{"hooks": {}}`
	writeFile(t, target, content)
	hooks := filepath.Join(".codex", "hooks.json")
	realpathtest.Symlink(t, target, hooks)

	result := codexTarget{}.Install(LocationLocal, InstallOptions{ExecPath: symlinkTestExec, PreToolNudge: PreToolNudgeOff})

	if !errorNaming(result, hooks, target) {
		t.Fatalf("errors %v do not name %s and %s", result.Errors, hooks, target)
	}
	if got := readFile(t, target); got != content {
		t.Fatalf("the outside file changed: %q", got)
	}
	assertSymlinkTo(t, hooks, target)
}
