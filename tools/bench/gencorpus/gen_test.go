package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// crossCallPattern matches a qualified cross-package call to a generated
// symbol, e.g. "pkg0001.Fn0001_0000(" — proof that Generate wires real
// cross-file/cross-package edges rather than emitting zero-edge files
// (RESEARCH Pattern 2).
var crossCallPattern = regexp.MustCompile(`pkg\d{4}\.Fn\d{4}_\d{4}\(`)

// treeHash walks dir and returns a deterministic sha256 digest over the
// sorted (relative path, file bytes) pairs found there, plus the number of
// files walked. This is the tool TestDeterministic uses to prove Generate's
// core reproducibility contract: the same seed always materializes a
// byte-identical directory tree (T-08-09) — never map-iteration order or
// filesystem readdir order, which are not guaranteed stable across runs.
func treeHash(t *testing.T, dir string) (string, int) {
	t.Helper()

	var paths []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	sort.Strings(paths)

	h := sha256.New()
	for _, rel := range paths {
		data, readErr := os.ReadFile(filepath.Join(dir, rel))
		if readErr != nil {
			t.Fatalf("read %s: %v", rel, readErr)
		}
		h.Write([]byte(rel))
		h.Write([]byte{0})
		h.Write(data)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), len(paths)
}

// TestDeterministic proves gencorpus's central property: Generate with a
// fixed seed materializes a byte-identical directory tree every run (same
// file count, paths, and content hashes) — the property that makes a
// committed baseline meaningful and the CI gate non-flaky (D-04, T-08-09).
// A different seed must NOT reproduce the same tree, confirming the RNG
// genuinely drives generation rather than being seeded-but-unused.
func TestDeterministic(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	optsA := Options{Seed: 42, FileCount: 500, OutDir: dirA}
	optsB := Options{Seed: 42, FileCount: 500, OutDir: dirB}

	statsA, err := Generate(optsA)
	if err != nil {
		t.Fatalf("Generate(A): %v", err)
	}
	statsB, err := Generate(optsB)
	if err != nil {
		t.Fatalf("Generate(B): %v", err)
	}

	if statsA.FilesWritten != statsB.FilesWritten {
		t.Fatalf("FilesWritten differ for identical seed: A=%d B=%d", statsA.FilesWritten, statsB.FilesWritten)
	}
	if statsA.BytesWritten != statsB.BytesWritten {
		t.Fatalf("BytesWritten differ for identical seed: A=%d B=%d", statsA.BytesWritten, statsB.BytesWritten)
	}

	hashA, countA := treeHash(t, dirA)
	hashB, countB := treeHash(t, dirB)
	if countA != countB {
		t.Fatalf("on-disk file counts differ for identical seed: A=%d B=%d", countA, countB)
	}
	if hashA != hashB {
		t.Fatalf("same seed produced different trees: hashA=%s hashB=%s", hashA, hashB)
	}

	dirC := t.TempDir()
	optsC := Options{Seed: 43, FileCount: 500, OutDir: dirC}
	if _, err := Generate(optsC); err != nil {
		t.Fatalf("Generate(C): %v", err)
	}
	hashC, _ := treeHash(t, dirC)
	if hashC == hashA {
		t.Fatalf("different seed produced identical tree hash %s — RNG is not actually driving generation", hashC)
	}
}

// TestFileCountExceeds100k asserts INDX-06's "100k+ files" requirement is
// comfortably met by the production corpus size. The pure PlannedFileCount
// property runs on every invocation (no I/O, always fast); the full 120k
// materialization is gated behind testing.Short() so `-short` CI stays fast
// while a full run still proves the real Generate output, not just the
// plan.
func TestFileCountExceeds100k(t *testing.T) {
	opts := Options{Seed: 42, FileCount: ProductionFileCount, OutDir: t.TempDir()}

	if planned := PlannedFileCount(opts); planned <= 100000 {
		t.Fatalf("PlannedFileCount(production opts) = %d, want > 100000", planned)
	}

	if testing.Short() {
		t.Skip("skipping full 120k materialization in -short mode")
	}

	stats, err := Generate(opts)
	if err != nil {
		t.Fatalf("Generate(production opts): %v", err)
	}
	if stats.FilesWritten <= 100000 {
		t.Fatalf("Stats.FilesWritten = %d, want > 100000", stats.FilesWritten)
	}
}

// TestHasCrossFileRefs proves the corpus is not a zero-edge pile of
// isolated files: at least one generated Go file imports another generated
// package, and at least one generated Go file calls a symbol qualified by
// another generated package's name — real cross-file reference edges, per
// RESEARCH Pattern 2 ("a corpus of 100k files with zero edges would
// understate indexing cost, since cross-file resolution is a real cost
// center this project's own architecture explicitly measures").
func TestHasCrossFileRefs(t *testing.T) {
	dir := t.TempDir()
	opts := Options{Seed: 42, FileCount: 200, OutDir: dir}
	if _, err := Generate(opts); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var goFiles []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".go" {
			goFiles = append(goFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	if len(goFiles) == 0 {
		t.Fatalf("no .go files generated under %s", dir)
	}

	foundImport := false
	foundCall := false
	for _, f := range goFiles {
		data, readErr := os.ReadFile(f)
		if readErr != nil {
			t.Fatalf("read %s: %v", f, readErr)
		}
		src := string(data)
		if strings.Contains(src, "import") && strings.Contains(src, "corpus.local/pkg/") {
			foundImport = true
		}
		if crossCallPattern.MatchString(src) {
			foundCall = true
		}
	}

	if !foundImport {
		t.Fatalf("no generated Go file imports another generated package (corpus.local/pkg/...) — corpus has zero cross-package import edges")
	}
	if !foundCall {
		t.Fatalf("no generated Go file calls a qualified cross-package symbol (pkgNNNN.FnNNNN_NNNN(...)) — corpus has zero cross-file call edges")
	}
}
