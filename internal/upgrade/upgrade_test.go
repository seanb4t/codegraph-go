package upgrade

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestUpgradeRun_CheckReportsAvailabilityWithoutDownloading asserts D-11's
// no-download guarantee: --check resolves and reports current-vs-latest
// but never invokes the download/verify/swap seams.
func TestUpgradeRun_CheckReportsAvailabilityWithoutDownloading(t *testing.T) {
	var downloadCalled, verifyCalled, swapCalled bool

	var out bytes.Buffer
	opts := Options{
		Check: true,
		Out:   &out,
		resolveLatest: func(repoSlug string) (string, error) {
			return "v1.2.3", nil
		},
		download: func(v string) ([]byte, []byte, error) {
			downloadCalled = true
			return nil, nil, nil
		},
		verify: func(binary, bundleJSON []byte) error {
			verifyCalled = true
			return nil
		},
		swap: func(targetPath string, newBinary []byte) error {
			swapCalled = true
			return nil
		},
	}

	if err := Run("v1.0.0", "/does/not/matter", opts); err != nil {
		t.Fatalf("Run(--check): %v", err)
	}
	if downloadCalled || verifyCalled || swapCalled {
		t.Fatalf("Run(--check) invoked download=%v verify=%v swap=%v, want all false", downloadCalled, verifyCalled, swapCalled)
	}
	if !strings.Contains(out.String(), "v1.2.3") {
		t.Errorf("Run(--check) output = %q, want it to mention v1.2.3", out.String())
	}
}

// TestUpgradeRun_TamperedDownloadNeverSwaps is the flagship Pitfall-7 assertion at
// the orchestrator level: a verification failure MUST abort before swap is
// ever invoked, and the original binary at targetPath MUST remain
// untouched.
func TestUpgradeRun_TamperedDownloadNeverSwaps(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "codegraph")
	if err := os.WriteFile(target, []byte("original-binary"), 0o755); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	swapCalled := false
	opts := Options{
		resolveLatest: func(repoSlug string) (string, error) { return "v1.2.3", nil },
		download: func(v string) ([]byte, []byte, error) {
			return []byte("tampered-bytes"), []byte("bundle"), nil
		},
		verify: func(binary, bundleJSON []byte) error {
			return errors.New("digest mismatch")
		},
		swap: func(targetPath string, newBinary []byte) error {
			swapCalled = true
			return nil
		},
	}

	if err := Run("v1.0.0", target, opts); err == nil {
		t.Fatal("Run: expected an error for a tampered download, got nil")
	}
	if swapCalled {
		t.Fatal("Run: swap was invoked despite a verification failure (Pitfall 7 violation)")
	}

	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read target: %v", readErr)
	}
	if string(got) != "original-binary" {
		t.Errorf("target changed despite verification failure: %q", got)
	}
}

// TestUpgradeRun_ValidPathVerifiesBeforeSwap asserts call ORDER: download, then
// verify, then swap — never swap before verify.
func TestUpgradeRun_ValidPathVerifiesBeforeSwap(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "codegraph")
	if err := os.WriteFile(target, []byte("original-binary"), 0o755); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	var order []string
	opts := Options{
		resolveLatest: func(repoSlug string) (string, error) { return "v1.2.3", nil },
		download: func(v string) ([]byte, []byte, error) {
			order = append(order, "download")
			return []byte("new-binary"), []byte("bundle"), nil
		},
		verify: func(binary, bundleJSON []byte) error {
			order = append(order, "verify")
			return nil
		},
		swap: func(targetPath string, newBinary []byte) error {
			order = append(order, "swap")
			return nil
		},
	}

	if err := Run("v1.0.0", target, opts); err != nil {
		t.Fatalf("Run: %v", err)
	}

	want := []string{"download", "verify", "swap"}
	if len(order) != len(want) {
		t.Fatalf("call order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("call order = %v, want %v", order, want)
		}
	}
}

// TestUpgradeRun_SameVersionNoOpWithoutForce asserts the same-version guard
// (T-08-03-01): when the resolved target equals currentVersion and Force is
// false, Run prints an advisory and returns nil WITHOUT calling
// download/verify/swap.
func TestUpgradeRun_SameVersionNoOpWithoutForce(t *testing.T) {
	var downloadCalled, verifyCalled, swapCalled bool

	var out bytes.Buffer
	opts := Options{
		Out: &out,
		resolveLatest: func(repoSlug string) (string, error) {
			return "v1.0.0", nil
		},
		download: func(v string) ([]byte, []byte, error) {
			downloadCalled = true
			return nil, nil, nil
		},
		verify: func(binary, bundleJSON []byte) error {
			verifyCalled = true
			return nil
		},
		swap: func(targetPath string, newBinary []byte) error {
			swapCalled = true
			return nil
		},
	}

	if err := Run("v1.0.0", "/does/not/matter", opts); err != nil {
		t.Fatalf("Run(same-version, no force): %v", err)
	}
	if downloadCalled || verifyCalled || swapCalled {
		t.Fatalf("Run(same-version, no force) invoked download=%v verify=%v swap=%v, want all false", downloadCalled, verifyCalled, swapCalled)
	}
	if !strings.Contains(out.String(), "v1.0.0") {
		t.Errorf("Run(same-version, no force) output = %q, want it to mention v1.0.0", out.String())
	}
}

// TestUpgradeRun_ForceReinstallsSameVersion asserts Force=true bypasses the
// same-version no-op guard and proceeds through download → verify → swap
// even when target == currentVersion.
func TestUpgradeRun_ForceReinstallsSameVersion(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "codegraph")
	if err := os.WriteFile(target, []byte("original-binary"), 0o755); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	var order []string
	opts := Options{
		Force: true,
		resolveLatest: func(repoSlug string) (string, error) {
			return "v1.0.0", nil
		},
		download: func(v string) ([]byte, []byte, error) {
			order = append(order, "download")
			return []byte("new-binary"), []byte("bundle"), nil
		},
		verify: func(binary, bundleJSON []byte) error {
			order = append(order, "verify")
			return nil
		},
		swap: func(targetPath string, newBinary []byte) error {
			order = append(order, "swap")
			return nil
		},
	}

	if err := Run("v1.0.0", target, opts); err != nil {
		t.Fatalf("Run(same-version, force): %v", err)
	}

	want := []string{"download", "verify", "swap"}
	if len(order) != len(want) {
		t.Fatalf("call order = %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("call order = %v, want %v", order, want)
		}
	}
}

// TestUpgradeRun_ForceStillVerifiesBeforeSwap asserts T-08-03-01's flagship
// claim: Force=true does NOT weaken verification — a failing fake verify
// still aborts before swap is ever invoked, even on a forced same-version
// reinstall.
func TestUpgradeRun_ForceStillVerifiesBeforeSwap(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "codegraph")
	if err := os.WriteFile(target, []byte("original-binary"), 0o755); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	swapCalled := false
	opts := Options{
		Force: true,
		resolveLatest: func(repoSlug string) (string, error) {
			return "v1.0.0", nil
		},
		download: func(v string) ([]byte, []byte, error) {
			return []byte("tampered-bytes"), []byte("bundle"), nil
		},
		verify: func(binary, bundleJSON []byte) error {
			return errors.New("digest mismatch")
		},
		swap: func(targetPath string, newBinary []byte) error {
			swapCalled = true
			return nil
		},
	}

	if err := Run("v1.0.0", target, opts); err == nil {
		t.Fatal("Run(force, failing verify): expected an error, got nil")
	}
	if swapCalled {
		t.Fatal("Run(force, failing verify): swap was invoked despite a verification failure (T-08-03-01 violation)")
	}

	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read target: %v", readErr)
	}
	if string(got) != "original-binary" {
		t.Errorf("target changed despite verification failure: %q", got)
	}
}

// TestUpgradeRun_RefusesNonWritableTargetBeforeDownloading asserts D-13: a
// non-writable target aborts before the download seam is ever invoked (no
// wasted download for an upgrade that can't be installed anyway).
func TestUpgradeRun_RefusesNonWritableTargetBeforeDownloading(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root bypasses permission bits")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "codegraph")
	if err := os.WriteFile(target, []byte("original-binary"), 0o755); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("chmod dir read-only: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	downloadCalled := false
	opts := Options{
		resolveLatest: func(repoSlug string) (string, error) { return "v1.2.3", nil },
		download: func(v string) ([]byte, []byte, error) {
			downloadCalled = true
			return []byte("new-binary"), []byte("bundle"), nil
		},
		verify: func(binary, bundleJSON []byte) error { return nil },
		swap:   func(targetPath string, newBinary []byte) error { return nil },
	}

	if err := Run("v1.0.0", target, opts); err == nil {
		t.Fatal("Run: expected an error for a non-writable target, got nil")
	}
	if downloadCalled {
		t.Fatal("Run: download was invoked despite a non-writable target (D-13 violation)")
	}
}

// TestUpgradeRun_RefusesBrewManagedCask asserts D-05/D-08: Run() invoked
// through a symlink into a receipt-bearing Caskroom tree refuses with the
// resolved install directory and `brew upgrade codegraph`, having called
// none of the four orchestration seams — the refusal is reached before any
// of them fire (D-11).
func TestUpgradeRun_RefusesBrewManagedCask(t *testing.T) {
	root := t.TempDir()

	versionDir := filepath.Join(root, "Caskroom", "codegraph", "0.8.0")
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatalf("mkdir version dir: %v", err)
	}
	payload := filepath.Join(versionDir, "codegraph")
	if err := os.WriteFile(payload, []byte("real-binary-bytes"), 0o755); err != nil {
		t.Fatalf("seed payload: %v", err)
	}

	metadataDir := filepath.Join(root, "Caskroom", "codegraph", ".metadata")
	if err := os.MkdirAll(metadataDir, 0o755); err != nil {
		t.Fatalf("mkdir metadata dir: %v", err)
	}
	receipt := filepath.Join(metadataDir, "INSTALL_RECEIPT.json")
	if err := os.WriteFile(receipt, []byte(`{"used_options":[]}`), 0o644); err != nil {
		t.Fatalf("seed receipt: %v", err)
	}

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}
	symlink := filepath.Join(binDir, "codegraph")
	if err := os.Symlink(payload, symlink); err != nil {
		t.Fatalf("symlink payload: %v", err)
	}

	// t.TempDir() on macOS is itself a symlink into /private, so compute
	// the expected directory via EvalSymlinks rather than raw string
	// concatenation — a raw comparison would be flaky (RESEARCH Pitfall 1).
	resolvedPayload, err := filepath.EvalSymlinks(payload)
	if err != nil {
		t.Fatalf("EvalSymlinks(payload): %v", err)
	}
	expectedInstallDir := filepath.Dir(resolvedPayload)
	if !filepath.IsAbs(expectedInstallDir) {
		t.Fatalf("expected install dir %q is not absolute", expectedInstallDir)
	}
	if !strings.HasSuffix(expectedInstallDir, filepath.Join("Caskroom", "codegraph", "0.8.0")) {
		t.Fatalf("expected install dir %q does not end with Caskroom/codegraph/0.8.0", expectedInstallDir)
	}

	var resolveLatestCalled, downloadCalled, verifyCalled, swapCalled bool
	opts := Options{
		resolveLatest: func(repoSlug string) (string, error) {
			resolveLatestCalled = true
			return "v1.2.3", nil
		},
		download: func(v string) ([]byte, []byte, error) {
			downloadCalled = true
			return nil, nil, nil
		},
		verify: func(binary, bundleJSON []byte) error {
			verifyCalled = true
			return nil
		},
		swap: func(targetPath string, newBinary []byte) error {
			swapCalled = true
			return nil
		},
	}

	runErr := Run("v1.0.0", symlink, opts)
	if runErr == nil {
		t.Fatal("Run: expected a non-nil error for a Homebrew-managed cask, got nil")
	}
	if resolveLatestCalled || downloadCalled || verifyCalled || swapCalled {
		t.Fatalf("Run: invoked resolveLatest=%v download=%v verify=%v swap=%v, want all false", resolveLatestCalled, downloadCalled, verifyCalled, swapCalled)
	}
	if !strings.Contains(runErr.Error(), "brew upgrade codegraph") {
		t.Errorf("Run error = %q, want it to contain %q", runErr.Error(), "brew upgrade codegraph")
	}
	if !strings.Contains(runErr.Error(), expectedInstallDir) {
		t.Errorf("Run error = %q, want it to contain the resolved install dir %q", runErr.Error(), expectedInstallDir)
	}

	wantPattern := `^upgrade: codegraph is managed by Homebrew \(.*/Caskroom/codegraph/0\.8\.0\)\. Upgrade with: brew upgrade codegraph$`
	matched, matchErr := regexp.MatchString(wantPattern, runErr.Error())
	if matchErr != nil {
		t.Fatalf("regexp.MatchString: %v", matchErr)
	}
	if !matched {
		t.Errorf("Run error = %q, does not match expected shape %q", runErr.Error(), wantPattern)
	}
}
