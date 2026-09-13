package indexer

import (
	"go/build"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/parser"
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

	// Plan 02 also records go.mod itself as an UNSUPPORTED_EXTENSION
	// exclusion (decision point 2, since ".mod" is not a registered
	// language extension) — this test pins only the BUILD_TAG record's
	// own shape, so look it up by path rather than assuming it is the
	// walk's only record.
	byPath := make(map[string]*schema.ExcludedFile, len(d.Excluded))
	for _, x := range d.Excluded {
		byPath[x.GetPath()] = x
	}
	got, ok := byPath["tagged.go"]
	if !ok {
		t.Fatalf("Excluded = %+v, want a record for tagged.go", d.Excluded)
	}
	if got.GetReason() != schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG {
		t.Fatalf("tagged.go Reason = %v, want EXCLUSION_REASON_BUILD_TAG", got.GetReason())
	}
	wantDetail := build.Default.GOOS + "/" + build.Default.GOARCH
	if got.GetDetail() != wantDetail {
		t.Fatalf("tagged.go Detail = %q, want %q (build.Default, never runtime.GOOS which a GOOS override would desynchronise)", got.GetDetail(), wantDetail)
	}
	wantSize := int64(len("//go:build ignore\n\npackage tmp\n\nfunc Tagged() {}\n"))
	if got.GetSizeBytes() != wantSize {
		t.Fatalf("tagged.go SizeBytes = %d, want %d", got.GetSizeBytes(), wantSize)
	}
	if modGot, ok := byPath["go.mod"]; !ok || modGot.GetReason() != schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION {
		t.Fatalf("Excluded = %+v, want a go.mod UNSUPPORTED_EXTENSION record too", d.Excluded)
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

// writeExactSizeGoFile writes a syntactically valid, build-tag-free Go
// source file at exactly size bytes: a package clause and one trivial
// function (so go/build.Context.MatchFile — decision point 3 — finds no
// import section to scan and stops immediately), padded to the exact
// requested length with a single trailing line comment.
func writeExactSizeGoFile(t *testing.T, path string, size int64) {
	t.Helper()
	header := []byte("package tmp\n\nfunc Placeholder() {}\n\n// ")
	if int64(len(header))+1 > size {
		t.Fatalf("requested size %d too small for header+trailer (%d)", size, len(header)+1)
	}
	buf := make([]byte, size)
	copy(buf, header)
	for i := len(header); i < len(buf)-1; i++ {
		buf[i] = 'x'
	}
	buf[len(buf)-1] = '\n'
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TestDirExclusionReasonAgreesWithShouldSkipDir pins dirExclusionReason's
// verdict to ShouldSkipDir's for every row — the watcher (Phase 4 D-04)
// and the coverage record (Phase 10 D-02) can never diverge on which
// directories are pruned. dirExclusionReason's own body does not call
// ShouldSkipDir (see its doc comment) — this test is the guard, not a
// tautology.
func TestDirExclusionReasonAgreesWithShouldSkipDir(t *testing.T) {
	tests := []struct {
		name       string
		wantReason schema.ExclusionReason
	}{
		{"vendor", schema.ExclusionReason_EXCLUSION_REASON_DIR_VENDOR},
		{".git", schema.ExclusionReason_EXCLUSION_REASON_DIR_DOTPREFIX},
		{".codegraph", schema.ExclusionReason_EXCLUSION_REASON_DIR_DOTPREFIX},
		{".", schema.ExclusionReason_EXCLUSION_REASON_DIR_DOTPREFIX},
		{"src", schema.ExclusionReason_EXCLUSION_REASON_UNSPECIFIED},
		{"vendored", schema.ExclusionReason_EXCLUSION_REASON_UNSPECIFIED},
		{"node_modules", schema.ExclusionReason_EXCLUSION_REASON_UNSPECIFIED},
	}
	if len(tests) < 7 {
		t.Fatalf("table has %d rows, want at least 7", len(tests))
	}

	var sawOK, sawNotOK bool
	for _, tt := range tests {
		reason, ok := dirExclusionReason(tt.name)
		want := ShouldSkipDir(tt.name)
		if ok != want {
			t.Errorf("dirExclusionReason(%q) ok = %v, want ShouldSkipDir(%q) = %v", tt.name, ok, tt.name, want)
		}
		if ok {
			sawOK = true
			if reason != tt.wantReason {
				t.Errorf("dirExclusionReason(%q) reason = %v, want %v", tt.name, reason, tt.wantReason)
			}
		} else {
			sawNotOK = true
		}
	}
	if !sawOK || !sawNotOK {
		t.Fatalf("table must log at least one true and one false verdict: sawOK=%v sawNotOK=%v", sawOK, sawNotOK)
	}
	t.Logf("dirExclusionReason/ShouldSkipDir agreement rows checked: %d", len(tests))
}

// TestExceedsSizeLimitIsStrictlyGreaterThan pins HLT-05's boundary: a file
// of exactly parser.MaxSourceBytes is NOT excluded; one byte over IS.
func TestExceedsSizeLimitIsStrictlyGreaterThan(t *testing.T) {
	if exceedsSizeLimit(parser.MaxSourceBytes - 1) {
		t.Error("exceedsSizeLimit(MaxSourceBytes-1) = true, want false")
	}
	if exceedsSizeLimit(parser.MaxSourceBytes) {
		t.Error("exceedsSizeLimit(MaxSourceBytes) = true, want false (boundary is strict >)")
	}
	if !exceedsSizeLimit(parser.MaxSourceBytes + 1) {
		t.Error("exceedsSizeLimit(MaxSourceBytes+1) = false, want true")
	}
	if exceedsSizeLimit(0) {
		t.Error("exceedsSizeLimit(0) = true, want false")
	}
}

// TestDiscoverAll_RecordsAllFourReasons builds a tree exercising every one
// of the four decision points in one pass and asserts the exact Excluded
// slice (path, reason, detail, size) in byte-sorted path order, plus the
// unaffected Files list — the phantom-row check (D-02) that no path under
// a pruned directory ever gets its own record.
func TestDiscoverAll_RecordsAllFourReasons(t *testing.T) {
	root := t.TempDir()

	writeFixtureFile(t, root, "go.mod", "module example.com/tmp\n\ngo 1.26\n")
	writeFixtureFile(t, root, "real.go", "package tmp\n\nfunc Real() {}\n")
	writeFixtureFile(t, root, "notes.md", "# notes\n")
	writeFixtureFile(t, root, "data.JSON", "{}\n")
	writeFixtureFile(t, root, "tagged.go", "//go:build ignore\n\npackage tmp\n\nfunc Tagged() {}\n")
	writeFixtureFile(t, root, "vendor/x.go", "package x\n")
	writeFixtureFile(t, root, ".hidden/y.go", "package y\n")
	writeFixtureFile(t, root, ".hidden/deep/z.go", "package deep\n")
	writeFixtureFile(t, root, "vendor/sub/w.go", "package sub\n")

	hugeSize := int64(parser.MaxSourceBytes + 1)
	writeExactSizeGoFile(t, filepath.Join(root, "huge.go"), hugeSize)

	d, err := DiscoverAll(root)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}

	if len(d.Files) != 1 || d.Files[0].RelPath != "real.go" {
		t.Fatalf("Files = %+v, want exactly [real.go]", d.Files)
	}

	wantSizeDetail := strconv.FormatInt(hugeSize, 10) + " bytes > " + strconv.Itoa(parser.MaxSourceBytes)
	want := []*schema.ExcludedFile{
		newExcludedFile(".hidden", schema.ExclusionReason_EXCLUSION_REASON_DIR_DOTPREFIX, ".hidden", 0),
		newExcludedFile("data.JSON", schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION, ".json", 0),
		newExcludedFile("go.mod", schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION, ".mod", 0),
		newExcludedFile("huge.go", schema.ExclusionReason_EXCLUSION_REASON_SIZE_LIMIT, wantSizeDetail, hugeSize),
		newExcludedFile("notes.md", schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION, ".md", 0),
		newExcludedFile("tagged.go", schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG, build.Default.GOOS+"/"+build.Default.GOARCH, 0),
		newExcludedFile("vendor", schema.ExclusionReason_EXCLUSION_REASON_DIR_VENDOR, "vendor", 0),
	}

	t.Logf("excluded records: %d", len(d.Excluded))
	if len(d.Excluded) != 7 {
		t.Fatalf("len(Excluded) = %d, want 7: %+v", len(d.Excluded), d.Excluded)
	}
	for i, got := range d.Excluded {
		// The size/detail-independent fields we don't pin exactly above
		// (extension-file sizes and go.mod's size) vary by fixture
		// content — zero them out on both sides except for the two
		// records whose size IS load-bearing (huge.go).
		w := want[i]
		if got.GetPath() != w.GetPath() {
			t.Fatalf("Excluded[%d].Path = %q, want %q (order must be byte-sorted)", i, got.GetPath(), w.GetPath())
		}
		if got.GetReason() != w.GetReason() {
			t.Errorf("Excluded[%d] (%s) Reason = %v, want %v", i, got.GetPath(), got.GetReason(), w.GetReason())
		}
		if got.GetDetail() != w.GetDetail() {
			t.Errorf("Excluded[%d] (%s) Detail = %q, want %q", i, got.GetPath(), got.GetDetail(), w.GetDetail())
		}
		// Only huge.go and the two directory records have a pinned,
		// non-content-dependent size; assert those exactly.
		switch got.GetPath() {
		case "huge.go":
			if got.GetSizeBytes() != hugeSize {
				t.Errorf("Excluded[%d] (huge.go) SizeBytes = %d, want %d", i, got.GetSizeBytes(), hugeSize)
			}
		case ".hidden", "vendor":
			if got.GetSizeBytes() != 0 {
				t.Errorf("Excluded[%d] (%s) SizeBytes = %d, want 0 (directory records)", i, got.GetPath(), got.GetSizeBytes())
			}
		}
	}

	for _, got := range d.Excluded {
		p := got.GetPath()
		if strings.HasPrefix(p, "vendor/") || strings.HasPrefix(p, ".hidden/") {
			t.Errorf("phantom per-file record under a pruned directory: %s", p)
		}
	}
}

// TestDiscoverAll_ExactSizeLimitFileIsNotExcluded pins the other half of
// HLT-05's boundary: a file of EXACTLY parser.MaxSourceBytes is discovered
// normally, never excluded.
func TestDiscoverAll_ExactSizeLimitFileIsNotExcluded(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module example.com/tmp\n\ngo 1.26\n")
	writeExactSizeGoFile(t, filepath.Join(root, "exact.go"), int64(parser.MaxSourceBytes))

	d, err := DiscoverAll(root)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}
	if len(d.Files) != 1 || d.Files[0].RelPath != "exact.go" {
		t.Fatalf("Files = %+v, want exactly [exact.go] (exact MaxSourceBytes must NOT be excluded)", d.Files)
	}
	for _, x := range d.Excluded {
		if x.GetPath() == "exact.go" {
			t.Fatalf("exact.go was excluded (%v), want it discovered — boundary must be strict >", x)
		}
	}
}

// TestDiscoverAll_ExcludedSortedAndDeterministic asserts two calls over
// the same tree return proto.Equal Excluded slices in the same order.
func TestDiscoverAll_ExcludedSortedAndDeterministic(t *testing.T) {
	root := t.TempDir()
	writeFixtureFile(t, root, "go.mod", "module example.com/tmp\n\ngo 1.26\n")
	writeFixtureFile(t, root, "real.go", "package tmp\n")
	writeFixtureFile(t, root, "notes.md", "# notes\n")
	writeFixtureFile(t, root, "tagged.go", "//go:build ignore\n\npackage tmp\n")
	writeFixtureFile(t, root, "vendor/x.go", "package x\n")
	writeFixtureFile(t, root, ".hidden/y.go", "package y\n")

	first, err := DiscoverAll(root)
	if err != nil {
		t.Fatalf("DiscoverAll (first): %v", err)
	}
	second, err := DiscoverAll(root)
	if err != nil {
		t.Fatalf("DiscoverAll (second): %v", err)
	}

	if len(first.Excluded) != len(second.Excluded) {
		t.Fatalf("len(Excluded) first=%d second=%d, want equal", len(first.Excluded), len(second.Excluded))
	}
	if len(first.Excluded) == 0 {
		t.Fatal("Excluded is empty, fixture should have produced records")
	}
	for i := range first.Excluded {
		if first.Excluded[i].GetPath() != second.Excluded[i].GetPath() {
			t.Fatalf("Excluded[%d] path first=%q second=%q, want same order", i, first.Excluded[i].GetPath(), second.Excluded[i].GetPath())
		}
		if !proto.Equal(first.Excluded[i], second.Excluded[i]) {
			t.Fatalf("Excluded[%d] not proto.Equal: first=%v second=%v", i, first.Excluded[i], second.Excluded[i])
		}
	}
}
