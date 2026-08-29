---
phase: 03-browse-inspect-navigation
plan: 10
subsystem: testing
tags: [golangci-lint, gofmt, errcheck, staticcheck, ci, taskfile, go-tool-modfile]

requires:
  - phase: 01-engine-seam-wire-protocol-secure-transport
    provides: "go.tool.mod / go.tool-lint.mod / go.tool-proto.mod isolated tool-modfile pattern and TestToolModfilesRemainIsolated guard shape this plan extends"
provides:
  - "A pinned golangci-lint gate (.golangci.yml, go.tool-golangci.mod) enforcing gofmt formatting plus errcheck/ineffassign/staticcheck/unused, wired into task lint:go and CI's lint-go job"
  - "A fully cleared first-run backlog (47 issues across 26 files) — fixed, not suppressed, with one justified inline exception"
  - "Closure of the pre-existing go.tool-proto.mod isolation/vuln-scan gap (present since Phase 1), plus a population-vs-disk guard (TestToolModfilesPopulationMatchesDisk) preventing recurrence for any future tool modfile"
  - "The third-and-final fold of the 2026-08-10 golangci-lint todo, now in .planning/todos/completed/"
affects: []

actuals:
  tokens: 38800
  tasks: 3
  commits: 5

tech-stack:
  added: ["golangci-lint/v2 v2.13.2 (211 modules, own isolated go.tool-golangci.mod)"]
  patterns: ["golangci-lint v2 formatters/exclusions.presets/issues.max-*-issues config shape, verified empirically rather than assumed from v1 muscle memory", "isolatedModfilePaths + TestToolModfilesPopulationMatchesDisk — a named slice checked against an on-disk glob, replacing an inline hardcoded slice that silently passed when a new modfile was absent from it"]

key-files:
  created:
    - .golangci.yml
    - go.tool-golangci.mod
    - go.tool-golangci.sum
  modified:
    - Taskfile.yml
    - .github/workflows/ci.yml
    - internal/upgrade/taskfile_shape_test.go
    - internal/agents/antigravity_test.go
    - internal/agents/opencode.go
    - internal/cli/present/sanitize_test.go
    - internal/cli/upgrade.go
    - internal/cli/upgrade_test.go
    - internal/corpora/coverage.go
    - internal/corpora/coverage_test.go
    - internal/daemon/daemon_test.go
    - internal/graphstore/export.go
    - internal/graphstore/pebble_store.go
    - internal/graphstore/store_test.go
    - internal/indexer/languages_python.go
    - internal/mcp/skill_claims_drift_test.go
    - internal/mcp/tools.go
    - internal/query/node.go
    - internal/query/render_status_test.go
    - internal/uiserver/degrade.go
    - internal/uiserver/handlers_test.go
    - internal/uiserver/server_test.go
    - internal/uiserver/spa_test.go
    - internal/upgrade/release_workflow_shape_test.go
    - internal/upgrade/swap_test.go
    - internal/upgrade/upgrade_test.go
    - tools/corpora/measure_test.go
    - tools/corpora/prose.go
    - tools/corpora/prose_test.go
    - tools/transcriptfreeze/classify.go
    - .planning/todos/completed/2026-08-10-add-golangci-lint-with-gofmt-and-idiomatic-go-linters.md (moved from todos/pending/)

key-decisions:
  - "golangci-lint pinned in a FOURTH isolated tool modfile (go.tool-golangci.mod), never attempting co-location with go.tool-lint.mod — go.tool-lint.mod's own header already documents a live-verified MVS collision from exactly that pattern, and golangci-lint's 211-module tree is the most likely candidate in this repo to win or lose a similar bid."
  - "gofmt (not gofumpt) chosen for the formatter: the ecosystem's own idiomatic tool, distributed with the Go toolchain, for this repository's FIRST formatting gate. gofumpt's stricter superset can be layered on later as a deliberate ratchet (8-file backlog under gofmt vs 23 under gofumpt, 3 of which are generated and excluded either way)."
  - "linters.exclusions.presets: [std-error-handling] enabled — golangci-lint's own documented, maintained convention for errcheck's non-actionable Close/Flush/Print* pattern, and v1's former default-on behavior (v2 made presets opt-in). This is NOT the 'narrow the linter set to fit the backlog' the plan prohibits: it reduced errcheck findings from 50 to 19, and every one of the remaining 19 was fixed by hand, not suppressed."
  - "issues.max-issues-per-linter and issues.max-same-issues both set to 0 (unlimited) — their non-zero defaults (50 and 3) were measured live to silently truncate real findings (gofmt under-reported 3 of the real 8-file backlog), which is precisely the 'gate that cannot fail while appearing to guard' defect this plan's own threat register (T-03-36) warns against."
  - "Task 2's backlog was 47 issues across 26 files, not the 8-file formatting-only set the plan pre-measured and declared in <files> — Task 1's chosen linter set (errcheck/staticcheck/unused, not just gofmt) surfaces a much wider first-run set than gofmt -l alone. Per the plan's own explicit fallback ('fix, never suppress; narrowing the linter set to fit the backlog is the same act as suppressing, one level up'), all 47 were fixed rather than the config narrowed or the extra 19 files left untouched."
  - "The pre-existing go.tool-proto.mod isolation/vuln-scan gap (present since Phase 1, absent from both TestToolModfilesRemainIsolated and the vuln target's binary set) was CLOSED in this plan rather than left as a second recorded-but-unfixed instance of the exact defect this plan diagnoses for golangci-lint's own registration — an orchestrator-level constraint that supersedes the plan's own stated 'do NOT fix it here, record it' scoping. A new TestToolModfilesPopulationMatchesDisk glob-count guard now makes this class of gap impossible to reintroduce silently for any future tool modfile."

requirements-completed: [TODO-CI-01]

coverage:
  - id: D1
    description: "golangci-lint pinned in its own isolated go.tool-golangci.mod, registered with the isolation guard and in scope for the vuln gate"
    requirement: TODO-CI-01
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestToolModfilesRemainIsolated"
        status: pass
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestToolModfilesPopulationMatchesDisk"
        status: pass
      - kind: other
        ref: "command: task vuln (golangci-lint binary scanned, exit 0, advisory)"
        status: pass
    human_judgment: false
  - id: D2
    description: ".golangci.yml authored: v2 schema, errcheck/ineffassign/staticcheck/unused + gofmt formatter, std-error-handling preset, unlimited issue caps, machine-readable # enabled-linters anchor"
    requirement: TODO-CI-01
    verification:
      - kind: other
        ref: "command: rg -o '^# enabled-linters: [0-9]+$' .golangci.yml | wc -l -> 1"
        status: pass
      - kind: other
        ref: "command: task lint:go -> 0 issues, exit 0"
        status: pass
    human_judgment: false
  - id: D3
    description: "task lint:go target exists and lint:go is a leg of the lint wrapper"
    requirement: TODO-CI-01
    verification:
      - kind: other
        ref: "command: task --list-all | rg '^\\* lint:go:' -> 1 match"
        status: pass
    human_judgment: false
  - id: D4
    description: "The full first-run backlog (47 issues, 26 files) was fixed, never suppressed, with one justified nolint exception"
    requirement: TODO-CI-01
    verification:
      - kind: other
        ref: "command: task lint:go -> 0 issues; task test:unit -> 50/50 ok; go vet ./... -> clean"
        status: pass
    human_judgment: false
  - id: D5
    description: "lint-go CI job wired into ci.yml and bound to the single-definition guard (inScopeJobs)"
    requirement: TODO-CI-01
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestWorkflowRunBodiesInvokeTask"
        status: pass
    human_judgment: false
  - id: D6
    description: "The gate was demonstrated RED twice (a formatting violation, an idiomatic violation) in internal/corpora/coverage_test.go, both reverted byte-identically"
    requirement: TODO-CI-01
    verification:
      - kind: other
        ref: "command sequence: shasum before (083de1ad...dde30) -> plant -> task lint:go non-zero, names file -> git checkout -- file -> shasum after (083de1ad...dde30, identical) -> task lint:go exit 0, repeated for both violation classes"
        status: pass
    human_judgment: false
  - id: D7
    description: "The twice-folded todo is closed with a phase marker and Resolution section"
    requirement: TODO-CI-01
    verification:
      - kind: other
        ref: "command: ls .planning/todos/completed/ | rg '2026-08-10-add-golangci-lint...' -> 1; ls .planning/todos/pending/ | rg 'golangci' -> 0"
        status: pass
    human_judgment: false
  - id: D8
    description: "The pre-existing go.tool-proto.mod isolation/vuln-scan gap was closed alongside golangci-lint's own registration"
    requirement: TODO-CI-01
    verification:
      - kind: other
        ref: "command: task vuln (buf/protoc-gen-go/protoc-gen-connect-go built and scanned from go.tool-proto.mod, exit 0, advisory)"
        status: pass
    human_judgment: false

duration: ~90min
completed: 2026-08-29
status: complete
---

# Phase 3 Plan 10: golangci-lint gate — pinned, configured, backlog cleared, CI-wired Summary

**Third-and-final fold of a golangci-lint todo landed for real: a fourth isolated tool modfile, a v2 config with gofmt + errcheck/ineffassign/staticcheck/unused, a 47-issue first-run backlog fixed by hand across 26 files, CI wiring proven RED-then-GREEN on a planted violation, and a pre-existing sibling isolation gap (go.tool-proto.mod) closed in the same pass.**

## Performance

- **Duration:** ~90 min (not precisely epoch-tracked from session start; commit range spans 2026-08-29T00:08–00:19 for the final three commits, plus prior investigation/fix time not captured by a `date -u` timestamp at session start)
- **Tasks:** 3 (plus one orchestrator-directed extension of Task 1's scope, documented below)
- **Files modified:** 34 (3 created, 31 modified/moved)
- **Commits:** 5

## Accomplishments

- `.golangci.yml` + `go.tool-golangci.mod`/`.sum`: golangci-lint v2.13.2 pinned in its own isolated tool modfile (211 modules), config enabling gofmt formatting plus errcheck/ineffassign/staticcheck/unused (the todo's own four named categories), with a machine-readable `# enabled-linters: 5` anchor line and unlimited issue-reporting caps.
- `task lint:go` runs the gate with the linter's own exit status as the verdict (no capture, no pipe); added as a leg of the `lint` contributor wrapper.
- The full first-run backlog — 47 issues across 26 files, not the 8-file formatting-only set originally pre-measured — was fixed by hand, file by file, with exactly one justified `//nolint:staticcheck` exception.
- CI wiring: a `lint-go` job in `.github/workflows/ci.yml`, bound by the single-definition guard, demonstrated RED on a planted formatting violation and a planted idiomatic violation, each reverted byte-identically (sha256-verified) and re-confirmed green.
- The `vuln` gate's binary set grew from four to eight (golangci-lint plus, closing a pre-existing gap, buf/protoc-gen-go/protoc-gen-connect-go), with the coverage figure in its `desc:` recomputed from a stale "357" to a freshly measured 571.
- `TestToolModfilesRemainIsolated` was restructured around a named `isolatedModfilePaths` slice and paired with a new `TestToolModfilesPopulationMatchesDisk` glob-count guard, so a future fifth tool modfile landing without registration fails loudly instead of silently passing.
- The twice-folded todo is closed: `.planning/todos/completed/2026-08-10-add-golangci-lint-with-gofmt-and-idiomatic-go-linters.md`.

## Task Commits

1. **Task 1: Pin the linter, author the config, and add the task target** — `2fd3cf14` (feat)
2. **Task 2: Clear the first-run backlog by fixing, never suppressing** — `dd90ecfb` (fix)
3. **Task 3: Wire it into CI, bind it to the single-definition guard, and prove it RED** — `1e20225d` (feat, todo-move only — see deviation below) + `3957e7e6` (feat, the actual CI wiring content)
4. **Orchestrator-directed extension of Task 1: close the go.tool-proto.mod isolation/vuln gap** — `ee925da5` (fix)

## Files Created/Modified

See frontmatter `key-files`. The 26 files fixed in Task 2 beyond the plan's declared 8-file formatting backlog are listed there in full; the enumeration and reasoning for each is in the Deviations section below and in the Task 2 commit message (`dd90ecfb`).

## Decisions Made

See frontmatter `key-decisions`. In addition, recorded here per the plan's own acceptance-criteria obligations:

**Enabled linters (N=5) and reasons** — see `.golangci.yml`'s own header comment for the full text; summarized:
- `errcheck` — "unused-result checking" (todo's own wording)
- `ineffassign` — "ineffectual assignments" (todo's own wording)
- `staticcheck` — "static analysis" (todo's own wording; v2 absorbs the old gosimple/stylecheck)
- `unused` — "unused identifiers" (todo's own wording)
- `gofmt` (formatter) — this repository's first formatting gate

**Considered and rejected:** `govet` (redundant with the existing `task vet` target's unconditional `go vet ./...`), `revive` (large, team-tunable rule set — deferred to a dedicated future pass rather than enabled with unreviewed defaults), `gofumpt` (stricter superset — deferred as a later ratchet).

**Exact module path / resolved version installed:** `github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.2`, pinned character-for-character (the `/v2` element intact, no lookalike path). No checksum-verification setting (`GOFLAGS`/`GONOSUMDB`/`GONOSUMCHECK`/`GOPRIVATE`/`GOSUMDB`) was touched — `go env GOSUMDB` still reports `sum.golang.org` after resolution.

**Module count arithmetic (vuln `desc:`):** the pre-existing "357 measured" figure was already stale on its own (a deduplicated union of just `go.tool.mod` + `go.tool-lint.mod` measured 369 on this working tree, before this plan's changes). golangci-lint's own modfile contributes 211 modules; closing the go.tool-proto.mod gap (see below) adds its 89. The final recomputed, deduplicated union across all four scanned tool modfiles is **571**, written into the `vuln` target's `desc:`.

**Formatting-strictness choice:** plain `gofmt`, not `gofumpt` — see `.golangci.yml`'s header for the full reasoning (landing the first formatting gate is itself the win; gofumpt is a deliberate future ratchet).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Default golangci-lint issue-reporting caps would have made this exact gate vacuous**
- **Found during:** Task 1, while measuring the true formatting backlog
- **Issue:** golangci-lint's default `issues.max-issues-per-linter` (50) and `issues.max-same-issues` (3) silently drop findings beyond those caps. Measured live: with defaults, `gofmt` under-reported only 3 of the real 8-file backlog. A gate that silently truncates its own findings is precisely the "reports N of the real backlog and calls it clean" defect this plan's own threat register (T-03-36) names.
- **Fix:** Added `issues.max-issues-per-linter: 0` and `issues.max-same-issues: 0` to `.golangci.yml` (unlimited).
- **Files modified:** `.golangci.yml`
- **Verification:** Re-ran with caps disabled — the true backlog (47 issues) surfaced, matching `gofmt -l`'s independently-measured 8-file count exactly.
- **Committed in:** `2fd3cf14`

**2. [Rule 2 - Missing Critical] `linters.exclusions.presets: [std-error-handling]` enabled**
- **Found during:** Task 1, first full lint run (68 raw issues, 50 from errcheck)
- **Issue:** golangci-lint v2 made the presets that shipped default-on in v1 opt-in. Without `std-error-handling`, errcheck flags every ignored `defer f.Close()`, `fmt.Fprintln(stderr, ...)`, etc. — the exact non-actionable pattern the preset exists to filter.
- **Fix:** Enabled the preset (golangci-lint's own documented, maintained convention). This is explicitly NOT the "narrow the linter set to fit the backlog" the plan prohibits: measured effect was 50 → 19 errcheck findings, and every one of the remaining 19 was fixed by hand in Task 2, not suppressed.
- **Files modified:** `.golangci.yml`
- **Verification:** `task lint:go` after the preset still surfaced and required fixing 19 real errcheck findings.
- **Committed in:** `2fd3cf14`

**3. [Rule 1 - Bug] The declared Task 2 backlog (8 files, gofmt only) was a substantial undercount of the true first-run backlog**
- **Found during:** Task 2, first `task lint:go` run
- **Issue:** The plan's `<files>` list for Task 2 was pre-measured against `gofmt -l`/`gofumpt -l` only. Task 1's actual linter set (errcheck/ineffassign/staticcheck/unused, not just gofmt) surfaced 47 issues across 26 files — 19 files and 39 findings beyond the declared 8.
- **Fix:** Per the plan's own explicit fallback instruction ("If the backlog turns out to be large enough that fixing it all would swamp this plan, STOP and report the count rather than narrowing the linter set to make it smaller"), judged the backlog (all single/few-line mechanical fixes, no architectural changes) tractable within this plan's budget and fixed all 47 — none suppressed except one justified case (see #4).
- **Files modified:** 26 files total (see frontmatter `key-files`); full pre/post inventory in commit `dd90ecfb`.
- **Verification:** `task lint:go` → 0 issues; `task test:unit` → 50/50 ok; `go vet ./...` clean; `.golangci.yml` proven byte-unchanged since Task 1's commit.
- **Committed in:** `dd90ecfb`

**4. [Rule 1 - Bug, one justified suppression] `internal/uiserver/degrade.go`'s ST1005 finding**
- **Found during:** Task 2
- **Issue:** `indexingInProgressMessage` is used both as an `errors.New()` argument (where Go convention says error strings should not end in punctuation) and as the exact user-facing prose in a typed `IndexingInProgress.Message` detail crossing to an unauthenticated browser caller (its own doc comment: "the ONE fixed, generic sentence"). Splitting the string into two would violate that explicit single-source-of-truth design.
- **Fix:** `//nolint:staticcheck // ST1005: intentional, see comment above` on the `errors.New(...)` line, with an explanatory comment above stating exactly why this is a deliberate exception, not a suppressed bug.
- **Files modified:** `internal/uiserver/degrade.go`
- **Verification:** `rg -o 'nolint' --glob '!vendor' .` count: 4 before this plan → 5 after (this one addition, accounted for).
- **Committed in:** `dd90ecfb`

**5. [Rule 2 - Missing Critical, orchestrator-directed] Pre-existing `go.tool-proto.mod` isolation/vuln-scan gap closed**
- **Found during:** Task 1, while registering `go.tool-golangci.mod`
- **Issue:** `TestToolModfilesRemainIsolated` and the `vuln` gate's binary set both used hardcoded, inline lists. `go.tool-proto.mod` (present since Phase 1) was absent from both — inspected by nothing, scanned by nothing, with both guards passing green regardless. The plan's own text ("do NOT fix it here — RECORD it") would have left this defect as accepted debt in the same commit that adds a fourth modfile literally next to it. An orchestrator-level constraint (not present in the plan's own text) required closing this now.
- **Fix:** `isolatedModfilePaths` replaces the inline hardcoded slice with a named slice of all four modfiles; new `TestToolModfilesPopulationMatchesDisk` globs `go.tool*.mod` and asserts the on-disk count matches, closing the class of gap (not just this one instance) for any future modfile. `vuln` extended to build/scan buf, protoc-gen-go, protoc-gen-connect-go from `go.tool-proto.mod` (eight binaries total). `github.com/bufbuild/buf` added to `forbiddenToolPackages`; `google.golang.org/protobuf` and `connectrpc.com/connect` deliberately NOT added — both are legitimate runtime dependencies of the main module's generated `.pb.go`/`.connect.go` files.
- **Files modified:** `internal/upgrade/taskfile_shape_test.go`, `Taskfile.yml`
- **Verification:** `TestToolModfilesRemainIsolated` and `TestToolModfilesPopulationMatchesDisk` both pass; `task vuln` built and scanned all 8 binaries (buf: DETECTED, 6 Go-stdlib CVEs go1.26.5→go1.26.6, advisory; protoc-gen-go/protoc-gen-connect-go: CLEAN).
- **Committed in:** `ee925da5`

---

**Total deviations:** 5 auto-fixed (2 Rule 2 config-correctness additions, 2 Rule 1 backlog/suppression handling, 1 Rule 2 orchestrator-directed scope extension). **Impact:** All necessary for the gate to actually discriminate (caps/preset fixes), for "fix don't suppress" to hold in practice (backlog + one justified exception), and for the isolation-guard defect this plan itself diagnoses not to have a second live instance immediately beside the fix. No scope creep beyond what correctness/the orchestrator's explicit constraint required.

## Non-vacuity proof (rule `84d1gfpywd`)

The lint gate was demonstrated to DISCRIMINATE, per the plan's own requirement, in `internal/corpora/coverage_test.go` (already in Task 2's fixed backlog, in the `go list ./...` package graph):

| Plant | Before hash | After `task lint:go` | Naming | After revert | After hash | Confirmed green |
|---|---|---|---|---|---|---|
| Extra indentation level (formatting) | `083de1ad1dd3450d05a994ad06ec91ac09d5858e7098b89026c2a3c0576dde30` | exit 201, `gofmt: 1` | `internal/corpora/coverage_test.go:123:1: File is not properly formatted (gofmt)` | `git checkout -- internal/corpora/coverage_test.go` | `083de1ad1dd3450d05a994ad06ec91ac09d5858e7098b89026c2a3c0576dde30` (identical) | exit 0, `0 issues.` |
| Ineffectual assignment (idiomatic) | `083de1ad1dd3450d05a994ad06ec91ac09d5858e7098b89026c2a3c0576dde30` | exit 201, `ineffassign: 1` | `internal/corpora/coverage_test.go:163:2: ineffectual assignment to scratch (ineffassign)` | `git checkout -- internal/corpora/coverage_test.go` | `083de1ad1dd3450d05a994ad06ec91ac09d5858e7098b89026c2a3c0576dde30` (identical) | exit 0, `0 issues.` |

## Issues Encountered

None beyond the deviations documented above — no unresolved problems.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

Per the plan's own text, this plan's outcome is reported separately and does not gate any of Phase 3's five browse success criteria. The gate is live, green, and CI-enforced going forward; `task test:unit` (50/50 ok), `go vet ./...` (clean), `task lint`, `task lint:actions`, and `task vuln` (advisory, unchanged posture) all pass on the current tree. No blockers for subsequent phases.

---
*Phase: 03-browse-inspect-navigation*
*Completed: 2026-08-29*

## Self-Check: PASSED

All key files found on disk (.golangci.yml, go.tool-golangci.mod, go.tool-golangci.sum, completed todo, this SUMMARY). All 6 commit hashes (2fd3cf14, dd90ecfb, 1e20225d, 3957e7e6, ee925da5, fa11db1a) confirmed present in git log.
