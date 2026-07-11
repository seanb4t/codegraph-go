package graphstore

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/cockroachdb/pebble/v2"
	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Record kind tags for the bulk export/import framing (ARCH-01). Each
// framed record is [kind byte][uvarint length][protobuf bytes] — the kind
// byte is what makes the stream self-describing without a separate
// side-channel schema, so an older/newer reader can always tell which
// message type to unmarshal into.
const (
	exportKindMeta uint8 = 1
	exportKindNode uint8 = 2
	exportKindEdge uint8 = 3
	exportKindFile uint8 = 4
)

// maxImportRecordBytes bounds the per-record length prefix read by Import,
// mirroring parser.MaxSourceBytes's role on the parse path: it stops an
// untrusted or corrupted stream's uvarint length prefix from driving an
// unbounded make([]byte, length) allocation (a DoS vector on the one code
// path in this phase explicitly designed to read externally-produced
// data).
const maxImportRecordBytes = 64 * 1024 * 1024 // 64 MiB

// Export streams every record in the store — the Meta record first (per
// D-04/ARCH-01: schema_version is always the first thing a reader of the
// stream sees), then every Node, Edge, and File — as self-describing,
// schema-versioned protobuf frames. The whole export is taken from a
// single Pebble snapshot, so a writer committing after Export has captured
// its snapshot cannot tear the in-flight stream (INDX-05).
func (s *pebbleStore) Export(w io.Writer) error {
	// WR-01: unlike Snapshot/NewWriter — which hand a long-lived Reader/
	// Writer back to the caller and only need their own creation call
	// guarded — Export is a single bounded, synchronous streaming
	// operation, and a captured *pebble.Snapshot does NOT remain safely
	// usable across a concurrent Close: pebble/v2's DB.Close() can leave
	// an already-open snapshot's own NewIter call panicking with "pebble:
	// closed" (confirmed via TestStoreConcurrentCloseNeverPanics — the
	// panic was NOT at snapshot creation, it was at a later NewIter call
	// on a snapshot captured while still open). So Export holds s.mu's
	// RLock for its ENTIRE duration, not just the initial check — Close
	// cannot proceed until any in-flight Export has finished, which is an
	// acceptable, bounded-duration cost for this one non-hot-path
	// operation (unlike Snapshot/NewWriter's per-call readers/writers,
	// whose lifetimes are caller-controlled and would make this same
	// approach block Close indefinitely).
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.closed.Load() {
		return ErrClosed
	}
	snap := s.db.NewSnapshot()
	defer snap.Close()

	bw := bufio.NewWriter(w)

	var meta schema.Meta
	err := getProto(snap, metaKey(metaRecordName), &meta)
	switch {
	case err == nil:
		if err := writeExportRecord(bw, exportKindMeta, &meta); err != nil {
			return fmt.Errorf("export meta: %w", err)
		}
	case err == ErrNotFound:
		// No meta record has ever been written to this store — nothing to
		// export for it; nodes/edges/files can still legitimately exist.
	default:
		return fmt.Errorf("export meta: %w", err)
	}

	if err := exportNamespace(bw, snap, prefixNode, exportKindNode, func() proto.Message { return &schema.Node{} }); err != nil {
		return fmt.Errorf("export nodes: %w", err)
	}
	if err := exportNamespace(bw, snap, prefixEdge, exportKindEdge, func() proto.Message { return &schema.Edge{} }); err != nil {
		return fmt.Errorf("export edges: %w", err)
	}
	if err := exportNamespace(bw, snap, prefixFile, exportKindFile, func() proto.Message { return &schema.File{} }); err != nil {
		return fmt.Errorf("export files: %w", err)
	}

	return bw.Flush()
}

// exportNamespace streams every record under the single-byte prefix ns,
// calling newMsg to allocate a fresh proto.Message per record so the
// unmarshal target's type matches ns's record kind.
func exportNamespace(bw *bufio.Writer, snap *pebble.Snapshot, ns byte, kind uint8, newMsg func() proto.Message) error {
	lower := []byte{ns}
	upper := rangeUpperBound(lower)
	iter, err := snap.NewIter(&pebble.IterOptions{LowerBound: lower, UpperBound: upper})
	if err != nil {
		return err
	}
	defer iter.Close()

	for ok := iter.First(); ok; ok = iter.Next() {
		msg := newMsg()
		if err := proto.Unmarshal(iter.Value(), msg); err != nil {
			return err
		}
		if err := writeExportRecord(bw, kind, msg); err != nil {
			return err
		}
	}
	return iter.Error()
}

// writeExportRecord frames one record as [kind][uvarint length][bytes].
func writeExportRecord(w io.Writer, kind uint8, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	header := make([]byte, 0, 1+binary.MaxVarintLen64)
	header = append(header, kind)
	header = binary.AppendUvarint(header, uint64(len(data)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// Import reconstructs a graph exported by Export into dst — typically a
// fresh, empty GraphStore. It applies every record through dst's batched
// Writer and commits once, so the whole import is atomic (D-04). This is
// the counterpart the ARCH-01 round-trip test drives: Export followed by
// Import into a fresh store must reproduce an identical graph.
//
// nodeFilePath tracks each imported node's id -> FilePath as node records
// stream past, so a later edge record can supply PutEdge's ownerPath
// (Phase 4 D-02) and rebuild the x/ file-index for the imported store —
// Export's own namespace ordering (nodes, then edges, then files) guarantees
// every edge's source node has already been seen by the time the edge
// arrives.
func Import(dst GraphStore, r io.Reader) error {
	w, err := dst.NewWriter()
	if err != nil {
		return err
	}

	nodeFilePath := make(map[string]string)
	br := bufio.NewReader(r)
	for {
		kind, err := br.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("import: read kind: %w", err)
		}
		length, err := binary.ReadUvarint(br)
		if err != nil {
			return fmt.Errorf("import: read length: %w", err)
		}
		if length > maxImportRecordBytes {
			return fmt.Errorf("import: record length %d exceeds ceiling %d", length, uint64(maxImportRecordBytes))
		}
		data := make([]byte, length)
		if _, err := io.ReadFull(br, data); err != nil {
			return fmt.Errorf("import: read record: %w", err)
		}

		if err := importRecord(w, kind, data, nodeFilePath); err != nil {
			return err
		}
	}

	return w.Commit()
}

func importRecord(w Writer, kind uint8, data []byte, nodeFilePath map[string]string) error {
	switch kind {
	case exportKindMeta:
		var m schema.Meta
		if err := proto.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("import: unmarshal meta: %w", err)
		}
		if err := w.PutMeta(&m); err != nil {
			return fmt.Errorf("import: put meta: %w", err)
		}
	case exportKindNode:
		var n schema.Node
		if err := proto.Unmarshal(data, &n); err != nil {
			return fmt.Errorf("import: unmarshal node: %w", err)
		}
		nodeFilePath[n.GetId()] = n.GetFilePath()
		if err := w.PutNode(&n); err != nil {
			return fmt.Errorf("import: put node: %w", err)
		}
	case exportKindEdge:
		var e schema.Edge
		if err := proto.Unmarshal(data, &e); err != nil {
			return fmt.Errorf("import: unmarshal edge: %w", err)
		}
		if err := w.PutEdge(&e, nodeFilePath[e.GetSource()]); err != nil {
			return fmt.Errorf("import: put edge: %w", err)
		}
	case exportKindFile:
		var f schema.File
		if err := proto.Unmarshal(data, &f); err != nil {
			return fmt.Errorf("import: unmarshal file: %w", err)
		}
		if err := w.PutFile(&f); err != nil {
			return fmt.Errorf("import: put file: %w", err)
		}
	default:
		return fmt.Errorf("import: unknown record kind %d", kind)
	}
	return nil
}
