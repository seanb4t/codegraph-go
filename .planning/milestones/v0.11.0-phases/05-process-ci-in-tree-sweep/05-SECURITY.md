---
phase: 05
slug: process-ci-in-tree-sweep
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on (high)
threats_open: 0
asvs_level: 1
register_authored_at_plan_time: true
threats_total: 35
threats_closed: 35
threats_mitigated: 31
threats_accepted: 4
threats_critical: 1
threats_high: 18
created: 2026-08-16
verified: 2026-08-16
audit_mode: retroactive-L1-shortcircuit
---

# Phase 05 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Audit mode:** retroactive. **All 8** PLAN files carry a threat model — 2 in the `<threat_model>`
XML form, 6 as `## Threat Model` markdown headings — so `register_authored_at_plan_time: true`. With
`asvs_level: 1` and `threats_open: 0`, the L1 short-circuit applies; verified against the tree at
commit `615795b`.

This is the milestone's largest register (35 distinct threats across 8 plans) and reflects the
phase's breadth: it swept issue/PR templates, workflow display names, in-tree code comments, schema
doc comments, the MCP archtest, the agents marker contract, and the behavioral corpus — while
deleting `codegraph migrate` outright under maintainer ruling D-04.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| workflow job `name:` → GitHub ruleset 20157557 | Seven required-status-check contexts are bound **by name**; renaming one silently unbinds branch protection | Check context names |
| `go.mod`/`go.sum` → linked binary | Removing `internal/migrate` must also remove its sole-use `modernc.org/sqlite` dependency without breaking the link | Dependency graph |
| graphstore API → migrate call sites | Removing the migration record API must leave no orphaned fake or interface method | Type surface |
| in-tree comment sweep → product surface | The sweep removes framing but must not de-list TypeScript-the-indexed-language | Capability listings |
| `markerBegin` bytes → previously installed git hooks | A drifted begin marker orphans every hook a prior install wrote, leaving executables nothing can find or remove | Hook marker bytes |
| agents marker constants → installed agent files | Marker constant bodies are a byte contract with files already on disk | Marker bytes |
| comment-only plan → frozen golden oracle | An accidental re-freeze would let a comment-only plan author its own oracle | Golden bytes |
| census result → phase completion claim | A census reporting a zero it did not earn spoofs completion — the precise failure that created this phase's gap-closure plans | Coverage claim |
| `/gsd-ship` gate → `WINDOWS.md` ledger | The gate blocks on `open_count > 0`, creating direct pressure to close genuine open entries to reach zero | Ledger integrity |

---

## Threat Register

### Critical

| Threat ID | Plan | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|------|----------|-----------|----------|-------------|------------|--------|
| T-05-09 | 05-03 | Tampering | required-check job-name edit breaking the ruleset binding | **critical** | mitigate | The 7 required names stay byte-identical; only *free* framing-bearing display names change. **Verified by execution:** `TestRequiredCheckNamesPreserved` **PASS**, paired with `TestRequiredCheckNamesPreserved_ZeroJobsIsError` **PASS** — a non-vacuity guard on the guard itself. The fixture pins all seven contexts against `gh api repos/seanb4t/codegraph-go/rulesets/20157557`, re-verified live 2026-08-01, and carries an explicit warning that a stale fixture would make the guard *assert the wrong thing rather than fail loudly* | closed |

The seven pinned contexts: `test`, `govulncheck (DIST-03, blocking)`,
`reproducibility (double-build hash-diff, DIST-04)`, `perf regression gate (PERF-02, INDX-06)`,
`actionlint (workflow static analysis)`, `goreleaser check (config validation, DIST-01)`, `pr-title`.

### High

| Threat ID | Plan | Component | Disposition | Mitigation & verification | Status |
|-----------|------|-----------|-------------|---------------------------|--------|
| T-05-01 | 05-01 | `modernc` left in `go.mod` after migrate removal | mitigate | `go mod tidy` + empty scan; build must still link. **Verified:** 0 hits in `go.mod` **and** `go.sum`; `go build ./...` exit 0 | closed |
| T-05-02 | 05-01 | ts-* fixtures deleted, subtests left behind | mitigate | Fixtures + subtests in one edit. **Verified:** 0 `ts-*` files under `testdata/`; golden suite 79 PASS | closed |
| T-05-03 | 05-01 | cursor API removed but a fake remains | mitigate | Positive scan expecting 0 aggregate. **Verified:** 0 hits for `GetMigration\|PutMigration\|MigrationRecord` | closed |
| T-05-08 | 05-02 | removing the issue-approval / output-shape guard | mitigate | The `feature_request` and `enhancement` rewrites keep "output shapes consumed by agents" and the issue-first rule; verify asserts the blocking step survives | closed |
| T-05-13 | 05-04 | `internal/query` sweep (CODE-01) | mitigate | Census-driven term-by-term, never find-replace; each row lands with a recorded reason. **Verified:** query + capability **398 PASS** | closed |
| T-05-14 | 05-04 | synthetic rename dropping a call site | mitigate | Definition and every call site in one commit. **Verified:** same 398 PASS, golden suite green | closed |
| T-05-16 | 05-04 | the sweep de-listing a supported language | mitigate | File-by-file hand edits (D-02); `tsextract`/registry/matrix never de-listed. **Verified:** `internal/indexer/tsextract` present, 23 `"typescript"` registry hits, 3 matrix rows | closed |
| T-05-17 | 05-05 | schema doc comments drift (proto/pb mislock) | mitigate | Both files edited in one commit as paired side-by-side changes with a diff check | closed |
| T-05-18 | 05-05 | mcp archtest `packages.Load` edited instead of prose | mitigate | Do-not-touch instruction; verify greps the literal out unchanged. **Verified:** mcp + archtest packages ok | closed |
| T-05-19 | 05-05 | agents byte contract mutated | mitigate | Only comments change; marker constant bodies stay byte-identical. **Verified:** `internal/agents` **207 PASS** | closed |
| T-05-20 | 05-05 | the sweep de-listing a capability | mitigate | Product surface untouched — no name or listing deletion | closed |
| T-05-21 | 05-06 | corpus reword **without** re-freeze | mitigate | The two tasks are bound: task 2 runs `task golden:regen` and verifies the log; a comment-only change without the goldens fails the check. **Verified:** 26/26, 30/30 | closed |
| T-05-22 | 05-06 | a full golden re-baseline slipping in | mitigate | Verify checks that only the four-src + two-json arrow is in the diff; **STOP** on anything else | closed |
| T-05-07-02 | 05-07 | `markerBegin` / `markerBeginBytes` drift orphaning installed hooks | mitigate | Marker bytes asserted unchanged. **Verified:** 29 marker/hook tests PASS | closed |
| T-05-08-02 | 05-08 | frozen oracle fixtures accidentally re-frozen | mitigate | A re-freeze would make a comment-only plan author its own oracle — the exact anti-pattern the milestone forbids. **Verified:** 26/26 goldens byte-valid, scenario total 30/30 | closed |
| T-05-08-03 | 05-08 | the acceptance census spoofing a zero it did not earn | mitigate | Three independent assertions rather than one grep. This is the precise failure that produced this gap-closure plan | closed |
| T-05-08-04 | 05-08 | `WINDOWS.md` ledger integrity under ship-gate pressure | mitigate | `/gsd-ship` blocks on `open_count > 0`, creating direct pressure to close entries 12, 13 and 16 to reach zero. **Verified:** ledger still reports **3 open** — the pressure was not yielded to | closed |
| T-05-08-05 | 05-08 | deleting TypeScript-the-indexed-language to satisfy a framing census | mitigate | A capability regression dressed as a sweep. **Verified:** same product-surface evidence as T-05-16 | closed |

### Medium

| Threat ID | Plan | Component | Disposition | Status |
|-----------|------|-----------|-------------|--------|
| T-05-04 | 05-01 | `go test ./...` passing while the golden suite is broken (`./...` excludes `testdata`) | mitigate | closed |
| T-05-05 | 05-01 | same-change re-freeze interactions | mitigate | closed |
| T-05-06 | 05-02 | issue/PR template reword (PROC-01/02) | mitigate | closed |
| T-05-07 | 05-02 | pr-template-policy gate keyed on preserved headings | mitigate | closed |
| T-05-10 | 05-03 | shell `run:`-body accident or JS-parse break | mitigate | closed |
| T-05-11 | 05-03 | accidentally decoupling the bench runner (Phase-6 scope) | mitigate | closed |
| T-05-12 | 05-03 | the sweep degenerating into a regex queue | mitigate | closed |
| T-05-15 | 05-04 | matrix comment / DOC-cell drift | mitigate | closed |
| T-05-23 | 05-06 | accidental case-caption change | mitigate | closed |
| T-05-07-01 | 05-07 | `serve.go` D-12/D-13 disabled-message format literal (asserted by `serve_test.go:160`) | mitigate | closed |
| T-05-07-04 | 05-07 | `WINDOWS.md` ledger completeness — a silently-swept out-of-scope find reproduces the failure this plan closes | mitigate | closed |
| T-05-08-01 | 05-08 | `nodeMultiDefHeaderPattern` regex pinning the NODE-01/02 header template | mitigate | closed |

### Accepted

| Threat ID | Plan(s) | Component | Status |
|-----------|---------|-----------|--------|
| T-05-SC | 05-02…05-06 | npm/pip/cargo installs | closed |
| T-05-07-03 | 05-07 | `files.go:90` `--dir` help text (information disclosure) | closed |
| T-05-07-SC | 05-07 | npm/pip/cargo installs | closed |
| T-05-08-SC | 05-08 | npm/pip/cargo installs | closed |

**Totals:** 35 distinct threats — 31 mitigated, 4 accepted, **0 open**. At/above the block threshold: 1 critical + 18 high, all verified.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-05-01 | T-05-SC, T-05-07-SC, T-05-08-SC | No package-manager install task exists in any of these plans; no dependency is added, removed or upgraded, and `go.mod`/`go.sum` are not in `files_modified`. The Package Legitimacy Gate is therefore N/A. Recorded independently in each plan rather than omitted. | plan authors (05-02…05-08) | 2026-08-15 |
| R-05-02 | T-05-07-03 | `internal/cli/files.go:90`'s `--dir` help text is already public output; deleting a parenthetical from it reveals nothing. No test or golden asserts the string — **verified repo-wide** before accepting, not assumed. | plan author (05-07) | 2026-08-15 |

*Note: `T-05-01`'s dependency removal is the inverse of a supply-chain addition — it **removes**
`modernc.org/sqlite` from the module graph. A reduction in dependency surface does not trigger the
Package Legitimacy Gate, but it is recorded here because it is the milestone's only `go.mod` change.*

---

## Threat Flags from Execution

No Phase 5 summary carries a `## Threat Flags` section. `05-VERIFICATION.md` records
`status: passed`, 5/5 must-haves, and a re-verification that closed CODE-01 across all 10 previously
cited `file:line` locations (first pass was `gaps_found` 4/5).

**The `WINDOWS.md` ledger is this phase's substantive flag record**: 14 fixed, 2 waived
(adjudicated D-02 product truth), **3 open** (entries 12, 13, 16). Entries 12 and 13 are
PRE-EXISTING and unrelated to the sweep. Entry 16 is now **substantively resolved** — its two cited
strings are absent from `internal/bench/rss.go` and `tools/bench/runner/main.go`, closed by Phase 6's
BENCH-02 exactly as the entry predicted — but its ledger row still reads `open`. That is bookkeeping
lag, not an open threat, and it is recorded in `.planning/v0.11.0-MILESTONE-AUDIT.md` as tech debt.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-16 | 35 | 35 | 0 | `/gsd-secure-phase 5` (orchestrator, L1 short-circuit — no auditor subagent required) |

**Verification method:** L1 grep depth against the tree at commit `615795b`, plus ~660 executed
tests across `internal/upgrade`, `internal/agents`, `internal/githooks`, `internal/cli`,
`internal/query`, `internal/indexer/capability`, `internal/mcp` and `testdata/golden`.

**One phantom command, on the critical threat.** `T-05-09`'s mitigation names
`TestWorkflowRequiredChecks`, which matches **zero tests** — `go test -run` exits 0 on no match, so
the named guard would have read green while running nothing. The real guards are
`TestRequiredCheckNamesPreserved` and `TestRequiredCheckNamesPreserved_ZeroJobsIsError`
(`internal/upgrade/taskfile_shape_test.go:704,748`), both executed and both PASS. The plan named a
test that does not exist; the protection it describes does. This was only visible by counting
`--- PASS` lines rather than trusting exit status.

**Observation worth carrying forward:** this repository contains **49** non-vacuity guards — tests
whose only job is to prove another guard can fail (`*IsNotVacuous`, `*ZeroJobsIsError`,
`*FailLoudly`, `*IsError`), 19 of them in `taskfile_shape_test.go` alone. Rule `84d1gfpywd` is not
aspirational here; it is institutionalized in the test suite.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-16
