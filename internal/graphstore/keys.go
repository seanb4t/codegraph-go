package graphstore

import (
	"encoding/binary"
	"fmt"
)

// Key namespace prefixes (D-03). Every stored record's key begins with
// exactly one of these single bytes, so a whole namespace — or, for
// prefixFile, a single file's own record — is addressable by prefix scan or
// prefix range-delete. No code outside this file may construct a raw
// []byte key; every read/write path (Plan 01-06's pebbleStore) goes through
// the builders below.
const (
	prefixMeta byte = 'm' // meta/... — store-wide metadata (schema version, etc.)
	prefixNode byte = 'n' // n/<node-id>
	prefixEdge byte = 'e' // e/<src>/<kind>/<dst>, ordered by src for range scans

	prefixFile byte = 'f' // f/<path> — file record + content hash

	// prefixAnnotation is RESERVED for post-v1 embeddings/community
	// assignments (ARCH-01 physical slot). No v1 code writes under this
	// prefix; it exists so those future record kinds get their own
	// namespace without touching meta/n/e/f layout or requiring a schema
	// migration.
	prefixAnnotation byte = 'a'

	// prefixFileIndex is the file-owned secondary index (Phase 4 D-02):
	// x/<path>/<marker>/<node-id | src/kind/dst> maps a file path to the
	// node ids and outgoing edge triples it owns, so DeleteFileSubgraph
	// can prune a file's scattered, content-hash-keyed n/ and e/ records
	// in O(subgraph) time instead of a full-graph scan. Additive
	// namespace — SchemaVersion stays 1 (internal/schema/meta.go's
	// additive-only-no-bump rule; a new key namespace is not a
	// record-format break).
	prefixFileIndex byte = 'x'
)

// fileIndexKindNode and fileIndexKindEdge are fixed, code-controlled marker
// bytes (never attacker/caller data) that distinguish a file-index entry's
// two sub-ranges — 0x01 sorts before 0x02, so a file's node entries and
// edge entries each form their own contiguous sub-scan, while
// fileIndexPrefix still bounds both together as one range-delete window.
const (
	fileIndexKindNode byte = 0x01
	fileIndexKindEdge byte = 0x02
)

// appendSegment appends a length-prefixed encoding of seg to buf.
//
// The varint length prefix — not a literal separator byte like '/' — is
// what makes a segment's boundary unambiguous: a decoder always knows
// exactly how many bytes belong to this segment before the next one
// starts, so a crafted seg containing the namespace delimiter ('/'), a NUL
// byte (0x00), or 0xFF cannot be mistaken for a segment boundary or bleed
// into an adjacent segment or namespace. This is the mitigation for
// Security Domain V5 / T-01-02 (Tampering via crafted path/id): raw
// string concatenation with a literal separator would let an attacker-
// controlled id or path forge a key that lands in another namespace's
// range, or widen/narrow a range-delete window.
func appendSegment(buf []byte, seg string) []byte {
	var lenBuf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(lenBuf[:], uint64(len(seg)))
	buf = append(buf, lenBuf[:n]...)
	buf = append(buf, seg...)
	return buf
}

// nodeKey encodes a node id into its Pebble key under the n/ namespace.
// Distinct ids never collide or overlap: appendSegment's length prefix
// fixes the id's exact byte span regardless of its content, so no crafted
// id (containing '/', 0x00, or 0xFF) can be encoded to equal, or sort
// inside the range of, another node's key or a different namespace.
func nodeKey(id string) []byte {
	buf := make([]byte, 0, 1+binary.MaxVarintLen64+len(id))
	buf = append(buf, prefixNode)
	buf = appendSegment(buf, id)
	return buf
}

// edgeKey encodes an edge under the e/ namespace with src as the primary
// segment, so edgeSrcPrefix(src) isolates exactly that source's outgoing
// edges as one contiguous byte range (callers/callees/impact queries are a
// single Pebble prefix iteration, D-03).
//
// Design note (RESEARCH Pitfall 2 / Open Question 3): line/col are
// intentionally NOT part of this key in v1. Two structurally distinct call
// sites sharing the same (src, kind, dst) collapse to one stored edge
// here — this is deliberate dedup behavior, not a bug, mirroring the fix
// for the TS original's historical edge-duplication issue. Phase 2
// extractor design must revisit this if a future edge kind needs to
// preserve multiple call sites between the same two symbols as distinct
// edges. The Edge record itself already carries optional line/col fields
// (Plan 01-02), so no data is lost at extraction time — only the current
// key shape collapses it.
func edgeKey(src, kind, dst string) []byte {
	buf := make([]byte, 0, 1+3*binary.MaxVarintLen64+len(src)+len(kind)+len(dst))
	buf = append(buf, prefixEdge)
	buf = appendSegment(buf, src)
	buf = appendSegment(buf, kind)
	buf = appendSegment(buf, dst)
	return buf
}

// edgeSrcPrefix returns the prefix that a Pebble prefix iteration can use
// to select every edge whose source is src, in one contiguous range. The
// length-prefixed src segment ensures this prefix cannot be a partial
// match of a different, longer src (e.g. src "a" never matches edges whose
// real source is "ab") and cannot be crossed by a crafted src containing a
// delimiter or control byte.
func edgeSrcPrefix(src string) []byte {
	buf := make([]byte, 0, 1+binary.MaxVarintLen64+len(src))
	buf = append(buf, prefixEdge)
	buf = appendSegment(buf, src)
	return buf
}

// fileKey encodes a file record (path, content hash, aggregate counts —
// Plan 01-02's File message) under the f/ namespace.
func fileKey(path string) []byte {
	buf := make([]byte, 0, 1+binary.MaxVarintLen64+len(path))
	buf = append(buf, prefixFile)
	buf = appendSegment(buf, path)
	return buf
}

// fileSubgraphPrefix returns the inclusive lower bound of the byte range
// that a single Pebble DeleteRange call uses to prune path's own record
// under the f/ namespace (Plan 01-06's DeleteFileSubgraph). Paired with
// rangeUpperBound, [fileSubgraphPrefix(path), rangeUpperBound(prefix))
// covers exactly that file's own keys and excludes a lexicographically
// adjacent file whose path is a naive prefix or suffix of it (e.g. "foo"
// vs "foobar") — the length-prefixed path segment is what prevents that
// bleed; naive `"f/" + path` concatenation would not.
//
// NOTE: v1 does not (yet) key node/edge records by owning file, so this
// range-delete prunes the file's own f/ record only. Extending a single
// call to also prune that file's node/edge records is a Plan 01-06
// storage-layout decision (e.g. a file-scoped secondary index), not part
// of this key-encoder foundation.
func fileSubgraphPrefix(path string) []byte {
	return fileKey(path)
}

// fileIndexPrefix returns the inclusive lower bound of the byte range that
// bounds ALL of path's own file-index entries — both node and edge
// sub-ranges together — for a single Pebble DeleteRange call
// (DeleteFileSubgraph, Phase 4 D-02). Paired with rangeUpperBound,
// [fileIndexPrefix(path), rangeUpperBound(prefix)) covers exactly path's
// entries and excludes a lexicographically adjacent file whose path is a
// naive prefix or suffix of it (e.g. "foo" vs "foobar") — the same
// length-prefixed-segment argument as fileSubgraphPrefix.
func fileIndexPrefix(path string) []byte {
	buf := make([]byte, 0, 1+binary.MaxVarintLen64+len(path))
	buf = append(buf, prefixFileIndex)
	buf = appendSegment(buf, path)
	return buf
}

// fileIndexNodePrefix returns the lower bound of the contiguous byte range
// covering exactly path's owned node entries (the fileIndexKindNode
// sub-range within fileIndexPrefix(path)).
func fileIndexNodePrefix(path string) []byte {
	buf := fileIndexPrefix(path)
	return append(buf, fileIndexKindNode)
}

// fileIndexNodeKey encodes a (path, nodeID) file-index entry: path owns the
// node identified by nodeID. No value payload is stored under this key —
// the key itself encodes the reference (Sync's prune step decodes it
// straight from the key bytes, RESEARCH Pattern 2).
func fileIndexNodeKey(path, nodeID string) []byte {
	buf := fileIndexNodePrefix(path)
	return appendSegment(buf, nodeID)
}

// fileIndexEdgePrefix returns the lower bound of the contiguous byte range
// covering exactly path's owned outgoing-edge entries (the
// fileIndexKindEdge sub-range within fileIndexPrefix(path)).
func fileIndexEdgePrefix(path string) []byte {
	buf := fileIndexPrefix(path)
	return append(buf, fileIndexKindEdge)
}

// fileIndexEdgeKey encodes a (path, src, kind, dst) file-index entry: path
// owns the outgoing edge (src, kind, dst) — i.e. src is one of path's own
// nodes (the edge's ownerPath, threaded from resolve.go's writeGraph).
func fileIndexEdgeKey(path, src, kind, dst string) []byte {
	buf := fileIndexEdgePrefix(path)
	buf = appendSegment(buf, src)
	buf = appendSegment(buf, kind)
	return appendSegment(buf, dst)
}

// metaKey encodes a store-wide metadata entry (e.g. schema version) under
// the m/ namespace.
func metaKey(name string) []byte {
	buf := make([]byte, 0, 1+binary.MaxVarintLen64+len(name))
	buf = append(buf, prefixMeta)
	buf = appendSegment(buf, name)
	return buf
}

// decodeSegment reads one appendSegment-encoded segment from buf starting
// at offset, returning the segment's string value and the offset
// immediately after it. It is the decode counterpart to appendSegment,
// used by pebbleFileIndexIterator to reconstruct a file-index entry's
// id/src/kind/dst fields directly from key bytes — the x/ namespace stores
// no value payload, so the key alone encodes the reference.
func decodeSegment(buf []byte, offset int) (string, int, error) {
	if offset > len(buf) {
		return "", 0, fmt.Errorf("graphstore: segment offset %d beyond key length %d", offset, len(buf))
	}
	length, n := binary.Uvarint(buf[offset:])
	if n <= 0 {
		return "", 0, fmt.Errorf("graphstore: invalid varint segment-length prefix at offset %d", offset)
	}
	start := offset + n
	end := start + int(length)
	if end > len(buf) {
		return "", 0, fmt.Errorf("graphstore: segment length %d at offset %d exceeds key length %d", length, offset, len(buf))
	}
	return string(buf[start:end]), end, nil
}

// rangeUpperBound computes the exclusive upper bound for a Pebble range
// scan or range-delete over every key sharing prefix as its prefix. It is
// the standard byte-string "prefix successor": increment the last byte
// that is not already 0xFF, dropping any trailing 0xFF bytes first. If
// prefix is empty or consists entirely of 0xFF bytes, there is no finite
// successor; callers get a nil bound, which Pebble treats as "no upper
// limit".
func rangeUpperBound(prefix []byte) []byte {
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		if end[i] != 0xff {
			end[i]++
			return end[:i+1]
		}
	}
	return nil
}
