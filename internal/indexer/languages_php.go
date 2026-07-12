package indexer

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/indexer/mainstream/phpextract"
	"github.com/seanb4t/codegraph-go/internal/parser"
	"github.com/seanb4t/codegraph-go/internal/parser/cgo"
)

// phpProjectDescriptor wraps a composer.json's resolved PSR-4 autoload map
// (namespace prefix -> directory prefix) — PHP's own first
// ProjectDescriptor implementation (D-03). ModulePath returns "" (unused):
// unlike C#'s single root-namespace string, PHP's identity is a whole MAP
// consulted per-file by ModuleKey below, not one repo-wide base string.
type phpProjectDescriptor struct {
	psr4 map[string]string
}

func (d phpProjectDescriptor) ModulePath() string { return "" }

// composerJSON is the minimal composer.json shape this project reads: only
// the autoload.psr-4 map, never executed or otherwise interpreted.
type composerJSON struct {
	Autoload struct {
		PSR4 map[string]string `json:"psr-4"`
	} `json:"autoload"`
}

// readPHPDescriptor resolves a PHP repo root's PSR-4 autoload map from
// composer.json. Returns an error if composer.json is absent, unparseable,
// or carries no autoload.psr-4 entries — the same "descriptor absent"
// signal Discover already contractually treats as "fall back to path-based
// identity" (D-03), never a hard failure (T-05-Manifest: accept, no crash,
// no code execution — encoding/json never executes anything from the
// manifest).
func readPHPDescriptor(root string) (phpProjectDescriptor, error) {
	data, err := os.ReadFile(filepath.Join(root, "composer.json"))
	if err != nil {
		return phpProjectDescriptor{}, fmt.Errorf("indexer: no composer.json found under %s: %w", root, err)
	}
	var cj composerJSON
	if err := json.Unmarshal(data, &cj); err != nil {
		return phpProjectDescriptor{}, fmt.Errorf("indexer: parsing composer.json under %s: %w", root, err)
	}
	if len(cj.Autoload.PSR4) == 0 {
		return phpProjectDescriptor{}, fmt.Errorf("indexer: no autoload.psr-4 map found in composer.json under %s", root)
	}
	return phpProjectDescriptor{psr4: cj.Autoload.PSR4}, nil
}

// phpNamespaceFor resolves relPath's directory against psr4's map, picking
// the LONGEST matching directory prefix (PSR-4's own longest-prefix-match
// rule); ties break on the lexicographically first namespace prefix
// (map iteration order is otherwise non-deterministic — D-01a). A
// repo-root PSR-4 entry (an empty or "." directory prefix) is skipped for
// this bounded implementation (an accepted, documented simplification).
func phpNamespaceFor(psr4 map[string]string, relPath string) (string, bool) {
	dir := path.Dir(relPath)

	keys := make([]string, 0, len(psr4))
	for ns := range psr4 {
		keys = append(keys, ns)
	}
	sort.Strings(keys)

	var bestNS, bestDir string
	for _, ns := range keys {
		d := strings.TrimSuffix(psr4[ns], "/")
		if d == "" || d == "." {
			continue
		}
		if dir == d || strings.HasPrefix(dir, d+"/") {
			if len(d) > len(bestDir) {
				bestDir, bestNS = d, ns
			}
		}
	}
	if bestDir == "" {
		return "", false
	}

	rest := strings.TrimPrefix(dir, bestDir)
	rest = strings.TrimPrefix(rest, "/")
	ns := strings.TrimSuffix(bestNS, "\\")
	if rest != "" {
		ns += "\\" + strings.ReplaceAll(rest, "/", "\\")
	}
	return ns, true
}

// init registers PHP as a LanguageSpec (LANG-06, mainstream tier).
//
// ModuleKey's signature (descriptor, relPath) cannot see file content, so
// it can only ever compute a PATH-BASED (or, here, PSR-4-derived)
// placeholder — PHP's real cross-file identity is declared IN the source
// (`namespace Foo\Bar;`, independent of directory layout). phpextract.Extract
// parses that declaration and overrides FileResult.ImportPath with it once
// parsing has actually happened; this ModuleKey is only ever load-bearing
// for a file with no namespace declaration at all (mirrors
// languages_csharp.go's identical parse-time-override rationale).
func init() {
	registerLanguage(LanguageSpec{
		ID:         "php",
		Extensions: []string{".php"},
		NewParser: func() (parser.Parser, error) {
			return cgo.NewPHPParser()
		},
		Extract: phpextract.Extract,
		ModuleKey: func(descriptor ProjectDescriptor, relPath string) string {
			if descriptor == nil {
				// D-03 path-identity fallback (mirrors Go/Java/C#/Python's
				// own nil-descriptor fallback) -- phpextract.Extract
				// overrides this with a parsed `namespace` declaration
				// whenever one is present.
				return relPath
			}
			d, ok := descriptor.(phpProjectDescriptor)
			if !ok || len(d.psr4) == 0 {
				return relPath
			}
			if ns, ok := phpNamespaceFor(d.psr4, relPath); ok {
				return ns
			}
			return relPath
		},
		Descriptor: func(root string) (ProjectDescriptor, error) {
			d, err := readPHPDescriptor(root)
			if err != nil {
				return nil, err
			}
			return d, nil
		},
	})
}
