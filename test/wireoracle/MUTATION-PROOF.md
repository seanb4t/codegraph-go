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

## Closing statement

`git status --porcelain` is checked AFTER `test/wireoracle/MUTATION-PROOF.md` and
`test/wireoracle/COVERAGE-BASELINE.md` are committed, per this plan's own sequencing instruction —
the cleanliness check's purpose is to prove no *mutation residue* survives, not to forbid this
task's own deliverables. Both documents were committed as this task's final step before the
cleanliness check ran.

All four mutations were reverted; the tree was confirmed clean (`git status --porcelain` empty)
before this document and `COVERAGE-BASELINE.md` were written and staged.
