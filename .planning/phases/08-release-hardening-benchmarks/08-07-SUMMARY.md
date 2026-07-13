---
phase: 08-release-hardening-benchmarks
plan: 07
subsystem: infra
tags: [benchmarking, ci-gate, os-exec, real-repos, tree-sitter, pebble]

# Dependency graph
requires:
  - phase: 08-release-hardening-benchmarks (plan 02)
    provides: "internal/bench.PeakRSSBytes / internal/bench.Metrics — the measurement core this runner drives"
  - phase: 08-release-hardening-benchmarks (plan 03)
    provides: "tools/bench/gencorpus.Generate / ProductionFileCount — the deterministic synthetic corpus this runner materializes via subprocess"
  - phase: 08-release-hardening-benchmarks (plan 06)
    provides: "internal/bench.CheckRegression — the tolerance-band + absolute-ceiling gate this runner's regression mode calls"
provides:
  - "tools/bench/realcorpus.Corpora() / Entry.Resolve() — pinned real-repo manifest (weft-go, colbymchenry-codegraph, cockroachdb-pebble) with full provenance"
  - "tools/bench/runner — CLI with -mode {headtohead,regression}, -rebless, -ceiling-bytes, median-of-5 measurement, real Go-vs-TS head-to-head + PERF-02/INDX-06 gate"
  - "tools/bench/baseline.json — the committed initial regression baseline, real runner output at the full 120000-file production scale"
affects: [08-08-ci-gate-workflows]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Package-main tools (gencorpus) are invoked as built subprocesses rather than imported, since Go disallows importing a package named main — confirmed empirically ('is a program, not an importable package') before designing around it"
    - "Self-measured file/byte counts (independent directory walk) rather than trusting either subject binary's own self-reported stats, so throughput numbers stay subject-agnostic and comparable across Go and TS"
    - "Per-(entry,subject) isolated work directories (copyTree into a fresh temp dir) so the Go binary's Pebble store and the TS binary's SQLite store never collide, and neither writes .codegraph/ into a real sibling-checkout directory the operator owns"
    - "Real repos resolved via env-var override -> sibling-checkout convention -> pinned-commit shallow clone (never HEAD), reusing testdata/golden's own CODEGRAPH_WEFT_CORPUS pattern"

key-files:
  created:
    - tools/bench/realcorpus/manifest.go
    - tools/bench/realcorpus/manifest_test.go
    - tools/bench/runner/main.go
    - tools/bench/baseline.json
    - tools/bench/BASELINE.md
  modified:
    - .gitignore

key-decisions:
  - "Reused testdata/golden/README.md's own D-06a pinned commits for weft-go and colbymchenry-codegraph instead of introducing a second, drifting pin for the same two repos — the plan's example names (spf13/cobra, pallets/flask) don't actually exist as pinned corpora in this repo; the real, already-established pins were used instead"
  - "Chose cockroachdb/pebble@v2.1.6 (dbdc1acb859689dc4237b40ef8fcdbb877526a84, BSD-3-Clause) as the 'larger real repo' entry — this project's own Pebble storage dependency, exercising PERF-01 at scale beyond weft-go's compact 84 files"
  - "gencorpus is package main and cannot be imported (verified empirically); the regression mode builds and shells out to it as a subprocess instead of the plan's literal 'calls gencorpus.Generate' phrasing — same no-network, no-repo-clone guarantee (D-04), different call mechanism"
  - "Established the committed baseline.json at the FULL 120000-file production scale (not a smaller stand-in) — a real end-to-end run took only ~8s per from-scratch index on the capture machine, so no like-for-like scale mismatch exists between the committed baseline and the eventual CI gate"
  - "TS CLI's init has no --quiet flag (unlike its index) — dropped --quiet from the init warmup call for both subjects rather than special-casing per-subject flags, since runOnce already discards child stdout/stderr regardless"

patterns-established:
  - "Symlinks are skipped (not resolved) during corpus copyTree — a symlinked directory reports IsDir()==false via WalkDir's Lstat-based DirEntry, which would otherwise make copyFile wrongly try to open it as a regular file"
  - "Per-entry-error resilience in copyTree (log + skip via filepath.SkipDir/nil, never abort the whole walk) so one broken symlink or permission-denied file in a real, uncurated OSS repo doesn't kill an entire head-to-head run"

requirements-completed: [PERF-01, PERF-02, INDX-06]

coverage:
  - id: D1
    description: "The runner's headtohead mode shells out to both the freshly-built Go binary and the installed TS codegraph@1.3.1 over all three pinned real repos, capturing per-subject Metrics (throughput, query latency, peak RSS, cold start) via a real end-to-end run"
    requirement: "PERF-01"
    verification:
      - kind: e2e
        ref: "go run ./tools/bench/runner -mode headtohead (manual verification run, this session — produced real JSON metrics for weft-go/colbymchenry-codegraph/cockroachdb-pebble against both go and ts subjects)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The runner's regression mode materializes the synthetic corpus (network-free), runs the Go binary, and gates the result against the committed baseline via CheckRegression, exiting non-zero on a real induced regression"
    requirement: "PERF-02"
    verification:
      - kind: e2e
        ref: "go run ./tools/bench/runner -mode regression (manual verification run, this session — passed against its own baseline, and failed loudly with a descriptive error when the baseline was artificially inflated 10x)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The pinned real-repo manifest carries full provenance (SourceURL, CommitSHA, License, SelectionRule, QueryTerms) for every entry, asserted by a unit test"
    requirement: "PERF-01"
    verification:
      - kind: unit
        ref: "tools/bench/realcorpus/manifest_test.go#TestCorporaProvenanceComplete"
        status: pass
    human_judgment: false
  - id: D4
    description: "A real, committed baseline.json exists at the full 120000-file INDX-06 production scale, produced by an actual runner invocation (not fabricated), and the gate passes cleanly against it on a second real run"
    requirement: "INDX-06"
    verification:
      - kind: e2e
        ref: "go run ./tools/bench/runner -mode regression -rebless (baseline capture) followed by a second non-rebless run against the same seed/count (this session)"
        status: pass
    human_judgment: false

# Metrics
duration: 2h45min
completed: 2026-07-13
status: complete
---

# Phase 08 Plan 07: Benchmark Runner — Head-to-Head + Regression Gate Summary

**`tools/bench/runner` drives both the Go and installed TS codegraph@1.3.1 binaries over three real, commit-pinned repos (weft-go, colbymchenry-codegraph, cockroachdb/pebble) for PERF-01, and gates the deterministic 120k-file synthetic corpus against a real, committed `baseline.json` for PERF-02/INDX-06 — every number in this plan's artifacts is genuine runner output, verified end to end in this session.**

## Performance

- **Duration:** 2h 45min
- **Started:** 2026-07-13T14:33:27-04:00 (Task 1 commit)
- **Completed:** 2026-07-13T17:16:18-04:00 (Task 3 commit)
- **Tasks:** 3 completed
- **Files modified:** 6 (5 created, 1 modified)

## Accomplishments
- `tools/bench/realcorpus.Corpora()` pins three real repos by full 40-char commit SHA — weft-go (Apache-2.0, reusing testdata/golden's own pin), colbymchenry-codegraph (MIT, ditto — the original TS project this Go port replaces), and cockroachdb/pebble@v2.1.6 (BSD-3-Clause, the larger-scale entry, and this project's own storage dependency)
- Every manifest entry carries SourceURL, CommitSHA, License, SelectionRule, and a fixed `QueryTerms` set of real symbol names confirmed present at the pinned commit (`Load`/`Install` for weft-go, `ExtractionOrchestrator`/`TreeSitterExtractor` for colbymchenry-codegraph, `Open`/`DB` for pebble) — so the query-latency metric always exercises a genuine, non-empty lookup
- `Entry.Resolve()` checks an env-var override then a conventional sibling-checkout directory before reporting `ErrNeedsClone`; the runner itself shallow-clones at the exact pinned commit (never HEAD) when neither is present — verified with a real network clone of cockroachdb/pebble in this session
- `tools/bench/runner` implements both modes end to end: `-mode headtohead` (real Go-vs-TS numbers over all three pinned repos, verified in this session) and `-mode regression` (real synthetic-corpus run gated via `bench.CheckRegression`, verified both passing and correctly failing on an induced regression)
- Peak RSS is always read via `bench.PeakRSSBytes(cmd.ProcessState)` on the completed child process — `grep -q runtime.MemStats` finds no match anywhere in the runner
- `-rebless` is the only code path that writes `baseline.json`; a normal gating run never mutates it
- `tools/bench/baseline.json` is real runner output at the **full 120000-file production scale** (INDX-06's own 100k+ requirement, not a smaller stand-in) — a from-scratch index of the full corpus took only ~8s on the capture machine, so the committed baseline and the eventual CI gate compare like-for-like from day one

## Task Commits

Each task was committed atomically:

1. **Task 1: Pinned real-repo corpus manifest** - `42a8057` (feat)
2. **Task 2: The runner CLI — headtohead + regression modes, median-of-5, OS-level RSS** - `2be0618` (feat)
3. **Task 3: Establish and commit the initial baseline.json from a first real regression run** - `e7aa091` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified
- `tools/bench/realcorpus/manifest.go` - `Entry`/`Corpora()`/`Resolve()`/`ErrNeedsClone` — the pinned real-repo manifest
- `tools/bench/realcorpus/manifest_test.go` - `TestCorporaProvenanceComplete`, `TestReferencesExistingGoldenCorpora`
- `tools/bench/runner/main.go` - the full CLI: flag parsing, shared `runOnce`/`medianOfN` measurement primitive, `runHeadToHead`/`measureSubject`/`resolveOrClone`, `runRegression`/`generateSyntheticCorpus`/`readBaseline`/`writeBaseline`, `copyTree`/`countTree`/`buildGoBinary`/`repoRootDir` helpers
- `tools/bench/baseline.json` - the committed initial baseline (seed 42, 120000 files)
- `tools/bench/BASELINE.md` - capture provenance + re-bless command documentation
- `.gitignore` - added `/runner` (stray dev-binary path, mirrors the existing `/codegraph` and `/gencorpus` entries)

## Decisions Made
- Reused the project's own already-pinned real repos (weft-go, colbymchenry-codegraph via testdata/golden/README.md's D-06a table) rather than the plan's example names (spf13/cobra, pallets/flask), which don't exist as pinned corpora anywhere in this repo — see Deviations.
- Chose cockroachdb/pebble as the "larger real repo" entry: a real, substantially bigger Go codebase that's also this project's own direct dependency, so its provenance/relevance is self-evident.
- gencorpus is invoked as a built subprocess, not imported — Go disallows importing a package named `main` (verified empirically before committing to this design).
- Established the committed baseline at the full 120000-file scale rather than a smaller documented stand-in, since a real run proved fast enough (~8s/index) to make the smaller-scale caveat in the plan's acceptance criteria unnecessary.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `codegraph index --force` requires .codegraph/ to already exist; `codegraph init` has no `--force` flag on either binary**
- **Found during:** Task 2 (first end-to-end smoke test of regression mode)
- **Issue:** The initial implementation passed `--force` to both `init` and `index`. The Go CLI's `init` command has no `--force` flag at all (only `--quiet`/`--verbose`); it's `index --force` (not `init --force`) that supports the flag. Running `init workDir --force --quiet` failed with exit status 1.
- **Fix:** Removed `--force` from the warmup `init` call for both subjects; `init` only needs to run once against a fresh (non-existent) `.codegraph/`, so no force flag is needed there. The measured `index --force` call (which does require and use `--force`) was unaffected.
- **Files modified:** `tools/bench/runner/main.go`
- **Verification:** `go run ./tools/bench/runner -mode regression -count 300 -rebless` completed successfully after the fix.
- **Committed in:** `2be0618` (Task 2 commit)

**2. [Rule 1 - Bug] TS CLI's `init` has no `--quiet` flag (unlike its `index`)**
- **Found during:** Task 2 (headtohead smoke test against the installed TS binary)
- **Issue:** `codegraph init <path> --quiet` failed against the real TS 1.3.1 binary with `error: unknown option '--quiet'` — TS's `init --help` lists only `-i/--index`, `-f/--force`, `-v/--verbose`. Only TS's `index` subcommand supports `-q/--quiet`.
- **Fix:** Dropped `--quiet` from the warmup `init` call entirely for both subjects (the Go CLI does support it on `init`, but dropping it keeps one code path instead of branching per subject) — `runOnce` already discards the child's stdout/stderr unconditionally (nil `Cmd.Stdout`/`Stderr` → `/dev/null`), so output suppression was unnecessary regardless.
- **Files modified:** `tools/bench/runner/main.go`
- **Verification:** A subsequent real headtohead run against the installed TS binary completed successfully for all three repos.
- **Committed in:** `2be0618` (Task 2 commit)

**3. [Rule 1 - Bug] `copyTree` crashed on a symlinked directory in a real OSS repo**
- **Found during:** Task 2 (headtohead smoke test against cockroachdb/pebble, a real, uncurated repo)
- **Issue:** `internal/mkbench/testdata/data-symlink` in pebble's real tree is a symlink to a directory. `filepath.WalkDir`'s `fs.DirEntry.IsDir()` is Lstat-based and reports `false` for a symlink regardless of its target, so `copyTree` treated it as a regular file and called `os.Open` on it, which failed with "is a directory".
- **Fix:** Added an explicit symlink check (`d.Type()&fs.ModeSymlink != 0`) that skips symlinks entirely (rather than resolving them, which risks escaping `src`), plus a broader `!d.Type().IsRegular()` guard, and made per-entry walk errors resilient (log + skip via `filepath.SkipDir`/`nil`) instead of aborting the whole copy — a real-world repo can contain permission-denied or broken-link entries that shouldn't kill an entire head-to-head run.
- **Files modified:** `tools/bench/runner/main.go`
- **Verification:** The full headtohead run against all three pinned real repos (including pebble) completed successfully after the fix, producing real per-repo/per-subject metrics for all six (repo × subject) combinations.
- **Committed in:** `2be0618` (Task 2 commit)

**4. [Rule 2 - Missing Critical] Added `/runner` to `.gitignore`**
- **Found during:** Task 2 verification (`go build ./tools/bench/runner/` without `-o` wrote a stray 3.5 MB binary to repo root)
- **Issue:** Same class of hygiene gap Plan 08-03 already fixed for `gencorpus` — an unignored stray dev binary would show up as an untracked file in every future `git status`.
- **Fix:** Added `/runner` to `.gitignore`, mirroring the existing `/codegraph` and `/gencorpus` entries.
- **Files modified:** `.gitignore`
- **Verification:** `git status --short` clean after the fix.
- **Committed in:** `2be0618` (Task 2 commit)

**5. [Rule 4-adjacent, resolved without a stop] Plan's example real-repo names don't exist in this repo**
- **Found during:** Task 1 (writing the manifest)
- **Issue:** The plan's action text named `spf13/cobra` and `pallets/flask` as "the already-pinned in-repo corpora" to reuse. Neither exists anywhere in this repo — `testdata/golden/corpus/` actually contains `weft-go` and `colbymchenry-codegraph` (per `testdata/golden/README.md`'s own D-06a table), and those are golden JSON *output* fixtures, not vendored source trees.
- **Resolution:** Used the manifest's own acceptance criterion ("references the existing testdata/golden/corpus entries by their recorded pinned commit") as the authoritative instruction over the plan's incorrect example names, and pinned weft-go + colbymchenry-codegraph at their real, already-established commits instead. This is a same-effort substitution of concrete facts for an apparent plan-authoring error, not an architectural change, so it was resolved inline rather than raised as a Rule 4 checkpoint.
- **Files modified:** `tools/bench/realcorpus/manifest.go`
- **Verification:** `tools/bench/realcorpus/manifest_test.go#TestReferencesExistingGoldenCorpora` asserts the manifest's commits match `testdata/golden/README.md`'s own table.
- **Committed in:** `42a8057` (Task 1 commit)

---

**Total deviations:** 5 auto-fixed (3 bug fixes surfaced by real end-to-end runs against real binaries/repos, 1 hygiene fix, 1 factual correction to the plan's example data)
**Impact on plan:** All fixes were necessary for the runner to actually work against the real TS binary and real, uncurated OSS repos — none change the plan's architecture or scope. No scope creep.

## Issues Encountered
None beyond the deviations documented above — every issue surfaced during real end-to-end verification runs (not left undiscovered) and was fixed and re-verified within this plan.

## User Setup Required

None - no external service configuration required. The installed TS `codegraph@1.3.1` binary at `/opt/homebrew/bin/codegraph` (used for headtohead mode) was already present in this environment; no action was needed to make it available.

## Next Phase Readiness
- `tools/bench/runner` is ready for Plan 08-08 to wire into `.github/workflows/ci.yml` (regression mode, blocking) and `.github/workflows/bench.yml` (headtohead mode, on-demand/scheduled, not blocking)
- The committed `tools/bench/baseline.json` is real, full-scale (120000 files) output — Plan 08-08's CI gate can run against it immediately without needing a separate re-bless step, though re-blessing on actual CI runner hardware (per `tools/bench/BASELINE.md`) is recommended once that hardware is provisioned, since `internal/bench.CheckRegression`'s tolerance bands assume a consistent measurement host
- The default `-ceiling-bytes` (4 GiB) is a documented starting point comfortably above the ~803 MiB peak RSS observed at capture time; Plan 08-08 or a later hardening pass should retune it against real CI hardware numbers
- No blockers

---
*Phase: 08-release-hardening-benchmarks*
*Completed: 2026-07-13*

## Self-Check: PASSED
- FOUND: tools/bench/realcorpus/manifest.go
- FOUND: tools/bench/realcorpus/manifest_test.go
- FOUND: tools/bench/runner/main.go
- FOUND: tools/bench/baseline.json
- FOUND: tools/bench/BASELINE.md
- FOUND: .gitignore (modified)
- FOUND: 42a8057 (Task 1 commit)
- FOUND: 2be0618 (Task 2 commit)
- FOUND: e7aa091 (Task 3 commit)
