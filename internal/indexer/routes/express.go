package routes

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// expressVerbs maps an Express route-registration method name to its HTTP
// verb — mirrors ginVerbs' shape: the receiver is deliberately ANY
// identifier (app/router/anyCustomRouterVar), verb + string-literal path +
// handler argument is the precision gate (Pattern 4), not the receiver's
// own name.
var expressVerbs = map[string]string{
	"get":     "GET",
	"post":    "POST",
	"put":     "PUT",
	"patch":   "PATCH",
	"delete":  "DELETE",
	"options": "OPTIONS",
	"head":    "HEAD",
	"all":     "ANY",
}

// nestjsVerbs maps a NestJS HTTP-method decorator's bare name to its HTTP
// verb.
var nestjsVerbs = map[string]string{
	"Get":     "GET",
	"Post":    "POST",
	"Put":     "PUT",
	"Patch":   "PATCH",
	"Delete":  "DELETE",
	"Options": "OPTIONS",
	"Head":    "HEAD",
	"All":     "ANY",
}

// expressLanguages are every TS/JS LanguageSpec.ID a real Express/NestJS
// repo's route-registration or decorator call-shape can appear in — the
// AST shape (call_expression/decorator over the shared TS/JS grammar
// family) is identical across all three, so the SAME Walk function is
// registered once per language rather than special-cased.
var expressLanguages = []string{"typescript", "javascript", "tsx"}

func init() {
	for _, lang := range expressLanguages {
		Register(Detector{
			ID:       "express-route",
			Language: lang,
			Signature: func(manifestText string) bool {
				return strings.Contains(manifestText, `"express"`)
			},
			Walk: walkExpressRoutes,
		})
	}
	// NestJS decorators require TypeScript's experimental-decorators
	// support — a real NestJS repo's route decorators are, in practice,
	// always authored in .ts (not plain .js), so this detector is
	// registered for "typescript" and "tsx" only.
	for _, lang := range []string{"typescript", "tsx"} {
		Register(Detector{
			ID:       "nestjs-route",
			Language: lang,
			Signature: func(manifestText string) bool {
				return strings.Contains(manifestText, "@nestjs")
			},
			Walk: walkNestJSRoutes,
		})
	}
}

// walkExpressRoutes walks call_expression nodes for the shape
// `<anyIdentifier>.<verb>("<path>", <handler>)` — the TS/JS grammar
// family's member_expression + call_expression shape verified via a live
// parse this session (identical structure to Go's selector_expression
// case in gin.go, different node-kind names).
func walkExpressRoutes(root *tree_sitter.Node, src []byte, resolve HandlerResolver) []Route {
	var out []Route
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		if n.Kind() != "call_expression" {
			return true
		}
		fn := n.ChildByFieldName("function")
		if fn == nil || fn.Kind() != "member_expression" {
			return true
		}
		property := fn.ChildByFieldName("property")
		if property == nil {
			return true
		}
		verb, ok := expressVerbs[property.Utf8Text(src)]
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

// walkNestJSRoutes walks every class_body's direct named children,
// tracking the most recently seen "decorator" sibling and pairing it with
// the method_definition that immediately follows it (verified via a live
// parse this session: unlike Java/C#'s in-span annotations/attributes,
// TS's per-method decorator is a SEPARATE preceding sibling within
// class_body, not part of the method_definition node's own span) for a
// NestJS HTTP-method decorator (`@Get(...)`, `@Post`, ...). The handler is
// the decorated method_definition itself, resolved via ResolveByLine
// against ITS OWN StartPosition — tsextract.go's emitMethod records
// exactly that position (the method_definition's own span, excluding the
// preceding decorator) as the method node's StartLine.
func walkNestJSRoutes(root *tree_sitter.Node, src []byte, resolve HandlerResolver) []Route {
	var out []Route
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		if n.Kind() != "class_body" {
			return true
		}
		var pending *tree_sitter.Node
		for i := uint(0); i < n.NamedChildCount(); i++ {
			c := n.NamedChild(i)
			if c.Kind() == "decorator" {
				pending = c
				continue
			}
			if c.Kind() != "method_definition" {
				pending = nil
				continue
			}
			dec := pending
			pending = nil
			if dec == nil {
				continue
			}
			if rt, ok := nestjsRouteFromDecorator(dec, c, src, resolve); ok {
				out = append(out, rt)
			}
		}
		return true
	})
	return out
}

// nestjsRouteFromDecorator reports whether dec is a NestJS HTTP-method
// decorator call (`@Get(...)`/`@Post`/...) applied to method, resolving
// method's own node id via ResolveByLine.
func nestjsRouteFromDecorator(dec, method *tree_sitter.Node, src []byte, resolve HandlerResolver) (Route, bool) {
	call := dec.NamedChild(0)
	if call == nil || call.Kind() != "call_expression" {
		return Route{}, false
	}
	fn := call.ChildByFieldName("function")
	if fn == nil || fn.Kind() != "identifier" {
		return Route{}, false
	}
	verb, ok := nestjsVerbs[fn.Utf8Text(src)]
	if !ok {
		return Route{}, false
	}
	args := call.ChildByFieldName("arguments")
	path := ""
	if args != nil && args.NamedChildCount() > 0 {
		path, _ = stringValue(args.NamedChild(0), src)
	}

	methodPos := method.StartPosition()
	handlerID, ok := resolve.ResolveByLine(int32(methodPos.Row) + 1)
	if !ok {
		return Route{}, false
	}
	decPos := dec.StartPosition()
	return Route{
		HTTPMethod: verb,
		Path:       path,
		HandlerID:  handlerID,
		Line:       int32(decPos.Row) + 1,
		Col:        int32(decPos.Column),
	}, true
}
