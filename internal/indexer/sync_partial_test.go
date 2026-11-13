package indexer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/graphstore/storetest"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/invariant"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// edgesFrom returns the "kind target-name" of each edge from the node of
// the given kind and name in the store at storeDir.
func edgesFrom(t *testing.T, r graphstore.Reader, kind, name string) []string {
	t.Helper()
	var out []string
	for _, n := range storeNodesIn(t, r, kind, name) {
		it, err := r.IterateEdges(n.Id)
		if err != nil {
			t.Fatalf("IterateEdges: %v", err)
		}
		for it.Next() {
			e := it.Edge()
			if e.Source != n.Id {
				continue
			}
			target := e.Target
			if tn, err := r.GetNode(e.Target); err == nil {
				target = tn.Name
			}
			out = append(out, e.Kind+" "+target)
		}
		it.Close()
	}
	return out
}

func has(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// TestSyncPartialTypeMatchesIndex changes a tree after a full index and
// syncs. In each row the store after the sync must equal the store of a
// full index over the changed tree: a sync merges the copies of a node id
// that several files emit as a full index does, and removes what a part
// that is gone leaves behind.
func TestSyncPartialTypeMatchesIndex(t *testing.T) {
	const ns = "namespace Shop;\n\n"
	iface := func(name, body string) string { return ns + "public interface " + name + "\n{\n" + body + "}\n" }
	part := func(modifier, base string) string {
		return ns + modifier + "partial class Order : " + base + "\n{\n}\n"
	}
	base := map[string]string{
		"Shop.csproj": "<Project Sdk=\"Microsoft.NET.Sdk\">\n  <PropertyGroup>\n    <RootNamespace>Shop</RootNamespace>\n  </PropertyGroup>\n</Project>\n",
		"IA.cs":       iface("IA", ""),
		"IB.cs":       iface("IB", ""),
	}
	with := func(extra map[string]string) map[string]string {
		out := map[string]string{}
		for k, v := range base {
			out[k] = v
		}
		for k, v := range extra {
			out[k] = v
		}
		return out
	}
	remove := func(name string) func(t *testing.T, root string) {
		return func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, name)); err != nil {
				t.Fatalf("remove %s: %v", name, err)
			}
		}
	}
	write := func(name, content string) func(t *testing.T, root string) {
		return func(t *testing.T, root string) { writeFile(t, filepath.Join(root, name), content) }
	}

	for _, tc := range []struct {
		name   string
		files  map[string]string
		before func(t *testing.T, r graphstore.Reader)
		change func(t *testing.T, root string)
		after  func(t *testing.T, r graphstore.Reader)
	}{
		{
			name:   "a part drops its public modifier",
			files:  with(map[string]string{"A.cs": part("", "IA"), "B.cs": part("public ", "IB")}),
			change: write("B.cs", part("", "IB")),
			after: func(t *testing.T, r graphstore.Reader) {
				if n := storeNodesIn(t, r, goextract.KindStruct, "Order"); len(n) != 1 || n[0].Visibility != "internal" || n[0].IsExported {
					t.Errorf("Order nodes = %v, want one internal node that is not exported", n)
				}
			},
		},
		{
			name:   "a part is deleted",
			files:  with(map[string]string{"A.cs": part("", "IA"), "B.cs": part("", "IB")}),
			change: remove("B.cs"),
			after: func(t *testing.T, r graphstore.Reader) {
				if got := edgesFrom(t, r, goextract.KindStruct, "Order"); has(got, "implements IB") || !has(got, "implements IA") {
					t.Errorf("edges from Order = %v, want the edge to IA and none to IB", got)
				}
			},
		},
		{
			name: "a part is deleted, and its base declares a method that another part overrides",
			files: with(map[string]string{
				"IB.cs": iface("IB", "    int Total();\n"),
				"A.cs":  ns + "partial class Order : IA\n{\n    public int Total()\n    {\n        return 0;\n    }\n}\n",
				"B.cs":  part("", "IB"),
			}),
			before: func(t *testing.T, r graphstore.Reader) {
				// Positive assertion: the override exists while B.cs does.
				if got := edgesFrom(t, r, goextract.KindMethod, "Total"); !has(got, "overrides Total") {
					t.Errorf("edges from the Total methods before the sync = %v, want an overrides edge", got)
				}
			},
			change: remove("B.cs"),
			after: func(t *testing.T, r graphstore.Reader) {
				if got := edgesFrom(t, r, goextract.KindMethod, "Total"); has(got, "overrides Total") {
					t.Errorf("edges from the Total methods = %v, want no overrides edge once Order has no base IB", got)
				}
			},
		},
		{
			name:   "a part is added",
			files:  with(map[string]string{"A.cs": part("public ", "IA")}),
			change: write("B.cs", part("", "IB")),
			after: func(t *testing.T, r graphstore.Reader) {
				if n := storeNodesIn(t, r, goextract.KindStruct, "Order"); len(n) != 1 || n[0].Visibility != "public" {
					t.Errorf("Order nodes = %v, want one public node", n)
				}
				if got := edgesFrom(t, r, goextract.KindStruct, "Order"); !has(got, "implements IA") || !has(got, "implements IB") {
					t.Errorf("edges from Order = %v, want the base edges of both parts", got)
				}
			},
		},
		{
			name:   "the only part is deleted",
			files:  with(map[string]string{"A.cs": part("", "IA")}),
			change: remove("A.cs"),
			after: func(t *testing.T, r graphstore.Reader) {
				if n := storeNodesIn(t, r, goextract.KindStruct, "Order"); len(n) != 0 {
					t.Errorf("the store holds %d Order nodes, want none", len(n))
				}
			},
		},
		{
			name:   "a part leaves the type",
			files:  with(map[string]string{"A.cs": part("", "IA"), "B.cs": part("public ", "IB")}),
			change: write("B.cs", ns+"public class Invoice : IB\n{\n}\n"),
			after: func(t *testing.T, r graphstore.Reader) {
				if n := storeNodesIn(t, r, goextract.KindStruct, "Order"); len(n) != 1 || n[0].Visibility != "internal" {
					t.Errorf("Order nodes = %v, want one internal node", n)
				}
				if got := edgesFrom(t, r, goextract.KindStruct, "Order"); has(got, "implements IB") {
					t.Errorf("edges from Order = %v, want none to IB", got)
				}
			},
		},
		{
			name:  "the only part is deleted, and another file names the type",
			files: with(map[string]string{"A.cs": part("", "IA"), "C.cs": ns + "class Sub : Order\n{\n}\n"}),
			before: func(t *testing.T, r graphstore.Reader) {
				// Positive assertion: the edge exists before the sync.
				if got := edgesFrom(t, r, goextract.KindStruct, "Sub"); !has(got, "extends Order") {
					t.Errorf("edges from Sub before the sync = %v, want an extends edge to Order", got)
				}
			},
			change: remove("A.cs"),
			after: func(t *testing.T, r graphstore.Reader) {
				for _, e := range edgesFrom(t, r, goextract.KindStruct, "Sub") {
					if strings.HasPrefix(e, "extends ") {
						t.Errorf("Sub still has the edge %q after its base type is gone", e)
					}
				}
			},
		},
		{
			name: "a changed Go file adds an interface that an unchanged struct satisfies",
			files: map[string]string{
				"s.go": "package p\n\ntype S struct{}\n\nfunc (S) Read() int { return 0 }\n",
				"i.go": "package p\n\nfunc Unused() {}\n",
			},
			change: write("i.go", "package p\n\ntype Reader interface {\n\tRead() int\n}\n"),
			after: func(t *testing.T, r graphstore.Reader) {
				if got := edgesFrom(t, r, goextract.KindStruct, "S"); !has(got, "implements Reader") {
					t.Errorf("edges from S = %v, want the implements edge to Reader", got)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := writeFixture(t, tc.files)
			storeDir := filepath.Join(t.TempDir(), "store")
			if _, err := Run(root, storeDir, Options{}); err != nil {
				t.Fatalf("Run: %v", err)
			}
			if tc.before != nil {
				r, closeFn := openSnapshot(t, storeDir)
				tc.before(t, r)
				closeFn()
			}

			tc.change(t, root)
			if _, err := Sync(root, storeDir, Options{}); err != nil {
				t.Fatalf("Sync: %v", err)
			}

			r, closeFn := openSnapshot(t, storeDir)
			tc.after(t, r)
			closeFn()
			synced := storetest.ExportNormalized(t, storeDir, true)
			if fresh := freshExport(t, root); !bytes.Equal(synced, fresh) {
				t.Errorf("the store after the sync differs from a full index of the changed tree:\n%s", exportDiff(t, fresh, synced))
			}
			// The metadata counts equal the stored records after the sync.
			ir, closeIR := openSnapshot(t, storeDir)
			if _, err := invariant.Check(ir); err != nil {
				t.Errorf("invariant.Check after the sync:\n%v", err)
			}
			closeIR()
		})
	}
}

// TestSyncCountsAnOverloadSetOnceWhenItsFileGoes pins the node count of a
// sync that removes an overload set which changed from one file to
// several. The set is stored first with the path of its one file, so that
// file's index names it. A later sync stores it with no path, and the
// entry in the file index stays. When the file goes, the prune of the file
// and the removal of sets that no file declares both reach the set. The
// sync must count the node once.
func TestSyncCountsAnOverloadSetOnceWhenItsFileGoes(t *testing.T) {
	part := func(iface string, methods ...string) string {
		var b strings.Builder
		fmt.Fprintf(&b, "namespace Prop;\n\npublic interface %s\n{\n}\n\npartial class Shared : %s\n{\n", iface, iface)
		for _, m := range methods {
			fmt.Fprintf(&b, "    public void %s()\n    {\n    }\n", m)
		}
		b.WriteString("}\n")
		return b.String()
	}
	root := writeFixture(t, map[string]string{
		"PartA.cs": part("IPartA", "alpha"),
		"PartB.cs": part("IPartB", "beta", "gamma"),
	})
	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}

	sets := func() int {
		r, closeFn := openSnapshot(t, storeDir)
		defer closeFn()
		return len(storeNodesIn(t, r, goextract.KindOverloadSet, "gamma"))
	}
	for _, step := range []struct {
		name     string
		change   func()
		wantSets int
	}{
		{"the set is in one file", func() { writeFile(t, filepath.Join(root, "PartB.cs"), part("IPartB", "gamma", "gamma")) }, 1},
		{"the set spans two files", func() { writeFile(t, filepath.Join(root, "PartA.cs"), part("IPartA", "alpha", "gamma")) }, 1},
		{"the first file of the set is deleted", func() {
			if err := os.Remove(filepath.Join(root, "PartB.cs")); err != nil {
				t.Fatalf("remove PartB.cs: %v", err)
			}
		}, 0},
	} {
		step.change()
		if _, err := Sync(root, storeDir, Options{}); err != nil {
			t.Fatalf("%s: Sync: %v", step.name, err)
		}
		// Positive assertion: the set exists before the step that removes
		// it, so the last step has a set to count.
		if got := sets(); got != step.wantSets {
			t.Fatalf("%s: the store holds %d overload sets named gamma, want %d", step.name, got, step.wantSets)
		}
		r, closeFn := openSnapshot(t, storeDir)
		_, err := invariant.Check(r)
		closeFn()
		if err != nil {
			t.Fatalf("%s: invariant.Check after the sync:\n%v", step.name, err)
		}
	}
}

// exportDiff lists the frames that only one of two exports holds.
func exportDiff(t *testing.T, want, got []byte) string {
	t.Helper()
	count := map[string]int{}
	for _, f := range decodeExportFrames(t, want) {
		count[fmt.Sprintf("kind %d %q", f.Kind, f.Payload)]++
	}
	var out bytes.Buffer
	for _, f := range decodeExportFrames(t, got) {
		k := fmt.Sprintf("kind %d %q", f.Kind, f.Payload)
		if count[k] > 0 {
			count[k]--
			continue
		}
		fmt.Fprintf(&out, "+ only after the sync: %.300s\n", k)
	}
	for k, n := range count {
		for ; n > 0; n-- {
			fmt.Fprintf(&out, "- only in the full index: %.300s\n", k)
		}
	}
	return out.String()
}

// TestSharedIDEmittersReadContains checks that the lookup of the files that
// emit a shared id reads only the contains entries of the reverse
// adjacency. A file that calls into the id names it and does not emit it.
func TestSharedIDEmittersReadContains(t *testing.T) {
	root := writeFixture(t, map[string]string{
		"A.cs": "namespace Shop;\n\npartial class Order\n{\n}\n",
		"C.cs": "namespace Shop;\n\nclass User\n{\n    void Run()\n    {\n    }\n}\n",
	})
	storeDir := filepath.Join(t.TempDir(), "store")
	if _, err := Run(root, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	r, closeFn := openSnapshot(t, storeDir)
	defer closeFn()
	orders := storeNodesIn(t, r, goextract.KindStruct, "Order")
	runs := storeNodesIn(t, r, goextract.KindMethod, "Run")
	if len(orders) != 1 || orders[0].FilePath != "" || len(runs) != 1 {
		t.Fatalf("the store holds the Order nodes %v and %d Run methods; want one Order node without a file and one method", orders, len(runs))
	}
	rev, err := buildReverseAdjacency(r, goextract.RefKindCalls, goextract.RefKindContains)
	if err != nil {
		t.Fatalf("buildReverseAdjacency: %v", err)
	}
	// One more entry for the shared id, of the other kind: a calls edge
	// from a method of C.cs.
	id := orders[0].Id
	rev[id] = append(rev[id], &schema.Edge{Source: runs[0].Id, Kind: goextract.RefKindCalls, Target: id})
	kinds := map[string]int{}
	for _, e := range rev[id] {
		kinds[e.Kind]++
	}
	if kinds[goextract.RefKindContains] != 1 || kinds[goextract.RefKindCalls] != 1 {
		t.Fatalf("the adjacency holds the entries %v for the shared id, want one contains and one calls entry", kinds)
	}

	got, err := sharedIDEmitters(r, rev, id)
	if err != nil {
		t.Fatalf("sharedIDEmitters: %v", err)
	}
	if len(got) != 1 || got[0] != "A.cs" {
		t.Errorf("sharedIDEmitters = %v, want only A.cs", got)
	}
}

// TestDependentsReadCallsOnly checks that only a caller of a pruned node is
// reparsed as its dependent. An extra reparse leaves the store as it is,
// so the equality tests cannot see it, and this test counts the files.
func TestDependentsReadCallsOnly(t *testing.T) {
	for _, tc := range []struct {
		name         string
		files        map[string]string
		edit, source string
		want         int
	}{
		{
			// The contains edge from T into M crosses the two files. It
			// adds no dependent when the sync prunes M.
			name: "a contains edge into the pruned node",
			files: map[string]string{
				"a.go": "package p\n\ntype T struct{}\n",
				"b.go": "package p\n\nfunc (T) M() {}\n",
			},
			edit: "b.go", source: "package p\n",
			want: 1,
		},
		{
			// The positive control: a calls edge into the pruned node
			// does add the file of the caller.
			name: "a calls edge into the pruned node",
			files: map[string]string{
				"a.go": "package p\n\nfunc F() {}\n",
				"b.go": "package p\n\nfunc G() { F() }\n",
			},
			edit: "a.go", source: "package p\n",
			want: 2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := writeFixture(t, tc.files)
			storeDir := filepath.Join(t.TempDir(), "store")
			if _, err := Run(root, storeDir, Options{}); err != nil {
				t.Fatalf("Run: %v", err)
			}
			writeFile(t, filepath.Join(root, tc.edit), tc.source)
			stats, err := Sync(root, storeDir, Options{})
			if err != nil {
				t.Fatalf("Sync: %v", err)
			}
			if stats.FilesReparsed != tc.want {
				t.Errorf("Sync reparsed %d files, want %d", stats.FilesReparsed, tc.want)
			}
		})
	}
}
