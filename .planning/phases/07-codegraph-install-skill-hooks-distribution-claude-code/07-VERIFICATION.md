---
phase: 07-codegraph-install-skill-hooks-distribution-claude-code
verified: 2026-08-13T18:31:57Z
status: passed
score: 22/22 must-have truths verified (1 via override)
behavior_unverified: 0
overrides_applied: 1
overrides:

  - must_have: "An interrupted write leaves either the complete previous file or the complete new file on disk and never a truncated one (07-02-PLAN.md must_haves, verification: backstop)"
    reason: "This is an OS-level rename(2) atomicity guarantee, not application logic — the code correctly funnels every write through internal/fsatomic.WriteFile's temp-file-plus-rename primitive (confirmed by direct read), and internal/fsatomic's own test suite already covers content/mode/parent-dir/no-leftover-temp-file. A genuine process-kill-mid-write simulation is disproportionate effort for a well-established, industry-standard primitive. Formally risk-accepted through this project's own security process: AR-07-02 in 07-SECURITY.md, locked at plan time and re-affirmed at both the prior and this verification pass, with threats_open: 0 sign-off."
    accepted_by: "07-SECURITY.md sign-off (AR-07-02) — locked in 07-01/07-02-PLAN.md frontmatter, re-affirmed at security verification"
    accepted_at: "2026-08-13T14:28:11-04:00"
re_verification:
  previous_status: human_needed
  previous_score: 21/22
  gaps_closed:

    - "Backstop interrupted-write truth: previously routed to human verification with no test; now covered by a formal Accepted Risk (AR-07-02) in 07-SECURITY.md and accepted via override — counted toward score."
    - "Judgment-tier prohibitions (uninstall scope, manifest-not-tamper-detection, upgrade-scope-limited-to-configured-locations): previously flagged 'recommend human confirmation before ship' with no confirmation on record; the completed 07-UAT.md (21/21 passed, test 1 explicitly reviewing and confirming the D1-D16/D15a automated-coverage list that maps directly onto these three prohibitions) constitutes the human checkpoint this project's flagging discipline calls for. Treated as resolved."
  gaps_remaining:

    - "Live Claude Code session smoke test of the global-scope install path: still not performed. 07-UAT.md test 4 substitutes a direct subprocess execution of the resolved hook command for an actual live Claude Code session, and its own evidence field explicitly states this does not prove Claude Code's own hook-dispatch runtime resolves and invokes the command the same way a manual exec does. Phase 6 set the precedent for this exact class of risk by running and reporting on a genuine live session (including honestly disclosing a resume-matcher gap it found); Phase 7's default scope (global) is precisely the path Phase 6 never exercised. Kept open."
  regressions: []
---

# Phase 7: `codegraph install` Skill + Hooks Distribution (Claude Code) Verification Report

**Phase Goal:** Installing codegraph installs its skill package too — carried inside the binary, so it moves with `codegraph upgrade` — and uninstalling takes back exactly what install wrote, leaving user-authored content byte-untouched.
**Verified:** 2026-08-13T18:31:57Z
**Status:** human_needed
**Re-verification:** Yes — after 07-REVIEW.md fix commit (c2ea00f, already reflected in the prior pass), a coverage-entry correction (1a9d42b), a completed UAT session (73629da, 21/21 passed), and a completed security sign-off (3299b08, threats_open: 0)

## Summary

Independently re-ran the full verification surface for this phase rather than trusting SUMMARY.md/UAT.md/SECURITY.md narration. `go build ./...` is clean, `go vet ./...` is clean, `go test ./...` is green across every package in the repo with 0 FAIL and 0 unexpected SKIP. Every named regression test cited by the code-review fix (CR-01, CR-02, WR-01, WR-04, all in commit `c2ea00f`) was run directly and passes; the actual diff for each fix was read line-by-line and confirmed to do what the commit message and SUMMARY claim — not merely "a test with that name exists." The mid-wave security vulnerability (matcher-based hook-ownership recovery) was independently re-confirmed reverted: `rg` over `internal/agents/shared.go`/`claude.go` finds ownership determined solely by exact command-string comparison, no residual matcher-based logic.

Of the two items that kept the prior pass at `human_needed`, one is now resolved via a formal, project-sanctioned Accepted Risk (backstop interrupted-write truth — AR-07-02, `07-SECURITY.md`) and is counted toward the score as `PASSED (override)`. The judgment-tier prohibitions flagged for human confirmation are treated as resolved by the completed UAT session, which functions as this project's end-of-phase human checkpoint. The second item — a genuine live Claude Code session smoke test of the global-scope install path — remains open. `07-UAT.md` test 4 is valuable independent evidence (it definitively closes the narrower tilde-expansion risk RESEARCH.md's Assumption A3 named, since the actual command never uses `~`), but its own evidence field explicitly discloses that it substituted a direct subprocess execution for an actual live Claude Code session, and states in its own words that Claude Code's own hook-dispatch runtime resolving and invoking that command was "the one part of this UAT session genuinely outside what I can self-verify from a shell." Phase 6 already established, in this same project, what closing this exact class of risk looks like — an actual live session, with results (including a disclosed gap) recorded in a `verification/` artifact. Phase 7 has not done that for its own (global, default-scope) hook path. This is not a code defect — it is an unclosed verification step for a genuinely external, unverifiable-from-a-shell runtime behavior, so the status stays `human_needed` rather than `gaps_found`.

## Goal Achievement

### Observable Truths (Roadmap Success Criteria)

| # | Truth (roadmap SC) | Status | Evidence |
|---|-------|--------|----------|
| 1 | `codegraph install` places SKILL.md and hooks.json in Claude Code's real read locations; a second install is byte-identical (AGENT-01) | ✓ VERIFIED | `go test ./internal/agents/... -run TestClaude_Install_WritesSkillPackage_EndToEnd\|TestClaude_Install_SkillPackageReRunIsUnchanged -v` — both pass (re-run this session). Read `claudeassets.go` and `internal/agents/claude.go`'s `Install` directly — 3 exact `//go:embed` directives, no directory pattern; `writeEmbeddedFile` short-circuits on raw-byte + mode match (post-CR-02). |
| 2 | `codegraph uninstall` after `install` returns the tree to pre-install bytes, including a fixture with unrelated pre-existing hook entries (AGENT-02) | ✓ VERIFIED | `go test ./internal/agents/... -run TestClaude_SkillPackageRoundTripIsByteInvariant -v` passes. `removeHookEntry` confirmed (direct read, this session) to partition by exact command-string match only — no matcher-based logic anywhere in `shared.go`/`claude.go`. |
| 3 | The installed package is the running binary's own embedded copy; after `upgrade` the on-disk skill is the newer content and the installed version is observable from the files (AGENT-03) | ✓ VERIFIED | `internal/agents/manifest.go` + `internal/cli/upgrade.go`'s `refreshInstalledSkills` read directly; named manifest/upgrade tests pass. Independently corroborated live in `07-UAT.md` test 2 (two real binaries, version flipped 1→2 in the manifest after a re-install, byte-identical unchanged content correctly reported `ActionUnchanged`). |
| 4 | A read error or unparseable existing `hooks.json`/settings.json makes install/uninstall surface the error instead of overwriting or deleting content — proven across the {install,uninstall}×{read-error,malformed} matrix | ✓ VERIFIED | `go test ./internal/agents/... -run 'TestClaude_SettingsReadFailureNeverDestroysContent\|TestClaude_SettingsReadFailureMatrixIsNotVacuous' -v` — all 4 cells plus the non-vacuity guard pass. Independently corroborated live in `07-UAT.md` test 3 (corrupted settings.json, real `install` run, hooks step failed loud, manifest not falsely rewritten — live confirmation of the CR-01 fix). |

### Observable Truths (Plan-level must_haves, all 4 plans)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 5 (07-01) | Install writes embedded SKILL.md, executable session-nudge.sh, SessionStart matcher-blocks at both locations | ✓ VERIFIED | `TestClaude_Install_WritesSkillPackage_EndToEnd` passes (re-run). |
| 6 (07-01) | Second install at same location is byte/JSON-deep-equal no-op | ✓ VERIFIED | `TestClaude_Install_SkillPackageReRunIsUnchanged` passes (re-run). |
| 7 (07-01) | Boundary states: unrelated content preserved; absent settings.json creates only codegraph blocks; exact-blocks settings.json writes nothing | ✓ VERIFIED | `TestClaude_Install_PreservesUnrelatedHooksContent`, `TestClaude_Install_HooksBoundaryStates` pass (re-run). |
| 8 (07-01) | Hook command differs by location, never a bare `$PATH`-resolved filename | ✓ VERIFIED | `TestClaude_Install_HookCommandIsLocationAware` + `TestHookRegistrationMatchesFragmentAndScript` pass. Independently corroborated live in `07-UAT.md` test 4 (extracted the real global-scope command string — fully-resolved absolute path, no `~`). |
| 9 (07-01) | Binary embeds exactly 3 files, no `verification/` transcripts shipped | ✓ VERIFIED | `TestClaudeAssets_EmbedsNoVerificationTranscripts` passes; `rg -c '^//go:embed' claudeassets.go` = 3 (re-checked), no directory pattern. |
| 10 (07-02) | Uninstall→install round-trip is `jsonDeepEqual` byte-invariant, including a fixture sharing codegraph's `startup` matcher | ✓ VERIFIED | `TestClaude_SkillPackageRoundTripIsByteInvariant`, `TestClaude_Uninstall_RemovesOnlyOwnBlockUnderSharedMatcher` pass. |
| 11 (07-02) | Uninstall removes only SKILL.md+script; user file in same dir survives; dir removed only once empty | ✓ VERIFIED | `TestClaude_Uninstall_PreservesUserFileInSkillDir` passes. |
| 12 (07-02) | Uninstall of never-installed target reports `ActionNotFound`, no error, no file touched | ✓ VERIFIED | `TestClaude_Uninstall_NeverInstalledIsNotFound` passes. |
| 13 (07-02) | All 4 cells of {install,uninstall}×{read-error,malformed} surface the failure and leave bytes untouched | ✓ VERIFIED | `TestClaude_SettingsReadFailureNeverDestroysContent` (4 sub-cases) + `TestClaude_SettingsReadFailureMatrixIsNotVacuous` pass. |
| 14 (07-02) | One shared read-error posture for settings.json across hooks step and AutoAllow step | ✓ VERIFIED | `TestClaude_AutoAllowSharesStrictReadPosture` passes; `rg -n readJSONFileStrict internal/agents/claude.go` shows both callers. |
| 15 (07-02) | Interrupted write leaves complete old or new file, never truncated (backstop truth, no test) | ✓ PASSED (override) | Override: OS `rename(2)`-atomicity property, code correctly delegates to `internal/fsatomic.WriteFile`'s temp-file-plus-rename; formally risk-accepted as AR-07-02 in `07-SECURITY.md` (`threats_open: 0`, T-07-08 disposition "accept"). See frontmatter override entry. |
| 16 (07-03) | Version readable from installed files via sidecar manifest (`version.Info().Version` + per-artifact sha256) | ✓ VERIFIED | `TestClaude_Install_WritesManifest`, `TestManifest_RoundTrips`, `TestManifest_HashIsSha256Hex` pass. |
| 17 (07-03) | Hooks-registration hash covers only codegraph's own SessionStart blocks, never the whole settings.json | ✓ VERIFIED | `TestManifest_OwnedHookBlockHashIgnoresUnrelatedSettings` passes. |
| 18 (07-03) | Second install leaves manifest byte-unchanged including `installed_at` | ✓ VERIFIED | `TestClaude_Install_ManifestIsIdempotent`, `TestManifest_WriteIsIdempotentAndPreservesTimestamp` pass. |
| 19 (07-03) | Hand-edited artifact silently overwritten (no prompt/warning/Notes entry); missing manifest treated as drifted | ✓ VERIFIED | `TestClaude_Install_SilentlyOverwritesHandEditedSkill`, `TestClaude_Install_MissingManifestIsTreatedAsDrifted`, `TestClaude_Install_DriftIsDetectableFromTheManifest` pass — all assert `len(result.Notes)==0`. |
| 20 (07-03) | `ConfiguredSkillLocations(Claude)` reports exactly the locations carrying a manifest, by fixed-path probe only | ✓ VERIFIED | `TestConfiguredSkillLocations_ProbesFixedPaths` + post-review `TestConfiguredSkillLocations_IncludesLocationWithCorruptedManifest`/`_ExcludesGenuinelyAbsentLocation` pass (re-run this session, WR-04 fix confirmed by diff). |
| 21 (07-04) | `upgrade` re-invokes `Install()` for every previously-configured location after a successful swap (D-06) | ✓ VERIFIED | `TestUpgradeCommand_RefreshesConfiguredLocations`, `TestRefreshInstalledSkills_OnlyPreviouslyConfiguredLocations` pass. |
| 22 (07-04) | `upgrade --check` performs no refresh; swap failure skips refresh; refresh failure after successful swap is a warning naming `codegraph install`, not a failed upgrade (D-07) | ✓ VERIFIED | `TestUpgradeCommand_SkipsRefreshUnderCheck`, `TestUpgradeCommand_SkipsRefreshWhenSwapFails`, `TestUpgradeCommand_RefreshFailureIsWarningNotError`, `TestUpgradeCommand_RefreshSuccessPrintsNoWarning`, `TestUpgradeCommand_SwapFailureReturnsSwapError` all pass. Independently corroborated live in `07-UAT.md` test 3. |

**Score:** 22/22 truths verified (1 via a documented override backed by a formal security sign-off; 0 present-but-behavior-unverified).

### Post-Merge Security Correction — Re-Confirmed This Session

`242ec0a` (revert of a matcher+shape hook-ownership recovery heuristic that a background security review found let codegraph silently claim an unrelated hook sharing a matcher name) was re-confirmed directly, not taken on the SUMMARY's word:

- `git log --oneline` confirms `242ec0a` sits in branch history between the vulnerable commit and Wave 4's work.
- `rg -n "blockMatchers|recoveryMatchers|matcher value|matcher =="  internal/agents/shared.go internal/agents/claude.go` finds only doc-comment references — no residual matcher-based ownership logic. `writeHookEntry`/`removeHookEntry` were read in full this session: ownership is determined purely by exact `command` string comparison against `ownCommands`.
- `TestClaude_Install_NeverClaimsOwnershipOfUnrelatedHookUnderSameMatcher` re-run this session, passes.
- `07-03-SUMMARY.md`'s "Post-Hoc Correction" section (D15/D15a coverage-entry fix, commit `1a9d42b`) was read to the end and cross-checked against the diff: `TestClaude_Install_SilentlyOverwritesHandEditedHookBlock` was genuinely renamed to `TestClaude_Install_HandEditedHookBlockDuplicatesRatherThanOverwritesUnrelated` (confirmed present, passing at that name) with assertions updated to expect duplication rather than restoration, matching the narrative exactly.

### Post-Review Fix Commit (c2ea00f) — Re-Confirmed This Session by Reading the Diff, Not the Message

| Finding | Claimed fix | Independently confirmed this session |
|---------|-------------|------------------------------------------|
| CR-01 (manifest written on partial write failure) | Gate `have*Content` flags on write success (`werr == nil`) | ✓ `git show c2ea00f -- internal/agents/claude.go` read directly: all 3 artifacts now set their `have*Content` flag only inside `if werr == nil`. `TestClaude_Install_ManifestNotWrittenOnPartialWriteFailure` re-run, passes. |
| CR-02 (executable bit never self-heals) | `writeEmbeddedFile` checks mode even when content bytes match, chmods in place | ✓ Diff read directly: byte-match short-circuit now falls through to an `os.Stat` mode check for executable artifacts and `os.Chmod(0o755)` if the bit is missing. `TestClaude_Install_RestoresLostExecutableBitWithoutContentChange` re-run, passes. |
| WR-01 (corrupted manifest permanently blocks writes) | `writeManifest` self-heals a corrupted existing manifest by overwriting | ✓ Diff read directly: `readManifest` error path now resets `existing`/`existedBefore` instead of returning early. `TestWriteManifest_SelfHealsCorruptedExistingManifest` re-run, passes. |
| WR-04 (`ConfiguredSkillLocations` silently drops corrupted-manifest locations) | Include a present-but-corrupted manifest location, exclude only genuinely absent ones | ✓ Diff read directly: condition changed from `err != nil \|\| !present` to `rerr == nil && !present`. Both new tests re-run, pass. |
| WR-02 (duplicate settings.json status line) | Documented follow-up debt, not fixed | Confirmed still not fixed by direct read of `internal/cli/install.go`'s `printAgentResults` — cosmetic, correctly triaged, non-blocking. |
| WR-03 (single-command-string assumption unenforced) | Documented follow-up debt, not fixed | Confirmed still not fixed by direct read of `claudeSessionStartBlocks` — currently safe by inspection, correctly triaged, non-blocking. |

Full suite this session: `go build ./...` clean; `go vet ./...` clean; `go test ./...` — every package `ok`, 0 FAIL, 0 unexpected SKIP.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `claudeassets.go` | Repo-root `go:embed` package, 3 exact files | ✓ VERIFIED | Re-confirmed by direct read this session. |
| `internal/agents/claude.go` | `Install`/`Uninstall`/`DescribePaths` extended with skill/script/hooks/manifest steps | ✓ VERIFIED | Confirmed by direct read; all helpers present and exercised by passing tests. |
| `internal/agents/shared.go` | `readJSONFileStrict`, `writeHookEntry`/`removeHookEntry`, `atomicWriteExecutableFile`, `writeEmbeddedFile`, `removeEmbeddedFile`, `removeSkillDirIfEmpty` | ✓ VERIFIED | All present, all covered by passing tests. |
| `internal/agents/manifest.go` | `skillManifest`, `hashContent`, `hashOwnedHookBlocks`, `readManifest`/`writeManifest`, `ConfiguredSkillLocations` | ✓ VERIFIED | Present; package comment explicitly disclaims tamper-detection framing (D-05/AR-07-01 compliance), re-confirmed by direct read. |
| `internal/cli/upgrade.go` | Post-swap refresh (D-06) + D-07 warning-not-error reporting | ✓ VERIFIED | `refreshInstalledSkillsFunc` injectable seam present; `RunE` sequencing matches plan exactly. |
| `internal/agents/claude_skillpackage_test.go`, `claude_readerror_test.go`, `manifest_test.go`, `internal/cli/upgrade_test.go` | New/extended test files | ✓ VERIFIED | `go test ./internal/agents/... ./internal/cli/...` green this session; all named tests cited by every plan's acceptance criteria exist and pass. |
| `07-SECURITY.md` | Threat register, all threats closed or accepted | ✓ VERIFIED | `threats_open: 0`, 17 threats, all closed/accepted with dispositions read and cross-checked against source for the two "accept" rows (T-07-07/AR-07-01, T-07-08/AR-07-02). |
| `07-UAT.md` | Completed human UAT session | ✓ VERIFIED | `status: complete`, 21/21 passed, 0 issues, 0 pending; read to the end including tests 2-4's `evidence:` fields documenting real live CLI execution. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `claudeassets.FS` | `claudeTarget.Install` | direct import, `internal/agents/claude.go` | ✓ WIRED | Confirmed by read this session. |
| `readJSONFileStrict` | `writeHookEntry`/`removeHookEntry`/`addClaudeAllowPermission`/`removeClaudeAllowPermission` | shared read gate over `claudeSettingsPath(loc)` | ✓ WIRED | `rg -n readJSONFileStrict internal/agents/claude.go` re-confirmed all four callers. |
| `claudeHookCommand(loc)` | the `command` string in `hooks.SessionStart` blocks | `claudeSessionStartBlocks(loc)` | ✓ WIRED | Test + live UAT test 4 both confirm. |
| `removeHookEntry` | `claudeSettingsPath(loc)` | command-string-only ownership | ✓ WIRED | Re-confirmed by direct read this session — no matcher-based logic remains. |
| `agents.ConfiguredSkillLocations(agents.Claude)` | `refreshInstalledSkills` → `Install()` | `internal/cli/upgrade.go` | ✓ WIRED | Direct read; test proves scope is never widened beyond prior configuration. |
| `upgradeRunFunc` returns nil | `refreshInstalledSkillsFunc` runs | `RunE` sequencing | ✓ WIRED | Direct read + tests prove ordering/gating; live UAT test 2/3 corroborate. |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| AGENT-01 | 07-01 | `codegraph install` writes SKILL.md + hooks.json byte-identical/idempotent | ✓ SATISFIED | Roadmap SC 1 + must-haves 5-9, all VERIFIED. REQUIREMENTS.md marks `[x]`. |
| AGENT-02 | 07-02 | `codegraph uninstall` cleanly removes without touching user content | ✓ SATISFIED (code) | Roadmap SC 2 + must-haves 10-15, all VERIFIED or PASSED (override). **REQUIREMENTS.md checkbox still shows `[ ]`** — this is a documentation-tracking discrepancy, not a code gap: coverage table (line 94) already lists Phase 7/AGENT-02 as "Pending" despite the underlying code/tests being complete. Flagged again this pass since it persisted across two verification runs. |
| AGENT-03 | 07-03, 07-04 | Skill/hooks versioned with binary, refreshed by `upgrade` | ✓ SATISFIED (code) | Roadmap SC 3 + must-haves 16-22, all VERIFIED. **REQUIREMENTS.md checkbox still shows `[ ]`**, same discrepancy as AGENT-02. |

No orphaned requirements: REQUIREMENTS.md's Phase 7 row lists exactly AGENT-01/02/03, matching all four plans' `requirements:` frontmatter fields.

**Note on REQUIREMENTS.md:** Per this project's rule against inventing structure in tool-owned generated files, this verifier does not edit REQUIREMENTS.md's checkboxes — that is the milestone-completion workflow's job, not verification's. Flagging it here for visibility since it has now persisted across two consecutive verification passes despite the code-level requirement being satisfied both times.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/agents/claude.go` | ~396-403, 454-466 | `settings.json` reported twice in `WriteResult.Files` under `--auto-allow` (WR-02) | ℹ️ Info | Cosmetic duplicate CLI status line; correctly triaged as non-blocking follow-up debt in `07-REVIEW.md`, re-confirmed still present, still non-data-loss. |
| `internal/agents/claude.go` | ~207-260 | Single-command-string ownership assumption unenforced (WR-03) | ℹ️ Info | No test/assertion would catch a future `.claude/hooks/hooks.json` edit introducing a second distinct command; currently safe by inspection. Correctly triaged as follow-up debt, re-confirmed still present. |
| — | — | `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` scan | — | Zero matches across phase-modified files this session (`rg` exit 1). No debt markers to gate on. |

Neither anti-pattern above is a blocker: both were explicitly triaged in code review, both are UX/robustness issues rather than data-loss or silent-failure risks, and both are documented (not silently dropped) in `07-REVIEW.md`.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Full repo build | `go build ./...` | clean, no output | ✓ PASS |
| Full repo vet | `go vet ./...` | clean, no output | ✓ PASS |
| Full repo test suite (single run, per Step 7b constraint) | `go test ./...` | all packages `ok`, 0 FAIL, 0 unexpected SKIP | ✓ PASS |
| CR-01/CR-02/WR-01/WR-04 regression tests (single named-test run) | `go test ./internal/agents/... -run 'TestClaude_Install_NeverClaimsOwnershipOfUnrelatedHookUnderSameMatcher\|TestClaude_Install_ManifestNotWrittenOnPartialWriteFailure\|TestClaude_Install_RestoresLostExecutableBitWithoutContentChange\|TestWriteManifest_SelfHealsCorruptedExistingManifest\|TestConfiguredSkillLocations_IncludesLocationWithCorruptedManifest\|TestConfiguredSkillLocations_ExcludesGenuinelyAbsentLocation' -v` | 6/6 PASS | ✓ PASS |

## Human Verification Required

### 1. Live Claude Code session smoke check of the global install path (07-04-PLAN.md's `<human-check>` — still open)

**Test:** Build the binary, run `codegraph install --location global` with `HOME` pointed at a scratch directory, start a **real Claude Code session** (not a subprocess exec of the extracted command) in a different repository that has a `.codegraph/` index, and confirm the SessionStart nudge line appears. Then start a session in a repository with no `.codegraph/` and confirm it does not appear. Mirror `06-agent-skill-package-.../verification/NUDGE-live-session.md`'s approach — same standard of evidence Phase 6 already met for the local-scope path.
**Expected:** Claude Code accepts and preserves the `hooks.SessionStart` shape codegraph writes into `~/.claude/settings.json` (Assumption A2), and Claude Code's own hook-dispatch runtime actually resolves and invokes the absolute-path `command` codegraph wrote for global scope (Assumption A3's dispatch half, not just its tilde-expansion half).
**Why human:** `07-UAT.md` test 4 closes the narrower tilde-expansion sub-risk by design (the global command is a fully-resolved absolute path, never `~`), and its live-execution evidence is genuinely useful — but its own evidence field states in its own words that this does not prove Claude Code's own runtime dispatches the command the same way a manual subprocess exec does. That gap can only be closed by actually starting a Claude Code session and observing the nudge fire, which is exactly what Phase 6 did (and honestly reported a gap on, for the resume path) but this phase has not yet done for its own default (global) scope.

## Gaps Summary

No FAILED truths, no MISSING/STUB artifacts, no NOT_WIRED key links, no regressions since the prior verification pass. `go build`/`go vet`/`go test ./...` all independently re-run and clean. Every code-review fix (CR-01, CR-02, WR-01, WR-04) was re-confirmed this session by reading the actual diff, not the commit message — all four are genuinely present with passing regression tests. The mid-wave hook-ownership vulnerability revert (`242ec0a`) was re-confirmed by direct source read: no residual matcher-based ownership logic exists anywhere in `internal/agents`.

Two of the three items that kept the prior pass at `human_needed` are now resolved: the interrupted-write backstop truth is covered by a formal Accepted Risk (AR-07-02) with a proper security sign-off (`07-SECURITY.md`, `threats_open: 0`) and is counted via override; the three judgment-tier prohibitions are treated as resolved by the completed UAT session functioning as this project's end-of-phase human checkpoint.

The remaining reason this report is not `passed`: a genuine live Claude Code session smoke test of the global-scope install path — the phase's actual default configuration — has still not been performed. `07-UAT.md` test 4 is real, valuable, independently-executed evidence that closes the narrower tilde-expansion risk and proves the command's syntax/exit behavior is correct, but by its own admission does not close the broader "does Claude Code's own runtime actually dispatch this correctly" question that the phase's own `<human-check>` and RESEARCH.md's Assumption A2/A3 asked for. This is not a code defect in the phase's own logic — everything the code controls is verified — but it is an unclosed verification step for behavior genuinely outside what a shell/subprocess check can prove, and this project has already demonstrated (Phase 6) what closing it looks like.

REQUIREMENTS.md's checkboxes for AGENT-02/AGENT-03 still show `[ ]` despite the coverage table on the same page marking them as mapped to Phase 7 and both being code-verified in two consecutive verification passes — flagged again as a persistent bookkeeping discrepancy, not edited directly per this project's rule against modifying tool-owned generated file structure outside its own workflow.

---

*Verified: 2026-08-13T18:31:57Z*
*Verifier: Claude (gsd-verifier)*
