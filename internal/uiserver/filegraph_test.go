package uiserver

import (
	"context"
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
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
	"github.com/seanb4t/codegraph-go/internal/uiproto/uiv1/uiv1connect"
)

// TestFileGraphProjectsEngineResult proves FileGraph carries every value
// ENG-03/GRF-02/GRF-04 need, compared field-for-field against an
// independently-computed eng.FileGraph() call for the SAME fixture — not
// merely that the response is non-empty (RPC-01/RPC-02). Nodes and edges
// are compared element for element (both scans of the same store snapshot
// produce the SAME deterministic sorted order, internal/query/traverse.go),
// not by length alone — a length-only assertion passes on entirely wrong
// contents.
func TestFileGraphProjectsEngineResult(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.FileGraph(context.Background(), connect.NewRequest(&uiv1.FileGraphRequest{}))
	if err != nil {
		t.Fatalf("FileGraph: %v", err)
	}

	eng, closer, err := query.OpenAt(dir)
	if err != nil {
		t.Fatalf("query.OpenAt (independent verification path): %v", err)
	}
	defer closer.Close()
	want, err := eng.FileGraph()
	if err != nil {
		t.Fatalf("eng.FileGraph (independent verification path): %v", err)
	}

	got := resp.Msg

	if got.GetExcludedPackageNodeCount() != want.ExcludedPackageNodes {
		t.Errorf("excluded_package_node_count = %d, want %d", got.GetExcludedPackageNodeCount(), want.ExcludedPackageNodes)
	}
	if got.GetExcludedSelfEdgeCount() != want.ExcludedSelfEdges {
		t.Errorf("excluded_self_edge_count = %d, want %d", got.GetExcludedSelfEdgeCount(), want.ExcludedSelfEdges)
	}
	if got.GetExcludedContainsEdgeCount() != want.ExcludedContainsEdges {
		t.Errorf("excluded_contains_edge_count = %d, want %d", got.GetExcludedContainsEdgeCount(), want.ExcludedContainsEdges)
	}
	if int(got.GetCycleCount()) != want.CycleCount {
		t.Errorf("cycle_count = %d, want %d", got.GetCycleCount(), want.CycleCount)
	}

	if len(want.Nodes) == 0 {
		t.Fatal("test fixture assumption broken: eng.FileGraph's Nodes is empty")
	}
	if len(got.GetNodes()) != len(want.Nodes) {
		t.Fatalf("nodes has %d entries, want %d", len(got.GetNodes()), len(want.Nodes))
	}
	for i, wn := range want.Nodes {
		gn := got.GetNodes()[i]
		if gn.GetPath() != wn.Path {
			t.Errorf("nodes[%d].path = %q, want %q", i, gn.GetPath(), wn.Path)
		}
		if gn.GetLanguage() != wn.Language {
			t.Errorf("nodes[%d].language = %q, want %q", i, gn.GetLanguage(), wn.Language)
		}
		if gn.GetSymbolCount() != wn.SymbolCount {
			t.Errorf("nodes[%d].symbol_count = %d, want %d", i, gn.GetSymbolCount(), wn.SymbolCount)
		}
		if int(gn.GetCycleId()) != wn.CycleID {
			t.Errorf("nodes[%d].cycle_id = %d, want %d", i, gn.GetCycleId(), wn.CycleID)
		}
	}

	if len(want.Edges) == 0 {
		t.Fatal("test fixture assumption broken: eng.FileGraph's Edges is empty")
	}
	if len(got.GetEdges()) != len(want.Edges) {
		t.Fatalf("edges has %d entries, want %d", len(got.GetEdges()), len(want.Edges))
	}
	for i, we := range want.Edges {
		ge := got.GetEdges()[i]
		if ge.GetSourceFile() != we.SourceFile {
			t.Errorf("edges[%d].source_file = %q, want %q", i, ge.GetSourceFile(), we.SourceFile)
		}
		if ge.GetTargetFile() != we.TargetFile {
			t.Errorf("edges[%d].target_file = %q, want %q", i, ge.GetTargetFile(), we.TargetFile)
		}
		if ge.GetTotalCount() != we.TotalCount {
			t.Errorf("edges[%d].total_count = %d, want %d", i, ge.GetTotalCount(), we.TotalCount)
		}
		if ge.GetInCycle() != we.InCycle {
			t.Errorf("edges[%d].in_cycle = %v, want %v", i, ge.GetInCycle(), we.InCycle)
		}
		if len(ge.GetKindCounts()) != len(we.KindCounts) {
			t.Fatalf("edges[%d].kind_counts has %d entries, want %d", i, len(ge.GetKindCounts()), len(we.KindCounts))
		}
		for kind, count := range we.KindCounts {
			if ge.GetKindCounts()[kind] != count {
				t.Errorf("edges[%d].kind_counts[%q] = %d, want %d", i, kind, ge.GetKindCounts()[kind], count)
			}
		}
	}
}

// TestFileGraphKindCountsAreSparse proves the sparse-map convention
// (mirroring GetHealthResponse.edges_by_kind's, status.go): no returned
// edge's kind-count map contains a key whose value is 0, and at least one
// map is non-empty — the positive control that proves the loop below
// actually inspected something rather than passing vacuously over zero
// edges.
func TestFileGraphKindCountsAreSparse(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.FileGraph(context.Background(), connect.NewRequest(&uiv1.FileGraphRequest{}))
	if err != nil {
		t.Fatalf("FileGraph: %v", err)
	}

	if len(resp.Msg.GetEdges()) == 0 {
		t.Fatal("test fixture assumption broken: FileGraph returned 0 edges")
	}

	nonEmpty := 0
	for _, e := range resp.Msg.GetEdges() {
		if len(e.GetKindCounts()) > 0 {
			nonEmpty++
		}
		for kind, count := range e.GetKindCounts() {
			if count == 0 {
				t.Errorf("edge %s->%s: kind_counts[%q] = 0, want absent (sparse-map convention)", e.GetSourceFile(), e.GetTargetFile(), kind)
			}
		}
	}
	if nonEmpty == 0 {
		t.Fatal("no edge carried a non-empty kind_counts map — this guard is vacuous")
	}
}

// TestFileGraphOpensEngineExactlyOnce proves the withEngine open-use-close
// discipline (SRV-04): one FileGraph call invokes the package's
// openEngine seam exactly once, using the same counting-wrapper
// technique TestGetHealthOpensEngineExactlyOnce already establishes in
// health_test.go.
func TestFileGraphOpensEngineExactlyOnce(t *testing.T) {
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

	if _, err := client.FileGraph(context.Background(), connect.NewRequest(&uiv1.FileGraphRequest{})); err != nil {
		t.Fatalf("FileGraph: %v", err)
	}
	if got := atomic.LoadInt64(&opens); got != 1 {
		t.Fatalf("openEngine invoked %d times for one FileGraph call, want exactly 1", got)
	}
}

// TestFileGraphDoesNotDegrade proves FileGraph is NOT GetStatus: with the
// store held past graphstore.Open's bounded retry budget by a REAL
// graphstore.Open handle in this test process (the same technique
// TestGetHealthDoesNotDegrade uses to prove GetHealth's identical
// non-degrading behavior), FileGraph returns an ERROR through
// mapEngineError — never a successful empty response.
func TestFileGraphDoesNotDegrade(t *testing.T) {
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

	_, err = client.FileGraph(context.Background(), connect.NewRequest(&uiv1.FileGraphRequest{}))
	if err == nil {
		t.Fatal("FileGraph while the store is locked past the retry budget succeeded, want an error (FileGraph must not degrade-and-answer)")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnavailable {
		t.Fatalf("FileGraph while locked: code = %v, want CodeUnavailable", code)
	}
}

// TestFileGraphRequestPathIsIgnored proves T-05-09: a request carrying a
// non-empty path field returns the SAME response as an empty request —
// the field is declared for a future milestone and is not read, and no
// path from the request reaches the store (05-02 Task 1 checkpoint
// sub-decision, matching GetHealthRequest.path's existing precedent at
// internal/uiproto/uiv1/ui.proto:626).
func TestFileGraphRequestPathIsIgnored(t *testing.T) {
	dir := copyGofixture(t)
	indexGofixture(t, dir)

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	empty, err := client.FileGraph(context.Background(), connect.NewRequest(&uiv1.FileGraphRequest{}))
	if err != nil {
		t.Fatalf("FileGraph (empty path): %v", err)
	}
	populated, err := client.FileGraph(context.Background(), connect.NewRequest(&uiv1.FileGraphRequest{Path: "../../attacker/attacker-repo"}))
	if err != nil {
		t.Fatalf("FileGraph (populated path): %v", err)
	}

	if !proto.Equal(empty.Msg, populated.Msg) {
		t.Fatalf("FileGraph response differs between an empty path request and a populated one — the path field must not reach the store\nempty:      %+v\npopulated:  %+v", empty.Msg, populated.Msg)
	}
}

// TestFileGraphResponseSizeGuava is GRF-01's wire-size measurement
// (05-02 Task 3, corpora/graph-render-threshold.json's
// fileGraphResponseBytes bar): gated on CODEGRAPH_GUAVA_STORE naming an
// indexed guava checkout, it drives FileGraph through a REAL server and
// client — the same wire path a browser uses, not a direct in-process
// call — and marshals the returned FileGraphResponse to log the EXACT
// serialized byte count, replacing 05-RESEARCH.md's unverified 5-6 MB
// estimate with a measurement. Skipped by default (no test in this suite
// touches a multi-thousand-file external corpus) and only runs when
// CODEGRAPH_GUAVA_STORE is set.
func TestFileGraphResponseSizeGuava(t *testing.T) {
	dir := os.Getenv("CODEGRAPH_GUAVA_STORE")
	if dir == "" {
		t.Skip("CODEGRAPH_GUAVA_STORE is unset; set it to an indexed google/guava checkout root to run this measurement")
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("CODEGRAPH_GUAVA_STORE=%q does not exist: %v", dir, err)
	}

	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	resp, err := client.FileGraph(context.Background(), connect.NewRequest(&uiv1.FileGraphRequest{}))
	if err != nil {
		t.Fatalf("FileGraph against guava corpus: %v", err)
	}
	if len(resp.Msg.GetNodes()) == 0 || len(resp.Msg.GetEdges()) == 0 {
		t.Fatalf("test fixture assumption broken: guava's FileGraph returned %d nodes / %d edges, want both > 0", len(resp.Msg.GetNodes()), len(resp.Msg.GetEdges()))
	}

	b, err := proto.Marshal(resp.Msg)
	if err != nil {
		t.Fatalf("proto.Marshal(FileGraphResponse): %v", err)
	}

	t.Logf("guava FileGraphResponse: %d bytes serialized, %d nodes, %d edges, %d excluded package nodes, %d excluded self edges, %d excluded contains edges, %d cycles",
		len(b), len(resp.Msg.GetNodes()), len(resp.Msg.GetEdges()), resp.Msg.GetExcludedPackageNodeCount(), resp.Msg.GetExcludedSelfEdgeCount(), resp.Msg.GetExcludedContainsEdgeCount(), resp.Msg.GetCycleCount())

	// Upper bound alone is satisfied by 0 (rule 84d1gfpywd) — the floor
	// proves the measurement actually ran against a corpus of real size,
	// not a degenerate empty response.
	const floor = 1_000_000
	const ceiling = 16 * 1024 * 1024 // transportSendMaxBytes, internal/uiserver/truncate.go
	if len(b) <= floor {
		t.Fatalf("serialized FileGraphResponse = %d bytes, want > %d (floor: proves this measured a real corpus)", len(b), floor)
	}
	if len(b) >= ceiling {
		t.Fatalf("serialized FileGraphResponse = %d bytes, want < %d (transportSendMaxBytes ceiling) — GRF-01's fileGraphResponseBytes bar FAILED", len(b), ceiling)
	}
}
