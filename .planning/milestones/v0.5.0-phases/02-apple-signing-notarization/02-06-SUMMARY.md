---
phase: 02-apple-signing-notarization
plan: 06
subsystem: infra
tags: [github-actions, goreleaser, cosign, apple-notarization, taskfile, yaml-decoding, workflow-shape-tests]

# Dependency graph
requires:
  - phase: 02-01
    provides: "verify:gatekeeper (the D-19 spctl oracle, parameterised by GATEKEEPER_EXPECT) — reused unchanged for the post-release GREEN check"
  - phase: 02-02
    provides: ".goreleaser.yaml's notarize: enabled: template (five-credential conjunction) and signs: {ids: [raw], artifacts: binary}"
  - phase: 02-03
    provides: "resolveTestBinPath + CODEGRAPH_TEST_BIN in both test/integration and test/wireoracle TestMains, and the recorded verdict that wireoracle is IN SCOPE for a release-shaped binary"
  - phase: 02-04
    provides: "the local rehearsal's evidence: byte-identity chain proven, Assumption A5 (no secret leakage in quill's error output) resolved, and the darwin/amd64 assessment-not-execution distinction this plan's architecture-coverage decision relies on"
provides:
  - "The five Apple credential secrets wired into release.yml's release job (step-level env: only), with TestAppleSecretsScopedToSingleReleaseJob enforcing exclusivity across every workflow file"
  - "post-release-verify.yml's gatekeeper job — re-fires verify:gatekeeper against the published tag's darwin assets, both architectures, expecting accepted"
  - "post-release-verify.yml's notarized-suite job — runs test/integration + test/wireoracle against the re-downloaded, cosign-verified, notarized darwin/arm64 binary"
  - "Taskfile.yml's verify:notarized-suite target (integrity verified before execution, executed-test count from go test -json events)"
  - "TestPostReleaseJobsDeclareCheckoutPolicy — the two-class (latest-verifier / release-matched-test) checkout-pinning guard with completeness assertion"
  - "A filed todo (T-02-24) for the pre-existing, unmitigated download-then-execute exposure in verify:self-upgrade"
affects: [02-07]

# Actuals (#2632)
actuals:
  tokens: 11700
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Struct-tag YAML decoding (fullWorkflowDoc/fullWorkflowJob/fullWorkflowStep, map[string]any for env:/permissions:/with: values) rather than line-scanning regex, for workflow shapes where scope (workflow- vs job- vs step-level env:) determines correctness — empirically verified the 'on:' key does not hit YAML 1.1's implicit boolean-key resolution when bound via a struct field tag"
    - "Two-class completeness assertion (TestPostReleaseJobsDeclareCheckoutPolicy): every job id in a workflow file must appear in exactly one named policy class, both directions checked, so a newly added job fails loudly instead of silently escaping coverage"
    - "Verify-then-execute ordering enforced BOTH in the Taskfile target body (cosign verify-blob before chmod) AND in the job graph (needs: verify-supply-chain) — belt and braces, with the reporting-independence tradeoff this costs recorded in-comment against the sibling job that deliberately makes the opposite choice"

key-files:
  created:
    - .planning/todos/pending/2026-08-09-verify-self-upgrade-download-then-execute-has-no-signature-check.md
  modified:
    - .github/workflows/release.yml
    - .github/workflows/post-release-verify.yml
    - Taskfile.yml
    - internal/upgrade/release_workflow_shape_test.go
    - internal/upgrade/taskfile_shape_test.go

key-decisions:
  - "Architecture coverage for notarized-suite: darwin/arm64 NATIVE ONLY, darwin/amd64 deliberately excluded. No outward-facing CI dispatch was available in this execution context to empirically confirm Rosetta translation availability on namespace-profile-macos-6x14-tahoe, and the local rehearsal (02-04) only ASSESSED the amd64 binary with spctl, never executed it. Recorded as an explicit verdict in the job's own comment rather than improvised; deferred to a future phase that can confirm translation availability first."
  - "gatekeeper joins notarized-suite in the release-matched-test checkout-pinning class, even though it doesn't run go test specifically — the classifying property is the PINNING REQUIREMENT (Task 2 mandates checkout at the resolved tag so the Taskfile target invoked matches the release under test), not whether the job happens to invoke `go test`. Documented in both the test's var comment and the job's own comment."
  - "TestPostReleaseJobsDeclareCheckoutPolicy landed in internal/upgrade/release_workflow_shape_test.go, not internal/upgrade/taskfile_shape_test.go as Task 3's own <files> line named — its content (workflow YAML shape parsing: jobs, checkout steps, ref: values) matches release_workflow_shape_test.go's existing pattern, not taskfile_shape_test.go's task-block pattern. Both files are within the plan frontmatter's files_modified list, so this is a placement judgment call, not a scope violation."
  - "No masking step added to release.yml's Release step env: block. Assumption A5 (02-EVIDENCE.md) tested a real quill failure with a deliberately wrong MACOS_SIGN_PASSWORD and found zero certificate/key/password material in the error output — the P12 decode fails locally, in under a second, before any Apple network contact. Recorded in-comment rather than left as an unexplained absence."

requirements-completed: [SIGN-01, SIGN-02]

coverage:
  - id: D1
    description: "The five Apple credentials are readable by exactly one job in exactly one workflow (release.yml's release job), never by a pull_request/pull_request_target-triggerable workflow, always at step-level env: — asserted by a runtime-enumerating test, not by inspection"
    requirement: SIGN-01
    verification:
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestAppleSecretsScopedToSingleReleaseJob"
        status: pass
      - kind: unit
        ref: "Non-vacuity proof 1: a credential reference temporarily added to darwin-toolchain-canary.yml (pull_request-triggerable) — observed red, reverted, re-verified green"
        status: pass
      - kind: unit
        ref: "Non-vacuity proof 2: a credential temporarily moved to release.yml's job-level env: — observed red, reverted, re-verified green"
        status: pass
    human_judgment: false
  - id: D2
    description: "release.yml's file name, tag trigger, and single id-token: write job are unchanged; the credentials-scoping change did not widen the OIDC signing surface"
    requirement: SIGN-01
    verification:
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestOIDCWriteScopedToSingleGoreleaserJob"
        status: pass
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestReleaseWorkflowFileMatchesPattern and TestReleaseWorkflowTriggerIsTagPushOnly"
        status: pass
    human_judgment: false
  - id: D3
    description: "Every job in post-release-verify.yml, including the two new ones, carries the verbatim event-aware conclusion guard, and a dry evaluation confirms it runs (not skips) under workflow_dispatch"
    requirement: SIGN-02
    verification:
      - kind: other
        ref: "rg -c of the guard string == 5 (job count), verified via Bash during execution"
        status: pass
    human_judgment: false
  - id: D4
    description: "The gatekeeper job's and the notarized-suite job's structural correctness (checkout pinning, cosign verify-before-execute ordering, executed-test counting from go test -json, precondition sets) — whether they actually pass when dispatched against a real published release is plan 02-07's evidence, not this plan's"
    requirement: SIGN-02
    verification:
      - kind: unit
        ref: "internal/upgrade/taskfile_shape_test.go#TestVerifyNotarizedSuiteDeclaresNamedPreconditions"
        status: pass
      - kind: unit
        ref: "internal/upgrade/release_workflow_shape_test.go#TestPostReleaseJobsDeclareCheckoutPolicy, both directions of non-vacuity proven"
        status: pass
      - kind: other
        ref: "task lint:actions (actionlint over .github/workflows/*.yml) — exits 0"
        status: pass
    human_judgment: true
    rationale: "This plan wires and structurally verifies the CI jobs; it cannot dispatch them (outward-facing actions are explicitly prohibited for this executor). Whether the gatekeeper and notarized-suite jobs actually PASS against a real published release is plan 02-07's evidence, per this plan's own <success_criteria>."
duration: "not precisely tracked — commits landed within a 7-minute window (16:12–16:19 local) preceded by substantial multi-file context-gathering; no explicit session-start marker was recorded"
completed: 2026-08-09
status: complete
---

# Phase 2 Plan 06: Apple Credentials Wiring + Post-Release Verification Jobs Summary

**Wired the five Apple signing/notarization secrets into release.yml's single OIDC-bearing job (step-level env: only, machine-enforced exclusivity), and added two new post-release-verify.yml jobs — gatekeeper (re-fires the D-19 spctl oracle against published darwin assets, both architectures) and notarized-suite (runs the real test/integration + test/wireoracle suites against the cosign-verified, re-downloaded darwin/arm64 binary, arm64-only by a recorded architecture-coverage decision).**

## Performance

- **Duration:** not precisely tracked (see frontmatter `duration`)
- **Tasks:** 3
- **Files modified:** 5 modified, 1 created

## Accomplishments

- `MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_ISSUER_ID`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_KEY` now flow into release.yml's release job's `Release` step `env:` — the only place that can make `.goreleaser.yaml`'s `notarize: enabled:` template evaluate true — with a comment recording the config block it feeds, the silent-failure mode of its absence, and why no masking step was added (Assumption A5, no leakage observed).
- `TestAppleSecretsScopedToSingleReleaseJob` (`internal/upgrade`) enumerates `.github/workflows/` at runtime and proves: the five names appear only in release.yml's release job; only at step-level `env:`; no `pull_request`/`pull_request_target`-triggerable workflow references them or holds `id-token: write`; release.yml still holds `id-token: write` in exactly one job. Proven non-vacuous twice against real mutations, both reverted.
- `post-release-verify.yml` gained a `gatekeeper` job: matrix over both darwin architectures (`fail-fast: false`), checkout pinned to the resolved tag, invokes the SAME `verify:gatekeeper` Taskfile target plan 02-01 proved RED against v0.5.1, now expecting `accepted`. Deliberately independent of `verify-supply-chain` (assessment, not execution — the ordering hazard doesn't apply).
- `post-release-verify.yml` gained a `notarized-suite` job and `Taskfile.yml`'s `verify:notarized-suite` target: downloads the published darwin/arm64 binary and its cosign bundle, verifies the signature (byte-identical issuer/identity flags to `verify:release-assets`) BEFORE `chmod +x`, then runs `test/integration` and `test/wireoracle` against it with `CODEGRAPH_TEST_BIN` and `-count=1`. Executed-test count comes from `go test -json` events (a zero count is a hard failure regardless of exit status — proven reachable with a `-run` filter selecting no tests). `needs: [resolve-tag, verify-supply-chain]` — a deliberate trade against the sibling `self-upgrade` job's opposite independence choice, recorded in-comment on both jobs.
- `TestPostReleaseJobsDeclareCheckoutPolicy` encodes two checkout-pinning classes (latest-verifier: unpinned; release-matched-test: pinned to `needs.resolve-tag.outputs.tag`) as named data, with a completeness assertion that every job id in the file is classified. Proven non-vacuous twice: removing `ref:` from the suite job, and adding an unclassified dummy job — both observed red, reverted.
- Filed `.planning/todos/pending/2026-08-09-verify-self-upgrade-download-then-execute-has-no-signature-check.md` (T-02-24): `verify:self-upgrade` has the same download-then-execute ordering hazard this plan closed for `verify:notarized-suite`, left unmitigated because closing it here was out of this plan's scope — a concrete artifact, not a prose-only flag.

## Task Commits

Each task was committed atomically:

1. **Task 1: Apple credentials on the single OIDC-bearing release job** - `d63a006` (feat)
2. **Task 2: The post-release Gatekeeper job** - `df83d33` (feat)
3. **Task 3: The integration suite against the notarized published binary** - `708812f` (feat)

**Plan metadata:** this SUMMARY.md (worktree mode — STATE.md/ROADMAP.md excluded, orchestrator-owned)

## Files Created/Modified

- `.github/workflows/release.yml` - five Apple credential env: entries added to the Release step, step-level only; masking-not-needed comment
- `.github/workflows/post-release-verify.yml` - gatekeeper job (Task 2) and notarized-suite job (Task 3) added
- `Taskfile.yml` - `verify:notarized-suite` target added
- `internal/upgrade/release_workflow_shape_test.go` - `TestAppleSecretsScopedToSingleReleaseJob`, `TestPostReleaseJobsDeclareCheckoutPolicy`, and the shared `fullWorkflowDoc`/`fullWorkflowJob`/`fullWorkflowStep` YAML-decode types
- `internal/upgrade/taskfile_shape_test.go` - `TestVerifyNotarizedSuiteDeclaresNamedPreconditions`
- `.planning/todos/pending/2026-08-09-verify-self-upgrade-download-then-execute-has-no-signature-check.md` - new, T-02-24

## Decisions Made

See frontmatter `key-decisions`: architecture-coverage verdict (arm64-only, no confirmed Rosetta translation evidence), gatekeeper's checkout-class placement, `TestPostReleaseJobsDeclareCheckoutPolicy`'s file placement, and the no-masking-step decision.

## Deviations from Plan

### Auto-fixed Issues

None — no Rule 1/2/3 auto-fixes were required. `verify:gatekeeper`'s unclassifiable-verdict handling (Task 2's acceptance criterion asking to confirm this) was already correct as shipped by plan 02-01 — no change needed there.

**Total deviations:** 0 auto-fixed.
**Impact on plan:** None — plan executed as written, with two documented placement/scope judgment calls (see Decisions Made) rather than corrections.

## Issues Encountered

- **Self-inflicted near-miss, caught before it caused any loss.** During the `TestAppleSecretsScopedToSingleReleaseJob` non-vacuity proof (Task 1), `git checkout -- .github/workflows/release.yml` was used to revert a deliberate mutation — but this also discarded Task 1's own legitimate, still-uncommitted credential-wiring edit to the same file, since `git checkout --` reverts the ENTIRE working-tree file to HEAD, not just the most recent edit. Caught immediately via `git diff --stat` showing an empty diff where a real change was expected; the legitimate edit was re-applied and re-verified before committing. For the two later non-vacuity proofs (Task 2's checkout-guard, Task 3's checkout-policy completeness check), reverts were done via the `Edit` tool instead of `git checkout --`, specifically to avoid repeating this mistake against files carrying other uncommitted, not-yet-committed changes.
- **Acceptance-criterion literal-text mismatch, not a real gap.** Task 2's acceptance criteria state `rg -c 'write' post-release-verify.yml returns 0`. The file's own pre-existing header comment ("No write scope of any kind, anywhere in this file") already contains the literal substring "write", so the count is 1 both before and after this plan's edits — unchanged, and the file genuinely still grants no write PERMISSION (the intent the criterion was checking for). Recorded here rather than silently reinterpreting the criterion.

## Known Stubs

None.

## Next Phase Readiness

- The five Apple credentials are wired and scoped; the maintainer's `user_setup` step (creating the five REPOSITORY secrets on GitHub's dashboard) is the only remaining prerequisite for `.goreleaser.yaml`'s notarize pipe to activate on the next tag push.
- Both new post-release-verify.yml jobs (`gatekeeper`, `notarized-suite`) are structurally complete, actionlint-clean, and covered by shape tests — but neither has been dispatched against a real release in this execution context (outward-facing actions were explicitly prohibited). Their actual pass/fail behavior against a real published, notarized release is plan 02-07's evidence to gather.
- T-02-24 (verify:self-upgrade's own unmitigated download-then-execute exposure) is filed as a pending todo, not resolved — a maintainer decision for a future phase.
- No blockers for plan 02-07.

## Self-Check: PASSED

- All 6 key files confirmed present on disk: `.github/workflows/release.yml`, `.github/workflows/post-release-verify.yml`, `Taskfile.yml`, `internal/upgrade/release_workflow_shape_test.go`, `internal/upgrade/taskfile_shape_test.go`, `.planning/todos/pending/2026-08-09-verify-self-upgrade-download-then-execute-has-no-signature-check.md`.
- All three task commit hashes confirmed present in `git log`: `d63a006` (Task 1), `df83d33` (Task 2), `708812f` (Task 3).
- `go test ./internal/upgrade/ -v` — all tests pass (86 PASS, 0 FAIL, 1 SKIP with a stated, unrelated reason).
- `task lint:actions` (actionlint) exits 0.
- `task test:unit` passes across all packages.

---
*Phase: 02-apple-signing-notarization*
*Completed: 2026-08-09*
