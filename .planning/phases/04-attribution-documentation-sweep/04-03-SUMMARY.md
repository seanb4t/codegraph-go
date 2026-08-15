---
phase: 04
plan: 03
title: Doc sweep — CONTRIBUTING and capability matrix
agents: inline (orchestrator, after background-agent failures)
status: complete
---

# Plan 04-03 — Doc Sweep (DOCS-01/03/04)

## What was done

Swept comparison framing from `CONTRIBUTING.md` and `docs/LANGUAGE-CAPABILITY-MATRIX.md`, and verified the DOCS-04 surfaces clean. Committed inline (`cbaeeb2`) after the background-agent path failed three times on this phase.

## CONTRIBUTING.md (DOCS-01)

- **Issue-first paragraph** (was "this project ports observable behavior from another implementation, and a change that looks like an obvious improvement is sometimes a deliberate parity decision recorded in `docs/FLAG-PARITY.md` or `.planning/`") → "behavior is pinned by the project's own golden-corpus suite and its planning record, and a change that looks like an obvious improvement can still contradict a deliberate decision recorded in `.planning/`." Removes the comparison framing AND the `docs/FLAG-PARITY.md` link (the file 04-02 deletes).
- **`task test` description:** "unit, golden parity, subprocess integration, isolated daemon, and race" → "unit, golden-corpus behavioral, subprocess integration, isolated daemon, and race" (Phase 2 rename).
- **Test-suite description:** "golden-corpus behavioral fixtures diffed against frozen TypeScript CodeGraph v1.3.1 output" → "golden-corpus behavioral fixtures frozen from codegraph-go's own output against pinned third-party corpora" — the prior text was stale (Phase 2's goldens are frozen from Go's own output, not TS).

## docs/LANGUAGE-CAPABILITY-MATRIX.md (DOCS-03)

- Coverage legend: "corresponding golden-parity test" → "corresponding behavioral test".
- `go` gaps: "Go is the reference implementation, validated end-to-end since Phase 2/3 against the pinned `weft` golden-parity corpus" → "Go is fully covered, validated end-to-end against the pinned golden corpora" (weft corpus deleted in Phase 2 FIXT-04).
- Imports-edge limitation: "golden-parity/self-consistency tests" → "behavioral/self-consistency tests".

## DOCS-04 (verify-only, no edits)

SECURITY.md, CODE_OF_CONDUCT.md, PARSER-DECISION.md, RELEASE.md, RELEASE-PROCEDURES.md, MCP-2026-07-28-SCOPING.md, MCP-8-AGENT-AUDIT.md — verified clean with a word-boundary sweep. Keep-with-reason (recorded): RELEASE-PROCEDURES's `gsd/v1.0-drop-in-parity-human-ux` is a historical branch name (not a comparison claim); its "Recorded divergences" are plan-vs-shipped decision records (not implementation comparison); "upstream" is pipeline direction; matrix:181's "original fwcd recommendation" is a historical research reference.

## Verification

- `matrix_test.go` green (`TestMatrix` ok).
- Word-boundary framing sweep on both edited files: clean (matrix:181 is the recorded keep).
- `go test -count=1 ./...` green except the documented daemon extreme-load flake (passes in isolation, unrelated).

## Deviations

Executed inline by the orchestrator. The background-agent path failed three times on this phase (04-01 first attempt, 04-03 first attempt, 04-03 retry — empty-body/truncated returns, idle timeouts, DSML corruption). The remaining Phase 4 work was small and well-specified by the plans and the research census, so it was completed inline per the workflow's `--interactive` mode. README verification was 04-01's single owner (not re-touched).
