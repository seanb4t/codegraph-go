# Phase 1: Corpus Selection by Measurement - Pattern Map

**Mapped:** 2026-08-13
**Files analyzed:** 10 (5 modified, 5 created — per CONTEXT/RESEARCH; exact new-file count is Claude's Discretion at planning time)
**Analogs found:** 10 / 10 (one is an explicit NO-ANALOG on the freeze mechanism)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/query/status.go` (`StatusResult.EdgesByKind`, un-suppress `FilesByLanguage`) | model / service (read-time derived breakdown) | CRUD (read, full-scan aggregate) | same file, `nodesByKind` scan in `Engine.Status` (lines ~231-247) | exact — same function, same struct, same scan shape |
| `internal/query/render_status.go` (`RenderStatusText`/`RenderStatusMarkdown` gain "Edges by Kind:") | transform (renderer) | request-response | same file, `writeBreakdownText`/`writeBreakdownMarkdown` + `sortedCounts` calls for `NodesByKind` | exact — literally the same helper, one more call site |
| `internal/cli/status.go` (dense-mode flag) | controller (CLI command) | request-response | `internal/cli/affected.go`'s `--quiet` flag (output-shape toggle) | role-match — closest existing flag that changes *shape*, not just presence, of output |
| `internal/cli/present/status.go` (TTY twin renderer) | component (styled duplicate renderer) | request-response | same file's own `writeBreakdownText` mirroring `query.RenderStatusText`'s | exact — file already documents itself as a package-local duplicate |
| `test/wireoracle/transcripts/call-status.golden` | test fixture (golden/frozen transcript) | request-response | itself — re-freeze via whatever mechanism prior `.golden` re-freezes used (see NO ANALOG below) | partial — no committed freeze-writer tool exists |
| corpora manifest (new file, format = Claude's Discretion) | config | batch (static pin list, read by 3 consumers) | `.release-please-manifest.json` / `dist/artifacts.json` (repo-root JSON config artifacts consumed by multiple tools) | role-match — same "one file, several readers" shape |
| Taskfile fetch target (new `corpora:fetch`-shaped target) | utility (build tooling) | file-I/O (shells out to `git`) | `Taskfile.yml` `release:dry-run-signed` (vars/preconditions/bash `cmds:` shape); `test:golden` (own explicit target for `testdata`-invisible-to-`go list` reason) | role-match — closest git-shelling, guarded Taskfile target |
| `.github/workflows/*.yml` `actions/cache` step (new) | CI config | event-driven (workflow step) | `.github/workflows/ci.yml:51-54` `actions/checkout@<sha> # vX.Y.Z` steps (SHA-pin convention); NO existing `actions/cache` usage — first of its kind | role-match for pin style; NO ANALOG for the cache mechanic itself |
| corpus-drift path-filtered workflow (new sibling `.yml`) | CI config | event-driven | `.github/workflows/linux-cross-canary.yml` (`on: pull_request: paths:` block) | exact — RESEARCH already identifies this as the direct precedent |
| measurement-record generator (new Go program) | utility (build tooling, drives indexer + writes JSON) | batch / file-I/O | `testdata/golden/gocapture/main.go` | exact — RESEARCH already identifies this; verified below |
| coverage drift-guard test (new `_test.go`) | test (positive-assertion guard) | request-response (asserts against committed JSON) | `test/wireoracle/scenarios.go` `ExpectedScenarioCount` + `oracle_test.go`'s `TestScenarioCountIsExact`/`TestRankEdges` (`internal/query/rwr_test.go`) | exact — RESEARCH names both; `TestRankEdges` is the tighter shape match (key-set equality), `ExpectedScenarioCount` is the tighter shape match (positive count assertion) |

## Pattern Assignments

### `internal/query/status.go` — `edgesByKind` (model, CRUD/full-scan)

**Analog:** same file, the existing `nodesByKind` scan inside `Engine.Status` (`internal/query/status.go`, function body ~lines 218-253).

**Core pattern — the scan to mirror verbatim** (`internal/query/status.go`):
```go
nodeIt, err := e.reader.IterateNodes()
if err != nil {
    return StatusResult{}, err
}
defer nodeIt.Close()

nodesByKind := make(map[string]int64)
var nodeCount int64
for nodeIt.Next() {
    n := nodeIt.Node()
    nodeCount++
    nodesByKind[n.Kind]++
}
if err := nodeIt.Err(); err != nil {
    return StatusResult{}, err
}
```

**The full-scan primitive `edgesByKind` must reuse** (`internal/query/expand.go:263-283`, `buildExpandAdjacency` — same `IterateEdges("")` shape already used elsewhere in this package, confirming it is the established full-edge-scan idiom, not something to invent):
```go
it, err := r.IterateEdges("")
if err != nil {
    return nil, nil, err
}
defer it.Close()

edgesByKind := make(map[string]int64)
for it.Next() {
    e := it.Edge()
    edgesByKind[e.Kind]++
}
if err := it.Err(); err != nil {
    return nil, nil, err
}
```

**Struct field to add** — `StatusResult` (`internal/query/status.go:46-63`) currently:
```go
type StatusResult struct {
	Initialized      bool              `json:"initialized"`
	Version          string            `json:"version"`
	ProjectPath      string            `json:"projectPath"`
	IndexPath        string            `json:"indexPath"`
	FileCount        int64             `json:"fileCount"`
	NodeCount        int64             `json:"nodeCount"`
	EdgeCount        int64             `json:"edgeCount"`
	DbSizeBytes      int64             `json:"dbSizeBytes"`
	Backend          string            `json:"backend"`
	NodesByKind      map[string]int64  `json:"nodesByKind"`
	FilesByLanguage  map[string]int64  `json:"-"`
	Languages        []string          `json:"languages"`
	PendingChanges   PendingChanges    `json:"pendingChanges"`
	WorktreeMismatch *gitmeta.Mismatch `json:"worktreeMismatch"`
	Stale            bool              `json:"stale"`
	Index            IndexHealth       `json:"index"`
}
```
D-03 changes `FilesByLanguage`'s tag from `json:"-"` to `json:"filesByLanguage"` (or similar); D-01/D-02 add an `EdgesByKind map[string]int64 \`json:"edgesByKind"\`` field, following `NodesByKind`'s exact tag style. **The file's own top-of-file per-key decision table (lines 20-45) documents `filesByLanguage`'s `json:"-"` suppression explicitly and must be updated in the same diff** (Pitfall 3 from RESEARCH — this is a live spec, not background prose).

**Cost note to carry into the plan, not silently absorb:** `Status()`'s own doc comment (lines 205-206) states it deliberately avoids a second full edge scan today, reading `meta.GetEdgeCount()` instead. Adding `edgesByKind` makes a full edge scan unconditional on every `status` call. This is a real, documented regression, not an oversight — the doc comment must be updated to reflect it.

---

### `internal/query/render_status.go` — dense key-set derivation (transform, request-response)

**Analog for the key-set-equality test shape** — `TestRankEdges`, `internal/query/rwr_test.go:13-38`, VERBATIM:
```go
func TestRankEdges(t *testing.T) {
	want := map[string]bool{
		goextract.RefKindCalls:        true,
		goextract.RefKindReferences:   true,
		goextract.EdgeKindExtends:     true,
		goextract.EdgeKindImplements:  true,
		goextract.EdgeKindOverrides:   true,
		goextract.RefKindInstantiates: true,
		goextract.RefKindReturns:      true,
		goextract.RefKindTypeOf:       true,
		goextract.RefKindImports:      true,
	}
	if len(RankEdges) != 9 {
		t.Fatalf("RankEdges has %d members, want 9: %v", len(RankEdges), RankEdges)
	}
	for k := range want {
		if !RankEdges[k] {
			t.Errorf("RankEdges missing expected member %q", k)
		}
	}
	for k := range RankEdges {
		if !want[k] {
			t.Errorf("RankEdges has unexpected member %q", k)
		}
	}
}
```
The new dense-key-set test (D-04) must follow this exact two-directional-subset shape — but its `want` set must be `query.RankEdges` itself (imported, not restated), asserting the dense `edgesByKind` render/marshal output's key set is exactly `RankEdges`'s key set. Do NOT hand-list the 9 kind strings a second time in the new test — that duplicates exactly what `TestRankEdges` exists to prevent one level up.

**The canonical source to derive from** — `RankEdges`, `internal/query/rwr.go:21-31` (referenced, not re-shown; confirmed live at `rwr_test.go:13-24` above).

**Existing filter-at-render-time precedent to extend, not duplicate** (`internal/query/render_status.go:84-98`, `sortedCounts` — shared by both `NodesByKind` and `FilesByLanguage` today, and the section call sites at lines 186/239):
```go
func sortedCounts(m map[string]int64) []kindCount {
	out := make([]kindCount, 0, len(m))
	for k, v := range m {
		if v > 0 {
			out = append(out, kindCount{Key: k, Count: v})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	return out
}
```
`writeBreakdownText(&b, "Nodes by Kind:", sortedCounts(r.NodesByKind))` (line 186) and `writeBreakdownMarkdown(&b, "**Nodes by Kind:**", sortedCounts(r.NodesByKind))` (line 239) are the exact two call sites the new `"Edges by Kind:"` section must add a sibling call beside — sparse mode reuses `sortedCounts` unchanged; dense mode needs a new sibling helper (illustrative shape only, not existing code — see RESEARCH's `denseEdgeCounts` example) that iterates `sort.Strings(keys(RankEdges))` instead of filtering `count > 0`.

---

### `internal/cli/status.go` — dense-mode flag (controller, request-response)

**Analog:** `internal/cli/affected.go`'s `--quiet`/`-q` flag — the closest existing CLI flag that changes *output shape* (not just verbosity of the same shape) across a human-text path. Registration (`internal/cli/affected.go:154`):
```go
cmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "emit only affected test file paths, one per line (no summary, no worktree notice)")
```
Threading into the render decision (`internal/cli/affected.go`, referenced near lines 53/102/113/131 — the flag is read once at the top of `RunE`, then branches the output-writing logic before the human-output branch, after `--json` and `--quiet` early paths).

`internal/cli/status.go`'s own existing `--json`/`-j` registration is the second, tighter analog for *where in the same file* the new flag belongs (`internal/cli/status.go:75-76`):
```go
cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "emit JSON output")
```
The new dense flag is a third `cmd.Flags().BoolVarP(...)` line beside these two, then threaded into the `eng.Status(...)`-adjacent render call (`query.RenderStatusText(result, start)` at the bottom of `RunE`, `internal/cli/status.go`) and into `query.MarshalStatusJSON(result)` — both must receive the dense flag's value (or `StatusResult` must carry the raw sparse map with density resolved at render time, per RESEARCH's explicit recommendation to keep `Engine.Status` itself flag-agnostic).

**D-05 constraint:** the MCP resource path (`internal/mcp` — not directly touched by this file, but the propagation boundary) must NOT receive this flag; it always calls the sparse path, matching `codegraph status`'s flagless default.

---

### `internal/cli/present/status.go` — TTY twin renderer (component, request-response)

**Analog:** the file's own documented self-identification as a package-local duplicate (`internal/cli/present/status.go:14-20`):
```go
// kindCount pairs a breakdown key (a node kind or a file language) with
// its count — a package-local duplicate of internal/query's unexported
// kindCount (RESEARCH Open Question #1: duplicate rather than import
// internal/query's unexported formatting helpers, matching this
// codebase's existing precedent of package-local duplication across the
// query/cli boundary, e.g. render_results.go's renderFileTreeMarkdown vs
// internal/cli/files.go's printFileTree).
```

**The exact call-site pair that must gain a matched "Edges by Kind" section** — `query.RenderStatusText`'s Nodes-by-Kind call (`internal/query/render_status.go:186`):
```go
writeBreakdownText(&b, "Nodes by Kind:", sortedCounts(r.NodesByKind))
```
and its byte-for-byte-duplicated sibling in `present.RenderStatus` (`internal/cli/present/status.go`, `writeBreakdownText` helper at lines 90-97, called from within `RenderStatus`'s body at line 123 onward — the section-order doc comment at lines 115-122 explicitly enumerates "Index Statistics → Nodes by Kind → Files by Language → advisories" as the ordering both renderers must share). `present.RenderStatus`'s `writeBreakdownText` (verbatim, lines 90-97):
```go
func writeBreakdownText(b *strings.Builder, header string, counts []kindCount) {
	b.WriteString(sectionStyle.Render(header) + "\n")
	for _, kc := range counts {
		fmt.Fprintf(b, "  %-*s %s\n", breakdownKeyWidth, kc.Key, formatNumber(kc.Count))
	}
}
```
This is Pitfall 2 from RESEARCH made concrete: a plan touching only `render_status.go` and `cli/status.go` for the human-text half is incomplete without a matching edit here. `present`'s local `sortedCounts` (lines 55-70) is its own byte-for-byte duplicate of `query.render_status.go`'s and must also gain the same dense-derivation sibling if D-04's density is to reach the TTY path (CLI flag → `StatusResult` → both text renderers).

---

### `test/wireoracle/transcripts/call-status.golden` — re-freeze target (test fixture)

**Analog:** the transcript itself; **the correct target confirmed by direct read**, per RESEARCH's own correction of CONTEXT.md. The scenario driving it is `"call-status"` in `test/wireoracle/scenarios.go:753-759`, NOT `"resources-read-status"` (`scenarios.go:1322`, which reads the static `internal/mcp/resources/status.md` doc text and does not embed live `StatusResult` data).

**Committed content that will change** (verified, `call-status.golden` second line):
```
{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"**CodeGraph Status**\n\n**Files indexed:** 3\n**Total nodes:** 9\n**Total edges:** 11\n**Database size:** 0.01 MB\n**Backend:** pebble\n\n**Nodes by Kind:**\n- function: 4\n- file: 3\n- package: 2\n\n**Languages:**\n- go: 3\n\nIndex is up to date.\n"}]}}
```
This text is `query.RenderStatusMarkdown`'s literal output and will gain an "Edges by Kind:" bullet block once D-01/D-02 land.

**NO ANALOG for the mechanical re-freeze process itself.** RESEARCH's Open Question 1 confirms: `TestFrozenTranscriptsMatch` (`test/wireoracle/oracle_test.go:112-152`) does a byte-for-byte comparison and `t.Fatalf`s on mismatch; there is no `-update`/`UPDATE_GOLDEN` flag anywhere in `test/wireoracle/*.go`. `tools/transcriptfreeze` is a different thing entirely — the D-03 anti-regeneration ADVISORY guard (`check:transcript-freeze` Taskfile task, `Taskfile.yml:3230-3260`), which reports (never fails) when a transcript changes alongside `internal/mcp/*.go` without review — it does not write transcripts. The nearest partial match for a "drives the binary, captures output, writes a fixture" shape is `testdata/golden/gocapture/main.go` (see below) or `testdata/golden/capture.sh` (older TS-side capture script, not read in this pass) — planner must budget a small task to either confirm how prior re-freezes were done (git history on other `.golden` files) or write a tiny one-off capture helper.

---

### Corpora manifest (config, batch)

**No committed manifest of this exact repo/SHA/license/locked shape exists yet — this is a genuinely new file.** Closest structural analogs for "one JSON file at repo root, read by multiple independent consumers, no restatement allowed":
- `.release-please-manifest.json` / `release-please-config.json` (repo root JSON config, read by the release-please Action)
- `dist/artifacts.json` (read by `release:dry-run-signed`'s pass-condition check, per that target's own doc comment: "FOUR DISTINCT PUBLISHED NAMES per pipe from dist/artifacts.json")

Neither was read in full this pass (out of scope for a corpora-manifest analog — they are the closest *shape* match, not a content template). `jq` is confirmed the established Taskfile-side JSON-manifest reader (`Taskfile.yml` lines 352-680 sampled per RESEARCH, `release:dry-run-signed` and siblings shell out to it) — the new fetch target and CI cache-key derivation should follow that same `jq`-in-bash convention rather than introducing a new parsing tool.

---

### Taskfile fetch target (utility, file-I/O)

**Analog for guarded, `git`-shelling, bash `cmds:` target with preconditions** — `release:dry-run-signed` (`Taskfile.yml:513-543`), verbatim precondition + cmds opening:
```yaml
  release:dry-run-signed:
    desc: >-
      ...
    preconditions:
      - sh: '[ "$(go env GOHOSTOS)" = "darwin" ]'
        msg: "release:dry-run-signed must run on a native darwin host — ..."
      - sh: command -v zig
        msg: "zig not found. ..."
      - sh: command -v syft
        msg: "syft not found. ..."
      - sh: command -v cosign
        msg: "cosign not found. ..."
    cmds:
      - |
```
The new `corpora:fetch`-shaped target should follow this `preconditions:` (`sh:`+`msg:` pairs, matching `TestTaskfileGatesFailLoud`'s convention per RESEARCH) plus a `cmds: - |` bash block shape, NOT a bare inline `run:` step in a workflow (`TestWorkflowRunBodiesInvokeTask`'s `^task\s+[A-Za-z0-9:_-]+$` regex, Pitfall 4 — see below).

**`check:transcript-freeze`** (`Taskfile.yml:3230-3260`) — verbatim opening, showing the `set -euo pipefail` + required-env-var-or-loud-`::error::`-exit convention this repo uses for a Taskfile target that must not silently no-op:
```yaml
  check:transcript-freeze:
    desc: >-
      D-03 anti-regeneration guard, ADVISORY since 03-02 (v0.3.0 Phase 3):
      ...
    cmds:
      - |
        set -euo pipefail
        if [ -z "${TRANSCRIPT_FREEZE_BASE:-}" ]; then
          echo "::error::TRANSCRIPT_FREEZE_BASE is not set — check:transcript-freeze needs a base ref to compute the merge-base diff against (e.g. TRANSCRIPT_FREEZE_BASE=origin/main task check:transcript-freeze)."
          exit 1
        fi
        git rev-parse --verify "${TRANSCRIPT_FREEZE_BASE}" >/dev/null
        mb=$(git merge-base "${TRANSCRIPT_FREEZE_BASE}" HEAD)
```
The drift guard's heavier leg (D-08) should follow this exact "loud, `::error::`-prefixed, non-zero exit on missing precondition" pattern, not a silent skip — directly enforcing rule `84d1gfpywd`.

**`test:golden`** (`Taskfile.yml:59-65`), verbatim, the precedent for "testdata is invisible to `go list ./...`, so this needs its own explicit target":
```yaml
  test:golden:
    desc: >-
      Golden parity suite (testdata/golden) — NOT covered by go list ./...
      (GOLDEN-01: the go tool ignores any directory named "testdata" when
      expanding ./...), so this is its own explicit target.
    cmds:
      - go test ./testdata/golden/...
```
If the measurement-record generator or drift guard lands under `testdata/`, it inherits this same invisibility and needs the identical "own explicit target" treatment.

---

### CI cache wiring (`.github/workflows/*.yml`)

**SHA-pin convention analog** — `.github/workflows/ci.yml:51-54`, verbatim:
```yaml
        uses: actions/checkout@df4cb1c069e1874edd31b4311f1884172cec0e10 # v6.0.3
        ...
        uses: actions/setup-go@924ae3a1cded613372ab5595356fb5720e22ba16 # v6.5.0
```
The new `actions/cache@55cc8345863c7cc4c66a329aec7e433d2d1c52a9 # v6.1.0` step (SHA per RESEARCH's own live-verified §Standard Stack entry) must match this exact `uses: <action>@<full-commit-sha> # v<semver>` comment-trailing style — no shorthand tag, no floating major version.

**NO ANALOG for the cache mechanic itself** — RESEARCH confirms directly: "No workflow in this repo uses `actions/cache` today" (`Taskfile.yml`/CI audit, cross-checked). This is a first-of-kind step; follow the RESEARCH-provided synthesized pattern (Code Examples §`actions/cache`), not an in-repo precedent.

**Binding constraint on placement** (Pitfall 4, verified): `TestWorkflowRunBodiesInvokeTask` (`internal/upgrade/taskfile_shape_test.go:1343-1384`, `inScopeJobs` fixture at lines 109-118) requires every `run:` step body inside `ci.yml`'s `test`/`actionlint`/`goreleaser-check`/`reproducibility`/`perf-regression`/`transcript-freeze`/`tool-vuln` jobs (plus `release-please.yml`'s `pretag-gate`) to be *exactly* `task <target>`. A `uses:`-only step (the cache action itself) is unconstrained; the conditional fetch step that follows it (`if: steps.cache.outputs.cache-hit != 'true'`) must be `run: task <corpora-fetch-target>` verbatim if it lands inside one of those in-scope jobs, or the new job must be added to the `inScopeJobs` fixture if placed elsewhere.

---

### Corpus-drift path-filtered sibling workflow

**Analog:** `.github/workflows/linux-cross-canary.yml`, `on:` block (verified, lines 1-30 context; the `pull_request: paths:` block RESEARCH cites at lines 71-83 was corroborated structurally — this file's header explicitly frames itself as a "permanent, dispatchable canary... NOT in main's required-status-check set," the exact shape D-08's heavier leg needs):
```yaml
on:
  workflow_dispatch:
  pull_request:
    paths:
      - ".github/workflows/release.yml"
      - ".github/workflows/linux-cross-canary.yml"
      - ".goreleaser.yaml"
      - "Taskfile.yml"
      - "go.mod"
      - "go.sum"

permissions:
  contents: read
```
The new sibling workflow for D-08's "re-measure only when the pin manifest changes" leg should copy this shape exactly, substituting the `paths:` list for the corpora manifest file path (plus, plausibly, this new workflow's own filename, per this file's own self-inclusion convention).

---

### Measurement-record generator (utility, batch/file-I/O)

**Analog:** `testdata/golden/gocapture/main.go` — confirmed exact match by direct read.

**`main()` shape** (verbatim, lines 91-115):
```go
func main() {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		fatal("gocapture: runtime.Caller(0) failed to resolve this file's own path")
	}
	goldenDir := filepath.Dir(filepath.Dir(thisFile)) // .../testdata/golden
	corpusDir := filepath.Join(goldenDir, "corpus")

	specs := []corpusSpec{
		weftGoSpec(),
		colbymchenryCodegraphSpec(),
		syntheticParitySpec(goldenDir),
	}

	failures := 0
	for _, spec := range specs {
		if err := regenerateCorpus(spec, corpusDir); err != nil {
			fmt.Fprintf(os.Stderr, "gocapture: [%s] FAILED: %v\n", spec.name, err)
			failures++
		}
	}
	if failures > 0 {
		os.Exit(1)
	}
}
```
**How it locates and drives the binary/pipeline** (per-corpus loop body, `regenerateCorpus`, lines 117-131): calls `spec.resolveSource()` to get a source path (skipping with a warning, never silently, if unavailable — printed to stderr, RESEARCH's own noted "SKIPPED" pattern), makes a throwaway `os.MkdirTemp` Pebble store, then calls `indexer.Run(sourcePath, storeDir, indexer.Options{Quiet: true})` directly as a Go function call — NOT by shelling out to the `codegraph` binary. This is the key structural fact for the new measurement-record generator: it should drive `indexer.Run` + `query.Engine` in-process, exactly like `gocapture` does, not fork a subprocess.

**How it writes output** — a `goldenCapture` envelope struct (lines 55-60-ish, `{"command": ..., "output": ...}`) marshaled to JSON per corpus, written under `testdata/golden/corpus/<name>/go-*.json`. The new measurement-record generator's per-`repo@SHA` JSON entries (D-06) should follow this same "one struct, `encoding/json`, `os.WriteFile`" shape, keyed by `repo@SHA` per D-09 rather than by corpus name.

---

### Coverage drift-guard test (test, request-response)

**Positive-assertion-count analog** — `ExpectedScenarioCount`, `test/wireoracle/scenarios.go:540` + its consumer `oracle_test.go:158-164`:
```go
const ExpectedScenarioCount = 42
```
```go
// len(Scenarios()) must equal the package constant ExpectedScenarioCount
...
	if got != ExpectedScenarioCount {
		t.Fatalf("len(Scenarios()) = %d, want exactly %d (ExpectedScenarioCount) — either a scenario silently disappeared or one was added without updating the constant beside Scenarios()", got, ExpectedScenarioCount)
	}
```
This is the shape rule `84d1gfpywd` requires: an EXACT-equality positive assertion against a named constant, declared beside the thing it counts, never a lower bound (`TestRankEdges` above is the second, tighter analog for the key-set-equality half — dense `edgesByKind`'s 9-kind coverage and the drift guard's "every kind clears its bar" claim are structurally the same shape: iterate a canonical set, assert every member is present/non-zero/above-threshold, fail loudly and exactly if not).

**The new drift guard should combine both shapes**: `TestRankEdges`'s two-directional membership check (does every `RANK_EDGES` kind appear in the committed measurement JSON, and does the JSON carry no unexpected kind) plus `ExpectedScenarioCount`'s named-constant-with-exact-comparison discipline for the per-kind threshold (D-15's frozen N) and the priority-4-language non-zero bar.

## Shared Patterns

### Full-graph-scan-to-per-kind-map
**Source:** `internal/query/status.go`'s `nodesByKind` loop (lines ~231-240) and `internal/query/expand.go:263-283`'s `buildExpandAdjacency`
**Apply to:** `edgesByKind`'s computation inside `Engine.Status`
```go
edgesByKind := make(map[string]int64)
for it.Next() {
    e := it.Edge()
    edgesByKind[e.Kind]++
}
```

### Filter/derive-at-render-time, never inside `Status()`
**Source:** `internal/query/render_status.go`'s `sortedCounts` (lines 84-98), called from both `RenderStatusText` and `RenderStatusMarkdown`
**Apply to:** sparse-vs-dense decision for `edgesByKind` — `Engine.Status` should compute the full sparse tally unconditionally; sparse filtering (`count > 0`) and dense derivation (from `RankEdges`) both happen at the render/marshal boundary, matching how `NodesByKind`/`FilesByLanguage` are already handled. Do NOT give `Engine.Status` a `dense bool` parameter (RESEARCH Anti-Pattern, explicit).

### Derive constants, never restate them
**Source:** `internal/query/rwr_test.go:13-38` (`TestRankEdges`)
**Apply to:** every new file that needs "the 9 edge kinds" — dense render helper, dense marshal helper, the manifest schema (if it lists edge kinds anywhere), the drift-guard's expected-kinds list. All must `import query` (or `goextract`) and iterate `RankEdges`, never hard-code the 9 literal strings a second time.

### Loud, non-silent failure on a missing precondition
**Source:** `Taskfile.yml`'s `check:transcript-freeze` (`set -euo pipefail` + `::error::`-prefixed message + `exit 1` on missing env var) and `release:dry-run-signed`'s `preconditions: [{sh, msg}]` list
**Apply to:** the corpora fetch target (fail loud on a `git fetch` failure, no fallback-to-skip) and the drift guard's heavier leg (D-08 explicitly forbids silent skip, unlike `resolveWeftCorpus`/`resolveColbymchenryCorpus`'s existing `t.Skip` precedent, which this phase's new guard must NOT copy).

### `uses: <action>@<full-commit-sha> # v<semver>` pin style
**Source:** `.github/workflows/ci.yml:51-54`
**Apply to:** the new `actions/cache@55cc8345863c7cc4c66a329aec7e433d2d1c52a9 # v6.1.0` step

## No Analog Found

> **⚠ TWO OF THE THREE ENTRIES BELOW ARE WRONG.** See "Orchestrator Corrections" at the end of this
> file before acting on this table. Rows 1 and 2 report a real *string* absence but draw a false
> *capability* conclusion. Row 3 stands.

| File | Role | Data Flow | Reason |
|---|---|---|---|
| ~~Mechanical wire-oracle transcript re-freeze process~~ **RETRACTED — see C-2** | test tooling | file-I/O | No `-update`/`UPDATE_GOLDEN` flag or committed freeze-writer script exists anywhere in `test/wireoracle/*.go`. `tools/transcriptfreeze` is a different thing (an advisory anti-regeneration guard, not a freeze-writer). Nearest partial match: `testdata/golden/gocapture/main.go`'s "drive pipeline, write JSON" shape, or the older (unread this pass) `testdata/golden/capture.sh`. Planner must budget a small task to determine the actual mechanic (git-history check on prior `.golden` re-freezes, or write a one-off capture helper). |
| ~~`actions/cache` usage itself (the caching mechanic, not the pin style)~~ **RETRACTED — see C-3** | CI config | event-driven | Confirmed first-of-kind in this repo by both RESEARCH and this pass's own grep of `ci.yml`'s `uses:` lines (only `checkout`/`setup-go` present). Follow RESEARCH's synthesized pattern (from `actions/cache`'s own README), not an in-repo precedent. |
| Corpora manifest content/schema | config | batch | No committed manifest with this repo/SHA/license/locked shape exists. Closest *consumption* shape (one JSON file, several readers) is `.release-please-manifest.json`/`dist/artifacts.json`, but neither was read in full and neither is a content template — this is genuinely new. **STANDS.** |

## Metadata

**Analog search scope:** `internal/query/`, `internal/cli/`, `internal/cli/present/`, `internal/agents/`, `test/wireoracle/`, `testdata/golden/`, `Taskfile.yml`, `.github/workflows/`

---

## Orchestrator Corrections (2026-08-14, post-mapping)

Added by the plan-phase orchestrator after verifying this file's absence claims against precedent
`p92np6bzct`, which records **two prior FALSE "no analog found" claims** from this same agent. Three
of the four corrections below are absence errors of the same shape.

### C-1 — Wrong transcript path (factual)

This file says `test/wireoracle/transcripts/call-status.golden` in the file table, the excerpt
section and the Metadata scan list. **The file is at `testdata/wireoracle/transcripts/call-status.golden`.**
The `testdata/` prefix is load-bearing: `go list ./...` deliberately skips `testdata`, which is why
several wire-oracle Taskfile targets exist as explicit targets rather than relying on `./...`.

The re-freeze *target* identification is correct and remains so — it is `call-status.golden`
(the `codegraph_status` tool call, rendered via `query.RenderStatusMarkdown`), **not**
`resources-read-status.golden`, which serves a static per-tool doc resource that will not change.
That corrects `01-CONTEXT.md`, which named the wrong one.

### C-2 — RETRACTED: the re-freeze mechanic exists

The "No Analog Found" row is wrong. `test/wireoracle/cmd/wireoracle/main.go` is a **purpose-built
human-redirect entrypoint**; its package doc states that freezing is *"a deliberate, reviewable,
human-run `> file` redirect, never an automated regeneration path."*

What is genuinely absent is only the **flag** — zero hits repo-wide for `UPDATE_GOLDEN` /
`UPDATE_TRANSCRIPTS` / a `-update` flag. That absence is **deliberate and locked**: v0.3.0 Phase 1
chose "no-regenerate-flag + CI cross-change guard", and the standing control when goldens are
regenerated is "an approved reviewed diff." A plan that "closes the gap" by adding an update flag
would reverse a decision two milestones old.

**Lesson:** absence of the flag you expected is not absence of the capability. The grep found no
flag; the capability was one directory over, under a different name.

### C-3 — RETRACTED: the repo already has a CI caching precedent

The row claims `actions/cache` usage is first-of-kind, "confirmed by this pass's own grep of
`ci.yml`'s `uses:` lines." The string `actions/cache` does have zero hits repo-wide — but the
conclusion drawn from it is false.

**`namespacelabs/nscloud-cache-action@c5f8dab7560444c4bf8dbc64f1b203431873c547 # v1.6.1`** is used
in **nine** places, always `cache: go`, always paired with `actions/setup-go` carrying `cache: false`:

| File | Lines |
|---|---|
| `.github/workflows/ci.yml` | 60, 203, 232, 356, 386, 415 |
| `.github/workflows/bench.yml` | 112, 374, 473 |

This is a **locked decision**, not incidental: v1.0 Phase 10 D-06 reads *"Namespace runners
everywhere / nscloud-cache-action cache go / setup-go cache false."* The repo runs on
`namespace-profile-linux-amd64-4x8` / `-2x4` / `namespace-profile-macos-6x14-tahoe`, so D-13's stated
premises ("GitHub 10 GB per-repo ceiling, LRU eviction") describe a cache service this CI does not
use. The repo has also already ruled on cache *trust* twice in committed comments —
`release.yml:113-120` excludes the cache action from the `id-token: write` job because "a release
build should not read from a mutable cache any PR on any branch can populate" (finding S2, threat
T-01-29), and both canaries exclude it so a red run means the toolchain broke, not the cache action.

**Two independent agents (research and pattern-mapping) made the identical narrowing**, because both
were handed the same expected string. A shared premise defeats independent verification.

**The correct instrument** — written here so the next reader inherits the method, not just the answer:

```bash
# Enumerate every action actually used, then read them. Do NOT grep for the vendor you expect.
rg -n --no-ignore -o 'uses:\s*\S+' .github/workflows/ | sed 's/.*uses: *//' | sort -u
```

### C-4 — `testdata/golden/capture.sh` exists and was not read

Row 1 calls it "the older (unread this pass) `testdata/golden/capture.sh`." It exists — 9091 bytes,
executable — and is a real analog for the "drive a binary, capture output, write fixtures" shape.
It is also **milestone-relevant**: it requires the live TS CodeGraph CLI (v1.3.1+) plus `sqlite3`,
`jq`, `git`, `node`, and pins `WEFT_REPO` / `TS_REPO_URL` — i.e. it depends on exactly the upstream
this milestone is decoupling from. Worth a look before any later phase touches golden capture.

### Verified and standing

- Row 3 (corpora manifest is genuinely new) — **confirmed**, no correction.
- Zero `UPDATE_GOLDEN` / `UPDATE_TRANSCRIPTS` / `-update` hits repo-wide — **confirmed** by a search
  scoped to the whole tree, not just `test/wireoracle/`.
**Files scanned:** `internal/query/status.go`, `internal/query/render_status.go`, `internal/query/expand.go`, `internal/query/rwr.go`, `internal/query/rwr_test.go`, `internal/cli/status.go`, `internal/cli/present/status.go`, `internal/cli/affected.go`, `internal/cli/*.go` (flag grep), `internal/agents/opencode.go`, `testdata/golden/gocapture/main.go`, `testdata/golden/golden_parity_test.go`, `test/wireoracle/scenarios.go`, `test/wireoracle/oracle_test.go`, `test/wireoracle/transcripts/call-status.golden`, `Taskfile.yml` (multiple ranges), `.github/workflows/ci.yml`, `.github/workflows/linux-cross-canary.yml`
**Pattern extraction date:** 2026-08-13
