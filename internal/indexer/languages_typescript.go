package indexer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/indexer/tsextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// tsProjectDescriptor wraps a resolved package.json "name" — TS/JS's own
// first ProjectDescriptor implementation (D-03). Unlike Java's package/C#'s
// namespace (both declared IN the source) and unlike Python's directory-
// structure-derived dotted path, TS/JS's ModuleKey (below) needs NO
// descriptor content at all — see tsextract/types.go's package doc,
// "Cross-file module resolution", for why ModuleKey is unconditionally
// NormalizeModuleKey(relPath) regardless of whether a descriptor was
// resolved. This descriptor exists to satisfy the ProjectDescriptor
// interface (informational only, languages.go's own doc: "a TS/JS resolved
// package name for later languages") and, more importantly, as the vehicle
// readTSDescriptor uses to call tsextract.SetConfig exactly once per repo
// root — the tsconfig.json baseUrl/paths side-channel tsextract.Extract's
// module-specifier resolution consults (tsextract/types.go's package doc
// explains the full architectural rationale for this package-level
// singleton).
type tsProjectDescriptor struct {
	packageName string
}

func (d tsProjectDescriptor) ModulePath() string { return d.packageName }

type tsconfigCompilerOptions struct {
	BaseURL string              `json:"baseUrl"`
	Paths   map[string][]string `json:"paths"`
}

type tsconfigFile struct {
	CompilerOptions tsconfigCompilerOptions `json:"compilerOptions"`
}

type packageJSONFile struct {
	Name string `json:"name"`
}

// readTSDescriptor resolves a TS/JS repo root's tsconfig.json baseUrl/paths
// (installed into tsextract's module-resolution config via SetConfig,
// unconditionally, even when neither manifest exists — an EMPTY config
// prevents any stale config from a previous readTSDescriptor call in the
// same process, relevant mainly to test binaries exercising multiple
// scenarios) and package.json "name" (this repo's own ProjectDescriptor
// identity). Neither manifest's absence is a hard failure (D-03/
// T-05-Manifest): this function returns an error ONLY when BOTH are
// missing, the same "descriptor absent" signal Discover already
// contractually treats as "fall back to path-based identity" — which, for
// TS/JS, is unconditionally already ModuleKey's own behavior regardless of
// this return value (see this file's package doc).
//
// tsconfig.json's `extends` chain is not followed, and a tsconfig.json
// containing comments/trailing commas (JSONC, extremely common in real
// repos) simply fails encoding/json.Unmarshal and degrades to an EMPTY
// (no baseUrl, no paths) config rather than a hand-rolled JSONC parser —
// both explicit, documented, accepted gaps (tsextract/types.go's package
// doc, RESEARCH Assumptions Log A1), never a crash.
func readTSDescriptor(root string) (tsProjectDescriptor, error) {
	hasTSConfig := false
	var cfg tsextract.Config
	if data, err := os.ReadFile(filepath.Join(root, "tsconfig.json")); err == nil {
		hasTSConfig = true
		var tc tsconfigFile
		if json.Unmarshal(data, &tc) == nil {
			cfg.BaseURL = tc.CompilerOptions.BaseURL
			cfg.Paths = tc.CompilerOptions.Paths
		}
	}

	hasPackageJSON := false
	packageName := ""
	if data, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil {
		hasPackageJSON = true
		var pkg packageJSONFile
		if json.Unmarshal(data, &pkg) == nil {
			packageName = pkg.Name
		}
	}

	tsextract.SetConfig(cfg)

	if !hasTSConfig && !hasPackageJSON {
		return tsProjectDescriptor{}, fmt.Errorf("indexer: no tsconfig.json/package.json found under %s", root)
	}
	return tsProjectDescriptor{packageName: packageName}, nil
}

// tsModuleKey is shared verbatim by all three registrations below —
// TS/JS's cross-file identity is unconditionally the extension-stripped,
// repo-root-relative path (tsextract.NormalizeModuleKey), regardless of
// descriptor presence/content (see tsextract/types.go's package doc,
// "Cross-file module resolution", for the full rationale: relative-
// specifier resolution inside tsextract.Extract must match this file's own
// key by construction, in every repo, descriptor or not).
func tsModuleKey(_ ProjectDescriptor, relPath string) string {
	return tsextract.NormalizeModuleKey(relPath)
}

// tsDescriptor is shared verbatim by all three registrations below so a
// repo with files of more than one of the three grammars (extremely common
// — a real TS/JS repo is rarely 100% one extension) resolves tsconfig.json/
// package.json IDENTICALLY regardless of which registration's Descriptor
// hook Discover happens to call (Discover attempts each PRESENT language's
// Descriptor exactly once, per language ID, per repo root — see
// discover.go's descriptorAttempted map).
func tsDescriptor(root string) (ProjectDescriptor, error) {
	d, err := readTSDescriptor(root)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// init registers TypeScript, TSX, and JavaScript as three separate
// LanguageSpecs sharing ONE Extract function (LANG-05) — tsextract.Extract
// self-derives which of the three grammars produced a given file purely
// from its own relPath extension (tsextract.go's languageForPath), so the
// three registrations differ only in ID/Extensions/NewParser.
func init() {
	registerLanguage(LanguageSpec{
		ID:         "typescript",
		Extensions: []string{".ts"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewTypeScriptParser()
		},
		Extract:    tsextract.Extract,
		ModuleKey:  tsModuleKey,
		Descriptor: tsDescriptor,
	})
	registerLanguage(LanguageSpec{
		ID:         "tsx",
		Extensions: []string{".tsx"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewTSXParser()
		},
		Extract:    tsextract.Extract,
		ModuleKey:  tsModuleKey,
		Descriptor: tsDescriptor,
	})
	registerLanguage(LanguageSpec{
		ID:         "javascript",
		Extensions: []string{".js", ".jsx", ".mjs", ".cjs"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewJavaScriptParser()
		},
		Extract:    tsextract.Extract,
		ModuleKey:  tsModuleKey,
		Descriptor: tsDescriptor,
	})
}
