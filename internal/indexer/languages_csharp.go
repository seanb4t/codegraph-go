package indexer

import (
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/csharpextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// csharpProjectDescriptor wraps a *.csproj's resolved root-namespace
// identity — C#'s own first ProjectDescriptor implementation (D-03),
// mirroring javaProjectDescriptor's shape.
type csharpProjectDescriptor struct {
	rootNamespace string
}

func (d csharpProjectDescriptor) ModulePath() string { return d.rootNamespace }

// csprojPropertyGroup is the minimal *.csproj shape this project reads: an
// SDK-style project file may declare several <PropertyGroup> blocks (e.g.
// one unconditional, one per build Configuration) — this project reads
// every one it finds and takes the first non-empty <RootNamespace>,
// falling back to the first non-empty <AssemblyName> (MSBuild's own
// default identity when RootNamespace is omitted).
type csprojPropertyGroup struct {
	RootNamespace string `xml:"RootNamespace"`
	AssemblyName  string `xml:"AssemblyName"`
}

type csprojProject struct {
	XMLName        xml.Name              `xml:"Project"`
	PropertyGroups []csprojPropertyGroup `xml:"PropertyGroup"`
}

// readCSharpDescriptor resolves a C# repo root's root-namespace identity
// from the first *.csproj found directly under root, returning an error if
// none is present or parseable — the same "descriptor absent, malformed, or
// missing" signal Discover already contractually treats as "fall back to
// path-based identity" (D-03), never a hard failure (T-05-Manifest: accept,
// no crash, no code execution — encoding/xml never executes anything from
// the manifest).
func readCSharpDescriptor(root string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(root, "*.csproj"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("indexer: no *.csproj found under %s", root)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return "", fmt.Errorf("indexer: reading %s: %w", matches[0], err)
	}
	var proj csprojProject
	if err := xml.Unmarshal(data, &proj); err != nil {
		return "", fmt.Errorf("indexer: parsing %s: %w", matches[0], err)
	}
	for _, pg := range proj.PropertyGroups {
		if pg.RootNamespace != "" {
			return pg.RootNamespace, nil
		}
	}
	for _, pg := range proj.PropertyGroups {
		if pg.AssemblyName != "" {
			return pg.AssemblyName, nil
		}
	}
	return "", fmt.Errorf("indexer: no RootNamespace/AssemblyName found in %s", matches[0])
}

// init registers C# as a LanguageSpec (LANG-03).
//
// ModuleKey's signature (descriptor, relPath) cannot see file content, so
// it can only ever compute a PATH-BASED placeholder — C#'s real cross-file
// identity is declared IN the source (`namespace Foo.Bar;`, independent of
// directory layout, RESEARCH Pitfall 2). csharpextract.Extract parses that
// declaration and overrides FileResult.ImportPath with it once parsing has
// actually happened; this ModuleKey is only ever load-bearing for a file
// with no namespace declaration at all.
func init() {
	registerLanguage(LanguageSpec{
		ID:         "csharp",
		Extensions: []string{".cs"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewCSharpParser()
		},
		Extract: csharpextract.Extract,
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			if descriptor == nil {
				// D-03 path-identity fallback (mirrors Go/Java's own
				// nil-descriptor fallback) — csharpextract.Extract
				// overrides this with the parsed `namespace` declaration
				// whenever one is present.
				return relPath
			}
			base := descriptor.ModulePath()
			dir := path.Dir(relPath)
			if dir == "." {
				return base
			}
			return base + "." + strings.ReplaceAll(dir, "/", ".")
		},
		Descriptor: func(root string) (ProjectDescriptor, error) {
			ns, err := readCSharpDescriptor(root)
			if err != nil {
				return nil, err
			}
			return csharpProjectDescriptor{rootNamespace: ns}, nil
		},
	})
}
