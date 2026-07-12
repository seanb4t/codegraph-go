package indexer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/mainstream/rubyextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// rubyProjectDescriptor records only that a Gemfile was found — Ruby's own
// identity is not namespace/crate-declared the way Java/C#/Rust's is (a
// Gemfile names dependencies, not the app's own module/namespace root), so
// this descriptor carries no usable ModulePath and is present purely to
// satisfy D-03's "per-language project-descriptor hook" contract and to
// record Gemfile presence for the D-11 capability matrix.
type rubyProjectDescriptor struct{}

func (rubyProjectDescriptor) ModulePath() string { return "" }

// readRubyDescriptor resolves Gemfile presence at root. Returns an error if
// absent — the same "descriptor absent" signal Discover already
// contractually treats as "fall back to path-based identity" (D-03); in
// Ruby's case ModuleKey below ignores the descriptor entirely regardless
// (see rubyModuleKey's own doc comment), so this presence check is
// informational only.
func readRubyDescriptor(root string) (rubyProjectDescriptor, error) {
	if _, err := os.Stat(filepath.Join(root, "Gemfile")); err != nil {
		return rubyProjectDescriptor{}, fmt.Errorf("indexer: no Gemfile found under %s", root)
	}
	return rubyProjectDescriptor{}, nil
}

// rubyModuleKey computes Ruby's own directory-relative identity: the ".rb"
// extension stripped from relPath, nothing else. Unlike Python's dotted
// path, Ruby needs no "/"->separator conversion here — require_relative's
// own resolution (rubyextract.resolveRequireRelative) does its path
// arithmetic directly against this same extension-stripped-relPath format,
// so the two MUST stay in lockstep (mirrors tsextract's unconditional,
// descriptor-independent ModuleKey pattern, 05-PATTERNS.md).
func rubyModuleKey(relPath string) string {
	return strings.TrimSuffix(relPath, ".rb")
}

// init registers Ruby as a LanguageSpec (LANG-06, mainstream tier).
func init() {
	registerLanguage(LanguageSpec{
		ID:         "ruby",
		Extensions: []string{".rb"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewRubyParser()
		},
		Extract: rubyextract.Extract,
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			// Unconditional (descriptor-independent) — see rubyModuleKey's
			// own doc comment for why: require_relative resolution is pure
			// directory-relative path arithmetic that must match the
			// target file's own key by construction, Gemfile or not.
			return rubyModuleKey(relPath)
		},
		Descriptor: func(root string) (ProjectDescriptor, error) {
			return readRubyDescriptor(root)
		},
	})
}
