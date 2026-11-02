package javaextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// collectParams lists the parameters of a Java method or constructor. A
// spread_parameter (`T... rest`) is variadic.
func collectParams(decl *tree_sitter.Node, src []byte) []goextract.Param {
	var params []goextract.Param
	list := decl.ChildByFieldName("parameters")
	if list == nil {
		return nil
	}
	for i := uint(0); i < list.NamedChildCount(); i++ {
		p := list.NamedChild(i)
		var prm goextract.Param
		switch p.Kind() {
		case "formal_parameter":
			if t := p.ChildByFieldName("type"); t != nil {
				prm.Type = t.Utf8Text(src)
			}
			if n := p.ChildByFieldName("name"); n != nil {
				prm.Name = n.Utf8Text(src)
			}
		case "spread_parameter":
			prm.Variadic = true
			for j := uint(0); j < p.NamedChildCount(); j++ {
				c := p.NamedChild(j)
				switch c.Kind() {
				case "variable_declarator":
					if n := c.ChildByFieldName("name"); n != nil {
						prm.Name = n.Utf8Text(src)
					}
				case "modifiers":
				default:
					prm.Type = c.Utf8Text(src)
				}
			}
		default:
			// receiver_parameter (`Foo this`).
			continue
		}
		params = append(params, prm)
	}
	return params
}

// sigKey is the signature key of a method or constructor declaration.
func (ex *extractor) sigKey(decl *tree_sitter.Node) string {
	return goextract.SignatureKeyOf(collectParams(decl, ex.src))
}
