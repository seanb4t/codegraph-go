package javaextract

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// fieldsOf returns the environment of a class body's fields: each
// field_declaration's declarators with the field's type.
func (ex *extractor) fieldsOf(body *tree_sitter.Node) *goextract.TypeEnv {
	env := goextract.NewTypeEnv(nil)
	if body == nil {
		return env
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		f := body.NamedChild(i)
		if f.Kind() != "field_declaration" {
			continue
		}
		typ := ""
		if t := f.ChildByFieldName("type"); t != nil {
			typ = t.Utf8Text(ex.src)
		}
		for j := uint(0); j < f.NamedChildCount(); j++ {
			if d := f.NamedChild(j); d.Kind() == "variable_declarator" {
				if n := d.ChildByFieldName("name"); n != nil {
					env.Declare(n.Utf8Text(ex.src), typ)
				}
			}
		}
	}
	return env
}

// bodyEnv returns the environment of one method body: `this`, the
// parameters, and every local declared in the body. A `var` local takes
// the type of its initializer when exprType knows it.
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
		case "local_variable_declaration":
			typ := ""
			if t := n.ChildByFieldName("type"); t != nil {
				typ = t.Utf8Text(ex.src)
			}
			for i := uint(0); i < n.NamedChildCount(); i++ {
				d := n.NamedChild(i)
				if d.Kind() != "variable_declarator" {
					continue
				}
				name := d.ChildByFieldName("name")
				if name == nil {
					continue
				}
				t := typ
				if t == "var" {
					t = ""
					if v := d.ChildByFieldName("value"); v != nil {
						t = ex.exprType(env, v)
					}
				}
				env.DeclareIn(name.Utf8Text(ex.src), t, uint32(n.EndByte()), goextract.ScopeEnd(n, "java"))
			}
		case "enhanced_for_statement", "catch_formal_parameter", "resource":
			typ := ""
			if t := n.ChildByFieldName("type"); t != nil {
				typ = t.Utf8Text(ex.src)
			}
			if name := n.ChildByFieldName("name"); name != nil && name.Kind() == "identifier" {
				env.DeclareIn(name.Utf8Text(ex.src), typ, uint32(n.StartByte()), goextract.ScopeEnd(n, "java"))
			}
		case "lambda_expression":
			if ps := n.ChildByFieldName("parameters"); ps != nil {
				if ps.Kind() == "identifier" {
					env.Declare(ps.Utf8Text(ex.src), "")
				}
				for i := uint(0); i < ps.NamedChildCount(); i++ {
					p := ps.NamedChild(i)
					switch p.Kind() {
					case "identifier":
						env.Declare(p.Utf8Text(ex.src), "")
					case "formal_parameter", "inferred_parameters":
						if name := p.ChildByFieldName("name"); name != nil {
							env.Declare(name.Utf8Text(ex.src), "")
						}
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
	case "string_literal", "text_block":
		return "String"
	case "decimal_integer_literal", "hex_integer_literal", "octal_integer_literal", "binary_integer_literal":
		if strings.HasSuffix(strings.ToLower(n.Utf8Text(ex.src)), "l") {
			return "long"
		}
		return "int"
	case "decimal_floating_point_literal", "hex_floating_point_literal":
		if strings.HasSuffix(strings.ToLower(n.Utf8Text(ex.src)), "f") {
			return "float"
		}
		return "double"
	case "character_literal":
		return "char"
	case "true", "false":
		return "boolean"
	case "object_creation_expression", "cast_expression":
		if t := n.ChildByFieldName("type"); t != nil {
			return t.Utf8Text(ex.src)
		}
	case "identifier":
		typ, _ := env.LookupAt(n.Utf8Text(ex.src), uint32(n.StartByte()))
		return typ
	case "this":
		typ, _ := env.Lookup("this")
		return typ
	case "method_invocation":
		return goextract.DeferredType(n)
	case "field_access":
		if obj := n.ChildByFieldName("object"); obj != nil && obj.Kind() == "this" {
			if f := n.ChildByFieldName("field"); f != nil {
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

// callArgs reads an argument_list.
func (ex *extractor) callArgs(env *goextract.TypeEnv, args *tree_sitter.Node) goextract.CallArgs {
	var ca goextract.CallArgs
	if args == nil {
		return ca
	}
	for i := uint(0); i < args.NamedChildCount(); i++ {
		a := args.NamedChild(i)
		ca.Count++
		ca.Types = append(ca.Types, ex.exprType(env, a))
	}
	return ca
}

// receiver classifies the object of a member call or field access: what
// it names and its static type (D17, D18).
func (ex *extractor) receiver(env *goextract.TypeEnv, object *tree_sitter.Node) (pkgAlias, receiverType string) {
	switch {
	case object == nil:
		return "", ""
	case object.Kind() == "this":
		typ, _ := env.Lookup("this")
		return "", typ
	case object.Kind() == "super":
		return "", ""
	case object.Kind() == "identifier":
		return goextract.ReceiverAt(env, ex.result.Imports, object.Utf8Text(ex.src), uint32(object.StartByte()))
	}
	if typ := ex.exprType(env, object); typ != "" {
		return "", typ
	}
	return goextract.UnknownAlias(object.Kind()), ""
}
