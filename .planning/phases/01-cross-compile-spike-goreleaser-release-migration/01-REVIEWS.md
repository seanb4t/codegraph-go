---
phase: 1
reviewers: [codex, pi]
reviewed_at: 2026-08-08T17:05:01Z
review_cycle: 3
plans_reviewed:
  - 01-01-PLAN.md
  - 01-02-PLAN.md
  - 01-03-PLAN.md
  - 01-04-PLAN.md
  - 01-05-PLAN.md
  - 01-06-PLAN.md
---

# Cross-AI Plan Review — Phase 1 — Cycle 3 (FINAL)

## Codex Review

## Summary

The cycle-2 fixes landed correctly. The key-injection rehearsal is executable against GoReleaser v2.17.1, the RED-path signing remediation is sufficient, and the post-release workflow’s dispatch guard, tag-output wiring, and predecessor selection are coherently specified. No materially unresolved concern remains. The main residual risk is empirical—the first real publish pipe execution—which the plans already identify and gate appropriately.

## Cycle-2 Fix Verdicts

| Cycle-2 fix | Verdict | Evidence |
|---|---|---|
| HIGH-1 — cosign rehearsal lacked `--key` | **RESOLVED** | Plan 01-06 injects `--key=<absolute path>` immediately after `sign-blob` in a generated config and invokes it with `-f` ([01-06-PLAN.md:204](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-06-PLAN.md:204)). GoReleaser expands each configured argument and passes the resulting list directly to `exec.CommandContext` ([sign.go:205](/Users/sean/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.1/internal/pipe/sign/sign.go:205), [sign.go:249](/Users/sean/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.1/internal/pipe/sign/sign.go:249)). The process environment is inherited before pipe-specific additions, so `COSIGN_PASSWORD` reaches cosign ([sign.go:180](/Users/sean/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.1/internal/pipe/sign/sign.go:180), [sign.go:250](/Users/sean/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.1/internal/pipe/sign/sign.go:250)). |
| HIGH-2 — no remediation after signature-name collision | **RESOLVED** | The plan now classifies collision, wrong-but-distinct, and green results, then authorizes two remediation candidates with mandatory rerun and static-test updates ([01-06-PLAN.md:271](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-06-PLAN.md:271), [01-06-PLAN.md:326](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-06-PLAN.md:326)). Candidate A is supported by v2.17.1: `signs.artifacts` accepts `binary`, `ids` filtering is applied, and raw-format archives become `UploadableBinary` artifacts carrying archive ID `raw` and the renamed asset name ([config.go:984](/Users/sean/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.1/pkg/config/config.go:984), [sign.go:111](/Users/sean/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.1/internal/pipe/sign/sign.go:111), [archive.go:296](/Users/sean/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.1/internal/pipe/archive/archive.go:296)). If its filesystem-name validation rejects that shape, Candidate B remains authorized and validated. |
| MEDIUM — dead `workflow_dispatch` fallback | **RESOLVED** | The required guard now explicitly admits non-`workflow_run` events, and the acceptance criteria require a dispatch observed executing rather than merely appearing green ([01-05-PLAN.md:172](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-05-PLAN.md:172), [01-05-PLAN.md:354](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-05-PLAN.md:354)). |
| MEDIUM — tag resolution unvalidated | **RESOLVED** | `resolve-tag` validates tag shape and existence, binds the peeled tag commit to the upstream `head_sha`, emits `TAG-EVIDENCE`, and exposes one validated job output consumed via `needs` by both downstream jobs ([01-05-PLAN.md:189](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-05-PLAN.md:189), [01-05-PLAN.md:217](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-05-PLAN.md:217)). Standard GitHub Actions job outputs remain available through `needs` under `workflow_run`; nothing in this trigger changes that mechanism. |
| MEDIUM — predecessor selection unspecified | **RESOLVED** | The plan specifies draft removal, semver parsing, strict `< $TAG` filtering, maximum selection, and a stable/prerelease policy, with behavioral checks for historical dispatches, RC adjacency, drafts, and no-predecessor cases ([01-05-PLAN.md:301](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-05-PLAN.md:301), [01-05-PLAN.md:371](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-05-PLAN.md:371)). |
| LOW — stale `release: published` text | **RESOLVED** | Task 3 now explicitly reads evidence from the successful `workflow_run` path and its validated `TAG-EVIDENCE`, not the rejected publication event ([01-05-PLAN.md:489](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-05-PLAN.md:489)). |
| LOW — unconditional GoReleaser source build contradicted install strategy | **RESOLVED** | Plan 01-03 now mandates one prefer-then-build algorithm and a version check on either path ([01-03-PLAN.md:177](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-03-PLAN.md:177)). This preserves the existing v2.17.1 pin in `go.tool.mod` and the workflow’s matching environment value ([go.tool.mod:234](/Volumes/Code/github.com/seanb4t/codegraph-go/go.tool.mod:234), [release.yml:55](/Volumes/Code/github.com/seanb4t/codegraph-go/.github/workflows/release.yml:55)). |
| LOW — “third job” mutation named no concrete job | **RESOLVED** | The plan now explicitly creates and removes a throwaway scratch job, and acceptance requires that no scratch job survives and the final token-holder count returns to two ([01-03-PLAN.md:424](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-03-PLAN.md:424), [01-03-PLAN.md:461](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-03-PLAN.md:461)). |

## Concerns

No unresolved actionable concerns.

One implementation nuance is worth recognizing but does not require a plan change: Candidate A’s signature artifact may have the correct published `Name` while its on-disk signature basename follows the underlying binary path. GoReleaser initially expands `${artifact}` from `art.Path`, then records the signature artifact name using `art.Name` ([sign.go:179](/Users/sean/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.1/internal/pipe/sign/sign.go:179), [sign.go:274](/Users/sean/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.1/internal/pipe/sign/sign.go:274)). The plan’s mandatory filesystem validation will catch this and route to Candidate B, so it cannot silently reach publication.

The external-config location is also safe: v2.17.1 loads the supplied path but does not change the working directory ([config.go:37](/Users/sean/go/pkg/mod/github.com/goreleaser/goreleaser/v2@v2.17.1/cmd/config.go:37)); `dist` therefore remains rooted at the invocation working directory, as the plan additionally verifies.

## Risk Assessment

**MEDIUM.** The plans are implementation-ready and the cycle-2 repairs converge. Remaining risk comes from inherently empirical operations: macOS-hosted CGo cross-compilation, real runner behavior, and the first live GoReleaser publish-pipe execution. Those risks are explicitly gated by recorded canary evidence, sign-pipe rehearsal, a one-way publish checkpoint, and post-release verification against re-downloaded assets.

---

## pi Review

I now have everything I need. All claims verified against the pinned `goreleaser/v2@v2.17.1` module source in the local module cache and the repo tree. Writing the review.

---

# Cross-AI Plan Review — Phase 1 — Cycle 3 (FINAL)

## 1. Summary

The cycle-2 fixes mostly landed correctly, and I verified the load-bearing mechanisms against the **pinned `goreleaser/v2@v2.17.1` module source** in the local module cache, not just the plan text. HIGH-1's key-injection mechanism is sound: `-f <copy-outside-repo>` does **not** re-root `dist/` (it stays CWD-relative), and `--key` in the `sign-blob` args is valid. HIGH-2's premise is **confirmed real** — all four build ids name the binary `codegraph` (`.goreleaser.yaml:49,67,84,98`), `binary_signs:` filters the raw `artifact.Binary` build outputs (`internal/pipe/sign/sign_binary.go:80`), and the published signature artifact name derives from `art.Name` (`sign.go:275-276`), so the shipped config would publish four colliding `codegraph.sigstore.json` assets. Candidate A (`signs:` over the raw archive outputs) is valid for v2.17.1 with the enum value `binary` plus `ids: [raw]` (`sign.go:113-115` → `UploadableBinary`; `archive.go:308-325` → renamed name + `ExtraID: raw`).

**However, tracing the sign pipe end-to-end surfaced two genuinely unresolved HIGH findings in the same collision class** — one in 01-06's new GREEN/RED oracle (it observes the wrong signal and would reject the correct fix), and one in 01-02's `sboms:` block (the identical `${artifact}` collision, statically pinned as "correct" by a demonstrated-RED test, with no rehearsal and no remediation branch). Both are invisible to the executing agent and both defeat verification integrity — exactly what this final cycle exists to catch.

## 2. Cycle-2 Fix Verdicts

| Cycle-2 fix | Verdict | Evidence |
|---|---|---|
| **HIGH-1** — key injection via generated config copy, `-f` outside repo | **RESOLVED** | `dist` is a plain relative default (`internal/pipe/dist/dist.go:31-32`) used via `os.Stat`/`os.MkdirAll` against the process CWD (`dist.go:23,41,61`) — no `filepath.Join(dir-of-config, …)` anywhere in the pipe. So `-f "$TMPDIR/.goreleaser.signtest.yaml"` run from the repo root leaves `dist/` at the repo root, and the plan's post-run `dist/artifacts.json`-exists assertion is the correct guard. `--key` position: the sign pipe executes `exec.CommandContext(ctx, cfg.Cmd, args...)` with args in config order (`sign.go:206-215, 251`), and `cosign sign-blob --key=<path> --bundle=… <artifact> --yes` is valid keyful signing (no OIDC). The additions-only diff guard and trap-cleaned `mktemp -d` are sound. |
| **HIGH-2** — on-RED remediation branch | **PARTIALLY RESOLVED** | Branch structure, classification, candidate ordering, and the five-step validation are all sound, and Candidate A is mechanically valid for v2.17.1 (see Finding 2 below for the exact enum). **But the observation oracle the branch is keyed on is broken — see Finding 1.** As specified, applying the *correct* Candidate A would still report RED, sending the executor into the plan's own "both candidates failed → stop and report" deadlock branch. |
| MEDIUM — event-aware `workflow_dispatch` guard | **RESOLVED** | `github.event_name != 'workflow_run' \|\| github.event.workflow_run.conclusion == 'success'` is correct Actions semantics: under `workflow_dispatch`, `github.event.workflow_run` is null and the disjunct short-circuits true, so jobs execute; under `workflow_run` with conclusion≠success they skip. The behavioral criterion (observed dispatched run, not a green skip) is the right evidence bar. |
| MEDIUM — `resolve-tag` job | **RESOLVED** | Job outputs via `needs: resolve-tag` work identically under `workflow_run` and `workflow_dispatch` triggers; the shape/existence/peeled-commit validation plus exactly-one-tag fallback is sound. `TAG-EVIDENCE` mirrors the phase's `REL05-EVIDENCE` convention. |
| MEDIUM — semver predecessor selection | **RESOLVED** | The five-step algorithm (drop drafts → parse → strictly-less-than `$TAG` → max under stated prerelease policy → `PRIOR-RELEASE` log) correctly handles the stable-after-RC, historical-dispatch, and no-predecessor cases the criteria name. |
| LOW — stale `release: published` text in 01-05 Task 3 | **RESOLVED** | Task 3 `read_first` now says `workflow_run` and points at `TAG-EVIDENCE`. Verified in plan text. |
| LOW — 01-03 Task 1 prefer-then-build vs C6 | **RESOLVED** | The single prefer-then-build algorithm with the either-way `goreleaser --version` pin assertion removes the contradiction and closes the stale-local-binary hole in `TestGoreleaserPinParity`'s guarantee. |
| LOW — "THIRD job" scratch-job rewording | **RESOLVED** | The mutation now explicitly creates and reverts a scratch job, preserving the load-bearing "must produce a third holder" point, with a no-leftover criterion. |

## 3. Concerns

### Finding 1 — HIGH / ACTIONABLE — 01-06's oracle observes on-disk basenames, which cannot distinguish the broken config from the correct fix

**Evidence (pinned `goreleaser/v2@v2.17.1`):**

- `sign.go:182`: `env["artifact"] = art.Path` — at signature-path computation time, `${artifact}` is the artifact's **filesystem path**.
- `sign.go:195-197` + `tmplPath` (`sign.go:171-177`) + `relativeToDist` (`sign.go:160-168`): the on-disk signature file is written at `<art.Path>.sigstore.json`, kept verbatim when already under `dist/`.
- `archive.go:308-325`: for `formats: [binary]`, the archive pipe's `UploadableBinary` artifact has `Path: binary.Path` — **the same build-output path** (`dist/<build-id>_<goos>_<goarch>/codegraph`). No renamed copy ever exists on disk; the renamed name lives only in artifact metadata.
- `sign.go:275-276` (`env["artifact"] = art.Name` after signing) is what computes the artifact **record** name — the thing the release pipe publishes (`release.go:188-191` uploads by `artifact.Name`; `artifact.go:149` includes `Signature` in `ReleaseUploadableTypes`).

**Consequence:** under *both* the shipped `binary_signs:` config (broken: record names all `codegraph.sigstore.json`) *and* Candidate A (`signs: {artifacts: binary, ids: [raw]}`, correct: record names `codegraph_<tag>_<goos>_<goarch>.sigstore.json`), the **on-disk files are identical**: four files at `dist/<per-build-dir>/codegraph.sigstore.json`, basename `codegraph.sigstore.json` in every case. Plan 01-06 Task 1 asserts "their basenames are FOUR DISTINCT strings… each basename equals `<raw asset name>.sigstore.json`" against a filesystem listing. That oracle correctly goes RED on the shipped config — but goes RED **again** after the correct Candidate A remediation, at which point the plan's own instructions say "stop and report to the maintainer… the gate working correctly." The phase deadlocks on a false negative at the exact branch HIGH-2 exists to make safe. (The plan's "at most one survives" phrasing is also inaccurate — the four files coexist in per-build directories — but that is cosmetic.)

**Plan change needed (01-06 Task 1):** change the SIGN-EVIDENCE oracle from filesystem basenames to the artifact metadata: assert via `dist/artifacts.json` (the metadata pipe writes all artifacts including their `name`, `metadata.go:65`) that exactly four artifacts with `type == "Signature"` exist, that their `name` fields are four distinct strings, and that each equals `<raw asset stem>.sigstore.json` for one platform. This is the same `artifacts.json`-first idiom the phase already uses everywhere else, and it observes the quantity that actually collides at publish.

### Finding 2 — HIGH / ACTIONABLE — the identical `${artifact}` collision afflicts `sboms:`, and 01-02 statically pins the broken template with no rehearsal and no remediation path

**Evidence (pinned `goreleaser/v2@v2.17.1`):**

- `sbom.go:65-66`: `sboms.artifacts` defaults to `archive` — the plan's D-17 warning is correct.
- `sbom.go:124` + `artifact.go:734-750`: `artifacts: binary` filters `ByBinaryLikeArtifacts`, which dedupes by path *preferring `UploadableBinary`* — i.e. it catalogs the renamed raw-asset artifacts (same build paths).
- `sbom.go:268-283` (`applyTemplate`): for `documents:`, `${artifact}` expands against `env["artifact"]` = the artifact's **dist path** (`sbom.go:275-279`), so `"${artifact}.spdx.json"` yields `dist/<build-dir>/codegraph.spdx.json`.
- `sbom.go:250-253`: the SBOM artifact record name is `filepath.Base(path)` — **`codegraph.spdx.json` for all four platforms**. `SBOM` is in `ReleaseUploadableTypes` (`artifact.go:145`), so all four publish under one colliding name. With `replace_existing_artifacts: true` (01-02 Task 3), the clobbering is **silent** — the release succeeds with one `.spdx.json` asset instead of four, breaking D-17/DIST-03, and the first gate that can see it is 01-05's post-publish REQUIRED-set check — i.e. after the irreversible publish, under D-07 patch-forward. This is precisely the failure class 01-06 was created to catch for signatures, present in the neighboring pipe.

**Worse than untested:** 01-02 Task 2's `TestSbomsArePerBinaryWithSpdxNames` asserts `documents:` contains **exactly** `"${artifact}.spdx.json"` — it statically pins the colliding template and is demonstrated-RED against any correction, and no plan carries an on-RED branch for it.

**Plan change needed (01-02 Task 2 + 01-06 Task 1):**
1. Change the `sboms:` document template to derive from the artifact **name**, not path — e.g. `documents: ["{{ .ArtifactName }}.spdx.json"]` (the template fields map includes `ArtifactName`, `tmpl.go:89,212`; for the deduped `UploadableBinary` it is the renamed `codegraph_<tag>_<goos>_<goarch>`, which reproduces D-17's `<asset>.spdx.json` exactly). Verify against the pinned module at execution time as the phase's convention requires.
2. Update `TestSbomsArePerBinaryWithSpdxNames` to pin the corrected template.
3. Extend 01-06's `release:dry-run-signed` leg (which already runs the sbom pipe, since only `publish` is skipped) to assert four distinct SBOM artifact **names** in `dist/artifacts.json` alongside the four signature names — same oracle fix as Finding 1, same run, near-zero marginal cost.

### No other unresolved concerns

The remaining open questions the review prompt named all check out: Candidate A's enum is `binary` (not `archive` — `artifacts: archive` would select only the zips, `sign.go:111-112` → `UploadableArchive`); `resolve-tag` output wiring holds under `workflow_run`; the do-not-relitigate items were all respected. REL-05's spike, the JOINT criteria, the binding `gh attestation verify` decision, and the accepted publish-pipe residual are unchanged and unaffected by these findings.

## 4. Risk Assessment

**MEDIUM.** The phase's architecture, sequencing, gates, and threat model remain sound, and 25 of 27 prior findings are verifiably resolved. But the two open findings sit in the phase's highest-stakes spot: the published-asset naming contracts that `codegraph upgrade` and DIST-03 depend on, discoverable only at publish time if the plans execute as written. Finding 1 turns the phase's own safety rehearsal into a deadlock trap; Finding 2 is a silent, clobbered-asset supply-chain regression statically locked in by a demonstrated-RED test. Both fixes are small, local, and fully specified above (a jq-over-`artifacts.json` oracle and a `{{ .ArtifactName }}` template change), require no re-planning of scope, and can be applied as a targeted revision without another convergence cycle. With those two edits applied, the risk drops to LOW and the phase is ready to execute.

---

## Consensus Summary

Both reviewers had full repo access and both verified against the pinned
`goreleaser/v2@v2.17.1` module source rather than plan text alone. They agree that **all eight
cycle-2 fixes landed**, with one qualification: pi grades HIGH-2 PARTIALLY RESOLVED because the
remediation branch's *observation oracle* is keyed on the wrong signal.

The orchestrator independently re-derived every load-bearing claim below from the pinned module
source in the local module cache, and — for the mechanism questions — empirically against the
installed `goreleaser` binary with an instrumented `cosign` stand-in. Nothing here rests on
reviewer assertion.

**Net result: 25 of 27 prior findings are verifiably resolved. Two HIGH findings remain, both in
the same defect class — GoReleaser expands `${artifact}` from an artifact's PATH, while the
published release asset is named from its NAME, and for `formats: [binary]` outputs those two
differ.** One is a cycle-2 fix that is structurally right but measures the wrong quantity; the
other is a pre-existing defect in 01-02 that no prior cycle caught.

### Independently verified — the questions cycle 2 left open

| Question | Verdict | Evidence |
|---|---|---|
| Is `--key=` valid immediately after `sign-blob`? | **YES** | GoReleaser expands `${…}` placeholders and passes the args array to `exec.CommandContext` verbatim — no reordering, no validation (`internal/pipe/sign/sign.go:206-215`, `:251`). Recorded argv from a real run is index-for-index identical to the YAML. GoReleaser's own keyed-cosign example uses exactly this position. |
| Does `COSIGN_PASSWORD` actually reach cosign? | **YES** | `cmd.Env = env.Strings()` replaces the child environment, but `ctx.Env` is seeded from `ToEnv(append(os.Environ(), config.Env...))` (`pkg/context/context.go:141`), so an exported `COSIGN_PASSWORD` is inherited. 01-06's conditional "add it to the pipe environment if the pipe does not already inherit" resolves to "not needed". |
| Does `--key` perturb `${artifact}`/`${signature}` templating? | **NO** | Follows necessarily from the args being unparsed. The plan's flagged assumption is correct. |
| Does `goreleaser -f <config outside repo root>` re-root `dist/`? | **NO** | `dist` resolves against the process working directory, not the config's directory (`internal/pipe/dist/dist.go:23,31-32,41,61` — a plain relative default used via `os.Stat`/`os.MkdirAll`, no join against the config dir). Verified empirically: config at `…/gr-cfg/`, CWD `…/proj` → `…/proj/dist/` created, `…/gr-cfg/dist/` absent. `--clean` wipes only the configured dist. 01-06's post-run `dist/artifacts.json`-at-repo-root assertion is correct and its rationale is accurate. |
| Is candidate A (`signs:` over `archives[id=raw]`) valid for the pinned version? | **YES, with the enum `binary` — and its stated rationale is wrong** | See HIGH-A. |
| Does `resolve-tag`'s `needs:`-based output wiring hold under `workflow_run`? | **YES** | Job outputs behave identically under `workflow_run` and `workflow_dispatch`. `outputs: tag` is declared once (`01-05-PLAN.md:199`) and consumed as `needs.resolve-tag.outputs.tag` at all seven reference sites with no name drift. |

### HIGH-A — 01-06's sign oracle observes on-disk basenames, which cannot see the published name (ACTIONABLE)

Raised by pi as HIGH/ACTIONABLE; independently found and confirmed by Codex and the orchestrator,
though Codex graded it a non-actionable nuance.

**Mechanism, from pinned source.** `${artifact}` expands to `art.Path` at signature-path
computation time (`internal/pipe/sign/sign.go:179-182`, with the telling
`env["artifactName"] = art.Name // shouldn't be used`). **After** signing, GoReleaser re-expands the
same template with `env["artifact"] = art.Name` to compute the *published* artifact record
(`sign.go:274-276`); the release pipe uploads by `artifact.Name`, and `Signature` is in
`ReleaseUploadableTypes` (`internal/artifact/artifact.go:145`). For a `formats: [binary]` archive
the rename lives only in `.name` — `.path` still points at the pre-rename build output
(`internal/pipe/archive/archive.go:308-310`: `Name: finalName, Path: binary.Path`). No renamed copy
is ever written to disk.

**Consequence.** Under both the shipped `binary_signs:` config (broken) and candidate A (correct),
the on-disk files are byte-identical in naming: four files at
`dist/<per-build-dir>/codegraph.sigstore.json`. 01-06 Task 1 asserts on a filesystem listing —
"exactly FOUR files matching `*.sigstore.json`… their basenames are FOUR DISTINCT strings"
(`01-06-PLAN.md:243-246`), reused verbatim as validation step 1 (`:328`). That oracle correctly goes
RED on the shipped config, but goes RED **again** on a correctly-applied candidate A.

**Two corrections to the reviewers.** (1) pi concludes this deadlocks the phase. It does not:
candidate B derives `signature:` from tag/OS/arch rather than from `${artifact}`, so
`relativeToDist` (`sign.go:160-168`) places four distinctly-named sidecars at the dist root and the
re-templated published name is equally correct — **B passes the oracle and is genuinely correct.**
The plan authorizes B on exactly this trigger ("ONLY if A is rejected by … the pinned module's
actual behavior", `01-06-PLAN.md:319-320`), so the branch fails closed onto a working config.
(2) The plan's worry that B "may fix the on-disk filename and still publish a colliding asset name"
(`:321-324`) is over-pessimistic in B's favour, not a live risk. (3) The plan's "at most one
survives" phrasing (`:245`) is inaccurate — the four files coexist in per-build directories.

**Why it is still actionable.** The oracle never observes the quantity that actually collides at
publish. It happens to give the right verdict for the two reachable configs, but it false-REDs the
PREFERRED candidate, and it would commit into `.goreleaser.yaml` a rationale comment asserting
something untrue about `${artifact}` (`01-06-PLAN.md:302-316` instructs recording that reasoning).
Decisively: **this same oracle blindness is what lets HIGH-B through undetected.**

**Plan change needed (01-06 Task 1).** Derive `SIGN-EVIDENCE` from `dist/artifacts.json` — which
the plan already asserts lands at the repo root — rather than from a filesystem glob: assert exactly
four artifacts with `type == "Signature"`, four distinct `name` values, each equal to
`<raw asset stem>.sigstore.json` for one platform. Correct candidate A's rationale to name
`artifacts: binary` (not `archive`) and to stop claiming `${artifact}` yields the renamed asset.

Two supporting traps worth recording in the plan: `signs.artifacts: archive` filters
`UploadableArchive` (`sign.go:111`) and so matches **nothing** for a `formats: [binary]` archive —
and a zero match is `log.Warn("no artifacts matching the given filters found")` + `return nil`, not
an error. `goreleaser check` does **not** validate this enum (a bogus value returns "1 configuration
file(s) validated"), so validation step 3 is vacuous for this class of error. Note also that the
literal `binary` means `UploadableBinary` in a `signs:` block but `artifact.Binary` in a
`binary_signs:` block.

### HIGH-B — the identical collision afflicts `sboms:`, and 01-02 statically pins the broken template (ACTIONABLE, NEW)

Raised by pi. **Independently confirmed by the orchestrator against pinned source and against
today's pipeline.** This is a new finding — no prior cycle caught it.

**Mechanism.** In `documents:`, `${artifact}` expands to the artifact's dist-relative **path**
(`internal/pipe/sbom/sbom.go:268-283`, via `subprocessDistPath`). The SBOM artifact record is then
named `filepath.Base(match)` (`sbom.go:250-253`). `artifacts: binary` filters
`ByBinaryLikeArtifacts` (`sbom.go:124`), which dedupes by path preferring `UploadableBinary` — whose
`Path` is still the pre-rename build output. So `documents: ["${artifact}.spdx.json"]` yields
`dist/<build-dir>/codegraph.spdx.json` and a record name of **`codegraph.spdx.json` for all four
platforms**. `SBOM` is in `ReleaseUploadableTypes` (`artifact.go:145`), so all four publish under
one name.

**Corroboration from GoReleaser itself.** Upstream's own default for `artifacts: binary` is
`documents: ["{{ .Binary }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}.sbom.json"]` (`sbom.go:70-71`) — a
name-shaped, per-platform template, chosen precisely because the path-derived one collides.

**This is a regression, not a pre-existing quirk.** Today's pipeline iterates the already-renamed
assets and runs `syft "$f" -o spdx-json="${f}.spdx.json"` (`.github/workflows/release.yml:275-277`),
producing four distinct `codegraph_<tag>_<goos>_<goarch>.spdx.json` assets. The planned block would
publish one. With `replace_existing_artifacts: true` (01-02 Task 3) the clobbering is **silent** —
the release succeeds with one `.spdx.json` instead of four, breaking D-17/DIST-03 and REL-09's
name-shape invariant. The first gate that could see it is 01-05's post-publish REQUIRED-set check,
i.e. after the irreversible publish, under D-07 patch-forward.

**Worse than untested.** 01-02's plan text states the correct *intent* — "preserves today's
`<asset>.spdx.json` names" (`01-02-PLAN.md:25`, `:59`, `:350`) — but pins the wrong *mechanism* at
`:35`, `:348`, `:375`, and `TestSbomsArePerBinaryWithSpdxNames` asserts `documents:` contains
**exactly** `${artifact}.spdx.json` (`:321-324`), demonstrated RED against any correction. The test
would actively resist the fix. There is no rehearsal for the SBOM pipe and no on-RED branch anywhere
in the phase.

**Plan change needed.** (1) In 01-02 Task 2, change the template to derive from the artifact name
rather than its path — pi proposes `documents: ["{{ .ArtifactName }}.spdx.json"]`, to be confirmed
against the pinned module at execution time as this phase's convention requires. (2) Update
`TestSbomsArePerBinaryWithSpdxNames` to pin the corrected template. (3) Extend 01-06's
`release:dry-run-signed` leg — which already runs the SBOM pipe, since only `publish` is skipped —
to assert four distinct SBOM artifact **names** in `dist/artifacts.json` alongside the four
signature names. Same oracle fix as HIGH-A, same run, near-zero marginal cost.

### Agreed Strengths

- **HIGH-1 is fully closed**, verified on all three of its load-bearing assumptions (arg position,
  password inheritance, dist location). All three reviewers and the orchestrator's source check
  agree. The run-scoped `mktemp -d`, trap cleanup, additions-only diff guard, and
  `-f <copy>` invocation are sound.
- **HIGH-2's branch structure is right** — classification, candidate ordering, mandatory re-run,
  mandatory static-test re-pointing with a demonstrated RED, and record-observation-before-amending
  ordering are all present and coherent. Only its oracle needs correcting.
- **Candidate A is mechanically valid** for v2.17.1 with `artifacts: binary` + `ids: [raw]`; `ids:`
  filtering is real and applied (`sign.go:124`).
- **The dispatch guard is behaviorally gated**, not merely textually present: the criterion requires
  an observed dispatched run that actually EXECUTED, closing the "a skip is also green" hole
  (`01-05-PLAN.md:354-356`). Guard coverage is enforced by count with a negative criterion banning
  the bare form.
- **`resolve-tag` is sound** in wiring, shape/existence/peeled-commit validation, and its
  exactly-one-tag fallback; `TAG-EVIDENCE` mirrors the phase's evidence-line convention.
- Prior-release selection is fully specified with six behavioral criteria.

### Agreed Concerns

Both reviewers independently traced the sign pipe to the same `art.Path` vs `art.Name` split. They
diverge only on grading (below). No other concern was raised by both.

### Divergent Views

**HIGH-A grading.** pi: HIGH/ACTIONABLE, and claims the phase deadlocks. Codex: real, but
non-actionable — "the plan's mandatory filesystem validation will catch this and route to Candidate
B, so it cannot silently reach publication." The orchestrator's source trace resolves the
disagreement in favour of pi on *actionability* and in favour of Codex on *safety*: the branch does
fail closed onto a correct candidate B, so there is no deadlock — but the oracle still cannot
observe the published name, and that same blindness is exactly what lets HIGH-B reach production
undetected. Both should be fixed by the one `artifacts.json` change.

Codex did not examine the `sboms:` pipe and so did not surface HIGH-B; pi did. This is the cycle's
clearest demonstration of adversarial-review value.

### Residual observations (recorded, NOT actionable)

1. **`resolve-tag` names no ref-resolution mechanism.** It specifies "verify `refs/tags/$TAG`
   exists", "the tag's PEELED commit", and "querying the tag refs pointing at `$HEAD_SHA`"
   (`01-05-PLAN.md:206-210`) — git vocabulary — but declares no `actions/checkout` and no
   `fetch-depth`/`git fetch --tags`; checkout appears only for `verify-supply-chain` (`:238`) and
   `self-upgrade` (`:293`). Graded non-actionable because: the step's `env:` carries `REPO` and
   `GH_TOKEN` with no checkout, which points unambiguously at `gh api`; the repo's release tags are
   **lightweight** (`git cat-file -t v0.4.0` → `commit`), so there is no annotated-tag dereferencing
   fork; the job fails closed and blocks both downstream jobs via `needs:`; and the behavioral
   dispatch criterion exercises it. Both reviewers graded this fix RESOLVED. Worth one clarifying
   sentence if 01-05 is touched anyway.
2. **The copy-generation tool for `.goreleaser.signtest.yaml` is unnamed** (`01-06-PLAN.md:214-220`
   gives the semantic target but no `sed`/`yq`/heredoc). A `yq` round-trip would likely reflow the
   file and trip the additions-only guard — which is the guard working as intended. The
   additions-only requirement is correctly worded to reject deletions and modifications, not only
   non-matching additions.
3. **The publish pipe remains unexercisable.** Re-confirmed: `--skip=publish` structurally cannot
   reach it and no rehearsal exists that would exercise it without publishing. Correctly carried to
   the one-way checkpoint. (Note that HIGH-B is *not* an instance of this: it is observable in the
   snapshot rehearsal once the oracle reads `artifacts.json`.)
4. `attestations: read` is granted workflow-wide though only one job consumes it — least-privilege
   polish.

### Not Re-Litigated

All ten adjudicated items were supplied as out of scope and neither reviewer reopened any: the
arm64 runner provisioning, REL-07 wave-2 separation, the `gh attestation verify` attestor (D-10),
the deliberately-stale `REQUIREMENTS.md` REL-08 entry owned by 01-04, the ratified `zig cc`
byte/glibc change, `releaseAssetName()` at `internal/upgrade/upgrade.go:209`, the 7 spec-less probe
rows as flagged assumptions, 01-04's removed checkpoint and its expected `verify.plan-structure`
warning, the unexercisable publish pipe, and `files_modified` not being an execution-time allowlist.

The orchestrator additionally checked the one cross-plan interaction that could have been a real
contradiction — whether candidate A would invert 01-02's `rg -n '^signs:' .goreleaser.yaml`
returns-nothing criterion (`01-02-PLAN.md:376`). It does not: that is a plan-scoped acceptance
criterion evaluated at 01-02's wave-2 execution, not a durable gate, and the one durable Go test
(`TestBinarySignsSidecarMatchesUpgradeContract`) is explicitly re-pointed by 01-06 validation step 2.
Wave-3 file sets were re-confirmed disjoint: 01-04 touches `release.yml` +
`release_workflow_shape_test.go`; 01-06 touches `Taskfile.yml` + `linux-cross-canary.yml`, with
`.goreleaser.yaml` + `goreleaser_shape_test.go` only on its RED branch.

### Recommendation

Both remaining findings are small, local, and fully specified: one changes an oracle from a
filesystem glob to a `dist/artifacts.json` read; the other changes one template string plus the test
that pins it. Neither requires re-planning scope, and the second fix rides along in the first's
rehearsal run. Applying them as a targeted revision — rather than opening a fourth convergence
cycle — is the proportionate response.
