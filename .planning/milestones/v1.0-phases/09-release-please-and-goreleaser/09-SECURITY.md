---
phase: 9
slug: release-please-and-goreleaser
status: secured
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
block_on: high
created: 2026-08-01
---

# Phase 9 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
>
> **Register origin:** authored at plan time. All 8 PLAN.md files carry a `<threat_model>`
> block; the auditor verified mitigations rather than scanning for new threats
> (`register_authored_at_plan_time: true`).

---

## Trust Boundaries

Consolidated from the eight plans' threat models. The recurring theme is that an *automated*
actor gained write access to `main` and to tag refs, and tag refs are what bind the cosign
signing identity every already-shipped binary checks.

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| GitHub App private key → repository secret store → workflow runtime | the PEM leaves the maintainer's machine, lives as a repo secret, and is consumed by a runner process that mints an installation token | private key material (high sensitivity) |
| App installation scope → everything the App can do in this repository | the installation grant is the real authorisation surface, independent of any workflow's `permissions:` block | authorisation grant |
| release-please workflow → `main` ref and tag refs | an automated actor gains write access to the default branch and the ability to create refs | commits, tags, release PRs |
| tag ref → `release.yml` signed build → shipped binaries' cosign SAN | the tag binds the cosign OIDC identity; every published binary hard-codes the workflow filename and tag-ref pattern it will accept | signing identity (LOCKED contract) |
| workflow context (`github.*`) / `$TAG` / `$REPO` → `run:` shell | attacker-influenced text would reach a shell interpreter if interpolated directly | untrusted strings |
| contributor-supplied PR title → lint step → merged commit → release-please's parser → published changelog | fully attacker-controlled text from any fork PR author, ending in release notes users read | untrusted strings |
| `assemble` job → GitHub Releases API | the job holds `contents: write` and mutates a public artifact set | published artifacts |
| cosign keyless signing → public Sigstore transparency log | signing events are permanent, public, and unredactable | public record |
| the published release → every existing user's upgrade path | this release becomes what `codegraph upgrade` resolves as "latest", permanently | end-user binaries |
| runbook prose → maintainer action on a live repo | the document is the only instruction set for irreversible actions (tags, merges, rollbacks) | operator instructions |
| repository Actions configuration → whether an automated actor may open PRs | a setting outside any file in this repo gates the whole pipeline | platform config |

---

## Threat Register

**52 threats. 38 mitigated-CLOSED · 5 accepted (live) · 8 moot · 1 eliminated · 0 open.**

Full per-threat evidence is in the audit output; this table records disposition and status.
Every CLOSED verdict was backed by a cited artifact — file:line, workflow job, test name, or
a command run and its output — not by the plan's own claim that a mitigation was planned.

| Threat ID | Category | Severity | Disposition | Status |
|-----------|----------|----------|-------------|--------|
| T-09-01-01 | Spoofing | critical | mitigate | closed |
| T-09-01-02 | Elevation of Privilege | high | mitigate | closed |
| T-09-01-03 | Information Disclosure | high | mitigate | closed |
| T-09-01-04 | Tampering | high | mitigate | closed |
| T-09-01-05 | Tampering | high | mitigate | closed |
| T-09-01-06 | Denial of Service | low | accept | accepted |
| T-09-01-07 | Repudiation | high | mitigate | closed |
| T-09-02-01 | Tampering | critical | mitigate | closed |
| T-09-02-02 | Tampering | high | mitigate | closed |
| T-09-02-03 | Tampering | high | mitigate | closed |
| T-09-02-04 | Denial of Service | high | mitigate | closed |
| T-09-02-05 | Tampering | medium | mitigate | closed |
| T-09-02-06 | Repudiation | high | mitigate | closed |
| T-09-02-07 | Tampering | medium | mitigate | closed |
| T-09-03-01 | Tampering | high | mitigate | closed |
| T-09-03-02 | Repudiation | high | mitigate | closed |
| T-09-03-03 | Elevation of Privilege | high | mitigate | closed |
| T-09-03-04 | Denial of Service | medium | mitigate | closed |
| T-09-03-05 | Tampering | medium | mitigate | closed |
| T-09-04-01 | Spoofing | high | mitigate | closed |
| T-09-04-02 | Elevation of Privilege | high | mitigate | closed |
| T-09-04-03 | Information Disclosure | high | mitigate | closed |
| T-09-04-04 | Repudiation | medium | mitigate | closed |
| T-09-04-05 | Tampering | high | mitigate | closed |
| T-09-04-06 | Denial of Service | low | accept | accepted |
| T-09-05-01 | Elevation of Privilege | high | mitigate | closed ¹ |
| T-09-05-02 | Information Disclosure | high | mitigate | closed ² |
| T-09-05-03 | Spoofing | **high** | **accept** | **accepted — see log** |
| T-09-05-04 | Repudiation | medium | mitigate | closed |
| T-09-05-05 | Denial of Service | medium | mitigate | closed |
| T-09-06-01 | Tampering | high | mitigate | closed ³ |
| T-09-06-02 | Elevation of Privilege | high | mitigate | closed |
| T-09-06-03 | Tampering | high | mitigate | closed |
| T-09-06-04 | Denial of Service | medium | mitigate | closed |
| T-09-06-05 | Repudiation | high | mitigate | closed |
| T-09-07-01 | Spoofing | critical | mitigate | **moot** — see below |
| T-09-07-02 | Tampering | high | mitigate | **moot** |
| T-09-07-03 | Tampering | high | mitigate | **moot** |
| T-09-07-04 | Elevation of Privilege | high | mitigate | **moot** |
| T-09-07-05 | Repudiation | high | mitigate | **moot** |
| T-09-07-06 | Information Disclosure | low | accept | **moot** |
| T-09-07-07 | Tampering | medium | mitigate | **moot** |
| T-09-07-08 | Tampering | medium | mitigate | **moot** |
| T-09-08-01 | Spoofing | critical | mitigate | closed |
| T-09-08-02 | Tampering | critical | mitigate | closed ⁴ |
| T-09-08-03 | Tampering | high | mitigate | closed |
| T-09-08-04 | Tampering | medium | mitigate | closed |
| T-09-08-05 | Repudiation | high | mitigate | closed |
| T-09-08-06 | Tampering | critical | mitigate | closed ⁵ |
| T-09-08-07 | Information Disclosure | low | accept | accepted |
| T-09-08-08 | Tampering | ~~medium~~ n/a | ~~mitigate~~ **eliminated** | eliminated |
| T-09-08-09 | Tampering | medium | accept | accepted |

### Guards break-tested rather than merely observed

This repo has a documented history of guards that were present but could never fire
(Phase-8 `CR-01`/`WR-02`, and an inverted `rg -qv` guard). Three were forced to fail:

| Guard | Break test | Result |
|---|---|---|
| actionlint injection check | mutated `pr-title.yml`'s `"$TITLE"` → a direct `${{ github.event.pull_request.title }}` interpolation | exit 1 with an injection diagnostic — **fires** |
| `^Release-As:` version-forcing scan | same pattern against a synthetic `Release-As: 9.9.9` footer | 1 match — **fires** (real scan over `v0.1.0..v0.2.0`: 0) |
| `pretag-gate` 6-target sweep | injected a `bogusos/amd64` target into the loop | `::error::` emitted, exit 1 — **fires** |

### Caveats on CLOSED verdicts

**¹ T-09-05-01 — permissions verified, install breadth not.** The App's live permissions are
exactly the three at write plus mandatory `metadata:read`, `events: []`. The mitigation's
other clause — "account-only installation, single-repository install" — is **unverified**:
`repository_selection` requires an App-signed JWT, and the read returns HTTP 401. The
executor recorded this as unverified rather than asserting it. Not a gap in the threat's
declared component (over-scoped *permissions*), but the breadth claim remains unproven.

**² T-09-05-02 — two residuals, flagged at plan time not discovered late.** (a) Deletion of
the maintainer's local `.pem` is maintainer-attested, not machine-observable. (b) A
**shared** App (`fzy-release-please`, ID 3982691) was reused rather than a purpose-built one,
so a key leak's blast radius spans every repository it is installed on.

**³ T-09-06-01 — mechanism substituted.** `git reset --hard` was used instead of
`git merge --ff-only` (a standing `Bash(git merge *)` deny). `reset --hard` does not
intrinsically refuse to discard commits; that property was supplied manually by
pre-verifying `git merge-base --is-ancestor` on a clean tree. Outcome independently
confirmed — `git rev-list --count --merges origin/main` = **0** — and now durably enforced
by the active ruleset, which is stronger than the one-shot check it replaced.

**⁴ T-09-08-02 — the gate's mechanic was not executed as written.** No executor presented
the checkpoint; the maintainer was shown the computed version, changelog composition and
full evidence set, then merged by hand. The substance held. Separately, the standing
`Bash(gh pr merge *)` deny forced a second human decision at the moment of consequence,
which caught a proposed `--merge` that would have been `main`'s first merge commit in ~940.

**⁵ T-09-08-06 — the one declared mitigation that no longer holds as written.** The
mitigation specified "a whole-phase `git diff --quiet` assertion over `verify.go`,
`release.yml` and `.goreleaser.yaml`." Independently re-checked at audit time:

| File | `git diff v0.2.0..HEAD` |
|---|---|
| `internal/upgrade/verify.go` | unchanged |
| `.goreleaser.yaml` | unchanged |
| `.github/workflows/release.yml` | **changed** — commit `6486ac4` (PR #7) |

The sole functional line added after the release was cut:

```yaml
provenance-name: codegraph_${{ github.ref_name }}.intoto.jsonl
```

**The LOCKED identity did not drift.** `name: release`, the `v[0-9]*` tag trigger, the OIDC
issuer and the SAN are all intact; both workflow-shape guards are green; and the shipped
`v0.2.0` signature still verifies under `verify.go`'s compiled-in production constants. The
added expression sits in a reusable-workflow `with:` block, not a `run:` body, so it is not
injection surface, and actionlint passes. The change was disclosed in `09-08-SUMMARY.md`.

Closed **on the guards, not on the diff assertion.** The diff was a point-in-time check and
is now stale by design — the phase ended, the repo kept moving. The two workflow-shape tests
are the durable enforcement and remain green.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-09-01 | **T-09-05-03** (**high**) | A compromised App token authoring a tag push. Accepted because the signing identity every shipped binary checks is anchored to the **ref shape, not the actor** — an App-authored and a human-authored tag produce the same signing identity, so the token grants no ability to produce a differently-trusted artifact. Residual risk bounded by the three-permission scope and key rotation. **Now empirically true rather than argued:** `v0.2.0`'s tag was authored by `fzy-release-please[bot]`, the resulting certificate SAN contains no actor field, and a `v0.1.0` binary compiled before the App existed verified that signature. | maintainer (plan time) | 2026-07-29 |
| AR-09-02 | T-09-08-09 (medium) | Publishing through a pipeline no live run had ever exercised — the formally accepted cost of skipping 09-07. Accepted on an asymmetry: the one unproven fact could not yield a *wrongly signed* artifact, because producing any artifact at all requires the workflow to have run; its failure mode is an empty Release — loud, immediate, forward-fixable. **Outcome: did not materialise** (run `30675077940`, 11/11 green). | maintainer | 2026-07-31 |
| AR-09-03 | T-09-01-06 (low) | Six `go list` invocations per push to `main`; cost strictly dominated by the risk of a platform-specific `go.sum` failure reaching a release PR unnoticed. | maintainer (plan time) | 2026-07-29 |
| AR-09-04 | T-09-04-06 (low) | A maintainer misreading "no release PR opened" as a broken pipeline; residual risk is a wasted investigation, not a bad artifact. | maintainer (plan time) | 2026-07-29 |
| AR-09-05 | T-09-08-07 (low) | Permanent Sigstore transparency-log entries for this release; identical in kind to what the existing `v0.1.0` release already published. | maintainer (plan time) | 2026-07-29 |

> **AR-09-01 sits at the `high` block threshold.** It is recorded here explicitly rather than
> folded into the pass, because an accepted high-severity risk is a decision, not an absence
> of risk.

---

## Moot — the 09-07 group (8 threats)

`09-07-SUMMARY.md` carries `status: SKIPPED`. Task 1's `blocking-human` gate was approved,
Task 2 then halted on a `Bash(gh pr merge *)` deny, and during that pause the maintainer
re-examined the premise and chose to skip. **Nothing was merged, tagged, released,
dispatched or pushed; Tasks 2 and 3 never ran.**

These 8 threats are classified **moot**, deliberately neither OPEN nor CLOSED:

- Not **open** — the activity that would have created the exposure did not occur.
- Not **closed-by-implementation** — no mitigation was built, because nothing was built.
  Recording them as mitigated would be a false attestation.

**T-09-07-01 (critical) was nonetheless answered — by 09-08, not by 09-07.** The question
"does the cosign SAN issued for an App-token-triggered run satisfy
`releaseWorkflowRefPattern`?" was settled empirically by the real release: the `v0.2.0`
certificate's SAN is
`https://github.com/seanb4t/codegraph-go/.github/workflows/release.yml@refs/tags/v0.2.0`,
issued for a tag authored by `fzy-release-please[bot]`, verifying both through
`cosign verify-blob` and through `verify.go`'s own constants. The disposable rehearsal would
have proven exactly this and nothing more.

### Eliminated — T-09-08-08

*Reusing a tag name plan 09-07's disposable cut already consumed.* Skipping 09-07 meant no
disposable cut ever consumed a version number, so the threat ceased to exist and its
mitigation was correspondingly dropped. The plan records the row rather than deleting it,
deliberately: **this is the single respect in which skipping 09-07 left Phase 09 strictly
safer**, and the plan wanted that visible alongside the risk the skip added (T-09-08-09).

---

## Unregistered Surfaces — advisory, non-blocking

No SUMMARY in this phase carries a `## Threat Flags` section, so there are no
executor-declared flags. Two surfaces observed at audit time that the plan-time register
does not cover — neither undermines a closed threat, both are recorded so they are not lost:

1. **Four `pull_request_target` workflows with write scopes** — `require-issue-link.yml`,
   `pr-template-format.yml`, `auto-close-unsolicited-prs.yml`, `close-draft-prs.yml`. These
   landed *after* this register was authored and are outside its scope. Issue **#15**
   identifies fork-controlled file paths written to `$GITHUB_OUTPUT` with a fixed
   `PRFILES_EOF` heredoc delimiter. This does **not** undermine T-09-03-03, whose declared
   component is `pr-title.yml` specifically — that file uses `pull_request` (not `_target`),
   `contents: read`, no checkout, no token, zero `uses:`, all verified. But the *threat
   class* T-09-03-03 names now recurs elsewhere with no register entry. Recommend a register
   entry in whichever phase owns those workflows.
2. **Issue #14** — docs still describe SLSA provenance as attested over the checksums file
   in three places. This does **not** undermine T-09-04-01, whose component is the *cosign*
   identity regexp in §6(a); that section is byte-consistent with `verify.go` and was run
   verbatim to `Verified OK`. The stale prose is in the §6(b) SLSA neighbourhood, whose
   commands were confirmed correct.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Accepted | Moot | Eliminated | Open | Run By |
|------------|---------------|--------|----------|------|------------|------|--------|
| 2026-08-01 | 52 | 38 | 5 | 8 | 1 | **0** | gsd-security-auditor (opus), orchestrator-verified |

**Verification depth.** The auditor did not accept the orchestrator's supplied facts on
trust. It re-derived the load-bearing ones: downloaded the real `v0.2.0` darwin/arm64
artifact, ran `TestVerifyReleaseE2E` against it (PASS, both subtests, not skipped), ran the
runbook's `cosign verify-blob` verbatim (`Verified OK`), decoded the certificate and the SLSA
DSSE envelope, and confirmed one sha256 —
`a64c1549f012b065d077b89e63b683629e7a897a7a016b9e03d4ae8dea19c00b` — is simultaneously the
local file, the checksums entry, the SLSA subject and the cosign-verified digest. The
certificate carries `1.3.6.1.4.1.57264.1.20 = push` and `.19 = cce95f3c…`, equal to
`git rev-parse v0.2.0^{commit}`.

The orchestrator independently re-confirmed the two claims most likely to be wrong: the
`release.yml` post-release drift (caveat ⁵) and `git rev-list --count --merges origin/main`
= 0.

---

## Sign-Off

- [x] Every threat in the plan-time register accounted for — 52/52
- [x] All CLOSED verdicts backed by a cited artifact, not by the plan's own claim
- [x] Guards whose mitigation *is* a guard were break-tested, not merely observed
- [x] Accepted risks recorded with rationale, including one at the `high` block threshold
- [x] Skipped-plan threats classified moot, not silently marked mitigated
- [x] Stale mitigation (T-09-08-06) surfaced rather than quietly re-closed
- [x] `threats_open: 0` at severity ≥ `high` — and at every severity

**Approval:** secured 2026-08-01 via `/gsd-secure-phase 9`.
