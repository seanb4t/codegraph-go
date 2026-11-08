package symbols

import (
	"sort"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// StoredSet is an overload set as the store holds it: the ids of its
// owner and its members. It has no nodes, unlike OverloadSet, because a
// sync reads it from the files it does not reparse.
type StoredSet struct {
	Owner   string
	Members []string
}

// data is the declaration universe that a Builder fills and a View reads.
type data struct {
	// candidates holds, per module key and declared name, every distinct
	// symbol that Add received, in the order Add received them. The
	// declared name is a node's bare Name: for a method the bare method
	// name, never "Recv.Method", matching how the extractors record a
	// call site.
	candidates map[string]map[string][]Candidate

	// membersByOwner holds, per owner node id and member name, every
	// member a contains edge puts under a non-file owner (design D3): the
	// candidate set of a call through a receiver of that type.
	membersByOwner map[string]map[string][]Candidate

	// collisions counts every (module key, name) that Add received more
	// than one symbol for.
	collisions int

	kinds, names      map[string]string
	startLines, arity map[string]int32
	nodes             map[string]*schema.Node
	interfaceMethods  map[string][]goextract.MethodSpec

	// moduleOf maps each file to its module key. A deferred type is
	// written in the callee's file, so it resolves in that module first.
	moduleOf map[string]string

	// storedSets are the overload sets the store holds for the files a
	// sync does not reparse, and storedEdges the contains, embeds,
	// extends and declared implements edges those files own.
	storedSets  map[string]StoredSet
	storedEdges []*schema.Edge

	// typeNames are the names of the type nodes, for the known-type rule
	// of the type filter, and typesByName the ids of the type nodes under
	// each name, for the subtype walk. Freeze fills both.
	typeNames   map[string]bool
	typesByName map[string][]string
}

// Builder collects the declarations of one resolution: the results of
// pass 1 and, for a sync, the stored graph of the files it does not
// reparse. Freeze ends the collection and returns the read-only View.
type Builder struct {
	d      *data
	frozen bool
}

// NewBuilder returns an empty Builder.
func NewBuilder() *Builder {
	return &Builder{d: &data{
		candidates:       make(map[string]map[string][]Candidate),
		membersByOwner:   make(map[string]map[string][]Candidate),
		kinds:            make(map[string]string),
		names:            make(map[string]string),
		startLines:       make(map[string]int32),
		arity:            make(map[string]int32),
		nodes:            make(map[string]*schema.Node),
		interfaceMethods: make(map[string][]goextract.MethodSpec),
		moduleOf:         make(map[string]string),
		storedSets:       make(map[string]StoredSet),
	}}
}

func (b *Builder) write() *data {
	if b.frozen {
		panic("symbols: write to a Builder after Freeze")
	}
	return b.d
}

// Add files n under moduleKey and its bare name. Re-adding a candidate
// with the same id, such as a sync that overlays an unchanged file, is
// not a collision. Every other candidate for a name that already has one
// counts as a collision.
func (b *Builder) Add(moduleKey string, n *schema.Node) {
	d := b.write()
	names := d.candidates[moduleKey]
	if names == nil {
		names = make(map[string][]Candidate)
		d.candidates[moduleKey] = names
	}
	cands := names[n.Name]
	for _, c := range cands {
		if c.ID == n.Id {
			return
		}
	}
	if len(cands) > 0 {
		d.collisions++
	}
	names[n.Name] = append(cands, CandidateOf(n))
}

// AddMember records that the type ownerID contains member, from a
// contains edge. Re-adding the same member is not a change.
func (b *Builder) AddMember(ownerID string, member *schema.Node) {
	d := b.write()
	byName := d.membersByOwner[ownerID]
	if byName == nil {
		byName = make(map[string][]Candidate)
		d.membersByOwner[ownerID] = byName
	}
	for _, c := range byName[member.Name] {
		if c.ID == member.Id {
			return
		}
	}
	byName[member.Name] = append(byName[member.Name], CandidateOf(member))
}

// AddNode records n in the node lookups, whether or not it is a symbol.
func (b *Builder) AddNode(n *schema.Node) {
	d := b.write()
	d.kinds[n.Id] = n.Kind
	d.names[n.Id] = n.Name
	d.startLines[n.Id] = n.StartLine
	d.nodes[n.Id] = n
}

// SetArity records the parameter count of the method id.
func (b *Builder) SetArity(id string, arity int32) {
	b.write().arity[id] = arity
}

// AddInterfaceMethods records methods that the interface id declares.
func (b *Builder) AddInterfaceMethods(id string, methods ...goextract.MethodSpec) {
	d := b.write()
	d.interfaceMethods[id] = append(d.interfaceMethods[id], methods...)
}

// SetModule records the module key of file.
func (b *Builder) SetModule(file, moduleKey string) {
	b.write().moduleOf[file] = moduleKey
}

// AddStoredSet records the stored overload set id.
func (b *Builder) AddStoredSet(id string, set StoredSet) {
	b.write().storedSets[id] = set
}

// AddStoredEdge records a contains, embeds, extends or declared
// implements edge of a file the sync does not reparse.
func (b *Builder) AddStoredEdge(e *schema.Edge) {
	d := b.write()
	d.storedEdges = append(d.storedEdges, e)
}

// AddResults adds the declarations of results, in the order given: every
// node to the node lookups, the reachable ones as symbols under their
// file's module key, the members of each type from the file's contains
// edges, the method arities and the interface method specs. A skipped
// file (FileResult.Err != nil) declares no symbol.
func (b *Builder) AddResults(results []goextract.FileResult) {
	for _, r := range results {
		for _, en := range r.Nodes {
			b.AddNode(en.Node)
		}
		for id, arity := range r.MethodArity {
			b.SetArity(id, arity)
		}
		for ifaceID, specs := range r.InterfaceMethods {
			b.AddInterfaceMethods(ifaceID, specs...)
		}
		b.SetModule(r.RelPath, r.ImportPath)
		if r.Err != nil {
			continue
		}
		b.addSymbols(r)
	}
}

func (b *Builder) addSymbols(r goextract.FileResult) {
	byID := make(map[string]*schema.Node, len(r.Nodes))
	fileExports := false
	for _, en := range r.Nodes {
		byID[en.Node.Id] = en.Node
		fileExports = fileExports || en.Node.IsExported
	}
	for _, en := range r.Nodes {
		if en.Node.Kind == goextract.KindFile {
			// A file node is never a callable or embeddable target, and
			// its synthetic name, its RelPath, must not collide with a
			// declared symbol of the module.
			continue
		}
		if !Reachable(en.Node, fileExports) {
			continue
		}
		b.Add(r.ImportPath, en.Node)
	}
	for _, ie := range r.IntraEdges {
		if ie.Edge.Kind != goextract.RefKindContains {
			continue
		}
		owner, member := byID[ie.Edge.Source], byID[ie.Edge.Target]
		if owner == nil || member == nil || owner.Kind == goextract.KindFile {
			continue
		}
		b.AddMember(owner.Id, member)
	}
}

// Node returns the node id, or nil.
func (b *Builder) Node(id string) *schema.Node { return b.d.nodes[id] }

// Kind returns the kind of the node id, or "".
func (b *Builder) Kind(id string) string { return b.d.kinds[id] }

// Collisions returns the number of names that received more than one
// symbol.
func (b *Builder) Collisions() int { return b.d.collisions }

// Freeze ends the collection and returns the View over it. A write to b
// after Freeze panics.
func (b *Builder) Freeze() View {
	d := b.write()
	b.frozen = true
	d.typeNames = make(map[string]bool)
	d.typesByName = make(map[string][]string)
	for id, n := range d.nodes {
		if IsTypeKind(n.Kind) {
			d.typeNames[n.Name] = true
			d.typesByName[n.Name] = append(d.typesByName[n.Name], id)
		}
	}
	for _, ids := range d.typesByName {
		sort.Strings(ids)
	}
	return View{d: d}
}
