package upgrade

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
)

// Fixture identity/digest constants for internal/upgrade/testdata's
// bundle+trust-root pair. The pair is copied verbatim from sigstore-go's
// own embedded hermetic test data (pkg/testing/data — real signed sigstore-
// js release provenance + the public-good trust root), per Open Question
// 1's recommendation (a): this proves the wiring/policy-construction/
// error-handling code path end to end against a real, valid bundle, fully
// offline. The identity below is sigstore-js's own GitHub Actions release
// workflow — NOT this project's identity (that's a Phase-8 finalize step,
// D-14) — the fixture only needs to prove verifyRelease's plumbing is
// correct.
const (
	fixtureSanRegex        = "^https://github.com/sigstore/sigstore-js/"
	fixtureArtifactSHA512  = "46d4e2f74c4877316640000a6fdf8a8b59f1e0847667973e9859f774dd31b8f1e0937813b777fb66a2ac67d50540fe34640966eee9fc2ccca387082b4c85cd3c"
	fixtureDigestAlgorithm = "sha512"
)

func loadFixtureBundle(t *testing.T) *bundle.Bundle {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "valid-bundle.json"))
	if err != nil {
		t.Fatalf("read fixture bundle: %v", err)
	}
	b, err := loadBundle(data)
	if err != nil {
		t.Fatalf("parse fixture bundle: %v", err)
	}
	return b
}

func loadFixtureTrustedRoot(t *testing.T) root.TrustedMaterial {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "trusted-root.json"))
	if err != nil {
		t.Fatalf("read fixture trusted root: %v", err)
	}
	tr, err := root.NewTrustedRootFromJSON(data)
	if err != nil {
		t.Fatalf("parse fixture trusted root: %v", err)
	}
	return tr
}

func mustHexDecode(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	return b
}

// TestVerifyRelease_AcceptsValidBundle proves the accept-path wiring end to
// end against a real Sigstore-signed bundle: correct digest + correct
// identity + the embedded (offline) public-good trust root all verify
// cleanly. Fully offline — no live TUF/Rekor/Fulcio network call, per Open
// Question 1.
func TestVerifyRelease_AcceptsValidBundle(t *testing.T) {
	b := loadFixtureBundle(t)
	tr := loadFixtureTrustedRoot(t)
	digest := mustHexDecode(t, fixtureArtifactSHA512)

	if err := verifyRelease(b, tr, fixtureDigestAlgorithm, digest, fixtureSanRegex); err != nil {
		t.Fatalf("verifyRelease: expected nil error for a valid bundle, got: %v", err)
	}
}

// TestVerifyRelease_RejectsTamperedArtifact is the security-critical
// reject-path assertion (Pitfall 7, T-06-06-01): a downloaded artifact
// whose digest does NOT match what the bundle actually signed for — the
// realistic MITM/tamper scenario, since an attacker who swaps the binary
// bytes cannot also forge a matching Fulcio-signed digest — MUST make
// verifyRelease return a non-nil error. Runs fully offline; must always be
// runnable (never network/short-mode gated).
func TestVerifyRelease_RejectsTamperedArtifact(t *testing.T) {
	b := loadFixtureBundle(t)
	tr := loadFixtureTrustedRoot(t)
	tamperedDigest := mustHexDecode(t, strings.Repeat("00", 64))

	err := verifyRelease(b, tr, fixtureDigestAlgorithm, tamperedDigest, fixtureSanRegex)
	if err == nil {
		t.Fatal("verifyRelease: expected a non-nil error for a tampered/digest-mismatched artifact, got nil")
	}
}

// TestVerifyRelease_RejectsWrongIdentity proves the certificate-identity
// half of the policy is enforced independently of the artifact digest: a
// correct digest with the wrong signer identity MUST also be rejected —
// the "wrong identity" scenario from the plan's behavior spec.
func TestVerifyRelease_RejectsWrongIdentity(t *testing.T) {
	b := loadFixtureBundle(t)
	tr := loadFixtureTrustedRoot(t)
	digest := mustHexDecode(t, fixtureArtifactSHA512)

	err := verifyRelease(b, tr, fixtureDigestAlgorithm, digest, "^https://github.com/some-other-org/some-other-repo/")
	if err == nil {
		t.Fatal("verifyRelease: expected a non-nil error for a wrong certificate identity, got nil")
	}
}

// TestReleaseWorkflowRefPattern_RejectsNonReleaseWorkflowInSameRepo is the
// WR-08 regression test: the pre-fix pattern was an unanchored PREFIX
// match, so a SAN merely starting with this repo's GitHub URL would pass
// — including a signature produced by an unrelated, weaker-trust-boundary
// workflow (e.g. a pull_request-triggered CI workflow) in the same repo,
// not just the intended tag-triggered release workflow.
func TestReleaseWorkflowRefPattern_RejectsNonReleaseWorkflowInSameRepo(t *testing.T) {
	re := regexp.MustCompile(releaseWorkflowRefPattern)
	unrelated := "https://github.com/" + releaseRepoSlug + "/.github/workflows/ci.yml@refs/heads/main"
	if re.MatchString(unrelated) {
		t.Fatalf("releaseWorkflowRefPattern must reject a non-release workflow in the same repo, matched: %q", unrelated)
	}
}

// TestReleaseWorkflowRefPattern_AcceptsReleaseWorkflowTagRef proves the
// anchored pattern still accepts the identity it's meant to authorize: the
// release workflow itself, triggered by a version-tag push.
func TestReleaseWorkflowRefPattern_AcceptsReleaseWorkflowTagRef(t *testing.T) {
	re := regexp.MustCompile(releaseWorkflowRefPattern)
	valid := "https://github.com/" + releaseRepoSlug + "/.github/workflows/release.yml@refs/tags/v1.2.3"
	if !re.MatchString(valid) {
		t.Fatalf("releaseWorkflowRefPattern must accept the release workflow's own tag-triggered ref, got no match for: %q", valid)
	}
}

// TestVerifyGo_NoExecUsage is a belt-and-suspenders in-process guard
// (mirrors the plan's own acceptance-criteria grep): verify.go must never
// shell out to a cosign CLI or any external tool (D-12).
func TestVerifyGo_NoExecUsage(t *testing.T) {
	data, err := os.ReadFile("verify.go")
	if err != nil {
		t.Fatalf("read verify.go: %v", err)
	}
	if strings.Contains(string(data), "os/exec") || strings.Contains(string(data), "exec.Command") {
		t.Fatal("verify.go must not use os/exec or exec.Command (D-12: in-process verification only)")
	}
}
