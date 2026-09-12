# Architecture Integration Research: v0.13.0 Guard Hardening & UI Follow-through

**Domain:** integration architecture for a new-features milestone on an established codebase
**Researched:** 2026-09-08
**Confidence:** HIGH — every claim below is either a quoted source line from this repository at HEAD, or an explicit inference labelled as such.

## How to read this file

This is not a green-field architecture doc. `internal/query.Engine` is the single read-only
seam; `internal/uiserver` is a ConnectRPC surface over it guarded by two tests
(`internal/uiserver/readonly_test.go`) that this milestone must clear for every new symbol;
`internal/schema/graph.proto` (on-disk) and `internal/uiproto/uiv1/ui.proto` (wire) are two
independently-evolving, additive-only proto surfaces (D-02a). Each numbered section below
answers one milestone question with file paths, quoted source, and an explicit
new-vs-modified verdict.

---

## 1. HLT-04 — coverage denominator ("files discovered but NOT indexed, with reason")

### What `internal/indexer.Discover` actually knows and drops

`internal/indexer/discover.go:97` (`func Discover(root string) ([]DiscoveredFile, string, error)`)
walks the tree with `filepath.WalkDir` and for every candidate file either appends a
`pendingFile` or returns `nil` from the callback with **no record kept**. Three exclusion
reasons exist in the walk today, none of them persisted or even counted:

1. **Directory skip** — `discover.go:106`: `if p != root && ShouldSkipDir(d.Name()) { return fs.SkipDir }`. `ShouldSkipDir` (`discover.go:53`) excludes `vendor` and any dot-prefixed directory.
2. **Unregistered extension** — `discover.go:113-116`: `spec, ok := lookupLanguageByExt(ext); if !ok { return nil }`.
3. **Go build-tag mismatch** — `discover.go:128-134`: `ctx.MatchFile(...)`, Go-only, via `go/build.Context`.

None of these three produce any record — the file's relative path is never captured before
the early `return nil`. This is genuinely new discovery-time data the store does not persist
today, exactly as the milestone context predicted.

### A second, already-persisted category the milestone context did not anticipate

Files that **do** pass `Discover` and reach Pass 1 (`internal/indexer/extract.go`) but fail to
parse or read are **already recorded**, end to end:

- `extract.go:122-129`: a read failure is captured on `goextract.FileResult.Err`.
- `internal/indexer/resolve.go:296-298`:
  ```go
  if r.Err != nil {
      f.Errors = []string{r.Err.Error()}
  }
  ```
  — a `schema.File` record is still minted for the failed file, with `Errors` populated.
- `internal/schema/graph.proto:113-117`:
  ```proto
  // errors records per-file extraction failures (e.g. oversized or
  // unparseable source) so one bad file does not abort the whole index run
  repeated string errors = 6;
  ```

So `schema.File.errors` **already exists on disk, schema version 1, no migration needed** —
but nothing reads it out. `rg -n "Errors|errors" internal/query/status.go internal/query/files.go`
returns zero hits outside `"errors"` the stdlib import and two `errors.Is` calls — confirmed:
no Engine method surfaces `File.Errors` today.

**HLT-04 is therefore two distinct sub-problems, not one:**

| Category | Source of truth | Persisted today? | What's needed |
|---|---|---|---|
| Discovered-but-excluded-before-extraction (vendor/dot-dir, unsupported ext, Go build-tag) | Not captured anywhere | No | New discovery-time collection, computed fresh at query time (see below) |
| Discovered, extracted, but failed (parse/read error) | `schema.File.Errors` | **Yes**, already on disk | Just a new Engine read + wire projection |

### Additive-only feasibility (D-02a)

No schema change is required for either category if computed at query time rather than
persisted:
- Category 2 already has a field (`File.errors`, number 6, spent since Phase 2/3) — reading
  it is pure Engine work, zero proto changes to `internal/schema/graph.proto`.
- Category 1 does **not** need a schema field at all if implemented as a live filesystem
  walk at query time (see next section) — the "reserved 50 to 59" annotation slots on
  `Node`/`Edge` (`graph.proto:65`, `:96`) are explicitly earmarked for a **different**
  concern ("embedding vector, community/cluster assignment" — see §2) and should not be
  overloaded for this.

### Recommended integration: query-time, not index-time — following `Status()`'s own precedent

`internal/query.Engine` already does **live filesystem work alongside store reads**, not just
pure store scans — this is the load-bearing precedent that makes a new Coverage-style method
architecturally consistent rather than a special case:

- `Engine` carries `repoRoot string` (`internal/query/engine.go:42`).
- `Status()` conditionally walks the filesystem when `e.repoRoot != ""`
  (`internal/query/status.go:317-320`, `newestSourceMtime` at `status.go:113`), and degrades
  gracefully to a zero value when no repo root is configured (New vs OpenAt) — exactly the
  degrade discipline a new method should copy.

**Recommended new Engine surface** (new, per milestone context's own framing): an
`Engine.Coverage()`-shaped method (naming below is illustrative, not binding) that:
1. Re-walks `e.repoRoot` reusing `indexer.ShouldSkipDir` and the language-extension registry
   (or a small sibling of `indexer.Discover` that records exclusions with a reason string
   instead of silently continuing) to enumerate category 1 — computed **fresh per call**, no
   cache, mirroring `FileGraph()`'s and `BuildReverseAdjacency`'s fresh-per-call discipline
   (`traverse.go:150-152`: *"It is built FRESH inside every call, with no package-level
   cache and no once-latch"*).
2. Scans `IterateFiles()` for records with non-empty `Errors` for category 2 — a cheap
   addition to the same scan `Status()` already runs (`status.go:260-269`).
3. Degrades to an empty/zero result when `e.repoRoot == ""`, never erroring — same contract
   `Status()`'s `DbSizeBytes` follows (`status.go:317`, D-07 precedent).

This needs a genuinely **new package-level function in `internal/indexer`** (or a query-local
duplicate, mirroring the `shouldSkipStaleDir` precedent in `status.go:107-110`, which
duplicates `indexer.ShouldSkipDir` specifically "to avoid an internal/query -> internal/indexer
dependency edge" — the same choice must be made here, and the T-01-18 archtest in §6 will
directly govern it).

### Wire integration — extend `GetHealth`, do not add a 15th rpc

`GetHealthResponse` (`internal/uiproto/uiv1/ui.proto:742-812`) is the health-page-only,
heavier-than-`GetStatus` surface — its own doc comment says it exists "separately from
GetStatus (which stays cheap for every-navigation polling, D-01)" (readonly_test.go's
plan-04-03 comment). Coverage is inherently a health/diagnostic question ("why is my file
missing"), and the last spent field number on `GetHealthResponse` is 16
(`commit_sha`, `ui.proto:809`) — **field 17 is free and additive**. The frontend precedent
(`web/src/routes/health/+page.svelte`) already exists as the natural consumer; no existing
"coverage" concept is in the frontend today (`rg -n "coverage|Coverage" web/src` finds
nothing outside unrelated `highlight.ts`/`SourcePane.svelte` matches), confirming this is new
UI, not rewiring.

**Verdict: no new rpc is required.** Extend `GetHealthResponse` with a new nested message
(mirroring the `IndexHealth`/`PendingChanges`/`WorktreeMismatch` per-concept message
pattern already established at fields 12-14) at field 17, mapped by a new
`coverageToProto`-named function in `internal/uiserver/handlers.go` alongside
`healthToProto` (`handlers.go:900`), `fileGraphToProto` (`:997`), `fileSymbolsToProto`
(`:1082`) — **never an inline literal at the call site**, matching every existing mapper's
documented discipline.

**Naming gotcha if a new rpc is chosen instead** (e.g. if planning decides Coverage warrants
its own call for pagination/cost reasons): `mutatingVerbs`
(`internal/uiserver/readonly_test.go`) forbids the *substring* `"Index"` anywhere in a method
name — `TestUIServiceDeclaresNoMutatingMethod` matches with `strings.Contains(name, verb)`.
A tempting name like `GetIndexCoverage` or `IndexCoverage` **fails this guard outright**,
despite being purely read-only. Safe names: `GetCoverage`, `Coverage`. This is a real,
source-verified trap, not a style note.

### New vs modified — HLT-04

| Component | New / Modified | File |
|---|---|---|
| Discovery-exclusion collector | **New** function (likely `internal/indexer`) | `internal/indexer/discover.go` or a new sibling file |
| `Engine.Coverage()` (or extension of `Status()`) | **New** Engine method | `internal/query/status.go` or a new `internal/query/coverage.go` |
| `GetHealthResponse` field 17 + nested message | **Modified** (additive) | `internal/uiproto/uiv1/ui.proto` |
| `coverageToProto` mapper | **New** function | `internal/uiserver/handlers.go` |
| `GetHealth` handler | **Modified** (calls the new Engine method) | `internal/uiserver/health.go` (or wherever `GetHealth` is implemented — verify exact file at plan time; `health_test.go` exists in `internal/uiserver/`) |
| Health page UI | **New** rendering | `web/src/routes/health/+page.svelte` |
| `uiProtoFieldNumbers` fixture | **Modified** (extended, never rewritten) | `internal/uiserver/readonly_test.go` |

---

## 2. GRF-06 — community clustering

### Where the schema "reserves annotation space" — and why that is the wrong slot for this feature

`internal/schema/graph.proto` reserves field ranges 50-59 on **`Node`** (`:65`), **`Edge`**
(`:96`), and a third message (`:166`, reserved for "team-scale provenance") — this is the
**on-disk, persisted** schema, explicitly earmarked:

```proto
// Reserved now, before any consumer exists, so a future embedding-vector
// or community/cluster-assignment field lands at a pre-agreed number
// instead of colliding with whatever the next organic field would be.
reserved 50 to 59; // future: embedding vector, community/cluster assignment
```

This IS the literal slot the milestone context refers to. But using it would mean **persisting
community assignments at index time** into `Node` records on every re-index — a structural
choice with real consequences (index-time compute cost, staleness between re-indexes,
migration of the reserved-field convention from "future" to "spent").

### The established counter-precedent: `FileGraphNode.CycleID` is computed fresh, at query time

`internal/query/traverse.go`'s `FileGraph()` already solves the **structurally identical**
problem — partitioning the file-adjacency graph into components — for cycle detection, and it
does so **without touching the persisted schema at all**:

- `traverse.go:87-89`: `CycleID` is "0 for a node in no strongly-connected cycle and is
  populated by the cycle detector, not by this scan."
- `traverse.go:305-314`: cycle detection runs **inside** `FileGraph()`, over the same
  aggregated `cycleAdj` adjacency map the edge-rollup scan just built, via
  `stronglyConnectedCycles(cycleAdj)` — a pure in-memory graph algorithm, no store write.
- `traverse.go:150-152` (doc comment on `FileGraph` itself): *"It is built FRESH inside every
  call, with no package-level cache and no once-latch — mirroring `BuildReverseAdjacency`'s
  fresh-per-call discipline."*

`FileGraphNode` (`ui.proto:824-839`) and `FileGraphEdge` (`ui.proto:842-871`) carry **no
`reserved` clause at all** — unlike `Node`, `SourceBlob`, and the graph.proto messages, this
message was never given a pre-agreed annotation band, because nothing anticipated needing one:
the pattern established for extending it is a plain **additive new field**, exactly like
`CycleID`/`cycle_id` (field 4) and `InCycle`/`in_cycle` (field 5) were added.

### Recommendation: community detection is computed at QUERY TIME inside `FileGraph()`, following the `CycleID` pattern exactly — not persisted, and not using the schema's reserved 50-59 slot

This is a direct, source-backed answer to the milestone's open question. The reserved
50-59 range on `schema.Node`/`Edge` remains earmarked for a genuinely different, later
concern (embeddings, team-scale provenance) and should be left untouched by GRF-06.

Concretely:
1. `internal/query.FileGraphNode` (`traverse.go:74-90`) gets a new field, e.g. `CommunityID
   int`, populated the same way `CycleID` is — a second graph-partitioning pass over the same
   `cycleAdj`-shaped adjacency (or a purpose-built one, since community detection algorithms
   like label propagation or Louvain are not the same as Tarjan SCC) inside `FileGraph()`,
   after the existing cycle-detection block (`traverse.go:305-330`).
2. `FileGraphResult` (`traverse.go:112-129`) gets a corresponding aggregate field (mirroring
   `CycleCount`, `traverse.go:128`), e.g. `CommunityCount int`.
3. `uiv1.FileGraphNode` (`ui.proto:824-839`) gets an additive `community_id` field at the next
   free number (5 — `cycle_id` is 4, no gap), and `FileGraphResponse` (`ui.proto:881-905`)
   gets a `community_count` field at 7 (`cycle_count` is 6, `ui.proto:905`).
4. **No new rpc** — `FileGraph` (the 12th method, already in `wantUIServiceMethods`) is
   extended additively, exactly as `GRF-04`'s cycle detection was folded into the same rpc
   rather than minting a separate one. This directly avoids the exact mistake the milestone's
   own question flags ("a prior milestone's research claimed an rpc supplied data it did not,
   costing an unbudgeted 13th rpc") — `FileGraph` genuinely already computes and returns the
   file-adjacency graph GRF-06 needs to partition; there is no missing rpc here.
5. `fileGraphToProto` (`internal/uiserver/handlers.go:997`) is modified, not replaced.

### New vs modified — GRF-06

| Component | New / Modified | File |
|---|---|---|
| Community-detection algorithm | **New** function | `internal/query/traverse.go` (near `stronglyConnectedCycles`) |
| `FileGraphNode.CommunityID`, `FileGraphResult.CommunityCount` | **Modified** (additive struct fields) | `internal/query/traverse.go:74-129` |
| `FileGraphNode.community_id`, `FileGraphResponse.community_count` | **Modified** (additive proto fields) | `internal/uiproto/uiv1/ui.proto` |
| `fileGraphToProto` | **Modified** | `internal/uiserver/handlers.go:997` |
| `uiProtoFieldNumbers` fixture | **Modified** (extended) | `internal/uiserver/readonly_test.go` |
| Graph view color/grouping-by-community | **New** rendering | `web/src/routes/graph/+page.svelte`, `web/src/lib/components/graph/file-graph-transform.ts` |

---

## 3. BRW-11 — editor handoff (configurable URI scheme)

### The exact architectural precedent already exists: `GetPermalink`

`internal/uiserver/permalink.go` is a **near-complete template** for this feature. It already:
- Confines a caller-supplied `path`+`line`+`end_line` through
  `eng.ValidateRepoRelativePath(path)` (`permalink.go:138`) — "the SAME `resolveSourcePath`
  gate `GetNodeDetail` and `Explore` already share (SRV-05)."
- Builds a URL server-side from validated inputs, never trusting the client to assemble one
  (`buildGitHubBlobURL`, `permalink.go:259-277`).
- Establishes the **exact precedent for the absolute-path tension** the milestone context
  flags. `GetHealthResponse.worktree_mismatch` (`ui.proto:685-695`, `WorktreeMismatch`
  message) is documented as *"the ONE scoped exception to this service's
  project_path/index_path privacy stance... because the warning is useless without naming
  both trees, and a clean tree still leaks nothing (T-04-09, accepted)."* This is a
  **maintainer-approved precedent for a deliberate, scoped absolute-path exception at the RPC
  boundary** — exactly the shape BRW-11 needs.

### Resolving "how the absolute path reaches the browser without weakening repo-root confinement"

The confinement gate (`ValidateRepoRelativePath`) is **only** about which files the server
will read/reference — not about whether the *response* may carry an absolute path once that
gate has passed. `GetPermalink` proves the pattern: validate the repo-relative path
server-side, then **compute the full destination URL server-side** (there, a GitHub blob URL;
here, an editor URI such as `vscode://file/<abs-path>:<line>`), and return only the
already-assembled string. The client never independently learns or reconstructs the server's
absolute filesystem prefix — it receives an opaque URI it can hand to `window.open()`. This is
architecturally identical to `WorktreeMismatch.worktree_root`/`index_root` carrying absolute
paths only when the maintainer scoped that exception explicitly (T-04-09) — BRW-11's plan
should record the same kind of explicit, scoped, one-line decision rather than reopening the
general privacy stance.

Given the whole UI is **already loopback-only bound** (v0.12.0 scope: "Loopback-only bind +
Origin/Host validation, read-only by construction"), the practical exposure of an editor URI
carrying the server's own absolute repo path is materially lower than a public service would
have — the same host that can reach the UI already has filesystem access to that path.

### Where the configurable URI scheme lives

`uiserver.Options` (`internal/uiserver/server.go:61-73`) currently has exactly two fields:
`RepoPath` and `Addr`. `Addr`'s own doc comment states the established pattern for adding a
new configurable knob:

```go
// Nothing outside this package writes this field in v1 — no flag, no
// environment variable, no configuration file reads it (D-08). A later
// `--host`/`--port` is therefore an override of a field that already
// exists, not a restructuring (SRV-03).
```

`internal/cli/ui.go`'s `newUiCmd()` (`ui.go:29-80`) registers exactly two flags today
(`--path`/`-p`, `--no-open`) and passes them into `uiserver.Options{RepoPath: start}`
(`ui.go:57`). **Recommendation**: add a new `Options.EditorURIScheme string` field (default
`"vscode"` when empty, following the `Addr`-defaults-to-`DefaultAddr`-when-empty pattern
already established at `server.go:66-67`), wired from a new `codegraph ui --editor` flag —
this is additive to `Options` and to `newUiCmd`, not a restructuring, exactly matching the
`Addr` precedent's own stated intent for future flags.

A **per-request override** (an optional field on the request message, so a user can switch
editors without restarting the server) is architecturally cheap to add later and should not
block v1 — same "field added speculatively can never be taken back" discipline the
`FileSymbolsRequest` doc comment states (`ui.proto:906-913`: *"There is no limit field in v1
... a limit can be added additively later"*). Client-side localStorage for a *display*
preference (which scheme label is shown/selected in a dropdown) is a legitimate frontend-only
concern independent of the wire contract — it does not need any server awareness.

### Does it need a new rpc at all?

**Recommendation: no.** `GetPermalink` already accepts `path`+`line`+`end_line` and returns a
computed URL — the mechanically simplest, most economical extension is an additive
`editor_url` field on `GetPermalinkResponse` (`ui.proto:651-668`, next field number would be
4 after `url`=1, `availability`=2, `reason`=3), computed unconditionally alongside the GitHub
URL whenever the path resolves.

**The one real design risk with reusing `GetPermalink`**: its `PermalinkAvailability` enum
(`LINKABLE` / `LINKABLE_UNVERIFIED` / `NO_LINK`) encodes GitHub-specific uncertainty (remote
presence, commit SHA verification) that has **no analog** for an editor link — an editor URI
is either buildable (path resolves) or not (it doesn't), with no "unverified" middle state.
Mixing two different availability semantics onto one response risks exactly the kind of
"same field, two meanings" ambiguity this codebase's own conventions elsewhere go out of their
way to avoid (e.g. `NodeDefinition.detail_gathered` exists specifically so emptiness is never
overloaded with two meanings, `ui.proto:459-472`). If planning decides the semantics really
don't fit together, the fallback is a small, purpose-built new message
(`GetEditorLinkResponse`) reusing `ValidateRepoRelativePath` — **still no new rpc is
mandatory if it's folded as fields onto `GetNodeDetailResponse` or kept as a `GetPermalink`
extension with its own always-populated field and no availability enum at all** (simplest:
`editor_url` is just empty string when the path does not resolve, no enum needed, since there
is no "maybe available" state to express).

### New vs modified — BRW-11

| Component | New / Modified | File |
|---|---|---|
| `Options.EditorURIScheme` | **New** field (additive) | `internal/uiserver/server.go:61-73` |
| `--editor` flag | **New** flag | `internal/cli/ui.go` (`newUiCmd`) |
| `GetPermalinkResponse.editor_url` | **Modified** (additive field) | `internal/uiproto/uiv1/ui.proto:651-668` |
| Editor-URI builder function | **New** function, mirroring `buildGitHubBlobURL` | `internal/uiserver/permalink.go` (new sibling function, same file) |
| `GetPermalink` handler | **Modified** | `internal/uiserver/permalink.go:86-203` |
| `uiProtoFieldNumbers` fixture | **Modified** (extended) | `internal/uiserver/readonly_test.go` |
| UI "open in editor" affordance | **New** rendering | `web/src/lib/components/browse/SourcePane.svelte` (or wherever the permalink button already lives) |

---

## 4. BRW-10 — "where am I" breadcrumb

### The exact data already exists — via a call that is not yet wired into the file view

`GetNodeDetailResponse` in `NODE_DETAIL_MODE_FILE` mode returns **only** `path` and `source`
(`ui.proto:493-524`: *"NODE_DETAIL_MODE_FILE | path, source"*) — **no per-symbol line ranges**.
The breadcrumb cannot be computed from this response alone.

But `FileSymbols` — the 13th rpc, already shipped in Phase 5 (`readonly_test.go`'s
plan-05-06 comment) — returns exactly the data needed:

```proto
message FileSymbolsResponse {
  repeated Node symbols = 1;   // reuses the shared Node message
  int32 total_count = 2;
  bool truncated = 3;
}
```

And the shared `Node` message (`ui.proto:131-145`) carries `start_line = 7` and `end_line = 8`
for every symbol. This is confirmed by `FileSymbols`'s own doc comment
(`ui.proto:906-913`), which explicitly contrasts it with `GetNodeDetail`'s file mode: *"...
separately from GetNodeDetail (whose file mode carries a path and a source blob only, and does
not enumerate symbols) so that every-navigation message stays cheap (D-01 precedent)."* —
i.e., the split between "cheap file view" and "symbol enumeration" was a **deliberate,
already-made** architectural decision, not an oversight.

### Confirmed: `FileSymbols` is not yet consumed by the file-detail view

`rg -n "fileSymbols|FileSymbols" web/src` returns exactly three hits:
`web/src/routes/graph/+page.svelte`, `web/src/lib/gen/ui_pb.ts` (generated client), and
`web/src/lib/components/graph/file-graph-transform.ts` — all on the `/graph` route (drilling
into a file for its symbol list from the file/package graph view). The file-source rendering
component, `web/src/lib/components/browse/SourcePane.svelte` (found via
`rg -ln "GetNodeDetail|getNodeDetail" web/src`, alongside `web/src/routes/browse/+page.svelte`),
does **not** currently call `FileSymbols`.

### Recommendation: no backend work at all — reuse `FileSymbols`, purely a frontend integration

This directly answers the milestone's open question ("whether a `FileSymbols`-style call must
be reused") — yes, and it is a pure reuse, zero proto/Engine changes:
1. When the browse/file view opens a file in `NODE_DETAIL_MODE_FILE`, also call `FileSymbols`
   for that path (a second, parallel rpc call — the same two-call pattern the `/graph` route
   already exercises for its own file drill-down).
2. Client-side, on scroll, compute the containing symbol by interval containment over each
   returned `Node.start_line`/`end_line` against the current scroll position — pure frontend
   logic, no new wire shape.
3. `MaxFileSymbols` (referenced in `ui.proto:910`, `internal/query.Engine.FileSymbols`,
   file `internal/query/filesymbols.go`) already caps and truncates server-side — the
   breadcrumb calculation should treat a `truncated=true` response as "breadcrumb may be
   incomplete for very large files" rather than erroring.

### New vs modified — BRW-10

| Component | New / Modified | File |
|---|---|---|
| Breadcrumb component + scroll-position tracking | **New** | `web/src/lib/components/browse/SourcePane.svelte` (or a new sibling component) |
| `FileSymbols` client call wiring | **Modified** (new call site, existing rpc) | `web/src/routes/browse/+page.svelte` |
| Everything backend (`internal/query`, `internal/uiserver`, `ui.proto`) | **Untouched** | — |

This is the cheapest of the four UI follow-ons in real engineering terms, and the ROADMAP's
own "ordered by increasing scope" placement (BRW-11, BRW-10, HLT-04, GRF-06) is consistent
with this finding only if BRW-10 is read as "smaller than HLT-04/GRF-06" rather than smaller
than BRW-11 — worth flagging to the roadmapper: on this research, BRW-10 is architecturally the
**smallest** item in the whole follow-through set (frontend-only), smaller than BRW-11 (which
touches `Options`, a new flag, and a proto field even in the reuse-`GetPermalink` design).

---

## 5. 999.2 — tmux real-PTY e2e harness

### There is no existing precedent for building/exec'ing the release binary in tests

Confirmed by direct search: `rg -n "go build -o" internal -g '*_test.go'` returns **zero
hits**, and `rg -n "tmux"` across the whole repository (`.go`, `.yml` files) returns **zero
hits**. The only `exec.Command` usage in `internal/cli/*_test.go` is `git`
(`githooks_test.go:25`, `notice_test.go:30`) — never the `codegraph` binary itself. This
confirms the milestone context's framing directly: this is genuinely new infrastructure, not
an extension of an existing exec-based test.

### Why the existing suite structurally cannot cover this (the ROADMAP's own stated reason)

The existing "piped integration tests" in `internal/cli` invoke Cobra commands **in-process**
against buffered stdin/stdout — this never allocates a real pty, so escape-sequence
handshakes (DECRQM capability probes) and alternate-screen entry/exit (which only a real
terminal driver honors) are invisible to them. `ROADMAP.md`'s own Phase 999.2 scope entry
names the exact two bugs (G-07-1, G-07-2) that both "the full piped automated suite AND a deep
multi-agent code review missed" for precisely this reason.

### Where it should live

No `//go:build` tag convention exists in this repo for anything but OS gating today
(`internal/daemon/stop_posix.go: //go:build !windows`, `procstart_linux.go: //go:build linux`,
`procstart_other.go: //go:build !linux` — `rg -n "^//go:build" internal`). A new **feature**
build tag (e.g. `//go:build tmux`) would be the first of its kind in this codebase — this is
worth flagging explicitly to whoever plans it, since it is establishing a new convention, not
following one.

**Recommended location**: a new package, e.g. `test/tmux/` (top-level, alongside no existing
sibling — there is currently no top-level `test/` directory; verify at plan time whether one
should be introduced or whether this belongs under `internal/tmuxtest` following the
`internal/`-only convention every other test package in this repo uses). Given every existing
test package in this repo lives under `internal/`, favor `internal/tmuxtest/` unless a
deliberate decision is made to break that pattern for e2e-only scope.

### How it obtains the built binary

No existing test builds the binary via `go build`. The harness needs a `TestMain` (or a
`sync.Once`-guarded helper) that runs `go build -o <tmp>/codegraph .` once per test binary
invocation, or — more in keeping with this repo's CI-first discipline (GoReleaser owns the
release build) — consumes a path to an **already-built** binary passed via an environment
variable set by the CI job (e.g. `CODEGRAPH_BIN`), falling back to a local `go build` for
local development. This mirrors the project's own stated preference (v0.5.0 scope: "the
release pipeline is now one `goreleaser release` invocation") for not duplicating build logic.

### How CI provides tmux

`.github/workflows/ci.yml`'s job list (`rg -n "^  [a-zA-Z0-9_-]+:$"`) currently has: `test`,
`govulncheck`, `tool-vuln`, `reproducibility`, `perf-regression`, `actionlint`,
`transcript-freeze`, `goreleaser-check`. None install `tmux`. A new job (or a new step within
`test`) would need `apt-get install -y tmux` (or the `ubuntu-latest` runner's package
manager equivalent) — cheap, and the ROADMAP scope already anticipates this ("gate the suite
behind a build tag / CI job that has tmux available; skips cleanly elsewhere").

### How it relates to the existing piped integration tests

Not a replacement — an **additional rung**. `ROADMAP.md`'s own framing: *"the missing rung
between the piped never-hang/byte-identity integration tests (necessary, TTY-blind) and manual
human UAT (thorough, unautomated)."* The existing `internal/cli` piped suite stays exactly as
is; this is a wholly new, orthogonal test surface that exercises the **same built binary**
through a **different I/O path** (a real pty via tmux send-keys/capture-pane) rather than a
buffered `bytes.Buffer`.

### Build-order implication (see §7's global ordering)

The ROADMAP explicitly states this "lands **before** the UI work so that work has a real-
terminal rung" — but note this refers to the *existing* TUI surfaces (daemon picker,
install/uninstall checkbox picker), not to the new `codegraph ui` web surface, which has no
terminal-rendering component at all and is therefore unaffected by tmux ordering.

### New vs modified — 999.2

| Component | New / Modified | File |
|---|---|---|
| tmux harness package | **New** | `internal/tmuxtest/` (recommended) or `test/tmux/` |
| Binary-build/locate helper | **New** | same package |
| `//go:build tmux` tag | **New convention** — first feature build tag in this repo | same package |
| CI job/step installing tmux | **New** | `.github/workflows/ci.yml` |
| `Taskfile.yml` `test:tmux` leg | **New** task; **not** added to the `test:` wrapper by default (see `TestTaskfileWrapperIsSerial`, `internal/upgrade`, which set-equality-guards the wrapper's leg list — a new leg must be deliberately added there or deliberately left out, either way a conscious edit) | `Taskfile.yml` |

---

## 6. Guard hardening — five items, each with exact source location

### 999.4 — `CheckRegression` current-metrics positivity

**Exact location**: `internal/bench/regression.go:110-111`:
```go
if baseline.PeakRSSBytes <= 0 {
    return fmt.Errorf("bench: invalid baseline: PeakRSSBytes must be positive, got %d", baseline.PeakRSSBytes)
}
```
This is the **only** positivity check in the function — confirmed by `rg -n "PeakRSSBytes|Throughput" internal/bench/*.go`, which shows no equivalent check on `current`. `current.PeakRSSBytes`,
`current.FilesPerSec` (and any other current-side metric consumed by the relative-delta and
absolute-ceiling checks at `regression.go:115-133`) are unguarded.

**Non-vacuous replacement structurally**: mirror the existing baseline check exactly, on
`current`, naming the degenerate field — `regression.go` already establishes the pattern (the
GOOS/GOARCH and Runner checks earlier in the same function, `:51-59`, `:74-...`, are
similarly named, field-specific refusals). Demonstrated RED per the repo's standing rule: a
test constructing `current.PeakRSSBytes = 0` with an otherwise-matching, non-regressing frame,
asserting `CheckRegression` returns a non-nil error naming `PeakRSSBytes` — currently this
returns `nil` (the exact bypass `10-SECURITY.md` and code-review finding WR-06 already
reproduced).

### T-01-18 — `internal/query` dependency-direction archtest

**Reusable shape already exists**: `internal/graphstore/archtest/import_graph_test.go` (full
file read above) is a **direct, one-for-one template**:
```go
cfg := &packages.Config{
    Mode:  packages.NeedImports | packages.NeedName | packages.NeedDeps,
    Tests: true,
}
pkgs, err := packages.Load(cfg, "github.com/seanb4t/codegraph-go/...")
```
plus a **non-vacuity positive control** (`archtest`'s own: *"if internal/graphstore itself no
longer imports pebble/v2 ... the check above is vacuously true for the wrong reason ... this
test cannot verify enforcement"*) — the todo's own suggested fix already names the exact same
pattern with `google.golang.org/protobuf` (via `internal/schema`) as the positive control.

**Recommended location**: a new sibling test file — either a new `internal/query/archtest/`
subpackage (mirroring `internal/graphstore/archtest`'s own placement one level down from the
package it guards) or directly inside `internal/query` as `archtest_test.go`. Given
`internal/graphstore/archtest` is a **separate package** specifically so the archtest itself
cannot accidentally import the thing it's checking, the same isolation argument favors a
separate `internal/query/archtest/` package over a same-package test file.

**Assertions**: `go list -deps` (or `packages.Load` with `NeedDeps`) over
`internal/query`'s package path contains zero entries for `connectrpc.com/connect` and zero
for `internal/uiproto` (any subpackage), with `google.golang.org/protobuf` asserted present as
the positive control (arriving legitimately via `internal/schema`).

### `release:dry-run-signed` additions-only diff guard

**Exact location, verified at HEAD** (line numbers differ from the todo file, which predates
several unrelated Taskfile edits — cite the live location, not the stale one):
- `Taskfile.yml:1771` — task `release:dry-run-signed:`
- `Taskfile.yml:1841` — the awk anchor: `/^      - "sign-blob"$/ { print keyline }`
- `Taskfile.yml:2225` / `:2381` — the second consumer, `release:rehearse-notarize:`, reusing
  the identical anchor (confirmed by the task's own comment at `:2367-2368`: *"the SAME awk
  anchor and additions-only diff guard `release:dry-run-signed` already uses"*).

**Non-vacuous replacement**: add a positive assertion immediately after the awk injection in
both tasks — count the injected `--key=` lines and hard-fail if the count is zero, **before**
running the additions-only diff. Demonstrated RED by deliberately re-indenting the `sign-blob`
args block in a generated copy (not the committed file) and observing the new assertion fail
loudly rather than the diff silently reporting zero additions.

### `post-release-verify.yml` event-aware conclusion guard

**Exact locations**: `.github/workflows/post-release-verify.yml:303`, `:408` (job-level `if:`
guards), cross-checked by `internal/upgrade/release_workflow_shape_test.go:1369`
(`TestPostReleaseJobsDeclareCheckoutPolicy`), which enumerates the same file's jobs via
`postReleaseCheckoutShapes` (`release_workflow_shape_test.go:1306-1325`) but — confirmed by
reading `fullWorkflowJob`'s struct definition (`:1042-1046`) — **carries no `If` field at
all**: `Env`, `Permissions`, `Steps` only. This directly confirms the todo's claim: no
existing parser even sees the `if:` guard, let alone asserts it.

**Non-vacuous replacement, reusing existing scaffolding**: add `If string \`yaml:"if"\`` to
`fullWorkflowJob` (additive struct field, `release_workflow_shape_test.go:1042-1046`), extend
`postReleaseCheckoutShape` (`:1297-1302`) with the same field, and extend
`postReleaseCheckoutShapes` (`:1306-1325`) to capture it — the exact wrapper
`TestPostReleaseJobsDeclareCheckoutPolicy` (`:1582`) already iterates every job and classifies
it (`latestVerifierJobIDs`/`releaseMatchedTestJobIDs`, referenced at `:1593-1596`); add a new
assertion in that same loop requiring every job's `If` to contain the verbatim disjunct
`github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'`.
Pair with a non-vacuity companion mirroring
`TestAppleSecretsScopedToSingleReleaseJob_EmptyDocIsError` (`:1277-1285`): an empty/malformed
document must error, never silently pass.

### `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets`

**Exact location**: `internal/upgrade/release_workflow_shape_test.go:1565-1574`. Confirmed
tautological by direct read:
```go
func TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets(t *testing.T) {
    releasePleaseAppSecretNames := []string{"APP_ID", "APP_PRIVATE_KEY"}
    for _, tapName := range homebrewTapCredentialNames[:2] {
        for _, rpName := range releasePleaseAppSecretNames {
            if tapName == rpName { t.Errorf(...) }
```
Both sides are in-test constants (`homebrewTapCredentialNames` at `:1366-1370`); the function
reads zero workflow files.

**Non-vacuous replacement, reusing existing scaffolding**: `findHomebrewTapCredentialReferences`
(`:1382-1415`) already exists and is exercised by the sibling test
`TestHomebrewTapTokenScopedToReleaseJob` (`:1450-1558`) — it decodes real workflow files
(`decodeFullWorkflowDoc`, `:1059-1069`) and finds actual secret-name references at
env:/with: scope. The fix: read `release.yml`'s actual referenced secret names (via this
existing walker, filtered to the release job) and `release-please.yml`'s actual referenced
secret names (the same walker, parameterized with `releasePleaseAppSecretNames`), then assert
the two **observed, file-derived** sets are disjoint — never comparing two hardcoded literals
against each other. Add the same positive-floor discipline
`TestHomebrewTapTokenScopedToReleaseJob` already documents (`:1495-1509`, rule `84d1gfpywd`):
both sets must be non-empty before the disjointness assertion is trusted. Demonstrated RED by
pointing `release.yml`'s mint at `APP_ID`/`APP_PRIVATE_KEY` (the todo's own suggested
mutation).

### New vs modified — guard hardening (all five)

Every item in this section is a **modification** to an existing file — no new packages, no new
production code paths, purely test/CI-config hardening. This is the cheapest set in the whole
milestone in terms of surface area touched, though not necessarily in effort (each requires a
genuine RED demonstration per this repo's `84d1gfpywd` rule, not just a code change).

| Item | File(s) modified |
|---|---|
| 999.4 | `internal/bench/regression.go`, `internal/bench/regression_test.go` |
| T-01-18 | New file: `internal/query/archtest/import_graph_test.go` (or in-package) |
| `dry-run-signed` guard | `Taskfile.yml` (both `release:dry-run-signed:` and `release:rehearse-notarize:`) |
| `post-release-verify` guard | `internal/upgrade/release_workflow_shape_test.go` |
| Tap-secret distinctness | `internal/upgrade/release_workflow_shape_test.go` |

---

## 7. Suggested build order

Ordering rationale, not a rigid phase-count prescription — the roadmapper decomposes phases;
this is the dependency graph that should drive that decomposition.

### Wave 0 — guard hardening (§6), fully independent, do first

All five guard-hardening items touch disjoint files, have zero dependency on each other or on
anything else in this milestone, and each is small (test/CI-config only). They can run in
parallel across sub-agents/plans and should land **before** any UI work, matching the
ROADMAP's own stated rationale ("every item in it already has a written RED demonstration in
mind"). T-01-18 in particular should land early since it **governs** how the HLT-04
discovery-exclusion helper is allowed to be wired (internal/query must not gain a new
dependency edge toward internal/indexer without violating the same invariant the archtest
checks the other direction — verify at plan time whether the archtest should also assert
`internal/query -> internal/indexer` is forbidden or merely `internal/query -> connectrpc`/
`internal/uiproto`; the todo's stated scope is the latter only).

### Wave 1 — 999.2 tmux harness, before any TUI-adjacent verification

Genuinely new infrastructure (new package, new build tag, new CI job) with no dependency on
Wave 0 or on any of the UI follow-ons — it exercises the *terminal* UI (daemon/install
pickers), not the *web* UI. Can run in parallel with Wave 0. The ROADMAP places it "before the
UI work" for terminal-rung reasons that do not extend to `codegraph ui` (a browser surface with
no pty involvement) — sequencing it before §§1-4 is about narrative/CI-availability
convenience, not a hard dependency.

### Wave 2 — UI follow-through, ordered by real dependency and scope, not the ROADMAP's listed order

The ROADMAP lists BRW-11, BRW-10, HLT-04, GRF-06 "by increasing scope." This research finds a
different real ordering by scope and risk:

1. **BRW-10 (breadcrumb)** — smallest by a wide margin: **zero backend changes**, pure
   frontend reuse of an already-shipped rpc (`FileSymbols`). No proto edit, no new Engine
   method, no new fixture entries. Should go first regardless of the ROADMAP's listed order,
   since it has no risk of blocking anything else and can be demonstrated complete fastest.
2. **BRW-11 (editor handoff)** — small: one new `Options` field, one new CLI flag, one
   additive proto field on an existing message (`GetPermalinkResponse`), reusing
   `ValidateRepoRelativePath` entirely. The one open design question (whether the
   `PermalinkAvailability` enum semantics fit an editor link) should be resolved at discuss-
   phase time before planning, since it changes whether this is "add one field" or "add one
   small new message."
3. **HLT-04 (coverage denominator)** — medium: genuinely new discovery-time logic
   (`internal/indexer` or a query-local duplicate), a new Engine method, and an additive
   `GetHealthResponse` field/message. Depends on T-01-18 (Wave 0) if the new discovery-
   exclusion helper is placed in a way that could create a forbidden dependency edge —
   verify the archtest's scope covers this before writing the helper.
4. **GRF-06 (community clustering)** — largest of the four: a new graph-partitioning
   algorithm (a real algorithmic choice — label propagation, Louvain, etc. — not just wiring),
   plus additive fields on `FileGraphNode`/`FileGraphResult`/their proto projections, plus
   frontend rendering (grouping/coloring by community) on top of the `/graph` view that
   already exists. This is also the item most likely to hit the guava-scale rendering ceiling
   the milestone context explicitly warns about (v0.12.0's GRF-01 threshold failure on a
   3,233-node view) — GRF-07 (whole-symbol graph) was explicitly parked for exactly this
   reason, and community clustering adds visual complexity to the same file-graph view GRF-01
   already found expensive. Budget a measured-render-time checkpoint for this item
   specifically, mirroring GRF-01's own "committed the threshold before measuring" discipline.

### Wave 3 — DOCS-05 (self-authored CLI reference + drift guard)

Independent of everything above, but logically last since it should document the **final**
flag surface, including any new flags §3 (BRW-11's `--editor`) introduces. The exact reusable
shape is `internal/cli/flag_parity_test.go` (deleted at v0.11.0, commit `5139e60c`, git-log
confirmed) — resurrect its `newRootCmd()` + `cmd.Flags().VisitAll` recursive walk +
substring-presence-in-doc pattern verbatim, retargeted from `docs/FLAG-PARITY.md` at
`internal/cli/flag_parity_test.go:16` to a new `docs/CLI-REFERENCE.md`. This is a fully
proven, previously-shipped pattern — the lowest-risk item in the whole milestone.

### Summary table

| Wave | Items | Parallelizable within wave | Blocks |
|---|---|---|---|
| 0 | 999.4, T-01-18, dry-run-signed guard, post-release-verify guard, tap-secret test | Yes, all 5 | Wave 2's HLT-04 (via T-01-18) |
| 1 | 999.2 tmux harness | N/A (one item) | Nothing downstream (terminal-only) |
| 2a | BRW-10 | N/A | Nothing |
| 2b | BRW-11 | N/A | Nothing (independent of 2a) |
| 2c | HLT-04 | Depends on Wave 0 (T-01-18) | Nothing |
| 2d | GRF-06 | Independent, but budget render-time risk | Nothing |
| 3 | DOCS-05 | Should follow Wave 2 (documents its flags) | — |

---

## Sources

Every claim above is either a direct quote from a file at this repository's HEAD (2026-09-08,
tag `v0.12.0`, commit `17e88d67`) verified via `Read`/`rg`/`git log`, or is explicitly labelled
as inference. No external documentation lookup was needed or performed — this is a pure
codebase-integration research task, and the source of truth is the repository itself.

- `internal/query/engine.go`, `internal/query/status.go`, `internal/query/traverse.go` — Engine surface, `Status()`'s filesystem-degrade precedent, `FileGraph()`'s fresh-per-call cycle detection
- `internal/indexer/discover.go`, `internal/indexer/extract.go`, `internal/indexer/resolve.go` — discovery/extraction exclusion paths, existing `File.Errors` persistence
- `internal/schema/graph.proto` — on-disk schema, reserved annotation ranges, additive-only discipline
- `internal/uiproto/uiv1/ui.proto` — wire schema, all 14 rpcs, message field numbers
- `internal/uiserver/readonly_test.go` — `mutatingVerbs`, `wantUIServiceMethods`, `uiProtoFieldNumbers` fixture mechanics
- `internal/uiserver/permalink.go`, `internal/uiserver/server.go`, `internal/uiserver/handlers.go` — GetPermalink precedent, Options shape, mapper-function convention
- `internal/cli/ui.go` — `codegraph ui` command, flag registration
- `internal/bench/regression.go` — `CheckRegression` positivity gap
- `internal/graphstore/archtest/import_graph_test.go` — reusable archtest shape for T-01-18
- `Taskfile.yml` — `release:dry-run-signed`, `release:rehearse-notarize`, `test:` wrapper, `TestTaskfileWrapperIsSerial` reference
- `internal/upgrade/release_workflow_shape_test.go` — `fullWorkflowJob`/`fullWorkflowDoc` parsing shape, `TestPostReleaseJobsDeclareCheckoutPolicy`, `TestHomebrewTapAppSecretsDistinctFromReleasePleaseAppSecrets`, `findHomebrewTapCredentialReferences`
- `.github/workflows/ci.yml`, `.github/workflows/post-release-verify.yml` — job lists, conclusion guard locations
- `git log --oneline --all -- docs/FLAG-PARITY.md internal/cli/flag_parity_test.go` + `git show v0.10.0:internal/cli/flag_parity_test.go` — DOCS-05's reusable prior-art
- `web/src/routes/{health,graph,browse}/`, `web/src/lib/components/browse/SourcePane.svelte` — confirmed frontend consumption gaps for BRW-10/HLT-04/GRF-06
- `.planning/PROJECT.md` — milestone goal, target features, key context
- `.planning/todos/pending/*.md` — exact guard descriptions and threat refs for §6
- `.planning/ROADMAP.md` — Backlog 999.2 and 999.4 verbatim scopes
