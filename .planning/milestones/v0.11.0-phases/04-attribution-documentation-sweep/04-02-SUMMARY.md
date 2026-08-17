---
phase: 04
plan: 02
title: Delete FLAG-PARITY and its drift guard, sweep all references
agents: inline (orchestrator, after background-agent failures)
status: complete
---

# Plan 04-02 — FLAG-PARITY Deletion (DOCS-02)

## What was done

Deleted `docs/FLAG-PARITY.md` and `internal/cli/flag_parity_test.go` (the comparison-framed doc and its drift guard) and swept every reference to either. Committed inline (`5139e60`) after the background-agent path failed on this phase.

## Deleted

- `docs/FLAG-PARITY.md` — the per-command TS 1.3.1 vs Go matrix (comparison-framed by construction).
- `internal/cli/flag_parity_test.go` — its drift guard (`TestFlagParityDocCoversRegisteredFlags`).

## Reference sweep (word-boundary identifier family: FLAG-PARITY, flag_parity, flag-parity, flagParity, TestFlagParityDocCoversRegisteredFlags)

| File | Change |
|---|---|
| `internal/cli/man.go` | "the FLAG-PARITY divergence footprint stays one documented hidden command" → "its divergence footprint stays documented in this comment" |
| `internal/mcp/instructions_contract_test.go` | dropped the `flagParityDocPath` convention reference and the "internal/cli's flag-parity test could only verify by hand" reference |
| `internal/mcp/tools_schema_drift_test.go` | dropped the `docs/FLAG-PARITY.md` and `TestFlagParityDocCoversRegisteredFlags` references |
| `.github/PULL_REQUEST_TEMPLATE/feature.md` | removed `docs/FLAG-PARITY.md` from the docs-to-update checklist |
| `.github/pull_request_template.md` | `docs/FLAG-PARITY.md` or `.planning/` → `.planning/` |
| `.github/workflows/auto-close-unsolicited-prs.yml` | `docs/FLAG-PARITY.md` or `.planning/` → `.planning/` |
| `.github/ISSUE_TEMPLATE/enhancement.yml` | "documented divergence in docs/FLAG-PARITY.md" → "documented divergence in `.planning/`" |

The `.github` re-points are Phase 4's link removal (nothing references the deleted file); the framing prose rewrite is Phase 5's PROC scope (contributor-facing issue/PR text).

## Verification (phase-final gate)

| Check | Result |
|---|---|
| Full-tree reference sweep (identifier family, excluding `.planning/`, `graphify-out/`, worktrees) | **0 residual references** |
| `go build ./...` | clean |
| `go test -count=1 ./...` | green except the documented daemon extreme-load flake (`TestRunWatchdogCancelsRunOnSimulatedReparent`, passes in isolation 1.431s, unrelated to doc edits + guard deletion) |
| `git diff --exit-code HEAD -- LICENSE` | byte-identical |
| `gh api repos/seanb4t/codegraph-go/license --jq .license.spdx_id` | `MIT` (live) |

## Deviations

Executed inline by the orchestrator (background-agent path failed on this phase). The deletion removes a live drift guard; the replacement (`DOCS-05`, a self-authored CLI reference with its own guard) is deliberately deferred per the ROADMAP note — a knowing, recorded reduction in flag-documentation coverage, not an oversight.
