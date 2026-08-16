---
phase: 05-process-ci-in-tree-sweep
plan: 05
subsystem: docs
tags: [comment-sweep, de-framing, code-01, agents, cli, githooks, gitmeta, watch, indexer, mcp, schema, claude-md]

requires:
  - phase: 05-process-ci-in-tree-sweep (plan 05-01, parallel)
    provides: codegraph migrate removed entirely (this plan does not touch internal/migrate, internal/cli/migrate*.go, internal/graphstore, README.md, go.mod/go.sum)
provides:
  - internal/agents (10 files) installer comments de-framed — parity-table/parity-oracle vocabulary removed, byte-format marker contract and past-tense history intact
  - internal/cli, internal/githooks, internal/gitmeta, internal/watch comment sweep — TS-parity/verbatim-port/golden-parity phrasing reworded to own-terms, zero runtime/assert/flag changes
  - internal/indexer remaining rows + internal/corpora stale-filename fix — golden_parity_test.go/parity_python_test.go/parity_tsjs_test.go citations re-pointed to the current behavioral_*_test.go names
  - internal/mcp prose-only stale-path fixes (archtest packages.Load runtime line untouched)
  - internal/schema graph.proto + graph.pb.go field-doc comments reworded in lockstep (6 fields, both files)
  - tools/bench/realcorpus/manifest.go stale golden citations re-pointed; discovered and fixed a deeper staleness (the cited testdata/golden mechanism no longer exists at all, not just renamed)
  - .claude/CLAUDE.md + .planning/PROJECT.md core-value/parity-command-surface/migration-reader rows rewritten on the project's own terms (D-10)
affects: [06-process-ci-in-tree-sweep-plan-06, ship-gate]

actuals:
  tokens: 20666
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Term-by-term sweep with recorded KEEP/REMAINING classification, never a bare-word regex — the naive-regex failure mode (breaking tsextract, de-listing a language) is explicitly avoided"
    - "Schema doc comments edited in lockstep across .proto and generated .pb.go in the same commit (no protoc pipeline in this repo)"

key-files:
  created: []
  modified:
    - internal/agents/instructions.go
    - internal/agents/gemini.go
    - internal/agents/hermes.go
    - internal/agents/hermes_test.go
    - internal/agents/kiro.go
    - internal/agents/registry.go
    - internal/agents/registry_test.go
    - internal/agents/toml.go
    - internal/agents/testhelpers_test.go
    - internal/agents/types.go
    - internal/cli/explore.go
    - internal/cli/daemon.go
    - internal/cli/status.go
    - internal/cli/githooks.go
    - internal/cli/init.go
    - internal/cli/serve_test.go
    - internal/cli/affected_test.go
    - internal/cli/install_test.go
    - internal/cli/present/status_test.go
    - internal/cli/man.go
    - internal/githooks/githooks.go
    - internal/githooks/githooks_test.go
    - internal/gitmeta/detect.go
    - internal/gitmeta/githooks.go
    - internal/gitmeta/notice.go
    - internal/gitmeta/worktree.go
    - internal/gitmeta/cache.go
    - internal/watch/policy.go
    - internal/indexer/nodeid/nodeid_test.go
    - internal/indexer/resolve.go
    - internal/indexer/routes/walk.go
    - internal/indexer/languages_python.go
    - internal/indexer/pyextract/types.go
    - internal/indexer/mainstream/phpextract/resolution_test.go
    - internal/corpora/record.go
    - internal/mcp/archtest/protocol_version_test.go
    - internal/mcp/markdown_test.go
    - internal/schema/graph.proto
    - internal/schema/graph.pb.go
    - tools/bench/realcorpus/manifest.go
    - .claude/CLAUDE.md
    - .planning/PROJECT.md

key-decisions:
  - "gitmeta/cache.go (not in the plan's files_modified) was swept too — found during task 2's package-wide recheck carrying the same 'TS's own cache key' framing family as its siblings in the same package"
  - "manifest.go's cited testdata/golden mechanism (resolveWeftCorpus / CODEGRAPH_WEFT_CORPUS / D-06a corpus table) was verified to no longer exist anywhere in the repo (removed by commit 3f1d4da, Phase 2 FIXT-04) — not a stale-filename rename, a fully orphaned citation. Re-pointed to the real current pin (testdata/golden/corpus/weft-go/) instead of a renamed-but-still-nonexistent name."
  - ".planning/PROJECT.md's Core Value / What-This-Is paragraphs were edited to match .claude/CLAUDE.md's rewrite — CLAUDE.md's own GSD marker (source:PROJECT.md) cites PROJECT.md as its source; leaving PROJECT.md stale would silently reintroduce the swept framing on any future regeneration, and PROJECT.md is itself the first file a future GSD session reads"
  - "8 additional CODE-01 framing hits were found outside this plan's declared file scope (internal/cli/{search,node,files,uninit,serve}.go, internal/cli/githooks_test.go, internal/mcp/tools.go, internal/agents/codex.go) — not fixed here (touching undeclared files risks conflicting with sibling-plan ownership and scope creep), logged individually to .planning/WINDOWS.md as open deviation entries for phase-gate visibility"

patterns-established:
  - "Positive-inventory verification (A3): every remaining TS|parity|port|original|upstream hit in the touched packages was read and classified KEEP (product truth/history) or REMAINING (fixed), never inferred from a zero-hit grep alone"

requirements-completed: [CODE-01]

coverage:
  - id: D1
    description: "internal/agents installer comments de-framed (parity-table/parity-oracle vocabulary removed); marker-byte contract and past-tense history intact"
    requirement: "CODE-01"
    verification:
      - kind: unit
        ref: "go test -count=1 ./internal/agents/..."
        status: pass
      - kind: other
        ref: "rg -c parity|Parity|'a TS install' internal/agents/ == 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "internal/cli, internal/githooks, internal/gitmeta, internal/watch comment sweep — TS-parity/verbatim-port/golden-parity phrasing reworded, zero runtime/assert/flag changes"
    requirement: "CODE-01"
    verification:
      - kind: unit
        ref: "go test -count=1 ./internal/cli/... ./internal/githooks/... ./internal/gitmeta/... ./internal/watch/..."
        status: pass
      - kind: other
        ref: "rg -c 'ported verbatim from TS'|'TS parity'|'TS-parity'|'Golden parity'|'for parity'|'verbatim port of' over the 4 packages == 0"
        status: pass
    human_judgment: false
  - id: D3
    description: "internal/indexer remaining rows, internal/corpora, internal/mcp prose, internal/schema proto/pb.go lockstep, tools/bench/realcorpus/manifest.go, .claude/CLAUDE.md core-value all de-framed / stale citations re-pointed"
    requirement: "CODE-01"
    verification:
      - kind: unit
        ref: "go test -count=1 ./internal/indexer/... ./internal/mcp/... ./internal/schema/... ./tools/bench/realcorpus/... ./internal/corpora/..."
        status: pass
      - kind: other
        ref: "rg -c Go-parity|golden-parity|parity_tsjs_test|parity_ts_test over internal/indexer(-capability) internal/corpora internal/mcp internal/schema == 0; rg -c golden_parity_test.go|'the original TS CodeGraph' tools/bench/realcorpus/manifest.go == 0; rg -c 'uninstall TS CodeGraph'|'works the same'|'migrate their indexes'|'parity command surface'|'One-way migration...' .claude/CLAUDE.md == 0"
        status: pass
    human_judgment: false

duration: 65min
completed: 2026-08-16
status: complete
---

# Phase 5 Plan 05: In-tree comment sweep (agents/cli/githooks/gitmeta/watch/indexer/mcp/schema/manifest/CLAUDE.md) Summary

**Swept ~90 comparison-framing comment rows across 10 packages term-by-term, re-pointed 9 stale golden-filename citations to their current names, kept the schema `.proto`/`.pb.go` doc pair in lockstep, and rewrote `.claude/CLAUDE.md`'s + `.planning/PROJECT.md`'s core-value paragraph off the "uninstall TS CodeGraph" framing — with zero runtime, assertion, flag, or marker-byte changes anywhere.**

## Performance

- **Duration:** ~65 min
- **Completed:** 2026-08-16T00:14:14Z
- **Tasks:** 3
- **Files modified:** 41

## Accomplishments

- `internal/agents` (10 files): de-framed the "Corrected Per-Agent Parity Table" heading, the "TS parity oracle" vocabulary family, and the cross-implementation marker-byte contract prose — while leaving the actual marker constant bytes (`codegraphSectionStart`/`End`) byte-identical and keeping the past-tense `#529/#704` history row untouched (D-02 KEEP).
- `internal/cli` + `internal/githooks` + `internal/gitmeta` + `internal/watch`: reworded ~50 rows across 18 files — package doc comments, WSL2 mount-detection rationale, git-hook marker semantics, disabled-watch message provenance — without touching any string literal a test asserts against, any regex pattern, or any env-var/flag name.
- `internal/indexer` + `internal/corpora` + `internal/mcp` + `internal/schema`: re-pointed every `golden_parity_test.go`/`parity_python_test.go`/`parity_tsjs_test.go` citation this plan owns to the current `behavioral_*_test.go` names; reworded all 6 "Additive Go-parity field" doc comments to "Additive schema field" in **both** `graph.proto` and `graph.pb.go` in the same commit (no protoc pipeline in this repo — a one-sided edit would have created a doc/registry mismatch); left the mcp archtest `packages.Load` runtime path line untouched.
- `tools/bench/realcorpus/manifest.go`: re-pointed stale golden citations and, on investigation, found the cited mechanism (`golden_parity_test.go`'s `resolveWeftCorpus`, the `CODEGRAPH_WEFT_CORPUS` convention, the README's "D-06a corpus table") no longer exists anywhere in the repo — it was deleted by commit `3f1d4da` (Phase 2 FIXT-04) and the manifest comment was simply never updated. Re-pointed to the real current pin location (`testdata/golden/corpus/weft-go/`) instead of a renamed-but-still-nonexistent name.
- `.claude/CLAUDE.md` (D-10) + `.planning/PROJECT.md`: rewrote the "Core Value" paragraph and the opening project description from "An agent user can uninstall TypeScript CodeGraph, install the Go binary, migrate their indexes..." to a description of the project on its own terms; dropped the "parity command surface (…, migrate)" phrase and every `modernc.org/sqlite`/migration-reader tech-stack row (6 locations: Core Technologies table, Alternatives Considered, What NOT to Use, Version Compatibility, Sources); dropped the dead "One-way migration from its SQLite format still binds" sentence (the migration contract no longer exists post-CODE-03); kept the TS CodeGraph v1.3.x compatibility-baseline constraint (recorded KEEP below) and every TypeScript-as-indexed-language reference.

## Task Commits

1. **Task 1: internal/agents de-framing** — `0c36853` (docs)
2. **Task 2: cli/githooks/gitmeta/watch comment sweep** — `b4376ee` (docs)
3. **Task 3: indexer/corpora/mcp/schema/manifest/CLAUDE.md sweep** — `435432a` (docs)

**Plan metadata:** _pending — this SUMMARY's own commit_

_Note: this is a comment/prose-only sweep — no test-driven-development gates apply._

## Files Created/Modified

See `key-files.modified` in frontmatter (41 files across 10 packages plus 2 root-level docs).

## Term-by-Term Retained-Use Classification (A3 inventory, this plan's scope)

Every hit in the packages this plan touches was read in context and classified. **KEEP** = product truth / past-tense history / in-repo restatement, left unchanged. **FIXED** = comparison framing or stale citation, reworded.

### KEEP (recorded reasons)

| Location | Text | Reason |
|---|---|---|
| `internal/agents/instructions.go:19` | "the old full playbook TS removed in #529/#704" | Past-tense history (D-02) — describes a historical PR event, not a live comparison claim |
| `internal/gitmeta/detect.go:77` | "CONTEXT.md D-02 / TS issues #1031, #1033" | Historical bug-tracker citation (issue numbers), not a comparison-baseline claim — informative provenance for future maintainers |
| `internal/gitmeta/worktree.go:27` (post-edit) | "tracked as issue #1139" | Same class — historical issue-number citation, TS name dropped, issue number kept |
| `internal/corpora/record.go:100/106`, `internal/indexer/languages_python.go`, `tools/bench/realcorpus/manifest.go:106/114` | "TS/JS" as an indexed-language pair | Product surface (D-02) — `tsextract`, the language registry, and TS-as-indexed-language are untouched by the framing sweep |
| `internal/upgrade/taskfile_shape_test.go` (not touched by this plan) | `TestGoreleaserPinParity` etc. | Out of this plan's file scope; per research §4.6, these are in-repo restatement-agreement guards, not TS comparisons |
| `.claude/CLAUDE.md` / `.planning/PROJECT.md`: "TS CodeGraph v1.3.x is a **functionality baseline, not a binding constraint** (maintainer ruling 2026-08-09)..." | Compatibility constraint sentence | **KEEP, not swept** — this is live internal engineering guidance (how much weight TS behavior carries in design decisions), distinct from the Core Value marketing paragraph that WAS swept. Recorded per task 3's discretion clause. |
| `internal/agents/opencode.go:39/233-234`, `internal/agents/claude.go:22` | "TS issue #535" / "TS issue #207" | Out of this plan's file scope (not in task 1's 10-file list); appear to be the same historical-citation KEEP class as detect.go's, but not verified line-by-line since these files were never in scope — flagged, not touched |

### FIXED (this plan)

~80 rows across `internal/agents` (10), `internal/cli`+`internal/githooks`+`internal/gitmeta`+`internal/watch` (18 files, ~50 rows), `internal/indexer`+`internal/corpora` (7 rows), `internal/mcp` (3 rows), `internal/schema` (6 field-doc pairs = 12 rows), `tools/bench/realcorpus/manifest.go` (5 rows), `.claude/CLAUDE.md`+`.planning/PROJECT.md` (9 rows). All are `git diff`-visible in the three task commits above; each was hand-reworded per the plan's `<action>` guidance, never by regex.

### REMAINING — found but NOT fixed here (out of this plan's declared scope)

Logged individually to `.planning/WINDOWS.md` as open `deviation` entries (phase 05) so they're visible at the ship gate:

| File:line | Text | Why not fixed here |
|---|---|---|
| `internal/cli/search.go:55` | "no TS precedent" | Not in task 2's declared `<files>` list |
| `internal/cli/node.go:60` | "no TS precedent for this CLI placement" | Not in task 2's declared `<files>` list |
| `internal/cli/files.go:18,90` | "matches TS files --filter" | Not in task 2's declared `<files>` list |
| `internal/cli/uninit.go:56` | "mirrors TS" | Not in task 2's declared `<files>` list |
| `internal/cli/serve.go:137` | "D-12/D-13 verbatim TS disabled message" | Not in task 2's declared `<files>` list (its test companion `serve_test.go` WAS in scope and IS fixed) |
| `internal/cli/githooks_test.go:50` | "verbatim TS sync/git-hooks.js begin marker" | Distinct file from `internal/githooks/githooks_test.go` (which was in scope and fixed) — not declared |
| `internal/mcp/tools.go:369,528` | "Go-vs-TS divergence...", "mirroring TS's" | Not in task 3's declared `<files>` list |
| `internal/agents/codex.go:14,17` | "TS integrates with", "mirrors TS's own toml.ts" | Task 1's `<files>` list named 10 specific agents files; `codex.go`/`opencode.go`/`claude.go` were not among them |

These are genuine `research/05-RESEARCH.md` census gaps (the phase's own census table did not enumerate them), not files this plan silently declined — each was discovered only via the phase-wide A3 grep this task's own `<verification>` block requires. Touching them here would edit files outside this plan's declared `files_modified`, which is exactly the "concurrent editors collide" risk the phase's wave-1/wave-2 file-ownership discipline exists to prevent.

## Decisions Made

1. **`gitmeta/cache.go` swept even though absent from `files_modified`.** Found during task 2's package-wide recheck of `internal/gitmeta` carrying the same "TS's own cache key" framing as its three siblings in the same package (which WERE declared). Fixing it kept the package internally consistent; documented as a deviation, not left half-swept.
2. **`.planning/PROJECT.md` edited alongside `.claude/CLAUDE.md`.** CLAUDE.md's own `<!-- GSD:project-start source:PROJECT.md -->` marker names PROJECT.md as its source. `research/STACK.md` (the other marker's cited source) has since been repurposed for a later milestone's research and no longer contains any of the reworded content — confirming CLAUDE.md's tech-stack section is frozen prose with no live regeneration path, so editing it directly was sufficient there. PROJECT.md's Core Value paragraph, by contrast, is still byte-identical to CLAUDE.md's pre-edit text and IS the live, always-read GSD project doc — leaving it stale would have silently reintroduced the exact sentence the maintainer ruled needs to go, the first time any future GSD session reads it.
3. **`tools/bench/realcorpus/manifest.go`'s citation was more than a stale filename.** The plan classified this as "S = stale filename," but the actual referenced mechanism (a `resolveWeftCorpus` function, a `CODEGRAPH_WEFT_CORPUS` env-var convention read by the golden suite, a "D-06a corpus table" in `testdata/golden/README.md`) does not exist under ANY name in the current tree — verified via `git log -S` (deleted in `3f1d4da`, Phase 2 FIXT-04) and a repo-wide grep. Re-pointing the filename alone would have produced a comment that is still factually false. Reworded to describe the manifest's own actual behavior instead.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `internal/gitmeta/cache.go` framing row, not in files_modified**
- **Found during:** Task 2
- **Issue:** `cache.go:22` carried "keyed on TS's own cache key" — same framing family as `detect.go`/`notice.go`/`worktree.go`/`githooks.go`, all of which were declared, but `cache.go` itself was not.
- **Fix:** Reworded to "keyed on the documented cache key".
- **Files modified:** `internal/gitmeta/cache.go`
- **Committed in:** `b4376ee` (Task 2 commit)

**2. [Rule 1 - Bug] `tools/bench/realcorpus/manifest.go` cited a mechanism that no longer exists**
- **Found during:** Task 3
- **Issue:** Comments claimed the manifest "mirrors testdata/golden's own CODEGRAPH_WEFT_CORPUS convention" and reused "testdata/golden/golden_parity_test.go's resolveWeftCorpus" — neither exists anywhere in the current tree (deleted by `3f1d4da`).
- **Fix:** Reworded to describe the manifest's own EnvVar/SiblingDir resolution logic standalone, and re-pointed the "same pin" claim to the real current location (`testdata/golden/corpus/weft-go/`).
- **Files modified:** `tools/bench/realcorpus/manifest.go`
- **Verification:** `go test -count=1 ./tools/bench/realcorpus/...` passes; `rg` confirms zero remaining `golden_parity_test.go`/`resolveWeftCorpus`/`CODEGRAPH_WEFT_CORPUS`-in-testdata/golden citations
- **Committed in:** `435432a` (Task 3 commit)

**3. [Rule 2 - Missing critical] `.planning/PROJECT.md` left unsynced with `.claude/CLAUDE.md`'s D-10 rewrite**
- **Found during:** Task 3
- **Issue:** `.claude/CLAUDE.md`'s Core Value paragraph carries a `source:PROJECT.md` GSD marker. `.planning/PROJECT.md` had the byte-identical competitive-framing sentence. Fixing only the derived copy would leave the canonical, always-read source document contradicting the fix.
- **Fix:** Mirrored the same Core Value / What-This-Is rewrite into `.planning/PROJECT.md`.
- **Files modified:** `.planning/PROJECT.md`
- **Committed in:** `435432a` (Task 3 commit)

**4. [Rule 1 - Bug] Two residual task-1 framing rows found during the task-3 A3 inventory**
- **Found during:** Task 3 (phase-wide retained-use scan)
- **Issue:** `internal/agents/toml.go:10` still said "per TS's own file header"; `internal/agents/registry_test.go:185/188` still said "want the exact TS marker text" — both survived task 1's narrower per-file greps.
- **Fix:** Reworded both to drop the TS reference while keeping the technical substance (the byte-format contract, the quoted size-tradeoff rationale).
- **Files modified:** `internal/agents/toml.go`, `internal/agents/registry_test.go`
- **Committed in:** `435432a` (Task 3 commit)

---

**Total deviations:** 4 auto-fixed (2 Rule 1 bugs beyond plan scope but within touched packages, 1 Rule 1 stale-mechanism discovery, 1 Rule 2 missing-sync fix)
**Impact on plan:** All four are within the spirit of CODE-01's "term-by-term, nothing left referencing" mandate; none change runtime behavior, test assertions, flags, or marker bytes. The 8 out-of-scope hits (see REMAINING table above) were deliberately NOT auto-fixed since they live in files this plan does not own — logged to `.planning/WINDOWS.md` instead.

## Issues Encountered

- `go test -count=1 ./...` (full suite) surfaced one pre-existing flaky failure: `TestDaemonSharedWriter` in `internal/daemon` (store-lock contention timeout under load). Re-ran in isolation — passed cleanly in 0.5s. `internal/daemon` was not touched by this plan (not in `files_modified`, no `daemon` package edits); this is unrelated pre-existing test flakiness under this environment's load, not a regression introduced here. Not auto-fixed per the SCOPE BOUNDARY rule (out-of-scope failures in unrelated files).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plan 05-06 (corpus/behavioral/src re-freeze) and plan 05-04 (internal/query + indexer/capability sweep, wave 2) are unaffected by this plan's edits — no file overlap.
- The 8 logged `.planning/WINDOWS.md` deviations (out-of-scope CODE-01 census gaps) should be triaged before `/gsd-ship` — either folded into a follow-up plan or explicitly waived.
- `go build ./...` and `go test -count=1 ./...` are green (modulo the pre-existing daemon flake noted above) as of this plan's final commit.

---
*Phase: 05-process-ci-in-tree-sweep*
*Completed: 2026-08-16*
