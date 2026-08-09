---
phase: 2
slug: apple-signing-notarization
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-09
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test (Go stdlib testing) + Taskfile targets |
| **Config file** | Taskfile.yml |
| **Quick run command** | `task test:unit` |
| **Full suite command** | `task test` |
| **Estimated runtime** | ~60 seconds (unit); full suite longer — `test:integration` / `test:wireoracle` / `test:daemon` each shell out to `go build` |

---

## Sampling Rate

- **After every task commit:** Run `task test:unit`
- **After every plan wave:** Run `task test`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 02-01 T1 | 02-01 | 1 | SIGN-03, SIGN-02 | T-02-01, T-02-02 | Cannot assess without a confirmed quarantine attribute; unclassifiable verdict is fatal; a missing GitHub digest is fatal on the GREEN path rather than failing open | unit (config shape) | `go test ./internal/upgrade/ -run TestVerifyGatekeeperDeclaresNamedPreconditions -v` | ❌ created by this task | ⬜ pending |
| 02-01 T2 | 02-01 | 1 | SIGN-03 | T-02-01, T-02-02, T-02-03 | RED baseline observed against the real published asset via `spctl -a -vv -t install` (D-19), verdict by exit status; positive control confirms the oracle can reach green | manual (checkpoint) | — real macOS + Gatekeeper; see Manual-Only below | n/a | ⬜ pending |
| 02-01 T3 | 02-01 | 1 | SIGN-03 | T-02-02 | Recorded evidence names its source asset and its measured assumption verdicts | unit + evidence grep | `go test ./internal/upgrade/ -run TestVerifyGatekeeperDeclaresNamedPreconditions -v && rg -c 'GATEKEEPER-EVIDENCE' .planning/phases/02-apple-signing-notarization/02-EVIDENCE.md` | ❌ created by this task | ⬜ pending |
| 02-02 T1 | 02-02 | 2 | SIGN-01, SIGN-04 | T-02-05, T-02-06 | Config-shape properties are asserted by resolving templates, never by literal match | unit (TDD, RED first) | `go test ./internal/upgrade/ -run 'TestSignsSidecarMatchesUpgradeContract\|TestNotarizeMacos\|TestParseGoreleaserSigns_NoSignsBlockIsError\|TestParseGoreleaserNotarize_NoNotarizeBlockIsError' -v` (now includes TestNotarizeMacosHasExactlyOneEntry) | ❌ created by this task | ⬜ pending |
| 02-02 T2 | 02-02 | 2 | SIGN-01, SIGN-04 | T-02-05, T-02-06, T-02-07 | cosign signs post-notarize bytes; the notarize pipe cannot silently no-op | unit + config check | same as 02-02 T1, plus `task check:goreleaser` | ✅ from T1 | ⬜ pending |
| 02-02 T3 | 02-02 | 2 | SIGN-01 | T-02-06, T-02-08 | Every goreleaser caller carries a recorded notarize-reachability verdict | unit (workflow/taskfile shape) | `go test ./internal/upgrade/ -run 'TestWorkflowRunBodiesInvokeTask\|TestGoreleaserPinParity\|TestCheckCrossMatchesGoreleaserTargets' -v && task check:goreleaser` | ✅ existing | ⬜ pending |
| 02-03 T1 | 02-03 | 2 | SIGN-02 | T-02-09 | A bad binary override aborts before any test runs; no silent rebuild | unit (TDD table) | `go test ./test/integration/ -run TestResolveTestBinPath -v` | ❌ created by this task | ⬜ pending |
| 02-03 T2 | 02-03 | 2 | SIGN-02 | T-02-09, T-02-10 | Same contract in the wire-oracle harness; scope verdict stated, not assumed | unit (TDD table) | `go test ./test/wireoracle/ -run TestResolveTestBinPath -v && go test ./test/integration/ -run TestResolveTestBinPath -v` | ❌ created by this task | ⬜ pending |
| 02-04 T1 | 02-04 | 3 | SIGN-01, SIGN-04 | T-02-11, T-02-12, T-02-13 | Rehearsal halts by name on any missing credential; baseline double-build asserts determinism under its own `BASELINE-NONDETERMINISTIC` label before any subject comparison | unit (taskfile shape) | `go test ./internal/upgrade/ -run 'TestRehearseNotarizeDeclaresCredentialPreconditions\|TestVerifyGatekeeperDeclaresNamedPreconditions' -v` | ❌ created by this task | ⬜ pending |
| 02-04 T2 | 02-04 | 3 | SIGN-01, SIGN-04 | T-02-11, T-02-12, T-02-13 | Real Apple round-trip and the mis-order mutation, both observed | manual (checkpoint) | — real Developer ID credentials; see Manual-Only below | n/a | ⬜ pending |
| 02-04 T3 | 02-04 | 3 | SIGN-04 | T-02-13 | Divergent and convergent hash sets both recorded; committed config unchanged | unit + evidence grep | `go test ./internal/upgrade/ -run TestRehearseNotarizeDeclaresCredentialPreconditions -v && rg -c 'SIGN04-EVIDENCE' .planning/phases/02-apple-signing-notarization/02-EVIDENCE.md` | ✅ from T1 | ⬜ pending |
| 02-05 T1 | 02-05 | 3 | SIGN-02, SIGN-03 | T-02-14, T-02-15 | Documented reproduction cannot produce a misleading pass; guarantee carries its limitation | doc assertion (grep) | `rg -c 'not stapled' docs/RELEASE.md && rg -c 'xattr -p com.apple.quarantine' docs/RELEASE.md && rg -c 'spctl -a -vv -t install' docs/RELEASE.md` | ✅ file exists | ⬜ pending |
| 02-05 T2 | 02-05 | 3 | SIGN-02 | T-02-15 | No claim in the document exceeds recorded evidence | doc assertion (grep) | `rg -c 'not stapled' docs/RELEASE.md && rg -n 'v0.5.1' docs/RELEASE.md` | ✅ file exists | ⬜ pending |
| 02-06 T1 | 02-06 | 4 | SIGN-01 | T-02-16, T-02-17 | Credentials scoped to one job; OIDC scope not widened; no PR-triggerable exposure | unit (workflow shape) + lint | `go test ./internal/upgrade/ -run 'TestAppleSecretsScopedToSingleReleaseJob\|TestOIDCWriteScopedToSingleGoreleaserJob\|TestWorkflowRunBodiesInvokeTask' -v && task lint:actionlint` | ❌ test created by this task | ⬜ pending |
| 02-06 T2 | 02-06 | 4 | SIGN-02 | T-02-18 | Event-aware guard verbatim; gate cannot pass by skipping | lint + unit | `task lint:actionlint && go test ./internal/upgrade/ -run 'TestWorkflowRunBodiesInvokeTask\|TestOIDCWriteScopedToSingleGoreleaserJob' -v` | ✅ existing | ⬜ pending |
| 02-06 T3 | 02-06 | 4 | SIGN-02 | T-02-18, T-02-19, T-02-24 | cosign-verifies-before-chmod; checkout pinned to the resolved tag so tests match the binary; aborts on a bad download; fails if zero tests ran (counted from `-json` events) | unit (taskfile shape) + lint | `go test ./internal/upgrade/ -run 'TestVerifyNotarizedSuiteDeclaresNamedPreconditions\|TestVerifyGatekeeperDeclaresNamedPreconditions\|TestRehearseNotarizeDeclaresCredentialPreconditions' -v && task lint:actionlint` | ❌ test created by this task | ⬜ pending |
| 02-07 T1 | 02-07 | 5 | SIGN-04 | T-02-22 | `final_local_sha256` is observable and honestly labelled; the recording step FAILS LOUDLY on absent metadata | full gate sweep | `go test ./internal/upgrade/ -v && task check:goreleaser && task lint:actionlint && task test:unit` | ✅ existing | ⬜ pending |
| 02-07 T2 | 02-07 | 5 | SIGN-01, SIGN-02, SIGN-04 | T-02-22 | Notarize pipe observed submitting, not skipping; published asset assessed accepted | manual (checkpoint) | — irreversible publish; see Manual-Only below | n/a | ⬜ pending |
| 02-07 T3 | 02-07 | 5 | SIGN-02, SIGN-03, SIGN-04 | T-02-22, T-02-23 | Every recorded value traces to the re-downloaded published asset | unit + evidence grep | `rg -c 'SIGN04-PUBLISH-EVIDENCE\|GATEKEEPER-EVIDENCE' .planning/phases/02-apple-signing-notarization/02-EVIDENCE.md && go test ./internal/upgrade/ -v && task test:unit` | ✅ from T1 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

No separate Wave 0 is needed: every task in this phase carries a real
`<automated>` command, and no task declares a MISSING verification. Each of the
four gaps the research flagged is created by the task that first needs it, under
TDD where the subject has defined I/O.

- [x] Binary-path seam for the real-binary harnesses — **scoped into plan 02-03,
      wave 2**, as a TDD task with a pure resolver and a table test, not a Wave 0
      scaffold. It is created and verified in the same task that needs it, and the
      transplanted precedent is `internal/upgrade/verify_release_e2e_test.go`'s
      environment-override resolver — with its skip-when-absent policy deliberately
      inverted to abort-when-bad.
- [x] `verify:gatekeeper` — **plan 02-01, wave 1** (the phase's tracer). Its
      automated verification is a new Taskfile shape test created in the same task.
- [x] `release:rehearse-notarize` — **plan 02-04, wave 3**, with its own shape test
      created in the same task.
- [x] Post-release verification jobs — **plan 02-06, wave 4**, verified by
      actionlint plus the existing and new workflow-shape tests.

**Sampling continuity check:** no three consecutive tasks lack an automated
verify. The three checkpoint tasks (02-01 T2, 02-04 T2, 02-07 T2) are each
immediately followed by a task whose automated command asserts the recorded
result, and each is separated from the others by at least two automated tasks.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Local notarization rehearsal against Apple's service, incl. the `cosign verify-blob` subject determination and the single-arch mis-order mutation | SIGN-01, SIGN-04 | D-08 locks rehearsal to the maintainer's Mac; requires a Developer ID Application certificate and App Store Connect API key that CI-triggerable workflows must not be able to reach (a `pull_request`-reachable credential is a security regression, not a fix) | Run the guarded maintainer-only Taskfile target the planner adds; it must hard-fail BY NAME on a missing cert or API key before any network round-trip (D-09) |
| `spctl -a -vv -t install` RED baseline on the v0.5.1 published darwin asset (D-19; verdict by exit status 3) | SIGN-03 | Requires real macOS with Gatekeeper; `spctl` has no CI-usable equivalent, and the quarantine xattr must be confirmed present via `xattr -p` first or the result is not evidence | Download the v0.5.1 darwin asset, set `com.apple.quarantine`, confirm with `xattr -p`, run `spctl`, record `rejected` before any green run |
| `syspolicy_check distribution` on the notarized asset — NON-GATING observation only | SIGN-02 | D-19 CLOSED its semantics: `Notary Ticket Missing`, Severity Fatal, exit 70 for anything unstapled, which D-16 puts permanently out of scope. Recorded, never gated — a gate on it could never pass | Run it, record the output verbatim under a NON-GATING label; a Fatal verdict is the expected result and must not fail any target or job |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies — 16 of 19 tasks
      carry an automated command; the 3 checkpoints are `type="checkpoint:*"` and
      are exempt, each followed by an automated task asserting its recorded result
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references — no task declares MISSING; each gap is
      created by the task that needs it (see Wave 0 Requirements above)
- [x] No watch-mode flags — every command is a single-shot `go test` / `task`
      invocation
- [x] Feedback latency < 60s for the per-task commands (`go test ./internal/upgrade/`
      and the two harness resolver tables). The phase-gate commands (`task test:unit`,
      the real rehearsal, the real release) are deliberately slower and are declared
      as wave-merge / phase-gate sampling, not per-task
- [ ] `nyquist_compliant: true` — set by `/gsd-validate-phase` after execution, not
      by the planner

**Approval:** populated by `/gsd-plan-phase 2` on 2026-08-09; awaiting execution
