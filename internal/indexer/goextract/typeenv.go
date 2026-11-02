package goextract

import (
	"fmt"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// TypeEnv holds the names that one function body can see and their static
// types (design D17 of the index-invariants change): the parameters, the
// locals, the fields of the enclosing type through a parent environment,
// and the receiver word (`this`, `self`). A name declared with an unknown
// type is in the environment with the type "". A name declared twice with
// different types becomes "", because the extractor does not follow
// control flow.
type TypeEnv struct {
	parent *TypeEnv
	names  map[string][]binding
}

// binding is one declaration of a name: its type, the byte where the
// declaration starts and the byte where its scope ends. A parameter, a
// field or a receiver word is visible in the whole body: at 0, end
// WholeBody.
type binding struct {
	typ     string
	at, end uint32
}

// WholeBody is the end of a binding that the whole body can see.
const WholeBody = ^uint32(0)

// NewTypeEnv returns an empty environment whose lookups fall back to
// parent. parent can be nil.
func NewTypeEnv(parent *TypeEnv) *TypeEnv {
	return &TypeEnv{parent: parent, names: make(map[string][]binding)}
}

// Declare records name with its type as written, or "" for none, visible
// in the whole body.
func (e *TypeEnv) Declare(name, typ string) {
	e.DeclareIn(name, typ, 0, WholeBody)
}

// DeclareIn records a declaration of name at byte at whose scope ends at
// byte end: a local is visible from its declaration to the end of the
// block that holds it. A second declaration with the same span and a
// different type makes the type unknown.
func (e *TypeEnv) DeclareIn(name, typ string, at, end uint32) {
	if e == nil || name == "" {
		return
	}
	typ = TypeName(typ)
	for i, b := range e.names[name] {
		if b.at == at && b.end == end {
			if b.typ != typ {
				e.names[name][i].typ = ""
			}
			return
		}
	}
	e.names[name] = append(e.names[name], binding{typ: typ, at: at, end: end})
}

// Lookup returns name's type and whether name is declared anywhere in
// the body. When the declarations disagree the type is "".
func (e *TypeEnv) Lookup(name string) (string, bool) {
	for env := e; env != nil; env = env.parent {
		bs, ok := env.names[name]
		if !ok {
			continue
		}
		typ := bs[0].typ
		for _, b := range bs[1:] {
			if b.typ != typ {
				return "", true
			}
		}
		return typ, true
	}
	return "", false
}

// LookupAt returns the type of the declaration of name that the position
// pos can see: the latest one whose scope holds pos. A local declared
// after pos, or in another block, is not visible, so an import alias
// that a later local shadows still names the import there.
func (e *TypeEnv) LookupAt(name string, pos uint32) (string, bool) {
	for env := e; env != nil; env = env.parent {
		var best *binding
		for i := range env.names[name] {
			b := &env.names[name][i]
			if b.at <= pos && pos < b.end && (best == nil || b.at >= best.at) {
				best = b
			}
		}
		if best != nil {
			return best.typ, true
		}
	}
	return "", false
}

// DeferredType is the type of an expression that is a call: pass 2 fills
// it from the return type of the call at n's position (D17).
func DeferredType(n *tree_sitter.Node) string {
	pos := n.StartPosition()
	return fmt.Sprintf("%s%d:%d", DeferredTypePrefix, pos.Row+1, pos.Column)
}

// CallArgs is the argument data of one call site.
type CallArgs struct {
	Count  int32
	Spread bool
	Types  []string
	Labels []string
}

// Receiver classifies a bare receiver name without reading its case
// (D18), in the language's lookup order. A declared name with a known
// type gives that type. A declared name with an unknown type cannot
// resolve, which LocalAlias marks. An imported name gives its alias and,
// as a candidate type, the name itself, so pass 2 tries a type of that
// name in the imported module first and a module-level name second. Any
// other name is a candidate type for pass 2 to find in the caller's own
// module, or to leave unresolved.
func Receiver(env *TypeEnv, imports map[string]string, name string) (pkgAlias, receiverType string) {
	return classifyReceiver(env.Lookup(name))(imports, name)
}

// ReceiverAt is Receiver as seen from byte pos: only a declaration whose
// scope holds pos counts.
func ReceiverAt(env *TypeEnv, imports map[string]string, name string, pos uint32) (pkgAlias, receiverType string) {
	return classifyReceiver(env.LookupAt(name, pos))(imports, name)
}

func classifyReceiver(typ string, declared bool) func(imports map[string]string, name string) (string, string) {
	return func(imports map[string]string, name string) (string, string) {
		if declared {
			if typ == "" {
				return LocalAlias(name), ""
			}
			return "", typ
		}
		if _, ok := imports[name]; ok {
			return name, name
		}
		return "", name
	}
}

// LocalAlias is the PkgAlias of a reference through a variable whose
// type is unknown. It matches no import, so the reference stays
// unresolved instead of matching a same-named symbol by accident.
func LocalAlias(name string) string {
	return "<local:" + name + ">"
}

// UnknownAlias is the PkgAlias of a reference through an expression the
// extractor does not read, named by the expression's syntax kind.
func UnknownAlias(kind string) string {
	return "<" + kind + ">"
}
