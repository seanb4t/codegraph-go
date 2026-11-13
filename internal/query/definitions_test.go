package query

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"pgregory.net/rapid"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
	"github.com/seanb4t/codegraph-go/internal/testtree"
)

// defGraph builds a graph for the fake reader of traverse_test.go. A node
// id is the key the test gives, so a test controls which id sorts first.
type defGraph struct {
	nodes map[string]*schema.Node
	edges []*schema.Edge
}

func newDefGraph() *defGraph { return &defGraph{nodes: map[string]*schema.Node{}} }

func (g *defGraph) node(id, kind, name, file string, line int32) *schema.Node {
	n := &schema.Node{Id: id, Kind: kind, Name: name, QualifiedName: name, FilePath: file, StartLine: line}
	g.nodes[id] = n
	return n
}

func (g *defGraph) fn(id, name, file string, line int32) *schema.Node {
	return g.node(id, goextract.KindFunction, name, file, line)
}

// set adds an overload_set node over the members, at the place of the
// first member, as the indexer stores it.
func (g *defGraph) set(id string, members ...*schema.Node) {
	first := members[0]
	g.node(id, goextract.KindOverloadSet, first.Name, first.FilePath, first.StartLine)
	for _, m := range members {
		g.edges = append(g.edges, &schema.Edge{Source: id, Target: m.Id, Kind: goextract.RefKindContains})
	}
}

func (g *defGraph) call(src, dst string) {
	g.edges = append(g.edges, &schema.Edge{Source: src, Target: dst, Kind: goextract.RefKindCalls})
}

func (g *defGraph) engine() *Engine {
	return New(&traverseFakeReader{nodes: g.nodes, edges: g.edges})
}

func rowNames(locs []Location) []string {
	var out []string
	for _, l := range locs {
		out = append(out, l.Name)
	}
	sort.Strings(out)
	return out
}

func labels(locs []Location) []string {
	var out []string
	for _, l := range locs {
		out = append(out, l.Definition)
	}
	return out
}

// twoRuns is two functions named run in two files. The one in a.py calls
// load and is called by main. The one in b.py calls save. The ids put
// b.py first, so a pick by lowest id answers for b.py.
func twoRuns() *defGraph {
	g := newDefGraph()
	g.fn("z-run-a", "run", "a.py", 1)
	g.fn("a-run-b", "run", "b.py", 1)
	g.fn("load", "load", "a.py", 9)
	g.fn("save", "save", "b.py", 9)
	g.fn("main", "main", "m.py", 1)
	g.call("z-run-a", "load")
	g.call("a-run-b", "save")
	g.call("main", "z-run-a")
	return g
}

// TestCalleesOfANameWithTwoDefinitions is the scenario "Two functions of
// one name in two files".
func TestCalleesOfANameWithTwoDefinitions(t *testing.T) {
	got, err := twoRuns().engine().Callees("run", 0)
	if err != nil {
		t.Fatalf("Callees: %v", err)
	}
	if diff := cmp.Diff([]string{"load", "save"}, rowNames(got.Callees)); diff != "" {
		t.Errorf("callees of run (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"a.py:1", "b.py:1"}, labels(got.Definitions)); diff != "" {
		t.Errorf("definitions of run (-want +got):\n%s", diff)
	}
}

// treeEngine writes files with testtree.Write, indexes the tree and opens
// an Engine on it.
func treeEngine(t *testing.T, files map[string]string) *Engine {
	t.Helper()
	return indexedEngine(t, testtree.Write(t, files))
}

// TestCallersOfAStubAndTheRealFunction is the scenario "A typing stub and
// the real function", through the real indexer.
func TestCallersOfAStubAndTheRealFunction(t *testing.T) {
	engine := treeEngine(t, map[string]string{
		"lib.py": "from typing import overload\n\n\n@overload\ndef to_list(value: None) -> None: ...\n@overload\ndef to_list(value: int) -> list: ...\ndef to_list(value):\n    return value\n",
		"app.py": "from lib import to_list\n\n\ndef build(x):\n    return to_list(x)\n",
	})
	got, err := engine.Callers("to_list", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	if diff := cmp.Diff([]string{"build"}, rowNames(got.Callers)); diff != "" {
		t.Errorf("callers of to_list (-want +got):\n%s", diff)
	}
	if len(got.Definitions) != 3 {
		t.Errorf("definitions of to_list = %v, want the two stubs and the real function", labels(got.Definitions))
	}
}

// TestCalleesOfAMethodAndATestHelper is the scenario "A method and a test
// helper of one name", through the real indexer.
func TestCalleesOfAMethodAndATestHelper(t *testing.T) {
	engine := treeEngine(t, map[string]string{
		"src/models.py":     "def requote_uri(u):\n    return u\n\n\nclass Request:\n    def prepare_url(self, url):\n        return requote_uri(url)\n",
		"tests/conftest.py": "def prepare_url(value):\n    return value\n",
	})
	got, err := engine.Callees("prepare_url", 0)
	if err != nil {
		t.Fatalf("Callees: %v", err)
	}
	if diff := cmp.Diff([]string{"requote_uri"}, rowNames(got.Callees)); diff != "" {
		t.Errorf("callees of prepare_url (-want +got):\n%s", diff)
	}
}

// twoResets is two functions named Reset in two packages. A calls the
// first, B calls the second, and main calls both.
func twoResets() *defGraph {
	g := newDefGraph()
	g.fn("z-reset-a", "Reset", "a.go", 1)
	g.fn("a-reset-b", "Reset", "b.go", 1)
	g.fn("A", "A", "a.go", 5)
	g.fn("B", "B", "b.go", 5)
	g.fn("main", "main", "main.go", 3)
	g.call("A", "z-reset-a")
	g.call("B", "a-reset-b")
	g.call("main", "a-reset-b")
	g.call("main", "z-reset-a")
	return g
}

// TestImpactOverTwoDefinitions is the scenario "Impact over two
// definitions".
func TestImpactOverTwoDefinitions(t *testing.T) {
	got, err := twoResets().engine().Impact("Reset", 1)
	if err != nil {
		t.Fatalf("Impact: %v", err)
	}
	if diff := cmp.Diff([]string{"A", "B", "Reset", "Reset", "main"}, rowNames(got.Affected)); diff != "" {
		t.Errorf("affected by Reset (-want +got):\n%s", diff)
	}
}

// TestImpactReachesOneSymbolFromTwoDefinitions is the scenario of that
// name: each definition names itself, and main appears once with no
// definition.
func TestImpactReachesOneSymbolFromTwoDefinitions(t *testing.T) {
	got, err := twoResets().engine().Impact("Reset", 1)
	if err != nil {
		t.Fatalf("Impact: %v", err)
	}
	seen := map[string][]string{}
	for _, l := range got.Affected {
		seen[l.Name] = append(seen[l.Name], l.Definition)
	}
	if diff := cmp.Diff([]string{"a.go:1", "b.go:1"}, seen["Reset"]); diff != "" {
		t.Errorf("definition labels of the Reset rows (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{""}, seen["main"]); diff != "" {
		t.Errorf("definition labels of the main rows (-want +got):\n%s", diff)
	}

	// In the JSON, a Reset entry has the definition key and the main
	// entry has none.
	data, err := MarshalImpactJSON(got)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Affected []map[string]json.RawMessage `json:"affected"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal %s: %v", data, err)
	}
	hasKey := map[string][]bool{}
	for _, entry := range parsed.Affected {
		var name string
		if err := json.Unmarshal(entry["name"], &name); err != nil {
			t.Fatalf("name of %v: %v", entry, err)
		}
		_, ok := entry["definition"]
		hasKey[name] = append(hasKey[name], ok)
	}
	if diff := cmp.Diff([]bool{true, true}, hasKey["Reset"]); diff != "" {
		t.Errorf("definition key on the Reset entries (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]bool{false}, hasKey["main"]); diff != "" {
		t.Errorf("definition key on the main entry (-want +got):\n%s", diff)
	}
}

// TestOneCallerOfTwoDefinitionsIsOrderedByDefinition is the scenario "One
// caller of two definitions". The two rows of main tie on file, name and
// line, and the lower id belongs to b.go, so an order by id fails here.
func TestOneCallerOfTwoDefinitionsIsOrderedByDefinition(t *testing.T) {
	got, err := twoResets().engine().Callers("Reset", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	var mainRows []string
	for _, l := range got.Callers {
		if l.Name == "main" {
			mainRows = append(mainRows, l.Definition)
		}
	}
	if diff := cmp.Diff([]string{"a.go:1", "b.go:1"}, mainRows); diff != "" {
		t.Errorf("definitions of the two main rows, in order (-want +got):\n%s", diff)
	}
}

// TestCallerNamesTheDefinitionItCalls is the scenario of that name.
func TestCallerNamesTheDefinitionItCalls(t *testing.T) {
	got, err := twoRuns().engine().Callers("run", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	if len(got.Callers) != 1 || got.Callers[0].Name != "main" || got.Callers[0].Definition != "a.py:1" {
		t.Fatalf("callers of run = %+v, want main with the definition a.py:1", got.Callers)
	}
}

// javaLogs is Logger.java with log(String) and log(String, Throwable) in
// one set, and Audit.java with log(String) outside it. Main.start and
// Main.stop call the Logger overloads, Job.run calls the Audit one, and
// Amb.use calls the Logger set.
func javaLogs() *defGraph {
	g := newDefGraph()
	a := g.node("m-log-1", goextract.KindMethod, "log", "Logger.java", 4)
	a.Signature, a.QualifiedName = "(String msg)", "Logger.log"
	b := g.node("m-log-2", goextract.KindMethod, "log", "Logger.java", 6)
	b.Signature, b.QualifiedName = "(String msg, Throwable t)", "Logger.log"
	c := g.node("a-log-audit", goextract.KindMethod, "log", "Audit.java", 7)
	c.Signature, c.QualifiedName = "(String msg)", "Audit.log"
	for _, n := range []*schema.Node{a, c} {
		n.Params = []*schema.Param{{Name: "msg", Type: "String"}}
	}
	b.Params = []*schema.Param{{Name: "msg", Type: "String"}, {Name: "t", Type: "Throwable"}}
	g.set("set-logger", a, b)
	g.fn("start", "start", "Main.java", 3)
	g.fn("stop", "stop", "Main.java", 8)
	g.fn("run", "run", "Job.java", 3)
	g.fn("use", "use", "Amb.java", 3)
	g.call("start", "m-log-1")
	g.call("stop", "m-log-2")
	g.call("run", "a-log-audit")
	g.call("use", "set-logger")
	return g
}

// TestCallersOfASetAndADefinitionOutsideIt is the scenario "An overload
// set and a definition outside it".
func TestCallersOfASetAndADefinitionOutsideIt(t *testing.T) {
	got, err := javaLogs().engine().Callers("log", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	if diff := cmp.Diff([]string{"run", "start", "stop", "use"}, rowNames(got.Callers)); diff != "" {
		t.Errorf("callers of log (-want +got):\n%s", diff)
	}
}

// TestSignatureThatMatchesInTwoFilesSpansDefinitions is the scenario "A
// signature that matches in two files".
func TestSignatureThatMatchesInTwoFilesSpansDefinitions(t *testing.T) {
	got, err := javaLogs().engine().CallersSelected(Selector{Symbol: "log", Signature: "String"}, "", 0)
	if err != nil {
		t.Fatalf("CallersSelected: %v", err)
	}
	byName := map[string]string{}
	for _, l := range got.Callers {
		if !l.Ambiguous {
			byName[l.Name] = l.Definition
		}
	}
	want := map[string]string{"start": "Logger.java:4", "run": "Audit.java:7"}
	if diff := cmp.Diff(want, byName); diff != "" {
		t.Errorf("definition of each caller (-want +got):\n%s", diff)
	}
}

// TestAmbiguousCallerInASpanningQueryHasNoDefinition is the scenario "An
// ambiguous caller in a query that spans definitions".
func TestAmbiguousCallerInASpanningQueryHasNoDefinition(t *testing.T) {
	got, err := javaLogs().engine().Callers("log", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	found := false
	for _, l := range got.Callers {
		if l.Name == "use" {
			found = true
			if !l.Ambiguous || l.Definition != "" {
				t.Errorf("row of use = %+v, want ambiguous with no definition", l)
			}
		}
	}
	if !found {
		t.Fatalf("callers of log = %+v, want a row for use", got.Callers)
	}
}

// TestAmbiguousCalleeInASpanningQueryNamesItsDefinition is the scenario
// "An ambiguous callee in a query that spans definitions": the callee is
// an overload set, and the row names the definition that makes the call.
func TestAmbiguousCalleeInASpanningQueryNamesItsDefinition(t *testing.T) {
	g := newDefGraph()
	g.fn("z-run-job", "run", "Job.java", 3)
	g.fn("a-run-main", "run", "Main.java", 8)
	str := g.node("log-1", goextract.KindMethod, "log", "Logger.java", 4)
	str.Signature = "(String msg)"
	thr := g.node("log-2", goextract.KindMethod, "log", "Logger.java", 6)
	thr.Signature = "(Throwable t)"
	g.set("set-log", str, thr)
	g.call("z-run-job", "set-log")
	got, err := g.engine().Callees("run", 0)
	if err != nil {
		t.Fatalf("Callees: %v", err)
	}
	want := []Location{{Name: "log", Kind: goextract.KindOverloadSet, FilePath: "Logger.java", StartLine: 4, Ambiguous: true, Definition: "Job.java:3"}}
	if diff := cmp.Diff(want, got.Callees); diff != "" {
		t.Errorf("callees of run (-want +got):\n%s", diff)
	}
	data, err := MarshalCalleesJSON(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"ambiguous":true,"definition":"Job.java:3"`) {
		t.Errorf("JSON %s has no ambiguous entry with the definition Job.java:3", data)
	}
}

// TestDefinitionsWithNoFileOrderByQualifiedName pins the last key of the
// order of definitions: two packages of one name have no file and no
// line, and the ids sort opposite to the qualified names.
func TestDefinitionsWithNoFileOrderByQualifiedName(t *testing.T) {
	g := newDefGraph()
	a := g.node("z-pkg", "package", "cache", "", 0)
	a.QualifiedName = "example.com/a/cache"
	b := g.node("a-pkg", "package", "cache", "", 0)
	b.QualifiedName = "example.com/b/cache"
	got, err := g.engine().Callers("cache", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	if diff := cmp.Diff([]string{"package example.com/a/cache", "package example.com/b/cache"}, labels(got.Definitions)); diff != "" {
		t.Errorf("definitions of cache, in order (-want +got):\n%s", diff)
	}
}

// TestImpactWhenOneDefinitionCallsTheOther keeps each start definition
// once, with its own label, when one of them calls the other.
func TestImpactWhenOneDefinitionCallsTheOther(t *testing.T) {
	g := twoResets()
	g.call("z-reset-a", "a-reset-b")
	got, err := g.engine().Impact("Reset", 1)
	if err != nil {
		t.Fatalf("Impact: %v", err)
	}
	var resets []string
	for _, l := range got.Affected {
		if l.Name == "Reset" {
			resets = append(resets, l.Definition)
		}
	}
	if diff := cmp.Diff([]string{"a.go:1", "b.go:1"}, resets); diff != "" {
		t.Errorf("Reset rows of the impact (-want +got):\n%s", diff)
	}
}

// TestTiedResultsOfOneDefinition is the scenario of that name: two
// callers tie on file, name and line, and the kind orders them, not the
// ids, which sort the other way.
func TestTiedResultsOfOneDefinition(t *testing.T) {
	g := newDefGraph()
	g.fn("run", "run", "a.go", 1)
	g.node("a-x", goextract.KindMethod, "x", "c.go", 5)
	g.node("b-x", "function", "x", "c.go", 5)
	g.call("a-x", "run")
	g.call("b-x", "run")
	got, err := g.engine().Callers("run", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	var kinds []string
	for _, l := range got.Callers {
		kinds = append(kinds, l.Kind)
	}
	if diff := cmp.Diff([]string{"function", goextract.KindMethod}, kinds); diff != "" {
		t.Errorf("kinds of the callers of run, in order (-want +got):\n%s", diff)
	}
}

// TestTwoOverloadSetsSpanDefinitions pins the third case of design D2:
// two sets and no definition outside them.
func TestTwoOverloadSetsSpanDefinitions(t *testing.T) {
	g := newDefGraph()
	var sets [][]*schema.Node
	for i, file := range []string{"A.java", "B.java"} {
		one := g.node(fmt.Sprintf("m%d-1", i), goextract.KindMethod, "log", file, 4)
		one.Signature = "(String msg)"
		two := g.node(fmt.Sprintf("m%d-2", i), goextract.KindMethod, "log", file, 6)
		two.Signature = "(int n)"
		g.set(fmt.Sprintf("set%d", i), one, two)
		sets = append(sets, []*schema.Node{one, two})
	}
	g.fn("caller", "caller", "C.java", 2)
	g.call("caller", sets[1][0].Id)
	got, err := g.engine().Callers("log", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	if diff := cmp.Diff([]string{"A.java:4", "A.java:6", "B.java:4", "B.java:6"}, labels(got.Definitions)); diff != "" {
		t.Errorf("definitions of log (-want +got):\n%s", diff)
	}
	if len(got.Callers) != 1 || got.Callers[0].Definition != "B.java:4" || got.Callers[0].Signature != "(String msg)" {
		t.Errorf("callers of log = %+v, want caller with the definition B.java:4 and its signature", got.Callers)
	}
}

// TestCalleesOrderOfASetAndItsFirstMember pins the kind key of the row
// order: a set node has the file, name and line of its first member, and
// one function calls both. The set has the lower id.
func TestCalleesOrderOfASetAndItsFirstMember(t *testing.T) {
	g := newDefGraph()
	one := g.node("z-member", goextract.KindMethod, "add", "Calc.java", 4)
	two := g.node("y-member", goextract.KindMethod, "add", "Calc.java", 6)
	g.set("a-set", one, two)
	g.fn("use", "use", "Run.java", 2)
	g.call("use", "a-set")
	g.call("use", "z-member")
	got, err := g.engine().Callees("use", 0)
	if err != nil {
		t.Fatalf("Callees: %v", err)
	}
	var kinds []string
	for _, l := range got.Callees {
		kinds = append(kinds, l.Kind)
	}
	if diff := cmp.Diff([]string{goextract.KindMethod, goextract.KindOverloadSet}, kinds); diff != "" {
		t.Errorf("kinds of the two rows at Calc.java:4, in order (-want +got):\n%s", diff)
	}
}

// TestAFunctionAndAPackageOfOneName is the scenario of that name: a
// definition with no file is named by its kind and qualified name.
func TestAFunctionAndAPackageOfOneName(t *testing.T) {
	g := newDefGraph()
	g.fn("f", "cache", "main.go", 3)
	pkg := g.node("p", "package", "cache", "", 0)
	pkg.QualifiedName = "example.com/app/cache"
	got, err := g.engine().Callers("cache", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	if diff := cmp.Diff([]string{"package example.com/app/cache", "main.go:3"}, labels(got.Definitions)); diff != "" {
		t.Errorf("definitions of cache (-want +got):\n%s", diff)
	}
}

func jsonKeys(t *testing.T, data []byte) []string {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", data, err)
	}
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func callersJSON(t *testing.T, e *Engine, symbol string) (object []string, firstRow []string) {
	t.Helper()
	got, err := e.Callers(symbol, 0)
	if err != nil {
		t.Fatalf("Callers(%s): %v", symbol, err)
	}
	if len(got.Callers) == 0 {
		t.Fatalf("Callers(%s) has no row, so the test has no entry to read", symbol)
	}
	data, err := MarshalCallersJSON(got)
	if err != nil {
		t.Fatal(err)
	}
	row, err := json.Marshal(got.Callers[0])
	if err != nil {
		t.Fatal(err)
	}
	return jsonKeys(t, data), jsonKeys(t, row)
}

// TestJSONOfANameWithTwoDefinitionsHasDefinitions is the scenario "JSON
// output over several definitions".
func TestJSONOfANameWithTwoDefinitionsHasDefinitions(t *testing.T) {
	e := twoRuns().engine()
	object, row := callersJSON(t, e, "run")
	if diff := cmp.Diff([]string{"callers", "definitions", "symbol"}, object); diff != "" {
		t.Errorf("keys of the object (-want +got):\n%s", diff)
	}
	got, err := e.Callers("run", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	data, err := MarshalCallersJSON(got)
	if err != nil {
		t.Fatal(err)
	}
	var parsed struct {
		Definitions []struct {
			Definition string `json:"definition"`
		} `json:"definitions"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal %s: %v", data, err)
	}
	var defs []string
	for _, d := range parsed.Definitions {
		defs = append(defs, d.Definition)
	}
	if diff := cmp.Diff([]string{"a.py:1", "b.py:1"}, defs); diff != "" {
		t.Errorf("definitions in the JSON (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"definition", "filePath", "kind", "name", "startLine"}, row); diff != "" {
		t.Errorf("keys of the entry (-want +got):\n%s", diff)
	}
}

// TestJSONOfANameWithOneDefinitionHasNoNewKeys is the scenario "One
// definition in JSON".
func TestJSONOfANameWithOneDefinitionHasNoNewKeys(t *testing.T) {
	g := newDefGraph()
	g.fn("alpha", "Alpha", "a.go", 1)
	g.fn("beta", "Beta", "b.go", 1)
	g.call("beta", "alpha")
	object, row := callersJSON(t, g.engine(), "Alpha")
	if diff := cmp.Diff([]string{"callers", "symbol"}, object); diff != "" {
		t.Errorf("keys of the object (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"filePath", "kind", "name", "startLine"}, row); diff != "" {
		t.Errorf("keys of the entry (-want +got):\n%s", diff)
	}
}

// TestJSONOfOneOverloadSetHasNoDefinitionKeys is the scenario "One
// overload set alone", through the real indexer.
func TestJSONOfOneOverloadSetHasNoDefinitionKeys(t *testing.T) {
	object, row := callersJSON(t, overloadFixture(t), "log")
	if diff := cmp.Diff([]string{"callers", "symbol"}, object); diff != "" {
		t.Errorf("keys of the object (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff([]string{"filePath", "kind", "name", "signature", "startLine"}, row); diff != "" {
		t.Errorf("keys of the entry (-want +got):\n%s", diff)
	}
}

// TestCalleesWithFileSelectsOneDefinition is the scenario "Callees of the
// definition in one file".
func TestCalleesWithFileSelectsOneDefinition(t *testing.T) {
	got, err := twoRuns().engine().CalleesSelected(Selector{Symbol: "run"}, "a.py", 0)
	if err != nil {
		t.Fatalf("CalleesSelected: %v", err)
	}
	if diff := cmp.Diff([]string{"load"}, rowNames(got.Callees)); diff != "" {
		t.Errorf("callees of run in a.py (-want +got):\n%s", diff)
	}
	if got.Definitions != nil || got.Callees[0].Definition != "" {
		t.Errorf("a query over one definition carries definition fields: %+v", got)
	}
}

// TestCalleesWithTheEndOfAFilePath is the scenario "A file named by the
// end of its path". The value also differs in letter case.
func TestCalleesWithTheEndOfAFilePath(t *testing.T) {
	g := newDefGraph()
	g.fn("z-m", "prepare_url", "src/app/models.py", 4)
	g.fn("a-t", "prepare_url", "tests/conftest.py", 1)
	g.fn("q", "requote_uri", "src/app/utils.py", 1)
	g.fn("x", "noise", "tests/conftest.py", 9)
	g.call("z-m", "q")
	g.call("a-t", "x")
	got, err := g.engine().CalleesSelected(Selector{Symbol: "prepare_url"}, "Models.py", 0)
	if err != nil {
		t.Fatalf("CalleesSelected: %v", err)
	}
	if diff := cmp.Diff([]string{"requote_uri"}, rowNames(got.Callees)); diff != "" {
		t.Errorf("callees of prepare_url in models.py (-want +got):\n%s", diff)
	}
}

// TestCalleesWithABackslashFilePath pins the spec rule that --file treats
// a backslash as a slash: a Windows-style value selects the definition.
func TestCalleesWithABackslashFilePath(t *testing.T) {
	g := newDefGraph()
	g.fn("z-m", "prepare_url", "src/app/models.py", 4)
	g.fn("a-t", "prepare_url", "tests/conftest.py", 1)
	g.fn("q", "requote_uri", "src/app/utils.py", 1)
	g.fn("x", "noise", "tests/conftest.py", 9)
	g.call("z-m", "q")
	g.call("a-t", "x")
	got, err := g.engine().CalleesSelected(Selector{Symbol: "prepare_url"}, `app\models.py`, 0)
	if err != nil {
		t.Fatalf("CalleesSelected: %v", err)
	}
	if diff := cmp.Diff([]string{"requote_uri"}, rowNames(got.Callees)); diff != "" {
		t.Errorf(`callees of prepare_url in app\models.py (-want +got):`+"\n%s", diff)
	}
}

// TestCallersWithAFileThatMatchesNothingListsEachFileOnce is the scenario
// "A file with no definition of the name". a.py holds two definitions.
func TestCallersWithAFileThatMatchesNothingListsEachFileOnce(t *testing.T) {
	g := twoRuns()
	g.fn("run-a-2", "run", "a.py", 20)
	_, err := g.engine().CallersSelected(Selector{Symbol: "run"}, "c.py", 0)
	if err == nil {
		t.Fatal("CallersSelected(run, c.py): no error")
	}
	const want = `query: no definition of "run" is in a file that matches "c.py"; the files are a.py, b.py`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error %q is not classified as not found", err)
	}
}

// TestFileThatMatchesNothingWithASignatureListsEveryFile pins design D6:
// the error lists each file that holds a definition, also the files
// whose definitions the signature left out.
func TestFileThatMatchesNothingWithASignatureListsEveryFile(t *testing.T) {
	_, err := javaLogs().engine().CallersSelected(Selector{Symbol: "log", Signature: "String, Throwable"}, "Audit.java", 0)
	if err == nil {
		t.Fatal("CallersSelected(log(String, Throwable), Audit.java): no error")
	}
	const want = `query: no definition of "log" is in a file that matches "Audit.java"; the files are Audit.java, Logger.java`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
}

// TestFileWhenNoDefinitionHasAFile covers a name whose definitions all
// have no file, a package and a type in parts: the error says so and
// does not end in an empty list.
func TestFileWhenNoDefinitionHasAFile(t *testing.T) {
	g := newDefGraph()
	g.node("pkg", "package", "cache", "", 0)
	g.node("typ", goextract.KindStruct, "cache", "", 0)
	_, err := g.engine().CallersSelected(Selector{Symbol: "cache"}, "x.go", 0)
	if err == nil {
		t.Fatal("CallersSelected(cache, x.go): no error")
	}
	const want = `query: no definition of "cache" is in a file that matches "x.go"; no definition of "cache" has a file`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error %q is not classified as not found", err)
	}
}

// TestRowsOfOneCallerOrderBySignature pins the signature key of the row
// order. One caller calls both overloads of one set, so its two rows
// tie on file, name, line and definition. The overload with the later
// signature has the earlier line and the lower id, so an order by start
// node or by id puts it first.
func TestRowsOfOneCallerOrderBySignature(t *testing.T) {
	g := newDefGraph()
	str := g.node("z-str", goextract.KindMethod, "add", "Calc.java", 6)
	str.Signature = "(String s)"
	num := g.node("a-int", goextract.KindMethod, "add", "Calc.java", 4)
	num.Signature = "(int n)"
	g.set("set-add", num, str)
	g.fn("use", "use", "Run.java", 2)
	g.call("use", "z-str")
	g.call("use", "a-int")
	got, err := g.engine().Callers("add", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	var sigs []string
	for _, l := range got.Callers {
		sigs = append(sigs, l.Signature)
	}
	if diff := cmp.Diff([]string{"(String s)", "(int n)"}, sigs); diff != "" {
		t.Errorf("signatures of the rows of use, in order (-want +got):\n%s", diff)
	}
}

// TestTwoCallersWithEqualRowsAreBothListed keeps one row for each caller
// node of a name with one definition, also when two callers have the same
// name, kind, file and line, as two nodes of generated code can.
func TestTwoCallersWithEqualRowsAreBothListed(t *testing.T) {
	g := newDefGraph()
	g.fn("target", "target", "t.go", 1)
	g.fn("caller-1", "init", "gen.go", 3)
	g.fn("caller-2", "init", "gen.go", 3)
	g.call("caller-1", "target")
	g.call("caller-2", "target")
	got, err := g.engine().Callers("target", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	if len(got.Callers) != 2 {
		t.Errorf("callers of target = %+v, want one row for each of the two caller nodes", got.Callers)
	}
}

// TestAmbiguousCallerOfTwoSetsIsListedOnce pins design D4: one caller
// makes an ambiguous call into each of two sets of one name. The two
// rows carry no definition and no signature, so they are equal, and the
// result lists the caller once.
func TestAmbiguousCallerOfTwoSetsIsListedOnce(t *testing.T) {
	g := newDefGraph()
	for i, file := range []string{"A.java", "B.java"} {
		one := g.node(fmt.Sprintf("m%d-1", i), goextract.KindMethod, "log", file, 4)
		one.Signature = "(String msg)"
		two := g.node(fmt.Sprintf("m%d-2", i), goextract.KindMethod, "log", file, 6)
		two.Signature = "(int n)"
		g.set(fmt.Sprintf("set%d", i), one, two)
	}
	g.fn("use", "use", "C.java", 2)
	g.call("use", "set0")
	g.call("use", "set1")
	got, err := g.engine().Callers("log", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	want := []Location{{Name: "use", Kind: goextract.KindFunction, FilePath: "C.java", StartLine: 2, Ambiguous: true}}
	if diff := cmp.Diff(want, got.Callers); diff != "" {
		t.Errorf("callers of log (-want +got):\n%s", diff)
	}
}

// TestCallersWithFileAndSignature is the scenario "A file and a
// signature".
func TestCallersWithFileAndSignature(t *testing.T) {
	got, err := javaLogs().engine().CallersSelected(Selector{Symbol: "log", Signature: "String"}, "Logger.java", 0)
	if err != nil {
		t.Fatalf("CallersSelected: %v", err)
	}
	var resolved []string
	for _, l := range got.Callers {
		if !l.Ambiguous {
			resolved = append(resolved, l.Name)
		}
	}
	if diff := cmp.Diff([]string{"start"}, resolved); diff != "" {
		t.Errorf("resolved callers of Logger.log(String) (-want +got):\n%s", diff)
	}
}

// TestFileThatExcludesASetDropsItsAmbiguousCallers pins design D6: the
// Logger set has an ambiguous caller, and a query for Audit.java does
// not list it.
func TestFileThatExcludesASetDropsItsAmbiguousCallers(t *testing.T) {
	g := javaLogs()
	// Positive assertion: the ambiguous caller is there with no file.
	all, err := g.engine().Callers("log", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	if !strings.Contains(strings.Join(rowNames(all.Callers), ","), "use") {
		t.Fatalf("callers of log = %v, want use among them", rowNames(all.Callers))
	}
	got, err := g.engine().CallersSelected(Selector{Symbol: "log"}, "Audit.java", 0)
	if err != nil {
		t.Fatalf("CallersSelected: %v", err)
	}
	if diff := cmp.Diff([]string{"run"}, rowNames(got.Callers)); diff != "" {
		t.Errorf("callers of log in Audit.java (-want +got):\n%s", diff)
	}
}

// drawnGraph is a drawn graph of same-named functions: defs definitions
// of "target" in drawn files at drawn lines, callers that each call a
// drawn subset, and one callee for each definition. The first caller
// calls the first two definitions, so one caller of two definitions
// always exists. When set is true, the first two definitions are the
// members of one overload set with two signatures, and ambiguous lists
// the callers whose call targets the set. A definition with an empty
// file has no file, as a package.
type drawnGraph struct {
	files     []string // file of each definition
	lines     []int32  // line of each definition
	calls     [][]int  // for each caller, the definitions it calls
	set       bool
	ambiguous []int // callers that call the set
}

// drawGraph draws a graph. noFile allows a definition with no file.
func drawGraph(t *rapid.T, noFile bool) drawnGraph {
	n := rapid.IntRange(2, 4).Draw(t, "defs")
	files := []string{"a.go", "b.go", "c.go"}
	if noFile {
		files = append(files, "")
	}
	var d drawnGraph
	for range n {
		d.files = append(d.files, rapid.SampledFrom(files).Draw(t, "file"))
		d.lines = append(d.lines, rapid.Int32Range(1, 2).Draw(t, "line"))
	}
	d.calls = append(d.calls, []int{0, 1})
	for c := rapid.IntRange(0, 4).Draw(t, "callers"); c > 0; c-- {
		d.calls = append(d.calls, rapid.SliceOfNDistinct(rapid.IntRange(0, n-1), 1, n, rapid.ID[int]).Draw(t, "calls"))
	}
	d.set = rapid.Bool().Draw(t, "set")
	if d.set {
		d.ambiguous = rapid.SliceOfNDistinct(rapid.IntRange(0, len(d.calls)-1), 0, len(d.calls), rapid.ID[int]).Draw(t, "ambiguous")
	}
	return d
}

// build makes the graph with the node ids that id gives. Every field
// other than the id is the same for each id function.
func (d drawnGraph) build(id func(string) string) *defGraph {
	g := newDefGraph()
	var members []*schema.Node
	for i, file := range d.files {
		def := fmt.Sprintf("def%d", i)
		n := g.fn(id(def), "target", file, d.lines[i])
		if d.set && i < 2 {
			n.Kind = goextract.KindMethod
			n.Signature = fmt.Sprintf("(int%d n)", i)
			members = append(members, n)
		}
		callee := fmt.Sprintf("callee%d", i)
		g.fn(id(callee), callee, "lib.go", int32(i+1))
		g.call(id(def), id(callee))
	}
	if d.set {
		g.set(id("set"), members...)
	}
	for c, defs := range d.calls {
		caller := fmt.Sprintf("caller%d", c)
		g.fn(id(caller), caller, "use.go", int32(c+1))
		for _, i := range defs {
			g.call(id(caller), id(fmt.Sprintf("def%d", i)))
		}
	}
	for _, c := range d.ambiguous {
		g.call(id(fmt.Sprintf("caller%d", c)), id("set"))
	}
	return g
}

// TestQueryResultDoesNotDependOnNodeIDs builds one drawn graph under two
// id schemes that sort in opposite orders, and requires equal results:
// rows, their order, their labels and the list of definitions, also
// under a drawn limit.
func TestQueryResultDoesNotDependOnNodeIDs(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		d := drawGraph(t, true)
		limit := rapid.IntRange(0, 6).Draw(t, "limit")
		forward := func(name string) string { return "id-" + name }
		backward := func(name string) string {
			b := []byte(name)
			for i := range b {
				b[i] = 255 - b[i]
			}
			return fmt.Sprintf("id-%x", b)
		}
		// Positive assertion: the two schemes order the definitions in
		// opposite ways, so a result that follows the ids differs.
		if (forward("def0") < forward("def1")) == (backward("def0") < backward("def1")) {
			t.Fatalf("the two id schemes order def0 and def1 the same way")
		}
		a, b := d.build(forward).engine(), d.build(backward).engine()

		ca, err := a.Callers("target", limit)
		if err != nil {
			t.Fatalf("Callers: %v", err)
		}
		cb, _ := b.Callers("target", limit)
		if diff := cmp.Diff(ca, cb); diff != "" {
			t.Fatalf("callers differ between the id schemes:\n%s", diff)
		}
		if limit == 0 && len(ca.Callers) < 2 {
			t.Fatalf("callers = %+v, want the two rows of caller0 at least", ca.Callers)
		}
		ea, _ := a.Callees("target", limit)
		eb, _ := b.Callees("target", limit)
		if diff := cmp.Diff(ea, eb); diff != "" {
			t.Fatalf("callees differ between the id schemes:\n%s", diff)
		}
		ia, _ := a.Impact("target", 2)
		ib, _ := b.Impact("target", 2)
		if diff := cmp.Diff(ia, ib); diff != "" {
			t.Fatalf("impact differs between the id schemes:\n%s", diff)
		}
	})
}

// TestFileSelectionPartitionsTheUnion checks that the rows of a query
// with no file are the union of the rows of the same query for each file
// that holds a definition. The comparison leaves the definition label
// out, because a query over one definition carries none.
func TestFileSelectionPartitionsTheUnion(t *testing.T) {
	key := func(l Location) string { return fmt.Sprintf("%s %s:%d", l.Name, l.FilePath, l.StartLine) }
	twoFiles := 0
	rapid.Check(t, func(t *rapid.T) {
		d := drawGraph(t, false)
		e := d.build(func(name string) string { return name }).engine()
		files := map[string]bool{}
		for _, f := range d.files {
			files[f] = true
		}
		if len(files) >= 2 {
			twoFiles++
		}

		all, err := e.Callers("target", 0)
		if err != nil {
			t.Fatalf("Callers: %v", err)
		}
		want := map[string]bool{}
		for _, l := range all.Callers {
			want[key(l)] = true
		}
		got := map[string]bool{}
		for f := range files {
			part, err := e.CallersSelected(Selector{Symbol: "target"}, f, 0)
			if err != nil {
				t.Fatalf("CallersSelected(%s): %v", f, err)
			}
			for _, l := range part.Callers {
				got[key(l)] = true
			}
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("rows with no file against the union of the rows for each file (-want +got):\n%s", diff)
		}
	})
	// Positive assertion: the property ran over more than one file.
	if twoFiles == 0 {
		t.Fatal("no run drew definitions in two files, so the union had one part")
	}
}
