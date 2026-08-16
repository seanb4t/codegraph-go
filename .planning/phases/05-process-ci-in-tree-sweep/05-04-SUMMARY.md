---
phase: 05-process-ci-in-tree-sweep
plan: 04
subsystem: internal-query
tags: [comment-sweep, de-framing, code-01, query, capability-matrix, golden, identifier-rename]

requires:
  - phase: 05-process-ci-in-tree-sweep (plan 05-01, wave 1)
    provides: codegraph migrate removed entirely — the migrate-era ts-* golden fixtures and migration-cursor API are gone, so this census counts the real internal/query population, not migrate-inflated noise
provides:
  - "internal/query's 12 flagship files (expand.go, explore_gate.go, explore.go, engine.go, files.go, worktree_notice.go, gather.go, scoring.go, seeding.go, tokenize.go, rwr.go, node.go + node_test.go) reworded off TS-comparison framing — 'ports TS CodeGraph 1.3.1's' headers, 'ported VERBATIM'/'ported verbatim' claims, and 'TS's X'/'mirrors TS'/'matching TS' possessives all resolved on the project's own terms"
  - "the renderers/search/validate/status tail (render_status.go, render_results.go, render_markdown.go+test, search.go+test, validate.go, status.go, files_status_test.go, render_status_test.go) reworded — incl. the five review-H3 rows (engine.go:179, explore.go:111/211, files.go:25/97-99, worktree_notice.go:43) and render_status_test.go:132/143's comment+message reword with the node:sqlite sink comparison itself untouched"
  - "syntheticParity* identifier family renamed to behavioral names (D-09): copySyntheticParityFixture -> copyBehavioralFixture, syntheticParityEngine -> behavioralEngine (internal/query/explore_test.go), syntheticParitySrc -> behavioralCorpusSrc (testdata/golden/behavioral_test.go) — definitions + all call sites in the same commit"
  - "internal/indexer/capability/matrix.go + matrix_test.go's 'golden-parity' code comments reworded to 'behavioral golden test/fixture'/'golden harness' — doc table values and Gaps cells byte-identical"
  - "phase-wide A3 retained-use inventory run over internal/query/ (rg -e TS -e parity -e port -e original -e upstream) — every hit classified KEEP (historical/dataflow/in-repo/false-positive) or FIXED; 9 additional in-scope hits found and closed (expand.go, explore_gate.go, rwr.go, traverse.go); 1 genuine out-of-scope hit (traverse_test.go:780) logged to WINDOWS.md rather than edited"
affects: [06-process-ci-in-tree-sweep-plan-06, ship-gate]

actuals:
  tokens: 19500
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Row-by-row census-driven reword (research §4.1 as the work list), never a regex find-replace over TS/port/original — the same term-by-term discipline plan 05-05 established for its packages"
    - "Positive A3 retained-use inventory as the phase-wide verification gate: every remaining TS|parity|port|original|upstream hit in the touched package is read and classified, not inferred from a zero-hit grep"

key-files:
  created: []
  modified:
    - internal/query/expand.go
    - internal/query/explore_gate.go
    - internal/query/explore.go
    - internal/query/engine.go
    - internal/query/files.go
    - internal/query/worktree_notice.go
    - internal/query/gather.go
    - internal/query/node.go
    - internal/query/node_test.go
    - internal/query/render_markdown.go
    - internal/query/render_markdown_test.go
    - internal/query/render_results.go
    - internal/query/render_status.go
    - internal/query/render_status_test.go
    - internal/query/rwr.go
    - internal/query/scoring.go
    - internal/query/seeding.go
    - internal/query/tokenize.go
    - internal/query/search.go
    - internal/query/search_test.go
    - internal/query/validate.go
    - internal/query/status.go
    - internal/query/traverse.go
    - internal/query/files_status_test.go
    - internal/query/explore_test.go
    - internal/indexer/capability/matrix.go
    - internal/indexer/capability/matrix_test.go
    - testdata/golden/behavioral_test.go
    - .planning/WINDOWS.md

key-decisions:
  - "render_status_test.go is edited by this plan (task 2's <files> and review-H3 rows explicitly require it) even though it is not separately listed in this plan's frontmatter files_modified — the task body's explicit instruction takes precedence over the frontmatter omission; noted here rather than silently expanding scope."
  - "traverse.go IS in files_modified but wasn't assigned to any task's <files> list — its census row (:532, R class) and two adjacent TS-comparison rows (:526, :625) were fixed as part of closing the phase-wide A3 gate, since the file is within the plan's declared scope even though no task explicitly named it."
  - "traverse_test.go carries the identical framing (line 780) but is NOT in files_modified — left unedited and logged to .planning/WINDOWS.md (entry 14) per the scope-discipline rule (editing undeclared files risks colliding with a sibling plan's ownership), rather than silently expanding this plan's file set."
  - "status.go's D-05 decision table (TS-key -> Go/Pebble-rendering -> rationale, lines 21-46) is classified KEEP wholesale except the one R-flagged row (:29, 'shape parity' -> 'output shape stability') and the explicit K row (:44, historical). The table is a one-time schema-migration record, not ongoing comparison framing, and task 2's own automated verify explicitly scopes status.go out of its reword grep — consistent with the census (§4.1) flagging only those two rows in this file."
  - "'Historical/provenance' TS mentions ('the live TS dist is no longer readable on this machine', 'the TS install's dist JS became unreadable') are KEPT verbatim across expand.go/gather.go/scoring.go/seeding.go/explore.go — they explain WHY a constant is a documented default rather than a captured exact value (a one-time provenance fact), not an ongoing 'we match TS' comparison claim."

patterns-established:
  - "Dataflow-sense word disambiguation: 'upstream'/'original'/'port' appear throughout this package in non-comparison senses (upstream = earlier pipeline stage; original = this codebase's own prior code path; port = Go import statement / 'exports' substring) — the A3 gate's raw grep over-matches these by design (a CLASSIFY gate, not a zero-assertion gate) and each was read in context rather than blanket-reworded."

requirements-completed: [CODE-01]

coverage:
  - id: D1
    description: "internal/query's 12 flagship files + node_test.go reworded off 'ports TS CodeGraph 1.3.1's'/'ported VERBATIM'/'mirrors TS'/'matching TS' framing, including the five review-H3 rows; status.go:44's D-05 history note preserved verbatim"
    requirement: "CODE-01"
    verification:
      - kind: unit
        ref: "go test -count=1 ./internal/query/..."
        status: pass
      - kind: other
        ref: "rg -n -e 'ports CodeGraph' -e 'ported VERBATIM' -e 'ported verbatim' -e 'matches CodeGraph 1.3.1' -e 'TS-parity explore' -e 'mirrors TS' -e 'matching TS' over the 12 flagship files == 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "renderers/search/validate/status tail reworded (render_status.go, render_results.go stale golden_parity_test.go citation re-pointed, render_markdown.go+test, search.go+test, validate.go, files_status_test.go, render_status_test.go:132/143 comment+message reword with the node:sqlite sink comparison untouched)"
    requirement: "CODE-01"
    verification:
      - kind: unit
        ref: "go test -count=1 ./internal/query/..."
        status: pass
      - kind: other
        ref: "rg -n -e 'TS CodeGraph' -e \"TS.s\" -e 'matching TS' -e 'shape parity' -e 'golden-parity harness' -e 'golden_parity_test' over the task-2 file set == 0"
        status: pass
    human_judgment: false
  - id: D3
    description: "syntheticParity* identifier family renamed to behavioral names (definitions + all call sites); capability/matrix.go+matrix_test.go 'golden-parity' comments reworded with doc table values untouched"
    requirement: "CODE-01"
    verification:
      - kind: unit
        ref: "go test -count=1 ./internal/query/... ./internal/indexer/capability/... ./testdata/golden/..."
        status: pass
      - kind: other
        ref: "rg -n 'syntheticParity' internal/ testdata/ == 0"
        status: pass
    human_judgment: false
  - id: D4
    description: "phase-wide A3 retained-use inventory over internal/query/ — every TS|parity|port|original|upstream hit classified; 9 in-scope hits closed (expand.go, explore_gate.go, rwr.go, traverse.go); 1 out-of-scope hit (traverse_test.go:780) logged to WINDOWS.md; full go test -count=1 ./... green except the pre-existing, documented internal/daemon watchdog flake"
    requirement: "CODE-01"
    verification:
      - kind: unit
        ref: "go test -count=1 ./... (all packages pass except internal/daemon's pre-existing TestRunWatchdogCancelsRunOnSimulatedReparent load-sensitive flake, WINDOWS.md entry 12, unrelated to this plan's files)"
        status: pass
    human_judgment: false

duration: ~25min
completed: 2026-08-16
status: complete
---

# Phase 5 Plan 04: internal/query CODE-01 sweep (flagship surface) Summary

**Reworded ~110 comparison-framing rows across `internal/query`'s 12 flagship files, its renderers/search/validate/status tail, and the capability matrix's golden-parity comments — plus the D-09 `syntheticParity*` -> behavioral identifier rename — leaving every retained `TS`/`parity`/`port`/`original`/`upstream` hit in the package explicitly classified in a phase-wide A3 inventory.**

## Performance

- **Duration:** ~25 min
- **Started:** 2026-08-15 (session start)
- **Completed:** 2026-08-16T00:52:59Z
- **Tasks:** 3/3 completed (plus one supplementary A3-closure commit)
- **Files modified:** 29 (28 source/test files + `.planning/WINDOWS.md`)

## Accomplishments

- The 12 flagship `internal/query` files (`expand.go`, `explore_gate.go`, `explore.go`, `engine.go`, `files.go`, `worktree_notice.go`, `gather.go`, `scoring.go`, `seeding.go`, `tokenize.go`, `rwr.go`, `node.go`) plus `node_test.go` no longer claim to "port TS CodeGraph 1.3.1's" anything — every header now opens with "implements"/"provides" and describes the algorithm on its own terms; `node.go`'s `GENERATED_PATTERNS` list keeps its byte-for-byte-transcription behavior but the comment no longer claims a TS port. Includes all five review-H3 rows (`engine.go:179`, `explore.go:111/211`, `files.go:25/97-99`, `worktree_notice.go:43`).
- The renderers/search/validate/status tail (`render_status.go`, `render_results.go`, `render_markdown.go`+test, `search.go`+test, `validate.go`, `status.go`, `files_status_test.go`, `render_status_test.go`) reworded — `render_results.go`'s stale `golden_parity_test.go` citation now points at `behavioral_test.go`; `render_status_test.go:132/143` (review H3) reword the comment and error-message text only, leaving the `node:sqlite` string comparison itself byte-identical.
- `syntheticParity*` identifiers renamed to behavioral names in the same commit as their call sites (D-09): `copySyntheticParityFixture` -> `copyBehavioralFixture`, `syntheticParityEngine` -> `behavioralEngine` (`internal/query/explore_test.go`), `syntheticParitySrc` -> `behavioralCorpusSrc` (`testdata/golden/behavioral_test.go`, 5 call sites). `rg 'syntheticParity' internal/ testdata/` is empty.
- `internal/indexer/capability/matrix.go`/`matrix_test.go`'s "golden-parity" code comments reworded to "behavioral golden test/fixture"/"golden harness" — the doc table values and Gaps cells are byte-identical (`matrix_test.go`'s own byte-compare still passes).
- Ran the phase-wide A3 retained-use inventory gate (`rg -e TS -e parity -e port -e original -e upstream internal/query/`) after tasks 1-3 and found 9 additional in-scope hits (in `expand.go`, `explore_gate.go`, `rwr.go`, and `traverse.go` — all declared in this plan's `files_modified` but not assigned to any task's per-file list) plus one genuine out-of-scope hit (`traverse_test.go:780`, not in `files_modified`) — closed the 9, logged the 1 to `.planning/WINDOWS.md` (entry 14).

## Task Commits

Each task was committed atomically:

1. **Task 1 (tracer): The flagship file headers and their direct body claims** — `60a489f` (docs)
2. **Task 2: The renderers, the tail files, and the stale harness renames** — `5d138b5` (docs)
3. **Task 3: The identifier and matrix renames** — `ac810f5` (docs)
4. **Supplementary: A3 phase-wide inventory closure** — `45055be` (docs)

**Plan metadata:** this commit (docs: complete plan)

_Note: this is a comment/identifier-only sweep — no test-driven-development gates apply; every commit is `docs(05-04): ...` per the plan's Conventional Commits scope._

## Term-by-Term Retained-Use Classification (A3 inventory)

Ran `rg -n -e "\bTS\b" -e "parity" -e "port" -e "original" -e "upstream" internal/query/` after all edits landed (143 raw hits; the bare `port`/`original`/`upstream` patterns over-match by design — `import`/`export`/`report`/`support` all contain `port`, and `original`/`upstream` have ordinary non-comparison senses in Go doc comments). After filtering `import (` statements and the `report(ed)/export(ed)/import(ed/able)` substring family, 34 rows remained for hand classification:

### KEEP (recorded reasons)

| Location | Text (excerpt) | Reason |
|---|---|---|
| `node.go:312`, `node.go:415` | "via the original single-def RenderNode path" | In-repo restatement — refers to this file's own pre-existing code path, not a TS comparison |
| `render_status.go:30` | "decided upstream of these renderers" | Dataflow sense of "upstream" (earlier in the call chain), not the origin project |
| `tokenize.go:10` | "(exports.STOP_WORDS, query-utils.js:...)" | `port` substring inside `exports` (JS module property citation) — not the word "port" |
| `expand.go:5`, `explore.go:108`, `scoring.go:174`, `gather.go:51/574/671` | "the live TS dist is no longer readable on this machine" / "the TS install's dist JS became unreadable" | Past-tense provenance — explains why a constant is a documented default rather than a captured exact value, not an ongoing "we match TS" claim |
| `status.go:21,25,35,41,44,76` | the D-05 decision table (TS-key -> Go/Pebble rendering -> rationale) | One-time schema-migration record (historical), not ongoing comparison framing; census (§4.1) flags only `:29` (fixed) and `:44` (explicit K) in this file; task 2's automated verify explicitly scopes `status.go` out of its reword grep |
| `explore_gate.go:20` | "the functions below accept the upstream per-file..." | Dataflow sense of "upstream" |
| `files_status_test.go:489,511` | "the frozen TS golden oracle keeps it stripped" | Describes the golden fixture's own historical provenance/naming, not an ongoing comparison |
| `engine.go:72` | "OpenAt supplies the caller's original starting directory" | Ordinary English "original" (initial/first), not TS-comparison |
| `rwr.go:94` | "upstream subgraph-gathering caps" | Dataflow sense of "upstream" |
| `explore.go:111`, `scoring.go:6`, `seeding.go:27`, `gather.go:4` | "not the exact original step function" / "original source is no longer readable" | Already-reworded neutral text from this plan's own edits (task 1) — no TS attribution remains |
| `scoring.go:16` | "not the exact upstream wiring of" | Dataflow sense — the next sentence names "earlier pipeline stages" |
| `seeding_test.go:435,439` | `"Widget MyProject Report myproject"` / `seedingContainsID(got, "Report")` | Literal test-fixture strings ("Report" as an English word), unrelated to any origin project |
| `gather.go:477` | "fileDir returns the directory **port**ion..." | `port` substring inside "portion" |
| `render_results.go:32` | "ordered upstream before it gets here" | Dataflow sense of "upstream" |
| `render_results.go:151` | "renderFileTreeMarkdown ports internal/cli/files.go's printFileTree" | In-repo pipeline reference (census K class) — this function reuses/adapts another function IN THIS codebase, not TS |
| `render_markdown_test.go:500` | "must still render via the original single-def RenderNode path" | In-repo restatement, same class as node.go:312/415 |
| `traverse.go:239` | "upstream iteration-order contract" | Dataflow term (census K class) |

### REMAINING — found but NOT fixed here (out of this plan's declared scope)

| File:line | Text | Why not fixed here |
|---|---|---|
| `internal/query/traverse_test.go:780` | "to prove Affected's depth-bounded BFS and TS test-files-as-leaves pruning" | Genuine CODE-01 framing, but `traverse_test.go` is not in this plan's `files_modified` (only `traverse.go` is) and no task body names it. Logged to `.planning/WINDOWS.md` (entry `14`, kind `deviation`) rather than edited, per the phase's file-ownership discipline — editing an undeclared file risks colliding with a sibling plan's scope. |

No work-item regex was used anywhere in this plan; every FIXED row above (task 1/2/3 commits) was a hand-edit against a specific census row or A3-gate finding.

## Files Created/Modified

See `key-files.modified` in frontmatter (28 source/test files across `internal/query`, `internal/indexer/capability`, and `testdata/golden`, plus `.planning/WINDOWS.md`).

## Decisions Made

1. **`render_status_test.go` edited despite a `files_modified` frontmatter omission.** Task 2's `<files>` list and its review-H3 instructions explicitly require editing `render_status_test.go:132/143`; the task body's explicit, line-numbered instruction takes precedence over the frontmatter list's apparent omission. Noted rather than silently expanded.
2. **`traverse.go` fixed even though no task's `<files>` list named it.** It IS declared in `files_modified`, and the census (§4.1) explicitly flags `traverse.go:532` (R class). Closed as part of the A3-gate supplementary commit rather than left half-swept.
3. **`traverse_test.go` left untouched and logged, not edited.** It carries the identical framing (line 780) but is absent from BOTH `files_modified` and every task's `<files>` list — unlike `traverse.go`, there is no textual basis to treat it as in-scope. Logged to `.planning/WINDOWS.md` per the scope-discipline pattern plan 05-05 established for its own out-of-scope finds.
4. **`status.go`'s D-05 decision table left almost entirely as-is.** The table's purpose IS documenting a one-time TS-key -> Go-key schema mapping (historical/migration record) — the census only flagged 2 of its ~7 TS-mentioning rows as needing rewording, and task 2's own automated verify command deliberately excludes `status.go` from its file list. Fixed only the one R-flagged row (`:29`); left the rest, consistent with the census's own scoping.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical] 9 additional CODE-01 framing rows found by the phase-wide A3 gate, in files already declared in `files_modified` but not assigned to any task's per-file list**
- **Found during:** Post-task-3 A3 retained-use inventory (`rg -e TS -e parity -e port -e original -e upstream internal/query/`)
- **Issue:** `expand.go`'s `expandHierarchyKinds` comment ("that whole TS set" / "H10's slightly narrower TS wording"), `explore_gate.go`'s `centralFileSelection`/`fiveTierFileSort` tie-break comments ("TS's own Map/Set iteration order" / "TS files carry no stable Id"), `rwr.go`'s "ported in later plans" self-reference, and `traverse.go`'s `Affected` doc comment + `isTestSymbol` branch comment ("TS 1.3.1's test-files-as-leaves pruning rule" / "parity is structural" / "TS test-files-as-leaves semantics") all carried unreworded comparison framing that neither task 1's nor task 2's automated verify grep happened to match (the patterns are phrase-specific, not a blanket "TS" scan).
- **Fix:** Reworded all 9 rows to own-terms language ("the documented X", "implemented in later plans", "behavior is proved structurally"), consistent with the rest of the sweep. No function body, export, or test-asserted string changed.
- **Verified:** `go build ./...` and `go test -count=1 ./internal/query/... ./internal/indexer/capability/... ./testdata/golden/...` green; `gofmt -l` clean.
- **Files modified:** `internal/query/expand.go`, `internal/query/explore_gate.go`, `internal/query/rwr.go`, `internal/query/traverse.go`.
- **Commit:** `45055be`.

**2. [Rule 2 - Missing critical] 1 genuine out-of-scope CODE-01 hit logged to the defect ledger**
- **Found during:** Same A3 pass as above
- **Issue:** `traverse_test.go:780` carries the identical "TS test-files-as-leaves pruning" framing as the `traverse.go` rows just fixed, but the test file itself is absent from this plan's `files_modified`.
- **Fix:** Did NOT edit it (scope discipline). Logged to `.planning/WINDOWS.md` as entry 14 (`kind: deviation`, `phase: 05`) so it's visible at the ship gate.
- **Files modified:** `.planning/WINDOWS.md` only.
- **Commit:** `45055be`.

---

**Total deviations:** 2 auto-fixed (1 Rule 2 missing-critical closure across 4 files, 1 Rule 2 missing-critical ledger entry)
**Impact on plan:** Both are within CODE-01's "term-by-term, nothing left referencing" mandate; neither changes runtime behavior, test assertions, or exported symbols. Zero scope creep — the 4 files fixed were already declared in `files_modified`; the 1 file logged instead of fixed was correctly excluded.

## Issues Encountered

- `go test -count=1 ./...` (full suite, run after all four commits) reports `FAIL github.com/seanb4t/codegraph-go/internal/daemon` (`TestRunWatchdogCancelsRunOnSimulatedReparent`, 250s timeout under this session's concurrent load). This is the repo's documented, pre-existing "daemon extreme-load timeout tail" (already logged as `.planning/WINDOWS.md` entry 12 by a sibling plan in this wave; STATE.md: "52/52 real CI runs clean — the flake was always local"). `internal/daemon` has zero import relationship to any file this plan touches (`internal/query`, `internal/indexer/capability`, `testdata/golden`). Not fixed (out of scope, pre-existing, already tracked).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `internal/query`'s flagship surface, its renderers/search/validate/status tail, the `syntheticParity*` identifier family, and the capability matrix's golden-parity comments are fully swept — no comparison framing remains except the recorded KEEP class (historical provenance, dataflow-sense words, in-repo restatement, and the D-05 decision table).
- `traverse_test.go:780` is the one open CODE-01 gap from this plan, tracked in `.planning/WINDOWS.md` entry 14 for a future sweep pass or explicit waiver before `/gsd-ship`.
- `go build ./...` and `go test -count=1 ./...` are green (modulo the pre-existing, already-tracked `internal/daemon` flake noted above) as of this plan's final commit.
- No file overlap with plan 05-06 (`corpus/behavioral/src/**` re-freeze) or the other wave-2 plans — this plan's edits are confined to `internal/query`, `internal/indexer/capability`, and `testdata/golden/behavioral_test.go`.

---
*Phase: 05-process-ci-in-tree-sweep*
*Completed: 2026-08-16*

## Self-Check

- Commit `60a489f` — FOUND (`git log --oneline --all | rg 60a489f`)
- Commit `5d138b5` — FOUND
- Commit `ac810f5` — FOUND
- Commit `45055be` — FOUND
- `internal/query/expand.go` — FOUND
- `internal/query/explore_gate.go` — FOUND
- `internal/query/traverse.go` — FOUND
- `internal/query/explore_test.go` — FOUND
- `testdata/golden/behavioral_test.go` — FOUND
- `internal/indexer/capability/matrix.go` — FOUND
- `.planning/WINDOWS.md` — FOUND (entry 14 present)
- `rg 'syntheticParity' internal/ testdata/` — 0 hits
- `go test -count=1 ./internal/query/... ./internal/indexer/capability/... ./testdata/golden/...` — PASS
- `go test -count=1 ./...` — PASS except the pre-existing, already-tracked `internal/daemon` watchdog flake (WINDOWS.md entry 12)

## Self-Check: PASSED
