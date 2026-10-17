---
phase: "3"
slug: "rename-stub-removal"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-26"
---

# Phase 3 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| user argv → cobra root | Untrusted command-line input; no `DisableFlagParsing` stub accepts arbitrary arguments unexamined any more | CLI arguments (untrusted, non-sensitive) |
| scratch worktree mutations → main checkout | Deliberately broken code exists briefly in a detached worktree and must never reach a commit | Source edits (integrity) |
| census positive control → repository | An untracked file containing the forbidden phrase exists briefly under `.github/` | A 3-line text file (integrity) |
| phase SUMMARY → capability model judgment → published changelog | SUMMARY text chooses the fragment kind and body that become release notes and drive the version bump | Release-note text and version (integrity) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-03-01 | Tampering | root command tree (revived alias, hint or hidden command) | medium | mitigate | `test/integration/removed_verbs_test.go` `TestRemovedVerbsAreUnknownCommands` passes 4 of 4 with the `executed 4 of 4 cases` guard line; `03-MUTATION-LOG.md` §Family (a) (line 223) shows it RED against a revived alias and a `SuggestFor` hint | closed |
| T-03-02 | Tampering | scratch-worktree mutations leaking into history | medium | mitigate | `feat(cli)!` commit `90dc419a` touches exactly 4 paths (`renamed.go` D, `renamed_test.go` D, `root.go` M, `cli-reference-allowlist.txt` M); `git worktree list` shows no scratch worktree left behind | closed |
| T-03-03 | Tampering | `.github/__census_control__.md` committed by accident | low | mitigate | File absent on disk; `git ls-files .github` has no `census_control` entry | closed |
| T-03-04 | Repudiation / Spoofing | a hand-written fragment presented as the capability's | medium | mitigate | No commit in the phase range `84778a82..d9892beb^` touches `.changes/unreleased/`; the only fragment commit is `d9892beb` (`docs(03): add changelog fragments`) carrying `Changie-Phase: 3` / `Changie-Summaries: 03-01` trailers, written by the capability's `write-fragments.sh` | closed |
| T-03-05 | Tampering | release-notes integrity (wrong kind or wrong replacement) | medium | mitigate | Fragment `kind: Breaking`; body names `search --full <term>` and `daemon unlock [path]`; `task changie -- next auto` → `v0.15.0` | closed |
| T-03-06 | Denial of Service | scripts still invoking the removed verbs | low | accept | See Accepted Risks Log AR-03-01 | closed |
| T-03-07 | Information Disclosure | gh output during the capability's PR lookup | low | transfer | Transferred to capability v0.1.2's redaction (Phase 2 T-02-22); this phase does not change the capability | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-03-01 | T-03-06 | The intended breaking change v0.14.0 promised for this minor; announced by the `BREAKING CHANGE` footer and the `Breaking` fragment; replacements have existed since v0.14.0 | plan 03-01 threat model (disposition `accept`) | 2026-09-26 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-26 | 7 | 7 | 0 | secure-phase orchestrator (L1 grep-depth; auditor not spawned per the ASVS-1 short-circuit) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-26
