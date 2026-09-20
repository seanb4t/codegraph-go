---
status: complete
phase: 05-agent-reach-capability-model-skill-in-every-harness
source: [05-VERIFICATION.md]
started: 2026-09-19T01:19:46Z
updated: 2026-09-19T01:19:46Z
---

## Current Test

[testing complete]

## Tests

### 1. Live Cursor session (D-09) and the D-11 AGENTS.md probe
expected: Cursor is shown reading the shared `.agents/skills/codegraph` package (AGENT-09's Cursor row moves from `[ASSUMED]` to verified), and the D-11 probe records AGENTS.md pickup or non-pickup, deciding whether Cursor's instructions target switches to the repo-root `AGENTS.md`.
result: skipped
reason: "Deferred follow-up: maintainer, 2026-09-18 — \"cursor is blocked, I don't have an account. Session's ready to go though\" (recorded as the D-10/D-11 maintainer decisions, f83724ee; Cursor is `[ASSUMED]`, D-11 verdict `not probed`, 05-07 took the no-change branch)."

### 2. Live Gemini CLI and Kiro sessions (D-09/D-12)
expected: The `[ASSUMED]` rows for AGENT-10 and AGENT-11 are confirmed or corrected by an observed transcript against `.gemini/skills/codegraph` and `.kiro/skills/codegraph`.
result: skipped
reason: "Deferred follow-up: D-10, accepted by the maintainer in discuss-phase — live verification covers only the harnesses installed on the maintainer's machine; Gemini CLI and Kiro are not installed (`command -v gemini`/`kiro`/`kiro-cli` exit 1, 05-LIVE-SESSIONS.md Pre-flight) and installing them is listed under Deferred Ideas as the maintainer's call."

## Summary

total: 2
passed: 0
issues: 0
pending: 0
skipped: 2
blocked: 0

## Gaps

## Deferred Follow-Ups

- test: 1
  idea: "Obtain an authenticated cursor-agent session, then run the D-09 skills/where-is-X/negative-control session and the D-11 AGENTS.md probe; apply D-11 only on an observed pickup."
  deferred_at: 2026-09-18
- test: 2
  idea: "Install Gemini CLI and Kiro, then run the D-09/D-12 live-session method against `.gemini/skills/codegraph` and `.kiro/skills/codegraph` to upgrade AGENT-10/AGENT-11 from `[ASSUMED]`."
  deferred_at: 2026-09-18
