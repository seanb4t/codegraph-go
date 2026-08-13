---
phase: 07
slug: codegraph-install-skill-hooks-distribution-claude-code
status: verified
threats_open: 0
asvs_level: 1
created: 2026-08-13
---

# Phase 07 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|----------------|
| binary → user's `~/.claude/` and `./.claude/` | codegraph writes into directories owned by a third-party tool and co-inhabited by user-authored content | file writes into a shared, partially-user-owned tree |
| existing on-disk `settings.json` → codegraph's read-modify-write | untrusted-shape input (a file codegraph did not write and cannot assume is well-formed) crosses into a write path that can delete content | JSON of unknown provenance/validity |
| embedded FS → installed artifacts | build-time content crosses into the user's runtime execution surface (a shell script Claude Code will execute) | shell script content, executable bit |
| on-disk manifest → codegraph's decision logic | a file any local process can rewrite is read back and its recorded hashes influence reporting and refresh behavior | sha256 hashes, version string |
| shared `settings.json` → the manifest's hooks hash | only a fragment of a co-inhabited file is codegraph's own, and the hash must not overreach | JSON sub-tree |
| user-authored files in `~/.claude/skills/codegraph/` → codegraph's removal path | content codegraph did not create sits inside a directory codegraph names | filesystem entries |
| the newly swapped binary → the user's agent configuration | a freshly downloaded binary's embedded content is written into the user's `~/.claude/` without a second confirmation | embedded SKILL.md/script/hooks content |
| manifest presence → refresh consent | the on-disk manifest is the only record of what the user previously agreed to configure | boolean configured/not-configured signal |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-07-01 (Plan 01) | Tampering | `writeHookEntry` read path over `claudeSettingsPath(loc)` | high | mitigate | `readJSONFileStrict` errors on unreadable/undecodable content; caller writes nothing, file left byte-untouched. Verified live this session (corrupted `settings.json`, real `install` run, file untouched) in addition to `TestClaude_SettingsReadFailureNeverDestroysContent`. | closed |
| T-07-01 (Plan 02) | Tampering | `readJSONFileStrict` callers over `claudeSettingsPath(loc)` — hooks step *and* AutoAllow step | high | mitigate | Both steps share one strict read posture; `TestClaude_AutoAllowSharesStrictReadPosture` proves one step cannot fail loud while the other silently overwrites. | closed |
| T-07-01 (Plan 03) | Tampering | `hashOwnedHookBlocks` scope | medium | mitigate | Hash covers only codegraph's own SessionStart blocks, never the whole shared file. `TestManifest_OwnedHookBlockHashIgnoresUnrelatedSettings` verified passing. | closed |
| T-07-02 (Plan 03) | Tampering | `readManifest` over `.codegraph-manifest.json` | medium | mitigate | Built on `readJSONFileStrict`; an unreadable/undecodable manifest returns an error, never a zero-valued fallback acted on as real. Note: `writeManifest`'s WR-01 fix (code review, commit `c2ea00f`) self-heals a corrupted manifest by overwriting rather than blocking — a *different* code path than `readManifest`'s own read contract, which is unchanged; the self-heal is a deliberate D-05-consistent choice for a wholly codegraph-owned file, not a regression of this mitigation. | closed |
| T-07-02 (Plan 04) | Tampering | post-swap refresh scope | medium | mitigate | Refresh runs only for locations already carrying a readable manifest (`ConfiguredSkillLocations`), discovered by two fixed `os.Stat`-class probes, never a filesystem walk. Note: WR-04 (code review) extended this to also include a *present-but-corrupted* manifest, since that is still proof of prior configuration — verified via `TestConfiguredSkillLocations_IncludesLocationWithCorruptedManifest` / `TestConfiguredSkillLocations_ExcludesGenuinelyAbsentLocation`. | closed |
| T-07-03 (Plan 01) | Elevation of Privilege | installed `session-nudge.sh` file mode | medium | mitigate | `atomicWriteExecutableFile` sets `0o755`, owner-writable only. CR-02 (code review) closed a self-heal gap: a lost executable bit with unchanged content now restores on next install/upgrade (`TestClaude_Install_RestoresLostExecutableBitWithoutContentChange`), not just at first install. | closed |
| T-07-04 (Plan 01) | Spoofing | `command` string written into `hooks.SessionStart` | medium | mitigate | Always a fully-resolved absolute path (global) or `${CLAUDE_PROJECT_DIR}`-anchored (local), never a bare filename. Verified live this session: extracted the real global-scope command and executed it directly — correct output, no PATH ambiguity possible. | closed |
| T-07-04 (Plan 02) | Spoofing | ownership identity used for removal | medium | mitigate | Blocks matched by exact `command` string only, never `matcher` value — this is the exact property that failed during Wave 3 execution (an ownership-recovery heuristic briefly weakened it to matcher+shape) and was reverted the same day after independent security review; `TestClaude_Install_NeverClaimsOwnershipOfUnrelatedHookUnderSameMatcher` now pins it directly. | closed |
| T-07-04 (Plan 04) | Spoofing | `ExecPath` written into the refreshed MCP entry | low | mitigate | `os.Executable()`'s resolved path, never a `$PATH` lookup — confirmed by direct source read (`internal/cli/upgrade.go:107`), same convention as `codegraph install`. | closed |
| T-07-05 (Plan 02) | Tampering | uninstall's skill-directory cleanup | high | mitigate | Per-file removal only; directory removed via `os.Remove` (fails harmlessly if non-empty) — no recursive delete anywhere in the phase. `TestClaude_Uninstall_PreservesUserFileInSkillDir` verified passing. | closed |
| T-07-05 (Plan 03) | Tampering | uninstall's manifest removal ordering | medium | mitigate | Manifest removed before SKILL.md so the subsequent empty-directory sweep sees an already manifest-free directory — confirmed by direct source read (`internal/agents/claude.go:538-559`). | closed |
| T-07-06 (Plan 01) | Information Disclosure | `//go:embed` pattern scope | low | mitigate | Three exact file patterns, never a directory pattern — Phase 6's `verification/` rehearsal transcripts never ship. `TestClaudeAssets_EmbedsNoVerificationTranscripts` verified passing. | closed |
| T-07-07 (Plan 03) | Repudiation | manifest hash mistaken for tamper-evidence | low | accept | Documented in `manifest.go`'s package comment as an advisory drift signal only; D-05 fixes the response as a silent overwrite, never a security event. The file carries no signature and any local process can rewrite it, so a tamper-detection claim would be false — accepting this is the honest disposition. | closed (accepted) |
| T-07-08 (Plan 01/02) | Denial of Service | interrupted write / concurrent invocation | low | accept | `fsatomic.WriteFile`'s temp-file-plus-rename guarantees no truncated artifact survives an interrupt. Concurrent install/uninstall against the same location is an explicit non-goal, not an unnoticed gap (this is the one item the goal-backward verifier flagged as a `backstop` truth with no interruption-simulating test — accepted per the plan's own disposition, not silently dropped). | closed (accepted) |
| T-07-09 (Plan 04) | Elevation of Privilege | refresh runs with the freshly downloaded binary's embedded content | medium | mitigate | Refresh runs strictly after `internal/upgrade`'s existing cosign-keyless verification and atomic swap succeed — the embedded content written is by construction from a binary that already cleared release verification. No new trust extended. | closed |
| T-07-10 (Plan 04) | Repudiation | conflating swap success with refresh failure | medium | mitigate | D-07 splits the two outcomes: swap reporting unchanged, refresh failure is a distinct warning naming `codegraph install`. Verified live this session: corrupted `settings.json`, re-ran the exact refresh call — failed loud without corrupting other state, matching the code's split-reporting design (confirmed by direct source read of `RunE`). | closed |
| T-07-SC (all plans) | Tampering | npm/pip/cargo installs | low | accept | This phase installs no package via any package manager — every mechanism is Go standard library (`embed`, `encoding/json`, `os`, `crypto/sha256`) already used elsewhere in this repository. Package Legitimacy Gate skipped by its own scope condition. | closed (accepted) |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `high` (workflow.security_block_on) count toward `threats_open`*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|--------------|------|
| AR-07-01 | T-07-07 | Manifest hash is a drift signal only, never tamper-evidence — the file has no signature and any local process can rewrite it; claiming otherwise would be false. | Locked in CONTEXT.md D-05/RESEARCH.md Pattern 4; re-affirmed at plan time (07-03-PLAN.md). | 2026-08-13 |
| AR-07-02 | T-07-08 | Interrupted-write atomicity is inherited from `fsatomic.WriteFile`'s existing temp-file-plus-rename guarantee; concurrent install/uninstall against one location is an explicit non-goal, not a silent gap. Flagged as a `backstop` truth by the goal-backward verifier — accepted, not overlooked. | Locked at plan time (07-01/07-02-PLAN.md); re-affirmed by verifier (07-VERIFICATION.md). | 2026-08-13 |
| AR-07-03 | T-07-SC | No package-manager install anywhere in this phase; every mechanism is Go standard library already vendored/used elsewhere in this repository. | Locked at plan time (all 4 PLAN.md files). | 2026-08-13 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|----------------|--------|------|--------|
| 2026-08-13 | 17 | 17 | 0 | claude (orchestrator, L1 grep-depth per ASVS-1 short-circuit — several mitigations independently verified far beyond L1 depth during execution: live CLI runs, RED→GREEN regression tests, and direct source reads, not merely grep) |

**Note on process:** This phase's own execution surfaced a real vulnerability (T-07-04 (Plan 02) — the exact threat this register already named) mid-Wave 3, independently found via a background security review, confirmed RED, and reverted the same day (commit `242ec0a`). A subsequent deep code review found 2 CRITICAL + 4 WARNING findings; 2 CRITICALs (T-07-03 (Plan 01), and the manifest-write CR-01 finding which is a correctness issue, not itself a new STRIDE threat — recorded as a code-review finding, not a new threat register row) plus 2 WARNINGs (T-07-02 (Plan 03)'s self-heal extension, T-07-02 (Plan 04)'s corrupted-manifest inclusion) were fixed with regression tests in commit `c2ea00f`. Two lower-severity WARNINGs (duplicate status-line reporting; an unenforced single-command assumption) remain as documented follow-up debt in `07-REVIEW.md` — neither is a STRIDE threat against this register (no tampering/spoofing/DoS/etc. surface), both are UX/robustness gaps.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-13
