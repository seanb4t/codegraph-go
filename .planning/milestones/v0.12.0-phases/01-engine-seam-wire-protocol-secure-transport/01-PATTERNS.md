# Phase 1: Engine Seam, Wire Protocol & Secure Transport - Pattern Map

**Mapped:** 2026-08-22
**Files analyzed:** 15 (new + modified)
**Analogs found:** 12 / 15

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `internal/query/node.go` (extract gather) | service | CRUD (read) | itself, pre-extraction (`renderSingleDefNode`/`renderMultiDefNode`) | exact — refactor-in-place |
| `internal/query/explore.go` (extract gather) | service | CRUD (read) | itself, pre-extraction (render-input assembly at :568-596) | exact — refactor-in-place |
| `internal/query/node.go` (new `NodeDetail` struct) | model | transform | `internal/query/status.go`'s `StatusResult` struct | role-match |
| `internal/query/explore.go` (new `ExploreResult` struct) | model | transform | `internal/query/status.go`'s `StatusResult` struct | role-match |
| `testdata/golden/golden_test.go` (new byte-identity test) | test | batch | `TestReFrozenGoldensValid` (same file, :243-297) + `behavioral_test.go`'s live `eng.Node(...)`/`eng.Explore(...)` invocations | role-match, structural only (no byte-diff precedent exists) |
| `internal/schema/graph.proto` (Meta field 8) | model/config | CRUD (additive schema) | `has_file_index = 7` (same file, :137-151) | exact |
| `internal/schema/meta.go` (if touched for field 8 helpers) | model | transform | itself — `NewMeta()`/`IsCurrentSchemaVersion()` | exact |
| `internal/indexer/sync.go` (populate commit SHA) | service | event-driven (sync pipeline) | existing `mtime_unix_ns`/`size_bytes` population points | role-match |
| `internal/uiproto/uiv1/ui.proto` | config (schema) | request-response | `internal/schema/graph.proto` | role-match (new proto surface, same discipline) |
| `internal/uiproto/uiv1/*.pb.go`, `uiv1connect/*.connect.go` | generated | request-response | `internal/schema/graph.pb.go` | role-match (generated artifact) |
| `internal/uiserver/server.go` | service/controller | request-response | `internal/cli/serve.go` (command/lifecycle) + `internal/cli/daemon.go` (`signal.NotifyContext` foreground pattern) | role-match — no RPC-server analog exists |
| `internal/uiserver/originguard.go` | middleware | request-response | none in-repo (net/http middleware is a new shape) | no analog — see below |
| `internal/uiserver/handlers.go` | controller | request-response | `internal/mcp/tools.go` (tool dispatch → `query.Engine` calls) | role-match, structural only (MCP is stdio JSON-RPC, not HTTP/connect) |
| `internal/uiserver/truncate.go` | utility | transform | `internal/mcp/session_line.go:18-85` | exact |
| `internal/cli/ui.go` | controller (CLI command) | request-response | `internal/cli/serve.go` + `internal/cli/daemon.go` (`newDaemonStartCmd`) | exact |
| `internal/mcp/server.go` (FIX-01, `pendingWriter`) | middleware | event-driven | itself, `pendingWriter`/`stdinLingerReader` (:214-349) | exact — bugfix-in-place |
| `Taskfile.yml` (new proto task) + drift guard test | config / test | batch | `corpora:drift` task (`Taskfile.yml:3639-3677`) + `internal/upgrade/taskfile_shape_test.go` non-vacuity idiom | role-match |

## Pattern Assignments

### `internal/query/node.go` — extract `gatherSingleDefNode`/`gatherMultiDefNode`/`gatherFileNode` (ENG-01, D-01/D-02)

**Analog:** the file itself, current `renderSingleDefNode`/`renderMultiDefNode`/`Node()` (lines 316-465, read in full this session).

**Current gather+render coupling** (`internal/query/node.go:420-441`, the pattern to split):
```go
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
```

**Shared fetch primitives already extracted and reusable as-is** — `fetchCalls` (:369-390) skips a dangling calls-edge target via `errors.Is(err, graphstore.ErrNotFound)` rather than aborting (WR-04 discipline); `fetchCalledBy` (:392-408) takes a pre-built `rev map[string][]*schema.Edge` so `BuildReverseAdjacency`'s O(edges) scan is paid once and shared across single-def and multi-def paths — the new `gatherXxx` functions MUST preserve this "build once, pass down" discipline, not rebuild `rev` per call.

**Multi-def path** (`internal/query/node.go:442-465`) closes over a `fetch` closure passed into `RenderNodeMultiDef` — this closure is exactly the per-candidate gather; extracting it into a `gatherMultiDefNode` returning `[]NodeDetail`-shaped data (or the fetch closure reused as-is) satisfies D-02's multi-def coverage.

**File-only path**: `Node()` at :316-319 calls `e.readSourceFile(file)` directly and returns `renderNumberedSource(content)` — this is the third shape D-02 requires `NodeDetail` to cover; there is no existing gather/render split here since there's nothing to gather beyond the read — `NodeDetail`'s `Source []byte` field is populated directly from `readSourceFile`'s return.

**Repo-root confinement (reuse, do not reimplement)** — `resolveSourcePath`/`readSourceFile` (`internal/query/node.go:18-88`, read in full): two-stage confinement (string-level `filepath.Clean`/`Rel` check, THEN `filepath.EvalSymlinks` re-verification against resolved paths) — this is the ONE safety gate for any file-path-driven read; the new UI RPC handlers reading verbatim source MUST call through `Engine.readSourceFile`, never a new `os.ReadFile`.

---

### `internal/query/explore.go` — extract gather (ENG-02, D-01/D-02, Pattern 2 in RESEARCH)

**Analog:** the file itself; the render-input assembly at `internal/query/explore.go:568-596` (already read/quoted in RESEARCH verbatim) is the exact stop-before-`RenderExplore` boundary — `groups, symbolCount := groupMatchesByFile(...)` through `return RenderExplore(query, len(groups), symbolCount, groups, blasts, sources, stale, skeletonFiles), nil`.

**Non-trivial part to carry into the plan:** `exploreZeroResult(query, stale)` and several other early `return ..., nil` branches (RESEARCH lines ~293, ~301, ~356, ~404, ~466, ~474) each return an already-rendered STRING for the zero-match case — `ExploreResult` needs an explicit empty/no-match mode field so these branches funnel into the same structured type rather than being skipped.

**Shared BuildReverseAdjacency reuse** — same discipline as node.go: built once, shared across Explore's blast-radius computation and the multi-def node path; the gather extraction must not duplicate this scan.

---

### `internal/schema/graph.proto` — `Meta` field 8 (ENG-04, D-05)

**Analog:** `has_file_index = 7`, the immediately preceding field in the same message (`internal/schema/graph.proto:137-151`, read in full).

```protobuf
message Meta {
  uint32 schema_version = 1;
  int64 node_count = 2;
  int64 edge_count = 3;
  int64 last_sync_unix_ms = 4;
  bool healthy = 5;
  string health_message = 6;

  // has_file_index reports whether this graph's on-disk store has been
  // populated with the Phase-4 `x/` file-owned secondary index (D-02).
  // Additive Phase-4 field (D-02b); false means a pre-Phase-4 graph that
  // lacks the index, signaling `sync` to perform a one-time full
  // re-index backfill before switching to incremental updates.
  bool has_file_index = 7;
}
```

Field 8 (`commit_sha` or similar) must follow the identical doc-comment convention: name the field, cite this phase/decision (D-05), and state the degrade behavior for absence ("absent means a pre-upgrade graph"). This is a **one-way, additive-only** change per `internal/schema/meta.go`'s package-comment discipline (read in full):

```go
// Additive-only discipline (D-02a): within a single SchemaVersion, fields
// on Node, Edge, File, and Meta are NEVER renumbered or reused. Retiring a
// field means adding its number to a `reserved` clause in graph.proto, not
// deleting or repurposing it.
```

`NewMeta()` (`internal/schema/meta.go:14-17`) is the single construction point — do not construct `Meta{}` literals elsewhere; if a helper is needed for the new field, add it here following `IsCurrentSchemaVersion`'s shape (nil-safe getter check).

---

### `internal/mcp/server.go` — FIX-01 `pendingWriter` fix

**Analog:** itself; the bug and its exact location were read in full this session (`internal/mcp/server.go:200-349`).

**The bug** — `pendingWriter.Write` decrements on EVERY write:
```go
func (p *pendingWriter) Write(b []byte) (int, error) {
	n, err := p.w.Write(b)
	p.pending.Add(-1)
	return n, err
}
```
while only client-initiated call lines increment, inside `stdinLingerReader.Read` (:276):
```go
if looksLikeJSONRPCCall(line) {
	s.pending.Add(1)
}
```
A server-initiated notification write (never counted on the increment side) still decrements — driving `pending` negative and shortening `waitForDrain`'s effective wait.

**Fix mechanics are Claude's discretion** (CONTEXT D-discretion) — either `pendingWriter` learns to distinguish response writes from notification writes (parse the outgoing line for a response shape, mirroring `looksLikeJSONRPCCall`'s sniff-before-classify pattern), or move the decrement so only response writes count. Whichever is chosen, follow the file's existing doc-comment discipline: explain the invariant being protected, reference the specific race it fixes, and keep the change **confined to this method** — "no change to the Server interface, to BuildServer, to internal/cli/serve.go" (per the file's own `ServeStdio` doc comment, already true of the surrounding code and must remain true of the fix).

**Test analog:** `internal/mcp/session_line_test.go` (read :60-90, :230-260) — table-driven `t.Run` subtests, an explicit "prove it fails" assertion style (`t.Fatalf` with both got/want and WHY it matters in the message). A new `TestPendingWriter` should mirror this: construct a scenario with a server-initiated notification interleaved with client calls, assert `pending` never goes negative and `waitForDrain` still blocks correctly.

---

### `internal/mcp/session_line.go` — truncation pattern (RPC-05, D-11) — the load-bearing shared pattern

**Analog:** `internal/mcp/session_line.go:1-90` (read in full) + `internal/mcp/session_line_test.go:60-90,230-260` (read in full).

```go
// clientFieldMaxBytes is the single documented boundary sanitizeClientField
// truncates every client-supplied field to, on a UTF-8 rune boundary. Named
// so the limit is one number referenced from both the truncation call and
// this doc comment, rather than a magic 256 repeated at each call site.
const clientFieldMaxBytes = 256

func truncateOnRuneBoundary(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}
```

**Test discipline to mirror** (`session_line_test.go:72-80`):
```go
func TestSanitizeClientFieldTruncatesOnRuneBoundary(t *testing.T) {
	long := strings.Repeat("中", 100) // 300 bytes
	got := sanitizeClientField(long)
	if len(got) > clientFieldMaxBytes {
		t.Fatalf("sanitizeClientField result is %d bytes, want <= %d", len(got), clientFieldMaxBytes)
	}
	if !utf8.ValidString(got) {
		t.Fatalf("sanitizeClientField truncated mid-rune, producing invalid UTF-8: %q", got)
	}
}
```

`internal/uiserver/truncate.go` must reproduce this exact discipline for source-response bounding: named constants (`sourceLineCap`, `sourceByteCap` — exact values are this phase's discretion per RESEARCH Open Question 3), rune-boundary-safe byte cutting (`truncateOnRuneBoundary` can likely be reused verbatim or duplicated with attribution — it's an unexported function in package `mcp`, so `uiserver` needs its own copy or a shared `internal/textutil` extraction; either is defensible, note it as a planning decision), and a companion test asserting no split runes, matching `TestSanitizeClientFieldTruncatesOnRuneBoundary`'s exact shape.

---

### `internal/cli/ui.go` — new `codegraph ui` command (SRV-01/SRV-03)

**Analog:** `internal/cli/serve.go` (read :1-60, :230-310 in full) + `internal/cli/daemon.go`'s `newDaemonStartCmd` (read :100-160 in full).

**Foreground signal-handling pattern to copy verbatim** (`internal/cli/daemon.go:135-140`):
```go
ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

err = d.Run(ctx)
```
This is the exact shape `codegraph ui` must follow for D-10's "foreground until Ctrl-C" — mirrors both `serve` and `daemon start`.

**`--path/-p` flag convention** (`internal/cli/serve.go` end-of-file pattern, confirmed at daemon.go's `resolveStartPath(path)` call and serve.go's `cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")`) — `ui.go` must register the identical flag.

**`errors.Is(err, graphstore.ErrStoreLocked)` handling** (`internal/cli/serve.go`, the startup-reconcile branch, read in full):
```go
if _, err := indexer.Sync(repoPath, storeDir, indexer.Options{Quiet: true}); err != nil {
	if !errors.Is(err, graphstore.ErrStoreLocked) {
		return err
	}
	fmt.Fprintf(cmd.ErrOrStderr(),
		"codegraph serve: startup reconcile skipped — the graph store is locked by another codegraph process...: %v\n", err)
}
```
`codegraph ui` likely does not need this exact reconcile-skip branch (it never syncs), but the classification idiom (`errors.Is` against the exported sentinel, never a broader string/type check) is the one to reuse when SRV-04's degrade path needs to detect the same condition inside the RPC layer.

**Cobra registration point:** `internal/cli/root.go:44-50` — `newUiCmd()` (or similarly named) must be added to the `root.AddCommand(...)` list alongside `newServeCmd(), newSyncCmd(), newDaemonCmd(), ...`.

---

### `internal/uiserver/server.go` — bind/serve lifecycle (SRV-01, D-07/D-08/D-09/D-10) and `internal/uiserver/originguard.go` (SRV-02)

**No RPC-server analog exists in this repo.** `internal/mcp/` is stdio JSON-RPC, not HTTP — its handler-dispatch shape (`internal/mcp/tools.go`'s tool-registration → `query.Engine` calls) is a legitimate STRUCTURAL analog for "how a protocol handler calls into `internal/query`" but NOT for HTTP listener lifecycle, middleware, or Origin/Host validation, none of which exist anywhere in this codebase today.

**Foreground/signal pattern:** reuse `internal/cli/daemon.go:135-140`'s `signal.NotifyContext` shown above — this is the one lifecycle precedent that transfers directly (RESEARCH's own Code Examples section sketches this composition, uncredited to a specific file since it doesn't exist yet; the signal-handling half is directly lifted from daemon.go).

**Origin/Host middleware — genuinely no analog.** RESEARCH's own sketch (illustrative, explicitly marked "not verbatim from any existing file") is the closest starting point:
```go
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
			ok := origin == "http://127.0.0.1:"+port || origin == "http://localhost:"+port || origin == "http://[::1]:"+port
			if !ok {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
```
Must run as a plain `net/http` middleware wrapping the WHOLE mux (not a connect interceptor — interceptors run after routing/parsing has begun).

---

### `internal/uiserver/handlers.go` — RPC method implementations (RPC-02, SRV-04)

**Structural analog only:** `internal/mcp/tools.go` — the pattern of "resolve repo path → `query.OpenAt` → call an `Engine` method → map errors" is the same shape, but the transport is entirely different (stdio JSON-RPC vs connect-go/HTTP). Read `internal/query/status.go:230-260`'s `Status(ctx)` signature and its per-key remapping table (the file's own extensive doc comment, read in full) as the template for how `NodeDetail`/`ExploreResult`/`StatusResult` map onto wire messages — same "table of what each field becomes and why" documentation discipline should carry over to the uiv1 mapping code.

**D-16 degrade path — genuinely new plumbing, not reused.** `Engine.Status()` only runs after a successful `Engine` construction; `query.OpenAt` fails outright on a past-budget `ErrStoreLocked`. The handler must catch `errors.Is(err, graphstore.ErrStoreLocked)` from `OpenAt` BEFORE constructing an `Engine`, and build the degraded response from `query.ResolveCodegraphDir` (an `os.Stat`-only check, independent of the store lock) — this exact reasoning is spelled out in RESEARCH Pitfall 3 and must be treated as authoritative; there is no fallback code to copy since none exists.

**D-14 typed error detail** (Connect's `NewErrorDetail`, confirmed via Context7 official docs, no in-repo analog since this is the first Connect usage):
```go
detail, err := connect.NewErrorDetail(&uiv1.IndexingInProgress{
	Message: "the graph store is locked by another codegraph process (an in-flight sync); try again shortly",
})
if err != nil {
	return nil, connect.NewError(connect.CodeInternal, err)
}
return nil, connect.NewError(connect.CodeUnavailable, errors.New("store locked")).WithDetails(detail)
```
Per RESEARCH's Security Domain table: the detail message must stay GENERIC — never embed raw OS lock-file paths or PIDs (Information Disclosure risk to an unauthenticated loopback caller).

---

### `Taskfile.yml` + drift guard test (BLD-04)

**Analog:** `corpora:drift` task (`Taskfile.yml:3639-3677`, cited in RESEARCH but not re-read this session — RESEARCH already quotes its exact shape: "regenerate, then `git diff --name-only`, fail if non-empty") — but it does **not** report a positive count, a gap BLD-04 must close (rule `84d1gfpywd`).

**Non-vacuity test analog:** `internal/upgrade/taskfile_shape_test.go` (read :1-60 in full) — the established idiom for asserting Taskfile/workflow structure from Go tests:
```go
const (
	rootGoModPath    = "../../go.mod"
	toolModfilePath  = "../../go.tool.mod"
	lintModfilePath  = "../../go.tool-lint.mod"
	taskfilePath     = "../../Taskfile.yml"
	goreleaserPath   = "../../.goreleaser.yaml"
	checkCrossTaskID = "check:cross"
)
```
followed by fixtures like `requiredCheckNames`/`forbiddenToolPackages` (literal slices, each with a provenance comment citing how/when they were verified) and `reflect`/`regexp`/`yaml.v3`-based parsing of the actual on-disk files. A new `proto_task_test.go` (or an addition to this file) should follow the identical convention: a literal fixture of the expected task name/structure, parsed from the real `Taskfile.yml`, with a companion test proving it fails against a deliberately stale/missing task (per CONTEXT's explicit constraint: "must be... watched fail against a deliberately stale checked-in file").

---

### `testdata/golden/golden_test.go` — new byte-identity oracle (D-04, ENG-01/ENG-02's actual safety net)

**No byte-identity analog exists — this is the phase's most important "No Analog" finding, already surfaced by RESEARCH's Pitfall 1 and CONTEXT's ⚠ CORRECTED block.** `TestReFrozenGoldensValid` (`testdata/golden/golden_test.go:243-297`) checks only existence/non-emptiness/JSON-shape. `TestCorpusBehavior_Go` (`testdata/golden/behavioral_test.go:685-691` per RESEARCH quote) asserts named properties, not byte-diffs.

**Closest structural analogs to build from:**
1. `TestReFrozenGoldensValid`'s enumeration loop over `expectedGoCaptures` (`testdata/golden/golden_test.go:169-224` per RESEARCH) — the (corpus, filename) pair iteration shape to reuse for a NEW test that, per pair, invokes the live engine and compares to the golden's `Output` field.
2. `testdata/golden/behavioral_test.go`'s live `eng.Node(...)`/`eng.Explore(...)` invocations (RESEARCH cites :874, :909, :931, :961, :1043, :1051, :1282, :1326, :1378) and `buildEngineAt` (:225) — the pattern for constructing a live `Engine` over a corpus and calling it, to be reused for the byte-diff comparison rather than just property assertions.
3. `testdata/golden/gocapture/main.go`'s `lockedCorpusArgs` table (:63-99 per RESEARCH) — the exact symbol/query parameters each golden was captured with; the new test MUST call `Node`/`Explore` with these same locked arguments, not re-derive its own.

**Scoring discipline (D-04):** count `--- PASS` lines from `go test -v -run <name>`, never rely on exit status — CONTEXT explicitly warns `go test -run PATTERN` exits 0 when the pattern matches nothing ("three phantom commands were found this way in v0.11.0").

## Shared Patterns

### Rune-safe truncation (applies to `internal/uiserver/truncate.go`)
**Source:** `internal/mcp/session_line.go:78-85` (`truncateOnRuneBoundary`)
**Apply to:** RPC-05's line/byte capping for verbatim source responses.
```go
func truncateOnRuneBoundary(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	for maxBytes > 0 && !utf8.RuneStart(s[maxBytes]) {
		maxBytes--
	}
	return s[:maxBytes]
}
```

### `errors.Is` sentinel classification, never string/type matching (applies to `internal/uiserver/handlers.go`, `internal/cli/ui.go` if it needs the same check)
**Source:** `internal/cli/serve.go`'s reconcile branch and `internal/graphstore/pebble_store.go`'s `Open` (per RESEARCH, `openLockRetryAttempts`/`ErrStoreLocked` classified exactly once inside the retry loop).
**Apply to:** SRV-04's degrade-to-`CodeUnavailable` mapping — never re-classify a broader error class as lock contention; check `errors.Is(err, graphstore.ErrStoreLocked)` specifically, as serve.go's own comment explains ("a permission error anywhere in Sync's chain can never masquerade as lock contention here").

### Foreground long-running command with `signal.NotifyContext` (applies to `internal/cli/ui.go`, `internal/uiserver/server.go`)
**Source:** `internal/cli/daemon.go:135-140`
```go
ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
defer stop()
err = d.Run(ctx)
```
**Apply to:** D-10's "foreground until Ctrl-C" for `codegraph ui`, adapted with an added `srv.Shutdown(shutdownCtx)` step (RESEARCH's Code Examples section already sketches the HTTP-server variant of this).

### Additive-only proto field numbering, one field per decision, with an explanatory doc comment
**Source:** `internal/schema/graph.proto:137-151` (`has_file_index = 7`), `internal/schema/meta.go`'s package comment.
**Apply to:** ENG-04's `Meta` field 8, and every field in the new `internal/uiproto/uiv1/ui.proto` surface — RPC-01 inherits this discipline explicitly per RESEARCH's Architecture Patterns section.

### Non-vacuity guard: literal fixture + parse-and-assert + "prove it fails" companion test
**Source:** `internal/upgrade/taskfile_shape_test.go` (19 of the repo's 49 such guards live here).
**Apply to:** BLD-04's drift guard and D-04's mutation-proof — both inherit rule `84d1gfpywd` ("guards carry positive assertions"), and CONTEXT explicitly requires both to be demonstrated failing against a deliberately broken/stale input before being trusted passing.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `internal/uiserver/originguard.go` | middleware | request-response | No `net/http` server of any kind exists in this repo today; RESEARCH's sketch is illustrative only, explicitly marked "not verbatim from any existing file" |
| `internal/uiserver/server.go` (connect-go mounting, `net.Listen`, `http.Server`) | service | request-response | First HTTP/connect-go surface in the repo; `internal/mcp/` is stdio JSON-RPC and only transfers at the "handler calls into `internal/query`" level, not listener/transport lifecycle |
| `internal/uiproto/uiv1/ui.proto` + generated files | config/generated | request-response | Second proto surface but a genuinely new package tree (`internal/uiproto/`); `internal/schema/graph.proto` is the discipline analog (additive-only, field numbering) but not a structural one (different message/service shape — RPC service definitions vs. graph node/edge/meta storage records) |
| `testdata/golden/golden_test.go`'s new byte-identity comparison test | test | batch | Confirmed by CONTEXT's ⚠ CORRECTED block and RESEARCH Pitfall 1: no test in this repo currently byte-diffs live engine output against a frozen golden — the TS-era capture path was retired in FIXT-04. Closest structural precedents (enumeration loop, live engine invocation, locked corpus args) are listed above as things to compose from, not a single analog to copy. |

## Metadata

**Analog search scope:** `internal/query/`, `internal/schema/`, `internal/cli/`, `internal/mcp/`, `internal/upgrade/`, `testdata/golden/`, `Taskfile.yml`
**Files read in full or targeted-range this session:** `internal/cli/serve.go`, `internal/cli/daemon.go`, `internal/cli/root.go`, `internal/mcp/session_line.go`, `internal/mcp/session_line_test.go`, `internal/mcp/server.go` (:200-350), `internal/query/node.go` (:1-90, :300-470), `internal/query/status.go` (:1-60, :230-260), `internal/schema/graph.proto` (:120-160), `internal/schema/meta.go`, `internal/upgrade/taskfile_shape_test.go` (:1-60)
**Pattern extraction date:** 2026-08-22
