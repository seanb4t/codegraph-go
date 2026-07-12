// Package dispatch synthesizes Go's implicit structural interface
// satisfaction into explicit "implements" edges (RES-02, Phase 5 Pattern
// 3): a struct whose method set (by name+arity, D-06's bounded-matching
// discipline) is a superset of an interface's own method-spec set is
// synthesized as implementing that interface. Go's declared-implements
// counterpart for Java/C# (Pattern 2: promoting a resolved extends/
// implements reference when its target is an interface node) lives in
// resolve.go instead, since that promotion happens inline while resolving
// a RefKindEmbeds unresolved ref and needs no separate synthesis pass.
//
// Every edge this package emits carries Provenance="heuristic" +
// Metadata["synthesizedBy"] (RES-03/D-07) — additive within SchemaVersion
// 1, no schema change. Matching is bounded via an inverted
// methodName->[]interfaceID pre-filter built BEFORE the struct×interface
// comparison loop (D-06) — never an O(structs × interfaces) nested loop
// over all pairs.
package dispatch

import (
	"sort"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// SynthesizedBy is the Metadata["synthesizedBy"] value every edge this
// package emits carries (RES-03) — the ONE heuristic this package
// implements. Java/C#'s declared-implements promotion (resolve.go, Pattern
// 2) uses its own distinct value ("declared-implements") so a reader of
// the committed graph can tell the two heuristics apart.
const SynthesizedBy = "go-structural-methodset"

// TypeMethods maps a struct node id to the method specs it declares —
// after Pass 2 has resolved every struct's full (possibly cross-file)
// method set via "contains" edges, exactly like
// TestResolve_CrossFileMethodContainment.
type TypeMethods map[string][]goextract.MethodSpec

// InterfaceSpecs maps an interface node id to the method specs it
// declares directly in its own body (embedded interfaces are NOT
// pre-flattened here — SynthesizeImplements composes them itself via
// interfaceEmbeds).
type InterfaceSpecs map[string][]goextract.MethodSpec

// SynthesizeImplements returns one "implements" edge per (struct,
// interface) pair where structMethods[structID]'s method set is a
// superset — by (Name, Arity) — of the interface's own method-spec set,
// composed transitively through interfaceEmbeds (interface embeds
// interface). An interface with zero methods, even after composing
// embeds, is never a synthesis target (Go's `interface{}` is trivially
// satisfied by everything; synthesizing an edge for that would be pure
// noise, not a useful dispatch target).
//
// Bounding (D-06): an inverted methodName->[]interfaceID index is built
// ONCE, before any struct is compared against a candidate interface — a
// struct is only ever compared against interfaces sharing at least one
// method name, never against the full interface set. interfaceEmbeds
// composition is bounded by the number of INTERFACES alone (a
// visited-set-guarded walk, cycle-safe), independent of struct count, so
// a wide or deeply-embedded interface graph cannot blow up the struct
// comparison either.
func SynthesizeImplements(structMethods TypeMethods, interfaceMethods InterfaceSpecs, interfaceEmbeds map[string][]string) []*schema.Edge {
	composed := composeEmbeddedInterfaceMethods(interfaceMethods, interfaceEmbeds)
	methodNameIndex := invertMethodIndex(composed)

	structIDs := make([]string, 0, len(structMethods))
	for id := range structMethods {
		structIDs = append(structIDs, id)
	}
	sort.Strings(structIDs)

	var edges []*schema.Edge
	for _, structID := range structIDs {
		specs := structMethods[structID]
		have := methodSet(specs)
		for _, ifaceID := range candidateInterfaces(specs, methodNameIndex) {
			if isSuperset(have, composed[ifaceID]) {
				edges = append(edges, &schema.Edge{
					Source:     structID,
					Target:     ifaceID,
					Kind:       goextract.EdgeKindImplements,
					Provenance: "heuristic",
					Metadata:   map[string]string{"synthesizedBy": SynthesizedBy},
				})
			}
		}
	}
	return edges
}

// composeEmbeddedInterfaceMethods returns, per interface id, the union of
// its own method specs plus every transitively embedded interface's
// specs — bounded by a visited-set-guarded walk over interfaces only
// (never touching structMethods), so this composition step's cost is a
// function of interface-graph size alone.
func composeEmbeddedInterfaceMethods(own InterfaceSpecs, embeds map[string][]string) InterfaceSpecs {
	composed := make(InterfaceSpecs, len(own))
	for ifaceID := range own {
		composed[ifaceID] = collectTransitive(ifaceID, own, embeds, make(map[string]bool))
	}
	return composed
}

func collectTransitive(ifaceID string, own InterfaceSpecs, embeds map[string][]string, visited map[string]bool) []goextract.MethodSpec {
	if visited[ifaceID] {
		return nil
	}
	visited[ifaceID] = true

	specs := append([]goextract.MethodSpec(nil), own[ifaceID]...)
	for _, embedded := range embeds[ifaceID] {
		specs = append(specs, collectTransitive(embedded, own, embeds, visited)...)
	}
	return specs
}

// invertMethodIndex builds the methodName->[]interfaceID pre-filter (D-06)
// that bounds the struct×interface comparison loop — built once, before
// any struct is compared.
func invertMethodIndex(interfaceMethods InterfaceSpecs) map[string][]string {
	idx := make(map[string][]string)
	ids := make([]string, 0, len(interfaceMethods))
	for id := range interfaceMethods {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, ifaceID := range ids {
		seen := make(map[string]bool)
		for _, spec := range interfaceMethods[ifaceID] {
			if seen[spec.Name] {
				continue
			}
			seen[spec.Name] = true
			idx[spec.Name] = append(idx[spec.Name], ifaceID)
		}
	}
	return idx
}

// candidateInterfaces returns the deduplicated, sorted set of interface
// ids sharing at least one method NAME with structSpecs — the D-06
// pre-filter's output: a struct is compared against these candidates
// only, never every interface in the graph.
func candidateInterfaces(structSpecs []goextract.MethodSpec, methodNameIndex map[string][]string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range structSpecs {
		for _, ifaceID := range methodNameIndex[s.Name] {
			if seen[ifaceID] {
				continue
			}
			seen[ifaceID] = true
			out = append(out, ifaceID)
		}
	}
	sort.Strings(out)
	return out
}

func methodSet(specs []goextract.MethodSpec) map[goextract.MethodSpec]struct{} {
	set := make(map[goextract.MethodSpec]struct{}, len(specs))
	for _, s := range specs {
		set[s] = struct{}{}
	}
	return set
}

// isSuperset reports whether have contains every spec in want (Go's
// structural interface satisfaction, D-06's name+arity bound). An
// interface with zero method specs (even after composing embeds) never
// matches — synthesizing "every struct implements interface{}" would be
// pure noise for a dispatch-traversal consumer.
func isSuperset(have map[goextract.MethodSpec]struct{}, want []goextract.MethodSpec) bool {
	if len(want) == 0 {
		return false
	}
	for _, w := range want {
		if _, ok := have[w]; !ok {
			return false
		}
	}
	return true
}
