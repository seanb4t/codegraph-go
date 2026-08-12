---
phase: 1
slug: cross-compile-spike-goreleaser-release-migration
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-11
---

# Phase 1 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Register origin:** `register_authored_at_plan_time: true` — all six PLAN files carry a
parseable `<threat_model>` block. 45 unique threat IDs (`T-01-01`…`T-01-44` plus `T-01-SC`,
which recurs once per plan): **31 high, 12 medium, 2 low**; 42 `mitigate`, 2 `accept`,
1 `mitigate + accept (residual)`. Config: `asvs_level: 1`, `block_on: high`.

## The ASVS-L1 short-circuit was refused — read this before trusting the register

With `register_authored_at_plan_time: true`, a preliminary `threats_open: 0`, and
`asvs_level == 1`, the workflow permits skipping the auditor entirely and writing this file
from the plan-time register alone. **That was refused here, deliberately**, for the reason
recorded in memory `xkbc8m36hm` and applied again in Phase 4 (`04-SECURITY.md`): the same
short-circuit would have skipped Phase 3's audit, which found six unregistered flags, two of
them live defects. Phase 1 carries **31 high rows**, the repository's entire supply-chain
trust anchor, and a signing-identity boundary — well past the recorded "spawn anyway" bar.

**Who ran this and how.** Verified inline by the orchestrator on 2026-08-11, not by a
subagent, following the Phase 4 precedent. Every row below cites a command that was executed
against the tree at HEAD `42427c5` or a recorded CI run that was re-read, never a plan-time
assertion carried forward. This distinction is written here so a future reader cannot mistake
inherited depth for verified depth.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| GitHub workflow context → runner shell | `github.ref_name` and other context are attacker-influenceable (a crafted tag or branch name) and cross into `run:` script text | Untrusted strings into shell |
| Third-party Action → release-path runner | Any Action executes with the runner's filesystem and network access on the same profile family the real release job uses | Arbitrary code |
| macOS build host → Linux exec host | Cross-compiled artifacts move between jobs as CI artifacts; the exec jobs run whatever `cross-build` uploaded | Executable binaries |
| `.goreleaser.yaml` config → cosign signing identity | The sign block determines WHICH artifacts are signed and under WHAT sidecar name; drift changes what a downstream verifier checks | Signing scope |
| release config → `codegraph upgrade` runtime | Two independent implementations of one asset-name string (`archives[raw].name_template` and `releaseAssetName()`) must agree | Asset-name contract |
| published asset set → external consumers | The checksums file's coverage set is a contract external scripts and the Phase-3 cask depend on | Integrity claims |
| GitHub OIDC issuer → cosign certificate SAN | The job holding `id-token: write` is the only thing that can mint a certificate `internal/upgrade` accepts | Signing authority |
| `release.yml` file identity → published verification instructions | `releaseWorkflowRefPattern` and `docs/RELEASE.md`'s `--certificate-identity-regexp` both hard-code this file's path and ref shape | Verification anchor |
| GitHub Attestations API → external verifier | The attestation format determines which verifier can prove anything; a documented command that cannot verify the published format is a repudiation hole | Provenance |
| published documentation → a user's shell | SECURITY.md and README.md instructions are copy-pasted by people who are, by definition, trying to check whether they can trust us | Executable instructions |
| published GitHub release → any downloader | The boundary the entire phase exists to keep trustworthy; every claim must be provable from the outside against re-downloaded bytes | Release artifacts |
| prior shipped binary → new release | `codegraph upgrade` fetches, verifies in-process, and REPLACES its own executable; a bad asset or sidecar name here is RCE on every upgrading client | Executable replacement |
| verification workflow → repository | A verification job that can write is a verification job that can be turned into a publishing job | Write capability |
| throwaway signing key → production signing identity | A rehearsal key must never be mistakable for, or capable of producing, a certificate the production SAN policy accepts | Signing identity |
| canary job (PR-triggerable) → release trust boundary | This job runs on pull requests, so it must be incapable of minting anything the release path trusts | OIDC / trust |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-01-01 | Tampering | canary `run:` steps | high | mitigate | G1: 5 run bodies in `linux-cross-canary.yml`, **0** contain `${{ }}` | closed |
| T-01-02 | Tampering | canary third-party Actions | high | mitigate | G2: 111 `uses:` across 13 workflows — 91 SHA-pinned + 20 local, remainder **0** | closed |
| T-01-03 | Spoofing | cosign SAN anchor | high | mitigate | G3: `release.yml` path intact; trigger still `{"push":{"tags":["v[0-9]*"]}}` | closed |
| T-01-04 | Elevation of Privilege | canary permissions | medium | mitigate | File-scope `permissions: {contents: read}`; no job declares `id-token` (G5) | closed |
| T-01-05 | Repudiation | REL-05 evidence | medium | mitigate | Canary run 31273571889 re-read: 0 non-success jobs, two resolved `REL05-EVIDENCE` lines | closed |
| T-01-06 | Tampering | sign signature template | high | mitigate | `signature` is field-templated (`.ProjectName/.Tag/.Os/.Arch`), never `${artifact}`; live `SIGN-EVIDENCE count=4 distinct=4` | closed |
| T-01-07 | Spoofing | raw asset name shape | high | mitigate | `archives[raw].name_template` unchanged; `TestReleaseAssetNameMatchesGoReleaser` green | closed |
| T-01-08 | Repudiation | checksum coverage set | medium | mitigate | `checksum.ids: [raw, zip]` explicit; v0.5.1 checksums file = 8 lines, 4 raw + 4 zip | closed |
| T-01-09 | Tampering | SBOM artifact scope | medium | mitigate | `sboms[0].artifacts: binary` set explicitly, not left to the `archive` default | closed |
| T-01-10 | Elevation of Privilege | circular signing | medium | mitigate | `signs.artifacts: binary` and `checksum.ids: [raw, zip]` both exclude sidecars; 0 sidecars in the published checksums file | closed |
| T-01-11 | Elevation of Privilege | `id-token: write` job scope | high | mitigate | G5: 34 jobs across 13 workflow files, **exactly one** holds `id-token: write` | closed |
| T-01-12 | Spoofing | cosign SAN anchor | high | mitigate | G3 | closed |
| T-01-13 | Tampering | second checksums writer | high | mitigate | `rg -c sha256sum .github/workflows/release.yml` → **0**; declarative `checksum:` live | closed |
| T-01-14 | Tampering | `${{ }}` injection in the release job | high | mitigate | G1: 12 run bodies across the 3 workflows, **0** interpolations. See Finding F-1 — the literal "exactly one run body" has drifted to two | closed |
| T-01-15 | Tampering | third-party Action supply chain | high | mitigate | G2 | closed |
| T-01-16 | Repudiation | darwin cross-link regression | medium | mitigate | `TestDarwinLegsBuildNatively` present at `release_workflow_shape_test.go:582`, green | closed |
| T-01-17 | Repudiation | attestation-format substitution | high | mitigate | All 5 published docs name `gh attestation verify`; the 2 residual `slsa-verifier` mentions are inside labelled pre-migration notes | closed |
| T-01-18 | Tampering | `attest-build-provenance` supply chain | high | mitigate | Pinned `@4d101475d8b20a2381f78447822ac1eab6504dd8` (v4.2.2) with trailing version comment | closed |
| T-01-19 | Elevation of Privilege | `id-token: write` after provenance removal | high | mitigate | G5; `jobs.provenance` absent from `release.yml` | closed |
| T-01-20 | Spoofing | cosign SAN anchor | high | mitigate | G3; `releaseWorkflowRefPattern` intact in `internal/upgrade/verify.go` | closed |
| T-01-21 | Information Disclosure | private-repository provenance opt-in | low | accept | Risk **removed**, not merely accepted: `provenance:` job absent and `private-repository` occurs 0 times in any workflow. See AR-01 | closed |
| T-01-22 | Spoofing | cosign SAN after job collapse | high | mitigate | `verify-supply-chain` job green in run 31285981504 against re-downloaded v0.5.1 assets | closed |
| T-01-23 | Tampering | `codegraph upgrade` swapping a wrong binary | high | mitigate | `SELF-UPGRADE-EVIDENCE` asserts sha256 byte identity; both legs green in run 31285981504 | closed |
| T-01-24 | Repudiation | verifying against a local `dist/` copy | high | mitigate | All checks `gh release download` from the published release; 17 published assets re-enumerated live | closed |
| T-01-25 | Elevation of Privilege | verification workflow token scope | high | mitigate | `post-release-verify.yml` file-scope `{contents: read, attestations: read}` — no `write` of any kind | closed |
| T-01-26 | Tampering | destruction of Phase 2's RED baseline | high | mitigate | `docs/RELEASE-PROCEDURES.md`: 3 patch-forward/never-delete statements, 3 `v0.5.1`/`SIGN-03` baseline references | closed |
| T-01-27 | Tampering | `${{ }}` injection in post-release-verify | high | mitigate | G1: 5 run bodies, **0** interpolations | closed |
| T-01-28 | Spoofing | version forcing bypassing release-please | medium | mitigate | 0 `Release-As`/`release-as` occurrences; `TestGsdTagCreationIsDisabled` and `TestReleasePleaseStaysPreMajor` present | closed |
| T-01-29 | Elevation of Privilege | `id-token: write` INTRA-JOB surface | high | mitigate + accept (residual) | G5 + G1: the single holder's 11 steps carry 2 run bodies, both `task` invocations, 0 interpolations. Residual recorded as AR-02 | closed |
| T-01-30 | Tampering | overwriting the release-please body | medium | mitigate | `.release` = `{replace_existing_artifacts: true, prerelease: auto}` only — no `name_template`/`header`/`footer`/`draft`/`disable` | closed |
| T-01-31 | Repudiation | a gate that structurally cannot pass | high | mitigate | Trigger is `[workflow_dispatch, workflow_run]`; `on.release` absent | closed |
| T-01-32 | Denial of Service | unauthenticated `gh` in the verifier | high | mitigate | `GH_TOKEN` set 6× in `post-release-verify.yml`; 9 `GH_TOKEN` references in `Taskfile.yml` incl. named preconditions | closed |
| T-01-33 | Repudiation | brittle fixed-total asset assertion | medium | mitigate | The 197-line `verify:release-assets` block (`Taskfile.yml:2362-2558`) contains **0** literal `17` | closed |
| T-01-34 | Spoofing | throwaway key as production identity | high | mitigate | Key generated into a temp dir and trapped; the job declares no `id-token` (G5 — only `release.yml` does) | closed |
| T-01-35 | Tampering | colliding `.sigstore.json` names | high | mitigate | Live `SIGN-EVIDENCE count=4 distinct=4` with per-platform names | closed |
| T-01-36 | Tampering | `${{ }}` injection in `sign-snapshot` | high | mitigate | G1 | closed |
| T-01-37 | Repudiation | a green exit code recorded as evidence | medium | mitigate | Pass condition is the `count=N distinct=N` assertion line, not the exit code | closed |
| T-01-38 | Repudiation | a gate that silently SKIPS | high | mitigate | 5 jobs / **5** event-aware guards (`event_name != 'workflow_run' \|\| conclusion == 'success'`) — 1:1, see Finding F-2 | closed |
| T-01-39 | Spoofing | wrong tag under verification | medium | mitigate | Every executable `head_branch` reference is inside `resolve-tag` (lines 113–152); 58–65 are header prose | closed |
| T-01-40 | Repudiation | wrong self-upgrade predecessor | medium | mitigate | `PRIOR-RELEASE` semver-predecessor resolution present; explicit `upgrade "$TAG"`, never bare | closed |
| T-01-41 | Tampering | colliding published SBOM names | high | mitigate | `sboms.documents` is `{{ .ArtifactName }}.spdx.json` (NAME-derived); live `SBOM-EVIDENCE count=4 distinct=4` | closed |
| T-01-42 | Tampering | colliding `.spdx.json` names | high | mitigate | Same evidence as T-01-41 | closed |
| T-01-43 | Repudiation | an oracle blind to the colliding quantity | high | mitigate | The oracle reads published names from `dist/artifacts.json`, not filesystem basenames; re-run live this session | closed |
| T-01-44 | Denial of Service | rerun against a partially-published release | high | mitigate | `release.replace_existing_artifacts: true` pinned explicitly (GoReleaser defaults it false) | closed |
| T-01-SC | Tampering | npm/pip/cargo installs | low | accept | 0 package-manager installs across all six Phase 1 plans' files. See AR-03 and Finding F-3 | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `block_on` count toward `threats_open`*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01 | T-01-21 | Private-repository provenance opt-in. The `provenance:` job that carried `private-repository: true` was deleted whole, so the opt-in no longer exists to accept — verified as 0 occurrences across all 13 workflows. Recorded as accepted per the plan's disposition; in practice the risk was removed rather than tolerated. | maintainer (plan time) | 2026-08-08 |
| AR-02 | T-01-29 | Residual intra-job execution surface. One job legitimately needs `id-token: write`, so everything in that job runs inside the OIDC-capable context. Bounded rather than eliminated: 11 steps, 2 `run:` bodies, both `task` invocations, 0 `${{ }}` interpolations, all Actions SHA-pinned. Accepted as the irreducible cost of a single-job release. | maintainer (plan time) | 2026-08-08 |
| AR-03 | T-01-SC | Supply-chain package installs. All six Phase 1 plans add zero package-manager installs and zero new Go module dependencies; `cosign`/`syft` were already present. | maintainer (plan time) | 2026-08-08 |

---

## Findings from this audit (non-blocking)

**F-1 — T-01-14's literal assertion has drifted; its property has not.** The row states the
`release` job carries "exactly one `run:` body (`task release:goreleaser`)". It now carries
two: `task release:goreleaser` and `task release:record-final-hashes`, the latter added by
Phase 2 for SIGN-04 hash recording. Both are `task` invocations and neither interpolates
`${{ }}`, so the injection property the threat exists to protect is intact. Recorded because a
future audit re-reading the plan-time text would find a mismatch and should not treat it as a
regression.

**F-2 — T-01-38's guard coverage strengthened after Phase 1.** The plan recorded "exactly 3
occurrences, matching the 3 jobs". `post-release-verify.yml` now has 5 jobs (Phase 2 added
`gatekeeper` and `notarized-suite`) and **5** guards. The 1:1 invariant held across the
addition, which is the property that matters — a count-based assertion would have gone red on a
correct change.

**F-3 — one package-manager install exists in the repo, outside Phase 1's scope.**
`.github/workflows/bench.yml:122` runs `npm install -g @colbymchenry/codegraph@1.3.1` to install
the TypeScript CodeGraph for head-to-head benchmarking. Added by v1.0 phase 08-08 (`3c44a83`),
not by any Phase 1 plan, so `T-01-SC`'s scoped claim holds. Recorded so a whole-repo sweep does
not read it as a Phase 1 contradiction. It is a pinned version of the upstream project this
port replaces; no action proposed here.

**No unregistered threats found.** Unlike Phase 3 (6 unregistered flags) and Phase 4 (3), this
audit surfaced none. Note the contributing hygiene gap: **none of the six SUMMARY files carries
a `## Threat Flags` section**, so there were no executor-reported flags to reconcile — absence
of findings here is partly absence of reporting, not proof of absence. Same recommendation as
Phase 4: the executor prompt should require the section.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-11 | 45 | 45 | 0 | orchestrator (inline, ASVS-L1 short-circuit refused) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer) — 42 mitigate, 2 accept, 1 mitigate+accept
- [x] Accepted risks documented in Accepted Risks Log — AR-01, AR-02, AR-03
- [x] `threats_open: 0` confirmed — 45/45 closed, 0 open at or above `high`
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-11
