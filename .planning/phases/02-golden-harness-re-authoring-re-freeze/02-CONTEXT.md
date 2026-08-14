# Phase 2: Golden Harness Re-authoring & Re-freeze - Context

**Gathered:** 2026-08-14
**Status:** Ready for planning

<domain>
## Phase Boundary

Re-author the golden suite so it reads as codegraph-go's own behavioral regression suite — its files, tests and fixtures named for what they assert, its goldens frozen from codegraph-go's own output against the Phase-1-locked third-party corpora — and remove the origin-driven capture path entirely.

**In scope:** CODE-02 (rename comparison framing out of the golden suite), FIXT-04 (delete the TS-era capture path and its corpora), FIXT-05 (survive the purpose-built behavioral corpus with framing stripped), FIXT-06 (re-freeze every golden from Go's own output against the locked corpora).

**Blocks:** Phase 3 (Non-Vacuity Proof / FIXT-03, FIXT-07) runs after this — a re-baseline that authors its own proof in the same change certifies its own oracle. This phase authors the suite; Phase 3 proves it is not vacuous.

**Does NOT touch:** `internal/corpora`, `tools/corpora`, `corpora/manifest.json` contents (the locked set is Phase 1's output and stays as-is), the benchmark path (`tools/bench/realcorpus`, `bench.yml` — Phase 6).
</domain>

<decisions>
## Implementation Decisions

### Renaming convention (CODE-02)

- **D-01:** The rename uses **behavioral vocabulary**. `parity_*_test.go` → `behavioral_*_test.go`; `TestGoldenParity*` → `TestCorpusBehavior*` / `TestBehavioral*`. The word `parity` and other comparison framing (`synthetic-parity`, "vs TS", head-to-head) are removed from identifiers, file names, and fixture-directory names. — **Reversibility:** reversible — pure rename, no behavior change; the re-freeze pass is separate.

- **D-02:** `golden` survives only as the neutral fixture concept (`testdata/golden/`, `gocapture`) — it names the *storage mechanism*, not a comparison target. Comparison words name a second implementation; `golden` alone names a frozen expected output, which is self-standing. `golden_parity_test.go` → `golden_test.go`'s behavioral equivalent; `golden_test.go` keeps or similarly re-names depending on what it asserts. The exact per-file names are Claude's discretion within the vocabulary, recorded in the plan.

### Behavioral corpus (FIXT-05)

- **D-03:** The purpose-built targeted corpus — currently `testdata/golden/corpus/synthetic-parity/` — **moves to an in-tree `corpus/behavioral/` directory**, with its four targeted cases intact. The `synthetic-parity` name is the framing to strip; it is dropped, not carried. `corpus/` is a committed, hand-authored, in-tree corpus — deliberately distinct from `corpora/` (Phase 1's third-party pinned-SHA manifest/observations/selection). The two are not merged: one is a pinned third-party fetch authority, the other is committed authored test input. — **Reversibility:** costly — moves committed source and every test that reads it; cheap to revert only before the re-freeze pass lands on top.

- **D-04:** The **case map is a committed data file** — `corpus/behavioral/CASES.json` — that the tests load, not a Go table embedded in test source. One readable source of truth for the four cases (overloaded same-named symbols; multi-word queries; the `Test*`-heavy weakly-connected cluster; structural-beats-lexical ranking); a later case author edits data, not test code.

### Re-freeze (FIXT-06)

- **D-05:** The re-freeze is driven by **extending `gocapture`** (`testdata/golden/gocapture/main.go`, currently a single-file program) so it covers every golden in scope, then re-freezing **all of them in ONE reviewed diff** against the locked third-party corpora (hugo, guava, serilog, requests). gocapture is extended, not replaced — it is the Go-side capture path FIXT-04 elevates to sole authority. — **Reversibility:** costly — this is the re-baseline; once landed the old byte content is gone except in history.

- **D-06:** The **rename and the re-freeze land as two separate reviewed diffs**, in that order: rename first (changes no golden byte — pure identifier move), re-freeze second (changes no identifier). One diff containing both makes any regression un-attributable, which is exactly what criteria 2 forbids. Every changed line in each diff traces to one named cause.

### Deletion scope (FIXT-04)

- **D-07:** Deleted: `testdata/golden/capture.sh`, `testdata/golden/mcp-capture.mjs`, the `weft-go` and `colbymchenry-codegraph` corpora (`testdata/golden/corpus/colbymchenry-codegraph/` and its captured fixtures), **and every in-tree reference to them** — the synthetic-parity/colbymchenry fixture files, doc references in `testdata/golden/README.md`, and any test code that reads them. This removes the TS-CodeGraph-heightened capture path the milestone exists to retire.

- **D-08:** `tools/bench/realcorpus`'s pinned `colbymchenry-codegraph` Entry **stays in place**, unchanged. It is bench tooling, not the golden harness; Phase 6 (Benchmark De-coupling) owns its reconciliation with the benchmark rewrite (BENCH-01/BENCH-02 retire the head-to-head numbers it feeds). Phase 2's sweep covers the harness, not the benchmark. Anything Phase 6 must then do with `realcorpus` is noted, not done here. — **Reversibility:** reversible — leaves an intact bench manifest Phase 6 edits deliberately.

### Assertion philosophy & corpus target (from research fork, ruled 2026-08-14)

- **D-09:** Tests assert **named behavioral properties of live output directly** — structural-beats-lexical, overloaded-symbol dedup, weakly-connected-cluster handling, multi-word query behavior — and the re-frozen goldens exist as **regression snapshots**, not the primary oracle. A failing test says *what* broke; a golden diff only says *something* changed. Goldens are evidence, not the assertion. — **Reversibility:** costly — rewrites the suite's assertion style, not just its names.

- **D-10:** The renamed behavioral tests resolve corpora **through `internal/corpora` (`Entry.Dir`, the pinned SHA-bearing path from Phase 1)** — never hardcoded SHAs, never a user env override as the default. The tests are hermetic: they run against the same pinned trees the fetch target and CI cache provide. This is what makes Phase 3's FIXT-03 ("runs against the fetched corpora on every CI run") achievable. — **Reversibility:** reversible — changes test fixture resolution, not product behavior.

### Claude's Discretion

- The exact new per-test/per-file names within the behavioral vocabulary (D-02 records the word-level rule; the planner picks concrete identifiers).
- The internal structure of `gocapture` extension (which functions/extraction the new capture cases use), provided it remains the Go-side path and does not re-introduce a TS dependency.
- Whether small, clearly-cosmetic comment text inside the swept files is dropped or reworded at rename time — cosmetic comment edits ride the rename diff; anything that changes an assertion or a byte of a golden belongs to the re-freeze diff.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase 1 (locked corpus set this phase freezes against)
- `.planning/phases/01-corpus-selection-by-measurement/01-CONTEXT.md` — D-01..D-17, especially D-04/D-05 (sparse/dense + MCP sparse), D-09 (manifest = sole pin authority), D-11 (shallow fetch keeps `.git`)
- `corpora/manifest.json` — the sole pin authority; four locked entries, pinned SHAs
- `corpora/observations.json` + `corpora/selection.json` — measured per-kind counts, frozen thresholds
- `docs/CORPUS-MEASUREMENT.md` — the generated record with coverage table, candidate ledger, threshold rationale

### The surfaces this phase touches
- `testdata/golden/gocapture/main.go` — the Go-side capture path, extended not replaced
- `testdata/golden/capture.sh` + `testdata/golden/mcp-capture.mjs` — the TS-era capture scripts deleted by FIXT-04
- `testdata/golden/corpus/synthetic-parity/` — the behavioral corpus that moves to `corpus/behavioral/`
- `testdata/wireoracle/cmd/wireoracle/main.go` — the human-run wire-oracle freeze entrypoint (deliberately not automated, locked v0.3.0)
- `test/wireoracle/oracle_test.go` — `TestFrozenTranscriptsMatch`, the byte-identity oracle the re-freeze must satisfy
- `.planning/phases/01-corpus-selection-by-measurement/01-03-SUMMARY.md` — how the one wire-oracle transcript was re-frozen in Phase 1; the same reviewed-diff discipline applies

### What stays deliberate
- `tools/bench/realcorpus/manifest.go` — pinned bench corpus; D-08 leaves it for Phase 6
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `tools/corpora` (`-mode entries/measure/select/kinds`) — the locked-corpus driver Phase 1 built; the re-freeze targets the locked corpora through it.
- `tools/corpora`/fetch + `internal/corpora` four-part integrity check — confirms the corpora the re-freeze reads are the pinned trees.
- `testdata/golden/gocapture/main.go` — the existing Go capture shape to extend.

### Established Patterns
- **Byte-identity oracle:** `TestFrozenTranscriptsMatch` compares golden bytes exactly — a re-freeze is a deliberate, atomically-reviewable full-file rewrite, and the rename pass must not move a single golden byte.
- **One-diff-one-cause:** every changed line traces to exactly one of {rename, re-freeze}; the two never share a diff.
- **`testdata/golden/README.md` volatile-fields policy** — the documenting authority for what the goldens may contain; its comparison-era wording is itself in scope to re-author.

### Integration Points
- `TestGoldenParity*` tests currently read the parity-named corpora; they move to reading `corpus/behavioral/` + the locked third-party corpora.
- The wire-oracle transcripts under `testdata/wireoracle/transcripts/` are golden-field inputs — those whose bytes move are re-frozen, owned here.
</code_context>

<specifics>
## Specific Ideas

- The four behavioral cases that must survive (FIXT-05, casemap-driven): overloaded same-named symbols; multi-word queries; the `Test*`-heavy weakly-connected cluster; structural-beats-lexical ranking. Each is a named case in `corpus/behavioral/CASES.json`.
- Comparison framing to strip from names/content: `parity`, `synthetic-parity`, "vs TS", "head-to-head", "colbymchenry-codegraph", "weft-go" as test/fixture/directory identifiers.
- Self-authored framing to keep: the corpus reads as codegraph-go's own regression suite — names describe the assertion, not a comparison.

</specifics>

<deferred>
## Deferred Ideas

- Reconciling `tools/bench/realcorpus` (its `colbymchenry-codegraph` pin, and its broader-than-MIT/Apache licence policy including BSD-3-Clause pebble) with this milestone — **Phase 6 (Benchmark De-coupling) owns it.** D-08 leaves it intact.
- Re-pointing or removing the `colbymchenry-codegraph` references in `bench.yml` and the `tools/bench/headtohead-*.json` captured benchmark outputs — those are BENCH surface, Phase 6, not this phase.
- `tools/bench/runner/main.go:482` `pinnedAt()` validates a checkout by `git rev-parse HEAD` alone (the pre-existing defect filed in STATE.md's Pending Todos) — fixed when Phase 6 adopts Phase 1's four-part integrity pattern.

</deferred>

---

*Phase: 02-golden-harness-re-authoring-re-freeze*
*Context gathered: 2026-08-14*
