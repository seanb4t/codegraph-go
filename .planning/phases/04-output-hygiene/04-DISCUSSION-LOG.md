# Phase 4: Output Hygiene - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-16
**Phase:** 4-output-hygiene
**Mode:** --auto --all --chain (all areas auto-selected; recommended option chosen per question, no interactive prompts)
**Areas discussed:** Pebble logger routing shape, Error destination & seam, Debug escape hatch, HYG-02 enforcement mechanism, Verification/mutation-proofing

---

## Pebble logger routing shape (HYG-01)

| Option | Description | Selected |
|--------|-------------|----------|
| Quiet `Options.Logger` in graphstore | Unexported 3-method logger (Infof→discard, Errorf→stderr, Fatalf→pebble-default) injected at the single Open seam | ✓ |
| `NoopLoggerAndTracer` | Wholesale silence via pebble's built-in noop | |
| Custom `EventListener` | Per-event filtering | |

**Auto-selected:** Quiet `Options.Logger` — the requirement text names `pebble.Options.Logger` (locked), ROADMAP explicitly forbids wholesale silence, and `EnsureDefaults` derives the EventListener from `Options.Logger` so one field covers the whole noise surface.

---

## Error destination & seam (HYG-01)

| Option | Description | Selected |
|--------|-------------|----------|
| Package-level injectable writer → os.Stderr | Test-capturable seam, production always stderr, provenance prefix | ✓ |
| Hardcoded os.Stderr | No test seam | |
| Writer parameter on `Open` | Changes every call site | |

**Auto-selected:** Injectable package-level seam — matches the Phase-3 test-only-seam convention and the T-03-07-Leak stderr rule without touching `Open`'s signature.

---

## Debug escape hatch (HYG-01)

| Option | Description | Selected |
|--------|-------------|----------|
| No new env surface in v1.0 | Unconditional INFO discard; defer a verbose knob | ✓ |
| `CODEGRAPH_PEBBLE_LOG=1` env | Re-enable INFO for troubleshooting | |

**Auto-selected:** No new env surface — TS has no analogue; a new env var is new documented/audited surface for an unrequested convenience; real errors still surface via Errorf/Fatalf. Deferred idea recorded.

---

## HYG-02 enforcement mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| Both harness assertion + structural archtest | TEST-04 subprocess case parsing every stdout line as JSON-RPC, plus an os.Stdout/bare-Print/log.SetOutput archtest over the six serve-reachable non-CLI packages | ✓ |
| Harness assertion only | End-to-end but no build-time guard | |
| Archtest only | Structural but never proves the real transport | |

**Auto-selected:** Both — belt-and-braces is the established Phase-3 convention; neither layer alone survived past review rounds. `internal/cli` excluded (legit stdout rendering).

---

## Verification / mutation-proofing

| Option | Description | Selected |
|--------|-------------|----------|
| Mutation-proof wiring test + behavioral stderr-capture + CLI noise-absence check | Reverting `&pebble.Options{}` must go red; captured seam shows zero INFO noise while Errorf passes; one subprocess CLI case asserts no pebble-shaped stderr noise | ✓ |
| Wiring test only | Doesn't prove behavior | |
| Behavioral only | Replica-test trap (Phase-2 CR-01 lesson) | |

**Auto-selected:** All three layers — the 8×-recurred green-suite lesson demands asserting the wiring at its root cause AND observing real behavior.

---

## Claude's Discretion

- Logger type name/file placement in `internal/graphstore`; exact provenance-prefix wording; archtest mechanism (go/ast vs scanner); D-09 command choice/placement; optional trivial rate-limiting of repeated Errorf lines.

## Deferred Ideas

- Verbose/debug knob to re-enable pebble INFO logs (Phase 8 candidate at earliest)
- TUI-01 lipgloss archtest (Phase 6) mirrors the new stdout guard
- Shared in-process store handle for the >400ms lock window (Phase-3 residual)

## Reviewed Todos (not folded)

- `2026-07-14-document-release-cut-procedures-runbook.md` (score 0.4, generic keywords) — belongs with Phase 8 release hardening; identical call to Phases 1–3. NOTE: auto-mode's ≥0.4 fold threshold was deliberately overridden here — three consecutive prior phases explicitly reviewed-and-deferred this same todo, and re-litigating a locked deferral on a borderline generic-keyword match would violate the carry-forward rule. Fourth consecutive review; retitle the todo.
