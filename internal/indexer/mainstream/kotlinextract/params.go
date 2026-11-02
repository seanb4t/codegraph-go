package kotlinextract

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// collectParams lists the parameters of a Kotlin function and its return
// type text. The grammar puts a parameter's default value and its
// `vararg` modifier next to the parameter, not inside it: a
// parameter_modifiers node applies to the parameter after it, and a value
// node belongs to the parameter before it. The return type is the type
// node right after function_value_parameters.
func collectParams(decl *tree_sitter.Node, src []byte) ([]goextract.Param, string) {
	var params []goextract.Param
	ret := ""
	variadic := false
	for i := uint(0); i < decl.NamedChildCount(); i++ {
		c := decl.NamedChild(i)
		if c.Kind() != "function_value_parameters" {
			continue
		}
		for j := uint(0); j < c.NamedChildCount(); j++ {
			p := c.NamedChild(j)
			switch p.Kind() {
			case "parameter":
				var prm goextract.Param
				if id := goextract.FirstNamedChildOfKind(p, "identifier", "simple_identifier"); id != nil {
					prm.Name = id.Utf8Text(src)
				}
				for k := uint(0); k < p.NamedChildCount(); k++ {
					if t := p.NamedChild(k); strings.HasSuffix(t.Kind(), "_type") {
						prm.Type = t.Utf8Text(src)
						break
					}
				}
				prm.Variadic = variadic
				variadic = false
				params = append(params, prm)
			case "parameter_modifiers":
				variadic = strings.Contains(p.Utf8Text(src), "vararg")
			default:
				if len(params) > 0 {
					params[len(params)-1].HasDefault = true
				}
			}
		}
		if i+1 < decl.NamedChildCount() {
			if t := decl.NamedChild(i + 1); strings.HasSuffix(t.Kind(), "_type") {
				ret = t.Utf8Text(src)
			}
		}
		break
	}
	return params, ret
}

// sigKey is the signature key of a function declaration.
func (ex *extractor) sigKey(decl *tree_sitter.Node) string {
	params, _ := collectParams(decl, ex.src)
	return goextract.SignatureKeyOf(params)
}
