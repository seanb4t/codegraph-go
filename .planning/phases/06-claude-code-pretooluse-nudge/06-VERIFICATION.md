---
phase: 06-claude-code-pretooluse-nudge
verified: 2026-09-19T12:25:49Z
status: passed
score: 9/9 truths verified; 1 item routed to human verification (not counted against the score)
covered_files:

  - .claude/hooks/hooks.json
  - .claude/hooks/pretooluse-nudge.sh
  - .claude/settings.json
  - .planning/phases/06-claude-code-pretooluse-nudge/06-01-PLAN.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-01-SUMMARY.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-02-PLAN.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-02-SUMMARY.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-03-PLAN.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-03-SUMMARY.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-04-PLAN.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-04-SUMMARY.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-05-PLAN.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-05-SUMMARY.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-06-PLAN.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-06-SUMMARY.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-07-PLAN.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-07-SUMMARY.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-CONTEXT.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-LIVE-SESSIONS.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-MUTATION-LOG.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-PATTERNS.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-RESEARCH.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-REVIEW-FIX.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-REVIEW.md
  - .planning/phases/06-claude-code-pretooluse-nudge/06-VALIDATION.md
  - claudeassets.go
  - docs/CLI-REFERENCE.md
  - internal/agents/capabilities.go
  - internal/agents/capabilities_test.go
  - internal/agents/claude.go
  - internal/agents/claude_pretooluse.go
  - internal/agents/claude_pretooluse_lifecycle_test.go
  - internal/agents/claude_pretooluse_test.go
  - internal/agents/claude_skillpackage_test.go
  - internal/agents/hookpackage_test.go
  - internal/agents/manifest.go
  - internal/agents/ownership_test.go
  - internal/agents/shared.go
  - internal/agents/shared_test.go
  - internal/agents/skillshared.go
  - internal/agents/types.go
  - internal/cli/hook_pretooluse.go
  - internal/cli/hook_pretooluse_test.go
  - internal/cli/install.go
  - internal/cli/install_test.go
  - internal/cli/root.go
  - internal/cli/testdata/cli-reference-allowlist.txt
  - internal/cli/testdata/plain/uninstall-local.golden
  - internal/cli/uninstall.go
  - internal/cli/upgrade.go
  - internal/cli/upgrade_test.go
  - internal/mcp/skill_claims_drift_test.go
  - internal/nudge/classify.go
  - internal/nudge/classify_test.go
  - internal/nudge/cooldown.go
  - internal/nudge/cooldown_test.go
  - internal/nudge/testdata/false-positives.json
  - internal/nudge/testdata/true-positives.json
  - internal/nudge/text.go

covered_digest: "v1:sha256:ab70d0411fad85c40db29480ebf8a891c6090e5ad7ab41d6fc87ad3094b9cf25"
behavior_unverified: 0
overrides_applied: 0
human_verification:

  - test: "In an environment where Claude Code exposes a native `Grep` tool (the maintainer's global config in the 06-06 live session did not), run the same D-18 protocol and confirm the `Grep` and `Glob` PreToolUse blocks actually fire `additionalContext` identically to the proven `Bash(rg *)`/`Bash(find *)`/`find` path."
    expected: "The first qualifying `Grep`/`Glob` tool call in a fresh, indexed session produces exactly one `hook_additional_context` attachment carrying the pinned nudge text, with the same cooldown and silence properties already proven live for Bash."
    why_human: "Claude Code's own tool matching and delivery are never unit-testable (D-00, this phase's own standing rule). The 06-06 live session (`06-LIVE-SESSIONS.md`) is thorough and honest about this: the maintainer's global Claude Code configuration exposes no `Grep` tool, so every live search went through Bash `rg`/`find`, and the `Grep`/`Glob` PreToolUse blocks were exercised only by `TestPreToolUseRegistrationShape` (byte-exact registration) and by the harness-neutral unit tests in `internal/cli/hook_pretooluse_test.go` (`TestHookPreToolUse_GrepFiresPinnedContext`), never by an actual Claude Code `Grep`/`Glob` tool call. The registration bytes and the adapter's internal handling of `Grep`/`Glob` are provably identical in shape to the proven `Bash`/`Read` paths (same guard script, same subcommand, same envelope, same cooldown gate), and the fetched hooks reference states matcher grammar is exact-string matching for every tool name — so there is strong indirect evidence this generalizes — but it has not been directly observed."
---

# Phase 6: Claude Code PreToolUse Nudge Verification Report

**Phase Goal:** In a `.codegraph/`-indexed repo, Claude Code is pointed at `codegraph_explore` the first time it reaches for grep/find/Read — as added context that never denies or blocks a tool call — then at most once a minute per session and per subagent, with zero overhead in an un-indexed repo, and registered and removed by `codegraph install`/`uninstall` as an opt-in through the existing exact-identity hook writer.
**Verified:** 2026-09-19T12:25:49Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | NUDGE-03: A PreToolUse hook on `Bash` (grep/rg/find first-word), `Grep`, `Glob`, `Read` in an indexed repo returns only `hookSpecificOutput.additionalContext` pointing at `codegraph_explore`, exits 0 unconditionally, never emits `permissionDecision`, matcher grammar/output shape verified against the current hooks reference | ✓ VERIFIED | `06-LIVE-SESSIONS.md` quotes the fetched `hooks.md` (sha256 `e0a14d…`, fetched 2026-09-19) verbatim for `additionalContext`, the `if` field, matcher grammar, exit-2 blocking, default timeout. `TestPreToolUseRegistrationShape` pins the exact 6-block/1-handler-each byte shape in both `.claude/settings.json` and `.claude/hooks/hooks.json` (reproduced live: `jq` dump matches). Real-binary reproduction (this verification, see below) fired the pinned JSON on `Grep` stdin and exited 0; `hook_pretooluse.go` never sets a permission-decision key (source read). See human-verification item below: the `Grep`/`Glob` matchers were not exercised by an actual Claude Code tool call in the 06-06 live session, only by unit tests and byte-shape pinning. |
| 2 | NUDGE-04: fires on first matched call, then at most once a minute per session and separately per subagent, via a session-scoped sentinel; silent with zero overhead (one directory check, no binary) in an un-indexed repo | ✓ VERIFIED | `internal/nudge/cooldown.go` read directly: `CooldownWindow = 60 * time.Second`, `SessionKey` keys on `(sessionID, agentID-or-"main")`, symlink-safe sentinel with `O_NOFOLLOW`+`Futimes`. `06-LIVE-SESSIONS.md` C1-C5 verdicts PASS with verbatim fire-timestamp table (gaps 60.4-143.2 s; subagent keys fire independently; `/clear` re-fires). This verification reproduced live: opt-in guard fired once then stayed silent on an immediate repeat call with the same session id; guard on a repo with `.codegraph/` removed printed nothing, exit 0, matching the guard script's `[ -d … ] || exit 0` first line (read directly). |
| 3 | NUDGE-05: validated against a false-positive corpus (legitimate grep use) and a true-positive corpus (where-is-X), fire rate measured in a genuinely fresh live session | ✓ VERIFIED | Re-ran `TestCorporaShape` myself: `true-positive corpus: 16/18 fire; false-positive corpus: 3/22 fire (accepted, noted)` — byte-identical to the SUMMARY's claimed rate. `06-LIVE-SESSIONS.md` records live "Matched calls: 18 / Fires: 9 / Fire rate: 9/18 / True-positive fires: 7/9" with a verbatim fire log and timestamped events. |
| 4 | NUDGE-06: `install`/`uninstall` register/remove the hook through `writeHookEntry`/`removeHookEntry` as an opt-in; a hand-edited own entry duplicates rather than overwrites; an unrelated `PreToolUse` entry under the same event survives | ✓ VERIFIED | `TestOwnershipExactIdentity`: re-ran myself, 32/32 PASS (all 8 harnesses × 2 scopes × clean/foreign-dir), Claude leaves plant an unrelated same-matcher `PreToolUse` block that survives install and uninstall byte-identical. Independent real-binary reproduction (this verification): opt-in install wrote guard + 6 blocks; default install wrote neither; hand-editing one `Bash(rg *)` handler's command and reinstalling produced 7 blocks (6 own + the edited one, byte-identical); `uninstall` removed the guard and all 6 owned blocks while leaving the hand-edited block untouched. |
| 5 | This repository dogfoods the exact registration it ships: `.claude/settings.json` and the embedded fragment `.claude/hooks/hooks.json` are deep-equal for `hooks.PreToolUse` (D-13) | ✓ VERIFIED | `jq -S '.hooks.PreToolUse'` on both files, diffed myself: identical. `TestHookRegistrationMatchesFragmentAndScript/PreToolUse` and `TestPreToolUseRegistrationShape/{settings.json,hooks.json}` re-run, PASS. |
| 6 | The hidden `codegraph hook pretooluse` subcommand is unreachable from `cmd/codegraph/main.go`'s exit-1 path (`RunE` always returns nil, panics recovered) and is hidden from `--help`/`commandGroups` | ✓ VERIFIED | Source read (`internal/cli/hook_pretooluse.go`): `defer func(){ _ = recover() }()` is the first statement in `RunE`, which always `return nil`. Built the real binary and ran `--help`: `hook` does not appear (only unrelated `githooks`). Both allowlist lines (`codegraph hook`, `codegraph hook pretooluse`) present in `internal/cli/testdata/cli-reference-allowlist.txt`; `docs/CLI-REFERENCE.md` unaffected (drift gate green, see below). |
| 7 | Code review converged clean; the CR-01 (settings.json evidence when manifest absent), WR-01 (sentinel-dir permission re-check), WR-02 (shared ownership-identity helpers), WR-03 (doc-comment accuracy) findings are actually fixed in the current source | ✓ VERIFIED | `06-REVIEW.md` (iteration 3, deep, 34 files): 0 critical/warning/info findings, status clean. Read `internal/agents/claude_pretooluse.go`'s `preToolNudgeEvidenced` directly: implements the CR-01 widening exactly as `06-REVIEW-FIX.md` describes (manifest-absent-but-settings.json-owns-a-block case). Read `internal/nudge/cooldown.go`'s `Due`: re-checks `info.Mode().Perm() != 0o700` on every call (WR-01), not only at directory creation. `internal/agents/shared.go`'s `blockOwnsAnyCommand`/`commandIsOwned` helpers present and used at all three call sites named in the review (WR-02). |
| 8 | Phase gate: `go build ./...` and the full suite are green (≥ 53 `ok` packages excluding `internal/daemon`, then `internal/daemon` alone), `task docs:cli:drift` exits 0, 06-MUTATION-LOG.md holds all 19 family sections with RED evidence and clean reverts, no debt markers in phase-touched files | ✓ VERIFIED | Independently re-ran: `go build ./...` clean; full suite minus `internal/daemon` = 53 `ok`, 0 FAIL; `internal/daemon` alone = `ok` (64.4s); `task docs:cli:drift` exit 0 ("byte-identical to a fresh regeneration"); `rg -c '^## Family \([a-e][1-6]\)' 06-MUTATION-LOG.md` = 19, spot-checked Family (c3)'s structure (pre-mutation gate, diff, RED transcript, revert, green control) — matches the claimed rigor. No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` found in any phase-touched production or script file. |
| 9 | `install`/`uninstall` help text documents the opt-in and its removal; `docs/CLI-REFERENCE.md` reflects it | ✓ VERIFIED | Built the real binary: `install --help` names `--pretool-nudge`, states it never blocks a tool call and is remembered until `--pretool-nudge=false` or uninstall; `uninstall --help` names removing "the Claude Code PreToolUse nudge hook and its guard script." `docs/CLI-REFERENCE.md` regenerated via `task docs:cli:drift` (exit 0, confirmed above). |

**Score:** 9/9 truths verified (0 present-but-behavior-unverified; 1 separate item routed to human verification, not counted against the score per the framework — see below)

### Human-visible / environment-constrained item (not a code gap)

The maintainer's global Claude Code configuration exposes no native `Grep` tool, so the 06-06 live session's every search went through Bash `rg`/`find`. The `Grep` and `Glob` PreToolUse blocks are byte-verified (registration shape, dogfood deep-equality) and unit-verified (the harness-neutral adapter treats them identically to `Bash`/`Read`), but **Claude Code's own act of matching and delivering a PreToolUse hook specifically for the `Grep`/`Glob` tools was never observed live** — this is exactly the class of fact D-00 says a unit test cannot establish, and the phase's own `06-LIVE-SESSIONS.md` and `06-07-SUMMARY.md` (advisory #12) disclose this openly rather than paper over it. Given the hooks reference's matcher grammar (exact-string tool-name matching, no special-casing) and the code-level proof that `Grep`/`Glob` route through the exact same guard/subcommand/envelope as the proven `Bash` path, the residual risk is low — but it is unproven, not merely unlikely, so it is routed to human verification rather than marked `✓ VERIFIED`.

This does not, on its own, contradict any ROADMAP success-criterion text: SC1 asks for the matcher grammar/output shape to be "verified against the current hooks reference," which was done via the dated doc citation, not for every one of the four registered matchers to be exercised by an actual live tool call.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/nudge/classify.go` | `Qualifies` (D-02/D-03), pinned corpora-backed classifier | ✓ VERIFIED | Read source; first-word shell rule with assignment-skip and parse-doubt-silent exactly as specced; Read extension exclusion list exact match. |
| `internal/nudge/cooldown.go` | `CooldownWindow`, `SessionKey`, `Gate.Due`, symlink-safe sentinel | ✓ VERIFIED | Read source; matches D-05..D-08 exactly, including the WR-01 permission re-check. |
| `internal/nudge/text.go` | pinned `Text` constant | ✓ VERIFIED | Byte-matches the string reproduced live in `06-LIVE-SESSIONS.md` and in this verification's own guard invocation. |
| `internal/cli/hook_pretooluse.go` | hidden `hook pretooluse`, never-erroring `RunE`, cooldown-gated envelope | ✓ VERIFIED | Read source; recover-first, classify-before-sentinel-IO, env-then-stdin session id, pinned JSON output only. |
| `.claude/hooks/pretooluse-nudge.sh` | embedded/dogfooded POSIX guard | ✓ VERIFIED | Read source; D-04 directory check first, token-based ExecPath render with dogfood PATH fallback, exit 0 unconditional tail. |
| `internal/agents/claude_pretooluse.go` | render/path/lifecycle helpers, CR-01 widening | ✓ VERIFIED | Read source; `renderPreToolGuard`, `shellSingleQuote`, `preToolNudgeEvidenced` all present and match the documented behavior. |
| `.claude/settings.json` / `.claude/hooks/hooks.json` | dogfooded 6-block/1-handler PreToolUse registration | ✓ VERIFIED | `jq` diff: byte-identical `hooks.PreToolUse` in both files. |
| `internal/agents/manifest.go` | sticky opt-in manifest keys | ✓ VERIFIED | `TestPreToolNudge_*` (10 tests) re-run, PASS; real-binary reproduction confirms Keep/On/Off behavior. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `internal/cli/hook_pretooluse.go runHookPreToolUse` | `internal/nudge.Gate.Due` | gate consulted only after `Qualifies`/`SessionKey` succeed | ✓ WIRED | Read source; order confirmed. |
| `internal/nudge/cooldown.go recordFire` | `syscall.O_NOFOLLOW` + `syscall.Futimes` | fd-based mtime update, no path re-resolution | ✓ WIRED | Read source; confirmed. |
| `internal/cli/install.go RunE` | `agents.InstallOptions.PreToolNudge` | `cmd.Flags().Changed("pretool-nudge")` tri-state | ✓ WIRED | `TestInstall_PreToolNudge_StickyAcrossPlainInstall`/`ExplicitFalseRemoves`/`OptInRegisters` re-run, PASS; real-binary reproduction confirms. |
| `internal/cli/upgrade.go refreshInstalledSkills` | `agents.PreToolNudgeKeep` | explicit pass-through so a moved binary's guard refreshes | ✓ WIRED | `TestRefreshInstalledSkills_CarriesPreToolNudge` re-run, PASS (both subtests). |
| `.planning/phases/06-claude-code-pretooluse-nudge/06-LIVE-SESSIONS.md` `D-18 verdict:` line | 06-07-PLAN.md phase gate | precondition read by the next plan | ✓ WIRED | `D-18 verdict: PASS` present, all 7 criteria PASS with excerpts. |

### Behavioral Spot-Checks (this verification, real binary, not from SUMMARY claims)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Opt-in install writes guard + 6 PreToolUse blocks | `codegraph install --target claude --location local --pretool-nudge` in an indexed scratch repo | guard + 6 blocks written | ✓ PASS |
| Default install writes neither | `codegraph install --target claude --location local` in an un-indexed scratch repo | `hooks.PreToolUse` absent, no guard file | ✓ PASS |
| Guard fires once then cools down | piped Grep-event stdin through the real guard twice, same session id | fire 1: pinned JSON, exit 0; fire 2: empty, exit 0 | ✓ PASS |
| Guard silent on non-qualifying Read | Read `.md` path piped through guard | empty, exit 0 | ✓ PASS |
| Hand-edited handler duplicates on reinstall | edited `Bash(rg *)` command via `jq`, reinstalled with `--pretool-nudge` | 7 blocks (6 own + 1 edited), edited block byte-identical | ✓ PASS |
| Uninstall removes owned blocks, leaves hand-edit | `codegraph uninstall --target claude --location local` after the above | guard removed; 5 codegraph blocks gone; hand-edited block survives byte-identical | ✓ PASS |
| `codegraph hook` hidden from `--help` | `codegraph --help \| grep -i hook` | only unrelated `githooks` listed | ✓ PASS |
| `task docs:cli:drift` | `task docs:cli:drift` | "byte-identical to a fresh regeneration" | ✓ PASS |
| `install`/`uninstall --help` document the opt-in | `codegraph install --help`, `codegraph uninstall --help` | matches must-have text | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|--------------|--------|----------|
| NUDGE-03 | 06-01, 06-02, 06-03, 06-05, 06-06 | additionalContext-only PreToolUse hook, verified against current hooks reference | ✓ SATISFIED (with one human-verification note on Grep/Glob live delivery) | See Truth #1 |
| NUDGE-04 | 06-01, 06-03, 06-06 | first-fire-then-per-minute cooldown per session/subagent, zero overhead un-indexed | ✓ SATISFIED | See Truth #2 |
| NUDGE-05 | 06-02, 06-03, 06-06 | false/true-positive corpora + live fire-rate measurement | ✓ SATISFIED | See Truth #3 |
| NUDGE-06 | 06-01, 06-04, 06-05, 06-07 | install/uninstall opt-in via exact-identity writer, hand-edit duplication | ✓ SATISFIED | See Truth #4 |

No orphaned requirements: `.planning/REQUIREMENTS.md` maps exactly NUDGE-03..06 to Phase 6, all four are declared across the phase's plans (`requirements:` frontmatter) and all four are marked `Complete` in the REQUIREMENTS.md traceability table.

### Anti-Patterns Found

None. Scanned every phase-touched production/script file (`internal/nudge/*`, `internal/cli/hook_pretooluse.go`, `internal/agents/claude_pretooluse.go`, `internal/agents/claude.go`, `internal/agents/manifest.go`, `internal/agents/skillshared.go`, `internal/agents/capabilities.go`, `internal/cli/install.go`, `internal/cli/upgrade.go`, `internal/cli/uninstall.go`, `.claude/hooks/pretooluse-nudge.sh`) for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` — zero matches.

### Human Verification Required

#### 1. Live Grep/Glob PreToolUse delivery

**Test:** In a Claude Code session whose configuration exposes a native `Grep` tool, repeat the D-18 protocol's prompt 1 using the `Grep` tool explicitly (and similarly for `Glob`), in a fresh indexed scratch repo with the opt-in installed.
**Expected:** The first qualifying `Grep`/`Glob` call produces exactly one `hook_additional_context` attachment carrying the pinned nudge text, with the same 60-second per-(session, agent) cooldown already proven for the `Bash` path.
**Why human:** Claude Code's own tool-matching and hook-delivery behavior is not unit-testable (D-00, this phase's own standing rule), and the 06-06 live session's environment (the maintainer's global Claude Code configuration) exposed no `Grep` tool, so every live search went through Bash `rg`/`find` instead. The `Grep`/`Glob` registration bytes and the harness-neutral adapter code are proven correct and are structurally identical to the proven `Bash`/`Read` paths, and the hooks reference's matcher grammar gives good reason to expect this generalizes — but it has not been directly observed, so it cannot be marked verified on inspection alone.

### Gaps Summary

No gaps. All four ROADMAP success criteria (NUDGE-03..06) and every must-have truth checked in the phase's seven plans hold against the actual codebase, confirmed independently in this verification via direct source reading, an independent full-suite re-run (`internal/daemon` run separately per instructions), and live reproduction against a freshly built binary (opt-in/default install, fire/cooldown/silence, hand-edit duplication, uninstall, hidden-command check, help text, docs-drift gate). The phase's own code review converged clean at iteration 3 after real fixes (CR-01, WR-01, WR-02, WR-03), independently confirmed against the current source rather than trusted from the fix report. The only open item is the disclosed, environment-constrained inability to observe live Claude Code delivery of the `Grep`/`Glob` PreToolUse matchers specifically (the maintainer's own Claude Code configuration lacks a `Grep` tool) — routed to human verification rather than either passed or failed, since it is a genuine unknown about a third party's runtime behavior, not a defect in this codebase.

---

_Verified: 2026-09-19T12:25:49Z_
_Verifier: Claude (gsd-verifier)_
