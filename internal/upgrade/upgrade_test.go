package upgrade

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
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
