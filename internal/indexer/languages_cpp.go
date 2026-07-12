package indexer

import (
	"github.com/seanb4t/codegraph-go/internal/indexer/mainstream/cextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// init registers C++ as a LanguageSpec (LANG-06, mainstream tier), sharing
// cextract.Extract and the CMakeLists.txt-presence descriptor with C
// (languages_c.go — see that file's own doc comment for the shared
// extractor/descriptor rationale). C++ claims only unambiguously-C++
// extensions (".hpp"/".hh", not the shared ".h" — the documented C/C++
// header-ambiguity disposition).
func init() {
	registerLanguage(LanguageSpec{
		ID:         "cpp",
		Extensions: []string{".cpp", ".cc", ".cxx", ".hpp", ".hh"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewCppParser()
		},
		Extract: cextract.Extract,
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			// Unconditional path identity — C++ has no in-source module
			// declaration to override it with (mirrors languages_c.go's
			// identical rationale).
			return relPath
		},
		Descriptor: func(root string) (ProjectDescriptor, error) {
			return readCProjectDescriptor(root)
		},
	})
}
