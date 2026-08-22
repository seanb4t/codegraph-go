---
phase: 1
reviewers: [codex]
reviewed_at: 2026-08-22T21:52:00Z
review_cycle: 2
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
