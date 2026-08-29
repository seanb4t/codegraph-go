// Package web_test binds web/src/lib/highlight.ts's declared
// EXTENSION_LANGUAGE map back to internal/indexer's own LanguageSpec
// registry (WR-02, memory-v4zqxrz6b3 shape).
//
// EXTENSION_LANGUAGE is a hand-enumerated population transcribed from
// internal/indexer/languages_*.go's LanguageSpec.Extensions lists, with
// nothing in the type system binding the two together. That shape narrows
// SILENTLY: adding an extension to a LanguageSpec (or changing which
// language ID it maps to) leaves EXTENSION_LANGUAGE untouched, and nothing
// fails — the SPA simply falls through to the honest-but-wrong "no
// language" plaintext-escape path for that extension, forever, with no
// test anywhere noticing. This file closes that gap the same way
// highlight_coverage_test.go closes it for registered LANGUAGE IDs: by
// parsing the TS declaration as text (Go cannot import TypeScript) and
// asserting set equality — key AND value — against
// indexer.RegisteredLanguageExtensions(), the disk-backed source of truth.
//
// Sited in web_test for the same reason highlight_coverage_test.go is
// (see that file's header): package web stays free of an internal/indexer
// import, and this package's own directory is web/, so the source path is
// one hop away.
package web_test

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

var extensionMapRE = regexp.MustCompile(`(?s)EXTENSION_LANGUAGE:\s*Record<string,\s*string>\s*=\s*\{(.*?)\}`)
var extensionEntryRE = regexp.MustCompile(`'(\.[A-Za-z0-9]+)':\s*'([A-Za-z0-9_-]+)'`)

// parseExtensionLanguageSource extracts the EXTENSION_LANGUAGE map's
// entries from raw TEXT. Returns a named error rather than an empty map on
// any parse failure, so a moved or reshaped declaration fails LOUDLY
// instead of silently comparing two empty sets and passing.
func parseExtensionLanguageSource(src string) (map[string]string, error) {
	m := extensionMapRE.FindStringSubmatch(src)
	if m == nil {
		return nil, fmt.Errorf("EXTENSION_LANGUAGE map declaration not found")
	}
	entries := extensionEntryRE.FindAllStringSubmatch(m[1], -1)
	if len(entries) == 0 {
		return nil, fmt.Errorf("EXTENSION_LANGUAGE map declaration found but parsed zero entries")
	}
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		out[e[1]] = e[2]
	}
	return out, nil
}

// compareExtensionSets reports, in the same missing/extra/mismatched shape
// the rest of this package's guards use: "missing" is a registered
// extension absent from the TS map (would silently render unhighlighted);
// "extra" is a TS map entry for an extension the indexer does not
// register; "mismatched" is an extension present on both sides but mapped
// to a DIFFERENT language ID (a positive-but-wrong entry, the failure mode
// a bare set-equality-of-keys check would miss).
func compareExtensionSets(registered, tsSide map[string]string) (missing, extra, mismatched []string) {
	for ext, id := range registered {
		tsID, ok := tsSide[ext]
		if !ok {
			missing = append(missing, ext)
			continue
		}
		if tsID != id {
			mismatched = append(mismatched, fmt.Sprintf("%s: indexer=%q ts=%q", ext, id, tsID))
		}
	}
	for ext := range tsSide {
		if _, ok := registered[ext]; !ok {
			extra = append(extra, ext)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	sort.Strings(mismatched)
	return missing, extra, mismatched
}

// TestExtensionLanguageCoversRegisteredExtensions is the primary guard:
// EXTENSION_LANGUAGE's declared population must be set-equal, key and
// value, to indexer.RegisteredLanguageExtensions() — no missing, no extra,
// no mismatched mapping.
func TestExtensionLanguageCoversRegisteredExtensions(t *testing.T) {
	path := highlightSourcePath(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	tsSide, err := parseExtensionLanguageSource(string(raw))
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	registered := indexer.RegisteredLanguageExtensions()

	t.Logf("observed: %d indexer-registered extensions %v", len(registered), registered)
	t.Logf("observed: %d EXTENSION_LANGUAGE entries %v", len(tsSide), tsSide)

	if len(tsSide) < 20 {
		t.Errorf("EXTENSION_LANGUAGE parsed only %d entries (want at least 20) — the parse may have silently matched nothing", len(tsSide))
	}

	missing, extra, mismatched := compareExtensionSets(registered, tsSide)
	if len(missing) > 0 {
		t.Errorf("indexer-registered extensions missing from EXTENSION_LANGUAGE (would render unhighlighted): %v", missing)
	}
	if len(extra) > 0 {
		t.Errorf("EXTENSION_LANGUAGE entries for extensions the indexer does not register: %v", extra)
	}
	if len(mismatched) > 0 {
		t.Errorf("EXTENSION_LANGUAGE entries mapped to a different language than the indexer registers: %v", mismatched)
	}
}

// fixtureExtensionSource is a planted, deliberately-wrong fixture: it
// omits ".rs" (a real extension the indexer registers — a MISSING entry),
// adds ".zig" (an extension the indexer has no LanguageSpec for — an EXTRA
// entry), and maps ".py" to "ruby" instead of "python" (a MISMATCHED
// entry). It exists solely to prove compareExtensionSets can fail in all
// three directions at once, not to exercise the real highlight.ts.
const fixtureExtensionSource = `
export const EXTENSION_LANGUAGE: Record<string, string> = {
	'.c': 'c',
	'.h': 'c',
	'.cpp': 'cpp',
	'.cc': 'cpp',
	'.cxx': 'cpp',
	'.hpp': 'cpp',
	'.hh': 'cpp',
	'.cs': 'csharp',
	'.go': 'go',
	'.java': 'java',
	'.js': 'javascript',
	'.jsx': 'javascript',
	'.mjs': 'javascript',
	'.cjs': 'javascript',
	'.kt': 'kotlin',
	'.kts': 'kotlin',
	'.php': 'php',
	'.py': 'ruby',
	'.rb': 'ruby',
	'.swift': 'swift',
	'.ts': 'typescript',
	'.tsx': 'tsx',
	'.zig': 'zig'
};
`

// TestExtensionLanguageComparisonDiscriminates is the planted positive
// control (rule 84d1gfpywd): a set-equality assertion that has only ever
// been run against a matching pair has never demonstrated that it can
// fail. This proves it can, in all three directions the guard checks.
func TestExtensionLanguageComparisonDiscriminates(t *testing.T) {
	tsSide, err := parseExtensionLanguageSource(fixtureExtensionSource)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	registered := indexer.RegisteredLanguageExtensions()
	missing, extra, mismatched := compareExtensionSets(registered, tsSide)

	t.Logf("observed against planted fixture: missing=%v extra=%v mismatched=%v", missing, extra, mismatched)

	if len(missing) != 1 || missing[0] != ".rs" {
		t.Fatalf("expected exactly one missing entry (\".rs\"), got %v", missing)
	}
	if len(extra) != 1 || extra[0] != ".zig" {
		t.Fatalf("expected exactly one extra entry (\".zig\"), got %v", extra)
	}
	if len(mismatched) != 1 {
		t.Fatalf("expected exactly one mismatched entry, got %v", mismatched)
	}
}
