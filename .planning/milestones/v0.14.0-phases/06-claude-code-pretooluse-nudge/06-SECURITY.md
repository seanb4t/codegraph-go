---
phase: "6"
slug: "claude-code-pretooluse-nudge"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-19"
---

# Phase 6 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Claude Code → guard / `codegraph hook pretooluse` stdin | untrusted hook-event JSON (session_id, agent_id, tool_name, tool_input), capped by `io.LimitReader` and parsed silently on doubt (06-01, 06-03) | tool commands, file paths |
| install-time ExecPath → guard script bytes | the binary path is rendered into the guard as a single POSIX single-quoted token; only absolute, non-empty paths are accepted (06-01) | a filesystem path |
| guard → Claude Code | the exit code and stdout decide whether the tool call is blocked, denied or prompted; the guard always exits 0 and emits only `additionalContext` (06-01, 06-03) | hook output |
| `os.TempDir()` (often a shared `/tmp`) → sentinel directory | the per-(session, agent) cooldown sentinel lives in a directory other local users can reach (06-03) | mtimes only; filenames are SHA-256 hex |
| user's `settings.json` → writeHookEntry / removeHookEntry | exact-command ownership (`242ec0a`); a foreign or hand-edited entry is never overwritten or removed (06-04, 06-05) | hook registrations |
| manifest (any local process can rewrite it) → the Keep decision | the sticky opt-in record; a drift signal, not an authenticity control (06-04) | manifest Files keys |
| user flags / unattended `upgrade` → agent config | `Changed`-gated tri-state; upgrade carries Keep and never adds (06-05) | the opt-in |
| this repo's `.claude/settings.json` → every Claude Code session opened here | dogfooded registration; the unrendered guard falls back to PATH and still exits 0 (06-05) | hook command |
| live session → the maintainer's real `~/.claude`; transcript → verdict | pre-flight and post checksums, locked pass bar, positive-controlled counting (06-06) | config and evidence |
| generated reference ↔ live Cobra tree | `task docs:cli:drift` plus flag accounting; hidden commands allowlisted (06-07) | `docs/CLI-REFERENCE.md` |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-06-01 | Tampering (shell injection) | renderPreToolGuard / guard template | high | mitigate | `shellSingleQuote` (`'` → `'\''`), absolute and non-empty path, token count exactly 1; used only as quoted `"$codegraph_bin"`; `TestRenderPreToolGuard/quote_and_space_in_path` and friends; Family (a2) | closed |
| T-06-02 | DoS (blocking the tool call) | guard exit path | high | mitigate | no `set -e`, no `exec`, trailing `exit 0`; `TestPreToolUseGuard` (missing, non-executable, exit 3 and SIGSEGV binary; unset dir); Family (a1) | closed |
| T-06-03 | EoP/DoS (decision keys) | runHookPreToolUse output | high | mitigate | the output struct has only `hookEventName` and `additionalContext`; `assertHookContract` on every run; Families (c4) and (c5) | closed |
| T-06-04 | DoS (un-indexed overhead) | guard D-04 check | medium | mitigate | directory check is the first statement, before stdin is read or any process starts; `TestPreToolUseGuard/not_indexed`, `/codegraph_is_file`; live 2.8–9.0 ms | closed |
| T-06-05 | Tampering (ownership) | writeHookEntry("PreToolUse") | high | mitigate | exact command only via `blockOwnsAnyCommand`/`commandIsOwned` (WR-02); planted same-matcher block survives; `TestOwnershipExactIdentity` 32/32 | closed |
| T-06-06 | DoS (stdin) | runHookPreToolUse | low | mitigate | `io.LimitReader`; oversized, empty or undecodable input stays silent; `ForcedErrorContract/{oversized_input,malformed_json,no_stdin}` | closed |
| T-06-07 | Info Disclosure | local-scope rendered guard | low | accept | same exposure as the `.mcp.json` stdio entry (06-01 register) | closed |
| T-06-08 | EoP | unrendered guard PATH fallback | low | accept | rendered guards are always absolute; the fallback belongs only to the dogfooded copy; `TestPreToolUseGuardSourceFallsBackToPATH` | closed |
| T-06-SC | Tampering | package installs | low | accept | `go.mod`/`go.sum` unchanged across the phase; no new dependency | closed |
| T-06-09 | Tampering (misclassification) | Qualifies shell branch | low | mitigate | exact first-word lookup; quoted/expanded assignments rejected; corpora tests; Families (b1) and (b2) | closed |
| T-06-10 | Spoofing (instruction-like context) | nudge.Text | medium | mitigate | pinned factual constant plus hand-typed oracle; three `TestPreToolUseNudgeText*` drift guards; Family (b3) | closed |
| T-06-11 | DoS (regex cost) | Qualifies | low | mitigate | the package's only regexp is compiled once at package level | closed |
| T-06-12 | Tampering/EoP (symlink race in shared temp) | Gate.Due / recordFire | high | mitigate | Mkdir 0700; Lstat refuses a symlinked, non-directory, non-0700 (WR-01) or foreign-owned dir; `O_NOFOLLOW` open, no truncate or write, fstat and `Futimes` on the fd; sentinel and gate tests; Families (c2) and (c3) | closed |
| T-06-13 | Tampering (path traversal) | sentinel naming | medium | mitigate | the file name is the SHA-256 hex of the key; ids never appear in a path | closed |
| T-06-14 | Info Disclosure | sentinel directory | low | mitigate | content-free sentinel (size 0) | closed |
| T-06-15 | DoS (erroring hook) | RunE | high | mitigate | deferred `recover()` over the whole body; RunE only returns nil; no ancestor PreRun; Families (c4) and (c6) | closed |
| T-06-16 | EoP (decision keys) | output struct | high | mitigate | as T-06-03; Family (c5) | closed |
| T-06-17 | DoS (stdin) | stdin read | low | mitigate | as T-06-06 | closed |
| T-06-18 | Repudiation (double fire) | Gate.Due | low | accept | D-08: a rare double fire under a parallel race is harmless; parallel tests limit output to pinned-or-empty | closed |
| T-06-19 | Tampering (ownership) | PreToolUse install/uninstall | high | mitigate | exact `ownCommands`; hand-edit duplicates (one handler per block, 06-05); `TestBlockOwnsAnyCommand`; Family (d1) | closed |
| T-06-20 | Tampering/consent (re-adding) | Install Keep/Off | medium | mitigate | Keep enables only when recorded; Off removes and forgets; the CR-01 widening counts only own exact-command entries and only when the manifest is absent; Families (d2) and (d4) | closed |
| T-06-21 | Tampering (manifest-forced refresh) | preToolNudgeRecorded | low | accept | the manifest is a drift signal, not an authenticity control (`manifest.go:9-15`) | closed |
| T-06-22 | DoS (corrupt manifest) | Keep path | low | mitigate | unreadable manifest → touch nothing; `TestPreToolNudge_KeepWithUnreadableManifestTouchesNothing` | closed |
| T-06-23 | Tampering/consent | install flag mapping | medium | mitigate | the default is Keep; On or Off requires `Changed`; Family (e2) | closed |
| T-06-24 | Tampering/consent | upgrade refresh | medium | mitigate | an explicit `PreToolNudgeKeep`; never adds; Family (e1) | closed |
| T-06-25 | DoS (dogfood hook) | `.claude/settings.json` PreToolUse | low | accept | the guard exits 0 on every binary failure; mode 100755 | closed |
| T-06-26 | Tampering (registration drift) | settings.json vs fragment | low | mitigate | registration-equality and shape tests; Family (e3) | closed |
| T-06-27 | Tampering | maintainer's global config | medium | mitigate | pre-flight and post sha256 identical (be3ad316…), no global guard; installs only under `/tmp/06-live` | closed |
| T-06-28 | Spoofing (of evidence) | verdict lines | medium | mitigate | pass bar locked in D-18 before sessions; positive control before zero claims; negative control C6 | closed |
| T-06-29 | Repudiation (misattributed denial) | C7 | low | mitigate | attribution by hook command/output (the Boost entries belong to the maintainer's own hook) | closed |
| T-06-30 | Info Disclosure | pasted transcripts | low | accept | gofixture scratch repos only; the secret-pattern scan found nothing | closed |
| T-06-31 | Tampering | `docs/CLI-REFERENCE.md` drift | low | mitigate | `task docs:cli:drift` byte-identical; flag accounting; allowlisted hidden commands | closed |
| T-06-32 | Repudiation (partial record) | phase gate | low | mitigate | 19 mutation families present; `D-18 verdict: PASS`; untouched-surface diffs vs `603efc95` | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-06-01 | T-06-07 | The rendered local guard exposes the binary path, as `.mcp.json` already does | plan-time register | 2026-09-19 |
| AR-06-02 | T-06-08 | The PATH fallback exists only in the unrendered dogfooded copy; installed guards are always absolute | plan-time register | 2026-09-19 |
| AR-06-03 | T-06-SC | No dependency was added | plan-time register | 2026-09-19 |
| AR-06-04 | T-06-18 | A rare double fire under a parallel race is harmless (D-08) | CONTEXT D-08 (maintainer-approved) | 2026-09-19 |
| AR-06-05 | T-06-21 | The manifest is a drift signal, not an authenticity control | plan-time register | 2026-09-19 |
| AR-06-06 | T-06-25 | Dogfooding runs the guard in this repo's sessions; it exits 0 on every failure | plan-time register | 2026-09-19 |
| AR-06-07 | T-06-30 | Pasted evidence comes only from scratch fixture repos | plan-time register | 2026-09-19 |
| AR-06-08 | T-06-02 (pre-guard) | The registered command is unquoted, so a project dir or `$HOME` containing whitespace splits before the guard runs. The result is a non-blocking "hook error" notice, never a block and never injection (no command substitution). Mirrors the existing SessionStart entry; the exec-form/quoting change is deferred (06-CONTEXT Deferred Ideas) because it changes the owned identity | security audit observation, recorded by the orchestrator | 2026-09-19 |
| AR-06-09 | T-06-19 | Ownership is block-granular on write: a handler a user adds INSIDE a codegraph-owned PreToolUse block is dropped on the next refresh (removal is handler-granular). Pre-dates this phase; one handler per block (06-05) confines it to users editing our own blocks | security audit observation, recorded by the orchestrator | 2026-09-19 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-19 | 33 | 33 | 0 | gsd-security-auditor (opus) at `b98bf7b9`. It traced each mitigation to its boundary, deeper than L1. It re-ran the enforcing tests (`internal/nudge` 18/18, the cli/agents/mcp subsets and `task docs:cli:drift`), confirmed all 19 mutation families, and re-checked the global config checksum. |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-19
