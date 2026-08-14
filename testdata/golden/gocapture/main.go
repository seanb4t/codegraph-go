// Command gocapture regenerates the Go-side EXPECTED fixtures (F5,
// 01-RESEARCH.md) by running the CURRENT Go indexer + query.Engine
// pipeline against the behavioral corpus, writing its output as go-*.json
// fixtures under corpus/behavioral/.
//
// The TS-era capture path and external corpora are retired as of this
// phase (FIXT-04). gocapture now targets only the committed, always-in-repo
// behavioral corpus (D-03).
//
// Usage: go run ./testdata/golden/gocapture
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
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

// corpusSpec is one corpus's regeneration recipe: how to resolve its
// source tree, plus the symbol/query parameters for its behavioral
// fixtures. An empty baselineSymbol means "this corpus has no baseline"
// (the behavioral corpus is behavioral-only, with no baseline
// status/query/callers/callees/impact/explore/node fixtures).
type corpusSpec struct {
	name string

	// resolveSource returns the corpus's indexable source directory, or
	// ("", reason) if it cannot be resolved on this machine — reason is
	// printed as a skip warning, never a hard failure (mirrors the test
	// harness's t.Skip discipline, adapted to a standalone tool that has
	// no *testing.T to skip with).
	resolveSource func() (string, string)

	// baseline mirrors capture_repo's explore/node invocation
	// (--max-files 1 / -f <file>) — empty baselineSymbol skips it.
	baselineSymbol     string
	baselineSymbolFile string
	baselineQuery      string

	// behavioral mirrors capture_behavioral's invocation (no
	// --max-files, no -f).
	multiSymbol string
	multiQuery  string
}

func main() {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		fatal("gocapture: runtime.Caller(0) failed to resolve this file's own path")
	}
	goldenDir := filepath.Dir(filepath.Dir(thisFile)) // .../testdata/golden
	corpusDir := filepath.Join(goldenDir, "corpus")

	specs := []corpusSpec{
		behavioralCorpusSpec(goldenDir),
	}

	failures := 0
	for _, spec := range specs {
		if err := regenerateCorpus(spec, corpusDir); err != nil {
			fmt.Fprintf(os.Stderr, "gocapture: [%s] FAILED: %v\n", spec.name, err)
			failures++
		}
	}
	if failures > 0 {
		os.Exit(1)
	}
}

func regenerateCorpus(spec corpusSpec, corpusDir string) error {
	sourcePath, skipReason := spec.resolveSource()
	if sourcePath == "" {
		fmt.Fprintf(os.Stderr, "gocapture: [%s] SKIPPED: %s\n", spec.name, skipReason)
		return nil
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
	out := filepath.Join(corpusDir, spec.name)
	if err := os.MkdirAll(out, 0o755); err != nil {
		return fmt.Errorf("MkdirAll(%s): %w", out, err)
	}

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

	// The two "multi" invocations are best-effort/independent: one failing
	// must not silently discard the OTHER artifact that already succeeded,
	// nor silently mask the gap — it is warned loudly and folded into this
	// corpus's overall error return (main() still exits non-zero), just
	// not fatal to the sibling artifact.
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

	fmt.Fprintf(os.Stderr, "gocapture: [%s] wrote go-*.json fixtures to %s\n", spec.name, out)
	return multiErr
}

func writeCapture(path, command, output string) error {
	data, err := json.MarshalIndent(goldenCapture{Command: command, Output: output}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// behavioralCorpusSpec handles the committed, in-repo behavioral corpus
// at corpus/behavioral/src (D-03). Uses a two-hop filepath.Dir walk from
// gocapture's own location under testdata/golden/gocapture to reach the
// repo root, resolving to corpus/behavioral/src.
func behavioralCorpusSpec(goldenDir string) corpusSpec {
	repoRoot := filepath.Dir(filepath.Dir(goldenDir)) // testdata/golden -> testdata -> repo root
	src := filepath.Join(repoRoot, "corpus", "behavioral", "src")
	return corpusSpec{
		name: "behavioral",
		resolveSource: func() (string, string) {
			info, err := os.Stat(src)
			if err != nil || !info.IsDir() {
				return "", fmt.Sprintf("behavioral corpus source not found at %s", src)
			}
			return src, ""
		},
		multiSymbol: "Validate",
		multiQuery:  "user account",
	}
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
