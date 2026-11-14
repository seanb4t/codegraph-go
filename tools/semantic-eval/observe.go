package semanticeval

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
)

// Observed is what CodeGraph stored for one case's source tree.
type Observed struct {
	Declarations  []ObservedDeclaration
	Relationships []ObservedRelationship
	// Unindexed names source files without a file record, with the reason
	// CodeGraph recorded, or "no file record" when it recorded none.
	Unindexed map[string]string
	// Files maps the id of each file node to its path.
	Files map[string]string
}

// ObservedDeclaration is one stored node in semantic terms.
type ObservedDeclaration struct {
	NodeID string
	Path   string
	Owner  string
	Role   string
	Name   string
	Span   Span
}

// ObservedRelationship is one stored edge between two nodes.
type ObservedRelationship struct {
	Source string
	Kind   string
	Target string
}

// roleOf maps a stored node kind to an evaluation role. It returns "" for
// nodes that are not declarations, which are files, routes and overload sets.
func roleOf(kind string) string {
	switch kind {
	case "function", "method", "interface", "type_alias", "constant", "variable", "field", "enum":
		return kind
	case "struct", "class":
		return "type"
	}
	return ""
}

// Observe indexes dir into storeDir and reads the stored facts back.
func Observe(dir, storeDir string, sources []string) (*Observed, error) {
	if _, err := indexer.Run(dir, storeDir, indexer.Options{Quiet: true}); err != nil {
		return nil, fmt.Errorf("index %s: %w", dir, err)
	}
	return ReadStore(storeDir, sources)
}

// ReadStore reads the stored facts of an existing store.
func ReadStore(storeDir string, sources []string) (*Observed, error) {
	store, err := graphstore.Open(storeDir)
	if err != nil {
		return nil, err
	}
	defer store.Close()
	r, err := store.Snapshot()
	if err != nil {
		return nil, err
	}
	defer r.Close()

	type node struct{ kind, name, path string }
	nodes := map[string]node{}
	obs := &Observed{Unindexed: map[string]string{}}

	nit, err := r.IterateNodes()
	if err != nil {
		return nil, err
	}
	defer nit.Close()
	var decls []ObservedDeclaration
	for nit.Next() {
		n := nit.Node()
		nodes[n.Id] = node{n.Kind, n.Name, n.FilePath}
		if n.Kind == "file" {
			if obs.Files == nil {
				obs.Files = map[string]string{}
			}
			obs.Files[n.Id] = n.FilePath
		}
		if role := roleOf(n.Kind); role != "" {
			decls = append(decls, ObservedDeclaration{
				NodeID: n.Id, Path: n.FilePath, Role: role, Name: n.Name,
				Span: Span{
					Start: Position{Line: int(n.StartLine), Col: int(n.StartCol)},
					End:   Position{Line: int(n.EndLine), Col: int(n.EndCol)},
				},
			})
		}
	}
	if err := nit.Err(); err != nil {
		return nil, err
	}

	parent := map[string]string{}
	eit, err := r.IterateEdges("")
	if err != nil {
		return nil, err
	}
	defer eit.Close()
	for eit.Next() {
		e := eit.Edge()
		if e.Kind == "contains" {
			parent[e.Target] = e.Source
			continue
		}
		obs.Relationships = append(obs.Relationships, ObservedRelationship{Source: e.Source, Kind: e.Kind, Target: e.Target})
	}
	if err := eit.Err(); err != nil {
		return nil, err
	}

	for i := range decls {
		var owners []string
		for at := parent[decls[i].NodeID]; at != ""; at = parent[at] {
			n, ok := nodes[at]
			if !ok || roleOf(n.kind) == "" {
				break
			}
			owners = append([]string{n.name}, owners...)
		}
		decls[i].Owner = strings.Join(owners, ".")
	}
	sort.Slice(decls, func(i, j int) bool { return decls[i].NodeID < decls[j].NodeID })
	obs.Declarations = decls

	excluded := map[string]string{}
	xit, err := r.IterateExcludedFiles()
	if err != nil {
		return nil, err
	}
	defer xit.Close()
	for xit.Next() {
		x := xit.ExcludedFile()
		excluded[x.Path] = x.Reason.String()
	}
	if err := xit.Err(); err != nil {
		return nil, err
	}
	for _, s := range sources {
		if _, err := r.GetFile(s); err != nil {
			obs.Unindexed[s] = exclusionReason(s, excluded)
		}
	}
	return obs, nil
}

// exclusionReason finds the record of path or of its nearest excluded parent.
func exclusionReason(path string, excluded map[string]string) string {
	for at := path; at != "."; at = filepath.Dir(at) {
		if reason, ok := excluded[at]; ok {
			return reason
		}
	}
	return "no file record"
}
