---
phase: "2"
slug: "phase-close-fragment-capability"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-25"
---

# Phase 2 — Validation Strategy

> The per-phase validation contract that execution checks against.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Shell suite `test/run.sh` in the capability repository `seanb4t/gsd-capability-changie` (D-09). This phase touches no Go code in codegraph-go. |
| **Config file** | none — Wave 0 creates `test/run.sh` |
| **Quick run command** | `bash test/run.sh` (in the capability repository checkout) |
| **Full suite command** | `bash test/run.sh`, then the codegraph-go CAP-04 checks: `git status --porcelain`, `gsd-tools loop render-hooks verify:post --raw`, and the check that the skill is materialized (RESEARCH Pitfall 1) |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `bash test/run.sh` for capability-repository tasks, or `git status --porcelain` for codegraph-go tasks.
- **After every plan wave:** Run the full suite command.
- **Before `/gsd-verify-work`:** The full suite must be green. Paste its evidence into the codegraph-go phase artifacts; the capability repository's own history does not count (D-09).
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 2-TBD | TBD | 1 | CAP-01 | — | Manifest validates; id is not a reserved prefix | integration (shell) | `bash test/run.sh` (install leg) | ❌ W0 | ⬜ pending |
| 2-TBD | TBD | 1 | CAP-02 | — | Step dispatched only when `workflow.changie_fragments` is true; both directions observed | integration (shell) | `bash test/run.sh` (render-hooks leg) | ❌ W0 | ⬜ pending |
| 2-TBD | TBD | 1 | CAP-03 | T-2-shell-injection | SUMMARY-derived bodies passed as single argv elements; every skip case prints a named reason; the write case asserts the exact fragment file and trailers | integration (shell), plus one real skill run | `bash test/run.sh` (skip and write legs) | ❌ W0 | ⬜ pending |
| 2-TBD | TBD | 2 | CAP-04 | T-2-global-state | Install, then config-set; tree clean apart from allowed files; skill materialized and resolvable as `gsd-changie-fragments` | scripted verification | `git status --porcelain`; `gsd-tools loop render-hooks verify:post --raw`; skill-materialization check | N/A | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*
*The planner replaces the TBD task IDs with real `{phase}-{plan}-{task}` IDs.*

---

## Wave 0 Requirements

- [ ] `gsd-capability-changie/test/run.sh` — covers CAP-01, CAP-02 and CAP-03, prints an executed-case count, and fails on zero (rule `84d1gfpywd`)
- [ ] `gsd-capability-changie/skills/changie-fragments/scripts/write-fragments.sh` (or the inline fallback, D-05) — the deterministic half under test
- [ ] A codegraph-go verification command for the skill-materialization step (RESEARCH Pitfall 1)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| The model's judgment: user-visible vs planning-only classification, kind choice, sentence quality | CAP-03 | Not deterministic; it needs a real model run | Run the skill once against fixture SUMMARYs (one user-visible change, one planning-only change). Record the transcript in the phase SUMMARY. |
| Global-scope skill materialization writes to `~/.claude` | CAP-04 | Mutates the maintainer's global runtime state | Maintainer-approved checkpoint. Record the before/after listing of `~/.claude/skills/` for `*changie*`. |
| Draft milestone PR opened | D-04 (enables CAP-05) | Outward-facing GitHub action | Maintainer-approved checkpoint. Record the PR number. |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
