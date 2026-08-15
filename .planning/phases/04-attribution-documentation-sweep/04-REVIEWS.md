---
phase: 4
reviewers: [codex]
reviewed_at: 2026-08-15T16:28:07Z
plans_reviewed: [04-01-PLAN.md, 04-02-PLAN.md, 04-03-PLAN.md]
---

# Cross-AI Plan Review — Phase 4

## Consensus Summary

One external reviewer ran (Codex). Its verdict is source-grounded (it opened the actual NOTICE, README, CONTRIBUTING, capability-matrix, and test files and confirmed the plans' line-level claims). The two HIGH findings it raises are both against the *literal* ROADMAP/REQUIREMENTS wording ("and nothing else" for NOTICE; "no comparison framing" applied to the README Performance numbers table) — and both are **settled context** for this phase (see the Convergence verdict below): the locked decisions D-01/D-02/D-05 and the Phase-6 boundary explicitly retain the NOTICE license-warning block + fidelity guard + Third-party dependencies section and the README head-to-head numbers table. They are not plan defects.

The reviewer's actionable value is concentrated in the MEDIUM/LOW tier: the plans' automated verifies use narrow exact-phrase greps that are weaker than the phase's own term-by-term census contract, a few cross-plan verification claims are not mechanically wired, and the DOCS-02 reference sweep's grep pattern misses camelCase variants of the deleted artifacts' names.

### Agreed Strengths

- Line-level census accuracy confirmed against source: NOTICE structure, README Performance/Status/Relationship/License sites, CONTRIBUTING rows, capability-matrix rows, and the FLAG-PARITY reference inventory all match the actual files.
- The `man.go:27` census correction (an 11th FLAG-PARITY reference the research's enumerated-10 table missed) is real and verified in source.
- The DOCS-05 deferral is recorded as a knowing coverage reduction, not hidden.
- The capability-matrix doc↔code contract (matrix_test.go asserting coverage values + gap bullets) is genuinely backed by the test.
- Zero file overlap across the three parallel Wave-1 plans is confirmed.

### Agreed Concerns

- Narrow exact-phrase automated greps can pass while residual comparison-flavored terms survive (e.g. "known divergences", camelCase `FlagParity`/`TestFlagParityDocCoversRegisteredFlags`).
- Cross-plan verification claims (04-03 "verifies README"; the phase-final full-tree reference sweep) are asserted in prose but not installed as an executable plan-owned gate.

### Divergent Views

- Codex reads ATTR-01's "and nothing else" and the DOCS-01 "no comparison framing" criterion literally and concludes the plans fail them. This phase's binding decisions (04-CONTEXT D-01/D-02/D-05 and the ROADMAP's own Phase-6 note) settle both as intentional, recorded exceptions — the divergence is between requirement wording and locked decisions, resolved in favor of the locked decisions (which the ROADMAP Phase-4 Notes and REQUIREMENTS deferral rows already record).

---

## Codex Review

# Cross-AI Plan Review — Phase 4

## Summary

The plans are unusually explicit about file ownership, protected product truth, positive controls, and intentional coverage loss. The source confirms most cited locations and test mechanisms. However, Plan 04-01 does not satisfy two roadmap-level acceptance criteria: it preserves substantial extra NOTICE content despite "and nothing else," and it leaves inherently comparative benchmark claims in README. The verification design also lacks a single executable phase-wide gate proving the final cross-plan state. Overall, the plans need revision before execution.

## Plan 04-01 — NOTICE and README

### Summary

The plan correctly identifies the attribution argument and protects LICENSE, migration, and indexed-language functionality. Its proposed edits are precise. Nevertheless, the intended NOTICE result directly contradicts ATTR-01, and the README result still contains unmistakable comparison framing. The live GitHub license check also cannot validate uncommitted local edits.

### Strengths

- The plan accurately maps the current NOTICE structure:

  - The license-detection warning is at `NOTICE:3-10`.
  - The comparison argument occupies `NOTICE:14-37`.
  - The copyright transcription is at `NOTICE:39-46`.
  - Third-party dependency material begins at `NOTICE:52`.

- It protects the exact MIT text in `LICENSE:1-21` and explicitly prohibits editing it. This is an appropriate safeguard for ATTR-03.

- It separates origin framing from legitimate product facts:

  - `README.md:137` documents the real `migrate` capability.
  - `README.md:149-153` lists TypeScript, TSX, and JavaScript as supported languages.
  - The plan positively asserts that these rows survive.

- It correctly recognizes that `README.md:189-191` describes an obsolete harness. The current language claims a diff against frozen TypeScript output, while the plan replaces it with the post-reauthoring behavioral-corpus model.

- The proposed README License clause is past tense and points to NOTICE, satisfying the form intended by ATTR-02.

### Concerns

- **HIGH — The NOTICE result violates ATTR-01's explicit "and nothing else" criterion.**  
  The roadmap requires "the MIT copyright transcription plus one sentence of origin and nothing else" at `.planning/ROADMAP.md:207`; `.planning/REQUIREMENTS.md:12` likewise says NOTICE is trimmed to the transcription plus one origin sentence. Yet the plan requires preservation of the eight-line licensee warning, the fidelity commentary, and the entire Third-party dependencies section at `.planning/phases/04-attribution-documentation-sweep/04-01-PLAN.md:22,28,35-36`. Those are substantial additional contents currently visible at `NOTICE:3-10`, `NOTICE:48-50`, and `NOTICE:52-69`.

- **HIGH — README would still contain comparison framing.**  
  `README.md:160-164` publishes "8.1× faster," "12.9× lower," "2.8× lighter," and similar ratios. Those claims are comparative even if the introductory words "Measured against" are removed. Retitling the subject to "codegraph (TypeScript)" at `README.md:163` leaves the comparator explicit. This conflicts with the roadmap criterion that a README reader encounters no comparison framing at `.planning/ROADMAP.md:210`. The plan knowingly retains the table at `04-01-PLAN.md:25,126-137`.

- **MEDIUM — The proposed Status text can retain comparison language through "known divergences."**  
  The suggested replacement at `04-01-PLAN.md:124` ends with "known divergences are logged." A divergence is necessarily divergence from something. The automated check only rejects four exact old phrases at `04-01-PLAN.md:129`; it would accept this residual framing.

- **MEDIUM — `gh api` does not validate the local NOTICE edit.**  
  `gh api repos/seanb4t/codegraph-go/license` queries the repository state hosted by GitHub, not the executor's uncommitted working tree. Therefore, the claim that it verifies license detection "after the NOTICE edit" at `04-01-PLAN.md:24,94,102` is not mechanically true until the change is pushed to the queried branch. Since `LICENSE` is the detected file and is not changing, a local byte-identity check is the relevant pre-push protection.

- **MEDIUM — "Only origin mention" is checked with one chosen phrase, not as a census.**  
  The guard counts `began as a Go rewrite of CodeGraph`, but another origin formulation could remain and still pass. Current origin language appears throughout `README.md:193-202`, so the deletion is correct, but the postcondition needs a broader, reviewed origin-reference census.

### Suggestions

- Resolve ATTR-01 explicitly before execution:

  - Either reduce NOTICE to the transcription and one sentence, as the requirement says; or
  - amend ATTR-01/roadmap wording to enumerate the warning, fidelity note, and dependency section as permitted exceptions.

- Move or neutralize the README multiplier table in this phase. Deferring `docs/BENCHMARKS.md` to Phase 6 does not justify retaining comparison framing in README, which is directly governed by DOCS-01.

- Rewrite Status without "divergence," for example: "Behavioral regression coverage is provided by named property assertions over the in-repo behavioral corpus."

- Add local checks such as:

  - `git diff --exit-code HEAD -- LICENSE`
  - a recorded checksum or exact comparison against `git show HEAD:LICENSE`
  - a post-push `gh api` check as a separate release/integration gate

- Add an exact structural NOTICE assertion, such as comparing normalized expected content rather than counting a few retained phrases.

### Risk Assessment

**HIGH.** The proposed implementation can pass every listed automated check while failing the literal NOTICE and README phase criteria.

---

## Plan 04-02 — FLAG-PARITY Deletion and Reference Sweep

### Summary

This is the strongest of the three plans. The deletion targets and reference sites match the source, the removed test's actual behavior is understood correctly, and the deliberate loss of documentation coverage is recorded. The main weakness is that the final "nothing in the tree references either artifact" claim is not enforced by this plan's automated verification or by a clearly defined phase-level gate.

### Strengths

- The plan correctly describes the deleted test's mechanism.  
  `internal/cli/flag_parity_test.go:40-64` walks the complete Cobra command tree, visits registered flags, and checks whether each long flag name appears in the document. Missing documentation is accumulated at `internal/cli/flag_parity_test.go:49-69`.

- It correctly identifies fail-closed behavior.  
  `os.ReadFile` at `internal/cli/flag_parity_test.go:41-44` fails the test when the document is absent.

- The known reference inventory is source-backed:

  - `NOTICE:25,33`
  - `CONTRIBUTING.md:13`
  - `.github/pull_request_template.md:37`
  - `.github/PULL_REQUEST_TEMPLATE/feature.md:42`
  - `.github/ISSUE_TEMPLATE/enhancement.yml:77`
  - `.github/workflows/auto-close-unsolicited-prs.yml:91`
  - `internal/mcp/instructions_contract_test.go:16`
  - `internal/mcp/tools_schema_drift_test.go:36`
  - `internal/cli/man.go:27`

- The additional `man.go` census correction is valid.  
  The current comment at `internal/cli/man.go:24-31` contains the missed FLAG-PARITY reference.

- The plan avoids inventing a replacement for the deleted documentation and explicitly records the resulting coverage reduction at `04-02-PLAN.md:102-106`. That is aligned with the locked deferral of DOCS-05.

- The task-specific verification is appropriately scoped for parallel execution: it checks the seven owned files rather than failing because NOTICE or CONTRIBUTING has not yet been edited.

### Concerns

- **MEDIUM — No executable gate enforces the final full-tree reference claim.**  
  The action describes a comprehensive full-tree command at `04-02-PLAN.md:146-148`, but the actual automated verification at `04-02-PLAN.md:150-155` scans only owned files. The full-tree check is described as something that "must be empty after all three phase plans land," but it is not installed as a phase-final command.

- **MEDIUM — The reference vocabulary is narrower than the requirement.**  
  The full-tree pattern checks only `FLAG-PARITY|flag_parity`. It would not catch variants such as `FlagParityDoc`, `TestFlagParityDocCoversRegisteredFlags`, or a link spelling change unless they retain one of those exact fragments. Some surviving comments currently refer to the test semantically, so a broader deletion-artifact inventory would be safer.

- **LOW — Site-count language is confusing.**  
  The plan repeatedly says "8 owned reference sites," but enumerates nine edits at `04-02-PLAN.md:136-144`, across seven files. Some files contain multiple rows, so this is likely just inconsistent terminology, but it makes audit accounting harder.

- **LOW — Full `go test ./...` does not prove all specifically named unaffected mechanisms.**  
  It does prove the suite remains green, but the statement that this specifically proves `TestWorkflowRunBodiesInvokeTask` is somewhat stronger than necessary. The deleted test has no runtime integration with that workflow guard; absence of references can be established directly through source census.

### Suggestions

- Add a mandatory final phase gate:

  ```sh
  rg -n 'FLAG-PARITY|flag_parity|FlagParity|TestFlagParityDocCoversRegisteredFlags|flagParityDocPath' \
    --hidden --no-ignore \
    -g '!.git/**' -g '!.codegraph/**' -g '!.planning/**' \
    -g '!.claude/**' -g '!dist/**' -g '!graphify-out/**' .
  ```

- Make the plan's terminology precise: "seven owned files containing nine reference rows."

- Include `git ls-files` in the automated command, not only in prose acceptance criteria, to prove both deletion targets have left the index.

- Keep the intentional coverage reduction conspicuous in the summary and milestone verification; it is a real reduction, even though it is authorized.

### Risk Assessment

**LOW to MEDIUM.** The edits themselves are straightforward and well understood. Risk comes primarily from the missing phase-final reference sweep.

---

## Plan 04-03 — CONTRIBUTING, Capability Matrix, and Verify-Only Sweep

### Summary

The CONTRIBUTING and capability-matrix changes are well targeted, and the matrix contract is backed by real tests. The verify-only portion is less convincing: it depends on manual classification that the automated command never performs, omits README despite claiming to verify it, and overstates what the matrix test protects.

### Strengths

- The three CONTRIBUTING targets exactly match the current source:

  - Origin/parity rationale at `CONTRIBUTING.md:11-14`
  - "golden parity" at `CONTRIBUTING.md:83-85`
  - Frozen TypeScript output claim at `CONTRIBUTING.md:190-192`

- The rewrite preserves the contributor-process mechanism rather than merely deleting the paragraph. The "Issue first" requirement remains at `CONTRIBUTING.md:6-17`.

- The capability-matrix targets are accurate:

  - `golden-parity` legend at `docs/LANGUAGE-CAPABILITY-MATRIX.md:12-14`
  - Go/reference/weft wording at `docs/LANGUAGE-CAPABILITY-MATRIX.md:57-61`
  - Shared caveat at `docs/LANGUAGE-CAPABILITY-MATRIX.md:184-194`
  - Contextual "original" use at `docs/LANGUAGE-CAPABILITY-MATRIX.md:180-183`

- The matrix test genuinely checks table values.  
  `internal/indexer/capability/matrix_test.go:104-152` parses the documentation table and compares each axis with the Go descriptor.

- It also checks that each machine-readable gap appears verbatim in the document at `internal/indexer/capability/matrix_test.go:158-162`, supporting the warning not to alter gap text.

- It preserves indexed-language product truth in `docs/LANGUAGE-CAPABILITY-MATRIX.md:27-29`, which is important under D-05.

- The verify-only KEEP decisions are generally reasonable. For example, the historical branch name at `docs/RELEASE-PROCEDURES.md:60` and process-oriented "upstream" at `docs/RELEASE-PROCEDURES.md:581` are not project-origin comparisons.

### Concerns

- **MEDIUM — The automated verify-only task does not perform the promised census.**  
  The action requires a word-boundary-aware term census and classification at `04-03-PLAN.md:146-155`, but the automated command at `04-03-PLAN.md:157-163` only runs `git diff --quiet`. An executor could skip the census entirely, leave no edits, and pass.

- **MEDIUM — The plan says it verifies README, but it does not.**  
  The must-have at `04-03-PLAN.md:23` says this plan verifies Plan 04-01's README rows. README is absent from the task's `read_first`, verify commands, and acceptance criteria. This matters because Plan 04-01's README outcome remains comparative.

- **MEDIUM — The phrase "byte-for-byte coverage contract" is overstated.**  
  The test compares parsed table values and checks that each Go `Gaps` string occurs somewhere in the document; it does not prove every document gap bullet is machine-backed or that all relevant prose is byte-identical. The mechanism at `internal/indexer/capability/matrix_test.go:104-162` is strong but narrower than the plan's description at `04-03-PLAN.md:32,118,125-126`.

- **MEDIUM — The census vocabulary is not reproduced as an executable, reviewed command.**  
  The plan lists terms in prose at `04-03-PLAN.md:147`, but the phase verification at `04-03-PLAN.md:231-236` only runs narrow file-specific patterns. This makes repeatability dependent on executor judgment.

- **LOW — `git diff --quiet` assumes a clean baseline.**  
  In the current tree this is not a problem, but the mechanism cannot distinguish an executor edit from an unrelated pre-existing change. A baseline hash or pre-task status record would make the verify-only invariant more robust.

- **LOW — Scope presentation is inconsistent.**  
  `docs/CORPUS-MEASUREMENT.md` is included in the task and KEEP records, but it is not among DOCS-04's explicit remaining-doc list at `.planning/REQUIREMENTS.md:21`. The extra verification is harmless, but it should be labeled supplemental rather than requirement-critical.

### Suggestions

- Turn the prose census into an executable script or checked command and save its output in the summary.

- Add README to the phase-final verification, particularly the Performance and Status sections.

- Describe the matrix guard precisely:

  - exact axis equality for parsed table rows;
  - every descriptor gap must occur verbatim in the document;
  - separately review that no extra or modified prose changes capability meaning.

- Add a positive assertion for the rewritten CONTRIBUTING rationale, not only absence checks. For example, require the retained behavioral-contract concept and `.planning/` reference.

- Capture the verify-only baseline using `git diff --quiet HEAD -- <files>` before and after the task, or explicitly abort if those files are already dirty.

### Risk Assessment

**MEDIUM.** The intended edits are sound, but the manual census and cross-plan assertions are not mechanically enforced.

---

## Cross-Plan Concerns

- **HIGH — The three plans do not collectively achieve the literal Phase 4 goal.**  
  The roadmap says NOTICE has transcription plus one sentence "and nothing else" at `.planning/ROADMAP.md:207`, while Plan 04-01 mandates several other sections. It also says README contains no comparison framing at `.planning/ROADMAP.md:210`, while `README.md:160-164` remains a head-to-head multiplier table.

- **MEDIUM — There is no authoritative phase-final gate.**  
  All plans are Wave 1 with `depends_on: []` at each plan's lines 5-6. Parallel execution is safe for file ownership, but several truths are cross-plan:

  - NOTICE references are removed by 04-01.
  - CONTRIBUTING references are removed by 04-03.
  - 04-02 owns the full-tree deleted-artifact claim.
  - 04-03 claims to verify README but does not.

  A separate phase-final verification step should run only after all three complete.

- **MEDIUM — Exact-phrase absence checks are weaker than the stated term-by-term census.**  
  Proposed checks would accept synonyms and residual comparison concepts such as "divergences," multiplier claims, or newly phrased origin references.

- **LOW — Repeated full-suite executions are costly but not harmful.**  
  Nearly every task runs `CGO_ENABLED=1 go test -count=1 ./...`. Given that most changes are Markdown or comments, focused tests per task plus one final full-suite run would reduce execution time without reducing confidence.

## Overall Risk Assessment

**HIGH.**

The implementation work is mechanically low risk, but plan correctness is not. As written, all automated checks could pass while two explicit roadmap outcomes remain false. Before execution, reconcile the NOTICE contract, remove or defer the README benchmark comparison surface coherently, and add a single phase-final census/reference/test gate.

---

### Convergence verdict

**Cycle 1.** The plans are judged against the phase goal — origin acknowledged exactly once (legally, past tense); comparison framing gone everywhere else; `docs/FLAG-PARITY.md` + `internal/cli/flag_parity_test.go` deleted with every reference swept — under the settled context supplied for this cycle.

The two Codex HIGH findings are **settled-context false positives**, not plan defects:
- **NOTICE "and nothing else"** — the retained LICENSE-warning block (NOTICE:3-10), the transcription-fidelity guard (:48-50), and the Third-party dependencies section (:52-69) are explicitly settled retention (minimal attribution = MIT transcription + license blocks + third-party deps + one origin sentence). The ROADMAP's "and nothing else" is loose wording the locked decisions (D-01) refine; the plan implements D-01 exactly.
- **README multiplier table** — the head-to-head numbers table is explicitly settled as Phase-6 retirement scope ("the numbers table stays (Phase 6 retires it)"); this phase sweeps only framing words. Codex's reading of DOCS-01 as banning the table is out of scope.

The remaining Codex MEDIUM/LOW findings were evaluated one-by-one against the plans. Incorporated / not actionable: the manual term-by-term census is the phase's designed contract (04-CONTEXT D-04, VALIDATION manual-only rows) and the SUMMARY-record requirement already captures it; the "byte-for-byte" phrasing is conservative (a stricter-than-needed prohibition, safe); repeated `go test ./...` is a cost note, not a correctness gap; `CORPUS-MEASUREMENT.md` inclusion is harmless supplemental verification.

Actionable MEDIUM/LOW concerns that the latest PLAN.md files neither incorporate nor explicitly defer/reject:

1. **MEDIUM — 04-03 claims to verify README but no task includes it.** `04-03-PLAN.md:23` truth asserts "this plan verifies them" for README's DOCS-01 rows, but README appears in no task's `read_first`/`verify`/acceptance criteria. Either add README to 04-03 task 3's verify list or reword the truth to delegate verification to 04-01's own verify.
2. **MEDIUM — DOCS-02 sweep grep misses camelCase artifact names.** The automated verify and phase gate use `rg "FLAG-PARITY|flag_parity"`, which does not match `TestFlagParityDocCoversRegisteredFlags` (survives at `internal/mcp/tools_schema_drift_test.go:39`) or `flagParityDocPath`. The action enumerates the :39 edit, but a missed edit would pass both the plan verify and the phase gate. Broaden the pattern (e.g. `FlagParity`) or add a targeted positive-controlled grep over the two internal/mcp files.
3. **MEDIUM — No plan-owned executable phase-final gate.** Cross-plan truths (NOTICE/CONTRIBUTING sites swept by other plans; the full-tree deleted-artifact sweep "after all three plans land"; README rows) are asserted in prose and in VALIDATION.md but are not installed as a phase-final verification step any plan owns. VALIDATION.md defines the gate ("Before /gsd-verify-work: full suite green, rg 'FLAG-PARITY' (harness scope) empty, gh api ... MIT live") — the plans should reference it as an explicit phase-final task, or add a thin verify-only task.
4. **LOW — 04-01 Status rewrite may retain "known divergences".** The discretionary Status text at `04-01-PLAN.md:124` keeps "known divergences are logged, not hidden"; the word is comparison-flavored and the automated verify won't catch it. Add "divergence" to the drop list or offer a fully own-terms alternative.
5. **LOW — ATTR-03 automated verify has no local LICENSE byte-identity check.** `gh api` reads hosted repo state, not the working tree; the plan's acceptance criterion is a manual `git diff -- NOTICE` review. Add `git diff --exit-code HEAD -- LICENSE` to the automated verify for a mechanical pre-push guarantee.

**Verdict: PLANS APPROVED FOR EXECUTION with the 5 actionable non-HIGH items above.** No HIGH concerns remain unresolved. Items 1-3 are recommended for incorporation via `/gsd-plan-phase 4 --reviews` before or during execution; items 4-5 are optional tightenings that do not block.

CYCLE_SUMMARY: current_high=0 current_actionable=5

---

# Cross-AI Plan Review — Phase 4 (Cycle 2)

---
reviewers: [codex]
reviewed_at: 2026-08-15T16:28:07Z
plans_reviewed: [04-01-PLAN.md, 04-02-PLAN.md, 04-03-PLAN.md]
cycle: 2
---

## Cycle 2 — Convergence Review

Cycle 1 (commit `36748f0`) raised 0 HIGH and 5 actionable non-HIGH items. Plans were revised at `e471e72`. This cycle re-reviews the three PLAN.md files against the settled context and the revision's stated fixes: camelCase pattern broadening, a new phase-final gate task in 04-02 (wave 2), the wave restructure (04-02 moved to wave 2 with `depends_on: [04-01, 04-03]`), a "divergence" drop-list addition, a LICENSE byte-identity check, and a README single-ownership clarification.

### Cycle-1 Finding Resolution

| # | Cycle-1 finding (severity) | Status | Where incorporated |
|---|---------------------------|--------|--------------------|
| 1 | 04-03 claims to verify README but no task includes it (MEDIUM) | **RESOLVED** | `04-03-PLAN.md:23` — the truth now explicitly delegates README verification to 04-01 ("this plan does NOT claim to verify README and does NOT re-edit it"); the phase-final gate (04-02 task 3) re-asserts the README end-state. |
| 2 | DOCS-02 sweep grep misses camelCase artifact names (MEDIUM) | **RESOLVED for the specifically-named references** | `04-02-PLAN.md:153,156,179` — `TestFlagParityDocCoversRegisteredFlags` and `flagParityDocPath` (via `flagParity`) are now in the executable patterns and are verified caught against the live tree. A NEW residual spelling surfaced this cycle — see New Finding 1 below. |
| 3 | No plan-owned executable phase-final gate (MEDIUM) | **RESOLVED** | `04-02-PLAN.md:165-204` — task 3 is an executable verify task with a positive-count 13/13 assertion (rule `84d1gfpywd`), runs in wave 2 after 04-01/04-03. Count math verified: sweep(1) + NOTICE greps(4) + README greps(3) + CONTRIBUTING(1) + matrix(1) + LICENSE diff(1) + live license(1) + full suite(1) = 13. |
| 4 | 04-01 Status rewrite may retain "known divergences" (LOW) | **RESOLVED** | `04-01-PLAN.md:124,129,134` and `04-02-PLAN.md:182` — "divergence" added to the README drop grep; the action mandates a fully own-terms alternative. |
| 5 | ATTR-03 has no local LICENSE byte-identity check (LOW) | **RESOLVED** | `04-01-PLAN.md:24,94,102` and `04-02-PLAN.md:185,191` — `git diff --exit-code HEAD -- LICENSE` added to the task-1 verify and the gate, with the gh-api-vs-working-tree distinction stated explicitly. |

### Wave Restructure & Phase-Final Gate Review

The wave metadata is internally consistent with the execution machinery:

- 04-01 (wave 1, `depends_on: []`), 04-03 (wave 1, `depends_on: []`), 04-02 (wave 2, `depends_on: [04-01, 04-03]`).
- No `files_modified` overlap within wave 1 (NOTICE/README vs CONTRIBUTING/matrix), and 04-02's files (deletion + `.github/*` + `internal/mcp/*` + `man.go`) do not overlap either wave-1 plan.
- The GSD execute-phase workflow executes waves sequentially (the wave-safety check blocks wave 2 while any lower wave is incomplete) and merges each wave's worktrees to main before the next wave forks — so 04-02's task 2 full-tree sweep and task 3 gate run against a tree that already contains the 04-01 NOTICE/README and 04-03 CONTRIBUTING/matrix end-states. The cross-plan read of those files by the gate is therefore sound.
- The 13/13 positive-count gate is fail-closed: each of the 13 assertions increments a counter, the script prints "phase-final gate: N/13 assertions passed", and exits non-zero unless N == 13. A gate that checked nothing cannot report success.
- Census accuracy spot-verified against the live tree: NOTICE warning block (:3-10), fidelity guard (:48-50), deps (:52-69), README migrate row (:137), Full TSX row (:151), License section (:218-220), CONTRIBUTING parity count (2, matching the :93 positive control), README parity count (1, at :189).

### New Findings (introduced or surfaced by this revision)

1. **MEDIUM — the broadened identifier family omits the hyphenated `flag-parity` spelling from all three executable greps.** `04-02-PLAN.md` must_haves truth (:30) enumerates the full family as `FLAG-PARITY | flag_parity | flag-parity | flagParityDocPath | TestFlagParityDocCoversRegisteredFlags`, and the task-2 action (:136, edit #6) explicitly targets the phrase "flag-parity test" at `internal/mcp/instructions_contract_test.go:375` ("internal/cli's flag-parity test could only verify by hand"). But the three executable patterns — task-2 action sweep (:150), task-2 verify (:153), and the phase-final gate full-tree sweep (:179/:191) — are all `FLAG-PARITY|flag_parity|flagParity|TestFlagParityDocCoversRegisteredFlags`; none include `flag-parity`. Verified against the live tree: the :375 line is NOT matched by that pattern set. A missed edit #6 would therefore pass both the task-2 verify and the phase-final gate — directly contradicting the plan's own authority contract ("the positive-controlled grep is the authority", :31) and its done-criterion ("no mention of either deleted artifact in ANY identifier spelling", :162). **Fix:** add `-e 'flag-parity'` to the three patterns (safe — the only surviving hyphenated hit is the :375 comment being removed).
2. **LOW — README License clause: the verify/gate hard-code a phrase the action explicitly leaves discretionary.** `04-01-PLAN.md:119` says "(exact wording at your discretion, but it must be one clause, past tense, and name NOTICE)", yet the task-2 verify (:129) and the phase-final gate (`04-02-PLAN.md:182`) require `rg -c "began as a Go rewrite of CodeGraph" README.md == 1`. A valid discretionary alternative wording (as explicitly permitted) would false-red both. **Fix:** either mandate the exact clause or loosen the grep (e.g., `rg -c "Go rewrite of CodeGraph" README.md == 1`).
3. **LOW — README Status "parity" drop is phrase-scoped, not term-scoped.** The task-2 action (:124) requires the Status rewrite to "drop 'parity'/…", and README's only current `parity` occurrence is the stale :189 claim being removed — so a term-scoped grep is safe. But the task-2 verify (:129) and the gate (:182) only check the exact phrase "Behavioral parity". A newly-worded Status that retains "parity" in another phrase would pass both. **Fix:** add `\bparity\b` to the README drop grep (mirrors the gate's already-term-scoped CONTRIBUTING grep at :183).

### Convergence verdict

**Cycle 2.** The revision resolves 4 of 5 cycle-1 actionable items in full and resolves cycle-1 finding 2 for the specifically-named camelCase references. The wave restructure and the new 13/13 phase-final gate are internally consistent with the GSD execution machinery (sequential waves, merge-before-fork), and the gate's assertion math checks out. No HIGH concerns remain unresolved; the two cycle-1 HIGHs remain adjudicated settled-context false positives (NOTICE license-block retention and the README multiplier table are locked decisions / Phase-6 scope), and no new HIGH surfaced.

Three actionable non-HIGH items remain, all tightenings to executable grep patterns: the hyphenated `flag-parity` spelling is missing from all three DOCS-02 sweep patterns (MEDIUM — the only substantive one), the README License clause verify is over-specific relative to the action's granted discretion (LOW), and the README Status "parity" drop is phrase-scoped rather than term-scoped (LOW).

**Verdict: PLANS APPROVED FOR EXECUTION with the 3 actionable non-HIGH items above.** Item 1 (add `flag-parity` to the three sweep patterns) is recommended for incorporation before execution; items 2-3 are optional tightenings that do not block.

CYCLE_SUMMARY: current_high=0 current_actionable=3

---

# Cross-AI Plan Review — Phase 4 (Cycle 3)

---
reviewers: [codex]
reviewed_at: 2026-08-15T17:02:39Z
plans_reviewed: [04-01-PLAN.md, 04-02-PLAN.md, 04-03-PLAN.md]
cycle: 3
---

## Cycle 3 — Final Convergence Review

Cycle 2 (commit `5524e9f`) raised 0 HIGH and 3 actionable non-HIGH items, all tightenings to executable grep patterns. Plans were revised at `1b584df`. This final cycle re-verifies that the 3 fixes landed correctly, that the exact-text License mandate agrees verbatim with its own grep (and with D-02), and that the fixes introduced nothing new. The phase proceeds to execution after this review.

### Cycle-2 Finding Resolution

| # | Cycle-2 finding (severity) | Status | Verification |
|---|---------------------------|--------|--------------|
| 1 | Hyphenated `flag-parity` missing from all three executable sweep patterns (MEDIUM) | **RESOLVED** | `04-02-PLAN.md:149,150,153,156,179,191,260` — every executable family pattern now includes `-e 'flag-parity'`. Verified the surviving `internal/mcp/instructions_contract_test.go:375` comment ("internal/cli's flag-parity test could only verify by hand") is the only in-tree hyphenated occurrence and is exactly edit #6's target (`:143`); the full-tree sweep's recorded `.planning/**` exclusion means the plans' own prose mentions cannot false-red. A missed edit #6 would now fail both the task-2 verify and the phase-final gate. |
| 2 | README License clause verify/gate hard-code a phrase the action left discretionary (LOW) | **RESOLVED** | `04-01-PLAN.md:119` now mandates the EXACT clause, discretion removed. Red-wall check passed: the mandated text "This project began as a Go rewrite of CodeGraph; see [NOTICE](NOTICE)…" contains the grep `began as a Go rewrite of CodeGraph` verbatim, and D-02 (`04-CONTEXT.md:24`) records the same clause. Mandate, grep, and decision agree verbatim — no disagreement between the exact-text mandate and its own grep. |
| 3 | README Status "parity" drop is phrase-scoped, not term-scoped (LOW) | **RESOLVED** | `04-01-PLAN.md:129,134` and `04-02-PLAN.md:182,191` — the README drop grep is now term-scoped `\bparity\b`. Verified safe against the live tree: README's only word-boundary `parity` is the stale `:189` "Behavioral parity …" claim the Status rewrite removes; no other `\bparity\b` match exists in README. |

### Regression check — did the fixes introduce anything new?

- **Gate integrity preserved.** The 13/13 positive-count gate (rule `84d1gfpywd`) still totals 13: sweep(1) + NOTICE(4) + README(3) + CONTRIBUTING(1) + matrix(1) + LICENSE diff(1) + live license(1) + full suite(1). The verify script at `04-02-PLAN.md:191` contains exactly 13 counter increments. The `flag-parity` broadening adds terms to the same single sweep assertion, not new assertions.
- **Cross-file grep collisions:** the README License-clause grep (`began as a Go rewrite of CodeGraph`, scoped to `README.md`) and the NOTICE origin grep (`began as a ground-up Go rewrite`, scoped to `NOTICE`) are distinct phrases on distinct files; no overlap. Both mandates match their greps verbatim — the NOTICE origin sentence mandate in 04-01 task 1 equals D-01's recorded sentence (`04-CONTEXT.md:22`), so the same red-wall standard holds for NOTICE too.
- **No residual old patterns:** no executable family pattern remains that lacks `flag-parity`; the only remaining "Behavioral parity" strings in the plans are quotations of the stale README text being removed (`04-01-PLAN.md:25,:124`), not requirements.
- **Wave structure unchanged** by `1b584df` (grep/mandate text only): 04-01/04-03 wave 1 (`depends_on: []`), 04-02 wave 2 (`depends_on: [04-01, 04-03]`), so the phase-final gate still runs after the cross-plan end-states land.

### Convergence verdict

**Cycle 3 (final).** All three cycle-2 actionable items are verified resolved in the on-disk plans at `1b584df`: the hyphenated `flag-parity` spelling is in every executable sweep pattern; the README License clause is now an exact mandate that agrees verbatim with its own grep and with D-02 (no red wall — and the NOTICE origin sentence mandate passes the same verbatim check); the Status "parity" drop is term-scoped `\bparity\b`, verified safe against the live tree. The fixes introduced no new issues — the 13/13 gate count, wave ordering, and cross-plan grep scoping are all intact. Both prior-Codex-HIGH adjudications remain settled-context false positives (NOTICE license-block retention and the README multiplier table are locked decisions / Phase-6 scope) and are not re-counted.

**Verdict: PLANS APPROVED FOR EXECUTION.** No HIGH concerns remain unresolved; no actionable MEDIUM/LOW concerns remain — all three cycle-2 items are incorporated. Convergence complete; the phase proceeds to execution.

CYCLE_SUMMARY: current_high=0 current_actionable=0
