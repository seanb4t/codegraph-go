// Command gocapture regenerates ALL Go-side EXPECTED fixtures (F5, 01-RESEARCH.md,
// FIXT-06) by running the CURRENT Go indexer + query.Engine pipeline against both
// the Phase-1-locked corpora (resolved through internal/corpora — never a
// hardcoded SHA or path) and the committed behavioral corpus, writing its output
// as go-*.json / go-*-mcp.json fixtures.
//
// The TS-era capture path, external network-fetched corpora, and the
// skip-warn-on-missing contract are retired (FIXT-04, MED). gocapture is the
// sole capture authority — it FAILS CLOSED on a missing mandatory source.
//
// Usage: go run ./testdata/golden/gocapture
package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/seanb4t/codegraph-go/internal/corpora"
	"github.com/seanb4t/codegraph-go/internal/goldenspec"
	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// corpusSpec is one corpus's regeneration recipe: how to resolve its
// source tree, the output directory for its goldens, and the symbol/query
// parameters for its fixtures.
type corpusSpec struct {
	name string
	// outDir returns the directory where this spec's golden fixtures are written.
	outDir func() string
	// resolveSource returns the corpus's indexable source directory, or an error
	// if the source cannot be resolved (a hard failure — gocapture exits non-zero).
	resolveSource func() (string, error)
	baselineSymbol     string
	baselineSymbolFile string
	baselineQuery      string
	multiSymbol        string
	multiQuery         string
}

func main() {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		fatal("gocapture: runtime.Caller(0) failed to resolve this file's own path")
	}
	goldenDir := filepath.Dir(filepath.Dir(thisFile)) // .../testdata/golden
	repoRoot := filepath.Dir(filepath.Dir(goldenDir)) // ... (two hops to repo root)

	specs := buildSpecs(goldenDir, repoRoot)

	failures := 0
	for _, spec := range specs {
		if err := regenerateCorpus(spec); err != nil {
			fmt.Fprintf(os.Stderr, "gocapture: [%s] FAILED: %v\n", spec.name, err)
			failures++
		}
	}
	if failures > 0 {
		os.Exit(1)
	}
}

// buildSpecs constructs the full list of corpusSpecs: locked corpora first, then
// the behavioral corpus. Locked corpora resolve through internal/corpora.
func buildSpecs(goldenDir, repoRoot string) []corpusSpec {
	corpusRoot, err := corpora.CorpusRoot()
	if err != nil {
		fatal(fmt.Sprintf("gocapture: CorpusRoot: %v", err))
	}
	m, err := corpora.Load(filepath.Join(repoRoot, "corpora", "manifest.json"))
	if err != nil {
		fatal(fmt.Sprintf("gocapture: load manifest: %v", err))
	}
	locked := corpora.LockedEntries(m)

	// Build an index: repo slug -> locked Entry for fast lookup.
	lockedByRepo := make(map[string]corpora.Entry, len(locked))
	for _, e := range locked {
		lockedByRepo[e.Repo] = e
	}

	var specs []corpusSpec

	// Locked-corpus specs, one per slug in order.
	slugOrder := []string{"hugo", "guava", "serilog", "requests"}
	for _, slug := range slugOrder {
		repo := goldenspec.SlugToRepo[slug]
		e, ok := lockedByRepo[repo]
		if !ok {
			fatal(fmt.Sprintf("gocapture: locked entry %q (slug=%q) not found in manifest", repo, slug))
		}
		args := goldenspec.LockedCorpusArgs[slug]
		out := filepath.Join(goldenDir, "corpus", slug)
		specs = append(specs, corpusSpec{
			name: slug,
			outDir: func() string {
				return out
			},
			resolveSource: func() (string, error) {
				return e.Dir(corpusRoot), nil
			},
			baselineSymbol:     args.BaselineSymbol,
			baselineSymbolFile: args.BaselineSymbolFile,
			baselineQuery:      args.BaselineQuery,
			multiSymbol:        args.MultiSymbol,
			multiQuery:         args.MultiQuery,
		})
	}

	// Behavioral corpus spec.
	specs = append(specs, behavioralCorpusSpec(repoRoot))

	return specs
}

// regenerateCorpus indexes the corpus source and writes all golden fixtures
// (CLI and MCP surfaces). Fail-closed: any index or capture error returns
// a non-nil error so main() exits non-zero.
func regenerateCorpus(spec corpusSpec) error {
	sourcePath, err := spec.resolveSource()
	if err != nil {
		return fmt.Errorf("resolve source: %w", err)
	}
	if sourcePath == "" {
		return fmt.Errorf("resolve source returned an empty path with no error")
	}
	info, err := os.Stat(sourcePath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("source path %s is not a directory or is inaccessible: %w", sourcePath, err)
	}

	fmt.Fprintf(os.Stderr, "gocapture: [%s] indexing %s ...\n", spec.name, sourcePath)
	storeDir, err := os.MkdirTemp("", "gocapture-store-*")
	if err != nil {
		return fmt.Errorf("MkdirTemp: %w", err)
	}
	defer os.RemoveAll(storeDir)

	if _, err := indexer.Run(sourcePath, storeDir, indexer.Options{Quiet: true}); err != nil {
		return fmt.Errorf("indexer.Run(%s): %w", sourcePath, err)
	}

	store, err := graphstore.Open(storeDir)
	if err != nil {
		return fmt.Errorf("graphstore.Open: %w", err)
	}
	defer store.Close()

	reader, err := store.Snapshot()
	if err != nil {
		return fmt.Errorf("store.Snapshot: %w", err)
	}
	defer reader.Close()

	eng := query.NewWithRoot(reader, sourcePath)
	out := spec.outDir()
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fmt.Errorf("MkdirAll(%s): %w", out, err)
	}

	// --- CLI-surface captures ---
	if spec.baselineSymbol != "" {
		exploreOut, err := eng.Explore(spec.baselineQuery, 1)
		if err != nil {
			return fmt.Errorf("Explore(%q, maxFiles=1): %w", spec.baselineQuery, err)
		}
		if err := writeCapture(filepath.Join(out, "go-explore.json"),
			fmt.Sprintf("explore %q -p %s --max-files 1", spec.baselineQuery, spec.name), exploreOut); err != nil {
			return err
		}

		nodeOut, err := eng.Node(spec.baselineSymbol, spec.baselineSymbolFile, nil)
		if err != nil {
			return fmt.Errorf("Node(%q, %q): %w", spec.baselineSymbol, spec.baselineSymbolFile, err)
		}
		if err := writeCapture(filepath.Join(out, "go-node.json"),
			fmt.Sprintf("node %q -p %s -f %s", spec.baselineSymbol, spec.name, spec.baselineSymbolFile), nodeOut); err != nil {
			return err
		}
	}

	// Multi (best-effort) CLI captures. For locked-corpus specs, a missing
	// golden is a hard error (folded into the corpus's overall error return).
	var multiErr error

	exploreMultiOut, err := eng.Explore(spec.multiQuery, 0)
	if err != nil {
		multiErr = fmt.Errorf("Explore(%q, maxFiles=0): %w", spec.multiQuery, err)
		fmt.Fprintf(os.Stderr, "gocapture: [%s] WARNING: %v\n", spec.name, multiErr)
	} else if err := writeCapture(filepath.Join(out, "go-explore-multi.json"),
		fmt.Sprintf("explore %q -p %s", spec.multiQuery, spec.name), exploreMultiOut); err != nil {
		return err
	}

	nodeMultiOut, err := eng.Node(spec.multiSymbol, "", nil)
	if err != nil {
		nodeErr := fmt.Errorf("Node(%q, \"\"): %w", spec.multiSymbol, err)
		fmt.Fprintf(os.Stderr, "gocapture: [%s] WARNING: %v\n", spec.name, nodeErr)
		if multiErr == nil {
			multiErr = nodeErr
		}
	} else if err := writeCapture(filepath.Join(out, "go-node-multi.json"),
		fmt.Sprintf("node %q -p %s", spec.multiSymbol, spec.name), nodeMultiOut); err != nil {
		return err
	}

	// --- MCP-surface captures ---
	// Stage the corpus in a temp directory and index it so the MCP server
	// (which uses query.OpenAt) can find a real store on disk.
	mcpTmp, err := os.MkdirTemp("", "gocapture-mcp-*")
	if err != nil {
		return fmt.Errorf("MkdirTemp for MCP staging: %w", err)
	}
	defer os.RemoveAll(mcpTmp)

	if err := copyDir(sourcePath, mcpTmp); err != nil {
		return fmt.Errorf("copy corpus to MCP staging dir: %w", err)
	}
	mcpStoreDir := filepath.Join(mcpTmp, ".codegraph", "store")
	if err := os.MkdirAll(mcpStoreDir, 0o755); err != nil {
		return fmt.Errorf("mkdir MCP store dir: %w", err)
	}
	if _, err := indexer.Run(mcpTmp, mcpStoreDir, indexer.Options{Quiet: true}); err != nil {
		return fmt.Errorf("index MCP staging dir: %w", err)
	}

	if spec.baselineSymbol != "" {
		mcpExploreOut, err := goldenspec.CallExploreViaMCP(mcpTmp, spec.baselineQuery)
		if err != nil {
			return fmt.Errorf("MCP Explore(%q): %w", spec.baselineQuery, err)
		}
		if err := writeCapture(filepath.Join(out, "go-explore-mcp.json"),
			fmt.Sprintf("MCP explore %q -p %s", spec.baselineQuery, spec.name), mcpExploreOut); err != nil {
			return err
		}

		mcpNodeOut, err := goldenspec.CallNodeViaMCP(mcpTmp, spec.baselineSymbol)
		if err != nil {
			return fmt.Errorf("MCP Node(%q): %w", spec.baselineSymbol, err)
		}
		if err := writeCapture(filepath.Join(out, "go-node-mcp.json"),
			fmt.Sprintf("MCP node %q -p %s", spec.baselineSymbol, spec.name), mcpNodeOut); err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "gocapture: [%s] wrote go-*.json fixtures to %s\n", spec.name, out)
	return multiErr
}

// writeCapture writes the capture envelope to path via capture-to-temp-then-move
// (Pattern 2, carryover 01-03): writes to a temp path in the same directory,
// asserts the temp is non-empty and byte-prefixed with the `{` marker, then
// renames onto the committed path. Any failure (empty/marker-less/temp-create/
// rename) returns an error — never leaves a bare golden on the committed path
// and never partially writes.
func writeCapture(path, command, output string) error {
	data, err := json.MarshalIndent(goldenspec.GoldenCapture{Command: command, Output: output}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "*.gocapture-tmp")
	if err != nil {
		return fmt.Errorf("CreateTemp in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()

	cleanupOK := false
	defer func() {
		if !cleanupOK {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp %s: %w", tmpPath, err)
	}

	// Assert the temp file is non-empty and carries the envelope marker.
	fi, err := os.Stat(tmpPath)
	if err != nil {
		return fmt.Errorf("stat temp %s: %w", tmpPath, err)
	}
	if fi.Size() == 0 {
		return fmt.Errorf("temp %s is empty — refusing to rename onto committed path", tmpPath)
	}
	marker := make([]byte, 1)
	f, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("open temp %s for marker check: %w", tmpPath, err)
	}
	if _, err := f.Read(marker); err != nil {
		f.Close()
		return fmt.Errorf("read marker from temp %s: %w", tmpPath, err)
	}
	f.Close()
	if marker[0] != '{' {
		return fmt.Errorf("temp %s first byte is %q (want '{') — not a valid JSON envelope", tmpPath, marker)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", tmpPath, path, err)
	}
	cleanupOK = true
	return nil
}

// copyDir recursively copies src into dst, skipping ".codegraph" directories
// so the destination starts clean for indexing. Handles symlinks: resolves
// directory symlinks by walking their target, and copies file symlinks by
// reading the resolved path's content.
func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == ".codegraph" {
			return fs.SkipDir
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			if rInfo, err := os.Stat(resolved); err == nil && rInfo.IsDir() {
				if err := os.MkdirAll(target, 0o755); err != nil {
					return err
				}
				return filepath.WalkDir(resolved, func(rPath string, rD fs.DirEntry, rErr error) error {
					if rErr != nil {
						return rErr
					}
					rRel, rErr := filepath.Rel(resolved, rPath)
					if rErr != nil {
						return rErr
					}
					rTarget := filepath.Join(target, rRel)
					if rD.IsDir() {
						return os.MkdirAll(rTarget, 0o755)
					}
					data, rErr := os.ReadFile(rPath)
					if rErr != nil {
						return rErr
					}
					return os.WriteFile(rTarget, data, 0o644)
				})
			}
			data, err := os.ReadFile(resolved)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, 0o644)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// behavioralCorpusSpec handles the committed, in-repo behavioral corpus
// at corpus/behavioral/src (D-03). Uses a two-hop filepath.Dir walk from
// gocapture's own location under testdata/golden/gocapture to reach the
// repo root, resolving to corpus/behavioral/src.
func behavioralCorpusSpec(repoRoot string) corpusSpec {
	src := filepath.Join(repoRoot, "corpus", "behavioral", "src")
	return corpusSpec{
		name: "behavioral",
		outDir: func() string {
			return filepath.Join(repoRoot, "corpus", "behavioral")
		},
		resolveSource: func() (string, error) {
			info, err := os.Stat(src)
			if err != nil || !info.IsDir() {
				return "", fmt.Errorf("behavioral corpus source not found at %s: %w", src, err)
			}
			return src, nil
		},
		multiSymbol: "Validate",
		multiQuery:  "user account",
	}
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}