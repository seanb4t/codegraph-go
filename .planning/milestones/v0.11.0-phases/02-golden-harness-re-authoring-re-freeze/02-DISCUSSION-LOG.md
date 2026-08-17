# Phase 2: Golden Harness Re-authoring & Re-freeze — Discussion Log

**Gathered:** 2026-08-14
**Mode:** autonomous `--interactive` (discuss inline, user answers all questions)

---

## Phase Boundary (from ROADMAP)

Re-author the golden suite as codegraph-go's own behavioral regression suite; remove the origin-driven capture path. Requirements: CODE-02, FIXT-04, FIXT-05, FIXT-06. Blocks Phase 3.

## Areas selected for discussion

All four gray areas (user-selected all):

1. New suite naming
2. Behavioral corpus layout
3. Re-freeze mechanism
4. Capture-path deletion scope

## Decisions by area

### 1. New suite naming → Behavioral vocabulary
**Options presented:** Behavioral vocabulary / Assertion-named tests / Keep 'golden', drop 'parity'
**Selected:** Behavioral vocabulary.
`parity_*_test.go` → `behavioral_*_test.go`; `TestGoldenParity*` → `TestCorpusBehavior*`/`TestBehavioral*`; `golden` survives only as the neutral fixture concept. Matches the ROADMAP goal's own word "behavioral".

### 2. Behavioral corpus layout → Move files + data case map
**Options presented:** Move files + data case map (CASES.json) / Move files + Go table / Rebuild in place
**Selected:** Move files + data case map.
Corpus moves from `testdata/golden/corpus/synthetic-parity/` (name is the framing to strip) to in-tree `corpus/behavioral/`; the four targeted cases stay intact; the case map is a committed data file `corpus/behavioral/CASES.json` that tests load — one readable source of truth, later authors edit data not code.

### 3. Re-freeze mechanism → Extend gocapture, all goldens, one diff
**Options presented:** Extend gocapture + all goldens + one diff / Per-family diffs / Keep gocapture as-is, narrow scope
**Selected:** Extend gocapture, all goldens, one reviewed diff.
Extend the existing Go-side capture path to cover every golden in scope; re-freeze all in ONE reviewed diff against the locked corpora. Rename lands first (no golden byte), re-freeze second (no identifier) — two separate reviewed diffs.

### 4. Capture-path deletion scope → Sweep all; realcorpus entry stays, Phase 6 reconciles
**Options presented:** Sweep all + realcorpus stays for Phase 6 / Sweep harness + resolve realcorpus now / Narrowest sweep
**Selected:** Sweep all harness references; `realcorpus`'s `colbymchenry-codegraph` entry stays, Phase 6 reconciles with the benchmark rewrite.
Keeps Phase 2's blast radius to the harness; avoids double-handling the bench manifest.

## Claude's Discretion (recorded)
- Exact per-test/file names within the behavioral vocabulary
- gocapture's internal extension structure
- Cosmetic comment text at rename time (rides the rename diff)

## Deferred
- `tools/bench/realcorpus` reconciliation → Phase 6
- `bench.yml` / `headtohead-*.json` references → Phase 6 (BENCH)
- `tools/bench/runner/main.go:482 pinnedAt()` HEAD-only validation → Phase 6 (filed in STATE.md todos)

---

*All decisions captured in 02-CONTEXT.md (D-01 through D-08).*
