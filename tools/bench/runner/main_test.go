package main

import (
	"os"
	"path/filepath"
	"testing"
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
