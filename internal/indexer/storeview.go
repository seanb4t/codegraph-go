package indexer

import (
	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/dispatch"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// storeView is what a sync's resolution needs from the files it does not
// reparse, so that kind checks and implements, override and
// conformance-retry matching see the whole graph as a fresh index does.
type storeView struct {
	kinds, names     map[string]string
	startLines       map[string]int32
	arity            map[string]int32
	interfaceMethods dispatch.InterfaceSpecs

	// edges are the contains, embeds, extends and declared implements
	// edges that unchanged files own.
	edges []*schema.Edge

	// synthesized maps each structural implements or override edge an
	// unchanged file owns to that file. The sync synthesizes these again
	// over the whole graph and keeps, adds or deletes each one.
	synthesized map[edgeTriple]string
}

// loadStoreView reads r's nodes and edges, skipping what files in exclude
// own: the files the sync reparses or deleted.
func loadStoreView(r graphstore.Reader, exclude map[string]bool) (*storeView, error) {
	v := &storeView{
		kinds:            make(map[string]string),
		names:            make(map[string]string),
		startLines:       make(map[string]int32),
		arity:            make(map[string]int32),
		interfaceMethods: make(dispatch.InterfaceSpecs),
		synthesized:      make(map[edgeTriple]string),
	}

	fileOf := make(map[string]string)
	nit, err := r.IterateNodes()
	if err != nil {
		return nil, err
	}
	for nit.Next() {
		n := nit.Node()
		fileOf[n.Id] = n.FilePath
		if exclude[n.FilePath] || n.Kind == kindPackage {
			continue
		}
		v.kinds[n.Id] = n.Kind
		v.names[n.Id] = n.Name
		v.startLines[n.Id] = n.StartLine
		if n.Arity != 0 {
			v.arity[n.Id] = n.Arity
		}
		for _, m := range n.InterfaceMethods {
			v.interfaceMethods[n.Id] = append(v.interfaceMethods[n.Id], goextract.MethodSpec{Name: m.Name, Arity: m.Arity})
		}
	}
	err = nit.Err()
	nit.Close()
	if err != nil {
		return nil, err
	}

	eit, err := r.IterateEdges("")
	if err != nil {
		return nil, err
	}
	defer eit.Close()
	for eit.Next() {
		e := eit.Edge()
		owner, ok := fileOf[edgeOwnerID(e)]
		if !ok || exclude[owner] {
			continue
		}
		k := edgeTriple{e.Source, e.Kind, e.Target}
		switch {
		case isSynthesizedEdge(e):
			v.synthesized[k] = owner
		case e.Kind == "contains", e.Kind == "embeds", e.Kind == goextract.EdgeKindExtends, e.Kind == goextract.EdgeKindImplements:
			v.edges = append(v.edges, e)
		}
	}
	return v, eit.Err()
}

// isSynthesizedEdge reports whether e comes from structural implements or
// override synthesis, not from resolving a reference in its owner's file.
func isSynthesizedEdge(e *schema.Edge) bool {
	switch e.Kind {
	case goextract.EdgeKindOverrides:
		return true
	case goextract.EdgeKindImplements:
		return e.Metadata["synthesizedBy"] == dispatch.SynthesizedBy
	}
	return false
}
