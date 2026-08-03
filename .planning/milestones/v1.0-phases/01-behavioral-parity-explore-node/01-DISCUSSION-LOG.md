# Phase 1: Behavioral Parity — explore & node - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-14
**Phase:** 1-behavioral-parity-explore-node
**Mode:** `--auto` (single-pass, recommended option auto-selected per gray area)
**Areas discussed:** TS ground-truth capture, Fixture equivalence oracle, Fixture corpus, RWR determinism & tie-breaking

---

## TS Ground-Truth Capture

| Option | Description | Selected |
|--------|-------------|----------|
| White+black box | Read installed TS dist source for exact params AND capture live TS CLI+MCP golden outputs | ✓ |
| Black-box only | Capture live outputs; treat the algorithm as opaque | |
| White-box only | Read source; derive expected outputs by hand | |

**Auto-selected:** White+black box (recommended default → D-01)
**Notes:** TS 1.3.1 confirmed installed (`/opt/homebrew/lib/node_modules/@colbymchenry/codegraph/dist/`, `codegraph --version` → 1.3.1). Capture must happen while TS is still installed — the existing `testdata/golden/capture.sh` already requires the live CLI on PATH. Researcher pins every numeric constant from the dist source; no guessed approximations.

---

## Fixture Equivalence Oracle

| Option | Description | Selected |
|--------|-------------|----------|
| Normalized/structural equivalence | Canonicalize paths/whitespace/FP-jitter; assert order+membership+warnings+headers+counts; byte-identical where TS is deterministic; document allowed divergences | ✓ |
| Strict byte-identical | Require byte-for-byte identical output everywhere | |
| Loose quality check | Assert "good enough" relevance without strict ordering | |

**Auto-selected:** Normalized/structural equivalence (recommended default → D-02)
**Notes:** A JS float RWR and a Go float RWR are not guaranteed bit-identical. Byte-identity is the target only where TS output is deterministic (single-def node, header/warning strings, budget counts). Standing lesson: template-parity ≠ behavior-parity.

---

## Fixture Corpus

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse pinned corpus + add synthetic | Keep `testdata/golden/corpus/{colbymchenry-codegraph,weft-go}` and add an overload/multi-word/Test*-heavy fixture; run on BOTH CLI + MCP | ✓ |
| Existing corpus only | Reuse the v0.1 golden corpus unchanged | |
| Synthetic only | Build a fresh synthetic corpus, drop the existing one | |

**Auto-selected:** Reuse + add synthetic (recommended default → D-03)
**Notes:** The existing corpus proved template shape (single unambiguous symbols) — the v0.1 blind spot. The synthetic fixture targets exactly the gap: overloaded/same-named symbols (node multi-def), multi-word queries (tokenization), Test*-heavy connectivity (file-relevance gate + no-covering-tests), and structural-beats-lexical ordering (RWR).

---

## RWR Determinism & Tie-Breaking

| Option | Description | Selected |
|--------|-------------|----------|
| Fixed iters + stable tie-break + rounding | 25 fixed iterations, deterministic seed order, score-desc-then-Id-asc, round scores before compare | ✓ |
| Raw FP scores | Sort by raw float score, accept jitter | |
| Score-only sort | Sort by score with no secondary key | |

**Auto-selected:** Fixed iters + stable tie-break + rounding (recommended default → D-04)
**Notes:** The golden-corpus contract and the fixture harness both require reproducible ordering. No convergence-threshold early-exit (would vary run-to-run); reuse the codebase-wide lowest-`Id` tie-break convention.

---

## Claude's Discretion

- Internal package layout within `internal/query` for the RWR pipeline, score/rank data types, and the harness normalization helpers — planner/executor choose, subject to the shared-engine + plain-text constraints.

## Deferred Ideas

- Release maintainer runbook todo (score 0.40) — reviewed, NOT folded; belongs to Phase 8 (REL). Scope guardrail overrode the `--auto` ≥0.4 auto-fold default (docs-vs-algorithm mismatch).
- `query`/`search` relevance ranking — stays lexical; only `explore` gets RWR in this phase.
- Styling/color for explore/node — Phase 6 rendering seam.
- `status` content + worktree awareness — Phase 2.
