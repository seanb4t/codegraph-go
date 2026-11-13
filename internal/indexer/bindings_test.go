package indexer

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
	"github.com/seanb4t/codegraph-go/internal/testtree"
)

// bindingEdges returns the edges of kind from the node of kind srcKind and
// qualified name src in path, keyed by the target's kind and qualified
// name.
func bindingEdges(t *testing.T, r graphstore.Reader, path, srcKind, src, kind string) map[string]*schema.Edge {
	t.Helper()
	id := nodeID(t, r, path, srcKind, src)
	out := map[string]*schema.Edge{}
	it, err := r.IterateEdges(id)
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer it.Close()
	for it.Next() {
		e := it.Edge()
		if e.Kind != kind {
			continue
		}
		n, err := r.GetNode(e.Target)
		if err != nil {
			t.Fatalf("GetNode(%s): %v", e.Target, err)
		}
		out[n.Kind+" "+n.QualifiedName] = e
	}
	return out
}

// nodeID returns the id of the node of kind and qualified name in path.
func nodeID(t *testing.T, r graphstore.Reader, path, kind, qname string) string {
	t.Helper()
	it, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer it.Close()
	for it.Next() {
		n := it.Node()
		if n.FilePath == path && n.Kind == kind && n.QualifiedName == qname {
			return n.Id
		}
	}
	t.Fatalf("no %s %s in %s", kind, qname, path)
	return ""
}

// callsInto returns the sources of every calls edge into the node id.
func callsInto(t *testing.T, r graphstore.Reader, id string) []string {
	t.Helper()
	it, err := r.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer it.Close()
	var out []string
	for it.Next() {
		if e := it.Edge(); e.Kind == goextract.RefKindCalls && e.Target == id {
			out = append(out, e.Source)
		}
	}
	return out
}

// TestCallableBindingResolution covers calls through Go package
// variables that hold a function. A call through a variable is a read of
// the variable, and a call of the function only when the analysis
// context establishes one target: go-production leaves out the writes of
// _test.go files, and go-test counts every write. A competing write, an
// unknown value, a production reassignment, a cycle or a syntax error in
// the package withholds the target. An alias of an alias resolves through
// the chain, and an imported package variable resolves like a local one. A direct call and a call
// through a binding to the same function make one edge with both kinds of
// evidence.
func TestCallableBindingResolution(t *testing.T) {
	root := testtree.Write(t, map[string]string{
		"go.mod": "module example.com/m\n\ngo 1.22\n",
		"daemon/daemon.go": `package daemon

func List() []string { return nil }

func Other() []string { return nil }
`,
		"cli/cli.go": `package cli

import "example.com/m/daemon"

var daemonList = daemon.List

var stable = daemon.List

var alias = stable

var swapped = daemon.List

func init() {
	swapped = daemon.Other
}

func run() {
	daemonList()
}

func callStable() {
	stable()
}

func callAlias() {
	alias()
}

func callSwapped() {
	swapped()
}

func both() {
	daemon.List()
	stable()
}

var literal = func() []string { return nil }

func init() {
	literal = daemon.List
}

func callLiteral() {
	literal()
}

var loopA = loopB

var loopB = loopA

func callLoop() {
	loopA()
}

func callImported() {
	daemon.Handler()
}
`,
		"daemon/handler.go": `package daemon

var Handler = List
`,
		"broken/broken.go": `package broken

import "example.com/m/daemon"

var fn = daemon.List

func call() {
	fn()
}
`,
		"broken/damaged.go": `package broken

func damage() {
	fn = (
}
`,
		"cli/cli_test.go": `package cli

import "testing"

func TestRun(t *testing.T) {
	daemonList = func() []string { return nil }
	run()
	daemonList()
}

func TestStable(t *testing.T) {
	stable()
}
`,
	})
	r, _ := runAndSnapshot(t, root)
	const cli, cliTest = "cli/cli.go", "cli/cli_test.go"
	list := nodeID(t, r, "daemon/daemon.go", goextract.KindFunction, "List")

	for _, v := range []string{"daemonList", "stable", "alias", "swapped", "literal", "loopA"} {
		if got := callsInto(t, r, nodeID(t, r, cli, goextract.KindVariable, v)); len(got) != 0 {
			t.Errorf("calls edges target the variable %s from %v, want none", v, got)
		}
	}

	check := func(path, kind, fn, wantCtx string) {
		t.Helper()
		calls := bindingEdges(t, r, path, kind, fn, goextract.RefKindCalls)
		e := calls["function List"]
		if wantCtx == "" {
			if e != nil {
				t.Errorf("%s calls List with %v, want no call", fn, e.Metadata)
			}
			return
		}
		if e == nil {
			t.Errorf("%s does not call List; calls %v", fn, calls)
			return
		}
		if got := e.Metadata["binding_context"]; got != wantCtx {
			t.Errorf("%s -> List binding_context = %q, want %q", fn, got, wantCtx)
		}
		if e.Metadata["via"] == "" {
			t.Errorf("%s -> List has no binding path: %v", fn, e.Metadata)
		}
	}
	check(cli, goextract.KindFunction, "run", "go-production")
	check(cliTest, goextract.KindFunction, "TestRun", "")
	check(cli, goextract.KindFunction, "callStable", "go-production,go-test")
	check(cliTest, goextract.KindFunction, "TestStable", "go-test")
	check(cli, goextract.KindFunction, "callAlias", "go-production,go-test")
	check(cli, goextract.KindFunction, "callSwapped", "")
	check(cli, goextract.KindFunction, "callLiteral", "")
	check(cli, goextract.KindFunction, "callImported", "go-production,go-test")
	check("broken/broken.go", goextract.KindFunction, "call", "")
	if calls := bindingEdges(t, r, cli, goextract.KindFunction, "callLoop", goextract.RefKindCalls); len(calls) != 0 {
		t.Errorf("callLoop calls %v, want none: loopA and loopB form a cycle", calls)
	}
	if got := bindingEdges(t, r, cli, goextract.KindFunction, "callLoop", goextract.RefKindReferences)["variable loopA"]; got == nil {
		t.Error("callLoop keeps no read of loopA")
	}
	if calls := bindingEdges(t, r, cli, goextract.KindFunction, "callSwapped", goextract.RefKindCalls); len(calls) != 0 {
		t.Errorf("callSwapped calls %v, want none: production reassigns swapped", calls)
	}

	if got := bindingEdges(t, r, cli, goextract.KindFunction, "run", goextract.RefKindReferences)["variable daemonList"]; got == nil || got.Metadata["access"] != goextract.AccessRead {
		t.Errorf("run -> daemonList reference = %v, want a read", got)
	}
	if got := bindingEdges(t, r, cliTest, goextract.KindFunction, "TestRun", goextract.RefKindReferences)["variable daemonList"]; got == nil || got.Metadata["access"] != goextract.AccessReadWrite {
		t.Errorf("TestRun -> daemonList reference = %v, want read_write", got)
	}

	b := bindingEdges(t, r, cli, goextract.KindFunction, "both", goextract.RefKindCalls)["function List"]
	if b == nil || b.Metadata["direct"] != "true" || b.Metadata["via"] == "" {
		t.Errorf("both -> List = %v, want one edge with direct and binding evidence", b)
	}
	if b != nil && b.Target != list {
		t.Errorf("both -> List targets %s, want %s", b.Target, list)
	}
}
