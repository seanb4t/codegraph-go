package phpextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// enclosingClass returns the class, interface, trait or enum declaration
// whose body holds decl, or nil for a free function.
func enclosingClass(decl *tree_sitter.Node) *tree_sitter.Node {
	for p := decl.Parent(); p != nil; p = p.Parent() {
		switch p.Kind() {
		case "class_declaration", "interface_declaration", "trait_declaration", "enum_declaration":
			return p
		case "function_definition", "method_declaration":
			if p != decl {
				return nil
			}
		}
	}
	return nil
}

// fieldsOf returns the environment of a class body's properties, keyed by
// `$name`, with their declared types.
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
		m := body.NamedChild(i)
		switch m.Kind() {
		case "property_declaration":
			typ := ""
			if t := m.ChildByFieldName("type"); t != nil {
				typ = t.Utf8Text(ex.src)
			}
			for j := uint(0); j < m.NamedChildCount(); j++ {
				if pe := m.NamedChild(j); pe.Kind() == "property_element" {
					if vn := goextract.FirstNamedChildOfKind(pe, "variable_name"); vn != nil {
						env.Declare(vn.Utf8Text(ex.src), typ)
					}
				}
			}
		case "method_declaration":
			// Constructor property promotion: `public function
			// __construct(private Logger $logger)`.
			if n := m.ChildByFieldName("name"); n == nil || n.Utf8Text(ex.src) != "__construct" {
				continue
			}
			if ps := m.ChildByFieldName("parameters"); ps != nil {
				for j := uint(0); j < ps.NamedChildCount(); j++ {
					p := ps.NamedChild(j)
					if p.Kind() != "property_promotion_parameter" {
						continue
					}
					typ := ""
					if t := p.ChildByFieldName("type"); t != nil {
						typ = t.Utf8Text(ex.src)
					}
					if vn := p.ChildByFieldName("name"); vn != nil {
						env.Declare(vn.Utf8Text(ex.src), typ)
					}
				}
			}
		}
	}
	return env
}

// bodyEnv returns the environment of one function or method body. Names
// carry their `$`, as the source writes them.
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
	env.Declare("$this", className)
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
		case "function_definition", "method_declaration", "anonymous_function_creation_expression":
			if n != body {
				return false
			}
		case "assignment_expression":
			left := n.ChildByFieldName("left")
			if left == nil || left.Kind() != "variable_name" {
				return true
			}
			typ := ""
			if v := n.ChildByFieldName("right"); v != nil {
				typ = ex.exprType(env, v)
			}
			env.Declare(left.Utf8Text(ex.src), typ)
		case "foreach_statement":
			for _, f := range []string{"value", "key"} {
				if v := n.ChildByFieldName(f); v != nil && v.Kind() == "variable_name" {
					env.Declare(v.Utf8Text(ex.src), "")
				}
			}
		case "catch_clause":
			typ := ""
			if t := n.ChildByFieldName("type"); t != nil {
				typ = t.Utf8Text(ex.src)
			}
			if v := n.ChildByFieldName("name"); v != nil {
				env.Declare(v.Utf8Text(ex.src), typ)
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
	case "string", "encapsed_string", "heredoc", "nowdoc":
		return "string"
	case "integer":
		return "int"
	case "float":
		return "float"
	case "boolean":
		return "bool"
	case "array_creation_expression":
		return "array"
	case "object_creation_expression":
		if c := goextract.FirstNamedChildOfKind(n, "name", "qualified_name", "relative_name"); c != nil {
			simple, _ := phpSimpleAndFullName(c, ex.src)
			if simple != "" {
				return simple
			}
			return c.Utf8Text(ex.src)
		}
	case "cast_expression":
		if t := n.ChildByFieldName("type"); t != nil {
			return t.Utf8Text(ex.src)
		}
	case "function_call_expression", "member_call_expression", "scoped_call_expression":
		return goextract.DeferredType(n)
	case "variable_name":
		typ, _ := env.Lookup(n.Utf8Text(ex.src))
		return typ
	case "member_access_expression":
		obj, name := n.ChildByFieldName("object"), n.ChildByFieldName("name")
		if obj != nil && name != nil && obj.Kind() == "variable_name" && phpVariableName(obj, ex.src) == "this" {
			typ, _ := env.Lookup("$" + name.Utf8Text(ex.src))
			return typ
		}
	case "parenthesized_expression":
		if n.NamedChildCount() > 0 {
			return ex.exprType(env, n.NamedChild(0))
		}
	}
	return ""
}

// callArgs reads an arguments node: each argument's expression, a named
// argument's name as its label, and `...$xs` as spread.
func (ex *extractor) callArgs(env *goextract.TypeEnv, args *tree_sitter.Node) goextract.CallArgs {
	var ca goextract.CallArgs
	if args == nil {
		return ca
	}
	labeled := false
	for i := uint(0); i < args.NamedChildCount(); i++ {
		a := args.NamedChild(i)
		switch a.Kind() {
		case "variadic_unpacking":
			ca.Spread = true
			ca.Count++
			ca.Types = append(ca.Types, "")
			ca.Labels = append(ca.Labels, "")
		case "argument":
			ca.Count++
			label := ""
			if nm := a.ChildByFieldName("name"); nm != nil {
				label = nm.Utf8Text(ex.src)
				labeled = true
			}
			typ := ""
			if k := a.NamedChildCount(); k > 0 {
				typ = ex.exprType(env, a.NamedChild(k-1))
			}
			ca.Types = append(ca.Types, typ)
			ca.Labels = append(ca.Labels, label)
		}
	}
	if !labeled {
		ca.Labels = nil
	}
	return ca
}

// receiver classifies the object of a member call: a variable through the
// environment, a class name as a candidate type, `$this` as the enclosing
// class (D17, D18).
func (ex *extractor) receiver(env *goextract.TypeEnv, object *tree_sitter.Node) (pkgAlias, receiverType string) {
	if object == nil {
		return "", ""
	}
	switch object.Kind() {
	case "variable_name":
		// A `$name` is a variable whatever the environment knows: an
		// undeclared one has an unknown type and cannot name a type.
		name := object.Utf8Text(ex.src)
		if typ, declared := env.Lookup(name); declared && typ != "" {
			return "", typ
		}
		return goextract.LocalAlias(name), ""
	case "name", "qualified_name", "relative_name":
		simple, _ := phpSimpleAndFullName(object, ex.src)
		if simple == "" {
			simple = object.Utf8Text(ex.src)
		}
		return goextract.Receiver(env, ex.result.Imports, simple)
	}
	if typ := ex.exprType(env, object); typ != "" {
		return "", typ
	}
	return goextract.UnknownAlias(object.Kind()), ""
}

// scope classifies the scope of a static call `Foo::bar()`: a class name,
// or `self` and `static` as the enclosing class. `parent` stays unknown
// and resolves through the supertype chain.
func (ex *extractor) scope(env *goextract.TypeEnv, scope *tree_sitter.Node) (pkgAlias, receiverType string) {
	switch scope.Kind() {
	case "relative_scope":
		switch scope.Utf8Text(ex.src) {
		case "self", "static":
			typ, _ := env.Lookup("$this")
			return "", typ
		}
		return "", ""
	case "variable_name":
		name := scope.Utf8Text(ex.src)
		if typ, declared := env.Lookup(name); declared && typ != "" {
			return "", typ
		}
		return goextract.LocalAlias(name), ""
	case "name", "qualified_name", "relative_name":
		simple, _ := phpSimpleAndFullName(scope, ex.src)
		if simple == "" {
			simple = scope.Utf8Text(ex.src)
		}
		return goextract.Receiver(env, ex.result.Imports, simple)
	}
	return goextract.UnknownAlias(scope.Kind()), ""
}
