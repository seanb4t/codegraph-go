package tsextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// fieldsOf returns the environment of a class body's fields: each field
// definition with its annotation or the type of its initializer, and
// each constructor parameter with an accessibility modifier, which
// TypeScript turns into a field.
func (ex *extractor) fieldsOf(body *tree_sitter.Node) *goextract.TypeEnv {
	env := goextract.NewTypeEnv(nil)
	if body == nil {
		return env
	}
	for i := uint(0); i < body.NamedChildCount(); i++ {
		m := body.NamedChild(i)
		switch m.Kind() {
		case "public_field_definition", "field_definition":
			name := m.ChildByFieldName("name")
			if name == nil {
				continue
			}
			typ := ""
			if t := m.ChildByFieldName("type"); t != nil {
				typ = annotationType(t, ex.src)
			} else if v := m.ChildByFieldName("value"); v != nil {
				typ = ex.exprType(env, v)
			}
			env.Declare(name.Utf8Text(ex.src), typ)
		case "method_definition":
			if n := m.ChildByFieldName("name"); n == nil || n.Utf8Text(ex.src) != "constructor" {
				continue
			}
			ps := m.ChildByFieldName("parameters")
			if ps == nil {
				continue
			}
			for j := uint(0); j < ps.NamedChildCount(); j++ {
				p := ps.NamedChild(j)
				if goextract.FirstNamedChildOfKind(p, "accessibility_modifier") == nil {
					continue
				}
				pat, t := p.ChildByFieldName("pattern"), p.ChildByFieldName("type")
				if pat == nil || pat.Kind() != "identifier" {
					continue
				}
				typ := ""
				if t != nil {
					typ = annotationType(t, ex.src)
				}
				env.Declare(pat.Utf8Text(ex.src), typ)
			}
		}
	}
	return env
}

// bodyEnv returns the environment of one function body: `this`, the
// parameters of sig, and every local declared in body.
func (ex *extractor) bodyEnv(fields *goextract.TypeEnv, typeName string, sig, body *tree_sitter.Node) *goextract.TypeEnv {
	env := goextract.NewTypeEnv(fields)
	env.Declare("this", typeName)
	params, _ := collectParams(sig, ex.src)
	for _, p := range params {
		env.Declare(p.Name, p.Type)
	}
	if body == nil {
		return env
	}
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "variable_declarator":
			name := n.ChildByFieldName("name")
			if name == nil || name.Kind() != "identifier" {
				return true
			}
			typ := ""
			if t := n.ChildByFieldName("type"); t != nil {
				typ = annotationType(t, ex.src)
			} else if v := n.ChildByFieldName("value"); v != nil {
				typ = ex.exprType(env, v)
			}
			env.DeclareIn(name.Utf8Text(ex.src), typ, uint32(n.EndByte()), goextract.ScopeEnd(n, ex.language))
		case "for_in_statement":
			if left := n.ChildByFieldName("left"); left != nil && left.Kind() == "identifier" {
				env.DeclareIn(left.Utf8Text(ex.src), "", uint32(n.StartByte()), uint32(n.EndByte()))
			}
		case "catch_clause":
			if p := n.ChildByFieldName("parameter"); p != nil && p.Kind() == "identifier" {
				env.DeclareIn(p.Utf8Text(ex.src), "", uint32(n.StartByte()), uint32(n.EndByte()))
			}
		case "arrow_function", "function_expression", "function_declaration", "method_definition":
			if n == body {
				return true
			}
			params, _ := collectParams(n, ex.src)
			for _, p := range params {
				env.DeclareIn(p.Name, p.Type, uint32(n.StartByte()), uint32(n.EndByte()))
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
	case "string", "template_string":
		return "string"
	case "number":
		return "number"
	case "true", "false":
		return "boolean"
	case "new_expression":
		if c := n.ChildByFieldName("constructor"); c != nil {
			switch c.Kind() {
			case "identifier":
				return c.Utf8Text(ex.src)
			case "member_expression":
				if p := c.ChildByFieldName("property"); p != nil {
					return p.Utf8Text(ex.src)
				}
			}
		}
	case "as_expression", "satisfies_expression":
		if n.NamedChildCount() > 1 {
			return n.NamedChild(1).Utf8Text(ex.src)
		}
	case "non_null_expression", "parenthesized_expression", "await_expression":
		if n.NamedChildCount() > 0 {
			return ex.exprType(env, n.NamedChild(0))
		}
	case "identifier":
		typ, _ := env.LookupAt(n.Utf8Text(ex.src), uint32(n.StartByte()))
		return typ
	case "this":
		typ, _ := env.Lookup("this")
		return typ
	case "call_expression":
		return goextract.DeferredType(n)
	case "member_expression":
		if obj := n.ChildByFieldName("object"); obj != nil && obj.Kind() == "this" {
			if p := n.ChildByFieldName("property"); p != nil {
				typ, _ := env.Lookup(p.Utf8Text(ex.src))
				return typ
			}
		}
	}
	return ""
}

// callArgs reads an arguments node. A spread_element spreads.
func (ex *extractor) callArgs(env *goextract.TypeEnv, args *tree_sitter.Node) goextract.CallArgs {
	var ca goextract.CallArgs
	if args == nil {
		return ca
	}
	for i := uint(0); i < args.NamedChildCount(); i++ {
		a := args.NamedChild(i)
		if a.Kind() == "spread_element" {
			ca.Spread = true
			ca.Count++
			ca.Types = append(ca.Types, "")
			continue
		}
		ca.Count++
		ca.Types = append(ca.Types, ex.exprType(env, a))
	}
	return ca
}

// receiver classifies the object of a member access: what it names and
// its static type (D17, D18). A named import gives its local alias and
// the target module's own name for the symbol, so pass 2 looks the type
// up in that module.
func (ex *extractor) receiver(env *goextract.TypeEnv, object *tree_sitter.Node) (pkgAlias, receiverType string) {
	if object == nil {
		return "", ""
	}
	switch object.Kind() {
	case "this":
		typ, _ := env.Lookup("this")
		return "", typ
	case "super":
		return "", ""
	case "identifier":
		name := object.Utf8Text(ex.src)
		if _, declared := env.LookupAt(name, uint32(object.StartByte())); !declared {
			if origin, ok := ex.namedImportOrigin[name]; ok {
				return name, origin
			}
		}
		return goextract.ReceiverAt(env, ex.result.Imports, name, uint32(object.StartByte()))
	}
	if typ := ex.exprType(env, object); typ != "" {
		return "", typ
	}
	return goextract.UnknownAlias(object.Kind()), ""
}

// receiverAlias is receiver's alias alone, for the references that carry
// no receiver type.
func (ex *extractor) receiverAlias(object *tree_sitter.Node) string {
	alias, _ := ex.receiver(ex.env, object)
	return alias
}
