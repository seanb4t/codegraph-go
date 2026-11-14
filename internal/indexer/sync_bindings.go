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
func bindingDependents(idx *storeIndex, results []goextract.FileResult, paths []string, inBatch map[string]bool) ([]string, error) {
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
		f, err := idx.r0.GetFile(p)
		switch {
		case err == nil:
			if f.GetSyntaxErrors() {
				dirs[filepath.Dir(p)] = true
			}
		case !errors.Is(err, graphstore.ErrNotFound):
			return nil, err
		}
		it, err := idx.r0.IterateFileIndex(p)
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

	if err := idx.load(); err != nil {
		return nil, err
	}
	inPaths := make(map[string]bool, len(paths))
	for _, p := range paths {
		inPaths[p] = true
	}
	for _, n := range idx.data {
		if inPaths[n.path] || names[n.name] || dirs[filepath.Dir(n.path)] {
			seeds[n.id] = true
		}
	}
	for _, e := range idx.binding {
		if inPaths[idx.pathOf[resolve.EdgeOwnerID(e)]] {
			if e.Metadata[goextract.MetaBinding] == goextract.BindingInit {
				seeds[e.Source] = true
			} else {
				seeds[e.Target] = true
			}
		}
	}

	for grew := true; grew; {
		grew = false
		for _, e := range idx.binding {
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
		for _, p := range idx.readers[id] {
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
