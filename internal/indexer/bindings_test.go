package indexer

import (
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/indexer/resolve"
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
		"chain/chain.go": `package chain

import "example.com/m/daemon"

var eight1 = eight2

var eight2 = eight3

var eight3 = eight4

var eight4 = eight5

var eight5 = eight6

var eight6 = eight7

var eight7 = eight8

var eight8 = daemon.List

var nine1 = nine2

var nine2 = nine3

var nine3 = nine4

var nine4 = nine5

var nine5 = nine6

var nine6 = nine7

var nine7 = nine8

var nine8 = nine9

var nine9 = daemon.List

func callEight() {
	eight1()
}

func callNine() {
	nine1()
}
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
		// Each binding path names the context that establishes it.
		var pathContexts []string
		for _, p := range strings.Split(e.Metadata["via"], ",") {
			if _, ctx, ok := strings.Cut(p, "@"); ok {
				pathContexts = append(pathContexts, ctx)
			}
		}
		if got := strings.Join(pathContexts, ","); got != wantCtx {
			t.Errorf("%s -> List binding paths %q name the contexts %q, want %q", fn, e.Metadata["via"], got, wantCtx)
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
	// A binding path goes through at most eight data symbols.
	check("chain/chain.go", goextract.KindFunction, "callEight", "go-production,go-test")
	check("chain/chain.go", goextract.KindFunction, "callNine", "")
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
	if b == nil || b.Metadata["direct"] != "true" || !strings.Contains(b.Metadata["via"], "@"+resolve.ContextGoProduction) {
		t.Errorf("both -> List = %v, want one edge with direct evidence and a binding path with its context", b)
	}
	if b != nil && b.Target != list {
		t.Errorf("both -> List targets %s, want %s", b.Target, list)
	}
}

// TestBindingsWithheldBySyntaxErrors covers the syntax-error rule in
// TypeScript, JavaScript and Python. A call through a module variable
// that holds a function reaches the function in a clean file, and
// reaches nothing in a file with a syntax error.
func TestBindingsWithheldBySyntaxErrors(t *testing.T) {
	const tsCall = "import { run } from \"./run\";\n\nconst handler = run;\n\nfunction call() {\n  handler();\n}\n"
	const tsDamage = "\nfunction damage() {\n  let x = (;\n}\n"
	const pyCall = "from lib import run\n\nhandler = run\n\n\ndef call():\n    handler()\n"
	const pyDamage = "\n\ndef damage():\n    x = )\n"
	r, _ := runAndSnapshot(t, testtree.Write(t, map[string]string{
		"run.ts":     "export function run() { return 0; }\n",
		"clean.ts":   tsCall,
		"broken.ts":  tsCall + tsDamage,
		"run.js":     "export function run() { return 0; }\n",
		"clean.js":   tsCall,
		"broken.js":  tsCall + tsDamage,
		"lib.py":     "def run():\n    return 0\n",
		"clean.py":   pyCall,
		"broken.py":  pyCall + pyDamage,
		"target.py":  pyCall,
		"patcher.py": "import target\nfrom lib import run\n\n\ndef patch():\n    target.handler = run\n" + pyDamage,
	}))
	// A write from a file with a syntax error withholds the call, even
	// when it assigns the same function.
	if writes := dataEdges(t, r, "patcher.py", goextract.KindFunction, "patch", goextract.RefKindReferences); writes["variable handler"] != goextract.AccessWrite {
		t.Errorf("patch references %v, want a write of target.handler", writes)
	}
	if calls := dataEdges(t, r, "target.py", goextract.KindFunction, "call", goextract.RefKindCalls); len(calls) != 0 {
		t.Errorf("target.py: call calls %v, want none: patcher.py writes handler and has a syntax error", calls)
	}
	for _, c := range []struct {
		path, kind string
		want       bool
	}{
		{"clean.ts", goextract.KindConstant, true},
		{"broken.ts", goextract.KindConstant, false},
		{"clean.js", goextract.KindConstant, true},
		{"broken.js", goextract.KindConstant, false},
		{"clean.py", goextract.KindVariable, true},
		{"broken.py", goextract.KindVariable, false},
	} {
		calls := dataEdges(t, r, c.path, goextract.KindFunction, "call", goextract.RefKindCalls)
		if _, ok := calls["function run"]; ok != c.want {
			t.Errorf("%s: call calls %v, want a call of run = %v", c.path, calls, c.want)
		}
		if got := dataEdges(t, r, c.path, goextract.KindFunction, "call", goextract.RefKindReferences)[c.kind+" handler"]; got != "read" {
			t.Errorf("%s: call -> handler access = %q, want read", c.path, got)
		}
		f, err := r.GetFile(c.path)
		if err != nil {
			t.Fatalf("GetFile(%s): %v", c.path, err)
		}
		if f.GetSyntaxErrors() == c.want {
			t.Errorf("%s: syntax_errors = %v, want %v", c.path, f.GetSyntaxErrors(), !c.want)
		}
	}
}
