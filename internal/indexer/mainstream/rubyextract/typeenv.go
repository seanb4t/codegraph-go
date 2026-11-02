package rubyextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
)

// enclosingScope returns the class or module whose body holds decl, or
// nil for a top-level method.
func enclosingScope(decl *tree_sitter.Node) *tree_sitter.Node {
	for p := decl.Parent(); p != nil; p = p.Parent() {
		switch p.Kind() {
		case "class", "module", "singleton_class":
			return p
		case "method", "singleton_method":
			if p != decl {
				return nil
			}
		}
	}
	return nil
}

// bodyEnv returns the environment of one method body. Ruby writes no
// types, so a name has a type only when its initializer shows one:
// `x = Foo.new` gives `Foo`, and a literal gives its class.
func (ex *extractor) bodyEnv(decl *tree_sitter.Node) *goextract.TypeEnv {
	env := goextract.NewTypeEnv(nil)
	scopeName := ""
	if s := enclosingScope(decl); s != nil {
		if n := s.ChildByFieldName("name"); n != nil {
			scopeName = rubyConstantName(n, ex.src)
		}
	}
	env.Declare("self", scopeName)
	for _, p := range collectParams(decl, ex.src) {
		env.Declare(p.Name, "")
	}
	body := decl.ChildByFieldName("body")
	if body == nil {
		return env
	}
	walkDescendants(body, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "method", "singleton_method", "class", "module":
			return false
		case "assignment":
			left := n.ChildByFieldName("left")
			if left == nil || (left.Kind() != "identifier" && left.Kind() != "instance_variable") {
				return true
			}
			typ := ""
			if v := n.ChildByFieldName("right"); v != nil {
				typ = ex.exprType(env, v)
			}
			env.Declare(left.Utf8Text(ex.src), typ)
		case "block_parameters", "lambda_parameters":
			for i := uint(0); i < n.NamedChildCount(); i++ {
				if id := n.NamedChild(i); id.Kind() == "identifier" {
					env.Declare(id.Utf8Text(ex.src), "")
				}
			}
		}
		return true
	})
	return env
}

// exprType returns the static type of expression n when the syntax or the
// environment shows it, a deferred type for a call, and "" otherwise.
// `Foo.new(...)` gives Foo directly.
func (ex *extractor) exprType(env *goextract.TypeEnv, n *tree_sitter.Node) string {
	switch n.Kind() {
	case "string", "string_array", "heredoc_body", "chained_string":
		return "String"
	case "symbol", "simple_symbol", "delimited_symbol":
		return "Symbol"
	case "integer":
		return "Integer"
	case "float":
		return "Float"
	case "true", "false":
		return "Boolean"
	case "array":
		return "Array"
	case "hash":
		return "Hash"
	case "call":
		recv, method := n.ChildByFieldName("receiver"), n.ChildByFieldName("method")
		if recv != nil && method != nil && method.Utf8Text(ex.src) == "new" && recv.Kind() == "constant" {
			return recv.Utf8Text(ex.src)
		}
		return goextract.DeferredType(n)
	case "identifier", "instance_variable":
		typ, _ := env.Lookup(n.Utf8Text(ex.src))
		return typ
	case "self":
		typ, _ := env.Lookup("self")
		return typ
	case "parenthesized_statements":
		if n.NamedChildCount() == 1 {
			return ex.exprType(env, n.NamedChild(0))
		}
	}
	return ""
}

// callArgs reads an argument_list: each argument's type, a keyword pair's
// key as its label, and a splat as spread.
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
		case "splat_argument", "hash_splat_argument":
			ca.Spread = true
			ca.Types = append(ca.Types, "")
			ca.Labels = append(ca.Labels, "")
		case "pair":
			labeled = true
			label := ""
			if k := a.ChildByFieldName("key"); k != nil {
				label = k.Utf8Text(ex.src)
			}
			typ := ""
			if v := a.ChildByFieldName("value"); v != nil {
				typ = ex.exprType(env, v)
			}
			ca.Types = append(ca.Types, typ)
			ca.Labels = append(ca.Labels, label)
		case "block_argument":
			ca.Types = append(ca.Types, "")
			ca.Labels = append(ca.Labels, "")
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

// receiver classifies the receiver of a call: a constant is a candidate
// type, a variable goes through the environment, self is the enclosing
// scope (D17, D18).
func (ex *extractor) receiver(env *goextract.TypeEnv, r *tree_sitter.Node) (pkgAlias, receiverType string) {
	if r == nil {
		return "", ""
	}
	switch r.Kind() {
	case "constant", "scope_resolution":
		return "", rubyConstantName(r, ex.src)
	case "identifier", "instance_variable":
		return goextract.Receiver(env, ex.result.Imports, r.Utf8Text(ex.src))
	case "self":
		typ, _ := env.Lookup("self")
		return "", typ
	}
	if typ := ex.exprType(env, r); typ != "" {
		return "", typ
	}
	return goextract.UnknownAlias(r.Kind()), ""
}
