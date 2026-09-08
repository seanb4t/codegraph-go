package uiserver

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/query"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// buildBigSourceFixture indexes a synthetic single-file Go module whose
// ONE source file is exactly totalBytes long — a `package`/`func Big`
// header followed by a single long comment line padded to hit the target
// size exactly, mirroring D-11's own canonical example (a pathological
// minified single-line file). This isolates the byte cap from the line
// cap: the file's own line count stays tiny (a handful of short header
// lines plus one comment line), so any truncation this fixture triggers
// is attributable to sourceByteCap alone, never to sourceLineCap.
func buildBigSourceFixture(t *testing.T, totalBytes int) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/bigsource\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	const prefix = "package big\n\nfunc Big() {}\n\n//"
	padLen := totalBytes - len(prefix) - 1 // -1 reserves the trailing newline
	if padLen < 0 {
		t.Fatalf("buildBigSourceFixture: totalBytes %d is too small for the fixed %d-byte prefix", totalBytes, len(prefix)+1)
	}
	content := prefix + strings.Repeat("x", padLen) + "\n"
	if len(content) != totalBytes {
		t.Fatalf("buildBigSourceFixture: constructed content is %d bytes, want exactly %d", len(content), totalBytes)
	}
	if err := os.WriteFile(filepath.Join(dir, "big.go"), []byte(content), 0o644); err != nil {
		t.Fatalf("write big.go: %v", err)
	}

	indexGofixture(t, dir)
	return dir
}

// TestUIServiceSourceBlobTruncatesAnOversizedDefinitionSource proves an
// end-to-end GetNodeDetail (single-definition mode), through a real
// Connect client against a real listener, on a symbol whose file exceeds
// sourceByteCap returns a truncated, valid-UTF-8 source blob with the
// truncated flag set and the true totals — never a CodeResourceExhausted
// (RPC-05, D-11/D-12/D-13).
func TestUIServiceSourceBlobTruncatesAnOversizedDefinitionSource(t *testing.T) {
	const extraBytes = 5000
	wantTotalBytes := sourceByteCap + extraBytes
	dir := buildBigSourceFixture(t, wantTotalBytes)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "Big"}))
	if err != nil {
		t.Fatalf("GetNodeDetail(Big): %v, want a successful truncated response (not CodeResourceExhausted)", err)
	}
	if resp.Msg.GetMode() != uiv1.NodeDetailMode_NODE_DETAIL_MODE_SINGLE_DEF {
		t.Fatalf("GetNodeDetail(Big): mode = %v, want NODE_DETAIL_MODE_SINGLE_DEF (test fixture assumption)", resp.Msg.GetMode())
	}

	src := resp.Msg.GetSource()
	if src == nil {
		t.Fatal("GetNodeDetail(Big): source is nil, want a populated SourceBlob")
	}
	if !src.GetTruncated() {
		t.Fatal("GetNodeDetail(Big): truncated = false, want true for a source file exceeding sourceByteCap")
	}
	if int(src.GetTotalBytes()) != wantTotalBytes {
		t.Fatalf("GetNodeDetail(Big): total_bytes = %d, want %d (the real file size)", src.GetTotalBytes(), wantTotalBytes)
	}
	if len(src.GetContent()) > sourceByteCap {
		t.Fatalf("GetNodeDetail(Big): content is %d bytes, want at most sourceByteCap (%d)", len(src.GetContent()), sourceByteCap)
	}
	if !utf8.Valid(src.GetContent()) {
		t.Fatal("GetNodeDetail(Big): content is not valid UTF-8 — truncation split a rune")
	}
}

// TestUIServiceSourceBlobOnEveryAttachmentPoint proves the same property
// as the single-definition test above at the OTHER two attachment points
// this plan declares: each gathered candidate on a multi-definition
// symbol, and each group on an Explore response. Together with the
// single-definition attachment point (the test above), all three
// attachments SourceBlob is added to in this plan are asserted
// end-to-end.
func TestUIServiceSourceBlobOnEveryAttachmentPoint(t *testing.T) {
	t.Run("multi-def-candidates", func(t *testing.T) {
		dir := buildOverloadedFixture(t, "Overload", 2)
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		resp, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "Overload"}))
		if err != nil {
			t.Fatalf("GetNodeDetail(Overload): %v", err)
		}
		if len(resp.Msg.GetDefinitions()) != 2 {
			t.Fatalf("GetNodeDetail(Overload): got %d definitions, want 2 (test fixture assumption)", len(resp.Msg.GetDefinitions()))
		}

		var checked int
		for _, def := range resp.Msg.GetDefinitions() {
			if !def.GetDetailGathered() {
				continue
			}
			src := def.GetSource()
			if src == nil {
				t.Fatalf("GetNodeDetail(Overload): gathered definition[id=%s] has no source, want a populated SourceBlob", def.GetNode().GetId())
			}
			want, err := os.ReadFile(filepath.Join(dir, def.GetNode().GetFilePath()))
			if err != nil {
				t.Fatalf("independent read of %s: %v", def.GetNode().GetFilePath(), err)
			}
			wantTrunc := truncateSource(want)
			if !bytes.Equal(src.GetContent(), wantTrunc.Content) {
				t.Fatalf("definition[id=%s] source content does not match truncateSource computed independently over the same file", def.GetNode().GetId())
			}
			checked++
		}
		if checked == 0 {
			t.Fatal("no gathered definitions were checked — this subtest is vacuous")
		}
	})

	t.Run("explore-groups", func(t *testing.T) {
		dir := copyBehavioralFixture(t)
		indexGofixture(t, dir)
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		const q = "account balance"
		resp, err := client.Explore(context.Background(), connect.NewRequest(&uiv1.ExploreRequest{Query: q}))
		if err != nil {
			t.Fatalf("Explore(%q): %v", q, err)
		}
		if len(resp.Msg.GetGroups()) < 3 {
			t.Fatalf("Explore(%q) returned %d groups, want at least 3 (test fixture assumption)", q, len(resp.Msg.GetGroups()))
		}

		eng, closer, err := query.OpenAt(dir)
		if err != nil {
			t.Fatalf("query.OpenAt (independent verification path): %v", err)
		}
		defer closer.Close()
		want, err := eng.ExploreDetail(q, 0)
		if err != nil {
			t.Fatalf("eng.ExploreDetail(%q) (independent verification path): %v", q, err)
		}

		for _, g := range resp.Msg.GetGroups() {
			src := g.GetSource()
			if src == nil {
				t.Fatalf("group %q has no source, want a populated SourceBlob", g.GetPath())
			}
			wantSrc, ok := want.Sources[g.GetPath()]
			if !ok {
				t.Fatalf("group %q: no corresponding entry in independently-computed ExploreResult.Sources", g.GetPath())
			}
			wantTrunc := truncateSource(wantSrc)
			if !bytes.Equal(src.GetContent(), wantTrunc.Content) {
				t.Fatalf("group %q source content does not match truncateSource computed independently over ExploreResult.Sources[%q]", g.GetPath(), g.GetPath())
			}
		}
	})
}

// TestUIServiceSourceBlobGroupSourceMatchesItsGroupPath proves a group's
// source blob corresponds to THAT group's own path, verified by KEY
// lookup into the independently-computed ExploreResult.Sources — never
// by slice position — on a query producing at least three groups so a
// mapper that collapsed every group to the same or a positionally-shifted
// source could not pass this test by accident.
func TestUIServiceSourceBlobGroupSourceMatchesItsGroupPath(t *testing.T) {
	dir := copyBehavioralFixture(t)
	indexGofixture(t, dir)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	const q = "account balance"
	resp, err := client.Explore(context.Background(), connect.NewRequest(&uiv1.ExploreRequest{Query: q}))
	if err != nil {
		t.Fatalf("Explore(%q): %v", q, err)
	}
	if len(resp.Msg.GetGroups()) < 3 {
		t.Fatalf("Explore(%q) returned %d groups, want at least 3 (test fixture assumption)", q, len(resp.Msg.GetGroups()))
	}

	eng, closer, err := query.OpenAt(dir)
	if err != nil {
		t.Fatalf("query.OpenAt (independent verification path): %v", err)
	}
	defer closer.Close()
	want, err := eng.ExploreDetail(q, 0)
	if err != nil {
		t.Fatalf("eng.ExploreDetail(%q) (independent verification path): %v", q, err)
	}

	// Shuffle the verification order relative to response order by
	// looking every group up in the map keyed by ITS OWN path, rather
	// than iterating both slices in lockstep — a mapper that shifted
	// sources by one position would still pass a lockstep comparison if
	// every group's source happened to be the same length class, but
	// cannot pass a keyed lookup.
	for i, g := range resp.Msg.GetGroups() {
		wantSrc, ok := want.Sources[g.GetPath()]
		if !ok {
			t.Fatalf("group[%d] path %q has no corresponding key in the independently-computed ExploreResult.Sources map", i, g.GetPath())
		}
		wantTrunc := truncateSource(wantSrc)
		gotSrc := g.GetSource()
		if gotSrc == nil {
			t.Fatalf("group[%d] path %q has no source, want a populated SourceBlob", i, g.GetPath())
		}
		if !bytes.Equal(gotSrc.GetContent(), wantTrunc.Content) {
			t.Fatalf("group[%d] path %q: source content does not match ExploreResult.Sources[%q] (keyed lookup) run through truncateSource", i, g.GetPath(), g.GetPath())
		}
		if int(gotSrc.GetTotalBytes()) != wantTrunc.TotalBytes {
			t.Fatalf("group[%d] path %q: total_bytes = %d, want %d (ExploreResult.Sources[%q]'s real length)", i, g.GetPath(), gotSrc.GetTotalBytes(), wantTrunc.TotalBytes, g.GetPath())
		}
	}
}

// TestUIServiceSourceBlobAtTheCapClearsTheTransportBackstop proves a
// response at EXACTLY the application byte cap passes through the
// transport backstop without a CodeResourceExhausted (D-13): the failing
// input this test would catch is a backstop sized at or below
// sourceByteCap.
func TestUIServiceSourceBlobAtTheCapClearsTheTransportBackstop(t *testing.T) {
	dir := buildBigSourceFixture(t, sourceByteCap)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "Big"}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeResourceExhausted {
			t.Fatalf("GetNodeDetail(Big) at exactly sourceByteCap: CodeResourceExhausted, want success — the transport backstop is mis-sized at or below sourceByteCap: %v", err)
		}
		t.Fatalf("GetNodeDetail(Big) at exactly sourceByteCap: %v", err)
	}

	src := resp.Msg.GetSource()
	if src == nil {
		t.Fatal("GetNodeDetail(Big): source is nil, want a populated SourceBlob")
	}
	if src.GetTruncated() {
		t.Fatal("GetNodeDetail(Big): truncated = true at exactly sourceByteCap, want false (the byte-cap boundary is inclusive)")
	}
	if int(src.GetTotalBytes()) != sourceByteCap {
		t.Fatalf("GetNodeDetail(Big): total_bytes = %d, want exactly sourceByteCap (%d)", src.GetTotalBytes(), sourceByteCap)
	}
}

// TestUIServiceSourceBlobFileModeUsesTheSameTruncationPath proves a
// GetNodeDetail in file mode returns the file's source through the SAME
// truncation path as the definition modes, asserted by comparing against
// truncateSource's output computed independently in this test for the
// same input read straight off disk.
func TestUIServiceSourceBlobFileModeUsesTheSameTruncationPath(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	const relPath = "pkga/pkga.go"
	resp, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{File: relPath}))
	if err != nil {
		t.Fatalf("GetNodeDetail(file=%s): %v", relPath, err)
	}
	if resp.Msg.GetMode() != uiv1.NodeDetailMode_NODE_DETAIL_MODE_FILE {
		t.Fatalf("GetNodeDetail(file=%s): mode = %v, want NODE_DETAIL_MODE_FILE (test fixture assumption)", relPath, resp.Msg.GetMode())
	}

	want, err := os.ReadFile(filepath.Join(dir, relPath))
	if err != nil {
		t.Fatalf("independent read of %s: %v", relPath, err)
	}
	wantTrunc := truncateSource(want)

	src := resp.Msg.GetSource()
	if src == nil {
		t.Fatal("GetNodeDetail(file): source is nil, want a populated SourceBlob")
	}
	if !bytes.Equal(src.GetContent(), wantTrunc.Content) {
		t.Fatal("GetNodeDetail(file): content does not match truncateSource computed independently over the same input")
	}
	if src.GetTruncated() != wantTrunc.Truncated {
		t.Fatalf("GetNodeDetail(file): truncated = %v, want %v (independently computed)", src.GetTruncated(), wantTrunc.Truncated)
	}
	if int(src.GetTotalBytes()) != wantTrunc.TotalBytes || int(src.GetTotalLines()) != wantTrunc.TotalLines {
		t.Fatalf("GetNodeDetail(file): totals = (%d bytes, %d lines), want (%d bytes, %d lines) (independently computed)",
			src.GetTotalBytes(), src.GetTotalLines(), wantTrunc.TotalBytes, wantTrunc.TotalLines)
	}
	if int(src.GetReturnedBytes()) != wantTrunc.ReturnedBytes || int(src.GetReturnedLines()) != wantTrunc.ReturnedLines {
		t.Fatalf("GetNodeDetail(file): returned = (%d bytes, %d lines), want (%d bytes, %d lines) (independently computed)",
			src.GetReturnedBytes(), src.GetReturnedLines(), wantTrunc.ReturnedBytes, wantTrunc.ReturnedLines)
	}
}

// TestUIServiceSingleDefWithNoReadableFileStillAnswers proves CR-02's
// contract: a single-definition node whose source cannot be read is
// still answered — node, calls and called-by are returned and `source`
// is simply left UNSET — rather than failing the whole RPC. The source
// blob is an OPTIONAL enrichment on GetNodeDetailResponse (ui.proto's
// `optional SourceBlob source`); a definition that has no readable
// on-disk file is not thereby an unavailable node, and `codegraph node
// <symbol>` on the CLI answers both of these cases successfully today.
//
// Both reachable causes are covered, one subtest each:
//
//   - a package pseudo-node, which internal/indexer/resolve.go
//     constructs with Id/Kind/Name/QualifiedName and NO FilePath at all,
//     so the read is attempted against the empty string; and
//   - a stale index, where the indexed file was deleted from disk after
//     indexing (`stale` is a first-class state this product models).
//
// Each subtest asserts POSITIVELY that the response carries the node's
// own identity and that source is unset — never merely that no error
// occurred — so removing the degrade in nodeDetailToProto turns both
// subtests RED (the RPC fails outright).
func TestUIServiceSingleDefWithNoReadableFileStillAnswers(t *testing.T) {
	t.Run("package-pseudo-node-has-no-file-path", func(t *testing.T) {
		dir := copyGofixture(t)
		indexGofixture(t, dir)
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		// pkgb imports example.com/gofixture/pkga, so resolve.go creates
		// exactly one package pseudo-node named "pkga".
		resp, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "pkga"}))
		if err != nil {
			t.Fatalf("GetNodeDetail(pkga): %v, want a successful response with source left unset", err)
		}
		if resp.Msg.GetMode() != uiv1.NodeDetailMode_NODE_DETAIL_MODE_SINGLE_DEF {
			t.Fatalf("GetNodeDetail(pkga): mode = %v, want NODE_DETAIL_MODE_SINGLE_DEF (test fixture assumption)", resp.Msg.GetMode())
		}
		node := resp.Msg.GetNode()
		if node == nil {
			t.Fatal("GetNodeDetail(pkga): node is nil, want the package pseudo-node itself")
		}
		if node.GetName() != "pkga" {
			t.Fatalf("GetNodeDetail(pkga): node name = %q, want %q", node.GetName(), "pkga")
		}
		if node.GetFilePath() != "" {
			t.Fatalf("GetNodeDetail(pkga): node file_path = %q, want empty (test fixture assumption: package pseudo-nodes carry no FilePath)", node.GetFilePath())
		}
		if resp.Msg.Source != nil {
			t.Fatalf("GetNodeDetail(pkga): source = %v, want UNSET for a definition with no on-disk file", resp.Msg.Source)
		}
	})

	t.Run("indexed-file-deleted-after-indexing", func(t *testing.T) {
		dir := copyGofixture(t)
		indexGofixture(t, dir)
		if err := os.Remove(filepath.Join(dir, "pkga", "pkga.go")); err != nil {
			t.Fatalf("remove indexed source file: %v", err)
		}
		srv := startedServer(t, dir)
		client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

		resp, err := client.GetNodeDetail(context.Background(), connect.NewRequest(&uiv1.GetNodeDetailRequest{Symbol: "Alpha"}))
		if err != nil {
			t.Fatalf("GetNodeDetail(Alpha) after deleting its file: %v, want a successful response with source left unset", err)
		}
		if resp.Msg.GetMode() != uiv1.NodeDetailMode_NODE_DETAIL_MODE_SINGLE_DEF {
			t.Fatalf("GetNodeDetail(Alpha): mode = %v, want NODE_DETAIL_MODE_SINGLE_DEF (test fixture assumption)", resp.Msg.GetMode())
		}
		node := resp.Msg.GetNode()
		if node == nil || node.GetName() != "Alpha" {
			t.Fatalf("GetNodeDetail(Alpha): node = %v, want the Alpha definition itself", node)
		}
		if node.GetFilePath() != "pkga/pkga.go" {
			t.Fatalf("GetNodeDetail(Alpha): node file_path = %q, want %q (test fixture assumption)", node.GetFilePath(), "pkga/pkga.go")
		}
		if resp.Msg.Source != nil {
			t.Fatalf("GetNodeDetail(Alpha): source = %v, want UNSET once the indexed file no longer exists on disk", resp.Msg.Source)
		}
	})
}
