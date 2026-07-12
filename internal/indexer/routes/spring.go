package routes

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// springDirectVerbs maps a Spring mapping annotation's bare name directly
// to its HTTP verb — everything except the generic @RequestMapping, whose
// verb (if any) instead comes from its own "method" element (see
// springRequestMappingVerb).
var springDirectVerbs = map[string]string{
	"GetMapping":    "GET",
	"PostMapping":   "POST",
	"PutMapping":    "PUT",
	"DeleteMapping": "DELETE",
	"PatchMapping":  "PATCH",
}

func init() {
	Register(Detector{
		ID:       "spring-route",
		Language: "java",
		Signature: func(manifestText string) bool {
			return strings.Contains(manifestText, "org.springframework")
		},
		Walk: walkSpringRoutes,
	})
}

// walkSpringRoutes walks every method_declaration in the file, looking at
// its "modifiers" child (verified via a live parse this session: a Java
// method's annotations are NAMED children of its OWN "modifiers" node —
// annotation/marker_annotation siblings alongside the access-modifier
// keyword tokens) for a Spring HTTP-mapping annotation. The handler is the
// annotated method declaration ITSELF (Spring/annotation-based frameworks
// have no separate handler argument), resolved via ResolveByLine against
// the method_declaration's own StartPosition — the same position
// javaextract.go's emitMethod records as that method node's StartLine,
// since a Java method_declaration's span already starts at its first
// modifier/annotation (Pattern 2/RESEARCH's own verified grammar shape).
func walkSpringRoutes(root *tree_sitter.Node, src []byte, resolve HandlerResolver) []Route {
	var out []Route
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		if n.Kind() != "method_declaration" {
			return true
		}
		modifiers := findChildKind(n, "modifiers")
		if modifiers == nil {
			return true
		}
		startPos := n.StartPosition()
		handlerLine := int32(startPos.Row) + 1

		for i := uint(0); i < modifiers.NamedChildCount(); i++ {
			ann := modifiers.NamedChild(i)
			if ann.Kind() != "annotation" && ann.Kind() != "marker_annotation" {
				continue
			}
			nameNode := ann.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}
			verb, path, ok := springRouteFromAnnotation(nameNode.Utf8Text(src), ann, src)
			if !ok {
				continue
			}
			handlerID, ok := resolve.ResolveByLine(handlerLine)
			if !ok {
				continue
			}
			annPos := ann.StartPosition()
			out = append(out, Route{
				HTTPMethod: verb,
				Path:       path,
				HandlerID:  handlerID,
				Line:       int32(annPos.Row) + 1,
				Col:        int32(annPos.Column),
			})
		}
		return true
	})
	return out
}

// springRouteFromAnnotation reports whether ann is a Spring HTTP-mapping
// annotation and, if so, its HTTP verb and path. A direct verb annotation
// (@GetMapping, ...) always matches; the generic @RequestMapping matches
// too, deriving its verb from an element_value_pair keyed "method" (a
// RequestMethod.<VERB> field_access) or defaulting to "GET" when absent —
// a bounded, documented simplification (Claude's Discretion, D-08): Spring
// itself treats a method-less @RequestMapping as matching every verb, but
// this project's route representation carries exactly one HTTPMethod per
// Route.
func springRouteFromAnnotation(annName string, ann *tree_sitter.Node, src []byte) (verb, path string, ok bool) {
	args := ann.ChildByFieldName("arguments")

	if v, isDirect := springDirectVerbs[annName]; isDirect {
		return v, springAnnotationPath(args, src), true
	}
	if annName != "RequestMapping" {
		return "", "", false
	}
	return springRequestMappingVerb(args, src), springAnnotationPath(args, src), true
}

// springAnnotationPath extracts a mapping annotation's declared path: a
// single positional string_literal argument (`@GetMapping("/x")`), or an
// element_value_pair keyed "value" or "path"
// (`@RequestMapping(value = "/x", ...)`).
func springAnnotationPath(args *tree_sitter.Node, src []byte) string {
	if args == nil {
		return ""
	}
	for i := uint(0); i < args.NamedChildCount(); i++ {
		c := args.NamedChild(i)
		switch c.Kind() {
		case "string_literal":
			if path, ok := stringValue(c, src); ok {
				return path
			}
		case "element_value_pair":
			key := c.ChildByFieldName("key")
			if key == nil {
				continue
			}
			keyName := key.Utf8Text(src)
			if keyName != "value" && keyName != "path" {
				continue
			}
			if path, ok := stringValue(c.ChildByFieldName("value"), src); ok {
				return path
			}
		}
	}
	return ""
}

// springRequestMappingVerb extracts @RequestMapping's "method" element
// (`method = RequestMethod.POST`), returning its trailing field-access
// identifier as the HTTP verb, or "GET" when no "method" element is
// present (Spring's own multi-verb default collapsed to a single verb —
// see springRouteFromAnnotation's doc comment).
func springRequestMappingVerb(args *tree_sitter.Node, src []byte) string {
	if args == nil {
		return "GET"
	}
	for i := uint(0); i < args.NamedChildCount(); i++ {
		c := args.NamedChild(i)
		if c.Kind() != "element_value_pair" {
			continue
		}
		key := c.ChildByFieldName("key")
		if key == nil || key.Utf8Text(src) != "method" {
			continue
		}
		value := c.ChildByFieldName("value")
		if value == nil || value.Kind() != "field_access" {
			continue
		}
		field := value.ChildByFieldName("field")
		if field == nil {
			continue
		}
		return field.Utf8Text(src)
	}
	return "GET"
}
