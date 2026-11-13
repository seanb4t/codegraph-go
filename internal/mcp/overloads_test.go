package mcp

import (
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

// javaLoggers is a Logger with log(String) and log(String, Throwable), and
// useA and useB, which call one overload each.
var javaLoggers = map[string]string{
	"Logger.java": "package app;\n\npublic class Logger {\n    public void log(String msg) {}\n\n    public void log(String msg, Throwable t) {}\n}\n",
	"A.java":      "package app;\n\npublic class A {\n    public void useA(Logger logger) {\n        logger.log(\"a\");\n    }\n}\n",
	"B.java":      "package app;\n\npublic class B {\n    public void useB(Logger logger, Throwable e) {\n        logger.log(\"b\", e);\n    }\n}\n",
}

// TestCallersToolOnASetListsEveryOverload is the graph-queries scenario
// "An MCP callers call on a set": codegraph_callers with symbol "log"
// and no signature lists the callers of both overloads, each with its
// signature, and with a signature lists only that overload's caller.
func TestCallersToolOnASetListsEveryOverload(t *testing.T) {
	call := treeServer(t, javaLoggers, "callers")
	all := call("codegraph_callers", map[string]any{"symbol": "log"})
	for _, want := range []string{"| Overload |", "`useA`", "(String msg)", "`useB`", "(String msg, Throwable t)"} {
		if !strings.Contains(all, want) {
			t.Errorf("callers of log lacks %q:\n%s", want, all)
		}
	}
	one := call("codegraph_callers", map[string]any{"symbol": "log", "signature": "String, Throwable"})
	if !strings.Contains(one, "`useB`") || strings.Contains(one, "`useA`") {
		t.Errorf("callers of log with a signature = \n%s\nwant useB only", one)
	}
}

// TestMCPToolSchemasExposeSignature checks that the four symbol tools
// accept the signature input of the graph-queries spec.
func TestMCPToolSchemasExposeSignature(t *testing.T) {
	schemas := map[string]func() (*jsonschema.Schema, error){
		"codegraph_node":    func() (*jsonschema.Schema, error) { return jsonschema.For[NodeArgs](nil) },
		"codegraph_callers": func() (*jsonschema.Schema, error) { return jsonschema.For[CallersArgs](nil) },
		"codegraph_callees": func() (*jsonschema.Schema, error) { return jsonschema.For[CalleesArgs](nil) },
		"codegraph_impact":  func() (*jsonschema.Schema, error) { return jsonschema.For[ImpactArgs](nil) },
	}
	for tool, build := range schemas {
		s, err := build()
		if err != nil {
			t.Fatalf("%s: jsonschema.For: %v", tool, err)
		}
		prop, ok := s.Properties["signature"]
		if !ok || prop.Type != "string" || !strings.Contains(prop.Description, "overload") {
			t.Errorf("%s: signature property = %+v, want a string that selects one overload", tool, prop)
		}
	}
}

// TestSignatureInputOnEveryTool checks the signature input of the node,
// callees and impact tools: each one narrows to the named overload.
func TestSignatureInputOnEveryTool(t *testing.T) {
	call := treeServer(t, javaLoggers, "node", "callees", "impact")

	node := call("codegraph_node", map[string]any{"symbol": "log", "signature": "String, Throwable"})
	if !strings.Contains(node, "(String msg, Throwable t)") || strings.Contains(node, "2 definitions") {
		t.Errorf("codegraph_node with a signature = \n%s\nwant the two-parameter overload alone", node)
	}
	impact := call("codegraph_impact", map[string]any{"symbol": "log", "signature": "String"})
	if !strings.Contains(impact, "`useA`") || strings.Contains(impact, "`useB`") {
		t.Errorf("codegraph_impact with a signature = \n%s\nwant useA and not useB", impact)
	}
	callees := call("codegraph_callees", map[string]any{"symbol": "useB", "signature": "Logger, Throwable"})
	if !strings.Contains(callees, "`log`") {
		t.Errorf("codegraph_callees with a signature = \n%s\nwant the log callee", callees)
	}
}
