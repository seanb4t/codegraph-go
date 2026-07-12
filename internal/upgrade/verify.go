package upgrade

import (
	"fmt"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

// Named release-identity constants (D-12/D-14). releaseOIDCIssuer is
// GitHub Actions' Sigstore-public-good OIDC issuer — stable, not a
// placeholder. releaseRepoSlug and releaseWorkflowRefPattern ARE
// Phase-8-finalized: they compile and are wired into upgrade.Run's
// production identity policy today, but the actual production values (this
// project's own "owner/repo" and its release workflow's SAN pattern) are
// only meaningful once DIST-02 ships real signed releases — until then no
// production upgrade traffic exercises them (Phase 6 tests exercise
// verifyRelease directly against a fixture identity instead, see
// verify_test.go).
const (
	releaseOIDCIssuer         = "https://token.actions.githubusercontent.com"
	releaseRepoSlug           = "seanb4t/codegraph-go"
	releaseWorkflowRefPattern = "^https://github.com/" + releaseRepoSlug + "/"
)

// loadBundle parses a downloaded release's sigstore signature bundle from
// its raw JSON bytes. Never touches the network itself — the caller has
// already downloaded bundleJSON.
func loadBundle(bundleJSON []byte) (*bundle.Bundle, error) {
	b := &bundle.Bundle{}
	if err := b.UnmarshalJSON(bundleJSON); err != nil {
		return nil, fmt.Errorf("upgrade: parse release signature bundle: %w", err)
	}
	return b, nil
}

// fetchTrustedRoot fetches the live Sigstore public-good TUF trust root —
// the only network call this file itself makes. Production's upgrade.Run
// calls this once per invocation; verify_test.go never calls it (it loads
// an embedded fixture trust root instead, per Open Question 1), so `go
// test` never touches the network for the security-critical reject-path
// assertion.
func fetchTrustedRoot() (root.TrustedMaterial, error) {
	tr, err := root.FetchTrustedRoot()
	if err != nil {
		return nil, fmt.Errorf("upgrade: fetch sigstore trusted root: %w", err)
	}
	return tr, nil
}

// verifyRelease is the D-12 security gate: it checks, fully in-process
// (never shells out to a cosign CLI or any other external tool), that b
// was signed by an identity matching sanRegex under releaseOIDCIssuer AND
// that artifactDigest — the hash of the bytes ACTUALLY downloaded —
// matches the digest recorded inside the signed bundle. A non-nil error
// means the artifact is untrusted; callers (upgrade.Run) MUST treat this
// as fatal and MUST NOT proceed to the atomic swap (Pitfall 7,
// T-06-06-01/T-06-06-02): verification happens on the downloaded bytes
// BEFORE any file at the final target path is touched.
func verifyRelease(b *bundle.Bundle, trustedMaterial root.TrustedMaterial, digestAlgorithm string, artifactDigest []byte, sanRegex string) error {
	verifier, err := verify.NewVerifier(trustedMaterial,
		verify.WithSignedCertificateTimestamps(1),
		verify.WithTransparencyLog(1),
		verify.WithObserverTimestamps(1),
	)
	if err != nil {
		return fmt.Errorf("upgrade: build verifier: %w", err)
	}

	certID, err := verify.NewShortCertificateIdentity(releaseOIDCIssuer, "", "", sanRegex)
	if err != nil {
		return fmt.Errorf("upgrade: build identity policy: %w", err)
	}

	policy := verify.NewPolicy(
		verify.WithArtifactDigest(digestAlgorithm, artifactDigest),
		verify.WithCertificateIdentity(certID),
	)

	// A non-nil err here IS the verification failure — return it unchanged
	// so the caller can distinguish "verification ran and rejected this
	// bundle" from a construction error above, both of which are equally
	// fatal to the caller (never fall through to swap either way).
	_, err = verifier.Verify(b, policy)
	return err
}
