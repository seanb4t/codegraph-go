package indexer

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/seanb4t/codegraph-go/internal/indexer/mainstream/kotlinextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// kotlinProjectDescriptor records only that a build.gradle(.kts) was found
// at the repo root — Kotlin's own cross-file identity is declared IN the
// source (`package foo.bar`), independent of directory layout, so this
// descriptor carries no usable ModulePath and is present purely to satisfy
// D-03's "per-language project-descriptor hook" contract and to record
// Gradle build-file presence for the D-11 capability matrix (mirrors
// rubyProjectDescriptor's presence-only pattern).
type kotlinProjectDescriptor struct{}

func (kotlinProjectDescriptor) ModulePath() string { return "" }

// readKotlinDescriptor resolves build.gradle.kts/build.gradle presence at
// root. Returns an error if absent — the same "descriptor absent" signal
// Discover already contractually treats as "fall back to path-based
// identity" (D-03); this ModuleKey fallback is only ever load-bearing for a
// Kotlin file with NO declared `package` statement, since
// kotlinextract.Extract's parse-time override (mirroring csharpextract/
// phpextract) takes priority whenever one exists.
func readKotlinDescriptor(root string) (kotlinProjectDescriptor, error) {
	for _, f := range []string{"build.gradle.kts", "build.gradle"} {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			return kotlinProjectDescriptor{}, nil
		}
	}
	return kotlinProjectDescriptor{}, fmt.Errorf("indexer: no build.gradle(.kts) found under %s", root)
}

// init registers Kotlin as a LanguageSpec (LANG-06, mainstream tier).
// Gated entirely on 05-08's blocking human-verify approval of the [SUS]
// tree-sitter-grammars/tree-sitter-kotlin grammar pin — this registration
// exists at all only because that checkpoint was approved
// (05-08-SUMMARY.md).
func init() {
	registerLanguage(LanguageSpec{
		ID:         "kotlin",
		Extensions: []string{".kt", ".kts"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewKotlinParser()
		},
		Extract: kotlinextract.Extract,
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			// D-03 path-identity fallback (mirrors Go/Java/C#/Python's own
			// nil-descriptor fallback) -- kotlinextract.Extract overrides
			// this with a parsed `package` declaration whenever one is
			// present.
			return relPath
		},
		Descriptor: func(root string) (ProjectDescriptor, error) {
			return readKotlinDescriptor(root)
		},
	})
}
