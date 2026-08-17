---
plan: 06-06
title: Durable memory-store sweep (MEM-01)
status: complete
requirements-completed: [MEM-01]
---

# 06-06 — Durable memory-store sweep

## What was done

Enumerated the engram spine `repo:github.com/seanb4t/codegraph-go` to exhaustion (169 records,
5 pages, `next_cursor` empty on the last), plus the 3 rules in `rule:repo:...`. Classified every
record against the plan's criterion, took the classification to the maintainer at Task 2's blocking
checkpoint, and applied the 4 approved corrections by supersede.

| Superseded | Correcting record |
|---|---|
| `3ekc84hbqt` (parity ruling; "functionality floor and migration path still bind") | `xj1stbrsw6` |
| `gw79qy2a9z` (v0.11.0 scoping; migrate "PRESERVED-reframed") | `gxwkk3necn` |
| `agggksad53` (`FilesByLanguage` MUST be `json:"-"`) | `b9wjge7375` |
| `7f0pq2wepv` (same suppression ruling via parity reasoning) | `mw5z9s9bft` |

Full enumeration, classification table, per-record verdicts and reasons: `06-MEMORY-SWEEP.md`.

## Execution deviation

Task 1's precondition — engram MCP tools registered and callable — was UNMET in the spawned
worktree executor. `engram` is installed at user scope and is not in this repo's `.mcp.json`, so a
worktree-spawned executor inherits no `mcp__engram__*` tools. The executor HALTED per the
precondition's explicit instruction ("do not substitute recall context, do not infer the
population"), committed its halt documentation, and returned. That halt was correct and is
preserved in `06-MEMORY-SWEEP.md` as an accurate record of that session.

The orchestrator session DOES carry the engram tool surface, so it resumed and completed Tasks 1-3
directly — the first of the two resolutions the halt itself named. Per D-16 the plan was not marked
complete until a session with real tool access performed the live enumeration; that condition is
now satisfied.

## Verification

- `get_memory("3ekc84hbqt")` returns original content intact with `superseded_by` stamped —
  history preserved, nothing deleted or overwritten (T-06-18's mitigation upheld).
- A `search_memory` probe phrased in the superseded records' own terms returns all 4 correcting
  records and none of the 4 originals — recall is clean, which is the property a future session
  actually depends on.
- `MEM01_ENUMERATION_STATUS=complete-169-records-5-pages`
- `MEM01_STORE_STATUS=complete-4-superseded-0-deleted-0-overwritten`

## Lesson recorded

Four supersede attempts failed with `not found: ["<id>"]` and were misdiagnosed as an ownership
write-gate denial. The real cause was a parameter-shape error: `supersedes` takes a string and an
array was passed, so the error was echoing the argument back rather than reporting a missing
record. A "not found" that quotes your own argument verbatim is reporting your argument. Reading
the tool's schema found it in one step; the plausible-sounding infrastructure story had been built
without doing that.
