// Package indexer's resolve.go implements Pass 2 of the two-pass indexing
// pipeline (D-04): build a global symbol index over every Pass-1 result,
// resolve calls/imports/embeds/contains references into ground-truth
// edges, deterministically collapse duplicate edges (D-05), and commit the
// whole resolved graph through exactly one batched GraphStore.Writer
// (D-04a).
package indexer

import (
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// kindPackage is the synthetic node kind for an intra-module import target
// (RQ-1 recommendation (a)) — not one of goextract's declared node kinds,
// since no source declaration produces it; it exists purely so an
// `imports` edge has something in-repo to target.
const kindPackage = "package"

// resolveRefs builds the global symbol index over results and settles
// every file's Unresolved references into ground-truth edges. It returns:
//   - nodes: every symbol/file node Pass 1 already extracted, unchanged
//   - packageNodes: the synthetic "package" pseudo-nodes minted for
//     intra-module imports (RQ-1), deduplicated by id
//   - edges: every intra-file edge Pass 1 already extracted, PLUS every
//     successfully resolved calls/imports/embeds/contains edge
//   - files: one schema.File per Pass-1 result
//   - unresolvedCount: how many Unresolved refs could NOT be resolved
//     (never silently dropped — D-06a; surfaced later via --verbose)
//
// modulePath is the repo's own module path (from Discover), used to
// classify an "imports" ref's target as intra-module (gets a package
// pseudo-node + edge) vs. external/stdlib (no node, no edge).
func resolveRefs(results []goextract.FileResult, modulePath string) (nodes, packageNodes []*schema.Node, edges []*schema.Edge, files []*schema.File, unresolvedCount int) {
	idx := newSymbolIndex(results)
	packageNodeIDs := make(map[string]struct{})

	for _, r := range results {
		for _, en := range r.Nodes {
			nodes = append(nodes, en.Node)
		}
		for _, ie := range r.IntraEdges {
			edges = append(edges, ie.Edge)
		}

		resolvedForFile := 0
		for _, ref := range r.Unresolved {
			switch ref.Kind {
			case goextract.RefKindCalls:
				calleeID, ok := resolveNameRef(idx, r, ref.PkgAlias, ref.Name)
				if !ok {
					unresolvedCount++
					continue
				}
				edges = append(edges, &schema.Edge{
					Source: ref.FromID, Target: calleeID, Kind: "calls",
					Line: ref.Line, Col: ref.Col, Provenance: "ast",
				})
				resolvedForFile++

			case goextract.RefKindEmbeds:
				targetID, ok := resolveNameRef(idx, r, ref.PkgAlias, ref.Name)
				if !ok {
					unresolvedCount++
					continue
				}
				edges = append(edges, &schema.Edge{
					Source: ref.FromID, Target: targetID, Kind: "embeds",
					Line: ref.Line, Col: ref.Col, Provenance: "ast",
				})
				resolvedForFile++

			case goextract.RefKindContains:
				// A method whose receiver type lives in a different file
				// (02-03's 4th UnresolvedRef kind). The receiver type is
				// always declared in the method's own package in valid
				// Go, so this is always an unqualified lookup.
				typeID, ok := idx.resolveUnqualified(r.ImportPath, ref.Name)
				if !ok {
					unresolvedCount++
					continue
				}
				edges = append(edges, &schema.Edge{
					Source: typeID, Target: ref.FromID, Kind: "contains",
					Provenance: "ast",
				})
				resolvedForFile++

			case goextract.RefKindImports:
				if !isIntraModule(modulePath, ref.Name) {
					// External/stdlib import: no in-repo node to target,
					// and D-03a forbids inventing edges beyond ground
					// truth — no node, no edge (RQ-1 recommendation (a)).
					continue
				}
				pkgID := nodeid.NodeID(kindPackage, ref.Name, ref.Name)
				if _, seen := packageNodeIDs[pkgID]; !seen {
					packageNodeIDs[pkgID] = struct{}{}
					packageNodes = append(packageNodes, &schema.Node{
						Id:            pkgID,
						Kind:          kindPackage,
						Name:          lastPathSegment(ref.Name),
						QualifiedName: ref.Name,
					})
				}
				edges = append(edges, &schema.Edge{
					Source: ref.FromID, Target: pkgID, Kind: "imports",
					Line: ref.Line, Col: ref.Col, Provenance: "ast",
				})
				resolvedForFile++
			}
		}

		f := &schema.File{
			Path:        r.RelPath,
			ContentHash: r.ContentHash,
			Language:    r.Language,
			NodeCount:   int64(len(r.Nodes)),
			EdgeCount:   int64(len(r.IntraEdges) + resolvedForFile),
		}
		if r.Err != nil {
			f.Errors = []string{r.Err.Error()}
		}
		files = append(files, f)
	}

	return nodes, packageNodes, edges, files, unresolvedCount
}

// resolveNameRef resolves a (pkgAlias, name) reference against idx: an
// empty pkgAlias is an unqualified, same-package reference; a non-empty
// pkgAlias must be a real import alias in r's Imports map (RQ-2's
// narrowest-safe-set boundary — resolveSelector's own alias check is what
// keeps a local-variable-receiver call like `w.Describe()` from ever
// resolving here, since "w" is never a key in r.Imports).
func resolveNameRef(idx *symbolIndex, r goextract.FileResult, pkgAlias, name string) (string, bool) {
	if pkgAlias == "" {
		return idx.resolveUnqualified(r.ImportPath, name)
	}
	return idx.resolveSelector(r.Imports, pkgAlias, name)
}

// isIntraModule reports whether importPath belongs to the same module as
// modulePath (either the module root itself or a subpackage of it).
func isIntraModule(modulePath, importPath string) bool {
	if importPath == modulePath {
		return true
	}
	return strings.HasPrefix(importPath, modulePath+"/")
}

// lastPathSegment returns the final "/"-delimited segment of importPath,
// the conventional default package name TS-parity behavior expects on the
// synthetic package pseudo-node (RQ-1).
func lastPathSegment(importPath string) string {
	if i := strings.LastIndexByte(importPath, '/'); i >= 0 {
		return importPath[i+1:]
	}
	return importPath
}
