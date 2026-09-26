---
phase: 02-phase-close-fragment-capability
plan: 03
subsystem: release-tooling
tags: [github, gh-cli, git, capability, changie, milestone-pr, draft-pr]

# Dependency graph
requires:
  - phase: 02-02
    provides: "gsd-capability-changie's tested, clean HEAD 5700e60fb497e033d87ff953c4e957444993628a (29/29 suite, judgment rehearsal, mutation evidence)"
provides:
  - "Private GitHub repository seanb4t/gsd-capability-changie with pushed main and annotated tag v0.1.0 on the tested commit"
  - "A verified git+ssh install spec (git+ssh://git@github.com/seanb4t/gsd-capability-changie.git#v0.1.0) that 02-04 can consume"
  - "An open draft PR (#88, gsd/v0.15.0-milestone -> main) in codegraph-go that the changie-fragments skill's gh pr list lookup resolves"
  - "A STATE.md Blockers/Concerns entry recording the /gsd-ship existing-PR hazard, written through gsd-tools state add-blocker"
affects: [02-04, 02-05, "milestone close (/gsd-ship)"]

# Actuals (#2632)
actuals:
  tokens: 6000
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Outward-facing GitHub actions (repo creation, tag push, PR creation) gated behind a single blocking-human checkpoint that auto-mode cannot approve"
    - "Publish proof by installed artifact, not local tree: a clean SSH clone at the tag runs the repo's own suite, and a real git+ssh capability install is byte-compared against the tag's blob"

key-files:
  created: []
  modified:
    - ".planning/STATE.md"

key-decisions:
  - "Maintainer answered publish-both at Task 1's blocking-human checkpoint: publish the private capability repo and tag v0.1.0, and open the draft milestone PR, using the proposed title with no replacement."
  - "The stale .git/gsd-plan-head-before-02-03 commit-ledger file (dated 2026-09-15, from a same-numbered plan in a prior milestone before phase numbering reset) was overwritten with this plan's true pre-execution HEAD (932ed8a5) so actuals.commits measures this plan's own work, not 48 commits inherited from an unrelated earlier milestone."

patterns-established: []

requirements-completed: [CAP-01, CAP-04]

coverage:
  - id: D1
    description: "Private repository seanb4t/gsd-capability-changie exists with main pushed and annotated tag v0.1.0 on the tested HEAD"
    requirement: "CAP-01"
    verification:
      - kind: other
        ref: "gh repo view seanb4t/gsd-capability-changie --json visibility --jq .visibility -> PRIVATE"
        status: pass
      - kind: other
        ref: "git ls-remote --tags (remote tag object 60d4967a... matches local; peels to 5700e60... = gsd-capability-changie HEAD)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The published content, not the local tree, is proven: a fresh SSH clone at v0.1.0 passes the full suite, and a real git+ssh install produces a byte-identical capability.json"
    requirement: "CAP-01"
    verification:
      - kind: integration
        ref: "test/run.sh at clean clone of v0.1.0 -> 29 of 29 legs passed"
        status: pass
      - kind: integration
        ref: "gsd-tools capability install git+ssh://...#v0.1.0 --scope project (exit 0) + cmp against git show v0.1.0:capability.json (exit 0)"
        status: pass
    human_judgment: false
  - id: D3
    description: "codegraph-go has an open draft PR from gsd/v0.15.0-milestone to main that the changie-fragments skill's own lookup resolves, and it survives the close-draft-prs workflow via the OWNER exemption"
    requirement: "CAP-04"
    verification:
      - kind: other
        ref: "gh pr view 88 --json number,isDraft,state,headRefName,baseRefName -> isDraft true, state OPEN, head gsd/v0.15.0-milestone, base main"
        status: pass
      - kind: other
        ref: "gh pr list --head gsd/v0.15.0-milestone --state open --json number --jq '.[0].number // empty' -> 88"
        status: pass
      - kind: other
        ref: "gh run view 36244263678 (close-draft-prs.yml, event pull_request_target, headBranch gsd/v0.15.0-milestone) -> conclusion skipped; gh pr view 88 --json state -> OPEN"
        status: pass
    human_judgment: false
  - id: D4
    description: "STATE.md Blockers/Concerns records the /gsd-ship existing-PR hazard for milestone close, written only through gsd-tools state add-blocker"
    requirement: "CAP-04"
    verification:
      - kind: other
        ref: "rg -F -o 'gh pr ready' .planning/STATE.md | wc -l -> 1; git diff -- .planning/STATE.md showed only the new blocker line plus tool-owned frontmatter fields (last_updated, state_head)"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-09-26
status: complete
---

# Phase 2 Plan 3: Publish the Capability and Open the Milestone PR Summary

**Published seanb4t/gsd-capability-changie privately at tag v0.1.0 (proven by a clean SSH clone's own suite and a byte-identical git+ssh install), and opened codegraph-go's milestone draft PR #88, which survived the OWNER-exempt close-draft-prs workflow.**

## Performance

- **Duration:** 35 min (continuation from a resolved checkpoint)
- **Started:** 2026-09-26T13:05:00Z
- **Completed:** 2026-09-26T13:40:00Z
- **Tasks:** 2 (1 checkpoint:decision, resolved prior to this continuation; 1 auto)
- **Files modified:** 1 (`.planning/STATE.md`)

## Accomplishments

- Task 1 (checkpoint:decision, `gate=blocking-human`): the maintainer answered **`publish-both`** at the keyboard, approving both the capability publication and the milestone PR, with no replacement title. This continuation re-verified the pre-state read-only (CAP HEAD, clean tree, `gh auth status`, no existing remote/tag/branch/PR) before acting, exactly as instructed.
- Task 2: performed exactly the approved actions and produced the proofs the plan's `must_haves.artifacts` and `<verify>` require:
  1. **Published the capability.** `gh repo create seanb4t/gsd-capability-changie --private --description "gsd-core capability: changie changelog fragments at phase close" --source "$CAP" --remote origin --push` created the repo and pushed `main`. `gh repo view ... --json visibility --jq .visibility` printed `PRIVATE`.
  2. **Tagged and pushed.** `git -c tag.gpgsign=false tag -a v0.1.0 -m "gsd-capability-changie v0.1.0"` on HEAD `5700e60fb497e033d87ff953c4e957444993628a`, then `git push origin v0.1.0`.
  3. **Proved the published content**, not the local tree: a fresh `git clone --depth 1 --branch v0.1.0 git@github.com:...` of the tag ran `test/run.sh` end to end with the pinned `changie` binary built from codegraph-go's own `go.tool-changie.mod`, printing `29 of 29 legs passed`.
  4. **Proved the install spec.** In a scratch project, `gsd-tools capability install git+ssh://git@github.com/seanb4t/gsd-capability-changie.git#v0.1.0 --scope project` exited 0, and the installed `capability.json` was byte-identical (`cmp` exit 0) to `git show v0.1.0:capability.json`.
  5. **Opened the milestone PR.** `git push -u origin gsd/v0.15.0-milestone`, then `gh pr create --draft --base main --head gsd/v0.15.0-milestone --title "feat: v0.15.0 — changie release management" --body-file <file>` opened **PR #88**.
  6. **Proved the PR as the skill will see it.** `gh pr view 88` showed `isDraft: true`, `state: OPEN`, head `gsd/v0.15.0-milestone`, base `main`. The skill's exact lookup, `gh pr list --head gsd/v0.15.0-milestone --state open --json number --jq '.[0].number // empty'`, printed `88`.
  7. **Confirmed it survived the draft closer.** The `close-draft-prs.yml` run created for this PR's `opened` event (`databaseId 36244263678`, `pull_request_target`, `headBranch gsd/v0.15.0-milestone`) completed with `conclusion: skipped` — the OWNER exemption fired as expected — and `gh pr view 88 --json state` still read `OPEN` afterward.
  8. **Recorded the milestone-close hazard** through `gsd-tools state add-blocker` (never by hand), naming PR #88, the unconditional `gh pr create` in `/gsd-ship`'s `create_pr` step, and the alternative (`gh pr ready 88` + body update).

## Task Commits

Each task was committed atomically:

1. **Task 1: Maintainer approval (checkpoint:decision)** — no commit (a decision checkpoint makes no commit; resolved by a prior continuation).
2. **Task 2: Perform the approved actions and prove the published tag and live PR lookup** — `7ac65f80` (docs) — `.planning/STATE.md` (the ship-hazard blocker entry)

**Plan metadata:** committed separately after this SUMMARY (see below).

## Files Created/Modified

- `.planning/STATE.md` — added the `/gsd-ship` existing-PR hazard to Blockers/Concerns via `gsd-tools state add-blocker`

External (not in this repository's tree, per D-09 — the capability repo's own commit trail is outside this roadmap):
- `seanb4t/gsd-capability-changie` (new private GitHub repository): `main` pushed, annotated tag `v0.1.0` pushed
- `seanb4t/codegraph-go` PR #88 (draft, `gsd/v0.15.0-milestone` → `main`)

## Decisions Made

- **Maintainer decision (Task 1):** `publish-both` — publish the private capability at tag `v0.1.0` and open the draft milestone PR, using the proposed title `feat: v0.15.0 — changie release management` (no replacement supplied).
- **Executor decision (Rule 3, blocking issue):** the plan-commit ledger file `.git/gsd-plan-head-before-02-03` already existed on disk, dated 2026-09-15 — from a same-numbered plan in a *prior* milestone (phase numbers reset at each milestone; `--reset-phase-numbers`, ROADMAP note). Using it verbatim would have measured 48 commits spanning an unrelated earlier milestone's phase 2, not this plan's actual work. Overwrote it with this plan's true pre-execution HEAD (`932ed8a50626c619ea3c6eb030fb015eced53baa`, the HEAD this continuation's `<worktree_metadata>` capture also recorded as `expected_base`) so `actuals.commits` measures only this plan's own commits.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Stale cross-milestone commit-ledger file would have corrupted `actuals.commits`**
- **Found during:** Task 2, step 9 (recording actuals for the SUMMARY)
- **Issue:** `.git/gsd-plan-head-before-02-03` existed with a value (`5f5480c2...`) from a same-`{phase}-{plan}`-numbered plan in a prior milestone (phase numbering restarts each milestone), predating this plan's dependency (02-02) entirely. `git rev-list --count` against it returned 48 — commits from an unrelated earlier milestone, not this plan.
- **Fix:** Overwrote the ledger file with this plan's actual pre-execution HEAD, `932ed8a50626c619ea3c6eb030fb015eced53baa` (matching the `expected_base` this continuation's worktree-metadata capture recorded independently).
- **Files modified:** none in the repository tree (the ledger is a `.git`-internal scratch file, not a tracked artifact)
- **Verification:** `git rev-list --count 932ed8a5..HEAD` now returns 1 (the STATE.md hazard commit) prior to this SUMMARY's own commit, matching the plan's actual single-commit Task 2.
- **Committed in:** N/A (ledger is untracked `.git` scratch state, not committed)

---

**Total deviations:** 1 auto-fixed (1 Rule 3 — blocking issue, infra ledger correction, no source/plan-code change).
**Impact on plan:** No scope creep. The fix only corrects internal commit-count bookkeeping for this SUMMARY's `actuals` field; it changed nothing the plan's tasks, verification, or acceptance criteria examine.

## Issues Encountered

None beyond the ledger correction above, which is recorded as a deviation rather than an issue since it required no retry or escalation.

## Verification Results

All plan-level `<verify>` and `<acceptance_criteria>` checks were re-run and passed:

- `gh repo view seanb4t/gsd-capability-changie --json visibility --jq .visibility` = `PRIVATE` — PASS
- Remote tag `refs/tags/v0.1.0` (`60d4967a...`) equals the local annotated tag object; `git cat-file -t v0.1.0` = `tag` (annotated, not lightweight) — PASS
- `gh pr list --head gsd/v0.15.0-milestone --state open ...` = `88`; `gh pr view 88` = `true OPEN main` (isDraft, state, base) — PASS
- `rg -F -o 'gh pr ready' .planning/STATE.md | wc -l` = 1 — PASS
- Commit subject `docs(02-03): record the milestone-PR ship hazard` appears exactly once in `.planning/STATE.md`'s log — PASS
- `rg -F -o 'run.sh: 29 of 29 legs passed' 02-03-SUMMARY.md | wc -l` ≥ 1 (this file, above) — PASS (by construction)
- `git -C gsd-capability-changie rev-parse 'v0.1.0^{commit}'` equals `git -C gsd-capability-changie rev-parse HEAD` (`5700e60...` = `5700e60...`) — PASS
- `gh pr view "$(gh pr list ...)" --json number --jq .number` = `88`, matching the number recorded above — PASS

## Full Transcripts (Task 2, step 9)

**Repository and visibility:**
```
$ gh repo create seanb4t/gsd-capability-changie --private --description "gsd-core capability: changie changelog fragments at phase close" --source "$CAP" --remote origin --push
https://github.com/seanb4t/gsd-capability-changie
To https://github.com/seanb4t/gsd-capability-changie.git
 * [new branch]      HEAD -> main

$ gh repo view seanb4t/gsd-capability-changie --json visibility --jq .visibility
PRIVATE
```

**Tag SHAs (local and remote):**
```
local tag object (v0.1.0):      60d4967ad300643b30f4d947e64a459974c9ed9f
local peeled commit (v0.1.0^{}): 5700e60fb497e033d87ff953c4e957444993628a
remote refs/tags/v0.1.0:         60d4967ad300643b30f4d947e64a459974c9ed9f
remote refs/tags/v0.1.0^{}:      5700e60fb497e033d87ff953c4e957444993628a
```

**Clone-at-tag suite (29/29):**
```
run.sh: changie <scratch>/changie (changie version vdev)
run.sh: found 2 fixture SUMMARY files (floor 2)
ok [ordering] config-set workflow.changie_command is rejected before install
ok [install] capability install stages capability.json, SKILL.md, write-fragments.sh and the ledger
ok [manifest] installed capability.json has id/role/engines/runtimeCompat/steps/config exactly as required
ok [dispatch-default] render-hooks verify:post lists exactly one changie step with the key absent
ok [dispatch-false] render-hooks verify:post omits the changie step when the key is false
ok [dispatch-true] render-hooks verify:post lists exactly one changie step with the key true
ok [skip:gsd-tools-missing] GSD_TOOLS pointing at a nonexistent path skips cleanly
ok [skip:disabled] workflow.changie_fragments=false skips cleanly
ok [skip:changie-unavailable] an unresolvable configured changie command skips cleanly, naming the command
ok [skip:changie-empty] an empty workflow.changie_command is rejected or skips as changie-unavailable
ok [skip:no-changie-config] a missing .changie.yaml skips cleanly
ok [skip:phase-not-found] an unknown phase number skips cleanly
ok [skip:no-summaries] a phase directory with no SUMMARY files skips cleanly
ok [skip:gh-missing] an unresolvable gh executable skips cleanly
ok [skip:pr-lookup-failed] a failing gh lookup skips cleanly, detail carries gh's stderr
ok [skip:no-open-pr] no open PR for the branch skips cleanly, naming the branch
ok [pr-override] an explicit --pr wins over the gh lookup, which is not consulted
ok [bad-pr] --pr 0, --pr abc and --pr -5 all exit 2 with error: bad-pr
ok [write] --list prints pr/pending/kinds and --write commits exactly 3 fragments with correct trailers
ok [idempotent] re-running --list/--write for already-covered summaries yields skip: already-recorded, HEAD and fragment count unchanged
ok [gap-closure] a SUMMARY added after the fragment commit is the only pending id
ok [skip:no-user-visible-change] --write with an empty entries file skips cleanly, HEAD unchanged
ok [undeclared-kind] an undeclared kind exits 2 with error: undeclared-kind, listing the declared kinds, no file written
ok [malformed-entry] a line with no TAB exits 2 with error: malformed-entry, no file written
ok [rollback] a changie failure on the second entry rolls back the first, releases the lock, exits 1 with error: changie-failed
ok [skip:locked] a pre-held lock directory skips cleanly
ok [argv-literal] a body with $(...), backticks, quotes, ; and * reaches the fragment literally and nothing runs
ok [commit-scope] a pre-existing untracked fragment and an unstaged tracked edit are left exactly as they were
ok [no-side-effects] $CAP_ROOT and the global Claude skills listing are unchanged
run.sh: 29 of 29 legs passed against a scratch project
```

**Install-from-tag byte comparison:**
```
$ gsd-tools capability install git+ssh://git@github.com/seanb4t/gsd-capability-changie.git#v0.1.0 --scope project
{
  "status": "installed",
  "id": "changie",
  "version": "0.1.0",
  "scope": "project",
  ...
}
$ cmp <(git -C "$CAP" show v0.1.0:capability.json) <scratch-project>/.gsd/capabilities/changie/capability.json
(exit 0 — no output, byte-identical)
```

**PR and lookup:**
```
$ gh pr create --draft --base main --head gsd/v0.15.0-milestone --title "feat: v0.15.0 — changie release management" --body-file <file>
https://github.com/seanb4t/codegraph-go/pull/88

$ gh pr view 88 --json number,isDraft,state,headRefName,baseRefName
{"baseRefName":"main","headRefName":"gsd/v0.15.0-milestone","isDraft":true,"number":88,"state":"OPEN"}

$ GH_PROMPT_DISABLED=1 gh pr list --head gsd/v0.15.0-milestone --state open --json number --jq '.[0].number // empty'
88
```

**close-draft-prs.yml run:**
```
$ gh run view 36244263678 --json headBranch,event,workflowName,conclusion
{"conclusion":"skipped","event":"pull_request_target","headBranch":"gsd/v0.15.0-milestone","workflowName":"Close Draft PRs"}

$ gh pr view 88 --json state --jq .state
OPEN
```

## User Setup Required

None.

## Milestone-Close Note (not work for this phase)

At milestone close, do **not** run `/gsd-ship`'s create step for `gsd/v0.15.0-milestone` — it calls `gh pr create` unconditionally with no existing-PR probe and will fail. Instead run `gh pr ready 88` and update the PR body. This is recorded in `.planning/STATE.md` Blockers/Concerns.

## Self-Check: PASSED

- `[ -f .planning/phases/02-phase-close-fragment-capability/02-03-SUMMARY.md ]` → FOUND (this file)
- `git log --oneline --all | grep -q 7ac65f80` → FOUND (Task 2 STATE.md commit)
- All `<acceptance_criteria>` re-run above → PASS
- All plan-level `<verification>` items re-run above → PASS
