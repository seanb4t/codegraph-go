package mcp

import (
	"fmt"
	"io/fs"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
)

// GUARD-01/GUARD-02 (05-CONTEXT.md, T-05-05): ungated resource content is
// worse than none — it is confidently wrong, invisible to the reader, and
// re-earns SURF-01 (internal/mcp/tools.go advertised "default 5" for a
// whole phase after the engine constant became 2). Every checker below
// reads BOTH the claimed value and the expected value from their real
// source, following tools_schema_drift_test.go's and
// instructions_contract_test.go's established shape: derive, don't
// hand-type, and prove the checker itself can fire (T-05-06) before
// trusting what it says about the real tree.

// toolStemsFromNames strips the "codegraph_" prefix from every name in
// names, turning allToolNames()'s registered tool names into the short
// stems resourceURIFor and the embedded resources directory both key on
// (D-09).
func toolStemsFromNames(names []string) []string {
	stems := make([]string, 0, len(names))
	for _, n := range names {
		stems = append(stems, strings.TrimPrefix(n, "codegraph_"))
	}
	return stems
}

// resourceStemSetDiff reports, given an expected and an actual resource
// stem set, which stems are missing (named in expected but absent from
// actual) and which are orphaned (present in actual but never expected).
// Both returned slices are sorted for a stable failure message, and both
// are empty exactly when the two sets agree.
func resourceStemSetDiff(expected, actual []string) (missing, orphaned []string) {
	expectedSet := make(map[string]bool, len(expected))
	for _, s := range expected {
		expectedSet[s] = true
	}
	actualSet := make(map[string]bool, len(actual))
	for _, s := range actual {
		actualSet[s] = true
	}
	for s := range expectedSet {
		if !actualSet[s] {
			missing = append(missing, s)
		}
	}
	for s := range actualSet {
		if !expectedSet[s] {
			orphaned = append(orphaned, s)
		}
	}
	sort.Strings(missing)
	sort.Strings(orphaned)
	return missing, orphaned
}

// TestResourceStemSetDiffIsNotVacuous proves resourceStemSetDiff
// discriminates at the boundaries of its equivalence classes — a checker
// that unconditionally returned two empty slices would otherwise pass
// TestResourceFileSetMatchesToolNames forever (T-05-06).
func TestResourceStemSetDiffIsNotVacuous(t *testing.T) {
	cases := []struct {
		name                      string
		expected, actual          []string
		wantMissing, wantOrphaned []string
	}{
		{"equal sets", []string{"a", "b"}, []string{"a", "b"}, nil, nil},
		{"one missing", []string{"a", "b"}, []string{"a"}, []string{"b"}, nil},
		{"one orphaned", []string{"a"}, []string{"a", "b"}, nil, []string{"b"}},
		{"one of each simultaneously", []string{"a", "b"}, []string{"a", "c"}, []string{"b"}, []string{"c"}},
		{"empty expected set", nil, []string{"a"}, nil, []string{"a"}},
		{"empty actual set", []string{"a"}, nil, []string{"a"}, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			missing, orphaned := resourceStemSetDiff(tc.expected, tc.actual)
			if !slices.Equal(missing, tc.wantMissing) {
				t.Errorf("resourceStemSetDiff(%v, %v) missing = %v, want %v", tc.expected, tc.actual, missing, tc.wantMissing)
			}
			if !slices.Equal(orphaned, tc.wantOrphaned) {
				t.Errorf("resourceStemSetDiff(%v, %v) orphaned = %v, want %v", tc.expected, tc.actual, orphaned, tc.wantOrphaned)
			}
		})
	}
}

// toolNameTokenRe matches a lowercase "codegraph_<word>" token — the shape
// every registered tool name takes (e.g. the token formed by prefixing a
// companionNames entry with "codegraph_", or "explore"'s own token). It is
// deliberately narrower than a generic identifier pattern so it cannot
// accidentally match the env var name family (CODEGRAPH_ tokens are
// upper-case; see envVarTokenRe below).
var toolNameTokenRe = regexp.MustCompile(`codegraph_[a-z]+`)

// TestResourceFileSetMatchesToolNames is GUARD-02's structural half: it
// fails when a tool is added, removed, or renamed without its resource
// file, its resourceURIFor entry, and (for the tools-filter behavior doc)
// its mention in that doc's prose all moving with it — in whichever
// direction the mismatch runs, naming exactly which stems are missing and
// which are orphaned.
func TestResourceFileSetMatchesToolNames(t *testing.T) {
	toolStems := toolStemsFromNames(allToolNames())
	// tools-filter and index-state are unavoidable literals: D-10 named
	// these two behavior-doc URIs for their subject matter, not derived
	// from any runtime value, so there is no source to derive their stems
	// from either.
	behaviorDocStems := []string{"tools-filter", "index-state"}
	expected := append(append([]string{}, toolStems...), behaviorDocStems...)

	entries, err := fs.ReadDir(resourcesFS, "resources")
	if err != nil {
		t.Fatalf("read embedded resources dir: %v", err)
	}
	actual := make([]string, 0, len(entries))
	for _, e := range entries {
		actual = append(actual, strings.TrimSuffix(e.Name(), ".md"))
	}

	// Guard the guard (T-05-06): any of these being empty would make every
	// assertion below pass while verifying nothing.
	if len(expected) == 0 {
		t.Fatal("expected stem set is empty — this test would verify nothing")
	}
	if len(actual) == 0 {
		t.Fatal("actual stem set (embedded resources dir) is empty — this test would verify nothing")
	}
	if len(resourceURIFor) == 0 {
		t.Fatal("resourceURIFor is empty — this test would verify nothing")
	}

	t.Run("file set matches allToolNames plus behavior docs", func(t *testing.T) {
		missing, orphaned := resourceStemSetDiff(expected, actual)
		if len(missing) > 0 || len(orphaned) > 0 {
			t.Errorf("resource file set drifted from the tool roster: missing %v (a tool with no resource file), orphaned %v (a resource file with no matching tool)", missing, orphaned)
		}
	})

	t.Run("resourceURIFor keys match actual filenames", func(t *testing.T) {
		uriKeys := make([]string, 0, len(resourceURIFor))
		for name := range resourceURIFor {
			uriKeys = append(uriKeys, strings.TrimSuffix(name, ".md"))
		}
		missing, orphaned := resourceStemSetDiff(uriKeys, actual)
		if len(missing) > 0 || len(orphaned) > 0 {
			t.Errorf("resourceURIFor's key set drifted from the embedded files: missing %v (a file with no map entry — fatal at server construction), orphaned %v (a map entry left behind by a deleted file — silent without this check)", missing, orphaned)
		}
	})

	t.Run("per-tool URI shape (D-09)", func(t *testing.T) {
		for _, stem := range toolStems {
			file := stem + ".md"
			want := "codegraph://tools/" + stem
			got, ok := resourceURIFor[file]
			if !ok {
				t.Errorf("resourceURIFor has no entry for %s", file)
				continue
			}
			if got != want {
				t.Errorf("resourceURIFor[%q] = %q, want %q", file, got, want)
			}
		}
	})

	t.Run("behavior-doc URI shape (D-10)", func(t *testing.T) {
		wantURIs := map[string]string{
			"tools-filter.md": "codegraph://tools-filter",
			"index-state.md":  "codegraph://index-state",
		}
		for _, file := range slices.Sorted(maps.Keys(wantURIs)) {
			want := wantURIs[file]
			got, ok := resourceURIFor[file]
			if !ok {
				t.Errorf("resourceURIFor has no entry for %s", file)
				continue
			}
			if got != want {
				t.Errorf("resourceURIFor[%q] = %q, want %q", file, got, want)
			}
		}
	})

	// The prose half of GUARD-02: even when the file set and the URI map
	// both stay in sync, the tools-filter behavior doc's OWN prose could
	// still enumerate a stale tool name if a rename touched every other
	// site but this one. docNamesCompanionsWithoutTheFilter
	// (instructions_contract_test.go) checks a different property (whether
	// a doc naming companions also names the filter); this checks that the
	// SET of companion tokens named here is exactly right, in both
	// directions.
	t.Run("tools-filter prose names exactly the registered companions", func(t *testing.T) {
		doc, err := resourcesFS.ReadFile("resources/tools-filter.md")
		if err != nil {
			t.Fatalf("read resources/tools-filter.md: %v", err)
		}
		text := string(doc)

		for _, name := range companionNames {
			token := "codegraph_" + name
			if !strings.Contains(text, token) {
				t.Errorf("tools-filter.md never mentions %s, but it is a registered companion tool the filter can narrow away", token)
			}
		}

		known := make(map[string]bool, len(companionNames)+1)
		known["explore"] = true
		for _, name := range companionNames {
			known[name] = true
		}
		for _, m := range toolNameTokenRe.FindAllString(text, -1) {
			stem := strings.TrimPrefix(m, "codegraph_")
			if !known[stem] {
				t.Errorf("tools-filter.md names %s, which is neither a registered companion tool nor codegraph_explore — a renamed or removed tool left behind in this doc's prose", m)
			}
		}
	})
}

// toolSchemaClaims holds the text GUARD-01's numeric-claim comparison reads
// from a tool's own registered schema: its top-level Description plus every
// one of its jsonschema-inferred property Descriptions.
type toolSchemaClaims struct {
	description string
	schemaProps map[string]*jsonschema.Schema
}

// toolSchemaClaimsFor returns stem's tool-schema claim text, built the same
// way TestMCPToolSchemaNumericClaimsMatchEngineConstants
// (tools_schema_drift_test.go) already builds it: exploreTool()/
// companionTool(stem) for the Description, jsonschema.For[XArgs](nil) for
// the inferred property Descriptions — the exact inference mcp.AddTool
// performs at registration time (D-07). Not extracted into a shared helper
// with that file: its own tools map is shaped differently (keyed by full
// registered name, not stem) and this plan does not own that file.
//
// mustProps is a closure, not a package function: Go's multi-value-call
// spreading rule (f(g())) permits no parameter besides g's own return
// values, so a *testing.T parameter cannot ride along — closing over t
// instead keeps t.Fatalf available without violating that rule.
func toolSchemaClaimsFor(t *testing.T, stem string) toolSchemaClaims {
	t.Helper()

	mustProps := func(s *jsonschema.Schema, err error) map[string]*jsonschema.Schema {
		t.Helper()
		if err != nil {
			t.Fatalf("jsonschema.For: %v", err)
		}
		return s.Properties
	}

	if stem == "explore" {
		return toolSchemaClaims{
			description: exploreTool().Description,
			schemaProps: mustProps(jsonschema.For[ExploreArgs](nil)),
		}
	}

	var props map[string]*jsonschema.Schema
	switch stem {
	case "node":
		props = mustProps(jsonschema.For[NodeArgs](nil))
	case "search":
		props = mustProps(jsonschema.For[SearchArgs](nil))
	case "callers":
		props = mustProps(jsonschema.For[CallersArgs](nil))
	case "callees":
		props = mustProps(jsonschema.For[CalleesArgs](nil))
	case "impact":
		props = mustProps(jsonschema.For[ImpactArgs](nil))
	case "files":
		props = mustProps(jsonschema.For[FilesArgs](nil))
	case "status":
		props = mustProps(jsonschema.For[StatusArgs](nil))
	default:
		t.Fatalf("toolSchemaClaimsFor: unknown tool stem %q", stem)
	}
	return toolSchemaClaims{description: companionTool(stem).Description, schemaProps: props}
}

// schemaClaimsText concatenates c's description and every property
// description into one blob for numericClaimsMultiset to scan — the claim
// SITE (which field carried it) does not matter for this comparison, only
// the multiset of claims made.
func schemaClaimsText(c toolSchemaClaims) string {
	parts := make([]string, 0, 1+len(c.schemaProps))
	parts = append(parts, c.description)
	for _, prop := range c.schemaProps {
		parts = append(parts, prop.Description)
	}
	return strings.Join(parts, "\n")
}

// numericClaimsMultiset scans text with the existing package-scoped
// numericClaimRe (tools_schema_drift_test.go) and returns a multiset —
// lowercased "<kind> <value>" claim keyed by occurrence count — so two
// texts making the identical claims the identical number of times compare
// equal, and a duplicated or dropped claim does not.
func numericClaimsMultiset(text string) map[string]int {
	m := map[string]int{}
	for _, match := range numericClaimRe.FindAllStringSubmatch(text, -1) {
		key := strings.ToLower(match[1]) + " " + match[2]
		m[key]++
	}
	return m
}

// TestMCPResourceNumericClaimsMatchToolSchemas is GUARD-01's numeric half.
// It does NOT invent a third place where a number lives: it compares each
// resource file's numeric claims against the SAME tool's registered
// schema claims (description + jsonschema property descriptions), using
// the same numericClaimRe TestMCPToolSchemaNumericClaimsMatchEngineConstants
// already pins to internal/query's constants — so a resource claim is
// transitively pinned to the engine, with no second map to maintain.
//
// What this catches, and what it does not, stated plainly rather than left
// for a reader to discover: editing a resource file alone fails this test.
// Editing a tool description or its jsonschema tag alone fails this test.
// Editing internal/query/validate.go alone fails the EXISTING schema test
// (TestMCPToolSchemaNumericClaimsMatchEngineConstants), not this one —
// this test never reads validate.go. Editing both a constant and its
// schema consistently while leaving a resource file behind fails THIS
// test, because the schema side moves and the resource side does not. That
// enumeration is the honest statement of this test's coverage, not a claim
// that it alone closes every numeric-drift path.
func TestMCPResourceNumericClaimsMatchToolSchemas(t *testing.T) {
	toolStems := toolStemsFromNames(allToolNames())
	behaviorDocStems := []string{"tools-filter", "index-state"}

	compared := 0

	for _, stem := range toolStems {
		t.Run(stem, func(t *testing.T) {
			doc, err := resourcesFS.ReadFile("resources/" + stem + ".md")
			if err != nil {
				t.Fatalf("read resources/%s.md: %v", stem, err)
			}
			docClaims := numericClaimsMultiset(string(doc))

			schema := toolSchemaClaimsFor(t, stem)
			schemaClaims := numericClaimsMultiset(schemaClaimsText(schema))

			compared += len(docClaims) + len(schemaClaims)

			if !maps.Equal(docClaims, schemaClaims) {
				t.Errorf("resources/%s.md's numeric claims %v do not match its tool schema's claims %v", stem, docClaims, schemaClaims)
			}
		})
	}

	// The 2 behavior docs have no tool schema of their own to compare
	// against — asserting an empty multiset here closes that gap rather
	// than leaving it silently unchecked, so a free-floating number cannot
	// hide in the two files this comparison structurally cannot reach.
	for _, stem := range behaviorDocStems {
		t.Run(stem, func(t *testing.T) {
			doc, err := resourcesFS.ReadFile("resources/" + stem + ".md")
			if err != nil {
				t.Fatalf("read resources/%s.md: %v", stem, err)
			}
			claims := numericClaimsMultiset(string(doc))
			if len(claims) != 0 {
				t.Errorf("resources/%s.md is a behavior doc with no tool schema to compare against, but carries numeric claim(s) %v — nothing pins these to a source", stem, claims)
			}
		})
	}

	if compared == 0 {
		t.Fatal("compared zero numeric claims across every resource file and tool schema — this test would verify nothing")
	}
}

// countClaim is one integer-plus-noun count claim countClaimsIn extracted
// from a document — e.g. "7 companion tools" yields {noun: "companion",
// value: 7}.
type countClaim struct {
	noun  string // "companion" or "tool"
	value int
}

// countClaimsInRe matches an integer followed by whitespace and either a
// "companion(s)" or "tool(s)" noun, case-insensitively and tolerant of the
// noun's optional plural. Alternation order matters: "companions?" is
// tried before "tools?", so a phrase like "7 companion tools" binds to the
// companion rule only — the integer is adjacent to "companion", not to
// "tools" two words later — which is exactly the boundary
// TestResourceCountCheckerIsNotVacuous pins so a later regex edit cannot
// silently reclassify it.
var countClaimsInRe = regexp.MustCompile(`(?i)\b(\d+)\s+(companions?|tools?)\b`)

// countClaimsIn returns every integer-plus-noun count claim in doc, each
// tagged with which noun it matched.
func countClaimsIn(doc string) []countClaim {
	var claims []countClaim
	for _, m := range countClaimsInRe.FindAllStringSubmatch(doc, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		noun := "tool"
		if strings.HasPrefix(strings.ToLower(m[2]), "companion") {
			noun = "companion"
		}
		claims = append(claims, countClaim{noun: noun, value: n})
	}
	return claims
}

// TestResourceCountCheckerIsNotVacuous proves countClaimsIn discriminates
// rather than, say, always returning the correct answer regardless of what
// the document actually says (T-05-06) — including the companion/tool
// binding boundary its own doc comment describes.
func TestResourceCountCheckerIsNotVacuous(t *testing.T) {
	cases := []struct {
		name string
		doc  string
		want []countClaim
	}{
		{"no claim", "codegraph indexes your repo.", nil},
		{
			"one correct companion claim",
			fmt.Sprintf("narrows to %d companions.", len(companionNames)),
			[]countClaim{{noun: "companion", value: len(companionNames)}},
		},
		{
			"one wrong companion claim",
			"narrows to 3 companions.",
			[]countClaim{{noun: "companion", value: 3}},
		},
		{
			"one tool claim",
			fmt.Sprintf("%d tools are registered.", len(allToolNames())),
			[]countClaim{{noun: "tool", value: len(allToolNames())}},
		},
		{
			"companion-and-tool phrase binds to companion only",
			fmt.Sprintf("all %d companion tools register, for %d tools in total.", len(companionNames), len(allToolNames())),
			[]countClaim{{noun: "companion", value: len(companionNames)}, {noun: "tool", value: len(allToolNames())}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := countClaimsIn(tc.doc)
			if !slices.Equal(got, tc.want) {
				t.Errorf("countClaimsIn(%q) = %v, want %v", tc.doc, got, tc.want)
			}
		})
	}
}

// TestResourceCountClaimsMatchSourceSets is GUARD-01's count half: every
// companion-tagged claim found anywhere in the embedded resources must
// equal len(companionNames), and every tool-tagged claim must equal
// len(allToolNames()) — both derived lengths, never a hand-typed number.
func TestResourceCountClaimsMatchSourceSets(t *testing.T) {
	entries, err := fs.ReadDir(resourcesFS, "resources")
	if err != nil {
		t.Fatalf("read embedded resources dir: %v", err)
	}

	companionClaims := 0
	toolClaims := 0

	for _, entry := range entries {
		data, err := resourcesFS.ReadFile("resources/" + entry.Name())
		if err != nil {
			t.Fatalf("read resources/%s: %v", entry.Name(), err)
		}
		for _, claim := range countClaimsIn(string(data)) {
			switch claim.noun {
			case "companion":
				companionClaims++
				if claim.value != len(companionNames) {
					t.Errorf("%s claims %d companion tool(s), but len(companionNames) is %d", entry.Name(), claim.value, len(companionNames))
				}
			case "tool":
				toolClaims++
				if claim.value != len(allToolNames()) {
					t.Errorf("%s claims %d tool(s) in total, but len(allToolNames()) is %d", entry.Name(), claim.value, len(allToolNames()))
				}
			}
		}
	}

	if companionClaims == 0 {
		t.Fatal("found zero companion-count claims across every resource file — this test would verify nothing about the companion count")
	}
	if toolClaims == 0 {
		t.Fatal("found zero tool-count claims across every resource file — this test would verify nothing about the tool count")
	}
}

// envVarTokenRe matches a CODEGRAPH_-prefixed environment-variable-shaped
// token: upper-case letters, digits, and underscores following the
// literal prefix.
var envVarTokenRe = regexp.MustCompile(`CODEGRAPH_[A-Z0-9_]+`)

// TestResourceEnvVarNamesAreReal is GUARD-01's env-var half: the only
// environment variable resource content is permitted to name is
// allowlistEnvName (instructions_contract_test.go's own literal, reused
// here rather than re-typed). The rule is deliberately absolute — a doc
// that starts describing some other variable fails until someone decides
// that variable is real.
func TestResourceEnvVarNamesAreReal(t *testing.T) {
	entries, err := fs.ReadDir(resourcesFS, "resources")
	if err != nil {
		t.Fatalf("read embedded resources dir: %v", err)
	}

	found := 0
	for _, entry := range entries {
		data, err := resourcesFS.ReadFile("resources/" + entry.Name())
		if err != nil {
			t.Fatalf("read resources/%s: %v", entry.Name(), err)
		}
		for _, tok := range envVarTokenRe.FindAllString(string(data), -1) {
			found++
			if tok != allowlistEnvName {
				t.Errorf("resources/%s names environment variable %q, which is not %s — resource content may only name the narrowing filter", entry.Name(), tok, allowlistEnvName)
			}
		}
	}

	if found == 0 {
		t.Fatal("found zero CODEGRAPH_-prefixed tokens across every resource file — this test would verify nothing")
	}
}

// hostFactsRootDirs are the machine-specific POSIX root directory segments
// that make an absolute path a host fact rather than a portable statement
// about repository-relative structure.
var hostFactsRootDirs = []string{"users", "home", "var", "tmp", "opt", "etc", "private"}

// hostFactsPosixRe matches an absolute POSIX path rooted at one of
// hostFactsRootDirs, mirroring server.go's own no-interpolation rule for
// the instructions constant (T-05-01): "no repository path, no resolved
// index root, no hostname, no environment value."
var hostFactsPosixRe = regexp.MustCompile(`(?i)/(?:` + strings.Join(hostFactsRootDirs, "|") + `)(?:/\S*)?\b`)

// hostFactsWindowsRe matches a Windows drive-letter path.
var hostFactsWindowsRe = regexp.MustCompile(`\b[A-Za-z]:\\\S*`)

// hostFactsIn returns every host-specific path found in doc: every
// occurrence hostFactsPosixRe and hostFactsWindowsRe each match.
func hostFactsIn(doc string) []string {
	var found []string
	found = append(found, hostFactsPosixRe.FindAllString(doc, -1)...)
	found = append(found, hostFactsWindowsRe.FindAllString(doc, -1)...)
	return found
}

// TestResourceHostFactCheckerIsNotVacuous proves hostFactsIn discriminates:
// one crafted input per shape it claims to detect, plus one realistic
// clean fact-sheet it must leave alone.
func TestResourceHostFactCheckerIsNotVacuous(t *testing.T) {
	cases := []struct {
		name    string
		doc     string
		wantAny bool
	}{
		{"posix users path", "Index root: /Users/alice/repo/.codegraph", true},
		{"posix tmp path", "Wrote scratch data to /tmp/codegraph-scratch", true},
		{"posix private path (macOS /tmp symlink target)", "Resolved to /private/tmp/codegraph-scratch", true},
		{"windows drive path", `Index root: C:\Users\alice\repo\.codegraph`, true},
		{"clean fact-sheet", "Show a symbol's signature, calls, and callers, or a line-numbered file read.", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hostFactsIn(tc.doc)
			if tc.wantAny && len(got) == 0 {
				t.Errorf("hostFactsIn(%q) = %v, want at least one finding", tc.doc, got)
			}
			if !tc.wantAny && len(got) != 0 {
				t.Errorf("hostFactsIn(%q) = %v, want none", tc.doc, got)
			}
		})
	}
}

// TestResourceContentCarriesNoHostFacts is T-05-01's mitigation: no
// resource file may carry an absolute machine-specific POSIX path or a
// Windows drive-letter path, turning server.go's convention for the
// instructions constant into an enforced gate for resource content too.
func TestResourceContentCarriesNoHostFacts(t *testing.T) {
	entries, err := fs.ReadDir(resourcesFS, "resources")
	if err != nil {
		t.Fatalf("read embedded resources dir: %v", err)
	}
	for _, entry := range entries {
		data, err := resourcesFS.ReadFile("resources/" + entry.Name())
		if err != nil {
			t.Fatalf("read resources/%s: %v", entry.Name(), err)
		}
		if found := hostFactsIn(string(data)); len(found) > 0 {
			t.Errorf("resources/%s carries host-specific path(s) %v — no resource file may interpolate machine-specific filesystem layout", entry.Name(), found)
		}
	}
}
