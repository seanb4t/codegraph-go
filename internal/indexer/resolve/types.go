package resolve

import (
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// KindPackage is the synthetic node kind for an intra-module import target
// (RQ-1 recommendation (a)) — not one of goextract's declared node kinds,
// since no source declaration produces it; it exists purely so an
// `imports` edge has something in-repo to target.
const KindPackage = "package"

// TypeOutput is what Types resolves in one file: its nodes, its
// intra-file edges, the edges of every reference that is not a call, the
// package nodes its imports name, and its File record so far.
type TypeOutput struct {
	nodes, packageNodes []*schema.Node
	edges               []*schema.Edge
	// file is nil for a result without a path, such as the routes.
	file       *schema.File
	resolved   int
	unresolved int
}

// Edges returns the file's intra-file and type-level edges, the input of
// symbols.NewHierarchy.
func (o TypeOutput) Edges() []*schema.Edge { return o.edges }

// Types resolves every reference of r that is not a call against view:
// embeds, contains, imports, references, instantiates, returns and
// type-of. modulePath is the repo's own module path (from Discover); it
// classifies an imports target as intra-module, which gets a package
// node and an edge, or external, which gets neither. Types depends only
// on r and view, it changes neither, and it starts no goroutine. So
// several goroutines can call it at once over one view.
func Types(view symbols.View, modulePath string, r goextract.FileResult) TypeOutput {
	rs := newResolver(view, symbols.Hierarchy{})
	var out TypeOutput
	var waiting []*schema.UnresolvedName
	packageNodeIDs := make(map[string]struct{})
	for _, en := range r.Nodes {
		// The view holds the node that merges every copy of this id, so
		// the output carries that node and not the copy of this file.
		n := view.Node(en.Node.Id)
		if n == nil {
			n = en.Node
		}
		out.nodes = append(out.nodes, n)
	}
	for _, ie := range r.IntraEdges {
		out.edges = append(out.edges, ie.Edge)
	}
	wait := func(pkgAlias, name string) {
		if u := unresolvedName(r, pkgAlias, name); u != nil {
			waiting = append(waiting, u)
		}
	}
	for _, ref := range r.Unresolved {
		switch ref.Kind {
		case goextract.RefKindCalls:
			// Calls resolves the calls.

		case goextract.RefKindEmbeds:
			targetID, ok := rs.resolveRef(r, ref, true)
			if !ok {
				out.unresolved++
				wait(ref.PkgAlias, ref.Name)
				continue
			}
			// Pattern 2 (RES-02) + D-09 split (01-RESEARCH.md §B): a
			// class/struct extends/implements reference is
			// syntactically undistinguished at extraction time
			// (Java's implements/extends, C#'s comma-separated
			// base_list, Go's struct/interface embedding all emit
			// the same RefKindEmbeds shape). Now that the target is
			// resolved, this splits three ways:
			//   1. target is an interface AND source is NOT itself
			//      an interface -> promote to "implements"
			//      (unchanged from pre-D-09 behavior). A Go struct
			//      embedding an interface value
			//      (`type W struct { io.Reader }`) genuinely does
			//      promote that interface's methods onto the struct
			//      too (real Go semantics), so this rule applies
			//      uniformly across every language, not just Go.
			//   2. target is NOT an interface (a class/struct target
			//      — D-09's new case) -> "extends", distinct from
			//      the bare "embeds" string this branch used to fall
			//      through to unconditionally. This is the D-09
			//      addendum's answer to Open Question A2: Go's
			//      structural composition is the closest analog TS's
			//      `extends` RANK_EDGES kind has in Go, so a
			//      class/struct-extends-class/struct reference is
			//      reclassified rather than left as "embeds".
			//   3. target IS an interface but source is ALSO an
			//      interface (interface-embeds-interface) -> stays
			//      the plain "embeds" edge, unchanged — neither new
			//      branch's condition matches this case.
			kind := "embeds"
			provenance := "ast"
			var metadata map[string]string
			switch {
			case view.Kind(targetID) == goextract.KindInterface && view.Kind(ref.FromID) != goextract.KindInterface:
				kind = goextract.EdgeKindImplements
				provenance = "heuristic"
				metadata = map[string]string{"synthesizedBy": "declared-implements"}
			case view.Kind(targetID) != goextract.KindInterface:
				kind = goextract.EdgeKindExtends
				provenance = "heuristic"
				metadata = map[string]string{"synthesizedBy": "declared-extends"}
			}
			out.edges = append(out.edges, &schema.Edge{
				Source: ref.FromID, Target: targetID, Kind: kind,
				Line: ref.Line, Col: ref.Col, Provenance: provenance,
				Metadata: metadata,
			})
			out.resolved++

		case goextract.RefKindContains:
			// A method whose receiver type lives in a different file
			// (02-03's 4th UnresolvedRef kind). The receiver type is
			// always declared in the method's own package in valid
			// Go, so this is always an unqualified lookup.
			typeID, ok := rs.resolveRef(r, ref, true)
			if !ok {
				out.unresolved++
				wait("", ref.Name)
				continue
			}
			out.edges = append(out.edges, &schema.Edge{
				Source: typeID, Target: ref.FromID, Kind: "contains",
				Provenance: "ast",
			})
			out.resolved++

		case goextract.RefKindImports:
			if !isIntraModule(modulePath, ref.Name) {
				// External/stdlib import: no in-repo node to target,
				// and D-03a forbids inventing edges beyond ground
				// truth — no node, no edge (RQ-1 recommendation (a)).
				continue
			}
			pkgID := nodeid.NodeID(KindPackage, ref.Name, ref.Name)
			if _, seen := packageNodeIDs[pkgID]; !seen {
				packageNodeIDs[pkgID] = struct{}{}
				out.packageNodes = append(out.packageNodes, &schema.Node{
					Id:            pkgID,
					Kind:          KindPackage,
					Name:          lastPathSegment(ref.Name),
					QualifiedName: ref.Name,
				})
			}
			out.edges = append(out.edges, &schema.Edge{
				Source: ref.FromID, Target: pkgID, Kind: "imports",
				Line: ref.Line, Col: ref.Col, Provenance: "ast",
			})
			out.resolved++

		case goextract.RefKindReferences:
			// D-09 (01-RESEARCH.md §B): mirrors RefKindCalls' resolve
			// shape exactly — no Kind-check disambiguation on the
			// target (any resolvable symbol is a valid references
			// target, unlike instantiates/type_of/returns below).
			if rs.inheritedRef(r, ref) {
				continue
			}
			targetID, ok := rs.resolveDataRef(r, ref)
			if !ok {
				out.unresolved++
				wait(ref.PkgAlias, ref.Name)
				continue
			}
			metadata := rs.dataRefMetadata(r, ref)
			out.edges = append(out.edges, &schema.Edge{
				Source: ref.FromID, Target: targetID, Kind: goextract.RefKindReferences,
				Line: ref.Line, Col: ref.Col, Provenance: "ast", Metadata: metadata,
			})
			out.resolved++

		case goextract.RefKindInstantiates:
			// D-09 Kind-check disambiguation (RESEARCH §B): the
			// resolved target must actually be a type-Kind node a Go
			// composite literal can instantiate — a struct. (Go has
			// no "class" kind; an interface can never be
			// instantiated, and a type_alias's underlying type is not
			// resolved here, so it is intentionally excluded rather
			// than guessed at.) A resolved-but-wrong-Kind target
			// counts as unresolved — this is a real, deliberate
			// absence, not a silent drop of ground truth Pass 1 never
			// claimed in the first place.
			targetID, ok := rs.resolveRef(r, ref, true)
			if !ok || view.Kind(targetID) != goextract.KindStruct {
				out.unresolved++
				wait(ref.PkgAlias, ref.Name)
				continue
			}
			out.edges = append(out.edges, &schema.Edge{
				Source: ref.FromID, Target: targetID, Kind: goextract.RefKindInstantiates,
				Line: ref.Line, Col: ref.Col, Provenance: "ast",
			})
			out.resolved++

		case goextract.RefKindReturns, goextract.RefKindTypeOf:
			// D-09 (RESEARCH §B): both mirror RefKindCalls' resolve
			// shape (a declared type name resolved to its node), no
			// Kind-check disambiguation beyond "resolves to any
			// in-repo symbol at all" — a returns/type_of target may
			// legitimately be a struct, interface, or type_alias.
			targetID, ok := rs.resolveRef(r, ref, true)
			if !ok {
				out.unresolved++
				wait(ref.PkgAlias, ref.Name)
				continue
			}
			out.edges = append(out.edges, &schema.Edge{
				Source: ref.FromID, Target: targetID, Kind: ref.Kind,
				Line: ref.Line, Col: ref.Col, Provenance: "ast",
			})
			out.resolved++
		}
	}
	if r.RelPath != "" {
		out.file = &schema.File{
			Path:            r.RelPath,
			ContentHash:     r.ContentHash,
			Language:        r.Language,
			NodeCount:       int64(len(r.Nodes)),
			EdgeCount:       int64(len(r.IntraEdges) + out.resolved),
			MtimeUnixNs:     r.MtimeUnixNs,
			SizeBytes:       r.SizeBytes,
			UnresolvedNames: waiting,
			SyntaxErrors:    r.SyntaxErrors,
		}
		if r.ModuleKeyDeclared {
			out.file.DeclaredModuleKey = r.ImportPath
		}
		if r.Err != nil {
			out.file.Errors = []string{r.Err.Error()}
		}
	}
	return out
}
