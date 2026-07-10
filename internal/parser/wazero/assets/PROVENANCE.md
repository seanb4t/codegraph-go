# Vendored WASM Artifact Provenance

**File:** `ts.wasm.gz` (gzip of a 26,308,387-byte WASM module; ~2.4 MiB compressed)

## What this is

A statically-linked, self-contained `wasm32-wasi` build of the tree-sitter C
runtime (`lib.c`, `parser.c`, `lexer.c`, `subtree.c`, ...) plus 22 language
grammars — including Go and Python, the two languages this spike (Plan 01-07,
decision D-05) benchmarks — compiled together into a single WASM module. No
dynamic linking / dlopen step is required to use it: the runtime and every
grammar are already one binary, imported here purely through
`github.com/tetratelabs/wazero`'s WASI support.

## Source

- **Repository:** https://github.com/malivvan/tree-sitter (referenced in
  `.claude/CLAUDE.md` §"The Parser Decision" Option C / research as the
  closest prior art for a wazero tree-sitter binding — explicitly noted
  there as "pre-release/experimental").
- **Commit:** `46b39a70b658161c3b883dfad2a95f9387905f64` (default branch
  `master`, fetched 2026-07-10).
- **Upstream file:** `lib/ts.wasm.br` (brotli-compressed in the upstream
  repo). Re-packed here as gzip (`ts.wasm.gz`) so this project's
  `internal/parser/wazero` package can decompress it with the Go standard
  library's `compress/gzip` alone — no new third-party Go dependency
  (`andybalholm/brotli`, which upstream uses) was added to this project's
  `go.mod` to obtain this spike arm.
- **Embedded tree-sitter core version:** `v0.24.7` (per upstream's
  `treesitter.go` `Version` constant).
- **License:** MIT (upstream `LICENSE`, copyright malivvan, reproduced
  below in full per its terms).

## Why a third-party binary, not a project-built one

RESEARCH (`.planning/phases/01-foundation-storage-schema-parser-strategy/01-RESEARCH.md`
§"State of the Art" / §"Environment Availability") found no mature,
ready-made wazero distribution of tree-sitter, and this execution
environment does not have `zig` or `wasi-sdk` installed to build one from
source (the standard path per RESEARCH Pitfall 3 / Option B discussion).
Per the Plan 01-07 environment notes, a prebuilt/vendorable WASM grammar
artifact that can be obtained quickly and run through wazero to produce a
REAL parse-throughput number is preferred over either (a) fabricating a
number or (b) skipping the wazero arm entirely. This binary was the result
of that search: it is a real, functioning WASM build (verified: its export
table includes `ts_parser_new`, `ts_parser_set_language`,
`ts_parser_parse_string`, `ts_parser_delete`, `malloc`/`free`, and
`tree_sitter_go` / `tree_sitter_python` language constructors — confirmed
via `wasm-objdump -x`), obtained without needing any missing toolchain.

## Scope and disposition

This is a **spike-only, non-shipped artifact** (Security Domain / threat
T-01-SC in the Plan 01-07 threat model: "accept" disposition, "may be
removed from go.mod if Option A is selected" — the same applies to this
vendored binary). It exists solely to produce real measured numbers for
`PARSER-DECISION.md`. It is not proposed as a production dependency. If
D-05a selects Option A (CGo), this entire `internal/parser/wazero`
directory — including this asset — is expected to be deleted.

## ABI limitations discovered (recorded as spike findings)

The vendored binary's export table is a curated subset of the full
tree-sitter C API — sufficient for full-parse benchmarking and crash/trap
observation, but notably **does not export** `ts_tree_edit`,
`ts_tree_get_changed_ranges`, or `ts_tree_delete`. This means:

1. This arm cannot perform a true `InputEdit`-annotated incremental
   reparse (the RESEARCH Code Examples pattern) — see
   `PARSER-DECISION.md` for how this is handled in the benchmark and what
   it means for the decision.
2. Parsed trees cannot be explicitly freed via the WASM guest's allocator,
   so repeated parses accumulate guest-memory usage across a benchmark
   run. The spike harness sizes its iteration count and the wazero runtime
   memory limit to stay within bounds for the benchmark corpus used, and
   documents this constraint rather than silently absorbing it.

## License (verbatim, MIT)

```
MIT License

Copyright (c) 2025 malivvan

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
