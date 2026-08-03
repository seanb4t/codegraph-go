---
phase: 10-local-build-contribution-and-taskfile-yml-setup
plan: 02
subsystem: infra
tags: [taskfile, go-tool, govulncheck, github-actions, namespace-runners, ci, go-vet]

# Dependency graph
requires:
  - phase: 10-local-build-contribution-and-taskfile-yml-setup
    provides: "10-01's proven chain (go.tool.mod -> install-task -> Taskfile.yml -> Namespace runner -> task <target>), the Taskfile.yml file itself with default/check:goreleaser/lint:actions targets, and the parseX/mustX shape-guard idiom in internal/upgrade/taskfile_shape_test.go this plan extends"
provides:
  - Taskfile.yml build/test/vet/vuln target set — build, build:release, test:unit/golden/integration/daemon/race, vet, vet:windows, vet:daemon-windows, vuln — every one ported verbatim from ci.yml's test job, plus a resolution of the plan's flagged "which modfile hosts govulncheck" question (go.tool.mod, measured, no new modfile needed)
  - Contributor wrappers `task test` (5 host-only legs, serial via cmds: task: entries) and `task lint` (vet + lint:actions, serial) — never called by CI
  - internal/upgrade/taskfile_shape_test.go TestTaskfileGatesFailLoud (no status:/platforms: anywhere; every cross-toolchain task carries preconditions: with a non-empty msg:) and TestTaskfileWrapperIsSerial (test wrapper's leg set is sorted-set-equal to the D-10 fixture, no deps: key)
  - ci.yml's test job (required status check) rewired: every run: body is `task <target>` except the mingw-w64 apt-get install; runs on namespace-profile-linux-amd64-4x8; gains a new "Vet (go vet ./...)" step (DEV-01 new enforcement); all 6 required-check job names and the pre-change step-name set (plus exactly 3 new steps) verified byte-identical via sorted-set diff
affects: [10-03, 10-04, 10-05, 10-06, 10-07]

# Actuals (#2632)
actuals:
  tokens: 6756
  tasks: 3
  commits: 3

tech-stack:
  added:
    - golang.org/x/vuln/cmd/govulncheck v1.6.0 (added to the EXISTING go.tool.mod, not a new modfile — measured live: +2 net-new modules, no MVS conflict with task/goreleaser)
  patterns:
    - "Cross-toolchain gating via preconditions: with an actionable msg: only (never status:/platforms:) — enforced by a new shape-guard test, not just documented"
    - "Contributor wrapper = ordered cmds: list of '- task: <name>' entries (runs serially, in-process) rather than deps: (which go-task runs concurrently) — the mechanism that keeps test:daemon/test:race from overlapping"
    - "TZ=UTC + git's own --date=format-local formatter instead of GNU `date -d` for portable ldflags -X Date= construction across macOS/Linux contributor machines"

key-files:
  created: []
  modified:
    - Taskfile.yml
    - go.tool.mod
    - go.tool.sum
    - internal/upgrade/taskfile_shape_test.go
    - .github/workflows/ci.yml

key-decisions:
  - "govulncheck resolved to go.tool.mod on the FIRST attempt of the plan's measurement procedure (no need to try go.tool-lint.mod or a third go.tool-vuln.mod) — measured live: adding golang.org/x/vuln/cmd/govulncheck to go.tool.mod's tool() block adds exactly 2 net-new indirect modules (golang.org/x/telemetry, golang.org/x/vuln itself), builds clean alongside task v3.52.0 and goreleaser v2.17.1 with no MVS conflict. This resolves the plan's flagged_assumptions open question — the outcome, not a guess, is recorded here per the plan's own instruction."
  - "vet:daemon-windows's preconditions msg: was written as a single inline double-quoted string (matching RESEARCH.md/PATTERNS.md's own canonical example) rather than a YAML folded block scalar (>-) — the folded form is harder to regex-parse without reimplementing YAML folding logic, and TestTaskfileGatesFailLoud's parsePreconditionMessages needs to read the literal msg: value back off disk"
  - "build:release derives its Date ldflag via `TZ=UTC git log -1 --format=%cd --date=format-local:%Y-%m-%dT%H:%M:%SZ` rather than `date -u -d @<epoch>` (which ci.yml's reproducibility job uses) — the latter is GNU-only and would break on a contributor's macOS/BSD `date`; git's own date formatter is invoked identically on every platform"
  - "go vet ./... surfaced zero pre-existing findings on first run — DEV-01's 'new enforcement, expect findings' caveat did not require any fix-up work in this repo's current state"

patterns-established:
  - "Every Taskfile target carries a desc: explaining not just WHAT it runs but WHY it's separate (mirrors ci.yml's own comment-heavy convention) — `task --list` is a self-documenting command reference, not just a name list"

requirements-completed: [DEV-01]

coverage:
  - id: D1
    description: "Fine-grained build/test/vet/vuln Taskfile targets, each ported verbatim from ci.yml's test job; vet:daemon-windows gated with preconditions:+msg: (not status:/platforms:); build:release produces a git-describe-versioned local binary"
    requirement: "DEV-01"
    verification:
      - kind: other
        ref: "GOWORK=off go tool -modfile=go.tool.mod task vet test:golden test:integration (plan's own <verify>, exit 0)"
        status: pass
      - kind: other
        ref: "task build/test:unit/test:golden/test:integration/test:daemon/test:race/vet/vet:windows/vet:daemon-windows/vuln/build:release all run individually, exit 0 locally"
        status: pass
      - kind: other
        ref: "RED/GREEN non-vacuity: task vet — mutated Printf format-verb mismatch in a scratch file, task vet exits 201 naming the file; reverted, exit 0"
        status: pass
      - kind: other
        ref: "RED/GREEN non-vacuity: task vet:daemon-windows precondition — mingw-w64 removed from PATH, exit 201 with the install message; PATH restored, exit 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "Contributor wrappers task test / task lint run their legs SERIALLY (never concurrent deps:); TestTaskfileGatesFailLoud and TestTaskfileWrapperIsSerial guard against silent-skip fields and a drifting leg set, each demonstrated RED then GREEN"
    requirement: "DEV-01"
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/ -run 'TestTaskfileGatesFailLoud|TestTaskfileWrapperIsSerial' -v (4/4 subtests pass, including both edge-case tests)"
        status: pass
      - kind: other
        ref: "task --verbose test: 5 started/finished pairs in exact fixture order, zero interleaving (captured verbose log, not eyeballed)"
        status: pass
      - kind: other
        ref: "RED/GREEN non-vacuity: TestTaskfileGatesFailLoud — added platforms: [linux] to vet:daemon-windows, test fails naming that task; reverted, passes"
        status: pass
      - kind: other
        ref: "RED/GREEN non-vacuity (both directions): TestTaskfileWrapperIsSerial — removed test:race from the wrapper (fails, names the 4-element set missing test:race), then added an extra 'build' leg on top of the full 5 (fails, names the 6-element set); reverted, passes"
        status: pass
    human_judgment: false
  - id: D3
    description: "ci.yml's test job rewired: every run: body is task <target> except the mingw-w64 apt-get install; runs on namespace-profile-linux-amd64-4x8; new 'Vet (go vet ./...)' step added; job name test and the pre-change step-name set (+3 exactly) verified byte-identical"
    requirement: "DEV-01"
    verification:
      - kind: other
        ref: "GOWORK=off go tool -modfile=go.tool.mod task lint:actions (plan's own <verify>, exit 0 against the rewired workflow)"
        status: pass
      - kind: other
        ref: "sorted-set diff of all 6 required-status-check job name: values across every workflow file, before vs after this plan's commits — identical"
        status: pass
      - kind: other
        ref: "sorted-set diff of test job's step name: values, before vs after — pre-change set plus exactly 3 additions (Cache Go modules and build, Install Task, Vet (go vet ./...)), nothing dropped"
        status: pass
      - kind: other
        ref: "direct-parse check: every run: line in the rewired test job is a single 'task <target>' invocation except the one mingw-w64 apt-get line — no go test/go build/go vet direct invocation remains"
        status: pass
    human_judgment: true
    rationale: "Not verified and cannot be verified locally: a real pushed CI run actually scheduling the test job on namespace-profile-linux-amd64-4x8, and the new 'Vet (go vet ./...)' step observed failing on a deliberate vet finding pushed to a branch, then passing after removal. task vet's own RED/GREEN was demonstrated locally (see D1), but ci.yml's step invoking it end-to-end on a real Namespace runner needs a real push/PR — the same documented gap 10-01's coverage.D1/D2 left open for its two rewired jobs."

duration: unknown — session start time not captured (record_start_time step was not explicitly run before file-reading began); commit-to-commit span across all 3 tasks is ~7 min, but that excludes the preceding phase-context reading and live tool measurement (govulncheck modfile probe) that preceded the first commit
completed: 2026-08-01
status: complete
---

# Phase 10 Plan 2: ci.yml's test job onto task targets — build/test/vet/vuln Taskfile surface, serial contributor wrappers, DEV-01's go vet gate Summary

**Ported every `run:` body in `ci.yml`'s `test` job into `Taskfile.yml` verbatim, added serial contributor wrappers (`task test`/`task lint`) that CI never calls, closed the `go vet ./...` coverage gap as a new blocking CI step, and resolved the plan's open "where does govulncheck live" question by measurement: it fits cleanly in the existing `go.tool.mod`.**

## Performance

- **Duration:** unknown (not captured — see frontmatter `duration` note)
- **Completed:** 2026-08-01T20:58:45-04:00 (last task commit)
- **Tasks:** 3 / 3
- **Files modified:** 5 (0 created, 5 modified: `Taskfile.yml`, `go.tool.mod`, `go.tool.sum`, `internal/upgrade/taskfile_shape_test.go`, `.github/workflows/ci.yml`)

## Accomplishments

- `Taskfile.yml` gained 12 new targets: `build`, `build:release`, `test:unit`, `test:golden`, `test:integration`, `test:daemon`, `test:race`, `vet`, `vet:windows`, `vet:daemon-windows`, `vuln`, plus the two contributor wrappers `test` and `lint` — every fine-grained target's command body ported byte-for-byte from the `ci.yml` step it replaces (`<!-- verbatim -->` per D-02), including the load-bearing IN-08 materialize-before-filter shape in `test:unit` and the documented flake/rationale comments
- Resolved the plan's flagged "decide by measurement" question: `golang.org/x/vuln/cmd/govulncheck` was added to the **existing** `go.tool.mod` on the first attempt — measured live, it adds exactly 2 net-new indirect modules (`golang.org/x/telemetry`, `golang.org/x/vuln`) and builds clean alongside `task`/`goreleaser` with no MVS conflict, so no second/third modfile was needed
- `vet:daemon-windows` carries `preconditions:` with a single-line actionable `msg:` (mingw-w64 install instructions) — never `status:`/`platforms:`, both of which silently skip; `vet:windows` correctly carries no precondition since `internal/graphstore` is CGo-free
- `internal/upgrade/taskfile_shape_test.go` gained `TestTaskfileGatesFailLoud` (no `status:`/`platforms:` anywhere in `Taskfile.yml`; every cross-toolchain-referencing task carries a non-empty `msg:`) and `TestTaskfileWrapperIsSerial` (the `test` wrapper's leg set is sorted-set-equal to the D-10 fixture, catching both a missing and an extra leg, and carries no `deps:` key), each demonstrated RED (via a real mutation, confirmed landed via read-back) then GREEN (reverted)
- `ci.yml`'s `test` job (one of six required status checks) now invokes only `task <target>` calls (except the one CI-only mingw-w64 `apt-get` install), runs on `namespace-profile-linux-amd64-4x8`, and gains a new `Vet (go vet ./...)` step — DEV-01 new coverage, not a rewiring (plain `go vet ./...` was never a CI gate before); zero pre-existing findings surfaced, so no fix-up was required
- Job-name and step-name preservation verified by **sorted-set diff against the pre-change file**, not eyeballed: all 6 required-check job names byte-identical; the `test` job's step-name set is exactly the pre-change 11 plus the 3 expected additions (`Cache Go modules and build`, `Install Task`, `Vet (go vet ./...)`)

## Task Commits

1. **Task 1: Fine-grained build/test/vet/vuln targets, ported verbatim** - `521cc8f` (feat)
2. **Task 2: Contributor wrappers (serial) plus a guard that nothing silently skips** - `b998174` (test)
3. **Task 3: Rewire ci.yml's test job onto task targets and Namespace** - `7fbab75` (feat)

## Files Created/Modified

- `Taskfile.yml` - 12 new fine-grained targets + 2 contributor wrappers (`test`, `lint`)
- `go.tool.mod` / `go.tool.sum` - `golang.org/x/vuln/cmd/govulncheck` added to the isolated modfile (+2 net-new modules)
- `internal/upgrade/taskfile_shape_test.go` - `TestTaskfileGatesFailLoud`, `TestTaskfileWrapperIsSerial`, plus their edge-case tests and shared parse helpers (`parseTaskBlocks`, `blockDeclaresKey`, `blockReferencesToken`, `parsePreconditionMessages`, `parseTaskCallList`)
- `.github/workflows/ci.yml` - `test` job rewired onto `task <target>` calls + Namespace runner + new `Vet (go vet ./...)` step

## Decisions Made

- govulncheck lands in `go.tool.mod` (existing modfile), resolved by live measurement on the first attempt — see frontmatter `key-decisions` for the exact module-count delta
- `vet:daemon-windows`'s precondition `msg:` written as a single inline string, not a YAML folded block scalar — parseable by a plain regex, matches the plan's own canonical example
- `build:release`'s `Date` ldflag uses `TZ=UTC` + git's own date formatter instead of GNU `date -d`, for macOS/Linux portability
- `go vet ./...` needed zero fixes — the repo's current state has no pre-existing vet findings

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Modified `go.tool.mod`/`go.tool.sum`, outside the plan's stated `files_modified`**
- **Found during:** Task 1
- **Issue:** The plan's frontmatter `files_modified` lists only `Taskfile.yml`, `.github/workflows/ci.yml`, and `internal/upgrade/taskfile_shape_test.go`. Making the `vuln` target (a `must_haves.truths` and `key_links` requirement) actually work required adding `golang.org/x/vuln/cmd/govulncheck` as a tool directive to a modfile — the plan's own `<flagged_assumptions>` section explicitly anticipated this ("Task 1 carries the decision procedure; the outcome goes in the SUMMARY"), but the frontmatter's `files_modified` list was not updated to reflect it.
- **Fix:** Measured live (per the plan's own decision procedure): attempted adding the tool directive to the existing `go.tool.mod` first; it built clean with only 2 net-new modules and no MVS conflict, so no second/third modfile was needed.
- **Files modified:** `go.tool.mod`, `go.tool.sum` (both already tracked, isolated-tool-module files — not new files, not touching root `go.mod`/`go.sum`)
- **Verification:** `GOWORK=off go build -modfile=go.tool.mod -o /tmp/govulncheck-test golang.org/x/vuln/cmd/govulncheck` succeeds; `GOWORK=off go tool -modfile=go.tool.mod govulncheck ./...` runs clean against the real repo; `TestToolModfilesRemainIsolated` still passes (root `go.mod` untouched, confirmed via `git status`/`git diff --stat go.mod go.sum` showing zero changes)
- **Committed in:** `521cc8f` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (missing critical, explicitly anticipated by the plan's own flagged assumption)
**Impact on plan:** Necessary for the `vuln` target's stated acceptance criteria to hold at all. No scope creep — root `go.mod`/`go.sum` were never touched, and the plan's own text predicted exactly this outcome.

## Non-Vacuity Demonstrations (RED-then-GREEN), as performed

1. **`task vet` real finding:** added `internal/version/zzz_vet_mutation_test.go` with `fmt.Printf("%d\n", "not-an-int")`, confirmed via `cat`, ran `task vet` → exit 201 naming the exact file/line; deleted the file → exit 0.
2. **`task vet:daemon-windows` precondition:** narrowed `PATH` to exclude the directory containing `x86_64-w64-mingw32-gcc` (confirmed via `command -v` returning exit 1 under the narrowed `PATH`), ran the target → exit 201 with the mingw-w64 install message; restored `PATH` → exit 0.
3. **`TestTaskfileGatesFailLoud`:** added `platforms: [linux]` to `vet:daemon-windows` (confirmed via read-back), test → FAIL naming `vet:daemon-windows`; reverted via `mv Taskfile.yml.bak Taskfile.yml`, confirmed zero residual diff → PASS.
4. **`TestTaskfileWrapperIsSerial`, missing-leg direction:** removed the `- task: test:race` line from the `test` wrapper (confirmed via `rg`), test → FAIL naming the 4-element set missing `test:race`.
5. **`TestTaskfileWrapperIsSerial`, extra-leg direction:** on top of the missing-leg mutation reverted, added `- task: build` as a 6th leg (confirmed via `rg`), test → FAIL naming the 6-element set with the extra `build`; reverted, confirmed zero diff via `git diff Taskfile.yml` → PASS.

**Not performed — explicitly flagged, not implied passing (same gap 10-01 left open):** a real pushed CI run confirming the `test` job actually schedules on `namespace-profile-linux-amd64-4x8` and the new `Vet (go vet ./...)` step is observed failing against a real pushed vet finding, then passing after removal. See `coverage.D3` above.

## Issues Encountered

None beyond the flagged assumption resolved in Deviations above.

## User Setup Required

None during execution — the Namespace GitHub integration precondition (verified satisfied for 10-01) is unchanged for this plan; `test` moving to `namespace-profile-linux-amd64-4x8` uses the same already-verified integration.

## Next Phase Readiness

- The full `ci.yml`/`Taskfile.yml` chain for the `test` job (one of six required checks) is in place and locally verified; 10-03 onward can extend the same pattern to `bench.yml`, `release-please.yml`'s `pretag-gate`, and `release.yml` per the phase's `10-CONTEXT.md` D-06/D-09/D-15.
- **Blocker/concern carried forward (same shape as 10-01's):** the `test` job's move to `namespace-profile-linux-amd64-4x8` and the new `Vet (go vet ./...)` step have not been observed on a real pushed CI run. The next push/PR against this branch should be watched for: (a) the job scheduling on the Namespace runner rather than queuing indefinitely, and (b) all 14 steps — including the 3 new ones — completing, with `Vet (go vet ./...)` reporting green.
- `go.tool.mod`'s header comment now documents govulncheck's presence and rationale for a future reader who might otherwise wonder why a vulnerability scanner lives in a build-tool modfile.

## Self-Check: PASSED

All 5 modified deliverable files confirmed present on disk with the expected content; all 3 task commits (`521cc8f`, `b998174`, `7fbab75`) confirmed in `git log`; root `go.mod`/`go.sum` confirmed untouched (`git diff --stat go.mod go.sum` empty).

---
*Phase: 10-local-build-contribution-and-taskfile-yml-setup*
*Completed: 2026-08-01*
