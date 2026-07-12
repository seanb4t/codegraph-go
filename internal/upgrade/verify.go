package upgrade

import (
	"context"
	"fmt"
	"time"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

// fetchTrustedRootTimeout bounds fetchTrustedRoot's TUF network round trip
// (WR-02): a hung or slow-lorising Sigstore TUF endpoint must never block
// `codegraph upgrade` indefinitely with no way for the user to know it's
// stuck versus still working.
const fetchTrustedRootTimeout = 30 * time.Second

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
//
// releaseWorkflowRefPattern is a FULL-MATCH pattern (WR-08): anchored at
// both ^ and $, and scoped to the specific release-publishing workflow
// file and a tag-triggered ref — not merely a "starts with this repo's
// URL" prefix. An unanchored/prefix-only pattern would authorize a
// signature produced by ANY workflow in this repo (including a much
// weaker trust boundary, e.g. a pull_request-triggered CI workflow), not
// just the intended release workflow. The workflow filename
// (release.yml) matches goreleaser's own convention and is expected to
// be finalized alongside DIST-02 (D-14) — update this pattern if the
// actual shipped workflow file is ever renamed.
const (
	releaseOIDCIssuer         = "https://token.actions.githubusercontent.com"
	releaseRepoSlug           = "seanb4t/codegraph-go"
	releaseWorkflowRefPattern = `^https://github\.com/` + releaseRepoSlug + `/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^\s]*$`
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
//
// A fetchTrustedRootTimeout-bounded context is threaded through to the TUF
// client via tuf.Options.WithContext (WR-02) so a hung Sigstore endpoint
// can't block `codegraph upgrade` indefinitely.
func fetchTrustedRoot() (root.TrustedMaterial, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTrustedRootTimeout)
	defer cancel()

	tr, err := root.FetchTrustedRootWithOptions(tuf.DefaultOptions().WithContext(ctx))
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
