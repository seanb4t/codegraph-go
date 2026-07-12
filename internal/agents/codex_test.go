package agents

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCodex_ID(t *testing.T) {
	c := codexTarget{}
	if c.ID() != Codex {
		t.Fatalf("ID() = %v, want %v", c.ID(), Codex)
	}
}

func TestCodex_SupportsLocation_GlobalOnly(t *testing.T) {
	c := codexTarget{}
	if !c.SupportsLocation(LocationGlobal) {
		t.Fatalf("codex should support global")
	}
	if c.SupportsLocation(LocationLocal) {
		t.Fatalf("codex should NOT support local (D-05a)")
	}
}

func TestCodex_Install_Local_IsUnsupportedNoWrite(t *testing.T) {
	home := fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)
	c := codexTarget{}

	result := c.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(result.Files) != 0 {
		t.Fatalf("expected no files touched for unsupported local install, got %v", result.Files)
	}
	if fileExists(filepath.Join(home, ".codex", "config.toml")) {
		t.Fatalf("global config.toml should not have been written by a local install call")
	}
}

func TestCodex_GlobalInstall_WritesTOMLTableAndInstructions(t *testing.T) {
	home := fakeHome(t)
	c := codexTarget{}

	result := c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(result.Files) == 0 {
		t.Fatalf("expected files touched, got none")
	}

	configPath := filepath.Join(home, ".codex", "config.toml")
	got := readFile(t, configPath)
	if !strings.Contains(got, "[mcp_servers.codegraph]") ||
		!strings.Contains(got, `command = "/usr/local/bin/codegraph"`) ||
		!strings.Contains(got, `args = ["serve", "--mcp"]`) {
		t.Fatalf("unexpected config.toml content: %s", got)
	}

	instrPath := filepath.Join(home, ".codex", "AGENTS.md")
	instr := readFile(t, instrPath)
	if !strings.Contains(instr, codegraphSectionStart) || !strings.Contains(instr, "codegraph_explore") {
		t.Fatalf("unexpected AGENTS.md content: %s", instr)
	}
}

func TestCodex_GlobalInstall_PreservesUnrelatedTOMLTable(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".codex", "config.toml")
	pre := "[some_other_table]\n" + `key = "value"` + "\n"
	writeFile(t, configPath, pre)

	c := codexTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	got := readFile(t, configPath)
	if !strings.Contains(got, "[some_other_table]") || !strings.Contains(got, `key = "value"`) {
		t.Fatalf("unrelated table not preserved: %s", got)
	}
	if !strings.Contains(got, "[mcp_servers.codegraph]") {
		t.Fatalf("codegraph table missing: %s", got)
	}
}

func TestCodex_GlobalRoundTrip_ByteInvariant(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".codex", "config.toml")
	pre := "[some_other_table]\n" + `key = "value"` + "\n"
	writeFile(t, configPath, pre)

	c := codexTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	c.Uninstall(LocationGlobal)

	got := readFile(t, configPath)
	if got != pre {
		t.Fatalf("round trip not byte-invariant:\ngot=%q\nwant=%q", got, pre)
	}

	instrPath := filepath.Join(home, ".codex", "AGENTS.md")
	if fileExists(instrPath) {
		t.Fatalf("AGENTS.md should have been removed entirely on uninstall (never existed pre-install)")
	}
}

func TestCodex_Install_ReRunIsByteIdempotent(t *testing.T) {
	home := fakeHome(t)
	c := codexTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}

	c.Install(LocationGlobal, opts)
	configBefore := readFile(t, filepath.Join(home, ".codex", "config.toml"))
	instrBefore := readFile(t, filepath.Join(home, ".codex", "AGENTS.md"))

	c.Install(LocationGlobal, opts)
	configAfter := readFile(t, filepath.Join(home, ".codex", "config.toml"))
	instrAfter := readFile(t, filepath.Join(home, ".codex", "AGENTS.md"))

	if configBefore != configAfter {
		t.Fatalf("config.toml changed on idempotent re-run:\nbefore=%q\nafter=%q", configBefore, configAfter)
	}
	if instrBefore != instrAfter {
		t.Fatalf("AGENTS.md changed on idempotent re-run:\nbefore=%q\nafter=%q", instrBefore, instrAfter)
	}
}

func TestCodex_Uninstall_MissingConfigIsNotFoundNoError(t *testing.T) {
	fakeHome(t)
	c := codexTarget{}
	result := c.Uninstall(LocationGlobal)
	// Must not panic/error; files list should reflect not-found/kept status.
	for _, fr := range result.Files {
		if fr.Action == ActionUpdated || fr.Action == ActionRemoved {
			t.Fatalf("expected no removed/updated actions against a never-installed config, got %+v", fr)
		}
	}
}

func TestCodex_Detect_AfterInstallReportsConfigured(t *testing.T) {
	fakeHome(t)
	c := codexTarget{}
	c.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	got := c.Detect(LocationGlobal)
	if !got.AlreadyConfigured {
		t.Fatalf("expected AlreadyConfigured after install, got %+v", got)
	}
}

func TestCodex_DescribePaths_Global(t *testing.T) {
	c := codexTarget{}
	paths := c.DescribePaths(LocationGlobal)
	if len(paths) < 2 {
		t.Fatalf("expected at least config + instructions paths, got %v", paths)
	}
}

func TestCodex_DescribePaths_LocalEmpty(t *testing.T) {
	c := codexTarget{}
	paths := c.DescribePaths(LocationLocal)
	if len(paths) != 0 {
		t.Fatalf("expected no paths for unsupported local location, got %v", paths)
	}
}
