---
phase: 05-agent-reach-capability-model-skill-in-every-harness
verified: 2026-09-18T00:00:00Z
status: passed
score: 8/9 must-haves verified (1 partially verified — routed to human_verification)
covered_files:

  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-01-PLAN.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-01-SUMMARY.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-02-PLAN.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-02-SUMMARY.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-03-PLAN.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-03-SUMMARY.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-04-PLAN.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-04-SUMMARY.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-05-PLAN.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-05-SUMMARY.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-06-PLAN.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-06-SUMMARY.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-07-PLAN.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-07-SUMMARY.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-CONTEXT.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-LIVE-SESSIONS.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-MUTATION-LOG.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-RESEARCH.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-REVIEW-FIX.md
  - .planning/phases/05-agent-reach-capability-model-skill-in-every-harness/05-REVIEW.md
  - docs/CLI-REFERENCE.md
  - internal/agents/antigravity.go
  - internal/agents/antigravity_test.go
  - internal/agents/capabilities.go
  - internal/agents/capabilities_test.go
  - internal/agents/claude.go
  - internal/agents/claude_skillpackage_test.go
  - internal/agents/claude_symlink_test.go
  - internal/agents/codex.go
  - internal/agents/cursor.go
  - internal/agents/cursor_test.go
  - internal/agents/gemini.go
  - internal/agents/gemini_test.go
  - internal/agents/hermes.go
  - internal/agents/kiro.go
  - internal/agents/kiro_test.go
  - internal/agents/manifest.go
  - internal/agents/manifest_test.go
  - internal/agents/opencode.go
  - internal/agents/opencode_test.go
  - internal/agents/ownership_test.go
  - internal/agents/registry_test.go
  - internal/agents/shared.go
  - internal/agents/skillfrontmatter_test.go
  - internal/agents/skillshared.go
  - internal/agents/skillshared_test.go
  - internal/agents/types.go
  - internal/cli/install.go
  - internal/cli/install_test.go
  - internal/cli/plain_golden_test.go
  - internal/cli/printconfigstyle.go
  - internal/cli/printconfigstyle_test.go
  - internal/cli/testdata/plain/print-config-style-local.golden
  - internal/cli/testdata/plain/print-config-style.golden
  - internal/cli/tui/agentpicker_test.go
  - internal/cli/uninstall.go

covered_digest: "v1:sha256:3d772dfed1d4cfa86d011feb820f86c10b4211abd36bc5faa4f89065d826776b"
behavior_unverified: 0
overrides_applied: 0
behavior_unverified_items: []
human_verification:

  - test: "Obtain a Cursor account (or another authenticated cursor-agent session) and re-run the D-09 live session (install codegraph at project scope, list skills, ask a where-is-X question, negative control) plus the D-11 probe (plant a distinct sentence in a scratch repo's root AGENTS.md with no .cursor/rules/ file, ask a fresh cursor-agent to repeat project instructions)."
    expected: "Either: (a) Cursor is shown reading the shared .agents/skills/codegraph package, upgrading AGENT-09's Cursor row from [ASSUMED] to verified; and (b) the D-11 probe shows AGENTS.md pickup or non-pickup, resolving whether Cursor's instructions target should switch to repo-root AGENTS.md per D-11's rule (05-07 currently ships the no-change branch because the verdict is `not probed`, not because a probe returned negative)."
    why_human: "Requires an authenticated Cursor session and a live TTY-driven agent — categorically outside any unit test (D-00) and outside this verifier's tool access. No Cursor account exists on the maintainer's machine, so this could not be run inside the phase."
  - test: "Install Gemini CLI and Kiro on a machine that has them, then run the same live-session method (D-09/D-12: fresh session, skills-listing prompt, negative control) against `.gemini/skills/codegraph` and `.kiro/skills/codegraph`."
    expected: "Confirms (or corrects) the [ASSUMED] rows for AGENT-10/AGENT-11 that currently rest on primary-source docs (raw.githubusercontent.com/google-gemini/gemini-cli and kiro.dev, both fetched 2026-09-18) rather than an observed transcript."
    why_human: "Neither harness is installed on the maintainer's machine (confirmed via `command -v gemini`/`command -v kiro`/`command -v kiro-cli`, all exit 1, recorded in 05-LIVE-SESSIONS.md Pre-flight). Installing them is explicitly the maintainer's call (05-CONTEXT.md Deferred Ideas, D-10) — not something this phase or this verifier can do unilaterally."
---

# Phase 5: Agent Reach — Capability Model & Skill in Every Harness Verification Report

**Phase Goal:** One per-target capability table drives `install`, `uninstall`, `Detect` and `--print-config-style` for all eight targets, and every harness that has a skill mechanism receives the codegraph skill package — written once to the shared `.agents/skills/` path and verified live to be read, with a harness-specific directory only where a live session shows it is needed — every write exact-identity-owned and reversed by `uninstall` leaving unrelated content byte-identical.

**Verified:** 2026-09-18
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `AgentTarget.Capabilities()` is the single source for `SupportsLocation`/`DescribePaths`/`Detect`'s path inputs, bidirectionally guarded (AGENT-08, roadmap SC1) | ✓ VERIFIED | `TestCapabilitiesTableDrivesDerivations` (16 leaves) and `TestCapabilitiesMatchInstallWrites` (13 leaves) both PASS (re-run live, this session); both proven RED against planted mutations and reverted byte-clean in `05-MUTATION-LOG.md` Family (a1)/(a2) |
| 2 | `codegraph install --print-config-style` is read-only, honours `-t/--target`/`-l/--location`, and reports exactly what the table says for all 8 targets | ✓ VERIFIED | Real binary run this session: 8 lines printed, zero files written under a scratch `$HOME`/cwd; `print-config-style(.golden|-local.golden)` byte-match the binary's output; styled/plain parity test passes |
| 3 | The skill package (SKILL.md + sidecar manifest) is written **once** to a shared directory when several targets request it, with a requester set (`targets`) deciding deletion (AGENT-09) | ✓ VERIFIED | `internal/agents/skillshared.go` (`installSkillPackage`/`uninstallSkillPackage`); `TestSharedSkillPackage_TargetsInvariantOverAllSequences` (1554 sequences) and 11 other `TestSharedSkillPackage_*` tests pass; Cursor and opencode both declare `SkillDirs: sharedSkillDirs` and share one manifest (confirmed by reading `cursor.go`/`opencode.go`) |
| 4 | Each harness documented as reading the shared/harness-specific path is **verified live** to discover it (AGENT-09, roadmap SC2) | ⚠️ PARTIAL — see Human Verification | opencode: live-verified via `opencode debug skill` with a positive control, a negative control, and a byte-level duplicate-name check (05-LIVE-SESSIONS.md § opencode). Antigravity: live-proven that the originally-shipped path was **not** read, and the corrected path (`~/.gemini/config/skills/`) **is** read (probe-confirmed) — code moved accordingly (1A). **Cursor: not live-verified** — no Cursor account exists on the maintainer's machine, so both the D-09 shared-path session and the D-11 AGENTS.md probe recorded `not probed`; Cursor's shared-path read rests on `cursor.com/docs/skills.md` only. Gemini CLI and Kiro: not installed on the maintainer's machine, so both stay `[ASSUMED]` from primary docs, never run live. |
| 5 | opencode: skill package at a path it reads, SKILL.md frontmatter compatible, existing AGENTS.md instructions retained (AGENT-06) | ✓ VERIFIED | Live-read confirmed (see #4); `TestSkillFrontmatterMatchesEveryWrittenDir` passes; `opencodeInstructionsPath`/instructions write unchanged by this phase |
| 6 | Cursor: skill package installed at the shared path; AGENTS.md pickup probed live before any instructions-target change, outcome recorded either way (AGENT-04) | ✓ VERIFIED (probe outcome recorded as `not probed`, honestly) | Cursor declares `SkillDirs: sharedSkillDirs` (code); the D-11 probe could not run (no account) and is recorded as `not probed` with a dated maintainer-decision line in `05-LIVE-SESSIONS.md`; 05-07 correctly took the "no code change" branch per D-11's own rule ("only permits the switch after an **observed** pickup") — this satisfies AGENT-04's literal "probe outcome recorded either way," but the live-verification component itself did not happen (see Human Verification #1) |
| 7 | Antigravity: skill package written to the directory `agy` is live-proven to read; instructions reach it through the existing shared instructions file (AGENT-07) | ✓ VERIFIED, with a requirement-wording note | `antigravitySkillDirs(global)` = `[~/.gemini/config/skills/codegraph]` (code, confirmed by reading `antigravity.go`) — this is the directory a live `agy` 1.2.6 session proved reads user skills (a uniquely-named probe file was listed there; the originally-shipped `~/.gemini/antigravity-cli/skills/` was proven **not** read in the same session). Antigravity's instructions arrive via `~/.gemini/GEMINI.md` (live-read confirmed, `QUINCE-2208` sentinel round-tripped). **Requirement-wording mismatch:** AGENT-07's REQUIREMENTS.md text says "skill package via `.agents/skills/` and the instructions block in `AGENTS.md`" — neither literal path is what ships (Antigravity's own docs never list `.agents/skills/` at global scope, and its global instructions file is `GEMINI.md`, not `AGENTS.md`). This was identified in research *before* implementation (05-CONTEXT D-06(b)(c)) and confirmed live; the implementation matches the *live-proven* mechanism, not the requirement's literal wording. Flagged for the maintainer to update AGENT-07's wording, not treated as a functional gap. |
| 8 | Gemini CLI / Kiro: skill package at documented harness-specific paths, existing/steering instructions retained (AGENT-10, AGENT-11) | ✓ VERIFIED in code, `[ASSUMED]` live (see Human Verification #2) | `geminiSkillDirs`/`kiroSkillDirs` (code, confirmed); both declared `[ASSUMED]` honestly throughout `05-LIVE-SESSIONS.md`, `05-CONTEXT.md`, and REQUIREMENTS.md's own phase notes — never claimed verified. This is the disclosed, maintainer-accepted state per D-10 (neither harness is installed on this machine), not a hidden gap. |
| 9 | Every new write is exact-identity-owned; `uninstall` reverses every write leaving unrelated content byte-identical; commit `242ec0a` cited (AGENT-13, roadmap SC4) | ✓ VERIFIED | `TestOwnershipExactIdentity` re-run live this session: 32/32 leaves PASS (8 targets × 2 locations × {clean, foreign-codegraph-dir}); cites `242ec0a418703c6a4dab45188242149960cda77d` by SHA (confirmed a real commit via `git show`) and reproduces its exact precondition for `claude/*` leaves; `05-MUTATION-LOG.md` Families (a1)(a2)(b1)(b2)(c) each show a planted mutation going RED and a byte-clean revert; code review (`05-REVIEW.md`) found and the fix report (`05-REVIEW-FIX.md`) closed two real defects (CR-01: `declaredSkillFallback` compared against the wrong reference path and could falsely attribute Claude as a requester; WR-01: missing regression coverage through the real Cursor/opencode entry points) — both independently re-verified against current source in this session (`declaredSkillFallback` now compares via `sameSkillDir(dir, claudeSkillDirPath(loc))`, not the tautological `sharedSkillDirPath` comparison) |

**Score:** 8/9 truths fully verified; 1 (#4, the live-verification breadth of AGENT-09) partially verified and routed to human_verification — not a code defect, an inherent limit of live-session testing (no Cursor account, Gemini CLI/Kiro not installed) that the phase's own decision framework (D-09/D-10/D-11) explicitly anticipated and recorded honestly at every layer.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/agents/capabilities.go` | `Capabilities` type + derivations + `describeDeclaredPaths` | ✓ VERIFIED | Exists, substantive, wired into all 8 targets, referenced by `SupportsLocation`/`DescribePaths`/`Detect` |
| `internal/agents/skillshared.go` | manifest-owned writer (`installSkillPackage`/`uninstallSkillPackage`, `declaredSkillFallback`, `sameSkillDir`) | ✓ VERIFIED | Exists, substantive, wired into Claude/Cursor/opencode/Gemini/Kiro/Antigravity |
| `internal/agents/ownership_test.go` | `TestOwnershipExactIdentity` (32 leaves), cites `242ec0a` | ✓ VERIFIED | Present, ran green (32/32) this session |
| `internal/cli/printconfigstyle.go` | `printConfigStyle`, `configStyleFields` | ✓ VERIFIED | Wired into `install.go` RunE; real-binary run confirmed read-only |
| `docs/CLI-REFERENCE.md` | regenerated, describes skill package + `--print-config-style` | ✓ VERIFIED | `task docs:cli:drift` exit 0, byte-identical; contains `--print-config-style` and "skill package" language |
| `.planning/phases/.../05-MUTATION-LOG.md` | Families (a),(b),(c) | ✓ VERIFIED | 5 family entries present, each with pre-mutation gate, diff, RED transcript, byte-clean revert, green re-run |
| `.planning/phases/.../05-LIVE-SESSIONS.md` | live evidence, no PENDING | ✓ VERIFIED | Present, PENDING-free, records verdicts, transcripts, negative controls, and 5 dated maintainer decisions |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `AgentTarget.Capabilities()` (8 targets) | `internal/agents/capabilities.go` | one literal per target file | ✓ WIRED | Confirmed via `rg` (8/8 targets have exactly one `Capabilities() Capabilities {`) |
| `installDeclaredSkill`/`uninstallDeclaredSkill` | `installSkillPackageWithFallback`/`declaredSkillFallback` | table-derived skill step | ✓ WIRED | Read in `capabilities.go`; both Install and Uninstall of every skill-writing target call the pair |
| `internal/cli/install.go` RunE | `internal/cli/printconfigstyle.go printConfigStyle` | `--print-config-style` early branch | ✓ WIRED | Confirmed by real-binary behavior (short-circuits before any write) |
| `05-LIVE-SESSIONS.md` D-11 verdict | `internal/agents/cursor.go Capabilities().Instructions` | verdict-gated branch | ✓ WIRED | Verdict `not probed` → no code change (confirmed: no 05-07 commit touches `cursor.go`); golden shows `cursor: ... instructions=none` |

### Data-Flow Trace (Level 4)

Not applicable in the usual UI sense — this phase's "rendered value" is `--print-config-style`'s stdout, and its data flow was directly traced: `Capabilities()` struct literals → `configStyleFields` → `fmt.Fprintf`. No mock/static fallback found; every path resolves through the target's real path functions or returns a resolution error (never a silent default).

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `install --print-config-style` writes nothing | real binary, scratch `$HOME`/cwd, `find` afterward | 8 lines printed, `find` empty | ✓ PASS |
| Capability-table guard is discriminating (both directions) | `go test -run TestCapabilitiesTableDrivesDerivations\|TestCapabilitiesMatchInstallWrites` | both PASS (this session) | ✓ PASS |
| Ownership guard (32-leaf) | `go test -run TestOwnershipExactIdentity` | 32/32 PASS (this session) | ✓ PASS |
| `docs:cli:drift` | `task docs:cli:drift` | exit 0, byte-identical | ✓ PASS |
| Flag accounting | `TestEveryRegisteredFlagIsAccountedFor` | PASS, 115 flags accounted | ✓ PASS |
| Full module suite (minus `internal/daemon`) | `go test $(go list ./... \| grep -v daemon)` | 52 `ok`, 0 FAIL, exit 0 | ✓ PASS |
| `internal/daemon` alone (WINDOWS #37) | `go test ./internal/daemon/` | `ok`, 64.6s | ✓ PASS |
| `242ec0a` is a real, on-branch commit | `git show 242ec0a418703c6a4dab45188242149960cda77d` | matches the cited authorization-differential revert | ✓ PASS |

### Probe Execution

Not applicable — this phase's verification method is Go unit tests plus orchestrator-driven live agent sessions (already recorded in `05-LIVE-SESSIONS.md`), not `scripts/*/tests/probe-*.sh` shell probes. No such probes are declared by this phase's PLAN/SUMMARY files.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| AGENT-08 | 05-01, 05-07 | Capability table drives install/uninstall/Detect/--print-config-style | ✓ SATISFIED | Truths #1, #2 |
| AGENT-09 | 05-02, 05-03, 05-04 | Shared skill package, write-once, live-verified read | ⚠️ PARTIALLY SATISFIED | Truth #4 — code fully correct; live verification complete for opencode and Antigravity, absent for Cursor (no account) and [ASSUMED] for the global-shared-path alias itself (sessions ran at project scope only) |
| AGENT-04 | 05-04, 05-06, 05-07 | Cursor skill + AGENTS.md probe recorded either way | ✓ SATISFIED (probe non-outcome honestly recorded) | Truth #6; Human Verification #1 |
| AGENT-06 | 05-04, 05-06 | opencode skill + frontmatter + instructions retained | ✓ SATISFIED | Truth #5 |
| AGENT-07 | 05-05, 05-06, 05-07 | Antigravity skill via live-proven path + instructions | ✓ SATISFIED (implementation matches live evidence; requirement wording is stale) | Truth #7 |
| AGENT-10 | 05-05 | Gemini CLI skill dirs | ✓ SATISFIED in code, `[ASSUMED]` live | Truth #8; Human Verification #2 |
| AGENT-11 | 05-05 | Kiro skill dirs, AGENTS.md retained | ✓ SATISFIED in code, `[ASSUMED]` live | Truth #8; Human Verification #2 |
| AGENT-13 | 05-04, 05-05, review | Exact-identity ownership, `242ec0a` cited, reversibility | ✓ SATISFIED | Truth #9 |

**No orphaned requirements**: REQUIREMENTS.md maps exactly AGENT-04, AGENT-06, AGENT-07, AGENT-08, AGENT-09, AGENT-10, AGENT-11, AGENT-13 to Phase 5 (`| AGENT-XX | Phase 5 | Complete |` rows), matching this phase's own PLAN frontmatter `requirements:` fields exactly (verified by reading all 7 plans' frontmatter). AGENT-14 is explicitly Phase 7 (out of scope here, confirmed untouched: `instructions.go`'s "4 of 8" comment and `internal/mcp/` show zero diff since the phase's first commit).

### Anti-Patterns Found

None. `rg` for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` across every phase-touched production file (`capabilities.go`, `skillshared.go`, `antigravity.go`, `cursor.go`, `opencode.go`, `gemini.go`, `kiro.go`, `manifest.go`, `printconfigstyle.go`, `install.go`, `uninstall.go`) returned zero matches. Code review (`05-REVIEW.md`, iteration 3/3) found 0 critical, 0 warning findings; 3 Info-level items (IN-01 dead defensive nil-check, IN-02 missing styled-role comment/test for `kept (foreign)`, IN-03 comment-history verbosity) remain open by design (Info severity, explicitly out of scope for a critical/warning-only fix pass) — none affect correctness.

### Human Verification Required

### 1. Cursor live session and D-11 AGENTS.md probe

**Test:** Obtain an authenticated Cursor session (`cursor-agent`) and re-run the D-09 method (install codegraph at project scope in a scratch repo, ask the agent to list its skills and answer a where-is-X question, negative control) plus the D-11 probe (a scratch repo with a sentinel sentence in root `AGENTS.md` and no `.cursor/rules/` file, ask a fresh session to repeat project instructions).
**Expected:** A transcript showing whether Cursor reads `.agents/skills/codegraph` (upgrading AGENT-09's Cursor row from `[ASSUMED]` to verified) and whether Cursor picks up repo-root `AGENTS.md` (resolving D-11, which currently gates on "not probed" rather than a negative result).
**Why human:** No Cursor account exists on the maintainer's machine (recorded, dated maintainer decision in `05-CONTEXT.md` and `05-LIVE-SESSIONS.md`); a live authenticated TTY session is categorically outside any unit test (D-00) and outside this verifier's tool access.

### 2. Gemini CLI / Kiro installation and live session

**Test:** Install Gemini CLI and/or Kiro, then run the same D-09/D-12 method (fresh session, skills-listing prompt, negative control) against `.gemini/skills/codegraph` and `.kiro/skills/codegraph`.
**Expected:** Confirms or corrects the `[ASSUMED]` rows for AGENT-10/AGENT-11, which currently rest on primary-source docs (fetched 2026-09-18) rather than an observed transcript.
**Why human:** Neither harness is installed on the maintainer's machine (`command -v gemini`/`kiro`/`kiro-cli` all exit 1, recorded pre-flight); installing them is explicitly the maintainer's own call per `05-CONTEXT.md`'s Deferred Ideas — not something this verifier can do.

### Gaps Summary

No gaps. Every artifact, key link, and code-level truth this phase's goal requires exists, is substantive, is wired, and is proven by a passing test suite this session independently re-ran (32/32 ownership leaves, 16/13 capability-table derivation leaves, 52 packages green outside `internal/daemon`, `internal/daemon` green alone, `docs:cli:drift` clean, real-binary read-only proof). The two code-review findings (CR-01, WR-01) that were found during the phase's own review loop are confirmed fixed by direct source reading, not just by trusting `05-REVIEW-FIX.md`'s narrative.

What remains open is **live-verification breadth**, not implementation correctness: Cursor (no account) and Gemini CLI/Kiro (not installed) could not be probed live on this machine. This is not a hidden gap — it is exactly what the phase's own D-09/D-10/D-11 decision framework anticipated, and every artifact discloses it honestly (dated maintainer-decision lines, `[ASSUMED]` labels with doc citations, `not probed` verdicts with reasons) rather than papering over it. One genuine requirement-wording mismatch was found and is flagged for the maintainer: AGENT-07's REQUIREMENTS.md text ("skill package via `.agents/skills/` and the instructions block in `AGENTS.md`") no longer matches what ships (`~/.gemini/config/skills/` and `GEMINI.md`) — the implementation correctly follows live evidence gathered *during* this phase that falsified the requirement's original assumption; the requirement's wording, not the code, is stale.

---

_Verified: 2026-09-18_
_Verifier: Claude (gsd-verifier)_
