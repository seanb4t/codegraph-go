package resolve

import (
	"strconv"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/dispatch"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/symbols"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// resolver resolves the calls of pass 2 against candidate sets (design D3
// ): the overloads of a name under a
// receiver's type, the functions of a module, or the members of the
// calling method's own type. It selects as the language does, by arity
// and then by argument type, and it routes an undecidable call to the
// name's overload set (D14).
type resolver struct {
	view symbols.View
	h    symbols.Hierarchy

	// callTargets maps a file and a call position to the node the call
	// resolved to, for the deferred types of D17.
	callTargets map[callPos]string

	// fileNodes indexes the declarations of the file fileNodesPath by
	// name, for the same-file lookup by lexical scope (D15). It holds one
	// file at a time: a lookup in another file replaces it.
	fileNodes     map[string][]goextract.ExtractedNode
	fileNodesPath string
}

// callPos is the file and end position of a call.
type callPos struct {
	file      string
	line, col int32
}

// callOutcome is what resolveCall decides for one call. At most one of
// target, set, deferred and waitName is set. The zero outcome is a call
// that does not resolve and names nothing to wait on: a receiver whose
// deferred type did not fill, or a synthetic alias. Such a call is
// counted as unresolved and leaves no trace in File.UnresolvedNames, so
// no later sync revisits it on its own.
type callOutcome struct {
	// target is the one candidate the call reaches, or "".
	target string
	// set is the overload set the call reaches when several candidates
	// remain, or nil.
	set *symbols.OverloadSet
	// deferred reports that the call depends on a deferred type that no
	// resolved call fills yet.
	deferred bool
	// waitModule and waitName are the name the file waits on when the
	// call does not resolve, or "" for none.
	waitModule, waitName string
}

// waitOn is the outcome of a call that does not resolve and that the
// file waits on: a later sync that declares name in module reparses the
// file.
func waitOn(module, name string) callOutcome {
	if name == "" {
		return callOutcome{}
	}
	return callOutcome{waitModule: module, waitName: name}
}

// knownType reports whether the resolver can judge typ: a builtin of
// lang, or a type the graph declares.
func (rs *resolver) knownType(lang, typ string) bool {
	base := baseTypeName(typ)
	return dispatch.Builtin(lang, base) || rs.view.IsTypeName(base)
}

// subtypeOf reports whether a type named arg extends, implements or
// embeds a type named param, through the graph's supertype edges.
func (rs *resolver) subtypeOf(arg, param string) bool {
	for _, id := range rs.view.TypesNamed(arg) {
		found := rs.h.WalkSupertypes(id, func(sup string) bool {
			s := rs.view.Node(sup)
			return s != nil && s.Name == param
		})
		if found {
			return true
		}
	}
	return false
}

// recordTarget remembers where a call resolved, for deferred types. The
// key is the call's end position, which is unique per link of a chain.
func (rs *resolver) recordTarget(file string, ref goextract.UnresolvedRef, target string) {
	if ref.EndLine == 0 {
		// A reference with no end, such as a route handler, is never
		// the operand of another call.
		return
	}
	rs.callTargets[callPos{file: file, line: ref.EndLine, col: ref.EndCol}] = target
}

// parseDeferredPosition reads the "line:col" of a deferred type, as
// goextract.DeferredType writes it. Text that DeferredType does not
// write, such as a sign or a leading zero, does not parse.
func parseDeferredPosition(typ string) (line, col int32, ok bool) {
	pos, ok := strings.CutPrefix(typ, goextract.DeferredTypePrefix)
	if !ok {
		return 0, 0, false
	}
	lineText, colText, ok := strings.Cut(pos, ":")
	if !ok {
		return 0, 0, false
	}
	l, lineErr := strconv.ParseInt(lineText, 10, 32)
	c, colErr := strconv.ParseInt(colText, 10, 32)
	if lineErr != nil || colErr != nil || l < 0 || c < 0 || strconv.FormatInt(l, 10) != lineText || strconv.FormatInt(c, 10) != colText {
		return 0, 0, false
	}
	return int32(l), int32(c), true
}

// isDeferred reports whether typ is a deferred type (D17).
func isDeferred(typ string) bool {
	return strings.HasPrefix(typ, goextract.DeferredTypePrefix)
}

// hasDeferred reports whether the call's receiver or an argument has a
// deferred type.
func hasDeferred(ref goextract.UnresolvedRef) bool {
	if isDeferred(ref.ReceiverType) {
		return true
	}
	for _, t := range ref.ArgTypes {
		if isDeferred(t) {
			return true
		}
	}
	return false
}

// fillDeferred replaces a deferred type with the type of the call it
// names: the return type of a function or method, the type itself for a
// constructor-like call, or the return type that every overload of a set
// shares. It also returns the module the type is written in, the
// callee's, where the type resolves first. It returns false when the
// call did not resolve, when the declaration has no return type, or when
// the overloads of a set return different types.
func (rs *resolver) fillDeferred(file, typ string) (filled, module string, ok bool) {
	if !isDeferred(typ) {
		return typ, "", true
	}
	line, col, ok := parseDeferredPosition(typ)
	if !ok {
		return "", "", false
	}
	target, ok := rs.callTargets[callPos{file: file, line: line, col: col}]
	if !ok {
		return "", "", false
	}
	n := rs.view.Node(target)
	if n == nil {
		return "", "", false
	}
	module = rs.view.ModuleOf(n.FilePath)
	switch n.Kind {
	case goextract.KindStruct, goextract.KindInterface, goextract.KindTypeAlias:
		return n.Name, module, true
	case goextract.KindFunction, goextract.KindMethod:
		return n.ReturnTypeName, module, n.ReturnTypeName != ""
	case goextract.KindOverloadSet:
		// An ambiguous call still has a type when every overload returns
		// the same one.
		set, ok := rs.h.Set(n.Id)
		if !ok || len(set.Members()) == 0 {
			return "", "", false
		}
		members := set.Members()
		typ := members[0].Node.ReturnTypeName
		for _, m := range members {
			if m.Node.ReturnTypeName != typ {
				return "", "", false
			}
		}
		return typ, rs.view.ModuleOf(members[0].FilePath), typ != ""
	}
	return "", "", false
}

// baseTypeName reduces a type name to the name a type node has: no type
// arguments, no array brackets, no package qualifier.
func baseTypeName(typ string) string {
	t := goextract.TypeName(typ)
	if i := strings.IndexAny(t, "<("); i > 0 {
		t = t[:i]
	}
	t = strings.TrimPrefix(t, "[]")
	for strings.HasSuffix(t, "[]") {
		t = strings.TrimSuffix(t, "[]")
	}
	if i := strings.LastIndexAny(t, ".:"); i >= 0 {
		t = t[i+1:]
	}
	return strings.TrimSpace(t)
}

// findType finds the type node that typ names from file r, in the
// language's lookup order (D17, D18): the module the type was written
// in when it came from a deferred fill, the same-file declaration whose
// scope holds the reference (D15), the module an alias names, the
// module a simple-name import names, the caller's own module, and the
// module a qualifier names. It returns the module the type belongs to,
// which is also the module to wait on when the type is missing.
func (rs *resolver) findType(r goextract.FileResult, alias, typ, writtenIn string, line, col int32) (symbols.Candidate, string, bool) {
	base := baseTypeName(typ)
	if base == "" {
		return symbols.Candidate{}, "", false
	}
	waitModule := r.ImportPath
	if writtenIn != "" {
		waitModule = writtenIn
		if c, ok := rs.view.TypeIn(writtenIn, base); ok {
			return c, writtenIn, true
		}
	}
	if alias == "" && !strings.ContainsAny(goextract.TypeName(typ), ".:") {
		if local := rs.sameFile(r, base, line, col, symbols.IsTypeKind); len(local) > 0 {
			winner := symbols.Pick(local)
			for _, c := range local {
				if c.ID == winner {
					return c, r.ImportPath, true
				}
			}
		}
	}
	if alias != "" {
		if mod, ok := r.Imports[alias]; ok {
			waitModule = mod
			if c, ok := rs.view.TypeIn(mod, base); ok {
				return c, mod, true
			}
		}
	}
	if mod, ok := r.Imports[base]; ok {
		waitModule = mod
		if c, ok := rs.view.TypeIn(mod, base); ok {
			return c, mod, true
		}
	}
	if c, ok := rs.view.TypeIn(r.ImportPath, base); ok {
		return c, r.ImportPath, true
	}
	if i := strings.LastIndexAny(goextract.TypeName(typ), ".:"); i > 0 {
		qualifier := strings.TrimRight(goextract.TypeName(typ)[:i], ":")
		if mod, ok := r.Imports[qualifier]; ok {
			waitModule = mod
			if c, ok := rs.view.TypeIn(mod, base); ok {
				return c, mod, true
			}
		}
	}
	return symbols.Candidate{}, waitModule, false
}

// sameFile returns the declarations named name in file r that are
// visible at (line, col), by lexical scope (D15): the declarations in
// the innermost local scope that contains the position, or else the
// file's module-level declarations of that name. Only kinds that want
// accepts count.
func (rs *resolver) sameFile(r goextract.FileResult, name string, line, col int32, want func(kind string) bool) []symbols.Candidate {
	if rs.fileNodes == nil || rs.fileNodesPath != r.RelPath {
		rs.fileNodesPath = r.RelPath
		rs.fileNodes = make(map[string][]goextract.ExtractedNode, len(r.Nodes))
		for _, en := range r.Nodes {
			if symbols.BareNameCandidate(en.Node.Kind) {
				rs.fileNodes[en.Node.Name] = append(rs.fileNodes[en.Node.Name], en)
			}
		}
	}
	byName := rs.fileNodes
	var innermost []symbols.Candidate
	var scope goextract.Range
	var moduleLevel []symbols.Candidate
	for _, en := range byName[name] {
		if !want(en.Node.Kind) {
			continue
		}
		switch {
		case !en.Node.Nested:
			moduleLevel = append(moduleLevel, symbols.CandidateOf(en.Node))
		case !en.Scope.Contains(line, col):
		case len(innermost) == 0 || en.Scope.Within(scope):
			innermost, scope = []symbols.Candidate{symbols.CandidateOf(en.Node)}, en.Scope
		case en.Scope == scope:
			innermost = append(innermost, symbols.CandidateOf(en.Node))
		}
	}
	if len(innermost) > 0 {
		return innermost
	}
	return moduleLevel
}

func callableKind(kind string) bool {
	return kind == goextract.KindFunction || kind == goextract.KindMethod
}

// resolveRef resolves a reference that is not a call: an embeds,
// returns, type_of, instantiates, contains or references ref. A bare
// name resolves in the file's lexical scope first (D15), then in the
// module index. An alias names a module through the file's imports.
// When typeOnly is set the reference means a type, so a type candidate
// wins over a function or method of the same name, as a Go method's
// receiver names its type and never a function.
func (rs *resolver) resolveRef(r goextract.FileResult, ref goextract.UnresolvedRef, typeOnly bool) (string, bool) {
	want := func(kind string) bool { return kind != goextract.KindFile && kind != goextract.KindOverloadSet }
	if typeOnly {
		want = symbols.IsTypeKind
	}
	var cands []symbols.Candidate
	if ref.PkgAlias == "" {
		cands = rs.sameFile(r, ref.Name, ref.Line, ref.Col, want)
		if len(cands) == 0 {
			cands = rs.view.CandidatesIn(r.ImportPath, ref.Name)
		}
	} else {
		mod, ok := r.Imports[ref.PkgAlias]
		if !ok {
			return "", false
		}
		cands = rs.view.CandidatesIn(mod, ref.Name)
	}
	var kept []symbols.Candidate
	for _, c := range cands {
		if want(c.Kind) {
			kept = append(kept, c)
		}
	}
	if len(kept) == 0 {
		return "", false
	}
	if len(kept) == 1 {
		return kept[0].ID, true
	}
	if last, ok := lastDefinition(r.Language, kept); ok {
		return last, true
	}
	return symbols.Pick(kept), true
}

// resolveDataRef resolves a references ref. A ref through a receiver of
// known type, as `p.X`, names a member of that type: a field, or a method
// used as a value. It never falls back to a module-level name, so `p.X`
// cannot reach an unrelated declaration named X. A receiver that is both
// an alias and a type name, as `Point.X` or `pkg.X`, names a member of
// the type when one exists, and a module-level name of the module the
// alias imports otherwise. Every other ref resolves as resolveRef does.
func (rs *resolver) resolveDataRef(r goextract.FileResult, ref goextract.UnresolvedRef) (string, bool) {
	recv := ref.ReceiverType
	if recv == "" {
		return rs.resolveRef(r, ref, false)
	}
	moduleAlias := ref.PkgAlias != "" && ref.PkgAlias == recv
	if owner, _, ok := rs.findType(r, ref.PkgAlias, recv, "", ref.Line, ref.Col); ok {
		switch ms := rs.ownMembers(r.Language, owner.ID, ref.Name); len(ms) {
		case 0:
		case 1:
			return ms[0].ID, true
		default:
			return symbols.Pick(ms), true
		}
	}
	if moduleAlias {
		return rs.resolveRef(r, ref, false)
	}
	return "", false
}

// inheritedRef reports whether the references ref names a member of a
// receiver type that the type itself does not declare, in a language
// whose classes have the members of the classes they extend. Such a ref
// resolves through the class hierarchy, so Calls resolves it and Types
// leaves it.
func (rs *resolver) inheritedRef(r goextract.FileResult, ref goextract.UnresolvedRef) bool {
	recv := ref.ReceiverType
	if ref.Kind != goextract.RefKindReferences || recv == "" || !InheritsMembers(r.Language) {
		return false
	}
	owner, _, ok := rs.findType(r, ref.PkgAlias, recv, "", ref.Line, ref.Col)
	return ok && len(rs.view.Members(owner.ID, ref.Name)) == 0
}

// dataRefMetadata returns the metadata of the references edge of ref:
// its access role, goextract.BindingInit when ref is the whole initializer of its
// source, and for a write the value it binds, or goextract.BindsUnknown.
func (rs *resolver) dataRefMetadata(r goextract.FileResult, ref goextract.UnresolvedRef) map[string]string {
	if ref.Access == "" && !ref.Binding {
		return nil
	}
	md := map[string]string{}
	if ref.Access != "" {
		md[goextract.MetaAccess] = ref.Access
	}
	if ref.Binding {
		md[goextract.MetaBinding] = goextract.BindingInit
	}
	if goextract.IsWriteAccess(ref.Access) {
		md[goextract.MetaBinds] = goextract.BindsUnknown
		if v := ref.Value; v != nil && ref.Access == goextract.AccessWrite {
			if id, ok := rs.resolveDataRef(r, goextract.UnresolvedRef{
				FromID: ref.FromID, Name: v.Name, PkgAlias: v.PkgAlias, ReceiverType: v.ReceiverType,
				Kind: goextract.RefKindReferences, Line: ref.Line, Col: ref.Col,
			}); ok {
				md[goextract.MetaBinds] = id
			}
		}
	}
	return md
}

// membersOf returns the call candidates named name among the members of
// the type ownerID, or of the nearest supertype that declares one. A
// method wins over a field of the same name. With no method, the field
// that ownMembers finds is the candidate, and a call through it becomes
// a read of the field.
func (rs *resolver) membersOf(lang, ownerID, name string) []symbols.Candidate {
	if ms := callMembers(rs.view.Members(ownerID, name)); len(ms) > 0 {
		return ms
	}
	var found []symbols.Candidate
	rs.h.WalkSupertypes(ownerID, func(sup string) bool {
		found = callMembers(rs.view.Members(sup, name))
		return len(found) > 0
	})
	if len(found) > 0 {
		return found
	}
	for _, m := range rs.ownMembers(lang, ownerID, name) {
		if m.Kind == goextract.KindField {
			found = append(found, m)
		}
	}
	return found
}

// ownMembers returns the members named name of the type ownerID. In a
// language with class inheritance, a type that declares none has the
// members of the nearest supertype that declares the name, less the
// private ones, which a subtype cannot reach. Go has no such step: an
// embedded struct's fields are not the embedding type's members here.
func (rs *resolver) ownMembers(lang, ownerID, name string) []symbols.Candidate {
	ms := rs.view.Members(ownerID, name)
	if len(ms) > 0 || !InheritsMembers(lang) {
		return ms
	}
	rs.h.WalkSupertypes(ownerID, func(sup string) bool {
		declared := rs.view.Members(sup, name)
		for _, m := range declared {
			if inheritable(lang, m.Node) {
				ms = append(ms, m)
			}
		}
		return len(declared) > 0
	})
	return ms
}

// InheritsMembers reports whether a class of lang has the data members of
// the classes it extends.
func InheritsMembers(lang string) bool {
	switch lang {
	case "typescript", "tsx", "javascript", "python", "java":
		return true
	}
	return false
}

// inheritable reports whether a subclass written in lang reaches the
// member n of a base class. A private member is out of reach, and in
// Python a name with two leading underscores and no trailing pair is
// mangled with its class name, so a subclass reaches another attribute.
func inheritable(lang string, n *schema.Node) bool {
	if lang == "python" {
		name := n.GetName()
		return !strings.HasPrefix(name, "__") || strings.HasSuffix(name, "__")
	}
	return n.GetVisibility() != "private"
}

// callMembers returns ms without its fields.
func callMembers(ms []symbols.Candidate) []symbols.Candidate {
	var kept []symbols.Candidate
	for _, m := range ms {
		if m.Kind != goextract.KindField {
			kept = append(kept, m)
		}
	}
	return kept
}

// implicitThis reports whether a bare call in a method body of lang can
// name a method of the enclosing type.
func implicitThis(lang string) bool {
	switch lang {
	case "java", "csharp", "kotlin", "swift", "ruby", "cpp", "c":
		return true
	}
	return false
}

// resolveCall decides where ref, a call in file r, goes. When
// allowDeferred is false, a call with a deferred type comes back as
// deferred, to be resolved after the calls it depends on.
func (rs *resolver) resolveCall(r goextract.FileResult, ref goextract.UnresolvedRef, allowDeferred bool) callOutcome {
	recv := ref.ReceiverType
	recvWrittenIn := ""
	argTypes := ref.ArgTypes
	if hasDeferred(ref) {
		if !allowDeferred {
			return callOutcome{deferred: true}
		}
		filled, module, ok := rs.fillDeferred(r.RelPath, recv)
		if isDeferred(recv) && !ok {
			// The call it depends on did not resolve to one declaration
			// with a return type: the receiver is unknown, and there is
			// no name to wait on.
			return callOutcome{}
		}
		recv, recvWrittenIn = filled, module
		argTypes = make([]string, len(ref.ArgTypes))
		for i, t := range ref.ArgTypes {
			argTypes[i], _, _ = rs.fillDeferred(r.RelPath, t)
		}
	}

	switch {
	case recv != "":
		owner, module, ok := rs.findType(r, ref.PkgAlias, recv, recvWrittenIn, ref.Line, ref.Col)
		if ok {
			cands := rs.membersOf(r.Language, owner.ID, ref.Name)
			if len(cands) == 0 && ref.PkgAlias != "" && ref.PkgAlias == recv {
				// The alias named a module after all: `util.Parse()`.
				if mod, ok := r.Imports[ref.PkgAlias]; ok {
					cands = rs.view.CandidatesIn(mod, ref.Name)
					module = mod
				}
			}
			return rs.choose(r, ref, cands, argTypes, module)
		}
		if ref.PkgAlias != "" && ref.PkgAlias == recv {
			if mod, ok := r.Imports[ref.PkgAlias]; ok {
				return rs.choose(r, ref, rs.view.CandidatesIn(mod, ref.Name), argTypes, mod)
			}
		}
		return waitOn(module, ref.Name)
	case ref.PkgAlias != "":
		mod, ok := r.Imports[ref.PkgAlias]
		if !ok {
			// A synthetic alias: a receiver whose type is unknown.
			return callOutcome{}
		}
		return rs.choose(r, ref, rs.view.CandidatesIn(mod, ref.Name), argTypes, mod)
	default:
		// A bare name: the innermost same-file declaration in scope, the
		// enclosing type's member, the file's module-level declaration,
		// then the module's cross-file index (D15).
		// A call through a data symbol is a read of it, which Assemble
		// turns into the calls its bindings establish.
		local := rs.sameFile(r, ref.Name, ref.Line, ref.Col, func(kind string) bool {
			return callableKind(kind) || symbols.IsTypeKind(kind) || IsDataKind(kind)
		})
		if len(local) > 0 && local[0].Node.Nested {
			return rs.choose(r, ref, local, argTypes, r.ImportPath)
		}
		if implicitThis(r.Language) {
			if owner, ok := rs.h.OwnerOf(ref.FromID); ok {
				if cands := rs.membersOf(r.Language, owner, ref.Name); len(cands) > 0 {
					return rs.choose(r, ref, cands, argTypes, r.ImportPath)
				}
			}
		}
		if len(local) > 0 {
			return rs.choose(r, ref, local, argTypes, r.ImportPath)
		}
		return rs.choose(r, ref, rs.view.CandidatesIn(r.ImportPath, ref.Name), argTypes, r.ImportPath)
	}
}

// choose picks among cands as the language does: in a last-wins
// language the last definition per file first (D16), then arity, labels
// and argument types (D3). One survivor resolves. Several survivors that
// are overloads of one owner reach their set. Several survivors that are
// not overloads go to lastDefinition, and when that declines, to
// pickSymbol.
func (rs *resolver) choose(r goextract.FileResult, ref goextract.UnresolvedRef, cands []symbols.Candidate, argTypes []string, module string) callOutcome {
	if len(cands) == 0 {
		return waitOn(module, ref.Name)
	}
	callables := make([]symbols.Candidate, 0, len(cands))
	others := make([]symbols.Candidate, 0, len(cands))
	for _, c := range cands {
		switch c.Kind {
		case goextract.KindFunction, goextract.KindMethod:
			callables = append(callables, c)
		case goextract.KindOverloadSet:
			// A set is reached through its members, never as a candidate.
		default:
			others = append(others, c)
		}
	}
	if len(callables) > 1 && symbols.LastWins(r.Language) {
		// A redefinition replaces the earlier one at run time, whatever
		// its parameters (D16): only the last copy in each file is a
		// candidate, before any filter reads its arity.
		callables = lastPerFile(callables)
	}
	if len(callables) > 1 {
		callables = rs.filterCallables(r.Language, ref, callables, argTypes)
	}
	if len(callables) > 1 {
		callables = rs.mostSpecific(r.Language, callables)
	}
	remaining := append(callables, others...)
	switch len(remaining) {
	case 0:
		return waitOn(module, ref.Name)
	case 1:
		return callOutcome{target: remaining[0].ID}
	}
	if set := rs.h.SetFor(remaining); set != nil {
		return callOutcome{set: set}
	}
	if last, ok := lastDefinition(r.Language, remaining); ok {
		return callOutcome{target: last}
	}
	return callOutcome{target: symbols.Pick(remaining)}
}

// lastPerFile keeps, per file, the candidate that is last in source
// order.
func lastPerFile(cands []symbols.Candidate) []symbols.Candidate {
	last := make(map[string]symbols.Candidate)
	var order []string
	for _, c := range cands {
		prev, seen := last[c.FilePath]
		if !seen {
			order = append(order, c.FilePath)
			last[c.FilePath] = c
			continue
		}
		if c.Node.StartLine > prev.Node.StartLine || (c.Node.StartLine == prev.Node.StartLine && c.Node.StartCol > prev.Node.StartCol) {
			last[c.FilePath] = c
		}
	}
	out := make([]symbols.Candidate, 0, len(order))
	for _, f := range order {
		out = append(out, last[f])
	}
	return out
}

// lastDefinition applies the runtime rule of Python, Ruby and JavaScript
// to redefinitions in one file (design D16): the last definition of a
// name wins. It returns false for any other language, for candidates in
// several files, and for candidates that are not copies of one
// declaration.
func lastDefinition(lang string, cands []symbols.Candidate) (string, bool) {
	if !symbols.LastWins(lang) {
		return "", false
	}
	first := cands[0]
	last := first
	for _, c := range cands[1:] {
		if c.FilePath != first.FilePath || c.Kind != first.Kind || c.Key != first.Key {
			return "", false
		}
		if c.Node.StartLine > last.Node.StartLine || (c.Node.StartLine == last.Node.StartLine && c.Node.StartCol > last.Node.StartCol) {
			last = c
		}
	}
	return last.ID, true
}

// filterCallables applies the arity filter, then the label filter, then
// the type filter (D3). A candidate that fails a filter is out: when none
// is left, the call is unresolved. The type filter rejects only on two
// known types, so a foreign type cannot drop the right overload.
func (rs *resolver) filterCallables(lang string, ref goextract.UnresolvedRef, cands []symbols.Candidate, argTypes []string) []symbols.Candidate {
	keep := func(in []symbols.Candidate, fits func(symbols.Candidate) bool) []symbols.Candidate {
		var out []symbols.Candidate
		for _, c := range in {
			if fits(c) {
				out = append(out, c)
			}
		}
		return out
	}
	cands = keep(cands, func(c symbols.Candidate) bool { return arityFits(c.Node.Params, ref.ArgCount, ref.HasSpread) })
	if len(ref.ArgLabels) > 0 {
		cands = keep(cands, func(c symbols.Candidate) bool { return labelsFit(lang, c, ref.ArgLabels) })
	}
	cands = keep(cands, func(c symbols.Candidate) bool { return rs.typesFit(lang, c.Node.Params, argTypes, ref.HasSpread) })
	return cands
}

// arityFits reports whether argc arguments can call a function with
// params: at least the required parameters, and at most all of them
// unless one is variadic. A spread argument stands for any number of
// arguments, so with one the call fits when the other argc-1 arguments
// fit in params.
func arityFits(params []*schema.Param, argc int32, spread bool) bool {
	required, variadic := 0, false
	for _, p := range params {
		if p.Variadic {
			variadic = true
			continue
		}
		if !p.Optional && !p.HasDefault {
			required++
		}
	}
	n := int(argc)
	if spread {
		// The spread argument can stand for any number of arguments.
		return variadic || n-1 <= len(params)
	}
	if n < required {
		return false
	}
	return variadic || n <= len(params)
}

// labelsFit checks keyword and labeled arguments. Swift labels are part
// of the qualified name, move(to:by:), and a call's unlabeled argument
// matches a `_` label. In every other language a labeled argument must
// name a parameter.
func labelsFit(lang string, c symbols.Candidate, labels []string) bool {
	if lang == "swift" {
		declared := swiftLabels(c.QualifiedName)
		if declared == nil {
			return true
		}
		if len(labels) > len(declared) {
			return false
		}
		for i, l := range labels {
			switch {
			case l == "" && declared[i] != "_":
				return false
			case l != "" && l != declared[i]:
				return false
			}
		}
		return true
	}
	names := make(map[string]bool, len(c.Node.Params))
	for _, p := range c.Node.Params {
		names[p.Name] = true
	}
	for _, l := range labels {
		if l != "" && !names[l] {
			return false
		}
	}
	return true
}

// swiftLabels returns the labels in a Swift qualified name, or nil when
// the name has none written.
func swiftLabels(qualifiedName string) []string {
	i := strings.IndexByte(qualifiedName, '(')
	if i < 0 || !strings.HasSuffix(qualifiedName, ")") {
		return nil
	}
	inner := qualifiedName[i+1 : len(qualifiedName)-1]
	if inner == "" {
		return []string{}
	}
	return strings.Split(strings.TrimSuffix(inner, ":"), ":")
}

// mostSpecific applies the overload rule the languages share: when one
// candidate's parameter types are each assignable to the matching
// parameter of every other candidate, and not the reverse, that
// candidate wins, as `add(String)` wins over `add(Object)` for a String
// argument. Otherwise every candidate stays.
func (rs *resolver) mostSpecific(lang string, cands []symbols.Candidate) []symbols.Candidate {
	assignable := func(from, to string) bool {
		from, to = unqualified(goextract.TypeName(from)), unqualified(goextract.TypeName(to))
		if from == to {
			return true
		}
		if !rs.knownType(lang, from) || !rs.knownType(lang, to) {
			return false
		}
		return dispatch.Compatible(lang, from, to) || rs.subtypeOf(baseTypeName(from), baseTypeName(to))
	}
	narrower := func(a, b symbols.Candidate) bool {
		if len(a.Node.Params) != len(b.Node.Params) {
			return false
		}
		strict := false
		for i := range a.Node.Params {
			pa, pb := a.Node.Params[i].Type, b.Node.Params[i].Type
			if !assignable(pa, pb) {
				return false
			}
			if !assignable(pb, pa) {
				strict = true
			}
		}
		return strict
	}
	for _, c := range cands {
		wins := true
		for _, o := range cands {
			if o.ID != c.ID && !narrower(c, o) {
				wins = false
				break
			}
		}
		if wins {
			return []symbols.Candidate{c}
		}
	}
	return cands
}

// typesFit reports whether each known argument type is compatible with
// the parameter at its position. Extra arguments compare with the
// variadic parameter's element type. A foreign type on either side
// passes, and so does a deferred argument type that did not fill: an
// unknown type never rejects an overload (D3).
func (rs *resolver) typesFit(lang string, params []*schema.Param, argTypes []string, spread bool) bool {
	for i, arg := range argTypes {
		if arg == "" || isDeferred(arg) {
			continue
		}
		var p *schema.Param
		switch {
		case i < len(params) && !params[i].Variadic:
			p = params[i]
		case len(params) > 0 && params[len(params)-1].Variadic:
			p = params[len(params)-1]
		default:
			return false
		}
		paramType := p.Type
		if p.Variadic {
			paramType = strings.TrimSuffix(strings.TrimSuffix(paramType, "[]"), "...")
		}
		if spread && i == len(argTypes)-1 {
			continue
		}
		a, pt := unqualified(goextract.TypeName(arg)), unqualified(goextract.TypeName(paramType))
		if !rs.canJudge(lang, a, pt) {
			continue
		}
		if dispatch.Compatible(lang, a, pt) || rs.subtypeOf(baseTypeName(a), baseTypeName(pt)) {
			continue
		}
		return false
	}
	return true
}

// unqualified drops the package or namespace prefix of a type name and
// keeps its type arguments: java.util.List<String> is List<String>, and
// loggers.Logger is Logger. A parameter type is written with the
// qualifier its file uses, while a filled deferred type is the bare name
// of a declaration, so the two compare without it.
func unqualified(t string) string {
	base, rest := t, ""
	if i := strings.IndexAny(t, "<(["); i > 0 {
		base, rest = t[:i], t[i:]
	}
	if i := strings.LastIndexAny(base, ".:"); i >= 0 {
		base = base[i+1:]
	}
	return base + rest
}

// canJudge reports whether a mismatch between an argument type and a
// parameter type can reject a candidate. Both must be known, and the
// parameter type must be one the comparison can see the whole of: a
// final type, which nothing extends, or a type of the index whose
// supertype chain is in view, compared with an argument that is final or
// in the index too. A library interface or class as the parameter, such
// as CharSequence or Number, never rejects: a String is a CharSequence
// and an int boxes to a Number through relations the index cannot see.
func (rs *resolver) canJudge(lang, arg, param string) bool {
	if !rs.knownType(lang, arg) || !rs.knownType(lang, param) {
		return false
	}
	ab, pb := baseTypeName(arg), baseTypeName(param)
	if dispatch.Final(lang, pb) {
		return true
	}
	return rs.view.IsTypeName(pb) && (dispatch.Final(lang, ab) || rs.view.IsTypeName(ab))
}

// SynthesizedBySet marks the contains edges of an overload set. They are
// synthesized over the whole graph, as implements and overrides are, so
// a sync keeps, adds or deletes each one.
const SynthesizedBySet = "overload-set"

// newResolver returns a resolver for one file over view and h.
func newResolver(view symbols.View, h symbols.Hierarchy) *resolver {
	return &resolver{view: view, h: h, callTargets: make(map[callPos]string)}
}
