---
phase: "6"
slug: "first-release-through-the-chain"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-10-17"
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | none new (D-01) — live-run observation with read-only `gh`; existing `go test` / `task check:*` suites gate only the pre-merge `/gsd-quick` tasks (D-07), which run outside `/gsd-execute-phase 6` |
| **Config file** | `Taskfile.yml` |
| **Quick run command** | `task changie -- latest` |
| **Full suite command** | `task test` |
| **Estimated runtime** | ~5 seconds per observation command; the live chain itself runs for tens of minutes (release build, notarization) |

---

## Sampling Rate

- **After every task commit:** Run that task's observation command and paste its transcript into `06-EVIDENCE.md`
- **After every plan wave:** Re-read the chain state with `gh run list` for the workflows the wave waited on
- **Before `/gsd-verify-work`:** The single evidence summary is signed off once by the maintainer (D-12)
- **Max feedback latency:** 60 seconds per observation command (the waits on live runs are not feedback latency)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 06-01-01, 06-01-02 | 06-01 | 1 | SHIP-01 | T-06-03 | release PR authored by `fzy-release-please[bot]` | live observation + human checkpoint | `gh pr list --repo seanb4t/codegraph-go --head changie/release --state all --json number,title,author,state,url`; `task changie -- latest` | ✅ | ⬜ pending |
| 06-01-03 | 06-01 | 1 | SHIP-02 | T-06-01, T-06-02, T-06-04 | tag pushed by the App (`actor` = `fzy-release-please[bot]`, attempt 1 `triggering_actor` = `fzy-release-please[bot]`); Release body content-equal to `.changes/vX.Y.Z.md` (D-11) | live observation + human checkpoint | `gh run view <release-tag-run> --log \| rg "release:tag: pushed"`; `gh api repos/seanb4t/codegraph-go/actions/runs/<release-run>` and `.../attempts/1`; `norm` + `cmp` (06-RESEARCH.md Code Examples) | ✅ | ⬜ pending |
| 06-02-01, 06-02-03 | 06-02 | 2 | SHIP-03 | T-06-05, T-06-06, T-06-07, T-06-08, T-06-09 | new artifact verified against the unchanged cosign identity; upgrade runs with `HOME` isolated to a scratch dir | live observation + human checkpoint | `gh run view <post-release-verify-run> --json jobs`; `HOME="$SCRATCH" ./codegraph upgrade` then `--version` (darwin/arm64 local, linux/amd64 on a `devbox`) | ✅ | ⬜ pending |
| 06-02-02 | 06-02 | 2 | SHIP-04 | — | N/A | file edit | `jq '.entries.changie\|{version,source}' .gsd-capabilities.json`; `git diff -- .planning/seeds/SEED-005-changie-template-for-other-repos.md` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

Existing infrastructure covers all phase requirements.

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Review the rendered notes and version, then squash-merge the release PR | SHIP-01 | The merge is outward-facing and the maintainer's call (D-12) | Open the `chore(main): release vX.Y.Z` PR, read the body and version, squash-merge it |
| One blocking sign-off on the evidence summary | SHIP-02, SHIP-03 | The ROADMAP marks SC2 and SC3 as human checkpoints (D-12) | Read the evidence summary in `06-EVIDENCE.md` and approve or describe issues |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
