---
phase: 1
reviewers: [codex]
reviewed_at: 2026-08-22T21:20:00Z
review_cycle: 7
plans_reviewed:
  - 01-01-PLAN.md
  - 01-02-PLAN.md
  - 01-03-PLAN.md
  - 01-04-PLAN.md
  - 01-05-PLAN.md
  - 01-06-PLAN.md
  - 01-07-PLAN.md
  - 01-08-PLAN.md
  - 01-09-PLAN.md
  - 01-10-PLAN.md
  - 01-11-PLAN.md
models:
  codex: "gpt-5.6-sol (reasoning=low)"
model_sources:
  codex: "banner"
---

# Cross-AI Plan Review — Phase 1

## Codex Review

# Cross-AI Plan Review

## Summary

The phase architecture is directionally strong: it honors both correction blocks, puts the missing golden oracle before the Engine extraction, separates FIX-01 from the response-ordering flake, and sequences the secure middleware before the RPC expansion. However, several implementation contradictions make the current plans unsafe to execute unchanged. The most important are the `codegraph ui` URL/browser lifecycle, the incomplete multi-definition `NodeDetail` model, status/commit data plumbing, the claim that all RPCs can coexist with a long-lived store holder, and several tests that cannot prove what their acceptance criteria say. Overall risk is **HIGH until the blocking issues below are corrected**.

## Plan 01-01 — Transport tracer

### Strengths

- The guard correctly belongs outside the mux. This is consistent with the intended boundary: wrapping the whole `http.ServeMux` protects both Connect and future SPA routes.
- Exact Host matching is well chosen. The source listener will be a concrete loopback endpoint, and the proposed negative cases prevent prefix/suffix mistakes.
- Per-request Engine lifecycle follows the real `query.OpenAt` contract, which returns an Engine and idempotent closer together ([internal/query/engine.go:160](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/engine.go:160), [internal/query/engine.go:190](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/engine.go:190)).
- Root command registration is correctly identified as mandatory; commands are registered centrally in `root.AddCommand` ([internal/cli/root.go:52](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/root.go:52)).
- The plan correctly avoids h2c and Connect interceptors for the rebinding control.

### Concerns

- **HIGH — 01-01:** The command cannot print the bound URL before calling `Server.Run` under the proposed API. `New` only stores `127.0.0.1:0`; `Run` performs `net.Listen` and discovers the real port. Task 3 nevertheless says to call `srv.URL()`, print it, and only then call `srv.Run(ctx)`. At that point there is no port to print.
- **HIGH — 01-01:** Browser opening has the same sequencing problem. It must occur after successful bind and URL publication, but the plan splits the decision into CLI code while binding happens inside the blocking `Run`.
- **MEDIUM — 01-01:** `OpenBrowser` is placed in `uiserver.Options`, but Task 3 also assigns browser-launch responsibility to the CLI. Ownership is unclear and invites either an unused option or duplicate launch behavior.
- **MEDIUM — 01-01:** `TestUIServerHoldsNoStoreHandleBetweenCalls` opening the store after two calls proves only that the handle is closed at the observation point. It does not prove that nothing is cached but temporarily closed. The later structural test is the better complement.
- **LOW — 01-01:** The backstop statement about interrupted concurrent `buf generate` is explicitly unverifiable until 01-05 and should not be a `must_haves.truths` item for this plan.

### Suggestions

- Split lifecycle into `Listen` and `Serve`, or make `New` bind immediately:

  ```go
  srv, err := uiserver.Listen(opts)
  fmt.Fprintln(out, srv.URL())
  maybeOpen(srv.URL())
  return srv.Serve(ctx)
  ```

- Keep browser policy and launch in the CLI; remove `OpenBrowser` from server options.
- Add a test that the printed URL is connectable immediately after it is emitted.

---

## Plan 01-02 — Golden byte-identity oracle

### Strengths

- This plan correctly honors the D-04 correction. The current guard verifies only fixture existence, envelope shape, parsing, and non-empty output; it does not invoke the Engine ([testdata/golden/golden_test.go:243](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/golden/golden_test.go:243), [testdata/golden/golden_test.go:275](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/golden/golden_test.go:275)).
- Authoritative enumeration is already available and explicitly totals 26 frozen captures ([testdata/golden/golden_test.go:170](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/golden/golden_test.go:170), [testdata/golden/golden_test.go:317](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/golden/golden_test.go:317)).
- Reusing `buildEngineAt` is correct. It copies/indexes the corpus and opens the resulting store through `query.OpenAt`, preventing the documented repo-root pollution problem ([testdata/golden/behavioral_test.go:225](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/golden/behavioral_test.go:225)).
- The plan correctly uses the locked capture parameters rather than inventing new queries; the authoritative values are at [testdata/golden/gocapture/main.go:70](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/golden/gocapture/main.go:70).
- The RED mutation and permanent comparison-helper test are good complementary evidence.

### Concerns

- **HIGH — 01-02:** The `compared` counter is incremented inside subtests. If any subtest calls `t.Fatal`, control exits that subtest and the parent continues, but the final count can become an additional failure rather than a reliable statement of how many comparisons were attempted. More importantly, it is unsafe if subtests later become parallel. The plan should distinguish `attempted`, `completed`, and `matched`.
- **MEDIUM — 01-02:** The MCP captures cannot simply call `Engine.Node`/`Explore`; they must reproduce the actual MCP adapter semantics used by `gocapture`. The authoritative generator explicitly treats CLI and MCP as distinct capture paths. The plan mentions the same call shape but does not specify how inaccessible `package main` helpers are reproduced or shared.
- **MEDIUM — 01-02:** Copying `lockedCorpusArgs` creates a second authoritative table. A guard that merely checks every slug has an entry does not detect changed argument values in `gocapture`.
- **MEDIUM — 01-02:** “Every node-bearing pair fails when calledBy is removed” may be false for a valid node fixture that has zero callers. The mutation requirement should be based on measured affected fixtures, not assume every node output contains that section.

### Suggestions

- Extract locked capture specifications into an importable test-support package shared by `gocapture` and the oracle.
- Model each case with an explicit invocation kind: Engine Node, Engine Explore, MCP Node, or MCP Explore.
- Count cases before running subtests, and record outcomes through a parent-owned result table.
- Lock the expected mutation-failure set after measuring it rather than declaring that all node pairs must fail.

---

## Plan 01-03 — MCP pending-writer fix

### Strengths

- The plan correctly honors the second correction block. The current imbalance is real: inbound calls increment only when both method and id exist ([internal/mcp/server.go:271](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:271), [internal/mcp/server.go:323](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:323)), while every outbound write currently decrements ([internal/mcp/server.go:339](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:339)).
- Per-line classification is necessary because a Write may contain multiple JSON-RPC messages.
- The conservative malformed-output policy is reasonable: an extra bounded wait is safer than premature EOF.
- The plan appropriately leaves the response-ordering todo open and asserts responses by id, not arrival order.

### Concerns

- **HIGH — 01-03:** The proposed response sniff must classify only bytes successfully written. `io.Writer.Write` may return `n < len(b)` with an error. If the classifier scans all of `b`, it can decrement for a response that did not reach stdout, violating the source comment’s authoritative-wire invariant ([internal/mcp/server.go:218](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:218)).
- **HIGH — 01-03:** “Buffer partial lines” is ambiguous. `pendingWriter` must not delay bytes before forwarding them, because it is the protocol writer. It should write through immediately and buffer a copy of `b[:n]` solely for classification.
- **MEDIUM — 01-03:** A CAS clamp at zero can hide an invariant violation. It prevents negative values but can silently consume an unowed response classification. Prefer a helper returning whether a decrement happened, with a test-visible underflow counter or assertion.
- **MEDIUM — 01-03:** The planned `TestPipelinedToolsListOrderingIsIndependentOfNotifications` does not prove reordering “exhibits before and after.” By intentionally not asserting order, it proves only that both responses arrive. That is useful protocol coverage, but not an executable demonstration of nondeterministic ordering.
- **LOW — 01-03:** The exact-lost-response reproduction described at the primitive level does not demonstrate an actual response is abandoned; it demonstrates premature drain. The summary should phrase the evidence precisely unless a full stdio test confirms loss.

### Suggestions

- Forward first, then classify only `b[:n]`.
- Maintain an independent sniff buffer protected by a mutex if concurrent writes are possible.
- Add cases for short writes, write errors, `id:null`, JSON-RPC batch payloads if supported, and concurrent calls to `Write`.
- Rename the separability test to reflect what it actually proves: both pipelined responses are matched by id without notifications or ordering assumptions.

---

## Plan 01-04 — Engine seam extraction

### Strengths

- The extraction boundary is real and well located. Single-definition Node gathers calls and reverse callers immediately before `RenderNode` ([internal/query/node.go:420](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:420)).
- Multi-definition Node already builds reverse adjacency once and shares it across candidates ([internal/query/node.go:442](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:442)).
- Explore has five observed zero-result exits at lines 293, 357, 410, 473, and 558; the plan correctly requires all of them to funnel through a structured result ([internal/query/explore.go:293](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/explore.go:293), [internal/query/explore.go:558](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/explore.go:558)).
- The final Explore gather/render boundary is clear at [internal/query/explore.go:563](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/explore.go:563) through [internal/query/explore.go:595](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/explore.go:595).
- Dependency direction is correctly protected: query-layer structs should remain wire-independent.

### Concerns

- **HIGH — 01-04:** The proposed `NodeDetail` cannot represent the existing multi-definition output. `renderMultiDefNode` gathers source, calls, and called-by separately for every match ([internal/query/node.go:448](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:448)). The proposed public struct has only one `Node`, one `Source`, one `Calls`, one `CalledBy`, plus `Matches []*Node`. Meanwhile `gatherMultiDefNode` returns `[]NodeDetail`, but `gatherNode` returns only one `NodeDetail`. There is nowhere to retain the candidate details. `Node()` would have to regather them, violating D-01, or lose data, violating D-02.
- **HIGH — 01-04:** `ExploreResult` exposes fields typed as unexported `exploreFileGroup` and `exploreBlast` ([internal/query/explore.go:17](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/explore.go:17), [internal/query/explore.go:25](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/explore.go:25)). Although callers may access values obtained through fields, this is a poor exported seam and makes construction, documentation, and downstream testing awkward.
- **MEDIUM — 01-04:** The proposed assertion that nil versus empty Calls renders byte-identically does not prove the actual gather preserves nil/empty semantics elsewhere. It is incidental to renderer behavior, not an ENG-01 boundary requirement.
- **MEDIUM — 01-04:** `TestGatherReverseAdjacencyBuiltOnce` requires a new instrumentation seam not included in `files_modified`, or an invasive reader proxy. The plan should specify the mechanism.
- **LOW — 01-04:** The plan says to compare three error strings against literals but the source currently exposes at least argument and symbol-not-found strings in this path ([internal/query/node.go:317](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:317), [internal/query/node.go:341](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:341)); the third needs to be explicitly identified.

### Suggestions

Use an explicit sum shape:

```go
type NodeDetail struct {
    Mode       NodeDetailMode
    File       *FileDetail
    Definition *DefinitionDetail
    Candidates []DefinitionDetail
}
```

Export Explore component types (`ExploreFileGroup`, `ExploreBlast`) or introduce deliberately exported DTO-like query types. Keep render functions consuming those same types so there remains one gather.

---

## Plan 01-05 — Commit-aware Meta and proto drift

### Strengths

- Field 8 follows the existing additive Meta layout; fields 1–7 are occupied ([internal/schema/graph.proto:137](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/schema/graph.proto:137)).
- All three live metadata write sites are correctly identified: full indexing ([internal/indexer/resolve.go:754](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/indexer/resolve.go:754)), mtime-only sync ([internal/indexer/sync.go:165](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/indexer/sync.go:165)), and incremental sync ([internal/indexer/sync.go:393](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/indexer/sync.go:393)).
- The plan correctly avoids adding the SHA to CLI status. `StatusResult` is an explicit projection and currently has no commit field ([internal/query/status.go:47](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/status.go:47)).
- The drift guard covers both the new and pre-existing generated surfaces and includes a positive compared-file count.
- A one-way checkpoint for protobuf field allocation is appropriate.

### Concerns

- **HIGH — 01-05/01-06:** Adding `CommitSha` only to `schema.Meta` does not make it available to the UI Status mapper. `Engine.Status` currently returns `StatusResult`, which contains no Meta or commit field ([internal/query/status.go:47](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/status.go:47)). Plan 01-05 forbids touching `status.go`, while 01-06 says `statusToProto` sources `schema.IndexedCommitSHA`. The handler has only `StatusResult`, not the underlying Meta. The data path is missing.
- **MEDIUM — 01-05:** Running `git rev-parse` independently at each metadata write site adds a subprocess even for mtime-refresh-only sync. More importantly, the SHA should be resolved once per indexing operation and threaded to the chosen commit point to avoid inconsistent resolution if HEAD changes during a run.
- **MEDIUM — 01-05:** `TestMetaFieldNumbersAreStable` asserting exactly eight fields will fail on every legitimate future additive field. A descriptor compatibility fixture should assert known fields retain their numbers while permitting additional fields.
- **MEDIUM — 01-05:** In-place generation followed by `git diff` is not safe on a dirty worktree containing user edits to generated files; the task can overwrite those edits before reporting drift.
- **LOW — 01-05:** The test for no Git binary by emptying PATH may also prevent helper processes or test tooling from running, depending on implementation. Injecting the executable lookup is more deterministic.

### Suggestions

- Add `IndexedCommitSHA string` to `query.StatusResult` with `json:"-"`, or add a separate Engine metadata accessor consumed by the UI handler. This preserves CLI bytes while completing the data path.
- Resolve HEAD once at the beginning of an index/sync operation and pass it to all applicable Meta construction.
- Generate into a temporary tree and byte-compare against committed outputs.
- Make the compatibility test subset-based: known field-name → number mappings must remain stable; extra additive fields are allowed.

---

## Plan 01-06 — Full RPC surface

### Strengths

- The plan keeps mapping ownership in `internal/uiserver`, preserving the query/wire dependency direction.
- It correctly delegates validation and traversal limits to existing Engine methods. Search validates before scanning ([internal/query/search.go:153](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/search.go:153)); Impact validates and clamps depth ([internal/query/traverse.go:428](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/traverse.go:428)); Files validates depth and format ([internal/query/files.go:136](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/files.go:136)).
- Per-handler `query.OpenAt` plus deferred close is consistent with the actual store lifecycle.
- Positive method-set equality is stronger than a mutating-verb blacklist alone.
- Real Connect-client integration tests are appropriate.

### Concerns

- **HIGH — 01-06:** The Status commit-SHA mapping is impossible without resolving the missing data path from 01-05, as noted above.
- **HIGH — 01-06:** Error classification “by Engine sentinels” is underspecified and partly incompatible with current source. Node not-found is a formatted error string, not an exported sentinel ([internal/query/node.go:341](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:341)). The plan forbids string matching but does not schedule typed query errors.
- **MEDIUM — 01-06:** Exactly nine literal `query.OpenAt` occurrences duplicates lifecycle boilerplate and makes future correctness harder. A single `withEngine` helper can enforce open/defer-close/error mapping once; tests can assert nine handlers use it structurally or behaviorally.
- **MEDIUM — 01-06:** `TestUIProtoFieldNumbersAreContiguousAndUnreserved` conflicts with additive-only evolution. Legitimate reserved numbers make fields non-contiguous. It also cannot detect renumbering between revisions.
- **MEDIUM — 01-06:** The Files wire design must preserve both flat and tree formats. `FilesResult` is a union-like result with `Files` or `Tree` populated ([internal/query/files.go:75](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/files.go:75)); the proposed shared `FileEntry` alone is insufficient unless the proto also models `FileTreeNode`.
- **LOW — 01-06:** “Nine reads” omits `Query`, which returns full node records, while exposing `Search`. This may be intentional, but the plan should explicitly reconcile it with “every read the Engine offers.”

### Suggestions

- Introduce typed query errors before writing handler classification.
- Add a shared `withEngine` helper with an idempotent close and centralized error translation.
- Model Files’ tree response explicitly or scope the UI RPC to flat format and reject/omit `format=tree`.
- Replace contiguous-number tests with stable known-number descriptor fixtures and duplicate-number checks.

---

## Plan 01-07 — Bounds and degraded state

### Strengths

- The two-layer response-bound strategy is sound: application truncation returns useful content; Connect limits remain a backstop.
- UTF-8 boundary handling correctly follows the existing MCP discipline.
- The plan correctly recognizes that `Engine.Status` cannot run when `query.OpenAt` fails; opening happens before Engine construction ([internal/query/engine.go:160](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/engine.go:160)).
- `ErrStoreLocked` is a real exported sentinel with bounded retry already centralized in `graphstore.Open` ([internal/graphstore/pebble_store.go:92](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/pebble_store.go:92), [internal/graphstore/pebble_store.go:141](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/pebble_store.go:141)).
- No second retry layer is the right design.
- The explicit empty-source, exact-boundary, UTF-8, and transport-headroom cases are valuable.

### Concerns

- **HIGH — 01-07 and phase criterion:** The assertion that UI “still serves every RPC” while daemon/MCP holds the Pebble store long-term contradicts the actual lock semantics. `query.OpenAt` calls `graphstore.Open`, and a second open retries for roughly 400 ms then returns `ErrStoreLocked` ([internal/query/engine.go:166](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/engine.go:166), [internal/graphstore/pebble_store.go:143](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/pebble_store.go:143)). Non-Status RPCs will degrade, not serve real results. The plan quietly changes “serves every RPC” into “normal result or CodeUnavailable both pass,” which weakens the roadmap criterion.
- **HIGH — 01-07:** `TestDegradePrecedence` proposes a repo that is simultaneously uninitialized and lock-contended. Under current resolution this state is structurally impossible: `ResolveCodegraphDir` returns `ErrNotInitialized` when no `.codegraph` exists ([internal/query/resolve.go:25](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/resolve.go:25)); there is then no resolved store directory whose lock can be classified.
- **HIGH — 01-07:** `ExploreResponse` source modeling is unclear. Explore returns multiple sources keyed by group path ([internal/query/explore.go:579](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/explore.go:579)), but the plan says to attach `SourceBlob` “on ExploreResponse’s per-group source” without showing the necessary message restructuring.
- **MEDIUM — 01-07:** `bytes content` plus a UTF-8 guarantee is inconsistent at the contract level. If the content is required to be UTF-8 source text, protobuf `string` enforces and communicates that better. If arbitrary bytes are allowed, the UTF-8 guarantee should not be universal.
- **MEDIUM — 01-07:** Line-count semantics are not defined for empty input, trailing newline, or a final unterminated line. Yet the checkpoint permanently locks totals exposed to clients.
- **MEDIUM — 01-07:** Timing assertions around a 400 ms retry budget are likely flaky under CI. The store already has an event-synchronized retry seam specifically to avoid wall-clock guessing ([internal/graphstore/pebble_store.go:83](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/pebble_store.go:83)).
- **MEDIUM — 01-07:** The “real daemon/MCP holder plus trigger a re-index” exercise may not observe transient indexing if the holder already owns the store continuously. The expected state transitions need to match actual daemon ownership behavior.
- **MEDIUM — 01-07:** The fixed detail-message test banning any digit sequence resembling a PID is brittle and may reject harmless prose such as a retry duration.
- **LOW — 01-07:** Duplicating `truncateOnRuneBoundary` creates two security-relevant copies. A shared internal text utility is preferable.

### Suggestions

- Resolve the product-level contradiction explicitly:

  - Either UI RPCs degrade while another process owns the store, and success criterion 3 must say so; or
  - introduce an IPC/snapshot-sharing architecture, which current milestone context explicitly rejected.

- Remove the impossible combined uninitialized+locked test.
- Define line counting precisely before locking protobuf semantics.
- Prefer `string content` if UTF-8 is a protocol invariant.
- Use the graphstore retry test seam or injectable opener instead of wall-clock timing.
- Add `ExploreSource { path, SourceBlob source }` or embed `SourceBlob` in each `ExploreGroup`.

## Cross-plan ordering and scope

### Strengths

- Wave ordering correctly puts the golden oracle before Engine extraction.
- The RPC expansion waits for both structured Engine results and commit-aware schema.
- Bounds/degraded behavior waits until the full service exists.
- FIX-01 remains independent and does not block unrelated work.

### Concerns

- **HIGH — cross-plan:** Plan 01-04 depends only on 01-02, while 01-05 depends only on 01-01; this is fine mechanically, but 01-06 requires both. The real problem is that neither predecessor creates a viable Status SHA data path.
- **HIGH — cross-plan:** The phase declares “every RPC returns real results with daemon/MCP already holding the store,” while plan 01-07 tests accept unavailable degradation. This is a requirements mismatch, not merely a test detail.
- **MEDIUM — cross-plan:** Several plans mandate manual working-tree mutation using `git checkout --`. In a shared or dirty worktree, that can destroy overlapping user changes. The execution workflow should use `apply_patch` plus exact reverse patches, or an isolated temporary worktree.
- **MEDIUM — cross-plan:** Estimates of 60k–105k tokens per plan and repeated manual evidence exercises indicate the plans are too large. 01-04, 01-06, and 01-07 should each be split.
- **LOW — cross-plan:** Both blocking checkpoints merely reconfirm decisions already explicitly locked in CONTEXT. If human confirmation is still desired because of protobuf permanence, that is defensible, but it reduces autonomy and should be called out at roadmap level.

## Recommended changes before execution

1. Fix `uiserver` lifecycle so binding precedes URL printing and browser launch.
2. Redesign `NodeDetail` multi-definition mode to carry per-candidate gathered details.
3. Export usable Explore result component types.
4. Create a real commit-SHA path from Meta through Engine to the UI without changing CLI serialization.
5. Introduce typed query errors for Connect code mapping.
6. Decide whether locked-store non-Status RPCs degrade or must return real data; update requirements and tests consistently.
7. Replace impossible uninitialized+locked precedence and flaky timing tests.
8. Generate protobufs into a temporary directory before comparison.
9. Define SourceBlob encoding and line-count semantics before locking field numbers.
10. Replace destructive mutation/revert instructions with isolated or exact reversible edits.

## Risk Assessment

**Overall risk: HIGH.**

The plans show unusually strong attention to non-vacuity, security boundaries, correction-block compliance, and regression evidence. But the current design contains multiple execution-blocking contradictions:

- the URL is requested before the listener exists;
- multi-definition data does not fit the proposed structured type;
- commit SHA cannot reach the Status RPC through the planned types;
- typed error mapping lacks typed source errors;
- locked-store behavior does not satisfy the stated success criterion;
- at least one degraded-state test describes an impossible state.

These are architectural issues, not polish. Once corrected, the phase should drop to **MEDIUM risk**, driven mainly by the Engine extraction breadth, new generated wire contract, Pebble contention semantics, and the amount of manual evidence required.

---

## Consensus Summary

One reviewer lane ran (`codex`, explicitly selected via `--codex`), so there is no
multi-reviewer consensus to compute. In its place, the orchestrator ran an independent
**source-grounding adjudication pass** over every load-bearing claim in the Codex review —
resolving each cited symbol by reading the Go source rather than by trusting either the
plan text or the reviewer. The `drift-guard authority` on this repo resolves to `intel`,
but `.planning/intel/API-SURFACE.md` reports `symbolCount: 0` on this Go repo (its
extractor is regex/JS-only), so grading under that authority would have marked every Go
symbol UNCHECKABLE and hard-blocked nothing. The full coverage ledger is at the end of
this file.

Codex's review is genuinely source-grounded: it cites `file:line` throughout and its
citations check out. It is **not** carrying the `[reviewed-without-repo-access]` or
`[reviewed-without-source-citations]` markers, so its verdict counts at full weight.

**Correction-block compliance — both honored.** The reviewer did not misfire on either
correction block, and independent verification agrees with the plans:

- **D-04 correction.** `TestReFrozenGoldensValid` (`testdata/golden/golden_test.go:243`)
  asserts only file existence, non-empty bytes, a leading `{`, successful `goldenCapture`
  unmarshal, and a non-empty `Output` — it never constructs an `Engine` and never compares
  live output. Verified. Plan 01-02 builds the missing byte-identity oracle in Wave 1 with
  `depends_on: []`, and plan 01-04 carries `depends_on: ["01-02"]`, so the oracle
  provably precedes the extraction. The 26-pair count is correct: 24 under
  `testdata/golden/corpus/<slug>/` (4 corpora × 6 files) plus 2 under
  `corpus/behavioral/`. **The correction is honored.**
- **Folded-todo correction.** `github.com/modelcontextprotocol/go-sdk@v1.7.0`
  (`go.mod:19`) calls `jsonrpc2.Async(ctx)` at `mcp/server.go:1914`, guarded at `:1913` by
  `if req.IsCall() && req.Method != methodInitialize`. Verified in the populated module
  cache. Plan 01-03 keeps FIX-01 scoped to `pendingWriter`, adds an explicit separability
  disproof, names the wire oracle as the correct locus for the flake, and requires the todo
  to remain in `pending/` (`01-03-PLAN.md:207`, `:219`). **The correction is honored.**

**New-artifact exclusions applied.** `internal/uiserver/*`, `internal/uiproto/uiv1/*`,
`internal/cli/ui.go`, `internal/query/detail.go`, `internal/query.NodeDetail`/
`ExploreResult`/`NodeDetailMode`, `schema.Meta.commit_sha` field 8,
`schema.IndexedCommitSHA`, `internal/indexer/commit.go`, `internal/mcp/pending_writer_test.go`,
`looksLikeJSONRPCResponse`, `internal/upgrade/proto_task_test.go`, `buf.yaml`,
`buf.gen.yaml`, `go.tool-proto.mod`, `Taskfile.yml` tasks `proto:gen`/`proto:drift`, and
`testdata/golden/byte_identity_test.go` were all confirmed absent from the tree and were
**not** counted as missing references — each is listed under a plan's "Artifacts this
phase produces".

### Agreed Strengths

(Single grounded reviewer; these are Codex findings the adjudication pass independently
confirmed against source.)

- The guard wraps the whole `http.ServeMux` rather than being a Connect interceptor — the
  right boundary, and the only one that will also cover Phase 2's static routes.
- Per-call `query.OpenAt` + deferred close matches the real store contract
  (`internal/query/engine.go:160`) and the existing MCP precedent
  (`internal/mcp/tools.go:71`), which is also per call. There is no cached-handle pattern
  anywhere in the plans.
- `graphstore.ErrStoreLocked` is a real exported, `errors.Is`-able sentinel
  (`internal/graphstore/pebble_store.go:110`) with existing consumers at
  `internal/daemon/daemon.go:297` and `internal/cli/serve.go:241`. The RPC layer being a
  third consumer of the same seam — rather than a new mechanism — is correct.
- `Meta` field 8 is genuinely free: `internal/schema/graph.proto:137` occupies 1–7 and the
  `Meta` message has no `reserved` clause at all. D-05's premise is sound.
- Delegating bound/limit enforcement to the existing `validateLimit`/`validateDepth`/
  `clampDepth`/`clampAffectedDepth`/`MaxLimit` in `internal/query/validate.go` — rather
  than a second RPC-layer copy — is the right call and is what the code actually does.
- The non-vacuity discipline throughout (count `--- PASS` lines, never exit status;
  demonstrate RED before trusting a gate) is applied consistently and is the repo's own
  established rule.

### Agreed Concerns

The nine HIGH findings below survived source verification and are the priority set. Each
was re-checked against the tree, not taken on the reviewer's word.

1. **01-01 — the URL is printed before the listener exists.** `(*Server).Run(ctx)` is
   specified to do `net.Listen` and extract the port from `ln.Addr().(*net.TCPAddr).Port`
   (`01-01-PLAN.md:283-287`), while Task 3 says the CLI "prints the server's URL to
   `cmd.OutOrStdout()` before serving" and then calls `srv.Run(ctx)`
   (`01-01-PLAN.md:325-330`). With `DefaultAddr = 127.0.0.1:0` there is no port to print at
   that moment. The browser launch inherits the same defect. **Confirmed — execution-blocking.**
   The fix is a `Listen`/`Serve` split (or binding in `New`).
2. **01-04 — `NodeDetail` cannot represent the multi-definition shape, and the boundary is
   not where the plan assumes.** Verified stronger than Codex stated:
   `RenderNodeMultiDef` (`internal/query/render_markdown.go:207`) takes a `fetch` **callback**
   and invokes it *lazily*, only for matches below `nodeMultiDefHardCap`
   (`render_markdown.go:211-213`), with the body-budget decision (`used+len(section) <=
   nodeMultiDefBodyBudget`, `:221`) interleaved into the same loop. So (a) the proposed
   `NodeDetail{Mode, Node, Calls, CalledBy, Matches, Source}` has one slot where per-candidate
   data is needed, and (b) an eager pre-render gather over *all* matches does I/O the current
   code never does for capped-out matches and can surface a `readSourceFile` error where today
   there is none — a behavior change the 26 goldens would very likely not catch. **Confirmed
   and escalated.**
3. **01-05 / 01-06 — the commit SHA has no data path to the Status RPC.** `01-06-PLAN.md:146`
   sources `GetStatusResponse.commit_sha` from `schema.IndexedCommitSHA`, which takes a
   `*schema.Meta`. The handler holds only a `query.StatusResult`, which has no commit field
   (`internal/query/status.go:47-65`). `Engine.reader` is unexported
   (`internal/query/engine.go:39`), `Engine` exposes no `Meta` or `Reader` accessor (verified
   across the full exported method set), `graphstore.Reader.GetMeta()`
   (`internal/graphstore/store.go:49`) is unreachable from `internal/uiserver`, and a second
   in-process `graphstore.Open` on the same directory is itself lock-refused. Meanwhile
   `01-05-PLAN.md:228` explicitly forbids touching `status.go`. **Confirmed — 01-06's
   must_have "the `Status` response carries the commit SHA" is unachievable as planned.**
4. **01-06 — typed Connect error mapping has no typed source errors to map from.** Node
   not-found is `fmt.Errorf("query: symbol %q not found", symbol)` at
   `internal/query/node.go:341`; the argument error at `:318` is likewise a formatted string.
   The plan forbids string matching but schedules no typed query errors. **Confirmed.**
5. **01-07 — `TestDegradePrecedence` describes a structurally impossible state.**
   `ResolveCodegraphDir` returns `ErrNotInitialized` only after walking to the filesystem
   root without finding a `.codegraph/` directory (`internal/query/resolve.go:31-47`). If no
   `.codegraph/` exists there is no store directory whose lock could be contended, so
   "simultaneously uninitialized and lock-contended" cannot be constructed. **Confirmed.**
6. **01-03 — the outbound response sniff must classify only the bytes actually written.**
   `io.Writer.Write` may return `n < len(b)` with an error; classifying all of `b` decrements
   for a response that never reached stdout, breaking the authoritative-wire invariant the
   file's own doc comment states. **Confirmed as a correctness requirement.**
7. **01-03 — "buffer partial lines" must not delay forwarding.** `pendingWriter` is the
   protocol writer (`internal/mcp/server.go:334`, wired at `:229`); it must write through
   immediately and buffer only a *copy* of `b[:n]` for classification. **Confirmed.**
8. **01-04 — `ExploreResult` exports fields whose element types are unexported.** The real
   types are `exploreFileGroup` (`internal/query/explore.go:17`) and `exploreBlast`
   (`:25`). Their *fields* are exported, so `internal/uiserver` can read values — this
   compiles — but it cannot name the types, declare a mapper signature over them, or
   construct them in tests. As an exported seam that Phases 3–6 must consume, it is the
   wrong shape. **Confirmed (legal Go, poor contract).**
9. **01-07 — `ExploreResponse`'s per-group `SourceBlob` modeling is unspecified.** Explore
   carries sources as a `map[string][]byte` keyed by path (`internal/query/explore.go:579`,
   threaded into `RenderExplore`'s `sources` parameter at `render_markdown.go:360`). Attaching
   a `SourceBlob` "on `ExploreResponse`'s per-group source" requires message restructuring the
   plan does not show, and the field numbers are locked by a one-way checkpoint in the same
   plan. **Confirmed.**

### Divergent Views

With one lane there is no reviewer-vs-reviewer divergence, but the adjudication pass
**disagrees with Codex on one HIGH** and **adds one finding Codex missed**:

- **DOWNGRADED — Codex's "locked-store RPCs cannot serve real results" HIGH (01-07 and
  phase criterion 3).** Codex read roadmap criterion 3 as self-contradictory. It is not.
  Verified: nothing in this repo holds the Pebble store long-term. `internal/mcp/tools.go:71`
  opens per tool call; `internal/daemon/daemon.go` opens only inside `flush` and closes
  (`:272`); `serve --mcp` is stdio-only with no persistent store handle. `graphstore.Open`
  retries 5 × 100 ms (`internal/graphstore/pebble_store.go:78-80`), which rides out exactly
  the transient collisions those callers produce — and criterion 3's *own second clause*
  already routes the long case ("a re-index that outlasts `graphstore.Open`'s retry budget
  renders as 'indexing in progress'"). The two clauses partition the space rather than
  contradict. **Downgraded to MEDIUM**, and re-aimed at the real defect: plan 01-07's
  `TestRPCsCoexistWithAConcurrentStoreHolder` accepts "each succeed **or** degrade
  independently" (`01-07-PLAN.md:280`), an assertion with no failing input — vacuous under
  this repo's own rule `84d1gfpywd`, which every other gate in the phase respects.
- **ADDED (not raised by Codex) — the generated-header version stamp will produce a
  guaranteed first-run drift diff.** `internal/schema/graph.pb.go:11-15` records
  `protoc-gen-go v1.36.11` / `protoc v7.35.1`, and there is no codegen invocation checked in
  anywhere today: no `go:generate` directive, no Makefile, no proto Taskfile target, no buf
  config, no CI step — regeneration is currently manual and out-of-band. `buf generate` does
  not shell out to `protoc` and stamps its own compiler identity, so the first regeneration
  of the pre-existing surface will rewrite that header even with byte-identical message code.
  Plan 01-05 anticipates "latent drift" generically (`01-05-PLAN.md` objective) but names no
  expected header change and gives no ruling on whether a header-only rewrite satisfies its
  "produces no diff" must_have. Compounding it: `01-01-PLAN.md:245-246` has `proto:gen`
  regenerate **both** surfaces, while its acceptance criterion checks only
  `git status --porcelain internal/uiproto` (`:304`, `:416`) — so 01-01 can leave
  `internal/schema/graph.pb.go` dirty with nothing asserting on it. **MEDIUM, actionable in
  both 01-01 and 01-05.**
- **ADDED (minor) — `gather*` is already taken in `internal/query`.** The package declares
  `gatherChannel1`/`gatherChannel2`/`gatherChannel3`, `gatherMerge` and
  `applyPostMergeRerankers` in `internal/query/gather.go`, where "gather" means retrieval-channel
  merge. Plan 01-04 introduces `gatherNode`/`gatherFileNode`/`gatherSingleDefNode`/
  `gatherMultiDefNode`/`gatherExplore` into the same package with a different meaning. **LOW**,
  but worth one naming decision before the extraction lands.

### Top three concerns to act on first

1. The `uiserver` bind/URL/browser lifecycle (HIGH, 01-01 — blocks the tracer).
2. The `NodeDetail` multi-definition model and the lazy-`fetch` render boundary (HIGH,
   01-04 — blocks ENG-01 and everything in Waves 3–4).
3. The missing `Meta.commit_sha` → Status data path (HIGH, 01-05/01-06 — blocks ENG-04's
   only wire consumer).

---

## Verification coverage

Mandatory ledger for the source-grounding pass. **Authority note:** `drift-guard authority`
resolves to `intel` on this repo, but `.planning/intel/API-SURFACE.md` reports
`symbolCount: 0` because the intel extractor is regex/JS-only and does not parse Go. Under
that authority every Go symbol would grade UNCHECKABLE → INFO and nothing would hard-block.
**The intel map was therefore not consulted as evidence.** Every symbol below was resolved
by reading source (`rg` / `sed -n`, with `codegraph explore` where faster). The file itself
states "Treat absence here as 'unknown', not 'does not exist'."

**False-drift traps handled.** (a) `.proto` snake_case vs generated Go camelCase
(`commit_sha` vs `CommitSha`/`GetCommitSha`) was treated as expected, not drift.
(b) `rg` exits non-zero for both "no match" and "file not found", so **every zero result
below was positive-controlled** — each ABSENT verdict was produced by a search run in the
same invocation as a pattern that matched, or preceded by `rg -c . <file>` proving the file
readable.

### Resolved FOUND (checked against source)

`query.Engine` (`internal/query/engine.go:38`) · `(*Engine).Node` (`node.go:316`) ·
`(*Engine).Explore` (`explore.go:240`) · `(*Engine).Status` (`status.go:247`) ·
`Search` (`search.go:153`) · `Query` (`search.go:120`) · `Files` (`files.go:136`) ·
`Callers` (`traverse.go:351`) · `Callees` (`traverse.go:290`) · `Impact` (`traverse.go:428`) ·
`Affected` (`traverse.go:544`) · `query.OpenAt` (`engine.go:160`) · `StatusResult`
(`status.go:47`, full field set enumerated) · `fetchCalls` (`node.go:363`) ·
`fetchCalledBy` (`node.go:399`) · `RenderNode` (`render_markdown.go:127`) ·
`RenderNodeMultiDef` (`render_markdown.go:207`) · `RenderExplore` (`render_markdown.go:360`) ·
`renderSingleDefNode` (`node.go:420`) · `renderMultiDefNode` (`node.go:442`) ·
`BuildReverseAdjacency` (`traverse.go:32`) · `validateLimit` (`validate.go:107`) ·
`validateDepth` (`validate.go:137`) · `clampDepth` (`validate.go:71`) ·
`clampAffectedDepth` (`validate.go:86`) · `MaxLimit` (`validate.go:26`) ·
`resolveSourcePath` (`node.go:33`) · `readSourceFile` (`node.go:84`) ·
`FileEntry` (`files.go:57`) · `Location` (`search.go:17`) ·
`ResolveCodegraphDir` (`resolve.go:31`) · `ErrNotInitialized` (`resolve.go:18`) ·
`exploreFileGroup` (`explore.go:17`) · `exploreBlast` (`explore.go:25`) ·
`schema.Meta` (`internal/schema/graph.proto:137`, fields 1–7, no `reserved`) ·
`schema.NewMeta` (`internal/schema/meta.go:17`) ·
`Writer.PutMeta` (`internal/graphstore/store.go:188`, impl `batch.go:105`) ·
`Reader.GetMeta` (`store.go:49`) · `graphstore.Open` (`pebble_store.go:141`) ·
`graphstore.Reader` (`store.go:38`) · `GraphStore` (`store.go:17`) ·
`ErrStoreLocked` (`pebble_store.go:110`) · `openLockRetryAttempts`/`openLockRetryBackoff`
(`pebble_store.go:78-80`) · `openLockRetrySleep` test seam (`pebble_store.go:~90`) ·
`BuildServer` (`internal/mcp/server.go:485`) · `ServeStdio` (`server.go:170`, impl `:225`) ·
`pendingWriter` (`server.go:334`) + `.Write` (`:339`) + `.Close` (`:349`) ·
`(*stdinLingerReader).waitForDrain` (`server.go:298`) · `stdinLingerReader` (`server.go:258`) ·
`stdinLingerGrace` (`server.go:241`) · `sniffedMessage` (`server.go:309`) ·
`looksLikeJSONRPCCall` (`server.go:323`) · `root.AddCommand` (`internal/cli/root.go:52-58`,
24 commands) · `newServeCmd` (`serve.go:172`) · `newDaemonCmd` (`daemon.go:51`) ·
`TestReFrozenGoldensValid` (`testdata/golden/golden_test.go:243`) ·
`expectedGoCaptures` (`golden_test.go:169-217`) ·
`ExpectedGoldenScenarioCount` (`golden_test.go:233`) ·
`lockedCorpusArgs` (`testdata/golden/gocapture/main.go:70`) ·
`buildEngineAt` (`testdata/golden/behavioral_test.go:225`) ·
`TestWorkflowRunBodiesInvokeTask` (`internal/upgrade/taskfile_shape_test.go:1345`) ·
`TestGateStancesStated` (`taskfile_shape_test.go:805`) ·
`TestToolsListOrderIsDeterministic` (`test/wireoracle/oracle_test.go:611`) ·
`go.tool.mod` (3 pinned tools) · `Taskfile.yml` (45 tasks enumerated) ·
`github.com/modelcontextprotocol/go-sdk v1.7.0` (`go.mod:19`) ·
upstream `jsonrpc2.Async` (`go-sdk@v1.7.0/internal/jsonrpc2/conn.go:369`) and its guarded
call site (`go-sdk@v1.7.0/mcp/server.go:1913-1914`).

### Resolved ABSENT — expected, listed under a plan's "Artifacts this phase produces" (NOT flagged)

`internal/uiserver/*` (whole package: `Options`, `DefaultAddr`, `Server`, `New`, `Run`,
`URL`, `originHostGuard`, `allowedHosts`, `allowedOrigins`, `uiService`, `truncateSource`,
`sourceLineCap`, `sourceByteCap`, `truncateOnRuneBoundary`, `transportSendMaxBytes`,
`transportReadMaxBytes`, `degradedStatus`, `errIndexingInProgress`) ·
`internal/uiproto/uiv1/*` (whole package incl. `ui.proto`, `ui.pb.go`, `uiv1connect/ui.connect.go`,
`UIService`, all nine rpcs, all request/response messages, `SourceBlob`, `IndexingInProgress`,
`NodeDetailMode`, `ExploreGroup`, `BlastEntry` as proto messages) ·
`internal/cli/ui.go`, `newUiCmd`, `shouldOpenBrowser` · `internal/query/detail.go`,
`NodeDetail`, `NodeDetailMode`, `NodeDetailModeFile`, `ExploreResult`, `gatherNode`,
`gatherFileNode`, `gatherSingleDefNode`, `gatherMultiDefNode`, `gatherExplore` ·
`schema.Meta.commit_sha` field 8, `Meta.CommitSha`, `GetCommitSha`, `schema.IndexedCommitSHA` ·
`internal/indexer/commit.go`, `commit_test.go`, `resolveHeadCommitSHA` ·
`internal/schema/meta_commit_test.go` · `internal/mcp/pending_writer_test.go`,
`looksLikeJSONRPCResponse`, `sniffedOutbound` · `testdata/golden/byte_identity_test.go` ·
`internal/upgrade/proto_task_test.go` · `buf.yaml`, `buf.gen.yaml`, `go.tool-proto.mod` ·
`Taskfile.yml` tasks `proto:gen` / `proto:drift` · every new `Test*` name declared by a plan.

### Resolved ABSENT — load-bearing, and the reason it matters

| Symbol | Where cited | Why it matters |
|---|---|---|
| `ExploreGroup` / `BlastEntry` / `SourceBlob` as **`internal/query` Go types** | 01-04 `ExploreResult` field types (unspecified) | Real types are unexported `exploreFileGroup` / `exploreBlast`; sources are `map[string][]byte`. Drives HIGH #8 and HIGH #9. |
| No `Engine` accessor for `*schema.Meta` or `graphstore.Reader` | 01-06 `statusToProto` ← `schema.IndexedCommitSHA` | Drives HIGH #3 — no data path exists. |
| No exported error sentinel for node-not-found | 01-06 "classify by Engine sentinels" | Drives HIGH #4 — `node.go:341` is `fmt.Errorf`. |
| Any proto codegen invocation (`go:generate`, Makefile, Taskfile target, buf config, CI step) | 01-01/01-05 `proto:gen` / `proto:drift` | Confirms BLD-04 closes a genuine pre-existing gap; also drives the ADDED header-stamp MEDIUM. |
| `connectrpc.com/connect`, `github.com/pkg/browser` in `go.mod` | 01-01 new deps | Confirmed absent from `go.mod` (positive-controlled). Note `pkg/browser` *is* a stale `go.sum` entry and a `go.tool.mod` indirect — expected, not a conflict. |

### UNCHECKABLE / skipped — and why

| Symbol | Verdict | Reason |
|---|---|---|
| `connect.WithSendMaxBytes`, `connect.WithReadMaxBytes` | **UNCHECKABLE** | These belong to `connectrpc.com/connect`, which this phase *adds*. The module is in neither `go.mod`, `go.sum`, nor the module cache, so the API surface, its return code (`CodeResourceExhausted`), and its reject-not-truncate semantics could **not** be verified locally. Plan 01-07's key_link rests on all three. Verify at execution time, immediately after 01-01 adds the dependency. |
| `connect.CodeUnavailable`, `CodeInternal`, `CodeResourceExhausted` | **UNCHECKABLE** | Same reason — dependency not yet present. |
| `uiv1connect.NewUIServiceHandler` / `NewUIServiceClient` / `UIServiceHandler` | **SKIPPED (new)** | Generated by a codegen pipeline this phase creates; cannot exist before `buf generate` runs. |
| `buf` v1.72.0 / `protoc-gen-connect-go` v1.20.0 / `protoc-gen-go` v1.36.x MVS compatibility | **UNCHECKABLE** | Requires actually running `go mod tidy -modfile=<scratch>` and building each tool — an execution-time measurement, which is exactly what 01-01 Task 2 schedules. The version-bid-loss precedent is documented in `go.tool.mod`'s header and is real; the outcome for *these* tools is not knowable from the tree. |
| Whether `buf generate` reproduces `graph.pb.go` byte-identically apart from the header | **UNCHECKABLE** | No buf toolchain present. This is the basis of the ADDED MEDIUM and must be measured at execution, not assumed. |
| `.planning/intel/API-SURFACE.md` | **NOT CONSULTED** | `symbolCount: 0` on a Go repo; extractor is regex/JS-only. Absence there is "unknown", never "does not exist". Consulting it would have graded every symbol above UNCHECKABLE and hard-blocked nothing. |

**A clean review here does not mean "nothing was checked."** 66 symbols resolved FOUND
against source with `file:line`; 5 resolved ABSENT-and-load-bearing (each driving a HIGH);
the new-artifact set confirmed absent and deliberately excluded; 6 items recorded
UNCHECKABLE, every one of them blocked on the `connectrpc.com/connect` and `buf` toolchains
that this phase itself introduces.

---

# Cross-AI Plan Review — Phase 1 (cycle 2, post-replan)

Everything above this line is cycle 1, reviewed against the 7-plan / 4-wave shape.
The plans were replanned in response (commit `d1695a2`) and are now 11 plans / 7 waves.
This file is APPEND-style history: a cycle-1 finding is closed only if the section below
says so.

## Codex Review

# Cycle 2 Plan Review

## Summary

The replan is materially stronger and resolves all nine cycle-1 HIGH findings. The dependency graph is coherent, same-wave file ownership does not overlap, both correction blocks are honored, probe accounting remains exactly 21, and all four requested rejection decisions are recorded in the relevant plans with rationale.

Three newly introduced issues remain. Most importantly, plan 01-09 tries to descriptor-pin source fields that are only comments until plan 01-10 adds them; that guard cannot pass as specified. In addition, most automated test gates discard `go test`’s exit status and can pass despite failed tests, and the Origin/Host guard does not require the admitted Origin to correspond to the admitted Host. Overall risk is **MEDIUM-HIGH** until those are corrected.

## Strengths

- **01-01:** The `Listen`/`Serve` split fixes the ephemeral-port lifecycle problem. `Listen` binds before URL publication, while the CLI prints and optionally opens the bound URL before entering `Serve` ([01-01-PLAN.md:327](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-01-PLAN.md:327), [01-01-PLAN.md:349](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-01-PLAN.md:349)).

- **01-02:** The D-04 correction is fully honored. The plan adds a live-output oracle before either extraction, whereas the current test only checks the frozen envelope and never invokes the Engine ([golden_test.go:243](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/golden/golden_test.go:243), [golden_test.go:275](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/golden/golden_test.go:275)). It also correctly treats the existing behavioral suite as complementary property coverage, consistent with its source comment ([behavioral_test.go:684](/Volumes/Code/github.com/seanb4t/codegraph-go/testdata/golden/behavioral_test.go:684)).

- **01-03:** The pending-writer redesign now reflects the actual defect. The current implementation decrements on every write ([server.go:331](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:331), [server.go:339](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:339)), while only inbound calls increment ([server.go:271](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:271)). The replan forwards first and classifies only `b[:n]`, resolving both cycle-1 transport findings.

- **01-04:** The lazy `MultiDefDetail.Definition` protocol matches the real rendering boundary. The current renderer invokes its callback only before the hard cap and makes budget decisions inside that loop ([render_markdown.go:206](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/render_markdown.go:206), [render_markdown.go:210](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/render_markdown.go:210)). The proposed sum type therefore preserves behavior that an eager structure would change.

- **01-05:** Exporting `ExploreFileGroup` and `ExploreBlast`, funneling every zero-result branch into `ExploreResult`, and adding message-preserving error classes together form a usable Engine seam. Keeping `Error()` byte-identical while implementing `errors.Is` is the correct way to preserve shipped CLI behavior.

- **01-06:** `(*Engine).IndexMeta()` closes the previously missing data path without altering `StatusResult`. Its placement in `engine.go` rather than `status.go`, combined with the reflected field-set assertion, supports D-06 cleanly.

- **01-07:** The temp-tree drift guard is safer than in-place generation. It covers both generated surfaces, asserts a positive compared count, names mismatched files, and explicitly treats header-only changes as drift. That is coherent with the current generated header stamp ([graph.pb.go:11](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/schema/graph.pb.go:11)) and 01-01 now checks both `internal/uiproto` and `internal/schema`.

- **01-08:** The seven structured RPC mappings reuse one Engine lifecycle seam and delegate existing depth/limit rules to the Engine instead of duplicating them.

- **01-09:** The wire model now preserves per-candidate node details and per-group Explore source association. The explicit nine-method set guard is much stronger than a mutating-verb blacklist.

- **01-10:** Application truncation and Connect transport rejection are correctly treated as different layers. Using `bytes` with a scoped UTF-8 guarantee avoids protobuf string-marshal failures for non-UTF-8 source.

- **01-11:** The impossible filesystem precedence fixture is gone. Testing `classifyDegrade` directly with a multi-target error is appropriate, and transient versus sustained lock contention now has determinate expected outcomes.

- **Ordering:** There is no same-wave `files_modified` overlap. The byte oracle precedes the Node extraction, the Node extraction precedes Explore, and all complete view RPC work follows both seams. The roadmap’s seven-wave sequence reflects this ([ROADMAP.md:137](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/ROADMAP.md:137), [ROADMAP.md:148](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/ROADMAP.md:148), [ROADMAP.md:153](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/ROADMAP.md:153)).

- **Probe accounting:** The ledger still balances: 14 explicit edge truths, one backstop, and six flagged assumptions = 21. Only one actual `verification: backstop` marker exists, in the required `{statement, verification}` mapping form ([01-01-PLAN.md:46](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-01-PLAN.md:46)). The apparent second marker is prose discussing that same backstop, not another probe disposition.

- **Recorded rejections:** All four are present with rationale:

  - Locked-store real-results concern: [01-11-PLAN.md:394](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-11-PLAN.md:394)
  - JSON-RPC batch case: [01-03-PLAN.md:288](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-03-PLAN.md:288)
  - Checkpoint distinction: [01-06-PLAN.md:360](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-06-PLAN.md:360), [01-10-PLAN.md:370](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-10-PLAN.md:370)
  - Per-candidate partial failure deferral: [01-09-PLAN.md:174](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-09-PLAN.md:174)

## Concerns

- **HIGH — 01-09/01-10: the field-number guard cannot work one wave before the fields exist.**  
  Plan 01-09 says it only leaves comments reserving future source numbers ([01-09-PLAN.md:145](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-09-PLAN.md:145), [01-09-PLAN.md:190](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-09-PLAN.md:190)), but its descriptor-based known-number fixture must already include those fields ([01-09-PLAN.md:334](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-09-PLAN.md:334)). Protobuf descriptors contain declared fields and `reserved` ranges; they do not contain field-number allocations mentioned only in comments. Therefore:

  - If the fixture requires `GetNodeDetailResponse.source`, `NodeDefinition.source`, and `ExploreGroup.source`, 01-09 fails because those fields do not exist.
  - If the test skips absent fixture entries, the claimed protection is vacuous.
  - A proto `reserved` declaration would prevent 01-10 from using the numbers.

  The cycle-1 modeling concern is fixed, but this attempted one-wave separation introduces a new execution blocker.

- **HIGH — all plans: automated PASS-count gates discard failing test status.**  
  Typical verification captures piped output and then checks only a lower-bound PASS count, for example [01-01-PLAN.md:201](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-01-PLAN.md:201), [01-03-PLAN.md:202](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-03-PLAN.md:202), and [01-11-PLAN.md:371](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-11-PLAN.md:371). Without `set -o pipefail` or separately checking the captured exit status, a run with one failing test and enough passing subtests satisfies `[ "$COUNT" -ge N ]`.

  This recreates the vacuity class the plans are trying to eliminate. Counting protects against “matched no tests”; it is not a substitute for requiring the suite itself to exit zero.

- **MEDIUM — 01-01: admitted Host and Origin are validated independently, not as a pair.**  
  The plan builds separate three-member allowlists and accepts any admitted Origin with any admitted Host ([01-01-PLAN.md:179](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-01-PLAN.md:179), [01-01-PLAN.md:207](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-01-PLAN.md:207)). Thus `Host: 127.0.0.1:P` with `Origin: http://localhost:P` is admitted. That conflicts with the phase criterion saying the three spellings are admitted by exact match rather than treated as interchangeable ([ROADMAP.md:128](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/ROADMAP.md:128)). Require `Origin`’s authority to equal the admitted `Host` exactly, when Origin is present.

- **MEDIUM — 01-03: locking only the classification buffer does not make concurrent `Write` safe.**  
  The plan forwards via `p.w.Write(b)` before taking the classification-buffer mutex ([01-03-PLAN.md:168](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-03-PLAN.md:168), [01-03-PLAN.md:181](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-03-PLAN.md:181)). `io.Writer` does not promise concurrent-call safety. Concurrent underlying writes may interleave or return in an order different from classification-buffer acquisition, causing the copied line stream not to match wire order.

  Use one mutex around both the underlying write and classification of `b[:n]`. “Forward first” should mean write before interpreting bytes within the serialized critical section, not write outside synchronization.

- **MEDIUM — 01-01: `Serve` cancellation mechanics are underspecified.**  
  The plan says `Serve(ctx)` calls blocking `http.Server.Serve` and “on `ctx.Done()`” calls `Shutdown` ([01-01-PLAN.md:349](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-01-PLAN.md:349)). That requires a goroutine/select or a shutdown goroutine; a literal sequential implementation never reaches the context branch until serving has already stopped. The cancellation test should catch this, but the implementation instruction should prescribe the concurrency shape and error-channel handling.

- **MEDIUM — 01-11: “release comfortably inside the budget” remains timing-sensitive.**  
  The plan correctly avoids elapsed-time assertions, but scheduling release “comfortably inside” the roughly bounded retry period is still wall-clock coordination ([01-11-PLAN.md:247](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-11-PLAN.md:247), [01-11-PLAN.md:251](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-11-PLAN.md:251)). Under loaded race CI this can cross the boundary spuriously. The exact retry seam lives in `graphstore`, so either expose a test-only coordination seam to `uiserver` tests or keep only one integration smoke case and delegate deterministic boundary coverage entirely to `graphstore`.

- **LOW — 01-06: SHA validation assumes SHA-1 object format.**  
  Requiring exactly 40 lowercase hex characters excludes Git SHA-256 repositories. This may be acceptable for v1, but it should be stated as a scoped limitation or accept both 40- and 64-character object IDs.

## Suggestions

- Move the three `SourceBlob` attachment fields themselves into plan 01-09 using a placeholder message already declared there, or move the known-number descriptor guard to 01-10 after the fields exist. Do not attempt to reserve future usable numbers with comments or protobuf `reserved`.

- Change every automated Go-test gate to require both conditions:

  1. the test command exits zero; and
  2. the expected positive/exact execution count is observed.

  Capture output to a temporary file, save the command status, count PASS lines from the file, then assert both.

- Make Origin validation pairwise: after Host succeeds, require a present Origin to equal `"http://"+r.Host` exactly. Add negative cross-spelling cases in both directions.

- Serialize `pendingWriter.Write` across the underlying write and subsequent `b[:n]` classification. Keep forwarding before classification inside the lock.

- Specify `Serve(ctx)` as a goroutine/select lifecycle with a buffered serve-error channel, bounded shutdown, and normalization of `http.ErrServerClosed`.

- Replace timing-sensitive lock-release coordination with an event-controlled retry seam where practical.

## Risk Assessment

**Overall risk: MEDIUM-HIGH.**

The architecture is now sound and the cycle-1 HIGH issues are genuinely addressed. The remaining risk comes from two plan-level execution/verification defects: the impossible cross-wave descriptor guard and automated commands that can report green despite failed tests. Once those are fixed, risk should drop to **MEDIUM**, driven mainly by the breadth of the Engine extraction, the permanent protobuf surface, lock-contention integration behavior, and the amount of manual mutation evidence required.


---

## Consensus Summary

Single grounded reviewer this cycle (Codex, source-grounded, `file:line`-cited), independently
re-verified by the orchestrator against the tree. Where the two agree the finding is recorded as
confirmed; nothing below rests on the reviewer's word alone.

### Cycle-1 HIGHs: all nine verified CLOSED

| Cycle-1 HIGH | Verified closed by |
|---|---|
| URL printed before the listener binds | `Listen(Options) (*Server, error)` binds, extracts the concrete port, mounts the guard and returns before `Serve` blocks (01-01-PLAN.md:340-356); `TestListenPublishesABoundPortBeforeServe` dials the port with no `Serve` call anywhere before the assertion |
| `NodeDetail` cannot represent the multi-definition shape | Sum type `Mode`/`File`/`Definition`/`Multi` (01-04-PLAN.md:90-95) with `MultiDefDetail.Definition(n)` lazy per candidate; `TestNodeDetailMultiDefDoesNotReadSourceForUnrenderedCandidates` asserts BOTH that an unreadable out-of-cap candidate does not fail `Node`/`NodeDetail` AND that asking for it explicitly returns the read error |
| No data path from `Meta.commit_sha` to the Status RPC | `(*Engine).IndexMeta` added in `engine.go`, not `status.go`; D-06 held by a reflected `StatusResult` field-set assertion plus a recorded `status --json` byte capture (01-06-PLAN.md:353) |
| Error classification by string match | `query.ErrNotFound` / `query.ErrInvalidArgument` with message bytes held byte-identical (01-05-PLAN.md:35); site count measured and asserted |
| `TestDegradePrecedence` fixture was unreachable by construction | Removed; precedence now driven at `classifyDegrade` with a synthetic multi-target error, and the unreachability recorded as a key_link (01-11-PLAN.md:30) |
| Outbound sniff classified `b` not `b[:n]` | 01-03-PLAN.md:24 must_have + `TestPendingWriterClassifiesOnlyBytesActuallyWritten` |
| Classification delayed forwarding | Forward-first is step 1; buffer is an explicit classification-only copy (01-03-PLAN.md:25) |
| `exploreFileGroup` / `exploreBlast` unnameable from `internal/uiserver` | Exported as `ExploreFileGroup` / `ExploreBlast` with a positive-controlled zero-gate on the old names (01-05-PLAN.md:211) |
| Field numbers locked by a checkpoint in the same plan that invents them | Moved one wave earlier to 01-09 — **but see HIGH C2-1: the move as specified cannot be executed** |

### Probe accounting — balances at 21, no silent drops

15 edge-tagged truths authored into `must_haves` (SRV-02 ×1, RPC-01 ×1, ENG-01 ×3, ENG-02 ×3,
SRV-04 ×5, RPC-05 ×2) + 6 flagged assumptions (SRV-01, SRV-03, RPC-02 in 01-01; FIX-01 in 01-03;
ENG-04 in 01-06; BLD-04 in 01-07) = **21**. Exactly one `verification: backstop` marker exists, in
the required flat-scalar `{ statement, verification }` mapping form (01-01-PLAN.md:46-47). The
apparent second marker is prose at 01-01-PLAN.md:490 discussing that same backstop, and the
`backstop` hits in 01-10 are about the Connect transport backstop — a different concept, not a
probe disposition.

### Recorded rejections — all four present with rationale

| Rejection | Location |
|---|---|
| "locked-store RPCs cannot serve real results" | 01-11-PLAN.md:394 — Rejected per adjudication, with the partition argument |
| JSON-RPC batch-payload test case | 01-03-PLAN.md:288 — Rejected with the newline-delimited-transport rationale |
| "both checkpoints only reconfirm CONTEXT" | Accepted for 01-06 (01-06-PLAN.md:360, kept with the cost recorded); Rejected for 01-10 (01-10-PLAN.md:370, five new decisions named) |
| per-candidate partial failure on `GetNodeDetail` | 01-09-PLAN.md:170-177 — deferred in the action body, with SUMMARY recording required |

### Generated-header ruling — coherent, and both surfaces covered

`internal/schema/graph.pb.go:11-14` stamps `protoc-gen-go v1.36.11` / `protoc v7.35.1` (verified
present). The tree has no checked-in codegen invocation today: no `go:generate` directive, no
Makefile, no `proto` target in `Taskfile.yml` (positive-controlled — `rg -c "task" Taskfile.yml`
returns 55, `rg -n "proto" Taskfile.yml` returns nothing), and no `buf.yaml`/`buf.gen.yaml`. So the
first `buf generate` genuinely will restamp that header. 01-01 absorbs it deliberately, in its own
reviewed commit, verifying every changed hunk is inside the header block (01-01-PLAN.md:281-295),
and 01-07 then treats any header change as drift. 01-01's acceptance covers **both** surfaces:
`git status --porcelain internal/uiproto internal/schema` is asserted empty after `task proto:gen`.
The residual — that a toolchain bump presents identically to a source drift — is exactly what
BLD-04's flagged assumption records (01-07-PLAN.md:266-272). Ruling accepted.

### Wave ordering — clean

Waves are 1:{01-01,01-02,01-03} 2:{01-04,01-06} 3:{01-05,01-07} 4:{01-08} 5:{01-09} 6:{01-10}
7:{01-11}. No same-wave `files_modified` overlap: the wave-1 trio partitions into
`uiproto`/`uiserver`/`cli`/buf, `goldenspec`/`testdata/golden`, and `internal/mcp`; wave 2 splits
`internal/query/{detail,node,traverse}.go` from `internal/query/{engine,meta_test}.go`; wave 3
splits `internal/query/*` from `Taskfile.yml`/`internal/upgrade`/`ci.yml`/`*.pb.go`. Cross-wave
reuse of `internal/schema/graph.pb.go` (01-01 → 01-06 → 01-07) is sequential and therefore fine.
ROADMAP blocking order holds: SRV-02's guard is 01-01 Task 1, before that plan's own tracer handler
and four waves before the RPC surface; the byte oracle (01-02) precedes ENG-01 (01-04) which
precedes ENG-02 (01-05), and the seam-backed view RPCs (01-09) depend on both.

### Both correction blocks honored

- **D-04.** Verified against source that no test today byte-diffs live `Engine` output against the
  goldens: `TestReFrozenGoldensValid` (`testdata/golden/golden_test.go:243`) asserts existence,
  non-emptiness, envelope parse and a positive count — it never constructs an `Engine`. Building
  that oracle is 01-02, wave 0/1, before either extraction. Honored.
- **Folded Todos item 1.** 01-03 carries a dedicated must_have that closing FIX-01 is proven NOT to
  close `toolslist-repeat`, plus a prohibition against reporting otherwise (01-03-PLAN.md:29, and
  the prohibitions block). Honored.

### Vacuity guards after the split — no stranded guards found

Every zero-expecting `rg` gate in the phase carries an in-invocation positive control
(01-01:208, 01-02:174, 01-04:263, 01-05:211, 01-10:277) and strips comment lines first. Counted
guards carry positive assertions of what they inspected: `attempted == 26` computed before any
subtest runs (01-02), classified-error-site count (01-05), compared-generated-file count (01-07),
inspected message/field counts (01-09), `openEngine` invocation counts (01-08, 01-11). Rule
`84d1gfpywd` is satisfied at the assertion level — **but is defeated at the shell level by
HIGH C2-2 below**, which lets a failing test pass any of these gates.

### Agreed Concerns (both reviewers, confirmed against source)

1. **HIGH C2-1 — the cross-wave field-number guard cannot be executed as written.** 01-09 allocates
   the `SourceBlob` field numbers for `GetNodeDetailResponse`, `NodeDefinition` and `ExploreGroup`
   *as comments only* ("Reserve the next free number in each message here, with a comment naming
   plan 01-10 as the owner", 01-09-PLAN.md:145-150), while its acceptance requires "The known-number
   fixture includes the field numbers reserved for plan 01-10, so 01-10 cannot allocate them
   elsewhere without failing this test" (01-09-PLAN.md:334) on a test that reads the generated
   descriptor. A protobuf descriptor carries declared fields and `reserved` ranges; it does not
   carry a comment. Three outcomes, all bad: the fixture entry has no descriptor field to match and
   the test fails in 01-09; or the test skips absent entries and the claimed protection is vacuous;
   or a real `reserved N;` clause is emitted and `protoc`/`buf` then refuses 01-10's use of that
   same number. This is a *new* defect introduced by the fix for cycle-1 HIGH #9.
2. **HIGH C2-2 — 26 automated gates discard the test command's exit status.** Every `<automated>`
   PASS-count gate has the shape
   `COUNT=$(go test -v ... 2>&1 | rg -o -e '--- PASS' | wc -l); [ "$COUNT" -ge N ]`. The pipeline's
   status is `rg`'s, not `go test`'s, and no plan sets `set -o pipefail` or inspects `PIPESTATUS`
   (positive-controlled: `rg -n "pipefail|PIPESTATUS|STATUS="` over `01-*-PLAN.md` returns nothing,
   against 26 matches for the PASS-count idiom). A run with N passing subtests and one `--- FAIL`
   satisfies the gate. 01-01-PLAN.md:200 states the choice explicitly — "The gate is the PASS count,
   not the exit status: `go test -run PATTERN` exits 0 when the pattern matches nothing" — but that
   rationale argues for *adding* the count, not for *dropping* the status. Both are needed; this is
   the same vacuity class the phase spent its guard budget eliminating.
3. **MEDIUM — 01-01 admits cross-spelling Host/Origin pairs.** `allowedHosts(port)` and
   `allowedOrigins(port)` are two independent sets checked independently (01-01-PLAN.md:179-197), so
   `Host: 127.0.0.1:P` with `Origin: http://localhost:P` is admitted. ROADMAP success criterion 2
   requires the three spellings be "each admitted by exact match rather than treated as
   interchangeable" (`.planning/ROADMAP.md:128`). Pair them: when Origin is present, require its
   authority to equal the admitted `r.Host` exactly, and add negative cross-spelling cases in both
   directions to the rejection matrix.
4. **MEDIUM — 01-03 forwards outside the mutex.** The plan takes the classification-buffer mutex
   *after* `p.w.Write(b)` (01-03-PLAN.md:168, 181). `io.Writer` promises nothing about concurrent
   calls, so two concurrent `Write`s can reach the wire in one order and the classification buffer
   in another, and the copied line stream stops matching wire order. Serialize the underlying write
   and the `b[:n]` classification in one critical section — "forward first" should mean *within* the
   serialized section, not outside synchronization.
5. **MEDIUM — 01-01 `Serve(ctx)` concurrency shape unspecified.** The action says `Serve` calls
   blocking `srv.Serve(ln)` and, "on `ctx.Done()`", calls `Shutdown` (01-01-PLAN.md:349-353). Read
   literally that is unreachable — the blocking call must be in a goroutine with a buffered
   serve-error channel and a `select`. Prescribe the shape and the `http.ErrServerClosed`
   normalization rather than leaving it to the executor.
6. **MEDIUM — 01-11 transient-lock case still uses wall-clock coordination.** Releasing the holder
   "comfortably inside" `graphstore.Open`'s retry budget (01-11-PLAN.md:247, 251) is timing-based
   even though no elapsed-time assertion is made, and can cross the boundary spuriously under a
   loaded `-race` CI. Either expose an event seam to `uiserver` tests or keep one integration smoke
   case and leave deterministic boundary coverage entirely to `internal/graphstore`'s existing
   event-synchronized test.
7. **LOW — 01-06 SHA validation assumes SHA-1.** Accepting only 40 lowercase hex characters excludes
   Git SHA-256 repositories, which would silently record an absent SHA. Acceptable for v1, but state
   it as a scoped limitation in the flagged assumption or accept 64 characters too.
8. **LOW — cross-plan artifact manifest drift.** `(*Engine).SourceFor` is declared by 01-04
   (01-04-PLAN.md:97) and appears in 01-08, 01-09, 01-10 and 01-11's "Created elsewhere in this
   phase" manifests, but is missing from 01-02's and 01-05's. Those manifests exist so drift
   verification sees the whole set; two of them are incomplete.

### Divergent Views

None — single grounded reviewer. Every finding above was independently re-derived against the tree
by the orchestrator before being recorded; none is carried on the reviewer's assertion alone.

## Verification coverage (cycle 2 source-grounding pass)

**Adapter caveat applied.** `drift-guard authority` resolves to `intel`, but
`.planning/intel/API-SURFACE.md` reports `symbolCount: 0` on this Go repo (the extractor is
regex/JS-only). Under that authority every Go symbol would grade UNCHECKABLE → INFO and nothing
would hard-block. The intel map was therefore **not consulted as evidence**; every cited symbol
below was resolved by reading source. Absence in intel was treated as "unknown", never as "does not
exist", per that file's own instruction.

**False-drift traps observed.** `.proto` snake_case vs generated Go camelCase was not treated as
drift. `rg` exits non-zero for both "no match" and "file not found", so every zero result below was
positive-controlled with a second pattern known to match in the same invocation.

### Resolved FOUND against source

`query.OpenAt` (`internal/query/engine.go`), `(*Engine).Status` (`internal/query/status.go:247`),
`StatusResult`, `(*Engine).Node` (`internal/query/node.go:316`), `(*Engine).Explore`
(`internal/query/explore.go:240`), `RenderNode`, `RenderNodeMultiDef`, `renderMultiDefNode`,
`nodeMultiDefHardCap`, `(*Engine).readSourceFile` (`internal/query/node.go:84`),
`(*Engine).resolveSourcePath` (`internal/query/node.go:33`), `(*Engine).fetchCalls`
(`internal/query/node.go:363`), `(*Engine).fetchCalledBy` (`internal/query/node.go:399`),
`BuildReverseAdjacency`, `groupMatchesByFile`, `exploreZeroResult`, `validateDepth`, `clampDepth`,
`clampAffectedDepth`, `validateLimit`, `MaxLimit`, `validateMaxFiles`, `validateFilesDepth`,
`FilesOptions`, `FilesResult`, `ResolveCodegraphDir`, `ErrNotInitialized`,
`graphstore.ErrStoreLocked`, `graphstore.ErrNotFound`, `graphstore.Open`
(`internal/graphstore/pebble_store.go:141`), `openLockRetrySleep`, `schema.NewMeta`,
`Meta.HasFileIndex`, `Meta.LastSyncUnixMs`, `PutMeta`, `pendingWriter`, `stdinLingerReader`,
`looksLikeJSONRPCCall`, `waitForDrain`, `stdinLingerGrace`, `BuildServer`,
`goSDKServer.ServeStdio` (`internal/mcp/server.go:225`), `resolveStartPath`, `onSyncStart`,
`TestWorkflowRunBodiesInvokeTask`, `Engine.WorktreeMismatch`, `internal/mcp/session_line.go`,
`internal/query/{validate,files,search,traverse,render_markdown,engine}.go`,
`internal/indexer/{resolve,sync}.go`, `internal/schema/meta.go`, `.github/workflows/ci.yml`,
`testdata/golden/gocapture/main.go`, `internal/corpora` corpus resolution.

Also verified as facts the plans assert:
- `internal/schema/graph.pb.go:11-14` — the generated header block, `protoc-gen-go v1.36.11` /
  `protoc v7.35.1`. `v7.35.1` is a real protoc identity (protobuf's language-version prefix, i.e.
  protoc 35.1), not a placeholder.
- **26 goldens confirmed**, and the derivation matches the plans: 24 tracked under
  `testdata/golden/corpus/{hugo,guava,requests,serilog}/` (6 each) plus 2 at
  `corpus/behavioral/{go-explore-multi,go-node-multi}.json`. `testdata/golden/golden_test.go:317`
  asserts `goldenTotal == 26` from `expectedGoCaptures`, and `ExpectedGoldenScenarioCount = 30`
  (26 + 4 `CASES.json` property cases). 8 of the 26 are `-mcp.json` captures; 01-02 routes those
  through `goldenspec.CallNodeViaMCP`/`CallExploreViaMCP` rather than through `Engine` directly
  (01-02-PLAN.md:200), so the oracle's mechanism is correct even though the plan's summary truth at
  01-02-PLAN.md:27 says "Engine.Node/Engine.Explore" for all 26.
- `Taskfile.yml` has **no** proto target (zero result positive-controlled against 55 matches for
  `task`); no `buf.yaml`, `buf.gen.yaml` or `go.tool-proto.mod` exists; `internal/textutil` does not
  exist. All confirm BLD-04 closes a genuine pre-existing gap.

### Resolved ABSENT — and load-bearing

- `(*Engine).SourceFor` — **ABSENT** (zero result positive-controlled). Declared as an artifact
   01-04 produces (01-04-PLAN.md:97), so correctly excluded from drift grading; recorded here only
  because it is missing from 01-02's and 01-05's cross-plan manifests (LOW finding 8 above).
- `looksLikeJSONRPCResponse`, `sniffedOutbound`, `decrementPending`, `pendingUnderflows` — ABSENT,
  declared as 01-03 artifacts. Excluded.

### UNCHECKABLE / skipped — and why

| Symbol | Verdict | Reason |
|---|---|---|
| `connect.WithSendMaxBytes`, `connect.WithReadMaxBytes`, `connect.NewErrorDetail`, `connect.CodeOf`, `connect.CodeUnavailable`, `connect.CodeInternal`, `connect.CodeNotFound`, `connect.CodeInvalidArgument`, `connect.CodeResourceExhausted` | **UNCHECKABLE (unchanged from cycle 1)** | `connectrpc.com/connect` is added by this phase and is still absent from `go.mod`. 01-01's acceptance now requires these signatures be verified and recorded in the SUMMARY immediately after the dependency lands — the correct disposition. |
| `uiv1connect.NewUIServiceHandler` / `NewUIServiceClient` / `UIServiceHandler`, all `uiv1` messages | **SKIPPED (new)** | Generated by a codegen pipeline this phase creates. |
| `buf` v1.72.0 / `protoc-gen-connect-go` v1.20.0 / `protoc-gen-go` v1.36.x MVS compatibility | **UNCHECKABLE** | Requires actually running `go mod tidy -modfile=<scratch>` and building each tool. 01-01 Task 2 schedules exactly that measurement, with the outcome to be recorded in the modfile header. |
| Whether `buf generate` reproduces `graph.pb.go` byte-identically apart from the header | **UNCHECKABLE** | No buf toolchain present. This is the basis of the header-drift ruling and must be measured at execution. |
| Whether `buf` stamps a compiler identity at all (and so whether the restamp is a rewrite or a blanking) | **UNCHECKABLE** | Same reason. 01-01 handles either outcome by requiring every changed hunk to be inside the header block. |
| All `internal/uiserver`, `internal/goldenspec`, `internal/textutil`, `internal/query/{detail,errors}.go`, `internal/indexer/commit.go`, `internal/upgrade/proto_task_test.go` symbols | **SKIPPED (new)** | Listed under 01-02-PLAN.md's "Artifacts this phase produces" manifest, which enumerates the complete phase-wide new-symbol set. Excluded by the source-grounding rule. |
| `.planning/intel/API-SURFACE.md` | **NOT CONSULTED** | `symbolCount: 0` on a Go repo; extractor is regex/JS-only. Consulting it would have graded every symbol UNCHECKABLE and hard-blocked nothing. |
| `go-sdk@v1.7.0` calling `jsonrpc2.Async(ctx)` on every non-`initialize` call | **UNCHECKABLE** | Third-party module internal behaviour; the claim is CONTEXT's, and 01-03 does not depend on it beyond keeping `toolslist-repeat` open and separately attributed. |

**A clean line here does not mean "nothing was checked."** 50+ symbols resolved FOUND against source
with `file:line`; the 26-golden derivation independently reconstructed from disk and cross-checked
against `golden_test.go`'s own assertion; four zero-results positive-controlled; one symbol resolved
ABSENT-and-declared; the new-artifact set confirmed absent and deliberately excluded; and 7 items
recorded UNCHECKABLE, every one blocked on the `connectrpc.com/connect` and `buf` toolchains this
phase itself introduces.

---

# Cross-AI Plan Review — Phase 1 (cycle 3, final convergence)

Everything above this line is cycles 1 and 2. The plans were replanned in response
(commit `1b97fab`) and remain 11 plans / 7 waves. This file is APPEND-style history:
a prior finding is closed only if the section below says so.

## Codex Review

## Summary

The plans are substantially converged: the cycle-2 security, gate-hygiene, field-number, concurrency, and SHA-format findings are incorporated coherently. Source inspection supports the main architectural assumptions, including the lazy Node extraction boundary, five Explore zero-result branches, exclusive Pebble locking, and the `pendingWriter` defect. Two actionable issues remain: plan 01-11 cites a nonexistent final-attempt lock test, and its concurrent-holder test requires a fairness property the store does not guarantee.

## Strengths

- The golden oracle is necessary and correctly precedes extraction. The current `Node` path mixes gather/render logic at [internal/query/node.go:316](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:316), while `Explore` contains five separate zero-result returns at lines 293, 357, 410, 473, and 559. Plans 01-02, 01-04, and 01-05 address both risks before downstream RPC mapping.

- The lazy multi-definition design matches the existing behavior. Source, calls, and callers are fetched inside the callback at [internal/query/node.go:448](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/query/node.go:448), so plan 01-04 correctly avoids eager reads.

- FIX-01 is grounded in the actual imbalance: inbound calls increment at [internal/mcp/server.go:271](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:271), but every outbound write currently decrements at [internal/mcp/server.go:339](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/mcp/server.go:339). Plan 01-03 now correctly serializes write and classification and limits classification to bytes actually written.

- C2-1 is resolved cleanly. Plan 01-09 guards only fields it declares and requires every fixture entry to resolve; plan 01-10 extends that fixture by exactly nine entries after declaring the fields. Both halves are runnable in their respective waves.

- C2-2 is consistently corrected. All inspected automated test gates capture `STATUS` before piping and require both status zero and a PASS floor. The deliberate-mutation observations in plans 01-02 and 01-05 correctly retain nonzero exit as expected evidence. Plan 01-07 similarly preserves `task proto:drift` status in `DSTATUS`.

- Raised floors appear feasible:

  - Origin/Host describes at least 20 request cases for a floor of 19.
  - `pendingWriter` includes seven classification subcases plus the parent and several additional tests, comfortably supporting 15.
  - Server lifecycle names six top-level tests for a floor of 6.
  - SHA tests include both object formats, five invalid forms, and multiple top-level tests, supporting 9.

- The Pebble lifecycle premise is accurate. `Open` performs five exclusive-lock attempts at [internal/graphstore/pebble_store.go:141](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/pebble_store.go:141), and the retry seam explicitly exists to avoid wall-clock tests at [internal/graphstore/pebble_store.go:83](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/pebble_store.go:83).

- The probe ledger is internally reconciled: 15 edge/backstop items plus six explicitly flagged assumptions. The pending-writer concurrency refinement is documented as strengthening an existing concurrency probe rather than silently adding another; the SHA object-format uncertainty is closed through tests rather than retained as an assumption.

## Concerns

- **HIGH — ACTIONABLE:** Plan 01-11 depends on a final-attempt boundary test that does not exist. It instructs the executor to cite “the existing event-synchronized final-attempt test” and stop if none is found ([01-11-PLAN.md:286](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-11-PLAN.md:286)). The actual event-synchronized test is `TestOpenConvergesWhenHolderCloses`, but it releases when attempt 2’s sleep begins, leaving several attempts available—not on the final attempt ([internal/graphstore/open_lock_test.go:48](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/open_lock_test.go:48), [internal/graphstore/open_lock_test.go:76](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/open_lock_test.go:76)). As written, execution must stop.

- **MEDIUM — ACTIONABLE:** `TestConcurrentRPCsDoNotStarveAHolder` requires fairness that Pebble and the retry loop do not promise ([01-11-PLAN.md:238](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-11-PLAN.md:238)). Every open competes for one exclusive lock, and each caller only gets five attempts with fixed backoff ([internal/graphstore/pebble_store.go:67](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/pebble_store.go:67), [internal/graphstore/pebble_store.go:141](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/pebble_store.go:141)). Independent short-lived UI opens can legitimately reacquire between holder attempts. This may flake without indicating retained handles, and it tests more than SRV-04 requires.

- **MEDIUM — already covered as execution risk:** Plans 01-01, 01-09, 01-10, and 01-11 remain large at 70–96k estimated tokens. They are decomposed into atomic tasks with buildable boundaries, so no correctness change is mandatory, but context exhaustion remains plausible—especially plan 01-11 with tests plus three manual multi-process transcripts.

## Suggestions

- Amend plan 01-11 to cite `TestOpenConvergesWhenHolderCloses` accurately as an event-synchronized “release between attempts” test. Either add a true final-attempt boundary test to `internal/graphstore` or remove every claim that one already exists.

- Replace the holder-starvation test with a deterministic retained-handle property: complete concurrent RPC calls, then causally start the holder and require its open to succeed. Keep the existing structural no-handle-field check and per-call open/close instrumentation as the primary SRV-04 evidence.

- During execution, consider splitting plan 01-01 after the guarded server tracer and plan 01-11 before manual process evidence if context pressure becomes material. No wave redesign is needed.

## Risk Assessment

**MEDIUM.** The phase architecture and nearly all convergence corrections are sound. The remaining blocker is localized but real: plan 01-11 currently orders the executor to rely on a source test that does not exist. Correcting that citation/coverage and removing the unsupported fairness assertion should make the plan set ready to proceed.


---

## Verification coverage (cycle 3 source-grounding pass)

The `drift-guard authority` resolves to `intel`, but `.planning/intel/API-SURFACE.md` carries
`symbolCount: 0` on this Go repo — the extractor is regex/JS-only. It was **NOT CONSULTED**:
grading against it would have marked every Go symbol UNCHECKABLE → INFO and hard-blocked
nothing. Every symbol below was resolved by reading source.

### Method

Five independent verification passes over the 11 plans plus the repo tree: (1) an exhaustive
census of every PASS-count gate; (2) probe accounting and every raised floor counted against
the subtests the plans actually describe; (3) the cross-wave field-number guard and the 01-11
causal edge; (4) every negative/zero-count gate checked for self-invalidation, positive
control and wave reachability; (5) symbol source-grounding with artifact manifests excluded.

### Exclusions

Every plan carries an `## Artifacts this phase produces` section with a "Created by this
plan" and a "Created elsewhere in this phase" block. Both are creation manifests, so absence
today is correct and expected. Excluded and confirmed genuinely absent from `main`:
`internal/goldenspec` (whole pkg); `testdata/golden/byte_identity_test.go`;
`internal/uiproto/uiv1` (whole pkg, `ui.proto`, `UIService` + 9 rpcs, all messages,
`SourceBlob`, generated `ui.pb.go` / `uiv1connect/ui.connect.go`); `internal/uiserver`
(whole pkg, incl. `openEngine`, `withEngine`, `originHostGuard`, `allowedHosts`,
`allowedOrigins`, `uiService`, `mapEngineError`, `uiMultiDefCap`, `truncateSource`,
`classifyDegrade`, `degradeKind`); `internal/cli/ui.go`; `internal/textutil`;
`internal/query`'s `NodeDetail`/`ExploreResult` families, `ErrNotFound`,
`ErrInvalidArgument`, `(*Engine).IndexMeta`, `(*Engine).SourceFor`, the `build*` builders;
`schema.Meta.commit_sha` field 8 / `schema.IndexedCommitSHA`; `internal/indexer/commit.go`;
`internal/mcp`'s `looksLikeJSONRPCResponse`, `sniffedOutbound`, `decrementPending`;
`buf.yaml`, `buf.gen.yaml`, `go.tool-proto.mod`, `Taskfile.yml`'s `proto:gen`/`proto:drift`;
module deps `connectrpc.com/connect` and `github.com/pkg/browser`.

Not drift, recorded so a later reader does not re-raise it: an unrelated `openEngine`
function already exists at `internal/mcp/tools.go:65`. Different package, no collision — the
name is simply not novel in the tree. `.proto` snake_case vs generated Go camelCase was
likewise excluded by rule.

### Verified against source

Over 80 cited symbols resolved FOUND with `file:line`. The load-bearing ones:

| Claim | Verdict | Evidence |
|---|---|---|
| `(*Engine).Node` / `(*Engine).Explore` compute structured data before rendering | **VERIFIED** — the "pure extraction" premise holds | `node.go:420` computes `fetchCalls`→`BuildReverseAdjacency`→`fetchCalledBy` then calls `RenderNode` at `node.go:439`; `node.go:442` passes a lazy `fetch` closure to `RenderNodeMultiDef` at `:464`; `explore.go:240` computes `groups` `:558`, `blasts` `:568-577`, `sources` `:579-586`, `skeletonFiles` `:594`, then the file's single `RenderExplore(` call at `:596` |
| `Explore()` has five zero-result branches | **VERIFIED** — exactly five | `explore.go:294, 358, 411, 474, 560` |
| `schema.Meta` has exactly 7 fields and NO commit SHA | **VERIFIED**, field 8 free | `graph.proto:137-151` |
| `graphstore.Open` bounded retry budget | **VERIFIED** — 5 attempts / 4 sleeps = 400 ms floor | `pebble_store.go:79-80`, loop at `:141-155` |
| `openLockRetrySleep` test-only seam, unexported | **VERIFIED** | `pebble_store.go:83-90` |
| `pendingWriter` root cause | **VERIFIED exactly as described** | unconditional `p.pending.Add(-1)` on every write at `server.go:339`; the only increment is `s.pending.Add(1)` at `:276`, gated on `looksLikeJSONRPCCall` (`:330`, `msg.Method != "" && msg.ID != nil`) inside `stdinLingerReader.Read`. No mutex and no line buffering on `pendingWriter` today |
| 26 frozen goldens, and no byte-diff oracle exists | **VERIFIED** | `expectedGoCaptures` `golden_test.go:170-220` = behavioral(2) + hugo/guava/serilog/requests(6 each) = 26, cross-asserted `golden_test.go:318`; `TestReFrozenGoldensValid` (`:243-296`) never constructs an Engine; `loadBehavioralFixture` (`:186`) and `loadGoldenOutputIn` (`:443`) have **zero callers** |
| No proto codegen/drift guard exists today | **VERIFIED on four counts** | `rg -i 'proto' Taskfile.yml` → 0 hits; no `go:generate` proto directive; no `Makefile`; `rg -i 'proto\|buf' .github/workflows/ci.yml` → 0 hits; `buf.yaml`/`buf.gen.yaml` absent |
| `(*Engine).SourceFor` in all 11 artifact manifests | **VERIFIED 11/11** | absent from source (`rg -n 'SourceFor' --type go` → 0 hits), declared new by 01-04, named in all of 01-01…01-11 |
| Exactly three `PutMeta` write sites | **VERIFIED** | `resolve.go:774`, `sync.go:170`, `sync.go:398` |
| 01-11's causal edge is fired from the wrapped `openEngine` seam | **VERIFIED** | `01-11-PLAN.md:257-262` — wrap the `openEngine` package var, signal a channel on first invocation only via `sync.Once`, restore with `t.Cleanup`; holder blocks on that channel. No wall-clock coordination survives: every remaining `time.*` mention in 01-11 is prose *rejecting* timers |
| C2-1 on the `SourceBlob` path (01-09 → 01-10) | **RESOLVED** | 01-09's fixture is scoped to fields 01-09 declares (`01-09:329-336`), with mandatory resolution (`resolved == len(fixture)`, `len(fixture) > 0`, unresolved entries named and FAILED, `01-09:372`); 01-10 extends by exactly nine (`01-10:37`, `:373`). Both halves runnable in their declaring wave |
| C2-2 across all 27 PASS-count gates | **RESOLVED** | all 25 `go test` gates capture `OUT`/`STATUS` before any pipe, assert `STATUS -eq 0` AND a floor, and dump on failure; the two `01-07` drift gates capture `DRIFT`/`DSTATUS` first (`01-07:183`, `:245`). No bare count-only form survives. All 27 pass `sh -n`; no `STATUS=$?` follows a pipe anywhere |
| The deliberate-RED scoped exception | **CORRECTLY SPLIT** | all five mutation observations are scored by `--- FAIL`/`--- PASS` counts, never by `STATUS -eq 0`; each owning task's `<automated>` gate is fenced to the *restored* tree (`01-05:356`, `01-07:245` both guarded by `test -z "$(git status --porcelain …)"`). No plan requires a zero exit on a deliberate-RED observation |
| Probe accounting | **RECONCILES AT EXACTLY 21** | 15 edge-tagged `must_haves` (01-01:47,48; 01-02:29,30; 01-04:30,31; 01-05:37,38; 01-10:38,39; 01-11:29-33) + 6 flagged assumptions (01-01:526,530,534; 01-03:342; 01-06:365; 01-07:267). Exactly one flat-scalar `verification: backstop` (01-01:49) — the only `verification:` YAML key in the phase. Both "recorded in place" reconciliations are genuinely present: the deliberately-untagged 01-03 concurrency property at `01-03:25`, the closed 01-06 object-format item at `01-06:358-364`. Neither hides a drop |
| Both `01-CONTEXT.md` correction blocks | **PRESENT AND HONORED** | D-04 scoring correction at `01-CONTEXT.md:71-88`; the `toolslist-repeat`-is-a-different-bug disproof at `:259-260` |
| Raised PASS floors | **all four satisfiable**, but see M8 | 19 vs 20 described cases (`01-01:155-174`); 15 vs 13 guaranteed + 2 conditional (`01-03`); 6 vs exactly 6 test functions (`01-01:260-265`); 9 vs 14-15 (`01-06:260-268`) |
| Wave/dependency graph | **CONSISTENT** | every `depends_on` resolves to a strictly lower wave; no same-wave file-ownership overlap |
| Wave reachability of every gate target | **NO DEFECTS** | each negative gate's target file exists at its declaring wave, verified against the repo and earlier plans' `files_modified` |

### Positive controls and vacuous zeros

Four zero-results were positive-controlled. Three negative gates are correctly same-file
controlled (`01-02:176`, `01-05:213`, `01-10:289`), and `01-11:325` is the best-constructed
gate in the phase — same-file control, anchored pattern, and its own self-invalidation risk
named in the criterion. Two are **not** (see M2). Positive controls that read zero *today*
were confirmed correct-post-change, not broken: `ExploreFileGroup` (0 → 19 after 01-05's
rename), `textutil\.` in `session_line.go`, `goldenspec\.` in `gocapture/main.go`.

Verified in-repo rather than assumed: `git status --porcelain <missing-path>` exits **0**
with empty stdout (the "could not open directory" warning goes to stderr), which is what
makes M4 real; and `rg -v '^\s*//' /nonexistent | rg -o … | wc -l` prints `0`, which is what
makes M2 real.

### UNCHECKABLE — and why

| Item | Reason |
|---|---|
| `connect.WithSendMaxBytes` / `WithReadMaxBytes` / `NewErrorDetail` / `CodeOf` and the `connect.Code*` constants | `connectrpc.com/connect` is added by this phase and is still absent from `go.mod`. 01-01's acceptance already requires these signatures be verified and recorded in the SUMMARY as soon as the dependency lands — the correct disposition, unchanged from cycles 1-2 |
| `buf` / `protoc-gen-connect-go` / `protoc-gen-go` MVS compatibility; whether `buf generate` reproduces `graph.pb.go` byte-identically; whether `buf` stamps a compiler identity at all | No buf toolchain present. 01-01 Task 2 and 01-07 schedule exactly these measurements and handle either outcome |
| "Nineteen of this repo's forty-nine non-vacuity guards live in `taskfile_shape_test.go`" | Not a symbol; the 49-guard census has no machine-checkable definition. The file is 1350+ lines of literal-fixture guards, consistent with the claim, but the counts cannot be reproduced |
| go-sdk `jsonrpc2.Async(ctx)` on every non-`initialize` call | Third-party internal behaviour. The call site itself was confirmed at `go-sdk@v1.7.0/mcp/server.go:1910-1915`; 01-03 does not depend on it beyond keeping `toolslist-repeat` separately attributed |

**A short UNCHECKABLE list is not an unchecked review.** 80+ symbols resolved against source
with `file:line`; the 26-golden derivation independently reconstructed from disk and
cross-checked against `golden_test.go:318`; the absence of a golden byte-diff oracle proven
by zero-caller analysis; the no-proto-codegen claim proven on four independent counts; 27
gates parenthesis-matched, `sh -n`-checked and behaviourally simulated against both failure
modes; and every remaining UNCHECKABLE blocked on the `connectrpc.com/connect` and `buf`
toolchains this phase itself introduces.

---

## Cycle 3 findings

### HIGH — unresolved

**H1. C2-1 survives untouched on the `01-08` → `01-11` path: `GetStatusResponse` fields 8
and 9 are pinned by nothing, and 01-11 claims otherwise.**

The cycle-2 fix was applied rigorously to the `SourceBlob` path and is closed there. The
identical defect class is live in a message the fix never covered:

- `01-09:329` states the rule: "The fixture covers ONLY fields this plan actually declares."
- `GetStatusResponse` is declared by **01-08** (wave 4). `rg 'GetStatusResponse' 01-09-PLAN.md`
  returns **zero hits** — 01-09's fixture contains no `GetStatusResponse` entry at all.
- 01-08 has no field-number fixture test; it records the allocation only in a proto comment
  (`01-08:166-168`) and a `must_haves` line (`01-08:39`).
- Fields 8 and 9 do not exist until **01-11** (wave 7), and nothing instructs 01-11 to extend
  the fixture — unlike 01-10, which is told to explicitly.
- `01-11:221` nevertheless asserts "plan 01-09's `TestUIProtoFieldNumbersAreStableAndUnique`
  still passes with fields 8 and 9 landing where they were allocated." The test *will* pass —
  and asserts nothing whatsoever about fields 8 or 9. That is a vacuous cross-wave guard.
- `01-11:429` goes further and is simply false: "the field numbers it uses were allocated by
  plan 01-08 and are already pinned by plan 01-09's fixture."

Fix: have 01-11 EXTEND the fixture with `GetStatusResponse.store_exists = 8` and
`.indexing_in_progress = 9` (mirroring 01-10's `+9` pattern), and correct `01-11:429`.
Optionally give `GetStatusResponse`'s fields 1-7 fixture entries in 01-08.

**H2. 01-11 delegates the final-attempt boundary to a `graphstore` test that does not exist,
leaving the SRV-04 adjacency-edge probe unbacked.**

`01-11:32` — an edge-tagged probe, one of the 21 — asserts "the exact final-attempt boundary
stays owned by `internal/graphstore`'s existing event-synchronized test." `01-11:121`, `:233`
and `:298-309` repeat it and instruct the executor to cite that test by name in the shipped
package comment.

`internal/graphstore/open_lock_test.go` has exactly two `Open` tests and neither is one:

- `TestOpenConvergesWhenHolderCloses` (`:62`) is event-synchronized, but releases at
  **attempt 2's** sleep. Its own doc comment (`:59-61`) says "some remaining attempt
  (budget: 5) deterministically finds the LOCK free" — a release-*between*-attempts test.
- The other (`:36-45`) asserts elapsed time against the budget — wall-clock, not
  event-synchronized, and it is the never-releases case.

The plan's stop-instruction ("if no such NAMED test is found, stop and report it") does not
fire, because a named event-synchronized test *is* found. And `01-11:326` only greps that the
cited name exists — it verifies existence, not the property. The executor will therefore cite
`TestOpenConvergesWhenHolderCloses`, ship a false attribution in a package comment, and leave
the adjacency edge unproven.

Fix: either add a genuine final-attempt boundary test to `internal/graphstore`, or restate
`01-11:32`/`:121`/`:233`/`:298-309` to describe what `TestOpenConvergesWhenHolderCloses`
actually covers and drop the "final-attempt boundary is owned there" claim.

### MEDIUM — actionable

**M1.** 01-02's central non-divergence guarantee is false as written. Its `must_haves` claims
"exactly one authoritative table" and "ONE authoritative MCP call shape", and its action says
to move `callExploreViaMCP` / `callNodeViaMCP` / `mcpResultText` **from
`testdata/golden/gocapture/main.go`**. But `testdata/golden/behavioral_test.go` independently
declares a second full set — `languageToLockedSlug:43`, `slugToRepo:52`,
`callExploreViaMCP:1181`, `callNodeViaMCP:1202`, `mcpResultText:1242` — in the very package
the new oracle file joins, and those are the copies `TestExploreCLIMatchesMCP` (`:1282`) and
`TestNodeCLIMatchesMCP` (`:1326`) already use. 01-02's zero-gate is scoped to
`gocapture/main.go` only, so it passes green while the duplicates survive. Fix: extend the
move and the gate to `behavioral_test.go`, or narrow the `must_haves` claim.

**M2.** Two zero-count gates are positive-controlled against a **different file**, so a
typo'd or renamed target reads green while the guarded file is never opened:
`01-01:238` (target `internal/uiserver/originguard.go`, control `internal/query/files.go`)
and `01-04:264` (target `node.go` + `detail.go`, control `gather.go`). Verified: the
pipeline prints `0` for a nonexistent path. Fix: same-file positive control. Secondary for
`01-01:238`: `internal/query/files.go` yields exactly 1 match today, so an unrelated
refactor there turns the gate red for the wrong reason.

**M3.** `01-04:264`'s forbidden pattern `\bgather[A-Z]` is unanchored, while `01-04:190-197`
spends six lines explaining the prohibition and naming `gatherChannel1/2/3` and `gatherMerge`
— all of which match it. A trailing comment or a `/* */` block survives `rg -v '^\s*//'` and
fails the gate for the wrong reason. Fix: anchor to `func gather[A-Z]`, the shape
`01-10:289` already uses.

**M4.** `test -z "$(git status --porcelain <path>)"` passes vacuously on a missing or typo'd
pathspec — verified in-repo, `internal/uiprotoTYPO/` exits 0 with empty stdout. Nine
`<automated>` sites: `01-05:356`, `01-07:245`, `01-08:212`, `01-08:273`, `01-08:328`,
`01-09:199`, `01-09:270`, `01-10:366`, `01-11:211`. Fix: prefix `test -d <path> &&`.

**M5.** `01-06:341` asserts `git diff --stat internal/query/status.go` is empty. `git diff`
reports only unstaged working-tree changes, so under GSD's atomic-commit-per-task rule this
is empty **by construction** at verification time regardless of whether the file was edited.
Fix: use a `<phase-base>..HEAD` revision range.

**M6.** `01-07:183` and `01-07:245` score the drift guard's non-vacuity with
`rg -q -e 'compared [0-9]+ generated files'`. `[0-9]+` matches `0`, so this is a presence
check on the count, not a floor — structurally the same vacuity the phase-wide conjunction
rule exists to close. The plan's own criteria demand at least 3 (`01-07:186`, `01-07:306`).
Fix: capture the count and assert `-ge 3`.

**M7.** `01-10:288`'s numeric-literal check has no command, no path scope and no positive
control, and the searched value is chosen at execution time. Unscoped it would match the
committed generated `internal/schema/graph.pb.go`. Fix: name the scope and exclude
`*.pb.go` / `*.connect.go`.

**M8.** 01-03's PASS floor of 15 is not derivable from the plan. It names 6 test functions
(`01-03:87-92`) and explicitly describes one 7-case table (`:131`) — 13 guaranteed `--- PASS`
lines. The remaining 2 must come from `:132`'s two cases being `t.Run` subtests, which no
behavior bullet or acceptance criterion requires. Separately, behaviors `:136` (two responses
in one `Write`) and `:137` (a response split across two `Write`s) are **orphaned** — each
names a distinct decrement case but is assigned to no test function. An executor who writes
`:132` as sequential assertions in one body produces 13 and the gate cannot pass. Fix: assign
`:136`/`:137` to named tests and state `:132`'s subtest structure, or lower the floor to 13.

**M9.** `01-11:241`'s `TestConcurrentRPCsDoNotStarveAHolder` requires a fairness property
neither Pebble's exclusive directory LOCK nor the retry loop provides. Every open competes
for one lock and each caller gets 5 attempts on a fixed 100 ms backoff
(`pebble_store.go:79-80`, `:141`); independent short-lived UI opens can legitimately
reacquire between the holder's attempts, so the test can fail with no retained handle
present. It also asserts more than SRV-04 requires. Fix: replace with a deterministic
retained-handle property — complete the concurrent RPCs, then causally start the holder and
require its open to succeed — keeping the structural no-handle-field check and the
per-call open/close instrumentation as the primary SRV-04 evidence.

**M10.** `01-10:373` asserts the fixture "has GROWN by exactly nine ... compared against
01-09's recorded length plus nine", but 01-09 never pins a length — it asserts only
`len(fixture) > 0` and `resolved == len(fixture)`. The `+9` arithmetic therefore depends on a
number carried across two waves in a prose SUMMARY rather than in code. Fix: pin 01-09's
fixture length as a named constant the extension can reference.

### LOW — actionable

**L1.** The `--- PASS` counting convention (parent line **plus** subtests) is never stated
anywhere in the phase. It is inferable only by arithmetic from `01-05:356` (floor 27 against
26 goldens). The gates' own echo labels contradict each other inside one file: `01-01:231`
prints `PASS subtests:`, `01-01:420` prints `PASS tests:`, everything else prints
`PASS lines:`. Every floor in the phase depends on this convention.

**L2.** `01-07:307`'s exit criterion states a PASS floor with no exit-status clause — the
count-only shape `01-07:37`'s own prohibition forbids. It is the only such bullet across the
11 plans; the matching gate at `01-07:183` is compliant, so this is a documentation
inconsistency rather than an executable defect. (`01-01:510` also omits the clause but is a
sub-criterion of `01-01:506`, which states the conjunction — subsumed, not a finding.)

**L3.** `01-11:325`'s forbidden-timer alternation omits `time.Tick`, and both degrade tests
must still bound "within the budget" somehow. `context.WithTimeout` is the sanctioned escape
and is nowhere named, so an executor may reach for a banned primitive or an unmentioned one.
Fix: add `time.Tick` to the pattern and state the sanctioned bounding mechanism.

**L4.** Six `read_first` citations point at use sites or the wrong file rather than the
declaration, so an executor following them loads the wrong lines: `goldenCapture` (01-02
cites `golden_test.go` 150-300; declared `behavioral_test.go:419`); `nodeSectionFetch` (01-04
cites 195-260; declared `render_markdown.go:158`); `nodeMultiDefHardCap` /
`nodeMultiDefBodyBudget` (01-04 and 01-09 both cite 206-225; declared `:145-148`, so the
values 16 and 12000 are never seen); `FilesOptions` and `validateFilesDepth` (01-08 cites
`files.go` 100-170; declared `:14` and `:86`); go-sdk `mcp/server.go:1908-1913` (actual
1910-1915); and `01-01`'s claim that `internal/daemon`'s `onSyncStart` is an established
**package-var** precedent — it is a struct field at `daemon.go:113`, so only one of the two
cited precedents matches the shape being copied.

### Closed this cycle

- **C2-1 on the `SourceBlob` path** — resolved. The withdrawal is real, 01-09's fixture is
  correctly scoped with mandatory resolution, and 01-10's `+9` extension re-runs it. Both
  halves are runnable in their declaring wave. (Residual on a different path: **H1**.)
- **C2-2** — resolved across all 27 gates, with the deliberate-RED exception correctly
  scoped to the five mutation observations and no plan requiring `STATUS -eq 0` on one.
- **Probe accounting** — reconciles at exactly 21; both "recorded in place" reconciliations
  are genuinely present in the plan text.
- **Paired Host/Origin check** — incorporated. Set membership AND `Origin == "http://" +
  r.Host`, four cross-spelling negatives in both directions, three admitted pairs asserted
  positively, floor 19 against 20 described cases.
- **`pendingWriter` write+classify in one critical section** — incorporated
  (`01-03:23`, `:172-176`). The floor is the issue, not the design (**M8**).
- **Prescribed `Serve(ctx)` shape** — incorporated; floor 6 against exactly 6 test functions.
- **01-11 wall-clock → causal edge** — incorporated and verified. The channel is fired from
  the wrapped `openEngine` var on first invocation via `sync.Once`; no wall-clock
  coordination survives.
- **SHA at 40 or 64 lowercase hex** — incorporated with negatives at 39/41/63/65/uppercase.
- **`(*Engine).SourceFor` in all artifact manifests** — verified 11/11.

### Scope

No plan has grown past what one execution context can hold, but four sit at the ceiling:
01-01 (96k), 01-09 and 01-11 (72k), 01-10 (70k). Each is decomposed into atomic tasks with
buildable boundaries, so no correctness change is required. If context pressure becomes
material at execution, the natural split points are 01-01 after the guarded server tracer
and 01-11 before the manual multi-process evidence. No wave redesign is needed.

### Risk

**MEDIUM.** The architecture, both correction blocks, the probe ledger and nearly all
convergence corrections are sound and source-verified. Two HIGH findings remain, both
localized and both surgical: a vacuous cross-wave field-number guard (H1) and a delegated
test that does not exist (H2). Neither requires re-planning the phase.

---

# Cross-AI Plan Review — Phase 1 (cycle 5, class-termination verification)

This cycle asked a narrower question than cycles 1-3: **not "are there more findings" but "did
cycle 4's termination of the gate-vacuity class actually hold".** Every census claim below was
re-derived independently rather than taken from cycle 4's report.

## Codex Review

## Summary

The vacuity class appears **TERMINATED**. The cycle-4 census holds across the 11 plans: all 27 automated gates parse, all 26 Go-test gates preserve exit status before counting output, all nine automated `git status --porcelain` checks have existence guards, the five prohibitions are canonical, and H1 is fixed at the root with bidirectional descriptor coverage. H2’s concurrency design is sound, but one new acceptance grep cannot match gofmt-formatted Go. That is the only unresolved issue I found.

## Verification Results

### A. Vacuity census — HELD

- Exactly 27 `<automated>` gates were extracted and all passed `sh -n`.
- Exactly 26 invoke `go test`; all capture output and test status before piping into `rg`/`wc`.
- Exactly nine automated gates contain `git status --porcelain`; all nine first require the target directory with `test -d`. Representative corrected gate: [01-05-PLAN.md:411](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-05-PLAN.md:411).
- The two count-bearing proto-drift gates capture `task` status and numerically require at least three files rather than accepting `compared 0`: [01-07-PLAN.md:224](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-07-PLAN.md:224), [01-07-PLAN.md:292](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-07-PLAN.md:292).
- The sole unranged-diff concern was corrected to a revision-range assertion with a same-range positive control: [01-06-PLAN.md:395](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-06-PLAN.md:395). Other unranged `git diff --stat` uses are mutation-applied confirmations where non-empty output is explicitly required, e.g. [01-05-PLAN.md:375](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-05-PLAN.md:375).
- Duplicate golden declarations are removed by plan rather than merely prohibited: both existing declaration sites and wrapper conversions are explicitly covered at [01-02-PLAN.md:190](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-02-PLAN.md:190) and [01-02-PLAN.md:259](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-02-PLAN.md:259).

### B. New prohibitions — HELD

All 11 `must_haves.prohibitions` lists carry the same five descriptor-less rules. The canonical block begins, for example, at [01-01-PLAN.md:65](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-01-PLAN.md:65).

The prohibitions are precise:

- existence guard before `git status --porcelain`;
- same-file positive control;
- numeric floor rather than a zero-accepting regex;
- declaration-anchored identifier grep;
- explicit revision range for emptiness assertions.

They appear only as prose inside the plan files; executable absence checks generally strip comment lines and scope themselves to source files, so the prohibition text does not poison those checks.

### C. PASS convention and floors — HELD

- The canonical “Gate conventions” sections hash to exactly one distinct digest across all 11 plans.
- All automated test gates print `PASS lines:`.
- Each ordinary gate requires the conjunction of status zero and its PASS floor. The deliberate-mutation exception remains scoped to expected-RED observations, as stated in [01-05-PLAN.md:396](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-05-PLAN.md:396).
- The described floor arithmetic is conservative and sufficient. For example:
  - 26 golden subtests plus parent = 27: [01-02-PLAN.md:390](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-02-PLAN.md:390).
  - Seven MCP writer functions plus table subtests = 18 against floor 15: [01-03-PLAN.md:302](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-03-PLAN.md:302).
  - Final degrade task derives 22 lines against floor 10: [01-11-PLAN.md:467](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-11-PLAN.md:467).

One minor prose slip says “eight test functions” in 01-06 while its arithmetic describes seven functions plus seven subtests; the resulting total of 14 is still correct and the floor is unaffected.

### D. HIGH root fixes

#### H1 — HELD

The fix is genuinely bidirectional:

- Every fixture entry must resolve, with `resolved == len(fixture)`: [01-09-PLAN.md:403](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-09-PLAN.md:403).
- The test must also walk every generated descriptor field and fail if the fixture omits it: [01-09-PLAN.md:457](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-09-PLAN.md:457).
- Earlier-wave `GetStatusResponse` fields 1–7 and `IndexingInProgress` are explicitly in scope: [01-09-PLAN.md:393](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-09-PLAN.md:393).
- The fixture-length chain is compiler-visible: baseline at [01-09-PLAN.md:409](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-09-PLAN.md:409), `+9` at [01-10-PLAN.md:431](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-10-PLAN.md:431), and `+2` at [01-11-PLAN.md:275](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-11-PLAN.md:275).
- The 01-11 gate actually runs the fixture test: [01-11-PLAN.md:296](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-11-PLAN.md:296).

#### H2 — DID NOT FULLY HOLD

The concurrency mechanism itself is correct. `Open` sleeps synchronously immediately before attempts 2–5 at [pebble_store.go:143](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/pebble_store.go:143). The planned unbuffered ordinal send followed by an unbuffered `resume` receive, with the helper closing the holder before sending `resume`, establishes:

`holder.Close returns` → `resume send/receive` → sleep hook returns → final `pebble.Open`.

That fixes the race present in the existing release-at-attempt-2 test, whose later-send drain supplies its synchronization at [open_lock_test.go:70](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/open_lock_test.go:70).

However, the source-property acceptance grep is impossible after `gofmt`; see the concern below.

### E. Phase invariants — HELD

- Probe ledger reconciles to 21: 15 edge-tagged truths, including the flat-scalar backstop, plus six flagged assumptions. The backstop is marked at [01-01-PLAN.md:48](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-01-PLAN.md:48).
- All 12 requirement IDs are covered.
- Wave graph is acyclic.
- Same-wave file ownership overlap is zero; `internal/graphstore/open_lock_test.go` belongs only to wave-7 plan 01-11: [01-11-PLAN.md:15](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-11-PLAN.md:15).
- Golden gates explicitly target `./testdata/golden/`, including the automated oracle at [01-02-PLAN.md:386](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-02-PLAN.md:386).
- Every plan contains both “Artifacts this phase produces” and `<threat_model>`.
- All 11 frontmatters parse under Ruby’s strict safe YAML loader.

### F. Context overrides — HELD

All three correction blocks remain authoritative:

1. Missing live byte oracle and Wave-1 remedy: [01-CONTEXT.md:98](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-CONTEXT.md:98), implemented by plan 01-02.
2. Exit-status/count conjunction: [01-CONTEXT.md:71](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-CONTEXT.md:71), reflected in all ordinary gates.
3. `toolslist-repeat` separability: [01-CONTEXT.md:259](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-CONTEXT.md:259), preserved by the separate test and explicit non-closure in plan 01-03.

## Concerns

### MEDIUM — H2’s property grep cannot match gofmt output

[01-11-PLAN.md:476](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-11-PLAN.md:476) searches for the literal:

```text
openLockRetryAttempts-1
```

Go formats binary expressions with spaces. Existing source demonstrates the canonical form at [open_lock_test.go:43](/Volumes/Code/github.com/seanb4t/codegraph-go/internal/graphstore/open_lock_test.go:43):

```go
openLockRetryAttempts - 1
```

Because comment lines are stripped, the plan’s unspaced prose occurrences cannot rescue the check. A correctly implemented and formatted `TestOpenSucceedsOnTheFinalAttempt` therefore produces zero matches and fails this acceptance criterion.

## Suggestions

Replace the H2 grep with a whitespace-tolerant expression:

```sh
rg -o -e 'openLockRetryAttempts[[:space:]]*-[[:space:]]*1'
```

Keep the same-file function-declaration positive control and the minimum count of two.

Optionally correct 01-06’s “eight test functions” wording to “seven test functions”; no gate change is needed.

## Risk Assessment

**MEDIUM.** The vacuity class is terminated and both HIGH fixes are architecturally sound. The remaining issue is narrow and mechanical, but it blocks a correct H2 implementation at acceptance time until the grep is fixed.

---

## Cycle 5 findings

### Verdict

The vacuity class is **not terminated**. Every shape-based sweep cycle 4 claimed holds under
independent re-derivation, and both cycle-3 HIGHs are fixed at the root rather than at the
symptom — that part worked. What did not is the part the census's own method could not see: a
gate is vacuous here not because the command is malformed but because a described behavior is
**absent** from everything the gate reaches. Three live members remain (**H3**, **M4**, **M5**),
plus a class of uncontrolled absence assertion the census did not enumerate (**L3**). See the
Addendum, which was written after a second, deeper pass and which corrects two rows of the census
table below.

### HIGH — unresolved

**H3. `01-10` Task 2's entire deliverable is guarded by nothing executable: the gate's `-run`
pattern names a test that is declared nowhere, and its floor is already cleared by tests
belonging to another plan.**

Four facts, each independently checked:

1. The gate at `01-10:423` runs
   `-run 'TestUIServiceSourceBlob|TestUIServiceNodeDetail|TestUIServiceExplore'`.
   `rg -n 'TestUIServiceSourceBlob' 01-10-PLAN.md` returns exactly three hits — `:423` (the
   gate), `:426` (the acceptance bullet restating the gate command) and `:427` (the derivation
   prose, "this task's new `TestUIServiceSourceBlob*` cases"). **The name is declared nowhere**:
   no artifacts entry, no behavior bullet, no `<files>` test file.
   *Positive control, identical pipeline against the same file class:* the only two other `-run`
   atoms in the phase that a strict-declaration check flags both resolve —
   `TestOriginHostGuard` at `01-01:126` ("Tests: `internal/uiserver/originguard_test.go`
   (`TestOriginHostGuard`, …)") and
   `TestPipelinedToolsListResponsesAreMatchedByIDWithoutNotifications` at `01-03:99`. The zero
   for `TestUIServiceSourceBlob` is a real absence, not a broken query.
2. Task 2's `<behavior>` block (`01-10:365-371`) has five bullets and **none names a test
   function**. Every other task in the phase names its test functions in the behavior block or in
   the plan's artifacts section; this is the only one that does not.
3. Task 2's `<files>` (`01-10:354`) is `internal/uiproto/uiv1/ui.proto, …ui.pb.go,
   …ui.connect.go, internal/uiserver/handlers.go, internal/uiserver/readonly_test.go,
   internal/uiserver/server.go`. Its only test file is `readonly_test.go`, which `01-10:106`
   scopes to the field-number fixture extension. **There is no file in which the five behaviors
   could be written.**
4. The floor is 14, and the plan's own derivation at `01-10:427` states the reason it cannot
   bind: *"This `-run` pattern re-runs plan 01-09's `TestUIServiceNodeDetail*` (10 lines by
   01-09's own derivation) and `TestUIServiceExplore*` (6 lines) … The inherited sixteen alone
   clear the floor; the new blob tests are headroom."*

Taken together: an executor can complete `01-10` Task 2 without writing a single end-to-end
`SourceBlob` test and **both halves of the D-04 conjunction still pass** — `STATUS -eq 0` from
the inherited 01-09 tests, and `COUNT` 16 ≥ 14 from the same. `TestUIServiceSourceBlob` matching
nothing is precisely the "`-run` pattern matches nothing" failure the phase's own canon says the
count floor exists to close; here the count floor does not close it, because it is satisfied by
*another plan's* tests sharing the pattern.

Nothing downstream rescues it. `01-10` has exactly two gates — `:331` (Task 1's truncation
tests) and `:423` — so no sibling gate covers Task 2. The phase's only whole-package sweep,
`01-11:550` (`go test -race ./internal/uiserver/`, floor 60 against a derived ≥111), is loose by
design and would not notice five missing tests either.

This is a counterexample both to the "13 orphaned behavior bullets across 6 tasks, all adopted"
census and to the class-termination claim. The plan frames the unwritten tests as "headroom",
which is why the census did not see it: the gate is well-formed, `sh -n`-clean, status-honoring
and positively controlled. It is vacuous for a reason no shape-based sweep detects — the work it
scores is not in its scope.

Fix (surgical, no re-plan): name the test functions in Task 2's behavior block, add the test file
to `<files>`, and give Task 2 a gate leg scoped to those tests alone with its own derivation —
the way `01-11:463` already conjoins a `graphstore` leg to a `uiserver` leg.

### MEDIUM — unresolved

**M1. `01-03:291` is an unranged `git diff` assertion evaluated at verification time — the one
member the "the other 11 uses are applied-confirmations" split missed.**

The distinction is real for five of the six `git diff` uses inside `<acceptance_criteria>`.
`01-02:396`, `01-05:414`, `01-06:296` and `01-07:301` all record a `git diff --stat` **captured
mid-task while the mutation is still uncommitted**, where non-empty genuinely is the evidence;
`01-06:396` is the one already fixed to a revision range with a same-range positive control.
`01-03:291` is neither:

> `git diff --stat` for this task touches exactly `internal/mcp/server.go` and
> `internal/mcp/pending_writer_test.go`. The `Server` interface, `BuildServer` and
> `internal/cli/serve.go` are untouched.

It sits inside `<acceptance_criteria>` (`01-03:280-293`), so it is evaluated after the task's
atomic commit. Its own plan's canon bullet 5 (`01-03:170-171`) states exactly why that fails:
*"An unranged `git diff` is empty by construction at verification time under
atomic-commit-per-task. Assert over `<phase-base>..HEAD`."* The bullet's first half then fails
for a bookkeeping reason; its second half ("…are untouched") **passes vacuously**, which is the
banned emptiness shape.

Fix: adopt `01-06:396`'s form verbatim — a `merge-base`-derived `PB` with
`git diff --stat "$PB"..HEAD -- <paths>` and its positive control.

**M2. `01-11:476`'s H2 property grep is gofmt-form-dependent, and the same file's existing idiom
is the non-matching form.**

The criterion requires
`rg -v '^\s*//' internal/graphstore/open_lock_test.go | rg -o -e 'openLockRetryAttempts-1' | wc -l`
to print at least `2`. Codex reports this is impossible after `gofmt`; that is **overstated**,
and the corrected reading matters. Measured directly against `gofmt`:

```go
if ord == openLockRetryAttempts-1 {             // gofmt KEEPS this unspaced
t.Fatalf("...", ord, openLockRetryAttempts-1)   // gofmt SPACES this
want := openLockRetryAttempts - 1               // gofmt SPACES this
```

`gofmt` elides the spaces only when the subtraction is an operand of a lower-precedence operator
(or an index expression). So the grep is satisfiable — but only if **both** cited sites happen to
be direct comparison operands, and the plan never says so. The nearest in-file precedent is the
spaced form: `internal/graphstore/open_lock_test.go:43` reads
`if want := (openLockRetryAttempts - 1) * openLockRetryBackoff;`. That file is in this task's
`read_first`, so an executor following its style writes a **correct** test against a **red**
gate — and the pressure is then to edit the test to satisfy the grep.

The failure direction is safe (false-FAIL, not vacuity), but it stalls execution on the exact
criterion cycle 4 wrote to close H2. Fix: make the pattern whitespace-tolerant,
`'openLockRetryAttempts[[:space:]]*-[[:space:]]*1'`, keeping the same-file
`func TestOpenSucceedsOnTheFinalAttempt` positive control and the floor of 2.

**M3. `01-11:463`'s first floor carries no derivation — the one exception to "all 26 floors now
carry derivations".**

`01-11:463` is a compound gate with two floors: `GCOUNT -ge 1` for
`TestOpenSucceedsOnTheFinalAttempt`, and `COUNT -ge 10` for the five degrade tests. Only the
second has a **Floor derivation** bullet (`01-11:467`); the floor of 1 is stated bare at
`01-11:475`. The correct total is 1 (one test function, no subtests), so the floor is trivially
satisfiable — and unlike the phase's two other floor-1 gates (`01-02:260`, `01-03:342`), which
both say so explicitly and note that `STATUS -eq 0` is what actually carries them, this one does
not. Fix: one bullet.

### LOW — unresolved

**L1. `01-01:289`'s derivation double-counts one behavior bullet.** It credits
`TestOriginHostGuard` with 20 `t.Run` rows *and* adds `TestOriginHostGuardAdmitsOriginlessGET`
separately — but behavior bullet 19 ("No Origin header at all on a GET with an admitted Host →
200") **is** that separate function, named as such at `01-01:126`. True total is 21, not the
stated 22. The floor of 19 is still met with two lines of headroom, so there is no gate
consequence — but the arithmetic is off by one in a derivation whose purpose is to be checked.

**L2. `01-06:388`'s derivation says "Eight test functions match" and then enumerates seven**
(`TestResolveHeadCommitSHA`, `…OnNonGitTree`, `…WithNoGitBinary`,
`TestHeadIsResolvedOncePerOperation`, `TestIndexMetaCarriesTheStoredMeta`,
`TestIndexMetaOnAGraphWithNoMeta`, `TestStatusResultFieldSetIsUnchanged`). The total of 14
(8 + 6×1) is correct **for seven functions**; only the count word is wrong. Independently found
by Codex. One-word fix.

### Noted, not actionable

- **`01-05:336`'s floor of 10 is an exact fit against a lower-bound derivation.** The derivation
  reads "at least eight" subtests + parent + 1, and its enumerated site list contains exactly
  eight entries, one of which (the `Query`/`Search` empty-term case) is a compound that could be
  written as one subtest or two. Eight subtests give exactly 10 against a floor of 10 — zero
  headroom on a floor phrased as a bound rather than a count. Not wrong; fragile.
- **`01-07:236`'s derivation lives in `<verification>` (`01-07:366`), not in the task's
  `<acceptance_criteria>`** where the other 25 sit; the acceptance bullet at `:238` states the
  floor bare. Content is correct (4 named functions, no subtests, zero headroom).
- **Toolchain observation.** `gsd-tools query progress` reported `"plans": 11` /
  `"total_plans": 11` cleanly for this phase throughout. It counts `*-PLAN.md` files and does not
  parse their YAML frontmatter, so it reported 11/11 during the window in which cycle 4's new
  prohibition entries carried trailing-comma YAML defects. A file count is not a health check;
  nothing in the GSD surface would have surfaced that breakage. Worth an upstream note.

### Census results — what HELD

| Sweep | Claim | Verified |
|---|---|---|
| `<automated>` gate count | 27 | **27** (`</automated>` closers; the 57 `<automated>` occurrences include 30 prose mentions) |
| `sh -n` cleanliness | 27/27 | **27/27**, zero syntax failures |
| `OUT`/`STATUS` captured before any pipe | all `go test` gates | **26/26** — no gate invokes `go test` without `STATUS=$?` |
| unguarded `git status --porcelain` in `<automated>` | 9 sites, 9 fixed, exactly that set | **9/9 guarded**, and every `test -d X` guard matches the pathspec `X/` it protects |
| count regexes accepting zero | 2, both in 01-07 | **both numericized** (`NFILES=…; [ "$NFILES" -ge 3 ]`) at `01-07:236` and `01-07:298` |
| gates with no control | 1 (01-10 numeric-literal) | **claim stale in both directions.** Its one named instance is genuinely fixed — `01-10:343-345` is now path-scoped, `*.pb.go`/`*.connect.go`-excluded and positively controlled, with the reason each part is load-bearing written out — so it is no longer an instance. But three uncontrolled absence assertions are absent from the claim's list: see **L3** |
| zero-count assertions | all controlled | **8/8 executable `rg … | wc -l` → `0` gates carry same-file positive controls** (`01-01:293`, `01-02:266-267`, `01-04:320-321`, `01-05:267`, `01-10:343`, `01-10:346`, `01-11:472`). This row covers only the executable ones; three absence claims written as bare prose carry no pipeline at all — see **L3** |
| unranged `git diff` emptiness | exactly 1, now ranged | **the split is real for 5 of 6 — see M1** |
| PASS floors carrying derivations | 26 of 26 | **25 of 26 — see M3** |
| floor arithmetic | correct | **24 of 26 exactly correct; 2 off in prose only (L1, L2), no floor affected** |
| five vacuity prohibitions | in all 11 | **11/11**, descriptor-less scalars in `must_haves.prohibitions`, naming the shape rather than the symptom |
| prohibition text vs absence-greps | must not self-trip | **safe** — the prohibition prose lives only in plan files; every absence-gate strips `^\s*//` and scopes itself to source paths |
| `## Gate conventions (phase-wide canon)` | byte-identical in 11 | **11/11 identical**, `sha1 5cd30a03d939` |
| echo labels | 6 variants normalized to `PASS lines:` | **26/26 PASS echoes use `PASS lines:`**; the only 2 other echoes are the drift gates' `compared=` |
| trailing-comma YAML defect | fixed pre-commit, never landed | **confirmed** — 0 present in any frontmatter; `git show e81dc32` deletes 0 lines ending in a quoted comma |
| frontmatter strict parse | 11/11 | **11/11 clean under `yq -e`** |
| orphaned behavior bullets | 13 across 6 tasks, all adopted | **three live members remain** — `01-05:198` (**M4**), `01-06:319` (**M5**) and `01-10` Task 2's five bullets (**H3**). The adoption work itself is real and well done in `01-03`, `01-04`, `01-08` and `01-09`, whose derivations each name the home for every unnamed bullet; it was simply not applied to all 13 tasks that have unnamed bullets |
| duplicate declarations in `behavioral_test.go` | 7 | **confirmed in source** — see below |
| duplicate test-function declarations | — | **0 collisions** — no (package, test-function-name) pair is declared by two different tasks anywhere in the phase |
| `read_first` citation errors | 8 of 85, corrected | **spot-check clean** — see below |
| threat model IDs | — | `T-01-01`..`T-01-39`, contiguous, no gaps; the eight IDs appearing in more than one plan are the same threat restated per surface (identical category, severity and disposition), not collisions |

The seven duplicates are real and `01-02` scopes them correctly. Verified by reading source:
`languageToLockedSlug`, `slugToRepo`, `goldenCapture`, `callExploreViaMCP`, `mcpResultText` and
the two `callNodeViaMCP*` variants are each declared in **both**
`testdata/golden/behavioral_test.go` and `testdata/golden/gocapture/main.go` — seven declarations
across six names. `01-02`'s cited line numbers are exact:
`behavioral_test.go:43` = `var languageToLockedSlug`, `:52` = `var slugToRepo`,
`:419` = `type goldenCapture struct`, and `gocapture/main.go:36` = `type goldenCapture struct`.
Positive control: `rg -c '^func Test' testdata/golden/behavioral_test.go` = 8, so the pipeline
that produced the zeros above works against that file.

`read_first` spot-check, line-ranged citations resolved by reading the cited range:
`internal/query/render_markdown.go:138-148` (`nodeMultiDefHardCap`) ✓;
`testdata/golden/golden_test.go:243-297` (`TestReFrozenGoldensValid`) ✓;
`testdata/golden/behavioral_test.go:685-691` ("byte-diffs") ✓;
`internal/mcp/server.go:200-350` (`waitForDrain`) ✓;
`internal/mcp/session_line_test.go:60-90` (`t.Run`) ✓;
`internal/query/explore.go` ~236 (`T-01-25`) ✓. The two `01-CONTEXT.md` D-04 citations are among
these, so the correction block's own evidence is sound.

### The two HIGHs — root fixes verified

**H1 — HELD, and the fix is genuinely bidirectional.** `01-09:457` requires the test to *"walk
the descriptor, collect every `message.field` it finds, and fail naming any that the fixture does
not cover"*, alongside `resolved == len(fixture)` and `len(fixture) > 0` at `:456`. Both
directions, stated as such: *"the coverage assertion runs in BOTH directions"*. The wave-5 scope
explicitly includes `GetStatusResponse` 1-7 and `IndexingInProgress` (`01-09:393`, `:457`) and
explicitly excludes 8/9 with the reason (`:458`). The baseline is compiler-carried, not
prose-carried: `uiProtoFieldFixtureLenAtPlan0109` → `+9` (`01-10:431`) → `+2` (`01-11:275`).
`TestUIProtoFieldNumbersAreStableAndUnique` is in `01-11:296`'s `-run` with floor 9, and the
derivation at `01-11:300` correctly nets the re-run test into the total of 10. The false statement
is corrected in place at `01-11:581` and again at `01-11:308`, both marked as corrections rather
than silently rewritten. Additionally checked: `01-11` declares no proto field beyond the two —
`IndexingInProgress.message` is *populated*, not newly declared (`01-11:104`) — so `+2` really
does keep the bidirectional walk satisfied at wave 7.

**H2 — the concurrency argument is correct.** Checked as a concurrency argument, against
`internal/graphstore/pebble_store.go:141-147`. The loop runs `openLockRetryAttempts` (5) attempts
with `openLockRetrySleep` called at the top of attempts 1-4, so there are exactly
`openLockRetryAttempts-1` = 4 sleeps and the fourth immediately precedes the final attempt — the
plan's ordinal (`01-11:409-410`) is 1-based and lands on it correctly. The happens-before chain
holds under the Go memory model: the helper receives ordinal 4, calls `holder.Close()` to
completion, *then* sends on the unbuffered `resume`; a send on an unbuffered channel happens
before the corresponding receive completes, so `Close` returns → `resume` send → `resume` receive
→ the seam returns → `pebble.Open`. No race, no wall clock. The plan is also right about why
`TestOpenConvergesWhenHolderCloses`'s drain loop does not generalise: that test's synchronisation
comes from a *later* sleep blocking (`internal/graphstore/open_lock_test.go:70-76`), and on the
final sleep there is no later sleep. The acceptance checks the property (`Open` returned nil
**and** the observed ordinal equals the final one), not merely that a cited name exists. The only
defect is the grep's source-form dependence — **M2**, which is about the criterion, not the
design.

### Invariants — all HELD

- **Probe ledger 21.** `rg ' edge\)'` returns 15 edge-tagged `must_haves` across 01-01, 01-02,
  01-04, 01-05, 01-10 and 01-11; `01-01:48`'s `verification: backstop` truth is one of them
  (tagged `RPC-01 / concurrency edge`), so it is inside the 15 rather than a 16th. Four plans
  carry a `## Flagged Assumptions` section (01-01, 01-03, 01-06, 01-07) totalling 6 open
  assumptions. 15 + 6 = 21, and `01-06:409-411` states that reconciliation in the same terms:
  *"the phase's probe ledger stays at 21 — 15 edge-tagged `must_haves` (including 01-01's
  `verification: backstop`) plus 6 flagged assumptions."* H2's fix touches `01-11`'s
  `concurrency edge` probe, which is still present and still reconciles.
- **12/12 requirement IDs.** The union of the 11 frontmatters' `requirements:` is set-equal to
  the ROADMAP line — `{BLD-04, ENG-01, ENG-02, ENG-04, FIX-01, RPC-01, RPC-02, RPC-05, SRV-01,
  SRV-02, SRV-03, SRV-04}` — no extras, no gaps.
- **Wave graph acyclic**, 1→7, every `depends_on` pointing to a strictly lower wave.
- **Zero same-wave file-ownership overlap** across all 87 `files_modified` entries.
  `internal/graphstore/open_lock_test.go` appears exactly once, in `01-11` (wave 7, sole
  occupant) — H2's new file introduces no contention.
- **Golden gates name their target explicitly**: every golden invocation carries
  `./testdata/golden/`, never `./...` (which `01-VALIDATION.md` notes would cover zero goldens).
- **`<threat_model>` and "Artifacts this phase produces" present in 11/11.**
- **Prohibitions are descriptor-less scalars** in `must_haves.prohibitions` in 11/11.

### The three `01-CONTEXT.md` correction blocks — all preserved

1. **D-04 / no live byte oracle** (`01-CONTEXT.md:98-112`) — intact, and discharged by `01-02`,
   whose `TestGoldensMatchLiveEngineOutput` gate (`01-02:386`, floor 30) is the missing oracle.
   `01-02:390` correctly separates the unanchored form (30, both tests) from `01-05:411`'s
   `$`-anchored form (27, the single test) — a distinction that would silently break both floors
   if collapsed.
2. **D-04 scoring conjunction** (`01-CONTEXT.md:71-89`) — preserved with its single scoped
   exception intact. `01-05:417` states it precisely: the RESTORED-tree gate requires
   `STATUS -eq 0`, and *"the deliberate-RED exception applies only to the mutation observations …
   scored by `--- FAIL`/`--- PASS` counts and never by status."* `01-05:432` additionally flags
   the residual D-04 reading question for the maintainer rather than resolving it silently —
   correctly, and since it is already in the plan it is not an open review item.
3. **`toolslist-repeat` separability** (`01-CONTEXT.md:259-272`) — preserved. `01-03` Task 2
   exists to make the disproof executable, and `01-03`'s prohibition list carries *"Never report
   FIX-01's closure as closing the `toolslist-repeat` …"*.

### Risk

**MEDIUM.** The mechanical half of the class-termination worked completely: every shape-based
sweep holds under independent re-derivation, both HIGHs are fixed at the root, and every invariant
is intact. The half that did not is the *absence* half — a behavior the plan describes that no
test, subtest or floor reaches. Three members are live: `01-10` Task 2's whole deliverable
(**H3**), `01-05:198`'s `maxFiles == 0` path (**M4**), and `01-06:319`'s ENG-04 end-to-end
stamping property (**M5**). Alongside those, three absence assertions carry no executable control
at all (**L3**), three criterion edits are needed (**M1**, **M2**, **M3**), and two derivations
have one-word arithmetic slips (**L1**, **L2**). Every one is surgical: no wave redesign, no
re-plan, no change to the architecture or to either HIGH fix.

The useful generalisation for the maintainer: the phase's prohibitions now cover every way a gate
can be *written* wrong, and none of the ways a gate can be *scoped* wrong. `01-10:423` satisfies
all five prohibitions and still scores nothing it owns. A sixth prohibition — *every gate's `-run`
pattern must name at least one test function the task itself declares, and its floor must be
derived from that task's own subtests* — would close the shape H3 exposes.

### Addendum — second-pass census results

A deeper sweep completed after the section above was first written and found **three more items**,
each verified independently against source. Two are further members of the orphaned-behavior class
that **H3** exposes; one is a class of absence-assertion the earlier "all controlled" row did not
cover. The two corrected census rows are marked in the table above.

**M4. `01-05:198` — the `maxFiles == 0` behavior is owned by no test.**

> `maxFiles == 0` behaves exactly as it does today (the multi-file capture path uses it, and the
> adaptive-budget override at H21 only fires when no explicit value was given).

The only candidate owner is `TestExploreResultMaxFilesBoundary`, but the floor derivation at
`01-05:261` pins that function to **exactly two** `t.Run` subtests, `exactly-at-match-count` and
`one-below`, and the acceptance bullet at `01-05:265` restates only those two. Zero is neither.
Nor is it a row of any other matrix the gate reaches: `TestExploreResultZeroMatchModeCoversEveryBranch`
has one subtest per zero-**result** branch, which is a different concept, and the remaining two
functions declare no subtests. The `must_haves` truth at `01-05:37` likewise covers only the
at-count boundary. Positive control: `MaxFilesZero` occurs **0** times in the file while
`TestExploreResultMaxFilesBoundary` occurs 3, so the zero is a real absence. This is not
bookkeeping — the bullet itself says the multi-file capture path uses `maxFiles == 0`, so a live
code path is described and then left unasserted. The plan nowhere reasons about omitting it.

**M5. `01-06:319` — ENG-04's end-to-end stamping property is owned by no test.**

> A `codegraph index` run over a git checkout produces a `Meta` whose commit field equals
> `git rev-parse HEAD` for that checkout; over a non-git directory it succeeds with an empty value.

It is the only plain-prose bullet among eight in that behavior block (`01-06:314-322`) that no
named function owns — and the contrast is instructive, because the *other* plain-prose bullet
there (`:315`, the 39/41/63/65/uppercase-hex lengths) **is** owned: the derivation names those five
explicitly as `t.Run` subtests of `TestResolveHeadCommitSHA`. All three ownership routes fail for
`:319`:

- The nearest named function, `TestHeadIsResolvedOncePerOperation`, asserts only the injected
  lookup's **invocation count** per `01-06:394` ("exactly 1 for a full index run and exactly 1 for
  a sync run"). Counting resolutions is not asserting the resolved value lands in `Meta`.
- The artifact lists (`01-06:103`, `:105`) name seven tests; none indexes a real git checkout.
  `TestIndexMetaCarriesTheStoredMeta` reads back whatever the store holds for a fixture, which does
  not pin the value to `git rev-parse HEAD`.
- The nearest acceptance criterion, `01-06:393`, is a **source grep** requiring exactly three
  assignments of the generated field across `internal/indexer/`. That proves assignment statements
  exist; it does not prove the resolved SHA reaches `Meta` — which is precisely the gap the bullet
  describes.

The gate at `01-06:384` would not match a remedial `TestIndexRunStampsHeadCommitSHA` either, so
the fix is a `-run` change as well as a test.

**L3. Three absence assertions carry no executable control.** Distinct from the eight
`rg … | wc -l` → `0` gates, which are all controlled: these are absence claims written as bare
prose with no pipeline at all, so nothing can fail them.

- `01-08:277` — "`mapEngineError` contains no string comparison against an error message;
  classification is `errors.Is`/`errors.As` only." Two lines above it, `01-08:275` is the fully
  controlled form on the same file, so the plan already knows the shape. The gate at `01-08:265`
  runs four `TestUIService*` functions, none of which inspects the implementation.
- `01-11:306`, **second clause only** — "…and no error classification in either file compares
  against an error message string." The first clause is executable with a floor and is fine; the
  second is welded onto the same bullet with no pipeline. Same shape and subject as the above.
- `01-01:317` — "…the response body does not contain a serialized Connect envelope." Uncontrolled,
  and additionally **dropped**: the acceptance criterion at `01-01:489` reduces the whole behavior
  to "receives HTTP 403 for `Host: evil.com`". Positive control: "Connect envelope" occurs exactly
  **1** time in the file (against `originHostGuard` at 5), confirming the phrase never reaches an
  acceptance criterion. The phase already uses the right pattern for this at `01-11:202`/`:305` —
  apply the same predicate to a known-good 200 response and assert it detects the envelope there.

Fix for all three: an executable pipeline over the guarded file plus a same-invocation control,
e.g. `rg -v '^\s*//' internal/uiserver/handlers.go | rg -o -e 'strings\.(Contains|HasPrefix|EqualFold)|err\.Error\(\) ==' | wc -l`
prints `0`, controlled by `errors\.(Is|As)` against that same file returning non-zero.

**What this does to the class verdict.** It sharpens it rather than changing it. The
orphaned-behavior class has **three** live members (`01-05:198`, `01-06:319`, and `01-10` Task 2's
five bullets), not one — and all three share H3's signature: the gate is well-formed and passes
every one of the five prohibitions, but nothing in it can fail when the described behavior is
absent. The sixth prohibition proposed at the end of the Risk section would catch `01-10` Task 2
directly; catching M4 and M5 additionally needs the rule that **every behavior bullet must name, or
be named by, a test function or `t.Run` subtest the task's own floor derivation counts.** That
rule is the generalisation of the work cycle 4 already did by hand in `01-03`, `01-04`, `01-08` and
`01-09`, where each derivation explicitly names the home for every unnamed bullet — it simply was
not applied to all thirteen tasks that have unnamed bullets.

## Verification coverage (cycle 5 source-grounding pass)

`.planning/config.json:67` sets `source_grounding_authority: "grep"`, and
`.planning/intel/API-SURFACE.md` states in its own header that `api-map.json` has no entries and
that absence there means "unknown", not "does not exist". The intel map was therefore **NOT
CONSULTED** — grading against it would have marked every Go symbol UNCHECKABLE → INFO and
hard-blocked nothing. Every symbol below was resolved by reading source.

### Method

Six independent passes over the 11 plans plus the repo tree:

1. **Gate extraction and mechanics.** All 27 `<automated>` gates extracted by matching
   `</automated>` closers (the 57 `<automated>` occurrences include 30 prose mentions), each run
   through `sh -n`, then checked for `STATUS=$?` capture before any pipe, `test -d` guards on
   every `git status --porcelain` pathspec, and numeric floors on every count regex.
2. **Floor re-derivation.** All 26 PASS floors re-derived independently from each plan's own
   `<behavior>` bullets under the phase counting convention, accounting for Go's `-run` being an
   unanchored regex and for test functions accumulating in a shared package across waves.
3. **`-run` atom resolution.** Every alternation atom in every gate checked against the set of
   test-function names the phase actually *declares* (behavior-bullet heads and artifact "Tests:"
   lines), with each zero positively controlled against atoms known to resolve.
4. **Invariant reconciliation.** Probe ledger, requirement-ID set equality, wave acyclicity,
   same-wave file-ownership overlap across all 87 `files_modified` entries, golden-target
   explicitness, `<threat_model>` / artifacts presence, prohibition uniformity, and strict YAML
   parse of all 11 frontmatters under `yq -e`.
5. **Concurrency review of H2.** The acknowledgement-channel design checked as a concurrency
   argument against `internal/graphstore/pebble_store.go:141-147` and
   `internal/graphstore/open_lock_test.go:60-90`, and `gofmt`'s actual spacing behaviour measured
   rather than assumed.
6. **Symbol source-grounding**, with each plan's "Artifacts this phase produces" manifest
   excluded.

### Symbols resolved by reading source

Confirmed present, at the cited package: `graphstore.Open` (`internal/graphstore/pebble_store.go:141`),
`graphstore.ErrStoreLocked`, `graphstore.ErrNotFound`, `classifyOpenError`, `openLockRetrySleep`,
`openLockRetryAttempts` / `openLockRetryBackoff` (`pebble_store.go:79-80`),
`TestOpenConvergesWhenHolderCloses` (`open_lock_test.go:60`), `query.OpenAt`,
`query.ErrNotInitialized` (`internal/query/resolve.go:18`), `query.StatusResult`, `Engine.Status`,
`schema.Meta`, `pendingWriter` and `pendingWriter.Write`, `stdinLingerReader`, `stdinLingerGrace`,
`waitForDrain`, `sniffedMessage`, `looksLikeJSONRPCCall`, `TestReFrozenGoldensValid`,
`TestCorpusBehavior_Go`.

### Exclusions (phase artifacts — absence today is correct)

Four cited symbols resolve to nothing on `main` and were verified to be **phase artifacts**, each
traced to the plan that creates it, so none is a grounding failure:

| Symbol | Created by |
|---|---|
| `query.ErrInvalidArgument` (and `internal/query/errors.go`) | `01-05:302` — "Create `internal/query/errors.go` with two exported sentinels"; cited downstream by `01-08:183` and `01-09:174` as an earlier-wave artifact |
| `goldenspec.LockedCorpusArgs` | `01-02:92` — `internal/goldenspec/spec.go` |
| `goldenspec.CallNodeViaMCP` | `01-02:93` — `internal/goldenspec/mcp.go` |
| `gitExecLookPath` | `01-06` — `internal/indexer/commit.go` |

Zero genuine grounding failures.

### Traps positive-controlled

- **`rg` exits non-zero for both "no match" and "file not found."** Every zero reported above was
  positive-controlled by running the identical pipeline with a pattern the same target genuinely
  contains — this is what established that `TestUIServiceSourceBlob`'s zero (**H3**) is a real
  absence while `TestOriginHostGuard`'s and
  `TestPipelinedToolsListResponsesAreMatchedByIDWithoutNotifications`'s apparent zeroes were
  artifacts of a too-strict declaration regex.
- **`.proto` snake_case vs generated Go camelCase is not drift** — `store_exists` /
  `indexing_in_progress` were read as proto field names throughout and never compared against
  generated Go identifiers.
- **`gofmt` spacing was measured, not assumed** (**M2**): `a-1` survives unspaced as an operand of
  a lower-precedence operator or inside an index, and is spaced everywhere else. Codex's claim
  that the H2 grep is *impossible* is therefore wrong; the correct finding is that it is
  *form-dependent*, which changes the fix.

---

# Cross-AI Plan Review — Phase 1 (cycle 7, SEMANTIC class-termination verification)

Reviewers: codex (`gpt-5.6-sol`, reasoning=low), plus an independent orchestrator verification
pass. Trajectory: 40 → 8 → 16 → (replan) → 9 → (replan) → **cycle 7**.

Cycle 6 was tasked with terminating the SCOPE class with a rule rather than another instance
sweep. This cycle tests that claim adversarially: whether the instance fix in `01-10` is real,
whether the new semantic prohibition actually binds, whether prose-homing reintroduces H3's
mechanism, and whether fixing scope regressed shape.

## Codex Review

## Summary

The specific `01-10` SourceBlob fix is sound: its task-owned leg has no inherited matches, its seven-line floor is exact, and all 27 automated gates are shell-syntax clean. However, the scope-vacuity class is **not fully terminated**. The new semantic prohibition’s OWNERSHIP and HOMING rules can both be satisfied by a trivial task-owned test unrelated to the deliverable while behavior bullets are “homed” to prose-only criteria. Thus, the general rule remains under-specified even though the known instance is fixed correctly.

## Strengths

- The SourceBlob gate is genuinely isolated. The five matching functions are declared only by plan 01-10 in [01-10-PLAN.md:113](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-10-PLAN.md:113) and detailed at [01-10-PLAN.md:381](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-10-PLAN.md:381). A repository-wide Go-source search found no existing `func TestUIServiceSourceBlob...`; searches across the other ten plans found no competing declaration.

- Leg 1’s arithmetic is correct. Four ordinary test functions contribute four lines, while `TestUIServiceSourceBlobOnEveryAttachmentPoint` contributes its parent plus two subtests, totaling three. Therefore `4 + 3 = 7`, exactly matching the zero-headroom floor at [01-10-PLAN.md:446](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-10-PLAN.md:446) and its derivation at [01-10-PLAN.md:451](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-10-PLAN.md:451). Omitting or renaming any declared function makes the leg fail.

- The inherited regression leg is honestly separated and explicitly disclaims ownership of the new deliverable at [01-10-PLAN.md:453](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-10-PLAN.md:453). It can no longer subsidize Leg 1.

- I extracted all 27 `<automated>` commands and checked each with `sh -n`: 27/27 passed. The nine actual `git status --porcelain` gates all retain preceding `test -d` guards; examples include [01-05-PLAN.md:424](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-05-PLAN.md:424), [01-07-PLAN.md:309](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-07-PLAN.md:309), and [01-10-PLAN.md:446](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-10-PLAN.md:446).

## Concerns

### HIGH — The semantic prohibition’s two operational halves still admit a scope-vacuous gate

The prohibition says:

> “(a) OWNERSHIP — every gate carries at least one leg whose test-selection pattern matches ONLY test functions this task itself declares… (b) HOMING — every behavior bullet is named by… a test function… or… covered by a specific named criterion…”

This appears verbatim at [01-10-PLAN.md:59](/Volumes/Code/github.com/seanb4t/codegraph-go/.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-10-PLAN.md:59) and all eleven plans.

A counterexample satisfying both halves literally:

```sh
OUT=$(go test -v -run '^TestTaskGateExists$' ./internal/example 2>&1)
STATUS=$?
COUNT=$(printf '%s\n' "$OUT" | rg -o -e '--- PASS' | wc -l)
[ "$STATUS" -eq 0 ] && [ "$COUNT" -ge 1 ]
```

The task declares:

```go
func TestTaskGateExists(t *testing.T) {
    if 1+1 != 2 {
        t.Fatal("impossible")
    }
}
```

- OWNERSHIP passes: the pattern matches only a test declared by this task, and its floor derives solely from that test.
- HOMING passes literally if each behavior bullet says it is covered by a named acceptance criterion such as `AC-SourceBlob-Bounded`.
- The implementation, schema, and actual SourceBlob behavior can all be absent while the gate remains green.

The opening sentence forbids this outcome, but the two stated “mandatory and reader-checkable” tests do not operationalize that sentence. In particular, HOMING permits a “specific named criterion” without requiring that criterion to be executable or that its evidence artifact be machine-checked.

This is not a defect in the repaired SourceBlob gate itself; it means the claimed phase-wide termination rule remains incomplete.

## Suggestions

Change the semantic rule so each task-owned leg must have a demonstrated causal connection to the deliverable:

- Every counted test must reference or exercise at least one artifact or behavior owned by the task.
- Removing the named deliverable—or replacing it with a no-op—must make that leg fail.
- A behavior may be homed to a criterion only when that criterion is executable, or when it requires a named evidence artifact whose existence and relevant content are checked.
- Trivial “gate exists” tests must not satisfy OWNERSHIP.

For example: “OWNERSHIP requires task-owned tests whose assertions depend on the named deliverable, with the dependency stated in the floor derivation.”

## Risk Assessment

**MEDIUM.** The concrete cycle-6 SourceBlob defect is fixed correctly, and the 27 automated gates retain their established shell and exit-status protections. Risk remains because the new general prohibition can certify future scope-vacuous gates despite satisfying both advertised halves. The scope class is therefore reduced, but not terminated.

---

## Cycle 7 findings

Every check below was run by the orchestrator independently of Codex. Where a check asserts an
absence, the positive control uses the **identical invocation shape** on a path known to match.

### Verified clean

- **H3's fix is real, not cosmetic** (`01-10:446` leg 1). The atom `TestUIServiceSourceBlob`
  appears in exactly two plan files: `01-10-PLAN.md` (11 occurrences — the pattern, the five
  function names, the derivation) and `01-11-PLAN.md` (1 occurrence, a floor derivation that
  credits 01-10's tests inside a package sweep). **No other plan declares a matching test, and
  01-09 declares none.** `rg -n 'TestUIServiceSourceBlob' --glob '*.go' .` returns nothing
  (exit 1), positive-controlled by `rg -c 'func Test' --glob '*.go' .` on the same shape
  returning hits; `internal/uiserver` does not exist in the tree yet. Leg 1's floor is therefore
  unreachable without the deliverable.
- **Leg 1's arithmetic is exact.** Four single functions = 4 `--- PASS` lines;
  `TestUIServiceSourceBlobOnEveryAttachmentPoint` = 1 parent + 2 subtests = 3. Total **7**
  against `-ge 7`. Zero headroom in the direction that matters: dropping, renaming or omitting
  any one of the five turns the leg red. A `-run` pattern matching nothing yields 0 and fails
  the floor even though `go test` exits 0.
- **The sixth prohibition is present verbatim in 11/11 plans.**
- **Shape did not regress.** All 27 `<automated>` gates were extracted and run through `sh -n`:
  **27/27 clean**, positive-controlled against a deliberately unbalanced gate which `sh -n`
  correctly rejects. All 26 gates that invoke `go test` carry an explicit `STATUS -eq 0`
  conjunct (no gate has fewer than two `-eq 0` clauses); **zero** gates use `-ne 0`, which is
  consistent with D-04's one exception living in recorded mutation observations rather than in
  an automated gate.
- **9/9 `git status --porcelain` uses are guarded.** Both multi-pathspec cases guard *every*
  pathspec, not just the first: `01-07:309` carries `test -d internal/schema && test -d
  internal/uiproto &&`, and `01-08:276` carries `test -d internal/uiproto && test -d
  internal/schema &&`.
- **`-run` atom census reconciles.** Every atom across the 25 `-run`-bearing gates resolves to a
  declaring plan — **0 unresolved atoms**. Exactly **3** legs span plans, matching cycle 6's
  census: `01-05:424` (01-02's `TestGoldensMatchLiveEngineOutput$`), `01-10:446` leg 2 (01-09's
  `TestUIServiceNodeDetail|TestUIServiceExplore`), and `01-11:307` (01-09's
  `TestUIProtoFieldNumbersAreStableAndUnique` alongside four own atoms). Each carries a written
  justification.
- **M5 is load-bearing** (`01-06:401`). Floor raised 9 → 15 against a derived total of 17
  (16 if the `--object-format=sha256` subtest skips). Without `TestIndexRunStampsHeadCommitSHA`
  the total is 14 — below the floor, so the gate goes red. At the old floor of 9 the new test
  would have been pure headroom, which is exactly the shape the new prohibition forbids.
- **M1 is fixed** (`01-03:302-303`): `PB=$(git merge-base HEAD main); git diff -U0 "$PB"..HEAD`
  with declaration-anchored patterns. The remaining unranged `git diff --stat` occurrences
  (`01-02:372`, `01-05:387`, `01-07:285` and their SUMMARY criteria) are *presence*
  confirmations that a deliberate mutation was applied — the opposite of the emptiness assertion
  the prohibition bans, and a correct use.
- **L3's three absence assertions each carry an executable same-invocation control**
  (`01-01:305`, `01-05:279`, `01-10:361`), as do the additional pairs at `01-02:278-279` and
  `01-04:331-332`.
- **Self-invalidation holds.** The 27 gates contain only three distinct `rg -o -e` patterns:
  `'--- PASS'`, `'[0-9]+'` and `'compared [0-9]+ generated files'`. None is an absence gate and
  none reads `.planning/` — every comment-stripped grep in the phase targets Go source under
  `internal/`/`testdata/` or `task proto:drift` output.
- **Golden verification targets `./testdata/golden/` explicitly** in both gates that run it
  (`01-02:398`, `01-05:424`), never `./...` — which matters, because the go tool ignores
  `testdata` directories (see M-7-2 below).
- **Invariants.** `<threat_model>` and `## Artifacts this phase produces` in 11/11.
  Frontmatter strict-parses in 11/11 under Ruby `YAML.safe_load` with `Date`/`Time` permitted,
  positive-controlled against deliberately malformed YAML (`Psych::SyntaxError`). 88
  `files_modified` entries; **zero same-wave file-ownership overlap** (wave 6 has 01-10 as its
  sole occupant, so the new `sourceblob_test.go` collides with nothing). Waves 1→7 acyclic —
  every `depends_on` points to a strictly lower wave. 12/12 requirement IDs
  (BLD-04, ENG-01, ENG-02, ENG-04, FIX-01, RPC-01, RPC-02, RPC-05, SRV-01..04). Probe ledger
  reconciles at **21**: 15 `edge)`-tagged truths + 6 flagged assumptions
  (01-01 ×3, 01-03, 01-06, 01-07), with exactly one flat-scalar `verification:` key in the
  entire phase (`01-01:49`, the BLD-04 backstop).

### H-7-1 (HIGH) — the semantic prohibition's two halves do not operationalize its own headline

The prohibition opens with the correct rule — *"Never write a gate whose floor can be met
without the deliverable it names existing"* — then offers two "mandatory and both checkable"
halves. Neither half, as written, entails the headline.

**(a) OWNERSHIP** constrains *who declares the counted tests*, not *what those tests assert*. It
is satisfied by any leg whose pattern matches only tests the task declares, regardless of
whether those tests touch the deliverable.

**(b) HOMING** offers an explicit escape: a bullet may be homed by "the derivation states in
writing that the bullet is covered by a **specific named criterion**". Nothing requires that
criterion to be executable, or to appear in any `<automated>` gate.

Counterexample satisfying both halves literally while remaining scope-vacuous — a task declares
one trivial own test:

```go
func TestTaskGateExists(t *testing.T) { if 1+1 != 2 { t.Fatal("impossible") } }
```

```sh
OUT=$(go test -v -count=1 -run '^TestTaskGateExists$' ./internal/example 2>&1); STATUS=$?
COUNT=$(printf '%s\n' "$OUT" | rg -o -e '--- PASS' | wc -l | tr -d ' ')
[ "$STATUS" -eq 0 ] && [ "$COUNT" -ge 1 ]
```

OWNERSHIP passes (the pattern matches only a test this task declares; the floor derives from
that leg alone). HOMING passes (every behavior bullet is declared "covered by criterion
AC-Foo-Bounded"). The production deliverable, the schema change and the asserted behavior can
all be absent and the gate is green. Codex reached this counterexample independently.

This is not a defect in the repaired `01-10` gate — that one is correct — but the *general* rule
the phase now carries does not close the class it claims to close. The missing clause is a
**causal** one: the counted tests' assertions must depend on the named deliverable, such that
replacing the deliverable with a no-op turns the leg red; and a bullet may be homed to a
criterion only when that criterion is itself executed by an `<automated>` gate.

**Suggested amendment** to half (a), as one sentence: *"…and the floor derivation must state
which named artifact of this task each counted test depends on, such that replacing that
artifact with a no-op turns the leg red; a test that would still pass with the task's artifacts
absent does not satisfy OWNERSHIP."* And to half (b): *"…a named criterion qualifies as a home
only if that criterion's command appears in this task's `<automated>` gate."*

### M-7-1 (MEDIUM) — `01-05:424` has no own-only leg, which half (a) does not actually permit

Half (a) requires that **every** gate carry at least one leg matching only tests the task itself
declares. `01-05`'s third gate has no such leg: the task declares no test functions at all, and
its single `-run` atom is plan 01-02's `TestGoldensMatchLiveEngineOutput$`. The plan is candid
about this (`01-05:428` — *"This task declares no test of its own"*) and invokes half (a)'s
exception clause, but that clause covers a pattern that **also** matches inherited tests, not
one that matches **only** inherited tests. There is no exception for a task with no tests.

The practical consequence is real: on an untouched tree, `test -d internal/query && porcelain
clean && goldens pass` is already true, so the executable gate is green before the task starts.
The evidence that discriminates — non-zero `--- FAIL` counts from the two deliberate mutations —
lives only in `<acceptance_criteria>` prose destined for the SUMMARY.

**Change needed in PLAN.md:** either widen half (a) to name the sanctioned non-test evidence form
for a mutation-observation task (recorded `--- FAIL` counts with a stated zero-is-a-stop-condition
rule), or give `01-05` Task 3 an executable leg that cannot pass before the mutations were
performed. As written, the phase's own sixth prohibition is violated by one of its own gates.

### M-7-2 (MEDIUM) — `01-02`'s bullet 3 is homed to a criterion no gate executes, and `go build ./...` cannot cover it

`01-02:276` homes Task 1's third behavior bullet (*"`gocapture` still builds and runs after the
move"*) to the acceptance criterion at `01-02:284`: *"`go run ./testdata/golden/gocapture`
builds and its usage/entry path executes without a compile error."* That criterion appears in no
`<automated>` gate — `rg -l 'gocapture' ` over all 27 extracted gates returns nothing,
positive-controlled by `rg -c 'go build'` over the same directory returning hits.

The gate's `go build ./...` does not cover it either: **the go tool ignores directories named
`testdata` when expanding `...`**. Verified on this repo, not assumed —
`go list ./... | rg 'testdata'` exits 1 while the identical-shape control
`go list ./... | rg 'internal/query'` returns `github.com/seanb4t/codegraph-go/internal/query`,
and `go list ./testdata/golden/...` does list `.../testdata/golden/gocapture`.

So the bullet's claimed "executable owner" is executed by nothing, which is H-7-1's prose-homing
escape realized in-plan rather than hypothetically. The `01-02` case is benign in effect (a
compile break would surface at Task 2's `go test ./testdata/golden/`), but it is the exact
mechanism H3 was.

**Change needed in PLAN.md:** add `go build ./testdata/golden/gocapture/` to the `01-02:271`
gate, ahead of the existing `go vet`, so the bullet's home is executed by the gate that scores
the task.

### L-7-1 (LOW, not actionable) — homing-label bookkeeping

The literal label `**Behavior-bullet homes.**` appears 20 times against 21 `<behavior>` blocks.
The gap is `01-08` Task 3, which carries its homing and its ownership statement **inline inside
the floor derivation** (`01-08:344`: *"…which are the behavior block's remaining three bullets,
so none is orphaned — bullet 1 is the four per-rpc subtests… **Ownership:** the single atom
matches exactly one test function and THIS task declares it"*). It is substantively compliant;
only a mechanical label-counter would see a gap. `01-07` correctly has none — it has zero
`<behavior>` blocks. No plan change needed.

### Source-grounding pass

Codex's review cites `file:line` evidence throughout and is not marked
`[reviewed-without-repo-access]` or `[reviewed-without-source-citations]`.

### Verification coverage

`drift-guard authority` resolves to `intel`, but `.planning/intel/API-SURFACE.md` reports
`symbolCount: 0` on this Go repository — the extractor is regex/JS-only, so every symbol would
grade UNCHECKABLE → INFO and nothing would hard-block. **The empty intel map was therefore not
treated as evidence of absence.** Every citation below was resolved by reading source.

| Class | Count | Method | Result |
|---|---|---|---|
| Files cited in `<read_first>` blocks | 45 | filesystem existence | 32 present; 13 absent, **all 13 declared under a plan's "Artifacts this phase produces"** (`internal/uiserver/*`, `internal/uiproto/*`, `internal/goldenspec/spec.go`, `internal/query/detail.go`, `internal/query/errors.go`, `internal/schema/meta_commit_test.go`, `internal/jsonrpc2/conn.go`) — **0 unexplained** |
| Go identifiers cited in `<read_first>` | 165 | `rg --glob '*.go'`, positive-controlled on `OpenAt` | 130 resolve directly in-tree; 35 did not, and all 35 classify as phase-produced or external (below) — **0 unresolved** |
| External: `connect.WithReadMaxBytes`, `WithSendMaxBytes`, `CodeUnavailable` | 3 | module cache `connectrpc.com/connect@v1.20.0` | `option.go:257`, `option.go:269`, `code.go:96` — all present. `connectrpc.com/connect` is not yet in `go.mod`; plan 01-01 adds it and `01-01:511` requires the verified signatures be recorded in its SUMMARY |
| External: `http.Protocols`, `(*Protocols).SetUnencryptedHTTP2` | 2 | Go 1.26.7 stdlib | `net/http/http.go:30` and `net/http/http.go:56` — present, so the h2c mechanism needs no `x/net` dependency |
| Declared test-function names across all 11 plans | 87 | cross-plan ownership map | every `-run` atom in every gate maps to a declaring plan — **0 unresolved atoms** |
| Repo traps avoided | — | — | `.proto` snake_case vs generated Go camelCase never treated as drift; every zero-result assertion above carries a control using the **identical** invocation shape on a path known to match |

## Consensus Summary

Both the orchestrator's independent pass and Codex reach the same verdict, and both reach the
same single HIGH by independent construction.

### Agreed Strengths

- `01-10:446` leg 1 is genuinely zero-headroom and genuinely own-only. H3 is closed at the root:
  five named functions, a new dedicated test file in `<files>`/`files_modified`/`must_haves.artifacts`,
  and a floor equal to the exact derived count. Both reviewers reached 4 + 3 = 7 independently.
- All 27 gates remain `sh -n`-clean and status-honoring; 9/9 porcelain guards intact; the five
  shape prohibitions from cycle 4 all still hold. Fixing scope did **not** regress shape.
- The three cross-plan legs are honestly declared, and the inherited-regression leg at
  `01-10:446` leg 2 explicitly disclaims any ownership of this task's deliverable, so it can no
  longer subsidize leg 1.

### Agreed Concerns

- **HIGH — the general rule is under-specified (H-7-1).** OWNERSHIP constrains who *declares* the
  counted tests, not what they *assert*; HOMING accepts a non-executable "named criterion". A
  trivial task-owned test plus prose homing satisfies both halves literally while the deliverable
  is entirely absent. **The known instance is fixed; the class is reduced, not terminated.**

### Divergent Views

None on substance. The orchestrator additionally raises two MEDIUMs that Codex did not reach —
`M-7-1` (a gate in `01-05` that half (a) does not actually permit) and `M-7-2` (a bullet homed to
a criterion no gate executes, where `go build ./...` cannot cover it because the go tool ignores
`testdata`). Both are instances of H-7-1's mechanism appearing inside the phase rather than
hypothetically, which strengthens rather than contradicts the shared verdict.

### Risk Assessment

**MEDIUM.** The concrete cycle-6 deliverable is correct and every structural invariant holds.
Residual risk is confined to the generality of the new prohibition: as written it can certify a
future scope-vacuous gate, and two in-phase gates already sit in the gap it leaves.
