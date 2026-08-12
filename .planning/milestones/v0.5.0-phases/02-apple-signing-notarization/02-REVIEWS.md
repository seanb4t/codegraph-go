---
phase: 2
cycle: 3
reviewers: [codex, pi]
reviewed_at: 2026-08-09T15:27:56Z
plans_reviewed: [02-01-PLAN.md, 02-02-PLAN.md, 02-03-PLAN.md, 02-04-PLAN.md, 02-05-PLAN.md, 02-06-PLAN.md, 02-07-PLAN.md]
---

# Cross-AI Plan Review — Phase 2 (Cycle 3, FINAL)

Plans reviewed at commit `cd3e78b`. Both lanes were briefed that CodeGraph is indexed
here (430 files / 4,333 nodes / 10,453 edges, synced immediately before invocation) and
that D-19, D-04, D-18 and the 02-02 Task 2 one-way/no-checkpoint deviation are settled.
Neither lane re-opened any of them. Both lanes were also told which cycle-2 fixes were
the least-reviewed material and asked to scrutinise those hardest.

**Injection verified:** the CodeGraph briefing was appended to both
`gsd-review-prompt.md` and `gsd-review-instructions.md` (1 occurrence each, confirmed
with `rg -c` before either reviewer was invoked).

**Orchestrator verification note.** Codex's single HIGH is a specific, mechanically
checkable claim, so it was re-verified at source before being recorded. It is
**CONFIRMED** — see "Orchestrator source verification" below the two reviews.

## Codex Review

# Cross-AI Plan Review — Cycle 3

## Overall assessment

The phase is exceptionally thorough and the cycle-2 fixes are mostly sound. The digest asymmetry, Darwin double-build, resolved-tag checkout distinction, and accepted self-upgrade exposure are all defensible as written.

One genuine blocker remains in plan 02-04: the proposed mis-order mutation changes only `signs:` to `binary_signs:` while retaining `ids: [raw]`. Those two pipes interpret IDs in different namespaces. The mutated pipe will match zero build artifacts rather than sign the pre-notarization binaries, so the required inversion experiment cannot succeed. Overall risk remains **HIGH until this mutation is corrected**; after that, the phase is reasonable to proceed.

---

## Plan 02-01 — Gatekeeper RED tracer

### Summary

This is a strong tracer plan. It uses the corrected D-19 oracle, downloads the preserved published artifact, proves quarantine presence, classifies the result using exit status, and deliberately distinguishes evidence from observations. The RED/GREEN digest asymmetry is a visible compatibility concession rather than a hidden fail-open.

### Strengths

- Reuses the established published-asset download pattern rather than inspecting `dist/`; the existing verifier already downloads assets to a temporary directory at `Taskfile.yml:1050-1061`.
- Correctly recognizes that verification must survive an expected rejection. That is necessary because ordinary `set -e` handling would terminate before evidence could be recorded.
- The digest policy is appropriately stricter for the future GREEN claim. Existing release verification already treats missing assets or checksum coverage as fatal (`Taskfile.yml:1035-1047`, `1063-1086`).
- Named Taskfile preconditions and a real YAML-decoder shape test fit existing repository patterns.
- The synthetic-versus-browser comparison is a useful empirical closure of the remaining quarantine-rig assumption.

### Concerns

- **LOW:** The relationship between `observed` and `source_assertion` should be specified more mechanically. The plan says a disagreement fails, but a RED result should naturally have `observed=rejected` and `source_assertion=fail`. That pair is expected, not contradictory. Without an explicit truth table, an executor could accidentally make the RED target fail because the notarized source line is absent.
- **LOW:** The GREEN-path digest requirement depends on the runner’s installed `gh` exposing asset digests. This is intentionally fail-closed and therefore acceptable, but it should be proven before the irreversible release rather than first discovered afterward.

### Suggestions

- Add a two-row assertion table:

  | Observed | Required source assertion |
  |---|---|
  | accepted | pass |
  | rejected | fail |

- Exercise the real runner’s `gh release view --json assets` digest support during plan 02-01 or the plan 02-07 preflight.
- Keep the RED-only `digest_match=unknown` downgrade. It does not weaken the GREEN gate and is defensible for an older preserved baseline.

### Risk Assessment

**LOW.** The core mechanism is sound. Remaining concerns are specification clarity and early environment confirmation.

---

## Plan 02-02 — GoReleaser signing and notarization configuration

### Summary

The plan correctly targets the actual configuration defect: `binary_signs:` currently signs build artifacts before notarization, while the locked D-18 shape moves cosign to release-scoped `signs:`. The exact ID-set checks, five-credential conjunction, omission of entitlements, and caller enumeration are well designed.

### Strengths

- The current configuration really does use `binary_signs:` and contains the now-retracted rationale (`.goreleaser.yaml:180-231`), so the planned correction addresses live source rather than a hypothetical state.
- Exact ID scoping is justified by the repository’s distinct namespaces:

  - Darwin build IDs are `codegraph-darwin-amd64` and `codegraph-darwin-arm64` (`.goreleaser.yaml:90-105`).
  - The release-scoped sign pipe filters `UploadableBinary` artifacts and then applies `ByIDs` (`goreleaser/internal/pipe/sign/sign.go:113-126`).
- The five-way credential conjunction is stronger than checking only the certificate and directly addresses partial-credential enablement.
- Retargeting the existing sidecar-contract test is better than deleting it. The current test explicitly protects `internal/upgrade`’s sidecar naming contract (`internal/upgrade/goreleaser_shape_test.go:487-516`).
- The plan correctly records that existing `goreleaser build` callers currently assume `binary_signs:` reachability; the stale assumptions are visible in `Taskfile.yml:289-308`.
- Deferring PR-canary rewiring is sensible until the `enabled:` guard has been observed in both directions.

### Concerns

- **LOW:** `artifacts_this_phase_produces` still lists `MACOS_NOTARIZE_ENABLED` “only if” the explicit-flag option is chosen, although Task 2 explicitly rejects that option. This is contradictory residue and may confuse execution or artifact accounting.
- **LOW:** Static template evaluation with an injected predicate remains only a backstop, as the plan correctly admits. The runtime rehearsal is therefore essential and must not be waived.

### Suggestions

- Remove `MACOS_NOTARIZE_ENABLED` from the artifact list entirely.
- Have the caller-enumeration test or summary distinguish:

  - build commands, where `signs:` is unreachable but `notarize:` is reachable;
  - release commands, where both are reachable;
  - check commands, where neither executes.

### Risk Assessment

**LOW.** The planned configuration and tests are well aligned with the pinned GoReleaser behavior.

---

## Plan 02-03 — Published-binary test seam

### Summary

The resolver design is appropriately small and fail-closed. It makes criterion 4 reachable without silently rebuilding source when the published binary is missing or invalid.

### Strengths

- Both current harnesses unconditionally invoke `go build` (`test/integration/main_test.go:39-56`, `test/wireoracle/main_test.go:21-38`), so the seam closes a real gap.
- Absolute-path resolution is necessary because integration tests can change working directories.
- Rejecting missing files, directories, and non-executable regular files prevents the most likely configuration mistakes.
- The wrapper-script proof positively identifies the executable actually spawned; this is materially stronger than timing or build-cache inference.
- Keeping the downloaded file at its original path preserves the link between its recorded hash and executed bytes.
- Measuring wireoracle compatibility before adding it to criterion 4 avoids casually refreezing transcripts.

### Concerns

- **LOW:** The `must_haves` correctly say unset behavior is behaviorally equivalent, but Task 1’s `<done>` still says it is “byte-identical to today.” That is impossible after restructuring `TestMain` and contradicts the plan’s own correction.
- **LOW:** The suggested “make `go` unavailable using PATH” proof can also hide other tools needed by integration tests. The `GOFLAGS` sabotage option is cleaner because it targets only an accidental build.

### Suggestions

- Change Task 1’s `<done>` from “byte-identical” to “behaviorally equivalent.”
- Prefer a deliberately invalid build-only `GOFLAGS` value for the no-build proof, while leaving the environment otherwise unchanged.
- If the two resolver implementations begin to drift during execution, compare them mechanically in tests; extraction is still unnecessary for only two consumers.

### Risk Assessment

**LOW.** The implementation surface is small and has strong positive and negative identification tests.

---

## Plan 02-04 — Local notarization rehearsal and ordering mutation

### Summary

The cycle-2 double-build correction is sound: two identical notarization-disabled `release --snapshot` invocations isolate Darwin GoReleaser determinism, and the separate `BASELINE-NONDETERMINISTIC` and `SUBJECT-INDETERMINATE` labels make the failure classes distinguishable. However, the mutation itself is currently specified incorrectly and will match no binaries.

### Strengths

- The plan correctly refuses to borrow confidence from the standing reproducibility job. That job covers blocking Linux/amd64 only and makes Linux/arm64 non-blocking (`.github/workflows/ci.yml:216-273`); there is no Darwin leg.
- Using the same `goreleaser release --snapshot --skip=publish --clean` command for baseline and notarized runs avoids version/ldflag differences caused by comparing different command modes. The release config embeds tag, commit, and date (`.goreleaser.yaml:62-69`), so this correction matters.
- Double-building the disabled baseline before any subject verification genuinely separates:

  - build nondeterminism → `BASELINE-NONDETERMINISTIC`;
  - inability to identify which candidate cosign signed → `SUBJECT-INDETERMINATE`.

- Cryptographically verifying the bundle against both candidate files is the right oracle. Artifact metadata does not independently preserve the pre-mutation subject.
- Limiting the mutation to one architecture is a reasonable Apple-submission budget optimization because the pipe-order property is not architecture-specific.
- The plan honestly records that local rehearsal cannot close the SLSA or publish-path portions.

### Concerns

- **HIGH — blocking:** The mutation says to change only the top-level key from `signs:` to `binary_signs:` while leaving every other line, including `ids: [raw]`, untouched. That does not reproduce the pre-ruling configuration.

  `signs:` interprets `ids: [raw]` against uploadable archive artifacts (`goreleaser/internal/pipe/sign/sign.go:113-126`). In contrast, `binary_signs:` first filters raw build artifacts of type `Binary` and then applies its IDs against their build IDs (`goreleaser/internal/pipe/sign/sign_binary.go:66-84`). Those IDs are `codegraph-linux-*` and `codegraph-darwin-*`, not `raw` (`.goreleaser.yaml:45-105`). The current pre-ruling `binary_signs:` block consequently has no `ids:` field at all (`.goreleaser.yaml:223-231`).

  Therefore, the proposed mutated configuration will select zero binary subjects. It will not create the expected pre-notarization sidecar, so the experiment will end in `SUBJECT-INDETERMINATE` rather than showing the required inversion.
- **MEDIUM:** Because the mutation’s diff guard currently requires only one key-name change, it actively prevents the necessary removal of `ids: [raw]`. This converts the HIGH issue from an executor ambiguity into a plan-enforced failure.
- **LOW:** The plan says the generated configurations differ only in the notarize predicate, but the shipped/mutation configurations necessarily differ in signing-pipe shape too. The wording should clearly distinguish baseline-vs-enabled config comparison from shipped-vs-mutated config comparison.

### Suggestions

- Change the mutation transformation to exactly reproduce the pre-D-18 signing semantics:

  ```diff
  -signs:
  +binary_signs:
     - cmd: cosign
  -    ids: [raw]
       artifacts: binary
  ```

- Update the mutation diff guard to permit exactly two semantic edits:

  1. `signs:` → `binary_signs:`
  2. removal of `ids: [raw]`

- Add a pre-Apple mutation validation that parses the generated config and proves:

  - one `binary_signs:` entry exists;
  - no `signs:` entry exists;
  - the binary-sign entry has no ID filter;
  - its signature template and arguments remain unchanged.

- Clarify that only the two notarization-disabled baseline configs must differ solely in the `enabled` predicate. The mis-order mutation is a separate generated config with the two changes above.

### Risk Assessment

**HIGH.** This is a direct blocker: the required D-07 inversion experiment cannot work under the currently specified mutation.

---

## Plan 02-05 — Release documentation

### Summary

The documentation plan is careful about temporal truth: it avoids claiming notarization before the first notarized release exists, scopes the guarantee with boundary rows, and gives users the measured D-19 procedure rather than upstream boilerplate.

### Strengths

- The applicability table avoids making a false present-tense guarantee before plan 02-07 completes.
- The exact phrase “notarized, online-verified, not stapled” preserves the locked guarantee without implying offline safety.
- The plan properly distinguishes detached Sigstore verification from embedded Apple signing.
- Conditioning the quarantine instructions on the measured A2 result prevents publishing an unverified synthetic procedure.
- The not-verification list explicitly protects against the same misleading checks that affected planning.
- The reproducibility boundary is correctly narrowed: unsigned build determinism is different from reproducing Apple-signed Darwin bytes.

### Concerns

- **LOW:** The objective still says “D-02 decides … the assessment and the distribution-policy check,” reflecting the superseded pre-D-19 wording. The task body correctly says `syspolicy_check` is non-gating and should not be presented as a passing verification step.
- **LOW:** A “first notarized release: pending” table may temporarily be awkward on the default branch, but the explicit scoping makes it honest and plan 02-07 owns its replacement.

### Suggestions

- Rewrite the stale objective sentence to say D-19 selects `spctl -t install` and demotes `syspolicy_check` to an explanatory non-verification observation.
- Prefer one compact note about `syspolicy_check`; avoid repeating its failure mechanics in several parts of the document.

### Risk Assessment

**LOW.** The documentation plan accurately reflects the intended evidence ceiling.

---

## Plan 02-06 — CI secret wiring and post-release verification

### Summary

The release-secret containment and new post-release jobs are thoughtfully designed. Checking out the resolved tag for the notarized suite is necessary, while leaving the existing verifier jobs on the latest default-branch verifier is a sound distinction rather than rationalization.

### Strengths

- The current workflow explicitly documents that a `workflow_run` checkout defaults to the default branch tip (`.github/workflows/post-release-verify.yml:55-66`). Pinning the suite checkout to the resolved tag is therefore essential.
- The distinction between job types is sound:

  - `verify-supply-chain` and `self-upgrade` execute verifier targets from the repository (`.github/workflows/post-release-verify.yml:204-235`, `237-280`); using the newest verifier is useful for historical re-verification.
  - The notarized suite must pair tests@tag with binary@tag because it makes a compatibility claim about that release.

- In-job cosign verification before `chmod` closes a real execution-order hazard. The existing verifier’s authoritative issuer and identity flags are visible at `Taskfile.yml:1105-1114`.
- Adding `needs: verify-supply-chain` as an additional graph-level signal is appropriate, and the plan honestly records the resulting loss of independent diagnostics.
- Counting `go test -json` test events avoids mistaking an empty or cached run for successful coverage.
- Runtime enumeration of workflows strengthens the credential-scope test against future files.
- The planned test distinguishes workflow-name consumption from GitHub dashboard secret scope; that limitation is important.
- D-11’s event-aware guard is grounded in the live workflow, where every existing job uses it (`.github/workflows/post-release-verify.yml:103`, `212`, `248`).

### Concerns

- **MEDIUM:** `TestReleaseArtifactJobsCheckOutResolvedTag` is described as governing every release-artifact job, but the plan deliberately excludes two such jobs. Its name and stated invariant are broader than its intended scope. A future maintainer could reasonably infer that the test covers `verify-supply-chain` and `self-upgrade`, when it intentionally does not.
- **LOW:** The suite job’s dependency on `verify-supply-chain` may skip criterion 4 on an unrelated SBOM or Linux-side verification failure, even though its in-job cosign check would make Darwin execution safe. This is an explicitly accepted diagnostic tradeoff, not a blocker.
- **LOW:** Architecture selection remains an execution-time decision. This is acceptable, but the native arm64 leg must remain mandatory even if amd64 translation is unavailable.

### Suggestions

- Rename the test to reflect its real scope, such as `TestReleaseSuiteJobsCheckOutResolvedTag`, or encode explicit policy classes:

  - latest-verifier jobs;
  - release-matched-test jobs.

- Put that policy classification in the workflow comments and test data so adding a future job requires choosing one category.
- Keep the pre-existing checkout-less jobs unchanged. Their latest-verifier behavior is useful and independently justified.
- Preserve native Darwin/arm64 as the minimum criterion-4 execution surface.

### Risk Assessment

**MEDIUM.** The architecture is sound; the main residual risk is a misleadingly broad test name/invariant rather than a functional defect.

---

## Plan 02-07 — Irreversible release and final evidence

### Summary

This plan correctly treats publication as irreversible, makes the final local bytes observable under an honest name, and distinguishes direct hashes from cryptographic bindings. Its documented post-publish failure branch is particularly important because a failed hash-recording step suppresses automatic post-release verification.

### Strengths

- Renaming the observation to `final_local_sha256` avoids falsely claiming a mid-pipe measurement that GoReleaser does not expose.
- Making missing artifact metadata fatal is correct because publication has already happened; a silent continuation would leave SIGN-04 unprovable.
- The plan correctly notices the workflow consequence: post-release verification only proceeds from a successful upstream run (`.github/workflows/post-release-verify.yml:76-78`, `103`), so a post-publish recording failure requires manual dispatch.
- The five-point evidence model is accurate:

  - three directly comparable hashes;
  - cosign and attestation as cryptographic bindings, not invented hash observations.

- The re-download requirement follows existing practice: the repository’s current verifier explicitly downloads every raw and zip payload (`Taskfile.yml:1050-1061`) before checksum, cosign, and attestation checks.
- The plan keeps the browser-launch observation distinct from proxy measurements.
- The config-hash preflight meaningfully binds the rehearsal to the configuration about to publish.

### Concerns

- **MEDIUM:** This plan depends on plan 02-04’s mutation evidence. Until that plan’s ID-namespace bug is corrected, the preflight cannot legitimately declare the D-07 ordering proof complete.
- **LOW:** If `release:record-final-hashes` fails, the attestation step also does not run because it follows the failing step. Manual post-release verification will then correctly fail the attestation leg, but the runbook should state explicitly that patch-forward is required not only for missing hash evidence but potentially for a missing attestation.
- **LOW:** The “snapshot exercise” proves artifact selection logic only for empty-tag naming. The plan acknowledges the real-tag gap; ensure the real release evidence treats failure to locate tagged artifacts as fatal and does not reuse snapshot-derived paths.

### Suggestions

- Make plan 02-04’s corrected inversion evidence an explicit preflight prerequisite, not merely one item among general gates.
- In the post-publish failure runbook, distinguish:

  - assets published but final hashes unavailable;
  - attestation never executed;
  - manual dispatch expected to expose both;
  - new patch release required if either public supply-chain claim is absent.

- Preserve the narrow A3 conclusion: matching final-local and downloaded hashes proves no post-record publish-path mutation, not the behavior of every internal pipe boundary.

### Risk Assessment

**MEDIUM**, conditional on fixing plan 02-04. The irreversible flow is carefully managed, but it cannot proceed safely without a valid mutation experiment.

---

# Final proceed-or-stop assessment

## Blocking

- **Plan 02-04 must be revised before execution.** The mis-order mutation must remove `ids: [raw]` when moving from `signs:` to `binary_signs:`. Otherwise the build-scoped pipe filters for a nonexistent build ID and signs nothing.

## Acceptable recorded residue

- The five-point chain closing only after irreversible publication.
- Darwin GoReleaser determinism being measured by the rehearsal’s own double-build rather than standing CI.
- GREEN requiring a GitHub asset digest while the historical RED baseline may continue with a prominently labelled `unknown`.
- Existing verifier jobs checking out the latest verifier while the suite job checks out tests from the resolved tag.
- The pre-existing self-upgrade job downloading and executing an unverified prior binary (`Taskfile.yml:1279-1291`) being accepted as out-of-phase debt. This should receive a concrete follow-up issue/todo, but it does not block this notarization phase because the new suite job does not repeat the exposure.

**Overall risk: HIGH until the plan 02-04 mutation is corrected; MEDIUM after correction.**

---

## pi Review

# Cross-AI Plan Review — Phase 2: Apple Signing & Notarization (Cycle 3, Final)

**Verified against source at `cd3e78b`.** I read `.goreleaser.yaml:180-232`, `Taskfile.yml:1105-1114` + `1260-1291`, `ci.yml:250-275`, `post-release-verify.yml:205-275`, `release.yml:90-189`, `test/integration/main_test.go:39-57`, `test/wireoracle/main_test.go`, and `internal/upgrade/verify_release_e2e_test.go:94-133`. Every load-bearing claim in the plans that I checked matched the repo.

## Summary

This is a well-converged plan set. The cycle-3 changes address cycle-2's findings with mechanisms that actually hold up under source verification: the determinism-gap claim about `ci.yml` is accurate, the `self-upgrade` exposure in T-02-24 is real and correctly characterized, and the checkout-ref distinction in 02-06 is sound rather than rationalized. The two HIGHs from cycle 2 (mid-pipe hash capture impossibility; execute-before-verify ordering) are fixed with real mechanisms — a commensurate double-build baseline with its own failure label, and in-target `cosign verify-blob` before `chmod`. I found no new HIGH or MEDIUM issues. What remains is honestly-stated irreducible residue plus a small set of LOW observations.

## Strengths

- **02-04's two failure labels are genuinely distinguishable.** `BASELINE-NONDETERMINISTIC` halts *before* any `cosign verify-blob` invocation, so `SUBJECT-INDETERMINATE` can only fire with build nondeterminism already excluded. The claim that no standing gate covers darwin goreleaser determinism is verified: `ci.yml:250-275` runs `task check:reproducibility` (raw `go build`, linux/amd64 blocking, `ci.yml:269` arm64 `continue-on-error: true`, no darwin leg). The plan's framing is accurate, not overstated.
- **T-02-24 is a real exposure, correctly accepted.** `Taskfile.yml:1279-1291` downloads the prior release asset, `chmod +x`es it, and executes `"${PRIOR_BIN}" upgrade` with no verification before first execution — the exact hazard T-02-19 fixes for the new job. Recording it as accepted pre-existing debt (rather than silently leaving it or scope-creeping a fix) is the right call; the *upgrade itself* verifies in-process, which bounds the blast radius to the first exec of bytes GitHub served over TLS.
- **02-06's checkout distinction is sound.** Verified: both pre-existing jobs (`post-release-verify.yml:216` verify-supply-chain, `:262` self-upgrade) check out with no `ref:` and run *verifier targets* — newest verifier against an older tag is desired. The suite job runs *tests*, where tests@main vs binary@tag would make red ambiguous. Different properties, different treatment, recorded in-plan. Not rationalization.
- **02-01's digest asymmetry is defensible.** Fatal-on-GREEN / visible-downgrade-on-RED is not a fail-open: the RED path is a one-time recorded baseline against v0.5.1 (whose API digest may predate per-asset digests), the downgrade emits `digest_match=unknown` in the evidence, and the GREEN path — the claim that carries weight — refuses to proceed without provenance. The asymmetry is bound to `GATEKEEPER_EXPECT`, not ambient.
- **Current-state claims check out.** The false `signs:` rationale the plans retract is verbatim at `.goreleaser.yaml:183-186`; `binary_signs:` at `:223` still sits pre-notarize; `test/integration/main_test.go:39-57` and `test/wireoracle/main_test.go` both hardcode `go build` with no override seam; the `CODEGRAPH_E2E_BINARY` precedent exists at `verify_release_e2e_test.go:94-105`; the cosign flags 02-06 reuses are verbatim at `Taskfile.yml:1108-1113`; the event-aware guard appears exactly 4× in `post-release-verify.yml` and the count assertion will catch regressions.

## Concerns

- **LOW — 02-04 double-build output-directory mechanics are under-specified.** "Two distinct output directories" implies `--dist` on at least one run, but `goreleaser release --snapshot --clean` defaults to wiping `dist/`. An executor copying `dist/` aside between runs is fine; an executor passing `--dist` differently per run is fine; but the acceptance criteria don't pin which. Executor-recoverable, not a blocker.
- **LOW — 02-06's `needs: verify-supply-chain` makes the suite job's failure modes asymmetric with the Gatekeeper job's.** Gatekeeper job is deliberately independent (assesses, doesn't execute); suite job depends (executes). A supply-chain failure skips the suite, so a bad release reports Gatekeeper-red but suite-skipped — the plan records this tradeoff explicitly, and safety outranks breadth here. Consistent with the recorded verdicts; noting only because a future reader might see the skip as a gap.
- **LOW — `TestReleaseArtifactJobsCheckOutResolvedTag` uses an explicit job-id allowlist.** A new release-artifact job added later won't be covered until someone extends the list — the inverse of the runtime-enumeration discipline `TestAppleSecretsScopedToSingleReleaseJob` applies to workflow files. The plan acknowledges the scoping; a one-line doc-comment nudge ("add new artifact-executing jobs here") would close it.
- **LOW — 02-01 step 3a assumes the runner's `gh` version exposes per-asset `digest`.** Handled correctly on both paths (fatal GREEN / sentinel RED), so this is purely an executor-note: if v0.5.1's digests are absent, the RED baseline will legitimately record `digest_match=unknown`, and that's acceptable by design.
- **Irreducible residue — confirmed stated, not re-raised:** (a) the full five-point SIGN-04 chain first closes on an irreversible publish, acknowledged in 02-04 checkpoint step 6 with an explicit maintainer sign-off; (b) darwin goreleaser determinism is measured only by 02-04's own double-build, stated in its must_haves. Both are honestly carried.

## Suggestions

- In 02-04, pin the two-baseline mechanics in one acceptance line: name whether the second run uses `--dist` or a copy-out, so two executors don't diverge.
- Add the "extend this list when adding artifact-executing jobs" comment to `TestReleaseArtifactJobsCheckOutResolvedTag` (02-06 already requires a doc comment; one more sentence).
- When Phase 3 lands `homebrew_casks:`, re-run the A3 pipe-ordering check — the plan already flags this as a Phase 3 research item; just don't let it drop.

## Risk Assessment

**LOW.** All cycle-1 and cycle-2 HIGHs are resolved with mechanisms I could trace to real repo anchors; the D-19 oracle is measured and locked; the one irreversible step (02-07 publish) is preceded by a guarded rehearsal, a five-point evidence chain, three independent observers for the silent-skip failure, and a documented patch-forward recovery. Nothing here should block the maintainer's proceed decision. The residue is recorded, not hidden.

---

## Orchestrator source verification

Codex's HIGH was re-derived from source rather than taken on trust. It holds, and the
mechanism is worse than Codex stated — the failure is **silent**.

1. **`raw` is an archive id, not a build id.** `.goreleaser.yaml:129` declares
   `archives: - id: raw`. The build ids are `codegraph-linux-amd64` (`:51`),
   `codegraph-linux-arm64` (`:71`), `codegraph-darwin-amd64` (`:90`),
   `codegraph-darwin-arm64` (`:102`). There is no build with id `raw`.

2. **The two sign pipes resolve `ids:` in different namespaces.** In
   `goreleaser/v2@v2.17.1`, `BinaryPipe.Run` (`internal/pipe/sign/sign_binary.go:80-84`)
   builds `filters := []artifact.Filter{artifact.ByType(artifact.Binary)}` and then
   appends `artifact.ByIDs(cfg.IDs...)` — matching **build** ids. The release-scoped
   `signs:` pipe applies `ByIDs` to uploadable archive artifacts, matching **archive**
   ids. `ids: [raw]` is therefore correct under `signs:` and matches nothing under
   `binary_signs:`.

3. **The current committed `binary_signs:` block carries no `ids:` field at all**
   (`.goreleaser.yaml:223-231`) — which is the reason the pre-ruling configuration
   worked, and confirms that a faithful mis-order mutation must *remove* `ids: [raw]`,
   not merely rename the key.

4. **A zero-match sign pipe does not fail — it warns and returns nil.**
   `sign()` at `internal/pipe/sign/sign.go:137-140`:
   `if len(artifacts) == 0 { log.Warn("no artifacts matching the given filters found"); return nil }`.

   So the mutation as specified in `02-04` Task 1 step 3 ("leaving every other line
   untouched", guard asserting "the ONLY difference beyond the key injection is that
   one key name") produces **no sidecar at all**. `cosign verify-blob` then fails
   against both candidates, the run terminates as `SUBJECT-INDETERMINATE`, and the D-07
   inversion — the entire evidentiary purpose of the mutation — cannot be observed.

   This is precisely the failure class `02-02`'s own `must_haves` prohibits: *"MUST NOT
   let a filter, template, or `enabled:` guard that matches nothing report success — a
   pipe or gate that no-ops must fail loudly or be asserted against."* The phase's
   central mutation experiment currently contains the defect the phase exists to
   eliminate.

**Correction required in `02-04`:** the mutation must apply **two** edits —
`signs:` → `binary_signs:` **and** removal of `ids: [raw]` — and the mutation-mode diff
guard must be widened to permit exactly those two and nothing else. A pre-Apple
structural validation of the generated mutated config (one `binary_signs:` entry, no
`signs:` entry, no id filter, unchanged signature template and args) should gate the
submission, so a zero-match mutation is caught before an Apple round-trip rather than
after.

**Minor correction to a pi citation:** pi reported the event-aware guard appearing
"exactly 4×" in `post-release-verify.yml`; the count at HEAD is **3**
(`rg -c` on the verbatim disjunct). 4 is the post-`02-06`-Task-2 expectation, which is
what `02-06`'s acceptance criterion ("one more than at HEAD") encodes. No finding
depends on the difference.

Every other load-bearing citation from both lanes was spot-checked and held:
`ci.yml:269` `continue-on-error: true` with no darwin leg anywhere in the file;
`Taskfile.yml:1279-1291` download → `chmod +x` → execute with no verification between;
`post-release-verify.yml:216` and `:262` both checkout-less; `CODEGRAPH_E2E_BINARY`
precedent at `internal/upgrade/verify_release_e2e_test.go:101`;
`02-02-PLAN.md:490` `MACOS_NOTARIZE_ENABLED` residue against the rejecting criterion at
`:291`; `02-03-PLAN.md:140` and `:154` "byte-identical" against the must_have at `:25`
that explicitly says "Not byte-identical"; `02-05-PLAN.md:54` stale `D-02` wording.

## Consensus Summary

Two grounded, prompt-fed, repo-reading lanes. Neither is diff-only; neither carried a
`reviewed-without-repo-access` marker. Both verified their claims against real files
and cited `path:line`.

The lanes agree on the shape of the phase and disagree on exactly one thing: whether
anything still blocks. pi found no HIGH and no MEDIUM and called overall risk LOW.
Codex found one HIGH — a defect in the `02-04` mutation that pi did not examine at the
`ids:`-namespace level. That is a depth difference, not a contradiction: pi verified
that the two *failure labels* are distinguishable and that the double-build is sound
(both true), while Codex verified what the mutation would actually *select* (broken).
The orchestrator re-derived Codex's claim from GoReleaser source and confirms it.

### Agreed Strengths

- **The cycle-2 HIGHs are genuinely fixed.** Both lanes independently confirm the
  mid-pipe-capture impossibility is resolved by a commensurate notarize-disabled
  `release --snapshot --skip=publish --clean` baseline, and that execute-before-verify
  is closed by in-target `cosign verify-blob` before any `chmod`.
- **`02-04`'s two failure labels are distinguishable.** Both lanes confirm
  `BASELINE-NONDETERMINISTIC` halts before any `cosign verify-blob`, so
  `SUBJECT-INDETERMINATE` can only fire with build nondeterminism already excluded.
- **The darwin determinism gap is stated accurately, not overstated.** Both lanes
  verified `ci.yml`: blocking linux/amd64, `continue-on-error` linux/arm64 at `:269`,
  no darwin leg, raw `go build` throughout. Both confirm the plan refuses to claim
  DIST-04 covers it.
- **`02-06`'s checkout-ref distinction is sound, not rationalizing.** Both lanes reached
  this independently: the two pre-existing jobs run verification *targets* (newest
  verifier against an older tag is the desirable behaviour); the suite job runs *tests*,
  where tests@main against binary@tag would make a post-release red ambiguous. Different
  properties, different treatment, recorded in-plan. **Answers the cycle-3 question: sound.**
- **`02-01`'s digest asymmetry is defensible, not a fail-open.** Both lanes agree. The
  downgrade is bound to `GATEKEEPER_EXPECT`, is a labelled visible sentinel
  (`digest_match=unknown`) rather than a silent pass, and the GREEN path — the claim
  that carries weight — is fatal. **Answers the cycle-3 question: defensible.**
- **`T-02-24` is a real exposure, correctly accepted.** Both lanes verified
  `Taskfile.yml:1279-1291` and both endorse recording it as accepted pre-existing debt
  rather than scope-creeping a fix into this phase. **Answers the cycle-3 question:
  accepting it is correct** — Codex adds that it should get a concrete follow-up
  artifact rather than only a prose flag.
- **`02-02`'s five-way credential conjunction** is stronger than certificate-only gating
  and directly closes partial-credential enablement; the separate-flag rejection is
  sound.
- The `02-03` seam's positive-identification proof (shim log) is materially stronger
  than timing inference; `02-07` correctly makes post-publish recording failures fatal.

### Agreed Concerns

- **The `TestReleaseArtifactJobsCheckOutResolvedTag` scope seam** — raised by both lanes
  from different angles. Codex (MEDIUM): the test's name and stated invariant claim to
  govern *every* release-artifact job while deliberately excluding two. pi (LOW): the
  explicit job-id allowlist won't cover a future artifact-executing job. Same seam. The
  fix Codex proposes — rename, or encode two explicit policy classes (latest-verifier
  jobs vs release-matched-test jobs) so adding a job forces a choice — resolves both.
- **Irreducible residue is confirmed STATED by both lanes**, and neither re-raised it as
  new: (a) the five-point SIGN-04 chain closes for the first time only on an irreversible
  publish; (b) darwin goreleaser determinism rests on `02-04`'s own double-build. Both
  lanes explicitly classify these as honestly-carried residue rather than defects.

### Divergent Views

- **Overall risk.** pi: LOW, nothing should block the maintainer's proceed decision.
  Codex: HIGH until the `02-04` mutation is corrected, MEDIUM after. The orchestrator
  sides with Codex on the blocking question and with pi on everything else — pi's LOW
  verdict is correct about every area pi examined, but pi did not examine the mutation's
  id-namespace semantics, which is where the one real defect lives. The defect is
  narrow, mechanical, and cheap to fix; it does not reopen any settled decision.
- **`02-04` double-build mechanics.** pi (LOW) notes "two distinct output directories"
  is unpinned between `--dist` and copy-out while `--clean` wipes `dist/`; Codex did not
  raise it. Confirmed unpinned at `02-04-PLAN.md:328`. Executor-recoverable but worth one
  acceptance line.
