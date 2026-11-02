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
