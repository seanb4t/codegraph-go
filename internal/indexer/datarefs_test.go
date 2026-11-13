package indexer

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
	"github.com/seanb4t/codegraph-go/internal/testtree"
)

// dataEdges returns, for the node of kind and qualified name src in path,
// each target of its edges of kind edgeKind, keyed by the target's kind
// and qualified name, with the edge's access metadata as the value.
func dataEdges(t *testing.T, r graphstore.Reader, path, kind, src, edgeKind string) map[string]string {
	t.Helper()
	var from *schema.Node
	it, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	for it.Next() {
		n := it.Node()
		if n.FilePath == path && n.Kind == kind && n.QualifiedName == src {
			from = n
		}
	}
	it.Close()
	if from == nil {
		t.Fatalf("no %s %s in %s", kind, src, path)
	}
	out := map[string]string{}
	eit, err := r.IterateEdges(from.Id)
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer eit.Close()
	for eit.Next() {
		e := eit.Edge()
		if e.Kind != edgeKind {
			continue
		}
		n, err := r.GetNode(e.Target)
		if err != nil {
			t.Fatalf("GetNode(%s): %v", e.Target, err)
		}
		out[n.Kind+" "+n.QualifiedName] = e.Metadata["access"]
	}
	return out
}

// TestGoDataReferences covers the Go references of the data-symbols
// contract: a package variable's initializer links the variable to the
// function value it holds without making it a caller, an initializer
// that invokes a function makes a call, a function literal's body is not
// invoked at initialization, a local that shadows a package variable is
// not a reference to it, and reads and writes of package variables and
// struct fields carry their access role, merged when one function does
// both.
func TestGoDataReferences(t *testing.T) {
	root := testtree.Write(t, map[string]string{
		"go.mod": "module example.com/m\n\ngo 1.22\n",
		"daemon/daemon.go": `package daemon

func List() []string { return nil }
`,
		"cli/cli.go": `package cli

import "example.com/m/daemon"

var daemonList = daemon.List

var initial = compute()

var deferred = func() int { return compute() }

var counter int

var x = 1

type Point struct {
	X, Y int
}

type Other struct {
	X int
}

func compute() int { return 1 }

func use(int) {}

func shadow() {
	x := 2
	use(x)
}

func reads() {
	use(x)
}

func set() {
	counter = 5
}

func bump() {
	counter++
}

func both() {
	use(counter)
	counter = 2
}

func (p *Point) Move() {
	p.X = p.Y
}

func run() {
	daemonList()
}
`,
	})
	r, _ := runAndSnapshot(t, root)
	const cli = "cli/cli.go"

	refs := dataEdges(t, r, cli, goextract.KindVariable, "daemonList", goextract.RefKindReferences)
	if _, ok := refs["function List"]; !ok {
		t.Errorf("daemonList references %v, want daemon.List", refs)
	}
	if calls := dataEdges(t, r, cli, goextract.KindVariable, "daemonList", goextract.RefKindCalls); len(calls) != 0 {
		t.Errorf("daemonList calls %v, want no call from an initializer that only names a function", calls)
	}
	if calls := dataEdges(t, r, cli, goextract.KindVariable, "initial", goextract.RefKindCalls); len(calls) != 1 {
		t.Errorf("initial calls %v, want the call to compute", calls)
	}
	if calls := dataEdges(t, r, cli, goextract.KindVariable, "deferred", goextract.RefKindCalls); len(calls) != 0 {
		t.Errorf("deferred calls %v, want none: a function literal does not run at initialization", calls)
	}

	shadow := dataEdges(t, r, cli, goextract.KindFunction, "shadow", goextract.RefKindReferences)
	if _, ok := shadow["variable x"]; ok {
		t.Errorf("shadow references the package variable x through a local: %v", shadow)
	}
	if got := dataEdges(t, r, cli, goextract.KindFunction, "reads", goextract.RefKindReferences)["variable x"]; got != "read" {
		t.Errorf("reads -> x access = %q, want read", got)
	}
	for fn, want := range map[string]string{"set": "write", "bump": "read_write", "both": "read_write"} {
		if got := dataEdges(t, r, cli, goextract.KindFunction, fn, goextract.RefKindReferences)["variable counter"]; got != want {
			t.Errorf("%s -> counter access = %q, want %q", fn, got, want)
		}
	}

	move := dataEdges(t, r, cli, goextract.KindMethod, "Point.Move", goextract.RefKindReferences)
	if move["field Point.X"] != "write" || move["field Point.Y"] != "read" {
		t.Errorf("Move references %v, want Point.X write and Point.Y read", move)
	}
	if _, ok := move["field Other.X"]; ok {
		t.Errorf("Move references Other.X, a field of another owner: %v", move)
	}
}
