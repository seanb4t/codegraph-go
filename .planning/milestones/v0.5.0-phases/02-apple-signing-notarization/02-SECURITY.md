---
phase: 02
slug: apple-signing-notarization
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-09
---

# Phase 02 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Built retroactively from the seven `<threat_model>` blocks authored at plan time
in `02-01-PLAN.md` … `02-07-PLAN.md` (`register_authored_at_plan_time: true`), then
verified against the implementation by `gsd-security-auditor` at ASVS L1 with
`block_on: high`. The auditor verified mitigations only — it did not scan for new
threats — and re-ran every named mitigation test uncached.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| GitHub Releases CDN → local temp dir | An untrusted, network-fetched binary crosses into the local filesystem and is assessed (`verify:gatekeeper`) or executed (`verify:notarized-suite`) | Published release binaries + their cosign `.sigstore.json` bundles |
| Local test rig → Gatekeeper policy engine | A hand-written `com.apple.quarantine` xattr crosses into `syspolicyd`'s input, standing in for a real browser download | Synthetic quarantine attribute |
| Maintainer shell environment → quill → Apple's notary service | A Developer ID private key and an App Store Connect API key cross from the local environment into an in-process signer and a network submission | P12 private key, ASC API key, team/key/issuer identifiers |
| GitHub repository secrets → the release job's process environment | Five Apple credentials become readable by every step and action inside that job | Same five credential values |
| Pull-request-triggerable workflows → the same secret namespace | A fork-controllable trigger must never reach these values or the OIDC write scope | Credential names, `id-token: write` scope |
| GoReleaser pipe order → published artifact identity | What cosign signs and what SLSA attests must equal what a user downloads | Pre-sign vs post-notarize Mach-O bytes |
| The release run → the permanently-published artifact set | The irreversible boundary; nothing published here can be withdrawn, only superseded | Published release assets, tags |
| Recorded evidence → the phase's completion claim | If evidence describes anything other than the published bytes, the claim is false regardless of how green the jobs were | sha256 hashes, Gatekeeper verdicts, provenance labels |
| Published documentation → a user's verification decision | A reader substitutes the document's command for their own judgement | Documented reproduction sequence, guarantee scope |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-01 | Spoofing | `verify:gatekeeper` synthetic quarantine rig | high | mitigate | A2 verdict **CONFIRMED** — synthetic vs real Safari download on a byte-identical file (same sha256 `69325c30…`), both `rejected`/exit 3, scope limit stated. `02-EVIDENCE.md:224-254` | closed |
| T-02-02 | Repudiation | The gate itself (silent-pass and never-pass surfaces) | critical | mitigate | `xattr -p` read-back hard-fails **before** assessment (`Taskfile.yml:2266-2273` → `:2283`); exit outside {0,3} fatal via `*)` → `OK=0` (`:2300-2308`); verdict/source off-diagonal fatal (`:2328-2337`); D-19 `spctl -a -vv -t install` (`:2283`); `syspolicy_check` demoted NON-GATING (`:2339-2356`). Both cannot-fire and cannot-pass proofs recorded (`02-EVIDENCE.md:109`, `:123`) | closed |
| T-02-03 | Tampering | Published v0.5.1 darwin assets (RED baseline) | medium | mitigate | Read-only `gh release download` into `mktemp -d` + trap (`Taskfile.yml:2195-2197`); bytes pinned to GitHub's recorded per-asset digest, mismatch fatal, missing-digest asymmetry explicit (`:2200-2234`); no `gh release upload/delete/edit` anywhere in `:2116-2375` | closed |
| T-02-04 | Information Disclosure | `GH_TOKEN` in the target's environment | low | accept | Read-scoped `github.token`-class credential on existing `verify:release-assets` / `verify:self-upgrade` precedent; never echoed. Named precondition `Taskfile.yml:2143-2144`. See AR-01 | closed |
| T-02-05 | Tampering | `notarize.macos[0].ids` | high | mitigate | `TestNotarizeMacosIdsCoverDarwinBuildIDs` asserts the **exact** set via `sortedJoin` — fails on absent, empty, or superset (`goreleaser_shape_test.go:741-755`); parser non-vacuity companion `:716`; config `.goreleaser.yaml:253-260` | closed |
| T-02-06 | Tampering | `notarize.macos[0].enabled` | high | mitigate | Conjunctive five-term `and (isEnvSet …)` gate (`.goreleaser.yaml:252`); `TestNotarizeMacosEnabledIsEnvGated` resolves the template under **seven** environments → 1 TRUE / 6 FALSE including each single-credential-missing case (`goreleaser_shape_test.go:846-889`) | closed |
| T-02-07 | Repudiation | Retracted `signs:`-rejection rationale in `.goreleaser.yaml` | medium | mitigate | Claim deleted outright — `rg -c 'no longer cleanly apply' .goreleaser.yaml` → **0**, `^binary_signs:` → **0**, `^signs:` → **1**, `D-18` citation → 6. Replacement rationale `.goreleaser.yaml:29-37`. **Scope note:** the same claim survives outside this component — see UF-1 | closed |
| T-02-08 | Tampering | `release:dry-run-signed` awk key-injection anchor | medium | mitigate | **NOT IMPLEMENTED.** The registered acceptance criterion (indentation vs `HEAD`) exists only as plan text (`02-02-PLAN.md:294`) with no execution record and no durable control. Auditor additionally found the existing additions-only diff guard (`Taskfile.yml:586-605`) **passes vacuously** when the anchor stops matching. Anchor currently matches 1:1 against worktree and `HEAD`, so present state is benign. Todo filed | open — below `high` threshold (non-blocking) |
| T-02-09 | Repudiation | `CODEGRAPH_TEST_BIN` resolver fallback | high | mitigate | Two-outcome contract, no input yields `useEnv=false, nil` (`test/integration/main_test.go:42-95`); `TestMain` aborts before temp dir or build (`:116-125`); table test pins missing/dir/non-exec plus third-outcome invariants (`binpath_test.go:16-120`); negative e2e recorded `02-03-SUMMARY.md:67` | closed |
| T-02-10 | Tampering | The externally supplied binary | low | accept | CI value comes from `gh release download` against this repo's own release, already covered by sibling cosign/attestation verification; locally it is the developer's own path. A second integrity implementation would drift from `internal/upgrade/verify.go`. See AR-02 | closed |
| T-02-11 | Information Disclosure | Cert / API-key material in error output (assumption A5) | high | mitigate | **A5 RESOLVED, no leak observed** — deliberate wrong-`MACOS_SIGN_PASSWORD` run; zero PEM headers, zero 200+ char base64 runs, zero occurrences of the supplied password; failed in 1s before Apple network contact. No masking requirement carried into 02-06. `02-EVIDENCE.md:552-575`, cross-ref `.github/workflows/release.yml:181-190` | closed |
| T-02-12 | Denial of Service | The rehearsal's Apple round-trip | medium | mitigate | One named precondition per credential variable (`Taskfile.yml:1029-1038`); shape test pins the set + non-vacuity companion (`taskfile_shape_test.go:1813-1847`, `:1849`); measured halt **0.036s** (`02-04-SUMMARY.md:14,95`) | closed |
| T-02-13 | Tampering | Committed `.goreleaser.yaml` during the mutation experiment | medium | mitigate | Clean-worktree precondition at open (`Taskfile.yml:1021`); pre-hash `:1058`; temp tree under a single trap `:1065-1070`; byte-unchanged assertion at close `:1566-1575`; clean after all four runs including the mutation (`02-EVIDENCE.md:577-583`) | closed |
| T-02-14 | Spoofing | The documented reproduction sequence | high | mitigate | `xattr -p` read-back is step 3 and **precedes** the `spctl` line (`docs/RELEASE.md:251-256` → `:260`); expected pass output quoted `:268-274`; the does-not-count list has **six** items, a superset of the declared four (`:296-322`) | closed |
| T-02-15 | Repudiation | The guarantee statement's scope | medium | mitigate | "not stapled" sits inside the guarantee sentence (`docs/RELEASE.md:179-181`), expansion `:190-191`, offline limitation immediately adjacent `:193-199`; no stronger claim found anywhere in the file | closed |
| T-02-16 | Information Disclosure | Apple credentials reachable from a pull-request trigger | critical | mitigate | `rg -l 'MACOS_' .github/` → **release.yml only**; one step, step-level `env:` (`release.yml:191-200`); no `environment:` key; exactly one job (`:85`). `TestAppleSecretsScopedToSingleReleaseJob` enumerates the workflow dir at runtime and rejects non-release files, non-step scope, and any PR/`pull_request_target` workflow holding the names or `id-token: write` (`release_workflow_shape_test.go:1206-1284`); non-vacuity companion `:1289` | closed |
| T-02-17 | Elevation of Privilege | The OIDC write scope | high | mitigate | Single `id-token: write` grant (`release.yml:88-90`; the other two hits are comments); `TestOIDCWriteScopedToSingleGoreleaserJob:617`, `TestReleaseWorkflowTriggerIsTagPushOnly:917`, `TestReleaseWorkflowFileMatchesPattern:870`; independently re-asserted `release_workflow_shape_test.go:1274-1283` | closed |
| T-02-18 | Repudiation | The conclusion guard on the two new post-release jobs | high | mitigate | **2 of 3 legs present.** Verbatim event-aware disjunct on both new jobs and in fact all 5 (`post-release-verify.yml:303`, `:408`); dry evaluation recorded under **both** trigger events on real runs (`02-EVIDENCE.md:806` `workflow_run` 31338004416; `:841-855` `workflow_dispatch` 31338409898, all 7 jobs ran, none skipped). **Absent: the count assertion** — `rg 'workflow_run\|conclusion\|event_name' --glob '*_test.go'` → 0 hits repo-wide. Guard correct today; nothing prevents its regression. **Accepted risk AR-07**, todo filed | closed (accepted — AR-07) |
| T-02-19 | Tampering | The downloaded asset the suite executes | high | mitigate | `cosign verify-blob` with issuer + identity-regexp identical to `verify:release-assets`, under `set -euo pipefail`, **precedes** `chmod +x` (`Taskfile.yml:2452-2457` → `:2464`); sha256 recorded `:2461`; ordering also legible in the graph via `needs: [resolve-tag, verify-supply-chain]` (`post-release-verify.yml:407`) | closed |
| T-02-20 | Spoofing | `TestAppleSecretsScopedToSingleReleaseJob`'s scope of proof | low | accept | The test proves where credential **names** are consumed in workflow files; it cannot prove GitHub-side secret scope (repository vs org vs environment). Limitation written into the test's own doc comment (`release_workflow_shape_test.go:1197-1205`). See AR-03 | closed |
| T-02-21 | Information Disclosure | The release step's process environment | low | accept | Unavoidable — the signing tool needs the values. Blast radius kept minimal: the step body is `run: task release:goreleaser` only, with no shell logic, no env dump, no diagnostic config printing (`release.yml:191-200`). See AR-04 | closed |
| T-02-22 | Tampering | A published release that silently shipped un-notarized | critical | mitigate | Three independent observers all fired: pipe log showed `notarizing and waiting…`/`notarized`, not a skip (`02-EVIDENCE.md:620-628`); Gatekeeper job's own per-arch `GATEKEEPER-EVIDENCE` with `digest_match=true observed=accepted exit=0` (`:630-668`); maintainer Safari download with verbatim `source=Notarized Developer ID` (`:649-656`, `:841-855`) | closed |
| T-02-23 | Repudiation | Evidence recorded from a source other than the published asset | high | mitigate | Provenance-labelled entries per arch — 3 HASH + 2 BINDING, kinds explicit (`02-EVIDENCE.md:689-719`); quoted `gh api …` where a verification result stands in for a hash (`:703`); binding scope stated verbatim (`:721-732`); no release deleted or re-pushed (`02-07-SUMMARY.md:185`) | closed |
| T-02-24 | Tampering | Pre-existing `verify:self-upgrade` download-then-execute path | medium | accept | Pre-existing debt, not a phase-2 regression. Concrete artifact filed: `.planning/todos/pending/2026-08-09-verify-self-upgrade-download-then-execute-has-no-signature-check.md`, naming the job, line range (`Taskfile.yml:1969-1982`) and exposure. See AR-05 | closed |
| T-02-SC | Tampering | Package installs (npm/pip/cargo/go) | low | accept | No new dependency crosses `go.mod` or `go.tool.mod` this phase — quill is already an indirect dep of the pinned goreleaser tool module and `notarize:` is config-only. Audit table `02-RESEARCH.md:177-186` (`[SLOP]: none`, `[SUS]: none`); `git diff main...HEAD -- go.mod go.tool.mod go.sum` → empty. See AR-06 | closed |

*Status: open · closed · open — below `high` threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `workflow.security_block_on` count toward `threats_open`*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

**Verification depth.** All named mitigation tests re-run uncached by the auditor:
`go test -count=1 -run 'TestNotarizeMacos|TestAppleSecrets|TestOIDCWrite|TestPostReleaseJobs|TestRehearseNotarize|TestReleaseWorkflow|TestSignsSidecar' ./internal/upgrade/` → ok;
`-run TestResolveTestBinPath ./test/integration/` → ok.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01 | T-02-04 | `GH_TOKEN` is a read-scoped `github.token`-class credential already required by `verify:release-assets` and `verify:self-upgrade` under the same shape, and is required here as a reproducibility input so the identical target runs unchanged in the 02-06 CI job. The target never echoes it. | maintainer (plan 02-01) | 2026-08-09 |
| AR-02 | T-02-10 | The harness executes whatever `CODEGRAPH_TEST_BIN` names. In CI that is `gh release download` against this repository's own release, already covered by the cosign and attestation verification in the same workflow; locally it is the developer's own path. Duplicating signature verification inside a test harness would be a second, drifting implementation of `internal/upgrade/verify.go`. | maintainer (plan 02-03) | 2026-08-09 |
| AR-03 | T-02-20 | `TestAppleSecretsScopedToSingleReleaseJob` proves where credential **names** are consumed in workflow files. It cannot prove the GitHub-side scope of the secrets themselves — repository vs organization vs environment — nor their access policies. Those are dashboard facts, covered by `user_setup` and the 02-07 checkpoint. Limitation is written into the test's own doc comment so a later reader does not over-trust a green result. | maintainer (plan 02-06) | 2026-08-09 |
| AR-04 | T-02-21 | The five credentials are visible to every process the Release step launches; the signing tool needs them, so this is unavoidable. Mitigation is blast-radius containment: the step invokes a single Taskfile target with no additional shell logic, no environment dump, and no diagnostic printing of configuration. | maintainer (plan 02-06) | 2026-08-09 |
| AR-05 | T-02-24 | `verify:self-upgrade` downloads a prior release binary, `chmod +x`es it and executes it with no signature verification between — the same ordering hazard T-02-19 closes for the new suite job, three jobs away and untouched by this phase. Pre-existing debt, not a phase-2 regression; changing that job's shape would disturb a verification path the milestone depends on. Flagged as a concrete filed todo, not prose. | maintainer (plan 02-06) | 2026-08-09 |
| AR-06 | T-02-SC | No new dependency crosses `go.mod` or `go.tool.mod` in this phase. Package legitimacy audit recorded in `02-RESEARCH.md` with no `[ASSUMED]`, `[SUS]` or `[SLOP]` rows, so no legitimacy checkpoint is required. | maintainer (plan 02-02) | 2026-08-09 |
| AR-07 | T-02-18 | **Accepted during this audit.** The event-aware conclusion guard is present verbatim on all five jobs and has been empirically proven under **both** trigger events on real runs — so this is a regression risk, not a present exposure. The third mitigation leg, a count assertion over the file, was never written; nothing fails if a future edit drops the `github.event_name != 'workflow_run' \|\|` disjunct, and the resulting breakage is silent by construction (a green workflow whose jobs all skipped). Accepted on the strength of the recorded dual-trigger evidence, with the gap carried forward as a filed todo rather than prose. | maintainer (`/gsd-secure-phase 02`) | 2026-08-09 |

---

## Follow-Up Artifacts

Filed under `.planning/todos/pending/` so the gaps survive this phase's close:

| Todo | Threat Ref | Severity |
|------|------------|----------|
| `2026-08-09-post-release-verify-event-aware-conclusion-guard-has-no-regression-assertion.md` | T-02-18 | high |
| `2026-08-09-dry-run-signed-additions-only-diff-guard-passes-vacuously.md` | T-02-08 | medium |
| `2026-08-09-verify-self-upgrade-download-then-execute-has-no-signature-check.md` (pre-existing, filed by plan 02-06) | T-02-24 | medium |

---

## Unregistered Flags

Surfaced by the auditor outside the plan-time register. All three were already
disclosed in `02-REVIEW.md` or `02-07-SUMMARY.md`; none was ever mapped to a
threat ID. Recorded here so they stop being invisible to the threat register.

Note: four of seven summaries (`02-01`, `02-03`, `02-05`, `02-06`) contain no
`## Threat Flags` section at all, so that section was not a usable inventory for
this audit — these came from independent search.

- **UF-1 — stale `binary_signs:` rationale outside T-02-07's declared scope.**
  `.github/workflows/release.yml:34-39` and `:131-134` state that signing "is done
  declaratively by `.goreleaser.yaml`'s `binary_signs:` pipe", and
  `docs/RELEASE-PROCEDURES.md:130-133` narrates "**Sign** — `binary_signs:` shells
  out to cosign keyless". D-18 (this phase) removed `binary_signs:` precisely
  because it signs **pre-notarization** bytes. This is the identical repudiation
  vector T-02-07 registered — created by this phase's own change — but T-02-07
  scoped its component to `.goreleaser.yaml` only, leaving it unmapped. Disclosed
  as WR-01 (`02-REVIEW.md:45-49`).
- **UF-2 — darwin/amd64 cosign binding coverage gap.** `verify:release-assets`
  verifies `cosign verify-blob` + `gh attestation verify` against
  `codegraph_${TAG}_linux_amd64` only (`Taskfile.yml:~1903-1917`, recorded
  `02-EVIDENCE.md:721-732`); `verify:notarized-suite` covers darwin/arm64.
  **darwin/amd64 has no automated cosign binding check anywhere in the pipeline.**
  Disclosed `02-07-SUMMARY.md:176` / WR-03.
- **UF-3 — `resolve-tag` pagination fallback.** `post-release-verify.yml:158-165`
  wraps a `gh api --paginate --jq` result in a JSON array that does not merge
  cleanly across pages. This branch decides **which tag the entire post-release
  verification suite assesses** — a verification-integrity surface. Fails loudly
  rather than silently wrong at a page boundary. Disclosed as WR-02.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-09 | 25 | 24 | 1 (non-blocking, medium) | `gsd-security-auditor` via `/gsd-secure-phase 02` |

### Security Audit 2026-08-09

| Metric | Count |
|--------|-------|
| Threats found | 25 |
| Closed | 24 |
| Open | 1 |
| Open at or above `high` (blocking) | 0 |

Register origin: plan-time (`register_authored_at_plan_time: true`, all 7 plans).
Mode: verify-mitigations (not retroactive-STRIDE). ASVS L1, `block_on: high`.

Auditor verdict `## OPEN_THREATS` with 2 open (T-02-18 high, T-02-08 medium).
T-02-18 accepted as documented risk AR-07 by maintainer decision, closing it.
T-02-08 remains open at medium — below the `high` block threshold, therefore
non-blocking — with a filed todo. `threats_open: 0`.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log (AR-01 … AR-07)
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-09
