package graphstore

import (
	"bufio"
	"fmt"
	"io"

	"github.com/cockroachdb/pebble/v2"
	"google.golang.org/protobuf/proto"

	"github.com/seanb4t/codegraph-go/internal/recordio"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Record kind tags for the bulk export/import framing (ARCH-01). Each
// framed record is [kind byte][uvarint length][protobuf bytes] — the kind
// byte is what makes the stream self-describing without a separate
// side-channel schema, so an older/newer reader can always tell which
// message type to unmarshal into. The framing itself lives in
// internal/recordio. The tags are exported so that a reader of an export
// stream in another package names them instead of copying their values.
const (
	ExportKindMeta uint8 = 1
	ExportKindNode uint8 = 2
	ExportKindEdge uint8 = 3
	ExportKindFile uint8 = 4

	// ExportKindExcludedFile is Phase 10's fifth record kind (the c/
	// namespace, D-05). Unlike x/ (the file-owned secondary index,
	// which every PutNode/PutEdge regenerates on import), nothing else
	// regenerates a c/ record — it MUST be framed explicitly here
	// (RESEARCH Pitfall 2), or an Export/Import round trip would
	// silently lose every exclusion record.
	ExportKindExcludedFile uint8 = 5
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
	switch err {
	case nil:
		if err := writeExportRecord(bw, ExportKindMeta, &meta); err != nil {
			return fmt.Errorf("export meta: %w", err)
		}
	case ErrNotFound:
		// No meta record has ever been written to this store — nothing to
		// export for it; nodes/edges/files can still legitimately exist.
	default:
		return fmt.Errorf("export meta: %w", err)
	}

	if err := exportNamespace(bw, snap, prefixNode, ExportKindNode, func() proto.Message { return &schema.Node{} }); err != nil {
		return fmt.Errorf("export nodes: %w", err)
	}
	if err := exportNamespace(bw, snap, prefixEdge, ExportKindEdge, func() proto.Message { return &schema.Edge{} }); err != nil {
		return fmt.Errorf("export edges: %w", err)
	}
	if err := exportNamespace(bw, snap, prefixFile, ExportKindFile, func() proto.Message { return &schema.File{} }); err != nil {
		return fmt.Errorf("export files: %w", err)
	}
	if err := exportNamespace(bw, snap, prefixExcludedFile, ExportKindExcludedFile, func() proto.Message { return &schema.ExcludedFile{} }); err != nil {
		return fmt.Errorf("export excluded files: %w", err)
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
// Uses recordio.Marshal, not raw proto.Marshal, for the SAME reason PutX
// does: this re-marshals a message freshly unmarshaled from the store
// (exportNamespace, above), and a map-valued field (schema.Edge.Metadata)
// would otherwise re-randomize its key order on every Export() call
// regardless of what was originally persisted — exactly the
// byte-identical-rebuild guarantee determinism_test.go's
// TestDeterministicRebuild (and Phase 5's route-specific
// TestRoute_DeterministicRebuild) verify against this stream.
func writeExportRecord(w io.Writer, kind uint8, msg proto.Message) error {
	data, err := recordio.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = recordio.WriteFrame(w, kind, data)
	return err
}

// Import reconstructs a graph exported by Export into dst — typically a
// fresh, empty GraphStore. Export followed by Import into a fresh store
// reproduces an identical graph (ARCH-01).
//
// Import writes through a bounded Writer with DefaultBatchCap, so it
// commits in batches and no batch holds the whole graph. The store is not
// complete until Import returns nil: a reader of dst before that sees a
// part of the graph. An export holds its Meta frame first, and Import
// keeps that frame aside and writes it as the last record. So the
// metadata is in the last batch, which is the one durable commit, and a
// store without metadata is one that Import did not finish.
//
// nodeFilePath tracks each imported node's id -> FilePath as node records
// stream past, so a later edge record can supply PutEdge's ownerPath
// (Phase 4 D-02) and rebuild the x/ file-index for the imported store —
// Export's own namespace ordering (nodes, then edges, then files) guarantees
// every edge's source node has already been seen by the time the edge
// arrives.
func Import(dst GraphStore, r io.Reader) error {
	return importWith(dst, r, DefaultBatchCap, nil)
}

// importWith is Import with the batch cap and the commit callback of the
// bounded Writer as parameters, so a test can pass a low cap and watch the
// commits.
func importWith(dst GraphStore, r io.Reader, cap int, onCommit func(Commit) error) error {
	w, err := dst.NewBoundedWriter(cap, onCommit)
	if err != nil {
		return err
	}
	defer w.Close()

	nodeFilePath := make(map[string]string)
	var meta []byte
	fr := recordio.NewReader(r, maxImportRecordBytes)
	for {
		kind, data, err := fr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("import: %w", err)
		}
		if kind == ExportKindMeta {
			meta = data
			continue
		}
		if err := importRecord(w, kind, data, nodeFilePath); err != nil {
			return err
		}
	}
	if meta != nil {
		if err := importRecord(w, ExportKindMeta, meta, nodeFilePath); err != nil {
			return err
		}
	}
	return w.Commit()
}

func importRecord(w Writer, kind uint8, data []byte, nodeFilePath map[string]string) error {
	switch kind {
	case ExportKindMeta:
		var m schema.Meta
		if err := proto.Unmarshal(data, &m); err != nil {
			return fmt.Errorf("import: unmarshal meta: %w", err)
		}
		if err := w.PutMeta(&m); err != nil {
			return fmt.Errorf("import: put meta: %w", err)
		}
	case ExportKindNode:
		var n schema.Node
		if err := proto.Unmarshal(data, &n); err != nil {
			return fmt.Errorf("import: unmarshal node: %w", err)
		}
		nodeFilePath[n.GetId()] = n.GetFilePath()
		if err := w.PutNode(&n); err != nil {
			return fmt.Errorf("import: put node: %w", err)
		}
	case ExportKindEdge:
		var e schema.Edge
		if err := proto.Unmarshal(data, &e); err != nil {
			return fmt.Errorf("import: unmarshal edge: %w", err)
		}
		if err := w.PutEdge(&e, nodeFilePath[e.GetSource()]); err != nil {
			return fmt.Errorf("import: put edge: %w", err)
		}
	case ExportKindFile:
		var f schema.File
		if err := proto.Unmarshal(data, &f); err != nil {
			return fmt.Errorf("import: unmarshal file: %w", err)
		}
		if err := w.PutFile(&f); err != nil {
			return fmt.Errorf("import: put file: %w", err)
		}
	case ExportKindExcludedFile:
		var x schema.ExcludedFile
		if err := proto.Unmarshal(data, &x); err != nil {
			return fmt.Errorf("import: unmarshal excluded file: %w", err)
		}
		if err := w.PutExcludedFile(&x); err != nil {
			return fmt.Errorf("import: put excluded file: %w", err)
		}
	default:
		return fmt.Errorf("import: unknown record kind %d", kind)
	}
	return nil
}
