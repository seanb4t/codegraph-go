# Phase 4: Attribution & Documentation Sweep - Context

**Gathered:** 2026-08-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Sweep the project's own documentation so a reader finds the origin acknowledged exactly once — legally, and in the past tense — and encounters no comparison framing anywhere else.

**In scope:** ATTR-01 (NOTICE trimmed to MIT copyright transcription + one sentence of origin), ATTR-02 (README's `## Relationship to the original` gone; only origin mention is one past-tense clause in `## License` linking to NOTICE), ATTR-03 (LICENSE stays verbatim MIT, `gh api .../license` returns MIT, verified live), DOCS-01 (no comparison framing in README/CONTRIBUTING/SECURITY/CODE_OF_CONDUCT/PARSER-DECISION), DOCS-02 (delete `docs/FLAG-PARITY.md` + `internal/cli/flag_parity_test.go` and all references), DOCS-03 (`docs/LANGUAGE-CAPABILITY-MATRIX.md` states capability on its own terms), DOCS-04 (remaining docs/* carry no retired framing).

**Depends on:** Nothing — independent of the fixture work (Phases 1–3). PROC-01…03 (Phase 5) carries no fixture dependency and may start earlier, but Phase 5's in-tree sweep waits on Phase 3.

**Does NOT touch:** the golden harness, the corpus lock, the benchmark path (`docs/BENCHMARKS.md` is Phase 6), `docs/CLI-REFERENCE.md` (DOCS-05, deferred), `tsextract`, `codegraph migrate` (real product surface).
</domain>

<decisions>
## Implementation Decisions

### Retained attribution (ATTR-01, ATTR-02)

- **D-01:** The retained attribution is **minimal**. `NOTICE` keeps the verbatim MIT copyright transcription for CodeGraph, plus **one sentence of origin**: "CodeGraph Go began as a ground-up Go rewrite of CodeGraph (MIT)." No rationale, no comparison, no "originally based on / ported from / drop-in / ported-heuristics / flag-parity" argument — that is removed. — **Reversibility:** reversible — a NOTICE edit.
- **D-02:** `README`'s only origin mention is **one past-tense clause inside `## License`** that links to `NOTICE` — "This project began as a Go rewrite of CodeGraph; see NOTICE for the original copyright." The `## Relationship to the original` section is removed entirely. — **Reversibility:** reversible — a README edit.

### DOCS-02 deletion (FLAG-PARITY + guard)

- **D-03:** Delete `docs/FLAG-PARITY.md` and `internal/cli/flag_parity_test.go`, and **sweep every reference to either** — CI steps, Taskfile targets, other tests, docs that link to `docs/FLAG-PARITY.md`. `go test ./...` must still pass and nothing in the tree may reference either. The deletion removes a live drift guard; the replacement (`DOCS-05`, a self-authored CLI reference with its own guard) is deliberately deferred per the ROADMAP note. **This is a knowing, recorded reduction in flag-documentation coverage, not an oversight** — recorded in the plan. — **Reversibility:** costly — recreates no coverage; DOCS-05 must be authored to restore it.

### Comparison-framing vocabulary (DOCS-01, DOCS-03, DOCS-04)

- **D-04:** The comparison-vocabulary set is **removed term-by-term with recorded reasons, never by regex**: "parity", "the original", "upstream", "drop-in", "head-to-head", "vs TS", "based on", "ported from". — **Reversibility:** reversible — doc edits.
- **D-05:** The following are **product truth, kept**: `tsextract` (a real package name for TypeScript-the-indexed-language), `codegraph migrate` (a real capability), TypeScript/JavaScript as indexed languages in the capability matrix, and the `began as a Go rewrite` past-tense clause in `NOTICE`/`## License` (D-01/D-02). Removing these would remove capability, which the milestone explicitly forbids ("the sweep removes framing, never capability"). Each borderline reference is resolved individually with a recorded reason.

### Claude's Discretion

- The exact sentence structure of the one-sentence origin clause, provided it is minimal, past-tense, and carries no comparison argument (D-01/D-02 record the content rule).
- The exact ordering of the docs/* edits within the sweep (README, CONTRIBUTING, SECURITY, CODE_OF_CONDUCT, PARSER-DECISION, the remaining docs/*).
- Whether a borderline reference (e.g. a historical note in a doc body) is framed as past-tense-origin (keep, rewrite to past tense) or comparison (remove), recorded per instance with a reason.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### The attribution surfaces
- `NOTICE` — the file ATTR-01 trims (read its current full content first)
- `README.md` — `## Relationship to the original` (removed) and `## License` (the one clause added)
- `LICENSE` — verbatim MIT, untouched; verified live via `gh api repos/seanb4t/codegraph-go/license`
- `docs/FLAG-PARITY.md` + `internal/cli/flag_parity_test.go` — the deletion targets (DOCS-02)

### The doc sweep surfaces
- `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `PARSER-DECISION.md` — DOCS-01
- `docs/LANGUAGE-CAPABILITY-MATRIX.md` — DOCS-03 (states capability on its own terms; note Phase 2 already touched this file for the fail-loud posture)
- `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`, `docs/MCP-2026-07-28-SCOPING.md`, `docs/MCP-8-AGENT-AUDIT.md` — DOCS-04

### What stays out of scope
- `docs/BENCHMARKS.md` — Phase 6 (coupled to the comparison runner removal)
- `docs/CLI-REFERENCE.md` — DOCS-05, deferred
- `tools/bench/realcorpus`, `bench.yml`, `tools/bench/headtohead-*.json` — Phase 6
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- The Phase 2 edits to `docs/LANGUAGE-CAPABILITY-MATRIX.md` (fail-loud posture) — this phase re-edits the same file for DOCS-03; read its current state first.
- The `internal/cli/flag_parity_test.go` guard — its deletion must not break `TestWorkflowRunBodiesInvokeTask` or the capability matrix's references.

### Established Patterns
- **One named cause per diff** — the attribution edits, the FLAG-PARITY deletion, and the doc sweep are separate reviewed concerns; don't bundle them into one undifferentiated sweep.
- **Remove framing, never capability** — `tsextract`, `codegraph migrate`, TypeScript-as-indexed-language are product surface.

### Integration Points
- The FLAG-PARITY deletion touches CI steps / Taskfile targets / other tests that reference it.
- The capability matrix (docs + code) is touched by both DOCS-03 and the Phase 2 fail-loud posture — coordinate.
</code_context>

<specifics>
## Specific Ideas

- The retained origin sentence is the only origin text anywhere in the repo — one sentence in NOTICE, one clause in README's License. Everything else with origin/comparison framing goes.
- The sweep is term-by-term with recorded reasons, so a reviewer can see why each borderline reference was kept or removed.
- `LICENSE` is never edited; its MIT detection is verified live after the NOTICE change, not assumed.

</specifics>

<deferred>
## Deferred Ideas

- `docs/CLI-REFERENCE.md` (DOCS-05) — the self-authored replacement for FLAG-PARITY, with its own drift guard. A later milestone authors it; this one only deletes.
- `docs/BENCHMARKS.md` rewrite — Phase 6 (coupled to the comparison runner removal).
- Any rewording of `tools/bench/realcorpus` package docs (the PERF-01 head-to-head framing) — Phase 6.

</deferred>

---

*Phase: 04-attribution-documentation-sweep*
*Context gathered: 2026-08-15*
