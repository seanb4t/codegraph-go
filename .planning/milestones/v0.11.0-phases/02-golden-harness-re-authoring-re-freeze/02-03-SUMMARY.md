---
phase: 02-golden-harness-re-authoring-re-freeze
plan: 03
type: execute
subsystem: golden-harness
tags: [gocapture, locked-corpora, fail-loud, temp-then-move, mcp-capture, golden-guard]
requires: [02-02]
provides: [extended gocapture, fail-loud locked-corpus tests, golden:regen target, ReFrozenGoldensValid guard]
affects: [testdata/golden/gocapture, testdata/golden/*_test.go, Taskfile.yml, internal/indexer/capability, docs]
status: complete
actuals:
  tokens: 98250
  tasks: 3
  commits: 3
---

# Phase 2 Plan 3: gocapture extension — locked-corpora + behavioral specs, fail-loud tests, golden guard

## One-Liner

Extended `testdata/golden/gocapture` from a behavioral-only capture tool into the sole authority that captures every golden from codegraph-go's own output against the Phase-1-locked corpora and the committed behavioral corpus, adding capture-to-temp-then-move (T-02-03), MCP-surface capture via in-process server, per-spec output directories, a `golden:regen` Taskfile target, and a `TestReFrozenGoldensValid` byte-identity guard over every expected golden. Re-pointed the four locked-corpus tests from env-var-based resolvers with `t.Skip` to hermetic `internal/corpora` resolution that `t.Fatalf`s on absence (D-10). This builds the engine and guard; the actual re-freeze run is plan 02-04.

## Context

This is the "Diff B part 1" — the re-freeze's engine. Builds on 02-01 (rename) and 02-02 (cleanup + move), which are both merged and green. The behavioral corpus now lives at in-tree `corpus/behavioral/`. All locked corpora resolve exclusively through `internal/corpora` `Entry.Dir(root)` / `CorpusRoot()` / `LockedEntries(m)` — no hardcoded SHAs or paths anywhere.

## Tasks

### Task 1: Extend gocapture (D-05/FIXT-06)

**Files:** `testdata/golden/gocapture/main.go`

**Changes:**
- Added locked-corpus resolver helper that loads manifest via `internal/corpora.Load`, resolves through a committed language-to-slug map (`go->hugo, tsjs->hugo, java->guava, csharp->serilog, python->requests`), and locates the source directory via `Entry.Dir(CorpusRoot())` — never hardcodes a SHA or path.
- Added per-corpus `outDir` field: locked corpora write to `testdata/golden/corpus/<slug>/`; behavioral corpus writes to repo-root `corpus/behavioral/` (two-hop `filepath.Dir` form).
- Changed `resolveSource` contract from `func() (string, string)` (path + skip reason) to `func() (string, error)` — fail-closed: missing mandatory source causes hard error, not skip-warn.
- Converted `writeCapture` to temp-then-move: writes to `os.CreateTemp`, asserts non-empty + `{` marker before `os.Rename` onto committed path. Never leaves a bare or half-written golden.
- Added MCP-surface capture: stages the corpus in a temp directory, indexes it via `indexer.Run`, then captures `explore` and `node` output through `internalmcp.BuildServer` + in-process go-sdk client — no TS dependency.
- Committed per-corpus expected golden set: 6 goldens per locked corpus (`{explore, node, explore-multi, node-multi, explore-mcp, node-mcp}`).
- Retired skip-warn contract from the old weft/colbymchenry/online-clone era.

**Verify:** `go build ./testdata/golden/gocapture/... && go vet ./testdata/golden/gocapture/... && go test -count=1 ./internal/corpora/...` — all pass.

**Commit:** `7a8e789`

### Task 2: Re-point locked-corpus tests to fail-loud hermetic resolution (D-10)

**Files:** `testdata/golden/behavioral_test.go`, `behavioral_java_test.go`, `behavioral_tsjs_test.go`, `behavioral_csharp_test.go`, `behavioral_python_test.go`, `internal/indexer/capability/matrix.go`, `docs/LANGUAGE-CAPABILITY-MATRIX.md`

**Changes:**
- Added shared `lockedCorpusDir(t, language)` helper in `behavioral_test.go` that loads manifest via `internal/corpora`, resolves language slug via the committed map, and calls `t.Fatalf` on any failure — never `t.Skip`, never env-var default.
- Added `TestPriorityLanguagesResolveToLockedCorpus` (H3 positive guard): all 5 priority languages (go, java, csharp, python, tsjs) resolve, tsjs via hugo's JS files.
- Filled in `TestCorpusBehaviorLockedCorpora` with per-spec subtests (hugo/guava/serilog/requests) running shape-only explore/node assertions.
- Replaced `resolveJavaCorpus`/`resolveTSJSCorpus`/`resolveCSharpCorpus`/`resolvePythonCorpus` in the four language test files with `lockedCorpusDir` — the old env-var/sibling-checkout/t.Skip functions are deleted.
- Updated CLI==MCP case lists (`TestExploreCLIMatchesMCP`, `TestNodeCLIMatchesMCP`) to include behavioral + locked corpora.
- Re-synced `matrix.go` and `docs/LANGUAGE-CAPABILITY-MATRIX.md` prose from "self-skips / source-as-specification fallback" to "fail-loud locked-corpus resolution" (MED: prose staleness).
- Fixed `copyDir` to handle symlinks (requests corpus has a `ca -> ../../expired/ca/` directory symlink).

**Verify:** `go test -count=1 ./testdata/golden/... -run 'TestPriorityLanguagesResolveToLockedCorpus|TestCorpusBehaviorLockedCorpora|TestCorpusBehavior_Java|TestCorpusBehavior_TSJS|TestCorpusBehavior_CSharp|TestCorpusBehavior_Python' -v` — all pass. `go test -count=1 ./internal/indexer/capability/... -run TestMatrix_DocMirrorsDescriptor` — pass.

**Acceptance:**
- No `t.Skip(` calls in golden code (only comment references to retired approach).
- No `CODEGRAPH_{LANG}_CORPUS` references in executable code (only comments).
- `go vet ./testdata/golden/...` clean.

**Commit:** `3a71190`

### Task 3: Add byte-identity guard and golden:regen target (A3/T-02-03)

**Files:** `testdata/golden/golden_test.go`, `Taskfile.yml`

**Changes:**
- Added `TestReFrozenGoldensValid`: enumerates expected golden set from the committed `expectedGoCaptures` spec table (never `filepath.Glob`). For each expected golden: asserts existence, non-empty, `{` marker, parseable `goldenCapture` envelope with non-empty `Output`, and positively asserts a verified count (H5).
- Guard ships with behavioral expected set active and non-vacuous at 03 (2 goldens verified); locked-corpus entries are added by 02-04 after capture.
- Added `golden:regen` Taskfile target: `deps` on `corpora:fetch`, runs `task corpora:assert`, then `go run ./testdata/golden/gocapture` — the one-command re-freeze entrypoint.

**Verify:** `go test -count=1 ./testdata/golden/... -run TestReFrozenGoldensValid -v` — 2/2 behavioral goldens verified, passes. `go build ./... && go vet ./testdata/golden/...` — clean.

**Commit:** `b14a147`

## Key Decisions

| Decision | Rationale |
|----------|-----------|
| Language map stored as Go var, not config file | Shared across gocapture, tests, and guard — committed, typed, reviewable. Duplicated in both main.go and behavioral_test.go (same package boundary). |
| `copyDir` handles symlinks explicitly | requests corpus has a `ca -> ../expired/ca/` directory symlink; `os.ReadFile` on symlink to directory fails. |
| `resolveSource` returns error, not skip reason | Fail-closed contract for mandatory sources — no path left to silently skip. |
| MCP capture uses temp indexed repo | MCP handlers use `query.OpenAt` which requires an on-disk store; staging ensures the store exists. |
| Behavioral outDir uses two-hop repo-root form | `filepath.Dir(filepath.Dir(goldenDir))` ensures goldens land at repo-root `corpus/behavioral/`, not `testdata/corpus/behavioral/` (H2-2 fix). |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed `copyDir` to handle directory symlinks**

- **Found during:** Task 2 test execution
- **Issue:** The requests locked corpus has a symlink `ca -> ../../expired/ca/` pointing to a directory. `filepath.WalkDir` reports the symlink as a non-directory entry, but `os.ReadFile` following it finds a directory and fails with "is a directory".
- **Fix:** Added symlink detection via `os.Lstat` in both `behavioral_test.go`'s and `gocapture/main.go`'s `copyDir` functions. For directory symlinks, recursively walk the resolved target.
- **Files modified:** `testdata/golden/behavioral_test.go`, `testdata/golden/gocapture/main.go`

## Threat Surface Scan

No new security-relevant surface introduced beyond what the plan documents: all paths read locked corpora from `internal/corpora` (trusted after integrity assert), and MCP capture uses the in-process server (no new network endpoints).

## Verification

- `go build ./testdata/golden/gocapture/...` — clean
- `go vet ./testdata/golden/...` — clean
- `go test -count=1 ./internal/corpora/...` — pass
- `go test -count=1 ./internal/indexer/capability/...` — pass (including `TestMatrix_DocMirrorsDescriptor`)
- `go test -count=1 ./testdata/golden/... -run 'TestPriorityLanguagesResolveToLockedCorpus|TestCorpusBehaviorLockedCorpora|TestReFrozenGoldensValid' -v` — all pass
- `go test -count=1 ./testdata/golden/... -run 'TestCorpusBehavior_Java|TestCorpusBehavior_TSJS|TestCorpusBehavior_CSharp|TestCorpusBehavior_Python' -v` — all pass
- `go test -count=1 ./testdata/golden/... -run 'TestExploreCLIMatchesMCP|TestNodeCLIMatchesMCP' -v` — all pass
- `task golden:regen --help >/dev/null 2>&1` — resolves
- `task --list | grep golden:regen` — shows target

## Self-Check

All verification steps completed successfully. All three tasks committed atomically with conventional commit format.

## Output

Plan 02-03 produces the extended gocapture tool, fail-loud locked-corpus tests, `TestReFrozenGoldensValid` guard, and `golden:regen` target. The re-frozen golden bytes themselves are NOT produced here — that is plan 02-04's job.