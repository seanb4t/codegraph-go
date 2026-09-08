---
phase: 01-engine-seam-wire-protocol-secure-transport
reviewed: 2026-08-23T00:00:00Z
depth: deep
files_reviewed: 57
files_reviewed_list:
  - internal/uiserver/originguard.go
  - internal/uiserver/originguard_test.go
  - internal/uiserver/server.go
  - internal/uiserver/server_test.go
  - internal/uiserver/handlers.go
  - internal/uiserver/handlers_test.go
  - internal/uiserver/readonly_test.go
  - internal/uiserver/truncate.go
  - internal/uiserver/truncate_test.go
  - internal/uiserver/sourceblob_test.go
  - internal/uiserver/degrade.go
  - internal/uiserver/degrade_test.go
  - internal/uiproto/uiv1/ui.proto
  - internal/uiproto/uiv1/ui.pb.go
  - internal/uiproto/uiv1/uiv1connect/ui.connect.go
  - internal/cli/ui.go
  - internal/cli/ui_test.go
  - internal/cli/root.go
  - internal/schema/graph.proto
  - internal/schema/graph.pb.go
  - internal/schema/meta.go
  - internal/schema/meta_commit_test.go
  - internal/goldenspec/spec.go
  - internal/goldenspec/mcp.go
  - internal/goldenspec/spec_test.go
  - testdata/golden/byte_identity_test.go
  - testdata/golden/gocapture/main.go
  - testdata/golden/behavioral_test.go
  - testdata/golden/golden_test.go
  - internal/mcp/pending_writer_test.go
  - internal/mcp/server.go
  - internal/mcp/session_line.go
  - internal/query/detail.go
  - internal/query/detail_test.go
  - internal/query/detail_external_test.go
  - internal/query/node.go
  - internal/query/traverse.go
  - internal/query/errors.go
  - internal/query/errors_test.go
  - internal/query/explore.go
  - internal/query/render_markdown.go
  - internal/query/render_markdown_test.go
  - internal/query/files.go
  - internal/query/search.go
  - internal/query/validate.go
  - internal/query/engine.go
  - internal/query/meta_test.go
  - internal/indexer/commit.go
  - internal/indexer/commit_test.go
  - internal/indexer/resolve.go
  - internal/indexer/resolve_test.go
  - internal/indexer/sync.go
  - internal/indexer/pipeline.go
  - internal/indexer/pipeline_test.go
  - internal/textutil/truncate.go
  - internal/textutil/truncate_test.go
  - internal/upgrade/proto_task_test.go
  - internal/graphstore/open_lock_test.go
findings:
  critical: 3
  warning: 8
  info: 0
  total: 11
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-08-23
**Depth:** deep
**Files Reviewed:** 57
**Status:** issues_found

## Summary

The load-bearing invariants of this phase hold up under scrutiny. `originguard.go`'s
Origin/Host pairing is correct — I could not construct a cross-spelling admission, a
`null`-Origin admission, or a rebinding path through it. The generated `ui.pb.go` field
tags match `ui.proto` exactly (all 97+ numbers verified by tag extraction), with no
hand-edit drift and no renumbering. The MCP `pendingWriter` accounting is genuinely
symmetric now: a server-initiated notification is excluded on both sides, and a
server-initiated *request* is excluded on both sides too. `textutil.TruncateOnRuneBoundary`
and `uiserver.truncateSource` are rune-safe; `firstNLines` correctly ranges over `[]byte`
(not a string, which would have decoded runes and broken the index arithmetic).

The defects concentrate at the **mapping layer between the Engine and the wire**
(`internal/uiserver/handlers.go`), where three separate invariants that the Engine or the
CLI enforces are dropped on the floor: the aggregate response-size budget, the
"single-def has no source" contract, and the `MaxAffectedFiles` DoS cap. All three are
reachable from a well-formed request through the legitimate UI client, and none of them
is covered by the existing test suite.

## Structural Findings (fallow)

No structural pre-pass payload was supplied with this review. Cross-module facts below
were derived by direct call-graph tracing.

## Critical Issues

### CR-01: `ExploreResponse` has no aggregate `SourceBlob` bound — a correctly-truncated response is rejected by the transport backstop

**File:** `internal/uiserver/handlers.go:677-691`, `internal/uiserver/truncate.go:36-57`, `internal/uiserver/truncate_test.go:210-214`

**Issue:** `transportSendMaxBytes` (16 MiB) was sized against exactly one aggregate:
`uiMultiDefCap * sourceByteCap ≈ 5 MiB` for `GetNodeDetail`'s multi-def mode. That analysis
omits `Explore`, whose blob count is bounded by a *different, 50× larger* constant.

Trace: `Explore` (`handlers.go:680`) passes `req.Msg.GetMaxFiles()` straight through to
`eng.ExploreDetail`. `buildExploreResult` (`internal/query/detail.go:310-314`) accepts any
`max_files` up to `query.MaxFiles == 1000` (`internal/query/validate.go:29`), and because
`explicitMaxFiles > 0` the H21 adaptive budget at `detail.go:546` does **not** clamp it to
[1,20]. `groupMatchesByFile` therefore emits up to 1000 groups, `detail.go:645-652` reads a
full file per group, and `exploreGroupToProto` (`handlers.go:618`) attaches one
`SourceBlob` of up to `sourceByteCap` (256 KiB) to each.

Worst case is `1000 × 262144 ≈ 256 MiB`, sixteen times the 16 MiB backstop; a realistic
case (a monorepo with ~40 KB average source files at `max_files=400`) crosses 16 MiB with
no pathological input at all.

**Failure scenario:** UI issues `Explore{query: "http handler", max_files: 500}` against a
repo whose selected files average 35 KB. Every blob is correctly under `sourceByteCap`, so
`truncated` is `false` on all of them — yet `connect.WithSendMaxBytes` rejects the marshaled
message and the client receives `resource_exhausted` with no data at all. This is precisely
the "silently convert CORRECT truncation into a resource_exhausted error the user sees for
no reason" outcome that `truncate.go:41-47` says the two-layer design exists to prevent.

`TestTransportBackstopSitsAboveApplicationCap` (`truncate_test.go:210-214`) only asserts
`transportSendMaxBytes > sourceByteCap` — the *single-blob* inequality — so it cannot catch
this, and `TestUIServiceSourceBlobAtTheCapClearsTheTransportBackstop`
(`sourceblob_test.go:239`) exercises exactly one blob.

**Fix:** Give `Explore` its own UI-layer group cap, mirroring `uiMultiDefCap`'s role for
`GetNodeDetail`, and add the aggregate to the backstop assertion.

```go
// truncate.go
// uiExploreGroupCap bounds how many ExploreGroups carry a SourceBlob in one
// response. sourceByteCap*uiExploreGroupCap must stay strictly below
// transportSendMaxBytes (asserted below).
const uiExploreGroupCap = 32

// truncate_test.go — replace the single-blob assertion with the two aggregates
func TestTransportBackstopSitsAboveEveryAggregate(t *testing.T) {
    for _, tc := range []struct{ name string; agg int }{
        {"node-detail-multi", uiMultiDefCap * sourceByteCap},
        {"explore-groups", uiExploreGroupCap * sourceByteCap},
    } {
        if transportSendMaxBytes <= tc.agg {
            t.Fatalf("%s aggregate %d >= transportSendMaxBytes %d", tc.name, tc.agg, transportSendMaxBytes)
        }
    }
}
```

and in `handlers.go`, clamp before mapping:

```go
func (s *uiService) Explore(ctx context.Context, req *connect.Request[uiv1.ExploreRequest]) (...) {
    maxFiles := int(req.Msg.GetMaxFiles())
    if maxFiles <= 0 || maxFiles > uiExploreGroupCap {
        maxFiles = uiExploreGroupCap
    }
    ...eng.ExploreDetail(req.Msg.GetQuery(), maxFiles)
```

(If preserving the full 1000-file result is required, attach source only to the first
`uiExploreGroupCap` groups and add a `source_attached`/`total_groups` pair to
`ExploreGroup`/`ExploreResponse` at the next free field numbers, mirroring
`NodeDefinition.detail_gathered`.)

---

### CR-02: `GetNodeDetail` single-def mode hard-fails for any definition with no readable on-disk file

**File:** `internal/uiserver/handlers.go:542-550`

**Issue:** The single-definition branch unconditionally performs a source read the CLI
never performs:

```go
case query.NodeDetailModeSingleDef:
    ...
    src, err := eng.SourceFor(d.Definition.Node.FilePath)
    if err != nil {
        return nil, err          // <- fails the WHOLE rpc
    }
```

Two reachable inputs make `SourceFor` fail on a perfectly valid node:

1. **Package pseudo-nodes carry no `FilePath` at all.** `internal/indexer/resolve.go:206-212`
   constructs them with `Id`, `Kind`, `Name`, `QualifiedName` and *nothing else* — `FilePath`
   is the zero value. `enumerateSymbolDefs` (`internal/query/node.go:151-177`) matches purely
   on `n.Name`, so `GetNodeDetail{symbol: "uiserver"}` resolves to the single package node,
   takes the single-def branch, and calls `SourceFor("")` →
   `invalidArgumentf("query: empty file path")` (`node.go:37-39`) → `CodeInvalidArgument`.
   The request was well-formed; the client is told it was not.

2. **A stale index.** The file was deleted or renamed since the last `sync` (`stale` is a
   first-class state this product models and reports). `os.ReadFile` returns `ENOENT` →
   `mapEngineError` default → `CodeInternal`, and the whole node view is unavailable.

In both cases `codegraph node <symbol>` on the CLI succeeds — `Node()` renders the
single-def branch through `RenderNode(node, calls, calledBy)` (`node.go:322`), which takes
no source. This is a UI-only behavioral divergence introduced by the wire layer, not a
shared Engine limitation.

**Failure scenario:** User clicks the `uiserver` package node in the graph view; the detail
panel shows `invalid_argument: query: empty file path` instead of the node's callers and
callees.

**Fix:** `source` is an *optional* field on `GetNodeDetailResponse` (`ui.proto:455`) and the
proto already documents that reading a field for the wrong mode yields a zero value, not an
error. Degrade the read instead of failing the RPC:

```go
case query.NodeDetailModeSingleDef:
    resp.Node = nodeToProto(d.Definition.Node)
    resp.Calls = nodesToProto(d.Definition.Calls)
    resp.CalledBy = nodesToProto(d.Definition.CalledBy)
    // A definition may have no on-disk file (the synthetic "package"
    // pseudo-node kind carries no FilePath) or its file may have been
    // removed since the index was built. Neither makes the node itself
    // unavailable: leave source unset, exactly as multi-def mode does.
    if fp := d.Definition.Node.FilePath; fp != "" {
        if src, err := eng.SourceFor(fp); err == nil {
            resp.Source = sourceBlobToProto(truncateSource(src))
        }
    }
```

Add a regression test covering both a `FilePath == ""` node and a definition whose file was
deleted after indexing.

---

### CR-03: `Affected` RPC bypasses the `MaxAffectedFiles` input cap the CLI enforces

**File:** `internal/uiserver/handlers.go:431-448`

**Issue:** `query.ValidateAffectedFiles` exists specifically to bound this input — its own
doc comment (`internal/query/validate.go:143-157`) says it was added under CR-01/T-03-02-DoS
so "a caller (or an untrusted/compromised MCP client, or a hostile CI diff) that tries to
grow `affected --stdin`'s input past this ceiling gets an explicit error instead of an
ever-growing, unbounded seen/files/fileSet allocation", and that it is **exported precisely
so `internal/cli/affected.go` can enforce it before `Engine.Affected` is ever called**.

`Engine.Affected` (`internal/query/traverse.go:554-588`) does *not* enforce it — it validates
and clamps `depth` only, then builds `fileSet` from every entry. The new `Affected` handler
passes `req.Msg.GetFiles()` — an unbounded wire-supplied `repeated string` — straight into
it with no cap:

```go
result, err := eng.Affected(req.Msg.GetFiles(), int(req.Msg.GetDepth()))
```

The handler's own doc comment (`handlers.go:427-430`) claims "files and depth pass straight
through — validateDepth/clampAffectedDepth already bound and clamp depth for every caller",
which is true of `depth` and silently untrue of `files`.

The only remaining bound is `transportReadMaxBytes` (1 MiB). At ~4 bytes per entry that
admits on the order of 2·10⁵ paths — roughly 25× `MaxAffectedFiles == 10000`.

**Failure scenario:** A page on an allowed loopback origin (or a buggy UI paginating a huge
diff) POSTs `Affected{files: [...200k short paths...]}`. The server allocates a 200k-entry
`fileSet`, plus `BuildReverseAdjacency` + `BuildImplementsIndex` + `buildContainsIndex` over
the whole graph per request, with no cancellation (see WR-02). The documented ceiling that a
prior review cycle added is simply not in the path.

**Fix:** Enforce the exported validator at the new surface, mapping it to
`CodeInvalidArgument`:

```go
err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
    files := req.Msg.GetFiles()
    if verr := query.ValidateAffectedFiles(len(files)); verr != nil {
        return connect.NewError(connect.CodeInvalidArgument, verr)
    }
    result, err := eng.Affected(files, int(req.Msg.GetDepth()))
    ...
```

Better still, move the check inside `Engine.Affected` so *every* caller inherits it and no
future surface can repeat this omission — the CLI's own pre-check then becomes a cheap
early-out rather than the sole enforcement point.

## Warnings

### WR-01: `mapEngineError`'s default arm leaks absolute host filesystem paths to the browser

**File:** `internal/uiserver/handlers.go:78-80`

**Issue:** `default: return connect.NewError(connect.CodeInternal, err)` places `err.Error()`
verbatim into the wire message. The errors that land here are frequently
`*os.PathError` values carrying the **absolute** host path: `readSourceFile`
(`internal/query/node.go:84-90`) returns `os.ReadFile(abs)`'s error, and `resolveSourcePath`
returns raw `filepath.EvalSymlinks` errors (`node.go:65-72`), both containing `abs`.

This directly contradicts the posture the same phase applies one file over:
`indexingInProgressMessage` (`internal/uiserver/degrade.go:86-94`) is a fixed generic
sentence *specifically* because "this message crosses into an unauthenticated loopback
browser caller" (T-01-06), with `TestIndexingInProgressMessageLeaksNothing` asserting it
carries no path separator. The generic error path does the opposite with no scrubbing.

**Failure scenario:** `GetNodeDetail{symbol: "Foo"}` where `Foo`'s indexed file was deleted
returns `internal: open /Users/alice/work/private-client-repo/internal/billing/keys.go: no
such file or directory` into the browser's JS console and network tab — disclosing the
checkout's absolute path and directory layout.

**Fix:** Scrub the default arm the same way the degrade path is scrubbed. Log the full error
server-side; return a generic message on the wire.

```go
default:
    // Never surface a raw error to the loopback browser caller: os.PathError
    // values carry the absolute host path (T-01-06). Log it, return generic.
    log.Printf("uiserver: internal error: %v", err)
    return connect.NewError(connect.CodeInternal, errors.New("an internal error occurred"))
```

Add a test asserting no response message contains `os.PathSeparator` for a
deleted-source-file scenario, mirroring `TestIndexingInProgressMessageLeaksNothing`.

---

### WR-02: `withEngine`'s `ctx` parameter is never used — 8 of 9 RPCs have no cancellation or deadline

**File:** `internal/uiserver/handlers.go:35-46`

**Issue:** `withEngine(ctx context.Context, repoPath string, fn func(*query.Engine) error)`
never references `ctx` in its body. `openEngine(repoPath)` takes no context, and `fn(eng)`
is invoked with none. Every handler except `GetStatus` (which passes `ctx` to `eng.Status`)
therefore threads a context that is silently discarded — a signature that advertises
cancellation support the code does not have.

**Failure scenario:** A browser tab issues `Explore{max_files: 1000}`, then is closed. The
HTTP request context is cancelled; `net/http` notices nothing because the handler never
selects on it. The server continues the full pipeline — reverse-adjacency build, RWR,
`buildExpandAdjacency`, and up to 1000 whole-file reads — to completion. Repeated
navigations stack these up with no backpressure and no way to shed them. Combined with
CR-03's uncapped `Affected` input this is a straightforward local resource-exhaustion
amplifier.

**Fix:** Either honour the context or delete the parameter so the signature stops lying. The
honest minimum is a pre-flight check plus a cancellation-aware wrapper:

```go
func withEngine(ctx context.Context, repoPath string, fn func(context.Context, *query.Engine) error) error {
    if err := ctx.Err(); err != nil {
        return connect.NewError(connect.CodeCanceled, err)
    }
    eng, closer, err := openEngine(repoPath)
    ...
    if err := fn(ctx, eng); err != nil { return mapEngineError(err) }
```

and thread `ctx` into the Engine's own scan loops (`IterateNodes`/`IterateEdges` callers) so
a cancelled request actually stops work. If that is out of scope for this phase, drop the
parameter and record the gap explicitly rather than carrying an unused one.

---

### WR-03: `http.Server` is constructed with no timeouts (gosec G112)

**File:** `internal/uiserver/server.go:100-104`

**Issue:** `&http.Server{Handler: guarded}` sets no `ReadHeaderTimeout`, `ReadTimeout`,
`WriteTimeout` or `IdleTimeout`. Go's defaults are all "no limit".

The origin guard runs at the *handler* level, which is after headers are read — so it does
nothing against a client that connects and never completes its header block. Any local
process, and any web page (which can open sockets to `localhost:<port>` even though the
guard will later reject the request), can hold connections open indefinitely, each pinning
a goroutine and a file descriptor.

**Failure scenario:** A page runs `for (let i=0;i<10000;i++) fetch('http://127.0.0.1:PORT/')`
with slow/aborted bodies; the responses are rejected by the guard but the connections and
their goroutines accumulate until the process hits its fd limit and `codegraph ui` stops
accepting the legitimate UI's requests.

**Fix:**

```go
srv: &http.Server{
    Handler:           guarded,
    ReadHeaderTimeout: 5 * time.Second,
    ReadTimeout:       30 * time.Second,
    WriteTimeout:      60 * time.Second,  // must exceed the slowest legitimate Explore
    IdleTimeout:       120 * time.Second,
},
```

---

### WR-04: `renderSingleDefNode` and `renderMultiDefNode` are dead code after the extraction

**File:** `internal/query/node.go:404-410`, `internal/query/node.go:420-435`

**Issue:** Both methods have zero call sites in the tree (verified across production and
test files). Their own doc comments state "Node() no longer calls this directly", and the
justification given — "kept as a standalone single-def render entry point" — describes an
entry point nothing enters. Go does not warn on unused methods, so these will not be caught
by the compiler or `go vet`.

This matters beyond tidiness: they are a *second* rendering path over the same builders. A
future contributor fixing a render bug in `Node()` will not see these, and anyone who later
wires one up gets divergent behavior from the path the 26-pair golden suite actually
exercises — the exact "CLI, MCP and UI cannot disagree" invariant the extraction exists to
protect.

**Fix:** Delete both methods. If a standalone render entry point is genuinely wanted for a
later phase, add it then, with a caller and a golden.

---

### WR-05: `Sync` unconditionally overwrites `Meta.commit_sha`, erasing a known commit on any transient git failure

**File:** `internal/indexer/sync.go:61`, `internal/indexer/sync.go:176-182`, `internal/indexer/sync.go:405-411`

**Issue:** `headCommitSHA := resolveHeadCommitSHA(repoRoot)` returns `""` on *every* failure
mode by design (`internal/indexer/commit.go:38-49`): git absent from `PATH`, the 5-second
`resolveHeadCommitGitTimeout` firing, `git rev-parse` exiting non-zero because
`.git/index.lock` is held by a concurrent operation, or output failing the hex check. Both
meta write sites then assign it unconditionally:

```go
newMeta.CommitSha = headCommitSHA   // sync.go:181 and sync.go:410
```

Because `schema.NewMeta()` starts from a zero-valued record, an empty result does not leave
the previous value alone — it **replaces** a known-good commit SHA with absent.

**Failure scenario:** A watcher-triggered `Sync` fires while the user's `git rebase` holds
`.git/index.lock`. `git rev-parse HEAD` fails, `resolveHeadCommitSHA` returns `""`, and the
index's recorded commit is wiped. `codegraph ui`'s status panel flips from a real SHA to
"unknown" and stays there until the next successful full index — with no error anywhere to
explain it, since D-05 defines empty as "unknown, never an error".

**Fix:** Distinguish "resolved to empty" from "could not resolve", and preserve the prior
value in the second case:

```go
// preserve the previously recorded commit when HEAD could not be resolved
// this run: an unresolvable HEAD is not evidence that the old value is wrong.
if headCommitSHA != "" {
    newMeta.CommitSha = headCommitSHA
} else {
    newMeta.CommitSha = meta.GetCommitSha()
}
```

Apply at both sites, and add a test that a Sync with `gitExecLookPath` stubbed to fail
leaves a pre-existing `commit_sha` intact.

---

### WR-06: an unparseable inbound line permanently inflates `pending`, costing a full 5s hang on every subsequent exit

**File:** `internal/mcp/server.go:331-336`, `internal/mcp/server.go:369-381`, `internal/mcp/server.go:306-312`

**Issue:** The two sniffers' conservative defaults are documented as pointing in opposite
directions on purpose, but the combination is not self-healing.
`looksLikeJSONRPCCall` returns `true` on `json.Unmarshal` failure (`server.go:334`), so a
malformed line increments `pending`. The reply go-sdk produces for it is a JSON-RPC
parse-error response with `id: null`, which `looksLikeJSONRPCResponse` deliberately refuses
to count (`server.go:378`). Nothing else ever decrements it.

`pending` is therefore stuck at ≥1 for the remainder of the session, and `waitForDrain`
(`server.go:306-312`) burns the entire `stdinLingerGrace` (5 s) at EOF.

**Failure scenario:** A client sends one truncated line (a partial write, a keepalive
newline, a UTF-8 BOM prefix), then a hundred correct calls. Every one of those balances
correctly, but on stdin close the process still sleeps 5 seconds before exiting. To an agent
harness batching MCP invocations this reads as a 5-second hang on every session, not as a
"bounded wait" — and `pendingUnderflows` stays at zero, so the diagnostic counter built to
catch imbalance reports all-clear.

**Fix:** Make the over-count self-correcting. Track the malformed-line count separately and
subtract it from the drain target, or decrement on a `null`-id response when a malformed
line is outstanding:

```go
// stdinLingerReader
if looksLikeJSONRPCCall(line) {
    s.pending.Add(1)
    if !json.Valid(line) {
        s.unparseable.Add(1) // this increment will never be balanced by a real response
    }
}

func (s *stdinLingerReader) waitForDrain() {
    deadline := time.Now().Add(stdinLingerGrace)
    for s.pending.Load() > s.unparseable.Load() && time.Now().Before(deadline) {
        time.Sleep(stdinLingerPollInterval)
    }
}
```

---

### WR-07: a missing `Sources` entry renders as an empty, non-truncated `SourceBlob` indistinguishable from an empty file

**File:** `internal/uiserver/handlers.go:613-619`

**Issue:** `sourceBlobToProto(truncateSource(sources[g.Path]))` performs an unchecked map
read. A miss yields `nil`, and `truncateSource(nil)` returns
`{Content: nil, Truncated: false, TotalLines: 0, TotalBytes: 0, ...}` — byte-for-byte the
same wire value as a genuinely zero-length file. There is no way for a client to tell
"source unavailable" from "file is empty".

Today `buildExploreResult` populates `Sources` for every group (`detail.go:645-652`), so
this is latent rather than live — but the invariant lives in a different package from the
code depending on it, and the comment on `exploreGroupToProto` ("looked up by the group's
own Path — never by position") reads as if it is defending the association, which it does
not do for a miss.

**Failure scenario:** Any future change that makes `Sources` sparse (a per-file size cap, a
skeletonized-file skip, a partial-failure mode) silently renders every affected file as
empty in the UI instead of surfacing that the source was not retrieved.

**Fix:** Make the miss explicit rather than silent:

```go
func exploreGroupToProto(g query.ExploreFileGroup, skeletonFiles map[string]bool, sources map[string][]byte) *uiv1.ExploreGroup {
    out := &uiv1.ExploreGroup{
        Path:         g.Path,
        Symbols:      nodesToProto(g.Symbols),
        Skeletonized: skeletonFiles[g.Path],
    }
    // Leave source UNSET on a miss: an unset SourceBlob is distinguishable
    // from a populated one describing a genuinely empty file.
    if src, ok := sources[g.Path]; ok {
        out.Source = sourceBlobToProto(truncateSource(src))
    }
    return out
}
```

---

### WR-08: `resolveNodeForDetail`'s not-found error was not converted to `notFoundf` while every sibling was

**File:** `internal/query/node.go:290-292`

**Issue:** This phase converted essentially every caller-reachable rejection in
`internal/query` to `notFoundf`/`invalidArgumentf` so `mapEngineError` can classify by
`errors.Is` — `resolveSymbolNode` (`traverse.go:225`), `validateLimit`/`validateMaxFiles`/
`validateDepth` (`validate.go`), `validateFilesDepth` and the format check (`files.go`),
`ValidateKind`, `Query`/`Search`'s empty-term rejection. This one was missed:

```go
if len(candidates) == 0 {
    return nil, fmt.Errorf("query: symbol %q not found in file %q", symbol, file)
}
```

It is currently unreachable as a *surfaced* error because `buildNodeDetail`'s fast path
swallows it (`detail.go:206`, `if node, err := ...; err == nil`). That is exactly what makes
it dangerous: the moment anyone calls `resolveNodeForDetail` directly, or changes the fast
path to propagate, an unmistakable not-found becomes `CodeInternal` rather than
`CodeNotFound`, and the miss will not be obvious because the message reads correctly.

**Fix:**

```go
return nil, notFoundf("query: symbol %q not found in file %q", symbol, file)
```

The message bytes are unchanged (`classifiedError.Error()` returns `msg` verbatim,
`errors.go:36-38`), so no golden or CLI output moves.

---

_Reviewed: 2026-08-23_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
