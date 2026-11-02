package tsextract

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// collectParams lists the parameters of a TypeScript or JavaScript
// function, method or arrow function, and its return type text. The
// TypeScript grammar wraps each parameter in required_parameter or
// optional_parameter. The JavaScript grammar lists the patterns directly.
func collectParams(sig *tree_sitter.Node, src []byte) ([]goextract.Param, string) {
	var params []goextract.Param
	if list := sig.ChildByFieldName("parameters"); list != nil {
		for i := uint(0); i < list.NamedChildCount(); i++ {
			p := list.NamedChild(i)
			var prm goextract.Param
			switch p.Kind() {
			case "required_parameter", "optional_parameter":
				prm.Optional = p.Kind() == "optional_parameter"
				if pat := p.ChildByFieldName("pattern"); pat != nil {
					prm.Name, prm.Variadic = patternName(pat, src)
				}
				if t := p.ChildByFieldName("type"); t != nil {
					prm.Type = annotationType(t, src)
				}
				prm.HasDefault = p.ChildByFieldName("value") != nil
			case "identifier", "this":
				prm.Name = p.Utf8Text(src)
			case "assignment_pattern":
				if l := p.ChildByFieldName("left"); l != nil {
					prm.Name, _ = patternName(l, src)
				}
				prm.HasDefault = true
			case "rest_pattern", "object_pattern", "array_pattern":
				prm.Name, prm.Variadic = patternName(p, src)
			default:
				continue
			}
			params = append(params, prm)
		}
	} else if p := sig.ChildByFieldName("parameter"); p != nil {
		// An arrow function with one unparenthesized parameter: x => x.
		params = append(params, goextract.Param{Name: p.Utf8Text(src)})
	}
	ret := ""
	if rt := sig.ChildByFieldName("return_type"); rt != nil {
		ret = annotationType(rt, src)
	}
	return params, ret
}

// patternName returns the name a parameter pattern binds, or "" for a
// destructuring pattern, and whether it is a rest pattern.
func patternName(pat *tree_sitter.Node, src []byte) (string, bool) {
	switch pat.Kind() {
	case "identifier", "this":
		return pat.Utf8Text(src), false
	case "rest_pattern":
		if pat.NamedChildCount() > 0 {
			name, _ := patternName(pat.NamedChild(0), src)
			return name, true
		}
		return "", true
	}
	return "", false
}

// annotationType returns the type inside a type_annotation (`: T`).
func annotationType(t *tree_sitter.Node, src []byte) string {
	if t.Kind() == "type_annotation" && t.NamedChildCount() > 0 {
		return t.NamedChild(0).Utf8Text(src)
	}
	return strings.TrimSpace(strings.TrimPrefix(t.Utf8Text(src), ":"))
}

// sigKey is the signature key of a function, method or arrow function.
func (ex *extractor) sigKey(sig *tree_sitter.Node) string {
	params, _ := collectParams(sig, ex.src)
	return goextract.SignatureKeyOf(params)
}
