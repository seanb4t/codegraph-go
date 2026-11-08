package depclosure

import (
	"maps"
	"slices"
	"testing"
)

// indexerLayers are the layer packages under internal/indexer. below names
// the layer each one must reach, or "" for none.
var indexerLayers = []struct {
	pkg, below string
}{
	{module + "/internal/indexer/discover", module + "/internal/indexer/extract"},
	{module + "/internal/indexer/extract", ""},
	{module + "/internal/indexer/resolve", module + "/internal/indexer/symbols"},
	{module + "/internal/indexer/symbols", ""},
}

// TestIndexerLayers holds the import direction of the indexer layers: a
// layer reaches the layer below it, and never the store or the
// internal/indexer root that orchestrates the layers. Go rejects an import
// cycle, so a layer cannot also reach the layer above it.
func TestIndexerLayers(t *testing.T) {
	prod := goListDeps(t, false, module+"/internal/indexer/...")
	recordInputs(t, prod)

	for _, l := range indexerLayers {
		p, ok := prod[l.pkg]
		if !ok {
			t.Errorf("go list did not list %s", l.pkg)
			continue
		}
		deps := depSet(p)
		if l.below != "" && !deps[l.below] {
			t.Errorf("%s does not depend on %s, the layer below it", l.pkg, l.below)
		}
		for _, dep := range slices.Sorted(maps.Keys(deps)) {
			if underPath(dep, module+"/internal/graphstore") || dep == module+"/internal/indexer" {
				t.Errorf("%s depends on %s (perhaps indirectly); a layer below internal/indexer must not reach the store or the internal/indexer root", l.pkg, dep)
			}
		}
	}
}
