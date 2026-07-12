package upgrade

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"runtime"
)

// resolveLatestFunc/downloadFunc/verifyFunc/swapFunc are the injectable
// orchestration seams Options carries — upgrade_test.go drives Run's whole
// resolve→check?→download→verify→swap sequence with fakes so no test ever
// touches the network or performs a real crypto verification/file rename
// (those are covered directly by release_test.go/verify_test.go/
// swap_test.go). Production leaves these nil and Run substitutes the real
// implementations below.
type (
	resolveLatestFunc func(repoSlug string) (string, error)
	downloadFunc      func(version string) (binary []byte, bundleJSON []byte, err error)
	verifyFunc        func(binary, bundleJSON []byte) error
	swapFunc          func(targetPath string, newBinary []byte) error
)

// Options configures a single upgrade.Run invocation (D-11). The four
// unexported func fields are the injectable seams described above — zero
// value (nil) means "use the real implementation."
type Options struct {
	// Check, when true, resolves and reports the latest version WITHOUT
	// downloading, verifying, or swapping anything (D-11).
	Check bool

	// Version pins a specific release tag to install; empty means "the
	// resolved latest."
	Version string

	// Out receives human-readable status lines (the --check report, the
	// final "upgraded to vX" line). Defaults to io.Discard if nil.
	Out io.Writer

	resolveLatest resolveLatestFunc
	download      downloadFunc
	verify        verifyFunc
	swap          swapFunc
}

// Run executes `codegraph upgrade` end to end: resolve latest → (Check?
// report and return, no download) → refuse a non-writable targetPath
// BEFORE downloading (D-13) → download → verify (a non-nil error is FATAL
// and Run NEVER falls through to swap — Pitfall 7, T-06-06-01/02) → swap.
// targetPath is the self-replace target; the caller (internal/cli/
// upgrade.go) resolves it via os.Executable() so Run itself stays fully
// testable without touching the real running binary.
func Run(currentVersion, targetPath string, opts Options) error {
	out := opts.Out
	if out == nil {
		out = io.Discard
	}

	resolveLatest := opts.resolveLatest
	if resolveLatest == nil {
		resolveLatest = defaultResolveLatest
	}

	latest, err := resolveLatest(releaseRepoSlug)
	if err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}

	if opts.Check {
		if latest == currentVersion {
			fmt.Fprintf(out, "codegraph %s is up to date\n", currentVersion)
		} else {
			fmt.Fprintf(out, "codegraph %s is available (current: %s)\n", latest, currentVersion)
		}
		return nil
	}

	target := opts.Version
	if target == "" {
		target = latest
	}

	// D-13: refuse a non-writable target BEFORE downloading anything — no
	// wasted download for an upgrade that can't be installed anyway.
	if err := checkWritable(targetPath); err != nil {
		return err
	}

	download := opts.download
	if download == nil {
		download = defaultDownload
	}
	binary, bundleJSON, err := download(target)
	if err != nil {
		return fmt.Errorf("upgrade: download %s: %w", target, err)
	}

	verify := opts.verify
	if verify == nil {
		verify = defaultVerify
	}
	if err := verify(binary, bundleJSON); err != nil {
		// Pitfall 7 / T-06-06-01/02: a verification error is fatal. Never
		// fall through to swap — the original binary at targetPath is
		// left completely untouched.
		return fmt.Errorf("upgrade: refusing to install %s: signature verification failed: %w", target, err)
	}

	swap := opts.swap
	if swap == nil {
		swap = atomicSwap
	}
	if err := swap(targetPath, binary); err != nil {
		return fmt.Errorf("upgrade: install %s: %w", target, err)
	}

	fmt.Fprintf(out, "upgraded to %s\n", target)
	return nil
}

// defaultResolveLatest is production's resolveLatestFunc: the redirect
// trick against real GitHub Releases (release.go).
func defaultResolveLatest(repoSlug string) (string, error) {
	return resolveLatestVersion(repoSlug, newLatestRedirectClient())
}

// defaultVerify is production's verifyFunc: hash the downloaded binary,
// parse its accompanying bundle, fetch the live Sigstore trust root
// (verify.go's one network call), and run the full D-12 policy check
// pinned to this project's own release identity. releaseWorkflowRefPattern
// is Phase-8-finalized (D-14) — see verify.go's doc comment.
func defaultVerify(binary, bundleJSON []byte) error {
	b, err := loadBundle(bundleJSON)
	if err != nil {
		return err
	}
	trustedMaterial, err := fetchTrustedRoot()
	if err != nil {
		return err
	}
	digest := sha256.Sum256(binary)
	return verifyRelease(b, trustedMaterial, "sha256", digest[:], releaseWorkflowRefPattern)
}

// defaultDownload is production's downloadFunc: fetch the current
// os/arch's release asset plus its companion .sigstore.json bundle from
// this project's GitHub Releases for the given tag. The URL SHAPE
// (releases/download/<tag>/<asset>) is GitHub's own stable convention; the
// exact asset NAME is Phase-8-finalized (D-14, goreleaser's own naming
// convention, DIST-02) — this is a reasonable placeholder consistent with
// goreleaser's default archive-name template, not yet exercised against a
// real release. No Phase 6 test calls this directly — upgrade_test.go
// always injects Options.download.
func defaultDownload(version string) (binary []byte, bundleJSON []byte, err error) {
	assetName := releaseAssetName(version)

	binary, err = downloadReleaseAsset(version, assetName)
	if err != nil {
		return nil, nil, err
	}
	bundleJSON, err = downloadReleaseAsset(version, assetName+".sigstore.json")
	if err != nil {
		return nil, nil, err
	}
	return binary, bundleJSON, nil
}

// releaseAssetName is Phase-8-finalized (D-14): goreleaser's real archive
// naming template determines the production value once DIST-02 ships.
func releaseAssetName(version string) string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("codegraph_%s_%s_%s%s", version, runtime.GOOS, runtime.GOARCH, ext)
}

// downloadReleaseAsset fetches one named asset from this project's GitHub
// Releases for the given tag.
func downloadReleaseAsset(tag, assetName string) ([]byte, error) {
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", releaseRepoSlug, tag, assetName)

	resp, err := http.Get(url) //nolint:gosec,noctx // release asset URL is built from a fixed repo slug + caller-provided tag, not arbitrary user input
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: unexpected status %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}
