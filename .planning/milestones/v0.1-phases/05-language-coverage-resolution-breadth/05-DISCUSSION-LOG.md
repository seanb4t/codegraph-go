# Phase 5: Language Coverage & Resolution Breadth - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-11
**Phase:** 5-Language Coverage & Resolution Breadth
**Mode:** `--auto` (all gray areas auto-selected; recommended default chosen per area)
**Areas discussed:** Extractor & discovery architecture, Cross-file resolution fidelity, Dispatch synthesis & provenance, Framework-aware routing, Coverage policy, Validation method

---

## Extractor & Discovery Architecture

| Option | Description | Selected |
|--------|-------------|----------|
| Shared `LanguageExtractor` interface + per-language packages behind a registry, reusing FileResult/RefKind vocabulary | Mirrors the existing per-language parser-constructor pattern; most control over resolution; additive vocabulary | ✓ |
| Pure tree-sitter `.scm`-query-driven generic extractor | Scales to many languages with less code, but weaker control over per-language resolution and determinism | |
| Copy-paste per-language extractors | Fast to start, unmaintainable across 10+ languages | |

**Auto-selected:** Interface + registry (recommended). Discovery generalizes to an extension→language registry driving a generic walker with per-language project-descriptor hooks; go.mod becomes the first hook impl.
**Notes:** Parser layer already generalized this way (`NewGoParser`/`NewPythonParser`). `.scm` tag queries allowed as a boilerplate-reduction lever, not as the architecture (D-02).

---

## Cross-File Resolution Fidelity

| Option | Description | Selected |
|--------|-------------|----------|
| Per-language resolver behind shared seam; tiered fidelity (priority-4 full, mainstream-6 documented-partial); fold in deferred Go fixes | Matches the 5 success criteria exactly; language-correct import/namespace semantics | ✓ |
| One generic name-based resolver for all languages | Simplest, but wrong for languages with distinct import/namespace/module models | |

**Auto-selected:** Per-language tiered resolver (recommended).
**Notes:** Folds the three prior deferrals parked at Phase 5 — WR-01 (same-package name collision), WR-02 (selector-on-non-identifier), and the call-as-argument extraction gap (D-05). Non-call reference edges stay out of scope (architectural).

---

## Dispatch Synthesis & Provenance (RES-02 / RES-03)

| Option | Description | Selected |
|--------|-------------|----------|
| Synthesize `implements` edges (heuristic provenance + Line/Col); traverse at query time; no schema bump | Linear graph size; reuses Phase-3 query-time reverse adjacency; schema already reserves the fields | ✓ |
| Materialize caller→implementation dispatch edges directly | Direct, but O(callers × impls) edge explosion on wide interfaces | |
| Bump SchemaVersion to add dedicated provenance/dispatch fields | Unnecessary — `Edge.Provenance`/`Line`/`Col`/`Metadata` already exist (additive within v1) | |

**Auto-selected:** `implements` edges + query-time traversal, no schema bump (recommended).
**Notes:** `graph.pb.go` Edge `provenance` comment literally reserves `"heuristic"` for Phase 5. Persisted reverse index stays Phase 8.

---

## Framework-Aware Routing (LANG-07)

| Option | Description | Selected |
|--------|-------------|----------|
| Per-framework detector registry keyed by (language, framework signature); `route` nodes + heuristic `handles` edges; opt-in per detected dependency | Proportional cost, no false-positive routes, framework-correct detection | ✓ |
| Generic annotation/decorator scanner across all code | Cheaper to build but noisy and framework-blind | |

**Auto-selected:** Per-framework detector registry, opt-in (recommended).

---

## Coverage Policy (LANG-06)

| Option | Description | Selected |
|--------|-------------|----------|
| Language capability matrix — committed doc + machine-readable per-language descriptor (extraction/resolution/dispatch/routing) | Makes "documented-partial" concrete and auditable | ✓ |
| Prose-only note of partial support | Easy to let gaps go silently missing | |

**Auto-selected:** Capability matrix (recommended).

---

## Validation Method

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse Phase-3 TS-CodeGraph golden-parity harness for priority-4; lighter self-consistency + coverage matrix for mainstream-6 | Holds the drop-in-parity bar; proven machinery from Phase 3 | ✓ |
| Self-consistency only (byte-identical rebuild) | Proves determinism, not parity with TS | |

**Auto-selected:** Golden-parity harness reuse (recommended).

---

## Claude's Discretion

- Package layout/naming for per-language extractors and resolvers.
- Selection of the real-world validation-corpus repo per language.
- Whether the three Go fixes (D-05) land in one plan or split, provided all stay in Phase 5.
- Exact edge-kind strings (`route`/`implements`/`handles`) and metadata keys, pending TS parity check.

## Deferred Ideas

- Persisted reverse-edge index → Phase 8.
- wazero WASM parser backend → monitored future option behind `parser.Parser`.
- Non-call reference edges (constants, interface-type usage) → out of scope (architectural).
