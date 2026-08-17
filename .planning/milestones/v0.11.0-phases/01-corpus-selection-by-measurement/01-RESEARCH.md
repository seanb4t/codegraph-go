# Phase 1: Corpus Selection by Measurement - Research

**Researched:** 2026-08-14
**Domain:** Go CLI/MCP instrumentation extension + reproducible third-party corpus fetch/cache in CI
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Measurement instrument**

- **D-01:** The instrument is an extension of `codegraph status`, not a throwaway binary under `tools/`. `status` gains an **Edges by Kind** section alongside the existing Nodes by Kind. — Reversibility: costly.
- **D-02:** The new section lands on **all three surfaces** — human text, `--json` (`query.StatusResult` / `query.MarshalStatusJSON`), and the MCP status resource. Phase 1 owns a re-freeze of `testdata/wireoracle/transcripts/resources-read-status.golden`. — Reversibility: costly.
- **D-03:** `StatusResult.FilesByLanguage` (`internal/query/status.go:57`) already exists but carries `json:"-"`. Phase 1 un-suppresses it in the same diff as `edgesByKind`. `languages []string` stays. — Reversibility: costly.
- **D-04:** `edgesByKind` is **sparse by default, dense behind an opt-in flag**. In dense mode the map carries **all 9 `RANK_EDGES` kinds including explicit zeros**, key set **DERIVED from the existing `RANK_EDGES` constant, never hand-listed**, with a key-set-equality test. **The measurement run uses the dense flag.**
- **D-05:** `codegraph://status` (the MCP resource) is argument-less and cannot opt in. It emits **sparse**, matching the flagless CLI default. CLI-default and MCP therefore agree; density stays CLI-only.

**Measurement record shape**

- **D-06:** The record is **machine-readable JSON as source of truth, with generated prose**. Raw `status --json` output per candidate, keyed by `repo@SHA`, is committed; the human-readable markdown is generated from it.
- **D-07:** The record is **re-runnable, not frozen once**. A Taskfile target re-indexes the locked corpora and regenerates the JSON, and a **drift guard asserts the coverage claim still holds**.
- **D-08:** Guard cost is split by trigger. **Every CI run** performs the cheap check: assert the *committed* JSON satisfies the coverage claim — no indexing, no corpora needed. A **path-filtered job re-indexes and diffs only when the pinned-SHA manifest changes.**
- **D-09:** **One corpora manifest is the sole pin authority.** Carries, per entry: repository, commit SHA, license, and a `locked` flag. The Taskfile fetch target, the CI cache key, and the drift guard all read it; the measurements JSON references entries by `repo@SHA`. **Rejected candidates remain in the manifest marked unlocked.**

**Corpus scope & fetch**

- **D-10:** A corpus entry means a **whole repository, always** — no subtree pinning. Known consequence: likely rules `apache/arrow` out on size, making finding a dedicated C# corpus part of Phase 1's job. — Reversibility: reversible (an optional `subtree` field later is additive).
- **D-11:** Fetch is a **shallow git fetch at the pinned SHA** (`git init` + `git fetch --depth 1 origin <sha>` + checkout). Chosen over a codeload tarball specifically because **the fetched tree keeps a real `.git` directory**, exercising `internal/gitmeta` / `StatusResult.WorktreeMismatch`.
- **D-12:** Fetched corpora land **outside the working tree**, at an **XDG-relative default** — `${XDG_CACHE_HOME:-$HOME/.cache}/…` — **overridable via `CODEGRAPH_CORPUS_DIR`**. Echoes the existing `CODEGRAPH_{JAVA,PYTHON,TSJS}_CORPUS` convention. An in-tree gitignored path was explicitly not chosen.
- **D-13:** CI caching is **one `actions/cache` entry per corpus, keyed `corpus-{repo}-{sha}`.** First `actions/cache` usage in the repo. **No `restore-keys`** — a prefix match on a SHA key would be wrong by definition. GitHub gives a repo **10 GB total cache with LRU eviction**. **FIXT-02's requirement stands: a cache miss falls through to a fetch, never to a skip.**

**Coverage bar & fallback**

- **D-14:** The bar is a **threshold per kind, not the roadmap's stated "non-zero" floor.**
- **D-15:** **N is set from measured data, then frozen.** The rationale for the chosen N must be written into the measurement record explicitly.
- **D-16:** Fallback is **real corpus first, behavioral corpus as a recorded fallback.** When the fallback is used, the measurement record must state the kind is covered synthetically.
- **D-17:** The search is bounded by **candidate count** — a fixed number of additional candidates beyond the shortlist. Every candidate tried is named in the record with what it scored. The exact count is left to planning.

### Claude's Discretion

- The dense-mode flag's name (`--all-kinds` / `--verbose` / other), and whether `nodesByKind` should receive the same dense treatment for consistency.
- Manifest file format (YAML vs JSON) and its path; where the generated prose document lands (`docs/` vs `testdata/golden/`); whether the generated prose is regenerated-and-diffed in CI or written once.
- What "running it twice from clean produces the same tree" is verified *by* (tree hash vs file count vs other), and whether the fetch target is idempotent by default or needs `--force`.
- The exact extra-candidate count in D-17, and the exact N in D-15 (data-driven by construction).
- Whether the 5 priority-4 languages carry a file-count threshold or stay at non-zero — only the *edge kinds* got a threshold decision.
- Whether a behavioral-corpus fallback is permissible for a *language* miss as well as an edge-kind miss.

### Deferred Ideas (OUT OF SCOPE)

- **Dense `nodesByKind`** — raised, deliberately left open (see Discretion above); if adopted, a later phase/milestone, not required for this one.
- **Dropping `StatusResult.Languages`** — offered alongside D-03 and not chosen (breaking JSON removal).
- **A file-count threshold for the 5 priority-4 languages** — D-14 set a threshold for edge *kinds* only.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| FIXT-01 | Corpus selection decided by measurement — a blocking Phase 1 spike indexes candidate third-party MIT/Apache-2.0 repos and records actual per-kind edge counts and per-language file counts, locking a final set covering all 9 `RANK_EDGES` kinds and the 5 priority-4 languages | §Architecture Patterns (measurement instrument extension), §Code Examples (edge-scan addition, RankEdges-derived dense key set), §Coverage Risk reasoning below, §Candidate Corpora table (live-verified license/size/language data) |
| FIXT-02 | Locked corpora fetched at exact pinned SHAs by a Taskfile target, restored from CI cache; no corpus source vendored; a re-fetch at the pinned SHA reproduces the same tree | §Fetch Mechanism (shallow git fetch at SHA, `allowReachableSHA1InWant`), §GitHub Actions Cache (D-13 pattern, cache-miss-falls-through-to-fetch), §Common Pitfalls (XDG_CACHE_HOME cross-platform gotcha, `TestWorkflowRunBodiesInvokeTask` constraint) |
</phase_requirements>

## Summary

This phase has two independent halves. **Half 1** extends an existing, well-understood Go code path (`query.Engine.Status`) to add a ninth data point (`edgesByKind`) alongside the existing `nodesByKind`/`filesByLanguage` breakdowns — mechanically identical to those two, using the same `IterateEdges("")` full-scan primitive `buildExpandAdjacency`/`buildRWRAdjacency` already use, tallied by `schema.Edge.Kind`, filtered/keyed against the already-canonical `query.RankEdges` var. This is low-risk, well-precedented Go work with one real design fork (sparse/dense computed inside `Engine.Status` vs. computed always and filtered at render time) that the planner must resolve. **Half 2** is genuinely new to this repository: a first-ever `actions/cache` usage, a first-ever CI network `git fetch` of a third-party repo, and a Taskfile target that must satisfy this repo's own `TestWorkflowRunBodiesInvokeTask` guard (every CI `run:` step must be exactly `task <target>`).

The single most consequential correction this research makes to the phase's own canonical references: **the wire-oracle transcript that actually embeds live `StatusResult` data and will be invalidated by D-02 is `testdata/wireoracle/transcripts/call-status.golden`** (the `codegraph_status` tool *call*, rendering via `query.RenderStatusMarkdown`) — **not** `resources-read-status.golden`, which reads the static per-tool documentation resource `codegraph://tools/status` (`internal/mcp/resources/status.md`, literally the four lines "Report index health and counts. / Arguments: path / Result: Markdown text.") and will NOT change from this phase's work. This was verified by reading both the wire-oracle scenario definitions and the golden file contents directly (see §Golden/Transcript Blast Radius).

For corpus selection, the four shortlisted repos are all live, licensed correctly (verified via `gh api .../license`, not GitHub's less-reliable `license.spdx_id` field alone), and their extractor-emission mechanics are grounded in code: `overrides` requires a same-name+arity method on both a type and a real (non-interface-only) supertype method, reachable only through `"embeds"`/`EdgeKindExtends`/`EdgeKindImplements` edges — all four language extractors (Go, Java, C#, TS/JS, Python) emit the underlying `RefKindEmbeds` supertype edge, so all are structurally capable, but Go's own idiomatic style (composition over inheritance, rare explicit method-shadowing) is the documented reason it measures zero. `type_of` requires an explicitly-typed variable declaration (`var x T`), never an inferred one (`x := T{}` or `var y = expr`) — idiomatic Go's dominant `:=` style is the reason it measures zero, and this generalizes: heavily-typed idioms (Java field/local declarations, C# `Type x = new()`, TypeScript type annotations) should measure non-zero readily. `apache/arrow` is confirmed a poor fit independent of size: it does not even solve the C# gap (no C# in its language mix), and D-10's whole-repo rule means indexing 32M+ bytes of C++ to reach 3.5M bytes of Python. This research proposes `JamesNK/Newtonsoft.Json` (MIT, license-verified live) as the primary C# candidate — small, pure-C#, heavy on abstract-class/virtual-override patterns (`JsonConverter` subclasses) and explicit typing.

**Primary recommendation:** extend `Engine.Status` with one more `IterateEdges("")` full scan (mirroring the existing node scan exactly), derive the dense key set from `query.RankEdges` at render/marshal time (not inside `Status` itself) so sparse vs. dense stays a presentation-layer decision, re-freeze only `call-status.golden` (not `resources-read-status.golden`), and build the corpus fetch as a plain bash+`jq` Taskfile target reading one committed JSON manifest — matching every existing convention in this repo (SHA-pinned actions, `jq`-driven Taskfile steps, `task <target>`-only CI run bodies, path-filtered sibling-workflow precedent for the heavier drift-guard leg).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Edge-kind tallying (`edgesByKind`) | Backend / query engine (`internal/query.Engine.Status`) | — | Same tier as the existing `nodesByKind`/`filesByLanguage` scans it extends; a pure graph-store read, no new tier |
| Sparse/dense rendering decision | CLI presentation (`internal/cli`, `internal/query/render_status.go`, `internal/cli/present`) | Backend (data source) | Filtering-vs-full-key-set is a display concern; the underlying `StatusResult` data should not need two different Engine call shapes |
| MCP status exposure | MCP / Backend boundary (`internal/mcp/tools.go`) | — | Argument-less passthrough to the same `Engine.Status` + `RenderStatusMarkdown` path; no new MCP-tier logic |
| Corpora manifest (source of truth for repo/SHA/license/locked) | Repo-root config artifact (flat file, not code) | Backend (drift-guard reader), CI (fetch + cache key) | A single file three different consumers (Taskfile, CI cache step, Go drift-guard test) read — must not become "restated" in any consumer |
| Corpus fetch (shallow git @ SHA) | Build tooling / Taskfile | CI (workflow orchestration) | `Taskfile.yml` is this repo's sole CI-job-body definition (`TestWorkflowRunBodiesInvokeTask`); the fetch logic must live there, not inline in a workflow `run:` block |
| CI cache (`actions/cache`) | CI / GitHub Actions workflow | — | Native GitHub Actions primitive; no Go/Taskfile equivalent exists or should exist |
| Measurement record generation (JSON + prose) | Build tooling (Go program invoked by Taskfile, analogous to `testdata/golden/gocapture`) | Backend (`query.MarshalStatusJSON` is the payload source) | Precedent: `gocapture/main.go` already drives the indexer+query pipeline programmatically and writes structured fixtures — the measurement tool is the same shape, one level up |
| Drift guard (coverage-claim assertion) | Go test (`go test`), CI | — | Matches the "positive assertion" rule (`84d1gfpywd`) — must be a real `go test`, not a shell `jq` threshold check alone, so it participates in `task test` |

## Project Constraints (from CLAUDE.md)

`.claude/CLAUDE.md` (project-level, checked into this repo) establishes the following directives this phase must honor:

- **Tech stack**: Go only, single static binary per platform. No new runtime dependency this phase should introduce anything that breaks that (confirmed: this phase adds zero new `go.mod` entries — see §Standard Stack).
- **Supply chain**: minimal, audited dependencies; CGo tree-sitter is the *only* documented CGo exception. This phase touches no parser/CGo surface.
- **Compatibility**: TS CodeGraph v1.3.x is a functionality *baseline*, not a binding constraint (2026-08-09 ruling) — deliberate divergence is acceptable with documentation. Relevant here: D-05's MCP-argument-less-sparse-only design is a deliberate, documented divergence pattern, not a TS-parity concern.
- **Architecture**: v1 storage/process design must accommodate milestone-2 team-scale features without a rewrite. Not directly implicated by this phase (no storage schema change — `edgesByKind` is derived at read time, not persisted).
- **Licensing**: MIT with attribution. Directly binds §Candidate Corpora — every corpus MUST be MIT or Apache-2.0, verified live via `gh api repos/<org>/<repo>/license`, not assumed from README badges.
- Repo-standing rules also binding this phase (from `.planning/STATE.md` "Standing decisions"): **a guard must carry a positive assertion** (`84d1gfpywd` — directly shapes D-04/D-07/D-08's drift guard design); **a gate is not trusted until demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation** (relevant to whatever mutation-proof precedent the drift guard follows, though the actual mutation-proof work is Phase 3/FIXT-07 — this phase should still write the guard so it is falsifiable, not merely green); **`Taskfile.yml` is the single definition of every CI job body** (`TestWorkflowRunBodiesInvokeTask` — binding on the fetch target's CI wiring, verified directly, see §Common Pitfalls).

## Standard Stack

### Core

No new Go module dependency is required by this phase. `internal/query.Status` extension uses only what is already imported (`encoding/json`, the existing `graphstore.Reader` interface). The corpora manifest, fetch target, and drift guard use only tools already present in this repo's toolchain.

| Tool | Version | Purpose | Why Standard (for this repo) |
|------|---------|---------|-------------------------------|
| `git` | host-installed, already a hard CI/dev dependency | Shallow fetch at pinned SHA (D-11) | Already required by every existing workflow (`actions/checkout`) and by `gocapture`/`golden_parity_test.go`'s existing `git rev-parse`/`git clone` use |
| `jq` | host-installed, already used extensively | Parse the JSON corpora manifest inside Taskfile `cmds:` bash blocks | [VERIFIED: Taskfile.yml:352-680] — `jq` is already the standard manifest/JSON-parsing tool in this Taskfile (`release:dry-run-signed`, `verify:release-assets`, etc. all shell out to it); no new tool class introduced |
| `go.yaml.in/yaml/v3` v3.0.4 | already a direct `go.mod` dependency [VERIFIED: go.mod:33] | Optional: if the corpora manifest is authored as YAML instead of JSON, this is a zero-new-dependency parser for the Go-side drift guard | Already imported by `internal/upgrade/taskfile_shape_test.go` for `.goreleaser.yaml`/workflow parsing |
| `actions/cache` | `v6.1.0`, commit `55cc8345863c7cc4c66a329aec7e433d2d1c52a9` [VERIFIED: `gh api repos/actions/cache/releases/latest` + `gh api repos/actions/cache/commits/v6.1.0`, live-resolved 2026-08-14] | CI corpus cache (D-13) | First-party GitHub Action, matches this repo's existing full-commit-SHA-pin-with-version-comment convention [VERIFIED: `.github/workflows/ci.yml:37-52`, e.g. `actions/checkout@df4cb1c...# v6.0.3`] |

### Manifest format — Claude's Discretion, recommendation: JSON

| Format | Tradeoff |
|--------|----------|
| **JSON (recommended)** | Zero new dependency (stdlib `encoding/json` on the Go side); `jq` — already this Taskfile's standard JSON tool — reads it trivially in bash for both the fetch-target loop and the CI cache-key derivation. Matches the sibling precedent of `release-please-config.json`/`.release-please-manifest.json`/`dist/*.sigstore.json` already living at repo root as JSON config artifacts. |
| YAML | Also zero-new-dependency (`go.yaml.in/yaml/v3` already vendored) and matches `Taskfile.yml`/workflow-file style, but bash-side parsing without `jq`-equivalent is comparatively awkward — this repo has no `yq` precedent anywhere in Taskfile.yml [VERIFIED: no `yq` invocation found in Taskfile.yml]. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Full `git fetch --depth 1 origin <sha>` per D-11 | `codeload.github.com/<org>/<repo>/tar.gz/<sha>` tarball | Rejected by D-11 itself — no `.git`, so `internal/gitmeta`/`WorktreeMismatch` code path stays unexercised across every corpus. Not re-litigated here. |
| One `actions/cache` entry per corpus (D-13) | One combined cache entry keyed on a manifest-file hash | Rejected by D-13 itself — refetches every corpus on any single pin bump, more likely to hit the 10 GB ceiling. Not re-litigated here. |

## Package Legitimacy Audit

This phase introduces **zero new Go/npm/pip dependencies** — the npm/PyPI/crates-oriented Package Legitimacy Gate protocol does not apply to any language-ecosystem package here. The one new *supply-chain surface* is the GitHub Action `actions/cache`, audited informally below since it is not an npm/PyPI/crates package:

| Component | Registry/Source | Age | Popularity | Source Repo | Verdict | Disposition |
|-----------|-----------------|-----|------------|-------------|---------|-------------|
| `actions/cache` | GitHub Marketplace / `github.com/actions/cache` | First release 2019, latest `v6.1.0` released 2026-06-26 [VERIFIED: `gh api repos/actions/cache/releases/latest`] | First-party `actions` org (owns `actions/checkout`, `actions/setup-go`, already trusted in this repo) | `github.com/actions/cache` | OK | Approved — pin the exact commit SHA `55cc8345863c7cc4c66a329aec7e433d2d1c52a9` with a `# v6.1.0` trailing comment, matching this repo's existing convention exactly |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

## Architecture Patterns

### System Architecture Diagram

```
                    MEASUREMENT (Half 1: extend the instrument)
┌──────────────────────────────────────────────────────────────────────┐
│  graphstore.Reader                                                    │
│    IterateNodes() ──► nodesByKind, filesByLanguage  (existing scan)   │
│    IterateEdges("") ──► edgesByKind                 (NEW scan, same   │
│                                                        cost class)     │
│                          │                                            │
│                          ▼                                            │
│              query.Engine.Status(ctx) → StatusResult                  │
│                          │                                            │
│         ┌────────────────┼─────────────────┐                         │
│         ▼                ▼                 ▼                         │
│  RenderStatusText   MarshalStatusJSON  RenderStatusMarkdown           │
│  (CLI human text,   (CLI --json AND    (MCP codegraph_status          │
│   present.Render     measurement        tool call — ALWAYS sparse,    │
│   Status TTY twin)   record source)      D-05, no dense flag exists)  │
│         │                │                        │                  │
│    sparse or dense   sparse or dense          sparse only            │
│    per --all-kinds   per --all-kinds                                 │
└──────────────────────────────────────────────────────────────────────┘

                    CORPUS FETCH (Half 2: new to this repo)
┌──────────────────────────────────────────────────────────────────────┐
│  corpora manifest (JSON, repo root or docs/) ── sole pin authority    │
│         │                    │                    │                  │
│         ▼                    ▼                    ▼                  │
│  Taskfile fetch target   CI cache step        Go drift-guard test    │
│  (per entry:              (per entry:          (reads committed       │
│   git init +               actions/cache@v6     measurement JSON,     │
│   git remote add +         key: corpus-{repo}    asserts coverage     │
│   git fetch --depth 1      -{sha}, no            claim still holds —  │
│   origin <sha> +           restore-keys)         cheap, every PR)     │
│   git checkout             miss → run the                            │
│   FETCH_HEAD)               fetch task            path-filtered      │
│    lands at                (never skip)          heavier leg         │
│  ${XDG_CACHE_HOME:-                              (re-index + diff,   │
│   $HOME/.cache}/…                                 only when the      │
│  or $CODEGRAPH_CORPUS_DIR                          manifest changes) │
└──────────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/query/
├── status.go            # StatusResult gains EdgesByKind map[string]int64; FilesByLanguage json tag un-suppressed
├── render_status.go     # RenderStatusText/RenderStatusMarkdown gain an "Edges by Kind:" section
internal/cli/
├── status.go             # newStatusCmd gains the dense-mode flag (name: Claude's discretion)
internal/cli/present/
├── status.go             # package-local duplicate renderer (see Common Pitfalls) — needs the SAME section added for TTY parity
testdata/golden/
├── gocapture/            # existing precedent: a Go program driving indexer+query, writing structured fixtures — model for the measurement tool
corpora/                  # NEW — or wherever discretion lands the manifest; recommend repo-root `corpora.json` or `testdata/corpora/manifest.json`
├── manifest.json          # sole pin authority (D-09): [{repo, sha, license, locked, ...}]
docs/ or testdata/golden/
├── CORPUS-MEASUREMENT.md  # generated prose (D-06) — never hand-edited, regenerated from the committed JSON
.github/workflows/
├── ci.yml                 # gains actions/cache step(s) — likely inside an existing or new job; run: bodies must stay `task <target>`-only
├── corpus-drift.yml (or similar) # NEW sibling workflow for D-08's path-filtered heavier leg, mirroring linux-cross-canary.yml's `pull_request: paths:` pattern
```

### Pattern 1: Extending `Engine.Status` with a new full-graph-scan breakdown

**What:** `edgesByKind` is computed exactly like the existing `nodesByKind` — one more full iterator scan, tallied by a string key, filtered/sorted only at render time.
**When to use:** Any time `Status()` needs a new per-kind/per-category count derived from a full scan of an existing iterator.
**Example (existing precedent, verbatim, to mirror):**
```go
// Source: internal/query/status.go:238-253 [VERIFIED]
nodeIt, err := e.reader.IterateNodes()
if err != nil {
    return StatusResult{}, err
}
defer nodeIt.Close()

nodesByKind := make(map[string]int64)
var nodeCount int64
for nodeIt.Next() {
    n := nodeIt.Node()
    nodeCount++
    nodesByKind[n.Kind]++
}
if err := nodeIt.Err(); err != nil {
    return StatusResult{}, err
}
```
The new `edgesByKind` scan is the direct analog, using `e.reader.IterateEdges("")` — the SAME full-scan primitive `buildExpandAdjacency` already uses (`internal/query/expand.go:263-283` [VERIFIED]) — tallied by `e.Kind` instead of `n.Kind`. **Cost note:** `Status()` today deliberately avoids any edge scan, reading `meta.GetEdgeCount()` instead specifically to skip "a second full edge scan" [VERIFIED: internal/query/status.go:205-206, doc comment: "edgeCount from GetMeta (avoiding a second full edge scan — the indexer stamps Meta.EdgeCount at index time...)"]. Adding `edgesByKind` makes a full edge scan **unconditional** on every `status` call (sparse or dense — the per-kind tally must be computed to know which kinds are non-zero even for sparse display), a real, deliberate performance regression this phase should document, not silently introduce.

### Pattern 2: Deriving the dense key set from `RankEdges`, never hand-listing it

**What:** D-04 requires the 9-kind dense key set be DERIVED from the canonical set, with a key-set-equality test.
**When to use:** Any dense-mode rendering/marshaling of `edgesByKind`.
**Example (existing canonical source to derive from, verbatim):**
```go
// Source: internal/query/rwr.go:21-31 [VERIFIED]
var RankEdges = map[string]bool{
    goextract.RefKindCalls:        true,
    goextract.RefKindReferences:   true,
    goextract.EdgeKindExtends:     true,
    goextract.EdgeKindImplements:  true,
    goextract.EdgeKindOverrides:   true,
    goextract.RefKindInstantiates: true,
    goextract.RefKindReturns:      true,
    goextract.RefKindTypeOf:       true,
    goextract.RefKindImports:      true,
}
```
A dense render/marshal helper should build its 9-entry output by iterating `sort.Strings(keys(RankEdges))` and looking up each in the sparse `EdgesByKind` map (defaulting missing keys to `0`), NOT by hard-coding the 9 literal kind strings a second time — mirrors the existing precedent at `internal/query/rwr_test.go:13-37` (`TestRankEdges`), which already asserts `RankEdges` has exactly the 9 members sourced from `goextract`'s constants, "never re-declared literals."

### Pattern 3: Path-filtered sibling workflow for the "only when the manifest changes" heavier drift-guard leg (D-08)

**What:** D-08's two-tier guard (cheap check every run; heavier re-index+diff only when the corpora manifest changes) has a **direct precedent already in this repo** — a separate workflow file triggered by `pull_request: paths: [...]`, not an `if:` condition bolted onto an existing job.
**When to use:** The heavier leg of D-07/D-08's drift guard.
**Example (existing precedent, verbatim structure):**
```yaml
# Source: .github/workflows/linux-cross-canary.yml:71-83 [VERIFIED]
name: linux-cross-canary

on:
  workflow_dispatch:
  pull_request:
    paths:
      - ".github/workflows/release.yml"
      - ".github/workflows/linux-cross-canary.yml"
      - ".goreleaser.yaml"
      - "Taskfile.yml"
      - "go.mod"
      - "go.sum"

permissions:
  contents: read
```
This file is described as "a permanent, dispatchable canary... a sibling of darwin-toolchain-canary.yml, not a throwaway... intentionally NOT in main's required-status-check set" — the exact shape D-08's heavier leg needs (path-scoped to the corpora manifest file, not required on every PR, dispatchable for manual re-verification). `darwin-toolchain-canary.yml:29` uses the identical `paths:` pattern as a second confirming example.

### Anti-Patterns to Avoid

- **Using `os.UserCacheDir()` for D-12's XDG path.** Go's stdlib `os.UserCacheDir()` resolves to `~/Library/Caches` on Darwin, NOT `$HOME/.cache` — it does NOT implement the literal `${XDG_CACHE_HOME:-$HOME/.cache}` formula D-12 specifies on every OS. This repo's own precedent for this exact class of decision (`internal/agents/opencode.go:37-42`, XDG_CONFIG_HOME resolution) explicitly notes "no Windows special-case" — the equivalent discipline here is "no macOS special-case": hand-roll `os.Getenv("XDG_CACHE_HOME")` with a `filepath.Join(home, ".cache")` fallback, literally, on every platform.
- **Computing sparse and dense via two different `Engine.Status(ctx, dense bool)` call shapes.** This would mean the JSON/CLI dense flag changes what `Engine.Status` computes, while MCP always calls the sparse shape — two different Engine-level code paths for the same data is more surface to keep in sync than computing the full per-kind tally always (cheap relative to the scan itself) and deciding sparse-vs-dense purely at the render/marshal boundary (mirrors how `sortedCounts` already filters `count > 0` at render time for `nodesByKind`/`filesByLanguage`, never inside `Status()`).
- **Restating the 9 `RANK_EDGES` kind strings anywhere new** (in the manifest schema comments, in the drift-guard's expected-kinds list, in the measurement-record's prose generator). Every one of these MUST import and iterate `query.RankEdges`, exactly as `rwr_test.go` already does.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-platform "OS cache dir" resolution | A custom `runtime.GOOS`-switched cache-dir resolver | The literal `${XDG_CACHE_HOME:-$HOME/.cache}` formula, hand-rolled with two `os.Getenv`/fallback lines (per D-12's own explicit spec) | D-12 is deliberately narrower than Go's own `os.UserCacheDir()` (which diverges on Darwin) — this is one of the rare cases where NOT using the stdlib convenience function is correct, because the spec is XDG-literal, not OS-native |
| JSON manifest parsing in bash | A hand-rolled `grep`/`sed`/`awk` JSON scraper in the Taskfile fetch target | `jq`, already this Taskfile's standard tool for `dist/artifacts.json` | Every other JSON-consuming Taskfile target already uses `jq` [VERIFIED: Taskfile.yml lines 352-680 sampled] |
| CI cache miss/hit branching | A custom "does this cache dir exist and have the right marker file" shell check | `actions/cache`'s own `cache-hit` step output, checked via `if: steps.<id>.outputs.cache-hit != 'true'` | This is the documented, first-party mechanism `actions/cache` ships specifically for this — hand-rolling it duplicates logic GitHub's own action already gets right (and the action's own post-job hook auto-saves on miss, which a hand-rolled check cannot replicate without a second explicit save step) |
| Wire-oracle transcript re-freeze tooling | A bespoke "auto-freeze" script that overwrites `.golden` files on test failure | **No existing tool does this in this repo** (see Open Questions — this is a genuine gap, not a "don't hand-roll," but also not something to build ad hoc mid-phase without checking `tools/transcriptfreeze`'s actual purpose first — see below) | `tools/transcriptfreeze` exists but is the D-03 **anti-regeneration guard** (`check:transcript-freeze` task) that detects when a transcript changed together with `internal/mcp/*.go` without review — it is not a freeze-writer. Confirm this before assuming a freeze tool exists. |

**Key insight:** almost everything this phase needs already has an exact precedent somewhere in this repo's own Taskfile/CI/Go code. The risk is not "no library exists for X" (the classic Don't-Hand-Roll failure mode) — it is **inventing a new pattern where an existing repo-local one already applies**, which then fails this repo's own `TestWorkflowRunBodiesInvokeTask`/`TestTaskfileGatesFailLoud`/`TestTaskfileWrapperIsSerial` guards.

## Common Pitfalls

### Pitfall 1: Re-freezing the wrong wire-oracle transcript

**What goes wrong:** CONTEXT.md's own canonical references name `testdata/wireoracle/transcripts/resources-read-status.golden` as the transcript D-02 commits Phase 1 to re-freezing. This is **verified incorrect** by reading both files.
**Why it happens:** There are two differently-shaped MCP surfaces both involving the word "status": (1) `codegraph://tools/status` — a STATIC per-tool documentation resource, `resources/read`, content is the fixed file `internal/mcp/resources/status.md` (`# codegraph_status\n\nReport index health and counts.\n\n## Arguments\n\n- \`path\` (string, optional) — Repo path (default: server cwd).\n\n## Result\n\nMarkdown text.\n` [VERIFIED: internal/mcp/resources/status.md, read in full]); and (2) the `codegraph_status` TOOL CALL (`tools/call`), whose result IS `query.RenderStatusMarkdown(StatusResult)`'s live output. The wire-oracle scenario named `"resources-read-status"` (`test/wireoracle/scenarios.go:1322`) exercises path (1); the scenario named `"call-status"` (`test/wireoracle/scenarios.go:753-759`) exercises path (2).
**How to avoid:** Re-freeze `testdata/wireoracle/transcripts/call-status.golden` — its committed second line is `{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"**CodeGraph Status**\n\n**Files indexed:** 3\n**Total nodes:** 9\n**Total edges:** 11\n**Database size:** 0.01 MB\n**Backend:** pebble\n\n**Nodes by Kind:**\n- function: 4\n- file: 3\n- package: 2\n\n**Languages:**\n- go: 3\n\nIndex is up to date.\n"}]}}` [VERIFIED, transcript read in full] — this text literally changes once `RenderStatusMarkdown` grows an "Edges by Kind:" section. `resources-read-status.golden` does NOT need to change: its content is the static `status.md` doc text, which says only "Markdown text." and needs no edit for a new section within that markdown.
**Warning signs:** If a plan task lists `resources-read-status.golden` as the file to re-freeze and does NOT also list `call-status.golden`, that task is wrong and will either miss a real transcript break or waste effort re-freezing a transcript that never changed.

### Pitfall 2: `present/status.go` is a fourth, easy-to-miss human-text renderer

**What goes wrong:** D-02 names "all three surfaces" (human text, `--json`, MCP resource), but `internal/cli/present/status.go` is a **package-local duplicate** of `query.RenderStatusText`'s exact structure (its own doc comment states this explicitly: "a package-local duplicate of internal/query's unexported kindCount... matching this codebase's existing precedent of package-local duplication across the query/cli boundary" [VERIFIED: internal/cli/present/status.go:14-20]), used on a real TTY instead of the plain-text renderer. If `RenderStatusText` gains an "Edges by Kind:" section but `present.RenderStatus` does not, a TTY user sees a different `status` than a piped/CI user — a real behavioral inconsistency, not just a missed test.
**Why it happens:** The phase description's "all three surfaces" framing (human/JSON/MCP) undercounts — there are really 4 render call sites: `query.RenderStatusText` (piped), `present.RenderStatus` (TTY, lipgloss-styled), `query.MarshalStatusJSON` (`--json`), `query.RenderStatusMarkdown` (MCP).
**How to avoid:** Any plan task touching `RenderStatusText` for the new section must have a matching task (or the same task, widened) touching `internal/cli/present/status.go`'s `writeBreakdownText`/`RenderStatus` call sequence. `internal/cli/present/status_test.go`'s `fixtureStatusResult()` (line 22-38) will also need an `EdgesByKind` field added if a test asserts its presence.
**Warning signs:** A plan that only lists `internal/query/render_status.go` and `internal/cli/status.go` under "files touched" for the human-text half.

### Pitfall 3: The `status.go` decision-table doc comment goes stale

**What goes wrong:** `internal/query/status.go`'s own top-of-file doc comment is an explicit "per-key decision table" documenting, among other things, that `filesByLanguage` carries `json:"-"` and is "Go-internal only... NOT emitted in --json" [VERIFIED: internal/query/status.go:44]. D-03 un-suppresses this key. If the table isn't updated in the same diff, the file's own authoritative documentation contradicts its own struct tags.
**Why it happens:** The table is long (lines 20-45) and easy to treat as background context rather than a live spec that must track the code.
**How to avoid:** Include updating this table's `filesByLanguage` row (and adding a new `edgesByKind` row) as an explicit task, not an incidental edit.
**Warning signs:** `go vet`/tests pass but the doc comment still says `json:"-"` after the tag changes.

### Pitfall 4: `TestWorkflowRunBodiesInvokeTask` rejects a naive CI cache wiring

**What goes wrong:** This repo enforces (`internal/upgrade/taskfile_shape_test.go:1343-1384`, `TestWorkflowRunBodiesInvokeTask`) that every `run:` step's body in specific in-scope jobs (`ci.yml`'s `test`, `actionlint`, `goreleaser-check`, `reproducibility`, `perf-regression`, `transcript-freeze`, `tool-vuln`, plus `release-please.yml`'s `pretag-gate`) must be **exactly** `task <target>` (regex `^task\s+[A-Za-z0-9:_-]+$` after stripping comments/blanks) unless the step is in a literal, reasoned exception list [VERIFIED: internal/upgrade/taskfile_shape_test.go:109-153, `inScopeJobs`/`runBodyExceptions`]. A `run: echo "cache hit"` or any inline shell in a step within one of those jobs fails this test.
**Why it happens:** `actions/cache` steps use `uses:`, not `run:` — those are unconstrained by this guard [VERIFIED: `checkStepInvokesTask`, taskfile_shape_test.go:1295-1310, "a step with no run: body (a `uses:` step) is unconstrained"]. The risk is the conditional-fetch step that follows the cache step (`if: steps.cache.outputs.cache-hit != 'true'`) — if that step's `run:` body is anything other than `task <target>`, and it lands inside one of the `inScopeJobs`, it fails.
**How to avoid:** Either (a) add the fetch as a genuine Taskfile target invoked as a bare `run: task corpora:fetch` step, or (b) put the new corpus-fetch/cache job in a workflow/job NOT in the literal `inScopeJobs` fixture (a brand-new job name is NOT automatically in scope — the fixture is a hard-coded list, `taskfile_shape_test.go:109-118` — but the repo's own discipline strongly implies a new job should be ADDED to that fixture, not silently left out; leaving it out is technically guard-passing but inconsistent with the repo's stated convention).
**Warning signs:** A new `run:` step under `ci.yml`'s existing `test` job with inline bash for the cache-miss fetch — this job IS in `inScopeJobs` today.

### Pitfall 5: `apache/arrow` is a poor fit independent of size

**What goes wrong:** Treating `apache/arrow` as "ruled out on size, otherwise a fine C# source" is wrong — arrow's language mix has **no C# at all** [VERIFIED live: `gh api repos/apache/arrow/languages` returns `C++, Python, Ruby, Cython, R, C, MATLAB, CMake, Shell, Meson, Dockerfile, Thrift, Batchfile, Vala, Jinja, Objective-C++, Lua, Makefile, HTML, JavaScript, CSS` — no `C#` key present at all].
**Why it happens:** Arrow's ecosystem includes an "Arrow ADBC"/language-bindings story that might be conflated with a C# binding existing in the monorepo; it does not, as of this measurement.
**How to avoid:** Do not budget arrow as a candidate C# source under any framing. It could still be evaluated for its Python coverage (3.5M bytes, real material), but D-10's whole-repo rule means paying for ~32M bytes of C++ (and R, Ruby, Cython, MATLAB) to get there — the size argument the CONTEXT already flags stands independently of the C# question.
**Warning signs:** A plan task that frames arrow-vs-C#-candidate as an either/or tradeoff rather than arrow having zero C# to offer.

### Pitfall 6: GitHub repo `size` (KB) is full-history size, not shallow-clone size

**What goes wrong:** `gh api repos/<org>/<repo> --jq .size` reports the whole repository's packed size including full git history, not what a `--depth 1` fetch at one SHA will actually transfer/store. Using this number directly to reason about the 10 GB CI cache ceiling risks over- or under-estimating.
**Why it happens:** It is the only size figure readily available via a single, fast API call, and this research used it for a first-pass filter (see §Candidate Corpora — flagged there).
**How to avoid:** The measurement phase's own fetch-and-index run is the authoritative source for actual on-disk shallow-clone size per corpus — record it in the measurement JSON alongside the edge/language counts, and size the CI cache budget from THAT, not from the `gh api` `size` field.
**Warning signs:** A plan or the manifest documenting a per-corpus "size" figure sourced only from `gh api repos/.../` without a note that it's full-history size.

## Code Examples

### Full edge scan, tallied by kind (the new primitive `edgesByKind` needs)

```go
// Source: internal/query/expand.go:263-283 [VERIFIED] — buildExpandAdjacency
// already demonstrates the exact IterateEdges("") full-scan shape; the new
// edgesByKind computation in Engine.Status is a straight analog, tallying
// e.Kind into a map instead of building an adjacency list.
it, err := r.IterateEdges("")
if err != nil {
    return nil, nil, err
}
defer it.Close()

edgesByKind := make(map[string]int64)
for it.Next() {
    e := it.Edge()
    edgesByKind[e.Kind]++
}
if err := it.Err(); err != nil {
    return nil, nil, err
}
```
Note: `internal/graphstore/keys.go:96`'s `edgeKey(src, kind, dst)` confirms edges are keyed `edge/<src>/<kind>/<dst>` — kind is NOT a leading key-prefix segment [VERIFIED: internal/graphstore/keys.go:96], so there is no cheaper prefix-scan-per-kind shortcut available; a single full scan tallying all kinds at once (as above) is the correct and only reasonably efficient approach, matching `buildExpandAdjacency`'s existing precedent of one full scan rather than N per-kind scans.

### Shallow git fetch at a pinned SHA (D-11)

```bash
# Verified working pattern for GitHub-hosted repos (all four shortlist
# candidates + all proposed C# candidates are GitHub-hosted). GitHub
# advertises uploadpack.allowReachableSHA1InWant / allowTipSHA1InWant for
# every repo it serves [CITED: https://git-scm.com/docs/git-fetch,
# cross-confirmed via WebSearch — "GitHub (and most modern git servers)
# advertise uploadpack.allowReachableSHA1InWant... so git fetch origin <sha>
# generally succeeds directly against a bare commit SHA without needing a
# full clone"], so the depth-1-by-SHA fetch below works without a fallback
# for every candidate in this phase's shortlist.
mkdir -p "$dest" && cd "$dest"
git init -q
git remote add origin "https://github.com/${org}/${repo}.git"
git fetch --depth 1 origin "${sha}"
git checkout -q FETCH_HEAD
```
**Fallback (non-GitHub or a server without the capability enabled):** progressively deepen a branch-tip fetch (`git fetch --depth 1 origin <branch>`, then `git fetch --deepen=N origin` until `<sha>` is reachable) rather than falling back to a tarball, to preserve D-11's `.git`-must-be-real requirement. This repo's candidates do not need this fallback path today (all GitHub-hosted), but the Taskfile target should still handle a non-zero exit from the direct SHA fetch loudly (matching rule `84d1gfpywd`/this repo's `TestTaskfileGatesFailLoud` convention of `preconditions:`+`msg:` or an explicit `set -euo pipefail` failure) rather than silently degrading.

### `actions/cache` — one entry per corpus, no `restore-keys`, miss falls through to fetch (D-13)

```yaml
# Pattern synthesized from actions/cache's documented usage [CITED:
# github.com/actions/cache README, fetched 2026-08-14] plus this repo's
# existing SHA-pin convention [VERIFIED: .github/workflows/ci.yml:37-52].
- name: Restore corpus cache (gohugoio/hugo@<sha>)
  id: cache-corpus-hugo
  uses: actions/cache@55cc8345863c7cc4c66a329aec7e433d2d1c52a9 # v6.1.0
  with:
    path: ${{ env.CODEGRAPH_CORPUS_DIR }}/gohugoio-hugo
    key: corpus-gohugoio-hugo-${{ <pinned sha from manifest> }}
    # No restore-keys: (D-13) — a prefix match on a content-addressed SHA
    # key is wrong by definition.

- name: Fetch corpus (cache miss)
  if: steps.cache-corpus-hugo.outputs.cache-hit != 'true'
  run: task corpora:fetch -- gohugoio/hugo
```
`actions/cache` (the combined restore+save action, as opposed to the split `actions/cache/restore` + `actions/cache/save`) automatically saves the cache under `key` via its own post-job hook whenever the primary key did not hit [CITED: actions/cache README] — the explicit `if: cache-hit != 'true'` fetch step is therefore the ENTIRE mechanism FIXT-02 criterion 5 needs; no separate save step is required.

### Deriving the dense key set from `RankEdges` (D-04)

```go
// Illustrative — NOT existing code. Shows the pattern the planner should
// follow, built from the verified RankEdges var (rwr.go:21-31) and the
// verified sortedCounts precedent (render_status.go:84-98) that already
// filters-at-render-time rather than inside Status().
func denseEdgeCounts(sparse map[string]int64) []kindCount {
    keys := make([]string, 0, len(query.RankEdges))
    for k := range query.RankEdges {
        keys = append(keys, k)
    }
    sort.Strings(keys) // deterministic order, matching sortedCounts' own tiebreak discipline
    out := make([]kindCount, 0, len(keys))
    for _, k := range keys {
        out = append(out, kindCount{Key: k, Count: sparse[k]}) // 0 for a measured-absent kind — NOT omitted
    }
    return out
}
```

## State of the Art

Not applicable in the "old approach vs. new approach" sense this section usually captures — this phase is not adopting a newer version of an existing tool. The one relevant "state of the art" fact: `actions/cache` has moved from the v3/v4 line commonly seen in older tutorials to **v6.1.0** (Node 24 runtime) as of this research date [CITED, WebSearch cross-checked against `gh api`]. Any plan or example authored from training-data memory alone would likely propose `actions/cache@v3` or `@v4` — verify the live tag before pinning.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `overrides`/`type_of` yield reasoning for candidate corpora (guava's `Forwarding*` classes, nestjs's abstract-class lifecycle hooks, Newtonsoft.Json's `JsonConverter` subclasses producing real override/type_of edges) is **reasoned from extractor mechanics and general knowledge of these codebases' idioms, not measured by running the (not-yet-built) instrument against them** | Summary, Coverage risk reasoning throughout | If wrong, the phase's own measurement step (which this research cannot substitute for) will surface it — this is exactly what Phase 1 exists to determine empirically. Low risk to planning: the reasoning only informs *measurement order* (which candidates to try first), never the pass/fail bar itself. |
| A2 | `JamesNK/Newtonsoft.Json` and `serilog/serilog` are good C# candidates | §Candidate Corpora | Medium — license/size/language-mix facts were live-verified via `gh api`, but their actual `overrides`/`type_of` YIELD is unmeasured (see A1). If they measure poorly, D-17's bounded additional-candidate search should be exercised next: other reasonable MIT/Apache-2.0 C# candidates not yet checked include `dotnet/efcore` (MIT, `size_kb: 239378`, large), `restsharp/RestSharp` (Apache-2.0, `size_kb: 41439`) — both license-checked via `gh api` in this session but not deeply reasoned about override/type_of density. |
| A3 | GitHub's `uploadpack.allowReachableSHA1InWant` is enabled for all four shortlist repos and both C# candidates (all GitHub-hosted) | §Fetch Mechanism, Code Examples | Low — this is a GitHub-platform-wide default for public repos, not a per-repo opt-in, cross-confirmed via WebSearch; not verified by actually running the fetch command against each specific repo in this session (no write/clone access exercised here beyond `gh api` metadata calls). |
| A4 | The manifest format recommendation (JSON) and its exact repo path are non-binding recommendations, not verified requirements | §Standard Stack | None — explicitly Claude's Discretion per CONTEXT.md; presented as a recommendation with tradeoffs, not a claim of fact. |

**If this table is empty:** N/A — see entries above.

## Open Questions

1. **How is a wire-oracle transcript actually re-frozen (mechanically)?**
   - What we know: `TestFrozenTranscriptsMatch` (`test/wireoracle/oracle_test.go:112-152`) does a byte-for-byte comparison against the committed `.golden` file and simply fails (`t.Fatalf`) on mismatch — there is no `-update`/`UPDATE_GOLDEN`-style flag anywhere in `test/wireoracle/*.go` [VERIFIED: no such flag found via grep across the package]. `tools/transcriptfreeze` exists but is the D-03 anti-regeneration GUARD (`check:transcript-freeze` Taskfile task), not a freeze-writer — it reports when a transcript changed alongside `internal/mcp/*.go` without being caught by review, it does not generate transcripts.
   - What's unclear: the actual re-freeze mechanic is presumably "run the failing test, read its diff output (or capture the scenario's raw normalized bytes some other way), and manually `os.WriteFile`/copy the correct new bytes into the `.golden` file" — but no committed script does this today, unlike `testdata/golden/capture.sh` (which DOES exist for the older golden-parity fixtures).
   - Recommendation: the planner should budget a small task to either (a) confirm the manual copy-paste-from-test-failure-diff process is genuinely how prior re-freezes in this repo's history were done (check git blame/history on other `.golden` files for the actual mechanic used, e.g. the 05-02/05-04 re-freezes `test/wireoracle/COVERAGE-BASELINE.md` documents), or (b) write a tiny one-off capture helper (mirroring `gocapture/main.go`'s shape) if no such precedent exists. This is squarely inside Phase 1's stated ownership of the re-freeze (D-02).

2. **Exact CI wiring location for the new `actions/cache` step(s) — new job or existing `test` job?**
   - What we know: `ci.yml`'s `test` job is in `TestWorkflowRunBodiesInvokeTask`'s `inScopeJobs` fixture today, constraining any new `run:` step there to `task <target>`-only. `uses:`-only steps (the cache action itself) are unconstrained anywhere.
   - What's unclear: whether corpus fetch/cache belongs inside the existing `test` job (so `task test:golden`'s later self-skip removal in Phase 3 has corpora ready) or as its own new job. This is a genuine architecture decision, not purely mechanical — it interacts with Phase 3's FIXT-03 ("no golden test self-skips"), which is out of THIS phase's scope but shapes where the cache step should eventually live.
   - Recommendation: research does not resolve this — flag for the planner/discuss-phase as a design choice with a stated preference (a new job, since Phase 1's CI wiring is about the MEASUREMENT record, not yet about making `test:golden` corpus-dependent — that's explicitly Phase 3's job per the phase boundary).

3. **Does `Meta` need any change to accommodate `edgesByKind`, or is it purely a read-time derived value?**
   - What we know: `Meta.EdgeCount` is a stored, indexer-stamped aggregate [VERIFIED: internal/query/status.go:273, `meta.GetEdgeCount()`]. `edgesByKind` as designed here is NOT stored — it's derived fresh on every `Status()` call via a full scan, exactly like `nodesByKind` already is.
   - What's unclear: whether a future phase might want `edgesByKind` stored/cached in `Meta` for performance (avoiding the new unconditional full edge scan this phase introduces, per Pitfall/Pattern 1's cost note) — out of scope for THIS phase (D-01 scopes it to `status`, not to indexer/Meta changes) but worth a one-line note in the plan so a reviewer doesn't ask "why not just store it."
   - Recommendation: explicitly note in the plan that this is a deliberate, in-scope-only, read-time-derived design — not an oversight.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `git` | Corpus fetch (D-11), existing golden-parity tests | ✓ (already a hard project dependency) | host-installed | — |
| `jq` | Manifest parsing in Taskfile bash (recommended) | ✓ (already a hard project dependency, used extensively in Taskfile.yml) | host-installed | `go.yaml.in/yaml/v3` on the Go side if YAML manifest chosen instead |
| Network access to `github.com`/`codeload.github.com` | Corpus fetch, both locally and in CI | ✓ verified this session via live `gh api`/WebFetch calls | — | None — FIXT-02 explicitly requires network fetch; a network-sandboxed CI runner is an accepted, documented failure mode already precedented by `resolveColbymchenryCorpus`'s `t.Skip` on clone failure (existing test, not this phase's new drift guard, which per D-08 must NOT silently skip) |
| `gh` CLI (GitHub CLI) | Used in this research session for live license/SHA/language verification | ✓ | — | Not a hard requirement for the phase's own implementation — `gh api` was a research-time convenience; the manifest/fetch mechanism itself only needs `git` |
| `actions/cache` (GitHub-hosted runner feature) | CI corpus caching (D-13) | ✓ (native to GitHub-hosted Actions runners; this repo's CI already runs on `namespace-profile-linux-amd64-4x8` [VERIFIED: ci.yml:44]) | v6.1.0 (pin recommendation) | None needed — first-party platform feature |

**Missing dependencies with no fallback:** none identified.
**Missing dependencies with fallback:** none identified — every dependency this phase needs is already present in this repo's toolchain.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go standard `testing` package, `go test` |
| Config file | none — `Taskfile.yml` defines all suite groupings (`test:unit`, `test:golden`, `test:wireoracle`, etc.) |
| Quick run command | `go test ./internal/query/... ./internal/cli/...` (fast, no corpora needed) for the measurement-instrument half |
| Full suite command | `task test` (serial: `test:unit`, `test:golden`, `test:integration`, `test:wireoracle`, `test:daemon`, `test:race`) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| FIXT-01 | `edgesByKind`/`filesByLanguage` present and correct in `StatusResult`/`--json` | unit | `go test ./internal/query/... -run TestRenderStatus` | ✅ (`render_status_test.go` exists; new subtests needed, not a new file) |
| FIXT-01 | Dense key set is exactly the 9 `RANK_EDGES` kinds, key-set-equality asserted | unit | `go test ./internal/query/... -run TestEdgesByKindDense` (new test) | ❌ Wave 0 gap |
| FIXT-01 | Measurement record JSON satisfies the coverage claim (every kind clears its bar, every priority-4 language non-zero) | unit (drift guard) | `go test ./<wherever the guard lives> -run TestCorpusCoverageClaim` (new test) | ❌ Wave 0 gap |
| FIXT-01 | Wire-oracle `call-status.golden` re-frozen and matches live output | integration | `go test ./test/wireoracle/... -run TestFrozenTranscriptsMatch/call-status` | ✅ (existing scenario/test; golden file needs re-freezing, not new code) |
| FIXT-02 | Fetch target reproduces the same tree on a second clean run | integration/manual | `task corpora:fetch` run twice from clean, diff tree hash — see Open Discretion item "verified by what" | ❌ Wave 0 gap (no existing target) |
| FIXT-02 | CI cache miss falls through to fetch, never a skip | CI-only (cannot run locally in the same way) | Inspect a CI run's step summary / logs for the fetch step firing on a cold cache | ❌ Wave 0 gap — no CI-runnable local equivalent; this is inherently CI-observed behavior |

### Sampling Rate

- **Per task commit:** `go test ./internal/query/... ./internal/cli/...` (fast, covers the measurement-instrument half without needing corpora)
- **Per wave merge:** `task test:golden && task test:wireoracle` (covers the golden-parity "status" subtest and the re-frozen transcript)
- **Phase gate:** Full `task test` green, plus a real (not simulated) CI run showing the cache-miss-then-fetch and cache-hit-then-skip-fetch behaviors both firing correctly at least once each

### Wave 0 Gaps

- [ ] A test asserting dense-mode `edgesByKind`'s key set is exactly `query.RankEdges`'s key set (key-set equality, per D-04) — covers FIXT-01 criterion 2/3
- [ ] The drift-guard test itself (reads committed measurement JSON, asserts coverage claim) — covers D-07/D-08's positive-assertion requirement
- [ ] A Taskfile target `corpora:fetch` (or equivalent name) — none exists today
- [ ] The corpora manifest file itself — none exists today
- [ ] A "measurement record" generator (Go program analogous to `gocapture/main.go`, driving `codegraph status --json --all-kinds` [flag name TBD] against each fetched corpus and writing the keyed-by-`repo@SHA` JSON) — none exists today

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | no | This phase adds no auth surface |
| V3 Session Management | no | N/A |
| V4 Access Control | no | N/A |
| V5 Input Validation | **yes** | The corpora manifest is a repo-committed, review-gated file (not user/network input at fetch time) — but the Taskfile fetch target interpolates manifest-derived `repo`/`sha` values into `git remote add`/`git fetch` shell commands. Standard control: validate `sha` matches `^[0-9a-f]{40}$` and `repo` matches a strict `org/name` character allowlist before shell interpolation, to prevent a malformed or (future, if the manifest source ever becomes less trusted) adversarial manifest entry from achieving command injection into the Taskfile's bash `cmds:` block. |
| V6 Cryptography | no | No new crypto surface — the existing SHA-pinning-for-Actions convention is supply-chain integrity, not cryptography this phase implements |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Shell command injection via manifest-derived `repo`/`sha` fields interpolated into `git`/Taskfile bash | Tampering | Strict allowlist validation of both fields before interpolation (see V5 above); prefer `jq -r` extraction into shell variables with `"${var}"` quoting throughout, never unquoted interpolation |
| Supply-chain risk of indexing and (transiently) fetching third-party source into CI/dev environments | Tampering / Information Disclosure | Corpora are fetched at PINNED SHAs only (D-11) — never a floating branch/tag — so a compromised upstream commit landing after the pin is inert; this is the same rationale `pinnedWeftCommit`'s existing precedent already establishes for `weft-go`. The corpora are read-only inputs to the indexer, never executed. |
| GitHub Actions cache poisoning (a malicious PR from a fork populating a cache entry later restored by a trusted workflow run) | Tampering | Out of immediate scope for THIS phase's design (cache keys are content-addressed by `repo@sha`, and this repo's `ci.yml` triggers only on `pull_request`/`push` to `main`, not `pull_request_target` — no cross-trust-boundary cache write path is introduced here) — but worth a one-line note in the plan acknowledging `actions/cache`'s general cache-poisoning risk class exists and this repo already has documented advisory-only tracking of `pull_request_target` risk elsewhere (`.planning/STATE.md`'s "Advisory, unregistered surfaces" note) |
| Command/argument injection via the Go drift-guard reading manifest JSON | Tampering | Use `encoding/json`'s standard unmarshaling into a typed struct (never `exec.Command` built from raw manifest string concatenation) — matches this repo's own `TestTaskfileShapeHelpersFailLoudly`-style discipline of typed, validated parsing over raw string manipulation |

## Sources

### Primary (HIGH confidence — read directly this session)

- `internal/query/status.go` (full file) — `StatusResult` struct, `Status()` method, decision-table doc comment
- `internal/query/render_status.go` (full file) — `RenderStatusText`/`RenderStatusMarkdown`/`sortedCounts`
- `internal/cli/status.go` (full file) — `newStatusCmd`
- `internal/cli/present/status.go` (full file) — the fourth, package-local-duplicate TTY renderer
- `internal/query/rwr.go` (excerpt) — `RankEdges` canonical set
- `internal/query/rwr_test.go` (excerpt) — `TestRankEdges` key-set-equality precedent
- `internal/indexer/goextract/types.go` (excerpt) — `RefKind*`/`EdgeKind*` constant definitions and their doc comments explaining synthesis mechanics
- `internal/indexer/resolve.go` lines 440-540 — `synthesizeOverrides`, the exact mechanics of how `overrides` edges are derived
- `internal/indexer/goextract/goextract.go` lines 455-478, `goextract_test.go` lines 774-796 — `emitTypeOfRef`/`TestExtract_TypeOf`, proving `type_of` requires an explicit `var x T`, never inferred
- `internal/indexer/{csharpextract,javaextract,tsextract,pyextract}` (grep + targeted reads) — confirming all four non-Go extractors emit `RefKindEmbeds` for class inheritance
- `docs/LANGUAGE-CAPABILITY-MATRIX.md` (full file) — per-language extraction/resolution/dispatch/routing coverage and gaps
- `Taskfile.yml` (`test:golden`, `test:` wrapper, `check:cross`, `check:transcript-freeze` excerpts)
- `.github/workflows/ci.yml` (full `test` job + header comment)
- `.github/workflows/linux-cross-canary.yml` lines 1-90 — path-filtered sibling-workflow precedent
- `.github/actions/install-task/action.yml` (full file) — build-from-source-not-download convention
- `internal/upgrade/taskfile_shape_test.go` (full file, 2489 lines, read across two calls) — `TestWorkflowRunBodiesInvokeTask`, `TestTaskfileGatesFailLoud`, `TestTaskfileWrapperIsSerial`, `inScopeJobs`/`runBodyExceptions` fixtures
- `testdata/golden/README.md` (full file) — corpus provenance, capture conventions, volatile-fields policy
- `testdata/golden/golden_parity_test.go` (excerpts, lines 100-780) — `resolveWeftCorpus`, `resolveColbymchenryCorpus`, the `status` subtest, `findVolatileKeysExcept`
- `testdata/golden/gocapture/main.go` (excerpt) — precedent for a Go-driven measurement/capture tool
- `test/wireoracle/scenarios.go` (excerpts around lines 740-770, 1290-1350) — `call-status`/`resources-read-status` scenario definitions
- `test/wireoracle/oracle_test.go` (excerpts) — `TestFrozenTranscriptsMatch`, no auto-freeze mechanism
- `internal/mcp/resources.go` (excerpt), `internal/mcp/resources/status.md` (full file), `internal/mcp/tools.go` (excerpts) — confirming the doc-resource vs. tool-call distinction
- `internal/graphstore/keys.go` line 96 — `edgeKey` format, confirming no per-kind prefix scan shortcut exists
- `internal/query/expand.go` lines 256-283 — `buildExpandAdjacency`, the existing full-edge-scan precedent
- `internal/agents/opencode.go` lines 19-42 — existing XDG-resolution precedent and its "no OS special-case" discipline
- `go.mod` line 33 — `go.yaml.in/yaml/v3` already a direct dependency
- `.gitignore` line 38 — confirms no change needed for D-12's out-of-tree corpora
- `.planning/phases/01-corpus-selection-by-measurement/01-CONTEXT.md`, `.planning/REQUIREMENTS.md`, `.planning/STATE.md` — upstream phase context
- Live `gh api` calls (this session, 2026-08-14): `repos/{gohugoio/hugo,nestjs/nest,google/guava,apache/arrow}` (license/size/default_branch/language), `repos/.../languages`, `repos/.../license`, `repos/.../commits/{branch}` (current HEAD SHAs — NOT pins), `repos/JamesNK/Newtonsoft.Json`, `repos/serilog/serilog`, `repos/StackExchange/Dapper`, `repos/AutoMapper/AutoMapper`, `repos/dotnet/efcore`, `repos/restsharp/RestSharp` (license/size checks), `repos/actions/cache/releases/latest`, `repos/actions/cache/commits/v6.1.0`

### Secondary (MEDIUM confidence)

- WebFetch of `raw.githubusercontent.com/actions/cache/main/README.md` [CITED] — `actions/cache` YAML usage, `restore-keys` optionality, 10GB limit + LRU/7-day eviction, current major version
- WebSearch cross-check on `git fetch <sha>` / `uploadpack.allowReachableSHA1InWant` behavior on GitHub [CITED] — confirms D-11's fetch mechanism works without a fallback for GitHub-hosted repos

### Tertiary (LOW confidence)

- None used without a corroborating primary/secondary source.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies; every tool already precedented in this exact repo
- Architecture (measurement instrument extension): HIGH — verified against actual source across all 4 render surfaces plus the graphstore primitive
- Architecture (corpus fetch/cache): HIGH — verified against live GitHub API behavior and this repo's own CI conventions; the git-fetch-by-SHA mechanism is a well-documented, cross-checked git/GitHub feature
- Pitfalls: HIGH for the wire-oracle/render-surface findings (directly read and cross-checked source and golden files); MEDIUM for the coverage-risk reasoning about which candidate corpora will actually yield `overrides`/`type_of` (reasoned from extractor mechanics, not measured — flagged explicitly in the Assumptions Log)
- Candidate corpora facts (license/size/language mix): HIGH — every claim live-verified via `gh api` this session, not sourced from training data or README badges

**Research date:** 2026-08-14
**Valid until:** ~30 days for the repo-internal findings (stable unless the codebase changes underneath); the live `gh api` corpus facts (size/HEAD SHA) are point-in-time and WILL drift — the measurement phase itself re-verifies and pins its own SHAs, so this is expected and does not invalidate the research's structural findings
