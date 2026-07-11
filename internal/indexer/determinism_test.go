package indexer

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"runtime"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Export frame-kind tags, mirroring graphstore/export.go's own (unexported)
// exportKindMeta/Node/Edge/File constants — the documented wire contract is
// [kind byte][uvarint length][protobuf bytes], kind 1=Meta 2=Node 3=Edge
// 4=File. Duplicated here (rather than exported from graphstore) because
// this test only needs to decode far enough to normalize one volatile
// field and enumerate node kinds; that doesn't warrant a new cross-package
// dependency on graphstore's internal framing constants.
const (
	frameKindMeta uint8 = 1
	frameKindNode uint8 = 2
)

// exportFrame is one decoded [kind][data] record from an Export() stream.
type exportFrame struct {
	kind uint8
	data []byte
}

// decodeExportFrames parses raw Export() output into its constituent
// frames without interpreting anything beyond the kind tag.
func decodeExportFrames(t *testing.T, raw []byte) []exportFrame {
	t.Helper()
	var frames []exportFrame
	br := bufio.NewReader(bytes.NewReader(raw))
	for {
		kind, err := br.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("decodeExportFrames: read kind: %v", err)
		}
		length, err := binary.ReadUvarint(br)
		if err != nil {
			t.Fatalf("decodeExportFrames: read length: %v", err)
		}
		data := make([]byte, length)
		if _, err := io.ReadFull(br, data); err != nil {
			t.Fatalf("decodeExportFrames: read data: %v", err)
		}
		frames = append(frames, exportFrame{kind: kind, data: data})
	}
	return frames
}

// encodeExportFrames re-serializes frames back into the same
// [kind][uvarint length][data] wire format Export() itself uses.
func encodeExportFrames(frames []exportFrame) []byte {
	var buf bytes.Buffer
	var lenBuf [binary.MaxVarintLen64]byte
	for _, f := range frames {
		buf.WriteByte(f.kind)
		n := binary.PutUvarint(lenBuf[:], uint64(len(f.data)))
		buf.Write(lenBuf[:n])
		buf.Write(f.data)
	}
	return buf.Bytes()
}

// indexAndExport runs the full from-scratch pipeline against repoRoot into
// a fresh temp store, exports it, and returns the stream with
// Meta.last_sync_unix_ms normalized to zero in the leading Meta frame — the
// one genuinely volatile field a from-scratch run may stamp (RESEARCH
// §Pitfall 2). The determinism diff always targets this Export() output,
// never raw Pebble .sst/.log files — LSM internals aren't byte-stable
// across independently-built stores (RESEARCH §Pitfall 2).
func indexAndExport(t *testing.T, repoRoot string) []byte {
	t.Helper()

	storeDir := t.TempDir()
	if _, err := Run(repoRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	var buf bytes.Buffer
	if err := store.Export(&buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	frames := decodeExportFrames(t, buf.Bytes())
	for i, f := range frames {
		if f.kind != frameKindMeta {
			continue
		}
		var m schema.Meta
		if err := proto.Unmarshal(f.data, &m); err != nil {
			t.Fatalf("unmarshal meta frame: %v", err)
		}
		m.LastSyncUnixMs = 0
		normalized, err := proto.Marshal(&m)
		if err != nil {
			t.Fatalf("marshal normalized meta frame: %v", err)
		}
		frames[i].data = normalized
	}

	return encodeExportFrames(frames)
}

// TestDeterministicRebuild proves indexing the shared multi-package
// fixture twice from scratch, into two entirely separate temp stores,
// yields byte-identical Export() streams after normalizing the one
// volatile field (INDX-02, D-01a). GOMAXPROCS is forced high so any
// residual goroutine/map-order nondeterminism in Pass 1/2 fails loudly
// here (RESEARCH warning sign) rather than shipping; run this test with
// -race to also catch data races on the same property.
func TestDeterministicRebuild(t *testing.T) {
	prev := runtime.GOMAXPROCS(8)
	t.Cleanup(func() { runtime.GOMAXPROCS(prev) })

	first := indexAndExport(t, fixtureRoot)
	second := indexAndExport(t, fixtureRoot)

	if !bytes.Equal(first, second) {
		reportFirstDiff(t, first, second)
	}
}

// reportFirstDiff decodes both frame streams and fails on the first
// differing record, so a determinism regression is debuggable rather than
// a bare "bytes.Equal was false".
func reportFirstDiff(t *testing.T, first, second []byte) {
	t.Helper()
	ff := decodeExportFrames(t, first)
	sf := decodeExportFrames(t, second)

	if len(ff) != len(sf) {
		t.Fatalf("determinism gate: frame count differs: first=%d second=%d", len(ff), len(sf))
	}
	for i := range ff {
		if ff[i].kind != sf[i].kind || !bytes.Equal(ff[i].data, sf[i].data) {
			t.Fatalf("determinism gate: frame %d differs: kind first=%d second=%d\nfirst data:  %x\nsecond data: %x",
				i, ff[i].kind, sf[i].kind, ff[i].data, sf[i].data)
		}
	}
	t.Fatalf("determinism gate: byte streams differ but no single differing frame was found (unexpected)")
}

// expectedNodeKinds is the full node-kind taxonomy this pipeline can
// produce: every kind goextract emits (D-06) plus the synthetic "package"
// pseudo-node Pass 2 mints for intra-module imports (RQ-1) — the automated
// stand-in for a real-repo structural spot-check (success-criterion-4).
var expectedNodeKinds = []string{
	goextract.KindFile,
	goextract.KindFunction,
	goextract.KindMethod,
	goextract.KindStruct,
	goextract.KindInterface,
	goextract.KindTypeAlias,
	goextract.KindConstant,
	goextract.KindVariable,
	kindPackage,
}

// TestRealRepoStructure indexes the shared multi-package fixture once and
// asserts the committed graph contains every expected node kind plus the
// four named cross-file edges RES-01/LANG-01 depend on — proving
// cross-package call resolution, struct/interface embedding, and the
// intra-module import pseudo-node all landed correctly (RES-01, LANG-01).
func TestRealRepoStructure(t *testing.T) {
	storeDir := t.TempDir()
	if _, err := Run(fixtureRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	var buf bytes.Buffer
	if err := store.Export(&buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	seenKinds := make(map[string]struct{})
	for _, f := range decodeExportFrames(t, buf.Bytes()) {
		if f.kind != frameKindNode {
			continue
		}
		var n schema.Node
		if err := proto.Unmarshal(f.data, &n); err != nil {
			t.Fatalf("unmarshal node frame: %v", err)
		}
		seenKinds[n.Kind] = struct{}{}
	}
	for _, kind := range expectedNodeKinds {
		if _, ok := seenKinds[kind]; !ok {
			t.Errorf("expected node kind %q present in the graph, got kinds %v", kind, seenKinds)
		}
	}

	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer r.Close()

	runID := nodeid.NodeID(goextract.KindFunction, "Run", "pkgb/pkgb.go")
	alphaID := nodeid.NodeID(goextract.KindFunction, "Alpha", "pkga/pkga.go")
	derivedID := nodeid.NodeID(goextract.KindStruct, "Derived", "pkga/embed.go")
	baseID := nodeid.NodeID(goextract.KindStruct, "Base", "pkga/embed.go")
	rwID := nodeid.NodeID(goextract.KindInterface, "ReadWriter", "pkga/embed.go")
	readerID := nodeid.NodeID(goextract.KindInterface, "Reader", "pkga/embed.go")
	pkgbFileID := nodeid.NodeID(goextract.KindFile, "pkgb/pkgb.go", "pkgb/pkgb.go")
	importPath := "example.com/gofixture/pkga"
	pkgID := nodeid.NodeID(kindPackage, importPath, importPath)

	for _, want := range []struct{ src, kind, dst, label string }{
		{runID, "calls", alphaID, "pkgb.Run -calls-> pkga.Alpha"},
		{derivedID, "embeds", baseID, "Derived -embeds-> Base"},
		{rwID, "embeds", readerID, "ReadWriter -embeds-> Reader"},
		{pkgbFileID, "imports", pkgID, "pkgb/pkgb.go -imports-> package example.com/gofixture/pkga"},
	} {
		if !hasEdge(t, r, want.src, want.kind, want.dst) {
			t.Errorf("expected cross-file edge %s (%s -%s-> %s), not found", want.label, want.src, want.kind, want.dst)
		}
	}
}

// hasEdge reports whether r has an edge from src matching (kind, dst).
func hasEdge(t *testing.T, r graphstore.Reader, src, kind, dst string) bool {
	t.Helper()
	it, err := r.IterateEdges(src)
	if err != nil {
		t.Fatalf("IterateEdges(%s): %v", src, err)
	}
	defer it.Close()
	for it.Next() {
		if it.Edge().Kind == kind && it.Edge().Target == dst {
			return true
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("IterateEdges(%s) error: %v", src, err)
	}
	return false
}
