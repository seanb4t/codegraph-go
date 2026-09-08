// Package web_test binds web/src/lib/highlight.ts's declared language
// coverage to internal/indexer's own LanguageSpec registry (D-19/BRW-06,
// 03-04 Task 3).
//
// SITING IS LOAD-BEARING — do not move this test into internal/indexer,
// for two independent reasons:
//
//  1. Dependency direction: this file lives in package web_test (an
//     EXTERNAL test package for the `web` package, which only holds
//     embed.go's //go:embed directive) so the production `web` package
//     gains no import of internal/indexer. internal/indexer already has
//     no reason to know about the SPA's highlighter.
//
//  2. The number is WRONG inside internal/indexer's own test binary.
//     internal/indexer/extract_test.go registers a synthetic "go-dup"
//     language in an init() (verified this session: exactly one such
//     registration, confined to that file). Inside THAT package's test
//     binary, indexer.RegisteredLanguageIDs() returns 15, not 14 — a
//     future "simplification" moving this guard there would see 15,
//     go red, and the cheapest-looking repair (adding "go-dup" to the
//     expected set) would admit a TEST FIXTURE language into the set
//     that gates which highlighter modules ship in a signed binary.
//     Sited here, in web_test, RegisteredLanguageIDs() sees only the 14
//     real LanguageSpec.ID values every other consumer of this function
//     sees.
//
// THE 14-VS-13 ASYMMETRY: the indexer registers 14 LanguageSpec.ID
// values (verified this session:
// c, cpp, csharp, go, java, javascript, kotlin, php, python, ruby, rust,
// swift, tsx, typescript), covered by 13 highlight.js modules — one
// fewer, because internal/indexer/languages_typescript.go's single
// init() registers three IDs ("typescript", "tsx", "javascript") off
// one shared extractor, and highlight.js's own "typescript" grammar
// already declares "tsx" as an alias. That asymmetry is exactly why the
// alias map (HLJS_ALIAS_COVERAGE) is declared in the TypeScript file
// rather than inferred here: an alias relationship that is only true
// inside a third-party package is exactly the kind of fact that goes
// stale silently if re-derived by hand on this side of the language
// boundary.
//
// Go cannot import TypeScript, so this file reads web/src/lib/highlight.ts
// as TEXT and parses ONE tightly-constrained declaration out of it: the
// exported HIGHLIGHT_COVERAGE array, whose elements are quoted string
// literals, one per line, with no computed values (see that array's own
// doc comment in highlight.ts — changing its shape breaks this guard).
// This is deliberately narrower than extracting identifiers from
// free-form hljs.registerLanguage(...) call syntax: a computed key, a
// reordering, a multi-call line, or a refactor of the registration block
// could make a free-form extraction misread its input while still
// comparing two sets and reporting success. A single-purpose,
// single-consumer array removes the guessing.
package web_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/indexer"
)

var (
	coverageArrayRE = regexp.MustCompile(`(?s)HIGHLIGHT_COVERAGE:\s*string\[\]\s*=\s*\[(.*?)\]`)
	registerCallRE  = regexp.MustCompile(`hljs\.registerLanguage\('([A-Za-z0-9_-]+)'`)
	aliasMapRE      = regexp.MustCompile(`(?s)HLJS_ALIAS_COVERAGE:\s*Record<string,\s*string>\s*=\s*\{(.*?)\}`)
	quotedStringRE  = regexp.MustCompile(`'([A-Za-z0-9_-]+)'`)
	aliasEntryRE    = regexp.MustCompile(`([A-Za-z0-9_]+):\s*'([A-Za-z0-9_-]+)'`)
)

// parsedCoverage is what parseHighlightSource extracts from
// highlight.ts-shaped text: the machine-readable coverage array itself,
// the actual hljs.registerLanguage(...) call identifiers (what code
// really runs), the alias-map KEYS (the identifiers each alias declares
// coverage for, e.g. "tsx"), and the alias-map VALUES (the module each
// alias claims to be covered BY, e.g. "typescript").
type parsedCoverage struct {
	coverage      []string
	registrations []string
	aliasKeys     []string
	aliasTargets  []string
}

// parseHighlightSource extracts the three facts this guard binds
// together from raw TEXT — never TypeScript evaluation, since Go cannot
// import TypeScript. Returns a named error rather than a zero-value
// parsedCoverage on any parse failure, so a moved or reshaped
// declaration fails LOUDLY instead of silently comparing two empty sets
// and passing.
func parseHighlightSource(src string) (parsedCoverage, error) {
	var out parsedCoverage

	covMatch := coverageArrayRE.FindStringSubmatch(src)
	if covMatch == nil {
		return out, fmt.Errorf("HIGHLIGHT_COVERAGE array declaration not found")
	}
	for _, m := range quotedStringRE.FindAllStringSubmatch(covMatch[1], -1) {
		out.coverage = append(out.coverage, m[1])
	}
	if len(out.coverage) == 0 {
		return out, fmt.Errorf("HIGHLIGHT_COVERAGE array declaration found but parsed zero entries")
	}

	for _, m := range registerCallRE.FindAllStringSubmatch(src, -1) {
		out.registrations = append(out.registrations, m[1])
	}
	if len(out.registrations) == 0 {
		return out, fmt.Errorf("no hljs.registerLanguage(...) calls found")
	}

	aliasMatch := aliasMapRE.FindStringSubmatch(src)
	if aliasMatch == nil {
		return out, fmt.Errorf("HLJS_ALIAS_COVERAGE map declaration not found")
	}
	for _, m := range aliasEntryRE.FindAllStringSubmatch(aliasMatch[1], -1) {
		out.aliasKeys = append(out.aliasKeys, m[1])
		out.aliasTargets = append(out.aliasTargets, m[2])
	}

	return out, nil
}

func toSet(ids []string) map[string]bool {
	s := make(map[string]bool, len(ids))
	for _, id := range ids {
		s[id] = true
	}
	return s
}

// compareSets reports, in the same missing/extra shape
// internal/indexer/capability/matrix_test.go uses: "missing" is a
// registered language absent from coverage (would silently render
// unhighlighted); "extra" is a coverage entry for a language the indexer
// cannot produce (describes something that doesn't exist). Both are
// sorted for deterministic failure output.
func compareSets(registered, coverage []string) (missing, extra []string) {
	registeredSet := toSet(registered)
	coverageSet := toSet(coverage)

	for _, id := range registered {
		if !coverageSet[id] {
			missing = append(missing, id)
		}
	}
	for _, id := range coverage {
		if !registeredSet[id] {
			extra = append(extra, id)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}

// highlightSourcePath resolves web/src/lib/highlight.ts relative to this
// test file's own location — this package's directory IS web/, so the
// path is exactly one hop: src/lib/highlight.ts.
func highlightSourcePath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("could not resolve this test file's own path via runtime.Caller")
	}
	return filepath.Join(filepath.Dir(thisFile), "src", "lib", "highlight.ts")
}

// TestHighlightRegistrationCoversRegisteredLanguages is the primary
// guard: the highlighter's declared coverage must be set-equal to the
// indexer's own registry, no missing, no extra — plus three further
// structural properties (see inline comments) that bind the
// machine-readable HIGHLIGHT_COVERAGE array back to the code that
// actually registers a highlight.js module and to the alias map's own
// claims.
func TestHighlightRegistrationCoversRegisteredLanguages(t *testing.T) {
	path := highlightSourcePath(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	parsed, err := parseHighlightSource(string(raw))
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	registered := indexer.RegisteredLanguageIDs()

	t.Logf("observed: %d indexed language IDs %v", len(registered), registered)
	t.Logf("observed: %d HIGHLIGHT_COVERAGE entries %v", len(parsed.coverage), parsed.coverage)
	t.Logf("observed: %d hljs.registerLanguage(...) calls %v", len(parsed.registrations), parsed.registrations)
	t.Logf("observed: %d HLJS_ALIAS_COVERAGE alias targets %v", len(parsed.aliasTargets), parsed.aliasTargets)

	// The parse found at least 14 coverage entries — a pattern that
	// silently matched nothing must fail loudly here. Errorf, not
	// Fatalf: the checks below still run and report their own findings
	// (e.g. exactly which identifier is missing) rather than stopping at
	// the first symptom.
	if len(parsed.coverage) < 14 {
		t.Errorf("HIGHLIGHT_COVERAGE parsed only %d entries (want at least 14) — the parse may have silently matched nothing", len(parsed.coverage))
	}

	missing, extra := compareSets(registered, parsed.coverage)
	if len(missing) > 0 {
		t.Errorf("indexed languages missing from HIGHLIGHT_COVERAGE (would render unhighlighted): %v", missing)
	}
	if len(extra) > 0 {
		t.Errorf("HIGHLIGHT_COVERAGE entries for languages the indexer does not register: %v", extra)
	}

	// The registration calls number at least 13, and every identifier
	// passed to one binds back to a HIGHLIGHT_COVERAGE entry — this is
	// what stops the array from becoming a fiction that satisfies the
	// guard while the module actually registers something else. Errorf,
	// not Fatalf, so the reverse-binding check below still runs and can
	// name the specific stale identifier.
	if len(parsed.registrations) < 13 {
		t.Errorf("found only %d hljs.registerLanguage(...) calls (want at least 13)", len(parsed.registrations))
	}
	coverageSet := toSet(parsed.coverage)
	for _, id := range parsed.registrations {
		if !coverageSet[id] {
			t.Errorf("registerLanguage(%q) has no corresponding HIGHLIGHT_COVERAGE entry — the array has drifted from the registration calls that actually run", id)
		}
	}

	// Every alias-map VALUE is itself one of the registration
	// identifiers — an alias pointing at an unregistered module is a
	// claim that is simply false.
	registrationSet := toSet(parsed.registrations)
	for _, target := range parsed.aliasTargets {
		if !registrationSet[target] {
			t.Errorf("HLJS_ALIAS_COVERAGE points at %q, which is not one of the registered module identifiers", target)
		}
	}

	// The REVERSE binding: every HIGHLIGHT_COVERAGE entry must be backed
	// by either an actual registerLanguage(...) call or a declared alias
	// key — never a stale entry with nothing behind it. Without this
	// check, deleting a registerLanguage(...) call while leaving the
	// (unrelated, hand-maintained) HIGHLIGHT_COVERAGE array untouched
	// would pass the set-equality check against the indexer registry
	// silently — the array would claim coverage for a language nothing
	// registers, exactly the "fiction that satisfies the guard" this
	// file's own header comment warns against.
	backing := toSet(append(append([]string{}, parsed.registrations...), parsed.aliasKeys...))
	var stale []string
	for _, id := range parsed.coverage {
		if !backing[id] {
			stale = append(stale, id)
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("HIGHLIGHT_COVERAGE entries with no backing registerLanguage(...) call or alias-map key: %v", stale)
	}
}

// fixtureHighlightSource is a planted, deliberately-wrong fixture: it
// omits "typescript" from HIGHLIGHT_COVERAGE (a real language the
// indexer registers — a MISSING entry) and adds "zig" (a language the
// indexer has no LanguageSpec for — an EXTRA entry). It exists solely to
// prove compareSets can fail, not to exercise the real highlight.ts.
const fixtureHighlightSource = `
hljs.registerLanguage('c', c);
hljs.registerLanguage('cpp', cpp);
hljs.registerLanguage('csharp', csharp);
hljs.registerLanguage('go', go);
hljs.registerLanguage('java', java);
hljs.registerLanguage('javascript', javascript);
hljs.registerLanguage('kotlin', kotlin);
hljs.registerLanguage('php', php);
hljs.registerLanguage('python', python);
hljs.registerLanguage('ruby', ruby);
hljs.registerLanguage('rust', rust);
hljs.registerLanguage('swift', swift);

export const HLJS_ALIAS_COVERAGE: Record<string, string> = {
	tsx: 'typescript'
};

export const HIGHLIGHT_COVERAGE: string[] = [
	'c',
	'cpp',
	'csharp',
	'go',
	'java',
	'javascript',
	'kotlin',
	'php',
	'python',
	'ruby',
	'rust',
	'swift',
	'tsx',
	'zig'
];
`

// TestHighlightCoverageComparisonDiscriminates is the planted positive
// control (rule 84d1gfpywd): a set-equality assertion that has only ever
// been run against a matching pair has never demonstrated that it can
// fail. This test proves it can, in both directions at once.
func TestHighlightCoverageComparisonDiscriminates(t *testing.T) {
	parsed, err := parseHighlightSource(fixtureHighlightSource)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	registered := indexer.RegisteredLanguageIDs()
	missing, extra := compareSets(registered, parsed.coverage)

	t.Logf("observed against planted fixture: missing=%v extra=%v", missing, extra)

	if len(missing) != 1 || missing[0] != "typescript" {
		t.Fatalf("expected exactly one missing entry (\"typescript\"), got %v", missing)
	}
	if len(extra) != 1 || extra[0] != "zig" {
		t.Fatalf("expected exactly one extra entry (\"zig\"), got %v", extra)
	}
}
