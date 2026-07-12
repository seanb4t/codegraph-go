package cli

import (
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/version"
)

// TestVersionInfoDefaults asserts version.Info() reports the dev/unknown
// defaults when no -ldflags -X injection has occurred (go run/go test),
// per D-09.
func TestVersionInfoDefaults(t *testing.T) {
	info := version.Info()

	if info.Version != "dev" {
		t.Errorf("Version = %q, want %q", info.Version, "dev")
	}
	if info.Commit != "unknown" {
		t.Errorf("Commit = %q, want %q", info.Commit, "unknown")
	}
	if info.Date != "unknown" {
		t.Errorf("Date = %q, want %q", info.Date, "unknown")
	}
	if info.GoVersion != runtime.Version() {
		t.Errorf("GoVersion = %q, want %q", info.GoVersion, runtime.Version())
	}
	if info.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", info.OS, runtime.GOOS)
	}
	if info.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", info.Arch, runtime.GOARCH)
	}
}

// TestVersionCommandPlain asserts `codegraph version` prints one line
// containing the version, commit, date, go version, and os/arch tokens.
func TestVersionCommandPlain(t *testing.T) {
	stdout, _, err := execCmd("version")
	if err != nil {
		t.Fatalf("version: %v", err)
	}

	for _, want := range []string{"dev", "unknown", runtime.Version(), runtime.GOOS, runtime.GOARCH} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output %q missing %q", stdout, want)
		}
	}
}

// TestVersionCommandJSON asserts `codegraph version --json` emits valid
// JSON that unmarshals to all six Info fields.
func TestVersionCommandJSON(t *testing.T) {
	stdout, _, err := execCmd("version", "--json")
	if err != nil {
		t.Fatalf("version --json: %v", err)
	}

	var got version.VersionInfo
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("unmarshal %q: %v", stdout, err)
	}

	if got.Version == "" || got.Commit == "" || got.Date == "" ||
		got.GoVersion == "" || got.OS == "" || got.Arch == "" {
		t.Errorf("got = %+v, want all fields populated", got)
	}
}

// TestRootVersionFlag asserts `codegraph --version` prints a non-empty
// version line (Cobra's built-in --version wiring, D-09).
func TestRootVersionFlag(t *testing.T) {
	stdout, _, err := execCmd("--version")
	if err != nil {
		t.Fatalf("--version: %v", err)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Error("--version printed empty output")
	}
}
