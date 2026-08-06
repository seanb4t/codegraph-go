# Phase 1: Protocol Scoping & the SDK-Independent Wire Oracle - Research

**Researched:** 2026-08-04
**Domain:** MCP wire-protocol verification tooling (Go, stdio, `go/packages` archtests) + MCP `2026-07-28` spec scoping for a stdio/tools-only server
**Confidence:** HIGH — every SDK/spec claim below was either fetched live today (WebFetch against the official spec) or read directly from the vendored module source in `$GOMODCACHE` this session; every in-repo claim was read from the actual file this session, not recalled.

## Summary

This phase builds zero new product behavior — it builds the harness and scoping documents that make every later phase in v0.3.0 verifiable. Three things came out of this session's direct source-reading that materially change how the phase should be planned, none of which appear in the prior day's `.planning/research/*.md` set:

1. **mark3labs/mcp-go v0.56.0's server side has no version-rejection path at all.** `protocolVersion()` silently coerces any client-requested version it doesn't recognize to `LATEST_PROTOCOL_VERSION` — it never returns an error. `mcp.UnsupportedProtocolVersionError` exists in the package but is constructed only in the *client* (`client/client.go:257`), never the server. This directly changes what D-06's "one unsupported version" scenario will actually capture against today's binary: a **successful** `initialize` response with `protocolVersion` silently overridden to `"2025-11-25"`, not a JSON-RPC error. The plan must capture this as-is (frozen, not hand-authored) and document it as "Legacy behavior: silent coercion," so Phase 3's Modern-era rejection path has an honest contrast baseline instead of an assumed one.
2. **mark3labs v0.56.0 never checks `session.Initialized()` before `tools/list` or `tools/call`** — only `logging/setLevel` gates on it. Today's Legacy server already tolerates a `tools/call` as the very first message with no prior `initialize`, which is incidentally the exact behavior `2026-07-28`'s statelessness requirement mandates. This is a gift for VRFY-04/D-05: the oracle's "unknown-method"/edge scenarios do not need special sequencing to be realistic.
3. **`server.Hooks.AddAfterInitialize`** (`func(ctx, id, *mcp.InitializeRequest, *mcp.InitializeResult)`) is the exact, already-shipped mechanism for VRFY-03's stderr line — it hands you the client's requested version, the negotiated (server-decided) version, and `clientInfo` in one callback, with zero new SDK surface needed. This resolves the "how" for D-13/D-14 entirely inside `internal/mcp`, no `internal/cli` changes beyond passing a writer through.

**Primary recommendation:** Build the wire oracle as a standalone Go package (not `testdata/`) that spawns the real binary via `os/exec` and reads raw `stdout` bytes with `bufio.Scanner`, generalizing `test/integration/mcp_stdout_purity_test.go`'s existing pattern (already proven, already in the tree). Implement VRFY-02's guard as a `go/packages`-based archtest that forbids referencing any **externally-defined, non-struct-field** identifier whose name matches a protocol-version pattern (`(?i)protocol.?version`), which is package-identity-agnostic and therefore automatically survives Phase 2's SDK swap without maintenance. Implement VRFY-03 via `server.WithHooks` + `AddAfterInitialize` inside `internal/mcp.BuildServer`, which is also the natural place to introduce SDK-02's `internal/mcp.Server` interface (`BuildServer` changes its return type from `*server.MCPServer` to this new interface; `internal/cli/serve.go` drops its only `mark3labs/mcp-go/server` import).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-------------------|
| VRFY-01 | Harness asserts on raw stdio wire bytes, not SDK-typed objects, and never uses the SDK under test as its own oracle | `test/integration/mcp_stdout_purity_test.go` is the exact precedent to generalize (raw `bufio.Scanner` over `cmd.StdoutPipe()`, hand-framed `map[string]any` requests). See Architecture Patterns and Code Examples below. |
| VRFY-02 | Server's declared protocol version is a repo-owned literal; CI guard proves no stray SDK-owned constant reference remains | Concrete `go/packages` archtest design below (package-identity-agnostic predicate), built on the `internal/graphstore/archtest` and `internal/cli/present/archtest` precedents read this session. |
| VRFY-03 | Server logs negotiated protocol version to stderr on every connection, always on | `server.Hooks.AddAfterInitialize` (mark3labs v0.56.0, confirmed via module source) is the exact mechanism — no new SDK API needed. See Code Examples. |
| VRFY-04 | Harness passes against the current, pre-migration server before any SDK change lands | Structural: the harness is written and run against today's `mark3labs`-backed binary in this phase; Phase 2 must not modify it. |
| VRFY-05 | Dated audit records which protocol revision each of the 8 roster agent clients negotiates, measured not read from docs | Local-machine audit performed this session (see Environment Availability) + a concrete proxying-shim design (see Architecture Patterns § VRFY-05 shim). |
| SDK-02 | `internal/cli/serve.go` imports no MCP SDK package; bootstraps through a narrow `internal/mcp.Server` seam | Concrete interface design below, reconciled with VRFY-03's stderr hook so both land at the same `BuildServer` touch point. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Raw-wire capture tool (spawn binary, read stdio bytes) | Test/verification tooling (new) | — | Standalone process driver; not part of the production binary |
| Repo-owned protocol-version literal + CI guard | Production (`internal/mcp`) | Test/CI tooling (archtest) | The literal lives in production code; the guard is a `go test`-run archtest |
| Negotiated-version stderr line | Production (`internal/mcp`) | — | Wired at `BuildServer` construction time via SDK hooks; stdio process's own diagnostic output |
| `internal/mcp.Server` seam | Production (`internal/mcp` ↔ `internal/cli`) | — | Package-boundary fix; no new tier, closes an existing cross-package leak |
| 8-agent capture shim | Test/verification tooling (new, one-off) | — | A measurement instrument installed temporarily into each agent's own MCP config, not shipped product code |
| SEP applicability table + Team Scale read-out | Documentation (`docs/`, `.planning/`) | — | Per D-12, no code; pure scoping artifacts |

## Standard Stack

### Core

No new external dependencies are required for this phase.

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|---------------|
| `golang.org/x/tools/go/packages` | v0.48.0 (already required) [VERIFIED: go.mod:36] | Whole-tree AST/type-resolved scanning for VRFY-02's archtest | Already the house pattern for every existing archtest in this repo (`internal/graphstore/archtest`, `internal/cli/present/archtest`) — read directly this session |
| stdlib `os/exec`, `bufio`, `encoding/json`, `net` (none new) | Go 1.26.5 (go.mod:3) | The wire oracle's spawn/read/write loop | `test/integration/mcp_stdout_purity_test.go` already does exactly this with zero non-stdlib imports beyond the one `mcp.LATEST_PROTOCOL_VERSION` reference this phase removes |
| `github.com/mark3labs/mcp-go` | v0.56.0 (pinned, unchanged this phase) [VERIFIED: go.mod:13] | The server under test — VRFY-04 requires it stay exactly as-is | Confirmed still current: `go list -m -versions` run today shows v0.57.0 is the latest tag; no `2026-07-28` support has shipped since yesterday's research [VERIFIED: `go list -m -versions github.com/mark3labs/mcp-go`, run 2026-08-04] |

### Supporting

None. This phase deliberately does not touch `go.mod`.

### Alternatives Considered

Not applicable — no library decision is being made this phase (that is Phase 2's SDK-02/SDK-03 territory, already pre-decided per STATE.md's "Adopt `github.com/modelcontextprotocol/go-sdk@v1.7.0`").

**Installation:** none.

**Version verification (performed this session):**
```
$ go list -m -versions github.com/mark3labs/mcp-go
... v0.54.1 v0.55.0 v0.55.1 v0.56.0 v0.57.0
$ go list -m -versions github.com/modelcontextprotocol/go-sdk
... v1.6.1 v1.7.0-pre.1 v1.7.0-pre.2 v1.7.0-pre.3 v1.7.0
```
Both match the `.planning/research/STACK.md` findings from 2026-08-03 exactly — no drift in 24 hours. `mark3labs/mcp-go@v0.57.0` still has no `2026-07-28` support (Pitfall 3 stands unchanged).

## Package Legitimacy Audit

**Not applicable this phase.** No new external packages are introduced. `golang.org/x/tools` and `github.com/mark3labs/mcp-go` are both pre-existing, already-audited dependencies (verified present in `go.mod` this session, unchanged versions).

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│  Wire Oracle (NEW, standalone tool — VRFY-01/D-01)                  │
│  1. exec.Command(binPath, "serve", "--mcp") — spawns REAL binary     │
│  2. StdinPipe(): writes hand-framed JSON-RPC (map[string]any),       │
│     one scripted scenario at a time (~18 total, D-05)                │
│  3. StdoutPipe() + bufio.Scanner: reads EVERY line as raw bytes,      │
│     never through mcp-go's client (Trap A/B — the whole point)       │
│  4. Named-field placeholder substitution (D-04): <REPO>, <VERSION>,  │
│     <TS> — everything else byte-verbatim                             │
│  5. Emits a normalized transcript to STDOUT — a human redirects it   │
│     to testdata/wireoracle/<scenario>.golden (no --regenerate flag)  │
└───────────────────────────────┬───────────────────────────────────┘
                                 │ spawns / drives via real stdio
┌────────────────────────────────▼───────────────────────────────────┐
│  codegraph serve --mcp  (UNCHANGED this phase except SDK-02/VRFY-03) │
│   internal/cli/serve.go: mcp.BuildServer(...) → mcp.Server interface │
│                          (NEW — was *server.MCPServer, SDK-02)       │
│                          s.ServeStdio() (NEW — was                  │
│                          server.ServeStdio(s) in serve.go itself)    │
└────────────────────────────────┬───────────────────────────────────┘
                                 │
┌────────────────────────────────▼───────────────────────────────────┐
│  internal/mcp.BuildServer (production, mark3labs-backed, unchanged  │
│  wire behavior)                                                       │
│   server.NewMCPServer(..., server.WithHooks(hooks))  ← NEW (VRFY-03) │
│   hooks.AddAfterInitialize(func(...) {                               │
│     fmt.Fprintf(stderr, "codegraph: mcp-session requested=%s          │
│       negotiated=%s client=%s/%s tools=%d\n", ...)                    │
│   })                                                                  │
└──────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│  VRFY-02 archtest (separate from the oracle — runs under `go test`)  │
│  packages.Load(cfg, "github.com/seanb4t/codegraph-go/...",           │
│                Tests: true)  ← whole tree, including test files       │
│  → for every *ast.SelectorExpr, resolve via info.Uses                │
│  → flag iff: object's package is NOT this module, name matches       │
│    /(?i)protocol.?version/, and object is NOT a struct field          │
│    (this excludes req.Params.ProtocolVersion field access, which     │
│    stays legitimate wire-shape usage)                                 │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│  VRFY-05 capture shim (NEW, one-off measurement instrument)          │
│  Agent (Claude Code, Codex CLI, ...) → spawns SHIM (config rewritten  │
│  to point at shim instead of `codegraph`) → shim tees the FIRST       │
│  JSON-RPC frame off stdin to a log file (protocolVersion, clientInfo, │
│  capabilities) → shim execs/proxies the REAL `codegraph serve --mcp`  │
│  transparently in both directions, so the agent keeps working         │
└─────────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
internal/mcp/
├── server.go            # BuildServer's return type changes; hooks wired here (SDK-02, VRFY-03)
├── server_version.go    # NEW — the repo-owned protocol-version literal (VRFY-02)
test/wireoracle/         # NEW — the standalone capture tool (D-01, D-08); a real Go package,
│                         #   NOT testdata/ (GOLDEN-01's own lesson: `go list ./...` skips
│                         #   any directory literally named testdata)
├── main.go               # or a library + thin cmd/ entrypoint — planner's call (Claude's Discretion)
├── scenarios.go          # the ~18 scripted requests (D-05)
├── normalize.go          # named-field placeholder substitution (D-04)
testdata/wireoracle/      # frozen transcripts + the dedicated fixture (D-08) — testdata/ is
│                         #   correct HERE because these are inert fixtures, not a package
│                         #   go test must discover and run
├── fixture/              # D-08's dedicated purpose-built source tree
├── legacy-2025-11-25.golden
├── legacy-2025-06-18.golden
├── legacy-2025-03-26.golden
├── legacy-2024-11-05.golden
├── legacy-unsupported.golden   # captures the SILENT COERCION behavior (see Pitfall 1 below),
│                               #   not an error — must be documented as such in the file/test
internal/graphstore/archtest/   # existing precedent package — OR a new
│                               #   internal/mcp/archtest/ sibling; either is consistent with
│                               #   house style (VRFY-02's guard doesn't need graphstore's
│                               #   guardedPackages closure-walk, just a flat Tests:true scan)
docs/
├── MCP-2026-07-28-SCOPING.md    # SEP-by-SEP applicability table (D-12)
├── MCP-8-AGENT-AUDIT.md         # VRFY-05's dated audit (D-12)
.planning/
├── (STATE.md or a new decision doc) # Team Scale strategic read-out (D-12) — .planning/, not docs/
```

### Pattern 1: Raw-stdio spawn-and-scan (VRFY-01/VRFY-04's oracle)

**What:** Spawn the real binary, write hand-framed JSON-RPC as `map[string]any` → `json.Marshal`, read every stdout line through a raw `bufio.Scanner`, never through any SDK's client/decoder.
**When to use:** Any assertion whose entire purpose is proving wire-format correctness (Trap A/B from `.planning/research/PITFALLS.md`).
**Example:**
```go
// Source: test/integration/mcp_stdout_purity_test.go:71-210 (this repo, read verbatim
// this session) — the oracle generalizes this exact loop, adding scenario scripting
// and normalization instead of a single purity assertion.
cmd := exec.Command(binPath, "serve", "--mcp")
stdin, _ := cmd.StdinPipe()
stdout, _ := cmd.StdoutPipe()
cmd.Stderr = &syncBuffer{} // never asserted for purity; VRFY-03's line lives here instead
// ... writeLine(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize", ...})
scanner := bufio.NewScanner(stdout)
scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
for scanner.Scan() {
    // decode into a generic map[string]any or compare raw bytes — NEVER an SDK's typed struct
}
```

### Pattern 2: Package-identity-agnostic archtest predicate (VRFY-02)

**What:** Instead of enumerating known SDK package paths (which breaks the moment Phase 2 swaps SDKs), flag any reference to an externally-defined (non-module) package-level const/var whose *identifier name* matches a protocol-version pattern, while explicitly excluding struct-field accesses (which remain legitimate — `req.Params.ProtocolVersion = "..."` must still compile).
**When to use:** VRFY-02's guard, specifically because CONTEXT.md's D-instructions require it to "forbid the class... not one library's spelling," and because a future SDK (go-sdk today keeps its version constants unexported, but that could change) must trip the same guard with zero maintenance.
**Example:**
```go
// Adapted from internal/graphstore/archtest/stdout_confinement_test.go's
// isOSStdoutRef predicate shape (read verbatim this session, lines 237-254) —
// generalized from "package == os, name == Stdout" to "package != this module,
// name matches protocolVersion, not a struct field".
var protocolVersionNamePattern = regexp.MustCompile(`(?i)protocol.?version`)

func isExternalProtocolVersionConstant(sel *ast.SelectorExpr, info *types.Info) bool {
    obj, ok := info.Uses[sel.Sel]
    if !ok || obj.Pkg() == nil {
        return false
    }
    if strings.HasPrefix(obj.Pkg().Path(), modulePathPrefix) {
        return false // this module's own repo-owned literal — never flagged
    }
    if !protocolVersionNamePattern.MatchString(obj.Name()) {
        return false
    }
    if v, ok := obj.(*types.Var); ok && v.IsField() {
        return false // struct field access (req.Params.ProtocolVersion) stays legitimate
    }
    return true // an external const, or an external package-level var, named like a
                 // protocol-version constant — exactly LATEST_PROTOCOL_VERSION and
                 // ValidProtocolVersions today, and whatever a future SDK calls its own
                 // equivalent, with zero per-SDK maintenance required
}
```
`packages.Load` config: `Mode: NeedName|NeedFiles|NeedCompiledGoFiles|NeedImports|NeedTypes|NeedSyntax|NeedTypesInfo`, `Tests: true` (this scan MUST see test files — CONTEXT.md's D-instructions and PITFALLS Pitfall 2 both name test files as the actual blast radius today: 6 known sites, all in `_test.go` files), target `"github.com/seanb4t/codegraph-go/..."` — mirroring `internal/graphstore/archtest/import_graph_test.go`'s whole-module `Tests: true` load (read verbatim this session, lines 29-45), not the six-package `NeedDeps`-closure walk `stdout_confinement_test.go` uses (that pattern is for a *reachability* question; VRFY-02 is a *did anyone write this anywhere* question — a flat `./...` scan is both correct and simpler here).

**Mutation-proof requirement (D-07/this repo's own standing rule):** prove the guard fires using `packages.Config.Overlay` to inject a synthetic file that imports the real `mcp` package and references `mcp.LATEST_PROTOCOL_VERSION` (mirroring `stdout_closure_selftest_test.go`'s exact Overlay technique, read verbatim this session) — confirm red — then confirm the guard is green against the real tree only *after* migrating the 6 known sites off the constant.

### Pattern 3: SDK hook for the negotiated-version stderr line (VRFY-03)

**What:** `server.WithHooks` + `Hooks.AddAfterInitialize` fires with both the client's requested version and the server's negotiated result — no new SDK surface, no change to `serve.go`.
**When to use:** Exactly VRFY-03; this is also the natural place to compute the D-13 `tools=` count, since `BuildServer` already knows `hasIndex`/`allowlist` at this point.
**Example:**
```go
// Source: github.com/mark3labs/mcp-go@v0.56.0/server/hooks.go:74-75 (module cache,
// read verbatim this session) — the exact, already-shipped hook signature.
type OnAfterInitializeFunc func(ctx context.Context, id any, message *mcp.InitializeRequest, result *mcp.InitializeResult)

// internal/mcp/server.go, inside BuildServer (sketch — planner decides exact plumbing):
hooks := &server.Hooks{}
hooks.AddAfterInitialize(func(_ context.Context, _ any, req *mcp.InitializeRequest, res *mcp.InitializeResult) {
    fmt.Fprintf(stderr, "codegraph: mcp-session requested=%s negotiated=%s client=%s/%s tools=%d\n",
        sanitizeLogField(req.Params.ProtocolVersion), sanitizeLogField(res.ProtocolVersion),
        sanitizeLogField(req.Params.ClientInfo.Name), sanitizeLogField(req.Params.ClientInfo.Version),
        toolCount)
})
s := server.NewMCPServer("codegraph", version, server.WithToolCapabilities(true), server.WithHooks(hooks))
```
`sanitizeLogField` is not decorative — see Common Pitfalls below (log-injection via unsanitized `clientInfo`).

### Pattern 4: SDK-02's narrow seam, reconciled with VRFY-03

**What:** `BuildServer` returns a new `internal/mcp.Server` interface instead of `*server.MCPServer`; `internal/cli/serve.go` depends only on that interface and drops its `mark3labs/mcp-go/server` import.
**When to use:** Exactly SDK-02. This is cheap specifically because Phase 1 is already touching `BuildServer`'s signature for VRFY-03 (adding a stderr writer parameter) — landing the return-type change at the same time is near-zero marginal cost.
```go
// internal/mcp/server.go — sketch, not verbatim (planner's call per CONTEXT.md's
// "Claude's Discretion: concrete type vs interface")
type Server interface {
    ServeStdio() error
}

type mark3labsServer struct{ inner *server.MCPServer }

func (m *mark3labsServer) ServeStdio() error { return server.ServeStdio(m.inner) }

func BuildServer(hasIndex bool, allowlist map[string]bool, repoPath, startPath string, stderr io.Writer) Server {
    s := server.NewMCPServer(..., server.WithHooks(hooks))
    // ... existing AddTool wiring, unchanged ...
    return &mark3labsServer{inner: s}
}
```
```go
// internal/cli/serve.go — the two lines SDK-02 must change:
//   BEFORE: s := mcp.BuildServer(hasIndex, allowlist, repoPath, start)
//           return server.ServeStdio(s)          // imports mark3labs/mcp-go/server
//   AFTER:  s := mcp.BuildServer(hasIndex, allowlist, repoPath, start, cmd.ErrOrStderr())
//           return s.ServeStdio()                 // no mark3labs import anywhere in serve.go
```
Every existing test that builds a `*server.MCPServer` directly (`server_test.go`'s `listToolNames(t, s *server.MCPServer)`, `golden_parity_test.go`'s `callExploreViaMCP`) will need its signature updated to accept `mcp.Server` — or, since those tests need the CONCRETE mark3labs type to build an in-process `mcpclient`, they may need a second, test-only accessor. This is a real, non-trivial ripple this phase's plan must account for (not just the two production lines) — flagged here so the planner budgets for it rather than discovering it mid-implementation.

### Anti-Patterns to Avoid

- **Enumerating SDK package paths in the VRFY-02 guard:** works today, silently stops working the moment Phase 2 swaps to `modelcontextprotocol/go-sdk` unless someone remembers to update an allowlist. Pattern 2 above avoids this by keying off "external to this module + name pattern + not a struct field" instead.
- **Assuming the "unsupported version" scenario produces a JSON-RPC error today:** it does not (Pitfall 1 below). Freezing an assumed error response would silently fail the very first time the oracle actually runs.
- **Building the VRFY-02 guard as a six-package reachability closure** (the `stdout_confinement_test.go` pattern): that pattern answers "is this reachable from `serve --mcp`," which is the wrong question for VRFY-02 — a stray `LATEST_PROTOCOL_VERSION` reference in a `_test.go` file is never reachable from the production binary at all, but it is exactly what PITFALLS Pitfall 2 and CONTEXT.md's D-instructions require this guard to catch. Use the flat `Tests: true`, `./...` scan (`import_graph_test.go`'s pattern) instead.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Detecting "does this identifier come from an SDK package" | A per-SDK string-prefix allowlist maintained by hand | The package-identity-agnostic predicate (Pattern 2) | Automatically survives Phase 2's SDK swap; a hand-maintained allowlist is exactly the kind of thing that gets forgotten during a migration under time pressure |
| Version-negotiation observation | A custom JSON-RPC middle-man that re-parses every message to find `initialize` | `mark3labs/mcp-go`'s already-shipped `Hooks.AddAfterInitialize` | Zero new parsing code, zero risk of misparsing a message the SDK itself already parsed correctly |
| Raw wire capture | A hand-rolled JSON-RPC framing/dispatch loop from scratch | Generalize `test/integration/mcp_stdout_purity_test.go`'s existing loop | It already exists, is already proven (this exact pattern is why v1.0 Phase 4 could catch the SDK-client-swallows-malformed-lines failure mode), and reinventing it risks reintroducing the `syncBuffer`/`os/exec` stderr-goroutine race (WR-01) it already fixed |

**Key insight:** every piece of infrastructure this phase needs already exists in miniature somewhere in this repo (the purity test, three archtest precedents, the `WarnUnknownToolsTo` stderr-writer convention). This phase is almost entirely composition and generalization, not invention — treat any design that doesn't map to an existing precedent as a signal to look harder before building it.

## Common Pitfalls

### Pitfall 1: mark3labs's server-side "unsupported version" behavior is silent coercion, not an error

**What goes wrong:** A plan that assumes `handleInitialize` rejects an unrecognized `protocolVersion` with a JSON-RPC error will freeze the wrong expected value for D-06's "one unsupported version" scenario, or write a hand-authored anchor asserting an error that never fires.
**Why it happens:** `mcp.UnsupportedProtocolVersionError` exists in the `mcp` package and is genuinely used — but only inside the **client** package (`client/client.go:257`, confirmed by `rg` against the module cache this session), never inside `server/server.go`. The server's own `protocolVersion(clientVersion string) string` method [VERIFIED: `github.com/mark3labs/mcp-go@v0.56.0/server/server.go:1197-1210`]:
```go
func (s *MCPServer) protocolVersion(clientVersion string) string {
	if len(clientVersion) == 0 {
		clientVersion = "2025-03-26"
	}
	if slices.Contains(mcp.ValidProtocolVersions, clientVersion) {
		return clientVersion
	}
	return mcp.LATEST_PROTOCOL_VERSION
}
```
An unrecognized version silently becomes `LATEST_PROTOCOL_VERSION` ("2025-11-25") — the `initialize` call still succeeds.
**How to avoid:** Capture this scenario as-is (frozen, per D-02's "captured-and-frozen for the bulk" rule) and add an explicit test-file comment documenting that this is TODAY's Legacy behavior — a silent, spec-permitted-for-Legacy downgrade — so nobody mistakes the green oracle for evidence the server validates versions. This is exactly the kind of "spec-sanctioned silent failure" PITFALLS Pitfall 4 already names, now concretely located in code.
**Warning signs:** A plan task reading "assert `-32602`/`-32004`/`-32022` for the unsupported-version scenario" — none of these fire from today's mark3labs server on this path.

### Pitfall 2: mark3labs never gates `tools/list`/`tools/call` on `Initialized()`

**What goes wrong:** A plan might assume the oracle must always send `initialize` before any tool scenario, adding unneeded sequencing complexity, or might treat "no prior initialize" as an edge case requiring a dedicated confinement.
**Why it happens:** Only `handleSetLevel` checks `clientSession.Initialized()` [VERIFIED: `github.com/mark3labs/mcp-go@v0.56.0/server/server.go:1297` — `if clientSession == nil || !clientSession.Initialized() { ... ErrSessionNotInitialized ... }`]. `handleListTools` and `handleToolCall` [VERIFIED: same file, lines 1862-1910, read verbatim this session] contain no such check.
**How to avoid:** This is good news, not a gap to patch — it means today's Legacy server already tolerates the exact "stateless first request" shape `2026-07-28` will require, and the oracle can add a "tools/call with no prior initialize" scenario cheaply as a genuine, currently-passing behavior worth locking in (it is a real, if accidental, asset for Phase 3).
**Warning signs:** A plan spending effort to "add initialize-skip handling to the oracle" — the server already handles it; only the oracle's own scripting needs the scenario, not new server code.

### Pitfall 3: Log injection via unsanitized `clientInfo` in the VRFY-03 stderr line

**What goes wrong:** `clientInfo.name`/`clientInfo.version` are attacker-influenced strings (an MCP client is, per this project's own established threat model in `tools.go`'s CR-02 doc comment, potentially "an AI agent processing attacker-influenced content"). `mcp.Implementation` [VERIFIED: `github.com/mark3labs/mcp-go@v0.56.0/mcp/types.go:648`, struct with `Name string`/`Version string`] places no length or character restriction on these fields. D-14's fixed format (`codegraph: mcp-session requested=... negotiated=... client=name/version tools=N`) is "greppable by prefix, parseable without a JSON decoder" specifically because it assumes no embedded newlines or `=` characters — a hostile or buggy client supplying `clientInfo.name = "claude-code/1.0\ncodegraph: mcp-session requested=evil"` could inject a fabricated second diagnostic line into the same stream.
**Why it happens:** The spec itself explicitly disclaims `clientInfo`'s trustworthiness ("self-reported... SHOULD NOT rely on them for security decisions" — FEATURES.md Q2, PITFALLS' own sourcing) but that framing is about protocol *decisions*, not about safe *logging* of the value — a distinct, narrower but real concern.
**How to avoid:** Strip or escape control characters (at minimum `\n`, `\r`) from `clientInfo.name`/`version` (and the requested/negotiated version strings, which are also client-supplied on the request side) before formatting the D-14 line. This is a one-line `strings.Map`/`strings.ReplaceAll` fix, not a design change — but it must be a named task, since nothing in the existing CONTEXT.md decisions mentions it.
**Warning signs:** The stderr-line format test (D-16's "format test") only checks the happy path (a well-formed `clientInfo.name`) and never exercises an adversarial value containing `\n` or `=`.

### Pitfall 4: `internal/mcp.Server`'s return-type change ripples into every test building a concrete `*server.MCPServer`

**What goes wrong:** `server_test.go`'s `listToolNames(t, s *server.MCPServer)` and `testdata/golden/golden_parity_test.go`'s `callExploreViaMCP`/`callNodeViaMCPWithArgs` (both confirmed this session to call `internalmcp.BuildServer(...)` and then `mcpclient.NewInProcessClient(s)` directly against the concrete mark3labs type) will fail to compile the moment `BuildServer`'s return type changes to an interface, unless a parallel accessor is provided.
**Why it happens:** SDK-02's fix is described in ARCHITECTURE.md as "small, surgical, near-zero cost" for the **one production call site** — that framing is correct for `serve.go` but doesn't account for the in-process test client pattern used throughout `internal/mcp` and `testdata/golden`.
**How to avoid:** Budget an explicit task for this ripple: either (a) keep a test-only exported accessor that returns the concrete mark3labs type for in-process client construction (defeats none of SDK-02's intent, since `internal/cli` still never imports it), or (b) accept that this phase's SDK-02 fix is scoped to the interface + `serve.go` only, and defer the test-client migration to Phase 2 (where the whole test surface is being rewritten off mark3labs anyway per PITFALLS' "test-surface is the actually expensive part"). Either is defensible; the plan must pick one explicitly rather than discover the compile failure mid-implementation.
**Warning signs:** `go build ./...`/`go vet ./...` failing in `internal/mcp` or `testdata/golden` immediately after the `BuildServer` signature change, with no task in the plan accounting for it.

### Pitfall 5: `go/packages` load performance for a whole-tree, `Tests: true`, `NeedDeps` scan

**What goes wrong:** Loading `"github.com/seanb4t/codegraph-go/..."` with `Tests: true` and full type/syntax info touches every package in the module (this repo has 12+ tree-sitter grammar dependencies, Charm TUI packages, Pebble, sigstore — a large closure). If VRFY-02's archtest additionally requests `NeedDeps` (to resolve every external identifier fully), this could be measurably slower than the existing six-package `stdout_confinement_test.go` guard.
**Why it happens:** `NeedDeps` fans out to the full transitive closure for type-checking purposes, not just the flat set of module packages.
**How to avoid:** Benchmark this specific `packages.Load` call in isolation before committing to it as a per-PR CI gate; if slow, `NeedDeps` may not even be required (resolving `info.Uses` for an external symbol typically needs only that symbol's own package's exported types, which `go/packages` can often supply via export data without loading the full dependency graph — verify empirically rather than assuming). This is exactly the kind of tool-location/CI-scope tradeoff CONTEXT.md already flags as Claude's Discretion ("whether all ~18 scenarios run on every PR or a narrower set gates while the full sweep runs on a schedule") — the archtest has an analogous question worth deciding explicitly.
**Warning signs:** CI wall-clock time for `go test ./...` increasing noticeably after this guard lands, with no measurement taken beforehand to attribute the delta.

## Code Examples

### Verified: mark3labs error-code constants (for D-02's hand-authored anchors)
```go
// Source: github.com/mark3labs/mcp-go@v0.56.0/mcp/types.go:455,458 (module cache, read
// verbatim this session)
METHOD_NOT_FOUND = -32601
INVALID_PARAMS   = -32602
```
These are the two codes D-02 can safely hand-author as spec anchors independent of capture — `-32601` for the unknown-method scenario (D-05), `-32602` for malformed-args (D-05). Do **not** hand-author a code for the "unsupported version" scenario (Pitfall 1) — that one must be captured-and-frozen as a success, not asserted as an error.

### Verified: current protocol-version constants (the D-06 multi-era baseline's exact source values)
```go
// Source: github.com/mark3labs/mcp-go@v0.56.0/mcp/types.go:163,166-171 (module cache,
// read verbatim this session)
const LATEST_PROTOCOL_VERSION = "2025-11-25"
var ValidProtocolVersions = []string{
	LATEST_PROTOCOL_VERSION, // "2025-11-25"
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
}
```
This confirms D-06's four named eras (`2025-11-25`, `2025-06-18`, `2025-03-26`, `2024-11-05`) are exactly `ValidProtocolVersions`' contents — the oracle's multi-era capture list requires no guesswork about which four strings to send.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|-------------------|---------------|--------|
| `initialize`/`initialized` handshake carries session state | Per-request `_meta` carries `protocolVersion`/`clientCapabilities`/`clientInfo`; no handshake at all | `2026-07-28` (published day before this milestone was scoped) | Not implemented this phase (Phase 3's work) — but the SEP table below documents exactly what stays N/A for stdio vs. what Phase 3 must build |
| `mcp.LATEST_PROTOCOL_VERSION` referenced directly in tests | A repo-owned literal, CI-guarded | This phase (VRFY-02) | Closes PITFALLS Pitfall 2 |

## SEP-by-SEP Applicability Table (backlog 999.6 deliverable)

Fetched fresh today, 2026-08-04, directly from the official spec changelog — [CITED: `modelcontextprotocol.io/specification/2026-07-28/changelog`, fetched 2026-08-04]. This is the primary deliverable for `docs/MCP-2026-07-28-SCOPING.md`; the planner should treat the table below as the content to transcribe, not merely a research summary.

| SEP / Change | What it does (changelog's own words) | Applies to codegraph-go (stdio, tools-only)? | Reason |
|---|---|---|---|
| **SEP-2567** — remove protocol-level sessions, `Mcp-Session-Id` header | "Remove protocol-level sessions and the `Mcp-Session-Id` header from the Streamable HTTP transport." | **N/A — HTTP-only** | stdio never had `Mcp-Session-Id`; there is no header layer on stdio at all |
| **SEP-2575** — stateless core: `_meta` version/capabilities, remove `initialize` handshake | "Make MCP stateless: remove the `initialize`/`notifications/initialized` handshake. Every request now carries its protocol version and client capabilities in `_meta`." | **Applicable** (Phase 3 work) | Core protocol, transport-agnostic — stdio binds the same message semantics as any transport |
| **SEP-2575** — `server/discover` | "Add `server/discover`: servers MUST implement this RPC... Clients MAY call it before any other request for up-front version selection, or use it as a backward-compatibility probe on STDIO." | **Applicable, MUST** (Phase 3 work) | Explicitly named as a stdio-relevant probe mechanism by the spec's own text |
| **SEP-2575** — `subscriptions/listen` (replaces GET/subscribe) | "Replace the HTTP GET endpoint and `resources/subscribe`/`resources/unsubscribe` with `subscriptions/listen`..." | **Applicable in principle, differentiator not table-stakes** | Transport-agnostic mechanism; codegraph-go has no `resources` capability today and `tools.listChanged` is explicitly a Phase-5, second-wave item |
| **SEP-2575** — remove `ping`, `logging/setLevel`, `notifications/roots/list_changed` | "Remove `ping`, `logging/setLevel`, and `notifications/roots/list_changed`. Log level is now set per-request via `io.modelcontextprotocol/logLevel`..." | **Applicable, already-compliant** | codegraph-go implements none of roots/sampling/logging capabilities; nothing to remove |
| **SEP-2663** — tasks extension redesign | "Move experimental tasks out of the core protocol and into an official extension (`io.modelcontextprotocol/tasks`)..." | **N/A today** | codegraph's 8 tools are fast synchronous reads; TASK-01 explicitly deferred |
| **SEP-2322** — MRTR pattern, `resultType` field | "All results now carry a required `resultType` field: `\"complete\"`... Clients MUST treat results from earlier-protocol servers that omit the field as `\"complete\"`." | **`resultType` applicable (Phase 3, SPEC-03); MRTR itself deferred (MRTR-01)** | `resultType` is a transport-agnostic MUST; the `input_required` interaction pattern is new product behavior |
| **SEP-2243** — `Mcp-Method`/`Mcp-Name` headers, `x-mcp-header` | "Require standard MCP request headers... on Streamable HTTP POST requests, and add support for custom headers from tool parameters via `x-mcp-header`." | **N/A — HTTP-only** | "stdio has no header layer" (spec text, quoted in FEATURES.md); `x-mcp-header` is explicitly ignorable on stdio per spec |
| **SEP-2549** — `CacheableResult` (`ttlMs`/`cacheScope`) | "Require `ttlMs` and `cacheScope` fields on results returned by `tools/list`... via a new `CacheableResult` interface." | **Applicable, MUST** (Phase 3, SPEC-04) | Transport-agnostic; `ttlMs: 0`/`cacheScope: "private"` is the correctness-critical value per ARCHITECTURE.md's Q4 reconciliation |
| Resource-not-found code `-32002` → `-32602` | "Change resource not found error code from `-32002` to `-32602`..." | **N/A** | codegraph-go implements no `resources` capability |
| **RFC 9207 `iss`** (SEP-2468) | "Authorization servers SHOULD include the `iss` parameter... MCP clients MUST validate..." | **N/A** | stdio subprocess has no OAuth surface (per REQUIREMENTS.md Out of Scope) |
| **DCR → CIMD** (PR #2858) | "Deprecate the OAuth 2.0 Dynamic Client Registration Protocol... in favor of Client ID Metadata Documents." | **N/A** | same — no auth surface |
| **SEP-837** — `application_type` in DCR | OAuth redirect URI detail | **N/A** | same |
| **SEP-2352** — credential/issuer binding | OAuth detail | **N/A** | same |
| **SEP-2106** — loosen `inputSchema`/`outputSchema`, `structuredContent` | "Loosen `inputSchema` and `outputSchema` to allow any JSON Schema 2020-12 keywords..." | **Applicable, low-impact, verify-only** | codegraph-go's 8 tool schemas are simple (`mcp.WithString`/`WithNumber`); worth a quick check whether any new keyword capability is wanted, not required |
| **SEP-2577** — deprecate Roots/Sampling/Logging | "These features remain fully functional... but new implementations should not add support for them." | **N/A / already-compliant** | codegraph-go never declared these capabilities; already uses the spec-endorsed replacements (tool-arg paths, no sampling, stderr logging) |
| **SEP-2596** — HTTP+SSE reclassified Deprecated; `includeContext` values | Governance + specific value deprecations | **N/A** | stdio-only, no HTTP+SSE transport, no elicitation `includeContext` usage |
| **SEP-1850** — SEP process formalization | Governance/process only | **N/A** | not a technical requirement |
| **Error code allocation policy** (unnumbered, minor change #12) | "`-32000`-`-32019` remains implementation-defined... `-32020`-`-32099` reserved for MCP... `UnsupportedProtocolVersion` `-32004` → `-32022`." | **Applicable when Phase 3 lands Modern `_meta` validation** | Phase 1 stays on mark3labs (Legacy, emits neither code); Phase 3 must use `-32022`, not an old `-32004`-style value |
| Deterministic `tools/list` ordering (minor change #3) | "Servers SHOULD return tools from `tools/list` in a deterministic order..." | **Applicable, believed already-compliant, verify via oracle** | `companionNames` [VERIFIED: `internal/mcp/server.go:32` — `var companionNames = []string{"node", "search", "callers", "callees", "impact", "files", "status"}`] is a fixed slice; `codegraph_explore` is always registered first — the oracle's D-05 "call `tools/list` twice, diff order" scenario should confirm this rather than assume it |
| OTel `_meta` conventions (SEP-414) | Documents `traceparent`/`tracestate`/`baggage` conventions | **N/A / optional** | No OTel adoption in codegraph-go currently |
| Elicitation `notifications/elicitation/complete` removal | Removed under MRTR | **N/A** | codegraph-go never implemented elicitation |

## Team Scale Strategic Read-Out (backlog 999.6, D-12)

Already fully worked out in `.planning/research/ARCHITECTURE.md`'s Q1 (read this session) and does not need re-derivation — the planner should transcribe its conclusion into a `.planning/` decision record per D-12:

> The stateless core (SEP-2567/2575) removes the sticky-routing/shared-session-store infrastructure a future central codegraph MCP server would otherwise have needed, because `path` is already an explicit per-call tool argument (not implicit session state) and every handler already opens a fresh `query.Engine` snapshot per call — both decisions made in v1.0 for local, single-process reasons that happen to already match the shape `2026-07-28` recommends for stateless cross-call state. The one real structural gap is `BuildServer`'s four parameters (`hasIndex`, `allowlist`, `repoPath`, `startPath`) being constructor-time-only, which would need to become per-request-resolved for a multi-tenant server — a bounded, already-anticipated refactor, not a rewrite.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| Go toolchain | Everything | ✓ | matches `go.mod`'s `go 1.26.5` | — |
| `git` | test fixtures, `gitmeta` detector | ✓ (existing dependency, unchanged) | — | — |
| Claude Code CLI (VRFY-05 roster) | 8-agent audit | ✓ [VERIFIED: `claude --version` run 2026-08-04] | `2.1.222 (Claude Code)` | — |
| Codex CLI (VRFY-05 roster) | 8-agent audit | ✓ [VERIFIED: `codex --version` run 2026-08-04] | `codex-cli 0.146.0` | — |
| opencode (VRFY-05 roster) | 8-agent audit | ✓ [VERIFIED: `opencode --version` run 2026-08-04] | `1.18.10` | — |
| Cursor (VRFY-05 roster) | 8-agent audit | ✗ — no `cursor` CLI on PATH, not found in `/Applications` [VERIFIED: `command -v cursor`, `ls /Applications`, run 2026-08-04] | — | UNMEASURED (D-10) unless installed for this audit specifically |
| Gemini CLI (VRFY-05 roster) | 8-agent audit | ✗ — no `gemini` CLI on PATH; only a "Gemini 2.app" (unrelated desktop app, not the CLI) present in `/Applications` [VERIFIED: run 2026-08-04] | — | UNMEASURED unless installed |
| Hermes Agent (VRFY-05 roster) | 8-agent audit | ✗ — not found on PATH [VERIFIED: run 2026-08-04] | — | UNMEASURED unless installed |
| Antigravity (VRFY-05 roster) | 8-agent audit | Partial — `Antigravity.app`/`Antigravity IDE.app` present in `/Applications` [VERIFIED: `ls /Applications`, run 2026-08-04], but not confirmed launchable/scriptable for this audit without further setup | — | Investigate whether Antigravity's MCP config (shared with Gemini CLI per FEATURES.md, `~/.gemini/config/mcp_config.json`) can be edited and the app relaunched non-interactively; if not, UNMEASURED |
| Kiro (VRFY-05 roster) | 8-agent audit | ✗ — not found on PATH or `/Applications` [VERIFIED: run 2026-08-04] | — | UNMEASURED unless installed |

**Missing dependencies with no fallback:** Cursor, Gemini CLI, Hermes, Kiro have no measurement path on this machine today — per D-10, these rows must be recorded as structurally-distinct `UNMEASURED` entries with an explicit blocking reason ("not installed on the audit machine"), never silently omitted or filled in from documentation.

**Missing dependencies with fallback:** Antigravity may be measurable via its shared Gemini-CLI-style config file even without a CLI entrypoint — worth a short spike before marking it UNMEASURED outright.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` (`go test`), no third-party test framework |
| Config file | `Taskfile.yml` (`test:unit`, `test:integration`, `test:golden`, `test:daemon`, `test:race` targets — read verbatim this session) |
| Quick run command | `go test ./internal/mcp/... ./test/wireoracle/...` (new package) |
| Full suite command | `task test:unit && task test:integration && task test:golden` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|--------------|
| VRFY-01 | Oracle asserts raw wire bytes, not SDK types | unit/integration | `go test ./test/wireoracle/... -run TestOracleCoversAllScenarios` | ❌ Wave 0 (new package) |
| VRFY-02 | No SDK-owned protocol-version constant anywhere in tree | archtest | `go test ./internal/mcp/archtest/... -run TestNoExternalProtocolVersionConstantReferences` | ❌ Wave 0 (new archtest) |
| VRFY-03 | stderr session line present, parseable, correct keys | unit | `go test ./internal/mcp/... -run TestSessionLineFormat` | ❌ Wave 0 |
| VRFY-04 | Oracle green against pre-migration `mark3labs` binary | integration (this IS the gate) | `go test ./test/wireoracle/...` run against today's binary | ❌ Wave 0, but structurally satisfied by VRFY-01's own suite existing and passing this phase |
| VRFY-05 | 8-agent audit, dated, MEASURED/UNMEASURED | manual + shim tooling | n/a — a one-time measurement, recorded in `docs/MCP-8-AGENT-AUDIT.md` | ❌ Wave 0 (shim tool) |
| SDK-02 | `serve.go` imports no MCP SDK package | archtest | `go test ./internal/cli/archtest/... -run TestServeGoImportsNoMCPSDK` (or reuse `import_graph_test.go`'s existing pattern with a new negative-import assertion) | ❌ Wave 0 (new archtest, or extend existing) |

### Sampling Rate
- **Per task commit:** `go test ./internal/mcp/... ./test/wireoracle/...`
- **Per wave merge:** `task test:unit && task test:integration`
- **Phase gate:** Full suite green before `/gsd-verify-work`, plus VRFY-04's explicit condition: the oracle must be run and shown green against the pre-migration binary as this phase's own acceptance gate, not deferred to Phase 2.

### Wave 0 Gaps
- [ ] `test/wireoracle/` package — the standalone capture tool itself (D-01)
- [ ] `internal/mcp/archtest/` (or a new sibling package) — VRFY-02's guard test
- [ ] An archtest asserting `internal/cli/serve.go`'s import list excludes `mark3labs/mcp-go` — either a new file or an extension of the existing `internal/cli/present/archtest` pattern
- [ ] `testdata/wireoracle/fixture/` — D-08's dedicated, purpose-built source tree (must NOT be the shared `internal/indexer/testdata/gofixture` — that fixture is mutable-by-convention across many other tests, which is exactly what D-08 exists to avoid)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|--------------------|
| V2 Authentication | No | No auth surface introduced or touched this phase |
| V3 Session Management | No | Stateless-model discussion is scoping-only this phase; no session code changes |
| V4 Access Control | No (unchanged) | `confineToRepoRoot`/CR-02 is untouched by this phase |
| V5 Input Validation | **Yes** | Sanitize `clientInfo.name`/`version` before formatting into the VRFY-03 stderr line (Pitfall 3); the wire oracle itself must treat all captured bytes as untrusted display data, never `eval`/exec them |
| V6 Cryptography | No | Not applicable |
| V7 Error Handling & Logging | **Yes** | The stderr session line is new, always-on, client-influenced logged output — the exact category log-injection defenses belong to |

### Known Threat Patterns for this phase's stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|------------------------|
| Log injection via unsanitized `clientInfo` fields in the always-on stderr line | Tampering / Information Disclosure | Strip/escape control characters (`\n`, `\r`) from all client-supplied strings before formatting (Pitfall 3) |
| A malicious/malformed binary-under-test crashing or hanging the oracle | Denial of Service (of the test run, not production) | The oracle already has precedent for bounded timeouts (`test/integration/mcp_stdout_purity_test.go`'s 30-second `deadline` pattern) — reuse it |

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|-----------------|
| A1 | Antigravity's MCP config is genuinely shareable/editable the same way Gemini CLI's is, making it measurable for VRFY-05 without a dedicated CLI | Environment Availability | If wrong, Antigravity simply becomes a 5th UNMEASURED row instead of a measured one — no downstream breakage, just a smaller measured set than hoped |
| A2 | `NeedDeps` may not be required for VRFY-02's archtest to correctly resolve external identifiers via `info.Uses` | Pitfall 5 / Pattern 2 | If wrong (i.e. `NeedDeps` turns out necessary), the guard still works, just runs slower than hoped — purely a CI-time cost, not a correctness risk |
| A3 | `internal/mcp.Server`'s interface-only shape (`ServeStdio() error`) is sufficient for SDK-02, with no additional methods needed by Phase 2's swap | Pattern 4 | Low risk — Phase 2 can widen the interface later; this is explicitly framed as evolvable in ARCHITECTURE.md |

**If this table is empty:** N/A — see above; all three assumptions are low-risk and self-correcting at the next phase boundary.

## Open Questions

1. **Where exactly should the standalone capture tool live, and does it run in-process (a Go API other tests call) or purely as a `go run`-able CLI a human redirects?**
   - What we know: D-01 requires it to be a real, general-purpose spawn-any-binary-and-script-requests tool reused across Phases 2 and 3; `test/integration/mcp_stdout_purity_test.go` is the pattern to generalize.
   - What's unclear: whether "tests invoke it and diff against frozen transcripts" (D-01) means the tool is a Go library function called from `_test.go` files (simplest, keeps everything inside `go test`), or a genuinely separate `cmd/`-style binary a human runs by hand to produce a transcript for redirection (matches "a human redirects it deliberately," D-03's anti-regeneration mechanism more literally).
   - Recommendation: a Go package exposing a `Capture(t, binPath, scenario) []byte`-shaped function for in-suite comparison, PLUS a thin `main()` (or a `-capture` test flag) for the human-redirect capture step — this satisfies both readings without duplicating the scenario-scripting logic. Left as Claude's Discretion per CONTEXT.md; the plan should pick one explicitly.

2. **Does VRFY-02's guard need to run at every PR, or only on a schedule/pre-merge gate?**
   - What we know: CONTEXT.md names "whether all ~18 [oracle] scenarios run on every PR or a narrower set gates" as Claude's Discretion, but doesn't explicitly extend that framing to the archtest.
   - What's unclear: whether a full-tree `go/packages` load (Pitfall 5) is cheap enough to run on every PR without being folded into the same discretion.
   - Recommendation: measure it once during implementation; if it adds more than ~1-2 seconds to `go test ./...`, treat it the same as the oracle's own CI-scope question.
