package indexer

import (
	"slices"
	"strconv"
	"testing"

	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// TestSameFileMatchesFreshResolver looks up names in several files, in a
// random order, with one resolver. Each answer must equal the answer of a
// fresh resolver, so the index that sameFile keeps for one file never
// answers a lookup in another file.
func TestSameFileMatchesFreshResolver(t *testing.T) {
	names := []string{"a", "b", "c"}
	var switches, found int
	rapid.Check(t, func(t *rapid.T) {
		files := make([]goextract.FileResult, rapid.IntRange(2, 4).Draw(t, "files"))
		for i := range files {
			files[i].RelPath = "f" + strconv.Itoa(i) + ".go"
			for j := range rapid.IntRange(0, 6).Draw(t, "decls") {
				en := goextract.ExtractedNode{Node: &schema.Node{
					Id:       files[i].RelPath + "#" + strconv.Itoa(j),
					Kind:     goextract.KindFunction,
					Name:     rapid.SampledFrom(names).Draw(t, "name"),
					FilePath: files[i].RelPath,
					Nested:   rapid.Bool().Draw(t, "nested"),
				}}
				if en.Node.Nested {
					start := rapid.Int32Range(1, 10).Draw(t, "start")
					en.Scope = goextract.Range{StartLine: start, EndLine: start + rapid.Int32Range(0, 10).Draw(t, "length"), EndCol: 80}
				}
				files[i].Nodes = append(files[i].Nodes, en)
			}
		}

		shared := newResolver(newEmptySymbolIndex(), nil, nil, nil, nil)
		prev := -1
		for range rapid.IntRange(1, 30).Draw(t, "lookups") {
			i := rapid.IntRange(0, len(files)-1).Draw(t, "file")
			name := rapid.SampledFrom(names).Draw(t, "lookup")
			line := rapid.Int32Range(1, 25).Draw(t, "line")

			got := candidateIDs(shared.sameFile(files[i], name, line, 0, callableKind))
			fresh := newResolver(newEmptySymbolIndex(), nil, nil, nil, nil)
			want := candidateIDs(fresh.sameFile(files[i], name, line, 0, callableKind))
			if !slices.Equal(got, want) {
				t.Fatalf("lookup of %q at line %d in %s = %v, a fresh resolver gives %v", name, line, files[i].RelPath, got, want)
			}
			if prev >= 0 && prev != i {
				switches++
			}
			if len(got) > 0 {
				found++
			}
			prev = i
		}
	})
	if switches == 0 || found == 0 {
		t.Fatalf("the runs switched files %d times and found %d declarations, want both above zero", switches, found)
	}
}

func candidateIDs(cands []symbolCandidate) []string {
	ids := make([]string, len(cands))
	for i, c := range cands {
		ids[i] = c.id
	}
	return ids
}
