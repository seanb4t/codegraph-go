// Package indexer's resolve.go implements Pass 2 of the two-pass indexing
// pipeline (D-04): build a global symbol index over every Pass-1 result,
// resolve calls/imports/embeds/contains references into ground-truth
// edges, deterministically collapse duplicate edges (D-05), and commit the
// whole resolved graph through exactly one batched GraphStore.Writer
// (D-04a).
package indexer

import (
	"sort"
	"strings"
	"time"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/dispatch"
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
	return resolveRefsWithIndex(results, modulePath, newSymbolIndex(results))
}

// resolveRefsWithIndex is resolveRefs' implementation, parameterized on
// the symbol index (Phase 4 D-01/RESEARCH Pattern 1 step 11): resolveRefs
// keeps building its own from-scratch newSymbolIndex(results) unconditio-
// nally; Sync() instead injects a store-seeded index overlaid with the
// reparse batch (newSymbolIndexFromStore + symbolIndex.overlay), so an
// unqualified/qualified reference into an UNCHANGED file still resolves
// (RESEARCH Pitfall 1). Every other line here is reused verbatim by both
// callers.
func resolveRefsWithIndex(results []goextract.FileResult, modulePath string, idx *symbolIndex) (nodes, packageNodes []*schema.Node, edges []*schema.Edge, files []*schema.File, unresolvedCount int) {
	packageNodeIDs := make(map[string]struct{})

	// nodeKindByID/nodeNameByID are GLOBAL (cross-file) node-id lookups,
	// built BEFORE any ref is resolved (Phase 5 RES-02 Pattern 2/3): the
	// RefKindEmbeds branch below needs to know a resolved target's Kind
	// (interface vs. not) to decide whether to promote the edge to
	// "implements" — and that target may be declared in a DIFFERENT file
	// than the one currently being resolved, exactly like
	// TestResolve_CrossFileMethodContainment's cross-file struct/method
	// pair. synthesizeGoImplements (below) reuses nodeNameByID for O(1)
	// method-name lookups instead of a per-edge linear scan over every
	// result's Nodes. This is a cheap, separate pass over results' own
	// Nodes; it does not touch the `nodes`/`edges` slices this function
	// accumulates incrementally below.
	nodeKindByID := make(map[string]string)
	nodeNameByID := make(map[string]string)
	nodeStartLineByID := make(map[string]int32)
	// methodArity is a GLOBAL (cross-file) methodID -> declared-parameter-
	// count lookup, aggregated once here (D-09/RESEARCH §B) rather than
	// re-derived separately by both synthesizeGoImplements (which already
	// builds its own copy from r.MethodArity) and synthesizeOverrides
	// (which needs the identical data to structurally match a method
	// against its supertype's same-named method by name+arity).
	methodArity := make(map[string]int32)
	for _, r := range results {
		for _, en := range r.Nodes {
			nodeKindByID[en.Node.Id] = en.Node.Kind
			nodeNameByID[en.Node.Id] = en.Node.Name
			nodeStartLineByID[en.Node.Id] = en.Node.StartLine
		}
		for id, arity := range r.MethodArity {
			methodArity[id] = arity
		}
	}

	// pending accumulates every RefKindCalls ref that fails pass 1 —
	// Pitfall 3's conformance retry (Task 2) re-attempts these AFTER
	// implements/extends edges exist for the whole graph, walking the
	// supertype chain, rather than interleaving the retry into this same
	// loop. pc.file is filled in once this file's own *schema.File is
	// built below (files holds pointers, so mutating pc.file.EdgeCount
	// after appending to files is safe and visible).
	var pending []pendingCall

	for _, r := range results {
		for _, en := range r.Nodes {
			nodes = append(nodes, en.Node)
		}
		for _, ie := range r.IntraEdges {
			edges = append(edges, ie.Edge)
		}

		resolvedForFile := 0
		var filePending []goextract.UnresolvedRef
		for _, ref := range r.Unresolved {
			switch ref.Kind {
			case goextract.RefKindCalls:
				calleeID, ok := resolveNameRef(idx, r, ref.PkgAlias, ref.Name)
				if !ok {
					// Deferred, not counted as unresolved yet (Pitfall 3):
					// this call may still resolve via the conformance
					// retry below, once the calling method's enclosing
					// type's supertype chain is known.
					filePending = append(filePending, ref)
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
				case nodeKindByID[targetID] == goextract.KindInterface && nodeKindByID[ref.FromID] != goextract.KindInterface:
					kind = goextract.EdgeKindImplements
					provenance = "heuristic"
					metadata = map[string]string{"synthesizedBy": "declared-implements"}
				case nodeKindByID[targetID] != goextract.KindInterface:
					kind = goextract.EdgeKindExtends
					provenance = "heuristic"
					metadata = map[string]string{"synthesizedBy": "declared-extends"}
				}
				edges = append(edges, &schema.Edge{
					Source: ref.FromID, Target: targetID, Kind: kind,
					Line: ref.Line, Col: ref.Col, Provenance: provenance,
					Metadata: metadata,
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

			case goextract.RefKindReferences:
				// D-09 (01-RESEARCH.md §B): mirrors RefKindCalls' resolve
				// shape exactly — no Kind-check disambiguation on the
				// target (any resolvable symbol is a valid references
				// target, unlike instantiates/type_of/returns below).
				targetID, ok := resolveNameRef(idx, r, ref.PkgAlias, ref.Name)
				if !ok {
					unresolvedCount++
					continue
				}
				edges = append(edges, &schema.Edge{
					Source: ref.FromID, Target: targetID, Kind: goextract.RefKindReferences,
					Line: ref.Line, Col: ref.Col, Provenance: "ast",
				})
				resolvedForFile++

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
				targetID, ok := resolveNameRef(idx, r, ref.PkgAlias, ref.Name)
				if !ok || nodeKindByID[targetID] != goextract.KindStruct {
					unresolvedCount++
					continue
				}
				edges = append(edges, &schema.Edge{
					Source: ref.FromID, Target: targetID, Kind: goextract.RefKindInstantiates,
					Line: ref.Line, Col: ref.Col, Provenance: "ast",
				})
				resolvedForFile++

			case goextract.RefKindReturns, goextract.RefKindTypeOf:
				// D-09 (RESEARCH §B): both mirror RefKindCalls' resolve
				// shape (a declared type name resolved to its node), no
				// Kind-check disambiguation beyond "resolves to any
				// in-repo symbol at all" — a returns/type_of target may
				// legitimately be a struct, interface, or type_alias.
				targetID, ok := resolveNameRef(idx, r, ref.PkgAlias, ref.Name)
				if !ok {
					unresolvedCount++
					continue
				}
				edges = append(edges, &schema.Edge{
					Source: ref.FromID, Target: targetID, Kind: ref.Kind,
					Line: ref.Line, Col: ref.Col, Provenance: "ast",
				})
				resolvedForFile++
			}
		}

		// RelPath == "" marks a synthetic, non-file result (Phase 5 LANG-07:
		// pipeline.go's Run appends exactly one such result carrying every
		// detected route's nodes/IntraEdges, via detectRoutes) — it
		// contributes Nodes/IntraEdges above like any other result, but
		// mints NO schema.File record, since it has no real source file to
		// attribute node/edge counts to (a synthetic "" path would collide
		// with nothing but would still be a misleading phantom File entry
		// in every file-listing surface).
		if r.RelPath == "" {
			continue
		}

		f := &schema.File{
			Path:        r.RelPath,
			ContentHash: r.ContentHash,
			Language:    r.Language,
			NodeCount:   int64(len(r.Nodes)),
			EdgeCount:   int64(len(r.IntraEdges) + resolvedForFile),
			MtimeUnixNs: r.MtimeUnixNs,
			SizeBytes:   r.SizeBytes,
		}
		if r.Err != nil {
			f.Errors = []string{r.Err.Error()}
		}
		files = append(files, f)

		for _, ref := range filePending {
			pending = append(pending, pendingCall{ref: ref, file: f})
		}
	}

	edges = append(edges, synthesizeGoImplements(results, edges, nodeKindByID, nodeNameByID, nodeStartLineByID)...)

	// D-09's overrides Pass-2 synthesis (01-RESEARCH.md §B): runs AFTER
	// synthesizeGoImplements so a structurally-synthesized "implements"
	// edge is ALSO available as a supertype edge to walk — though in
	// practice interface method specs never have their own "contains"
	// (type->method) edge (only concrete struct methods do), so overrides
	// only ever fires across "embeds"/"extends" supertype edges today; see
	// synthesizeOverrides' doc comment.
	edges = append(edges, synthesizeOverrides(edges, nodeKindByID, nodeNameByID, methodArity, nodeStartLineByID)...)

	// Pitfall 3's conformance retry (Task 2): now that every "contains"
	// (type->method) and "embeds"/"implements" (type->supertype) edge
	// exists for the WHOLE graph — including this file's own
	// synthesized implements edges above — retry every call that failed
	// pass 1, walking the supertype chain from the calling method's own
	// enclosing type. This is a SEPARATE loop over the deferred refs, not
	// interleaved into the resolution loop above (Pitfall 3's explicit
	// requirement).
	edges = append(edges, retryConformanceCalls(pending, edges, nodeNameByID, &unresolvedCount)...)

	return nodes, packageNodes, edges, files, unresolvedCount
}

// pendingCall is a RefKindCalls ref that failed pass-1 resolution,
// deferred to the conformance retry (Pitfall 3), paired with the
// *schema.File its resolvedForFile/EdgeCount bookkeeping belongs to — a
// retry success increments file.EdgeCount exactly like a pass-1
// resolution would have.
type pendingCall struct {
	ref  goextract.UnresolvedRef
	file *schema.File
}

// retryConformanceCalls implements Pitfall 3's two-pass conformance
// retry: for each pending call, find the calling method's OWN enclosing
// type (via a "contains" edge, type->method) and walk that type's
// supertype chain (via "embeds"/"implements" edges, bounded by a
// visited-set-guarded BFS — cycle-safe, independent of graph width)
// looking for a method sharing the call's bare name. A call whose
// FromID has no enclosing type at all (e.g. a plain function, not a
// method — TestResolve_UnresolvedMethodCall's local-variable-receiver
// case) never resolves here, exactly matching pass 1's own behavior for
// that case. unresolvedCount is incremented for exactly those refs that
// fail BOTH passes — never double-counted (pass 1 no longer increments
// it for a deferred call).
func retryConformanceCalls(pending []pendingCall, edges []*schema.Edge, nodeNameByID map[string]string, unresolvedCount *int) []*schema.Edge {
	typeMethods := make(map[string]map[string]string) // typeID -> methodName -> methodID
	methodOwner := make(map[string]string)             // methodID -> typeID
	typeSupertypes := make(map[string][]string)        // typeID -> []supertypeID

	for _, e := range edges {
		switch e.Kind {
		case "contains":
			methods := typeMethods[e.Source]
			if methods == nil {
				methods = make(map[string]string)
				typeMethods[e.Source] = methods
			}
			methods[nodeNameByID[e.Target]] = e.Target
			methodOwner[e.Target] = e.Source
		case "embeds", goextract.EdgeKindImplements, goextract.EdgeKindExtends:
			// D-09: a class/struct-extends-class/struct RefKindEmbeds ref
			// now resolves to "extends" (not "embeds" — see the
			// RefKindEmbeds case above), so the conformance retry's
			// supertype walk must recognize "extends" as a type->
			// supertype edge too, or an inherited call through an
			// extends chain would silently stop resolving.
			typeSupertypes[e.Source] = append(typeSupertypes[e.Source], e.Target)
		}
	}

	var resolved []*schema.Edge
	for _, pc := range pending {
		ownerType, ok := methodOwner[pc.ref.FromID]
		var targetID string
		if ok {
			targetID, ok = walkSupertypesForMethod(ownerType, pc.ref.Name, typeMethods, typeSupertypes)
		}
		if !ok {
			*unresolvedCount++
			continue
		}
		resolved = append(resolved, &schema.Edge{
			Source: pc.ref.FromID, Target: targetID, Kind: "calls",
			Line: pc.ref.Line, Col: pc.ref.Col, Provenance: "ast",
		})
		pc.file.EdgeCount++
	}
	return resolved
}

// walkSupertypes performs a visited-set-guarded BFS from typeID's OWN
// supertypes (never typeID itself), invoking visit on each reachable
// supertype id in discovery order and stopping as soon as visit returns
// true. Bounded by the number of distinct types reachable through the
// supertype graph — cycle-safe regardless of malformed/cyclic
// embeds/extends/implements data. This is the shared BFS primitive both
// walkSupertypesForMethod (calls edge resolution, Pitfall 3) and
// synthesizeOverrides (D-09's overrides synthesis) build on, rather than
// each writing its own traversal.
func walkSupertypes(typeID string, typeSupertypes map[string][]string, visit func(supertypeID string) bool) bool {
	visited := map[string]bool{typeID: true}
	queue := append([]string(nil), typeSupertypes[typeID]...)
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if visited[cur] {
			continue
		}
		visited[cur] = true
		if visit(cur) {
			return true
		}
		queue = append(queue, typeSupertypes[cur]...)
	}
	return false
}

// walkSupertypesForMethod looks for methodName in typeMethods among
// typeID's own supertypes (never typeID itself — pass 1 already
// attempted the calling type's own method set via resolveUnqualified),
// returning the first match's node id. See walkSupertypes for the
// underlying BFS.
func walkSupertypesForMethod(typeID, methodName string, typeMethods map[string]map[string]string, typeSupertypes map[string][]string) (string, bool) {
	var found string
	ok := walkSupertypes(typeID, typeSupertypes, func(cur string) bool {
		if id, exists := typeMethods[cur][methodName]; exists {
			found = id
			return true
		}
		return false
	})
	return found, ok
}

// synthesizeOverrides implements D-09's overrides Pass-2 synthesis
// (01-RESEARCH.md §B): for every type with a supertype (via "embeds",
// EdgeKindImplements, or EdgeKindExtends — every kind resolve.go already
// treats as a type->supertype relationship), find each of the type's OWN
// methods (via the already-built "contains" type->method edges) that
// shares a name+arity with a method declared directly on one of its
// supertypes (walked transitively via walkSupertypes, mirroring
// retryConformanceCalls' supertype-walk shape rather than writing a new
// BFS), and emit an EdgeKindOverrides edge method -> supertype-method.
//
// Go has no `override` keyword: this is structural (name+arity) matching
// — a documented precision note per RESEARCH §B, not a drop. A Go method
// that happens to share a supertype method's name+arity is treated as an
// override even without an explicit language marker, exactly like Go's
// implicit interface satisfaction (dispatch.SynthesizeImplements) already
// works. In practice this only ever matches through "embeds"/"extends"
// supertype edges: an interface's own method signatures never get a real
// "contains" (type->method) edge (collectInterfaceMethods only records
// them as MethodSpecs for structural implements matching), so
// typeMethods[interfaceID] is always empty and an implements-only
// supertype chain never yields an override match.
func synthesizeOverrides(edges []*schema.Edge, nodeKindByID, nodeNameByID map[string]string, methodArity map[string]int32, nodeStartLineByID map[string]int32) []*schema.Edge {
	typeMethods := make(map[string]map[string]string) // typeID -> methodName -> methodID
	typeSupertypes := make(map[string][]string)        // typeID -> []supertypeID

	for _, e := range edges {
		switch e.Kind {
		case "contains":
			if nodeKindByID[e.Target] != goextract.KindMethod {
				continue
			}
			methods := typeMethods[e.Source]
			if methods == nil {
				methods = make(map[string]string)
				typeMethods[e.Source] = methods
			}
			methods[nodeNameByID[e.Target]] = e.Target
		case "embeds", goextract.EdgeKindImplements, goextract.EdgeKindExtends:
			typeSupertypes[e.Source] = append(typeSupertypes[e.Source], e.Target)
		}
	}

	// Deterministic iteration (D-05/D-09): map order is randomized, so
	// sort both the type ids and each type's own method names before
	// walking, ensuring the emitted edge slice's order never depends on
	// Go's map iteration order.
	typeIDs := make([]string, 0, len(typeMethods))
	for typeID := range typeMethods {
		typeIDs = append(typeIDs, typeID)
	}
	sort.Strings(typeIDs)

	var synthesized []*schema.Edge
	for _, typeID := range typeIDs {
		names := make([]string, 0, len(typeMethods[typeID]))
		for name := range typeMethods[typeID] {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			methodID := typeMethods[typeID][name]
			arity := methodArity[methodID]

			var superMethodID string
			walkSupertypes(typeID, typeSupertypes, func(cur string) bool {
				if id, ok := typeMethods[cur][name]; ok && methodArity[id] == arity {
					superMethodID = id
					return true
				}
				return false
			})
			if superMethodID == "" {
				continue
			}
			synthesized = append(synthesized, &schema.Edge{
				Source: methodID, Target: superMethodID, Kind: goextract.EdgeKindOverrides,
				Provenance: "heuristic", Line: nodeStartLineByID[methodID],
				Metadata: map[string]string{"synthesizedBy": "structural-override"},
			})
		}
	}
	return synthesized
}

// synthesizeGoImplements wires dispatch.SynthesizeImplements (RES-02
// Pattern 3, Go's structural method-set match) into Pass 2: it composes a
// struct node id -> method-spec-set map from the fully-resolved "contains"
// edges (a struct's method set may span multiple files, exactly like
// TestResolve_CrossFileMethodContainment) and an interface node id ->
// method-spec-set map directly from every FileResult's own
// InterfaceMethods, then hands both — plus an interface-embeds-interface
// adjacency derived from the already-resolved "embeds" edges — to
// dispatch.SynthesizeImplements. The returned edges are appended to the
// SAME edges slice this function already builds, so they flow through
// collapseEdges exactly like every other edge (D-06/D-07: no parallel
// dedup path, additive within SchemaVersion 1).
func synthesizeGoImplements(results []goextract.FileResult, edges []*schema.Edge, nodeKindByID, nodeNameByID map[string]string, nodeStartLineByID map[string]int32) []*schema.Edge {
	methodArity := make(map[string]int32)
	interfaceMethods := make(dispatch.InterfaceSpecs)
	for _, r := range results {
		for id, arity := range r.MethodArity {
			methodArity[id] = arity
		}
		for ifaceID, specs := range r.InterfaceMethods {
			interfaceMethods[ifaceID] = append(interfaceMethods[ifaceID], specs...)
		}
	}

	structMethods := make(dispatch.TypeMethods)
	interfaceEmbeds := make(map[string][]string)
	for _, e := range edges {
		switch e.Kind {
		case "contains":
			if nodeKindByID[e.Source] != goextract.KindStruct || nodeKindByID[e.Target] != goextract.KindMethod {
				continue
			}
			structMethods[e.Source] = append(structMethods[e.Source], goextract.MethodSpec{
				Name:  nodeNameByID[e.Target],
				Arity: methodArity[e.Target],
			})
		case "embeds":
			if nodeKindByID[e.Source] == goextract.KindInterface && nodeKindByID[e.Target] == goextract.KindInterface {
				interfaceEmbeds[e.Source] = append(interfaceEmbeds[e.Source], e.Target)
			}
		}
	}

	synthesized := dispatch.SynthesizeImplements(structMethods, interfaceMethods, interfaceEmbeds)
	// dispatch.SynthesizeImplements has no source-location data of its
	// own (it operates purely on name/arity sets) — anchor each
	// synthesized edge's Line at the implementing struct's OWN
	// declaration line (RES-03's "where a source location exists"
	// qualifier: unlike a declared `implements X` clause, Go's implicit
	// satisfaction has no single syntactic reference site, so the
	// struct's own declaration is the closest meaningful anchor).
	for _, e := range synthesized {
		e.Line = nodeStartLineByID[e.Source]
	}
	return synthesized
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

// Resolve runs the whole of Pass 2: builds the global symbol index,
// resolves every file's Unresolved references into edges, deterministically
// collapses duplicates, and commits the resolved graph through exactly one
// batched GraphStore.Writer (D-04a). It returns the count of references
// that could not be resolved (surfaced later via --verbose, never silently
// dropped — D-06a).
func Resolve(store graphstore.GraphStore, results []goextract.FileResult, modulePath string) (int, error) {
	nodes, packageNodes, edges, files, unresolvedCount := resolveRefs(results, modulePath)
	if err := writeGraph(store, nodes, packageNodes, edges, files); err != nil {
		return unresolvedCount, err
	}
	return unresolvedCount, nil
}

// edgeTriple is the (source, kind, target) identity a duplicate edge
// collapses on (D-05) — mirrors graphstore/keys.go's edgeKey shape exactly,
// without needing to import that package's unexported key-building code.
type edgeTriple struct {
	source, kind, target string
}

// collapseEdges aggregates every candidate edge sharing a (source, kind,
// target) triple, chooses ONE representative by sorting the candidates by
// a TOTAL ORDER — (filePath, line, col), via nodeFilePath[candidate.Source]
// — and takes the first (RESEARCH Pitfall 1). This is never "whichever
// candidate was appended first/last" (processing order); the same input
// set, in any order, always yields the same representative. The returned
// slice is itself in sorted triple order, so staging it via PutEdge is
// also deterministic.
func collapseEdges(edges []*schema.Edge, nodeFilePath map[string]string) []*schema.Edge {
	groups := make(map[edgeTriple][]*schema.Edge, len(edges))
	for _, e := range edges {
		k := edgeTriple{e.Source, e.Kind, e.Target}
		groups[k] = append(groups[k], e)
	}

	keys := make([]edgeTriple, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].source != keys[j].source {
			return keys[i].source < keys[j].source
		}
		if keys[i].kind != keys[j].kind {
			return keys[i].kind < keys[j].kind
		}
		return keys[i].target < keys[j].target
	})

	collapsed := make([]*schema.Edge, 0, len(keys))
	for _, k := range keys {
		candidates := groups[k]
		sort.Slice(candidates, func(i, j int) bool {
			ci, cj := candidates[i], candidates[j]
			fi, fj := nodeFilePath[ci.Source], nodeFilePath[cj.Source]
			if fi != fj {
				return fi < fj
			}
			if ci.Line != cj.Line {
				return ci.Line < cj.Line
			}
			return ci.Col < cj.Col
		})
		collapsed = append(collapsed, candidates[0])
	}
	return collapsed
}

// writeGraph collapses edges deterministically and stages the whole
// resolved graph — package pseudo-nodes and symbol nodes (sorted by id),
// then files (sorted by path), then collapsed edges (sorted by
// source/kind/target) — through exactly one GraphStore.Writer, committing
// once (D-04a). Any staging error releases the batch via Close() (never a
// partial Commit) and returns the error.
func writeGraph(store graphstore.GraphStore, nodes, packageNodes []*schema.Node, edges []*schema.Edge, files []*schema.File) error {
	nodeFilePath := make(map[string]string, len(nodes))
	for _, n := range nodes {
		nodeFilePath[n.Id] = n.FilePath
	}

	allNodes := make([]*schema.Node, 0, len(nodes)+len(packageNodes))
	allNodes = append(allNodes, nodes...)
	allNodes = append(allNodes, packageNodes...)
	sort.Slice(allNodes, func(i, j int) bool { return allNodes[i].Id < allNodes[j].Id })

	sortedFiles := make([]*schema.File, len(files))
	copy(sortedFiles, files)
	sort.Slice(sortedFiles, func(i, j int) bool { return sortedFiles[i].Path < sortedFiles[j].Path })

	collapsedEdges := collapseEdges(edges, nodeFilePath)

	w, err := store.NewWriter()
	if err != nil {
		return err
	}

	for _, n := range allNodes {
		if err := w.PutNode(n); err != nil {
			w.Close()
			return err
		}
	}
	for _, f := range sortedFiles {
		if err := w.PutFile(f); err != nil {
			w.Close()
			return err
		}
	}
	for _, e := range collapsedEdges {
		if err := w.PutEdge(e, nodeFilePath[e.Source]); err != nil {
			w.Close()
			return err
		}
	}

	meta := schema.NewMeta()
	meta.NodeCount = int64(len(allNodes))
	meta.EdgeCount = int64(len(collapsedEdges))
	// Phase 4 D-02b: every PutNode/PutEdge this Writer stages already
	// populates the x/ file-owned secondary index unconditionally (04-01)
	// — so any graph committed through writeGraph genuinely HAS a
	// complete x/ index by the time this Commit lands, regardless of
	// whether the caller was a from-scratch Run or Sync. Stamping the
	// flag here (not just in Sync's own backfill path) means only a
	// GENUINELY pre-Phase-4 store (built before this field/namespace
	// existed) is ever missing it.
	meta.HasFileIndex = true
	// Phase 4 D-04a: stamp LastSyncUnixMs here too, not just in Sync's own
	// meta-write step (internal/indexer/sync.go) — otherwise a graph built
	// via a from-scratch `codegraph index` (this path) carries a zero
	// last_sync_unix_ms forever until the first incremental sync runs,
	// which would make query.Engine.Status's newest-mtime-vs-last_sync
	// staleness fallback (04-06) report every freshly indexed repo as
	// permanently stale.
	meta.LastSyncUnixMs = time.Now().UnixMilli()
	if err := w.PutMeta(meta); err != nil {
		w.Close()
		return err
	}

	return w.Commit()
}
