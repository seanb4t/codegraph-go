package resolve

import (
	"cmp"
	"slices"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/dispatch"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
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

// schemaMethodSpecs converts an interface's extracted method specs to
// their stored form, or nil when there are none.
func schemaMethodSpecs(specs []goextract.MethodSpec) []*schema.MethodSpec {
	if len(specs) == 0 {
		return nil
	}
	out := make([]*schema.MethodSpec, len(specs))
	for i, s := range specs {
		out[i] = &schema.MethodSpec{Name: s.Name, Arity: s.Arity}
	}
	return out
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
func retryConformanceCalls(pending []pendingCall, edges []*schema.Edge, view symbols.View, unresolvedCount *int) []*schema.Edge {
	typeMethods := make(map[string]map[string]string) // typeID -> methodName -> methodID
	methodOwner := make(map[string]string)            // methodID -> typeID
	typeSupertypes := make(map[string][]string)       // typeID -> []supertypeID

	for _, e := range edges {
		switch e.Kind {
		case "contains":
			methods := typeMethods[e.Source]
			if methods == nil {
				methods = make(map[string]string)
				typeMethods[e.Source] = methods
			}
			methods[view.Name(e.Target)] = e.Target
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
			if pc.wait != nil {
				pc.file.UnresolvedNames = append(pc.file.UnresolvedNames, pc.wait)
			}
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

// walkSupertypesForMethod looks for methodName in typeMethods among
// typeID's own supertypes (never typeID itself: the resolver already
// tried the calling type's own method set),
// returning the first match's node id. See symbols.WalkSupertypes for the
// underlying BFS.
func walkSupertypesForMethod(typeID, methodName string, typeMethods map[string]map[string]string, typeSupertypes map[string][]string) (string, bool) {
	var found string
	ok := symbols.WalkSupertypes(typeID, typeSupertypes, func(cur string) bool {
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
//
// Overloads pair by signature key (parameter types) first; a supertype
// method with the same name and arity pairs only when it is the one
// such method, so two same-arity overloads never cross-link. methodKey
// holds the signature key of each function or method node.
func synthesizeOverrides(view symbols.View, edges []*schema.Edge, methodKey map[string]string) []*schema.Edge {
	// typeMethods is typeID -> methodName -> ALL methodIDs sharing that
	// name (CR-01 fix): a bare map[string]string here would collapse
	// same-name/different-arity overloads — legal and common in Java/C#
	// (impossible in Go, hence this can't be exercised through the Go
	// extractor; see TestResolveOverrides_OverloadedMethodsBothSynthesized
	// in resolve_test.go, which drives this function directly) — to a
	// single arbitrary candidate on BOTH the self side (line ~503 below)
	// and the supertype side (the walkSupertypes closure), silently
	// dropping or misattributing overrides edges for the discarded
	// overload(s).
	typeMethods := make(map[string]map[string][]string) // typeID -> methodName -> []methodID (all overloads)
	typeSupertypes := make(map[string][]string)         // typeID -> []supertypeID

	for _, e := range edges {
		switch e.Kind {
		case "contains":
			if view.Kind(e.Target) != goextract.KindMethod {
				continue
			}
			methods := typeMethods[e.Source]
			if methods == nil {
				methods = make(map[string][]string)
				typeMethods[e.Source] = methods
			}
			name := view.Name(e.Target)
			methods[name] = append(methods[name], e.Target)
		case "embeds", goextract.EdgeKindImplements, goextract.EdgeKindExtends:
			typeSupertypes[e.Source] = append(typeSupertypes[e.Source], e.Target)
		}
	}

	// Deterministic iteration (D-05/D-09): map order is randomized, so
	// sort the type ids, each type's own method names, AND (CR-01) each
	// name's own overload-id slice before walking, ensuring the emitted
	// edge slice's order never depends on Go's map iteration order or
	// edges' append order.
	typeIDs := make([]string, 0, len(typeMethods))
	for typeID := range typeMethods {
		typeIDs = append(typeIDs, typeID)
		for _, ids := range typeMethods[typeID] {
			sort.Strings(ids)
		}
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
			for _, methodID := range typeMethods[typeID][name] {
				arity := view.Arity(methodID)
				key := methodKey[methodID]

				var superMethodID string
				symbols.WalkSupertypes(typeID, typeSupertypes, func(cur string) bool {
					var byArity []string
					for _, candidateID := range typeMethods[cur][name] {
						if key != "" && methodKey[candidateID] == key {
							superMethodID = candidateID
							return true
						}
						if view.Arity(candidateID) == arity {
							byArity = append(byArity, candidateID)
						}
					}
					if len(byArity) == 1 {
						superMethodID = byArity[0]
						return true
					}
					return false
				})
				if superMethodID == "" {
					continue
				}
				synthesized = append(synthesized, &schema.Edge{
					Source: methodID, Target: superMethodID, Kind: goextract.EdgeKindOverrides,
					Provenance: "heuristic", Line: view.StartLine(methodID),
					Metadata: map[string]string{"synthesizedBy": "structural-override"},
				})
			}
		}
	}
	return synthesized
}

// synthesizeGoImplements wires dispatch.SynthesizeImplements (RES-02
// Pattern 3, Go's structural method-set match) into Pass 2: it composes a
// struct node id -> method-spec-set map from the fully-resolved "contains"
// edges (a struct's method set may span multiple files, exactly like
// TestResolve_CrossFileMethodContainment) and takes interfaceMethods, the
// interface node id -> method-spec-set map the caller built from every
// FileResult's own InterfaceMethods (and, in a sync, the store), then
// hands both — plus an interface-embeds-interface
// adjacency derived from the already-resolved "embeds" edges — to
// dispatch.SynthesizeImplements. The returned edges are appended to the
// SAME edges slice this function already builds, so they flow through
// collapseEdges exactly like every other edge (D-06/D-07: no parallel
// dedup path, additive within SchemaVersion 1).
func synthesizeGoImplements(view symbols.View, edges []*schema.Edge) []*schema.Edge {
	structMethods := make(dispatch.TypeMethods)
	interfaceEmbeds := make(map[string][]string)
	for _, e := range edges {
		switch e.Kind {
		case "contains":
			if view.Kind(e.Source) != goextract.KindStruct || view.Kind(e.Target) != goextract.KindMethod {
				continue
			}
			structMethods[e.Source] = append(structMethods[e.Source], goextract.MethodSpec{
				Name:  view.Name(e.Target),
				Arity: view.Arity(e.Target),
			})
		case "embeds":
			if view.Kind(e.Source) == goextract.KindInterface && view.Kind(e.Target) == goextract.KindInterface {
				interfaceEmbeds[e.Source] = append(interfaceEmbeds[e.Source], e.Target)
			}
		}
	}

	synthesized := dispatch.SynthesizeImplements(structMethods, view.InterfaceMethods(), interfaceEmbeds)
	// dispatch.SynthesizeImplements has no source-location data of its
	// own (it operates purely on name/arity sets) — anchor each
	// synthesized edge's Line at the implementing struct's OWN
	// declaration line (RES-03's "where a source location exists"
	// qualifier: unlike a declared `implements X` clause, Go's implicit
	// satisfaction has no single syntactic reference site, so the
	// struct's own declaration is the closest meaningful anchor).
	for _, e := range synthesized {
		e.Line = view.StartLine(e.Source)
	}
	return synthesized
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
