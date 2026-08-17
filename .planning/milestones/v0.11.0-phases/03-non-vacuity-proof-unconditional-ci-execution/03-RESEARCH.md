# Phase 3: Non-Vacuity Proof & Unconditional CI Execution - Research

**Researched:** 2026-08-15
**Domain:** golden-suite mutation rehearsal (FIXT-07) + corpus-aware CI job with executed-scenario-count assertion (FIXT-03)
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### FIXT-07 — the mutation matrix

- **D-01:** Each assertion family gets **one targeted mutation, recorded per family** — the mutation applied, the observed failure, and the byte-clean revert each recorded. The families in the re-baselined suite: (a) the CASES.json behavioral property assertions; (b) the golden byte-identity guard `TestReFrozenGoldensValid`; (c) the CLI==MCP byte-identity trio; (d) the locked-corpus hermetic (fail-loud) resolution; (e) Phase 1's coverage guard. Each gets a defining mutation that makes that family go RED (e.g. weaken an assertion; delete a golden; break the CLI/MCP parity; remove a corpus and confirm fail-not-skip; drop a kind below threshold). Phase 1/2 already RED-proved (d) and (e) — whether those are re-proven or cited as prior evidence is the planner's recorded call; the criterion is that each family has a recorded RED demonstration, not that all five are re-mutated this phase. — **Reversibility:** reversible — mutations are applied and reverted byte-clean; nothing lands.

#### FIXT-03 — the scenario-count mechanism

- **D-02:** The golden test **self-asserts the executed-scenario count** against an expected constant — the wire-oracle `ExpectedScenarioCount` precedent. A run that executed zero scenarios fails the test, so a silently-skipped CI run is red by construction, not by a CI grep step. The expected count is derived from the authoritative source (the gocapture spec table / CASES.json case map), not a hand-maintained number that can drift. — **Reversibility:** reversible — a test change.

#### Unconditional CI placement

- **D-03:** The golden suite runs in a **new job in the existing `corpora.yml` sibling workflow** (Phase 1). That workflow is already corpus-aware (fetch + assert + nscloud cache), path-filtered, and `contents: read` — the safe side of the repo's cache-trust line (`release.yml:115-120` excludes the cache from the `id-token: write` job). The golden job runs unconditionally (not gated on cache-hit), and a fetch or cache failure fails loudly rather than skipping. `ci.yml`'s general test job is left unchanged — the corpus concern stays in the corpus-aware workflow. — **Reversibility:** reversible — a workflow change.

### Claude's Discretion

- The exact expected-count value and how it's surfaced in the test output (constant vs derived display), provided the test self-asserts against it and the derivation is from the authoritative source.
- Whether Phase 1/2's prior RED demonstrations for the hermetic-resolution and coverage-guard families are re-mutated or cited as prior evidence, recorded in the plan.
- The new job's name and its exact trigger wiring within corpora.yml, provided it is unconditional and fails loudly on fetch/cache failure.

### Deferred Ideas (OUT OF SCOPE)

- Whether the hermetic-resolution and coverage-guard families are re-mutated this phase or cited as prior RED evidence — recorded in the plan, not decided here.
- Phase 5's in-tree comment sweep (waits on this phase so no comment change shares a diff with a golden change).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FIXT-03 | No golden test self-skips in CI — the suite runs against the fetched corpora on every CI run, a fetch/cache failure fails the job loudly rather than skipping, and the job carries a positive assertion that the suite actually executed (scenario count, not merely a non-failing exit) | New unconditional `golden` job in corpora.yml (Section: Architecture Patterns → Pattern 1); widened trigger paths (Section: Silent-Pass Vectors S2); `-count=1` (Pitfall 4); executed-count assertion design (Section: Scenario-Count Mechanism); `inScopeJobs` entry (Section: CI Shape Guard); the golden test family is already skip-free — zero `t.Skip` calls verified (Section: Silent-Pass Vectors) |
| FIXT-07 | The re-baselined golden suite is demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation per assertion family, proving it did not go vacuous | Per-family mutation surface enumeration (Section: Assertion Families); mutation-revert discipline precedent from 01-04-SUMMARY and 02-04-SUMMARY (Section: Mutation-Revert Discipline); recommended recording artifact = the phase VERIFICATION.md / a MUTATION-LOG.md |
</phase_requirements>

## Summary

Phase 3 proves the Phase-2 re-baselined golden suite is non-vacuous (FIXT-07) and makes CI unable to silently stop running it (FIXT-03). The entire phase is **test- and workflow-only**: it installs no new external packages, touches no golden byte, and changes no assertion. The two deliverables are (1) a per-assertion-family recorded RED demonstration (mutation applied → observed failure → byte-clean revert), and (2) a new unconditional CI job in `corpora.yml` that fetches the locked corpora, runs the golden suite against them, and positively asserts an executed-scenario count.

**Primary recommendation:** Add a second job `golden` to `.github/workflows/corpora.yml` that is self-contained (its own unconditional `corpora:fetch` + `corpora:assert` + a `-count=1` golden-suite run), surfaced via a new Taskfile target, added to `inScopeJobs` in `internal/upgrade/taskfile_shape_test.go`, and triggered by a **widened path filter that includes every golden-suite input** (`testdata/golden/**`, `corpus/**`, and the query/CLI/MCP surfaces the goldens freeze). The executed-count assertion extends `TestReFrozenGoldensValid`'s existing 26/26 machinery with a wire-oracle-style exact-equality count derived from the same `expectedGoCaptures` table, plus a CASES.json case-count assertion in `TestCorpusBehaviorSynthetic`. For FIXT-07, record each family's mutation → RED → revert in the phase VERIFICATION.md (the precedent set by 01-04-SUMMARY's tamper rehearsal and 02-04-SUMMARY's delete-a-golden rehearsal); families (d) and (e) have partial prior evidence — the planner must record whether they are re-mutated or cited.

**The single most important adversarial finding:** `ci.yml`'s existing "Test golden parity suite" step (`task test:golden`) now runs the locked-corpus tests, which fail loudly with `t.Fatalf` on an empty corpus root — verified empirically this session. D-03's "leave ci.yml unchanged" therefore needs a recorded reconciliation: either remove the `test:golden` step from ci.yml (the golden run moves wholly to the corpora.yml `golden` job) or accept a red ci.yml on fresh runners until a corpora.yml run populates the shared nscloud volume. This is unresolvable from local evidence (no CI run has ever executed on this milestone branch — `gh run list` returns zero runs for `gsd/v0.11.0-standalone-project-identity`) and must be settled by a first workflow run on this branch.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Executed-scenario-count self-assertion | API / Test code (golden package) | CI job (surfaces it) | The test self-asserts `verified == expected` (wire-oracle `TestScenarioCountIsExact` precedent); the CI job only runs the test and reports the log line. Never a CI grep step — D-02 |
| Corpus fetch at pinned SHAs | CI / infra (corpora.yml) | `internal/corpora` + `tools/corpora` (the fetch driver) | Phase 1 already owns fetch/assert; the golden job reuses the exact wiring, unconditional |
| Golden suite execution against fetched corpora | CI / infra (corpora.yml `golden` job) | `Taskfile.yml` target | `CODEGRAPH_CORPUS_DIR` job-level env is inherited by `task` → `go test` → `CorpusRoot()` |
| Per-family mutation rehearsal | Test engineering (this phase's VERIFICATION) | — | The rehearsal is a one-time applied-and-reverted act recorded per family, not a permanent test |
| Single-definition CI shape guard | Test code (`internal/upgrade/taskfile_shape_test.go`) | — | Every in-scope `run:` step must be exactly `task <target>`; the new job joins `inScopeJobs` |

## Standard Stack

### Core

This phase introduces **no new libraries or packages**. It is a pure Go standard-library test change plus a YAML workflow change. The "stack" is the existing golden suite's own machinery, enumerated here because the phase builds on it:

| Library / Surface | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `testing` (stdlib) | Go 1.26.5 (`go.mod` line 3, `[VERIFIED: go.mod:3]`) | All golden/count assertions | The suite already uses it; the executed-count assertion is `t.Fatalf` on mismatch |
| `encoding/json` (stdlib) | — | CASES.json + golden envelope parsing | `loadBehavioralCases` / `goldenCapture` already use it |
| `internal/corpora` | in-repo | `CorpusRoot()`, `Load`, `LockedEntries` | The hermetic resolver the golden job's env feeds (`[VERIFIED: internal/corpora/manifest.go:205-217]`) |
| `internal/upgrade` shape guard | in-repo | `TestWorkflowRunBodiesInvokeTask` | Enforces the `run: task <target>` single-definition property |
| `namespacelabs/nscloud-cache-action` | pinned `c5f8dab…` # v1.6.1 (`[VERIFIED: .github/workflows/corpora.yml:135]`) | Corpus volume cache | Already the repo's locked mechanism (D-06, v1.0 Phase 10); reused verbatim, not re-introduced |

### Supporting

| Surface | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `Taskfile.yml` (`corpora:fetch`, `corpora:assert`, `test:golden`) | — | Single definition of every CI job body | Every workflow `run:` step must be `task <target>`; add a `golden:verify` target if the job needs fetch+assert+test composed |
| `actions/checkout` | pinned `df4cb1c…` # v6.0.3 | Job checkout | Copy the existing corpora job's steps verbatim |
| `actions/setup-go` | pinned `924ae3a…` # v6.5.0, `cache: false` | Go toolchain | Same as existing corpora job |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| New job in corpora.yml (D-03) | Move the golden run into ci.yml's test job | ci.yml is not corpus-aware (no `CODEGRAPH_CORPUS_DIR` env, `cache: go` only); D-03 locks the corpora.yml placement |
| `go test -count=1` in the golden task | Rely on the go test cache | The cache can return PASS without executing against the current corpus tree — a silent-skip vector (Pitfall 4) |
| Widened corpora.yml path filter | A separate golden workflow | D-03 locks "existing corpora.yml sibling"; widening makes the expensive drift leg fire on golden-relevant paths — accepted, correctness-first (Silent-Pass Vector S2) |

**Installation:** none. No `go get`, no new GitHub Actions. All external actions are already pinned in corpora.yml.

**Version verification:** not applicable — no new packages. The Go version is `1.26.5` (`[VERIFIED: go.mod:3]`).

## Package Legitimacy Audit

> No external packages are installed by this phase. The only third-party elements are GitHub Actions already pinned and in use in `corpora.yml` (checkout, setup-go, nscloud-cache-action) — reuse, not new installs. No legitimacy gate is required. The workflow YAML strings this phase adds (job id, step names) are new *discrete values* in a repo-owned file, not package names.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| *(none — no new external packages this phase)* | — | — | — | — | — | N/A |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
                        ┌────────────────────────────────────────────────┐
                        │            .github/workflows/corpora.yml        │
                        │  (path-filtered trigger — MUST be widened, S2)  │
                        └────────────────────────────────────────────────┘
                                    │ fires on golden-relevant paths
        ┌───────────────────────────┴───────────────────────────┐
        │                                                       │
        ▼                                                       ▼
  job: corpora (existing)                               job: golden (NEW, D-03)
  fetch → assert → drift                                 fetch (UNCONDITIONAL, idempotent)
                                                         assert (4-part integrity)
                                                         run golden suite (-count=1)
                                                         └→ test self-asserts executed
                                                            scenario count (D-02)
        │                                                       │
        │  CODEGRAPH_CORPUS_DIR = ${{ github.workspace }}/../codegraph-corpora
        │  (job-level env, inherited by task → go test → CorpusRoot())
        ▼                                                       ▼
  ┌─────────────────────┐                            ┌───────────────────────────────┐
  │ nscloud volume       │◄─────────cache────────────┤ go test ./testdata/golden/... │
  │ <slug>@<sha> trees   │                            │ - TestReFrozenGoldensValid    │
  └─────────────────────┘                            │ - TestCorpusBehavior*         │
                                                      │ - CLI==MCP trio               │
                                                      │ - hermetic resolution         │
                                                      │ - coverage guard (cheap leg)  │
                                                      └───────────────────────────────┘
```

**Data-flow trace:** a PR touching a golden-suite input triggers corpora.yml → the new `golden` job checks out, mounts the corpus volume, runs unconditional `task corpora:fetch` (miss → real fetch at pinned SHAs), runs `task corpora:assert` (four-part integrity, fails loudly on tamper/absence), then runs the golden suite with `CODEGRAPH_CORPUS_DIR` in the environment. The suite's `lockedCorpusDir` resolves each language to its `Entry.Dir(CorpusRoot())` tree; the tests index live, assert behavioral properties, byte-compare CLI==MCP, and `TestReFrozenGoldensValid` reports "N/N goldens verified". Any of: empty corpus, tampered tree, zero executed scenarios, or a fetch failure → RED job.

### Recommended Project Structure

No new source directories. The phase touches only:

```
.github/workflows/corpora.yml        # add job: golden (D-03)
Taskfile.yml                          # add golden:verify target (fetch+assert+test -count=1)
internal/upgrade/taskfile_shape_test.go  # add {corpora.yml, golden} to inScopeJobs
testdata/golden/golden_test.go       # executed-scenario-count assertion (D-02)
testdata/golden/behavioral_test.go   # optional CASES.json case-count assertion (D-02)
.planning/phases/03-…/03-VERIFICATION.md  # per-family mutation → RED → revert record (FIXT-07)
```

### Pattern 1: The unconditional corpus-aware job (D-03)

**What:** a job whose fetch and assert steps are deliberately NOT gated on the cache-hit output, so a cache miss falls through to a real fetch and a fetch/integrity failure fails the job. The golden test run is the job's final step.

**When to use:** any job whose green signal must mean "the corpus-dependent suite actually exercised the corpus this run."

**Example — the existing fetch step this job reuses verbatim** (`[VERIFIED: .github/workflows/corpora.yml:156-168]`):

```yaml
      - name: Fetch corpora at pinned SHAs
        # UNCONDITIONAL — deliberately NOT gated on steps.cache-corpora.hit.
        # The fetch target is idempotent (it runs the four-part integrity
        # check on an existing destination and skips only a corpus that
        # passes all four), so making it unconditional removes the only path
        # by which a cache anomaly could result in nothing fetching.
        run: task corpora:fetch
```

The new `golden` job copies the existing job's Checkout / Set up Go / Cache corpora volume / Install Task steps verbatim, then: `run: task corpora:fetch`, `run: task corpora:assert`, `run: task golden:verify` (or `task test:golden`). Job-level `env: CODEGRAPH_CORPUS_DIR` is mandatory (same as the existing job at `[VERIFIED: .github/workflows/corpora.yml:122-123]`).

### Pattern 2: The wire-oracle executed-count precedent (D-02)

**What:** the test computes a count from an authoritative source and compares it with EXACT equality (never a lower bound) to a declared constant, failing loudly on any drift.

**When to use:** whenever "the suite ran N scenarios" must be a positive, drift-guarded claim.

**Example — verbatim, `[VERIFIED: test/wireoracle/oracle_test.go:161-166]`:**

```go
func TestScenarioCountIsExact(t *testing.T) {
	got := len(Scenarios())
	if got != ExpectedScenarioCount {
		t.Fatalf("len(Scenarios()) = %d, want exactly %d (ExpectedScenarioCount) — either a scenario silently disappeared or one was added without updating the constant beside Scenarios()", got, ExpectedScenarioCount)
	}
}
```

with `const ExpectedScenarioCount = 42` declared beside `Scenarios()` (`[VERIFIED: test/wireoracle/scenarios.go:540]`).

### Pattern 3: The derived-from-table count (D-02's drift guard)

**What:** the golden executed count is derived by enumerating the authoritative table, so it cannot drift from the fixtures. `TestReFrozenGoldensValid` already does this — `expectedTotal` accumulates from `expectedGoCaptures`, and the guard Fatals on `expectedTotal == 0` and on `verified < expectedTotal` (`[VERIFIED: testdata/golden/golden_test.go:255-308]`).

**When to use:** the count must be "derived from the authoritative source … not a hand-maintained number" (D-02). The `expectedGoCaptures` table is that source on the test side; the CASES.json case map is the source for the property cases.

### Anti-Patterns to Avoid

- **Gating the golden job on `cache-hit`:** reintroduces the exact skip FIXT-02 criterion 5 and FIXT-03 forbid. The existing fetch step's comment already warns "a later reader must NOT reintroduce a condition here."
- **A hand-maintained count literal with no derivation guard:** D-02 forbids "a hand-maintained number that can drift." If a constant is used, a test must prove it equals the table-derived value.
- **A "count of subtests passed" that passes with zero:** `go test` reports `ok` on a package whose tests all short-circuit. This is why (a) no `t.Skip` may exist in the golden family (verified none do) and (b) the golden task must run `-count=1`.
- **Leaving ci.yml's `test:golden` step running locked-corpus tests with no corpus:** verified RED on empty corpus root this session (see Open Question OQ-1).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Corpus fetch/cache in CI | A bespoke workflow cache or inline `git clone` | The existing `corpora:fetch`/`corpora:assert` Taskfile targets + `namespacelabs/nscloud-cache-action` | Phase 1 already solved fetch-at-pinned-SHA + four-part integrity + content-addressed cache; reusing it is the difference between "reuses proven wiring" and "second, divergent cache mechanism" |
| The executed-scenario-count assertion | A CI shell step that greps `go test` output | A Go test self-assertion (wire-oracle `TestScenarioCountIsExact` shape) | D-02: "red by construction, not by a CI grep step"; a grep step can miss and passes vacuously |
| The per-family mutation record | A one-off unrecorded local rehearsal | Structured prose in the phase VERIFICATION.md (precedent: 01-04-SUMMARY's tamper rehearsal) | The criterion is "recorded per family"; an unrecorded rehearsal proves nothing to a reviewer |

**Key insight:** every "don't hand-roll" row above is an existing, already-RED-proven mechanism. Phase 3's job is to *compose* them, not re-author them.

## Runtime State Inventory

> Not a rename/refactor/migration phase — this phase mutates no identifier, no stored string, and no runtime registration. It adds a CI job and a test assertion. Omitted per protocol (rename/refactor/migration phases only).

## Common Pitfalls

### Pitfall 1: The corpora.yml path filter silently excludes the golden suite
**What goes wrong:** a PR that changes only a golden fixture or a golden test file does NOT trigger corpora.yml (its filter covers `corpora/**`, `internal/corpora/**`, `tools/corpora/**`, `internal/indexer/**`, two query status files, `Taskfile.yml`, and the workflow itself — `[VERIFIED: .github/workflows/corpora.yml:66-88]`). The golden job never runs; the suite "passes" by not running.
**Why it happens:** the filter was authored for the drift leg's inputs, not the golden suite's.
**How to avoid:** widen the filter to every golden-suite input: `testdata/golden/**`, `corpus/**`, `internal/query/**`, `internal/cli/**`, `internal/mcp/**`, `cmd/**`.
**Warning signs:** a golden-only PR shows no corpora.yml run in `gh run list`.

### Pitfall 2: ci.yml's existing `test:golden` step without a corpus
**What goes wrong:** ci.yml's test job runs `task test:golden` (`[VERIFIED: .github/workflows/ci.yml:94-95]`) but mounts only `cache: go` — no corpus volume, no `CODEGRAPH_CORPUS_DIR`. Verified this session: with an empty corpus root, `TestPriorityLanguagesResolveToLockedCorpus`, `TestCorpusBehaviorLockedCorpora`, and the locked rows of the CLI==MCP trio all `t.Fatalf` loudly.
**Why it happens:** Phase 2 made the locked-corpus tests hermetic and fail-loud; ci.yml predates that.
**How to avoid:** reconcile with D-03 (Open Question OQ-1): either remove the `test:golden` step from ci.yml (golden run moves wholly to corpora.yml's `golden` job) or verify the shared nscloud volume makes the corpus visible to ci.yml. The first ci.yml run on this branch settles which is required.
**Warning signs:** ci.yml's "Test golden parity suite" step red on a fresh runner.

### Pitfall 3: The executed count is asserted but the go test cache returns a stale PASS
**What goes wrong:** `Taskfile.yml`'s `test:golden` is `go test ./testdata/golden/...` with no `-count=1` (`[VERIFIED: Taskfile.yml:59-65]`). The go test cache keys on package sources, not on external corpus tree content, so a cached PASS can stand in for a real execution.
**Why it happens:** the corpus trees are outside the package dir; their content is invisible to the test cache.
**How to avoid:** the golden job's run step must carry `-count=1` (repo discipline: "Every `go test` carried `-count=1`", 01-07-SUMMARY).
**Warning signs:** a golden job whose step output says "(cached)".

### Pitfall 4: The `expectedGoCaptures` table and gocapture's spec table drift apart
**What goes wrong:** `expectedGoCaptures` (`[VERIFIED: testdata/golden/golden_test.go:193-245]`) and gocapture's `slugOrder`/`lockedCorpusArgs` (`[VERIFIED: testdata/golden/gocapture/main.go:145-193]`) are two hand-maintained tables in different packages. A golden added to gocapture but not to the guard table is unguarded (silent under-coverage); the reverse fails loudly.
**Why it happens:** two sources of truth in two packages (`main` vs `golden`).
**How to avoid:** D-02's derivation rule — make the count (and ideally the table) a single source; the count assertion must derive from the same table the guard enumerates, and a derived-vs-constant equality test (Pattern 2 + 3) makes any drift red.
**Warning signs:** gocapture emits a new file the guard never checks.

### Pitfall 5: `go test ./...` never discovers `testdata/golden` (GOLDEN-01)
**What goes wrong:** the go tool ignores directories named `testdata` in `./...` expansion — the golden suite is invisible to the general test job.
**Why it happens:** documented in ci.yml's own GOLDEN-01 comment.
**How to avoid:** already handled — ci.yml has an explicit `task test:golden` step and the new corpora.yml `golden` job runs `./testdata/golden/...` directly.
**Warning signs:** a "full suite green" signal that omits the golden package.

## Code Examples

Verified patterns from official sources (all in-repo, read this session):

### 1. The executed-count assertion to mirror (D-02)
`[VERIFIED: test/wireoracle/oracle_test.go:161-166]` — `TestScenarioCountIsExact` shown in Architecture Patterns Pattern 2. Its constant: `[VERIFIED: test/wireoracle/scenarios.go:540]` `const ExpectedScenarioCount = 42`.

### 2. The golden guard that already self-asserts a count (D-02's base)
`[VERIFIED: testdata/golden/golden_test.go:255-308]` — `TestReFrozenGoldensValid` enumerates `expectedGoCaptures`, accumulates `expectedTotal` and `verified`, and Fatals on `expectedTotal == 0` and `verified < expectedTotal`. The phase's count work extends this (a wire-oracle-style exact-equality test over the same table, plus a CASES.json case-count assertion in `TestCorpusBehaviorSynthetic`).

### 3. The hermetic resolver that must fail, never skip (family d)
`[VERIFIED: testdata/golden/behavioral_test.go:67-106]` — `lockedCorpusDir(t, lang)` calls `t.Fatalf` on a missing map entry, missing slug, missing repo mapping, manifest load error, missing locked entry, or a resolved dir that is not a directory. Its final line: `t.Fatalf("lockedCorpusDir(%q): locked tree directory %s not found or not a directory: %v; run 'task corpora:fetch'", ...)`.

### 4. The coverage-guard count assertions (family e)
`[VERIFIED: internal/corpora/coverage_test.go:122-151]` — `TestCorpusCoverageClaim` asserts `res.CheckedKinds == len(query.RankEdges)` and `res.CheckedCorpora == len(LockedEntries(m))` and Fatals on `CheckedCorpora == 0`.

### 5. The CI shape guard's in-scope fixture to extend
`[VERIFIED: internal/upgrade/taskfile_shape_test.go:109-119]` — the current list ends with `{Workflow: "corpora.yml", JobID: "corpora"}`. A new `golden` job must be added as `{Workflow: "corpora.yml", JobID: "golden"}` (job ID = YAML map key, per the file's own doc at lines 89-93).

## Silent-Pass Vectors (adversarial)

The research question: *given Phase 2's suite, what are the remaining ways a CI run could pass without the suite exercising anything?*

| # | Vector | Status | Closing Mechanism |
|---|--------|--------|-------------------|
| S1 | A golden test calls `t.Skip` on a missing corpus | **CLOSED — verified** | Zero `t.Skip(` call sites in `testdata/golden/` and `internal/corpora/` (grep this session). The only `t.Skip` in the broader test tree is `test/wireoracle/capture_stderr_test.go:70` (POSIX sh stub) and platform/root-gated upgrade tests — all outside the golden surface. The per-language behavioral tests' headers literally say "t.Fatalf, never t.Skip" (`behavioral_java_test.go:6`) |
| S2 | A glob that matches nothing (vacuous enumeration) | **CLOSED at the guard level** | `TestReFrozenGoldensValid` enumerates from `expectedGoCaptures`, never `filepath.Glob`, and Fatals on `expectedTotal == 0` |
| S3 | The corpus-aware workflow never fires for golden-only changes | **OPEN — the #1 gap** | corpora.yml's path filter omits `testdata/golden/**`, `corpus/**`, and the query/CLI/MCP surfaces. A golden-only PR triggers no corpora.yml run. **Must widen the filter** |
| S4 | The golden job is gated on cache-hit (skip on miss) | **OPEN until the job exists** | D-03 forbids; the new job copies the existing unconditional fetch step and must not add a condition |
| S5 | Go test cache returns a stale PASS without executing | **OPEN until the job exists** | `test:golden` lacks `-count=1`; the golden job's run must add it |
| S6 | The executed-count assertion is derivable-but-wrong (drifts from fixtures) | **PARTIALLY OPEN** | `expectedGoCaptures` vs gocapture's `slugOrder` are two tables; a new golden in gocapture only is unguarded. D-02's derivation + exact-equality test closes it |
| S7 | A count assertion that passes at zero | **CLOSED at the test level** | `TestReFrozenGoldensValid` Fatals on `expectedTotal == 0` and `verified < expectedTotal`; `loadBehavioralCases` Fatals on `len(doc.Cases) == 0` |
| S8 | ci.yml runs the locked-corpus tests with no corpus and they fail loudly | **This is a RED, not a silent-pass** | Verified empirically (empty corpus root → `t.Fatalf`). It cannot silently pass, but it *will* red ci.yml on a fresh runner unless reconciled (OQ-1) |

## Mutation-Revert Discipline (FIXT-07)

### The established precedent — no dedicated mutation log exists
There is **no mutation-log file** in the repo. The established mechanism is **prose in a plan's SUMMARY.md / VERIFICATION.md recording the mutation, the observed failure, and the revert**:

- **01-04-SUMMARY.md** records a live integrity-check rehearsal: "appended a line to `.circleci/config.yml` in the fetched hugo corpus, `corpora:assert-one` failed part 3/4, restored byte-clean via `git checkout --`" (`[VERIFIED: .planning/phases/01-corpus-selection-by-measurement/01-04-SUMMARY.md:99]`), and: "the mutation-proof step (permissive SHA regex → red → byte-clean revert) and the tamper-detection rehearsal (append line → red → `git checkout --` restore) both behaved exactly as designed" (`01-04-SUMMARY.md:184`).
- **02-04-SUMMARY.md** records the golden-guard rehearsal: "`TestReFrozenGoldensValid` passes 26/26; a deliberately-deleted golden fails it (25/26), so it is not vacuous" (`[VERIFIED: 02-04-SUMMARY.md:34]`).
- The repo's standing rule: "A gate is not trusted until it has been demonstrated RED against a confirmed-applied mutation" (`[VERIFIED: .planning/STATE.md:131]`).

**Recommendation:** Phase 3 records each family's mutation → observed failure (paste the failing test output / count) → byte-clean revert (e.g. `git checkout -- <path>`) as a structured table in `03-VERIFICATION.md` (optionally a dedicated `03-MUTATION-LOG.md`). This matches the criterion "the mutation, the observed failure and the revert recorded per family" and the one-named-cause-per-diff discipline. The rehearsal is a live, applied-and-reverted act during execution — not a permanent test (the wire-oracle's permanent negative tests like `TestEmptyTranscriptNeverMatches` are a different pattern and do not satisfy the rehearsal criterion).

### Per-family mutation surface (the planner's matrix)

| Family | Tests (exact names) | Files | Defining mutation that makes it RED | Prior recorded evidence |
|--------|--------------------|-------|-------------------------------------|------------------------|
| (a) CASES.json behavioral property assertions | `TestCorpusBehaviorSynthetic` (4 cases: `overloaded-defs-distinct`, `multi-word-tokenization`, `cluster-surfaces-connected-non-test`, `structural-surfaces-zero-lexical-match` — `[VERIFIED: testdata/golden/behavioral_test.go:856-977]`), `TestCorpusBehavior_Go` | `testdata/golden/behavioral_test.go`, `corpus/behavioral/CASES.json` | Weaken an assertion (e.g. change `len(locs) != 2` to `!= 1`), delete a case from CASES.json, or delete a `tc.Files` entry | None recorded — must be re-mutated this phase |
| (b) Golden byte-identity guard | `TestReFrozenGoldensValid`, `TestGoldenFixturesExist`, `TestGoSideFixturesRegenerated` | `testdata/golden/golden_test.go` | Delete one golden file (e.g. `testdata/golden/corpus/hugo/go-explore.json`) → 25/26 | **Recorded in 02-04-SUMMARY (delete → 25/26)** — may be cited, but the record lacks the full observed-failure transcript; planner decides |
| (c) CLI==MCP byte-identity trio | `TestExploreCLIMatchesMCP` (5 rows), `TestNodeCLIMatchesMCP` (6 rows), `TestNodeLineHintCLIMatchesMCP`; related pin `TestBuildIndexedFixtureIgnoresInheritedStore` | `testdata/golden/behavioral_test.go` | Break parity in one surface (e.g. diverge the MCP render path / skip a normalization) → `cliOut != mcpOut` | None recorded — must be re-mutated this phase |
| (d) Locked-corpus hermetic (fail-loud) resolution | `TestPriorityLanguagesResolveToLockedCorpus` (5 langs), `TestCorpusBehaviorLockedCorpora` (4 corpora × explore-shape/node-shape), the 4 per-language `TestCorpusBehavior_{Java,TSJS,Python,CSharp}` | `testdata/golden/behavioral_test.go` + `behavioral_{java,tsjs,python,csharp}_test.go` | Remove/rename a locked corpus directory → `t.Fatalf` naming the missing tree (fail-NOT-skip) | Verified empirically this session (empty corpus root → loud RED); no written per-family record found in Phase 1/2 summaries — planner decides re-mutate vs. cite |
| (e) Coverage guard | `TestCorpusCoverageClaim` + the 15+ mutation-style tests in `internal/corpora/coverage_test.go` | `internal/corpora/coverage_test.go` | Drop a kind below its threshold, or name a non-locked repository in the locked set → `CheckCoverage` reports failures | 01-07-SUMMARY says both legs "are specified to be demonstrated RED … and reverted byte-clean" (`01-07-SUMMARY.md:24`) — the *plan* specified it; no observed-failure transcript found in 01-VERIFICATION — planner decides re-mutate vs. cite |

**Criterion restated for the planner:** each of the five families must have a *recorded* RED demonstration — either re-mutated this phase (with output pasted) or cited to a prior record that actually contains the observed failure. Families (b) has a partial record (count only); (d) and (e) have "specified" but no observed-output record in the summaries read.

## Scenario-Count Mechanism (FIXT-03 / D-02)

### What exists today
- `TestReFrozenGoldensValid` already computes `verified` and `expectedTotal` from `expectedGoCaptures` and asserts `verified == expectedTotal` with zero-tolerance (`[VERIFIED: testdata/golden/golden_test.go:255-308]`). The "zero executed scenarios fails the test" property **already holds at the test level** for the 26-golden set.
- `TestCorpusBehaviorSynthetic` iterates `loadBehavioralCases()`; a zero-case CASES.json Fatals (`[VERIFIED: testdata/golden/behavioral_test.go:165-181]`), but there is **no exact-count assertion** on the case map.

### The actual expected counts today
- **Golden fixtures: 26** = 4 locked corpora × 6 (`{explore, node, explore-multi, node-multi, explore-mcp, node-mcp}`) + 2 behavioral (`go-explore-multi.json`, `go-node-multi.json`). Verified on disk this session: hugo 6, guava 6, serilog 6, requests 6, behavioral 2.
- **CASES.json property cases: 4** (a-d).
- **Executed scenario surface (for reference):** 5 hermetic-resolution languages, 4 locked corpora × 2 shape subtests, 5+6+1 CLI==MCP rows, 4 per-language behavioral tests, 1 coverage claim.

### The D-02 design (planner's discretion on the exact constant)
Two wires fit the precedent, and both self-assert:
1. **Golden-set count (primary):** add a wire-oracle-style exact-equality test in `golden_test.go` — e.g. `TestGoldenScenarioCountIsExact` asserting the derived `expectedTotal` equals a package constant `ExpectedGoldenScenarioCount`, with the constant either derived (a func) or a literal guarded by the derivation test. Because the count *is* the table enumeration, the strongest form is: the guard derives the count from `expectedGoCaptures` and the exact-equality test proves any displayed constant equals it — so the constant cannot drift.
2. **Property-case count (secondary):** extend `TestCorpusBehaviorSynthetic` (or a sibling test) to assert `len(loadBehavioralCases())` equals the CASES.json-derived expected (4), so a case deletion is red by count, not merely by whatever case remains.

The CI job then reports the test's log line (e.g. "TestReFrozenGoldensValid: 26/26 goldens verified") — the test, not a grep, is the assertion.

## Architecture Patterns — the CI shape guard

`TestWorkflowRunBodiesInvokeTask` (`[VERIFIED: internal/upgrade/taskfile_shape_test.go:1334-1385]`) holds every in-scope job's `run:` step to exactly `task <target>` (comments/blanks stripped), except named `runBodyExceptions` (currently one ci.yml reproducibility step, `[VERIFIED: taskfile_shape_test.go:147-154]`). The current fixture ends `{Workflow: "corpora.yml", JobID: "corpora"}` (`[VERIFIED: taskfile_shape_test.go:118]`).

**The new `golden` job MUST be added to `inScopeJobs`** — otherwise its `run:` bodies are unguarded and a later contributor could insert an inline `go test` into it without failing anything. The job's steps must therefore each be exactly `task <target>` (fetch, assert, and the golden run — the last via a Taskfile target, since `task test:golden` already exists but lacks `-count=1`; either modify `test:golden` to add `-count=1` or add a `golden:verify` target). Note `forbiddenTaskfileGateKeys = ["status", "platforms"]` (`[VERIFIED: taskfile_shape_test.go:66-68]`) — the Taskfile-level silent-skip guard the new job's task must not trip.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The nscloud volume mount in ci.yml (`cache: go`) does not expose the corpus trees at the corpora.yml path, so ci.yml's `test:golden` step runs locked-corpus tests against an empty root on a fresh runner | Silent-Pass Vectors S8 / OQ-1 | If the volume DOES expose the corpus (shared per-repo volume), ci.yml's `test:golden` passes on warm runners and the reconciliation in OQ-1 is unnecessary |
| A2 | No CI has ever run on the milestone branch, so the current ci.yml golden-step behavior on this branch is unobserved | Open Question OQ-1 | If a prior run exists that I could not see, its conclusion would settle A1 — instrument: `gh run list --branch gsd/v0.11.0-standalone-project-identity` at planning time |
| A3 | The `expectedGoCaptures` table is the "authoritative source" D-02 refers to on the test side (gocapture's `slugOrder` is the capture-side source) | Scenario-Count Mechanism | If D-02 intends a single shared table exported from one package, the count test's location changes (still in the golden package) |
| A4 | A `-count=1` addition to the golden run is the intended cache-defeating mechanism | Pitfall 3 | If the go test cache already invalidates on `testdata` content, `-count=1` is belt-and-braces rather than load-bearing — harmless either way |

## Open Questions

1. **What happens to ci.yml's existing `test:golden` step?**
   - What we know: it runs `task test:golden` (`[VERIFIED: .github/workflows/ci.yml:94-95]`); the golden suite's locked-corpus tests `t.Fatalf` on an empty corpus root (verified empirically). D-03 says ci.yml's general test job is "left unchanged" and the corpus concern stays in corpora.yml.
   - What's unclear: whether "left unchanged" permits *removing* the now-corpus-dependent `test:golden` step (moving the golden run wholly to the new corpora.yml `golden` job), or mandates keeping it — which requires the corpus to be visible on ci.yml runners.
   - Recommendation: the plan removes `test:golden` from ci.yml (the golden run moves to corpora.yml), recording it as the D-03 reconciliation; verify with the first ci.yml run on this branch. **Instrument:** a `workflow_dispatch` of ci.yml on this branch after the golden job lands.
2. **Are families (d) and (e) re-mutated this phase or cited as prior evidence?**
   - What we know: D-01 leaves it to the planner. (d) and (e) have "specified" RED demonstrations but no observed-failure transcript in the summaries read; (b) has a count-only record.
   - What's unclear: whether the prior records satisfy "the observed failure … recorded per family."
   - Recommendation: re-mutate (d) and (e) this phase with pasted failure output — cheapest and removes all doubt; cite (b) only if the plan pastes the 25/26 output alongside.
3. **What is the exact executed-count surface the CI job asserts?**
   - What we know: 26 goldens + 4 CASES cases; the guard already asserts the 26. D-02 gives the planner discretion on constant vs. derived display.
   - What's unclear: whether the CI job asserts 26, 30, or a single combined figure.
   - Recommendation: assert 26 at the golden-guard level (already exists; add exact-equality form) and 4 at the CASES level; the CI job reports both from the test output.
4. **Which corpora.yml trigger paths must be added?**
   - What we know: the current filter omits the golden suite's inputs (S3).
   - What's unclear: the exact minimal-but-complete set.
   - Recommendation: `testdata/golden/**`, `corpus/**`, `internal/query/**`, `internal/cli/**`, `internal/mcp/**`, `cmd/**` — every input whose change can alter a golden or a behavior the suite asserts.

## Environment Availability

Step 2.6 applies — the phase depends on the corpus-aware CI workflow and the local corpus trees.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | running/verifying the golden suite | ✓ | 1.26.5 (`go.mod:3`) | — |
| Locked corpora locally (`~/.cache/codegraph/corpora`) | running the locked-corpus tests locally | ✓ | 4 locked trees present (hugo, guava, serilog, requests) | `task corpora:fetch` |
| GitHub Actions runner class (`namespace-profile-linux-amd64-4x8`) | the corpora.yml jobs | ✓ (existing workflows use it) | — | The nscloud cache action does not work on standard runners (`[VERIFIED: corpora.yml:97-99]`) |
| `gh` CLI | observing CI runs (this session) | ✓ | — | — |
| First CI run on this branch | settling OQ-1 / A1 | ✗ (never run) | — | A `workflow_dispatch` on ci.yml + corpora.yml is the first instrument |

**Missing dependencies with no fallback:** a first CI run on `gsd/v0.11.0-standalone-project-identity` — no local substitute for observing the workflow's actual behavior; the plan should gate the workflow change behind that observation (or at least record the expectation).

**Missing dependencies with fallback:** none blocking — the corpus fetch target is idempotent and locally available.

## Validation Architecture

> `workflow.nyquist_validation: true` (`[VERIFIED: .planning/config.json]` — key present and true), so this section is required. `security_enforcement: true` → Security Domain section below.

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` (Go 1.26.5), `go test -count=1 ./testdata/golden/...` |
| Config file | none — standard `go test` flags; `-count=1` is the repo's cache-defeating discipline |
| Quick run command | `go test -count=1 ./testdata/golden/... -run 'TestReFrozenGoldensValid|TestGoldenScenarioCountIsExact'` |
| Full suite command | `go test -count=1 ./testdata/golden/...` (with corpora fetched; `CODEGRAPH_CORPUS_DIR` set) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FIXT-03 | Executed-scenario-count self-assertion (goldens) | unit | `go test -count=1 ./testdata/golden/... -run 'TestReFrozenGoldensValid|TestGoldenScenarioCountIsExact'` | ✅ `testdata/golden/golden_test.go` (assertion to be added) |
| FIXT-03 | Executed-case-count self-assertion (CASES.json) | unit | `go test -count=1 ./testdata/golden/... -run TestCorpusBehaviorSynthetic` | ✅ `testdata/golden/behavioral_test.go` (case-count to be added) |
| FIXT-03 | Golden suite runs against fetched corpora in CI, unconditional | integration (workflow) | corpora.yml `golden` job (workflow, not unit-testable) | ✅ `.github/workflows/corpora.yml` (job to be added) |
| FIXT-03 | Fetch/cache failure fails loudly, never skips | integration (workflow) | corpora.yml `golden` job fetch/assert steps | ✅ existing `corpora.yml` steps reused |
| FIXT-03 | Every golden-job `run:` body is exactly `task <target>` | unit | `go test -count=1 ./internal/upgrade/... -run TestWorkflowRunBodiesInvokeTask` | ✅ `internal/upgrade/taskfile_shape_test.go` (inScopeJobs entry to be added) |
| FIXT-07 | Per-family mutation → RED → byte-clean-revert recorded | manual (rehearsal) | Manual rehearsal; recorded in `03-VERIFICATION.md` (or `03-MUTATION-LOG.md`) | ❌ Wave 0 — artifact created by this phase |

### Sampling Rate
- **Per task commit:** `go test -count=1 ./testdata/golden/... -run '<the family just rehearsed>'`
- **Per wave merge:** `go test -count=1 ./testdata/golden/... ./internal/corpora/... ./internal/upgrade/...` (full suite + shape guard + coverage guard)
- **Phase gate:** full suite green (with corpora fetched) before `/gsd-verify-work 3`; the VERIFICATION records all five family rehearsals

### Wave 0 Gaps
- [ ] `testdata/golden/golden_test.go` — add `TestGoldenScenarioCountIsExact` (or equivalent) covering FIXT-03's count self-assertion
- [ ] `testdata/golden/behavioral_test.go` — add CASES.json case-count assertion covering FIXT-03's property-case surface
- [ ] `internal/upgrade/taskfile_shape_test.go` — add `{Workflow: "corpora.yml", JobID: "golden"}` to `inScopeJobs`
- [ ] `.github/workflows/corpora.yml` — add the `golden` job + widened path filter
- [ ] `03-VERIFICATION.md` — the FIXT-07 per-family record (rehearsals are manual; no automated command)

## Security Domain

> `security_enforcement: true` (`[VERIFIED: .planning/config.json]`), so this section is required. ASVS level 1.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | partial | The golden job runs on the SAFE side of the cache-trust line: `contents: read` only, no `id-token: write`, no signing (D-03; `[VERIFIED: corpora.yml:90-91]` `permissions: contents: read`) |
| V5 Input Validation | n/a for the phase itself; the corpus trees are adversarial inputs to the parser | Existing posture: four-part integrity check before indexing (`corpora:assert-one`) — unchanged by this phase |
| V6 Cryptography | no | — |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| A mutable cache entry poisoned at the correct commit | Tampering | Reuse `corpora:assert`'s four-part integrity check (`.git` present, HEAD==pin, `git status --porcelain --ignored` EMPTY, tree resolves) before the golden suite indexes — the same control corpora.yml already applies |
| A fetch/cache failure silently becoming a skipped job | DoS (availability of the gate) | The golden job's fetch/assert steps are unconditional and fail loudly (D-03); never gated on `cache-hit` |
| Golden-only PR bypassing the corpus-aware workflow entirely | — (process gap) | Widened corpora.yml path filter (S3) so golden-relevant changes always trigger the job that proves the suite non-vacuous |

## Sources

### Primary (HIGH confidence — read in-repo this session)
- `.planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-CONTEXT.md` — D-01..D-03, locked decisions
- `.planning/REQUIREMENTS.md:39,43` — FIXT-03, FIXT-07 verbatim
- `.planning/STATE.md:131-132` — mutation-demonstration and positive-assertion standing rules
- `.planning/phases/02-golden-harness-re-authoring-re-freeze/02-CONTEXT.md` — D-01..D-10 (the suite being proven)
- `.planning/phases/02-golden-harness-re-authoring-re-freeze/02-04-SUMMARY.md` — 26-golden set, determinism, guard RED (25/26)
- `.planning/phases/02-golden-harness-re-authoring-re-freeze/02-VERIFICATION.md` — family wiring evidence
- `.planning/phases/01-corpus-selection-by-measurement/01-07-SUMMARY.md` — coverage guard + CI cache + mutation specification
- `.planning/phases/01-corpus-selection-by-measurement/01-04-SUMMARY.md` — recorded tamper-rehearsal precedent
- `testdata/golden/golden_test.go` — `TestReFrozenGoldensValid`, `expectedGoCaptures`, `TestGoldenFixturesExist`, `TestGoSideFixturesRegenerated`
- `testdata/golden/behavioral_test.go` — `lockedCorpusDir`, `TestCorpusBehaviorSynthetic`, `TestCorpusBehavior_Go`, CLI==MCP trio, hermetic tests
- `testdata/golden/behavioral_{java,tsjs,python,csharp}_test.go` — per-language hermetic property tests
- `testdata/golden/gocapture/main.go` — `buildSpecs` spec table
- `test/wireoracle/oracle_test.go:161-166` + `test/wireoracle/scenarios.go:540` — the executed-count precedent
- `internal/corpora/coverage_test.go` + `internal/corpora/manifest.go:205-217` — coverage guard + `CorpusRoot()`
- `internal/upgrade/taskfile_shape_test.go` — `inScopeJobs`, `TestWorkflowRunBodiesInvokeTask`, `forbiddenTaskfileGateKeys`
- `.github/workflows/corpora.yml` — full read (jobs, triggers, cache, trust posture)
- `.github/workflows/ci.yml:94-95` — the existing `test:golden` step
- `Taskfile.yml:59-65` — `test:golden` target
- `corpus/behavioral/CASES.json` — 4 cases (a-d)

### Secondary (MEDIUM confidence)
- `gh run list` output — no CI runs on `gsd/v0.11.0-standalone-project-identity`; last main ci.yml run 2026-08-13 (pre-baseline). This is a live-CI observation, not documentation.

### Tertiary (LOW confidence)
- `[ASSUMED]` nscloud volume visibility across ci.yml's `cache: go` mount (A1) — unresolvable locally; instrument is a first CI run.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — the phase adds no packages; it reuses verified in-repo machinery read this session
- Architecture: HIGH — the job shape, count mechanism, and inScopeJobs wiring are all directly evidenced in the read files
- Pitfalls: HIGH — S1 and S8 verified empirically (test runs this session); S3 is a direct reading of the workflow filter

**Research date:** 2026-08-15
**Valid until:** 2026-09-14 (the two workflow files and the golden test files are the moving parts; fast-moving if CI behavior on this branch is observed to differ)
