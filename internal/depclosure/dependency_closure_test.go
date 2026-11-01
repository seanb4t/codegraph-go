// Package depclosure holds the architecture rules that need a package's
// transitive dependency closure. A linter sees one package at a time, so
// the direct-import and call-site rules (D-04a, SDK-02, HYG-02, VRFY-02)
// live in .golangci.yml instead.
package depclosure

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const module = "github.com/seanb4t/codegraph-go"

// listedPackage is the subset of `go list -json` output these rules read.
type listedPackage struct {
	ImportPath string
	Dir        string
	Deps       []string
	CgoFiles   []string
}

// goListDeps runs `go list -deps` over patterns and returns every listed
// package by import path. With test set, the output also holds each test
// variant, such as "p [p.test]", "p_test [p.test]" and "p.test".
func goListDeps(t *testing.T, test bool, patterns ...string) map[string]listedPackage {
	t.Helper()
	args := []string{"list", "-deps", "-json=ImportPath,Dir,Deps,CgoFiles"}
	if test {
		args = append(args, "-test")
	}
	args = append(args, patterns...)
	cmd := exec.Command("go", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go %s: %v\n%s", strings.Join(args, " "), err, stderr.String())
	}
	pkgs := make(map[string]listedPackage)
	dec := json.NewDecoder(&stdout)
	for {
		var p listedPackage
		err := dec.Decode(&p)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		pkgs[p.ImportPath] = p
	}
	return pkgs
}

// basePath maps a test variant's import path back to the package it
// tests: "p [p.test]", "p_test [p.test]" and "p.test" all become "p".
func basePath(importPath string) string {
	if i := strings.IndexByte(importPath, ' '); i >= 0 {
		importPath = importPath[:i]
	}
	importPath = strings.TrimSuffix(importPath, "_test")
	return strings.TrimSuffix(importPath, ".test")
}

func isTestVariant(importPath string) bool {
	return importPath != basePath(importPath)
}

func underPath(importPath, root string) bool {
	return importPath == root || strings.HasPrefix(importPath, root+"/")
}

func depSet(p listedPackage) map[string]bool {
	set := make(map[string]bool, len(p.Deps))
	for _, d := range p.Deps {
		set[basePath(d)] = true
	}
	return set
}

// servePackages are the packages a `serve --mcp` session runs: the MCP
// transport, the query engine, the store, and the daemon, watch and
// indexer machinery that a session can trigger.
var servePackages = []string{
	module + "/internal/mcp",
	module + "/internal/graphstore",
	module + "/internal/daemon",
	module + "/internal/watch",
	module + "/internal/indexer",
	module + "/internal/query",
}

// recordInputs reads go.mod, go.sum and every directory and Go file of
// the module's listed packages. go test caches a result against the files
// the test process itself opened; the reads of the go list subprocess do
// not count. Without these reads, a new import would leave the cached
// result in place.
func recordInputs(t *testing.T, listings ...map[string]listedPackage) {
	t.Helper()
	for _, f := range []string{"../../go.mod", "../../go.sum"} {
		if _, err := os.ReadFile(f); err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
	}
	dirs := make(map[string]bool)
	for _, pkgs := range listings {
		for path, p := range pkgs {
			if underPath(basePath(path), module) && p.Dir != "" {
				dirs[p.Dir] = true
			}
		}
	}
	if len(dirs) == 0 {
		t.Fatal("go list listed zero package directories of this module")
	}
	for dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".go" {
				continue
			}
			if _, err := os.ReadFile(filepath.Join(dir, e.Name())); err != nil {
				t.Fatalf("read %s: %v", e.Name(), err)
			}
		}
	}
}

func TestDependencyClosure(t *testing.T) {
	prod := goListDeps(t, false, module+"/...")
	withTests := goListDeps(t, true, module+"/internal/query/...")
	recordInputs(t, prod, withTests)

	// T-01-18: internal/query is read-side. It must not depend on the
	// wire layer that consumes it, in any variant, test files included.
	t.Run("T-01-18 query reaches no wire layer", func(t *testing.T) {
		forbidden := []string{
			module + "/internal/uiserver",
			module + "/internal/mcp",
			module + "/internal/uiproto",
			"connectrpc.com/connect",
		}
		var checked, testVariants int
		for _, path := range slices.Sorted(maps.Keys(withTests)) {
			p := withTests[path]
			if !underPath(basePath(path), module+"/internal/query") {
				continue
			}
			checked++
			if isTestVariant(path) {
				testVariants++
			}
			deps := depSet(p)
			for _, f := range forbidden {
				if deps[f] {
					t.Errorf("%s depends on %s (perhaps indirectly); internal/query must not depend on the wire layer it is consumed by", path, f)
				}
			}
		}
		if checked == 0 || testVariants == 0 {
			t.Fatalf("checked %d internal/query packages, %d of them test variants; want more than zero of each", checked, testVariants)
		}
	})

	// T-01-18: production internal/query reads indexer output only through
	// internal/graphstore. The leaves internal/indexer/goextract and
	// internal/indexer/nodeid are allowed; the internal/indexer root is
	// not. internal/query/engine_test.go imports the root on purpose to
	// build fixtures, so this rule covers production code only.
	t.Run("T-01-18 production query reaches no indexer root", func(t *testing.T) {
		const indexerRoot = module + "/internal/indexer"
		var checked int
		for _, path := range slices.Sorted(maps.Keys(prod)) {
			p := prod[path]
			if !underPath(path, module+"/internal/query") {
				continue
			}
			checked++
			if depSet(p)[indexerRoot] {
				t.Errorf("production package %s depends on %s (perhaps indirectly); only internal/indexer/goextract and internal/indexer/nodeid are allowed", path, indexerRoot)
			}
		}
		if checked == 0 {
			t.Fatal("checked zero production internal/query packages")
		}
	})

	// TUI-01: the agent and MCP surface stays free of ANSI styling, so no
	// Charm package may be in the production closure of a serve package.
	t.Run("TUI-01 serve closure holds no Charm package", func(t *testing.T) {
		closure := make(map[string]bool)
		for _, root := range servePackages {
			p, ok := prod[root]
			if !ok {
				t.Fatalf("go list did not list %s", root)
			}
			closure[root] = true
			for _, d := range p.Deps {
				closure[d] = true
			}
		}
		if len(closure) <= len(servePackages) {
			t.Fatalf("serve closure holds %d packages, no more than its %d roots", len(closure), len(servePackages))
		}
		for _, path := range slices.Sorted(maps.Keys(closure)) {
			if strings.HasPrefix(path, "charm.land/") || strings.HasPrefix(path, "github.com/charmbracelet/") {
				t.Errorf("%s is in the production closure of the serve packages; Charm packages belong only under internal/cli", path)
			}
		}
	})

	// REL-01: the Charm closure of internal/cli adds no CGo. tree-sitter is
	// the one CGo dependency, and it is outside this closure.
	t.Run("REL-01 Charm closure has no CGo", func(t *testing.T) {
		charm := make(map[string]bool)
		for path, p := range prod {
			if !underPath(path, module+"/internal/cli") {
				continue
			}
			for _, d := range p.Deps {
				if strings.HasPrefix(d, "charm.land/") {
					charm[d] = true
				}
			}
		}
		if len(charm) == 0 {
			t.Fatal("the internal/cli closure holds zero charm.land packages")
		}
		for _, path := range slices.Sorted(maps.Keys(charm)) {
			if len(prod[path].CgoFiles) > 0 {
				t.Errorf("%s has CgoFiles %v; the Charm closure must add no CGo", path, prod[path].CgoFiles)
			}
		}
	})
}
