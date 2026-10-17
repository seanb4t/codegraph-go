---
phase: "6"
slug: "first-release-through-the-chain"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-10-17"
---

# Phase 6 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| GitHub App token → tag push | the only credential that may author a release tag | App installation token (secret, never logged) |
| release PR merge (maintainer) → public Release | the one-way door: every installed `codegraph upgrade` sees the result | public release artifacts and notes |
| executor → GitHub | the executor reads; its only writes are one D-10 re-run and local commits | read-only API responses, run logs |
| downloaded release binary → local and devbox shells | a binary from the project's own Release runs with a writable HOME | public release assets |
| new release artifact → installed v0.14.0 | the swap happens only after in-process sigstore verification against the pinned identity | signed binary + `.sigstore.json` bundle |
| executor → maintainer sign-off | the agent's evidence is accepted only by the maintainer's reply | evidence summary (planning artifact) |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-06-01 | Spoofing | tag author | medium | mitigate | release.yml run 39054491964 `actor` and attempt-1 `triggering_actor` = `fzy-release-please[bot]`; release-tag logged `release:tag: pushed v0.15.0 at 0246de0e…` equal to the tag commit (06-EVIDENCE.md `## SHIP-02 App-authored tag and release.yml`) | closed |
| T-06-02 | Tampering | Release body vs `.changes/<TAG>.md` | medium | mitigate | D-11 `norm` + `cmp`, both sides non-empty: `CONTENT-EQUAL lines=5` (06-EVIDENCE.md `## SHIP-02 Release body (D-11)`); re-run green at validate-phase | closed |
| T-06-03 | Elevation of Privilege | executor taking an outward or irreversible action | high | mitigate | The release PR merge happened only at the 06-01 Task 2 blocking-human checkpoint; `mergedBy.login` = `seanb4t` recorded; no agent tag, Release, CHANGELOG or `main` write in the branch history; the one D-10 re-run (ci.yml 39053062791, nondeterministic `DB Size` flake) is recorded with its cause | closed |
| T-06-04 | Repudiation | authorship read from a re-run attempt | low | mitigate | Attempt 1 read through `actions/runs/<R>/attempts/1`; run-level `triggering_actor` recorded, not gated | closed |
| T-06-05 | Tampering | `codegraph upgrade`'s post-swap skill refresh against the maintainer's `~/.claude` | medium | mitigate | Every upgrade ran with `HOME` and cwd set to an empty scratch dir; the maintainer's manifest sha256 is the same before and after (06-EVIDENCE.md `## SHIP-03 upgrade darwin/arm64`) | closed |
| T-06-06 | Spoofing | the v0.14.0 starting binary | low | accept | See Accepted Risks Log (AR-06-01) | closed |
| T-06-07 | Repudiation | a `skipped` or wrong post-release-verify run read as green | medium | mitigate | Run 39054786827 selected by headSha = tag commit only; 7 of 7 jobs `success` (06-EVIDENCE.md `## SHIP-03 post-release-verify`) | closed |
| T-06-08 | Repudiation | a self-approved sign-off | medium | mitigate | 06-02 Task 3 was `gate="blocking-human"` and surfaced to the maintainer; `## Sign-off` quotes the maintainer's verbatim reply `and approved` | closed |
| T-06-09 | Information Disclosure | GitHub credentials copied onto a shared devbox | low | mitigate | No `--setup_github`; public assets only; ephemeral devbox removed with `devbox expire --force` and confirmed gone from `devbox list` (06-EVIDENCE.md `## SHIP-03 upgrade linux/amd64`) | closed |
| T-06-SC | Tampering | npm/pip/cargo installs | low | accept | See Accepted Risks Log (AR-06-02) | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-06-01 | T-06-06 | The starting binary is the project's own published v0.14.0 asset, fetched over TLS from its Release and verified by CI when it was published; the artifact under test (v0.15.0) is verified in-process by sigstore-go against the pinned identity before any swap | plan-time disposition (06-02 threat_model), plans approved by the maintainer via #88 | 2026-10-17 |
| AR-06-02 | T-06-SC | No package-manager install of a project dependency happened in either plan; the devbox needed no `ca-certificates` install | plan-time disposition (06-01 and 06-02 threat_model) | 2026-10-17 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-17 | 10 | 10 | 0 | /gsd-secure-phase 6 (orchestrator, L1 grep-depth; auditor not spawned: threats_open 0, register authored at plan time, ASVS 1) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-17
