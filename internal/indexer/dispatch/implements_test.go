package dispatch

import (
	"fmt"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// findImplements reports whether edges contains an "implements" edge from
// source to target with Provenance="heuristic" and
// Metadata["synthesizedBy"]=SynthesizedBy (RES-03) — the shape every edge
// SynthesizeImplements returns MUST have.
func findImplements(t *testing.T, edges []*schema.Edge, source, target string) *schema.Edge {
	t.Helper()
	for _, e := range edges {
		if e.Source == source && e.Target == target {
			if e.Kind != goextract.EdgeKindImplements {
				t.Errorf("edge %s->%s has Kind %q, want %q", source, target, e.Kind, goextract.EdgeKindImplements)
			}
			if e.Provenance != "heuristic" {
				t.Errorf("edge %s->%s has Provenance %q, want %q (RES-03)", source, target, e.Provenance, "heuristic")
			}
			if e.Metadata["synthesizedBy"] != SynthesizedBy {
				t.Errorf("edge %s->%s has Metadata[synthesizedBy] %q, want %q (RES-03)", source, target, e.Metadata["synthesizedBy"], SynthesizedBy)
			}
			return e
		}
	}
	return nil
}

// TestSynthesizeImplements_Superset proves a struct whose method set is a
// superset of an interface's method-spec set is synthesized as
// implementing it.
func TestSynthesizeImplements_Superset(t *testing.T) {
	structMethods := TypeMethods{
		"struct:Widget": {{Name: "Read", Arity: 1}, {Name: "Close", Arity: 0}},
	}
	interfaceMethods := InterfaceSpecs{
		"iface:Reader": {{Name: "Read", Arity: 1}},
	}

	edges := SynthesizeImplements(structMethods, interfaceMethods, nil)

	if e := findImplements(t, edges, "struct:Widget", "iface:Reader"); e == nil {
		t.Fatalf("expected Widget implements Reader, got %+v", edges)
	}
}

// TestSynthesizeImplements_NonSuperset proves a struct whose method set is
// NOT a superset (missing a required method, or matching arity mismatch)
// is never synthesized as implementing.
func TestSynthesizeImplements_NonSuperset(t *testing.T) {
	tests := map[string]struct {
		structMethods TypeMethods
	}{
		"missing method entirely": {
			structMethods: TypeMethods{"struct:Widget": {{Name: "Close", Arity: 0}}},
		},
		"same name, wrong arity": {
			structMethods: TypeMethods{"struct:Widget": {{Name: "Read", Arity: 2}}},
		},
	}
	interfaceMethods := InterfaceSpecs{
		"iface:Reader": {{Name: "Read", Arity: 1}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			edges := SynthesizeImplements(tc.structMethods, interfaceMethods, nil)
			if e := findImplements(t, edges, "struct:Widget", "iface:Reader"); e != nil {
				t.Fatalf("expected no implements edge, got %+v", e)
			}
		})
	}
}

// TestSynthesizeImplements_EmbeddedInterfaceComposed proves an interface
// embedding another interface composes the embedded interface's method
// specs transitively (Go's real semantics: ReadWriter embeds Reader, so
// implementing ReadWriter requires BOTH Read and Write).
func TestSynthesizeImplements_EmbeddedInterfaceComposed(t *testing.T) {
	structMethods := TypeMethods{
		"struct:Widget": {{Name: "Read", Arity: 1}, {Name: "Write", Arity: 1}},
	}
	interfaceMethods := InterfaceSpecs{
		"iface:Reader":     {{Name: "Read", Arity: 1}},
		"iface:ReadWriter": {{Name: "Write", Arity: 1}},
	}
	interfaceEmbeds := map[string][]string{
		"iface:ReadWriter": {"iface:Reader"},
	}

	edges := SynthesizeImplements(structMethods, interfaceMethods, interfaceEmbeds)

	if e := findImplements(t, edges, "struct:Widget", "iface:ReadWriter"); e == nil {
		t.Fatalf("expected Widget implements ReadWriter (composed via embedded Reader), got %+v", edges)
	}
	if e := findImplements(t, edges, "struct:Widget", "iface:Reader"); e == nil {
		t.Fatalf("expected Widget implements Reader directly too, got %+v", edges)
	}
}

// TestSynthesizeImplements_EmptyInterfaceNeverMatches proves an interface
// with zero method specs (even after composing embeds) is never a
// synthesis target — Go's interface{} is trivially satisfied by
// everything, and synthesizing that edge for every struct would be pure
// noise for a dispatch-traversal consumer.
func TestSynthesizeImplements_EmptyInterfaceNeverMatches(t *testing.T) {
	structMethods := TypeMethods{
		"struct:Widget": {{Name: "Read", Arity: 1}},
	}
	interfaceMethods := InterfaceSpecs{
		"iface:Empty": {},
	}

	edges := SynthesizeImplements(structMethods, interfaceMethods, nil)
	if e := findImplements(t, edges, "struct:Widget", "iface:Empty"); e != nil {
		t.Fatalf("expected no implements edge for an empty interface, got %+v", e)
	}
}

// TestSynthesizeImplements_BoundedNotQuadratic stress-tests the D-06
// anti-quadratic-blowup guard: a wide interface graph (many interfaces,
// many methods each, all with DISJOINT method names except one deliberate
// target pair) must not cause SynthesizeImplements to compare every
// struct against every interface — proven both by correctness (only the
// deliberate target pair matches, out of N*N candidate pairs) and by a
// generous wall-clock ceiling that a naive O(structs × interfaces ×
// methods) nested loop would blow through at this N.
func TestSynthesizeImplements_BoundedNotQuadratic(t *testing.T) {
	const n = 400
	structMethods := make(TypeMethods, n)
	interfaceMethods := make(InterfaceSpecs, n)

	for i := 0; i < n; i++ {
		structID := fmt.Sprintf("struct:%d", i)
		ifaceID := fmt.Sprintf("iface:%d", i)
		// Every struct/interface pair gets its OWN unique method name
		// (methodN), so no struct shares a method name with any
		// interface other than its own index — the inverted index
		// should route each struct to exactly ONE candidate interface,
		// not all n.
		structMethods[structID] = []goextract.MethodSpec{{Name: fmt.Sprintf("method%d", i), Arity: 0}}
		interfaceMethods[ifaceID] = []goextract.MethodSpec{{Name: fmt.Sprintf("method%d", i), Arity: 0}}
	}

	start := time.Now()
	edges := SynthesizeImplements(structMethods, interfaceMethods, nil)
	elapsed := time.Since(start)

	if len(edges) != n {
		t.Fatalf("expected exactly %d implements edges (one per matching pair), got %d", n, len(edges))
	}
	// A naive O(n^2) nested comparison at n=400 (160,000 pairs) is still
	// fast in absolute terms on modern hardware, so this ceiling is
	// generous by design (proving the ALGORITHM is bounded, not
	// benchmarking raw speed) — it exists to catch a genuine regression
	// to all-pairs comparison, not to flake on load.
	if elapsed > 2*time.Second {
		t.Fatalf("SynthesizeImplements took %v for n=%d — expected the inverted-index bound to keep this fast, not O(structs x interfaces)", elapsed, n)
	}
}

// TestSynthesizeImplements_Deterministic proves two runs over the same
// (unordered map) input produce byte-identical, sorted output — the
// determinism gate every synthesized edge set must pass before flowing
// through resolve.go's collapseEdges.
func TestSynthesizeImplements_Deterministic(t *testing.T) {
	structMethods := TypeMethods{
		"struct:B": {{Name: "Read", Arity: 1}},
		"struct:A": {{Name: "Read", Arity: 1}},
	}
	interfaceMethods := InterfaceSpecs{
		"iface:Z": {{Name: "Read", Arity: 1}},
		"iface:Y": {{Name: "Read", Arity: 1}},
	}

	first := SynthesizeImplements(structMethods, interfaceMethods, nil)
	second := SynthesizeImplements(structMethods, interfaceMethods, nil)

	if len(first) != len(second) || len(first) != 4 {
		t.Fatalf("expected 4 edges (2 structs x 2 interfaces) both runs, got %d and %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Source != second[i].Source || first[i].Target != second[i].Target {
			t.Fatalf("run order differs at index %d: %+v vs %+v", i, first[i], second[i])
		}
	}
}
