package goextract

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Param is one parameter of a function or method as the source writes it
// (design D13 of the index-invariants change). Type is the type text as
// written, or "" when the language or the author wrote none. Every
// extractor fills the same shape, so the signature key, the arity filter
// and the type filter of the resolver work the same way in every language.
type Param struct {
	Name       string
	Type       string
	Optional   bool
	HasDefault bool
	Variadic   bool
}

// SchemaParams converts params to the stored shape.
func SchemaParams(params []Param) []*schema.Param {
	if len(params) == 0 {
		return nil
	}
	out := make([]*schema.Param, len(params))
	for i, p := range params {
		out[i] = &schema.Param{Name: p.Name, Type: p.Type, Optional: p.Optional, HasDefault: p.HasDefault, Variadic: p.Variadic}
	}
	return out
}

// SignatureKeyOf is the signature key of a declaration with params
// (nodeid.SignatureKey). A parameter with a default is optional for the
// key, because a call can leave it out.
func SignatureKeyOf(params []Param) string {
	keys := make([]nodeid.ParamKey, len(params))
	for i, p := range params {
		keys[i] = nodeid.ParamKey{Type: p.Type, Optional: p.Optional || p.HasDefault, Variadic: p.Variadic}
	}
	return nodeid.SignatureKey(keys)
}

// TypeName reduces a type as written to the name the resolver matches: no
// leading annotation colon, pointer, reference or nullable marker, and no
// trailing nullable, pointer or reference marker. Generic arguments and
// array brackets stay, because they tell types apart.
func TypeName(text string) string {
	t := strings.TrimSpace(text)
	t = strings.TrimSpace(strings.TrimPrefix(t, ":"))
	for {
		switch {
		case strings.HasPrefix(t, "&mut "):
			t = t[len("&mut "):]
		case strings.HasPrefix(t, "*"), strings.HasPrefix(t, "&"), strings.HasPrefix(t, "?"):
			t = t[1:]
		default:
			t = strings.TrimRight(t, "?!*& ")
			return strings.Join(strings.Fields(t), " ")
		}
		t = strings.TrimSpace(t)
	}
}

// ForEachChild calls fn for every child of n, named or not, with the
// field name the grammar gives it, or "" for none.
func ForEachChild(n *tree_sitter.Node, fn func(c *tree_sitter.Node, field string)) {
	for i := uint(0); i < n.ChildCount(); i++ {
		fn(n.Child(i), n.FieldNameForChild(uint32(i)))
	}
}

// FirstNamedChildOfKind returns n's first named child whose kind is in
// kinds, or nil.
func FirstNamedChildOfKind(n *tree_sitter.Node, kinds ...string) *tree_sitter.Node {
	for i := uint(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		for _, k := range kinds {
			if c.Kind() == k {
				return c
			}
		}
	}
	return nil
}

// FillSignature sets the structured signature of a function or method
// node: its parameters, arity and return type name.
func FillSignature(n *schema.Node, params []Param, returnType string) {
	n.Params = SchemaParams(params)
	n.Arity = int32(len(params))
	n.ReturnTypeName = TypeName(returnType)
}

// ParamsOf reads a stored node's parameters back into Params.
func ParamsOf(n *schema.Node) []Param {
	if len(n.Params) == 0 {
		return nil
	}
	out := make([]Param, len(n.Params))
	for i, p := range n.Params {
		out[i] = Param{Name: p.Name, Type: p.Type, Optional: p.Optional, HasDefault: p.HasDefault, Variadic: p.Variadic}
	}
	return out
}

// SignatureKeyOfNode is the signature key of a stored function or method
// node, and "" for a node of another kind.
func SignatureKeyOfNode(n *schema.Node) string {
	if n.Kind != KindFunction && n.Kind != KindMethod {
		return ""
	}
	return SignatureKeyOf(ParamsOf(n))
}
