---
phase: 01-engine-seam-wire-protocol-secure-transport
fixed_at: 2026-08-23T00:00:00Z
review_path: .planning/phases/01-engine-seam-wire-protocol-secure-transport/01-REVIEW.md
iteration: 1
findings_in_scope: 11
fixed: 11
skipped: 0
status: all_fixed
---

# Phase 01: Code Review Fix Report

**Fixed at:** 2026-08-23
**Source review:** `.planning/phases/01-engine-seam-wire-protocol-secure-transport/01-REVIEW.md`
**Iteration:** 1

**Summary:**

- Findings in scope: 11 (3 Critical, 8 Warning — `fix_scope: critical+warning`)
- Fixed: 11
- Skipped: 0
- Declined in part: 2 (CR-03 and WR-02 were fixed at the wire layer as directed; the
  reviewer's *deeper* variant of each was declined — see "Declined scope" below)

## Byte-identity gate

The phase's load-bearing constraint held. Counted `--- PASS` / `--- FAIL` lines, never
exit status:

```
CGO_ENABLED=1 go test ./testdata/golden/... -run 'Golden|Frozen' -count=1 -v
```

| Run | `--- PASS` | `--- FAIL` |
|---|---|---|
| Baseline (before any fix, in the isolated worktree) | 61 | 0 |
| After WR-04 (the only `internal/query` deletion) | 61 | 0 |
| Final (all 11 fixes applied) | 61 | 0 |

No golden was regenerated, re-frozen, or edited. CLI and MCP rendered bytes are unchanged.

## Verification environment

All gates below ran **in the isolated review-fix worktree**
(`.claude/worktrees/rf-01-82561-1787513137`, branch `gsd-reviewfix/01-82561`), not in the
main checkout. This is a pure-Go module with a shared module cache, so the worktree is a
faithful build environment; the commits were fast-forwarded onto `gsd/v0.12.0-local-graph-ui`
afterwards, so the same gates are reproducible from the main checkout.

- `CGO_ENABLED=1 go build ./...` — OK
- `CGO_ENABLED=1 go vet ./...` — OK (whole module, clean)
- `CGO_ENABLED=1 go test ./... -count=1` — every package `ok` **except** one pre-existing
  timing flake, detailed below.

### Pre-existing failures (not caused by these fixes)

1. **`internal/daemon` — `TestRunWatchdogCancelsRunOnSimulatedReparent`.** Failed at 250.29s
   under full-suite parallel load; passes in 1.07s when the package is run on its own, and
   the whole `internal/daemon` package passes isolated (65.1s). `git diff --name-only
   66336e6..HEAD` touches **no** file under `internal/daemon`. This is a load-induced
   watchdog-timing flake in an untouched package, not a regression.

2. **`golangci-lint` crashes on every package that transitively imports `sentry`.** The
   locally-installed golangci-lint's staticcheck `nilness` analyzer panics with
   `internal error: unhandled builtin recover` while analyzing the third-party `sentry`
   package. Reproduced identically on `internal/daemon`, which these fixes never touch;
   `internal/textutil` (no `sentry` in its dependency closure) lints `0 issues`. This is an
   analyzer/dependency incompatibility in the tool, pre-existing and environmental.
   `go vet ./...` — which does run clean module-wide — is the gate relied on here.

## RED-first evidence

Every new guard was watched failing **before** its fix landed, and every fix was
re-reverted afterwards to confirm the new test detects its removal. Counted lines below.

| Finding | Test | Without the fix | With the fix |
|---|---|---|---|
| CR-02 | `TestUIServiceSingleDefWithNoReadableFileStillAnswers` | 0 PASS / 3 FAIL (parent + both subtests) — `invalid_argument: query: empty file path` and `internal: lstat .../pkga/pkga.go: no such file or directory` | 3 PASS / 0 FAIL |
| CR-03 | `TestUIServiceAffectedEnforcesTheFileCountCap` | 1 PASS / 2 FAIL — the over-cap request *succeeded* | 3 PASS / 0 FAIL |
| CR-01 | `TestExploreGroupsBoundTheirAggregateSourceBudget` | 0 PASS / 1 FAIL — `group[32] ... source = content:"package pkg // 32\n"`, want UNSET | 1 PASS / 0 FAIL |
| WR-01 | `TestInternalErrorOnTheWireLeaksNoHostPath` | 0 PASS / 1 FAIL — wire message was `read /var/folders/.../pkg0/pkg0.go: is a directory` | 1 PASS / 0 FAIL |
| WR-02 | `TestWithEngineRefusesAnAlreadyCancelledRequest` | 1 PASS / 3 FAIL — `fn` ran for both a cancelled and an expired context | 4 PASS / 0 FAIL |
| WR-03 | `TestListenSetsEveryServerTimeout` | n/a — the four fields were zero (`no limit`) before the fix; the test asserts each by name and value | 5 PASS / 0 FAIL |
| WR-05 | `TestSyncPreservesAKnownCommitWhenHeadCannotBeResolved` | 1 PASS / 2 FAIL — `Meta.CommitSha = ""` after an unresolvable-HEAD Sync | 3 PASS / 0 FAIL |
| WR-06 | `TestUnparseableLineDoesNotCostAFullDrainGrace` | 1 PASS / 2 FAIL — the malformed line cost `5.0016835s` | 3 PASS / 0 FAIL |
| WR-07 | `TestExploreGroupDistinguishesAMissingSourceFromAnEmptyFile` | 0 PASS / 1 FAIL — the absent path rendered as a populated empty blob | 1 PASS / 0 FAIL |
| WR-08 | `TestResolveNodeForDetailClassifiesNotFound` | 1 PASS / 2 FAIL — the error was not `errors.Is(err, ErrNotFound)` | 3 PASS / 0 FAIL |

WR-04 is a pure deletion of unreferenced code and adds no guard; its evidence is the
unchanged 61/0 golden count plus `rg 'renderSingleDefNode|renderMultiDefNode'` returning no
references anywhere in the tree.

Every guard is **positive on both sides of its boundary**, per the standing rule that a
negative-only test passes vacuously once its anchor stops matching:

- CR-03 asserts `MaxAffectedFiles` entries are still **accepted**, not only that
  `MaxAffectedFiles+1` is rejected.
- CR-01 asserts every group is still **mapped** (the file list stays complete) and that the
  first 32 carry their own blob — not merely that group 33 does not.
- WR-01 asserts the real error, absolute path and all, **reached the diagnostic stream** —
  the scrub must relocate the detail, never discard it.
- WR-02 asserts a **live** context still runs `fn`.
- WR-05 asserts a Sync that *did* resolve HEAD still **advances** the SHA — preserving must
  not freeze.
- WR-06 asserts a genuinely unanswered call **still** costs the full `stdinLingerGrace`.
- WR-07 asserts a genuinely **empty** file still yields a populated zero-valued blob.
- WR-08 asserts the message bytes are unchanged, which is what proves the conversion moved
  no CLI or golden output.

## Fixed Issues

### CR-02: `GetNodeDetail` single-def mode hard-fails for any definition with no readable on-disk file

**Files modified:** `internal/uiserver/handlers.go`, `internal/uiserver/diag.go` (new),
`internal/uiserver/sourceblob_test.go`
**Commit:** `66e97b1`

Reproduced live exactly as the orchestrator described. Extracted the read into
`singleDefSourceBlob(eng, filePath)`, which returns `nil` — an unset optional `SourceBlob` —
whenever the source is unavailable, instead of propagating the error. `filePath == ""` short
-circuits before `SourceFor` is called at all; any read error degrades identically and is
written to the server-side diagnostic stream.

Chose to degrade on **every** read failure rather than enumerating "expected" causes. The
defensible rule is the simple one: the source blob is an optional enrichment and the node's
identity, calls and called-by are correct regardless — a browser cannot act on a taxonomy of
read errors, and an operator should not learn about them from a client-visible error code.

This fix introduced `internal/uiserver/diag.go`, an `os.Stderr`-backed diagnostic seam
mirroring `internal/graphstore/logger.go`'s `diagWriter` convention exactly (unexported
package-level var, no exported setter, `codegraph: uiserver: ` provenance prefix). It is
`os.Stderr` and never `os.Stdout`, per the repo-wide `T-03-07-Leak` rule. WR-01 reuses it.

---

### CR-03: `Affected` RPC bypasses the `MaxAffectedFiles` input cap the CLI enforces

**Files modified:** `internal/uiserver/handlers.go`, `internal/uiserver/handlers_test.go`
**Commit:** `9927b66`

`query.ValidateAffectedFiles(len(files))` now runs in the handler, mapped to
`connect.CodeInvalidArgument`, **before** `withEngine` — so an over-cap request is refused
without opening the store and without the per-request reverse-adjacency / implements /
contains index builds. Placing it before `withEngine` also avoids the double-wrap trap:
`mapEngineError`'s default arm would have re-wrapped an already-`*connect.Error` as
`CodeInternal`.

The handler's doc comment, which previously claimed files "pass straight through" as though
that were safe, now states plainly why `files` needs the caller-side pre-check and `depth`
does not.

---

### CR-01: `ExploreResponse` has no aggregate `SourceBlob` bound

**Files modified:** `internal/uiserver/handlers.go`, `internal/uiserver/truncate.go`,
`internal/uiserver/truncate_test.go`
**Commit:** `d833091`

Treated as hardening, per the orchestrator's note that the magnitude was not reproducible on
this repo (`fileRelevanceGate` bounds results upstream). Applied the minimal fix and did not
restructure the truncation design.

Added `uiExploreSourceGroupCap = 32` and capped the per-group **source attachment** only —
`exploreGroupsToProto` passes `attachSource: i < uiExploreSourceGroupCap`. Deliberately did
**not** clamp `max_files` as the reviewer's primary suggestion proposed: clamping would
silently shrink a client's requested file list from 500 to 32, whereas capping attachment
keeps every group's path, symbols and skeletonized flag and drops only the inline preview.
A group past the cap carries an unset source, which — with WR-07 — is already the wire's
meaning of "no blob for this group". No proto change, no new field.

Also corrected the now-inaccurate sizing comment on `transportSendMaxBytes`, which claimed
the 16 MiB headroom had been sized against a single aggregate, and added
`TestTransportBackstopSitsAboveEveryAggregate` asserting **both** aggregates by name. The
existing `TestTransportBackstopSitsAboveApplicationCap` was kept, not replaced — the
single-blob inequality is still true and still worth pinning.

---

### WR-01: `mapEngineError`'s default arm leaks absolute host filesystem paths to the browser

**Files modified:** `internal/uiserver/handlers.go`, `internal/uiserver/degrade_test.go`
**Commit:** `3049248`

The default arm now logs the real error via `writeDiagLine` and returns a fixed
package-level `errInternal` ("an internal error occurred"). Only the **default** arm is
scrubbed: the classified arms carry this repository's own
`invalidArgumentf`/`notFoundf` strings built from the caller's own request values, and
scrubbing those would destroy legitimately useful client-facing messages.

Confirmed no existing test depends on the internal message — `handlers_test.go:1037` asserts
the `CodeInternal` *code* only.

---

### WR-02: `withEngine`'s `ctx` parameter is never used

**Files modified:** `internal/uiserver/handlers.go`, `internal/uiserver/handlers_test.go`
**Commit:** `8a5e8d1`

Added a pre-flight `ctx.Err()` gate plus `mapContextError`, which renders `context.Canceled`
as `CodeCanceled` and `context.DeadlineExceeded` as `CodeDeadlineExceeded` — routing either
through `mapEngineError`'s default arm would have shown an ordinary browser-tab close to an
operator as `CodeInternal`.

The remaining gap is **stated in the doc comment**, not left implied by an unused parameter:
mid-flight cancellation is not observed, and the practical bound on a request that outlives
its client is WR-03's `WriteTimeout` plus the per-response caps.

---

### WR-03: `http.Server` is constructed with no timeouts (gosec G112)

**Files modified:** `internal/uiserver/server.go`, `internal/uiserver/server_test.go`
**Commit:** `401e428`

All four timeouts set, as named constants
(`readHeaderTimeout` 5s / `readTimeout` 30s / `writeTimeout` 60s / `idleTimeout` 120s) per
this package's discipline of never repeating a limit as a bare literal. The test asserts
each field **by name** — one subtest each, so a future edit that drops one fails on that
field rather than passing because the other three are set — plus the two orderings
(`writeTimeout > readTimeout`, `readHeaderTimeout < readTimeout`).

---

### WR-04: `renderSingleDefNode` and `renderMultiDefNode` are dead code

**Files modified:** `internal/query/node.go`, `internal/query/detail.go`
**Commit:** `a58a111`

Both methods deleted (41 lines). `rg` confirms no references remain anywhere in the tree.
Three doc comments in `detail.go` referred to the deleted methods by name and were reworded
to describe "the pre-extraction render path" instead, so no dangling reference survives.
Golden count re-verified immediately after this commit specifically: 61 PASS / 0 FAIL.

---

### WR-05: `Sync` unconditionally overwrites `Meta.commit_sha`

**Files modified:** `internal/indexer/sync.go`, `internal/indexer/commit.go`,
`internal/indexer/commit_test.go`
**Commit:** `a181656`

Added `syncCommitSHA(resolved, prev)` in `commit.go` — next to `resolveHeadCommitSHA`, whose
"returns a string, never an error" contract is the reason the bug exists — and used it at
both `sync.go` write sites (lines 181 and 410). Deliberately scoped to `Sync`: a full index
`Run` rebuilds from scratch and has no prior value to preserve.

---

### WR-06: an unparseable inbound line permanently inflates `pending`

**Files modified:** `internal/mcp/server.go`, `internal/mcp/pending_writer_test.go`
**Commit:** `6a7ffd6`

Added an `unparseable atomic.Int64` owned by `stdinLingerReader` (a value, not a shared
pointer — unlike `pending`, nothing else mutates it) and changed `waitForDrain`'s target
from `> 0` to `> s.unparseable.Load()`.

Introduced `classifyInboundLine(line) (isCall, unparseable bool)` rather than the reviewer's
`json.Valid(line)` re-test: both bits now come from **one** `Unmarshal`, so they can never
disagree about the same line. `json.Valid` would have misclassified `{"method":1,"id":2}` as
parseable — it is valid JSON that still fails to unmarshal into `sniffedMessage`. That exact
case is a test row. `looksLikeJSONRPCCall` survives as a thin wrapper, and the test asserts
the wrapper agrees with the classifier it delegates to.

---

### WR-07: a missing `Sources` entry renders as an empty, non-truncated `SourceBlob`

**Files modified:** `internal/uiserver/handlers.go`, `internal/uiserver/truncate_test.go`
**Commit:** `fd0abe6`

The map read is now comma-ok and a miss leaves `source` unset. The test pins all three
states in one pass — present, genuinely empty, absent — which is what makes it meaningful:
the empty-file assertion is the one a naive "drop every zero-length blob" fix would fail.

---

### WR-08: `resolveNodeForDetail`'s not-found error was not converted to `notFoundf`

**Files modified:** `internal/query/node.go`, `internal/query/detail_test.go`
**Commit:** `1415e55`

One-line conversion. Because the error is currently swallowed by `buildNodeDetail`'s fast
path, the only way to assert the classification is to call the resolver directly, which the
new test does. It also asserts the exact message bytes — `classifiedError.Error()` returns
`msg` verbatim — which is what proves no golden or CLI output moved. `fmt` is still used
elsewhere in `node.go`, so the import stays.

## Declined scope

Two findings offered a deeper alternative that was **not** taken. Both were fixed at the
wire layer instead, per the standing constraint that changing shared `internal/query` code
risks CLI/MCP rendered bytes.

1. **CR-03's "better still" — move `ValidateAffectedFiles` inside `Engine.Affected`.**
   Declined. `Engine` is the single read seam the CLI, MCP server and UI share; moving an
   input rejection into it changes behavior for callers whose bytes are frozen. The CLI
   already pre-checks, so today the move would be inert for the CLI and load-bearing only
   for the UI — which the handler-level check already covers. Worth revisiting as a
   deliberate Engine-contract change in its own phase, with its own goldens re-verified.

2. **WR-02's full remedy — thread `ctx` into the Engine's `IterateNodes`/`IterateEdges`
   scan loops.** Declined as out of scope for a review-fix pass: it is a signature change
   across `internal/query`'s scan surface, not a wire-layer fix. The pre-flight gate plus
   WR-03's `WriteTimeout` bound the exposure in the meantime, and the gap is now recorded
   in `withEngine`'s doc comment rather than implied by an unused parameter.

Nothing else was skipped. All 11 in-scope findings have a landed fix.

---

_Fixed: 2026-08-23_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
