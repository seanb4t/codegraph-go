package query

import (
	"errors"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Selector names the symbol a query starts from. Signature, when set,
// selects one overload by its parameter types, as `--signature "String,
// Throwable"` does on the command line. Without it, a name that has an
// overload set reports across every overload.
type Selector struct {
	Symbol    string
	Signature string
}

// target is one node a traversal starts from. ambiguous marks the
// overload_set node: an edge into it is a call that static analysis
// could not decide.
type target struct {
	node      *schema.Node
	ambiguous bool
}

// targets holds what symbolTargets resolved: the nodes, and whether the
// query spans overloads, in which case each result carries a signature.
type targets struct {
	nodes      []target
	overloaded bool
}

// symbolTargets resolves sel to the nodes a traversal starts from. With
// no signature: the members and the node of every overload set named
// sel.Symbol, or else the one node resolveSymbolNode picks. With a
// signature: the callables named sel.Symbol whose parameter types match
// it, plus the set that holds them, so the set's ambiguous callers are
// reported too. When no overload matches, the error lists the
// signatures that exist.
func (e *Engine) symbolTargets(sel Selector) (targets, error) {
	it, err := e.reader.IterateNodes()
	if err != nil {
		return targets{}, err
	}
	var named []*schema.Node
	for it.Next() {
		if n := it.Node(); n.Name == sel.Symbol {
			named = append(named, n)
		}
	}
	err = it.Err()
	it.Close()
	if err != nil {
		return targets{}, err
	}
	if len(named) == 0 {
		return targets{}, notFoundf("query: symbol %q not found", sel.Symbol)
	}
	sort.Slice(named, func(i, j int) bool { return named[i].Id < named[j].Id })

	var sets, members []*schema.Node
	for _, n := range named {
		if n.Kind == goextract.KindOverloadSet {
			sets = append(sets, n)
		}
	}
	setOfMember := make(map[string]*schema.Node)
	for _, set := range sets {
		ms, err := e.setMembers(set)
		if err != nil {
			return targets{}, err
		}
		for _, m := range ms {
			setOfMember[m.Id] = set
		}
		members = append(members, ms...)
	}

	if sel.Signature != "" {
		var chosen []target
		seenSet := make(map[string]bool)
		var signatures []string
		for _, n := range named {
			if n.Kind != goextract.KindFunction && n.Kind != goextract.KindMethod {
				continue
			}
			signatures = append(signatures, n.Signature)
			if !matchesSignature(n, sel.Signature) {
				continue
			}
			chosen = append(chosen, target{node: n})
			if set, ok := setOfMember[n.Id]; ok && !seenSet[set.Id] {
				seenSet[set.Id] = true
				chosen = append(chosen, target{node: set, ambiguous: true})
			}
		}
		if len(chosen) == 0 {
			sort.Strings(signatures)
			signatures = compactStrings(signatures)
			if len(signatures) == 0 {
				return targets{}, notFoundf("query: no overload of %q has the signature %q: %q is not a function or method", sel.Symbol, sel.Signature, sel.Symbol)
			}
			return targets{}, notFoundf("query: no overload of %q has the signature %q; the signatures are %s", sel.Symbol, sel.Signature, strings.Join(signatures, ", "))
		}
		return targets{nodes: chosen, overloaded: true}, nil
	}

	if len(sets) == 0 {
		n, err := e.resolveSymbolNode(sel.Symbol)
		if err != nil {
			return targets{}, err
		}
		return targets{nodes: []target{{node: n}}}, nil
	}
	out := targets{overloaded: true}
	for _, m := range members {
		out.nodes = append(out.nodes, target{node: m})
	}
	for _, set := range sets {
		out.nodes = append(out.nodes, target{node: set, ambiguous: true})
	}
	return out, nil
}

// setMembers returns the overloads an overload_set node contains, by id.
func (e *Engine) setMembers(set *schema.Node) ([]*schema.Node, error) {
	it, err := e.reader.IterateEdges(set.Id)
	if err != nil {
		return nil, err
	}
	defer it.Close()
	var members []*schema.Node
	for it.Next() {
		edge := it.Edge()
		if edge.Kind != goextract.RefKindContains {
			continue
		}
		m, err := e.reader.GetNode(edge.Target)
		if err != nil {
			if errors.Is(err, graphstore.ErrNotFound) {
				continue
			}
			return nil, err
		}
		members = append(members, m)
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	sort.Slice(members, func(i, j int) bool { return members[i].Id < members[j].Id })
	return members, nil
}

// matchesSignature reports whether n's parameter types are the types in
// signature: a comma-separated list, with or without parentheses, names
// or spaces. "String, Throwable" matches `log(String msg, Throwable t)`.
func matchesSignature(n *schema.Node, signature string) bool {
	want := normalizeTypeList(signature)
	var types []string
	for _, p := range n.Params {
		t := goextract.TypeName(p.Type)
		if p.Variadic {
			t += "..."
		}
		types = append(types, t)
	}
	if normalizeTypeList(strings.Join(types, ",")) == want {
		return true
	}
	return normalizeTypeList(n.Signature) == want
}

// normalizeTypeList strips parentheses and spaces from a type list, so
// "(String, Throwable)" and "String,Throwable" compare equal.
func normalizeTypeList(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = strings.Join(strings.Fields(p), " ")
	}
	return strings.Join(parts, ",")
}

func compactStrings(in []string) []string {
	var out []string
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}

// overloadLocation is nodeLocation plus the overload fields, for a row
// that belongs to target t of a query that spans overloads.
func overloadLocation(n *schema.Node, t target, overloaded bool) Location {
	l := nodeLocation(n)
	if !overloaded {
		return l
	}
	if t.ambiguous {
		l.Ambiguous = true
	} else {
		l.Signature = t.node.Signature
	}
	return l
}
