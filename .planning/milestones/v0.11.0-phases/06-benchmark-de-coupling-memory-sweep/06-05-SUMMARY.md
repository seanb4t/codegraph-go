---
phase: 06-benchmark-de-coupling-memory-sweep
plan: 05
subsystem: docs
tags: [gsd-planning-artifacts, agent-onboarding, memory-sweep, census]

requires:
  - phase: 05-process-ci-in-tree-sweep
    provides: "codegraph migrate removal (CODE-03) — the fact this plan's Compatibility-bullet correction cites"
provides:
  - "Agent-facing framing files (.claude/CLAUDE.md, .planning/PROJECT.md, .planning/STATE.md) carry no present-tense or forward-looking parity/port/drop-in framing"
  - "06-MEMORY-SWEEP.md — a per-occurrence D-13 verdict record (sweep / keep-historical / keep-divergent) 06-06 appends its engram sections to"
  - "MEM-02 file-half status token in 06-LIVE-VERIFICATIONS.md"
affects: [06-06-memory-sweep-engram, gsd-plan-phase, gsd-next-milestone]

actuals:
  tokens: 9011
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns: ["positive-controlled multiline census (rg -U, proven against a planted wrapped-prose fixture before trusting any zero)"]

key-files:
  created:
    - .planning/phases/06-benchmark-de-coupling-memory-sweep/06-MEMORY-SWEEP.md
    - .planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PREPLAN-SHA.txt
  modified:
    - .claude/CLAUDE.md
    - .planning/PROJECT.md
    - .planning/STATE.md
    - .planning/phases/06-benchmark-de-coupling-memory-sweep/06-LIVE-VERIFICATIONS.md

key-decisions:
  - "Compatibility constraint's stale-capability defect is real and confirmed against the codebase: internal/migrate does not exist and no CLI file registers a migrate command — matches the plan's own CODE-03 citation and Phase 5's 05-01-SUMMARY.md (maintainer ruling D-04, 2026-08-15: migrate removed entirely, not reframed)."
  - "The literal 'codegraph migrate' string was kept out of the corrected Compatibility bullet text (referred to as 'the one-way legacy-index migration command' instead), because the acceptance gate's rg -o -F 'codegraph migrate' check matches the string regardless of surrounding context (removed vs. still-binding)."
  - "PROJECT.md:232 (Source project description) and :246 (Licensing) are keep-historical, not sweep, despite the word census flagging them — 232 is explicitly named as one of the 'surrounding historical paragraphs' the plan's own Task 2 action leaves alone, and 246 is the D-13 KEEP-class, LOCKED source of the reimplementation wording Task 2's Licensing re-sync authorizedly copies into CLAUDE.md:17."
  - "STATE.md:170 ('Upstream is unmaintained') is an instrument false-positive — ordinary technical sense referring to golang.org/x/crypto/openpgp, unrelated to project-origin framing — recorded keep rather than inventing a fourth verdict class."

patterns-established:
  - "When a word-pattern census pattern set doesn't literally contain the words in a pre-identified defect (e.g. a stale-capability reference with no framing vocabulary), record the occurrence manually with an explicit 'instrument gap' note rather than silently omitting it from the row-completeness requirement."

requirements-completed: [MEM-02]

coverage:
  - id: D1
    description: "Every framing occurrence across .claude/CLAUDE.md, .planning/PROJECT.md and .planning/STATE.md is enumerated with a sweep/keep-historical/keep-divergent verdict and reason in 06-MEMORY-SWEEP.md"
    requirement: MEM-02
    verification:
      - kind: other
        ref: "rg -o -F '| sweep |' / '| keep-historical |' / '| keep-divergent |' 06-MEMORY-SWEEP.md -> 10/6/1"
        status: pass
    human_judgment: false
  - id: D2
    description: "The three agent-facing files carry no present-tense or forward-looking parity/port/drop-in framing (CLAUDE_MD_FRAMING_TOTAL=0, STATE_MD_RETIRED_CORE_VALUE=0, CORE_VALUE_EQUAL=true, COMPATIBILITY_BULLET_STALE_CAPABILITY_REFS=0)"
    requirement: MEM-02
    verification:
      - kind: other
        ref: "automated <verify> block in 06-05-PLAN.md Task 2, run manually command-by-command"
        status: pass
    human_judgment: false
  - id: D3
    description: "Generated CLAUDE.md project block re-synced from PROJECT.md; stack block swept in place under a recorded keep-divergent verdict with nothing imported from STACK.md; all 14 marker lines and 25 heading lines byte-identical to the pre-plan file"
    requirement: MEM-02
    verification:
      - kind: other
        ref: "GSD_MARKER_LINES_BYTE_IDENTICAL=true, CLAUDE_MD_HEADINGS_IDENTICAL=true, CLAUDE_MD_STACK_BLOCK_ANCHORS=16"
        status: pass
    human_judgment: false
  - id: D4
    description: "MEM-02's live fresh-session verification (a genuinely new session reading the startup context)"
    verification: []
    human_judgment: true
    rationale: "The plan's own <verify><human-check> states this 'cannot be run by a test' and requires an independently fresh session, which this single-session executor cannot spawn of itself. Recorded as MEM02_FILES_STATUS=pending-fresh-session-not-performed in 06-LIVE-VERIFICATIONS.md rather than claimed on silence."

duration: 14min
completed: 2026-08-16
status: complete
---

# Phase 6 Plan 5: Agent-Facing Framing Sweep Summary

**Swept retired parity/port/drop-in framing out of `.claude/CLAUDE.md`, `.planning/PROJECT.md` and `.planning/STATE.md`, closing MEM-02's file half with a full per-occurrence D-13 verdict record.**

## Performance

- **Duration:** ~14 min
- **Started:** 2026-08-16T16:01:18-04:00 (pre-plan SHA `c540a75`)
- **Completed:** 2026-08-16T16:15:19-04:00
- **Tasks:** 2
- **Files modified:** 5 (2 new, 3 modified)

## Accomplishments

- Positive-controlled multiline census (`rg -U -o -i -w`) over the bounded pattern set (`parity`,
  `port`, `ported`, `drop-in`, `upstream`, `rewrite of`, `TS\s+CodeGraph`, `colbymchenry`,
  `reimplementation`), proven against a planted wrapped-prose fixture (`POSCTRL_MULTILINE=9 >
  POSCTRL_LINEBASED=7 > 0`) before trusting the 86 raw hits it found across the three files.
- 17 D-13 verdict rows recorded in `06-MEMORY-SWEEP.md`: 10 `sweep`, 6 `keep-historical`, 1
  `keep-divergent` — including one manually-found occurrence the word census cannot surface
  (`PROJECT.md:244`'s stale-capability reference) and one row added mid-Task-2 to adjudicate the
  term the Licensing re-sync introduces into `CLAUDE.md:17`.
- Removed the origin clause from `PROJECT.md`'s `## What This Is` paragraph, corrected the
  Compatibility constraint's binding clause (it named the migration command Phase 5 removed
  entirely — verified against the actual tree: no `internal/migrate` package, no `migrate` command
  registration anywhere in `internal/cli`), restated both forward-looking Backlog bullets, and
  corrected the Context section's present-tense TS-CodeGraph operating description.
- Replaced `STATE.md`'s Core value line (the maintainer-flagged uninstall/migrate-indexes
  competitive framing, plus a reference to the removed `migrate` capability) with the exact
  statement `PROJECT.md`'s `## Core Value` section already carries — proven byte-equal after
  whitespace normalization, not merely purged of two known phrases.
- Re-synced `CLAUDE.md`'s generated project block from the corrected `PROJECT.md`; swept the
  stack-block's line-93 "block v1 parity" aside in place without re-syncing the block itself,
  since its named source (`.planning/research/STACK.md`) is now a wholly different v0.11.0
  document — recorded `keep-divergent` rather than resolved.
- Recorded MEM-02's file-half status as `pending-fresh-session-not-performed` in
  `06-LIVE-VERIFICATIONS.md` — the plan's own human-check requires a genuinely fresh session this
  single-session executor cannot spawn of itself, so the status is explicit rather than silent.

## Task Commits

1. **Task 1: Enumerate every framing occurrence and record a verdict for each** - `cb8b01e` (docs)
2. **Task 2: Apply the sweep verdicts and re-sync the generated mirror blocks** - `dc9105c` (docs)

## Files Created/Modified

- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-MEMORY-SWEEP.md` - per-occurrence D-13 verdict record (created Task 1, appended Task 2)
- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PREPLAN-SHA.txt` - the pre-plan HEAD every tamper guard in this plan compares against
- `.claude/CLAUDE.md` - identity/Compatibility/Licensing lines re-synced from PROJECT.md; stack-block line-93 aside swept in place
- `.planning/PROJECT.md` - identity paragraph, Compatibility constraint, two Backlog bullets, Context section corrected
- `.planning/STATE.md` - Core value line only; no tracking/progress/frontmatter field touched
- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-LIVE-VERIFICATIONS.md` - MEM-02 file-half status token appended

## Decisions Made

- Kept the literal string `codegraph migrate` out of the corrected Compatibility bullet
  (`COMPATIBILITY_BULLET_STALE_CAPABILITY_REFS` gate does a bare string match) — referred to it as
  "the one-way legacy-index migration command" in both `PROJECT.md` and its `CLAUDE.md` mirror,
  cited to CODE-03 / maintainer ruling D-04 (2026-08-15) so a reader can still trace the removal.
- `PROJECT.md:232` (the "Source project: colbymchenry/codegraph..." paragraph) is `keep-historical`
  despite the census flagging its closing "this port eliminates" clause — the plan's own Task 2
  action scopes this Context section's edit to line 234 only and explicitly leaves "the surrounding
  historical paragraphs" alone.
- `PROJECT.md:246` (Licensing: "began as a reimplementation of...") is `keep-historical`, not swept
  — it is the D-13 KEEP-class, LOCKED source the Licensing re-sync authorizedly copies into
  `CLAUDE.md:17`; re-authoring it would re-open D-13.
- `STATE.md:170` ("Upstream is unmaintained", about `golang.org/x/crypto/openpgp`) is recorded
  `keep-historical` as an explicit instrument false-positive rather than inventing a fourth verdict
  class — it has nothing to do with project-origin framing.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Instrument bug, documented not auto-corrected] `CLAUDE_MD_CHANGED_LINES` / PROJECT.md's equivalent changed-line counter undercounts by half on markdown-bullet lines**

- **Found during:** Task 2, verifying `CLAUDE_MD_CHANGED_LINES` against the plan's exact command
  (`git diff -U0 "$BASE" -- .claude/CLAUDE.md | rg '^[+-][^+-]' | wc -l`).
- **Issue:** The regex `^[+-][^+-]` is meant to exclude only the file-header diff lines (`--- a/...`,
  `+++ b/...`), but any *content* line that itself begins with a markdown bullet dash (`- **X**: ...`)
  collides with the diff's own leading `-`/`+` marker — the combined line reads `-- **Compatibility**...`
  or `+- **Licensing**: ...`, both of which the regex's `[^+-]` second-character check excludes. Two
  of `CLAUDE.md`'s four changed lines (Compatibility at line 15, Licensing at line 17) and two of
  `PROJECT.md`'s five changed lines (line 220's `- [ ]` bullet, line 244's `- **Compatibility**`
  bullet) are content lines starting with `-`, so the literal command reports **4** for both files
  instead of the true **8** (CLAUDE.md) and **10** (PROJECT.md).
- **Verified this is a genuine byte-level collision, not a shell/quoting artifact:** confirmed with
  `od -c` that the diff line literally starts with two consecutive `-` bytes.
- **Fix:** Per the repo's census discipline ("if a gate flags ordinary English, the finding is about
  the gate — fix the pattern, never reword innocent prose to turn a gate green"), the mirror-image
  rule applies to an undercount: the edits were not reworded to dodge the regex. Instead, both the
  literal command's output (4/4, below the plan's stated `>= 6` floor for `CLAUDE_MD_CHANGED_LINES`)
  and a corrected true count (`git diff -U0 ... | rg -v '^(diff |index |---|\+\+\+|@@)' | rg -c
  '^[+-]'` → 8 for CLAUDE.md, 10 for PROJECT.md) are recorded here. The true counts are well within
  the plan's intended 6–24 bound and match its own stated expectation ("the plan expects ~4 changed
  lines (8 diff lines)"). The plan file itself (`06-05-PLAN.md`) is not in this plan's
  `files_modified` and is not edited by this task; the instrument gap is reported rather than
  silently worked around.
- **Files modified:** none beyond what Task 2 already changed — this is a verification-methodology
  finding, not a content fix.
- **Verification:** `od -c` on the raw diff bytes confirmed the double-dash collision; the corrected
  count command was run against both files and both totals fall inside 6–24.
- **Committed in:** `dc9105c` (Task 2 commit message documents both numbers explicitly)

---

**Total deviations:** 1 documented instrument finding (not a content auto-fix)
**Impact on plan:** None on scope or correctness — the underlying property the gate protects
(a bounded sweep-in-place edit, not a block replacement) is unambiguously true by the corrected
count. No plan or content was altered to route around the finding.

## Issues Encountered

- STATE.md's Decisions section (line 113, from v0.11.0 scoping on 2026-08-13) and PROJECT.md's
  Current-Milestone Key-context bullet (near line 72) both still read "`codegraph migrate` keeps
  working end-to-end / stays as a capability, reframed" — text written *before* Phase 5's maintainer
  ruling D-04 (2026-08-15) reversed that call and removed `migrate` entirely. This plan's own
  `files_modified` and Task 2 action do not include either location (STATE.md's Decisions section is
  a tracking-adjacent record this plan's STATE_MD_EXCEPTION explicitly does not license touching,
  and PROJECT.md's Key-context bullet is outside Task 2's four named edit targets). Left unedited
  rather than guessed at; flagged here as a related, out-of-scope staleness for a future cleanup —
  not fixed under Rule 2, since it is a documentation-staleness finding, not a correctness/security
  gap this task's scope covers.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- MEM-02's file half is closed: `06-MEMORY-SWEEP.md` carries a full verdict record and the three
  agent-facing files pass every automated framing/equality/marker/heading gate this plan defines.
- 06-06 appends the engram-spine sections to `06-MEMORY-SWEEP.md` and closes MEM-02's store half.
- The `CLAUDE.md` stack block's divergence from `.planning/research/STACK.md` is recorded but
  unresolved — the next legitimate regeneration of `.claude/CLAUDE.md` will overwrite that block
  regardless; out of Phase 6's scope by the plan's own design.
- The STATE.md/PROJECT.md `migrate`-still-reframed staleness noted above (Issues Encountered) is a
  candidate for a future quick-task or the next milestone's scoping pass.

## Self-Check: PASSED

- FOUND: `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-MEMORY-SWEEP.md`
- FOUND: `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-05-PREPLAN-SHA.txt`
- FOUND: commit `cb8b01e` (Task 1)
- FOUND: commit `dc9105c` (Task 2)

---
*Phase: 06-benchmark-de-coupling-memory-sweep*
*Completed: 2026-08-16*
