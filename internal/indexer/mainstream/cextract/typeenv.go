package cextract

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// enclosingClass returns the class or struct specifier whose body holds
// decl, or nil for a free function or an out-of-line method.
func enclosingClass(decl *tree_sitter.Node) *tree_sitter.Node {
	for p := decl.Parent(); p != nil; p = p.Parent() {
		switch p.Kind() {
		case "class_specifier", "struct_specifier":
			return p
		case "function_definition":
			if p != decl {
				return nil
			}
		}
	}
	return nil
}

// fieldsOf returns the environment of a class or struct body's fields.
func (ex *extractor) fieldsOf(class *tree_sitter.Node) *goextract.TypeEnv {
	env := goextract.NewTypeEnv(nil)
	if class == nil {
		return env
	}
	body := class.ChildByFieldName("body")
	if body == nil {
		return env
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		f := body.NamedChild(i)
		if f.Kind() != "field_declaration" {
			continue
		}
		ex.declareDeclarators(env, f)
	}
	return env
}

// declareDeclarators declares each declarator of a declaration or field
// with the declaration's type text, plus a `*` per pointer declarator.
func (ex *extractor) declareDeclarators(env *goextract.TypeEnv, d *tree_sitter.Node) {
	typ := ""
	if t := d.ChildByFieldName("type"); t != nil {
		typ = t.Utf8Text(ex.src)
	}
	goextract.ForEachChild(d, func(c *tree_sitter.Node, field string) {
		if field != "declarator" {
			return
		}
		stars, name, init := 0, "", (*tree_sitter.Node)(nil)
		for n := c; n != nil; {
			switch n.Kind() {
			case "init_declarator":
				init = n.ChildByFieldName("value")
				n = n.ChildByFieldName("declarator")
			case "pointer_declarator", "reference_declarator":
				if n.Kind() == "pointer_declarator" {
					stars++
				}
				n = n.ChildByFieldName("declarator")
			case "identifier", "field_identifier":
				name = n.Utf8Text(ex.src)
				n = nil
			case "array_declarator", "function_declarator", "parenthesized_declarator":
				n = n.ChildByFieldName("declarator")
			default:
				n = nil
			}
		}
		if name == "" {
			return
		}
		t := typ + strings.Repeat("*", stars)
		if (typ == "auto" || typ == "") && init != nil {
			t = ex.exprType(env, init)
		}
		env.Declare(name, t)
	})
}

// bodyEnv returns the environment of one function body.
func (ex *extractor) bodyEnv(decl *tree_sitter.Node) *goextract.TypeEnv {
	var fields *goextract.TypeEnv
	className := ""
	if class := enclosingClass(decl); class != nil {
		fields = ex.fieldsOf(class)
		if n := class.ChildByFieldName("name"); n != nil {
			className = n.Utf8Text(ex.src)
		}
	} else if scope := outOfLineScope(decl, ex.src); scope != "" {
		className = scope
		fields = ex.fields[scope]
	}
	env := goextract.NewTypeEnv(fields)
	env.Declare("this", className)
	params, _ := collectParams(decl, ex.src)
	for _, p := range params {
		env.Declare(p.Name, p.Type)
	}
	body := decl.ChildByFieldName("body")
	if body == nil {
		return env
	}
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "declaration":
			ex.declareDeclarators(env, n)
		case "lambda_expression":
			return true
		}
		return true
	})
	return env
}

// outOfLineScope returns the class name of an out-of-line method
// definition, `void Foo::bar() {}`, or "".
func outOfLineScope(decl *tree_sitter.Node, src []byte) string {
	d := decl.ChildByFieldName("declarator")
	for d != nil && d.Kind() != "function_declarator" {
		d = d.ChildByFieldName("declarator")
	}
	if d == nil {
		return ""
	}
	if q := d.ChildByFieldName("declarator"); q != nil && q.Kind() == "qualified_identifier" {
		scope, _, ok := splitQualifiedMethod(q, src)
		if ok {
			return scope
		}
	}
	return ""
}

// exprType returns the static type of expression n when the syntax or the
// environment shows it, a deferred type for a call, and "" otherwise.
func (ex *extractor) exprType(env *goextract.TypeEnv, n *tree_sitter.Node) string {
	switch n.Kind() {
	case "string_literal", "concatenated_string":
		return "const char *"
	case "char_literal":
		return "char"
	case "number_literal":
		s := strings.ToLower(n.Utf8Text(ex.src))
		switch {
		case strings.HasSuffix(s, "f") && !strings.HasPrefix(s, "0x"):
			return "float"
		case strings.ContainsAny(s, ".e") && !strings.HasPrefix(s, "0x"):
			return "double"
		case strings.HasSuffix(s, "l"):
			return "long"
		}
		return "int"
	case "true", "false":
		return "bool"
	case "new_expression":
		if t := n.ChildByFieldName("type"); t != nil {
			return t.Utf8Text(ex.src) + "*"
		}
	case "cast_expression":
		if t := n.ChildByFieldName("type"); t != nil {
			return t.Utf8Text(ex.src)
		}
	case "call_expression":
		return goextract.DeferredType(n)
	case "identifier":
		typ, _ := env.Lookup(n.Utf8Text(ex.src))
		return typ
	case "this":
		typ, _ := env.Lookup("this")
		return typ
	case "field_expression":
		arg, field := n.ChildByFieldName("argument"), n.ChildByFieldName("field")
		if arg != nil && field != nil && arg.Kind() == "this" {
			typ, _ := env.Lookup(field.Utf8Text(ex.src))
			return typ
		}
	case "pointer_expression":
		if a := n.ChildByFieldName("argument"); a != nil {
			return ex.exprType(env, a)
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

// receiver classifies the argument of a field call, `a.m()` or
// `p->m()` (D17, D18).
func (ex *extractor) receiver(env *goextract.TypeEnv, arg *tree_sitter.Node) (pkgAlias, receiverType string) {
	if arg == nil {
		return "", ""
	}
	switch arg.Kind() {
	case "identifier":
		return goextract.Receiver(env, ex.result.Imports, arg.Utf8Text(ex.src))
	case "this":
		typ, _ := env.Lookup("this")
		return "", typ
	}
	if typ := ex.exprType(env, arg); typ != "" {
		return "", typ
	}
	return goextract.UnknownAlias(arg.Kind()), ""
}
