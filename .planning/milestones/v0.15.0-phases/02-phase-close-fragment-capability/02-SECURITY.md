---
phase: "2"
slug: "phase-close-fragment-capability"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-26"
---

# Phase 2 — Security

> The per-phase security contract: threat register, accepted risks and audit trail.
>
> **Register source:** the `<threat_model>` blocks in 02-01 to 02-06-PLAN.md, written at plan time. None of the SUMMARY files raised a Threat Flag.
>
> **Verification depth:** ASVS L1, grep-level evidence plus passing test legs. Threats are verified against the current state (capability `v0.1.1`), not the state when each plan was written.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| SUMMARY.md text → shell | Model-chosen, SUMMARY-derived body text reaches `changie new` argv | Untrusted text (possible shell metacharacters) |
| gh → script | gh stdout (a PR number) and stderr (error text) reach the script's notes and argv | Integers; error text that may contain tokens |
| script → host git repository | The script stages and commits in the host working tree | Fragment files and commit trailers |
| maintainer machine → GitHub | A repository, tags, a branch and a PR are created or pushed under the maintainer's credentials | Source, credentials via the gh credential helper |
| private GitHub tag → project and global installs | The bundle and SKILL.md are installed verbatim; skill bodies are not content-scanned | Agent-instruction text, a shell script |
| gsd-core surface pass → `~/.claude/skills` | One verb can re-sync the whole global skills directory | Skill files for every project on the machine |
| codegraph-go working tree → git history | Machine-specific install ledgers must not be published | Local paths, install metadata |
| codegraph-go config → CHG-04 guarantee | Making `PR` optional weakens a Phase 1 refusal | Changelog vocabulary |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-01 | Tampering / EoP | Entry handling in write-fragments.sh; SKILL.md entry writing | high | mitigate | Entries are written with the file tool and read with `IFS= read -r`. Each body is one element of the `NEW_ARGV` array (`-b "${body}"`, write-fragments.sh:462), and the script has no `eval`. The `[argv-literal]` leg passes in the 32-leg suite. | closed |
| T-02-02 | Tampering | Fragment commit scope | medium | mitigate | `git add --` with exactly the written paths. The `[commit-scope]` leg passes. | closed |
| T-02-03 | Repudiation / Spoofing | PR number in fragments | medium | mitigate | Superseded by D-13 (PR optional): "no PR → named skip" became "no PR → note, fragment without a PR". The residual risk is covered by T-02-23 and T-02-24, both closed. `--pr` is still validated as a positive integer (`bad-pr`). | closed |
| T-02-05 | Tampering | Concurrent or interrupted dispatch | low | mitigate | An atomic mkdir lock (`changie-fragments.lock`), trailer idempotency and rollback. The `[rollback]` leg passes. | closed |
| T-02-06 | DoS | Host workflow at verify:post | low | mitigate | Every skip exits 0. `CI=true` and `GH_PROMPT_DISABLED=1` are set. The manifest step has `onError: skip`. Observed live at Phase 2's own close: `skip: no-user-visible-change`, exit 0. | closed |
| T-02-07 | Tampering | Side effects of the test harness | low | mitigate | A `mktemp -d` workspace with an EXIT trap and an isolated `GSD_HOME` (02-06). The `[no-side-effects]` leg passes. | closed |
| T-02-08 | Tampering | SKILL.md instructions (agent-instruction surface) | medium | mitigate | A pre-publication rehearsal (02-02) and a real dispatch (02-05, 8/8 assertions). The no-`.claude/`-path rule is kept. | closed |
| T-02-09 | Tampering | Mutation demonstrations | low | mitigate | Detached worktrees under `mktemp -d`, byte-clean reverts, and a teardown check of the worktree count. Recorded in 02-CAPABILITY-LOG.md and 02-MUTATION-LOG.md. | closed |
| T-02-10 | Info Disclosure | README and the capability log | low | accept | Transcripts from throw-away projects and fixture PR 4242; no credentials; private repository. | closed |
| T-02-11 | Repudiation / EoP | Outward-facing GitHub actions | high | mitigate | A blocking-human checkpoint, answered `publish-both` by the maintainer. SHAs, URLs and the PR number are recorded in 02-03-SUMMARY.md. The v0.1.1 push was pre-approved by the maintainer, and D-13 records that approval. | closed |
| T-02-SC | Tampering | Capability source for both installs (v0.1.0, then v0.1.1) | high | mitigate | Annotated tags on the remote. The `v0.1.1` push was fast-forward only, and `v0.1.0` is unchanged. A clean HTTPS clone at the tag passes 32/32. The project `capability.json` and the global SKILL.md are `cmp`-equal to `git show v0.1.1:…`. The manifest declares no hooks. Transport is HTTPS, not the originally planned SSH (D-11 amendment). | closed |
| T-02-12 | Tampering | Tag integrity over time | medium | mitigate | Tag SHAs are recorded (v0.1.0 → 5700e60, v0.1.1 → 35b674e). Moving a tag is prohibited, and fixes ship as new tags. | closed |
| T-02-13 | Info Disclosure | Repository visibility | medium | mitigate | `gh repo view … --jq .visibility` returns `PRIVATE`. | closed |
| T-02-14 | DoS | Draft PR closed by `close-draft-prs.yml` | low | mitigate | The PR author is the OWNER and exempt. PR #88 is `OPEN`. | closed |
| T-02-15 | Info Disclosure | Milestone branch pushed before milestone close | low | accept | `.planning/` is public by design, and the maintainer chose the timing. | closed |
| T-02-16 | EoP / Tampering | Global materialization into `~/.claude/skills` | high | mitigate | A blocking-human checkpoint (`global-install`, D-14), then gsd-core verbs only. The before/after listing differs only by `gsd-changie-fragments`. The materialized SKILL.md is `cmp`-equal to v0.1.1's. | closed |
| T-02-17 | Info Disclosure | `.gsd-capabilities.json` and `.gsd/` | medium | mitigate | Each path is ignored by its own `.gitignore` line, and `git ls-files` shows neither. | closed |
| T-02-18 | Tampering | `.planning/config.json` (tool-owned) | medium | mitigate | Written only with `config-set`, after install. The values are confirmed with `config-get`. | closed |
| T-02-19 | DoS | Step active in every GSD project after the global install | low | accept | Projects with no `.changie.yaml` get a named `no-changie-config` skip, and it never blocks. Each project can opt out with `workflow.changie_fragments false`. | closed |
| T-02-20 | Spoofing | A mislabelled changelog entry from a misjudged dispatch | medium | mitigate | The 02-05 real dispatch asserted the kind, the absence of jargon, and the trailers. Fixes ship as new tags. | closed |
| T-02-21 | Info Disclosure | CONTRIBUTING references a private repository | low | accept | Only the name and install spec are published, and they are already public in ROADMAP and REQUIREMENTS. Access is gated by GitHub permissions. | closed |
| T-02-22 | Info Disclosure | gh stderr reaching notes or logs | high | mitigate | A redaction expression replaces every `gh[pousr]_…` and `github_pat_…` token with `[REDACTED]`, and gh stdout is never echoed. The `[note:pr-lookup-failed]` leg (noauth-token stub) passes. | closed |
| T-02-23 | Tampering / Spoofing | A looked-up PR number reaching a fragment | medium | mitigate | The lookup must match `^[1-9][0-9]*$`; otherwise the result is `note pr-lookup-invalid` with the PR unset. The `[note:pr-lookup-invalid]` leg passes. | closed |
| T-02-24 | Repudiation | A fragment silently written with no PR | low | mitigate | Exactly one named note is printed, both trailers are kept, and the Phase 4 fragment-required gate is the backstop. | closed |
| T-02-25 | Tampering | `.changie.yaml` relaxed beyond D-13 | medium | mitigate | TestChangieConfigShape pins the custom key set and `optional: true`. check:changie legs 5–8 plus mutation families (a)–(c) cover it (02-MUTATION-LOG.md). check:changie passes 12/12. | closed |
| T-02-26 | DoS | A host that still requires PR | low | accept | The host's changie refuses the write and the run rolls back with `changie-failed`. `onError: skip` keeps the host workflow unblocked. | closed |
| T-02-27 | Info Disclosure | HTTPS credentials during push and clone | low | mitigate | `GIT_TERMINAL_PROMPT=0` on every network call. The remote URL carries no embedded credential; gh's credential helper supplies it. | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low. Only open threats at or above workflow.security_block_on count toward threats_open.*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-02-01 | T-02-10 | Evidence transcripts contain no credentials, and the repository is private | plan 02-02 threat model | 2026-09-25 |
| AR-02-02 | T-02-15 | The milestone branch is published early; `.planning/` is public by design | maintainer (`publish-both`, 02-03) | 2026-09-26 |
| AR-02-03 | T-02-19 | The step is active as a named skip in other GSD projects on this machine | maintainer (`global-install`, 02-04 / D-14) | 2026-09-26 |
| AR-02-04 | T-02-21 | CONTRIBUTING names a private repository | plan 02-05 threat model | 2026-09-26 |
| AR-02-05 | T-02-26 | A host that still requires PR refuses no-PR writes, and the workflow stays unblocked | plan 02-06 threat model (D-13) | 2026-09-26 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-26 | 27 | 27 | 0 | /gsd-secure-phase 2 (orchestrator, L1 grep and test-leg evidence; auditor skipped per the short-circuit rule: register authored at plan time, ASVS L1, threats_open 0) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-26
