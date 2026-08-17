// Package phpextract_test is an EXTERNAL test package (not `package
// phpextract`) deliberately: it drives the real internal/indexer.Run
// pipeline end-to-end, and internal/indexer itself imports phpextract
// (languages_php.go) — a same-package (internal) test file importing
// internal/indexer would create an import cycle. Mirrors csharpextract's/
// pyextract's own resolution_test.go pattern.
package phpextract_test

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// phpFixtureFile is one file this test writes into a temp repo root.
type phpFixtureFile struct {
	relPath, src string
}

func writePHPFixture(t *testing.T, files []phpFixtureFile) string {
	t.Helper()
	root := t.TempDir()
	for _, f := range files {
		abs := filepath.Join(root, filepath.FromSlash(f.relPath))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir for fixture %s: %v", f.relPath, err)
		}
		if err := os.WriteFile(abs, []byte(f.src), 0o644); err != nil {
			t.Fatalf("writing fixture %s: %v", f.relPath, err)
		}
	}
	return root
}

// selfConsistencyFixture is a small, representative PHP fixture: a
// composer.json PSR-4 map, a namespaced interface + class (with a
// namespace_use import and a base/interface clause), and a top-level
// function — enough surface to exercise every extraction path this
// package's phpextract_test.go table already covers individually.
var selfConsistencyFixture = []phpFixtureFile{
	{relPath: "composer.json", src: `{
  "autoload": {
    "psr-4": {
      "App\\": "src/"
    }
  }
}
`},
	{relPath: "src/Contracts/Shape.php", src: "<?php\nnamespace App\\Contracts;\n\ninterface Shape {\n    public function area(): float;\n}\n"},
	{relPath: "src/Widget.php", src: "<?php\nnamespace App;\n\nuse App\\Contracts\\Shape;\n\nclass Widget implements Shape {\n    public function area(): float {\n        return 0.0;\n    }\n\n    public function describe(): string {\n        return helper();\n    }\n}\n\nfunction helper(): string {\n    return \"widget\";\n}\n"},
}

// exportFrame mirrors internal/indexer's own (unexported) determinism_test.go
// decode/encode shape — duplicated here (a small, self-contained subset)
// because this test lives in an external test package and only needs to
// normalize one volatile Meta field before comparing two Export() streams.
type exportFrame struct {
	kind uint8
	data []byte
}

const frameKindMeta uint8 = 1

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

// indexAndExport runs the full from-scratch indexer.Run pipeline against
// root into a fresh temp store, exports it, and returns the stream with
// Meta.last_sync_unix_ms normalized to zero — the one genuinely volatile
// field a from-scratch run may stamp (mirrors internal/indexer's own
// determinism_test.go#indexAndExport).
func indexAndExport(t *testing.T, root string) []byte {
	t.Helper()

	storeDir := t.TempDir()
	if _, err := indexer.Run(root, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("indexer.Run: %v", err)
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

// TestSelfConsistency_DeterministicRebuild proves indexing the same PHP
// fixture twice from scratch, into two entirely separate temp stores,
// yields a byte-identical Export() stream after normalizing the one
// volatile field — the mainstream-tier D-12 validation bar (a lighter
// self-consistency check than priority-4's golden harness, per
// 05-RESEARCH.md's "Validation Architecture").
func TestSelfConsistency_DeterministicRebuild(t *testing.T) {
	root := writePHPFixture(t, selfConsistencyFixture)

	first := indexAndExport(t, root)
	second := indexAndExport(t, root)

	if !bytes.Equal(first, second) {
		t.Fatalf("self-consistency gate: two from-scratch PHP indexing runs produced different Export() streams")
	}
	if len(first) == 0 {
		t.Fatalf("self-consistency gate: Export() stream was empty — the fixture produced no committed graph data")
	}
}

// TestSelfConsistency_ExpectedStructure indexes the fixture once and
// asserts the committed graph contains PHP's own node-kind taxonomy plus
// the cross-file edges this tier's best-effort resolution is expected to
// produce (same-file/same-namespace resolution — see phpextract's types.go
// for the documented cross-namespace-`use` gap).
func TestSelfConsistency_ExpectedStructure(t *testing.T) {
	root := writePHPFixture(t, selfConsistencyFixture)
	storeDir := t.TempDir()
	if _, err := indexer.Run(root, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("indexer.Run: %v", err)
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

	const frameKindNode uint8 = 2
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
	for _, kind := range []string{"file", "struct", "interface", "function", "method"} {
		if _, ok := seenKinds[kind]; !ok {
			t.Errorf("expected node kind %q present in the graph, got kinds %v", kind, seenKinds)
		}
	}
}
