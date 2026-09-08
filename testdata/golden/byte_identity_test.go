// testdata/golden/byte_identity_test.go
//
// This file is the byte-identity oracle 01-CONTEXT.md's D-04 assumed
// already existed and does not: it re-invokes the live Engine.Node/
// Engine.Explore (and, for the -mcp goldens, the real MCP adapter via
// internal/goldenspec) against every one of the 26 frozen golden pairs and
// asserts byte-for-byte equality against the committed Output field. It
// runs against UNMODIFIED Node()/Explore() (this plan, 01-02, is Wave-0 —
// it runs before plan 01-04's extraction) so the oracle's own correctness
// is established while the code under test is known-good.
//
// TestReFrozenGoldensValid (golden_test.go) is NOT superseded by this file
// — it guards a different property (envelope validity: exists, non-empty,
// parses) and both must keep passing.
package golden

import (
	"fmt"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/goldenspec"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// behavioralMultiSymbol and behavioralMultiQuery are the committed
// symbol/query values corpus/behavioral's two goldens
// (go-node-multi.json, go-explore-multi.json) were captured with. Unlike
// the four locked corpora, the behavioral corpus has no entry in
// goldenspec.LockedCorpusArgs — it is the in-repo, always-available corpus
// (D-03), not one of the four locked ones — so its two values are pinned
// here directly. Source: testdata/golden/gocapture/main.go's
// behavioralCorpusSpec (multiSymbol: "Validate", multiQuery: "user
// account").
const (
	behavioralMultiSymbol = "Validate"
	behavioralMultiQuery  = "user account"
)

// slugCanonicalLanguage picks one canonical language per locked slug for
// resolving that slug's source directory through the existing
// lockedCorpusDir(t, language) resolver, which is keyed by language (H3:
// hugo's "go" and "tsjs" languages both resolve to the same hugo tree, so
// either works; "go" is used here for readability). This is NOT a second
// copy of goldenspec.LanguageToLockedSlug — it is that map's inverse,
// scoped to picking a single resolvable language per slug for this file's
// own directory lookups, not a second table of locked capture arguments.
var slugCanonicalLanguage = map[string]string{
	"hugo":     "go",
	"guava":    "java",
	"serilog":  "csharp",
	"requests": "python",
}

// invocationKind names which of the four call shapes a goldenIdentityCase
// drives, so dispatch is a switch on data rather than a filename-shaped
// guess.
type invocationKind int

const (
	kindEngineNode invocationKind = iota
	kindEngineExplore
	kindMCPNode
	kindMCPExplore
)

func (k invocationKind) String() string {
	switch k {
	case kindEngineNode:
		return "Engine-Node"
	case kindEngineExplore:
		return "Engine-Explore"
	case kindMCPNode:
		return "MCP-Node"
	case kindMCPExplore:
		return "MCP-Explore"
	default:
		return fmt.Sprintf("invocationKind(%d)", int(k))
	}
}

// goldenIdentityCase is one (corpus, golden file) pair enumerated from
// expectedGoCaptures (golden_test.go's authoritative 26-pair table) joined
// against goldenspec.LockedCorpusArgs (or the behavioral constants above),
// plus the invocation kind and its arguments.
type goldenIdentityCase struct {
	slug     string
	name     string
	kind     invocationKind
	symbol   string
	file     string
	query    string
	maxFiles int
}

// goldenIdentityCases builds the full, enumerate-before-run case list by
// joining expectedGoCaptures's (slug, filename) pairs to
// goldenspec.LockedCorpusArgs (H3: the SAME table the generator used,
// never a re-derived copy). It Fatals on any filename this table does not
// recognize rather than silently skipping it, so a golden added to
// expectedGoCaptures without a matching case here fails loudly instead of
// shrinking the attempted count.
func goldenIdentityCases(t *testing.T) []goldenIdentityCase {
	t.Helper()

	var cases []goldenIdentityCase
	for _, cs := range expectedGoCaptures {
		for _, name := range cs.files {
			c := goldenIdentityCase{slug: cs.slug, name: name}
			if cs.slug == "behavioral" {
				switch name {
				case "go-node-multi.json":
					c.kind = kindEngineNode
					c.symbol = behavioralMultiSymbol
				case "go-explore-multi.json":
					c.kind = kindEngineExplore
					c.query = behavioralMultiQuery
					c.maxFiles = 0
				default:
					t.Fatalf("goldenIdentityCases: unexpected behavioral golden %q — no case shape known for it", name)
				}
				cases = append(cases, c)
				continue
			}

			args, ok := goldenspec.LockedCorpusArgs[cs.slug]
			if !ok {
				t.Fatalf("goldenIdentityCases: no goldenspec.LockedCorpusArgs entry for slug %q", cs.slug)
			}
			switch name {
			case "go-node.json":
				c.kind = kindEngineNode
				c.symbol = args.BaselineSymbol
				c.file = args.BaselineSymbolFile
			case "go-node-multi.json":
				c.kind = kindEngineNode
				c.symbol = args.MultiSymbol
			case "go-node-mcp.json":
				c.kind = kindMCPNode
				c.symbol = args.BaselineSymbol
			case "go-explore.json":
				c.kind = kindEngineExplore
				c.query = args.BaselineQuery
				c.maxFiles = 1
			case "go-explore-multi.json":
				c.kind = kindEngineExplore
				c.query = args.MultiQuery
				c.maxFiles = 0
			case "go-explore-mcp.json":
				c.kind = kindMCPExplore
				c.query = args.BaselineQuery
			default:
				t.Fatalf("goldenIdentityCases: unexpected golden filename %q for slug %q — no case shape known for it", name, cs.slug)
			}
			cases = append(cases, c)
		}
	}
	return cases
}

// corpusSourceDir resolves slug's indexable source directory: the
// committed behavioral corpus for "behavioral", or the locked corpus tree
// (through the existing, never-skip lockedCorpusDir resolver) for the four
// locked slugs.
func corpusSourceDir(t *testing.T, slug string) string {
	t.Helper()
	if slug == "behavioral" {
		return behavioralCorpusSrc(t)
	}
	lang, ok := slugCanonicalLanguage[slug]
	if !ok {
		t.Fatalf("corpusSourceDir: no canonical language mapped for slug %q", slug)
	}
	return lockedCorpusDir(t, lang)
}

// buildEngineAndDirAt is buildEngineAt (behavioral_test.go) plus the
// indexed directory it built the Engine from, so an MCP case can be driven
// through goldenspec.CallNodeViaMCP/CallExploreViaMCP against the SAME
// copied-and-indexed tree the Engine-Node/Engine-Explore cases for that
// same corpus use — not a second, independently-copied tree. Constructing
// the Engine any other way reintroduces the CR-02 pollution bug documented
// on buildEngineAt.
//
// The returned closer is NOT registered via t.Cleanup: the caller must
// close it before driving any MCP case against the same directory.
// query.OpenAt's underlying store open is exclusive — TestExploreCLIMatchesMCP
// and TestNodeCLIMatchesMCP (behavioral_test.go) already establish this
// pattern, closing the CLI-side Engine before calling the MCP helpers
// against the same on-disk index. Skipping that ordering here reproduces
// exactly the "returned an error result" failure this comment now
// prevents (found empirically running this test during 01-02).
func buildEngineAndDirAt(t *testing.T, sourceDir string) (*query.Engine, string, func()) {
	t.Helper()

	dst := buildIndexedFixture(t, sourceDir)
	eng, closer, err := query.OpenAt(dst)
	if err != nil {
		t.Fatalf("OpenAt(%s): %v", dst, err)
	}
	closeOnce := func() { _ = closer.Close() }
	return eng, dst, closeOnce
}

// loadGoldenOutputForCase loads c's frozen golden Output, resolving the
// behavioral corpus's different on-disk location (repo-root corpus/
// behavioral/) separately from the locked corpora's (testdata/golden/
// corpus/<slug>/), exactly as TestReFrozenGoldensValid does.
func loadGoldenOutputForCase(t *testing.T, c goldenIdentityCase) string {
	t.Helper()
	if c.slug == "behavioral" {
		return loadBehavioralFixture(t, c.name)
	}
	return loadGoldenOutputIn(t, c.slug, c.name)
}

// compareGoldenOutput is the ONE comparison helper both
// TestGoldensMatchLiveEngineOutput and its non-vacuity companion call, so
// the non-vacuity proof exercises the exact code path the real test does.
// Comparison is raw string equality — no trimming, no whitespace
// normalisation, no line-ending rewriting, no volatile-field stripping;
// the captures are already volatile-field-free by construction. On a
// mismatch it reports the first differing byte offset, a bounded context
// window either side, and both lengths, rather than a full-file dump.
func compareGoldenOutput(want, got string) (equal bool, firstDiff int, detail string) {
	if want == got {
		return true, -1, ""
	}

	n := len(want)
	if len(got) < n {
		n = len(got)
	}
	diff := n
	for i := 0; i < n; i++ {
		if want[i] != got[i] {
			diff = i
			break
		}
	}

	const window = 60
	lo := diff - window
	if lo < 0 {
		lo = 0
	}
	wantHi := diff + window
	if wantHi > len(want) {
		wantHi = len(want)
	}
	gotHi := diff + window
	if gotHi > len(got) {
		gotHi = len(got)
	}
	return false, diff, fmt.Sprintf(
		"first differing byte at offset %d (want len=%d, got len=%d)\nwant[%d:%d]=%q\ngot[%d:%d]=%q",
		diff, len(want), len(got), lo, wantHi, want[lo:wantHi], lo, gotHi, got[lo:gotHi])
}

// TestGoldensMatchLiveEngineOutput is the byte-identity oracle over all 26
// frozen golden pairs. It enumerates the full case list BEFORE any subtest
// runs (attempted := len(cases), asserted == 26 immediately — rule
// 84d1gfpywd: a suite that enumerated nothing must fail before it can read
// green), then drives one Engine+directory build per corpus (not per
// golden pair) and runs one t.Run subtest per pair. attempted, completed
// and matched are derived from a parent-owned results slice that each
// subtest writes into its OWN index — never a counter incremented from
// inside a subtest, which would silently under-count on a t.Fatal.
func TestGoldensMatchLiveEngineOutput(t *testing.T) {
	cases := goldenIdentityCases(t)
	attempted := len(cases)
	if attempted != 26 {
		t.Fatalf("goldenIdentityCases enumerated %d cases, want exactly 26 — the enumerator itself is broken or expectedGoCaptures shrank", attempted)
	}

	// Independently assert the flattened expectedGoCaptures total is also
	// 26: the two checks corroborate each other rather than one
	// constant restating itself.
	flattenedTotal := 0
	for _, cs := range expectedGoCaptures {
		flattenedTotal += len(cs.files)
	}
	if flattenedTotal != 26 {
		t.Fatalf("expectedGoCaptures flattened total = %d, want 26", flattenedTotal)
	}

	type verdict struct {
		completed bool
		matched   bool
	}
	results := make([]verdict, len(cases))

	// Group case indices by slug, in first-appearance order, so each
	// corpus is indexed exactly once regardless of how many golden pairs
	// it contributes.
	var slugOrder []string
	indicesBySlug := map[string][]int{}
	for i, c := range cases {
		if _, seen := indicesBySlug[c.slug]; !seen {
			slugOrder = append(slugOrder, c.slug)
		}
		indicesBySlug[c.slug] = append(indicesBySlug[c.slug], i)
	}

	for _, slug := range slugOrder {
		src := corpusSourceDir(t, slug)
		eng, dir, closeEngine := buildEngineAndDirAt(t, src)

		// Run the Engine-kind cases first, against the still-open eng,
		// then close it BEFORE any MCP-kind case: query.OpenAt's
		// underlying store open is exclusive, and internal/goldenspec's
		// MCP helpers each open the SAME on-disk directory again via the
		// MCP server's own handlers. Holding eng open across an MCP call
		// against the same directory produces a spurious
		// "returned an error result" rather than a real byte-identity
		// finding — TestExploreCLIMatchesMCP/TestNodeCLIMatchesMCP
		// (behavioral_test.go) already establish this close-before-MCP
		// ordering for exactly this reason.
		var mcpIndices []int
		for _, i := range indicesBySlug[slug] {
			c := cases[i]
			if c.kind == kindMCPNode || c.kind == kindMCPExplore {
				mcpIndices = append(mcpIndices, i)
				continue
			}
			t.Run(c.slug+"/"+c.name, func(t *testing.T) {
				want := loadGoldenOutputForCase(t, c)

				var got string
				var err error
				switch c.kind {
				case kindEngineNode:
					got, err = eng.Node(c.symbol, c.file, nil)
				case kindEngineExplore:
					got, err = eng.Explore(c.query, c.maxFiles)
				default:
					t.Fatalf("%s/%s: unknown invocation kind %v", c.slug, c.name, c.kind)
				}
				if err != nil {
					t.Fatalf("%s/%s: %s invocation failed: %v", c.slug, c.name, c.kind, err)
				}

				equal, _, detail := compareGoldenOutput(want, got)
				results[i] = verdict{completed: true, matched: equal}
				if !equal {
					t.Errorf("%s/%s: live output diverges from the frozen golden (%s):\n%s", c.slug, c.name, c.kind, detail)
				}
			})
		}

		closeEngine()

		for _, i := range mcpIndices {
			c := cases[i]
			t.Run(c.slug+"/"+c.name, func(t *testing.T) {
				want := loadGoldenOutputForCase(t, c)

				var got string
				var err error
				switch c.kind {
				case kindMCPNode:
					got, err = goldenspec.CallNodeViaMCP(dir, c.symbol)
				case kindMCPExplore:
					got, err = goldenspec.CallExploreViaMCP(dir, c.query)
				default:
					t.Fatalf("%s/%s: unknown invocation kind %v", c.slug, c.name, c.kind)
				}
				if err != nil {
					t.Fatalf("%s/%s: %s invocation failed: %v", c.slug, c.name, c.kind, err)
				}

				equal, _, detail := compareGoldenOutput(want, got)
				results[i] = verdict{completed: true, matched: equal}
				if !equal {
					t.Errorf("%s/%s: live output diverges from the frozen golden (%s):\n%s", c.slug, c.name, c.kind, detail)
				}
			})
		}
	}

	completed := 0
	matched := 0
	for _, r := range results {
		if r.completed {
			completed++
			if r.matched {
				matched++
			}
		}
	}
	t.Logf("attempted=%d completed=%d matched=%d of 26", attempted, completed, matched)
	if attempted != 26 || completed != 26 || matched != 26 {
		t.Fatalf("attempted=%d completed=%d matched=%d of 26 (want 26/26/26)", attempted, completed, matched)
	}
}

// TestGoldensMatchLiveEngineOutputIsNonVacuous is the permanent companion
// guard: it does not mutate production code. Instead it proves the
// comparison mechanism itself is live by taking one real golden's Output,
// perturbing a copy two different ways, and asserting compareGoldenOutput
// — the SAME helper the real test calls — reports a mismatch with the
// exact first-differing offset in both cases. This is what keeps the
// non-vacuity property observable after the hand-run mutation proof
// (recorded in the SUMMARY) scrolls out of history.
func TestGoldensMatchLiveEngineOutputIsNonVacuous(t *testing.T) {
	base := loadGoldenOutputIn(t, "hugo", "go-node.json")
	if base == "" {
		t.Fatal("baseline golden output is empty; cannot perturb it")
	}

	t.Run("appended-byte", func(t *testing.T) {
		mutated := base + "X"
		equal, firstDiff, detail := compareGoldenOutput(base, mutated)
		if equal {
			t.Fatal("compareGoldenOutput reported equal for an appended-byte mutation — the oracle cannot detect this class of change")
		}
		wantOffset := len(base)
		if firstDiff != wantOffset {
			t.Fatalf("first differing offset = %d, want %d\n%s", firstDiff, wantOffset, detail)
		}
	})

	t.Run("flipped-interior-byte", func(t *testing.T) {
		if len(base) == 0 {
			t.Fatal("baseline golden output is empty; cannot flip an interior byte")
		}
		mid := len(base) / 2
		b := []byte(base)
		if b[mid] == 'X' {
			b[mid] = 'Y'
		} else {
			b[mid] = 'X'
		}
		mutated := string(b)

		equal, firstDiff, detail := compareGoldenOutput(base, mutated)
		if equal {
			t.Fatal("compareGoldenOutput reported equal for a flipped-interior-byte mutation — the oracle cannot detect this class of change")
		}
		if firstDiff != mid {
			t.Fatalf("first differing offset = %d, want %d\n%s", firstDiff, mid, detail)
		}
	})
}
