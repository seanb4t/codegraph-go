# Phase 1: Engine Seam, Wire Protocol & Secure Transport - Research

**Researched:** 2026-08-22
**Domain:** Go RPC transport (connect-go/protobuf), engine-layer refactor, MCP server concurrency bug, loopback network security
**Confidence:** HIGH for verified code citations and library API surface; MEDIUM for tool-modfile MVS compatibility (not measured live); LOW/ASSUMED flagged inline where training knowledge was used without in-session verification

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Engine Seam (ENG-01, ENG-02)**
- D-01: `Node()` and `Explore()` become thin wrappers over new structured variants — not parallel implementations. `renderSingleDefNode` keeps its exact output but is reduced to `RenderNode(d.Node, d.Calls, d.CalledBy)` over a `NodeDetail` produced by a single extracted gather. Exactly ONE gather path.
- D-02: `NodeDetail` covers all three shapes `Node()` can return: file-only source, single-def, multi-def. Partial coverage rejected — ENG-01 blocks Phases 3–6.
- D-03: `NodeDetail`/`ExploreResult` are plain Go structs in `internal/query`, not generated protobuf types. The RPC layer owns the mapping to `uiv1` messages. `internal/query` stays wire-agnostic.
- D-04: The "goldens stay byte-identical" claim is proven by mutation-proof with the count asserted: deliberately break the extracted gather (drop `calledBy`), observe all 26 scenarios go RED, record the number that failed, revert. Scoring is by counting `--- PASS` lines, never by exit status.

**Commit-Aware Meta (ENG-04)**
- D-05: The indexed commit SHA is added to `schema.Meta` as field 8, following the `has_file_index = 7` precedent — absent means a pre-upgrade graph and degrades gracefully.
- D-06: The SHA is surfaced only via the `Status` RPC — `codegraph status` output is untouched. No golden covers `status`.

**Process & Launch (SRV-01, SRV-03)**
- D-07: `codegraph ui` binds ephemeral `127.0.0.1:0` and prints the resulting URL. No `--port` flag and no `--host` flag in v1.
- D-08: The bind address lives as a field on a server options struct (defaulting to `127.0.0.1:0`) that nothing wires from outside in v1. Explicitly rejected: a config file — this repo has no user-config mechanism; the established override convention is env vars paired with flags.
- D-09: The browser opens by default, with `--no-open`, and automatic suppression when stdout is not a TTY or `CI` is set. Needs a small cross-platform open helper — none exists in the tree today.
- D-10: The process runs foreground until Ctrl-C, matching `serve` and `daemon start`. No PID file, no `ui stop` verb.

**Bounded Responses (RPC-05)**
- D-11: A verbatim source response is bounded by a line cap as the primary limit, with a byte cap underneath it. Byte truncation MUST land on a UTF-8 rune boundary.
- D-12: Truncation is signalled by an explicit field on the response (`truncated` plus the true total), never by an error.
- D-13: Both `WithSendMaxBytes` and `WithReadMaxBytes` are set as a transport backstop above application truncation. `connect-go` defaults to unlimited on both sides.

**Degraded State (SRV-04)**
- D-14: A store that stays locked past `graphstore.Open`'s retry budget is reported as `CodeUnavailable` carrying a typed `IndexingInProgress` detail message.
- D-15: No retry beyond `Open`'s existing bounded budget.
- D-16: `Status` still answers when the store cannot be opened, reporting what is derivable without opening it (repo path, store-exists, lock-held, "indexing in progress") and degrading the graph-derived counts.

### Claude's Discretion

- The exact fix mechanics for FIX-01 — whether `pendingWriter` learns to distinguish server-initiated notifications from client responses, or the counter moves so only response writes decrement.
- The BLD-04 guard's shape — regenerate-into-tempdir-and-diff vs regenerate-in-place-and-`git diff --exit-code`, and whether a new `task proto` becomes a required CI check. Constraint: it must report how many generated files it actually compared and be watched fail against a deliberately stale checked-in file.
- Exact field names, and whether the truncation total is expressed in bytes or lines.
- `connectrpc.com/connect` version pinning. It is not currently a dependency; `google.golang.org/protobuf v1.36.11` already is, directly.

### Deferred Ideas (OUT OF SCOPE)

- Stable/bookmarkable UI URL — D-07's ephemeral port means Phase 3's deep links live only as long as the process.
- Surfacing the indexed commit SHA in `codegraph status` — deliberately excluded from v1 by D-06.
- Byte-range paging for source responses — rejected for RPC-05 in favor of line-capped truncation.
- A user config file mechanism — explicitly declined as out of scope for a port question.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| ENG-01 | Structured `NodeDetail` variant; `Node()` stays byte-identical | Exact extraction boundary mapped at `internal/query/node.go:316-465`; concrete struct sketch below |
| ENG-02 | Structured `ExploreResult` variant; `Explore()` stays byte-identical | Exact gather/render boundary mapped at `internal/query/explore.go:240-596`; concrete struct sketch below |
| ENG-04 | `schema.Meta` records indexed commit SHA, field 8, additive | Proto field layout confirmed at `internal/schema/graph.proto:137-151`; population point identified at `internal/indexer/sync.go:165,393` |
| RPC-01 | Protobuf schema for UI API, additive-only discipline | `connectrpc.com/connect` v1.20.0 verified current; codegen toolchain (buf/protoc-gen-go/protoc-gen-connect-go) verified locally installed |
| RPC-02 | connect-go handlers mount on `net/http` | Mounting pattern verified via Context7 (official connect-go docs); plain-HTTP/1.1 sufficient, no h2c needed |
| RPC-05 | Bounded message sizes; verbatim source handled boundedly | `WithSendMaxBytes`/`WithReadMaxBytes` verified via Context7; rune-safe truncation precedent at `internal/mcp/session_line.go:18-85` |
| SRV-01 | `codegraph ui` own process, loopback, prints+opens URL | `signal.NotifyContext` foreground pattern at `internal/cli/daemon.go:142-143`; `query.OpenAt` per-call seam at `internal/query/engine.go:160-196` already satisfies "never caches a handle" |
| SRV-02 | Origin/Host exact-match validation | CVE-2024-28224 (Ollama) and CVE-2025-66414/66416 (MCP SDKs) researched via WebSearch; NCC Group's documented mitigation shape confirmed |
| SRV-03 | Read-only by construction; bind/auth as seams, not flags | `graphstore.Reader`-only `Engine` already has no mutator (`internal/query/engine.go`); confirmed no config-file mechanism exists repo-wide |
| SRV-04 | Per-RPC open-snapshot-close; `ErrStoreLocked` past retry budget renders as "indexing in progress" | `graphstore.Open`'s bounded retry at `internal/graphstore/pebble_store.go:141-152`; D-16's degrade path is NEW plumbing — the existing `Status`/`openEngine` path does not have it (see Pitfall) |
| BLD-04 | Protobuf codegen drift guard, both surfaces | No proto task exists in `Taskfile.yml` today (confirmed by grep); `corpora:drift` (`Taskfile.yml:3639-3677`) and `taskfile_shape_test.go` idiom (`internal/upgrade/taskfile_shape_test.go`) are the reusable patterns |
| FIX-01 | `pendingWriter` counter not corrupted by server-initiated notifications | Root cause confirmed at `internal/mcp/server.go:334-341` vs `internal/mcp/server.go:261-300`; relationship to the toolslist-repeat flake investigated and DISPROVEN (see Pitfall/Sources) |
</phase_requirements>

## Summary

This phase has no UI surface — it is entirely plumbing: an engine-layer refactor that must produce byte-identical output, a brand-new wire protocol (connect-go + protobuf) that has zero prior art in this repo beyond an existing `graph.proto`/`graph.pb.go` pair with no codegen guard, a loopback HTTP server that must reject DNS-rebinding attempts before any handler runs, and an unrelated but folded-in MCP concurrency bug fix. None of this is exploratory: `connectrpc.com/connect` is a mature, actively-released library (v1.20.0, verified via the official Go module proxy and Context7-sourced docs) with a well-documented mounting pattern (`net/http.ServeMux` + generated `NewXServiceHandler`), and the two CVEs SRV-02 cites are real, publicly documented, with an unambiguous mitigation shape: exact-match Host/Origin allowlisting, not `Contains`/prefix matching.

The single most consequential finding is that **the existing "26 frozen goldens" safety net ENG-01/ENG-02 lean on does not currently perform byte-for-byte comparison anywhere in the automated test suite.** `TestReFrozenGoldensValid` (`testdata/golden/golden_test.go:243-297`) only asserts existence, non-emptiness, and JSON-envelope shape — it never re-invokes `Engine.Node`/`Engine.Explore` and diffs the result against the committed fixture. `TestCorpusBehavior_Go`'s own doc comment (`testdata/golden/behavioral_test.go:685-691`) states plainly that byte-diffing against a frozen golden was retired in FIXT-04 ("not byte-diffs against a frozen golden... formerly diffed against are gone as of this phase"). This means D-04's mutation-proof claim — "deliberately break the extracted gather... observe all 26 scenarios go RED" — cannot be satisfied by the CURRENT test suite; a genuine byte-identity oracle over the 26 (corpus, filename) pairs must be ADDED as part of this phase's own work, reusing `gocapture/main.go`'s already-committed `lockedCorpusArgs` (the exact symbol/query values used to produce each golden) as the parameter source. This is not optional scope creep — without it, "prove all 26 goldens byte-identical" is unverifiable by construction.

The second major finding is that FIX-01 (the `pendingWriter` counter bug) and the folded-in `toolslist-repeat` ordering flake are **different root causes**, verified by reading the actual `modelcontextprotocol/go-sdk@v1.7.0` source in the local module cache: every JSON-RPC call except `initialize` is released into concurrent execution via `jsonrpc2.Async(ctx)` (`go-sdk/mcp/server.go:1908-1913`, citing `go-sdk#26`), so two pipelined `tools/list` calls run their handlers in **separate goroutines** with no ordering guarantee on which finishes (and thus writes its response) first — a behavior entirely independent of the `pendingWriter` write-counter bug. The phase must still "prove it either way" per CONTEXT, and this research now provides that proof: they are not the same bug, and fixing `pendingWriter` will not resolve the ordering flake.

Third, the SRV-04/D-16 "Status still answers when the store cannot be opened" requirement is **new behavior**, not a reuse of an existing degrade path: today, `internal/mcp/tools.go`'s `codegraph_status` tool goes through the identical `openEngine`/`query.OpenAt` seam as every other tool, and a `graphstore.ErrStoreLocked` that survives the retry budget makes `OpenAt` fail outright — `Status()` never even runs. The UI RPC's Status handler must catch that specific error class before `Engine.Status` is reachable and construct a degraded response from filesystem facts alone (`ResolveCodegraphDir` succeeding is independent of the store lock).

**Primary recommendation:** Extract the Engine seam and add the missing byte-identity oracle as one unit of work (they are inseparable — D-04 cannot be proven without it); stand up `connectrpc.com/connect` v1.20.0 + a `buf`-driven codegen pipeline with a Go-test-shaped drift guard matching this repo's own `taskfile_shape_test.go` idiom; implement Origin/Host validation as a plain `net/http` middleware wrapping the entire mux (not a connect interceptor, so it runs before any protocol parsing); and treat FIX-01 and the toolslist-repeat flake as two separate deliverables with two separate reproductions.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Node/Explore structured gather (ENG-01/02) | Backend (Engine) | — | `internal/query.Engine` is the sole read seam for CLI, MCP, and now UI; wire-agnostic by D-03 |
| Commit SHA capture (ENG-04) | Backend (Indexer) | Backend (Engine, read) | Written at index/sync time (`internal/indexer/sync.go`), read via `Engine.Status` only |
| Protobuf schema + codegen (RPC-01, BLD-04) | Backend (build-time) | — | Codegen is a build-time concern; runtime types stay in `internal/query` per D-03 |
| connect-go handler mounting (RPC-02) | Backend (API) | — | `net/http.Server` process, no browser/CDN tier exists in this project |
| Origin/Host validation (SRV-02) | Backend (API), network boundary | — | Must run before protocol parsing — plain `net/http` middleware, not an RPC-layer concern |
| Bind address / no-flags (SRV-01, SRV-03) | Backend (process) | — | `codegraph ui` is its own OS process; no daemon/IPC tier in this milestone |
| Response bounding (RPC-05) | Backend (API) | — | Truncation decision belongs where the source bytes are read (Engine/RPC layer), not the future browser tier |
| Store lifecycle / degrade (SRV-04) | Backend (Engine/GraphStore) | Backend (API, error mapping) | `graphstore.Open`'s retry is already Engine-tier; the RPC layer's job is mapping `ErrStoreLocked` to a Connect error code, not retrying again |
| MCP counter fix (FIX-01) | Backend (MCP transport) | — | Isolated to `internal/mcp/server.go`; no interaction with the new UI transport |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `connectrpc.com/connect` | v1.20.0 (2026-05-20) `[VERIFIED: proxy.golang.org + Context7 official docs]` | Wire protocol: typed RPC over `net/http`, unary + streaming, gRPC/gRPC-Web/Connect protocol support | Already the milestone's locked wire-protocol decision (STATE.md); pure Go, no gRPC transport stack, smaller supply-chain surface than `grpc-go` |
| `google.golang.org/protobuf` | v1.36.11 (already a direct dependency, confirmed in `go.mod:37`) `[VERIFIED: go.mod]` | Protobuf runtime | Already in-tree backing `internal/schema/graph.pb.go`; the UI schema is a SECOND surface, not a greenfield one |
| `github.com/bufbuild/buf` (CLI, build-tool only) | v1.72.0 (2026-07-17) `[VERIFIED: proxy.golang.org]` | Protobuf compiler + codegen orchestration, no C++ `protoc` binary required | Pure-Go-installable (`go install github.com/bufbuild/buf/cmd/buf@v1.72.0`), fits this repo's tool-modfile pinning discipline (`go.tool.mod`/`go.tool-lint.mod` precedent); avoids adding a non-Go external binary dependency the way raw `protoc` would |
| `connectrpc.com/connect/cmd/protoc-gen-connect-go` | same module as connect-go, v1.20.0 `[VERIFIED: Context7 official docs — README.md's cmd/protoc-gen-connect-go references]` | Generates Connect service handlers/clients from `.proto` | Official code generator for the chosen wire protocol; ships in the SAME Go module as `connectrpc.com/connect`, so no separate version-skew risk |
| `google.golang.org/protobuf/cmd/protoc-gen-go` | pinned to the same v1.36.x line as the runtime dep `[VERIFIED: go.mod direct dependency + standard protobuf-go tooling convention]` | Generates Go message types from `.proto` | Standard companion generator; must track the runtime module's version to avoid `ErrIncompatibleGeneratedCode` panics at binary init |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/pkg/browser` | v0.0.0-20240102092130-5ac0b6a4141c `[VERIFIED: proxy.golang.org]` | Cross-platform "open URL in default browser" (`xdg-open`/`open`/`rundll32`) | D-09's browser-open helper — zero-dependency (only `golang.org/x/sys`, already transitively present), and structurally identical to Go's OWN internal tooling (`golang.org/x/pkgsite/internal/browser`, `cmd/internal/browser` use the same 3-OS-branch dispatch — this is the idiomatic Go approach, not a novel dependency) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `buf` for codegen | Raw `protoc` + `protoc-gen-go` + `protoc-gen-connect-go` invoked directly | `protoc` itself (confirmed installed locally: `libprotoc 35.1`) is a C++ binary not installable via `go install`/tool-modfile pinning — it would need to be a documented external prerequisite (like `zig`/`task` are today) rather than a pinned Go tool. `buf` avoids this: it has its own protobuf compiler and can invoke the Go plugins via `go run`/pinned tool-modfile binaries with zero external non-Go dependency. Recommend `buf`. |
| A single tool-modfile for buf + connect-go + protobuf generators | A dedicated `go.tool-proto.mod` (third isolated modfile) | This repo has PRECEDENT for splitting on MVS conflict (`go.tool.mod` vs `go.tool-lint.mod`, documented conflict with `actionlint`'s YAML API). Whether `buf`'s dependency tree (v1.72.0, large — cobra, protovalidate, etc.) MVS-conflicts with `go.tool.mod`'s existing tree (task/goreleaser/govulncheck) or `go.tool-lint.mod`'s (actionlint) is UNVERIFIED — this must be measured live in a scratch modfile before deciding, exactly as `go.tool.mod`'s own header documents doing for its own tools. See Open Questions. |
| Custom `IndexingInProgress` proto message for D-14's typed detail | `google.golang.org/genproto/googleapis/rpc/errdetails` (`errdetails.BadRequest`, etc.) | The stdlib-adjacent `errdetails` package models generic gRPC error shapes (bad request field violations, retry info) but has no purpose-built "indexing in progress" semantic. A custom message defined in the same UI `.proto` file (following the additive-only discipline RPC-01 already inherits) is more legible to the browser client and costs nothing extra since a new proto file already exists. |

**Installation:**
```bash
# Runtime dependency (main go.mod)
go get connectrpc.com/connect@v1.20.0

# Build tools (pin in an isolated tool modfile per this repo's MVS-conflict precedent —
# measure compatibility with go.tool.mod / go.tool-lint.mod before choosing which, see Open Questions)
go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.20.0
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
go install github.com/bufbuild/buf/cmd/buf@v1.72.0
```

**Version verification:** All four package versions above were checked live against `https://proxy.golang.org/<module>/@latest` (the authoritative Go module proxy) during this research session, on 2026-08-22. `connectrpc.com/connect`'s API surface (handler mounting, `WithSendMaxBytes`/`WithReadMaxBytes`, `NewError`/`NewErrorDetail`) was independently confirmed via Context7's fetch of the library's own README/`_autodocs` — not from training-data recall.

## Package Legitimacy Audit

The `gsd-tools query package-legitimacy check` seam supports only `npm|pypi|crates` ecosystems; this phase's new dependencies are all Go modules, so that specific tool could not run (`Usage: gsd-tools package-legitimacy check --ecosystem <npm|pypi|crates>`, confirmed by invocation). Verification was instead performed directly against the authoritative Go module proxy (`proxy.golang.org`), which is the Go ecosystem's equivalent registry-of-record.

| Package | Registry | Age | Source Repo | Verdict | Disposition |
|---------|----------|-----|-------------|---------|-------------|
| `connectrpc.com/connect` | Go module proxy | Active since ~2021, latest tag 2026-05-20 | `github.com/connectrpc/connect-go` (Context7 "High" source reputation, 464 code snippets) | OK | Approved |
| `github.com/bufbuild/buf` | Go module proxy | Active, latest tag 2026-07-17 | `github.com/bufbuild/buf` | OK | Approved |
| `google.golang.org/protobuf` | Go module proxy | Already a direct dependency (`go.mod:37`) | `go.googlesource.com/protobuf` (Google-official) | OK | Already approved (existing dependency) |
| `github.com/pkg/browser` | Go module proxy | Single pseudo-version, last commit 2024-01-02 | `github.com/pkg/browser` | OK — small, feature-complete utility; zero non-stdlib-adjacent deps (`golang.org/x/sys` only) | Approved; note low release cadence is expected for a small, feature-complete utility, not a red flag (mirrored by Go's own internal `cmd/internal/browser` never needing updates either) |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none — all four packages are either already-vetted repo dependencies or verified directly against the official module proxy plus Context7-sourced official documentation, satisfying the `[VERIFIED]` provenance bar (not merely `[ASSUMED]` + registry-exists).

## Architecture Patterns

### System Architecture Diagram

```
Browser (fetch/XHR, POST bodies, Connect protocol)
   |
   |  1. HTTP request arrives on 127.0.0.1:<ephemeral port>
   v
+-------------------------------------------------------------+
| net/http.Server                                              |
|                                                               |
|  +---------------------------------------------------------+ |
|  | Origin/Host validation middleware (SRV-02)               | |
|  | - exact-match Host against {127.0.0.1, localhost, [::1]} | |
|  |   + the bound port                                       | |
|  | - exact-match Origin (when present) the same way         | |
|  | - REJECTS before mux.ServeHTTP is ever called             | |
|  +---------------------------------------------------------+ |
|                          |                                    |
|                          v                                    |
|  +---------------------------------------------------------+ |
|  | http.ServeMux                                            | |
|  |   /codegraph.ui.v1.UIService/*  -> connect-go handler    | |
|  |   (Phase 2: SPA static + fallback routes)                | |
|  +---------------------------------------------------------+ |
|                          |                                    |
|                          v                                    |
|  +---------------------------------------------------------+ |
|  | connect-go generated handler (uiv1connect)                | |
|  | - WithReadMaxBytes / WithSendMaxBytes (RPC-05 backstop)   | |
|  | - unmarshals request, dispatches to RPC impl              | |
|  +---------------------------------------------------------+ |
|                          |                                    |
|                          v                                    |
|  +---------------------------------------------------------+ |
|  | RPC implementation (new package, e.g. internal/uiserver) | |
|  | - per-call: query.OpenAt(repoRoot) -> Engine, io.Closer   | |
|  |   (opens store, snapshots, NEVER cached across calls)     | |
|  | - maps ErrStoreLocked -> CodeUnavailable+IndexingInProgress| |
|  | - maps NodeDetail/ExploreResult (Go structs) -> uiv1 msgs | |
|  | - applies line+byte truncation cap before responding       | |
|  +---------------------------------------------------------+ |
|                          |                                    |
+--------------------------|------------------------------------+
                           v
+-------------------------------------------------------------+
| internal/query.Engine (wire-agnostic, D-03)                  |
|                                                                |
|  Node(symbol,file,line) --thin wrapper--> NodeDetail(...)     |
|  Explore(query,maxFiles) --thin wrapper--> ExploreResult(...) |
|      \                                          /              |
|       \-- ONE shared gather path (D-01) --------/              |
|                          |                                    |
|                          v                                    |
|              graphstore.Reader (Pebble snapshot)               |
+-------------------------------------------------------------+
```

The CLI (`codegraph node`/`codegraph explore`) and MCP (`internal/mcp/tools.go`) continue to call `Node()`/`Explore()` directly — they are UNCHANGED consumers of the same thin wrappers, never touching `NodeDetail`/`ExploreResult` or the wire types.

### Recommended Project Structure

```
internal/
├── query/                    # existing — gains NodeDetail/ExploreResult structs + the extracted gather
│   ├── node.go                # Node() becomes a thin wrapper; gather logic extracted
│   ├── explore.go             # Explore() becomes a thin wrapper; gather logic extracted
│   └── ...
├── schema/
│   ├── graph.proto             # existing — Meta gains field 8 (commit_sha)
│   └── graph.pb.go             # regenerated, now covered by the BLD-04 drift guard
├── uiproto/                   # NEW — the UI's own proto surface (naming: discretion)
│   └── uiv1/
│       ├── ui.proto            # UIService: Search/Callers/Callees/Impact/Affected/Files/Status/NodeDetail/Explore
│       ├── ui.pb.go             # generated (protoc-gen-go)
│       └── uiv1connect/
│           └── ui.connect.go   # generated (protoc-gen-connect-go)
├── uiserver/                   # NEW — RPC implementation + Origin/Host middleware + bind/serve lifecycle
│   ├── server.go               # http.Server wiring, options struct (D-08), signal handling (D-10)
│   ├── originguard.go          # SRV-02 middleware
│   ├── handlers.go             # RPC method implementations, per-call OpenAt
│   └── truncate.go             # RPC-05 line+byte cap (mirrors internal/mcp/session_line.go's pattern)
├── mcp/
│   └── server.go               # FIX-01 fix only — pendingWriter / stdinLingerReader
└── cli/
    └── ui.go                   # NEW — `codegraph ui` cobra command
```

### Pattern 1: Thin-wrapper extraction over a single gather (ENG-01/ENG-02, D-01)

**What:** `Node()`/`Explore()` keep their exact signatures and exact string output, but their bodies become: call a new unexported gather function that returns a plain struct, then call the existing pure `RenderNode`/`RenderNodeMultiDef`/`RenderExplore` function over that struct's fields.

**When to use:** Any time a second consumer (here, the UI) needs the SAME data a rendered-string API already computes, and the existing render function is already pure (takes its inputs as plain arguments, does no I/O itself) — which `render_markdown.go`'s three functions already are.

**Example (concrete extraction for the single-def path):**
```go
// Source: internal/query/node.go:420-441 (existing, to be reshaped)
// BEFORE: renderSingleDefNode both gathers AND renders.
func (e *Engine) renderSingleDefNode(node *schema.Node) (string, error) {
	calls, err := e.fetchCalls(node)
	if err != nil {
		return "", err
	}
	rev, err := BuildReverseAdjacency(e.reader)
	if err != nil {
		return "", err
	}
	calledBy, err := e.fetchCalledBy(node, rev)
	if err != nil {
		return "", err
	}
	return RenderNode(node, calls, calledBy), nil
}

// AFTER (sketch, D-01/D-02/D-03 — struct name/fields are this phase's own design,
// not lifted verbatim from any file): the gather is now the single source of
// truth; Node() and the new NodeDetail()-exposing entry point both call it.
type NodeDetail struct {
	Node     *schema.Node
	Calls    []*schema.Node   // nil for file-only mode
	CalledBy []*schema.Node   // nil for file-only mode
	Matches  []*schema.Node   // multi-def candidates, when len > 1
	Source   []byte           // file-only mode's raw content
}

func (e *Engine) gatherSingleDefNode(node *schema.Node) (NodeDetail, error) {
	calls, err := e.fetchCalls(node)
	if err != nil {
		return NodeDetail{}, err
	}
	rev, err := BuildReverseAdjacency(e.reader)
	if err != nil {
		return NodeDetail{}, err
	}
	calledBy, err := e.fetchCalledBy(node, rev)
	if err != nil {
		return NodeDetail{}, err
	}
	return NodeDetail{Node: node, Calls: calls, CalledBy: calledBy}, nil
}

func (e *Engine) renderSingleDefNode(node *schema.Node) (string, error) {
	d, err := e.gatherSingleDefNode(node)
	if err != nil {
		return "", err
	}
	return RenderNode(d.Node, d.Calls, d.CalledBy), nil
}
```
The multi-def path (`renderMultiDefNode`, `internal/query/node.go:442-465`) and the file-only path (`readSourceFile` at `internal/query/node.go:84`, called directly from `Node()` at `internal/query/node.go:79-83`) need the identical treatment — D-02 requires `NodeDetail` to cover all three shapes, so the struct sketch above must be extended (or a sum-type/mode-tagged variant used) to also carry `Matches`/`Source` for those two cases. The exact struct shape is this phase's own design work; the constraint is D-02's coverage requirement, not a specific field layout.

### Pattern 2: Explore's gather boundary (ENG-02)

**What:** `Explore()` (`internal/query/explore.go:240-596`) is a much longer pipeline (H1–H21 stages), but the LAST ~40 lines (from `groups, symbolCount := groupMatchesByFile(...)` through the final `return RenderExplore(...)`) are exactly the render-input assembly: `groups`, `symbolCount`, `blasts`, `sources`, `stale`, `skeletonFiles`. Everything before that is the gather; the extraction boundary is "stop right before `RenderExplore` is called" — identical in spirit to the Node case.

**Example (the exact stop-before point, verbatim from the current source):**
```go
// Source: internal/query/explore.go:568-596 (the render-input assembly a
// gatherExplore(...) function would return as an ExploreResult struct)
groups, symbolCount := groupMatchesByFile(ranked, maxFiles)
if len(groups) == 0 {
	return exploreZeroResult(query, stale), nil
}

rev, err := BuildReverseAdjacency(e.reader)
if err != nil {
	return "", err
}

blasts := make([]exploreBlast, 0, symbolCount)
for _, g := range groups {
	for _, n := range g.Symbols {
		bl, err := e.buildBlastEntry(n, rev)
		if err != nil {
			return "", err
		}
		blasts = append(blasts, bl)
	}
}

sources := make(map[string][]byte, len(groups))
for _, g := range groups {
	content, err := e.readSourceFile(g.Path)
	if err != nil {
		return "", err
	}
	sources[g.Path] = content
}

implementsIdx, err := BuildImplementsIndex(e.reader)
if err != nil {
	return "", err
}
skeletonFiles := computeSkeletonFiles(groups, centralFiles, implementsIdx)

return RenderExplore(query, len(groups), symbolCount, groups, blasts, sources, stale, skeletonFiles), nil
```
`exploreZeroResult(query, stale)` (the zero-match early return, appearing at several points earlier in the function, e.g. `internal/query/explore.go:293`) is itself a rendered STRING already — the zero-result case needs its own handling in the `ExploreResult` struct (e.g. an explicit "no matches" mode field) since RPC-05's caller still needs a structured answer, not a markdown string, for that case too.

**Non-trivial parts identified for planning:**
- `BuildReverseAdjacency` (`internal/query/traverse.go:32`) is shared across the multi-def node path AND explore's blast-radius computation — the extraction must not duplicate this O(edges) full scan per candidate; both existing call sites already build it once and pass it down, and the extraction must preserve that.
- `exploreZeroResult` and the several early `return ..., nil` zero-match branches throughout `Explore()` (lines ~293, ~301, ~356, ~404, ~466, ~474 per the gather stages) all need a corresponding `ExploreResult` "empty" representation — this is not a single stop-point extraction the way Node's is; Explore's early-return branches must all funnel into the same structured type.
- `centralFiles`/`skeletonFiles` (H19/H20) are computed from state assembled just before the render call and are cheap to include in the gathered struct as-is.

### Anti-Patterns to Avoid

- **Building a parallel query path for the UI:** D-01 explicitly forbids this — the UI must go through the SAME `NodeDetail`/`ExploreResult` gather CLI/MCP's thin wrappers now also use, never a bespoke Engine method.
- **Coupling `internal/query` to the wire format:** D-03 forbids `NodeDetail`/`ExploreResult` from being generated proto types — mapping to `uiv1` messages is the RPC layer's job, keeping `internal/query` a dependency of the wire layer, never the reverse.
- **Treating `connect.WithSendMaxBytes`/`WithReadMaxBytes` as sufficient on their own:** they return `CodeResourceExhausted` (a hard error) when exceeded — they are a BACKSTOP, not the application-level truncation D-11/D-12 require. Both must exist; the transport limit must sit strictly above the application cap or it silently converts "working truncation" into "an error the user sees for no reason" (`connectrpc.com/connect` `envelope.go`, cited in canonical_refs).
- **Reaching for `connect.WithInterceptors` for Origin/Host validation:** an interceptor runs AFTER `net/http` has already routed and begun parsing the request — CONTEXT requires the check to run "before any handler runs." Use a plain `net/http` middleware (`http.Handler` wrapping the whole `mux`), which is also the only way to protect Phase 2's static SPA routes with the identical guard.
- **Copying the `http.Protocols`/`SetUnencryptedHTTP2` snippet uncritically:** that pattern (surfaced by Context7's own connect-go docs) exists specifically "for gRPC clients" that need unencrypted HTTP/2. A browser `fetch()` client speaking the Connect protocol over plain HTTP/1.1 does not need it — `codegraph ui`'s server is a plain `http.Server{Handler: mux}` with no `Protocols` field set. Setting it anyway is not wrong, just unnecessary complexity/attack surface for a loopback-only server with no gRPC client ever expected.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Cross-platform "open URL in browser" | A `switch runtime.GOOS { case "darwin": exec.Command("open", ...) ... }` block | `github.com/pkg/browser` | Zero-dependency, already the pattern Go's own tooling (`cmd/internal/browser`) uses; three OS branches plus WSL detection is exactly the kind of "looks trivial, has edge cases" surface (WSL needs `wslview` or `cmd.exe /c start`, not `xdg-open`) a maintained package has already worked out |
| RPC message size limiting | A custom `io.LimitReader` wrapped around the request/response body | `connect.WithReadMaxBytes` / `connect.WithSendMaxBytes` | Connect's own envelope-aware limiting understands the wire framing (length-prefixed messages) and returns the correct `CodeResourceExhausted` semantics; a naive byte-count wrapper risks truncating mid-frame rather than rejecting cleanly |
| Store-lock retry/backoff | A second retry loop in the RPC handler around `query.OpenAt` | `graphstore.Open`'s existing bounded retry (`openLockRetryAttempts`/`openLockRetryBackoff`, `internal/graphstore/pebble_store.go:67-80`) | D-15 explicitly forbids a second retry layer — it would change the exact condition SRV-04 describes (a re-index that OUTLASTS the existing budget) and make the degrade path harder to demonstrate/test |
| Protobuf codegen invocation | A hand-written shell script calling `protoc` directly with manually-tracked plugin paths | `buf generate` driven by a committed `buf.gen.yaml` | `buf` understands import resolution, plugin versioning, and produces reproducible output across machines without each contributor needing a correctly-versioned `protoc` on PATH |

**Key insight:** every "don't hand-roll" item above already has an existing, in-repo precedent this phase should extend (the retry loop, the truncation-constant pattern) or a tiny, singly-focused external package (browser-open) rather than a bespoke reimplementation — consistent with this repo's own stated preference for minimal, audited dependencies over hand-rolled equivalents.

## Common Pitfalls

### Pitfall 1: The golden suite does not currently prove byte-identity — D-04 needs a new test, not just an extraction

**What goes wrong:** A plan that only extracts the gather and runs `task test:golden` will pass — even with a broken extraction — because the current suite (`TestReFrozenGoldensValid`) checks JSON-envelope shape only.

**Why it happens:** The byte-diff capture path that WOULD have caught this was explicitly retired in FIXT-04 (`testdata/golden/behavioral_test.go:685-691`'s own doc comment: "not byte-diffs against a frozen golden... formerly diffed against are gone as of this phase"). What remains is a reproducibility-by-regeneration story (`task golden:regen` + a human/CI `git diff`), not a `go test`-time comparison.

**How to avoid:** Add a new test (or extend `TestReFrozenGoldensValid`) that, for each of the 26 `(corpus, filename)` pairs enumerated in `expectedGoCaptures` (`testdata/golden/golden_test.go:169-224`), builds an `Engine` over the matching corpus (`buildEngineAt`, `testdata/golden/behavioral_test.go:225`), invokes the correct `Node`/`Explore`/MCP-tool call using the SAME symbol/query parameters `gocapture/main.go`'s `lockedCorpusArgs` table already encodes (`testdata/golden/gocapture/main.go:63-99`), and asserts `got == want` against the golden's `Output` field byte-for-byte. Run with `go test -v -run <name>`, count `--- PASS` lines (26 expected when the gather is intact), apply the deliberate mutation, re-run, and record the new (lower) count before reverting.

**Warning signs:** A plan step that says "run `task test:golden` to prove goldens are unaffected" without ALSO adding this new comparison test is not proving what D-04 claims.

### Pitfall 2: FIX-01 and the toolslist-repeat flake are NOT the same bug — do not "fix" one and assume it resolves the other

**What goes wrong:** CONTEXT folds the flake into FIX-01 on the hypothesis that both are "server-initiated writes racing client-initiated responses." Reading the actual go-sdk source shows this is false for the flake's specific scenario.

**Why it happens:** `pendingWriter.Write` (`internal/mcp/server.go:339-343`) decrements on every write including server-initiated notifications never counted by `stdinLingerReader` (`internal/mcp/server.go:276`) — a real bug, but it only affects PREMATURE EOF / lost-response-at-shutdown timing, not response ORDERING mid-session. The flake's scenario (`test/wireoracle/scenarios.go:559-573`) sends two plain `tools/list` calls, no notifications in play at all. The actual cause, verified by reading `github.com/modelcontextprotocol/go-sdk@v1.7.0` from the local module cache: `mcp/server.go:1908-1913` calls `jsonrpc2.Async(ctx)` for every call except `initialize` (citing upstream issue `go-sdk#26`), and `internal/jsonrpc2/conn.go`'s `handleAsync` (line ~644) dequeues and starts the NEXT queued request in a NEW goroutine as soon as the current one calls `Async` — meaning two pipelined `tools/list` calls run CONCURRENTLY, in separate goroutines, with no guarantee which one's response write reaches the transport first. This is intentional go-sdk behavior, not a corruption bug.

**How to avoid:** Fix `pendingWriter`'s counter bug on its own merits (it is real and independently confirmed). Separately, build a concurrency-focused reproduction for the flake (two pipelined same-method calls under goroutine-scheduling pressure) and demonstrate that it reproduces WITHOUT any notification traffic present — proving the two are unrelated. The likely correct long-term fix for the flake is at the wire-oracle harness level (compare responses by JSON-RPC `id`, not arrival position) or documenting that response ordering for pipelined calls is not a protocol guarantee — not a `pendingWriter` change.

**Warning signs:** A plan task that says "fix `pendingWriter`; this also resolves the toolslist-repeat flake" without a separate reproduction proving the flake persists (or doesn't) after the `pendingWriter` fix alone.

### Pitfall 3: D-16's Status degrade path does not exist today — it needs new plumbing, not a reused fallback

**What goes wrong:** Assuming `Engine.Status()`'s existing degrade-safely comments (e.g. `computeStale`'s "an Engine with no repoRoot... reports not stale rather than erroring") already cover "the store is locked."

**Why it happens:** `Engine.Status()` (`internal/query/status.go:247`) is called only AFTER a successful `Engine` construction — and today, EVERY caller (CLI, MCP's `codegraph_status` tool at `internal/mcp/tools.go:519-520`) reaches it via `openEngine`/`query.OpenAt` (`internal/query/engine.go:160-196`), which calls `graphstore.Open` and returns an error immediately if the store is locked past the retry budget. `Status()` itself never runs in that case today — there is no existing fallback to reuse.

**How to avoid:** The UI RPC's Status handler must catch the specific `errors.Is(err, graphstore.ErrStoreLocked)` case from `query.OpenAt` BEFORE constructing an `Engine`, and build a degraded response from filesystem-only facts: `query.ResolveCodegraphDir` (`internal/query/resolve.go:31`) succeeding is independent of the store lock (it only `os.Stat`s the `.codegraph/` directory), so repo-path/store-exists/lock-held/"indexing in progress" are all answerable without ever calling `graphstore.Open` successfully. This is new code, not a reused path.

**Warning signs:** A plan task for SRV-04/D-16 that only modifies `internal/query/status.go` — the actual degrade must live in the NEW RPC-layer Status handler, since the existing `Engine.Status()` signature requires an already-open `Engine`.

### Pitfall 4: The tool-modfile MVS conflict precedent may recur for `buf`

**What goes wrong:** Assuming `buf` (and the protoc-gen-go/protoc-gen-connect-go generators) can simply be added to the existing `go.tool.mod` or `go.tool-lint.mod` without measurement.

**Why it happens:** This repo has ALREADY hit this exact class of problem once: `go.tool-lint.mod`'s own header documents that co-locating `actionlint` with `task`/`goreleaser` failed to compile (`action_metadata.go:273:22: te.Errors[0].Error undefined`) because Go's MVS resolves one version per module across the whole graph, and `actionlint`'s expected YAML API lost that version bid. `buf` v1.72.0 has a large, opinionated dependency tree (protovalidate, cobra, its own YAML handling) that has NOT been measured against either existing tool modfile in this session.

**How to avoid:** Before deciding where to pin `buf`/the generators, create a scratch copy of the target modfile, add the `tool` directive, and run `go mod tidy` — exactly the measurement `go.tool.mod`'s own header documents doing for `task`/`goreleaser`/`govulncheck`. If either existing modfile shows a version-bid conflict, a THIRD isolated modfile (e.g. `go.tool-proto.mod`) is the established, already-precedented resolution.

**Warning signs:** A build failure with an "X undefined" error inside a generated or vendored file after adding a `tool` directive — this repo has seen this exact failure mode before and has a name for it (MVS version-bid loss).

### Pitfall 5: Origin header is absent on GET requests — do not reject on its absence uniformly

**What goes wrong:** A validator that requires `Origin` to be present on every request will reject legitimate GET requests (e.g. a browser loading the SPA's static assets, or navigating directly to the printed URL) since browsers only reliably send `Origin` on "unsafe" methods (POST, PUT, DELETE, etc.) and on genuinely cross-origin requests — not on same-origin navigation/GET.

**Why it happens:** Connect protocol unary RPCs are typically POST (so `Origin` WILL be present and must be exact-matched), but any GET-served paths (Phase 2's static SPA, or a future health-check endpoint) may see no `Origin` header at all on a legitimate same-origin browser navigation.

**How to avoid:** The Host header (always present, HTTP/1.1 mandatory) is the PRIMARY defense and must always be exact-matched. Origin, when PRESENT, must also be exact-matched — but its absence on a GET request is not itself a rejection reason. This matches NCC Group's and the MCP-SDK CVEs' documented mitigation shape: Host-header validation is what actually stops DNS rebinding (the attacker's page IS same-origin to itself, so Origin from the attacker's perspective looks "valid" for THEIR origin — Host is what reveals the rebind, since the browser sends the REBOUND hostname as Host while still connecting to the attacker-controlled IP that now resolves to the loopback target).

## Code Examples

### Origin/Host exact-match middleware (SRV-02)

```go
// Illustrative sketch — not verbatim from any existing file. Exact-match
// only; no strings.Contains/HasPrefix/regex, per the CVE mitigation shape
// (NCC Group's Ollama advisory: "this set of whitelisted host values should
// only contain localhost, and all reserved numeric addresses for the
// loopback interface, including 127.0.0.1").
func originHostGuard(port string, next http.Handler) http.Handler {
	allowed := map[string]bool{
		"127.0.0.1:" + port: true,
		"localhost:" + port:  true,
		"[::1]:" + port:      true,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowed[r.Host] {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			ok := origin == "http://127.0.0.1:"+port ||
				origin == "http://localhost:"+port ||
				origin == "http://[::1]:"+port
			if !ok {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
```

### connect-go handler mounting, plain HTTP/1.1, no h2c (RPC-02)

```go
// Source: pattern confirmed via Context7 (connectrpc/connect-go README),
// adapted to drop the "for gRPC clients" http.Protocols/SetUnencryptedHTTP2
// block this project does not need (see Anti-Patterns).
mux := http.NewServeMux()
mux.Handle(uiv1connect.NewUIServiceHandler(
	&uiServer{repoRoot: repoRoot},
	connect.WithReadMaxBytes(maxReadBytes),   // RPC-05 transport backstop (D-13)
	connect.WithSendMaxBytes(maxSendBytes),   // RPC-05 transport backstop (D-13)
))

ln, err := net.Listen("tcp", "127.0.0.1:0") // D-07: ephemeral port
if err != nil {
	return err
}
port := ln.Addr().(*net.TCPAddr).Port

srv := &http.Server{Handler: originHostGuard(strconv.Itoa(port), mux)}

ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM) // D-10, mirrors internal/cli/daemon.go:142-143
defer stop()

go func() { _ = srv.Serve(ln) }()
<-ctx.Done()
shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
_ = srv.Shutdown(shutdownCtx)
```

### Typed error detail for D-14 (`CodeUnavailable` + `IndexingInProgress`)

```go
// Source: pattern confirmed via Context7 (connectrpc/connect-go
// _autodocs/api-reference/error.md's NewErrorDetail example), adapted to
// this phase's own message type instead of google.golang.org/genproto's
// errdetails.BadRequest.
detail, err := connect.NewErrorDetail(&uiv1.IndexingInProgress{
	Message: "the graph store is locked by another codegraph process (an in-flight sync); try again shortly",
})
if err != nil {
	return nil, connect.NewError(connect.CodeInternal, err)
}
return nil, connect.NewError(connect.CodeUnavailable, errors.New("store locked")).WithDetails(detail)
```

### Rune-safe line+byte truncation (RPC-05, D-11), mirroring the existing MCP pattern

```go
// Existing precedent this phase's truncation MUST mirror the discipline of
// (named constant, rune-boundary cut): internal/mcp/session_line.go:18-85
const (
	sourceLineCap = 2000 // primary cap (D-11) — exact value is this phase's discretion
	sourceByteCap = 512 * 1024 // secondary cap under a pathological single-line file
)

func truncateOnRuneBoundary(s string, maxBytes int) string { // reusable from internal/mcp/session_line.go:78-85
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| gRPC over HTTP/2 (requires h2c or TLS for browser clients) | Connect protocol over plain HTTP/1.1 (works in any browser via `fetch`, no h2c needed) | connect-go's entire design premise since inception; reconfirmed current in v1.20.0 docs | This project's UI never needs an h2c wrapper or a TLS cert for local loopback serving |
| MCP go-sdk versions prior to the async-call default | `jsonrpc2.Async(ctx)` called automatically for every call except `initialize` (`go-sdk#26`) | Landed by the time this repo adopted v1.7.0 (already pinned) | Pipelined same-method calls are NOT guaranteed to respond in request order — a documented behavior change from a naive single-goroutine dispatch model, directly relevant to the toolslist-repeat flake's root cause |
| TS-era byte-diff golden capture (compare against an external TypeScript reference tool) | Go-native property/shape tests + a regenerate-and-eyeball-diff workflow (FIXT-04) | v0.11.0-era migration (retired the TS-parity byte-diff harness) | The byte-identity SAFETY NET this phase's D-04 assumes exists must be partially rebuilt, not merely relied upon (see Pitfall 1) |

**Deprecated/outdated:**
- The TS-parity Compatibility constraint that originally justified byte-diffing against an external reference tool was formally retired 2026-08-13 (per `PROJECT.md`/`STATE.md`) — the goldens now exist purely as this project's OWN regression oracle, which is exactly why they must be provably capable of catching a regression (D-04), not just "look plausible."

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The recommended `internal/uiproto/uiv1` + `internal/uiserver` package layout and naming | Architecture Patterns → Recommended Project Structure | Low — CONTEXT explicitly leaves package naming to planning discretion; a different name has zero functional impact |
| A2 | `NodeDetail`/`ExploreResult`'s exact field names and whether Explore's zero-match case is a struct field vs. a separate sum-type variant | Pattern 1 / Pattern 2 code sketches | Low-medium — CONTEXT explicitly reserves exact field names as discretion; but the STRUCTURE (must cover file/single/multi-def for Node; must cover the zero-match case for Explore) is a hard D-02 requirement the sketch must satisfy regardless of naming |
| A3 | `buf` will MVS-conflict (or not) with `go.tool.mod`/`go.tool-lint.mod` — NOT measured live this session | Alternatives Considered / Pitfall 4 | Medium — if unmeasured and simply added to an existing modfile, could reproduce the exact `actionlint`-class build failure this repo has already hit once; mitigated by explicitly flagging the required measurement step |
| A4 | `sourceLineCap`/`sourceByteCap` exact numeric values in the RPC-05 code example | Code Examples | Low — CONTEXT explicitly reserves these as discretion; the example values are illustrative placeholders, not researched-and-verified thresholds |
| A5 | The toolslist-repeat flake's ultimate correct fix is at the wire-oracle harness level (compare by `id`, not position) rather than a server-side change | Pitfall 2 | Low-medium — this is a recommendation based on reading go-sdk's documented async-by-default behavior, not a verified maintainer ruling; the planner should treat "harness-level fix" as the leading hypothesis to verify, not a locked conclusion |

**If this table is empty:** N/A — see rows above.

## Open Questions

1. **Where should `buf`/`protoc-gen-go`/`protoc-gen-connect-go` be pinned?**
   - What we know: This repo has an established, documented precedent (`go.tool.mod` vs `go.tool-lint.mod`) for splitting tool modfiles on MVS conflict, with the exact failure mode already characterized (`actionlint`'s YAML API losing its version bid).
   - What's unclear: Whether `buf` v1.72.0's dependency tree actually conflicts with either existing modfile — this requires a live `go mod tidy` measurement in a scratch copy, which this research session did not perform (no destructive repo mutation).
   - Recommendation: The planner's first proto-tooling task should include this measurement as an explicit step, with a fallback to a new `go.tool-proto.mod` if either existing modfile shows a conflict.

2. **Exact shape of the BLD-04 drift guard: tempdir-diff vs. in-place regenerate-and-`git diff --exit-code`?**
   - What we know: `corpora:drift` (`Taskfile.yml:3639-3677`) is the closest existing precedent (regenerate, then `git diff --name-only`, fail if non-empty) but does NOT report a positive count of files compared — only lists changed files. The `taskfile_shape_test.go` idiom (Go tests parsing Taskfile/workflow shape, with a companion `_EmptyXIsError` test) is the OTHER established pattern, used for structural assertions rather than content regeneration.
   - What's unclear: Whether BLD-04 should be a Taskfile `cmds:` script (matching `corpora:drift`, adapted to print `echo "compared N generated files"` before the diff check) or whether it also needs a Go-level shape test asserting the task's existence/structure (matching the `taskfile_shape_test.go` idiom used for the OTHER 19 guards CONTEXT references).
   - Recommendation: Do both — a Taskfile task that regenerates both proto surfaces into the SAME location and diffs (positive count printed via `git diff --name-only | wc -l` compared against the total generated-file count, not just exit code), plus a companion Go shape test proving the task exists and is wired into a CI-invoked workflow step, matching this repo's dual-layer guard convention.

3. **Exact numeric values for RPC-05's line cap / byte cap / transport backstop.**
   - What we know: The pattern (named constant, referenced from call site and test) is established by `internal/mcp/session_line.go`'s `clientFieldMaxBytes = 256`, but that value is sized for a diagnostic log LINE, not a verbatim source file — an unrelated scale.
   - What's unclear: What line/byte counts are appropriate for a source-viewer response (likely orders of magnitude larger than 256 bytes — probably low thousands of lines / hundreds of KB, similar to typical "large file" editor warnings).
   - Recommendation: CONTEXT already marks this as Claude's discretion; pick values conservatively large enough not to visibly truncate the vast majority of real source files (most files are under a few thousand lines), verified against the actual corpora already in this repo (`corpora/observations.json` likely has real file-size distributions from the locked corpora — worth checking before picking a number).

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `buf` CLI | Proto codegen orchestration (BLD-04, RPC-01) | ✓ | 1.72.0 (locally installed; matches latest module-proxy version) | — |
| `protoc` (C++ binary) | Alternative to `buf` if buf is rejected | ✓ | libprotoc 35.1 | Not needed if `buf` is adopted (recommended) |
| `protoc-gen-go` | protobuf Go codegen | ✓ | v1.36.11 (locally installed; matches the main module's pinned protobuf runtime version exactly) | — |
| `protoc-gen-connect-go` | Connect handler codegen | ✓ | 1.19.1 locally installed (module itself is now at 1.20.0 — recommend upgrading the local binary when pinning) | — |
| Go toolchain | Everything in this phase | ✓ | go1.26.7 darwin/arm64 | — |

**Missing dependencies with no fallback:** none — all required tooling is already present in the development environment.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go's standard `testing` package (`go test`), no third-party test framework anywhere in this repo |
| Config file | none — `Taskfile.yml` defines the invocation surface (`task test:unit`, `task test:golden`, `task test:race`, etc.) |
| Quick run command | `go test ./internal/query/... ./internal/mcp/... -run <TestName> -v` |
| Full suite command | `task test:unit && task test:golden` (golden suite is excluded from `go list ./...`, per `testdata/golden/README.md`'s own documented Go-tooling quirk) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|--------------------|--------------|
| ENG-01 | `Node()` byte-identical after extraction | golden (byte-identity, NEW) | `go test -v -run TestNodeGoldensMatchEngine ./testdata/golden/...` | ❌ Wave 0 — must be added (Pitfall 1) |
| ENG-02 | `Explore()` byte-identical after extraction | golden (byte-identity, NEW) | `go test -v -run TestExploreGoldensMatchEngine ./testdata/golden/...` | ❌ Wave 0 — must be added (Pitfall 1) |
| ENG-04 | `Meta` field 8 additive, absent degrades gracefully | unit | `go test ./internal/schema/... -run TestMeta` | ❌ Wave 0 — new test for the new field |
| RPC-01/RPC-02 | connect-go handlers dispatch correctly, mount on `net/http` | integration | `go test ./internal/uiserver/... -run TestUIService` | ❌ Wave 0 — new package |
| RPC-05 | Bounded response, `truncated` field set correctly, rune-safe cut | unit | `go test ./internal/uiserver/... -run TestTruncate` | ❌ Wave 0 |
| SRV-02 | Origin/Host rejection, exact-match not substring | unit (demonstrated RED first, per CONTEXT) | `go test ./internal/uiserver/... -run TestOriginHostGuard` | ❌ Wave 0 |
| SRV-04 | Locked store renders as "indexing in progress", never an error | integration | `go test ./internal/uiserver/... -run TestStatusDegradesOnLock` | ❌ Wave 0 |
| BLD-04 | Both proto surfaces regenerate with zero diff | shape + drift task | `task proto:drift` (new) + `go test ./internal/upgrade/... -run TestProtoTaskExists` | ❌ Wave 0 — both new |
| FIX-01 | `pendingWriter` counter never corrupted by notifications | unit (demonstrated RED first) | `go test ./internal/mcp/... -run TestPendingWriter` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** the specific new/touched test file(s) above
- **Per wave merge:** `task test:unit && task test:golden`
- **Phase gate:** Full suite green before `/gsd-verify-work`, PLUS the D-04 mutation-proof procedure run once and its PASS/FAIL count recorded in the phase's own evidence artifact

### Wave 0 Gaps
- [ ] A byte-identity comparison test over the 26 golden `(corpus, filename)` pairs — does not exist today (Pitfall 1); this is the single highest-priority Wave 0 item since ENG-01/ENG-02's own success criterion cannot otherwise be proven
- [ ] `internal/uiserver` package and its test files — entirely new
- [ ] A concurrency-focused reproduction test isolating the toolslist-repeat flake from any notification traffic (Pitfall 2)
- [ ] `internal/schema` test asserting `Meta` field 8 round-trips and degrades gracefully when absent

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V1 Architecture | yes | Read-only-by-construction Engine (no mutator on `graphstore.Reader`); bind/auth as explicit unwired seams (D-08) rather than exposed flags |
| V4 Access Control | yes | Exact-match Host/Origin allowlist as network-boundary access control — the CVE-2024-28224/CVE-2025-66414/66416 mitigation shape |
| V5 Input Validation | yes | connect-go's protobuf schema validation (typed messages, no free-form JSON parsing of untrusted structure); RPC-05's bounded response sizes |
| V9 Communication | yes | Loopback-only bind (`127.0.0.1:0`), no TLS needed for a same-host-only server, but the DNS-rebinding class of attack is exactly why loopback bind ALONE is insufficient (SRV-02's stated rationale) |
| V6 Cryptography | no | No cryptographic operations in this phase's scope (no auth tokens, no TLS termination) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| DNS rebinding against a loopback-bound HTTP server (CVE-2024-28224, CVE-2025-66414/66416) | Spoofing / Elevation of Privilege | Exact-match Host header allowlist (`localhost`, `127.0.0.1`, `[::1]`, each with the bound port), applied BEFORE any handler; Origin exact-matched when present |
| Unbounded response size / memory exhaustion from a verbatim-source RPC | Denial of Service | Application-level line+byte cap (D-11/D-12) PLUS `WithSendMaxBytes`/`WithReadMaxBytes` transport backstop (D-13) — two independent layers, matching this repo's existing "two layers" disposition pattern for the Explore query-rejection case (`internal/query/explore.go:236-238`'s own comment cites "two layers, per the threat model's T-01-25 disposition") |
| Path traversal via a UI-supplied file path reaching verbatim source read | Tampering / Information Disclosure | MUST reuse `readSourceFile`/`resolveSourcePath` (`internal/query/node.go:33-88`), which already re-verifies confinement after symlink resolution — never a new read path (SRV-05, Phase 3, but the precedent applies equally to any Phase 1 internal call) |
| Store-lock-induced error leaking internal filesystem detail to an unauthenticated loopback caller | Information Disclosure | D-14's typed `IndexingInProgress` detail is a deliberately GENERIC message — must not embed raw OS lock-file paths or process IDs in the client-visible detail |

## Sources

### Primary (HIGH confidence)
- `connectrpc.com/connect` — Context7 library ID `/connectrpc/connect-go` (Source Reputation: High, 464 code snippets) — handler mounting pattern, `WithSendMaxBytes`/`WithReadMaxBytes`, `NewError`/`NewErrorDetail`, error code table
- `proxy.golang.org` (the authoritative Go module proxy) — live-queried during this session for `connectrpc.com/connect` (v1.20.0, 2026-05-20), `github.com/bufbuild/buf` (v1.72.0, 2026-07-17), `google.golang.org/protobuf` (v1.36.11/v1.36.12), `github.com/pkg/browser` (single pseudo-version, 2024-01-02)
- `github.com/modelcontextprotocol/go-sdk@v1.7.0` — read directly from the local Go module cache (`~/go/pkg/mod/github.com/modelcontextprotocol/go-sdk@v1.7.0/mcp/server.go:1908-1913`, `internal/jsonrpc2/conn.go:365-372,620-686`) — the `jsonrpc2.Async` call-for-every-non-initialize-call behavior that explains the toolslist-repeat flake
- In-repo source files read directly this session (file:line citations throughout): `internal/query/node.go`, `internal/query/explore.go`, `internal/query/render_markdown.go`, `internal/query/engine.go`, `internal/query/status.go`, `internal/query/resolve.go`, `internal/graphstore/pebble_store.go`, `internal/mcp/server.go`, `internal/mcp/tools.go`, `internal/mcp/session_line.go`, `internal/mcp/session_line_test.go`, `internal/schema/graph.proto`, `internal/schema/meta.go`, `internal/cli/serve.go`, `internal/cli/daemon.go`, `internal/cli/root.go`, `internal/indexer/sync.go`, `Taskfile.yml`, `testdata/golden/golden_test.go`, `testdata/golden/behavioral_test.go`, `testdata/golden/README.md`, `testdata/golden/gocapture/main.go`, `.github/workflows/corpora.yml`, `internal/upgrade/taskfile_shape_test.go`, `go.tool.mod`, `go.tool-lint.mod`

### Secondary (MEDIUM confidence)
- NCC Group technical advisory, "Ollama DNS Rebinding Attack (CVE-2024-28224)" — WebSearch-sourced, cross-referenced against multiple independent advisory aggregators (cvedetails.com, GitLab Advisory Database) in the same search pass; documents the exact mitigation shape (Host-header allowlist limited to `localhost` + reserved loopback numeric addresses)
- Multiple independent CVE-database entries (SentinelOne, Miggo, GitHub Advisory Database `GHSA-9h52-p55h-vw2f`, Red Hat Bugzilla) for CVE-2025-66414/CVE-2025-66416 — cross-referenced across 5+ independent sources in the same WebSearch pass, consistent description of the DNS-rebinding-via-fetch-same-origin-illusion mechanism

### Tertiary (LOW confidence)
- None — every claim above was either read directly from a cached source file this session, confirmed via the authoritative Go module proxy, fetched via Context7 from official library documentation, or cross-referenced across multiple independent WebSearch results in the same pass.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every package version and API claim was verified live against `proxy.golang.org` and/or Context7-sourced official docs this session, not recalled from training data
- Architecture: HIGH for the extraction boundaries (read directly from source with line citations); MEDIUM for the new package layout/naming (explicitly left to planning discretion by CONTEXT)
- Pitfalls: HIGH — Pitfalls 1, 2, and 3 are each backed by direct reads of the relevant source files (golden suite, go-sdk internals, MCP tool wiring) that overturn an assumption CONTEXT.md's own phrasing could otherwise be read to imply; these are the highest-value findings in this document
- Security (SRV-02): HIGH for the CVE facts and mitigation shape (multiple independent public sources); MEDIUM for the exact browser Origin-header-on-GET behavior (well-established web-platform behavior, not independently re-verified via a live browser test this session — flagged as A-adjacent common knowledge, not a fabricated claim)

**Research date:** 2026-08-22
**Valid until:** 30 days for the Go-ecosystem package versions (fast-moving); the CVE/security research and in-repo code citations are stable until the cited files change
