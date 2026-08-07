---
phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface
plan: 04
subsystem: api
tags: [mcp, go-sdk, modelcontextprotocol, go-mod-tidy, govulncheck, sbom, archtest, dependency-audit]

requires:
  - phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface
    provides: "plan 02-01's production migration of internal/mcp onto modelcontextprotocol/go-sdk v1.7.0 behind the unchanged Server seam, leaving mark3labs/mcp-go in go.mod deliberately for this plan to remove"
provides:
  - "github.com/mark3labs/mcp-go absent from go.mod and go.sum — SDK-03 satisfied"
  - "The four remaining mark3labs client call sites (testdata/golden's two in-process CLI==MCP harness functions, three test/integration stdio subprocess drivers) ported onto go-sdk's mcp.NewInMemoryTransports / mcp.CommandTransport + mcp.NewClient"
  - "Both archtest non-vacuity self-tests (internal/mcp/archtest, internal/cli/archtest) re-pointed at go-sdk identifiers that genuinely resolve, closing the gap where the SDK-02 self-test only ever proved the guard could catch the departing SDK, never the one being adopted"
  - "Dependency closure re-audited: go mod tidy run against the real tree, govulncheck green over reachable code, SBOM (syft, spdx-json) confirmed naming go-sdk not mark3labs"
  - "NOTICE attribution corrected to modelcontextprotocol/go-sdk (Apache-2.0)"
affects: [02-05]

actuals:
  tokens: 9942
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "In-process MCP test harnesses outside internal/mcp (testdata/golden) reuse the exact mcp.NewInMemoryTransports + goroutine-run-server + Client.Connect shape internal/mcp/server_test.go's newTestSession established, rather than inventing a second pattern"
    - "test/integration subprocess drivers own their *exec.Cmd directly (exec.CommandContext(ctx, ...) + mcp.CommandTransport{Command: cmd}) instead of going through an SDK-provided stdio-client-with-options wrapper — stderr capture, env, and cwd are wired onto the Cmd before Connect, not through a transport option struct"
    - "archtest non-vacuity self-tests plant a real, syntactically-used identifier from the SDK actually in the tree (not a retired one), keeping the RED-demonstration protocol meaningful across an SDK swap"

key-files:
  created: []
  modified:
    - testdata/golden/golden_parity_test.go
    - test/integration/watch_default_test.go
    - test/integration/watch_live_sync_test.go
    - test/integration/worktree_notice_test.go
    - internal/mcp/archtest/protocol_version_test.go
    - internal/mcp/archtest/protocol_version_selftest_test.go
    - internal/cli/archtest/mcp_sdk_selftest_test.go
    - go.mod
    - go.sum
    - NOTICE

key-decisions:
  - "Deviation (Rule 1, mechanical/unavoidable): testdata/golden/golden_parity_test.go's t.Errorf/t.Fatalf call-site count dropped from 114 to 112, not unchanged as the plan's acceptance criterion literally asked. Root cause: go-sdk's Client.Connect atomically performs both connection establishment AND the initialize handshake, collapsing mark3labs' two separately-fallible steps (NewInProcessClient() then a later Initialize() call) into one — the same consolidation SDK-01's own internal/mcp/server_test.go newTestSession helper already established as this repo's pattern. No test ASSERTION (IsError checks, content/byte comparisons) was removed or weakened; every one is unchanged in both count and strength — only the setup-error-check plumbing shrank because the API surface it wrapped shrank."
  - "go mod tidy required the -e flag to complete. Reproduced identically against the untouched pre-task go.mod (confirmed by checking out the original file and re-running before making any change): alex-pinkus/tree-sitter-swift's own _test.go file imports github.com/tree-sitter/tree-sitter-swift/bindings/go, a different, unresolvable module — a pre-existing upstream quirk this repo has apparently never surfaced before because no CI job or Taskfile target has ever run go mod tidy to completion. Out of scope for SDK-03 (unrelated to mark3labs/go-sdk); -e lets tidy proceed past that one unresolvable test-only path. Two consecutive -e runs produced an identical, stable go.mod/go.sum diff, confirming the result converged rather than partially completing."
  - "NOTICE's go-sdk license tag is Apache-2.0, not MIT: the SDK's own LICENSE file documents an in-progress MIT->Apache-2.0 relicensing, with Apache-2.0 governing all new contributions going forward. Apache-2.0 is the more accurate forward-looking tag for a project consuming the current v1.7.0 release."

patterns-established:
  - "When an archtest self-test plants an SDK identifier to prove a guard fires, re-point the planted identifier at the SDK actually resolvable in the tree BEFORE removing the retired SDK from go.mod — never after, since a stale overlay stops type-checking for an unrelated reason the moment the import can no longer resolve and the self-test starts failing without proving anything."

requirements-completed: [SDK-03]

coverage:
  - id: D1
    description: "github.com/mark3labs/mcp-go is absent from go.mod and go.sum; go build ./... and go vet ./... both exit 0"
    requirement: "SDK-03"
    verification:
      - kind: unit
        ref: "rg -c mark3labs go.mod go.sum (zero matches in both)"
        status: pass
      - kind: unit
        ref: "go build ./... && go vet ./..."
        status: pass
    human_judgment: false
  - id: D2
    description: "govulncheck runs green over the post-swap dependency closure; the SBOM path (syft) names go-sdk instead of mark3labs"
    requirement: "SDK-03"
    verification:
      - kind: integration
        ref: "task vuln (GOWORK=off go tool -modfile=go.tool.mod govulncheck ./...) — 'Your code is affected by 0 vulnerabilities'"
        status: pass
      - kind: integration
        ref: "syft <local release build> -o spdx-json=... ; package list contains github.com/modelcontextprotocol/go-sdk v1.7.0, zero mark3labs matches"
        status: pass
    human_judgment: false
  - id: D3
    description: "VRFY-02's and SDK-02's non-vacuity self-tests plant go-sdk identifiers that genuinely resolve, and both were observed failing when their production predicate is neutered"
    requirement: "SDK-03"
    verification:
      - kind: unit
        ref: "go test ./internal/mcp/archtest/... ./internal/cli/archtest/... -count=1 -v (all pass)"
        status: pass
      - kind: unit
        ref: "manual RED demonstration: isExternalProtocolVersionConstant neutered to return false -> TestProtocolVersionGuardCatchesOverlaidViolation fails; forbiddenMCPSDKPrefixes emptied -> TestInternalCLIImportsNoMCPSDK_PlantedImportIsError fails; both reverted and confirmed green"
        status: pass
    human_judgment: false
  - id: D4
    description: "The four remaining mark3labs client call sites drive the server through go-sdk instead, with every existing assertion preserved in intent"
    requirement: "SDK-03"
    verification:
      - kind: integration
        ref: "task test:golden (testdata/golden) and task test:integration (test/integration) both exit 0"
        status: pass
    human_judgment: false

duration: ~1h (git commit timestamps span a compressed 7min window; session wall-clock across research/read/edit/verify was longer, exact start not machine-recorded)
completed: 2026-08-06
status: complete
---

# Phase 2 Plan 4: SDK Migration — official go-sdk on the existing surface Summary

**`github.com/mark3labs/mcp-go` is gone from `go.mod`/`go.sum`; the last four client call sites and both archtest non-vacuity self-tests now drive against `modelcontextprotocol/go-sdk` v1.7.0, and the dependency closure re-audited green through `govulncheck` and a local `syft` SBOM.**

## Performance

- **Duration:** ~1h (research reading + three tasks + verification)
- **Completed:** 2026-08-06
- **Tasks:** 3
- **Files modified:** 10

## Accomplishments

- `testdata/golden/golden_parity_test.go`'s two in-process CLI==MCP byte-identity harness functions (`callExploreViaMCP`, `callNodeViaMCPWithArgs`) moved from mark3labs' `NewInProcessClient` to go-sdk's `mcp.NewInMemoryTransports` pair, via a new `newGoldenSession` helper mirroring `internal/mcp/server_test.go`'s `newTestSession` pattern.
- The three `test/integration` subprocess-driven tests (`watch_default_test.go`, `watch_live_sync_test.go`, `worktree_notice_test.go`) moved from mark3labs' stdio client + `transport.WithCommandFunc` to `mcp.CommandTransport{Command: exec.CommandContext(ctx, ...)}` + `mcp.NewClient(...).Connect`, preserving every subprocess-spawn property (real binary, real stdio, env vars, cwd) these tests exist to exercise.
- Both archtest non-vacuity self-tests re-pointed at go-sdk identifiers that genuinely resolve and type-check: `internal/mcp/archtest/protocol_version_selftest_test.go` now plants `mcp.MetaKeyProtocolVersion`; `internal/cli/archtest/mcp_sdk_selftest_test.go` now plants an import of `github.com/modelcontextprotocol/go-sdk/mcp` plus a reference to `mcp.NewServer`. Both demonstrated RED against a neutered production predicate, then reverted to green.
- `github.com/mark3labs/mcp-go` removed from `go.mod`'s require block; `go mod tidy` run against the real tree (with `-e`, required to proceed past a pre-existing, unrelated upstream quirk — see Deviations); `go.sum` correspondingly pruned.
- `govulncheck` (`task vuln`) exits 0 — zero reachable vulnerabilities. `syft` run locally against a freshly built binary confirms the SBOM names `github.com/modelcontextprotocol/go-sdk v1.7.0` and zero `mark3labs` matches.
- `NOTICE`'s third-party attribution updated from mark3labs/mcp-go (MIT) to modelcontextprotocol/go-sdk (Apache-2.0).

## Task Commits

1. **Task 1: The last four client call sites onto go-sdk's client** - `01f854a` (feat)
2. **Task 2: Re-point both archtest self-tests at go-sdk identifiers that exist, and prove each still fires** - `7d24831` (test)
3. **Task 3: mark3labs out of go.mod, closure re-audited, attribution corrected** - `2220dee` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `testdata/golden/golden_parity_test.go` — `newGoldenSession` helper added; `callExploreViaMCP`/`callNodeViaMCPWithArgs` ported to go-sdk's in-memory transport; `initMCPClient` removed (go-sdk's `Client.Connect` performs the handshake internally); `mcpResultText` ported from `mcp.AsTextContent` to a `*mcp.TextContent` type assertion.
- `test/integration/watch_default_test.go` — `TestDefaultWatchHandshakePrompt`'s `ListTools(ctx, mcp.ListToolsRequest{})` call ported to `ListTools(ctx, nil)`; `TestNoWatchEnvDisablesViaStderr` rewritten to own its `*exec.Cmd` directly, wiring `cmd.Stderr` to the package's existing `syncBuffer` type (`mcp_stdout_purity_test.go`) instead of mark3labs' `client.GetStderr` seam.
- `test/integration/watch_live_sync_test.go` — `newServeClientWithEnv` rewritten onto `mcp.CommandTransport` + `mcp.NewClient(...).Connect`; `TestLiveEditAutoSyncReachesExplore`'s `CallTool` call ported to the pointer-params shape.
- `test/integration/worktree_notice_test.go` — `newServeClient` rewritten onto `mcp.CommandTransport`; `resultText`/`exploreAlpha` ported to go-sdk's `*mcp.ClientSession`/`*mcp.CallToolResult` types.
- `internal/mcp/archtest/protocol_version_selftest_test.go` — planted import/reference re-pointed from `mark3labs/mcp-go/mcp`'s `LATEST_PROTOCOL_VERSION` to go-sdk's `mcp.MetaKeyProtocolVersion`; doc comment records the finding that go-sdk exposes no exported protocol-version *value* constant at all.
- `internal/mcp/archtest/protocol_version_test.go` — `modernSDKPath` table fixture updated to go-sdk's import path; package doc comment's "What this guard CANNOT catch" section gains a note about `MetaKeyProtocolVersion`/`CodeUnsupportedProtocolVersion` as future false-positive candidates.
- `internal/cli/archtest/mcp_sdk_selftest_test.go` — planted import/reference re-pointed from `mark3labs/mcp-go/server`'s `ServeStdio` to go-sdk's `mcp.NewServer`.
- `go.mod` / `go.sum` — `mark3labs/mcp-go` removed; `go mod tidy -e` run against the real tree.
- `NOTICE` — third-party attribution bullet updated.

## Decisions Made

**t.Errorf/t.Fatalf count divergence (documented, not silently accepted):** the plan's Task 1 acceptance criteria asked for an unchanged call-site count in `testdata/golden/golden_parity_test.go`. The count dropped from 114 to 112 because go-sdk's `Client.Connect` atomically performs both connection establishment and the initialize handshake — collapsing mark3labs' two separately-fallible steps (`NewInProcessClient()` then a later `Initialize()` call, each with its own error check) into one. This mirrors the exact consolidation `internal/mcp/server_test.go`'s `newTestSession` helper already established in plan 02-01 as this repo's pattern for the same SDK. No test *assertion* (`IsError` checks, content/byte comparisons) was removed or weakened — every one is unchanged in count and strength; only setup-error-check plumbing shrank because the API surface it wrapped shrank.

**go.mod predicted-vs-actual delta (02-RESEARCH.md Q9 re-confirmed against the real tree):**

| Module | Q9 prediction | Actual (this task) | Note |
|---|---|---|---|
| `github.com/mark3labs/mcp-go` | REMOVED | REMOVED | as predicted |
| `github.com/google/jsonschema-go` | version bump v0.4.2→v0.4.3 | bump confirmed, **also flips indirect→direct** | `internal/mcp/tools_schema_drift_test.go` already imports it directly (added in plan 02-01) — a latent go.mod tidiness gap Q9 did not anticipate, not introduced by this task |
| `github.com/yosida95/uritemplate/v3` | UNCHANGED | UNCHANGED | as predicted |
| `github.com/segmentio/encoding` + `asm` | ADDED | already present (added by plan 02-01 alongside go-sdk) | unchanged by *this* task's diff |
| `golang.org/x/oauth2`, `golang.org/x/time` | ADDED | already present (added by plan 02-01) | unchanged by *this* task's diff |
| `github.com/google/uuid` | REMOVED (assuming no other reference) | **STAYS** | separate reachability path: `internal/migrate → modernc.org/sqlite → modernc.org/libc → google/uuid` — unrelated to mark3labs or sigstore-go, a path the research's `go mod why` scan at the time did not surface |
| `github.com/santhosh-tekuri/jsonschema/v6` | REMOVED | REMOVED | as predicted (`go mod why -m` confirms no reachable path) |
| `github.com/spf13/cast` | REMOVED | **go.mod LINE removed, module still present in the full graph** | `go mod why -m` confirms our own build reaches nothing in it; `go mod graph` shows it's still required transitively by `sigstore/rekor`, `sigstore/rekor-tiles/v2`, `sigstore/timestamp-authority/v2` (all part of the pre-existing `sigstore-go` dependency, unrelated to this task) |
| `github.com/stretchr/testify` | likely removed | confirmed absent from `go.mod`/`go.sum` | as predicted |
| `github.com/golang-jwt/jwt/v5` | predicted NOT to land as a go.mod line | **confirmed absent from go.mod**, but `go mod why -m` shows it IS reachable via a TEST-only file deep in go-sdk's own `oauthex` package (`internal/mcp → go-sdk/mcp → go-sdk/auth → go-sdk/oauthex → oauthex.test → golang-jwt/jwt/v5`) | the prediction holds for the visible go.mod surface; the deeper reachability nuance was not something Q9's scratch dry-run (no real importing source) could have found |
| `go.yaml.in/yaml/v3` | not discussed | **flips indirect→direct** | `internal/upgrade` already imports it directly — another latent tidiness gap, unrelated to the SDK swap, surfaced only because this is apparently the first time `go mod tidy` has run to completion in this repo |

**govulncheck result (verbatim summary):**

```
=== Symbol Results ===

No vulnerabilities found.

Your code is affected by 0 vulnerabilities.
This scan also found 1 vulnerability in packages you import and 1 vulnerability
in modules you require, but your code doesn't appear to call these
vulnerabilities.
```

Verbose run additionally shows two unreached, pre-existing findings unrelated to this task: `GO-2026-5841` (OOB read in `klauspost/compress/s2`, fixed in v1.18.7 — inherited via the `pebble`/`sigstore` chain) and `GO-2026-5932` (unmaintained `golang.org/x/crypto/openpgp`, no fix — inherited the same way). Neither is reachable from code this project calls, and neither is new to this task's dependency changes.

**SBOM verification (local, since the real path — `.github/workflows/release.yml`'s assembly job — only runs against a tagged release build in CI):**

```
go build -o codegraph-sbom-check ./cmd/codegraph
syft codegraph-sbom-check -o spdx-json=codegraph-sbom-check.spdx.json
```

The resulting SPDX document's package list contains `github.com/modelcontextprotocol/go-sdk` `v1.7.0` and zero matches for `mark3labs`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - mechanical/unavoidable] `t.Errorf`/`t.Fatalf` call-site count in `testdata/golden/golden_parity_test.go` dropped from 114 to 112**

- **Found during:** Task 1, applying the plan's acceptance criterion checking this count is unchanged.
- **Issue:** go-sdk's `Client.Connect` performs both connection establishment and the initialize handshake in one call, where mark3labs required two separately-fallible steps (`NewInProcessClient()` then `Initialize()`), each with its own error check. Consolidating both call sites into a single shared `newGoldenSession` helper (mirroring `internal/mcp/server_test.go`'s established `newTestSession` pattern) reduced the relevant `Fatalf` count from 3 source occurrences (2× `NewInProcessClient` + 1× shared `client Initialize`) to 1 (`client Connect`).
- **Fix:** Documented rather than artificially padded — inflating the call count to satisfy a literal grep would mean either duplicating setup code against the established repo pattern or inventing a fake second error check with no real failure mode behind it, both worse than an honest, explained divergence.
- **Files modified:** `testdata/golden/golden_parity_test.go`
- **Verification:** `git diff` confirms exactly 3 `Fatalf` lines removed and 1 added (net −2); every `IsError`/content-comparison assertion in the file is unchanged in both count and text.
- **Committed in:** `01f854a` (Task 1 commit)

**2. [Rule 1 - blocking, pre-existing/unrelated] `go mod tidy` fails without `-e` due to an upstream `alex-pinkus/tree-sitter-swift` quirk**

- **Found during:** Task 3, running `go mod tidy` per the plan's literal action text.
- **Issue:** `go mod tidy`'s default test-dependency resolution walks into `alex-pinkus/tree-sitter-swift`'s own `_test.go` file, which imports `github.com/tree-sitter/tree-sitter-swift/bindings/go` — a different module that does not contain that package at the resolved version. This aborts `go mod tidy` entirely. Reproduced identically against the pre-task, unmodified `go.mod` (confirmed via `git checkout -- go.mod` before making any Task 3 edit, then re-running), proving it is unrelated to the mark3labs removal and predates this task. No CI job or `Taskfile.yml` target currently runs `go mod tidy`, which is presumably why this has never surfaced before.
- **Fix:** Ran `go mod tidy -e`, which proceeds past the one unresolvable test-only import path. Two consecutive `-e` runs produced an identical `go.mod`/`go.sum` diff, confirming the result is stable/converged rather than a partial completion.
- **Files modified:** `go.mod`, `go.sum`
- **Verification:** `go build ./...` and `go vet ./...` both exit 0 against the tidied tree; the full predicted-vs-actual table above; idempotent re-run confirmed via `git diff --stat` showing zero further changes.
- **Committed in:** `2220dee` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 mechanical/unavoidable API-shape consequence, 1 blocking/pre-existing-and-unrelated)
**Impact on plan:** Neither represents scope creep. The first is a documented, honest divergence from a literal acceptance criterion that could not survive contact with go-sdk's actual API shape without either duplicating code against an established repo pattern or inventing a fake error check. The second is a pre-existing upstream quirk, confirmed reproducible against the untouched tree before this task began, worked around with the standard `-e` flag rather than papered over.

## Issues Encountered

None beyond the two deviations documented above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `github.com/mark3labs/mcp-go` is fully absent from `go.mod`, `go.sum`, and `go list -m all`'s output — SDK-03 is satisfied.
- `TestFrozenTranscriptsMatch` remains EXPECTED RED (confirmed via `task test:unit`: exactly one failing package, `test/wireoracle`, with every failing subtest under `TestFrozenTranscriptsMatch` — all 45 other packages pass) — untouched here, per this plan's explicit prohibition; plan 02-05 re-freezes it through a reviewed diff pass.
- `test/wireoracle/`, `testdata/wireoracle/`, `internal/mcp/tools.go`, and `internal/mcp/server.go` are all confirmed unmodified across this plan's three commits (`git diff --stat` against the pre-plan commit shows zero changes to any of them).
- `internal/cli/archtest`'s `forbiddenMCPSDKPrefixes` and `tools/transcriptfreeze`'s `mcpSDKModulePrefixes` both still name `mark3labs/mcp-go` as a forbidden/classified string — neither list was touched, preserving the re-introduction guard.
- One MCP SDK is now in the tree (`modelcontextprotocol/go-sdk` v1.7.0), the dependency closure has been re-audited through the existing `govulncheck` and (locally-verified) SBOM paths, and the research's predicted module delta has been confirmed or corrected against the real `go mod tidy` output.

---
*Phase: 02-sdk-migration-official-go-sdk-on-the-existing-surface*
*Completed: 2026-08-06*

## Self-Check: PASSED

- All 10 modified files confirmed present on disk with the expected content.
- Commits `01f854a`, `7d24831`, `2220dee` confirmed present in `git log`.
- `rg -c mark3labs go.mod go.sum` returns zero matches in both files.
- `go list -m all` names `github.com/modelcontextprotocol/go-sdk v1.7.0` and no `mark3labs` line.
