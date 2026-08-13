---
phase: 7
slug: codegraph-install-skill-hooks-distribution-claude-code
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-13
---

# Phase 7 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (stdlib), matching every existing test in `internal/agents` and `internal/upgrade` |
| **Config file** | none — plain `go test` |
| **Quick run command** | `go test ./internal/agents/... ./internal/cli/... ./internal/upgrade/...` |
| **Full suite command** | `task test` (runs `test:unit`, `test:golden`, `test:integration`, `test:wireoracle`, `test:daemon`, `test:race` serially) |
| **Estimated runtime** | ~30–90 seconds quick, several minutes full suite |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/agents/... ./internal/cli/... ./internal/upgrade/...`
- **After every plan wave:** Run `task test`
- **Before `/gsd-verify-work`:** Full suite must be green, plus a live-session smoke check of the GLOBAL install path (Assumption A3 — `~`-expansion / home-relative path correctness), given this phase's roadmap-declared "highest-risk phase of the milestone" status
- **Max feedback latency:** ~30 seconds (quick command)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 07-01-01 | 01 | 1 | AGENT-01 | T-07-01 | Double-install produces byte-identical (JSON-deep-equal) SKILL.md + settings.json hooks fragment | unit | `go test ./internal/agents/... -run TestClaude` | ❌ W0 | ⬜ pending |
| 07-01-02 | 01 | 1 | AGENT-02 | T-07-01 | Uninstall preserves unrelated pre-existing hooks matcher-blocks and permission entries | unit | `go test ./internal/agents/... -run TestClaude` | ❌ W0 | ⬜ pending |
| 07-01-03 | 01 | 1 | AGENT-03 | T-07-02 | Manifest records version + per-file hashes; `upgrade` re-invokes `Install()` for previously-configured targets after a successful swap | unit | `go test ./internal/agents/... ./internal/cli/...` | ❌ W0 | ⬜ pending |
| 07-01-04 | 01 | 1 | roadmap criterion 4 | T-07-01 | {install, uninstall} × {read-error, malformed} all surface an error, file left byte-untouched | unit | `go test ./internal/agents/...` | ❌ W0 | ⬜ pending |
| 07-01-05 | 01 | 1 | AGENT-01 | T-07-03 | Installed nudge script is executable (mode 0755) | unit | assertion on `os.Stat(scriptPath).Mode()&0o111 != 0` after a real `Install()` call | ❌ W0 | ⬜ pending |
| 07-01-06 | 01 | 1 | AGENT-01 | T-07-04 | Global-scope hook command resolves to the script's actual global-install location, distinct from local's `${CLAUDE_PROJECT_DIR}`-relative command | unit | assertion on the exact `command` string written for `LocationGlobal` vs. `LocationLocal` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] New root-level `go:embed` source file (e.g. `claudeassets.go`) — does not exist yet
- [ ] New array-scoped hooks merge helpers (`writeHookEntry`/`removeHookEntry`) in `internal/agents` — do not exist yet
- [ ] New strict JSON reader for the hooks-merge/manifest read path (modeled on `internal/githooks`'s skip-and-accumulate discipline, not `readJSONFile`'s empty-map fallback) — does not exist yet
- [ ] Manifest read/write/hash helpers (`crypto/sha256`) — do not exist yet
- [ ] `claudeSkillDirPath(loc)`/`claudeHooksScriptPath(loc)`/`claudeManifestPath(loc)` location-aware path helpers mirroring `claudeConfigPath` — do not exist yet
- [ ] Upgrade auto-refresh orchestration in `internal/cli/upgrade.go` (after `upgradeRunFunc` succeeds) — does not exist yet
- [ ] Test fixtures: a `settings.json` with an unrelated `SessionStart` matcher-block AND an unrelated event key (AGENT-02 byte-invariance proof); a malformed `settings.json` fixture (read-error matrix)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|--------------------|
| Global-scope (`~`-expanded or home-relative) hook command actually fires the nudge in a live Claude Code session | AGENT-01, Assumption A3 | Claude Code's own runtime interpretation of a global-scope `command` string was not independently re-verified against a live session this research pass (only local/`${CLAUDE_PROJECT_DIR}` was proven live in Phase 6) | Install to `--location global` in a scratch home directory, start a real Claude Code session in an unrelated `.codegraph/`-indexed repo, confirm the nudge line appears on `SessionStart` |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
