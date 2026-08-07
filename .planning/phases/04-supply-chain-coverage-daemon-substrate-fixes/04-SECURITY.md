---
phase: 04
slug: supply-chain-coverage-daemon-substrate-fixes
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-06
---

# Phase 04 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register origin: **authored at plan time**. All 3 plans (`04-01` … `04-03`) carry a
parseable `<threat_model>` block, so this audit verifies that each declared mitigation
is present in the implementation — it does not retroactively scan for new threats.
No summary emitted a `## Threat Flags` section; the register was verified against source
and CI config directly rather than against an executor-declared flag list. Row count
reconciles to **13 unique** rows: `T-04-01` … `T-04-12` plus `T-04-SC` (which appears in
all three plans).

**Compound-category note.** `T-04-01` and `T-04-04` carry *two* STRIDE categories each
(`Tampering, Elevation of Privilege` and `Tampering, Denial of Service`). A
single-category regex silently drops both rows — and both are `high` severity, i.e.
exactly the rows the `block_on: high` gate exists to catch. Preserved verbatim below.

**Threat-ID scope note.** `T-04-*` here means *phase* 04. `01-SECURITY.md` namespaces by
*plan* number and also contains a `T-04-02`, denoting an unrelated threat. Do not merge
threat IDs across phase files.

**This audit changed two dispositions.** `T-04-01` moved `mitigate` → `accept` and
`T-04-12` moved `transfer` → `accept`, in both cases because the implementation did not
match what the register declared. See the audit notes for why this is a correction to the
record rather than a relaxation of a bar.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| third-party module code → CI runner | `task` and `goreleaser` execute on a runner holding repository write access and release secrets (D-04); their transitive closure is arbitrary third-party code | Executable code, secrets in scope |
| vulnerability database (`vuln.go.dev`) → CI job | The scan's verdict depends on a network fetch the job does not control | Advisory records, exit status |
| repository tree → main-module build graph | A source file committed to this repo can be pulled into the module every other gate and every release binary is built from | Go package graph membership |
| repository workflow files → release path | `release.yml`'s env block selects the binary that builds, signs and publishes every released artifact | Tool version pins |
| in-repo assertions → out-of-repo branch protection | GitHub ruleset 20157557 decides which checks actually gate merges; **nothing in this repository can enforce that** | Required-context names |
| test harness → daemon process state | Tests drive `Daemon.Run`, which acquires the repo lockfile and writes the global daemon registry under the user's home directory; a seam pointed at the wrong place would touch real state | Lockfile, registry entries |
| spawned goroutine → package-level variable | An orphaned goroutine outliving its test reads variables a later, unrelated test writes | Package-level seam values |
| daemon → single-writer store invariant | `Run` releases the lock only after every flush has joined (INDX-05, SYNC-06); the tests asserting that are the only enforcement | Store write ordering |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-04-01 | Tampering, Elevation of Privilege | 357 modules linked into `task`/`goreleaser`/`govulncheck`/`actionlint`, executed with repo write access and secrets | high | **accept** (was `mitigate`) | See AR-04-01. Entry-point coverage **is** complete — all 4 tool binaries across both modfiles (`go.tool.mod:36-40`, `go.tool-lint.mod:24`) — but the control is **detect-only**: `Taskfile.yml:204-224` has no non-zero exit path and reports every exit-3 as `::warning::`, and `ci.yml:229` names the step "advisory — reports, never fails the build" | closed |
| T-04-02 | Tampering | `-mode=binary` symbol-level analysis (no call graph) | medium | accept | See AR-04-02. Limitation stated verbatim at `Taskfile.yml:172-178` ("symbol-level, NOT call-graph-aware… a materially weaker guarantee") and `ci.yml:187-190`. DIST-03 source-mode gate untouched: `git diff 9d82f3a..HEAD -- .github/workflows/ci.yml \| rg '^-'` → zero removed lines (pure insertion); `govulncheck (DIST-03, blocking)` confirmed live in ruleset 20157557's required contexts | closed |
| T-04-03 | Tampering | scan exit status other than 0 or 3 reported as a pass | medium | mitigate | **Partially met.** Declared: "classifies any non-0/non-3 status as FAIL." Found: classified *distinctly* (`Taskfile.yml:218`, `SCAN ERROR … never folded into CLEAN`) but never fails. Compensating control verified: `vuln:selftest` requires exit **exactly 3** (`Taskfile.yml:255-256`), so a total `vuln.go.dev` outage does redden the job. Residual: a scan error on one tool binary while the selftest succeeds still presents green with only an annotation | open — below high threshold (non-blocking) |
| T-04-04 | Tampering, Denial of Service | `testdata/vulnredpoc/main.go` entering the main-module package graph | high | mitigate | Ran live: `go list ./...` = 51 packages, zero `vulnredpoc`/`testdata` matches; `go list -deps ./...` contains **no** `openpgp` package. Built only by explicit path (`Taskfile.yml:245`, `-modfile=go.tool.mod ./testdata/vulnredpoc`). `git diff 9d82f3a..HEAD -- go.mod go.sum go.tool.mod` = empty. Self-detecting residual: relocating it out of `testdata/` would immediately redden the blocking DIST-03 required check | closed |
| T-04-05 | Spoofing | a future edit leaving a gate-shaped job that cannot fail | medium | mitigate | **Partially met — proven by mutation.** Stance text present at all four sites (`ci.yml:209`, `:229`, `:232`; `Taskfile.yml:150`), but the assertion that all three keep saying it is largely vacuous. Three mutations on a disposable `git archive` copy all stayed **GREEN**: (a) flipping the Taskfile desc `ADVISORY`→`BLOCKING`, (b) adding `continue-on-error: true` to `tool-vuln` — the literal second clause of this threat's own text, (c) deleting the `task vuln:selftest` step. Only whole-job deletion and job-name edits are caught | open — below high threshold (non-blocking) |
| T-04-06 | Tampering | deadline budget masking a teardown hang | medium | mitigate | Clamp present and documented: `internal/daemon/testbudget_test.go:84` (`testBudgetMaxScale = 40.0`), enforced at `:124-125`. Assertions preserved — `git show 7096b72 -- internal/daemon/ \| rg '^-' \| rg 't\.(Fatal\|Error)'` → no matches (zero assertion lines removed). `04-02-SUMMARY.md` records the after-measurement under the same unfiltered load, with residual timeouts reported rather than hidden | closed |
| T-04-07 | Tampering | concurrent read/write of `getppid` / `registryDir` package seams | low | mitigate | Join-before-restore ordering implemented: `testbudget_test.go:170` `joinDaemonRun` registers a `t.Cleanup`; seam restores registered *earlier*, so LIFO runs the join first (`watchdog_test.go:21`→`:42`, `:73`→`:93`; `daemon_test.go:307`→`:330`). Verified empirically, not structurally: `go test -race -count=1 ./internal/daemon/` → ok 73.1 s, zero `DATA RACE` | closed |
| T-04-08 | Repudiation | orphaned goroutine writing real `~/.codegraph/daemons/` or repo lockfile | low | mitigate | 18 `joinDaemonRun` sites cover every `Daemon.Run`/`RunWithRetry` spawn (13 in `daemon_test.go`, 3 in `soak_test.go`, 2 in `watchdog_test.go`) plus both watchdog `stop()` joins. `registryDir` seam appears only in `registry_test.go:20`, `:143`, which spawns no `Run`. Remaining unjoined spawns are non-daemon (`stop_test.go:53`, `lock_test.go:192`) | closed |
| T-04-09 | Tampering | GoReleaser pin divergence into the release path | medium | mitigate | `go.tool.mod:234` = `v2.17.1`; `.github/workflows/release.yml:61` = `"v2.17.1"`. `TestGoreleaserPinParity` (`internal/upgrade/taskfile_shape_test.go:762-784`) passes and is **proven fail-capable**: mutating the release.yml pin to v2.17.0 in a disposable copy → FAIL at `:782` | closed |
| T-04-10 | Spoofing | gate-shaped job whose stance is deleted from name/description | high | mitigate | `TestGateStancesStated` (`taskfile_shape_test.go:804-870`) runs inside the `test` job, which **is** a live required check. Proven RED, not merely wired up: job-name stance flipped advisory→blocking → FAIL `:862` + `:868`; stance word deleted from job name → FAIL `:839`; whole `tool-vuln` job deleted → FAIL `:1365` via the `inScopeJobs` binding (`:114`). Residual noted under T-04-05 | closed |
| T-04-11 | Tampering | modification to release-signing workflow `release.yml` | medium | mitigate | `git show ea01582 -- .github/workflows/release.yml`: diff confined to the `env:` block — one literal plus its comment, no job/permission/step change. `actionlint (workflow static analysis)` (a live required check) and `goreleaser check` both run against the file in `ci.yml` | closed |
| T-04-12 | Elevation of Privilege | new blocking job never becoming a required status check | high | **accept** (was `transfer`) | See AR-04-03. Live `gh api repos/seanb4t/codegraph-go/rulesets/20157557` returns exactly 6 required contexts — `test`, `actionlint (workflow static analysis)`, `perf regression gate (PERF-02, INDX-06)`, `pr-title`, `reproducibility (double-build hash-diff, DIST-04)`, `govulncheck (DIST-03, blocking)`. `tool-vuln (VULN-01/02/03, advisory)` is **absent**, so a PR can merge with it red | closed |
| T-04-SC | Tampering | npm/pip/cargo/go dependency installs | high | mitigate | No package installed: `git diff 9d82f3a..HEAD -- go.mod go.sum go.tool.mod go.tool-lint.mod` = empty. The 614 added `.sum` lines (commit `a8c3757`, post-phase) are **all** `/go.mod h1:` module-graph hashes — zero module *content* hashes added. `golang.org/x/crypto v0.54.0` pre-existing at `go.tool.mod:378`, the exact version govulncheck reported in the live selftest run | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-04-01 | T-04-01 | **Re-dispositioned `mitigate` → `accept`.** The register declared "a symbol match exits non-zero and fails the build"; the implementation is advisory-only and always exits 0. This was a deliberate, maintainer-accepted deviation taken at 04-01's Task 1 checkpoint, because a blocking gate would have been **permanently red from its first run** — `GO-2026-5932` is unpatched with `Fixed in: N/A`. Rejected alternatives were a narrow allowlist and dropping goreleaser from scope (~322 of 357 modules). The register was simply never updated to match the decision; this entry closes that gap. Entry-point coverage remains complete (all 4 tool binaries, both modfiles), so detection is real even though enforcement is not. | seanb4t | 2026-08-06 |
| AR-04-02 | T-04-02 | `-mode=binary` is symbol-level, not call-graph-aware — a materially weaker guarantee than source mode, and the limitation is stated verbatim to operators at `Taskfile.yml:172-178` and `ci.yml:187-190` rather than left implicit. Accepted because tool binaries have no source-mode equivalent available in CI. The blocking DIST-03 source-mode gate over the main module is untouched and remains a live required check, so the weaker mode applies only to the tool closure. | seanb4t | 2026-08-06 |
| AR-04-03 | T-04-12 | **Re-dispositioned `transfer` → `accept`.** A transfer is only closed if the receiving party accepted it; the live ruleset query proves `tool-vuln (VULN-01/02/03, advisory)` is not a required context, and `04-03-SUMMARY.md:185-187` recorded it as "a maintainer should decide separately" — a deferral, not an acceptance. The maintainer now explicitly accepts that `tool-vuln` **stays unrequired**: it is an advisory job by design (AR-04-01), so requiring a job that never fails would add ceremony without adding enforcement. Residual accepted: the job's genuinely fail-capable `vuln:selftest` step is also unrequired, so a detection regression reddens the job without blocking a merge. | seanb4t | 2026-08-06 |
| AR-04-04 | GO-2026-5932 | `golang.org/x/crypto/openpgp` — **unmaintained, Fixed in: N/A**. Symbol-reachable in the `goreleaser` binary this project builds and releases with, via `goreleaser/cmd → pipe/ko → google/ko → sigstore/cosign/oci → sigstore/rekor/pkg/pki/pgp`. **Unmitigated and unpatched.** Accepted at 04-01's Task 1 checkpoint on the reasoning in AR-04-01. The advisory `task vuln` annotation is the only thing surfacing it; it is surfaced, not resolved. Re-evaluate when cosign/rekor drop the unmaintained openpgp dependency. | seanb4t | 2026-08-06 |

---

## Advisory — Unregistered Surface & Gate Quality

Neither item counts toward `threats_open`. Recorded rather than left latent, following the
precedent where the v1.0 phase-10 audit's advisory section became backlog phase 999.4.

**`requiredCheckNames` fixture drift.** `internal/upgrade/taskfile_shape_test.go:43-51`
lists `goreleaser check (config validation, DIST-01)` as a ruleset-20157557 required
context; the live ruleset does **not** include it. `TestRequiredCheckNamesPreserved` only
checks fixture → workflow-job-name, never fixture → live ruleset, so this drift is
invisible to CI. It degrades the same out-of-repo transfer channel AR-04-03 accepts.

**T-04-10's Taskfile-description leg cannot fire on a stance flip.** Two of its three legs
are demonstrably fail-capable (job-name flip, whole-job deletion). The third is a
substring-presence check over ~40 lines of prose that mentions "advisory" repeatedly, so
flipping the description's stance leaves it green. The row is closed on the two working
legs; the third is decorative. Detail under T-04-05.

**Root `SECURITY.md:59-61` overstates coverage.** It states "`govulncheck` gates every
merge" without qualifying that tool-modfile coverage is advisory-only. Worth reconciling
so the public security policy matches AR-04-01.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-06 | 13 | 11 | 2 (both below high threshold) | Claude Opus 5 (`gsd-security-auditor` subagent, `/gsd-secure-phase 04`, ASVS L1) |

### Audit 2026-08-06 — notes

Audited retroactively, after the phase was verified and the milestone archived, to close
the coverage gap left by phases 02–05 shipping without a `SECURITY.md`. The
`gsd-security-auditor` subagent was spawned rather than taking the workflow's L1
short-circuit. **This is the one phase of the four where the audit returned
`OPEN_THREATS` rather than `SECURED`.**

**Two dispositions were corrections, not relaxations.** T-04-01 and T-04-12 each declared
a stronger control than the implementation provides. In both cases the maintainer had
already made the underlying call — advisory-not-blocking at 04-01's checkpoint, and
"decide separately" for the ruleset — but the register was never reconciled to it. Moving
them to `accept` with logged rationale makes the record match reality. The alternative,
leaving them as `mitigate`/`transfer`, would have left the file asserting enforcement that
does not exist, which is the failure mode this audit exists to catch.

**Both halves of the milestone's advisory/blocking split were verified against CI config,
not prose.** *Findings advisory* holds — `Taskfile.yml:204-224` has no non-zero exit path,
and `ci.yml:229`'s step name says so. *Detection regression still fails* also holds —
`ci.yml:232` runs `task vuln:selftest`, which exits 1 on status ≠ 3 or a missing advisory
ID (`Taskfile.yml:255-256`, `:260-261`), and neither the job nor that step carries
`continue-on-error` (its only occurrence in `ci.yml` is line 288, the reproducibility arm64
leg). Run live: `task vuln:selftest` → govulncheck exited 3, named `GO-2026-5932`, reported
33 vulnerable symbols, PASS. **The detection gate is non-vacuous today.**

**Gate quality was established by mutation, not by reading.** T-04-05, T-04-09 and T-04-10
were each tested by planting a defect in a disposable `git archive` copy and observing
whether the suite went red. That is what distinguishes T-04-09 and T-04-10 (real, fail-capable)
from T-04-05 (three mutations, all green). This repo has a documented class of gates that
cannot fire; a passing test is evidence of nothing until something is broken on purpose.

**Scope note (L1).** ASVS L1 verifies that each declared mitigation is *present*. It does
not perform L2 boundary-placement analysis or L3 end-to-end trace verification.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed (2 open threats remain, both below the `high` block threshold)
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-06
