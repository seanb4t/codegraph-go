package indexer

import "github.com/seanb4t/codegraph-go/internal/indexer/goextract"

// symbolIndex is the global (importPath, declaredName) -> nodeID index Pass
// 2 builds from every Pass-1 result before resolving any cross-file
// reference — the "global symbol index" RES-01/D-04 call for.
type symbolIndex struct {
	// byImportAndName is keyed importPath -> declaredName -> nodeID.
	// declaredName is a node's bare Name field: for functions, types,
	// constants, and variables this is the plain identifier; for a method
	// it is the bare method name (NOT "Recv.Method" — that lives in
	// QualifiedName), matching how goextract records call-site references
	// (UnresolvedRef.Name is always the bare identifier text at the call
	// site, never receiver-qualified).
	byImportAndName map[string]map[string]string
}

// newSymbolIndex builds the index by ranging results in the order given.
// Pass 1's results are already index-addressed in a stable, pre-sorted file
// order (RESEARCH Pattern 2) — this function never ranges a Go map to
// decide insertion order, so the index itself introduces no
// nondeterminism. A skipped file (FileResult.Err != nil) contributes no
// symbols, matching goextract's own "no nodes for a skipped file"
// contract.
func newSymbolIndex(results []goextract.FileResult) *symbolIndex {
	idx := &symbolIndex{byImportAndName: make(map[string]map[string]string, len(results))}
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		names := idx.byImportAndName[r.ImportPath]
		if names == nil {
			names = make(map[string]string)
			idx.byImportAndName[r.ImportPath] = names
		}
		for _, en := range r.Nodes {
			if en.Node.Kind == goextract.KindFile {
				// A file node is never a callable/embeddable symbol
				// target; excluding it also avoids a same-package
				// collision between a file's synthetic name (its
				// RelPath) and a genuinely declared symbol.
				continue
			}
			names[en.Node.Name] = en.Node.Id
		}
	}
	return idx
}

// resolveSelector resolves a package-qualified reference (pkg.Name) using
// the calling file's own Imports map (RESEARCH §Import Resolution):
// pkgAlias must be a real import alias present in callerImports — an alias
// that is NOT a key there is not a package reference at all (e.g. a local
// variable receiver in `w.Describe()`), and correctly never resolves here.
// This is what implements RQ-2's narrowest-safe-set boundary: resolve.go
// never attempts local-variable type tracking, so a method call through a
// variable simply falls through to "unresolved" via this same check.
func (idx *symbolIndex) resolveSelector(callerImports map[string]string, pkgAlias, name string) (string, bool) {
	importPath, ok := callerImports[pkgAlias]
	if !ok {
		return "", false
	}
	names, ok := idx.byImportAndName[importPath]
	if !ok {
		return "", false
	}
	id, ok := names[name]
	return id, ok
}

// resolveUnqualified resolves an unqualified reference against the calling
// file's OWN import path — any file in a Go package may reference any
// other file's package-level symbol, so the lookup is scoped by import
// path, not by the specific file.
func (idx *symbolIndex) resolveUnqualified(callerImportPath, name string) (string, bool) {
	names, ok := idx.byImportAndName[callerImportPath]
	if !ok {
		return "", false
	}
	id, ok := names[name]
	return id, ok
}
