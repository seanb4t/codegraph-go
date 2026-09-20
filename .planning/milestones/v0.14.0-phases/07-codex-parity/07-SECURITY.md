---
phase: "7"
slug: "codex-parity"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-19"
---

# Phase 7 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| a user's hand-edited `~/.codex/config.toml` or repo `.codex/config.toml` → `internal/agents/toml.go` | arbitrary third-party TOML that codegraph edits in place; a line scanner with multi-line-string and bracket-depth state, any-indent header recognition and a leading-BOM pseudo-line (07-01, CR-01, WR-03) | whole config bytes |
| conflicting `codegraph` definitions (inline, dotted, quoted, duplicate, array-of-tables, detached subtable) → `tomlTableConflict` | refused with a named error; `spliceTOMLTable` and `stripTOMLTable` both re-check internally and return content unchanged — never a duplicate key (07-01, 07-05) | a refusal, not a write |
| Codex hook engine → `.codex/hooks/codegraph-pretooluse{,-local,-global}.sh` | untrusted PreToolUse stdin on every Bash call; the `.codegraph` check is the first executable statement, before any spawn or stdin read; no PATH-dependent command (Codex invokes hooks with an empty PATH) (07-07, D-22) | hook event JSON |
| guard → `codegraph hook pretooluse --harness codex` → Codex | `io.LimitReader` cap, tolerant decode, recover-first `RunE`, `additionalContext`-only output, exit 0 on every path; the Go core independently re-checks stdin `cwd` and `.codegraph` (07-07, D-21/D-22) | hook output |
| install-time ExecPath → guard script bytes | `shellSingleQuote` into a token carrying its own quotes; absolute non-empty path required; token count exactly 1; used only as `"$codegraph_bin"` (07-07, D-19) | a filesystem path |
| ExecPath → `hooks.json` | never — the registered command names only the guard (D-19/D-20); local is the docs' quoted `"$(git rev-parse --show-toplevel)/…"` form, global a `shellSingleQuote`'d absolute path | nothing |
| a user's or team's `.codex/hooks.json` → `writeHookEntry` / `removeHookEntry` | exact-command ownership (`242ec0a`); appended last on first install, spliced back at its original index on update (WR-01), so Codex's position-keyed trust never re-flags an untouched foreign hook | hook registrations |
| `os.TempDir()` → the per-(session, agent) sentinel | the Phase 6 Gate reused unchanged (0700, `Lstat`, `O_NOFOLLOW`, SHA-256 filenames); symlinked and unwritable subtests in the Codex forced-error suite | mtimes only |
| codegraph → Codex's trust model | codegraph never writes `projects.*.trust_level`; the D-10 Note tells the user how; the D-19 Note points at `/hooks`; the bypass flag appears nowhere in shipped code, help or docs | advice only |
| one agent's uninstall → repo-root `AGENTS.md` (shared by Codex and opencode) | `instructionsRequestedElsewhere` gates both Uninstalls; `ActionKept` plus a Note while a sibling still declares it | a marker block |
| codegraph → `.agents/skills/codegraph` (shared) | manifest-presence ownership; a sibling's package is never destroyed | SKILL.md + manifest |
| codegraph → a user's `AGENTS.override.md` | a read-only existence check for a Note; never written | nothing |
| scratch HOME/CODEX_HOME → the maintainer's real HOME | four files sha256-identical before and after both live sessions; `auth.json` symlinked, never read or printed | checksums only |
| MCP `instructions` const → every client session | a compile-time ASCII literal, no host path, ≤600 bytes, skill sentence inside 512 bytes, harness-neutral | a string |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-07-01 | Tampering | `findTOMLTableRange` end scan | high | mitigate | `toml.go:233-235` any-indent recognition; `toml.go:122-172` state-aware scan; maintainer-layout tests (`toml_test.go:107,127,137,147`); Family (a1) reproduced RED during the audit | closed |
| T-07-02 | Tampering | splice over inline/dotted/quoted/duplicate forms | medium | mitigate | `toml.go:341-400` conflict scan; `toml.go:29-31` and `:68-70` return unchanged; `codex.go:193`/`:302` surface the error; `TestTOMLTableConflict`, `TestSpliceTOMLTable_ConflictLeavesContentUnchanged`, `TestCodex_Install_RefusesConflictingCodegraphTable` | closed |
| T-07-03 | Tampering | CRLF configs | low | mitigate | `toml.go:319-324`; `TestSpliceTOMLTable_CRLFPreserved`, `TestStripTOMLTable_CRLFRoundTrip` | closed |
| T-07-04 | Repudiation | released binaries carry the splice bug | medium | accept | AR-07-01 (D-08/B2); WINDOWS ledger row 38 open until the v0.14.0 release; the STATE.md warning is present | closed |
| T-07-05 | Repudiation | install/uninstall target switch | medium | mitigate | `install.go:114` and `uninstall.go:51` check `Changed("target")` before `case yes:`; Families (b1)/(b2) | closed |
| T-07-06 | DoS | picker footer overflow | low | mitigate | no trailing newline in `agentpicker.go:67` / `daemonpicker.go:64`; `TestAgentPickerFootprintFitsDefaultPane` asserts height ≤ 30 and `space: toggle` visible at 100×30 with 8 rows; the real-PTY leg runs in CI `tmux-e2e` (AR-07-08) | closed |
| T-07-07 | Tampering | the real `~/.codex` and `~/.agents` | medium | mitigate | 07-LIVE-SESSIONS.md C4 and T6: four sha256 `same` lines; `L7 … PASS` in both sessions | closed |
| T-07-08 | Info Disclosure | real config.toml / auth.json contents | medium | mitigate | `auth.json` symlinked and never read or printed; only checksums recorded; the audit's own secret scan found nothing | closed |
| T-07-09 | Spoofing (of evidence) | CODEX-01 verdict lines | medium | mitigate | the pass bar was locked in 07-CONTEXT D-06 before any session; fixed line forms; positive controls; the untrusted negative control (L2, A3, B8) | closed |
| T-07-10 | Tampering | `auth.json` through the symlink | low | accept | AR-07-02 (D-02); the credential store is Codex's own | closed |
| T-07-11 | EoP | Codex project trust | high | mitigate | no `trust_level` write anywhere in `codex.go`; `codexTrustNote` is advisory (`codex.go:121-133`); `TestCodex_Install_Local_TrustNote` asserts no `[projects` header at either scope; confirmed by the audit's real-binary tracer | closed |
| T-07-12 | Tampering | conflicting config.toml forms | medium | mitigate | `codex.go:193`, `:302`; Family (d3); the file is byte-identical on refusal | closed |
| T-07-13 | Tampering | local writes escaping the repo | medium | mitigate | every local path is cwd-relative (`codex.go:64,79`; `codex_pretooluse.go:27,44`); the audit's tracer wrote zero bytes under HOME | closed |
| T-07-14 | Info Disclosure | ExecPath in a committed `.codex/config.toml` | low | accept | AR-07-03 — the same exposure the local `.mcp.json` stdio entry already has | closed |
| T-07-15 | Tampering | shared repo-root `AGENTS.md` | medium | mitigate | `shared.go:802-830`, called from `codex.go:327` and `opencode.go:337`; `TestSharedAgentsMD_UninstallOrders`, `_KeptWhileOtherConfigured`, `_GlobalPathsNotShared`; the audit traced `kept: AGENTS.md` plus the Note, then byte-clean after the last uninstall | closed |
| T-07-16 | Tampering | `AGENTS.override.md` | low | mitigate | `codex.go:222-228` is a `fileExists` read only; `TestCodex_Install_OverrideNote` asserts the override is byte-identical at both scopes | closed |
| T-07-17 | EoP | a hook blocking or denying a tool call | high | mitigate | the output struct carries only `hookEventName` and `additionalContext` (`hook_pretooluse.go:99-107`, shared with the Codex path at `:265`); both templates have no `set -e`, no `exec`, and a trailing `exit 0`; `assertHookContract` on all 24 forced-error subtests; Family (f1) reproduced RED and a 7-path live guard run gave exit 0 everywhere | closed |
| T-07-18 | DoS | malformed or oversized stdin | medium | mitigate | `maxHookStdinBytes` with `io.LimitReader` (`:229`); `defer recover()` as the first statement of `RunE` (`:138`); `TestHookPreToolUseCodex_ForcedErrorContract` (24 cases, count asserted), `_PanicIsRecovered` | closed |
| T-07-19 | Tampering | ExecPath injection into the guard | high | mitigate | `codex_pretooluse.go:136-158`; empty and relative rejected; token count exactly 1; `TestRenderCodexPreToolGuard/quote_and_space_in_path` executes a binary under `it's a dir/` and asserts it started | closed |
| T-07-20 | Tampering | foreign `hooks.json` groups | high | mitigate | `blockOwnsAnyCommand`/`commandIsOwned` (`shared.go:185-219`) reused unchanged; the WR-01 splice at `shared.go:268-299`; `TestWriteHookEntry_UpdatePreservesForeignBlockPosition`; Family (j2) reproduced RED, and the audit's trace kept a foreign `^Bash$` group and an unrelated `SessionStart` | closed |
| T-07-21 | Info Disclosure | binary path in `hooks.json` | low | mitigate | `codexPreToolHookCommand` never uses ExecPath; assertions at `codex_pretooluse_test.go:137,169`; the audit's installed `hooks.json` carries no binary path | closed |
| T-07-22 | Spoofing | sentinel directory attacks | medium | mitigate | the Phase 6 `nudge.Gate` reused (`hook_pretooluse.go:261`); `symlinked_sentinel_dir` and `unwritable_sentinel_base` subtests in the Codex suite | closed |
| T-07-23 | EoP | a hook installed without consent, or against `hooks = false` | medium | mitigate | `codex.go:240-278` — Keep never adds; the 3-form `codexHooksExplicitlyDisabled` reader; `TestCodex_Install_DefaultWritesNoHooks`, `TestCodexPreToolNudge_SkippedWhenHooksDisabled`; both re-verified with the real binary | closed |
| T-07-24 | EoP | users told to bypass hook trust | medium | mitigate | a repo-wide `rg` finds the flag string only in a prohibition comment; `TestCodexNotesNeverAdviseTrustBypass` (7 branches), `TestInstallHelpNeverAdvisesTrustBypass` (install/uninstall help plus the CLI reference) | closed |
| T-07-25 | Tampering | a malformed `hooks.json` on Keep | low | mitigate | `codex.go:259` falls through silently; `TestCodexPreToolNudge_KeepWithMalformedHooksJSONTouchesNothing` asserts unchanged bytes, no error, no guard | closed |
| T-07-26 | Tampering | foreign `^Bash$` groups | high | mitigate | exact-command identity only, never the matcher; `ownershipCodexForeignBashGroup` planted in every codex ownership leaf (`ownership_test.go:193,207,559,653`); Family (g4) | closed |
| T-07-27 | Tampering | the real HOME during CODEX-05/06 | medium | mitigate | the T6 block: the same four sha256 values as the CODEX-01 pre-flight; `L7 … PASS` | closed |
| T-07-28 | Spoofing (of evidence) | L5/L6 verdicts | medium | mitigate | JSONL excerpts; positive controls; the probe hook proving hooks ran in the un-indexed control; the pass bar pre-locked | closed |
| T-07-29 | Info Disclosure | transcripts in the repo | low | accept | AR-07-04 — gofixture scratch repos only; the audit's secret scan was clean | closed |
| T-07-30 | EoP | hook-trust bypass in scripted runs | low | mitigate | the flag is used nowhere in 07-LIVE-SESSIONS.md; the real `/hooks` TUI trust path was exercised first (B2 untrusted → 0 fires; B5 trusted → 2 deliveries) | closed |
| T-07-31 | Repudiation | the capability table overclaiming | medium | mitigate | `TestCapabilityDoc_VerificationColumn` — 16 rows, two regex forms, `n/a` for unsupported, cursor/gemini/kiro forced `[ASSUMED]`, codex forced `verified … 07-LIVE-SESSIONS.md`; Family (h2) | closed |
| T-07-32 | Tampering | the doc drifting from the shipped table | medium | mitigate | `TestCapabilityDoc_MirrorsCapabilities` compares 8 columns × 16 rows against `Capabilities()`; Family (h1) | closed |
| T-07-33 | EoP | a doc advising a trust bypass | low | mitigate | `capability_doc_test.go:313-318` builds the flag name by concatenation, so the test file never carries the literal | closed |
| T-07-34 | Info Disclosure | the `instructions` const | medium | mitigate | `server.go:57` is a compile-time literal; `TestInstructionsCarriesNoWireContractViolation` (pure ASCII, no `/Users/`, `/home/`, `/private/`, `C:\`); `TestInstructionsReachesTheWireVerbatim` | closed |
| T-07-35 | Repudiation | a false skill claim for some clients | low | mitigate | `TestInstructionsSkillSentenceWithinFirst512Bytes` (ends ≤512 bytes, rejects "Claude Code"); `TestInstructionsSkillClaimIsResolvable` against the embedded `SkillMarkdown`; `TestInstructionsClaimGuardsAreNotVacuous` | closed |
| T-07-36 | Tampering | a transcript re-freeze masking a wire change | medium | mitigate | `e556e5e5`: 38 files, 38 insertions, 38 deletions, one line per file, every added and removed line carrying the instructions string (counted both directions) | closed |
| T-07-SC | Tampering | package installs | low | accept | AR-07-05 — no package was installed (the tmux step was skipped); `go.mod`/`go.sum` byte-unchanged across all 88 phase commits | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*
*Totals: 6 high, 19 medium, 12 low. 32 mitigate, 5 accept, 0 transfer. 0 open.*

---

## Carried Phase 6 contract — what still holds on the Codex path

Reused unchanged and re-verified against the Codex entry points: `shellSingleQuote` (T-06-01), the never-block guard shape (T-06-02), the decision-key-free output struct (T-06-03/16 — the same `claudeHookOutput` type serves both adapters), cheap-check-first ordering (T-06-04), exact-command ownership (T-06-05/19), the stdin cap (T-06-06/17), `nudge.Qualifies` and its single package-level regexp (T-06-09/11), the pinned `nudge.Text` (T-06-10, confirmed byte-for-byte in the audit's live fire), the sentinel gate and SHA-256 naming (T-06-12/13/14), the whole-body `recover()` (T-06-15 — one `RunE` covers both harnesses), the Keep/On/Off tri-state (T-06-20/23/24), and the accepted double fire (T-06-18).

Two Phase 6 items do **not** carry, both in the safer direction:

- **AR-06-08 is not inherited.** D-20 quotes the Codex command from day one: local is the docs' `"$(git rev-parse --show-toplevel)/…"` form, global a `shellSingleQuote`'d absolute path. AR-06-08 stays open only against Claude's pre-existing entries, which are out of Phase 7's scope.
- **T-06-08's PATH fallback has no Codex analogue.** Codex invokes hooks with an empty PATH, so both templates use only POSIX parameter expansion and a rendered absolute binary path — no `dirname`, `basename` or `command -v`.

Two mechanisms are strengthened here: the Go core re-checks stdin `cwd` and `.codegraph` independently of the guard (D-22 defence in depth, absent on the Claude path), and `writeHookEntry` preserves an owned block's index on update (WR-01) instead of always appending.

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-07-01 | T-07-04 | Released binaries carry the Codex TOML data-loss bug and no patch release was cut (maintainer decision B2, D-08). WINDOWS ledger row 38 is open until the v0.14.0 release, and STATE.md carries the do-not-run warning | CONTEXT D-08 (maintainer-approved) | 2026-09-19 |
| AR-07-02 | T-07-10 | `auth.json` is symlinked rather than copied so Codex token refresh keeps working; the credential store is Codex's own and codegraph never reads it (D-02) | plan-time register | 2026-09-19 |
| AR-07-03 | T-07-14 | A committed `.codex/config.toml` exposes the installing machine's binary path exactly as the local `.mcp.json` stdio entry already does; a teammate's Codex fails to start the server rather than running another binary | plan-time register | 2026-09-19 |
| AR-07-04 | T-07-29 | Pasted transcripts come only from gofixture scratch repos, and the audit's secret scan over every Phase 7 artifact found nothing. The four real-HOME sha256 lines and the `auth.json` symlink target disclose the maintainer's home path, already public in this repo's git authorship | plan-time register + audit scan | 2026-09-19 |
| AR-07-05 | T-07-SC | No package was installed (the tmux step was skipped by maintainer decision) and `go.mod`/`go.sum` are byte-unchanged across all 88 Phase 7 commits | plan-time register | 2026-09-19 |
| AR-07-06 | T-07-13 (adjacent) | `codex.go` and `codex_pretooluse.go` derive every global path from `os.UserHomeDir()` and ignore `CODEX_HOME` (the variable appears only in comments). A user whose `CODEX_HOME` points elsewhere gets a global install or uninstall against `~/.codex` instead — ineffective, and the edit is conflict-guarded and marker-scoped, so it cannot destroy content | security audit observation | 2026-09-19 |
| AR-07-07 | T-07-20 (adjacent) | `ActionRemoved` labels a `hooks.json` line `removed:` even when the file was rewritten keeping a foreign group (reproduced in the audit's live trace). A cosmetic reporting inaccuracy; the foreign group is byte-intact | security audit observation | 2026-09-19 |
| AR-07-08 | T-07-06 | FIX-03's real-PTY tmux assertion was not run locally (maintainer decision 2026-09-19, issue #75). The security-relevant property — the cancel-key footer is visible and the view fits — is asserted directly by the model-level guard, and the re-anchored TTY-05 assertion compiles clean and runs in CI's `tmux-e2e` job on every push. A local-evidence gap, not a security gap | maintainer decision, recorded in 07-VERIFICATION.md | 2026-09-19 |
| AR-07-09 | T-07-18 (adjacent) | IN-01: the Codex argv-shape decoder takes the last array element and has never been exercised live (Codex sends a string). A wrong pick yields a missed or spurious `additionalContext` line, never a block and never a decision key | 07-REVIEW.md, deferred by decision | 2026-09-19 |
| AR-07-10 | T-07-15 (adjacent) | Uninstall leaves emptied `.agents/`, `.agents/skills/` and `.codex/hooks/` directories behind (reproduced in the audit's live trace). Deliberate: these directories are codegraph-named but not codegraph-exclusive, and a recursive delete would risk a user's file. Carried todo | carried todo, audit-confirmed | 2026-09-19 |
| AR-07-11 | T-07-20 (adjacent) | Removing codegraph's hook group shifts later foreign groups' indices, and Codex's position-keyed trust re-flags them as "modified — review required" (live-verified; the D-23 line reads `yes`). Inherent to array removal; WR-01 fixed only the update path. The direction is fail-safe — Codex asks the user to re-review, never auto-trusts — and it is normally unreachable because our group is appended last on first install | security audit observation | 2026-09-19 |
| AR-07-12 | T-07-19 (adjacent) | Codex's hook-trust hash covers the registered command, not the guard script's bytes, so a re-render (for example after the binary moves) changes what the trusted hook executes without re-triggering `/hooks` review. This is the deliberate intent of D-19 (byte-stable registration across upgrades) and matches Claude's model; the guard is only ever written by codegraph, with identical content modulo the binary path | security audit observation | 2026-09-19 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-19 | 37 | 37 | 0 | gsd-security-auditor (opus) at `13c2dd3d`. Traced each mitigation to its boundary deeper than L1. Re-ran the enforcing suites (agents, cli/…, nudge, mcp/… — 9 packages green) and `go vet -tags tmux`. Independently reproduced four mutation-family positive controls (a1, f1, j1+j3, j2) in a throwaway worktree. Ran a real-binary install/uninstall tracer against a scratch HOME (zero bytes under HOME; a foreign `^Bash$` group, an unrelated `SessionStart` group, the shared `AGENTS.md` and the shared `SKILL.md` all survived), a 7-path live guard exercise (exit 0 everywhere, no decision keys) and consent checks for opt-in-only and `[features] hooks = false`. Confirmed the trust-bypass flag appears nowhere in shipped code, help or docs; `go.mod`/`go.sum` unchanged across all 88 phase commits; the 38-file transcript re-freeze changes one instructions-bearing line per file; and the Phase 6 real-HOME checksums are unchanged in both live sessions. |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log (AR-07-01..12)
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-19
