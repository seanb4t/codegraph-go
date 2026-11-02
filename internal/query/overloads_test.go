package query

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// overloadFixture writes and indexes a Java tree with two overloaded
// names: Logger.log(String) and log(String, Throwable), called from
// useA and useB, and Sink.add(int) and add(String), called with null
// from useC so that the call is ambiguous.
func overloadFixture(t *testing.T) *Engine {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"Logger.java": "package app;\n\npublic class Logger {\n    public void log(String msg) {}\n\n    public void log(String msg, Throwable t) {}\n}\n",
		"Sink.java":   "package app;\n\npublic class Sink {\n    public void add(int n) {}\n\n    public void add(String s) {}\n}\n",
		"A.java":      "package app;\n\npublic class A {\n    public void useA(Logger logger) {\n        logger.log(\"a\");\n    }\n}\n",
		"B.java":      "package app;\n\npublic class B {\n    public void useB(Logger logger, Throwable e) {\n        logger.log(\"b\", e);\n    }\n}\n",
		"C.java":      "package app;\n\npublic class C {\n    public void useC(Sink sink) {\n        sink.add(null);\n    }\n}\n",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	indexFixture(t, dir)
	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: %v", err)
	}
	t.Cleanup(func() { _ = closer.Close() })
	return engine
}

func callerRows(r CallersResult) map[string]Location {
	out := make(map[string]Location)
	for _, l := range r.Callers {
		out[l.Name+l.Signature] = l
	}
	return out
}

// TestCallersOfAnOverloadedName is the graph-queries scenario "Callers
// of an overloaded name": `callers log` lists both calls, each with the
// signature of the overload it targets.
func TestCallersOfAnOverloadedName(t *testing.T) {
	engine := overloadFixture(t)
	got, err := engine.Callers("log", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	rows := callerRows(got)
	if len(rows) != 2 {
		t.Fatalf("callers of log = %+v, want useA and useB", got.Callers)
	}
	if l, ok := rows["useA(String msg)"]; !ok || l.Ambiguous {
		t.Errorf("useA row = %+v, want signature (String msg) and not ambiguous", l)
	}
	if l, ok := rows["useB(String msg, Throwable t)"]; !ok || l.Ambiguous {
		t.Errorf("useB row = %+v, want signature (String msg, Throwable t) and not ambiguous", l)
	}
	md := RenderCallersMarkdown(got)
	if !strings.Contains(md, "| Overload |") || !strings.Contains(md, "(String msg, Throwable t)") {
		t.Errorf("markdown lacks the overload column:\n%s", md)
	}
}

// TestCallersMarksAnAmbiguousCaller is the scenario "An ambiguous
// caller": the call add(null) is listed once and marked ambiguous.
func TestCallersMarksAnAmbiguousCaller(t *testing.T) {
	engine := overloadFixture(t)
	got, err := engine.Callers("add", 0)
	if err != nil {
		t.Fatalf("Callers: %v", err)
	}
	if len(got.Callers) != 1 || got.Callers[0].Name != "useC" || !got.Callers[0].Ambiguous || got.Callers[0].Signature != "" {
		t.Fatalf("callers of add = %+v, want useC once, marked ambiguous", got.Callers)
	}
	if md := RenderCallersMarkdown(got); !strings.Contains(md, "| ambiguous |") {
		t.Errorf("markdown does not mark the row ambiguous:\n%s", md)
	}
}

// TestCallersWithSignatureSelectsOneOverload is the scenario "Callers of
// one overload": a signature lists only that overload's callers, and a
// caller that resolved to another overload is absent.
func TestCallersWithSignatureSelectsOneOverload(t *testing.T) {
	engine := overloadFixture(t)
	for signature, want := range map[string]string{"String, Throwable": "useB", "(String)": "useA", "String": "useA"} {
		got, err := engine.CallersSelected(Selector{Symbol: "log", Signature: signature}, 0)
		if err != nil {
			t.Fatalf("CallersSelected(%q): %v", signature, err)
		}
		if len(got.Callers) != 1 || got.Callers[0].Name != want {
			t.Errorf("callers of log --signature %q = %+v, want only %s", signature, got.Callers, want)
		}
	}
	got, err := engine.CallersSelected(Selector{Symbol: "add", Signature: "int"}, 0)
	if err != nil {
		t.Fatalf("CallersSelected(add int): %v", err)
	}
	if len(got.Callers) != 1 || !got.Callers[0].Ambiguous {
		t.Errorf("callers of add --signature int = %+v, want the ambiguous caller useC", got.Callers)
	}
}

// TestCallersWithUnknownSignatureListsTheSignatures is the scenario "A
// signature that matches no overload": the error names the symbol and
// lists the signatures that exist, from callers and from node.
func TestCallersWithUnknownSignatureListsTheSignatures(t *testing.T) {
	engine := overloadFixture(t)
	_, err := engine.CallersSelected(Selector{Symbol: "log", Signature: "int"}, 0)
	if err == nil {
		t.Fatal("CallersSelected(log int): no error")
	}
	for _, want := range []string{`no overload of "log"`, "(String msg)", "(String msg, Throwable t)"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q lacks %q", err, want)
		}
	}
	_, err = engine.NodeSelected(Selector{Symbol: "log", Signature: "int"}, "", nil)
	if err == nil || !strings.Contains(err.Error(), "(String msg, Throwable t)") {
		t.Errorf("NodeSelected(log int) error = %v, want the signatures listed", err)
	}
	out, err := engine.NodeSelected(Selector{Symbol: "log", Signature: "String, Throwable"}, "", nil)
	if err != nil {
		t.Fatalf("NodeSelected(log String, Throwable): %v", err)
	}
	if !strings.Contains(out, "Throwable") || strings.Contains(out, "2 definitions") {
		t.Errorf("node log --signature selected more than one definition:\n%s", out)
	}
}

// TestCalleesAndImpactSpanAnOverloadSet checks that callees and impact
// on a set name report across every overload, and that an ambiguous
// callee is marked.
func TestCalleesAndImpactSpanAnOverloadSet(t *testing.T) {
	engine := overloadFixture(t)
	impact, err := engine.Impact("log", 1)
	if err != nil {
		t.Fatalf("Impact: %v", err)
	}
	names := make(map[string]bool)
	for _, l := range impact.Affected {
		names[l.Name] = true
	}
	for _, want := range []string{"log", "useA", "useB"} {
		if !names[want] {
			t.Errorf("impact of log lacks %s: %+v", want, impact.Affected)
		}
	}
	callees, err := engine.Callees("useC", 0)
	if err != nil {
		t.Fatalf("Callees: %v", err)
	}
	if len(callees.Callees) != 1 || callees.Callees[0].Name != "add" || !callees.Callees[0].Ambiguous {
		t.Errorf("callees of useC = %+v, want the add set, marked ambiguous", callees.Callees)
	}
}

// TestExploreFoldsAnOverloadSet checks the decision on the open question
// of design.md: explore shows one row per overload set, with the count
// and signatures of its overloads, and the callers of every overload
// count on that row.
func TestExploreFoldsAnOverloadSet(t *testing.T) {
	engine := overloadFixture(t)
	out, err := engine.Explore("log", 0)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	if strings.Count(out, "`log`") > 1 {
		t.Errorf("explore lists log more than once:\n%s", out)
	}
	for _, want := range []string{"2 overloads", "(String msg)", "(String msg, Throwable t)", "— 2 callers"} {
		if !strings.Contains(out, want) {
			t.Errorf("explore lacks %q:\n%s", want, out)
		}
	}
}

// TestExploreFoldTruncatesLongSets checks the fold's truncation branch:
// a set with more overloads than the label shows ends with "and N more".
func TestExploreFoldTruncatesLongSets(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString("package app;\n\npublic class Wide {\n")
	for _, typ := range []string{"int", "long", "double", "String", "Object", "char", "boolean"} {
		b.WriteString("    public void put(" + typ + " v) {}\n\n")
	}
	b.WriteString("}\n")
	if err := os.WriteFile(filepath.Join(dir, "Wide.java"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	indexFixture(t, dir)
	engine, closer, err := OpenAt(dir)
	if err != nil {
		t.Fatalf("OpenAt: %v", err)
	}
	defer closer.Close()
	out, err := engine.Explore("put", 0)
	if err != nil {
		t.Fatalf("Explore: %v", err)
	}
	for _, want := range []string{"7 overloads", ", and 1 more)"} {
		if !strings.Contains(out, want) {
			t.Errorf("explore lacks %q:\n%s", want, out)
		}
	}
	label := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "- `put`") {
			label = line
		}
	}
	if strings.Count(label, " v)") != 6 {
		t.Errorf("the fold label shows a number of signatures other than six: %s", label)
	}
}

// TestSearchAndFileSymbolsSkipOverloadSets checks that a search and a
// file's symbol list show the overloads and not the set that stands for
// them.
func TestSearchAndFileSymbolsSkipOverloadSets(t *testing.T) {
	engine := overloadFixture(t)
	locs, err := engine.Search("log", "", 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	methods := 0
	for _, l := range locs {
		if l.Kind == goextract.KindOverloadSet {
			t.Errorf("search lists an overload set: %+v", l)
		}
		if l.Name == "log" && l.Kind == goextract.KindMethod {
			methods++
		}
	}
	if methods != 2 {
		t.Errorf("search lists %d log methods, want 2: %+v", methods, locs)
	}
	syms, err := engine.FileSymbols("Logger.java")
	if err != nil {
		t.Fatalf("FileSymbols: %v", err)
	}
	for _, n := range syms.Symbols {
		if n.Kind == goextract.KindOverloadSet {
			t.Errorf("file symbols list an overload set: %s", n.QualifiedName)
		}
	}
}

// TestNodeDetailWithFileAndSignature checks that a file hint without a
// line still honors the signature: the detail is the selected overload.
func TestNodeDetailWithFileAndSignature(t *testing.T) {
	engine := overloadFixture(t)
	d, err := engine.NodeDetailSelected(Selector{Symbol: "log", Signature: "String, Throwable"}, "Logger.java", nil)
	if err != nil {
		t.Fatalf("NodeDetailSelected: %v", err)
	}
	if d.Mode != NodeDetailModeSingleDef || d.Definition == nil {
		t.Fatalf("mode = %v, want a single definition", d.Mode)
	}
	if got := d.Definition.Node.Signature; got != "(String msg, Throwable t)" {
		t.Errorf("definition signature = %q, want the two-parameter overload", got)
	}
}

// TestCompactSignatureRendersOneLine checks that a label collapses a
// multi-line parameter list and drops control runes.
func TestCompactSignatureRendersOneLine(t *testing.T) {
	got := compactSignature("(\n    String msg,\n\tThrowable t\x07\n)")
	if got != "( String msg, Throwable t )" {
		t.Errorf("compactSignature = %q", got)
	}
}

// TestAffectedReachesACrossFileSet checks that a changed member of a set
// that spans files puts the set on the frontier, so the set's ambiguous
// callers count as affected.
func TestAffectedReachesACrossFileSet(t *testing.T) {
	set := goextract.KindOverloadSet + ":s"
	nodes := map[string]*schema.Node{
		set:         {Id: set, Kind: goextract.KindOverloadSet, Name: "add", QualifiedName: "Sink.add", Language: "csharp"},
		"method:m1": {Id: "method:m1", Kind: goextract.KindMethod, Name: "add", QualifiedName: "Sink.add", FilePath: "Sink.cs", Language: "csharp", StartLine: 3},
		"method:m2": {Id: "method:m2", Kind: goextract.KindMethod, Name: "add", QualifiedName: "Sink.add", FilePath: "Sink2.cs", Language: "csharp", StartLine: 3},
		"func:t":    {Id: "func:t", Kind: goextract.KindFunction, Name: "TestAdd", QualifiedName: "TestAdd", FilePath: "sink_test.go", Language: "go", StartLine: 5},
	}
	edges := []*schema.Edge{
		{Source: set, Target: "method:m1", Kind: goextract.RefKindContains, Provenance: "ast"},
		{Source: set, Target: "method:m2", Kind: goextract.RefKindContains, Provenance: "ast"},
		{Source: "func:t", Target: set, Kind: goextract.RefKindCalls, Provenance: "ast", Metadata: map[string]string{"ambiguous": "true"}},
	}
	e := New(&traverseFakeReader{nodes: nodes, edges: edges})
	got, err := e.Affected([]string{"Sink.cs"}, 1)
	if err != nil {
		t.Fatalf("Affected: %v", err)
	}
	found := false
	for _, l := range got.AffectedTests {
		if l.Name == "TestAdd" {
			found = true
		}
	}
	if !found {
		t.Errorf("Affected(Sink.cs) = %+v, want TestAdd through the cross-file set", got.AffectedTests)
	}
}
