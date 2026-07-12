package routes

import (
	"strconv"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// walkDescendants performs an iterative (stack-based, non-recursive)
// pre-order walk of n's descendants — mirrors goextract.walkDescendants'
// exact shape (05-PATTERNS.md §Per-file skip/error contract's sibling
// discipline: an explicit stack, never Go-stack recursion over a
// pathologically deep AST, T-02-04). visit returning false is not used by
// any detector today (kept for parity with the pattern this mirrors, not
// dead-code-flagged since it documents the walk's own extensibility).
func walkDescendants(root *tree_sitter.Node, visit func(*tree_sitter.Node) bool) {
	stack := make([]*tree_sitter.Node, 0, 64)
	stack = append(stack, root)
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if !visit(n) {
			continue
		}
		count := n.ChildCount()
		for i := int(count) - 1; i >= 0; i-- {
			if c := n.Child(uint(i)); c != nil {
				stack = append(stack, c)
			}
		}
	}
}

// findChildKind returns n's first NAMED child whose Kind() equals kind, or
// nil if none exists — a small scan helper every per-framework detector
// uses to find a "modifiers"/"attribute_list" wrapper node among a
// declaration's direct children (Java/C# do not expose these as a single
// ChildByFieldName field the way TS's "decorator" field does).
func findChildKind(n *tree_sitter.Node, kind string) *tree_sitter.Node {
	for i := uint(0); i < n.NamedChildCount(); i++ {
		if c := n.NamedChild(i); c.Kind() == kind {
			return c
		}
	}
	return nil
}

// stringValue extracts a string/interpreted-string/raw-string literal
// node's UNQUOTED text content, uniformly across every priority-4
// grammar's own string-literal shape (verified via a live parse this
// session, not guessed):
//   - Go: "interpreted_string_literal" (needs strconv.Unquote — handles
//     escape sequences) or "raw_string_literal" (backtick-delimited, no
//     escaping — strip the outer backticks).
//   - Java/C#/Python/TS: a wrapper node ("string_literal" or "string")
//     whose OWN named children include the actual text-bearing node
//     ("string_fragment" for Java/TS, "string_content" for Python,
//     "string_literal_content" for C#) — concatenated verbatim (an
//     interior "escape_sequence" child, if any, is skipped: this is a
//     route-path extractor, not a full string-literal evaluator, and a
//     route path containing an escape sequence is vanishingly rare).
//
// Returns ok=false for a node that is nil or not a recognizable string
// literal shape — callers MUST treat that as "this argument is not a
// string-literal path" and skip the match (Pattern 4: verb + string-
// literal path + handler argument is the precision gate; a non-literal
// path argument, e.g. a variable, is deliberately NOT detected — no
// false-positive route from an untraceable dynamic path).
func stringValue(n *tree_sitter.Node, src []byte) (string, bool) {
	if n == nil {
		return "", false
	}
	switch n.Kind() {
	case "interpreted_string_literal":
		txt := n.Utf8Text(src)
		if unquoted, err := strconv.Unquote(txt); err == nil {
			return unquoted, true
		}
		return trimQuotes(txt), true
	case "raw_string_literal":
		return trimQuotes(n.Utf8Text(src)), true
	case "string_literal", "string":
		var parts []string
		for i := uint(0); i < n.NamedChildCount(); i++ {
			switch c := n.NamedChild(i); c.Kind() {
			case "string_fragment", "string_content", "string_literal_content":
				parts = append(parts, c.Utf8Text(src))
			}
		}
		if len(parts) == 0 {
			return "", false
		}
		return strings.Join(parts, ""), true
	default:
		return "", false
	}
}

// trimQuotes strips a Go raw_string_literal's backticks, or (defensively)
// a leading/trailing quote character from any other quoted-looking text —
// used only for Go's two string-literal node kinds, both of which are
// always fully quoted (unlike the wrapper-node languages handled by
// stringValue's other branch, which never see a quote character at all —
// string_fragment/string_content/string_literal_content already exclude
// the delimiters).
func trimQuotes(s string) string {
	if len(s) < 2 {
		return s
	}
	first, last := s[0], s[len(s)-1]
	if (first == '"' && last == '"') || (first == '\'' && last == '\'') || (first == '`' && last == '`') {
		return s[1 : len(s)-1]
	}
	return s
}
