---
phase: 03-query-engine-mcp-server
fixed_at: 2026-07-11T19:02:00Z
review_path: .planning/phases/03-query-engine-mcp-server/03-REVIEW.md
iteration: 1
findings_in_scope: 7
fixed: 7
skipped: 0
status: all_fixed
---

# Phase 3: Code Review Fix Report

**Fixed at:** 2026-07-11T19:02:00Z
**Source review:** .planning/phases/03-query-engine-mcp-server/03-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 7 (2 Critical, 5 Warning — INFO-01 excluded per instructions, tracked as Phase-4 debt)
- Fixed: 7
- Skipped: 0

## Fixed Issues

### CR-01: `query`/`search`/`callers`/`callees` return an unbounded result set when `--limit`/`limit` is omitted

**Files modified:** `internal/query/search.go`, `internal/query/traverse.go`, `internal/query/search_test.go`, `internal/query/traverse_test.go`
**Commit:** `fdc41ea`
**Applied fix:** Added an unconditional `MaxLimit` post-cap to `Query`, `Search`, `Callees`, and `Callers` — applied after the existing explicit-limit logic, mirroring the pattern already present in `files.go`. Added `TestQueryDefaultCapAtMaxLimit`/`TestSearchDefaultCapAtMaxLimit` (fake reader with `MaxLimit+50` synthetic matches) and `TestCalleesDefaultCapAtMaxLimit`/`TestCallersDefaultCapAtMaxLimit` (new `traverseFakeReader` with `MaxLimit+50` synthetic edges), all proving `limit==0` now caps at `MaxLimit`.

### CR-02: MCP tools accept a client-supplied `path` argument with no confinement to the server's repo root

**Files modified:** `internal/mcp/tools.go`, `internal/mcp/server_test.go`
**Commit:** `6dd5c7b`
**Applied fix:** Added `confineToRepoRoot` (resolves both paths via `filepath.Abs`, rejects via `filepath.Rel` if the client-supplied path is not the repo root or a descendant), wired into `openEngine` before `query.OpenAt` is ever called. Added `TestOpenEnginePathConfinedToRepoRoot`, which indexes a second, independently-valid `.codegraph/` project outside the server's repo root and confirms a `codegraph_status` call targeting it is rejected with an MCP error result (not silently served from the other project) — proving this is a trust-boundary check, not just an "index not found" failure.

### WR-01: Zero-match results marshal `null` instead of `[]` for several JSON array fields

**Files modified:** `internal/query/traverse.go`, `internal/query/files.go`, `internal/query/traverse_test.go`, `internal/query/files_status_test.go`
**Commit:** `96337ae`
**Applied fix:** Changed `CalleesResult.Callees`, `CallersResult.Callers`, and `AffectedResult.AffectedTests` from `var x []T` to `x := []T{}` so a zero-match result marshals `[]`. Added `TestZeroMatchJSONShapesAreEmptyArraysNotNull` asserting all three via `MarshalCalleesJSON`/`MarshalCallersJSON`/`MarshalAffectedJSON`. Also applied the equivalent `entries := []FileEntry{}` change to `files.go` for consistency, plus a dedicated test — note: `FilesResult.Files` carries `json:"files,omitempty"` by design (the struct's documented flat-vs-tree mutually-exclusive shape), so this field was never observably `null` on the wire either before or after the fix; the `omitempty` tag causes the key to be omitted entirely for a zero-match flat result rather than emitted as `null`. Left `omitempty` in place to avoid breaking the existing flat/tree exclusivity test and added a test asserting the `"files"` key, if present, is never literal `null`.

### WR-02: Inconsistent negative-value handling across `--depth`/`--max-files`/`--limit`

**Files modified:** `internal/query/validate.go`, `internal/query/traverse.go`, `internal/query/engine_test.go`, `internal/query/traverse_test.go`
**Commit:** `959579b`
**Applied fix:** `validateLimit` and `validateMaxFiles` now reject a negative input outright (previously only rejected values above their ceiling); added a new `validateDepth` helper applying the same rule to `Impact`'s `--depth`, called before `clampDepth`. `0` still means "caller did not set a value" everywhere (unchanged), and an above-ceiling positive `--depth`/`--max-files` is still silently clamped rather than rejected (preserving `TestImpact`'s existing "absurdly large depth is clamped, not unbounded" contract) — only the negative-value convention was unified, per the review's stated scope. Added negative-value cases to `TestValidateLimit`/`TestValidateMaxFiles`, a new `TestValidateDepth`, and an `Impact`-level `"negative depth is rejected, not silently defaulted"` subtest.

### WR-03: `resolveSourcePath`'s repo-root confinement does not resolve symlinks

**Files modified:** `internal/query/node.go`, `internal/query/node_test.go` (new)
**Commit:** `31679ce`
**Applied fix:** After the existing string-level `Clean`/`Rel` confinement check passes, `resolveSourcePath` now resolves both the repo root and the candidate path via `filepath.EvalSymlinks` and re-verifies confinement against the *resolved* pair (resolved-vs-resolved, not resolved-vs-unresolved, since the repo root itself may sit under a symlinked path — e.g. macOS's `/tmp -> /private/tmp`). Added `internal/query/node_test.go` with `TestResolveSourcePathRejectsSymlinkEscape` (creates an in-repo symlink pointing at a file outside the repo root and confirms `Node` rejects it) and a control case, `TestResolveSourcePathAllowsRegularFileInRepo`.

### WR-04: A single dangling edge/node reference aborts the entire query instead of degrading gracefully

**Files modified:** `internal/query/traverse.go`, `internal/query/explore.go`, `internal/query/node.go`, `internal/query/traverse_test.go`
**Commit:** `3b7a1e1`
**Applied fix:** Every `GetNode` lookup in `Callees`, `Callers`, `Impact`'s BFS, `Affected` (same pattern, in scope though not explicitly cited by line number), `Explore`'s `buildBlastEntry`, and `Node`'s calls/called-by trail now checks `errors.Is(err, graphstore.ErrNotFound)` and skips the dangling edge rather than aborting the whole call. Added a new `traverseFakeReader` (full `graphstore.Reader` fake, also reused for CR-01's default-cap tests) with `TestCalleesSkipsDanglingEdgeInsteadOfFailing`, `TestCallersSkipsDanglingEdgeInsteadOfFailing`, and `TestImpactSkipsDanglingEdgeInsteadOfFailing`, all proving a dangling edge is skipped, not fatal. `explore.go`/`node.go`'s dangling-ref code paths are covered by build+vet+existing-test verification (Tier 1/2) rather than a dedicated fake-reader test — those two call sites need a disk-backed repo root or `IterateFiles` support the existing fakes don't provide, and adding that scaffolding was judged disproportionate to the fix's small, mechanical, identical-pattern change already proven correct at three other call sites in the same commit.

### WR-05: An empty search/query term matches every node in the graph

**Files modified:** `internal/query/search.go`, `internal/query/explore.go`, `internal/query/search_test.go`, `internal/query/explore_test.go` (new)
**Commit:** `094af7a`
**Applied fix:** `Query`, `Search`, and `Explore` now reject an empty or whitespace-only term/query (`strings.TrimSpace(term) == ""`) before any scan runs, mirroring `ValidateKind`'s "reject before any scan" posture. Added `TestQueryRejectsEmptyTerm`/`TestSearchRejectsEmptyTerm` (both assert the scan-counter fake reader sees zero `IterateNodes` calls) and `TestExploreRejectsEmptyQuery`.

## Skipped Issues

None — all 7 in-scope findings (CR-01, CR-02, WR-01–WR-05) were fixed. INFO-01 (`DeleteFileSubgraph` naming) was excluded from scope per the task instructions and remains open, tracked as Phase-4 debt.

## Verification

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./...` — all packages pass
- `CODEGRAPH_WEFT_CORPUS=<local weft checkout> go test ./testdata/golden/...` — `TestGoldenParity` and `TestGoldenFixturesExist` both pass (one pre-existing, documented D-05b informational `t.Logf` divergence in `impact(mergeStyle, depth=2)` node/edge counts — unrelated to this fix set, not a test failure)
- `go test ./internal/graphstore/archtest/...` — `TestNoPackageBypassesGraphStore` passes; `internal/query`/`internal/mcp` still never import `pebble` directly

---

_Fixed: 2026-07-11T19:02:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
