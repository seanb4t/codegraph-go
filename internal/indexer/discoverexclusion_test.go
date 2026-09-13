package indexer

import (
	"go/build"
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestDiscoverAll_RecordsBuildTagExclusion proves DiscoverAll's Discover
// decision point 3 (the Go build-tag MatchFile miss) now records an
// ExcludedFile with reason BUILD_TAG (Phase 10 HLT-05, D-01/D-08),
// alongside its unchanged Files/ModulePath result — the same walk result
// feeds both (D-01).
func TestDiscoverAll_RecordsBuildTagExclusion(t *testing.T) {
	root := t.TempDir()

	writeFixtureFile(t, root, "go.mod", "module example.com/tmp\n\ngo 1.24\n")
	writeFixtureFile(t, root, "real.go", "package tmp\n\nfunc Real() {}\n")
	writeFixtureFile(t, root, "tagged.go", "//go:build ignore\n\npackage tmp\n\nfunc Tagged() {}\n")

	d, err := DiscoverAll(root)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}

	if len(d.Files) != 1 || d.Files[0].RelPath != "real.go" {
		t.Fatalf("Files = %+v, want exactly [real.go]", d.Files)
	}
	if d.ModulePath != "example.com/tmp" {
		t.Fatalf("ModulePath = %q, want %q", d.ModulePath, "example.com/tmp")
	}

	if len(d.Excluded) != 1 {
		t.Fatalf("Excluded = %+v, want exactly 1 record", d.Excluded)
	}
	got := d.Excluded[0]
	if got.GetPath() != "tagged.go" {
		t.Fatalf("Excluded[0].Path = %q, want %q", got.GetPath(), "tagged.go")
	}
	if got.GetReason() != schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG {
		t.Fatalf("Excluded[0].Reason = %v, want EXCLUSION_REASON_BUILD_TAG", got.GetReason())
	}
	wantDetail := build.Default.GOOS + "/" + build.Default.GOARCH
	if got.GetDetail() != wantDetail {
		t.Fatalf("Excluded[0].Detail = %q, want %q (build.Default, never runtime.GOOS which a GOOS override would desynchronise)", got.GetDetail(), wantDetail)
	}
	wantSize := int64(len("//go:build ignore\n\npackage tmp\n\nfunc Tagged() {}\n"))
	if got.GetSizeBytes() != wantSize {
		t.Fatalf("Excluded[0].SizeBytes = %d, want %d", got.GetSizeBytes(), wantSize)
	}

	// Discover (the unchanged wrapper) still returns the same Files/
	// modulePath as before — 18 existing call sites must keep compiling
	// and behaving unchanged.
	files, modulePath, err := Discover(root)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(files) != 1 || files[0].RelPath != "real.go" {
		t.Fatalf("Discover Files = %+v, want exactly [real.go]", files)
	}
	if modulePath != "example.com/tmp" {
		t.Fatalf("Discover modulePath = %q, want %q", modulePath, "example.com/tmp")
	}
}

// writeFixtureFile writes contents to root/relPath, creating parent
// directories as needed, failing the test on any error.
func writeFixtureFile(t *testing.T, root, relPath, contents string) {
	t.Helper()
	abs := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(abs), err)
	}
	if err := os.WriteFile(abs, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
}
