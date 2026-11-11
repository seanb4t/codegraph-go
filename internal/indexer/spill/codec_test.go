package spill

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

func sampleRecord(i int) record {
	return record{
		Result: goextract.FileResult{
			RelPath:     "pkg/file.go",
			Imports:     map[string]string{"fmt": "fmt"},
			MethodArity: map[string]int32{"m": int32(i)},
			SizeBytes:   int64(i),
		},
		Nodes: []wireNode{{Proto: []byte{0}, Rest: goextract.ExtractedNode{}}},
		Edges: []wireEdge{{Proto: []byte{0}}},
	}
}

// TestStreamEncodersAgree checks what the record format depends on: every
// encoder writes the same type descriptions, so one decoder reads the
// records of all of them, in any order.
func TestStreamEncodersAgree(t *testing.T) {
	const n = 4
	var preludes, records [][]byte
	for i := range n {
		e, prelude, err := newStreamEncoder()
		if err != nil {
			t.Fatalf("newStreamEncoder: %v", err)
		}
		preludes = append(preludes, prelude)
		// Encoder i wrote i records before the one that the test reads, so
		// the encoders are at different points of their streams.
		for j := range i + 1 {
			e.buf.Reset()
			rec := sampleRecord(i*10 + j)
			if err := e.enc.Encode(&rec); err != nil {
				t.Fatalf("Encode: %v", err)
			}
		}
		records = append(records, bytes.Clone(e.buf.Bytes()))
	}
	if len(preludes[0]) == 0 {
		t.Fatal("the prelude is empty, so it holds no type description")
	}
	for i := 1; i < n; i++ {
		if !bytes.Equal(preludes[i], preludes[0]) {
			t.Fatalf("encoder %d wrote a prelude that differs from that of encoder 0", i)
		}
	}

	d, err := newStreamDecoder()
	if err != nil {
		t.Fatalf("newStreamDecoder: %v", err)
	}
	for _, i := range []int{2, 0, 3, 1, 2} {
		var got record
		d.src.Reset(records[i])
		if err := d.dec.Decode(&got); err != nil {
			t.Fatalf("decode the record of encoder %d: %v", i, err)
		}
		if want := sampleRecord(i*10 + i); !reflect.DeepEqual(got, want) {
			t.Fatalf("record of encoder %d: got %+v, want %+v", i, got, want)
		}
	}
}

// TestRecordHoldsNoTypeDescription checks that a record does not repeat
// the type descriptions: the record of a file with no symbol is shorter
// than they are.
func TestRecordHoldsNoTypeDescription(t *testing.T) {
	prelude, err := streamPrelude()
	if err != nil {
		t.Fatalf("streamPrelude: %v", err)
	}
	data, err := Encode(&goextract.FileResult{RelPath: "pkg/file.go"})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(data) == 0 || len(data) >= len(prelude)/4 {
		t.Fatalf("the record of an empty file has %d bytes and the type descriptions %d; want a record far below them", len(data), len(prelude))
	}
}

// TestDecodeAfterFailure checks that a failed Decode does not break the
// ones after it.
func TestDecodeAfterFailure(t *testing.T) {
	data, err := Encode(&goextract.FileResult{RelPath: "pkg/file.go"})
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	for _, bad := range [][]byte{nil, {0xFF, 0xFF, 0xFF}, data[:len(data)-1], append(bytes.Clone(data), data...)} {
		if _, err := Decode(bad); err == nil {
			t.Fatalf("Decode of %d damaged bytes gave no error", len(bad))
		}
		r, err := Decode(data)
		if err != nil || r.RelPath != "pkg/file.go" {
			t.Fatalf("Decode after a failure: %q, %v", r.RelPath, err)
		}
	}
}
