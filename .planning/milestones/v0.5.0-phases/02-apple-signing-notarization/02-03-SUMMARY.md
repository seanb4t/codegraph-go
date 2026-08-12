---
phase: 02-apple-signing-notarization
plan: 03
subsystem: testing
tags: [go-test, TestMain, integration-harness, wire-oracle, env-var-resolver, tdd]

# Dependency graph
requires:
  - phase: 02-01
    provides: "the phase's evidence-line schema convention and the RED-baseline discipline this plan's negative proof follows"
provides:
  - "resolveTestBinPath — a pure, unit-tested CODEGRAPH_TEST_BIN resolver in test/integration, with a deliberate second copy in test/wireoracle (Go test helpers are not importable across packages)"
  - "Both TestMain functions consume the resolver: unset variable is behaviorally unchanged (same build/temp-dir/cleanup), a valid override skips the build entirely, an invalid override aborts by name before any test result line prints"
  - "A recorded, empirical criterion-4 scope verdict for test/wireoracle: IN SCOPE — a release-shaped binary (real -X .../version.Version=<tag> ldflags) passed all 27 frozen scenarios via the override, because normalize.go's serverVersion rule already erases the differing version field"
  - "The seam plan 02-06's post-release CI job will point at the re-downloaded, notarized published binary (D-10)"
affects: [02-06, 02-07]

# Actuals (#2632)
actuals:
  tokens: 5936
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Pure two-outcome resolver (path,useEnv,err) with no third outcome — no os.Getenv/os.Exit/writes inside the resolver itself, so it is directly table-testable; TestMain is the sole caller that touches the environment or exit code"
    - "Deliberate per-package second copy of a ~20-line test helper (Go test helpers are not importable across packages), matching the repo's existing four-copy runGit* precedent — documented in-comment with the extraction trigger (a third harness) rather than pre-emptively generalized into a shared internal/testbin package"
    - "Empirical scope verdicts over reasoned ones: 'is this harness usable against a release binary' was answered by building a release-shaped binary and running the suite, not by inspecting normalize.go and asserting it should work"

key-files:
  created:
    - test/integration/binpath_test.go
    - test/wireoracle/binpath_test.go
  modified:
    - test/integration/main_test.go
    - test/wireoracle/main_test.go

key-decisions:
  - "GOFLAGS sabotage proof required precompiling the test binary first (go test -c), then running it directly — GOFLAGS=-mod=vendor breaks go test's OWN compile step too, not just TestMain's internal go build, so running `go test` directly under a hostile GOFLAGS would have proven nothing distinguishable."
  - "The macOS /var → /private/var symlink made a naive filepath.Join(preChdirDir, name) comparison in the relative-path subtest fail even though the resolver was correct; fixed the test assertion to derive its expectation via the same os.Getwd()-based filepath.Abs() mechanism the resolver itself uses, rather than joining a path captured before Chdir."
  - "wireoracle IS in scope for criterion 4, established empirically: a binary built with the same -X .../internal/version.Version=<tag> ldflags .goreleaser.yaml injects at release time, pointed at via the override, passed TestFrozenTranscriptsMatch's all 27 scenarios (including toolslist-repeat, the version-bearing tools/list response) with zero transcript changes — normalize.go's serverVersion rule already normalizes the differing version field before comparison."

patterns-established:
  - "CODEGRAPH_TEST_BIN environment-variable seam: set → use that exact file or abort by name (never a build fallback); unset → existing build path, byte-for-byte unchanged statements."

requirements-completed: [SIGN-02]

coverage:
  - id: D1
    description: "test/integration/main_test.go's TestMain consumes a pure resolveTestBinPath: unset variable behaves exactly as before (same go build, same temp dir, same cleanup — git diff shows those statements untouched); a valid override skips the build and spawns tests against the named binary; an invalid override aborts by name (message names both CODEGRAPH_TEST_BIN and the offending path) before creating a temp dir, before building, and before any test runs."
    requirement: SIGN-02
    verification:
      - kind: unit
        ref: "test/integration/binpath_test.go#TestResolveTestBinPath"
        status: pass
      - kind: integration
        ref: "go test ./test/integration/ -count=1 (variable unset) — ok, 6.514s"
        status: pass
      - kind: integration
        ref: "positive-identification shim proof: CODEGRAPH_TEST_BIN=<shim> go test ./test/integration/ -count=1 — ok, shim invocation log records 22 real subprocess spawns"
        status: pass
      - kind: integration
        ref: "GOFLAGS sabotage proof: precompiled test binary run directly with GOFLAGS=-mod=vendor (independently confirmed to break any go build) and CODEGRAPH_TEST_BIN set — PASS, proving no go build occurred"
        status: pass
      - kind: integration
        ref: "negative end-to-end proof: CODEGRAPH_TEST_BIN=/nonexistent/codegraph go test ./test/integration/ -count=1 — exit 1, stderr names the variable and the path, zero === RUN/--- PASS/--- FAIL lines"
        status: pass
    human_judgment: false
  - id: D2
    description: "test/wireoracle carries a deliberate second copy of the same resolver/TestMain (Go test helpers are not importable across packages), the same table test, and an in-file recorded scope verdict: IN SCOPE for criterion 4, proven by running the suite against a release-shaped binary rather than reasoning about it."
    requirement: SIGN-02
    verification:
      - kind: unit
        ref: "test/wireoracle/binpath_test.go#TestResolveTestBinPath"
        status: pass
      - kind: integration
        ref: "go test ./test/wireoracle/ -count=1 (variable unset) — ok, 19.340s (one pre-existing, documented toolslist-repeat ordering flake observed under full-suite load, isolated to PASS on re-run, unrelated to this plan's files)"
        status: pass
      - kind: integration
        ref: "CODEGRAPH_TEST_BIN=<release-shaped binary, -X .../version.Version=v0.5.1> go test ./test/wireoracle/ -run TestFrozenTranscriptsMatch -count=1 -v — PASS, all 27 scenarios including toolslist-repeat"
        status: pass
      - kind: unit
        ref: "task test:unit — full suite ok across all packages including test/integration and test/wireoracle"
        status: pass
    human_judgment: false

duration: ~45min
completed: 2026-08-09
status: complete
---

# Phase 2 Plan 03: Real-Binary Test Harness Resolver Seam Summary

**Added a pure, two-outcome `CODEGRAPH_TEST_BIN` resolver to both `test/integration` and `test/wireoracle`, proved it cannot silently fall back to a local rebuild, and established by running it (not reasoning about it) that the wireoracle wire-oracle suite is usable against a release-shaped binary — no transcript changes required.**

## Performance

- **Duration:** ~45 min
- **Started:** 2026-08-09
- **Completed:** 2026-08-09
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- `resolveTestBinPath` in `test/integration/main_test.go`: a pure function with exactly two outcomes for a non-empty input — `(path, true, nil)` (use the resolved absolute path) or `("", false, err)` (abort by name) — never a third outcome, so a bad override can never silently fall through to a local `go build`.
- `TestMain` rewritten to call the resolver first: unset variable is behaviorally identical to before (same `go build` command, same temp-dir lifecycle, same cleanup — confirmed via `git diff`); a valid override skips the build entirely and runs every test against the named binary; an invalid override prints a message naming `CODEGRAPH_TEST_BIN` and the offending path to stderr and exits non-zero before creating a temp dir, before building, and before running a single test.
- `test/integration/binpath_test.go`: a 6-subtest table test over every documented resolver behavior, plus an inline "no third outcome" invariant check per case.
- The exact same seam mirrored into `test/wireoracle` as a deliberate second copy (Go test helpers are not importable across packages — matches the repo's existing four-copy `runGit*` precedent), with the extraction trigger ("a third harness needing this seam") recorded in-comment.
- Established, by running it rather than reasoning about it, that `test/wireoracle` IS in scope for ROADMAP criterion 4: a binary built with the same `-X .../internal/version.Version=<tag>` ldflags `.goreleaser.yaml` injects at release time passed `TestFrozenTranscriptsMatch`'s all 27 frozen scenarios via the override, with zero transcript changes needed — `normalize.go`'s `serverVersion` rule already erases the differing version field before comparison. This verdict is recorded as an in-file comment in `test/wireoracle/main_test.go`.
- Four independent proofs recorded (see below): positive identification via an invocation-logging shim, a second independent proof via `GOFLAGS` build sabotage, a negative end-to-end abort-before-run proof, and unset-path behavioral equivalence via `git diff`.

## Task Commits

Each task was committed atomically:

1. **Task 1: The resolver and the integration harness seam** - `5280a7d` (feat)
2. **Task 2: Mirror the seam into the wireoracle harness and establish its release-binary suitability** - `5a820b8` (feat)

**Plan metadata:** this SUMMARY.md + REQUIREMENTS.md (worktree mode — STATE.md/ROADMAP.md excluded, orchestrator-owned)

## Files Created/Modified

- `test/integration/main_test.go` - added `testBinEnvVar` constant, `resolveTestBinPath`, and rewrote `TestMain` to call it first
- `test/integration/binpath_test.go` - new: `TestResolveTestBinPath` table test over all 6 documented behaviors
- `test/wireoracle/main_test.go` - the same additions as `test/integration`, plus an in-file criterion-4 scope verdict comment
- `test/wireoracle/binpath_test.go` - new: the mirrored table test

## Decisions Made

- **The `GOFLAGS` sabotage proof required precompiling the test binary first.** Running `go test` directly under a hostile `GOFLAGS` value breaks `go test`'s own compile step too, not just `TestMain`'s internal `go build` call — that would have proven nothing distinguishable (the whole invocation would fail regardless of whether the override skipped the internal build). Fixed by `go test -c -o <bin> ./test/integration/` first (clean environment), then running the precompiled binary directly with `CODEGRAPH_TEST_BIN` set and `GOFLAGS="-mod=vendor"` in the environment — this isolates the sabotage to exactly the internal `go build` call the override is supposed to skip.
- **`-mod=vendor` (not an unknown build tag) as the sabotage value.** An initial attempt using `GOFLAGS="-tags=nonexistent-tag"` exited 0 — an unrecognized build tag has no effect and doesn't fail a build. `-mod=vendor` against a repo with no `vendor/` directory reliably fails loudly (`inconsistent vendoring`), independently confirmed before using it as the sabotage value.
- **Fixed a self-inflicted test bug in the relative-path subtest** (Rule 1 — bug, found and fixed before the task's commit): on macOS, `/var` is a symlink to `/private/var`. `os.Getwd()` after `os.Chdir(dir)` returns the resolved `/private/var/...` form, but `dir` (captured before `Chdir`) does not carry that prefix — so `filepath.Abs(filepath.Join(dir, name))` disagreed byte-for-byte with the resolver's own `filepath.Abs(raw)` even though both point at the same file. Fixed by deriving the test's expectation via `filepath.Abs("valid-exec")` from within the chdir'd subtest — the same mechanism the resolver itself uses — rather than joining the pre-Chdir path.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Relative-path subtest assertion used a symlink-unstable expected-value derivation**
- **Found during:** Task 1, first `go test` run of the new table test (macOS)
- **Issue:** `TestResolveTestBinPath`'s relative-path subtest computed its expected absolute path via `filepath.Abs(filepath.Join(dir, "valid-exec"))` using the pre-`Chdir` `dir` value, which disagreed with the resolver's own `os.Getwd()`-based absolute path on macOS due to the `/var` → `/private/var` symlink — a test bug, not a resolver bug (the resolver's actual output was correct and pointed at the real file).
- **Fix:** Derive the expected value via `filepath.Abs("valid-exec")` called from inside the already-`Chdir`'d subtest, matching the resolver's own resolution mechanism exactly.
- **Files modified:** `test/integration/binpath_test.go` (and the mirrored fix applied directly when authoring `test/wireoracle/binpath_test.go`, since it was written after the fix was known)
- **Verification:** `go test ./test/integration/ -run TestResolveTestBinPath -v` — all 6 subtests pass.
- **Committed in:** `5280a7d` (part of Task 1's commit — caught and fixed before that commit, not a separate follow-up)

---

**Total deviations:** 1 auto-fixed (Rule 1 — a bug in newly-written test code, caught and fixed before its own commit)
**Impact on plan:** No scope creep; the underlying resolver was correct throughout.

## Issues Encountered

- **Worktree path discipline correction, mid-task, no functional impact.** Several of this session's earliest read-only `Bash`/`Read` calls used the main repository's absolute path (`.../codegraph-go/...`) rather than the worktree-prefixed path (`.../codegraph-go/.claude/worktrees/agent-addf14f530d920f9f/...`), because both currently contain byte-identical content at the shared base commit. Caught before any edit or commit: a `diff` confirmed no divergence, and every subsequent file operation (Read/Write/Edit, and all Bash commands, which default to the worktree cwd) used the correct worktree-scoped path. No file was read or written from the wrong location.
- **Pre-existing, unrelated test flake observed under full-suite load** (`test/wireoracle`'s `TestFrozenTranscriptsMatch/toolslist-repeat`): failed once during a full, unfiltered `go test ./test/wireoracle/ -count=1` run with a response-ordering mismatch (`id:2` vs `id:3`), then passed cleanly both in isolation (`-run TestFrozenTranscriptsMatch`) and on a second full-package re-run. This matches the exact, already-tracked flake in `STATE.md`'s Pending Todos ("Wire oracle `toolslist-repeat` response ordering flake — id-2 response overtaken by id-3 under parallel load"). Not caused by this plan's changes (neither modified file touches response ordering or scenario execution); not fixed (out of scope per the parallel-execution flake protocol); not newly logged (already a known, tracked class).

## Known Stubs

None.

## Next Phase Readiness

- The `CODEGRAPH_TEST_BIN` seam is live in both harnesses, ready for plan 02-06's post-release CI job to point at the re-downloaded, notarized published darwin binary (D-10) without any further harness changes.
- `test/wireoracle`'s criterion-4 scope verdict is settled and recorded in-file: **IN SCOPE**, so plan 02-06's CI job can run both `test/integration` and `test/wireoracle` against the same override target.
- No blockers for the next plan. No transcript re-freeze occurred and none is required.

## Self-Check: PASSED

- All 5 key files confirmed present on disk: `test/integration/main_test.go`, `test/integration/binpath_test.go`, `test/wireoracle/main_test.go`, `test/wireoracle/binpath_test.go`, this SUMMARY.md.
- Both task commit hashes confirmed present in `git log`: `5280a7d` (Task 1), `5a820b8` (Task 2).

---
*Phase: 02-apple-signing-notarization*
*Completed: 2026-08-09*
