package swiftextract

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// enclosingType returns the class, struct, enum, extension or protocol
// declaration whose body holds decl, or nil for a top-level function.
func enclosingType(decl *tree_sitter.Node) *tree_sitter.Node {
	for p := decl.Parent(); p != nil; p = p.Parent() {
		switch p.Kind() {
		case "class_declaration", "protocol_declaration":
			return p
		case "function_declaration":
			return nil
		}
	}
	return nil
}

// typeAnnotationOf returns the type of a property or parameter from its
// type_annotation child, or the first child whose kind ends in _type.
func typeAnnotationOf(n *tree_sitter.Node, src []byte) string {
	if ta := goextract.FirstNamedChildOfKind(n, "type_annotation"); ta != nil {
		for i := uint(0); i < ta.NamedChildCount(); i++ {
			if c := ta.NamedChild(i); strings.HasSuffix(c.Kind(), "_type") {
				return c.Utf8Text(src)
			}
		}
	}
	for i := uint(0); i < n.NamedChildCount(); i++ {
		if c := n.NamedChild(i); strings.HasSuffix(c.Kind(), "_type") {
			return c.Utf8Text(src)
		}
	}
	return ""
}

// declareProperty declares a let or var property: its name from its
// pattern, and its annotated type or the type of its initializer.
func (ex *extractor) declareProperty(env *goextract.TypeEnv, p *tree_sitter.Node) {
	name := ""
	if pat := goextract.FirstNamedChildOfKind(p, "pattern"); pat != nil {
		if id := goextract.FirstNamedChildOfKind(pat, "simple_identifier"); id != nil {
			name = id.Utf8Text(ex.src)
		} else if pat.NamedChildCount() == 0 {
			name = pat.Utf8Text(ex.src)
		}
	}
	if name == "" {
		return
	}
	typ := typeAnnotationOf(p, ex.src)
	if typ == "" {
		if v := p.ChildByFieldName("value"); v != nil {
			typ = ex.exprType(env, v)
		}
	}
	env.Declare(name, typ)
}

// fieldsOf returns the environment of a type body's properties.
func (ex *extractor) fieldsOf(typeDecl *tree_sitter.Node) *goextract.TypeEnv {
	env := goextract.NewTypeEnv(nil)
	if typeDecl == nil {
		return env
	}
	body := typeDecl.ChildByFieldName("body")
	if body == nil {
		return env
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		if p := body.NamedChild(i); p.Kind() == "property_declaration" {
			ex.declareProperty(env, p)
		}
	}
	return env
}

// bodyEnv returns the environment of one function body.
func (ex *extractor) bodyEnv(decl *tree_sitter.Node) *goextract.TypeEnv {
	var fields *goextract.TypeEnv
	typeName := ""
	if t := enclosingType(decl); t != nil {
		fields = ex.fieldsOf(t)
		if n := t.ChildByFieldName("name"); n != nil {
			typeName = n.Utf8Text(ex.src)
		}
	}
	env := goextract.NewTypeEnv(fields)
	env.Declare("self", typeName)
	params, _, _ := collectParams(decl, ex.src)
	for _, p := range params {
		env.Declare(p.Name, p.Type)
	}
	body := decl.ChildByFieldName("body")
	if body == nil {
		return env
	}
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "property_declaration":
			ex.declareProperty(env, n)
		case "lambda_function_type_parameters", "lambda_parameter":
			if id := goextract.FirstNamedChildOfKind(n, "simple_identifier"); id != nil {
				env.Declare(id.Utf8Text(ex.src), typeAnnotationOf(n, ex.src))
			}
		case "for_statement":
			if pat := goextract.FirstNamedChildOfKind(n, "pattern"); pat != nil {
				if id := goextract.FirstNamedChildOfKind(pat, "simple_identifier"); id != nil {
					env.Declare(id.Utf8Text(ex.src), "")
				}
			}
		}
		return true
	})
	return env
}

// exprType returns the static type of expression n when the syntax or the
// environment shows it, a deferred type for a call, and "" otherwise. A
// constructor call `Foo()` is a call: pass 2 resolves it to the type.
func (ex *extractor) exprType(env *goextract.TypeEnv, n *tree_sitter.Node) string {
	switch n.Kind() {
	case "line_string_literal", "multi_line_string_literal", "raw_string_literal":
		return "String"
	case "integer_literal", "hex_literal", "oct_literal", "bin_literal":
		return "Int"
	case "real_literal":
		return "Double"
	case "boolean_literal":
		return "Bool"
	case "call_expression":
		return goextract.DeferredType(n)
	case "simple_identifier":
		typ, _ := env.Lookup(n.Utf8Text(ex.src))
		return typ
	case "self_expression":
		typ, _ := env.Lookup("self")
		return typ
	case "as_expression":
		return typeAnnotationOf(n, ex.src)
	case "navigation_expression":
		target, suffix := n.ChildByFieldName("target"), n.ChildByFieldName("suffix")
		if target != nil && suffix != nil && target.Kind() == "self_expression" {
			if m := suffix.ChildByFieldName("suffix"); m != nil {
				typ, _ := env.Lookup(m.Utf8Text(ex.src))
				return typ
			}
		}
	case "tuple_expression":
		if n.NamedChildCount() == 1 {
			return ex.exprType(env, n.NamedChild(0))
		}
	}
	return ""
}

// callArgs reads a call's value_arguments: each argument's label as
// written, `_` for none, and its expression's type.
func (ex *extractor) callArgs(env *goextract.TypeEnv, call *tree_sitter.Node) goextract.CallArgs {
	var ca goextract.CallArgs
	suffix := goextract.FirstNamedChildOfKind(call, "call_suffix")
	if suffix == nil {
		return ca
	}
	args := goextract.FirstNamedChildOfKind(suffix, "value_arguments")
	if args == nil {
		return ca
	}
	for i := uint(0); i < args.NamedChildCount(); i++ {
		a := args.NamedChild(i)
		if a.Kind() != "value_argument" {
			continue
		}
		ca.Count++
		label := ""
		if l := a.ChildByFieldName("name"); l != nil {
			label = l.Utf8Text(ex.src)
		} else if l := goextract.FirstNamedChildOfKind(a, "value_argument_label"); l != nil {
			label = l.Utf8Text(ex.src)
		}
		typ := ""
		if v := a.ChildByFieldName("value"); v != nil {
			typ = ex.exprType(env, v)
		} else if k := a.NamedChildCount(); k > 0 {
			typ = ex.exprType(env, a.NamedChild(k-1))
		}
		ca.Types = append(ca.Types, typ)
		ca.Labels = append(ca.Labels, label)
	}
	return ca
}

// receiver classifies the target of a navigation call (D17, D18).
func (ex *extractor) receiver(env *goextract.TypeEnv, target *tree_sitter.Node) (pkgAlias, receiverType string) {
	if target == nil {
		return "", ""
	}
	switch target.Kind() {
	case "simple_identifier":
		return goextract.Receiver(env, ex.result.Imports, target.Utf8Text(ex.src))
	case "self_expression":
		typ, _ := env.Lookup("self")
		return "", typ
	case "super_expression":
		return "", ""
	}
	if typ := ex.exprType(env, target); typ != "" {
		return "", typ
	}
	return goextract.UnknownAlias(target.Kind()), ""
}
