package indexer

import (
	"errors"
	"maps"
	"sort"
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
// change counts: its types, their supertypes, or its fields and their
// visibility. The affected types are the types of such a path, before
// and after, and the supertypes they name, with all their stored
// ancestors. A file that refers to a field of an affected type is a
// dependent. So is a file with an unresolved name that is the name of a
// field of such a path, because that field can now resolve it.
func ownerDependents(r0 graphstore.Reader, results []goextract.FileResult, paths []string, inBatch map[string]bool) ([]string, error) {
	typeNames := make(map[string]bool)
	fieldNames := make(map[string]bool)
	byPath := make(map[string]*goextract.FileResult, len(results))
	for i := range results {
		byPath[results[i].RelPath] = &results[i]
	}
	for _, p := range paths {
		before, err := storedOwnerFacts(r0, p)
		if err != nil {
			return nil, err
		}
		after := newOwnerFacts(byPath[p])
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

	types := make(map[string]bool)
	pathOf := make(map[string]string)
	nit, err := r0.IterateNodes()
	if err != nil {
		return nil, err
	}
	for nit.Next() {
		n := nit.Node()
		pathOf[n.Id] = n.FilePath
		if typeNames[n.Name] && symbols.IsTypeKind(n.Kind) && resolve.InheritsMembers(n.Language) {
			types[n.Id] = true
		}
	}
	err = nit.Err()
	nit.Close()
	if err != nil {
		return nil, err
	}

	supers := make(map[string][]string)
	members := make(map[string][]string)
	readers := make(map[string][]string)
	eit, err := r0.IterateEdges("")
	if err != nil {
		return nil, err
	}
	for eit.Next() {
		e := eit.Edge()
		switch e.Kind {
		case goextract.RefKindEmbeds, goextract.EdgeKindExtends, goextract.EdgeKindImplements:
			supers[e.Source] = append(supers[e.Source], e.Target)
		case goextract.RefKindContains:
			if strings.HasPrefix(e.Target, goextract.KindField+":") {
				members[e.Source] = append(members[e.Source], e.Target)
			}
		case goextract.RefKindReferences:
			if strings.HasPrefix(e.Target, goextract.KindField+":") {
				readers[e.Target] = append(readers[e.Target], pathOf[e.Source])
			}
		}
	}
	err = eit.Err()
	eit.Close()
	if err != nil {
		return nil, err
	}

	queue := make([]string, 0, len(types))
	for id := range types {
		queue = append(queue, id)
	}
	for len(queue) > 0 {
		id := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		for _, s := range supers[id] {
			if !types[s] {
				types[s] = true
				queue = append(queue, s)
			}
		}
	}

	set := make(map[string]bool)
	add := func(p string) {
		if p != "" && !inBatch[p] {
			set[p] = true
		}
	}
	for id := range types {
		for _, f := range members[id] {
			for _, p := range readers[f] {
				add(p)
			}
		}
	}
	if len(fieldNames) > 0 {
		fit, err := r0.IterateFiles()
		if err != nil {
			return nil, err
		}
		for fit.Next() {
			f := fit.File()
			for _, u := range f.GetUnresolvedNames() {
				if fieldNames[u.GetName()] {
					add(f.GetPath())
					break
				}
			}
		}
		err = fit.Err()
		fit.Close()
		if err != nil {
			return nil, err
		}
	}

	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
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
