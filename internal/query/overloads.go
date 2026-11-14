package query

import (
	"errors"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

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

// targets holds what symbolTargets resolved: the nodes, and the two
// modes of a query over more than one node. overloaded marks a query
// that spans overloads or selects one, so each row carries a signature.
// spans marks a query whose definitions are not all members of one
// overload set, so each row carries its definition. definitions is the
// list of those definitions, and is nil when spans is false.
type targets struct {
	nodes       []target
	overloaded  bool
	spans       bool
	definitions []Location
}

// symbolTargets resolves sel to the nodes a traversal starts from: every
// definition named sel.Symbol (enumerateSymbolDefs, the meaning `node`
// uses), and every overload_set node of that name, marked ambiguous.
// sel.Signature keeps the callables whose parameter types match it, and
// file keeps the definitions in the files that match it. A set stays
// only while one of its members stays. When a filter keeps nothing, the
// error lists what exists.
//
// The start nodes are ordered by file, line, kind and qualified name,
// never by id, so no answer depends on the hash of a node.
func (e *Engine) symbolTargets(sel Selector, file string) (targets, error) {
	named, err := e.nodesNamed(sel.Symbol)
	if err != nil {
		return targets{}, err
	}
	defs := definitionsOf(named)
	var sets []*schema.Node
	for _, n := range named {
		if n.Kind == goextract.KindOverloadSet {
			sets = append(sets, n)
		}
	}
	if len(defs) == 0 && len(sets) == 0 {
		return targets{}, notFoundf("query: symbol %q not found", sel.Symbol)
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
	}

	kept := defs
	if sel.Signature != "" {
		if kept, err = narrowBySignature(sel.Symbol, kept, sel.Signature); err != nil {
			return targets{}, err
		}
	}
	if file != "" {
		inFile := filterByFile(kept, file)
		if len(inFile) == 0 {
			files := definitionFiles(defs)
			if len(files) == 0 {
				return targets{}, notFoundf("query: no definition of %q is in a file that matches %q; no definition of %q has a file", sel.Symbol, file, sel.Symbol)
			}
			return targets{}, notFoundf("query: no definition of %q is in a file that matches %q; the files are %s", sel.Symbol, file, strings.Join(files, ", "))
		}
		kept = inFile
	}
	kept = slices.Clone(kept)
	sortDefinitions(kept)

	filtered := sel.Signature != "" || file != ""
	out := targets{overloaded: sel.Signature != ""}
	keptSets := make(map[string]bool)
	for _, n := range kept {
		out.nodes = append(out.nodes, target{node: n})
		if set, ok := setOfMember[n.Id]; ok {
			keptSets[set.Id] = true
		}
	}
	sortDefinitions(sets)
	for _, set := range sets {
		if filtered && !keptSets[set.Id] {
			continue
		}
		out.overloaded = true
		out.nodes = append(out.nodes, target{node: set, ambiguous: true})
	}
	if spansDefinitions(kept, setOfMember) {
		out.spans = true
		for _, n := range kept {
			l := nodeLocation(n)
			l.Definition = DefinitionLabel(n)
			out.definitions = append(out.definitions, l)
		}
	}
	return out, nil
}

// spansDefinitions reports whether defs are not all members of one
// overload set: two or more definitions outside a set, a set and a
// definition outside it, or members of two sets.
func spansDefinitions(defs []*schema.Node, setOfMember map[string]*schema.Node) bool {
	if len(defs) < 2 {
		return false
	}
	first, ok := setOfMember[defs[0].Id]
	if !ok {
		return true
	}
	for _, n := range defs[1:] {
		if set, ok := setOfMember[n.Id]; !ok || set.Id != first.Id {
			return true
		}
	}
	return false
}

// sortDefinitions orders nodes by file, line, kind and qualified name.
func sortDefinitions(nodes []*schema.Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		a, b := nodes[i], nodes[j]
		if a.FilePath != b.FilePath {
			return a.FilePath < b.FilePath
		}
		if a.StartLine != b.StartLine {
			return a.StartLine < b.StartLine
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.QualifiedName < b.QualifiedName
	})
}

// definitionFiles returns the files that hold one of defs, each once,
// sorted. A definition with no file is left out.
func definitionFiles(defs []*schema.Node) []string {
	var files []string
	for _, n := range defs {
		if n.FilePath != "" {
			files = append(files, n.FilePath)
		}
	}
	sort.Strings(files)
	return slices.Compact(files)
}

// DefinitionLabel names a definition in the output: its file and line,
// or its kind and qualified name when it has no file, as a package or a
// type whose parts are in several files.
func DefinitionLabel(n *schema.Node) string {
	if n.FilePath == "" {
		return n.Kind + " " + n.QualifiedName
	}
	return n.FilePath + ":" + strconv.Itoa(int(n.StartLine))
}

// definitionLabelLess orders two DefinitionLabel values by file and
// then by line as a number, so "a.py:5" comes before "a.py:12". Labels
// without a file and line come after them and compare as text.
func definitionLabelLess(a, b string) bool {
	aFile, aLine, aOK := splitDefinitionLabel(a)
	bFile, bLine, bOK := splitDefinitionLabel(b)
	switch {
	case aOK != bOK:
		return aOK
	case !aOK:
		return a < b
	case aFile != bFile:
		return aFile < bFile
	default:
		return aLine < bLine
	}
}

func splitDefinitionLabel(label string) (string, int, bool) {
	i := strings.LastIndexByte(label, ':')
	if i < 0 {
		return "", 0, false
	}
	line, err := strconv.Atoi(label[i+1:])
	if err != nil {
		return "", 0, false
	}
	return label[:i], line, true
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
// signature: a comma-separated list, with or without parentheses or
// spaces, so "String, Throwable" matches `log(String msg, Throwable t)`.
// A list with parameter names matches only when it is the node's own
// signature text.
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

// normalizeTypeList strips parentheses and spaces from a type list, and
// reduces each type as goextract.TypeName does on the node side, so
// "(String, Throwable)" and "String,Throwable" compare equal, and "*T"
// matches a Go parameter stored as T.
func normalizeTypeList(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	parts := strings.Split(s, ",")
	for i, p := range parts {
		variadic := strings.HasSuffix(strings.TrimSpace(p), "...")
		t := goextract.TypeName(strings.TrimSuffix(strings.TrimSpace(p), "..."))
		if variadic {
			t += "..."
		}
		parts[i] = t
	}
	return strings.Join(parts, ",")
}

// compactSignature renders a signature on one line for a label: runs of
// whitespace, newlines included, become one space, and control runes are
// dropped. The source text of a multi-line parameter list stays in the
// node; only the label is compact.
func compactSignature(sig string) string {
	sig = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, sig)
	return strings.Join(strings.Fields(sig), " ")
}

// calleeLocation is the row of a callee reached from the overload t: the
// Signature names t, the overload the call was made from, and Ambiguous
// marks a callee that is itself an overload set, so one row can carry
// both.
func calleeLocation(callee *schema.Node, t target, tg targets) Location {
	l := overloadLocation(callee, t, tg)
	if callee.Kind == goextract.KindOverloadSet {
		l.Ambiguous = true
	}
	return l
}

// overloadLocation is nodeLocation plus the fields of a query over more
// than one node, for a row that belongs to target t: the signature of t
// or the ambiguous mark when the query is overloaded, and the definition
// label of t when the query spans definitions. A row that belongs to a
// set names no definition.
func overloadLocation(n *schema.Node, t target, tg targets) Location {
	l := nodeLocation(n)
	if tg.overloaded {
		if t.ambiguous {
			l.Ambiguous = true
		} else {
			l.Signature = compactSignature(t.node.Signature)
		}
	}
	if tg.spans && !t.ambiguous {
		l.Definition = DefinitionLabel(t.node)
	}
	return l
}
