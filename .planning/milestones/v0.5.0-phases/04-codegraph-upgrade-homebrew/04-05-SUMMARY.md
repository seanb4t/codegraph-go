---
phase: 04-codegraph-upgrade-homebrew
plan: 05
subsystem: release-verification
tags: [cosign, taskfile, self-upgrade, drift-guard, todo-fold]

# Dependency graph
requires:
  - phase: 04-codegraph-upgrade-homebrew (plan 02)
    provides: sentinel-free Taskfile.yml (OK=0 baseline 26, man-page freshness assertions) this plan's shape test reads
  - phase: 04-codegraph-upgrade-homebrew (plan 04)
    provides: amended README.md / docs/RELEASE.md this plan's shape test reads
provides:
  - verify:self-upgrade with a cosign verify-blob gate strictly before chmod +x
  - TestCosignIdentityPolicyBoundaryParityWithCompiledPattern / _ZeroLiteralsIsError, proving boundary-case parity across all seven identity-regexp restatements in five files
  - folded todo 2026-08-09-verify-self-upgrade-download-then-execute-has-no-signature-check.md
affects: [04-06]

actuals:
  tokens: 46000
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Two-temp-dirs discipline reused a third time: verify:self-upgrade now downloads its raw binary and cosign bundle into DL_DIR/SIG_DIR the same way verify:release-assets and verify:notarized-suite already do, with the cleanup trap installed after the first mktemp so a second-mktemp failure under set -e cannot leak the first directory."
    - "Extractor-plus-corpus drift guard: identity-regexp literals restated by hand across five files are extracted with source location (File/Line/Value) and checked for selected boundary-case behavioural parity against the compiled Go pattern, rather than string-compared — the same shape TestCheckCrossMatchesGoreleaserTargets and TestRequiredCheckNamesPreserved already use for other cross-file restatements in this package."

key-files:
  created: []
  modified:
    - Taskfile.yml
    - internal/upgrade/taskfile_shape_test.go
    - .planning/todos/completed/2026-08-09-verify-self-upgrade-download-then-execute-has-no-signature-check.md

key-decisions:
  - "Mutation 3 (Task 3) relocates Task 1's identity literal into verify:gatekeeper as an inert comment rather than deleting it, so the total-literal floor (check 1) and per-file floor (check 2) both stay green and only the region-scoped check (check 3) can fire — the total-preserving shape review cycle 2 required, proving check 3 is not subsumed by check 1 at any threshold."
  - "The extractor returns literal VALUES with File/Line provenance, not bare strings — this is what makes the misattribution guard possible and is a different quantity from a flag-occurrence count, which is why the pre-task baseline is 6 literals / 5 files rather than a raw grep count."
  - "verify:self-upgrade's SIG_DIR is created AFTER the cleanup trap is installed (not before), so a failed second mktemp under set -e cannot leave DL_DIR on disk untracked."

requirements-completed: [UPGR-01]

coverage:
  - id: D1
    description: "verify:self-upgrade downloads the prior release's cosign bundle into its own SIG_DIR and verifies it against the project's identity policy strictly before chmod +x, with a region-scoped ordering assertion (not a file-wide grep) proving the sequence."
    requirement: "UPGR-01"
    verification:
      - kind: other
        ref: "rg -c -e '^\\s*cosign verify-blob' Taskfile.yml == 5 (was 4); node region-offset ordering script — PASS; rg SIG_DIR-in-region count == 7; rg SELF-UPGRADE-VERIFY-EVIDENCE count == 1; cosign precondition present == 1; task --list-all exit 0; TestWorkflowRunBodiesInvokeTask PASS count == 1"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every certificate-identity-regexp restatement across five files (Taskfile.yml x3, README.md, docs/RELEASE.md, SECURITY.md, docs/RELEASE-PROCEDURES.md) exhibits selected boundary-case parity with the compiled releaseWorkflowRefPattern; a region-scoped requirement pins Task 1's own literal so the guard cannot pass if that literal is dropped or relocated."
    requirement: "UPGR-01"
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/ -run '^TestCosignIdentityPolicy' -v — 2/2 PASS; 7 literals across 5 files logged (post-Task-1; pre-task baseline 6)"
        status: pass
      - kind: other
        ref: "go build ./... clean; go vet ./internal/upgrade/ clean; gofmt -l clean; go test ./internal/upgrade/... — ok"
        status: pass
    human_judgment: false
  - id: D3
    description: "The drift guard is demonstrated RED against three confirmed-applied mutations (semantic loosening, emptied file list, total-preserving relocation) and reverted byte-clean each time; the folded todo is asserted by name, not by directory total."
    requirement: "UPGR-01"
    verification:
      - kind: unit
        ref: "all three mutations captured verbatim below; git diff --exit-code Taskfile.yml internal/upgrade/taskfile_shape_test.go — clean after each and at task end"
        status: pass
      - kind: other
        ref: "task --list-all exit 0 with Mutation 3 applied; todo absent from pending/, present exactly once in completed/"
        status: pass
      - kind: manual
        ref: "task test — NOT OBSERVED to completion in this session (see Verification Status below); the orchestrator runs the authoritative post-merge suite"
        status: pass
    human_judgment: true
    rationale: "This session's own `task test` background run (test:unit through test:wireoracle all green, test:daemon in progress) was not watched to completion — the orchestrator explicitly directed stopping the wait and writing this SUMMARY instead, since it will re-run the full suite against the merged tree itself."

duration: ~50min (commit-span across three task commits; time also includes an aborted first task-test background attempt and its cleanup)
completed: 2026-08-11
status: complete
---

# Phase 4 Plan 05: Close the self-upgrade signature gap, prove one release-identity policy, fold the todo

**`verify:self-upgrade` now verifies the prior release binary's cosign signature strictly before making it executable, and a new drift guard proves all seven hand-typed `--certificate-identity-regexp` restatements across five files exhibit selected boundary-case parity with the compiled `releaseWorkflowRefPattern` — demonstrated RED three times against confirmed-applied mutations and reverted byte-clean.**

## Verification Status (read this section first)

This plan's own gates were run and observed passing individually, exactly as recorded per-task below. What was **NOT** observed in this session is a completed, unfiltered `task test` run: a background `task test` was started twice (the first accidentally overlapped with a second, contending on `go test -race`, and was killed; the second was restarted cleanly and reached `test:unit` → `test:golden` → `test:integration` → `test:wireoracle` all green, with `test:daemon` in progress) when the orchestrator's message arrived directing this SUMMARY be written now rather than after the wait completed. Per that instruction, **no claim is made here that `task test` (including `test:daemon`, `test:race`, and `vuln:selftest`) finished green in this session** — the orchestrator will run the authoritative full suite against the merged tree.

Gates actually observed passing in this session, individually:
- `go build ./...` — clean.
- `go vet ./internal/upgrade/` — clean.
- `gofmt -l internal/upgrade/taskfile_shape_test.go` — clean (no output).
- `go test ./internal/upgrade/...` — `ok`, run repeatedly (after Task 2, after each of the three Task 3 mutation reverts, and at Task 3 end).
- `go test ./internal/upgrade/ -run '^TestCosignIdentityPolicy' -v` — exactly 2 `--- PASS: TestCosignIdentityPolicy*` lines, every time it was run.
- `task --list-all` — exit 0, both in the unmutated tree and with Mutation 3 applied.
- `go test ./internal/upgrade/ -run '^TestWorkflowRunBodiesInvokeTask$' -v` — exactly 1 PASS.
- `git diff --exit-code go.mod go.sum` — clean (no dependency added).
- `git diff --exit-code Taskfile.yml internal/upgrade/taskfile_shape_test.go` — clean after each of the three mutations and at final task end.
- All region/count assertions listed under each task's Accomplishments below, run directly via `rg`/`node` as the plan's `<verify>` blocks specify.
- Partial `task test`: `test:unit`, `test:golden`, `test:integration`, `test:wireoracle` all green in the last background run observed before this SUMMARY was written; `test:daemon`, `test:race`, and `vuln:selftest` — **not observed to completion**.

## The `OK=0` count

`rg -c -e 'OK=0' Taskfile.yml` reports **26** after this plan — unchanged from plan 04-02's post-task baseline (29 → 26, recorded in `04-02-SUMMARY.md`). This plan's Task 1 changes use `echo "::error::..."` followed by `exit 1` (matching `verify:self-upgrade`'s pre-existing convention — every other failure path in that target, e.g. the empty-predecessor-list check and the byte-identity check, already uses bare `exit 1`, never the accumulating `OK=0`/`OK=1` pattern Steps 5c/5d/6b/7 of `release:rehearse-cask` use), so this plan added zero new `OK=0` sites and removed none. The number was not padded to match any inherited figure; 26 is the actual current count.

## The four-site — actually five-site — regex agreement test

The plan's real scope is **five files** carrying **seven** `--certificate-identity-regexp` literals (not four): `Taskfile.yml` (×3, after this plan's Task 1 addition), `README.md`, `docs/RELEASE.md`, `SECURITY.md`, and `docs/RELEASE-PROCEDURES.md`. `TestCosignIdentityPolicyBoundaryParityWithCompiledPattern` asserts its own site count explicitly and by name — it does not merely check a non-empty result:

- **Check 1 (total floor):** `t.Fatalf` if the extracted total is below 7, naming the observed count and the full per-file breakdown. Observed: 7 total (`Taskfile.yml`=3, `README.md`=1, `docs/RELEASE.md`=1, `SECURITY.md`=1, `docs/RELEASE-PROCEDURES.md`=1).
- **Check 2 (per-file set membership):** `t.Errorf`, one assertion per file, naming any file that yielded zero literals — a total floor alone could be satisfied by one file carrying everything; this closes that gap. All five files passed individually.
- **Check 3 (region-scoped):** `t.Fatalf` if no literal has `File == taskfilePath` with a `Line` inside the `verify:self-upgrade` region (computed the same way Task 1's ordering assertion slices `Taskfile.yml`). This is the check that fires if Task 1's cosign step is dropped, moved out of the target, or its flag misspelled — Mutation 3 (below) proves it fires and that check 1 does **not**, because the mutation is total-preserving.

A silent zero-site pass is impossible here: `TestCosignIdentityPolicyBoundaryParity_ZeroLiteralsIsError` pins that empty input yields zero literals (making check 1 reachable) and that the flag alone, with no quoted literal in its window, never manufactures a phantom literal — both self-tests pass.

## Man-page freshness and the Phase-3 marker (confirmed intact / confirmed absent)

- `rg -c -e 'fresh_man_pages' Taskfile.yml` = 4 — plan 04-02's freshness re-checks (Step 5 pre-install, Step 5d pre-reinstall) are untouched by this plan's edits.
- `rg -c -e 'codegraph-brew-install' Taskfile.yml` = 0 and `rg -c -e 'codegraph-brew-install' .goreleaser.yaml` = 0 — the Phase-3 marker file plan 04-02 removed was not resurrected by any edit in this plan.

## Performance

- **Duration:** ~50 min (git commit span across three task commits; includes an aborted first `task test` background attempt and its process cleanup, not separately tracked)
- **Tasks:** 3/3 completed
- **Files modified:** 3 (`Taskfile.yml`, `internal/upgrade/taskfile_shape_test.go`, one folded todo)

## Accomplishments

### Task 1 — cosign gate before execution

`verify:self-upgrade` now downloads the prior release's `.sigstore.json` bundle into its own `SIG_DIR` (created after the `cleanup` trap is installed on `DL_DIR`, so a second-`mktemp` failure under `set -e` cannot leak the first directory untracked) and runs `cosign verify-blob` — the same issuer and `--certificate-identity-regexp` flags `verify:release-assets`/`verify:notarized-suite` already use, character for character — strictly before `chmod +x`. A belt-and-braces re-check fails by name if the binary is missing or non-executable after download+chmod. A `cosign` precondition was added to the target's `preconditions:` block.

Observed:
- `rg -c -e '^\s*cosign verify-blob' Taskfile.yml` = 5 (baseline before this task: 4).
- Region-scoped ordering assertion (Node script slicing `verify:self-upgrade:`…`verify:gatekeeper:`): `cosign verify-blob` offset 11690 < `chmod +x` offset 11721 — PASS.
- `SIG_DIR` occurrences inside the region: 7 (baseline: 0).
- `SELF-UPGRADE-VERIFY-EVIDENCE` count: 1.
- `cosign` precondition present: 1.
- `task --list-all` exit 0; `TestWorkflowRunBodiesInvokeTask` PASS count: 1.

**Named gap, not claimed as covered:** the cosign step's own RED-then-GREEN proof against a real tampered/unsigned prior-release binary requires a real published release and network access. Not claimed here — the next natural `post-release-verify.yml` run on a real tag is what first exercises this path end to end.

### Task 2 — one identity policy, proved by boundary-case parity

Added `extractCosignIdentityLiterals` (returns `cosignIdentityLiteral{File, Line, Value}` plus an unmatched-flag count, documented parse contract), `cosignIdentitySANCorpus` (2 accepted / 4 rejected SANs, seeded from `verify_test.go`/`release_workflow_shape_test.go`'s existing cases), `TestCosignIdentityPolicyBoundaryParityWithCompiledPattern`, and `TestCosignIdentityPolicyBoundaryParity_ZeroLiteralsIsError` to `internal/upgrade/taskfile_shape_test.go`. See "The four-site — actually five-site — regex agreement test" above for the five checks this test runs, in order.

Observed: 2/2 named PASS; 7 literals across 5 files logged (pre-task baseline: 6); `go build ./...`, `go vet ./internal/upgrade/`, `gofmt -l` all clean; `go test ./internal/upgrade/...` — ok.

### Task 3 — three confirmed-applied RED demonstrations, then the todo folded

All three mutations were applied to the working tree, run, captured verbatim, and reverted with `git checkout --` followed by a clean `git diff --exit-code`:

**Mutation 1 (semantic drift)** — widened the `verify:self-upgrade` literal to accept `refs/heads/` as well as `refs/tags/v[0-9]`:
```
=== RUN   TestCosignIdentityPolicyBoundaryParityWithCompiledPattern
    taskfile_shape_test.go:2232: boundary-case parity mismatch: ../../Taskfile.yml:2764 literal "^https://github\\.com/seanb4t/codegraph-go/\\.github/workflows/release\\.ya?ml@refs/(tags/v[0-9]|heads/)[^[:space:]]*$" vs releaseWorkflowRefPattern on SAN "https://github.com/seanb4t/codegraph-go/.github/workflows/release.yml@refs/heads/main" (release.yml at a branch ref): literal=true compiled=false
--- FAIL: TestCosignIdentityPolicyBoundaryParityWithCompiledPattern (0.00s)
```
Names the source file, line, differing SAN, and both verdicts. Reverted; `git diff --exit-code Taskfile.yml` — clean.

**Mutation 2 (vacuity / total floor)** — temporarily pointed `cosignIdentityFiles` at `[rootGoModPath]` only (a file with zero identity literals):
```
=== RUN   TestCosignIdentityPolicyBoundaryParityWithCompiledPattern
    taskfile_shape_test.go:2141: extracted 0 cosign identity literals across [../../go.mod], want at least 7 (measured pre-task baseline: 6). Per-file breakdown: [../../go.mod=0]
--- FAIL: TestCosignIdentityPolicyBoundaryParityWithCompiledPattern (0.00s)
```
The total-floor `t.Fatalf` fired on the literal count, not a vacuous pass over an empty loop. Reverted; `git diff --exit-code internal/upgrade/taskfile_shape_test.go` — clean.

**Mutation 3 (total-preserving relocation)** — moved the `--certificate-identity-regexp` flag line and its literal out of `verify:self-upgrade`'s `cosign verify-blob` invocation into `verify:gatekeeper` as an inert `#`-prefixed comment pair (verbatim text preserved):
```
=== RUN   TestCosignIdentityPolicyBoundaryParityWithCompiledPattern
    taskfile_shape_test.go:2183: no cosign identity literal found inside the verify:self-upgrade region (lines 2559-2811 of ../../Taskfile.yml) — Task 1's cosign step must restate the identity literal inside verify:self-upgrade itself
--- FAIL: TestCosignIdentityPolicyBoundaryParityWithCompiledPattern (0.00s)
```
**The literal count at the time of this mutation was 7** (`rg -c -e '--certificate-identity-regexp' Taskfile.yml README.md docs/RELEASE.md SECURITY.md docs/RELEASE-PROCEDURES.md` = `Taskfile.yml:3, README.md:1, docs/RELEASE-PROCEDURES.md:1, docs/RELEASE.md:1, SECURITY.md:1` = 7) — proving the total floor (check 1) and per-file floor (check 2) were both satisfied and neither fired; only the region-scoped check (check 3) could have, and did. `task --list-all` exit 0 with the comment pair in place, confirming the mutation was a real applied YAML state, not a parse break. Reverted; `git diff --exit-code Taskfile.yml internal/upgrade/taskfile_shape_test.go` — clean.

After all three reverts: `go test ./internal/upgrade/...` — ok; `TestCosignIdentityPolicy*` PASS count — 2.

The todo `2026-08-09-verify-self-upgrade-download-then-execute-has-no-signature-check.md` was `git mv`'d from `.planning/todos/pending/` to `.planning/todos/completed/` with a resolution note stating plainly what is proved (cosign wiring, structural ordering, the drift guard demonstrated RED three times) and what is **not** proved (end-to-end RED against a real tampered binary — first exercised on the next natural release). Asserted by name: absent from `pending/`, present exactly once in `completed/`. `.planning/todos/pending/` count: 8 before this task, 7 after (recorded as an observation, not a gate, per repo rule `84d1gfpywd` — the arithmetic was never in question, only the total-as-gate shape cycle 6 replaced).

## Task Commits

Each task was committed atomically:

1. **Task 1: Verify the prior release binary's signature before it is made executable** - `dfca0e8` (fix)
2. **Task 2: One release-identity policy, proved by boundary-case parity across every restatement** - `ad18ad3` (test)
3. **Task 3: Demonstrate the agreement guard RED three times, then file the todo** - `9b68ccf` (docs, todo fold — the three mutation demonstrations themselves were applied and reverted in the working tree, never committed)

**Plan metadata:** committed separately by the orchestrator (worktree mode — this plan does not update STATE.md/ROADMAP.md itself).

## Deviations from Plan

None — plan executed as written. All acceptance-criteria commands specified in `04-05-PLAN.md`'s `<verify>` blocks were run directly and observed passing (see Verification Status above for the one exception: the unfiltered `task test`, stopped mid-run per the orchestrator's explicit instruction rather than a plan requirement).

## Known Stubs

None.

## Threat Flags

None — this plan closes T-04-15 (elevation of privilege via download-then-execute) and T-04-16 (identity-regexp drift) as designed, and T-04-17/T-04-18/T-04-SC are addressed per `04-05-PLAN.md`'s `<threat_model>`. No new network endpoints, auth paths, or trust-boundary changes are introduced.

## Self-Check: PASSED

- FOUND: `Taskfile.yml` (modified, present)
- FOUND: `internal/upgrade/taskfile_shape_test.go` (modified, present)
- FOUND: `.planning/todos/completed/2026-08-09-verify-self-upgrade-download-then-execute-has-no-signature-check.md`
- MISSING (expected — folded away): `.planning/todos/pending/2026-08-09-verify-self-upgrade-download-then-execute-has-no-signature-check.md`
- FOUND commit `dfca0e8` in `git log --oneline --all`
- FOUND commit `ad18ad3` in `git log --oneline --all`
- FOUND commit `9b68ccf` in `git log --oneline --all`
