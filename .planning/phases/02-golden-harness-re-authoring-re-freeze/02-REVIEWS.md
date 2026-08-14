---
phase: 2
reviewers: [codex]
reviewed_at: 2026-08-14T22:08:44Z
plans_reviewed:
  - 02-01-PLAN.md (rename diff A — identifiers + matrix/doc-mirror/README/coverage gates)
  - 02-02-PLAN.md (delete TS-era capture path + weft/colbymchenry corporations; move corpus/behavioral + CASES.json; re-author TestCorpusBehaviorSynthetic to D-09 property style)
  - 02-03-PLAN.md (extend gocapture; hermetic fail-loud locked-corpus resolution; golden:regen; TestReFrozenGoldensValid guard)
  - 02-04-PLAN.md (run the re-freeze; review single-cause attribution + zero identifier change; human gate)
---

# Cross-AI Plan Review — Phase 2

Convergence cycle 1 (fresh phase, no prior cycle). One external reviewer — Codex (`--codex`). Reviewer had repo access and was instructed to verify every claim against source; its review is a grounded plan review weighted at full consensus value.

## Codex Review

### 1. Summary

The plans have a strong architecture, but they are not execution-ready. Several source-verified issues would make Wave 2 or Wave 3 fail: two live `internal/query` tests retain paths that 02-02 deletes, the new behavioral path is calculated incorrectly, and `lockedCorpusDir(t, "tsjs")` cannot resolve Hugo from the manifest as designed. The behavioral assertions and golden-completeness guard also need tighter contracts. Overall risk is **HIGH**. (Codex could not execute Go tests because its session sandbox blocked Go's temp build dir; findings are from full plan reads and direct source tracing.)

### 2. Strengths

- The matrix rename is correctly treated as atomic. `goldenTestFuncsByLanguage` names concrete functions, while `TestMatrix_FullPriority4EntriesHaveGoldenTest` parses every golden test file and fails when a mapped function is absent (`internal/indexer/capability/matrix_test.go:167`, `:223`). Carrying this dependency in 02-01 is necessary.
- The proposed hermetic resolution seam is appropriate. `Load` validates the manifest, `Entry.Dir` derives collision-resistant SHA-bearing paths, `CorpusRoot` centralizes cache resolution, `LockedEntries` preserves manifest order (`internal/corpora/manifest.go:110,188,205,221`).
- Capture-to-temp-then-move addresses a real weakness: current `writeCapture` writes directly onto the committed path with `os.WriteFile` (`testdata/golden/gocapture/main.go:209`). The Phase 1 precedent records why a failed direct capture can leave a zero-byte oracle.
- Reusing the in-process MCP server is sound — the existing harness already constructs paired in-memory transports and calls real SDK tools (`testdata/golden/golden_parity_test.go:1446,1471`).
- The second `golden:regen` run in 02-04 is the right capture-time reproducibility gate; the query engine has deterministic tie-breakers for RWR-ranked nodes (`internal/query/explore.go:531`).
- Scope boundaries (bench, NOTICE, benchmark attribution) are thoughtfully identified and separated.

### 3. Concerns

- **HIGH — Wave 2 overlooks two live readers, so `go test ./...` fails after the corpus deletion/move.** `internal/query/explore_test.go` copies `testdata/golden/corpus/synthetic-parity/src` (`explore_test.go:33,43`); `internal/query/render_markdown_test.go` loads `testdata/golden/corpus/weft-go/explore.json` (`render_markdown_test.go:23`). 02-02 deletes both paths but lists neither test in its file set (`02-02-PLAN.md`). Breaks both the intermediate Task 1 verify and the completed wave.
- **HIGH — The planned behavioral corpus path is one directory too shallow.** `goldenDir` is `.../testdata/golden` (`gocapture/main.go:91`), so `filepath.Dir(goldenDir)` is `.../testdata`, not the repo root. The proposed path (02-02 and 02-03) resolves to `testdata/corpus/behavioral`, not `corpus/behavioral`.
- **HIGH — `lockedCorpusDir(t, "tsjs")` cannot work with the specified language-equality lookup.** The helper matches `Language == language` while the plan claims Hugo serves `"go"` and `"tsjs"`; Hugo's manifest entry is `"language": "go"` only; no locked entry has language `tsjs`, so the TSJS test fails loudly as "no locked entry."
- **HIGH — The reviewed-diff classification does not match criterion 2 as written.** 02-02 calls itself part of the first rename diff and says it changes "ZERO golden bytes," yet deletes four captured JSON fixtures and moves two others; the fixtures are documented captured goldens (`synthetic-parity README`). 02-03 is labelled "Diff B part 1" but adds tests/identifiers and deletes four resolver identifiers. The physical 02-04 capture may stay pure, but the Diff A/Diff B terminology makes the locked invariant unverifiable.
- **HIGH — Case (d) specifies an assertion the existing source says is mathematically wrong.** 02-02 says `ReconcileLedger` should "rank above" `AccountBalanceHelper`; the authoritative test documents that a connected seed cannot outrank an equally-weighted dangling seed under this RWR formulation — the property is that structural expansion *surfaces* it, not that it ranks above (`internal/query/explore_test.go:153,167`).
- **MEDIUM — Cases (c) and (d) are not fully derivable from the proposed `CASES.json`.** The example data gives no query for either case (`02-RESEARCH.md`); the test plan leaves case (c)'s `Explore(...)` argument unspecified and describes a vague query for (d). Case (c) also needs careful parsing (output may legitimately mention `TestAccountRecovery`/`recovery_test.go` in comments).
- **HIGH — `TestReFrozenGoldensValid` is potentially vacuous and is not a byte-identity test.** It is specified to glob files that already exist; a glob cannot detect an expected golden that is absent. The plan leaves "assert only fixtures that exist OR gate it" open, and the current analogous guard admits it can't detect stale fixtures. Parsing an envelope proves validity, not byte identity or gocapture provenance.
- **MEDIUM — Locked capture still degrades gracefully when it should fail closed.** Current `regenerateCorpus` prints `SKIPPED` and returns success on a missing source (`gocapture/main.go:117`); 02-03 preserves this for locked specs, which after corpora are mandatory could leave old goldens untouched while `golden:regen` succeeds.
- **MEDIUM — MCP capture and the expected output set are underspecified.** MCP handlers reopen an index via `query.OpenAt` (not direct Engine), so MCP capture needs indexed-repository staging (source copied into a temp repo with `.codegraph/store`); the plan cites session helpers but not this staging. Symbols, queries, output layout, MCP names, and expected fixture count are left discretionary, so 02-04's "expected count" has no stable value.
- **MEDIUM — Matrix prose becomes factually stale, and the doc-mirror cannot detect it.** 02-01 changes only identifiers inside claims that Python/TSJS self-skip and Java/C# use a source-as-spec fallback (`matrix.go:113,136,147`); after 02-03 the tests use locked corpora and fail loudly. `TestMatrix_DocMirrorsDescriptor` checks only verbatim string equality and can't catch both copies being identically false.
- **MEDIUM — Aggregate-count determinism does not guard golden output ordering.** The language tests compare only `Files/Nodes/Edges`; CLI==MCP compares two consumers of the same index. Neither proves repeat captures have identical ordering; the 02-04 second `golden:regen` is the real guard but the count assertions are claimed as enforcing it.

### 4. Suggestions

- Add `internal/query/explore_test.go` and `internal/query/render_markdown_test.go` to 02-02's dependents; re-point the authored-corpus test to `corpus/behavioral/src` and replace the weft fixture dependency before running `go test ./...`.
- Fix repo-root calculation to `filepath.Dir(filepath.Dir(goldenDir))`, or use one tested `repoRoot` helper.
- Define a typed capture-spec table (repo, language aliases, queries, symbols, output dir, expected filenames) driving gocapture, test resolution, and completeness validation; map both `go` and `tsjs` to Hugo explicitly.
- Treat 02-03 as source/tooling prep, 02-04 alone as the byte-only re-freeze; classify 02-02 as a cleanup/move diff or define "zero golden byte change" precisely.
- Add `query` and assertion-mode to cases (c) and (d); assert the correct properties (recovery_test.go not selected/rendered; ReconcileLedger surfaced, not ranked above AccountBalanceHelper).
- Resolve the pre-capture guard with an atomic capture manifest written last by gocapture (validate before manifests absent; exact expected file set + hashes after).
- Make missing locked/behavioral sources fatal in gocapture; the old skip-warn contract fit optional weft/online cloning, not the mandatory locked set.
- Specify MCP staging explicitly (temp indexed repo with `.codegraph/store` beneath it).
- Update `matrix.go` + mirrored doc in 02-03 for the fail-loud posture (a factual synchronization, not a Phase 4 sweep).
- Snapshot hashes / `git diff --binary` after regeneration and compare across the second run rather than relying on aggregate index counts.

### 5. Overall Risk

**HIGH.** The overlooked `internal/query` readers, the incorrect behavioral path, and the impossible TSJS manifest lookup are deterministic execution blockers; the case-(d) assertion conflicts with the existing algorithm contract, and the incomplete fixture guard could let a partial/skipped capture look successful. The underlying architecture is sound once these are resolved.

---

## Consensus Summary

Only one reviewer ran this cycle (Codex), so "agreed" below means corroborated by independent source verification performed during the review.

### Agreed Strengths
- Atomic rename with the matrix gate is necessary and correctly identified (the matrix test parses golden files and fails on a stale name).
- Capture-to-temp-then-move addresses a real integrity gap in the current direct `os.WriteFile` capture.
- In-process MCP capture and the hermetic `internal/corpora` resolution seam are the right shape.

### Agreed Concerns (verified against source)
1. **Wave-2 breakage (HIGH):** `internal/query/explore_test.go` and `internal/query/render_markdown_test.go` read `synthetic-parity/src` and `weft-go/explore.json` — both deleted by 02-02 — and are absent from 02-02's file list. `go test ./...` breaks.
2. **Off-by-one corpus path (HIGH):** `goldenDir` is `testdata/golden`, so `filepath.Dir(goldenDir)` is `testdata`; the plan's `corpus/behavioral` path lands at `testdata/corpus/behavioral`.
3. **TSJS resolution (HIGH):** no locked `tsjs`/`typescript` manifest entry exists; `lockedCorpusDir(t,"tsjs")` with `Language == "tsjs"` cannot resolve Hugo, which is a single `language:"go"` entry.
4. **Case-(d) assertion (HIGH):** the authoritative test documents that a connected seed cannot outrank a same-weighted dangling seed; "ReconcileLedger ranks above AccountBalanceHelper" is wrong — the property is *surfacing*, not ranking.
5. **Guard vacuity (HIGH):** `TestReFrozenGoldensValid` as a glob-over-existing is vacuous against a missing golden; the plan's "assert existing OR gate it" escape leaves the anti-bare-golden requirement unenforced.
6. **Matrix prose staleness (MEDIUM):** `matrix.go`/`docs/LANGUAGE-CAPABILITY-MATRIX.md` "self-skips / source-as-spec fallback" claims become false after 02-03; the doc-mirror gate is verbatim-only and cannot catch it.
7. **`corpus/behavioral/src/go.mod` retains `module synthetic-parity`** — a framing leak no plan task renames.

### Divergent Views
None between reviewers (single reviewer + independent source verification). Two points were judged as MEDIUM rather than HIGH after verification: the diff-classification terminology (02-02's "zero golden bytes" vs deleting/moving fixtures — functionally attribute-able, terminologically loose) and the fail-closed vs skip-warn behavior of gocapture (real, but resolved by the 02-03 hard-error for bare locked goldens when fully specified).

### Convergence verdict

Cycle 1. The phase's architecture (two-diff separation, D-09 property assertions, D-10 hermetic resolution, gocapture as sole capture path) is sound, but the plans are not execution-ready: five HIGH, source-verified blockers would break Wave 2/Wave 3 or would land a wrong assertion, and seven MEDIUM/LOW gaps are unaddressed in the latest PLAN.md files. Recommend `/gsd-plan-phase 2 --reviews` to incorporate before execution.

CYCLE_SUMMARY: current_high=5 current_actionable=7