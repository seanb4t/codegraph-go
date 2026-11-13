package resolve

import (
	"fmt"
	"sort"

	"github.com/seanb4t/codegraph-go/internal/indexer/dispatch"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// IsStructural reports whether e is an edge that the passes over the whole
// graph read: a contains edge, or a supertype edge (embeds, implements or
// extends). Every other edge matters only to its own file.
func IsStructural(e *schema.Edge) bool {
	switch e.Kind {
	case goextract.RefKindContains, goextract.RefKindEmbeds, goextract.EdgeKindImplements, goextract.EdgeKindExtends:
		return true
	}
	return false
}

// StructuralEdges returns the structural edges of edges, in order.
func StructuralEdges(edges []*schema.Edge) []*schema.Edge {
	var out []*schema.Edge
	for _, e := range edges {
		if IsStructural(e) {
			out = append(out, e)
		}
	}
	return out
}

// tables are the two tables that the passes over the whole graph read.
// newTables fills them from one walk of the structural edges.
type tables struct {
	// members is the member index: for each source of a contains edge, the
	// targets in edge order. The source is a type for a member, but the
	// index keeps every source, a file node included, as the passes of
	// before did.
	members map[string][]string
	// owner maps a member to the source of the last contains edge that
	// targets it.
	owner map[string]string
	// supers is the supertype table: for each type, its supertype edges in
	// edge order, each with its kind.
	supers map[string][]supertypeEdge
}

type supertypeEdge struct {
	target, kind string
}

// newTables reads edges, then stored, into the tables. It keeps the two
// lists apart in supers, because the synthesized implements edges go
// between them (addOverlay).
func newTables(edges, stored []*schema.Edge) (t tables, storedSupers map[string][]supertypeEdge) {
	t = tables{
		members: make(map[string][]string),
		owner:   make(map[string]string),
		supers:  make(map[string][]supertypeEdge),
	}
	storedSupers = make(map[string][]supertypeEdge)
	read := func(list []*schema.Edge, supers map[string][]supertypeEdge) {
		for _, e := range list {
			switch e.Kind {
			case goextract.RefKindContains:
				t.members[e.Source] = append(t.members[e.Source], e.Target)
				t.owner[e.Target] = e.Source
			case goextract.RefKindEmbeds, goextract.EdgeKindImplements, goextract.EdgeKindExtends:
				supers[e.Source] = append(supers[e.Source], supertypeEdge{e.Target, e.Kind})
			}
		}
	}
	read(edges, t.supers)
	read(stored, storedSupers)
	return t, storedSupers
}

// supertypes returns the supertype ids of each type: the supertype edges
// of the results, then overlay, then the stored supertype edges. That is
// the order in which the passes of before met them.
func (t tables) supertypes(overlay []*schema.Edge, storedSupers map[string][]supertypeEdge) map[string][]string {
	out := make(map[string][]string, len(t.supers)+len(storedSupers))
	for id, list := range t.supers {
		for _, s := range list {
			out[id] = append(out[id], s.target)
		}
	}
	for _, e := range overlay {
		out[e.Source] = append(out[e.Source], e.Target)
	}
	for id, list := range storedSupers {
		for _, s := range list {
			out[id] = append(out[id], s.target)
		}
	}
	return out
}

// Global is what the passes over the whole graph give, made once at the
// second barrier of a full index: the synthesized edges and the overload
// set edges of each unit, the member pairs whose contains edge a set
// replaces, and the conformance index for the retry of pending calls.
//
// Each edge of the graph belongs to one unit, chosen by its owner path:
// the FilePath of the node that EdgeOwnerID names. A file unit holds the
// edges whose owner path is that file. The no-file unit holds the edges
// whose owner path is "", such as the base edges of a C# partial type and
// the edges of an overload set that spans files. The edges of the route
// result form one more unit. All candidates of one (source, kind, target)
// triple have one owner, so they are in one unit, and a collapse of each
// unit gives what one collapse of the whole graph gives.
//
// A Global is read-only after NewGlobal, so several goroutines can call
// Assemble at once.
type Global struct {
	view symbols.View
	h    symbols.Hierarchy

	units    map[string]*globalUnit
	replaced map[memberPair]struct{}

	// The conformance index: the last member of each name under each
	// source, the owner of each member, and the supertypes of each type
	// with the synthesized implements edges among them.
	lastByName map[string]map[string]string
	owner      map[string]string
	supertypes map[string][]string

	// bindings holds the writes of every data symbol, for the calls
	// through data symbols.
	bindings bindings
}

// globalUnit holds the edges that the whole-graph passes made for one
// unit, in the three categories that keep the order of before.
type globalUnit struct {
	implements, overrides, sets []*schema.Edge
}

type memberPair struct {
	owner, member string
}

// NewGlobal runs the passes that need every file. structural holds the
// structural edges of every result, in result order. NewGlobal reads the
// stored structural edges of view beside them, which a sync has and a full
// index does not.
//
// The order is that of the single global pass it replaces. The structural
// implements synthesis runs first. Its edges then count as supertype edges
// for the overrides synthesis and for the conformance index.
func NewGlobal(view symbols.View, h symbols.Hierarchy, structural []*schema.Edge) *Global {
	g := &Global{
		view:     view,
		h:        h,
		units:    make(map[string]*globalUnit),
		replaced: make(map[memberPair]struct{}),
	}
	t, storedSupers := newTables(structural, view.StoredEdges())
	g.owner = t.owner
	g.bindings = newBindings(view, structural, view.StoredEdges())

	implements := synthesizeGoImplements(view, t, storedSupers)
	g.supertypes = t.supertypes(implements, storedSupers)

	methodKey := make(map[string]string)
	for n := range view.Nodes() {
		if k := goextract.SignatureKeyOfNode(n); k != "" {
			methodKey[n.Id] = k
		}
	}
	overrides := synthesizeOverrides(view, t, g.supertypes, methodKey)

	g.lastByName = make(map[string]map[string]string, len(t.members))
	for source, ids := range t.members {
		byName := make(map[string]string, len(ids))
		for _, id := range ids {
			byName[view.Name(id)] = id
		}
		g.lastByName[source] = byName
	}

	for _, e := range implements {
		u := g.unit(g.OwnerPath(e))
		u.implements = append(u.implements, e)
	}
	for _, e := range overrides {
		u := g.unit(g.OwnerPath(e))
		u.overrides = append(u.overrides, e)
	}
	for _, e := range g.setEdges() {
		u := g.unit(g.OwnerPath(e))
		u.sets = append(u.sets, e)
	}
	return g
}

func (g *Global) unit(path string) *globalUnit {
	u := g.units[path]
	if u == nil {
		u = &globalUnit{}
		g.units[path] = u
	}
	return u
}

// OwnerPath returns the owner path of e: the FilePath of the node that
// EdgeOwnerID names. It decides the unit of e, and the file-index entry
// that the store keeps for e. The lookup covers the overload sets of the
// hierarchy and the nodes of the view. A node that neither holds has the
// path "".
func (g *Global) OwnerPath(e *schema.Edge) string {
	id := EdgeOwnerID(e)
	if set, ok := g.h.Set(id); ok {
		return set.Node().FilePath
	}
	if n := g.view.Node(id); n != nil {
		return n.FilePath
	}
	return ""
}

// retry resolves a call that did not resolve in its file: it walks the
// supertypes of the type that owns the calling method, and looks for a
// member with the name of the call.
func (g *Global) retry(ref goextract.UnresolvedRef) (string, bool) {
	ownerType, ok := g.owner[ref.FromID]
	if !ok {
		return "", false
	}
	var found string
	ok = symbols.WalkSupertypes(ownerType, g.supertypes, func(cur string) bool {
		if id, exists := g.lastByName[cur][ref.Name]; exists {
			found = id
			return true
		}
		return false
	})
	return found, ok
}

// setEdges returns the edges of the overload sets, and records in
// g.replaced each owner and member whose contains edge a set replaces. For
// every set, an edge goes from the owner to the set and one from the set
// to each member. A stored set that this pass does not declare is
// dissolved: each member that still exists, under an owner that still
// exists, gets a contains edge from that owner.
func (g *Global) setEdges() []*schema.Edge {
	setNodes := g.h.SetNodes()
	for _, n := range setNodes {
		set, _ := g.h.Set(n.Id)
		for _, m := range set.Members() {
			g.replaced[memberPair{set.OwnerID(), m.ID}] = struct{}{}
		}
	}
	setEdge := func(source, target string) *schema.Edge {
		return &schema.Edge{
			Source: source, Target: target, Kind: goextract.RefKindContains, Provenance: "ast",
			Metadata: map[string]string{"synthesizedBy": SynthesizedBySet},
		}
	}
	var out []*schema.Edge
	for _, n := range setNodes {
		set, _ := g.h.Set(n.Id)
		out = append(out, setEdge(set.OwnerID(), n.Id))
		for _, m := range set.Members() {
			out = append(out, setEdge(n.Id, m.ID))
		}
	}
	for _, id := range g.view.StoredSetIDs() {
		if _, live := g.h.Set(id); live {
			continue
		}
		stored, _ := g.view.StoredSet(id)
		if g.view.Node(stored.Owner) == nil {
			continue
		}
		for _, m := range stored.Members {
			if g.view.Node(m) == nil {
				continue
			}
			if _, inSet := g.replaced[memberPair{stored.Owner, m}]; inSet {
				continue
			}
			out = append(out, &schema.Edge{Source: stored.Owner, Target: m, Kind: goextract.RefKindContains, Provenance: "ast"})
		}
	}
	return out
}

// NoFileEdges are the edges of one result whose owner path is "". They
// stay apart by category, because the collapse of the no-file unit puts
// the categories of every result in the order of the single global pass.
type NoFileEdges struct {
	relPath                                          string
	types, calls, deferred, retried, retriedDeferred []*schema.Edge
}

// Unit is the finished part of the graph that one result gives.
type Unit struct {
	// Nodes are the nodes of the result, and PackageNodes the package
	// nodes that its imports name.
	Nodes, PackageNodes []*schema.Node
	// Edges are the collapsed edges of the unit, in triple order.
	Edges []*schema.Edge
	// File is the finished File record, or nil for a result without a
	// path, such as the route result.
	File *schema.File
	// NoFile are the edges of the result that belong to the no-file unit.
	NoFile NoFileEdges
	// Unresolved counts the references of the result that did not resolve.
	Unresolved int
}

// Assemble finishes one result: it retries the calls that did not resolve
// against the conformance index, drops the contains edges that an overload
// set replaces, adds the whole-graph edges of the unit, and collapses the
// unit. t and c are the outputs of Types and Calls for the same result.
//
// The edges go into the unit in the order of the single global pass,
// restricted to this result: the type-level edges, the calls without a
// deferred type, the deferred calls, the synthesized implements edges, the
// synthesized overrides, the retried calls and the set edges. CollapseEdges
// keeps the first of two candidates that tie, so the order is part of the
// result.
//
// Assemble enforces the units. In a file unit, each edge of the result must
// have the file or "" as its owner path. In the route unit, each edge must
// have a route node as its source. Any other edge is a defect of the
// resolver, and Assemble returns an error that wraps goextract.ErrInternal.
func (g *Global) Assemble(t TypeOutput, c CallOutput) (Unit, error) {
	u := Unit{Nodes: t.nodes, PackageNodes: t.packageNodes, File: t.file, Unresolved: t.unresolved}
	u.NoFile.relPath = c.relPath
	route := t.file == nil
	path := c.relPath

	var edges []*schema.Edge
	place := func(list []*schema.Edge, noFile *[]*schema.Edge) error {
		for _, e := range list {
			if e.Kind == goextract.RefKindContains {
				if _, drop := g.replaced[memberPair{e.Source, e.Target}]; drop {
					continue
				}
			}
			if route {
				if g.view.Kind(e.Source) != goextract.KindRoute {
					return fmt.Errorf("%w: the route result holds the edge %s -%s-> %s, whose source is not a route", goextract.ErrInternal, e.Source, e.Kind, e.Target)
				}
				edges = append(edges, e)
				continue
			}
			if g.view.Kind(e.Source) == goextract.KindRoute {
				return fmt.Errorf("%w: the result of %s holds the edge %s -%s-> %s, whose source is a route", goextract.ErrInternal, path, e.Source, e.Kind, e.Target)
			}
			switch owner := g.OwnerPath(e); owner {
			case path:
				edges = append(edges, e)
			case "":
				*noFile = append(*noFile, e)
			default:
				return fmt.Errorf("%w: the result of %s holds the edge %s -%s-> %s, which the file %s owns", goextract.ErrInternal, path, e.Source, e.Kind, e.Target, owner)
			}
		}
		return nil
	}
	retry := func(pending []pendingRef) []*schema.Edge {
		var resolved []*schema.Edge
		for _, p := range pending {
			targetID, ok := g.retry(p.ref)
			if !ok {
				u.Unresolved++
				if p.wait != nil && t.file != nil {
					t.file.UnresolvedNames = append(t.file.UnresolvedNames, p.wait)
				}
				continue
			}
			resolved = append(resolved, &schema.Edge{
				Source: p.ref.FromID, Target: targetID, Kind: "calls",
				Line: p.ref.Line, Col: p.ref.Col, Provenance: "ast",
			})
			if t.file != nil {
				t.file.EdgeCount++
			}
		}
		return resolved
	}
	// bind turns the calls into data symbols into reads, plus the calls
	// the bindings establish.
	bind := func(list []*schema.Edge) []*schema.Edge {
		out, added := g.bindings.throughBindings(list)
		if t.file != nil {
			t.file.EdgeCount += int64(added)
		}
		return out
	}

	if t.file != nil {
		t.file.EdgeCount += int64(c.resolved)
	}
	global := g.units[path]
	if route || global == nil {
		global = &globalUnit{}
	}
	for _, step := range []struct {
		list   []*schema.Edge
		noFile *[]*schema.Edge
	}{
		{t.edges, &u.NoFile.types},
		{bind(c.edges), &u.NoFile.calls},
		{bind(c.deferredEdges), &u.NoFile.deferred},
		{global.implements, nil},
		{global.overrides, nil},
		{bind(retry(c.pending)), &u.NoFile.retried},
		{bind(retry(c.deferredPending)), &u.NoFile.retriedDeferred},
		{global.sets, nil},
	} {
		if step.noFile == nil {
			// The whole-graph edges of this unit: NewGlobal chose the unit
			// by the owner path, so they need no check.
			edges = append(edges, step.list...)
			continue
		}
		if err := place(step.list, step.noFile); err != nil {
			return Unit{}, err
		}
	}
	if t.file != nil {
		t.file.UnresolvedNames = sortedUniqueNames(t.file.UnresolvedNames)
	}
	u.Edges = CollapseEdges(edges)
	return u, nil
}

// Rest returns the collapsed whole-graph edges of each file unit that has
// no result, in path order. hasResult reports whether a path has one. A
// full index has a result for every file, so it gets nothing here. A sync
// resolves only the files it reparses, and a whole-graph pass can still
// make an edge whose unit is an unchanged file: a changed file adds an
// interface that an unchanged struct satisfies.
func (g *Global) Rest(hasResult func(path string) bool) []*schema.Edge {
	var paths []string
	for path := range g.units {
		if path != "" && !hasResult(path) {
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	var out []*schema.Edge
	for _, path := range paths {
		u := g.units[path]
		edges := append(append(append([]*schema.Edge(nil), u.implements...), u.overrides...), u.sets...)
		out = append(out, CollapseEdges(edges)...)
	}
	return out
}

// CollapseNoFile returns the collapsed edges of the no-file unit. parts
// holds the NoFile edges of every result, in result order. The order of the
// candidates is that of the single global pass: the type-level edges and
// the calls in result order, the deferred calls in path order, the
// synthesized edges, the retried calls in the same two orders, and the set
// edges.
func (g *Global) CollapseNoFile(parts []NoFileEdges) []*schema.Edge {
	byPath := make([]int, len(parts))
	for i := range byPath {
		byPath[i] = i
	}
	sort.SliceStable(byPath, func(a, b int) bool { return parts[byPath[a]].relPath < parts[byPath[b]].relPath })

	global := g.units[""]
	if global == nil {
		global = &globalUnit{}
	}
	var edges []*schema.Edge
	for _, p := range parts {
		edges = append(edges, p.types...)
	}
	for _, p := range parts {
		edges = append(edges, p.calls...)
	}
	for _, i := range byPath {
		edges = append(edges, parts[i].deferred...)
	}
	edges = append(edges, global.implements...)
	edges = append(edges, global.overrides...)
	for _, p := range parts {
		edges = append(edges, p.retried...)
	}
	for _, i := range byPath {
		edges = append(edges, parts[i].retriedDeferred...)
	}
	edges = append(edges, global.sets...)
	return CollapseEdges(edges)
}

// synthesizeOverrides implements D-09's overrides synthesis
// (01-RESEARCH.md §B): for every type with a supertype, it finds each of
// the type's own methods that shares a name and an arity with a method
// declared directly on one of its supertypes, walked transitively, and
// emits an EdgeKindOverrides edge from the method to the supertype method.
// It reads the member index of t, kept to the members of kind method, and
// supertypes, which holds the synthesized implements edges too.
//
// Go has no `override` keyword: this is structural (name+arity) matching
// — a documented precision note per RESEARCH §B, not a drop. In practice
// it only ever matches through "embeds"/"extends" supertype edges: an
// interface's own method signatures never get a real contains edge, so an
// implements-only supertype chain never yields an override match.
//
// Overloads pair by signature key (parameter types) first; a supertype
// method with the same name and arity pairs only when it is the one such
// method, so two same-arity overloads never cross-link (CR-01). methodKey
// holds the signature key of each function or method node.
func synthesizeOverrides(view symbols.View, t tables, supertypes map[string][]string, methodKey map[string]string) []*schema.Edge {
	typeMethods := make(map[string]map[string][]string) // typeID -> methodName -> every methodID with that name
	for typeID, ids := range t.members {
		for _, id := range ids {
			if view.Kind(id) != goextract.KindMethod {
				continue
			}
			methods := typeMethods[typeID]
			if methods == nil {
				methods = make(map[string][]string)
				typeMethods[typeID] = methods
			}
			name := view.Name(id)
			methods[name] = append(methods[name], id)
		}
	}

	// Deterministic iteration (D-05/D-09): sort the type ids, the method
	// names of each type and the ids of each name, so the order of the
	// emitted edges depends on no map order and on no edge order.
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
				symbols.WalkSupertypes(typeID, supertypes, func(cur string) bool {
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
// Pattern 3, Go's structural method-set match) into the whole-graph
// passes. It takes the method set of each struct from the member index of
// t, which spans files, and the embeds between two interfaces from the
// supertype table, the stored edges included. The method specs of each
// interface come from the view.
func synthesizeGoImplements(view symbols.View, t tables, storedSupers map[string][]supertypeEdge) []*schema.Edge {
	structMethods := make(dispatch.TypeMethods)
	for typeID, ids := range t.members {
		if view.Kind(typeID) != goextract.KindStruct {
			continue
		}
		for _, id := range ids {
			if view.Kind(id) != goextract.KindMethod {
				continue
			}
			structMethods[typeID] = append(structMethods[typeID], goextract.MethodSpec{
				Name:  view.Name(id),
				Arity: view.Arity(id),
			})
		}
	}
	interfaceEmbeds := make(map[string][]string)
	for _, supers := range []map[string][]supertypeEdge{t.supers, storedSupers} {
		for id, list := range supers {
			if view.Kind(id) != goextract.KindInterface {
				continue
			}
			for _, s := range list {
				if s.kind == goextract.RefKindEmbeds && view.Kind(s.target) == goextract.KindInterface {
					interfaceEmbeds[id] = append(interfaceEmbeds[id], s.target)
				}
			}
		}
	}

	synthesized := dispatch.SynthesizeImplements(structMethods, view.InterfaceMethods(), interfaceEmbeds)
	// dispatch.SynthesizeImplements has no source location of its own.
	// Each synthesized edge takes the declaration line of the struct that
	// implements, the closest anchor there is for Go's implicit
	// satisfaction (RES-03).
	for _, e := range synthesized {
		e.Line = view.StartLine(e.Source)
	}
	return synthesized
}
