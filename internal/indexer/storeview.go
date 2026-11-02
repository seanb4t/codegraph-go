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
	// symbols is the store-seeded symbol index (Phase 4 RESEARCH Pattern 1
	// step 10 / Pitfall 1): Sync overlays the batch on it, so a reference
	// into an unchanged file still resolves. A stored symbol sits under
	// the key overlay would give it: the module its file declares
	// (File.declared_module_key) when it declares one, and otherwise the
	// file's ImportPath in discovered, which its language's
	// LanguageSpec.ModuleKey computed on this run. File and package nodes
	// are left out, as newSymbolIndex leaves them out, and collisions
	// route through the same pickSymbol tie-break (WR-01).
	symbols *symbolIndex

	kinds, names     map[string]string
	nodes            map[string]*schema.Node
	startLines       map[string]int32
	arity            map[string]int32
	interfaceMethods dispatch.InterfaceSpecs

	// edges are the contains, embeds, extends and declared implements
	// edges that unchanged files own.
	edges []*schema.Edge

	// synthesized maps each structural implements, override or overload
	// set edge an unchanged file owns to that file. The sync synthesizes
	// these again over the whole graph and keeps, adds or deletes each
	// one.
	synthesized map[edgeTriple]string

	// storedSets are the overload sets the unchanged files hold, by set
	// id: the resolver puts their members under the set's owner, and
	// gives the members of a set that dissolved back to it.
	storedSets map[string]storedSet

	// moduleOf maps each unchanged file to its module key, for the
	// deferred types of design D17: a return type resolves in the
	// callee's module.
	moduleOf map[string]string
}

// loadStoreView reads r's files, nodes and edges once each, skipping what
// files in exclude own: the files the sync reparses or deleted, whose
// symbols the batch supersedes or removes.
func loadStoreView(r graphstore.Reader, discovered map[string]DiscoveredFile, exclude map[string]bool) (*storeView, error) {
	v := &storeView{
		symbols:          newEmptySymbolIndex(),
		kinds:            make(map[string]string),
		names:            make(map[string]string),
		nodes:            make(map[string]*schema.Node),
		startLines:       make(map[string]int32),
		arity:            make(map[string]int32),
		interfaceMethods: make(dispatch.InterfaceSpecs),
		synthesized:      make(map[edgeTriple]string),
		storedSets:       make(map[string]storedSet),
		moduleOf:         make(map[string]string),
	}

	declared := make(map[string]string)
	fit, err := r.IterateFiles()
	if err != nil {
		return nil, err
	}
	for fit.Next() {
		if f := fit.File(); f.GetDeclaredModuleKey() != "" {
			declared[f.GetPath()] = f.GetDeclaredModuleKey()
		}
	}
	err = fit.Err()
	fit.Close()
	if err != nil {
		return nil, err
	}

	fileOf := make(map[string]string)
	fileExports := make(map[string]bool)
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
		v.nodes[n.Id] = n
		v.startLines[n.Id] = n.StartLine
		if n.Arity != 0 {
			v.arity[n.Id] = n.Arity
		}
		if n.IsExported {
			fileExports[n.FilePath] = true
		}
		for _, m := range n.InterfaceMethods {
			v.interfaceMethods[n.Id] = append(v.interfaceMethods[n.Id], goextract.MethodSpec{Name: m.Name, Arity: m.Arity})
		}
		moduleKey, ok := declared[n.FilePath]
		if !ok {
			moduleKey = discovered[n.FilePath].ImportPath
		}
		v.moduleOf[n.FilePath] = moduleKey
	}
	err = nit.Err()
	nit.Close()
	if err != nil {
		return nil, err
	}
	for _, n := range v.nodes {
		if n.Kind != goextract.KindFile && reachable(n, fileExports[n.FilePath]) {
			v.symbols.add(v.moduleOf[n.FilePath], n)
		}
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
			if e.Kind == "contains" {
				v.recordSetEdge(e)
			}
		case e.Kind == "contains", e.Kind == "embeds", e.Kind == goextract.EdgeKindExtends, e.Kind == goextract.EdgeKindImplements:
			v.edges = append(v.edges, e)
			if e.Kind == "contains" {
				if member, ok := v.nodes[e.Target]; ok && v.kinds[e.Source] != goextract.KindFile && v.kinds[e.Source] != "" {
					v.symbols.addMember(e.Source, member)
				}
			}
		}
	}
	if err := eit.Err(); err != nil {
		return nil, err
	}
	for _, set := range v.storedSets {
		if set.owner == "" || v.kinds[set.owner] == goextract.KindFile {
			continue
		}
		for _, m := range set.members {
			if member, ok := v.nodes[m]; ok {
				v.symbols.addMember(set.owner, member)
			}
		}
	}
	return v, nil
}

// recordSetEdge reads one stored overload set edge: owner to set, or set
// to member.
func (v *storeView) recordSetEdge(e *schema.Edge) {
	switch {
	case v.kinds[e.Target] == goextract.KindOverloadSet:
		set := v.storedSets[e.Target]
		set.owner = e.Source
		v.storedSets[e.Target] = set
	case v.kinds[e.Source] == goextract.KindOverloadSet:
		set := v.storedSets[e.Source]
		set.members = append(set.members, e.Target)
		v.storedSets[e.Source] = set
	}
}

// isSynthesizedEdge reports whether e comes from structural implements or
// override synthesis, not from resolving a reference in its owner's file.
func isSynthesizedEdge(e *schema.Edge) bool {
	switch e.Kind {
	case goextract.EdgeKindOverrides:
		return true
	case goextract.EdgeKindImplements:
		return e.Metadata["synthesizedBy"] == dispatch.SynthesizedBy
	case goextract.RefKindContains:
		return e.Metadata["synthesizedBy"] == SynthesizedBySet
	}
	return false
}
