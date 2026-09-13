package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/schema"
)

// committedThresholdPath is the real, committed GRF-09 threshold this test
// binary reads from — relative to this package's own directory
// (tools/graphcluster), never a copy or a fixture, so the loader's
// contract is proven against the actual artifact tools/graphcluster
// measures against in Task 2.
const committedThresholdPath = "../../corpora/graph-cluster-threshold.json"

// TestJudgeBoundary: judge(median, max) returns PASS for median <= max and
// FAIL for median > max — equality passes.
func TestJudgeBoundary(t *testing.T) {
	cases := []struct {
		median, max int64
		want        string
	}{
		{499, 500, "PASS"},
		{500, 500, "PASS"},
		{501, 500, "FAIL"},
	}
	for _, c := range cases {
		got := judge(c.median, c.max)
		t.Logf("judge(%d, %d) = %s", c.median, c.max, got)
		if got != c.want {
			t.Errorf("judge(%d, %d) = %s, want %s", c.median, c.max, got, c.want)
		}
	}
}

// equalInt64 reports whether a and b hold the same values in the same
// order — used to prove medianInt64 does not mutate its input slice.
func equalInt64(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestMedianInt64IsTheSortedMiddle: the binding statistic is the sorted
// MIDDLE of the input values, never an average, and the input slice is
// never mutated in place.
func TestMedianInt64IsTheSortedMiddle(t *testing.T) {
	in1 := []int64{1, 500, 1000}
	cp1 := append([]int64(nil), in1...)
	if got := medianInt64(in1); got != 500 {
		t.Errorf("medianInt64(%v) = %d, want 500", cp1, got)
	}
	if !equalInt64(in1, cp1) {
		t.Errorf("medianInt64 mutated its input: got %v, want %v", in1, cp1)
	}

	in2 := []int64{1, 2, 1000}
	got2 := medianInt64(in2)
	if got2 != 2 {
		t.Errorf("medianInt64(%v) = %d, want 2", in2, got2)
	}
	if got2 == 334 {
		t.Errorf("medianInt64(%v) = %d looks like an AVERAGE (334), not a median", in2, got2)
	}

	in3 := []int64{900, 100, 400}
	if got := medianInt64(in3); got != 400 {
		t.Errorf("medianInt64(%v) = %d, want 400", in3, got)
	}
}

// writeTempThresholdJSON marshals obj as JSON into dir/threshold.json and
// returns its path.
func writeTempThresholdJSON(t *testing.T, dir string, obj map[string]any) string {
	t.Helper()
	path := filepath.Join(dir, "threshold.json")
	data, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("marshal temp threshold: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write temp threshold: %v", err)
	}
	return path
}

// baseThresholdObj returns a well-formed threshold document (mirroring the
// committed one's shape) that callers mutate to omit or corrupt exactly
// one bar per sub-case.
func baseThresholdObj() map[string]any {
	return map[string]any{
		"schemaVersion": 1,
		"corpus": map[string]any{
			"repo": "google/guava",
			"sha":  "94f39958baf7ad51ddf9c70e406ed6b188194daa",
		},
		"metrics": map[string]any{
			"clusteringTimeMs": map[string]any{
				"max":       500,
				"unit":      "ms",
				"statistic": "median",
				"runs":      3,
			},
		},
		"minNodes": 2,
	}
}

// TestLoadThresholdRefusesMissingBar: loadThreshold refuses (named error)
// when any bar it reads is missing or malformed, and loads the real,
// committed threshold cleanly.
func TestLoadThresholdRefusesMissingBar(t *testing.T) {
	dir := t.TempDir()

	missingMax := baseThresholdObj()
	missingMax["metrics"] = map[string]any{"clusteringTimeMs": map[string]any{"unit": "ms", "statistic": "median", "runs": 3}}
	if _, _, err := loadThreshold(writeTempThresholdJSON(t, dir, missingMax)); err == nil || !strings.Contains(err.Error(), "clusteringTimeMs.max") {
		t.Errorf("missing max: err = %v, want containing %q", err, "clusteringTimeMs.max")
	}

	badRuns := baseThresholdObj()
	badRuns["metrics"] = map[string]any{"clusteringTimeMs": map[string]any{"max": 500, "unit": "ms", "statistic": "median", "runs": 2}}
	if _, _, err := loadThreshold(writeTempThresholdJSON(t, dir, badRuns)); err == nil || !strings.Contains(err.Error(), "runs") {
		t.Errorf("runs=2: err = %v, want containing %q", err, "runs")
	}

	badStatistic := baseThresholdObj()
	badStatistic["metrics"] = map[string]any{"clusteringTimeMs": map[string]any{"max": 500, "unit": "ms", "statistic": "mean", "runs": 3}}
	if _, _, err := loadThreshold(writeTempThresholdJSON(t, dir, badStatistic)); err == nil || !strings.Contains(err.Error(), "statistic") {
		t.Errorf("statistic=mean: err = %v, want containing %q", err, "statistic")
	}

	missingMinNodes := baseThresholdObj()
	delete(missingMinNodes, "minNodes")
	if _, _, err := loadThreshold(writeTempThresholdJSON(t, dir, missingMinNodes)); err == nil || !strings.Contains(err.Error(), "minNodes") {
		t.Errorf("missing minNodes: err = %v, want containing %q", err, "minNodes")
	}

	thr, raw, err := loadThreshold(committedThresholdPath)
	if err != nil {
		t.Fatalf("loadThreshold(committed): %v", err)
	}
	if thr.Metrics.ClusteringTimeMs.Max != 500 {
		t.Errorf("committed threshold: Max = %d, want 500", thr.Metrics.ClusteringTimeMs.Max)
	}
	if thr.Metrics.ClusteringTimeMs.Runs != 3 {
		t.Errorf("committed threshold: Runs = %d, want 3", thr.Metrics.ClusteringTimeMs.Runs)
	}
	if thr.MinNodes != 2 {
		t.Errorf("committed threshold: MinNodes = %d, want 2", thr.MinNodes)
	}
	if thr.Corpus.SHA != "94f39958baf7ad51ddf9c70e406ed6b188194daa" {
		t.Errorf("committed threshold: Corpus.SHA = %q, want the pinned guava sha", thr.Corpus.SHA)
	}
	digest := thresholdDigest(raw)
	if !strings.HasPrefix(digest, "sha256:") {
		t.Errorf("thresholdDigest = %q, want a sha256: prefix", digest)
	}
	if len(digest) != 71 {
		t.Errorf("thresholdDigest length = %d, want 71 (\"sha256:\" + 64 hex chars)", len(digest))
	}
}

// newTestStore opens a fresh Pebble store in a temp dir, writes nodes and
// edges through the real Writer/Commit path, closes it, and returns the
// store's directory — the harness itself opens it fresh via -store-dir.
func newTestStore(t *testing.T, nodes []*schema.Node, edges []*schema.Edge) string {
	t.Helper()
	dir := t.TempDir()
	store, err := graphstore.Open(dir)
	if err != nil {
		t.Fatalf("graphstore.Open: %v", err)
	}
	w, err := store.NewWriter()
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	for _, n := range nodes {
		if err := w.PutNode(n); err != nil {
			t.Fatalf("PutNode(%s): %v", n.Id, err)
		}
	}
	for _, e := range edges {
		if err := w.PutEdge(e, ""); err != nil {
			t.Fatalf("PutEdge(%+v): %v", e, err)
		}
	}
	if err := w.Commit(); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("store.Close: %v", err)
	}
	return dir
}

// TestRunRefusesFewerThanMinNodes: a store resolving to fewer than
// minNodes file nodes makes run refuse (exit 2) and write no observation.
func TestRunRefusesFewerThanMinNodes(t *testing.T) {
	storeDir := newTestStore(t, []*schema.Node{
		{Id: "only", Kind: goextract.KindFile, FilePath: "only.go"},
	}, nil)

	thrPath := writeTempThresholdJSON(t, t.TempDir(), baseThresholdObj())
	out := filepath.Join(t.TempDir(), "observation.json")

	var stdout, stderr bytes.Buffer
	rc := run([]string{"-threshold", thrPath, "-out", out, "-store-dir", storeDir}, &stdout, &stderr)
	if rc != 2 {
		t.Fatalf("run() = %d, want 2; stderr=%s", rc, stderr.String())
	}
	if !strings.Contains(stderr.String(), "refused") {
		t.Errorf("stderr = %q, want containing %q", stderr.String(), "refused")
	}
	if !strings.Contains(stderr.String(), "1 file nodes") {
		t.Errorf("stderr = %q, want containing %q", stderr.String(), "1 file nodes")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("observation file exists at %s after refusal, want absent", out)
	}
}

// TestRunWritesObservationWithDigestAndVerdict: a well-formed store and
// threshold produce a committed-shape observation carrying the threshold
// digest, the verdict, and no host-local path.
func TestRunWritesObservationWithDigestAndVerdict(t *testing.T) {
	storeDir := newTestStore(t,
		[]*schema.Node{
			{Id: "a", Kind: goextract.KindFile, FilePath: "a.go", Language: "go"},
			{Id: "b", Kind: goextract.KindFile, FilePath: "b.go", Language: "go"},
			{Id: "c", Kind: goextract.KindFile, FilePath: "c.go", Language: "go"},
		},
		[]*schema.Edge{
			{Source: "a", Target: "b", Kind: goextract.RefKindCalls},
			{Source: "b", Target: "a", Kind: goextract.RefKindCalls},
			{Source: "a", Target: "c", Kind: goextract.RefKindCalls},
		},
	)

	thrPath := writeTempThresholdJSON(t, t.TempDir(), baseThresholdObj())
	thrRaw, err := os.ReadFile(thrPath)
	if err != nil {
		t.Fatalf("read temp threshold: %v", err)
	}
	wantDigest := thresholdDigest(thrRaw)

	out := filepath.Join(t.TempDir(), "observation.json")
	var stdout, stderr bytes.Buffer
	rc := run([]string{"-threshold", thrPath, "-out", out, "-store-dir", storeDir}, &stdout, &stderr)
	if rc != 0 {
		t.Fatalf("run() = %d, want 0; stderr=%s", rc, stderr.String())
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read observation: %v", err)
	}
	var obs observation
	if err := json.Unmarshal(data, &obs); err != nil {
		t.Fatalf("decode observation: %v", err)
	}

	if obs.Verdict != "PASS" {
		t.Errorf("verdict = %q, want PASS", obs.Verdict)
	}
	if len(obs.Runs) != 3 {
		t.Errorf("len(runs) = %d, want 3", len(obs.Runs))
	}
	if len(obs.ClusteringTimeMs.Values) == 3 {
		wantMedian := medianInt64(obs.ClusteringTimeMs.Values)
		if obs.ClusteringTimeMs.Median != wantMedian {
			t.Errorf("clusteringTimeMs.median = %d, want %d", obs.ClusteringTimeMs.Median, wantMedian)
		}
	}
	if obs.ThresholdDigest != wantDigest {
		t.Errorf("thresholdDigest = %q, want %q", obs.ThresholdDigest, wantDigest)
	}
	if obs.StoreSource != "explicit" {
		t.Errorf("storeSource = %q, want explicit", obs.StoreSource)
	}
	if obs.NodeCount != 3 {
		t.Errorf("nodeCount = %d, want 3", obs.NodeCount)
	}
	if obs.CommunityCount < 1 {
		t.Errorf("communityCount = %d, want >= 1", obs.CommunityCount)
	}
	if _, err := time.Parse(time.RFC3339, obs.MeasuredAt); err != nil {
		t.Errorf("measuredAt = %q does not parse as RFC3339: %v", obs.MeasuredAt, err)
	}
	if bytes.Contains(data, []byte("/Users/")) {
		t.Errorf("observation leaks a host-local /Users/ path: %s", data)
	}
	if bytes.Contains(data, []byte("/home/")) {
		t.Errorf("observation leaks a host-local /home/ path: %s", data)
	}
	if bytes.Contains(data, []byte(storeDir)) {
		t.Errorf("observation leaks the temp store directory path %q: %s", storeDir, data)
	}
}

// TestRunRecordsFailVerbatimAndRunsInOrder: a FAIL verdict is recorded
// verbatim (not "fail"/"FAILED"), the run values stay in RUN order (not
// sorted), and the median is the sorted middle of those run-ordered
// values.
func TestRunRecordsFailVerbatimAndRunsInOrder(t *testing.T) {
	orig := measureRunsFn
	t.Cleanup(func() { measureRunsFn = orig })
	measureRunsFn = func(storeDir string, runs, minNodes int) ([]runResult, int, int, int, error) {
		return []runResult{
			{Run: 1, ClusteringTimeMs: 900},
			{Run: 2, ClusteringTimeMs: 100},
			{Run: 3, ClusteringTimeMs: 400},
		}, 10, 20, 3, nil
	}

	thrObj := baseThresholdObj()
	thrObj["metrics"] = map[string]any{"clusteringTimeMs": map[string]any{"max": 300, "unit": "ms", "statistic": "median", "runs": 3}}
	thrPath := writeTempThresholdJSON(t, t.TempDir(), thrObj)
	out := filepath.Join(t.TempDir(), "observation.json")

	var stdout, stderr bytes.Buffer
	rc := run([]string{"-threshold", thrPath, "-out", out, "-store-dir", "unused-because-measureRunsFn-is-stubbed"}, &stdout, &stderr)
	if rc != 1 {
		t.Fatalf("run() = %d, want 1; stderr=%s", rc, stderr.String())
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read observation: %v", err)
	}
	var obs observation
	if err := json.Unmarshal(data, &obs); err != nil {
		t.Fatalf("decode observation: %v", err)
	}

	if obs.Verdict != "FAIL" {
		t.Fatalf("verdict = %q, want exactly %q (verbatim)", obs.Verdict, "FAIL")
	}
	wantValues := []int64{900, 100, 400}
	if len(obs.ClusteringTimeMs.Values) != len(wantValues) {
		t.Fatalf("values = %v, want %v", obs.ClusteringTimeMs.Values, wantValues)
	}
	for i, v := range wantValues {
		if obs.ClusteringTimeMs.Values[i] != v {
			t.Errorf("values[%d] = %d, want %d (runs must stay in RUN order, not sorted)", i, obs.ClusteringTimeMs.Values[i], v)
		}
	}
	if obs.ClusteringTimeMs.Median != 400 {
		t.Errorf("median = %d, want 400 (sorted middle of [900,100,400])", obs.ClusteringTimeMs.Median)
	}
	if obs.ClusteringTimeMs.Max != 300 {
		t.Errorf("max = %d, want 300", obs.ClusteringTimeMs.Max)
	}
	if obs.ClusteringTimeMs.Comparison != "median <= max" {
		t.Errorf("comparison = %q, want %q", obs.ClusteringTimeMs.Comparison, "median <= max")
	}
	if !strings.Contains(stdout.String(), "verdict FAIL") {
		t.Errorf("stdout = %q, want containing %q", stdout.String(), "verdict FAIL")
	}
}

// TestHarnessSourceNeverIndexesOrWritesTheStore: main.go never re-indexes
// or writes the shared corpus store, never imports internal/indexer,
// never uses a float in the verdict path, and never carries a hardcoded
// threshold default — proven by a source scan with positive controls so
// an empty/renamed file cannot pass vacuously.
func TestHarnessSourceNeverIndexesOrWritesTheStore(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	t.Logf("inspected %d bytes of main.go", len(src))

	forbidden := []string{
		"indexer.Run(",
		`"github.com/seanb4t/codegraph-go/internal/indexer"`,
		"NewWriter(",
		"float32",
		"float64",
		`"500"`,
	}
	for _, f := range forbidden {
		if bytes.Contains(src, []byte(f)) {
			t.Errorf("main.go contains forbidden substring %q", f)
		}
	}

	positive := []struct {
		substr string
		want   int
	}{
		{"graphstore.Open(", 1},
		{".Snapshot()", 1},
		{"query.AssignCommunities(", 1},
		{"os.WriteFile(", 1},
	}
	for _, p := range positive {
		got := bytes.Count(src, []byte(p.substr))
		if got != p.want {
			t.Errorf("main.go contains %q %d times, want %d", p.substr, got, p.want)
		}
	}
}
