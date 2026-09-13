// Command graphcluster is GRF-09's measurement instrument: it reads every
// bar it judges against from the committed threshold file — never
// carrying its own defaults — opens the pinned corpus's already-indexed
// store READ-ONLY (graphstore.Open + Snapshot; never indexer.Run, never a
// Writer), times query.AssignCommunities in isolation over a fresh
// FileGraph() rollup per run, and writes exactly one observation file
// recording the verdict verbatim. It never writes the threshold file
// itself.
//
// Exit codes: 0 = PASS, 1 = FAIL, 2 = refused or error (a degenerate
// corpus, a malformed threshold, or a resolution failure — no observation
// is written in either case).
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/seanb4t/codegraph-go/internal/corpora"
	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// errRefused is the sentinel wrapped by measureRuns when the resolved
// corpus has fewer than the threshold's minNodes file nodes — a
// degenerate rollup can never read as a meaningful PASS.
var errRefused = errors.New("refused")

// threshold decodes the committed GRF-09 pass condition. Unknown keys
// (purpose, minNodesNote, recordedNonBinding, measurementProtocol,
// onFailure, lockedBy, prohibitions, ...) are ignored — the threshold file
// carries prose this loader does not need.
type threshold struct {
	SchemaVersion int `json:"schemaVersion"`
	Corpus        struct {
		Repo string `json:"repo"`
		SHA  string `json:"sha"`
	} `json:"corpus"`
	Metrics struct {
		ClusteringTimeMs struct {
			Max       int64  `json:"max"`
			Unit      string `json:"unit"`
			Statistic string `json:"statistic"`
			Runs      int    `json:"runs"`
		} `json:"clusteringTimeMs"`
	} `json:"metrics"`
	MinNodes int `json:"minNodes"`
}

// loadThreshold reads path, decodes it, and refuses with a named error
// when any bar this harness judges against is missing or malformed. It
// returns the raw bytes alongside the decoded value so the caller can
// compute thresholdDigest over the exact bytes read.
func loadThreshold(path string) (threshold, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return threshold{}, nil, fmt.Errorf("threshold: read %s: %w", path, err)
	}
	var t threshold
	if err := json.Unmarshal(raw, &t); err != nil {
		return threshold{}, nil, fmt.Errorf("threshold: decode %s: %w", path, err)
	}
	if t.Metrics.ClusteringTimeMs.Max <= 0 {
		return threshold{}, nil, fmt.Errorf("threshold: metrics.clusteringTimeMs.max missing or not positive")
	}
	if t.Metrics.ClusteringTimeMs.Statistic != "median" {
		return threshold{}, nil, fmt.Errorf("threshold: metrics.clusteringTimeMs.statistic must be %q (got %q)", "median", t.Metrics.ClusteringTimeMs.Statistic)
	}
	if runs := t.Metrics.ClusteringTimeMs.Runs; runs < 3 || runs%2 == 0 {
		return threshold{}, nil, fmt.Errorf("threshold: metrics.clusteringTimeMs.runs must be an odd integer >= 3 (got %d)", runs)
	}
	if t.MinNodes < 2 {
		return threshold{}, nil, fmt.Errorf("threshold: minNodes must be >= 2 (got %d)", t.MinNodes)
	}
	if t.Corpus.Repo == "" || t.Corpus.SHA == "" {
		return threshold{}, nil, fmt.Errorf("threshold: corpus.repo and corpus.sha must both be set")
	}
	return t, raw, nil
}

// thresholdDigest returns a stable "sha256:<hex>" digest over raw — the
// exact bytes read from the threshold file — so a later edit to that file
// is detectable by comparing against a committed observation's own
// thresholdDigest.
func thresholdDigest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// runGit runs git -C dir <args...> and returns its stdout, mirroring
// tools/corpora/main.go's runGit shape.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// resolveCorpusStore finds the manifest entry pinned to (repo, sha),
// verifies its checkout's HEAD equals sha, and returns the path to its
// already-indexed, read-only store directory. It never re-indexes and
// never opens the store itself — that is measureRuns' job.
func resolveCorpusStore(manifestPath, corpusRoot, repo, sha string) (string, error) {
	m, err := corpora.Load(manifestPath)
	if err != nil {
		return "", fmt.Errorf("resolveCorpusStore: %w", err)
	}

	var entry corpora.Entry
	found := false
	for _, e := range m.Corpora {
		if e.Repo == repo && e.SHA == sha {
			entry = e
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("resolveCorpusStore: no manifest entry for repo %q sha %q", repo, sha)
	}

	root := corpusRoot
	if root == "" {
		root, err = corpora.CorpusRoot()
		if err != nil {
			return "", fmt.Errorf("resolveCorpusStore: %w", err)
		}
	}

	dir := entry.Dir(root)
	head, err := runGit(dir, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("resolveCorpusStore: git rev-parse HEAD at %s: %w", dir, err)
	}
	head = strings.TrimSpace(head)
	if head != sha {
		return "", fmt.Errorf("resolveCorpusStore: checkout at %s has HEAD %q, want %q", dir, head, sha)
	}

	storeDir := filepath.Join(dir, ".codegraph", "store")
	if _, err := os.Stat(storeDir); err != nil {
		return "", fmt.Errorf("resolveCorpusStore: stat %s: %w", storeDir, err)
	}
	return storeDir, nil
}

// runResult is one cold measurement run's timings.
type runResult struct {
	Run              int   `json:"run"`
	ClusteringTimeMs int64 `json:"clusteringTimeMs"`
	FileGraphWallMs  int64 `json:"fileGraphWallMs"`
}

// measureRuns opens storeDir READ-ONLY exactly once, then performs runs
// cold measurements: a fresh Snapshot and a fresh FileGraph() rollup per
// run (no cache, no warm-up call excluded), timing only the
// query.AssignCommunities call in isolation. It refuses (wrapping
// errRefused) the first time a run's rollup resolves to fewer than
// minNodes file nodes — a degenerate rollup can never read as a
// meaningful PASS. It also returns the last run's node count, edge count,
// and distinct community count (recorded non-binding).
func measureRuns(storeDir string, runs, minNodes int) ([]runResult, int, int, int, error) {
	store, err := graphstore.Open(storeDir)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("graphstore.Open: %w", err)
	}
	defer store.Close()

	results := make([]runResult, 0, runs)
	var nodeCount, edgeCount, communityCount int
	for i := 0; i < runs; i++ {
		reader, err := store.Snapshot()
		if err != nil {
			return nil, 0, 0, 0, fmt.Errorf("store.Snapshot: %w", err)
		}

		t0 := time.Now()
		res, ferr := query.New(reader).FileGraph()
		wall := time.Since(t0)
		if ferr != nil {
			reader.Close()
			return nil, 0, 0, 0, fmt.Errorf("FileGraph: %w", ferr)
		}
		if len(res.Nodes) < minNodes {
			reader.Close()
			return nil, 0, 0, 0, fmt.Errorf("%w: corpus resolved to %d file nodes (< minNodes %d)", errRefused, len(res.Nodes), minNodes)
		}

		t1 := time.Now()
		ids := query.AssignCommunities(res.Nodes, res.Edges)
		clustering := time.Since(t1)
		reader.Close()

		results = append(results, runResult{
			Run:              i + 1,
			ClusteringTimeMs: clustering.Milliseconds(),
			FileGraphWallMs:  wall.Milliseconds(),
		})
		nodeCount = len(res.Nodes)
		edgeCount = len(res.Edges)
		distinct := make(map[int]struct{}, len(ids))
		for _, id := range ids {
			distinct[id] = struct{}{}
		}
		communityCount = len(distinct)
	}
	return results, nodeCount, edgeCount, communityCount, nil
}

// measureRunsFn is the test seam: production code always calls it, tests
// swap it (restored via t.Cleanup) to exercise run's observation-writing
// and verdict logic without opening a real store.
var measureRunsFn = measureRuns

// medianInt64 returns the sorted MIDDLE value of vals — the binding
// GRF-09 statistic, never an average — over a sorted COPY, so the
// caller's slice (and its run-order) is never mutated.
func medianInt64(vals []int64) int64 {
	sorted := make([]int64, len(vals))
	copy(sorted, vals)
	slices.Sort(sorted)
	return sorted[len(sorted)/2]
}

// judge returns "PASS" when median <= max, else "FAIL". Equality passes.
// Int64 only — no float anywhere in the verdict path.
func judge(median, max int64) string {
	if median <= max {
		return "PASS"
	}
	return "FAIL"
}

// corpusRef names the corpus a measurement ran against.
type corpusRef struct {
	Repo string `json:"repo"`
	SHA  string `json:"sha"`
}

// clusteringTimeSummary is the observation's binding-metric block.
type clusteringTimeSummary struct {
	Values     []int64 `json:"values"`
	Median     int64   `json:"median"`
	Max        int64   `json:"max"`
	Unit       string  `json:"unit"`
	Statistic  string  `json:"statistic"`
	Comparison string  `json:"comparison"`
	Verdict    string  `json:"verdict"`
}

// observation is the committed measurement record.
type observation struct {
	SchemaVersion    int                   `json:"schemaVersion"`
	ThresholdRef     string                `json:"thresholdRef"`
	ThresholdDigest  string                `json:"thresholdDigest"`
	Corpus           corpusRef             `json:"corpus"`
	StoreSource      string                `json:"storeSource"`
	StoreRelative    string                `json:"storeRelative"`
	NodeCount        int                   `json:"nodeCount"`
	EdgeCount        int                   `json:"edgeCount"`
	CommunityCount   int                   `json:"communityCount"`
	Runs             []runResult           `json:"runs"`
	ClusteringTimeMs clusteringTimeSummary `json:"clusteringTimeMs"`
	Verdict          string                `json:"verdict"`
	MeasuredAt       string                `json:"measuredAt"`
	GoVersion        string                `json:"goVersion"`
	GOOS             string                `json:"goos"`
	GOARCH           string                `json:"goarch"`
}

// writeObservation marshals obs with stable, readable indentation and
// writes it to path with a trailing newline. This is the harness's ONLY
// write path — it never writes the threshold file.
func writeObservation(path string, obs observation) error {
	data, err := json.MarshalIndent(obs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal observation: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// run is main's testable core: it never calls os.Exit itself.
func run(args []string, stdout, stderr io.Writer) int {
	flag := flag.NewFlagSet("graphcluster", flag.ContinueOnError)
	flag.SetOutput(stderr)
	thresholdPath := flag.String("threshold", "corpora/graph-cluster-threshold.json", "threshold JSON file this harness reads every bar from — never carries its own defaults")
	outPath := flag.String("out", "corpora/graph-cluster-observations.json", "observation file this harness writes")
	manifestPath := flag.String("manifest", "corpora/manifest.json", "corpus manifest resolving the pinned repo/sha to a cached store directory")
	corpusRoot := flag.String("corpus-root", "", "override the resolved corpus root (default: corpora.CorpusRoot())")
	storeDirFlag := flag.String("store-dir", "", "explicit store directory (tests only) — skips manifest/corpus-root resolution and the checkout SHA check; records storeSource \"explicit\"")
	if err := flag.Parse(args); err != nil {
		return 2
	}

	thr, raw, err := loadThreshold(*thresholdPath)
	if err != nil {
		fmt.Fprintf(stderr, "graphcluster: %v\n", err)
		return 2
	}

	var storeDir, storeSource string
	if *storeDirFlag != "" {
		storeDir = *storeDirFlag
		storeSource = "explicit"
	} else {
		storeDir, err = resolveCorpusStore(*manifestPath, *corpusRoot, thr.Corpus.Repo, thr.Corpus.SHA)
		if err != nil {
			fmt.Fprintf(stderr, "graphcluster: %v\n", err)
			return 2
		}
		storeSource = "manifest"
	}

	runResults, nodeCount, edgeCount, communityCount, err := measureRunsFn(storeDir, thr.Metrics.ClusteringTimeMs.Runs, thr.MinNodes)
	if err != nil {
		if errors.Is(err, errRefused) {
			fmt.Fprintf(stderr, "graphcluster: %v\n", err)
			return 2
		}
		fmt.Fprintf(stderr, "graphcluster: measure: %v\n", err)
		return 2
	}

	values := make([]int64, len(runResults))
	for i, r := range runResults {
		values[i] = r.ClusteringTimeMs
	}
	median := medianInt64(values)
	max := thr.Metrics.ClusteringTimeMs.Max
	verdict := judge(median, max)

	obs := observation{
		SchemaVersion:   1,
		ThresholdRef:    *thresholdPath,
		ThresholdDigest: thresholdDigest(raw),
		Corpus:          corpusRef{Repo: thr.Corpus.Repo, SHA: thr.Corpus.SHA},
		StoreSource:     storeSource,
		StoreRelative:   ".codegraph/store",
		NodeCount:       nodeCount,
		EdgeCount:       edgeCount,
		CommunityCount:  communityCount,
		Runs:            runResults,
		ClusteringTimeMs: clusteringTimeSummary{
			Values:     values,
			Median:     median,
			Max:        max,
			Unit:       "ms",
			Statistic:  "median",
			Comparison: "median <= max",
			Verdict:    verdict,
		},
		Verdict:    verdict,
		MeasuredAt: time.Now().UTC().Format(time.RFC3339),
		GoVersion:  runtime.Version(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}

	if err := writeObservation(*outPath, obs); err != nil {
		fmt.Fprintf(stderr, "graphcluster: write observation: %v\n", err)
		return 2
	}

	fmt.Fprintf(stdout, "graphcluster: verdict %s — median %d ms vs max %d ms over %d runs %v; %d nodes, %d edges, %d communities\n",
		verdict, median, max, len(runResults), values, nodeCount, edgeCount, communityCount)

	if verdict == "FAIL" {
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
