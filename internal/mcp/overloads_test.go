package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestCallersToolOnASetListsEveryOverload is the graph-queries scenario
// "An MCP callers call on a set": codegraph_callers with symbol "log"
// and no signature lists the callers of both overloads, each with its
// signature, and with a signature lists only that overload's caller.
func TestCallersToolOnASetListsEveryOverload(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"Logger.java": "package app;\n\npublic class Logger {\n    public void log(String msg) {}\n\n    public void log(String msg, Throwable t) {}\n}\n",
		"A.java":      "package app;\n\npublic class A {\n    public void useA(Logger logger) {\n        logger.log(\"a\");\n    }\n}\n",
		"B.java":      "package app;\n\npublic class B {\n    public void useB(Logger logger, Throwable e) {\n        logger.log(\"b\", e);\n    }\n}\n",
	}
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	indexFixture(t, dir)
	s := BuildServer(true, map[string]bool{"callers": true}, dir, dir)

	text := func(result *mcp.CallToolResult) string {
		var b strings.Builder
		for _, c := range result.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				b.WriteString(tc.Text)
			}
		}
		return b.String()
	}
	all := text(callTool(t, s, "codegraph_callers", map[string]any{"symbol": "log"}))
	for _, want := range []string{"| Overload |", "`useA`", "(String msg)", "`useB`", "(String msg, Throwable t)"} {
		if !strings.Contains(all, want) {
			t.Errorf("callers of log lacks %q:\n%s", want, all)
		}
	}
	one := text(callTool(t, s, "codegraph_callers", map[string]any{"symbol": "log", "signature": "String, Throwable"}))
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
