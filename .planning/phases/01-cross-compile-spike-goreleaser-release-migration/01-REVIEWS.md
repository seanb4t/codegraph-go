---
phase: 1
reviewers: [codex, pi]
reviewed_at: 2026-08-08T16:28:42Z
plans_reviewed:
  - 01-01-PLAN.md
  - 01-02-PLAN.md
  - 01-03-PLAN.md
  - 01-04-PLAN.md
  - 01-05-PLAN.md
  - 01-06-PLAN.md
---

# Cross-AI Plan Review — Phase 1

## Codex Review

## Summary

Most cycle-1 fixes hold, including the ordering, authentication, OIDC-surface, SBOM-tooling, and machine-checkable REL-05 repairs. Two release-critical issues remain. First, 01-02’s proposed `binary_signs.signature: "${artifact}.sigstore.json"` is incompatible with GoReleaser’s actual binary-signing semantics: `${artifact}` is the pre-archive binary path/name, and all four builds produce a binary named `codegraph`. Plan 01-06 will therefore expose the predicted collision, but it provides no remediation branch and forbids changing the faulty configuration. Second, 01-05 assumes `workflow_run.head_branch` reliably contains the triggering tag, while GitHub only guarantees that the downstream workflow’s own `GITHUB_REF`/`GITHUB_SHA` refer to the default branch. The verifier needs an explicit, validated tag-resolution mechanism. Consequently, executing all six plans as written does not reliably achieve the phase goal.

## Cycle-1 Fix Verification

| Finding | Verdict | Evidence and mechanism |
|---|---|---|
| H1: `release:[published]` races uploads | **PARTIAL** | Switching to `workflow_run` correctly waits for the upstream workflow to complete. This fits the current sequence: release-please creates the tag/release ([release-please.yml:57](.github/workflows/release-please.yml#L57)), then the tag triggers `release` ([release.yml:47](.github/workflows/release.yml#L47)). However, 01-05 derives `$TAG` from `github.event.workflow_run.head_branch` without proving tag semantics. GitHub documents that a `workflow_run` job’s own `GITHUB_REF` and `GITHUB_SHA` are the default branch and its tip, not the upstream tag. The ordering fix holds; tag recovery remains under-specified. |
| H2: publish pipe unconfigured; `--clobber` deleted | **HOLDS** | 01-02 Task 3 pins `replace_existing_artifacts: true` and `prerelease: auto`; 01-03 JOINT #2 requires that configuration to coexist with removal of `--clobber`. This correctly preserves the behavior currently provided at [release.yml:318](.github/workflows/release.yml#L318)-[325](.github/workflows/release.yml#L325). The unexercised publish pipe is explicitly carried to the one-way checkpoint. |
| H3: OIDC reach broadens | **HOLDS** | 01-03 now acknowledges the expanded privileged surface instead of treating job count as sufficient. It forbids GoReleaser hooks, removes the mutable cache currently used at [release.yml:126](.github/workflows/release.yml#L126), limits the job to one Taskfile shell entry point, and records the CGo build inside the OIDC boundary as an accepted residual. |
| H4: SBOM activation breaks dry run/canary | **HOLDS** | 01-01 adds `syft`/`cosign` preconditions and installs both before the dry run. This correctly anticipates the currently separate syft installation at [release.yml:266](.github/workflows/release.yml#L266) and the fact that `--skip=publish,sign` does not skip SBOM generation. |
| H5: verifier has no `gh` authentication | **HOLDS** | 01-05 places `GH_TOKEN` on every `task verify:*` step and adds Taskfile preconditions. The distinction from GoReleaser’s `GITHUB_TOKEN` is correct; the current `gh` publisher likewise uses `GH_TOKEN` at [release.yml:297](.github/workflows/release.yml#L297)-[300](.github/workflows/release.yml#L300). |
| P2: `${artifact}` never dynamically exercised | **BROKEN** | 01-06 adds the right kind of experiment, but the configuration it tests is very likely already known to fail. Every build names its binary `codegraph` at [.goreleaser.yaml:47](.goreleaser.yaml#L47), [65](.goreleaser.yaml#L65), [84](.goreleaser.yaml#L84), and [96](.goreleaser.yaml#L96). GoReleaser’s pinned implementation assigns `${artifact}` from the binary artifact’s path and later publishes the signature artifact name using `art.Name`; see `sign.go:179–197` and `274–293` in `goreleaser/v2@v2.17.1`. Its default signature template adds OS/architecture specifically to avoid collisions (`sign_binary.go:16`). Moreover, keyed cosign signing requires `--key=<path>` in the arguments; `COSIGN_PASSWORD` only decrypts the selected key. 01-06 does not specify how the unchanged production args gain `--key`. |
| S1: REL-05 evidence not machine-checkable | **HOLDS** | 01-01 now requires nonzero status counts and emits two machine-readable `REL05-EVIDENCE` lines. The intended JSON fields exist as `fileCount` and `nodeCount` at [status.go:51](internal/query/status.go#L51)-[52](internal/query/status.go#L52), and Task 2 explicitly instructs the executor to confirm them before scripting. |

Spot checks of the remaining cycle-1 repairs:

- C2, C3, C4, C5, C8, C9, C10, C11, P1, P6, S2, and C7/P5 are incorporated coherently.
- C6 is only partial: 01-03 Task 1 says to always build GoReleaser from source, while Task 2 says the target should prefer the Action-installed binary and build only as a fallback.
- The external-scanner fixture rejection is explicitly recorded and was not silently dropped.

## Strengths

- The phase now distinguishes static shape checks, snapshot experiments, real-Linux execution, and published-release verification instead of treating them as interchangeable evidence.
- REL-05 is genuinely gated on two real architectures and the CGo parsing path, not build success or `--version`.
- The two-sided JOINT checks in wave 2 accurately encode branch end-state invariants without reopening the adjudicated atomicity question.
- The post-release verifier uses least privilege and classifies required, allowed, and unclassified assets rather than hard-coding a fragile total.
- `codegraph upgrade "$TAG"` correctly avoids the timing-sensitive latest-release resolver implemented at [release.go:55](internal/upgrade/release.go#L55).
- Typed YAML decoding matches the established repository pattern at [taskfile_shape_test.go:548](internal/upgrade/taskfile_shape_test.go#L548)-[590](internal/upgrade/taskfile_shape_test.go#L590).

## Concerns

- **HIGH — The proposed signature naming mechanism cannot produce the required published sidecars as specified.**

  `binary_signs` operates on `artifact.Binary` objects before archiving. GoReleaser sets `${artifact}` to the binary path, then sets the published signature artifact’s name using the binary’s `art.Name`. All four binary names are `codegraph`, so `"${artifact}.sigstore.json"` does not naturally become `codegraph_<tag>_<goos>_<goarch>.sigstore.json`.

  Plan 01-06 correctly anticipates this exact failure but says that `.goreleaser.yaml` must not be modified and defines no fail branch other than an unmet merge prerequisite. That means the six-plan execution can stall after wave 3 without a valid production configuration.

  Concrete plan change: move the dynamic experiment ahead of finalizing 01-02’s signing mechanism, or let 01-06 amend `.goreleaser.yaml` after observing RED. Evaluate a post-archive `signs:` configuration over the raw `formats: [binary]` artifacts, or use a signature template that explicitly derives the published tag/OS/arch name. The chosen configuration must then be exercised in both keyed rehearsal and production-keyless argument forms.

- **HIGH — The keyed rehearsal does not currently select the throwaway key.**

  01-02’s proposed production args are:

  ```yaml
  sign-blob --bundle=${signature} ${artifact} --yes
  ```

  01-06 generates a key and sets `COSIGN_PASSWORD`, but a password does not select a private key. GoReleaser’s official keyed-cosign example includes `--key=cosign.key`; the unchanged config does not. Therefore the rehearsal will attempt keyless signing and fail without OIDC, or otherwise exercise a different mechanism than intended.

  Concrete plan change: specify the exact mechanism. Prefer a temporary `cosign` wrapper placed first on `PATH` that injects `--key="$TEMP_KEY"` only for rehearsal while preserving GoReleaser’s artifact/signature substitutions, or add a templated optional key argument proven to be absent in production and present in rehearsal. Do not rely on an unspecified environment variable.

- **MEDIUM — `workflow_run.head_branch` is not a sufficiently proven tag source.**

  The upstream workflow is tag-triggered at [release.yml:47](.github/workflows/release.yml#L47)-[50](.github/workflows/release.yml#L50), but GitHub documents the downstream `workflow_run` context’s own `GITHUB_REF` and `GITHUB_SHA` as the default branch and its latest commit. The plan assumes `workflow_run.head_branch` is the tag and passes it straight to release verification.

  Concrete plan change: add a first job/step that resolves and validates the tag. It should require a `v[0-9]*` value, verify `refs/tags/$TAG` exists, and verify that the tag’s peeled commit equals `github.event.workflow_run.head_sha`. If `head_branch` is empty or not a tag, query tag refs pointing to `head_sha` and require exactly one matching release tag. Emit a `TAG-EVIDENCE` line and use its job output everywhere. Keep the manual dispatch input as the explicit historical-tag path.

- **MEDIUM — Prior-release selection is ambiguous for historical dispatches and prereleases.**

  01-05 says to obtain “the release immediately preceding `$TAG` via the `gh` API,” but does not define ordering or filtering. API list order is publication chronology, not necessarily semantic-version predecessor order. A historical re-verification can therefore select a newer release, and an RC can select the wrong stable/prerelease neighbor.

  Concrete plan change: enumerate non-draft releases, parse semantic versions, filter to versions strictly less than `$TAG`, and select the maximum under an explicit prerelease policy. Add tests for stable-after-RC, historical dispatch, drafts, and a first-release/no-predecessor case.

- **LOW — GoReleaser installation behavior remains contradictory.**

  01-03 Task 1 instructs the target to build GoReleaser natively before invocation; Task 2 instructs it to prefer the Action-installed binary and only build locally as fallback. The latter is the stated cycle-1 decision and preserves the meaning of `GORELEASER_VERSION`.

  Concrete plan change: rewrite Task 1 to one algorithm: use `command -v goreleaser` when available, otherwise build the pinned `go.tool.mod` version into a temporary path. Assert `goreleaser --version` matches the pin before release.

## Suggestions

- Add a small config-level integration test using the pinned GoReleaser module’s actual `dist/artifacts.json` after signing. Assert both signature artifact `Name` and `Path`; filesystem basename count alone can miss distinct paths whose published names collide.
- In 01-05, verify the upstream workflow identity using `github.event.workflow_run.path == ".github/workflows/release.yml"` or equivalent metadata in addition to the workflow display name.
- Have the post-release workflow record upstream run ID, upstream `head_sha`, resolved tag, and tag commit in one evidence line. That makes the workflow-run-to-release binding independently inspectable.
- Resolve the 01-06 signing failure branch before execution: “if RED, choose mechanism X and repeat” is needed just as REL-05 has an enumerated fallback.

## Risk Assessment

**Overall risk: HIGH.**

The cycle-1 revisions materially improved the plan set, and five of the seven principal repairs hold. However, the release-critical sidecar contract remains technically unsound under GoReleaser’s actual `binary_signs` implementation, and the new rehearsal cannot currently select its throwaway key. The downstream verifier also lacks a fully reliable binding from the completed upstream run to the exact tag being verified.

**Verdict: NO-GO as written.** Executing these six plans now is likely to reach 01-06, expose colliding or wrongly named signature artifacts—or fail by attempting keyless signing without OIDC—and then halt without an authorized remediation path. Fix the signing mechanism/rehearsal and make tag resolution explicit; after those changes, the remaining structure is strong enough to achieve the phase goal.

---

## pi Review

I have enough evidence. A few mechanism-level checks surfaced real issues; writing the review.

# Cross-AI Plan Review — Cycle 2

## 1. Summary

The cycle-1 fixes overwhelmingly hold, and they hold in the right way: the revisions do not just add prose, they add machine-checkable assertions (JOINT criteria, `REL05-EVIDENCE`/`SIGN-EVIDENCE` lines, negative-grep criteria) that I traced against the live tree and found correctly anchored (e.g., `release.yml:238-244` really does contain the `sha256sum` step the new tests will catch; `internal/upgrade/upgrade.go:209` is `releaseAssetName()` as cited; `internal/cli/upgrade.go` really does accept `upgrade [version]` positionally, confirming C10's fix). However, cycle-2 verification surfaced **two genuinely new mechanism bugs**: (a) plan 01-05's `workflow_dispatch` fallback is silently dead because both jobs carry an unconditional `workflow_run.conclusion` guard, and (b) plan 01-06's "point cosign at the throwaway key via the environment" is not executable as specified — the `binary_signs:` args shipped by 01-02 contain no `--key` flag, and cosign's `env://` key reference does not work through GoReleaser's sign pipe (goreleaser/goreleaser#4221), so the sign leg will fall into the keyless OIDC flow it was designed to avoid. Both are fixable with small plan-text edits. Overall risk: **MEDIUM**, down from cycle 1.

## 2. Cycle-1 Fix Verification

| Finding | Verdict | Evidence |
|---|---|---|
| H1 `release:[published]` race | **HOLDS, with one new bug** (see Concern 1) | `release-please` does create tag+Release together (release.yml:289-296 comment confirms); the `workflow_run` + `conclusion=='success'` + bounded-retry design is correct. `workflows: ["release"]` matches `name: release` at `release.yml:46`. Negative-grep criterion is sound. |
| H2 `release:` pipe unconfigured | **HOLDS** | 01-02 Task 3 pins `replace_existing_artifacts: true` (correct: today's idempotency comes from `gh release upload --clobber` at `release.yml:325`, which 01-03 deletes); JOINT #2 is a real two-sided check; the "unexercisable" framing in 01-05 Task 2's NAMED UNEXERCISED SURFACES is honest and correct — no rehearsal exists short of publishing. |
| H3 OIDC intra-job reach | **HOLDS** | T-01-29's enumeration matches the tree: `nscloud-cache-action` is at `release.yml:126-129`, `id-token: write` at `:221`, top-level `permissions: contents: read` at `:52-53`. `TestNoGoreleaserHooksInReleaseConfig` is the right closure for the arbitrary-command path. Accepted-residual framing for CGo-in-boundary is honest. |
| H4 sboms breaks dry-run | **HOLDS** | Correct sequencing: preconditions + installer steps land in wave 1, ahead of 01-02's `sboms:` activation in wave 2. `--skip=publish,sign` indeed does not skip the sboms pipe. |
| H5 no auth env for `gh` | **HOLDS** | `GH_TOKEN: ${{ github.token }}` on verify steps is correct for the `gh` CLI; the explicit contrast with 01-03's `GITHUB_TOKEN` (GoReleaser's release pipe) is right — today's publisher uses `GH_TOKEN` at `release.yml:297` because it's `gh release upload`. Not an inconsistency. |
| P2 `${artifact}` never exercised | **HOLDS in intent, mechanism broken** (Concern 2) | The rehearsal design is exactly right — all four build ids name the binary `codegraph` (`.goreleaser.yaml:49,67,84,98`, verified), so a wrong resolution collides, and there is a real chance it *does* resolve to the binary path (see Concern 2). The plan correctly gates 01-05's merge on it. But the key-injection mechanism doesn't work as written. |
| S1 REL-05 human-only bar | **HOLDS** | `REL05-EVIDENCE` line + Task-3 criteria block checkable from `gh run view` output converts the bar to machine-checked. Runner labels match provisioned profiles (do-not-re-litigate #1 confirmed: arm64 leg is a `runs-on:` value). |
| C2/C3/C4/C5/C6/C8/C9/C10/C11/P1/P6/S2/C7/P5 | **HOLD (spot-checked)** | C3: `CHANGELOG.md` exists at repo root, so the corrected default-set wording is accurate. C10: `cobra.MaximumNArgs(1)` + `Use: "upgrade [version]"` at `internal/cli/upgrade.go:31-37` confirms `upgrade "$TAG"`. C9: classification replaces fixed-17 correctly. C6: `goreleaser-action` with `install-only: true` is already the pattern at `release.yml:137-143`, so the decision is consistent with the tree. |

## 3. Strengths

- **The JOINT criteria are real and correctly scoped.** JOINT #1/#2 read the other plan's file without adding it to `files_modified`, preserving the wave-2 zero-overlap invariant; and the "why no commit-ordering criterion" paragraph in 01-03 is the right argument (tag-push-only trigger at `release.yml:47-50` + squash-only + release-please tag authority ⇒ `main` never observes a half-migrated state).
- **The throwaway-key insight is the best idea in the revision set.** It converts the one contract that bricks every user's `codegraph upgrade` from a static assertion into a dynamic observation *before* the one-way publish, and the distinctness-separate-from-count assertion targets the actual failure mode (all four build ids produce `codegraph`, verified above).
- **01-04's binding-decision block is a model for adjudicated choices** — verbatim command string, no paraphrase, reversibility preserved at the correct gate (01-05 Task 2), and the `verify.plan-structure` warning pre-annotated as expected.
- **Threat-model honesty improved materially**: T-01-29 records the CGo-compile-inside-the-OIDC-boundary as an *accepted* residual rather than claiming mitigation; the REL-09 transient-window assumption is recorded as accepted-not-covered (P6). This is how residual risk should be written down.
- **01-06's flagged-assumption row on snapshot-vs-tag equivalence** correctly narrows rather than claims closure, and hands the remainder to 01-05's post-release check.

## 4. Concerns

- **[MEDIUM] 01-05: the `workflow_dispatch` fallback is dead.** Task 1 requires BOTH jobs to carry `if: ${{ github.event.workflow_run.conclusion == 'success' }}`. On manual dispatch, `github.event.workflow_run` is null, the guard evaluates false, and both jobs silently skip — so the retained "re-verify historical tags" trigger (an acceptance criterion: `workflow_dispatch:` with a `tag` input) can never run, and it skips rather than failing, so nobody would notice. **Fix:** change the guard in both jobs to `if: github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'`, and add an acceptance criterion asserting that shape.
- **[MEDIUM] 01-06: the keyed-signing mechanism is unexecutable as written.** The `binary_signs:` args shipped by 01-02 are fixed: `["sign-blob", "--bundle=${signature}", "${artifact}", "--yes"]` — no `--key`. Without `--key`, cosign performs the *keyless* OIDC flow, which requires a browser/device interaction and fails on a headless runner (or hangs). Passing the key "via the environment" does not work: cosign's `env://KEY_VAR` key reference must still appear in the `--key` flag, and GoReleaser's pipe has been reported not to substitute it (goreleaser/goreleaser#4221). 01-06's own prohibition forbids editing `.goreleaser.yaml`, so the executor will be forced to improvise mid-task. **Fix (plan-text change, one paragraph in Task 1):** specify that the target generates a *temporary copy* of `.goreleaser.yaml` (e.g. `dist/.goreleaser.signtest.yaml`) with `--key=<temp key path>` appended to the `binary_signs` args and `COSIGN_PASSWORD` in the pipe env, invokes `goreleaser release -f <copy> --snapshot --skip=publish --clean`, and deletes the copy on exit — keeping the committed config untouched, which is what the prohibition actually protects. Note this slightly weakens the rehearsal's equivalence claim (args differ from production by exactly one flag); record that in the flagged-assumption row rather than hiding it.
- **[MEDIUM] 01-06/01-02: if the rehearsal fails (likely), there is no remediation path.** For `artifacts: binary`, GoReleaser's sign pipes template `${artifact}` from the *binary* artifact (named `codegraph` in all four builds), and sidecars land next to the binary in per-build `dist/` subdirs with **four identical basenames** — publishing collides on GitHub's basename-keyed asset store. This is precisely what 01-06 exists to catch, but neither plan says what to do when the expected failure occurs, so the executor will invent the fix at the worst moment. **Fix:** add a short "on failure" branch to 01-06 Task 1 naming the candidate remediation (switch signing from `binary_signs:` to `signs:` over the `archives[id=raw]` outputs — whose names are already the templated `codegraph_<tag>_<goos>_<goarch>` — or a `signature` template keyed on per-build id), routed back through 01-02's static test which must then assert the *new* template.
- **[LOW] 01-05 Task 3 `read_first` contradicts the H1 fix.** Its first bullet still says the post-release workflow runs "automatically on `release: published`" — pre-revision text. An executor reading it re-learns the racing trigger the plan just removed. **Fix:** update that bullet to say `workflow_run`.
- **[LOW] 01-03: the OIDC mutation-RED demonstration is awkward as specified.** After the collapse, `release.yml` has exactly two jobs; "add `id-token: write` to a THIRD job" requires inventing a job to mutate. **Fix:** reword the demonstration as "add `id-token: write` to any job other than `release` (e.g. a scratch job)" and note it must be reverted, or mutate the `provenance:` allowance's staleness companion instead.

## 5. Suggestions

- **01-06:** record the resolved four sidecar names into `dist/export/` as an artifact upload, so 01-05's merge prerequisite 5 can cite an artifact, not just a log line (logs expire; the checkpoint is human).
- **01-05 Task 1:** `verify:self-upgrade` runs `codegraph upgrade "$TAG"` — note that `upgrade.Run` takes `Options.Version` positionally and hits `releases/latest` only when empty (`internal/upgrade/upgrade.go:93-109`); also consider asserting the prior binary's version string first, so a "prior == target" resolution failure reads clearly.
- **01-01:** the exec legs `actions/download-artifact` across jobs in the same run needs no token — fine — but add `if-no-files-found`-equivalent (`download-artifact` fails hard by default; just confirm and note it).
- **Phase-wide:** when 01-05 records the `release` job's wall-clock (Open Question 3), also record per-pipe durations from GoReleaser's own log; Phase 2's notarization lands in the same serialized job and the number will matter.

## 6. Risk Assessment

**MEDIUM.** The cycle-1 fixes hold against the tree; the architecture (spike-first, gated publish, permanent canaries) is sound and the wave order is internally consistent — nothing in wave N depends on an artifact that wave N+1 produces, and the one structural impossibility (rehearsing the publish pipe) is correctly accepted rather than faked. The residual risk concentrates in two executable-mechanism bugs (Concerns 1 and 2) that are small plan-text edits away from resolution, plus one likely-eventual discovery (Concern 3) that the plan set is *designed* to surface but should pre-arm with a remediation path.

**Verdict on the phase goal:** executing the six plans in the declared wave order — *with Concerns 1–3 addressed* — makes the phase goal true: REL-05 is decided on re-inspectable evidence before anything is built on it, REL-06/07/09 land atomically via squash-merge with joint end-state checks, and REL-08 is proven against re-downloaded published assets by automation that re-fires on every future release. Without the two MEDIUM fixes, the phase still likely succeeds, but 01-06 (the gate on the publish checkpoint) will stall on an unexecutable mechanism, and the historical re-verification path will be silently dead.

---

## Consensus Summary

This is **cycle 2** of the convergence loop. Both lanes (Codex, pi) were prompted to judge whether
the cycle-1 fixes hold, not to re-derive cycle 1, and both had repo access and cited `file:line`
evidence throughout. Both are prompt-fed, source-grounded lanes — neither carries a `diff-only` or
`[reviewed-without-repo-access]` caveat, so both count at full consensus weight.

**Verdict: the cycle-1 repairs substantially hold.** Both reviewers independently graded H2, H3, H4,
H5 and S1 as HOLDS with concrete evidence, and both confirmed the C2–C11 / P1 / P6 / S2 / C7-P5
incorporations landed coherently. Both also confirmed the C1 rejection was recorded rather than
silently dropped. Neither reviewer re-raised any of the nine adjudicated items.

**But cycle 2 surfaced two genuinely new, converging mechanism defects in the signing rehearsal
(plan 01-06 + the config plan 01-02 ships), and both are blocking.** They were found independently
by both lanes from different starting points, which is the strongest signal this review produced.

Overall risk: the lanes diverge on the headline number — Codex says **HIGH / NO-GO as written**, pi
says **MEDIUM**. The divergence is about consequence, not about mechanism: both agree the signing
rehearsal cannot execute as specified. Codex weights that as phase-blocking because 01-06 gates the
one-way publish checkpoint in 01-05 Task 2; pi weights it as "small plan-text edits away." Since the
defect sits on the gate protecting the single irreversible action in the phase, this review adopts
the more conservative reading: **the two signing findings are HIGH and must be resolved before
execution.**

### Agreed Strengths

- **The cycle-1 fixes are machine-checkable, not prose.** Both lanes specifically praised that the
  revisions added JOINT criteria, `REL05-EVIDENCE` / `SIGN-EVIDENCE` greppable lines, and negative-grep
  criteria — and both traced those anchors to real tree locations (`release.yml:238-244`,
  `internal/upgrade/upgrade.go:209`, `internal/query/status.go:51-52`,
  `internal/cli/upgrade.go:31-37`).
- **The phase now separates evidence classes** — static shape checks, snapshot experiments, real-Linux
  execution, and published-release verification are no longer treated as interchangeable.
- **REL-05 is gated on real execution on two architectures through the CGo parsing path**, not on
  build success or `--version`.
- **The two-sided JOINT checks in wave 2** encode branch end-state invariants correctly without
  reopening the adjudicated REL-07 atomicity question.
- **Threat-model honesty improved materially** — T-01-29 records CGo-compile-inside-the-OIDC-boundary
  as an *accepted* residual rather than claiming mitigation, and P6's REL-09 transient window is
  recorded as accepted-not-covered.
- **The throwaway-key idea itself is sound** (pi calls it "the best idea in the revision set"): it
  converts the one contract that bricks every user's `codegraph upgrade` from a static assertion into
  a dynamic observation *before* the irreversible publish. The idea is right; only its mechanism is
  broken.
- **`codegraph upgrade "$TAG"`** correctly sidesteps the timing-sensitive `releases/latest` resolver.

### Agreed Concerns

Ordered by severity. The first two are the blocking findings.

1. **[HIGH — both lanes, independently] The keyed signing rehearsal in 01-06 cannot select its
   throwaway key, so it will attempt keyless OIDC and fail on a headless runner.**
   Plan 01-02 pins the args verbatim as
   `args: ["sign-blob", "--bundle=${signature}", "${artifact}", "--yes"]` (`01-02-PLAN.md:311`,
   `:337`) — there is **no `--key` flag**. Plan 01-06 supplies only `cosign generate-key-pair` plus
   `COSIGN_PASSWORD` through the step `env:` block (`01-06-PLAN.md:161`, `:267`). A password decrypts
   a selected key; it does not select one. Without `--key`, `cosign sign-blob` performs the keyless
   Fulcio/OIDC flow — and 01-06's own prohibition (`01-06-PLAN.md:36`) correctly denies the job
   `id-token: write`, so the rehearsal hangs or hard-fails. pi additionally notes that cosign's
   `env://` key reference must still appear inside the `--key` flag and has been reported not to
   substitute through GoReleaser's sign pipe (goreleaser/goreleaser#4221).
   Compounding it: `01-06-PLAN.md:37` prohibits modifying `.goreleaser.yaml`, so an executor hitting
   this has no sanctioned move and will improvise mid-task.
   **Plan change needed:** name the exact key-injection mechanism in 01-06 Task 1. pi's concrete
   proposal — generate a *temporary copy* of `.goreleaser.yaml` (e.g. `dist/.goreleaser.signtest.yaml`)
   with `--key=<temp key path>` appended to the `binary_signs` args, invoke
   `goreleaser release -f <copy> --snapshot --skip=publish --clean`, delete on exit — keeps the
   committed config untouched, which is what the prohibition actually protects. Codex's alternative is
   a `cosign` wrapper first on `PATH` injecting `--key` only for rehearsal. Either way, record in the
   flagged-assumption row that the rehearsal args now differ from production by exactly one flag.

2. **[HIGH — Codex HIGH / pi MEDIUM] `binary_signs.signature: "${artifact}.sigstore.json"` is very
   likely the wrong template, and no plan has a remediation branch for the RED that 01-06 exists to
   produce.**
   All four builds name the binary `codegraph` (`.goreleaser.yaml:47/65/84/96`, verified by both
   lanes). For `artifacts: binary`, GoReleaser templates `${artifact}` from the pre-archive *binary*
   artifact and publishes the signature under the binary's `art.Name`, so the four sidecars collide on
   a single published `codegraph.sigstore.json` rather than becoming
   `codegraph_<tag>_<goos>_<goarch>.sigstore.json`. Codex cites `sign.go:179-197` / `274-293` in the
   pinned `goreleaser/v2@v2.17.1`; both lanes independently note GoReleaser's *default* binary-sign
   template embeds OS/arch precisely to avoid this collision. 01-02's static test asserts the template
   *string*, so it passes while the runtime behavior is wrong — exactly the gap 01-06 was created to
   close.
   The problem is what happens next: 01-06 gates 01-05's merge, prohibits editing `.goreleaser.yaml`,
   and defines no fail branch. The six-plan execution can therefore reach wave 3, correctly detect the
   defect, and halt with no authorized path forward.
   **Plan change needed:** add an explicit "on RED" branch to 01-06 Task 1 naming the candidate
   remediation and routing it back through 01-02's static test, which must then assert the *new*
   template. Both lanes name the same candidates: move signing to `signs:` over the
   `archives[id=raw]` outputs (already named `codegraph_<tag>_<goos>_<goarch>`), or use a signature
   template that explicitly derives the tag/OS/arch name. Codex additionally recommends sequencing —
   run the dynamic experiment *before* 01-02 finalizes the signing mechanism, or explicitly authorize
   01-06 to amend `.goreleaser.yaml` after observing RED.

3. **[MEDIUM — pi, new; Codex did not catch this] 01-05's retained `workflow_dispatch` fallback is
   silently dead.**
   `01-05-PLAN.md:168` retains `workflow_dispatch:` with a `tag` input for re-verifying historical
   tags, but `01-05-PLAN.md:163` and acceptance criterion `01-05-PLAN.md:282` *require* BOTH jobs to
   carry `if: ${{ github.event.workflow_run.conclusion == 'success' }}` (the criterion mandates the
   grep return exactly 2). On a manual dispatch `github.event.workflow_run` is null, the guard
   evaluates false, and both jobs **skip rather than fail** — so the historical re-verification path
   can never run and nothing surfaces the fact.
   **Plan change needed:** change the mandated guard in both jobs to
   `if: github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'`, and
   update acceptance criterion `01-05-PLAN.md:282` to assert that shape instead of the current one.

4. **[MEDIUM — Codex; pi graded this side as holding] The tag the verifier checks is not proven to be
   the tag that was released.**
   `01-05-PLAN.md:166` asserts parenthetically that "a tag push populates
   `github.event.workflow_run.head_branch` with the tag name" and passes it straight into
   verification. Codex's objection is that GitHub only guarantees the *downstream* job's own
   `GITHUB_REF`/`GITHUB_SHA` refer to the default branch, and that the plan asserts the `head_branch`
   tag semantics without validating them. This is a **divergent view** (see below), but the hardening
   is cheap and is needed anyway once concern 3 is fixed, because the dispatch path must resolve `$TAG`
   from its own input.
   **Plan change needed:** add a first job/step that resolves and validates `$TAG` — require a
   `v[0-9]*` shape, verify `refs/tags/$TAG` exists, and verify the tag's peeled commit equals
   `github.event.workflow_run.head_sha`; fall back to querying tag refs pointing at `head_sha` and
   requiring exactly one match. Emit a `TAG-EVIDENCE` line and consume the job output everywhere.
   Keep `workflow_dispatch`'s input as the explicit historical path.

5. **[MEDIUM — Codex] Prior-release selection is unspecified and will mis-select.**
   `01-05-PLAN.md:249` says only "the release immediately preceding `$TAG` via the `gh` API". The API
   lists in publication chronology, not semver predecessor order, so a historical dispatch can select a
   *newer* release and an RC can select the wrong stable/prerelease neighbour.
   **Plan change needed:** specify the algorithm — enumerate non-draft releases, parse semver, filter
   to strictly-less-than `$TAG`, select the maximum under a stated prerelease policy — and add the
   stable-after-RC, historical-dispatch, drafts, and no-predecessor cases as criteria.

6. **[LOW — pi] `01-05-PLAN.md:405` still describes the workflow as running "automatically on
   `release: published`"** — un-updated pre-revision text inside Task 3's `read_first`. An executor
   reading it re-learns the exact racing trigger the H1 fix removed.
   **Plan change needed:** update that bullet to say `workflow_run`.

7. **[LOW — Codex] 01-03's GoReleaser install instruction contradicts its own C6 decision.**
   `01-03-PLAN.md:177-179` (Task 1) instructs unconditionally to "build the tool with
   `GOWORK=off go build -modfile=go.tool.mod` … then invoke that binary", while Task 2 (`:335-340`,
   recorded at `:486`) decides to keep `goreleaser-action install-only` in CI and have the Taskfile
   *prefer* a `goreleaser` already on `PATH`, building from source **only as the local fallback**.
   As written, Task 1 pays a from-source build inside the OIDC-bearing job — the precise cost Task 2's
   rationale says to avoid.
   **Plan change needed:** rewrite Task 1 to the single algorithm (use `command -v goreleaser` when
   present, else build the pinned `go.tool.mod` version to a temp path; assert `goreleaser --version`
   matches the pin).

8. **[LOW — pi] 01-03's OIDC mutation-RED demonstration names a job that will not exist.**
   `01-03-PLAN.md:395-396` and `:468` specify demonstrating RED by adding `id-token: write` to "a
   THIRD job", but after the collapse `release.yml` has exactly two (`release` + the temporarily
   retained `provenance:`), so the executor must invent one.
   **Plan change needed:** reword as "add `id-token: write` to any job other than `release` (e.g. a
   scratch job), then revert".

### Divergent Views

- **Overall risk rating.** Codex: **HIGH, NO-GO as written**. pi: **MEDIUM**, "still likely succeeds"
  without the fixes. They agree on every mechanism; they disagree on blast radius. This review sides
  with Codex because the defective mechanism sits on the gate guarding the phase's only irreversible
  action.
- **`workflow_run.head_branch` as a tag source (concern 4).** pi graded H1 as holding and treated
  `head_branch` carrying the tag name as correct; Codex treats it as an unvalidated assumption and
  wants an explicit resolve-and-verify step. pi is probably right about GitHub's actual payload
  behavior for tag pushes, but Codex's hardening costs little and becomes *required* once the
  `workflow_dispatch` guard bug (concern 3) is fixed. Recorded as actionable on those grounds rather
  than as a disputed factual claim.
- **Whether concern 2 is a defect or a successful detection.** pi frames the `${artifact}` collision as
  "a likely-eventual discovery the plan set is *designed* to surface"; Codex frames shipping a
  probably-wrong template in 01-02 as itself the defect. Both nonetheless converge on the same required
  plan edit: pre-arm the remediation path.
- **C6 (GoReleaser install).** Codex flags the Task-1/Task-2 contradiction; pi graded C6 as holding
  (it checked the recorded decision at `01-03-PLAN.md:486`, not Task 1's action text). Independently
  verified against the tree: Codex is correct — the contradiction is real.

### Not Re-Litigated

Neither reviewer re-raised any of the nine adjudicated items (arm64 profile provisioning, REL-07
atomicity, the `gh attestation verify` attestor, the deliberately-stale REQUIREMENTS.md REL-08 row,
`zig cc` byte changes, `releaseAssetName()`'s location, the seven spec-less probe rows, 01-04's
removed checkpoint, or the structurally-unexercisable `release:` publish pipe). pi explicitly
confirmed several of them against the tree and called the "NAMED UNEXERCISED SURFACES" framing
"honest and correct — no rehearsal exists short of publishing". The C1 rejection was likewise
confirmed as recorded, not dropped.
