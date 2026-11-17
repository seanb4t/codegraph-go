package query

import (
	"errors"
	"os"
	"path/filepath"
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
	link := filepath.Join(repo, codegraphDirName)
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
	link := filepath.Join(repo, codegraphDirName, storeSubdir)
	symlinkT(t, outside, link)

	_, err := ResolveIndex(repo)
	wantOutsideRoot(t, err, link)

	if _, _, err := OpenAt(repo); err == nil {
		t.Fatal("OpenAt through a store link outside the repository: want an error")
	}
	wantEmptyDir(t, outside)
}

func TestResolveIndexAcceptsIndexLinkedInside(t *testing.T) {
	repo := t.TempDir()
	mkdirAll(t, filepath.Join(repo, "cg-real"))
	symlinkT(t, "cg-real", filepath.Join(repo, codegraphDirName))

	got, err := ResolveIndex(repo)
	if err != nil {
		t.Fatalf("ResolveIndex: %v", err)
	}
	if got.IndexRoot != repo {
		t.Fatalf("IndexRoot = %q, want %q", got.IndexRoot, repo)
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
	symlinkT(t, outside, filepath.Join(main, codegraphDirName))

	_, err := ResolveIndex(wt)
	wantOutsideRoot(t, err, filepath.Join(realpathT(t, main), codegraphDirName))
}

func TestCheckIndexInsideRefusesDanglingLink(t *testing.T) {
	tmp := t.TempDir()
	repo := mkdirAll(t, filepath.Join(tmp, "repo"))
	target := filepath.Join(tmp, "victim")
	link := filepath.Join(repo, codegraphDirName)
	symlinkT(t, target, link)

	err := CheckIndexInside(repo)
	wantOutsideRoot(t, err, link)
	var outside *IndexOutsideRootError
	if errors.As(err, &outside) && outside.Target != filepath.Join(realpathT(t, tmp), "victim") {
		t.Fatalf("Target = %q, want the link's target", outside.Target)
	}
}
