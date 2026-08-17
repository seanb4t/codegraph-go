# Phase 2: Golden Harness Re-authoring & Re-freeze - Pattern Map

**Mapped:** 2026-08-14
**Files analyzed:** 13 new/modified files
**Analogs found:** 6 / 7 (1 NO ANALOG, auditable below)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `testdata/golden/behavioral_java_test.go` (was `parity_java_test.go`) | test | request-response + transform | `testdata/golden/parity_java_test.go` | exact (self) |
| `testdata/golden/behavioral_tsjs_test.go` (was `parity_tsjs_test.go`) | test | request-response + transform | `testdata/golden/parity_tsjs_test.go` | exact (self) |
| `testdata/golden/behavioral_csharp_test.go` (was `parity_csharp_test.go`) | test | request-response + transform | `testdata/golden/parity_csharp_test.go` | exact (self) |
| `testdata/golden/behavioral_python_test.go` (was `parity_python_test.go`) | test | request-response + transform | `testdata/golden/parity_python_test.go` | exact (self) |
| `testdata/golden/behavioral_test.go` (was `golden_parity_test.go`) | test | request-response + event-driven (MCP) | `testdata/golden/golden_parity_test.go` | exact (self) |
| `testdata/golden/golden_test.go` | test | CRUD (existence/volatility guard) | `testdata/golden/golden_test.go` | exact (self, survives as-is) |
| `testdata/golden/gocapture/main.go` | tool/main | batch (capture) | `testdata/golden/gocapture/main.go` | exact (self, extended) |
| `corpus/behavioral/CASES.json` (new; was `corpus/synthetic-parity/README.md` case map) | data | transform (test input) | `corpora/manifest.json` + the README table | role-match |
| `testdata/golden/README.md` | doc | — | `testdata/golden/README.md` | exact (self) |
| `testdata/golden/corpus/behavioral/` (moved from `synthetic-parity/`) | fixture/corpus | file-I/O | `testdata/golden/corpus/synthetic-parity/` | exact (self) |
| `internal/indexer/capability/matrix.go` + `matrix_test.go` (MODIFIED, forced) | config + test | CRUD (name map) | self | forced by rename |
| `internal/corpora/coverage_test.go` (reference to update) | test | — | self | forced by rename |

## Pattern Assignments

### 1. The per-language behavioral test shape (Q1)

**Analog:** `testdata/golden/parity_java_test.go` (fully read; the other three `parity_{tsjs,csharp,python}_test.go` are the same shape — confirmed `resolve{Lang}Corpus` + `t.Skip` + `indexer.Run` + shape counts via grep).

**Imports pattern** (lines 24-31):
```go
package golden

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/indexer"
)
```

**Corpus-resolution code that changes when tests drive from `internal/corpora`** (lines 38-65) — this is the exact block the planner re-points. Note both the env var and the sibling-checkout fallback are replaced by `internal/corpora Entry.Dir` resolution per D-10:
```go
func resolveJavaCorpus(t *testing.T) string {
	t.Helper()

	if env := os.Getenv("CODEGRAPH_JAVA_CORPUS"); env != "" {
		if info, err := os.Stat(env); err == nil && info.IsDir() {
			return env
		}
		t.Skipf("CODEGRAPH_JAVA_CORPUS=%s is not a directory", env)
	}

	if repoRoot, err := filepath.Abs(filepath.Join("..", "..")); err == nil {
		candidate := filepath.Join(filepath.Dir(repoRoot), "java-corpus")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}

	t.Skip("no Java validation corpus configured — set CODEGRAPH_JAVA_CORPUS=...")
	return ""
}
```

**Core index-and-assert pattern** (lines 84-151) — `indexer.Run` into `t.TempDir()`, open `graphstore.Open` → `Snapshot`, iterate `IterateNodes`/`IterateEdges`, count kinds, assert non-zero per kind, then a second from-scratch run asserting byte-identical `stats.Files/Nodes/Edges` (determinism):
```go
storeDir1 := t.TempDir()
stats1, err := indexer.Run(corpusDir, storeDir1, indexer.Options{Quiet: true})
if err != nil { t.Fatalf("indexer.Run (first pass): %v", err) }
if stats1.Files == 0 { t.Fatalf("indexer.Run found 0 files in the configured Java corpus %s", corpusDir) }

store1, err := graphstore.Open(storeDir1)
...
reader1, err := store1.Snapshot()
...
nit, err := reader1.IterateNodes()
for nit.Next() { n := nit.Node(); if n.Language == "java" { nodeKindCounts[n.Kind]++ } }
...
for _, kind := range []string{"struct", "method"} {
	if nodeKindCounts[kind] == 0 { t.Errorf("shape check: 0 %q nodes extracted ... want > 0", kind) }
}
if edgeKindCounts["calls"] == 0 { t.Error(`shape check: 0 "calls" edges resolved ... want > 0`) }
```

**Flagship finding for the planner:** ALL FOUR per-language tests `t.Skip()` on missing corpus (confirmed `CODEGRAPH_{JAVA,TSJS,CSHARP,PYTHON}_CORPUS` + sibling-checkout + `t.Skip` in every one). This is exactly what D-10/rule `84d1gfpywd`/FIXT-03 says the re-authored hermetic tests must NOT do — they must resolve the locked corpus and fail loudly. The planner replaces this whole skip path, not just its source.

---

### 2. A test that loads a committed data file as its case source (Q2)

**Closest analog (role-match, nearest partial — no exact "case map `CASES.json` loaded by tests" analog exists):** `internal/corpora/coverage_test.go` `TestCorpusCoverageClaim` (lines 123-152), which loads three committed corpus documents from the repo (`corpora/manifest.json`, `observations.json`, `selection.json`) via `Load` and drives assertions from the loaded data. This is the exact shape D-04's `corpus/behavioral/CASES.json` loading should copy:
```go
func TestCorpusCoverageClaim(t *testing.T) {
	m, err := Load(filepath.Join("..", "..", "corpora", "manifest.json"))
	if err != nil { t.Fatalf("Load manifest: %v", err) }
	obs, err := LoadObservations(filepath.Join("..", "..", "corpora", "observations.json"))
	if err != nil { t.Fatalf("LoadObservations: %v", err) }
	...
	wantCorpora := len(LockedEntries(m))
	if res.CheckedCorpora != wantCorpora { t.Fatalf(...) }
	if res.CheckedCorpora == 0 { t.Fatalf("... empty locked set") }
}
```

**Audit of the NO-ANALOG claim:** I searched `rg -l "ReadFile.*\.json|testdata|embed." --glob '*_test.go'` (returned 30 files, all fixture-loading, none a data-driven case map of the CASES.json kind) plus read of `internal/indexer/routes_detect_test.go`, `internal/indexer/capability/matrix_test.go`, `internal/corpora/coverage_test.go`. The committed JSON used as *case source* does not exist; the committed JSON closest to it is the corpora docs loaded above. The nearest "a Go table builds the case list" shape lives in `golden_parity_test.go` — the CLI==MCP `cases := []struct{...}` tables (lines 1551-1564, 1594-1601), which D-04 explicitly replaces with the data file.

**The case-map source of truth (was `corpus/synthetic-parity/README.md`, verbatim):** `CASES.json` must name the four cases — (a) `Validate` defined in `accounts/validate.go`+`orders/validate.go` (overloaded); (b) `UserAccountManager`/`user account` in `accounts/manager.go` (multi-word, EXPL-01); (c) `TestAccountRecovery` (`recovery/recovery_test.go`, zero inbound edges) vs `recoverAccount`/`validateRecovery` (`recovery/recovery.go`) — weakly-connected cluster; (d) `AccountBalanceHelper` (isolated) vs `ReconcileLedger` (heavily connected) in `ledger/ledger.go` — structural-beats-lexical.

---

### 3. gocapture's capture-and-write pattern (Q3)

**Analog:** `testdata/golden/gocapture/main.go` (fully read). The `corpusSpec` seam the planner extends (lines 69-89 + the three spec constructors 224-290):
```go
type corpusSpec struct {
	name string
	resolveSource func() (string, string) // ("", reason) to skip-warn, never hard-fail
	baselineSymbol     string
	baselineSymbolFile string
	baselineQuery      string
	multiSymbol string
	multiQuery  string
}
specs := []corpusSpec{ weftGoSpec(), colbymchenryCodegraphSpec(), syntheticParitySpec(goldenDir) }
```
**main() drive loop** (lines 91-115): `runtime.Caller(0)` → `goldenDir`/`corpusDir` derivation (`filepath.Dir(filepath.Dir(thisFile))` — NOTE this computes `testdata/golden`, so when a locked-corpus golden must land under `testdata/golden/corpus/<lang>/` the existing `corpusDir` join works; when it lands under `corpus/behavioral/` the path derivation must change), iterate specs calling `regenerateCorpus`, count failures, `os.Exit(1)` on any.

**Core capture body** `regenerateCorpus` (lines 117-207) — the exact shape to replicate for each locked corpus: resolve source → `os.MkdirTemp` throwaway store → `indexer.Run` → `graphstore.Open` → `Snapshot` → `query.NewWithRoot(reader, sourcePath)` → `eng.Explore(maxFiles)` / `eng.Node` → `writeCapture`. The write is **direct to the committed path** (line 148 `out := filepath.Join(corpusDir, spec.name)`), which is the anti-pattern the re-freeze should convert to capture-to-temp-then-move (research Pattern 2, carryover from 01-03). The multi-write is best-effort/warn-not-fatal (lines 182-203) which research Assumption A3 flags as a risk for locked corpora.

**writeCapture** (lines 209-219) is the `{command, output}` JSON envelope writer:
```go
func writeCapture(path, command, output string) error {
	data, err := json.MarshalIndent(goldenCapture{Command: command, Output: output}, "", "  ")
	if err != nil { return fmt.Errorf("marshal %s: %w", path, err) }
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil { return fmt.Errorf("write %s: %w", path, err) }
	return nil
}
```

**Extension seam per D-05:** the shared `Entry.Dir(root)` resolver replaces each spec's `resolveSource` (which today shell out to `git clone` for colbymchenry and read `WEFT_REPO` for weft). `syntheticParitySpec` (lines 276-290) is the "committed in-repo source" template — its `src` path literal `filepath.Join(goldenDir, "corpus", "synthetic-parity", "src")` must become `corpus/behavioral/src`.

---

### 4. How tests currently resolve a corpus path + the skip precedent (Q4)

**Analog:** `testdata/golden/golden_parity_test.go`.

`resolveWeftCorpus` (lines 116-158) — env `CODEGRAPH_WEFT_CORPUS` → sibling `../weft`, validates `gitHead` against a hardcoded `pinnedWeftCommit` (`f89ae3ea...`, line 67), and **`t.Skipf` on every failure mode** (not-dir, rev-parse fail, wrong commit):
```go
head, err := gitHead(c.path)
...
if head != pinnedWeftCommit {
	reasons = append(reasons, fmt.Sprintf("%s at %s: at commit %s, want pinned %s", ...))
	continue
}
...
t.Skipf("weft corpus unavailable at the pinned commit %s (tried: %s) — ...", pinnedWeftCommit, reasons)
return ""
```
`resolveWeftGoCorpusLoose` (lines 240-253) and `resolveColbymchenryCorpus` (lines 261-271) both **`t.Skipf`** on absence/clone-failure too.

**Explicit note for the planner:** research already records `parity_*_test.go`'s `t.Skip` (Q2 open question) AND `resolveWeftCorpus`'s loud-skip discipline (the file's own doc comment: "loud skip, never a silent pass or a hard CI failure"). The re-authored hermetic tests must NOT follow any of these `t.Skip` precedents for the locked corpora — D-10 + FIXT-03 direction + rule `84d1gfpywd` require failing loudly. The skip-swallowing that is fine to keep is gocapture's standalone `resolveSource` skip-warn (a `main` program with no `*testing.T` to skip with, lines 74-77, 117-122).

---

### 5. How `internal/corpora` resolves a locked corpus to a filesystem path (Q5)

**Analog:** `internal/corpora/manifest.go` (lines 188-229). `Entry.Dir(root)` + `CorpusRoot()` + `LockedEntries(m)`:
```go
func (e Entry) Dir(root string) string {
	digest := sha256.Sum256([]byte(e.Repo))
	short := hex.EncodeToString(digest[:])[:8]
	return filepath.Join(root, fmt.Sprintf("%s-%s@%s", e.Slug(), short, e.SHA))
}

func CorpusRoot() (string, error) {
	if v := os.Getenv("CODEGRAPH_CORPUS_DIR"); v != "" { return v, nil }
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" { return filepath.Join(v, "codegraph", "corpora"), nil }
	home, err := os.UserHomeDir()
	if err != nil { return "", fmt.Errorf("corpora: resolve home directory: %w", err) }
	return filepath.Join(home, ".cache", "codegraph", "corpora"), nil
}

func LockedEntries(m Manifest) []Entry { /* returns only Locked entries, in order */ }
```

**The exact call a test/gocapture makes to get a pinned tree path:**
```go
root, err :=  corpora.CorpusRoot()
for _, e := range corpora.LockedEntries(m) {   // m from corpora.Load("corpora/manifest.json")
	path := e.Dir(root)                          // <root>/<slug>-<hash8>@<pinnedSHA>
}
```

---

### 6. A test that indexes a repo and asserts on the graph (Q6)

**Closest analog (role-match, strong):** `internal/indexer/routes_detect_test.go` — `runAndSnapshot` + kind/edge-count assertions over a committed `testdata/routesfixture/` repo. This is the shape D-09's property-assertion tests should follow: index a repo, open a snapshot, assert named node/edge properties.
```go
func runAndSnapshot(t *testing.T, repoRoot string) (graphstore.Reader, Stats) {
	t.Helper()
	storeDir := t.TempDir()
	stats, err := Run(repoRoot, storeDir, Options{})
	if err != nil { t.Fatalf("Run(%s): %v", repoRoot, err) }
	store, err := graphstore.Open(storeDir)
	...
	r, err := store.Snapshot()
	...
	return r, stats
}
```
and the property assertion (lines 91-97): `routeNodes := collectRouteNodes(t, r); if len(routeNodes) != 2 { t.Fatalf(...) }` — collecting nodes of a specific kind via `r.IterateNodes()` and asserting an exact count.

**Second analog (node/edge-kind counts over a corpus):** `internal/indexer/parallel_test.go`'s sibling `parity_java_test.go` (Q1, above) — its `nodeKindCounts`/`edgeKindCounts` maps and non-zero-per-kind assertions are the D-09 "locked-corpus shape/count/edge-coverage invariants" template. Both are the property-assertion (option b) style research recommends (Open Question 1).

**Also relevant (assertion via `query.Engine` result types):** `loadGoldenFixture[T]`/`loadGoldenFixtureIn[T]` (golden_parity_test.go lines 406-468) decode fixtures directly into `internal/query` result types (`query.CallersResult` etc.) — the field-for-field decode seam if goldens are kept as byte oracles.

---

### 7. The rename's mechanics — what breaks on the `parity_*` / `TestGoldenParity*` rename (Q7)

**Critical mechanical break (the one `go test ./...` WILL catch):** `internal/indexer/capability/matrix_test.go` `TestMatrix_FullPriority4EntriesHaveGoldenTest` (lines 223-238) parses every `testdata/golden/*_test.go` with `go/parser` and asserts the mapped `TestGoldenParity*` function names actually exist:
```go
var goldenTestFuncsByLanguage = map[string]string{
	"go":         "TestGoldenParity",
	"java":       "TestGoldenParity_Java",
	"csharp":     "TestGoldenParity_CSharp",
	"python":     "TestGoldenParity_Python",
	"typescript": "TestGoldenParity_TSJS",
	"tsx":        "TestGoldenParity_TSJS",
	"javascript": "TestGoldenParity_TSJS",
}
// ...
if !declared[wantFuncName] {
	t.Errorf("%s: Resolution/Dispatch is full but %s is not declared under testdata/golden/", id, wantFuncName)
}
```
**Renaming `TestGoldenParity*` in `testdata/golden/` FAILS this test unless (a) `goldenTestFuncsByLanguage` in matrix_test.go AND (b) the doc strings in `internal/indexer/capability/matrix.go` (lines 113,124,136,147,158,170) are updated in the same diff.** This is the grep-able identifier map that catches a stale reference. `docs/LANGUAGE-CAPABILITY-MATRIX.md` also carries the names but is derived/documented, not a compile/test gate.

**Also breaking mechanically:** every `TestGoldenParity*` reference *within* `testdata/golden/golden_parity_test.go` itself (6 occurrences) and each per-language file's doc-comment naming its own test — these rename in-place. The `Test*CLIMatchesMCP` trio is self-consistent and survives (research Pattern 1 keeps them). `TestGoSideFixturesRegenerated` (golden_test.go line 158) reads `corpus/synthetic-parity/...` — it breaks on the corpus move (FIXT-05) unless re-pointed to the moved path. `tools/bench/realcorpus/manifest.go`'s `colbymchenry-codegraph` ref is OUT of scope (D-08 keeps it).

**No free-standing "stale test-name reference" checker exists** beyond the matrix test above; the research recommends `rg "parity" testdata/golden/` → expected empty as the rename acceptance gate (CODE-02 validation map).

---

## NO ANALOG (with auditable search)

| File | Role | Data Flow | Reason / Nearest Partial |
|------|------|-----------|--------------------------|
| `corpus/behavioral/CASES.json` (D-04 case map loaded by tests) | data | transform | **NO ANALOG.** Search run: `rg -l "ReadFile.*\.json\|testdata\|embed." --glob '*_test.go'` (30 hits, all fixture-loading) + read of `internal/indexer/routes_detect_test.go`, `capability/matrix_test.go`, `corpora/coverage_test.go`. No test loads a committed case-map data file as its case source. Nearest partial: `corpora` doc-loading in `coverage_test.go:123-132` (committed-JSON-as-input), and the Go-table `cases := []struct{}` in `golden_parity_test.go:1551-1601` (the exact shape D-04 replaces). |

The four per-language behavioral tests and the two corpus-driven tests (`TestGoldenBehavioralSyntheticParity`/`TestGoldenBehavioralRealCorpora`, lines 1180-1357) are re-pointed from the `synthetic-parity` path to `corpus/behavioral/` — their assertion bodies (header-template matches, def-set equality via `parseNodeMultiDefBlocks`, `exploreSelectedFiles` membership, warning-clause checks) are unchanged, only the corpus-path resolution and case source change.

## Shared Patterns

### Hermetic corpus resolution (D-10, applies to all locked-corpus tests + gocapture)
**Source:** `internal/corpora/manifest.go:188-217` (`Entry.Dir` + `CorpusRoot`)
**Apply to:** all `behavioral_*_test.go`, `behavioral_test.go`, `gocapture`
Resolve every locked corpus via `corpora.CorpusRoot()` + `e.Dir(root)` against `corpora.Load("corpora/manifest.json")`. Never hardcode a SHA, never a user env default, never `t.Skip`.

### Capture-to-temp-then-move (FIXT-06, apply to gocapture's extension)
**Source:** `.planning/phases/01-corpus-selection-by-measurement/01-03-SUMMARY.md` (Pattern 2)
**Apply to:** every golden `gocapture` writes
Write to a temp path, assert non-empty + marker-bearing, THEN move onto the committed path — the current `writeCapture` (gocapture/main.go:209-219) writes straight to the destination and must gain the temp-then-move step.

### The index+snapshot+iterate-assert test skeleton
**Source:** `internal/indexer/routes_detect_test.go:16-60` (`runAndSnapshot` + `collectRouteNodes`)
**Apply to:** D-09's locked-corpus property-assertion tests
`Run` → `graphstore.Open` → `Snapshot` → `IterateNodes`/`IterateEdges` → assert named kind/edge invariants. Closely matches `parity_java_test.go:84-151`.

### CLI==MCP byte-identity harness (for the MCP-surface re-freeze clause)
**Source:** `testdata/golden/golden_parity_test.go:1446-1588` (`newGoldenSession`, `callExploreViaMCP`, `callNodeViaMCP`, `mcpResultText`)
**Apply to:** `gocapture`'s MCP-surface capture + the re-authored CLI==MCP trio
In-process `internalmcp.BuildServer` + go-sdk client — never a live TS server.

### Byte-identity freeze enforcement
**Source:** `test/wireoracle/oracle_test.go` `TestFrozenTranscriptsMatch`
**Apply to:** the re-freeze — a re-freeze must satisfy a byte-identity test over the goldens; the rename must not move a single golden byte. (Re-freeze model: `testdata/golden/corpus/` JSON fixtures are `{command, output}` `goldenCapture` envelopes.)

## Metadata

**Analog search scope:** `testdata/golden/*_test.go` (all 6 parsed/read), `testdata/golden/gocapture/main.go`, `internal/corpora/manifest.go` + `coverage_test.go`, `internal/indexer/capability/matrix.go` + `matrix_test.go`, `internal/indexer/routes_detect_test.go`, `corpus/synthetic-parity/README.md`, and a tree-wide `rg` of `TestGoldenParity|parity_|synthetic-parity|golden_parity` across non-fixture files to map the rename blast radius.
**Files scanned:** 14
**Pattern extraction date:** 2026-08-14
