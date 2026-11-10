package resolve

import (
	"cmp"
	"slices"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// callEdge is the calls edge of ref to target, with the call-site data
// the resolver decided from (D13). An ambiguous call is marked.
func callEdge(ref goextract.UnresolvedRef, target string, ambiguous bool) *schema.Edge {
	e := &schema.Edge{
		Source: ref.FromID, Target: target, Kind: goextract.RefKindCalls,
		Line: ref.Line, Col: ref.Col, Provenance: "ast",
		ReceiverType: ref.ReceiverType, ArgCount: ref.ArgCount, HasSpread: ref.HasSpread,
		ArgTypes: ref.ArgTypes, ArgLabels: ref.ArgLabels,
	}
	if ambiguous {
		e.Metadata = map[string]string{"ambiguous": "true"}
	}
	return e
}

func unresolvedName(r goextract.FileResult, pkgAlias, name string) *schema.UnresolvedName {
	if pkgAlias == "" {
		return &schema.UnresolvedName{ModuleKey: r.ImportPath, Name: name}
	}
	moduleKey, ok := r.Imports[pkgAlias]
	if !ok {
		return nil
	}
	return &schema.UnresolvedName{ModuleKey: moduleKey, Name: name}
}

// sortedUniqueNames sorts names by (module key, name) and drops repeats,
// so a File record's bytes do not depend on reference order.
func sortedUniqueNames(names []*schema.UnresolvedName) []*schema.UnresolvedName {
	if len(names) == 0 {
		return nil
	}
	compare := func(a, b *schema.UnresolvedName) int {
		return cmp.Or(cmp.Compare(a.ModuleKey, b.ModuleKey), cmp.Compare(a.Name, b.Name))
	}
	slices.SortFunc(names, compare)
	return slices.CompactFunc(names, func(a, b *schema.UnresolvedName) bool { return compare(a, b) == 0 })
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
// the conventional default package name for the synthetic package
// pseudo-node (RQ-1).
func lastPathSegment(importPath string) string {
	if i := strings.LastIndexByte(importPath, '/'); i >= 0 {
		return importPath[i+1:]
	}
	return importPath
}
