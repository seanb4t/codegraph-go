// Command gocapture regenerates the Go-side EXPECTED fixtures (F5,
// 01-RESEARCH.md §A) by running the CURRENT Go indexer + query.Engine
// pipeline (plans 03-16: the full H1-H21 explore pipeline, D-09's 9-kind
// RANK_EDGES set, NODE-01/02 multi-def enumeration) against the golden
// corpora, and writing its output as go-*.json fixtures alongside the
// FROZEN TS 1.3.1 oracle fixtures under testdata/golden/corpus/<name>/.
//
// This is NOT a converter and does NOT touch the TS-side oracle
// (explore.json/node.json/explore-multi.json/node-multi.json/
// explore-mcp.json/node-mcp.json remain byte-for-byte what capture.sh
// captured from the live TS 1.3.1 install in plan 01 — that install is
// now gone, per this phase's own frozen-ground-truth policy documented in
// README.md's "Re-running the capture" section). go-*.json is Go's OWN
// output: a regression baseline proving F5 is closed (the committed
// explore/node fixtures went stale the moment the D-09 re-index (01-15)
// landed under the OLD pre-pipeline-wiring Go code) and a snapshot future
// re-runs of this tool can diff against to catch unintended behavior
// drift in the ported pipeline itself.
//
// Each corpus is indexed fresh into a throwaway temp Pebble store (never
// mutating the corpus's own checkout/.codegraph/ — mirrors
// testdata/golden/golden_parity_test.go's buildWeftEngine pattern), then
// driven with the SAME symbol/query parameters capture.sh uses for that
// corpus (see README.md's "Per-corpus queries/symbols" table), so the
// Go-side fixtures are a same-input, same-corpus counterpart to the TS
// oracle the parity harness (golden_parity_test.go) diffs against.
//
// weft-go and colbymchenry-codegraph's source trees are not committed to
// this repo (see README.md's Corpus table) — weft-go resolves via
// WEFT_REPO (default: the local sibling checkout capture.sh also uses),
// and colbymchenry-codegraph is cloned fresh to a temp dir, mirroring
// capture.sh's own pattern. Either is skipped (with a clear warning, not
// a silent no-op) if unavailable, so this tool degrades gracefully on a
// machine without network access or the local weft checkout — exactly
// like capture.sh itself already tolerates for its own TS-side capture.
//
// Usage: go run ./testdata/golden/gocapture
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// goldenCapture mirrors capture.sh's wrap_text envelope shape
// ({"command": ..., "output": ...}), so go-*.json fixtures are
// structurally identical to their TS-side siblings and can be loaded with
// the exact same loadGoldenOutput helper golden_parity_test.go already
// has.
type goldenCapture struct {
	Command string `json:"command"`
	Output  string `json:"output"`
}

// corpusSpec is one corpus's regeneration recipe: how to resolve its
// source tree, plus the same symbol/query parameters capture.sh's
// capture_repo (baseline)/capture_behavioral (multi) use for it. An empty
// baselineSymbol means "this corpus has no capture_repo baseline" (only
// synthetic-parity, per README.md: "behavioral-only, no baseline
// status/query/callers/callees/impact/explore/node fixtures").
type corpusSpec struct {
	name string

	// resolveSource returns the corpus's indexable source directory, or
	// ("", reason) if it cannot be resolved on this machine — reason is
	// printed as a skip warning, never a hard failure (mirrors
	// golden_parity_test.go's resolveWeftCorpus t.Skip pattern, adapted
	// to a standalone tool that has no *testing.T to skip with).
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
		weftGoSpec(),
		colbymchenryCodegraphSpec(),
		syntheticParitySpec(goldenDir),
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

	// The two "multi" invocations are best-effort/independent: a symbol
	// TS's extractor resolves (e.g. colbymchenry-codegraph's "resolve",
	// AD-02 in golden_parity_test.go) may not resolve under Go's own
	// extraction-coverage scope (an already-documented, real gap — see
	// AD-02's doc comment there). One failing must not silently discard
	// the OTHER artifact that already succeeded, nor silently mask the
	// gap — it is warned loudly and folded into this corpus's overall
	// error return (main() still exits non-zero), just not fatal to the
	// sibling artifact.
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
		nodeErr := fmt.Errorf("Node(%q, \"\"): %w (see AD-02 in golden_parity_test.go if this is colbymchenry-codegraph's \"resolve\" — a documented TS/JS object-literal-method extraction-coverage gap, not a bug this tool should paper over)", spec.multiSymbol, err)
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

// weftGoSpec mirrors capture.sh's WEFT_REPO default and its
// capture_repo("weft-go", ...)/capture_behavioral("weft-go", ...)
// parameters (README.md's per-corpus table).
func weftGoSpec() corpusSpec {
	return corpusSpec{
		name: "weft-go",
		resolveSource: func() (string, string) {
			repo := os.Getenv("WEFT_REPO")
			if repo == "" {
				repo = "/Volumes/Code/github.com/seanb4t/weft"
			}
			info, err := os.Stat(repo)
			if err != nil || !info.IsDir() {
				return "", fmt.Sprintf("WEFT_REPO not found at %s (set WEFT_REPO=/path/to/weft to regenerate this corpus's go-*.json fixtures)", repo)
			}
			return repo, ""
		},
		baselineSymbol:     "mergeStyle",
		baselineSymbolFile: "internal/cli/finish.go",
		baselineQuery:      "main function",
		multiSymbol:        "Run",
		multiQuery:         "epic worktree",
	}
}

// colbymchenryCodegraphSpec mirrors capture.sh's fresh-clone-to-temp-dir
// pattern for the TS CodeGraph repo itself (its source is never committed
// to this repo — README.md's Corpus table).
func colbymchenryCodegraphSpec() corpusSpec {
	return corpusSpec{
		name: "colbymchenry-codegraph",
		resolveSource: func() (string, string) {
			tmpRoot, err := os.MkdirTemp("", "gocapture-colbymchenry-*")
			if err != nil {
				return "", fmt.Sprintf("MkdirTemp: %v", err)
			}
			cmd := exec.Command("git", "clone", "--depth", "1", "--quiet",
				"https://github.com/colbymchenry/codegraph.git", tmpRoot)
			if out, err := cmd.CombinedOutput(); err != nil {
				os.RemoveAll(tmpRoot)
				return "", fmt.Sprintf("git clone failed (no network access?): %v: %s", err, string(out))
			}
			return tmpRoot, ""
		},
		baselineSymbol:     "searchNodes",
		baselineSymbolFile: "src/db/queries.ts",
		baselineQuery:      "search nodes",
		multiSymbol:        "resolve",
		multiQuery:         "generated file detection",
	}
}

// syntheticParitySpec mirrors capture.sh's synthetic-parity handling:
// committed in-repo source, behavioral-only (no capture_repo baseline —
// baselineSymbol left empty).
func syntheticParitySpec(goldenDir string) corpusSpec {
	src := filepath.Join(goldenDir, "corpus", "synthetic-parity", "src")
	return corpusSpec{
		name: "synthetic-parity",
		resolveSource: func() (string, string) {
			info, err := os.Stat(src)
			if err != nil || !info.IsDir() {
				return "", fmt.Sprintf("synthetic-parity source not found at %s", src)
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
