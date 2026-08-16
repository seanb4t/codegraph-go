---
status: gaps_found
verified_at: 2026-08-15T21:30:00Z
phase: "05"
gaps:
  - truth: "Comments across internal/, tools/ and test/ carry no comparison framing (CODE-01)"
    status: failed
    reason: "A multiline (rg -U) census independently reproduces 10 file:line locations still carrying live TS-comparison framing after all 6 plans executed. Nine are outside the file scope any Phase 5 plan declared (WINDOWS 4-11, 14); one is an in-scope miss inside a file 05-04 explicitly listed as modified (WINDOWS 15), missed because the flagship censuses used line-based rg that cannot see a phrase wrapped across a comment line break."
    artifacts:
      - path: "internal/cli/search.go"
        issue: "line 55-56: 'no TS / precedent for this CLI placement' — comparison-baseline framing, not product truth"
      - path: "internal/cli/node.go"
        issue: "line 60-61: 'no TS precedent for this CLI placement' — same framing"
      - path: "internal/cli/files.go"
        issue: "lines 18 and 90: 'matches TS's files --filter <dir> semantics' / 'matches TS --filter <dir>' — comparison-baseline framing"
      - path: "internal/cli/uninit.go"
        issue: "line 56: 'mirrors TS bin/codegraph.js ~629-636's uninit cleanup' — comparison-baseline framing"
      - path: "internal/cli/serve.go"
        issue: "line 137: 'D-12/D-13: verbatim TS disabled message' — comparison-baseline framing (its test companion serve_test.go was in scope and fixed; serve.go itself was not)"
      - path: "internal/cli/githooks_test.go"
        issue: "line 50: 'the verbatim TS sync/git-hooks.js begin marker' — comparison-baseline framing (distinct file from internal/githooks/githooks_test.go, which was fixed)"
      - path: "internal/mcp/tools.go"
        issue: "line 369: 'Go-vs-TS divergence: TS returns markdown from every MCP tool'; line 528: 'mirroring TS's withWorktreeNotice' — comparison-baseline framing"
      - path: "internal/agents/codex.go"
        issue: "line 14: 'TS integrates with'; line 17: 'mirrors TS's own toml.ts' — comparison-baseline framing (05-05's task 1 covered 10 named files in internal/agents; codex.go was not among them)"
      - path: "internal/query/traverse_test.go"
        issue: "line 780: 'TS test-files-as-leaves pruning' — comparison-baseline framing (05-04's declared files_modified lists only traverse.go, not its test file)"
      - path: "testdata/golden/behavioral_test.go"
        issue: "lines 702-704: 'the authoritative TS-key-to-Go/Pebble-analog mapping table', 'not byte-identical TS values'; line 835: 'diverge from TS's historical output' — this file IS in 05-04's declared files_modified, so this is an in-scope miss, not a scope gap"
    missing:
      - "Reword the 10 locations above to remove comparison-baseline framing while preserving the underlying technical content (the D-12/D-13 disabled-message contract, the worktree-notice placement rationale, the markdown-vs-JSON design note, the status.go key-mapping table, the AD-04 allowed-divergence note, etc. all still need to exist — only the TS-as-comparison-baseline wording needs to go, per D-01/D-02)."
      - "Re-run a multiline (rg -U) census over internal/, tools/, test/, testdata/ after the fix and record a zero count, not merely a passing exit."
deferred:
  - truth: "internal/bench and tools/bench comparison-runner framing is removed"
    addressed_in: "Phase 6"
    evidence: "ROADMAP.md Phase 6 success criterion 2 (BENCH-02): 'tools/bench contains no comparison runner, and internal/bench.CheckRegression... still fires on a real regression.' WINDOWS #16 explicitly records internal/bench/rss.go and tools/bench/runner/main.go as deferred here BY DESIGN. This verification additionally found internal/bench/metrics.go:7 ('the head-to-head Go-vs-TS runner (Plan 08-07)') carrying the same class of framing, describing the same runner Phase 6 removes — same disposition applies."
human_verification: []
---

# Phase 5: Process, CI & In-Tree Sweep Verification Report

**Phase Goal:** A contributor filing an issue or opening a PR, and an agent reading the source, meet a project described on its own terms — with TypeScript-the-indexed-language intact and `codegraph migrate` removed (maintainer ruling 2026-08-15).

**Verified:** 2026-08-15
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP.md Phase 5 Success Criteria — the sole contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A contributor filing an issue sees no comparison framing in any of the 5 issue templates (PROC-01) | ✓ VERIFIED | Multiline `rg -Uwi` census over `.github/ISSUE_TEMPLATE/*.yml` for TS/parity/drop-in/upstream/ports-observable-behavior vocabulary: zero hits. All 5 files present. `bug_report.yml`'s "Migrated from TypeScript CodeGraph" install-method option is gone (05-02-SUMMARY.md, confirmed by re-read of the file). |
| 2 | A contributor opening a PR sees none in `pull_request_template.md` or the 3 `PULL_REQUEST_TEMPLATE/*` variants (PROC-02) | ✓ VERIFIED | Same census over `.github/pull_request_template.md` + `.github/PULL_REQUEST_TEMPLATE/{enhancement,feature,fix}.md`: zero hits. One incidental match on the bare word "comparison" in `enhancement.md:51` is benchmark-hygiene guidance ("a comparison across machines... measures the machines, not the change") — ordinary English, not TS-comparison framing. |
| 3 | Every workflow still passes `actionlint` and its own required status checks after the sweep, with job names, step names and comments carrying no retired framing (PROC-03) | ✓ VERIFIED | `actionlint .github/workflows/*.yml` (all 14 files) exits 0. A per-file job-ID/job-`name:` line diff against pre-phase base `524fe61` shows only `bench.yml` changed a `name:` line (`headtohead` job display name only — job ID and the 6 live-enforced required-check context strings, confirmed via `gh api repos/seanb4t/codegraph-go/rulesets/20157557`, are byte-identical to base). `TestRequiredCheckNamesPreserved` and `TestWorkflowRunBodiesInvokeTask` pass. `ci.yml`/`post-release-verify.yml` "upstream" hits are GitHub-Actions technical vocabulary (an unmaintained Go dependency, a `workflow_run` event's originating run) — ordinary English, not TS-comparison framing. `bench.yml`'s remaining "comparison binary"/`colbymchenry` content is the still-live head-to-head runner, explicitly scoped to Phase 6 (BENCH-02) per ROADMAP notes and 05-03-SUMMARY.md; only its display names were reworded here, as designed. |
| 4 | Comments across `internal/`, `tools/` and `test/` carry no comparison framing, while `internal/indexer/tsextract`, the language registry and the capability matrix are preserved as product surface (CODE-01) | ✗ FAILED | See Gaps below. 10 file:line locations independently reproduced with live TS-comparison framing, all still present verbatim in the working tree. Product surface confirmed intact (see below) — the failure is scoped entirely to residual framing, not capability loss. |
| 5 | `codegraph migrate` is removed entirely: the command, `internal/migrate/`, its fixture, and the `modernc.org/sqlite` sole-use dependency are gone, nothing references it (CODE-03 amended) | ✓ VERIFIED | `rg -il 'modernc\.org/sqlite\|internal/migrate\|codegraph migrate' --glob '!.planning' .` → zero hits. `internal/migrate/` directory absent. `go.mod`/`go.sum` have no `modernc` entries. `internal/cli/root.go` has no `migrate` registration; `go run ./cmd/codegraph --help \| rg -i migrate` → zero hits. `testdata/golden/ts-*` fixtures absent. `go build ./...` and `go vet ./...` both exit 0. |

**Score:** 4/5 truths verified

### Product Surface Preserved

Independently confirmed that the sweep removed framing, not capability, per the milestone's own "sweep removes framing, never capability" rule (with migrate as the one recorded exception):

- `internal/indexer/tsextract/` package intact: `tsextract.go`, `types.go`, `resolution_test.go`, `d09_test.go` all present.
- `internal/indexer/languages_typescript.go:123-133` still registers TypeScript, TSX and JavaScript as three separate languages via `cgo.NewTypeScriptParser()`.
- `internal/indexer/languages_test.go` still has `TestLanguageRegistry_TypeScript`.
- `docs/LANGUAGE-CAPABILITY-MATRIX.md` still documents TypeScript (6 occurrences).
- `go build ./...` and `go vet ./...` both exit 0 — the migrate removal did not break the build.

### Census Instrument

**Positive control (proving the instrument can see what a line-based one cannot):** planted a two-line comment (`// no TS` / `// precedent here`) in a scratch file and confirmed `rg -U -o 'no TS\s*\n\s*//?\s*precedent'` matches it (a line-based `rg` without `-U` cannot span the newline). Only after this control passed did the zero-hit results for criteria 1-3 above count as evidence.

**CODE-01 census (criterion 4):** ran `rg -U` (multiline) with word-bounded comparison-framing terms over `internal/`, `tools/`, `test/`, `testdata/` restricted to `*.go`, then manually read every hit in context (42 raw pattern matches across 25 files) to separate:
- **False positives from my own pattern's over-match** (e.g. `drop.in` matching inside "dropping" — a bug in my regex, not the codebase; "parity" matching `TestGoreleaserPinParity` and `TestCosignIdentityPolicyBoundaryParityWithCompiledPattern`, both legitimate mathematical/logical parity between two independent implementations of the same check, not TS comparison).
- **Legitimate keeps**: `internal/agents/opencode.go`/`claude.go` citing "TS issue #535"/"TS issue #207" as historical bug-precedent citations (not comparison-baseline framing under D-01); `internal/indexer/languages_ruby.go:42` "mirrors tsextract's... pattern" (referring to the Go package `tsextract`, not the origin project); `internal/query/status.go:44` documenting that a "TS parity" constraint "was formally retired 2026-08-13" — past-tense history of a retired constraint, which the phase's own D-02 explicitly carves out ("past-tense history stays").
- **Confirmed genuine framing**: the 10 locations listed in Gaps, all independently re-read in full context, all matching WINDOWS.md's already-open entries 4-11, 14, 15 exactly. No new locations beyond what WINDOWS.md already logs were found in `internal/`, `tools/`, `test/`, `testdata/` — this verification's independent census corroborates rather than expands the orchestrator's finding.

The root cause WINDOWS.md records — both flagship censuses (05-04, 05-05) used line-based `rg` that cannot match a phrase split across a comment line wrap — is independently confirmed: all three phrases in `testdata/golden/behavioral_test.go` (an in-scope file per 05-04's own `files_modified` list) that were missed span line breaks (`// the authoritative\n// TS-key-to-Go/Pebble-analog mapping table`, etc.).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| PROC-01 | 05-02 | 5 issue templates, no comparison framing | ✓ SATISFIED | Criterion 1 above |
| PROC-02 | 05-02 | PR template + 3 variants, no comparison framing | ✓ SATISFIED | Criterion 2 above |
| PROC-03 | 05-03 | Workflows pass actionlint + own required checks, no framing in names/comments | ✓ SATISFIED | Criterion 3 above |
| CODE-01 | 05-04, 05-05, 05-06 | Comments in internal/, tools/, test/ carry no comparison framing; product surface preserved | ✗ BLOCKED | Criterion 4 above — 10 locations still carry live framing |
| CODE-03 (amended) | 05-01 | `codegraph migrate` removed entirely | ✓ SATISFIED | Criterion 5 above |

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX`/`HACK`/`PLACEHOLDER` debt markers found in any file listed across the 6 plans' `key-files` sections. The 10 CODE-01 gap locations are pre-existing comparison-framing prose, not stub/placeholder code — this is a documentation/comment defect, not a functional regression.

### Build & Test State

- `go build ./...` → exit 0
- `go vet ./...` → exit 0
- `internal/daemon` diff vs pre-phase base `524fe61` → 0 lines (confirmed independently: zero relationship to this phase's changes)
- `internal/graphstore` diff vs base → deletions-only (216 deletions across 5 files, 0 insertions — confirmed independently)
- `go test ./internal/upgrade/... -run 'TestRequiredCheckNamesPreserved$|TestWorkflowRunBodiesInvokeTask'` → PASS
- Full `go test ./...` was launched once; `internal/daemon`'s `TestRunWatchdogCancelsRunOnSimulatedReparent` load-sensitivity (WINDOWS #12, already diagnosed and unrelated to this phase per the zero-diff evidence above) was not re-litigated further since the diff evidence alone settles attribution.

### Gaps Summary

Four of five ROADMAP.md success criteria are true of the codebase. Criterion 4 (CODE-01) is not: a multiline census — run independently of, and cross-checked against, WINDOWS.md's already-open ledger entries — confirms 10 file:line locations still carry TS-comparison-baseline framing (e.g. "no TS precedent for this CLI placement", "mirrors TS bin/codegraph.js", "Go-vs-TS divergence: TS returns markdown from every MCP tool", "the authoritative TS-key-to-Go/Pebble-analog mapping table"). Nine of the ten were outside every plan's declared file scope (WINDOWS 4-11, 14); the tenth is an in-scope miss inside a file 05-04 itself declared as modified (WINDOWS 15), caused by both flagship censuses using a line-based grep pattern that cannot see a phrase wrapped across a comment line break.

This is not a manufactured gap: the orchestrator's own WINDOWS.md ledger already records all 10 as open, and this verification's independent multiline census — built on its own positive-control instrument and manually adjudicated term-by-term — reproduces the exact same set with no additions and no false subtractions. The gap is narrow (10 comment-only edits, no capability loss, no build/test breakage) and does not touch product surface (tsextract, the language registry, and the capability matrix were all independently confirmed intact).

Not counted as gaps: `internal/bench/rss.go`, `tools/bench/runner/main.go`, and this verification's own additional find `internal/bench/metrics.go:7` — all describe the still-live head-to-head comparison runner, which ROADMAP.md Phase 6 (BENCH-02) explicitly scopes for removal; WINDOWS #16 already records this deferral by design. `internal/query/status.go:44`'s "TS parity is no longer owed" is past-tense documentation of a retired constraint, matching the phase's own D-02 carve-out for historical framing, and is not counted as a gap.

---

*Verified: 2026-08-15*
*Verifier: Claude (gsd-verifier)*
