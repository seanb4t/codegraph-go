package routes

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// flaskFastapiDirectVerbs maps a Flask/FastAPI direct-verb decorator
// method name (`@app.get(...)`) to its HTTP verb — the two frameworks use
// an IDENTICAL decorator-call shape for this pattern (both wrap a Flask-
// or FastAPI-typed app/router object), so one Walk function covers both;
// only the opt-in Signature differs per registered Detector below.
var flaskFastapiDirectVerbs = map[string]string{
	"get":     "GET",
	"post":    "POST",
	"put":     "PUT",
	"patch":   "PATCH",
	"delete":  "DELETE",
	"head":    "HEAD",
	"options": "OPTIONS",
}

func init() {
	Register(Detector{
		ID:       "django-route",
		Language: "python",
		Signature: func(manifestText string) bool {
			return strings.Contains(manifestText, "django") || strings.Contains(manifestText, "Django")
		},
		Walk: walkDjangoRoutes,
	})
	Register(Detector{
		ID:       "flask-route",
		Language: "python",
		Signature: func(manifestText string) bool {
			return strings.Contains(manifestText, "flask") || strings.Contains(manifestText, "Flask")
		},
		Walk: walkFlaskFastapiRoutes,
	})
	Register(Detector{
		ID:       "fastapi-route",
		Language: "python",
		Signature: func(manifestText string) bool {
			return strings.Contains(manifestText, "fastapi") || strings.Contains(manifestText, "FastAPI")
		},
		Walk: walkFlaskFastapiRoutes,
	})
}

// walkDjangoRoutes finds every module-level `urlpatterns = [...]`
// assignment and walks its list literal for `path(...)`/`re_path(...)`
// call entries (verified via a live parse this session: both are plain
// call nodes, function=identifier "path"/"re_path", NOT decorators — a
// Django URLconf routes by TABLE ENTRY, not by decorating the view). The
// route's HTTP verb is "ANY": a Django path() entry routes to a view by
// URL alone, with no verb of its own — the view itself dispatches by
// request.method internally, so no single HTTPMethod is derivable from
// the URLconf entry (Claude's Discretion, D-08).
func walkDjangoRoutes(root *tree_sitter.Node, src []byte, resolve HandlerResolver) []Route {
	var out []Route
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		if n.Kind() != "assignment" {
			return true
		}
		left := n.ChildByFieldName("left")
		if left == nil || left.Kind() != "identifier" || left.Utf8Text(src) != "urlpatterns" {
			return true
		}
		right := n.ChildByFieldName("right")
		if right == nil || right.Kind() != "list" {
			return true
		}
		for i := uint(0); i < right.NamedChildCount(); i++ {
			call := right.NamedChild(i)
			if call.Kind() != "call" {
				continue
			}
			fn := call.ChildByFieldName("function")
			if fn == nil || fn.Kind() != "identifier" {
				continue
			}
			if name := fn.Utf8Text(src); name != "path" && name != "re_path" {
				continue
			}
			args := call.ChildByFieldName("arguments")
			if args == nil || args.NamedChildCount() < 2 {
				continue
			}
			path, ok := stringValue(args.NamedChild(0), src)
			if !ok {
				continue
			}
			handlerName, ok := pythonReferenceName(args.NamedChild(1), src)
			if !ok {
				continue
			}
			handlerID, ok := resolve.ResolveByName(handlerName)
			if !ok {
				continue
			}
			pos := call.StartPosition()
			out = append(out, Route{
				HTTPMethod: "ANY",
				Path:       path,
				HandlerID:  handlerID,
				Line:       int32(pos.Row) + 1,
				Col:        int32(pos.Column),
			})
		}
		return true
	})
	return out
}

// walkFlaskFastapiRoutes walks every decorated_definition in the file
// (verified via a live parse this session: Flask/FastAPI's route
// decorator wraps a function_definition exactly like any other Python
// decorator) for a `@<anyIdentifier>.<verb-or-route>(...)` decorator call.
// `.route("/x", methods=[...])` derives its verb from the "methods"
// keyword argument's first entry (defaulting to "GET", Flask's own
// default when methods is omitted); a direct verb decorator
// (`.get/.post/...`) uses that verb literally. The handler is the
// decorated function itself, resolved via ResolveByLine against the
// WRAPPED function_definition's own StartPosition — pyextract.go's
// unwrapDecorated/emitFunction records THAT position (the `def` line, not
// the `@decorator` line) as the function node's StartLine, so this must
// match, not the decorated_definition's or decorator's own start.
func walkFlaskFastapiRoutes(root *tree_sitter.Node, src []byte, resolve HandlerResolver) []Route {
	var out []Route
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		if n.Kind() != "decorated_definition" {
			return true
		}
		def := n.ChildByFieldName("definition")
		if def == nil || def.Kind() != "function_definition" {
			return true
		}
		defPos := def.StartPosition()
		handlerLine := int32(defPos.Row) + 1

		for i := uint(0); i < n.NamedChildCount(); i++ {
			dec := n.NamedChild(i)
			if dec.Kind() != "decorator" {
				continue
			}
			call := dec.NamedChild(0)
			if call == nil || call.Kind() != "call" {
				continue
			}
			fn := call.ChildByFieldName("function")
			if fn == nil || fn.Kind() != "attribute" {
				continue
			}
			verbName := fn.ChildByFieldName("attribute")
			if verbName == nil {
				continue
			}
			args := call.ChildByFieldName("arguments")

			var verb, path string
			var ok bool
			switch verbName.Utf8Text(src) {
			case "route":
				verb = flaskRouteMethodsVerb(args, src)
				path, ok = flaskFirstStringArg(args, src)
			default:
				var isDirect bool
				verb, isDirect = flaskFastapiDirectVerbs[verbName.Utf8Text(src)]
				if !isDirect {
					continue
				}
				path, ok = flaskFirstStringArg(args, src)
			}
			if !ok {
				continue
			}
			handlerID, resolved := resolve.ResolveByLine(handlerLine)
			if !resolved {
				continue
			}
			decPos := dec.StartPosition()
			out = append(out, Route{
				HTTPMethod: verb,
				Path:       path,
				HandlerID:  handlerID,
				Line:       int32(decPos.Row) + 1,
				Col:        int32(decPos.Column),
			})
		}
		return true
	})
	return out
}

// flaskFirstStringArg returns the decorator call's first POSITIONAL
// (non-keyword) string-literal argument — the route path.
func flaskFirstStringArg(args *tree_sitter.Node, src []byte) (string, bool) {
	if args == nil {
		return "", false
	}
	for i := uint(0); i < args.NamedChildCount(); i++ {
		c := args.NamedChild(i)
		if c.Kind() == "keyword_argument" {
			continue
		}
		return stringValue(c, src)
	}
	return "", false
}

// flaskRouteMethodsVerb extracts `.route(...)`'s "methods" keyword
// argument (a list of string literals) and returns its FIRST entry,
// uppercased, or "GET" (Flask's own default) when "methods" is absent —
// a bounded, documented simplification (D-08, mirroring Spring's
// method-less @RequestMapping default): a Route only ever carries one
// HTTPMethod, so a multi-verb `methods=["GET","POST"]` collapses to its
// first declared verb.
func flaskRouteMethodsVerb(args *tree_sitter.Node, src []byte) string {
	if args == nil {
		return "GET"
	}
	for i := uint(0); i < args.NamedChildCount(); i++ {
		c := args.NamedChild(i)
		if c.Kind() != "keyword_argument" {
			continue
		}
		name := c.ChildByFieldName("name")
		if name == nil || name.Utf8Text(src) != "methods" {
			continue
		}
		value := c.ChildByFieldName("value")
		if value == nil || value.Kind() != "list" || value.NamedChildCount() == 0 {
			continue
		}
		if v, ok := stringValue(value.NamedChild(0), src); ok {
			return strings.ToUpper(v)
		}
	}
	return "GET"
}

// pythonReferenceName extracts the bare, rightmost identifier of a
// handler-reference node — either a plain identifier (`my_view`) or an
// attribute expression (`views.get_user` -> "get_user", the common
// Django `path("x", views.some_view)` shape) — the name HandlerResolver's
// same-file/global-by-name lookup is keyed by.
func pythonReferenceName(n *tree_sitter.Node, src []byte) (string, bool) {
	switch n.Kind() {
	case "identifier":
		return n.Utf8Text(src), true
	case "attribute":
		attr := n.ChildByFieldName("attribute")
		if attr == nil {
			return "", false
		}
		return attr.Utf8Text(src), true
	default:
		return "", false
	}
}
