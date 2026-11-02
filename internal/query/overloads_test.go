package query

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
