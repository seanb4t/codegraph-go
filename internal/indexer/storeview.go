package indexer

import (
	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/discover"
	"github.com/seanb4t/codegraph-go/internal/indexer/dispatch"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/resolve"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// storeView is what a sync's resolution needs from the files it does not
// reparse, beyond the declarations loadStoreView adds to the Builder.
type storeView struct {
	// synthesized maps each structural implements, override or overload
	// set edge an unchanged file owns to that file. The sync synthesizes
	// these again over the whole graph and keeps, adds or deletes each
	// one.
	synthesized map[resolve.EdgeTriple]string
}

// loadStoreView reads r's files, nodes and edges once each, skipping what
// files in exclude own: the files the sync reparses or deleted, whose
// symbols the batch supersedes or removes. It also skips each node in
// shared, the shared ids of the cycle, and their edges without an owner
// file: every file that emits one of them is in the reparse. It adds what the other files
// declare to b, so that kind checks and implements, override and
// conformance-retry matching see the whole graph as a fresh index does.
//
// A stored symbol goes under the key that Builder.AddResults would give
// it: the module its file declares (File.declared_module_key) when it
// declares one, and otherwise the file's ImportPath in discovered, which
// its language's extract.LanguageSpec.ModuleKey computed on this run.
// Only the nodes that symbols.Reachable admits become symbols, as in
// AddResults, so a nested declaration, an unexported TypeScript
// declaration and an overload set stay out, and collisions route
// through the same symbols.Pick tie-break (WR-01).
func loadStoreView(r graphstore.Reader, discovered map[string]discover.DiscoveredFile, exclude map[string]bool, shared map[string]bool, b *symbols.Builder) (*storeView, error) {
	v := &storeView{synthesized: make(map[resolve.EdgeTriple]string)}

	declared := make(map[string]string)
	fit, err := r.IterateFiles()
	if err != nil {
		return nil, err
	}
	for fit.Next() {
		f := fit.File()
		if f.GetDeclaredModuleKey() != "" {
			declared[f.GetPath()] = f.GetDeclaredModuleKey()
		}
		if f.GetSyntaxErrors() && !exclude[f.GetPath()] {
			b.MarkSyntaxErrors(f.GetPath())
		}
	}
	err = fit.Err()
	fit.Close()
	if err != nil {
		return nil, err
	}

	fileOf := make(map[string]string)
	fileExports := make(map[string]bool)
	moduleOf := make(map[string]string)
	var stored []*schema.Node
	nit, err := r.IterateNodes()
	if err != nil {
		return nil, err
	}
	for nit.Next() {
		n := nit.Node()
		fileOf[n.Id] = n.FilePath
		if exclude[n.FilePath] || n.Kind == resolve.KindPackage {
			continue
		}
		if shared[n.Id] {
			// The merged node of the reparse replaces the stored one.
			continue
		}
		b.AddNode(n)
		stored = append(stored, n)
		if n.Arity != 0 {
			b.SetArity(n.Id, n.Arity)
		}
		if n.IsExported {
			fileExports[n.FilePath] = true
		}
		moduleKey, ok := declared[n.FilePath]
		if !ok {
			moduleKey = discovered[n.FilePath].ImportPath
		}
		moduleOf[n.FilePath] = moduleKey
		b.SetModule(n.FilePath, moduleKey)
	}
	err = nit.Err()
	nit.Close()
	if err != nil {
		return nil, err
	}
	for _, n := range stored {
		// A stored overload set is not a candidate: the resolver declares
		// sets again from their members, as a fresh index does.
		// A field is reached through its owner only, never by bare name.
		if n.Kind == goextract.KindFile || n.Kind == goextract.KindOverloadSet || n.Kind == goextract.KindField {
			continue
		}
		if symbols.Reachable(n, fileExports[n.FilePath]) {
			b.Add(moduleOf[n.FilePath], n)
		}
	}

	storedSets := make(map[string]symbols.StoredSet)
	eit, err := r.IterateEdges("")
	if err != nil {
		return nil, err
	}
	defer eit.Close()
	for eit.Next() {
		e := eit.Edge()
		owner, ok := fileOf[resolve.EdgeOwnerID(e)]
		if !ok || exclude[owner] {
			continue
		}
		if shared[resolve.EdgeOwnerID(e)] {
			// An edge that a shared id of the cycle owns: a base edge
			// from it, or a contains edge into it. The sync deletes it
			// and stages the ones that still hold. A stale one must not
			// reach the hierarchy or the whole-graph passes: the base
			// edge of a part that is gone would make an override that a
			// full index does not have.
			continue
		}
		k := resolve.EdgeTriple{Source: e.Source, Kind: e.Kind, Target: e.Target}
		switch {
		case isSynthesizedEdge(e):
			v.synthesized[k] = owner
			if e.Kind == "contains" {
				recordSetEdge(b, storedSets, e)
			}
		case resolve.IsBindingEdge(e):
			b.AddStoredEdge(e)
		case e.Kind == "contains", e.Kind == "embeds", e.Kind == goextract.EdgeKindExtends, e.Kind == goextract.EdgeKindImplements:
			b.AddStoredEdge(e)
			if e.Kind == "contains" {
				if member := b.Node(e.Target); member != nil && b.Kind(e.Source) != goextract.KindFile && b.Kind(e.Source) != "" {
					b.AddMember(e.Source, member)
				}
			}
		}
	}
	if err := eit.Err(); err != nil {
		return nil, err
	}
	for id, set := range storedSets {
		b.AddStoredSet(id, set)
		if set.Owner == "" || b.Kind(set.Owner) == goextract.KindFile {
			continue
		}
		for _, m := range set.Members {
			if member := b.Node(m); member != nil {
				b.AddMember(set.Owner, member)
			}
		}
	}
	return v, nil
}

// recordSetEdge reads one stored overload set edge: owner to set, or set
// to member.
func recordSetEdge(b *symbols.Builder, sets map[string]symbols.StoredSet, e *schema.Edge) {
	switch {
	case b.Kind(e.Target) == goextract.KindOverloadSet:
		set := sets[e.Target]
		set.Owner = e.Source
		sets[e.Target] = set
	case b.Kind(e.Source) == goextract.KindOverloadSet:
		set := sets[e.Source]
		set.Members = append(set.Members, e.Target)
		sets[e.Source] = set
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
		return e.Metadata["synthesizedBy"] == resolve.SynthesizedBySet
	}
	return false
}
