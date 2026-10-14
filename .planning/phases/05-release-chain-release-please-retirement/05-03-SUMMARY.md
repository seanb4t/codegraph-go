---
phase: 05-release-chain-release-please-retirement
plan: 03
subsystem: infra
tags: [goreleaser, changie, taskfile, release-notes, dry-run, guard-tests]

requires:
  - phase: 05-release-chain-release-please-retirement
    provides: "05-01: release:batch, releaseChain* test helpers, 05-MUTATION-LOG.md; 05-02: release-tag.yml, release:tag, the GRD-16 scan, the D-06 amendment shape this plan mirrors"
provides:
  - "Taskfile release:goreleaser derives TAG and NOTES=.changes/${TAG}.md, refuses an absent or empty file, and runs goreleaser release --clean --release-notes \"${NOTES}\" (REL-12, D-08, D-12, C6)"
  - "Taskfile release:dry-run in changie's pre-tag form (GORELEASER_CURRENT_TAG from changie latest, --skip=publish,sign,validate, no --snapshot) with the same refusal and a RELEASE-NOTES-EVIDENCE line (REL-13, D-14)"
  - "TestReleaseGoreleaserPassesReleaseNotes, TestReleaseDryRunRehearsesReleaseNotes, TestGoreleaserChangelogNotDisabled; TestReleaseBlockDoesNotRewriteReleaseBody restated for the changie chain (GRD-15, partial)"
  - "release:dry-run-signed desc names release-pr.yml and release-tag.yml (DOCS-14, partial)"
  - "REL-12, REL-13 and ROADMAP Phase 5 SC2 amended to the wiring as built (D-12, D-14)"
  - "05-MUTATION-LOG.md: 05-03 RED and GREEN transcripts, the REL-12 diff evidence, Families 05-03/a-e, the two-run REL-13 rehearsal"
affects: [05-04, 05-05, 05-07, phase-06-live-run]

actuals:
  tokens: 5231
  tasks: 3
  commits: 6

tech-stack:
  added: []
  patterns:
    - "the flag that makes the Release body lives in the Taskfile target release.yml already calls, so release.yml keeps its run-body convention and moves by one comment"
    - "a rehearsal target switches from --snapshot to GORELEASER_CURRENT_TAG plus --skip=publish,sign,validate when the thing under rehearsal is a pipe that --snapshot skips"
    - "a guard against a missing or empty input is held by an ordered-token test over non-comment text and by a mutation that deletes the guard"

key-files:
  created: []
  modified:
    - Taskfile.yml
    - .goreleaser.yaml
    - .github/workflows/release.yml
    - internal/upgrade/changie_release_shape_test.go
    - internal/upgrade/goreleaser_shape_test.go
    - .planning/REQUIREMENTS.md
    - .planning/ROADMAP.md
    - .planning/phases/05-release-chain-release-please-retirement/05-MUTATION-LOG.md

key-decisions:
  - "D-12 followed as written: the notes flag is in Taskfile.yml's release:goreleaser; release.yml differs from 7b12cb92 by one comment hunk; no hash pin of release.yml was added"
  - "D-14 followed as written: nothing in this plan asserts how GoReleaser renders the Release body; the evidence line's sha256 records which file was passed, nothing more"
  - "No precondition was added for the five MACOS_* variables (D-01): the rehearsals ran under env -u for those five names instead"
  - "release:dry-run keeps the four-binary jq and file -b assertions untouched; release:dry-run-signed keeps --snapshot because it rehearses signing, not notes"

patterns-established:
  - "releaseChainTaskBlock and releaseChainOrderedTokens: shared helpers in package upgrade for reading a Taskfile target's cmds and asserting tokens at increasing byte offsets"

requirements-completed: [REL-12, REL-13]

coverage:
  - id: D1
    description: "release:goreleaser passes --release-notes .changes/${TAG}.md for the tag at HEAD, refuses an absent or empty file before GoReleaser runs, has one release invocation and one TAG definition; release.yml run bodies stay the two task invocations"
    requirement: REL-12
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_release_shape_test.go#TestReleaseGoreleaserPassesReleaseNotes"
        status: pass
      - kind: other
        ref: "Families 05-03/b and 05-03/c in 05-MUTATION-LOG.md"
        status: pass
    human_judgment: false
  - id: D2
    description: "release.yml differs from 7b12cb92 by one hunk of comment lines; its trigger and OIDC scoping guards stay green"
    requirement: REL-12
    verification:
      - kind: other
        ref: "git diff -U0 7b12cb92 -- .github/workflows/release.yml (one @@ hunk, comment lines only)"
        status: pass
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestReleaseWorkflowTriggerIsTagPushOnly, TestOIDCWriteScopedToSingleGoreleaserJob"
        status: pass
    human_judgment: false
  - id: D3
    description: ".goreleaser.yaml sets no changelog.disable (absent or boolean false) and the release block declares none of name_template, header, footer, draft, disable"
    requirement: REL-12
    verification:
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestGoreleaserChangelogNotDisabled, TestReleaseBlockDoesNotRewriteReleaseBody"
        status: pass
      - kind: other
        ref: "Families 05-03/a and 05-03/d in 05-MUTATION-LOG.md"
        status: pass
    human_judgment: false
  - id: D4
    description: "release:dry-run derives TAG from changie latest, refuses an absent or empty notes file, prints RELEASE-NOTES-EVIDENCE and runs the non-snapshot invocation with --release-notes; the four-binary assertions remain"
    requirement: REL-13
    verification:
      - kind: unit
        ref: "internal/upgrade/changie_release_shape_test.go#TestReleaseDryRunRehearsesReleaseNotes"
        status: pass
      - kind: other
        ref: "Family 05-03/e in 05-MUTATION-LOG.md"
        status: pass
    human_judgment: false
  - id: D5
    description: "task release:dry-run exited 0 on this darwin host twice before any v0.15 tag existed: on HEAD (v0.14.1) and after release:batch in a scratch worktree (v0.15.0), each with its RELEASE-NOTES-EVIDENCE line and the four-binary success line"
    requirement: REL-13
    verification:
      - kind: integration
        ref: "05-MUTATION-LOG.md ## 05-03 REL-13 rehearsal (Run A and Run B)"
        status: pass
    human_judgment: false
  - id: D6
    description: "REL-12, REL-13 and Phase 5 SC2 amended per D-12 and D-14 in the D-06 shape, heading counts unchanged"
    requirement: REL-12
    verification:
      - kind: other
        ref: "05-03-PLAN.md Task 2 verify (rg over REQUIREMENTS.md, ROADMAP.md and heading-count equality against 5a8603f5)"
        status: pass
    human_judgment: false
  - id: D7
    description: "The changed release:dry-run runs in CI's cross-build job on namespace-profile-macos-6x14-tahoe, and its linux binaries run on real linux/amd64 and linux/arm64"
    requirement: REL-13
    verification: []
    human_judgment: true
    rationale: "Needs a push to PR #88 so linux-cross-canary.yml runs; this plan does not push. Deferred to phase verification (see Deferred Checks)."

duration: 8min
completed: 2026-10-14
status: complete
plan_head_before: e22a3322cad0782e2797e0cd3ab8bf598bf81795
plan_head_after: 9666a9f1ddfad3640bbce6a2a748276d2f748cc4
commits: 6
---

# Phase 5 Plan 03: Release notes from the changie file Summary

**`task release:goreleaser` now runs `goreleaser release --clean --release-notes .changes/${TAG}.md` and stops on an absent or empty notes file, `task release:dry-run` rehearses exactly that in changie's pre-tag form (no `--snapshot`) and exited 0 twice on darwin before any v0.15 tag existed, and `release.yml` moved by one comment line.**

This plan changes release tooling and tests only. No codegraph user-visible behavior changes. Nothing was pushed, no tag was created, no PR was touched, no release was published.

## Performance

- **Duration:** 8 min
- **Started:** 2026-10-14T21:41:53Z
- **Completed:** 2026-10-14T21:50Z (last task commit; metadata commits follow)
- **Tasks:** 3
- **Files modified:** 8 (0 created)

## Accomplishments

- `release:goreleaser` hoists `TAG="$(git describe --exact-match --tags HEAD)"` and `NOTES=".changes/${TAG}.md"`, refuses a missing or empty file with `::error::release:goreleaser: ... is missing or empty` and `exit 1` (C6: GoReleaser alone only warns on an empty file), then makes its single release call `"${GORELEASER_BIN}" release --clean --release-notes "${NOTES}"`. The `GITHUB_OUTPUT` block reuses the hoisted `TAG`; its own definition is deleted, so `TAG` has one definition.
- `release:dry-run` takes `TAG` from `changie latest` through `GO_TOOL_CHANGIE`, applies the same refusal, prints `RELEASE-NOTES-EVIDENCE tag=... file=... bytes=... sha256=...`, and runs `GORELEASER_CURRENT_TAG="${TAG}" "${GORELEASER_BIN}" release --skip=publish,sign,validate --clean --release-notes "${NOTES}"`. No executed line carries `--snapshot` (it skips GoReleaser's changelog pipe, Pitfall 2). The four-binary `jq` and `file -b` assertions are unchanged.
- `release.yml` differs from `7b12cb92` by exactly one hunk, one comment line (the census rewording near line 188). The flag reaches goreleaser because the unchanged step runs `task release:goreleaser`.
- Three new shape tests plus the restated `TestReleaseBlockDoesNotRewriteReleaseBody`. `.goreleaser.yaml`'s release-block comment now names the changie chain and `--release-notes`; no YAML key changed.
- `release:dry-run-signed`'s desc names `release-pr.yml` and `release-tag.yml` (DOCS-14).
- REL-12, REL-13 and Phase 5 SC2 amended per D-12 and D-14 in the D-06 shape. Heading counts are unchanged.
- Five mutation families (a-e), each seen RED against a confirmed-applied mutation and reverted byte-clean. Two `release:dry-run` runs recorded with exit 0.

## TDD Gate Compliance

RED evidence is the `test(05-03)` commit that landed before the implementation plus the pasted `--- FAIL:` transcript. `gsd-tools check tdd-red-evidence` was not used: it parses TAP only and always reports INVALID_RED for `go test` (rule x1cjy9vyhq).

| Gate | Commit | Subject |
|------|--------|---------|
| RED | `594bb30d` | test(05-03): expect release notes from the changie file in the release and its rehearsal (REL-12, REL-13) |
| GREEN | `b390a9d2` | ci(05-03): pass the changie version file to goreleaser as the release body (REL-12, REL-13) |
| follow-up | `c4adcfd5` | ci(05-03): name Phase 1 D-12 where release:goreleaser's comment cites the tag rule (REL-12), comment only |

The RED commit holds exactly `internal/upgrade/changie_release_shape_test.go` and `internal/upgrade/goreleaser_shape_test.go`; the GREEN commit holds exactly `Taskfile.yml`, `.goreleaser.yaml` and `.github/workflows/release.yml`. The first two commits on `goreleaser_shape_test.go` and `.goreleaser.yaml` since `5a8603f5` are `test(05-03)` then `ci(05-03)`.

`go test ./internal/upgrade/ -count=1 -v` at `594bb30d` (assertion failures on the missing wiring, no build error; exactly these two fail, `TestGoreleaserChangelogNotDisabled` passes because no changelog block exists and its RED is Family 05-03/a):

```
    changie_release_shape_test.go:1573: release:goreleaser cmds: token "NOTES=\".changes/${TAG}.md\"" (notes path from the tag) not found at or after byte offset 1622
    changie_release_shape_test.go:1573: release:goreleaser cmds: token "[ ! -s \"${NOTES}\" ]" (empty-notes check) not found at or after byte offset 1622
    changie_release_shape_test.go:1573: release:goreleaser cmds: token "::error::release:goreleaser: ${NOTES} is missing or empty" (empty-notes error) not found at or after byte offset 1622
    changie_release_shape_test.go:1573: release:goreleaser cmds: token "\"${GORELEASER_BIN}\" release --clean --release-notes \"${NOTES}\"" (release call with notes) not found at or after byte offset 1869
    changie_release_shape_test.go:1582: release:goreleaser release invocation is "\"${GORELEASER_BIN}\" release --clean", want exactly `"${GORELEASER_BIN}" release --clean --release-notes "${NOTES}"`
--- FAIL: TestReleaseGoreleaserPassesReleaseNotes (0.00s)
    changie_release_shape_test.go:1654: release:dry-run runs a --snapshot line, which skips the notes pipe: "\"${GORELEASER_BIN}\" release --snapshot --skip=publish,sign --clean"
    changie_release_shape_test.go:1677: release:dry-run invokes changie 0 times, want exactly 1 (latest)
--- FAIL: TestReleaseDryRunRehearsesReleaseNotes (0.00s)
FAIL	github.com/seanb4t/codegraph-go/internal/upgrade	0.606s
```

(Abridged here; the full transcript, with all seven dry-run token failures, is in `05-MUTATION-LOG.md` under `## 05-03 RED-first transcript`.) `TestReleaseBlockDoesNotRewriteReleaseBody` changed prose only, so it has no behavioral RED; Family 05-03/d shows its unchanged logic still bites.

GREEN: the seven named tests pass (7 `--- PASS` lines), the whole `internal/upgrade` package passes, `task lint:actions` exits 0 and `task check:goreleaser` exits 0.

## REL-12 evidence

`git diff 7b12cb92c407ae7e495a62b4ac38d8685e64307c -- .github/workflows/release.yml` (one hunk, comment lines only):

```diff
@@ -185,7 +185,7 @@ jobs:
       # separate-job placement RESEARCH.md Open Question 2 recommended is
       # not usable here. This step therefore mints inside the release job,
       # immediately before the step that consumes it, pinned to the exact
-      # same Action + SHA release-please.yml already carries. This places
+      # same Action + SHA release-pr.yml and release-tag.yml carry. This places
       # one more Action inside the OIDC boundary (T-01-29's accepted
       # surface, now widened by one step) — chosen because the alternative
       # was measured not to work, not because it was preferred. Compensating
```

The `release:goreleaser` cmds hunks of `git diff 5a8603f5 -- Taskfile.yml` (the desc hunk adds the sentence about `.changes/${TAG}.md` and the hand-pushed rc tag; the full diff is in the log):

```diff
-        "${GORELEASER_BIN}" release --clean
+        #
+        # The Release body is the changie version file for the tag being
+        # released (REL-12, D-08), handed to GoReleaser with --release-notes.
+        # GoReleaser only warns on an empty notes file, so an absent or empty
+        # one stops here instead (C6): a Release published with an empty
+        # body cannot be repaired by re-cutting the tag (Phase 1 D-12).
+        TAG="$(git describe --exact-match --tags HEAD)"
+        NOTES=".changes/${TAG}.md"
+        if [ ! -s "${NOTES}" ]; then
+          echo "::error::release:goreleaser: ${NOTES} is missing or empty — GoReleaser would publish a release with an empty body (REL-12, C6)"
+          exit 1
+        fi
+        "${GORELEASER_BIN}" release --clean --release-notes "${NOTES}"
...
         if [ -n "${GITHUB_OUTPUT:-}" ]; then
-          TAG="$(git describe --exact-match --tags HEAD)"
           CHECKSUMS_FILE="dist/codegraph_${TAG}_checksums.txt"
```

## REL-13 rehearsal

Section `## 05-03 REL-13 rehearsal` of `05-MUTATION-LOG.md`. Host darwin, `gh api .../matching-refs/tags/v0.15 --jq length` returned 0 before both runs. The five `MACOS_*` variables are defined in this host's environment, so both runs used `env -u` for them (Pitfall 8); no precondition was added (D-01).

```
Run A (HEAD, changie latest = v0.14.1)
RELEASE-NOTES-EVIDENCE tag=v0.14.1 file=.changes/v0.14.1.md bytes=353 sha256=c7703c6606ccff8991260605d9a5ab9a3fc0f9114109baaa300ea5d9f8c5479d
release:dry-run: four correctly-typed binaries produced in one goreleaser release invocation.
release:dry-run exit=0

Run B (scratch worktree after task release:batch, changie latest = v0.15.0)
RELEASE-NOTES-EVIDENCE tag=v0.15.0 file=.changes/v0.15.0.md bytes=328 sha256=3dca5a9b89b3aebf73790697c91d4312c347c979bef9072a8c0a763ec7f67456
release:dry-run: four correctly-typed binaries produced in one goreleaser release invocation.
release:dry-run exit=0
```

Scope (D-14): this rehearsal proves the notes path passed for the tag, the repository's own non-empty guard and a zero exit before any real tag exists. How GoReleaser renders the Release body is GoReleaser's behavior and is not tested in this phase. The sha256 in each evidence line records which file was passed, nothing more. `shasum -a 256` of each file matched its evidence line. The main checkout's tracked files were unchanged after Run A, and Run B's worktree held only the three lines `release:batch` writes. The scratch worktree was removed.

## Mutation families

All in `05-MUTATION-LOG.md` (`## Family 05-03/a` to `## Family 05-03/e`), each with a pre-mutation gate, applied-confirmation hunk, verbatim RED transcript, byte-clean revert proof and GREEN re-run.

| Family | Mutation | RED |
|---|---|---|
| a | `changelog: disable: true` appended to .goreleaser.yaml | TestGoreleaserChangelogNotDisabled |
| b | ` --release-notes "${NOTES}"` dropped from release:goreleaser | TestReleaseGoreleaserPassesReleaseNotes |
| c | release:goreleaser's `[ ! -s` check block deleted | TestReleaseGoreleaserPassesReleaseNotes |
| d | `header: "x"` planted in the release block | TestReleaseBlockDoesNotRewriteReleaseBody |
| e | `--snapshot` added to release:dry-run's invocation | TestReleaseDryRunRehearsesReleaseNotes |

## Deferred Checks

**Open: linux-cross-canary.yml on PR #88.** `linux-cross-canary.yml` is the only workflow that runs `task release:dry-run`; it triggers on `pull_request` with a `paths:` filter that includes `Taskfile.yml`. This plan does not push, so the changed target has not run in CI. Phase verification must confirm that the latest completed `linux-cross-canary` run on PR #88 whose head commit contains the ci(05-03) commit `b390a9d2bda1b7d96aabb7607fc3367974daf11f` concluded `success`. Read-only evidence: `gh run list --workflow linux-cross-canary.yml --branch gsd/v0.15.0-milestone --event pull_request --json headSha,status,conclusion`, then `git merge-base --is-ancestor b390a9d2bda1b7d96aabb7607fc3367974daf11f <headSha>` for the run cited. The phase does not close until that is confirmed.

## Task Commits

1. **Task 1: RED then GREEN** - `594bb30d` (test), `b390a9d2` (ci), `3a23e66f` (docs, transcripts), `c4adcfd5` (ci, comment-only follow-up)
2. **Task 2: D-12 and D-14 amendments, Families 05-03/a-e** - `e0c70d17` (docs)
3. **Task 3: REL-13 rehearsal** - `9666a9f1` (docs)

**Plan metadata:** the `docs(05-03): complete release-notes plan` commit that follows this file, then the state-update commit.

## Files Created/Modified

- `Taskfile.yml` - release:goreleaser notes flag and guard, release:dry-run non-snapshot rehearsal, release:dry-run-signed desc
- `.goreleaser.yaml` - release-block comment only
- `.github/workflows/release.yml` - one comment line
- `internal/upgrade/changie_release_shape_test.go` - two tests and the `releaseChainTaskBlock` / `releaseChainOrderedTokens` helpers
- `internal/upgrade/goreleaser_shape_test.go` - `TestGoreleaserChangelogNotDisabled`; restated `TestReleaseBlockDoesNotRewriteReleaseBody`
- `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md` - D-12 and D-14 value edits to REL-12, REL-13 and Phase 5 SC2
- `.planning/phases/05-release-chain-release-please-retirement/05-MUTATION-LOG.md` - transcripts, diff evidence, Families a-e, rehearsal

## Decisions Made

- D-12 and D-14 were applied as the maintainer decided them; neither was reopened.
- `release:dry-run` carries `--skip=validate` as the plan states; the run shows `pipe skipped or partially skipped reason=validation is disabled` for GoReleaser's git-state check on a HEAD that is not the tagged commit. Nothing asserts that log text.
- GRD-15 is partial: this plan restated the release-block guard and comment. The release-please registry entries and the rest of the census belong to 05-04, 05-05 and 05-07. DOCS-14 is partial: this plan covers the `release:dry-run-signed` desc only.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] A comment in release:goreleaser cited "D-12" ambiguously**
- **Found during:** Task 2 (building the REL-12 diff evidence)
- **Issue:** the comment I wrote in `release:goreleaser` said "D-12 forbids re-cutting the tag". In this phase D-12 is the flag-placement decision; the tag rule is Phase 1 D-12 (the label `release:record-final-hashes`'s desc already uses). A reader of the diff would take it for the wrong decision.
- **Fix:** reworded to "a Release published with an empty body cannot be repaired by re-cutting the tag (Phase 1 D-12)".
- **Files modified:** `Taskfile.yml` (comment only)
- **Verification:** the `internal/upgrade` package and `task lint:actions` pass after the edit.
- **Committed in:** `c4adcfd5`, a separate `ci(05-03)` commit after the GREEN commit (no amend). The first two commits on the guarded files are still `test(05-03)` then `ci(05-03)`.

**2. [Environment note] The plan commit ledger**
- `.git/gsd-plan-head-before-05-03` did not exist, so there was no stale ledger to reset. It was created holding `e22a3322` (HEAD at start). `commits: 6` is measured from it.

---

**Total deviations:** 1 auto-fixed (1 Rule 1 comment fix), 1 environment note
**Impact on plan:** None on scope. The follow-up changes a comment only.

## Issues Encountered

None. `internal/daemon` TestSoak was not run (package-scoped runs only). `dist/` was left as the last dry run built it; it is gitignored.

## Known Stubs

None.

## Threat Flags

None. The rehearsal ran with `--skip=publish,sign,validate`, no credential, and the five `MACOS_*` variables unset (T-05-15). T-05-12, T-05-13 and T-05-14 are mitigated and tested above.

## User Setup Required

None - no external service configuration required in this plan.

## Next Phase Readiness

- Requirements: REL-12 and REL-13 are done in code and evidence, with REL-13's CI consumer (the linux-cross-canary run) deferred to phase verification as listed above. GRD-15 and DOCS-14 are also declared by later plans in this phase, so they stay open.
- 05-04 onward can rely on `release:goreleaser` carrying the notes flag; the GRD-16 scan from 05-02 covers the new lines (the package passes).
- Unchanged from `5a8603f5` apart from the one comment line: `release.yml`; `internal/upgrade/verify.go`, `tag_authority_test.go`, CHANGELOG.md and every `.changes/v*.md` are untouched.

## Self-Check: PASSED

Modified files exist (Taskfile.yml, .goreleaser.yaml, release.yml, both test files, 05-MUTATION-LOG.md); commits `594bb30d`, `b390a9d2`, `3a23e66f`, `c4adcfd5`, `e0c70d17` and `9666a9f1` are ancestors of HEAD; every task verify command and acceptance criterion was run and passes; the main checkout has no tracked change; no scratch worktree remains.

---
*Phase: 05-release-chain-release-please-retirement*
*Completed: 2026-10-14*
