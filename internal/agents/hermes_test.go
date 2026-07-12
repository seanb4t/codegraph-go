package agents

import (
	"path/filepath"
	"strings"
	"testing"
)

const hermesPyYAMLDefaultFixture = `log_level: info
platform_toolsets:
  cli:
  - shell
  - browser
`

const hermesDeeperIndentFixture = `log_level: info
platform_toolsets:
  cli:
    - shell
    - browser
`

func TestHermes_ID(t *testing.T) {
	h := hermesTarget{}
	if h.ID() != Hermes {
		t.Fatalf("ID() = %v, want %v", h.ID(), Hermes)
	}
}

func TestHermes_SupportsLocation_GlobalOnly(t *testing.T) {
	h := hermesTarget{}
	if !h.SupportsLocation(LocationGlobal) {
		t.Fatalf("hermes should support global")
	}
	if h.SupportsLocation(LocationLocal) {
		t.Fatalf("hermes should NOT support local")
	}
}

func TestHermes_Install_Local_IsUnsupportedNoWrite(t *testing.T) {
	home := fakeHome(t)
	dir := t.TempDir()
	t.Chdir(dir)
	h := hermesTarget{}

	result := h.Install(LocationLocal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	if len(result.Files) != 0 {
		t.Fatalf("expected no files touched for unsupported local install, got %v", result.Files)
	}
	if fileExists(filepath.Join(home, ".hermes", "config.yaml")) {
		t.Fatalf("global config.yaml should not have been written by a local install call")
	}
}

func TestHermes_ConfigPath_UsesHermesHomeWhenSet(t *testing.T) {
	home := fakeHome(t)
	custom := filepath.Join(home, "custom-hermes")
	t.Setenv("HERMES_HOME", custom)

	got, err := hermesConfigPath()
	if err != nil {
		t.Fatalf("hermesConfigPath: %v", err)
	}
	want := filepath.Join(custom, "config.yaml")
	if got != want {
		t.Fatalf("hermesConfigPath() = %q, want %q", got, want)
	}
}

func TestHermes_ConfigPath_DefaultsToDotHermesWhenUnset(t *testing.T) {
	home := fakeHome(t)
	t.Setenv("HERMES_HOME", "")

	got, err := hermesConfigPath()
	if err != nil {
		t.Fatalf("hermesConfigPath: %v", err)
	}
	want := filepath.Join(home, ".hermes", "config.yaml")
	if got != want {
		t.Fatalf("hermesConfigPath() = %q, want %q", got, want)
	}
}

func TestHermes_GlobalInstall_WritesMcpServersBlock_PreservesUnrelatedContent(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".hermes", "config.yaml")
	writeFile(t, configPath, hermesPyYAMLDefaultFixture)

	h := hermesTarget{}
	h.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	got := readFile(t, configPath)
	if !strings.Contains(got, "log_level: info") {
		t.Fatalf("unrelated top-level key lost: %s", got)
	}
	if !strings.Contains(got, "mcp_servers:") || !strings.Contains(got, "codegraph:") {
		t.Fatalf("mcp_servers.codegraph block missing: %s", got)
	}
	if !strings.Contains(got, "/usr/local/bin/codegraph") {
		t.Fatalf("command value missing: %s", got)
	}
	if !strings.Contains(got, "- serve") || !strings.Contains(got, "- --mcp") {
		t.Fatalf("args list missing: %s", got)
	}
	if !strings.Contains(got, "timeout: 120") || !strings.Contains(got, "connect_timeout: 60") {
		t.Fatalf("timeout fields missing: %s", got)
	}
}

func TestHermes_Install_AppendsCliToolset_PyYAMLDefaultIndent(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".hermes", "config.yaml")
	writeFile(t, configPath, hermesPyYAMLDefaultFixture)

	h := hermesTarget{}
	h.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	got := readFile(t, configPath)
	// PyYAML-default: list items sit at the SAME indent as "cli:" (2
	// spaces), not 4 — the appended item must match that, not a
	// hardcoded deeper indent (Pitfall 5).
	if !strings.Contains(got, "\n  - mcp-codegraph\n") && !strings.HasSuffix(got, "\n  - mcp-codegraph\n") {
		t.Fatalf("appended cli item not at parent (2-space) indent:\n%s", got)
	}
	if strings.Contains(got, "\n    - mcp-codegraph") {
		t.Fatalf("appended cli item incorrectly used a hardcoded deeper indent:\n%s", got)
	}
	if !strings.Contains(got, "- shell") || !strings.Contains(got, "- browser") {
		t.Fatalf("existing cli items lost: %s", got)
	}
}

func TestHermes_Install_AppendsCliToolset_HandAuthoredDeeperIndent(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".hermes", "config.yaml")
	writeFile(t, configPath, hermesDeeperIndentFixture)

	h := hermesTarget{}
	h.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	got := readFile(t, configPath)
	// Hand-authored fixture uses a 4-space list-item indent — the
	// appended item must match THAT existing indent, not assume 2.
	if !strings.Contains(got, "\n    - mcp-codegraph\n") && !strings.HasSuffix(got, "\n    - mcp-codegraph\n") {
		t.Fatalf("appended cli item did not match existing deeper (4-space) indent:\n%s", got)
	}
}

func TestHermes_Install_CliToolsetIdempotent_NoDuplicateOnReRun(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".hermes", "config.yaml")
	writeFile(t, configPath, hermesPyYAMLDefaultFixture)

	h := hermesTarget{}
	opts := InstallOptions{ExecPath: "/usr/local/bin/codegraph"}
	h.Install(LocationGlobal, opts)
	afterFirst := readFile(t, configPath)
	if strings.Count(afterFirst, "mcp-codegraph") != 1 {
		t.Fatalf("want exactly one mcp-codegraph cli entry after first install, got: %s", afterFirst)
	}

	h.Install(LocationGlobal, opts)
	afterSecond := readFile(t, configPath)
	if strings.Count(afterSecond, "mcp-codegraph") != 1 {
		t.Fatalf("re-run duplicated the cli toolset entry: %s", afterSecond)
	}
	if afterFirst != afterSecond {
		t.Fatalf("re-run not byte-idempotent:\nfirst=%q\nsecond=%q", afterFirst, afterSecond)
	}
}

func TestHermes_GlobalRoundTrip_ByteInvariant(t *testing.T) {
	home := fakeHome(t)
	configPath := filepath.Join(home, ".hermes", "config.yaml")
	pre := hermesPyYAMLDefaultFixture
	writeFile(t, configPath, pre)

	h := hermesTarget{}
	h.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	h.Uninstall(LocationGlobal)

	got := readFile(t, configPath)
	if got != pre {
		t.Fatalf("round trip not byte-invariant:\ngot=%q\nwant=%q", got, pre)
	}
}

func TestHermes_Uninstall_MissingConfigNoError(t *testing.T) {
	fakeHome(t)
	h := hermesTarget{}
	result := h.Uninstall(LocationGlobal)
	for _, fr := range result.Files {
		if fr.Action == ActionUpdated || fr.Action == ActionRemoved {
			t.Fatalf("expected no removed/updated actions against a never-installed config, got %+v", fr)
		}
	}
}

func TestHermes_Detect_AfterInstallReportsConfigured(t *testing.T) {
	fakeHome(t)
	h := hermesTarget{}
	h.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})
	got := h.Detect(LocationGlobal)
	if !got.AlreadyConfigured {
		t.Fatalf("expected AlreadyConfigured after install, got %+v", got)
	}
}

func TestHermes_DescribePaths_Global(t *testing.T) {
	h := hermesTarget{}
	paths := h.DescribePaths(LocationGlobal)
	if len(paths) != 1 {
		t.Fatalf("expected exactly one config path, got %v", paths)
	}
}

func TestHermes_DescribePaths_LocalEmpty(t *testing.T) {
	h := hermesTarget{}
	paths := h.DescribePaths(LocationLocal)
	if len(paths) != 0 {
		t.Fatalf("expected no paths for unsupported local location, got %v", paths)
	}
}

func TestHermes_NoInstructionsFileWritten(t *testing.T) {
	home := fakeHome(t)
	h := hermesTarget{}
	h.Install(LocationGlobal, InstallOptions{ExecPath: "/usr/local/bin/codegraph"})

	// Hermes has no AGENTS.md-equivalent instructions surface — assert no
	// marker-fenced file was created anywhere under the fake home.
	if fileExists(filepath.Join(home, ".hermes", "AGENTS.md")) {
		t.Fatalf("hermes must NOT write an instructions file (parity regression)")
	}
}
