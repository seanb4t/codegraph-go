// Package storetest holds helpers for tests that compare stores. Only
// tests import it.
package storetest

import (
	"bytes"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/recordio"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// ExportNormalized opens the store at storeDir, exports it, and returns
// the stream with Meta.last_sync_unix_ms set to zero in the Meta frame: the
// one field that a run stamps from the clock (INDX-02). It decodes and
// encodes the frames through recordio, the framing of the export itself.
//
// With clearGeneration, it sets Meta.coverage_generation to zero too. That
// field counts the commits of one store, not the content of the graph, so
// two stores with one graph and different histories differ in it (WR-01).
// A test that compares such stores clears it. A test that compares one
// store with itself keeps it.
func ExportNormalized(t testing.TB, storeDir string, clearGeneration bool) []byte {
	t.Helper()

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()

	var buf bytes.Buffer
	if err := store.Export(&buf); err != nil {
		t.Fatalf("Export: %v", err)
	}

	frames, err := recordio.ReadAll(&buf, 0)
	if err != nil {
		t.Fatalf("read export frames: %v", err)
	}
	for i, f := range frames {
		if f.Kind != graphstore.ExportKindMeta {
			continue
		}
		var m schema.Meta
		if err := proto.Unmarshal(f.Payload, &m); err != nil {
			t.Fatalf("unmarshal meta frame: %v", err)
		}
		m.LastSyncUnixMs = 0
		if clearGeneration {
			m.CoverageGeneration = 0
		}
		normalized, err := proto.Marshal(&m)
		if err != nil {
			t.Fatalf("marshal normalized meta frame: %v", err)
		}
		frames[i].Payload = normalized
	}
	return recordio.AppendFrames(nil, frames)
}
