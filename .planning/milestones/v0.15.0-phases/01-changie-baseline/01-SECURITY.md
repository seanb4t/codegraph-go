---
phase: "1"
slug: "changie-baseline"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-25"
verified: "2026-09-25"
---

# Phase 1 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| proxy.golang.org → local/CI build | Third-party changie v1.26.0 and 63 transitive modules are compiled into a build tool | Go module source, checksum-verified against go.tool-changie.sum / sumdb |
| CHANGELOG.md → .changes/v*.md | The published release history is moved into seed files that changie re-renders from | Public release notes (verbatim byte slices) |
| CI runner → third-party build tool | ci.yml's `test` job (`contents: read`, no secrets) compiles and executes changie | None beyond the repository checkout |
| check:changie scratch dir → contributor working tree | The guard runs a tool that writes files; those writes must never reach the real tree | Scratch fragments and a scratch CHANGELOG copy only |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-01-SC | Tampering | go.tool-changie.mod / go.tool-changie.sum; changie binary built and executed in CI | high | mitigate | Exact pin `github.com/miniscruff/changie v1.26.0` (`go.tool-changie.mod:90`); `GOWORK=off go mod verify -modfile=go.tool-changie.mod` → `all modules verified`; default `-mod=readonly`; CI step body is exactly `run: task check:changie` (`ci.yml:212`), no marketplace action or download; changie is in the `task vuln` nine-binary scan loop (`Taskfile.yml:1913`), pinned by `TestChangieBinaryInToolVulnScan`; Package Legitimacy Audit in 01-RESEARCH.md OK | closed |
| T-01-01 | Elevation of Privilege | root go.mod | medium | mitigate | `github.com/miniscruff/changie` listed in `forbiddenToolPackages` (`internal/upgrade/taskfile_shape_test.go:138`); `TestToolModfilesRemainIsolated` fails if the root module ever requires or declares the tool | closed |
| T-01-02 | Tampering | .changes/v*.md seeds versus CHANGELOG.md | medium | mitigate | `TestChangieVersionSeedsMatchChangelog` (set equality both directions, floor 14); `task changie -- merge --dry-run \| cmp - CHANGELOG.md` exits 0; `git diff --quiet main -- CHANGELOG.md` exits 0 (verified 2026-09-25) | closed |
| T-01-03 | Tampering | .changie.yaml vocabulary (PR requirement, kinds, newlines) | medium | mitigate | `TestChangieConfigShape` asserts `custom[0]` keys are exactly `{key, minInt, type}` with `minInt: 1` and `newlines` exactly `{afterChangelogHeader}` (`changie_shape_test.go:257-272`); making PR optional is a failing test (D-11) | closed |
| T-01-04 | Tampering | fragment body text rendered into CHANGELOG.md | low | accept | Fragments land only through reviewed PRs; changie inserts `{{.Body}}` as data, never re-parsed as a template; output is markdown, never executed | closed — accepted (AR-01) |
| T-01-05 | Spoofing | `task changie` argument passthrough | low | accept | Developer/CI tool with no secrets in scope; go-task re-quotes `CLI_ARGS`, proven by the multi-word body passthrough check in 01-01 Task 2 | closed — accepted (AR-02) |
| T-01-06 | Tampering | check:changie versus the contributor working tree | medium | mitigate | `scratch=$(mktemp -d)` with `trap 'rm -rf "${scratch}"' EXIT`; before/after `cksum` snapshot of `.changie.yaml`, `.changes/`, `CHANGELOG.md` fails the run on any change; real fragments never copied or written back (`Taskfile.yml` check:changie body) | closed |
| T-01-07 | Repudiation | check:changie legs (false assurance from exit codes) | medium | mitigate | Seed count printed before comparing with floor 14 and a named `::error::` on shortfall; every refusal asserts changie's stderr text, non-zero exit, and unchanged fragment count; executed-leg counter must reach 11; RED families (a)–(d) committed in 01-MUTATION-LOG.md (4 families) | closed |
| T-01-08 | Elevation of Privilege | ci.yml test job | low | accept | Workflow-level `permissions:` unchanged (`contents: read`); no secrets, no new `uses:`; run body exactly `task check:changie`; asserted by `TestChangieCheckWiredIntoCI` and `task lint:actions` | closed — accepted (AR-03) |
| T-01-09 | Denial of Service | CI time spent building changie | low | accept | Existing Go module/build cache action covers the build; seconds when warm; 64-module standalone graph | closed — accepted (AR-04) |
| T-01-10 | Tampering | scratch path handling in the Taskfile body | low | mitigate | Every expansion double-quoted (`"${scratch}"`, `"${work}"`), paths originate only from `mktemp -d`, `set -euo pipefail`, no user-controlled path input | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-01 | T-01-04 | Fragment bodies are reviewed-PR content rendered as markdown data; no template evaluation or execution path exists | gsd-secure-phase (auto mode, plan-time disposition) | 2026-09-25 |
| AR-02 | T-01-05 | No secrets or privileged actions reachable through the wrapper; go-task re-quoting prevents argument splitting | gsd-secure-phase (auto mode, plan-time disposition) | 2026-09-25 |
| AR-03 | T-01-08 | CI job permissions, secrets, and action set are unchanged and guarded by tests | gsd-secure-phase (auto mode, plan-time disposition) | 2026-09-25 |
| AR-04 | T-01-09 | Build cost is cached and bounded; no availability impact beyond seconds | gsd-secure-phase (auto mode, plan-time disposition) | 2026-09-25 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-25 | 11 | 11 | 0 | gsd-secure-phase orchestrator (L1 grep-depth short-circuit; register authored at plan time) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-25
