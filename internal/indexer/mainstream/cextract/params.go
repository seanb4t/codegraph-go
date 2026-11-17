package cextract

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// collectParams lists the parameters of a C or C++ function definition
// and its return type text. The parameter list hangs off the
// function_declarator, which pointer declarators can wrap: `char *f()`.
// A parameter's type is its text without its name, so `const char *b`
// gives `const char *`. A lone `void` is no parameter.
func collectParams(decl *tree_sitter.Node, src []byte) ([]goextract.Param, string) {
	stars := 0
	d := decl.ChildByFieldName("declarator")
	for d != nil && d.Kind() != "function_declarator" {
		if d.Kind() == "pointer_declarator" {
			stars++
		}
		d = d.ChildByFieldName("declarator")
	}
	ret := ""
	if t := decl.ChildByFieldName("type"); t != nil {
		ret = t.Utf8Text(src) + strings.Repeat("*", stars)
	}
	if d == nil {
		return nil, ret
	}
	var params []goextract.Param
	if list := d.ChildByFieldName("parameters"); list != nil {
		for i := uint(0); i < list.NamedChildCount(); i++ {
			p := list.NamedChild(i)
			switch p.Kind() {
			case "variadic_parameter":
				params = append(params, goextract.Param{Variadic: true})
			case "parameter_declaration", "optional_parameter_declaration":
				name := ""
				typ := p.Utf8Text(src)
				if nd := innermostIdentifier(p.ChildByFieldName("declarator")); nd != nil {
					name = nd.Utf8Text(src)
					// Cut the name out at its own position: a text search
					// would cut the first letter that matches it.
					start, end := nd.StartByte()-p.StartByte(), nd.EndByte()-p.StartByte()
					typ = strings.TrimSpace(typ[:start] + typ[end:])
				} else if typ == "void" {
					continue
				}
				if v := p.ChildByFieldName("default_value"); v != nil {
					typ = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(strings.TrimSuffix(typ, v.Utf8Text(src))), "="))
				}
				params = append(params, goextract.Param{Name: name, Type: typ, HasDefault: p.Kind() == "optional_parameter_declaration"})
			}
		}
	}
	return params, ret
}

// innermostIdentifier follows declarator fields down to the identifier a
// declarator names, or returns nil.
func innermostIdentifier(d *tree_sitter.Node) *tree_sitter.Node {
	for d != nil {
		switch d.Kind() {
		case "identifier", "field_identifier":
			return d
		}
		next := d.ChildByFieldName("declarator")
		if next == nil {
			return goextract.FirstNamedChildOfKind(d, "identifier")
		}
		d = next
	}
	return nil
}

// sigKey is the signature key of a function definition.
func (ex *extractor) sigKey(decl *tree_sitter.Node) string {
	params, _ := collectParams(decl, ex.src)
	return goextract.SignatureKeyOf(params) + goextract.ConditionalBlockKey(decl, ex.src)
}

// paramKey is the match key of the parameter types of a C or C++
// function (design D5): for each parameter, its leaf tokens joined by
// single spaces, without comments, default values and declarator names;
// the parameters joined by ",". A list of `(void)` keys as empty. Two
// declarations of one function have equal keys whatever their spacing
// and parameter names.
func paramKey(decl *tree_sitter.Node, src []byte) string {
	declarator := decl.ChildByFieldName("declarator")
	if declarator == nil {
		return ""
	}
	fd := findFunctionDeclarator(declarator)
	if fd == nil {
		return ""
	}
	list := fd.ChildByFieldName("parameters")
	if list == nil {
		return ""
	}
	var keys []string
	for i := uint(0); i < list.ChildCount(); i++ {
		p := list.Child(i)
		switch {
		case p == nil || p.Kind() == "comment":
		case p.IsNamed():
			keys = append(keys, paramTokens(p, src))
		case p.Kind() == "...":
			// The C++ grammar leaves the ellipsis of a variadic
			// function unnamed.
			keys = append(keys, "...")
		}
	}
	if len(keys) == 1 && keys[0] == "void" {
		return ""
	}
	return strings.Join(keys, ",")
}

// paramTokens joins the leaf tokens of the parameter p that paramKey
// keeps. Each parameter declaration in p, an inner parameter of a
// function-pointer type included, loses the name of its declarator.
func paramTokens(p *tree_sitter.Node, src []byte) string {
	skip := make(map[uintptr]bool)
	var tokens []string
	walkDescendants(p, func(n *tree_sitter.Node) bool {
		if skip[n.Id()] {
			return false
		}
		switch n.Kind() {
		case "comment":
			return false
		case "parameter_declaration", "optional_parameter_declaration":
			if name := declaratorName(n.ChildByFieldName("declarator")); name != nil {
				skip[name.Id()] = true
			}
			if v := n.ChildByFieldName("default_value"); v != nil {
				skip[v.Id()] = true
				for i := uint(0); i < n.ChildCount(); i++ {
					if c := n.Child(i); c != nil && c.Kind() == "=" {
						skip[c.Id()] = true
					}
				}
			}
		}
		if n.ChildCount() == 0 {
			tokens = append(tokens, n.Utf8Text(src))
		}
		return true
	})
	return strings.Join(tokens, " ")
}

// declaratorName returns the identifier that the declarator d names, or
// nil for an abstract declarator. It follows the declarator field of a
// pointer, array or function declarator, and the one named child of a
// parenthesized or reference declarator. It never takes another child,
// so the size N of `a[N]` is no name.
func declaratorName(d *tree_sitter.Node) *tree_sitter.Node {
	for d != nil {
		switch d.Kind() {
		case "identifier", "field_identifier":
			return d
		case "function_declarator":
			d = d.ChildByFieldName("declarator")
		default:
			d = unwrapDeclarator(d)
		}
	}
	return nil
}

// unwrapDeclarator returns the declarator that a pointer, array, reference
// or parenthesized declarator d wraps, or nil when d is none of those. A
// reference declarator has no declarator field, so it yields its last named
// child.
func unwrapDeclarator(d *tree_sitter.Node) *tree_sitter.Node {
	switch d.Kind() {
	case "pointer_declarator", "array_declarator":
		return d.ChildByFieldName("declarator")
	case "parenthesized_declarator", "reference_declarator":
		return lastNamedDeclarator(d)
	}
	return nil
}

// lastNamedDeclarator returns the named child of a parenthesized or
// reference declarator that is not a comment or a qualifier.
func lastNamedDeclarator(d *tree_sitter.Node) *tree_sitter.Node {
	for i := int(d.NamedChildCount()) - 1; i >= 0; i-- {
		switch c := d.NamedChild(uint(i)); c.Kind() {
		case "comment", "type_qualifier", "ms_call_modifier", "attribute_declaration":
		default:
			return c
		}
	}
	return nil
}
