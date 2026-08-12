package cli

import (
	"errors"
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
