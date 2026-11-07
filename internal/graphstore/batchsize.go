package graphstore

import (
	"encoding/binary"
	"math"

	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

const (
	// setOverhead is the most bytes a batch adds to one Set beyond its
	// key and value: a kind byte and two length varints. A batch grows
	// before a Set unless this much room is free, so an estimate that
	// counts it for every record never makes the batch grow.
	setOverhead = 1 + 2*binary.MaxVarintLen32

	// fixedAllowance covers the batch header, the range deletes of
	// DeleteGraph and DeleteAllExcludedFiles, and the meta record.
	fixedAllowance = 4096

	// maxSizeHint caps BatchSize.Bytes below the largest batch that
	// Pebble accepts.
	maxSizeHint = math.MaxInt32
)

// BatchSize sums an upper bound on the bytes that a Writer stages for a
// full graph write: DeleteGraph, DeleteAllExcludedFiles, PutMeta and one
// Put for each record added here. Each method mirrors the Writer method
// of the same record kind and must change with its key shape.
type BatchSize struct {
	n int
}

// Node adds the records of PutNode(n).
func (b *BatchSize) Node(n *schema.Node) {
	id := segmentLen(n.GetId())
	b.n += setOverhead + 1 + id + proto.Size(n)
	if p := n.GetFilePath(); p != "" {
		b.n += setOverhead + 1 + segmentLen(p) + 1 + id
	}
}

// Edge adds the records of PutEdge(e, ownerPath).
func (b *BatchSize) Edge(e *schema.Edge, ownerPath string) {
	triple := segmentLen(e.GetSource()) + segmentLen(e.GetKind()) + segmentLen(e.GetTarget())
	b.n += setOverhead + 1 + triple + proto.Size(e)
	if ownerPath != "" {
		b.n += setOverhead + 1 + segmentLen(ownerPath) + 1 + triple
	}
}

// File adds the record of PutFile(f).
func (b *BatchSize) File(f *schema.File) {
	b.n += setOverhead + 1 + segmentLen(f.GetPath()) + proto.Size(f)
}

// ExcludedFile adds the record of PutExcludedFile(x).
func (b *BatchSize) ExcludedFile(x *schema.ExcludedFile) {
	b.n += setOverhead + 1 + segmentLen(x.GetPath()) + proto.Size(x)
}

// Bytes returns the size to pass to GraphStore.NewWriterSized.
func (b *BatchSize) Bytes() int {
	return min(b.n+fixedAllowance, maxSizeHint)
}

// segmentLen is the length of appendSegment's encoding of seg.
func segmentLen(seg string) int {
	var buf [binary.MaxVarintLen64]byte
	return binary.PutUvarint(buf[:], uint64(len(seg))) + len(seg)
}
