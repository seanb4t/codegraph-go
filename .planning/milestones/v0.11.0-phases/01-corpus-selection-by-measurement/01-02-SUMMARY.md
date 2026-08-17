---
phase: 01-corpus-selection-by-measurement
plan: 02
subsystem: query,cli
tags: [go, status, edgesByKind, dense-mode, rank-edges, tty-parity]

# Dependency graph
requires:
  - "01-01 — query.StatusResult.EdgesByKind (sparse tally) and un-suppressed FilesByLanguage, both live in Engine.Status"
provides:
  - "query.DenseEdgesByKind — derives the dense edgesByKind key set from query.RankEdges, never restates it; preserves unranked kinds; never mutates its input"
  - "The Edges by Kind: / **Edges by Kind:** section on all FOUR render surfaces: RenderStatusText (piped), present.RenderStatus (TTY), MarshalStatusJSON (--json, via the same StatusResult), RenderStatusMarkdown (MCP)"
  - "codegraph status --all-kinds — CLI-only opt-in to the dense form, applied once before any output branch"
affects: [01-03, 01-06]

# Actuals (#2632)
actuals:
  tokens: 7818
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Derive-a-key-set-from-a-canonical-var, never restate: DenseEdgesByKind ranges over query.RankEdges for its key set instead of hand-listing the 9 kind strings a second time, following TestRankEdges's own two-directional membership-check discipline (rwr_test.go)."
    - "Density-at-one-point, renderer-agnostic: neither query.RenderStatusText/RenderStatusMarkdown nor present.RenderStatus takes a density parameter — both always call the new unfiltered edgeCounts helper on whatever r.EdgesByKind they are handed. internal/cli/status.go is the single place that decides sparse vs dense, by conditionally reassigning result.EdgesByKind before any output branch — so the MCP surface (which never calls DenseEdgesByKind) stays sparse by construction, not by a branch someone could get wrong."
    - "edgeCounts as sortedCounts's zero-preserving sibling, duplicated package-locally in internal/cli/present exactly as sortedCounts/formatNumber/formatMB/kindCount already are, per that package's own documented duplication precedent."

key-files:
  created: []
  modified:
    - internal/query/render_status.go
    - internal/query/render_status_test.go
    - internal/cli/status.go
    - internal/cli/status_cli_test.go
    - internal/cli/present/status.go
    - internal/cli/present/status_test.go
    - docs/FLAG-PARITY.md

key-decisions:
  - "DenseEdgesByKind computes the union of query.RankEdges's 9 members and r.EdgesByKind's actual keys, not the intersection — an edge kind outside RankEdges (e.g. \"contains\", which resolve.go emits literally) with a real positive count is preserved in the dense output rather than dropped, matching the plan's explicit 'never silently drop an unranked kind' requirement."
  - "--all-kinds over --verbose (naming, Claude's Discretion per CONTEXT.md): the chosen name states what changes (the edgesByKind key set) rather than implying broader command-wide chattiness."
  - "nodesByKind dense treatment stays explicitly out of scope (recorded in status.go's doc comment as a deliberate deferral, not an inconsistency) — matches CONTEXT.md's Deferred Ideas list."
  - "Doc comments that would have restated a helper's own name in a way that inflated its `grep -c` count (e.g. mentioning DenseEdgesByKind or the literal string \"Edges by Kind:\" a second time in prose) were reworded to keep each acceptance-criteria grep at exactly 1 occurrence — the source of truth is the single call site, not the number of times it's described in comments."

patterns-established:
  - "A cross-renderer equality test (TestRenderStatus_MatchesPipedSectionOrder) that derives BOTH the TTY and piped section-header sequences from live renderer output and asserts they're equal, rather than two independently hand-maintained expectation lists that could silently drift apart. Confirmed to actually catch the failure mode by a real RED/GREEN cycle: removing the new writeBreakdownText call from present.RenderStatus turned the test red, and reverting it byte-clean turned it green again."

requirements-completed: [FIXT-01]

coverage:
  - id: D4-key-set
    description: "The dense edgesByKind key set is DERIVED from query.RankEdges and never hand-listed — a key-set-equality test asserts the two sets are identical in both directions (D-04)."
    requirement: FIXT-01
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestDenseEdgesByKindKeySetEqualsRankEdges"
        status: pass
      - kind: source
        ref: "grep -c 'func DenseEdgesByKind' internal/query/render_status.go == 1; grep -c 'range RankEdges' internal/query/render_status.go >= 1; grep -vE '^\\s*//' internal/query/render_status.go | grep -c type_of == 0"
        status: pass
    human_judgment: false
  - id: D4-absent-vs-zero
    description: "Dense mode carries an explicit 0 for a measured-zero RankEdges kind; sparse mode omits an unmeasured kind entirely — absent and measured-zero are un-confusable."
    requirement: FIXT-01
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestDenseEdgesByKindExplicitZero"
        status: pass
      - kind: unit
        ref: "internal/query/render_status_test.go#TestDenseEdgesByKindPreservesUnrankedKind"
        status: pass
      - kind: unit
        ref: "internal/query/render_status_test.go#TestDenseEdgesByKindDoesNotMutateInput"
        status: pass
      - kind: e2e
        ref: "codegraph status -p . --all-kinds human output contains an overrides row reading 0, confirmed via internal/cli/status_cli_test.go#TestStatusCmdAllKindsFlag and directly against a real built binary + this repo's own index"
        status: pass
    human_judgment: false
  - id: D-ordering
    description: "edgesByKind render order is count-descending / key-ascending-tiebreak and byte-identical across repeated renders, including an all-zero dense map."
    requirement: FIXT-01
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestRenderStatusEdgeOrderIsDeterministic"
        status: pass
    human_judgment: false
  - id: D-four-surfaces
    description: "All four render surfaces carry the new section: RenderStatusText (piped), present.RenderStatus (TTY), the JSON contract, and RenderStatusMarkdown (MCP)."
    requirement: FIXT-01
    verification:
      - kind: unit
        ref: "internal/query/render_status_test.go#TestRenderStatusTextEdgesByKindSection, #TestRenderStatusMarkdownEdgesByKindSection"
        status: pass
      - kind: unit
        ref: "internal/cli/present/status_test.go#TestRenderStatus_EdgesByKindSection, #TestRenderStatus_MatchesPipedSectionOrder"
        status: pass
      - kind: unit
        ref: "internal/cli/status_cli_test.go#TestStatusCmdEdgesByKindSection, #TestStatusCmdAllKindsFlag, #TestStatusCmdJSONDense"
        status: pass
      - kind: e2e
        ref: "test/wireoracle TestFrozenTranscriptsMatch/call-status now fails on the new **Edges by Kind:** section appearing in live MCP markdown output — confirming the MCP surface (RenderStatusMarkdown) does carry the section. This failure is EXPECTED and is Plan 01-03's subject to resolve (the golden re-freeze), not a defect of this plan."
        status: pass
    human_judgment: false
  - id: D-05-mcp-stays-sparse
    description: "codegraph://status and codegraph_status stay SPARSE — density is CLI-only, reachable only via --all-kinds."
    requirement: FIXT-01
    verification:
      - kind: source
        ref: "git diff --name-only across all 3 task commits does not name internal/mcp/tools.go"
        status: pass
      - kind: unit
        ref: "internal/cli/status_cli_test.go#TestStatusCmdJSONDense (sparse subtest: every decoded edgesByKind value > 0 with no flag)"
        status: pass
    human_judgment: false

# Metrics
duration: ~50min
completed: 2026-08-14
status: complete
---

# Phase 1 Plan 2: Dense Edge-Kind Mode + Edges by Kind on All Four Render Surfaces Summary

**`codegraph status` gained an opt-in `--all-kinds` flag and an `Edges by Kind:` section on all four render surfaces — piped text, TTY, `--json`, and MCP markdown — with the dense key set derived from `query.RankEdges` in both directions so a future 10th ranked edge kind is measured automatically instead of silently dropped.**

## Performance

- **Duration:** ~50 min
- **Tasks:** 3
- **Files modified:** 7 (6 planned + 1 deviation fix)

## Accomplishments

- `query.DenseEdgesByKind(r StatusResult) map[string]int64` builds the dense form by ranging over `RankEdges` for its key set (never hand-listing the 9 kind strings), assigning `0` for every unmeasured member, and copying across any kind outside `RankEdges` (e.g. `contains`) unchanged — the result is the union of `RankEdges` and the sparse tally's keys, so an unranked kind is never silently dropped. Returns a new map; the caller's `r.EdgesByKind` is never mutated.
- `edgeCounts`, a sibling to the existing `sortedCounts`, shares its exact count-descending/key-ascending-tiebreak comparison but does **not** filter zero-valued entries — the filter is correct for `NodesByKind`/`FilesByLanguage` and wrong for a dense `EdgesByKind`.
- Both `internal/query` renderers (`RenderStatusText`, `RenderStatusMarkdown`) now call `edgeCounts(r.EdgesByKind)` unconditionally between their `Nodes by Kind:` and `Files by Language:`/`Languages:` sections. Density is **not** a renderer parameter — the decision of sparse vs. dense is made entirely upstream, by whichever map the caller hands the renderer.
- `codegraph status` gained `--all-kinds` (no short flag), applied exactly once — `result.EdgesByKind = query.DenseEdgesByKind(result)` — immediately after `Engine.Status` and before any of the three output branches (human, `--json`, TTY-via-`present`), so all three agree without a density parameter reaching any renderer. `internal/mcp/tools.go` is untouched by this plan, confirmed by `git diff --name-only` across all three task commits — the argument-less MCP `codegraph_status` tool call stays sparse by construction (D-05).
- `internal/cli/present/status.go` — RESEARCH's fourth render surface, a package-local **duplicate** renderer used only on a real TTY — gained a byte-for-byte-duplicated `edgeCounts` (matching this file's existing precedent of duplicating `kindCount`/`formatNumber`/`formatMB`/`sortedCounts`) and its own `Edges by Kind:` call site at the same position as the piped renderer's. `TestRenderStatus_MatchesPipedSectionOrder` renders one fixture through both `present.RenderStatus` and `query.RenderStatusText`, extracts each renderer's live section-header sequence, and asserts they're equal — a cross-renderer check rather than two hand-maintained lists that could drift apart silently. Confirmed to actually work as a gate: temporarily removing the new `writeBreakdownText` call turned this test RED, and reverting byte-clean turned it GREEN again (both states directly observed, per the plan's acceptance criterion).
- `docs/FLAG-PARITY.md` gained a row documenting `--all-kinds` as `Go-only` net-new surface — a Rule-3 blocking fix, not part of the planned task list; see Deviations.

## Task Commits

Each task was committed atomically:

1. **Task 1: Derive the dense key set from RankEdges and render the section in both query renderers** — `c7ca3c2` (feat)
2. **Task 2: Add the `--all-kinds` opt-in flag to `codegraph status`** — `e4c4070` (feat)
3. **Task 3: Close the fourth render surface — the TTY twin in `internal/cli/present`** — `c3a77e6` (feat, also carries the FLAG-PARITY.md fix)

## Files Created/Modified

- `internal/query/render_status.go` — `DenseEdgesByKind`, `edgeCounts`, both renderers' new `Edges by Kind:`/`**Edges by Kind:**` call sites, updated header/section-order doc comments
- `internal/query/render_status_test.go` — 7 new tests (key-set equality, explicit zero, unranked-kind preservation, no-mutation, both renderers' section presence/position, deterministic byte-identical repeated rendering); extended 2 existing section-list tests
- `internal/cli/status.go` — `--all-kinds` flag registration and its single application point; updated doc comment (also records the deliberate `nodesByKind` dense-treatment deferral and the `--all-kinds` vs `--verbose` naming rationale)
- `internal/cli/status_cli_test.go` — 3 new tests driving the real CLI entry point (sparse section has no zero rows; `--all-kinds` section has all 9 kinds incl. an explicit `overrides: 0`; `--json`/`--json --all-kinds` decoded key-set equality against `query.RankEdges`, member by member, both directions)
- `internal/cli/present/status.go` — package-local duplicate `edgeCounts`, new call site, updated `RenderStatus` doc comment
- `internal/cli/present/status_test.go` — fixture gained an asymmetric `EdgesByKind` map with an explicit zero; 2 new tests (section presence incl. zero row; cross-renderer section-order equality, RED-verified)
- `docs/FLAG-PARITY.md` — added the `--all-kinds` row under `## status` (deviation fix)

## Decisions Made

- **Union, not intersection, for unranked kinds** — `DenseEdgesByKind` preserves a kind outside `RankEdges` (e.g. `contains`) if it carries a real count, matching the plan's explicit requirement that an unranked kind is never silently dropped.
- **`--all-kinds` over `--verbose`** (Claude's Discretion, CONTEXT.md) — the name states what changes (the `edgesByKind` key set) rather than implying broader command-wide verbosity.
- **`nodesByKind` dense treatment stays deferred** — recorded explicitly in `status.go`'s doc comment as this milestone's known scope boundary, not an oversight.
- **Doc-comment wording avoided restating a symbol/section name a second time** where doing so would have inflated an acceptance-criteria `grep -c` count above 1 (e.g. `DenseEdgesByKind` mentioned once in `internal/cli/status.go`, `Edges by Kind:` mentioned once in `internal/cli/present/status.go`) — the grep-count acceptance criteria are counting *implementation* occurrences (one call site = one true source of behavior), and prose describing that call site doesn't need to restate the identical string to be clear.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] `docs/FLAG-PARITY.md` drift guard failed after adding `--all-kinds`**
- **Found during:** Task 3's full-suite verification run (`go test ./internal/cli/...`)
- **Issue:** `TestFlagParityDocCoversRegisteredFlags` asserts every registered cobra flag is documented in `docs/FLAG-PARITY.md`. Task 2's new `--all-kinds` flag on `codegraph status` was registered but undocumented, failing this pre-existing guard.
- **Fix:** Added a `--all-kinds` row to the `## status` table, marked `Go-only` (net-new — TS has no `edgesByKind` measurement to diverge from).
- **Files modified:** `docs/FLAG-PARITY.md`
- **Verification:** `go test -count=1 ./internal/cli/...` passes; the guard's own check (`grep`-derived cobra-flag enumeration vs. the doc table) is exact-equality, not a lower bound.
- **Committed in:** `c3a77e6` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (Rule 3)
**Impact on plan:** Directly caused by Task 2's own flag registration (same causal chain as adding the flag, not a scope expansion). No other files needed adjustment.

## Issues Encountered

- Two doc-comment edits initially over-restated a symbol's own name (`DenseEdgesByKind` mentioned twice in `internal/cli/status.go`; `Edges by Kind:` mentioned twice in `internal/cli/present/status.go`), which pushed two acceptance-criteria `grep -c ... == 1` checks to `2`. Both were caught immediately by running the greps before moving on, and fixed by rewording the prose to describe the call site without restating its literal name a second time — no functional change, comment wording only.
- The sandboxed Bash tool rejected `cd <dir> && git ...` compound commands as "too complex to verify... stays inside the worktree" (same friction 01-01's SUMMARY recorded) — worked around by relying on the tool's own default cwd (already the worktree root) rather than an explicit `cd`.
- `test/wireoracle`'s `TestFrozenTranscriptsMatch/call-status` fails as expected after this plan (the new `**Edges by Kind:**` bullets appear in live MCP markdown output but not in the frozen golden). This is the plan's own `<verification>` block's stated expectation, explicitly Plan 01-03's subject to resolve via a reviewed re-freeze — not papered over, not treated as a defect here.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- All four render surfaces (piped, TTY, `--json`, MCP markdown) now carry the dense-capable `Edges by Kind:` section, with `--all-kinds` as the sole CLI-only opt-in and the MCP surface staying sparse by construction (D-05, no `internal/mcp` change).
- Plan 01-03 can now proceed to re-freeze `testdata/wireoracle/transcripts/call-status.golden` against this plan's live `RenderStatusMarkdown` output — the failure this plan intentionally leaves behind is exactly what that plan exists to resolve, in its own reviewed diff (per this milestone's "one reviewed diff, one named cause" discipline).
- Plan 01-06's corpus measurement run can now use `codegraph status --json --all-kinds` to get the dense, key-set-complete tally FIXT-01's coverage claim depends on.
- No blockers.

## Self-Check: PASSED

- FOUND: `internal/query/render_status.go` (modified)
- FOUND: `internal/cli/status.go` (modified)
- FOUND: `internal/cli/present/status.go` (modified)
- FOUND: `docs/FLAG-PARITY.md` (modified)
- FOUND: commit `c7ca3c2` (Task 1)
- FOUND: commit `e4c4070` (Task 2)
- FOUND: commit `c3a77e6` (Task 3)

---
*Phase: 01-corpus-selection-by-measurement*
*Completed: 2026-08-14*
