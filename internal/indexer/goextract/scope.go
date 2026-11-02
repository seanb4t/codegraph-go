package goextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// FillScopes sets the Scope of each node in nodes and marks its Node as
// nested (design D15 of the index-invariants change). The scope of a
// declaration is the nearest local scope around it in the syntax tree:
// a function body, a block, a lambda or an anonymous class body. A
// declaration with no such ancestor is module-level or a type member,
// and keeps the zero Scope. The scope kinds are the body kinds of every
// grammar the indexer parses; a kind that means a class body in one
// grammar and a block in another is told apart by its parent.
func FillScopes(root *tree_sitter.Node, nodes []ExtractedNode) {
	if root == nil {
		return
	}
	for i := range nodes {
		n := nodes[i].Node
		if n == nil || n.StartLine <= 0 {
			continue
		}
		at := tree_sitter.Point{Row: uint(n.StartLine - 1), Column: uint(n.StartCol)}
		deep := root.DescendantForPointRange(at, at)
		if deep == nil {
			continue
		}
		start := deep.StartByte()
		for p := deep.Parent(); p != nil; p = p.Parent() {
			if p.StartByte() == start {
				// The declaration itself, or a wrapper that starts with
				// it, such as an export statement.
				continue
			}
			if !isLocalScope(p, n.Language) {
				continue
			}
			s, e := p.StartPosition(), p.EndPosition()
			nodes[i].Scope = Range{
				StartLine: int32(s.Row) + 1, StartCol: int32(s.Column),
				EndLine: int32(e.Row) + 1, EndCol: int32(e.Column),
			}
			n.Nested = true
			break
		}
	}
}

// isLocalScope reports whether p is a local scope: a declaration inside
// it is visible only there.
func isLocalScope(p *tree_sitter.Node, lang string) bool {
	parent := ""
	if pp := p.Parent(); pp != nil {
		parent = pp.Kind()
	}
	switch p.Kind() {
	case "statement_block", "function_body", "constructor_body",
		"lambda_literal", "lambda_body", "do_block", "anonymous_function", "closure_expression":
		return true
	case "compound_statement":
		// PHP has no block scope: a declaration under a top-level `if`
		// is global. Only a function body is a local scope there.
		if lang == "php" {
			switch parent {
			case "function_definition", "method_declaration", "anonymous_function_creation_expression", "arrow_function", "function_static_declaration":
				return true
			}
			return false
		}
		return true
	case "block":
		// Python's class body is also a block.
		return lang != "python" || parent != "class_definition"
	case "body_statement":
		// Ruby: a method or block body, not a class or module body.
		switch parent {
		case "method", "singleton_method", "do_block", "block", "lambda":
			return true
		}
		return false
	case "class_body", "declaration_list", "enum_body":
		// An anonymous class or object literal body is a local scope.
		switch parent {
		case "object_creation_expression", "anonymous_object_creation_expression", "object_literal":
			return true
		}
		return false
	}
	return false
}

// ScopeEnd returns the byte where the scope of a local declared by n
// ends: the end of the nearest local scope around n (isLocalScope), or
// of the statement that owns n as an initializer or loop variable, such
// as a Go `if x := f(); ...` or a `for` clause. With no such ancestor it
// returns WholeBody.
func ScopeEnd(n *tree_sitter.Node, lang string) uint32 {
	for p := n.Parent(); p != nil; p = p.Parent() {
		switch p.Kind() {
		case "if_statement", "for_statement", "for_in_statement", "for_range_loop", "enhanced_for_statement",
			"foreach_statement", "switch_statement", "type_switch_statement", "expression_switch_statement",
			"try_statement", "try_with_resources_statement", "using_statement", "catch_clause", "func_literal",
			"lambda_expression", "arrow_function", "function_expression":
			return uint32(p.EndByte())
		}
		if isLocalScope(p, lang) {
			return uint32(p.EndByte())
		}
	}
	return WholeBody
}
