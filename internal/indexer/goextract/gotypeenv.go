package goextract

import tree_sitter "github.com/tree-sitter/go-tree-sitter"

// structFieldsOf returns the environment of a struct_type's fields, so a
// method body can type `s.field` through its receiver.
func (ex *extractor) structFieldsOf(structType *tree_sitter.Node) *TypeEnv {
	env := NewTypeEnv(nil)
	list := FirstNamedChildOfKind(structType, "field_declaration_list")
	if list == nil {
		return env
	}
	for i := uint(0); i < list.NamedChildCount(); i++ {
		fd := list.NamedChild(i)
		if fd.Kind() != "field_declaration" {
			continue
		}
		typ := ""
		if t := fd.ChildByFieldName("type"); t != nil {
			typ = t.Utf8Text(ex.src)
		}
		for j := uint(0); j < fd.NamedChildCount(); j++ {
			if c := fd.NamedChild(j); c.Kind() == "field_identifier" {
				env.Declare(c.Utf8Text(ex.src), typ)
			}
		}
	}
	return env
}

// bodyEnv returns the environment of one function or method body: the
// receiver, the parameters, and every local declared in the body. A
// short variable declaration takes the type of its right-hand side when
// exprType knows it.
func (ex *extractor) bodyEnv(decl *tree_sitter.Node) *TypeEnv {
	env := NewTypeEnv(nil)
	if recv := decl.ChildByFieldName("receiver"); recv != nil {
		if pd := FirstNamedChildOfKind(recv, "parameter_declaration"); pd != nil {
			typ := ""
			if t := pd.ChildByFieldName("type"); t != nil {
				typ = t.Utf8Text(ex.src)
			}
			if name := pd.ChildByFieldName("name"); name != nil {
				// A Go field is reached through the receiver only, so the
				// fields are not bare names of the body.
				env.Declare(name.Utf8Text(ex.src), typ)
			}
		}
	}
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
		case "var_spec":
			typ := ""
			if t := n.ChildByFieldName("type"); t != nil {
				typ = t.Utf8Text(ex.src)
			}
			var names []string
			for i := uint(0); i < n.NamedChildCount(); i++ {
				if c := n.NamedChild(i); c.Kind() == "identifier" {
					names = append(names, c.Utf8Text(ex.src))
				}
			}
			values := n.ChildByFieldName("value")
			for i, name := range names {
				t := typ
				if t == "" && values != nil && uint(i) < values.NamedChildCount() {
					t = ex.exprType(env, values.NamedChild(uint(i)))
				}
				env.DeclareIn(name, t, uint32(n.EndByte()), ScopeEnd(n, "go"))
			}
		case "short_var_declaration":
			left, right := n.ChildByFieldName("left"), n.ChildByFieldName("right")
			if left == nil {
				return true
			}
			for i := uint(0); i < left.NamedChildCount(); i++ {
				id := left.NamedChild(i)
				if id.Kind() != "identifier" {
					continue
				}
				t := ""
				if right != nil && right.NamedChildCount() == left.NamedChildCount() {
					t = ex.exprType(env, right.NamedChild(i))
				}
				// The scope of `x := f(x)` starts after the statement: the
				// right side still sees an outer x or an import alias x.
				env.DeclareIn(id.Utf8Text(ex.src), t, uint32(n.EndByte()), ScopeEnd(n, "go"))
			}
		case "range_clause":
			if left := n.ChildByFieldName("left"); left != nil {
				for i := uint(0); i < left.NamedChildCount(); i++ {
					if id := left.NamedChild(i); id.Kind() == "identifier" {
						env.DeclareIn(id.Utf8Text(ex.src), "", uint32(n.StartByte()), ScopeEnd(n, "go"))
					}
				}
			}
		case "func_literal":
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
func (ex *extractor) exprType(env *TypeEnv, n *tree_sitter.Node) string {
	switch n.Kind() {
	case "interpreted_string_literal", "raw_string_literal":
		return "string"
	case "int_literal":
		return "int"
	case "float_literal":
		return "float64"
	case "rune_literal":
		return "rune"
	case "true", "false":
		return "bool"
	case "composite_literal", "type_assertion_expression":
		if t := n.ChildByFieldName("type"); t != nil {
			return t.Utf8Text(ex.src)
		}
	case "unary_expression":
		if op := n.ChildByFieldName("operand"); op != nil && op.Kind() == "composite_literal" {
			return ex.exprType(env, op)
		}
	case "call_expression":
		if fn := n.ChildByFieldName("function"); fn != nil && fn.Kind() == "identifier" && fn.Utf8Text(ex.src) == "new" {
			if args := n.ChildByFieldName("arguments"); args != nil && args.NamedChildCount() > 0 {
				return args.NamedChild(0).Utf8Text(ex.src)
			}
		}
		return DeferredType(n)
	case "identifier":
		typ, _ := env.LookupAt(n.Utf8Text(ex.src), uint32(n.StartByte()))
		return typ
	case "selector_expression":
		operand, field := n.ChildByFieldName("operand"), n.ChildByFieldName("field")
		if operand != nil && field != nil && operand.Kind() == "identifier" {
			if typ, _ := env.LookupAt(operand.Utf8Text(ex.src), uint32(operand.StartByte())); typ != "" {
				if fields := ex.fields[TypeName(typ)]; fields != nil {
					t, _ := fields.Lookup(field.Utf8Text(ex.src))
					return t
				}
			}
		}
	case "parenthesized_expression":
		if n.NamedChildCount() > 0 {
			return ex.exprType(env, n.NamedChild(0))
		}
	}
	return ""
}

// callArgs reads an argument_list. A `...` after the last argument spreads
// it.
func (ex *extractor) callArgs(env *TypeEnv, args *tree_sitter.Node) CallArgs {
	var ca CallArgs
	if args == nil {
		return ca
	}
	for i := uint(0); i < args.ChildCount(); i++ {
		c := args.Child(i)
		if !c.IsNamed() {
			if c.Kind() == "..." {
				ca.Spread = true
			}
			continue
		}
		if c.Kind() == "variadic_argument" {
			ca.Spread = true
			ca.Count++
			typ := ""
			if c.NamedChildCount() > 0 {
				typ = ex.exprType(env, c.NamedChild(0))
			}
			ca.Types = append(ca.Types, typ)
			continue
		}
		ca.Count++
		ca.Types = append(ca.Types, ex.exprType(env, c))
	}
	return ca
}

// receiver classifies the operand of a selector: what it names and its
// static type (D17, D18).
func (ex *extractor) receiver(env *TypeEnv, operand *tree_sitter.Node) (pkgAlias, receiverType string) {
	switch {
	case operand == nil:
		return "", ""
	case operand.Kind() == "identifier":
		return ReceiverAt(env, ex.result.Imports, operand.Utf8Text(ex.src), uint32(operand.StartByte()))
	}
	if typ := ex.exprType(env, operand); typ != "" {
		return "", typ
	}
	return UnknownAlias(operand.Kind()), ""
}
