package csharpextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// collectParams lists the parameters of a C# method or constructor. A
// plain parameter is a parameter node with type and name fields, and a
// value child after the name means a default. The grammar inlines a
// `params T[] rest` parameter: its `params` keyword, type and name hang
// directly on the parameter list, so the walk assembles it from those
// children and marks it variadic.
func collectParams(decl *tree_sitter.Node, src []byte) []goextract.Param {
	var params []goextract.Param
	list := decl.ChildByFieldName("parameters")
	if list == nil {
		return nil
	}
	var inlined *goextract.Param
	goextract.ForEachChild(list, func(c *tree_sitter.Node, field string) {
		switch {
		case c.Kind() == "parameter":
			params = append(params, oneParam(c, src))
		case c.Kind() == "params":
			inlined = &goextract.Param{Variadic: true}
		case field == "type" && inlined != nil:
			inlined.Type = c.Utf8Text(src)
		case field == "name" && inlined != nil:
			inlined.Name = c.Utf8Text(src)
			params = append(params, *inlined)
			inlined = nil
		}
	})
	return params
}

func oneParam(p *tree_sitter.Node, src []byte) goextract.Param {
	var prm goextract.Param
	goextract.ForEachChild(p, func(c *tree_sitter.Node, field string) {
		switch {
		case field == "type":
			prm.Type = c.Utf8Text(src)
		case field == "name":
			prm.Name = c.Utf8Text(src)
		case c.Kind() == "params":
			prm.Variadic = true
		case c.IsNamed() && c.Kind() != "modifier" && c.Kind() != "attribute_list":
			prm.HasDefault = true
		}
	})
	return prm
}

// sigKey is the signature key of a method or constructor declaration.
func (ex *extractor) sigKey(decl *tree_sitter.Node) string {
	return goextract.SignatureKeyOf(collectParams(decl, ex.src)) + goextract.ConditionalBlockKey(decl, ex.src)
}
