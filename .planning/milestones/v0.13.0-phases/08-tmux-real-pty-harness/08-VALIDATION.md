---
phase: "08"
slug: "tmux-real-pty-harness"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: validated
nyquist_compliant: true
wave_0_complete: true
created: "2026-09-09"
---

# Phase 08 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Seeded from `08-RESEARCH.md` § Validation Architecture, whose numbers came from
> running tmux and the real binary on this machine rather than from reasoning.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` stdlib, build-tag-gated (`//go:build tmux`) |
| **Config file** | none — the `test:tmux` Taskfile target is the config surface, following `test:integration` / `test:wireoracle` precedent |
| **Quick run command** | `task test:tmux` (lenient: prints executed and skipped counts, exits 0 without tmux) |
| **Full suite command** | `task test:tmux` with `CI=1` supplied via the job/step `env:` key (strict: asserts executed-count equals the committed constant) |
| **Estimated runtime** | ~30-60 seconds (each case spawns a real tmux session and a real binary; the stability poll is bounded) |

**Toolchain prefix (repo landmine).** Local `go build` fails on any machine whose Go is newer
than 1.26.6 because of a `cockroachdb/swiss` incompatibility. Prefix local runs with
`GOTOOLCHAIN=go1.26.6`. This already affects `test/integration`'s `TestMain` and will affect
`test/tmux`'s identically.

---

## Sampling Rate

- **After every task commit:** `GOTOOLCHAIN=go1.26.6 task test:tmux`
- **After every plan wave:** the same, strict, if tmux is available locally; otherwise defer to CI
- **Before `/gsd-verify-work`:** the `tmux-e2e` CI job green **with its executed-count assertion actually exercised** — confirm by reading the job log, not the checkmark
- **Max feedback latency:** ~60 seconds

---

## Per-Task Verification Map

Task IDs are assigned by the planner; this map binds requirements to their automated commands so
the planner can attach them. Test names are illustrative until the planner fixes them.

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 08-01 T2 | 08-01 | 1 | TTY-01 | — | Harness refuses to pass silently when tmux is absent — the skip reason is a named, always-executed test | integration | `go test -tags tmux -run TestRequireTmuxReportsSkipReasonWhenAbsent ./test/tmux/...` (`test/tmux/skip_contract_test.go`) | ✅ | ✅ green |
| 08-05 T1/T2 | 08-05 | 5 | TTY-02 | — | No assertion runs against a single unstable capture — pollUntilStable requires a readiness predicate and converges only on ready AND byte-equal | integration | `go test -tags tmux -run TestPollUntilStableDoesNotConvergeOnPreOutputFrame ./test/tmux/...` (`test/tmux/poll_contract_test.go`; watched RED e994b0a5 → GREEN b9dfc849) | ✅ | ✅ green |
| 08-01 T1 | 08-01 | 1 | TTY-03 | — | Empty registry leaks no mode-query response bytes (anchored on "no running daemons" since 08-05) | integration | `go test -tags tmux -run TestDaemonEmptyRegistryLeaksNoModeQueryBytes ./test/tmux/...` (`test/tmux/daemon_empty_test.go`; mutation family (a) watched RED) | ✅ | ✅ green |
| 08-02 T1 | 08-02 | 2 | TTY-04 | — | Picker enters alt-screen over a really-seeded daemon and restores the main buffer on quit | integration | `go test -tags tmux -run TestDaemonPickerEntersAltScreenAndRestoresMainBuffer ./test/tmux/...` (`test/tmux/daemon_picker_test.go`; mutation family (b) watched RED) | ✅ | ✅ green |
| 08-02 T2 | 08-02 | 2 | TTY-05 | — | Cancel writes zero config files — whole-tree sha256 before/after | integration | `go test -tags tmux -run TestInstallPickerCancelWritesNoConfig ./test/tmux/...` (`test/tmux/install_cancel_test.go`; mutation family (c) watched RED) | ✅ | ✅ green |
| 08-02 T3 | 08-02 | 2 | TTY-06 | — | Idle picker holds byte-identical across N=5 captures, N reported via t.Logf | integration | `go test -tags tmux -run TestInstallPickerFrameStableWhileIdle ./test/tmux/...` (`test/tmux/frame_stability_test.go`; family (d) non-discriminating — deferred, see 08-UAT.md) | ✅ | ✅ green |
| 08-01 T3 + 08-03 T1/T2 | 08-01, 08-03 | 1, 3 | TTY-07 | — | CI fails rather than passing empty — exact executed-count equality (TMUX_EXPECTED_TESTS=6) and tmux-version pin (tmux 3.4), both strict only under CI | CI job | `task test:tmux` under `CI=1` via `.github/workflows/ci.yml` `tmux-e2e`; shape pinned by `go test ./internal/upgrade/ -run TestInScopeJobsPopulationMatchesDisk` | ✅ | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `test/tmux/main_test.go` — `TestMain` plus its own `resolveTestBinPath` with the mandatory no-silent-fallback contract stated in the doc comment (D-09)
- [x] `test/tmux/` tmux argv wrappers — session create/teardown, `send-keys`, and D-12's `capture-pane -p -e -C -S -` capture, plus D-14's bounded stability poll
- [x] `test/tmux/` daemon seeding helper — runs the real `codegraph daemon start` as a background subprocess against a throwaway `$HOME`, with clean SIGTERM teardown
- [x] `Taskfile.yml` `test:tmux` target — D-01/D-02/D-03's `jq`-based exact-count gate, following the positive-count-gate shape at `Taskfile.yml:3548-3559` but asserting `-ne "${EXPECTED}"` rather than `-eq 0`, with a `jq` precondition
- [x] `.github/workflows/ci.yml` `tmux-e2e` job — `runs-on: ubuntu-latest`, install tmux, assert `tmux -V` against the committed constant, then a single `run:` step whose body is the literal line `task test:tmux`
- [x] `internal/upgrade/taskfile_shape_test.go` `inScopeJobs` — add the `tmux-e2e` entry, or `TestInScopeJobsPopulationMatchesDisk` fails once the job exists on disk
- [x] `08-MUTATION-LOG.md` — four RED demonstrations, family (a) using the corrected two-file mutation

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The committed `tmux -V` constant matches what `ubuntu-latest` actually ships | TTY-07 | Not observable from this machine. D-11's design deliberately defers it to a bootstrap CI run | Land the job with a placeholder, read the first run's `tmux -V` output, commit the real value, and confirm the assertion then passes for the right reason. **Performed 2026-09-11:** run 34606828356 failed at the pin by design printing `tmux 3.4`; pinned in 3899e6de; runs 34607117422 and 34658987243 (HEAD 5ffdc2a0) green with `tmux 3.4` matching the constant. |
| The `tmux-e2e` job's executed-count assertion actually fired rather than being skipped | TTY-07 | A green checkmark cannot distinguish "asserted and passed" from "never reached" | Read the job log for the printed executed and skipped counts before calling the phase done. **Performed 2026-09-11:** job 103457354620 (run 34658987243, HEAD 5ffdc2a0) log prints `test:tmux: executed=6 skipped=0 expected=6` with six `--- PASS` lines; recorded as 08-UAT.md test 21. |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 60s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved 2026-09-11

---

## Validation Audit 2026-09-11

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

All seven TTY requirements bind to a green automated test re-executed at HEAD 5ffdc2a0 during
`/gsd-verify-work 8` (local `task test:tmux` → `executed=6 skipped=0 expected=6`; CI run
34658987243 job 103457354620 → the same line, `tmux 3.4`). No new tests were generated; the
two TTY-07 manual-only checks were performed and their evidence recorded above.
