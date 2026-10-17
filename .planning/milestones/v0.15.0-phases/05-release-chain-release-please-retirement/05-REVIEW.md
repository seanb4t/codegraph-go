---
phase: 05-release-chain-release-please-retirement
reviewed: 2026-10-16T00:00:00Z
depth: deep
files_reviewed: 1
files_reviewed_list:
  - docs/RELEASE-PROCEDURES.md
findings:
  critical: 0
  warning: 5
  info: 4
  total: 9
status: issues_found
---

# Phase 5: Code Review Report

**Reviewed:** 2026-10-16
**Depth:** deep
**Files Reviewed:** 1
**Status:** issues_found

## Summary

`docs/RELEASE-PROCEDURES.md` is a maintainer runbook, so I checked its claims against the code it describes. That meant `.github/workflows/release-pr.yml`, `release-tag.yml`, `release.yml`, `post-release-verify.yml`, the `Taskfile.yml` targets (`release:batch`, `release:tag`, `release:goreleaser`, `check:cross`), `.changie.yaml`, `.goreleaser.yaml`, and `internal/upgrade/verify.go`.

The parts this phase rewrote (§1 to §4, §7, §9) match the workflows:
- The `verify.go` constants in §5 are byte-identical to the source.
- `check:cross` covers the four targets the doc lists.
- `release:batch`, `release:tag` and `release:goreleaser` behave as described.
- `app-id` is deprecated and `client-id` is current in the pinned `create-github-app-token` v3.2.0.

The defects below are internal contradictions and stale passages. The most important one is a section that describes a pipeline that no longer exists. Another is a rollback procedure that conflicts with a rule the diff just made unconditional. No BLOCKER-class defect was found: nothing in the doc causes data loss or a security regression if followed literally, but two of the items below can push a maintainer toward an action the same doc forbids.

Verification note: the `files` scope is a single document. I did not review `docs/RELEASE.md`, which the runbook cross-references.

## Warnings

### WR-01: §10(a) describes a pipeline that no longer exists and cites a step §4 does not contain

**File:** `docs/RELEASE-PROCEDURES.md:504-516`
**Issue:** §10(a) says:
- "This repo has never run `goreleaser release`."
- `archives:` and `checksum:` are "documented-inert".
- Signing, SBOM and publishing "live in `release.yml`'s hand-written `assemble` job".
- Idempotent replacement is satisfied by "§4's `gh release upload --clobber` step".
- The reasons given are the "native darwin/linux two-OS matrix" and the "SLSA generic-generator handoff".

All of this is false for the current tree:
- `release.yml:76` and `:143` say the `assemble` job is deleted.
- `release.yml` has a single `release` job that runs `task release:goreleaser`, which runs `goreleaser release --clean`.
- `.goreleaser.yaml:399` sets `replace_existing_artifacts: true`.
- Provenance now uses `actions/attest-build-provenance`, not the SLSA generic generator.
- §4 of this same document describes the collapsed single-job `goreleaser release` flow and contains no `gh release upload --clobber` step.

A maintainer reading §10 gets the opposite architecture from §4 and a dangling cross-reference. The phase's goal was to remove stale passages, and this one was left behind.
**Fix:** Rewrite §10(a) as a historical note, or delete it. For example: "Superseded: Phase 9 originally recorded a divergence from the `goreleaser release` upload model. Phase 1 (plans 01-03/01-04) collapsed the pipeline into a single `goreleaser release` job, so the divergence no longer exists. See §4." Also drop the "(§ below)" ROADMAP pointer if it has no target (see IN-03).

### WR-02: §7 forbids reusing a stable tag name, but its own rollback re-creates exactly that

**File:** `docs/RELEASE-PROCEDURES.md:340-347` (with `:176-181`, `:267-278`, `:464-469`)
**Issue:** This phase's diff deleted "Prereleases (`-rc.N`) are safer to delete/retag". That sentence was the exemption that made the disposable live-proof teardown coherent. With the prerelease shape removed (§2, §4), every cut, including the §9 live proof, is a stable-shaped `vX.Y.Z`.

§7 still says:
- Never reuse a deleted stable tag name.
- The delete-release-and-tag plus revert procedure applies to the §9 live-proof cut.

Following that procedure has this result:
- The revert restores the fragments.
- `release-pr.yml` reopens a release PR for the same derived version.
- Merging it makes `release-tag.yml` push the same `vX.Y.Z` again.

So the documented teardown either violates "never reuse" or leaves the maintainer stuck. The doc never says which rule wins for a disposable cut.
**Fix:** Add an explicit carve-out to the "Never reuse a tag name" paragraph. For example: "Exception: the §9 live-proof cut, which has no installed base, may be re-cut under the same name. If `codegraph upgrade` could already have resolved it as latest, patch forward instead." Alternatively, state that after a teardown the next release PR must produce a different version, for example by landing a new fragment first.

### WR-03: §12 tap-failure remedy re-runs the full `goreleaser release`, which replaces live signed and notarized assets, while claiming "nothing about the release needs to be touched"

**File:** `docs/RELEASE-PROCEDURES.md:673-682`
**Issue:** The remedy is "re-run `goreleaser release` for the SAME tag". The doc does not say how:
- `goreleaser release --clean` is only invoked through `task release:goreleaser`, which needs the macOS runner, the Apple notarization secrets and the tap token.
- The documented mechanism is therefore re-running the `release` workflow job.

A full re-run does more than push the cask:
- It rebuilds, re-notarizes, re-signs and re-generates SBOMs, then re-uploads.
- `release.replace_existing_artifacts: true` (`.goreleaser.yaml:399`) overwrites the live assets.
- The `Attest` step then re-attests the new bytes.
- If the rebuild is not byte-identical, the published assets, cosign bundles and checksums change under users who already verified them.
- It also invalidates the SIGN-04 byte-identity record, and conflicts with §7.1's "do not replace" for preserved baselines.

That contradicts the paragraph just above it, which says the release is untouched. Calling the cask push idempotent describes only the cask step, not the surrounding job.
**Fix:** Either specify a tap-only re-publish (for example a documented `goreleaser` flag combination, or a manual push of the rendered cask from `dist/`) and note that it must not re-upload release assets. Or state plainly that a re-run replaces the assets and that the maintainer must re-verify per §6 afterward. Do not describe the re-run as leaving the release untouched.

### WR-04: §4 step 5 says only a release-PR merge fires `release-tag.yml`, but the doc elsewhere and the workflow say otherwise

**File:** `docs/RELEASE-PROCEDURES.md:108-113` (vs `:334-338`, and `.github/workflows/release-tag.yml:1-9`)
**Issue:** §4 step 5 asserts the trigger is "a push to `main` that changes `CHANGELOG.md`, so only a release-PR merge fires it." The facts are:
- §7 says a revert of the release commit also fires it.
- `release-tag.yml`'s own header says the milestone/integration-branch merge, which changes `CHANGELOG.md`, also fires it as a no-op.
- §3 step 3 never warns that merging the integration branch will trigger a tagging run.

If the integration merge brings a `.changes/vX.Y.Z.md` and `CHANGELOG.md` entry whose tag is missing, `changie latest` resolves to an untagged version and the job tags it. The doc's "only" is wrong and hides that case.
**Fix:** Reword to "a push to `main` that changes `CHANGELOG.md` (normally only a release-PR merge; an integration-branch merge or a revert also fires it and is a no-op when `changie latest` is already tagged)". Add one sentence to §3 step 3 saying the same.

### WR-05: Decision IDs collide within the same edit and across sections, so cross-references are ambiguous

**File:** `docs/RELEASE-PROCEDURES.md:11`, `:37`, `:51`, `:153`, `:450`, `:529`
**Issue:** The same IDs mean different decisions in different sections, with no phase qualifier:
- `D-15` appears at line 37 as `task check:cross (D-15)`, and at line 51 and the §4 heading as "No hand-pushed prerelease tag (D-15)". Both are in the section this diff edited.
- `D-09` is the pre-tag gate at line 11 and the attestor swap at line 153.
- `D-03` is the App authoring the release PR at line 450, but the `workflow_dispatch` fallback at line 529.
- `D-07` is used for both the Phase 1 patch-forward rule and the Phase 5 pretag-gate in the workflows.

A maintainer grepping for "D-15" cannot tell which decision is meant. The new text added in this phase made the collision worse.
**Fix:** Qualify new references with the phase, for example `05-D-15` or "Phase 5 D-15", and at minimum disambiguate line 51 and the §4 heading.

## Info

### IN-01: `rm -P` is BSD/macOS-only and does not securely erase on APFS or SSD

**File:** `docs/RELEASE-PROCEDURES.md:655-656`
**Issue:** The step tells the maintainer to "shred" key copies with `rm -P`. `-P` is a BSD option that GNU `rm` rejects and macOS has deprecated. On APFS and SSDs it gives no overwrite guarantee. The sentence gives false assurance about a private key.
**Fix:** Say "delete the file (`rm -f`) and confirm it was never committed or logged". Add that secure erasure is not guaranteed on modern filesystems, so rotating the key is the real control.

### IN-02: Stale integration-branch name

**File:** `docs/RELEASE-PROCEDURES.md:64-65`
**Issue:** The doc hard-codes `gsd/v1.0-drop-in-parity-human-ux` "for this milestone". The branch in use is `gsd/v0.15.0-milestone`, and the project has already shipped v0.x releases past that milestone.
**Fix:** Replace with a generic "the milestone integration branch".

### IN-03: Dangling and opaque references

**File:** `docs/RELEASE-PROCEDURES.md:354`, `:498`, `:515`, `:568`
**Issue:** Four references cannot be resolved by a reader:
- Line 354 cites "repo rule `xmz3xknbj0`", an opaque identifier with no location.
- §10's heading says "(Phase 9)" and line 515 says "`.planning/ROADMAP.md` (§ below)" with no section below.
- Line 568 says "gated to a `success` conclusion (§ above)" with no section named.
- Line 54 points at "§4, 'No hand-pushed prerelease tag (D-15)'", which exists as a bold paragraph and not a heading, so it is hard to find.

**Fix:** Name the sections explicitly, for example "§4, paragraph 'No hand-pushed prerelease tag'", and cite the real file for the signing rule.

### IN-04: `gh` commands omit the repository, so they target whatever checkout the maintainer is in

**File:** `docs/RELEASE-PROCEDURES.md:314`, `:442`
**Issue:** `gh release delete vX.Y.Z --cleanup-tag -y` and `gh secret list` take no `-R seanb4t/codegraph-go`, unlike the neighbouring `gh api repos/...` calls. From a fork clone or another repository's checkout, the destructive `gh release delete ... -y` could act on the wrong repository, with no confirmation prompt.
**Fix:** Add `-R seanb4t/codegraph-go` to both commands. For the destructive one, consider dropping `-y` so there is a prompt.

---

_Reviewed: 2026-10-16_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
