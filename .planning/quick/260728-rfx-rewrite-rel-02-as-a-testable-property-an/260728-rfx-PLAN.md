---
phase: quick-260728-rfx
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - .planning/REQUIREMENTS.md
  - .planning/ROADMAP.md
  - .planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-VERIFICATION.md
  - .planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-UAT.md
  - .planning/STATE.md
  - .planning/PROJECT.md
autonomous: true
requirements: [REL-02]
must_haves:
  truths:
    - "REQUIREMENTS.md's REL-02 states a testable property of the release automation (release-please owns bump/CHANGELOG/tag; cosign SAN still satisfies verify.go), not a one-time event (ITEM-1)"
    - "REQUIREMENTS.md's requirement-to-phase mapping assigns REL-02 to Phase 9 (ITEM-2)"
    - "ROADMAP.md Phase 8 no longer lists or claims REL-02 in its Requirements line, success criteria, or Notes (ITEM-3)"
    - "ROADMAP.md Phase 9 Requirements is REL-02, not TBD (ITEM-4)"
    - "08-VERIFICATION.md records the REL-02 must-have as out of scope / moved to Phase 9 — neither failed nor passed — with its status frontmatter untouched (ITEM-5)"
    - "08-UAT.md Test 1 carries a note that its requirement is no longer Phase 8's, with result: blocked, history, and scope_note intact (ITEM-6)"
    - "STATE.md and PROJECT.md no longer assert Phase 8 ownership of REL-02 or a maintainer-manual tag as the pending final action (ITEM-7)"
  artifacts:
    - .planning/REQUIREMENTS.md
    - .planning/ROADMAP.md
    - .planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-VERIFICATION.md
    - .planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-UAT.md
    - .planning/STATE.md
    - .planning/PROJECT.md
  key_links:
    - "REQUIREMENTS.md REL-02 text <-> ROADMAP.md Phase 9 Requirements line <-> Phase 9 Success Criteria 1/2 (they already describe the same property)"
    - "REQUIREMENTS.md mapping table row <-> ROADMAP.md Phase 8 Requirements line (must not both claim REL-02)"
    - "ROADMAP.md Phase 8 heading text <-> phase list entry (~line 44) <-> Progress table row (~line 397) <-> on-disk phase dir name 08-surface-reconciliation-signed-v1-0-0-release <-> STATE.md current_phase_name (all five must stay byte-consistent)"
---

<objective>
Reclassify REL-02 from an unsatisfiable event ("a signed v1.0.0 release is cut") into a testable
property of the release automation, and move its ownership from Phase 8 to Phase 9.

Purpose: REL-02 is the sole item holding Phase 8 open. Its testable content (per-binary cosign +
SLSA + SBOM) is already Complete and PROVEN under v0.1's DIST-02 on real release v0.0.0-rc.3; what
remained was only a version-string labeling judgment. That category error is why it could not be
automated (08-VALIDATION.md records it manual-only, the reason nyquist_compliant is false) and why
it blocked UAT twice. Phase 9 (release-please + GoReleaser) already owns the mechanism, and its
Success Criteria 1 and 2 already describe exactly the property REL-02 is being rewritten to state.

Output: six planning documents edited in place. No source code, no tests, no build steps.

Scope discipline: this plan is documentation/planning-state only. It edits LIVING planning state
(REQUIREMENTS, ROADMAP, VERIFICATION, UAT, STATE, PROJECT). It MUST NOT edit HISTORICAL records —
see <forbidden_files>. This project has already been bitten twice by retconning history (T-08-09-04
caught PROJECT.md claiming "v1.0.0 shipped" with no tag existing).
</objective>

<execution_context>
@$HOME/.claude/gsd-core/workflows/execute-plan.md
@$HOME/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/REQUIREMENTS.md
@.planning/ROADMAP.md
@.planning/STATE.md
@.planning/PROJECT.md
@.planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-VERIFICATION.md
@.planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-UAT.md
</context>

<forbidden_files>
Do NOT read-for-edit, edit, or "reconcile" any of the following. They are historical records of what
was planned and done at the time; editing them to match a later decision falsifies the audit trail.

- .planning/phases/08-*/08-*-PLAN.md   (any plan file)
- .planning/phases/08-*/08-*-SUMMARY.md (any summary file)
- .planning/phases/08-*/08-RESEARCH.md
- .planning/phases/08-*/08-CONTEXT.md
- .planning/phases/08-*/08-DISCUSSION-LOG.md
- .planning/phases/08-*/08-SECURITY.md
- .planning/phases/08-*/08-VALIDATION.md
- .planning/phases/08-*/08-PATTERNS.md
- .planning/phases/06-*/06-CONTEXT.md, .planning/phases/06-*/06-DISCUSSION-LOG.md
- .planning/phases/07-*/07-CONTEXT.md, .planning/phases/07-*/07-DISCUSSION-LOG.md
- .planning/milestones/**  (the v0.1 archive, including DIST-02)
- .planning/v1.0-MILESTONE-AUDIT.md

Several of these contain REL-02 references. Leaving them stale is CORRECT and intended.
</forbidden_files>

<decision_record>
## Phase 8 heading: KEEP "& Signed v1.0.0 Release" (do not retitle)

Decided during planning, per ITEM-3's "DECIDE AND JUSTIFY" instruction. Rationale:

1. **The title is load-bearing on disk.** The phase directory is
   `.planning/phases/08-surface-reconciliation-signed-v1-0-0-release/`. That slug is baked into
   every committed artifact path, every SUMMARY, and every git commit message for the phase.
   Retitling the ROADMAP without renaming the directory creates a title/path mismatch that is worse
   than the status quo; renaming the directory would rewrite history and break every recorded path —
   forbidden by <forbidden_files>.
2. **It would ripple into records this plan may not touch.** `08-VERIFICATION.md`'s own report
   heading and the frozen 08-*-SUMMARY.md files all carry the title verbatim.
3. **The title is historically accurate.** Phase 8 did deliver release *readiness*: the runbook,
   the CGo/govulncheck/SBOM audit, per-binary cosign + SLSA verified end-to-end on v0.0.0-rc.3, and
   the reproducible double-build. What it did not deliver is the version-string label — which is
   precisely the non-requirement this plan is deleting.

The ownership change is therefore communicated in the Phase 8 **Notes**, not the heading (ITEM-3's
Notes amendment). Because the title is being kept, all three occurrence sites MUST remain byte-identical:

- `.planning/ROADMAP.md` ~line 44 — phase list entry `- [ ] **Phase 8: Surface Reconciliation & Signed v1.0.0 Release** - ...`
- `.planning/ROADMAP.md` ~line 298 — Phase Details heading `### Phase 8: Surface Reconciliation & Signed v1.0.0 Release`
- `.planning/ROADMAP.md` ~line 397 — Progress table row `| 8. Surface Reconciliation & Signed v1.0.0 Release | v1.0 | 9/9 | In Progress| |`

(A fourth consistency site, `.planning/STATE.md` `current_phase_name`, is also unchanged by this
decision — see Task 3.)
</decision_record>

<tasks>

<task type="auto">
  <name>Task 1: Rewrite REL-02 as a property and reassign it to Phase 9 in REQUIREMENTS.md</name>
  <files>.planning/REQUIREMENTS.md</files>
  <read_first>
    Line 89 (the REL-02 bullet inside the `### v1.0.0 Release (REL)` block, ~lines 86-91) and
    line 178 (the `| REL-02 | Phase 8 | Pending |` row of the requirement-to-phase mapping table,
    ~lines 177-180).
  </read_first>
  <action>
Per ITEM-1, replace the entire line-89 REL-02 bullet with EXACTLY this single line, verbatim,
preserving the leading unchecked checkbox:

- [ ] **REL-02**: releases are cut by release-please from Conventional Commits — version bump, `CHANGELOG.md`, and tag creation all happen without a human running `git tag` — and the resulting signed artifacts still satisfy `internal/upgrade/verify.go`'s cosign identity (`releaseWorkflowRefPattern`, SAN anchored to `release.yml@refs/tags/v[0-9]*`), so `codegraph upgrade` keeps working for already-shipped binaries

The replacement drops the trailing DIST-02 clause entirely; DIST-02 is already Complete and PROVEN
in the v0.1 milestone archive, so REL-02 must not re-claim it. Do not touch REL-01, REL-03, or
REL-04 (all `[x]`), and leave REL-02's own checkbox unchecked — the property is not yet true.

Per ITEM-2, in the requirement-to-phase mapping table change ONLY the REL-02 row's phase cell from
Phase 8 to Phase 9, leaving its status cell as Pending and leaving the REL-01/03/04 rows untouched.

Do not renumber, reorder, or reflow any other line in the file.
  </action>
  <verify>
    <automated>cd /Volumes/Code/github.com/seanb4t/codegraph-go && rg -q 'release-please from Conventional Commits' .planning/REQUIREMENTS.md && rg -q 'releaseWorkflowRefPattern' .planning/REQUIREMENTS.md && rg -q '^- \[ \] \*\*REL-02\*\*:' .planning/REQUIREMENTS.md && rg -q '^\| REL-02 \| Phase 9 \| Pending \|' .planning/REQUIREMENTS.md && [ "$(rg -c 'DIST-02' .planning/REQUIREMENTS.md || echo 0)" = "0" ] && [ "$(rg -c '^\| REL-0[134] \| Phase 8 \| Complete \|' .planning/REQUIREMENTS.md)" = "3" ] && echo PASS</automated>
  </verify>
  <done>
    REL-02 reads as a release-automation property citing release-please, `releaseWorkflowRefPattern`,
    and `codegraph upgrade`; it remains unchecked; no DIST-02 reference survives anywhere in
    REQUIREMENTS.md; the mapping table shows `| REL-02 | Phase 9 | Pending |`; REL-01/03/04 still map
    to Phase 8 / Complete.
  </done>
</task>

<task type="auto">
  <name>Task 2: Move REL-02 ownership from Phase 8 to Phase 9 in ROADMAP.md</name>
  <files>.planning/ROADMAP.md</files>
  <read_first>
    The `### Phase 8: Surface Reconciliation &amp; Signed v1.0.0 Release` block (~lines 298-343) —
    specifically its Requirements line (~302), Success Criteria item 4 (~308), Notes paragraph (~341),
    and the ⚠-prefixed dated 2026-07-27 paragraph immediately below Notes (~343). Then the
    `### Phase 9: release-please + GoReleaser` block's Requirements line (~349).
  </read_first>
  <action>
Apply ITEM-3 and ITEM-4 to `.planning/ROADMAP.md` with four scoped Edit calls. Use Edit, never Write —
a whole-file rewrite would destroy phase entries outside the diff window.

(a) Phase 8 Requirements line (~302): drop REL-02 so the line reads exactly
`**Requirements**: SURF-01, SURF-02, SURF-03, SURF-04, SURF-05, REL-01, REL-03, REL-04`.

(b) Phase 8 Success Criteria item 4 (~308): it currently covers two requirements in one sentence.
Rewrite it to cover the benchmark half only — head-to-head benchmarks vs TS 1.3.1 are re-run and
published, closing v0.1's pending PERF-01 — and end it with a single-requirement attribution of
`(REL-03)`. Delete the signed-release clause from this criterion entirely, including its
DIST-02-closing tail. Do not renumber criteria 1, 2, 3, or 5.

(c) Phase 8 Notes (~341) — two edits in one region:
    - Delete the entire ⚠-prefixed dated 2026-07-27 paragraph that sits immediately below the Notes
      paragraph. It described this change as not-yet-applied; it is now applied and therefore stale.
    - Amend the Notes paragraph's trailing sentence so it no longer frames the signed tag as this
      phase's manual final action. Keep the D-01 SURF-green-before-REL clause. Replace the tail with
      a statement that release automation — and the rewritten REL-02 with it — now belongs to
      Phase 9. The amended Notes paragraph must mention the string `Phase 9`, and must mention
      `REL-02` exactly once and on that single line (the completeness gate below counts on this).

(d) Phase 9 Requirements line (~349): replace the TBD placeholder so the line reads exactly
`**Requirements**: REL-02`.

Per <decision_record>, DO NOT retitle Phase 8. Leave the heading (~298), the phase list entry (~44),
and the Progress table row (~397) byte-identical. Also leave the Plans checklist line for
`08-09-PLAN.md` (~339) untouched — it is a historical record of what that plan was scoped to deliver.
Do not touch Phase 9's Goal, Success Criteria, or Notes; its Success Criteria 1 and 2 already state
the property REL-02 now carries, which is why no rewrite is needed there.
  </action>
  <reversibility rating="reversible">Planning-doc text under git; a single `git revert` restores prior ownership wording.</reversibility>
  <verify>
    <automated>cd /Volumes/Code/github.com/seanb4t/codegraph-go && rg -q '^\*\*Requirements\*\*: SURF-01, SURF-02, SURF-03, SURF-04, SURF-05, REL-01, REL-03, REL-04$' .planning/ROADMAP.md && rg -q '^\*\*Requirements\*\*: REL-02$' .planning/ROADMAP.md && rg -q 'PERF-01 \(REL-03\)' .planning/ROADMAP.md && [ "$(rg -c 'REL-02/03' .planning/ROADMAP.md || echo 0)" = "0" ] && [ "$(rg -c 'rescope pending' .planning/ROADMAP.md || echo 0)" = "0" ] && [ "$(rg -c 'REL-02' .planning/ROADMAP.md)" = "3" ] && [ "$(rg -c 'Surface Reconciliation & Signed v1\.0\.0 Release' .planning/ROADMAP.md)" = "3" ] && rg -q '^### Phase 8: Surface Reconciliation & Signed v1\.0\.0 Release$' .planning/ROADMAP.md && echo PASS</automated>
  </verify>
  <done>
    Phase 8's Requirements line lists 8 IDs without REL-02; criterion 4 attributes to REL-03 alone;
    the stale dated warning paragraph is gone; the Notes paragraph points at Phase 9; Phase 9's
    Requirements is REL-02. Exactly 3 REL-02 occurrences remain in ROADMAP.md (the historical
    08-09 plan checklist line, the amended Phase 8 Notes line, and the Phase 9 Requirements line),
    and the Phase 8 title still appears at exactly its 3 original sites.
  </done>
</task>

<task type="auto">
  <name>Task 3: Record REL-02 as out of scope in the Phase 8 artifacts, and correct STATE/PROJECT ownership claims</name>
  <files>.planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-VERIFICATION.md, .planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-UAT.md, .planning/STATE.md, .planning/PROJECT.md</files>
  <read_first>
    08-VERIFICATION.md: frontmatter `requirements_coverage.REL-02` (~line 15), the second
    `human_verification` entry (~lines 28-31), must-have row 8 in the truths table (~line 53), the
    Score line (~56), the evidence-table row (~96), the coverage-table row (~112), the
    `#### 2. Signed v1.0.0 release cut ...` human-verification section (~lines 130-134), and the
    no-blocking-gaps paragraph (~lines 138-140).
    08-UAT.md: Test 1 (~lines 24-90), including `result: blocked`, `note:`, and `scope_note:`.
    STATE.md: lines 11, 33, and the Deferred Items table row for DIST-02 (~line 251).
    PROJECT.md: line 13 (Goal) and line 72 (Repo state).
  </read_first>
  <action>
**08-VERIFICATION.md (ITEM-5).** Record REL-02 as out of scope — NOT failed, and under no
circumstances fabricated as passed. At each of the REL-02 sites listed in read_first, change the
disposition marker to carry the exact literal `OUT OF SCOPE (moved to Phase 9)` and append a one-clause
reason: the requirement was rewritten as a release-automation property and reassigned to Phase 9;
its testable content was already proven under v0.1's DIST-02 on release v0.0.0-rc.3. Update the
frontmatter `requirements_coverage` REL-02 value to `out_of_scope (moved to Phase 9)`. Remove the
REL-02 tag-push entry from the `human_verification` list — a Phase 8 human-verification item for a
requirement Phase 8 no longer owns is a live instruction to do the wrong thing. Leave the Affected()
BFS ordering item (4b) and every other human_verification / behavior_unverified entry exactly as-is.

CRITICAL: leave the top-level `status: human_needed` frontmatter value UNCHANGED. A separate
`/gsd-verify-work 8` run closes the phase; this plan does not close it. Do not adjust the Score line's
denominator arithmetic beyond noting the REL-02 item is now out of scope rather than deferred.

**08-UAT.md (ITEM-6).** Under Test 1, add a note containing the exact literal
`Ownership moved to Phase 9` recording that the requirement this test exercises is no longer owned by
Phase 8 — REL-02 was rewritten as a release-automation property (release-please owns bump/CHANGELOG/tag)
and reassigned to Phase 9, so this test is not a Phase 8 obligation and must not be re-run as written.
KEEP `result: blocked`. KEEP the existing `note:` block with both block records and the readiness
recheck. KEEP `scope_note:` verbatim. KEEP the file's `status: partial` frontmatter. Do not delete or
rewrite any existing history — append only.

**STATE.md (ITEM-7).** Two live status assertions and one mapping row assert Phase 8 ownership; edit
exactly these three and no others:
  - line 11 `last_activity_desc` — remove the trailing claim that Phase 08 is still awaiting the
    maintainer tag; replace with a note that REL-02 was rewritten and reassigned to Phase 9.
  - line 33 `Status:` — replace the blocked-on-maintainer-tag clause with: phase built and verified;
    REL-02 reassigned to Phase 9; awaiting a `/gsd-verify-work 8` close.
  - Deferred Items table (~251) — change the DIST-02 row's Status cell to the exact literal
    `Scoped into REL-02 (Phase 9)`.
Leave UNCHANGED: `current_phase_name` (line 6) and `Current focus` (line 27), per <decision_record>;
`stopped_at` (line 8) and `Session Continuity > Stopped at` (line 257), which are session history;
and the Accumulated Context decision-log entry at line 226, which is an accurate record of what was
decided at the time.

**PROJECT.md (ITEM-7).** Two sites assert the superseded Phase-8 manual-tag mechanism:
  - line 13 (Goal) — the parenthetical attributing the uncut tag to a pending maintainer go-ahead.
  - line 72 (Repo state) — the sentence describing the signed v1.0.0 release as a maintainer-manual
    action that has not been cut.
Amend both minimally: keep the factually-true statement that no `v1.0.0` tag exists (this is the exact
claim T-08-09-04 was created to protect), and replace only the mechanism attribution with the exact
literal `REL-02, now Phase 9` plus a short clause that releases are moving to release-please
automation. Do NOT assert that the release is done, imminent, or scheduled. Do not touch the
v0.0.0-rc.3 signing evidence, the benchmark numbers, or the Key Decisions table.

**Report (ITEM-7).** In the SUMMARY, list every REL-02 occurrence found in STATE.md and PROJECT.md
with its line number and the disposition applied (edited / left as history / left as title), so the
audit trail shows the check was exhaustive rather than opportunistic.
  </action>
  <verify>
    <automated>cd /Volumes/Code/github.com/seanb4t/codegraph-go && V=.planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-VERIFICATION.md && U=.planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-UAT.md && rg -q 'OUT OF SCOPE \(moved to Phase 9\)' "$V" && rg -q '^status: human_needed$' "$V" && rg -q 'REL-02: out_of_scope \(moved to Phase 9\)' "$V" && [ "$(rg -c 'executor-prohibited maintainer action' "$V" || echo 0)" = "0" ] && rg -q 'Ownership moved to Phase 9' "$U" && rg -q '^result: blocked$' "$U" && rg -q '^scope_note: \|$' "$U" && rg -q '^status: partial$' "$U" && rg -q 'BLOCKED TWICE' "$U" && rg -q 'Scoped into REL-02 \(Phase 9\)' .planning/STATE.md && rg -q '^current_phase_name: Surface Reconciliation & Signed v1\.0\.0 Release$' .planning/STATE.md && rg -q 'REL-02, now Phase 9' .planning/PROJECT.md && [ "$(rg -c 'REL-02' .planning/PROJECT.md)" = "2" ] && ! git -C . diff --name-only | rg -q '08-(RESEARCH|CONTEXT|DISCUSSION-LOG|SECURITY|VALIDATION|PATTERNS)\.md|08-[0-9]+-(PLAN|SUMMARY)\.md|milestones/|v1\.0-MILESTONE-AUDIT\.md' && echo PASS</automated>
  </verify>
  <done>
    08-VERIFICATION.md marks REL-02 out of scope at every site with `status: human_needed` intact and
    the REL-02 tag-push human_verification entry removed; 08-UAT.md Test 1 carries the Phase 9
    ownership note with `result: blocked`, `status: partial`, its two-block history, and `scope_note`
    all intact; STATE.md's DIST-02 row points at Phase 9 while `current_phase_name` is unchanged;
    PROJECT.md attributes REL-02 to Phase 9 without claiming any release happened; and `git diff`
    touches zero files from &lt;forbidden_files&gt;.
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| planning docs -> release verification chain | REQUIREMENTS/ROADMAP text is the human-facing statement of what the release gate requires; weakening it silently weakens the gate |
| planning docs -> audit trail | VERIFICATION/UAT/SUMMARY files are the evidentiary record; editing history rewrites what a reviewer believes happened |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-RFX-01 | Repudiation | 08-*-PLAN.md / 08-*-SUMMARY.md / milestones/** | high | mitigate | Explicit &lt;forbidden_files&gt; list; Task 3's verify gate fails if `git diff --name-only` names any forbidden path |
| T-RFX-02 | Tampering | 08-VERIFICATION.md REL-02 disposition | high | mitigate | Task 3 forbids marking REL-02 passed; gate requires the literal out-of-scope marker AND `status: human_needed` unchanged, so no path fabricates a phase close |
| T-RFX-03 | Information disclosure (false assurance) | PROJECT.md release claims | high | mitigate | Task 3 preserves the "no v1.0.0 tag exists" statement verbatim and forbids done/imminent language — the exact regression T-08-09-04 caught |
| T-RFX-04 | Tampering | REQUIREMENTS.md REL-02 rewrite | medium | mitigate | New text names `releaseWorkflowRefPattern` and the tag-anchored SAN, so the rewrite preserves rather than relaxes the LOCKED cosign contract; gate greps for both |
| T-RFX-05 | Denial of service (ROADMAP truncation) | .planning/ROADMAP.md | medium | mitigate | Task 2 mandates scoped `Edit` calls, never `Write`; gate asserts the Phase 8 title still appears at all 3 sites, which fails loudly on a truncating rewrite |
| T-RFX-06 | Elevation of privilege | package installs | low | accept | No package-manager install, no dependency change, no executable code in this plan |
</threat_model>

<verification>
Run from the repo root after all three tasks:

1. `rg -n 'REL-02' .planning/REQUIREMENTS.md .planning/ROADMAP.md .planning/STATE.md .planning/PROJECT.md`
   — every surviving hit must be either the rewritten property, a Phase 9 attribution, or an
   explicitly-preserved historical line (ROADMAP 08-09 plan checklist; STATE decision log).
2. `git diff --stat` — exactly the 6 files in `files_modified`, and nothing from &lt;forbidden_files&gt;.
3. `rg -n 'Surface Reconciliation & Signed v1.0.0 Release' .planning/ROADMAP.md .planning/STATE.md`
   — 3 ROADMAP hits + 1 STATE hit, all unchanged (the keep-the-title decision).
4. No build, no test, no lint: this plan touches zero source files.
</verification>

<success_criteria>
- REQUIREMENTS.md REL-02 is a property statement about release-please automation and cosign SAN
  compatibility, still unchecked, with no DIST-02 reference anywhere in the file.
- REQUIREMENTS.md maps REL-02 to Phase 9 / Pending.
- ROADMAP.md Phase 8 owns SURF-01..05 + REL-01/03/04 only; criterion 4 covers REL-03 alone; the stale
  dated rescope warning is deleted; Notes points release automation at Phase 9.
- ROADMAP.md Phase 9 Requirements is REL-02.
- Phase 8 heading text is unchanged at all three ROADMAP sites (decision recorded and justified above).
- 08-VERIFICATION.md records REL-02 as out of scope, neither failed nor passed, `status:` untouched.
- 08-UAT.md Test 1 notes the Phase 9 ownership move; `result: blocked`, history, and `scope_note` intact.
- STATE.md and PROJECT.md no longer assert Phase 8 ownership of REL-02 or a pending manual tag, while
  still stating truthfully that no `v1.0.0` tag exists.
- Zero files from &lt;forbidden_files&gt; appear in `git diff --name-only`.
- SUMMARY reports every REL-02 occurrence found in STATE.md and PROJECT.md with its disposition.
</success_criteria>

<output>
Create `.planning/quick/260728-rfx-rewrite-rel-02-as-a-testable-property-an/260728-rfx-SUMMARY.md` when done.
</output>
