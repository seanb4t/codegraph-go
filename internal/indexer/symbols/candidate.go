package symbols

import (
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// Candidate is one declaration that a name can resolve to.
type Candidate struct {
	ID, Kind, QualifiedName, FilePath, Key string
	// Node is the declaration itself: its parameters and return type
	// drive the arity and type filters and the deferred-type fill.
	Node *schema.Node
}

// CandidateOf returns the candidate for the declaration n.
func CandidateOf(n *schema.Node) Candidate {
	return Candidate{ID: n.Id, Kind: n.Kind, QualifiedName: n.QualifiedName, FilePath: n.FilePath, Key: goextract.SignatureKeyOfNode(n), Node: n}
}

// IsTypeKind reports whether kind is a struct, interface or type alias.
func IsTypeKind(kind string) bool {
	switch kind {
	case goextract.KindStruct, goextract.KindInterface, goextract.KindTypeAlias:
		return true
	}
	return false
}

// Reachable reports whether another file can refer to n (design D15): a
// nested declaration never, and in TypeScript only an exported one. A
// JavaScript file that uses ESM exports follows the TypeScript rule; a
// file without any, a CommonJS module or a script, keeps every
// module-level declaration reachable, since the extractor does not read
// `module.exports`. Every other language reaches each module-level
// declaration through its module key.
func Reachable(n *schema.Node, fileExports bool) bool {
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

// Pick chooses the id that a name resolves to (WR-01, design D3), as a
// pure function of the candidate set. First, in each group of
// candidates with one kind, qualified name, file and signature key, the
// copies of a repeated declaration, it keeps the copy whose id is the
// DeclID of the group: the first copy in source order, when the
// extractor keyed it by its parameters alone. A copy under a
// conditional-compilation block has a key the group cannot see
// (goextract.ConditionalBlockKey), so a group without such a candidate
// keeps its lowest id. Second, across the groups, it keeps the lowest
// id, mirroring query.Engine.resolveSymbolNode's own lowest-Id
// tie-break (internal/query/traverse.go).
func Pick(cands []Candidate) string {
	if len(cands) == 1 {
		return cands[0].ID
	}
	// groupKey names the copies of one declaration.
	type groupKey struct{ kind, qualifiedName, filePath, key string }
	best := make(map[groupKey]string, len(cands))
	for _, c := range cands {
		group := groupKey{c.Kind, c.QualifiedName, c.FilePath, c.Key}
		first := nodeid.DeclID(c.Kind, c.QualifiedName, c.FilePath, c.Key)
		cur, ok := best[group]
		switch {
		case !ok:
			best[group] = c.ID
		case cur == first:
		case c.ID == first || c.ID < cur:
			best[group] = c.ID
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
