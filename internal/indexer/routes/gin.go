package routes

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// ginVerbs maps a Gin route-registration method name to its HTTP verb —
// the receiver is deliberately ANY identifier (group vars like v1/
// userRouter, not a fixed "router"/"r"/"app" name list), matching
// RESEARCH Pattern 4's precision gate (verb + string-literal path +
// handler argument), not the receiver's own name.
var ginVerbs = map[string]string{
	"GET":     "GET",
	"POST":    "POST",
	"PUT":     "PUT",
	"PATCH":   "PATCH",
	"DELETE":  "DELETE",
	"OPTIONS": "OPTIONS",
	"HEAD":    "HEAD",
	"Any":     "ANY",
}

func init() {
	Register(Detector{
		ID:       "gin-route",
		Language: "go",
		Signature: func(manifestText string) bool {
			return strings.Contains(manifestText, "gin-gonic/gin")
		},
		Walk: walkGinRoutes,
	})
}

// walkGinRoutes walks call_expression nodes already parsed for this file
// (Pattern 4: AST-node matching, never a second regex pass over raw
// source — T-05-ReDoS is structurally eliminated, not just avoided) for
// the shape `<anyIdentifier>.<VERB>("<path>", <handler>)`. The handler
// argument must be a bare identifier (a function reference) — an inline
// closure or a non-identifier expression is not detected (Pattern 4's
// precision gate; a dynamic/computed handler cannot be traced to a
// symbol node anyway).
func walkGinRoutes(root *tree_sitter.Node, src []byte, resolve HandlerResolver) []Route {
	var out []Route
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		if n.Kind() != "call_expression" {
			return true
		}
		fn := n.ChildByFieldName("function")
		if fn == nil || fn.Kind() != "selector_expression" {
			return true
		}
		field := fn.ChildByFieldName("field")
		if field == nil {
			return true
		}
		verb, ok := ginVerbs[field.Utf8Text(src)]
		if !ok {
			return true
		}
		args := n.ChildByFieldName("arguments")
		if args == nil || args.NamedChildCount() < 2 {
			return true
		}
		path, ok := stringValue(args.NamedChild(0), src)
		if !ok {
			return true
		}
		handlerNode := args.NamedChild(1)
		if handlerNode.Kind() != "identifier" {
			return true
		}
		handlerID, ok := resolve.ResolveByName(handlerNode.Utf8Text(src))
		if !ok {
			return true
		}
		pos := n.StartPosition()
		out = append(out, Route{
			HTTPMethod: verb,
			Path:       path,
			HandlerID:  handlerID,
			Line:       int32(pos.Row) + 1,
			Col:        int32(pos.Column),
		})
		return true
	})
	return out
}
