---
phase: 01-changie-baseline
plan: 02
subsystem: build-tooling
tags: [changie, changelog, taskfile, ci, govulncheck, mutation-testing]

requires:
  - phase: 01-changie-baseline (plan 01)
    provides: ".changie.yaml, the 14 .changes/v*.md seeds, go.tool-changie.mod, and the task changie wrapper"
provides:
  - "check:changie: an 11-leg live-tool guard proving CHG-03 (latest/next auto/merge --dry-run) and CHG-04 (non-interactive fragment write plus four refusal shapes) against a scratch copy that never mutates the source tree"
  - "ci.yml's test job runs task check:changie on every PR and push to main, immediately after task docs:cli:drift"
  - "task vuln scans the changie binary alongside the other eight tool binaries (nine total), re-measuring the third-party module-count figure live"
  - "01-MUTATION-LOG.md: four RED-then-GREEN mutation demonstrations proving check:changie and the Go seed/heading guard have teeth"
affects: ["Phase 2 (CAP-*, fragment-writing capability)", "Phase 4 (GATE-01, fragment-required gate)", "Phase 5 (release workflow replacing release-please)"]

actuals:
  tokens: 10628
  tasks: 3
  commits: 4
  plan_head_before: b312ec72

tech-stack:
  added: []
  patterns:
    - "check:changie copies docs:cli:drift's scratch-dir/count-before-compare/named-::error:: shape onto a live third-party CLI tool rather than a code-generation drift check"
    - "Mutation-log convention (03-MUTATION-LOG.md house style): pre-mutation cleanliness gate, applied-confirmation, verbatim RED transcript, byte-clean revert proof, GREEN re-run — reused verbatim in a scratch git worktree rather than in place"

key-files:
  created:
    - .planning/phases/01-changie-baseline/01-MUTATION-LOG.md
  modified:
    - Taskfile.yml
    - .github/workflows/ci.yml
    - internal/upgrade/changie_shape_test.go

key-decisions:
  - "D-07 implemented exactly: check:changie's 11 legs run in a mktemp scratch copy, print the seeded-file count (found 14, floor 14) before any comparison, and assert every refusal on changie's own stderr substring plus a non-zero exit — never a bare exit code"
  - "silent: true was added to check:changie (not specified by the plan's acceptance criteria for the Taskfile shape, but required to satisfy the plan's own <verify> commands, which count literal-string occurrences in `task check:changie`'s output — go-task's default non-silent mode echoes the whole multi-line cmds: script before executing it, so any fixed-text echo line would otherwise count twice and fail every exact-count verify)"
  - "D-08 implemented exactly: ci.yml's test job runs `task check:changie` as the single step immediately after `task docs:cli:drift`, with no new permissions, secrets, or uses: lines"
  - "vuln's module-count figure was re-measured live rather than trusted: neither the bare-path nor the path@version deduplicated-union form reproduces the previously-recorded 571 (now 1257 across the first four modfiles, 1261 with go.tool-changie.mod unioned in, +4 net-new) — recorded as measured, not carried forward unverified, per the plan's own 'never keep a number you did not measure' instruction"
  - "Family (d)'s first mutation (rm -f .changes/v0.5.1.md) collides with changieSeedFloor=14 — the guard's floor check fires before the set-mismatch check, so a bare deletion never names the missing version. A floor-compensating filler seed was added alongside the deletion to reach the set-mismatch branch that literally names v0.5.1, satisfying the plan's acceptance criterion without touching the guard's own (correct) floor-first logic. Documented as a deviation in 01-MUTATION-LOG.md"

requirements-completed: [CHG-01, CHG-03, CHG-04]

coverage:
  - id: D1
    description: "check:changie's 11-leg live-tool guard proves CHG-03 (latest/next auto/merge --dry-run) and CHG-04 (non-interactive write, four refusal shapes, sub-second collision guard) against a scratch copy, with a before/after checksum snapshot proving the source tree is never mutated"
    requirement: "CHG-03"
    verification:
      - kind: integration
        ref: "task check:changie"
        status: pass
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestTaskfileGatesFailLoud"
        status: pass
    human_judgment: false
  - id: D2
    description: "check:changie's four refusal legs (missing PR, undeclared kind, PR below minInt, non-integer PR) each assert changie's own stderr text plus a non-zero exit and an unchanged fragment count"
    requirement: "CHG-04"
    verification:
      - kind: integration
        ref: "task check:changie"
        status: pass
    human_judgment: false
  - id: D3
    description: "ci.yml's test job runs task check:changie immediately after task docs:cli:drift, with no new permissions/secrets/uses: lines, RED-first via TestChangieCheckWiredIntoCI"
    requirement: "CHG-03"
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_shape_test.go#TestChangieCheckWiredIntoCI"
        status: pass
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestWorkflowRunBodiesInvokeTask"
        status: pass
      - kind: other
        ref: "task lint:actions"
        status: pass
    human_judgment: false
  - id: D4
    description: "task vuln builds changie from go.tool-changie.mod and scans it in the nine-binary loop, RED-first via TestChangieBinaryInToolVulnScan; the module-count figure is live-remeasured"
    requirement: "CHG-01"
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_shape_test.go#TestChangieBinaryInToolVulnScan"
        status: pass
      - kind: integration
        ref: "task vuln"
        status: pass
    human_judgment: false
  - id: D5
    description: "RED-first TDD discipline for Task 2: test(01-02) commit lands before ci(01-02)"
    requirement: "CHG-01"
    verification:
      - kind: other
        ref: "git log --reverse --format=%s main..HEAD"
        status: pass
    human_judgment: false
  - id: D6
    description: "Four mutation families (byte-reproduction, missing-PR refusal, collision, Go seed/heading set equality both directions) each shown RED against a confirmed mutation, byte-cleanly reverted, and GREEN again, entirely in a disposable scratch git worktree — no main-tree mutation and no leaked worktree"
    requirement: "CHG-03"
    verification:
      - kind: other
        ref: ".planning/phases/01-changie-baseline/01-MUTATION-LOG.md"
        status: pass
    human_judgment: false

duration: ~30min
completed: 2026-09-25
status: complete
---

# Phase 1 Plan 2: Changie Live-Tool Guard and CI Wiring Summary

**An 11-leg `check:changie` Taskfile guard proves every CHG-03/CHG-04 claim against a scratch copy of the pinned changie tool, wired into `ci.yml` right after the CLI drift guard, with four RED-then-reverted mutation demonstrations proving the guard actually has teeth.**

## Performance

- **Duration:** ~30 min (commit-to-commit span ~10 min; total including mutation-worktree demonstrations, transcripts, and this summary is longer)
- **Started:** 2026-09-25T16:12:54-04:00 (first commit)
- **Completed:** 2026-09-25T16:22:21-04:00 (last commit)
- **Tasks:** 3/3 completed
- **Files modified:** 4 (`Taskfile.yml`, `.github/workflows/ci.yml`, `internal/upgrade/changie_shape_test.go`, plus the new `01-MUTATION-LOG.md`)

## Accomplishments

- `check:changie` (Taskfile.yml) runs 11 numbered legs in a `mktemp -d` scratch copy: `changie latest` == `v0.14.0` (never below baseline), `changie next auto` refuses on an empty `unreleased/` with changie's own text, `changie merge --dry-run` byte-reproduces `CHANGELOG.md`, a valid `new` writes exactly one fragment, four refusal shapes (missing PR, undeclared kind, PR below minInt, non-integer PR) are each proven on changie's own stderr text, a same-second collision is proven (and the sub-second `fragmentFileFormat` proven to prevent it), and the patch/minor auto-bump successors are derived and asserted (`v0.14.1`, then `v0.15.0` once a Breaking fragment lands — pre-1.0 Breaking maps to minor, never major).
- A before/after `cksum` snapshot of `.changie.yaml`, `.changes/` and `CHANGELOG.md` proves `check:changie` never mutates the real tree; the seeded-file count (14, floor 14) is printed before any comparison runs (rule `84d1gfpywd`). The target is idempotent — two consecutive runs both end `11 of 11 checks passed`.
- `ci.yml`'s `test` job now runs `task check:changie` as the single step immediately after `task docs:cli:drift` (D-08) — no new `permissions:`, `secrets.`, or `uses:` lines. `TestChangieCheckWiredIntoCI` pins the wiring and landed RED first.
- `task vuln` now builds `changie` from `go.tool-changie.mod` and scans it alongside the other eight tool binaries (nine total); `TestChangieBinaryInToolVulnScan` pins this and landed RED first. The desc's third-party module-count figure was re-measured live rather than trusted (see below).
- `01-MUTATION-LOG.md` records four mutation families, each demonstrated RED against a confirmed mutation in a disposable scratch git worktree, byte-cleanly reverted, and GREEN again — the main working tree was never touched.

## Task Commits

Each task was committed atomically:

1. **Task 1: check:changie, the 11-leg live-tool guard** - `3903f5b6` (feat)
2. **Task 2 (RED half): failing CI-wiring and tool-vuln coverage guards** - `1d023e70` (test)
3. **Task 2 (GREEN half): wire check:changie into ci.yml and scan changie in task vuln** - `b85d2eea` (ci)
4. **Task 3: mutation log (four families, RED then GREEN)** - `1e8b0c8b` (docs)

`git log --reverse --format=%s main..HEAD` confirms `test(01-02):` precedes `ci(01-02):` (rule `x1cjy9vyhq` RED-first discipline).

## Full Green `task check:changie` Transcript

```
$ task check:changie
check:changie: found 14 seeded version files under .changes/ (floor 14)
check:changie: [1/11] changie latest = v0.14.0 (newest CHANGELOG.md heading v0.14.0; baseline v0.14.0) — ok
check:changie: [2/11] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/11] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
check:changie: [3/11] changie merge --dry-run byte-identical to CHANGELOG.md (dry run wrote nothing) — ok
check:changie: [4/11] changie new -k Fixes writes exactly one fragment with kind/body set — ok
check:changie: [5/11] changie new with no PR refused: custom missing and prompt is disabled: custom key 'PR' — ok
check:changie: [6/11] changie new -k Undeclared refused: invalid kind: Undeclared — ok
check:changie: [7/11] changie new -m PR=0 refused: input below minimum: 0 < 1 — ok
check:changie: [8/11] changie new -m PR=abc refused: invalid number — ok
check:changie: [9/11] two back-to-back new -k Fixes calls raised the fragment count to 3 (sub-second fragmentFileFormat holds) — ok
check:changie: [10/11] with only Fixes fragments present, next auto = v0.14.1 (patch successor of v0.14.0) — ok
check:changie: [11/11] a Breaking fragment derives the minor successor v0.15.0 (pre-1.0: Breaking -> minor) — ok
check:changie: 11 of 11 checks passed against a scratch copy (source tree byte-unchanged)
```

Confirmed idempotent (two consecutive runs, both exit 0, both end `11 of 11 checks passed`, source tree unchanged both times).

## Task 2 RED Evidence

Committed as `1d023e70` (`test(01-02): add failing CI-wiring and tool-vuln coverage guards for changie`), before ci.yml's step or the vuln target's changie build existed:

```
$ GOWORK=off go test ./internal/upgrade/ -run '^TestChangie(CheckWiredIntoCI|BinaryInToolVulnScan)$' -count=1 -v
changie_shape_test.go:567: ../../.github/workflows/ci.yml job "test": no step's run: body strips to "task check:changie" — observed steps: [    task build task vet task lint:go task test:unit task corpora:verify  task web:deps task web:test task web:deps:strict task web:audit task web:build:verify task web:drift task proto:drift task docs:cli:drift  task check:gonum task check:no-force-layout bash scripts/check-ruleset-drift.sh task test:integration task test:wireoracle task test:daemon task test:race]
--- FAIL: TestChangieCheckWiredIntoCI (0.00s)
    changie_shape_test.go:617: vuln task block does not contain "-modfile=go.tool-changie.mod" — changie is not built from the pinned modfile
--- FAIL: TestChangieBinaryInToolVulnScan (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.178s
FAIL
```

Both are real assertion failures (missing wiring), not build errors.

## Task 2 GREEN Evidence

After ci.yml's step and the vuln target's changie build/loop entries landed:

```
$ GOWORK=off go test ./internal/upgrade/ -run '^(TestChangieCheckWiredIntoCI|TestChangieBinaryInToolVulnScan|TestWorkflowRunBodiesInvokeTask|TestGateStancesStated)$' -count=1 -v
--- PASS: TestChangieCheckWiredIntoCI (0.00s)
--- PASS: TestChangieBinaryInToolVulnScan (0.00s)
--- PASS: TestGateStancesStated (0.00s)
--- PASS: TestWorkflowRunBodiesInvokeTask (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.229s
```

## Vuln Module-Count Measurement

The plan required re-measuring the desc's "571 measured third-party modules" figure live rather than trusting it. Neither form reproduces 571:

```
$ for f in go.tool.mod go.tool-lint.mod go.tool-proto.mod go.tool-golangci.mod; do GOWORK=off go list -m -modfile="$f" -f '{{if not .Main}}{{.Path}}{{end}}' all; done | sort -u | wc -l
1257

$ for f in go.tool.mod go.tool-lint.mod go.tool-proto.mod go.tool-golangci.mod; do GOWORK=off go list -m -modfile="$f" -f '{{if not .Main}}{{.Path}}@{{.Version}}{{end}}' all; done | sort -u | wc -l
1384

$ for f in go.tool.mod go.tool-lint.mod go.tool-proto.mod go.tool-golangci.mod go.tool-changie.mod; do GOWORK=off go list -m -modfile="$f" -f '{{if not .Main}}{{.Path}}{{end}}' all; done | sort -u | wc -l
1261
```

Per the plan's own fallback instruction ("If neither form reproduces 571, write the measured current four- and five-modfile counts and say so"), the desc now states **1257** (deduplicated union of module Path across the first four modfiles) growing to **1261** with `go.tool-changie.mod` unioned in (+4 net-new — matching that modfile's own header comment, which independently measured the same +4 delta from co-locating changie into `go.tool.mod`). The drift from the previously-recorded 571 reflects genuine dependency-graph growth across earlier phases (more tool modfiles, newer transitive deps added since that number was recorded), not a measurement-methodology error.

```
$ task vuln 2>&1 | tail -10
    ...
    changie: CLEAN (exit 0 — no symbol-matched vulnerability, or a module-level-only match govulncheck does not consider a symbol hit)
```

## Family (a) Mutation Section (also in 01-MUTATION-LOG.md)

**Test/guard:** `check:changie` leg [3/11] (`changie merge --dry-run` byte-compared against `CHANGELOG.md`).

**Pre-mutation gate:** `git -C "$S/wt" diff --quiet -- CHANGELOG.md` exited 0 (clean).

**Mutation applied:** flipped exactly one byte on line 1 (`# Changelog` -> `# changelog`) via `sed -i.bak '1s/^# Changelog$/# changelog/' "$S/wt/CHANGELOG.md"`, then `rm -f "$S/wt/CHANGELOG.md.bak"`.

**Applied-confirmation:** `cmp -l <(git -C "$S/wt" show HEAD:CHANGELOG.md) "$S/wt/CHANGELOG.md" | wc -l` printed `1`.

**RED transcript** (verbatim):

```
$ task -d "$S/wt" check:changie
check:changie: found 14 seeded version files under .changes/ (floor 14)
check:changie: [1/11] changie latest = v0.14.0 (newest CHANGELOG.md heading v0.14.0; baseline v0.14.0) — ok
check:changie: [2/11] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/11] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
::error::check:changie: [3/11] changie merge --dry-run does not reproduce CHANGELOG.md byte-for-byte
--- CHANGELOG.md	2026-09-25 16:17:24
+++ /var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/tmp.GvmiQjLnMx/merged.md	2026-09-25 16:17:25
@@ -1,4 +1,4 @@
-# changelog
+# Changelog
 
 ## [0.14.0](https://github.com/seanb4t/codegraph-go/compare/v0.13.0...v0.14.0) (2026-09-20)
 
task: Failed to run task "check:changie": exit status 1
```

**Revert:** `git -C "$S/wt" checkout -- CHANGELOG.md`. **Byte-clean revert proof:** `git -C "$S/wt" status --porcelain` empty. **GREEN re-run:** `11 of 11 checks passed against a scratch copy (source tree byte-unchanged)`.

See `01-MUTATION-LOG.md` for families (b), (c), and (d), including the documented deviation on family (d)'s first half (the `changieSeedFloor` interaction with a bare single-file deletion).

## Files Created/Modified

- `Taskfile.yml` — `check:changie` (11-leg live-tool guard, `silent: true`) added after the `changie` wrapper; `vuln` target extended with the changie build/scan and re-measured module-count desc
- `.github/workflows/ci.yml` — one new `test` job step, `Changie baseline and fragment guard (CHG-03/CHG-04)`, running `task check:changie` immediately after `task docs:cli:drift`
- `internal/upgrade/changie_shape_test.go` — `TestChangieCheckWiredIntoCI` and `TestChangieBinaryInToolVulnScan` added (landed RED first)
- `.planning/phases/01-changie-baseline/01-MUTATION-LOG.md` — new, four mutation families with pre-mutation gates, applied confirmations, RED transcripts, byte-clean revert proofs, and GREEN re-runs

## Decisions Made

- `silent: true` was added to `check:changie` — not required by the plan's Go-test acceptance criteria, but required for the plan's own literal `<verify>` commands (which count exact string occurrences in `task check:changie 2>&1` output) to pass: go-task's default non-silent mode echoes the entire multi-line `cmds:` script text before executing it, so any fixed-text `echo` line inside the script would otherwise appear twice (once as echoed source, once as real output) and every exact-count `rg -F -o ... | wc -l` check would read 2 instead of 1. Confirmed this is go-task's stock behavior, not a defect of this guard, by reproducing the identical doubling on the pre-existing `docs:cli:drift` target.
- The module-count re-measurement (Task 2) found the previously-recorded 571 unreproducible by either form the plan named; rather than force a matching number, the actually-measured 1257/1261 figures were written, with the method and the discrepancy both stated in the desc and here.
- Family (d)'s first mutation (deleting `.changes/v0.5.1.md`) collides with `changieSeedFloor = 14`: the guard's floor check runs before its set-mismatch check, so a bare deletion (14→13) short-circuits on the floor message without naming the specific missing version. A floor-compensating filler seed (`v0.14.0-filler.md`) was added alongside the deletion, reverted together with it, to reach the set-mismatch branch and literally satisfy the plan's "must name v0.5.1" acceptance criterion — documented as a deviation rather than silently reworded away.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `check:changie` needed `silent: true` to satisfy the plan's own exact-count `<verify>` commands**
- **Found during:** Task 1, verifying the plan's `<verify>` block
- **Issue:** `task check:changie 2>&1 | rg -F -o 'check:changie: 11 of 11 checks passed' | wc -l` returned `2`, not `1` — go-task's default non-silent output echoes the whole `cmds:` script (including every literal `echo` string) once, then the real execution output a second time
- **Fix:** Added `silent: true` to the `check:changie` task block, which suppresses the pre-execution echo but leaves the script's own stdout/stderr untouched
- **Files modified:** `Taskfile.yml`
- **Verification:** Re-ran all of Task 1's `<verify>` commands; both exact-count assertions now return `1`, matching the plan's literal expectation
- **Commit:** `3903f5b6` (part of Task 1's commit)

**2. [Rule 3 - Blocking] Vuln module-count figure could not reproduce the plan's named target (571) with either specified measurement form**
- **Found during:** Task 2, GREEN part 2 (vuln target edit)
- **Issue:** Both `{{.Path}}` and `{{.Path}}@{{.Version}}` deduplicated-union forms measured 1257 and 1384 respectively across the first four modfiles — neither matches the previously-recorded 571
- **Fix:** Followed the plan's own explicit fallback instruction: recorded the measured current four-modfile (1257) and five-modfile (1261, +4 net-new with go.tool-changie.mod) counts, named the method, and stated the discrepancy rather than silently keeping the stale 571 or fabricating a matching number
- **Files modified:** `Taskfile.yml` (vuln desc)
- **Verification:** Measurement commands and their output are pasted verbatim above and in the commit message
- **Commit:** `b85d2eea` (part of Task 2's GREEN commit)

**3. [Rule 3 - Blocking] Family (d)'s first mutation, as literally specified, never names the missing version**
- **Found during:** Task 3, Family (d) first half
- **Issue:** `rm -f .changes/v0.5.1.md` drops the seed count from 14 to 13, tripping `TestChangieVersionSeedsMatchChangelog`'s `changieSeedFloor = 14` check before the set-mismatch check ever runs, so the `--- FAIL` output never contains the literal string `v0.5.1` as the plan's acceptance criteria require
- **Fix:** Added a floor-compensating filler seed (`v0.14.0-filler.md`, a byte-copy of an existing seed) alongside the `v0.5.1.md` deletion, keeping the total count at the floor (14) so the guard proceeds to the set-mismatch branch, which does literally name `v0.5.1`. Both the deletion and the filler were reverted together
- **Files modified:** none (scratch worktree only; documented in `01-MUTATION-LOG.md`)
- **Verification:** RED transcript pasted in `01-MUTATION-LOG.md` shows `--- FAIL: TestChangieVersionSeedsMatchChangelog` naming `v0.5.1`; GREEN re-run and byte-clean revert both confirmed
- **Commit:** `1e8b0c8b` (part of Task 3's commit)

---

**Total deviations:** 3 auto-fixed (all Rule 3 — blocking issues preventing the plan's own stated verification/acceptance commands from passing as literally written).
**Impact on plan:** All three were necessary to make the plan's own machine-checked criteria pass against real tool behavior (go-task's echo semantics, the actual current module graph size, and the guard's floor-before-mismatch check order). No scope creep — no production code changed beyond what the plan specified; each fix is either a Taskfile flag, an honestly-measured number, or a demonstration-methodology adjustment recorded in the mutation log.

## Issues Encountered

- `task lint:go` fails under the local `go 1.27.1` toolchain with `undefined: pebble` typecheck errors inside `internal/graphstore/pebble_store.go` — an environmental toolchain mismatch unrelated to this plan's files (confirmed: `GOTOOLCHAIN=go1.26.6 task lint:go` passes with `0 issues`). Matches the repo's own standing note that a whole-module build/lint under 1.27.1 fails inside third-party code compiled against the pinned 1.26.6 toolchain. Out of scope for this plan; not touched.

## User Setup Required

None — no external service configuration required.

## Hand-off Notes (not work for this plan, per the plan's `<verification>` block)

- `release-please.yml` is still live and still writes `CHANGELOG.md` until Phase 5 retires it. **No release may be cut between Phase 1 and Phase 5** without reconciling the two writers.
- `check:changie`'s `latest` and successor legs derive from the newest `CHANGELOG.md` heading, so they stay green when Phase 5's release PR batches a new version — Phase 5 must still run `check:changie` on that PR.
- Flagged assumption A-01 (from `01-CONTEXT.md`) transfers to Phase 2 and Phase 4 planning: changie v1.26.0 silently overwrites a same-second same-kind fragment (mitigated here by the sub-second `fragmentFileFormat`, proven by leg [9/11] and Family (c)); a written fragment's `custom.PR` is a quoted YAML string despite the declared `type: int`; `changie` only reads `<envPrefix>_CUSTOM_<key>` when `envPrefix` is configured (none is).

## Next Phase Readiness

- Phase 1's success criteria are now fully discharged: `check:changie` is a committed, CI-wired Taskfile target (not a one-off shell transcript) proving CHG-03 and CHG-04, and `task vuln` scans the changie binary CHG-01's CI install path exercises.
- `REQUIREMENTS.md` CHG-01, CHG-03, and CHG-04 are complete as of this plan's SUMMARY.
- No blockers for Phase 2.

## Self-Check: PASSED

- All key files confirmed present on disk (`Taskfile.yml`, `.github/workflows/ci.yml`, `internal/upgrade/changie_shape_test.go`, `.planning/phases/01-changie-baseline/01-MUTATION-LOG.md`).
- All four commits (`3903f5b6`, `1d023e70`, `b85d2eea`, `1e8b0c8b`) confirmed in `git log`.
- `git log --reverse --format=%s main..HEAD` confirms `test(01-02):` precedes `ci(01-02):`.
- Re-ran every `<acceptance_criteria>` command from all three tasks and every plan-level `<verification>` command — all pass (see transcripts above; `task lint:go`'s environmental failure is documented separately as out-of-scope, confirmed non-caused-by-this-plan via the `GOTOOLCHAIN` pin).
- `git worktree list` confirmed to show only the main checkout (no leaked scratch worktree).
- `git status --porcelain -- CHANGELOG.md .changie.yaml .changes` confirmed empty (no leaked mutation).

---
*Phase: 01-changie-baseline*
*Completed: 2026-09-25*
