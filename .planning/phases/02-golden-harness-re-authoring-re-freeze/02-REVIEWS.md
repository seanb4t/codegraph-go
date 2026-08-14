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

## Consensus Summary

### Convergence cycle 2 (post-revision review)

Cycle 1 (commit `37f5686`) raised **5 HIGH** blockers and **7 actionable** gaps. Plans were revised at `fbdd17f`. This cycle re-checks every cycle-1 finding against the revised `PLAN.md` text and the actual source/golden bytes, and checks the revision for knock-on contradictions. Independent source verification performed during this review (read the four PLAN.md files, gocapture/main.go, golden_test.go, render_markdown_test.go, and the on-disk behavioral goldens).

### Cycle-1 findings — RESOLVED (verified in revised plan text + source)

| Cycle-1 finding | Disposition now | Where resolved |
|---|---|---|
| H1 — Wave-2 breakage: `internal/query/explore_test.go` + `render_markdown_test.go` read deleted paths | RESOLVED | 02-02 Task 1 re-points `copySyntheticParityFixture` → `corpus/behavioral/src`, `loadGoldenOutput` → `corpus/behavioral/go-explore-multi.json`, neutralizes the traverse_test comment; files list + verify/acceptance cover it. Verified `loadGoldenOutput`'s re-point target (`go-explore-multi.json`) carries the `> ...` explore disclaimer the verbatim subtest needs. |
| H2 — behavioral path one directory too shallow (`testdata/corpus/behavioral`) | RESOLVED | 02-02 Task 2: `repoRoot := filepath.Dir(filepath.Dir(goldenDir))`; acceptance `test ! -e testdata/corpus/behavioral`. Two-hop fix is correct against `goldenDir = .../testdata/golden` (verified in main.go:96). |
| H3 — `lockedCorpusDir(t,"tsjs")` cannot resolve via `Language==language` | RESOLVED | 02-03 Task 1/2: explicit language→slug map (`go->hugo, tsjs->hugo, java->guava, csharp->serilog, python->requests`) + shared `lockedCorpusDir` helper + `TestPriorityLanguagesResolveToLockedCorpus`. Hugo's single `language:"go"` entry correctly supplies the tsjs leg via the map. |
| H4 — case-(d) assertion wrong (ranking vs surfacing) | RESOLVED | 02-02 Task 3 (case d) asserts the surfacing contract from `TestExploreStructuralBeatsLexical`, explicitly forbids "ranks above AccountBalanceHelper"; acceptance has a negative-wording grep. |
| H5 — `TestReFrozenGoldensValid` vacuous (glob-over-existing) | RESOLVED | 02-03 Task 3 + 02-04: enumerates the EXPECTED set from the gocapture spec table (never a glob), requires exist + non-empty + `{` marker + parse + positive verified-count; 02-04 arms strict full-set completeness with a delete-a-golden fails-rehearsal. |
| A1 — cases c/d query + assertion underspecified | RESOLVED | 02-02 Task 2: CASES.json now carries `query` + `assertion` per case; acceptance asserts all 4 cases carry both. Case-(c) parsing handled explicitly (selected-file/tests-clause vs bare text mention). |
| A2 — locked capture degrades gracefully, not fail-closed | RESOLVED | 02-03 Task 1: missing locked/behavioral source is a hard error (`regenerateCorpus` returns it, `main()` exits non-zero); skip-warn contract removed. |
| A3 — MCP staging + expected output set underspecified | RESOLVED | 02-03 Task 1: temp indexed repo staged then `query.OpenAt`; committed expected set per locked corpus = {explore, node, explore-multi, node-multi, explore-mcp, node-mcp}; MCP via `internalmcp.BuildServer` + go-sdk client, never mcp-capture.mjs. |
| A4 — matrix prose staleness + doc-mirror can't detect | RESOLVED | 02-03 Task 2 re-synchronizes matrix.go + mirrored doc to the fail-loud posture; `TestMatrix_DocMirrorsDescriptor` kept green. |
| A5 — aggregate-count determinism overstates ordering | RESOLVED | 02-03 truth + NOTE and 02-04 explicitly disclaim Files/Nodes/Edges != golden ordering; byte-stability via the repeated `golden:regen` is named the ordering oracle. |
| A6 — diff-classification terminology ("zero golden bytes") | RESOLVED | 02-02 CLASSIFICATION block: cleanup + byte-identical move, "no net-new golden byte", the re-freeze is the only production. Reconciles criterion-2 wording. |
| A7 — `corpus/behavioral/src/go.mod` retains `module synthetic-parity` | RESOLVED | 02-02 Task 2 renames the module line off `synthetic-parity`; framing gate scoped to cover `corpus/`. |

All 5 cycle-1 HIGHs and all 7 actionable findings are incorporated into task/action/acceptance_criteria/verify/must_haves/threat-model or explicitly reconciled. **None re-counted.**

### NEW defects introduced by the revision (source-verified, unresolved)

**1. HIGH — The framing ghost in the moved behavioral goldens contradicts 02-02's own framing gate.**
The two goldens 02-02 moves byte-identically (`go-explore-multi.json`, `go-node-multi.json`) embed the framing string in their `command` field. Verified on-disk: `go-explore-multi.json` command = `explore "user account" -p synthetic-parity`; `go-node-multi.json` = `node "Validate" -p synthetic-parity` (gocapture's `writeCapture` builds the `command` from `spec.name`, which the plan renames `synthetic-parity` → `behavioral` in the same revision). 02-02 moves those goldens to `corpus/behavioral/` preserving bytes and **forbids re-capturing/altering them**, yet its verify block (`! rg -qi "synthetic-parity|module synthetic" corpus/`) and acceptance criterion (`rg -qi "synthetic-parity" corpus/` returns nothing) run over `corpus/` — which now contains `corpus/behavioral/go-explore-multi.json` carrying the literal `synthetic-parity` string. **02-02's verify fails on the very goldens it claims to land byte-identically.** The plan never reconciles where/when the command-field framing is retired. A related knock-on: 02-04's human checkpoint gate (`git diff HEAD~1 HEAD | grep -iE 'parity|...synthetic-parity'` "expect no returned framing lines — any hit is a violation") will also false-flag the diff when the legitimate first re-capture removes `-p synthetic-parity` (the removal line itself matches the grep).
*Needed:* scope 02-02's framing gate to exclude the two moved goldens' `command` field (src/CASES.json/go.mod only), or explicitly defer command-field framing retirement to the 02-04 re-capture; and make 02-04's checkpoint grep treat *removal* lines as acceptable framing retirement, flagging only *newly introduced* framing.

**2. HIGH — 02-03 Task 1 reintroduces the one-hop behavioral outDir off-by-one.**
02-03 Task 1 action: behavioral spec outDir → `repo-root corpus/behavioral/` **`(filepath.Dir(goldenDir)+"/corpus/behavioral")`**. With `goldenDir = .../testdata/golden` (main.go:96), `filepath.Dir(goldenDir)` is `.../testdata` — ONE hop short. The formula resolves to `testdata/corpus/behavioral`, not the repo-root `corpus/behavioral` the prose names and that 02-02's corrected two-hop `filepath.Dir(filepath.Dir(goldenDir))` fixes. If implemented literally, gocapture writes the behavioral goldens to `testdata/corpus/behavioral`, while the re-pointed readers (`TestGoSideFixturesRegenerated`, `TestGoldenFixturesExist` glob `../../corpus/behavioral/*.json`, `TestReFrozenGoldensValid`) all look at repo-root `corpus/behavioral` — the re-freeze output would land in the wrong place and the guard/suite would not see it. This is the same defect class as cycle-1 H2, reintroduced in a different plan.
*Needed:* 02-03 must use `filepath.Dir(filepath.Dir(goldenDir))` (or the shared tested `repoRoot` helper 02-02 establishes) for the behavioral outDir, consistent with 02-02.

### Divergent Views
None between reviewers this cycle (single grounded reviewer + independent source verification). Both NEW findings were verified against the actual golden bytes and the plan text; neither is merely impressionistic.

### Strengths (unchanged from cycle 1, verified intact)
- The two-diff separation (rename/delete/move vs re-freeze) and the one-diff-one-cause attribution discipline remain sound and are carried consistently across 02-01→02-04.
- The language→slug map, hermetic fail-loud `lockedCorpusDir`, non-vacuous `TestReFrozenGoldensValid`, and run-fail-closed `regenerateCorpus` are all correctly specified.
- The MCP staging (temp indexed repo + `query.OpenAt`) and the per-corpus expected-set commitment give 02-04 a stable review number.

### Convergence verdict

Cycle 2. All five cycle-1 HIGHs and all seven actionables are fully resolved in the revised plans. However, the revision introduced **two new HIGH, source-verified knock-on defects** that would break execution if followed literally: (1) 02-02's framing gate contradicts the byte-identical moved goldens whose `command` field still embeds `-p synthetic-parity`, making 02-02's own verify fail and the 02-04 checkpoint grep false-flag the legitimate framing retirement; and (2) 02-03's behavioral outDir formula is the one-hop off-by-one (resolving to `testdata/corpus/behavioral`), contradicting the repo-root path 02-02 correctly established. Recommend a focused revision of 02-02's gate scoping and 02-03's outDir formula (`/gsd-plan-phase 2 --reviews`) before execution; no cycle-1 item needs re-examination.

### Convergence cycle 3 (post-revision review — configured-final cycle)

Cycle 2 (commit `0279431`) raised **2 HIGH** findings (both source-verified knock-on defects). Plans were revised at `b521194`. This cycle re-checks both HIGHs against the revised `PLAN.md` text, and checks the revision for new knock-on contradictions before the phase proceeds to execution.

#### Cycle-2 findings — RESOLVED (verified in revised plan text, not merely acknowledged)

| Cycle-2 finding | Disposition now | Where resolved |
|---|---|---|
| HIGH 1 — framing gate contradicts byte-identical moved goldens (`-p synthetic-parity` in `command` field), and 02-04 checkpoint grep false-flags the first re-capture's removal line | RESOLVED | 02-02 must_haves truths 46-47 scope the framing gate to identifiers/paths/names + authored data (module line, CASES.json), PARKS the moved goldens' embedded command-field byte explicitly OUT of the gate (byte-identical move cannot legally change it), and defer its retirement to 02-04, with the recurrence rule "a framing gate checks identifiers/paths/names; a golden-content gate checks the re-frozen output; never conflate." Task 2 verify (line 221) greps only go.mod + CASES.json + file NAMES (`find corpus/behavioral -iname '*synthetic*'`), never golden bytes — no false flag. 02-04 Task 2 action (133) + checkpoint step 4 (165) are scoped to ADDED (`^+`) lines only and explicitly name the `^-` `-p synthetic-parity` removal as the intended retirement, not a violation. |
| HIGH 2 — 02-03 Task 1 reintroduced the one-hop behavioral outDir off-by-one (`filepath.Dir(goldenDir)` → `testdata/corpus/behavioral`) | RESOLVED | 02-03 Task 1 action (line 122) now uses `filepath.Dir(filepath.Dir(goldenDir))` (TWO hops), explicitly names H2-2 as the same defect class as cycle-1 H2, and matches 02-02's repo-root correction; acceptance (line 146) asserts `-d corpus/behavioral && ! -e testdata/corpus/behavioral`. The locked-corpus outDir under `testdata/golden/corpus/<slug>` (one hop) is correct and deliberately different, matching the reader tests. |

Both cycle-2 HIGHs are incorporated into the plan text (task/action/acceptance_criteria + explicit reasoning), not merely acknowledged. **None re-counted.**

#### NEW defects from this revision? — none found

The committed-final revision does not introduce a knock-on:
- **Framing-gate scoping (02-02):** the parked `-p synthetic-parity` byte in `corpus/behavioral/go-{explore,node}-multi.json` is loaded only as a secondary regression cross-check (02-02 Task 3), never byte-diffed as a primary oracle between 02-02 and 02-04, so no test turns red on it in the window before the re-freeze retires it. The gate's `find -iname '*synthetic*'` matches file names only, which the moved goldens (`go-explore-multi.json`, `go-node-multi.json`) do not carry.
- **Two-hop path (02-03):** the behavioral outDir now uses the same two-hop formula 02-02 established; no one-hop residual remains on the behavioral side. The one-hop form survives only in the locked-corpus path where it is correct (readers live under `testdata/golden/corpus`).
- **Added-lines-only checkpoint grep (02-04):** both the grep in Task 2 action (133) and the human checkpoint (165) restrict to `^+` lines and treat `^-` removal lines as the intended retirement. No remaining false-flag source on the framing retirement.

No remaining actionable MEDIUM/LOW finding is visible that the latest `PLAN.md` files neither incorporate nor explicitly defer/reject.

### Convergence verdict

Cycle 3. Both cycle-2 HIGHs — the framing-gate/golden-content contradiction and the reintroduced one-hop behavioral outDir defect — are fully resolved in the revised PLAN.md text at `b521194`, with explicit scoping (identifiers/paths/names + authored data, parking the moved goldens' command-field byte and retiring it at 02-04's re-freeze) and the corrected two-hop formula (accompanied by a concrete `-d corpus/behavioral && ! -e testdata/corpus/behavioral` acceptance). This revision introduces no new knock-on defect across 02-01→02-04, and no actionable MEDIUM/LOW concern remains unresolved and invisible to /gsd-execute-phase. The phase's two-diff attribution discipline, hermetic fail-loud locked-corpus resolution, and the non-vacuous byte-identity guard are intact. Converged as configured-final: the phase may proceed to execution.

CYCLE_SUMMARY: current_high=0 current_actionable=0