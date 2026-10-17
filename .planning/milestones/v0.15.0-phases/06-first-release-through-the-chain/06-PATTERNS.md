# Phase 6: First Release Through the Chain - Pattern Map

**Mapped:** 2026-10-17
**Files analyzed:** 4 (all planning/doc files; no code, workflow, test or guard)
**Analogs found:** 3 / 4

## File Classification

| Modified/New File | Role | Data Flow | Closest Analog | Match |
|---|---|---|---|---|
| `.planning/REQUIREMENTS.md` (SHIP-02 line 65) | doc/requirement | in-place value edit | commit `67df2d1d` REL-11 amendment (Phase 5 D-06) | exact |
| `.planning/ROADMAP.md` (Phase 6 SC2, line 331) | doc/roadmap | in-place value edit | same commit `67df2d1d` | exact |
| `.planning/seeds/SEED-005-changie-template-for-other-repos.md` | seed | annotate existing shape | none (no seed annotated post-trigger) | no analog |
| `.planning/phases/06-.../06-VERIFICATION.md` (live-run evidence) | evidence | transcript record | `05-VERIFICATION.md` frontmatter, `05-UAT.md` Deferred Follow-Ups, `05-MUTATION-LOG.md` | role-match |

## Pattern Assignments

### REQUIREMENTS.md SHIP-02 and ROADMAP Phase 6 SC2 (D-11)

**Analog:** commit `67df2d1d` (`docs(05-02): amend REL-11 ... (D-06) and record ...`). It changed 1 line in each file (`2 insertions, 2 deletions`), with heading and checkbox counts unchanged.

Pattern: keep the `- [ ] **REL-11**:` / `  1.` shape, rewrite the clause in place, and add an inline provenance parenthetical naming the decision and date:

```
- [ ] **REL-11**: When a release PR merges (detected by changie's documented release trigger, ... ; amended by Phase 5 D-06, 2026-10-13), the workflow mints ...
```

Apply to SHIP-02 (REQUIREMENTS.md:65, currently "...equals `.changes/v0.15.0.md` byte-for-byte"). Replace "byte-for-byte" with the D-11 wording (content-equal after CRLF->LF, per-line trailing-whitespace strip, leading/trailing blank-line strip) and append "; amended by Phase 6 D-11, <date>". Edit ROADMAP.md:331 the same way. Leave the traceability row (`| SHIP-02 | Phase 6 | Pending |`, line 125) and the checkbox alone until verification passes. Commit message shape: `docs(06-NN): amend SHIP-02 to content-equal body check (D-11)`.

### SEED-005 annotation (D-14)

**No tracked analog.** SEED-001..004 carry no post-trigger annotation, and `gsd-tools list-seeds` / `plant-seed --enrich` have no verb for it. Follow RESEARCH line 158 (recommended minimal edit, and Open Question 1 for the alternative):
- Add one prose paragraph inside the existing `## When to Surface` section (the file's frontmatter keys are `id, status, planted, planted_during, trigger_when, scope`).
- Paragraph states: trigger fired, version, `https://github.com/seanb4t/codegraph-go/releases/tag/vX.Y.Z`, date, and the confirmed capability tag (verify installed vs `CONTRIBUTING.md`'s `gsd-capability-changie.git#v0.1.2`).
- Do not add headings or frontmatter keys. Leave `status: dormant` and `trigger_when` untouched. Per the generated-files rule, report the gap upstream rather than inventing structure.

### Evidence artifact `06-VERIFICATION.md` (Claude's Discretion)

**Analog:** `/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/05-release-chain-release-please-retirement/05-VERIFICATION.md`
- Frontmatter: `phase, verified, status, score, covered_files, behavior_unverified, overrides_applied, re_verification, deferred`. Copy the key set, but do NOT hand-invent `covered_digest` (tool-generated).
- Body: `# Phase N ... Verification Report`, Goal, Verified, Status, then "Goal Achievement" with per-success-criterion truths.
- Pull the live-run items from `05-VERIFICATION.md` `deferred:` and `05-UAT.md` `## Deferred Follow-Ups` (tests 3 and 4). They are the two this phase discharges. Quote the UAT shape:
```
- test: 3
  idea: "Phase 6 live run: the App opens the release PR on main and its required checks report"
```
- Transcripts: follow `05-MUTATION-LOG.md`, with dated header and per-step pasted real command output. The read-only `gh` commands and the `norm` + `cmp` comparison come from RESEARCH section 3/4 and the SC2(c) block near RESEARCH line 260.
- Checkpoints: Phase 5 PLAN files contain NO `checkpoint:human-verify` or `checkpoint:decision` tasks (rg found none). There is no in-repo analog, so use the GSD plan template's checkpoint task type for the merge and live-run steps (milestone PR merge, release-PR merge), per CONTEXT.

## Shared Patterns

- **Values only, existing shapes:** `generated-files` rule applies to REQUIREMENTS/ROADMAP/seed files. Verify with `git diff` that each edit changes only the intended line (`67df2d1d` is the model: `2 insertions, 2 deletions`).
- **Provenance parenthetical:** "amended by Phase N D-xx, date" on every wording change.
- **Commit style:** `docs(06-NN): ...`, with the Co-Authored-By trailer.
- **Tracked paths only:** all paths above are tracked (`git ls-files` confirmed for seeds; phase 5 docs are committed).

## No Analog Found

| File | Reason |
|---|---|
| SEED-005 post-trigger annotation | No seed has been annotated after firing. Use the RESEARCH line 158 recommendation. |
| `checkpoint:*` plan tasks | Phase 5 plans have none. Use the template. |

## Metadata

**Search scope:** `.planning/phases/05-*`, `.planning/seeds`, git history of REQUIREMENTS.md and ROADMAP.md
**Pattern extraction date:** 2026-10-17
