package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEphemeralExecPath(t *testing.T) {
	tmp := "/var/folders/xy/T"
	cache := "/Users/x/Library/Caches/go-build"
	hash := strings.Repeat("6959dc6a", 8)

	cases := []struct {
		path string
		want bool
	}{
		{"/usr/local/bin/codegraph", false},
		{tmp + "/go-build1/b001/cli.test", true},
		{tmp + "/go-build1/exe/codegraph", true},
		{"/opt/go-builder/codegraph", false},
		{"/home/x/foo.test", true},
		{cache + "/69/" + hash + "-d/codegraph", true},
		{cache + "/69/" + hash + "/codegraph", false},
	}
	for _, tc := range cases {
		if got := ephemeralExecPath(tc.path); got != tc.want {
			t.Errorf("ephemeralExecPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestStableExecPath(t *testing.T) {
	root := t.TempDir()
	file := func(rel string) string {
		t.Helper()
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("bin"), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	link := func(target, rel string) {
		t.Helper()
		if err := os.Symlink(target, filepath.Join(root, rel)); err != nil {
			t.Fatal(err)
		}
	}

	miseSame := file("mise/installs/codegraph/0.16.3/codegraph")
	link("./0.16.3", "mise/installs/codegraph/latest")

	miseOld := file("other/installs/codegraph/0.15.0/codegraph")
	file("other/installs/codegraph/0.16.3/codegraph")
	link("./0.16.3", "other/installs/codegraph/latest")

	asdf := file("asdf/installs/codegraph/0.16.3/bin/codegraph")

	broken := file("broken/installs/codegraph/0.16.3/codegraph")
	link("./0.99.0", "broken/installs/codegraph/latest")

	cask := file("brew/Caskroom/codegraph/0.16.3/codegraph")
	if err := os.MkdirAll(filepath.Join(root, "brew/bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	link("../Caskroom/codegraph/0.16.3/codegraph", "brew/bin/codegraph")

	caskNoLink := file("brew2/Caskroom/codegraph/0.16.3/codegraph")

	cases := []struct {
		name          string
		path          string
		wantRecorded  string
		wantVersioned bool
	}{
		{"mise latest names the same version", miseSame, filepath.Join(root, "mise/installs/codegraph/latest/codegraph"), false},
		{"mise latest names another version", miseOld, miseOld, true},
		{"asdf has no latest", asdf, asdf, true},
		{"broken latest link", broken, broken, true},
		{"cask with its bin link", cask, filepath.Join(root, "brew/bin/codegraph"), false},
		{"cask without the bin link", caskNoLink, caskNoLink, true},
		{"already latest", filepath.Join(root, "mise/installs/codegraph/latest/codegraph"), filepath.Join(root, "mise/installs/codegraph/latest/codegraph"), false},
		{"not a package-manager path", "/usr/local/bin/codegraph", "/usr/local/bin/codegraph", false},
		{"installs with one element after it", "/opt/installs/codegraph", "/opt/installs/codegraph", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorded, versioned := StableExecPath(tc.path)
			if recorded != tc.wantRecorded || versioned != tc.wantVersioned {
				t.Errorf("StableExecPath(%q) = (%q, %v), want (%q, %v)", tc.path, recorded, versioned, tc.wantRecorded, tc.wantVersioned)
			}
		})
	}
}

func TestInstallOptionsValidate(t *testing.T) {
	ephemeral := "/var/folders/xy/T/go-build1/b001/cli.test"

	if err := (InstallOptions{ExecPath: "/usr/local/bin/codegraph"}).Validate(); err != nil {
		t.Fatalf("durable path: Validate() = %v, want nil", err)
	}
	err := (InstallOptions{ExecPath: ephemeral}).Validate()
	if err == nil {
		t.Fatal("ephemeral path, no override: Validate() = nil, want an error")
	}
	for _, want := range []string{ephemeral, AllowEphemeralExecPathEnv} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Validate() error %q does not name %q", err, want)
		}
	}
	if err := (InstallOptions{ExecPath: ephemeral, AllowEphemeralExecPath: true}).Validate(); err != nil {
		t.Fatalf("ephemeral path, override set: Validate() = %v, want nil", err)
	}
}

// TestEveryTargetRefusesEphemeralExecPath: every registered target, at
// every location it supports, refuses a Go test binary path with one error
// naming it and writes nothing to the home or the project. The same call
// with the override writes there, so the empty directories are a refusal.
func TestEveryTargetRefusesEphemeralExecPath(t *testing.T) {
	ephemeral := filepath.Join(t.TempDir(), "go-build1", "b001", "agents.test")
	targets := AllTargets()
	if len(targets) == 0 {
		t.Fatal("the registry holds no targets")
	}

	ran := 0
	for _, target := range targets {
		for _, loc := range []Location{LocationGlobal, LocationLocal} {
			if !target.SupportsLocation(loc) {
				continue
			}
			t.Run(string(target.ID())+"/"+string(loc), func(t *testing.T) {
				ran++
				home := fakeHome(t)
				project := t.TempDir()
				t.Chdir(project)

				res := target.Install(loc, InstallOptions{ExecPath: ephemeral, PreToolNudge: PreToolNudgeOn})
				if len(res.Errors) != 1 || !strings.Contains(res.Errors[0].Error(), ephemeral) {
					t.Fatalf("Install errors = %v, want one error naming %s", res.Errors, ephemeral)
				}
				if len(res.Files) != 0 {
					t.Fatalf("Install reported files %v for a refused path", res.Files)
				}
				if n := dirEntryCount(t, home) + dirEntryCount(t, project); n != 0 {
					t.Fatalf("a refused install left %d entries in the home and project", n)
				}

				res = target.Install(loc, InstallOptions{ExecPath: ephemeral, AllowEphemeralExecPath: true})
				if len(res.Errors) != 0 {
					t.Fatalf("override set: Install errors: %v", res.Errors)
				}
				if dirEntryCount(t, home)+dirEntryCount(t, project) == 0 {
					t.Fatal("override set: Install wrote nothing to the home or project, so the refusal check cannot see a write")
				}
			})
		}
	}
	if ran < len(targets) {
		t.Fatalf("ran %d target/location cases, want at least one per target (%d)", ran, len(targets))
	}
}

// dirEntryCount returns how many entries dir holds.
func dirEntryCount(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	return len(entries)
}
