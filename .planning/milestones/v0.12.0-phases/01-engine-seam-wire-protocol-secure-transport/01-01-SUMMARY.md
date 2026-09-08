---
phase: 01-engine-seam-wire-protocol-secure-transport
plan: 01
subsystem: api
tags: [connect-rpc, protobuf, buf, http-security, dns-rebinding, cobra, go]

# Dependency graph
requires: []
provides:
  - "internal/uiserver package: originHostGuard (exact-match Host/Origin DNS-rebinding guard), Server/Listen/Serve/URL/Close (bind-before-publish lifecycle), uiService (per-call Connect RPC handler over internal/query.Engine)"
  - "internal/uiproto/uiv1 package: codegraph.ui.v1.UIService proto surface (GetStatus rpc) plus generated ui.pb.go / uiv1connect/ui.connect.go"
  - "buf.yaml / buf.gen.yaml / Taskfile.yml's proto:gen task: the repo's first protobuf codegen pipeline, covering both internal/schema/graph.proto and internal/uiproto/uiv1/ui.proto"
  - "go.tool-proto.mod: isolated tool modfile pinning buf v1.72.0 / protoc-gen-go v1.36.11 / protoc-gen-connect-go v1.20.0"
  - "internal/cli/ui.go: the `codegraph ui` command"
  - "connectrpc.com/connect@v1.20.0 and github.com/pkg/browser as main-module runtime dependencies, with verified API signatures for plans 01-10/01-11"
affects: ["01-02", "01-03", "01-04", "01-05", "01-06", "01-07", "01-08", "01-09", "01-10", "01-11"]

# Actuals (#2632)
actuals:
  tokens: 26426
  tasks: 3
  commits: 5

# Tech tracking
tech-stack:
  added:
    - "connectrpc.com/connect v1.20.0 (Connect RPC over plain HTTP/1.1)"
    - "github.com/bufbuild/buf v1.72.0 (build-time only, go.tool-proto.mod)"
    - "google.golang.org/protobuf/cmd/protoc-gen-go v1.36.11 (build-time only, go.tool-proto.mod)"
    - "connectrpc.com/connect/cmd/protoc-gen-connect-go v1.20.0 (build-time only, go.tool-proto.mod)"
    - "github.com/pkg/browser (cross-platform browser-open helper)"
  patterns:
    - "Exact-match Host/Origin allowlist as a plain net/http middleware wrapping the WHOLE mux, never a connect interceptor — the DNS-rebinding mitigation SRV-02 requires, with Origin membership AND per-request Host-pairing both required to close the cross-spelling gap"
    - "Bind-before-publish server lifecycle: Listen binds synchronously and returns a URL that is already connectable; Serve blocks separately over a buffered capacity-1 error channel selected against ctx.Done()"
    - "Per-call store discipline (SRV-04): openEngine/withEngine open, use, and defer-close an Engine on every rpc — no *query.Engine/GraphStore/Reader field survives on uiService, asserted both behaviorally and via reflection"
    - "Isolated tool modfile (go.tool-proto.mod) chosen over co-locating buf in go.tool.mod after MVS measurement showed co-location silently downgrades goreleaser to an unreleased nightly build"

key-files:
  created:
    - internal/uiserver/originguard.go
    - internal/uiserver/originguard_test.go
    - internal/uiserver/server.go
    - internal/uiserver/server_test.go
    - internal/uiserver/handlers.go
    - internal/uiproto/uiv1/ui.proto
    - internal/uiproto/uiv1/ui.pb.go
    - internal/uiproto/uiv1/uiv1connect/ui.connect.go
    - internal/cli/ui.go
    - internal/cli/ui_test.go
    - buf.yaml
    - buf.gen.yaml
    - go.tool-proto.mod
    - go.tool-proto.sum
  modified:
    - Taskfile.yml
    - go.mod
    - go.sum
    - internal/schema/graph.pb.go
    - internal/cli/root.go

key-decisions:
  - "Origin/Host guard: Host checked first (mandatory HTTP/1.1 header, the actual rebind signal); Origin's ABSENCE on GET admits (over-rejection is the predicted failure mode); Origin's PRESENCE requires BOTH allowlist membership AND exact pairing with \"http://\"+r.Host, closing the cross-spelling gap two independent allowlists would leave open"
  - "MVS measurement forced a third isolated go.tool-proto.mod: co-locating buf's tool directives in go.tool.mod compiled but silently downgraded goreleaser v2.17.1 to an unreleased v2.17.0-dcf5a6b3-nightly and cosign v3.1.1 to v3.0.6 — a correctness regression this repo's version-bid-loss precedent treats as disqualifying even without a hard compile failure"
  - "internal/schema/graph.pb.go's first-ever buf-generated header restamp (protoc v7.35.1 -> protoc (unknown)) is committed as its own reviewed, header-confined diff; from this commit onward a header change IS drift"
  - "mapEngineError in handlers.go is a minimal generic-to-CodeInternal mapper for this tracer only; SRV-04's CodeUnavailable+IndexingInProgress degrade path and the remaining eight error-class mappers are plan 01-11's deliverable"
  - "GetStatusRequest.path is declared on the wire but not consulted by the v1 handler — uiService always answers for its own configured RepoPath; the field is reserved for a future multi-repository milestone"

patterns-established:
  - "originHostGuard: exact map-key equality only, zero substring/prefix/suffix/case-fold matching, three independent loopback-spelling literals paired per-request with Host — the template every future loopback-serving surface in this repo should reuse"
  - "uiserver's Listen/Serve split (bind synchronously, serve separately over a buffered error channel) — the template for any future long-running network server in this codebase"

requirements-completed: [SRV-02, SRV-01, SRV-03, RPC-01, RPC-02]

coverage:
  - id: D1
    description: "Origin/Host exact-match guard rejects DNS-rebinding-shaped requests before any handler runs, admitted spellings never treated as interchangeable"
    requirement: "SRV-02"
    verification:
      - kind: unit
        ref: "internal/uiserver/originguard_test.go#TestOriginHostGuard"
        status: pass
      - kind: unit
        ref: "internal/uiserver/originguard_test.go#TestOriginHostGuardAdmitsOriginlessGET"
        status: pass
    human_judgment: false
  - id: D2
    description: "codegraph.ui.v1.UIService protobuf/Connect wire surface with one rpc (GetStatus), generated via a new buf-driven codegen pipeline covering both proto surfaces in the repo"
    requirement: "RPC-01"
    verification:
      - kind: unit
        ref: "internal/uiserver/server_test.go#TestUIServerServesGetStatusEndToEnd"
        status: pass
    human_judgment: false
  - id: D3
    description: "uiserver.Listen binds and publishes a connectable URL before uiserver.Serve blocks; Serve returns nil (not http.ErrServerClosed) on context cancellation within a 5-second shutdown budget"
    requirement: "SRV-01"
    verification:
      - kind: unit
        ref: "internal/uiserver/server_test.go#TestListenPublishesABoundPortBeforeServe"
        status: pass
      - kind: unit
        ref: "internal/uiserver/server_test.go#TestServeReturnsNilOnContextCancellation"
        status: pass
    human_judgment: false
  - id: D4
    description: "codegraph ui command: binds, prints a working URL, opens a browser only when a human is watching (D-09 suppression), serves in the foreground until Ctrl-C, exposes only --path/-p and --no-open"
    requirement: "SRV-03"
    verification:
      - kind: unit
        ref: "internal/cli/ui_test.go#TestUICommandFlagSetIsExactlyPathAndNoOpen"
        status: pass
      - kind: unit
        ref: "internal/cli/ui_test.go#TestShouldOpenBrowserSuppression"
        status: pass
      - kind: unit
        ref: "internal/cli/ui_test.go#TestUICommandPrintsConnectableURLBeforeServing"
        status: pass
      - kind: manual_procedural
        ref: "CI=1 codegraph ui --path <fixture> and the same command with stdout piped, run by hand: URL printed, no browser launched, exit 0 on SIGINT"
        status: pass
    human_judgment: false
  - id: D5
    description: "Connect handlers mount on net/http via a per-call Engine open/close discipline — no store handle survives between rpc calls"
    requirement: "RPC-02"
    verification:
      - kind: unit
        ref: "internal/uiserver/server_test.go#TestUIServerHoldsNoStoreHandleBetweenCalls"
        status: pass
      - kind: unit
        ref: "internal/uiserver/server_test.go#TestUIServiceHoldsNoStoreTypedField"
        status: pass
    human_judgment: false

duration: 10h17m (wall-clock across commit timestamps in a multi-agent session; not continuous active work time)
completed: 2026-08-23
status: complete
---

# Phase 1 Plan 1: Engine Seam, Wire Protocol & Secure Transport — Tracer Summary

**A DNS-rebinding-proof Origin/Host guard, the repo's first buf-driven protobuf/Connect-RPC codegen pipeline, and `codegraph ui` serving one real GetStatus call end-to-end over an ephemeral loopback listener.**

## Performance

- **Duration:** ~10h17m wall-clock between first and last commit (multi-agent session; not continuous work)
- **Started:** 2026-08-22T23:46:11-04:00
- **Completed:** 2026-08-23T10:03:10-04:00
- **Tasks:** 3
- **Files modified:** 20

## Accomplishments

- Origin/Host exact-match guard (`internal/uiserver/originguard.go`) mitigating the CVE-2024-28224 / CVE-2025-66414/66416 class of DNS-rebinding attacks against loopback dev servers, proven RED (14 failing subtests) against a bare-passthrough seam before the guard existed, then GREEN with 21 `--- PASS` lines under `-race`
- The repo's first protobuf codegen pipeline: `buf.yaml`/`buf.gen.yaml`, an isolated `go.tool-proto.mod` (buf v1.72.0, protoc-gen-go v1.36.11, protoc-gen-connect-go v1.20.0), and `task proto:gen` — regenerating BOTH `internal/schema/graph.proto` and the new `internal/uiproto/uiv1/ui.proto` reproducibly (verified zero-diff on a second run)
- `internal/schema/graph.pb.go`'s first-ever regeneration by a checked-in toolchain, producing exactly the anticipated one-line, header-confined restamp — absorbed and committed here per the plan's ruling that a header-only rewrite still counts as drift going forward
- A working `codegraph.ui.v1.UIService` Connect RPC service (`GetStatus`) mounted on `net/http`, answering a real client at a `Listen`-published URL from the repository's own index, with no store handle retained between calls (SRV-04)
- `codegraph ui`: binds an ephemeral loopback port, prints a URL that is already connectable, opens a browser only when a human is plausibly watching a terminal, and serves in the foreground until Ctrl-C — exposing exactly two flags and no bind-address or credential flag

## Task Commits

1. **Task 1: Origin/Host exact-match guard, demonstrated RED before the middleware exists** — `d3ae89f` (test, RED) + `0b4a5b2` (feat, GREEN)
2. **Task 2: Tracer — bind, publish, then serve one Connect RPC end-to-end to the Engine** — `fa30d9a` (feat, codegen toolchain setup) + `8c9c49c` (feat, proto surface + uiserver runtime)
3. **Task 3: `codegraph ui` — bind, print, open, then serve in the foreground** — `745736e` (feat)

_Task 1 followed the full TDD RED/GREEN cycle: the RED commit's `newGuardedHandler` seam was a bare passthrough (originHostGuard did not exist yet), recording 14 failing subtests in its own commit message; the GREEN commit added `originguard.go` and rewired the seam to call the real guard._

## Files Created/Modified

- `internal/uiserver/originguard.go` — exact-match Host/Origin guard, `allowedHosts`/`allowedOrigins`
- `internal/uiserver/originguard_test.go` — 19-row table matrix + concurrency row + origin-less-GET function (21 PASS lines)
- `internal/uiserver/server.go` — `Options`/`DefaultAddr`/`Listen`/`(*Server).Serve`/`URL`/`Close`
- `internal/uiserver/server_test.go` — the six named lifecycle/end-to-end tests
- `internal/uiserver/handlers.go` — `openEngine`, `withEngine`, `mapEngineError`, `uiService.GetStatus`
- `internal/uiproto/uiv1/ui.proto` — `codegraph.ui.v1.UIService`, one rpc, additive-only field discipline
- `internal/uiproto/uiv1/ui.pb.go`, `internal/uiproto/uiv1/uiv1connect/ui.connect.go` — generated
- `internal/cli/ui.go`, `internal/cli/ui_test.go` — `codegraph ui` command
- `internal/cli/root.go` — registers `newUiCmd()`
- `buf.yaml`, `buf.gen.yaml` — buf v2 module/gen config
- `go.tool-proto.mod`, `go.tool-proto.sum` — isolated pinned tool modfile
- `Taskfile.yml` — new `proto:gen` task, `GO_TOOL_PROTO` var
- `go.mod`, `go.sum` — `connectrpc.com/connect@v1.20.0`, `github.com/pkg/browser` added
- `internal/schema/graph.pb.go` — one-line generated-header restamp

## Decisions Made

- **Origin/Host pairing, not two independent allowlists.** When Origin is present, the guard requires BOTH set membership in `allowedOrigins(port)` AND exact equality with `"http://"+r.Host`. Two independent allowlists would admit `Host: 127.0.0.1:<port>` carrying `Origin: http://localhost:<port>` — both individually admitted spellings, but ROADMAP success criterion 2 requires the three spellings be admitted "by exact match rather than treated as interchangeable."
- **Third isolated `go.tool-proto.mod`, not co-location.** Measured live: adding buf's tool directives to `go.tool.mod` produced six binaries that all compiled, but MVS silently downgraded `goreleaser` from the intentionally pinned v2.17.1 to an unreleased v2.17.0-dcf5a6b3-nightly (and cosign v3.1.1 -> v3.0.6), losing the version bid to buf's own older `go-github` dependency. Silently swapping the release toolchain's pinned version for a nightly build is a correctness regression this repo's own version-bid-loss precedent (documented in `go.tool-lint.mod`'s header) treats as disqualifying, even though nothing failed to *compile* the way `actionlint` did. A third modfile keeps buf's large dependency tree from ever perturbing `task`/`goreleaser`'s resolved versions.
- **Header-only restamp is drift, committed once, deliberately.** `internal/schema/graph.pb.go` had never been regenerated by any checked-in invocation; `buf generate` restamps its `protoc` compiler-identity line (`v7.35.1` -> `(unknown)`, since buf never shells out to `protoc`). Verified the diff is confined to exactly that one header line, then committed it as its own reviewed change — the ruling that binds plan 01-07's later drift guard.
- **`GetStatusRequest.path` is declared but not consulted.** v1's server answers for exactly one `Options.RepoPath` set at `Listen` time; the wire field exists for a future multi-repository milestone but `uiService.GetStatus` ignores it, documented plainly in the `.proto` comment.
- **`mapEngineError` is intentionally minimal in this plan.** It wraps any error as `connect.CodeInternal` for now; SRV-04's `CodeUnavailable`+`IndexingInProgress` degrade path and the remaining eight error-class mappers are plan 01-11's deliverable, per this phase's own artifact manifest.

## Deviations from Plan

None — plan executed exactly as written, including the MVS measurement step, which was performed live rather than assumed, and produced the isolated-modfile outcome the plan anticipated as one of two possible results.

### Logged, not fixed (out of scope)

**1. [Scope boundary] Pre-existing `go mod tidy` failure, unrelated to this plan's changes**
- **Found during:** Task 2, attempting to clean up now-stale `// indirect` markers on `connectrpc.com/connect`/`github.com/pkg/browser` after they became directly imported
- **Issue:** `go mod tidy` (no `-modfile`) fails resolving `github.com/tree-sitter/tree-sitter-swift/bindings/go`, a renamed-module reference inside `internal/parser/cgo`'s dependency tree — reproduces identically without any of this plan's changes present
- **Not fixed:** out of scope for a UI-transport phase; touches a different subsystem this plan's `files_modified` does not include
- **Logged in:** `.planning/phases/01-engine-seam-wire-protocol-secure-transport/deferred-items.md`
- **Consequence:** `connectrpc.com/connect`'s `// indirect` comment in `go.mod` is now cosmetically stale (it is directly imported); `go build`/`go vet`/`go test` are unaffected — the comment carries no build-time meaning

## Issues Encountered

None beyond the MVS measurement itself, which the plan anticipated might force a third modfile and which did.

## User Setup Required

None — no external service configuration required. All new dependencies (`connectrpc.com/connect`, `github.com/pkg/browser`, `github.com/bufbuild/buf`) were pre-verified `[OK]/[Approved]` in 01-RESEARCH.md's Package Legitimacy Audit; no blocking legitimacy checkpoint was owed.

## Verified Connect API Signatures (for plans 01-10/01-11)

Recorded per Task 2's instruction, verified against `connectrpc.com/connect@v1.20.0` in the local module cache:

- `func WithReadMaxBytes(maxBytes int) Option` — `option.go:257`
- `func WithSendMaxBytes(maxBytes int) Option` — `option.go:269`
- `func NewErrorDetail(msg proto.Message) (*ErrorDetail, error)` — `error.go:61`
- `CodeUnavailable Code = 14` — `code.go:96`
- `CodeResourceExhausted Code = 8` — `code.go:70`
- Attaching a detail to an error: `(*Error).AddDetail(d *ErrorDetail)` — `error.go:225` (there is no `WithDetails` builder method; construct via `connect.NewError(...)` then call `AddDetail`)

## Generated-Header Restamp (recorded per Task 2's instruction)

```
before: // 	protoc        v7.35.1
after:  // 	protoc        (unknown)
```

Every other line of `internal/schema/graph.pb.go` is byte-identical; `git diff internal/schema/graph.pb.go` confirms exactly one changed line.

## Known Stubs

None. Every deliverable this plan claims is wired to real, working code: the guard runs live, the wire surface answers a real client, and `codegraph ui` binds/serves for real. `mapEngineError`'s single-generic-mapping shape and `GetStatusRequest.path` being unconsumed are both **documented, deliberate scope boundaries** for later plans in this phase (01-11 for the former; a future multi-repo milestone for the latter) — not stubs standing in for missing functionality this plan was supposed to deliver.

## Next Phase Readiness

- The transport spine is proven end-to-end: `codegraph ui` -> `uiserver.Listen`/`Serve` -> `originHostGuard` -> Connect handler -> `internal/query.Engine` -> typed response, on one real path.
- Plans 01-08/01-09 can add the remaining eight `UIService` rpcs on top of the same `withEngine`/`mapEngineError` seams without re-deriving the lifecycle.
- Plan 01-07's `proto:drift` guard has a clean baseline to check against: both proto surfaces regenerate byte-identical right now.
- Plan 01-11 has a clearly-scoped `mapEngineError` extension point and a reserved, wire-stable `IndexingInProgress` message to populate.
- No blockers for downstream plans in this wave.

## Self-Check: PASSED

All 16 claimed files confirmed present on disk; all 5 claimed commit
hashes confirmed present in `git log --oneline --all`. No missing
items.

---
*Phase: 01-engine-seam-wire-protocol-secure-transport*
*Completed: 2026-08-23*
