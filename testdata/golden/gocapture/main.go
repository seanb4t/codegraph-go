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
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seanb4t/codegraph-go/internal/corpora"
	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	internalmcp "github.com/seanb4t/codegraph-go/internal/mcp"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// goldenCapture mirrors the wrap_text envelope shape
// ({"command": ..., "output": ...}), so go-*.json fixtures are
// structurally identical to their siblings and can be loaded with
// the loadGoldenOutputIn helper.
type goldenCapture struct {
	Command string `json:"command"`
	Output  string `json:"output"`
}

// languageToLockedSlug is the EXPLICIT committed language->locked-slug map (H3).
// hugo supplies the tsjs leg from its JS files even though its manifest
// language is "go". The map is shared by the gocapture resolver, the hermetic
// test resolver (lockedCorpusDir), and the completeness guard.
var languageToLockedSlug = map[string]string{
	"go":     "hugo",
	"tsjs":   "hugo",
	"java":   "guava",
	"csharp": "serilog",
	"python": "requests",
}

// slugToRepo maps each locked-slug to its manifest repo slug for lookup.
var slugToRepo = map[string]string{
	"hugo":     "gohugoio/hugo",
	"guava":    "google/guava",
	"serilog":  "serilog/serilog",
	"requests": "psf/requests",
}

// perCorpusArgs holds the symbol/query parameters for one locked corpus.
type perCorpusArgs struct {
	baselineSymbol     string
	baselineSymbolFile string
	baselineQuery      string
	multiSymbol        string
	multiQuery         string
}

// lockedCorpusArgs defines the committed symbol/query values per locked corpus.
// These produce the expected golden set per corpus: {explore, node, explore-multi,
// node-multi, explore-mcp, node-mcp} — 6 goldens.
var lockedCorpusArgs = map[string]perCorpusArgs{
	"hugo": {
		baselineSymbol:     "Page",
		baselineSymbolFile: "",
		baselineQuery:      "page content",
		multiSymbol:        "Site",
		multiQuery:         "page content template",
	},
	"guava": {
		baselineSymbol:     "Preconditions",
		baselineSymbolFile: "",
		baselineQuery:      "check precondition",
		multiSymbol:        "ImmutableList",
		multiQuery:         "immutable collection",
	},
	"serilog": {
		baselineSymbol:     "LoggerConfiguration",
		baselineSymbolFile: "",
		baselineQuery:      "configure logger",
		multiSymbol:        "ILogger",
		multiQuery:         "log configuration",
	},
	"requests": {
		baselineSymbol:     "Session",
		baselineSymbolFile: "",
		baselineQuery:      "http session",
		multiSymbol:        "Request",
		multiQuery:         "http request session",
	},
}

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
		repo := slugToRepo[slug]
		e, ok := lockedByRepo[repo]
		if !ok {
			fatal(fmt.Sprintf("gocapture: locked entry %q (slug=%q) not found in manifest", repo, slug))
		}
		args := lockedCorpusArgs[slug]
		out := filepath.Join(goldenDir, "corpus", slug)
		specs = append(specs, corpusSpec{
			name: slug,
			outDir: func() string {
				return out
			},
			resolveSource: func() (string, error) {
				return e.Dir(corpusRoot), nil
			},
			baselineSymbol:     args.baselineSymbol,
			baselineSymbolFile: args.baselineSymbolFile,
			baselineQuery:      args.baselineQuery,
			multiSymbol:        args.multiSymbol,
			multiQuery:         args.multiQuery,
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
		mcpExploreOut, err := callExploreViaMCP(mcpTmp, spec.baselineQuery)
		if err != nil {
			return fmt.Errorf("MCP Explore(%q): %w", spec.baselineQuery, err)
		}
		if err := writeCapture(filepath.Join(out, "go-explore-mcp.json"),
			fmt.Sprintf("MCP explore %q -p %s", spec.baselineQuery, spec.name), mcpExploreOut); err != nil {
			return err
		}

		mcpNodeOut, err := callNodeViaMCP(mcpTmp, spec.baselineSymbol)
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
	data, err := json.MarshalIndent(goldenCapture{Command: command, Output: output}, "", "  ")
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
// so the destination starts clean for indexing.
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

// --- MCP helpers ---

// callExploreViaMCP drives codegraph_explore through a real, in-process MCP
// server (internalmcp.BuildServer) over in-memory transports.
func callExploreViaMCP(repoDir, query string) (string, error) {
	s := internalmcp.BuildServer(true, map[string]bool{}, repoDir, repoDir)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go func() {
		_ = s.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "codegraph-gocapture", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return "", fmt.Errorf("client Connect: %w", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "codegraph_explore",
		Arguments: map[string]any{"query": query},
	})
	if err != nil {
		return "", fmt.Errorf("CallTool codegraph_explore(%q): %w", query, err)
	}
	if result.IsError {
		return "", fmt.Errorf("codegraph_explore(%q) returned an error result", query)
	}
	return mcpResultText(result)
}

// callNodeViaMCP drives codegraph_node the same way callExploreViaMCP drives
// codegraph_explore.
func callNodeViaMCP(repoDir, symbol string) (string, error) {
	s := internalmcp.BuildServer(true, map[string]bool{"node": true}, repoDir, repoDir)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go func() {
		_ = s.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "codegraph-gocapture", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return "", fmt.Errorf("client Connect: %w", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "codegraph_node",
		Arguments: map[string]any{"symbol": symbol},
	})
	if err != nil {
		return "", fmt.Errorf("CallTool codegraph_node(%q): %w", symbol, err)
	}
	if result.IsError {
		return "", fmt.Errorf("codegraph_node(%q) returned an error result", symbol)
	}
	return mcpResultText(result)
}

// mcpResultText extracts the first text content block from a successful
// CallTool result.
func mcpResultText(result *mcp.CallToolResult) (string, error) {
	if len(result.Content) == 0 {
		return "", fmt.Errorf("CallTool result has no content")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		return "", fmt.Errorf("CallTool result content[0] is not text: %+v", result.Content[0])
	}
	return text.Text, nil
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}