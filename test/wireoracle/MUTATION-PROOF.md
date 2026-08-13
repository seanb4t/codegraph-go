# Wire Oracle Mutation Matrix — D-07 Non-Vacuity Proof

**Date:** 2026-08-05
**Phase:** 01-protocol-scoping-the-sdk-independent-wire-oracle, plan 01-07, Task 2
**Requirement:** D-07 — a gate is not trusted until it has been demonstrated RED against a
confirmed-applied mutation.

This is the one-time mutation matrix run against the real, built `codegraph` binary. For each
mutation: the edit was applied, **confirmed applied** by inspecting the changed file (`git diff`)
and rebuilding, `task test:wireoracle` was run and its verbatim failure recorded, then the edit was
reverted and the suite confirmed green again. All mutations below were executed exactly as
described — none are described without having been run.

Baseline precondition, confirmed before any mutation: `task test:wireoracle` exits 0 on the
unmodified tree. Mutations 1-4 were run against a 23-scenario tree (phase 1); mutations 5 and 6
below were run against a 28-scenario tree (phase 5 plan 01, SPEC-09) — the four scenarios added by
phases 3 and 4 in between are not re-litigated here.

---

## Mutation 1 — a stray non-JSON stdout line

**Edit:** `internal/mcp/server.go`, first line of `BuildServer` (reached on every `serve --mcp`
startup):

```diff
 func BuildServer(hasIndex bool, allowlist map[string]bool, repoPath, startPath string, opts ...Option) *server.MCPServer {
+	fmt.Println("MUTATION-1-STRAY-STDOUT-LINE")
 	cfg := &buildConfig{}
```

**Confirmed applied:** `git diff internal/mcp/server.go` showed the added line; `go build ./...`
exited 0 (rebuild picked it up — `test/wireoracle`'s `TestMain` builds the binary fresh via
`go build` on every `go test` invocation, so no separate rebuild step was needed once the source
change landed).

**Gate that went red:** BOTH `TestFrozenTranscriptsMatch` (byte comparison against every one of
the 23 frozen transcripts) and `TestSpecAnchorsHold`'s framing invariant (`assertFramingInvariant`,
which requires every stdout line to parse as JSON). The oracle detected the runtime corruption
itself, independent of any other package's guard.

**`task test:wireoracle`'s own verbatim failure** (run in isolation, recorded separately from the
archtest below — this is the requirement that the oracle itself, not a corroborating check, is what
caught it):

```
task: [test:wireoracle] go test ./test/wireoracle/...
--- FAIL: TestFrozenTranscriptsMatch (3.97s)
    --- FAIL: TestFrozenTranscriptsMatch/handshake-explore (0.67s)
        oracle_test.go:130: scenario "handshake-explore": normalized transcript differs at line 1:
             got: "MUTATION-1-STRAY-STDOUT-LINE"
            want: "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"protocolVersion\":\"2025-11-25\",\"capabilities\":{\"tools\":{\"listChanged\":true}},\"serverInfo\":{\"name\":\"codegraph\",\"version\":\"<VERSION>\"}}}"
    [... all remaining 22 scenarios under TestFrozenTranscriptsMatch fail identically, "got" always
    the stray line, "want" always that scenario's frozen first line ...]
--- FAIL: TestSpecAnchorsHold (4.30s)
    --- FAIL: TestSpecAnchorsHold/handshake-explore (0.27s)
        oracle_test.go:574: scenario "handshake-explore": stdout line is not valid JSON: invalid character 'M' looking for beginning of value: "MUTATION-1-STRAY-STDOUT-LINE"
    [... all remaining 22 scenarios under TestSpecAnchorsHold's framing invariant fail identically ...]
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/wireoracle	17.861s
FAIL
task: Failed to run task "test:wireoracle": exit status 1
```

All 23 of 23 scenarios under `TestFrozenTranscriptsMatch` and all 23 under `TestSpecAnchorsHold`'s
framing invariant failed — the mutation is caught in every session, not just some, because
`fmt.Println` fires on every `BuildServer` call regardless of scenario.

**Corroborating evidence (recorded separately, run independently via
`go test ./internal/graphstore/archtest/... -count=1 -v`):** the pre-existing static
stdout-confinement archtest also fired, confirming the mutation landed in a serve-reachable
package:

```
=== RUN   TestNoStdoutNoiseInServeReachablePackages
    stdout_confinement_test.go:224: github.com/seanb4t/codegraph-go/internal/mcp: calls a bare fmt.Print*/Printf/Println (no explicit writer) — stdout is reserved for the MCP JSON-RPC transport (HYG-02)
--- FAIL: TestNoStdoutNoiseInServeReachablePackages (0.66s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/graphstore/archtest	1.718s
FAIL
```

This is corroborating evidence only — the load-bearing result is `task test:wireoracle`'s own
failure above, which proves the *oracle itself* (a runtime, wire-level check) detected the
corruption, not merely a static source-scan of `internal/mcp`'s package body.

**Revert confirmation:** the `fmt.Println` line was removed; `git diff internal/mcp/server.go`
showed zero lines changed; `go build ./...` exited 0; `go test ./internal/graphstore/archtest/...
-count=1` and `go test ./test/wireoracle/... -count=1` both exited 0.

---

## Mutation 2 — a dropped tool

**Edit:** `internal/mcp/server.go`, inside `BuildServer`'s companion-tool registration loop:

```diff
 	for _, name := range companionNames {
+		if name == "status" {
+			continue
+		}
 		if allowlist[name] {
 			s.AddTool(companionTool(name), companionHandler(name, repoPath, startPath, detector))
 			toolCount++
 		}
 	}
```

**Confirmed applied:** `git diff internal/mcp/server.go` showed the added `continue` branch;
`go build ./...` exited 0.

**Gate that went red:** the allowlisted `tools/list` transcripts (`toolslist-allowlist`,
`toolslist-repeat`) — both of which enumerate `codegraph_status` when it is registered — and
`call-status`'s own `tools/call` transcript (the dropped tool's call now returns a top-level
`error`, not the successful result the frozen transcript recorded). The exact-set list assertion
(`TestToolsListExactSets`) and the structural coverage assertion
(`TestEveryRegisteredToolHasASuccessfulCallScenario`) both independently name the missing tool.

**`task test:wireoracle`'s verbatim failure:**

```
task: [test:wireoracle] go test ./test/wireoracle/...
--- FAIL: TestFrozenTranscriptsMatch (3.91s)
    --- FAIL: TestFrozenTranscriptsMatch/toolslist-allowlist (0.14s)
        oracle_test.go:130: scenario "toolslist-allowlist": normalized transcript differs at line 2:
             got: [tools array with 2 entries: codegraph_explore, codegraph_node]
            want: [tools array with 3 entries: codegraph_explore, codegraph_node, codegraph_status]
    --- FAIL: TestFrozenTranscriptsMatch/toolslist-repeat (0.14s)
        oracle_test.go:130: scenario "toolslist-repeat": normalized transcript differs at line 2:
             got: [tools array with 7 entries, codegraph_status absent]
            want: [tools array with 8 entries, codegraph_status present]
    --- FAIL: TestFrozenTranscriptsMatch/call-node (0.18s)
        oracle_test.go:148: scenario "call-node": session line tools="7", want "8"
    --- FAIL: TestFrozenTranscriptsMatch/call-search (0.18s)
        oracle_test.go:148: scenario "call-search": session line tools="7", want "8"
    --- FAIL: TestFrozenTranscriptsMatch/call-callers (0.18s)
        oracle_test.go:148: scenario "call-callers": session line tools="7", want "8"
    --- FAIL: TestFrozenTranscriptsMatch/call-callees (0.19s)
        oracle_test.go:148: scenario "call-callees": session line tools="7", want "8"
    --- FAIL: TestFrozenTranscriptsMatch/call-impact (0.19s)
        oracle_test.go:148: scenario "call-impact": session line tools="7", want "8"
    --- FAIL: TestFrozenTranscriptsMatch/call-files (0.16s)
        oracle_test.go:148: scenario "call-files": session line tools="7", want "8"
    --- FAIL: TestFrozenTranscriptsMatch/call-status (0.13s)
        oracle_test.go:130: scenario "call-status": normalized transcript differs at line 2:
             got: "{\"jsonrpc\":\"2.0\",\"id\":2,\"error\":{\"code\":-32602,\"message\":\"tool 'codegraph_status' not found: tool not found\"}}"
            want: "{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"content\":[{\"type\":\"text\",\"text\":\"**CodeGraph Status**\\n\\n**Files indexed:** 3\\n**Total nodes:** 9\\n**Total edges:** 11\\n**Database size:** 0.01 MB\\n**Backend:** pebble\\n\\n**Nodes by Kind:**\\n- function: 4\\n- file: 3\\n- package: 2\\n\\n**Languages:**\\n- go: 3\\n\\nIndex is up to date.\\n\"}]}}"
--- FAIL: TestToolsListExactSets (0.30s)
    --- FAIL: TestToolsListExactSets/toolslist-allowlist (0.14s)
        oracle_test.go:703: scenario "toolslist-allowlist": tools/list advertised [codegraph_explore codegraph_node], want exactly [codegraph_explore codegraph_node codegraph_status]
--- FAIL: TestEveryRegisteredToolHasASuccessfulCallScenario (0.13s)
    oracle_test.go:772: toolslist-repeat: registered 7 tools, want 8: [codegraph_callees codegraph_callers codegraph_explore codegraph_files codegraph_impact codegraph_node codegraph_search]
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/wireoracle	16.349s
FAIL
task: Failed to run task "test:wireoracle": exit status 1
```

Every call-scenario carrying `Env: envAllowlistAllCompanions` reports `tools="7"` (the session
line's tool count) instead of the frozen `"8"`, since `codegraph_status` never registers — this is
the exact "the allowlisted transcript and that tool's call transcript go red" result the plan
predicted, plus the exact-set assertion naming the missing tool explicitly.

**Revert confirmation:** the `continue` branch was removed; `git diff internal/mcp/server.go`
showed zero lines changed; `go build ./...` exited 0; `go test ./test/wireoracle/... -count=1`
exited 0 (a retry was needed once — the first retry hit the pre-existing session-line flake noted
in the plan SUMMARY's Issues Encountered, unrelated to this mutation since the source diff was
already confirmed empty at that point).

---

## Mutation 3 — a changed JSON-RPC error shape

**Handler and request named exactly, per the plan's requirement:** `exploreHandler`
(`internal/mcp/tools.go:103-123`), the missing-`query` branch at `:105-107`. The request that
reliably reaches this branch is a `tools/call` for `codegraph_explore` with an `arguments` object
that omits `query`. This request is **not** one of the 23 frozen scenarios — driving it would
disturb `ExpectedScenarioCount` and the transcript-set equality guard — so it was driven directly
through `Capture` from a temporary, never-committed test file
(`test/wireoracle/mutation3_probe_test.go`, deleted immediately after this mutation's evidence was
captured, never staged or committed).

**Edit:**

```diff
 	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
 		q, err := req.RequireString("query")
 		if err != nil {
-			return mcp.NewToolResultError(err.Error()), nil
+			return nil, err
 		}
```

**Confirmed applied:** `git diff internal/mcp/tools.go` showed the change; `go build ./...` exited
0.

**Pre-mutation baseline** (the probe run before the edit, proving the gate's normal-path shape):

```
=== RUN   TestMutationProbe3Temporary
    mutation3_probe_test.go:55: codegraph_explore/omitted-query response (id=2): {"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"required argument \"query\" not found"}],"isError":true}}
--- PASS: TestMutationProbe3Temporary (0.65s)
```

**Gate that went red (the probe itself, run with `go test ./test/wireoracle/... -count=1 -run
TestMutationProbe3Temporary -v`):**

```
=== RUN   TestMutationProbe3Temporary
    mutation3_probe_test.go:53: codegraph_explore tools/call with omitted query returned a top-level JSON-RPC error instead of a tool-visible isError result: {"jsonrpc":"2.0","id":2,"error":{"code":-32603,"message":"required argument \"query\" not found"}}
--- FAIL: TestMutationProbe3Temporary (0.67s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/wireoracle	4.431s
FAIL
```

The captured error shape differs from the tool-result shape exactly as the plan predicted: what was
a successful `result.isError=true` response became a top-level `error` object with
`code:-32603` (`mcp.INTERNAL_ERROR` — mark3labs v0.56.0's `handleToolCall`, when a
`ToolHandlerFunc` returns a non-nil `error`, wraps it as `&requestError{code: mcp.INTERNAL_ERROR,
...}`, verified by reading
`github.com/mark3labs/mcp-go@v0.56.0/server/server.go:2007-2013`).

**Which transcripts changed, and which did not — the measured blast radius, recorded as input to
Phase 2's SDK-04 audit:** running `go test ./test/wireoracle/... -count=1` (the full 23-scenario
suite, mutation still applied) reported:

```
--- FAIL: TestMutationProbe3Temporary (0.64s)
    mutation3_probe_test.go:53: codegraph_explore tools/call with omitted query returned a top-level JSON-RPC error instead of a tool-visible isError result: {"jsonrpc":"2.0","id":2,"error":{"code":-32603,"message":"required argument \"query\" not found"}}
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/wireoracle	17.498s
FAIL
```

**Zero of the 23 frozen transcripts changed.** `TestFrozenTranscriptsMatch`, `TestSpecAnchorsHold`,
`TestEveryRegisteredToolHasASuccessfulCallScenario`, and every other permanent guard passed
unaffected — only the temporary, un-frozen probe (constructed specifically to exercise this one
path) detected the mutation. This is the honest, load-bearing finding: none of today's frozen
scenarios call `codegraph_explore` with an omitted `query` argument (`handshake-explore` and
`edge-call-before-initialize` both supply a real query), so this specific error-shape mutation is
**invisible to the frozen pre-migration baseline**. Phase 2's SDK-04 audit (auditing every
handler's error-return shape across the SDK swap) should treat "does the frozen suite exercise
every handler's own argument-validation failure path" as an open question this measurement answers
"not yet, for at least this one handler" — not something this plan's scope extends the frozen set
to fix (extending the frozen set is itself out of scope per `ExpectedScenarioCount`'s "cannot be
extended after Phase 2" constraint recorded in `COVERAGE-BASELINE.md`).

**Revert confirmation:** the handler body was restored to `return mcp.NewToolResultError(err.Error()), nil`;
`git diff internal/mcp/tools.go` showed zero lines changed; `test/wireoracle/mutation3_probe_test.go`
was deleted (`git status --short` showed no trace of it, confirming it was never staged); `go build
./...` exited 0; `go test ./test/wireoracle/... -count=1` exited 0.

---

## Mutation 4 — a changed protocol-version literal

**Edit:** `internal/mcp/protocol_version.go`:

```diff
-const ProtocolVersion = "2025-11-25"
+const ProtocolVersion = "9999-99-99"
```

**Confirmed applied:** `git diff internal/mcp/protocol_version.go` showed the changed literal;
`go build ./...` exited 0.

**Gate that went red — the D-02 spec anchor, run in isolation
(`go test ./test/wireoracle/... -count=1 -run '^TestSpecAnchorsHold$' -v`), independent of the
frozen transcripts (this test re-captures rather than reading `testdata/`):**

```
=== RUN   TestSpecAnchorsHold/handshake-explore/protocolVersion
    oracle_test.go:578: initialize response protocolVersion = "2025-11-25", want "9999-99-99": "{\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{\"protocolVersion\":\"2025-11-25\",\"capabilities\":{\"tools\":{\"listChanged\":true}},\"serverInfo\":{\"name\":\"codegraph\",\"version\":\"0.1.0\"}}}"
--- FAIL: TestSpecAnchorsHold (4.38s)
    --- FAIL: TestSpecAnchorsHold/handshake-explore (0.74s)
        --- FAIL: TestSpecAnchorsHold/handshake-explore/protocolVersion (0.00s)
    --- PASS: TestSpecAnchorsHold/toolslist-default (0.13s)
    [... every other scenario under TestSpecAnchorsHold PASSes — the protocolVersion anchor is
    only asserted for handshake-explore within this test ...]
```

**The precise asymmetry D-02 requires, demonstrated:** the captured wire value is
`"2025-11-25"` — **unchanged**, because mark3labs v0.56.0 negotiates its own
`LATEST_PROTOCOL_VERSION` internally and never reads this repository's `ProtocolVersion` constant
(Phase 1's documented mechanism-honesty limit, see `protocol_version.go`'s own doc comment). Only
the *comparison value* — the repo-owned literal this anchor is pinned against — moved. The anchor
fails; the wire behavior does not change.

**Also observed, and recorded precisely rather than glossed over:** running
`go test ./test/wireoracle/... -count=1 -run '^TestFrozenTranscriptsMatch$' -v` shows this same
mutation ALSO fails 16 of the 23 `TestFrozenTranscriptsMatch` subtests — but for a reason that
confirms rather than contradicts the asymmetry above. `TestFrozenTranscriptsMatch` calls
`assertBytesEqualLineByLine` (the byte-for-byte transcript comparison) FIRST, then — for every
non-era, non-`NoInitialize` scenario — calls `assertSessionLine`/`assertProtocolVersionAnchor`
against `internalmcp.ProtocolVersion` (the mutated literal) SECOND, in the same subtest. The
verbatim failures are:

```
=== RUN   TestFrozenTranscriptsMatch/handshake-explore
=== RUN   TestFrozenTranscriptsMatch/toolslist-default
    oracle_test.go:148: scenario "toolslist-default": session line requested="2025-11-25", want "9999-99-99"
=== RUN   TestFrozenTranscriptsMatch/toolslist-allowlist
    oracle_test.go:148: scenario "toolslist-allowlist": session line requested="2025-11-25", want "9999-99-99"
[... 13 more non-era scenarios, identical shape: requested="2025-11-25", want "9999-99-99" ...]
--- FAIL: TestFrozenTranscriptsMatch (3.98s)
    --- FAIL: TestFrozenTranscriptsMatch/handshake-explore (0.69s)
    --- FAIL: TestFrozenTranscriptsMatch/toolslist-default (0.14s)
    [... 14 more FAIL lines for the remaining non-era scenarios ...]
    --- PASS: TestFrozenTranscriptsMatch/edge-call-before-initialize (0.16s)
    --- PASS: TestFrozenTranscriptsMatch/legacy-2025-11-25 (0.13s)
    [... all 6 era scenarios PASS — they compare against their own EraOfferedVersion/
    EraNegotiatedVersion, never internalmcp.ProtocolVersion, so this mutation cannot touch them ...]
```

Every one of these 16 failure messages is a `requested=.../want=...` **session-line** mismatch —
**not one is a byte-diff message** (the `"normalized transcript differs at line N"` shape Mutations
1 and 2 produced above). Since `assertBytesEqualLineByLine` runs before the session-line check in
the same subtest body and never fires here, its silence is itself the proof the frozen transcript
bytes are unchanged: had the wire bytes actually differed, the byte-diff message would have fired
first and the session-line check would never have run. This is the precise, code-level form of "the
frozen transcripts stay green while the anchor goes red" — the anchor and the byte match are two
separately-verified things (D-02's whole point), coincidentally co-located in the same test function
for 16 of 23 scenarios; the isolated `TestSpecAnchorsHold` run above is the cleaner, single-purpose
demonstration of the same fact.

**Revert confirmation:** the literal was restored to `"2025-11-25"`; `git diff
internal/mcp/protocol_version.go` showed zero lines changed; `go build ./...` exited 0; `go test
./test/wireoracle/... -count=1` exited 0; `go test ./internal/graphstore/archtest/... -count=1`
exited 0 (unaffected by this mutation, run as a sanity check since Phase 3's SPEC-06 will later
depend on this same literal).

---

## Mutation 5 — the wire gate, capability off (05-01-PLAN Task 3, SPEC-09)

**Requirement:** the standing rule that a gate is not trusted until demonstrated RED against a
confirmed-applied mutation, applied to phase 5 plan 01's two new gates: the
`modern-listen-catalog-change` wire proof and the D-02 acknowledgment-echo anchor.

**Edit:** `internal/mcp/server.go`, `BuildServer`'s capability construction:

```diff
 	s := mcp.NewServer(&mcp.Implementation{Name: "codegraph", Version: version}, &mcp.ServerOptions{
 		Capabilities: &mcp.ServerCapabilities{
-			Tools: &mcp.ToolCapabilities{ListChanged: true},
+			Tools: &mcp.ToolCapabilities{ListChanged: false},
 		},
 		Instructions: instructions,
 	})
```

**Confirmed applied:** `git diff -- internal/mcp/server.go` showed exactly this one-line change;
`go build ./...` exited 0.

**Gate that went red:** `TestFrozenTranscriptsMatch/modern-listen-catalog-change` failed at
`Capture` itself — the capture never completes — and `TestSpecAnchorsHold/modern-listen-catalog-change`
and both `TestToolsListExactSets/modern-listen-catalog-change-*` cases failed the same way.

**`task test:wireoracle`'s verbatim failure** (excerpted — the capability flip is global, so every
scenario whose `initialize`/`server/discover` response carries `capabilities.tools.listChanged`
shows the identical collateral diff; one representative instance is kept, the rest are named):

```
--- FAIL: TestFrozenTranscriptsMatch (34.60s)
    --- FAIL: TestFrozenTranscriptsMatch/handshake-explore (0.87s)
        oracle_test.go:130: scenario "handshake-explore": normalized transcript differs at line 1:
             got: "...{\"capabilities\":{\"tools\":{}},...\"protocolVersion\":\"2025-11-25\",...}"
            want: "...{\"capabilities\":{\"tools\":{\"listChanged\":true}},...\"protocolVersion\":\"2025-11-25\",...}"
    [... 20 more scenarios FAIL identically on this same collateral diff: toolslist-default,
    toolslist-allowlist, toolslist-no-index, toolslist-repeat, call-node, call-search,
    call-callers, call-callees, call-impact, call-files, call-status, error-unknown-method,
    error-unknown-tool, error-malformed-args, error-confinement-reject, legacy-2025-11-25,
    legacy-2025-06-18, legacy-2025-03-26, legacy-2024-11-05, legacy-unsupported-2026-07-28,
    legacy-omitted-version, modern-discover-explore, index-appears-mid-session ...]
    --- FAIL: TestFrozenTranscriptsMatch/modern-listen-catalog-change (30.00s)
        oracle_test.go:120: capture scenario "modern-listen-catalog-change": wireoracle: scenario
        "modern-listen-catalog-change": capture deadline exceeded waiting for method
        "notifications/tools/list_changed"; 3/2 responses observed; stderr:
--- FAIL: TestEveryDeclaredFiringRuleActuallyFires (33.86s)
    oracle_test.go:471: capture scenario "modern-listen-catalog-change": wireoracle: scenario
    "modern-listen-catalog-change": capture deadline exceeded waiting for method
    "notifications/tools/list_changed"; 3/2 responses observed; stderr:
--- FAIL: TestSpecAnchorsHold (33.85s)
    --- FAIL: TestSpecAnchorsHold/handshake-explore (0.17s)
        --- FAIL: TestSpecAnchorsHold/handshake-explore/capabilities.tools.listChanged_==_true_on_the_Legacy_initialize_path_(SPEC-09_criterion_1) (0.00s)
            oracle_test.go:601: scenario "handshake-explore": result.capabilities.tools.listChanged
            = false or absent, want true: "..."
    --- FAIL: TestSpecAnchorsHold/modern-discover-explore (0.18s)
        --- FAIL: TestSpecAnchorsHold/modern-discover-explore/capabilities.tools.listChanged_==_true_on_the_Modern_server/discover_path_(SPEC-09_criterion_1) (0.00s)
            oracle_test.go:601: scenario "modern-discover-explore": result.capabilities.tools.listChanged
            = false or absent, want true: "..."
    --- FAIL: TestSpecAnchorsHold/modern-listen-catalog-change (30.01s)
        oracle_test.go:589: capture scenario "modern-listen-catalog-change": wireoracle: scenario
        "modern-listen-catalog-change": capture deadline exceeded waiting for method
        "notifications/tools/list_changed"; 3/2 responses observed; stderr:
--- FAIL: TestToolsListExactSets (60.29s)
    --- FAIL: TestToolsListExactSets/modern-listen-catalog-change-pre-init (30.01s)
        oracle_test.go:727: capture scenario "modern-listen-catalog-change": wireoracle: scenario
        "modern-listen-catalog-change": capture deadline exceeded waiting for method
        "notifications/tools/list_changed"; 3/2 responses observed; stderr:
    --- FAIL: TestToolsListExactSets/modern-listen-catalog-change-post-init (30.00s)
        oracle_test.go:727: capture scenario "modern-listen-catalog-change": wireoracle: scenario
        "modern-listen-catalog-change": capture deadline exceeded waiting for method
        "notifications/tools/list_changed"; 3/2 responses observed; stderr:
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/wireoracle	168.881s
task: Failed to run task "test:wireoracle": exit status 1
```

**The precise mechanism, traced rather than assumed:** with the capability off,
`allowedSubscriptions` grants nothing (`allowed.ToolsListChanged` stays `false`), so
`shouldSendListChangedNotification` never fires and `changeAndNotify` never schedules a
notification — the opted-in stream receives nothing at all, matching the plan's prediction exactly.

**Collateral, matching the plan's prediction with one honest correction:** the plan predicted "the
listen handler no longer blocks and does answer its own request id, so the exactly-zero check over
`NoResponseRequests` ... also go[es] red." The FIRST half is confirmed true — go-sdk's own
`subscriptionsListen` only awaits `<-ctx.Done()` when at least one subscription was actually
granted; with nothing granted it returns immediately, so request id 1 DOES receive an (unawaited)
response this run ("3/2 responses observed" — id 1, 2, and 3 all arrived, one more than the 2 the
scenario's `NoResponseRequests` exemption expects). The SECOND half does not happen the way the
plan predicted: `assertFramingInvariant`'s exactly-zero check never gets a chance to run, because
`Capture` itself never returns — it is still blocked inside the write loop on `AwaitAfterRequest`'s
wait for `notifications/tools/list_changed` on index 3, which this mutation ensures never arrives,
so the 30-second deadline fires first and `mustCaptureScenario` fails the whole subtest before
`assertFramingInvariant` is ever reached. Recorded here as the honest, measured behavior rather
than silently restating the plan's prediction — this is exactly the kind of "measure, don't assume"
finding this milestone is built on.

**Revert confirmation:** the `ListChanged: false` was restored to `ListChanged: true`; `git diff
--exit-code -- internal/mcp/server.go` exited 0; `go build ./...` exited 0; `go test
./test/wireoracle/... -count=1` exited 0 (19-22s, all 28 scenarios).

---

## Mutation 6 — the acknowledgment-echo anchor, capability off AND the notification wait removed (05-01-PLAN Task 3, D-02)

**Requirement:** Mutation 5 alone cannot prove `assertSubscriptionAckEcho` is non-vacuous, because
`Capture` never completes under that mutation — the anchor never gets a chance to run. This
mutation removes the scenario's `AwaitAfterRequest` entry for the LAST request so the capture
completes without waiting for a notification that will never come, isolating the anchor's own
behavior.

**Edit 1** (repeats mutation 5's capability flip): `internal/mcp/server.go`:

```diff
 		Capabilities: &mcp.ServerCapabilities{
-			Tools: &mcp.ToolCapabilities{ListChanged: true},
+			Tools: &mcp.ToolCapabilities{ListChanged: false},
 		},
```

**Edit 2:** `test/wireoracle/scenarios.go`, the `modern-listen-catalog-change` scenario's
`AwaitAfterRequest` map:

```diff
 			AwaitAfterRequest: map[int]string{
 				1: notificationSubscriptionsAcknowledgedMethod,
-				3: notificationToolsListChangedMethod,
 			},
```

**Confirmed applied:** `git diff -- internal/mcp/server.go test/wireoracle/scenarios.go` showed
exactly these two edits; `go build ./...` exited 0.

**Gate that went red:** `TestFrozenTranscriptsMatch/modern-listen-catalog-change`'s byte-comparison
against the frozen transcript — its very first line is now the D-02 dead-subscription shape.

**`task test:wireoracle`'s verbatim failure (the load-bearing excerpt):**

```
--- FAIL: TestFrozenTranscriptsMatch/modern-listen-catalog-change (0.07s)
    oracle_test.go:130: scenario "modern-listen-catalog-change": normalized transcript differs at line 1:
         got: "{\"jsonrpc\":\"2.0\",\"method\":\"notifications/subscriptions/acknowledged\",\"params\":{\"_meta\":{\"io.modelcontextprotocol/subscriptionId\":1},\"notifications\":{}}}"
        want: "{\"jsonrpc\":\"2.0\",\"method\":\"notifications/subscriptions/acknowledged\",\"params\":{\"_meta\":{\"io.modelcontextprotocol/subscriptionId\":1},\"notifications\":{\"toolsListChanged\":true}}}"
```

That `"notifications":{}` on the `got` side is exactly 05-CONTEXT.md D-02's dead-subscription
shape, reproduced deliberately.

**A second, honestly-recorded, non-obvious collateral effect:** in the SAME run,
`TestSpecAnchorsHold/modern-listen-catalog-change` and
`TestToolsListExactSets/modern-listen-catalog-change-post-init` both failed for a DIFFERENT
reason — the id-3 `tools/list` response itself went missing:

```
--- FAIL: TestSpecAnchorsHold/modern-listen-catalog-change (0.07s)
    oracle_test.go:591: scenario "modern-listen-catalog-change": request id 3 has 0 response lines, want exactly 1
--- FAIL: TestToolsListExactSets/modern-listen-catalog-change-post-init (0.07s)
    oracle_test.go:727: scenario "modern-listen-catalog-change": no tools/list response (id=3) found in captured stdout
```

**Traced, not assumed:** go-sdk dispatches `tools/list` asynchronously too
(`jsonrpc2.Async` fires for every call except `initialize` — the same fact `AwaitAfterRequest`'s
own doc comment already cites for the acknowledgment). With the wait on index 3 removed, `Capture`
writes the id-3 request and immediately closes stdin with nothing left to wait for; the async id-3
handler now races process shutdown exactly the way `AwaitAfterRequest`'s doc comment warns a
removed wait would race the notification — except here it is racing the RESPONSE, not just the
notification, and in this run it lost the race. This means the exactly-one `assertFramingInvariant`
check reached RED first in `TestSpecAnchorsHold`, before that subtest's per-scenario anchor loop
ever got to invoke `assertSubscriptionAckEcho`'s own `t.Run` — the same "the earlier structural
check masks the later semantic one" shape mutation 5 hit for a different reason.

**Because of that masking, the anchor was ALSO proven directly, isolated from the id-3 race**, via a
temporary, never-committed probe file (`test/wireoracle/mutation6_probe_test.go`, deleted
immediately after this evidence was captured, never staged or committed — mirroring mutation 3's
precedent above) that calls `assertSubscriptionAckEcho` directly against a fresh capture, bypassing
`assertFramingInvariant`:

```
=== RUN   TestMutationProbe6Temporary
    mutation6_probe_test.go:23: raw captured stdout:
        {"jsonrpc":"2.0","method":"notifications/subscriptions/acknowledged","params":{"_meta":{"io.modelcontextprotocol/subscriptionId":1},"notifications":{}}}
        {"jsonrpc":"2.0","id":1,"result":{"resultType":"complete","_meta":{"io.modelcontextprotocol/serverInfo":{"name":"codegraph","version":"0.1.0"},"io.modelcontextprotocol/subscriptionId":1}}}
        {"jsonrpc":"2.0","id":2,"result":{"resultType":"complete","_meta":{"io.modelcontextprotocol/serverInfo":{"name":"codegraph","version":"0.1.0"}},"ttlMs":0,"cacheScope":"private","tools":[]}}
    mutation6_probe_test.go:24: scenario "modern-listen-catalog-change": acknowledgment params.notifications = map[], want exactly map[toolsListChanged:true]: "{\"jsonrpc\":\"2.0\",\"method\":\"notifications/subscriptions/acknowledged\",\"params\":{\"_meta\":{\"io.modelcontextprotocol/subscriptionId\":1},\"notifications\":{}}}"
--- FAIL: TestMutationProbe6Temporary (0.59s)
FAIL
```

This is `assertSubscriptionAckEcho` failing with the observed notifications object empty against
the wanted single-entry set — exactly the plan's predicted RED, demonstrated directly rather than
inferred from the byte-diff alone. It also shows id 1's `SubscriptionsListenResult` now arriving
immediately (nothing granted, so the handler no longer blocks — the same mechanism mutation 5
observed) and confirms id 3 never arrived in this particular capture (only 3 lines total).

**Revert confirmation:** both edits were reverted; `git diff --exit-code -- internal/
test/wireoracle/scenarios.go` exited 0; the probe file was deleted (`git status --short --
test/wireoracle/mutation6_probe_test.go` showed nothing, confirming it was never staged); `go build
./...` exited 0; `go test ./test/wireoracle/... -count=1` exited 0 (22.6s, all 28 scenarios).

---

## Criterion 3 evidence (05-CONTEXT.md D-03/D-04)

SPEC-09 criterion 3 — "a client that does not opt in observes no change in session behavior" — is
satisfied by the 27 pre-existing frozen transcripts staying byte-unchanged across this phase, not by
a new no-op scenario (05-CONTEXT.md D-03: the existing corpus already is that assertion). Evidence,
gathered on the reverted, committed tree at the end of this plan:

- `git diff --name-status -- testdata/wireoracle/transcripts/` (against this phase's base) shows
  exactly one `A` line (`modern-listen-catalog-change.golden`) and zero `M` lines — none of the 27
  pre-existing transcripts moved.
- `go test ./test/wireoracle/... -count=1` passes all 28 scenarios, including
  `TestFrozenTranscriptsMatch`'s byte-for-byte comparison against all 27 pre-existing goldens.
- The advisory transcript-freeze CI job (`.github/workflows/ci.yml`'s `transcript-freeze` job,
  `task check:transcript-freeze` — Phase 4's advisory-as-of guard) reports the one addition and
  exits 0; it was not silenced, exempted, or edited by this plan (05-CONTEXT.md D-04).

This — not a new scenario — is what satisfies criterion 3.

---

## Full-suite wall-clock measurement (clean, reverted tree, 23 scenarios)

Measured via `time task test:wireoracle` after all four mutations above were reverted and
`git status --porcelain` confirmed empty:

```
task: [test:wireoracle] go test ./test/wireoracle/...
ok  	github.com/seanb4t/codegraph-go/test/wireoracle	16.888s
task test:wireoracle  6.75s user 9.25s system 90% cpu 17.683 total
```

**~16.9s** `go test`'s own reported time; **~17.7s** wall-clock including `task` CLI overhead.

Measured series across the phase (chars-of-scope proxy: scenario count):

| Plan | Scenario count | `go test` reported | Wall-clock (incl. `task` overhead) |
|---|---|---|---|
| 01 (tracer) | 1 | — | ~5.8s |
| 04 (full D-05 bar) | 17 | ~11.8s | ~12.6s |
| 07 (this plan, six-era + structural guards) | 23 | ~16.9s | ~17.7s |

Marginal cost from plan 04 to plan 07: roughly (16.9 − 11.8) / 6 ≈ **0.85s per additional era
scenario** — higher than plan 04's own measured ~0.38s/scenario marginal cost, consistent with
`TestEveryDeclaredFiringRuleActuallyFires` (this plan's new addition) re-capturing all 23 scenarios
a second time inside the same test binary run, on top of the pre-existing per-scenario re-captures
`TestSpecAnchorsHold`, `TestToolsListExactSets`, `TestToolsListOrderIsDeterministic`, and
`TestEveryRegisteredToolHasASuccessfulCallScenario` already perform.

**~17.7s is well within the 3-minute budget** CONTEXT's "Claude's Discretion" note left open (all
scenarios on every PR versus a narrower gating set) — no explicit flag is needed; running the full
suite on every PR remains cheap.

---

## Open gap recorded (not a failure — an honest finding)

**Mutation 3's blast radius on the frozen 23-scenario suite is zero.** No frozen scenario exercises
`codegraph_explore`'s (or any handler's) own required-argument-validation failure path — every
frozen `tools/call` in `Scenarios()` either supplies valid required arguments or targets a path that
fails for an unrelated reason (unknown tool, unknown method, confinement rejection). A change to
*any* handler's argument-validation error-return shape (the `mcp.NewToolResultError` vs. raw-`error`
distinction Mutation 3 exercises) is therefore currently undetectable by the frozen baseline alone —
only a purpose-built probe like the temporary one used here would catch it. This is exactly the
SDK-04 risk category Phase 2 must audit, and this measurement — a live rehearsal against the real,
unmodified pre-migration binary — is offered as input to that audit, not as something this plan's
scope resolves by extending the frozen set (`COVERAGE-BASELINE.md` records that the set cannot be
extended after Phase 2 removes `mark3labs/mcp-go`).

---

## A note on numbering (05-03-PLAN Task 3)

05-03-PLAN's own Task 3 text was written against an assumption that "five mutations exist" and
that the three mutations below would be numbered 6, 7, and 8. By the time this task actually ran,
05-01-PLAN's own Task 3 had already appended Mutations 5 and 6 above (the SPEC-09 capability-off
and acknowledgment-echo proofs) — six mutations existed, not five, and slot 6 was already taken.
Rather than renumber or overwrite an already-committed, already-reverted mutation record, the three
mutations below continue the sequence as Mutations 7, 8, and 9. This is recorded here plainly
rather than silently reconciled: the plan's literal acceptance-criteria commands (`rg -c '^##
Mutation' … returns 8`, and `rg -n '^## Mutation 6'` naming the tool-rename proof) were written
against the stale assumption and do not hold against the actual, current file; the corrected
verification commands are `rg -c '^## Mutation' test/wireoracle/MUTATION-PROOF.md` returning **9**,
with the tool-rename/GUARD-02 proof at `## Mutation 7`, the code-side/GUARD-01 proof at `##
Mutation 8`, and the prose-side/GUARD-01 proof at `## Mutation 9`. All three mutations named in
05-03-PLAN's success criteria (GUARD-02 criterion 3, GUARD-01 criterion 4 with its asymmetry) are
demonstrated below regardless of the number attached to each.

---

## Mutation 7 — a renamed tool, `status` to `health` (05-03-PLAN Task 3, GUARD-02)

**Requirement:** GUARD-02 — a tool added, removed, or renamed without its resource file and URI
moving with it must turn `TestResourceFileSetMatchesToolNames` red, in whichever direction the
mismatch runs, naming both a missing and an orphaned stem from one mutation.

**Edit:** three sites renamed the `status` companion to `health`, deliberately leaving
`internal/mcp/resources/status.md` and `resourceURIFor` untouched — that omission IS the drift
being demonstrated.

`internal/mcp/server.go`, the `companionNames` slice:

```diff
-var companionNames = []string{"node", "search", "callers", "callees", "impact", "files", "status"}
+var companionNames = []string{"node", "search", "callers", "callees", "impact", "files", "health"}
```

`internal/mcp/tools.go`, `companionTool`'s matching case:

```diff
-	case "status":
+	case "health":
 		return &mcp.Tool{
-			Name:        "codegraph_status",
+			Name:        "codegraph_health",
 			Description: "Report index health and counts",
 			Annotations: toolAnnotations(),
 		}
```

`internal/mcp/tools.go`, `companionHandler`'s matching case:

```diff
-	case "status":
+	case "health":
 		mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args StatusArgs) (*mcp.CallToolResult, any, error) {
```

**Confirmed applied:** `git diff -- internal/mcp/server.go internal/mcp/tools.go` showed exactly
these three hunks; `go build ./...` exited 0.

**Gate that went red — the named gate, run in isolation
(`go test ./internal/mcp/... -run TestResourceFileSetMatchesToolNames -count=1 -v`):**

```
=== RUN   TestResourceFileSetMatchesToolNames
=== RUN   TestResourceFileSetMatchesToolNames/file_set_matches_allToolNames_plus_behavior_docs
    resources_schema_drift_test.go:146: resource file set drifted from the tool roster: missing [health] (a tool with no resource file), orphaned [status] (a resource file with no matching tool)
=== RUN   TestResourceFileSetMatchesToolNames/resourceURIFor_keys_match_actual_filenames
=== RUN   TestResourceFileSetMatchesToolNames/per-tool_URI_shape_(D-09)
    resources_schema_drift_test.go:167: resourceURIFor has no entry for health.md
=== RUN   TestResourceFileSetMatchesToolNames/behavior-doc_URI_shape_(D-10)
=== RUN   TestResourceFileSetMatchesToolNames/tools-filter_prose_names_exactly_the_registered_companions
    resources_schema_drift_test.go:212: tools-filter.md never mentions codegraph_health, but it is a registered companion tool the filter can narrow away
    resources_schema_drift_test.go:224: tools-filter.md names codegraph_status, which is neither a registered companion tool nor codegraph_explore — a renamed or removed tool left behind in this doc's prose
--- FAIL: TestResourceFileSetMatchesToolNames (0.00s)
    --- FAIL: TestResourceFileSetMatchesToolNames/file_set_matches_allToolNames_plus_behavior_docs (0.00s)
    --- PASS: TestResourceFileSetMatchesToolNames/resourceURIFor_keys_match_actual_filenames (0.00s)
    --- FAIL: TestResourceFileSetMatchesToolNames/per-tool_URI_shape_(D-09) (0.00s)
    --- PASS: TestResourceFileSetMatchesToolNames/behavior-doc_URI_shape_(D-10) (0.00s)
    --- FAIL: TestResourceFileSetMatchesToolNames/tools-filter_prose_names_exactly_the_registered_companions (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/mcp	0.341s
```

The named gate fails naming **both directions from one mutation** — `missing [health]` and
`orphaned [status]` in the same sub-test — which is what makes GUARD-02's structural check
bidirectional rather than a one-way existence check. The per-tool URI shape sub-test and the
tools-filter prose sub-test independently name the same drift a second and third way.

**Other tests that also went red (more evidence, not the named gate):** running the full package
(`go test ./internal/mcp/... -count=1`) with the mutation still applied additionally panics inside
`registerResources` — `resourceDescriptionFor` still switches on `companionNames`, which no longer
contains `"status"`, so any test that constructs a server via `BuildServer` (e.g.
`TestHandlerErrorIsToolResultNotProtocolError`) fails with `panic: mcp: resourceDescriptionFor:
unknown resource stem status`. This is corroborating evidence that the rename is structurally
unsafe well beyond the one named gate, not a substitute for it.

**Revert confirmation:** all three hunks were reverted with `git checkout -- internal/mcp/server.go
internal/mcp/tools.go`; `git status --porcelain -- internal/` and `git diff --exit-code --
internal/` both showed nothing changed; `go build ./...` exited 0; `go test ./internal/mcp/...
-count=1` exited 0 (15.1s).

---

## Mutation 8 — a changed engine constant, code side only (05-03-PLAN Task 3, GUARD-01)

**Requirement:** GUARD-01's code-side half — mutating `internal/query/validate.go`'s
`defaultDepth` must turn `TestMCPToolSchemaNumericClaimsMatchEngineConstants` red, while
`TestMCPResourceNumericClaimsMatchToolSchemas` — which never reads `validate.go` at all, only
compares the resource markdown against the tool schema — must stay green, because both sides of
THAT comparison (the schema's own claim, the resource's own claim) are unchanged by this edit.

**Edit:** `internal/query/validate.go`:

```diff
-	defaultDepth = 2
+	defaultDepth = 4
```

**Confirmed applied:** `git diff -- internal/query/validate.go` showed exactly this one-line
change; `go build ./...` exited 0.

**Gate that went red, and the gate that stayed green — run together
(`go test ./internal/mcp/... -run 'TestMCPToolSchemaNumericClaimsMatchEngineConstants|TestMCPResourceNumericClaimsMatchToolSchemas' -count=1 -v`):**

```
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/explore
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/node
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/search
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/callers
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/callees
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/impact
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/files
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/status
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/tools-filter
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/index-state
--- PASS: TestMCPResourceNumericClaimsMatchToolSchemas (0.00s)
    [all 10 subtests PASS]
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_callers
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_callees
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_impact
    tools_schema_drift_test.go:115: codegraph_impact advertises default 2 but internal/query.defaultDepth is 4 — MCP clients are being told the wrong value (description: "BFS depth (default 2, max 50)")
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_files
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_status
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_explore
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_node
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_search
--- FAIL: TestMCPToolSchemaNumericClaimsMatchEngineConstants (0.00s)
    --- PASS: TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_callers (0.00s)
    --- PASS: TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_callees (0.00s)
    --- FAIL: TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_impact (0.00s)
    --- PASS: TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_files (0.00s)
    --- PASS: TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_status (0.00s)
    --- PASS: TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_explore (0.00s)
    --- PASS: TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_node (0.00s)
    --- PASS: TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_search (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/mcp	0.285s
```

**The precise asymmetry, recorded honestly rather than glossed:**
`TestMCPToolSchemaNumericClaimsMatchEngineConstants` fired, because it compares `codegraph_impact`'s
schema claim ("default 2", still literally in `tools.go`'s jsonschema tag) against the mutated
`internal/query.defaultDepth` (now 4) — the two sides it compares moved apart.
`TestMCPResourceNumericClaimsMatchToolSchemas` stayed green, because it compares
`resources/impact.md`'s claim ("default 2") against `codegraph_impact`'s SCHEMA claim ("default
2") — neither side of THAT comparison was touched by this mutation, since it never reads
`validate.go`. This is exactly the coverage boundary
`TestMCPResourceNumericClaimsMatchToolSchemas`'s own doc comment states: "Editing
internal/query/validate.go alone fails the EXISTING schema test … not this one." Mutation 9 below
is this mutation's mirror image, moving the OTHER side of the resource-vs-schema comparison
instead.

**Revert confirmation:** the constant was restored to `2`; `git checkout --
internal/query/validate.go`; `git status --porcelain -- internal/` and `git diff --exit-code --
internal/` both showed nothing changed; `go build ./...` exited 0; `go test ./internal/mcp/...
-count=1` exited 0.

---

## Mutation 9 — a changed resource claim, prose side only (05-03-PLAN Task 3, GUARD-01)

**Requirement:** GUARD-01's prose-side half — mutating `internal/mcp/resources/impact.md`'s stated
BFS depth default, touching nothing else, must turn `TestMCPResourceNumericClaimsMatchToolSchemas`
red while `TestMCPToolSchemaNumericClaimsMatchEngineConstants` — which never reads any `.md` file,
only compares the tool schema against `internal/query`'s constants — stays green. The exact mirror
image of Mutation 8.

**Edit:** `internal/mcp/resources/impact.md`:

```diff
-- `depth` (integer, optional) — BFS depth (default 2, max 50).
+- `depth` (integer, optional) — BFS depth (default 9, max 50).
```

**Confirmed applied:** `git diff -- internal/mcp/resources/impact.md` showed exactly this one-line
change; `go build ./...` exited 0 (the mutated file is embedded via `//go:embed`, so a rebuild
picked it up with no separate step).

**Gate that went red, and the gate that stayed green — run together (same command as Mutation
8):**

```
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/explore
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/node
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/search
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/callers
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/callees
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/impact
    resources_schema_drift_test.go:356: resources/impact.md's numeric claims map[default 9:1 max 50:1] do not match its tool schema's claims map[default 2:1 max 50:1]
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/files
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/status
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/tools-filter
=== RUN   TestMCPResourceNumericClaimsMatchToolSchemas/index-state
--- FAIL: TestMCPResourceNumericClaimsMatchToolSchemas (0.00s)
    --- PASS: TestMCPResourceNumericClaimsMatchToolSchemas/explore (0.00s)
    --- PASS: TestMCPResourceNumericClaimsMatchToolSchemas/node (0.00s)
    --- PASS: TestMCPResourceNumericClaimsMatchToolSchemas/search (0.00s)
    --- PASS: TestMCPResourceNumericClaimsMatchToolSchemas/callers (0.00s)
    --- PASS: TestMCPResourceNumericClaimsMatchToolSchemas/callees (0.00s)
    --- FAIL: TestMCPResourceNumericClaimsMatchToolSchemas/impact (0.00s)
    --- PASS: TestMCPResourceNumericClaimsMatchToolSchemas/files (0.00s)
    --- PASS: TestMCPResourceNumericClaimsMatchToolSchemas/status (0.00s)
    --- PASS: TestMCPResourceNumericClaimsMatchToolSchemas/tools-filter (0.00s)
    --- PASS: TestMCPResourceNumericClaimsMatchToolSchemas/index-state (0.00s)
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_node
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_search
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_callers
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_callees
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_impact
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_files
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_status
=== RUN   TestMCPToolSchemaNumericClaimsMatchEngineConstants/codegraph_explore
--- PASS: TestMCPToolSchemaNumericClaimsMatchEngineConstants (0.00s)
    [all 8 subtests PASS]
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/mcp	0.304s
```

`resources/impact.md`'s claim moved to "default 9" while `codegraph_impact`'s schema claim stayed
"default 2" (`tools.go`'s jsonschema tag untouched) — the resource-vs-schema comparison catches it,
naming the `impact` stem and printing both multisets. `TestMCPToolSchemaNumericClaimsMatchEngineConstants`
never reads `impact.md` at all, so it cannot see this edit and stays green — the schema claim
("default 2") and `internal/query.defaultDepth` (still 2) remain in agreement.

**Revert confirmation:** the file was restored to "default 2, max 50"; `git checkout --
internal/mcp/resources/impact.md`; `git status --porcelain -- internal/` and `git diff --exit-code
-- internal/` both showed nothing changed; `go build ./...` exited 0; `go test ./internal/mcp/...
-count=1` exited 0.

---

## A note on numbering (05-04-PLAN Task 3)

05-04-PLAN's own Task 3 text was written against an assumption that these two mutations would be
numbered 9 and 10, and its acceptance criteria literally check `rg -c '^## Mutation' … returns 10`
with the capability-off proof named `## Mutation 9`. By the time this task actually ran,
05-03-PLAN's own Task 3 had already appended Mutations 7, 8, and 9 above (the tool-rename/GUARD-02
proof and the two GUARD-01 code/prose-side proofs) — nine mutations existed, not eight, and slot 9
was already taken. Rather than renumber or overwrite an already-committed, already-reverted
mutation record, the two mutations below continue the sequence as Mutations 10 and 11. This is
recorded here plainly rather than silently reconciled, matching 05-03-PLAN's own precedent for the
identical situation (see "A note on numbering (05-03-PLAN Task 3)" above): the plan's literal
acceptance-criteria commands were written against a stale assumption and do not hold against the
actual, current file; the corrected verification command is `rg -c '^## Mutation'
test/wireoracle/MUTATION-PROOF.md` returning **11**, with the capability-off/RSRC-03-capability
proof at `## Mutation 10` and the registration-gating/RSRC-03-structural proof at `## Mutation 11`.
Both mutations named in 05-04-PLAN's success criteria (criterion 3's non-vacuity re-proof,
criterion 3's RSRC-03 index-independence proof) are demonstrated below regardless of the number
attached to each.

---

## Mutation 10 — the resources capability, explicit zero value removed (05-04-PLAN Task 3, D-11/T-05-03)

**Requirement:** criterion 5's "the oracle re-proved non-vacuous" — demonstrate what
`Resources: &mcp.ResourceCapabilities{}`'s EXPLICIT zero value in `BuildServer`'s
`mcp.ServerCapabilities` literal prevents, mirroring Mutation 5's capability-off proof for
`Tools.ListChanged` and D-11's own stated rationale for never leaving a capability's presence to
SDK inference.

**Edit:** `internal/mcp/server.go`, `BuildServer`'s capability construction:

```diff
 		Capabilities: &mcp.ServerCapabilities{
-			// RSRC-03/D-11 extension: explicit zero value, not omission.
-			// go-sdk's capabilities() (server.go:645-653) would otherwise
-			// auto-populate caps.Resources with ListChanged: true purely
-			// because s.resources.len() > 0 once registerResources below
-			// runs — this phase implements neither listChanged nor
-			// subscribe, so the explicit zero value is what keeps the
-			// advertised capability truthful, mirroring D-11's own
-			// rationale for Tools above.
-			Resources: &mcp.ResourceCapabilities{},
-			Tools:     &mcp.ToolCapabilities{ListChanged: true},
+			Tools: &mcp.ToolCapabilities{ListChanged: true},
 		},
```

**Confirmed applied:** `git diff -- internal/mcp/server.go` showed exactly this hunk (the
`Resources:` field and its doc comment deleted, `Tools:` left unchanged); `go build ./...` exited
0.

**Nothing removed from the wire — a strictly more interesting failure than absence:** this does NOT
remove `capabilities.resources` from the wire. go-sdk's own `capabilities()` method
(`server.go:645-653`) auto-populates `caps.Resources` with `ListChanged: true` once any resource is
registered — which `registerResources` still does unconditionally, since this mutation does not
touch its call site. So the advertised capability silently CHANGES rather than disappears: it
becomes a promise of `listChanged` support this server does not implement. This is the concrete,
observed justification for D-11's rule that a capability is never left to SDK inference.

**Gate that went red:** `TestFrozenTranscriptsMatch` failed on every one of the 38 scenarios whose
`initialize`/`server/discover` response carries a `capabilities` object — the same 38 = 42 total
scenarios minus the 4 with no `capabilities` object at all (`edge-call-before-initialize`,
`modern-listen-catalog-change`, `modern-meta-invalid-params`, `modern-meta-unsupported-version`),
confirmed by exact count against the full scenario list. Measured blast radius: **38**.

**`go test ./test/wireoracle/... -count=1`'s verbatim failure** (one representative instance kept,
the remaining 37 named):

```
--- FAIL: TestFrozenTranscriptsMatch (4.43s)
    --- FAIL: TestFrozenTranscriptsMatch/handshake-explore (0.63s)
        oracle_test.go:130: scenario "handshake-explore": normalized transcript differs at line 1:
             got: "...{\"capabilities\":{\"resources\":{\"listChanged\":true},\"tools\":{\"listChanged\":true}},...\"protocolVersion\":\"2025-11-25\",...}"
            want: "...{\"capabilities\":{\"resources\":{},\"tools\":{\"listChanged\":true}},...\"protocolVersion\":\"2025-11-25\",...}"
    [... 37 more scenarios FAIL identically on this same collateral diff: toolslist-default,
    toolslist-narrowed, toolslist-filter-empty, toolslist-no-index, toolslist-repeat, call-node,
    call-search, call-callers, call-callees, call-impact, call-files, call-status,
    error-unknown-method, error-unknown-tool, error-malformed-args, error-confinement-reject,
    edge-call-before-initialize is NOT in this list (no capabilities object), legacy-2025-11-25,
    legacy-2025-06-18, legacy-2025-03-26, legacy-2024-11-05, legacy-unsupported-2026-07-28,
    legacy-omitted-version, modern-discover-explore, index-appears-mid-session,
    resources-list, resources-read-explore, resources-read-node, resources-read-search,
    resources-read-callers, resources-read-callees, resources-read-impact, resources-read-files,
    resources-read-status, resources-read-tools-filter, resources-read-index-state,
    resources-list-no-index, resources-read-unknown ...]
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/wireoracle	22.072s
```

**The observed wire value:** every failing scenario's `got` shows `"capabilities":{"resources":
{"listChanged":true},...}` where the frozen `want` shows `"capabilities":{"resources":{},...}` —
exactly the SDK-inferred `ListChanged: true` the "nothing removed" paragraph above predicted,
confirmed on the wire rather than asserted from reading `server.go`.

**Revert confirmation:** the `Resources: &mcp.ResourceCapabilities{}` field and its doc comment were
restored via `git checkout -- internal/mcp/server.go`; `git status --porcelain -- internal/` and
`git diff --exit-code -- internal/` both showed nothing changed; `go build ./...` exited 0; `go test
./test/wireoracle/... -count=1` exited 0 (all 42 scenarios); `go test ./internal/mcp/... -count=1`
exited 0.

---

## Mutation 11 — resource registration moved inside `if hasIndex` (05-04-PLAN Task 3, RSRC-03)

**Requirement:** proves RSRC-03 (resources register unconditionally, independent of index state) is
a STRUCTURAL property of `registerResources`'s call-site position, not an incidental one — the
asymmetry this mutation produces (9 `Index: false` resource scenarios red, the 2 `Index: true`
resource scenarios green) is what demonstrates that, not just that removing the call site breaks
something.

**Edit:** `internal/mcp/server.go`, `BuildServer` — moves the `registerResources(s)` call from its
position immediately after `mcp.NewServer(...)` (unconditional) into the `if hasIndex {` block that
gates `registerTools`:

```diff
@@ construction, immediately after mcp.NewServer(...) @@
-	// RSRC-03: registerResources runs unconditionally, immediately after
-	// construction and BEFORE the `if hasIndex {` tool-registration branch
-	// below — this call site's position outside that branch is the
-	// structural property that makes resources available even in an
-	// unindexed repository, mirroring how Capabilities.Tools above is set
-	// regardless of hasIndex.
-	registerResources(s)
-
 	// One gitmeta.CachingDetector per SERVER, not per handler or per call
@@ later, at the existing hasIndex gate @@
 	if hasIndex {
+		registerResources(s)
 		toolCount.Store(int64(registerTools(s, companions, repoPath, startPath, detector)))
 	}
```

**Confirmed applied:** `git diff -- internal/mcp/server.go` showed exactly these two hunks (the
unconditional call site and its doc comment deleted; a bare `registerResources(s)` call added as
the first statement inside the existing `if hasIndex {` block); `go build ./...` exited 0.

**Gate that went red, and the gate that stayed green:** `go test ./test/wireoracle/... -count=1`:

```
--- FAIL: TestFrozenTranscriptsMatch (4.49s)
    --- FAIL: TestFrozenTranscriptsMatch/resources-read-node (0.02s)
        oracle_test.go:130: scenario "resources-read-node": normalized transcript differs at line 2:
             got: "{\"jsonrpc\":\"2.0\",\"id\":2,\"error\":{\"code\":-32602,\"message\":\"Resource not found\",\"data\":{\"uri\":\"codegraph://tools/node\"}}}"
            want: "{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"ttlMs\":0,\"cacheScope\":\"private\",\"contents\":[{\"uri\":\"codegraph://tools/node\",...\"text\":\"# codegraph_node\\n\\n...\"}]}}"
    [... 7 more Index: false read scenarios FAIL identically on the same -32602 "Resource not
    found" collateral: resources-read-search, resources-read-callers, resources-read-callees,
    resources-read-impact, resources-read-files, resources-read-status,
    resources-read-tools-filter, resources-read-index-state]
    --- FAIL: TestFrozenTranscriptsMatch/resources-list-no-index (0.01s)
        oracle_test.go:130: scenario "resources-list-no-index": normalized transcript differs at line 2:
             got: "{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"ttlMs\":0,\"cacheScope\":\"private\",\"resources\":[]}}"
            want: "{\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{\"ttlMs\":0,\"cacheScope\":\"private\",\"resources\":[{...10 entries...}]}}"
--- FAIL: TestEveryAdvertisedResourceURIHasASuccessfulReadScenario (0.37s)
    oracle_test.go:1000: no scenario provides a successful resources/read for: [codegraph://index-state
    codegraph://tools-filter codegraph://tools/callees codegraph://tools/callers
    codegraph://tools/files codegraph://tools/impact codegraph://tools/node codegraph://tools/search
    codegraph://tools/status]
FAIL
FAIL	github.com/seanb4t/codegraph-go/test/wireoracle	22.072s
```

**Scenarios that went RED (10):** `resources-read-node`, `resources-read-search`,
`resources-read-callers`, `resources-read-callees`, `resources-read-impact`,
`resources-read-files`, `resources-read-status`, `resources-read-tools-filter`,
`resources-read-index-state` — all 9 `Index: false` resource read scenarios — plus
`resources-list-no-index` (the full-catalog-with-no-index scenario). Every one of these 10 shares
`Index: false` (10 total). `TestEveryAdvertisedResourceURIHasASuccessfulReadScenario` also went red,
naming the same 9 URIs as having no successful wire read.

**Scenarios that stayed GREEN (2):** `resources-list` and `resources-read-explore` — both
`Index: true`. Their capture runs `codegraph init` before `serve --mcp` starts, so `hasIndex` is
true at construction time and `registerResources` still runs from its new position inside the
`if hasIndex {` block.

**The precise asymmetry this demonstrates:** every scenario that failed is `Index: false`; every
resource scenario that stayed green is `Index: true`. This is not "resources broke" — it is
specifically "resources now depend on index state," which is exactly RSRC-03's negation. The
9-red/2-green split is deterministic and structural, not incidental: it tracks `Index:` on the
scenario one-for-one, which is the proof that the guard is measuring index-independence
specifically, not merely that resources exist somewhere in the corpus.

**Same property caught at the unit tier:** `go test ./internal/mcp/... -run
TestResourcesRegisterWithoutIndex -count=1 -v` also goes red under this mutation:

```
=== RUN   TestResourcesRegisterWithoutIndex
    resources_test.go:123: resources/list without an index = [], want the same URI set as with an
    index [codegraph://index-state codegraph://tools-filter codegraph://tools/callees
    codegraph://tools/callers codegraph://tools/explore codegraph://tools/files
    codegraph://tools/impact codegraph://tools/node codegraph://tools/search
    codegraph://tools/status]
--- FAIL: TestResourcesRegisterWithoutIndex (0.05s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/mcp	0.335s
```

RSRC-03's index-independence property is therefore caught at both the unit tier (in-process,
`BuildServer(false, ...)`, no subprocess) and the wire tier (a real spawned `serve --mcp`) —
independent proofs of the same structural claim.

**Revert confirmation:** the `registerResources(s)` call was moved back to its unconditional
position via `git checkout -- internal/mcp/server.go`; `git status --porcelain -- internal/` and
`git diff --exit-code -- internal/` both showed nothing changed; `go build ./...` exited 0; `go test
./test/wireoracle/... -count=1` exited 0 (all 42 scenarios); `go test ./internal/mcp/... -count=1`
exited 0.

---

## Non-vacuity proof for the remaining GUARD-01 checkers (05-03-PLAN Task 3)

The count checker (`TestResourceCountClaimsMatchSourceSets`/`countClaimsIn`), the env-var checker
(`TestResourceEnvVarNamesAreReal`), and the host-fact checker
(`TestResourceContentCarriesNoHostFacts`/`hostFactsIn`) are deliberately proven by synthetic
non-vacuity sub-tests (`TestResourceCountCheckerIsNotVacuous`,
`TestResourceHostFactCheckerIsNotVacuous`) rather than by a fourth, fifth, and sixth real-tree
mutation. This is the stronger form for these three, not a shortcut: a mutation proof is a
one-time, point-in-time demonstration recorded in this document, while a synthetic non-vacuity
sub-test asserts the checker's discriminating power on EVERY `go test ./internal/mcp/...` run,
forever, including runs long after this document is written and possibly forgotten. This is the
same honesty `instructions_contract_test.go`'s own doc comment applies to its literal anchors, and
09-03-PLAN's own text asked for it to be stated here rather than left for a reviewer to discover.
The env-var checker has no meaningful "wrong value" mutation to demonstrate against the real tree
either — `allowlistEnvName` is a single hard-coded constant with only one real value in this
codebase, so a mutation would only prove the checker can detect a typo it was never at risk of
missing (its own extraction logic, not a comparison against a second derived source, is what
`TestResourceEnvVarNamesAreReal` itself already exercises directly on the real tree, unmutated).

`go test ./internal/mcp/... -run 'IsNotVacuous' -count=1 -v` confirms all named non-vacuity tests
in this package pass, including the three above:

```
--- PASS: TestREADMEGateCheckerIsNotVacuous (0.00s)
--- PASS: TestResourceStemSetDiffIsNotVacuous (0.00s)
--- PASS: TestResourceCountCheckerIsNotVacuous (0.00s)
--- PASS: TestResourceHostFactCheckerIsNotVacuous (0.00s)
--- PASS: TestResourcesReadIsNotVacuous (0.05s)
```

---

## Closing statement (05-03-PLAN Task 3)

`git status --porcelain -- internal/` was checked after all three mutations above were reverted and
before this section and `internal/mcp/resources_schema_drift_test.go` were committed — the
cleanliness check's purpose is to prove no mutation residue survives, matching the sequencing
discipline the original "Closing statement" above set for Phase 1's four mutations. All three
mutations were reverted; `go build ./...`, `go test ./internal/mcp/... -count=1`, `go test
./test/wireoracle/... -count=1`, and `go test ./... -count=1` all passed on the reverted,
committed tree.

---

## Closing statement

`git status --porcelain` is checked AFTER `test/wireoracle/MUTATION-PROOF.md` and
`test/wireoracle/COVERAGE-BASELINE.md` are committed, per this plan's own sequencing instruction —
the cleanliness check's purpose is to prove no *mutation residue* survives, not to forbid this
task's own deliverables. Both documents were committed as this task's final step before the
cleanliness check ran.

All four mutations were reverted; the tree was confirmed clean (`git status --porcelain` empty)
before this document and `COVERAGE-BASELINE.md` were written and staged.

---

## Closing statement (05-04-PLAN Task 3)

`git status --porcelain -- internal/` was checked after Mutations 10 and 11 above were reverted and
before this section and `COVERAGE-BASELINE.md`'s finalization were committed — the same sequencing
discipline 05-03-PLAN's own "Closing statement" set. Both mutations were reverted; `go build ./...`,
`go test ./internal/mcp/... -count=1`, `go test ./test/wireoracle/... -count=1`, and `go test
./... -count=1` all passed on the reverted, committed tree.

---

## A note on numbering (06-03-PLAN Task 3)

`grep -c '^## Mutation' test/wireoracle/MUTATION-PROOF.md` was run before appending anything below,
per this plan's own explicit instruction not to assume a number from the plan text — 05-03-PLAN's
own summary records that exact assumption failing once already. The count returned **11**, with the
highest existing heading `## Mutation 11 — resource registration moved inside \`if hasIndex\`
(05-04-PLAN Task 3, RSRC-03)`. The five mutations below continue the sequence as Mutations 12
through 16.

---

## Mutation 12 — a renamed companion tool, `node` to `peek` (06-03-PLAN Task 3, T-06-05)

**Requirement:** GUARD-01 extended to SKILL.md — a tool renamed at its source without SKILL.md's
own text moving with it must turn `TestSkillNamesOnlyRealTools` red, naming the now-unregistered
token SKILL.md still carries.

**Edit:** three sites renamed the `node` companion to `peek`, deliberately leaving
`.claude/skills/codegraph/SKILL.md` untouched (it still names `codegraph_node` three times, in the
decision table, the full-reference list, and nowhere else) — that omission IS the drift being
demonstrated.

`internal/mcp/server.go`, the `companionNames` slice:

```diff
-var companionNames = []string{"node", "search", "callers", "callees", "impact", "files", "status"}
+var companionNames = []string{"peek", "search", "callers", "callees", "impact", "files", "status"}
```

`internal/mcp/tools.go`, `companionTool`'s matching case:

```diff
-	case "node":
+	case "peek":
 		return &mcp.Tool{
-			Name:        "codegraph_node",
+			Name:        "codegraph_peek",
 			Description: "Show a symbol's signature, calls, and callers, or a line-numbered file read",
```

`internal/mcp/tools.go`, `companionHandler`'s matching case:

```diff
-	case "node":
+	case "peek":
 		mcp.AddTool(s, tool, func(ctx context.Context, req *mcp.CallToolRequest, args NodeArgs) (*mcp.CallToolResult, any, error) {
```

**Confirmed applied:** `git diff -- internal/mcp/server.go internal/mcp/tools.go` showed exactly
these three hunks; `go build ./...` exited 0.

**Gate that went red — the named gate, run in isolation
(`go test ./internal/mcp/... -run TestSkillNamesOnlyRealTools -count=1 -v`):**

```
=== RUN   TestSkillNamesOnlyRealTools
    skill_claims_drift_test.go:310: ../../.claude/skills/codegraph/SKILL.md names codegraph_node, which is not a member of allToolNames() — a renamed or removed tool left behind in the skill
    skill_claims_drift_test.go:310: ../../.claude/skills/codegraph/SKILL.md names codegraph_node, which is not a member of allToolNames() — a renamed or removed tool left behind in the skill
    skill_claims_drift_test.go:310: ../../.claude/skills/codegraph/SKILL.md names codegraph_node, which is not a member of allToolNames() — a renamed or removed tool left behind in the skill
--- FAIL: TestSkillNamesOnlyRealTools (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/mcp	0.345s
```

The gate fires once per occurrence of `codegraph_node` in SKILL.md (three sites: the decision
table, the full-reference list entry, and the closing "8 tools" sentence's list) — the same
"skill is the third surface" bug class T-06-05 names, this time demonstrated on the real tree
rather than only asserted.

**Revert confirmation:** all three hunks were reverted with `git checkout -- internal/mcp/server.go
internal/mcp/tools.go`; `git status --porcelain -- internal/` and `git diff --exit-code --
internal/` both showed nothing changed; `go build ./...` exited 0; `go test ./internal/mcp/...
-count=1` exited 0 (4.6s).

---

## Mutation 13 — a dead resource URI in SKILL.md's full-reference pointer (06-03-PLAN Task 3, T-06-05)

**Requirement:** GUARD-01 extended to SKILL.md — a `codegraph://` URI pointing at a resource stem
the server does not serve must turn `TestSkillResourceURIsResolve` red, and this entry additionally
records which OTHER SKILL.md guards stayed green from the same one-line edit, per 05-03-PLAN's own
asymmetry-recording discipline (Mutation 9's note applies the same practice here).

**Edit:** `.claude/skills/codegraph/SKILL.md`'s full-reference pointer for `codegraph_status`:

```diff
-- `codegraph_status` → `codegraph://tools/status`
+- `codegraph_status` → `codegraph://tools/healthcheck`
```

**Confirmed applied:** `git diff -- .claude/skills/codegraph/SKILL.md` showed exactly this hunk.

**Gate that went red — the named gate
(`go test ./internal/mcp/... -run TestSkillResourceURIsResolve -count=1 -v`):**

```
=== RUN   TestSkillResourceURIsResolve
    skill_claims_drift_test.go:337: ../../.claude/skills/codegraph/SKILL.md names codegraph://tools/healthcheck, which is not a value in resourceURIFor — the skill points at a resource the server does not serve
--- FAIL: TestSkillResourceURIsResolve (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/mcp	0.298s
```

**Guards that stayed green from the same edit (asymmetry, recorded per 05-03-PLAN's discipline):**
running the full `TestSkill*` family with the mutation still applied
(`go test ./internal/mcp/... -run TestSkill -count=1 -v`) showed every other test passing —
`TestSkillFrontmatterIsSpecCompliant`, `TestSkillLeadsWithDecisionTable`,
`TestSkillStaysWithinBudget`, `TestSkillNamesOnlyRealTools` (the tool NAME `codegraph_status` is
still real; only the URI text next to it broke), `TestSkillDefersNumericFactsToResources`,
`TestSkillCountClaimsMatchSourceSets`, `TestSkillEnvVarNamesAreReal`, `TestSkillCarriesNoHostFacts`,
`TestSkillNamesTheFilterWhenItNamesCompanions`, `TestSkillCarriesExactlyThreeWorkedExamples`, and
both non-vacuity tests. This is the expected asymmetry: a broken resource pointer is a narrower
defect than a broken tool name, and only the one checker built to catch exactly that shape fired.

**Revert confirmation:** the hunk was reverted with `git checkout -- .claude/skills/codegraph/SKILL.md`;
`git diff --exit-code -- .claude/skills/codegraph/SKILL.md` showed nothing changed; `go test
./internal/mcp/... -run TestSkill -count=1` exited 0.

---

## Mutation 14 — the `resume` matcher changed in the go:embed fragment, not the live registration (06-03-PLAN Task 3, D-04/A2)

**Requirement:** the fragment/registration parity guard (`internal/agents/hookpackage_test.go`'s
`TestHookRegistrationMatchesFragmentAndScript`) must turn red when `.claude/hooks/hooks.json`
(Phase 7's `go:embed` source) drifts from `.claude/settings.json` (what this repository actually
runs), naming both files. 06-01-PLAN's own summary records having exercised "a targeted one-field
mutation to `hooks.json`" during that plan's own execution, but that run is not itself recorded in
this document and its target field is not specified in the summary text — this entry runs it fresh
against the field 06-03-PLAN names explicitly (the `resume` matcher) rather than relying on an
unspecified prior claim.

**Edit:** `.claude/hooks/hooks.json`'s second `SessionStart` entry:

```diff
       {
-        "matcher": "resume",
+        "matcher": "clear",
         "hooks": [
```

**Confirmed applied:** `git diff -- .claude/hooks/hooks.json` showed exactly this hunk.

**Gate that went red — the named gate
(`go test ./internal/agents/... -run 'TestHookRegistrationMatchesFragmentAndScript$' -count=1 -v`):**

```
=== RUN   TestHookRegistrationMatchesFragmentAndScript
    hookpackage_test.go:344: hooks.SessionStart differs between ../../.claude/settings.json and ../../.claude/hooks/hooks.json — Phase 7 would embed a fragment that differs from what actually runs here.
        settings.json: []interface {}{map[string]interface {}{"hooks":[]interface {}{map[string]interface {}{"command":"${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh", "type":"command"}}, "matcher":"startup"}, map[string]interface {}{"hooks":[]interface {}{map[string]interface {}{"command":"${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh", "type":"command"}}, "matcher":"resume"}}
        hooks.json:    []interface {}{map[string]interface {}{"hooks":[]interface {}{map[string]interface {}{"command":"${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh", "type":"command"}}, "matcher":"startup"}, map[string]interface {}{"hooks":[]interface {}{map[string]interface {}{"command":"${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh", "type":"command"}}, "matcher":"clear"}}
--- FAIL: TestHookRegistrationMatchesFragmentAndScript (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.091s
```

The failure names both `../../.claude/settings.json` and `../../.claude/hooks/hooks.json` in its
message and prints the full decoded block from each side, exactly the "names both files" property
the plan requires.

**Revert confirmation:** the hunk was reverted with `git checkout -- .claude/hooks/hooks.json`;
`git diff --exit-code -- .claude/hooks/hooks.json` showed nothing changed; `go test
./internal/agents/... -count=1` exited 0.

---

## Mutation 15 — one character changed in the nudge message (06-03-PLAN Task 3, D-06/NUDGE-01)

**Requirement:** `TestSessionNudgeBehavesPerIndexPresence`'s byte-equality assertion must go red on
a single-character change to the emitted text, proving the check is a byte-equality comparison and
not a substring/prefix check that a near-miss could slip past.

**Edit:** `.claude/hooks/session-nudge.sh`'s `printf` argument, trailing "questions" → "question"
(dropped one character):

```diff
-  printf '%s\n' 'This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y questions.'
+  printf '%s\n' 'This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y question.'
```

**Confirmed applied:** `git diff -- .claude/hooks/session-nudge.sh` showed exactly this hunk.

**Gate that went red — the named gate
(`go test ./internal/agents/... -run TestSessionNudgeBehavesPerIndexPresence -count=1 -v`):**

```
=== RUN   TestSessionNudgeBehavesPerIndexPresence
=== RUN   TestSessionNudgeBehavesPerIndexPresence/codegraph_dir_present,_env_set
    hookpackage_test.go:188: stdout = "This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y question.\n", want "This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y questions.\n"
=== RUN   TestSessionNudgeBehavesPerIndexPresence/no_codegraph_entry_at_all,_env_set
=== RUN   TestSessionNudgeBehavesPerIndexPresence/codegraph_present_as_a_regular_file,_env_set
=== RUN   TestSessionNudgeBehavesPerIndexPresence/codegraph_present_as_an_empty_directory,_env_set
    hookpackage_test.go:188: stdout = "This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y question.\n", want "This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y questions.\n"
=== RUN   TestSessionNudgeBehavesPerIndexPresence/env_unset,_cmd.Dir_indexed
    hookpackage_test.go:188: stdout = "This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y question.\n", want "This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y questions.\n"
=== RUN   TestSessionNudgeBehavesPerIndexPresence/env_unset,_cmd.Dir_unindexed
--- FAIL: TestSessionNudgeBehavesPerIndexPresence (0.06s)
    --- FAIL: TestSessionNudgeBehavesPerIndexPresence/codegraph_dir_present,_env_set (0.03s)
    --- PASS: TestSessionNudgeBehavesPerIndexPresence/no_codegraph_entry_at_all,_env_set (0.01s)
    --- PASS: TestSessionNudgeBehavesPerIndexPresence/codegraph_present_as_a_regular_file,_env_set (0.01s)
    --- FAIL: TestSessionNudgeBehavesPerIndexPresence/codegraph_present_as_an_empty_directory,_env_set (0.01s)
    --- FAIL: TestSessionNudgeBehavesPerIndexPresence/env_unset,_cmd.Dir_indexed (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.124s
```

The two sub-cases where `.codegraph/` is absent still pass unaffected (no output expected either
way), and every sub-case where the nudge fires fails on the exact byte mismatch — confirming the
assertion is byte-equality, not a substring check that a dropped character could pass silently.

**Revert confirmation:** the hunk was reverted with `git checkout -- .claude/hooks/session-nudge.sh`;
`git diff --exit-code -- .claude/hooks/session-nudge.sh` showed nothing changed; `go test
./internal/agents/... -count=1` exited 0.

---

## Mutation 16 — the nudge script renamed on disk (06-03-PLAN Task 3, NUDGE-01/T-06-07)

**Requirement:** `TestHookRegistrationMatchesFragmentAndScript`'s command-path resolution assertion
must go red when the script a registration names no longer exists on disk — the check that a
silently-disabled nudge fails the build instead of failing quietly, per the plan's own framing.

**Edit:** `.claude/hooks/session-nudge.sh` renamed to `.claude/hooks/session-nudge-renamed.sh` via
`mv` (a filesystem rename, not a source edit — there is no diff to show; `ls .claude/hooks/`
confirmed the script was absent under its registered name and present under the new one).

**Gate that went red — the named gate
(`go test ./internal/agents/... -run 'TestHookRegistrationMatchesFragmentAndScript$' -count=1 -v`):**

```
=== RUN   TestHookRegistrationMatchesFragmentAndScript
    hookpackage_test.go:377: command path "${CLAUDE_PROJECT_DIR}/.claude/hooks/session-nudge.sh" (resolved "../../.claude/hooks/session-nudge.sh") does not exist: stat ../../.claude/hooks/session-nudge.sh: no such file or directory
--- FAIL: TestHookRegistrationMatchesFragmentAndScript (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.053s
```

This is the failure mode the plan calls out explicitly: a renamed or deleted script does not make
the nudge silently stop firing with nothing noticing — it fails the build.

**Revert confirmation:** the file was renamed back with `mv .claude/hooks/session-nudge-renamed.sh
.claude/hooks/session-nudge.sh`; `git status --porcelain -- .claude/` showed nothing changed (the
file was never staged or committed under the renamed name, so there is no git-tracked residue to
revert); `go test ./internal/agents/... -count=1` exited 0.

---

## Closing statement (06-03-PLAN Task 3)

`git status --porcelain` was checked after all five mutations above were reverted and before this
section was written and staged — the same cleanliness-proves-no-residue discipline every prior
Closing statement in this document set. All five mutations were reverted; the tree showed no diff
(`git diff --exit-code` exited 0) before this document's edits were staged.

`go test ./... -count=1` was then run on the reverted tree TWICE (once to verify the reverted tree
directly, once again to satisfy this task's own `<verify>` block). `internal/daemon` failed each
time, but on a DIFFERENT named test each run —
`TestRunWatchdogCancelsRunOnSimulatedReparent` the first time, `TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock`
the second — both plain timeouts inside the same package under the same full-suite concurrent load.
This is the load-dependent condition this repository's own `STATE.md` already documents (GitHub
issue #17, "Daemon extreme-load tail (ACCEPTED, not a gap)" — orphaned-goroutine root cause fixed in
Phase 4, one plain-timeout failure still observed under pathological workstation load, 52/52 real
CI runs clean on the actual runner class); a different test name surfacing each run is consistent
with a load-timing race rather than a deterministic regression this plan's changes introduced — none
of this task's five mutations touch `internal/daemon` or anything it imports. Both named tests were
re-run in isolation and both passed:
`go test ./internal/daemon/... -run TestRunWatchdogCancelsRunOnSimulatedReparent -count=1 -v` passed
in 1.06s; `go test ./internal/daemon/... -run TestDaemonRunWaitsForInFlightFlushBeforeReleasingLock
-count=1 -v` passed in 5.17s — both matching STATE.md's own description of this condition exactly
("fails under full-suite load, passes isolated"). This was identified as the documented pre-existing
condition, not absorbed as a new failure, per this plan's own acceptance criteria. Every other
package in both `go test ./... -count=1` runs passed, including `internal/mcp` (4.1-4.6s),
`internal/agents` (0.4s, part of the passing set), and `test/wireoracle` (36.9-38.7s).

`grep -c '^## Mutation' test/wireoracle/MUTATION-PROOF.md` on the committed file returns **16** —
the pre-task count of 11 plus the 5 entries appended by this task.
