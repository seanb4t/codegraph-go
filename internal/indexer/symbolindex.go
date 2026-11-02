package indexer

import (
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// symbolIndex is the global (moduleKey, declaredName) -> nodeID index Pass
// 2 builds from every Pass-1 result before resolving any cross-file
// reference — the "global symbol index" RES-01/D-04 call for. moduleKey is
// a per-language cross-file symbol-index key (Phase 5 D-04/Pitfall 2): Go's
// import path is the first instance of this concept (still carried on
// goextract.FileResult.ImportPath — the field name is unchanged, only its
// semantic source generalizes per language via LanguageSpec.ModuleKey).
type symbolIndex struct {
	// byModuleKeyAndName is keyed moduleKey -> declaredName -> nodeID.
	// declaredName is a node's bare Name field: for functions, types,
	// constants, and variables this is the plain identifier; for a method
	// it is the bare method name (NOT "Recv.Method" — that lives in
	// QualifiedName), matching how goextract records call-site references
	// (UnresolvedRef.Name is always the bare identifier text at the call
	// site, never receiver-qualified). The id is the winner pickSymbol
	// chose from candidates.
	byModuleKeyAndName map[string]map[string]string

	// candidates holds, per moduleKey and declaredName, every distinct
	// symbol that add received, so that the winner is a function of the
	// set and not of the insertion order.
	candidates map[string]map[string][]symbolCandidate

	// membersByOwner holds, per type node id and member name, the methods
	// that a type contains (design D3): the candidate set of a call
	// through a receiver of that type.
	membersByOwner map[string]map[string][]symbolCandidate

	// Collisions counts every same-(moduleKey, name) collision addSymbol
	// resolved deterministically (WR-01) instead of silently
	// last-write-wins overwriting — surfaced here so a collision is never
	// silently dropped (D-06a), even though this plan's file scope does
	// not yet wire it through to Sync's Stats/CLI --verbose output.
	Collisions int
}

// symbolCandidate is one declaration that a name can resolve to.
type symbolCandidate struct {
	id, kind, qualifiedName, filePath, key string
	// node is the declaration itself: its parameters and return type
	// drive the arity and type filters and the deferred-type fill.
	node *schema.Node
}

func candidateOf(n *schema.Node) symbolCandidate {
	return symbolCandidate{id: n.Id, kind: n.Kind, qualifiedName: n.QualifiedName, filePath: n.FilePath, key: goextract.SignatureKeyOfNode(n), node: n}
}

func newEmptySymbolIndex() *symbolIndex {
	return &symbolIndex{
		byModuleKeyAndName: make(map[string]map[string]string),
		candidates:         make(map[string]map[string][]symbolCandidate),
		membersByOwner:     make(map[string]map[string][]symbolCandidate),
	}
}

// reachable reports whether another file can refer to n (design D15): a
// nested declaration never, and in TypeScript only an exported one. A
// JavaScript file that uses ESM exports follows the TypeScript rule; a
// file without any, a CommonJS module or a script, keeps every
// module-level declaration reachable, since the extractor does not read
// `module.exports`. Every other language reaches each module-level
// declaration through its module key.
func reachable(n *schema.Node, fileExports bool) bool {
	if n.Nested {
		return false
	}
	switch n.Language {
	case "typescript", "tsx":
		return n.IsExported
	case "javascript":
		return !fileExports || n.IsExported
	}
	return true
}

// addMember records that the type ownerID contains member, from a
// contains edge. Re-adding the same member is not a change.
func (idx *symbolIndex) addMember(ownerID string, member *schema.Node) {
	byName := idx.membersByOwner[ownerID]
	if byName == nil {
		byName = make(map[string][]symbolCandidate)
		idx.membersByOwner[ownerID] = byName
	}
	for _, c := range byName[member.Name] {
		if c.id == member.Id {
			return
		}
	}
	byName[member.Name] = append(byName[member.Name], candidateOf(member))
}

// members returns the candidates named name that ownerID contains.
func (idx *symbolIndex) members(ownerID, name string) []symbolCandidate {
	return idx.membersByOwner[ownerID][name]
}

// candidatesIn returns every candidate named name in moduleKey.
func (idx *symbolIndex) candidatesIn(moduleKey, name string) []symbolCandidate {
	return idx.candidates[moduleKey][name]
}

// typeIn returns the type named name in moduleKey: a struct, interface or
// type alias, chosen by pickSymbol when there are several.
func (idx *symbolIndex) typeIn(moduleKey, name string) (symbolCandidate, bool) {
	var types []symbolCandidate
	for _, c := range idx.candidates[moduleKey][name] {
		switch c.kind {
		case goextract.KindStruct, goextract.KindInterface, goextract.KindTypeAlias:
			types = append(types, c)
		}
	}
	if len(types) == 0 {
		return symbolCandidate{}, false
	}
	winner := pickSymbol(types)
	for _, c := range types {
		if c.id == winner {
			return c, true
		}
	}
	return types[0], true
}

// newSymbolIndex builds the index by ranging results in the order given.
// Pass 1's results are already index-addressed in a stable, pre-sorted file
// order (RESEARCH Pattern 2) — this function never ranges a Go map to
// decide insertion order, so the index itself introduces no
// nondeterminism. A skipped file (FileResult.Err != nil) contributes no
// symbols, matching goextract's own "no nodes for a skipped file"
// contract.
func newSymbolIndex(results []goextract.FileResult) *symbolIndex {
	idx := newEmptySymbolIndex()
	idx.overlay(results)
	return idx
}

// overlay adds results' own declared symbols into idx via add — the
// shared per-result loop both newSymbolIndex (fresh index) and Sync's
// store-seeded index (layering the reparse batch on top) use, so the two
// never drift (Phase 4 RESEARCH Pattern 1 step 10).
func (idx *symbolIndex) overlay(results []goextract.FileResult) {
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		byID := make(map[string]*schema.Node, len(r.Nodes))
		fileExports := false
		for _, en := range r.Nodes {
			byID[en.Node.Id] = en.Node
			fileExports = fileExports || en.Node.IsExported
		}
		for _, en := range r.Nodes {
			if en.Node.Kind == goextract.KindFile {
				// A file node is never a callable/embeddable symbol
				// target; excluding it also avoids a same-module
				// collision between a file's synthetic name (its
				// RelPath) and a genuinely declared symbol.
				continue
			}
			if !reachable(en.Node, fileExports) {
				continue
			}
			idx.add(r.ImportPath, en.Node)
		}
		for _, ie := range r.IntraEdges {
			if ie.Edge.Kind != goextract.RefKindContains {
				continue
			}
			owner, member := byID[ie.Edge.Source], byID[ie.Edge.Target]
			if owner == nil || member == nil || owner.Kind == goextract.KindFile {
				continue
			}
			idx.addMember(owner.Id, member)
		}
	}
}

// add files n under moduleKey and its bare name, and updates the winner
// for that name. Re-adding a candidate with the same id (e.g. re-overlay
// of an unchanged file during Sync) is not a collision. Every other
// candidate for a name that already has one increments idx.Collisions.
func (idx *symbolIndex) add(moduleKey string, n *schema.Node) {
	names := idx.byModuleKeyAndName[moduleKey]
	if names == nil {
		names = make(map[string]string)
		idx.byModuleKeyAndName[moduleKey] = names
		idx.candidates[moduleKey] = make(map[string][]symbolCandidate)
	}
	cands := idx.candidates[moduleKey][n.Name]
	for _, c := range cands {
		if c.id == n.Id {
			return
		}
	}
	if len(cands) > 0 {
		idx.Collisions++
	}
	cands = append(cands, candidateOf(n))
	idx.candidates[moduleKey][n.Name] = cands
	names[n.Name] = pickSymbol(cands)
}

// pickSymbol chooses the id that a name resolves to (WR-01, design D3 of
// the index-invariants change), as a pure function of the candidate set.
// First, in each group of candidates with one kind, qualified name and
// file and signature key — the copies of a repeated declaration — it
// keeps the first copy in source order, whose id is DeclID. A
// group without such a candidate keeps its lowest id. Second, across the
// groups, it keeps the lowest id, mirroring
// query.Engine.resolveSymbolNode's own lowest-Id tie-break
// (internal/query/traverse.go).
func pickSymbol(cands []symbolCandidate) string {
	if len(cands) == 1 {
		return cands[0].id
	}
	best := make(map[symbolCandidate]string, len(cands))
	for _, c := range cands {
		group := symbolCandidate{kind: c.kind, qualifiedName: c.qualifiedName, filePath: c.filePath, key: c.key}
		first := nodeid.DeclID(c.kind, c.qualifiedName, c.filePath, c.key)
		cur, ok := best[group]
		switch {
		case !ok:
			best[group] = c.id
		case cur == first:
		case c.id == first || c.id < cur:
			best[group] = c.id
		}
	}
	winner := ""
	for _, id := range best {
		if winner == "" || id < winner {
			winner = id
		}
	}
	return winner
}

// resolveUnqualified resolves an unqualified reference against the calling
// file's OWN moduleKey — any file sharing a module/package/namespace may
// reference any other file's module-level symbol, so the lookup is scoped
// by moduleKey, not by the specific file.
func (idx *symbolIndex) resolveUnqualified(callerModuleKey, name string) (string, bool) {
	names, ok := idx.byModuleKeyAndName[callerModuleKey]
	if !ok {
		return "", false
	}
	id, ok := names[name]
	return id, ok
}
