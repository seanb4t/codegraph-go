package pyextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// enclosingClass returns the name of the class whose body holds decl, or
// "" for a module-level function.
func (ex *extractor) enclosingClass(decl *tree_sitter.Node) string {
	for p := decl.Parent(); p != nil; p = p.Parent() {
		if p.Kind() == "class_definition" {
			if n := p.ChildByFieldName("name"); n != nil {
				return n.Utf8Text(ex.src)
			}
			return ""
		}
		if p.Kind() == "function_definition" {
			return ""
		}
	}
	return ""
}

// fieldsOf returns the environment of a class body's attributes: a class
// level annotated assignment `x: T = ...`, and `self.x: T` or `self.x =
// expr` in any method of the class.
func (ex *extractor) fieldsOf(class *tree_sitter.Node) *goextract.TypeEnv {
	env := goextract.NewTypeEnv(nil)
	if class == nil {
		return env
	}
	body := class.ChildByFieldName("body")
	if body == nil {
		return env
	}
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		if n.Kind() == "class_definition" && n != class {
			return false
		}
		if n.Kind() != "assignment" {
			return true
		}
		left := n.ChildByFieldName("left")
		if left == nil {
			return true
		}
		name := ""
		switch left.Kind() {
		case "identifier":
			if p := n.Parent(); p != nil && p.Kind() == "expression_statement" && sameNode(p.Parent(), body) {
				name = left.Utf8Text(ex.src)
			}
		case "attribute":
			obj, attr := left.ChildByFieldName("object"), left.ChildByFieldName("attribute")
			if obj != nil && attr != nil && obj.Kind() == "identifier" && obj.Utf8Text(ex.src) == "self" {
				name = attr.Utf8Text(ex.src)
			}
		}
		if name == "" {
			return true
		}
		typ := ""
		if t := n.ChildByFieldName("type"); t != nil {
			typ = t.Utf8Text(ex.src)
		} else if v := n.ChildByFieldName("right"); v != nil {
			typ = ex.exprType(env, v)
		}
		env.Declare(name, typ)
		return true
	})
	return env
}

// bodyEnv returns the environment of one function body: `self` or `cls`,
// the parameters, and every name the body assigns.
func (ex *extractor) bodyEnv(decl *tree_sitter.Node) *goextract.TypeEnv {
	var fields *goextract.TypeEnv
	className := ex.enclosingClass(decl)
	if className != "" {
		for p := decl.Parent(); p != nil; p = p.Parent() {
			if p.Kind() == "class_definition" {
				fields = ex.fieldsOf(p)
				break
			}
		}
	}
	env := goextract.NewTypeEnv(fields)
	params, _ := collectParams(decl, ex.src)
	for i, p := range params {
		if i == 0 && className != "" && (p.Name == "self" || p.Name == "cls") && p.Type == "" {
			env.Declare(p.Name, className)
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
		case "function_definition", "class_definition":
			return false
		case "assignment":
			left := n.ChildByFieldName("left")
			if left == nil || left.Kind() != "identifier" {
				return true
			}
			typ := ""
			if t := n.ChildByFieldName("type"); t != nil {
				typ = t.Utf8Text(ex.src)
			} else if v := n.ChildByFieldName("right"); v != nil {
				typ = ex.exprType(env, v)
			}
			env.Declare(left.Utf8Text(ex.src), typ)
		case "for_statement", "with_item", "as_pattern":
			if left := n.ChildByFieldName("left"); left != nil && left.Kind() == "identifier" {
				env.Declare(left.Utf8Text(ex.src), "")
			}
			if alias := n.ChildByFieldName("alias"); alias != nil {
				env.Declare(alias.Utf8Text(ex.src), "")
			}
		case "lambda":
			if ps := n.ChildByFieldName("parameters"); ps != nil {
				for i := uint(0); i < ps.NamedChildCount(); i++ {
					if id := ps.NamedChild(i); id.Kind() == "identifier" {
						env.Declare(id.Utf8Text(ex.src), "")
					}
				}
			}
		}
		return true
	})
	return env
}

// exprType returns the static type of expression n when the syntax or the
// environment shows it, a deferred type for a call, and "" otherwise. A
// call of a class, `Foo()`, is a call for the extractor: pass 2 resolves
// it to the class and takes the class as the type.
func (ex *extractor) exprType(env *goextract.TypeEnv, n *tree_sitter.Node) string {
	switch n.Kind() {
	case "string", "concatenated_string":
		return "str"
	case "integer":
		return "int"
	case "float":
		return "float"
	case "true", "false":
		return "bool"
	case "list", "list_comprehension":
		return "list"
	case "dictionary", "dictionary_comprehension":
		return "dict"
	case "tuple":
		return "tuple"
	case "set":
		return "set"
	case "call":
		return goextract.DeferredType(n)
	case "identifier":
		typ, _ := env.Lookup(n.Utf8Text(ex.src))
		return typ
	case "attribute":
		obj, attr := n.ChildByFieldName("object"), n.ChildByFieldName("attribute")
		if obj != nil && attr != nil && obj.Kind() == "identifier" && (obj.Utf8Text(ex.src) == "self" || obj.Utf8Text(ex.src) == "cls") {
			typ, _ := env.Lookup(attr.Utf8Text(ex.src))
			return typ
		}
	case "parenthesized_expression", "await":
		if n.NamedChildCount() > 0 {
			return ex.exprType(env, n.NamedChild(0))
		}
	}
	return ""
}

// callArgs reads an argument_list: positional arguments, keyword
// arguments with their names as labels, and splats as spread.
func (ex *extractor) callArgs(env *goextract.TypeEnv, args *tree_sitter.Node) goextract.CallArgs {
	var ca goextract.CallArgs
	if args == nil {
		return ca
	}
	labeled := false
	for i := uint(0); i < args.NamedChildCount(); i++ {
		a := args.NamedChild(i)
		ca.Count++
		switch a.Kind() {
		case "list_splat", "dictionary_splat":
			ca.Spread = true
			ca.Types = append(ca.Types, "")
			ca.Labels = append(ca.Labels, "")
		case "keyword_argument":
			labeled = true
			label := ""
			if n := a.ChildByFieldName("name"); n != nil {
				label = n.Utf8Text(ex.src)
			}
			typ := ""
			if v := a.ChildByFieldName("value"); v != nil {
				typ = ex.exprType(env, v)
			}
			ca.Types = append(ca.Types, typ)
			ca.Labels = append(ca.Labels, label)
		default:
			ca.Types = append(ca.Types, ex.exprType(env, a))
			ca.Labels = append(ca.Labels, "")
		}
	}
	if !labeled {
		ca.Labels = nil
	}
	return ca
}

// receiver classifies the object of an attribute call: what it names and
// its static type (D17, D18).
func (ex *extractor) receiver(env *goextract.TypeEnv, object *tree_sitter.Node) (pkgAlias, receiverType string) {
	if object == nil {
		return "", ""
	}
	if object.Kind() == "identifier" {
		return goextract.Receiver(env, ex.result.Imports, object.Utf8Text(ex.src))
	}
	if typ := ex.exprType(env, object); typ != "" {
		return "", typ
	}
	return goextract.UnknownAlias(object.Kind()), ""
}

// sameNode reports whether a and b are the same syntax node. Two Node
// values that name one node are different Go values, so == cannot
// compare them.
func sameNode(a, b *tree_sitter.Node) bool {
	return a != nil && b != nil && a.Kind() == b.Kind() && a.StartByte() == b.StartByte() && a.EndByte() == b.EndByte()
}
