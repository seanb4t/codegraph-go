---
phase: 12-cli-reference-docs-tail
plan: 02
subsystem: docs
tags: [homebrew, security-framing, readme, cli-reference, todo-resolution]

# Dependency graph
requires:
  - phase: 12-cli-reference-docs-tail (plan 01)
    provides: "docs/CLI-REFERENCE.md generated and committed; task docs:cli:drift gate"
provides:
  - "docs/RELEASE.md untrusted-tap paragraph recommending brew trust --cask seanb4t/tap/codegraph as THE command, with one sentence of security framing"
  - "BREW-01 verification blockquote narrowed to the same narrow form (parenthetical + closing sentence)"
  - "README.md link to docs/CLI-REFERENCE.md inside the existing Homebrew paragraph"
  - "2026-08-10 brew-trust todo resolved via the GSD todo tool"
affects: [12-03-mutation-log-and-security]

# Actuals (#2632)
actuals:
  tokens: 1555
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Wording-only doc change verified via one-time source assertions embedded in the plan's own <verify> block, with no committed test/guard/grep (D-12)"

key-files:
  created: []
  modified:
    - docs/RELEASE.md
    - README.md
    - .planning/STATE.md
    - .planning/todos/completed/2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md (moved from .planning/todos/pending/)

key-decisions:
  - "Compressed the framing sentence's prose (dropped the man-pages/version-check parenthetical the plan's <action> text suggested) to fit the task's own git diff --numstat added-line budget (<=14 for docs/RELEASE.md), while keeping every clause the acceptance criteria actually require: arbitrary Ruby at install time, brew trust as a security control being opted out of, and --cask scoping that opt-out to one cask."

patterns-established: []

requirements-completed: []  # DOCS-05 and DOCS-07 are shared with 12-01/12-03 (shared-ID gate #2388). `requirements.ready-ids` returned 0/2 ready — 12-03 (the last plan declaring both) has not landed yet — so neither is marked complete here.

coverage:
  - id: D1
    description: "docs/RELEASE.md's untrusted-tap paragraph and BREW-01 blockquote recommend `brew trust --cask seanb4t/tap/codegraph` as THE command, with one sentence of security framing, the tap-wide grant named but never spelled, and the quoted Homebrew error trimmed with an ellipsis (DOCS-07)"
    requirement: DOCS-07
    verification:
      - kind: manual_procedural
        ref: "one-time rg -o ... | wc -l assertions from this plan's own <verify> (not committed as a repo test) — narrow-form count 3 (>=3), tap-wide spellings/--tap count 0, ellipsis-quote count 1, 'arbitrary Ruby' count 1 (>=1), 'opting out' count 1 (>=1), 'tap-wide' count 2 (>=2), quoted error first line unchanged count 1, both old 'run the command ... names' phrasings count 0"
        status: pass
    human_judgment: true
    rationale: "D-12: no test, guard, or docs-grep is committed for DOCS-07 by maintainer decision — the wording is verified by reading the diff (pasted below) plus this plan's one-time source assertions, which exist only in this PLAN.md's <verify> block and are not a repo-persisted gate. A human should read the pasted diff to confirm the wording itself reads correctly, per D-12's own design."
  - id: D2
    description: "README.md gains exactly one link to the generated docs/CLI-REFERENCE.md, appended as one sentence inside the existing Homebrew paragraph that already links docs/RELEASE.md, with no other restructuring (D-06)"
    requirement: DOCS-05
    verification:
      - kind: other
        ref: "one-time rg -o counts from this plan's own <verify>: exactly 1 markdown link, 2 total docs/CLI-REFERENCE.md mentions, link 10 lines below the docs/RELEASE.md pointer, README diff +3/-1 lines vs ca015c4b, heading and `| \\`` table-row counts unchanged vs ca015c4b, 'brew trust' absent from README, docs/CLI-REFERENCE.md untouched, commit carries only README.md"
        status: pass
    human_judgment: false

# Metrics
duration: ~15min
completed: 2026-09-13
status: complete
---

# Phase 12 Plan 02: Brew-Trust Security Framing + README CLI-Reference Link Summary

**`docs/RELEASE.md` now tells a Homebrew user to run the narrow `brew trust --cask seanb4t/tap/codegraph` grant, names the security control they're opting out of in one sentence, never spells the tap-wide alternative, and `README.md` gains one link to the generated `docs/CLI-REFERENCE.md` — the 2026-08-10 todo that raised the wording gap is now closed through the GSD todo tool.**

## Performance

- **Duration:** ~15 min
- **Started:** 2026-09-13 (approx, immediately following 12-01)
- **Completed:** 2026-09-13
- **Tasks:** 2
- **Files modified:** 4 (docs/RELEASE.md, README.md, .planning/STATE.md, one todo file moved)

## Accomplishments

- `docs/RELEASE.md`'s untrusted-tap paragraph: the quoted Homebrew error is trimmed to its narrow form with an ellipsis, one framing sentence names the security control (arbitrary Ruby at install time, `brew trust` as the opt-out, `--cask` scoping it to one cask), and the `sh` block runs `brew trust --cask seanb4t/tap/codegraph` — the tap-wide grant is named in prose as existing and not recommended, with no command spelled anywhere
- The BREW-01 verification blockquote's parenthetical and closing sentence are narrowed to the same form, staying a true record of what Homebrew's error actually offered
- `README.md` gains exactly one sentence, inside the existing Homebrew paragraph, linking `docs/CLI-REFERENCE.md` (12-01's generated, drift-guarded reference) — no restructuring
- The 2026-08-10 todo (`brew trust instructions recommend the broader tap grant with no security framing`) is resolved through `gsd_run todo complete`, moved to `.planning/todos/completed/` with `completed:`/`status: completed` stamped by the tool, in the same commit as the wording change
- `.planning/STATE.md`'s Pending Todos → Resolved and filed table move happened in that same commit, values only, no heading/column change

## Task Commits

Each task was committed atomically:

1. **Task 1: `docs/RELEASE.md` brew-trust wording + todo resolution (D-11, D-12)** — `910d5e61` (docs)
2. **Task 2: README.md link to `docs/CLI-REFERENCE.md` (D-06)** — `b9d75ecc` (docs)

**Plan metadata:** this SUMMARY (committed separately per the objective's scope — no ROADMAP.md edit, no further STATE.md edit beyond the todo-row move already committed in Task 1)

## Files Created/Modified

- `docs/RELEASE.md` — untrusted-tap paragraph rewritten (narrow grant, framing sentence, tap-wide named not spelled); BREW-01 blockquote narrowed
- `README.md` — one sentence appended to the Homebrew paragraph, linking `docs/CLI-REFERENCE.md`
- `.planning/todos/completed/2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md` — moved from `.planning/todos/pending/`, `completed: 2026-09-13` / `status: completed` stamped by the tool
- `.planning/STATE.md` — the brew-trust row moved from the Pending Todos table to the `Resolved and filed` table (values only)

## Full Diffs (D-12's verification — read these, not a test)

### `git diff ca015c4b -- docs/RELEASE.md`

```diff
diff --git a/docs/RELEASE.md b/docs/RELEASE.md
index 6bdc1e2d..6daaa347 100644
--- a/docs/RELEASE.md
+++ b/docs/RELEASE.md
@@ -521,16 +521,22 @@ specific to this cask:
 
 ```
 Error: Refusing to load cask seanb4t/tap/codegraph from untrusted tap seanb4t/tap.
-Run `brew trust --cask seanb4t/tap/codegraph` or `brew trust seanb4t/tap` to trust it.
+Run `brew trust --cask seanb4t/tap/codegraph` … to trust it.
 ```
 
-Run the command the error names, then re-run `brew install codegraph`:
+Homebrew refuses because a third-party cask runs arbitrary Ruby on your
+machine at install time — this cask's post-install hook does — so `brew trust`
+is a security control you are opting out of; `--cask` limits that opt-out to
+this one cask. Trust this cask alone, then re-run `brew install codegraph`:
 
 ```sh
-brew trust --tap seanb4t/tap
+brew trust --cask seanb4t/tap/codegraph
 brew install codegraph
 ```
 
+Homebrew also offers a tap-wide grant that trusts every current and future
+cask and command in the tap; this project does not recommend it.
+
 **Upgrading.** A brew-managed install is upgraded with `brew upgrade
 codegraph`, not `codegraph upgrade`. `codegraph upgrade` detects a
 brew-managed install by resolving the running binary through symlinks and
@@ -578,11 +584,11 @@ longer on disk.
 > of any cask from a newly-tapped, non-official tap is refused
 > (`Error: Refusing to load cask seanb4t/tap/codegraph from untrusted tap
 > seanb4t/tap.`) until the tap is explicitly trusted
-> (`brew trust --tap seanb4t/tap` or `brew trust --cask
-> seanb4t/tap/codegraph`) — a real, general Homebrew mechanism, not a defect
-> in this cask or tap, but one this document did not previously mention. If
-> `brew install codegraph` refuses with that message, run the `brew trust`
-> command it names, then re-run `brew install codegraph`.
+> (`brew trust --cask seanb4t/tap/codegraph` — Homebrew's error also offers a
+> tap-wide form) — a real, general Homebrew mechanism, not a defect in this
+> cask or tap, but one this document did not previously mention. If `brew
+> install codegraph` refuses with that message, run `brew trust --cask
+> seanb4t/tap/codegraph`, then re-run `brew install codegraph`.
 >
 > **One release only was cut for this verification, not two.** GoReleaser's
 > tap-push *update* path (writing a second commit to an already-existing
```

Diff shape: +14/-8 lines, both hunks confined to the untrusted-tap paragraph and the BREW-01 blockquote. Nothing else in the file changed (headings and `03-EVIDENCE.md` reference counts confirmed unchanged against `ca015c4b`).

### `git diff ca015c4b -- README.md`

```diff
diff --git a/README.md b/README.md
index 86326710..8986b79b 100644
--- a/README.md
+++ b/README.md
@@ -80,7 +80,9 @@ upgrades work under a brew-managed install. Upgrade with `brew upgrade codegraph
 not `codegraph upgrade` — running `codegraph upgrade` against a brew-managed
 install refuses with a pointer to that command and exits non-zero, rather
 than mutating the install behind Homebrew's bookkeeping; `codegraph upgrade
---check` reports the same pointer and exits zero.
+--check` reports the same pointer and exits zero. The full command and
+flag reference, generated from the binary's own command tree, is
+[`docs/CLI-REFERENCE.md`](docs/CLI-REFERENCE.md).
 
 ## Quick start
 
```

Diff shape: +3/-1 lines, one sentence appended to the existing Homebrew paragraph. Headings and the `## Commands` table are byte-identical to `ca015c4b`.

## Todo-Tool Invocation and Output

```
$ node "$HOME/.claude/gsd-core/bin/gsd-tools.cjs" todo complete 2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md --dry-run
{
  "dry_run": true,
  "would_complete": true,
  "file": "2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md",
  "date": "2026-09-13",
  "would_move": {
    "source": ".planning/todos/pending/2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md",
    "target": ".planning/todos/completed/2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md"
  },
  "would_set": {
    "completed": "2026-09-13",
    "status": "completed"
  }
}

$ node "$HOME/.claude/gsd-core/bin/gsd-tools.cjs" todo complete 2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md
{
  "completed": true,
  "file": "2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md",
  "date": "2026-09-13"
}
```

No `git mv` fallback was needed — the tool ran successfully and stamped `completed: 2026-09-13` / `status: completed` in the moved file's frontmatter, confirmed by reading the completed file after the run.

## Decisions Made

- **Compressed the framing-sentence prose** rather than using the plan's suggested wording verbatim (which included a parenthetical: "it generates man pages and checks the installed version"). The full-detail version pushed `docs/RELEASE.md`'s `git diff --numstat` added-line count to 17, exceeding the task's own `<verify>` gate ceiling of 14. The compressed sentence keeps every clause the must-haves and acceptance criteria actually require — arbitrary Ruby at install time, `brew trust` as the opted-out-of security control, `--cask` scoping the opt-out to one cask — and drops only the supplementary grounding detail about what the post-install hook does. Net diff: 14 added / 8 removed lines, within budget.
- **BREW-01 blockquote rewrap** kept at exactly 5 lines (same as the original) by tightening word-wrap width slightly, so the total file diff stayed within the numeric ceiling asserted by the task's own verify gate.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] Framing-sentence wording exceeded the task's own diff-size verify gate**
- **Found during:** Task 1, first verification pass
- **Issue:** The plan's `<action>` text for the framing sentence (including "it generates man pages and checks the installed version") produced a `docs/RELEASE.md` diff of +17/-8 lines against `ca015c4b`, exceeding the task's own `<verify>` assertion of `git diff --numstat` added lines `<= 14`
- **Fix:** Tightened the framing-sentence prose (dropped the supplementary parenthetical, kept all required substantive clauses) and re-wrapped the BREW-01 blockquote at a slightly wider column, bringing the diff to +14/-8
- **Files modified:** `docs/RELEASE.md`
- **Verification:** Re-ran the full task 1 `<verify>` battery (10 assertions in the first automated block, 2 numstat/heading/evidence assertions in the second) — all passed after the edit
- **Committed in:** `910d5e61`

---

**Total deviations:** 1 auto-fixed (Rule 3 — blocking issue, the plan's own numeric gate).
**Impact on plan:** No semantic content was lost that the must-haves or acceptance criteria require; only an optional grounding detail (the specific mechanism of the post-install hook) was trimmed to fit the plan's own asserted line budget. All required substrings (`arbitrary Ruby`, `opting out`, `tap-wide` x2, narrow-form command x3+) are present and verified.

## Issues Encountered

None beyond the diff-size deviation documented above.

## User Setup Required

None — no external service configuration required.

## Requirements Note

`requirements-completed` is intentionally empty in this SUMMARY's frontmatter. DOCS-05 is declared by all three plans in this phase (12-01, 12-02, 12-03) and DOCS-07 is declared by 12-02 and 12-03 — per the shared-ID gate (#2388), neither requirement should read `Complete` in REQUIREMENTS.md until every declaring plan has finished. Ran `gsd_run query requirements.ready-ids .planning/phases/12-cli-reference-docs-tail/12-02-PLAN.md DOCS-07 DOCS-05 --raw` and confirmed `0/2 requirement(s) ready to mark complete` (12-03 has not landed yet). This plan's own contribution to both requirements is fully verified above; `requirements.mark-complete` is deferred to whichever plan finishes last (12-03).

## Scope Note (per this plan's own dispatch instructions)

Per explicit objective-level scoping for this plan's execution, this SUMMARY does NOT trigger `roadmap.update-plan-progress`, `state.advance-plan`, `state.update-progress`, or any further `.planning/STATE.md` edit beyond the todo-row move already committed in Task 1 (per the plan's own D-12 instruction). `.planning/ROADMAP.md` is untouched by this plan.

## Next Phase Readiness

- `docs/RELEASE.md` and `README.md` now carry the DOCS-07/D-06 wording this phase set out to land; the 2026-08-10 todo is closed.
- Ready for 12-03 (MUTATION-LOG RED families, SECURITY.md, VALIDATION.md, and the final `requirements.mark-complete` pass for DOCS-05/06/07 once it lands its own SUMMARY).
- No blockers.

---
*Phase: 12-cli-reference-docs-tail*
*Completed: 2026-09-13*

## Self-Check: PASSED

All 4 key files confirmed present on disk (`docs/RELEASE.md`, `README.md`, `.planning/STATE.md`, `.planning/todos/completed/2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md`). Both task commits (`910d5e61`, `b9d75ecc`) confirmed present in `git log`. `commits: 2` measured via `git rev-list --count 81ca6a71..HEAD` against the plan-head ledger (`81ca6a71`, the commit immediately preceding this plan's first task), matching the two task commits with no code changes left uncommitted.
