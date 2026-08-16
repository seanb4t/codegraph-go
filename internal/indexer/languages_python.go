package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/pyextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// pythonProjectDescriptor wraps the resolved sys.path package root — a
// relative, POSIX-style directory name (from the repo root) that
// dotted-module-path computation is rooted at — Python's own first
// ProjectDescriptor implementation (D-03). Unlike Java's package/C#'s
// namespace (both declared IN the source), Python's dotted module path is
// entirely directory-structure-derived (RESEARCH "Don't Hand-Roll":
// sys.path/package-root-relative dotted paths), so — unlike javaextract/
// csharpextract — pyextract.Extract never needs to override this at parse
// time; the ModuleKey computed below is already authoritative for every
// file.
type pythonProjectDescriptor struct {
	packageRoot string // "" (repo root) or a single top-level directory, e.g. "src"
}

func (d pythonProjectDescriptor) ModulePath() string { return d.packageRoot }

// readPythonDescriptor resolves a Python repo root's package root. This is
// a bounded, documented heuristic (Claude's Discretion, per D-03/RESEARCH
// Assumptions Log A1's PEP 420 namespace-package caveat), not a full
// pyproject.toml parse: a top-level "src" directory is treated as the
// package root (the increasingly common "src layout"); otherwise the repo
// root itself is the package root ("flat layout", the more common case for
// smaller/older projects). A project deviating from both conventions (a
// custom package-dir override in pyproject.toml's build-system table)
// degrades gracefully to a slightly-wrong-but-still-deterministic dotted
// path, self-detected via the D-12 behavioral golden diff (testdata/golden/behavioral_python_test.go)
// rather than a hard failure.
//
// Returns an error if neither pyproject.toml nor setup.py exists at root —
// the same "descriptor absent" signal Discover already contractually
// treats as "fall back to path-based identity" (D-03), never a hard
// failure (T-05-Manifest: accept, no crash, no code execution — this
// function never even parses the manifest's contents, only checks its
// presence).
func readPythonDescriptor(root string) (string, error) {
	hasManifest := false
	for _, name := range []string{"pyproject.toml", "setup.py"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			hasManifest = true
			break
		}
	}
	if !hasManifest {
		return "", fmt.Errorf("indexer: no pyproject.toml/setup.py found under %s", root)
	}
	if info, err := os.Stat(filepath.Join(root, "src")); err == nil && info.IsDir() {
		return "src", nil
	}
	return "", nil
}

// init registers Python as a LanguageSpec (LANG-04). The Python parser
// itself (cgo.NewPythonParser) already existed from Phase 1 — this plan
// adds only the extractor + resolver wiring.
func init() {
	registerLanguage(LanguageSpec{
		ID:         "python",
		Extensions: []string{".py"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewPythonParser()
		},
		Extract: pyextract.Extract,
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			if descriptor == nil {
				// D-03 path-identity fallback (mirrors Go/Java/C#'s own
				// nil-descriptor fallback).
				return relPath
			}
			packageRoot := descriptor.ModulePath()
			rel := relPath
			if packageRoot != "" {
				prefix := packageRoot + "/"
				if strings.HasPrefix(rel, prefix) {
					rel = strings.TrimPrefix(rel, prefix)
				}
				// A file outside the declared package root (e.g. a
				// top-level tests/ directory alongside a src-layout
				// package) is still dotted-path-computed, just from its
				// own repo-root-relative path instead — an accepted,
				// documented gap (this project models no per-file
				// sys.path membership).
			}
			return dottedModulePath(rel)
		},
		Descriptor: func(root string) (ProjectDescriptor, error) {
			packageRoot, err := readPythonDescriptor(root)
			if err != nil {
				return nil, err
			}
			return pythonProjectDescriptor{packageRoot: packageRoot}, nil
		},
	})
}

// dottedModulePath converts a (package-root-relative) file path into
// Python's own dotted module-path identity: "/" -> ".", the ".py"
// extension stripped, and a trailing ".__init__" collapsed away (a
// package's __init__.py module IS the package itself, not a
// "package.__init__" submodule of it).
func dottedModulePath(rel string) string {
	dotted := strings.TrimSuffix(rel, ".py")
	dotted = strings.ReplaceAll(dotted, "/", ".")
	if dotted == "__init__" {
		return ""
	}
	return strings.TrimSuffix(dotted, ".__init__")
}
