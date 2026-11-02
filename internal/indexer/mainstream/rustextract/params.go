package rustextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// collectParams lists the parameters of a Rust function and its return
// type text. A self_parameter counts as a parameter named self, as
// written.
func collectParams(decl *tree_sitter.Node, src []byte) ([]goextract.Param, string) {
	var params []goextract.Param
	if list := decl.ChildByFieldName("parameters"); list != nil {
		for i := uint(0); i < list.NamedChildCount(); i++ {
			p := list.NamedChild(i)
			var prm goextract.Param
			switch p.Kind() {
			case "parameter":
				if pat := p.ChildByFieldName("pattern"); pat != nil && pat.Kind() == "identifier" {
					prm.Name = pat.Utf8Text(src)
				}
				if t := p.ChildByFieldName("type"); t != nil {
					prm.Type = t.Utf8Text(src)
				}
			case "self_parameter":
				prm.Name = "self"
			case "variadic_parameter":
				prm.Variadic = true
			default:
				continue
			}
			params = append(params, prm)
		}
	}
	ret := ""
	if rt := decl.ChildByFieldName("return_type"); rt != nil {
		ret = rt.Utf8Text(src)
	}
	return params, ret
}

// sigKey is the signature key of a function item.
func (ex *extractor) sigKey(decl *tree_sitter.Node) string {
	params, _ := collectParams(decl, ex.src)
	return goextract.SignatureKeyOf(params)
}
