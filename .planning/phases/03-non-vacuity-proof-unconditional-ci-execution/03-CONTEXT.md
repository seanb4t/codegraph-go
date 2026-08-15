# Phase 3: Non-Vacuity Proof & Unconditional CI Execution - Context

**Gathered:** 2026-08-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Prove the re-baselined golden suite (Phase 2) is trusted because it has been watched fail, and make CI unable to silently stop running it.

**In scope:** FIXT-07 (demonstrate each assertion family RED against a confirmed-applied, byte-cleanly-reverted mutation, recorded per family), FIXT-03 (no golden test self-skips in CI — the suite runs against the fetched corpora on every CI run, a fetch/cache failure fails loudly, and the job carries a positive executed-scenario-count assertion).

**Depends on:** Phase 2 (the re-baselined suite that must exist in final form before it is proven). Deliberately separate from the re-freeze — a re-baseline that authors its own proof in the same change certifies its own oracle.

**Blocks:** Phase 5 (Process, CI & In-Tree Sweep), which waits so no in-tree comment change shares a diff with a golden change.

**Does NOT touch:** the golden content or assertions themselves (Phase 2 froze them); the corpus lock (Phase 1); the benchmark path (Phase 6).
</domain>

<decisions>
## Implementation Decisions

### FIXT-07 — the mutation matrix

- **D-01:** Each assertion family gets **one targeted mutation, recorded per family** — the mutation applied, the observed failure, and the byte-clean revert each recorded. The families in the re-baselined suite: (a) the CASES.json behavioral property assertions; (b) the golden byte-identity guard `TestReFrozenGoldensValid`; (c) the CLI==MCP byte-identity trio; (d) the locked-corpus hermetic (fail-loud) resolution; (e) Phase 1's coverage guard. Each gets a defining mutation that makes that family go RED (e.g. weaken an assertion; delete a golden; break the CLI/MCP parity; remove a corpus and confirm fail-not-skip; drop a kind below threshold). Phase 1/2 already RED-proved (d) and (e) — whether those are re-proven or cited as prior evidence is the planner's recorded call; the criterion is that each family has a recorded RED demonstration, not that all five are re-mutated this phase. — **Reversibility:** reversible — mutations are applied and reverted byte-clean; nothing lands.

### FIXT-03 — the scenario-count mechanism

- **D-02:** The golden test **self-asserts the executed-scenario count** against an expected constant — the wire-oracle `ExpectedScenarioCount` precedent. A run that executed zero scenarios fails the test, so a silently-skipped CI run is red by construction, not by a CI grep step. The expected count is derived from the authoritative source (the gocapture spec table / CASES.json case map), not a hand-maintained number that can drift. — **Reversibility:** reversible — a test change.

### Unconditional CI placement

- **D-03:** The golden suite runs in a **new job in the existing `corpora.yml` sibling workflow** (Phase 1). That workflow is already corpus-aware (fetch + assert + nscloud cache), path-filtered, and `contents: read` — the safe side of the repo's cache-trust line (`release.yml:115-120` excludes the cache from the `id-token: write` job). The golden job runs unconditionally (not gated on cache-hit), and a fetch or cache failure fails loudly rather than skipping. `ci.yml`'s general test job is left unchanged — the corpus concern stays in the corpus-aware workflow. — **Reversibility:** reversible — a workflow change.

### Claude's Discretion

- The exact expected-count value and how it's surfaced in the test output (constant vs derived display), provided the test self-asserts against it and the derivation is from the authoritative source.
- Whether Phase 1/2's prior RED demonstrations for the hermetic-resolution and coverage-guard families are re-mutated or cited as prior evidence, recorded in the plan.
- The new job's name and its exact trigger wiring within corpora.yml, provided it is unconditional and fails loudly on fetch/cache failure.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### The suite being proven (Phase 2 output)
- `.planning/phases/02-golden-harness-re-authoring-re-freeze/02-CONTEXT.md` — D-01..D-10, especially D-09 (property-assertion style) and D-10 (hermetic fail-loud resolution)
- `.planning/phases/02-golden-harness-re-authoring-re-freeze/02-04-SUMMARY.md` — the re-frozen set (26 goldens), determinism proof, and the guard
- `testdata/golden/behavioral_test.go` — the re-authored CASES.json-driven property assertions
- `testdata/golden/golden_test.go` — `TestReFrozenGoldensValid` (expected-set enumeration) and `TestGoSideFixturesRegenerated`
- `corpus/behavioral/CASES.json` — the case map

### The CI surface (Phase 1 output)
- `.github/workflows/corpora.yml` — the corpus-aware sibling workflow the golden job extends
- `Taskfile.yml` — the `golden:regen` target and the `corpora:*` targets; single CI definition (`TestWorkflowRunBodiesInvokeTask`)
- `.planning/phases/01-corpus-selection-by-measurement/01-07-SUMMARY.md` — the coverage guard + CI cache this phase's golden job runs alongside

### The proof discipline
- Repo rule `84d1gfpywd` — a guard MUST carry a positive assertion that it did its work
- `.planning/STATE.md` standing decisions — "A gate is not trusted until it has been demonstrated RED against a confirmed-applied mutation"
- `test/wireoracle/oracle_test.go` — `ExpectedScenarioCount` / `TestScenarioCountIsExact`, the precedent D-02 follows
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `test/wireoracle/oracle_test.go` `ExpectedScenarioCount` — the executed-count precedent D-02 mirrors; the golden test self-asserts an executed count the same way.
- `.github/workflows/corpora.yml` — already has fetch + assert + nscloud cache + unconditional steps; the golden job reuses that wiring.
- `TestReFrozenGoldensValid` (golden_test.go) — the expected-set enumeration D-02's count assertion builds on.

### Established Patterns
- **Fail-loud, never skip** — Phase 2's hermetic tests fail loudly on a missing corpus; Phase 3 makes the same posture hold in CI.
- **Positive assertion (rule `84d1gfpywd`)** — the scenario count is asserted, not merely "did not fail."
- **One named cause per diff** — the mutation demonstrations are recorded per family, not bundled.

### Integration Points
- The golden job joins corpora.yml's existing fetch/assert/cache steps.
- The count assertion joins `TestReFrozenGoldensValid`'s expected-set machinery.
</code_context>

<specifics>
## Specific Ideas

- The mutation families and their defining mutations are recorded per family in a visible artifact (the VERIFICATION / a mutation log), not scattered across commits.
- A silently-skipped CI run must be RED, not merely "not green" — the count assertion is the mechanism.
- The golden job is unconditional — a cache miss falls through to a real fetch (never a skip), matching FIXT-02's requirement.

</specifics>

<deferred>
## Deferred Ideas

- Whether the hermetic-resolution and coverage-guard families are re-mutated this phase or cited as prior RED evidence — recorded in the plan, not decided here.
- Phase 5's in-tree comment sweep (waits on this phase so no comment change shares a diff with a golden change).

</deferred>

---

*Phase: 03-non-vacuity-proof-unconditional-ci-execution*
*Context gathered: 2026-08-15*
