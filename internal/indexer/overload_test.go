package indexer

import (
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

const overloadfixtureRoot = "testdata/overloadfixture"

// overloadGraph is the indexed overloadfixture, read once for every
// subtest of TestOverloadsResolveAsTheLanguageDoes.
type overloadGraph struct {
	nodes []*schema.Node
	edges []*schema.Edge
}

func loadOverloadGraph(t *testing.T) *overloadGraph {
	t.Helper()
	storeDir := t.TempDir()
	if _, err := Run(overloadfixtureRoot, storeDir, Options{}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	requireInvariants(t, storeDir)
	return readGraph(t, storeDir)
}

// readGraph reads every node and edge of the store at storeDir.
func readGraph(t *testing.T, storeDir string) *overloadGraph {
	t.Helper()
	r, closeFn := openSnapshot(t, storeDir)
	defer closeFn()
	g := &overloadGraph{}
	nit, err := r.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	for nit.Next() {
		g.nodes = append(g.nodes, nit.Node())
	}
	nit.Close()
	eit, err := r.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	for eit.Next() {
		g.edges = append(g.edges, eit.Edge())
	}
	eit.Close()
	if len(g.nodes) == 0 || len(g.edges) == 0 {
		t.Fatalf("overloadfixture indexed to %d nodes and %d edges", len(g.nodes), len(g.edges))
	}
	return g
}

// node returns the one node in file with kind and name, and with signature
// when signature is not empty.
func (g *overloadGraph) node(t *testing.T, file, kind, name, signature string) *schema.Node {
	t.Helper()
	var found []*schema.Node
	for _, n := range g.nodes {
		if n.FilePath == file && n.Kind == kind && n.Name == name && (signature == "" || n.Signature == signature) {
			found = append(found, n)
		}
	}
	if len(found) != 1 {
		var sigs []string
		for _, n := range found {
			sigs = append(sigs, n.Signature)
		}
		t.Fatalf("%s: got %d nodes for %s %s %q, want 1 (signatures %q)", file, len(found), kind, name, signature, sigs)
	}
	return found[0]
}

// callTargets returns the targets of src's calls edges, with the ambiguous
// metadata of each.
func (g *overloadGraph) callTargets(src *schema.Node) map[string]bool {
	out := make(map[string]bool)
	for _, e := range g.edges {
		if e.Source == src.Id && e.Kind == goextract.RefKindCalls {
			out[e.Target] = e.Metadata["ambiguous"] == "true"
		}
	}
	return out
}

// requireOnlyCall fails unless src's only calls edge targets dst.
func (g *overloadGraph) requireOnlyCall(t *testing.T, src, dst *schema.Node) {
	t.Helper()
	targets := g.callTargets(src)
	if _, ok := targets[dst.Id]; !ok {
		t.Errorf("%s has no calls edge to %s %q at line %d", src.Name, dst.Kind, dst.Signature, dst.StartLine)
	}
	for id := range targets {
		if id != dst.Id {
			t.Errorf("%s also calls %s", src.Name, g.describe(id))
		}
	}
}

func (g *overloadGraph) describe(id string) string {
	for _, n := range g.nodes {
		if n.Id == id {
			return n.Kind + " " + n.QualifiedName + n.Signature + " in " + n.FilePath
		}
	}
	return id
}

// TestOverloadsResolveAsTheLanguageDoes has one subtest per resolution
// scenario of the graph-index spec: a call reaches the overload the
// language selects, the set when static analysis cannot decide, and
// nothing when no overload fits or the receiver is unknown.
func TestOverloadsResolveAsTheLanguageDoes(t *testing.T) {
	g := loadOverloadGraph(t)
	app := "java/com/example/App.java"
	logger := "java/com/example/Logger.java"
	sink := "java/com/example/Sink.java"
	builder := "java/com/example/Builder.java"
	method := goextract.KindMethod

	t.Run("resolution by arity", func(t *testing.T) {
		g.requireOnlyCall(t, g.node(t, app, method, "byArity", ""), g.node(t, logger, method, "log", "(String msg, Throwable t)"))
	})
	t.Run("resolution by argument type", func(t *testing.T) {
		g.requireOnlyCall(t, g.node(t, app, method, "byType", ""), g.node(t, sink, method, "add", "(String s)"))
	})
	t.Run("a variadic parameter", func(t *testing.T) {
		g.requireOnlyCall(t, g.node(t, app, method, "byVariadic", ""), g.node(t, logger, method, "format", "(String fmt, Object... args)"))
	})
	t.Run("an argument whose type is unknown", func(t *testing.T) {
		src := g.node(t, app, method, "unknownArg", "")
		set := g.node(t, sink, goextract.KindOverloadSet, "add", "")
		targets := g.callTargets(src)
		ambiguous, ok := targets[set.Id]
		if !ok || !ambiguous {
			t.Errorf("unknownArg: calls = %v, want one ambiguous edge to the add set", targets)
		}
		for id := range targets {
			if id != set.Id {
				t.Errorf("unknownArg also calls %s", g.describe(id))
			}
		}
	})
	t.Run("no overload fits", func(t *testing.T) {
		if targets := g.callTargets(g.node(t, app, method, "noFit", "")); len(targets) != 0 {
			t.Errorf("noFit calls %v, want no call edge", targets)
		}
	})
	t.Run("a call through a field", func(t *testing.T) {
		g.requireOnlyCall(t, g.node(t, app, method, "throughField", ""), g.node(t, logger, method, "log", "(String msg)"))
	})
	t.Run("a call through a parameter", func(t *testing.T) {
		cs := "csharp/Logging.cs"
		g.requireOnlyCall(t, g.node(t, cs, method, "Emit", ""), g.node(t, cs, method, "Write", "(LogEvent evt)"))
	})
	t.Run("a lowercase Go package", func(t *testing.T) {
		g.requireOnlyCallAmong(t, g.node(t, "main.go", goextract.KindFunction, "main", ""), g.node(t, "util/util.go", goextract.KindFunction, "Parse", ""))
	})
	t.Run("a Go local receiver", func(t *testing.T) {
		g.requireOnlyCallAmong(t, g.node(t, "main.go", goextract.KindFunction, "main", ""), g.node(t, "main.go", method, "Run", ""))
	})
	t.Run("a lowercase Java type", func(t *testing.T) {
		g.requireOnlyCall(t, g.node(t, app, method, "lowercaseType", ""), g.node(t, "java/com/example/helper.java", method, "assist", ""))
	})
	t.Run("an unknown receiver", func(t *testing.T) {
		if targets := g.callTargets(g.node(t, app, method, "fromUnresolved", "")); len(targets) != 0 {
			t.Errorf("fromUnresolved calls %v, want no call edge", targets)
		}
	})
	t.Run("a local initialized with new", func(t *testing.T) {
		g.requireOnlyCall(t, g.node(t, app, method, "fromNew", ""), g.node(t, builder, method, "add", "(String s)"))
	})
	t.Run("a local initialized from a call into another file", func(t *testing.T) {
		g.requireOnlyCallNamed(t, g.node(t, app, method, "fromOtherFile", ""), g.node(t, builder, method, "add", "(String s)"))
	})
	t.Run("alternatives under #if", func(t *testing.T) {
		cs := "csharp/Logging.cs"
		src := g.node(t, cs, method, "Run", "")
		set := g.node(t, cs, goextract.KindOverloadSet, "Process", "")
		targets := g.callTargets(src)
		if ambiguous, ok := targets[set.Id]; !ok || !ambiguous {
			t.Errorf("Run: calls = %v, want one ambiguous edge to the Process set", targets)
		}
	})
}

// requireOnlyCallAmong fails unless src has a calls edge to dst. Other
// targets are allowed: main calls two functions.
// requireOnlyCallNamed fails unless, among src's calls to anything named
// like dst, the only target is dst. The body's other calls do not count.
func (g *overloadGraph) requireOnlyCallNamed(t *testing.T, src, dst *schema.Node) {
	t.Helper()
	targets := g.callTargets(src)
	if _, ok := targets[dst.Id]; !ok {
		t.Errorf("%s has no calls edge to %s", src.Name, g.describe(dst.Id))
	}
	for id := range targets {
		if id == dst.Id {
			continue
		}
		for _, n := range g.nodes {
			if n.Id == id && n.Name == dst.Name {
				t.Errorf("%s also calls %s", src.Name, g.describe(id))
			}
		}
	}
}

func (g *overloadGraph) requireOnlyCallAmong(t *testing.T, src, dst *schema.Node) {
	t.Helper()
	if _, ok := g.callTargets(src)[dst.Id]; !ok {
		t.Errorf("%s has no calls edge to %s", src.Name, g.describe(dst.Id))
	}
}
