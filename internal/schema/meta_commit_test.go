package schema

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
)

// TestMetaCommitSHARoundTrips is the ENG-04/D-05 happy-path proof: a Meta
// with the commit field set marshals and unmarshals with the value intact.
func TestMetaCommitSHARoundTrips(t *testing.T) {
	const sha = "e08b1a3b3e9dfea6b06e17df8a2c8e37e3c9d0f1"

	original := NewMeta()
	original.CommitSha = sha

	b, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Meta
	if err := proto.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.GetCommitSha() != sha {
		t.Fatalf("CommitSha did not round-trip: want %q, got %q", sha, decoded.GetCommitSha())
	}

	got, ok := IndexedCommitSHA(&decoded)
	if !ok || got != sha {
		t.Fatalf("IndexedCommitSHA(decoded) = (%q, %v), want (%q, true)", got, ok, sha)
	}
}

// TestMetaCommitSHAAbsentDegrades is the D-05 backward-compatibility proof:
// a Meta deserialized from raw wire bytes that genuinely OMIT field 8 (not
// a Go struct left at its zero value) must still unmarshal cleanly, the
// getter must return the empty string, and IndexedCommitSHA must report
// absent — never an error. This is the wire-level proof that a pre-upgrade
// graph (or any record a peer wrote before this field existed) decodes
// fine under a build that knows about field 8.
func TestMetaCommitSHAAbsentDegrades(t *testing.T) {
	// Hand-build wire bytes for a Meta carrying only fields 1-7 (schema
	// fields that predate field 8), with NO tag for field 8 anywhere in
	// the byte stream — this is what genuinely distinguishes this test
	// from constructing a Go Meta{} literal and leaving CommitSha at its
	// zero value, which would prove nothing about wire-level absence.
	var raw []byte
	raw = protowire.AppendTag(raw, protowire.Number(1), protowire.VarintType)
	raw = protowire.AppendVarint(raw, uint64(SchemaVersion))
	raw = protowire.AppendTag(raw, protowire.Number(2), protowire.VarintType)
	raw = protowire.AppendVarint(raw, 42)
	raw = protowire.AppendTag(raw, protowire.Number(7), protowire.VarintType)
	raw = protowire.AppendVarint(raw, 1) // has_file_index = true

	var decoded Meta
	if err := proto.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("a record genuinely missing field 8 must unmarshal through the current reader without error: %v", err)
	}

	if got := decoded.GetCommitSha(); got != "" {
		t.Fatalf("GetCommitSha() on a record with no field 8 = %q, want empty string", got)
	}

	sha, ok := IndexedCommitSHA(&decoded)
	if ok || sha != "" {
		t.Fatalf("IndexedCommitSHA(decoded) = (%q, %v), want (\"\", false) for a record with no field 8", sha, ok)
	}

	// Nil safety, and the empty-string-set case: both are "absent", not an
	// error — matching has_file_index's own degrade contract.
	if sha, ok := IndexedCommitSHA(nil); ok || sha != "" {
		t.Fatalf("IndexedCommitSHA(nil) = (%q, %v), want (\"\", false)", sha, ok)
	}
	empty := NewMeta()
	empty.CommitSha = ""
	if sha, ok := IndexedCommitSHA(empty); ok || sha != "" {
		t.Fatalf("IndexedCommitSHA(empty CommitSha) = (%q, %v), want (\"\", false)", sha, ok)
	}
}

// knownMetaFieldNumber is one transcribed (name, number) fact from
// graph.proto's Meta message, dated at transcription time so a future
// reader can tell how stale the fixture is relative to the source.
type knownMetaFieldNumber struct {
	protoName string
	number    protowire.Number
}

// knownMetaFieldNumbers is a literal fixture transcribed from
// internal/schema/graph.proto's Meta message as of 2026-08-23 (Phase 1,
// 01-06, D-05's field-8 addition). It is deliberately a SUBSET fixture,
// not an exact-cardinality assertion: TestKnownMetaFieldNumbersAreStable
// checks that every name below still maps to its expected number and that
// no two fields in the message share a number, but never asserts the
// message has exactly this many fields — a future additive field 9 must
// pass this test, not break it.
var knownMetaFieldNumbers = []knownMetaFieldNumber{
	{"schema_version", 1},
	{"node_count", 2},
	{"edge_count", 3},
	{"last_sync_unix_ms", 4},
	{"healthy", 5},
	{"health_message", 6},
	{"has_file_index", 7},
	{"commit_sha", 8},
}

// TestKnownMetaFieldNumbersAreStable is D-02a's guard against exactly the
// two things the additive-only discipline forbids: renumbering a shipped
// field, and two fields colliding on one number. It reads the GENERATED
// DESCRIPTOR (not a source grep, which could not tell a real Meta field
// from a comment), asserts every known name keeps its expected number,
// asserts no duplicate numbers exist anywhere in the message, and asserts
// (logging) a non-zero count of known mappings actually checked — so a
// typo'd or emptied fixture cannot pass vacuously.
func TestKnownMetaFieldNumbersAreStable(t *testing.T) {
	desc := (&Meta{}).ProtoReflect().Descriptor()
	fields := desc.Fields()

	byNumber := make(map[protowire.Number]string, fields.Len())
	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		num := protowire.Number(fd.Number())
		if existing, dup := byNumber[num]; dup {
			t.Fatalf("field number %d is shared by both %q and %q — D-02a forbids two fields colliding on one number", num, existing, fd.Name())
		}
		byNumber[num] = string(fd.Name())
	}

	checked := 0
	for _, known := range knownMetaFieldNumbers {
		gotName, present := byNumber[known.number]
		if !present {
			t.Fatalf("known field %q (number %d) is missing from the generated descriptor entirely", known.protoName, known.number)
		}
		if gotName != known.protoName {
			t.Fatalf("field number %d is named %q in the generated descriptor, want %q — this is exactly the renumber D-02a forbids", known.number, gotName, known.protoName)
		}
		checked++
	}

	t.Logf("checked %d known Meta field-number mappings against the generated descriptor", checked)
	if checked == 0 {
		t.Fatal("checked 0 known field mappings — the fixture is empty and this guard is vacuous")
	}
}
