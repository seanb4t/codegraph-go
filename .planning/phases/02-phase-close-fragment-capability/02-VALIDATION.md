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
| 02-01-01 | 02-01 | 1 | CAP-01, CAP-02, CAP-03 | T-02-01 | Manifest installs on a scratch project with a non-reserved id; step dispatched with the key true or absent, never false; installed script writes and commits fragments with the D-07 trailers (tracer, RED first) | integration (shell) | `CHANGIE_BIN="$(cd codegraph-go && GOWORK=off go tool -modfile=go.tool-changie.mod -n changie)" /bin/bash gsd-capability-changie/test/run.sh` ends `7 of 7 legs passed` | ❌ W0 (created by this task) | ⬜ pending |
| 02-01-02 | 02-01 | 1 | CAP-03 | T-02-01, T-02-02, T-02-03, T-02-05 | Every named skip; bad-pr; entry guards; SUMMARY-derived body passed as one argv element; commit scope; idempotency; gap closure; rollback; lock (RED first) | integration (shell) | same command ends `29 of 29 legs passed`, with 29 distinct `ok [label]` lines; `shellcheck` clean | ❌ W0 (extended by this task) | ⬜ pending |
| 02-02-01 | 02-02 | 2 | CAP-01, CAP-03 | T-02-08 | LICENSE verbatim; README covers D-10; SKILL.md followed as written yields one plain Features fragment and none for planning-only work | doc checks, plus a judgment rehearsal | `cmp` of the LICENSE files; README token loop; suite `29 of 29` | ✅ | ⬜ pending |
| 02-02-02 | 02-02 | 2 | CAP-03 | T-02-09 | Disabled-key skip, explicit-pathspec staging and trailer idempotency each go RED on a confirmed mutation and are reverted byte-cleanly | mutation (shell) | RED leg names in 02-CAPABILITY-LOG.md; one worktree and a clean status in the capability repository | ✅ | ⬜ pending |
| 02-03-01 | 02-03 | 3 | CAP-01, CAP-04 | T-02-11 | Blocking-human approval before any outward-facing GitHub action | checkpoint | — (maintainer answer) | N/A | ⬜ pending |
| 02-03-02 | 02-03 | 3 | CAP-01, CAP-04 | T-02-SC, T-02-12, T-02-13, T-02-14 | Private repo; annotated v0.1.0 identical locally and remotely; a clone at the tag passes 29 of 29; the draft PR is found by the skill's lookup | scripted verification | `gh repo view … --jq .visibility` = PRIVATE; ls-remote tag compare; `gh pr list --head gsd/v0.15.0-milestone --state open …` | N/A | ⬜ pending |
| 02-04-01 | 02-04 | 4 | CAP-04 | T-02-17, T-02-18, T-02-SC | Install from the tag, then config-set; ledger and bundle ignored; commit touches only .gitignore and config.json; live `--list` finds the PR | scripted verification | `git check-ignore -v` both paths; node config check; commit file set plus `git status --porcelain` | N/A | ⬜ pending |
| 02-04-02 | 02-04 | 4 | CAP-04 | T-02-16 | Maintainer chooses how the skill is materialized (global-state mutation) | checkpoint | — (maintainer answer) | N/A | ⬜ pending |
| 02-04-03 | 02-04 | 4 | CAP-04 | T-02-16 | Skill resolvable as `gsd-changie-fragments`, byte-identical to the tag; only one global skill directory added | scripted verification | `test -f …/skills/gsd-changie-fragments/SKILL.md` plus `cmp` against `git show v0.1.0:…` plus the marker check | N/A | ⬜ pending |
| 02-05-01 | 02-05 | 5 | CAP-03 | T-02-01, T-02-20 | Real Skill-tool dispatch writes one jargon-free Features fragment with the trailers; a second dispatch reports already-recorded | integration, plus model judgment | transcript assertions recorded in 02-05-SUMMARY.md | N/A | ⬜ pending |
| 02-05-02 | 02-05 | 5 | CAP-04 | T-02-21 | CONTRIBUTING D-12 subsection in place as one hunk; task-target guard green | unit plus doc checks | `GOWORK=off go test ./internal/upgrade/ -run '^TestContributingReferencesRealTaskTargets$' -count=1 -v` | ✅ | ⬜ pending |

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
