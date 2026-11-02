package kotlinextract

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// enclosingClass returns the class declaration whose body holds decl, or
// nil for a top-level function.
func enclosingClass(decl *tree_sitter.Node) *tree_sitter.Node {
	for p := decl.Parent(); p != nil; p = p.Parent() {
		switch p.Kind() {
		case "class_declaration", "object_declaration":
			return p
		case "function_declaration":
			return nil
		}
	}
	return nil
}

// typeOfParameterLike returns the type child of a parameter or property
// node: the first named child whose kind ends in _type.
func typeOfParameterLike(n *tree_sitter.Node, src []byte) string {
	for i := uint(0); i < n.NamedChildCount(); i++ {
		if c := n.NamedChild(i); strings.HasSuffix(c.Kind(), "_type") {
			return c.Utf8Text(src)
		}
	}
	return ""
}

// fieldsOf returns the environment of a class's properties: the
// properties of its body and the val or var parameters of its primary
// constructor.
func (ex *extractor) fieldsOf(class *tree_sitter.Node) *goextract.TypeEnv {
	env := goextract.NewTypeEnv(nil)
	if class == nil {
		return env
	}
	if ctor := firstChildOfKind(class, "primary_constructor"); ctor != nil {
		if ps := firstChildOfKind(ctor, "class_parameters"); ps != nil {
			for i := uint(0); i < ps.NamedChildCount(); i++ {
				p := ps.NamedChild(i)
				if p.Kind() != "class_parameter" {
					continue
				}
				if id := goextract.FirstNamedChildOfKind(p, "identifier", "simple_identifier"); id != nil {
					env.Declare(id.Utf8Text(ex.src), typeOfParameterLike(p, ex.src))
				}
			}
		}
	}
	if body := firstChildOfKind(class, "class_body"); body != nil {
		for i := uint(0); i < body.NamedChildCount(); i++ {
			if p := body.NamedChild(i); p.Kind() == "property_declaration" {
				ex.declareProperty(env, p)
			}
		}
	}
	return env
}

// declareProperty declares a val or var: its declared type, or the type of
// its initializer when exprType knows it.
func (ex *extractor) declareProperty(env *goextract.TypeEnv, p *tree_sitter.Node) {
	vd := firstChildOfKind(p, "variable_declaration")
	if vd == nil {
		return
	}
	id := goextract.FirstNamedChildOfKind(vd, "identifier", "simple_identifier")
	if id == nil {
		return
	}
	typ := typeOfParameterLike(vd, ex.src)
	if typ == "" {
		for i := uint(0); i < p.NamedChildCount(); i++ {
			if c := p.NamedChild(i); c.Kind() != "variable_declaration" && c.Kind() != "modifiers" {
				typ = ex.exprType(env, c)
			}
		}
	}
	env.Declare(id.Utf8Text(ex.src), typ)
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
	}
	env := goextract.NewTypeEnv(fields)
	env.Declare("this", className)
	params, _ := collectParams(decl, ex.src)
	for _, p := range params {
		env.Declare(p.Name, p.Type)
	}
	body := firstChildOfKind(decl, "function_body")
	if body == nil {
		return env
	}
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "property_declaration":
			ex.declareProperty(env, n)
		case "lambda_parameters":
			for i := uint(0); i < n.NamedChildCount(); i++ {
				if id := goextract.FirstNamedChildOfKind(n.NamedChild(i), "identifier", "simple_identifier"); id != nil {
					env.Declare(id.Utf8Text(ex.src), "")
				}
			}
		case "for_statement":
			if vd := firstChildOfKind(n, "variable_declaration"); vd != nil {
				if id := goextract.FirstNamedChildOfKind(vd, "identifier", "simple_identifier"); id != nil {
					env.Declare(id.Utf8Text(ex.src), typeOfParameterLike(vd, ex.src))
				}
			}
		}
		return true
	})
	return env
}

// exprType returns the static type of expression n when the syntax or the
// environment shows it, a deferred type for a call, and "" otherwise. A
// constructor call `Foo()` is a call: pass 2 resolves it to the class.
func (ex *extractor) exprType(env *goextract.TypeEnv, n *tree_sitter.Node) string {
	switch n.Kind() {
	case "string_literal":
		return "String"
	case "number_literal", "integer_literal", "hex_literal", "bin_literal":
		if strings.HasSuffix(strings.ToUpper(n.Utf8Text(ex.src)), "L") {
			return "Long"
		}
		return "Int"
	case "long_literal":
		return "Long"
	case "float_literal", "real_literal":
		if strings.HasSuffix(strings.ToLower(n.Utf8Text(ex.src)), "f") {
			return "Float"
		}
		return "Double"
	case "boolean_literal":
		return "Boolean"
	case "character_literal":
		return "Char"
	case "call_expression":
		return goextract.DeferredType(n)
	case "identifier", "simple_identifier":
		// The grammar parses the keywords true and false as identifiers.
		switch text := n.Utf8Text(ex.src); text {
		case "true", "false":
			return "Boolean"
		case "null":
			return ""
		default:
			typ, _ := env.Lookup(text)
			return typ
		}
	case "this_expression":
		typ, _ := env.Lookup("this")
		return typ
	case "as_expression":
		return typeOfParameterLike(n, ex.src)
	case "navigation_expression":
		if n.NamedChildCount() >= 2 && n.NamedChild(0).Kind() == "this_expression" {
			if suffix := n.NamedChild(n.NamedChildCount() - 1); suffix.Kind() == "navigation_suffix" || suffix.Kind() == "identifier" || suffix.Kind() == "simple_identifier" {
				name := suffix
				if suffix.Kind() == "navigation_suffix" && suffix.NamedChildCount() > 0 {
					name = suffix.NamedChild(0)
				}
				typ, _ := env.Lookup(name.Utf8Text(ex.src))
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

// callArgs reads the value_arguments of a call: each value_argument's
// expression, its name as a label when written (`flag = true` is two
// identifiers), and a spread_expression `*xs` as spread.
func (ex *extractor) callArgs(env *goextract.TypeEnv, call *tree_sitter.Node) goextract.CallArgs {
	var ca goextract.CallArgs
	args := firstChildOfKind(call, "value_arguments")
	if args == nil {
		return ca
	}
	labeled := false
	for i := uint(0); i < args.NamedChildCount(); i++ {
		a := args.NamedChild(i)
		if a.Kind() != "value_argument" {
			continue
		}
		ca.Count++
		label, typ := "", ""
		switch k := a.NamedChildCount(); {
		case k == 0:
		case a.NamedChild(0).Kind() == "spread_expression":
			ca.Spread = true
		case k >= 2 && (a.NamedChild(0).Kind() == "identifier" || a.NamedChild(0).Kind() == "simple_identifier"):
			label = a.NamedChild(0).Utf8Text(ex.src)
			labeled = true
			typ = ex.exprType(env, a.NamedChild(k-1))
		default:
			typ = ex.exprType(env, a.NamedChild(k-1))
		}
		ca.Types = append(ca.Types, typ)
		ca.Labels = append(ca.Labels, label)
	}
	if !labeled {
		ca.Labels = nil
	}
	return ca
}

// receiver classifies the receiver of a navigation call (D17, D18).
func (ex *extractor) receiver(env *goextract.TypeEnv, r *tree_sitter.Node) (pkgAlias, receiverType string) {
	if r == nil {
		return "", ""
	}
	switch r.Kind() {
	case "identifier", "simple_identifier":
		return goextract.Receiver(env, ex.result.Imports, r.Utf8Text(ex.src))
	case "this_expression":
		typ, _ := env.Lookup("this")
		return "", typ
	case "super_expression":
		return "", ""
	}
	if typ := ex.exprType(env, r); typ != "" {
		return "", typ
	}
	return goextract.UnknownAlias(r.Kind()), ""
}
