---
phase: 01-engine-seam-wire-protocol-secure-transport
verified: 2026-08-23T23:59:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
deferred:
  - truth: "The browser codegraph ui opens shows a meaningful app at its root URL"
    addressed_in: "Phase 2"
    evidence: "UAT gap G-01-1 (01-UAT.md) records GET / -> 404 because internal/uiserver mounts only the Connect handler prefix, no SPA. Phase 2 success criterion 1: 'A developer with only Go on PATH builds the binary, runs codegraph ui, and gets the real app in the browser from the embedded assets' — mounts the embedded SPA on this same mux, closing the 404 by construction. Phase 1's own scope boundary (01-CONTEXT.md: 'This phase ships no pixels') and ROADMAP success criterion 1 both define success as the Connect RPC surface answering real data, not the root path rendering a UI — independently reproduced live in this verification (GET / -> 404; POST /codegraph.ui.v1.UIService/GetStatus -> 200 with real counts)."
---

# Phase 1: Engine Seam, Wire Protocol & Secure Transport Verification Report

**Phase Goal:** `codegraph ui` runs as its own process, serving typed, bounded, read-only RPCs over a loopback listener that refuses a rebinding request — backed by structured Engine results and a commit-aware graph schema, with every existing CLI and MCP byte unchanged.
**Verified:** 2026-08-23
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `codegraph ui` runs standalone, prints a loopback URL, and a Connect client gets real search/callers/callees/impact/affected/files/status/node-detail/explore results including the indexed commit SHA — while `codegraph node`/`codegraph explore` stay byte-identical and every frozen golden passes (SRV-01, RPC-01, RPC-02, ENG-01, ENG-02, ENG-04) | ✓ VERIFIED | Built the binary from HEAD (`4a3c368`) and ran `codegraph ui --no-open` live against this repo's real `.codegraph/` index **while `codegraph serve --mcp` (pid 64748) already held the store**. Printed `http://127.0.0.1:59776`. `curl` POSTs to `GetStatus`, `Search`, `GetNodeDetail`, `Explore` all returned real index-derived data (5306 nodes, real symbol locations, real base64 source). `handlers_test.go` (`TestUIServiceStatusReportsCommitSha`, line ~173-191) proves `commit_sha` is populated for a fresh git fixture and empty for a non-git one — this repo's own long-lived index predates field 8 so its live `GetStatus` correctly showed no `commitSha` key (proto3 omits empty strings), which is exactly D-05's degrade-gracefully contract, not a defect. `go build ./...` and `go vet ./...` clean. Byte-identity oracle re-run live: `TestGoldensMatchLiveEngineOutput` 26/26 attempted=completed=matched, `TestReFrozenGoldensValid` 26/26, full `testdata/golden/...` suite 61 `--- PASS` / 0 `--- FAIL` (counted with `rg -c '^\s*--- PASS'`, matching 01-REVIEW-FIX.md's independently-recorded count exactly). |
| 2 | A foreign `Host`/`Origin` is rejected before any handler runs, with the three loopback spellings admitted by exact match (not interchangeably) and no v1 flag exposing a bind address or auth credential; no RPC can mutate the index (SRV-02, SRV-03) | ✓ VERIFIED | `internal/uiserver/originguard.go` wraps the whole mux (verified in `server.go:125`, outermost handler, before `http.NewServeMux()` dispatch). Live curl tests against the running server: foreign `Host: evil.example.com` → 403 `forbidden host`; valid `Host` + foreign `Origin` → 403 `forbidden origin`; **valid `Host: 127.0.0.1:<port>` paired with admitted-but-mismatched `Origin: http://localhost:<port>`** → 403 `forbidden origin` (proves the pairing rule, not just membership); `Host: localhost:<port>` + matching `Origin` → 200 real data. `internal/cli/ui.go` registers exactly two flags (`--path`/`-p`, `--no-open`) — read directly, no bind/host/port/auth flag exists. `TestUIServiceMethodSetIsExactlyTheReadSet` and `TestUIServiceDeclaresNoMutatingMethod` (`readonly_test.go`) both pass, asserting the 9-method `UIServiceHandler` set by name in both directions. |
| 3 | `codegraph ui` serves every RPC concurrently with `daemon`/`serve --mcp` holding the store and retains no handle between calls; a store locked past `graphstore.Open`'s retry budget renders as "indexing in progress," never an error; an oversized source response comes back bounded and marked truncated (SRV-04, RPC-05) | ✓ VERIFIED | Live: `codegraph ui` answered real RPCs while `serve --mcp` held the same store's lock (no contention observed — each RPC opens/snapshots/closes independently, confirmed structurally via `uiService` holding only a `repoPath string` field, asserted by `TestUIServiceHoldsNoStoreTypedField`). `internal/uiserver/degrade.go`'s `classifyDegrade`/`errIndexingInProgress`/`degradedStatus` implement D-14/D-15/D-16 exactly (CodeUnavailable + typed `IndexingInProgress` detail, ordered not-initialized-before-locked classification, `Status` answering from filesystem facts alone). `internal/uiserver/truncate.go`'s `truncateSource` implements the two-tier line-then-byte cap with `textutil.TruncateOnRuneBoundary` (shared with `internal/mcp/session_line.go` — one implementation, not two) and explicit `Truncated`/totals fields (D-11/D-12); transport backstop (`WithSendMaxBytes`/`WithReadMaxBytes`) sits strictly above both aggregate caps (`TestTransportBackstopSitsAboveEveryAggregate`). Full `internal/uiserver` package: 146 `--- PASS` / 0 `--- FAIL` under `-v`. |
| 4 | Regenerating both proto surfaces (new UI schema + pre-existing `graph.proto`) produces no diff, the guard reports a compared-file count, and has been watched fail against a deliberately stale file (BLD-04) | ✓ VERIFIED | Independently reproduced live, not just re-read from SUMMARY: `task proto:drift` → exit 0, `"proto:drift: compared 3 generated files"`, `"all 3 generated files byte-identical…"`. Then deliberately appended a stale marker line to the committed `internal/schema/graph.pb.go` and re-ran: exit 1, `"::error::proto:drift: internal/schema/graph.pb.go differs from the pinned toolchain's regeneration"` — genuine RED, independently observed in this verification session, then cleanly reverted (`git status --porcelain` empty afterward). `internal/upgrade/proto_task_test.go`'s 4 structural shape tests (`TestProtoTasksExist`, `TestProtoDriftGuardReportsAComparedCount`, `TestProtoDriftGeneratesIntoATemporaryTree`, `TestProtoDriftTaskIsInvokedByCI`) all pass, confirming the guard runs in a temp tree, reports a non-vacuous count with a failure floor, and is wired into `ci.yml`. |
| 5 | An MCP session no longer loses an in-flight response when a server-initiated notification is written concurrently, proven against a reproduction showing the loss (FIX-01) | ✓ VERIFIED | `internal/mcp/pending_writer_test.go`'s `TestPendingWriterDrainsEarlyWithoutFix` is the committed reproduction: constructs the exact premature-drain scenario (pending=1 for an accepted call, a notification write, asserts `waitForDrain` stays blocked, then the real response lands and it unblocks) — read in full, matches the ROADMAP's "reproduction that showed the loss" requirement precisely. Ran `go test ./internal/mcp/... -race -run 'PendingWriter\|...'`: all pass, including `TestPendingWriterNeverGoesNegative` and `TestPendingWriterSerializesWriteAndClassification` under `-race`. `pendingWriter.Write` (`server.go`) now classifies only actually-written bytes and decrements only response lines (verified by reading the fixed code). The folded `toolslist-repeat` flake is correctly *not* claimed as closed by FIX-01 — `TestPipelinedToolsListResponsesAreMatchedByIDWithoutNotifications` documents it as a separate, disproven-hypothesis finding, matching 01-CONTEXT.md's explicit correction. |

**Score:** 5/5 truths verified (0 present, behavior-unverified)

### Deferred Items

| # | Item | Addressed In | Evidence |
|---|------|-------------|----------|
| 1 | `codegraph ui`'s auto-opened browser root URL renders 404 (no SPA yet) | Phase 2 | `01-UAT.md` gap `G-01-1`, reproduced live in this verification (`GET /` → 404; the RPC surface itself → 200 with real data). Phase 2 success criterion 1 mounts the embedded SPA on the same mux, closing this by construction. Phase 1's own scope ("ships no pixels") and ROADMAP success criterion 1 define success in terms of the RPC surface, not the root route. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/uiproto/uiv1/ui.proto` + generated `.pb.go`/`.connect.go` | UI wire schema, additive-only | ✓ VERIFIED | Exists, builds, `TestUIProtoFieldNumbersAreStableAndUnique` passes both directions (stability + coverage) |
| `internal/uiserver/{originguard,server,handlers,degrade,truncate,diag}.go` | Origin guard, listener, RPC handlers, degrade path, truncation, diagnostic seam | ✓ VERIFIED | All present, wired, exercised live and by 146 passing tests |
| `internal/cli/ui.go` | `codegraph ui` command | ✓ VERIFIED | Registered on root (`root.go:61`), exactly 2 flags, D-07/D-08/D-09/D-10 all match code |
| `internal/query/detail.go` (`NodeDetail`, `ExploreResult`, `MultiDefDetail`) | Structured Engine seam | ✓ VERIFIED | Sum-type shapes match D-01/D-02/D-03 exactly; single gather path; byte-identity oracle proves no behavior change |
| `internal/schema/graph.proto`/`.pb.go` field 8 (`commit_sha`) | Commit-aware Meta | ✓ VERIFIED | Additive field, 3/3 `PutMeta` sites populate it, `status.go` untouched |
| `internal/textutil/truncate.go` | Shared rune-boundary cut | ✓ VERIFIED | Single implementation, consumed by both `internal/mcp` and `internal/uiserver` |
| `internal/upgrade/proto_task_test.go` + `Taskfile.yml`'s `proto:drift` | Two-surface drift guard | ✓ VERIFIED | Live-executed RED and GREEN in this verification session |
| `internal/mcp/pending_writer_test.go` | FIX-01 reproduction + fix | ✓ VERIFIED | Reproduction test read in full and re-run passing under `-race` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `originHostGuard` | whole `http.ServeMux` | outermost `http.Handler` wrap in `Listen` | ✓ WIRED | Read `server.go:111-125`; live curl confirms rejection happens before any RPC data is returned |
| `internal/cli/ui.go` | `uiserver.Listen`/`Serve` | direct call, ordered bind→print→open-browser→serve | ✓ WIRED | Live: URL printed before browser-launch attempt, server answers immediately after |
| `uiv1connect.NewUIServiceHandler` | `internal/uiserver.uiService` methods | `withEngine` → `query.OpenAt` per call | ✓ WIRED | Live RPCs returned real per-repo data; `uiService` struct has exactly one field (`repoPath`) |
| `(*Engine).IndexMeta` | `GetStatus` RPC | `schema.IndexedCommitSHA` accessor | ✓ WIRED | `statusToProto`'s doc table + `TestUIServiceStatusReportsCommitSha` |
| `task proto:drift` | committed `.pb.go`/`.connect.go` | temp-tree regeneration + `cmp -s` | ✓ WIRED | Independently re-executed live, both green and RED paths |
| `pendingWriter.Write` | `stdinLingerReader.waitForDrain` | shared `pending *atomic.Int64` | ✓ WIRED | `TestPendingWriterDrainsEarlyWithoutFix` exercises the full path end to end |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `codegraph ui` serves real GetStatus while `serve --mcp` holds the store | live binary + curl, concurrent with running daemon (pid 64748) | `{"initialized":true,"nodeCount":"5306",...}` | ✓ PASS |
| Foreign Host rejected pre-handler | curl with `Host: evil.example.com` | `403 forbidden host` | ✓ PASS |
| Cross-spelling Host/Origin pair rejected | curl `Host: 127.0.0.1:<port>` + `Origin: http://localhost:<port>` | `403 forbidden origin` | ✓ PASS |
| Matching localhost Host+Origin admitted | curl `Host/Origin: localhost:<port>` | `200` real data | ✓ PASS |
| `task proto:drift` finds no diff | `task proto:drift` | exit 0, "compared 3 generated files" | ✓ PASS |
| `task proto:drift` fails on stale generated file | staled `graph.pb.go`, re-ran | exit 1, named error, clean revert | ✓ PASS |
| Golden byte-identity oracle | `go test ./testdata/golden/... -run 'Golden\|Frozen' -v` | 61 `--- PASS` / 0 `--- FAIL` | ✓ PASS |
| FIX-01 reproduction | `go test ./internal/mcp/... -race -run PendingWriter` | all PASS | ✓ PASS |
| `internal/uiserver` full suite | `go test ./internal/uiserver/... -v` | 146 `--- PASS` / 0 `--- FAIL` | ✓ PASS |
| Full module build/vet | `go build ./...`, `go vet ./...` | clean | ✓ PASS |
| Full module test suite | `go test ./... -count=1` | 49/50 packages ok; 1 pre-existing timing flake in `internal/daemon` (untouched by this phase, passes in isolation) | ✓ PASS (with noted environmental flake) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| SRV-01 | 01-01 | `codegraph ui` own process, prints/opens URL, no shared lifecycle | ✓ SATISFIED | Live run, code read |
| SRV-02 | 01-01 | Origin/Host exact-match rebinding defense | ✓ SATISFIED | Live curl matrix |
| SRV-03 | 01-01, 01-09 | Read-only by construction, no bind/auth flag exposed | ✓ SATISFIED | Flag inspection, method-set tests |
| SRV-04 | 01-11 | Per-call store lifecycle, degrade-not-error | ✓ SATISFIED | degrade.go read, live concurrent-daemon test, 146 passing tests |
| RPC-01 | 01-01, 01-08, 01-09 | UI protobuf schema, additive-only | ✓ SATISFIED | Field-number stability test |
| RPC-02 | 01-01, 01-08, 01-09 | connect-go handlers on net/http | ✓ SATISFIED | Live RPC calls |
| RPC-05 | 01-10 | Bounded, explicitly-truncated responses | ✓ SATISFIED | truncate.go read + tests |
| ENG-01 | 01-02, 01-04, 01-05 | Structured NodeDetail, Node() unchanged | ✓ SATISFIED | detail.go read, byte-identity oracle |
| ENG-02 | 01-02, 01-05 | Structured ExploreResult, Explore() unchanged | ✓ SATISFIED | explore.go read, byte-identity oracle |
| ENG-04 | 01-06 | Commit-aware Meta field 8 | ✓ SATISFIED | 3/3 PutMeta sites, status.go untouched |
| BLD-04 | 01-07 | Two-surface proto drift guard | ✓ SATISFIED | Live RED+GREEN reproduction |
| FIX-01 | 01-03 | pendingWriter counter fix | ✓ SATISFIED | Committed reproduction re-run passing |

No orphaned requirements: REQUIREMENTS.md's Phase 1 row set (SRV-01/02/03/04, RPC-01/02/05, ENG-01/02/04, BLD-04, FIX-01 — 12 total) matches exactly the union of `requirements:` fields across all 11 plan frontmatters.

**Note (documentation staleness, not a code gap):** REQUIREMENTS.md's traceability table still shows most of these 12 rows as "Pending" (only SRV-04 shows "Complete") even though the phase is fully executed, reviewed, fixed, and UAT'd. This is a planning-artifact bookkeeping lag, not a codebase defect, and does not affect the verification verdict — GSD-owned generated files are not hand-edited by this verifier per project convention.

### Anti-Patterns Found

None. Scanned all phase-touched files (`internal/uiserver/*`, `internal/uiproto/*`, `internal/cli/ui.go`, `internal/query/{detail,errors}.go`, `internal/schema/{meta,graph.proto}`, `internal/mcp/{server,session_line}.go`, `internal/textutil/*`, `internal/upgrade/proto_task_test.go`, `internal/goldenspec/*`, `internal/graphstore/open_lock_test.go`, `internal/indexer/commit.go`) for `TBD`/`FIXME`/`XXX`, `TODO`/`HACK`/`PLACEHOLDER`, and "not yet implemented"/"coming soon" patterns — zero matches.

### Code Review Disposition

01-REVIEW.md found 3 Critical + 8 Warning findings. 01-REVIEW-FIX.md fixed all 11 (0 skipped), each with a RED-first table (test failing without fix, passing with it) and a re-verified 61/0 golden count at three checkpoints (baseline, after the one `internal/query` deletion, final). Spot-checked several fix commits directly in this verification (`CR-02`'s `singleDefSourceBlob` degrade path, `WR-03`'s four named timeout constants, `WR-05`'s `syncCommitSHA`, `WR-06`'s `unparseable` counter) — code matches the described fix exactly.

### Human Verification Required

None. Every must-have in this phase is either a structural/behavioral property provable by reading code and running tests, or a live-process behavior independently reproduced in this verification session (server start, RPC calls, origin rejection, drift-guard RED/GREEN, concurrent daemon holding the store).

### Gaps Summary

No gaps. All 5 ROADMAP success criteria hold against live, independently-reproduced evidence, not just re-stated SUMMARY claims. The one open item (UAT gap G-01-1, browser root 404) is explicitly deferred to and closed-by-construction in Phase 2, consistent with this phase's stated "ships no pixels" scope boundary. The one test-suite anomaly (`internal/daemon` flake under full-parallel load) is confirmed pre-existing, confirmed to touch no file this phase modified, and confirmed to pass cleanly in isolation.

---

_Verified: 2026-08-23_
_Verifier: Claude (gsd-verifier)_
