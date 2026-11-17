package query

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func symlinkT(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("symlink %s -> %s: %v", link, target, err)
	}
}

func wantOutsideRoot(t *testing.T, err error, link string) {
	t.Helper()
	var outside *IndexOutsideRootError
	if !errors.As(err, &outside) {
		t.Fatalf("err = %v, want an *IndexOutsideRootError", err)
	}
	if !errors.Is(err, ErrIndexOutsideRoot) {
		t.Fatalf("errors.Is(%v, ErrIndexOutsideRoot) = false", err)
	}
	if outside.Link != link {
		t.Fatalf("Link = %q, want %q", outside.Link, link)
	}
}

func wantEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	if len(entries) != 0 {
		t.Fatalf("%s holds %d entries, want none", dir, len(entries))
	}
}

func TestResolveIndexRefusesRootIndexLinkedOutside(t *testing.T) {
	tmp := t.TempDir()
	repo := mkdirAll(t, filepath.Join(tmp, "repo"))
	outside := mkdirAll(t, filepath.Join(tmp, "outside"))
	link := filepath.Join(repo, IndexDirName)
	symlinkT(t, outside, link)

	_, err := ResolveIndex(repo)
	wantOutsideRoot(t, err, link)

	if _, _, err := OpenAt(repo); err == nil {
		t.Fatal("OpenAt through a .codegraph link outside the repository: want an error")
	}
	wantEmptyDir(t, outside)
}

func TestResolveIndexRefusesRootStoreLinkedOutside(t *testing.T) {
	tmp := t.TempDir()
	repo := mkdirAll(t, filepath.Join(tmp, "repo"))
	outside := mkdirAll(t, filepath.Join(tmp, "outside"))
	markIndexed(t, repo)
	link := filepath.Join(repo, IndexDirName, StoreDirName)
	symlinkT(t, outside, link)

	_, err := ResolveIndex(repo)
	wantOutsideRoot(t, err, link)

	if _, _, err := OpenAt(repo); err == nil {
		t.Fatal("OpenAt through a store link outside the repository: want an error")
	}
	wantEmptyDir(t, outside)
}

func TestResolveIndexRefusesDanglingIndexLinkOutside(t *testing.T) {
	tmp := t.TempDir()
	parent := mkdirAll(t, filepath.Join(tmp, "parent"))
	markIndexed(t, parent)
	repo := mkdirAll(t, filepath.Join(parent, "repo"))
	link := filepath.Join(repo, IndexDirName)
	symlinkT(t, filepath.Join(tmp, "victim"), link)

	_, err := ResolveIndex(repo)
	wantOutsideRoot(t, err, link)
}

func TestResolveIndexRefusesSubdirectoryIndexLinkedToAncestorIndex(t *testing.T) {
	repo := t.TempDir()
	markIndexed(t, repo)
	sub := mkdirAll(t, filepath.Join(repo, "sub"))
	link := filepath.Join(sub, IndexDirName)
	symlinkT(t, filepath.Join("..", IndexDirName), link)

	_, err := ResolveIndex(sub)
	wantOutsideRoot(t, err, link)
}

func TestResolveIndexAcceptsIndexLinkedInside(t *testing.T) {
	repo := t.TempDir()
	mkdirAll(t, filepath.Join(repo, ".cg-real"))
	symlinkT(t, ".cg-real", filepath.Join(repo, IndexDirName))

	got, err := ResolveIndex(repo)
	if err != nil {
		t.Fatalf("ResolveIndex: %v", err)
	}
	if got.IndexRoot != repo {
		t.Fatalf("IndexRoot = %q, want %q", got.IndexRoot, repo)
	}
}

func TestResolveIndexAcceptsStoreLinkedInside(t *testing.T) {
	repo := t.TempDir()
	markIndexed(t, repo)
	mkdirAll(t, filepath.Join(repo, ".store-real"))
	symlinkT(t, filepath.Join("..", ".store-real"), filepath.Join(repo, IndexDirName, StoreDirName))

	if _, err := ResolveIndex(repo); err != nil {
		t.Fatalf("ResolveIndex: %v", err)
	}
}

func TestResolveIndexAcceptsPlainIndexDirectory(t *testing.T) {
	repo := t.TempDir()
	markIndexed(t, repo)
	start := mkdirAll(t, filepath.Join(repo, "pkg"))

	got, err := ResolveIndex(start)
	if err != nil {
		t.Fatalf("ResolveIndex: %v", err)
	}
	if got.IndexRoot != repo {
		t.Fatalf("IndexRoot = %q, want %q", got.IndexRoot, repo)
	}
}

func TestResolveIndexRefusesMainWorktreeIndexLinkedOutside(t *testing.T) {
	main, wt := siblingWorktree(t)
	outside := t.TempDir()
	symlinkT(t, outside, filepath.Join(main, IndexDirName))

	_, err := ResolveIndex(wt)
	wantOutsideRoot(t, err, filepath.Join(realpathT(t, main), IndexDirName))
}

func TestProjectIndexedFalseForIndexLinkedOutside(t *testing.T) {
	repo := t.TempDir()
	symlinkT(t, t.TempDir(), filepath.Join(repo, IndexDirName))

	if ProjectIndexed(repo) {
		t.Fatal("ProjectIndexed = true for a .codegraph that leads outside the project")
	}
}

func TestIndexDirsRefusesDanglingLink(t *testing.T) {
	tmp := t.TempDir()
	repo := mkdirAll(t, filepath.Join(tmp, "repo"))
	target := filepath.Join(tmp, "victim")
	link := filepath.Join(repo, IndexDirName)
	symlinkT(t, target, link)

	_, err := IndexDirs(repo)
	wantOutsideRoot(t, err, link)
	var outside *IndexOutsideRootError
	errors.As(err, &outside)
	if want := filepath.Join(realpathT(t, tmp), "victim"); outside.Target != want {
		t.Fatalf("Target = %q, want %q", outside.Target, want)
	}
	if want := realpathT(t, repo); outside.Root != want {
		t.Fatalf("Root = %q, want %q", outside.Root, want)
	}
	msg := err.Error()
	for _, part := range []string{strconv.Quote(link), strconv.Quote(outside.Target), "remove the link"} {
		if !strings.Contains(msg, part) {
			t.Errorf("message %q does not contain %s", msg, part)
		}
	}
}

func TestIndexDirsReturnsResolvedPaths(t *testing.T) {
	repo := t.TempDir()
	mkdirAll(t, filepath.Join(repo, ".cg-real"))
	symlinkT(t, ".cg-real", filepath.Join(repo, IndexDirName))

	got, err := IndexDirs(repo)
	if err != nil {
		t.Fatalf("IndexDirs: %v", err)
	}
	want := IndexPaths{
		Dir:   filepath.Join(realpathT(t, repo), ".cg-real"),
		Store: filepath.Join(realpathT(t, repo), ".cg-real", StoreDirName),
	}
	if got != want {
		t.Fatalf("IndexDirs = %+v, want %+v", got, want)
	}
}

func TestIndexDirsAcceptsIndexThatIsARegularFile(t *testing.T) {
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, IndexDirName), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := IndexDirs(repo); err != nil {
		t.Fatalf("IndexDirs: %v", err)
	}
}
