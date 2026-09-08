package uiserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/query"
	"github.com/seanb4t/codegraph-go/internal/schema"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// TestFileSymbolsReturnsSymbolsInEngineOrder proves FileSymbols carries
// every value the engine returns for an indexed path, compared element
// for element against an independently-computed eng.FileSymbols() call
// on the SAME fixture (mirroring TestFileGraphProjectsEngineResult's
// convention) — not merely that the response is non-empty.
// pkga/pkga.go carries seven real symbols at distinct lines, so the
// element-for-element comparison is not satisfiable by length alone.
func TestFileSymbolsReturnsSymbolsInEngineOrder(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.FileSymbols(context.Background(), connect.NewRequest(&uiv1.FileSymbolsRequest{Path: "pkga/pkga.go"}))
	if err != nil {
		t.Fatalf("FileSymbols: %v", err)
	}

	eng, closer, err := query.OpenAt(dir)
	if err != nil {
		t.Fatalf("query.OpenAt (independent verification path): %v", err)
	}
	defer closer.Close()
	want, err := eng.FileSymbols("pkga/pkga.go")
	if err != nil {
		t.Fatalf("eng.FileSymbols (independent verification path): %v", err)
	}

	if len(want.Symbols) == 0 {
		t.Fatal("test fixture assumption broken: eng.FileSymbols's Symbols is empty")
	}
	if int(resp.Msg.GetTotalCount()) != int(want.Total) {
		t.Errorf("total_count = %d, want %d", resp.Msg.GetTotalCount(), want.Total)
	}
	if resp.Msg.GetTruncated() != want.Truncated {
		t.Errorf("truncated = %v, want %v", resp.Msg.GetTruncated(), want.Truncated)
	}
	if len(resp.Msg.GetSymbols()) != len(want.Symbols) {
		t.Fatalf("symbols has %d entries, want %d", len(resp.Msg.GetSymbols()), len(want.Symbols))
	}
	for i, ws := range want.Symbols {
		gs := resp.Msg.GetSymbols()[i]
		if gs.GetId() != ws.Id {
			t.Errorf("symbols[%d].id = %q, want %q", i, gs.GetId(), ws.Id)
		}
		if gs.GetKind() != ws.Kind {
			t.Errorf("symbols[%d].kind = %q, want %q", i, gs.GetKind(), ws.Kind)
		}
		if gs.GetName() != ws.Name {
			t.Errorf("symbols[%d].name = %q, want %q", i, gs.GetName(), ws.Name)
		}
		if gs.GetFilePath() != ws.FilePath {
			t.Errorf("symbols[%d].file_path = %q, want %q", i, gs.GetFilePath(), ws.FilePath)
		}
		if gs.GetStartLine() != ws.StartLine {
			t.Errorf("symbols[%d].start_line = %d, want %d", i, gs.GetStartLine(), ws.StartLine)
		}
	}
}

// TestFileSymbolsRejectsPathEscapingRepoRoot proves T-05-28: a path
// escaping the repository root returns a Connect invalid-argument error.
func TestFileSymbolsRejectsPathEscapingRepoRoot(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	_, err := client.FileSymbols(context.Background(), connect.NewRequest(&uiv1.FileSymbolsRequest{Path: "../../../../etc/passwd"}))
	if err == nil {
		t.Fatal("FileSymbols(../../../../etc/passwd): got nil error, want CodeInvalidArgument")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("FileSymbols(../../../../etc/passwd): code = %v, want CodeInvalidArgument", code)
	}
}

// TestFileSymbolsRejectsAbsolutePath proves T-05-28's second case: an
// absolute path returns a Connect invalid-argument error.
func TestFileSymbolsRejectsAbsolutePath(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	_, err := client.FileSymbols(context.Background(), connect.NewRequest(&uiv1.FileSymbolsRequest{Path: "/etc/passwd"}))
	if err == nil {
		t.Fatal("FileSymbols(/etc/passwd): got nil error, want CodeInvalidArgument")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("FileSymbols(/etc/passwd): code = %v, want CodeInvalidArgument", code)
	}
}

// TestFileSymbolsUnknownPathReturnsEmptyNotError proves a path the index
// does not carry returns an empty list, a zero total, and no error.
// go.mod exists on disk in the gofixture but is never indexed as a
// symbol source, and pkga/pkga.go — the sibling FileSymbols DOES index —
// proves the empty result is a real miss rather than a broken lookup.
func TestFileSymbolsUnknownPathReturnsEmptyNotError(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.FileSymbols(context.Background(), connect.NewRequest(&uiv1.FileSymbolsRequest{Path: "go.mod"}))
	if err != nil {
		t.Fatalf("FileSymbols(go.mod): %v", err)
	}
	if resp.Msg.GetTotalCount() != 0 {
		t.Fatalf("FileSymbols(go.mod): total_count = %d, want 0", resp.Msg.GetTotalCount())
	}
	if len(resp.Msg.GetSymbols()) != 0 {
		t.Fatalf("FileSymbols(go.mod): len(symbols) = %d, want 0", len(resp.Msg.GetSymbols()))
	}
	if resp.Msg.GetTruncated() {
		t.Fatal("FileSymbols(go.mod): truncated = true, want false")
	}

	sibling, err := client.FileSymbols(context.Background(), connect.NewRequest(&uiv1.FileSymbolsRequest{Path: "pkga/pkga.go"}))
	if err != nil {
		t.Fatalf("FileSymbols(pkga/pkga.go): %v", err)
	}
	if sibling.Msg.GetTotalCount() == 0 {
		t.Fatal("FileSymbols(pkga/pkga.go): total_count = 0, want > 0 (sibling path must still return its symbols)")
	}
}

// newFileSymbolsBigStore builds a real graphstore-backed fixture directly
// through graphstore.Writer (not the full indexer pipeline) — count
// symbol records, all with FilePath "big.go", plus a real "big.go" file
// on disk so ValidateRepoRelativePath's filesystem re-verification
// succeeds. This is what makes the cap test tractable: hand-writing (or
// parsing) a source file with thousands of real declarations is not.
func newFileSymbolsBigStore(t *testing.T, count int) string {
	t.Helper()
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "big.go"), []byte("package p\n"), 0o644); err != nil {
		t.Fatalf("write fixture file big.go: %v", err)
	}

	storeDir := filepath.Join(dir, ".codegraph", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store dir: %v", err)
	}
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("big-%05d", i)
		if err := w.PutNode(&schema.Node{
			Id:        id,
			Name:      fmt.Sprintf("Sym%05d", i),
			Kind:      "function",
			FilePath:  "big.go",
			Language:  "go",
			StartLine: int32(i + 1),
			EndLine:   int32(i + 1),
		}); err != nil {
			t.Fatalf("PutNode: %v", err)
		}
	}
	if err := w.PutMeta(&schema.Meta{SchemaVersion: schema.SchemaVersion, NodeCount: int64(count), Healthy: true}); err != nil {
		t.Fatalf("PutMeta: %v", err)
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	return dir
}

// TestFileSymbolsCappedResponseCarriesTruncationFlag proves T-05-30 at
// the wire layer: a file carrying more symbols than
// query.MaxFileSymbols returns a response whose total_count is strictly
// greater than the returned symbol count, with truncated set.
func TestFileSymbolsCappedResponseCarriesTruncationFlag(t *testing.T) {
	dir := newFileSymbolsBigStore(t, query.MaxFileSymbols+500)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.FileSymbols(context.Background(), connect.NewRequest(&uiv1.FileSymbolsRequest{Path: "big.go"}))
	if err != nil {
		t.Fatalf("FileSymbols(big.go): %v", err)
	}
	if len(resp.Msg.GetSymbols()) != query.MaxFileSymbols {
		t.Fatalf("FileSymbols(big.go): len(symbols) = %d, want exactly MaxFileSymbols (%d)", len(resp.Msg.GetSymbols()), query.MaxFileSymbols)
	}
	if int(resp.Msg.GetTotalCount()) <= len(resp.Msg.GetSymbols()) {
		t.Fatalf("FileSymbols(big.go): total_count = %d, want strictly greater than the returned length %d", resp.Msg.GetTotalCount(), len(resp.Msg.GetSymbols()))
	}
	if !resp.Msg.GetTruncated() {
		t.Fatal("FileSymbols(big.go): truncated = false, want true")
	}
}

// TestFileSymbolsCappedResponseFitsTransportCeiling is the wire layer's
// own stake in the engine's cap (Task 1 checkpoint sub-decision 2): it
// does not own query.MaxFileSymbols's value, it owns the check that the
// value is safe for THIS transport. A response carrying exactly
// query.MaxFileSymbols symbols with realistic field contents (every
// field a real symbol carries populated, not zero-valued) marshals to a
// byte count above zero — the floor proving this measured something
// real, not a degenerate empty message (rule 84d1gfpywd) — and below
// transportSendMaxBytes.
func TestFileSymbolsCappedResponseFitsTransportCeiling(t *testing.T) {
	symbols := make([]*uiv1.Node, query.MaxFileSymbols)
	for i := range symbols {
		symbols[i] = &uiv1.Node{
			Id:            fmt.Sprintf("sym-%05d", i),
			Kind:          "function",
			Name:          fmt.Sprintf("Symbol%05d", i),
			QualifiedName: fmt.Sprintf("bigpkg.Symbol%05d", i),
			FilePath:      "big.go",
			Language:      "go",
			StartLine:     int32(i + 1),
			EndLine:       int32(i + 2),
			StartCol:      1,
			EndCol:        40,
			Signature:     "func() error",
			Docstring:     "Symbol is a realistically-sized doc comment describing a generated symbol, long enough to be representative of a real hand-written one.",
			Visibility:    "public",
			IsExported:    true,
			ReturnType:    "error",
		}
	}
	resp := &uiv1.FileSymbolsResponse{
		Symbols:    symbols,
		TotalCount: int32(query.MaxFileSymbols + 500),
		Truncated:  true,
	}

	b, err := proto.Marshal(resp)
	if err != nil {
		t.Fatalf("proto.Marshal(FileSymbolsResponse): %v", err)
	}

	const ceiling = 16 * 1024 * 1024 // transportSendMaxBytes, internal/uiserver/truncate.go
	if len(b) == 0 {
		t.Fatal("serialized FileSymbolsResponse = 0 bytes, want > 0 (floor: proves this measured a fully-capped response)")
	}
	if len(b) >= ceiling {
		t.Fatalf("serialized FileSymbolsResponse = %d bytes, want < %d (transportSendMaxBytes ceiling)", len(b), ceiling)
	}
}

// TestFileSymbolsOpensEngineExactlyOnce proves the withEngine
// open-use-close discipline (SRV-04): one FileSymbols call invokes the
// package's openEngine seam exactly once, using the same
// counting-wrapper technique TestFileGraphOpensEngineExactlyOnce
// already establishes.
func TestFileSymbolsOpensEngineExactlyOnce(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	orig := openEngine
	t.Cleanup(func() { openEngine = orig })
	var opens int64
	openEngine = func(start string) (*query.Engine, io.Closer, error) {
		atomic.AddInt64(&opens, 1)
		return orig(start)
	}

	if _, err := client.FileSymbols(context.Background(), connect.NewRequest(&uiv1.FileSymbolsRequest{Path: "pkga/pkga.go"})); err != nil {
		t.Fatalf("FileSymbols: %v", err)
	}
	if got := atomic.LoadInt64(&opens); got != 1 {
		t.Fatalf("openEngine invoked %d times for one FileSymbols call, want exactly 1", got)
	}
}

// TestFileSymbolsDoesNotDegrade proves FileSymbols is NOT GetStatus:
// with the store held past graphstore.Open's bounded retry budget by a
// REAL graphstore.Open handle in this test process (the same technique
// TestFileGraphDoesNotDegrade uses), FileSymbols returns an ERROR
// through mapEngineError's ordinary translation — never a degraded
// empty response.
func TestFileSymbolsDoesNotDegrade(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)
	storeDir := filepath.Join(dir, ".codegraph", "store")

	holder, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("acquire holder graphstore.Open: %v", err)
	}
	t.Cleanup(func() { holder.Close() })

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	_, err = client.FileSymbols(context.Background(), connect.NewRequest(&uiv1.FileSymbolsRequest{Path: "pkga/pkga.go"}))
	if err == nil {
		t.Fatal("FileSymbols while the store is locked past the retry budget succeeded, want an error (FileSymbols must not degrade-and-answer)")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnavailable {
		t.Fatalf("FileSymbols while locked: code = %v, want CodeUnavailable", code)
	}
}
