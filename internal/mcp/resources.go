package mcp

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// resourcesFS embeds every fact-sheet/behavior-doc markdown file this
// server serves over resources/list and resources/read (D-07). This is
// the first //go:embed use in this repository. The 8 per-tool fact-sheets
// and 2 behavior docs live side by side in one directory and register
// through one loop (registerResources below) — no special-casing.
//
//go:embed resources/*.md
var resourcesFS embed.FS

// resourceURIFor is the ONE place the resource URI scheme lives —
// D-08 (custom "codegraph" scheme, visually distinct from file/http(s) URIs), D-09
// (per-tool URIs use the short companion/CLI name, not the full
// registered tool name), D-10 (behavior-doc URIs name what the doc is
// about, not its current implementation mechanism). Both
// registerResources and every later drift guard read this map; no second
// hand-typed URI list exists anywhere else in this package.
var resourceURIFor = map[string]string{
	"explore.md": "codegraph://tools/explore",
	"node.md":    "codegraph://tools/node",
	"search.md":  "codegraph://tools/search",
	"callers.md": "codegraph://tools/callers",
	"callees.md": "codegraph://tools/callees",
}

// resourceDescriptionFor returns the Description a registered resource
// should carry, for a stem naming a codegraph tool (RSRC-01/GUARD-01).
// It always calls back into the tool's own Description — via
// exploreTool() or companionTool(stem) — rather than a hand-typed copy,
// so a resources/list payload's description can never drift from
// tools.go. A stem matching one of companionNames resolves through
// companionTool(stem), since D-09's URI naming already uses the short
// companion name as both the map key and the tool lookup key. Panics on
// a stem with no known source, a build-time invariant rather than a
// runtime condition — mirroring companionTool's and companionHandler's
// existing panic-on-unknown-name convention.
func resourceDescriptionFor(stem string) string {
	switch stem {
	case "explore":
		return exploreTool().Description
	default:
		for _, name := range companionNames {
			if stem == name {
				return companionTool(stem).Description
			}
		}
		panic("mcp: resourceDescriptionFor: unknown resource stem " + stem)
	}
}

// registerResources iterates the embedded resources directory and calls
// s.AddResource once per file, unconditionally (RSRC-03) — called from
// BuildServer immediately after mcp.NewServer(...) and BEFORE the
// `if hasIndex {` tool-registration branch, which is the structural
// property that makes resources available in an unindexed repository:
// this function's call site does not sit inside any conditional that
// depends on hasIndex.
//
// A ReadDir failure, a filename with no resourceURIFor entry, or a
// ReadFile failure all panic naming the filename — these are build-time
// invariants (a missing/misnamed embedded file is a programmer error
// caught at server construction, always before any client connects),
// not runtime conditions, mirroring this package's existing
// companionTool/companionHandler panic-on-unknown-name convention.
func registerResources(s *mcp.Server) {
	entries, err := fs.ReadDir(resourcesFS, "resources")
	if err != nil {
		panic(fmt.Sprintf("registerResources: read embedded resources dir: %v", err))
	}
	for _, entry := range entries {
		name := entry.Name()
		uri, ok := resourceURIFor[name]
		if !ok {
			panic(fmt.Sprintf("registerResources: %s has no entry in resourceURIFor", name))
		}
		data, err := resourcesFS.ReadFile("resources/" + name)
		if err != nil {
			panic(fmt.Sprintf("registerResources: read %s: %v", name, err))
		}
		stem := name[:len(name)-len(".md")]
		text := string(data)
		s.AddResource(&mcp.Resource{
			URI:         uri,
			Name:        name,
			Description: resourceDescriptionFor(stem),
			MIMEType:    "text/markdown",
		}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			// MIMEType is deliberately left unset here — go-sdk's
			// readResource back-fills ResourceContents.MIMEType from the
			// registered Resource.MIMEType above (server.go:1040-1048)
			// whenever the handler leaves it empty.
			return &mcp.ReadResourceResult{
				Contents: []*mcp.ResourceContents{{URI: req.Params.URI, Text: text}},
			}, nil
		})
	}
}
