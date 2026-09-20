---
phase: "5"
slug: "agent-reach-capability-model-skill-in-every-harness"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-19"
---

# Phase 5 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| process environment → path resolution → terminal | `HOME`, `XDG_CONFIG_HOME`, `HERMES_HOME` feed every declared path. `--print-config-style` prints them, sanitized, on plain and styled output (05-01) | local path strings |
| CLI flags → `install` RunE | `--print-config-style` returns before `os.Executable`, the target switch and the picker, so it writes nothing (05-01) | argv |
| filesystem state under a skill root → skill writer | users, other tools and cloned repos can place content at `…/skills/codegraph`. Ownership is decided only by whether codegraph's sidecar manifest is present, never by name (05-02) | skill files, manifests |
| one requester's uninstall → another requester's package | the shared `.agents/skills/codegraph` package has one manifest with a `targets` list. It is deleted only when that list is empty (05-02) | shared skill package |
| user-managed skill-root symlinks → Claude's installer | the `npx skills` layout `~/.claude/skills/codegraph → ../../.agents/skills/codegraph` is resolved with `EvalSymlinks` and treated as one package (D-17, 05-03) | symlinks, skill package |
| manifest at Claude's path → `codegraph upgrade` refresh | refresh consent is inferred from the manifest's `targets` including `claude` (05-03) | manifest |
| user/tool content in agent configs and skill roots → install/uninstall | a foreign MCP entry, a foreign sibling skill, a foreign manifest-less `codegraph/` directory and a foreign instructions section all survive the round trip byte-identical (05-04) | agent config files |
| content under `~/.gemini`, `~/.kiro`, project `.gemini`/`.kiro` → install/uninstall | harness-specific skill dirs, including Antigravity's `~/.gemini/config/skills/codegraph` (1A), which sits beside the user's own skills (05-05, 05-07) | skill files |
| live session → the maintainer's real `$HOME` | the Antigravity session: backups, sha256 before and after, `cmp` restores, cleanup judged on the paths codegraph owns (3A) (05-06) | `mcp_config.json`, `GEMINI.md` |
| transcript → recorded verdict | the negative-space rule, negative controls, and a fixed verdict vocabulary with `Maintainer decision:` lines (05-06) | evidence text |
| repo-root `AGENTS.md` shared by several agents | only if a D-11 pickup is observed. The recorded verdict is `not probed`, so this boundary is not built (05-07) | instructions file |
| generated reference ↔ live Cobra tree | `task docs:cli:drift` + `TestEveryRegisteredFlagIsAccountedFor` (05-07) | `docs/CLI-REFERENCE.md` |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-05-01 | Tampering | `install --print-config-style` | medium | mitigate | read-only branch at `install.go:87-96` before any write; `TestInstallPrintConfigStyle_ReadOnly` (picker stub fatals, 0 entries in HOME/cwd, incl. `--yes`); real binary wrote 0 files | closed |
| T-05-02 | Tampering (terminal injection) | printed paths | low | mitigate | `sanitizePathForDisplay` on every printed path; styled branch also through `present.KV` → `sanitizeControl`; ESC/BEL probe leaked nothing | closed |
| T-05-03 | Tampering | table drift (DescribePaths/Detect copies) | medium | mitigate | `TestCapabilitiesTableDrivesDerivations`, `TestCapabilitiesMatchInstallWrites`; Families (a1)/(a2) RED at HEAD | closed |
| T-05-04 | Tampering | `antigravityConfigPath` narrowing | low | mitigate | unified path unless only legacy exists (`antigravity.go:123`); `TestCapabilitiesDeclared_AntigravityMigrationAware`; migration tests' bodies byte-unchanged | closed |
| T-05-SC | Tampering | package installs | low | accept | no dependency added (`go.mod`/`go.sum` diff empty across the phase) | closed |
| T-05-05 | Tampering / EoP | skill-package ownership (the `242ec0a` class) | high | mitigate | single presence-only ownership signal (`skillshared.go:88`); foreign dir `kept (foreign)` on install and uninstall; every write through `writeSkillFile`; Family (b1) RED | closed |
| T-05-06 | DoS / Tampering | one harness's uninstall deleting a shared package | high | mitigate | delete only when `remaining` is empty (`skillshared.go:344-360`); `TestSharedSkillPackage_TargetsInvariantOverAllSequences` (1554 sequences); CR-01 fallback scoped via `claudeSkillDirPath` (`TestCursor_CorruptedManifestAtSharedDir_*`, `TestGemini_CorruptedManifestAtHarnessExclusiveDir_*`) | closed |
| T-05-07 | Tampering | `removeSkillDirIfEmpty` unlinking a user symlink | medium | mitigate | Lstat guard `shared.go:507-509`; `TestRemoveSkillDirIfEmpty_NeverUnlinksSymlink` | closed |
| T-05-08 | Tampering | writes following a symlink planted in a cloned repo | medium | mitigate (+ accepted residual) | a non-empty manifest-less dir is foreign; file writes are temp + `os.Rename` (`fsatomic.go`), so a file-level link is replaced, not written through; residual accepted in the 05-02 register | closed |
| T-05-09 | Repudiation | manifest hash read as authenticity | low | accept | documented as a drift signal, not tamper detection (`manifest.go:9-15`); `TestSharedSkillPackage_HandEditedOwnFileRewritten` (D-16) | closed |
| T-05-10 | Tampering | older binary rewrites a schema-2 manifest at schema 1 | medium | accept | reversibility "costly" (05-02-PLAN, CONTEXT D-07 amendment); legacy/corrupt self-heal pinned by `TestSharedSkillPackage_LegacyAndCorruptManifestReadAsClaude` | closed |
| T-05-11 | DoS / Tampering | `claudeTarget.Uninstall` in a symlinked layout | high | mitigate | last-requester rule (`claude.go:629-637`); `TestSymlinkedSkillDir_ClaudeUninstallKeepsOtherRequester` | closed |
| T-05-12 | Tampering | Claude install overwriting foreign content through a symlink | high | mitigate | `claudeSkillPolicy` → `refuseUnmanifested` when same dir (`claude.go:223-240`); `TestSymlinkedSkillDir_ForeignContentKeptForeign` | closed |
| T-05-13 | EoP (consent) | upgrade refresh installing Claude where only others wrote | medium | mitigate | `ConfiguredSkillLocations` requires `claude` ∈ `targets` (`manifest.go:297`); `TestConfiguredSkillLocations_RequiresClaudeInTargets` | closed |
| T-05-14 | Tampering | unlinking the user's symlink | medium | mitigate | Lstat guard; link asserted present after each uninstall in `claude_symlink_test.go` (see coverage note 1) | closed |
| T-05-15 | Tampering / EoP | Cursor/opencode shared skill writes | high | mitigate | declared-skill helpers with `refuseUnmanifested` (`capabilities.go:245,267`); `TestOwnershipExactIdentity` foreign leaves | closed |
| T-05-16 | Tampering | Claude SessionStart hook ownership by matcher | high | mitigate | exact command-string ownership (`shared.go:214-236`, `:381-388`); claude/* leaves reproduce the `242ec0a` precondition; Family (b2) RED | closed |
| T-05-17 | DoS | Cursor uninstall removing opencode's package | high | mitigate | `TestOpencode_CursorShareOnePackage` | closed |
| T-05-18 | Tampering | foreign MCP/instructions content altered by a round trip | medium | mitigate | every supported leaf plants foreign content and byte-compares it; unsupported leaves compare tree snapshots; 32/32 | closed |
| T-05-19 | Tampering | name-based deletion of legacy `.cursor/rules/codegraph.mdc` / `.kiro/steering/codegraph.md` | medium | accept | pre-existing self-heal, pinned by `TestCursor_Install_SelfHealsLegacyRulesFile` / `TestKiro_Install_SelfHealsLegacySteeringFile`; advisory in 05-07-SUMMARY | closed |
| T-05-20 | Tampering / EoP | Gemini/Kiro/Antigravity skill writes | high | mitigate | declared-skill helpers (refuse); `newSkillDirs` plants foreign dirs at `.gemini/skills`, `.kiro/skills` and both Antigravity roots | closed |
| T-05-21 | Tampering | writing an unverified path | low | mitigate | `TestAntigravity_Install_WritesConfigSkillDir` (no `antigravity-cli/skills`, no `~/.agents/skills`, no `~/.gemini/AGENTS.md`); `TestGemini_Install_WritesHarnessSkillDir`; `TestKiro_Install_WritesNoAgentsMd`. The written dir is the live-proven one (1A) | closed |
| T-05-22 | Info Disclosure / confusion | Kiro doubled instructions via AGENTS.md | low | accept | D-06(d); documented at `kiro.go:52-58`; 05-07-SUMMARY advisory 3 | closed |
| T-05-23 | Tampering | the maintainer's `~/.gemini` config, GEMINI.md, `.migrated`, skill dirs | medium | mitigate | backups + sha256 before and after, `cmp` restores, `.migrated` untouched, leftover dir removed; raw tree diff resolved by 3A (05-LIVE-SESSIONS.md) | closed |
| T-05-24 | Tampering | the user's `~/.claude/skills/codegraph` symlink | medium | mitigate | readlink before and after; `~/.agents`/`~/.claude` hashes unchanged; 05-06 gate `test -L` re-run PASS | closed |
| T-05-25 | Spoofing (of evidence) | verdicts without transcripts | medium | mitigate | negative-space rule and negative controls; fixed verdict vocabulary; `[ASSUMED]`/`not probed` with 5 `Maintainer decision:` lines | closed |
| T-05-26 | Info Disclosure | transcripts pasted into the repo | low | accept | gofixture scratch projects only; secret-pattern scan of 05-LIVE-SESSIONS.md empty | closed |
| T-05-27 | DoS / Tampering | `uninstallDeclaredInstructions` on a shared AGENTS.md | medium | mitigate | not applicable: D-11 verdict `not probed`, so the branch is not built. No two targets declare the same instructions path, and Cursor declares none (`TestCursor_Install_NoInstructionsFileWritten`) | closed |
| T-05-28 | Tampering | foreign content in AGENTS.md | medium | mitigate | marker-fenced upsert/remove (`shared.go:595-687`); planted foreign section byte-compared in every declared instructions file | closed |
| T-05-29 | Tampering | `docs/CLI-REFERENCE.md` drift | low | mitigate | `task docs:cli:drift` (CI `ci.yml:203`) + `TestEveryRegisteredFlagIsAccountedFor` | closed |
| T-05-30 | Tampering | foreign user skills beside codegraph in `~/.gemini/config/skills/` | medium | mitigate | only `…/skills/codegraph` declared and swept; `gh-stack` sibling byte-compared after install and uninstall; Family (c) RED | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-05-01 | T-05-SC | No dependency added in the phase; supply-chain surface unchanged (05-01-PLAN register, 05-RESEARCH) | plan-time register (maintainer-approved plan) | 2026-09-18 |
| AR-05-02 | T-05-08 (residual) | A planted directory link to a dir that already holds a codegraph manifest receives only codegraph's own embedded `SKILL.md`/manifest (05-02-PLAN register) | plan-time register | 2026-09-18 |
| AR-05-03 | T-05-09 | The manifest hash is a drift signal, not an authenticity control; a hand-edited own file is rewritten (D-16) | plan-time register | 2026-09-18 |
| AR-05-04 | T-05-10 | An older binary may downgrade a schema-2 manifest; the next current install self-heals it (D-07 amendment, reversibility "costly") | CONTEXT D-07 planner amendment (accepted) | 2026-09-18 |
| AR-05-05 | T-05-19 | Pre-existing name-based self-heal of legacy Cursor/Kiro files is kept unchanged; flagged as a candidate todo | plan-time register | 2026-09-18 |
| AR-05-06 | T-05-22 | Kiro may read the codegraph block twice via an AGENTS.md written by opencode/Codex; advisory, not changed (D-06(d)) | CONTEXT D-06(d) (accepted) | 2026-09-18 |
| AR-05-07 | T-05-26 | Transcripts in the evidence file come only from gofixture scratch projects; no secrets | plan-time register | 2026-09-18 |

*Accepted risks do not resurface in future audit runs.*

---

## Coverage Notes (not counted)

1. **T-05-14:** the committed end-to-end symlink tests uninstall Claude before Cursor, so Claude is never the last requester through the link, and removing the Lstat guard turns only the unit test RED. A throwaway scratch test showed the guard does protect that path at HEAD. A committed Claude-last test would close this.
2. **T-05-08:** no committed test pins "a file-level link is replaced, not written through". The auditor's scratch probes confirmed it.
3. **T-05-27:** if D-11 is ever probed and a pickup observed, the shared-AGENTS.md mitigation must be built and this threat re-audited.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-19 | 31 | 31 | 0 | gsd-security-auditor (opus) at `9d5efb67`: agents+cli suites `ok`, 32/32 ownership leaves, drift gate green, the 05-06 evidence gates re-run, and guard mutations (a1, a2, b1, b2, c-i plus six new ones) re-applied in a scratch archive copy, each RED |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-19
