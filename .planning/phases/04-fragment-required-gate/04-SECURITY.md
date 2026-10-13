---
phase: "4"
slug: "fragment-required-gate"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-10-13"
---

# Phase 4 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| fork PR → pull_request_target job | A fork author controls the PR body, file paths and branch content, while the job runs with a base-repo token | PR body, changed paths, branch content (untrusted, integrity) |
| gh REST response → verdict | The changed-file list is the gate's only view of the change; truncation or malformation would hide shipped code | Changed-file list (integrity) |
| script stdout → Actions runner | The runner interprets `::` lines as workflow commands, and the printed paths and reasons are fork-authored | Log lines (untrusted, integrity) |
| scratch-worktree mutations → main checkout | Deliberately broken code exists briefly and must never reach a commit | Source edits (integrity) |
| census positive control → repository | An untracked file holding the removed tokens exists briefly under .github/ | A short text file (integrity) |
| repository → live ruleset | After D-08, nothing in git mirrors protect-main's required contexts | Required-check context names (integrity) |
| milestone branch → main | Milestone-only commits (a Breaking CLI removal, changie baseline, planning trail) must not cross | Commits and file content (integrity) |
| agent → public GitHub | Pushing, opening issues and opening PRs are outward-facing acts | Branch, issue and PR text (integrity) |
| agent → main | Merging to main is a maintainer act | Merge to the protected branch (authorization) |
| main → milestone history | D-06 requires a history-preserving merge | Branch history (integrity) |
| agent → repository settings | A ruleset write changes merge rules for every contributor | Ruleset configuration (integrity, authorization) |
| ruleset read → PUT body | A body built from an incomplete read would silently drop protections | Ruleset JSON (integrity) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-04-01 | Tampering / Elevation of Privilege | fragment-required.yml checkout | high | mitigate | `.github/workflows/fragment-required.yml` line 28 is `pull_request_target`; the checkout at line 47 sets no `ref:` (line 46 comment records why), and `rg 'ref:'` finds only that comment; the `uses:` set is exactly checkout, setup-go and the local install-task action (lines 47, 50, 57), all SHA-pinned except the local one | closed |
| T-04-02 | Tampering | PR_BODY and paths reaching a shell | high | mitigate | Untrusted values enter only as step env (lines 60-64); the run body is the single line `task check:fragment-required` (line 65); `scripts/fragment_policy.py` `_run_gh` (line 250) calls `subprocess.run(list(argv))` with no shell, and `rg 'shell=True\|os.system'` finds nothing | closed |
| T-04-03 | Tampering | workflow-command injection through printed paths and reasons | medium | mitigate | `escape_data` (`scripts/fragment_policy.py` line 175) is applied in `report` to the message, the exempt reason and every listed path (lines 271-298); `task check:fragment-policy` runs `test_escape_data_encodes_command_characters` and `test_untrusted_paths_cannot_start_workflow_commands`, both `ok` | closed |
| T-04-04 | Spoofing / Repudiation | a truncated or malformed file list hiding shipped code | high | mitigate | `fetch_changed_files` (`scripts/fragment_policy.py` line 211) uses `gh api --paginate --slurp`, compares the fetched count with the PR's `changed_files`, and returns an api-error for malformed JSON; `test_main_fails_closed_on_truncated_file_list` and `test_main_fails_closed_on_malformed_json` both `ok`; mutation c2 shows the truncation test RED (`04-MUTATION-LOG.md` Family (c)) | closed |
| T-04-05 | Elevation of Privilege | token scope and cache under pull_request_target | medium | mitigate | `fragment-required.yml` lines 35-37 grant exactly `contents: read` and `pull-requests: read`; line 54 sets setup-go `cache: false` | closed |
| T-04-06 | Denial of Service | gate fails closed during a GitHub API outage | low | accept | See Accepted Risks Log AR-04-01 | closed |
| T-04-07 | Repudiation | vacuous pass of the unit suite | medium | mitigate | `Taskfile.yml` `check:fragment-policy` (line 2258) requires ran == defined, a floor of 36 and an exact `OK` line; run live: `Ran 36 tests`, `OK`, `executed 36 of 36 defined test methods (floor 36)`; `04-MUTATION-LOG.md` Family (d) (line 1289) shows both mutations RED with `unittest ran 24 of 36 defined test methods` and `result line is not exactly OK` | closed |
| T-04-08 | Repudiation | removal of the committed required-context mirror | low | accept | See Accepted Risks Log AR-04-02 | closed |
| T-04-09 | Tampering | mutations leaking into history | medium | mitigate | `git log` on `scripts/fragment_policy.py`, `scripts/test_fragment_policy.py` and `fragment-required.yml` shows only `(04-01)` commits; `git worktree list` shows no Phase 4 scratch worktree; `04-MUTATION-LOG.md` verdict records all ten mutations gated clean, reverted byte-clean (`git diff --quiet` exit 0) and the scratch worktree removed | closed |
| T-04-10 | Tampering | `.github/__census_control__.md` committed by accident | low | mitigate | File absent on disk; `git ls-files .github` has no `census_control` entry; `git log --all -- .github/__census_control__.md` is empty; `04-02-SUMMARY.md` records the file created under a trap and never staged | closed |
| T-04-11 | Tampering | cherry-pick carrying milestone-only changes to main | high | mitigate | `git diff --name-only fbfd8a0c 65a2b9a9` lists exactly the nine Phase 4 paths; `gh pr view 95` lists six cherry-picked ci/test commits; `git diff --quiet origin/main HEAD` over the three gate files exits 0 on the milestone; `04-03-SUMMARY.md` records per-path +/- lines equal to the milestone's Phase 4 diff | closed |
| T-04-12 | Tampering | release-please opening a release from the landing merge | high | mitigate | PR #95 title is `ci: add the fragment-required gate and retire the ruleset-drift guard`; its six commits are all `ci(04-0N)` or `test(04-01)` typed (`gh pr view 95`); the merged squash commit on main is `65a2b9a9 ci: ... (#95)`; none is a releasable type | closed |
| T-04-13 | Repudiation / Spoofing | publishing without maintainer consent | medium | mitigate | `04-03-SUMMARY.md` records the Task 2 blocking-human checkpoint and the maintainer's verbatim answer `open-new-issue`; branch push and PR #95 followed that answer, and the remote branch was verified empty before the push | closed |
| T-04-14 | Tampering | PR body or title text reaching a shell | low | mitigate | `04-03-PLAN.md` fixes `--body-file` for `gh issue create` and `gh pr create` (lines 197, 198, 246, 252) and the title as one quoted argument; live `gh pr view 95` body equals `04-03-PR-BODY.md` apart from one trailing newline | closed |
| T-04-15 | Elevation of Privilege | an agent merging to main | high | mitigate | Blocking-human checkpoint in 04-04; `gh api repos/seanb4t/codegraph-go/pulls/95` reports `merged_by: seanb4t`; `04-04-SUMMARY.md` records the maintainer reply `95 merged` and no `gh pr merge` run by the executor | closed |
| T-04-16 | Tampering | milestone history rewritten during the merge-back | medium | mitigate | `git rev-list --parents -n1 b47a52ce` lists parents `a5a40dba` then `65a2b9a9` (a `--no-ff` merge); `git reflog show gsd/v0.15.0-milestone` has no rebase or reset entry; `04-04-SUMMARY.md` records `git diff a5a40dba HEAD` empty | closed |
| T-04-17 | Tampering | review edits on the PR silently diverging from the milestone | low | mitigate | `04-04-SUMMARY.md` records `git diff ci/fragment-required-gate origin/main` over the nine paths as empty (0 bytes) before the merge-back; milestone and main still agree on the three gate files | closed |
| T-04-18 | Tampering / Elevation of Privilege | PUT body dropping bypass actors, contexts or rules | high | mitigate | Re-run read-only: the sorted contexts of `04-05-ruleset-put-body.json` equal the pre-read's 8 plus `fragment-required`; `bypass_actors`, `conditions` and `enforcement` of the pre and post reads are equal; the live ruleset (id 20157557) shows 9 contexts, `enforcement: active`, `bypass_actors` unchanged | closed |
| T-04-19 | Elevation of Privilege | an agent editing the ruleset unprompted | high | mitigate | `04-05-SUMMARY.md` records the checkpoint first answered `hold`, then the maintainer's reply `run-prepared`, and one PUT run once with the prepared command and no retry | closed |
| T-04-20 | Repudiation | an unrecorded settings change | medium | mitigate | `04-05-ruleset-pre.json`, `04-05-ruleset-put-body.json` and `04-05-ruleset-post.json` are committed (`0f2643b4`); the summary records `updated_at` `2026-10-13T12:16:55.257-04:00` and observation time `2026-10-13T16:17:04Z`; the live ruleset's `updated_at` matches | closed |
| T-04-21 | Denial of Service | a misbehaving required gate blocking all merges | medium | accept | See Accepted Risks Log AR-04-03 | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-04-01 | T-04-06 | Failing closed is GATE-03's requirement. A blocked merge is recoverable through a re-run or the ruleset's admin bypass (RepositoryRole 5) | plan 04-01 threat model (disposition `accept`) | 2026-10-13 |
| AR-04-02 | T-04-08 | D-08: a required context that never reports blocks every PR at "Expected — waiting", which is loud, not silent. 04-05 re-reads the live ruleset before and after its edit and records both reads (D-07) | plan 04-02 threat model (disposition `accept`) | 2026-10-13 |
| AR-04-03 | T-04-21 | The gate is proven by 36 literal-input cases and nine RED mutations (04-01, 04-02). Recovery is the admin bypass (RepositoryRole 5) or a ruleset edit, a maintainer act stated at the checkpoint | plan 04-05 threat model (disposition `accept`) | 2026-10-13 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-13 | 21 | 21 | 0 | secure-phase orchestrator (L1 grep-depth; auditor not spawned per the ASVS-1 short-circuit) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-13
