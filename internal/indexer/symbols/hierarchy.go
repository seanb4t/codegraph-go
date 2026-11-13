package symbols

import (
	"slices"
	"sort"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// OverloadSet is the node that holds two or more overloads of one name
// under one owner.
type OverloadSet struct {
	node    *schema.Node
	ownerID string
	members []Candidate
}

// Node returns the set's node.
func (s *OverloadSet) Node() *schema.Node { return s.node }

// OwnerID returns the id of the type or file that owns the set.
func (s *OverloadSet) OwnerID() string { return s.ownerID }

// Members returns the overloads, by file and start line.
func (s *OverloadSet) Members() []Candidate { return s.members }

// Hierarchy is the read-only type hierarchy of one resolution: the
// supertypes of each type, the owner of each member and the declared
// overload sets. NewHierarchy builds it from the type-level edges of
// every file, which resolution needs before it can resolve a call.
type Hierarchy struct {
	h *hierarchy
}

type hierarchy struct {
	view View

	// typeSupertypes maps a type to the types it embeds, extends or
	// implements, and ownerOf maps a member to the type that contains it.
	typeSupertypes map[string][]string
	ownerOf        map[string]string
	// subtyped holds the types that another type embeds, extends or
	// implements.
	subtyped map[string]bool

	// sets holds every overload set by id.
	sets map[string]*OverloadSet
}

// NewHierarchy reads the supertype and ownership edges of typeEdges and
// of the view's stored edges, then declares the overload set of every
// overloaded name in view: the members of each type and the functions of
// each file. A set exists even when no call is ambiguous (D14).
func NewHierarchy(view View, typeEdges []*schema.Edge) Hierarchy {
	h := &hierarchy{
		view:           view,
		typeSupertypes: make(map[string][]string),
		ownerOf:        make(map[string]string),
		subtyped:       make(map[string]bool),
		sets:           make(map[string]*OverloadSet),
	}
	edges := typeEdges
	if stored := view.StoredEdges(); len(stored) > 0 {
		edges = append(slices.Clip(typeEdges), stored...)
	}
	h.learnEdges(edges)
	h.declareSets()
	return Hierarchy{h: h}
}

// learnEdges reads the supertype and ownership edges. A member the store
// holds under an overload set belongs to the set's owner.
func (h *hierarchy) learnEdges(edges []*schema.Edge) {
	for _, e := range edges {
		switch e.Kind {
		case "embeds", goextract.EdgeKindExtends, goextract.EdgeKindImplements:
			h.typeSupertypes[e.Source] = append(h.typeSupertypes[e.Source], e.Target)
			h.subtyped[e.Target] = true
		case goextract.RefKindContains:
			if IsTypeKind(h.view.Kind(e.Source)) {
				h.ownerOf[e.Target] = e.Source
			}
		}
	}
	for _, id := range h.view.StoredSetIDs() {
		set, _ := h.view.StoredSet(id)
		if !IsTypeKind(h.view.Kind(set.Owner)) {
			continue
		}
		for _, m := range set.Members {
			h.ownerOf[m] = set.Owner
		}
	}
}

func (h *hierarchy) declareSets() {
	declare := func(cands []Candidate) {
		if owner, ok := h.overloadGroup(cands); ok {
			h.setOf(owner, cands[0].Node.Name, cands)
		}
	}
	for ownerID, byName := range h.view.d.membersByOwner {
		for _, cands := range byName {
			for _, c := range cands {
				if _, ok := h.ownerOf[c.ID]; !ok {
					h.ownerOf[c.ID] = ownerID
				}
			}
			declare(cands)
		}
	}
	// Free functions group by file and name over every node of the pass,
	// reachable or not: a file's unexported functions overload too.
	type fileName struct{ file, name string }
	groups := make(map[fileName][]Candidate)
	for _, n := range h.view.d.nodes {
		if n.Kind != goextract.KindFunction || n.Nested {
			continue
		}
		k := fileName{n.FilePath, n.Name}
		groups[k] = append(groups[k], CandidateOf(n))
	}
	keys := make([]fileName, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].file != keys[j].file {
			return keys[i].file < keys[j].file
		}
		return keys[i].name < keys[j].name
	})
	for _, k := range keys {
		declare(groups[k])
	}
}

// overloadGroup reports whether cands are the overloads of one name
// under one owner (design D14): the same kind, the same owner, no nested
// declaration, and distinct signature keys. Copies with one key are
// redefinitions, not overloads, except in the languages where a
// repeated signature can only be a conditional-compilation alternative
// (D16), and a language that replaces a definition at run time has no
// overloads at all.
func (h *hierarchy) overloadGroup(cands []Candidate) (string, bool) {
	if len(cands) < 2 || LastWins(cands[0].Node.Language) {
		return "", false
	}
	owner, ok := h.groupOwner(cands)
	if !ok {
		return "", false
	}
	keys := make(map[string]bool, len(cands))
	for _, c := range cands {
		if c.Node.Nested {
			return "", false
		}
		if keys[c.Key] && !duplicatesAreAlternatives(c.Node.Language) {
			return "", false
		}
		keys[c.Key] = true
	}
	return owner, true
}

// LastWins reports whether lang replaces an earlier definition of a name
// with a later one at run time (design D16), so that a repeat is never
// an overload.
func LastWins(lang string) bool {
	switch lang {
	case "python", "ruby", "javascript":
		return true
	}
	return false
}

// duplicatesAreAlternatives reports whether two declarations with one
// signature under one owner are alternatives under `#if` or a
// `function_exists` guard in lang, since the language rejects a plain
// repeat (design D16).
func duplicatesAreAlternatives(lang string) bool {
	switch lang {
	case "c", "cpp", "csharp", "php":
		return true
	}
	return false
}

// groupOwner returns the owner shared by every candidate: the type that
// contains them, or the file that declares them, and false when they
// differ.
func (h *hierarchy) groupOwner(cands []Candidate) (string, bool) {
	owner := ""
	for i, c := range cands {
		o, ok := h.ownerOf[c.ID]
		if !ok {
			if c.Kind != goextract.KindFunction {
				return "", false
			}
			o = nodeid.NodeID(goextract.KindFile, c.FilePath, c.FilePath)
		}
		if i == 0 {
			owner = o
		} else if o != owner || c.Kind != cands[0].Kind {
			return "", false
		}
	}
	return owner, owner != ""
}

// setOf returns the overload set for name under ownerID, creating its
// node from members on first use (D14). The node spans the members in
// their file. When they span files, the node has no location: its
// FilePath is "", the sentinel a merged C# partial type uses too. The
// sync relies on it: no file prune reaches such a set, so the sync
// deletes it when a resolve stops declaring it, and treats it as owned
// by no file.
func (h *hierarchy) setOf(ownerID, name string, members []Candidate) *OverloadSet {
	id := nodeid.NodeID(goextract.KindOverloadSet, name, ownerID)
	if set, ok := h.sets[id]; ok {
		return set
	}
	sorted := append([]Candidate(nil), members...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].FilePath != sorted[j].FilePath {
			return sorted[i].FilePath < sorted[j].FilePath
		}
		return sorted[i].Node.StartLine < sorted[j].Node.StartLine
	})
	first := sorted[0].Node
	n := &schema.Node{
		Id:            id,
		Kind:          goextract.KindOverloadSet,
		Name:          first.Name,
		QualifiedName: first.QualifiedName,
		Language:      first.Language,
		Visibility:    first.Visibility,
		IsExported:    first.IsExported,
	}
	sameFile := true
	for _, m := range sorted {
		if m.FilePath != first.FilePath {
			sameFile = false
		}
	}
	if sameFile {
		last := sorted[len(sorted)-1].Node
		n.FilePath = first.FilePath
		n.StartLine, n.StartCol = first.StartLine, first.StartCol
		n.EndLine, n.EndCol = last.EndLine, last.EndCol
		for _, m := range sorted {
			if m.Node.EndLine > n.EndLine {
				n.EndLine, n.EndCol = m.Node.EndLine, m.Node.EndCol
			}
		}
	}
	set := &OverloadSet{node: n, ownerID: ownerID, members: sorted}
	h.sets[id] = set
	return set
}

// OwnerOf returns the type that contains the member id.
func (h Hierarchy) OwnerOf(id string) (string, bool) {
	o, ok := h.h.ownerOf[id]
	return o, ok
}

// WalkSupertypes visits the supertypes of typeID, breadth first; see
// WalkSupertypes. The zero Hierarchy has no supertypes.
func (h Hierarchy) WalkSupertypes(typeID string, visit func(supertypeID string) bool) bool {
	if h.h == nil {
		return false
	}
	return WalkSupertypes(typeID, h.h.typeSupertypes, visit)
}

// HasSubtypes reports whether another type embeds, extends or implements
// typeID.
func (h Hierarchy) HasSubtypes(typeID string) bool {
	return h.h != nil && h.h.subtyped[typeID]
}

// Set returns the overload set id.
func (h Hierarchy) Set(id string) (*OverloadSet, bool) {
	s, ok := h.h.sets[id]
	return s, ok
}

// SetFor returns the overload set that NewHierarchy declared for cands:
// the set of their name under their shared owner. It makes no set
// itself, so a call never creates a set that a fresh index would not
// hold. It returns nil when the candidates have no shared owner, when
// one of them is nested, or when no set exists for them.
func (h Hierarchy) SetFor(cands []Candidate) *OverloadSet {
	owner, ok := h.h.groupOwner(cands)
	if !ok {
		return nil
	}
	for _, c := range cands {
		if c.Node.Nested {
			return nil
		}
	}
	return h.h.sets[nodeid.NodeID(goextract.KindOverloadSet, cands[0].Node.Name, owner)]
}

// SetNodes returns the set nodes in id order.
func (h Hierarchy) SetNodes() []*schema.Node {
	ids := make([]string, 0, len(h.h.sets))
	for id := range h.h.sets {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]*schema.Node, len(ids))
	for i, id := range ids {
		out[i] = h.h.sets[id].node
	}
	return out
}

// WalkSupertypes performs a visited-set-guarded breadth-first search
// from typeID's own supertypes, never typeID itself. It calls visit on
// each reachable supertype in discovery order and stops as soon as visit
// returns true. It is bounded by the number of distinct types the
// supertype graph reaches, so a cycle in malformed embeds, extends or
// implements data ends it.
func WalkSupertypes(typeID string, typeSupertypes map[string][]string, visit func(supertypeID string) bool) bool {
	visited := map[string]bool{typeID: true}
	queue := append([]string(nil), typeSupertypes[typeID]...)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur] {
			continue
		}
		visited[cur] = true
		if visit(cur) {
			return true
		}
		queue = append(queue, typeSupertypes[cur]...)
	}
	return false
}
