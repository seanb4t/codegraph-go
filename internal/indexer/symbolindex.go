package indexer

import (
	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// symbolIndex is the global (moduleKey, declaredName) -> nodeID index Pass
// 2 builds from every Pass-1 result before resolving any cross-file
// reference — the "global symbol index" RES-01/D-04 call for. moduleKey is
// a per-language cross-file symbol-index key (Phase 5 D-04/Pitfall 2): Go's
// import path is the first instance of this concept (still carried on
// goextract.FileResult.ImportPath — the field name is unchanged, only its
// semantic source generalizes per language via LanguageSpec.ModuleKey).
type symbolIndex struct {
	// byModuleKeyAndName is keyed moduleKey -> declaredName -> nodeID.
	// declaredName is a node's bare Name field: for functions, types,
	// constants, and variables this is the plain identifier; for a method
	// it is the bare method name (NOT "Recv.Method" — that lives in
	// QualifiedName), matching how goextract records call-site references
	// (UnresolvedRef.Name is always the bare identifier text at the call
	// site, never receiver-qualified).
	byModuleKeyAndName map[string]map[string]string

	// Collisions counts every same-(moduleKey, name) collision addSymbol
	// resolved deterministically (WR-01) instead of silently
	// last-write-wins overwriting — surfaced here so a collision is never
	// silently dropped (D-06a), even though this plan's file scope does
	// not yet wire it through to Sync's Stats/CLI --verbose output.
	Collisions int
}

// newSymbolIndex builds the index by ranging results in the order given.
// Pass 1's results are already index-addressed in a stable, pre-sorted file
// order (RESEARCH Pattern 2) — this function never ranges a Go map to
// decide insertion order, so the index itself introduces no
// nondeterminism. A skipped file (FileResult.Err != nil) contributes no
// symbols, matching goextract's own "no nodes for a skipped file"
// contract.
func newSymbolIndex(results []goextract.FileResult) *symbolIndex {
	idx := &symbolIndex{byModuleKeyAndName: make(map[string]map[string]string, len(results))}
	idx.overlay(results)
	return idx
}

// overlay adds results' own declared symbols into idx via addSymbol — the
// shared per-result loop both newSymbolIndex (fresh index) and Sync's
// store-seeded index (layering the reparse batch on top) use, so the two
// never drift (Phase 4 RESEARCH Pattern 1 step 10).
func (idx *symbolIndex) overlay(results []goextract.FileResult) {
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		names := idx.byModuleKeyAndName[r.ImportPath]
		if names == nil {
			names = make(map[string]string)
			idx.byModuleKeyAndName[r.ImportPath] = names
		}
		for _, en := range r.Nodes {
			if en.Node.Kind == goextract.KindFile {
				// A file node is never a callable/embeddable symbol
				// target; excluding it also avoids a same-module
				// collision between a file's synthetic name (its
				// RelPath) and a genuinely declared symbol.
				continue
			}
			idx.addSymbol(names, en.Node.Name, en.Node.Id)
		}
	}
}

// addSymbol inserts (name -> id) into names, the moduleKey-scoped bucket
// addSymbol's caller looked up/created. A same-(moduleKey, name) collision
// (WR-01 — e.g. two files in the same Go package both declaring a
// same-named func/method) is resolved deterministically: the candidate
// with the lexicographically lowest node Id wins, mirroring
// query.Engine.resolveSymbolNode's own lowest-Id tie-break
// (internal/query/traverse.go) — so the resolved edge target is stable
// across runs regardless of file-processing order. Every genuine collision
// increments idx.Collisions; re-inserting the identical id (e.g. re-overlay
// of an unchanged file during Sync) is not a collision.
func (idx *symbolIndex) addSymbol(names map[string]string, name, id string) {
	existing, ok := names[name]
	if !ok {
		names[name] = id
		return
	}
	if existing == id {
		return
	}
	idx.Collisions++
	if id < existing {
		names[name] = id
	}
}

// newSymbolIndexFromStore seeds a symbolIndex from the graph already
// committed to r — every node's moduleKey is recomputed via
// importPathFor(modulePath, node.FilePath) rather than read off a
// goextract.FileResult, since the store holds no FileResult (Phase 4
// RESEARCH Pattern 1 step 10 / Pitfall 1). Nodes belonging to a path in
// exclude are skipped entirely — the caller (Sync) is about to supersede
// or has already removed those files' symbols and overlays fresh ones on
// top, so any store-seeded entry for them would be stale. Collisions
// discovered here route through the SAME addSymbol tie-break as overlay
// (WR-01), so Sync's store-seeded index and a from-scratch newSymbolIndex
// never disagree on which node id wins a same-(moduleKey, name) collision.
//
// goextract.KindFile and kindPackage nodes are skipped, mirroring
// newSymbolIndex's own exclusions: a file node is never a callable
// target, and a package pseudo-node has no declaring source to seed a
// (name -> id) entry from in this scan (it is re-minted, not looked up,
// by resolveRefsWithIndex's imports-ref branch).
func newSymbolIndexFromStore(r graphstore.Reader, modulePath string, exclude map[string]bool) (*symbolIndex, error) {
	idx := &symbolIndex{byModuleKeyAndName: make(map[string]map[string]string)}

	it, err := r.IterateNodes()
	if err != nil {
		return nil, err
	}
	defer it.Close()

	for it.Next() {
		n := it.Node()
		if n.Kind == goextract.KindFile || n.Kind == kindPackage || exclude[n.FilePath] {
			continue
		}
		moduleKey := importPathFor(modulePath, n.FilePath)
		names := idx.byModuleKeyAndName[moduleKey]
		if names == nil {
			names = make(map[string]string)
			idx.byModuleKeyAndName[moduleKey] = names
		}
		idx.addSymbol(names, n.Name, n.Id)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	return idx, nil
}

// resolveSelector resolves a package-qualified reference (pkg.Name) using
// the calling file's own Imports map (RESEARCH §Import Resolution):
// pkgAlias must be a real import alias present in callerImports — an alias
// that is NOT a key there is not a package reference at all (e.g. a local
// variable receiver in `w.Describe()`), and correctly never resolves here.
// This is what implements RQ-2's narrowest-safe-set boundary: resolve.go
// never attempts local-variable type tracking, so a method call through a
// variable simply falls through to "unresolved" via this same check. This
// narrowest-safe-set boundary is preserved verbatim (unchanged this plan)
// as the pattern every future per-language selector resolver follows.
func (idx *symbolIndex) resolveSelector(callerImports map[string]string, pkgAlias, name string) (string, bool) {
	moduleKey, ok := callerImports[pkgAlias]
	if !ok {
		return "", false
	}
	names, ok := idx.byModuleKeyAndName[moduleKey]
	if !ok {
		return "", false
	}
	id, ok := names[name]
	return id, ok
}

// resolveUnqualified resolves an unqualified reference against the calling
// file's OWN moduleKey — any file sharing a module/package/namespace may
// reference any other file's module-level symbol, so the lookup is scoped
// by moduleKey, not by the specific file.
func (idx *symbolIndex) resolveUnqualified(callerModuleKey, name string) (string, bool) {
	names, ok := idx.byModuleKeyAndName[callerModuleKey]
	if !ok {
		return "", false
	}
	id, ok := names[name]
	return id, ok
}
