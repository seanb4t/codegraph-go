package javaextract

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// emitFields emits one field for each declarator of each field
// declaration in the body of the type typeName, and records the type and
// the initializer of each. A field of an interface is public.
func (ex *extractor) emitFields(typeID, typeName string, body *tree_sitter.Node, iface bool) {
	fields := ex.fieldsOf(body)
	for i := uint(0); i < body.NamedChildCount(); i++ {
		fd := body.NamedChild(i)
		if fd.Kind() != "field_declaration" && fd.Kind() != "constant_declaration" {
			continue
		}
		vis := javaVisibility(findModifiersChild(fd))
		if iface {
			vis = "public"
		}
		for j := uint(0); j < fd.NamedChildCount(); j++ {
			d := fd.NamedChild(j)
			if d.Kind() != "variable_declarator" {
				continue
			}
			nameNode := d.ChildByFieldName("name")
			if nameNode == nil {
				continue
			}
			id := ex.emitField(typeID, typeName, nameNode.Utf8Text(ex.src), vis, fd, nameNode)
			ex.emitNamedTypeRef(id, fd.ChildByFieldName("type"), goextract.RefKindTypeOf)
			if value := d.ChildByFieldName("value"); value != nil {
				ex.env = goextract.NewTypeEnv(fields)
				ex.env.Declare("this", typeName)
				ex.fields, ex.owner = fields, typeName
				ex.captureInitializer(id, value)
				ex.env, ex.fields, ex.owner = nil, nil, ""
			}
		}
	}
}

// emitField emits the field name of the type typeName that the
// declaration fd declares at nameNode, and returns its id.
func (ex *extractor) emitField(typeID, typeName, name, vis string, fd, nameNode *tree_sitter.Node) string {
	qualifiedName := typeName + "." + name
	id := ex.alloc.Declare(goextract.KindField, qualifiedName, "", nameNode.StartByte())
	start, end := fd.StartPosition(), fd.EndPosition()
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          goextract.KindField,
		Name:          name,
		QualifiedName: qualifiedName,
		FilePath:      ex.relPath,
		Language:      "java",
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
	return id
}

// captureInitializer records what the initializer value of the field
// fromID reads and creates, and the calls it makes when it runs. The
// body of a lambda or an anonymous class in the value does not run then.
func (ex *extractor) captureInitializer(fromID string, value *tree_sitter.Node) {
	ex.captureExprRead(fromID, value)
	walkDescendants(value, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "lambda_expression", "class_body":
			return false
		case "method_invocation":
			ex.recordCall(fromID, n)
		}
		return true
	})
}

// captureName records a reference with access to the bare name n. A
// parameter or a local in scope is not a field. Any other name of a field
// of the enclosing type reads it through the implicit receiver this. Any
// other name may be a field the enclosing type inherits, so it names a
// member of the enclosing type when one exists.
func (ex *extractor) captureName(fromID string, n *tree_sitter.Node, access string) {
	name := n.Utf8Text(ex.src)
	pos := n.StartPosition()
	ref := goextract.UnresolvedRef{
		FromID: fromID, Name: name, Kind: goextract.RefKindReferences,
		Line: int32(pos.Row) + 1, Col: int32(pos.Column), Access: access,
	}
	switch _, field := ex.fields.Lookup(name); {
	case ex.env.DeclaredAt(name, uint32(n.StartByte())):
		ref.PkgAlias = goextract.LocalAlias(name)
	case field:
		ref.ReceiverType = ex.owner
	case ex.owner != "":
		ref.ReceiverType, ref.ImplicitReceiver = ex.owner, true
	}
	ex.result.Unresolved = append(ex.result.Unresolved, ref)
}

// captureWrite records the target expr of an assignment or an update
// with access. An array element target reads the array it changes.
func (ex *extractor) captureWrite(fromID string, expr *tree_sitter.Node, access string) {
	switch expr.Kind() {
	case "identifier":
		ex.captureName(fromID, expr, access)
	case "field_access":
		ex.captureFieldAccess(fromID, expr, access)
	case "parenthesized_expression":
		if expr.NamedChildCount() > 0 {
			ex.captureWrite(fromID, expr.NamedChild(0), access)
		}
	default:
		ex.captureExprRead(fromID, expr)
	}
}

// captureAssignment records the write and the reads of the assignment n.
// A compound assignment such as += reads its target too.
func (ex *extractor) captureAssignment(fromID string, n *tree_sitter.Node) {
	access := goextract.AccessReadWrite
	if op := n.ChildByFieldName("operator"); op != nil && op.Kind() == "=" {
		access = goextract.AccessWrite
	}
	if left := n.ChildByFieldName("left"); left != nil {
		ex.captureWrite(fromID, left, access)
	}
	if right := n.ChildByFieldName("right"); right != nil {
		ex.captureExprRead(fromID, right)
	}
}
