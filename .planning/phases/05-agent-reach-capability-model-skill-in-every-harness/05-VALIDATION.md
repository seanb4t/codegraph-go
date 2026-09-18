---
phase: "5"
slug: "agent-reach-capability-model-skill-in-every-harness"
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: "2026-09-18"
---

# Phase 5 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | `go test` (stdlib) — `internal/agents` table tests over all eight targets using `fakeHome(t)` (isolates `$HOME`/`$XDG_CONFIG_HOME`/`$HERMES_HOME`); `internal/cli` plain-golden harness for `--print-config-style`; `tools/clidoc` + `task docs:cli:drift`; RED demonstrations recorded in `05-MUTATION-LOG.md`; live harness evidence in `05-LIVE-SESSIONS.md` (Herdr agent panes — never a `go test` assertion, D-00) |
| **Config file** | `Taskfile.yml` (`docs:cli`, `docs:cli:drift`); `internal/cli/testdata/cli-reference-allowlist.txt`; `go.mod` pins go 1.26.6 — every local Go gate runs under `GOTOOLCHAIN=go1.26.6` |
| **Quick run command** | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/agents/... ./internal/cli/...` |
| **Full suite command** | `GOTOOLCHAIN=go1.26.6 go build ./... && GOTOOLCHAIN=go1.26.6 go test -count=1 $(go list ./... \| rg -v 'internal/daemon$') && GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/daemon/ && GOTOOLCHAIN=go1.26.6 task docs:cli:drift` (daemon runs alone — WINDOWS #37) |
| **Estimated runtime** | ~45 s quick; ~5 min full |

---

## Sampling Rate

- **After every task commit:** Run the quick command
- **After every plan wave:** Run the full suite command
- **Before `/gsd-verify-work`:** Full suite green; `05-MUTATION-LOG.md` carries a RED entry per new guard; `05-LIVE-SESSIONS.md` has a verdict (with negative control) for Cursor, opencode and Antigravity and an `[ASSUMED]` row with URL + date for Gemini CLI and Kiro
- **Max feedback latency:** 60 seconds (quick command)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 05-01-T1 | 05-01 | 1 | AGENT-08 | T-05-01 | `--print-config-style` writes nothing, opens no picker | unit + real binary | `go test ./internal/agents/ ./internal/cli/ -run 'TestCapabilitiesDeclared$\|TestInstallPrintConfigStyle'` | ❌ W0 | ⬜ pending |
| 05-01-T2 | 05-01 | 1 | AGENT-08 | T-05-03 | table == DescribePaths == what Install writes | unit (guard) | `go test ./internal/agents/ -run 'TestCapabilitiesTableDrivesDerivations$\|TestCapabilitiesMatchInstallWrites$'` | ❌ W0 | ⬜ pending |
| 05-01-T3 | 05-01 | 1 | AGENT-08 | T-05-02, T-05-03 | plain golden frozen; Family (a) RED | golden + drift + mutation | `go test ./internal/cli/ -run 'TestPlainGolden$\|TestInstallPrintConfigStyle_StyledStripsToPlain$' && task docs:cli:drift` | ❌ W0 | ⬜ pending |
| 05-02-T1 | 05-02 | 2 | AGENT-09, AGENT-13 | T-05-05, T-05-06 | foreign dir kept; package deleted only when `targets` empties | unit (RED first) | `go test ./internal/agents/ -run 'TestSharedSkillPackage_\|TestManifest_'` | ❌ W0 | ⬜ pending |
| 05-02-T2 | 05-02 | 2 | AGENT-09 | T-05-07, T-05-08 | never unlink a user symlink; dangling link healed | unit (RED first) | `go test ./internal/agents/ -run 'TestRemoveSkillDirIfEmpty_NeverUnlinksSymlink$\|TestSameSkillDir_\|TestSkillPackage_'` | ❌ W0 | ⬜ pending |
| 05-03-T1 | 05-03 | 3 | AGENT-09, AGENT-13 | T-05-11, T-05-12 | symlinked Claude/shared dir = one package (D-17) | unit (RED first) | `go test ./internal/agents/ -run 'TestSymlinkedSkillDir_\|TestClaude_'` | ❌ W0 | ⬜ pending |
| 05-03-T2 | 05-03 | 3 | AGENT-09 | T-05-13 | upgrade refresh only where claude ∈ targets | unit (RED first) | `go test ./internal/agents/ -run 'TestConfiguredSkillLocations_\|TestSharedSkillWriter_NotesSameDirAsClaude$'` | ✅ extend | ⬜ pending |
| 05-04-T1 | 05-04 | 4 | AGENT-13, AGENT-04, AGENT-06, AGENT-09 | T-05-15, T-05-17, T-05-18 | 8 × 2 × 2 planted-foreign ownership table | unit (RED first) | `go test ./internal/agents/ -run 'TestOwnershipExactIdentity$\|TestCursor_\|TestOpencode_'` | ❌ W0 | ⬜ pending |
| 05-04-T2 | 05-04 | 4 | AGENT-13, AGENT-06 | T-05-15, T-05-16 | Family (b1)/(b2) RED; frontmatter contract | unit + mutation | `go test ./internal/agents/ ./internal/cli/ -run 'TestSkillFrontmatterMatchesEveryWrittenDir$\|TestInstallStatus_KeptForeignIsNotAChange$'` | ❌ W0 | ⬜ pending |
| 05-05-T1 | 05-05 | 5 | AGENT-10, AGENT-11 | T-05-20, T-05-21 | harness dirs; no Kiro AGENTS.md | unit (RED first) | `go test ./internal/agents/ -run 'TestGemini_\|TestKiro_\|TestOwnershipExactIdentity$'` | ✅ extend | ⬜ pending |
| 05-05-T2 | 05-05 | 5 | AGENT-07 | T-05-20, T-05-21 | agy CLI dir; 2.0/IDE path never created | unit (RED first) | `go test ./internal/agents/ -run 'TestAntigravity_\|TestOwnershipExactIdentity$'` | ✅ extend | ⬜ pending |
| 05-06-T1 | 05-06 | 6 | AGENT-09, AGENT-10, AGENT-11 | T-05-25 | scaffold + `[ASSUMED]` rows with URL + date | file checks | Task 1 verify in 05-06-PLAN.md | ❌ W0 | ⬜ pending |
| 05-06-T2 | 05-06 | 6 | AGENT-04, AGENT-06, AGENT-09 | T-05-25 | live verdicts with negative controls; D-11; D-12 | manual (orchestrator) + verdict-line gate | Task 2 verify in 05-06-PLAN.md | ❌ W0 | ⬜ pending |
| 05-06-T3 | 05-06 | 6 | AGENT-07 | T-05-23, T-05-24 | real `$HOME` byte-identical after cleanup | manual (orchestrator) + completeness gate | Task 3 verify in 05-06-PLAN.md | ❌ W0 | ⬜ pending |
| 05-07-T1 | 05-07 | 7 | AGENT-04 | T-05-27, T-05-28 | D-11 applied as recorded; shared AGENTS.md kept until last requester | unit (conditional, RED first if observed) | `go test ./internal/agents/ ./internal/cli/ -run 'TestCapabilitiesDeclared$\|TestCursor\|TestOpencode_\|TestOwnershipExactIdentity$\|TestPlainGolden$'` | ✅ | ⬜ pending |
| 05-07-T2 | 05-07 | 7 | AGENT-08 | T-05-29 | reference regenerated; full suite green | full suite + drift | full suite command above | ✅ | ⬜ pending |

Requirement → test map (from `05-RESEARCH.md` Validation Architecture; tests assert only what the repo owns — D-00):

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AGENT-08 | `DescribePaths(loc)` == path set derived from `Capabilities()` for 8 targets × 2 locations; `--print-config-style` renders exactly the table (plain output golden-frozen, read-only: no writes, no picker) | unit + plain golden | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/agents/ -run TestCapabilities && GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run 'TestPlainGolden$\|TestPrintConfigStyle'` | ❌ W0 |
| AGENT-09 | Shared writer writes once per run (`unchanged` for later targets); manifest `targets` grows/shrinks; package removed only when `targets` empties; D-17 symlinked Claude/shared dir = one package | unit (RED first) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/agents/ -run 'TestSharedSkillPackage\|TestSymlinkedSkillDir'` | ❌ W0 |
| AGENT-04/06/07/10/11 | Each harness's install writes exactly the paths its `Capabilities()` declares (Gemini `.gemini/skills` + shared; Kiro `.kiro/skills`; Antigravity `~/.gemini/antigravity-cli/skills`; Cursor/opencode shared only); uninstall reverses them | unit | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/agents/ -run 'TestCursor_\|TestOpencode_\|TestAntigravity_\|TestGemini_\|TestKiro_'` | ✅ extend |
| AGENT-13 | Planted foreign MCP entry, foreign `.agents/skills/other/`, foreign manifest-less `codegraph/` dir and foreign instructions section survive install→uninstall byte-identical for 8 × 2; a planted ownership-recovery mutation (the `242ec0a` shape) goes RED | unit (RED first) | `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/agents/ -run TestOwnershipExactIdentity` | ❌ W0 |
| AGENT-08 / docs | Reference regenerated for `--print-config-style` in a reviewed diff; flag accounting green with no new allowlist entry | drift gate | `GOTOOLCHAIN=go1.26.6 task docs:cli:drift && GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/cli/ -run TestEveryRegisteredFlagIsAccountedFor` | ✅ |

---

## Wave 0 Requirements

- [ ] `Capabilities()` stubs on `registry_test.go`'s `fakeTarget` and `internal/cli/tui/agentpicker_test.go`'s `fakeAgentTarget` (05-RESEARCH Pitfall 1) — in the same commit as the interface change, or nothing compiles
- [ ] `internal/agents/capabilities_test.go` — D-03 table-equality guard (Family (a) RED)
- [ ] `internal/agents/ownership_test.go` — D-13 planted-foreign-entry table, RED first, doc comment citing `242ec0a` (Family (b) RED)
- [ ] `internal/agents/skillshared_test.go` — D-05/D-07/D-08 shared writer + D-17 symlinked layout (RED first)
- [ ] `--print-config-style` plain golden (`internal/cli/testdata/plain/print-config-style.golden`) captured from the first implementation and frozen
- [ ] `05-MUTATION-LOG.md` — Phase 2/3/4 shape
- [ ] `05-LIVE-SESSIONS.md` — per-harness command, transcript excerpt, negative control, verdict

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Cursor reads the shared skill; repo-root `AGENTS.md` pickup probed | AGENT-04, AGENT-09 | Harness read behaviour is live evidence, never a unit test (D-00) | Herdr `agent start … --kind cursor` in a scratch repo at project scope; list skills + where-is-X; negative control in an uninstalled scratch repo; `AGENTS.md` sentinel probe (D-11) before any instructions-target change |
| opencode reads the shared skill; duplicate with Claude's package observed | AGENT-06, AGENT-09 | Same | Herdr `--kind opencode`, project scope, WITH Claude's package also installed (D-12); record whether `codegraph` appears once, twice, or warns |
| Antigravity CLI reads `~/.gemini/antigravity-cli/skills/codegraph/` | AGENT-07 | Same; global-only so it uses the real `$HOME` | Pre-flight `readlink` (D-12 correction); before/after listing; Herdr `--kind agy`; `codegraph uninstall` cleanup |
| Gemini CLI, Kiro | AGENT-10, AGENT-11 | Not installed on the maintainer's machine (D-10) | Recorded `[ASSUMED]` with doc URL + fetch date in `05-LIVE-SESSIONS.md` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
