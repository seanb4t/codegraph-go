package csharpextract

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// fieldsOf returns the environment of a type body's fields and
// properties with their declared types.
func (ex *extractor) fieldsOf(body *tree_sitter.Node) *goextract.TypeEnv {
	env := goextract.NewTypeEnv(nil)
	if body == nil {
		return env
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		m := body.NamedChild(i)
		switch m.Kind() {
		case "field_declaration", "event_field_declaration":
			if vd := goextract.FirstNamedChildOfKind(m, "variable_declaration"); vd != nil {
				ex.declareVariables(env, vd, false)
			}
		case "property_declaration":
			typ := ""
			if t := m.ChildByFieldName("type"); t != nil {
				typ = t.Utf8Text(ex.src)
			}
			if n := m.ChildByFieldName("name"); n != nil {
				env.Declare(n.Utf8Text(ex.src), typ)
			}
		}
	}
	return env
}

// declareVariables declares each declarator of a variable_declaration. A
// `var` declaration takes the type of the declarator's initializer when
// exprType knows it.
func (ex *extractor) declareVariables(env *goextract.TypeEnv, vd *tree_sitter.Node, local bool) {
	typ := ""
	if t := vd.ChildByFieldName("type"); t != nil {
		typ = t.Utf8Text(ex.src)
	}
	for i := uint(0); i < vd.NamedChildCount(); i++ {
		d := vd.NamedChild(i)
		if d.Kind() != "variable_declarator" {
			continue
		}
		name := d.ChildByFieldName("name")
		if name == nil {
			name = goextract.FirstNamedChildOfKind(d, "identifier")
		}
		if name == nil {
			continue
		}
		t := typ
		if t == "var" {
			t = ""
			for j := uint(0); j < d.NamedChildCount(); j++ {
				if c := d.NamedChild(j); c.Kind() != "identifier" {
					t = ex.exprType(env, c)
				}
			}
		}
		if local {
			env.DeclareIn(name.Utf8Text(ex.src), t, uint32(vd.EndByte()), goextract.ScopeEnd(vd, "csharp"))
		} else {
			env.Declare(name.Utf8Text(ex.src), t)
		}
	}
}

// bodyEnv returns the environment of one method body: `this`, the
// parameters, and every local declared in the body.
func (ex *extractor) bodyEnv(fields *goextract.TypeEnv, typeName string, decl *tree_sitter.Node) *goextract.TypeEnv {
	env := goextract.NewTypeEnv(fields)
	env.Declare("this", typeName)
	for _, p := range collectParams(decl, ex.src) {
		env.Declare(p.Name, p.Type)
	}
	body := decl.ChildByFieldName("body")
	if body == nil {
		return env
	}
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "variable_declaration":
			ex.declareVariables(env, n, true)
		case "foreach_statement", "catch_declaration":
			typ := ""
			if t := n.ChildByFieldName("type"); t != nil {
				typ = t.Utf8Text(ex.src)
			}
			name := n.ChildByFieldName("left")
			if name == nil {
				name = n.ChildByFieldName("name")
			}
			if name != nil && name.Kind() == "identifier" {
				env.DeclareIn(name.Utf8Text(ex.src), typ, uint32(n.StartByte()), goextract.ScopeEnd(n, "csharp"))
			}
		case "lambda_expression":
			if ps := n.ChildByFieldName("parameters"); ps != nil {
				if ps.Kind() == "identifier" {
					env.Declare(ps.Utf8Text(ex.src), "")
				}
				for i := uint(0); i < ps.NamedChildCount(); i++ {
					if id := goextract.FirstNamedChildOfKind(ps.NamedChild(i), "identifier"); id != nil {
						env.Declare(id.Utf8Text(ex.src), "")
					} else if ps.NamedChild(i).Kind() == "identifier" {
						env.Declare(ps.NamedChild(i).Utf8Text(ex.src), "")
					}
				}
			}
		}
		return true
	})
	return env
}

// exprType returns the static type of expression n when the syntax or the
// environment shows it, a deferred type for a call, and "" otherwise.
func (ex *extractor) exprType(env *goextract.TypeEnv, n *tree_sitter.Node) string {
	switch n.Kind() {
	case "string_literal", "verbatim_string_literal", "interpolated_string_expression", "raw_string_literal":
		return "string"
	case "integer_literal":
		s := strings.ToLower(n.Utf8Text(ex.src))
		switch {
		case strings.HasSuffix(s, "ul"), strings.HasSuffix(s, "lu"):
			return "ulong"
		case strings.HasSuffix(s, "l"):
			return "long"
		case strings.HasSuffix(s, "u"):
			return "uint"
		}
		return "int"
	case "real_literal":
		s := strings.ToLower(n.Utf8Text(ex.src))
		switch {
		case strings.HasSuffix(s, "f"):
			return "float"
		case strings.HasSuffix(s, "m"):
			return "decimal"
		}
		return "double"
	case "character_literal":
		return "char"
	case "boolean_literal":
		return "bool"
	case "object_creation_expression", "cast_expression":
		if t := n.ChildByFieldName("type"); t != nil {
			return t.Utf8Text(ex.src)
		}
	case "as_expression":
		if n.NamedChildCount() > 1 {
			return n.NamedChild(1).Utf8Text(ex.src)
		}
	case "identifier":
		typ, _ := env.LookupAt(n.Utf8Text(ex.src), uint32(n.StartByte()))
		return typ
	case "this_expression", "this":
		typ, _ := env.Lookup("this")
		return typ
	case "invocation_expression":
		return goextract.DeferredType(n)
	case "member_access_expression":
		if obj := n.ChildByFieldName("expression"); obj != nil && (obj.Kind() == "this_expression" || obj.Kind() == "this") {
			if f := n.ChildByFieldName("name"); f != nil {
				typ, _ := env.Lookup(f.Utf8Text(ex.src))
				return typ
			}
		}
	case "parenthesized_expression":
		if n.NamedChildCount() > 0 {
			return ex.exprType(env, n.NamedChild(0))
		}
	}
	return ""
}

// callArgs reads an argument_list. A named argument's expression is its
// last named child.
func (ex *extractor) callArgs(env *goextract.TypeEnv, args *tree_sitter.Node) goextract.CallArgs {
	var ca goextract.CallArgs
	if args == nil {
		return ca
	}
	for i := uint(0); i < args.NamedChildCount(); i++ {
		a := args.NamedChild(i)
		if a.Kind() != "argument" {
			continue
		}
		ca.Count++
		typ := ""
		if k := a.NamedChildCount(); k > 0 {
			typ = ex.exprType(env, a.NamedChild(k-1))
		}
		ca.Types = append(ca.Types, typ)
	}
	return ca
}

// receiver classifies the expression a member access reads: what it
// names and its static type (D17, D18). A qualified name keeps today's
// namespace-prefix import so pass 2 can look the type up in it.
func (ex *extractor) receiver(env *goextract.TypeEnv, object *tree_sitter.Node) (pkgAlias, receiverType string) {
	if object == nil {
		return "", ""
	}
	switch object.Kind() {
	case "this_expression", "this":
		typ, _ := env.Lookup("this")
		return "", typ
	case "base_expression", "base":
		return "", ""
	case "identifier", "predefined_type":
		return goextract.ReceiverAt(env, ex.result.Imports, object.Utf8Text(ex.src), uint32(object.StartByte()))
	case "qualified_name":
		q, name := object.ChildByFieldName("qualifier"), object.ChildByFieldName("name")
		if q != nil && name != nil {
			return ex.namespacePrefixAlias(q.Utf8Text(ex.src), ""), name.Utf8Text(ex.src)
		}
		return goextract.UnknownAlias(object.Kind()), ""
	}
	if typ := ex.exprType(env, object); typ != "" {
		return "", typ
	}
	if object.Kind() == "member_access_expression" {
		root := chainRoot(object)
		if root.Kind() == "identifier" {
			if _, declared := env.LookupAt(root.Utf8Text(ex.src), uint32(root.StartByte())); declared {
				return goextract.LocalAlias(root.Utf8Text(ex.src)), ""
			}
			inner, name := object.ChildByFieldName("expression"), object.ChildByFieldName("name")
			if inner != nil && name != nil {
				return ex.namespacePrefixAlias(inner.Utf8Text(ex.src), ""), name.Utf8Text(ex.src)
			}
		}
	}
	return goextract.UnknownAlias(object.Kind()), ""
}
