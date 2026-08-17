---
phase: 02
plan: 04
title: Re-freeze goldens from Go output against the locked corpora
agents: gsd-executor (worktree isolation) + human-verify checkpoint
status: complete
---

# Plan 02-04 — Re-freeze (FIXT-06, Diff B part 2)

## What was done

Re-froze the golden suite from codegraph-go's own output against the Phase-1-locked corpora, then surfaced the diff at a blocking human-verify checkpoint before commit. Approved by the user and committed (`b8523e8`).

## The re-frozen set — 26 goldens

| Corpus | Goldens | Source |
|--------|---------|--------|
| hugo (go/tsjs) | 6 | CLI explore/node, multi, MCP explore/node |
| guava (java) | 6 | CLI explore/node, multi, MCP explore/node |
| serilog (csharp) | 6 | CLI explore/node, multi, MCP explore/node |
| requests (python) | 6 | CLI explore/node, multi, MCP explore/node |
| behavioral (in-repo) | 2 | CLI explore-multi, node-multi |

All 26 non-empty, marker-bearing, parseable — verified by `TestReFrozenGoldensValid` (26/26), which enumerates the expected set rather than globbing existing fixtures.

## Sequence

`task corpora:fetch` → `task corpora:assert` (4/4 locked corpora, four-part integrity check) → `task golden:regen` (`go run ./testdata/golden/gocapture`).

## Proofs

- **Determinism:** sha256 `5ae00723…1d5446` identical across two consecutive `golden:regen` runs.
- **Guard:** `TestReFrozenGoldensValid` passes 26/26; a deliberately-deleted golden fails it (25/26), so it is not vacuous.
- **Attribution:** no renames in the diff; no added framing strings; every changed golden byte traces to re-capture from Go output. The parked `-p synthetic-parity` command-field byte is retired by re-capture (H2-1).
- **Criterion 2:** the rename diff (02-01/02-02) moved no golden byte; this diff moves no identifier.

## Deviation (Rule 2 — missing critical functionality, user-approved)

`testdata/golden/gocapture/main.go` appears in the diff outside the plan's stated "no identifier" scope: the serilog multi-symbol was `ILogger`, which the C# tree-sitter parser does not resolve as a named symbol, so the serilog golden set could not be produced. Changed to `LogEvent` (one line). Surfaced at the checkpoint and approved.

## Deviations (execution)

The re-freeze output was produced and verified but left uncommitted awaiting the human-verify checkpoint; the executor returned the checkpoint. The orchestrator independently verified the attribution (no renames, no added framing, guard green, determinism, behavioral envelope) before surfacing it to the user, who approved. Committed by the orchestrator with the user's approval.
