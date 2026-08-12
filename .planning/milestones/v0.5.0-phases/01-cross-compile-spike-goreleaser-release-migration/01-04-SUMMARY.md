---
phase: 01-cross-compile-spike-goreleaser-release-migration
plan: 04
subsystem: infra
tags: [github-actions, attestations, cosign, slsa, provenance, release, ci]

# Dependency graph
requires:
  - phase: 01-03 (this phase, wave 2)
    provides: "release.yml collapsed to one `release` job (plan 01-03), plus the temporary `provenance:`-job `id-token: write` allowance and `hashes` job output plan 01-03 left in place naming this plan as the remover"
provides:
  - "release.yml's `provenance:` job (slsa-framework/slsa-github-generator's generic generator) deleted entirely, replaced by a SHA-pinned `actions/attest-build-provenance` step inside the goreleaser-invoking job, over the same 8-payload subject set the checksums file covers (D-09/D-12)"
  - "release.yml: `attestations: write` added to the release job's permissions; `id-token: write` now has exactly one holder in the whole file, no allowance (D-11)"
  - "internal/upgrade/release_workflow_shape_test.go: attestStepShape/parseAttestStep/mustAttestStep/attestActionPinnedRe (replacing provenanceJobShape/parseReleaseProvenanceJob/mustReleaseProvenanceJob/slsaGeneratorTaggedRe); TestProvenanceAttestorIsPinnedNativeAction (cross-file resolves subject-checksums: against .goreleaser.yaml's checksum.name_template through the real template engine); TestParseAttestStep_NoAttestStepIsError; TestOIDCWriteScopedToSingleGoreleaserJob simplified to assert exactly one holder"
  - "docs/RELEASE.md, docs/RELEASE-PROCEDURES.md, SECURITY.md, README.md rewritten to name `gh attestation verify` in place of `slsa-verifier verify-artifact` for post-migration releases, with pre-migration verification instructions retained but explicitly labelled historical"
  - ".planning/REQUIREMENTS.md's REL-08 reworded to name `gh attestation verify`, dated and rationale-annotated"
  - "docs/RELEASE-PROCEDURES.md's rollback section states D-07's patch-forward recovery posture explicitly and removes the prior 'just delete the tag directly' instruction"
affects: ["01-05 (must name the exact migration-cutover tag in docs/RELEASE.md § b's placeholder once cut; is the authoritative closer for REL-08 against a real published release; must reconcile ROADMAP.md's Phase 1 Success Criterion 3, still naming the pre-migration verifier command)", "01-06"]

# Actuals (#2632)
actuals:
  tokens: 12230
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "attestStepShape/parseAttestStep: scans every top-level job's line range (shared job-boundary-scanning technique with parseReleaseJobShapes) for the one step whose uses: references actions/attest-build-provenance, extracting its job id and subject-checksums: input — generalizes the single-job-lookup shape parseReleaseProvenanceJob established, now searching across jobs rather than for one named job"
    - "Cross-file template resolution over pattern matching: TestProvenanceAttestorIsPinnedNativeAction resolves both release.yml's subject-checksums: (GitHub Actions expression syntax) and .goreleaser.yaml's checksum.name_template (Go template syntax) against the SAME pinned tag literal via resolveGoreleaserFieldTemplate, then compares resulting filenames — not template source strings (review Codex Plan-04 LOW, C8)"

key-files:
  created: []
  modified:
    - .github/workflows/release.yml
    - internal/upgrade/release_workflow_shape_test.go
    - docs/RELEASE.md
    - docs/RELEASE-PROCEDURES.md
    - SECURITY.md
    - README.md
    - .planning/REQUIREMENTS.md

key-decisions:
  - "The Taskfile.yml base64/hashes-emission logic plan 01-03 folded into `release:goreleaser` (guarded on $GITHUB_OUTPUT) becomes dead code after this plan — its only consumer, the `provenance:` job's `needs.release.outputs.hashes` reference, is deleted. Taskfile.yml is outside this plan's file scope (owned concurrently by plan 01-06), so it was left untouched rather than edited out of scope; the job's own `outputs:` block in release.yml WAS deleted (in scope) since nothing references it anymore. Flagged here rather than silently left for someone else to notice."
  - "The mismatched-checksum-stem mutation-RED demonstration (acceptance criterion for TestProvenanceAttestorIsPinnedNativeAction) was run against an out-of-repo scratch copy of .goreleaser.yaml, never the tracked file — .goreleaser.yaml is outside this plan's file scope. Mirrors plan 01-03's identical precedent for its TestNoGoreleaserHooksInReleaseConfig mutation."
  - "actions/attest-build-provenance pinned to v4.2.2 / SHA 4d101475d8b20a2381f78447822ac1eab6504dd8, resolved live via `gh api repos/actions/attest-build-provenance/releases/latest` then `gh api repos/actions/attest-build-provenance/git/ref/tags/v4.2.2` — not assumed from the plan's [ASSUMED] research note."
  - "docs/RELEASE.md § b's migration-cutover tag is left as an explicit placeholder ('the first release cut by the migrated pipeline') per the plan's own instruction — plan 01-05 will know the real tag and must fill it in."

patterns-established:
  - "A step-level shape guard (parseAttestStep) reuses an existing job-level shape guard's (parseReleaseJobShapes) line-range-scanning primitives rather than re-implementing job-boundary detection — the same reuse discipline plan 01-03 established for parseGoreleaserInvokingJob."

requirements-completed: [REL-08]

coverage:
  - id: D1
    description: "release.yml: provenance: job (SLSA generic generator) deleted entirely; actions/attest-build-provenance@4d101475d8b20a2381f78447822ac1eab6504dd8 runs as the last step of the goreleaser-invoking job, over the 8-payload subject set the checksums file covers; attestations: write added; id-token: write has exactly one holder, no allowance"
    requirement: "REL-08"
    verification:
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestProvenanceAttestorIsPinnedNativeAction"
        status: pass
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestParseAttestStep_NoAttestStepIsError"
        status: pass
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestOIDCWriteScopedToSingleGoreleaserJob"
        status: pass
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestWorkflowSourceHelpersFailLoudly"
        status: pass
      - kind: other
        ref: "task lint:actions (actionlint, exit 0)"
        status: pass
      - kind: other
        ref: "three mutation-RED demonstrations against the real file, recorded below and reverted; a fourth against an out-of-repo scratch .goreleaser.yaml copy"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/RELEASE.md, docs/RELEASE-PROCEDURES.md, SECURITY.md, README.md, .planning/REQUIREMENTS.md rewritten to name gh attestation verify in place of slsa-verifier verify-artifact for post-migration releases, with historical instructions retained but explicitly labelled"
    requirement: "REL-08"
    verification:
      - kind: other
        ref: "rg -n 'slsa-verifier verify-artifact' SECURITY.md README.md .planning/REQUIREMENTS.md (0 matches)"
        status: pass
      - kind: other
        ref: "rg -c 'gh attestation verify' docs/RELEASE.md docs/RELEASE-PROCEDURES.md SECURITY.md README.md (all >=1)"
        status: pass
      - kind: other
        ref: "rg -n 'assemble' docs/RELEASE-PROCEDURES.md (both remaining hits are non-live/historical, not describing a current pipeline job)"
        status: pass
      - kind: other
        ref: "git diff docs/RELEASE.md — certificate-identity-regexp line byte-identical (no diff)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Reconciliation note: .planning/ROADMAP.md's Phase 1 Success Criterion 3 still names the pre-migration verifier command and was deliberately NOT edited here — ROADMAP.md's phase criteria are a tool-owned generated artifact (per this repo's planning-artifacts rule), updated through the roadmap workflow, not by hand-editing prose"
    requirement: "REL-08"
    verification: []
    human_judgment: true
    rationale: "Requires the roadmap-update tooling, not a plan-execution edit; recorded here per this plan's own <verification> instruction so it is not silently missed at phase verification."

duration: 21min
completed: 2026-08-08
status: complete
---

# Phase 1 Plan 4: Native GitHub Attestor Swap (D-09/D-10) Summary

**Replaced the third-party SLSA generic-generator reusable workflow with a SHA-pinned `actions/attest-build-provenance` step inside the single goreleaser job, and rewrote every published verification instruction (`docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`, `SECURITY.md`, `README.md`, REL-08 itself) to name `gh attestation verify` instead of the now-architecturally-incompatible `slsa-verifier verify-artifact`.**

## Performance

- **Duration:** ~21 min
- **Started:** 2026-08-08T17:33:15-04:00 (base commit)
- **Completed:** 2026-08-08T17:54:27-04:00
- **Tasks:** 2 of 2 completed
- **Files modified:** 7

## Accomplishments

- `release.yml`'s `provenance:` job (`slsa-framework/slsa-github-generator`'s generic generator, its hand-rolled base64 subject-list job output, and the temporary `id-token: write` allowance plan 01-03 left for it) is gone entirely. In its place, a SHA-pinned `actions/attest-build-provenance@4d101475d8b20a2381f78447822ac1eab6504dd8 # v4.2.2` step runs as the last step of the goreleaser-invoking job, attesting the same 8-payload subject set (4 raw binaries + 4 `.zip` archives) the checksums file covers.
- `id-token: write` now has exactly one holder in `release.yml`; `attestations: write` is new on that same job.
- `internal/upgrade/release_workflow_shape_test.go` gained `attestStepShape`/`parseAttestStep`/`mustAttestStep`/`attestActionPinnedRe`, replacing the deleted `provenanceJobShape`/`parseReleaseProvenanceJob`/`mustReleaseProvenanceJob`/`slsaGeneratorTaggedRe`. The new `TestProvenanceAttestorIsPinnedNativeAction` proves the workflow's `subject-checksums:` input resolves to the SAME concrete filename `.goreleaser.yaml`'s `checksum.name_template` resolves to — using the real template engine on both sides, not a loose pattern match (review Codex Plan-04 LOW / C8).
- Every published verification instruction that named `slsa-verifier verify-artifact` — a command that architecturally cannot verify `actions/attest-build-provenance` output — now names `gh attestation verify <asset> -R seanb4t/codegraph-go` instead, in `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`, `SECURITY.md`, `README.md`, and REL-08's own wording. Historical/pre-migration instructions are retained but explicitly labelled, not deleted.
- `docs/RELEASE-PROCEDURES.md`'s rollback section now states D-07's patch-forward recovery posture explicitly and removes the prior "just delete the tag directly" instruction, with an explanation of why (re-pushing a tag re-fires `release.yml` and is human-touched tag authority release-please owns, D-06R).

## Task Commits

1. **Task 1: Swap the attestor — native action in, reusable workflow out (D-09)** — `9fa6fd8` (feat) — TDD: RED confirmed (new tests failed with no attest step present), then GREEN
2. **Task 2: Rewrite every published verification instruction and REL-08's own wording (D-10)** — `b56f459` (docs)

**Plan metadata:** committed separately (this SUMMARY, per worktree mode)

## Files Created/Modified

- `.github/workflows/release.yml` — `provenance:` job deleted; `Attest build provenance (8 subjects)` step added to the `release` job; `attestations: write` added to its permissions; the job's now-unreferenced `outputs: hashes:` block deleted; header comment's SHA-pinning statement made unconditional (the SLSA generator's tag-pin exception left with it)
- `internal/upgrade/release_workflow_shape_test.go` — `attestStepShape`/`parseAttestStep`/`mustAttestStep`/`attestActionPinnedRe` added; `provenanceJobShape`/`parseReleaseProvenanceJob`/`mustReleaseProvenanceJob`/`slsaGeneratorTaggedRe` deleted (zero remaining references, including comments); `TestProvenanceAttestorIsPinnedNativeAction` and `TestParseAttestStep_NoAttestStepIsError` added; `TestProvenanceJobUsesTaggedSLSAGenerator` deleted; `TestOIDCWriteScopedToSingleGoreleaserJob` simplified (no allowance); `TestWorkflowSourceHelpersFailLoudly`'s table entry swapped
- `docs/RELEASE.md` — published-asset list gains `.zip` archives, loses `.intoto.jsonl`; § a unchanged plus one sentence; § b rewritten for `gh attestation verify` with a realistic sample output and a labelled pre-migration historical note
- `docs/RELEASE-PROCEDURES.md` — §4's job-by-job description rewritten for the single `release` job; §4's rc-escape-hatch paragraph reworded; §6's post-release verification block rewritten; §7's rollback section states D-07's patch-forward posture and drops the tag-delete-directly instruction
- `SECURITY.md` — SLSA-provenance sentence rewritten
- `README.md` — Provenance supply-chain bullet rewritten
- `.planning/REQUIREMENTS.md` — REL-08 reworded (dated parenthetical; other two claims unchanged)

## Decisions Made

1. **Taskfile.yml's dead hashes-emission logic left in place, flagged not fixed.** Plan 01-03 folded the SLSA-provenance base64/hashes computation into `Taskfile.yml`'s `release:goreleaser` target, guarded on `$GITHUB_OUTPUT`. Its only consumer was the `provenance:` job's `needs.release.outputs.hashes` reference, which this plan deletes along with the job. `Taskfile.yml` is outside this plan's file scope (owned concurrently by plan 01-06 this wave), so the now-dead computation inside it was left untouched rather than edited out of scope. The `outputs:` block in `release.yml` itself (in scope) WAS deleted since nothing references it. This is a harmless no-op today (still guarded on `$GITHUB_OUTPUT`, computes a value nobody reads) but is worth a follow-up cleanup pass in `Taskfile.yml` — flagged here rather than silently left for someone to notice later.
2. **Mismatched-checksum-stem mutation-RED demonstrated against a scratch copy, not the tracked `.goreleaser.yaml`.** That file is outside this plan's scope. Mirrors plan 01-03's identical precedent (`TestNoGoreleaserHooksInReleaseConfig`): a standalone Go program reproducing the exact `checksum.name_template` resolution + comparison logic confirmed detection on a mutated scratch copy (`_sums.txt` stem) and confirmed clean against the real file. No repo file was mutated for this demonstration.
3. **Pin resolved live, not assumed.** `gh api repos/actions/attest-build-provenance/releases/latest` returned `v4.2.2`; `gh api repos/actions/attest-build-provenance/git/ref/tags/v4.2.2` resolved it to commit `4d101475d8b20a2381f78447822ac1eab6504dd8` (a direct commit object, not an annotated tag). RESEARCH.md had explicitly marked this `[ASSUMED]` and required resolution at implementation time.
4. **`docs/RELEASE.md` § b's cutover tag left as a placeholder.** Per the plan's own instruction: "until then, name it as 'the first release cut by the migrated pipeline' and leave a marker for 01-05 to fill in." Plan 01-05 is the authoritative closer and will know the real tag.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Removed stray literal references to deleted symbol names from surviving comments**
- **Found during:** Task 1, post-edit grep for the acceptance criterion "contains no reference to `slsaGeneratorTaggedRe`, `parseReleaseProvenanceJob`, or `provenanceJobShape`"
- **Issue:** Three doc comments on symbols that survived (`parseReleaseJobShapes`, `attestStepShape`, `attestActionPinnedRe`) still named the deleted symbols in prose explaining what they generalize/replace — technically satisfying "the code compiles" but failing the plan's own mechanical grep check.
- **Fix:** Reworded all three comments to describe the same relationship without using the deleted identifiers as literal text.
- **Files modified:** `internal/upgrade/release_workflow_shape_test.go`
- **Verification:** `rg -n 'slsaGeneratorTaggedRe|parseReleaseProvenanceJob|provenanceJobShape|mustReleaseProvenanceJob'` returns nothing; full package test suite still green.
- **Committed in:** `9fa6fd8` (Task 1 commit)

**2. [Rule 1 - Bug] Reworded REL-08's dated parenthetical to avoid re-introducing the forbidden string**
- **Found during:** Task 2, post-edit grep for the acceptance criterion "`rg -n 'slsa-verifier verify-artifact' SECURITY.md README.md .planning/REQUIREMENTS.md` returns nothing"
- **Issue:** The first draft of REL-08's rationale parenthetical explained the reword by naming the literal old command (`slsa-verifier verify-artifact`), which is exactly the string the acceptance criterion forbids appearing in that file.
- **Fix:** Reworded to "the pre-migration verifier command" — same meaning, no forbidden literal.
- **Files modified:** `.planning/REQUIREMENTS.md`
- **Verification:** the grep returns 0 matches across all three files.
- **Committed in:** `b56f459` (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — mechanical corrections to satisfy the plan's own stated acceptance criteria, no scope creep)
**Impact on plan:** Both fixes were required for the acceptance criteria to actually pass as literally stated; no functional change beyond comment/prose wording.

## Issues Encountered

**Pre-existing, out-of-scope observation:** a full `go test ./...` run surfaced `internal/daemon`'s `TestRunWatchdogCancelsRunOnSimulatedReparent` failing ("Run did not return after a simulated reparent — watchdog is not wired into Run") under this session's concurrent multi-worktree CPU load (a sibling `go test -race` process was observed running concurrently via `ps aux`). This is STATE.md's documented, accepted MAINT-02 flake ("fails under full-suite load, passes isolated," CI-load ruled the governing standard, 2026-08-06). Re-ran the same test in isolation and it passed in 1.09s. `git diff --stat` confirms zero files under `internal/daemon/` were touched by either of this plan's two commits. Not fixed here, per the executor scope boundary; not added to `deferred-items.md` (that file is for out-of-scope work THIS plan's own changes caused, which this is not — mirrors plan 01-03's identical precedent for an unrelated `test/wireoracle` failure).

## User Setup Required

None. No external service configuration required. `gh attestation verify` requires only the `gh` CLI, already ambient in every GitHub Actions job and already used elsewhere in this pipeline.

## Next Phase Readiness

**This plan is COMPLETE** for its own file scope. Both tasks executed, all stated acceptance-criteria greps pass, both TDD gates (RED confirmed against the pre-swap file, GREEN confirmed after) pass, `task lint:actions` is clean, and `go test ./internal/upgrade/...` and the full `go test ./...` suite (module-wide) both pass except the pre-existing, unrelated, load-dependent `internal/daemon` flake documented above.

**Outstanding for plan 01-05 (the authoritative closer, per this plan's own text):**
- Fill in `docs/RELEASE.md` § b's migration-cutover tag placeholder once the real tag is known.
- Prove `gh attestation verify` and `cosign verify-blob` both succeed against a real published release.
- Reconcile `.planning/ROADMAP.md`'s Phase 1 Success Criterion 3, which still names the pre-migration verifier command — not editable here since it is a tool-owned generated artifact (`.planning/ROADMAP.md`'s phase criteria are updated through the roadmap workflow, not by hand-editing prose).
- Re-run the two JOINT `rg` acceptance commands from plan 01-03's Task 2 (REL-06/REL-07 end-state check across all of wave 2's concurrent commits) — unaffected by this plan but still outstanding per 01-03-SUMMARY.md.

**Flagged for a later cleanup pass (not blocking):** `Taskfile.yml`'s `release:goreleaser` target still computes the base64-encoded checksums hash for `$GITHUB_OUTPUT` — dead code now that this plan removed its sole consumer. Outside this plan's file scope; see Decisions Made #1.

---
*Phase: 01-cross-compile-spike-goreleaser-release-migration*
*Completed: 2026-08-08*

## Self-Check: PASSED

- FOUND: `.github/workflows/release.yml`
- FOUND: `internal/upgrade/release_workflow_shape_test.go`
- FOUND: `docs/RELEASE.md`
- FOUND: `docs/RELEASE-PROCEDURES.md`
- FOUND: `SECURITY.md`
- FOUND: `README.md`
- FOUND: `.planning/REQUIREMENTS.md`
- FOUND commit: `9fa6fd8` (Task 1)
- FOUND commit: `b56f459` (Task 2)
