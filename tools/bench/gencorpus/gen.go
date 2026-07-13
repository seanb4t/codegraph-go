// Command gencorpus generates a deterministic, network-free synthetic
// source-code corpus for the PERF-02 CI regression gate and the INDX-06
// bounded-memory / peak-RSS assertion (see 08-RESEARCH.md Pattern 2).
//
// Reproducibility contract: Generate(opts) with the same Options.Seed
// ALWAYS materializes a byte-identical directory tree — same file set,
// same paths, same contents (T-08-09). Every random decision is drawn from
// a single rand.New(rand.NewSource(opts.Seed)) instance consumed in a
// fixed, deterministic order (package index, then file index, low to
// high) — never the global math/rand source, never a wall-clock-derived
// value, and never anything ordered by map iteration.
//
// This is a CI-gate build/test-time tool only: it is never shipped in the
// production binary, must never touch the network or clone a repo (D-04),
// and must never panic — every failure path returns a wrapped error.
package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
)

// ProductionFileCount is the default corpus size for the PERF-02/INDX-06
// CI gate: comfortably above the literal "100k+ files" requirement so
// internal/indexer's own Discover skip logic (build-tag-excluded files,
// vendor/ and dot-prefixed directories) can never push the effective
// indexed file count back under 100k.
const ProductionFileCount = 120000

// Language population weights. Go dominates (this project's first-priority
// extractor language); Python and JavaScript add "a realistic spread of a
// few other supported languages" per RESEARCH Pattern 2. Cross-file
// reference generation (imports + qualified calls) is only wired for Go —
// that's the population TestHasCrossFileRefs asserts against — Python and
// JS files exist purely for extractor-registry language diversity.
const (
	goWeight = 0.85
	pyWeight = 0.10
	// the remainder (~0.05) goes to JavaScript.

	filesPerGoPackage = 25
	filesPerPyModule  = 20
	filesPerJSModule  = 20
)

// Options configures one Generate call.
type Options struct {
	// Seed drives every random decision Generate makes. The same Seed
	// always produces the same output tree; a different Seed always
	// produces a different one.
	Seed int64
	// FileCount is the total number of source files Generate writes
	// across all generated languages. Scaffolding files (go.mod) are not
	// counted toward it.
	FileCount int
	// OutDir is the directory Generate writes into. Generate creates it
	// if needed and only ever writes under it — it never reads from an
	// existing repo and never touches the network.
	OutDir string
}

// Stats reports what one Generate call actually wrote.
type Stats struct {
	FilesWritten int
	BytesWritten int64
}

// PlannedFileCount is a pure, I/O-free prediction of how many files
// Generate(opts) will write. Kept separate from Generate so a test (or a
// CI preflight check) can assert the production corpus size exceeds
// 100k without paying the cost of materializing 120k files.
func PlannedFileCount(opts Options) int {
	return opts.FileCount
}

// Generate deterministically materializes opts.FileCount source files
// under opts.OutDir and returns what it wrote. It never touches the
// network, never clones a repo, and never panics.
func Generate(opts Options) (Stats, error) {
	if opts.FileCount <= 0 {
		return Stats{}, fmt.Errorf("gencorpus: FileCount must be > 0, got %d", opts.FileCount)
	}
	if opts.OutDir == "" {
		return Stats{}, fmt.Errorf("gencorpus: OutDir must not be empty")
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return Stats{}, fmt.Errorf("gencorpus: create OutDir %s: %w", opts.OutDir, err)
	}

	rng := rand.New(rand.NewSource(opts.Seed))

	goCount := int(float64(opts.FileCount) * goWeight)
	pyCount := int(float64(opts.FileCount) * pyWeight)
	jsCount := opts.FileCount - goCount - pyCount
	if jsCount < 0 {
		// Rounding pushed the remainder negative for small FileCount
		// values; reclaim it from goCount so the total still equals
		// opts.FileCount exactly.
		goCount += jsCount
		jsCount = 0
	}

	if err := writeGoModule(opts.OutDir); err != nil {
		return Stats{}, err
	}

	stats := Stats{}

	written, bytes, err := generateGo(opts.OutDir, goCount, rng)
	if err != nil {
		return Stats{}, fmt.Errorf("gencorpus: generate go population: %w", err)
	}
	stats.FilesWritten += written
	stats.BytesWritten += bytes

	written, bytes, err = generatePython(opts.OutDir, pyCount, rng)
	if err != nil {
		return Stats{}, fmt.Errorf("gencorpus: generate python population: %w", err)
	}
	stats.FilesWritten += written
	stats.BytesWritten += bytes

	written, bytes, err = generateJS(opts.OutDir, jsCount, rng)
	if err != nil {
		return Stats{}, fmt.Errorf("gencorpus: generate js population: %w", err)
	}
	stats.FilesWritten += written
	stats.BytesWritten += bytes

	return stats, nil
}

// writeGoModule writes a minimal go.mod at the corpus root so the Go
// population's import paths (corpus.local/pkg/...) resolve to a real
// module root, matching how internal/indexer.Discover locates a project's
// module boundary. Not counted in Stats.FilesWritten — it's scaffolding,
// not a source file under test.
func writeGoModule(outDir string) error {
	return writeFile(filepath.Join(outDir, "go.mod"), []byte("module corpus.local\n\ngo 1.21\n"))
}

// generateGo writes count Go source files across a package tree rooted at
// <outDir>/pkg. Package p's first file (fileIdx 0, p>0) imports package
// p-1 and calls one of its exported functions — a real cross-package
// import+call edge (RESEARCH Pattern 2: "a corpus of 100k files with zero
// edges would understate indexing cost, since cross-file resolution is a
// real cost center this project's own architecture explicitly measures").
// Every other file calls the immediately preceding file's function in the
// same package — a same-package, cross-file call edge that needs no
// import in Go, exercising a second, cheaper resolution path.
func generateGo(outDir string, count int, rng *rand.Rand) (int, int64, error) {
	if count <= 0 {
		return 0, 0, nil
	}

	written := 0
	var bytesWritten int64
	pkgIdx := 0
	fileIdx := 0

	for written < count {
		if fileIdx == filesPerGoPackage {
			fileIdx = 0
			pkgIdx++
		}

		pkgName := fmt.Sprintf("pkg%04d", pkgIdx)
		fnName := fmt.Sprintf("Fn%04d_%04d", pkgIdx, fileIdx)
		lit := rng.Int63n(1 << 30)

		var body string
		switch {
		case fileIdx == 0 && pkgIdx == 0:
			// The very first file overall has no prior generated symbol
			// to reference; it anchors the chain with a deterministic,
			// seed-derived literal.
			body = fmt.Sprintf(
				"package %s\n\n// %s is the corpus reference chain's root symbol.\nfunc %s() int {\n\treturn %d\n}\n",
				pkgName, fnName, fnName, lit,
			)
		case fileIdx == 0:
			// First file of a non-root package: real cross-package
			// import + qualified call. prevPkg is guaranteed to hold
			// exactly filesPerGoPackage files, since pkgIdx only ever
			// advances once its predecessor filled that many.
			prevPkg := fmt.Sprintf("pkg%04d", pkgIdx-1)
			prevFn := fmt.Sprintf("Fn%04d_%04d", pkgIdx-1, filesPerGoPackage-1)
			body = fmt.Sprintf(
				"package %s\n\nimport \"corpus.local/pkg/%s\"\n\n// %s cross-references %s.%s (cross-package call edge).\nfunc %s() int {\n\treturn %s.%s() + %d\n}\n",
				pkgName, prevPkg, fnName, prevPkg, prevFn, fnName, prevPkg, prevFn, lit,
			)
		default:
			// Same-package, cross-file call: no import needed in Go.
			prevFn := fmt.Sprintf("Fn%04d_%04d", pkgIdx, fileIdx-1)
			body = fmt.Sprintf(
				"package %s\n\n// %s cross-references %s (same-package, cross-file call edge).\nfunc %s() int {\n\treturn %s() + %d\n}\n",
				pkgName, fnName, prevFn, fnName, prevFn, lit,
			)
		}

		path := filepath.Join(outDir, "pkg", pkgName, fmt.Sprintf("file%04d.go", fileIdx))
		if err := writeFile(path, []byte(body)); err != nil {
			return written, bytesWritten, err
		}
		bytesWritten += int64(len(body))
		written++
		fileIdx++
	}

	return written, bytesWritten, nil
}

// generatePython writes count standalone Python files under
// <outDir>/py. Cross-file references are only asserted (and only wired)
// for the dominant Go population; Python exists to add extractor-registry
// language diversity (RESEARCH Pattern 2's "realistic spread") without
// complicating the corpus's one tested cross-reference chain.
func generatePython(outDir string, count int, rng *rand.Rand) (int, int64, error) {
	if count <= 0 {
		return 0, 0, nil
	}

	written := 0
	var bytesWritten int64
	modIdx := 0
	fileIdx := 0

	for written < count {
		if fileIdx == filesPerPyModule {
			fileIdx = 0
			modIdx++
		}

		fnName := fmt.Sprintf("fn_%04d_%04d", modIdx, fileIdx)
		lit := rng.Int63n(1 << 30)
		body := fmt.Sprintf(
			"\"\"\"Generated corpus module mod%04d.\"\"\"\n\n\ndef %s():\n    return %d\n",
			modIdx, fnName, lit,
		)

		path := filepath.Join(outDir, "py", fmt.Sprintf("mod%04d", modIdx), fmt.Sprintf("file%04d.py", fileIdx))
		if err := writeFile(path, []byte(body)); err != nil {
			return written, bytesWritten, err
		}
		bytesWritten += int64(len(body))
		written++
		fileIdx++
	}

	return written, bytesWritten, nil
}

// generateJS writes count standalone JavaScript files under <outDir>/js —
// language diversity only, same rationale as generatePython.
func generateJS(outDir string, count int, rng *rand.Rand) (int, int64, error) {
	if count <= 0 {
		return 0, 0, nil
	}

	written := 0
	var bytesWritten int64
	modIdx := 0
	fileIdx := 0

	for written < count {
		if fileIdx == filesPerJSModule {
			fileIdx = 0
			modIdx++
		}

		fnName := fmt.Sprintf("fn%04d_%04d", modIdx, fileIdx)
		lit := rng.Int63n(1 << 30)
		body := fmt.Sprintf(
			"// Generated corpus module mod%04d.\n\nfunction %s() {\n  return %d;\n}\n\nmodule.exports = { %s };\n",
			modIdx, fnName, lit, fnName,
		)

		path := filepath.Join(outDir, "js", fmt.Sprintf("mod%04d", modIdx), fmt.Sprintf("file%04d.js", fileIdx))
		if err := writeFile(path, []byte(body)); err != nil {
			return written, bytesWritten, err
		}
		bytesWritten += int64(len(body))
		written++
		fileIdx++
	}

	return written, bytesWritten, nil
}

// writeFile creates path's parent directory (if needed) and writes data to
// it. Shared by every language generator so directory-creation and
// file-write error wrapping stays in one place.
func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("gencorpus: mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("gencorpus: write %s: %w", path, err)
	}
	return nil
}
