---
phase: 2
cycle: 2
reviewers: [codex, pi]
reviewed_at: 2026-08-09T15:07:35Z
plans_reviewed: [02-01-PLAN.md, 02-02-PLAN.md, 02-03-PLAN.md, 02-04-PLAN.md, 02-05-PLAN.md, 02-06-PLAN.md, 02-07-PLAN.md]
---

# Cross-AI Plan Review — Phase 2 (Cycle 2)

Both lanes were briefed that CodeGraph is indexed here and that D-19, D-04, D-18 and
the 02-02 Task 2 deviation are settled and must not be re-litigated. Neither lane
re-opened them. The index they were pointed at (430 files / 4,333 nodes / 10,453 edges)
was confirmed current against `codegraph status` before invocation.

**Codex lane caveat:** Codex reported it could not open `.codegraph/store/LOCK` under
its sandbox and fell back to direct source reads. Its `path:line` citations were
therefore produced by file reads, not graph queries — which is the stronger evidence
form anyway, and the orchestrator re-verified every load-bearing citation below.

## Codex Review

# Cross-AI Plan Review — Cycle 2

## Overall assessment

The revised plan set is substantially stronger. D-19 fixes the previously unreachable Gatekeeper gate; post-release execution now verifies integrity before running downloaded code; evidence distinguishes hashes from cryptographic bindings; and the test-binary seam correctly refuses silent fallback.

Two material issues remain:

1. Plan 02-04’s separate-build “pre-sign baseline” is not yet a sound substitute for the bytes entering quill.
2. Plan 02-06 does not require the post-release test jobs to check out the resolved release tag, even though the workflow itself documents that its default ref is the default branch.

Overall phase risk remains **HIGH** until those are corrected because both affect the validity of evidence used to authorize an irreversible release.

CodeGraph could not open `.codegraph/store/LOCK` under the current sandbox, so Go call paths were verified through direct source reads. Context7 also failed to return documentation before timeout; findings below therefore rely on the pinned repository configuration and source-level contracts.

---

## 02-01 — Gatekeeper RED tracer

### Summary

This is a well-designed tracer with a meaningful negative control, exit-status classification, quarantine read-back, and reusable evidence format. One fail-open digest path should be tightened.

### Strengths

- The oracle now matches D-19 exactly: `spctl -a -vv -t install`, classified by exit status rather than output text ([02-01-PLAN.md:185](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-apple-signing-notarization/02-01-PLAN.md:185)).
- The plan correctly treats source text as corroboration rather than the verdict and detects contradictory status/output combinations ([02-01-PLAN.md:197](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-apple-signing-notarization/02-01-PLAN.md:197)).
- The schema is practical: versioned, fixed-order, and explicit about unavailable values. That should make later evidence joins less fragile.
- Synthetic-versus-browser quarantine comparison directly addresses the test-rig trust boundary instead of assuming it away.

### Concerns

- **MEDIUM — Missing GitHub digest does not fail the provenance check.** The plan calls digest cross-checking a must-have, but instructs the target to emit a sentinel and continue if GitHub supplies no digest ([02-01-PLAN.md:167](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-apple-signing-notarization/02-01-PLAN.md:167)). That allows the target to pass while the claimed identification of served bytes was never established.
- **LOW — Source contradiction is stricter than the requirement.** Rejecting exit 0 when the expected source line is absent is defensible, but it makes output formatting an additional gate despite the plan saying exit status is the oracle. This should be described as an intentional second assertion, not a contradiction in the oracle.

### Suggestions

- Make a missing or malformed GitHub digest fatal when the API version used is expected to provide it. If older releases genuinely lack digests, explicitly downgrade `digest_match` for the RED baseline only and require the field for GREEN.
- Represent source-line verification as a separate property such as `source_assertion=pass|fail`; keep `observed` derived solely from status.

### Risk assessment

**MEDIUM.** The Gatekeeper mechanism is sound; the remaining issue affects provenance strength, not the RED verdict itself.

---

## 02-02 — GoReleaser signing and notarization configuration

### Summary

The plan accurately targets the current repository shape and replaces the stale `binary_signs:` mechanism intentionally. Exact-set tests for both build IDs and archive IDs are particularly valuable.

### Strengths

- The existing configuration confirms why the change is necessary: `binary_signs:` is currently active at [.goreleaser.yaml:223](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:223), with a now-invalid rationale at [.goreleaser.yaml:183](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:183).
- Retargeting rather than deleting `TestBinarySignsSidecarMatchesUpgradeContract` preserves the real upgrade-sidecar contract. The current test resolves all four names rather than matching a literal ([goreleaser_shape_test.go:507](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/goreleaser_shape_test.go:507)).
- Exact `[raw]` and exact darwin-build-ID assertions defend both under-selection and over-selection.
- The plan correctly labels the injected `text/template` test as a static backstop, not proof of GoReleaser runtime behavior ([02-02-PLAN.md:29](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-apple-signing-notarization/02-02-PLAN.md:29)).
- Caller enumeration is justified. The current `check:darwin-release-build` comment and cosign precondition are indeed stale after D-18 ([Taskfile.yml:287](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:287), [Taskfile.yml:307](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:307)).

### Concerns

- **LOW — The chosen enablement mechanism is left to execution.** The plan permits either a five-way conjunction or a separate enable flag. Both can work, but they create different operational contracts and different secret-presence failure modes.
- **LOW — Mutation-based acceptance checks are numerous and manual.** They are useful, but the executor must be disciplined about restoring the config after every mutation.

### Suggestions

- Prefer the explicit five-variable conjunction. A separate enable flag introduces a sixth input whose accidental presence could enable an incomplete credential set unless the template still validates all five.
- Have mutation checks operate on temporary config fixtures wherever possible, rather than editing the working file.

### Risk assessment

**LOW–MEDIUM.** The configuration transition is detailed, source-aligned, and well defended by non-vacuous tests.

---

## 02-03 — External test-binary seam

### Summary

The resolver design is appropriately fail-closed. Rejecting both a shared production package and probe-execution is reasonable for two test-only consumers.

### Strengths

- Current harnesses unconditionally build source binaries ([integration/main_test.go:39](/Volumes/Code/github.com/seanb4t/codegraph-go/test/integration/main_test.go:39), [wireoracle/main_test.go:21](/Volumes/Code/github.com/seanb4t/codegraph-go/test/wireoracle/main_test.go:21)), so the seam closes a real criterion-4 gap.
- The proposed resolver’s two-outcome rule prevents the dangerous “invalid override → local rebuild” path.
- Converting relative paths to absolute paths is necessary because integration tests execute from varying working directories.
- Rejecting probe-execution is sound. Stat-level validation handles configuration mistakes; actual architecture, Mach-O, and hardened-runtime compatibility should be exposed by the suite itself ([02-03-PLAN.md:93](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-apple-signing-notarization/02-03-PLAN.md:93)).
- The shim-based positive-identification proof is substantially stronger than elapsed-time inference.
- Two small test-only copies are acceptable. A production `internal/testbin` package would give runtime code a test-harness concern and is premature with only two consumers.

### Concerns

- **LOW — “Byte-identical unset behavior” is overstated.** `TestMain` necessarily changes structurally. What can be preserved is behavioral equivalence of the build path, not literal byte identity of the function.
- **LOW — Executable-mode validation is platform-specific.** This is fine for the current macOS/Linux harness, but the resolver comment should say it intentionally follows Unix mode semantics.

### Suggestions

- Change “byte-identically to today” to “behaviorally equivalent, using the unchanged build command and cleanup path.”
- Consider one shared table of behavior descriptions copied into both packages’ tests, but do not introduce a production helper package yet.

### Risk assessment

**LOW.** The plan closes the silent-fallback risk cleanly without unnecessary abstraction.

---

## 02-04 — Local notarization and ordering rehearsal

### Summary

The cryptographic two-candidate verification is conceptually correct, but the proposed separate-build baseline is not yet proven commensurable with the release run. This is the main unresolved defect in the plan set.

### Strengths

- `cosign verify-blob` against both candidates is the right way to identify the bundle’s subject. Exactly one successful verification distinguishes the subject cryptographically rather than inferring it from metadata ([02-04-PLAN.md:222](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-apple-signing-notarization/02-04-PLAN.md:222)).
- The mutation should invert the verification relationship if the candidates are genuine pre- and post-sign versions of the same build.
- A single-architecture mutation is reasonable. The tested property is pipe ordering and subject selection; both are configured once, outside architecture-specific build declarations. The correct configuration still rehearses both architectures.
- Credential preconditions, config-copy mutation, clean-worktree checks, and secret-leak observation are strong controls.
- The plan correctly stopped pretending a shell wrapper can capture a mid-pipeline hash.

### Concerns

- **HIGH — The separate `goreleaser build` baseline is not established as the release run’s pre-sign bytes.** The plan assumes reproducibility ([02-04-PLAN.md:173](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-apple-signing-notarization/02-04-PLAN.md:173)), but the binaries embed GoReleaser template values for tag, commit, and date ([.goreleaser.yaml:62](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:62)). The existing Taskfile explicitly says snapshot mode changes the resolved version string ([Taskfile.yml:282](/Volumes/Code/github.com/seanb4t/codegraph-go/Taskfile.yml:282)). A separate `build` and `release --snapshot` can therefore differ before quill for metadata or command-mode reasons. If so:

  - `final_sha256 != baseline_sha256` does not prove quill rewrote the file.
  - Under the mutation, cosign may verify neither candidate, because it signed a third pre-quill byte stream from the release invocation.
  - The “exactly one” gate fails without distinguishing nondeterminism from ordering.

- **MEDIUM — The repository’s reproducibility evidence is not sufficiently tied to this exact invocation pair.** Even if a prior CI gate double-builds successfully, it must use the same command, snapshot/version inputs, config, toolchain, output identity, and environment to license this substitution. The plan currently calls that a backstop rather than proving equivalence.
- **LOW — The rehearsal does multiple Apple submissions for an ordering experiment whose crucial cosign mutation could potentially be isolated without resubmitting both architectures.** The one-architecture reduction helps, but the cost remains notable.

### Suggestions

Before relying on the baseline, add a credential-free control:

1. Run the exact same generated config and exact same command twice with notarization disabled.
2. Confirm each darwin output is byte-identical.
3. Confirm the version/commit/date values embedded in both runs are identical.
4. Only then compare one of those outputs with the notarized run.

Preferably, make the baseline and notarized runs both use `goreleaser release --snapshot` with identical flags and generated config, differing only in the notarize enable predicate. Do not compare `goreleaser build` with `goreleaser release`.

If identical-command unsigned runs do not reproduce, treat the two-candidate inversion experiment as blocked rather than weakening “exactly one verifies.”

### Risk assessment

**HIGH.** The ordering experiment’s oracle depends on a baseline equivalence the plan does not presently establish.

---

## 02-05 — Release documentation

### Summary

The documentation plan is careful about temporal truth and accurately reflects D-19. The per-tag table is useful initially but risks becoming a maintenance burden.

### Strengths

- The plan avoids claiming notarization before a notarized release exists.
- It explicitly separates Apple’s online notarization record from stapling, preventing the misleading “carries a ticket” phrasing.
- It requires the documented xattr read-back before assessment and aligns commands with the measured gate.
- It correctly relegates `syspolicy_check distribution` and `-t exec` to the “does not verify this claim” explanation.
- It scopes reproducibility correctly: unsigned build reproducibility and signed Darwin artifact reproducibility are different claims.

### Concerns

- **LOW — Listing every released tag is unnecessary and will become stale.** The acceptance criterion requires the applicability table to list every tag known at execution time ([02-05-PLAN.md:175](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-apple-signing-notarization/02-05-PLAN.md:175)). This turns a focused release-verification document into a manually maintained release ledger.
- **LOW — The section may be overlong for end users.** The six-item insufficiency list is valuable, but could bury the actionable three-command path.

### Suggestions

- Use boundary rows rather than every tag: “through v0.5.1,” “first notarized release,” and “later releases.”
- Put the concise user procedure first, then a collapsible or clearly secondary “Why these other checks do not count” subsection.

### Risk assessment

**LOW.** Mostly documentation maintainability and presentation risk.

---

## 02-06 — CI secrets and post-release verification

### Summary

The integrity-before-execution correction is strong, but the notarized suite may test a release binary against source and tests from the default branch rather than the resolved release tag.

### Strengths

- Step-level secret scoping is better than job-level scoping and matches the minimum exposure possible inside the existing release job.
- Runtime enumeration of all workflow files makes the secret-scope test forward-looking.
- The test honestly states that it cannot verify GitHub dashboard-level secret scope.
- The suite now verifies the downloaded binary with cosign before `chmod` or execution and also declares `needs: verify-supply-chain` ([02-06-PLAN.md:289](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-apple-signing-notarization/02-06-PLAN.md:289)). This correctly fixes the former parallel-execution security hole.
- Counting `go test -json` `run` events is an appropriate non-vacuity check.
- Keeping Gatekeeper independent is defensible because it assesses but does not execute the downloaded file.

### Concerns

- **HIGH — The suite job is not instructed to check out the resolved tag.** The workflow’s own header states that a `workflow_run` job’s `GITHUB_REF/GITHUB_SHA` refer to the default branch and its tip ([post-release-verify.yml:55](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/post-release-verify.yml:55)). Existing checkout steps provide no `ref:` ([post-release-verify.yml:215](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/post-release-verify.yml:215), [post-release-verify.yml:261](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/post-release-verify.yml:261)), and plan 02-06 says to mirror that skeleton ([02-06-PLAN.md:210](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-apple-signing-notarization/02-06-PLAN.md:210)). Consequently, the job can execute the released binary against newer or otherwise different integration tests and fixtures from the default branch. This weakens both reproducibility and diagnosis.
- **MEDIUM — `needs: verify-supply-chain` can make the suite skip after a supply-chain failure.** The in-job cosign verification already protects execution. If the phase also wants independent diagnostic evidence from the suite, the job dependency sacrifices it. This is not unsafe, but the plan should acknowledge the reporting tradeoff.
- **LOW — Requiring both in-job cosign and prior supply-chain success duplicates network verification.** The belt-and-braces choice is reasonable for safety; it carries modest latency.

### Suggestions

- Add `with: ref: ${{ needs.resolve-tag.outputs.tag }}` to checkout in both new post-release jobs that consume repository scripts or tests. For the suite job this is essential.
- Add a workflow-shape test that asserts every job executing release-tag artifacts checks out the same resolved tag.
- Decide explicitly whether diagnostic independence or graph ordering is preferred. If independent diagnostics matter, keep in-job cosign and remove the `needs: verify-supply-chain` dependency; if graph clarity wins, retain both and state that a supply-chain failure intentionally suppresses the suite.

### Risk assessment

**HIGH.** Security ordering is fixed, but source/tag mismatch can invalidate the claim that the released artifact passed its own release’s suite.

---

## 02-07 — Real release and final evidence

### Summary

The final plan now labels observed values accurately and distinguishes hashes from bindings. It is structurally sound once the 02-04 baseline and 02-06 checkout issues are fixed.

### Strengths

- `final_local_sha256` accurately describes post-pipeline local bytes rather than claiming an unavailable mid-pipe observation.
- The five-point chain correctly distinguishes three hashes from two cryptographic bindings ([02-07-PLAN.md:24](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/02-apple-signing-notarization/02-07-PLAN.md:24)).
- Making the final-hash recorder fail loudly is correct because GoReleaser has already published by then; silent continuation would preserve the release but destroy the evidence.
- Placement between release and attestation matches the current workflow ordering: GoReleaser publishes at [release.yml:170](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:170), and attestation follows at [release.yml:188](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:188).
- The checkpoint preserves logs before analysis and retains patch-forward recovery.
- The RED/GREEN comparison uses the same gate and differs only by tag, which is exactly the right evidence shape.

### Concerns

- **MEDIUM — A failing final-hash step makes the release workflow conclude failure after assets were published.** Because post-release verification is guarded on upstream success, automatic Gatekeeper and suite jobs will then skip. The manual-dispatch recovery exists, but the plan does not explicitly instruct it for this failure branch.
- **MEDIUM — A3 is only partially closed by local-versus-published equality.** Equality proves nothing after `final_local_sha256` changed the local file and before download, but it cannot isolate which internal pipe would have mutated the file. That is acceptable for the user-facing identity property, but the evidence should say “no post-record/publish-path mutation observed,” not over-attribute the result to a particular GoReleaser interval.
- **LOW — The `<done>` wording in Task 1 says the recorder “cannot break a publish.”** It cannot prevent already-completed publishing, but it deliberately can fail the release workflow. That distinction should be explicit.

### Suggestions

- Add a failure branch: if `release:record-final-hashes` fails after publish, manually dispatch post-release verification against the resolved tag after preserving logs; do not wait for the workflow-run trigger.
- Describe A3’s conclusion narrowly: final local bytes matched served bytes; therefore no mutation occurred after the recorded point. Do not claim this alone traces all internal pipe boundaries.
- Change “cannot break a publish” to “cannot prevent already-completed publication, but intentionally fails the workflow when evidence cannot be recorded.”

### Risk assessment

**MEDIUM**, conditional on correcting plans 02-04 and 02-06. The final publication/evidence workflow is otherwise coherent.

---

## Priority changes before execution

1. **Block 02-04 until the pre-sign baseline uses an identical command/config/input path and demonstrates unsigned-run byte identity.**
2. **Make post-release suite checkout the resolved tag, not the workflow’s default branch tip.**
3. **Decide whether missing GitHub asset digests are fatal or explicitly acceptable only for the preserved RED baseline.**
4. **Document the manual-dispatch recovery when a post-publish hash-recording step fails.**

With those changes, the plan set would move from **HIGH** to **MEDIUM** risk, with the remaining uncertainty concentrated appropriately in the real Apple release event rather than in avoidable evidence ambiguity.

---

## pi Review

I have enough verified evidence. Here is my review.

---

# Cross-AI Plan Review — Phase 2: Apple Signing & Notarization (Cycle 2)

## Summary

This is an unusually well-grounded plan set. Every structural claim I checked against the repository holds: the awk injection anchor in `release:dry-run-signed` really does match `/^      - "sign-blob"$/` at 6-space indent (Taskfile.yml:548 vs .goreleaser.yaml:227), the event-aware guard string in 02-06's acceptance criterion is verbatim the one on `post-release-verify.yml:103,212,248`, `id-token: write` exists exactly once repo-wide (release.yml:90), the `binary_signs:` comment block at .goreleaser.yaml:183-186 really does contain the false `signs:` rationale D-18 retracts, and the notary pipe source in the pinned v2.17.1 module confirms both the sequential per-binary loop and the corrected concurrency framing. The D-19 oracle amendment is correctly propagated everywhere (I found zero remaining `-t exec` gating references in the plans). The plans' main residual weakness is that 02-04's pre-sign baseline rests on a *darwin* build-determinism assumption the repo's own gates never prove — and proving it in-situ would be nearly free.

## Strengths

- **The cryptographic subject-identification design (02-04 step 6a) is sound and I verified its premises at source.** The mutation arm inverts because `binary_signs:` signs pre-notarize bytes while `signs:` signs post-notarize bytes; `cosign verify-blob` against two candidate files with an exactly-one-must-verify rule is a genuine discriminator, and the "both or neither verifies = hard failure" clause correctly converts a broken experiment into an inconclusive result rather than false evidence. This is materially better than the cycle-1 re-hash-a-path design, which quill's in-place Mach-O rewrite (`notary/macos.go:101`, `quill.Sign(bin.Path, …)`) would indeed have corrupted.
- **The exit-status-only verdict discipline (02-01) is correct.** spctl's stderr contains `rejected (the code is valid but does not seem to be an app)` — a substring search for "accepted" semantics would misclassify. Verified the plan greps nothing for classification and treats non-{0,3} exits as fatal, which is the safe failure direction.
- **Partial-credential gating is not paranoia — the pipe signs-and-continues without notarizing.** `notary/macos.go:106-111`: when `IssuerID/KeyID/Key` are empty, quill *signs* the binary, logs `will not try to notarize`, and continues — a green log shipping a signed-but-unnotarized binary. 02-02's conjunctive all-five `enabled:` gate is the correct mitigation, and `TestNotarizeMacosEnabledIsEnvGated`'s partial-environment cases pin exactly this.
- **02-06's verify-before-chmod ordering closes a real hole.** The existing `self-upgrade` job is deliberately independent (post-release-verify.yml:241-247, `needs: resolve-tag` only), and the plan correctly refuses to copy that shape for a job that *executes* the download — while preserving the gatekeeper job's independence with a recorded rationale (assessment ≠ execution). The in-target cosign flags matching `verify:release-assets` (Taskfile.yml:1104-1111) keeps the two verifiers from drifting.
- **The `isEnvSet` idiom is valid in the pinned module** — `internal/tmpl/tmpl.go:299` registers it. The backstop-labeled FuncMap emulation honestly scopes what the static test proves.
- **The rehearsal path genuinely reaches the notarize pipe under `--snapshot`.** `MacOS.Skip` (notary/macos.go:27-29) checks only `skips.Notarize` and empty config — no snapshot gate — so `release --snapshot --skip=publish` exercises sign+notarize for real. The plan's central rehearsal mechanism is viable.
- **Evidence schema discipline** (schema=1, fixed key order, `not-found`/`unknown` sentinels) matches the existing `TAG-EVIDENCE`/`SIGN-EVIDENCE` conventions and fixes the real cross-plan fragility cycle 1 identified.
- 02-03's two rejections are well-reasoned: a probe-execute in the resolver would both duplicate the suite's own purpose and run untrusted downloaded bytes earlier than necessary; the two-copy resolver mirrors the repo's existing four-copy `runGit*` precedent (visible at test/integration/main_test.go:57-60).

## Concerns

**HIGH**

1. **02-04's pre-sign baseline assumes *darwin goreleaser* determinism that no existing gate proves — and the assumption is load-bearing twice.** The repo's determinism evidence is `check:reproducibility` (ci.yml:216-273, Taskfile.yml:1559-1635): a blocking *linux/amd64* same-host double-build and a report-only linux/arm64 leg, both via raw `go build`. Nothing covers darwin, and nothing covers determinism *between two separate goreleaser invocations* (the baseline build vs. the release run's build pipe). The plan's `final != baseline` assertion is safe (quill provably rewrites the file — notary/macos.go:101), but a false difference doesn't prove quill did it, and the mutation arm's `verify-blob` against the baseline requires byte-equality to invert at all. The plan records this as a `backstop` truth — honest, but measuring it directly costs one extra notarize-disabled build in the same rehearsal: build the baseline **twice** and assert `baseline1 == baseline2` before trusting any comparison against it. As written, a nondeterministic darwin build surfaces as a confusing "neither verifies" hard failure with no label telling the maintainer *why*.

**MEDIUM**

2. **The timeout/pending behavior the plans carry as an unresolved assumption is readable at source *now*, and it sharpens the risk.** `notary/macos.go:129-138`: `TimeoutStatus` → log "notarize timeout" and continue; default (pending) → log and continue. Neither fails the pipe. So a slow Apple response on release day ships signed-but-unnotarized binaries behind a green release log — the rehearsal's checkpoint watches for this, but 02-02's flagged-assumption framing ("NOT source-confirmed… believed") understates what the pinned module already shows. More importantly, this makes 02-06's gatekeeper post-release job the *only* hard gate for a timeout-on-release-day — which is fine (it's designed for exactly this) but deserves to be stated as the compensating control rather than discovered as one.

3. **02-07's `release:record-final-hashes` runs after publish, so its hard-fail can't break a publish — but it *can* mask one.** The step correctly fails loudly on missing metadata (a real improvement over cycle 1), but a hard failure *after* a successful publish leaves the release job red while assets are live, and the post-release-verify workflow's `workflow_run` trigger fires on `completed` with `conclusion == 'success'` — so a failed hash-recording step means **post-release-verify never runs for that tag automatically**. The maintainer must then know to use `workflow_dispatch`. The plan never names this interaction; one sentence in 02-07 Task 1 (failure ⇒ manual re-dispatch required) would close it.

4. **02-01's digest cross-check depends on `gh release view --json assets` exposing a `digest` field.** That field is a relatively recent gh/API addition; the plan's sentinel fallback ("record the sentinel rather than skipping") is the right defensive shape, but the acceptance criteria never require the sentinel-vs-real-digest outcome to be recorded. If the runner's gh version lacks the field, every evidence line silently carries `digest_match=unknown` and the provenance strengthening (the stated justification for keeping `jq`) evaporates without anyone noticing. Require the checkpoint to state which case occurred.

**LOW**

5. **02-04: the `signs:`-arm evidence labels rely on snapshot naming** (empty tag segment), correctly flagged as expected — but the *attestation* leg of criterion 3 is entirely absent from rehearsal because `actions/attest-build-provenance` only runs in CI. Assumption A3 is carried to 02-07 appropriately; just note the rehearsal's "converged hash set" can never include the SLSA subject, so criterion 3's first *complete* five-point closure is only ever observed on the irreversible release. The plans say this; the checkpoint should ask the maintainer to acknowledge it explicitly.

6. **02-06: `TestAppleSecretsScopedToSingleReleaseJob` enumerates `.github/workflows/` at runtime** — good — but the repo also has `release-please.yml` as a *tag-adjacent* workflow; a quick manual confirmation that the test's pull_request-trigger detection covers `pull_request_target` (not just `pull_request`) is worth one line, since `pull_request_target` is the more dangerous fork-reachable trigger and the repo has 13 workflow files.

7. **02-01: spctl exit-code drift across macOS versions.** Exit 3 was measured on macOS 27.0; the CI runner is Tahoe-class (26). The non-{0,3}-is-fatal rule makes drift a loud failure rather than a wrong verdict — acceptable, but worth a one-line comment in the target.

## Suggestions

1. **(02-04, addresses Concern 1)** Add a second notarize-disabled baseline build to the rehearsal target and assert `baseline1 == baseline2` with a distinctly-labelled failure (`BASELINE-NONDETERMINISTIC`) so the failure mode names itself. Zero extra Apple submissions; converts the backstop truth into a measurement.
2. **(02-02)** Record the source-confirmed timeout/pending continue behavior (`notary/macos.go:129-138`) in the notarize block's comment now, replacing the "unresolved" framing — and state that the post-release gatekeeper job is the compensating control for a release-day timeout.
3. **(02-07 Task 1)** Add one comment line: a hard failure in `release:record-final-hashes` after publish requires a manual `workflow_dispatch` of post-release-verify for that tag, since `workflow_run` only fires green.
4. **(02-01 Task 2)** Add a checkpoint step: state whether GitHub's recorded digest was present or the sentinel was recorded, per asset.
5. **(02-06 Task 1)** Extend the secrets-scoping test (or its doc comment) to name `pull_request_target` explicitly.
6. **(02-03)** The shim-based positive-identification proof is clever; note in the test plan that the shim must `exec "$@"`-forward *and* be chmod +x, and that a suite failure *through* the shim would be indistinguishable from a real failure — record the log first, suite result second.

## Risk Assessment

**MEDIUM.** Down from HIGH in cycle 1. The two structural risks that made cycle 1 dangerous (an oracle that could never pass; an unsatisfiable mid-pipe hash capture) are both resolved with mechanisms I verified against the pinned GoReleaser source and this repo's actual files. What remains is concentrated where it should be: one unproven determinism assumption in the rehearsal baseline (Concern 1, cheaply fixable), and the irreducible fact — honestly carried by the plans — that the full evidence chain can only close on an irreversible publish. The wave structure, RED-first gate, cryptographic subject identification, and verify-before-execute ordering are all correct. With Concern 1 addressed, I'd rate this LOW-MEDIUM.

---

## Consensus Summary

Cycle 1's seven HIGH findings are resolved. Both lanes independently confirmed the
substance of each fix at source: the D-19 oracle amendment is propagated with zero
remaining `-t exec` gating references; `02-04`'s mid-pipe hash capture is gone and
replaced with cryptographic subject identification; `02-06` verifies with cosign
before `chmod`; `02-07`'s recorder is renamed `release:record-final-hashes` and now
hard-fails on absent metadata; `02-05` carries a per-tag applicability table with a
pending marker instead of an unqualified present-tense guarantee.

**Two HIGH concerns remain, both newly raised this cycle, both confirmed by the
orchestrator against the working tree.**

Overall risk: Codex rates the set **HIGH** until both are corrected, then MEDIUM.
pi rates it **MEDIUM**, down from cycle 1, and LOW-MEDIUM once its Concern 1 is
addressed. The gap is scope, not disagreement: pi did not assess the CI checkout-ref
surface that produced Codex's second HIGH.

### Agreed Strengths

- **The two-candidate `cosign verify-blob` design (02-04 step 6a) is sound.** Both
  lanes verified its premise: `binary_signs:` signs pre-notarize bytes while `signs:`
  signs post-notarize bytes, so an exactly-one-must-verify rule is a genuine
  cryptographic discriminator, and the "both or neither verifies is a hard failure"
  clause converts a broken experiment into an inconclusive result rather than false
  evidence. Both agree this is materially better than cycle 1's re-hash-a-path design,
  which quill's in-place Mach-O rewrite would have corrupted.
- **Exit-status-only verdict classification (02-01) is correct.** pi confirmed spctl's
  own output contains the string `rejected` in diagnostic text, so a substring search
  really would misclassify; both lanes endorse treating non-{0,3} as fatal as the safe
  failure direction.
- **The evidence-line schema** (`schema=1`, fixed key order, `not-found`/`unknown`
  sentinels) fixes the real cross-plan fragility cycle 1 identified, and matches the
  repo's existing `TAG-EVIDENCE`/`SIGN-EVIDENCE` conventions.
- **02-06's verify-before-chmod ordering closes a real hole**, and both lanes endorse
  keeping the Gatekeeper job independent because it *assesses* without *executing*.
- **02-03's two rejections are adequately reasoned.** Both lanes accepted them
  independently. The orchestrator confirmed the load-bearing precedent: the repo does
  carry exactly four deliberate copies of its git helper
  (`internal/indexer/capability/matrix_test.go`, `internal/migrate/migratetest/fixture.go`,
  `tools/bench/runner/main.go`, `tools/bench/realcorpus/manifest.go`), and the plan
  writes down the extraction trigger rather than leaving it to taste.
- **Single-arch mis-order mutation is a sound reduction.** Both lanes agree the tested
  property is pipe ordering and subject selection, which are configured once outside
  the architecture-specific build declarations.
- **02-02's conjunctive all-five credential gate is the right mitigation**, and pi
  supplied the reason it matters more than cycle 1 knew: at `notary/macos.go:106-111`
  the pipe *signs* and logs "will not try to notarize" and continues when the notary
  credentials are empty — a green log shipping signed-but-unnotarized bytes.

### Agreed Concerns

- **HIGH — `02-04`'s separate-build pre-sign baseline is not established as
  commensurable with the release run's pre-quill bytes.** Raised independently by both
  lanes and confirmed by the orchestrator. `02-04-PLAN.md:173-185` substitutes a
  separate notarize-disabled `goreleaser build` for the unobtainable mid-pipe capture
  and rests it on "this repo already asserts that property (the reproducibility flags
  exist for the double-build determinism gate)". At source that assertion is narrower
  than the use:
  - The determinism gate is `ci.yml:216-273` / `Taskfile.yml:1559-1635`. Its blocking
    leg is **linux/amd64 only**; the linux/arm64 leg is `continue-on-error: true`,
    reported and non-blocking per D-03. **No leg covers darwin at all**, and every leg
    uses raw `go build`, not goreleaser.
  - The comparison is across **two different commands** — `goreleaser build` versus
    `goreleaser release --snapshot`. `Taskfile.yml:282-285` states the resolved version
    string is what `--snapshot` changes, and the darwin ldflags at
    `.goreleaser.yaml:62-69` embed `{{ .Tag }}`, `{{ .FullCommit }}` and
    `{{ .CommitDate }}` directly into the binary. `02-04` step 4 never specifies
    `--snapshot` for the baseline invocation, so the two runs can differ *before* quill
    for version-string reasons alone.
  - The assumption is load-bearing **twice**: it underwrites the `final != baseline`
    reading, and the mutation arm's inversion cannot happen at all unless the baseline
    is byte-equal to the release run's pre-quill bytes. A non-reproducible darwin build
    surfaces as a "neither verifies" hard failure that is indistinguishable from a
    genuine ordering finding.
  - Both lanes converge on the same remedy shape. pi: build the baseline twice and
    assert `baseline1 == baseline2` under a distinct `BASELINE-NONDETERMINISTIC` label
    before trusting any comparison — zero extra Apple submissions. Codex: additionally
    make both runs `goreleaser release --snapshot` with identical flags and generated
    config, differing *only* in the notarize enable predicate, rather than comparing
    `build` against `release`. Adopting both closes the finding completely.

- **HIGH (Codex only, orchestrator-confirmed) — the new post-release suite job is not
  instructed to check out the resolved release tag, so it would test the released
  binary against the default branch's tests.** `02-06-PLAN.md:210` tells the executor
  to mirror the existing job skeleton "checkout, Go setup and the Task install action,
  in the established order". Both existing checkout steps —
  `post-release-verify.yml:216` and `:262` — carry no `with: ref:`, and the workflow's
  own header at `:55-60` states that a `workflow_run` job's `GITHUB_REF`/`GITHUB_SHA`
  refer to the default branch and its tip. The new job runs `test/integration` against
  the downloaded release asset, so mirroring that skeleton pairs *binary@tag* with
  *tests@main*. Criterion 4's evidence ("the published binary passes its own release's
  suite") does not hold under that pairing, and a post-release red becomes ambiguous
  between a bad artifact and drifted tests. pi did not assess this surface. Remedy:
  `with: ref: ${{ needs.resolve-tag.outputs.tag }}` on the suite job's checkout, plus a
  workflow-shape test asserting that every job executing release-tag artifacts checks
  out the same resolved tag.

- **MEDIUM — a post-publish failure of `release:record-final-hashes` silently
  suppresses post-release verification.** Raised by both lanes. The step correctly
  fails loudly now, but it runs after GoReleaser has published, so the release workflow
  concludes failure with assets already live — and `post-release-verify.yml`'s
  `workflow_run` trigger fires only on `conclusion == 'success'`. Automatic Gatekeeper
  and suite jobs therefore never run for that tag. Neither `02-07` nor `02-06` names
  this interaction or instructs the manual `workflow_dispatch` recovery.

- **MEDIUM — `02-01`'s GitHub-digest cross-check fails open.** Both lanes flagged it.
  `02-01-PLAN.md:167` instructs the target to emit a sentinel and continue when GitHub
  supplies no `digest` field, while the digest cross-check is carried as a must-have
  and is the stated justification for the `jq` precondition. As written the target can
  pass with `digest_match=unknown` on every line and nobody notices; pi adds that the
  field is a relatively recent gh/API addition, so this is a live path rather than a
  theoretical one.

### Divergent Views

- **Overall risk: Codex HIGH vs pi MEDIUM.** Not a contradiction — different surfaces.
  Codex's second HIGH (post-release checkout ref) is in a file pi did not assess for
  that property; pi's risk rating is otherwise consistent with Codex's "MEDIUM once
  corrected".
- **`02-06`'s `needs: verify-supply-chain`.** pi reads it as a clean strengthening.
  Codex accepts the safety but notes it costs independent diagnostic evidence: a
  supply-chain failure now *skips* the suite instead of producing a second, separately
  interpretable signal — the exact tradeoff the neighbouring `self-upgrade` job
  deliberately refuses at `post-release-verify.yml:241-244`. Codex asks the plan to
  choose consciously and record which it chose; the plan currently says "belt and
  braces" without naming what the belt costs.
- **`02-05`'s per-tag applicability table.** Codex alone objects that requiring every
  released tag turns a verification document into a hand-maintained release ledger,
  and proposes boundary rows ("through v0.5.1", "first notarized release", "later
  releases") instead. pi did not flag it.

### Orchestrator source-grounding

Every HIGH and every MEDIUM above was re-verified by direct file read before being
recorded. Two additional checks worth carrying forward:

- **CONFIRMED — the pre-existing `self-upgrade` job has the same execute-before-verify
  exposure that `02-06` fixes for the new job, and no plan touches it.**
  `Taskfile.yml:1279-1291` does `gh release download`, then `chmod +x`, then executes
  `"${PRIOR_BIN}" upgrade "${TAG}"` with no cosign verification anywhere in between.
  This is pre-existing debt rather than a Phase 2 regression, and the job is
  deliberately independent for a recorded reason — but `T-02-19`'s corrected mitigation
  now asserts the team understands this ordering hazard while an identical one sits
  three jobs away untouched. Worth an explicit "accepted pre-existing exposure" note in
  `02-06`'s threat model rather than silence.
- **CONFIRMED — threat IDs are unique across the phase** after the `T-02-20`/`T-02-21`
  → `T-02-22`/`T-02-23` renumbering, and `02-02`'s `T-02-06` mitigation text still says
  the enabled-template test "resolves the template under both environments and asserts
  two different results" (`02-02-PLAN.md:400`) while Task 1 now specifies seven
  environments asserting TRUE once and FALSE six times (`02-02-PLAN.md:115,167`). A
  stale count in a threat-model row, not a design defect.
