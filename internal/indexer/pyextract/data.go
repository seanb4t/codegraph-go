package pyextract

import (
	"sort"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// receiverKind is what the first parameter of a method stands for.
type receiverKind int

const (
	// noReceiver: a static method, a function, or a method under a
	// decorator that can change its first argument.
	noReceiver receiverKind = iota
	instanceReceiver
	classReceiver
)

// instanceDecorators are the decorators that keep the first parameter of
// a method an instance of its class.
var instanceDecorators = map[string]bool{
	"property": true, "functools.cached_property": true, "cached_property": true,
	"abstractmethod": true, "abc.abstractmethod": true, "override": true, "typing.override": true,
}

// methodReceiver returns what the first parameter of the method decl
// stands for, and its name. Only a function directly in a class body has
// a receiver. An explicit classmethod has the class, a staticmethod
// none, and any decorator outside instanceDecorators leaves it unknown.
func (ex *extractor) methodReceiver(decl *tree_sitter.Node) (receiverKind, string) {
	if ex.enclosingClass(decl) == "" {
		return noReceiver, ""
	}
	params, _ := collectParams(decl, ex.src)
	if len(params) == 0 {
		return noReceiver, ""
	}
	kind := instanceReceiver
	if p := decl.Parent(); p != nil && p.Kind() == "decorated_definition" {
		for i := uint(0); i < p.NamedChildCount(); i++ {
			d := p.NamedChild(i)
			if d.Kind() != "decorator" || d.NamedChildCount() == 0 {
				continue
			}
			text := d.NamedChild(0).Utf8Text(ex.src)
			switch {
			case text == "staticmethod":
				return noReceiver, ""
			case text == "classmethod":
				kind = classReceiver
			case instanceDecorators[text], strings.HasSuffix(text, ".setter"), strings.HasSuffix(text, ".getter"), strings.HasSuffix(text, ".deleter"):
			default:
				return noReceiver, ""
			}
		}
	}
	return kind, params[0].Name
}

// staticMethod reports whether a staticmethod decorator wraps decl.
func (ex *extractor) staticMethod(decl *tree_sitter.Node) bool {
	p := decl.Parent()
	if p == nil || p.Kind() != "decorated_definition" {
		return false
	}
	for i := uint(0); i < p.NamedChildCount(); i++ {
		if d := p.NamedChild(i); d.Kind() == "decorator" && d.NamedChildCount() > 0 && d.NamedChild(0).Utf8Text(ex.src) == "staticmethod" {
			return true
		}
	}
	return false
}

// assignTargets returns the identifiers that the target of an assignment
// binds, and whether the target is one name, which takes the whole value.
func assignTargets(left *tree_sitter.Node) ([]*tree_sitter.Node, bool) {
	if left.Kind() == "identifier" {
		return []*tree_sitter.Node{left}, true
	}
	var out []*tree_sitter.Node
	var visit func(n *tree_sitter.Node)
	visit = func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "identifier":
			out = append(out, n)
		case "pattern_list", "tuple_pattern", "list_pattern", "list_splat_pattern":
			for i := uint(0); i < n.NamedChildCount(); i++ {
				visit(n.NamedChild(i))
			}
		}
	}
	visit(left)
	return out, false
}

// walkScope visits the nodes of root that run in root's own scope: it
// leaves out the bodies of functions, classes and lambdas.
func walkScope(root *tree_sitter.Node, visit func(*tree_sitter.Node)) {
	walkDescendants(root, func(n *tree_sitter.Node) bool {
		switch n.Kind() {
		case "function_definition", "class_definition", "lambda", "decorated_definition":
			return n == root
		}
		visit(n)
		return true
	})
}

// collectModuleData emits a variable for each name that a module-level
// assignment or annotation binds, outside functions and classes. The
// first binding of a name is its declaration and holds its initializer.
// A later assignment, and an assignment to a name that an import, a
// function or a class declares, is a write by the file.
func (ex *extractor) collectModuleData(root *tree_sitter.Node) {
	declared := make(map[string]bool)
	for alias := range ex.result.Imports {
		declared[alias] = true
	}
	for i := uint(0); i < root.NamedChildCount(); i++ {
		d := unwrapDecorated(root.NamedChild(i))
		switch d.Kind() {
		case "function_definition", "class_definition":
			if n := d.ChildByFieldName("name"); n != nil {
				declared[n.Utf8Text(ex.src)] = true
			}
		}
	}
	vars := make(map[string]string)
	walkScope(root, func(n *tree_sitter.Node) {
		switch n.Kind() {
		case "assignment":
			left := n.ChildByFieldName("left")
			if left == nil {
				return
			}
			right := n.ChildByFieldName("right")
			targets, whole := assignTargets(left)
			if left.Kind() == "attribute" || left.Kind() == "subscript" {
				ex.captureAssignment(ex.fileID, n)
				return
			}
			readDone := false
			for _, t := range targets {
				name := t.Utf8Text(ex.src)
				if _, seen := vars[name]; !seen && !declared[name] {
					id := ex.emitModuleVariable(name, t, n)
					vars[name] = id
					ex.emitNamedTypeRef(id, n.ChildByFieldName("type"), goextract.RefKindTypeOf)
					if right != nil {
						ex.captureInitializer(id, right, whole)
					}
					continue
				}
				if right == nil {
					continue
				}
				var value *goextract.ValueRef
				if whole {
					value = ex.valueRef(right)
				}
				ex.captureWrite(ex.fileID, t, goextract.AccessWrite, value)
				if !readDone {
					ex.captureExprRead(ex.fileID, right)
					readDone = true
				}
			}
		case "augmented_assignment":
			ex.captureAssignment(ex.fileID, n)
		}
	})
}

// emitModuleVariable emits the variable name that the target t of the
// statement stmt declares, and returns its id.
func (ex *extractor) emitModuleVariable(name string, t, stmt *tree_sitter.Node) string {
	id := ex.alloc.Declare(goextract.KindVariable, name, "", t.StartByte())
	start, end := stmt.StartPosition(), stmt.EndPosition()
	vis := pythonVisibility(name)
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          goextract.KindVariable,
		Name:          name,
		QualifiedName: name,
		FilePath:      ex.relPath,
		Language:      "python",
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Visibility:    vis,
		IsExported:    vis == "public",
	}})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: goextract.RefKindContains, Provenance: "ast",
	}})
	return id
}

// fieldDecl is one place that declares the attribute of a class: a class
// body assignment or annotation, or a write through the receiver of one
// of its methods.
type fieldDecl struct {
	name   *tree_sitter.Node
	stmt   *tree_sitter.Node
	inBody bool
}

// emitFieldsForClass emits one field for each attribute name that the
// class body binds, or that a method writes through its receiver: an
// instance method through its first parameter, an explicit classmethod
// through its class parameter. The earliest such place anchors the
// field. A name the class declares as a method, such as a property, gets
// no field. A class body binding holds the field's initializer, and each
// later one is a write by the class.
func (ex *extractor) emitFieldsForClass(classID, className string, body *tree_sitter.Node) {
	methods := make(map[string]bool)
	var decls []fieldDecl
	for i := uint(0); i < body.NamedChildCount(); i++ {
		stmt := body.NamedChild(i)
		if fn := unwrapDecorated(stmt); fn.Kind() == "function_definition" {
			if n := fn.ChildByFieldName("name"); n != nil {
				methods[n.Utf8Text(ex.src)] = true
			}
			decls = append(decls, ex.receiverWrites(fn)...)
			continue
		}
		if stmt.Kind() != "expression_statement" || stmt.NamedChildCount() == 0 {
			continue
		}
		if a := stmt.NamedChild(0); a.Kind() == "assignment" {
			if left := a.ChildByFieldName("left"); left != nil && left.Kind() == "identifier" {
				decls = append(decls, fieldDecl{name: left, stmt: a, inBody: true})
			}
		}
	}
	sort.SliceStable(decls, func(i, j int) bool { return decls[i].name.StartByte() < decls[j].name.StartByte() })
	fields := make(map[string]string)
	for _, d := range decls {
		name := d.name.Utf8Text(ex.src)
		if methods[name] {
			continue
		}
		if _, seen := fields[name]; !seen {
			id := ex.emitField(classID, className, name, d)
			fields[name] = id
			if !d.inBody {
				continue
			}
			ex.emitNamedTypeRef(id, d.stmt.ChildByFieldName("type"), goextract.RefKindTypeOf)
			if right := d.stmt.ChildByFieldName("right"); right != nil {
				ex.captureInitializer(id, right, true)
			}
			continue
		}
		if d.inBody {
			if right := d.stmt.ChildByFieldName("right"); right != nil {
				pos := d.name.StartPosition()
				ex.result.Unresolved = append(ex.result.Unresolved, goextract.UnresolvedRef{
					FromID: classID, Name: name, PkgAlias: "", ReceiverType: className, Kind: goextract.RefKindReferences,
					Line: int32(pos.Row) + 1, Col: int32(pos.Column), Access: goextract.AccessWrite, Value: ex.valueRef(right),
				})
				ex.captureExprRead(classID, right)
			}
		}
	}
}

// receiverWrites returns the attributes that the method fn assigns or
// annotates through its receiver.
func (ex *extractor) receiverWrites(fn *tree_sitter.Node) []fieldDecl {
	kind, recv := ex.methodReceiver(fn)
	body := fn.ChildByFieldName("body")
	if kind == noReceiver || body == nil {
		return nil
	}
	var out []fieldDecl
	walkScope(body, func(n *tree_sitter.Node) {
		if n.Kind() != "assignment" && n.Kind() != "augmented_assignment" {
			return
		}
		left := n.ChildByFieldName("left")
		if left == nil || left.Kind() != "attribute" {
			return
		}
		obj, attr := left.ChildByFieldName("object"), left.ChildByFieldName("attribute")
		if obj != nil && attr != nil && obj.Kind() == "identifier" && obj.Utf8Text(ex.src) == recv {
			out = append(out, fieldDecl{name: attr, stmt: n})
		}
	})
	return out
}

// emitField emits the field name of the class className, anchored at d,
// and returns its id.
func (ex *extractor) emitField(classID, className, name string, d fieldDecl) string {
	qualifiedName := className + "." + name
	id := ex.alloc.Declare(goextract.KindField, qualifiedName, "", d.name.StartByte())
	start, end := d.stmt.StartPosition(), d.stmt.EndPosition()
	vis := pythonVisibility(name)
	ex.result.Nodes = append(ex.result.Nodes, goextract.ExtractedNode{Node: &schema.Node{
		Id:            id,
		Kind:          goextract.KindField,
		Name:          name,
		QualifiedName: qualifiedName,
		FilePath:      ex.relPath,
		Language:      "python",
		StartLine:     int32(start.Row) + 1,
		EndLine:       int32(end.Row) + 1,
		StartCol:      int32(start.Column),
		EndCol:        int32(end.Column),
		Visibility:    vis,
		IsExported:    vis == "public",
	}})
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: classID, Target: id, Kind: goextract.RefKindContains, Provenance: "ast",
	}})
	return id
}

// captureInitializer records what the initializer value of the data
// symbol fromID reads, and the calls it makes when it runs. The body of a
// lambda in the value does not run then. whole is false for a name that a
// tuple or list target binds: it takes one part of the value, so no name
// is its value.
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
		case "lambda":
			return false
		case "call":
			ex.recordCall(fromID, n)
		}
		return true
	})
}

// unknownWrite records that the symbol id holds a value that has no
// name, such as a lambda, so no later write is its only value.
func (ex *extractor) unknownWrite(id string) {
	ex.result.IntraEdges = append(ex.result.IntraEdges, goextract.IntraEdge{Edge: &schema.Edge{
		Source: ex.fileID, Target: id, Kind: goextract.RefKindReferences, Provenance: "ast",
		Metadata: map[string]string{goextract.MetaAccess: goextract.AccessWrite, goextract.MetaBinds: goextract.BindsUnknown},
	}})
}

// mayHoldFunction reports whether the value of expr can be a function. A
// literal, a container display and an operator expression cannot.
func mayHoldFunction(expr *tree_sitter.Node) bool {
	switch expr.Kind() {
	case "integer", "float", "string", "concatenated_string", "true", "false", "none",
		"list", "tuple", "dictionary", "set", "list_comprehension", "dictionary_comprehension",
		"set_comprehension", "binary_operator", "unary_operator", "not_operator", "comparison_operator",
		"boolean_operator":
		return false
	}
	return true
}

// valueRef returns the name that expr is, when expr is a bare identifier
// or an attribute of a name, and nil for any other expression.
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
	case "attribute":
		object, attr := expr.ChildByFieldName("object"), expr.ChildByFieldName("attribute")
		if object == nil || attr == nil || object.Kind() != "identifier" {
			return nil
		}
		alias, receiverType := ex.receiver(ex.env, object)
		return &goextract.ValueRef{Name: attr.Utf8Text(ex.src), PkgAlias: alias, ReceiverType: receiverType}
	}
	return nil
}

// bareName returns the alias and the name that a bare identifier
// reference carries. A parameter or a local of the body shadows a module
// name, and a from import names its module's own declaration.
func (ex *extractor) bareName(n *tree_sitter.Node) (alias, name string) {
	text := n.Utf8Text(ex.src)
	if _, local := ex.env.Lookup(text); local {
		return goextract.LocalAlias(text), text
	}
	if origin, ok := ex.fromImportOrigin[text]; ok {
		return text, origin
	}
	return "", text
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
	case "attribute":
		before := len(ex.result.Unresolved)
		ex.captureAttribute(fromID, expr, access)
		if value != nil && len(ex.result.Unresolved) > before {
			ex.result.Unresolved[len(ex.result.Unresolved)-1].Value = value
		}
	case "pattern_list", "tuple_pattern", "list_pattern":
		targets, _ := assignTargets(expr)
		for _, t := range targets {
			ex.captureWrite(fromID, t, access, nil)
		}
	case "parenthesized_expression":
		if expr.NamedChildCount() > 0 {
			ex.captureWrite(fromID, expr.NamedChild(0), access, value)
		}
	default:
		ex.captureExprRead(fromID, expr)
	}
}

// captureAssignment records the write and the reads of one assignment or
// augmented assignment n. An annotation without a value writes nothing.
func (ex *extractor) captureAssignment(fromID string, n *tree_sitter.Node) {
	left, right := n.ChildByFieldName("left"), n.ChildByFieldName("right")
	if right == nil {
		return
	}
	switch n.Kind() {
	case "assignment":
		var value *goextract.ValueRef
		if left != nil && (left.Kind() == "identifier" || left.Kind() == "attribute") {
			value = ex.valueRef(right)
		}
		ex.captureWrite(fromID, left, goextract.AccessWrite, value)
	case "augmented_assignment":
		ex.captureWrite(fromID, left, goextract.AccessReadWrite, nil)
	}
	ex.captureExprRead(fromID, right)
}

// globalNames returns the names that a global or nonlocal statement of
// body names: the body's assignments to them write the outer binding.
func (ex *extractor) globalNames(body *tree_sitter.Node) map[string]bool {
	names := make(map[string]bool)
	walkScope(body, func(n *tree_sitter.Node) {
		if n.Kind() != "global_statement" && n.Kind() != "nonlocal_statement" {
			return
		}
		for i := uint(0); i < n.NamedChildCount(); i++ {
			if id := n.NamedChild(i); id.Kind() == "identifier" {
				names[id.Utf8Text(ex.src)] = true
			}
		}
	})
	return names
}
