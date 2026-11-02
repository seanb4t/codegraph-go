package swiftextract

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// collectParams lists the parameters of a Swift function, their argument
// labels, and the return type text. The grammar puts each parameter, its
// default value and the return type directly under function_declaration:
// a default_value field belongs to the parameter before it, and the
// return type is the child under the field `name` whose kind is a type.
// A parameter's label is its external name, or its name when it has no
// external name. A `_` external name means no label.
func collectParams(decl *tree_sitter.Node, src []byte) (params []goextract.Param, labels []string, ret string) {
	goextract.ForEachChild(decl, func(c *tree_sitter.Node, field string) {
		switch {
		case c.Kind() == "parameter":
			var prm goextract.Param
			ext := ""
			goextract.ForEachChild(c, func(pc *tree_sitter.Node, pf string) {
				switch {
				case pf == "external_name":
					ext = pc.Utf8Text(src)
				case pf == "name" && pc.Kind() == "simple_identifier":
					prm.Name = pc.Utf8Text(src)
				case pc.IsNamed() && strings.HasSuffix(pc.Kind(), "_type"):
					if prm.Type == "" {
						prm.Type = pc.Utf8Text(src)
					}
				case !pc.IsNamed() && pc.Kind() == "...":
					prm.Variadic = true
				}
			})
			if ext == "" {
				ext = prm.Name
			}
			labels = append(labels, ext)
			params = append(params, prm)
		case field == "default_value":
			if len(params) > 0 {
				params[len(params)-1].HasDefault = true
			}
		case field == "name" && c.Kind() != "simple_identifier":
			ret = c.Utf8Text(src)
		}
	})
	return params, labels, ret
}

// labeledName is the name of a Swift function with its argument labels,
// as Swift writes it: move(to:by:), or move() for no parameters.
func labeledName(name string, labels []string) string {
	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('(')
	for _, l := range labels {
		b.WriteString(l)
		b.WriteByte(':')
	}
	b.WriteByte(')')
	return b.String()
}

// sigKey is the signature key of a function declaration.
func (ex *extractor) sigKey(decl *tree_sitter.Node) string {
	params, _, _ := collectParams(decl, ex.src)
	return goextract.SignatureKeyOf(params)
}
