package routes

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// aspnetVerbs maps an ASP.NET Http-verb attribute's bare name to its HTTP
// method. A bare [Route("...")] attribute with no accompanying Http<Verb>
// attribute is NOT detected as a route on its own (attribute-routing
// convention leaves the verb implicit/all-verbs in that case — this
// project's Route representation requires exactly one HTTPMethod, so only
// the unambiguous Http<Verb> attributes are matched, mirroring Spring's
// same simplification for a method-less @RequestMapping).
var aspnetVerbs = map[string]string{
	"HttpGet":     "GET",
	"HttpPost":    "POST",
	"HttpPut":     "PUT",
	"HttpDelete":  "DELETE",
	"HttpPatch":   "PATCH",
	"HttpHead":    "HEAD",
	"HttpOptions": "OPTIONS",
}

func init() {
	Register(Detector{
		ID:       "aspnet-route",
		Language: "csharp",
		Signature: func(manifestText string) bool {
			return strings.Contains(manifestText, "Microsoft.AspNetCore")
		},
		Walk: walkAspNetRoutes,
	})
}

// walkAspNetRoutes walks every method_declaration in the file, looking at
// its own direct "attribute_list" children (verified via a live parse
// this session: unlike Java, C# exposes each [Attr]/[Attr,Attr2] bracket
// group as its OWN "attribute_list" node — a direct, unnamed-field child
// of method_declaration, one per bracket group, not nested under a single
// "modifiers" wrapper) for an Http<Verb> attribute. The handler is the
// attributed method declaration itself, resolved via ResolveByLine against
// the method_declaration's own StartPosition — csharpextract.go's
// emitMethod records that SAME position as the method node's StartLine
// (a C# method_declaration's span already starts at its first
// attribute_list, exactly like Java's method_declaration/modifiers).
func walkAspNetRoutes(root *tree_sitter.Node, src []byte, resolve HandlerResolver) []Route {
	var out []Route
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		if n.Kind() != "method_declaration" {
			return true
		}
		startPos := n.StartPosition()
		handlerLine := int32(startPos.Row) + 1

		for i := uint(0); i < n.NamedChildCount(); i++ {
			attrList := n.NamedChild(i)
			if attrList.Kind() != "attribute_list" {
				continue
			}
			for j := uint(0); j < attrList.NamedChildCount(); j++ {
				attr := attrList.NamedChild(j)
				if attr.Kind() != "attribute" {
					continue
				}
				nameNode := attr.ChildByFieldName("name")
				if nameNode == nil {
					continue
				}
				verb, ok := aspnetVerbs[nameNode.Utf8Text(src)]
				if !ok {
					continue
				}
				handlerID, ok := resolve.ResolveByLine(handlerLine)
				if !ok {
					continue
				}
				attrPos := attr.StartPosition()
				out = append(out, Route{
					HTTPMethod: verb,
					Path:       aspnetAttributePath(attr, src),
					HandlerID:  handlerID,
					Line:       int32(attrPos.Row) + 1,
					Col:        int32(attrPos.Column),
				})
			}
		}
		return true
	})
	return out
}

// aspnetAttributePath extracts an Http<Verb> attribute's single positional
// string-literal argument (`[HttpGet("{id}")]`), or "" for a bare,
// argument-less attribute (`[HttpPost]` — the route's path is then
// whatever the controller-level [Route] prefix declares, which this
// bounded detector does not compose — Claude's Discretion, D-08).
func aspnetAttributePath(attr *tree_sitter.Node, src []byte) string {
	for i := uint(0); i < attr.NamedChildCount(); i++ {
		c := attr.NamedChild(i)
		if c.Kind() != "attribute_argument_list" {
			continue
		}
		for j := uint(0); j < c.NamedChildCount(); j++ {
			arg := c.NamedChild(j)
			if arg.Kind() != "attribute_argument" {
				continue
			}
			for k := uint(0); k < arg.NamedChildCount(); k++ {
				if path, ok := stringValue(arg.NamedChild(k), src); ok {
					return path
				}
			}
		}
	}
	return ""
}
