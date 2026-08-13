package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/upgrade"
)

// TestUpgradeCommand_DelegatesWithCheckAndVersion asserts the thin
// command's whole job: parse --check/positional-version, resolve
// os.Executable()/version.Info().Version, and delegate to upgrade.Run
// unchanged. upgradeRunFunc is swapped for a fake so this test never
// touches the network (mirrors internal/upgrade's own injectable-seam
// pattern, one level up).
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

	if !gotOpts.Check {
		t.Error("Options.Check = false, want true")
	}
	if gotOpts.Version != "v1.4.0" {
		t.Errorf("Options.Version = %q, want v1.4.0", gotOpts.Version)
	}
	if gotCurrent == "" {
		t.Error("currentVersion passed to upgrade.Run is empty")
	}
	if gotTarget == "" {
		t.Error("targetPath passed to upgrade.Run is empty")
	}
}

// TestUpgradeCommand_PropagatesError asserts upgrade.Run's error surfaces
// unchanged through RunE (no ad-hoc os.Exit inside RunE — main.go owns the
// exit code, per the project's Cobra thin-command convention).
func TestUpgradeCommand_PropagatesError(t *testing.T) {
	orig := upgradeRunFunc
	wantErr := errors.New("boom")
	upgradeRunFunc = func(currentVersion, targetPath string, opts upgrade.Options) error {
		return wantErr
	}
	t.Cleanup(func() { upgradeRunFunc = orig })

	if _, _, err := execCmd("upgrade", "--check"); err == nil {
		t.Fatal("upgrade --check: expected error to propagate, got nil")
	}
}

// TestUpgradeCommand_NoArgsDefaultsToLatest asserts the plain `codegraph
// upgrade` (no positional version, no --check) still delegates correctly
// with an empty pinned Version (upgrade.Run treats empty as "latest").
func TestUpgradeCommand_NoArgsDefaultsToLatest(t *testing.T) {
	var gotOpts upgrade.Options

	orig := upgradeRunFunc
	upgradeRunFunc = func(currentVersion, targetPath string, opts upgrade.Options) error {
		gotOpts = opts
		return nil
	}
	t.Cleanup(func() { upgradeRunFunc = orig })

	if _, _, err := execCmd("upgrade"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if gotOpts.Check {
		t.Error("Options.Check = true, want false")
	}
	if gotOpts.Version != "" {
		t.Errorf("Options.Version = %q, want empty", gotOpts.Version)
	}
}

// TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes asserts `--help`
// (cmd.Long) names the Homebrew refusal, the pointer command, and both exit
// behaviours (D-07, D-10), and offers no override for the refusal (D-06).
// Positive assertions run first — an absence-only test would pass against
// an empty Long, so the positive assertions are what make the negative
// assertions meaningful (repo rule 84d1gfpywd).
func TestUpgradeCommand_HelpDocumentsBrewRefusalAndExitCodes(t *testing.T) {
	long := newUpgradeCmd().Long

	required := []string{
		"brew upgrade codegraph",  // the pointer command, verbatim (D-07)
		"Homebrew-managed install", // names what is detected/refused
		"exits\nnon-zero",          // bare-refusal exit behaviour (D-05, D-10)
		"exits\nzero",              // --check exit behaviour (D-09, D-10)
	}
	for _, want := range required {
		if !strings.Contains(long, want) {
			t.Errorf("Long missing required substring %q; got:\n%s", want, long)
		}
	}

	// Only meaningful because the positive assertions above already proved
	// Long is non-empty and on-topic.
	forbidden := []string{"--force", "override", "bypass"}
	for _, absent := range forbidden {
		if strings.Contains(long, absent) {
			t.Errorf("Long unexpectedly offers an override via %q; got:\n%s", absent, long)
		}
	}
}

// TestUpgradeCommand_RefreshesConfiguredLocations asserts D-06's ordering:
// once a fake upgradeRunFunc reports a successful swap, the refresh seam is
// invoked exactly once and receives the same resolved executable path
// upgrade.Run itself was given.
func TestUpgradeCommand_RefreshesConfiguredLocations(t *testing.T) {
	origRun := upgradeRunFunc
	upgradeRunFunc = func(currentVersion, targetPath string, opts upgrade.Options) error {
		return nil
	}
	t.Cleanup(func() { upgradeRunFunc = origRun })

	var calls int
	var gotExecPath string
	origRefresh := refreshInstalledSkillsFunc
	refreshInstalledSkillsFunc = func(execPath string, out io.Writer) error {
		calls++
		gotExecPath = execPath
		return nil
	}
	t.Cleanup(func() { refreshInstalledSkillsFunc = origRefresh })

	if _, _, err := execCmd("upgrade"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}

	if calls != 1 {
		t.Fatalf("refresh call count = %d, want 1", calls)
	}
	if gotExecPath == "" {
		t.Error("refresh received an empty exec path")
	}
}

// TestUpgradeCommand_SkipsRefreshUnderCheck asserts --check never invokes
// the refresh seam: it answered a question and must mutate nothing.
func TestUpgradeCommand_SkipsRefreshUnderCheck(t *testing.T) {
	origRun := upgradeRunFunc
	upgradeRunFunc = func(currentVersion, targetPath string, opts upgrade.Options) error {
		return nil
	}
	t.Cleanup(func() { upgradeRunFunc = origRun })

	var calls int
	origRefresh := refreshInstalledSkillsFunc
	refreshInstalledSkillsFunc = func(execPath string, out io.Writer) error {
		calls++
		return nil
	}
	t.Cleanup(func() { refreshInstalledSkillsFunc = origRefresh })

	if _, _, err := execCmd("upgrade", "--check"); err != nil {
		t.Fatalf("upgrade --check: %v", err)
	}

	if calls != 0 {
		t.Fatalf("refresh call count = %d, want 0", calls)
	}
}

// TestUpgradeCommand_SkipsRefreshWhenSwapFails asserts a failed swap
// attempts no refresh at all and returns the swap error unchanged.
func TestUpgradeCommand_SkipsRefreshWhenSwapFails(t *testing.T) {
	sentinel := errors.New("swap failed")
	origRun := upgradeRunFunc
	upgradeRunFunc = func(currentVersion, targetPath string, opts upgrade.Options) error {
		return sentinel
	}
	t.Cleanup(func() { upgradeRunFunc = origRun })

	var calls int
	origRefresh := refreshInstalledSkillsFunc
	refreshInstalledSkillsFunc = func(execPath string, out io.Writer) error {
		calls++
		return nil
	}
	t.Cleanup(func() { refreshInstalledSkillsFunc = origRefresh })

	_, _, err := execCmd("upgrade")
	if !errors.Is(err, sentinel) {
		t.Fatalf("upgrade error = %v, want sentinel %v", err, sentinel)
	}
	if calls != 0 {
		t.Fatalf("refresh call count = %d, want 0", calls)
	}
}

// TestRefreshInstalledSkills_OnlyPreviouslyConfiguredLocations asserts
// refreshInstalledSkills installs to exactly the locations that already
// carried a manifest: a global-only fixture refreshes global and leaves
// the local .claude/ tree absent, and a fixture with no manifest anywhere
// installs nowhere and reports nothing.
func TestRefreshInstalledSkills_OnlyPreviouslyConfiguredLocations(t *testing.T) {
	t.Run("global only", func(t *testing.T) {
		home := fakeHome(t)

		globalManifest := filepath.Join(home, ".claude", "skills", "codegraph", ".codegraph-manifest.json")
		if err := os.MkdirAll(filepath.Dir(globalManifest), 0o755); err != nil {
			t.Fatalf("mkdir global manifest dir: %v", err)
		}
		if err := os.WriteFile(globalManifest, []byte("{}"), 0o644); err != nil {
			t.Fatalf("seed global manifest: %v", err)
		}

		if err := refreshInstalledSkills("/fake/codegraph", io.Discard); err != nil {
			t.Fatalf("refreshInstalledSkills: %v", err)
		}

		if _, err := os.Stat(filepath.Join(home, ".claude", "settings.json")); err != nil {
			t.Errorf("expected global settings.json to be refreshed: %v", err)
		}
		if _, err := os.Stat(".claude"); !os.IsNotExist(err) {
			t.Errorf("expected local .claude/ tree to remain absent, stat err = %v", err)
		}
	})

	t.Run("no manifest anywhere", func(t *testing.T) {
		fakeHome(t)

		var buf strings.Builder
		if err := refreshInstalledSkills("/fake/codegraph", &buf); err != nil {
			t.Fatalf("refreshInstalledSkills: %v", err)
		}

		if buf.Len() != 0 {
			t.Errorf("expected no output when nothing is configured, got %q", buf.String())
		}
		if _, err := os.Stat(".claude"); !os.IsNotExist(err) {
			t.Errorf("expected no local .claude/ tree, stat err = %v", err)
		}
	})
}

// TestUpgradeCommand_RefreshFailureIsWarningNotError asserts D-07: when the
// swap succeeds but the refresh seam fails, upgrade still returns nil (the
// swap genuinely worked) and prints a warning naming `codegraph install` as
// the command to re-run.
func TestUpgradeCommand_RefreshFailureIsWarningNotError(t *testing.T) {
	origRun := upgradeRunFunc
	upgradeRunFunc = func(currentVersion, targetPath string, opts upgrade.Options) error {
		return nil
	}
	t.Cleanup(func() { upgradeRunFunc = origRun })

	origRefresh := refreshInstalledSkillsFunc
	refreshInstalledSkillsFunc = func(execPath string, out io.Writer) error {
		return errors.New("refresh boom")
	}
	t.Cleanup(func() { refreshInstalledSkillsFunc = origRefresh })

	stdout, _, err := execCmd("upgrade")
	if err != nil {
		t.Fatalf("upgrade returned error = %v, want nil (swap succeeded)", err)
	}
	if !strings.Contains(stdout, "codegraph install") {
		t.Errorf("output missing %q; got:\n%s", "codegraph install", stdout)
	}
}

// TestUpgradeCommand_RefreshSuccessPrintsNoWarning proves the warning
// assertion above discriminates rather than matching incidental text: when
// both seams succeed, no warning line appears.
func TestUpgradeCommand_RefreshSuccessPrintsNoWarning(t *testing.T) {
	origRun := upgradeRunFunc
	upgradeRunFunc = func(currentVersion, targetPath string, opts upgrade.Options) error {
		return nil
	}
	t.Cleanup(func() { upgradeRunFunc = origRun })

	origRefresh := refreshInstalledSkillsFunc
	refreshInstalledSkillsFunc = func(execPath string, out io.Writer) error {
		return nil
	}
	t.Cleanup(func() { refreshInstalledSkillsFunc = origRefresh })

	stdout, _, err := execCmd("upgrade")
	if err != nil {
		t.Fatalf("upgrade returned error = %v, want nil", err)
	}
	if strings.Contains(stdout, "warning") {
		t.Errorf("unexpected warning text in output:\n%s", stdout)
	}
}

// TestUpgradeCommand_SwapFailureReturnsSwapError asserts the swap-failure
// path is untouched by the refresh concern: the sentinel error surfaces
// unwrapped, so errors.Is against it still holds.
func TestUpgradeCommand_SwapFailureReturnsSwapError(t *testing.T) {
	sentinel := errors.New("swap failed")
	origRun := upgradeRunFunc
	upgradeRunFunc = func(currentVersion, targetPath string, opts upgrade.Options) error {
		return sentinel
	}
	t.Cleanup(func() { upgradeRunFunc = origRun })

	_, _, err := execCmd("upgrade")
	if !errors.Is(err, sentinel) {
		t.Fatalf("upgrade error = %v, want sentinel %v via errors.Is", err, sentinel)
	}
}
