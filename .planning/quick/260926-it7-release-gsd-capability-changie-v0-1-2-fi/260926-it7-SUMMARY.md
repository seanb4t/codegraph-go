---
phase: quick-260926-it7
plan: "01"
subsystem: infra
tags: [gsd-core, capability, changie, changelog, bash, taskfile, tdd, worktree-mutation]

requires:
  - phase: 02-phase-close-fragment-capability
    provides: gsd-capability-changie install, check:changie 12-leg guard, CONTRIBUTING install docs, 02-REVIEW.md findings (WR-01, CR-01, IN-01)
provides:
  - gsd-capability-changie v0.1.2 (whole-string --pr validation, 33/33 legs) published privately and installed at both project and global scope
  - check:changie leg 13 proving the real `task changie --` path passes a hostile body byte-literally, with RED proven by a confirmed-applied unquoted-splice mutation
  - CONTRIBUTING.md and 02-REVIEW.md updated to close out Phase 2's three code review findings
affects: [changelog-fragment-capability, gsd-capability-changie, ci-check-changie]

actuals:
  tokens: 8700
  tasks: 3
  commits: 6
  plan_head_before: a335232d (codegraph-go); v0.1.1 (gsd-capability-changie)

tech-stack:
  added: []
  patterns:
    - "Whole-string validation with an explicit-digit-set `case` statement (bash 3.2-safe), replacing per-line grep pipelines that a multi-line value can slip through"
    - "check:changie leg driving the real configured `task <cmd> --` path with a hostile payload built from single-quoted literal segments, so the check's own shell never expands the injected command"
    - "RED-by-mutation in a disposable detached `git worktree add --detach`, never on the main tree, with byte-clean revert and worktree/porcelain equality proofs"

key-files:
  created: []
  modified:
    - "gsd-capability-changie:skills/changie-fragments/scripts/write-fragments.sh (is_positive_int whole-string helper)"
    - "gsd-capability-changie:test/run.sh ([bad-pr:multiline] leg, EXPECTED_LEGS=33)"
    - "gsd-capability-changie:capability.json, SKILL.md, README.md (v0.1.2 metadata and install spec)"
    - "Taskfile.yml (check:changie leg 13, /13 renumbering)"
    - "CONTRIBUTING.md (install specs #v0.1.1 -> #v0.1.2)"
    - ".planning/phases/02-phase-close-fragment-capability/02-REVIEW.md (append-only resolution note)"

key-decisions:
  - "Maintainer pre-approved the v0.1.2 tag push and the private-repo publish/upgrade cycle on 2026-09-26, so Task 1 carried no checkpoint (D-13 pattern)."
  - "CR-01 was not reproduced against the real go-task 3.52.0 (it shell-quotes CLI_ARGS), so the residual was closed as a committed test-coverage guard (check:changie leg 13) rather than a code fix, proven able to fail via a deliberate mutation."
  - "Maintainer decision (interactive, at the keyboard, 2026-09-26): the one-line content-snapshot diff produced by the global surface pass (Claude Code's own `skills/synced/<id>/.last-complete-round` sync-round bookkeeping) is host bookkeeping, not a guard violation. Recorded here as a deviation with the recommendation that future snapshot guards exclude `skills/synced/*/.last-complete-round` (or all of `skills/synced/`)."

requirements-completed: [CAP-03, CAP-04, CHG-04]

duration: multi-session (2026-09-25 through 2026-09-26)
completed: 2026-09-26
status: complete
---

# Phase quick-260926-it7 Plan 01: Release gsd-capability-changie v0.1.2 and Close Phase 2 Findings Summary

**gsd-capability-changie v0.1.2 fixes a multi-line `--pr` validation hole (WR-01), both codegraph-go installs are repointed to it through gsd-core verbs, and `check:changie` gained a 13th leg proving the real `task changie --` path never re-shells a fragment body — proven by a confirmed-applied unquoted-splice mutation.**

## Performance

- **Tasks:** 3/3 completed
- **Files modified:** 3 in codegraph-go (Taskfile.yml, CONTRIBUTING.md, 02-REVIEW.md), 5 in gsd-capability-changie (write-fragments.sh, test/run.sh, capability.json, SKILL.md, README.md), plus machine state (both capability installs)

## Accomplishments

- **WR-01 fixed and released.** `write-fragments.sh` validates `--pr` and the looked-up PR with a single whole-string `is_positive_int` helper (`case "$1" in ''|0*|*[!0123456789]*)`), so an embedded or trailing newline, a leading zero, a sign or any non-digit character is refused before anything runs. `[bad-pr:multiline]` went RED first, then GREEN. Released as the annotated tag `v0.1.2` on `main`'s tip, pushed fast-forward over HTTPS. `v0.1.0` and `v0.1.1` are untouched; the repository stays `PRIVATE`.
- **CR-01 residual closed as test coverage.** `check:changie` gained leg 13, which drives the real, pinned go-task 3.52.0 `task changie --` path with a hostile body (`$(...)`, backticks, `;`) and proves the body arrives as a byte-literal `body: <payload>` line with no marker file created. The leg was proven able to fail against a confirmed-applied mutation that unquotes `{{.CLI_ARGS}}`.
- **IN-01 fixed.** All `# Leg N/11` / `# Leg N/12` comments and `[N/12]` strings in `check:changie` are renumbered to `/13`.
- **Both installs repointed to v0.1.2** through `gsd-tools capability install`/`capability set` — never a hand copy, never `GSD_HOME`. The project install's bundled script and `capability.json` are byte-identical to the tag; so is the global `SKILL.md`.
- **CONTRIBUTING.md and 02-REVIEW.md updated.** CONTRIBUTING names `#v0.1.2` in both install specs; `02-REVIEW.md` carries an append-only resolution note closing WR-01, CR-01 and IN-01.

## Task Commits

**gsd-capability-changie** (`$CAP`, `v0.1.1..v0.1.2`):

1. `801c5df` — `test(quick-260926-it7): add failing multi-line --pr refusal leg (WR-01)`
2. `24abce3` — `fix(quick-260926-it7): validate --pr and the looked-up PR as whole strings (WR-01)`
3. `2631063` — `docs(quick-260926-it7): prepare v0.1.2 metadata and install spec`
4. Annotated tag `v0.1.2` (tag object `3760a7a6f845988a30e18dcc4c7defb164e4711f`, peeling to `2631063f05fffc7152d70a9c98eb0b7479a0029a`), pushed to `origin/main` and `origin/refs/tags/v0.1.2`.

**codegraph-go** (`a335232d..HEAD`):

1. `79f13598` — `test(quick-260926-it7): prove task changie -- passes a hostile body literally and renumber check:changie legs (CR-01, IN-01)`
2. `2d229c0d` — `docs(quick-260926-it7): point CONTRIBUTING at capability v0.1.2 (CAP-04)`
3. `4d6049a4` — `docs(quick-260926-it7): record WR-01, CR-01 and IN-01 resolutions in 02-REVIEW`

_Note: the quick-task orchestrator commits this SUMMARY.md, STATE.md and the PLAN.md separately; they are not part of the task commits above._

## Files Created/Modified

- `gsd-capability-changie:skills/changie-fragments/scripts/write-fragments.sh` — new `is_positive_int` helper; both `--pr` and the looked-up PR validate against it
- `gsd-capability-changie:test/run.sh` — new `[bad-pr:multiline]` leg, `EXPECTED_LEGS=33`
- `gsd-capability-changie:capability.json`, `skills/changie-fragments/SKILL.md`, `README.md` — v0.1.2 metadata, `#v0.1.2` install specs, updated `bad-pr` error-table row
- `Taskfile.yml` — `check:changie` leg 13 (real `task changie --` path with a hostile payload) and full `/13` renumbering
- `CONTRIBUTING.md` — both install specs changed from `#v0.1.1` to `#v0.1.2`
- `.planning/phases/02-phase-close-fragment-capability/02-REVIEW.md` — append-only `## Resolution (quick 260926-it7, 2026-09-26)` section

## TDD Gate Compliance

RED-first at the test commit `801c5df` (before `24abce3` existed), against `v0.1.1` behavior. `CHANGIE_BIN` pointed at codegraph-go's pinned changie build. Exactly 19 `ok [` lines, then the expected failure:

```
ok [dispatch-default] render-hooks verify:post lists exactly one changie step with the key absent
run.sh: [dispatch-false] matching changie steps (key false) = 0
ok [dispatch-false] render-hooks verify:post omits the changie step when the key is false
run.sh: [dispatch-true] matching changie steps (key true) = 1
ok [dispatch-true] render-hooks verify:post lists exactly one changie step with the key true
ok [skip:gsd-tools-missing] GSD_TOOLS pointing at a nonexistent path skips cleanly
ok [skip:disabled] workflow.changie_fragments=false skips cleanly
ok [skip:changie-unavailable] an unresolvable configured changie command skips cleanly, naming the command
ok [skip:changie-empty] an empty workflow.changie_command is rejected or skips as changie-unavailable
ok [skip:no-changie-config] a missing .changie.yaml skips cleanly
ok [skip:phase-not-found] an unknown phase number skips cleanly
ok [skip:no-summaries] a phase directory with no SUMMARY files skips cleanly
ok [note:gh-missing] an unresolvable gh executable is a note, and --list continues with pr none
ok [note:pr-lookup-failed] a failing gh lookup is a note carrying gh's stderr with tokens redacted
ok [note:no-open-pr] no open PR for the branch is a note naming the branch, and --list continues with pr none
ok [note:pr-lookup-invalid] a gh lookup result that is not a positive integer is a note and never reaches a fragment
ok [pr-override] an explicit --pr wins over the gh lookup, which is not consulted
ok [bad-pr] --pr 0, --pr abc and --pr -5 all exit 2 with error: bad-pr
::error::run.sh: [bad-pr:multiline] expected exit 2, got 0: changie-fragments: phase 01 /private/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/tmp.6hEvYBrpZC/work/.planning/phases/01-fixture
changie-fragments: range main..HEAD
changie-fragments: kinds Breaking,Features,Fixes,Performance,Dependencies
changie-fragments: pr 12
DANGEROUS
```

19 `ok [` lines confirmed by count (`grep -c '^ok \['` = 19). This RED was reproduced live from a disposable detached worktree at `801c5df` for this SUMMARY (the original session's transcript was not persisted to a durable file); the worktree was removed immediately after capture and `git -C "$CAP" worktree list` / `status --porcelain` returned to the pre-check state.

GREEN at `v0.1.2` (HEAD, commit `2631063`):

```
ok [no-pr-required-host] a host that still requires PR refuses a no-PR write with error: changie-failed and nothing is written
ok [no-side-effects] $CAP_ROOT and the global Claude skills listing are unchanged
run.sh: 33 of 33 legs passed against a scratch project
```

`shellcheck` is clean on `write-fragments.sh`, `test/run.sh`, `test/fixtures/bin/gh` and `test/fixtures/bin/fake-task`.

## Mutation (CR-01 leg 13)

Rule 84d1gfpywd — RED proven by a confirmed-applied mutation in a disposable detached worktree, never on the main tree. Reproduced for this SUMMARY (the original session's transcript was not persisted to a durable file); the worktree was created fresh, exercised, reverted and removed within this task.

**Pre-mutation GREEN** (worktree at HEAD `4d6049a4`):

```
check:changie: [10/13] with only Fixes fragments present, next auto = v0.14.1 (patch successor of v0.14.0) — ok
check:changie: [11/13] a Breaking fragment derives the minor successor v0.15.0 (pre-1.0: Breaking -> minor) — ok
check:changie: [12/13] changie batch auto --dry-run renders the no-PR fragment without a link and a PR fragment with its link (dry run wrote nothing) — ok
check:changie: [13/13] task changie -- passed a body carrying command substitution, backticks and ; byte-literal, and nothing executed — ok
check:changie: 13 of 13 checks passed against a scratch copy (source tree byte-unchanged)
```

**Mutation applied proof:**

- `git -C "$WT" diff --numstat`: `1	1	Taskfile.yml` (exactly one line changed)
- Mutated line present once: `      - "{{.GO_TOOL_CHANGIE}} changie{{range .CLI_ARGS_LIST}} {{.}}{{end}}"`
- Original line `      - "{{.GO_TOOL_CHANGIE}} changie {{.CLI_ARGS}}"` absent from the worktree's `Taskfile.yml`

**RED transcript** (exit 201; legs 1-12 all `— ok`, then the failure):

```
check:changie: found 14 seeded version files under .changes/ (floor 14)
check:changie: [1/13] changie latest = v0.14.0 (newest CHANGELOG.md heading v0.14.0; baseline v0.14.0) — ok
check:changie: [2/13] scratch unreleased/ holds 0 fragments before next auto
check:changie: [2/13] changie next auto (empty unreleased/) refused: no unreleased changes found for automatic bumping — ok
check:changie: [3/13] changie merge --dry-run byte-identical to CHANGELOG.md (dry run wrote nothing) — ok
check:changie: [4/13] changie new -k Fixes writes exactly one fragment with kind/body set — ok
check:changie: [5/13] changie new with no PR accepted: fragment written without a PR field — ok
check:changie: [6/13] changie new -k Undeclared refused: invalid kind: Undeclared — ok
check:changie: [7/13] changie new -m PR=0 refused: input below minimum: 0 < 1 — ok
check:changie: [8/13] changie new -m PR=abc refused: invalid number — ok
check:changie: [9/13] two back-to-back new -k Fixes calls raised the fragment count to 4 (sub-second fragmentFileFormat holds) — ok
check:changie: [10/13] with only Fixes fragments present, next auto = v0.14.1 (patch successor of v0.14.0) — ok
check:changie: [11/13] a Breaking fragment derives the minor successor v0.15.0 (pre-1.0: Breaking -> minor) — ok
check:changie: [12/13] changie batch auto --dry-run renders the no-PR fragment without a link and a PR fragment with its link (dry run wrote nothing) — ok
::error::check:changie: [13/13] task changie -- re-shelled the fragment body: the injected command created /var/folders/.../leg13.marker
task: Failed to run task "check:changie": exit status 1
```

**Byte-clean revert proof:**

- `git -C "$WT" checkout -- Taskfile.yml`
- `git -C "$WT" status --porcelain --untracked-files=all` → empty
- `cmp` against `git show HEAD:Taskfile.yml` → equal

**GREEN re-run** (worktree, post-revert):

```
check:changie: [12/13] changie batch auto --dry-run renders the no-PR fragment without a link and a PR fragment with its link (dry run wrote nothing) — ok
check:changie: [13/13] task changie -- passed a body carrying command substitution, backticks and ; byte-literal, and nothing executed — ok
check:changie: 13 of 13 checks passed against a scratch copy (source tree byte-unchanged)
```

**Worktree and porcelain before/after** — both byte-equal:

- Before: `worktree /Volumes/Code/github.com/seanb4t/codegraph-go` (HEAD at the time) and the pre-existing, untouched `worktree-brave-river-a240` (unrelated session, branch `chore/octopusignore`). Main-tree porcelain: only the pre-existing untracked `.planning/quick/260926-it7-.../260926-it7-PLAN.md`.
- After: identical listing (`diff` of the two `worktree list --porcelain` captures produced no output) and identical porcelain (`diff` produced no output). `git worktree remove` succeeded without `--force`; `git worktree prune` was a no-op. No path under the mutation's `mktemp -d` parent remains in the worktree listing.

## Publish Proof

- `gh repo view seanb4t/gsd-capability-changie --json visibility --jq .visibility` → `PRIVATE`
- `GIT_TERMINAL_PROMPT=0 git ls-remote https://github.com/seanb4t/gsd-capability-changie.git`:
  ```
  2631063f05fffc7152d70a9c98eb0b7479a0029a	HEAD
  2631063f05fffc7152d70a9c98eb0b7479a0029a	refs/heads/main
  60d4967ad300643b30f4d947e64a459974c9ed9f	refs/tags/v0.1.0
  5700e60fb497e033d87ff953c4e957444993628a	refs/tags/v0.1.0^{}
  7305b9e46a5a60e62a2b9ec4b94b6561c3367ea9	refs/tags/v0.1.1
  35b674eb13a7dbd64932648840dc9f447c95dda3	refs/tags/v0.1.1^{}
  3760a7a6f845988a30e18dcc4c7defb164e4711f	refs/tags/v0.1.2
  2631063f05fffc7152d70a9c98eb0b7479a0029a	refs/tags/v0.1.2^{}
  ```
  `v0.1.0` and `v0.1.1` unchanged; `v0.1.2` is an annotated tag; both `refs/heads/main` and `refs/tags/v0.1.2^{}` peel to `2631063`.
- Clean HTTPS clone (`git clone --branch v0.1.2 https://github.com/seanb4t/gsd-capability-changie.git`) into a fresh `mktemp -d`, `CHANGIE_BIN` pointed at codegraph-go's pinned changie build: `run.sh: 33 of 33 legs passed against a scratch project`.

## Project and Global Install Records

**Project install** (codegraph-go, `--scope project`):

- Before: 0.1.1 from `#v0.1.1` (per plan `<interfaces>` baseline).
- After: `{"id":"changie","version":"0.1.2","source":"https://github.com/seanb4t/gsd-capability-changie.git#v0.1.2",...}`.
- `.gsd/capabilities/changie/capability.json` `cmp`-equal to `git -C "$CAP" show v0.1.2:capability.json`.
- `.gsd/capabilities/changie/skills/changie-fragments/scripts/write-fragments.sh` `cmp`-equal to the tag's copy.
- `gsd-tools query config-get workflow.changie_command --raw` → `task changie --`.
- render-hooks `verify:post` changie-step count → `1`.
- Live WR-01 proof against the installed bundle: `--phase 2 --pr $'12\nDANGEROUS' --list` → exit `2`, `changie-fragments: error: bad-pr: not a positive integer: 12\nDANGEROUS`, no `changie-fragments: pr ` line.
- Live `--list` proof: `gh pr view 88 --json state --jq .state` → `OPEN` at the time of the check, and `--phase 2 --list` printed `changie-fragments: pr 88`.
- `git status --porcelain --untracked-files=all -- . ':(exclude).planning'` was empty both before and after the install and both live runs.

**Global install** (maintainer's machine, `--scope global`):

- Before: global ledger read 0.1.1 from `#v0.1.1`; `$SKILLS` had 142 top-level entries (67 symlinks, 75 directories); `find -L` outside `gsd-changie-fragments/` reached 1933 files.
- Commands run: `gsd-tools capability install "$SPEC" --scope global`, then `gsd-tools capability set changie --enable --runtime claude --scope global`.
- After, verified now: global ledger `{"id":"changie","version":"0.1.2","source":"https://github.com/seanb4t/gsd-capability-changie.git#v0.1.2",...}`; `$SKILLS/gsd-changie-fragments/SKILL.md` `cmp`-equal to `git -C "$CAP" show v0.1.2:skills/changie-fragments/SKILL.md`; `.gsd-capability-skill` marker contains `changie`; `~/.gsd/capabilities/changie/capability.json` `cmp`-equal to the tag's copy.
- Surface unchanged (re-verified for this SUMMARY): `$SKILLS` top-level entries still 142 (67 symlinks, 75 directories), `find -L` outside `gsd-changie-fragments/` still 1933 files — matching the pre-install baseline exactly.
- codegraph-go's render-hooks changie-step count is still `1`.
- **Deviation (recorded, resolved):** during the original global surface pass, the content-snapshot comparison differed in exactly one line: `~/.claude/skills/synced/<id>/.last-complete-round`. This path is Claude Code's own account-skill sync bookkeeping (`skills/synced/<id>/` holds claude.ai-provided skills such as `built-in-browser`, `chrome-browser`, `computer-use`; `.last-complete-round` is a sync-round id Claude Code rewrites on every sync round). No gsd-core verb writes under `skills/synced/`. **Maintainer decision (interactive, at the keyboard, 2026-09-26): Proceed** — treat this as host bookkeeping, not a guard violation. Recommendation carried forward: future snapshot guards should exclude `skills/synced/*/.last-complete-round` (or all of `skills/synced/`).

## Upgrade

Three commands used, following the D-13/D-14 publish-and-upgrade pattern:

```
gsd-tools capability install https://github.com/seanb4t/gsd-capability-changie.git#v0.1.2 --scope project
gsd-tools capability install https://github.com/seanb4t/gsd-capability-changie.git#v0.1.2 --scope global
gsd-tools capability set changie --enable --runtime claude --scope global
```

`capability install` over an existing entry is the upgrade swap (`capability update` re-fetches the recorded spec and cannot move to a new tag). A git source is fetched as a depth-1 clone of the default branch, so now that `main` has moved to `v0.1.2`, `#v0.1.1` no longer installs — this is why `CONTRIBUTING.md` was repointed in the same run.

## Decisions Made

- Maintainer pre-approved the `v0.1.2` tag push on 2026-09-26 (no checkpoint needed for Task 1's costly/irreversible step).
- CR-01 residual closed as test coverage rather than a code change, since go-task 3.52.0 already shell-quotes `CLI_ARGS`; the new leg 13 makes that property a permanent, checked guarantee instead of a one-off manual finding.
- The `.last-complete-round` snapshot diff was accepted as host bookkeeping per the maintainer's interactive decision (see Deviations below).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 4 - Architectural/judgment, resolved interactively] Global skills-directory content snapshot diverged by one host-bookkeeping file**
- **Found during:** Task 3, step 1 (global surface pass)
- **Issue:** The plan's guard requires the `find -L` content snapshot of everything outside `gsd-changie-fragments/` to be byte-equal before and after the global install. One line differed: `~/.claude/skills/synced/<id>/.last-complete-round`.
- **Analysis:** `skills/synced/<id>/` holds claude.ai-provided skills (`built-in-browser`, `chrome-browser`, `computer-use`, …) that Claude Code itself materializes and syncs; `.last-complete-round` is a round id Claude Code rewrites on every sync round, independent of any gsd-core action. No gsd-core verb touches `skills/synced/`.
- **Resolution:** Surfaced to the maintainer as a checkpoint. Maintainer decision (interactive, at the keyboard, 2026-09-26): **Proceed** — this is host bookkeeping, not a guard violation.
- **Files affected:** none (host-side sync state only, outside this repo and outside gsd-core's write surface).
- **Follow-up recommendation:** future snapshot guards should exclude `skills/synced/*/.last-complete-round` (or all of `skills/synced/`) to avoid re-flagging this same host behavior.

---

**Total deviations:** 1 (resolved via explicit maintainer decision, not auto-fixed by rule)
**Impact on plan:** No code or install-state change required. The global install itself is verified byte-identical to the tag on every other measured surface.

## Issues Encountered

The original session's RED-transcript and mutation-transcript files were not persisted to durable storage in the scratchpad in the exact form needed for this SUMMARY. Both were reproduced fresh in this continuation from the already-committed, already-tagged artifacts (test commit `801c5df` for the WR-01 RED, and a fresh disposable detached worktree at the current HEAD for the CR-01 mutation), so the transcripts pasted above are live, verified evidence rather than a paraphrase.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 2's code review (`02-REVIEW.md`) is fully closed: WR-01 fixed and released, CR-01 residual closed as test coverage, IN-01 fixed.
- `gsd-capability-changie` `v0.1.2` is the installed version at both project and global scope, matching `CONTRIBUTING.md`'s documented install spec.
- STATE.md shows the project already transitioned to Phase 3 (Rename-Stub Removal); this quick task did not touch phase sequencing.

---
*Phase: quick-260926-it7*
*Completed: 2026-09-26*

## Self-Check: PASSED

- FOUND: `Taskfile.yml`, `CONTRIBUTING.md`, `.planning/phases/02-phase-close-fragment-capability/02-REVIEW.md`, `.planning/quick/260926-it7-release-gsd-capability-changie-v0-1-2-fi/260926-it7-SUMMARY.md`
- FOUND (codegraph-go): `79f13598`, `2d229c0d`, `4d6049a4`
- FOUND (gsd-capability-changie): `801c5df`, `24abce3`, `2631063`, tag `v0.1.2`
