package mcp

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

// numericClaimRe matches a human-readable numeric claim in a tool or
// argument description — "default 2", "max 50", "default 5". These are the
// statements an MCP agent client reads as authoritative about engine
// behavior, and the ones that silently drift when a constant changes.
var numericClaimRe = regexp.MustCompile(`(?i)\b(default|max)\s+(\d+)`)

// engineConstantFor maps a numeric claim in a registered MCP tool schema to
// the internal/query constant that is its single source of truth. Keys are
// "<tool>.<property>.<claim kind>"; the empty property means the tool's own
// top-level description. Any claim matched by numericClaimRe that is NOT in
// this map fails the test — a newly added "default N"/"max N" description
// must be pinned to a constant, not left free-floating.
var engineConstantFor = map[string]string{
	"codegraph_impact.depth.default":      "defaultDepth",
	"codegraph_impact.depth.max":          "MaxDepth",
	"codegraph_explore.max_files.default": "defaultMaxFiles",
}

// TestMCPToolSchemaNumericClaimsMatchEngineConstants pins SURF-01's
// demonstrated escape: Phase 8 changed internal/query's defaultDepth from 5
// to 2 and updated the CLI help and docs/FLAG-PARITY.md, but
// internal/mcp/tools.go kept advertising "BFS depth (default 5, max 50)" —
// so MCP agent clients were told the wrong default for a whole phase.
// internal/cli's TestFlagParityDocCoversRegisteredFlags structurally cannot
// catch this: it walks the cobra command tree and never inspects MCP tool
// schemas.
//
// This test reads BOTH sides from their source of truth rather than
// restating literals: the claimed VALUES come from calling
// jsonschema.For[XArgs](nil) directly on each tool's typed input struct —
// the exact inference mcp.AddTool performs internally (D-07) — since
// go-sdk's Tool.InputSchema is typed any and is populated by AddTool at
// registration time, not present on the *mcp.Tool value exploreTool()/
// companionTool() return on their own. Each tool's top-level description
// still comes from exploreTool()/companionTool(), which set it directly.
// The expected values are parsed out of internal/query/validate.go's const
// block with go/parser — necessary because defaultDepth and defaultMaxFiles
// are unexported and internal/query cannot be imported for their values
// from a package it does not export them to. Changing either side alone
// fails the test.
func TestMCPToolSchemaNumericClaimsMatchEngineConstants(t *testing.T) {
	consts := parseQueryConstants(t)

	type toolClaims struct {
		description string
		properties  map[string]*jsonschema.Schema
	}

	mustSchema := func(s *jsonschema.Schema, err error) map[string]*jsonschema.Schema {
		t.Helper()
		if err != nil {
			t.Fatalf("jsonschema.For: %v", err)
		}
		return s.Properties
	}

	tools := map[string]toolClaims{
		"codegraph_explore": {description: exploreTool().Description, properties: mustSchema(jsonschema.For[ExploreArgs](nil))},
		"codegraph_node":    {description: companionTool("node").Description, properties: mustSchema(jsonschema.For[NodeArgs](nil))},
		"codegraph_search":  {description: companionTool("search").Description, properties: mustSchema(jsonschema.For[SearchArgs](nil))},
		"codegraph_callers": {description: companionTool("callers").Description, properties: mustSchema(jsonschema.For[CallersArgs](nil))},
		"codegraph_callees": {description: companionTool("callees").Description, properties: mustSchema(jsonschema.For[CalleesArgs](nil))},
		"codegraph_impact":  {description: companionTool("impact").Description, properties: mustSchema(jsonschema.For[ImpactArgs](nil))},
		"codegraph_files":   {description: companionTool("files").Description, properties: mustSchema(jsonschema.For[FilesArgs](nil))},
		"codegraph_status":  {description: companionTool("status").Description, properties: mustSchema(jsonschema.For[StatusArgs](nil))},
	}

	// Guard the guard: if the tool set or its descriptions ever stop
	// carrying any numeric claim at all, this test would silently verify
	// nothing.
	claimsChecked := 0

	for toolName, tool := range tools {
		t.Run(toolName, func(t *testing.T) {
			texts := map[string]string{"": tool.description}
			for prop, schema := range tool.properties {
				texts[prop] = schema.Description
			}

			for prop, text := range texts {
				for _, m := range numericClaimRe.FindAllStringSubmatch(text, -1) {
					kind, claimed := m[1], m[2]
					key := fmt.Sprintf("%s.%s.%s", toolName, prop, kind)

					constName, mapped := engineConstantFor[key]
					if !mapped {
						t.Errorf("unpinned numeric claim %q in %s description %q — add %q to engineConstantFor so it cannot drift from the engine", m[0], describeSite(prop), text, key)
						continue
					}
					want, ok := consts[constName]
					if !ok {
						t.Fatalf("%s: engine constant %q not found in internal/query/validate.go", key, constName)
					}
					got, err := strconv.Atoi(claimed)
					if err != nil {
						t.Fatalf("%s: unparseable claimed value %q: %v", key, claimed, err)
					}
					claimsChecked++
					if got != want {
						t.Errorf("%s advertises %s %d but internal/query.%s is %d — MCP clients are being told the wrong value (description: %q)", toolName, kind, got, constName, want, text)
					}
				}
			}
		})
	}

	if claimsChecked < len(engineConstantFor) {
		t.Fatalf("checked only %d numeric claims but %d are pinned in engineConstantFor — a pinned description lost its claim without the map being updated", claimsChecked, len(engineConstantFor))
	}
}

func describeSite(prop string) string {
	if prop == "" {
		return "the tool's own"
	}
	return "argument " + strconv.Quote(prop)
}

// parseQueryConstants reads internal/query/validate.go's untyped integer
// constants straight from source. internal/query deliberately keeps
// defaultDepth and defaultMaxFiles unexported (they are engine policy, not
// API), so source parsing is the only way to assert against their real
// values from another package without duplicating them here.
func parseQueryConstants(t *testing.T) map[string]int {
	t.Helper()

	src := filepath.Join("..", "query", "validate.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, src, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", src, err)
	}

	out := map[string]int{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) != len(vs.Values) {
				continue
			}
			for i, name := range vs.Names {
				lit, ok := vs.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.INT {
					continue
				}
				v, err := strconv.Atoi(lit.Value)
				if err != nil {
					continue
				}
				out[name.Name] = v
			}
		}
	}
	if len(out) == 0 {
		t.Fatalf("parse %s: found no integer constants — this test cannot verify anything", src)
	}
	return out
}
