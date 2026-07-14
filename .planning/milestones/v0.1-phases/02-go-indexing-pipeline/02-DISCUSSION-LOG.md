# Phase 2: Go Indexing Pipeline - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-07-10
**Phase:** 2-Go Indexing Pipeline
**Mode:** `--auto` (fully autonomous; all gray areas auto-selected to recommended defaults, no interactive prompts)
**Areas discussed:** CLI lifecycle & commands, Node identity, Schema field parity, Two-pass architecture, Edge multiplicity, Go extraction scope

---

## CLI Lifecycle & Commands (INDX-01, INDX-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Cobra `init`/`index`/`uninit`; `init`=create+full-index; existing-dir errors | Parity surface, safe idempotency, guidance on re-init | ✓ |
| `init` silently rebuilds an existing `.codegraph/` | Fewer prompts but risks clobbering a good index | |
| Single `index` command, no `init` | Simpler surface but breaks INDX-01 "one step" + TS parity | |

**Choice (recommended default):** Cobra root with `init`/`index`/`uninit`; `init` creates `.codegraph/` and runs a full index; re-`init` errors with guidance; `--force`/`--quiet`/`--verbose`; deterministic from-scratch rebuild.
**Notes:** Determinism promoted to a first-class requirement — Phase 3 golden-diff and Phase 8 reproducibility depend on it.

---

## Node Identity

| Option | Description | Selected |
|--------|-------------|----------|
| TS-parity `<kind>:<hash>` stable content hash (SHA-256-derived) | Parity + clean Phase 7 migration + stable across re-index for resolution/sync | ✓ |
| Positional `file:line` ids | Simple but unstable across edits; churns Phase 4 sync; no clean TS mapping | |
| Sequential/auto-increment ids | Non-deterministic across runs; breaks reproducibility + migration | |

**Choice (recommended default):** `<kind>:<hash>` where hash is a deterministic SHA-256-derived hex over (kind + qualified_name + file_path).
**Notes:** Load-bearing beyond Phase 2 — Phases 3/4/7 all bind to this id scheme. Hash strengthened from TS's 128-bit sample to SHA-256 per Security Domain V6; id *shape* stays parity-compatible.

---

## Schema Field Parity

| Option | Description | Selected |
|--------|-------------|----------|
| Additively extend Node (signature/docstring/visibility/is_exported/return_type) + Edge (provenance/metadata); SchemaVersion stays 1 | Carries parity data now; honors additive-only D-02a; no format break | ✓ |
| Add all TS fields incl. is_async/decorators/type_parameters now | Fields Go can't populate; premature, better added when Phase-5 languages need them | |
| Keep Phase-1 minimal schema, no extension | Loses parity fields the golden corpus shows (signature etc.) | |

**Choice (recommended default):** Additive extension with only the Go-applicable parity fields; new field numbers below reserved 50–59; regenerate graph.pb.go; no SchemaVersion bump.
**Notes:** `provenance` added now but Phase 2 emits only ground-truth (`ast`) edges — `heuristic` tagging is Phase 5.

---

## Two-Pass Architecture (RES-01)

| Option | Description | Selected |
|--------|-------------|----------|
| Parallel extract (NumCPU pool) → sequential resolve owns the single writer; in-memory unresolved refs | Matches roadmap mandate + GraphStore single-writer model | ✓ |
| Fully sequential single pass | Simpler but forfeits parallelism and the mandated two-pass shape | |
| On-disk unresolved_refs staging like TS | Unneeded at Phase-2 scale; adds write churn; revisit only at monorepo scale | |

**Choice (recommended default):** Two-pass parallel-extract → sequential-resolve; batched IndexedBatch writes (never per-symbol); unresolved refs held in memory.
**Notes:** Extract workers each hold their own `Parser` (tree-sitter parsers are not goroutine-safe).

---

## Edge Multiplicity (Phase-1-deferred key-identity question)

| Option | Description | Selected |
|--------|-------------|----------|
| Keep Phase-1 collapse: one edge per (src,kind,dst), record carries representative line/col | Matches keys.go dedup design; sufficient for callers/callees/impact | ✓ |
| Extend key with line/col for distinct call sites (TS `idx_edges_identity`) | Higher fidelity but diverges from keys.go design; more edges; not needed by Phase-3 queries | |

**Choice (recommended default):** Collapse (dedup); documented intentional divergence from TS line/col-distinct edges; Edge record still carries line/col so the key shape can change later without data loss.
**Notes:** Revisit only if a Phase-3 golden-diff shows the collapse changes an agent-facing output shape.

---

## Go Extraction Scope (LANG-01)

| Option | Description | Selected |
|--------|-------------|----------|
| Nodes file/function/method/type/interface/constant/variable; edges contains/imports/calls/embedding; NO interface→impl synthesis | Parity vocabulary from golden corpus; keeps Phase-5 dispatch synthesis out | ✓ |
| Include synthesized interface→implementation dispatch edges now | Pulls Phase-5 RES-02 heuristic synthesizer forward; out of Phase-2 scope | |

**Choice (recommended default):** Ground-truth Go node/edge vocabulary only; "type inheritance" = concrete embedding + type references; interface→impl dispatch deferred to Phase 5.
**Notes:** Extractor shaped so a second language (Phase 5) is a new extractor behind the same two-pass engine, not a rewrite.

---

## Claude's Discretion

- tree-sitter node-type → codegraph node-kind mapping and tree-walk queries (design against `testdata/golden/`).
- Exact hash input tuple + truncation length for node ids (deterministic + collision-resistant).
- Worker-pool sizing, resolve-pass batch granularity, in-memory intermediate structures.
- `.codegraph/` internal subdirectory naming; whether `init` writes a `.gitignore` hint.
- Whether `field` nodes and `docstring` extraction are full or minimal in Phase 2.

## Deferred Ideas

- Query/`explore`/MCP surface + golden-output diffing — Phase 3.
- `sync`/watcher/rename-delete pruning/daemon — Phase 4.
- Interface→impl dispatch synthesis + `provenance: heuristic` + framework routing — Phase 5.
- Additional languages (Java/C#/Python/TS-JS/mainstream) — Phase 5.
- On-disk `unresolved_refs` + `name_segment_vocab` — TS impl details, revisit at Phase 3/8 if needed.
- 100k-file monorepo scale / peak-RSS bounding — Phase 8.
