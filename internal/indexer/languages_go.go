package indexer

import (
	"github.com/seanb4t/codegraph-go/internal/indexer/goextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// goProjectDescriptor wraps Go's existing go.mod-derived module path as the
// FIRST implementation of the ProjectDescriptor hook (D-03) — every later
// language's descriptor follows this exact shape: parsed once per repo
// root, exposing a single resolved base identity.
type goProjectDescriptor struct {
	modulePath string
}

func (d goProjectDescriptor) ModulePath() string { return d.modulePath }

// init registers Go as the first LanguageSpec, proving the multi-language
// seam end-to-end against an existing, working extractor (goextract.Extract)
// before any new language is added.
func init() {
	registerLanguage(LanguageSpec{
		ID:         "go",
		Extensions: []string{".go"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewGoParser()
		},
		// Go's importPath and this registry's moduleKey are the same
		// concept — goextract.Extract's signature already matches
		// LanguageSpec.Extract verbatim, no adapter needed.
		Extract: goextract.Extract,
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			if descriptor == nil {
				return relPath
			}
			return importPathFor(descriptor.ModulePath(), relPath)
		},
		Descriptor: func(root string) (ProjectDescriptor, error) {
			modulePath, err := readModulePath(root)
			if err != nil {
				return nil, err
			}
			return goProjectDescriptor{modulePath: modulePath}, nil
		},
	})
}
