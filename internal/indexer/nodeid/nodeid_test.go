package nodeid

import (
	"regexp"
	"testing"
)

var idShape = regexp.MustCompile(`^[^:]*:[0-9a-f]{32}$`)

// TestNodeID_Shape proves the id has the TS-parity <kind>:<32-hex> shape
// and that the kind prefix is preserved verbatim.
func TestNodeID_Shape(t *testing.T) {
	id := NodeID("function", "mergeStyle", "internal/cli/finish.go")

	if !idShape.MatchString(id) {
		t.Fatalf("NodeID() = %q, want shape <kind>:<32 lowercase hex chars>", id)
	}

	const wantPrefix = "function:"
	if id[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("NodeID() = %q, want kind prefix %q preserved verbatim", id, wantPrefix)
	}
}

// TestNodeID_Deterministic proves re-running node-id construction on the
// same identity tuple yields a byte-identical id.
func TestNodeID_Deterministic(t *testing.T) {
	a := NodeID("function", "mergeStyle", "internal/cli/finish.go")
	b := NodeID("function", "mergeStyle", "internal/cli/finish.go")
	if a != b {
		t.Fatalf("NodeID() not deterministic: %q != %q", a, b)
	}
}

// TestNodeID_Distinct proves distinct identity tuples never collide.
func TestNodeID_Distinct(t *testing.T) {
	cases := []struct {
		name                string
		kind1, name1, path1 string
		kind2, name2, path2 string
	}{
		{
			name:  "different file path",
			kind1: "function", name1: "F", path1: "a.go",
			kind2: "function", name2: "F", path2: "b.go",
		},
		{
			name:  "different kind",
			kind1: "function", name1: "F", path1: "a.go",
			kind2: "method", name2: "F", path2: "a.go",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			id1 := NodeID(c.kind1, c.name1, c.path1)
			id2 := NodeID(c.kind2, c.name2, c.path2)
			if id1 == id2 {
				t.Fatalf("NodeID collision: %q == %q for distinct tuples", id1, id2)
			}
		})
	}
}

// TestNodeID_InjectionSafe is the T-02-01 gate: a crafted qualified_name
// containing the segment bytes of a file path must NOT collide with the
// naive concatenation. The length-prefixed preimage fixes each segment's
// exact byte span so a boundary cannot slide.
func TestNodeID_InjectionSafe(t *testing.T) {
	id1 := NodeID("t", "a", "bc")
	id2 := NodeID("t", "ab", "c")
	if id1 == id2 {
		t.Fatalf("NodeID injection-unsafe: NodeID(\"t\",\"a\",\"bc\") == NodeID(\"t\",\"ab\",\"c\") = %q", id1)
	}
}

// TestNodeID_EmptyArgs proves empty-string arguments are handled without
// panic and still produce a well-formed <kind>:<hex> id.
func TestNodeID_EmptyArgs(t *testing.T) {
	id := NodeID("", "", "")
	if !idShape.MatchString(id) {
		t.Fatalf("NodeID(\"\",\"\",\"\") = %q, want shape <kind>:<32 lowercase hex chars>", id)
	}
}
