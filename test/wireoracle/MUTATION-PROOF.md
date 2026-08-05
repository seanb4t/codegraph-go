# Wire Oracle Mutation Matrix — D-07 Non-Vacuity Proof

**Date:** 2026-08-05
**Phase:** 01-protocol-scoping-the-sdk-independent-wire-oracle, plan 01-07, Task 2
**Requirement:** D-07 — a gate is not trusted until it has been demonstrated RED against a
confirmed-applied mutation.

This is the one-time mutation matrix run against the real, built `codegraph` binary. For each
mutation: the edit was applied, **confirmed applied** by inspecting the changed file (`git diff`)
and rebuilding, `task test:wireoracle` was run and its verbatim failure recorded, then the edit was
reverted and the suite confirmed green again. All four mutations below were executed exactly as
described — none are described without having been run.

Baseline precondition, confirmed before any mutation: `task test:wireoracle` exits 0 on the
unmodified tree (23 scenarios).

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
