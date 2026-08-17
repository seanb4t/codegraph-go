# Phase 6: Benchmark De-coupling & Memory Sweep - Research

**Researched:** 2026-08-16
**Domain:** In-repo CI/benchmark-harness de-coupling (Go + GitHub Actions YAML) + external durable-memory-store sweep (engram MCP)
**Confidence:** HIGH for the in-repo surface (every claim grounded by direct `Read` of the actual source); LOW for the engram tool-call mechanics (no `mcp__engram__*` tools are registered in this research session — see Environment Availability)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** The six committed `tools/bench/headtohead-*.json` captures are **deleted**, not archived. Git history preserves them (mirrors the Phase 5 `codegraph migrate` removal precedent). Reversible.
- **D-02:** `docs/BENCHMARKS.md` is **rewritten from scratch**, not surgically edited. Its current title is "Benchmarks: codegraph-go vs TS CodeGraph" and ~80 of its 300 lines are comparison content. **Consequence the plan MUST resolve:** the document's current absolute figures live *inside* the comparison tables — a from-scratch rewrite depends on a fresh absolute-only measurement run producing numbers to publish. This is a plan-level task dependency, not an editing task. Costly to reverse (methodology rationale must be re-authored, not copied).
- **D-03:** `tools/bench/BASELINE.md` is **in scope** — sweep comparison framing while keeping the investigation findings (guard rationale for `CheckRegression`'s platform/runner-class/`scratch_fs` refusals) intact; it is load-bearing for BENCH-02, not an archive. Reversible.
- **D-04:** The in-tree comparison-framing sweep is proven complete by a **positive-controlled multiline census**: `rg -U` with a bounded pattern set, positive-controlled by planting a known phrase and observing it caught *before* trusting any zero result, counted with `rg -o | wc -l` (never `rg -c`).
- **D-05:** The `headtohead` job in `.github/workflows/bench.yml` (line 96) is **deleted entirely**; a fresh absolute-numbers publishing job is authored in its place — not renamed-and-stripped. Costly to reverse.
- **D-06:** The fresh publisher **MUST carry forward all four** properties (checkable list): (1) runner pinning + `CODEGRAPH_BENCH_RUNNER` env contract; (2) the D-13/D-01 no-Taskfile exception (inline `go run` invocations, never behind a `task bench:*` target, so `-rebless` stays unreachable by tab-completion); (3) non-blocking publish-not-gate contract (a slower number must never fail CI); (4) artifact upload with `if-no-files-found: error` plus the `GITHUB_STEP_SUMMARY` block.
- **D-07:** The runner's `-mode headtohead` (`tools/bench/runner/main.go:161`) is **replaced by a new single-subject `publish` mode** emitting absolute per-repo `Metrics` JSON for the Go binary only. The `headtohead` case, the two-subject measurement loop, and the `-ts-binary` flag are deleted. Costly to reverse (emitted JSON shape changes).
- **D-08:** BENCH-02's "the gate still fires" proof **reuses Phase 3's FIXT-07 protocol verbatim**: pre-mutation `git diff --quiet` gate, confirm the mutation actually applied, observe RED, revert (EXIT-trap where a rename/move is involved), byte-clean revert verified, all recorded in a committed `06-MUTATION-LOG.md`.
- **D-09:** `tools/bench/realcorpus`'s `colbymchenry-codegraph` entry (and its `SiblingDir: codegraph-ts`) is **dropped**, leaving `weft-go` and `cockroachdb-pebble`. Reversible.
- **D-10:** The prose rewrite reaches **realcorpus plus both `internal/corpora` cross-references**: `realcorpus`'s package doc and field comments re-authored on absolute-measurement terms; `internal/corpora`'s package-doc paragraph **and** its `Manifest.Note` text updated to match — the latter matters because it is a committed *data* field shipping inside `corpora/manifest.json`. Reversible.
- **D-11:** `cockroachdb-pebble` (BSD-3-Clause) is **kept**; the license-policy difference between the two manifests is **re-justified on its own terms** (fetch-only, never-vendored measurement corpus vs. FIXT-01's golden-fixture bar), citing no head-to-head benchmark as the reason. **Standing constraint (do not re-litigate):** the two-manifest separation itself is a deliberate Phase 1 decision (`01-04-PLAN.md`, `internal/corpora/manifest.go:6-15`) — Phase 6 touches the text, never the split.
- **D-12:** The memory sweep covers **the engram spine plus the agent-facing framing files**: `.claude/CLAUDE.md`, `.planning/PROJECT.md`, `.planning/STATE.md`. Reversible.
- **D-13:** The supersede test is **present-tense OR forward-looking**. Records describing what was decided/done at a point in time are true and **must survive untouched**. One-way: superseding is an append to durable history.
- **D-14:** The sweep covers **spine + `rule:repo:` scope + workspace overlay scopes** — every scope a fresh session recalls.
- **D-15:** Completeness is evidenced by a **committed `06-MEMORY-SWEEP.md`** enumerating every record by `short_id` with a per-record verdict (supersede / leave-historical) and reason. Post-sweep re-query output is **not** required (accepted trade-off, recorded deliberately).
- **D-16 (precondition):** Engram MCP endpoint reachability is an explicit precondition — **fail loud** if the store cannot be enumerated; never record a sweep as complete against a store that could not be read. (Was unreachable — `getaddrinfo ENOTFOUND mcp-gw.fzymgc.house` — at context-gathering time; see Environment Availability for this session's re-check.)

### Claude's Discretion

None — every area presented was decided by the user. No "you decide" options were selected.

### Deferred Ideas (OUT OF SCOPE)

- **Replacement third benchmark corpus entry.** Dropping `colbymchenry-codegraph` narrows the corpus from three repos to two; whether published absolute numbers want broader language/size coverage is a future-phase measurement-quality question.
- **Post-sweep re-query verification for the memory sweep.** Explicitly declined (D-15).
- **Renaming the `realcorpus` package itself.** D-10 rewrites its prose but keeps the package name.
- Two reviewed-and-not-folded todos (post-release-verify guard test, golangci-lint) — out of scope, unrelated to the benchmark surface.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| BENCH-01 | `docs/BENCHMARKS.md` publishes absolute throughput/query-latency/peak-RSS figures with methodology, no head-to-head table | See "Absolute-only benchmark methodology" pattern below; exact section-by-section fate of the current 300-line doc is mapped in Architecture Patterns → Pattern 1 |
| BENCH-02 | Comparison runner removed from `tools/bench`; `internal/bench.CheckRegression` + committed `baseline.json` still fires on a real regression, demonstrated RED | See "The FIXT-07 mutation protocol, applied to CheckRegression" — mutation target identified, protocol steps reused verbatim from `03-MUTATION-LOG.md` |
| BENCH-03 | `bench.yml` runs and publishes absolute numbers without invoking another implementation | See Pattern 3 — exact deletions and the four D-06 properties mapped line-by-line against the current `headtohead` job |
| MEM-01 | Every engram spine record asserting retired framing is superseded, not overwritten/deleted | See "engram supersede + scope enumeration" — tool surface, reachability re-check, and the fail-loud precondition |
| MEM-02 | A session started after the sweep recalls no port/parity framing in the present tense | Same as above; exact target files and line-level quotes given in Runtime State Inventory |

</phase_requirements>

## Summary

This phase has two structurally independent halves that happen to share a phase boundary: an **in-repo Go/CI de-coupling** (BENCH-01/02/03) fully groundable by reading the actual source, and an **external durable-memory sweep** (MEM-01/02) that this research session cannot execute or verify — the engram MCP server is not registered in this repository's `.mcp.json`, and no `mcp__engram__*` tools were available to this session. Every claim about the code surface below is `[VERIFIED]` against a specific `file:line`; every claim about the engram tool surface is `[ASSUMED]` and flagged for the plan to treat as a live precondition, per D-16.

The benchmark half is a 6-file surgical job with load-bearing constraints, not a rewrite from a blank slate: `internal/bench.CheckRegression` (`regression.go`) is explicitly **out of scope for behavior change** and already carries 21 passing unit-test cases in `regression_test.go` that exercise every guard (platform, runner, scratch_fs, throughput, RSS, ceiling) — this existing suite is the single cheapest, most direct "fires on a real regression" oracle for BENCH-02's RED demonstration, no CI dispatch required. The runner (`tools/bench/runner/main.go`) currently interleaves head-to-head-only code (`-mode headtohead`, `-ts-binary`, `measureSubject`'s two-subject loop, `resolveTSBinary`) with regression-only code (`-mode regression`, `measureRegressionTrial`, `medianMetrics`) inside one `package main` — D-07's replacement `publish` mode is a straightforward reuse of `measureSubject`'s single-subject path (already parameterized by `binary string`), not a new implementation. `bench.yml`'s `headtohead` job (lines 96-158) already carries all the CI scaffolding (runner pinning, artifact upload, job summary) the fresh publisher needs — D-06's four properties are copy-forward, not new engineering, with the Node/npm install steps and `-ts-binary` flag deleted.

**Primary recommendation:** Sequence the phase as (1) drop the corpus entry and rewrite `realcorpus`/`internal/corpora` prose first (D-09/D-10/D-11 — no dependency on anything else), (2) build the `publish` runner mode and fresh `bench.yml` job second (D-05/D-06/D-07 — depends on (1) since the runner iterates `realcorpus.Corpora()`), (3) run a real measurement and rewrite `docs/BENCHMARKS.md` third (D-02 — depends on (2) existing and having run at least once to produce numbers), (4) prove the regression gate via the existing unit-test suite mutation fourth (D-08 — independent of 1-3, can run any time), and (5) run the memory sweep last, exactly as CONTEXT.md's own accumulated-context section states ("the engram memory sweep is last and is verified by inspection... the memory sweep is most accurate once the documentation and code sweeps have landed").

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Absolute performance measurement (indexing throughput, query latency, peak RSS, cold start) | CLI / Backend tooling (`tools/bench/runner`) | — | Pure Go subprocess harness, no service boundary; runs the built `codegraph` binary and reads OS-level `rusage` |
| Regression gating | CI / Backend tooling (`internal/bench.CheckRegression` invoked from `ci.yml`) | — | Pure-logic comparison function, gated in CI against a committed data file (`baseline.json`); no I/O of its own |
| Published benchmark documentation | Static docs (`docs/BENCHMARKS.md`) | CI (`bench.yml`'s publish job writes the source numbers) | Docs are a downstream artifact of a CI-produced JSON; the doc itself is hand-authored prose around machine-measured numbers |
| Corpus pin authority | Data / Config (`tools/bench/realcorpus/manifest.go`, `internal/corpora/manifest.go` + `corpora/manifest.json`) | — | Two deliberately separate pinned-repo manifests (Phase 1 decision); neither performs measurement itself |
| Durable project memory (framing correction) | External store (engram spine, outside git) | Agent-facing framing files in-repo (`.claude/CLAUDE.md`, `.planning/PROJECT.md`, `.planning/STATE.md`) | The spine is genuinely out-of-repo state (D-16's "no CI gate can hold it"); the framing files are ordinary git-tracked docs that happen to be read at session start |

## Standard Stack

**This phase introduces no new libraries or dependencies.** It is a pure de-coupling/removal phase over the existing Go toolchain, `ripgrep`, GitHub Actions YAML, and the engram MCP tool surface (already provisioned outside this repo's own `.mcp.json` — see Environment Availability). The `npm install -g @colbymchenry/codegraph@1.3.1` step in `bench.yml`'s `headtohead` job (`.github/workflows/bench.yml:122-123`, quote: `run: npm install -g @colbymchenry/codegraph@1.3.1`) `[VERIFIED: .github/workflows/bench.yml:122-123]` and the accompanying `actions/setup-node` step (`:117-120`) are **removed**, not replaced — the fresh publisher measures the Go binary only.

### Core

No new core dependencies. Existing Go standard library only (`encoding/json`, `flag`, `os/exec`, `time`, `syscall`) in the runner and `internal/bench` package, unchanged.

### Supporting

No new supporting dependencies.

### Alternatives Considered

Not applicable — no library selection decision exists in this phase.

**Installation:** No `go install`/`npm install`/`pip install` step is added by this phase. One existing `npm install -g` CI step is deleted.

## Package Legitimacy Audit

**Not applicable.** This phase installs zero new external packages. The only package-manager-relevant change is a **removal**: `bench.yml`'s `npm install -g @colbymchenry/codegraph@1.3.1` step is deleted as part of D-05/D-07 (the comparison binary is no longer invoked). No `Package Legitimacy Gate` protocol run is required — there is nothing to check into the registry.

**Packages removed:** `@colbymchenry/codegraph@1.3.1` (npm, global install in CI only — never a `go.mod`/`package.json` dependency of this repo). **Packages flagged as suspicious:** none.

## Architecture Patterns

### System Architecture Diagram (current state, before Phase 6)

```
                     ┌─────────────────────────────┐
                     │   tools/bench/realcorpus     │
                     │   (weft-go, colbymchenry-    │
                     │    codegraph, pebble)        │
                     └───────────┬──────────────────┘
                                 │ Corpora()
                                 ▼
  bench.yml: headtohead job ──► tools/bench/runner -mode headtohead
  (npm install TS binary,        │
   go build Go binary)           ├─► measureSubject("go", goBinary,  entry) ──┐
                                 │                                            │
                                 └─► measureSubject("ts", tsBinary,  entry) ──┤
                                                                              ▼
                                                              []bench.Metrics JSON
                                                              (both subjects, per repo)
                                                                              │
                                                        ┌─────────────────────┴──────┐
                                                        ▼                            ▼
                                          job summary (head-to-head table)   headtohead-results
                                                                              artifact + committed
                                                                              headtohead-*.json files
                                                                              (feed docs/BENCHMARKS.md's
                                                                               ratio tables, by hand)

  ci.yml: perf-regression job ──► task bench:regression ──► tools/bench/runner -mode regression
                                                                │
                                                                ├─► gencorpus (synthetic, offline)
                                                                ├─► measureRegressionTrial x N trials
                                                                ├─► medianMetrics(trials)
                                                                └─► bench.CheckRegression(baseline.json, current, ceiling)
                                                                        │
                                                                        ├─ PASS → "regression gate passed"
                                                                        └─ FAIL → non-zero exit, job fails
```

### System Architecture Diagram (target state, after Phase 6)

```
                     ┌─────────────────────────────┐
                     │   tools/bench/realcorpus     │
                     │   (weft-go, pebble ONLY —    │
                     │    colbymchenry-codegraph    │
                     │    entry dropped, D-09)      │
                     └───────────┬──────────────────┘
                                 │ Corpora()
                                 ▼
  bench.yml: publish job (fresh, replaces headtohead job) ──►
       tools/bench/runner -mode publish
       (go build Go binary only — NO npm/Node step)
                                 │
                                 └─► measureSubject("go", goBinary, entry)   [only subject]
                                                                              ▼
                                                              []bench.Metrics JSON
                                                              (Go only, per repo)
                                                                              │
                                                        ┌─────────────────────┴──────┐
                                                        ▼                            ▼
                                          job summary (absolute numbers table)  publish-results
                                                                                 artifact
                                                                                 (if-no-files-found: error)
                                                                                       │
                                                                                       ▼
                                                                        docs/BENCHMARKS.md
                                                                        (rewritten from scratch,
                                                                         absolute-only, D-02)

  ci.yml: perf-regression job ──► task bench:regression ──► tools/bench/runner -mode regression
       (UNCHANGED — this path never touched another implementation to begin with)
                                                                │
                                                                └─► bench.CheckRegression(baseline.json, current, ceiling)
                                                                        [UNCHANGED behavior — BENCH-02 proves it,
                                                                         via a mutation rehearsal, never edits it]
```

### Recommended Project Structure

No new directories. Files touched:

```
docs/BENCHMARKS.md                         # rewritten from scratch (D-02)
tools/bench/BASELINE.md                    # framing swept, guard rationale kept (D-03)
tools/bench/runner/main.go                 # -mode headtohead → -mode publish (D-07)
tools/bench/runner/main_test.go            # tests for -ts-binary/resolveTSBinary removed
tools/bench/realcorpus/manifest.go         # colbymchenry-codegraph entry dropped (D-09), prose reauthored (D-10)
tools/bench/realcorpus/manifest_test.go    # test asserting colbymchenry-codegraph entry removed
tools/bench/headtohead-*.json (6 files)    # deleted (D-01)
internal/corpora/manifest.go               # package-doc paragraph + Manifest.Note text reviewed/updated (D-10)
internal/bench/rss.go                      # WINDOWS #16: "cannot be compared fairly against the TS Node process" reworded
internal/bench/regression.go               # UNCHANGED behavior (mutated only transiently for D-08's rehearsal, then reverted)
.github/workflows/bench.yml                # headtohead job deleted, fresh publish job authored (D-05/D-06)
.planning/phases/06-.../06-MUTATION-LOG.md # new — D-08's FIXT-07-style record
.planning/phases/06-.../06-MEMORY-SWEEP.md # new — D-15's enumeration artifact
.claude/CLAUDE.md, .planning/PROJECT.md,
.planning/STATE.md                         # framing corrected (D-12)
```

### Pattern 1: Absolute-only benchmark methodology (BENCH-01, D-02)

**What:** `docs/BENCHMARKS.md`'s 301 lines split cleanly into content that survives largely as-is and content that must be dropped or replaced, verified section-by-section:

| Section (current line range) | Comparison content? | Fate |
|---|---|---|
| Title, intro (`:1-5`) | Yes — title is literally "Benchmarks: codegraph-go vs TS CodeGraph" | Rewritten |
| `## 1. Methodology` → What is measured, median-of-5, Peak RSS is OS-level (`:7-48`) | Partial — the *measurement discipline* (median-of-5, external `getrusage`-based RSS) is absolute-measurement-neutral and reusable; only the closing paragraph justifying OS-level RSS by contrast with "a TS-CodeGraph-on-Node process" (`:40-48`) is comparison framing | Methodology kept, comparison justification reworded to state the design on its own terms (external measurement is the only way to get a comparable-across-runs number, full stop — no second subject needed to justify it) |
| `### Comparison target` (`:50-55`) | Yes, entirely — names the "installed TS `@colbymchenry/codegraph@1.3.1`" as what is measured against | Deleted section |
| `### Pinned real repos (PERF-01)` (`:57-73`) | Partial — the reproducibility discipline (pinned SHA, never branch/tag) is reusable; the `colbymchenry-codegraph` table row must be dropped (D-09) alongside its "why this repo" framing ("The original TS project this port replaces") | Table becomes 2 rows (`weft-go`, `cockroachdb-pebble`), "why this repo" column reworded per-entry on absolute terms |
| `### Regenerating the numbers` (`:75-87`) | Yes — cites `-mode headtohead` and `bench.yml`'s weekly schedule/`workflow_dispatch` running that mode | Rewritten to cite the new `-mode publish` invocation |
| `## 2. Raw numbers` (`:89-241`), including the full head-to-head table, "Go vs TS 1.3.1 — summary", "Run-to-run repeatability", "Superseded: v0.1 provisional darwin/arm64" | Yes — entirely comparison tables and ratio prose (8.1×/12.9×/etc.) | **Requires a fresh absolute-only measurement run** (D-02's "plan-level task dependency, not an editing task") — replaced with a from-scratch table of Go-only numbers (files/s, bytes/s, query latency, peak RSS, cold start) per corpus, no ratios |
| `### The one real, committed number: the synthetic regression baseline` (`:213-240`) | No — already describes `baseline.json` on its own terms | Keep near-verbatim (still the one committed, reproducible number) |
| `## 3. Regression gate (PERF-02, INDX-06)` (`:242-300`) | No — describes `CheckRegression`'s tolerance bands, the offline synthetic corpus, and the re-blessing procedure entirely in absolute terms | Keep near-verbatim; this section is the regression-gate description BENCH-01 explicitly requires the rewritten doc to carry |

**When to use:** This section-by-section audit is the concrete "what changes vs. what survives" map D-02's plan-level dependency implies — the planner does not need to re-derive it.

**Example (query-latency percentile framing):** the current doc reports "median wall-clock time of a single `query <term>` invocation" (`docs/BENCHMARKS.md:16`) — this is already a single-number-per-metric methodology (median, not a percentile distribution), consistent with the rest of the harness's median-of-5/median-of-3 discipline documented in `tools/bench/runner/main.go:56-86`. No new percentile methodology needs to be invented; state the existing median discipline as the project's own choice rather than re-deriving a percentile scheme.

### Pattern 2: The single-subject `publish` runner mode (BENCH-02/03, D-07)

**What:** `tools/bench/runner/main.go`'s `measureSubject` function (`:342-433`) is **already single-subject-shaped** — it takes one `binary string` parameter and measures exactly one subject over one corpus entry. The two-subject behavior lives entirely in the *caller*, `runHeadToHead` (`:306-340`), which loops `[]struct{name, binary}{{"go", cfg.goBinary}, {"ts", cfg.tsBinary}}` (`:321-327`) `[VERIFIED: tools/bench/runner/main.go:321-327]`.

Concretely, D-07 requires:
1. Delete the `case "headtohead":` branch (`main.go:161-162`) and `runHeadToHead` (`:306-340`).
2. Delete the two-subject loop inside what becomes the new function — replace it with a single call to `measureSubject("go", cfg.goBinary, entry, srcDir, scratchRoot, cfg.runner)`, dropping the `subject.name == "ts"` iteration entirely.
3. Delete `-ts-binary` flag registration (`main.go:178`, quote: `fs.StringVar(&cfg.tsBinary, "ts-binary", "", "path to the installed TS codegraph@1.3.1 binary (headtohead mode only); ...")`) `[VERIFIED: tools/bench/runner/main.go:178]`, the `tsBinary` config field (`:129`), and `resolveTSBinary` (`:210-218`) plus its `macOSHomebrewTSBinary` constant (`:109-116`).
4. Delete the `cfg.mode == "headtohead" && cfg.tsBinary == ""` resolution branch (`main.go:156-158`).
5. Rename `"headtohead"` mode string to `"publish"` in the `-mode` flag usage string (`:176`) and the mode switch (`:166`).
6. `measureSubject`'s existing per-run `Metrics.MedianOfTrials: 1` comment (`:421-426`, quote: `"headtohead measures each (repo, subject) pair in exactly one session — it publishes raw numbers and never gates..."`) `[VERIFIED: tools/bench/runner/main.go:421-426]` needs only its opening word reworded (`headtohead` → `publish`); the underlying provenance-recording behavior is unchanged.

**JSON shape compatibility with the gate:** `internal/bench/metrics.go`'s `Metrics` struct (`:8-71`) is a **shared type** between publish mode and regression mode — the same JSON tags (`subject`, `repo`, `goos`, `goarch`, `runner`, `scratch_fs`, `median_of_trials`, `files_per_sec`, `bytes_per_sec`, `query_latency_median_ms`, `peak_rss_bytes`, `cold_start_ms`) `[VERIFIED: internal/bench/metrics.go:8-71]` are emitted by both modes already. `CheckRegression` (`internal/bench/regression.go:36-136`) only ever compares a `baseline Metrics` against a `current Metrics` passed in-process by `runRegression` (`main.go:660-681`) — publish mode's JSON never flows into the gate path at all (they are two independent CLI invocations against two independent code paths: `runHeadToHead`/its replacement vs. `runRegression`). **There is no compatibility risk to manage** — publish mode's output shape can change freely (e.g., drop the now-unused `subject` field's "ts" value) without touching the gate.

**When to use:** Cite this pattern when the plan authors the `publish` mode task — it is a subtraction from existing code, not new code, and the subtraction points are enumerated above with exact line numbers.

**Test-file fallout (found via direct read, not assumed):** `tools/bench/runner/main_test.go` has three test functions that reference the doomed surface and must be removed or rewritten: `TestParseFlags_OverridesApply` includes a `-ts-binary` assertion at `:482-487` `[VERIFIED: tools/bench/runner/main_test.go:482-487]`; `TestResolveTSBinary_FindsOnPath` (`:496-511`) and `TestResolveTSBinary_EmptyWhenNotFound` (`:512-533`) test `resolveTSBinary` directly and must be deleted along with the function. `tools/bench/realcorpus/manifest_test.go` has a test at `:72-77` `[VERIFIED: tools/bench/realcorpus/manifest_test.go:72-77]` (quote: `tscg, ok := byName["colbymchenry-codegraph"]` / `t.Fatal("manifest missing colbymchenry-codegraph entry")`) that asserts the entry D-09 drops exists — this test must be deleted or repointed at a surviving entry, or `go test ./tools/bench/...` will fail red for the wrong reason (a stale test, not a real regression).

### Pattern 3: The fresh `bench.yml` publisher job (BENCH-03, D-05/D-06)

**What:** The current `headtohead` job (`.github/workflows/bench.yml:96-158`) already contains every piece of scaffolding D-06 requires kept; the fresh job is this job with five concrete edits:

| D-06 property | Where it already lives in the `headtohead` job | Action |
|---|---|---|
| (1) Runner pinning + env contract | `runs-on: namespace-profile-linux-amd64-4x8` (`:98`) + `CODEGRAPH_BENCH_RUNNER: namespace-profile-linux-amd64-4x8` (`:100-101`) `[VERIFIED: .github/workflows/bench.yml:98-101]` | Carry forward verbatim — this job is non-gating, so it can stay on the faster Namespace profile per the existing top-of-file comment (`:42-52`) explaining why `headtohead` (now `publish`) never needed to match `rebless`'s runner class |
| (2) No-Taskfile exception | The comment block at `:160-171` documents the exception for `rebless`'s three `go run` invocations; the `headtohead` job's own single `go run ./tools/bench/runner -mode headtohead ...` (`:138-141`) already follows the same discipline (inline, not `task`-wrapped) — though `publish` mode never touches `-rebless` at all, so this property is trivially satisfied, not merely inherited | Note in the fresh job's own comment that `publish` mode has no `-rebless` flag to protect (D-07 removed the two-subject/TS surface entirely, not just `-rebless`), so the D-13/D-01 exception is inherited context, not a live risk for this specific job |
| (3) Non-blocking publish-not-gate | Job header comment: `"# a failure here never fails this job's overall exit beyond reporting, since headtohead mode itself only errors on a genuinely broken measurement, not on a slower number"` (`:129-135`) `[VERIFIED: .github/workflows/bench.yml:129-135]` | Carry forward — `publish` mode inherits `runHeadToHead`'s existing behavior of not gating on slow numbers, only on genuine measurement failure |
| (4) Artifact upload + job summary | `Publish results to job summary` step (`:143-151`) + `Upload raw results artifact` step with `if-no-files-found: error` (`:153-158`) `[VERIFIED: .github/workflows/bench.yml:143-158]` | Carry forward verbatim, renaming `headtohead-results(.json)` → e.g. `publish-results(.json)` |

**Deletions specific to BENCH-03 (no invocation of another implementation anywhere in the job):**
- `Set up Node (for the installed comparison binary)` step (`:117-120`)
- `Install the comparison binary @1.3.1` step (`:122-123`, quote: `run: npm install -g @colbymchenry/codegraph@1.3.1`)
- The `-ts-binary "$(command -v codegraph)"` flag on the `go run` invocation (`:140`)
- Job `name:` string `"benchmark publish (PERF-01, non-blocking)"` still reads fine (no rename strictly required by BENCH-03's wording, but the job's own internal comment naming "the installed comparison binary" and "TS/Node subject" throughout must be swept per D-04's census)

**The `workflow_dispatch.inputs.job` choice list** (`:66-76`, currently `headtohead | rebless | both | cpu-diag | scratch-fs-compare | disk-control-github`) `[VERIFIED: .github/workflows/bench.yml:66-76]` needs its `headtohead` option renamed to match the new job id (e.g. `publish`), and every `if:` conditional referencing `inputs.job == 'headtohead'` (there are two: `:99` and the `rebless`/`both` combined condition doesn't reference it, but `headtohead`'s own `if:` at `:99` does) updated to match.

**Triggers unchanged:** `workflow_dispatch` (with the `job`/`trials` inputs) and the weekly `schedule: cron: "0 6 * * 1"` (`:85-90`) both stay — BENCH-03 does not ask for a trigger change, only for what happens inside the triggered job.

### Pattern 4: The FIXT-07 mutation protocol, applied to `CheckRegression` (BENCH-02, D-08)

**What:** Phase 3's `03-MUTATION-LOG.md` establishes the exact five-step shape for every family: (1) state the exact mutation and file:line, (2) run `git diff --quiet -- <file>` before mutating and confirm exit 0 ("clean"), (3) apply the mutation, run the real test/gate command, paste the observed FAILING output verbatim, (4) revert via `git checkout -- <file>` (or an `EXIT`-trap `mv` for a rename-class mutation), (5) prove `git diff --stat` is empty after revert and re-run the same command to confirm green.

**Identifying the right mutation target for `CheckRegression`:** `internal/bench/regression_test.go` already contains 21 subtests exercising every branch of `CheckRegression` (`:8-523`) `[VERIFIED: internal/bench/regression_test.go:8-523]`, including two subtests that are direct, named "fires on a real regression" oracles:
- `"throughput 11% slower: exceeds band fails"` (`:43-51`, `wantErr: true, errHint: "throughput"`)
- `"peak RSS 16% larger: exceeds band fails"` (`:62-70`, `wantErr: true, errHint: "RSS"`)

The lowest-risk, most directly-targeted mutation is **inside `internal/bench/regression.go`'s comparison itself** — for example, temporarily widening `DefaultThroughputTolerance` (`regression.go:9`, quote: `const DefaultThroughputTolerance = 0.10`) `[VERIFIED: internal/bench/regression.go:9]` from `0.10` to e.g. `1.0`, or flipping the `throughputDelta > DefaultThroughputTolerance` comparison (`:113`) to always-false. Running `go test -count=1 ./internal/bench/... -run TestCheckRegression` after that mutation causes exactly the "throughput 11% slower: exceeds band fails" subtest to go RED (`wantErr: true` but `CheckRegression` returns `nil`), which is the existing suite proving the gate would silently stop catching a real regression if this line were ever broken. This mirrors Phase 3's Family (e) shape exactly: mutate a threshold constant, observe a named existing test go RED naming the exact violated expectation, revert, re-run green — no CI dispatch, no runner-class dependency, fully reproducible on any machine `go test` runs on.

**Why not mutate `tools/bench/baseline.json` instead:** the committed `baseline.json` is explicitly listed in CONTEXT.md's Out of Scope ("Any change to `CheckRegression`'s gate semantics or the committed `baseline.json` values"). A *temporary, reverted* rehearsal mutation to that file (inflating `files_per_sec` so a live measurement trips the tolerance) is not literally forbidden by that clause read narrowly, but it introduces a second risk this repo has already been burned by once: `CheckRegression` refuses any comparison whose `GOOS`/`GOARCH`, `runner`, or `scratch_fs` don't match the baseline (`regression.go:49-103`) — a rehearsal run on a non-`ubuntu-latest`/non-`disk`-scratch machine (this research session's own `darwin` environment, for instance) would fail with a **platform/runner/scratch_fs mismatch error**, not a **throughput regression error**, defeating the purpose of the rehearsal and requiring either a real GitHub Actions dispatch on `ubuntu-latest` or a fabricated local baseline that is itself not "the committed `baseline.json`". The `internal/bench/regression.go` mutation avoids this entirely: it exercises the pure function directly via its existing, already-platform-attributed-and-unattributed-mixed unit tests, with no runner-class dependency.

**When to use:** Recommend the plan target `internal/bench/regression.go`'s tolerance constant or comparison operator as the D-08 mutation, run `go test ./internal/bench/...`, and record the two-existing-named-subtest RED output verbatim in `06-MUTATION-LOG.md`, following `03-MUTATION-LOG.md`'s exact five-step template per mutation.

**Known adjacent gap, do not conflate:** `.planning/STATE.md`'s open-issues list records `#16 CheckRegression still never compares Metrics.Repo (corpus identity — note this touches BENCH-02's surface)` `[VERIFIED: .planning/STATE.md:167]`. This is a pre-existing, separate gap (the gate doesn't verify it's comparing metrics from the *same* corpus identity) — out of scope per D-06/CONTEXT.md's explicit "no change to `CheckRegression`'s gate semantics", but the plan should note it exists rather than let a reviewer assume BENCH-02's mutation rehearsal was meant to close it.

### Pattern 5: The positive-controlled multiline census (D-04), adapted to Phase 6's file surface

**What:** Phase 5's `05-07-PLAN.md`/`05-08-PLAN.md` establish the exact reusable shape (already proven twice in this repo): plant a synthetic positive-control fixture containing the target phrase(s) wrapped across a comment line break, run the bounded pattern set with `rg -U -o` against it and confirm every pattern matches (0 dead patterns), run the SAME patterns without `-U` against the SAME fixture and confirm it finds strictly fewer (proving the multiline instrument actually adds coverage), then run the real census over the declared file scope with `rg -o | wc -l` (never `rg -c`), and assert the scanned-file count exceeds a sane floor (guards against a wrong-cwd or over-broad-exclusion vacuous zero).

**Confirmed hit counts on Phase 6's actual file surface (this session, `rg -U -o -i -e '\bTS\b'`, word-bounded):**

| File | `\bTS\b` count |
|---|---|
| `docs/BENCHMARKS.md` | 21 `[VERIFIED: ran rg against the file this session]` |
| `.github/workflows/bench.yml` (TS/comparison/colbymchenry patterns) | 9 |
| `tools/bench/realcorpus/manifest.go` (TS/head-to-head/headtohead patterns) | 10 |
| `tools/bench/BASELINE.md` | 0 — this file already reads on absolute terms; D-03's "sweep" here is closer to a confirming pass than a rewrite |

`internal/bench/rss.go`'s doc comment (`:1-9`) carries the exact phrase WINDOWS.md entry #16 already flagged: `"...which cannot be compared fairly against the TS Node process (D-05)"` `[VERIFIED: internal/bench/rss.go:1-9]`. `tools/bench/runner/main.go`'s `measureSubject` doc comment (`:342-348`) carries the WINDOWS-#16-flagged phrase `"...the Go binary's Pebble store and the TS binary's SQLite store never collide"` `[VERIFIED: tools/bench/runner/main.go:342-348]` — but this sentence describes the two-subject isolation rationale, which D-07 makes moot by deleting the TS subject entirely; the comment should be reworded to describe why each `measureSubject` call still gets its own isolated work directory (avoiding collision with a prior run's own `.codegraph/`), not deleted outright, since the isolation-per-invocation behavior survives even with one subject.

**Bounded pattern set to reuse (Phase 5's proven set, scoped to this phase's vocabulary):** word-bounded `\bTS\b`, `\bhead-to-head\b` / `\bheadtohead\b` (only where used as a *comparison* concept — `internal/bench`'s package doc, `runner/main.go`'s doc comment, `bench.yml` job names — not as a surviving Go identifier like a `-mode` flag value if one is kept for the regression path, which it isn't), `\b(matches|matching|mirrors|mirroring)[[:space:]/*]+(the[[:space:]/*]+)?TS\b`, `\bcomparison\b` (only inside `tools/bench/**`, `.github/workflows/bench.yml`, `docs/BENCHMARKS.md` — this word is legitimately used elsewhere in the repo for unrelated things), `\bcolbymchenry\b`, `\bcodegraph-ts\b`. Exclude `internal/indexer/**` (TypeScript-the-indexed-language product surface, per the already-recorded D-02 KEEP class from Phase 5) and any already-recorded RECORDED-KEEP class from `05-04-SUMMARY.md`/`05-05-SUMMARY.md` if the census is run repo-wide rather than scoped to the Phase 6 file list.

**When to use:** Reuse this exact instrument-first-then-trust-the-zero discipline for the phase's own acceptance gate over `docs/BENCHMARKS.md`, `tools/bench/**`, `internal/bench/**`, `internal/corpora/manifest.go`, and `.github/workflows/bench.yml` — do not invent a new census shape.

### Pattern 6: engram supersede workflow (MEM-01/02, D-13/D-14/D-15) — LOW confidence, unverified this session

**What the additional_context names as available tools:** `mcp__engram__list_memory`, `search_memory`, `supersede_memory`, `list_rules`, `store_rule`, `get_memory`, `set_visibility`. **None of these tools were registered in this research session** (this repository's own `.mcp.json` — `/Volumes/Code/github.com/seanb4t/codegraph-go/.mcp.json` — registers only `gsd-workflow` and `gsd-browser`, no `engram` server `[VERIFIED: .mcp.json, full file read this session]`). Everything below about engram's actual call semantics is `[ASSUMED]` from the project's own `engram:curating-memory` skill description (visible in this session's skill listing, not independently verified by invocation) and must be re-confirmed live by whichever session executes the MEM-01/02 plan.

**Assumed workflow shape**, pending live confirmation:
1. Enumerate every record in the repo spine scope (`repo:github.com/seanb4t/codegraph-go`) via `list_memory` with pagination — the additional_context specifically flags a `next_cursor`-style pagination contract to check for.
2. For each record, read its content via `get_memory` (or the content already returned by `list_memory`) and classify: does it assert retired framing (parity/port/upstream/drop-in) as a **present-tense standing fact** or a **forward-looking obligation** (D-13's supersede test), or does it describe a **past decision/event** (D-13's survives-untouched class)?
3. For each record needing correction, call `supersede_memory` with the corrected content — this is asserted (not verified this session) to append a new record and mark the old one superseded rather than deleting or overwriting it, matching D-13's "one-way... append to the store's durable history" description in CONTEXT.md.
4. Repeat across `rule:repo:` scope (via `list_rules`/`store_rule` — rules are MUST-follow injected ground truth per D-14, the highest-stakes scope) and workspace overlay scopes.
5. Record every enumerated record's `short_id`, verdict, and reason in `06-MEMORY-SWEEP.md` (D-15) — the completeness evidence, not a re-query.

**A specific record already named in CONTEXT.md as a precedent:** memory `myywc0y9vm` is cited (`06-CONTEXT.md:147-148`) as the maintainer correction this phase executes for the `.claude/CLAUDE.md`/`.planning/PROJECT.md` Core Value framing sweep — the plan should `get_memory` this specific ID first as a concrete, named starting point rather than beginning from an unscoped `list_memory` enumeration.

**When to use:** Treat this entire pattern as needing live re-verification against the actual `mcp__engram__*` tool signatures at plan-execution time — do not let the plan hard-code parameter names or pagination field names from this research, since none of it was exercised against a live tool call this session.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Proving a guard isn't vacuous | A bespoke "does the gate work" smoke test | The FIXT-07 protocol (Pattern 4) — mutate, observe RED via the pre-existing `regression_test.go` suite, revert, verify byte-clean | Already proven five times in this repo (`03-MUTATION-LOG.md`); reinventing the shape risks missing a step (e.g. the pre-mutation `git diff --quiet` cleanliness gate) that exists because of a specific prior incident |
| Detecting comment-prose framing across a wrapped line | A line-based `rg` census | The `rg -U` + positive-control instrument (Pattern 5) | This repo's own history: two separate line-based censuses (05-04, 05-05) missed 10 and then 34 wrapped occurrences respectively — the failure mode is documented and the fix is a known, working instrument, not a hypothesis |
| OS-level peak-RSS measurement | A new cross-platform memory-sampling library | `bench.PeakRSSBytes` (`internal/bench/rss.go`, unchanged this phase) | Already handles the `darwin`/`linux` unit-normalization difference (`ru_maxrss` is KB on Linux, bytes on Darwin) correctly; this phase touches only its doc comment, never its logic |
| Runner/platform/scratch-fs identity comparison in the gate | A new "is this measurement comparable" check | `CheckRegression`'s existing three category-error guards (`regression.go:49-103`) | Already hardened against exactly the failure mode (`tools/bench/BASELINE.md`'s full incident history) this phase's own mutation rehearsal exercises; BENCH-02 explicitly forbids touching this logic |

**Key insight:** every mechanism this phase needs (mutation-rehearsal protocol, multiline census, OS-level RSS, cross-runner comparison guards) already exists in this repo, proven against a real incident. The work is subtraction (delete the comparison runner, delete six JSON files, delete a corpus entry) and prose (rewrite two docs, sweep comment framing) — not new engineering.

## Runtime State Inventory

**Trigger for this section:** MEM-01/02 is a framing-correction sweep touching state that lives outside git (the engram spine) — the canonical rename/refactor question applies: *after every file in the repo is updated, what runtime systems still have the old framing cached, stored, or registered?*

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data (durable memory store) | The engram spine for `repo:github.com/seanb4t/codegraph-go`: an unknown number of records, at minimum the one named precedent `myywc0y9vm` (`06-CONTEXT.md:147-148`). **Not enumerated this session** — no engram tool access (see Environment Availability). | Enumerate live via `list_memory` at plan-execution time; supersede records matching D-13's present-tense/forward-looking test |
| Live service config (rules, workspace overlays) | `rule:repo:` scope (injected as MUST-follow ground truth, D-14) and workspace overlay scopes — **not enumerated this session**. | Enumerate live via `list_rules` at plan-execution time |
| OS-registered state | None applicable — this phase touches no OS-level task scheduler, process manager, or service registration. | None |
| Secrets/env vars | None — no secret or env-var name changes in this phase (`CODEGRAPH_BENCH_RUNNER`'s value stays `ubuntu-latest`/`namespace-profile-linux-amd64-4x8` per D-06's runner-pinning carry-forward; the env var name itself is unchanged). | None |
| Build artifacts / installed packages | None applicable to the memory sweep. For the benchmark side: `tools/bench/headtohead-*.json` (6 committed files) are themselves a category of "stored data that must go" per D-01 — these are git-tracked, not build artifacts, but flagged here because they are exactly the kind of state a grep-only audit could miss if searched by filename pattern rather than by content. | `git rm` the six files; git history preserves them (D-01) |
| Agent-facing framing files (in-repo, but session-start-read like runtime state) | `.claude/CLAUDE.md` (opener: `"Originated as a ground-up Go rewrite of [CodeGraph]..."` and Core Value line, `.claude/CLAUDE.md:1-19` `[VERIFIED: .claude/CLAUDE.md:1-19, quote: "Originated as a ground-up Go rewrite of [CodeGraph](https://github.com/colbymchenry/codegraph) (MIT-licensed; ported with attribution)."`); `.planning/PROJECT.md` (identical opener + Core Value at `:5,7,9` `[VERIFIED: .planning/PROJECT.md:1-9]`); `.planning/STATE.md`'s Core Value line (`:26`, quote: `"An agent user can uninstall TS CodeGraph, install the Go binary, migrate their indexes, and everything works the same or better..."`) `[VERIFIED: .planning/STATE.md:26]` — this STATE.md line additionally names a capability (`migrate`) removed in Phase 5 per `CODE-03`. | Reword per D-12; note the STATE.md line is a compound defect (framing + a reference to a retired command) |

**Nothing found in category "OS-registered state" and "Secrets/env vars"** — verified by reading the phase's own declared scope in `06-CONTEXT.md` (`## Phase Boundary` → `In scope`/`Out of scope`) and by direct inspection of the touched files; no `.env`, no task-scheduler registration, no CI secret name appears anywhere in the diff surface this phase declares.

## Common Pitfalls

### Pitfall 1: Trusting a census zero without the positive control
**What goes wrong:** A line-based or unproven-pattern census reports zero framing hits, and the plan marks BENCH-01/BENCH-02/BENCH-03's comment-sweep sub-requirement complete on that basis.
**Why it happens:** This repo's own history — twice (05-04, 05-05) — a green line-based census hid real hits because the phrase wrapped across a Go comment line break.
**How to avoid:** Follow Pattern 5 exactly: prove the instrument first (positive control on a synthetic wrapped fixture), then trust the zero, and paste the full verbatim output (patterns, counts, scanned-file count) into the phase's own gap-closure record — not just a passing exit code.
**Warning signs:** A census command whose acceptance criterion is "exit 0" rather than "the printed count is 0, and the instrument was shown live to catch a planted positive first."

### Pitfall 2: Running the mutation rehearsal on the wrong machine/runner class
**What goes wrong:** A local `go run ./tools/bench/runner -mode regression` invocation against the mutated `baseline.json` (or in genuine CI on a mismatched profile) fails with a **platform/runner/scratch_fs mismatch** error instead of a **throughput/RSS regression** error, and the mutation rehearsal is reported as "RED" when it never actually exercised the tolerance-band logic at all.
**Why it happens:** `CheckRegression`'s category-error guards (`regression.go:49-103`) fire *before* any numeric comparison — a category-error RED looks superficially like a "gate fired" success but proves nothing about BENCH-02's actual claim.
**How to avoid:** Use Pattern 4's recommended target — mutate `internal/bench/regression.go` directly and drive it via the existing `go test ./internal/bench/...` unit suite, which constructs `Metrics` values in-process with matching (or deliberately mismatched, per its own dedicated test cases) platform fields — no real hardware/runner-class dependency at all.
**Warning signs:** The observed failure message contains "platform mismatch", "runner mismatch", or "scratch filesystem mismatch" instead of "throughput regressed" or "peak RSS grew".

### Pitfall 3: Treating `internal/corpora/manifest.go`'s `Manifest.Note` cross-reference as needing a large rewrite
**What goes wrong:** The plan schedules significant work to rewrite `corpora/manifest.json`'s `Note` field for D-10, based on the assumption it mirrors `realcorpus`'s comparison-heavy prose.
**Why it happens:** D-10's wording ("internal/corpora's package-doc paragraph AND its Manifest.Note text are updated to match") could be read as implying equally heavy edits on both sides.
**How to avoid:** This session confirmed by direct read that `corpora/manifest.json`'s `Note` field (`corpora/manifest.json:2`) already reads entirely on its own terms — it names `tools/bench/realcorpus/manifest.go` as "this repository's OTHER pinned-corpus manifest" with no TS/comparison language at all. The actual D-10 work on this specific field is a **verification pass** (confirm it stays accurate once `realcorpus`'s prose changes, since the field cross-references that package by name and location, both unchanged), not a rewrite.
**Warning signs:** A task estimate for `corpora/manifest.json` that assumes multi-paragraph prose changes rather than a one-sentence confirmation.

## Code Examples

### `publish` mode replacing `headtohead` mode (skeleton, D-07)

```go
// Source: tools/bench/runner/main.go, adapted from the existing
// runHeadToHead (:306-340) and measureSubject (:342-433), both read this
// session. This is the SUBTRACTION shape, not new code.

switch cfg.mode {
case "publish": // was "headtohead"
    return runPublish(cfg)
case "regression":
    return runRegression(cfg) // UNCHANGED
default:
    return fmt.Errorf("unknown -mode %q (want publish or regression)", cfg.mode)
}

func runPublish(cfg config) error {
    scratchRoot, err := os.MkdirTemp("", "codegraph-bench-publish-")
    if err != nil {
        return fmt.Errorf("create scratch root: %w", err)
    }
    defer os.RemoveAll(scratchRoot)

    var results []bench.Metrics
    for _, entry := range realcorpus.Corpora() { // now 2 entries, D-09
        srcDir, err := resolveOrClone(entry, scratchRoot) // UNCHANGED
        if err != nil {
            fmt.Fprintf(os.Stderr, "runner: skipping %s: %v\n", entry.Name, err)
            continue
        }
        // ONLY the Go subject — the {"go", "ts"} loop is gone.
        m, err := measureSubject("go", cfg.goBinary, entry, srcDir, scratchRoot, cfg.runner)
        if err != nil {
            fmt.Fprintf(os.Stderr, "runner: %s/go: %v\n", entry.Name, err)
            continue
        }
        results = append(results, m)
    }

    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    return enc.Encode(results)
}
```

### FIXT-07-style mutation rehearsal for `CheckRegression` (D-08 — concrete command sequence)

```bash
# Pattern from .planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md,
# read this session. Every step's shape is reused verbatim.

# 1. Pre-mutation cleanliness gate
git diff --quiet -- internal/bench/regression.go   # must exit 0

# 2. Apply the mutation (example target: widen the throughput tolerance
#    so a real 11%-slower regression no longer trips it)
#    Change: const DefaultThroughputTolerance = 0.10
#    To:     const DefaultThroughputTolerance = 1.0

# 3. Confirm the mutation actually applied
git diff --stat -- internal/bench/regression.go   # must be non-empty

# 4. Observe RED — paste this output verbatim into 06-MUTATION-LOG.md
go test -count=1 ./internal/bench/... -run TestCheckRegression -v
# expect: --- FAIL: TestCheckRegression/throughput_11%_slower:_exceeds_band_fails
#         regression_test.go:511: CheckRegression() = nil, want error

# 5. Revert
git checkout -- internal/bench/regression.go

# 6. Byte-clean proof + green re-run
git diff --stat -- internal/bench/regression.go   # must be empty
go test -count=1 ./internal/bench/... -run TestCheckRegression   # ok
```

### Positive-controlled multiline census, scoped to Phase 6's file surface (D-04)

```bash
# Pattern from .planning/phases/05-process-ci-in-tree-sweep/05-08-PLAN.md,
# read this session (its Task 2 <verify> block is the direct template).

S="[[:space:]/*]+"
F="${TMPDIR:-/tmp}/bench-poscontrol.txt"
printf "%b\n" \
  "// the installed comparison binary (TS" \
  "// codegraph@1.3.1) is invoked here." \
  "// this mirrors TS's own" \
  "// head-to-head harness." > "$F"

# Positive control: multiline MUST exceed line-based, and MUST exceed 0
pc=$(rg -U -o -i -e "\bTS\b" -e "\b(mirrors|matches)${S}TS" "$F" | wc -l | tr -d ' ')
lb=$(rg    -o -i -e "\bTS\b" -e "\b(mirrors|matches)${S}TS" "$F" | wc -l | tr -d ' ')
test "$pc" -gt "$lb" && test "$pc" -gt 0

# Real census over the declared file scope
n=$(rg -U -o -i -e "\bTS\b" -e "\bcolbymchenry\b" -e "\bcodegraph-ts\b" \
  docs/BENCHMARKS.md tools/bench/BASELINE.md tools/bench/runner/main.go \
  tools/bench/realcorpus/manifest.go internal/bench/rss.go \
  internal/corpora/manifest.go .github/workflows/bench.yml \
  | wc -l | tr -d ' ')
echo "TOTAL=$n"   # target: 0
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `docs/BENCHMARKS.md` publishes both subjects' numbers and derived ratios | Absolute-only, single-subject numbers | This phase (BENCH-01) | Ratios (8.1×/12.9×/etc.) are dropped entirely, not reworded — BENCH-01 explicitly forbids "no head-to-head comparison table or multipliers" |
| `tools/bench/runner -mode headtohead` measures Go + TS | `-mode publish` measures Go only | This phase (D-07) | `-ts-binary` flag, `resolveTSBinary`, and the Node/npm CI setup all removed |
| `bench.yml`'s `headtohead` job installs `@colbymchenry/codegraph@1.3.1` via npm | Fresh publish job builds only the Go binary | This phase (D-05/D-06) | One fewer external dependency touched per CI run of this workflow |
| `tools/bench/realcorpus` carries 3 entries (weft-go, colbymchenry-codegraph, cockroachdb-pebble) | 2 entries (weft-go, cockroachdb-pebble) | This phase (D-09) | Corpus narrows; a future phase may need to widen it again (explicitly deferred, not this phase's concern) |
| 6 committed `tools/bench/headtohead-*.json` snapshot files | Deleted | This phase (D-01) | Recoverable from git history; not archived in-tree |

**Deprecated/outdated:** the "PERF-01" requirement ID itself (`docs/BENCHMARKS.md`'s `### Pinned real repos (PERF-01)` heading, `### Regenerating the numbers` referencing it) is a v1.0-era requirement that this phase's rewrite should probably stop citing by that ID in fresh prose — it was satisfied by the now-being-removed head-to-head publication; BENCH-01/02/03 are its replacement for this milestone. (Flagging this as an observation for the plan, not a locked decision — CONTEXT.md does not address whether "PERF-01" as a label should be retired from the rewritten doc.)

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The `mcp__engram__*` tool call signatures (parameter names, pagination field name `next_cursor`, `supersede_memory`'s exact append-not-overwrite behavior) match the project skill's description | Pattern 6 | If the actual tool signatures differ, the MEM-01/02 plan's task descriptions referencing specific parameter names will need live correction at execution time — D-16's fail-loud precondition already requires the plan to verify reachability/enumerability first, so this risk is contained by that gate |
| A2 | Engram MCP endpoint reachability, re-checked this session via `dig`/`curl` against `mcp-gw.fzymgc.house`, reflects the same reachability the plan-execution session will observe | Environment Availability | This session ran on a different host/network context than the eventual execution session; D-16 already requires a live re-check at execution time regardless of this finding |
| A3 | The recommended `internal/bench/regression.go` mutation target (widening `DefaultThroughputTolerance` or flipping the comparison operator) is what the plan will actually choose — CONTEXT.md's D-08 says "reuses the protocol verbatim" but does not name the specific line to mutate | Pattern 4 | Low risk — any mutation to a `CheckRegression` comparison branch that an existing `regression_test.go` subtest already asserts against will produce an equally valid RED demonstration; the specific constant named is a recommendation, not the only correct choice |
| A4 | "PERF-01" as a requirement-ID citation should probably be dropped from the rewritten `docs/BENCHMARKS.md` | State of the Art | Low risk, purely editorial — CONTEXT.md does not address this explicitly, so the plan should treat it as an open call rather than a locked decision |

## Open Questions

1. **Does the fresh `bench.yml` publish job need a new job `name:`/id, or can `headtohead` simply become `publish` in place?**
   - What we know: D-05 says "deleted entirely... a fresh absolute-numbers publishing job is authored in its place — not renamed-and-stripped." The job's YAML `id` (`headtohead:` at `bench.yml:96`) and its `workflow_dispatch.inputs.job` choice value both need to change together for the dispatch UI to make sense.
   - What's unclear: whether "deleted entirely... authored in its place" is a literal git-history instruction (delete the job block, add a new block elsewhere in the file) or a description of the *scope* of change (rewrite the job's body) — the practical diff is nearly identical either way.
   - Recommendation: treat this as a scope description, not a literal git-mechanics instruction; the practical result (old `headtohead:` job id gone, new `publish:` job id present, with the D-06 properties carried forward) satisfies D-05's stated intent either way.

2. **Should the `## 2. Raw numbers` table's replacement be per-corpus (2 rows) or per-corpus-per-metric (wider table)?**
   - What we know: the current head-to-head table format is one row per (repo, subject) pair; with one subject only, the natural replacement is one row per repo.
   - What's unclear: CONTEXT.md doesn't specify a table shape for the absolute-only rewrite.
   - Recommendation: one row per corpus entry (2 rows: weft-go, cockroachdb-pebble), columns = files/s, bytes/s, query latency, peak RSS, cold start — directly mirroring the existing table's column set minus the `Subject` column.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `rg` (ripgrep) | D-04's census instrument | ✓ | (session default) | — |
| `go` toolchain | Building/testing the runner and `internal/bench` | ✓ | per `go.mod` | — |
| `mcp__engram__*` tools | MEM-01/02 (enumerate, supersede, list rules) | ✗ (this research session) — **not registered in this repo's `.mcp.json`** (`/Volumes/Code/github.com/seanb4t/codegraph-go/.mcp.json` lists only `gsd-workflow`, `gsd-browser`) | — | Must be provisioned (likely via a user/global-level MCP config, since it is absent from this project's own config) before the MEM-01/02 plan can execute — D-16's fail-loud precondition applies |
| `mcp-gw.fzymgc.house` network reachability | The engram MCP server's transport | Re-checked this session: `dig +short mcp-gw.fzymgc.house` → `192.168.20.155` (resolves); `curl -m 5 https://mcp-gw.fzymgc.house/` → `HTTP 404` in 0.036s (host reachable, connection accepted, root path just isn't a valid endpoint) | — | The `getaddrinfo ENOTFOUND` failure recorded at CONTEXT.md context-gathering time did **not** reproduce this session — network reachability appears restored, but this is orthogonal to whether the *tool* is registered in the executing session (see row above) |
| `npm`/Node | Currently used by `bench.yml`'s `headtohead` job to install the comparison binary | N/A after this phase | — | This dependency is **removed**, not needed going forward (D-05/D-07) |

**Missing dependencies with no fallback:**
- `mcp__engram__*` tool registration for the session that executes the MEM-01/02 plan. This is the single hard blocker for MEM-01/02 specifically — BENCH-01/02/03 have no dependency on it and can proceed independently.

**Missing dependencies with fallback:**
- None — the one missing dependency (engram tools) has no fallback; D-16 already anticipates this by requiring the plan to fail loud rather than proceed against an unreachable/unregistered store.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go's built-in `testing` package (`go test`) |
| Config file | none — no `pytest.ini`/`jest.config.*` equivalent; Go test discovery is convention-based (`*_test.go`) |
| Quick run command | `go test -count=1 ./internal/bench/... ./tools/bench/...` |
| Full suite command | `go build ./... && go vet ./... && go test -count=1 ./internal/bench/... ./tools/bench/... ./testdata/golden/...` (golden suite included because `docs/BENCHMARKS.md`/`realcorpus` changes must not regress the golden corpus's own use of the `weft-go` pin, shared between the two manifests) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BENCH-01 | `docs/BENCHMARKS.md` has no head-to-head table/multipliers | inspection (docs are not Go-testable) | The multiline census over `docs/BENCHMARKS.md` (Pattern 5), asserting TOTAL=0 for comparison patterns | N/A — census is the test |
| BENCH-02 | `CheckRegression` still fires on a real regression | unit + mutation rehearsal | `go test -count=1 ./internal/bench/... -run TestCheckRegression` (green baseline) + the D-08 mutation rehearsal (RED then reverted-green) | ✅ `internal/bench/regression_test.go` exists, 21 subtests already cover this |
| BENCH-02 | Comparison runner removed from `tools/bench` | build + census | `go build ./... && go vet ./...` (catches dangling references to deleted `-ts-binary`/`resolveTSBinary`) + census for `\bTS\b`/`\bheadtohead\b` over `tools/bench/**` | ✅ existing build/vet tooling |
| BENCH-03 | `bench.yml` publishes absolute numbers, no other implementation invoked | CI dispatch (manual — `workflow_dispatch`) + static census over the YAML | Dispatch the workflow's `publish` job on a branch and inspect the job summary/artifact; census `.github/workflows/bench.yml` for `npm install`, `ts-binary`, `TS`, `comparison` | Manual-only — GitHub Actions jobs are not locally `go test`-able |
| MEM-01 | Every retired-framing spine record superseded | inspection (external store, no CI gate) | `06-MEMORY-SWEEP.md`'s enumeration — no automated command possible, per D-15/CONTEXT.md's own "verified by inspection, not by a test" note | Manual-only, by design |
| MEM-02 | Fresh session recalls no port/parity framing | inspection (a live session, not a test) | Not automatable — D-15 explicitly declines post-sweep re-query verification | Manual-only, by design |

### Sampling Rate
- **Per task commit:** `go test -count=1 ./internal/bench/... ./tools/bench/...`
- **Per wave merge:** the full suite command above, plus a fresh run of the Pattern 5 census over the full Phase 6 file surface
- **Phase gate:** Full suite green + census TOTAL=0 (recorded verbatim, not just exit-code) before `/gsd-verify-work`; MEM-01/02 verified by inspection of the committed `06-MEMORY-SWEEP.md`, never by an automated gate (this is explicit in CONTEXT.md's own accumulated-context notes: "the engram memory sweep is last and is verified by inspection")

### Wave 0 Gaps
None — `internal/bench/regression_test.go`, `tools/bench/runner/main_test.go`, and `tools/bench/realcorpus/manifest_test.go` all already exist and already cover the relevant behaviors (regression gate logic; runner flag parsing; corpus manifest shape). The only test-file *edits* needed are subtractive (removing the `-ts-binary`/`resolveTSBinary`/`colbymchenry-codegraph` assertions identified in Pattern 2), not new test infrastructure.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No auth surface touched |
| V3 Session Management | No | N/A |
| V4 Access Control | No | N/A |
| V5 Input Validation | Marginal | `resolveOrClone` (`tools/bench/runner/main.go:441-464`, unchanged this phase) already pins every clone to `entry.CommitSHA`, never `HEAD` — this phase's only touch to input handling is *removing* the `-ts-binary` flag's path input, net-reducing the input surface, not adding to it |
| V6 Cryptography | No | N/A — no crypto in this phase's surface |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| A mutation rehearsal (D-08) accidentally left un-reverted, shipping a broken/weakened `CheckRegression` | Tampering | The FIXT-07 protocol's own pre-mutation `git diff --quiet` cleanliness gate + post-revert `git diff --stat` empty assertion (Pattern 4) — mechanically proves no mutation byte survives into the commit, exactly as `03-MUTATION-LOG.md`'s closing line states: "the final commit carries no mutation byte" |
| A census exclusion list quietly widened to make a framing sweep's TOTAL read zero | Repudiation | Every exclusion must trace to a recorded prior adjudication (Phase 5's own discipline, `05-08-PLAN.md:177-179`) — logged in `.planning/WINDOWS.md` (or this phase's equivalent tracking) rather than silently added |
| A `bench.yml` workflow edit accidentally re-adds a write-permission grant to the `rebless` job while restructuring the `headtohead`→`publish` job nearby | Elevation of Privilege | `rebless`'s existing `permissions: contents: read` at the workflow level (`bench.yml:92-93`) plus its own job-level comment (`:178-180`, "deliberately has no write token") should be spot-checked to confirm the fresh `publish` job's edit doesn't touch `rebless`'s block at all — they are siblings in the same file |
| Third-party Action pins drifting during the YAML edit | Tampering | Every `uses:` line in `bench.yml` is already pinned to a full commit SHA (`actions/checkout@df4cb1c...`, `actions/setup-go@924ae3a...`, etc.) — the plan must not introduce a floating tag/branch reference when copying steps into the fresh job |

## Sources

### Primary (HIGH confidence — direct `Read` of repository source this session)
- `.planning/phases/06-benchmark-de-coupling-memory-sweep/06-CONTEXT.md` — all 16 locked decisions, canonical refs, code context
- `.planning/REQUIREMENTS.md` — BENCH-01/02/03, MEM-01/02 verbatim, traceability table
- `.planning/STATE.md` — accumulated decisions, open issue #16, milestone framing
- `docs/BENCHMARKS.md` — full 301-line document, section-by-section audit
- `tools/bench/BASELINE.md` — full document, confirmed zero TS mentions
- `tools/bench/runner/main.go` — full 1038-line file, `measureSubject`/`runHeadToHead`/`resolveTSBinary`/flag parsing
- `internal/bench/regression.go`, `internal/bench/metrics.go`, `internal/bench/regression_test.go`, `internal/bench/rss.go` — full files
- `.github/workflows/bench.yml` — full 564-line workflow file
- `tools/bench/realcorpus/manifest.go` — full file, `Corpora()`, `Resolve()`
- `internal/corpora/manifest.go` — full file, `Manifest`/`Validate`/`Note` field
- `corpora/manifest.json` — full file, `Note` field content
- `.planning/phases/03-non-vacuity-proof-unconditional-ci-execution/03-MUTATION-LOG.md` — full FIXT-07 mutation protocol, all five families
- `.planning/phases/05-process-ci-in-tree-sweep/05-07-PLAN.md`, `05-08-PLAN.md` — full census-instrument methodology
- `.planning/WINDOWS.md` — entries 14-19, specifically entry 16 (bench-package framing, Phase 6 by design)
- `tools/bench/runner/main_test.go`, `tools/bench/realcorpus/manifest_test.go` — grep-confirmed test fallout for D-07/D-09
- `.mcp.json` (this repo) — confirmed no `engram` server registered
- `.planning/config.json` — confirmed `nyquist_validation: true`, `security_enforcement: true`

### Secondary (MEDIUM confidence)
- This session's live `dig`/`curl` check of `mcp-gw.fzymgc.house` — a point-in-time network probe from this session's own host, not the eventual plan-execution session

### Tertiary (LOW confidence — flagged `[ASSUMED]`, not verified this session)
- `mcp__engram__*` tool call signatures and pagination contract (Pattern 6) — sourced from the `engram:curating-memory` skill's description text visible in this session's tool listing, never exercised via an actual tool call

## Metadata

**Confidence breakdown:**
- Standard stack: N/A — no new dependencies this phase
- Architecture (benchmark/CI half): HIGH — every file, line range, and quoted string was read directly this session
- Architecture (memory-sweep half): LOW — no engram tool access this session; entirely dependent on live re-verification at execution time (already required by D-16)
- Pitfalls: HIGH — Pitfall 1 and 3 are grounded in this repo's own recorded incidents (Phase 5's census misses); Pitfall 2 is derived directly from reading `regression.go`'s guard ordering

**Research date:** 2026-08-16
**Valid until:** 14 days (the in-repo half is stable until the phase itself edits the files it describes; the engram-reachability finding is explicitly time-sensitive and must be re-checked at plan-execution time regardless of this window)
