---
phase: 02-phase-close-fragment-capability
plan: 02
subsystem: capability
tags: [gsd-core, capability, changie, changelog, documentation, mutation-testing]

# Dependency graph
requires:
  - phase: 02-01
    provides: "A validated capability.json, SKILL.md and write-fragments.sh, plus a 29-leg green test suite in gsd-capability-changie"
provides:
  - "gsd-capability-changie:README.md and gsd-capability-changie:LICENSE, satisfying every D-10 README/LICENSE item"
  - "A pre-publication judgment rehearsal proving SKILL.md, followed exactly as written against the fixture phase, turns one user-visible change into exactly one plain Features fragment and skips the planning-only change (D-06)"
  - "Three deterministic write-fragments.sh guards (disabled-key skip, explicit-pathspec staging, trailer idempotency) shown RED against a confirmed mutation and GREEN after a byte-clean revert, in a disposable detached worktree (D-09)"
affects: ["02-03", "02-04", "02-05"]

actuals:
  tokens: 7500
  tasks: 2
  commits: 5
  plan_head_before: 8ca26b4e

tech-stack:
  added: []
  patterns:
    - "Self-referential wording defect: a rule forbidding a literal substring must not itself contain that substring — caught by a source `rg -F` count, not by reading the prose"
    - "Judgment rehearsal as a pre-publication gate (T-02-08): the executor manually follows a to-be-dispatched skill's SKILL.md exactly as written, against a disposable fixture project, and asserts every concrete output before the tag that pins the wording is cut"
    - "Detached-worktree mutation demonstration (01-MUTATION-LOG precedent) reused for a bash script guard: pre-mutation gate, one-file mutation, applied confirmation via `diff --numstat`, verbatim RED transcript, byte-clean revert, GREEN re-run"

key-files:
  created:
    - gsd-capability-changie:README.md
    - gsd-capability-changie:LICENSE
  modified:
    - gsd-capability-changie:skills/changie-fragments/SKILL.md
    - .planning/phases/02-phase-close-fragment-capability/02-CAPABILITY-LOG.md

key-decisions:
  - "SKILL.md's Rules section told readers to 'Keep any `.claude/` path out of this file's body' — a rule about not containing a literal substring that itself contained that substring. Caught by this task's own acceptance criterion (`rg -F -o '.claude/' SKILL.md | wc -l` must print 0), not by the six rehearsal assertions, which all passed unmodified. Reworded to 'Keep any host-specific installed-skill directory path out of this file's body' and re-verified: 29/29 suite, and the rehearsal repeated cleanly in a second fresh scratch project."
  - "The judgment rehearsal's --write call passed --summaries 01-01,01-02 (both pending ids), per SKILL.md's own instruction to pass 'the exact ids from the pending lines' — even though only 01-01 produced an entry. This is what SKILL.md specifies, and it is why the Changie-Summaries trailer correctly reads 01-01,01-02 even though 01-02 contributed no fragment."
  - "Family (b)'s mutation (broadening git add -- \"${WRITTEN_FILES[@]}\" to git add -A) was caught by the [write] leg, not [commit-scope] — the first leg that compares HEAD's file set against the expected 3 fragments fires before the later leg that checks pre-existing untracked/unstaged files are left alone. Both legs share the same underlying guard; which one names the RED is a leg-ordering fact, not a defect."

patterns-established:
  - "Pre-publication judgment rehearsal: before a capability's first tag, manually follow its skill exactly as written against a disposable fixture and assert every concrete output — catches SKILL.md wording defects while the tag is still free to move (D-10 costly-reversibility)."

requirements-completed: [CAP-01, CAP-03]

coverage:
  - id: D1
    description: "gsd-capability-changie:LICENSE is byte-identical to codegraph-go's LICENSE — verbatim MIT text, no appended paragraph (D-10)"
    requirement: "CAP-01"
    verification:
      - kind: other
        ref: "cmp codegraph-go/LICENSE gsd-capability-changie/LICENSE"
        status: pass
    human_judgment: false
  - id: D2
    description: "gsd-capability-changie:README.md covers install (both scopes), both config keys, manual re-run, every skip/error id, and trailer semantics (D-10, CAP-01)"
    requirement: "CAP-01"
    verification:
      - kind: other
        ref: "README token-presence loop over 11 required substrings (plan <verify> block 2)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Following SKILL.md exactly as written against the fixture phase produces exactly one plain Features fragment for the user-visible change and skips the planning-only change (D-06)"
    requirement: "CAP-03"
    verification:
      - kind: manual_procedural
        ref: "02-CAPABILITY-LOG.md#Judgment rehearsal (SKILL.md followed by the executor, pre-publication) — all six assertions PASS, repeated in a second fresh scratch project after the SKILL.md wording fix"
        status: pass
    human_judgment: true
    rationale: "This rehearsal proves the mechanical output of one manual, executor-performed pass over a fixed fixture. The real Skill-tool dispatch — an actual model reading arbitrary real SUMMARYs and exercising genuine judgment — is exercised for the first time in 02-05, after materialization. No automated test can assert an LLM's judgment quality in the general case; a human review of that real dispatch is the backstop this rehearsal cannot replace."
  - id: D4
    description: "The disabled-key skip, explicit-pathspec staging, and trailer idempotency guards each go RED on a named leg against a confirmed one-file mutation, and return to 29 of 29 after a byte-clean revert, in a disposable detached worktree (D-09, rule 84d1gfpywd)"
    requirement: "CAP-03"
    verification:
      - kind: integration
        ref: "02-CAPABILITY-LOG.md#Mutation families (three RED transcripts: [skip:disabled], [write], [idempotent]; three GREEN re-runs at 29 of 29)"
        status: pass
    human_judgment: false
  - id: D5
    description: "After this plan, gsd-capability-changie's HEAD still passes 29 of 29 legs, its working tree is clean, and it has exactly one worktree"
    verification:
      - kind: other
        ref: "test/run.sh at HEAD 5700e60; git worktree list; git status --porcelain"
        status: pass
    human_judgment: false

duration: ~70min
completed: 2026-09-26
status: complete
---

# Phase 2 Plan 2: README, LICENSE, Judgment Rehearsal, and Mutation Evidence Summary

**A verbatim MIT LICENSE and a full D-10 README for the changie capability, a pre-publication rehearsal proving SKILL.md turns the fixture phase into exactly one plain Features fragment, and three write-fragments.sh guards shown RED against confirmed mutations and GREEN after a byte-clean revert — catching and fixing a self-referential wording defect in SKILL.md before the tag is cut.**

## Performance

- **Duration:** ~70 min
- **Completed:** 2026-09-26T01:16Z
- **Tasks:** 2
- **Files created:** 2 (`gsd-capability-changie:README.md`, `gsd-capability-changie:LICENSE`)
- **Files modified:** 2 (`gsd-capability-changie:skills/changie-fragments/SKILL.md`, `02-CAPABILITY-LOG.md`)

## Accomplishments
- `gsd-capability-changie:LICENSE` is byte-identical to codegraph-go's `LICENSE` (verbatim MIT, `cmp` confirms).
- `gsd-capability-changie:README.md` covers what the capability does, both federated config keys, the per-clone project-scope install command pinned to `#v0.1.0`, the per-machine global-scope step that makes `/gsd-changie-fragments` dispatchable in Claude Code (and why a project-scope install alone is not enough), the manual re-run command, all 12 script skip ids plus the SKILL.md-level `script-missing`, all 6 error ids with exit codes, idempotency/trailer semantics, the commit convention, `test/run.sh` development instructions, and the no-CI discretion call.
- A pre-publication judgment rehearsal followed `SKILL.md` exactly as written (arguments `1 --pr 4242 --repo "$R"`) against a disposable fixture project mirroring `test/run.sh`'s own fixture build. Every one of the six required assertions passed: commit subject `docs(01): add changelog fragments`, `Changie-Summaries: 01-01,01-02` trailer, exactly one fragment file, `kind: Features` and `PR: "4242"`, a body mentioning `--json`, and zero jargon tokens (`WIDGET-`/`D-NN`/plan-number) in the body.
- The rehearsal's *own* task acceptance criteria caught a second, unrelated defect: SKILL.md's Rules section told readers to avoid a `` `.claude/` `` path while itself containing that literal substring. Fixed (`fix(02-02)`), suite re-confirmed at 29/29, and the full rehearsal repeated in a fresh scratch project with all six assertions passing again.
- Three `write-fragments.sh` guards — the disabled-key skip, the explicit-pathspec commit staging, and the trailer-based idempotency check — were each shown RED against a one-line, confirmed-applied mutation in a disposable detached `git worktree`, reverted byte-cleanly, and returned to 29 of 29. No mutation or edit touched `$CAP`'s real working tree; teardown confirmed one worktree, a clean status, and HEAD unchanged.
- `gsd-capability-changie`'s final HEAD (`5700e60fb497e033d87ff953c4e957444993628a`) passes 29 of 29 legs, has a clean working tree, one worktree, and no remote — ready for `02-03`'s tag and publish.

## Task Commits

Task commits live in **two repositories** (D-09 — `$CAP`'s history is outside this roadmap):

**`gsd-capability-changie`** (`git -C /Volumes/Code/github.com/seanb4t/gsd-capability-changie log --reverse --format='%h %s'`, this plan's commits only):
1. `715cc9a` — `docs(02-02): add README and MIT license`
2. `5700e60` — `fix(02-02): reword the no-.claude/-path rule to avoid the literal substring it forbids`

**codegraph-go** (evidence + metadata, this plan's own repo):
3. `d395e19b` — `docs(02-02): record the judgment rehearsal`
4. `3bf9a89b` — `docs(02-02): record capability mutation evidence`
5. (this commit) — `docs(02-02): complete README, LICENSE, judgment rehearsal, and mutation evidence plan` (plan metadata: this SUMMARY, STATE.md, ROADMAP.md, REQUIREMENTS.md)

**`gsd-capability-changie`'s final HEAD for this plan, which 02-03 tags:** `5700e60fb497e033d87ff953c4e957444993628a`

## Files Created/Modified
- `gsd-capability-changie:README.md` — what the capability does, requirements, install (both scopes), configuration table, manual re-run, skip-reason and error tables, idempotency/trailer semantics, commit convention, development instructions, no-CI discretion call
- `gsd-capability-changie:LICENSE` — verbatim copy of codegraph-go's MIT `LICENSE`
- `gsd-capability-changie:skills/changie-fragments/SKILL.md` — one-line wording fix: the no-`.claude/`-path rule no longer contains the substring it forbids
- `.planning/phases/02-phase-close-fragment-capability/02-CAPABILITY-LOG.md` — two new sections: the judgment rehearsal (both runs, including the wording-fix cycle) and the three mutation families (setup, three RED/GREEN cycles, teardown)

## Decisions Made
See `key-decisions` in the frontmatter: the SKILL.md self-referential wording fix, the
`--summaries 01-01,01-02` (both pending ids) instruction from SKILL.md itself, and the
[write]-vs-[commit-scope] leg-ordering note for family (b).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] SKILL.md's no-`.claude/`-path rule contained the literal substring it forbade**
- **Found during:** Task 1, running this task's own acceptance criteria after the first rehearsal pass (all six rehearsal assertions themselves passed unmodified)
- **Issue:** The Rules section read `` Keep any `.claude/` path out of this file's body — the script resolves its own location without one. `` — a rule about avoiding a literal substring that itself contained that substring, defeating the acceptance check `rg -F -o '.claude/' SKILL.md | wc -l` prints 0.
- **Fix:** Reworded to `` Keep any host-specific installed-skill directory path out of this file's body — the script resolves its own location without one. `` — same restriction, no longer self-violating.
- **Files modified:** `gsd-capability-changie:skills/changie-fragments/SKILL.md`
- **Verification:** `rg -F -o '.claude/' SKILL.md | wc -l` now prints `0`; `test/run.sh` re-confirmed 29/29 at the new HEAD; the full judgment rehearsal was repeated end-to-end in a second fresh scratch project (`$R2`), with all six assertions passing again.
- **Committed in:** `5700e60` (`fix(02-02): reword the no-.claude/-path rule to avoid the literal substring it forbids`)

---

**Total deviations:** 1 auto-fixed bug (Rule 1).
**Impact on plan:** The fix was necessary for the task's own acceptance criteria to pass and was caught before publication, exactly as the pre-publication rehearsal (T-02-08 mitigation) is designed to do. No scope creep — the fix is a one-line wording change with no behavioral effect on the script.

## Issues Encountered
None beyond the deviation above. Each 29-leg suite run takes roughly 60-100 seconds wall-clock
(consistent with 02-01's SUMMARY note); this plan ran it seven times (one after the README/LICENSE
commit, three RED and three GREEN runs across the three mutation families, plus the positive
control) — all backgrounded and polled rather than run inline, to stay within command timeouts.

## User Setup Required
None - no external service configuration required. This plan publishes nothing (`git -C $CAP remote` is empty); publishing is 02-03's maintainer-approved step.

## Next Phase Readiness
- The capability repository has a complete README and LICENSE, a rehearsed and now-correct SKILL.md, and three deterministic guards proven non-vacuous by recorded mutation — ready for `02-03` (publish + tag `v0.1.0` + draft PR) and `02-04` (codegraph-go install).
- `02-03` tags `gsd-capability-changie`'s HEAD `5700e60fb497e033d87ff953c4e957444993628a`.
- No blockers.

---
*Phase: 02-phase-close-fragment-capability*
*Completed: 2026-09-26*

## Self-Check: PASSED

Both `gsd-capability-changie:` files (`README.md`, `LICENSE`) and the modified
`skills/changie-fragments/SKILL.md` exist on disk. `.planning/phases/02-phase-close-fragment-capability/02-CAPABILITY-LOG.md`
carries both the `## Judgment rehearsal` and `## Mutation families` sections. All commit hashes
(`715cc9a`, `5700e60` in `gsd-capability-changie`; `d395e19b`, `3bf9a89b` in codegraph-go) resolve
via `git log --oneline --all`. `test/run.sh` at `gsd-capability-changie` HEAD `5700e60` reports
`29 of 29 legs passed against a scratch project`. `git -C gsd-capability-changie status --porcelain`
is empty, and `git -C gsd-capability-changie worktree list` shows exactly one entry.
