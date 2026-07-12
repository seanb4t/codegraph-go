package indexer

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/mainstream/rustextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// rustProjectDescriptor wraps a Cargo.toml-resolved crate name — Rust's own
// first ProjectDescriptor implementation (D-03), mirroring
// csharpProjectDescriptor's shape.
type rustProjectDescriptor struct {
	crateName string
}

func (d rustProjectDescriptor) ModulePath() string { return d.crateName }

// readRustDescriptor resolves a Rust repo root's crate name from
// Cargo.toml's [package] name field. This is a bounded, documented
// heuristic (mirroring pyextract's presence-not-parse discipline) — a
// line-oriented scan for the FIRST `name = "..."` line inside a `[package]`
// section, not a real TOML parse (no TOML dependency needed, consistent
// with this project's minimal-dependency constraint). Returns an error if
// Cargo.toml is absent or carries no resolvable [package] name — the same
// "descriptor absent" signal Discover already contractually treats as
// "fall back to path-based identity" (D-03), never a hard failure
// (T-05-Manifest: accept, no crash, no code execution).
func readRustDescriptor(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "Cargo.toml"))
	if err != nil {
		return "", fmt.Errorf("indexer: no Cargo.toml found under %s: %w", root, err)
	}
	inPackage := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inPackage = trimmed == "[package]"
			continue
		}
		if !inPackage {
			continue
		}
		rest, ok := strings.CutPrefix(trimmed, "name")
		if !ok {
			continue
		}
		rest = strings.TrimSpace(rest)
		rest, ok = strings.CutPrefix(rest, "=")
		if !ok {
			continue
		}
		name := strings.Trim(strings.TrimSpace(rest), `"`)
		if name != "" {
			return name, nil
		}
	}
	return "", fmt.Errorf("indexer: no [package] name found in Cargo.toml under %s", root)
}

// rustModulePath converts a repo-root-relative .rs file path into Rust's
// own Cargo-convention module-path suffix (everything after the crate
// name): a leading "src/" is stripped, the ".rs" extension is stripped, a
// trailing "mod"/"lib"/"main" filename (a module's own root file) is
// dropped in favor of its enclosing directory, and "/" becomes "::". A
// repo-root-relative path outside "src/" (tests/, examples/, benches/ — each
// technically its OWN crate root under Cargo's conventions) is still
// module-path-computed the same way, an accepted, documented approximation
// for this mainstream tier.
func rustModulePath(rel string) string {
	rel = strings.TrimPrefix(rel, "src/")
	rel = strings.TrimSuffix(rel, ".rs")
	base := path.Base(rel)
	if base == "mod" || base == "lib" || base == "main" {
		rel = path.Dir(rel)
		if rel == "." {
			return ""
		}
	}
	return strings.ReplaceAll(rel, "/", "::")
}

// init registers Rust as a LanguageSpec (LANG-06, mainstream tier).
func init() {
	registerLanguage(LanguageSpec{
		ID:         "rust",
		Extensions: []string{".rs"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewRustParser()
		},
		Extract: rustextract.Extract,
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			if descriptor == nil {
				// D-03 path-identity fallback (mirrors Go/Java/C#/Python's
				// own nil-descriptor fallback).
				return relPath
			}
			crate := descriptor.ModulePath()
			modPath := rustModulePath(relPath)
			if modPath == "" {
				return crate
			}
			return crate + "::" + modPath
		},
		Descriptor: func(root string) (ProjectDescriptor, error) {
			crateName, err := readRustDescriptor(root)
			if err != nil {
				return nil, err
			}
			return rustProjectDescriptor{crateName: crateName}, nil
		},
	})
}
