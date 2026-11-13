package tsextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// collectModuleBindings emits the module-level var, let and const
// bindings, exported or not. A binding whose value is an arrow function
// or a function expression is a function. Any other binding is a
// variable, or a constant for const. A destructuring pattern declares
// each identifier it binds.
func (ex *extractor) collectModuleBindings(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		decl := root.NamedChild(i)
		if decl.Kind() == "export_statement" {
			decl = decl.ChildByFieldName("declaration")
		}
		if decl == nil {
			continue
		}
		var kind string
		switch decl.Kind() {
		case "variable_declaration":
			kind = goextract.KindVariable
		case "lexical_declaration":
			kind = goextract.KindVariable
			if k := decl.ChildByFieldName("kind"); k != nil && k.Utf8Text(ex.src) == "const" {
				kind = goextract.KindConstant
			}
		default:
			continue
		}
		for j := uint(0); j < decl.NamedChildCount(); j++ {
			if d := decl.NamedChild(j); d.Kind() == "variable_declarator" {
				ex.emitBinding(kind, decl, d)
			}
		}
	}
}

// emitBinding emits the declarations of one variable_declarator d of the
// declaration decl.
func (ex *extractor) emitBinding(kind string, decl, d *tree_sitter.Node) {
	nameNode := d.ChildByFieldName("name")
	if nameNode == nil {
		return
	}
	value := d.ChildByFieldName("value")
	if nameNode.Kind() != "identifier" {
		for _, n := range patternIdentifiers(nameNode) {
			id := ex.emitDataNode(kind, decl, d, n)
			if value != nil {
				ex.captureInitializer(id, value, false)
			}
		}
		return
	}
	if value != nil {
		switch value.Kind() {
		case "arrow_function", "function_expression", "generator_function":
			ex.emitFunctionBinding(decl, d, nameNode.Utf8Text(ex.src), value)
			return
		}
	}
	id := ex.emitDataNode(kind, decl, d, nameNode)
	if value != nil {
		ex.captureInitializer(id, value, true)
	}
}

// emitDataNode emits the variable or constant that nameNode declares and
// returns its id. The node spans the declarator d.
func (ex *extractor) emitDataNode(kind string, decl, d, nameNode *tree_sitter.Node) string {
	name := nameNode.Utf8Text(ex.src)
	at := nameNode.StartByte()
	if d.ChildByFieldName("name") == nameNode {
		at = d.StartByte()
	}
	id := ex.alloc.Declare(kind, name, "", at)
	exported := ex.exported(decl, name)
	start, end := d.StartPosition(), d.EndPosition()
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          kind,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      ex.language,
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Visibility:    visibilityFor(exported),
		IsExported:    exported,
	}})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: "contains", Provenance: "ast",
	}})
	if t := d.ChildByFieldName("type"); t != nil {
		ex.emitNamedTypeRef(id, t, goextract.RefKindTypeOf)
	}
	return id
}

// patternIdentifiers returns the identifiers that the destructuring
// pattern binds, in source order. A computed key binds no name of its own.
func patternIdentifiers(pattern *tree_sitter.Node) []*tree_sitter.Node {
	var out []*tree_sitter.Node
	var visit func(n *tree_sitter.Node)
	visit = func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "identifier", "shorthand_property_identifier_pattern":
			out = append(out, n)
		case "pair_pattern":
			if v := n.ChildByFieldName("value"); v != nil {
				visit(v)
			}
		case "assignment_pattern", "object_assignment_pattern":
			if l := n.ChildByFieldName("left"); l != nil {
				visit(l)
			}
		case "object_pattern", "array_pattern", "rest_pattern":
			for i := uint(0); i < n.NamedChildCount(); i++ {
				visit(n.NamedChild(i))
			}
		}
	}
	visit(pattern)
	return out
}

// emitField emits the data field fd of the class typeName when its name
// is an identifier or a private name. A computed or quoted name declares
// no field here, because the extractor does not evaluate it.
func (ex *extractor) emitField(typeID, typeName string, fields *goextract.TypeEnv, fd *tree_sitter.Node) {
	nameNode := fd.ChildByFieldName("name")
	if nameNode == nil {
		nameNode = fd.ChildByFieldName("property")
	}
	if nameNode == nil {
		return
	}
	switch nameNode.Kind() {
	case "property_identifier", "private_property_identifier":
	default:
		return
	}
	name := nameNode.Utf8Text(ex.src)
	qualifiedName := typeName + "." + name
	id := ex.alloc.Declare(goextract.KindField, qualifiedName, "", nameNode.StartByte())
	vis := methodVisibility(fd, ex.src)
	if nameNode.Kind() == "private_property_identifier" {
		vis = "private"
	}
	start, end := fd.StartPosition(), fd.EndPosition()
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          goextract.KindField,
		Name:          name,
		QualifiedName: qualifiedName,
		FilePath:      ex.relPath,
		Language:      ex.language,
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Visibility:    vis,
		IsExported:    vis == "public",
	}})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: typeID, Target: id, Kind: goextract.RefKindContains, Provenance: "ast",
	}})
	if t := fd.ChildByFieldName("type"); t != nil {
		ex.emitNamedTypeRef(id, t, goextract.RefKindTypeOf)
	}
	if value := fd.ChildByFieldName("value"); value != nil {
		ex.env = ex.bodyEnv(fields, typeName, fd, nil)
		ex.captureInitializer(id, value, true)
		ex.env = nil
	}
}

// captureInitializer records what the initializer value of the data
// symbol fromID reads, and the calls it makes when it runs. The body of a
// function or a class in the value does not run then, so its calls are
// not the symbol's. whole is false for a name that a destructuring
// pattern binds: it takes one part of the value, which the extractor
// does not follow, so no name is its value.
func (ex *extractor) captureInitializer(fromID string, value *tree_sitter.Node, whole bool) {
	before := len(ex.result.Unresolved)
	ex.captureExprRead(fromID, value)
	switch {
	case !whole:
		ex.unknownWrite(fromID)
	case ex.valueRef(value) != nil:
		if len(ex.result.Unresolved) > before {
			// The whole value is one name, whose ref captureExprRead
			// appended last: the symbol starts out bound to it.
			ex.result.Unresolved[len(ex.result.Unresolved)-1].Binding = true
		}
	case mayHoldFunction(value):
		ex.unknownWrite(fromID)
	}
	walkDescendants(value, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "arrow_function", "function_expression", "generator_function", "class":
			return false
		case "call_expression":
			ex.recordCall(fromID, n)
		}
		return true
	})
}

// unknownWrite records that the symbol id holds a value that has no
// name, such as a function expression, so no later write is its only
// value.
func (ex *extractor) unknownWrite(id string) {
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: goextract.RefKindReferences, Provenance: "ast",
		Metadata: map[string]string{goextract.MetaAccess: goextract.AccessWrite, goextract.MetaBinds: goextract.BindsUnknown},
	}})
}

// mayHoldFunction reports whether the value of expr can be a function. A
// literal, an object or array literal and an operator expression cannot.
func mayHoldFunction(expr *tree_sitter.Node) bool {
	switch expr.Kind() {
	case "number", "string", "template_string", "regex", "true", "false", "null", "undefined",
		"object", "array", "binary_expression", "unary_expression", "update_expression":
		return false
	}
	return true
}

// valueRef returns the name that expr is, when expr is a bare identifier
// or a member access on a name or on this, and nil for any other
// expression.
func (ex *extractor) valueRef(expr *tree_sitter.Node) *goextract.ValueRef {
	for expr != nil && expr.Kind() == "parenthesized_expression" {
		expr = expr.NamedChild(0)
	}
	if expr == nil {
		return nil
	}
	switch expr.Kind() {
	case "identifier":
		alias, name := ex.bareName(expr)
		return &goextract.ValueRef{Name: name, PkgAlias: alias}
	case "member_expression":
		object, prop := expr.ChildByFieldName("object"), expr.ChildByFieldName("property")
		if object == nil || prop == nil || (object.Kind() != "identifier" && object.Kind() != "this") {
			return nil
		}
		alias, receiverType := ex.receiver(ex.env, object)
		return &goextract.ValueRef{Name: prop.Utf8Text(ex.src), PkgAlias: alias, ReceiverType: receiverType}
	}
	return nil
}

// bareName returns the alias and the name that a bare identifier
// reference carries. A local or a parameter in scope shadows a module
// name, and a named import names its target's own declaration.
func (ex *extractor) bareName(n *tree_sitter.Node) (alias, name string) {
	text := n.Utf8Text(ex.src)
	if shadow := goextract.ShadowAlias(ex.env, text, uint32(n.StartByte())); shadow != "" {
		return shadow, text
	}
	return ex.resolveBareIdentifier(text)
}

// captureWrite records the write target expr of an assignment with
// access, and value, the name it assigns, or nil. A subscript target
// reads the container it changes.
func (ex *extractor) captureWrite(fromID string, expr *tree_sitter.Node, access string, value *goextract.ValueRef) {
	if expr == nil {
		return
	}
	switch expr.Kind() {
	case "identifier":
		alias, name := ex.bareName(expr)
		pos := expr.StartPosition()
		ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
			FromID: fromID, Name: name, PkgAlias: alias, Kind: goextract.RefKindReferences,
			Line: int32(pos.Row) + 1, Col: int32(pos.Column), Access: access, Value: value,
		})
	case "member_expression":
		before := len(ex.result.Unresolved)
		ex.captureMemberAccess(fromID, expr, access)
		if value != nil && len(ex.result.Unresolved) > before {
			ex.result.Unresolved[len(ex.result.Unresolved)-1].Value = value
		}
	case "parenthesized_expression":
		if expr.NamedChildCount() > 0 {
			ex.captureWrite(fromID, expr.NamedChild(0), access, value)
		}
	case "object_pattern", "array_pattern":
		for _, n := range patternIdentifiers(expr) {
			ex.captureWrite(fromID, n, access, nil)
		}
	case "shorthand_property_identifier_pattern":
		ex.captureWrite(fromID, expr, access, nil)
	default:
		ex.captureExprRead(fromID, expr)
	}
}

// collectModuleWrites records the assignments in module-level statements
// as writes by the file, so that a binding the module code reassigns
// does not keep its initializer as its only value.
func (ex *extractor) collectModuleWrites(root *tree_sitter.Node) {
	for i := uint(0); i < root.NamedChildCount(); i++ {
		stmt := root.NamedChild(i)
		switch stmt.Kind() {
		case "import_statement", "export_statement", "lexical_declaration", "variable_declaration",
			"function_declaration", "generator_function_declaration", "class_declaration",
			"abstract_class_declaration", "interface_declaration", "type_alias_declaration":
			continue
		}
		walkDescendants(stmt, func(n *tree_sitter.Node) bool {
			switch n.Kind() {
			case "arrow_function", "function_expression", "generator_function", "class", "class_declaration", "function_declaration":
				return false
			case "assignment_expression", "augmented_assignment_expression", "update_expression":
				ex.captureAssignment(ex.fileID, n)
			}
			return true
		})
	}
}

// captureAssignment records the write and the reads of one assignment,
// compound assignment or update expression n.
func (ex *extractor) captureAssignment(fromID string, n *tree_sitter.Node) {
	switch n.Kind() {
	case "assignment_expression":
		right := n.ChildByFieldName("right")
		var value *goextract.ValueRef
		if right != nil {
			value = ex.valueRef(right)
		}
		ex.captureWrite(fromID, n.ChildByFieldName("left"), goextract.AccessWrite, value)
		if right != nil {
			ex.captureExprRead(fromID, right)
		}
	case "augmented_assignment_expression":
		ex.captureWrite(fromID, n.ChildByFieldName("left"), goextract.AccessReadWrite, nil)
		if right := n.ChildByFieldName("right"); right != nil {
			ex.captureExprRead(fromID, right)
		}
	case "update_expression":
		ex.captureWrite(fromID, n.ChildByFieldName("argument"), goextract.AccessReadWrite, nil)
	}
}
