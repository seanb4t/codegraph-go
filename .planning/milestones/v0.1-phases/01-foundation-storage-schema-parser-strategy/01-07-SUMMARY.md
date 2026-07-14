---
phase: 01-foundation-storage-schema-parser-strategy
plan: 07
subsystem: parser
tags: [parser, tree-sitter, cgo, wazero, spike, benchmark, decision, go]

# Dependency graph
requires:
  - phase: 01-foundation-storage-schema-parser-strategy (plan 03)
    provides: internal/parser narrow Parser interface + CGo tree-sitter backend (the benchmark's Option-A arm)
provides:
  - PARSER-DECISION.md — the ratified parser-strategy decision record (D-05/D-05a): Option A (CGo) selected, with measured throughput, static-build, and crash-isolation data
  - "Decision: CGo tree-sitter is the v1 parser and the single documented CGo exception (feeds DIST-05); wazero (Option B) rejected for v1 and deferred"
affects: [02-indexer (parses Go via the CGo backend), 05-languages (adds grammars behind the same CGo Parser interface), 08-release (must handle CGO_ENABLED cross-compile via zig, per this decision)]

# Tech tracking
tech-stack:
  removed:
    - "github.com/tetratelabs/wazero — dropped from go.mod after CGo was ratified; the wazero spike arm and benchmark harness were deleted"
  measured:
    - "CGo full-parse throughput: 16.79 MB/s (Go), 16.54 MB/s (Python)"
    - "wazero full-parse throughput: 9.50 MB/s (Go), 9.33 MB/s (Python) — ~1.77x slower"
    - "CGO_ENABLED=0 breaks the CGo arm build (confirmed); wazero arm built pure-Go"

# Requirements
requirements: [DIST-05]
success_criteria_addressed: ["SC3: parser strategy selected from a head-to-head spike with throughput + static-build impact documented"]
---

# Plan 01-07 Summary: Parser Spike & Decision (CGo vs wazero)

## What was built

A head-to-head parser spike comparing the two viable tree-sitter backends behind the
narrow `parser.Parser` interface (D-05b), and a ratified decision record.

1. **wazero (Option B) arm** — implemented `parser.Parser` for Go + Python by running a
   prebuilt, statically-linked tree-sitter WASM binary (MIT, from the experimental
   `malivvan/tree-sitter` prior art the research named) on `github.com/tetratelabs/wazero`,
   because `zig`/`wasi-sdk` are not installed here. Provenance and a real ABI limitation
   (`ts_tree_edit`/`ts_tree_get_changed_ranges`/`ts_tree_delete` not exported → no true
   incremental reparse) were documented. *(Committed at `ef54627`, then removed — see below.)*
2. **Benchmark harness** (`tools/spike/`) — real `go test -bench` throughput + incremental
   reparse measurements for both arms on pinned Go (`spf13/cobra@v1.8.1`) and Python
   (`pallets/flask@3.0.3`) fixture corpora, plus a crash-isolation test feeding
   truncated/garbage/deeply-nested adversarial input to both backends. *(Committed at
   `e5da8e7`, then removed.)*
3. **`PARSER-DECISION.md`** — the durable decision record: measured numbers, the
   `CGO_ENABLED=0` static-build result, crash-isolation analysis, and the decision against
   the D-05a criterion.

## Measured results

| Dimension | CGo (Option A) | wazero (Option B) |
|---|---|---|
| Full-parse throughput (Go / Python) | 16.79 / 16.54 MB/s | 9.50 / 9.33 MB/s (~1.77x slower) |
| Incremental reparse | true `InputEdit`-annotated (104–199 µs) | hint-only (missing `ts_tree_edit` ABI) |
| `CGO_ENABLED=0` static build | breaks (needs C toolchain) | builds pure-Go |
| Crash isolation (≤4 MiB adversarial input) | no crash observed | no crash observed (honest negative) |
| Pipeline maturity | production grammars | experimental prebuilt WASM, incomplete ABI |

## Decision (human-ratified 2026-07-10)

**Option A — CGo tree-sitter — selected for v1.** Neither of D-05a's conditions for adopting
Option B held (parse overhead is not invisible; the WASM pipeline is not production-ready).
CGo is the single documented CGo exception feeding DIST-05; static cross-compilation is
handled later (Phase 8) via `zig cc`. The narrow `parser.Parser` interface is retained so a
future wazero re-evaluation remains a backend swap (D-05b).

## Decisions & deviations

- **Spike artifacts deleted (user-ratified disposition).** After ratification, the wazero arm
  (`internal/parser/wazero/` + vendored WASM) and the benchmark harness (`tools/spike/`) were
  removed and `wazero` dropped from `go.mod`, to keep the dependency tree minimal/audited (a
  core project value). `PARSER-DECISION.md` preserves the numbers as the record. *(Removal
  committed at `8cf8d51`.)*
- **zig unavailable** in this environment, so the wazero arm used a prebuilt WASM rather than a
  from-source `zig cc` build; the from-source WASM pipeline is deferred (Phase 8 / future
  Option-B revisit).
- **Signing interruptions:** 1Password's SSH-signing agent auto-locked several times during the
  long spike runs; commits were driven from the orchestrator after unlock. No `--no-verify` or
  signing bypass was used.

## Self-Check: PASSED

- `CGO_ENABLED=1 go build ./...` — clean
- `CGO_ENABLED=1 go test ./... -race -count=1` — all 5 packages ok
- `wazero` absent from `go.mod`/`go.sum`; no `.go` file imports it
- `PARSER-DECISION.md` present with measured data and the ratified decision
