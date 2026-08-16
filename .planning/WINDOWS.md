---
schema_version: 1
open_count: 11
waived_count: 0
fixed_count: 3
total_count: 14
last_updated: 2026-08-16T00:52:44.851Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 03 | unrun-verify | Taskfile.yml |  | release:rehearse-cask could not reach a PASS in this session — Homebrew Cask unconditionally quarantines every download, so the post-install hook's system_command execution of the installed (ad-hoc-signed, non-notarized) binary is SIGKILLed by Gatekeeper. Requires maintainer-supplied MACOS_SIGN_P12/MACOS_SIGN_PASSWORD/MACOS_NOTARY_ISSUER_ID/MACOS_NOTARY_KEY_ID/MACOS_NOTARY_KEY to complete; A1/A3 assumptions remain unconfirmed. | fixed |  | 2026-08-10T14:33:31.562Z | 2026-08-10T17:06:15.507Z |
| 2 | 03 | unrun-verify | .planning/phases/03-homebrew-tap-cask/03-03-PLAN.md |  | Task 2 GitHub App creation, Task 1 job-output-survival measurement, and Task 3 release.yml wiring all blocked: no authenticated browser session reachable for agent-browser to drive GitHub App creation UI. See 03-03-SUMMARY.md Deviations. | fixed |  | 2026-08-10T17:04:43.293Z | 2026-08-10T18:14:21.285Z |
| 3 | 03 | unrun-verify | .goreleaser.yaml |  | 03-02 Task 1 halted before any code changes: its precondition ('task release:rehearse-cask from plan 03-01 exits 0 on this machine') is unmet in this executor's worktree — MACOS_SIGN_P12/MACOS_SIGN_PASSWORD/MACOS_NOTARY_ISSUER_ID/MACOS_NOTARY_KEY_ID/MACOS_NOTARY_KEY are unset, no .env is present in the worktree (gitignored, not checked out), and 'op run' cannot substitute since the reference file is unreachable. Confirmed: 'task release:rehearse-cask' exits non-zero immediately at its own precondition gate (no side effects). All of Task 1/2/3 in 03-02-PLAN.md are blocked pending an orchestrator-run rehearsal with real credentials, mirroring how 03-01's tracer PASS was ultimately obtained. | fixed |  | 2026-08-10T17:21:02.719Z | 2026-08-10T18:04:15.448Z |
| 4 | 05 | deviation | internal/cli/search.go | 55 | CODE-01 census gap: 'no TS precedent' comparison framing found outside plan 05-05's declared file scope; not fixed by any plan in this wave | open |  | 2026-08-16T00:13:57.761Z |  |
| 5 | 05 | deviation | internal/cli/node.go | 60 | CODE-01 census gap: 'no TS precedent for this CLI placement' comparison framing found outside plan 05-05's declared file scope | open |  | 2026-08-16T00:13:58.816Z |  |
| 6 | 05 | deviation | internal/cli/files.go | 18 | CODE-01 census gap: 'matches TS files --filter' comparison framing (also :90) found outside plan 05-05's declared file scope | open |  | 2026-08-16T00:13:59.903Z |  |
| 7 | 05 | deviation | internal/cli/uninit.go | 56 | CODE-01 census gap: 'mirrors TS' comparison framing found outside plan 05-05's declared file scope | open |  | 2026-08-16T00:14:01.091Z |  |
| 8 | 05 | deviation | internal/cli/serve.go | 137 | CODE-01 census gap: 'D-12/D-13 verbatim TS disabled message' comparison framing found outside plan 05-05's declared file scope (its test companion serve_test.go was in scope and fixed) | open |  | 2026-08-16T00:14:02.104Z |  |
| 9 | 05 | deviation | internal/cli/githooks_test.go | 50 | CODE-01 census gap: 'verbatim TS sync/git-hooks.js begin marker' comparison framing found outside plan 05-05's declared file scope (distinct file from internal/githooks/githooks_test.go, which was in scope and fixed) | open |  | 2026-08-16T00:14:03.772Z |  |
| 10 | 05 | deviation | internal/mcp/tools.go | 369 | CODE-01 census gap: 'Go-vs-TS divergence: TS returns markdown from every MCP tool' and 'mirroring TS's' (:528) comparison framing found outside plan 05-05's declared file scope | open |  | 2026-08-16T00:14:04.811Z |  |
| 11 | 05 | deviation | internal/agents/codex.go | 14 | CODE-01 census gap: 'TS integrates with' / 'mirrors TS's own toml.ts' (:17) comparison framing found outside plan 05-05's declared internal/agents file list (task 1 covered 10 named files; codex.go/opencode.go/claude.go were not among them) | open |  | 2026-08-16T00:14:05.866Z |  |
| 12 | 05 | unrun-verify | internal/daemon/daemon_test.go | 352 | TestRunWatchdogCancelsRunOnSimulatedReparent is load-sensitive: passes isolated (1.4s) and as a lone package (64.7s, identical to pre-merge base 65.7s), but times out at 250s and FAILS inside 'go test ./...' alongside ~49 parallel packages. NOT caused by phase 5 — internal/daemon has zero diff and the internal/graphstore diff is deletions-only (zero added lines). The test asserts a wall-clock watchdog deadline a loaded runner cannot meet, making the full-suite gate non-deterministic. | open |  | 2026-08-16T00:28:48.387Z |  |
| 13 | 05 | deviation | docs/RELEASE.md | 337 | PRE-EXISTING doc staleness (not phase-5 caused): the dependency paragraph states '27 direct requires' with '14 tree-sitter' leaving 'the remaining 13', but go.mod now has 32 direct requires (14 tree-sitter, 18 remaining). It also credits the MCP server to 'mark3labs/mcp-go' while go.mod actually requires 'modelcontextprotocol/go-sdk v1.7.0'. Phase 5 removed only the now-false modernc.org/sqlite migration-tool clause; the counts and the MCP attribution were already drifted and were deliberately NOT renumbered, since inventing a corrected figure inside independently-stale arithmetic would substitute one wrong number for another. | open |  | 2026-08-16T00:29:32.708Z |  |
| 14 | 05 | deviation | internal/query/traverse_test.go | 780 | traverse_test.go:780 doc comment cites 'TS test-files-as-leaves pruning' — genuine CODE-01 comparison-framing hit, but traverse_test.go is not in 05-04's declared files_modified (only traverse.go is), so left unedited per scope discipline; a future sweep pass should fold this file into its edit set | open |  | 2026-08-16T00:52:44.851Z |  |

````json
[
  {
    "id": 1,
    "kind": "unrun-verify",
    "phase": "03",
    "file": "Taskfile.yml",
    "line": null,
    "description": "release:rehearse-cask could not reach a PASS in this session — Homebrew Cask unconditionally quarantines every download, so the post-install hook's system_command execution of the installed (ad-hoc-signed, non-notarized) binary is SIGKILLed by Gatekeeper. Requires maintainer-supplied MACOS_SIGN_P12/MACOS_SIGN_PASSWORD/MACOS_NOTARY_ISSUER_ID/MACOS_NOTARY_KEY_ID/MACOS_NOTARY_KEY to complete; A1/A3 assumptions remain unconfirmed.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-10T14:33:31.562Z",
    "resolved_at": "2026-08-10T17:06:15.507Z"
  },
  {
    "id": 2,
    "kind": "unrun-verify",
    "phase": "03",
    "file": ".planning/phases/03-homebrew-tap-cask/03-03-PLAN.md",
    "line": null,
    "description": "Task 2 GitHub App creation, Task 1 job-output-survival measurement, and Task 3 release.yml wiring all blocked: no authenticated browser session reachable for agent-browser to drive GitHub App creation UI. See 03-03-SUMMARY.md Deviations.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-10T17:04:43.293Z",
    "resolved_at": "2026-08-10T18:14:21.285Z"
  },
  {
    "id": 3,
    "kind": "unrun-verify",
    "phase": "03",
    "file": ".goreleaser.yaml",
    "line": null,
    "description": "03-02 Task 1 halted before any code changes: its precondition ('task release:rehearse-cask from plan 03-01 exits 0 on this machine') is unmet in this executor's worktree — MACOS_SIGN_P12/MACOS_SIGN_PASSWORD/MACOS_NOTARY_ISSUER_ID/MACOS_NOTARY_KEY_ID/MACOS_NOTARY_KEY are unset, no .env is present in the worktree (gitignored, not checked out), and 'op run' cannot substitute since the reference file is unreachable. Confirmed: 'task release:rehearse-cask' exits non-zero immediately at its own precondition gate (no side effects). All of Task 1/2/3 in 03-02-PLAN.md are blocked pending an orchestrator-run rehearsal with real credentials, mirroring how 03-01's tracer PASS was ultimately obtained.",
    "status": "fixed",
    "reason": "",
    "recorded_at": "2026-08-10T17:21:02.719Z",
    "resolved_at": "2026-08-10T18:04:15.448Z"
  },
  {
    "id": 4,
    "kind": "deviation",
    "phase": "05",
    "file": "internal/cli/search.go",
    "line": 55,
    "description": "CODE-01 census gap: 'no TS precedent' comparison framing found outside plan 05-05's declared file scope; not fixed by any plan in this wave",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-16T00:13:57.761Z",
    "resolved_at": null
  },
  {
    "id": 5,
    "kind": "deviation",
    "phase": "05",
    "file": "internal/cli/node.go",
    "line": 60,
    "description": "CODE-01 census gap: 'no TS precedent for this CLI placement' comparison framing found outside plan 05-05's declared file scope",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-16T00:13:58.816Z",
    "resolved_at": null
  },
  {
    "id": 6,
    "kind": "deviation",
    "phase": "05",
    "file": "internal/cli/files.go",
    "line": 18,
    "description": "CODE-01 census gap: 'matches TS files --filter' comparison framing (also :90) found outside plan 05-05's declared file scope",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-16T00:13:59.903Z",
    "resolved_at": null
  },
  {
    "id": 7,
    "kind": "deviation",
    "phase": "05",
    "file": "internal/cli/uninit.go",
    "line": 56,
    "description": "CODE-01 census gap: 'mirrors TS' comparison framing found outside plan 05-05's declared file scope",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-16T00:14:01.091Z",
    "resolved_at": null
  },
  {
    "id": 8,
    "kind": "deviation",
    "phase": "05",
    "file": "internal/cli/serve.go",
    "line": 137,
    "description": "CODE-01 census gap: 'D-12/D-13 verbatim TS disabled message' comparison framing found outside plan 05-05's declared file scope (its test companion serve_test.go was in scope and fixed)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-16T00:14:02.104Z",
    "resolved_at": null
  },
  {
    "id": 9,
    "kind": "deviation",
    "phase": "05",
    "file": "internal/cli/githooks_test.go",
    "line": 50,
    "description": "CODE-01 census gap: 'verbatim TS sync/git-hooks.js begin marker' comparison framing found outside plan 05-05's declared file scope (distinct file from internal/githooks/githooks_test.go, which was in scope and fixed)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-16T00:14:03.772Z",
    "resolved_at": null
  },
  {
    "id": 10,
    "kind": "deviation",
    "phase": "05",
    "file": "internal/mcp/tools.go",
    "line": 369,
    "description": "CODE-01 census gap: 'Go-vs-TS divergence: TS returns markdown from every MCP tool' and 'mirroring TS's' (:528) comparison framing found outside plan 05-05's declared file scope",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-16T00:14:04.811Z",
    "resolved_at": null
  },
  {
    "id": 11,
    "kind": "deviation",
    "phase": "05",
    "file": "internal/agents/codex.go",
    "line": 14,
    "description": "CODE-01 census gap: 'TS integrates with' / 'mirrors TS's own toml.ts' (:17) comparison framing found outside plan 05-05's declared internal/agents file list (task 1 covered 10 named files; codex.go/opencode.go/claude.go were not among them)",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-16T00:14:05.866Z",
    "resolved_at": null
  },
  {
    "id": 12,
    "kind": "unrun-verify",
    "phase": "05",
    "file": "internal/daemon/daemon_test.go",
    "line": 352,
    "description": "TestRunWatchdogCancelsRunOnSimulatedReparent is load-sensitive: passes isolated (1.4s) and as a lone package (64.7s, identical to pre-merge base 65.7s), but times out at 250s and FAILS inside 'go test ./...' alongside ~49 parallel packages. NOT caused by phase 5 — internal/daemon has zero diff and the internal/graphstore diff is deletions-only (zero added lines). The test asserts a wall-clock watchdog deadline a loaded runner cannot meet, making the full-suite gate non-deterministic.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-16T00:28:48.387Z",
    "resolved_at": null
  },
  {
    "id": 13,
    "kind": "deviation",
    "phase": "05",
    "file": "docs/RELEASE.md",
    "line": 337,
    "description": "PRE-EXISTING doc staleness (not phase-5 caused): the dependency paragraph states '27 direct requires' with '14 tree-sitter' leaving 'the remaining 13', but go.mod now has 32 direct requires (14 tree-sitter, 18 remaining). It also credits the MCP server to 'mark3labs/mcp-go' while go.mod actually requires 'modelcontextprotocol/go-sdk v1.7.0'. Phase 5 removed only the now-false modernc.org/sqlite migration-tool clause; the counts and the MCP attribution were already drifted and were deliberately NOT renumbered, since inventing a corrected figure inside independently-stale arithmetic would substitute one wrong number for another.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-16T00:29:32.708Z",
    "resolved_at": null
  },
  {
    "id": 14,
    "kind": "deviation",
    "phase": "05",
    "file": "internal/query/traverse_test.go",
    "line": 780,
    "description": "traverse_test.go:780 doc comment cites 'TS test-files-as-leaves pruning' — genuine CODE-01 comparison-framing hit, but traverse_test.go is not in 05-04's declared files_modified (only traverse.go is), so left unedited per scope discipline; a future sweep pass should fold this file into its edit set",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-16T00:52:44.851Z",
    "resolved_at": null
  }
]
````
