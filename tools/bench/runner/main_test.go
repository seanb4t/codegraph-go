package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/bench"
)

// --- medianFloat64 / medianInt64 (WR-03, Phase 8 re-review) ---
//
// These are the pure arithmetic helpers behind D-05's "median-of-5" per
// metric. An off-by-one here would silently corrupt every published
// throughput/RSS number without any test catching it.

func TestMedianFloat64_OddLength(t *testing.T) {
	got := medianFloat64([]float64{1, 2, 3})
	if got != 2 {
		t.Fatalf("medianFloat64(odd) = %v, want 2", got)
	}
}

func TestMedianFloat64_EvenLength(t *testing.T) {
	got := medianFloat64([]float64{1, 2, 3, 4})
	if got != 2.5 {
		t.Fatalf("medianFloat64(even) = %v, want 2.5", got)
	}
}

func TestMedianFloat64_Empty(t *testing.T) {
	got := medianFloat64(nil)
	if got != 0 {
		t.Fatalf("medianFloat64(empty) = %v, want 0", got)
	}
}

func TestMedianFloat64_Single(t *testing.T) {
	got := medianFloat64([]float64{42})
	if got != 42 {
		t.Fatalf("medianFloat64(single) = %v, want 42", got)
	}
}

func TestMedianInt64_OddLength(t *testing.T) {
	got := medianInt64([]int64{10, 20, 30})
	if got != 20 {
		t.Fatalf("medianInt64(odd) = %v, want 20", got)
	}
}

func TestMedianInt64_EvenLength(t *testing.T) {
	got := medianInt64([]int64{10, 20, 30, 40})
	if got != 25 {
		t.Fatalf("medianInt64(even) = %v, want 25", got)
	}
}

func TestMedianInt64_Empty(t *testing.T) {
	got := medianInt64(nil)
	if got != 0 {
		t.Fatalf("medianInt64(empty) = %v, want 0", got)
	}
}

func TestMedianOfN_RejectsNonPositiveN(t *testing.T) {
	_, err := medianOfN(0, func() (measuredRun, error) { return measuredRun{}, nil })
	if err == nil {
		t.Fatal("medianOfN(0, ...) should error, got nil")
	}
}

func TestMedianOfN_ComputesIndependentMedians(t *testing.T) {
	// Deliberately construct a run sequence where the duration-median
	// and RSS-median come from DIFFERENT individual runs, proving
	// medianOfN computes each metric's median over its own sorted
	// sample rather than picking "the run whose duration happened to be
	// the median" (D-05).
	runs := []measuredRun{
		{durationMS: 100, peakRSS: 5},
		{durationMS: 10, peakRSS: 50},
		{durationMS: 20, peakRSS: 40},
	}
	i := 0
	got, err := medianOfN(len(runs), func() (measuredRun, error) {
		r := runs[i]
		i++
		return r, nil
	})
	if err != nil {
		t.Fatalf("medianOfN: %v", err)
	}
	if got.durationMS != 20 {
		t.Fatalf("median duration = %v, want 20", got.durationMS)
	}
	if got.peakRSS != 40 {
		t.Fatalf("median peakRSS = %v, want 40", got.peakRSS)
	}
}

// --- medianMetrics (perf-gate-throughput-regress) ---
//
// The outer, session-level median. medianOfN collapses jitter INSIDE one
// measurement session; this collapses variance BETWEEN sessions, which is
// the variance that actually flipped the CI verdict (2.4% spread across
// three runs against a 10% budget) and the variance a -rebless would
// otherwise bake into the committed baseline.

func TestMedianMetrics_TakesEachMetricsOwnMedian(t *testing.T) {
	// As with medianOfN, each metric's median is deliberately sourced
	// from a DIFFERENT trial, so an implementation that picked "the
	// trial whose throughput was the median" and copied its other
	// fields wholesale would fail here (D-05).
	trials := []bench.Metrics{
		{FilesPerSec: 100, BytesPerSec: 30, QueryLatencyMedianMS: 5, PeakRSSBytes: 900, ColdStartMS: 3},
		{FilesPerSec: 10, BytesPerSec: 20, QueryLatencyMedianMS: 50, PeakRSSBytes: 100, ColdStartMS: 2},
		{FilesPerSec: 20, BytesPerSec: 10, QueryLatencyMedianMS: 40, PeakRSSBytes: 500, ColdStartMS: 1},
	}
	got, err := medianMetrics(trials)
	if err != nil {
		t.Fatalf("medianMetrics: %v", err)
	}

	if got.FilesPerSec != 20 {
		t.Errorf("FilesPerSec = %v, want 20", got.FilesPerSec)
	}
	if got.BytesPerSec != 20 {
		t.Errorf("BytesPerSec = %v, want 20", got.BytesPerSec)
	}
	if got.QueryLatencyMedianMS != 40 {
		t.Errorf("QueryLatencyMedianMS = %v, want 40", got.QueryLatencyMedianMS)
	}
	if got.PeakRSSBytes != 500 {
		t.Errorf("PeakRSSBytes = %v, want 500", got.PeakRSSBytes)
	}
	if got.ColdStartMS != 2 {
		t.Errorf("ColdStartMS = %v, want 2", got.ColdStartMS)
	}
}

func TestMedianMetrics_RejectsASingleOutlierTrial(t *testing.T) {
	// The exact failure this change exists to prevent: two sessions agree,
	// one is a tail draw. The mean would drag toward the outlier and, on
	// the -rebless path, commit it as the new normal; the median must
	// discard it entirely.
	trials := []bench.Metrics{
		{FilesPerSec: 11400},
		{FilesPerSec: 11450},
		{FilesPerSec: 6000}, // pathological tail session
	}
	got, err := medianMetrics(trials)
	if err != nil {
		t.Fatalf("medianMetrics: %v", err)
	}
	if got.FilesPerSec != 11400 {
		t.Fatalf("FilesPerSec = %v, want 11400 (the outlier must not move the result)", got.FilesPerSec)
	}
}

func TestMedianMetrics_RecordsTrialCountAndCarriesIdentityFields(t *testing.T) {
	trials := []bench.Metrics{
		{Subject: "go", Repo: "synthetic-seed42-count120000", GOOS: "linux", GOARCH: "amd64", FilesPerSec: 3},
		{Subject: "go", Repo: "synthetic-seed42-count120000", GOOS: "linux", GOARCH: "amd64", FilesPerSec: 1},
		{Subject: "go", Repo: "synthetic-seed42-count120000", GOOS: "linux", GOARCH: "amd64", FilesPerSec: 2},
	}
	got, err := medianMetrics(trials)
	if err != nil {
		t.Fatalf("medianMetrics: %v", err)
	}

	if got.MedianOfTrials != 3 {
		t.Errorf("MedianOfTrials = %d, want 3", got.MedianOfTrials)
	}
	// GOOS/GOARCH must survive aggregation: internal/bench.CheckRegression
	// now refuses to compare a baseline against a run on a different
	// platform, so dropping them here would make every reblessed baseline
	// unattributed and permanently unusable.
	if got.GOOS != "linux" || got.GOARCH != "amd64" {
		t.Errorf("platform = %s/%s, want linux/amd64", got.GOOS, got.GOARCH)
	}
	if got.Subject != "go" || got.Repo != "synthetic-seed42-count120000" {
		t.Errorf("identity = %s/%s, want go/synthetic-seed42-count120000", got.Subject, got.Repo)
	}
}

func TestMedianMetrics_Empty(t *testing.T) {
	got, err := medianMetrics(nil)
	if err != nil {
		t.Fatalf("medianMetrics(nil): %v", err)
	}
	if got != (bench.Metrics{}) {
		t.Fatalf("medianMetrics(nil) = %+v, want zero Metrics", got)
	}
}

func TestMedianMetrics_SingleTrialIsRecordedAsSuch(t *testing.T) {
	got, err := medianMetrics([]bench.Metrics{{FilesPerSec: 42}})
	if err != nil {
		t.Fatalf("medianMetrics: %v", err)
	}
	if got.FilesPerSec != 42 {
		t.Errorf("FilesPerSec = %v, want 42", got.FilesPerSec)
	}
	if got.MedianOfTrials != 1 {
		t.Errorf("MedianOfTrials = %d, want 1 — a single-sample number must say so", got.MedianOfTrials)
	}
}

// --- Runner identity (10-04-PLAN, D-09) ---
//
// baseline.json records only goos/goarch, and CheckRegression compares
// exactly those two fields — but namespace-profile-linux-amd64-4x8 IS
// linux/amd64, so moving bench.yml to a new runner class is structurally
// invisible to the existing platform guard. Runner closes that blind spot
// (comparison itself lands in a later plan, deliberately).

func TestParseFlags_RunnerFromEnv(t *testing.T) {
	t.Setenv("CODEGRAPH_BENCH_RUNNER", "namespace-profile-linux-amd64-4x8")
	cfg, err := parseFlags([]string{"-mode", "regression"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if cfg.runner != "namespace-profile-linux-amd64-4x8" {
		t.Errorf("runner = %q, want env value namespace-profile-linux-amd64-4x8", cfg.runner)
	}
}

func TestParseFlags_RunnerFlagOverridesEnv(t *testing.T) {
	t.Setenv("CODEGRAPH_BENCH_RUNNER", "env-value")
	cfg, err := parseFlags([]string{"-mode", "regression", "-runner", "flag-value"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if cfg.runner != "flag-value" {
		t.Errorf("runner = %q, want flag-value (explicit flag must win over env)", cfg.runner)
	}
}

func TestParseFlags_RunnerEmptyWhenNeitherSet(t *testing.T) {
	t.Setenv("CODEGRAPH_BENCH_RUNNER", "")
	cfg, err := parseFlags([]string{"-mode", "regression"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if cfg.runner != "" {
		t.Errorf("runner = %q, want empty — measurement must not require this field", cfg.runner)
	}
}

func TestMedianMetrics_CarriesRunnerWhenIdentical(t *testing.T) {
	trials := []bench.Metrics{
		{Runner: "namespace-profile-linux-amd64-4x8", FilesPerSec: 1},
		{Runner: "namespace-profile-linux-amd64-4x8", FilesPerSec: 2},
		{Runner: "namespace-profile-linux-amd64-4x8", FilesPerSec: 3},
	}
	got, err := medianMetrics(trials)
	if err != nil {
		t.Fatalf("medianMetrics: %v", err)
	}
	if got.Runner != "namespace-profile-linux-amd64-4x8" {
		t.Errorf("Runner = %q, want namespace-profile-linux-amd64-4x8", got.Runner)
	}
}

func TestMedianMetrics_RejectsMixedRunner(t *testing.T) {
	// A mixed-runner aggregate is a category error of exactly the kind
	// internal/bench.CheckRegression already refuses for GOOS/GOARCH —
	// silently picking the first value would let a mid-migration run
	// (some trials on ubuntu-latest, some on Namespace) masquerade as a
	// single, coherent measurement.
	trials := []bench.Metrics{
		{Runner: "namespace-profile-linux-amd64-4x8"},
		{Runner: "ubuntu-latest"},
	}
	_, err := medianMetrics(trials)
	if err == nil {
		t.Fatal("medianMetrics with mixed runner values across trials should error, got nil")
	}
	if !strings.Contains(err.Error(), "namespace-profile-linux-amd64-4x8") || !strings.Contains(err.Error(), "ubuntu-latest") {
		t.Errorf("error %q should name both runner values", err)
	}
}

func TestMetricsRunner_MarshalsToRunnerKey(t *testing.T) {
	m := bench.Metrics{Runner: "namespace-profile-linux-amd64-4x8"}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"runner":"namespace-profile-linux-amd64-4x8"`) {
		t.Errorf("marshalled JSON missing runner key: %s", data)
	}
}

func TestMetricsRunner_RoundTrips(t *testing.T) {
	m := bench.Metrics{Runner: "namespace-profile-linux-amd64-4x8"}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got bench.Metrics
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Runner != m.Runner {
		t.Errorf("round-tripped Runner = %q, want %q", got.Runner, m.Runner)
	}
}

func TestReadBaseline_LegacyFileWithoutRunnerKeyYieldsEmptyString(t *testing.T) {
	// The currently committed tools/bench/baseline.json has no "runner"
	// key. Unmarshalling it must succeed and yield Runner == "" — an
	// empty value means "recorded before this field existed", not an
	// error.
	dir := t.TempDir()
	path := filepath.Join(dir, "baseline.json")
	legacy := `{"subject":"go","repo":"synthetic-seed42-count120000","goos":"linux","goarch":"amd64","median_of_trials":7,"files_per_sec":11279.591291175333,"bytes_per_sec":1708549.4453683186,"query_latency_median_ms":280.943,"peak_rss_bytes":907202560,"cold_start_ms":13.285}`
	mustWriteFile(t, path, legacy)

	got, err := readBaseline(path)
	if err != nil {
		t.Fatalf("readBaseline(legacy, no runner key): %v", err)
	}
	if got.Runner != "" {
		t.Errorf("Runner = %q, want empty string for a legacy baseline with no runner key", got.Runner)
	}
}

// --- copyTree / countTree (WR-03, Phase 8 re-review) ---
//
// copyTree/countTree decide what actually gets measured; a wrong
// skipDirs entry or symlink-handling bug would silently corrupt every
// downstream metric.

func TestCopyTree_SkipsGitAndCodegraphDirs(t *testing.T) {
	src := t.TempDir()
	mustWriteFile(t, filepath.Join(src, "keep.txt"), "keep")
	mustWriteFile(t, filepath.Join(src, ".git", "HEAD"), "ref: refs/heads/main")
	mustWriteFile(t, filepath.Join(src, ".codegraph", "index.db"), "binary-ish")
	mustWriteFile(t, filepath.Join(src, "sub", "nested.txt"), "nested")

	dst := filepath.Join(t.TempDir(), "dst")
	if err := copyTree(src, dst); err != nil {
		t.Fatalf("copyTree: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, "keep.txt")); err != nil {
		t.Errorf("expected keep.txt to be copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "sub", "nested.txt")); err != nil {
		t.Errorf("expected sub/nested.txt to be copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, ".git")); err == nil {
		t.Error(".git should not have been copied")
	}
	if _, err := os.Stat(filepath.Join(dst, ".codegraph")); err == nil {
		t.Error(".codegraph should not have been copied")
	}
}

func TestCopyTree_DoesNotFollowSymlinks(t *testing.T) {
	src := t.TempDir()
	mustWriteFile(t, filepath.Join(src, "real.txt"), "real content")

	linkPath := filepath.Join(src, "link.txt")
	if err := os.Symlink(filepath.Join(src, "real.txt"), linkPath); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}

	dst := filepath.Join(t.TempDir(), "dst")
	if err := copyTree(src, dst); err != nil {
		t.Fatalf("copyTree: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, "real.txt")); err != nil {
		t.Errorf("expected real.txt to be copied: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(dst, "link.txt")); err == nil {
		t.Error("link.txt should not have been copied (symlinks are skipped)")
	}
}

func TestCountTree_CountsFilesAndBytes(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "a.txt"), "12345")     // 5 bytes
	mustWriteFile(t, filepath.Join(dir, "sub", "b.txt"), "67") // 2 bytes
	mustWriteFile(t, filepath.Join(dir, ".git", "HEAD"), "ignored bytes that should not be counted")

	files, bytes, err := countTree(dir)
	if err != nil {
		t.Fatalf("countTree: %v", err)
	}
	if files != 2 {
		t.Errorf("files = %d, want 2 (skipDirs entries excluded)", files)
	}
	if bytes != 7 {
		t.Errorf("bytes = %d, want 7", bytes)
	}
}

func TestCountTree_SkipsCodegraphDir(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "keep.txt"), "x")
	mustWriteFile(t, filepath.Join(dir, ".codegraph", "index.db"), "should not be counted")

	files, _, err := countTree(dir)
	if err != nil {
		t.Fatalf("countTree: %v", err)
	}
	if files != 1 {
		t.Errorf("files = %d, want 1 (.codegraph excluded)", files)
	}
}

// --- parseFlags (WR-03, Phase 8 re-review) ---

func TestParseFlags_ModeRequired(t *testing.T) {
	_, err := parseFlags([]string{})
	if err == nil {
		t.Fatal("parseFlags with no -mode should error")
	}
}

func TestParseFlags_Defaults(t *testing.T) {
	cfg, err := parseFlags([]string{"-mode", "regression"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if cfg.mode != "regression" {
		t.Errorf("mode = %q, want regression", cfg.mode)
	}
	if cfg.count != regressionFileCount {
		t.Errorf("count = %d, want %d", cfg.count, regressionFileCount)
	}
	if cfg.ceilingBytes != defaultCeilingBytes {
		t.Errorf("ceilingBytes = %d, want %d", cfg.ceilingBytes, defaultCeilingBytes)
	}
	if cfg.rebless {
		t.Error("rebless should default to false")
	}
	// The safe procedure must be the DEFAULT one. If -trials defaulted to
	// 1, every caller that forgot the flag — including the CI gate and the
	// rebless workflow — would silently fall back to the single-sample
	// measurement this change exists to eliminate.
	if cfg.trials != defaultTrials {
		t.Errorf("trials = %d, want %d", cfg.trials, defaultTrials)
	}
}

func TestParseFlags_RejectsNonPositiveTrials(t *testing.T) {
	if _, err := parseFlags([]string{"-mode", "regression", "-trials", "0"}); err == nil {
		t.Error("parseFlags with -trials 0 should error")
	}
	if _, err := parseFlags([]string{"-mode", "regression", "-trials", "-1"}); err == nil {
		t.Error("parseFlags with -trials -1 should error")
	}
}

func TestParseFlags_TrialsOverrideApplies(t *testing.T) {
	cfg, err := parseFlags([]string{"-mode", "regression", "-trials", "1"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if cfg.trials != 1 {
		t.Errorf("trials = %d, want 1", cfg.trials)
	}
}

func TestDefaultTrialsIsAtLeastThree(t *testing.T) {
	// Guards the choice, not just the plumbing: N=2 has no true median
	// (medianFloat64 averages the pair, so an outlier still moves the
	// result), and N=1 is the single-sample case. Anything below 3 silently
	// reopens the defect.
	if defaultTrials < 3 {
		t.Fatalf("defaultTrials = %d, want >= 3 — below 3 an outlier session still moves the result", defaultTrials)
	}
}

func TestParseFlags_OverridesApply(t *testing.T) {
	cfg, err := parseFlags([]string{"-mode", "headtohead", "-ts-binary", "/custom/codegraph", "-count", "5"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if cfg.tsBinary != "/custom/codegraph" {
		t.Errorf("tsBinary = %q, want /custom/codegraph", cfg.tsBinary)
	}
	if cfg.count != 5 {
		t.Errorf("count = %d, want 5", cfg.count)
	}
}

// --- resolveTSBinary (IN-02, Phase 8 re-review) ---

func TestResolveTSBinary_FindsOnPath(t *testing.T) {
	dir := t.TempDir()
	fakeBinary := filepath.Join(dir, "codegraph")
	mustWriteFile(t, fakeBinary, "#!/bin/sh\n")
	if err := os.Chmod(fakeBinary, 0o755); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	t.Setenv("PATH", dir)

	got := resolveTSBinary()
	if got != fakeBinary {
		t.Errorf("resolveTSBinary() = %q, want %q", got, fakeBinary)
	}
}

func TestResolveTSBinary_EmptyWhenNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	got := resolveTSBinary()
	if got != "" && got != macOSHomebrewTSBinary {
		t.Errorf("resolveTSBinary() = %q, want empty or the Homebrew fallback", got)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
