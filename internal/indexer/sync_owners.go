package indexer

import (
	"errors"
	"maps"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/resolve"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// ownerDependents returns the files outside inBatch whose field
// references can resolve differently after the reparse of paths, in path
// order. It covers the languages whose classes reach the fields of their
// base classes. results are the new extraction results of paths.
//
// A reference to a field depends on the owner chain of the receiver: a
// subclass can gain or lose a shadowing field, a base class can gain or
// lose the field or change its access, and a new or removed subclass
// opens or closes a field to replacement. Only a path whose owner facts
// change counts: its types, their supertypes that name a stored or new
// type, or its fields and their visibility. The affected types are the types of such a path, before
// and after, and the supertypes they name, with all their stored
// ancestors. A file that refers to a field of an affected type is a
// dependent. So is a file with an unresolved name that is the name of a
// field of such a path, because that field can now resolve it.
func ownerDependents(idx *storeIndex, results []goextract.FileResult, paths []string, inBatch map[string]bool) ([]string, error) {
	typeNames := make(map[string]bool)
	fieldNames := make(map[string]bool)
	byPath := make(map[string]*goextract.FileResult, len(results))
	for i := range results {
		byPath[results[i].RelPath] = &results[i]
	}
	// known holds the names of the stored types and of the types of
	// results. It is filled when the first path seems to change.
	var known map[string]bool
	for _, p := range paths {
		before, err := storedOwnerFacts(idx.r0, p)
		if err != nil {
			return nil, err
		}
		after := newOwnerFacts(byPath[p])
		if !before.equal(after) {
			if known == nil {
				if known, err = knownTypeNames(idx, results); err != nil {
					return nil, err
				}
			}
			before.keepKnownSupers(known)
			after.keepKnownSupers(known)
		}
		if before.equal(after) {
			continue
		}
		for _, f := range []ownerFacts{before, after} {
			for name := range f.types {
				typeNames[name] = true
			}
			for name := range f.supers {
				typeNames[name] = true
			}
			for _, field := range f.fields {
				fieldNames[field.name] = true
			}
		}
	}
	if len(typeNames) == 0 && len(fieldNames) == 0 {
		return nil, nil
	}

	if err := idx.load(); err != nil {
		return nil, err
	}
	types := make(map[string]bool)
	for name := range typeNames {
		for _, id := range idx.types[name] {
			types[id] = true
		}
	}

	for _, id := range sortedBoolKeys(types) {
		symbols.WalkSupertypes(id, idx.supers, func(s string) bool {
			types[s] = true
			return false
		})
	}

	set := make(map[string]bool)
	add := func(p string) {
		if p != "" && !inBatch[p] {
			set[p] = true
		}
	}
	for id := range types {
		for _, f := range idx.members[id] {
			for _, p := range idx.readers[f] {
				add(p)
			}
		}
	}
	if len(fieldNames) > 0 {
		skip := func(path string) bool { return inBatch[path] }
		match := func(u *schema.UnresolvedName) bool { return fieldNames[u.GetName()] }
		waiting, err := filesWithUnresolved(idx.r0, skip, match)
		if err != nil {
			return nil, err
		}
		for _, p := range waiting {
			add(p)
		}
	}

	return sortedBoolKeys(set), nil
}

// ownerFact is one field of a file: its name and its visibility.
type ownerFact struct{ name, visibility string }

// ownerFacts are the facts of one file that the owner chains of other
// files depend on: its types, the names of their supertypes, and its
// fields by id. Only the languages whose classes inherit fields count.
type ownerFacts struct {
	types, supers map[string]bool
	fields        map[string]ownerFact
}

func (a ownerFacts) equal(b ownerFacts) bool {
	return maps.Equal(a.types, b.types) && maps.Equal(a.supers, b.supers) && maps.Equal(a.fields, b.fields)
}

func (f *ownerFacts) add(n *schema.Node) {
	if !resolve.InheritsMembers(n.Language) {
		return
	}
	switch {
	case symbols.IsTypeKind(n.Kind):
		f.types[n.Name] = true
	case n.Kind == goextract.KindField:
		f.fields[n.Id] = ownerFact{n.Name, n.Visibility}
	}
}

// keepKnownSupers removes the supertypes whose names are not in known.
// The store has an edge only to a supertype that resolves, and the
// extraction names every supertype. A supertype that no indexed type
// can be, such as a library base class, is left out of both sides.
func (f *ownerFacts) keepKnownSupers(known map[string]bool) {
	maps.DeleteFunc(f.supers, func(name string, _ bool) bool { return !known[name] })
}

// knownTypeNames returns the names of the stored types and of the types
// of results, in the languages whose classes inherit members.
func knownTypeNames(idx *storeIndex, results []goextract.FileResult) (map[string]bool, error) {
	if err := idx.load(); err != nil {
		return nil, err
	}
	known := make(map[string]bool, len(idx.types))
	for name := range idx.types {
		known[name] = true
	}
	for i := range results {
		f := newOwnerFacts(&results[i])
		maps.Copy(known, f.types)
	}
	return known, nil
}

func emptyOwnerFacts() ownerFacts {
	return ownerFacts{types: map[string]bool{}, supers: map[string]bool{}, fields: map[string]ownerFact{}}
}

// newOwnerFacts returns the owner facts of a new extraction result. A
// supertype is the last part of the name that an embeds ref names.
func newOwnerFacts(r *goextract.FileResult) ownerFacts {
	f := emptyOwnerFacts()
	if r == nil {
		return f
	}
	for _, en := range r.Nodes {
		f.add(en.Node)
	}
	if len(f.types) > 0 {
		for _, ref := range r.Unresolved {
			if ref.Kind == goextract.RefKindEmbeds {
				f.supers[ref.Name[strings.LastIndex(ref.Name, ".")+1:]] = true
			}
		}
	}
	return f
}

// storedOwnerFacts returns the owner facts that r0 holds for path.
func storedOwnerFacts(r0 graphstore.Reader, path string) (ownerFacts, error) {
	f := emptyOwnerFacts()
	it, err := r0.IterateFileIndex(path)
	if err != nil {
		return f, err
	}
	var ids []string
	for it.Next() {
		if e := it.Entry(); e.IsNode {
			ids = append(ids, e.NodeID)
		}
	}
	err = it.Err()
	it.Close()
	if err != nil {
		return f, err
	}
	for _, id := range ids {
		n, err := r0.GetNode(id)
		if err != nil {
			return f, err
		}
		f.add(n)
		if !symbols.IsTypeKind(n.Kind) || !resolve.InheritsMembers(n.Language) {
			continue
		}
		eit, err := r0.IterateEdges(id)
		if err != nil {
			return f, err
		}
		var targets []string
		for eit.Next() {
			switch e := eit.Edge(); e.Kind {
			case goextract.RefKindEmbeds, goextract.EdgeKindExtends, goextract.EdgeKindImplements:
				targets = append(targets, e.Target)
			}
		}
		err = eit.Err()
		eit.Close()
		if err != nil {
			return f, err
		}
		for _, target := range targets {
			s, err := r0.GetNode(target)
			switch {
			case errors.Is(err, graphstore.ErrNotFound):
				continue
			case err != nil:
				return f, err
			}
			f.supers[s.Name] = true
		}
	}
	return f, nil
}
