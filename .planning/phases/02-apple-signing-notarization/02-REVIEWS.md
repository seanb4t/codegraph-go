---
phase: 2
reviewers: [codex, pi]
reviewed_at: 2026-08-09T10:27:00Z
plans_reviewed:
  - 02-01-PLAN.md
  - 02-02-PLAN.md
  - 02-03-PLAN.md
  - 02-04-PLAN.md
  - 02-05-PLAN.md
  - 02-06-PLAN.md
  - 02-07-PLAN.md
---

# Cross-AI Plan Review — Phase 2

Both lanes ran prompt-fed and source-grounded (neither carries the
`[reviewed-without-repo-access]` marker), and both were briefed that this
repository is CodeGraph-indexed. Neither lane was trimmed.

## Codex Review

# Cross-AI Plan Review — Phase 2

## Overall assessment

The phase is thoughtfully decomposed and unusually disciplined about false-positive verification. The sequencing—RED Gatekeeper baseline, config mechanics, real-binary seams, rehearsal, documentation, CI wiring, then irreversible release—is sound.

However, three design gaps should be fixed before execution:

1. Plan 02-04 cannot obtain a true “immediately after build” versus “immediately after notarize” hash merely by wrapping one opaque `goreleaser release` invocation.
2. Plan 02-06 executes a downloaded release binary independently and potentially concurrently with supply-chain verification.
3. Plan 02-07 labels a hash captured after the entire GoReleaser release as “immediately after notarize,” which it is not.

There is also a documentation-timing problem in 02-05: it can publish a present-tense notarization guarantee before any notarized release exists.

Overall risk: **HIGH until those four issues are corrected**, then **MEDIUM**, driven mostly by Apple credentials and the irreversible real-release gate.

---

# Plan 02-01 — Gatekeeper RED baseline

## Summary

This is a strong tracer plan. It grounds SIGN-03 in the permanently preserved `v0.5.1` assets, separates the non-quarantined control from real evidence, and explicitly resolves the two Apple-tooling uncertainties on real macOS. The proposed target fits the existing published-asset verification style.

## Strengths

- The published-asset-only rule matches the existing release verification posture: `verify:release-assets` is explicitly designed around re-downloading rather than inspecting `dist/` ([Taskfile.yml:924](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:924)).
- The asset name matches the upgrade contract exactly. `releaseAssetName` produces `codegraph_<version>_<goos>_<goarch>` with no archive suffix ([internal/upgrade/upgrade.go:209](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:209)).
- Recording the pre-xattr control separately is excellent. It directly prevents a never-quarantined assessment from being mistaken for SIGN-02 evidence.
- Comparing synthetic quarantine with a genuine browser download appropriately addresses the largest uncertainty in the test rig.
- The gate deliberately captures `spctl` stderr and nonzero status rather than letting `set -e` erase expected RED evidence.

## Concerns

- **MEDIUM — Verdict classification is underspecified.** Searching merged `spctl` output for literal `accepted` or `rejected` can misclassify diagnostic text containing those words. The plan captures the exit status but does not make it authoritative.
- **MEDIUM — The xattr write is described as “genuine” too early.** A synthetic four-field xattr is only a simulation until the browser comparison confirms assumption A2. The target and evidence should consistently call it synthetic.
- **LOW — Tool precondition asymmetry is awkward.** `jq` is required even though the described target body does not clearly need JSON processing. Conversely, `syspolicy_check` is optional because its existence is being researched. That may be intentional, but the resulting exact-set shape test will freeze `jq` even if implementation does not use it.
- **LOW — `GH_TOKEN` is stricter than existing local `gh` authentication needs.** The checkpoint already assumes `gh` authentication. Requiring `GH_TOKEN` is reasonable for CI reuse, but the rationale should say it is a reproducibility input, not a technical requirement of local `gh`.

## Suggestions

- Classify the Gatekeeper result using both exit status and a narrowly matched final verdict line. Reject contradictory results such as exit 0 with `rejected`.
- Keep evidence fields escaped or encoded. Raw tool output containing spaces should remain on separate labelled lines rather than inside the machine-readable record.
- Drop `jq` unless the implementation actually uses it.
- Record the downloaded asset’s GitHub asset ID and size alongside its hash to strengthen baseline provenance.

## Risk assessment

**MEDIUM.** The test design is good; remaining risk lies in interpreting undocumented Apple CLI behavior.

---

# Plan 02-02 — GoReleaser signing and notarization configuration

## Summary

The D-18 implementation is well aligned with the current repository. The existing config really does use `binary_signs:` and contains the now-retracted rationale that `signs:` requires a project-wide binary archive format ([.goreleaser.yaml:180](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:180), [.goreleaser.yaml:223](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:223)). Retargeting the existing resolved-name test rather than deleting it is the right approach.

## Strengths

- The plan preserves the sidecar contract consumed by `defaultDownload`, which downloads `assetName + ".sigstore.json"` ([internal/upgrade/upgrade.go:188](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:188)).
- The existing test already resolves the template for all four release pairs and enforces distinctness ([internal/upgrade/goreleaser_shape_test.go:507](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/goreleaser_shape_test.go:507)). Retargeting it preserves meaningful coverage.
- Exact-set assertions for notarize build IDs and signing archive IDs correctly defend against both silent underscope and accidental zip inclusion.
- The plan correctly treats YAML block position as documentation only, not execution ordering.
- The caller enumeration is justified by current stale assumptions. `check:darwin-release-build` presently says every `goreleaser build` reaches `binary_signs:` and requires cosign ([Taskfile.yml:270](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:270), [Taskfile.yml:308](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:308)).
- Deferring PR-canary rewiring is prudent. The current canary explicitly avoids granting OIDC to a PR-triggerable workflow ([darwin-toolchain-canary.yml:71](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/darwin-toolchain-canary.yml:71)).

## Concerns

- **MEDIUM — The `enabled:` test may not faithfully emulate GoReleaser.** A locally invented `text/template` FuncMap can prove the expected predicate but not necessarily GoReleaser’s precise environment/template evaluation behavior. This is static backstop coverage, not runtime proof.
- **MEDIUM — Credential gating on only `MACOS_SIGN_P12` permits partial credentials.** If the certificate exists but any password/notary input is absent, the pipe becomes enabled and fails later. The rehearsal target checks all five, but ordinary snapshot callers with a partially populated environment remain exposed.
- **LOW — The plan edits Phase 1’s deferred-items file for a Phase 2 decision.** This is workable but weakens artifact ownership and may make the deferral harder to discover later.
- **LOW — The asserted rerun idempotency is only a config-level backstop.** `replace_existing_artifacts: true` is present ([.goreleaser.yaml:286](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:286)), but it does not prove Apple will accept repeat submissions of the same input.

## Suggestions

- Gate notarization on all required credential variables, or use a single explicit `MACOS_NOTARIZE_ENABLED` variable that is only set by guarded release/rehearsal targets.
- Add a config test that resolves `enabled` under partial credential sets and documents the chosen behavior.
- Record the canary deferral in the current phase’s deferred file, with a backlink from Phase 1 if desired.
- Ensure the new parser rejects multiple `notarize.macos` entries, not merely that the main test expects one.

## Risk assessment

**MEDIUM.** The core config change is well reasoned, but template behavior and partial credentials remain runtime hazards.

---

# Plan 02-03 — Published-binary test seams

## Summary

The need is real: both integration harnesses currently build the checked-out source unconditionally ([test/integration/main_test.go:39](/Volumes/Code/github.com/seanb4t/codegraph-go/test/integration/main_test.go:39), [test/wireoracle/main_test.go:21](/Volumes/Code/github.com/seanb4t/codegraph-go/test/wireoracle/main_test.go:21)). A strict override that cannot fall back silently is the correct design.

## Strengths

- The resolver’s two-outcome contract directly prevents a bad override from becoming a local rebuild.
- Converting relative input to an absolute path is necessary because integration tests execute from multiple working directories.
- The unset path preserves the existing hermetic build behavior.
- Mirroring the seam into wireoracle avoids an unexplained asymmetric harness contract.
- Deferring transcript changes if release metadata causes a mismatch respects the frozen-oracle discipline.

## Concerns

- **MEDIUM — Elapsed time does not prove that no `go build` occurred.** Machine load and Go cache state make timing a weak oracle.
- **MEDIUM — “Executable mode bit” does not prove executability on the running OS.** It rejects obvious errors but cannot prove architecture compatibility or that the file is a valid executable.
- **LOW — Duplicating resolver logic can drift.** The reason that `_test.go` helpers are not importable across packages is true, but a small non-test internal harness package could be imported by both.
- **LOW — Checking only mode bits can behave unexpectedly on non-Unix development hosts.** Native Windows support is dropped, but contributors may still run tooling in unusual environments.

## Suggestions

- Prove no build by supplying a wrapper executable whose output or invocation side effect is unique, or run with a deliberately unavailable `go` in `PATH` while the override is set.
- Add an end-to-end test using a tiny executable shim that records every invocation, proving the exact override path was spawned.
- Consider a shared `internal/testbin` resolver if this seam is likely to spread further.
- Preserve the override path rather than copying the file so the suite’s recorded sha256 stays joinable to CI evidence.

## Risk assessment

**LOW–MEDIUM.** The implementation is localized and testable; the main issue is strengthening the honesty proof.

---

# Plan 02-04 — Local notarization rehearsal and ordering mutation

## Summary

The intent is excellent, but the proposed implementation contains a fundamental observability gap. A Task wrapper around one `goreleaser release` process cannot naturally hash a binary “immediately after the build pipe” and again “after the notarize pipe.” GoReleaser owns those internal transitions.

## Strengths

- Named preconditions for all five Apple credentials directly implement D-09.
- Generated configuration and a surgical mutation avoid contaminating the committed `.goreleaser.yaml`.
- Rehearsing both correct and deliberately misordered configurations is a strong empirical test.
- The plan correctly distinguishes local rehearsal from the published-asset GREEN criterion.
- A deliberate wrong-password run is a useful secret-leak check.
- The use of `dist/artifacts.json` is consistent with existing practice; current release tasks already distrust exit status and verify artifact records ([Taskfile.yml:594](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:594), [Taskfile.yml:625](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:625)).

## Concerns

- **HIGH — The plan does not explain how it can observe the pre-notarize hash.** Step 5 invokes one opaque `goreleaser release`; step 4 claims hashes will be captured between internal pipes. A shell target cannot pause between `build.Pipe` and `notary.MacOS` unless it adds a GoReleaser hook, patches/builds an instrumented GoReleaser, or performs an independent equivalent build before release.
- **HIGH — The mutation may not prove the stated four-way relationship.** Under `binary_signs:`, the signature sidecar records a signature over pre-quill bytes, but `artifacts.json` after completion points at the mutated on-disk binary path. Rehashing that path after completion does not recover the bytes cosign actually saw. Verification against the post-notarization file can show divergence, but the plan must use `cosign verify-blob` or extract the bundle subject digest—not “the file the Signature record was computed over.”
- **MEDIUM — A snapshot’s tag-dependent asset names differ from a real tag.** The relationship can still be tested, but evidence must distinguish snapshot naming from release naming.
- **MEDIUM — The plan states that pending/time-out submission may log and continue without source-backed confirmation in the current repository.** This is an open behavior that the rehearsal should determine, not assume.
- **LOW — Repeated real Apple submissions increase operational cost/noise.** The plan asks for correct run, mutated run, wrong-password run, and an idempotency rerun. That is several Apple-facing executions.

## Suggestions

- Redesign the hash experiment around observable facts:

  - Create a separate deterministic pre-sign build of each Darwin binary and hash it.
  - Run GoReleaser sign/notarize and hash the final binary.
  - Under the misordered config, use `cosign verify-blob` against both the preserved pre-sign copy and final binary to prove which one the bundle authenticates.
  - Under the correct config, verify the bundle only against the final binary.

- Alternatively, instrument the pinned GoReleaser binary with explicit before/after hash logging, but make that mechanism part of the plan.
- Do not infer the cosign subject from `artifacts.json`; verify the bundle cryptographically.
- Reduce Apple submissions by performing the misordering proof on one Darwin architecture if that is enough to establish the mechanism, while retaining both-arch correct rehearsal.

## Risk assessment

**HIGH.** As written, the central SIGN-04 rehearsal evidence cannot be produced reliably.

---

# Plan 02-05 — Release documentation

## Summary

The documentation content is well scoped, but its execution timing conflicts with its present-tense guarantee. This plan runs before the first notarized release, yet its must-have says the document states that the shipped guarantee is “notarized, online-verified, not stapled.”

## Strengths

- The exact guarantee is appropriately bounded and includes the offline limitation.
- Requiring `xattr -p` before `spctl` prevents a misleading reproduction procedure.
- Distinguishing detached Sigstore verification from the embedded Apple signature is essential and consistent with the upgrade path, which downloads a separate `.sigstore.json` bundle ([internal/upgrade/upgrade.go:188](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:188)).
- Scoping the reproducibility claim to unsigned build output is necessary once Developer ID signing mutates Darwin assets.
- The plan correctly avoids claiming that notarization creates additional release assets; the current asset shapes are raw plus zip ([.goreleaser.yaml:128](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:128)).

## Concerns

- **HIGH — It can publish a false present-tense guarantee.** Plan 02-05 precedes 02-07, so the latest published Darwin assets remain deliberately un-notarized when the documentation change lands. The plan simultaneously requires the exact shipped guarantee and says unmeasured claims should be marked pending.
- **MEDIUM — “Apple notarization ticket” may imply attachment/stapling.** For an unstapled executable, the accepted notarization record exists with Apple; the file does not carry an attached ticket. Wording must be precise.
- **MEDIUM — The docs claim a synthetic xattr reproduces browser behavior only if plan 02-01 confirms A2.** The branch is mentioned, but the acceptance criteria still require the synthetic write command unconditionally.
- **LOW — Stating that bare Mach-O and zip are “categorically unstaplable” is accurate in phase context but unnecessary detail in user verification instructions.** It risks distracting from the actionable guarantee.

## Suggestions

- Split the documentation work:

  - Before release, document verification as applying “from the first notarized release onward,” with the tag marked pending.
  - In 02-07, replace the pending tag boundary with the actual release tag after evidence exists.

- Phrase the mechanism as “accepted by Apple’s notarization service; the unstapled file relies on online ticket lookup.”
- If A2 is refuted, document a real browser-download procedure rather than retaining synthetic xattr commands.
- Add a clear applicability table by tag rather than prose trying to cover empty `v0.5.0`, un-notarized `v0.5.1`, and the future notarized release.

## Risk assessment

**HIGH** for truthfulness if executed unchanged; otherwise **LOW** once the release boundary is explicit.

---

# Plan 02-06 — CI credentials and post-release verification

## Summary

The secret-scoping and event-guard work is strong, but the notarized-suite job creates a significant trust-ordering problem: it executes a downloaded binary while the independent supply-chain verification job may still be running or may fail.

## Strengths

- The release job is presently the only job with `id-token: write` ([release.yml:85](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:85), [release.yml:90](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:90)). Adding credentials only to its Release step limits exposure within the job.
- The workflow filename and tag-trigger identity remain untouched, preserving the verifier’s SAN contract.
- Runtime enumeration of every workflow file is much stronger than a fixed allowlist.
- The existing post-release workflow demonstrates the correct event-aware guard on every current job ([post-release-verify.yml:103](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/post-release-verify.yml:103), [post-release-verify.yml:212](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/post-release-verify.yml:212), [post-release-verify.yml:248](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/post-release-verify.yml:248)).
- Resolving the tag through `needs.resolve-tag.outputs.tag` follows the existing validated flow.
- The Gatekeeper job’s independence from supply-chain verification is defensible because it assesses rather than executes the file.

## Concerns

- **HIGH — The suite executes an unverified network-fetched binary.** The proposed job depends only on `resolve-tag`, just as the current `self-upgrade` job does ([post-release-verify.yml:245](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/post-release-verify.yml:245)). The threat model claims the same workflow independently verifies the asset, but parallel independent jobs provide no ordering guarantee. A tampered asset can execute even if `verify-supply-chain` later fails.
- **MEDIUM — Secret reference scanning does not prove actual GitHub secret scope.** It proves workflow consumption sites, not that names are repository-scoped rather than organization-scoped or that access policies are correct. The user setup handles this manually, so tests should not overstate what they prove.
- **MEDIUM — Running both Darwin architectures may not be feasible on the selected runner.** The plan acknowledges this, but criterion 4 says the full suite runs against “the binary itself” without defining required architecture coverage.
- **MEDIUM — Test-count detection from `go test` text is fragile.** Standard `go test` does not provide a simple total count line. Parsing verbose output can confuse cached/package summaries with executed tests.
- **LOW — Secrets are exposed to every process launched by the Release step.** That is unavoidable for GoReleaser, but the step should minimize shell logic and avoid dumping environment/config.

## Suggestions

- Make `notarized-suite` depend on both `resolve-tag` and `verify-supply-chain`, or perform cosign verification inside the same job before `chmod +x` and execution.
- If independence of reporting is desired, split download/verification from execution: the execution job should only run after integrity verification succeeds.
- Use `go test -json -count=1` and count `Action:"run"` or `Action:"pass"` test events instead of parsing human output.
- Add a test proving Apple secret references occur only under a step-level `env`, not job-level or workflow-level environment.
- Define architecture coverage explicitly: native arm64 is mandatory; amd64 is additional only if Rosetta execution is confirmed.

## Risk assessment

**HIGH.** Executing the artifact before its integrity verification completes is a material CI security flaw.

---

# Plan 02-07 — Real release and final evidence

## Summary

The final checkpoint is appropriately irreversible and evidence-oriented, but its “post-notarize” hash collection is mislabeled and deliberately non-failing in a way that can leave SIGN-04 impossible to prove after publication.

## Strengths

- Release-please remains the only tag authority.
- The checkpoint explicitly checks secret names before merging the release PR, addressing the silent-disabled-notarization risk.
- It observes both automated Gatekeeper results and an actual browser launch.
- Manual dispatch verifies that the event-aware guards do not turn a rerun into a vacuous green.
- The RED/GREEN comparison uses the same target and procedure, which is strong evidence.
- The plan correctly requires patch-forward recovery and preservation of earlier releases.

## Concerns

- **HIGH — `release:record-notarize-hashes` is not “immediately after notarize.”** It runs after the whole `goreleaser release` step finishes ([release.yml:170](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:170)) and before the separate attestation step ([release.yml:188](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:188)). It records final local bytes after archive, SBOM, checksum, release signing, and publish—not the boundary immediately after `notary.MacOS`.
- **HIGH — Making the recording target unable to fail undermines a required criterion.** If metadata is absent or one Darwin artifact is missing, the plan prints `NOT-FOUND` and continues publishing. Since this is the only claimed first comparison point, SIGN-04 may become permanently unprovable for that release.
- **MEDIUM — The four “sha256 values” are not all independently available.** `cosign verify-blob` and `gh attestation verify` prove that a downloaded subject matches signed/attested data; they do not necessarily print a separate subject hash. Task 3 acknowledges this, but the must-have still describes four hash values as if each is directly recorded.
- **MEDIUM — A local snapshot exercise does not prove the recording step will locate a real tagged release artifact under identical names.**
- **MEDIUM — The pre-flight config equality is underspecified.** It asks to compare the current config hash to what plan 02-04 measured, but 02-04 does not explicitly require recording the committed config’s sha256.
- **LOW — The checkpoint asks the user to merge a release PR and monitor multiple external workflows.** This is appropriate, but failure branches should explicitly stop evidence recording until the relevant workflow logs are downloaded and preserved.

## Suggestions

- Rename the captured value to `final-goreleaser-local-sha256`; do not call it immediate-post-notarize.
- Obtain the true immediate-post-sign/notarize hash through the corrected 02-04 instrumentation. Then compare:

  1. instrumented post-quill hash,
  2. final local raw asset hash,
  3. re-downloaded asset hash,
  4. cryptographic verification of the same hash by cosign and attestation.

- Make missing metadata or missing Darwin artifacts fail the release before attestation/publish if a reliable pre-publish hook exists. If publication already occurred inside GoReleaser, at minimum fail the workflow loudly rather than return success.
- Treat cryptographic verification as bindings to the re-downloaded hash, not invented additional hash observations.
- Record `.goreleaser.yaml` sha256 in the rehearsal evidence so pre-flight equality is executable.
- Make the suite execution wait for supply-chain verification as recommended for 02-06.

## Risk assessment

**HIGH.** The release checkpoint is correctly cautious, but the current hash-observation design does not establish SIGN-04 as stated.

---

# Cross-plan dependency and scope review

## What works well

- The dependency graph is mostly coherent:

  - 02-01 establishes the oracle.
  - 02-02 and 02-03 independently prepare pipeline and harnesses.
  - 02-04 rehearses pipeline behavior.
  - 02-06 wires CI after rehearsal.
  - 02-07 performs the irreversible release.

- The current repository supports the need for each change:

  - Signing is still build-scoped under `binary_signs:` ([.goreleaser.yaml:223](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:223)).
  - Release assets are raw and zip in parallel ([.goreleaser.yaml:128](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:128)).
  - Both test harnesses rebuild locally ([test/integration/main_test.go:47](/Volumes/Code/github.com/seanb4t/codegraph-go/test/integration/main_test.go:47), [test/wireoracle/main_test.go:29](/Volumes/Code/github.com/seanb4t/codegraph-go/test/wireoracle/main_test.go:29)).
  - Post-release verification already has validated tag resolution and non-vacuous guards.

## Cross-plan concerns

- **HIGH — SIGN-04’s measurement mechanism is inconsistent across 02-04 and 02-07.** Both assume access to a post-notarize boundary neither plan actually defines.
- **HIGH — Documentation is scheduled before the evidence it claims.**
- **HIGH — Integrity verification and execution are ordered incorrectly in 02-06.**
- **MEDIUM — The phase is over-specified in places.** Several plans devote extensive permanent tests to Taskfile precondition exact sets while the most critical runtime properties—Apple submission completion and post-quill byte identity—remain one-time manual observations.
- **MEDIUM — Evidence parsing lacks a formal schema.** Multiple shell-generated evidence prefixes are introduced, but quoting, escaping, missing values, and versioning are not defined. That makes later automated joining fragile.

## Recommended plan changes before execution

1. Amend 02-04 with a concrete mechanism for observing pre-quill and post-quill bytes and cryptographically identifying the subject cosign signed.
2. Amend 02-06 so no downloaded binary is executed before signature and attestation verification succeeds.
3. Split or condition 02-05 so the present-tense guarantee only applies from the first successfully verified notarized tag.
4. Rename 02-07’s hash point to final-local-release bytes, and make missing evidence fail loudly.
5. Define a minimal evidence-line format—fixed keys, shell-safe values, and explicit `unknown`/`not-found` states.

# Final risk assessment

**Overall risk: HIGH.**

The plans are unusually strong on traceability, RED-before-GREEN discipline, secret scoping, and preserving the raw upgrade contract. But SIGN-04—the most technically subtle requirement—currently relies on byte observations the described tooling cannot make, and the post-release suite may execute an artifact before its integrity is established. Those are goal-level and security-level problems, not documentation polish. Once corrected, the remaining phase risk should fall to **MEDIUM**, primarily because Apple notarization and Gatekeeper semantics can only be conclusively tested through real external systems and an irreversible release.

---

## pi Review

All key mechanism claims verified against source. Here is the review.

# Cross-AI Plan Review — Phase 2: Apple Signing & Notarization

## Summary

This is an exceptionally well-grounded plan set. Every load-bearing technical claim I checked against the pinned `goreleaser/v2@v2.17.1` module cache and this repo's files held up: the pipe-order finding (`sign.BinaryPipe{}` before `notary.MacOS{}` in `BuildPipeline`, `internal/pipeline/pipeline.go:102-106`), the `signs:` fix landing after `checksums.Pipe{}` (`pipeline.go:143`), the shared `signone()` Path/Name rebinding (`internal/pipe/sign/sign.go:182,275`), the `UploadableBinary` filter (`sign.go:114`), the `[ProjectName]` ids default and non-fatal skip (`internal/pipe/notary/macos.go:35-37,87-88`), and the skip-key asymmetry (`internal/skips/skips.go`: `Notarize` is in `Release`, absent from `Build`). Repo-side claims also check out: one `id-token: write` job (`release.yml:90`), the verbatim event-aware guard on all three existing post-release jobs (`post-release-verify.yml:103,212,248`), the hardcoded `go build` in both test harnesses (`test/integration/main_test.go:39-57`, `test/wireoracle/main_test.go:31`), the awk anchor's exact 6-space indentation match (`Taskfile.yml:547-551` vs `.goreleaser.yaml:228`), and the sidecar download contract (`internal/upgrade/upgrade.go:195`). The wave structure is sound and the RED-before-GREEN discipline is enforced structurally, not rhetorically. Two substantive issues below, both MEDIUM, are about factual accuracy of in-plan claims and an observability gap in D-05's first measurement point — neither invalidates the design.

## Strengths

- **Source-verified foundation.** The central design move (D-18: `binary_signs:` → `signs: {ids: [raw], artifacts: binary}`) is correct against the pinned source. `sign.go:182` binds `env["artifact"] = art.Path` for both pipes, and `archive.go`'s skip branch gives the `raw` entry's `UploadableBinary` the same `Path` quill mutates in place — so the rename is genuinely byte-safe and the `internal/upgrade` contract survives untouched. The plan's insistence on keeping the `signature:` Go-template (because `sign.go:275` rebinds to `art.Name` for the publish pass) is exactly right.
- **The plan corrects Phase 1's false rationale in-place, with a zero-count grep assertion** (`rg -c 'no longer cleanly apply'`). I confirmed the false claim is really there today at `.goreleaser.yaml:183-186`. Treating comment retraction as a tested deliverable is the right response to this repo's documented failure class.
- **Failure-mode-first design throughout.** `verify:gatekeeper` hard-fails on absent xattr read-back and on unclassifiable `spctl` output; the resolver in 02-03 has exactly two outcomes for a non-empty override (use it or abort by name), which structurally forecloses the silent-rebuild false green; 02-06's secrets-scoping test enumerates `.github/workflows/` at runtime and is required to be proven red by deliberate violation.
- **Dependency ordering is honest.** 02-01 (RED baseline on published `v0.5.1`) precedes everything; 02-06 correctly defers all CI wiring until after local rehearsal (02-04); 02-07 is the only plan that can close A3, and it says so. The checkpoint gates (02-01 Task 2, 02-04 Task 2, 02-07 Task 2) sit exactly where irreversible or human-only actions occur.
- **The `enabled:` two-direction hazard and the `[ProjectName]` silent-skip default are both turned into exact-set, resolve-under-both-environments tests** rather than documentation — matching the pinned source behavior I verified.

## Concerns

- **MEDIUM — Plan 02-02's flagged-assumption "correction" about notarize concurrency is itself wrong for the config shape this phase ships.** The plan asserts darwin/amd64 and darwin/arm64 "are signed and submitted to Apple CONCURRENTLY, not sequentially — a detail RESEARCH.md Pattern 1 states the opposite way," and reasons that the configured timeout "bounds the pair rather than accumulating." At source: `MacOS.Run` (`macos.go:43-50`) parallelizes across `notarize.macos` **config entries** via `semerrgroup`, but this phase ships **one entry with two ids**, and inside `signAndNotarize` the per-binary loop (`macos.go:90` onward) is a plain sequential `for _, bin := range binaries.List()`, each iteration getting its own `StatusConfig{Timeout: cfg.Notarize.Timeout}`. So with the shipped shape, the two binaries are processed **sequentially**, each with its own full timeout budget — the research was right and the plan's correction is the error. Consequences: (a) the timeout-sizing rationale in 02-02 is wrong (two sequential 20m budgets, not one bounded pair); (b) worse, plan 02-04 Task 3 instructs the executor to "correct" any comment claiming sequential processing "since the pipe runs its entries under a parallel error group" — i.e., the plan orders a true comment to be replaced with a false one during execution. Fix: strike the concurrency correction from both plans; keep RESEARCH.md's sequential framing; size the timeout as per-binary, sequential.
- **MEDIUM — D-05's first comparison point ("sha256 immediately after the notarize pipe") is not observable as literally specified, and 02-07's label overstates what was measured.** GoReleaser exposes no mid-pipe hook; `release:record-notarize-hashes` (02-07 Task 1) runs after **all** pipes complete, so its hash is "post-everything," and calling it "immediately after notarize" silently assumes A3 — the very assumption the measurement exists to settle. Similarly 02-04 Task 1 step 4 says "capture each darwin binary's sha256 immediately after the build pipe and again after the notarize pipe," which cannot be done within one invocation. The four-point comparison still has teeth (any inter-pipe mutation makes points diverge), but point one's provenance label should be honest: either approximate pre-notarize bytes via a separate `goreleaser build` run (accepting a build-reproducibility assumption and saying so), or relabel point one as "sha256 recorded in the release job from `dist/` post-run." As written, executors will either improvise mid-run or record a mislabeled value — and this repo's own rules treat a mislabeled provenance as a failed gate.
- **LOW — Snapshot-mode `.Tag` emptiness vs. sidecar-name assertions in 02-04.** The rehearsal runs `release --snapshot`; under snapshot, `.Tag` resolves empty (as `release:dry-run`'s own comment notes, "the resolved version string is the only thing --snapshot changes"). Sidecar names will be `codegraph__darwin_arm64.sigstore.json` — still four distinct names (Os/Arch differ), so the distinctness assertion holds, but "matching the per-platform sidecar shape the existing target already asserts" won't literally match the `releaseAssetName` contract with a real tag. The plan should state the expected empty-tag name shape so the recorded evidence isn't misread as a divergence.
- **LOW — 02-01 precondition list inconsistency.** `must_haves.truths` names tools `gh, xattr, spctl`; Task 1's action adds `jq` (used for what, exactly, is never stated — `gh release download` needs no JSON parsing; presumably asset resolution). Also `GATEKEEPER_EXPECT` has a named precondition for presence but no stated validation that its value is in `{accepted, rejected}` — an invalid value would make every observation a mismatch, which fails safe but confusingly.
- **LOW — 02-06 Task 3's amd64 leg on the macOS runner.** The plan handles this ("scope to the native arch if the runner can't execute both"), but it's buried in the task body; on Apple Silicon hardware the amd64 notarized binary runs only under Rosetta, and a hardened-runtime library-validation failure could present differently under translation. Worth promoting to a `must_haves` truth so the scoping decision is recorded, not improvised.
- **LOW — 02-07 Task 1's not-found path exits zero by design.** Defensible (a recording step must not break a publish), but combined with `--snapshot`-vs-real differences it means a silently absent `dist/artifacts.json` yields no first-point evidence and a green release. The checkpoint (Task 2 step 3) requires pasting the `SIGN04-PUBLISH-EVIDENCE` lines, which mitigates this — but the mitigation is human attention, and the plan should say that explicitly.

## Suggestions

1. Fix the concurrency error in 02-02's flagged assumptions and delete the corresponding "correct the comment" instruction in 02-04 Task 3; size `notarize.timeout` against two sequential per-binary submissions (the plan's 20m-over-10m-default instinct survives, but for the right reason).
2. In 02-04 and 02-07, restate D-05 point one as what is actually captured (post-run `dist/` hash, or a separate `goreleaser build` baseline) and add one sentence acknowledging that A3 is settled by *divergence detection across points*, not by a literal mid-pipe snapshot.
3. In 02-04, record the expected snapshot-mode asset names (empty tag segment) as the rehearsal's expected shape, so the evidence file doesn't invite a false "names don't match the contract" reading later.
4. In 02-01, either use `jq` for something specific (e.g., resolving the asset digest from `gh release view --json assets` to cross-check the downloaded sha256 against GitHub's recorded digest — a genuinely useful addition) or drop it from the precondition set; and add value validation for `GATEKEEPER_EXPECT`.
5. In 02-06 Task 3, lift the Rosetta/arch-scoping decision into `must_haves.truths` with a required recorded verdict.
6. Consider asserting in 02-07's checkpoint that `gh api repos/.../releases` asset digests (GitHub now records per-asset SHA256 digests) match the re-downloaded hash — a fifth, independent provenance point that costs one command.

## Risk Assessment

**MEDIUM-LOW.** The architecture is correct and verified at source level; the two MEDIUM findings are accuracy/observability defects in the plan text, not design flaws, and both are cheaply repairable before execution. The irreducible risks are the ones the plans already name honestly: Apple credential provisioning is out-of-repo and blocking; `syspolicy_check`/quarantine-xattr semantics are MEDIUM-confidence pending 02-01's tracer (correctly sequenced first); and the single unrehearsable CI path remains genuinely unrehearsed (D-08's accepted consequence). Nothing in the wave ordering, security posture (D-14/D-17 verified intact against the current files), or upgrade-path contract (`upgrade.go:195,209`, `verify.go:44`) gives me concern beyond what the plans themselves surface.

---

## Consensus Summary

Both reviewers independently verified the phase's central mechanism claims against
the pinned `goreleaser/v2@v2.17.1` module source and agreed the D-18 design is
correct. Both also independently landed on the same structural defect: **D-05's
first comparison point — "the sha256 immediately after the notarize pipe" — is not
observable by any mechanism either plan defines.** Codex rates it HIGH, pi rates it
MEDIUM, and they converge on the same remedy (relabel the captured value honestly,
and obtain a genuine pre-notarize baseline from a separate run).

The two lanes diverge sharply on overall risk (Codex HIGH, pi MEDIUM-LOW). The
divergence is explained by scope: Codex reviewed the CI trust-ordering and
documentation-timing dimensions that pi did not examine, while pi went deeper on
the pinned GoReleaser source and caught a factual error in the plans' own
flagged-assumption block that Codex missed.

### Agreed Strengths

- **The D-18 config move is correct at source level.** Both lanes independently
  confirmed `sign.BinaryPipe{}` precedes `notary.MacOS{}` in `BuildPipeline`, and
  that the release-scoped `signs:` pipe runs after `notary.MacOS` — so cosign
  genuinely signs post-notarize bytes by construction.
- **The `internal/upgrade` sidecar download contract survives the rename**
  untouched, and keeping the explicit Go-template (rather than `${artifact}`) is
  the right call given `signone()`'s Path/Name rebinding.
- **Exact-set assertions on `notarize.macos[].ids` and `signs[].ids`** correctly
  defend both directions — silent underscope via the `[ProjectName]` default, and
  accidental zip inclusion.
- **RED-before-GREEN discipline is structural, not rhetorical**, and the
  checkpoints sit exactly where irreversible or human-only actions occur.
- **02-03's two-outcome resolver** structurally forecloses the silent-rebuild
  false green; both harnesses do currently `go build` unconditionally.
- **Secret scoping in 02-06** is well designed: one job holds `id-token: write`,
  and the runtime enumeration of every workflow file beats a fixed allowlist.

### Agreed Concerns

- **HIGH/MEDIUM — D-05 point one is unobservable as specified** (Codex HIGH, pi
  MEDIUM). `02-04` Task 1 step 4 asks for a hash "immediately after the build pipe
  and again after the notarize pipe" inside one opaque `goreleaser release`
  invocation; `02-07` Task 1 labels a post-everything `dist/` hash as
  "immediately after notarize". Both reviewers note the label silently assumes A3,
  the very assumption the measurement exists to settle.
- **02-07's not-found path exiting zero** (Codex HIGH, pi LOW) leaves SIGN-04
  potentially unprovable for a release that can never be re-cut (D-12).
- **02-01's `jq` precondition is unjustified and `GATEKEEPER_EXPECT` is
  unvalidated** (both LOW).
- **02-06's amd64-on-Apple-Silicon scoping decision is buried in the task body**
  rather than recorded as a must-have (Codex MEDIUM, pi LOW).

### Divergent Views

- **Overall risk: Codex HIGH vs pi MEDIUM-LOW.** Not a contradiction — different
  surfaces reviewed. Codex's HIGHs on CI trust-ordering (02-06) and documentation
  timing (02-05) are areas pi did not assess; pi's source-level MEDIUM on notarize
  concurrency is an area Codex did not assess. Treat the union, not the average.
- **Notarize concurrency (pi only, CONFIRMED).** pi found the plans' own
  flagged-assumption "correction" is factually wrong. Codex did not examine this.
- **Executing an unverified downloaded binary (Codex only).** pi did not raise it;
  Codex's reading of the plan text is accurate.

---

## Verification coverage — orchestrator source-grounding

The orchestrator independently re-derived every load-bearing claim below from the
pinned module cache, the working tree, and live macOS 27.0 (Tahoe) tooling. This
section records what was **confirmed**, what was **refuted**, and two **new HIGH
findings neither lane produced**.

### Reviewer claims CONFIRMED at source

| Claim | Verdict | Evidence |
|---|---|---|
| `sign.BinaryPipe{}` runs before `notary.MacOS{}`; release-scoped `sign.Pipe{}` runs after both | CONFIRMED | `internal/pipeline/pipeline.go:103,105,143` (v2@v2.17.1) |
| `signs[].ids` filters **archive** ids; `notarize.macos[].ids` filters **build** ids | CONFIRMED | `archive.go:308-322` sets `ExtraID: archive.ID` on `UploadableBinary`; `notary/macos.go:80-83` filters `artifact.Binary` by `ByIDs` |
| `signone()` rebinds `env["artifact"]` from `art.Path` to `art.Name` for the publish-naming pass | CONFIRMED | `internal/pipe/sign/sign.go:182` then `:275` |
| `notarize.macos[].ids` defaults to `[ProjectName]` and an empty match is a **non-fatal skip** | CONFIRMED | `notary/macos.go:35-37`; `pipe.Skipf` at `:86-88` |
| quill mutates the Mach-O **in place** at `bin.Path` (so sha256 necessarily changes) | CONFIRMED | `quill/sign.go:130-160` — `AddEmptyCodeSigningCmd` on `cfg.Path` |
| Assumption **A3** (no later release-scoped pipe re-mutates the binary after `sign.Pipe`) | CONFIRMED by inspection | Every pipe after `sign.Pipe{}` (`aur`, `nix`, `winget`, `brew`, `cask`, `krew`, `scoop`, publish) emits manifests only |
| The retracted-rationale text really is present today | CONFIRMED | `.goreleaser.yaml:183-186` |
| Both harnesses `go build` unconditionally | CONFIRMED | `test/integration/main_test.go:47`, `test/wireoracle/main_test.go:29` |
| `self-upgrade` deliberately omits `needs: verify-supply-chain`, with rationale | CONFIRMED | `.github/workflows/post-release-verify.yml:240-247` |
| `record-notarize-hashes` would sit after publish, before attestation | CONFIRMED | `release.yml:169-173` (Release step) then `:188` (attest step) |
| pi's notarize-concurrency refutation | **CONFIRMED — the plans are wrong** | `notary/macos.go:43-50` parallelizes across *config entries*; `:90` loops binaries **sequentially**. This phase ships one entry with two ids ⇒ sequential. Plan text at `02-02-PLAN.md:377-378` and `02-04-PLAN.md:297-298` |

### Maintainer-flagged items independently re-verified

- **Threat-ID duplication is benign.** `T-02-11`, `T-02-16`, `T-02-17` each appear
  twice **within their own file only**; a cross-file collision scan returns empty.
  Not the Phase-1 collision class. Confirmed.
- **D-18 involves no upstream GoReleaser defect.** The pipe order is deliberate;
  neither reviewer proposed reporting an upstream bug.

### NEW HIGH findings — measured, not inferred

Both were produced by running the actual tools on macOS 27.0 (build 26A5388g),
the same OS class as the `namespace-profile-macos-6x14-tahoe` runner. Neither
reviewer had a macOS host and neither raised these.

**NEW-HIGH-1 — `spctl -a -vv -t exec` categorically rejects *any* bare Mach-O,
so SIGN-02's stated proof can never pass and 02-01's RED baseline is a false RED.**

A Developer ID-signed, hardened-runtime, notarized bare executable (Docker's
`docker` binary, `Authority=Developer ID Application: Docker Inc (9BNSXJN65R)`,
`flags=0x10000(runtime)`), copied and given a `com.apple.quarantine` xattr:

```
/tmp/sp-devid: rejected (the code is valid but does not seem to be an app)
origin=Developer ID Application: Docker Inc (9BNSXJN65R)
EXIT=3
```

An Apple-signed bare binary behaves identically (`origin=macOS Software Signing`,
same `rejected … does not seem to be an app`, exit 3).

Consequences:
- `REQUIREMENTS.md:25` requires `spctl -a -vv -t exec` to report
  `source=Notarized Developer ID`. For a bare Mach-O it **never will**, however
  perfectly notarized. **Criterion 2 / SIGN-02 is unachievable as written.**
- `02-01`'s RED baseline will observe `rejected` and be recorded as a working
  gate — but the rejection reason is *"does not seem to be an app"*, not *"not
  notarized"*. The gate would report RED for a correctly notarized binary too.
  This is precisely the false-confidence failure mode SIGN-03 exists to prevent,
  reproduced inside the mechanism meant to prevent it.
- `02-07` would run the full phase to an **irreversible release** and only then
  discover the GREEN criterion is unreachable — with D-12/D-13 forbidding a re-cut.

This does not mean the phase's *user-facing* goal is wrong. Gatekeeper's runtime
policy for a bare executable launched from a shell differs from `spctl`'s
bundle-oriented assessment. What is refuted is the **chosen instrument**, not the
guarantee. The phase needs a different oracle (e.g. LaunchServices-mediated open,
or assessing a `.app`/zip shape), decided **before** 02-01 records a baseline.

**NEW-HIGH-2 — `syspolicy_check distribution` fatally requires a *stapled* ticket,
which this phase's guarantee explicitly excludes.**

```
$ syspolicy_check distribution <bare-macho>
App has failed one or more pre-distribution checks.
    Notary Ticket Missing
    Severity: Fatal
    Full Error: A Notarization ticket is not stapled to this application.
$ echo $?   # 70
```

`REQUIREMENTS.md:25` requires `syspolicy_check distribution` to **pass**. The
phase's own three-part guarantee is *"notarized, online-verified, **not
stapled**"*, and the plans state a bare Mach-O is categorically unstaplable.
These two requirements are in **direct contradiction**: the command's failure is
fatal-by-design for exactly the artifact shape this project ships.

Two secondary measurements that settle the plans' open questions on this tool:
- **Exit status is authoritative** — `70` on fatal failure, so an exit-code gate
  is sound (no silent green).
- **`--json` stdout is clean and `jq`-parseable**; the `"Only one signature found
  … skipping dual signature check"` preamble goes to **stderr**. This *justifies*
  02-01's `jq` precondition, contradicting both reviewers' LOW suggesting it be
  dropped — provided the target actually consumes `--json`.
- Argument type is `<bundle-path>` — `syspolicy_check distribution --help` names
  it "The application bundle to check for distribution."

### Still genuinely open

- Whether a synthetic `com.apple.quarantine` xattr behaves identically to a
  browser-applied one (assumption A2) — untested here; 02-01's comparison stands.
- What the correct Gatekeeper oracle *is* for an unbundled CLI binary, given
  NEW-HIGH-1. This is now the phase's blocking design question.
