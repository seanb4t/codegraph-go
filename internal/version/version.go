// Package version holds the codegraph binary's build-identity metadata
// (D-09). Version/Commit/Date are ldflags-injected at build time via:
//
//	-X github.com/seanb4t/codegraph-go/internal/version.Version=<semver>
//	-X github.com/seanb4t/codegraph-go/internal/version.Commit=<sha>
//	-X github.com/seanb4t/codegraph-go/internal/version.Date=<RFC3339>
//
// When unset (go run/go test/plain go build), they default to
// dev/unknown so the binary always reports something sensible. Info()
// assembles these with the runtime Go version and target os/arch — no
// other computation happens here; this package has no network or crypto
// surface (T-06-05-02: build identity is only as trustworthy as the
// build that set these vars).
package version

import "runtime"

// Version, Commit, and Date are set via -ldflags -X at release build time
// (see package doc). They default to dev/unknown for unlinked builds.
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// VersionInfo is the full build-identity snapshot: the ldflags-injected
// triple plus the runtime Go version and target os/arch, json-tagged for
// `codegraph version --json`.
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// Info returns the current build's VersionInfo, combining the
// ldflags-injected Version/Commit/Date vars with runtime.Version()/
// GOOS/GOARCH.
func Info() VersionInfo {
	return VersionInfo{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}
