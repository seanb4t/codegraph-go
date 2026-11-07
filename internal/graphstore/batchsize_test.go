package graphstore

import (
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestBatchSizeCoversStagedRecords stages the records of a full graph
// write and compares BatchSize with the bytes the batch holds. The sum
// for the records must cover the bytes they staged, so the batch never
// grows, and must exceed them by no more than the varint slack of each
// record. The fixed allowance must cover the rest of the batch.
func TestBatchSizeCoversStagedRecords(t *testing.T) {
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	text := rapid.StringN(0, 200, -1)
	rapid.Check(t, func(t *rapid.T) {
		w, err := store.NewWriterSized(0)
		if err != nil {
			t.Fatalf("NewWriterSized: %v", err)
		}
		defer w.Close()
		var size BatchSize
		puts := 0

		batch := w.(*pebbleWriter).batch
		if err := w.DeleteGraph(); err != nil {
			t.Fatalf("DeleteGraph: %v", err)
		}
		recordBytes := -batch.Len()
		for range rapid.IntRange(0, 20).Draw(t, "nodes") {
			n := &schema.Node{
				Id:       text.Draw(t, "id"),
				Name:     text.Draw(t, "name"),
				FilePath: rapid.SampledFrom([]string{"", text.Draw(t, "path")}).Draw(t, "filePath"),
			}
			size.Node(n)
			if err := w.PutNode(n); err != nil {
				t.Fatalf("PutNode: %v", err)
			}
			puts += 1 + boolInt(n.FilePath != "")
		}
		for range rapid.IntRange(0, 20).Draw(t, "files") {
			f := &schema.File{Path: text.Draw(t, "path"), ContentHash: text.Draw(t, "hash")}
			size.File(f)
			if err := w.PutFile(f); err != nil {
				t.Fatalf("PutFile: %v", err)
			}
			puts++
		}
		for range rapid.IntRange(0, 20).Draw(t, "edges") {
			e := &schema.Edge{Source: text.Draw(t, "source"), Kind: text.Draw(t, "kind"), Target: text.Draw(t, "target")}
			owner := rapid.SampledFrom([]string{"", text.Draw(t, "owner")}).Draw(t, "ownerPath")
			size.Edge(e, owner)
			if err := w.PutEdge(e, owner); err != nil {
				t.Fatalf("PutEdge: %v", err)
			}
			puts += 1 + boolInt(owner != "")
		}
		recordBytes += batch.Len()
		if err := w.DeleteAllExcludedFiles(); err != nil {
			t.Fatalf("DeleteAllExcludedFiles: %v", err)
		}
		recordBytes -= batch.Len()
		for range rapid.IntRange(0, 20).Draw(t, "excluded") {
			x := &schema.ExcludedFile{Path: text.Draw(t, "path"), Detail: text.Draw(t, "detail")}
			size.ExcludedFile(x)
			if err := w.PutExcludedFile(x); err != nil {
				t.Fatalf("PutExcludedFile: %v", err)
			}
			puts++
		}
		recordBytes += batch.Len()
		meta := schema.NewMeta()
		meta.CommitSha = "0123456789abcdef0123456789abcdef01234567"
		meta.NodeCount, meta.EdgeCount, meta.CoverageGeneration = 1<<40, 1<<40, 1<<40
		meta.HasFileIndex, meta.HasCoverage = true, true
		if err := w.PutMeta(meta); err != nil {
			t.Fatalf("PutMeta: %v", err)
		}

		const rangeDeletes, metaPuts = 5, 1
		if got, want := int(batch.Count()), puts+rangeDeletes+metaPuts; got != want {
			t.Fatalf("batch holds %d records, staged %d", got, want)
		}
		if size.n < recordBytes {
			t.Fatalf("estimate %d for the records is below the %d bytes they staged", size.n, recordBytes)
		}
		if limit := recordBytes + (setOverhead-3)*puts; size.n > limit {
			t.Fatalf("estimate %d for the records is above the limit %d for %d staged bytes", size.n, limit, recordBytes)
		}
		if rest := batch.Len() - recordBytes; rest > fixedAllowance {
			t.Fatalf("header, range deletes and meta take %d bytes, above the allowance %d", rest, fixedAllowance)
		}
		if got := size.Bytes(); got < batch.Len() {
			t.Fatalf("Bytes() = %d is below the batch length %d", got, batch.Len())
		}
	})
}

func TestBatchSizeIsCapped(t *testing.T) {
	b := BatchSize{n: maxSizeHint}
	if got := b.Bytes(); got != maxSizeHint {
		t.Fatalf("Bytes() = %d, want the cap %d", got, maxSizeHint)
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
