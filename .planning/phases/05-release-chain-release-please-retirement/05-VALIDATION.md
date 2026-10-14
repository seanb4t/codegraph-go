---
phase: "5"
slug: "release-chain-release-please-retirement"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-10-14"
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (`internal/upgrade`) + Python `unittest` (`scripts/test_fragment_policy.py`) + Taskfile gate targets (`check:changie`, `check:fragment-policy`) + actionlint (`task lint:actions`) |
| **Config file** | none beyond `Taskfile.yml`, `.golangci.yml`, `.github/workflows/ci.yml` |
| **Quick run command** | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade -count=1 -run '<the plan's tests>'` |
| **Full suite command** | `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade -count=1 && task check:fragment-policy && task check:changie && task lint:actions`, then `task test` at the phase gate |
| **Estimated runtime** | ~60 seconds (package 0.7 s; `check:changie` and `check:fragment-policy` dominate) |

---

## Sampling Rate

- **After every task commit:** Run `GOTOOLCHAIN=go1.26.9 go test ./internal/upgrade -count=1 -run '<the plan's tests>'`, plus `task lint:actions` when a workflow file changed
- **After every plan wave:** Run the full `internal/upgrade` package, `task check:fragment-policy`, `task check:changie`, `task lint:actions`
- **Before `/gsd-verify-work`:** Full suite plus `task test` green; scratch-tree rehearsal (copy the tree, run `release:batch`, then the `Changie` Go tests and `task check:changie` on it); `task release:dry-run` transcript captured; census clean
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| (filled by planner) | | | REL-10 / REL-11 / REL-12 / REL-13 / REL-14 / GRD-15 / GRD-16 / DOCS-12 / DOCS-13 / DOCS-14 | | | | | | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

Every guard is shown RED against a confirmed-applied, byte-cleanly-reverted mutation, with a positive assertion that it counted a non-zero population (rule `84d1gfpywd`). The requirement-to-test map is `05-RESEARCH.md` §Validation Architecture.

---

## Wave 0 Requirements

- [ ] `internal/upgrade/changie_release_shape_test.go` — workflow and Taskfile shape for REL-10/REL-11, GRD-16, `release:goreleaser` and `release:dry-run`
- [ ] `changelog.disable` guard in `internal/upgrade/goreleaser_shape_test.go` — REL-12 assumes one exists; none does
- [ ] `check:changie` legs for the release-file shape, the empty-set no-op and `release:batch` output; leg-count constants raised consistently
- [ ] `scripts/test_fragment_policy.py` additions (exemption swap, parity, WR-01 timing/grammar) and the `floor=36` bump in `check:fragment-policy`
- [ ] REL-14 census Taskfile target with a planted-reference self-test, and `05-CENSUS.md`

No new framework install is needed.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `task release:dry-run` renders a non-empty body equal to a `.changes/vX.Y.Z.md` file | REL-13 | Needs the darwin host GoReleaser toolchain; the transcript is captured evidence, not a CI check | Run `task release:dry-run` (with `env -u` for the empty `MACOS_*` variables) against an existing seed and a scratch-batched file; paste the evidence line into the plan SUMMARY |
| `release.yml` differs from post-#99 `main` by one change | REL-12 | One-time diff evidence | `git diff 7b12cb92 -- .github/workflows/release.yml` pasted into the plan SUMMARY |
| Merge-commit push fires the `CHANGELOG.md` path filter; App-token tag starts `release.yml`; Release body matches the file | REL-10 / REL-11 / REL-13 | Live GitHub behavior | Phase 6 (live proof), not this phase |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
