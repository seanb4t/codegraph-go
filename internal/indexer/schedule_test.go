package indexer

import (
	"bytes"
	"context"
	"math/rand/v2"
	"path/filepath"
	"sync"
	"testing"

	"pgregory.net/rapid"
)

// scheduleTrees returns the trees of the schedule tests: fixtures with
// several languages, an overload set, a C# partial type, a call that only
// the retry resolves, and a route that one file registers twice.
func scheduleTrees(t *testing.T) []fixtureTree {
	t.Helper()
	twice := writeFixture(t, map[string]string{
		"package.json": "{\n  \"name\": \"twice\",\n  \"dependencies\": {\n    \"express\": \"^4.18.0\"\n  }\n}\n",
		"app.ts":       "import express from \"express\";\nconst app = express();\napp.get(\"/x\", first);\n\nfunction first() {}\nfunction second() {}\napp.get(\"/x\", second);\n",
		"other.ts":     "export function helper() {}\n",
	})
	return []fixtureTree{
		{"gofixture", "testdata/gofixture"},
		{"mixedlangfixture", "testdata/mixedlangfixture"},
		{"overloadfixture", "testdata/overloadfixture"},
		{"partialbasefixture", "testdata/partialbasefixture"},
		{"splitmethodfixture", "testdata/splitmethodfixture"},
		{"route registered twice", twice},
	}
}

// exportWith runs a full index of root with the given workers, and returns
// the export of its store with the last-sync time cleared.
func exportWith(t *testing.T, root string, workers int) []byte {
	t.Helper()
	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := run(context.Background(), root, storeDir, Options{Workers: workers}); err != nil {
		t.Fatalf("run: %v", err)
	}
	return exportNormalized(t, storeDir, false)
}

// TestIndexScheduleIndependence checks that the store of a full index does
// not depend on its schedule: the number of workers, the order in which
// they take files, and where the batches end. Each drawn run must store
// the bytes of a run with one worker, path order and the default cap.
func TestIndexScheduleIndependence(t *testing.T) {
	trees := scheduleTrees(t)
	reference := make([][]byte, len(trees))
	for i, tr := range trees {
		reference[i] = exportWith(t, tr.root, 1)
	}

	rapid.Check(t, func(rt *rapid.T) {
		i := rapid.IntRange(0, len(trees)-1).Draw(rt, "tree")
		workers := rapid.IntRange(1, 8).Draw(rt, "workers")
		seed := rapid.Uint64().Draw(rt, "claimOrder")
		// One permutation for each loop length, from the drawn seed: the
		// steps of a run have different numbers of positions.
		var mu sync.Mutex
		perms := map[int][]int{}
		origOrder, origCap := claimOrder, batchCap
		claimOrder = func(counter, n int) int {
			mu.Lock()
			defer mu.Unlock()
			if perms[n] == nil {
				perms[n] = rand.New(rand.NewPCG(seed, uint64(n))).Perm(n)
			}
			return perms[n][counter]
		}
		batchCap = rapid.IntRange(4<<10, 1<<20).Draw(rt, "batchCap")
		defer func() { claimOrder, batchCap = origOrder, origCap }()

		if got := exportWith(t, trees[i].root, workers); !bytes.Equal(got, reference[i]) {
			rt.Fatalf("%s: the store of this schedule differs from the reference:\n%s", trees[i].name, exportDiff(t, reference[i], got))
		}
	})
}

// TestClaimOrderReverse is the claim-order scenario of INDX-02: four
// workers that take the files in path order and in reverse path order
// store the same bytes.
func TestClaimOrderReverse(t *testing.T) {
	for _, tr := range scheduleTrees(t) {
		t.Run(tr.name, func(t *testing.T) {
			forward := exportWith(t, tr.root, 4)
			setHook(t, &claimOrder, func(counter, n int) int { return n - 1 - counter })
			if reverse := exportWith(t, tr.root, 4); !bytes.Equal(forward, reverse) {
				t.Errorf("the store of the reverse claim order differs:\n%s", exportDiff(t, forward, reverse))
			}
		})
	}
}
