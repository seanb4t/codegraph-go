package resolve

import (
	"maps"
	"slices"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// The metadata keys and values of data references and of the calls that
// go through a data symbol.
const (
	// MetaAccess is the role of a references edge: goextract.AccessRead,
	// AccessWrite or AccessReadWrite.
	MetaAccess = goextract.MetaAccess
	// MetaBinding is BindingInit on the reference from a data symbol to
	// the name its initializer holds.
	MetaBinding = "binding"
	// MetaBinds is on a write: the ids of the values it assigns, sorted
	// and comma-separated. BindsUnknown stands for a value that is not a
	// resolved name.
	MetaBinds = goextract.MetaBinds
	// MetaVia is on a call through data symbols: each binding path, the
	// data symbol ids from the called one to the last, joined by ">".
	// Several paths are sorted and comma-separated.
	MetaVia = "via"
	// MetaContext is on a call through a Go data symbol: the analysis
	// contexts that establish the target, sorted and comma-separated.
	MetaContext = "binding_context"
	// MetaDirect is "true" on a call that is both direct and through a
	// data symbol.
	MetaDirect = "direct"

	BindingInit  = "init"
	BindsUnknown = goextract.BindsUnknown

	// ContextGoProduction leaves out the writes of _test.go files, and
	// ContextGoTest counts every write.
	ContextGoProduction = "go-production"
	ContextGoTest       = "go-test"

	// maxBindingChain is the most data symbols one binding path goes
	// through.
	maxBindingChain = 8
)

// IsBindingEdge reports whether e carries a binding fact: the initial
// value of a data symbol, or a write.
func IsBindingEdge(e *schema.Edge) bool {
	if e.Kind != goextract.RefKindReferences {
		return false
	}
	if e.Metadata[MetaBinding] == BindingInit {
		return true
	}
	a := e.Metadata[MetaAccess]
	return a == goextract.AccessWrite || a == goextract.AccessReadWrite
}

// GlobalEdges returns the edges of a result that the passes over the
// whole graph read: the structural edges and the binding edges.
func GlobalEdges(edges []*schema.Edge) []*schema.Edge {
	var out []*schema.Edge
	for _, e := range edges {
		if IsStructural(e) || IsBindingEdge(e) {
			out = append(out, e)
		}
	}
	return out
}

// isDataKind reports whether a node of kind holds a value: a call through
// it is a read, never a call of it.
func isDataKind(kind string) bool {
	switch kind {
	case goextract.KindVariable, goextract.KindConstant, goextract.KindField:
		return true
	}
	return false
}

// isGoTestFile reports whether n is in a Go _test.go file.
func isGoTestFile(n *schema.Node) bool {
	return n != nil && n.Language == "go" && strings.HasSuffix(n.FilePath, "_test.go")
}

// bindingWrite is one value assigned to a data symbol. test is true for a
// write in a Go _test.go file.
type bindingWrite struct {
	value string
	test  bool
}

// bindings holds the values written to each data symbol, from the binding
// edges of every file.
type bindings struct {
	view   symbols.View
	h      symbols.Hierarchy
	writes map[string][]bindingWrite
	// damaged holds the module keys of the files with syntax errors.
	damaged map[string]bool
}

func newBindings(view symbols.View, h symbols.Hierarchy, lists ...[]*schema.Edge) bindings {
	b := bindings{view: view, h: h, writes: make(map[string][]bindingWrite), damaged: view.SyntaxErrorModules()}
	for _, list := range lists {
		for _, e := range list {
			if !IsBindingEdge(e) {
				continue
			}
			test := isGoTestFile(view.Node(e.Source))
			if e.Metadata[MetaBinding] == BindingInit {
				b.writes[e.Source] = append(b.writes[e.Source], bindingWrite{e.Target, test})
				continue
			}
			binds := e.Metadata[MetaBinds]
			if e.Metadata[MetaAccess] != goextract.AccessWrite || binds == "" {
				binds = BindsUnknown
			}
			for _, v := range strings.Split(binds, ",") {
				b.writes[e.Target] = append(b.writes[e.Target], bindingWrite{v, test})
			}
		}
	}
	return b
}

// target returns the function that the data symbol id holds in the
// context, and the binding path to it. A symbol holds a function when
// every write the context counts assigns the same value, and that value
// is a function or a data symbol that holds one. A write of an unknown
// value, two values, no value, a cycle, a path longer than
// maxBindingChain, a data symbol in a package with a syntax error, or
// an open field establish no target.
func (b bindings) target(id string, test bool) (string, []string, bool) {
	var via []string
	for len(via) < maxBindingChain {
		if slices.Contains(via, id) {
			return "", nil, false
		}
		if n := b.view.Node(id); n == nil || b.damaged[b.view.ModuleOf(n.FilePath)] || b.open(n) {
			return "", nil, false
		}
		via = append(via, id)
		value := ""
		for _, w := range b.writes[id] {
			switch {
			case w.test && !test:
				continue
			case w.value == BindsUnknown:
				return "", nil, false
			case value == "":
				value = w.value
			case value != w.value:
				return "", nil, false
			}
		}
		switch k := b.view.Kind(value); {
		case value == "":
			return "", nil, false
		case k == goextract.KindFunction || k == goextract.KindMethod:
			return value, via, true
		case isDataKind(k):
			id = value
		default:
			return "", nil, false
		}
	}
	return "", nil, false
}

// open reports whether n is a field that a subclass can write. The
// writes through a subclass resolve only after the bindings are read, so
// the field's writes are not all known.
func (b bindings) open(n *schema.Node) bool {
	if n.Kind != goextract.KindField || !InheritsMembers(n.Language) {
		return false
	}
	owner, ok := b.h.OwnerOf(n.Id)
	return ok && b.h.HasSubtypes(owner)
}

// throughBindings replaces each calls edge of list into a data symbol
// with a read of that symbol, plus a call of each function the symbol
// holds in the call site's contexts. A Go call site outside a _test.go
// file runs in both contexts, and one inside runs under go-test only. A
// call site in another language counts every write and names no context.
// It returns the edges and the number of calls it added.
func (b bindings) throughBindings(list []*schema.Edge) ([]*schema.Edge, int) {
	var out []*schema.Edge
	added := 0
	for _, e := range list {
		if e.Kind != goextract.RefKindCalls || !isDataKind(b.view.Kind(e.Target)) {
			out = append(out, e)
			continue
		}
		out = append(out, &schema.Edge{
			Source: e.Source, Target: e.Target, Kind: goextract.RefKindReferences,
			Line: e.Line, Col: e.Col, Provenance: "ast",
			Metadata: map[string]string{MetaAccess: goextract.AccessRead},
		})
		for _, d := range b.derive(e) {
			out = append(out, d)
			added++
		}
	}
	return out, added
}

// derive returns the calls that the call e into a data symbol makes, one
// per function target, in target order.
func (b bindings) derive(e *schema.Edge) []*schema.Edge {
	type evidence struct{ via, contexts []string }
	found := make(map[string]*evidence)
	add := func(test bool, context string) {
		fn, via, ok := b.target(e.Target, test)
		if !ok {
			return
		}
		ev := found[fn]
		if ev == nil {
			ev = &evidence{}
			found[fn] = ev
		}
		ev.via = append(ev.via, strings.Join(via, ">"))
		if context != "" {
			ev.contexts = append(ev.contexts, context)
		}
	}
	switch src := b.view.Node(e.Source); {
	case src == nil || src.Language != "go":
		add(true, "")
	case isGoTestFile(src):
		add(true, ContextGoTest)
	default:
		add(false, ContextGoProduction)
		add(true, ContextGoTest)
	}
	var out []*schema.Edge
	for _, fn := range slices.Sorted(maps.Keys(found)) {
		ev := found[fn]
		md := map[string]string{MetaVia: joinSet(ev.via)}
		if len(ev.contexts) > 0 {
			md[MetaContext] = joinSet(ev.contexts)
		}
		out = append(out, &schema.Edge{
			Source: e.Source, Target: fn, Kind: goextract.RefKindCalls,
			Line: e.Line, Col: e.Col, Provenance: "ast", Metadata: md,
		})
	}
	return out
}

// joinSet returns the sorted, deduplicated items, comma-separated.
func joinSet(items []string) string {
	items = slices.Clone(items)
	slices.Sort(items)
	return strings.Join(slices.Compact(items), ",")
}

// unionSet merges two comma-separated sets.
func unionSet(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	return joinSet(append(strings.Split(a, ","), strings.Split(b, ",")...))
}
