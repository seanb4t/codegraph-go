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

	"github.com/seanb4t/codegraph-go/internal/indexer"
	internalmcp "github.com/seanb4t/codegraph-go/internal/mcp"
	"github.com/seanb4t/codegraph-go/internal/query"
)

// mbShapeRE pins D-07's MB-rendering contract at the parity layer, where
// dbSizeBytes comes from a real corpus index rather than a synthetic
// fixture value: fmt.Sprintf("%.2f MB", bytes/1024/1024).
var mbShapeRE = regexp.MustCompile(`^\d+\.\d{2} MB$`)

// findVolatileKeysExcept wraps golden_test.go's findVolatileKeys (shared
// with TestGoldenFixturesExist, which correctly continues to enforce that
// the FROZEN TS oracle fixtures in corpus/*/*.json never re-include a
// volatile key such as dbSizeBytes) with a named exemption list, used
// ONLY at this file's own-output call site. It never mutates the shared
// volatileKeys map in golden_test.go — that map keeps governing the
// frozen corpus fixtures, and must keep failing if a TS golden ever
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

// syntheticParitySrc resolves the committed, in-repo synthetic-parity
// corpus source tree (D-03) — always available, no network/external
// checkout required.
func syntheticParitySrc(t *testing.T) string {
	t.Helper()

	src, err := filepath.Abs(filepath.Join("corpus", "synthetic-parity", "src"))
	if err != nil {
		t.Fatalf("resolve synthetic-parity source path: %v", err)
	}
	if info, err := os.Stat(src); err != nil || !info.IsDir() {
		t.Fatalf("synthetic-parity source not found at %s (err=%v) — see corpus/synthetic-parity/README.md", src, err)
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
// fixture of that kind (e.g. synthetic-parity has no baseline
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
// (a genuine false positive — our engine reporting something TS's ground
// truth doesn't have), or if got is empty while want is not. It does NOT
// require got == want: per D-05b, our engine may legitimately report
// fewer distinct entries than TS (edge dedup, and the buildReverseAdjacency
// RefKindCalls-only scoping discovered building this harness — see the
// "callees" subtest).
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
// RefKindCalls-only extraction reporting something TS's ground truth
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
// order-independent comparison (Assumption A3: TS's un-ordered SQLite
// SELECT has no meaningful tie-break semantic to replicate) should key
// off Location as a set, not slice position.
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
// required).\n" — byte-identical TS wording, N/X/M/K are captured as the
// VARIABLE content D-02 does not require Go/TS equality on (definition
// counts depend on extraction coverage, not on the template being
// correct).
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
// engine output, not byte-diffs against a frozen TS golden. The TS-era
// capture path and external corpora that TestCorpusBehavior_Go formerly
// diffed against are gone as of this phase (FIXT-04); the status.json
// key-loop inheritance from probe-01 is retired with them.
func TestCorpusBehavior_Go(t *testing.T) {
	engine := buildEngineAt(t, syntheticParitySrc(t))

	t.Run("status", func(t *testing.T) {
		got, err := engine.Status(context.Background())
		if err != nil {
			t.Fatalf("Status: %v", err)
		}

		// D-05c: status.go's own doc comment is the authoritative
		// TS-key-to-Go/Pebble-analog mapping table; assert the
		// Go-truthful values it documents (not byte-identical TS
		// values, which don't exist for a Pebble backend).
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
// frozen TS golden.
//
// Allowed-divergence notes (retained from the D-02 harness):
//
//   - AD-04 (file-selection breadth + blast-radius bullet scope): even on
//     synthetic-parity (the corpus purpose-built and validated for this),
//     Go's RWR-selected file set and blast-radius bullet set diverge from
//     TS's historical output in both directions: TS pulls in ledger/ledger.go
//     via a broader partial "account" token match that Go's tokenizer does
//     not apply (Go's 3-file selection is a subset of TS's 4), and Go renders
//     a blast-radius bullet for every selected candidate (including
//     zero-caller structs/files) where TS appears to render bullets more
//     selectively. Asserted as CORE-symbol membership (the specific symbols
//     the D-03 corpus was purpose-built to test), not full bullet-set or
//     file-count equality.
//   - A3 (already documented, 01-RESEARCH.md Architecture Patterns
//     Pattern 2): TS's un-ordered SQLite SELECT gives its own multi-def
//     ordering no meaningful semantic to replicate; node-multi def-set
//     comparisons below are Location-SET based (order-independent), never
//     slice-position based.

// TestCorpusBehaviorSynthetic runs the FULL D-02 oracle against
// the synthetic-parity corpus (D-03) — the one corpus co-designed to be
// tractable for byte/structural-level assertions, always available
// in-repo (no network/external checkout).
func TestCorpusBehaviorSynthetic(t *testing.T) {
	const corpus = "synthetic-parity"
	eng := buildEngineAt(t, syntheticParitySrc(t))

	t.Run("node-multi", func(t *testing.T) {
		// Symbol/query per README.md's per-corpus table: "Validate" has
		// exactly 2 real defs (accounts/validate.go + orders/validate.go,
		// D-03 case a).
		got, err := eng.Node("Validate", "", nil)
		if err != nil {
			t.Fatalf("Node(Validate, \"\"): %v", err)
		}
		want := loadGoldenOutputIn(t, corpus, "node-multi.json")

		// Header wording is deterministic (pure enumeration, no ranking
		// involved) — assert the FULL template, including the exact
		// def/return counts (2/2, no overflow clause), since this
		// corpus's def count is small and unambiguous.
		wantHeader := "**2 definitions named \"Validate\"**\nReturning 2 in full — pick the one you need (no Read required).\n\n"
		if !strings.HasPrefix(got, wantHeader) {
			t.Errorf("Node(Validate) header mismatch:\ngot prefix:  %q\nwant prefix: %q", firstNChars(got, len(wantHeader)+20), wantHeader)
		}
		if !strings.HasPrefix(want, wantHeader) {
			t.Fatalf("golden node-multi.json header does not match the expected literal — golden fixture may have changed shape:\n%q", firstNChars(want, len(wantHeader)+20))
		}

		// Def SET equality, order-independent (Assumption A3) — both
		// defs must be present on both sides at the exact same
		// (file:line), since this is pure deterministic enumeration.
		gotBlocks, wantBlocks := parseNodeMultiDefBlocks(got), parseNodeMultiDefBlocks(want)
		gotLocs, wantLocs := locationSet(gotBlocks), locationSet(wantBlocks)
		if len(gotLocs) != len(wantLocs) {
			t.Errorf("Node(Validate) def count = %d, want %d (locations: got=%v want=%v)", len(gotLocs), len(wantLocs), gotLocs, wantLocs)
		}
		for loc := range wantLocs {
			if !gotLocs[loc] {
				t.Errorf("Node(Validate) missing expected def at %s (got locations: %v)", loc, gotLocs)
			}
		}

		// Per-def Calls trail: D-05b subset (Go's RefKindCalls-only
		// extraction must never report a call the golden lacks; it may
		// legitimately report fewer, e.g. the golden's "errEmptyOrder"
		// entry, which is a variable reference TS's trail includes and
		// Go's calls-only trail correctly omits).
		gotByLoc, wantByLoc := blockByLocation(gotBlocks), blockByLocation(wantBlocks)
		for loc, wantBlock := range wantByLoc {
			gotBlock, ok := gotByLoc[loc]
			if !ok {
				continue // already reported above
			}
			assertNameFileLineSubset(t, fmt.Sprintf("Node(Validate)@%s Calls trail", loc), gotBlock.Calls, wantBlock.Calls)
		}
	})

	t.Run("explore-multi", func(t *testing.T) {
		// Query per README.md's per-corpus table: "user account"
		// tokenizes to match UserAccountManager (D-03 case b).
		got, err := eng.Explore("user account", 0)
		if err != nil {
			t.Fatalf("Explore(user account, 0): %v", err)
		}
		want := loadGoldenOutputIn(t, corpus, "explore-multi.json")

		// Header wording is TS-verbatim and deterministic.
		wantHeader := "**Exploration: user account**\n\n"
		if !strings.HasPrefix(got, wantHeader) {
			t.Errorf("Explore(user account) header mismatch:\ngot prefix: %q\nwant: %q", firstNChars(got, len(wantHeader)+20), wantHeader)
		}

		// Disclaimer blockquote is a static string (D-05a) — byte-
		// identical on both sides regardless of query/ranking.
		gotDisclaimer := extractDisclaimer(t, got)
		wantDisclaimer := extractDisclaimer(t, want)
		if gotDisclaimer != wantDisclaimer {
			t.Errorf("Explore(user account) disclaimer diverges (must be verbatim):\ngot:  %q\nwant: %q", gotDisclaimer, wantDisclaimer)
		}

		// AD-04: core-symbol/file membership, not full set equality.
		// UserAccountManager (case b's target) and recoverAccount (case
		// c's structurally-connected symbol) must both be surfaced by
		// Go, and their owning files must be among Go's selected files
		// — the specific behaviors this corpus was purpose-built to
		// prove (EXPL-01 multi-word tokenization, EXPL-03's gate
		// preferring the structurally-connected non-test symbol).
		gotFiles := exploreSelectedFiles(got)
		for _, wantFile := range []string{"accounts/manager.go", "recovery/recovery.go"} {
			if !gotFiles[wantFile] {
				t.Errorf("Explore(user account) selected files = %v, want %q among them (AD-04 core-file membership)", gotFiles, wantFile)
			}
		}
		if !strings.Contains(got, "UserAccountManager") {
			t.Error("Explore(user account) output does not mention UserAccountManager (D-03 case b target)")
		}
		if !strings.Contains(got, "recoverAccount") {
			t.Error("Explore(user account) output does not mention recoverAccount (D-03 case c target)")
		}
		// EXPL-03/EXPL-04: recoverAccount is covered by
		// recovery_test.go — both sides must render the "tests:"
		// clause for it (not the "no covering tests" warning).
		if !strings.Contains(want, "tests: `recovery/recovery_test.go`") {
			t.Fatalf("golden explore-multi.json does not contain the expected recoverAccount tests: clause — golden fixture may have changed shape")
		}
		if !strings.Contains(got, "tests:") || !strings.Contains(got, "recovery_test.go") {
			t.Errorf("Explore(user account) output missing recoverAccount's tests: clause (EXPL-03/04):\n%s", got)
		}
	})
}

// TestCorpusBehaviorLockedCorpora scaffold — reserved for 02-03 locked-corpus
// subtests (the external pinned corpora that replace the retired
// network-fetched corpora after FIXT-04).
func TestCorpusBehaviorLockedCorpora(t *testing.T) {
	// Reserved for 02-03 locked-corpus subtests.
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

	client := mcp.NewClient(&mcp.Implementation{Name: "codegraph-golden-parity-test", Version: "0.0.0"}, nil)
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
		{"synthetic-parity", syntheticParitySrc, "user account"},
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
	cases := []struct {
		corpus     string
		sourceFunc func(t *testing.T) string
		symbol     string
	}{
		{"synthetic-parity", syntheticParitySrc, "Validate"},   // multi-def (2)
		{"synthetic-parity", syntheticParitySrc, "AuditEntry"}, // single-def
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
