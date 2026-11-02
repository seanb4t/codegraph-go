package phpextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// collectParams lists the parameters of a PHP function or method and its
// return type text. A variadic_parameter (`T ...$rest`) is variadic, and a
// parameter with a default_value has a default.
func collectParams(decl *tree_sitter.Node, src []byte) ([]goextract.Param, string) {
	var params []goextract.Param
	if list := decl.ChildByFieldName("parameters"); list != nil {
		for i := uint(0); i < list.NamedChildCount(); i++ {
			p := list.NamedChild(i)
			switch p.Kind() {
			case "simple_parameter", "property_promotion_parameter", "variadic_parameter":
			default:
				continue
			}
			var prm goextract.Param
			if t := p.ChildByFieldName("type"); t != nil {
				prm.Type = t.Utf8Text(src)
			}
			if n := p.ChildByFieldName("name"); n != nil {
				prm.Name = n.Utf8Text(src)
			}
			prm.HasDefault = p.ChildByFieldName("default_value") != nil
			prm.Variadic = p.Kind() == "variadic_parameter"
			params = append(params, prm)
		}
	}
	ret := ""
	if rt := decl.ChildByFieldName("return_type"); rt != nil {
		ret = rt.Utf8Text(src)
	}
	return params, ret
}

// sigKey is the signature key of a function or method declaration.
func (ex *extractor) sigKey(decl *tree_sitter.Node) string {
	params, _ := collectParams(decl, ex.src)
	return goextract.SignatureKeyOf(params) + goextract.ConditionalBlockKey(decl, ex.src)
}
