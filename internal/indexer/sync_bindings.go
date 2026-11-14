package indexer

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/nodeid"
	"github.com/seanb4t/codegraph-go/internal/indexer/resolve"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// isDataID reports whether the node id names a data symbol.
func isDataID(id string) bool {
	return resolve.IsDataKind(nodeid.KindOf(id))
}

// bindingDependents returns the files outside inBatch that read a data
// symbol whose binding facts the reparse of paths can change, in path
// order. results are the new extraction results of those paths. A call
// through a data symbol is a read of it, so these files hold every call
// whose derived target can change.
//
// The affected symbols are the data symbols that paths declare, before
// and after; the symbols that their old binding edges write or initialize;
// the stored data symbols named by their new writes; and the data symbols
// of a directory in which one of the paths has or had a syntax error. The
// set then grows over the stored binding edges: a symbol initialized to
// or assigned an affected symbol is affected.
func bindingDependents(r0 graphstore.Reader, results []goextract.FileResult, paths []string, inBatch map[string]bool) ([]string, error) {
	seeds := make(map[string]bool)
	names := make(map[string]bool)
	dirs := make(map[string]bool)
	touched := false
	for _, r := range results {
		if r.SyntaxErrors {
			dirs[filepath.Dir(r.RelPath)] = true
		}
		for _, n := range r.Nodes {
			if isDataID(n.Node.Id) {
				seeds[n.Node.Id] = true
			}
		}
		for _, ref := range r.Unresolved {
			if goextract.IsWriteAccess(ref.Access) {
				names[ref.Name] = true
			}
		}
	}
	for _, p := range paths {
		f, err := r0.GetFile(p)
		switch {
		case err == nil:
			if f.GetSyntaxErrors() {
				dirs[filepath.Dir(p)] = true
			}
		case !errors.Is(err, graphstore.ErrNotFound):
			return nil, err
		}
		it, err := r0.IterateFileIndex(p)
		if err != nil {
			return nil, err
		}
		for it.Next() {
			e := it.Entry()
			if (e.IsNode && isDataID(e.NodeID)) || (!e.IsNode && e.Kind == goextract.RefKindReferences && isDataID(e.Target)) {
				touched = true
			}
		}
		err = it.Err()
		it.Close()
		if err != nil {
			return nil, err
		}
	}
	if !touched && len(seeds) == 0 && len(names) == 0 && len(dirs) == 0 {
		return nil, nil
	}

	inPaths := make(map[string]bool, len(paths))
	for _, p := range paths {
		inPaths[p] = true
	}
	pathOf := make(map[string]string)
	nit, err := r0.IterateNodes()
	if err != nil {
		return nil, err
	}
	for nit.Next() {
		n := nit.Node()
		pathOf[n.Id] = n.FilePath
		if !isDataID(n.Id) {
			continue
		}
		if inPaths[n.FilePath] || names[n.Name] || dirs[filepath.Dir(n.FilePath)] {
			seeds[n.Id] = true
		}
	}
	err = nit.Err()
	nit.Close()
	if err != nil {
		return nil, err
	}

	var binding []*schema.Edge
	readers := make(map[string][]string)
	eit, err := r0.IterateEdges("")
	if err != nil {
		return nil, err
	}
	for eit.Next() {
		e := eit.Edge()
		if e.Kind != goextract.RefKindReferences {
			continue
		}
		if resolve.IsBindingEdge(e) {
			binding = append(binding, e)
			if inPaths[pathOf[resolve.EdgeOwnerID(e)]] {
				if e.Metadata[goextract.MetaBinding] == goextract.BindingInit {
					seeds[e.Source] = true
				} else {
					seeds[e.Target] = true
				}
			}
		}
		if isDataID(e.Target) {
			readers[e.Target] = append(readers[e.Target], pathOf[e.Source])
		}
	}
	err = eit.Err()
	eit.Close()
	if err != nil {
		return nil, err
	}

	for grew := true; grew; {
		grew = false
		for _, e := range binding {
			from, to := "", ""
			if e.Metadata[goextract.MetaBinding] == goextract.BindingInit {
				from, to = e.Target, e.Source
			} else {
				to = e.Target
				for _, v := range strings.Split(e.Metadata[goextract.MetaBinds], ",") {
					if seeds[v] {
						from = v
					}
				}
			}
			if from != "" && seeds[from] && !seeds[to] {
				seeds[to] = true
				grew = true
			}
		}
	}

	set := make(map[string]bool)
	for id := range seeds {
		for _, p := range readers[id] {
			if p != "" && !inBatch[p] {
				set[p] = true
			}
		}
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}
