package upgrade

import (
	"crypto/sha256"
	"fmt"
	"os"
	"runtime"
	"testing"
)

// goReleaserAssetName reproduces .goreleaser.yaml's archives.name_template
// literally:
//
//	{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}
//
// GoReleaser's default .Os/.Arch template values are the same strings as Go's
// own GOOS/GOARCH (this project sets no goarch/goos `replacements:` block in
// .goreleaser.yaml), so goos/goarch here are passed through unchanged. The
// template carries no `.exe` conditional because native Windows is not a
// supported platform (quick task 260807-gho).
func goReleaserAssetName(tag, goos, goarch string) string {
	return fmt.Sprintf("codegraph_%s_%s_%s", tag, goos, goarch)
}

// TestReleaseAssetNameMatchesGoReleaser pins D-14: the asset name
// internal/upgrade downloads (releaseAssetName) MUST equal, byte for byte,
// the asset name .goreleaser.yaml's archives.name_template actually produces
// for every one of the 4 shipped release targets. A divergence here means
// `codegraph upgrade` 404s on a genuine, correctly-published release.
func TestReleaseAssetNameMatchesGoReleaser(t *testing.T) {
	const tag = "v1.2.3"

	// One literal expectation per shipped (goos,goarch) pair (RESEARCH.md
	// Finding 2 / .goreleaser.yaml's builds: list) — pinned independently of
	// both goReleaserAssetName and releaseAssetName so a bug shared by both
	// implementations can't hide from this test. Windows pairs were removed
	// with native Windows support (quick task 260807-gho); a WSL2 user's
	// host reports linux, so the linux pairs already cover them.
	pairs := []struct {
		goos, goarch string
		want         string
	}{
		{"linux", "amd64", "codegraph_v1.2.3_linux_amd64"},
		{"linux", "arm64", "codegraph_v1.2.3_linux_arm64"},
		{"darwin", "amd64", "codegraph_v1.2.3_darwin_amd64"},
		{"darwin", "arm64", "codegraph_v1.2.3_darwin_arm64"},
	}

	if len(pairs) != 4 {
		t.Fatalf("expected exactly 4 os/arch pairs (the shipped release matrix), got %d", len(pairs))
	}

	hostMatched := false
	for _, p := range pairs {
		t.Run(p.goos+"_"+p.goarch, func(t *testing.T) {
			got := goReleaserAssetName(tag, p.goos, p.goarch)
			if got != p.want {
				t.Fatalf("goReleaserAssetName(%q, %q, %q) = %q, want %q", tag, p.goos, p.goarch, got, p.want)
			}
		})

		// releaseAssetName reads runtime.GOOS/runtime.GOARCH directly, so it
		// can only be exercised end-to-end for the CURRENT host's pair. When
		// this pair matches the host, assert releaseAssetName(tag) agrees
		// with the SAME pinned literal the template check above just used —
		// this is the actual D-14 cross-check, not just a self-consistent
		// template re-derivation.
		if p.goos == runtime.GOOS && p.goarch == runtime.GOARCH {
			hostMatched = true
			if got := releaseAssetName(tag); got != p.want {
				t.Fatalf("releaseAssetName(%q) = %q, want %q (must agree with GoReleaser's name_template for this host's os/arch)", tag, got, p.want)
			}
		}
	}

	if !hostMatched {
		t.Fatalf("host os/arch (%s/%s) is not one of the 4 pinned release pairs — releaseAssetName was never cross-checked against a real template literal", runtime.GOOS, runtime.GOARCH)
	}
}

// e2eFixtureBinaryPath/e2eFixtureBundlePath are the COMMITTED fixture pair's
// location (primary source, per the plan): a real codegraph_<tag>_<os>_<arch>
// binary and its cosign v3 .sigstore.json bundle, captured from an actual
// seanb4t/codegraph-go release. Neither exists yet — DIST-02 has not
// produced a real signed release as of this plan — so e2eArtifactPaths falls
// through to the env-var live-artifact variant, and finally to a clean skip.
const (
	e2eFixtureBinaryPath = "testdata/e2e-release-binary"
	e2eFixtureBundlePath = "testdata/e2e-release-binary.sigstore.json"
)

// e2eArtifactPaths resolves the real signed artifact TestVerifyReleaseE2E
// needs, in priority order: (1) the committed testdata fixture pair, (2) the
// CODEGRAPH_E2E_BINARY/CODEGRAPH_E2E_BUNDLE env vars for a live-artifact run
// (e.g. against a real tag's release from CI). ok is false when neither
// source is present — the caller must t.Skip, never fail, in that case.
func e2eArtifactPaths() (binaryPath, bundlePath string, ok bool) {
	if fileExists(e2eFixtureBinaryPath) && fileExists(e2eFixtureBundlePath) {
		return e2eFixtureBinaryPath, e2eFixtureBundlePath, true
	}
	if envBinary, envBundle := os.Getenv("CODEGRAPH_E2E_BINARY"), os.Getenv("CODEGRAPH_E2E_BUNDLE"); envBinary != "" && envBundle != "" {
		return envBinary, envBundle, true
	}
	return "", "", false
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// TestVerifyReleaseE2E is Finding 1's loop-closer (RESEARCH.md Finding 1
// action item, CONTEXT.md <specifics>): a real signed release artifact,
// hashed exactly the way defaultVerify hashes it (sha256 over the downloaded
// BINARY, never a checksums file), must pass verifyRelease under the
// PRODUCTION identity constants — releaseOIDCIssuer and
// releaseWorkflowRefPattern from verify.go — not the fixture SAN used by the
// offline TestVerifyRelease_* tests in verify_test.go.
//
// No real signed artifact exists yet (DIST-02's first real tag-triggered
// release hasn't shipped — see 08-VALIDATION.md's "First real signed-release
// verify" manual-only row), so this test currently skips cleanly. Once a
// fixture pair is committed under testdata/ (or CODEGRAPH_E2E_BINARY/
// CODEGRAPH_E2E_BUNDLE point at a real release download), it exercises the
// full chain against a live-fetched Sigstore trusted root — this is
// intentionally the one path in this package that touches the network,
// scoped to only run when a real artifact is actually supplied.
func TestVerifyReleaseE2E(t *testing.T) {
	binaryPath, bundlePath, ok := e2eArtifactPaths()
	if !ok {
		t.Skip("no real signed release artifact available: commit a fixture pair at " +
			e2eFixtureBinaryPath + " / " + e2eFixtureBundlePath + ", or set " +
			"CODEGRAPH_E2E_BINARY/CODEGRAPH_E2E_BUNDLE to a real release download — " +
			"this test only runs against a genuinely signed artifact, never fabricated data " +
			"(see 08-VALIDATION.md's \"First real signed-release verify\" manual-only row)")
	}

	binary, err := os.ReadFile(binaryPath)
	if err != nil {
		t.Fatalf("read e2e binary %s: %v", binaryPath, err)
	}
	bundleJSON, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read e2e bundle %s: %v", bundlePath, err)
	}

	b, err := loadBundle(bundleJSON)
	if err != nil {
		t.Fatalf("parse e2e sigstore bundle: %v", err)
	}

	// The captured bundle was signed against the REAL, live Sigstore
	// public-good instance (not the offline embedded fixture trust root used
	// by verify_test.go) — fetchTrustedRoot is verify.go's one production
	// network call, exercised here deliberately.
	tr, err := fetchTrustedRoot()
	if err != nil {
		t.Fatalf("fetch live sigstore trusted root: %v", err)
	}

	// Finding 1's exact call path: sha256 of the raw binary bytes, never a
	// checksums.txt digest.
	digest := sha256.Sum256(binary)

	t.Run("accepts production identity", func(t *testing.T) {
		// releaseOIDCIssuer is baked into verifyRelease's own
		// verify.NewShortCertificateIdentity call — asserted indirectly here
		// by requiring the overall policy (issuer + SAN + digest) to accept
		// a bundle that was genuinely produced by GitHub Actions' OIDC
		// issuer running the real release.yml workflow.
		t.Logf("verifying against production issuer=%s SAN pattern=%s", releaseOIDCIssuer, releaseWorkflowRefPattern)
		if err := verifyRelease(b, tr, "sha256", digest[:], releaseWorkflowRefPattern); err != nil {
			t.Fatalf("verifyRelease: expected nil for a genuine signed artifact under the production identity, got: %v", err)
		}
	})

	t.Run("rejects wrong identity", func(t *testing.T) {
		const wrongSAN = "^https://github.com/some-other-org/some-other-repo/"
		if err := verifyRelease(b, tr, "sha256", digest[:], wrongSAN); err == nil {
			t.Fatal("verifyRelease: expected a non-nil error for a wrong certificate identity, got nil")
		}
	})
}
