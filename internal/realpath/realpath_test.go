package realpath_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/realpath"
	"github.com/seanb4t/codegraph-go/internal/realpath/realpathtest"
)

func mkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}

func wantResolve(t *testing.T, path, want string) {
	t.Helper()
	got, err := realpath.Resolve(path)
	if err != nil {
		t.Fatalf("Resolve(%q): %v", path, err)
	}
	if got != want {
		t.Fatalf("Resolve(%q) = %q, want %q", path, got, want)
	}
}

func TestWithin_NameWithDotDotPrefixIsInside(t *testing.T) {
	if !realpath.Within("/p", "/p/..notes.md") {
		t.Fatal(`Within("/p", "/p/..notes.md") = false, want true`)
	}
}

func TestWithin_ParentIsOutside(t *testing.T) {
	if realpath.Within("/p/q", "/p/notes.md") {
		t.Fatal(`Within("/p/q", "/p/notes.md") = true, want false`)
	}
	if realpath.Within("/p/q", "/p") {
		t.Fatal(`Within("/p/q", "/p") = true, want false`)
	}
}

func TestWithin_SiblingWithSharedPrefixIsOutside(t *testing.T) {
	for _, path := range []string{"/pq/x", "/p-other", "/p.md"} {
		if realpath.Within("/p", path) {
			t.Errorf("Within(%q, %q) = true, want false", "/p", path)
		}
	}
}

func TestWithin_RootItselfIsInside(t *testing.T) {
	if !realpath.Within("/p", "/p") {
		t.Fatal(`Within("/p", "/p") = false, want true`)
	}
}

func TestResolve_RelativePathIsMadeAbsolute(t *testing.T) {
	dir := realpathtest.TempDir(t)
	t.Chdir(dir)
	wantResolve(t, "new.json", filepath.Join(dir, "new.json"))
}

func TestResolve_FollowsAChainOfLinks(t *testing.T) {
	dir := realpathtest.TempDir(t)
	if err := os.WriteFile(filepath.Join(dir, "real"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	realpathtest.Symlink(t, "real", filepath.Join(dir, "two"))
	realpathtest.Symlink(t, filepath.Join(dir, "two"), filepath.Join(dir, "one"))
	wantResolve(t, filepath.Join(dir, "one"), filepath.Join(dir, "real"))
}

func TestResolve_LinkToAMissingTarget(t *testing.T) {
	dir := realpathtest.TempDir(t)
	realpathtest.Symlink(t, filepath.Join(dir, "dotfiles", "claude.json"), filepath.Join(dir, "link"))
	wantResolve(t, filepath.Join(dir, "link"), filepath.Join(dir, "dotfiles", "claude.json"))
}

func TestResolve_LinkedParentOfAMissingPath(t *testing.T) {
	dir := realpathtest.TempDir(t)
	mkdir(t, filepath.Join(dir, "real"))
	realpathtest.Symlink(t, "real", filepath.Join(dir, "link"))
	wantResolve(t, filepath.Join(dir, "link", "sub", "new.json"), filepath.Join(dir, "real", "sub", "new.json"))
}

// With home/.claude -> dotfiles/claude and CLAUDE.md -> ../shared/CLAUDE.md,
// the target is dotfiles/shared/CLAUDE.md. A textual join names
// home/shared/CLAUDE.md.
func TestResolve_RelativeDotDotTargetUnderALinkedDirectory(t *testing.T) {
	dir := realpathtest.TempDir(t)
	mkdir(t, filepath.Join(dir, "home"))
	mkdir(t, filepath.Join(dir, "dotfiles", "claude"))
	realpathtest.Symlink(t, filepath.Join(dir, "dotfiles", "claude"), filepath.Join(dir, "home", ".claude"))
	realpathtest.Symlink(t, filepath.Join("..", "shared", "CLAUDE.md"), filepath.Join(dir, "dotfiles", "claude", "CLAUDE.md"))
	wantResolve(t, filepath.Join(dir, "home", ".claude", "CLAUDE.md"), filepath.Join(dir, "dotfiles", "shared", "CLAUDE.md"))
}

// With sub -> outside/a and AGENTS.md -> sub/../x.md, the target is
// outside/x.md. A textual join names x.md beside the link.
func TestResolve_MissingTargetWithDotDotAfterALinkedDirectory(t *testing.T) {
	dir := realpathtest.TempDir(t)
	mkdir(t, filepath.Join(dir, "project"))
	mkdir(t, filepath.Join(dir, "outside", "a"))
	realpathtest.Symlink(t, filepath.Join(dir, "outside", "a"), filepath.Join(dir, "project", "sub"))
	realpathtest.Symlink(t, "sub/../x.md", filepath.Join(dir, "project", "AGENTS.md"))
	wantResolve(t, filepath.Join(dir, "project", "AGENTS.md"), filepath.Join(dir, "outside", "x.md"))
}

func TestResolve_DotDotAfterAMissingElementIsAnError(t *testing.T) {
	dir := realpathtest.TempDir(t)
	realpathtest.Symlink(t, "missing/../x.md", filepath.Join(dir, "link"))
	if got, err := realpath.Resolve(filepath.Join(dir, "link")); err == nil {
		t.Fatalf("Resolve = %q, want an error", got)
	}
}

func TestResolve_LoopIsAnError(t *testing.T) {
	dir := realpathtest.TempDir(t)
	realpathtest.Symlink(t, filepath.Join(dir, "b"), filepath.Join(dir, "a"))
	realpathtest.Symlink(t, filepath.Join(dir, "a"), filepath.Join(dir, "b"))
	if got, err := realpath.Resolve(filepath.Join(dir, "a")); err == nil {
		t.Fatalf("Resolve = %q, want an error", got)
	}
}

// chain makes links l1 -> l2 -> ... -> ln -> missing/target and returns l1.
func chain(t *testing.T, dir string, n int) string {
	t.Helper()
	for i := 1; i <= n; i++ {
		target := filepath.Join(dir, fmt.Sprintf("l%d", i+1))
		if i == n {
			target = filepath.Join(dir, "missing", "target")
		}
		realpathtest.Symlink(t, target, filepath.Join(dir, fmt.Sprintf("l%d", i)))
	}
	return filepath.Join(dir, "l1")
}

func TestResolve_EightHopsThroughMissingTargetsResolve(t *testing.T) {
	dir := realpathtest.TempDir(t)
	wantResolve(t, chain(t, dir, 8), filepath.Join(dir, "missing", "target"))
}

func TestResolve_NineHopsThroughMissingTargetsFail(t *testing.T) {
	dir := realpathtest.TempDir(t)
	if got, err := realpath.Resolve(chain(t, dir, 9)); err == nil {
		t.Fatalf("Resolve = %q, want an error", got)
	}
}

// A missing directory does not count against the bound on links.
func TestResolve_MissingTargetUnderTenMissingDirectoriesResolves(t *testing.T) {
	dir := realpathtest.TempDir(t)
	deep := filepath.Join(dir, "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "file")
	realpathtest.Symlink(t, deep, filepath.Join(dir, "link"))
	wantResolve(t, filepath.Join(dir, "link"), deep)
}

// TestResolve_AgreesWithTheOperatingSystem compares Resolve with the
// operating system on random trees. For a path that exists, Resolve must
// return what filepath.EvalSymlinks returns. For a location that Resolve
// names for a path that does not exist, the test creates a file there and
// then asks the operating system where the path leads.
func TestResolve_AgreesWithTheOperatingSystem(t *testing.T) {
	base := realpathtest.TempDir(t)
	existing, created := 0, 0
	rapid.Check(t, func(rt *rapid.T) {
		root, err := os.MkdirTemp(base, "tree")
		if err != nil {
			rt.Fatal(err)
		}
		tree := realpathtest.Draw(rt, root)
		for _, p := range tree.Paths {
			osReal, osErr := filepath.EvalSymlinks(p)
			got, err := realpath.Resolve(p)
			if osErr == nil {
				if err != nil || got != osReal {
					rt.Fatalf("Resolve(%q) = %q, %v; the operating system reaches %q", p, got, err, osReal)
				}
				existing++
				continue
			}
			if err != nil {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(got), 0o755); err != nil {
				rt.Fatalf("Resolve(%q) = %q, and its directory cannot be made: %v", p, got, err)
			}
			if err := os.WriteFile(got, []byte("x"), 0o644); err != nil {
				rt.Fatalf("Resolve(%q) = %q, and no file can be written there: %v", p, got, err)
			}
			after, err := filepath.EvalSymlinks(p)
			if err != nil || after != got {
				rt.Fatalf("Resolve(%q) = %q, but after a file was made there the operating system reaches %q, %v", p, got, after, err)
			}
			created++
		}
	})
	if existing == 0 || created == 0 {
		t.Fatalf("compared %d existing paths and %d created paths, want both above zero", existing, created)
	}
}
