package schema

import (
	"testing"

	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/google/go-cmp/cmp"
)

// TestSchemaRoundTripsUnknownFields is the ARCH-01 forward-compatibility
// proof: a record written by a schema newer than this build's (carrying a
// field number inside Node's reserved 50-59 range) must unmarshal cleanly
// through the current reader, and re-marshaling must not drop the unknown
// field. This is the literal mechanism ("no format break on future
// node/edge annotation fields") that D-02/D-02a and the reserved ranges in
// graph.proto exist to guarantee.
func TestSchemaRoundTripsUnknownFields(t *testing.T) {
	original := &Node{
		Id:            "func:pkg.Foo",
		Kind:          "function",
		Name:          "Foo",
		QualifiedName: "pkg.Foo",
		FilePath:      "pkg/foo.go",
		Language:      "go",
		StartLine:     10,
		EndLine:       20,
		StartCol:      1,
		EndCol:        2,
	}

	knownBytes, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("marshal original Node: %v", err)
	}

	// Simulate a record written by a future schema: append a field at
	// number 55, inside Node's reserved 50-59 range, that this build's
	// Node message does not define. Field 55 as a varint carrying an
	// arbitrary placeholder value (e.g. a future embedding-dimension
	// count or similar scalar annotation).
	const futureFieldNumber = protowire.Number(55)
	const futureFieldValue = uint64(42)
	newerWriterBytes := append([]byte{}, knownBytes...)
	newerWriterBytes = protowire.AppendTag(newerWriterBytes, futureFieldNumber, protowire.VarintType)
	newerWriterBytes = protowire.AppendVarint(newerWriterBytes, futureFieldValue)

	var decoded Node
	if err := proto.Unmarshal(newerWriterBytes, &decoded); err != nil {
		t.Fatalf("a record with an unknown/future field must unmarshal through the current reader without error: %v", err)
	}

	// The known fields must still be intact. The injected unknown field is
	// deliberately excluded here (via protocmp.IgnoreUnknown) — its
	// preservation is asserted separately below by scanning the
	// re-marshaled bytes, which is the actual "no format break" proof.
	if diff := cmp.Diff(original, &decoded, protocmp.Transform(), protocmp.IgnoreUnknown()); diff != "" {
		t.Fatalf("decoded Node differs from original on known fields (-want +got):\n%s", diff)
	}

	// Re-marshal and assert the unknown field survived the round trip —
	// this is the actual "no format break" proof: an intermediate reader
	// that merely passes a record through (e.g. a future re-index or
	// export path) must not silently drop data it doesn't understand.
	remarshaled, err := proto.Marshal(&decoded)
	if err != nil {
		t.Fatalf("remarshal decoded Node: %v", err)
	}

	found := false
	b := remarshaled
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			t.Fatalf("failed to consume tag while scanning remarshaled bytes: %v", protowire.ParseError(n))
		}
		b = b[n:]
		if num == futureFieldNumber && typ == protowire.VarintType {
			val, n := protowire.ConsumeVarint(b)
			if n < 0 {
				t.Fatalf("failed to consume varint for injected field: %v", protowire.ParseError(n))
			}
			if val != futureFieldValue {
				t.Fatalf("injected field %d survived but value changed: want %d, got %d", futureFieldNumber, futureFieldValue, val)
			}
			found = true
			b = b[n:]
			continue
		}
		n2 := protowire.ConsumeFieldValue(num, typ, b)
		if n2 < 0 {
			t.Fatalf("failed to skip field %d while scanning remarshaled bytes: %v", num, protowire.ParseError(n2))
		}
		b = b[n2:]
	}
	if !found {
		t.Fatalf("re-marshaled bytes lost the injected unknown field %d — this is exactly the ARCH-01 format break the reserved range guards against", futureFieldNumber)
	}
}

// TestNodeRoundTripsEqual asserts a plain marshal->unmarshal cycle with no
// unknown fields produces an equal Node under protocmp.
func TestNodeRoundTripsEqual(t *testing.T) {
	original := &Node{
		Id:       "type:pkg.Bar",
		Kind:     "type",
		Name:     "Bar",
		FilePath: "pkg/bar.go",
		Language: "go",
	}

	b, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Node
	if err := proto.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if diff := cmp.Diff(original, &decoded, protocmp.Transform()); diff != "" {
		t.Fatalf("Node did not round-trip (-want +got):\n%s", diff)
	}
}

// TestMetaRoundTripsSchemaVersion asserts Meta.schema_version survives a
// marshal->unmarshal cycle.
func TestMetaRoundTripsSchemaVersion(t *testing.T) {
	original := NewMeta()
	original.NodeCount = 1221
	original.EdgeCount = 4063

	b, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Meta
	if err := proto.Unmarshal(b, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.GetSchemaVersion() != SchemaVersion {
		t.Fatalf("SchemaVersion did not round-trip: want %d, got %d", SchemaVersion, decoded.GetSchemaVersion())
	}
	if !IsCurrentSchemaVersion(&decoded) {
		t.Fatalf("IsCurrentSchemaVersion returned false for a record stamped with the current SchemaVersion")
	}
	if diff := cmp.Diff(original, &decoded, protocmp.Transform()); diff != "" {
		t.Fatalf("Meta did not round-trip (-want +got):\n%s", diff)
	}
}
