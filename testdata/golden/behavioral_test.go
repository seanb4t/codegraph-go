// testdata/golden/behavioral_test.go
//
// This file holds the behavioral harness: it runs the production
// indexer.Run pipeline over the committed, always-in-repo behavioral corpus
// (corpus/behavioral, D-03), drives internal/query.Engine for the traced
// behavioral surfaces (explore, node), and asserts NAMED behavioral
// properties of the live output in the D-09 style — overloaded-def
// enumeration, multi-word tokenization, the file-relevance gate's
// connected-non-test preference, and structural-surfacing. Goldens (the
// go-*.json fixtures) remain committed as regression snapshots but are NOT
// the primary oracle: a failing test names which behavior broke, not merely
// "a golden diff appeared".
//
// The TS-era capture path and the external, network-fetched corpora this
// harness formerly diffed against are gone as of this phase (FIXT-04); the
// behavioral corpus is the sole committed, in-repo source. TestCorpusBehavior_Go
// is the "go" matrix gate and stays green as a pure property test over it.
package golden

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/seanb4t/codegraph-go/internal/corpora"
	"github.com/seanb4t/codegraph-go/internal/indexer"
	internalmcp "github.com/seanb4t/codegraph-go/internal/mcp"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// languageToLockedSlug is the EXPLICIT committed language->locked-slug map (H3),
// shared with gocapture. hugo supplies the tsjs leg from its JS files even
// though its manifest language is "go".
var languageToLockedSlug = map[string]string{
	"go":     "hugo",
	"tsjs":   "hugo",
	"java":   "guava",
	"csharp": "serilog",
	"python": "requests",
}

// slugToRepo maps each locked-slug to its manifest repo slug for lookup.
var slugToRepo = map[string]string{
	"hugo":     "gohugoio/hugo",
	"guava":    "google/guava",
	"serilog":  "serilog/serilog",
	"requests": "psf/requests",
}

// lockedCorpusDir is the single hermetic resolver for locked corpus directories
// (D-10, constraint 2). It loads the manifest via internal/corpora, resolves the
// language's locked slug through the language map, and calls e.Dir(CorpusRoot())
// to locate the fetched tree. On any failure — map missing for a priority
// language, manifest unreadable, no locked entry for the slug, or the resolved
// dir is not a directory — it calls t.Fatalf with the named cause. It NEVER
// reads an env-var default, NEVER uses t.Skip, and NEVER matches on
// e.Language directly (which would miss tsjs's manifest language "go").
func lockedCorpusDir(t *testing.T, language string) string {
	t.Helper()

	slug, ok := languageToLockedSlug[language]
	if !ok {
		t.Fatalf("lockedCorpusDir(%q): no slug found in language map", language)
	}
	repo, ok := slugToRepo[slug]
	if !ok {
		t.Fatalf("lockedCorpusDir(%q): slug %q has no repo mapping", language, slug)
	}

	m, err := corpora.Load(filepath.Join("..", "..", "corpora", "manifest.json"))
	if err != nil {
		t.Fatalf("lockedCorpusDir(%q): load manifest: %v", language, err)
	}

	var entry *corpora.Entry
	for _, e := range corpora.LockedEntries(m) {
		if e.Repo == repo {
			entry = &e
			break
		}
	}
	if entry == nil {
		t.Fatalf("lockedCorpusDir(%q): no locked entry found for repo %q", language, repo)
	}

	root, err := corpora.CorpusRoot()
	if err != nil {
		t.Fatalf("lockedCorpusDir(%q): CorpusRoot: %v", language, err)
	}

	dir := entry.Dir(root)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Fatalf("lockedCorpusDir(%q): locked tree directory %s not found or not a directory: %v; run 'task corpora:fetch'", language, dir, err)
	}
	return dir
}

// mbShapeRE pins D-07's MB-rendering contract, where
// dbSizeBytes comes from a real corpus index rather than a synthetic
// fixture value: fmt.Sprintf("%.2f MB", bytes/1024/1024).
var mbShapeRE = regexp.MustCompile(`^\d+\.\d{2} MB$`)

// findVolatileKeysExcept wraps golden_test.go's findVolatileKeys (shared
// with TestGoldenFixturesExist, which correctly continues to enforce that
// the frozen oracle fixtures in corpus/*/*.json never re-include a
// volatile key such as dbSizeBytes) with a named exemption list, used
// ONLY at this file's own-output call site. It never mutates the shared
// volatileKeys map in golden_test.go — that map keeps governing the
// frozen corpus fixtures, and must keep failing if a frozen golden ever
// re-includes dbSizeBytes (D-08, RESEARCH Pitfall 2).
func findVolatileKeysExcept(v interface{}, path string, except ...string) []string {
	exempt := make(map[string]bool, len(except))
	for _, k := range except {
		exempt[k] = true
	}
	var kept []string
	for _, k := range findVolatileKeys(v, path) {
		// findVolatileKeys reports dotted paths like "our status.json.dbSizeBytes";
		// match on the trailing key segment against the exempt set.
		leaf := k
		if i := strings.LastIndexByte(k, '.'); i >= 0 {
			leaf = k[i+1:]
		}
		if exempt[leaf] {
			continue
		}
		kept = append(kept, k)
	}
	return kept
}

// behavioralCase is one entry in corpus/behavioral/CASES.json's case map.
type behavioralCase struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	What      string   `json:"what"`
	Symbol    string   `json:"symbol"`
	ZeroEdge  string   `json:"zero_edge,omitempty"`
	Connected []string `json:"connected,omitempty"`
	Files     []string `json:"files"`
	Query     string   `json:"query"`
	Assertion string   `json:"assertion"`
	Command   string   `json:"command"`
}

// behavioralCases is the top-level CASES.json envelope.
type behavioralCasesDoc struct {
	Cases []behavioralCase `json:"cases"`
}

// loadBehavioralCases reads corpus/behavioral/CASES.json and returns its
// case list. It Fatals on any I/O or parse error (the corpus is committed
// in-repo, so a missing or malformed CASES.json is a real failure, never
// a skip).
func loadBehavioralCases(t *testing.T) []behavioralCase {
	t.Helper()

	path := filepath.Join("..", "..", "corpus", "behavioral", "CASES.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read CASES.json: %v", err)
	}
	var doc behavioralCasesDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal CASES.json: %v", err)
	}
	if len(doc.Cases) == 0 {
		t.Fatal("CASES.json has an empty case list")
	}
	return doc.Cases
}

// loadBehavioralFixture reads a golden envelope from corpus/behavioral/<name>
// and returns its output field. The path is resolved from testdata/golden/
// working directory up to the repo root via two ".." hops.
func loadBehavioralFixture(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "corpus", "behavioral", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read behavioral golden %s: %v", path, err)
	}
	var capture goldenCapture
	if err := json.Unmarshal(data, &capture); err != nil {
		t.Fatalf("unmarshal behavioral golden %s: %v", path, err)
	}
	if capture.Output == "" {
		t.Fatalf("behavioral golden %s has an empty output field", path)
	}
	return capture.Output
}

// buildEngineAt is the shared corpus-agnostic engine builder (plan 17,
// TEST-01/D-02): it copies sourceDir into a fresh t.TempDir() and indexes
// it on disk at <dst>/.codegraph/store (via buildIndexedFixture), then
// opens it via OpenAt exactly as production does. Used by the D-02
// behavioral harness to build a live Go engine over a corpus source tree.
//
// ★ CR-02 fix: this USED TO run indexer.Run directly into a bare
// t.TempDir() and wrap it via NewWithRoot(reader, sourceDir) — an Engine
// whose repoRoot (sourceDir, e.g. a developer checkout) had NOTHING to
// do with the store the test actually built (a sibling, unrelated temp
// dir). Status()'s dbSizeBytes derives its directory from
// repoRoot+".codegraph/store" purely by convention (status.go), so that
// assertion measured whatever HAPPENED to exist at
// <sourceDir>/.codegraph/store — on the reviewer's machine, a stale
// developer-created sibling `.codegraph/store` from an old `codegraph init`,
// invisible pollution that made the assertion pass locally while failing
// on any clean checkout (`dbSizeBytes = 0, want a positive integer`).
// Opening through OpenAt(dst) makes repoRoot the SAME directory the store
// was written to, so the assertion is honest by construction — no
// filesystem pollution required to pass, and none possible to cause a
// false pass.
func buildEngineAt(t *testing.T, sourceDir string) *query.Engine {
	t.Helper()

	dst := buildIndexedFixture(t, sourceDir)
	eng, closer, err := query.OpenAt(dst)
	if err != nil {
		t.Fatalf("OpenAt(%s): %v", dst, err)
	}
	t.Cleanup(func() { _ = closer.Close() })
	return eng
}

// behavioralCorpusSrc resolves the committed, in-repo behavioral corpus
// source tree at corpus/behavioral/src (D-03).
func behavioralCorpusSrc(t *testing.T) string {
	t.Helper()

	src, err := filepath.Abs(filepath.Join("..", "..", "corpus", "behavioral", "src"))
	if err != nil {
		t.Fatalf("resolve behavioral corpus source path: %v", err)
	}
	if info, err := os.Stat(src); err != nil || !info.IsDir() {
		t.Fatalf("behavioral corpus source not found at %s (err=%v)", src, err)
	}
	return src
}

// copyDir recursively copies src into dst, mirroring
// internal/mcp/server_test.go's copyFixture helper (unexported there,
// reimplemented here since this file lives in the external golden
// package) — used so the CLI==MCP byte-identity harness (Task 3) can
// index a corpus onto a fresh on-disk location without mutating the
// original checkout.
//
// WR-03 (02-REVIEW-2.md): skips any ".codegraph" directory entirely,
// rather than copying it verbatim. buildIndexedFixture's whole purpose is
// building a store the test itself indexes from a KNOWN, empty starting
// point — but src is often a real developer checkout (e.g.
// corpus/behavioral/src), and a stray
// `codegraph init` run against it at any point in that machine's history
// leaves a live Pebble store sitting right there in the source tree.
// indexer.Run MERGES into an existing store rather than replacing it
// (internal/cli/index.go's os.RemoveAll owns the wipe, not the indexer
// itself), so copying that store verbatim and then indexing "into" it
// would silently resurrect however many stale files/nodes/languages the
// inherited store happens to contain — proven by the reviewer with a
// 1-Go-file fixture that reported fileCount=69, nodeCount=762. Skipping
// .codegraph/ at the copy makes every fixture this function builds
// pollution-proof BY CONSTRUCTION: there is nothing to inherit, so there
// is nothing to clear.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()

	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == ".codegraph" {
			return fs.SkipDir
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		// Handle symlinks: WalkDir uses Lstat and does NOT walk into symlink
		// targets that are directories; it reports the symlink itself as a
		// non-directory entry. Resolve and copy content accordingly.
		if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			if rInfo, err := os.Stat(resolved); err == nil && rInfo.IsDir() {
				if err := os.MkdirAll(target, 0o755); err != nil {
					return err
				}
				return filepath.WalkDir(resolved, func(rPath string, rD fs.DirEntry, rErr error) error {
					if rErr != nil {
						return rErr
					}
					rRel, rErr := filepath.Rel(resolved, rPath)
					if rErr != nil {
						return rErr
					}
					rTarget := filepath.Join(target, rRel)
					if rD.IsDir() {
						return os.MkdirAll(rTarget, 0o755)
					}
					data, rErr := os.ReadFile(rPath)
					if rErr != nil {
						return rErr
					}
					return os.WriteFile(rTarget, data, 0o644)
				})
			}
			data, err := os.ReadFile(resolved)
			if err != nil {
				return err
			}
			return os.WriteFile(target, data, 0o644)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copyDir(%s -> %s): %v", src, dst, err)
	}
}

// buildIndexedFixture copies src into a fresh t.TempDir() and indexes it
// on disk at <tempdir>/.codegraph/store (mirroring
// internal/mcp/server_test.go's copyFixture+indexFixture pattern), so
// BOTH query.OpenAt (the CLI's own code path, internal/cli/explore.go and
// node.go) and internalmcp.BuildServer (the MCP server's own code path,
// internal/mcp/tools.go) can resolve the SAME on-disk index from a real
// filesystem location — each opening its own fresh snapshot, exactly as
// production does — proving EXPL-05/NODE-04 end-to-end rather than merely
// calling the same Go function twice.
func buildIndexedFixture(t *testing.T, src string) string {
	t.Helper()

	dst := t.TempDir()
	copyDir(t, src, dst)

	storeDir := filepath.Join(dst, ".codegraph", "store")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatalf("mkdir store dir: %v", err)
	}
	if _, err := indexer.Run(dst, storeDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("index fixture at %s: %v", dst, err)
	}
	return dst
}

// TestBuildIndexedFixtureIgnoresInheritedStore is WR-03's regression pin
// (02-REVIEW-2.md): a src tree containing an UNRELATED, pre-existing
// .codegraph/store (exactly the "developer ran `codegraph init` in this
// checkout at some point" scenario copyDir used to copy verbatim) must
// not leak into the fixture buildIndexedFixture builds. src here has
// exactly ONE Go file; before the WR-03 fix, indexer.Run merged into the
// copied store and the reviewer proved this exact shape reports
// fileCount=69/nodeCount=762 — this test pins fileCount=1 instead.
func TestBuildIndexedFixtureIgnoresInheritedStore(t *testing.T) {
	src := t.TempDir()

	goFile := "package onefile\n\nfunc Hello() string { return \"hi\" }\n"
	if err := os.WriteFile(filepath.Join(src, "onefile.go"), []byte(goFile), 0o644); err != nil {
		t.Fatalf("write fixture source file: %v", err)
	}

	// Plant an UNRELATED, populated store at src/.codegraph/store — the
	// exact shape a stray `codegraph init` leaves behind. buildIndexedFixture
	// must never let indexer.Run merge into this.
	staleSrc := t.TempDir()
	for i := 0; i < 5; i++ {
		name := filepath.Join(staleSrc, fmt.Sprintf("stale%d.go", i))
		if err := os.WriteFile(name, []byte(fmt.Sprintf("package stale\n\nfunc Stale%d() {}\n", i)), 0o644); err != nil {
			t.Fatalf("write stale source file: %v", err)
		}
	}
	staleStoreDir := filepath.Join(src, ".codegraph", "store")
	if err := os.MkdirAll(staleStoreDir, 0o755); err != nil {
		t.Fatalf("mkdir stale store dir: %v", err)
	}
	if _, err := indexer.Run(staleSrc, staleStoreDir, indexer.Options{Quiet: true}); err != nil {
		t.Fatalf("build stale store: %v", err)
	}

	dst := buildIndexedFixture(t, src)

	eng, closer, err := query.OpenAt(dst)
	if err != nil {
		t.Fatalf("OpenAt(%s): %v", dst, err)
	}
	t.Cleanup(func() { _ = closer.Close() })

	got, err := eng.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if got.FileCount != 1 {
		t.Fatalf("WR-03 REGRESSION: status.FileCount = %d, want 1 — the fixture inherited an unrelated pre-existing .codegraph/store instead of indexing only its own 1-file source tree", got.FileCount)
	}
}

// goldenCapture mirrors the explore.json/node.json wrapper shape:
// {"command": "...", "output": "<markdown text>"}.
type goldenCapture struct {
	Command string `json:"command"`
	Output  string `json:"output"`
}

// loadGoldenFixtureIn decodes a golden/behavioral corpus JSON fixture from
// a named corpus directory under corpus/ (plan 17, D-02).
func loadGoldenFixtureIn[T any](t *testing.T, corpus, name string) T {
	t.Helper()

	path := filepath.Join("corpus", corpus, name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden fixture %s: %v", path, err)
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("unmarshal golden fixture %s: %v", path, err)
	}
	return v
}

// loadGoldenOutputIn is loadGoldenOutput generalized to any corpus
// directory (plan 17, D-02).
func loadGoldenOutputIn(t *testing.T, corpus, name string) string {
	t.Helper()

	capture := loadGoldenFixtureIn[goldenCapture](t, corpus, name)
	if capture.Output == "" {
		t.Fatalf("golden fixture %s/%s has an empty output field", corpus, name)
	}
	return capture.Output
}

// goldenFixtureExistsIn reports whether corpus/<corpus>/<name> exists —
// used to skip a subtest cleanly (not fail) when a corpus has no golden
// fixture of that kind (e.g. the behavioral corpus has no baseline
// explore.json/node.json — README.md: "behavioral-only... no
// baseline... fixtures").
func goldenFixtureExistsIn(corpus, name string) bool {
	_, err := os.Stat(filepath.Join("corpus", corpus, name))
	return err == nil
}

// extractDisclaimer pulls the "> ..." blockquote paragraph out of an
// explore markdown output (golden or rendered) — the text between "> " and
// the following blank line. Mirrors internal/query/render_markdown_test.go's
// helper of the same name/behavior (D-05a, T-03-06-Drift), reimplemented
// here because it operates on golden's own package boundary.
func extractDisclaimer(t *testing.T, output string) string {
	t.Helper()

	marker := "> "
	start := strings.Index(output, marker)
	if start == -1 {
		t.Fatalf("output has no %q blockquote disclaimer:\n%s", marker, output)
	}
	start += len(marker)
	rest := output[start:]
	end := strings.Index(rest, "\n\n")
	if end == -1 {
		t.Fatalf("output's disclaimer blockquote has no terminating blank line:\n%s", output)
	}
	return rest[:end]
}

// locTuple is the D-05a stable-field projection (name/kind/filePath/
// startLine) callers/callees/impact-affected records are compared on,
// ignoring id (D-05a) and comparing as a SET rather than an ordered list
// (D-05b — tolerates edge-dedup multiplicity and scope differences).
type locTuple struct {
	Name      string
	Kind      string
	FilePath  string
	StartLine int32
}

func toLocSet(locs []query.Location) map[locTuple]bool {
	s := make(map[locTuple]bool, len(locs))
	for _, l := range locs {
		s[locTuple{Name: l.Name, Kind: l.Kind, FilePath: l.FilePath, StartLine: l.StartLine}] = true
	}
	return s
}

// assertSubset fails the test if got contains any entry absent from want
// (a genuine false positive — our engine reporting something the golden
// corpus's ground truth doesn't have), or if got is empty while want is
// not. It does NOT require got == want: per D-05b, this engine may
// legitimately report fewer distinct entries than the golden corpus
// (edge dedup, and the buildReverseAdjacency RefKindCalls-only scoping
// discovered building this harness — see the "callees" subtest).
func assertSubset(t *testing.T, label string, got, want map[locTuple]bool) {
	t.Helper()

	var extra []locTuple
	for k := range got {
		if !want[k] {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		t.Errorf("%s: got %d entr(y/ies) absent from the golden corpus (possible false positive, not a documented D-05b divergence): %v", label, len(extra), extra)
	}
	if len(got) == 0 && len(want) > 0 {
		t.Errorf("%s: got zero entries, want at least a non-empty subset of the golden corpus's %d", label, len(want))
	}
}

// sortedKeys returns m's keys sorted, for deterministic key-shape
// comparisons.
func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// nameFileLine is node.json's markdown "Name (file:line)" trail-entry
// shape — no kind is rendered in this format (D-05b, RenderNode), so
// comparisons against this fixture drop the kind field toLocSet's other
// callers carry.
type nameFileLine struct {
	Name     string
	FilePath string
	Line     string
}

var trailEntryPattern = regexp.MustCompile(`(\S+) \(([^:]+):(\d+)\)`)

// parseTrailLine extracts every "Name (file:line)" entry from the single
// line following marker (e.g. "**Calls →**" or "**Called by ←**") in a
// node.json-shaped markdown output, as a set.
func parseTrailLine(output, marker string) map[nameFileLine]bool {
	set := make(map[nameFileLine]bool)
	idx := strings.Index(output, marker)
	if idx == -1 {
		return set
	}
	rest := output[idx+len(marker):]
	if nl := strings.Index(rest, "\n"); nl != -1 {
		rest = rest[:nl]
	}
	for _, m := range trailEntryPattern.FindAllStringSubmatch(rest, -1) {
		set[nameFileLine{Name: m[1], FilePath: m[2], Line: m[3]}] = true
	}
	return set
}

func nameFileLineSetsEqual(a, b map[nameFileLine]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// assertNameFileLineSubset is nameFileLineSetsEqual's D-05b-style subset
// sibling (mirrors assertSubset above, for the parseTrailLine shape): it
// fails if got contains any entry absent from want (our narrower
// RefKindCalls-only extraction reporting something the expected set
// lacks), but tolerates got having FEWER entries than want.
func assertNameFileLineSubset(t *testing.T, label string, got, want map[nameFileLine]bool) {
	t.Helper()

	var extra []nameFileLine
	for k := range got {
		if !want[k] {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		t.Errorf("%s: got %d entr(y/ies) absent from the golden corpus's Calls trail (possible false positive, not a documented D-05b divergence): %v", label, len(extra), extra)
	}
}

// nodeLocationPattern matches a multi-def node section's "**Location:**
// file:line" line (RenderNodeMultiDef/renderNodeSection).
var nodeLocationPattern = regexp.MustCompile(`\*\*Location:\*\* (\S+):(\d+)`)

// multiDefBlock is one parsed "**<Symbol>** (kind)\n\n**Location:**
// ..." section out of a NODE-02 multi-def markdown body.
type multiDefBlock struct {
	Location string // "file:line"
	Calls    map[nameFileLine]bool
}

// parseNodeMultiDefBlocks splits a NODE-02 multi-def body (everything up
// to the optional "**Other definitions**" overflow list, per
// RenderNodeMultiDef's "\n\n---\n\n" section separator) into its
// per-definition blocks, extracting each one's Location and Calls trail.
// Blocks are returned in RENDER order — callers that need an
// order-independent comparison (Assumption A3: render order carries no
// meaningful tie-break semantic) should key off Location as a set, not
// slice position.
func parseNodeMultiDefBlocks(output string) []multiDefBlock {
	body := output
	if idx := strings.Index(body, "\n\n**Other definitions**"); idx != -1 {
		body = body[:idx]
	}
	var out []multiDefBlock
	for _, blk := range strings.Split(body, "\n\n---\n\n") {
		m := nodeLocationPattern.FindStringSubmatch(blk)
		if m == nil {
			continue
		}
		out = append(out, multiDefBlock{
			Location: m[1] + ":" + m[2],
			Calls:    parseTrailLine(blk, "**Calls →**"),
		})
	}
	return out
}

// locationSet projects a []multiDefBlock onto its Location set, for
// order-independent def-set comparisons (Assumption A3).
func locationSet(blocks []multiDefBlock) map[string]bool {
	s := make(map[string]bool, len(blocks))
	for _, b := range blocks {
		s[b.Location] = true
	}
	return s
}

// blockByLocation indexes blocks by Location for per-def Calls-trail
// lookups after an order-independent Location match.
func blockByLocation(blocks []multiDefBlock) map[string]multiDefBlock {
	m := make(map[string]multiDefBlock, len(blocks))
	for _, b := range blocks {
		m[b.Location] = b
	}
	return m
}

// nodeMultiDefHeaderPattern matches NODE-01/02's exact two-line header
// template (RenderNodeMultiDef): "**N definitions named "X"**\nReturning
// M in full[; K more listed below] — pick the one you need (no Read
// required).\n" — the wording is a fixed contract (rewording it silently
// breaks NODE-01/02); N/X/M/K are the VARIABLE content D-02 does not
// require equality on (definition counts depend on extraction coverage,
// not on the template being correct).
var nodeMultiDefHeaderPattern = regexp.MustCompile(
	`^\*\*(\d+) definitions named "([^"]*)"\*\*\nReturning (\d+) in full(; \d+ more listed below)? — pick the one you need \(no Read required\)\.\n`)

// exploreFileHeaderPattern matches an explore markdown's per-file source
// header ("**`path`** — sym(kind), ..."), for extracting the SET of files
// Explore actually selected (RenderExplore).
var exploreFileHeaderPattern = regexp.MustCompile("\\*\\*`([^`]+)`\\*\\* — ")

// exploreSelectedFiles returns the set of file paths an explore markdown
// output rendered a source block for.
func exploreSelectedFiles(output string) map[string]bool {
	set := make(map[string]bool)
	for _, m := range exploreFileHeaderPattern.FindAllStringSubmatch(output, -1) {
		set[m[1]] = true
	}
	return set
}

// TestCorpusBehavior_Go is the "go" matrix gate's live-Go property test.
// It runs the real indexer + query.Engine over the committed, always-in-repo
// behavioral corpus (D-03) and asserts the Go-truthful properties of its
// LIVE output in the D-09 style — named behavioral properties of live
// engine output, not byte-diffs against a frozen golden. The TS-era
// capture path and external corpora that TestCorpusBehavior_Go formerly
// diffed against are gone as of this phase (FIXT-04); the status.json
// key-loop inheritance from probe-01 is retired with them.
func TestCorpusBehavior_Go(t *testing.T) {
	engine := buildEngineAt(t, behavioralCorpusSrc(t))

	t.Run("status", func(t *testing.T) {
		got, err := engine.Status(context.Background())
		if err != nil {
			t.Fatalf("Status: %v", err)
		}

		// D-05c: status.go's own doc comment is the authoritative
		// key-mapping table; assert the Go-truthful values it
		// documents (values with no Pebble analog do not exist in
		// this output).
		if got.Backend != "pebble" {
			t.Errorf("status.Backend = %q, want %q (D-05c backend remap)", got.Backend, "pebble")
		}
		if !got.Initialized {
			t.Error("status.Initialized = false, want true")
		}
		if got.Index.PendingRefs != 0 {
			t.Errorf("status.Index.PendingRefs = %d, want 0 (Phase 2 resolves all refs at index time)", got.Index.PendingRefs)
		}
		if got.WorktreeMismatch != nil {
			t.Errorf("status.WorktreeMismatch = %v, want nil (Phase-4 sync placeholder)", *got.WorktreeMismatch)
		}
		if got.PendingChanges != (query.PendingChanges{}) {
			t.Errorf("status.PendingChanges = %+v, want the zero value (Phase-4 sync placeholder)", got.PendingChanges)
		}
		// The behavioral corpus is Go-only (D-03), so the registered
		// language set is exactly {"go"}.
		wantLanguages := []string{"go"}
		gotJoined, wantJoined := strings.Join(got.Languages, ","), strings.Join(wantLanguages, ",")
		if gotJoined != wantJoined {
			t.Errorf("status.Languages = %v, want %v (Go-only corpus)", got.Languages, wantLanguages)
		}

		raw, err := query.MarshalStatusJSON(got)
		if err != nil {
			t.Fatalf("MarshalStatusJSON: %v", err)
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("unmarshal our own status JSON: %v", err)
		}
		// D-05c: journalMode has no Pebble analog and must be dropped
		// entirely, not merely unused.
		if _, ok := decoded["journalMode"]; ok {
			t.Error(`status --json unexpectedly includes "journalMode" — D-05c documents this key as dropped (no Pebble analog)`)
		}
		if _, ok := decoded["score"]; ok {
			t.Error(`status --json unexpectedly includes "score" — D-05d: no FTS5/BM25 ranking exists`)
		}

		// edgesByKind and filesByLanguage are emitted as of v0.11.0
		// Phase 1 (FIXT-01, D-02/D-03) — asserted here as positive
		// presence checks against OUR OWN status --json output (D-09
		// property style: the key must be present, an object, and
		// non-empty, not merely equal to a frozen byte string).
		rawEdgesByKind, ok := decoded["edgesByKind"]
		if !ok {
			t.Fatal(`our status --json output is missing "edgesByKind" (FIXT-01)`)
		}
		edgesByKind, ok := rawEdgesByKind.(map[string]interface{})
		if !ok {
			t.Fatalf("edgesByKind = %v (%T), want a JSON object", rawEdgesByKind, rawEdgesByKind)
		}
		if len(edgesByKind) == 0 {
			t.Error("edgesByKind is present but empty — sparse mode must still report the kinds it measured")
		}
		for kind, rawCount := range edgesByKind {
			count, ok := rawCount.(float64)
			if !ok || count != float64(int64(count)) || count <= 0 {
				t.Errorf("edgesByKind[%q] = %v, want a positive integer", kind, rawCount)
			}
		}

		rawFilesByLanguage, ok := decoded["filesByLanguage"]
		if !ok {
			t.Fatal(`our status --json output is missing "filesByLanguage" (FIXT-01)`)
		}
		filesByLanguage, ok := rawFilesByLanguage.(map[string]interface{})
		if !ok {
			t.Fatalf("filesByLanguage = %v (%T), want a JSON object", rawFilesByLanguage, rawFilesByLanguage)
		}
		gotFilesByLanguageKeys := make([]string, 0, len(filesByLanguage))
		for k := range filesByLanguage {
			gotFilesByLanguageKeys = append(gotFilesByLanguageKeys, k)
		}
		sort.Strings(gotFilesByLanguageKeys)
		wantFilesByLanguageKeys := append([]string(nil), got.Languages...)
		sort.Strings(wantFilesByLanguageKeys)
		if strings.Join(gotFilesByLanguageKeys, ",") != strings.Join(wantFilesByLanguageKeys, ",") {
			t.Errorf("filesByLanguage key set = %v, want it to equal status.Languages = %v — both are derived from the same scan and disagreement means one derivation broke", gotFilesByLanguageKeys, wantFilesByLanguageKeys)
		}

		// D-08: dbSizeBytes is exempted from the volatility check HERE,
		// at this call site only — the shared volatileKeys map in
		// golden_test.go is deliberately left untouched. Our OWN status
		// --json output intentionally carries the key (Pebble has no
		// dbPath-file analog, so a directory byte sum is the
		// Go-truthful reading — see status.go's decision table). The
		// plausibility assertions immediately below replace the blanket
		// "must be absent" check for this one key.
		if volatile := findVolatileKeysExcept(decoded, "our status.json", "dbSizeBytes"); len(volatile) > 0 {
			t.Errorf("our status --json output contains volatile field(s) that must never be emitted: %v", volatile)
		}

		// D-08 plausibility assertions: presence, integer type, > 0, and a
		// well-formed MB rendering — never cross-run byte stability
		// (Pebble's LSM compaction makes the on-disk total genuinely
		// nondeterministic across identical reindexes).
		rawSize, present := decoded["dbSizeBytes"]
		if !present {
			t.Fatal(`our status --json output is missing "dbSizeBytes" (STAT-01)`)
		}
		size, ok := rawSize.(float64)
		if !ok {
			t.Fatalf("dbSizeBytes = %v (%T), want a JSON number", rawSize, rawSize)
		}
		if size != float64(int64(size)) || size <= 0 {
			t.Fatalf("dbSizeBytes = %v, want a positive integer", rawSize)
		}
		mbRendering := fmt.Sprintf("%.2f MB", size/1024/1024)
		if !mbShapeRE.MatchString(mbRendering) {
			t.Errorf("dbSizeBytes MB rendering %q does not match %s (D-07)", mbRendering, mbShapeRE.String())
		}
	})
}

// ============================================================================
// Behavioral harness — property assertions over live Go engine output
// ============================================================================
//
// TestCorpusBehavior* exercises the query.Engine against the committed,
// always-in-repo behavioral corpus (D-03), asserting named behavioral
// properties of live output in the D-09 style rather than byte-diffing a
// frozen golden.
//
// Allowed-divergence notes (retained from the D-02 harness):
//
//   - AD-04 (file-selection breadth + blast-radius bullet scope): even on
//     the behavioral corpus (purpose-built and validated for this),
//     Go's RWR-selected file set and blast-radius bullet set are known
//     to differ from the historical expected sets in both directions:
//     this tokenizer does not apply the broader partial "account" token
//     match that would pull in ledger/ledger.go, and Go renders a
//     blast-radius bullet for every selected candidate (including
//     zero-caller structs/files). Asserted as CORE-symbol membership
//     (the specific symbols the D-03 corpus was purpose-built to test),
//     not full bullet-set or file-count equality.
//   - A3 (already documented, 01-RESEARCH.md Architecture Patterns
//     Pattern 2): multi-def render order carries no meaningful semantic
//     to replicate; node-multi def-set comparisons below are
//     Location-SET based (order-independent), never slice-position
//     based.

// TestCorpusBehaviorSynthetic asserts the four named behavioral properties
// (D-09) of live Go engine output, driven by corpus/behavioral/CASES.json.
// Each case in the case map carries a query + assertion mode; the test
// derives symbols/files/query/assertion from data, not from a Go literal
// table (D-04). The go-* goldens remain committed as regression snapshots
// but are NOT the primary oracle — a failing test names which behavior
// broke, not merely "a golden diff appeared".
func TestCorpusBehaviorSynthetic(t *testing.T) {
	cases := loadBehavioralCases(t)
	eng := buildEngineAt(t, behavioralCorpusSrc(t))

	// executedCases is the EXECUTION-keyed counter (review finding #1): it
	// is incremented inside the per-case closure, so it tracks how many
	// cases actually RAN to completion, not how many the inventory says
	// exist. After the loop it must exactly equal len(cases) — a loop that
	// is annihilated (early return, skipped iteration) leaves it short and
	// fails the positive assertion, so a zero-executed run is red by
	// construction (rule 84d1gfpywd).
	executedCases := 0
	for _, tc := range cases {
		t.Run(tc.ID+"-"+tc.Name, func(t *testing.T) {
			switch tc.Assertion {
			case "overloaded-defs-distinct":
				// Case (a): Node("Validate") returns exactly 2 distinct
				// definitions whose locations are accounts/validate.go
				// and orders/validate.go (overloaded dedup stays distinct).
				got, err := eng.Node(tc.Symbol, "", nil)
				if err != nil {
					t.Fatalf("Node(%q, \"\"): %v", tc.Symbol, err)
				}
				blocks := parseNodeMultiDefBlocks(got)
				locs := locationSet(blocks)
				if len(locs) != 2 {
					t.Errorf("Node(%q): got %d defs, want 2 (locations: %v)", tc.Symbol, len(locs), locs)
				}
				for _, wantFile := range tc.Files {
					found := false
					for loc := range locs {
						if strings.HasPrefix(loc, wantFile+":") {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Node(%q): missing expected def in %s (got locations: %v)", tc.Symbol, wantFile, locs)
					}
				}
				// Verify the header template matches the exact def count.
				wantHeader := fmt.Sprintf("**2 definitions named %q**\nReturning 2 in full — pick the one you need (no Read required).\n\n", tc.Symbol)
				if !strings.HasPrefix(got, wantHeader) {
					t.Errorf("Node(%q) header mismatch:\ngot prefix: %q\nwant: %q", tc.Symbol, firstNChars(got, len(wantHeader)+20), wantHeader)
				}
				headerCount := nodeMultiDefCount(t, got)
				if headerCount != 2 {
					t.Errorf("Node(%q) header def count = %d, want 2", tc.Symbol, headerCount)
				}

			case "multi-word-tokenization":
				// Case (b): Explore("user account") surfaces
				// UserAccountManager and selects accounts/manager.go
				// (camelCase multi-word tokenization).
				got, err := eng.Explore(tc.Query, 0)
				if err != nil {
					t.Fatalf("Explore(%q, 0): %v", tc.Query, err)
				}
				wantHeader := fmt.Sprintf("**Exploration: %s**\n\n", tc.Query)
				if !strings.HasPrefix(got, wantHeader) {
					t.Errorf("Explore(%q) header mismatch:\ngot prefix: %q\nwant: %q", tc.Query, firstNChars(got, len(wantHeader)+20), wantHeader)
				}
				gotFiles := exploreSelectedFiles(got)
				if !gotFiles[tc.Files[0]] {
					t.Errorf("Explore(%q) selected files = %v, want %q among them", tc.Query, gotFiles, tc.Files[0])
				}
				if !strings.Contains(got, tc.Symbol) {
					t.Errorf("Explore(%q) output does not mention %q", tc.Query, tc.Symbol)
				}

			case "cluster-surfaces-connected-non-test":
				// Case (c): Explore("user account") surfaces the
				// structurally-connected non-test symbols (recoverAccount)
				// over the zero-inbound TestAccountRecovery. The output
				// carries a "tests: recovery/recovery_test.go" clause
				// for the connected symbol.
				got, err := eng.Explore(tc.Query, 0)
				if err != nil {
					t.Fatalf("Explore(%q, 0): %v", tc.Query, err)
				}
				for _, connected := range tc.Connected {
					if !strings.Contains(got, connected) {
						t.Errorf("Explore(%q): expected %q to be surfaced (structurally-connected non-test symbol), got:\n%s", tc.Query, connected, got)
					}
				}
				// The weakly-connected Test* symbol must not be promoted.
				// However, its file may appear in the "tests:" clause of
				// a connected symbol. Check the test file is not SELECTED
				// as its own rendered section (via the **`file`** pattern).
				if strings.Contains(got, "**`recovery/recovery_test.go`**") {
					t.Errorf("Explore(%q): the weakly-connected Test*-only file must not be selected/rendered (H15 hard test exclusion)", tc.Query)
				}
				if !strings.Contains(got, "tests:") || !strings.Contains(got, "recovery_test.go") {
					t.Errorf("Explore(%q) output missing recovery_test.go tests: clause:\n%s", tc.Query, got)
				}

			case "structural-surfaces-zero-lexical-match":
				// Case (d): Explore("account balance") must (a) select
				// ledger/ledger.go, (b) surface GetBalance (the partial-
				// lexical structural bridge), and (c) surface
				// ReconcileLedger (zero lexical match) via structural
				// expansion. The property is SURFACING (present and
				// reachable), not strict ranking above the isolated
				// AccountBalanceHelper — matching the authoritative
				// TestExploreStructuralBeatsLexical contract
				// (internal/query/explore_test.go:144-183).
				got, err := eng.Explore(tc.Query, 5)
				if err != nil {
					t.Fatalf("Explore(%q, 5): %v", tc.Query, err)
				}
				for _, wantFile := range tc.Files {
					if !strings.Contains(got, wantFile) {
						t.Errorf("Explore(%q): expected file %q to be selected, got:\n%s", tc.Query, wantFile, got)
					}
				}
				if !strings.Contains(got, "GetBalance") {
					t.Errorf("Explore(%q): expected GetBalance (the structural bridge into ReconcileLedger) to be surfaced despite its lower raw gather score", tc.Query)
				}
				if !strings.Contains(got, "ReconcileLedger") {
					t.Errorf("Explore(%q): expected ReconcileLedger (zero lexical match) to be surfaced via structural expansion", tc.Query)
				}
				// DO NOT assert ranking above AccountBalanceHelper under
				// this RWR formulation; assert surfacing only.

			default:
				t.Fatalf("unknown assertion mode %q in CASES.json case %s", tc.Assertion, tc.ID)
			}

			executedCases++
		})
	}

	if executedCases != len(cases) {
		t.Fatalf("executed %d of %d behavioral cases — the executed count must EXACTLY equal the case total (a loop that never runs, or returns early before the per-case closure completes, cannot satisfy the positive claim; rule 84d1gfpywd)", executedCases, len(cases))
	}
}

// ============================================================================
// Locked-corpus hermetic resolution (02-03, D-10)
// ============================================================================

// TestPriorityLanguagesResolveToLockedCorpus (H3 positive guard): iterates
// the five priority-4 languages (go, java, csharp, python, tsjs), calls
// lockedCorpusDir(t, lang) for each, and asserts every one resolves to a
// directory that exists. A language whose map entry or locked corpus is
// missing FAILS (rule 84d1gfpywd) — it never skips and never silently covers
// less than all five languages.
func TestPriorityLanguagesResolveToLockedCorpus(t *testing.T) {
	priorityLangs := []string{"go", "java", "csharp", "python", "tsjs"}
	for _, lang := range priorityLangs {
		t.Run(lang, func(t *testing.T) {
			dir := lockedCorpusDir(t, lang)
			info, err := os.Stat(dir)
			if err != nil {
				t.Fatalf("lockedCorpusDir(%q) resolved to %s but stat failed: %v", lang, dir, err)
			}
			if !info.IsDir() {
				t.Fatalf("lockedCorpusDir(%q) resolved to %s which is not a directory", lang, dir)
			}
			t.Logf("%s -> %s (files present)", lang, dir)
		})
	}
}

// TestCorpusBehaviorLockedCorpora runs the shape-only tier over each locked
// corpus: index, build engine, assert non-empty explore output (header,
// blast-radius, source sections) and node output (header template, at least
// one full def body). This is the D-09 property-shape tier over the locked
// corpora; the byte-level goldens are the re-freeze's regression snapshots
// (02-04).
func TestCorpusBehaviorLockedCorpora(t *testing.T) {
	lockedSpecs := []struct {
		lang, slug string
		query      string
		symbol     string
	}{
		{"go", "hugo", "page content", "Page"},
		{"java", "guava", "check precondition", "Preconditions"},
		{"csharp", "serilog", "configure logger", "LoggerConfiguration"},
		{"python", "requests", "http session", "Session"},
	}

	for _, s := range lockedSpecs {
		t.Run(s.slug, func(t *testing.T) {
			src := lockedCorpusDir(t, s.lang)
			eng := buildEngineAt(t, src)

			t.Run("explore-shape", func(t *testing.T) {
				got, err := eng.Explore(s.query, 0)
				if err != nil {
					t.Fatalf("Explore(%q, 0): %v", s.query, err)
				}
				assertExploreShape(t, got, s.query)
				t.Logf("explore(%q) produced valid shape over %s (%s)", s.query, s.slug, s.lang)
			})
			t.Run("node-shape", func(t *testing.T) {
				got, err := eng.Node(s.symbol, "", nil)
				if err != nil {
					t.Fatalf("Node(%q, \"\"): %v", s.symbol, err)
				}
				// The symbol may be single-def or multi-def in a given corpus.
				// Validate at least: non-empty output with location info.
				if len(got) == 0 {
					t.Fatalf("Node(%q) returned empty output", s.symbol)
				}
				if !strings.Contains(got, "**Location:**") && !strings.Contains(got, s.symbol) {
					t.Errorf("Node(%q) output missing location and symbol name", s.symbol)
				}
				t.Logf("node(%q) produced valid shape over %s (%s)", s.symbol, s.slug, s.lang)
			})
		})
	}
}

// assertExploreShape is the shape-only tier's explore assertion: valid
// header, non-empty blast-radius section, non-empty source section.
func assertExploreShape(t *testing.T, got, query string) {
	t.Helper()

	wantHeader := fmt.Sprintf("**Exploration: %s**\n\n", query)
	if !strings.HasPrefix(got, wantHeader) {
		t.Errorf("Explore(%s) header mismatch:\ngot prefix: %q\nwant: %q", query, firstNChars(got, len(wantHeader)+20), wantHeader)
	}
	if !strings.Contains(got, "**Blast radius") {
		t.Errorf("Explore(%s) output missing the blast-radius section", query)
	}
	if !strings.Contains(got, "**Source Code**") {
		t.Errorf("Explore(%s) output missing the Source Code section", query)
	}
	if len(exploreSelectedFiles(got)) == 0 {
		t.Errorf("Explore(%s) selected zero files", query)
	}
}

// assertNodeMultiDefShape is the shape-only tier's node-multi assertion:
// the exact NODE-01/02 header TEMPLATE (byte-identical wording, variable
// counts) and at least one full def body.
func assertNodeMultiDefShape(t *testing.T, got, symbol string) {
	t.Helper()

	m := nodeMultiDefHeaderPattern.FindStringSubmatch(got)
	if m == nil {
		t.Fatalf("Node(%s) output does not match the NODE-01/02 header template:\n%s", symbol, firstNChars(got, 300))
	}
	if m[2] != symbol {
		t.Errorf("Node(%s) header symbol = %q, want %q", symbol, m[2], symbol)
	}
	blocks := parseNodeMultiDefBlocks(got)
	if len(blocks) == 0 {
		t.Errorf("Node(%s) rendered zero full def bodies", symbol)
	}
}

// nodeMultiDefCount extracts the "N definitions named" count from a
// NODE-01/02 header for a t.Logf comparison (not an assertion — AD-03).
func nodeMultiDefCount(t *testing.T, output string) int {
	t.Helper()

	m := nodeMultiDefHeaderPattern.FindStringSubmatch(output)
	if m == nil {
		t.Fatalf("output does not match the NODE-01/02 header template:\n%s", firstNChars(output, 300))
	}
	n := 0
	for _, c := range m[1] {
		n = n*10 + int(c-'0')
	}
	return n
}

// firstNChars safely truncates s to at most n runes, for readable
// t.Errorf/t.Fatalf output on long markdown bodies.
func firstNChars(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// ============================================================================
// CLI==MCP byte-identity (plan 17, EXPL-05/NODE-04)
// ============================================================================
//
// TestExploreCLIMatchesMCP / TestNodeCLIMatchesMCP lock in EXPL-05/NODE-04's
// shared-Engine guarantee end-to-end: both surfaces (internal/cli's direct
// query.OpenAt+Engine call, and internal/mcp's real stdio-shaped server via
// an in-process client) are driven against the SAME on-disk index and must
// produce byte-identical output for every behavioral fixture query/symbol.
// This is structural (both code paths call the same Engine.Explore/
// Engine.Node) — the test exists to CATCH a future divergence, not to prove
// something surprising.

// newGoldenSession builds an in-memory client/server session pair for s,
// mirroring internal/mcp/server_test.go's newTestSession (unexported
// there; reimplemented here since this file lives in the external golden
// package). go-sdk's Client.Connect performs the MCP initialize handshake
// itself — there is no separate, repeatable Initialize call the way
// mark3labs' client had (02-RESEARCH.md Q1's PROVEN-ABSENT finding: no
// ServerOptions field or client method lets a caller inject a
// ProtocolVersion, so internalmcp.ProtocolVersion keeps its role as the
// asserted pin rather than a value this harness sends on the wire).
func newGoldenSession(t *testing.T, s *mcp.Server) *mcp.ClientSession {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go func() {
		_ = s.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "codegraph-behavioral-test", Version: "0.0.0"}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })

	return session
}

// callExploreViaMCP drives codegraph_explore through a real, in-process
// MCP server (internalmcp.BuildServer + newGoldenSession), mirroring
// internal/mcp/server_test.go's TestExploreHandlerDelegatesToEngine
// pattern (reimplemented here since that file's helpers are unexported and
// this file lives in the external golden package).
func callExploreViaMCP(t *testing.T, repoDir, query string) string {
	t.Helper()

	s := internalmcp.BuildServer(true, map[string]bool{}, repoDir, repoDir)
	session := newGoldenSession(t, s)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "codegraph_explore",
		Arguments: map[string]any{"query": query},
	})
	if err != nil {
		t.Fatalf("CallTool codegraph_explore(%q): %v", query, err)
	}
	if result.IsError {
		t.Fatalf("codegraph_explore(%q) returned an error result: %+v", query, result)
	}
	return mcpResultText(t, result)
}

// callNodeViaMCP drives codegraph_node the same way callExploreViaMCP
// drives codegraph_explore.
func callNodeViaMCP(t *testing.T, repoDir, symbol string) string {
	t.Helper()
	return callNodeViaMCPWithArgs(t, repoDir, symbol, "", nil)
}

// callNodeViaMCPWithArgs is callNodeViaMCP's fuller sibling (CR-02): it
// additionally accepts the "file" and "line" args codegraph_node's schema
// exposes, so TestNodeLineHintCLIMatchesMCP can drive the SAME
// codegraph_node MCP call the CLI's --line flag now reaches, proving
// EXPL-05/NODE-04 byte-identity extends to the new NODE-03 narrowing
// parameter, not just the pre-CR-02 (symbol, file) surface.
func callNodeViaMCPWithArgs(t *testing.T, repoDir, symbol, file string, line *int) string {
	t.Helper()

	s := internalmcp.BuildServer(true, map[string]bool{"node": true}, repoDir, repoDir)
	session := newGoldenSession(t, s)

	args := map[string]any{"symbol": symbol}
	if file != "" {
		args["file"] = file
	}
	if line != nil {
		args["line"] = float64(*line)
	}

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "codegraph_node",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool codegraph_node(%q): %v", symbol, err)
	}
	if result.IsError {
		t.Fatalf("codegraph_node(%q) returned an error result: %+v", symbol, result)
	}
	return mcpResultText(t, result)
}

// mcpResultText extracts the first text content block from a successful
// CallTool result.
func mcpResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	if len(result.Content) == 0 {
		t.Fatal("CallTool result has no content")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("CallTool result content[0] is not text: %+v", result.Content[0])
	}
	return text.Text
}

// TestExploreCLIMatchesMCP drives explore's CLI code path
// (query.OpenAt+Engine.Explore, exactly what internal/cli/explore.go
// does) and its MCP code path (a real in-process server) against the SAME
// on-disk index for every behavioral fixture query, and asserts
// byte-identical output (EXPL-05).
func TestExploreCLIMatchesMCP(t *testing.T) {
	cases := []struct {
		corpus     string
		sourceFunc func(t *testing.T) string
		query      string
	}{
		{"behavioral", behavioralCorpusSrc, "user account"},
		{"hugo", func(t *testing.T) string { return lockedCorpusDir(t, "go") }, "page content"},
		{"guava", func(t *testing.T) string { return lockedCorpusDir(t, "java") }, "check precondition"},
		{"serilog", func(t *testing.T) string { return lockedCorpusDir(t, "csharp") }, "configure logger"},
		{"requests", func(t *testing.T) string { return lockedCorpusDir(t, "python") }, "http session"},
	}

	for _, tc := range cases {
		t.Run(tc.corpus, func(t *testing.T) {
			src := tc.sourceFunc(t)
			dir := buildIndexedFixture(t, src)

			eng, closer, err := query.OpenAt(dir)
			if err != nil {
				t.Fatalf("query.OpenAt(%s): %v", dir, err)
			}
			cliOut, err := eng.Explore(tc.query, 0)
			closer.Close()
			if err != nil {
				t.Fatalf("Engine.Explore(%q): %v", tc.query, err)
			}

			mcpOut := callExploreViaMCP(t, dir, tc.query)

			if cliOut != mcpOut {
				t.Errorf("explore(%q) on %s: CLI and MCP output diverge (EXPL-05):\nCLI:\n%s\nMCP:\n%s", tc.query, tc.corpus, cliOut, mcpOut)
			}
		})
	}
}

// TestNodeCLIMatchesMCP is TestExploreCLIMatchesMCP's node sibling
// (NODE-04): both single-def and multi-def symbols are covered, since
// NODE-04's byte-comparability claim spans both shapes.
func TestNodeCLIMatchesMCP(t *testing.T) {
	lockedExploreCase := func(lang string) func(t *testing.T) string {
		return func(t *testing.T) string { return lockedCorpusDir(t, lang) }
	}
	cases := []struct {
		corpus     string
		sourceFunc func(t *testing.T) string
		symbol     string
	}{
		{"behavioral", behavioralCorpusSrc, "Validate"},   // multi-def (2)
		{"behavioral", behavioralCorpusSrc, "AuditEntry"}, // single-def
		{"hugo", lockedExploreCase("go"), "Site"},
		{"guava", lockedExploreCase("java"), "ImmutableList"},
		{"serilog", lockedExploreCase("csharp"), "LoggerConfiguration"},
		{"requests", lockedExploreCase("python"), "Session"},
	}

	for _, tc := range cases {
		t.Run(tc.corpus+"/"+tc.symbol, func(t *testing.T) {
			src := tc.sourceFunc(t)
			dir := buildIndexedFixture(t, src)

			eng, closer, err := query.OpenAt(dir)
			if err != nil {
				t.Fatalf("query.OpenAt(%s): %v", dir, err)
			}
			cliOut, err := eng.Node(tc.symbol, "", nil)
			closer.Close()
			if err != nil {
				t.Fatalf("Engine.Node(%q, \"\"): %v", tc.symbol, err)
			}

			mcpOut := callNodeViaMCP(t, dir, tc.symbol)

			if cliOut != mcpOut {
				t.Errorf("node(%q) on %s: CLI and MCP output diverge (NODE-04):\nCLI:\n%s\nMCP:\n%s", tc.symbol, tc.corpus, cliOut, mcpOut)
			}
		})
	}
}

// TestNodeLineHintCLIMatchesMCP is CR-02's CLI==MCP regression pin for
// NODE-03's new `line` narrowing parameter: it builds a tiny real,
// indexed fixture with two same-named "Dup" definitions at deliberately
// distinct line numbers, drives the CLI-facing call (Engine.Node with a
// line hint, exactly what internal/cli/node.go's new --line flag now
// calls) and the MCP-facing call (codegraph_node's new "line" arg, via
// callNodeViaMCPWithArgs) with the SAME hint, and asserts both: (a) the
// hint actually narrowed the 2-def match down to a single-def render
// (proving NODE-03 is reachable, not just byte-identical-but-still-dead),
// and (b) CLI and MCP output are byte-identical (EXPL-05/NODE-04 extended
// to the new parameter).
func TestNodeLineHintCLIMatchesMCP(t *testing.T) {
	src := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		full := filepath.Join(src, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	write("go.mod", "module example.com/nodeline\n\ngo 1.24\n")
	write("a/dup.go", "package a\n\nfunc Dup() int {\n\treturn 1\n}\n")
	// b/dup.go pads the function's start line well past a/dup.go's (3),
	// so a line hint can unambiguously select one over the other.
	write("b/dup.go", "package b\n\n// padding\n// padding\n// padding\n// padding\n// padding\n// padding\n// padding\nfunc Dup() int {\n\treturn 2\n}\n")

	dir := buildIndexedFixture(t, src)

	line := 10 // b/dup.go's "func Dup() int {" line

	eng, closer, err := query.OpenAt(dir)
	if err != nil {
		t.Fatalf("query.OpenAt(%s): %v", dir, err)
	}
	cliOut, err := eng.Node("Dup", "", &line)
	closer.Close()
	if err != nil {
		t.Fatalf("Engine.Node(Dup, line=%d): %v", line, err)
	}
	if strings.Contains(cliOut, "definitions named") {
		t.Fatalf("Engine.Node(Dup, line=%d): expected the line hint to narrow to a single def, got:\n%s", line, cliOut)
	}
	if !strings.Contains(cliOut, "**Location:** b/dup.go:10\n") {
		t.Fatalf("Engine.Node(Dup, line=%d): expected b/dup.go's def to be selected, got:\n%s", line, cliOut)
	}

	mcpOut := callNodeViaMCPWithArgs(t, dir, "Dup", "", &line)

	if cliOut != mcpOut {
		t.Errorf("node(Dup, line=%d): CLI and MCP output diverge:\nCLI:\n%s\nMCP:\n%s", line, cliOut, mcpOut)
	}
}
