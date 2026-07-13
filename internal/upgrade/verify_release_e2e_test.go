package upgrade

import (
	"fmt"
	"runtime"
	"testing"
)

// goReleaserAssetName reproduces .goreleaser.yaml's archives.name_template
// literally:
//
//	{{ .ProjectName }}_{{ .Tag }}_{{ .Os }}_{{ .Arch }}{{ if eq .Os "windows" }}.exe{{ end }}
//
// GoReleaser's default .Os/.Arch template values are the same strings as Go's
// own GOOS/GOARCH (this project sets no goarch/goos `replacements:` block in
// .goreleaser.yaml), so goos/goarch here are passed through unchanged.
func goReleaserAssetName(tag, goos, goarch string) string {
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("codegraph_%s_%s_%s%s", tag, goos, goarch, ext)
}

// TestReleaseAssetNameMatchesGoReleaser pins D-14: the asset name
// internal/upgrade downloads (releaseAssetName) MUST equal, byte for byte,
// the asset name .goreleaser.yaml's archives.name_template actually produces
// for every one of the 6 shipped release targets. A divergence here means
// `codegraph upgrade` 404s on a genuine, correctly-published release.
func TestReleaseAssetNameMatchesGoReleaser(t *testing.T) {
	const tag = "v1.2.3"

	// One literal expectation per shipped (goos,goarch) pair (RESEARCH.md
	// Finding 2 / .goreleaser.yaml's builds: list) — pinned independently of
	// both goReleaserAssetName and releaseAssetName so a bug shared by both
	// implementations can't hide from this test.
	pairs := []struct {
		goos, goarch string
		want         string
	}{
		{"linux", "amd64", "codegraph_v1.2.3_linux_amd64"},
		{"linux", "arm64", "codegraph_v1.2.3_linux_arm64"},
		{"windows", "amd64", "codegraph_v1.2.3_windows_amd64.exe"},
		{"windows", "arm64", "codegraph_v1.2.3_windows_arm64.exe"},
		{"darwin", "amd64", "codegraph_v1.2.3_darwin_amd64"},
		{"darwin", "arm64", "codegraph_v1.2.3_darwin_arm64"},
	}

	if len(pairs) != 6 {
		t.Fatalf("expected exactly 6 os/arch pairs (the shipped release matrix), got %d", len(pairs))
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
		t.Fatalf("host os/arch (%s/%s) is not one of the 6 pinned release pairs — releaseAssetName was never cross-checked against a real template literal", runtime.GOOS, runtime.GOARCH)
	}
}
