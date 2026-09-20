package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/agents"
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
		"brew upgrade codegraph",   // the pointer command, verbatim (D-07)
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

// TestRefreshInstalledSkills_CarriesPreToolNudge: upgrade's refresh passes
// PreToolNudgeKeep (D-10), so an opted-in location's guard is re-rendered
// for the new binary and a location that never opted in gains nothing.
func TestRefreshInstalledSkills_CarriesPreToolNudge(t *testing.T) {
	t.Run("opted_in_refreshed", func(t *testing.T) {
		home := fakeHome(t)
		if _, _, err := execCmd("install", "--target", "claude", "--location", "global", "--pretool-nudge"); err != nil {
			t.Fatalf("install --pretool-nudge: %v", err)
		}

		if err := refreshInstalledSkills("/opt/new/codegraph", io.Discard); err != nil {
			t.Fatalf("refreshInstalledSkills: %v", err)
		}

		guard := readFileString(t, filepath.Join(home, ".claude", "hooks", "pretooluse-nudge.sh"))
		if !strings.Contains(guard, "codegraph_bin='/opt/new/codegraph'") {
			t.Fatalf("the refreshed guard does not point at the new binary:\n%s", guard)
		}
		hooks, _ := readJSONMap(t, filepath.Join(home, ".claude", "settings.json"))["hooks"].(map[string]any)
		if _, ok := hooks["PreToolUse"]; !ok {
			t.Fatalf("refresh dropped hooks.PreToolUse at an opted-in location")
		}
	})

	t.Run("never_opted_not_added", func(t *testing.T) {
		home := fakeHome(t)
		if _, _, err := execCmd("install", "--target", "claude", "--location", "global"); err != nil {
			t.Fatalf("plain install: %v", err)
		}

		if err := refreshInstalledSkills("/opt/new/codegraph", io.Discard); err != nil {
			t.Fatalf("refreshInstalledSkills: %v", err)
		}

		if _, err := os.Stat(filepath.Join(home, ".claude", "hooks", "pretooluse-nudge.sh")); !os.IsNotExist(err) {
			t.Fatalf("refresh added the guard at a location that never opted in (stat err %v)", err)
		}
		hooks, _ := readJSONMap(t, filepath.Join(home, ".claude", "settings.json"))["hooks"].(map[string]any)
		if _, ok := hooks["PreToolUse"]; ok {
			t.Fatalf("refresh added hooks.PreToolUse at a location that never opted in")
		}
		if _, err := os.Stat(filepath.Join(home, ".claude", "settings.json")); err != nil {
			t.Fatalf("positive control: refresh did not touch the configured location: %v", err)
		}
	})
}

// TestRefreshInstalledSkills_ForeignSkillDirLocationIsAcceptedLimitation
// pins CR-01's residual gap (06-REVIEW.md), deliberately left unresolved by
// the CR-01 fix in internal/agents (preToolNudgeEvidenced): when Claude's
// skill directory is a symlinked shared directory holding pre-existing
// foreign, unmanifested content (D-14), Claude's manifest step never runs
// there under ANY PreToolNudge mode — there is no manifest codegraph is
// permitted to write into that directory. agents.ConfiguredSkillLocations
// discovers a location purely through manifest presence — its own
// documented, separately-pinned contract (internal/agents'
// TestConfiguredSkillLocations_* suite) — so upgrade's refresh loop, which
// iterates only that list, can never reach this location, even though the
// guard and its settings.json registration are present and evidenced.
//
// This is intentionally NOT fixed by widening ConfiguredSkillLocations
// itself: that function has several other pinned callers/tests asserting
// manifest-only discovery, and widening its general contract to
// Claude-and-PreToolUse-specific settings.json evidence was judged a
// disproportionate blast radius for a gap whose existing, documented
// recovery already works — a direct `codegraph install` run FROM the
// affected location (proven below via the exported AgentTarget.Install
// path exactly as refreshInstalledSkills itself calls it) still refreshes
// the guard correctly, and `codegraph upgrade`'s own warning path already
// names that exact recovery command when a refresh problem is suspected.
func TestRefreshInstalledSkills_ForeignSkillDirLocationIsAcceptedLimitation(t *testing.T) {
	home := fakeHome(t)
	if err := os.MkdirAll(filepath.Join(home, ".agents", "skills", "codegraph"), 0o755); err != nil {
		t.Fatalf("mkdir shared skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".agents", "skills", "codegraph", "SKILL.md"), []byte("# foreign, never written by codegraph\n"), 0o644); err != nil {
		t.Fatalf("seed foreign content: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".claude", "skills"), 0o755); err != nil {
		t.Fatalf("mkdir .claude/skills: %v", err)
	}
	if err := os.Symlink(filepath.Join("..", "..", ".agents", "skills", "codegraph"), filepath.Join(home, ".claude", "skills", "codegraph")); err != nil {
		t.Fatalf("symlink claude skill dir: %v", err)
	}

	if _, _, err := execCmd("install", "--target", "claude", "--location", "global", "--pretool-nudge"); err != nil {
		t.Fatalf("install --pretool-nudge: %v", err)
	}
	guardPath := filepath.Join(home, ".claude", "hooks", "pretooluse-nudge.sh")
	if _, err := os.Stat(guardPath); err != nil {
		t.Fatalf("precondition: guard was not written despite the foreign skill dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "codegraph", ".codegraph-manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("precondition broken: a manifest exists despite foreign content — this test no longer isolates the gap")
	}

	// The gap: ConfiguredSkillLocations-driven refresh cannot discover this
	// location, so it leaves the guard untouched for a new binary path.
	if err := refreshInstalledSkills("/opt/new/codegraph", io.Discard); err != nil {
		t.Fatalf("refreshInstalledSkills: %v", err)
	}
	afterBlindRefresh := readFileString(t, guardPath)
	if strings.Contains(afterBlindRefresh, "opt/new/codegraph") {
		t.Fatal("refreshInstalledSkills unexpectedly reached the foreign-skill-dir location; the accepted-limitation premise this test pins no longer holds — update this test and 06-REVIEW-FIX.md")
	}

	// The documented recovery: calling Install directly for the location
	// (exactly what a plain `codegraph install` run from there does) DOES
	// refresh it, because preToolNudgeEvidenced widens "recorded" to
	// settings.json's own registration (CR-01's actual fix).
	targets, err := agents.ResolveTargetFlag(string(agents.Claude), agents.LocationGlobal)
	if err != nil {
		t.Fatalf("ResolveTargetFlag: %v", err)
	}
	for _, tg := range targets {
		result := tg.Install(agents.LocationGlobal, agents.InstallOptions{
			ExecPath:     "/opt/new/codegraph",
			PreToolNudge: agents.PreToolNudgeKeep,
		})
		if len(result.Errors) != 0 {
			t.Fatalf("recovery Install returned errors: %v", result.Errors)
		}
	}
	afterRecovery := readFileString(t, guardPath)
	if !strings.Contains(afterRecovery, "opt/new/codegraph") {
		t.Fatalf("a direct Keep install did not refresh the guard for the new binary despite CR-01's fix; guard:\n%s", afterRecovery)
	}
}
