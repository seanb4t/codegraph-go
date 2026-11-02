package goextract

import tree_sitter "github.com/tree-sitter/go-tree-sitter"

// collectParams lists the parameters of a Go function or method
// declaration and its result text. A parameter_declaration with several
// names, `a, b int`, gives one Param per name.
func collectParams(decl *tree_sitter.Node, src []byte) ([]Param, string) {
	var params []Param
	if list := decl.ChildByFieldName("parameters"); list != nil {
		for i := uint(0); i < list.NamedChildCount(); i++ {
			p := list.NamedChild(i)
			variadic := p.Kind() == "variadic_parameter_declaration"
			if !variadic && p.Kind() != "parameter_declaration" {
				continue
			}
			typ := ""
			if t := p.ChildByFieldName("type"); t != nil {
				typ = t.Utf8Text(src)
			}
			names := 0
			for j := uint(0); j < p.NamedChildCount(); j++ {
				if c := p.NamedChild(j); c.Kind() == "identifier" {
					params = append(params, Param{Name: c.Utf8Text(src), Type: typ, Variadic: variadic})
					names++
				}
			}
			if names == 0 {
				params = append(params, Param{Type: typ, Variadic: variadic})
			}
		}
	}
	ret := ""
	if r := decl.ChildByFieldName("result"); r != nil {
		ret = r.Utf8Text(src)
	}
	return params, ret
}

// sigKey is the signature key of a function or method declaration.
func (ex *extractor) sigKey(decl *tree_sitter.Node) string {
	params, _ := collectParams(decl, ex.src)
	return SignatureKeyOf(params)
}
