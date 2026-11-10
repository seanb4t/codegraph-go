package indexer

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// updateLinkGoldens is the only way to write a golden file under
// testdata/link. A golden is never edited by hand.
// Usage: go test ./internal/indexer/ -run 'TestLinkUnitsMatchGolden$' -update-link-goldens -count=1
var updateLinkGoldens = flag.Bool("update-link-goldens", false, "rewrite internal/indexer/testdata/link/*.golden from what a full index stores")

// linkGoldenFixtures are the trees of TestLinkUnitsMatchGolden: each
// fixture with several files, and the Express fixture for the route unit.
var linkGoldenFixtures = []struct{ name, root string }{
	{"censusfixture", "testdata/censusfixture"},
	{"coverage", "testdata/coverage"},
	{"gofixture", "testdata/gofixture"},
	{"mixedlangfixture", "testdata/mixedlangfixture"},
	{"overloadfixture", "testdata/overloadfixture"},
	{"partialbasefixture", "testdata/partialbasefixture"},
	{"prunefixture", "testdata/prunefixture"},
	{"repeatfixture", "testdata/repeatfixture"},
	{"routes-express", "testdata/routesfixture/express"},
	{"splitmethodfixture", "testdata/splitmethodfixture"},
}

// jsonLine renders msg as one line of JSON. protojson does not promise
// stable whitespace, so the line goes through json.Compact.
func jsonLine(t *testing.T, msg proto.Message) string {
	t.Helper()
	raw, err := protojson.Marshal(msg)
	if err != nil {
		t.Fatalf("protojson.Marshal: %v", err)
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		t.Fatalf("json.Compact: %v", err)
	}
	return buf.String()
}

// linkProjection indexes root into a new store and returns what the test
// compares: the edge records and the file records in key order, then the
// file-index entries of each file in path order. The file-index entries
// prove the unit of each edge, because the owner path of an edge decides
// its entry. Node records stay out, because a merged node changes them on
// purpose.
func linkProjection(t *testing.T, root string) string {
	t.Helper()
	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	defer store.Close()
	var export bytes.Buffer
	if err := store.Export(&export); err != nil {
		t.Fatalf("Export: %v", err)
	}

	var out strings.Builder
	var paths []string
	for _, f := range decodeExportFrames(t, export.Bytes()) {
		switch f.Kind {
		case graphstore.ExportKindEdge:
			var e schema.Edge
			if err := proto.Unmarshal(f.Payload, &e); err != nil {
				t.Fatalf("unmarshal edge: %v", err)
			}
			fmt.Fprintf(&out, "edge %s\n", jsonLine(t, &e))
		case graphstore.ExportKindFile:
			var file schema.File
			if err := proto.Unmarshal(f.Payload, &file); err != nil {
				t.Fatalf("unmarshal file: %v", err)
			}
			// The modification time is that of the checkout, not of the graph.
			file.MtimeUnixNs = 0
			fmt.Fprintf(&out, "file %s\n", jsonLine(t, &file))
			paths = append(paths, file.Path)
		}
	}

	r, err := store.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	defer r.Close()
	sort.Strings(paths)
	for _, path := range paths {
		it, err := r.IterateFileIndex(path)
		if err != nil {
			t.Fatalf("IterateFileIndex(%s): %v", path, err)
		}
		for it.Next() {
			if e := it.Entry(); e.IsNode {
				fmt.Fprintf(&out, "index %s node %s\n", path, e.NodeID)
			} else {
				fmt.Fprintf(&out, "index %s edge %s %s %s\n", path, e.Source, e.Kind, e.Target)
			}
		}
		err = it.Err()
		it.Close()
		if err != nil {
			t.Fatalf("file index of %s: %v", path, err)
		}
	}
	return out.String()
}

// TestLinkUnitsMatchGolden compares the edges, the file records and the
// file-index entries of a full index over each fixture with a golden file.
// The goldens were recorded before the resolver split into units, so the
// test holds the pipeline to the graph of the single global pass.
func TestLinkUnitsMatchGolden(t *testing.T) {
	for _, fx := range linkGoldenFixtures {
		t.Run(fx.name, func(t *testing.T) {
			got := linkProjection(t, fx.root)
			golden := filepath.Join("testdata", "link", fx.name+".golden")
			if *updateLinkGoldens {
				if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
					t.Fatalf("write %s: %v", golden, err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read %s: %v (run with -update-link-goldens to record it)", golden, err)
			}
			if got != string(want) {
				t.Errorf("%s differs from what a full index stores:\n%s", golden, lineDiff(string(want), got))
			}
		})
	}
}

// lineDiff lists the lines that only one of the two texts holds.
func lineDiff(want, got string) string {
	count := make(map[string]int)
	for _, l := range strings.Split(want, "\n") {
		count[l]++
	}
	var out []string
	for _, l := range strings.Split(got, "\n") {
		if count[l] > 0 {
			count[l]--
			continue
		}
		out = append(out, "+ "+l)
	}
	for l, n := range count {
		for ; n > 0; n-- {
			out = append(out, "- "+l)
		}
	}
	sort.Strings(out)
	if len(out) > 60 {
		out = append(out[:60], fmt.Sprintf("... and %d more lines", len(out)-60))
	}
	return strings.Join(out, "\n")
}
