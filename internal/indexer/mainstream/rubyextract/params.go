package rubyextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// collectParams lists the parameters of a Ruby method. Ruby writes no
// types. A splat or hash splat is variadic, and an optional or keyword
// parameter with a value has a default.
func collectParams(decl *tree_sitter.Node, src []byte) []goextract.Param {
	var params []goextract.Param
	list := decl.ChildByFieldName("parameters")
	if list == nil {
		return nil
	}
	for i := uint(0); i < list.NamedChildCount(); i++ {
		p := list.NamedChild(i)
		var prm goextract.Param
		if n := p.ChildByFieldName("name"); n != nil {
			prm.Name = n.Utf8Text(src)
		}
		switch p.Kind() {
		case "identifier":
			prm.Name = p.Utf8Text(src)
		case "optional_parameter":
			prm.HasDefault = true
		case "keyword_parameter":
			prm.HasDefault = p.ChildByFieldName("value") != nil
		case "splat_parameter", "hash_splat_parameter":
			prm.Variadic = true
		case "block_parameter", "destructured_parameter":
		default:
			continue
		}
		params = append(params, prm)
	}
	return params
}

// sigKey is the signature key of a method.
func (ex *extractor) sigKey(decl *tree_sitter.Node) string {
	return goextract.SignatureKeyOf(collectParams(decl, ex.src))
}
