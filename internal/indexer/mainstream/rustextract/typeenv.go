package rustextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// enclosingImplType returns the type an impl block implements, for a
// function item inside it, or "".
func (ex *extractor) enclosingImplType(decl *tree_sitter.Node) string {
	for p := decl.Parent(); p != nil; p = p.Parent() {
		if p.Kind() == "impl_item" {
			if t := p.ChildByFieldName("type"); t != nil {
				if name, ok := rustTypeName(t, ex.src); ok {
					return name
				}
			}
			return ""
		}
		if p.Kind() == "function_item" {
			return ""
		}
	}
	return ""
}

// structFieldsOf returns the environment of a struct item's fields.
func (ex *extractor) structFieldsOf(structItem *tree_sitter.Node) *goextract.TypeEnv {
	env := goextract.NewTypeEnv(nil)
	body := structItem.ChildByFieldName("body")
	if body == nil {
		return env
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		f := body.NamedChild(i)
		if f.Kind() != "field_declaration" {
			continue
		}
		name, typ := f.ChildByFieldName("name"), f.ChildByFieldName("type")
		if name != nil {
			t := ""
			if typ != nil {
				t = typ.Utf8Text(ex.src)
			}
			env.Declare(name.Utf8Text(ex.src), t)
		}
	}
	return env
}

// bodyEnv returns the environment of one function body: self, the
// parameters, and every let binding.
func (ex *extractor) bodyEnv(decl *tree_sitter.Node) *goextract.TypeEnv {
	implType := ex.enclosingImplType(decl)
	env := goextract.NewTypeEnv(ex.fields[implType])
	env.Declare("self", implType)
	params, _ := collectParams(decl, ex.src)
	for _, p := range params {
		if p.Name == "self" {
			continue
		}
		env.Declare(p.Name, p.Type)
	}
	body := decl.ChildByFieldName("body")
	if body == nil {
		return env
	}
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "function_item", "closure_expression":
			if n.Kind() == "closure_expression" {
				if ps := n.ChildByFieldName("parameters"); ps != nil {
					for i := uint(0); i < ps.NamedChildCount(); i++ {
						p := ps.NamedChild(i)
						if p.Kind() == "identifier" {
							env.Declare(p.Utf8Text(ex.src), "")
						} else if pat := p.ChildByFieldName("pattern"); pat != nil && pat.Kind() == "identifier" {
							t := ""
							if ty := p.ChildByFieldName("type"); ty != nil {
								t = ty.Utf8Text(ex.src)
							}
							env.Declare(pat.Utf8Text(ex.src), t)
						}
					}
				}
				return true
			}
			return false
		case "let_declaration":
			pat := n.ChildByFieldName("pattern")
			if pat == nil {
				return true
			}
			if pat.Kind() == "mut_pattern" && pat.NamedChildCount() > 0 {
				pat = pat.NamedChild(0)
			}
			if pat.Kind() != "identifier" {
				return true
			}
			typ := ""
			if t := n.ChildByFieldName("type"); t != nil {
				typ = t.Utf8Text(ex.src)
			} else if v := n.ChildByFieldName("value"); v != nil {
				typ = ex.exprType(env, v)
			}
			env.Declare(pat.Utf8Text(ex.src), typ)
		case "for_expression":
			if pat := n.ChildByFieldName("pattern"); pat != nil && pat.Kind() == "identifier" {
				env.Declare(pat.Utf8Text(ex.src), "")
			}
		}
		return true
	})
	return env
}

// exprType returns the static type of expression n when the syntax or the
// environment shows it, a deferred type for a call, and "" otherwise. A
// path call such as `Foo::new()` is a call: pass 2 resolves it.
func (ex *extractor) exprType(env *goextract.TypeEnv, n *tree_sitter.Node) string {
	switch n.Kind() {
	case "string_literal", "raw_string_literal":
		return "str"
	case "char_literal":
		return "char"
	case "boolean_literal":
		return "bool"
	case "struct_expression":
		if name := n.ChildByFieldName("name"); name != nil {
			if t, ok := rustTypeName(name, ex.src); ok {
				return t
			}
			return name.Utf8Text(ex.src)
		}
	case "call_expression", "macro_invocation":
		if n.Kind() == "macro_invocation" {
			return ""
		}
		return goextract.DeferredType(n)
	case "identifier":
		typ, _ := env.Lookup(n.Utf8Text(ex.src))
		return typ
	case "self":
		typ, _ := env.Lookup("self")
		return typ
	case "reference_expression", "unary_expression":
		if v := n.ChildByFieldName("value"); v != nil {
			return ex.exprType(env, v)
		}
		if n.NamedChildCount() > 0 {
			return ex.exprType(env, n.NamedChild(n.NamedChildCount()-1))
		}
	case "type_cast_expression":
		if t := n.ChildByFieldName("type"); t != nil {
			return t.Utf8Text(ex.src)
		}
	case "field_expression":
		value, field := n.ChildByFieldName("value"), n.ChildByFieldName("field")
		if value != nil && field != nil && value.Kind() == "self" {
			typ, _ := env.Lookup(field.Utf8Text(ex.src))
			return typ
		}
	case "parenthesized_expression":
		if n.NamedChildCount() > 0 {
			return ex.exprType(env, n.NamedChild(0))
		}
	}
	return ""
}

// callArgs reads an arguments node.
func (ex *extractor) callArgs(env *goextract.TypeEnv, args *tree_sitter.Node) goextract.CallArgs {
	var ca goextract.CallArgs
	if args == nil {
		return ca
	}
	for i := uint(0); i < args.NamedChildCount(); i++ {
		a := args.NamedChild(i)
		if a.Kind() == "attribute_item" {
			continue
		}
		ca.Count++
		ca.Types = append(ca.Types, ex.exprType(env, a))
	}
	return ca
}

// receiver classifies the value of a field call (D17, D18).
func (ex *extractor) receiver(env *goextract.TypeEnv, value *tree_sitter.Node) (pkgAlias, receiverType string) {
	if value == nil {
		return "", ""
	}
	switch value.Kind() {
	case "identifier":
		return goextract.Receiver(env, ex.result.Imports, value.Utf8Text(ex.src))
	case "self":
		typ, _ := env.Lookup("self")
		return "", typ
	}
	if typ := ex.exprType(env, value); typ != "" {
		return "", typ
	}
	return goextract.UnknownAlias(value.Kind()), ""
}
