# Tasks

This change adds no behavior, so its implementation work is to confirm that the tests named below still pin each requirement, and to record what nothing pins.

## Pin map

| Requirement | Pinned by |
|---|---|
| Stdio transport behind an explicit flag | `TestServeCmdRequiresMCPFlag` (`internal/cli/query_cli_test.go`); every `test/wireoracle` scenario |
| Stdout carries only protocol messages | `TestServeMCPStdoutIsPureJSONRPC` (`test/integration`); `TestNoStdoutNoiseInServeReachablePackages` (`internal/graphstore/archtest`); `TestNoCharmInServeReachablePackages` (`internal/cli/present/archtest`) |
| Owed responses survive the end of stdin | `TestFrozenTranscriptsMatch` (`test/wireoracle`); `TestPendingWriter*`, `TestUnparseableLineDoesNotCostAFullDrainGrace` (`internal/mcp/pending_writer_test.go`) |
| One stderr session line per initialize | `assertSessionLine` / `assertNoSessionLine` in `TestFrozenTranscriptsMatch`; `TestSessionLineFormat`, `TestSessionLineSanitizesHostileClientInfo`, `TestSanitizeClientFieldTruncatesOnRuneBoundary`, `TestSessionLineIsAlwaysOnByConstruction` (`internal/mcp`) |
| Classic initialize negotiation | transcripts `legacy-*`, `handshake-explore`; `TestLegacyEraBaselineIsDocumented` |
| Sessionless 2026-07-28 requests and server discovery | transcripts `modern-discover-explore`, `modern-meta-invalid-params`, `modern-meta-unsupported-version`; `TestSpecAnchorsHold` |
| Result metadata on 2026-07-28 requests | transcripts `modern-discover-explore`, `modern-listen-catalog-change`, and the classic `call-*` transcripts |
| Server identity | every classic transcript (name); `modern-discover-explore` (version) |
| Fixed capabilities, independent of index state | byte-exact capabilities in `toolslist-no-index`, `handshake-explore`; capability anchors in `test/wireoracle/anchors.go` |
| Initialize comes first in a classic session | transcript `edge-call-before-initialize` |
| Server instructions | the nine tests in `internal/mcp/instructions_contract_test.go`; byte-exact in the transcripts |
| Default tool set | transcript `toolslist-default`; `TestToolsListExactSets`; `TestDefaultToolVisibility` |
| Tool input schemas and listing order | transcripts `toolslist-default`, `toolslist-repeat`; `TestToolsListOrderIsDeterministic`; `TestMCPToolSchemaNumericClaimsMatchEngineConstants`; `TestUnknownArgumentIsRejected` |
| Tools are read-only and closed-world | `TestEveryToolIsAnnotatedReadOnlyClosedWorld` |
| Narrowing the tool set | `TestToolFilterNarrowsToNamedCompanions`, `TestEmptyToolFilterNarrowsToExploreOnly`; transcripts `toolslist-narrowed`, `toolslist-filter-empty` |
| No index means no tools | transcript `toolslist-no-index`; `TestNoIndexZeroTools`; `TestServeServerPathsNoIndex` |
| The tool set follows the index during a session | transcript `index-appears-mid-session`; `TestIndexAppearingMidSessionRegistersTools`, `TestIndexAppearingMidSessionHonorsToolFilter`, `TestIndexDisappearingMidSessionUnregistersTools`, `TestRepeatedListsDoNotDuplicateTools` |
| Tool-list change notifications | transcript `modern-listen-catalog-change` |
| Catalog and resource answers are not cacheable | cache-control anchors in `test/wireoracle/anchors.go` |
| Tool calls are read-only and confined to the index root | transcript `error-confinement-reject`; `TestOpenEnginePathConfinedToRepoRoot`, `TestConfinementAnchoredOnRepoRootNotStartPath`; `TestServeKeepsStartPathDistinctFromConfinementRoot` |
| Tool results are markdown shared with the CLI | transcripts `handshake-explore`, `call-*`; `TestMarkdownOutput`, `TestStatusMarkdownOutput`; `TestExploreCLIMatchesMCP`, `TestNodeCLIMatchesMCP`, `TestNodeLineHintCLIMatchesMCP` (`testdata/golden`) |
| Worktree notice on a borrowed index | `TestWorktreeNoticeReachesServeMCPExplore` (`test/integration`); `TestWorktreeNoticeOnMismatch`, `TestWorktreeNoticeNotAppliedOnError` (`internal/mcp`) |
| Tool failures are tool results and unroutable requests are protocol errors | `internal/mcp/error_mapping_test.go`; transcripts `error-unknown-tool`, `error-malformed-args`, `error-unknown-method` |
| A briefly locked store does not fail or hang a call | store level only: `internal/graphstore/open_lock_test.go` |
| Reference resources are always listed | transcripts `resources-list`, `resources-list-no-index`; `TestResourcesRegisterWithoutIndex`, `TestResourcesListAdvertisesRegisteredURIs` |
| Reading a resource | the `resources-read-*` transcripts; `TestResourcesReadReturnsMarkdown` |
| Resource text matches the server | `internal/mcp/resources_schema_drift_test.go` |
| Finding the index | `TestServeKeepsStartPathDistinctFromConfinementRoot`, `TestServeServerPathsNoIndex`; `TestWorktreeNoticeReachesServeMCPExplore` |
| Startup sync before serving | `TestReconnectReconcile` (in-process replica of the startup sync) |
| Startup sync failures | none |
| File watcher on by default | `TestServeWatchStartDeferred`, `TestServeWatchStartDisabledPrintsVerbatimMessage` (`internal/cli`); `TestDefaultWatchHandshakePrompt`, `TestNoWatchEnvDisablesViaStderr`, `TestLiveEditAutoSyncReachesExplore` (`test/integration`); `TestWatchDisabledReason`, `TestDetectWSL` (`internal/watch`) |
| Watcher limits within a session | none |

## Unpinned behavior

Specified because it is current, observable behavior, but no test asserts it at the MCP level today. Each can get a test in a later change.

- Stdio transport: the error text for a missing `--mcp` (only the error itself is pinned).
- No index: a `tools/call` answers `-32602` for an unknown tool.
- Narrowing: `codegraph_node` and `explore` count as unknown names.
- A locked store: the bounded retry and the `isError` result at the MCP level.
- Startup sync: the server-level wiring, and both failure paths.
- Watcher limits: no watcher for an index created mid-session, and watcher errors not stopping the server.

## Not specified

Left out on purpose and listed in the proposal: the #74 and #113 defects, a malformed JSON line ending the session, `cacheScope: "public"` on two go-sdk default methods, a relative tool `path` resolving against the process directory, and nested indexes inside the index root.

## 1. Confirm the pins

- [x] 1.1 Run `task test:wireoracle` and verify it passes; it holds the transcripts the pin map cites.
- [x] 1.2 Run `go test ./internal/mcp/...` and verify it passes; it holds the session-line, instructions, tool, filter, error-mapping, resource and worktree tests.
- [x] 1.3 Run `task test:integration` and `task test:golden` and verify both pass; they hold the stdout-purity, watcher, worktree and CLI-equals-MCP tests.
- [x] 1.4 Run `go test ./internal/cli -run 'Serve'` with `HOME` set to a scratch directory (the #73 leak) and `go test ./internal/watch/ ./internal/graphstore/...`, and verify both pass.
- [x] 1.5 Check that every test name in the pin map exists, with `rg -n 'func <Name>\('` per name or `rg -n '<scenario>' test/wireoracle/scenarios.go` per transcript, and fix any row that names a missing test.

## 2. Tidy up

- [x] 2.1 Remove the two comments that say not to gate on `gsd-tools check tdd-red-evidence` from `internal/upgrade/changie_shape_test.go` and `internal/upgrade/changie_release_shape_test.go`; verify `rg -n 'gsd-tools' internal/` finds nothing and `go test ./internal/upgrade/` passes.

## 3. Validate and record

- [x] 3.1 Run `openspec validate baseline-mcp-server --strict --no-interactive` and verify it reports the change as valid.
- [x] 3.2 Record `<!-- changelog-exempt: adds a baseline spec and removes two test comments; no behavior change -->` for the PR body.

## Workflow follow-up

- Archive in the same PR with `openspec archive baseline-mcp-server -y`, then verify `openspec/specs/mcp-server/spec.md` exists and `openspec validate --all --strict --no-interactive` passes.
- The PR body carries `Refs #122` and a `pr-issue-exempt` marker, because the change touches test files outside the housekeeping paths and must not close the tracker.
- Offer the maintainer issues for the defects this research found that are not tracked yet: the malformed-JSON-line session end, the `cacheScope` inconsistency, and the doc drift in `test/wireoracle/COVERAGE-BASELINE.md` and `docs/MCP-2026-07-28-SCOPING.md`.
