---
phase: "6"
slug: "claude-code-pretooluse-nudge"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-19"
---

# Phase 6 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (stdlib) — `internal/nudge` pure-core tests (corpora, injected-clock cooldown, planted mtimes, no sleeps — D-17); `internal/cli` in-process subcommand tests via `execCmdWithInput` (Execute() must return nil — D-16); `internal/agents` exec tests of the REAL embedded guard rendered with stub binaries (D-16 guard level), lifecycle/ownership/capability tables with `fakeHome(t)`; `internal/mcp` drift guards over the Go constant; RED demonstrations in `06-MUTATION-LOG.md`; live Claude Code evidence in `06-LIVE-SESSIONS.md` (Herdr panes — never a `go test` assertion, D-00) |
| **Config file** | `go.mod` pins go 1.26.6 — every local Go gate runs under `GOTOOLCHAIN=go1.26.6`; `Taskfile.yml` (`docs:cli`, `docs:cli:drift`); `internal/cli/testdata/cli-reference-allowlist.txt`; `internal/nudge/testdata/{true,false}-positives.json` |
| **Quick run command** | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/nudge/ ./internal/agents/ ./internal/cli/ ./internal/mcp/` |
| **Full suite command** | `GOTOOLCHAIN=go1.26.6 go build ./... && GOTOOLCHAIN=go1.26.6 go test -count=1 $(GOTOOLCHAIN=go1.26.6 go list ./... \| rg -v 'internal/daemon$') && GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/daemon/ && GOTOOLCHAIN=go1.26.6 task docs:cli:drift` (daemon runs alone — WINDOWS #37) |
| **Estimated runtime** | ~50 s quick; ~5 min full |

---

## Sampling Rate

- **After every task commit:** Run the quick command (plus `-race` on `./internal/nudge/ ./internal/cli/` for 06-03)
- **After every plan wave:** Run the full suite command
- **Before `/gsd-verify-work`:** Full suite green; `06-MUTATION-LOG.md` carries 18 RED families (a1-a2, b1-b3, c1-c6, d1-d4, e1-e3); `06-LIVE-SESSIONS.md` reads `D-18 verdict: PASS` with C1-C7 excerpts and the un-indexed negative control
- **Max feedback latency:** 60 seconds (quick command)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 06-01-T1 | 06-01 | 1 | NUDGE-03, NUDGE-04, NUDGE-06 | T-06-01, T-06-03, T-06-04, T-06-05 | opt-in install writes a quoted-ExecPath guard + 4 owned PreToolUse blocks; fire = pinned additionalContext only, exit 0; un-indexed silent | tracer (RED first) + real binary | `go test ./internal/cli/ ./internal/agents/ -run 'TestHookPreToolUse_GrepFiresPinnedContext$\|TestHookPreToolUse_NoSessionIsSilent$\|TestHookCmd_HiddenTwoLevel$\|TestClaude_Install_PreToolNudgeOn_WritesGuardAndBlocks$\|TestClaude_Install_DefaultWritesNoPreToolUse$'` + real-binary install/fire/silent run | ❌ W0 | ⬜ pending |
| 06-01-T2 | 06-01 | 1 | NUDGE-03, NUDGE-04 | T-06-01, T-06-02, T-06-04, T-06-08 | guard exits 0 on every binary failure; no process started un-indexed; ExecPath quoting; Family (a) RED | unit (exec of rendered template) + mutation | `go test ./internal/agents/ -run 'TestPreToolUseGuard$\|TestRenderPreToolGuard$\|TestPreToolUseGuardSourceFallsBackToPATH$'` | ❌ W0 | ⬜ pending |
| 06-02-T1 | 06-02 | 2 | NUDGE-05 | T-06-09, T-06-11 | first-word shell rule, parse doubt silent, Read exclusions | unit (RED first) over D-15 corpora | `go test ./internal/nudge/ -run 'TestQualifiesCorpora$\|TestCorporaShape$\|TestQualifiesShellFirstWord$\|TestQualifiesReadNonCodeExtensions$'` | ❌ W0 | ⬜ pending |
| 06-02-T2 | 06-02 | 2 | NUDGE-03, NUDGE-05 | T-06-10 | nudge text names only real tools, no unpinned facts, factual one-liner; Family (b) RED | drift guard + mutation | `go test ./internal/mcp/ -run 'TestPreToolUseNudgeText'` | ❌ W0 | ⬜ pending |
| 06-03-T1 | 06-03 | 3 | NUDGE-04, NUDGE-05 | T-06-12, T-06-13, T-06-14 | per-(session, agent) 60 s gate; symlink/foreign refusal at read and record layers; no session content on disk | unit (RED first, injected clock, planted mtimes, -race) | `go test -race ./internal/nudge/ -run 'TestSessionKey$\|TestGate_\|TestSentinel_\|TestDefaultDirHonoursTMPDIR$'` | ❌ W0 | ⬜ pending |
| 06-03-T2 | 06-03 | 3 | NUDGE-03, NUDGE-04 | T-06-15, T-06-16, T-06-17 | every forced error path → Execute() nil, stdout empty or pinned, no decision key | unit (RED first, -race) | `go test -race ./internal/cli/ -run 'TestHookPreToolUse_'` | ❌ W0 | ⬜ pending |
| 06-03-T3 | 06-03 | 3 | NUDGE-03, NUDGE-04 | T-06-12, T-06-15, T-06-16 | D-16's three named mutations + cooldown/sentinel guards go RED | mutation | Task 3 verify in 06-03-PLAN.md (Family (c)) | ❌ W0 | ⬜ pending |
| 06-04-T1 | 06-04 | 4 | NUDGE-06 | T-06-20, T-06-22 | Keep refreshes only when recorded; Off removes + forgets; uninstall always attempts; idempotent | unit (RED first) + plain golden | `go test ./internal/agents/ -run 'TestPreToolNudge_' && go test ./internal/cli/ -run 'TestPlainGolden$'` | ❌ W0 | ⬜ pending |
| 06-04-T2 | 06-04 | 4 | NUDGE-06 | T-06-19 | capability table names the guard; 32-leaf ownership table with a planted same-matcher PreToolUse block | unit (RED first) | `go test ./internal/agents/ -run 'TestCapabilitiesTableDrivesDerivations$\|TestCapabilitiesMatchInstallWrites$\|TestOwnershipExactIdentity$\|TestClaude_DescribePaths_IncludesManifest$'` | ✅ (extend) | ⬜ pending |
| 06-04-T3 | 06-04 | 4 | NUDGE-06 | T-06-19, T-06-20 | 242ec0a-class ownership, stickiness and table guards go RED | mutation | Task 3 verify in 06-04-PLAN.md (Family (d)) | ❌ W0 | ⬜ pending |
| 06-05-T1 | 06-05 | 5 | NUDGE-06, NUDGE-03 | T-06-25, T-06-26 | dogfood registration == fragment; shape pinned (matchers, if rules, timeout 5, no statusMessage) | unit (RED first) | `go test ./internal/agents/ -run 'TestHookRegistrationMatchesFragmentAndScript$\|TestPreToolUseRegistrationShape$'` | ✅ (extend) | ⬜ pending |
| 06-05-T2 | 06-05 | 5 | NUDGE-06 | T-06-23, T-06-24 | Changed tri-state; stderr note; upgrade carries Keep and never adds | unit (RED first) + drift gate | `go test ./internal/cli/ -run 'TestInstall_PreToolNudge_\|TestRefreshInstalledSkills_' && task docs:cli:drift` | ❌ W0 | ⬜ pending |
| 06-05-T3 | 06-05 | 5 | NUDGE-06 | T-06-23, T-06-24, T-06-26 | upgrade/stickiness/dogfood guards go RED | mutation | Task 3 verify in 06-05-PLAN.md (Family (e)) | ❌ W0 | ⬜ pending |
| 06-06-T1 | 06-06 | 6 | NUDGE-03, NUDGE-05 | T-06-27, T-06-28 | scratch repos + locked pass bar + current hooks-reference quotes; $HOME untouched | file checks | Task 1 verify in 06-06-PLAN.md | ❌ W0 | ⬜ pending |
| 06-06-T2 | 06-06 | 6 | NUDGE-03, NUDGE-04, NUDGE-05 | T-06-28, T-06-29 | C1-C5 live with excerpts; fire rate; five open points | manual (orchestrator) + verdict-line gate | Task 2 verify in 06-06-PLAN.md | ❌ W0 | ⬜ pending |
| 06-06-T3 | 06-06 | 6 | NUDGE-04, NUDGE-03 | T-06-27, T-06-29 | C6 un-indexed control 0 fires; C7 no hook error/prompt/deny; D-18 verdict | manual (orchestrator) + completeness gate | Task 3 verify in 06-06-PLAN.md | ❌ W0 | ⬜ pending |
| 06-07-T1 | 06-07 | 7 | NUDGE-06 | T-06-31 | help + reference describe the shipped opt-in | drift gate | `task docs:cli:drift && go test ./internal/cli/ -run 'TestEveryRegisteredFlagIsAccountedFor$\|TestPlainGolden$'` | ✅ | ⬜ pending |
| 06-07-T2 | 06-07 | 7 | NUDGE-03, NUDGE-04, NUDGE-05, NUDGE-06 | T-06-32 | full suite, 18 families, D-18 PASS, untouched surfaces | full suite + drift | full suite command above + Task 2 verify in 06-07-PLAN.md | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

Requirement → test map (from `06-RESEARCH.md` Validation Architecture, refined by the plans; tests assert only what the repo owns — D-00):

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NUDGE-03 | Emits only `hookSpecificOutput.additionalContext`, exit 0 on every forced error path, never a decision key; guard exits 0 when the binary is missing, non-executable, failing or crashing | unit (subcommand + guard level, D-16) + mutation | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run TestHookPreToolUse_ && GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/agents/ -run 'TestPreToolUseGuard$\|TestRenderPreToolGuard$'` | ❌ W0 |
| NUDGE-03 | Registration bytes (matchers, `if` rules, timeout, no statusMessage) pinned in the fragment and the dogfood settings | unit | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/agents/ -run 'TestPreToolUseRegistrationShape$\|TestHookRegistrationMatchesFragmentAndScript$'` | ✅ (extend) |
| NUDGE-04 | First call fires, then ≤ once per 60 s per (session, agent); subagents keyed separately; D-08 symlink/foreign refusals | unit (injected clock, planted mtimes, D-17) | `GOTOOLCHAIN=go1.26.6 go test -count=1 -race ./internal/nudge/ -run 'TestGate_\|TestSentinel_\|TestSessionKey$'` | ❌ W0 |
| NUDGE-04 | Silent with zero overhead when un-indexed — no binary started | unit (guard level, D-04) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/agents/ -run 'TestPreToolUseGuard$'` (subtests not_indexed, codegraph_is_file) | ❌ W0 |
| NUDGE-05 | True/false-positive corpus classification, rates logged | unit (D-15 table test) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/nudge/ -run 'TestQualifiesCorpora$\|TestCorporaShape$'` | ❌ W0 |
| NUDGE-05 | Fire rate against matched calls in a genuinely fresh live session | manual-only (D-18) | 06-06 Tasks 2-3 verdict-line gates | N/A — manual by design |
| NUDGE-06 | Opt-in registration/removal via exact-identity writeHookEntry/removeHookEntry; hand-edit duplicates; unrelated same-matcher entry untouched | unit (lifecycle + extended D-13 ownership table) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/agents/ -run 'TestPreToolNudge_\|TestOwnershipExactIdentity$'` | ✅ (extend) + ❌ W0 |
| NUDGE-06 | Sticky across plain install and upgrade; explicit `--pretool-nudge=false` removes; misdirected flag notes | unit (CLI) + drift gate | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run 'TestInstall_PreToolNudge_\|TestRefreshInstalledSkills_CarriesPreToolNudge$' && GOTOOLCHAIN=go1.26.6 task docs:cli:drift` | ❌ W0 |

---

## Wave 0 Requirements

- [ ] `internal/nudge/` package — `classify.go`, `text.go` (06-01), `cooldown.go` (06-03) with `classify_test.go`, `cooldown_test.go`, `testdata/true-positives.json`, `testdata/false-positives.json` (06-02) — D-01a/D-15/D-17
- [ ] `internal/cli/hook_pretooluse.go` + `hook_pretooluse_test.go` — hidden `hook pretooluse`, the pinned-JSON oracle, the D-16 subcommand suite (06-01, 06-03)
- [ ] `.claude/hooks/pretooluse-nudge.sh` (tracked 100755) + `claudeassets.go` embed + `hooks.PreToolUse` in `.claude/hooks/hooks.json` (06-01)
- [ ] `internal/agents/claude_pretooluse.go` + `claude_pretooluse_test.go` (guard exec suite with stub binaries) and `claude_pretooluse_lifecycle_test.go` (06-01, 06-04)
- [ ] `internal/cli/testdata/cli-reference-allowlist.txt` — `codegraph hook` and `codegraph hook pretooluse` (06-01)
- [ ] `internal/mcp/skill_claims_drift_test.go` — three `TestPreToolUseNudgeText*` guards over the Go constant (06-02)
- [ ] `06-MUTATION-LOG.md` — Phase 2-5 shape, Families (a)-(e) (06-01…06-05)
- [ ] `06-LIVE-SESSIONS.md` — scaffolded by 06-06 Task 1

*Framework install: none — `go test` is already configured; `jq` and `shellcheck` are present on the planning machine and used only in verify commands.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Claude Code matches the registered handlers and delivers the added context; first call fires once (C1) | NUDGE-03, NUDGE-05 | D-00: Claude Code's matcher and delivery are never unit-tested | 06-06 Task 2 prompt 1 in a fresh `claude --debug-file` session in the indexed scratch repo; count the pinned substring in the session JSONL |
| No same-key fire within 60 s; re-fire after ≥ 60 s (C2, C3) | NUDGE-04 | real session keys and real wall-clock spacing | 06-06 Task 2 prompts 2-3 with a `sleep 65` gap; Fire log table |
| Subagent fires once for itself regardless of main's cooldown (C4); /clear fires (C5) | NUDGE-04 | agent_id/session_id values come from Claude Code | 06-06 Task 2 prompts 4 and 8 |
| Un-indexed control has 0 fires (C6); no hook error, prompt, deny or block attributable to the hook (C7) | NUDGE-04, NUDGE-03 | live negative control; attribution by hook command in the debug log | 06-06 Task 3, positive-controlled searches |
| Recorded, not gated: fire rate, true-positive share, uptake, hook wall time; the five open points (decision-less additionalContext, subagent session_id, CLAUDE_CODE_SESSION_ID export, same-command `if` dedup, stdin key order) | NUDGE-05 | D-18 recorded-not-gated list | 06-06 Task 2 recorded lines |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
