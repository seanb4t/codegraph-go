package graphstore

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestExcludedFileNamespaceRoundTrip proves the c/ namespace's basic
// put/commit/iterate lifecycle (Phase 10 D-05/D-07): records staged on
// one Writer and committed are read back, in path-sorted order, with
// every field intact; an empty store yields zero records and no error.
func TestExcludedFileNamespaceRoundTrip(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	// A fresh, empty store: zero records, Err() == nil.
	emptySnap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot (empty): %v", err)
	}
	emptyIt, err := emptySnap.IterateExcludedFiles()
	if err != nil {
		t.Fatalf("IterateExcludedFiles (empty): %v", err)
	}
	count := 0
	for emptyIt.Next() {
		count++
	}
	if err := emptyIt.Err(); err != nil {
		t.Fatalf("empty iteration Err: %v", err)
	}
	if err := emptyIt.Close(); err != nil {
		t.Fatalf("empty iteration Close: %v", err)
	}
	if count != 0 {
		t.Fatalf("empty store yielded %d ExcludedFile records, want 0", count)
	}
	if err := emptySnap.Close(); err != nil {
		t.Fatalf("emptySnap.Close: %v", err)
	}

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	recB := &schema.ExcludedFile{Path: "b.md", Reason: schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION, Detail: ".md", SizeBytes: 12}
	recA := &schema.ExcludedFile{Path: "a/tagged.go", Reason: schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG, Detail: "linux/amd64", SizeBytes: 34}

	// Stage in reverse-of-key order to prove the READ side sorts, not
	// insertion order. Note: the expected order below is the encoded KEY
	// byte order (excludedFileKey's length-prefixed appendSegment
	// encoding, T-01-02), not raw path-string lexical order — a
	// length-prefixed key sorts by its length byte before its content
	// bytes when two paths' lengths differ, so "b.md" (length 4, prefix
	// byte 0x04) sorts before "a/tagged.go" (length 11, prefix byte
	// 0x0B) even though "a" < "b" lexically. This is the same encoding
	// fileKey/nodeKey already use and is required by T-01-02 (a
	// non-length-prefixed scheme would let a crafted path forge a key
	// that lands in the wrong range).
	if err := w.PutExcludedFile(recB); err != nil {
		t.Fatalf("PutExcludedFile(b.md): %v", err)
	}
	if err := w.PutExcludedFile(recA); err != nil {
		t.Fatalf("PutExcludedFile(a/tagged.go): %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	t.Cleanup(func() { _ = snap.Close() })

	it, err := snap.IterateExcludedFiles()
	if err != nil {
		t.Fatalf("IterateExcludedFiles: %v", err)
	}
	t.Cleanup(func() { _ = it.Close() })

	var got []*schema.ExcludedFile
	for it.Next() {
		got = append(got, it.ExcludedFile())
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iteration Err: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2", len(got))
	}
	if got[0].GetPath() != "b.md" || got[1].GetPath() != "a/tagged.go" {
		t.Fatalf("records not in key byte order: got [%q, %q], want [\"b.md\", \"a/tagged.go\"]", got[0].GetPath(), got[1].GetPath())
	}
	if got[0].GetReason() != schema.ExclusionReason_EXCLUSION_REASON_UNSUPPORTED_EXTENSION || got[0].GetDetail() != ".md" || got[0].GetSizeBytes() != 12 {
		t.Fatalf("record 0 = %+v, want reason=UNSUPPORTED_EXTENSION detail=.md size=12", got[0])
	}
	if got[1].GetReason() != schema.ExclusionReason_EXCLUSION_REASON_BUILD_TAG || got[1].GetDetail() != "linux/amd64" || got[1].GetSizeBytes() != 34 {
		t.Fatalf("record 1 = %+v, want reason=BUILD_TAG detail=linux/amd64 size=34", got[1])
	}
}

// TestExcludedFileKeyStartsWithPrefixAndIsolatesNeighbours proves the
// length-prefixed key encoding used elsewhere in this package (T-01-02)
// also protects the c/ namespace: distinct paths never alias, the key
// never starts with a foreign namespace's prefix byte, and an
// adversarial path (containing '/', NUL, 0xFF) round-trips through
// Put/Iterate unchanged.
func TestExcludedFileKeyStartsWithPrefixAndIsolatesNeighbours(t *testing.T) {
	foo := excludedFileKey("foo")
	foobar := excludedFileKey("foobar")
	if string(foo) == string(foobar) {
		t.Fatalf("excludedFileKey(%q) == excludedFileKey(%q), want distinct", "foo", "foobar")
	}
	for _, k := range [][]byte{foo, foobar} {
		if len(k) == 0 || k[0] != prefixExcludedFile {
			t.Fatalf("excludedFileKey key %x does not start with prefixExcludedFile %q", k, prefixExcludedFile)
		}
	}
	for _, foreign := range []byte{prefixMeta, prefixNode, prefixEdge, prefixFile, prefixAnnotation, prefixFileIndex} {
		if foo[0] == foreign {
			t.Fatalf("excludedFileKey leading byte %q collides with foreign namespace %q", foo[0], foreign)
		}
	}

	// The path is a Node/File/ExcludedFile proto3 STRING field, which
	// protobuf-go validates as well-formed UTF-8 at marshal time — a raw
	// invalid byte (e.g. a lone 0xFF) would fail Put with a marshal
	// error unrelated to the key-encoding property under test here. "\xc3\xbf"
	// (U+00FF, valid two-byte UTF-8) exercises the same "goes near byte
	// 0xFF" boundary intent without violating that separate contract; the
	// slash and NUL are both legal UTF-8 on their own.
	adversarial := "a/b\x00cÿd"
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	rec := &schema.ExcludedFile{Path: adversarial, Reason: schema.ExclusionReason_EXCLUSION_REASON_DIR_VENDOR, Detail: "vendor", SizeBytes: 0}
	if err := w.PutExcludedFile(rec); err != nil {
		t.Fatalf("PutExcludedFile(adversarial): %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	snap, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	t.Cleanup(func() { _ = snap.Close() })

	it, err := snap.IterateExcludedFiles()
	if err != nil {
		t.Fatalf("IterateExcludedFiles: %v", err)
	}
	t.Cleanup(func() { _ = it.Close() })

	if !it.Next() {
		t.Fatal("expected one record, got none")
	}
	got := it.ExcludedFile()
	if got.GetPath() != adversarial {
		t.Fatalf("adversarial path did not round-trip: got %q, want %q", got.GetPath(), adversarial)
	}
	if it.Next() {
		t.Fatal("expected exactly one record")
	}
	if err := it.Err(); err != nil {
		t.Fatalf("iteration Err: %v", err)
	}
}
