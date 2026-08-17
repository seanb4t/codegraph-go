package nodeid

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

// appendSegment appends a varint-length-prefixed encoding of seg to buf.
//
// This reimplements internal/graphstore/keys.go's appendSegment technique
// locally (that function is unexported and lives in a different package).
// The length prefix — not a literal separator byte — is what makes a
// segment's boundary unambiguous: a crafted qualified_name or file_path
// cannot slide the boundary between segments to forge a collision with a
// different (kind, qualified_name, file_path) tuple. This is the T-02-01
// mitigation applied to the id hash preimage.
func appendSegment(buf []byte, seg string) []byte {
	var lenBuf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(lenBuf[:], uint64(len(seg)))
	buf = append(buf, lenBuf[:n]...)
	buf = append(buf, seg...)
	return buf
}

// NodeID computes the deterministic, collision-resistant identifier for a
// symbol identified by (kind, qualifiedName, filePath), per D-02/D-02a.
//
// The preimage is built from varint-length-prefixed segments (mirroring
// keys.go's appendSegment discipline) so a crafted qualifiedName or
// filePath cannot bleed across the segment boundary and forge/collide an
// id with a different identity tuple (T-02-01, Tampering).
//
// The preimage is hashed with SHA-256 — never MD5 or another
// collision-prone hash (D-02a, T-02-05, ASVS V6) — and the digest is
// hex-encoded and truncated to 32 characters, matching the project's
// <kind>:<32-hex> id shape while retaining SHA-256's collision
// resistance for the truncated portion.
//
// The result is a pure function of its arguments: re-running NodeID on the
// same tuple always yields the same id, which is how the resolver matches
// a reference back to the symbol it points to (RES-01).
func NodeID(kind, qualifiedName, filePath string) string {
	buf := make([]byte, 0, 3*binary.MaxVarintLen64+len(kind)+len(qualifiedName)+len(filePath))
	buf = appendSegment(buf, kind)
	buf = appendSegment(buf, qualifiedName)
	buf = appendSegment(buf, filePath)

	sum := sha256.Sum256(buf)
	return kind + ":" + hex.EncodeToString(sum[:])[:32]
}
