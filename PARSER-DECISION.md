# Parser Strategy Decision (D-05 / D-05a)

**Status:** Spike complete, numbers measured. **Pending human ratification** at Plan 01-07 Task 4.

**Decision: Option A — CGo tree-sitter (`tree-sitter/go-tree-sitter` + per-language grammar modules).**

This follows D-05a's default: CGo is selected unless the wazero (Option B) arm's
parse-time overhead proves invisible against the full indexing pipeline AND its
grammar-to-WASM pipeline cost proves acceptable. Neither condition held —
see below. CGo becomes the single documented CGo exception to the project's
pure-Go / minimal-dependency constraint, feeding **DIST-05**.

## Spike Method

Both arms (`internal/parser/cgo`, `internal/parser/wazero`) implement the
identical `parser.Parser` interface (D-05b: `Parse([]byte, *Tree) (*Tree, error)`
+ `Close() error`). `tools/spike/` benchmarks and adversarially tests both
through that one interface — never touching either backend's native types
directly except where explicitly unwrapped for the incremental-reparse
benchmark (see below).

**Corpus (pinned for CI reproducibility, per RESEARCH Open Question 1):**

| Language | Repo | Ref | Commit | Files | Size |
|----------|------|-----|--------|-------|------|
| Go | `spf13/cobra` | `v1.8.1` | `e94f6d0dd9a5e5738dca6bce03c4b1207ffbc0ec` | 14 files | 193,315 bytes |
| Python | `pallets/flask` | `3.0.3` | `c12a5d874c5a014495eb2db8a73f40037bc813ac` | 18 files | 213,956 bytes |

Committed as static fixtures under `tools/spike/testdata/` (see
`tools/spike/testdata/ATTRIBUTION.md`) — no network access required to
reproduce these numbers.

**Environment:** go1.26.5, darwin/arm64, Apple M4 Max, Apple clang 21
(CGO_ENABLED default on). `zig` and `wasi-sdk` are **not installed** in this
environment (confirmed via `which`), which bounds both the wazero arm's
construction (Task 1 — a prebuilt third-party WASM binary was vendored
instead of building one from source) and the cross-compile measurement below.

## Measured Numbers

All benchmarks run via `go test ./tools/spike/ -bench=. -benchmem -benchtime=1000x -run '^$'`
(a fixed 1000-iteration count rather than the default time-based target — the
wazero arm recreates a full module instance every iteration to bound guest
memory growth per its known ABI limitation, described below, which makes
per-iteration setup cost dominate wall-clock time and makes Go's default
time-based iteration search overshoot badly for that arm specifically).

### Full-parse throughput (entire corpus, one pass per iteration)

| Benchmark | ns/op | Throughput | Relative to CGo |
|-----------|-------|------------|------------------|
| `CGoFullParse_Go` | 11,515,956 | 16.79 MB/s | 1.00x (baseline) |
| `WazeroFullParse_Go` | 20,343,833 | 9.50 MB/s | **1.77x slower** |
| `CGoFullParse_Python` | 12,933,405 | 16.54 MB/s | 1.00x (baseline) |
| `WazeroFullParse_Python` | 22,942,439 | 9.33 MB/s | **1.77x slower** |

### Incremental reparse (single-keystroke append-at-EOF edit on the corpus's largest file)

| Benchmark | ns/op | Throughput | Note |
|-----------|-------|------------|------|
| `CGoIncrementalReparse_Go` | 104,496 | 536.56 MB/s | True `InputEdit`-annotated reparse (RESEARCH Code Examples pattern) |
| `CGoIncrementalReparse_Python` | 198,828 | 302.49 MB/s | True `InputEdit`-annotated reparse |
| `WazeroIncrementalReparse_Go` | 175,395 | 319.67 MB/s | **Weak hint-only reparse** — see caveat below |
| `WazeroIncrementalReparse_Python` | 57,930 | 1038.21 MB/s | **Weak hint-only reparse** — see caveat below |

**Incremental-reparse caveat (important, not glossed over):** the vendored
wazero WASM binary's export table (`internal/parser/wazero/assets/PROVENANCE.md`)
does **not** export `ts_tree_edit`, `ts_tree_get_changed_ranges`, or
`ts_tree_delete`. This means the wazero arm's "incremental reparse" benchmark
above is **not the same operation** as the CGo arm's: it cannot annotate the
edited byte range via `InputEdit` before reparsing, so it can only pass the
previous tree's guest pointer as a bare, unannotated hint to
`ts_parser_parse_string`. The CGo numbers above ARE a true incremental
reparse (edit annotated, `ChangedRanges` computed per RESEARCH's Code
Examples pattern); the wazero numbers are a structurally weaker operation
and are **not directly comparable** to the CGo incremental numbers — in
particular, `WazeroIncrementalReparse_Python`'s very low ns/op (faster than
its own full-parse-per-file average) is suspicious and consistent with the
grammar accepting the unannotated hint as if nothing changed rather than
performing genuine incremental work; this project's limited ABI surface
(only `ts_tree_root_node` / `ts_node_is_error` exported, no node
span/position accessors) does not allow verifying the resulting tree's
correctness beyond "did not error." This is recorded as a spike finding,
not silently smoothed over: **Option B's incremental-reparse story is
currently unverified and likely broken**, which independently weighs
against adopting it, since incremental reparse (not full-parse throughput)
is tree-sitter's headline advantage for an editor/agent-facing indexer.

### Retiring RESEARCH Assumption A1

RESEARCH's Assumptions Log (A1) flagged the informal "~2x slower"
WASM-vs-CGo figure (sourced from a single low-confidence community
benchmark cited in `.claude/CLAUDE.md`) as something that "should not
survive past the spike." It is now retired: **the measured full-parse
overhead on this project's actual corpus is ~1.77x**, close to but measurably
better than the informal figure — Option B's raw full-parse cost is real but
somewhat less severe than assumed. This does not change the decision (see
Decision Rationale below), since the incremental-reparse gap (the more
architecturally important dimension for this project) is far worse than a
uniform ~1.77x, and the pipeline-maturity gap is decisive on its own.

## Static-Build Impact (RESEARCH Pitfall 3)

Measured via `go run ./tools/spike static-build`:

```json
{
  "cgo_static_build_ok": false,
  "cgo_static_build_error": "github.com/tree-sitter/tree-sitter-python/bindings/go: build constraints exclude all Go files ...\ngithub.com/tree-sitter/tree-sitter-go/bindings/go: build constraints exclude all Go files ...",
  "wazero_static_build_ok": true,
  "zig_available": false,
  "cross_compile_note": "zig not found on PATH in this spike environment; cross-compile was NOT executed. The documented approach (per goreleaser/example-zig-cgo) is to set CC/CXX to `zig cc -target <triple>` / `zig c++ -target <triple>` with CGO_ENABLED=1 and GOOS/GOARCH set to the cross target; full validation deferred to Phase 8."
}
```

- **`CGO_ENABLED=0 go build ./internal/parser/cgo/...` fails outright** — the
  CGo grammar bindings' build constraints exclude all Go files when CGo is
  disabled, confirming RESEARCH Pitfall 3: "single static binary for users"
  is NOT the same claim as "no CGo toolchain needed anywhere." Building this
  arm for any target requires a working C toolchain at build time.
- **`CGO_ENABLED=0 go build ./internal/parser/wazero/...` succeeds** — the
  wazero arm is pure Go end-to-end; the WASM binary is embedded data
  (`//go:embed`), not a C build dependency of the host.
- **Cross-compile (e.g. darwin/arm64 → linux/arm64) was NOT executed** — `zig`
  is absent from this environment (RESEARCH Environment Availability
  correctly predicted this). The known-good approach (`zig cc`/`zig c++` as
  `CC`/`CXX`, per `goreleaser/example-zig-cgo`) is documented but unverified
  against tree-sitter's specific external-scanner C sources (RESEARCH
  Assumption A2 flags this as a real, not merely theoretical, risk — SQLite's
  amalgamated single-file C is a different build shape than tree-sitter
  grammars' external scanners). **Full cross-compile validation is deferred
  to Phase 8**, consistent with RESEARCH's own recommendation.

## Crash Isolation (RESEARCH Pitfall 5)

Measured via `go test ./tools/spike/ -run TestCrashIsolation -v`, spawning
each malformed-input parse as an isolated child process (so an uncatchable
CGo-side segfault would show up as an observed signal from the parent,
rather than killing the test binary itself — see
`tools/spike/crashisolation_test.go` file doc for why this design is
necessary).

| Case | cgo/go | cgo/python | wazero/go | wazero/python |
|------|--------|------------|-----------|----------------|
| Truncated real file (cut mid-stream) | SURVIVED | SURVIVED | SURVIVED | SURVIVED |
| Random byte garbage (64 KiB) | SURVIVED | SURVIVED | SURVIVED | SURVIVED |
| Deep nesting (Go: 2,000,000 nested parens, ~4.0 MiB; Python: 2,800-level indent nest, ~3.95 MiB — both sized just under the 4 MiB ceiling) | SURVIVED | SURVIVED | SURVIVED | SURVIVED |
| Oversized (4 MiB + 1 byte) | rejected pre-parse (`ErrSourceTooLarge`) | rejected pre-parse | rejected pre-parse | rejected pre-parse |

**Finding: no crash was observed in either arm, for any case tested, within
this session's spike budget.** Both backends survived truncated input,
random garbage, and pathologically deep nesting pushed to just under the
`parser.MaxSourceBytes` (4 MiB) ceiling that both arms enforce before
invoking their underlying parser (Security Domain V5 / threat T-01-03 — this
ceiling itself was confirmed working identically on both arms via the
"oversized" case). This is an honest negative result, not an "eliminated
risk" claim: it does not prove the CGo arm's uncatchable-segfault tail risk
(T-01-01) is impossible — only that it did not materialize against the
adversarial shapes and sizes tested here, which were bounded by the same
4 MiB ceiling that would bound any real attacker's input in production. A
plausible explanation consistent with this result: tree-sitter's GLR/LR
parsing core is not naively recursive per nesting level the way a
textbook recursive-descent parser would be (it's stack-machine-driven), so
nesting-depth attacks specifically may be a weaker vector against
tree-sitter than against some other parser architectures — this is an
inference from the observed result, not something separately verified
against tree-sitter's internals in this spike.

**Because the size ceiling (already a hard requirement, Security Domain V5)
bounds the practical adversarial input space to ≤4 MiB for BOTH arms, the
crash-isolation dimension currently contributes less differentiating weight
to this decision than the throughput and pipeline-maturity dimensions
below** — the ceiling itself, not the backend choice, is doing most of the
practical risk mitigation here. The narrow interface (D-05b) preserves the
option to move to wazero's isolation guarantee later if this changes (e.g.
if the ceiling is ever raised, or a crash is observed in the wild).

## Decision Rationale (against D-05a's criterion)

D-05a: default to Option A (CGo) for v1 unless the spike shows Option B's
parse-time overhead is invisible against the full indexing pipeline **and**
the grammar-to-WASM compilation cost is acceptable. Both conditions must
hold to justify switching off the default; **neither holds**:

1. **Overhead is not invisible.** Full-parse throughput is ~1.77x slower on
   both languages tested — a real, measurable, consistent gap, not noise.
   The incremental-reparse dimension (arguably more important for an
   editor/agent-facing incremental indexer than raw full-parse throughput)
   is worse than "1.77x slower": Option B literally cannot perform the same
   operation as Option A here (see caveat above), because the vendored WASM
   binary's ABI omits `ts_tree_edit`. This is not an overhead number to
   weigh — it's a missing capability.

2. **Pipeline cost is not acceptable.** Per RESEARCH's "State of the Art" and
   this project's own `.claude/CLAUDE.md`, no mature, ready-made wazero
   distribution of tree-sitter exists. Task 1 obtained a REAL, working
   wazero arm only by vendoring a third-party prebuilt binary
   (`malivvan/tree-sitter`, explicitly pre-release/experimental — 3 commits,
   3 stars, single maintainer) rather than building the
   grammar-to-WASM-via-`zig`/`wasi-sdk` pipeline from source, because neither
   toolchain is installed in this environment. That vendored binary is
   missing `ts_tree_edit`, `ts_tree_get_changed_ranges`, and
   `ts_tree_delete` — i.e., no real incremental reparse and no way to free
   individual parsed trees (unbounded guest-memory growth under sustained
   use, mitigated in the benchmark harness only by recreating the whole
   module instance per iteration). Standing up a production-grade Option B
   would mean either (a) building the missing exports into a real
   grammar-to-WASM pipeline from source — genuine, unbounded-scope
   engineering work RESEARCH already flagged as "this project's own
   engineering work if D-05a selects Option B" — or (b) depending
   indefinitely on an experimental single-maintainer third-party binary for
   a load-bearing production dependency. Neither is "acceptable" pipeline
   cost for v1.

Since Option A remains the only path with full language coverage today
(RESEARCH "State of the Art"/"The Parser Decision"), top-tier measured
performance, a complete incremental-reparse story, and no unresolved ABI
gaps — **Option A (CGo tree-sitter) is selected** for v1.

## DIST-05 Consequence

CGo tree-sitter (`tree-sitter/go-tree-sitter` + `tree-sitter-go` +
`tree-sitter-python`, extending to further per-language grammar modules as
LANG-0x requirements are added) is the **single documented CGo exception**
to this project's pure-Go / minimal-dependency constraint. This must be
carried into Phase 8's release engineering (DIST-05): CI builds require a
working C toolchain per target platform (not just `GOOS`/`GOARCH`), and
cross-compilation needs a validated `zig cc`/`zig c++` setup — the static
release *artifact* remains one binary per platform either way, but the
*build environment* producing it is not CGo-free. Phase 8 must validate the
zig cross-compile path this spike could not (no `zig` installed here) before
shipping multi-platform releases.

## D-05b Note

Both arms were built and benchmarked entirely behind the narrow
`parser.Parser` interface (`internal/parser/parser.go`). Nothing outside
`internal/parser/cgo` or `internal/parser/wazero` depends on either
backend's concrete types. This decision is therefore a backend selection,
not an architecture change — a future revisit (e.g. if a mature Option B
pipeline emerges, or if the crash-isolation risk materializes in
production) remains a bounded swap behind this same interface.

## Disposition of Spike Artifacts

Per this decision, `internal/parser/wazero/` (including its vendored WASM
asset) and `tools/spike/` are spike-scope artifacts, not production code.
They are being **retained in the repository** (not deleted) as the
documented evidence trail for this decision and to keep the benchmark
reproducible for future re-evaluation, per the plan's own artifact-produced
note ("kept only if Option B is selected" — recorded here as an explicit,
deliberate deviation from that default: the benchmark harness and the
losing arm are kept as living documentation rather than deleted, since
`internal/parser/wazero` is clearly package-doc-marked as a spike artifact
and `tools/spike/` is build/test-time-only tooling never shipped in the
production binary). If this is not the desired outcome, deletion is a
follow-up, not a blocker to ratifying the decision itself.
