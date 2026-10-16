---
phase: "5"
slug: "release-chain-release-please-retirement"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-10-14"
---

# Phase 5 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| merged fragment text → release PR body and squash commit message | fragment bodies authored in reviewed PRs flow into the PR body, CHANGELOG.md and, on merge, the commit message on main | contributor-authored text (public) |
| App installation token → third-party node action | create-pull-request receives the App token and pushes `changie/release` | short-lived App installation token (secret) |
| GITHUB_OUTPUT → `with:` inputs | the derived version crosses from a run body into action inputs | version string |
| App token → git push credential | the release-tag checkout persists the App token, and release:tag's push uses it | App installation token (secret) |
| push to main → tag | any change to CHANGELOG.md that reaches main can reach release:tag | release trigger |
| Taskfile target → origin | a target that pushes tags is runnable by any contributor | tag ref on origin |
| .changes file → public Release body | the reviewed notes become the Release text | release notes (public) |
| release.yml (locked contract) → codegraph upgrade | the cosign SAN and trigger every installed binary verifies against | signing identity |
| fork PR body → Python regex | the PR body is attacker-controlled under pull_request_target | untrusted text |
| PR changed-file list → hygiene verdict | paths decide whether template and issue-link checks apply | file paths |
| repository config → tag authority | `.planning/config.json` git.create_tag and `.changie.yaml`'s Breaking bump decide whether a release can be cut or a version can jump | config values |
| runbook → maintainer action | a maintainer follows the doc during a release or a rollback | operator instructions |
| milestone branch → main (Phase 6) | the release-please deletion reaches main only with the milestone merge | workflow files |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-05-01 | Denial of Service / Spoofing | release-pr.yml PR authorship | high | mitigate | create-pull-request `token: ${{ steps.app-token.outputs.token }}`; no non-comment default-token reference; TestReleasePRWorkflowShape passes; Family 05-01 | closed |
| T-05-02 | Tampering | skip-ci phrase smuggled through a fragment into the squash commit message | medium | mitigate | bracketed skip-phrase scan in Taskfile `release:batch`; TestReleaseBatchTargetShape passes; Family 05-01/c | closed |
| T-05-03 | Tampering | fragment text reaching a shell | medium | mitigate | release-pr.yml run bodies are exactly `task check:cross` and `task release:batch`, no `${{ }}` in any run body; version reaches the action only via `with:` | closed |
| T-05-04 | Denial of Service | draft release PR closed by close-draft-prs.yml | low | mitigate | `draft: false` in release-pr.yml, pinned by TestReleasePRWorkflowShape; Family 05-01/e | closed |
| T-05-05 | Information Disclosure | rendered notes published as the PR body | low | accept | see Accepted Risks Log AR-05-01 | closed |
| T-05-SC | Tampering | new third-party Action `peter-evans/create-pull-request` | high | mitigate | full-SHA pin `5f6978faf089d4d20b00c7766989d076bb2fc7f1 # v8.1.1` present in release-pr.yml; no package install | closed |
| T-05-06 | Repudiation / Denial of Service | tag pushed with the default token never releases | high | mitigate | release-tag.yml checkout `token:` is the app-token output; no non-comment default-token reference; TestReleaseTagWorkflowShape and TestTagCreationHasSingleAppTokenSite pass; Family 05-02/a | closed |
| T-05-07 | Tampering | a second, unreviewed tag path in a workflow, local action or Taskfile target | high | mitigate | GRD-16 scan (TestTagCreationHasSingleAppTokenSite, TestTagCreationScanFlagsPlantedSites) passes, exactly one site; Families 05-02/b, c, f-i | closed |
| T-05-08 | Tampering | a human runs release:tag locally (D-06R) | medium | mitigate | C8 `GITHUB_ACTIONS` precondition in `release:tag`; rehearsal run 4 shows the refusal | closed |
| T-05-09 | Tampering | a hand-written `.changes/vX.Y.Z.md` plus a CHANGELOG.md edit merged to main produces a tag and a release | medium | accept | see Accepted Risks Log AR-05-02 | closed |
| T-05-10 | Tampering | a tag-creating command hidden in a script a workflow invokes by path | low | accept | see Accepted Risks Log AR-05-03 | closed |
| T-05-11 | Tampering | an ls-remote transport error read as "absent" | low | mitigate | `release:tag` `case` treats only rc 2 as absent; rehearsal run 3; Family 05-02/e | closed |
| T-05-12 | Tampering | GoReleaser config rewriting or emptying the body | medium | mitigate | TestGoreleaserChangelogNotDisabled and TestReleaseBlockDoesNotRewriteReleaseBody pass; Families 05-03/a, d | closed |
| T-05-13 | Denial of Service | an empty or missing notes file publishing an empty Release | medium | mitigate | `[ ! -s "${NOTES}" ]` refusal in both `release:goreleaser` and `release:dry-run` (C6); Family 05-03/c | closed |
| T-05-14 | Tampering | drift in release.yml's locked contract | high | mitigate | release.yml differs from 7b12cb92 by one comment-only hunk; TestReleaseWorkflowTriggerIsTagPushOnly and TestOIDCWriteScopedToSingleGoreleaserJob pass | closed |
| T-05-15 | Elevation of Privilege | a rehearsal reaching a publish or sign path | low | mitigate | `release:dry-run` runs `--skip=publish,sign,validate` with no `--snapshot` flag in its commands; TestReleaseDryRunRehearsesReleaseNotes passes | closed |
| T-05-16 | Denial of Service | catastrophic backtracking in the exemption markers (WR-01) | medium | mitigate | linear marker patterns; test_exempt_markers_are_linear_on_adversarial_bodies passes (46 of 46); Family 05-04/c | closed |
| T-05-17 | Tampering | a `.changes/`-only PR skips the template and issue-link checks | low | accept | see Accepted Risks Log AR-05-04 | closed |
| T-05-18 | Denial of Service | many unterminated markers scanned once per opener | low | accept | see Accepted Risks Log AR-05-05 | closed |
| T-05-19 | Tampering | the two allowlists drift so one gate exempts what the other blocks | low | mitigate | test_hygiene_allowlists_mirror_require_issue_link (set equality) passes; Family 05-04/b | closed |
| T-05-20 | Repudiation | a re-pointed guard that passes vacuously | medium | mitigate | RED-against-replacement simulation recorded first, post-re-point simulation, mutations 05-05/a-c | closed |
| T-05-21 | Tampering | an unintended 1.0 bump from one Breaking fragment | medium | mitigate | TestChangieBreakingBumpsMinorPreOne and TestChangieConfigShape pass; Family 05-05/a | closed |
| T-05-22 | Tampering | GSD milestone tagging re-enabled | high | mitigate | TestGsdTagCreationIsDisabled passes with code tokens byte-identical to 5a8603f5; Family 05-05/b | closed |
| T-05-23 | Tampering | a doc instructing a hand-pushed tag (the old rc escape hatch) | medium | mitigate | no `git tag v…` / tag-push instruction in CONTRIBUTING.md or docs/RELEASE*.md; rc hatch removed with a verdict row (D-15) | closed |
| T-05-24 | Information Disclosure | quoting the App slug in docs | low | mitigate | docs say "the release GitHub App"; no slug match in CONTRIBUTING.md, docs/, .github/ or Taskfile.yml | closed |
| T-05-25 | Repudiation | a reference silently dropped without a recorded decision | low | mitigate | 05-CENSUS.md holds R01-R45, set-equal to the 45 base reference lines | closed |
| T-05-26 | Repudiation | a red suite or an unguarded window between deletion and re-pointing | medium | mitigate | deletion commit ffd84c4a touches exactly four paths (three files plus their registry entries); package green at that commit; RED against the replacement recorded first | closed |
| T-05-27 | Repudiation | a vacuous census (blind instrument or over-broad exclusion) | medium | mitigate | recorded instrument in 05-CENSUS.md with exactly REL-14's three exclusion clauses; positive control found the `.changes/header.tpl.md` plant and not the excluded plants; real census empty | closed |
| T-05-28 | Tampering | an outward-facing action during the phase (tag, push, PR) | high | mitigate | local tag list, remote tag refs, origin/main and the PR #88 head ref fingerprinted before and after the tag-touching plans: unchanged; remote v0.15 tag-ref count 0 | closed |
| T-05-29 | Elevation of Privilege | runbook section 2's table and section 7's sentence presenting a hand-pushed prerelease tag as normal | low | mitigate | 5653cbf9 (05-08, G-05-1): section 2 lists only the Stable and Internal milestone marker shapes and points to section 4's D-15 paragraph; section 7's prerelease-retag sentence and section 1's prerelease parenthetical are removed; 05-08 automated verify exits 0; downstream `release:goreleaser` refuses a tag with no `.changes/<tag>.md` (T-05-13) | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-05-01 | T-05-05 | The release PR body is the same text that becomes the public Release; a human reviews the release PR before merge (D-04) | plan-time disposition, 05-01-PLAN.md threat model | 2026-10-14 |
| AR-05-02 | T-05-09 | Merging to main needs protect-main's human merge, and anyone who can merge can already push a `v*` tag (protect-main has no tag rule); CONTRIBUTING forbids hand-written version files | plan-time disposition, 05-02-PLAN.md threat model | 2026-10-14 |
| AR-05-03 | T-05-10 | Stated GRD-16 scan boundary, mirroring resolveReleasePathClosure's boundary 1; no workflow invokes git through a script today | plan-time disposition, 05-02-PLAN.md threat model | 2026-10-14 |
| AR-05-04 | T-05-17 | Hygiene checks only; fragment-required and human review are unaffected, and the release PR needs this exemption (D-05) | plan-time disposition, 05-04-PLAN.md threat model | 2026-10-14 |
| AR-05-05 | T-05-18 | Bounded by GitHub's PR body limit at about a second, versus hours before the WR-01 fix | plan-time disposition, 05-04-PLAN.md threat model | 2026-10-14 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-14 | 29 | 29 | 0 | secure-phase orchestrator (ASVS L1 grep-depth, register authored at plan time) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-16 (re-audit after gap closure 05-08)

## Security Audit 2026-10-16

| Metric | Count |
|---|---|
| Threats found | 30 |
| Closed | 30 |
| Open | 0 |
