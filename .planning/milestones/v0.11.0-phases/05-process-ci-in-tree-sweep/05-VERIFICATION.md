---
phase: 05-process-ci-in-tree-sweep
verified: 2026-08-16T02:09:16Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 4/5
  gaps_closed:
    - "Comments across internal/, tools/ and test/ carry no comparison framing (CODE-01) — all 10 previously-cited file:line locations (internal/cli/{search,node,files,uninit,serve}.go, internal/cli/githooks_test.go, internal/mcp/tools.go, internal/agents/codex.go, internal/query/traverse_test.go, testdata/golden/behavioral_test.go) independently reproduce as zero-hit"
  gaps_remaining: []
  regressions: []
deferred:
  - truth: "internal/bench and tools/bench comparison-runner framing is removed"
    addressed_in: "Phase 6"
    evidence: "ROADMAP.md Phase 6 success criterion 2 (BENCH-02): 'tools/bench contains no comparison runner, and internal/bench.CheckRegression... still fires on a real regression.' WINDOWS #16 explicitly records internal/bench/rss.go and tools/bench/runner/main.go as deferred here BY DESIGN. This re-verification's tree-wide backstop also re-confirms internal/bench/metrics.go:7 and tools/bench/realcorpus/manifest.go:3 carry the same class of framing describing the same runner — same disposition applies."
human_verification: []
---

# Phase 5: Process, CI & In-Tree Sweep Verification Report

**Phase Goal:** A contributor filing an issue or opening a PR, and an agent reading the source, meet a project described on its own terms — with TypeScript-the-indexed-language intact and `codegraph migrate` removed (maintainer ruling 2026-08-15).

**Verified:** 2026-08-16T02:09:16Z
**Status:** passed
**Re-verification:** Yes — after gap closure (plans 05-07, 05-08)

## Goal Achievement

### Observable Truths (ROADMAP.md Phase 5 Success Criteria — the sole contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A contributor filing an issue sees no comparison framing in any of the 5 issue templates (PROC-01) | ✓ VERIFIED (regression) | `rg -Uwi -o 'TS\|parity\|drop-in\|upstream'` over `.github/ISSUE_TEMPLATE/*.yml` → 0 hits. Unchanged since prior pass; no plan in this closure wave touched these files. |
| 2 | A contributor opening a PR sees none in `pull_request_template.md` or the 3 `PULL_REQUEST_TEMPLATE/*` variants (PROC-02) | ✓ VERIFIED (regression) | Same census over `.github/pull_request_template.md` + `.github/PULL_REQUEST_TEMPLATE/{enhancement,feature,fix}.md` → 0 hits. |
| 3 | Every workflow still passes `actionlint` and its own required status checks after the sweep, with job names, step names and comments carrying no retired framing (PROC-03) | ✓ VERIFIED (regression) | `actionlint .github/workflows/*.yml` exits 0 (all 14 files). No workflow file was touched by 05-07/05-08. |
| 4 | Comments across `internal/`, `tools/` and `test/` carry no comparison framing, while `internal/indexer/tsextract`, the language registry and the capability matrix are preserved as product surface (CODE-01) | ✓ VERIFIED | All 10 previously-cited locations independently re-checked: zero bare-`TS` hits remain in `internal/cli/{search,node,files,uninit,serve}.go`, `internal/cli/githooks_test.go`, `internal/mcp/tools.go`, `internal/agents/codex.go`, `internal/query/traverse_test.go`. `internal/cli/root.go` (05-08's own additional find, WINDOWS #17) is also clean. `testdata/golden/behavioral_test.go` carries exactly 2 residual `TS` hits, both the ledger's own PAST-TENSE-HISTORY sentence ("The TS-era capture path... are gone as of this phase (FIXT-04)"), verbatim, per D-02. A full no-exclusion tree-wide backstop (below) found no additional unresolved framing beyond WINDOWS entries 18/19, which this verification independently adjudicates as legitimate D-02 product truth, not D-01 framing (see adjudication section). |
| 5 | `codegraph migrate` is removed entirely: the command, `internal/migrate/`, its fixture, and the `modernc.org/sqlite` sole-use dependency are gone, nothing references it (CODE-03 amended) | ✓ VERIFIED (regression) | `rg -il 'modernc\.org/sqlite\|internal/migrate\|codegraph migrate' --glob '!.planning' .` → 0 hits. `internal/migrate/` absent. `go build ./...` and `go vet ./...` both exit 0. |

**Score:** 5/5 truths verified

### Entries 18/19 Adjudication (Independent, Not Deferred to Orchestrator/Executor)

WINDOWS.md records entries 18 and 19 as **waived**, but the ledger's own text withdraws the numeric justification ("open_count==3") that originally produced that status and explicitly leaves the substance "open for adjudication." I read both lines in full surrounding context (not the one-line WINDOWS description) and adjudicate independently against D-01/D-02 and criterion 4's own text.

**Entry 18 — `internal/indexer/resolve.go:152`:**

```
//   2. target is NOT an interface (a class/struct target
//      — D-09's new case) -> "extends", distinct from
//      the bare "embeds" string this branch used to fall
//      through to unconditionally. This is the D-09
//      addendum's answer to Open Question A2: Go's
//      structural composition is the closest analog TS's
//      `extends` RANK_EDGES kind has in Go, so a
//      class/struct-extends-class/struct reference is
//      reclassified rather than left as "embeds".
```

Read in full context (`internal/indexer/resolve.go:126-159`), "TS" here means TypeScript-the-indexed-language's `extends` keyword, cited as the reference point for how the cross-language `RANK_EDGES` edge-kind taxonomy classifies Go's struct embedding. This is schema/kind-classification design rationale spanning codegraph-go's *own* multi-language extractor family (Java's `implements`/`extends`, C#'s base_list, Go's embedding, TS's `extends` all map onto the same `RANK_EDGES` vocabulary) — structurally identical to the `internal/indexer/mainstream/phpextract/types.go:49` and `rustextract/types.go:40,54` comments ("exactly like Java/C#/TS/Python's own extends/implements-shaped refs") that 05-08's own backstop table classified KEEP-PRODUCT-SURFACE for the same reason. It is not a claim that codegraph-go's design "matches," "mirrors," or achieves "parity" with the prior TS CodeGraph *project* (D-01's illustrative vocabulary) — it never mentions TS CodeGraph, a tool, an install, or a behavior comparison. **Verdict: D-02 product truth ("TypeScript as an indexed language," "the language registry") — not D-01 framing. Criterion 4 is not violated by this line.**

**Entry 19 — `internal/indexer/goextract/goextract.go:858`:**

```
// deliberate, bounded scope, not a silent drop of ground truth: TS's own
// "references" semantic is already a broad, heuristic "identifier use"
// signal, and Go's syntax makes exhaustive coverage categorically harder
// to bound safely than calls/embeds...
```

Traced "TS's own 'references' semantic" to a concrete code referent: `internal/indexer/tsextract/tsextract.go:1018,1082` emit `goextract.RefKindReferences` — the *same* Go type this Go-extractor comment is discussing — for TypeScript's bare-identifier/member-access value reads. "TS's own" is self-referential within codegraph-go's own extractor family (comparing the Go extractor's bounded references-walk scope to the TS extractor's already-broad references-walk scope, both living in this repository), not a reference to the prior TS CodeGraph tool's `find_references`-style feature. Same disposition as entry 18. **Verdict: D-02 product truth (TS-the-indexed-language's own extraction behavior, cited for cross-extractor design consistency) — not D-01 framing.**

**Conclusion:** Both entries resolve term-by-term as legitimate retained uses under D-02, satisfying criterion 4's "every retained use resolved term-by-term with a recorded reason, never by regex" clause — this section is that recorded reason. Criterion 4 is TRUE. This adjudication does not require or request any change to `.planning/WINDOWS.md`'s `waived` status (a tool-owned ledger — out of this verifier's write scope); it stands as the independent judgment the ledger itself deferred to a future reader.

### Census Instrument

**Positive control (proven before trusting any zero):** planted a synthetic two-line comment (`// no TS` / `// precedent here`) and confirmed both `rg -U -o -w 'TS'` and `rg -U -o 'no TS\s*\n\s*//?\s*precedent'` match across the line wrap — a line-based `rg` (no `-U`) cannot see this, which is precisely the failure mode that produced the original 10-location undercount. Only after this control passed did zero-hit results below count as evidence.

**Bare backstop (this verification's own instrument, independent of 05-08's 13-pattern set):** `rg -U -o -w -n 'TS' internal/ tools/ test/ testdata/` — **no directory exclusions** — over the full tree: 116 raw word-bounded hits across 36 files. Every file was read in context and classified:

| Classification | Files (representative) | Disposition |
|---|---|---|
| PRODUCT-SURFACE (TypeScript-the-indexed-language machinery) | `internal/indexer/{languages,languages_test,languages_typescript}.go`, `internal/indexer/tsextract/*.go`, `internal/indexer/routes/{express,registry,walk,express_test}.go`, `internal/indexer/mainstream/{phpextract,rustextract}/types.go`, `internal/corpora/{coverage_test,record}.go` | KEEP — explicitly named as preserved surface by criterion 4 itself |
| RECORDED-KEEP (05-04's ledger, unedited this phase) | `internal/query/{status,expand,explore,gather,scoring}.go`, `internal/query/files_status_test.go` | KEEP — historical "TS install's dist JS unreadable on this machine" provenance notes, out of this plan's edit scope |
| BUG-PRECEDENT-CITATION (D-02 carve-out) | `internal/agents/{claude,instructions,opencode}.go`, `internal/gitmeta/detect.go`, `internal/graphstore/keys.go` | KEEP — cites a historical bug-tracker issue number, not ongoing comparison |
| PAST-TENSE-HISTORY (D-02 carve-out) | `testdata/golden/behavioral_test.go` (2 hits), `testdata/golden/gocapture/main.go` | KEEP — "TS-era capture path... retired" |
| False positive (not TypeScript at all) | `test/wireoracle/normalize.go`, `normalize_test.go` | KEEP — `<TS>` is a Timestamp redaction placeholder |
| Provenance/generated | `internal/schema/graph.pb.go`, `graph.proto` | KEEP — one-time design-provenance note; `.pb.go` is `protoc-gen-go` output |
| Phase-6-deferred (WINDOWS #16, out of CODE-01 scope) | `internal/bench/{rss,metrics}.go`, `tools/bench/runner/main.go`, `tools/bench/realcorpus/manifest.go` | NOT COUNTED — the still-live head-to-head runner, explicitly Phase 6 BENCH-02's scope |
| Adjudicated this pass (WINDOWS #18, #19) | `internal/indexer/resolve.go:152`, `internal/indexer/goextract/goextract.go:858` | KEEP — D-02 product truth, see adjudication above |

**Zero unresolved D-01 comparison-baseline framing found** outside the two now-adjudicated `internal/indexer/**` lines. No new locations beyond what WINDOWS.md's ledger and 05-08's own backstop table already enumerate were found.

**Previously-failed 10 locations, individually re-checked:** `internal/cli/search.go`, `node.go`, `files.go`, `uninit.go`, `serve.go`, `githooks_test.go`, `internal/mcp/tools.go`, `internal/agents/codex.go`, `internal/query/traverse_test.go`, `internal/cli/root.go` — each independently queried with `rg -U -o -n -w 'TS'`, all return zero hits.

### Product Surface Preserved

- `internal/indexer/tsextract/` intact: `tsextract.go`, `tsextract_test.go`, `types.go`, `d09_test.go`, `resolution_test.go` all present.
- `internal/indexer/languages_typescript.go` still registers TypeScript, TSX and JavaScript as three separate languages (4 `TypeScript`/`TSX`/`JavaScript` mentions).
- `internal/indexer/languages_test.go` still has `TestLanguageRegistry_TypeScript` (1 func definition, 3 total mentions — the extra 2 are a doc comment and a legitimate cross-reference added by 05-07's edits, not a regression).
- `docs/LANGUAGE-CAPABILITY-MATRIX.md` still documents TypeScript (3 occurrences).
- `go build ./...` and `go vet ./...` both exit 0.
- `go test -count=1 ./internal/cli/... ./internal/githooks/... ./internal/mcp/... ./internal/agents/... ./internal/query/... ./testdata/golden/... ./internal/indexer/...` — all packages pass, including `internal/indexer/tsextract` and every mainstream-tier extractor.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| PROC-01 | 05-02 | 5 issue templates, no comparison framing | ✓ SATISFIED | Criterion 1 above |
| PROC-02 | 05-02 | PR template + 3 variants, no comparison framing | ✓ SATISFIED | Criterion 2 above |
| PROC-03 | 05-03 | Workflows pass actionlint + own required checks, no framing in names/comments | ✓ SATISFIED | Criterion 3 above |
| CODE-01 | 05-04, 05-05, 05-06, 05-07, 05-08 | Comments in internal/, tools/, test/ carry no comparison framing; product surface preserved | ✓ SATISFIED | Criterion 4 above — all 10 gap locations fixed, tree-wide backstop clean modulo the independently-adjudicated indexer entries |
| CODE-03 (amended) | 05-01 | `codegraph migrate` removed entirely | ✓ SATISFIED | Criterion 5 above |

Note: `.planning/REQUIREMENTS.md` still shows PROC-01/02/03 as unchecked `[ ]` checkboxes despite being satisfied since the initial verification pass — this is a tracking artifact in a downstream document, not a gap in the underlying truth, and is out of this verifier's write scope (checkbox maintenance belongs to `/gsd-ship`/state-update tooling, not to hand-editing a tool-owned file).

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX` debt markers in any file touched by 05-07 or 05-08 (`internal/cli/{search,node,files,uninit,serve}.go`, `internal/cli/githooks_test.go`, `internal/mcp/tools.go`, `internal/agents/codex.go`, `internal/query/traverse_test.go`, `internal/cli/root.go`, `testdata/golden/behavioral_test.go`, `.planning/WINDOWS.md`). No `TBD`/`FIXME`/`XXX`/`HACK`/`PLACEHOLDER` found in any of the 36 files the tree-wide backstop touched.

### Build & Test State

- `go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test -count=1 ./internal/cli/... ./internal/githooks/... ./internal/mcp/... ./internal/agents/... ./internal/query/... ./testdata/golden/... ./internal/indexer/...` → all PASS (24 packages)
- `actionlint .github/workflows/*.yml` → exit 0
- `.planning/WINDOWS.md` ledger: `open_count: 3` (entries 12, 13, 16 — daemon test load-sensitivity, pre-existing docs/RELEASE.md drift, Phase-6-deferred bench framing; none CODE-01), `waived_count: 2` (entries 18, 19 — independently adjudicated above as legitimate product truth, not open gaps), `fixed_count: 14`.

### Gaps Summary

None. The one failed criterion from the initial verification (CODE-01) is now true: all 10 previously-cited locations independently re-verified as fixed, and this re-verification's own no-exclusion tree-wide backstop census over `internal/`, `tools/`, `test/`, `testdata/` found no further unresolved comparison-baseline framing. The two `internal/indexer/**` findings the closure plan (05-08) itself flagged as borderline and left "waived... needs adjudication" were independently traced to their full surrounding code context and adjudicated as TypeScript-the-indexed-language schema/extraction design rationale (D-02 product truth), not references to the prior TS CodeGraph project as a comparison baseline (D-01). All 5 ROADMAP.md Phase 5 success criteria hold. Product surface (`tsextract`, the language registry, the capability matrix, TypeScript/TSX/JavaScript registration) is confirmed intact, not collapsed by the sweep. `codegraph migrate` removal (CODE-03) remains fully in place. Build, vet, `actionlint`, and every touched package's test suite are green.

Not counted as gaps: `internal/bench/{rss,metrics}.go`, `tools/bench/{runner/main,realcorpus/manifest}.go` — all describe the still-live head-to-head comparison runner, which ROADMAP.md Phase 6 (BENCH-02) explicitly scopes for removal; WINDOWS #16 already records this deferral by design.

---

*Verified: 2026-08-16T02:09:16Z*
*Verifier: Claude (gsd-verifier)*
