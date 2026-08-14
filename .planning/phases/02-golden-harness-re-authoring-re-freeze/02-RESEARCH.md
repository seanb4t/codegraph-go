# Phase 2: Golden Harness Re-authoring & Re-freeze - Research

**Researched:** 2026-08-14
**Domain:** Golden test-suite re-authoring, capture-path retirement, corpus-driven re-baselining (Go, `testdata/` golden harness)
**Confidence:** HIGH (every load-bearing claim verified in-tree this session; the only unknowns are provenance details documented in Open Questions)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01 (rename convention):** Use **behavioral vocabulary**. `parity_*_test.go` → `behavioral_*_test.go`; `TestGoldenParity*` → `TestCorpusBehavior*` / `TestBehavioral*`. Remove `parity` and other comparison framing (`synthetic-parity`, "vs TS", head-to-head) from identifiers, file names, fixture-directory names. Reversible — pure rename, no behavior change; re-freeze pass separate.
- **D-02 (`golden` survives):** `golden` survives only as the neutral fixture concept (`testdata/golden/`, `gocapture`) — the storage mechanism, not a comparison target. `golden_parity_test.go` → `golden_test.go`'s behavioral equivalent; `golden_test.go` keeps or re-names depending on what it asserts. Exact per-file names = Claude's discretion within the vocabulary.
- **D-03 (behavioral corpus moves):** `testdata/golden/corpus/synthetic-parity/` → in-tree `corpus/behavioral/`, four targeted cases intact. `synthetic-parity` name is dropped, not carried. `corpus/` (committed authored input) is deliberately distinct from `corpora/` (Phase-1 pinned third-party fetch authority); the two are NOT merged. Costly — moves committed source and every test that reads it; cheap to revert only before the re-freeze lands.
- **D-04 (case map is a committed data file):** `corpus/behavioral/CASES.json` loaded by tests, not a Go table in test source. One readable source of truth for the four cases; a later case author edits data, not test code.
- **D-05 (re-freeze driven by extending `gocapture`):** extend `testdata/golden/gocapture/main.go` (single-file today) so it covers every golden in scope, then re-freeze ALL in ONE reviewed diff against the locked corpora (hugo, guava, serilog, requests). gocapture is extended, not replaced. Costly — the re-baseline; old bytes gone except in history.
- **D-06 (rename and re-freeze are separate reviewed diffs, in that order):** rename first (changes no golden byte), re-freeze second (changes no identifier). One diff containing both makes regression un-attributable. Every changed line traces to one named cause.
- **D-07 (deletion scope):** Delete `testdata/golden/capture.sh`, `testdata/golden/mcp-capture.mjs`, the `weft-go` and `colbymchenry-codegraph` corpora and their captured fixtures, and every in-tree reference to them — synthetic-parity/colbymchenry fixture files, doc references in `testdata/golden/README.md`, and any test code that reads them.
- **D-08 (`tools/bench/realcorpus` stays):** `tools/bench/realcorpus`'s pinned `colbymchenry-codegraph` Entry stays unchanged — bench tooling, owned by Phase 6 (BENCH-01/02). Phase 2's sweep covers the harness, not the benchmark. Reversible — leaves an intact bench manifest Phase 6 edits deliberately.

### Claude's Discretion

- The exact new per-test/per-file names within the behavioral vocabulary (D-02 gives the word-level rule; planner picks concrete identifiers).
- The internal structure of the `gocapture` extension (which functions/extraction the new capture cases use), provided it remains the Go-side path and does not re-introduce a TS dependency.
- Whether small, clearly-cosmetic comment text inside the swept files is dropped or reworded at rename time — cosmetic comment edits ride the rename diff; anything changing an assertion or a golden byte belongs to the re-freeze diff.

### Deferred Ideas (OUT OF SCOPE)

- Reconciling `tools/bench/realcorpus` (its `colbymchenry-codegraph` pin and its broader-than-MIT/Apache licence policy) — **Phase 6 owns it**. D-08 leaves it intact.
- Re-pointing/removing `colbymchenry-codegraph` references in `bench.yml` and `tools/bench/headtohead-*.json` — BENCH surface, Phase 6.
- `tools/bench/runner/main.go:482` `pinnedAt()` HEAD-only validation defect — fixed when Phase 6 adopts Phase 1's four-part integrity pattern.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CODE-02 | Test and fixture identifiers no longer encode comparison framing — `parity_*_test.go` and `TestGoldenParity*` renamed or removed — and `go test ./...` plus `go test ./testdata/golden/...` both pass | Full identifier inventory in "Rename blast radius" (Summary) and "Architecture Patterns → Pattern 1". Rename is pure identifier/file-name move; no package path changes so `go test ./...` and `go test ./testdata/golden/...` reach the same packages and pass. |
| FIXT-04 | Delete `weft-go`/`colbymchenry-codegraph` corpora + captured fixtures; delete `capture.sh` and `mcp-capture.mjs` in favor of `testdata/golden/gocapture` | Deletion inventory in "Runtime State Inventory". Exact deletion set: `capture.sh`, `mcp-capture.mjs`, `corpus/weft-go/` (15 files), `corpus/colbymchenry-codegraph/` (14 files). In/out-of-scope reference boundary derived in "Common Pitfalls → Pitfall 2" (D-08). |
| FIXT-05 | Purpose-built behavioral corpus survives as `corpus/behavioral/` with case map intact and framing stripped — no targeted case lost | Case map transcribed verbatim from `corpus/synthetic-parity/README.md` in "Code Examples → the CASES.json contract". Tests that read it today enumerated; all four cases (a/b/c/d) mapped to the surviving named tests. |
| FIXT-06 | Every golden re-frozen from codegraph-go's own output against the locked corpora | Re-freeze mechanics in "Architecture Patterns → Capture-to-temp-then-move". gocapture's current coverage enumerated (the go-* fixture list); the locked-corpus provenance gap is the crux — NO current golden is produced from the locked set, so the re-freeze is a net-new capture against hugo/guava/serilog/requests, not a byte-diff of existing goldens. |

**Scope note from ROADMAP Notes (verbatim):** "Both `explore` and `node`, on both the CLI and MCP surfaces, are in the re-freeze." — gocapture's extension must cover both surfaces, not just `Engine.Explore`/`Engine.Node`.
</phase_requirements>

## Summary

Phase 2 re-authors the golden suite from a TS-vs-Go "*parity* oracle" into a self-standing behavioral regression suite, retires the entire TS-era capture path, and re-baselines every surviving golden from codegraph-go's own output against the Phase-1-locked corpora (hugo/go+tsjs, guava/java, serilog/csharp, psf/requests/python). The single most important finding: **not one current golden is produced from the locked corpora.** Every existing golden in `testdata/golden/corpus/{weft-go,colbymchenry-codegraph,`synthetic-parity`}/` was captured from the live TS 1.3.1 CLI (via `capture.sh`/`mcp-capture.mjs`) against the two external corpora being deleted, or is a Go-side `go-*.json` fixture (`gocapture`) still keyed to those same external corpora. FIXT-06 therefore is **not** a byte re-freeze of what exists — it is a *net-new capture* that (a) adds locked-corpus capture specs to `gocapture`, (b) runs it against the fetched locked set, and (c) ships the resulting fixtures as the suite's own goldens. The `synthetic-parity` corpus survives (moved to `corpus/behavioral/`) because its purpose-built cases encode behaviors no third-party repo reproduces (ROADMAP Notes, verbatim above).

**The behavioral assertions the current harness actually holds live almost entirely in `golden_parity_test.go`**, not `golden_test.go`. `golden_test.go` is a thin existence/volatility guard (`TestGoldenFixturesExist`, `TestGoSideFixturesRegenerated`) that survives essentially as-is. `golden_parity_test.go` contains `TestGoldenParity` (the TS-vs-Go subset oracle against `weft-go`), `TestGoldenBehavioralSyntheticParity` (the strong, byte/structural behavioral oracle for the four cases against synthetic-parity), `TestGoldenBehavioralRealCorpora` (shape-only, against weft/colbymchenry), and the CLI==MCP byte-identity harness (`TestExploreCLIMatchesMCP`, `TestNodeCLIMatchesMCP`, `TestNodeLineHintCLIMatchesMCP`). The rename (CODE-02) strips the comparison naming; the re-authoring re-points the corpus-driving tests at `corpus/behavioral/` + the locked corpora; the CLI==MCP trio is a self-consistency oracle that does NOT compare against TS and survives the rename largely intact (only its corpus case-list updates).

**Primary recommendation:** Split the phase into exactly two reviewed diffs as D-06 mandates — (1) a pure rename/delete pass that changes zero golden bytes (CODE-02 + FIXT-04 deletions + wrap of `corpus/synthetic-parity/` into `corpus/behavioral/` with `CASES.json`, re-pointing tests to the moved paths), then (2) the re-freeze pass, which first extends `gocapture` with locked-corpus specs and a single "regenerate all" entrypoint, runs `task corpora:fetch` + `task corpora:assert` + `go run ./testdata/golden/gocapture`, reviews the resulting golden-byte diff, and commits it with the single named cause FIXT-06. Do not build either pass on top of the other, or criterion 2 (attributability) is structurally broken.

**Gap to flag to the discuss-phase:** whether the re-authored *test assertions* (not just the goldens) must be re-pointed from "compare Go output to frozen fixtures" toward "assert named behavioral properties of Go's own output" is genuinely open. The old assertions were built as subset-vs-TS relationships; with the TS oracle gone those comparisons have no target. D-05/D-06 only lock the *capture* mechanism and the *diff granularity*, not the assertion philosophy. This is the largest planning decision and is documented in Open Questions Q1 with a recommendation.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Golden byte capture | Test tooling (`testdata/golden/gocapture`) | — | gocapture drives `indexer.Run` + `query.NewWithRoot(reader, sourcePath)` over a corpus and writes `{command, output}` JSON envelopes — the existing single capture authority D-05 elevates to sole source. |
| Behavioral corpus authoring | In-tree fixture (`corpus/behavioral/`) | — | Committed, hand-authored test input (D-03); the four targeted cases live in its source tree + `CASES.json` case map (D-04). |
| Locked-corpus source resolution | `internal/corpora` (`Entry.Dir(root)`, `CorpusRoot()`) | `task corpora:fetch` / `corpora:assert` | gocapture's extensions must resolve the four locked checkout dirs from the manifest/root — the D-12 seam, never a hardcoded path or a re-stated SHA. |
| Byte-identity freeze enforcement | `test/wireoracle/oracle_test.go` (`TestFrozenTranscriptsMatch`) | — | The repository's standing byte-identity oracle pattern; the re-freeze must satisfy a byte-identity test over the goldens, and the rename must not disturb a single byte. |
| CLI==MCP surface consistency | Self-referential test harness (CLI vs in-process MCP server) | — | The `Test*CLIMatchesMCP` trio drives the same on-disk index through both surfaces and asserts byte equality — a self-consistency oracle, not a comparison target. |

## Standard Stack

> This phase introduces **no new third-party dependencies.** It re-consumes existing in-tree machinery and the Phase-1-pinned corpora. There is no `go get`/install step to version-verify; the only "install" is `task corpora:fetch` (network + git), not a package.

### Core (existing, re-consumed — nothing new)
| Component | Purpose | Why it's the standard |
|---------|---------|----------------------|
| `testdata/golden/gocapture/main.go` | The Go-side capture path D-05 extends to sole authority [VERIFIED: testdata/golden/gocapture/main.go:1-52] | Already re-indexes a corpus into a throwaway temp Pebble store and writes `{command, output}` goldens; adding `corpusSpec`s for the locked corpora + `corpus/behavioral/` is additive, not a rewrite. |
| `internal/corpora` (`Entry.Dir`, `CorpusRoot`) | Resolves the four locked-checkout destination dirs from the manifest [VERIFIED: internal/corpora/manifest.go:178-212] | Dir = `<root>/<slug>-<short>@<sha>` with root from `CODEGRAPH_CORPUS_DIR` then `${XDG_CACHE_HOME:-$HOME/.cache}/codegraph-corpora` — the D-12 content-addressed seam gocapture's specs must read, never hardcode. |
| `task corpora:fetch` + `corpora:assert` | Idempotent whole-set fetch + four-part integrity check [VERIFIED: Taskfile.yml:3502-3560] | The repo's sole source of the locked pinned trees; the re-freeze's precondition (fetch, then assert 4/4). |
| `internal/indexer.Run` + `query.NewWithRoot` / `query.OpenAt` | The production indexing + query pipeline gocapture drives [VERIFIED: testdata/golden/gocapture/main.go:131-147] | Matches how production indexes; no bespoke capture path. |
| `internalmcp.BuildServer` + `go-sdk` in-process client | MCP-surface capture for the "both CLI and MCP surfaces in the re-freeze" clause [VERIFIED: testdata/golden/golden_parity_test.go:1446-1528] | The `newGoldenSession`/`callExploreViaMCP`/`callNodeViaMCP` helpers already prove CLI==MCP byte identity; re-freeze reuses this to capture the MCP surface. |

### Supporting
| Library | Purpose | When to Use |
|---------|---------|-------------|
| `github.com/seanb4t/codegraph-go/internal/query` result types | `CallersResult`/`CalleesResult`/`ImpactResult` decode goldens field-for-field | Only if the re-authored status/query/callers/callers-shaped assertions keep fixture decoding (currently `TestGoldenParity` does — `loadGoldenFixture[query.CallersResult]`). |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Extending `gocapture` (D-05) | A fresh capture binary under `tools/` | D-05 is explicit: extend gocapture, don't replace; it is the path FIXT-04 elevates to authority. A new binary fragments the "one capture path" guarantee. |
| Locked-corpus specs via `internal/corpora` | Re-stating SHAs in `gocapture` as strings | Re-stating any SHA violates the manifest-as-sole-pin-authority rule (D-09, Phase 1) — two copies of a SHA create a third invariant to police. |
| Assertion re-pointing to "compare against frozen Go goldens" | "Assert named behavioral properties of live output" | See Open Questions Q1 — this is the one genuine design fork; the locked-corpus goldens support both, but only the property-assertion style truly decouples the suite from its own captured bytes. |

**Installation:**
```bash
# No package install. Precondition for the re-freeze only:
task corpora:fetch
task corpora:assert
# The re-freeze capture itself (after gocapture is extended):
go run ./testdata/golden/gocapture
```

## Package Legitimacy Audit

> **Not applicable / no new packages installed this phase.** Phase 2 installs no external Go package, Python/Rust package, or npm module. Its only external inputs are the four locked corpora, which are fetched at pinned SHAs via `task corpora:fetch` (Phase 1, FIXT-02) — not installed as dependencies — and whose MIT/Apache-2.0 licences and pinned SHAs were live-verified in Phase 1 (recorded in `corpora/manifest.json` and `docs/CORPUS-MEASUREMENT.md`). `gocapture` already depends only on in-repo packages (`internal/graphstore`, `internal/indexer`, `internal/query`) [VERIFIED: testdata/golden/gocapture/main.go:40-51].

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| *(none — no new external package)* | — | — | — | — | — | — |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.
*The only "package-like" input, the four locked corpora (hugo/guava/serilog/requests), are pinned-fetch inputs already audited in Phase 1; FIXT-02 criteria (fetch at pinned SHA, CI cache, no vendoring) are already met and untouched by this phase.*

## Architecture Patterns

### System Architecture Diagram

```
                        ┌─────────────────────────────────────────────────────────────┐
                        │                 INVARIANT: one-diff-one-cause (D-06)       │
                        │   rename pass → changes NO golden byte                     │
                        │   re-freeze pass → changes NO identifier                   │
                        └─────────────────────────────────────────────────────────────┘

 REPO-PRESENT INPUTS                                  EXTERNAL INPUTS
 ┌─────────────────────────────┐                     ┌──────────────────────────────┐
 │ testdata/golden/gocapture    │                     │ corpora/manifest.json (pin)  │
 │ (extended, D-05)             │                     └──────────────┬───────────────┘
 │  . corpusSpec per locked     │                                    │ repo@sha
 │    corpus (hugo/guava/       │                     ┌──────────────▼───────────────┐
 │    serilog/requests)         │                     │ task corpora:fetch/assert   │
 │  . corpusSpec corpus/        │                     │  → ${CORPUS_DIR}/<slug>@<sha>│ (D-12)
 │    behavioral/ (4 cases)     │                     └──────────────┬───────────────┘
 └──────────────┬──────────────┘                                    │ .git-holding pinned tree
                │ indexer.Run + query.NewWithRoot
                ▼                                                   │
   ┌───────────────────────────────┐                                │
   │  throwaway temp Pebble store  │◄───────────────────────────────┘
   └──────────────┬────────────────┘
                  │ Engine.Explore / Engine.Node (CLI surface)
                  │ BuildServer + in-process client (MCP surface)
                  ▼
   ┌────────────────────────────────────────────────┐
   │  gocapture → corpus/behavioral/*.json  +       │
   │            locked-corpus <lang>/*.json goldens │  ← the re-freeze diff (FIXT-06)
   └────────────────────────────────────────────────┘
                  │ consumed by tests
                  ▼
   ┌───────────────────────────────────────────────────────┐
   │ testdata/golden/*_test.go (renamed ← parity naming)   │
   │  - behavioral corpus tests  (CASES.json task map, D-04)│
   │  - locked-corpus tests       (hugo/guava/serilog/...)   │
   │  - CLI==MCP byte-identity trio (self-consistency)      │
   └───────────────────────────────────────────────────────┘
```

### Recommended Project Structure
```
corpus/                                  # NEW top-level (committed authored test input, D-03)
└── behavioral/                          # ← was testdata/golden/corpus/synthetic-parity/
    ├── CASES.json                       # D-04 case map — four named cases (a/b/c/d)
    ├── src/
    │   ├── go.mod
    │   ├── accounts/{validate.go,manager.go}
    │   ├── orders/validate.go
    │   ├── recovery/{recovery.go,recovery_test.go}
    │   └── ledger/ledger.go
    ├── go-explore-multi.json            # Go-captured goldens (move + keep)
    └── go-node-multi.json

testdata/golden/
├── gocapture/main.go                    # EXTENDED (D-05): locked-corpus + behavioral specs
├── golden_test.go                       # keeps — fixture-existence/volatility guard
├── behavioral_test.go                   # ← was golden_parity_test.go (strongest behavioral oracle)
├── behavioral_java_test.go              # ← was parity_java_test.go
├── behavioral_tsjs_test.go              # ← was parity_tsjs_test.go
├── behavioral_csharp_test.go            # ← was parity_csharp_test.go
├── behavioral_python_test.go            # ← was parity_python_test.go
├── ts-schema.sql / ts-schema.dump.sql   # KEEP — legacy-format migration ground truth (Phase 7), not framing
├── ts-version.txt                       # provenance for the migration schema — keep or move (discretion)
└── README.md                            # re-authored: neutral fixture authority, framing stripped
```
*(The full file-tree of the current `testdata/golden/` is inventoried under "Runtime State Inventory".)*

### Pattern 1: The Rename = a pure identifier/file-name move (CODE-02, D-01/D-02)
**What:** Strip comparison framing from every identifier, file name, and fixture-directory name inside the golden harness, changing no golden byte and no assertion.
**When to use:** Land as the first of D-06's two diffs.
**Example (recommended concrete mapping — planner discretion, entries verified this session):**
```go
// Source: verified by Read of testdata/golden/*.go this session; names are the CURRENT identifiers.
parity_java_test.go            → behavioral_java_test.go     ; TestGoldenParity_Java    → TestCorpusBehavior_Java
parity_tsjs_test.go            → behavioral_tsjs_test.go     ; TestGoldenParity_TSJS    → TestCorpusBehavior_TSJS
parity_csharp_test.go          → behavioral_csharp_test.go   ; TestGoldenParity_CSharp  → TestCorpusBehavior_CSharp
parity_python_test.go          → behavioral_python_test.go   ; TestGoldenParity_Python  → TestCorpusBehavior_Python
golden_parity_test.go          → behavioral_test.go
  TestGoldenParity             → (re-authored, see Q1)
  TestGoldenBehavioralSyntheticParity → TestCorpusBehaviorSynthetic
  TestGoldenBehavioralRealCorpora     → TestCorpusBehaviorLockedCorpora
  TestExploreCLIMatchesMCP / TestNodeCLIMatchesMCP / TestNodeLineHintCLIMatchesMCP → keep (self-consistency, no TS target)
  resolveWeftCorpus / buildWeftEngine / pinnedWeftCommit / resolveWeftGoCorpusLoose /
  resolveColbymchenryCorpus / defaultWeftRepo → removed or re-pointed to corpus/behavioral + locked corpora
golden_test.go (TestGoldenFixturesExist, TestGoSideFixturesRegenerated) → keep or re-name (D-02 discretion)
```

### Pattern 2: Capture-to-temp-then-move for the re-freeze (FIXT-06 — carries over from 01-03)
**What:** Never redirect capture output straight onto the committed golden path. Write to a temp path, assert non-empty and marker-bearing, then move into place.
**When to use:** Every golden a re-freeze writes.
**Why (from 01-03-SUMMARY, `gocapture`'s sibling):** a capture failure that truncates before producing bytes leaves a zero-byte oracle that the byte-identity test then refuses for a second, misleading reason.
**Mechanism for "one reviewed re-freeze diff":** gocapture already writes to well-defined `{command, output}` JSON files under `corpusDir/<name>/` [VERIFIED: testdata/golden/gocapture/main.go:148-206]. Extend it to (a) resolve each locked corpus's source from `internal/corpora` `Entry.Dir`, (b) still skip-warn (never hard-fail) a corpus whose source is absent — matching the existing `weftGoSpec`/`syntheticParitySpec` degrade-gracefully contract [VERIFIED: testdata/golden/gocapture/main.go:224-290], and (c) capture both Engine (CLI) and BuildServer (MCP) surfaces for the "explore + node, CLI + MCP" clause. Then the reviewed re-freeze is: `task corpora:fetch` → `task corpora:assert` → `go run ./testdata/golden/gocapture` → `git diff` and review. There is **no existing Taskfile target that runs gocapture** — the planner should add one (e.g. `golden:regen`) so "regenerate all goldens" is one named, reviewable command.

### Anti-Patterns to Avoid
- **Splitting rename and re-freeze into one commit (or landing them together in review):** makes any regression un-attributable; database .golden test fail = can't tell if it's the move or the re-baseline. D-06 is explicit; criterion 2 depends on it.
- **Re-stating a locked-corpus SHA or path inside gocapture:** the manifest is the sole pin authority (Phase 1 D-09) — resolve via `internal/corpora`, never hardcode "hugo@0805c…". A hardcoded SHA that drifts from the manifest is exactly the third-invariant the phase's predecessors forbid.
- **Deleting `ts-schema.sql`/`ts-schema.dump.sql`/`ts-version.txt` as "TS-era":** these are the DDL ground truth the Phase-7 migration reader imports legacy `.codegraph/` indexes with (CODE-03 frames migration as legacy-index import, explicitly preserved). They are not comparison framing. Deleting them breaks Phase 7 with no replacement authority.
- **Renaming CI step names / Taskfile `test:golden` desc now:** CI step names and workflow job names are PROC-03 / Phase 4; the Taskfile `test:golden` desc "Golden parity suite" is a comment → CODE-01 / Phase 4. Phase 2 scope is test+fixture identifiers (CODE-02) only. The `run: task test:golden` body and the {command} stay valid unchanged.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Resolving a locked corpus's on-disk location | A hardcoded path or env map in gocapture | `internal/corpora` `Entry.Dir(root)` / `CorpusRoot()` | The pinned SHA + collision-free dir is already derived [VERIFIED: internal/corpora/manifest.go:178-212]; hand-rolling a second resolver duplicates D-09's pin logic. |
| N-gram/reading a query multi-word tokenizer | — | already built | `internal/query`'s multi-word tokenization (EXPL-01/02) already exists; the re-freeze only captures its output, it does not rebuild it. |
| MCP-surface capture | A new MCP client or `mcp-capture.mjs` re-implementation | The in-process `BuildServer` + `go-sdk` client (`newGoldenSession`) | Already proven CLI==MCP byte-identity in-tree [VERIFIED: golden_parity_test.go:1446-1528,1541-1632]; re-capturing via a live TS server would re-introduce the exact TS dependency FIXT-04 retires. |

**Key insight:** This phase's trap is *attribution*, not *capability*. All four capabilities it needs (indexing, querying, capture, CLI==MCP equivalence) already exist in-tree. The risk is doing the mechanical re-authorization in a way that makes the re-baseline un-reviewable or re-introduces a deleted dependency. There is nothing genuinely new to build except the gocapture corpus-spec extension.

## Runtime State Inventory

> Rename/refactor/migration phase — the five categories below are answered explicitly.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| **Stored data (golden bytes on disk)** | Every JSON fixture under `testdata/golden/corpus/`: `weft-go/` (15 files: 11 TS-captured status/query/callers/callees/impact/explore/node/explore-multi/node-multi/explore-mcp/node-mcp + 4 gocapture `go-*.json`) [VERIFIED: find output this session]; `colbymchenry-codegraph/` (14 files, same 11 TS + 3–4 go-*; gocapture multi-writes are best-effort, a failed symbol like `resolve` (AD-02) omits that artifact) [VERIFIED: gocapture/main.go:182-206]; `synthetic-parity/` (4 TS-captured explore-multi/explore-mcp/node-multi/node-mcp + 2 gocapture go-explore-multi/go-node-multi + `src/`) | **Delete:** `weft-go/` + `colbymchenry-codegraph/` entirely and the 4 synthetic-parity TS-captured fixtures (FIXT-04/D-07). **Move** `src/` + the 2 `go-*.json` to `corpus/behavioral/` (FIXT-05). No data migration of records — these are captured files, replaced wholesale. |
| **Stored data (the re-freeze target)** | New goldens for the locked corpora (hugo/guava/serilog/requests) do NOT exist today — none of the current goldens derive from the locked set. | Net-new capture via extended gocapture (FIXT-06). Also re-capture CLI + MCP surfaces for explore + node per ROADMAP Notes. |
| **Live service config** | None found — no external service (n8n / Datadog / etc.) references this golden suite. The corpora live at `${CORPUS_ROOT}` on local/CI disks only, not in any service config. | None. Explicitly: "None — no live-service configuration references the golden corpus or the deleted corpora." |
| **OS-registered state** | None — no Task Scheduler/pm2/launchd registrations reference the deleted corpora or capture scripts. | None — "None — verified by absence of OS-level registration tasks in this repo." |
| **Secrets and env vars** | The golden harness reads env vars: `CODEGRAPH_WEFT_CORPUS`, `WEFT_REPO`, `CODEGRAPH_{JAVA,PYTHON,TSJS,CSHARP}_CORPUS`, `CODEGRAPH_CORPUS_DIR` [VERIFIED: golden_parity_test.go:120,243,38-66; gocapture/main.go:228,243]. Deleting weft/colbymchenry retires `CODEGRAPH_WEFT_CORPUS`/`WEFT_REPO`; the per-language `CODEGRAPH_{LANG}_CORPUS` vars (used by the 4 `parity_*_test.go`) may be retired if those tests re-point to the locked corpora (discretion — see Q2). `CODEGRAPH_CORPUS_DIR` (Phase 1, D-12) stays live | Retire deleted-corpora env vars; keep `CODEGRAPH_CORPUS_DIR`. `.env`-in-git: none reference these. |
| **Build artifacts / installed packages** | The deleted corpora were never installed as packages; `corpus/synthetic-parity/src/.codegraph/` is gitignored local index data [VERIFIED: .gitignore:38] and rebuilt per-machine — no action. `gocapture` builds from source (`go run`), no retained binary. TS CLI (`codegraph` v1.3.1) is no longer on PATH (the on-PATH `codegraph` is the Go v0.9.0 build) and is not required by the re-freeze. | None — nothing to uninstall; `.codegraph/` under `src/` should be excluded when copying into `corpus/behavioral/` (the existing `copyDir` already skips `.codegraph` [VERIFIED: golden_parity_test.go:296-319]). |

**The canonical question — answered:** *After every file in the repo is updated, what runtime systems still hold the old string cached/stored/registered?* → The deleted TS-era golden bytes exist only as git history / the deleted files (gone from the tree by FIXT-04); `synthetic-parity`'s name persists in the committed `src/` + tests until the rename moves them to `corpus/behavioral/`; no OS/service/secret holds the framing. The only "still old name in a live system" risk is a local developer's `../weft` sibling checkout and `$WEFT_REPO` — not part of this repo's tree and outside the sweep's authority.

## Common Pitfalls

### Pitfall 1: The rename is not really a "pure rename" because tests couple to the *corpus directory names* `weft-go`/`synthetic-parity`
**What goes wrong:** Renaming `synthetic-parity`→`corpus/behavioral` silently breaks every test that does `filepath.Join("corpus", "synthetic-parity", …)` (`loadGoldenFixtureIn`, `syntheticParitySrc`, `gocapture`'s `syntheticParitySpec`, `TestGoSideFixturesRegenerated`) if the move isn't applied atomically with the identifier pass.
**Why it happens:** FIXT-05's "survive the rename with framing stripped" means both the *directory move* AND *every reader* change together; CODE-02 criterion 1 requires `go test ./testdata/golden/...` green at the end of the rename diff.
**How to avoid:** Do the `git mv` of the corpus + all reader/test paths in the SAME rename reviewed-diff, and make the rename diff's verification explicitly include `go test ./testdata/golden/...`. The rename changes source *paths* even though it changes no golden *bytes* — that is fine and expected that the rename diff also touches test source that references the moved paths; what it must not touch is the `.json` golden byte content.
**Warning signs:** A golden `.json` file shows up in the rename diff; `TestGoSideFixturesRegenerated` or any `loadGoldenFixtureIn` reads a stale path after the move.

### Pitfall 2: Sweeping the benchmark references by mistake (D-08 boundary)
**What goes wrong:** `tools/bench/realcorpus/manifest.go`, `bench.yml`, `tools/bench/headtohead-*.json`, `docs/BENCHMARKS.md` and the repo-root `README.md` benchmark table all carry `colbymchenry-codegraph`/`weft-go`. A wide grep of "colbymchenry" will sweep them.
**Why it happens:** the same string names the deleted golden corpus and the retained bench corpus.
**How to avoid — precise boundary [VERIFIED this session]:**
- **IN scope (deletions must leave nothing referencing them):** `testdata/golden/corpus/{weft-go,colbymchenry-codegraph}/`, `capture.sh`, `mcp-capture.mjs`, `testdata/golden/README.md` wording, and any test code in `testdata/golden/` that reads them.
- **IN scope (renames, CODE-02):** the `parity_*_test.go` files and `TestGoldenParity*` in `testdata/golden/`.
- **IN scope where they *are* the harness:** gocapture's weft/colbymchenry specs (deleted — the corpora are gone; those specs get replaced by locked-corpus specs).
- **OUT of scope (D-08 / Phase 6 / Phase 4, do not touch):** `tools/bench/realcorpus/manifest.go`, `manifest_test.go`, `bench.yml`, `tools/bench/headtohead-*.json`, `docs/BENCHMARKS.md`, repo-root `README.md` benchmark table (line 162) and origin attribution (line 196), `NOTICE` (colbymchenry MIT attribution MUST remain — licence, not framing), `internal/query/*.go` and `internal/migrate/*.go` doc comments (CODE-01, Phase 4; see Pitfall 3), `docs/FLAG-PARITY.md` (deleted by Phase 4's flag-parity retirement).
**Warning signs:** the rename/delete diff touches anything under `tools/bench/`, `bench.yml`, `NOTICE`, `internal/`, `docs/FLAG-PARITY.md`, `CHANGELOG.md`, or `.planning/` archives.

### Pitfall 3: Dangling doc-comment references from the corpus deletion
**What goes wrong:** `internal/query/{status,traverse,render_markdown}.go` and `internal/migrate/*_test.go` reference `testdata/golden/corpus/weft-go/*.json` in doc comments [VERIFIED: internal/query/status.go:20,33; traverse.go:419; render_markdown.go:13]. Deleting `weft-go/` orphans these comments.
**Why it happens:** the sweep discipline (STATE.md) is explicit that Phase 4's CODE-01 owns all `internal/` doc comments; Phase 2's scope is the harness.
**How to avoid:** Leave `internal/` comments to Phase 4 (record the gap so it isn't lost), OR if the re-freeze naturally rewrites them, keep that in the re-freeze diff (it's doc refresh tied to the re-baseline, not a rename). Do NOT let them ride the rename diff (that would mix scope). The `internal/migrate` "weft" refs are the bd/dependency tool's own term in the legacy schema content — they are migration-schema ground truth, not framing; leave entirely.
**Warning signs:** a rename diff that edits files under `internal/`.

## Code Examples

Verified patterns from the in-tree sources (this session):

### The `corpusSpec` extension seam (gocapture, D-05)
```go
// Source: testdata/golden/gocapture/main.go:69-103 (current corpusSpec shape, verified by Read)
type corpusSpec struct {
	name string
	resolveSource func() (string, string) // ("", reason) to skip-warn, never hard-fail
	baselineSymbol string; baselineSymbolFile string; baselineQuery string
	multiSymbol string; multiQuery string
}
specs := []corpusSpec{
	// ADD: one per locked corpus — hugo (go+tsjs), guava (java),
	// serilog (csharp), requests (python) — resolveSource = internal/corpora Entry.Dir(root)
	// ADD/keep: corpus/behavioral/ (was synthetic-parity), which is the D-05 "survives" corpus
}
```
The existing `syntheticParitySpec` already demonstrates the committed-in-repo resolution pattern (`filepath.Join(goldenDir,"corpus","synthetic-parity","src")`) that becomes `corpus/behavioral/src` [VERIFIED: gocapture/main.go:276-290].

### The re-freeze's temp-then-move discipline (02-03's carryover)
```sh
# Source: .planning/phases/01-corpus-selection-by-measurement/01-03-SUMMARY.md (Pattern: capture-to-temp-then-move)
# Write to temp, assert non-empty + marker-bearing, THEN move onto the golden path.
# Applies to every golden the re-freeze writes; never redirect capture straight onto the committed path.
```

### The CASES.json contract (D-04) — the four cases transcribed verbatim from the current case map
```jsonc
// Source: testdata/golden/corpus/synthetic-parity/README.md (D-03 case map, verbatim this session)
{
  "cases": [
    { "id": "a", "name": "overloaded-same-named-symbols",
      "what": "node <name> multi-def definition enumeration (NODE-01); two top-level defs sharing one symbol name",
      "symbol": "Validate", "files": ["accounts/validate.go", "orders/validate.go"] },
    { "id": "b", "name": "multi-word-query",
      "what": "explore's CamelCase multi-word <query...> tokenization (EXPL-01)",
      "symbol": "UserAccountManager", "file": "accounts/manager.go", "query": "user account" },
    { "id": "c", "name": "test-heavy-weakly-connected-cluster",
      "what": "file-relevance gate (EXPL-03): a Test*-named function with zero inbound graph edges beside a structurally-connected non-test symbol",
      "zero-edge": "TestAccountRecovery", "connected": ["recoverAccount", "validateRecovery"], "files": ["recovery/recovery_test.go", "recovery/recovery.go"] },
    { "id": "d", "name": "structural-beats-lexical",
      "what": "RWR ranking (EXPL-02): name-loose-but-isolated vs name-miss-but-heavily-connected",
      "isolated": "AccountBalanceHelper", "connected": "ReconcileLedger", "file": "ledger/ledger.go" }
  ]
}
```
*(The exact JSON schema shape is planner discretion within D-04 — it must name the four cases, their symbols, and their files so a test can load them; the values above are the verified case map.)*

### CLI==MCP byte-identity reuse for the MCP-surface re-freeze
```go
// Source: testdata/golden/golden_parity_test.go:1545-1588 (TestExploreCLIMatchesMCP), verified by Read.
// The re-freeze drives the MCP surface through the SAME in-process BuildServer + go-sdk client
// (newGoldenSession/callExploreViaMCP) these tests already prove; extend the case list to the
// locked corpora + corpus/behavioral/ rather than re-introducing a live TS server.
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Golden fixtures captured from the **live TS 1.3.1 CLI** via `capture.sh`/`mcp-capture.mjs` against weft/colbymchenry/synthetic | Goldens captured **only from codegraph-go's own output** via `gocapture` against the locked corpora + `corpus/behavioral/` | This phase (FIXT-04/FIXT-06) | The comparison-vs-TS target is gone entirely; goldens become the Go suite's own regression baseline; capture needs no `node`/`sqlite3`/`jq` or a TS binary. |
| Golden corpus naming carries "parity"/"synthetic-parity"/"vs TS" framing | Behavioral vocabulary names assert what a test proves (`behavioral_*`, `TestCorpusBehavior*`) | This phase (CODE-02/D-01) | Tests read as codegraph-go's own regression suite, not a comparison to the abandoned origin project. |
| The wire-oracle re-freeze (Phase 1, 01-03) was a single human-redirect `> file` of a `call-status.golden` [VERIFIED: 01-03-SUMMARY.md; wireoracle/main.go:1-18 deliberately has NO `-update` flag] | gocapture is extended to a single "regenerate all goldens" authority, still reviewed as one diff | This phase (D-05/D-06) | The capture mechanism stays human-in-the-loop and reviewable; no automated in-place overwrite path that could silently truncate. |

**Deprecated/outdated:**
- `codegraph status` parity subtest and TS-fixture key-loop assertions: the `golden_parity_test.go` status subtest keys against the frozen `weft-go/status.json` — that oracle is deleted; the assertions re-point to our own `--json` output (already the pattern for `edgesByKind`/`filesByLanguage`).
- `ts-version.txt`/`ts-schema.*` comparison framing: keep the schema as migration ground truth, but the README's "ground truth from live TypeScript" framing is re-authored to neutral "schema for legacy-index import".

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The four locked corpora (hugo/guava/serilog/requests) each index and produce non-trivial `explore`/`node` output that yields stable goldens under gocapture's `{command, output}` envelope. | Re-freeze | If a locked corpus is too large/slow to index in a test-time gocapture run (hugo/guava are large), the re-freeze may need corpus-scoped symbol selection rather than the whole-tree multi-style capture. Phase 1 already indexes all four in `corpora:drift`, so it is proven indexable; symbol-level output stability is the only residual risk. |
| A2 | Hydrating the same JSON envelope shape (`goldenCapture{Command,Output}`) for all goldens is the retained format. | Pattern 1/Re-freeze | If the planner prefers a different golden format, gocapture's `writeCapture` [VERIFIED: gocapture/main.go:209-219] and the `goldenCapture`/`loadGoldenOutput` helpers change together — bounded, but touches many tests. |
| A3 | gocapture's current "best-effort multi-symbol write (a failed symbol warns, doesn't fatal)" behavior is kept for the locked corpora. | Re-freeze | If a locked-corpus multi-symbol (e.g. a TS/JS symbol in hugo) fails to resolve under Go extraction, hugging the current warn-not-fatal contract yields a missing golden that FIXT-06 could read as satisfied but bare. Recommend the re-freeze assert every targeted lock-corpus golden is non-empty before merge. |

## Open Questions

1. **Assertion philosophy: does the re-authored suite compare live output to frozen Go goldens, or assert named behavioral properties of live output?**
   - What we know: today's `TestGoldenParity` is a subset-vs-TS comparison; `TestGoldenBehavioralSyntheticParity` is a strong byte/structural oracle against synthetic fixtures; the CLI==MCP trio is self-consistent. D-05/D-06 lock the *capture* mechanism and diff granularity only.
   - What's unclear: after the TS oracle and weft/colbymchenry corpora are deleted, what do the per-corpus assertions *assert against*? Options: (a) compare live Engine output to gocapture-frozen goldens (keeps the "golden" byte-test role, but couples assertions to captured bytes that must be re-frozen on any legit output change — a maintenance tax), or (b) assert named behavioral properties directly (the four cases + locked-corpus shape/count/edge-coverage invariants) with goldens as illustrative/regression snapshots. FIXT-03 (Phase 3) proves non-vacuity under whichever is chosen.
   - Recommendation: prefer (b) property-assertion style for the corpus tests (decouples the suite from re-freezing itself on normal evolution) with the gocapture goldens retained as regression snapshots; flag to discuss-phase for a user ruling since it shapes the whole re-authored test file.

2. **Do the four `parity_{java,tsjs,csharp,python}_test.go` (→`behavioral_*`) tests re-point to the locked corpora, or stay user-configured?**
   - What we know: they currently self-skip on `CODEGRAPH_{JAVA,TSJS,CSHARP,PYTHON}_CORPUS` env or tone-correct sibling checkouts, and validate extraction SHAPE (kind coverage + determinism), not goldens [VERIFIED: parity_java_test.go:33-76,81-167; parity_python_test.go:42-76]. The locked set fully covers all four languages (guava/serilog/requests/hugo-tsjs).
   - What's unclear: whether re-pointing them to resolve the locked corpus for each language (removing the env-var skip dependency and tying them to the locked set) is in scope vs. leaving them user-configured (a Phase 3/FIXT-03 concern about CI self-skips).
   - Recommendation: re-point them to the locked corpora — it is the idiomatic realization of "goldens come from the locked corpora", eliminates the dead env-var convention, and is pure test-repointing (no new surface). The exact resolver (a new shared helper reading `internal/corpora`) is a small, in-scope addition.

3. **Where does `corpus/behavioral/` physically live — repo root (as D-03/D-04's `corpus/` implies) or under `testdata/golden/`?**
   - What we know: D-03/D-04 name `corpus/behavioral/` and `corpus/behavioral/CASES.json`, and CONTEXT says `corpus/` is deliberately distinct from `corpora/` — implying a repo-root `corpus/` directory. But the corpus is `testdata`-shaped (its `.json` fixtures are test input, `go test ./...` should not discover it as a package).
   - What's unclear: moving it to repo-root `corpus/` means it's no longer under `testdata/`, so nothing special needed there (it has no `.go` files, so it's not a package anyway); but the ROADMAP criterion 1 scope mentions "under `testdata/golden/`" for the *rename*, and FIXT-05 speaks of "as `corpus/behavioral/`" without a root prefix.
   - Recommendation: honor D-03/D-04 literally — a committed top-level `corpus/behavioral/` (+ `CASES.json`), explicitly not merged with `corpora/`. Confirm with discuss-phase before it lands since it is a new top-level directory with ripple effects on every path reference.

4. **Are `ts-version.txt` (and the `ts-schema.*` "live TS" wording) kept as migration-schema provenance or retired?**
   - What we know: `ts-schema.sql`/`ts-schema.dump.sql` are DDL ground truth for Phase 7's migrate reader (per CODE-03, migration stays a legacy-index import) — they must stay. `ts-version.txt` records the TS version/date that captured that schema.
   - What's unclear: whether `ts-version.txt` reads as "comparison framing" or legitimate provenance for a retained schema.
   - Recommendation: keep `ts-version.txt` as schema provenance but re-word `testdata/golden/README.md` to describe the schema as "legacy-index import spec, captured 2026-07-15" rather than "ground truth from the live TS oracle, compared against". Planner discretion on whether it moves beside the schema.

## Environment Availability

> This phase has external dependencies: the four locked corpora (fetch-time network/git) and the Go toolchain. The deleted TS-era tools (`node`, `sqlite3`, `jq`, TS `codegraph` CLI) are no longer required after FIXT-04.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | build/test, gocapture run | ✓ | go1.26.5 (darwin/arm64) [probed] | — |
| `task` (Taskfile) | `task corpora:fetch` / `corpora:assert` / `test:golden` | ✓ | 3.52.0 [probed] | — |
| `git` | `corpora:fetch` shallow fetch, re-freeze review | ✓ | 2.55.0 [probed] | — |
| Locked corpora (hugo/guava/serilog/requests, pinned SHAs) | re-freeze capture (gocapture) | ✗ currently **not fetched locally** [probed: no `codegraph-corpora` dir] | — | `task corpora:fetch` first (network + git); Phase-1 `corpora.yml` wiring exists |
| Network + GitHub access | `corpora:fetch` | likely ✓ | — | re-freeze cannot run offline; the *tests* against `corpus/behavioral/` run offline |
| TS `codegraph` CLI / `node` / `sqlite3` / `jq` | old capture.sh/mcp-capture.mjs | n/a after FIXT-04 | (node/sqlite3/jq probed present; the on-PATH `codegraph` is the **Go** v0.9.0 build, not TS) | retired — no longer needed |

**Missing dependencies with no fallback:** none — the re-freeze's only true prerequisite (the locked corpora) has the established `corpora:fetch` path, and the local machine can fetch it on demand.
**Missing dependencies with fallback:** the locked corpora when not yet fetched — fallback is `task corpora:fetch` + `corpora:assert`. The behavioral-corpus tests (`corpus/behavioral/` + `gocapture`) run offline and need no external dependency.

## Validation Architecture

> `.planning/config.json` treated as enabling Nyquist validation (key not observed explicitly false this session — verify at plan time; if absent, enabled).

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` (stdlib), plus `go-sdk` MCP in-process client for the CLI==MCP trio |
| Config file | none — `testdata/golden/*_test.go` are a normal `testing` package reached only via `go test ./testdata/golden/...` (GOLDEN-01) |
| Quick run command | `go test ./testdata/golden/...` |
| Full suite command | `task test:golden` (CI runs this as its own step) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CODE-02 | no `parity_*_test.go` / `TestGoldenParity*` survive; both `go test` invocations pass | (rena-re) — verification via `rg`/`go test` | `gsd_run`/`rg "parity" testdata/golden/` + `go test ./... && go test ./testdata/golden/...` | rename diff — Wave 0 none |
| FIXT-04 | `capture.sh`/`mcp-capture.mjs`/weft-colbymchenry corpus absent, nothing references them | static (rg) | `rg -i "weft|colbymchenry|mcp-capture|capture.sh" testdata/golden/` → expected empty (harness scope) | n/a |
| FIXT-05 | four behavioral cases exercised by named tests, loaded from `CASES.json` | unit + behavioral (Go) | `go test ./testdata/golden/... -run TestCorpusBehavior` | ❌ Wave 0 — new `CASES.json` + re-authored `behavioral_test.go` |
| FIXT-06 | every golden byte produced by gocapture from Go output against locked corpora | e2e re-baseline (human-reviewed capture) | capture-to-temp-then-move + byte-identity test | ❌ Wave 0 — re-freeze diff + gocapture locked-corpus specs |

### Sampling Rate
- **Per task commit:** `go test ./testdata/golden/...` (fast; behavioral corpus is in-repo; locked-corpus tests skip-warn when corpora absent, matching FIXT-03's later concern).
- **Per wave merge:** `go test ./testdata/golden/...` over the whole re-authored suite.
- **Phase gate:** `go test ./...` AND `go test ./testdata/golden/...` green (CODE-02 criterion 1) before `/gsd-verify-work`.

### Wave 0 Gaps
- [ ] `testdata/golden/gocapture/main.go` — locked-corpus + `corpus/behavioral/` capture specs (no new test infra; gocapture is a `main` program driven manually + guarded by existing `TestGoSideFixturesRegenerated`).
- [ ] `corpus/behavioral/CASES.json` — the D-04 case map (a test data file, not a Go test; consumed by the re-authored corpus test).
- [ ] `testdata/golden/behavioral_test.go` — re-authored successor to `golden_parity_test.go` reading `CASES.json` and the locked corpora.
- [ ] A byte-identity or non-empty assertion over every re-frozen golden so a bare/missing golden cannot read as satisfied (see Assumption A3).

## Security Domain

> `security_enforcement` absent from config → treated as enabled. This phase has NO privileged code paths, NO new network surface beyond the Phase-1 `corpora:fetch` (already audited), and NO secrets. The applicable controls are boundary-respect and supply-chain-cleanliness.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — (no auth surface touched) |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | no | — (test-only code, no untrusted runtime input path) |
| V6 Cryptography | no | — (deleted TS-era capture scripts ran no crypto; gocapture does none) |

### Known Threat Patterns for {gocapture / golden harness}
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Deleted-corpora sweep silently reverting a licence notice | Tampering/Information disclosure | Do NOT touch `NOTICE` / `README.md` origin attribution (Pitfall 2 boundary) — removing colbymchenry MIT attribution would violate licensing, not just framing. |
| Re-introducing a TS/capture dependency that re-executes network or unvetted binaries | Spoofing/supply chain | `gocapture` extended on the Go-side path only (D-05); never re-add `mcp-capture.mjs` or a TS CLI dependency. |
| A re-freeze capture producing a bare/half-written golden that the byte-identity test rejects for a misleading reason | Tampering (integrity) | capture-to-temp-then-move (Pattern 2), non-empty + marker assertion before install. |
| Trusting `Entry.Dir` resolution without integrity verification | Tampering | the re-freeze precondition is `task corpora:assert` (four-part integrity check) before gocapture reads the locked trees — never index an unverified checkout. |

## Sources

### Primary (HIGH confidence) — read/verified in-tree this session
- `testdata/golden/golden_parity_test.go` (full Read) — all behavioral/corpus/CLI==MCP assertions, weft/colbymchenry/synthetic resolvers, `TestGoldenParity`/`TestGoldenBehavioral*`.
- `testdata/golden/golden_test.go` (full Read) — `TestGoldenFixturesExist`/`TestGoSideFixturesRegenerated`, volatile-key guard.
- `testdata/golden/gocapture/main.go` (full Read) — the capture program, `corpusSpec` seam, weft/colbymchenry/synthetic specs.
- `testdata/golden/capture.sh` (full Read) + `mcp-capture.mjs` (head) — the deleted TS-era capture path and exactly what it produced.
- `testdata/golden/README.md` (full Read) — the corpus table, provenance, volatile-fields authority; re-author target.
- `testdata/golden/parity_{java,python}_test.go` (full Read) + `{csharp,tsjs}` (grep) — the four D-12 language shape tests.
- `corpus/synthetic-parity/README.md` (full Read) — the verbatim a/b/c/d case map (FIXT-05 source).
- `corpora/manifest.json` (full) — locked set hugo/guava/serilog/requests with pinned SHAs + licences.
- `Taskfile.yml` (target-relevant sections), `.github/workflows/ci.yml` + `corpora.yml` (golden step + fetch/cache wiring), `test/wireoracle/cmd/wireoracle/main.go` + `oracle_test.go` (grep) — re-freeze discipline + CI touchpoints.
- `.planning/phases/01-*/01-CONTEXT.md`, `01-03/06/07-SUMMARY.md`, `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/STATE.md`, `.planning/phases/02-*/02-CONTEXT.md` — locked decisions, success criteria, Phase-1 delivered machinery.

### Secondary (MEDIUM confidence) — inferred from primary sources
- `internal/corpora/manifest.go:178-212` `Entry.Dir`/`CorpusRoot` derivation (line-grep-verified but not full-file read).
- `internal/query/{status,traverse,render_markdown}.go` doc-comment references to deleted `weft-go` paths (grep-verified; full sweep is Phase 4).

### Tertiary (LOW confidence)
- None — this phase's domain is fully grounded in the in-tree sources; no external web sources were required.

## Metadata

**Confidence breakdown:**
- Standard stack: **HIGH** — the phase reuses existing, Read-verified machinery (gocapture, internal/corpora, corpora targets, CLI==MCP harness); no new deps.
- Architecture: **HIGH** — the two-diff structure, gocapture-extension, and CASES.json contract are all directly anchored in D-01..D-08 + ROADMAP Notes (verbatim quoted).
- Pitfalls: **HIGH** — the four pitfalls are grounded in verified in-tree facts (file inventory, D-08 boundary, internal/ doc refs, .gitignore).
- **Open/assumed:** A1 (locked corpora index to stable symbol output) and Q1 (assertion philosophy) remain the genuine decision points; both flagged for the discuss-phase, neither presented as settled fact.

**Research date:** 2026-08-14
**Valid until:** 30 days (stable in-tree domain; re-freeze SHAs are pinned and static)
