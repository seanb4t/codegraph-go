---
phase: 01-changie-baseline
plan: 01
subsystem: build-tooling
tags: [changie, changelog, go-tool-modfile, taskfile, release-management]

requires: []
provides:
  - ".changie.yaml locked to CHG-01's kinds/auto/PR/format vocabulary (D-10, D-11)"
  - "14 verbatim byte-sliced .changes/v*.md seeds, byte-reproducing CHANGELOG.md via changie merge --dry-run"
  - "changie v1.26.0 pinned in a fifth isolated Go tool modfile (go.tool-changie.mod), registered in isolatedModfilePaths/forbiddenToolPackages"
  - "task changie wrapper target — the single recorded install path for CI and contributors (D-05)"
  - "CONTRIBUTING.md documents the changie install path"
affects: ["01-02 (check:changie live-tool proof + ci.yml wiring)", "Phase 5 (release workflow replacing release-please)"]

actuals:
  tokens: 19979
  tasks: 2
  commits: 3

tech-stack:
  added: ["github.com/miniscruff/changie v1.26.0"]
  patterns:
    - "Fifth isolated Go tool-modfile (go.tool-changie.mod), same header/registration protocol as go.tool.mod/go.tool-lint.mod/go.tool-proto.mod/go.tool-golangci.mod"
    - "Byte-sliced historical migration: existing CHANGELOG.md entries become verbatim seed files rather than re-authored/re-rendered content"

key-files:
  created:
    - internal/upgrade/changie_shape_test.go
    - .changie.yaml
    - .changes/header.tpl.md
    - .changes/unreleased/.gitkeep
    - ".changes/v0.2.0.md .. .changes/v0.14.0.md (14 seeds)"
    - go.tool-changie.mod
    - go.tool-changie.sum
  modified:
    - internal/upgrade/taskfile_shape_test.go
    - Taskfile.yml
    - CONTRIBUTING.md

key-decisions:
  - "D-01/D-02/D-09: 14 seeds byte-sliced from CHANGELOG.md at rg-measured line ranges; newlines.afterChangelogHeader: 1 is the only newlines override; reassembly proven byte-identical to CHANGELOG.md before any tool was built"
  - "D-04/D-05: changie v1.26.0 pinned in a new isolated go.tool-changie.mod (64 modules standalone); one recorded install path via `task changie` for both CI and contributors"
  - "D-06/x1cjy9vyhq: six TestChangie* guards committed RED first (test(01-01) before feat(01-01)), never gated on gsd-tools check tdd-red-evidence"
  - "D-10/D-11: .changie.yaml copies the design-note vocabulary verbatim; PR custom field stays required (no optional key)"

requirements-completed: [CHG-02]

coverage:
  - id: D1
    description: ".changie.yaml declares exactly the locked kinds/auto/PR/format vocabulary (CHG-01 static half)"
    requirement: "CHG-01"
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_shape_test.go#TestChangieConfigShape"
        status: pass
    human_judgment: false
  - id: D2
    description: ".changes/ baseline layout (header.tpl.md, unreleased/.gitkeep, v0.14.0.md) exists and is well-formed"
    requirement: "CHG-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_shape_test.go#TestChangieBaselineLayout"
        status: pass
    human_judgment: false
  - id: D3
    description: "14 .changes/v*.md seeds set-equal CHANGELOG.md's ## [x.y.z] headings, both directions, floor 14"
    requirement: "CHG-02"
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_shape_test.go#TestChangieVersionSeedsMatchChangelog"
        status: pass
    human_judgment: false
  - id: D4
    description: "changie v1.26.0 pinned in go.tool-changie.mod, registered in isolatedModfilePaths and forbiddenToolPackages"
    requirement: "CHG-01"
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_shape_test.go#TestChangieToolPinnedInIsolatedModfile"
        status: pass
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestToolModfilesRemainIsolated"
        status: pass
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestToolModfilesPopulationMatchesDisk"
        status: pass
    human_judgment: false
  - id: D5
    description: "Taskfile.yml changie wrapper target records the single install path (GO_TOOL_CHANGIE var + task changie target)"
    requirement: "CHG-01"
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_shape_test.go#TestChangieWrapperTaskRecordsInstallPath"
        status: pass
    human_judgment: false
  - id: D6
    description: "Tracer end-to-end proof: task changie -- latest prints v0.14.0; task changie -- merge --dry-run is byte-identical to CHANGELOG.md; CHANGELOG.md unchanged vs main"
    requirement: "CHG-02"
    verification:
      - kind: integration
        ref: "task changie -- latest"
        status: pass
      - kind: integration
        ref: "task changie -- merge --dry-run | cmp - CHANGELOG.md"
        status: pass
      - kind: integration
        ref: "git diff --quiet main -- CHANGELOG.md"
        status: pass
    human_judgment: false
  - id: D7
    description: "CONTRIBUTING.md documents changie's install path, scoped to the tool bullet only"
    requirement: "CHG-01"
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestContributingReferencesRealTaskTargets"
        status: pass
      - kind: integration
        ref: "CI=true task changie -- new --dry-run -k Fixes -b \"contributor path probe with spaces\" -m PR=1"
        status: pass
    human_judgment: false
  - id: D8
    description: "RED-first TDD discipline: test(01-01) commit lands before feat(01-01) commit"
    requirement: "CHG-01"
    verification:
      - kind: other
        ref: "git log --reverse --format=%s main..HEAD"
        status: pass
    human_judgment: false

duration: 30min
completed: 2026-09-25
status: complete
---

# Phase 1 Plan 1: Changie Baseline Summary

**Pinned changie v1.26.0 answers `v0.14.0` and reproduces `CHANGELOG.md` byte-for-byte from 14 verbatim seed files, proven by a RED-first Go test suite and a real `task changie` invocation.**

## Performance

- **Duration:** ~30 min (estimated — context/research load plus implementation; commit-to-commit span was ~5 min)
- **Started:** 2026-09-25T15:50:29-04:00 (first commit)
- **Completed:** 2026-09-25T15:55:40-04:00 (last commit)
- **Tasks:** 2/2 completed
- **Files modified:** 23 (16 new `.changes/` files, `.changie.yaml`, `go.tool-changie.mod`/`.sum`, `internal/upgrade/changie_shape_test.go` new, plus 3 modified files: `internal/upgrade/taskfile_shape_test.go`, `Taskfile.yml`, `CONTRIBUTING.md`)

## Accomplishments

- `.changie.yaml` locks the five kinds (`Breaking`/`Features`/`Fixes`/`Performance`/`Dependencies`) with `auto` bumps minor/minor/patch/patch/patch, the flip-to-major-at-1.0 inline comment, a required `PR` custom field (`type: int`, `minInt: 1`, no `optional`), the design-note's `versionFormat`/`kindFormat`/`changeFormat` copied byte for byte, a sub-second `fragmentFileFormat`, and the single `newlines.afterChangelogHeader: 1` override.
- 14 `.changes/vX.Y.Z.md` seeds are verbatim byte slices of today's `CHANGELOG.md`; reassembly (`header.tpl.md` + one newline + all 14 seeds newest-first) was proven byte-identical to `CHANGELOG.md` via `cmp` **before** any tooling was built, and again via the real `changie merge --dry-run | cmp - CHANGELOG.md` after.
- `changie` v1.26.0 is pinned in a new, fifth isolated Go tool modfile (`go.tool-changie.mod`, 64 modules standalone), registered in both `isolatedModfilePaths` and `forbiddenToolPackages` in the same edit.
- `Taskfile.yml` gained `GO_TOOL_CHANGIE` and a `changie` wrapper target — the single recorded install path contributors and CI both invoke identically.
- `CONTRIBUTING.md`'s tool bullet now names changie, `go.tool-changie.mod`, and `task changie`, scoped strictly to that bullet (no touch to `## Pull requests`).
- Six new `TestChangie*` Go guards landed RED first (commit `test(01-01)` before `feat(01-01)`), then went GREEN with the real implementation — see RED/GREEN transcripts below.

## RED Evidence

Committed as `01057ac1` (`test(01-01): add failing changie config-shape, seed-set and pin guards`), before `.changie.yaml`, `.changes/`, or `go.tool-changie.mod` existed:

```
$ GOWORK=off go test ./internal/upgrade/ -run '^TestChangie' -count=1 -v
changie_shape_test.go:166: read ../../.changie.yaml: open ../../.changie.yaml: no such file or directory
--- FAIL: TestChangieConfigShape (0.00s)
    changie_shape_test.go:307: read ../../.changes/header.tpl.md: open ../../.changes/header.tpl.md: no such file or directory
--- FAIL: TestChangieBaselineLayout (0.00s)
    changie_shape_test.go:339: listChangieVersionSeeds(../../.changes): listChangieVersionSeeds: zero v*.md files found in ../../.changes
--- FAIL: TestChangieVersionSeedsMatchChangelog (0.00s)
    changie_shape_test.go:392: read ../../go.tool-changie.mod: open ../../go.tool-changie.mod: no such file or directory
--- FAIL: TestChangieToolPinnedInIsolatedModfile (0.00s)
    changie_shape_test.go:442: ../../Taskfile.yml: does not contain the exact line "  GO_TOOL_CHANGIE: GOWORK=off go tool -modfile=go.tool-changie.mod"
--- FAIL: TestChangieWrapperTaskRecordsInstallPath (0.00s)
--- PASS: TestChangieShapeParsersFailLoudly (0.00s)
    (6 subtests, all PASS — helpers live in the test file itself)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.288s
```

5 real `--- FAIL: TestChangie…` lines (missing files, not build failures) plus `TestChangieShapeParsersFailLoudly` passing as expected. End-to-end RED:

```
$ task changie -- latest
task: Task "changie" does not exist
$ echo $?
200
```

## GREEN Evidence

After Task 1's implementation:

```
$ GOWORK=off go test ./internal/upgrade/ -run '^(TestChangie|TestToolModfiles)' -count=1 -v
--- PASS: TestChangieConfigShape (0.00s)
--- PASS: TestChangieBaselineLayout (0.00s)
    changie_shape_test.go:350: ../../.changes: 14 seed files; ../../CHANGELOG.md: 14 version headings
--- PASS: TestChangieVersionSeedsMatchChangelog (0.00s)
--- PASS: TestChangieToolPinnedInIsolatedModfile (0.00s)
--- PASS: TestChangieWrapperTaskRecordsInstallPath (0.00s)
--- PASS: TestChangieShapeParsersFailLoudly (0.00s)
--- PASS: TestToolModfilesRemainIsolated (0.00s)
--- PASS: TestToolModfilesPopulationMatchesDisk (0.00s)
--- PASS: TestToolModfilesRemainIsolated_AbsentModfileIsError (0.00s)
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.161s

$ GOWORK=off go test ./internal/upgrade/ -count=1
ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.377s

$ task changie -- latest
v0.14.0

$ task changie -- merge --dry-run | cmp - CHANGELOG.md
(no output — byte identical, exit 0)

$ git diff --quiet main -- CHANGELOG.md
(exit 0 — unchanged)

$ task changie -- next auto
Error: no unreleased changes found for automatic bumping
(exit 201 via task's wrapping of changie's exit 1)

$ gofmt -l internal/upgrade/
(empty)

$ GOWORK=off go vet ./internal/upgrade/
(exit 0)

$ GOWORK=off go tool -modfile=go.tool-golangci.mod golangci-lint run ./internal/upgrade/...
0 issues.
```

**Live module count:** `GOWORK=off go list -m -modfile=go.tool-changie.mod all | wc -l` → 64, matching the count stated in `go.tool-changie.mod`'s own header.

## Task 2 dry-run fragment output

```
$ CI=true task changie -- new --dry-run -k Fixes -b "contributor path probe with spaces" -m PR=1
kind: Fixes
body: contributor path probe with spaces
time: 2026-09-25T15:55:20.827636-04:00
custom:
    PR: "1"
```

Body line matches exactly (`rg -x -F 'body: contributor path probe with spaces'`); `.changes/` tree stayed clean afterward (`git status --porcelain -- .changes` empty).

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED half): six failing TestChangie* guards** - `01057ac1` (test)
2. **Task 1 (GREEN half): pin changie and seed the v0.14.0 baseline** - `4942a29a` (feat)
3. **Task 2: record the changie install path in CONTRIBUTING** - `9f1a6293` (docs)

`git log --reverse --format=%s main..HEAD` confirms `test(01-01):` precedes `feat(01-01):` (rule x1cjy9vyhq RED-first discipline).

## Files Created/Modified

- `internal/upgrade/changie_shape_test.go` — six new `TestChangie*` guards + `parseChangieConfig`/`mustChangieConfig`/`yamlMappingKeys`/`yamlMappingValue`/`parseChangelogVersionHeadings`/`listChangieVersionSeeds` helpers
- `.changie.yaml` — changie configuration, locked vocabulary, ownership comment header
- `.changes/header.tpl.md` — exactly `# Changelog\n`
- `.changes/unreleased/.gitkeep` — empty placeholder tracking the unreleased fragment dir
- `.changes/v0.2.0.md` … `.changes/v0.14.0.md` — 14 verbatim byte-sliced historical seeds
- `go.tool-changie.mod` / `go.tool-changie.sum` — fifth isolated Go tool modfile pinning changie v1.26.0
- `internal/upgrade/taskfile_shape_test.go` — registered `changieModfilePath` in `isolatedModfilePaths` and `github.com/miniscruff/changie` in `forbiddenToolPackages`
- `Taskfile.yml` — `GO_TOOL_CHANGIE` var + `changie` wrapper target
- `CONTRIBUTING.md` — tool bullet names changie, `go.tool-changie.mod`, `task changie`

## Decisions Made

- D-01/D-02/D-09 (seed byte-slicing, `newlines.afterChangelogHeader: 1` as the sole override, header content) followed exactly as researched — no new `newlines` combination needed; the reassembly `cmp` and the real `changie merge --dry-run` both confirmed byte identity on the first attempt.
- D-04/D-05 (separate isolated modfile, single recorded install path) followed exactly; live module count (64) matches the research measurement precisely.
- D-06/D-10/D-11 (RED-first Go test, locked vocabulary, required `PR`) followed exactly; `changieModfilePath` deliberately declared in `changie_shape_test.go` (not `taskfile_shape_test.go`'s const block) so the RED commit compiled before `go.tool-changie.mod` existed.
- No deviations from the plan's `Claude's Discretion` items were forced — the default `fragmentFileFormat` (with sub-second precision, as the plan already specified) was used, and the YAML-library parsing branch (rather than a line-scan) was chosen for exactness against `.changie.yaml`'s inline-comment requirement.

## Deviations from Plan

None — plan executed exactly as written. (One self-corrected authoring slip: an early edit to `forbiddenToolPackages` momentarily duplicated two existing entries; caught and fixed via `go vet`/review before the GREEN commit, never landed in any commit.)

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required. `go.tool-changie.mod`/`go.sum` were generated via `go mod tidy` against the live module proxy in a scratch directory; no credentials needed.

## Hand-off Notes (not work for this plan)

- `.github/workflows/release-please.yml` is still live and still writes `CHANGELOG.md` until Phase 5 retires it. **No release may be cut between Phase 1 and Phase 5** without reconciling the two writers — if one were, the new `TestChangieVersionSeedsMatchChangelog`/byte-identity guards would go red loudly (a new `## [0.14.1]` heading with no matching seed), which is the intended signal.
- Future batch spacing, measured at planning time: `changie batch auto --dry-run` renders no blank line after the version heading or the kind heading, so a merged new version file would abut the `## [0.14.0]` seed with no blank line between them. Batch-time `newlines` keys are Phase 5's decision (REL-10/REL-13); `TestChangieConfigShape`'s exact `newlines` key-set assertion makes adding one a deliberate, test-visible change.
- `.changie.yaml`, `.changes/**` and `go.tool-changie.{mod,sum}` are not yet in the `require-issue-link` or `pr_template_policy.py` exemption lists — deferred to Phase 4/5 per `01-CONTEXT.md`.
- `check:changie` (the live-tool D-07 proof: `changie next auto` refusal text, `changie new` refusal texts for missing `PR`/undeclared kind, wired into `ci.yml`) is **01-02-PLAN.md's** scope, not this plan's. `CHG-01` stays "blocked" in `REQUIREMENTS.md` (shared with 01-02's frontmatter) until that plan's `SUMMARY.md` exists — this plan only marked `CHG-02` complete.

## Next Phase Readiness

- The tracer's end-to-end path (`task changie -- latest` → `v0.14.0`; `task changie -- merge --dry-run` byte-identical to `CHANGELOG.md`) is proven and committed — 01-02-PLAN.md can build `check:changie` and the `ci.yml` wiring directly on top of this baseline with no further config-shape work needed.
- No blockers for 01-02.

## Self-Check: PASSED

- All key files confirmed present on disk (`.changie.yaml`, `.changes/header.tpl.md`, `.changes/unreleased/.gitkeep`, `.changes/v0.14.0.md`, `go.tool-changie.mod`, `go.tool-changie.sum`, `internal/upgrade/changie_shape_test.go`, `Taskfile.yml`, `CONTRIBUTING.md`).
- All three commits (`01057ac1`, `4942a29a`, `9f1a6293`) confirmed in `git log`.
- `git log --reverse --format=%s main..HEAD` confirms RED-before-GREEN ordering.
- Re-ran every `<acceptance_criteria>` command from both tasks and every plan-level `<verification>` command — all pass (see RED/GREEN Evidence sections above).

---
*Phase: 01-changie-baseline*
*Completed: 2026-09-25*
