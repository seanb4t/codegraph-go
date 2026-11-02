package indexer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/dispatch"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// resolver resolves the calls of pass 2 against candidate sets (design D3
// of the index-invariants change): the overloads of a name under a
// receiver's type, the functions of a module, or the members of the
// calling method's own type. It selects as the language does, by arity
// and then by argument type, and it routes an undecidable call to the
// name's overload set (D14).
type resolver struct {
	idx          *symbolIndex
	nodeKindByID map[string]string
	nodeByID     map[string]*schema.Node

	// typeSupertypes maps a type to the types it embeds, extends or
	// implements, and ownerOf maps a method to the type that contains it.
	// Both come from the edges of phase 1.
	typeSupertypes map[string][]string
	ownerOf        map[string]string

	// callTargets maps a file and a call position to the node the call
	// resolved to, for the deferred types of D17.
	callTargets map[string]map[string]string

	// sets holds every overload set by id.
	sets map[string]*overloadSet

	// storedSets are the sets the store holds for the files a sync did
	// not reparse, so a set that dissolved gives its members back to
	// their owner.
	storedSets map[string]storedSet

	// typeNames are the names of the type nodes the graph holds, for the
	// known-type rule of the type filter.
	typeNames map[string]bool

	// moduleOfFile maps each file to its module key. A deferred type is
	// written in the callee's file, so it resolves in that module first.
	moduleOfFile map[string]string

	// fileNodes indexes the declarations of each file of this pass by
	// name, for the same-file lookup by lexical scope (D15).
	fileNodes map[string]map[string][]goextract.ExtractedNode
}

// storedSet is an overload set as the store holds it: its owner and its
// members.
type storedSet struct {
	owner   string
	members []string
}

// overloadSet is the node that holds two or more overloads of one name
// under one owner.
type overloadSet struct {
	node    *schema.Node
	ownerID string
	members []symbolCandidate
}

// callOutcome is what resolveCall decides for one call.
type callOutcome struct {
	// target is the one candidate the call reaches, or "".
	target string
	// set is the overload set the call reaches when several candidates
	// remain, or nil.
	set *overloadSet
	// deferred reports that the call depends on a deferred type that no
	// resolved call fills yet.
	deferred bool
	// waitModule and waitName are the name the file waits on when the
	// call does not resolve, or "" for none.
	waitModule, waitName string
}

func newResolver(idx *symbolIndex, nodeKindByID map[string]string, nodeByID map[string]*schema.Node, storedSets map[string]storedSet, moduleOfFile map[string]string) *resolver {
	rs := &resolver{
		idx:            idx,
		nodeKindByID:   nodeKindByID,
		nodeByID:       nodeByID,
		typeSupertypes: make(map[string][]string),
		ownerOf:        make(map[string]string),
		callTargets:    make(map[string]map[string]string),
		sets:           make(map[string]*overloadSet),
		storedSets:     storedSets,
		typeNames:      make(map[string]bool),
		moduleOfFile:   moduleOfFile,
		fileNodes:      make(map[string]map[string][]goextract.ExtractedNode),
	}
	for _, n := range nodeByID {
		if isTypeKind(n.Kind) {
			rs.typeNames[n.Name] = true
		}
	}
	return rs
}

func isTypeKind(kind string) bool {
	switch kind {
	case goextract.KindStruct, goextract.KindInterface, goextract.KindTypeAlias:
		return true
	}
	return false
}

// learnEdges reads the supertype and ownership edges of phase 1. A
// member the store holds under an overload set belongs to the set's
// owner.
func (rs *resolver) learnEdges(edges []*schema.Edge) {
	for _, e := range edges {
		switch e.Kind {
		case "embeds", goextract.EdgeKindExtends, goextract.EdgeKindImplements:
			rs.typeSupertypes[e.Source] = append(rs.typeSupertypes[e.Source], e.Target)
		case goextract.RefKindContains:
			if isTypeKind(rs.nodeKindByID[e.Source]) {
				rs.ownerOf[e.Target] = e.Source
			}
		}
	}
	for _, set := range rs.storedSets {
		if !isTypeKind(rs.nodeKindByID[set.owner]) {
			continue
		}
		for _, m := range set.members {
			rs.ownerOf[m] = set.owner
		}
	}
}

// knownType reports whether the resolver can judge typ: a builtin of
// lang, or a type the graph declares.
func (rs *resolver) knownType(lang, typ string) bool {
	base := baseTypeName(typ)
	return dispatch.Builtin(lang, base) || rs.typeNames[base]
}

// subtypeOf reports whether a type named arg extends, implements or
// embeds a type named param, through the graph's supertype edges.
func (rs *resolver) subtypeOf(arg, param string) bool {
	for id, n := range rs.nodeByID {
		if n.Name != arg || !isTypeKind(n.Kind) {
			continue
		}
		found := walkSupertypes(id, rs.typeSupertypes, func(sup string) bool {
			s := rs.nodeByID[sup]
			return s != nil && s.Name == param
		})
		if found {
			return true
		}
	}
	return false
}

// recordTarget remembers where a call resolved, for deferred types.
func (rs *resolver) recordTarget(file string, ref goextract.UnresolvedRef, target string) {
	byPos := rs.callTargets[file]
	if byPos == nil {
		byPos = make(map[string]string)
		rs.callTargets[file] = byPos
	}
	byPos[fmt.Sprintf("%d:%d", ref.Line, ref.Col)] = target
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
// names: the return type of a function or method, or the type itself for
// a constructor-like call. It also returns the module the type is
// written in, the callee's, where the type resolves first. It returns
// false when that call did not resolve to one declaration.
func (rs *resolver) fillDeferred(file, typ string) (filled, module string, ok bool) {
	if !isDeferred(typ) {
		return typ, "", true
	}
	target, ok := rs.callTargets[file][strings.TrimPrefix(typ, goextract.DeferredTypePrefix)]
	if !ok {
		return "", "", false
	}
	n := rs.nodeByID[target]
	if n == nil {
		return "", "", false
	}
	module = rs.moduleOfFile[n.FilePath]
	switch n.Kind {
	case goextract.KindStruct, goextract.KindInterface, goextract.KindTypeAlias:
		return n.Name, module, true
	case goextract.KindFunction, goextract.KindMethod:
		return n.ReturnTypeName, module, n.ReturnTypeName != ""
	case goextract.KindOverloadSet:
		// An ambiguous call still has a type when every overload returns
		// the same one.
		set, ok := rs.sets[n.Id]
		if !ok || len(set.members) == 0 {
			return "", "", false
		}
		typ := set.members[0].node.ReturnTypeName
		for _, m := range set.members {
			if m.node.ReturnTypeName != typ {
				return "", "", false
			}
		}
		return typ, rs.moduleOfFile[set.members[0].filePath], typ != ""
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
// in when it came from a deferred fill, the module an alias names, the
// module a simple-name import names, the caller's own module, and the
// module a qualifier names. It returns the module the type belongs to,
// which is also the module to wait on when the type is missing.
func (rs *resolver) findType(r goextract.FileResult, alias, typ, writtenIn string, line, col int32) (symbolCandidate, string, bool) {
	base := baseTypeName(typ)
	if base == "" {
		return symbolCandidate{}, "", false
	}
	waitModule := r.ImportPath
	if writtenIn != "" {
		waitModule = writtenIn
		if c, ok := rs.idx.typeIn(writtenIn, base); ok {
			return c, writtenIn, true
		}
	}
	if alias == "" && !strings.ContainsAny(goextract.TypeName(typ), ".:") {
		if local := rs.sameFile(r, base, line, col, isTypeKind); len(local) > 0 {
			winner := pickSymbol(local)
			for _, c := range local {
				if c.id == winner {
					return c, r.ImportPath, true
				}
			}
		}
	}
	if alias != "" {
		if mod, ok := r.Imports[alias]; ok {
			waitModule = mod
			if c, ok := rs.idx.typeIn(mod, base); ok {
				return c, mod, true
			}
		}
	}
	if mod, ok := r.Imports[base]; ok {
		waitModule = mod
		if c, ok := rs.idx.typeIn(mod, base); ok {
			return c, mod, true
		}
	}
	if c, ok := rs.idx.typeIn(r.ImportPath, base); ok {
		return c, r.ImportPath, true
	}
	if i := strings.LastIndexAny(goextract.TypeName(typ), ".:"); i > 0 {
		qualifier := strings.TrimRight(goextract.TypeName(typ)[:i], ":")
		if mod, ok := r.Imports[qualifier]; ok {
			waitModule = mod
			if c, ok := rs.idx.typeIn(mod, base); ok {
				return c, mod, true
			}
		}
	}
	return symbolCandidate{}, waitModule, false
}

// sameFile returns the declarations named name in file r that are
// visible at (line, col), by lexical scope (D15): the declarations in
// the innermost local scope that contains the position, or else the
// file's module-level declarations of that name. Only kinds that want
// accepts count.
func (rs *resolver) sameFile(r goextract.FileResult, name string, line, col int32, want func(kind string) bool) []symbolCandidate {
	byName, ok := rs.fileNodes[r.RelPath]
	if !ok {
		byName = make(map[string][]goextract.ExtractedNode)
		for _, en := range r.Nodes {
			if en.Node.Kind != goextract.KindFile {
				byName[en.Node.Name] = append(byName[en.Node.Name], en)
			}
		}
		rs.fileNodes[r.RelPath] = byName
	}
	var innermost []symbolCandidate
	var scope goextract.Range
	var moduleLevel []symbolCandidate
	for _, en := range byName[name] {
		if !want(en.Node.Kind) {
			continue
		}
		switch {
		case !en.Node.Nested:
			moduleLevel = append(moduleLevel, candidateOf(en.Node))
		case !en.Scope.Contains(line, col):
		case len(innermost) == 0 || en.Scope.Within(scope):
			innermost, scope = []symbolCandidate{candidateOf(en.Node)}, en.Scope
		case en.Scope == scope:
			innermost = append(innermost, candidateOf(en.Node))
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
		want = isTypeKind
	}
	var cands []symbolCandidate
	if ref.PkgAlias == "" {
		cands = rs.sameFile(r, ref.Name, ref.Line, ref.Col, want)
		if len(cands) == 0 {
			cands = rs.idx.candidatesIn(r.ImportPath, ref.Name)
		}
	} else {
		mod, ok := r.Imports[ref.PkgAlias]
		if !ok {
			return "", false
		}
		cands = rs.idx.candidatesIn(mod, ref.Name)
	}
	var kept []symbolCandidate
	for _, c := range cands {
		if want(c.kind) {
			kept = append(kept, c)
		}
	}
	if len(kept) == 0 {
		if typeOnly && len(cands) > 0 {
			// No type of that name: fall back to any candidate, as
			// before, so a language whose types the extractor records
			// under another kind keeps its edges.
			kept = cands
		} else {
			return "", false
		}
	}
	if len(kept) == 1 {
		return kept[0].id, true
	}
	if last, ok := lastDefinition(r.Language, kept); ok {
		return last, true
	}
	return pickSymbol(kept), true
}

// membersOf returns the members named name of the type ownerID, or of the
// nearest supertype that declares the name.
func (rs *resolver) membersOf(ownerID, name string) []symbolCandidate {
	if ms := rs.idx.members(ownerID, name); len(ms) > 0 {
		return ms
	}
	var found []symbolCandidate
	walkSupertypes(ownerID, rs.typeSupertypes, func(sup string) bool {
		found = rs.idx.members(sup, name)
		return len(found) > 0
	})
	return found
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
			// with a return type: the receiver is unknown.
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
			cands := rs.membersOf(owner.id, ref.Name)
			if len(cands) == 0 && ref.PkgAlias != "" && ref.PkgAlias == recv {
				// The alias named a module after all: `util.Parse()`.
				if mod, ok := r.Imports[ref.PkgAlias]; ok {
					cands = rs.idx.candidatesIn(mod, ref.Name)
					module = mod
				}
			}
			return rs.choose(r, ref, cands, argTypes, module)
		}
		if ref.PkgAlias != "" && ref.PkgAlias == recv {
			if mod, ok := r.Imports[ref.PkgAlias]; ok {
				return rs.choose(r, ref, rs.idx.candidatesIn(mod, ref.Name), argTypes, mod)
			}
		}
		return callOutcome{waitModule: module, waitName: ref.Name}
	case ref.PkgAlias != "":
		mod, ok := r.Imports[ref.PkgAlias]
		if !ok {
			// A synthetic alias: a receiver whose type is unknown.
			return callOutcome{}
		}
		return rs.choose(r, ref, rs.idx.candidatesIn(mod, ref.Name), argTypes, mod)
	default:
		// A bare name: the innermost same-file declaration in scope, the
		// enclosing type's member, the file's module-level declaration,
		// then the module's cross-file index (D15).
		local := rs.sameFile(r, ref.Name, ref.Line, ref.Col, func(kind string) bool {
			return callableKind(kind) || isTypeKind(kind)
		})
		if len(local) > 0 && local[0].node.Nested {
			return rs.choose(r, ref, local, argTypes, r.ImportPath)
		}
		if implicitThis(r.Language) {
			if owner, ok := rs.ownerOf[ref.FromID]; ok {
				if cands := rs.membersOf(owner, ref.Name); len(cands) > 0 {
					return rs.choose(r, ref, cands, argTypes, r.ImportPath)
				}
			}
		}
		if len(local) > 0 {
			return rs.choose(r, ref, local, argTypes, r.ImportPath)
		}
		return rs.choose(r, ref, rs.idx.candidatesIn(r.ImportPath, ref.Name), argTypes, r.ImportPath)
	}
}

// choose picks among cands as the language does: arity first, argument
// types second (D3). One survivor resolves. Several survivors that are
// overloads of one owner reach their set. Several survivors that are not
// overloads, such as the copies of a redefinition, go to pickSymbol.
func (rs *resolver) choose(r goextract.FileResult, ref goextract.UnresolvedRef, cands []symbolCandidate, argTypes []string, module string) callOutcome {
	if len(cands) == 0 {
		return callOutcome{waitModule: module, waitName: ref.Name}
	}
	callables := make([]symbolCandidate, 0, len(cands))
	others := make([]symbolCandidate, 0, len(cands))
	for _, c := range cands {
		if c.kind == goextract.KindFunction || c.kind == goextract.KindMethod {
			callables = append(callables, c)
		} else {
			others = append(others, c)
		}
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
		return callOutcome{waitModule: module, waitName: ref.Name}
	case 1:
		return callOutcome{target: remaining[0].id}
	}
	if set := rs.setFor(remaining); set != nil {
		return callOutcome{set: set}
	}
	if last, ok := lastDefinition(r.Language, remaining); ok {
		return callOutcome{target: last}
	}
	return callOutcome{target: pickSymbol(remaining)}
}

// lastDefinition applies the runtime rule of Python, Ruby and JavaScript
// to redefinitions in one file (design D16): the last definition of a
// name wins. It returns false for any other language, for candidates in
// several files, and for candidates that are not copies of one
// declaration.
func lastDefinition(lang string, cands []symbolCandidate) (string, bool) {
	switch lang {
	case "python", "ruby", "javascript":
	default:
		return "", false
	}
	first := cands[0]
	last := first
	for _, c := range cands[1:] {
		if c.filePath != first.filePath || c.kind != first.kind || c.key != first.key {
			return "", false
		}
		if c.node.StartLine > last.node.StartLine || (c.node.StartLine == last.node.StartLine && c.node.StartCol > last.node.StartCol) {
			last = c
		}
	}
	return last.id, true
}

// filterCallables applies the arity filter, then the label filter, then
// the type filter (D3). A candidate that fails a filter is out: when none
// is left, the call is unresolved. The type filter rejects only on two
// known types, so a foreign type cannot drop the right overload.
func (rs *resolver) filterCallables(lang string, ref goextract.UnresolvedRef, cands []symbolCandidate, argTypes []string) []symbolCandidate {
	keep := func(in []symbolCandidate, fits func(symbolCandidate) bool) []symbolCandidate {
		var out []symbolCandidate
		for _, c := range in {
			if fits(c) {
				out = append(out, c)
			}
		}
		return out
	}
	cands = keep(cands, func(c symbolCandidate) bool { return arityFits(c.node.Params, ref.ArgCount, ref.HasSpread) })
	if len(ref.ArgLabels) > 0 {
		cands = keep(cands, func(c symbolCandidate) bool { return labelsFit(lang, c, ref.ArgLabels) })
	}
	cands = keep(cands, func(c symbolCandidate) bool { return rs.typesFit(lang, c.node.Params, argTypes, ref.HasSpread) })
	return cands
}

// arityFits reports whether argc arguments can call a function with
// params: at least the required parameters, and at most all of them
// unless one is variadic. A spread argument makes argc a lower bound on
// the arguments it stands for, so the required count can still be met.
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
// matches a `_` label. Python and Kotlin keyword arguments must name a
// parameter.
func labelsFit(lang string, c symbolCandidate, labels []string) bool {
	if lang == "swift" {
		declared := swiftLabels(c.qualifiedName)
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
	names := make(map[string]bool, len(c.node.Params))
	for _, p := range c.node.Params {
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
func (rs *resolver) mostSpecific(lang string, cands []symbolCandidate) []symbolCandidate {
	assignable := func(from, to string) bool {
		from, to = goextract.TypeName(from), goextract.TypeName(to)
		if from == to {
			return true
		}
		if !rs.knownType(lang, from) || !rs.knownType(lang, to) {
			return false
		}
		return dispatch.Compatible(lang, from, to) || rs.subtypeOf(baseTypeName(from), baseTypeName(to))
	}
	narrower := func(a, b symbolCandidate) bool {
		if len(a.node.Params) != len(b.node.Params) {
			return false
		}
		strict := false
		for i := range a.node.Params {
			pa, pb := a.node.Params[i].Type, b.node.Params[i].Type
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
			if o.id != c.id && !narrower(c, o) {
				wins = false
				break
			}
		}
		if wins {
			return []symbolCandidate{c}
		}
	}
	return cands
}

// typesFit reports whether each known argument type is compatible with
// the parameter at its position. Extra arguments compare with the
// variadic parameter's element type. A foreign type on either side
// passes.
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
		a, pt := goextract.TypeName(arg), goextract.TypeName(paramType)
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

// canJudge reports whether a mismatch between an argument type and a
// parameter type can reject a candidate. Both must be known. One of them
// must be final, or both must be types of the index, so that the
// supertype chain the comparison needs is in view: an indexed type can
// implement a library interface, and two library types can relate in
// ways the index cannot see.
func (rs *resolver) canJudge(lang, arg, param string) bool {
	if !rs.knownType(lang, arg) || !rs.knownType(lang, param) {
		return false
	}
	ab, pb := baseTypeName(arg), baseTypeName(param)
	if dispatch.Final(lang, ab) || dispatch.Final(lang, pb) {
		return true
	}
	return rs.typeNames[ab] && rs.typeNames[pb]
}

// setFor returns the overload set of cands when they are the overloads of
// one name under one owner: the same kind, the same owner, and distinct
// signature keys. Copies with one key are redefinitions, not overloads,
// and get no set, except in the languages where a repeated signature can
// only be a conditional-compilation alternative (design D16): there the
// copies form a set, and a call that fits both is ambiguous.
func (rs *resolver) setFor(cands []symbolCandidate) *overloadSet {
	owner, ok := rs.groupOwner(cands)
	if !ok {
		return nil
	}
	keys := make(map[string]bool, len(cands))
	for _, c := range cands {
		if keys[c.key] && !duplicatesAreAlternatives(c.node.Language) {
			return nil
		}
		keys[c.key] = true
	}
	return rs.setOf(owner, cands[0].node.Name, cands)
}

// duplicatesAreAlternatives reports whether two declarations with one
// signature under one owner are alternatives under `#if` or a
// `function_exists` guard in lang, since the language rejects a plain
// repeat (design D16).
func duplicatesAreAlternatives(lang string) bool {
	switch lang {
	case "c", "cpp", "csharp", "php":
		return true
	}
	return false
}

// groupOwner returns the owner shared by every candidate: the type that
// contains them, or the file that declares them, and false when they
// differ.
func (rs *resolver) groupOwner(cands []symbolCandidate) (string, bool) {
	owner := ""
	for i, c := range cands {
		o, ok := rs.ownerOf[c.id]
		if !ok {
			if c.kind != goextract.KindFunction {
				return "", false
			}
			o = nodeid.NodeID(goextract.KindFile, c.filePath, c.filePath)
		}
		if i == 0 {
			owner = o
		} else if o != owner || c.kind != cands[0].kind {
			return "", false
		}
	}
	return owner, owner != ""
}

// setOf returns the overload set for name under ownerID, creating its
// node from members on first use (D14). The node spans the members in
// their file, and takes the partial-type sentinel location when they
// span files.
func (rs *resolver) setOf(ownerID, name string, members []symbolCandidate) *overloadSet {
	id := nodeid.NodeID(goextract.KindOverloadSet, name, ownerID)
	if set, ok := rs.sets[id]; ok {
		return set
	}
	sorted := append([]symbolCandidate(nil), members...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].filePath != sorted[j].filePath {
			return sorted[i].filePath < sorted[j].filePath
		}
		return sorted[i].node.StartLine < sorted[j].node.StartLine
	})
	first := sorted[0].node
	n := &schema.Node{
		Id:            id,
		Kind:          goextract.KindOverloadSet,
		Name:          first.Name,
		QualifiedName: first.QualifiedName,
		Language:      first.Language,
		Visibility:    first.Visibility,
		IsExported:    first.IsExported,
	}
	sameFile := true
	for _, m := range sorted {
		if m.filePath != first.FilePath {
			sameFile = false
		}
	}
	if sameFile {
		last := sorted[len(sorted)-1].node
		n.FilePath = first.FilePath
		n.StartLine, n.StartCol = first.StartLine, first.StartCol
		n.EndLine, n.EndCol = last.EndLine, last.EndCol
		for _, m := range sorted {
			if m.node.EndLine > n.EndLine {
				n.EndLine, n.EndCol = m.node.EndLine, m.node.EndCol
			}
		}
	}
	set := &overloadSet{node: n, ownerID: ownerID, members: sorted}
	rs.sets[id] = set
	return set
}

// declareSets creates the set of every overloaded name in idx, so a set
// exists even when no call is ambiguous (D14): the members of each type,
// and the functions of each file.
func (rs *resolver) declareSets() {
	for ownerID, byName := range rs.idx.membersByOwner {
		for _, cands := range byName {
			for _, c := range cands {
				if _, ok := rs.ownerOf[c.id]; !ok {
					rs.ownerOf[c.id] = ownerID
				}
			}
			if len(cands) >= 2 {
				rs.setFor(cands)
			}
		}
	}
	for _, byName := range rs.idx.candidates {
		for _, cands := range byName {
			byFile := make(map[string][]symbolCandidate)
			for _, c := range cands {
				if c.kind == goextract.KindFunction {
					byFile[c.filePath] = append(byFile[c.filePath], c)
				}
			}
			for _, group := range byFile {
				if len(group) >= 2 {
					rs.setFor(group)
				}
			}
		}
	}
}

// setNodes returns the set nodes in id order.
func (rs *resolver) setNodes() []*schema.Node {
	ids := make([]string, 0, len(rs.sets))
	for id := range rs.sets {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]*schema.Node, len(ids))
	for i, id := range ids {
		out[i] = rs.sets[id].node
	}
	return out
}

// SynthesizedBySet marks the contains edges of an overload set. They are
// synthesized over the whole graph, as implements and overrides are, so
// a sync keeps, adds or deletes each one.
const SynthesizedBySet = "overload-set"

// rewireSetEdges replaces, for every set, the contains edges from the
// owner to each member with an edge from the owner to the set and one
// from the set to each member. Other edges pass through. A stored set
// that no longer exists gives each remaining member back to its owner.
func (rs *resolver) rewireSetEdges(edges []*schema.Edge) []*schema.Edge {
	setOfMember := make(map[string]*overloadSet)
	for _, set := range rs.sets {
		for _, m := range set.members {
			setOfMember[set.ownerID+"|"+m.id] = set
		}
	}
	out := make([]*schema.Edge, 0, len(edges)+2*len(rs.sets))
	for _, e := range edges {
		if e.Kind == goextract.RefKindContains {
			if _, drop := setOfMember[e.Source+"|"+e.Target]; drop {
				continue
			}
		}
		out = append(out, e)
	}
	setEdge := func(source, target string) *schema.Edge {
		return &schema.Edge{
			Source: source, Target: target, Kind: goextract.RefKindContains, Provenance: "ast",
			Metadata: map[string]string{"synthesizedBy": SynthesizedBySet},
		}
	}
	for _, n := range rs.setNodes() {
		set := rs.sets[n.Id]
		out = append(out, setEdge(set.ownerID, n.Id))
		for _, m := range set.members {
			out = append(out, setEdge(n.Id, m.id))
		}
	}
	ids := make([]string, 0, len(rs.storedSets))
	for id := range rs.storedSets {
		if _, live := rs.sets[id]; !live {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		stored := rs.storedSets[id]
		for _, m := range stored.members {
			if _, exists := rs.nodeByID[m]; !exists {
				continue
			}
			if _, inSet := setOfMember[stored.owner+"|"+m]; inSet {
				continue
			}
			out = append(out, &schema.Edge{Source: stored.owner, Target: m, Kind: goextract.RefKindContains, Provenance: "ast"})
		}
	}
	return out
}
