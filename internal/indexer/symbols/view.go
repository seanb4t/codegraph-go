package symbols

import (
	"iter"
	"maps"
	"slices"
	"sort"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// View is the read-only declaration universe of one resolution, from
// Builder.Freeze. It has no method that writes. The slices it returns
// belong to the View, and callers must not change them.
type View struct {
	d *data
}

// CandidatesIn returns every candidate named name in moduleKey, in the
// order the Builder received them.
func (v View) CandidatesIn(moduleKey, name string) []Candidate {
	return v.d.candidates[moduleKey][name]
}

// TypeIn returns the type named name in moduleKey: a struct, interface or
// type alias, chosen by Pick when there are several.
func (v View) TypeIn(moduleKey, name string) (Candidate, bool) {
	var types []Candidate
	for _, c := range v.d.candidates[moduleKey][name] {
		if IsTypeKind(c.Kind) {
			types = append(types, c)
		}
	}
	if len(types) == 0 {
		return Candidate{}, false
	}
	winner := Pick(types)
	for _, c := range types {
		if c.ID == winner {
			return c, true
		}
	}
	panic("symbols: Pick chose an id outside its candidates")
}

// Members returns the candidates named name that ownerID contains.
func (v View) Members(ownerID, name string) []Candidate {
	return v.d.membersByOwner[ownerID][name]
}

// Node returns the node id, or nil.
func (v View) Node(id string) *schema.Node { return v.d.nodes[id] }

// Nodes returns every node, in no fixed order.
func (v View) Nodes() iter.Seq[*schema.Node] { return maps.Values(v.d.nodes) }

// Kind returns the kind of the node id, or "".
func (v View) Kind(id string) string { return v.d.kinds[id] }

// Name returns the name of the node id, or "".
func (v View) Name(id string) string { return v.d.names[id] }

// StartLine returns the start line of the node id, or 0.
func (v View) StartLine(id string) int32 { return v.d.startLines[id] }

// Arity returns the parameter count of the method id, or 0.
func (v View) Arity(id string) int32 { return v.d.arity[id] }

// InterfaceMethods returns a copy of the method specs of every interface.
func (v View) InterfaceMethods() map[string][]goextract.MethodSpec {
	return maps.Clone(v.d.interfaceMethods)
}

// ModuleOf returns the module key of file, or "".
func (v View) ModuleOf(file string) string { return v.d.moduleOf[file] }

// IsTypeName reports whether a type node is named name.
func (v View) IsTypeName(name string) bool { return v.d.typeNames[name] }

// TypesNamed returns the ids of the type nodes named name, in id order.
func (v View) TypesNamed(name string) []string { return v.d.typesByName[name] }

// StoredSet returns the stored overload set id.
func (v View) StoredSet(id string) (StoredSet, bool) {
	s, ok := v.d.storedSets[id]
	return s, ok
}

// StoredSetIDs returns the ids of the stored overload sets, in id order.
func (v View) StoredSetIDs() []string {
	ids := slices.Collect(maps.Keys(v.d.storedSets))
	sort.Strings(ids)
	return ids
}

// StoredEdges returns the contains, embeds, extends, declared implements
// and binding edges of the files a sync does not reparse.
func (v View) StoredEdges() []*schema.Edge { return slices.Clip(v.d.storedEdges) }

// SyntaxErrorModules returns the module keys of the files in which the
// parser recovered from a syntax error.
func (v View) SyntaxErrorModules() map[string]bool {
	out := make(map[string]bool, len(v.d.syntaxErrorFiles))
	for f := range v.d.syntaxErrorFiles {
		out[v.d.moduleOf[f]] = true
	}
	return out
}

// Collisions returns the number of names that received more than one
// symbol.
func (v View) Collisions() int { return v.d.collisions }
