// Package tsextract_test is an EXTERNAL test package (not `package
// tsextract`, the internal test package tsextract_test.go uses)
// deliberately: it drives the real internal/indexer.Run pipeline
// end-to-end, and internal/indexer itself imports tsextract
// (languages_typescript.go) — a same-package (internal) test file
// importing internal/indexer would create an import cycle. The
// external-test-package pattern is the standard Go mechanism for
// exercising a package's consumer from its own test tree without that
// cycle — mirrors javaextract/csharpextract/pyextract's own
// resolution_test.go shape exactly.
package tsextract_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// tsFixtureFile is one file this test writes into a temp repo root.
type tsFixtureFile struct {
	relPath, src string
}

// writeTSFixture always includes a tsconfig.json (baseUrl "." + a
// `@app/*` -> `src/app/*` paths alias) and a package.json, so the real
// languages_typescript.go Descriptor resolves and installs tsextract's
// module-resolution Config — without it, the paths-aliased import
// statements these fixtures use could never resolve (relative-specifier
// resolution alone needs no descriptor, but this test exercises BOTH
// tiers per the plan's own Task 2 must_have).
func writeTSFixture(t *testing.T, files []tsFixtureFile) string {
	t.Helper()
	root := t.TempDir()
	all := append([]tsFixtureFile{
		{relPath: "tsconfig.json", src: `{"compilerOptions":{"baseUrl":".","paths":{"@app/*":["src/app/*"]}}}`},
		{relPath: "package.json", src: `{"name":"demo"}`},
	}, files...)
	for _, f := range all {
		abs := filepath.Join(root, filepath.FromSlash(f.relPath))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir for fixture %s: %v", f.relPath, err)
		}
		if err := os.WriteFile(abs, []byte(f.src), 0o644); err != nil {
			t.Fatalf("writing fixture %s: %v", f.relPath, err)
		}
	}
	return root
}

// nodeKey identifies a committed node by its stable, human-legible fields
// (never by its opaque hash id, which this test never needs to compute).
type nodeKey struct {
	kind, name, filePath string
}

// buildTSGraph runs the real indexer.Run pipeline against root into a
// fresh temp store and returns every committed node (indexed by nodeKey)
// and edge, for direct inspection.
func buildTSGraph(t *testing.T, root string) (map[nodeKey]*schema.Node, []*schema.Edge) {
	t.Helper()

	storeDir := t.TempDir()
	if _, err := indexer.Run(root, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("indexer.Run: %v", err)
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	reader, err := store.Snapshot()
	if err != nil {
		t.Fatalf("store.Snapshot: %v", err)
	}
	t.Cleanup(func() { _ = reader.Close() })

	nodes := make(map[nodeKey]*schema.Node)
	nit, err := reader.IterateNodes()
	if err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}
	defer nit.Close()
	for nit.Next() {
		n := nit.Node()
		nodes[nodeKey{kind: n.Kind, name: n.Name, filePath: n.FilePath}] = n
	}
	if err := nit.Err(); err != nil {
		t.Fatalf("IterateNodes: %v", err)
	}

	var edges []*schema.Edge
	eit, err := reader.IterateEdges("")
	if err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}
	defer eit.Close()
	for eit.Next() {
		edges = append(edges, eit.Edge())
	}
	if err := eit.Err(); err != nil {
		t.Fatalf("IterateEdges: %v", err)
	}

	return nodes, edges
}

func hasEdge(edges []*schema.Edge, source, target, kind string) bool {
	for _, e := range edges {
		if e.Source == source && e.Target == target && e.Kind == kind {
			return true
		}
	}
	return false
}

// TestResolve_RelativeSpecifierCrossFileCall proves a relative-specifier
// named import (`import { assist } from './helper'`) + a bare call
// resolves into a real "calls" edge via the resolved module specifier —
// the plan's own PRIORITY resolution tier (Task 2's first must_have
// truth).
func TestResolve_RelativeSpecifierCrossFileCall(t *testing.T) {
	root := writeTSFixture(t, []tsFixtureFile{
		{relPath: "src/app/helper.ts", src: "export function assist(): void {}\n"},
		{relPath: "src/app/caller.ts", src: "import { assist } from './helper';\n\nexport function run(): void {\n  assist();\n}\n"},
	})

	nodes, edges := buildTSGraph(t, root)

	assist, ok := nodes[nodeKey{kind: "function", name: "assist", filePath: "src/app/helper.ts"}]
	if !ok {
		t.Fatal("assist node not found in committed graph")
	}
	run, ok := nodes[nodeKey{kind: "function", name: "run", filePath: "src/app/caller.ts"}]
	if !ok {
		t.Fatal("run node not found in committed graph")
	}

	if !hasEdge(edges, run.Id, assist.Id, "calls") {
		t.Errorf("expected a calls edge run -> assist (cross-file, relative specifier), got edges: %+v", edges)
	}
}

// TestResolve_TSConfigPathsAliasedCrossFileCall proves a tsconfig.json
// `paths`-aliased import (`import { assist } from '@app/helper'`) + a bare
// call resolves into a real "calls" edge via the SAME resolved module
// specifier a relative import to the identical file would produce (Task
// 2's first must_have truth, the tsconfig-aware tier).
func TestResolve_TSConfigPathsAliasedCrossFileCall(t *testing.T) {
	root := writeTSFixture(t, []tsFixtureFile{
		{relPath: "src/app/helper.ts", src: "export function assist(): void {}\n"},
		{relPath: "src/consumer.ts", src: "import { assist } from '@app/helper';\n\nexport function run(): void {\n  assist();\n}\n"},
	})

	nodes, edges := buildTSGraph(t, root)

	assist, ok := nodes[nodeKey{kind: "function", name: "assist", filePath: "src/app/helper.ts"}]
	if !ok {
		t.Fatal("assist node not found in committed graph")
	}
	run, ok := nodes[nodeKey{kind: "function", name: "run", filePath: "src/consumer.ts"}]
	if !ok {
		t.Fatal("run node not found in committed graph")
	}

	if !hasEdge(edges, run.Id, assist.Id, "calls") {
		t.Errorf("expected a calls edge run -> assist (cross-file, tsconfig paths alias), got edges: %+v", edges)
	}
}

// TestResolve_CrossFileInheritance proves an imported base class
// (`import { Base } from './base'; class Derived extends Base {}`)
// resolves into a real "embeds" edge via the resolved module specifier.
func TestResolve_CrossFileInheritance(t *testing.T) {
	root := writeTSFixture(t, []tsFixtureFile{
		{relPath: "src/app/base.ts", src: "export class Base {}\n"},
		{relPath: "src/app/derived.ts", src: "import { Base } from './base';\n\nexport class Derived extends Base {}\n"},
	})

	nodes, edges := buildTSGraph(t, root)

	base, ok := nodes[nodeKey{kind: "struct", name: "Base", filePath: "src/app/base.ts"}]
	if !ok {
		t.Fatal("Base node not found in committed graph")
	}
	derived, ok := nodes[nodeKey{kind: "struct", name: "Derived", filePath: "src/app/derived.ts"}]
	if !ok {
		t.Fatal("Derived node not found in committed graph")
	}

	if !hasEdge(edges, derived.Id, base.Id, "embeds") {
		t.Errorf("expected an embeds edge Derived -> Base (cross-file, relative specifier), got edges: %+v", edges)
	}
}

// TestResolve_NamespaceImportCrossFileCall proves a namespace import
// (`import * as helper from './helper'`) + a member-access call
// (`helper.assist()`) resolves into a real "calls" edge.
func TestResolve_NamespaceImportCrossFileCall(t *testing.T) {
	root := writeTSFixture(t, []tsFixtureFile{
		{relPath: "src/app/helper.ts", src: "export function assist(): void {}\n"},
		{relPath: "src/app/caller.ts", src: "import * as helper from './helper';\n\nexport function run(): void {\n  helper.assist();\n}\n"},
	})

	nodes, edges := buildTSGraph(t, root)

	assist, ok := nodes[nodeKey{kind: "function", name: "assist", filePath: "src/app/helper.ts"}]
	if !ok {
		t.Fatal("assist node not found in committed graph")
	}
	run, ok := nodes[nodeKey{kind: "function", name: "run", filePath: "src/app/caller.ts"}]
	if !ok {
		t.Fatal("run node not found in committed graph")
	}

	if !hasEdge(edges, run.Id, assist.Id, "calls") {
		t.Errorf("expected a calls edge run -> assist (namespace import, member access), got edges: %+v", edges)
	}
}
