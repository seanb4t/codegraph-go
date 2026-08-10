---
schema_version: 1
open_count: 2
waived_count: 0
fixed_count: 1
total_count: 3
last_updated: 2026-08-10T17:21:02.719Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 03 | unrun-verify | Taskfile.yml |  | release:rehearse-cask could not reach a PASS in this session — Homebrew Cask unconditionally quarantines every download, so the post-install hook's system_command execution of the installed (ad-hoc-signed, non-notarized) binary is SIGKILLed by Gatekeeper. Requires maintainer-supplied MACOS_SIGN_P12/MACOS_SIGN_PASSWORD/MACOS_NOTARY_ISSUER_ID/MACOS_NOTARY_KEY_ID/MACOS_NOTARY_KEY to complete; A1/A3 assumptions remain unconfirmed. | fixed |  | 2026-08-10T14:33:31.562Z | 2026-08-10T17:06:15.507Z |
| 2 | 03 | unrun-verify | .planning/phases/03-homebrew-tap-cask/03-03-PLAN.md |  | Task 2 GitHub App creation, Task 1 job-output-survival measurement, and Task 3 release.yml wiring all blocked: no authenticated browser session reachable for agent-browser to drive GitHub App creation UI. See 03-03-SUMMARY.md Deviations. | open |  | 2026-08-10T17:04:43.293Z |  |
| 3 | 03 | unrun-verify | .goreleaser.yaml |  | 03-02 Task 1 halted before any code changes: its precondition ('task release:rehearse-cask from plan 03-01 exits 0 on this machine') is unmet in this executor's worktree — MACOS_SIGN_P12/MACOS_SIGN_PASSWORD/MACOS_NOTARY_ISSUER_ID/MACOS_NOTARY_KEY_ID/MACOS_NOTARY_KEY are unset, no .env is present in the worktree (gitignored, not checked out), and 'op run' cannot substitute since the reference file is unreachable. Confirmed: 'task release:rehearse-cask' exits non-zero immediately at its own precondition gate (no side effects). All of Task 1/2/3 in 03-02-PLAN.md are blocked pending an orchestrator-run rehearsal with real credentials, mirroring how 03-01's tracer PASS was ultimately obtained. | open |  | 2026-08-10T17:21:02.719Z |  |

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
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-10T17:04:43.293Z",
    "resolved_at": null
  },
  {
    "id": 3,
    "kind": "unrun-verify",
    "phase": "03",
    "file": ".goreleaser.yaml",
    "line": null,
    "description": "03-02 Task 1 halted before any code changes: its precondition ('task release:rehearse-cask from plan 03-01 exits 0 on this machine') is unmet in this executor's worktree — MACOS_SIGN_P12/MACOS_SIGN_PASSWORD/MACOS_NOTARY_ISSUER_ID/MACOS_NOTARY_KEY_ID/MACOS_NOTARY_KEY are unset, no .env is present in the worktree (gitignored, not checked out), and 'op run' cannot substitute since the reference file is unreachable. Confirmed: 'task release:rehearse-cask' exits non-zero immediately at its own precondition gate (no side effects). All of Task 1/2/3 in 03-02-PLAN.md are blocked pending an orchestrator-run rehearsal with real credentials, mirroring how 03-01's tracer PASS was ultimately obtained.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-10T17:21:02.719Z",
    "resolved_at": null
  }
]
````
