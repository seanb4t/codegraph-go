---
phase: 11-graph-view-community-clustering
plan: 05
subsystem: security
tags: [mutation-testing, security-register, validation, gonum, cgo, threshold, force-layout, phase-close]

requires:
  - phase: 11-graph-view-community-clustering
    provides: "11-01: gonum wire fields + determinism proof; 11-02: GRF-09 verdict PASS + ancestry/digest tests; 11-03: check:gonum supply-chain gate; 11-04: UI colouring + check-no-force-layout scan"
provides:
  - "11-MUTATION-LOG.md — four hand-authored RED demonstrations (seed counter, threshold widened, force-layout name planted, cgo control neutered), each confirmed applied, watched RED, and reverted byte-clean"
  - "11-SECURITY.md — 14 T-11-xx rows + T-11-SC deduplicated from plans 11-01..11-04, every row test-or-verdict, threats_open: 0"
  - "11-VALIDATION.md — Per-Task Verification Map filled with real test names/commands, Wave 0 ticked, wave_0_complete/nyquist_compliant: true"
  - "Phase-close gate green at the final commit: vet, build, test:unit, test:golden, lint:go, proto:drift, web:test, svelte-check, web:drift, check:gonum, check:no-force-layout, ancestry/digest tests"
affects: [phase-11-close, milestone-verification]

actuals:
  tokens: 12440
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Mutation demonstrated live (diff applied, instrument run, RED pasted verbatim, reverted via git checkout --, re-verified GREEN) rather than narrated — the house discipline this project applies to every phase-close mutation log"
    - "Working-tree-only mutations for tool-owned artifacts (the threshold JSON, Taskfile.yml) — never staged, never committed, single-commit/clean-diff invariant asserted before and after"

key-files:
  created:
    - .planning/phases/11-graph-view-community-clustering/11-MUTATION-LOG.md
    - .planning/phases/11-graph-view-community-clustering/11-SECURITY.md
  modified:
    - .planning/phases/11-graph-view-community-clustering/11-VALIDATION.md
    - internal/indexer/resolve_test.go
    - internal/query/community_test.go

key-decisions:
  - "Family (a)'s per-call seed counter mutation flips TestAssignCommunitiesDeterministic/tie (not on the first perturbed call, but by the second sub-test's own second iteration) while structured stays PASS — recorded as the seed-invariance of a well-separated graph, exactly as 11-01's own hardening test independently established, never rewritten to force an earlier flip"
  - "T-11-09 (11-02's threat model row for the D-07 index-time-persistence fallback) is excluded from the register with a one-line Notes footnote, per the plan's own instruction, because 11-02 Task 3 correctly did not execute (GRF-09 verdict was PASS)"
  - "Re-ran task web:build during Task 2's phase-close gate produced spurious Vite content-hash filename churn in the committed web/build tree (documented, expected non-determinism per Taskfile.yml's own web:build/web:drift doc comments) with zero web/src changes this plan — reverted the rebuild artifacts and verified the phase-close web:drift invariant by re-hashing the already-committed tree in place instead of rebuilding it a second time"
  - "gofmt-fixed internal/indexer/resolve_test.go and internal/query/community_test.go (pre-existing formatting drift from earlier commits, neither touched by this plan's own tasks) because they were blocking the required task lint:go phase-close gate — a Rule 3 blocking-issue fix, zero behavior change, verified via a full re-run of the affected test packages"

requirements-completed: [GRF-06, GRF-08, GRF-09, GRF-10]

coverage:
  - id: D1
    description: "11-MUTATION-LOG.md: four families (GRF-08 seed counter, GRF-09 threshold widened, GRF-06 force-layout name planted, GRF-10 cgo control neutered) each demonstrated RED against a confirmed-applied mutation and reverted byte-clean"
    requirement: "GRF-06"
    verification:
      - kind: other
        ref: "GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestAssignCommunitiesDeterministic' ./internal/query/ (RED then GREEN, transcripts in 11-MUTATION-LOG.md family (a))"
        status: pass
      - kind: other
        ref: "GOTOOLCHAIN=go1.26.6 go test -count=1 -v -run 'TestClusterThresholdDigestMatchesCommittedObservation' ./tools/graphcluster/ (RED then GREEN, family (b))"
        status: pass
      - kind: other
        ref: "node web/scripts/check-no-force-layout.mjs (RED then PASS, family (c))"
        status: pass
      - kind: other
        ref: "GOTOOLCHAIN=go1.26.6 task -s check:gonum (RED then PASS, family (d))"
        status: pass
    human_judgment: false
  - id: D2
    description: "11-SECURITY.md: every T-11-xx threat row (14 rows + T-11-SC) carries a named test or verdict; threats_open: 0 with the three high-severity rows (T-11-01, T-11-02, T-11-10) closed by name; GRF-10 [ASSUMED] legitimacy and GRF-09 PASS verdict recorded verbatim in Notes"
    requirement: "GRF-10"
    verification:
      - kind: other
        ref: "rg-based structural assertions on 11-SECURITY.md (row count >= 13, threats_open matches open-high-row count, ASSUMED + determinism-constant notes present) — all pass, transcript in this SUMMARY's Verification section"
        status: pass
    human_judgment: false
  - id: D3
    description: "11-VALIDATION.md's Per-Task Verification Map and Wave 0 checklist filled with real values, wave_0_complete/nyquist_compliant set true; full phase-close gate (vet, build, test:unit, test:golden, lint:go, proto:drift, web:test, svelte-check, web:drift, check:gonum, check:no-force-layout, ancestry/digest tests) green at the final commit"
    requirement: "GRF-09"
    verification:
      - kind: other
        ref: "rg-based structural assertions on 11-VALIDATION.md (zero TBD, flags true, status draft unchanged, headings unchanged, >=9 ticked) — all pass"
        status: pass
      - kind: other
        ref: "the 12 phase-close gate commands (tails pasted below) — all green"
        status: pass
    human_judgment: false

duration: 50min
completed: 2026-09-13
status: complete
---

# Phase 11 Plan 05: Mutation Log, Security Register, and Phase-Close Gate Summary

**Four RED demonstrations (seed-counter perturbation, threshold widening, force-layout planting, cgo-control neutering) proved the phase's guards discriminate, `11-SECURITY.md` closed all three high-severity threats by name with `threats_open: 0`, `11-VALIDATION.md`'s map is filled with the real tests that landed, and the full 12-command phase-close gate is green with the threshold still in its one, untouched commit.**

## Performance

- **Duration:** 50 min
- **Tasks:** 2
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- `11-MUTATION-LOG.md`: four families, each with a real diff applied to a tracked file, the instrument run, the RED output pasted verbatim, and the mutation reverted via `git checkout --` with a post-revert cleanliness gate — (a) a per-call seed counter in `internal/query/community.go` flipped `TestAssignCommunitiesDeterministic/tie` RED while `structured` correctly stayed PASS (the documented seed-invariance of a well-separated graph); (b) `corpora/graph-cluster-threshold.json`'s `max` widened 500→5000 in the working tree only turned `TestClusterThresholdDigestMatchesCommittedObservation` RED with both digests named, the threshold staying at exactly one commit throughout; (c) a real `name: 'cose'` literal planted in `web/src/lib/components/graph/graph-style.ts` turned `check-no-force-layout.mjs` RED, naming the exact file and line; (d) `Taskfile.yml`'s `check:gonum` cgo positive control swapped from `github.com/tree-sitter/go-tree-sitter` to the pure-Go `github.com/spf13/cobra` turned the target RED on `positive control found no cgo`, reaching the third half after the first two printed their own clean lines.
- `11-SECURITY.md`: the union of all 14 distinct `T-11-xx` threat ids plus `T-11-SC` from plans 11-01 through 11-04, deduplicated by id with merged mitigation text tagged by plan. `threats_open: 0` — the three `high`-severity rows (T-11-01 gonum supply chain, T-11-02 threshold tampering, T-11-10 cgo elevation) are each closed by name, citing the landed `check:gonum` gate and the ancestry/digest tests. Notes record the GRF-10 `[ASSUMED]` package-legitimacy assumption verbatim (no npm/pypi/crates seam covers a Go module, so no machine verdict and no checkpoint were possible), the GRF-09 verdict (PASS, median 106 ms, D-07 fallback not executed), and that `check:gonum`/`check:no-force-layout` are not yet wired into CI — a visible, recorded follow-up, not a silent gap.
- `11-VALIDATION.md`: every `TBD` cell in the Per-Task Verification Map replaced with the real plan/task ids and test names that landed (`11-02/T2` → `TestClusterThresholdCommitIsAncestorOfEveryMeasurement`, `11-03/T1` → `task -s check:gonum`, `11-01/T2+T3` → `TestAssignCommunitiesDeterministic`, `11-04/T3` → `check-no-force-layout.mjs`, `11-04/T2` → `graph-communities` route test, etc.); all 10 Wave 0 checklist items ticked; `wave_0_complete: true` and `nyquist_compliant: true` set; `status: draft` and all 6 `##` headings left untouched for `validate-phase` to own.
- Phase-close gate run at the final commit, all 12 commands green: `go vet ./...`, `go build ./...`, `task test:unit` (0 FAIL), `task test:golden` (`ok`), `task lint:go` (0 issues, after a gofmt fix — see Deviations), `task proto:drift` (4 files byte-identical), `task web:test` (583/583), `pnpm -C web check` (0 errors), `task -s web:drift` (both halves MATCH, no rebuild churn left behind), `task -s check:gonum` (PASS), `task -s check:no-force-layout` (PASS), and the ancestry + digest tests (both PASS, threshold still at exactly one commit, working tree clean).

## Task Commits

1. **Task 1: `11-MUTATION-LOG.md` — four families RED and reverted** — `43a0bca` (docs)
2. **Task 2: `11-SECURITY.md` register, `11-VALIDATION.md` map, phase-close gate + gofmt fix** — `6c03f0d` (docs+fix)

_plan_head_before: `cba1275c165cd20907e912bf148aa7137299791a`_ (2 commits total this plan, measured via `git rev-list --count`)

No separate plan-metadata commit beyond this SUMMARY's own commit (per orchestrator instruction: STATE.md/ROADMAP.md are NOT updated by this plan — the orchestrator owns those writes).

## Mutation transcripts (see `11-MUTATION-LOG.md` for full diffs and context)

### Family (a) — GRF-08 seed counter

RED:
```
community_test.go:121: run 2: AssignCommunities = map[a.go:1 b.go:2 c.go:2 d.go:1], want run 0's result map[a.go:1 b.go:1 c.go:2 d.go:2]
--- FAIL: TestAssignCommunitiesDeterministic (0.00s)
    --- PASS: TestAssignCommunitiesDeterministic/structured (0.00s)
    --- FAIL: TestAssignCommunitiesDeterministic/tie (0.00s)
```
GREEN (post-revert): `--- PASS: TestAssignCommunitiesDeterministic` (both sub-tests PASS).

### Family (b) — GRF-09 threshold widened

RED:
```
ancestry_test.go:129: observation.thresholdDigest = "sha256:6adc6722ff…", want "sha256:0e61d573…" (committed threshold digest) — the threshold was edited after measurement, or the observation is stale
--- FAIL: TestClusterThresholdDigestMatchesCommittedObservation (0.01s)
```
Threshold commit count before and after: **1**. GREEN (post-revert): `--- PASS: TestClusterThresholdDigestMatchesCommittedObservation`.

### Family (c) — GRF-06 force-layout planted

RED:
```
check-no-force-layout: scanned 106 files; elk layout refs 2; elk import refs 3; forbidden matches 1; verdict FAIL
  forbidden: web/src/lib/components/graph/graph-style.ts:187 — name: 'cose'
exit=1
```
GREEN (post-revert): `verdict PASS`, `forbidden matches 0`, exit 0.

### Family (d) — GRF-10 cgo control neutered

RED:
```
check:gonum: cgo closure over gonum.org/v1/gonum/graph/community — 91 packages inspected, 0 with cgo (want 0), blas/cgo references 0 (want 0); positive control github.com/tree-sitter/go-tree-sitter — 0 with cgo (want >= 1)
::error::check:gonum: positive control found no cgo in github.com/tree-sitter/go-tree-sitter — the scan cannot detect cgo, its clean verdict is meaningless
```
(The first two halves' lines — govulncheck clean, SBOM presence — printed before this failure.) GREEN (post-revert): `check:gonum: PASS`.

## Phase-close gate tails (all 12, at the final commit `6c03f0d4`)

**1. `go vet ./...`:** exit 0, no output.

**2. `go build ./...`:** exit 0, no output.

**3. `task test:unit`:**
```
ok  	github.com/seanb4t/codegraph-go/tools/graphcluster	1.170s
ok  	github.com/seanb4t/codegraph-go/web	(cached)
```
0 `FAIL` lines across the full suite.

**4. `task test:golden`:**
```
ok  	github.com/seanb4t/codegraph-go/testdata/golden	26.756s
```

**5. `task lint:go`:**
```
task: [lint:go] GOWORK=off go tool -modfile=go.tool-golangci.mod golangci-lint run ./...
0 issues.
```
(See Deviations — this required a gofmt fix to two pre-existing files not touched by this plan's own tasks.)

**6. `task proto:drift`:**
```
proto:drift: compared 4 generated files
proto:drift: all 4 generated files byte-identical to the pinned toolchain's regeneration (temporary tree only — source tree untouched)
```

**7. `task web:test`:**
```
web:test: observed numTotalTests=583 numPassedTests=583 (vitest exit 0)
web:test: PASS — 583 of 583 tests passed
```

**8. `pnpm -C web check`:**
```
1789321392691 START "/Volumes/Code/github.com/seanb4t/codegraph-go/web"
1789321392694 COMPLETED 1172 FILES 0 ERRORS 0 WARNINGS 0 FILES_WITH_PROBLEMS
```

**9. `task -s web:drift`** (re-hashing the already-committed tree in place, no rebuild — see Deviations for why a preceding `task web:build` was reverted rather than committed):
```
web:drift: hashed 116 source files
web:drift: manifested 32 output files
web:drift: source half MATCH (116 files, 5df163a2d14415cb827de27033522e8a4facf9f27d9134656c167a633eb7a1e1)
web:drift: output half MATCH (32 files, b7b124b3acb2e709c37ba2e5b7eafcb1883bc7c615a984317b781eb17a037ba6)
web:drift: PASS — hashed 116 source files, manifested 32 output files, committed web/build/ matches both digests
```

**10. `task -s check:gonum`:**
```
check:gonum: govulncheck (source mode, main module) clean — 26 gonum packages in the scanned set
check:gonum: SBOM lists 148 packages; gonum.org/v1/gonum present 1 time(s) at v0.17.0; positive control github.com/cockroachdb/pebble/v2 present 1 time(s)
check:gonum: cgo closure over gonum.org/v1/gonum/graph/community — 91 packages inspected, 0 with cgo (want 0), blas/cgo references 0 (want 0); positive control github.com/tree-sitter/go-tree-sitter — 3 with cgo (want >= 1)
check:gonum: PASS
```

**11. `task -s check:no-force-layout`:**
```
check-no-force-layout self-test: PASS — injected 'cose' detected at <self-test>/injected.ts:1
check-no-force-layout: scanned 106 files; elk layout refs 2; elk import refs 3; forbidden matches 0; verdict PASS
```

**12. Ancestry/digest tests (`go test -run 'TestClusterThreshold' ./tools/graphcluster/...`):**
```
--- PASS: TestClusterThresholdCommitIsAncestorOfEveryMeasurement (0.07s)
    ancestry_test.go:150: threshold digest sha256:6adc6722ff0148b5000e5653c77b4737fb698d27e6f632983baee474e766b971 matches; verdict PASS; median 106 ms
--- PASS: TestClusterThresholdDigestMatchesCommittedObservation (0.01s)
ok  	github.com/seanb4t/codegraph-go/tools/graphcluster	0.252s
```
Threshold commit count: **1**. `git diff --quiet -- corpora/graph-cluster-threshold.json`: exit 0.

**`git status --porcelain` at the final commit: empty.**

**`threats_open: 0`** (confirmed in `11-SECURITY.md` frontmatter; the three `high`-severity rows T-11-01, T-11-02, T-11-10 are each `closed`).

## Files Created/Modified

- `.planning/phases/11-graph-view-community-clustering/11-MUTATION-LOG.md` — four RED/GREEN family demonstrations
- `.planning/phases/11-graph-view-community-clustering/11-SECURITY.md` — the phase's threat register
- `.planning/phases/11-graph-view-community-clustering/11-VALIDATION.md` — Per-Task Verification Map and Wave 0 filled
- `internal/indexer/resolve_test.go` — gofmt-only fix (pre-existing drift from an earlier phase's commit, unblocking `task lint:go`)
- `internal/query/community_test.go` — gofmt-only fix (pre-existing drift from 11-01's own commit, unblocking `task lint:go`)

## Decisions Made

See `key-decisions` in frontmatter: the family (a) seed-flip timing, T-11-09's exclusion with a Notes footnote, the web:build/web:drift non-determinism handling, and the gofmt fix rationale.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking issue] `task lint:go` failed on pre-existing gofmt drift in two files this plan never edited**
- **Found during:** Task 2's phase-close gate
- **Issue:** `golangci-lint run ./...` reported gofmt issues in `internal/indexer/resolve_test.go` (from an earlier, unrelated phase-10 commit `83666cce`) and `internal/query/community_test.go` (from 11-01's own commit `b2c9ae25`). Neither file was touched by this plan's own tasks, and `git status --short` on both confirmed they were byte-identical to HEAD before any fix — the drift was already present in the committed tree, not introduced here. This blocked the required, literal `task lint:go` phase-close gate from passing.
- **Fix:** Ran `gofmt -w` on both files — a purely mechanical whitespace/line-wrap change with zero semantic difference (confirmed via `gofmt -d` diffs showing only spacing/line-break changes, and a full re-run of `go test ./internal/query/... ./internal/indexer/...` showing no behavior change). `golangci-lint run ./...` then reported `0 issues`.
- **Files modified:** `internal/indexer/resolve_test.go`, `internal/query/community_test.go`
- **Verification:** `task lint:go` exits 0 with `0 issues.`; `GOTOOLCHAIN=go1.26.6 go test -count=1 ./internal/query/... ./internal/indexer/...` all `ok`, zero `FAIL`
- **Committed in:** `6c03f0d4` (part of Task 2's commit)

**2. [Rule 1 - Plan verification arithmetic bug] Running the plan's literal `task web:build && task -s web:drift` sequence introduces spurious filename churn in the committed `web/build` tree when no `web/src` change exists this plan**
- **Found during:** Task 2's phase-close gate
- **Issue:** `Taskfile.yml`'s own `web:build`/`web:build:verify` doc comments explicitly document that Vite/Rollup builds of identical source are not guaranteed byte- or filename-identical across runs (`vitejs/vite#15555`, closed not-planned; `#13071`) — SvelteKit's `_app/version.json` embeds a build timestamp that changes every invocation, cascading into different content-hash filenames for every immutable chunk. Running `task web:build` a second time (this plan made zero `web/src` changes) rewrote the already-correct, already-committed `web/build/` with a fresh, differently-hashed build, leaving `git status --porcelain -- web/build` dirty (9 deleted + 9 new files + 2 modified) even though nothing about the shipped app actually changed.
- **Fix:** Reverted the spurious rebuild (`git checkout -- web/build`, then removed the newly-untracked chunk files by name) to restore the already-correct committed tree, then ran `task -s web:drift` ALONE (no preceding rebuild) — which re-hashes the already-committed source and output trees in place, exactly the invariant the plan's verify block actually cares about ("both halves MATCH", "committed `web/build/` matches both digests"). This is precisely the distinction `web:drift`'s own doc comment draws: it "hashes a tree that already exists on disk," which "has no such [non-determinism] dependency," unlike a regenerate-and-compare shape.
- **Files modified:** None (working-tree-only rebuild reverted, not committed)
- **Verification:** `task -s web:drift` reports `source half MATCH` and `output half MATCH`; `git status --porcelain -- web/build web/src` empty
- **Committed in:** N/A (no code change; the correct, already-committed `web/build` was left untouched)

---

**Total deviations:** 2 (1 auto-fixed in code, Rule 3; 1 documented verification-only finding, Rule 1). **Impact on plan:** No scope creep. The gofmt fix is a mechanical, zero-behavior-change correction required to make the plan's own literal `task lint:go` gate pass, consistent with this repo's established pattern of fixing blocking issues discovered while satisfying a required gate. The web:build/web:drift finding required no code change at all — the correct, already-passing committed state was preserved by NOT committing a spurious rebuild, and verified via the non-rebuild half of the same target the plan specifies.

## Issues Encountered

None beyond the deviations documented above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

Phase 11 (Graph View — Community Clustering) is fully closed: all four requirements (GRF-06, GRF-08, GRF-09, GRF-10) have landed, tested, and now proven RED against confirmed-applied mutations; `11-SECURITY.md` closes all three high-severity threats by name with `threats_open: 0`; `11-VALIDATION.md`'s map and Wave 0 reflect what actually shipped; and the full phase-close gate is green at commit `6c03f0d4` with `git status --porcelain` empty. Two items are recorded as visible, non-blocking follow-ups for a future phase: (1) wiring `check:gonum`/`check:no-force-layout` into `.github/workflows/ci.yml` (currently developer-run only; `ci.yml`'s existing `govulncheck` job remains the automated supply-chain gate); (2) re-verifying `gonum.org/v1/gonum`'s `[ASSUMED]` package-legitimacy tag with machine tooling once a Go-module seam exists.

No blockers.

---
*Phase: 11-graph-view-community-clustering*
*Completed: 2026-09-13*

## Self-Check: PASSED

All key files confirmed present on disk (`11-MUTATION-LOG.md`, `11-SECURITY.md`, `11-VALIDATION.md`, `internal/indexer/resolve_test.go`, `internal/query/community_test.go`). Both task commits confirmed present in `git log` (`43a0bca`, `6c03f0d4`). All rg-based structural assertions on the three planning artifacts re-confirmed passing. The 12-command phase-close gate re-confirmed green at HEAD. `git status --porcelain` confirmed empty.
