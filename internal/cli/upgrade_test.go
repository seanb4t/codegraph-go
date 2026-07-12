package cli

import (
	"errors"
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
