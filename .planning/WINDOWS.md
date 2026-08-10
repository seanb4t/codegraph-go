---
schema_version: 1
open_count: 1
waived_count: 0
fixed_count: 0
total_count: 1
last_updated: 2026-08-10T14:33:31.562Z
---

# Broken Windows Ledger

> Cross-phase defect register. With `workflow.windows_enforce` enabled, `/gsd-ship` blocks while `open_count > 0`.
> Waive with `gsd-tools windows waive <id> "<reason>"` (reason required).
> Mark fixed with `gsd-tools windows fixed <id>`.

| id | phase | kind | file | line | description | status | reason | recorded_at | resolved_at |
|----|-------|------|------|------|-------------|--------|--------|-------------|-------------|
| 1 | 03 | unrun-verify | Taskfile.yml |  | release:rehearse-cask could not reach a PASS in this session — Homebrew Cask unconditionally quarantines every download, so the post-install hook's system_command execution of the installed (ad-hoc-signed, non-notarized) binary is SIGKILLed by Gatekeeper. Requires maintainer-supplied MACOS_SIGN_P12/MACOS_SIGN_PASSWORD/MACOS_NOTARY_ISSUER_ID/MACOS_NOTARY_KEY_ID/MACOS_NOTARY_KEY to complete; A1/A3 assumptions remain unconfirmed. | open |  | 2026-08-10T14:33:31.562Z |  |

````json
[
  {
    "id": 1,
    "kind": "unrun-verify",
    "phase": "03",
    "file": "Taskfile.yml",
    "line": null,
    "description": "release:rehearse-cask could not reach a PASS in this session — Homebrew Cask unconditionally quarantines every download, so the post-install hook's system_command execution of the installed (ad-hoc-signed, non-notarized) binary is SIGKILLed by Gatekeeper. Requires maintainer-supplied MACOS_SIGN_P12/MACOS_SIGN_PASSWORD/MACOS_NOTARY_ISSUER_ID/MACOS_NOTARY_KEY_ID/MACOS_NOTARY_KEY to complete; A1/A3 assumptions remain unconfirmed.",
    "status": "open",
    "reason": "",
    "recorded_at": "2026-08-10T14:33:31.562Z",
    "resolved_at": null
  }
]
````
