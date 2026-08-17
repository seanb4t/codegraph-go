# Phase 3: Non-Vacuity Proof & Unconditional CI Execution — Discussion Log

**Gathered:** 2026-08-15
**Mode:** autonomous `--interactive` (discuss inline, user answers all questions)

---

## Phase Boundary (from ROADMAP)

Prove the re-baselined golden suite (Phase 2) is trusted because it has been watched fail, and make CI unable to silently stop running it. Requirements: FIXT-03, FIXT-07. Blocks Phase 5.

## Areas selected for discussion

All three gray areas (user-selected all):

1. FIXT-07 mutation matrix
2. FIXT-03 scenario-count mechanism
3. Unconditional CI placement

## Decisions by area

### 1. FIXT-07 mutation matrix → one mutation per family, per-family recorded
**Options:** one mutation per family / two representative / every family
**Selected:** one mutation per family, per-family recorded.
Each assertion family (behavioral property assertions, byte-identity guard, CLI==MCP trio, hermetic resolution, coverage guard) gets one targeted mutation: applied, observed failure, byte-clean revert, each recorded. Whether the hermetic-resolution and coverage-guard families are re-mutated or cited as prior RED evidence is the planner's recorded call.

### 2. FIXT-03 scenario-count mechanism → test self-asserts the count
**Options:** test reports + CI greps / test self-asserts / both
**Selected:** test self-asserts the count.
The golden test asserts the executed-scenario count against an expected constant derived from the authoritative source (spec table / CASES.json), mirroring the wire oracle's `ExpectedScenarioCount`. A silently-skipped run is red by construction, not by a CI grep step.

### 3. Unconditional CI placement → extend corpora.yml with a golden job
**Options:** extend corpora.yml / add to ci.yml's test job / new dedicated workflow
**Selected:** extend corpora.yml.
corpora.yml is already corpus-aware (fetch + assert + nscloud cache), path-filtered, and `contents: read` (safe side of the cache-trust line). The golden job runs unconditionally and fails loudly on fetch/cache failure. ci.yml's test job unchanged.

## Claude's Discretion (recorded)
- Exact expected-count value and its display (constant vs derived)
- Whether prior RED demos are re-mutated or cited
- Job name and trigger wiring within corpora.yml

## Deferred
- Re-mutation vs cited evidence for hermetic/coverage families → plan records
- Phase 5's in-tree sweep → waits on this phase

---

*All decisions captured in 03-CONTEXT.md (D-01 through D-03).*
