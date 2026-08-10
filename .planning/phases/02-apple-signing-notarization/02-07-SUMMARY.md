---
phase: 02-apple-signing-notarization
plan: 07
subsystem: infra
tags: [goreleaser, apple-notarization, gatekeeper, cosign, sigstore, github-attestations, release]

# Dependency graph
requires:
  - phase: 02-05
    provides: "Developer ID signing + notarize.macos pipe wired into .goreleaser.yaml"
  - phase: 02-06
    provides: "post-release-verify.yml's Gatekeeper/self-upgrade/notarized-suite/verify-supply-chain jobs"
provides:
  - "release:record-final-hashes — the post-publish, honestly-labelled final_local_sha256 evidence step wired into release.yml"
  - "The first real, published, notarized codegraph release (v0.7.0), with every ROADMAP Phase 2 criterion measured against it rather than rehearsed"
  - "02-EVIDENCE.md's SIGN-02 GREEN section, SIGN-04 five-point byte-identity section (with an honest per-arch binding-coverage gap recorded), and criterion-4 notarized-suite section"
  - "docs/RELEASE.md reconciled: both pending markers closed with citations, chmod/execute-bit guidance added"
affects: [03-homebrew-cask-distribution]

# Actuals (#2632)
actuals:
  tokens: 6209
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Post-everything provenance labelling: final_local_sha256 is named for what it measures (after archive/checksum/sign/publish all complete), never for when a reader might assume it was captured — the same discipline this plan required of its own evidence file"
    - "Binding vs. hash provenance points recorded distinctly: cosign verify-blob / gh attestation verify are quoted commands with verbatim results, never a fabricated fourth/fifth hash"
    - "Honest scope-limit recording over convenient silence: this plan documents that verify:release-assets only exercises linux/amd64, not darwin, rather than implying per-arch binding coverage exists where it doesn't"

key-files:
  created: []
  modified:
    - Taskfile.yml
    - .github/workflows/release.yml
    - .planning/phases/02-apple-signing-notarization/02-EVIDENCE.md
    - docs/RELEASE.md
    - docs/RELEASE-PROCEDURES.md

key-decisions:
  - "Task 1's release:record-final-hashes step hard-fails the workflow on absent metadata rather than degrading silently — a green release with SIGN-04 permanently unprovable was judged worse than a red one with a documented, workflow_dispatch-recoverable failure branch"
  - "PR title for the release-cutting merge was deliberately chosen as a feat: commit so the squash-merge triggered a MINOR version bump (0.6.0 -> 0.7.0) under release-please's Conventional Commits parsing, avoiding a repeat of a prior build(ci): title collapsing 17 feat commits into no version bump at all"
  - "The wireoracle stderr-capture race (test/wireoracle/capture.go reading stderrBuf.String() before cmd.Wait() was joined) was fixed BEFORE cutting the release (issue #49, PR #50, merged 29f6e3d) because Taskfile.yml's verify:notarized-suite depends on that harness to produce criterion 4's evidence against the real published binary"
  - "SIGN-04 points 4-5 (cosign/attestation bindings) are recorded with an explicit scope-limit: verify:release-assets exercises linux/amd64 only, never darwin, and darwin/amd64 has no cosign/attestation binding check anywhere in the pipeline (self-upgrade's matrix is darwin/arm64 + linux/amd64 only) — recorded plainly rather than presented as uniform per-arch coverage"
  - "docs/RELEASE.md's reproducibility-posture pending marker was not just closed but corrected: the prior text claimed the cert holder could reproduce the darwin signature bit-for-bit, which the SIGN-04 rehearsal's own non-reproducible-signature finding (Apple's embedded trusted timestamp varies per signing operation) disproves even for the cert holder — the doc now says so"

patterns-established:
  - "RED-vs-GREEN evidence tables: pairing an earlier deliberately-un-notarized baseline against the same target/procedure run on a real release, with an explicit one-sentence trustworthiness justification, rather than presenting the GREEN result in isolation"

requirements-completed: [SIGN-01, SIGN-02, SIGN-03, SIGN-04]

coverage:
  - id: D1
    description: "final_local_sha256 observable via release:record-final-hashes, honestly labelled as a post-everything measurement, wired into release.yml between the release and attestation steps, hard-failing on absent metadata"
    requirement: SIGN-04
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/ -v"
        status: pass
      - kind: other
        ref: "SIGN04-PUBLISH-EVIDENCE lines emitted by release run 31337821840 for both darwin platforms, recorded in 02-EVIDENCE.md"
        status: pass
    human_judgment: false
  - id: D2
    description: "The v0.7.0 release cut, notarized (not skipped), and observed: Gatekeeper GREEN on both darwin arches with quarantine confirmed, the notarized-suite job executing 142 tests against the real published arm64 binary, a manual-dispatch guard check proving no job silently skips, and the maintainer's own browser download launching without a Gatekeeper dialog"
    requirement: SIGN-02
    verification:
      - kind: other
        ref: "post-release-verify.yml automatic run 31338004416 (all 7 jobs SUCCESS) and manual-dispatch run 31338409898 (all 7 jobs SUCCESS, none skipped) — job conclusions recorded in 02-EVIDENCE.md"
        status: pass
    human_judgment: true
    rationale: "This deliverable's central claim (checkpoint step 8: no Gatekeeper dialog appeared on a real Safari download) is an unproxied human observation on the maintainer's own machine — by design, this is the one place in the phase where no automated instrument stands in for the actual user experience, so it cannot auto-pass on test status alone even though every supporting job is green."
  - id: D3
    description: "Criteria 2, 3 and 4 recorded against the published release in 02-EVIDENCE.md (SIGN-02 GREEN section + RED-vs-GREEN table, SIGN-04 five-point byte identity with narrow A3 verdict, criterion-4 notarized-suite section with arm64-only scope limit), and docs/RELEASE.md's two pending markers closed with citations plus chmod guidance added"
    requirement: SIGN-04
    verification:
      - kind: unit
        ref: "rg -c 'SIGN04-PUBLISH-EVIDENCE|GATEKEEPER-EVIDENCE' .planning/phases/02-apple-signing-notarization/02-EVIDENCE.md (8, >=2 required)"
        status: pass
      - kind: unit
        ref: "go test ./internal/upgrade/ -v"
        status: pass
      - kind: unit
        ref: "task test:unit"
        status: pass
      - kind: other
        ref: "rg -c 'pending' docs/RELEASE.md == 0"
        status: pass
    human_judgment: false

duration: ~90min (across the full plan; this continuation covered Task 3 only, ~25min)
completed: 2026-08-09
status: complete
---

# Phase 2 Plan 07: Cut and Prove the Notarized Release Summary

**Cut `v0.7.0` — the first real, published, Apple-notarized codegraph release — and closed every ROADMAP Phase 2 criterion against it: GREEN Gatekeeper on both darwin arches, a five-point byte-identity chain (with an honestly-recorded per-arch binding gap), 142 tests executed against the real notarized binary, and both `docs/RELEASE.md` pending markers settled with citations.**

## Performance

- **Duration:** ~90 min across the full plan (Task 1 + checkpoint + Task 3); this continuation executed Task 3 only, ~25 min
- **Started:** 2026-08-09 (Task 1); Task 3 continuation started 2026-08-09
- **Completed:** 2026-08-09
- **Tasks:** 3 (all complete)
- **Files modified:** 5 across the plan (`Taskfile.yml`, `.github/workflows/release.yml`, `docs/RELEASE-PROCEDURES.md` — Task 1; `.planning/phases/02-apple-signing-notarization/02-EVIDENCE.md`, `docs/RELEASE.md` — Task 3)

## Accomplishments

- `release:record-final-hashes` (Task 1): a post-publish, honestly-labelled `final_local_sha256` evidence step, hard-failing on absent metadata, wired into `release.yml` between the release and attestation steps.
- The release itself (Task 2, checkpoint): `v0.7.0` cut via release-please (PR #48, `7b34ab1`), notarize pipe confirmed to have actually submitted to Apple (not skipped) via its own log lines, all five required repository secrets confirmed present.
- An unplanned but necessary prerequisite fix, discovered during the checkpoint: `test/wireoracle/capture.go` read captured stderr before the subprocess was joined, truncating `Transcript.Stderr` under a real race — fixed in issue #49 / PR #50 (`29f6e3d`) *before* cutting the release, because `verify:notarized-suite` (criterion 4's evidence source) depends on that harness.
- Post-release verification: automatic run (`31338004416`) and a manual-dispatch guard-check run (`31338409898`) both green across all 7 jobs, with per-job conclusions individually verified (not just the workflow's overall status) — disarming the exact "all-green while skipping everything" failure mode `workflow_dispatch`'s null `workflow_run` context creates.
- The maintainer's own Mac: a genuine Safari download of the darwin/arm64 asset launched with no Gatekeeper dialog, `spctl` reporting `accepted` / `source=Notarized Developer ID` verbatim — the phase's one unproxied observation.
- Task 3 (this continuation): recorded SIGN-02's GREEN Gatekeeper section with a RED-vs-GREEN table, SIGN-04's five-point byte-identity chain (identical 64-hex hashes on points 1–3 for both arches; points 4–5 recorded as bindings with an explicit, honest per-arch scope-limit rather than a fabricated fourth/fifth hash), the narrow assumption-A3 verdict, and criterion 4's 142-test notarized-suite proof (arm64-only, scope limit stated explicitly). Reconciled `docs/RELEASE.md`: both pending markers closed with citations to the settling evidence, and the missing chmod/execute-bit guidance added with the verbatim `fish` error a browser download otherwise produces.

## Task Commits

1. **Task 1: Make final_local_sha256 observable + pre-flight** — `de551be` (feat; original work commit `9361d8d`, squash-merged via PR #47)
2. **Task 2: Cut the release and observe every claim** — checkpoint, no code commit by the executor. Interlude fix required first: `29f6e3d` (fix, PR #50). The release itself produced `7b34ab1` (release-please's own `chore(main)` commit, PR #48) and tag `v0.7.0`.
3. **Task 3: Record criteria 2, 3 and 4 against the published release** — `698d19d` (docs)

**Plan metadata:** this SUMMARY.md (commit follows)

## Files Created/Modified

- `Taskfile.yml` (Task 1) — added `release:record-final-hashes`
- `.github/workflows/release.yml` (Task 1) — wired the new step in after release, before attestation
- `docs/RELEASE-PROCEDURES.md` (Task 1) — documented the post-publish failure branch and its `workflow_dispatch` recovery
- `.planning/phases/02-apple-signing-notarization/02-EVIDENCE.md` (Task 3) — appended SIGN-02 GREEN section, SIGN-04 five-point byte-identity section, criterion-4 section, and the manual-dispatch/maintainer's-machine observation subsection
- `docs/RELEASE.md` (Task 3) — closed both pending markers, corrected the reproducibility-posture claim using the non-reproducible-signature finding, added chmod/execute-bit guidance

## Decisions Made

- The PR merging the release-cutting branch used a `feat:` title deliberately, so the squash-merge's Conventional Commits parsing produced a MINOR bump (`0.6.0` → `0.7.0`) rather than repeating an earlier incident where a `build(ci):` title collapsed 17 `feat` commits into no version bump.
- The wireoracle stderr-capture race was fixed before the release, not after, because criterion 4's evidence source (`verify:notarized-suite`) depends on that harness reporting real subprocess output.
- SIGN-04 points 4 and 5 are recorded with an explicit scope-limit rather than implied uniform per-arch coverage: `verify:release-assets` (the command this plan's own `read_first` names as the binding source) verifies `cosign verify-blob` / `gh attestation verify` against `linux/amd64` only; darwin/amd64 has no binding check anywhere in the pipeline (the `self-upgrade` job's matrix is `darwin/arm64` + `linux/amd64` only, and darwin/arm64's own binding check inside that job reported only a green conclusion, not a captured verbatim transcript).
- `docs/RELEASE.md`'s reproducibility-posture pending marker was corrected, not just closed: its prior wording ("only someone holding this project's actual Developer ID Application certificate can reproduce it bit-for-bit") is contradicted by the rehearsal's own measurement that the *final* signed/notarized binary is not byte-reproducible across separate signing operations even by the certificate holder, because Apple's notarization service embeds a per-operation trusted timestamp. The doc now states this correctly and points readers to compare pre-sign builds, never final signed ones.

## Deviations from Plan

### Auto-fixed Issues

None during Task 3 itself — Task 3 was pure evidence recording and documentation reconciliation against values already observed and supplied verbatim in the checkpoint transcript; no code was written or modified.

**Note on Task 2's interlude (not a Task 3 deviation, recorded for completeness):** the wireoracle stderr-join race fix (issue #49, PR #50, `29f6e3d`) was Rule 1 (auto-fix bug) work performed during the Task 2 checkpoint, before this continuation began. It is documented here because it materially affected what evidence Task 3 could later cite (criterion 4's `notarized-suite` job would otherwise have produced unreliable stderr assertions).

---

**Total deviations (Task 3):** 0
**Impact on plan:** None — Task 3 executed exactly as specified against the supplied checkpoint evidence.

## Issues Encountered

None during Task 3. The one substantive finding — that `verify:release-assets`'s cosign/attestation bindings are scoped to `linux/amd64` only, not matrixed over darwin — was discovered by reading `Taskfile.yml`'s source directly (per this plan's own `read_first` instruction) rather than assuming per-arch coverage from the plan's five-point framing. It is recorded as an honest scope-limit in `02-EVIDENCE.md` rather than worked around or silently generalized.

## User Setup Required

None — no external service configuration required.

## Known Stubs

None.

## Threat Flags

None — this task recorded evidence and reconciled documentation; it introduced no new network endpoints, auth paths, file access patterns, or schema changes at a trust boundary.

## Next Phase Readiness

- ROADMAP Phase 2's criteria 2, 3, and 4 are recorded and closed against the published `v0.7.0` release; combined with criterion 1 (plan 02-01) and criterion 5 (plan 02-05), the phase goal holds: a macOS user who downloads a release asset in a browser can run it without Gatekeeper blocking them, proven by a check the project has already watched fail (the `v0.5.1` RED baseline) and now watched pass.
- `docs/RELEASE.md` has no remaining unexplained `pending` markers (`rg -c 'pending' docs/RELEASE.md` → 0).
- Phase 3 (Homebrew cask distribution) can build on a notarized, cosign-signed, attested `v0.7.0` as its first real distributable release; no blockers identified.
- Residual, explicitly-recorded (not hidden) scope gaps for a future plan to consider, if ever load-bearing: darwin/amd64 has no cosign/attestation binding check anywhere in the pipeline, and criterion 4's executed-suite proof is darwin/arm64 only with no amd64 or Rosetta leg.

## Self-Check: PASSED

- `.planning/phases/02-apple-signing-notarization/02-EVIDENCE.md` confirmed present and containing the three new sections (`SIGN-02 — GREEN Gatekeeper verdict on the published release`, `SIGN-04 — five-point byte identity on the published release`, `Criterion 4 — the suite against the notarized binary`).
- `docs/RELEASE.md` confirmed present, `rg -c 'pending'` returns 0.
- Commit `698d19d` confirmed present in `git log --oneline -1`.
- `rg -c 'SIGN04-PUBLISH-EVIDENCE|GATEKEEPER-EVIDENCE' .planning/phases/02-apple-signing-notarization/02-EVIDENCE.md` → 8.
- `go test ./internal/upgrade/ -v` and `task test:unit` both exit 0.
- `gh release list --repo seanb4t/codegraph-go --limit 10` confirms `v0.5.1` and `v0.5.0` are still listed — no release or tag deleted or re-pushed.

---
*Phase: 02-apple-signing-notarization*
*Completed: 2026-08-09*
