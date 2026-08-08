---
phase: 1
reviewers: [codex, pi]
reviewed_at: 2026-08-08T15:49:21Z
plans_reviewed:
  - 01-01-PLAN.md
  - 01-02-PLAN.md
  - 01-03-PLAN.md
  - 01-04-PLAN.md
  - 01-05-PLAN.md
---

# Cross-AI Plan Review — Phase 1

## Codex Review

# Cross-AI Plan Review

## Overall Summary

The phase is thoughtfully decomposed and has unusually strong mutation-based verification, explicit trust-boundary analysis, and live-release evidence requirements. However, executing all five plans as written can still leave the phase goal false. Three issues are release-blocking: the post-release verifier races the asset-producing workflow, its `gh` commands lack authentication, and activating SBOM generation breaks the permanent cross-compile canary because `syft` is not installed there. Release reruns also lose the current `--clobber` behavior without enabling GoReleaser’s replacement setting.

Overall risk: **HIGH** until those execution seams are repaired.

---

# Plan 01 — Cross-Compile Spike

## Summary

Plan 01 preserves REL-05’s central evidence bar: both Linux binaries must execute on native hardware and index real source, not merely build or print a version. Its checkpoint makes the result externally inspectable. The main weakness is that the selected fixture does not conclusively exercise the external-scanner grammar surface that motivates the spike.

## Strengths

- The plan correctly requires actual indexing. `codegraph init [path]` creates the store and invokes the full indexer at [internal/cli/init.go:27](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/init.go:27), while `codegraph status --json` exposes machine-readable graph counts at [internal/cli/status.go:28](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/status.go:28). This supports a real non-zero graph assertion.

- Requiring architecture-specific execution is materially stronger than the current pipeline, where linux/arm64 is only cross-compiled using Zig at [.goreleaser.yaml:65](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:65).

- The permanent canary follows a proven repository pattern. The existing Darwin canary is dispatchable and path-scoped at [.github/workflows/darwin-toolchain-canary.yml:26](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/darwin-toolchain-canary.yml:26).

- The plan correctly separates compilation from execution evidence and requires logs containing `uname`, `file`, and non-zero graph counts.

## Concerns

- **MEDIUM — The fixture may not exercise the risky external-scanner path.** The checked-out repository is overwhelmingly Go; merely indexing it proves CGo tree-sitter works for the languages present, but not necessarily a grammar with a hand-written external C/C++ scanner. The plan itself acknowledges this uncertainty. Since REL-05 exists because of those scanners, a passing run could still miss the precise failure class under investigation.

- **LOW — Artifact handoff is underspecified.** GoReleaser writes binaries beneath generated `dist/` paths, while the plan asks `upload-artifact` to upload architecture-specific outputs found through `dist/artifacts.json`. The current release workflow performs this lookup in an explicit shell step at [.github/workflows/release.yml:164](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:164). The new canary needs an equally explicit, Taskfile-owned export/copy step or step outputs; `upload-artifact.path` cannot execute the `jq` lookup itself.

## Suggestions

- Commit a small multi-language fixture containing at least one grammar with an external C scanner and require specific nodes from it, in addition to indexing the repository.

- Add a Taskfile target that resolves `dist/artifacts.json` and copies both Linux binaries to fixed canary-export paths. Make the upload steps consume only those fixed paths.

- Prefer `status --json` and assert exact numeric JSON fields rather than parsing rendered human text.

## Risk Assessment

**MEDIUM.** The proof remains real-Linux execution plus indexing, but its fixture coverage should be tightened to prove the specific CGo/parser risk.

---

# Plan 02 — Archives, Checksums, Signing, and SBOMs

## Summary

Plan 02 has strong static contracts around raw asset naming, signature sidecars, checksum scope, and per-binary SBOMs. Two concrete integration defects remain: the archive-content claim is false under GoReleaser defaults, and enabling `sboms:` will break the dry-run/canary path because it does not install `syft`.

## Strengths

- The raw asset contract is grounded in real runtime behavior. `defaultDownload` downloads the raw asset and `<asset>.sigstore.json` at [internal/upgrade/upgrade.go:188](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:188), and `releaseAssetName` produces the expected stem at [internal/upgrade/upgrade.go:209](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:209).

- The existing test already independently pins all four raw asset names at [internal/upgrade/verify_release_e2e_test.go:30](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/verify_release_e2e_test.go:30).

- Explicit `checksum.ids: [raw, zip]` is preferable to relying on artifact-type defaults and makes the eight-subject scope testable.

- Explicit `sboms.artifacts: binary` correctly avoids accidentally generating archive-scoped SBOMs.

## Concerns

- **HIGH — Activating `sboms:` breaks `release:dry-run` and the permanent canary unless `syft` is installed.** Plan 01’s dry run skips only `publish,sign`; it does not skip SBOM generation. Plan 02 then requires `task release:dry-run` to remain green, but modifies neither the Taskfile nor the canary to install `syft`. The current pipeline installs Syft explicitly before invoking it at [.github/workflows/release.yml:266](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:266). After this plan, the Plan-01 canary will rerun on `.goreleaser.yaml` changes and fail before REL-05 execution evidence is reached.

- **MEDIUM — “Binary + LICENSE + README only” is false.** The repository has a real `CHANGELOG.md` at [CHANGELOG.md:1](/Volumes/Code/github.com/seanb4t/codegraph-go/CHANGELOG.md:1). GoReleaser v2.17.1’s default archive files include `LICENSE*`, `README*`, and `CHANGELOG*` at `/Users/sean/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.1/internal/pipe/archive/archive.go:83-91`. With no explicit `files:` override, the zip will contain the changelog too.

- **LOW — The custom text parser is needless risk.** The repository already uses a real YAML decoder for `.goreleaser.yaml` at [internal/upgrade/taskfile_shape_test.go:560](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/taskfile_shape_test.go:560). A hand-written folded-scalar/list parser increases brittleness without improving the invariant.

## Suggestions

- Expand Plan 02’s file set to install Syft in `linux-cross-canary.yml`, and add a `command -v syft` precondition to `release:dry-run`. Alternatively, explicitly skip SBOM generation in the cross-compile-only canary and add a separate full-composition dry run that installs Syft.

- Either accept and document `binary + LICENSE + README + CHANGELOG`, or declare an explicit `files:` list if “only” is truly required.

- Decode `.goreleaser.yaml` with the existing YAML library and typed structs instead of adding raw-text parsers.

## Risk Assessment

**HIGH.** As written, normal Wave-2 execution activates a pipe whose required tool is absent from the permanent canary and dry-run environment.

---

# Plan 03 — Single Release Job

## Summary

The single-job migration is coherent and preserves the SAN’s workflow-path/tag-ref identity. Its static tests are good. The major gaps are loss of release-rerun idempotency and broader exposure of the OIDC minting capability to every step and tool in the collapsed job.

## Strengths

- The SAN remains structurally compatible. Production verification accepts only the GitHub OIDC issuer and the anchored `release.yml@refs/tags/v*` workflow identity at [internal/upgrade/verify.go:41](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/verify.go:41).

- The existing workflow trigger matches that regex exactly at [.github/workflows/release.yml:47](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:47).

- Rewriting `TestDarwinLegsBuildNatively` instead of deleting it preserves the underlying DNS/linker safety property.

- Removing caller-supplied `GOOS`/`GOARCH` is correct for a full GoReleaser invocation because each build entry already declares its own target at [.goreleaser.yaml:46](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:46).

## Concerns

- **HIGH — Release reruns lose the current replacement behavior.** The existing publisher uses `gh release upload --clobber` at [.github/workflows/release.yml:318](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:318). The proposed migration deletes it, but `.goreleaser.yaml` has no `release.replace_existing_artifacts: true` block—it currently ends with `checksum:` at [.goreleaser.yaml:122](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:122). GoReleaser’s setting defaults false in `/Users/sean/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.1/pkg/config/config.go:671`. A rerun after partial upload can therefore fail on already-existing assets, undermining patch-forward recovery and the claim that GoReleaser handles asset replacement.

- **HIGH — The OIDC trust boundary expands materially.** Today `id-token: write` is confined to `assemble` at [.github/workflows/release.yml:215](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:215). In the collapsed job, checkout, setup-go, setup-zig, the local Task installer, Cosign installer, Syft installer, GoReleaser, repository Taskfile, and `.goreleaser.yaml` all execute in a job able to request an OIDC token. SHA pins mitigate Action drift, but repository-controlled build configuration and more third-party code now sit inside the signing boundary.

- **MEDIUM — The test described as “exactly one OIDC job” temporarily permits two.** The temporary allowance is honest, but the must-have wording overstates the Wave-2 state. Until Plan 04 removes the provenance job, the file still has two `id-token: write` permissions.

- **LOW — Tool installation strategy is left as an implementation-time fork.** The plan says either retain `goreleaser-action` or build from `go.tool.mod`. That decision affects the meaning of `GORELEASER_VERSION`, the tool-vulnerability gate, execution time, and supply-chain surface and should be fixed in the plan.

## Suggestions

- Add `release.replace_existing_artifacts: true` and a shape test for it, or retain an explicitly idempotent upload mechanism. Include a test or dry-run simulation of a rerun with pre-existing same-named assets.

- Fix the GoReleaser installation choice now. Prefer the pinned `go.tool.mod` build if preserving `TestGoreleaserPinParity` and `tool-vuln` visibility is the primary constraint.

- Document the expanded OIDC boundary explicitly. Minimize executable steps in that job, keep all Actions SHA-pinned, and add a test forbidding GoReleaser hooks or arbitrary `before`/`after` commands in `.goreleaser.yaml`.

- Change the interim must-have to “one application-signing job plus the explicitly temporary provenance holder,” then strengthen it to exactly one in Plan 04.

## Risk Assessment

**HIGH.** The topology is workable, but rerun safety and the expanded signing trust boundary require explicit controls before publishing.

---

# Plan 04 — Native GitHub Attestation

## Summary

Plan 04 correctly treats the attestation format change as a documentation and verification-contract migration, not merely a workflow edit. The proposed placement and permissions are sound. Its principal weakness is unnecessary decision overhead for a choice already locked by D-10.

## Strengths

- The plan correctly keeps application upgrade verification separate from provenance. `defaultVerify` hashes the downloaded binary and verifies its Cosign bundle at [internal/upgrade/upgrade.go:161](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:161); it has no SLSA dependency.

- Moving the attestor into the release job allows it to consume GoReleaser’s generated checksum file directly.

- Removing the reusable provenance job eliminates the second current OIDC holder at [.github/workflows/release.yml:350](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:350).

- The plan comprehensively updates the current verification surfaces instead of leaving an unusable command in public documentation.

## Concerns

- **LOW — The blocking command-choice checkpoint reopens a settled decision.** D-10 already names `gh attestation verify` as the accepted fallback, and the review request says that choice is binding. The checkpoint creates avoidable execution latency and a chance of divergence.

- **LOW — The attestation test should bind the actual checksum filename template, not merely look for a tag-templated path.** The source of truth is `.goreleaser.yaml`’s checksum template at [.goreleaser.yaml:122](/Volumes/Code/github.com/seanb4t/codegraph-go/.goreleaser.yaml:122). Independent loose parsing of the workflow string can permit a mismatch.

## Suggestions

- Remove Task 1’s choice and state `gh attestation verify` as the locked command.

- Add a cross-file shape test that resolves the checksum filename from `.goreleaser.yaml` and compares it exactly with `subject-checksums`.

- Retain the live Plan-05 attestation verification as the authoritative non-static proof.

## Risk Assessment

**MEDIUM.** The implementation direction is correct; most risk comes from its dependency on Plans 02–03 and the first real release.

---

# Plan 05 — Published Release Verification

## Summary

Plan 05 has the right goal—verification must use re-downloaded published assets—but its automation trigger is incorrectly ordered relative to this repository’s two-workflow release architecture. It also omits `GH_TOKEN` for every `gh` operation. Consequently, all five plans could execute through publication while the permanent verification workflow either races and fails or cannot authenticate.

## Strengths

- The checks validate exact sets, not lower bounds: eight checksum subjects, four raw binaries, four zips, and exclusions for signature/SBOM sidecars.

- The Cosign test uses the production SAN policy rather than a weaker identity. That policy is anchored at [internal/upgrade/verify.go:42](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/verify.go:42).

- Self-upgrade byte equality is the correct end-to-end assertion. The runtime really downloads the raw binary and matching sidecar at [internal/upgrade/upgrade.go:188](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:188), verifies before swap at [internal/upgrade/upgrade.go:136](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/upgrade.go:136), and supports a pinned version argument at [internal/cli/upgrade.go:31](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/upgrade.go:31).

- Read-only workflow permissions are appropriate for an external verifier.

## Concerns

- **HIGH — `release: published` races the asset-producing workflow and may never rerun.** Release-please creates the tag and GitHub Release itself; this is explicit at [.github/workflows/release-please.yml:1](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release-please.yml:1). The tag independently triggers the asset workflow at [.github/workflows/release.yml:47](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:47). Therefore, a `release: published` verification workflow starts when release-please publishes the initially empty release, not after GoReleaser finishes uploading assets. GoReleaser updating an already-published release does not guarantee a second `published` event. The verifier can race, fail on missing assets, and never automatically retry.

- **HIGH — The proposed `gh` commands have no authentication environment.** The plan passes only `TAG` and `REPO`. The current release workflow explicitly passes `GH_TOKEN` before using `gh` at [.github/workflows/release.yml:297](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:297). Merely granting `contents: read` does not export the token as `GH_TOKEN`; `gh release view`, `gh release download`, API queries, and likely attestation verification can fail before testing any claim.

- **HIGH — The self-upgrade job has the same release race.** Even if the prior binary is downloaded successfully, the new raw binary and its `.sigstore.json` may not exist when the job starts. A failure here does not distinguish a pipeline regression from the normal ordering of release-please and GoReleaser.

- **MEDIUM — “Exactly 17 assets” is stricter than the actual phase requirement and brittle across later phases.** Phase 2 notarization or Phase 3 cask publishing may legitimately add release artifacts. The permanent workflow is said to run on every future release, but a fixed total will become a maintenance trap. Exact equality is appropriate for the eight checksum subjects and required sidecars, but unrelated additional assets should be classified explicitly rather than automatically failing forever.

- **MEDIUM — The workflow needs an explicit target version for self-upgrade.** Plain `codegraph upgrade` resolves GitHub’s “latest” release at [internal/upgrade/release.go:55](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/upgrade/release.go:55), which is timing-sensitive. The automation should invoke `codegraph upgrade "$TAG"` using the supported positional version at [internal/cli/upgrade.go:31](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/upgrade.go:31).

## Suggestions

- Trigger verification after the release workflow completes, for example with `workflow_run` on the `release` workflow and `conclusion == success`, then derive the tag from the completed run. An even stronger design is a final verification job in `release.yml`, since it has a direct `needs:` edge to publication and attestation.

- Keep `workflow_dispatch` for re-verification of historical tags.

- Pass `GH_TOKEN: ${{ github.token }}` through each task step’s `env:`. Add a test ensuring every Taskfile target that calls `gh` has a token precondition.

- Add bounded retry/backoff for GitHub release-asset visibility even after workflow completion, since API/CDN consistency can lag briefly.

- Invoke the prior binary as `./codegraph upgrade "$TAG"` and assert that exact tag, avoiding the “latest” endpoint.

- Replace fixed total `17` with:

  - exact equality for the eight checksum subjects;
  - exactly one signature and SBOM per raw binary;
  - an explicit allowed set of auxiliary assets appropriate to the current milestone.

## Risk Assessment

**HIGH.** The verification workflow’s trigger and authentication are both incorrect for the repository’s actual release topology. These are phase-goal blockers, not polish issues.

---

# Final Risk Assessment

**Overall: HIGH**

The plans are strong at static invariants, mutation testing, and human-visible evidence, but four changes are required before execution:

1. Sequence post-release verification after `release.yml` completes, not on the initial `release: published` event.
2. Provide `GH_TOKEN` to every Taskfile target invoking `gh`.
3. Install or explicitly skip Syft in all dry-run/canary paths after `sboms:` becomes live.
4. Preserve rerun idempotency with `release.replace_existing_artifacts: true` or an equivalent tested mechanism.

After those fixes—and a scanner-bearing REL-05 fixture—the phase design becomes substantially safer and is likely to achieve its stated goal.

---

## pi Review

# Cross-AI Plan Review: Phase 1 — Cross-Compile Spike & `goreleaser release` Migration

I reviewed all five plans against the actual repository (branch `gsd/v0.5.0-macos-distribution-homebrew`), reading `.github/workflows/release.yml` (all 375 lines), `.goreleaser.yaml` (all 124), `internal/upgrade/verify.go`, `internal/upgrade/upgrade.go` (incl. `releaseAssetName()` at `upgrade.go:209`), `internal/upgrade/release_workflow_shape_test.go` (incl. the `needs_zig` assertion at `:477`), `internal/upgrade/taskfile_shape_test.go` (`inScopeJobs` at `:109`, `runBodyExceptions` at `:146`), `Taskfile.yml` (`check:darwin-release-build` at `:270`), `.github/workflows/release-please.yml`, and `docs/RELEASE.md` §a/b. Plan claims about the existing code check out almost everywhere; the concerns below are concentrated in GoReleaser/GitHub *runtime behavior* the plans assume but cannot currently exercise.

## 1. Summary

This is an unusually disciplined set of plans: the REL-05 spike has a genuinely falsifiable evidence bar (non-zero indexed graph on real, non-emulated hardware, with a pre-enumerated V1–V5 FAIL list written into the canary header before first dispatch), every new contract is held by a mutation-RED shape test, the cross-plan joint invariants (REL-07 single-writer, OIDC scoping) are explicitly assigned rather than assumed, and the prohibitions against the known failure modes (`${{ }}` injection, bare-tag Action pins, tag deletion, local-`dist/` verification) are carried as acceptance criteria rather than prose. The plans' claims about the existing codebase are accurate in every case I traced. The residual risk is not in the planning discipline but in three runtime behaviors of the GoReleaser `release:` pipe and the new workflows' event ordering that the dry-run configuration (`--skip=publish,sign`) structurally cannot exercise before the one-way publish — most sharply, the `post-release-verify.yml` `release: [published]` trigger races the very asset uploads it exists to verify.

## 2. Strengths

- **REL-05's evidence bar survives intact as checkable criteria.** 01-01 Task 2's `check:linux-cross-exec` requires `uname -m` + `file -b` ELF-machine agreement plus a *non-zero* file/symbol count from `codegraph status` after indexing a real tree — exit codes and `--version` are explicitly named as FAIL. The per-runner separation (`cross-build` on `namespace-profile-macos-6x14-tahoe`, exec legs on `namespace-profile-linux-amd64-4x8` / `-arm64-4x8`) means a red run names which layer broke. No silent degradation to build-exit-0 anywhere I could find.
- **The cosign SAN analysis is correct against the code.** `releaseWorkflowRefPattern` (`verify.go:43`) anchors on workflow file path + tag ref only, not job or runner — so collapsing three jobs into one genuinely does not move the SAN, and D-11's two mechanisms (live `cosign verify-blob` against re-downloaded assets + the single-`id-token: write` shape test) are the right way to convert "should" into "does".
- **Plan 01-01's transitional claims are accurate.** `TestDarwinLegsBuildNatively` does assert darwin `needs_zig: false` (`release_workflow_shape_test.go:477-478`), so the plan's "leave darwin entries untouched" instruction is verifiably correct, and the one-line amd64 flip to `needs_zig: true` is coherent with today's still-matrix pipeline.
- **The Rosetta/`xcrun` trap is correctly propagated.** 01-01 and 01-03 both carry the load-bearing native-GoReleaser-build ordering from `Taskfile.yml:296-305` ("build the tool with no GOOS/GOARCH, then invoke") — this is a real, documented failure class in this repo, not cargo-culted caution.
- **RESEARCH.md Pitfall 2 (GOOS/GOARCH env) is enforced in both directions** — as a plan prohibition in 01-03 and as an acceptance criterion (`rg -c 'GOOS:|GOARCH:'` returns 0).
- **The asset arithmetic is consistent.** 01-05's 17-asset expected count (4 raw + 4 zip + 4 `.sigstore.json` + 4 `.spdx.json` + 1 checksums) matches exactly what the 01-02 config shape (`binary_signs: artifacts: binary`, `sboms: artifacts: binary`, `checksum.ids: [raw, zip]`) will produce.
- **The stale-exception hygiene pattern is applied correctly**: 01-03's temporary `provenance:` allowance in the OIDC test mirrors the real `runBodyExceptions` staleness mechanism (`taskfile_shape_test.go:124-146`), so the allowance cannot silently outlive its job.

## 3. Concerns

- **HIGH — `post-release-verify.yml`'s `release: [published]` trigger races the asset uploads it verifies.** `release-please.yml` creates the tag **and** the GitHub Release in one API call (`.github/workflows/release-please.yml:1-14,57-92`); `release.yml` then fires on that tag push and uploads assets minutes later (today in `assemble`, lines 289-335; post-migration in the goreleaser job). GitHub emits the `release: published` event at Release-creation time, *before* the release pipeline has uploaded anything. 01-05 Task 1's `verify-supply-chain` job would therefore enumerate an empty asset list and hard-fail on every release — or worse, a maintainer re-running it via `workflow_dispatch` after assets land would mask that the auto path never works. The trigger should be `workflow_run` on completion of `release.yml` (or the job should poll `gh release view --json assets` until the 17-asset set is complete with a timeout). This is the phase's own stated philosophy — "a check that ran once being mistaken for a gate that keeps holding" — applying to the plan itself.
- **HIGH — the GoReleaser `release:` pipe's behavior against a release-please-created Release is never exercised before the one-way publish.** Today's hand-rolled publish step (release.yml:289-335) encodes three deliberate behaviors: (a) it never regenerates the release-please body or prerelease flag, (b) `--clobber` makes re-runs idempotent, (c) `*-*` tags get `--prerelease`. RESEARCH.md's own Don't-Hand-Roll table flags this: "verify its `--clobber`-equivalent behavior during D-06's `--snapshot` dry run before removing the hand-rolled branch entirely." But D-06's dry run is `goreleaser release --snapshot --skip=publish,sign` — **the publish pipe cannot be exercised by it at all**. Whether GoReleaser's default `release:` pipe preserves release-please's release body, how it handles asset replacement on re-run, and its `prerelease:` default for rc tags are all first exercised by the production `v0.5.0` release, where a wrong answer is permanent (D-07 forbids deletion). At minimum the plan should pin explicit `release:` config (`prerelease: auto`, body/name handling) rather than relying on defaults, and record the un-exercised surface as a named risk in 01-05's checkpoint context rather than leaving it implicit.
- **MEDIUM — 01-03 passes the wrong token env var to the goreleaser step.** Task 2's action specifies `GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}` on the `run: task release:goreleaser` step. `GH_TOKEN` is the `gh` CLI's variable (correct at release.yml:299 today, where `gh release upload` is the publisher); GoReleaser's `release:` pipe reads **`GITHUB_TOKEN`** by default. As written, the first real release would fail at the publish pipe. The `release:goreleaser` Taskfile precondition list also omits a token check, so nothing fails fast on this.
- **MEDIUM — the `binary_signs:` template resolution is never dynamically exercised pre-release.** The one contract whose breakage bricks every user's `codegraph upgrade` (the `${artifact}.sigstore.json` sidecar name; D-14) is held only by a *static* test asserting the template string in `.goreleaser.yaml`. Whether `${artifact}` in `binary_signs` resolves to the archive-templated asset name (as RESEARCH.md Pattern 2 assumes) rather than the raw build binary name (`codegraph`, identical across all four build ids — `.goreleaser.yaml:50,67,84,98`) is a runtime fact, and `--skip=sign` in every dry run means it is first resolved during the real release. If it resolves to `codegraph`, four colliding `codegraph.sigstore.json` files result. A cheap mitigation exists: a sign-enabled snapshot leg in the 01-01 canary using a throwaway local cosign key (`cosign generate-key-pair` + `COSIGN_PASSWORD`), which needs no OIDC.
- **MEDIUM — OIDC-minting reach genuinely broadens under the one-job topology, and the plans' guard counts jobs, not steps.** Today `id-token: write` lives on `assemble` (`namespace-profile-linux-amd64-2x4`, release.yml:221-227), which executes only pinned Actions plus hand-rolled shell. Post-migration the token-bearing job also executes the zig toolchain, a GoReleaser binary (built from `go.tool.mod` or installed by `goreleaser-action` — 01-03 deliberately leaves this either/or open), syft, and `actions/attest-build-provenance` (01-04), on the larger `macos-6x14-tahoe` profile. `TestOIDCWriteScopedToSingleGoreleaserJob` correctly asserts *one job* holds the token, but nothing bounds what executes *inside* it. This is inherent to D-09 and arguably fine — but the threat register (T-01-11) frames the mitigation as the job-count test, which does not address the intra-job surface. Worth an explicit accepted-risk sentence in 01-03's threat model rather than an implied one.
- **LOW — 01-01's `release:dry-run` gains an undeclared `syft` dependency once 01-02 lands.** The dry run skips `publish,sign` but *not* the `sboms:` pipe, so after 01-02's Task 2, `task release:dry-run` invokes `syft` (per the `sboms: cmd: syft` block). 01-02's acceptance criteria require `task release:dry-run` to exit 0, but neither plan adds `command -v syft` to the target's preconditions (contrast: `zig` gets one, and `check:darwin-release-build` gets `clang` at Taskfile.yml:290-291). On a contributor mac without syft this fails with a confusing mid-pipe error.
- **LOW — 01-04's Task 1 decision checkpoint re-opens a question this session has already adjudicated** (adjudication item 3: `gh attestation verify` is binding per D-10's stated fallback). The checkpoint recommends the right option, so this is wasted ceremony, not a defect — but the "halt" path and the `slsa-verifier verify-github-attestation` option are dead weight that could invite a relitigation the milestone has already closed.
- **LOW — 01-02's REL-09 flagged assumption (raw/zip atomic replacement on re-run) is real and unaddressed.** The plan assumes GoReleaser builds all archives before any upload, but on a *re-run* against a partially-published release, asset replacement granularity is per-artifact, not per-set — a stale-raw/new-zip window is possible. 01-05's set-equality check catches it after the fact on the happy path only. Minor, because re-runs are rare and D-07 makes recovery patch-forward, but the assumption row says "partially covered by 01-05" — it is weaker than that phrasing suggests.

## 4. Suggestions

1. **01-05 Task 1:** change the trigger to `workflow_run: workflows: [release.yml], types: [completed]` (with a conclusion check), or add an explicit wait-loop in `verify:release-assets` that polls `gh release view --json assets` until the expected 17-asset set is present (bounded, loud on timeout). Keep `workflow_dispatch` for re-runs.
2. **01-03 Task 2:** set `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}` (not `GH_TOKEN`) on the goreleaser step, and add a `release:goreleaser` precondition that a token is present.
3. **01-02 Task 2:** add an explicit `release:` block to `.goreleaser.yaml` pinning `prerelease: auto` and documented body/name behavior toward the release-please-created Release, replacing reliance on GoReleaser defaults for the three behaviors today's shell step encodes (release.yml:289-335).
4. **01-01:** extend the canary with one sign-enabled snapshot leg using a throwaway local cosign key, so the `${artifact}.sigstore.json` template resolution is dynamically proven before the production release; alternatively, have 01-05's checkpoint context explicitly name "sign-pipe template resolution unexercised" as a pre-merge prerequisite to verify another way.
5. **01-01 Task 3 / 01-02:** add `command -v syft` (and a `brew install syft` hint) to `release:dry-run`'s preconditions once the `sboms:` block is live.
6. **01-03:** pick the GoReleaser-install strategy in the plan text (recommend `goreleaser-action` `install-only` to match the pin `TestGoreleaserPinParity` already enforces) rather than delegating the either/or to the executor.
7. **01-04 Task 1:** collapse the checkpoint to a confirmation notice naming `gh attestation verify`, per the prior adjudication.

## 5. Risk Assessment

**MEDIUM.** The planning quality is high and every claim I traced against the repo is accurate — the existing-contract analysis (SAN anchor, `needs_zig` assertion, dead-config blocks, Rosetta trap, Taskfile single-definition convention) is correct in detail. What keeps this from LOW is that the phase's own core discipline — *don't trust a gate you haven't demonstrated* — has three holes at the migration's riskiest boundary: the publish pipe's behavior toward a release-please-created Release (unexercisable under `--skip=publish`), the `binary_signs` template resolution (unexercised under `--skip=sign`), and the verification workflow's trigger (races the uploads it checks). All three are correctable with small plan edits before execution; none requires re-architecture. With suggestions 1–3 applied, I'd assess LOW-to-MEDIUM, dominated by the intended, well-instrumented REL-05 spike risk itself.

### Sources (repo evidence)

- `.github/workflows/release.yml:215-335` — `assemble` job: hand-rolled checksums (`:238-244`), cosign loop (`:249-264`), publish step's body/prerelease/`--clobber` semantics (`:289-335`), `GH_TOKEN` usage (`:299`)
- `.github/workflows/release-please.yml:57-92` — Release object created at tag time, before `release.yml` uploads assets (the race in concern 1)
- `.goreleaser.yaml:48-103,115-124` — four build ids (all binaries named `codegraph`), dead `archives:`/`checksum:` blocks
- `internal/upgrade/verify.go:38-43` — `releaseWorkflowRefPattern` anchors file+ref, not job/runner
- `internal/upgrade/upgrade.go:195,209-211` — `assetName+".sigstore.json"` download contract; `releaseAssetName()` shape
- `internal/upgrade/release_workflow_shape_test.go:477` — darwin `needs_zig: false` assertion backing 01-01's transitional claim
- `internal/upgrade/taskfile_shape_test.go:109-119` — `release.yml` deliberately outside `inScopeJobs`
- `Taskfile.yml:270-348` — `check:darwin-release-build`'s `--snapshot` rationale and native-tool-build ordering (Rosetta/`xcrun` trap)

---

## Consensus Summary

Two prompt-fed, source-grounded reviewers (Codex, pi) independently reviewed all five plans with repo access. Neither ran without repo access; neither is diff-only; no lane was trimmed or stubbed, so both verdicts carry full consensus weight.

Both agree the planning discipline is unusually strong — mutation-RED shape tests on every new contract, explicit trust boundaries and a STRIDE register per plan, cross-plan joint invariants assigned rather than assumed, and prohibitions carried as acceptance criteria rather than prose. Both also agree the residual risk is concentrated in exactly one place: **GoReleaser and GitHub *runtime* behaviors that the chosen dry-run configuration (`--snapshot --skip=publish,sign`) structurally cannot exercise before a one-way, undeletable publish (D-07).** The plans' own stated philosophy — never trust a gate you have not demonstrated — is not applied to the publish and sign pipes themselves, nor to the post-release verifier's trigger.

Codex assessed overall risk **HIGH**; pi assessed **MEDIUM**, explicitly noting it would drop to LOW-MEDIUM with three small pre-execution plan edits. The divergence is one of severity weighting, not of mechanism: both name the same three seams.

The seven items adjudicated earlier this session were **not** re-litigated by either reviewer. pi explicitly acknowledged the `gh attestation verify` binding (item 3) and used the correct `internal/upgrade/upgrade.go:209` location for `releaseAssetName()` (item 6); Codex likewise cited `upgrade.go:209`. Both reviewers independently confirmed the arm64 runner is referenced without any provisioning task, and both confirmed the cosign SAN anchor is unaffected by the job collapse.

### Agreed Strengths

- **The cosign SAN survives the job collapse, and both reviewers verified it against source.** `releaseWorkflowRefPattern` (`internal/upgrade/verify.go:44`) anchors on workflow *file path + tag ref* only — never on job or runner — so collapsing three jobs into one genuinely cannot move the identity `codegraph upgrade` trusts. `release.yml:47-50`'s `push: tags: v[0-9]*` trigger matches it exactly.
- **REL-05's evidence bar is falsifiable and was not silently degraded.** Both reviewers confirm the plan requires `uname -m` + `file -b` ELF-machine agreement plus a *non-zero* file/symbol count from `codegraph status` after indexing a real tree, with exit codes and `--version` explicitly named as FAIL (01-01-PLAN.md:26, :42). pi verified `codegraph init`/`status --json` really do expose machine-readable graph counts (`internal/cli/init.go:27`, `internal/cli/status.go:28`). The D-04 V1–V5 FAIL list written into the canary header *before* first dispatch makes a FAIL declaration falsifiable rather than post-hoc.
- **Mutation-RED discipline is applied to every new contract**, and the raw-asset contract is grounded in real runtime behavior (`internal/upgrade/upgrade.go:188` downloads the raw asset plus `<asset>.sigstore.json`; `:209` builds the stem; `verify_release_e2e_test.go:30` already pins all four names independently).
- **Explicit `checksum.ids: [raw, zip]` and `sboms.artifacts: binary`** are preferred over relying on GoReleaser's default inclusion sets, making the 8-subject scope testable rather than assumed.
- **The stale-exception hygiene pattern is applied correctly** — 01-03's temporary `provenance:` allowance mirrors the real `runBodyExceptions` staleness mechanism (`taskfile_shape_test.go:124-146`), so the allowance cannot silently outlive its job, and 01-04 requires its deletion.
- **Rewriting `TestDarwinLegsBuildNatively` rather than deleting it** preserves the libresolv/DNS property that motivated native darwin builds; pi verified the test really does assert darwin `needs_zig: false` (`release_workflow_shape_test.go:477-478`), so 01-01's transitional claim is accurate.

### Agreed Concerns

Ordered by consensus weight, then severity.

1. **HIGH — `post-release-verify.yml`'s `release: [published]` trigger races the asset uploads it exists to verify.** *Both reviewers, independently, with the same mechanism.* `release-please.yml` creates the tag **and** the GitHub Release object in one API call (`.github/workflows/release-please.yml:57-92`); GitHub emits `release: published` at Release-creation time. `release.yml` only then fires on the tag push (`release.yml:47-50`) and uploads assets minutes later. The verifier therefore starts against an **empty asset list**, hard-fails on every release, and GoReleaser updating an already-published release does **not** guarantee a second `published` event — so it never automatically retries. Codex notes the self-upgrade job inherits the identical race, and that a failure there cannot be distinguished from a genuine pipeline regression. Fix: trigger on `workflow_run` (workflow `release.yml`, `conclusion == success`), or make it a final `needs:`-edged job inside `release.yml`; keep `workflow_dispatch` for historical re-verification. This is the phase's own "a check that ran once mistaken for a gate that keeps holding" failure mode, applied to the plan itself.

2. **HIGH — the GoReleaser `release:` pipe is unconfigured and never exercised before the one-way publish.** *Both reviewers.* Today's hand-rolled publish step (`release.yml:289-335`) encodes three deliberate behaviors that the migration deletes: it never regenerates the release-please body or prerelease flag; `gh release upload --clobber` (`release.yml:318`) makes re-runs idempotent; and `*-*` tags get `--prerelease`. `.goreleaser.yaml` has **no `release:` block at all** — it ends at `checksum:` (`:122-124`) — and GoReleaser v2.17.1's `replace_existing_artifacts` defaults to **false** (`pkg/config/config.go:671`). RESEARCH.md's own Don't-Hand-Roll table says to verify the `--clobber` equivalent "during D-06's `--snapshot` dry run", but that dry run is `--skip=publish,sign` — **the publish pipe cannot be exercised by it at all.** A rerun after a partial upload fails on existing assets, which is precisely the patch-forward recovery path D-07 makes mandatory. Fix: add an explicit `release:` block pinning `replace_existing_artifacts: true` and `prerelease: auto` plus documented body/name handling, with a shape test.

3. **HIGH — OIDC-minting reach broadens materially, and the guard counts jobs rather than steps.** *Both reviewers, plus independent source verification during this review.* Today `id-token: write` lives only on `assemble` (`release.yml:219-224`) and on the reusable `provenance` job (`:353-356`); the 4-leg `build` matrix — the job that compiles Go and CGo tree-sitter C via zig — declares **no `permissions:` key at all** and inherits only top-level `contents: read` (`release.yml:52-53`). After the collapse, `actions/checkout --fetch-depth 0`, `setup-go`, an unconditional `setup-zig`, the GoReleaser binary, `cosign`, `syft`, the repo Taskfile, `.goreleaser.yaml`, and four CGo cross-builds of every tree-sitter grammar all execute inside the single job able to mint a token whose SAN `internal/upgrade/verify.go:44` unconditionally trusts. Because that pattern is deliberately *job-agnostic* (it can only distinguish workflow file + ref, by design — `verify.go:31-40`), nothing downstream can notice. `T-01-11`'s mitigation (`TestOIDCWriteScopedToSingleGoreleaserJob`) asserts job **count and identity**; no threat row bounds intra-job execution, and `T-01-15` covers only third-party *Actions* ("no new Action is introduced"), which is true at file level but false relative to the *privileged job*. Fix: add an explicit accepted-risk row naming the intra-job surface, plus a test forbidding GoReleaser `hooks:`/`before:`/`after:` arbitrary commands in `.goreleaser.yaml`.

4. **HIGH — activating `sboms:` breaks `task release:dry-run` and the permanent canary, because neither installs `syft`.** *Codex HIGH, pi LOW — same mechanism, divergent severity.* The dry run skips `publish,sign` but **not** the `sboms:` pipe, so once 01-02 Task 2 lands, `release:dry-run` shells out to `syft`. 01-02's own acceptance criteria require that target to exit 0, but neither plan adds `syft` to the canary or a `command -v syft` precondition — contrast `zig`, which gets one, and `check:darwin-release-build`, which gets `clang` (`Taskfile.yml:290-291`). The current pipeline installs Syft explicitly before invoking it (`release.yml:266`). Consequence beyond a red dry run: 01-01's permanent canary re-fires on `.goreleaser.yaml` changes and will **fail before reaching REL-05's execution evidence**, blinding the phase's own blocking gate.

5. **HIGH — every `gh` invocation in 01-05 lacks an authentication environment.** *Codex.* The plan passes only `TAG` and `REPO`. Granting `contents: read` does **not** export `GH_TOKEN`; the current workflow passes it explicitly before using `gh` (`release.yml:297`). `gh release view`, `gh release download`, the API queries, and `gh attestation verify` can all fail before a single supply-chain claim is tested. Fix: `GH_TOKEN: ${{ github.token }}` on each step, plus a test asserting every Taskfile target invoking `gh` has a token precondition.

### Divergent Views

- **Overall risk level.** Codex: **HIGH** ("execution seams" are release-blocking). pi: **MEDIUM**, dropping to LOW-MEDIUM with three small plan edits. Both name the same seams; they disagree on whether these are blockers or pre-execution polish. Given that all five are correctable by plan edits before any code runs and none requires re-architecture, pi's framing is the more actionable — but Codex is right that executing as written publishes a broken verifier.

- **Whether REL-05's evidence bar survives as *checkable acceptance criteria*.** pi: yes, unequivocally — "No silent degradation to build-exit-0 anywhere I could find." Codex: yes in substance, but flags fixture coverage. **Independent source verification during this review found a real gap neither reviewer surfaced:** 01-01's Task 1 and Task 2 are the only tasks with `<acceptance_criteria>` blocks, and *every one of their eighteen criteria is satisfiable with zero Linux execution* — Task 1's strongest is `task release:dry-run` exits 0 plus a `file -b` static file-type inspection on the darwin host (`01-01-PLAN.md:215-216`, `:205`); Task 2's criteria are YAML text-shape assertions plus lint, and prove `check:linux-cross-exec` only by `task --list` **listing** it (`:339`). Task 3 — which carries the real bar (`:367-373`: non-zero counts, "a green step with no counts is a FAIL") — is a `checkpoint:human-verify` with **no `<acceptance_criteria>` block at all**, gated by a human typing `PASS <run-url>` (`:385`). The bar is therefore *falsifiable and precisely stated* but *human-checked, not machine-checked*. This is defensible — the canary can only run on GitHub Actions — but a cheap hardening exists and is not in the plan.

- **`binary_signs:` `${artifact}` template resolution.** pi raises it as MEDIUM and calls it the one contract whose breakage bricks every user's upgrade; Codex does not raise it. All four build entries name the binary `codegraph` (`.goreleaser.yaml:49,67,84,98`), so if `${artifact}` resolves to the build binary rather than the archive-templated asset name, four colliding `codegraph.sigstore.json` files result. `--skip=sign` in every dry run means this is first resolved during the real release. pi's mitigation — one sign-enabled snapshot leg in the canary using a throwaway local cosign key (no OIDC needed) — is cheap and worth adopting.

- **Fixture adequacy for REL-05.** Codex MEDIUM: indexing this predominantly-Go repository may not exercise a grammar with a hand-written external C scanner, which is the specific failure class motivating the spike. pi does not raise it. **Not counted as an open concern** — 01-01-PLAN.md:402 already carries this verbatim as a flagged assumption with an explicit "mitigated, not closed" disposition, which is the required handling.

---

## Verification coverage (source-grounding audit)

Both lanes were prompt-fed with the full source-grounding instruction block and both demonstrated real repo access. Neither output carries the `[reviewed-without-repo-access]` marker; neither is a diff-only lane. No lane was budget-trimmed and no lane produced a diagnostic stub, so both verdicts are weighted at full consensus strength.

**Citation density.** Codex cited 24 distinct `file:line` locations across 9 files, including two into the GoReleaser v2.17.1 module source in the local module cache (`internal/pipe/archive/archive.go:83-91`, `pkg/config/config.go:671`) — evidence of reading the dependency, not just the repo. pi cited 8 source clusters and listed them explicitly, including four the review request never named (`release_workflow_shape_test.go:477`, `taskfile_shape_test.go:109-119`, `Taskfile.yml:270-348`, `release-please.yml:57-92`).

**Independent corroboration performed during synthesis.** The following reviewer claims were re-verified directly against the working tree before being promoted to consensus concerns:

| Claim | Verified | Evidence |
|---|---|---|
| `build` job holds no OIDC permission | Yes | `release.yml:73` declares no `permissions:` key; top-level is `contents: read` (`:52-53`) |
| `assemble` and `provenance` both hold `id-token: write` today | Yes | `release.yml:221`, `release.yml:355` |
| SAN pattern is job-agnostic | Yes | `internal/upgrade/verify.go:44`; rationale comment `:31-40` |
| `.goreleaser.yaml` has no `release:` block | Yes | file ends at `checksum:` `:122-124` |
| All four build ids name the binary `codegraph` | Yes | `.goreleaser.yaml:49,67,84,98` |
| Hand-rolled `sha256sum` step exists | Yes | `release.yml:238-244` |
| `01-01`'s canary file does not yet exist | Yes | `.github/workflows/linux-cross-canary.yml` absent; plan unexecuted |
| `01-04` closes the OIDC count to exactly one | Yes | `01-04-PLAN.md:27,47,217-219,242-243,279-280` |

**Two findings originated in synthesis rather than in either lane**, and are recorded above: the REL-05 machine-checkability gap (Divergent Views §2) and the `nscloud-cache-action` ambiguity in 01-03's step list — `release.yml:126-129` carries it in `build` today, and 01-03's replacement step enumeration (`01-03-PLAN.md:281-299`) neither retains nor removes it.

**Adjudicated-item hygiene.** Zero of the seven previously adjudicated items were re-raised as concerns by either lane. pi affirmatively acknowledged items 3 and 6; both lanes independently used the corrected `internal/upgrade/upgrade.go:209` location for `releaseAssetName()`.
