package indexer

import (
	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/indexer/resolve"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// storeIndex holds what bindingDependents and ownerDependents read from
// every stored node and edge of the sync snapshot r0. One scan of the
// nodes and one of the edges fill it on first use. r0 is a point-in-time
// snapshot that a sync never writes, so one index serves every wave.
type storeIndex struct {
	r0     graphstore.Reader
	loaded bool

	// pathOf maps a node id to the path of its file.
	pathOf map[string]string
	// data are the stored data symbols.
	data []dataNode
	// types maps a name to the ids of the types of that name in the
	// languages whose classes inherit members.
	types map[string][]string
	// binding are the stored binding edges.
	binding []*schema.Edge
	// readers maps a data symbol id to the paths of the files that refer
	// to it, one entry per references edge.
	readers map[string][]string
	// supers maps a type id to the targets of its embeds, extends and
	// implements edges.
	supers map[string][]string
	// members maps a type id to the ids of the fields it contains.
	members map[string][]string
}

// dataNode is one stored data symbol.
type dataNode struct{ id, name, path string }

func newStoreIndex(r0 graphstore.Reader) *storeIndex {
	return &storeIndex{r0: r0}
}

// load fills the index from r0 unless an earlier call did.
func (x *storeIndex) load() error {
	if x.loaded {
		return nil
	}
	x.pathOf = make(map[string]string)
	x.types = make(map[string][]string)
	x.readers = make(map[string][]string)
	x.supers = make(map[string][]string)
	x.members = make(map[string][]string)

	nit, err := x.r0.IterateNodes()
	if err != nil {
		return err
	}
	for nit.Next() {
		n := nit.Node()
		x.pathOf[n.Id] = n.FilePath
		switch {
		case isDataID(n.Id):
			x.data = append(x.data, dataNode{n.Id, n.Name, n.FilePath})
		case symbols.IsTypeKind(n.Kind) && resolve.InheritsMembers(n.Language):
			x.types[n.Name] = append(x.types[n.Name], n.Id)
		}
	}
	err = nit.Err()
	nit.Close()
	if err != nil {
		return err
	}

	eit, err := x.r0.IterateEdges("")
	if err != nil {
		return err
	}
	for eit.Next() {
		e := eit.Edge()
		switch e.Kind {
		case goextract.RefKindEmbeds, goextract.EdgeKindExtends, goextract.EdgeKindImplements:
			x.supers[e.Source] = append(x.supers[e.Source], e.Target)
		case goextract.RefKindContains:
			if nodeid.KindOf(e.Target) == goextract.KindField {
				x.members[e.Source] = append(x.members[e.Source], e.Target)
			}
		case goextract.RefKindReferences:
			if resolve.IsBindingEdge(e) {
				x.binding = append(x.binding, e)
			}
			if isDataID(e.Target) {
				x.readers[e.Target] = append(x.readers[e.Target], x.pathOf[e.Source])
			}
		}
	}
	err = eit.Err()
	eit.Close()
	if err != nil {
		return err
	}
	x.loaded = true
	return nil
}
