package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/mainstream/swiftextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// swiftProjectDescriptor records only that a Package.swift (Swift Package
// Manager manifest) was found at the repo root — Swift has no single
// repo-wide module/namespace root the way Java/C#/Rust's descriptor
// resolves (an SPM package can declare MULTIPLE targets, each its own
// module), so this descriptor carries no usable ModulePath and is present
// purely to satisfy D-03's "per-language project-descriptor hook" contract
// and to record SPM manifest presence for the D-11 capability matrix
// (mirrors rubyProjectDescriptor's presence-only pattern).
type swiftProjectDescriptor struct{}

func (swiftProjectDescriptor) ModulePath() string { return "" }

// readSwiftDescriptor resolves Package.swift presence at root. Returns an
// error if absent — the same "descriptor absent" signal Discover already
// contractually treats as "fall back to path-based identity" (D-03);
// swiftModuleKey below ignores the descriptor entirely regardless
// (informational only).
func readSwiftDescriptor(root string) (swiftProjectDescriptor, error) {
	if _, err := os.Stat(filepath.Join(root, "Package.swift")); err != nil {
		return swiftProjectDescriptor{}, fmt.Errorf("indexer: no Package.swift found under %s", root)
	}
	return swiftProjectDescriptor{}, nil
}

// swiftModuleKey computes a best-effort SPM target-name identity: under the
// `Sources/<Target>/...` convention, the target's own directory name; any
// other layout falls back to path identity (types.go's documented
// approximation — Swift's own true module identity is resolved by SPM's
// build graph, which this project does not run).
func swiftModuleKey(relPath string) string {
	rest, ok := strings.CutPrefix(relPath, "Sources/")
	if !ok {
		return relPath
	}
	if i := strings.Index(rest, "/"); i > 0 {
		return rest[:i]
	}
	return relPath
}

// init registers Swift as a LanguageSpec (LANG-06, mainstream tier). Gated
// entirely on 05-08's blocking human-verify approval of the [SUS]
// alex-pinkus/tree-sitter-swift grammar pin — this registration exists at
// all only because that checkpoint was approved (05-08-SUMMARY.md).
func init() {
	registerLanguage(LanguageSpec{
		ID:         "swift",
		Extensions: []string{".swift"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewSwiftParser()
		},
		Extract: swiftextract.Extract,
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			// Unconditional (descriptor-independent) -- swiftModuleKey's own
			// SPM-convention path arithmetic needs no manifest content, only
			// Package.swift's PRESENCE recorded for the D-11 matrix.
			return swiftModuleKey(relPath)
		},
		Descriptor: func(root string) (ProjectDescriptor, error) {
			return readSwiftDescriptor(root)
		},
	})
}
