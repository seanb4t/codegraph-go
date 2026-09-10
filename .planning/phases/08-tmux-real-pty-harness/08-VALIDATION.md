---
phase: "08"
slug: "tmux-real-pty-harness"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
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
| TBD | TBD | 0 | TTY-01 | — | Harness refuses to pass silently when tmux is absent | integration | `GOTOOLCHAIN=go1.26.6 task test:tmux` run with `PATH` stripped of tmux, observing the skip path | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TTY-02 | — | No assertion runs against a single unstable capture | integration | Shares TTY-03's fixture — the DECRQM arrival delay IS the race case | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TTY-03 | — | Empty registry leaks no mode-query response bytes | integration | `GOTOOLCHAIN=go1.26.6 go test -tags tmux -run TestDaemonEmptyRegistry ./test/tmux/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TTY-04 | — | Picker enters alt-screen and restores the main buffer on quit | integration | `GOTOOLCHAIN=go1.26.6 go test -tags tmux -run TestDaemonPicker ./test/tmux/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TTY-05 | — | Cancel writes zero config files | integration | `GOTOOLCHAIN=go1.26.6 go test -tags tmux -run TestInstallPickerCancel ./test/tmux/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TTY-06 | — | Idle picker holds frame-stable across N captures, N reported | integration | `GOTOOLCHAIN=go1.26.6 go test -tags tmux -run TestPickerFrameStability ./test/tmux/...` | ❌ W0 | ⬜ pending |
| TBD | TBD | TBD | TTY-07 | — | CI fails rather than passing empty | CI job | `tmux-e2e` job in `ci.yml`, a thin `task test:tmux` caller with `CI=1` in `env:` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `test/tmux/main_test.go` — `TestMain` plus its own `resolveTestBinPath` with the mandatory no-silent-fallback contract stated in the doc comment (D-09)
- [ ] `test/tmux/` tmux argv wrappers — session create/teardown, `send-keys`, and D-12's `capture-pane -p -e -C -S -` capture, plus D-14's bounded stability poll
- [ ] `test/tmux/` daemon seeding helper — runs the real `codegraph daemon start` as a background subprocess against a throwaway `$HOME`, with clean SIGTERM teardown
- [ ] `Taskfile.yml` `test:tmux` target — D-01/D-02/D-03's `jq`-based exact-count gate, following the positive-count-gate shape at `Taskfile.yml:3548-3559` but asserting `-ne "${EXPECTED}"` rather than `-eq 0`, with a `jq` precondition
- [ ] `.github/workflows/ci.yml` `tmux-e2e` job — `runs-on: ubuntu-latest`, install tmux, assert `tmux -V` against the committed constant, then a single `run:` step whose body is the literal line `task test:tmux`
- [ ] `internal/upgrade/taskfile_shape_test.go` `inScopeJobs` — add the `tmux-e2e` entry, or `TestInScopeJobsPopulationMatchesDisk` fails once the job exists on disk
- [ ] `08-MUTATION-LOG.md` — four RED demonstrations, family (a) using the corrected two-file mutation

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The committed `tmux -V` constant matches what `ubuntu-latest` actually ships | TTY-07 | Not observable from this machine. D-11's design deliberately defers it to a bootstrap CI run | Land the job with a placeholder, read the first run's `tmux -V` output, commit the real value, and confirm the assertion then passes for the right reason |
| The `tmux-e2e` job's executed-count assertion actually fired rather than being skipped | TTY-07 | A green checkmark cannot distinguish "asserted and passed" from "never reached" | Read the job log for the printed executed and skipped counts before calling the phase done |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
