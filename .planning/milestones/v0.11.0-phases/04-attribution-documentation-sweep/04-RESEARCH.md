# Phase 4: Attribution & Documentation Sweep - Research

**Researched:** 2026-08-15
**Domain:** Documentation sweep / attribution law / in-tree reference blast radius
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

This section is copied verbatim from `04-CONTEXT.md`. The planner MUST honor these as locked decisions.

### Locked Decisions

- **D-01:** The retained attribution is **minimal**. `NOTICE` keeps the verbatim MIT copyright transcription for CodeGraph, plus **one sentence of origin**: "CodeGraph Go began as a ground-up Go rewrite of CodeGraph (MIT)." No rationale, no comparison, no "originally based on / ported from / drop-in / ported-heuristics / flag-parity" argument — that is removed. — **Reversibility:** reversible — a NOTICE edit.
- **D-02:** `README`'s only origin mention is **one past-tense clause inside `## License`** that links to `NOTICE` — "This project began as a Go rewrite of CodeGraph; see NOTICE for the original copyright." The `## Relationship to the original` section is removed entirely. — **Reversibility:** reversible — a README edit.
- **D-03:** Delete `docs/FLAG-PARITY.md` and `internal/cli/flag_parity_test.go`, and **sweep every reference to either** — CI steps, Taskfile targets, other tests, docs that link to `docs/FLAG-PARITY.md`. `go test ./...` must still pass and nothing in the tree may reference either. The deletion removes a live drift guard; the replacement (`DOCS-05`, a self-authored CLI reference with its own guard) is deliberately deferred per the ROADMAP note. **This is a knowing, recorded reduction in flag-documentation coverage, not an oversight** — recorded in the plan. — **Reversibility:** costly — recreates no coverage; DOCS-05 must be authored to restore it.
- **D-04:** The comparison-vocabulary set is **removed term-by-term with recorded reasons, never by regex**: "parity", "the original", "upstream", "drop-in", "head-to-head", "vs TS", "based on", "ported from". — **Reversibility:** reversible — doc edits.
- **D-05:** The following are **product truth, kept**: `tsextract` (a real package name for TypeScript-the-indexed-language), `codegraph migrate` (a real capability), TypeScript/JavaScript as indexed languages in the capability matrix, and the `began as a Go rewrite` past-tense clause in `NOTICE`/`## License` (D-01/D-02). Removing these would remove capability, which the milestone explicitly forbids ("the sweep removes framing, never capability"). Each borderline reference is resolved individually with a recorded reason.

### Claude's Discretion

- The exact sentence structure of the one-sentence origin clause, provided it is minimal, past-tense, and carries no comparison argument (D-01/D-02 record the content rule).
- The exact ordering of the docs/* edits within the sweep (README, CONTRIBUTING, SECURITY, CODE_OF_CONDUCT, PARSER-DECISION, the remaining docs/*).
- Whether a borderline reference (e.g. a historical note in a doc body) is framed as past-tense-origin (keep, rewrite to past tense) or comparison (remove), recorded per instance with a reason.

### Deferred Ideas (OUT OF SCOPE)

- `docs/CLI-REFERENCE.md` (DOCS-05) — the self-authored replacement for FLAG-PARITY, with its own drift guard. A later milestone authors it; this one only deletes.
- `docs/BENCHMARKS.md` rewrite — Phase 6 (coupled to the comparison runner removal).
- Any rewording of `tools/bench/realcorpus` package docs (the PERF-01 head-to-head framing) — Phase 6.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ATTR-01 | NOTICE trimmed to MIT copyright transcription + one sentence of origin; drop-in/ported-heuristics/flag-parity argument removed | § Current State (NOTICE); full current content at `NOTICE:1-70`; retention question for the LICENSE-warning block and Third-party deps section in Open Questions |
| ATTR-02 | README's `## Relationship to the original` gone; only origin mention is one past-tense clause inside `## License` linking to NOTICE | § Current State (README); exact current text of both sections quoted |
| ATTR-03 | LICENSE stays verbatim MIT; `gh api .../license` reports MIT, verified live | § LICENSE Verification; **run live this session: returned `spdx_id: MIT`** |
| DOCS-01 | No comparison framing in README/CONTRIBUTING/SECURITY/CODE_OF_CONDUCT/PARSER-DECISION | § The Comparison-Framing Vocabulary Census — the planner's work list |
| DOCS-02 | Delete `docs/FLAG-PARITY.md` + `internal/cli/flag_parity_test.go` and all references; `go test ./...` passes | § DOCS-02 Deletion Blast Radius — every in-scope reference enumerated by file:line; baseline `go test ./...` = 49 packages ok |
| DOCS-03 | `docs/LANGUAGE-CAPABILITY-MATRIX.md` states capability on its own terms, no reference to another implementation's coverage | § Capability Matrix State — 4 stale/retired terms identified, all else already self-authored |
| DOCS-04 | Remaining docs/* carry no retired framing | § Census rows for `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`, `docs/MCP-2026-07-28-SCOPING.md`, `docs/MCP-8-AGENT-AUDIT.md`, `docs/CORPUS-MEASUREMENT.md` — two KEEP-with-reason lines, everything else clean |
</phase_requirements>

## Summary

Phase 4 is a documentation sweep with no runtime-architecture surface and no new dependencies. The complete blast radius has been enumerated this session by reading every attribution surface in full and running a term census over every in-scope document with word-boundary-aware patterns (a loose `port`-matching regex produces false positives on "report"/"export"/"supported" — the precise census avoids them).

**The NOTICE attribution argument** (drop-in, ported-heuristics, flag-parity, benchmark-reference) lives at `NOTICE:14-37`; the MIT copyright transcription that survives is at `NOTICE:39-46`. **README's `## Relationship to the original`** is `README:193-202`, and the one-clause `## License` is `README:218-220`. **The DOCS-02 deletion** touches 10 in-scope reference sites beyond the two deleted files themselves; critically, **no Taskfile target and no CI workflow references either deleted artifact** (verified by targeted rg), so `go test ./...` and `TestWorkflowRunBodiesInvokeTask` survive the deletion — confirmed against a live green baseline (49 packages ok, run this session). **ATTR-03's live check returns `MIT`** via `gh api repos/seanb4t/codegraph-go/license`.

The comparison-vocabulary census found real framing in **4 files only**: README (Performance framing, Status "parity" claim, the Relationship section), CONTRIBUTING (the Issue-first "ports observable behavior / parity" paragraph + two stale harness descriptions), the capability matrix (stale "golden-parity"/"reference implementation"/`weft` prose), and `docs/RELEASE-PROCEDURES.md` (one branch-name and one process-"upstream" line, both KEEP-with-reason). SECURITY, CODE_OF_CONDUCT, PARSER-DECISION, the MCP docs, `docs/RELEASE.md`, and `docs/CORPUS-MEASUREMENT.md` are clean.

**Primary recommendation:** run the sweep as three separate reviewed diffs (attribution edits, FLAG-PARITY deletion, doc sweep — per the CONTEXT's "one named cause per diff"), sweep the vocabulary term-by-term with a recorded reason per instance (never by regex), and scope the DOCS-02 "nothing references either" verification with explicit recorded exclusions for `.planning/`, `graphify-out/`, and `.claude/worktrees/`.

## Architectural Responsibility Map

This phase manipulates documentation surfaces, not runtime tiers; the standard Browser/API/DB tier map does not apply. The table below maps each capability to the file surface that owns it, which is what the planner actually assigns tasks against.

| Capability | Owning Surface | Requirement | Rationale |
|------------|---------------|-------------|-----------|
| Attributed origin (legal) | `NOTICE` | ATTR-01 | The single legal acknowledgment point; README links here |
| README origin mention | `README.md` (`## License`) | ATTR-02 | One past-tense clause, link to NOTICE; Relationship section deleted |
| Verbatim MIT license file | `LICENSE` | ATTR-03 | Never edited; verified live against GitHub licensee |
| No comparison framing | `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `PARSER-DECISION.md` | DOCS-01 | All five read in full this session; SECURITY/CoC/PARSER-DECISION clean |
| Flag matrix + drift guard deletion | `docs/FLAG-PARITY.md`, `internal/cli/flag_parity_test.go` + 10 reference sites | DOCS-02 | Complete blast radius in § DOCS-02 |
| Capability on own terms | `docs/LANGUAGE-CAPABILITY-MATRIX.md` | DOCS-03 | Phase 2 already re-synced gap prose (fail-loud); 4 stale terms remain |
| No retired framing | `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`, `docs/MCP-2026-07-28-SCOPING.md`, `docs/MCP-8-AGENT-AUDIT.md` | DOCS-04 | Two KEEP-with-reason lines; everything else clean |
| Live license detection | `gh api` (GitHub licensee) | ATTR-03 | Authoritative detector; project history records NOASSERTION when LICENSE was edited (NOTICE:3-10) |

## Current State: NOTICE (ATTR-01)

Read in full this session (`NOTICE:1-70`). Current structure:

- **`NOTICE:3-10`** — LICENSE warning block: "Do not move any of this into `LICENSE`." Documents the observed NOASSERTION regression. **Not comparison framing — see Open Question 1.**
- **`NOTICE:12-50` — `## Attribution`** — this is what ATTR-01 trims. Current content:
  - `NOTICE:14-16`: "`codegraph-go` is an independent, ground-up Go reimplementation of **CodeGraph**: https://github.com/colbymchenry/codegraph" — origin statement, **superseded by the one-sentence clause**.
  - `NOTICE:18-19`: "CodeGraph is a TypeScript project distributed under the MIT License. This **port** reimplements its behavior in Go; it does not vendor, copy, or link its source." — **REMOVE** (ported-framing).
  - `NOTICE:21-30`: "The debt is nonetheless substantial and specific. This project deliberately reproduces CodeGraph's observable behavior so that it can be a **drop-in replacement**:" + the four bullets (CLI surface "audited in `docs/FLAG-PARITY.md`" at line 25, MCP tools, "explore ranking heuristics and the `node` disambiguation rules, **ported** against frozen goldens captured from CodeGraph v1.3.1" at 27-29, "`.codegraph/` project layout"). — **REMOVE** (the drop-in/ported-heuristics/flag-parity argument; also removes the two `docs/FLAG-PARITY.md` references at lines 25 and 33).
  - `NOTICE:32-33`: "Behavioral divergences that were found and kept are documented rather than hidden — see `docs/FLAG-PARITY.md` and `docs/LANGUAGE-CAPABILITY-MATRIX.md`." — **REMOVE** (carries the second FLAG-PARITY reference).
  - `NOTICE:35-37`: "CodeGraph v1.3.1 is also the reference implementation benchmarked against in `docs/BENCHMARKS.md`..." — **REMOVE** (benchmark comparison framing; BENCHMARKS is Phase 6 anyway).
  - `NOTICE:39-46`: the retained MIT transcription, verbatim:
    ```
    MIT License

    Copyright (c) 2026 Colby Mchenry
    ```
  - `NOTICE:48-50`: "Do not 'correct' the capitalization of that name..." — transcription-fidelity guard. **Not framing; see Open Question 1** (whether it counts as retained "rationale").
- **`NOTICE:52-69` — `## Third-party dependencies`** — SBOM/disclosure section (tree-sitter, Pebble, go-sdk, cobra; references `PARSER-DECISION.md`). **Not comparison framing — see Open Question 1.**

**The exact retained content rule (D-01):** the transcription plus exactly one sentence: "CodeGraph Go began as a ground-up Go rewrite of CodeGraph (MIT)." The URL to `https://github.com/colbymchenry/codegraph` may be folded into that sentence at Claude's discretion (D-04 clause + discretion bullet 1 — the sentence must remain one sentence).

## Current State: README (ATTR-02, DOCS-01)

Read in full this session. The two sections the planner edits:

**`README:193-202` — `## Relationship to the original`** (removed entirely by ATTR-02). Exact current text:

```markdown
## Relationship to the original

This is an independent Go reimplementation of
[CodeGraph](https://github.com/colbymchenry/codegraph) (TypeScript, MIT). It
reproduces that project's CLI semantics, MCP tools, and ranking behavior closely
enough to be a drop-in replacement, and `codegraph migrate` imports an existing
TypeScript index.

The original is the reason this project has a specification to hit at all. See
[NOTICE](NOTICE) for attribution.
```

**`README:218-220` — `## License`** (gets the one past-tense clause). Exact current text:

```markdown
## License

MIT — see [LICENSE](LICENSE).
```

ATTR-02 replaces the middle with a link to NOTICE plus the one clause, e.g. "MIT — see [LICENSE](LICENSE). This project began as a Go rewrite of CodeGraph; see [NOTICE](NOTICE) for the original copyright." (exact wording at Claude's discretion).

## The Comparison-Framing Vocabulary Census (DOCS-01/03/04)

The term sweep named by D-04 ("parity", "the original", "upstream", "drop-in", "head-to-head", "vs TS", "based on", "ported from") plus derived origin vocabulary ("reimplementation", "port(s)", "TS original", "reference implementation", "derived from"), run with word-boundary-aware patterns over every in-scope surface. **This table is the planner's work list.** Every row is `[VERIFIED: read this session]` — the quoted text was read and the governing term matched against D-04's set.

| File:Line | Exact current text (governing term) | Classification | Action |
|-----------|-------------------------------------|----------------|--------|
| `README.md:157` | "Measured against the TypeScript implementation it ports — median-of-3, on three real repositories, on the same machine:" | COMPARISON (ports / against-TS) | Rewrite to absolute: "Median-of-3, on three real repositories, on the same machine:" — numbers table stays (Phase 6 retires it) |
| `README.md:163` | "\| `codegraph` (the TS original) \| 4.3× faster \| ..." | COMPARISON ("the TS original" — the *original* positioning DOCS-01 bans) | Retitle row "`codegraph` (TypeScript)" — the repo identity is product truth, the *original* positioning is not |
| `README.md:166-167` | "so treat any single number with suspicion, including the flattering one." | BORDERLINE (implies the flattery is vs-TS) | Recommend dropping "including the flattering one"; record the reason |
| `README.md:189` | "Behavioral parity with TypeScript CodeGraph v1.3.x is covered by a fixture harness that diffs against frozen goldens; known divergences are logged, not hidden." | COMPARISON ("parity") **and STALE** — FIXT-04/06 removed the TS-diffed capture path; the harness runs the production indexer over `corpus/behavioral` and asserts named behavioral properties (`testdata/golden/behavioral_test.go:1-18`) | Rewrite on codegraph-go's own terms (describe the fail-loud property-test harness); "parity with TypeScript CodeGraph v1.3.x" gone |
| `README.md:193-202` | `## Relationship to the original` — quoted in full above | REMOVE (ATTR-02) | Delete section; `codegraph migrate` remains documented at `README:137` |
| `CONTRIBUTING.md:11-14` | "This is not bureaucracy for its own sake: this project ports observable behavior from another implementation, and a change that looks like an obvious improvement is sometimes a deliberate parity decision recorded in `docs/FLAG-PARITY.md` or `.planning/`. The issue is where we find that out before you've written the code." | COMPARISON (ports / another implementation / parity) **and a DOCS-02 dangling reference** at line 13 | Rewrite the Issue-first rationale on own terms; drop the FLAG-PARITY link (keep "or `.planning/`" if desired) |
| `CONTRIBUTING.md:83` | "`task test` covers every host-only leg — unit, golden parity, subprocess integration, isolated daemon, and race" | STALE NAME ("golden parity" — the golden test files are now `behavioral_*.go`, renamed by CODE-02) | Reword to "unit, golden," |
| `CONTRIBUTING.md:191` | "golden-corpus behavioral fixtures diffed against frozen TypeScript CodeGraph v1.3.1 output" | COMPARISON **and STALE** — goldens no longer diff against TS v1.3.1 output (`behavioral_test.go:16-18` says the TS-era capture path "is gone") | Rewrite to describe the current fail-loud golden suite |
| `docs/LANGUAGE-CAPABILITY-MATRIX.md:13` | "full = validated end-to-end (priority-4 via a corresponding golden-parity test in `testdata/golden/`..." | STALE NAME ("golden-parity" — the tests are `behavioral_csharp_test.go` etc., `TestCorpusBehavior_*` functions) | Reword to "golden corpus behavioral test" |
| `docs/LANGUAGE-CAPABILITY-MATRIX.md:59-61` | "No named gaps — Go is the reference implementation, validated end-to-end since Phase 2/3 against the pinned `weft` golden-parity corpus (`testdata/golden/behavioral_test.go#TestCorpusBehavior_Go`)." | COMPARISON ("reference implementation" positioning of Go-vs-{TS|other languages}) **and STALE** — the `weft` corpus is gone (FIXT-04); `TestCorpusBehavior_Go` runs over `corpus/behavioral` (`behavioral_test.go:684-692`) | Rewrite on own terms: Go is the first-class language for the in-repo behavioral corpus; drop `weft` and "golden-parity" |
| `docs/LANGUAGE-CAPABILITY-MATRIX.md:181` | "05-RESEARCH.md's original `fwcd` recommendation because the originally-approved source failed to build" | KEEP — "original" here means *the earlier recommendation in a research doc*, not CodeGraph-the-original | Record the reason; no edit |
| `docs/LANGUAGE-CAPABILITY-MATRIX.md:192` | "golden-parity/self-consistency tests do prove resolves" | STALE NAME ("golden-parity") | Reword to "golden/self-consistency tests" |
| `docs/RELEASE-PROCEDURES.md:60` | "(`gsd/v1.0-drop-in-parity-human-ux` for this milestone)." | KEEP — quoted historical branch name (proper noun); rewording would falsify the record | Record the reason; no edit |
| `docs/RELEASE-PROCEDURES.md:581` | "this step can fail — nothing upstream of it was skipped." | KEEP — "upstream" in the process/git sense, not comparison framing | Record the reason; no edit |
| `docs/RELEASE.md:345` | "covers TS/JSX parsing" | KEEP — TypeScript-the-indexed-language grammar coverage (product truth, D-05) | Record the reason; no edit |
| `docs/CORPUS-MEASUREMENT.md:121,198` | "typescript: 1684" / "TS/JS is supplied by gohugoio/hugo's javascript files" | KEEP — TypeScript/JavaScript-as-indexed-languages corpus data (product truth, D-05) | Record the reason; no edit |
| `SECURITY.md` | (all matched lines are "report"/"reported"/"derived from the workflow" — false positives) | CLEAN | No edits |
| `CODE_OF_CONDUCT.md` | (matched lines are "reported" — false positives) | CLEAN | No edits |
| `PARSER-DECISION.md` | (tight regex: zero hits) | CLEAN | No edits |
| `docs/MCP-2026-07-28-SCOPING.md` | (all hits are `codegraph-go`/`codegraph_*` product references) | CLEAN | No edits |
| `docs/MCP-8-AGENT-AUDIT.md` | (all hits are `codegraph` binary/tool references) | CLEAN | No edits |
| `docs/BENCHMARKS.md` | — | OUT OF SCOPE — Phase 6 | Do not touch |
| `docs/CLI-REFERENCE.md` | does not exist (verified `ls`) | OUT OF SCOPE — DOCS-05, later milestone | Do not touch |

## DOCS-02 Deletion Blast Radius

Both deletion targets and every in-tree reference, verified this session. **Nothing else in the tree references either artifact** — the enumerated rows below are exhaustive.

**The two deleted files:**
- `docs/FLAG-PARITY.md` — the SURF-05 per-command TS 1.3.1 vs Go flag/default/status matrix (320 lines, read in full). Fully comparison-framed by construction; deleted wholesale.
- `internal/cli/flag_parity_test.go` — `TestFlagParityDocCoversRegisteredFlags` (read in full, 71 lines). Self-contained: reads `flagParityDocPath = "../../docs/FLAG-PARITY.md"`, walks `newRootCmd()`. **No other test imports it, calls it, or shares its constants.**

**In-scope reference sites (sweep these):**

| # | File:Line | Text (governing reference) | Phase-4 action |
|---|-----------|---------------------------|----------------|
| 1 | `NOTICE:25` | "...audited in `docs/FLAG-PARITY.md`" | Removed by ATTR-01's rewriter |
| 2 | `NOTICE:33` | "see `docs/FLAG-PARITY.md` and `docs/LANGUAGE-CAPABILITY-MATRIX.md`" | Removed by ATTR-01's rewriter |
| 3 | `CONTRIBUTING.md:13` | "recorded in `docs/FLAG-PARITY.md` or `.planning/`" | Rewritten by DOCS-01 (same paragraph) |
| 4 | `.github/pull_request_template.md:37` | "sometimes deliberate parity decisions recorded in `docs/FLAG-PARITY.md` or `.planning/`" | Remove the link (mechanical); the *parity* vocabulary here is Phase 5 PROC-02 |
| 5 | `.github/PULL_REQUEST_TEMPLATE/feature.md:42` | "(`docs/FLAG-PARITY.md`, `docs/LANGUAGE-CAPABILITY-MATRIX.md`, README)" — a checklist line under "## Parity" | Remove the FLAG-PARITY item (mechanical); the `## Parity` heading + TS-check item are Phase 5 PROC-02 |
| 6 | `.github/ISSUE_TEMPLATE/enhancement.yml:77` | "I checked whether this is a documented divergence in docs/FLAG-PARITY.md" | Remove/reword the option label (mechanical link removal); Phase 5 PROC-01 owns the rest of this template |
| 7 | `.github/workflows/auto-close-unsolicited-prs.yml:91` | "...deliberate parity decisions recorded in `docs/FLAG-PARITY.md` or `.planning/`..." (comment-body string) | Remove the link (mechanical); Phase 5 PROC-03 owns the parity/ports vocabulary in this workflow (ROADMAP Phase 5 Notes explicitly lists `auto-close-unsolicited-prs`) |
| 8 | `internal/mcp/instructions_contract_test.go:16` | "Mirrors `internal/cli/flag_parity_test.go`'s flagParityDocPath convention..." | Reword to drop the deleted-file name; Phase 5 CODE-01 owns the comment's framing |
| 9 | `internal/mcp/instructions_contract_test.go:375` | "...the self-defeat property internal/cli's flag-parity test could only verify by hand." | Reword to drop the deleted-test reference; Phase 5 CODE-01 owns the rest |
| 10 | `internal/mcp/tools_schema_drift_test.go:36,39` | "...updated the CLI help and `docs/FLAG-PARITY.md`..." / "`internal/cli`'s TestFlagParityDocCoversRegisteredFlags structurally cannot catch this..." | Reword to drop the deleted artifacts' names; Phase 5 CODE-01 owns the framing |

**Recorded exclusions (NOT in scope — do not sweep, record in the plan):**
- `.planning/**` — append-only project history parsed by GSD tooling; the milestone explicitly rules it out (REQUIREMENTS.md → Out of Scope). Contains `docs/FLAG-PARITY.md` / `flag_parity_test.go` mentions in archived phase artifacts (`04-CONTEXT.md`, `ROADMAP.md`, `PROJECT.md`, archived DEBUG logs, phase SUMMARYs).
- `graphify-out/**` — gitignored generated index (verified: `.gitignore:29`; `git ls-files graphify-out/` empty).
- `.claude/worktrees/**` — untracked agent scratch worktrees (verified: `git ls-files .claude/worktrees/` empty).

**Breakage analysis (verified, not assumed):**
- **Taskfile.yml: zero references.** `rg -n "flag_parity|FLAG-PARITY|flag-parity" Taskfile.yml` → no hits. No Taskfile target invokes `-run FlagParity`.
- **CI workflows: the only reference is the auto-close comment string above.** No workflow step runs the deleted test, and `ci.yml` has no `-run` filter naming it.
- **`TestWorkflowRunBodiesInvokeTask`** (`internal/upgrade/taskfile_shape_test.go:1345`) is the D-01/D-02 guard that every CI step's `run:` body invokes a Taskfile target. It reads workflow YAML + Taskfile bodies; since neither references the deleted artifact, the test is unaffected by the deletion. **Its continued greenness was re-confirmed indirectly: `go test ./...` passed 49 packages this session with the guard present, and the deletion only removes a self-contained test file.**
- **Capability matrix code** (`internal/indexer/capability/matrix.go`, `matrix_test.go`) does **not** reference `flag_parity_test.go` or `FLAG-PARITY.md` (its "parity" hits are `golden-parity` in comments — Phase 5 CODE-01). **DOCS-03's doc-side reword does not break `matrix_test.go`**: the test asserts coverage values and gap bullets match the Go descriptor byte-for-byte, and no census row above edits a coverage value or a gap bullet.
- **`go test ./...` baseline:** CGO_ENABLED=1, run this session: 49 packages `ok`, zero failures.

## Product-Truth References That Must NOT Be Swept (D-05)

| Reference | Where (in-scope surfaces) | Why kept |
|-----------|--------------------------|----------|
| `codegraph migrate` (legacy-index import) | `README.md:137` (Commands table: "`migrate` \| import an existing TypeScript CodeGraph index"); also `README:198` (removed with the Relationship section — no action needed) | Real capability; the milestone forbids removing capability. Note `docs/RELEASE.md` and `docs/RELEASE-PROCEDURES.md` "migrated pipeline/release" hits are the *release-please pipeline* migration, a different sense — no edit |
| `tsextract` package | **No reference in any doc surface** (census over README/CONTRIBUTING/SECURITY/CoC/PARSER-DECISION/docs/* returned zero hits). Lives in code: `internal/indexer/tsextract/{tsextract,types,resolution,d09}_test.go`, `languages_typescript.go` | Product surface (TypeScript-the-indexed-language). The name is a real package path; a regex sweep over "TS" would break it. Phase 4 touches no code — a doc-only phase cannot harm it, and Phase 5 (CODE-01) must not rename it |
| TypeScript/JavaScript as indexed languages | `README.md:149-153` (Language support table "Full" rows), capability matrix TSX/TS/JS rows, `docs/RELEASE.md:345`, `docs/CORPUS-MEASUREMENT.md:121,198` | Indexed-language product truth; de-listing would remove capability |
| The `began as a Go rewrite` past-tense clause | `NOTICE` (D-01 sentence) + `README ## License` (D-02 clause) | The retained attribution — the only origin text allowed anywhere |
| `.planning/` mentions in sweepable docs (e.g. `CONTRIBUTING.md:13` "or `.planning/`") | CONTRIBUTING paragraph | Referencing `.planning/` as a place decisions are recorded is fine; `.planning/` is out-of-scope to *rewrite*, not to *mention* |

## Capability Matrix State (DOCS-03)

`docs/LANGUAGE-CAPABILITY-MATRIX.md` was read in full (202 lines). Phase 2's 02-03 already re-synced the gap prose to the fail-loud posture (the "FAILS LOUDLY on absence (02-03, D-10)" sentences at lines 66, 72, 79, 85, 91, 98). The matrix is **already self-authored in substance** — it never references another implementation's coverage, and its TypeScript/TSX/JS rows are indexed-language product truth.

What remains for DOCS-03 is the four stale/retired terms inventoried in the census: `:13` and `:192` ("golden-parity"), `:59-60` (the "reference implementation" positioning of Go + the stale `weft` corpus reference), and `:181` (KEEP — "original" = the earlier research recommendation). **The doc↔code parity contract is safe**: `matrix_test.go` asserts only coverage values and gap bullets match `matrix.go` exactly, and no DOCS-03 edit touches a value or gap bullet. The matching `golden-parity` *code comments* in `matrix.go:18,32` and `matrix_test.go:217,221` are Phase 5 (CODE-01) and are not part of this phase.

**Coordination note:** one residual identifier survives CODE-02 — `syntheticParitySrc` at `testdata/golden/behavioral_test.go:239`, called by `TestCorpusBehavior_Go` (line 690). It is code, not docs, so it is out of Phase 4 scope; flag it to Phase 5 (CODE-01 territory or a CODE-02 follow-up) rather than touching it from this phase.

## LICENSE Verification (ATTR-03)

- `LICENSE` read in full this session — verbatim MIT License text, 21 lines, "Copyright (c) 2026 Sean Brandt", no appended attribution block.
- **Live check executed this session (2026-08-15):**
  ```bash
  gh api repos/seanb4t/codegraph-go/license --jq '{spdx_id: .license.spdx_id, name: .license.name, path: .path}'
  # {"spdx_id":"MIT","name":"MIT License","path":"LICENSE"}
  ```
  Returns `spdx_id: MIT` — **ATTR-03 confirmed live, not assumed** (gh 2.97.0).
- The project records why LICENSE must never be edited: `NOTICE:3-10` documents that an earlier LICENSE revision carrying a trailing attribution block made the GitHub API report `NOASSERTION`. The ATTR-03 verification must be **re-run after** the NOTICE change lands (the requirement's "run live against the repository after the change, not assumed"), and the plan should re-run it at the phase gate.
- Instrument: `gh api repos/seanb4t/codegraph-go/license` (the `license` endpoint runs GitHub's own `licensee` detection server-side).

## Standard Stack

This phase installs **no external packages** — it is a markdown edit + file deletion + verification sweep. The "stack" is the existing toolchain, all verified present:

| Tool | Version | Purpose | Why standard |
|------|---------|---------|--------------|
| Go toolchain (`go`) | latest (present; tests ran) | `go test ./...`, `go vet` — the green baseline the deletion must preserve | Project's mandated stack |
| `gh` CLI | 2.97.0 | ATTR-03 live license detection | Authoritative GitHub `licensee` endpoint |
| `task` | 3.52.0 | `task test` etc. — the single definition of CI job bodies (standing decision) | Existing repo convention; `TestWorkflowRunBodiesInvokeTask` enforces it |
| `rg` | present | Positive-control greps for the census and the DOCS-02 "nothing references" gate | Repo's search standard; **must be used with positive controls** (standing decision: a transcript grep is a claim about the grep) |
| `git` | present | Diff review; one-named-cause commits | Repo discipline |

There is no `npm install` / `pip install` equivalent for this phase.

## Package Legitimacy Audit

**Not applicable.** Phase 4 installs no external packages (no `go.mod` changes, no new dependencies). The Package Legitimacy Gate protocol is skipped for this phase; nothing was added to the supply chain.

## Architecture Patterns

### System Sweep Flow

```
                    ┌─────────────────────────────────────────────────┐
                    │             SOURCE SURFACES (all in git)        │
                    │  NOTICE  README  CONTRIBUTING  SECURITY  CoC    │
                    │  PARSER-DECISION  docs/*.md  .github/*  code    │
                    └───┬───────────┬──────────────┬──────────────────┘
        ATTR-01   ──────┘           │              │
        ATTR-02   ──────────────────┘              │
        DOCS-01/03/04 ─────────────────────────────┘   (term-by-term, D-04)
        DOCS-02    ─── delete FLAG-PARITY.md + flag_parity_test.go
                        then sweep the 10 reference sites

                    ┌─────────────────────────────────────────────────┐
                    │                VERIFICATION GATES               │
                    │  1. go test ./...  (49 pkgs, currently green)   │
                    │  2. rg census, POSITIVE-CONTROLLED per term     │
                    │  3. rg "FLAG-PARITY|flag_parity" -- no in-scope │
                    │     hits outside recorded exclusions            │
                    │  4. gh api .../license -> MIT (after NOTICE edit)│
                    │  5. doc assertions: 1 origin in NOTICE, 1 clause│
                    │     in README License, no Relationship section  │
                    └─────────────────────────────────────────────────┘
```

### Pattern 1: Term-by-term sweep with recorded reasons (D-04)
**What:** Each vocabulary occurrence is individually classified (COMPARISON / STALE-NAME / BORDERLINE / KEEP-with-reason) before editing; never a global find-and-replace.
**When:** Every edit in DOCS-01/03/04. The census table in this research is the pre-made classification — the planner turns rows into tasks.
**Why:** A regex over "TypeScript" breaks `tsextract` and de-lists a supported language (named failure mode, ROADMAP Phase 5 Notes); a regex over "port" hits "report"/"export"/"supported".

### Pattern 2: Verified-stale rewrites, not just vocabulary swaps
**What:** Three census rows are not merely framing — they describe a harness that no longer exists (CONTRIBUTING:191 "diffed against frozen TypeScript CodeGraph v1.3.1 output"; matrix:60 "pinned weft corpus"; README:189 "Behavioral parity ... diffs against frozen goldens"). The goldens are now codegraph-go's own re-frozen output or assertion-based behavioral properties (FIXT-04/06, `behavioral_test.go:1-18`).
**When:** Those three rewrites must describe current reality or they leave a new lie in place of an old one.

### Anti-Patterns to Avoid
- **Global replace of "parity" or "original"**: breaks product surface; use the census.
- **Editing LICENSE** in any way: observed NOASSERTION regression in project history.
- **Bundling the three concerns** (attribution, deletion, sweep) into one diff: violates the repo's one-named-cause discipline.
- **A verification grep with no positive control**: a grep that returns zero hits proves nothing about the grep's ability to find the term (standing decision, STATE.md).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| License detection | A local scanner/regex over `LICENSE` | `gh api repos/seanb4t/codegraph-go/license` (`licensee` server-side) | The requirement's instrument is the GitHub API; a local guess is what the project already learned fails (NOASSERTION history) |
| Vocabulary drift guard | A term-blocklist CI check | Nothing — one-time sweep + review discipline (VOCAB-01 deliberately declined, REQUIREMENTS.md → v2) | The blocklist either goes vacuous or fights legitimate uses like `tsextract` (rule `84d1gfpywd` precedent) |
| A grep gate for "nothing references either" | A bare `rg` with no self-test | Positive control: first prove the grep finds the term in a known-containing fixture, then run the sweep | A transcript grep is a claim about the grep, not about the product — established twice in v0.10.0 Phase 6 rehearsal |

## Runtime State Inventory

This phase deletes two in-git files and edits in-git markdown; it does not rename or migrate runtime-identifying strings. Each category answered explicitly:

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | **None** — no database/datastore keys or collections embed `FLAG-PARITY`/`parity` names (Pebble keys are symbol/edge keys, not doc names; verified by reasoning over what the phase touches — nothing writes data) | none |
| Live service config | **None in this phase's scope.** The `.github` templates/workflows that reference `docs/FLAG-PARITY.md` are in-git tree state (lines inventoried above), not live service config; the only CI-registered artifact is the deleted drift guard, which has no workflow registration (verified: no workflow references it) | In-git edits per the blast-radius table |
| OS-registered state | **None** — no launchd/systemd/scheduler/task registrations embed either deleted name | none |
| Secrets/env vars | **None** — `gh api` needs only the ambient `gh` auth (already working this session); no env-var names change | none |
| Build artifacts | **None** — deleting a `_test.go` file produces no artifact; `go test ./...` recompiles cleanly (49 packages ok baseline) | none |

## Common Pitfalls

### Pitfall 1: The regex sweep subtracts capability instead of framing
**What goes wrong:** A find-and-replace over "TypeScript" or "TS" hits the `tsextract` package name, "TS/JS" indexed-language rows, and grammar coverage prose — capability removal, exactly what the milestone forbids.
**Why it happens:** D-04 forbids regex yet the temptation is a global replace for speed.
**How to avoid:** Work from the census table; every edit is a classified row.
**Warning signs:** A diff touching `TypeScript`/`TS` outside the census rows.

### Pitfall 2: LICENSE edited "just a little"
**What goes wrong:** An attribution paragraph appended to LICENSE downgrades GitHub detection to `NOASSERTION`.
**Why it happens:** The NOTICE:3-10 warning exists because an earlier revision did exactly this and the API reported `NOASSERTION` — observed, not theoretical.
**How to avoid:** Never touch LICENSE; verify `licenses.spdx_id` stays `MIT` via `gh api` after the NOTICE edit.
**Warning signs:** Any diff touching `LICENSE`.

### Pitfall 3: A leftover reference to the deleted artifacts breaks the "nothing in the tree" gate
**What goes wrong:** `rg "FLAG-PARITY" .` finds `.planning/` archives, `graphify-out/`, and `.claude/worktrees/`, and the verification fails on noise — or, worse, a real in-scope reference (the 10 rows above) is missed and the plan ships a dangling link.
**Why it happens:** Unscoped grep; the exclusions are not recorded.
**How to avoid:** Record the three exclusions in the plan's verification task and grep `--hidden --no-ignore -g '!.planning/**' -g '!graphify-out/**' -g '!.claude/**'`, positive-controlled.
**Warning signs:** A "nothing references either" task with no exclusion list.

### Pitfall 4: Rewriting framing but leaving stale harness prose
**What goes wrong:** "golden-parity corpus", "diffed against frozen TS v1.3.1 output", "pinned weft corpus" get their vocabulary swapped but keep describing a deleted harness — the doc invents history.
**Why it happens:** FIXT-02/04/06 changed the harness after these sentences were written; Phase 2 resynced gap prose but not every prose sentence.
**How to avoid:** The three stale rows (README:189, CONTRIBUTING:83/191, matrix:13/60/192) must describe the current fail-loud behavioral suite.
**Warning signs:** A rewrite that still names `weft` or "TS ... output" as the goldens' source.

## Code Examples

Verified patterns used or needed by this phase:

### ATTR-03 live license check (run this session, returns MIT)
```bash
gh api repos/seanb4t/codegraph-go/license --jq '.license.spdx_id'
# MIT
```
Source: the `license` REST endpoint's `license.spdx_id` field. Re-run at the phase gate after the NOTICE change.

### Positive-controlled "nothing references either" grep (DOCS-02)
```bash
# 1. Prove the grep can find the term: point it at a file known to contain it
rg -l "FLAG-PARITY" .github/pull_request_template.md        # must print the file
# 2. Run the real sweep with recorded exclusions (no in-scope hits expected)
rg -n "FLAG-PARITY|flag_parity" --hidden --no-ignore \
  -g '!.git/**' -g '!.codegraph/**' -g '!dist/**' \
  -g '!.planning/**' -g '!graphify-out/**' -g '!.claude/**' .
# Expect: zero hits after the sweep; the 10 inventoried sites edited/removed
```

### go baseline (already green: 49 ok)
```bash
CGO_ENABLED=1 go test ./...
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `NOTICE` carrying a drop-in/ported-heuristics/flag-parity attribution argument | `NOTICE` carrying transcription + one past-tense origin sentence | This phase (ATTR-01) | Origin acknowledged legally, exactly once |
| Goldens diffed against frozen TS CodeGraph v1.3.1 output | Goldens re-frozen from codegraph-go's own output (FIXT-06) + named-property behavioral assertions | Phases 1–3 (already landed) | Any prose still describing the TS-diff harness is stale — three census rows |
| `docs/FLAG-PARITY.md` matrix + `flag_parity_test.go` drift guard | Deleted; replacement `docs/CLI-REFERENCE.md` (DOCS-05) deferred | This phase (DOCS-02) | Recorded, knowing reduction in flag-documentation coverage |
| Head-to-head benchmark summary in README Performance | Absolute-figure publication (BENCH-01) | Phase 6 | Phase 4 sweeps only the framing words in README's Performance section; the numbers table is Phase 6's retirement |

**Deprecated/outdated:**
- "golden-parity" as a descriptor for `testdata/golden/` tests: the files are now `behavioral_*.go` / `TestCorpusBehavior_*` (CODE-02). Doc prose at matrix:13/60/192 and CONTRIBUTING:83 must drop the word; **code comments** carrying it (matrix.go:18,32, matrix_test.go:217,221, corpora/record.go:104, and the realcorpus manifest — the last is Phase 6) are Phase 5 CODE-01.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The ROADMAP's "and nothing else" (NOTICE criterion) means "no attribution argument" rather than "delete the Third-party dependencies section" | Current State: NOTICE | If the strict reading is intended, NOTICE loses its SBOM/disclosure section — a compliance reduction the milestone's "never capability" rule argues against. Planner should record the decision (see Open Question 1) |
| A2 | README's Performance-section framing words are DOCS-01 scope while the head-to-head numbers table is Phase 6's retirement | Census | If the boundary is drawn differently, either Phase 4 under-sweeps DOCS-01 or overlaps BENCH-01. Planner records the boundary (see Open Question 2) |
| A3 | The 3 `.github` template references + 1 workflow comment string + 3 internal/mcp comment lines are DOCS-02's "nothing in the tree references either" targets, with `.planning/`, `graphify-out/`, `.claude/worktrees/` as recorded exclusions | DOCS-02 Blast Radius | If the planner defers the .github/internal sites to Phase 5, the Phase-4 "nothing references either" criterion is not literally met at the Phase 4 gate. See Open Question 3 |

*All other claims in this research are `[VERIFIED]` — every file:line was read this session, the live `gh api` check was executed, and the `go test ./...` baseline was run.*

## Open Questions

1. **What exactly does "NOTICE ... and nothing else" trim?** (ROADMAP criterion 1)
   - What we know: D-01 mandates transcription + one origin sentence; the drop-in/ported/parity argument (NOTICE:14-37) is removed. The LICENSE warning block (NOTICE:3-10) and the `## Third-party dependencies` section (NOTICE:52-69) are not attribution framing and contain no comparison vocabulary.
   - What's unclear: whether they survive. The ROADMAP's literal "nothing else" could strip NOTICE to two sentences + transcription, deleting the SBOM disclosure.
   - Recommendation: keep both — "the sweep removes framing, never capability/disclosure" — and record the ROADMAP wording as scoped to the attribution argument. The warning block directly serves ATTR-03; the dependencies section is the SPDX/SBOM pointer.

2. **Is README's Performance section in DOCS-01 scope?**
   - What we know: DOCS-01 names README.md and bans "the original" positioning; the section's words ("it ports", "the TS original", "including the flattering one") are banned vocabulary, while the head-to-head table is BENCH-01's Phase-6 retirement.
   - What's unclear: whether Phase 4 should touch the section at all, given the phase boundary's "benchmark path" exclusion.
   - Recommendation: sweep the words only (rows README:157,163,166-167), leave the numbers table intact, and record the boundary. The exclusion in CONTEXT ("does NOT touch the benchmark path") names `docs/BENCHMARKS.md`, not README.

3. **Who sweeps the .github and internal/mcp references to the deleted artifacts?**
   - What we know: DOCS-02's criterion is "nothing in the tree references either"; 4 reference sites live in `.github/` (PROC-01/02/03 territory, Phase 5) and 3 comment lines live in `internal/mcp/` (CODE-01 territory, Phase 5); ROADMAP's Phase 5 Notes explicitly own `auto-close-unsolicited-prs` vocabulary.
   - What's unclear: whether Phase 4 (DOCS-02) performs mechanical link/name removal at those sites or defers the entire site to Phase 5.
   - Recommendation: Phase 4 removes the dangling links/names at all 7 sites (mechanical, one-line edits — it is the only way the literal criterion holds at the Phase 4 gate), Phase 5 rewrites the *framing* in the same lines. Record the two-phase same-line edit explicitly so the executors don't collide.

4. **The NOTICE capitalization guard (lines 48-50)** — keep (it protects the transcription's fidelity) or drop (it is "rationale")? Recommendation: keep it alongside the transcription; it is a transcription guard, not an origin argument.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | `go test ./...` baseline + gate | ✓ | current (tests ran; 49 pkgs ok) | — |
| `gh` CLI | ATTR-03 live license check | ✓ | 2.97.0 | `curl` the `https://api.github.com/repos/seanb4t/codegraph-go/license` endpoint (unauthenticated) |
| `task` | `task test` / repo convention | ✓ | 3.52.0 | `go test ./...` directly |
| `rg` | census + positive-control grep gates | ✓ | present | `grep -r` (per grepping rules) |
| `git` | diff review, one-cause commits | ✓ | — | — |
| network + `gh` auth | `gh api ...` | ✓ | verified working this session | — |

**Missing dependencies with no fallback:** none.

## Validation Architecture

`workflow.nyquist_validation: true` (config.json, verified). This phase adds no unit-test source (docs sweep); its validation surface is the verification gate list. The existing golden suite and Taskfile infra are untouched.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (existing) — no new test files in this phase |
| Config file | none — `go test ./...` is the gate |
| Quick run command | `CGO_ENABLED=1 go test ./...` |
| Full suite command | `CGO_ENABLED=1 go test ./...` (49 packages, green baseline) |

### Phase Requirements → Test Map (verification tasks, not new unit tests)
| Req ID | Behavior | Test Type | Automated/V Command | File Exists? |
|--------|----------|-----------|---------------------|-------------|
| ATTR-01 | NOTICE holds transcription + one origin sentence, no argument | assertion (manual grep is disallowed blind — positive-control it) | `rg -n "drop-in|ported|FLAG-PARITY|parity" NOTICE` → zero hits | is the file itself; verified in-repo |
| ATTR-02 | README has no `## Relationship to the original`; License has one NOTICE-linked clause | assertion | `rg -c "Relationship to the original" README.md` → 0 | ✅ |
| ATTR-03 | `gh api` reports MIT | live API | `gh api repos/seanb4t/codegraph-go/license --jq '.license.spdx_id'` → `MIT` | ✅ (endpoint) |
| DOCS-01 | census rows for README/CONTRIBUTING/SECURITY/CoC/PARSER-DECISION are resolved | grep + read | positive-controlled `rg` per term; read each edited file | ✅ |
| DOCS-02 | both files deleted; in-scope references gone; `go test ./...` green | go test + grep | `go test ./...`; positive-controlled `rg "FLAG-PARITY|flag_parity"` with exclusions → zero | ✅ (exclusions recorded) |
| DOCS-03 | matrix has no "golden-parity"/"reference implementation"/`weft`; `matrix_test.go` still green | go test + grep | `CGO_ENABLED=1 go test ./internal/indexer/capability/...` | ✅ |
| DOCS-04 | RELEASE/RELEASE-PROCEDURES/MCP docs carry no retired framing | read | census rows confirmed; the two KEEP rows have recorded reasons | ✅ |

### Sampling Rate
- **Per task commit:** `CGO_ENABLED=1 go test ./internal/...` (fast) or targeted.
- **Per wave merge:** `CGO_ENABLED=1 go test ./...` + the positive-controlled greps.
- **Phase gate:** full suite green + `gh api` → `MIT` + census/`FLAG-PARITY` greps clean before `/gsd-verify-work`.

### Wave 0 Gaps
- None — this phase creates no test infrastructure. The one gate that must be authored fresh is the **positive-controlled** verification command for the DOCS-02 sweep and each vocabulary term (standing decision: a transcript grep is a claim about the grep; prove it can find the term when present).

## Security Domain

`security_enforcement: true` (config.json, verified). The phase edits markdown and deletes one `_test.go` file; it changes no authentication, session, access-control, input-validation, or cryptography code.

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | n/a — no auth code touched |
| V3 Session Management | no | n/a |
| V4 Access Control | no | n/a |
| V5 Input Validation | no | n/a — no user input handled; doc edits only |
| V6 Cryptography | no | n/a — cosign/provenance docs unchanged; no key material touched |
| (V8/others) | no | n/a |

### Known Threat Patterns
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Accidental LICENSE modification downgrading GitHub license detection to `NOASSERTION` | Tampering | Never edit LICENSE; ATTR-03 live check after the NOTICE change; the NOTICE warning block documents the observed failure |
| A dangling reference to `docs/FLAG-PARITY.md`/`flag_parity_test.go` misdirecting contributors after deletion | Tampering | The enumerated 10-site sweep + recorded exclusions; positive-controlled grep gate |
| A regex draft over "TypeScript"/"TS" removing the `tsextract` indexed-language surface | DoS (capability loss) | D-04 term-by-term discipline; D-05 product-truth keeps; CODE-01's named "naive-regex failure mode" (ROADMAP Phase 5 Notes) |

No new attack surface is introduced by this phase.

## Sources

### Primary (HIGH confidence — read/executed this session, all in-repo)
- `NOTICE` (read in full, lines 1-70) — the ATTR-01 surface; exact argument + transcription text quoted
- `README.md` (read in full) — Relationship/License/Performance/Status sections quoted verbatim
- `LICENSE` (read in full) — verbatim MIT confirmed
- `docs/FLAG-PARITY.md` (read in full) + `internal/cli/flag_parity_test.go` (read in full) — deletion targets
- `docs/LANGUAGE-CAPABILITY-MATRIX.md` (read in full) — DOCS-03 surface
- `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `PARSER-DECISION.md` (census-verified; hotspot lines read) — DOCS-01 surfaces
- `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md` (`:60`, `:581` verified), `docs/MCP-2026-07-28-SCOPING.md`, `docs/MCP-8-AGENT-AUDIT.md`, `docs/CORPUS-MEASUREMENT.md` — DOCS-04 surfaces
- `04-CONTEXT.md`, `04-DISCUSSION-LOG.md`, `REQUIREMENTS.md`, `STATE.md`, `ROADMAP.md` (`:195-231`) — locked decisions, requirement texts, exclusions
- `testdata/golden/behavioral_test.go` (`:1-18`, `:684-692`) — harness reality for the stale-prose rows
- `internal/mcp/instructions_contract_test.go` (`:16`, `:375`), `internal/mcp/tools_schema_drift_test.go` (`:36,39`), `internal/upgrade/taskfile_shape_test.go` (`:1345`) — DOCS-02 reference sites + unaffected-guard proof
- `.planning/config.json` — `nyquist_validation: true`, `security_enforcement: true`
- Live `gh api repos/seanb4t/codegraph-go/license` → `spdx_id: MIT` — ATTR-03 executed
- `CGO_ENABLED=1 go test ./...` → 49 packages ok — deletion baseline executed

### Secondary (MEDIUM confidence)
- None required — every claim was verified against an in-repo source of truth this session; no external documentation was needed for a self-applicable docs sweep.

### Tertiary (LOW confidence)
- None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — the phase installs no packages; tool versions verified live
- Blast radius / census: HIGH — every file:line read this session; exclusions verified via `.gitignore` and `git ls-files`
- Architecture: HIGH — sweep flow is a straightforward docs gate chain; the only judgment calls are surfaced as Open Questions with recommendations

**Research date:** 2026-08-15
**Valid until:** this phase (docs sweep) — validity is bounded by in-repo file state, which the plan's executor re-reads at execution time. Re-verify the census rows if any Phase-5 work precedes Phase 4 in the same files.