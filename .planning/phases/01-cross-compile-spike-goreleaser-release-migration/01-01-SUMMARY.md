---
phase: 01-cross-compile-spike-goreleaser-release-migration
plan: 01
subsystem: infra
tags: [goreleaser, zig, cgo, cross-compilation, github-actions, ci]

# Dependency graph
requires:
  - phase: v1.0 Phase 9/10 — release-please + GoReleaser, local build tooling
    provides: ".goreleaser.yaml with a working 4-target build matrix, Taskfile.yml as the single CI-body definition, darwin-toolchain-canary.yml as the canary-workflow shape precedent"
provides:
  - "codegraph-linux-amd64 .goreleaser.yaml build id now cross-compiles via zig cc/zig c++ (matching the existing codegraph-linux-arm64 shape)"
  - "Taskfile.yml release:dry-run target — proves the full `goreleaser release --snapshot --skip=publish,sign` composition (all four targets, one invocation) on a native darwin host"
  - "Taskfile.yml check:linux-cross-export / check:linux-cross-exec targets — export the two zig-crossed linux binaries and assert they index a real tree to a non-zero graph on real Linux hardware"
  - ".github/workflows/linux-cross-canary.yml — permanent, dispatchable 3-job canary (cross-build on macOS, exec-linux-amd64, exec-linux-arm64) carrying the D-04 FAIL-bar variation list"
  - "internal/upgrade/goreleaser_shape_test.go — TestLinuxBuildIdsCrossCompileViaZig / TestParseGoreleaserBuildEnv_MissingBuildIDIsError, both directions of mutation demonstrated RED"
affects: ["01-02 (activates sboms:/binary_signs:)", "01-03 (collapses release.yml's build matrix into the single-runner job this canary rehearses)", "01-05", "01-06"]

# Actuals (#2632)
actuals:
  tokens: 8400
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "zig cc/zig c++ CC/CXX override on a .goreleaser.yaml linux build id, mirroring the pre-existing codegraph-linux-arm64 shape"
    - "REL05-EVIDENCE machine-parseable log line as the structural pass/fail signal for a real-hardware execution proof, not a green exit code"
    - "dist/artifacts.json filtered on extra.Format==\"binary\" to select the archives: pipe's release asset, distinct from the raw build-output entry goreleaser release also records"

key-files:
  created:
    - internal/upgrade/goreleaser_shape_test.go
    - .github/workflows/linux-cross-canary.yml
  modified:
    - .goreleaser.yaml
    - .github/workflows/release.yml
    - Taskfile.yml
    - CONTRIBUTING.md
    - .github/actionlint.yaml

key-decisions:
  - "ZIG_LOCAL_CACHE_DIR/ZIG_GLOBAL_CACHE_DIR exported inside the release:dry-run Taskfile target itself (not in CI workflow env), so both local runs and the canary's `task release:dry-run` step get the fix for free without a second place to keep in sync."
  - "dist/artifacts.json Binary entries are filtered on extra.Format==\"binary\" everywhere this plan reads them (release:dry-run's count assertion, check:linux-cross-export's binary lookup) — goreleaser release records type==\"Binary\" twice per platform (raw build output + archives: pipe's renamed release asset) and only the second is the actual shipped asset."

patterns-established:
  - "Any future .goreleaser.yaml env-parsing Go test decodes with yaml.Unmarshal into typed structs (parseGoreleaserBuildEnv), never a hand-written line scanner — binds plans 01-02/01-03 per the plan's <parser_strategy>."

requirements-completed: []  # REL-05 NOT YET satisfied — Task 3 (the human-verify checkpoint that dispatches the canary and records PASS/FAIL) has not run. See Next Phase Readiness.

coverage:
  - id: D1
    description: "codegraph-linux-amd64 build id cross-compiles via zig cc/zig c++ (x86_64-linux-gnu), matching codegraph-linux-arm64; darwin build ids carry no CC/CXX override"
    requirement: "REL-05"
    verification:
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestLinuxBuildIdsCrossCompileViaZig"
        status: pass
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestParseGoreleaserBuildEnv_MissingBuildIDIsError"
        status: pass
    human_judgment: false
  - id: D2
    description: "task release:dry-run proves one goreleaser release --snapshot --skip=publish,sign invocation produces all four release binaries (2 ELF, 2 Mach-O) with a file(1)-verified case statement, not a bare exit code"
    requirement: "REL-05"
    verification:
      - kind: other
        ref: "task release:dry-run (local darwin/arm64 run, exit 0, printed four correctly-typed binaries)"
        status: pass
    human_judgment: false
  - id: D3
    description: "release:dry-run halts on an actionable, named precondition message (not a mid-pipe failure) when syft is absent from PATH, and resumes cleanly once restored"
    requirement: "REL-05"
    verification:
      - kind: manual_procedural
        ref: "syft symlink renamed off /opt/homebrew/bin, task release:dry-run halted with the precondition msg text, restored, re-ran green"
        status: pass
    human_judgment: false
  - id: D4
    description: "permanent, dispatchable linux-cross-canary workflow exists with the D-04 FAIL-bar variation list (V1-V5) written before any dispatch, contents=read-only permissions, all third-party Actions SHA-pinned, no ${{ }} in any run: body"
    requirement: "REL-05"
    verification:
      - kind: other
        ref: "task lint:actions (actionlint, exit 0)"
        status: pass
      - kind: other
        ref: "rg -v '^\\s*#' .github/workflows/linux-cross-canary.yml | rg -c 'uses:.*@[0-9a-f]{40}' == 13 (all third-party uses: lines)"
        status: pass
    human_judgment: false
  - id: D5
    description: "check:linux-cross-exec converts a linked object into REL-05's real evidence: ELF-machine-vs-host-arch assertion, real `codegraph init` against the checked-out repo, codegraph status --json fileCount/nodeCount both asserted non-zero, one REL05-EVIDENCE line emitted"
    requirement: "REL-05"
    verification: []
    human_judgment: true
    rationale: "check:linux-cross-exec's precondition requires GOHOSTOS=linux and was only syntax-checked (bash -n) plus precondition-halt-verified on this darwin execution host — it has never actually run end-to-end. The REAL evidence (both linux legs executing on namespace-profile-linux-amd64-4x8 / namespace-profile-linux-arm64-4x8 and reporting non-zero REL05-EVIDENCE lines) can only come from Task 3's canary dispatch, which is a blocking human-verify checkpoint this plan halted at — not auto-approvable per the orchestrator's explicit instruction."

duration: 95min
completed: 2026-08-08
status: halted
---

# Phase 1 Plan 1: Cross-Compile Spike & `goreleaser release` Migration Summary

**Both linux .goreleaser.yaml build ids now zig-cross from a native macOS host in one `goreleaser release` invocation, proven locally with file(1)-verified ELF/Mach-O output; a permanent linux-cross-canary workflow exists to prove the harder half — real execution on real Linux hardware — but has not yet been dispatched.**

## Performance

- **Duration:** ~95 min
- **Started:** 2026-08-08 (approx, continuation from prior session's context load)
- **Completed:** 2026-08-08T18:52:41Z
- **Tasks:** 2 of 3 completed (Task 3 is a blocking human-verify checkpoint, not yet run)
- **Files modified:** 7 (2 created, 5 modified)

## Accomplishments

- `codegraph-linux-amd64`'s `.goreleaser.yaml` build entry now carries `CC=zig cc -target x86_64-linux-gnu` / `CXX=zig c++ -target x86_64-linux-gnu`, mirroring the pre-existing `codegraph-linux-arm64` shape — both linux legs now cross-compile via zig, which is the structural precondition for a single macOS runner producing all four release targets (D-01/D-02).
- `task release:dry-run` — a new Taskfile target — runs `goreleaser release --snapshot --skip=publish,sign --clean` on a native darwin host and **verified locally, end-to-end, right now**: all four binaries produced in one invocation, each independently confirmed via `file -b` to be the correct object type (2 ELF x86-64/aarch64, 2 Mach-O x86_64/arm64) — not merely a green exit code.
- `task check:linux-cross-export` and `task check:linux-cross-exec` — the two Taskfile targets the permanent canary drives — exist, are registered (`task --list`), lint clean, and (for `-export`) were run successfully against a real `release:dry-run` output locally. `check:linux-cross-exec` is gated on `GOHOSTOS=linux` and was confirmed to halt correctly on this darwin host with its named precondition message; its actual pass/fail behavior on real Linux hardware is what Task 3's checkpoint exists to observe.
- `.github/workflows/linux-cross-canary.yml` — a new permanent, `workflow_dispatch`-triggerable canary (3 jobs: `cross-build` on `namespace-profile-macos-6x14-tahoe`, `exec-linux-amd64` on `namespace-profile-linux-amd64-4x8`, `exec-linux-arm64` on `namespace-profile-linux-arm64-4x8`) carrying the D-04 FAIL-bar variation list (V1-V5) written into the header before any dispatch.
- `internal/upgrade/goreleaser_shape_test.go` — new shape tests holding the zig-cross invariant, decoding `.goreleaser.yaml` with `yaml.Unmarshal` into typed structs (no hand-written scanner, per the plan's `<parser_strategy>`). Both mutation-RED demonstrations recorded below.

## Task Commits

1. **Task 1: End-to-end "one goreleaser invocation cross-compiles all four targets from macOS"** — `e25cbe4` (feat)
2. **Task 2: Permanent linux-cross canary — build on macOS, EXECUTE on real Linux, index a real tree** — `5609765` (feat)

Task 3 (checkpoint:human-verify, `gate="blocking"`) has **not** run — this plan halts here pending maintainer dispatch of the canary. No plan-metadata commit exists yet; that lands after Task 3 resolves.

## Files Created/Modified

- `.goreleaser.yaml` — `codegraph-linux-amd64` gains `CC=zig cc -target x86_64-linux-gnu` / `CXX=zig c++ -target x86_64-linux-gnu`; comment above the linux entries updated
- `.github/workflows/release.yml` — linux/amd64 matrix leg's `needs_zig` flipped `false` → `true` (transitional; plan 01-03 collapses this matrix)
- `Taskfile.yml` — new `release:dry-run`, `check:linux-cross-export`, `check:linux-cross-exec` targets
- `internal/upgrade/goreleaser_shape_test.go` — new file: `parseGoreleaserBuildEnv`/`mustGoreleaserBuildEnv`, `TestLinuxBuildIdsCrossCompileViaZig`, `TestParseGoreleaserBuildEnv_MissingBuildIDIsError`
- `.github/workflows/linux-cross-canary.yml` — new permanent canary workflow
- `.github/actionlint.yaml` — registered `namespace-profile-linux-arm64-4x8` as a known self-hosted-runner label
- `CONTRIBUTING.md` — references `task release:dry-run`, `task check:linux-cross-export`, `task check:linux-cross-exec`, and the new canary

## Decisions Made

- `ZIG_LOCAL_CACHE_DIR`/`ZIG_GLOBAL_CACHE_DIR` are exported inside `release:dry-run` itself, not in the CI workflow's `env:`. Since the canary's `cross-build` job also calls `task release:dry-run`, this is one fix location instead of two, and it survives a future contributor running the target locally without CI-specific setup.
- `dist/artifacts.json` `Binary` entries are filtered on `extra.Format=="binary"` everywhere this plan reads them. `goreleaser release` records `type=="Binary"` **twice** per platform — once for the raw `go build` output (no `extra.Format`) and once for the `archives:` pipe's renamed/copied release asset (`extra.Format=="binary"`) — and only the second is the actual shipped `codegraph_<tag>_<goos>_<goarch>` asset `internal/upgrade` downloads.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] zig cc/zig c++ AccessDenied on Go's read-only module cache**
- **Found during:** Task 1 — first real `task release:dry-run` invocation
- **Issue:** `zig cc`/`zig c++` default their local cache directory to a path next to the C source file being compiled unless overridden. This repo's CGo tree-sitter dependencies (e.g. `tree-sitter-c`) compile directly from Go's module cache (`GOMODCACHE`), which is read-only by design. The first real cross-build failed with `error: unable to open local cache directory '.../tree-sitter-c@v0.24.2/.zig-cache': AccessDenied`.
- **Fix:** Export `ZIG_LOCAL_CACHE_DIR`/`ZIG_GLOBAL_CACHE_DIR` pointing at a `mktemp -d` scratch directory inside `release:dry-run`, cleaned up by the same `trap` as the GoReleaser tool binary.
- **Files modified:** `Taskfile.yml`
- **Verification:** Confirmed the exact failure via a standalone `CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC="zig cc -target x86_64-linux-gnu" go build` repro without the env vars, then confirmed it succeeds (both amd64 and arm64) with them set. `task release:dry-run` subsequently ran clean end-to-end.
- **Committed in:** `e25cbe4` (Task 1 commit)

**2. [Rule 1 - Bug] Binary-count assertion counted the wrong dist/artifacts.json entries**
- **Found during:** Task 1 — after the zig-cache fix, the first full `task release:dry-run` run succeeded but the binary-count check failed with "8 Binary entries, want exactly 4"
- **Issue:** The initial `release:dry-run` script filtered `dist/artifacts.json` on `type=="Binary"` alone. `goreleaser release` records that type twice per platform: once for the raw `go build` output (named literally `codegraph`) and once for the `archives:` pipe's renamed release asset — 4 platforms × 2 = 8 entries, not 4.
- **Fix:** Filter on `.extra.Format=="binary"` in addition to `.type=="Binary"`, which selects only the actual release asset. Applied the same filter in `check:linux-cross-export`'s per-pair binary lookup for consistency.
- **Files modified:** `Taskfile.yml`
- **Verification:** `task release:dry-run` re-run clean (`BINARY_COUNT=4`), all four `file -b` assertions passed.
- **Committed in:** `e25cbe4` (Task 1 commit — this fix landed before Task 1 was committed, so it is not a separate commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug) — both discovered and resolved during Task 1's first real dry-run execution, both folded into Task 1's single commit.
**Impact on plan:** Both fixes were necessary for `task release:dry-run` to be a real, trustworthy target rather than one that appears to work on a green exit code alone — directly in the spirit of REL-05's "a green exit code is explicitly not the check" prohibition. No scope creep.

## Mutation-RED Demonstrations (Task 1 acceptance criteria)

Both recorded live during execution, both reverted before continuing:

**1. Deleting `CC=` from `codegraph-linux-amd64`:**
```
=== RUN   TestLinuxBuildIdsCrossCompileViaZig
    goreleaser_shape_test.go:92: codegraph-linux-amd64: CC = "", want a zig cc override targeting x86_64-linux-gnu
--- FAIL: TestLinuxBuildIdsCrossCompileViaZig (0.00s)
```
Restored → `PASS`.

**2. Adding `CC=zig cc -target aarch64-macos` to `codegraph-darwin-arm64`:**
```
=== RUN   TestLinuxBuildIdsCrossCompileViaZig
    goreleaser_shape_test.go:116: codegraph-darwin-arm64: has a CC="zig cc -target aarch64-macos" override, want none — darwin must never cross-link via zig (libresolv/DNS-resolver risk)
--- FAIL: TestLinuxBuildIdsCrossCompileViaZig (0.00s)
```
Restored → `PASS`.

## Precondition-Halt Demonstration (Task 1 acceptance criteria)

`syft`'s symlink was renamed off `/opt/homebrew/bin` and `task release:dry-run` was re-run:
```
task: syft not found. --skip=publish,sign only skips the SIGN pipe — it does NOT skip the sboms: pipe. The moment plan 01-02 activates .goreleaser.yaml's sboms: block, this target shells out to syft, and this precondition turns that into a named, actionable halt instead of a confusing mid-pipe failure (review HIGH-4). CI installs it via anchore/sbom-action/download-syft; locally: brew install syft.
task: Failed to run task "release:dry-run": task: precondition not met
```
`syft` restored, `task release:dry-run` re-ran green.

## Local `release:dry-run` Toolchain Environment (record, not REL-05 evidence)

Per orchestrator note: local zig is **0.16.0**; the canary and `release.yml` pin CI zig to **0.15.1** via `mlugg/setup-zig`. A green local dry-run therefore proves the `.goreleaser.yaml`/Taskfile config shape and the linker-level cross-compile, but is **explicitly not** REL-05 evidence — only a real canary run URL on the pinned CI toolchain is. This divergence is recorded here verbatim, not glossed over: `go 1.26.5 darwin/arm64`, `zig 0.16.0`, `syft 1.50.0`, `cosign v3.1.3`, all local.

## Issues Encountered

None beyond the two deviations documented above.

## User Setup Required

None - no external service configuration required for Tasks 1-2. Task 3 requires a maintainer (or an authorized agent with `gh` access) to dispatch the canary — see Next Phase Readiness.

## Next Phase Readiness

**This plan is HALTED, not complete.** Task 3 — `checkpoint:human-verify`, `gate="blocking"` — is the task carrying REL-05's actual evidence bar (both linux legs executing on real Namespace Linux hardware and reporting non-zero `REL05-EVIDENCE` counts) and was explicitly withheld from auto-approval by the orchestrator's dispatch instructions: *"Auto-mode is NOT authorized for it… Do NOT dispatch the canary yourself, do NOT self-approve, and do NOT write a REL-05 PASS into the SUMMARY. The maintainer decides."*

**What is proven so far:** the `.goreleaser.yaml`/Taskfile config shape is correct and the zig-cross **link** succeeds on a real macOS host for both linux legs, verified locally end-to-end (`task release:dry-run` green, four correctly-typed objects). What is **not yet proven**: whether the resulting linux/amd64 and linux/arm64 binaries actually **run** and index a real tree on real, non-emulated Linux hardware — the harder half of REL-05, and the one a wrong toolchain can fail silently.

**Blocked on:** maintainer dispatch of `.github/workflows/linux-cross-canary.yml` (`gh workflow run linux-cross-canary.yml --ref <this branch>`), then recording the run URL and both `REL05-EVIDENCE` lines. If the run passes, plans 01-02 onward are unblocked. If it fails after exhausting the V1-V5 variation list, REL-05 is FAIL and the phase escalates to the GoReleaser Pro fallback per the plan header and PROJECT.md.

---
*Phase: 01-cross-compile-spike-goreleaser-release-migration*
*Completed: 2026-08-08 (halted at Task 3 checkpoint)*
