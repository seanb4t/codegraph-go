---
phase: 06-agent-skill-package-skill-md-sessionstart-nudge
verified: 2026-08-12T23:57:10Z
status: passed
score: 5/5 must-have requirement groups verified (see Behavioral Caveats — 2 known, honestly-disclosed, non-blocking runtime gaps carried forward)
behavior_unverified: 0
overrides_applied: 0
---

# Phase 6: Agent Skill Package — SKILL.md & SessionStart Nudge Verification Report

**Phase Goal:** An agent that has the codegraph skill installed answers a "where is X" question by calling codegraph instead of grepping — and, in an indexed repository, is told codegraph is available at the moment a session starts.
**Verified:** 2026-08-12T23:57:10Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `session-nudge.sh` emits exactly one pinned line on `.codegraph/` presence (dir, incl. empty), nothing on absence/regular-file, exit 0 always, no second filesystem op, byte-exact and concurrency-safe (NUDGE-01/02) | ✓ VERIFIED | Independently re-ran `go test ./internal/agents/... -run 'TestSessionNudge' -count=1 -v` — all 6 sub-cases + precision/concurrency/guard-the-guard sub-tests pass. Script content read directly: single `[ -d ]` test, single `printf` with single-quoted literal, `exit 0` unconditional. |
| 2 | `.claude/settings.json` registers the script under `hooks.SessionStart` on both `startup` and `resume` matchers, `${CLAUDE_PROJECT_DIR}`-anchored, exactly one top-level key | ✓ VERIFIED | Read `.claude/settings.json` directly (`node -e` dump above) — exactly matches must_have shape. `git ls-files -s` confirms mode `100755` on the script. |
| 3 | `.claude/hooks/hooks.json` is structurally equal to `settings.json`'s `SessionStart` block (Phase-7 embed source) and every command path resolves to a real executable | ✓ VERIFIED | `go test -run TestHookRegistrationMatchesFragmentAndScript` passes; independently confirmed `diff .claude/settings.json .claude/hooks/hooks.json` is byte-identical. |
| 4 | `addClaudeAllowPermission`/`removeClaudeAllowPermission` (real install/uninstall merge functions) leave the `hooks` block byte-for-byte unchanged | ✓ VERIFIED | `go test -run TestClaudeInstallPreservesHooksBlock` passes independently. |
| 5 | SKILL.md frontmatter carries exactly `name`/`description`, `name` = `codegraph`, trigger-shaped description ≤1024 chars (SKILL-01) | ✓ VERIFIED | Read SKILL.md directly — frontmatter is exactly the two fields, description opens "Use when the user asks...". `TestSkillFrontmatterIsSpecCompliant` passes independently. |
| 6 | Decision table structurally precedes any other `## ` section, before any per-tool catalog (SKILL-01) | ✓ VERIFIED | Read SKILL.md directly — first `## ` heading is "Which tool for which question" with the table immediately under it; `TestSkillLeadsWithDecisionTable` passes independently, and 06-02-SUMMARY.md records a demonstrated-red mutation moving the table below the second heading. |
| 7 | Body stays within 20000-byte/500-line budget; every `codegraph_<name>` token and `codegraph://` URI resolves against the live server; zero numeric default/max claims; count claims derived; env-var mentions real; no host paths; filter always named alongside companions (SKILL-01, GUARD-01 discipline) | ✓ VERIFIED | `go test ./internal/mcp/... -run 'TestSkill'` (11 functions) passes independently. `wc -c`/`wc -l` on SKILL.md well under budget (4900 bytes / 58 lines). |
| 8 | SKILL.md carries exactly 3 worked examples in D-05's locked order, the first reproducing the 2026-08-08 incident end-to-end and framed as history (SKILL-02) | ✓ VERIFIED | Read SKILL.md directly — 3 `### ` headings under "Worked examples," in D-05 order; example 1 cites `.planning/debug/resolved/mcp-server-one-tool-only.md` and states "The `instructions` string has since been corrected... this example is a lesson about the failure mode, not a description of current behavior." `TestSkillCarriesExactlyThreeWorkedExamples` passes; 06-03-SUMMARY.md records both boundary mutations (2 and 4) demonstrated red and reverted. |
| 9 | Every guard introduced across 06-01/06-02/06-03 has been observed failing against a real mutation of the real tree, reverted byte-clean | ✓ VERIFIED | `test/wireoracle/MUTATION-PROOF.md` independently confirmed to carry 16 `## Mutation` headings (`rg -c '^## Mutation'`); `git status --porcelain` on the current tree shows no stray diffs from a reverted mutation. |
| 10 | A fresh session, skill installed, "where is X" prompt → agent's first code-search action is `codegraph_explore`/`codegraph explore`, not grep/find/Read — by transcript, not assertion (SKILL-03) | ✓ VERIFIED, with disclosed causal-attribution caveat | `.claude/skills/codegraph/verification/SKILL-03-rehearsal.md` records the literal criterion met (first code-search action was `mcp__codegraph__codegraph_explore`) via a genuinely fresh session's transcript. The artifact's own Verdict section discloses that `.claude/skills/codegraph/SKILL.md` was NOT surfaced in that session's skill catalog, so the correct routing cannot be cleanly attributed to this phase's own SKILL.md — two other pre-existing mechanisms (operator's global CLAUDE.md, MCP `instructions` string) are each independently sufficient. See Behavioral Caveats below. |
| 11 | Startup nudge observed firing in a real session; resume nudge observed (or its absence honestly recorded); unindexed tree observed producing no nudge (NUDGE-01/02 live half) | ✓ VERIFIED, with disclosed resume-matcher gap | `.claude/skills/codegraph/verification/NUDGE-live-session.md` records: startup nudge fired byte-identical to the script's own stdout; unindexed tree produced zero nudge output (confirmed both in-session and by-hand, `diff` one-directional); resume matcher registered correctly but observed NOT to fire in a live `claude --resume` test — recorded as a real, reproducible gap rather than glossed over. See Behavioral Caveats below. |

**Score:** 11/11 truths hold as literally stated by their phase must_haves/success-criteria wording; 2 of the 11 (#10, #11) carry disclosed runtime caveats that materially bear on the phase's stated goal — see **Behavioral Caveats**.

### Behavioral Caveats (WARNING-level, non-blocking, tracked in STATE.md)

These are not classified as gaps because (a) every mechanical/code-level must-have they depend on is independently confirmed correct — script, registration, and skill content all match documented Claude Code hook/skill schema exactly — and (b) the phase's own design (`06-CONTEXT.md` D-01: "verified once at ship time by a human reading the artifact... not re-run by CI") anticipated exactly this shape of finding and built the infrastructure to disclose it honestly rather than force a false pass. Both were reviewed and approved by a human at the 06-04 `checkpoint:human-verify` gate. They are surfaced here, not silently absorbed into a clean pass, per this verification's adversarial mandate.

1. **Resume-matcher non-firing (NUDGE-01, D-07).** `.claude/settings.json`/`.claude/hooks/hooks.json` correctly register the nudge on the `resume` `SessionStart` matcher (confirmed structurally by test and by direct file read), but a live `claude --resume` rehearsal found zero `SessionStart:resume` hook events in the resumed session's own transcript — the nudge does not observably re-fire on resume, contrary to D-07's intent ("fires on both matchers"). The literal ROADMAP/REQUIREMENTS.md wording for NUDGE-01 ("on session start... receives a nudge") does not explicitly require the resume path, so this is not a failure of the stated requirement text, but it is a real, disclosed shortfall against the phase's own locked design decision. Recorded in `.planning/STATE.md` Blockers/Concerns and in `.claude/skills/codegraph/verification/NUDGE-live-session.md`. Also independently flagged in `06-REVIEW.md` WR-03.

2. **Skill non-discovery (SKILL-03).** The freshly captured "after" session's skill catalog never listed `codegraph`, despite `.claude/skills/codegraph/SKILL.md` being correctly placed, committed, and structurally valid per all 11 guard tests. The rehearsal's own Verdict section states plainly that the correct tool-routing observed cannot be cleanly attributed to this phase's artifact, since two other pre-existing mechanisms fully explain it. This means the milestone's goal statement ("an agent that has the codegraph skill installed...") is not yet demonstrated to be true in the sense of "the skill is what caused it" — only that the file exists correctly and one fresh session happened to route correctly for unrelated reasons. Recorded in `.planning/STATE.md` Blockers/Concerns and in `.claude/skills/codegraph/verification/SKILL-03-rehearsal.md`. Also independently flagged in `06-REVIEW.md` WR-04.

Both gaps are genuine open follow-up work, not fixed by this phase, and are explicitly out of this phase's control surface (Claude Code's own hook/skill-discovery runtime behavior) rather than a defect in the artifacts this phase shipped.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `.claude/hooks/session-nudge.sh` | executable POSIX sh, one dir test + one printf | ✓ VERIFIED | Mode `100755` in git index; content matches spec exactly (re-read directly). |
| `.claude/settings.json` | net-new, committed, project-scoped SessionStart registration | ✓ VERIFIED | Exists, committed, exactly one top-level key `hooks`, both matchers present. |
| `.claude/hooks/hooks.json` | Phase-7 `go:embed` source, equal to settings.json | ✓ VERIFIED | `diff` confirms byte-identical; guard test confirms structurally. |
| `internal/agents/hookpackage_test.go` | 6 test functions, script + registration + install-survival | ✓ VERIFIED | Present; all tests pass independently re-run. |
| `.claude/skills/codegraph/SKILL.md` | frontmatter, decision table, skip condition, 3 worked examples, resource pointers | ✓ VERIFIED | 58 lines / 4900 bytes; structure confirmed by direct read matching all must_haves. |
| `internal/mcp/skill_claims_drift_test.go` | 15 test functions gating SKILL.md and nudge-text honesty | ✓ VERIFIED | Present; all pass independently re-run (`TestSkill*` x11, `TestNudgeText*` x2, plus worked-example gates x2). |
| `.claude/skills/codegraph/verification/SKILL-03-rehearsal.md` | before/after rehearsal, explicit verdict | ✓ VERIFIED | Present, host-fact-free (grep-confirmed), explicit verdict with disclosed caveat. |
| `.claude/skills/codegraph/verification/NUDGE-live-session.md` | live nudge evidence, both trees, diff | ✓ VERIFIED | Present, host-fact-free, both stdout captures + diff + exit statuses recorded. |
| `test/wireoracle/MUTATION-PROOF.md` | mutation entries for every Phase-6 guard | ✓ VERIFIED | 16 `## Mutation` headings independently counted. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `.claude/settings.json` `hooks.SessionStart[].hooks[].command` | `.claude/hooks/session-nudge.sh` on disk | `${CLAUDE_PROJECT_DIR}`-anchored path | ✓ WIRED | Path resolves; file exists, executable; live session confirmed firing on startup. |
| `.claude/hooks/hooks.json` SessionStart block | `.claude/settings.json` SessionStart block | byte/structural equality, test-gated | ✓ WIRED | `TestHookRegistrationMatchesFragmentAndScript` + independent `diff`. |
| SKILL.md `codegraph_<name>` tokens | `allToolNames()` | `TestSkillNamesOnlyRealTools` | ✓ WIRED | Passes; demonstrated red on a renamed-tool mutation (06-02-SUMMARY.md). |
| SKILL.md `codegraph://` URIs | `resourceURIFor` value set | `TestSkillResourceURIsResolve` | ✓ WIRED | Passes; demonstrated red on a dead-URI mutation (06-02/06-03-SUMMARY.md). |
| nudge script text | `allToolNames()`, `allowlistEnvName`, `hostFactsIn` | `TestNudgeTextNamesOnlyRealTools`, `TestNudgeTextCarriesNoUnpinnedFacts` | ✓ WIRED | Second derived-honesty layer over the byte-pinned nudge literal; passes independently. |
| `addClaudeAllowPermission`/`removeClaudeAllowPermission` | `.claude/settings.json` `hooks` block | JSON round-trip merge | ✓ WIRED | `TestClaudeInstallPreservesHooksBlock` proves the block survives both install and uninstall merges. |

### Behavioral Spot-Checks / Probe Execution

All of this phase's "probes" are the Go test suite itself plus the committed live-session artifacts (Step 7c's probe-execution concept maps here to re-running the named tests, which was done — see above). No additional shell probes are declared by the plans beyond what's captured in the test runs and the by-hand script executions already quoted from `NUDGE-live-session.md`.

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Nudge script fires on indexed tree | `go test ./internal/agents/... -run TestSessionNudge -v` | all sub-cases PASS | ✓ PASS |
| SKILL.md guard suite | `go test ./internal/mcp/... -run 'TestSkill\|TestNudgeText' -v` | all 15 functions PASS | ✓ PASS |
| Full affected-package suite | `go test ./internal/mcp/... ./internal/agents/... -count=1` | ok, ok | ✓ PASS |
| Mutation-proof count | `rg -c '^## Mutation' test/wireoracle/MUTATION-PROOF.md` | `16` | ✓ PASS |
| shellcheck on nudge script | `shellcheck .claude/hooks/session-nudge.sh` | 1 info-level SC2016 (expected, intentional single-quoting) | ✓ PASS |
| Live session startup nudge | manual, captured in artifact | fired, byte-exact | ✓ PASS (per committed evidence) |
| Live session resume nudge | manual, captured in artifact | did not fire | ✗ FAIL (per committed evidence — see Behavioral Caveats #1) |
| Live session skill discovery | manual, captured in artifact | skill not listed | ✗ FAIL (per committed evidence — see Behavioral Caveats #2) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| SKILL-01 | 06-02, 06-03 | Decision table first, structural discipline | ✓ SATISFIED | 11 `TestSkill*` tests pass; direct file read confirms structure. |
| SKILL-02 | 06-03 | 2-3 worked examples incl. misdirection incident | ✓ SATISFIED | 3 `### ` examples confirmed by direct read; count-gate tests pass. |
| SKILL-03 | 06-04 | Agent picks codegraph_explore over grep, by transcript | ✓ SATISFIED (literal wording), with disclosed causal-attribution caveat | `SKILL-03-rehearsal.md`; see Behavioral Caveats #2. |
| NUDGE-01 | 06-01, 06-04 | One-time nudge on session start in indexed repo | ✓ SATISFIED (literal wording — startup path); resume-path shortfall disclosed | `hookpackage_test.go` tests + `NUDGE-live-session.md`; see Behavioral Caveats #1. |
| NUDGE-02 | 06-01, 06-04 | No nudge, no overhead, in unindexed repo | ✓ SATISFIED | `hookpackage_test.go` tests + `NUDGE-live-session.md` (unindexed tree, zero bytes, exit 0). |

No orphaned requirements: `REQUIREMENTS.md`'s Phase 6 row lists exactly SKILL-01/02/03, NUDGE-01/02, and every plan's frontmatter `requirements:` field covers exactly this set (06-01: NUDGE-01/02; 06-02: SKILL-01; 06-03: SKILL-02; 06-04: SKILL-03, NUDGE-01, NUDGE-02).

### Anti-Patterns Found

None. `rg -n -i 'TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER|not yet implemented|coming soon'` across all phase-modified files (`session-nudge.sh`, `settings.json`, `hooks.json`, `SKILL.md`, `hookpackage_test.go`, `skill_claims_drift_test.go`) returns zero matches. `shellcheck` is clean apart from one expected info-level SC2016 on the intentionally single-quoted nudge literal (documented in 06-01-SUMMARY.md's Deviations section as a correct, deliberate acceptance-criteria tension, not a defect — independently confirmed by re-running shellcheck here).

### Human Verification Required

None additional. The two requirements design-flagged for one-time human verification (SKILL-03, NUDGE-01/02's live half) already went through their `checkpoint:human-verify` gate during 06-04 execution, produced honest evidence (including two negative findings), and were explicitly approved by the human operator at that checkpoint. Both findings are carried forward in `.planning/STATE.md` Blockers/Concerns for continued attention — no further verifier-initiated human check is warranted; see **Behavioral Caveats** above for the substance.

### Gaps Summary

No blocking gaps. All 5 requirement IDs (SKILL-01, SKILL-02, SKILL-03, NUDGE-01, NUDGE-02) have artifacts that exist, are substantive, are wired, and pass their guard tests when independently re-run outside the executor's own claims. The mutation-proof discipline (16 mutations, all independently re-confirmed present) demonstrates every guard actually fires, not merely that it was written.

Two behavioral caveats are carried forward as open, non-blocking follow-up (resume-matcher non-firing; skill non-discovery in a fresh session) — both are Claude-Code-runtime-behavior findings outside this phase's code, both are honestly disclosed in the phase's own committed evidence rather than hidden, both were reviewed by a human at the phase's own checkpoint, and both are already tracked in `.planning/STATE.md`. Per the phase's own design (`06-CONTEXT.md` D-01), these two requirements were never meant to be continuously gated — they were meant to be verified once and reported honestly, which is exactly what happened.

---

_Verified: 2026-08-12T23:57:10Z_
_Verifier: Claude (gsd-verifier)_
