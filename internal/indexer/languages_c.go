package indexer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/indexer/mainstream/cextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// cProjectDescriptor records only that a CMakeLists.txt was found at the
// repo root — C/C++ have no in-source module/namespace declaration the way
// Rust/Java/C#/Python do, so ModuleKey below is UNCONDITIONALLY path-based
// (mirrors rubyProjectDescriptor's presence-only pattern); this descriptor
// exists purely to satisfy D-03's "per-language project-descriptor hook"
// contract and to record build-system presence for the D-11 capability
// matrix. Shared by both languages_c.go (this file) and languages_cpp.go —
// see cextract's own package doc comment for why C and C++ share ONE
// extractor across two LanguageSpec registrations.
type cProjectDescriptor struct{}

func (cProjectDescriptor) ModulePath() string { return "" }

// readCProjectDescriptor resolves CMakeLists.txt presence at root. Returns
// an error if absent — the same "descriptor absent" signal Discover already
// contractually treats as "fall back to path-based identity" (D-03); C/C++'s
// ModuleKey ignores the descriptor entirely regardless (informational
// only), mirroring readRubyDescriptor's identical rationale.
func readCProjectDescriptor(root string) (cProjectDescriptor, error) {
	if _, err := os.Stat(filepath.Join(root, "CMakeLists.txt")); err != nil {
		return cProjectDescriptor{}, fmt.Errorf("indexer: no CMakeLists.txt found under %s", root)
	}
	return cProjectDescriptor{}, nil
}

// init registers C as a LanguageSpec (LANG-06, mainstream tier). ".h" is
// claimed by C only (the documented default disposition for the C/C++
// header ambiguity — cextract's own package doc comment and the D-11
// matrix); languages_cpp.go claims only unambiguously-C++ extensions.
func init() {
	registerLanguage(LanguageSpec{
		ID:         "c",
		Extensions: []string{".c", ".h"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewCParser()
		},
		Extract: cextract.Extract,
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			// Unconditional path identity — C has no in-source module
			// declaration to override it with (mirrors rubyModuleKey's
			// descriptor-independent pattern; D-03 nil-descriptor
			// fallback is the same value regardless).
			return relPath
		},
		Descriptor: func(root string) (ProjectDescriptor, error) {
			return readCProjectDescriptor(root)
		},
	})
}
