package graphstore

import "encoding/binary"

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

// metaKey encodes a store-wide metadata entry (e.g. schema version) under
// the m/ namespace.
func metaKey(name string) []byte {
	buf := make([]byte, 0, 1+binary.MaxVarintLen64+len(name))
	buf = append(buf, prefixMeta)
	buf = appendSegment(buf, name)
	return buf
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
