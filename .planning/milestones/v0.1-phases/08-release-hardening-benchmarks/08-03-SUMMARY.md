---
phase: 08-release-hardening-benchmarks
plan: 03
subsystem: infra
tags: [benchmarking, corpus-generation, tdd, math-rand, ci-gate]

# Dependency graph
requires:
  - phase: 08-release-hardening-benchmarks (plan 02)
    provides: "internal/bench.Metrics / PeakRSSBytes shape the runner (08-07) will populate against this corpus"
provides:
  - "tools/bench/gencorpus.Generate(Options) (Stats, error) — deterministic, network-free 100k+ file corpus writer"
  - "tools/bench/gencorpus main() CLI (-seed/-out/-count)"
  - "ProductionFileCount=120000 default sizing INDX-06's 100k+ requirement"
affects: [08-06-regression-gate, 08-07-head-to-head-runner]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Single rand.New(rand.NewSource(seed)) threaded through all generation functions in a fixed iteration order (package index then file index, never map order) — the reproducibility contract a committed CI baseline depends on"
    - "Pure prediction function (PlannedFileCount) split from the I/O-performing Generate so a scale assertion can run cheaply every test invocation while the expensive full materialization is gated behind testing.Short()"
    - "Language population weighting (85% Go / 10% Python / 5% JS) with cross-file/cross-package reference wiring scoped only to the dominant Go population, per RESEARCH Pattern 2's 'zero-edge files understate indexing cost' guidance"

key-files:
  created:
    - tools/bench/gencorpus/gen.go
    - tools/bench/gencorpus/gen_test.go
    - tools/bench/gencorpus/main.go
  modified:
    - .gitignore

key-decisions:
  - "Cross-file reference wiring (imports + qualified calls) implemented only for the Go population, not Python/JS — Go is the population TestHasCrossFileRefs asserts against and this project's first-priority extractor language; Python/JS exist purely for extractor-registry language diversity without complicating the one tested reference chain"
  - "Added /gencorpus to .gitignore (mirrors existing /codegraph dev-binary entry) after an errant `go build ./tools/bench/gencorpus/` (missing -o) during verification wrote a 2.6MB binary to repo root — prevents that from ever being accidentally committed"

patterns-established:
  - "gencorpus's package/file naming (pkgNNNN/FnPPPP_FFFF) is deterministic from loop indices only, never from rand.Rand for structural identity — only the numeric literal inside each function body is seed-derived, keeping path/name determinism trivially provable independent of RNG behavior"

requirements-completed: [PERF-02, INDX-06]

coverage:
  - id: D1
    description: "Generate(opts) with a fixed seed materializes a byte-identical directory tree every run (same file count, paths, and content hashes); a different seed produces a different tree"
    requirement: "PERF-02"
    verification:
      - kind: unit
        ref: "tools/bench/gencorpus/gen_test.go#TestDeterministic"
        status: pass
    human_judgment: false
  - id: D2
    description: "The production corpus (ProductionFileCount=120000) exceeds INDX-06's 100k+ file requirement, both as a pure planned-count check and via full materialization"
    requirement: "INDX-06"
    verification:
      - kind: unit
        ref: "tools/bench/gencorpus/gen_test.go#TestFileCountExceeds100k"
        status: pass
    human_judgment: false
  - id: D3
    description: "Generated files contain real cross-file references (imports + qualified calls between generated symbols), not zero-edge files"
    requirement: "PERF-02"
    verification:
      - kind: unit
        ref: "tools/bench/gencorpus/gen_test.go#TestHasCrossFileRefs"
        status: pass
    human_judgment: false
  - id: D4
    description: "The generator builds and runs as a standalone CLI accepting -seed/-out/-count, uses a seeded (never global/crypto) RNG, and contains no panics"
    verification:
      - kind: unit
        ref: "go build ./tools/bench/gencorpus/ && manual CLI invocation (-seed 7 -out /tmp/... -count 300)"
        status: pass
    human_judgment: false

# Metrics
duration: 35min
completed: 2026-07-13
status: complete
---

# Phase 08 Plan 03: Synthetic Corpus Generator Summary

**Deterministic, network-free `tools/bench/gencorpus` generator that materializes a 120k-file, cross-referencing Go/Python/JS monorepo from a single seeded `math/rand` source — proven byte-identical per seed via RED-first TDD.**

## Performance

- **Duration:** 35 min
- **Started:** 2026-07-13T13:16:59-04:00 (RED commit)
- **Completed:** 2026-07-13T13:51:09-04:00 (GREEN commit)
- **Tasks:** 2 completed
- **Files modified:** 4 (3 created, 1 modified)

## Accomplishments
- `Generate(opts Options) (Stats, error)` deterministically writes `opts.FileCount` source files (Go/Python/JS, 85/10/5 weighted) under `opts.OutDir`, driven entirely by `rand.New(rand.NewSource(opts.Seed))`
- Go population wires real cross-file and cross-package reference edges: each package's first file imports and calls the previous package's last exported function; every other file calls its same-package predecessor — never a zero-edge corpus
- `ProductionFileCount = 120000` comfortably clears INDX-06's literal 100k+ requirement even after accounting for `internal/indexer.Discover`'s own skip logic (vendor/, dot-dirs, build tags)
- Standalone CLI (`main.go`) with `-seed`/`-out`/`-count` flags, defaulting to seed 42 and the production file count
- TDD gate proven in git history: `test(08-03)` commit fails to compile (RED — `Generate`/`Options`/`Stats` undefined), `feat(08-03)` commit passes all three properties including the non-short 120k materialization test (~12s)

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): Write failing determinism + scale + cross-reference tests** - `0ee362c` (test)
2. **Task 2 (GREEN): Implement the deterministic generator + CLI wrapper** - `34dc26b` (feat)

**Plan metadata:** (this commit)

_Note: no REFACTOR commit — the GREEN implementation needed no cleanup pass._

## Files Created/Modified
- `tools/bench/gencorpus/gen_test.go` - `TestDeterministic` (tree-hash equality across two runs, same seed; inequality for a different seed), `TestFileCountExceeds100k` (pure `PlannedFileCount` check always runs; full materialization gated behind `testing.Short()`), `TestHasCrossFileRefs` (asserts real cross-package import + qualified-call edges exist)
- `tools/bench/gencorpus/gen.go` - `Options`/`Stats`/`Generate`/`PlannedFileCount`/`ProductionFileCount`, plus `generateGo`/`generatePython`/`generateJS`/`writeGoModule`/`writeFile` — the deterministic, seeded, cross-referencing corpus writer
- `tools/bench/gencorpus/main.go` - `main()` CLI wrapper parsing `-seed`/`-out`/`-count`
- `.gitignore` - added `/gencorpus` (stray dev-binary path, mirrors the existing `/codegraph` entry)

## Decisions Made
- Scoped cross-file/cross-package reference generation to the Go population only (see key-decisions above) — Python/JS contribute language diversity for the extractor registry without complicating the one tested reference chain.
- Added `/gencorpus` to `.gitignore` after verification accidentally produced a 2.6MB binary at repo root via `go build ./tools/bench/gencorpus/` (flags-after-package-path invocation, no `-o`) — a Rule 1/Rule 2 style hygiene fix to prevent that artifact from ever landing in a real commit.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added `/gencorpus` to `.gitignore`**
- **Found during:** Task 2 verification (manual CLI smoke test)
- **Issue:** Running `go build ./tools/bench/gencorpus/` without `-o` (or with `-o` positioned incorrectly for the shell) writes a binary literally named `gencorpus` to the current working directory — the repo root in this session. Left unignored, this stray binary would show up as an untracked file in every future `git status` and risk being accidentally committed.
- **Fix:** Removed the stray binary and added `/gencorpus` to `.gitignore`, mirroring the project's existing `/codegraph` dev-binary entry.
- **Files modified:** `.gitignore`
- **Verification:** `git status --short` shows a clean tree after the fix; the entry follows the same convention as `/codegraph`.
- **Committed in:** `34dc26b` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 hygiene fix, Rule 2 — prevents an accidental future commit of a build artifact)
**Impact on plan:** Cosmetic/hygiene only — no behavioral change to the generator itself. No scope creep.

## Issues Encountered
None beyond the stray-binary hygiene fix documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- `tools/bench/gencorpus` is ready for Plan 08-07's runner to invoke (`gencorpus -seed 42 -out <scratch-dir>` or `Generate` called in-process) against `internal/indexer` for the PERF-02 CI regression gate and the INDX-06 100k+ file / bounded-memory assertion
- The generated corpus is intentionally NOT committed to git — only the generator is (per the plan's explicit prohibition); regeneration happens at CI/benchmark time into a scratch directory
- No blockers

---
*Phase: 08-release-hardening-benchmarks*
*Completed: 2026-07-13*

## Self-Check: PASSED
- FOUND: tools/bench/gencorpus/gen.go
- FOUND: tools/bench/gencorpus/gen_test.go
- FOUND: tools/bench/gencorpus/main.go
- FOUND: 0ee362c (Task 1 commit)
- FOUND: 34dc26b (Task 2 commit)
