---
phase: 01-engine-seam-wire-protocol-secure-transport
plan: 06
subsystem: api
tags: [protobuf, git, indexer, meta, query-engine, wire-protocol]

# Dependency graph
requires:
  - phase: 01-01
    provides: "buf-driven proto codegen pipeline (task proto:gen) covering both internal/schema/graph.proto and internal/uiproto/uiv1/ui.proto"
provides:
  - "schema.Meta field 8 (string commit_sha), additive per D-02a, with a permanent reserved 50-59 range for future graph-level provenance"
  - "schema.IndexedCommitSHA(m) — the nil-safe, empty-means-absent accessor"
  - "internal/indexer's resolveHeadCommitSHA(repoPath) — a bounded, never-erroring git HEAD resolver accepting both SHA-1 (40) and SHA-256 (64) lowercase hex"
  - "HEAD resolved exactly once per operation (full index run, or one Sync call) and threaded to all three PutMeta write sites"
  - "(*Engine).IndexMeta() — the wire layer's only path to the stored Meta record, deliberately outside status.go"
affects: ["01-09", "phase-3-permalinks (BRW-09)"]

# Actuals (#2632)
actuals:
  tokens: 10707
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "gitExecLookPath: an unexported, test-only exec.LookPath seam with no exported setter, mirroring internal/graphstore's openLockRetrySleep — used to make the no-git-binary path deterministically testable without emptying the process's real PATH"
    - "Resolve-once-and-thread: a value with a hard consistency requirement (HEAD must not move mid-operation) is resolved exactly once at the operation's entry point and threaded down through every constructor that needs it, rather than re-resolved at each write site"

key-files:
  created:
    - internal/indexer/commit.go
    - internal/indexer/commit_test.go
    - internal/schema/meta_commit_test.go
    - internal/query/meta_test.go
  modified:
    - internal/schema/graph.proto
    - internal/schema/graph.pb.go
    - internal/schema/meta.go
    - internal/indexer/resolve.go
    - internal/indexer/sync.go
    - internal/query/engine.go
    - internal/indexer/pipeline.go
    - internal/indexer/pipeline_test.go
    - internal/indexer/resolve_test.go

key-decisions:
  - "Checkpoint ruling (pre-ruled by the maintainer, recorded verbatim below) — locked string commit_sha = 8 AND additionally reserved 50 to 59 on Meta for future graph-level provenance, mirroring Node/Edge's identical clauses"
  - "SHA validation accepts 40 OR 64 lowercase hex (Git SHA-1 and SHA-256), with negatives at 39/41/63/65/uppercase — a 40-only check would silently record an absent SHA on a SHA-256 repository"
  - "HEAD resolution lives in pipeline.go's run() (full index run) and sync.go's Sync() (incremental), never inside Resolve/writeGraph itself — repoRoot is only in scope at the operation's entry point"
  - "(*Engine).IndexMeta lives in engine.go, not status.go — D-06 keeps codegraph status untouched; the accessor's placement outside status.go records that boundary in the file layout itself"

patterns-established:
  - "Resolve-once-and-thread for any value with a within-operation consistency requirement (HEAD, in this case) — resolve at the entry point, never at the write site"

requirements-completed: [ENG-04]

coverage:
  - id: D1
    description: "schema.Meta carries the indexed commit SHA as additive field 8, following has_file_index's precedent; absent/empty degrades gracefully, never an error"
    requirement: "ENG-04"
    verification:
      - kind: unit
        ref: "internal/schema/meta_commit_test.go#TestMetaCommitSHARoundTrips"
        status: pass
      - kind: unit
        ref: "internal/schema/meta_commit_test.go#TestMetaCommitSHAAbsentDegrades"
        status: pass
      - kind: unit
        ref: "internal/schema/meta_commit_test.go#TestKnownMetaFieldNumbersAreStable"
        status: pass
    human_judgment: false
  - id: D2
    description: "HEAD is resolved exactly once per operation (full index run, incremental sync) and stamped at all three PutMeta write sites; accepts both SHA-1 and SHA-256 lengths; degrades to empty on any failure, never an error"
    requirement: "ENG-04"
    verification:
      - kind: unit
        ref: "internal/indexer/commit_test.go#TestResolveHeadCommitSHA"
        status: pass
      - kind: unit
        ref: "internal/indexer/commit_test.go#TestResolveHeadCommitSHAOnNonGitTree"
        status: pass
      - kind: unit
        ref: "internal/indexer/commit_test.go#TestResolveHeadCommitSHAWithNoGitBinary"
        status: pass
      - kind: unit
        ref: "internal/indexer/commit_test.go#TestHeadIsResolvedOncePerOperation"
        status: pass
      - kind: unit
        ref: "internal/indexer/commit_test.go#TestIndexRunStampsHeadCommitSHA"
        status: pass
    human_judgment: false
  - id: D3
    description: "(*Engine).IndexMeta gives the wire layer a real path to the stored Meta record, degrading to (nil, nil) on a graph with no Meta"
    requirement: "ENG-04"
    verification:
      - kind: unit
        ref: "internal/query/meta_test.go#TestIndexMetaCarriesTheStoredMeta"
        status: pass
      - kind: unit
        ref: "internal/query/meta_test.go#TestIndexMetaOnAGraphWithNoMeta"
        status: pass
    human_judgment: false
  - id: D4
    description: "codegraph status output is unchanged, proven three independent ways: status.go untouched by diff, StatusResult's field set pinned, and a before/after --json capture diffed"
    requirement: "ENG-04"
    verification:
      - kind: unit
        ref: "internal/query/meta_test.go#TestStatusResultFieldSetIsUnchanged"
        status: pass
      - kind: other
        ref: "git diff --stat <merge-base HEAD main>..HEAD -- internal/query/status.go (empty, positive-controlled against internal/indexer/ non-empty over the same range)"
        status: pass
      - kind: manual_procedural
        ref: "codegraph status --json captured for a fixture repo from both the phase-base binary and this plan's binary; diffed by hand (see below)"
        status: pass
    human_judgment: false

duration: 9m2s (wall-clock between the two feat commits; excludes read/verification time)
completed: 2026-08-23
status: complete
---

# Phase 1 Plan 6: Commit-Aware Meta & the IndexMeta Accessor Summary

**`schema.Meta` gains an additive commit-SHA field (accepting both Git SHA-1 and SHA-256 lengths), resolved once per index/sync operation and stamped at all three write sites, with `(*Engine).IndexMeta()` giving the UI wire layer its first real path to that metadata — while `codegraph status`'s output stays byte-identical, proven three independent ways.**

## Performance

- **Duration:** ~9 min wall-clock between the Task 1 and Task 2 commits (excludes read/planning/verification time)
- **Started:** 2026-08-23T11:01:44-04:00 (Task 1 commit)
- **Completed:** 2026-08-23T11:10:46-04:00 (Task 2 commit)
- **Tasks:** 2 (plus one pre-ruled checkpoint, recorded below, not paused on)
- **Files modified:** 13 (4 created, 9 modified — includes a necessary deviation, see below)

## Checkpoint Ruling (recorded verbatim, not paused on)

Per the orchestrator's instruction, this checkpoint was **pre-ruled by the maintainer** and execution proceeded without pausing. The ruling had two parts:

1. **Lock `Meta` field 8 — APPROVED as planned.** `string commit_sha = 8;` following the `has_file_index = 7` precedent (D-05, ENG-04).
2. **ADDITIONALLY reserve a future range** — `reserved 50 to 59;` on `Meta`, mirroring the identical clause already carried by `Node` (`graph.proto:65`, pre-existing) and `Edge` (`:96`, pre-existing), with a graph-level provenance comment (`// future: team-scale provenance (central server, CI-distributed indexes)`) distinct from their per-record one.

Both halves are recorded as **one-way** under D-02a: once shipped, neither field 8 nor the reserved range can be renumbered or reused. Both were implemented exactly as ruled in Task 1's commit (`2a5d3c7`).

## Accomplishments

- `schema.Meta` field 8 (`string commit_sha`), additive per D-02a, with the mandated `reserved 50 to 59;` clause, regenerated via `task proto:gen` with a diff confined to exactly the new field, its getter, and the raw descriptor bytes — zero drift elsewhere, including zero diff to the `uiproto` surface `proto:gen` also regenerates
- `schema.IndexedCommitSHA(m)` — nil-safe, empty-means-absent accessor mirroring `IsCurrentSchemaVersion`'s shape
- `resolveHeadCommitSHA(repoPath)` (`internal/indexer/commit.go`): shells out to `git -C <repoPath> rev-parse HEAD` via a fixed `exec.CommandContext` argument vector with a 5s timeout; accepts exactly 40 or 64 lowercase hex characters (SHA-1 or SHA-256), rejects 39/41/63/65/uppercase; returns `""` — never an error — on every failure path
- HEAD resolved exactly once per operation and threaded to all three `PutMeta` sites: `internal/indexer/pipeline.go`'s `run()` (full index run, before Discover/Extract) and `internal/indexer/sync.go`'s `Sync()` (after the backfill check, reused by whichever of its two mutually-exclusive write sites fires)
- `(*Engine).IndexMeta()` (`internal/query/engine.go`) — the UI wire layer's only real path to the stored `Meta` record; returns `(nil, nil)` on a graph with no Meta, matching `HasFileIndex`'s absent-graph contract
- Three independent proofs that `codegraph status` stays byte-identical: `status.go` unmodified over the phase-base range, `StatusResult`'s reflected field set pinned by a literal fixture, and a before/after `--json` capture diffed (see below)

## Task Commits

1. **Task 1: `Meta` field 8, its absent-value contract, and a subset-stable field-number guard** — `2a5d3c7` (feat)
2. **Task 2: Resolve HEAD once, stamp it at all three sites, and expose the metadata to the wire** — `c0afd89` (feat)

## Files Created/Modified

- `internal/schema/graph.proto` — `Meta` gains `string commit_sha = 8` plus `reserved 50 to 59`
- `internal/schema/graph.pb.go` — regenerated: `Meta.CommitSha`, `(*Meta).GetCommitSha()`, updated raw descriptor
- `internal/schema/meta.go` — `IndexedCommitSHA(m) (string, bool)`
- `internal/schema/meta_commit_test.go` — `TestMetaCommitSHARoundTrips`, `TestMetaCommitSHAAbsentDegrades` (from raw wire bytes genuinely omitting field 8), `TestKnownMetaFieldNumbersAreStable` (subset-based descriptor guard)
- `internal/indexer/commit.go` — `resolveHeadCommitSHA`, `gitExecLookPath` seam, `isLowercaseHexCommitSHA`
- `internal/indexer/commit_test.go` — the five commit-resolution test functions (8 total subtests across two of them)
- `internal/indexer/resolve.go` — `Resolve`/`writeGraph` gain a threaded `commitSHA` parameter; `writeGraph` stamps `meta.CommitSha`
- `internal/indexer/sync.go` — `Sync` resolves HEAD once after the backfill check; both `PutMeta` sites stamp `CommitSha`
- `internal/query/engine.go` — `(*Engine).IndexMeta()`
- `internal/query/meta_test.go` — `TestIndexMetaCarriesTheStoredMeta`, `TestIndexMetaOnAGraphWithNoMeta`, `TestStatusResultFieldSetIsUnchanged`
- `internal/indexer/pipeline.go` — deviation, see below: `resolveFunc`'s type and `run()`'s call site thread the resolved commit SHA
- `internal/indexer/pipeline_test.go` — deviation, see below: stub `failingResolve` signature updated to match
- `internal/indexer/resolve_test.go` — deviation, see below: three pre-existing call sites (`writeGraph` x2, `Resolve` x1) updated to pass `""` for the new parameter

## Decisions Made

- **SHA validation widened to two lengths, not one.** `resolveHeadCommitSHA` accepts exactly 40 OR 64 lowercase hex characters (Git's SHA-1 and SHA-256 object formats). A 40-only check would silently record an absent commit on a SHA-256 repository — the failure mode is invisible (indistinguishable from a pre-upgrade graph by design), so it would have surfaced downstream in Phase 3 as a permalink that never renders. Verified locally: the sandbox's git (2.55.0) supports `--object-format=sha256`, so `TestResolveHeadCommitSHA`'s `sha256-64-hex` subtest ran for real rather than skipping.
- **HEAD resolved at the operation's entry point, never at the write site.** `run()` (pipeline.go) and `Sync()` (sync.go) each resolve HEAD exactly once, before any `PutMeta` construction, and thread the value down. This is what makes `TestHeadIsResolvedOncePerOperation`'s "exactly 1" assertion possible and prevents HEAD moving mid-run (a rebase, a concurrent commit) from producing two different recorded commits within one operation.
- **`(*Engine).IndexMeta` placed in `engine.go`, not `status.go`.** D-06 requires `codegraph status`'s output stay untouched; keeping the new accessor entirely out of `status.go` records that boundary in the file layout itself, not just in a comment.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 — Blocking issue] Threading `commitSHA` through `Resolve` required touching `pipeline.go`, `pipeline_test.go`, and `resolve_test.go` beyond this plan's declared 10 `files_modified`**

- **Found during:** Task 2, while implementing "resolve HEAD once at the top of the full index run and thread it down to writeGraph"
- **Issue:** The plan's own action text requires HEAD be resolved once at the top of the full index run and carried down to `writeGraph`. Tracing the actual call graph: `writeGraph` is called only by `Resolve` (`internal/indexer/resolve.go`, in scope), and `Resolve` is called only by `run()` (`internal/indexer/pipeline.go`, **not** in the plan's declared files). `repoRoot` — the value `resolveHeadCommitSHA` needs — is a parameter of `run()`/`Run()` and is never threaded into `Resolve`'s existing three-parameter signature (`store, results, modulePath`); `goextract.FileResult` carries no absolute path or repo-root information for `Resolve` to recover it independently. There is no way to satisfy "resolve once at the top of the full index run, thread to writeGraph" without either (a) changing `Resolve`'s signature — which necessarily changes its sole caller `run()` and the `resolveFunc` type both defined in `pipeline.go` — or (b) resolving HEAD from an unrelated/fragile source (CWD, storeDir path-reverse-engineering), which would be both wrong and a worse deviation. `Sync`'s own two write sites needed no such change — `Sync(repoRoot, storeDir, opts)` already carries `repoRoot` as its own parameter, so that half of the task stayed entirely within the declared files.
- **Fix:** Added a `commitSHA string` parameter to `resolveFunc`'s type and to `Resolve`/`writeGraph`'s signatures (all files already in scope except `pipeline.go`); `run()` in `pipeline.go` now resolves HEAD once at its own top (before Discover/Extract) and passes the value through to `resolve(...)`. This forced two purely mechanical, no-behavior-change test-file edits to keep the build compiling: `pipeline_test.go`'s stub `failingResolve` closure gained the matching parameter, and `resolve_test.go`'s three pre-existing direct calls to `writeGraph`/`Resolve` were updated to pass `""` for the new parameter (their assertions are unchanged; none of them test commit-SHA behavior).
- **Files modified beyond the declared 10:** `internal/indexer/pipeline.go`, `internal/indexer/pipeline_test.go`, `internal/indexer/resolve_test.go`
- **Verification:** `go build ./...` and `go vet ./...` clean; full `task test:unit` and `task test:golden` both pass (`attempted=26 completed=26 matched=26`); `TestHeadIsResolvedOncePerOperation` and `TestIndexRunStampsHeadCommitSHA` both exercise the new `run()` code path directly and pass.
- **Committed in:** `c0afd89` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 3 — blocking issue, forced by the plan's own literal requirement)
**Impact on plan:** Necessary to make the task's stated design ("resolve once at the top of the full index run") buildable at all; no scope creep — the added edits are pure signature-threading with zero behavior change to any pre-existing test assertion.

## Issues Encountered

None beyond the `pipeline.go` scope gap documented above as a deviation.

## User Setup Required

None — no external service configuration required.

## `codegraph status --json` Before/After Capture

Built two binaries — one from the phase-base commit (`git merge-base HEAD main` = `3609fc0`, via `git archive` into a scratch tree, never touching the current worktree), one from this plan's final state — and ran `init` + `status --json` against an identical single-file Go fixture repo with both:

```
before: {"initialized":true, ..., "dbSizeBytes":4056, ...}
after:  {"initialized":true, ..., "dbSizeBytes":4101, ...}
```

The **only** difference across the entire JSON output is `dbSizeBytes` (4056 → 4101, a 45-byte increase from the new `commit_sha` string now genuinely stored on disk in every `Meta` record). Every key, every value, and key ordering are otherwise byte-identical. `dbSizeBytes` is `status.go`'s own documented volatile-but-shown field (D-08) — a real on-disk-byte measurement that necessarily grows when a new field is added to the on-disk `Meta` record, not a shape leak of the new field into `codegraph status`'s output. No new key appeared; `commit_sha`/`CommitSha` is not present anywhere in either capture.

## `internal/schema/graph.pb.go` Regeneration Diff

```
before: (Meta had fields 1-7, no reserved clause, size_bytes-terminated message)
after:  + CommitSha string field 8 + its getter + updated raw descriptor bytes
        (encodes the new field AND the reserved 50-59 range: rawDesc segment `J\x04\b2\x10<` = reserved [50,60))
```

`git status --porcelain internal/schema/` confirmed empty after Task 1's commit; no other file under `internal/schema/` changed.

## Known Stubs

None. Every deliverable is wired to real, working code: `resolveHeadCommitSHA` genuinely shells out to `git`, all three `PutMeta` sites genuinely stamp the resolved value, and `IndexMeta` genuinely reads the store's Meta record — proven end-to-end by `TestIndexRunStampsHeadCommitSHA` against a real temp git checkout compared to an independently-run `git rev-parse HEAD`.

## Next Phase Readiness

- Plan 01-09's Status RPC now has a real, tested data path from `Meta.commit_sha` to `(*Engine).IndexMeta()` — the gap the phase's first cross-AI review flagged as making that plan's must-have unachievable is closed.
- Phase 3's `BRW-09` (GitHub permalinks pointing at the indexed commit) is unblocked: the field exists, is populated on every write path, and degrades gracefully on a non-git checkout or a pre-upgrade graph.
- `codegraph status`'s CLI output is unchanged, proven three independent ways — no golden needs updating, no CLI consumer needs to change.
- No blockers for downstream plans in this wave.

## Self-Check: PASSED

All 13 claimed files confirmed present on disk (`internal/schema/graph.proto`, `internal/schema/graph.pb.go`, `internal/schema/meta.go`, `internal/schema/meta_commit_test.go`, `internal/indexer/commit.go`, `internal/indexer/commit_test.go`, `internal/indexer/resolve.go`, `internal/indexer/sync.go`, `internal/query/engine.go`, `internal/query/meta_test.go`, `internal/indexer/pipeline.go`, `internal/indexer/pipeline_test.go`, `internal/indexer/resolve_test.go`); both claimed commit hashes (`2a5d3c7`, `c0afd89`) confirmed present in `git log --oneline --all`. No missing items.

---
*Phase: 01-engine-seam-wire-protocol-secure-transport*
*Completed: 2026-08-23*
