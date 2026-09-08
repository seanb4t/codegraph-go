---
phase: 01-engine-seam-wire-protocol-secure-transport
plan: 07
subsystem: infra
tags: [protobuf, buf, codegen-drift, taskfile, ci, go-yaml]

# Dependency graph
requires:
  - phase: 01-engine-seam-wire-protocol-secure-transport (plan 01-01)
    provides: "task proto:gen, buf.yaml/buf.gen.yaml, go.tool-proto.mod — the buf-driven codegen pipeline this plan adds a drift guard around"
  - phase: 01-engine-seam-wire-protocol-secure-transport (plan 01-06)
    provides: "internal/schema/graph.proto's committed commit_sha field 8 — part of the surface proto:drift now guards"
provides:
  - "Taskfile.yml task proto:drift: regenerates both proto surfaces into a temporary tree, byte-compares against committed output, reports a positive compared-count, fails on a zero/short count or any byte difference (header included)"
  - "internal/upgrade/proto_task_test.go: TestProtoTasksExist, TestProtoDriftGuardReportsAComparedCount, TestProtoDriftGeneratesIntoATemporaryTree, TestProtoDriftTaskIsInvokedByCI"
  - ".github/workflows/ci.yml step 'Proto codegen drift guard (BLD-04)' in the test job, invoking task proto:drift"
  - "Recorded RED evidence: the guard demonstrated failing independently against a stale internal/schema/graph.pb.go, a stale internal/uiproto/uiv1/ui.pb.go, and a header-only staleness, each restored via an exact reverse patch"
affects: []

# Actuals (#2632)
actuals:
  tokens: 5449
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Codegen drift guard regenerates into a mktemp -d temporary tree and byte-compares, never regenerating in place over committed generated files — protects an uncommitted developer edit from being clobbered before the guard reports on it"
    - "buf generate invoked with an explicit absolute-path input (never bare \".\") to keep its module-root resolution deterministic regardless of invocation context — worked around a buf 1.72.0 defect where its upward workspace search misidentifies the controlling workspace inside a git linked worktree nested under its own main checkout"

key-files:
  created:
    - internal/upgrade/proto_task_test.go
  modified:
    - Taskfile.yml
    - .github/workflows/ci.yml
    - .planning/phases/01-engine-seam-wire-protocol-secure-transport/deferred-items.md

key-decisions:
  - "buf generate is invoked with an explicit absolute-path input (repo_root=$(pwd); buf generate -o \"${scratch}/gen\" \"${repo_root}\") rather than the implicit \".\" both proto:gen and the corpora:drift precedent use — discovered live that buf 1.72.0's upward workspace-root search misidentifies the controlling workspace when run from inside a git linked worktree nested under its own main checkout (its gitlink file is not a real .git directory, so buf's git-awareness appears to keep walking up past it), duplicating proto declarations from sibling trees (\"Node declared multiple times\"). The explicit path sidesteps the whole search. Logged as a deferred item rather than fixed in proto:gen, since that task is 01-01's deliverable and out of this plan's files_modified, and the defect is CI-inert (CI checks out a normal, non-nested tree)."
  - "Enumeration uses git ls-files against the three concrete glob patterns (internal/schema/*.pb.go, internal/uiproto/uiv1/*.pb.go, internal/uiproto/uiv1/uiv1connect/*.connect.go) rather than a raw filesystem find — 'committed generated files' is the literal contract the guard is checking, so scoping the enumeration to tracked files is what the wording demands, and it is also what let Task 1's zero-count demonstration cleanly retarget the enumeration at a nonexistent path pattern without touching the filesystem."
  - "The Go-level shape guard (proto_task_test.go) decodes Taskfile.yml's tasks: map generically as map[string]yaml.Node rather than a strictly-typed per-task struct, deferring the strict decode to the one named task under test. Several unrelated tasks elsewhere in Taskfile.yml use go-task's map-form cmds: entries (cmd:/for:/vars: sub-task calls); a strictly-typed []string Cmds field failed to decode the WHOLE document over those tasks' shape even though this guard never reads their cmds:."

patterns-established:
  - "Any future Taskfile.yml shape-test parser reading the tasks: map generically (map[string]yaml.Node) before strictly decoding just the one task under test, to stay resilient to unrelated tasks' cmds: shape (string vs. map-form sub-task call)."

requirements-completed: [BLD-04]

coverage:
  - id: D1
    description: "proto:drift regenerates both proto surfaces into a temporary tree and byte-compares against the committed files, reporting a positive count of files compared and failing on a zero/short count"
    requirement: "BLD-04"
    verification:
      - kind: unit
        ref: "internal/upgrade/proto_task_test.go#TestProtoTasksExist"
        status: pass
      - kind: unit
        ref: "internal/upgrade/proto_task_test.go#TestProtoDriftGuardReportsAComparedCount"
        status: pass
      - kind: unit
        ref: "internal/upgrade/proto_task_test.go#TestProtoDriftGeneratesIntoATemporaryTree"
        status: pass
      - kind: other
        ref: "task proto:drift (clean tree): exit 0, 'compared 3 generated files', 'all 3 generated files byte-identical...'"
        status: pass
    human_judgment: false
  - id: D2
    description: "The drift guard has been watched fail independently against a stale internal/schema/graph.pb.go, a stale internal/uiproto/uiv1/ui.pb.go, and a header-only staleness, with every mutation restored via an exact reverse patch and the tree confirmed clean afterward"
    requirement: "BLD-04"
    verification:
      - kind: manual_procedural
        ref: "Three deliberate-RED exercises against task proto:drift, verbatim output recorded below (see '## Deliberate-RED Observations (Task 2)')"
        status: pass
    human_judgment: false
  - id: D3
    description: "The guard runs in CI as a Taskfile-defined job body (task proto:drift), and TestWorkflowRunBodiesInvokeTask's single-definition property still holds"
    requirement: "BLD-04"
    verification:
      - kind: unit
        ref: "internal/upgrade/proto_task_test.go#TestProtoDriftTaskIsInvokedByCI"
        status: pass
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestWorkflowRunBodiesInvokeTask"
        status: pass
    human_judgment: false

duration: ~50min (wall-clock, single-agent session)
completed: 2026-08-23
status: complete
---

# Phase 1 Plan 7: Codegen Drift Guard Over Both Protobuf Surfaces Summary

**`task proto:drift` regenerates `internal/schema/graph.proto` and `internal/uiproto/uiv1/ui.proto` into a temporary tree with the pinned buf toolchain, byte-compares (header included) against the three committed generated files, reports a positive compared-count, and has been watched fail independently against each surface and against a header-only staleness.**

## Performance

- **Duration:** ~50min wall-clock
- **Started:** 2026-08-23T11:15:00-04:00 (approx.)
- **Completed:** 2026-08-23T11:34:44-04:00
- **Tasks:** 2
- **Files modified:** 3 (Taskfile.yml, .github/workflows/ci.yml, internal/upgrade/proto_task_test.go created) + 1 doc (deferred-items.md)

## Accomplishments

- `Taskfile.yml`'s new `proto:drift` task: regenerates both proto surfaces into a `mktemp -d` temporary tree (never in place), byte-compares each of the three committed generated files (`internal/schema/graph.pb.go`, `internal/uiproto/uiv1/ui.pb.go`, `internal/uiproto/uiv1/uiv1connect/ui.connect.go`) against its fresh counterpart, echoes `compared N generated files` before comparing anything, and fails with a named error below a floor of 3 or on any byte difference — including a header-only one
- `internal/upgrade/proto_task_test.go`: four Go-level shape guards (`TestProtoTasksExist`, `TestProtoDriftGuardReportsAComparedCount`, `TestProtoDriftGeneratesIntoATemporaryTree`, `TestProtoDriftTaskIsInvokedByCI`) — 4 `--- PASS` lines, matching the plan's floor derivation
- `task proto:drift` wired into `ci.yml`'s `test` job as a bare `task proto:drift` step, keeping `TestWorkflowRunBodiesInvokeTask`'s single-definition property intact
- Discovered and worked around a live buf 1.72.0 defect (misidentified controlling workspace inside a nested git linked worktree) that blocked both `proto:gen` and the drafted `proto:drift` from running at all in this execution's sandbox — fixed in `proto:drift` via an explicit absolute-path input to `buf generate`, logged as a deferred item for `proto:gen` (out of this plan's `files_modified`)
- Three deliberate-RED observations recorded: the guard fails, names the specific drifted file, and reports the compared-count, for a stale `graph.pb.go`, a stale `ui.pb.go`, and a header-only staleness — each restored via an exact `git apply -R` reverse patch, confirmed clean before and after
- A fourth demonstration (Task 1 acceptance criterion): temporarily pointing the enumeration at a pattern matching zero files produces `compared 0 generated files` and a named, non-zero-exit failure rather than a silent pass

## Task Commits

1. **Task 1: `proto:drift` — regenerate into a temp tree, byte-compare, report the count** — `40cb8e9` (feat)
   - Related discovery, logged separately: `ee322cb` (docs: buf worktree-detection quirk)
2. **Task 2: Watch the guard fail against a deliberately stale file, once per surface** — no commit (Form A3 observation task; every mutation was made, observed, and reverted via an exact reverse patch, leaving `git status --porcelain` clean — there is no persisted code change to commit for this task, only the recorded evidence below and in this SUMMARY)

## Files Created/Modified

- `Taskfile.yml` — new `proto:drift` task (regenerate-into-temp-tree, count-then-compare, header-inclusive byte comparison)
- `internal/upgrade/proto_task_test.go` — the four companion shape guards
- `.github/workflows/ci.yml` — one new step in the `test` job: `task proto:drift`
- `.planning/phases/01-engine-seam-wire-protocol-secure-transport/deferred-items.md` — logged the buf/nested-worktree discovery (out of scope for `proto:gen`, CI-inert)

## Deliberate-RED Observations (Task 2)

All three mutations were applied directly to the tracked file, confirmed applied via `git diff --stat`, run against `task proto:drift`, confirmed the local edit survived the guard run untouched, then restored via `git diff -- <path> > <patch>` captured immediately after the edit and `git apply -R <patch>` to undo it exactly — never `git checkout --`.

### Exercise 1 — stale `internal/schema/graph.pb.go`

**Edit (verbatim, one line):**
```diff
-	Id            string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
+	Id            string                 `protobuf:"bytes,1,opt,name=identifier,proto3" json:"id,omitempty"`
```

**Applied confirmation:** `git diff --stat internal/schema/` → `internal/schema/graph.pb.go | 2 +-` (1 insertion, 1 deletion)

**`task proto:drift` output (verbatim):**
```
proto:drift: compared 3 generated files
::error::proto:drift: internal/schema/graph.pb.go differs from the pinned toolchain's regeneration (generated header included — a header-only change IS drift)
task: Failed to run task "proto:drift": exit status 1
EXIT=201
```

Names the drifted file (`internal/schema/graph.pb.go`) and prints the compared-count (3) before failing. The edit survived the run untouched (`git diff --stat` unchanged after).

**Restore:** `git apply -R` against the captured patch. `git status --porcelain internal/schema/` → empty.

### Exercise 2 — stale `internal/uiproto/uiv1/ui.pb.go`

**Edit (verbatim, one line):**
```diff
-	Initialized bool `protobuf:"varint,1,opt,name=initialized,proto3" json:"initialized,omitempty"`
+	Initialized bool `protobuf:"varint,1,opt,name=is_initialized,proto3" json:"initialized,omitempty"`
```

**Applied confirmation:** `git diff --stat internal/uiproto/` → `internal/uiproto/uiv1/ui.pb.go | 2 +-` (1 insertion, 1 deletion)

**`task proto:drift` output (verbatim):**
```
proto:drift: compared 3 generated files
::error::proto:drift: internal/uiproto/uiv1/ui.pb.go differs from the pinned toolchain's regeneration (generated header included — a header-only change IS drift)
task: Failed to run task "proto:drift": exit status 1
EXIT=201
```

Names the drifted file (`internal/uiproto/uiv1/ui.pb.go`, the NEW surface — proving the guard does not silently skip it) and prints the compared-count (3). The edit survived the run untouched.

**Restore:** `git apply -R` against the captured patch. `git status --porcelain internal/uiproto/` → empty.

### Exercise 3 — header-only staleness (`internal/schema/graph.pb.go`)

**Edit (verbatim, one line — the generator version stamp only):**
```diff
-// 	protoc        (unknown)
+// 	protoc        v9.99.9
```

**Applied confirmation:** `git diff --stat internal/schema/` → `internal/schema/graph.pb.go | 2 +-` (1 insertion, 1 deletion)

**`task proto:drift` output (verbatim):**
```
proto:drift: compared 3 generated files
::error::proto:drift: internal/schema/graph.pb.go differs from the pinned toolchain's regeneration (generated header included — a header-only change IS drift)
task: Failed to run task "proto:drift": exit status 1
EXIT=201
```

Proves the guard does NOT normalise the generated header away: a change confined to exactly the `protoc` version-stamp line still fails, exactly as 01-01's ruling requires (a toolchain downgrade must surface, not pass silently). The edit survived the run untouched.

**Restore:** `git apply -R` against the captured patch. `git status --porcelain internal/schema/ internal/uiproto/` → empty (whole scope, both surfaces, confirmed clean together).

### Final clean run

```
proto:drift: compared 3 generated files
proto:drift: all 3 generated files byte-identical to the pinned toolchain's regeneration (temporary tree only — source tree untouched)
EXIT=0
```

Compared-count (3) matches the count recorded in every RED exercise above and in Task 1.

### Stop-condition check (A3 requirement 2)

None of the three exercises above exited 0 against its confirmed-applied stale file — the stop condition ("a `proto:drift` run that exits 0 against a confirmed-applied stale file means the guard cannot see that surface") never triggered. Had it triggered, the guard would have been fixed and the exercise re-run rather than recorded as an inconclusive pass.

### What this leg CAN and CANNOT catch (A3 requirement 4)

**CAN catch:** a generated surface this task mutated and failed to restore (the post-exercise `git status --porcelain` checks would show it), and a `proto:drift` that stopped enumerating mid-session (the compared-count would drop below 3) — both failures belonging to this task's own execution.

**CANNOT catch:** this task doing nothing at all and simply asserting the guard works without running it. That is exactly why the three exercises above are recorded individually, per surface, with verbatim output, rather than as a single unsupported claim.

### Task 1 acceptance-criterion demonstration: zero-count failure

Separately from Task 2's three surface exercises, Task 1's own acceptance criteria requires demonstrating the guard fails loud (not silently) when the enumeration finds zero files. Temporarily repointed `git ls-files`'s glob patterns in `Taskfile.yml` at a nonexistent path pattern (`tmp-empty-proto-dir-260823/*.pb.go`), confirmed applied via `git diff --stat Taskfile.yml`, ran:

```
proto:drift: compared 0 generated files
::error::proto:drift: enumerated only 0 committed generated files under internal/schema/ and internal/uiproto/uiv1/ (expected at least 3: graph.pb.go, ui.pb.go, ui.connect.go) — a broken enumeration must fail loud, never read as a clean pass
task: Failed to run task "proto:drift": exit status 1
EXIT=201
```

Restored via the same captured-diff/`git apply -R` procedure; `git status --porcelain Taskfile.yml` confirmed empty afterward, and a follow-up clean `task proto:drift` run reconfirmed `compared 3 generated files` / exit 0.

## Decisions Made

- **Explicit absolute-path input to `buf generate`**, not the implicit `"."` both `proto:gen` and `corpora:drift`'s precedent use. See `key-decisions` in frontmatter for the full rationale — this was forced by a live buf 1.72.0 defect discovered while first running `task proto:gen` in this execution's own worktree (see Deviations below).
- **`git ls-files` enumeration**, not a raw filesystem walk. "Committed generated files" is the guard's literal scope; scoping to tracked files also made the zero-count acceptance-criterion demonstration a clean glob-pattern edit rather than a filesystem mutation.
- **`[]yaml.Node` for `Cmds`** in the Go-level shape guard, not `[]string`, to avoid a whole-document decode failure over unrelated tasks' map-form `cmds:` entries elsewhere in `Taskfile.yml`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] buf 1.72.0 misidentifies the controlling workspace inside a nested git worktree, blocking both `proto:gen` and the drafted `proto:drift`**
- **Found during:** Task 1, first invocation of `task proto:gen` to confirm the reuse baseline before writing `proto:drift`
- **Issue:** `buf build`/`buf generate` invoked with the implicit `"."` input failed with `` `Node` declared multiple times `` / `` `UIService` declared multiple times ``. `buf ls-files` showed the SAME two `.proto` files enumerated from both this worktree's own tree and the outer main checkout's tree (and, transiently, a sibling agent's worktree) at identical relative paths — buf's own `--debug` log showed its workspace-root search terminating at the outer checkout rather than this worktree's own directory, most likely because this worktree's `.git` is a gitlink FILE rather than a directory and buf's git-awareness keeps walking upward past it.
- **Fix:** `proto:drift`'s own `buf generate` invocation passes an explicit absolute path (`repo_root=$(pwd)`) instead of relying on the implicit `"."`, which sidesteps the upward search entirely — verified both via a standalone `buf build <abspath>` returning cleanly and via `task proto:drift` reporting `compared 3 generated files` with zero drift.
- **Files modified:** `Taskfile.yml` (`proto:drift`'s own `cmds:`, not `proto:gen`)
- **Verification:** `task proto:drift` runs cleanly from this worktree; the three deliberate-RED exercises above confirm the guard still correctly detects real drift with this input form
- **Committed in:** `40cb8e9` (Task 1 commit); the discovery itself logged separately in `ee322cb`
- **Not fixed in `proto:gen`:** `proto:gen`'s task body is 01-01's deliverable, outside this plan's `files_modified`, and the plan's artifact contract explicitly forbids re-authoring it. `proto:gen` therefore still uses the implicit `"."` input and would exhibit the same failure if ever run from inside a similarly-nested worktree — this is understood to be CI-inert, since this repository's CI runners (`namespace-profile-linux-amd64-4x8`) check out a normal, non-nested tree. Logged in `deferred-items.md` for a future plan touching `proto:gen` to consider.

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary to make `proto:drift` runnable and verifiable at all in this execution's sandbox; does not touch `proto:gen`'s task body, and the fix generalizes correctly to CI (an explicit absolute path is not worktree-specific — it simply happens to also fix a worktree-specific defect).

## Issues Encountered

None beyond the buf/worktree discovery documented above, which was diagnosed and worked around within Task 1.

## User Setup Required

None — no external service configuration required.

## Known Stubs

None. `proto:drift` is a real, working Taskfile task exercised against the actual pinned toolchain and the actual committed generated files, not a stub or placeholder.

## Next Phase Readiness

- Both protobuf surfaces (`internal/schema/graph.proto`, `internal/uiproto/uiv1/ui.proto`) now carry a codegen drift guard that runs in CI, reports what it compared, and has been demonstrated to fail correctly against real staleness on both surfaces plus a header-only case.
- Any later plan that touches `proto:gen`'s own task body should consider applying the same explicit-absolute-path fix to `buf generate`, both for consistency with `proto:drift` and to protect contributors who develop from inside a nested worktree layout (see `deferred-items.md`).
- No blockers for downstream plans in this wave.

## Self-Check: PASSED

All claimed files confirmed present on disk (`Taskfile.yml`, `.github/workflows/ci.yml`,
`internal/upgrade/proto_task_test.go`, `.planning/phases/01-engine-seam-wire-protocol-secure-transport/deferred-items.md`);
both claimed commit hashes (`40cb8e9`, `ee322cb`) confirmed present in
`git log --oneline --all`. No missing items.

---
*Phase: 01-engine-seam-wire-protocol-secure-transport*
*Completed: 2026-08-23*
