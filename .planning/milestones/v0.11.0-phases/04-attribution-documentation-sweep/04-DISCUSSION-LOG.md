# Phase 4: Attribution & Documentation Sweep — Discussion Log

**Gathered:** 2026-08-15
**Mode:** autonomous `--interactive` (discuss inline, user answers all questions)

---

## Phase Boundary (from ROADMAP)

Sweep the project's own documentation so the origin is acknowledged exactly once, legally, in the past tense, with comparison framing gone everywhere else. Requirements: ATTR-01/02/03, DOCS-01/02/03/04. Independent of Phases 1–3.

## Areas selected for discussion

All three gray areas (user-selected all):

1. Retained attribution wording
2. DOCS-02 deletion blast radius
3. Comparison-framing vocabulary

## Decisions by area

### 1. Retained attribution → Minimal (name, licence, "began as a Go rewrite")
**Options:** Minimal / Minimal + upstream link / Short origin paragraph
**Selected:** Minimal.
NOTICE keeps the verbatim MIT copyright transcription + one sentence: "CodeGraph Go began as a ground-up Go rewrite of CodeGraph (MIT)." README keeps one past-tense clause in `## License` linking to NOTICE ("This project began as a Go rewrite of CodeGraph; see NOTICE for the original copyright."). No rationale, no comparison, no drop-in/ported-heuristics/flag-parity argument.

### 2. DOCS-02 deletion → Delete both + all references, note the gap
**Options:** Delete both + all refs / Delete both + harness refs only / Delete files, keep a stub test
**Selected:** Delete both + all references, note the gap.
Delete `docs/FLAG-PARITY.md` and `internal/cli/flag_parity_test.go`, sweep every reference (CI steps, Taskfile targets, other tests, docs linking to FLAG-PARITY.md). `go test ./...` passes, nothing references either. DOCS-05 (the self-authored replacement) is deliberately deferred — a knowing, recorded coverage reduction.

### 3. Comparison-framing vocabulary → Remove 'the original/parity/upstream'; keep capability names
**Options:** Remove framing, keep capability / Broader sweep incl. 'rewrite' / Narrower, keep historical notes
**Selected:** Remove the comparison-vocabulary set, keep capability names.
Removed term-by-term with recorded reasons (never by regex): "parity", "the original", "upstream", "drop-in", "head-to-head", "vs TS", "based on", "ported from". KEPT as product truth: `tsextract`, `codegraph migrate`, TypeScript/JavaScript as indexed languages, and the `began as a Go rewrite` past-tense clause in NOTICE/`## License`.

## Claude's Discretion (recorded)
- Exact sentence structure of the one-sentence origin clause
- Ordering of the docs/* edits
- Borderline-reference resolution (past-tense-origin vs comparison), recorded per instance

## Deferred
- `docs/CLI-REFERENCE.md` (DOCS-05) → later milestone
- `docs/BENCHMARKS.md` rewrite → Phase 6
- `tools/bench/realcorpus` package-doc rewording → Phase 6

---

*All decisions captured in 04-CONTEXT.md (D-01 through D-05).*
