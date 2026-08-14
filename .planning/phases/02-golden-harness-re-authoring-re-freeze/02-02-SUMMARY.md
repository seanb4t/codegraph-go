---
phase: 02-golden-harness-re-authoring-re-freeze
plan: 02
subsystem: golden-harness
tags:
  - fix-04
  - fix-05
  - d-03
  - d-04
  - d-09
requires:
  - 02-01
  - 01-06
status: complete
key-files:
  created:
    - corpus/behavioral/CASES.json
    - corpus/behavioral/src/ (moved)
    - corpus/behavioral/go-explore-multi.json (moved)
    - corpus/behavioral/go-node-multi.json (moved)
  modified:
    - testdata/golden/behavioral_test.go
    - testdata/golden/golden_test.go
    - testdata/golden/gocapture/main.go
    - testdata/golden/README.md
    - internal/query/explore_test.go
    - internal/query/render_markdown_test.go
    - internal/query/traverse_test.go
    - internal/corpora/coverage_test.go
    - testdata/golden/behavioral_java_test.go
    - testdata/golden/behavioral_python_test.go
    - testdata/golden/behavioral_csharp_test.go
    - testdata/golden/behavioral_tsjs_test.go
    - corpus/behavioral/src/go.mod
  deleted:
    - testdata/golden/capture.sh
    - testdata/golden/mcp-capture.mjs
    - testdata/golden/corpus/weft-go/ (14 fixtures)
    - testdata/golden/corpus/colbymchenry-codegraph/ (14 fixtures)
    - testdata/golden/corpus/synthetic-parity/ (7 files: 4 TS fixtures + README + 2 go-* moved)
decisions:
  - identifiers/paths/names framing gate is scoped to module line, CASES.json, file/dir names
  - go-* golden command-field byte (`-p synthetic-parity`) is PARKED, retired by 02-04 re-capture
  - test sequencing adjustments: H1 reader re-pointing moved from Task 1 to Task 2
metrics:
  duration_minutes: 35
  completed_date: 2026-08-14
actuals:
  tokens: 74000
  tasks: 3
  commits: 3
---

# Phase 2 Plan 2: Cleanup + Corpus Move (FIXT-04/FIXT-05/D-09)

**One-liner:** TS-era capture path and external corpora deleted; behavioral corpus moved to `corpus/behavioral/` with committed `CASES.json` case map and CASES.json-driven D-09 property assertions over live engine output.

## Tasks Completed

### Task 1: Delete TS-era capture path and weft-go / colbymchenry-codegraph corpora (FIXT-04/D-07)

**Commit:** 72b640e — 43 files, 189 insertions / 2072 deletions

Deleted `capture.sh`, `mcp-capture.mjs`, the entire `weft-go/` (14 fixtures) and `colbymchenry-codegraph/` (14 fixtures) directories. Retired all in-tree references:

- **behavioral_test.go:** Removed resolvers (`resolveWeftCorpus`, `resolveWeftGoCorpusLoose`, `resolveColbymchenryCorpus`, `gitHead`, `buildWeftEngine`, `defaultWeftRepo`, `dirExists`, `corpusCandidate`, `pinnedWeftCommit`). Rewrote `TestCorpusBehavior_Go` as pure property assertions over the synthetic-parity corpus (Go-only `{"go"}` language set, edgesByKind/filesByLanguage key-presence, dbSizeBytes plausibility). Emptied `TestCorpusBehaviorLockedCorpora` scaffold. Removed weft rows from CLI==MCP trio. Deleted unused `loadGoldenFixture`/`loadGoldenOutput` (weft-specific). Trimmed D-02 harness doc comment.
- **gocapture/main.go:** Removed `weftGoSpec`, `colbymchenryCodegraphSpec`, `os/exec` import.
- **golden_test.go:** Cleaned capture.sh references from comments.
- **internal/corpora/coverage_test.go:** Removed `resolveWeftCorpus`/`resolveColbymchenryCorpus` doc references.
- **internal/query/render_markdown_test.go:** Re-pointed to synthetic-parity `go-explore-multi.json` (temp repoint; final repoint in task 2).
- **internal/query/traverse_test.go:** Neutralized weft-go/impact.json comment.
- **behavioral_{java,python,csharp,tsjs}_test.go:** Removed weft analog comments.
- **testdata/golden/README.md:** Rewrote without capture.sh/weft/colbymchenry narrative.
- **D-08 boundary:** NOTICE, tools/bench/, bench.yml, repo-root README bench table, ts-schema.* untouched.

### Task 2: Move behavioral corpus to corpus/behavioral/, author CASES.json, re-point readers (FIXT-05/D-03/D-04)

**Commit:** cfda0c1 — 21 files, 156 insertions / 113 deletions

- **Moved (git mv):** `synthetic-parity/src/` (7 files) → `corpus/behavioral/src/`; `go-explore-multi.json` + `go-node-multi.json` → `corpus/behavioral/`. All show as `R100` rename (byte-identical).
- **Deleted:** 4 TS-captured fixtures + README from synthetic-parity dir.
- **Authored:** `corpus/behavioral/CASES.json` with 4 cases (a/b/c/d), each carrying `query` + `assertion` mode.
- **Authored:** `loadBehavioralCases` + `behavioralCase` struct + `loadBehavioralFixture` helpers in behavioral_test.go.
- **Renamed:** `corpus/behavioral/src/go.mod` module line: `synthetic-parity` → `behavioral`.
- **Re-pointed:** `syntheticParitySrc` → `corpus/behavioral/src` (two-hop repo-root path); `golden_test.go` glob merges `corpus/*/*.json` + `../../corpus/behavioral/*.json`; `TestGoSideFixturesRegenerated` → `corpus/behavioral/go-*.json`; gocapture `syntheticParitySpec` → `behavioralCorpusSpec` (two-hop repoRoot); internal/query `explore_test.go` src → `corpus/behavioral/src`; `render_markdown_test.go` → `corpus/behavioral/go-explore-multi.json`.
- **Patched (stopgap):** TestCorpusBehaviorSynthetic reads go-* goldens from corpus/behavioral (Task 3 supersedes this).

### Task 3: Re-author TestCorpusBehaviorSynthetic as CASES.json-driven D-09 property assertions (FIXT-05/D-09)

**Commit:** 5eb5138 — 7 files, 139 insertions / 121 deletions

Rewrote `TestCorpusBehaviorSynthetic` as a data-driven loop over `loadBehavioralCases()`, switching on each case's `assertion` mode:

- **Case (a) overloaded-defs-distinct:** `Node("Validate")` returns exactly 2 defs (`accounts/validate.go:10` + `orders/validate.go:10`), header template matches, def count = 2.
- **Case (b) multi-word-tokenization:** `Explore("user account")` surfaces `UserAccountManager`, selects `accounts/manager.go`.
- **Case (c) cluster-surfaces-connected-non-test:** `Explore("user account")` surfaces `recoverAccount`/`validateRecovery` over zero-inbound `TestAccountRecovery`, with `tests: recovery/recovery.go` clause.
- **Case (d) structural-surfaces-zero-lexical-match:** `Explore("account balance")` selects `ledger/ledger.go`, surfaces `GetBalance` (partial-lexical bridge) and `ReconcileLedger` (zero lexical match) — surfacing only, not ranking above `AccountBalanceHelper` (matches authoritative contract in `internal/query/explore_test.go:144-183`).

Cleaned up all remaining "parity", "synthetic-parity" string references in testdata/golden/* (language behavioral tests, golden_test.go, gocapture comments).

## Deviations from Plan

### Sequencing Adjustment — H1 Internal/Query Reader Re-pointing

**Not a functional deviation.** The plan's Task 1 action listed the H1 reader re-pointing (`internal/query/explore_test.go`, `render_markdown_test.go`) within Task 1's file scope. These re-points required the target corpus to exist at `corpus/behavioral/`, which only Task 2 creates. To keep every commit green, the re-pointing was split: `render_markdown_test.go` was temporarily re-pointed to synthetic-parity in Task 1, then finally re-pointed to `corpus/behavioral` in Task 2. The other internal/query reader was re-pointed in Task 2. End state matches plan.

### TestCorpusBehaviorSynthetic Stopgap (Task 2)

The plan's Task 2 deletes the TS-captured fixtures that `TestCorpusBehaviorSynthetic` reads from `synthetic-parity/`. To keep the Task 2 commit green, the test was temporarily patched to read from the moved go-* goldens (`corpus/behavioral/go-node-multi.json`, `go-explore-multi.json`) as a regression snapshot. This was a stopgap superseded by Task 3's full CASES.json-driven re-author.

## Auto-fixed Issues

None. Plan executed as written.

## Known Stubs

None. The four behavioral cases are fully wired through CASES.json to the property assertions, and all tests pass against live engine output.

## Threat Flags

None identified.

## Verification

- `rg -i "weft|colbymchenry|mcp-capture|capture\.sh" testdata/golden/ | grep -v ts-schema` = 0 hits.
- `rg "parity" testdata/golden/` (harness scope) = 0 hits (after cleanup).
- `rg -i "synthetic-parity|module synthetic" corpus/` matches are in .go source doc comments (corporus documentation) — exempt per framing gate rule (identifiers/paths/names + authored data only). Module line neutralized.
- `go test -count=1 ./testdata/golden/...` = PASS
- `go test -count=1 ./internal/query/...` = PASS
- `go test -count=1 ./internal/corpora/...` = PASS
- `go build ./...` = OK
- `go vet ./testdata/golden/...` = clean
- tools/bench/, NOTICE, bench.yml, repo-root README bench table: byte-identical per D-08 boundary.
- Renames detected: go-* goldens moved as `R100` (byte-identical).
- D-08 boundary: NOTICE, tools/bench/, bench.yml, repo-root README bench table, ts-schema.* untouched.

## Self-Check: PASSED