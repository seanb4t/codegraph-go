package pyextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// collectParams lists the parameters of a Python function and its return
// annotation text. `self` and `cls` count as parameters, as written.
// `*args` and `**kwargs` are variadic.
func collectParams(decl *tree_sitter.Node, src []byte) ([]goextract.Param, string) {
	var params []goextract.Param
	if list := decl.ChildByFieldName("parameters"); list != nil {
		for i := uint(0); i < list.NamedChildCount(); i++ {
			p := list.NamedChild(i)
			var prm goextract.Param
			switch p.Kind() {
			case "identifier":
				prm.Name = p.Utf8Text(src)
			case "typed_parameter":
				if id := goextract.FirstNamedChildOfKind(p, "identifier"); id != nil {
					prm.Name = id.Utf8Text(src)
				}
				if t := p.ChildByFieldName("type"); t != nil {
					prm.Type = t.Utf8Text(src)
				}
			case "default_parameter", "typed_default_parameter":
				if n := p.ChildByFieldName("name"); n != nil {
					prm.Name = n.Utf8Text(src)
				}
				if t := p.ChildByFieldName("type"); t != nil {
					prm.Type = t.Utf8Text(src)
				}
				prm.HasDefault = true
			case "list_splat_pattern", "dictionary_splat_pattern":
				if id := goextract.FirstNamedChildOfKind(p, "identifier"); id != nil {
					prm.Name = id.Utf8Text(src)
				}
				prm.Variadic = true
			default:
				// keyword_separator (`*`) and positional_separator (`/`).
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

// sigKey is the signature key of a function or method declaration.
func (ex *extractor) sigKey(decl *tree_sitter.Node) string {
	params, _ := collectParams(decl, ex.src)
	return goextract.SignatureKeyOf(params)
}
