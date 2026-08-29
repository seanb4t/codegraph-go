# Phase 4: Query Workbench & Index Health - Pattern Map

**Mapped:** 2026-08-29
**Files analyzed:** ~20 (new + modified)
**Analogs found:** 17 / 20 (3 have no true analog — new library-composition surfaces, noted below)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/uiproto/uiv1/ui.proto` (+`GetHealth` rpc, +response msg) | route/config (proto IDL) | request-response | `GetStatusResponse` message + `GetPermalink` rpc addition, same file | exact |
| `internal/uiserver/handlers.go` (+`GetHealth` handler, +mapper) | controller | request-response, CRUD-read | `Callers`/`Callees` handlers + `statusToProto` mapper, same file | exact |
| `internal/uiserver/readonly_test.go` (2 literal edits) | test | — | itself, same shape as the prior `GetPermalink` addition (comment documents the precedent) | exact |
| `internal/query/files.go` (glob swap) | service | transform | itself — two existing `filepath.Match` call sites | exact |
| `web/src/lib/workbench-url.ts` | utility (URL grammar) | transform | `web/src/lib/browse-url.ts` | role-match, deliberate divergence on D-11 |
| `web/src/lib/components/ui/table/*` (9 files, vendored) | component | render | none in-tree (new shadcn-svelte registry addition) | no analog — vendored, see below |
| `web/src/lib/components/ui/tabs/*` (5 files, vendored) | component | render | none in-tree (new shadcn-svelte registry addition) | no analog — vendored, see below |
| `web/src/lib/components/workbench/DataTable.svelte` | component | render, request-response | `web/src/lib/components/browse/NeighborsPanel.svelte` (props-in, RPC-client-shaped data in, no self-navigation) | role-match |
| `web/src/lib/components/workbench/*-columns.ts` (4 files) | config (column defs) | transform | none — new TanStack `ColumnDef<Location>` shape, no in-tree precedent | no analog — new library surface |
| `web/src/lib/components/workbench/FilePicker.svelte` | component | request-response, debounced | `web/src/lib/search.ts` (controller pattern) + `web/src/lib/components/browse/SearchPanel.svelte` (Command usage) | role-match |
| `web/src/routes/workbench/+page.svelte` (filled) | route (SvelteKit page) | request-response | `web/src/routes/health/+page.svelte`'s sibling placeholder / Browse's `+page.svelte` orchestration shape | role-match |
| `web/src/routes/health/+page.svelte` (filled) | route (SvelteKit page) | request-response | `web/src/lib/components/StatusBanner.svelte` + `web/src/lib/status.ts`'s gate contract | role-match |
| `web/tests/workbench-url.test.ts` | test | — | `web/tests/browse-url.test.ts` | exact (structure), deliberate divergence on multi-value key |
| `web/tests/health.test.ts` (or similar) | test | — | `web/tests/status.test.ts` | role-match |
| `Taskfile.yml` (+`web:components:drift` target) | config (build tooling) | batch | `proto:drift` (regenerate-and-byte-compare shape), NOT `web:drift` (hash-and-compare shape) | exact (shape to copy), explicit divergence documented |
| `go.mod` (+`doublestar/v4`) | config | — | n/a (dependency addition, no code pattern) | n/a |

## Pattern Assignments

### `internal/uiproto/uiv1/ui.proto` (+`GetHealth` rpc)

**Analog:** same file — `GetStatusResponse` message (lines 100-130) and the `GetPermalink` rpc/message addition (lines 50-67, 545-597) as the precedent for "add the 11th rpc additively."

**RPC declaration pattern** (`ui.proto:50-67`, verbatim):
```protobuf
service UIService {
  rpc GetStatus(GetStatusRequest) returns (GetStatusResponse);
  ...
  rpc Callers(CallersRequest) returns (CallersResponse);
  ...
  rpc Impact(ImpactRequest) returns (ImpactResponse);
  rpc Affected(AffectedRequest) returns (AffectedResponse);
  ...
  // GetPermalink is plan 03-05's tenth rpc (D-06): it turns a repo-relative
  // path and optional line anchor into a permalink URL...
  rpc GetPermalink(GetPermalinkRequest) returns (GetPermalinkResponse);
}
```
Copy this shape for the 11th rpc: `rpc GetHealth(GetHealthRequest) returns (GetHealthResponse);` — additive, with a doc comment stating why it is separate from `GetStatus` (cite D-01/D-02 verbatim as `GetPermalink`'s comment cites D-06).

**`Location` message, the shared row shape all four analyses already emit** (`ui.proto:100-105`, verbatim — this is what every `columns.ts` in D-06 is built against, do not re-derive it):
```protobuf
message Location {
  string name = 1;
  string kind = 2;
  string file_path = 3;
  int32 start_line = 4;
}
```

**Map field precedent** (`internal/schema/graph.proto:94`, verbatim, cited by RESEARCH):
```protobuf
map<string, string> metadata = 7;
```
`GetHealthResponse`'s `FilesByLanguage`/`NodesByKind`/`EdgesByKind` fields should read `map<string, int64> files_by_language = N;` etc., following this exact syntax with `int64` values.

**Nested message field for `WorktreeMismatch`** (per RESEARCH Pitfall 5, `internal/gitmeta/detect.go:10-13`, verbatim):
```go
type Mismatch struct {
	WorktreeRoot string `json:"worktreeRoot"`
	IndexRoot    string `json:"indexRoot"`
}
```
Define a plain (non-`optional`) `WorktreeMismatch` message mirroring these two fields; proto3 message fields are natively nil-able in generated Go — no `optional` keyword needed (unlike the scalar `optional int32 line = 3;` precedent at `ui.proto:338`, which does NOT apply here).

---

### `internal/uiserver/handlers.go` (+`GetHealth` handler, +`healthToProto`-style mapper)

**Analog:** `statusToProto` (handlers.go:218-227) for the mapper shape; `Callers` handler (handlers.go:461-478) for the `withEngine` wrapping shape — **NOT** `GetStatus`'s handler, which is a deliberate, documented exception (see below).

**Mapper pattern** (`internal/uiserver/handlers.go:218-227`, verbatim):
```go
func statusToProto(result query.StatusResult, commitSHA string) *uiv1.GetStatusResponse {
	return &uiv1.GetStatusResponse{
		Initialized: result.Initialized,
		Version:     result.Version,
		NodeCount:   result.NodeCount,
		EdgeCount:   result.EdgeCount,
		FileCount:   result.FileCount,
		Stale:       result.Stale,
		CommitSha:   commitSHA,
		StoreExists: true,
	}
}
```

**Handler pattern to copy — `Callers`** (`internal/uiserver/handlers.go:461-478`, verbatim):
```go
func (s *uiService) Callers(ctx context.Context, req *connect.Request[uiv1.CallersRequest]) (*connect.Response[uiv1.CallersResponse], error) {
	var resp *uiv1.CallersResponse
	err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
		result, err := eng.Callers(req.Msg.GetSymbol(), int(req.Msg.GetLimit()))
		if err != nil {
			return err
		}
		resp = &uiv1.CallersResponse{
			Symbol:  result.Symbol,
			Callers: locationsToProto(result.Callers),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
```
`GetHealth`'s handler should follow this exact three-part shape: `withEngine(ctx, s.repoPath, func(eng *query.Engine) error { ... })`, call `eng.Status(ctx)` (same call `GetStatus` already makes) to get `query.StatusResult`, then call `gitmeta.DetectIndexMismatch` **only here** (D-02), then map via a new `healthToProto(result, mismatch)` function shaped like `statusToProto`.

**Do NOT copy `GetStatus`'s handler shape** (`handlers.go:305-347`) — its `openEngine` direct-call, degrade-and-still-answer pattern is a documented, single exception (its own doc comment: *"GetStatus is the single, deliberate exception to the withEngine rule (SRV-04, D-16, plan 01-11)"*). `GetHealth` has no such partial-availability requirement — it should use `withEngine` like every other handler.

**Locations mapper reused unchanged** (`internal/uiserver/handlers.go:246-253`, verbatim) — not needed by `GetHealth` itself, but is the mapper convention every `columns.ts` on the frontend is keying off of:
```go
func locationsToProto(locs []query.Location) []*uiv1.Location {
	out := make([]*uiv1.Location, len(locs))
	for i, l := range locs {
		out[i] = locationToProto(l)
	}
	return out
}
```

**Engine data source** — `query.StatusResult` already carries every value `GetHealth` needs (`internal/query/status.go:47-65`), including `WorktreeMismatch *gitmeta.Mismatch` as a struct field already populated by `eng.Status(ctx)` — verify at implementation time whether `eng.Status` already calls `gitmeta.DetectIndexMismatch` internally (it appears to, per `StatusResult`'s own field and mapping-table comment) or whether D-02's "only inside GetHealth" constraint requires calling `Engine.WorktreeMismatch()` (referenced in status.go's comment) separately and NOT through `eng.Status`. This determines whether `GetHealth`'s handler makes one `eng.Status(ctx)` call or two calls (`eng.Status` variant that skips the mismatch + a separate `eng.WorktreeMismatch()` call) — read `internal/query/status.go`'s `Status` method body before writing the handler to settle this.

---

### `internal/uiserver/readonly_test.go` (literal updates)

**Analog:** itself — the exact precedent already recorded in its own comments for the prior `GetPermalink` addition.

**Positive set-equality fixture** (`internal/uiserver/readonly_test.go:31-42`, verbatim, current 10-entry state):
```go
var wantUIServiceMethods = map[string]struct{}{
	"GetStatus":     {},
	"Search":        {},
	"Files":         {},
	"Callers":       {},
	"Callees":       {},
	"Impact":        {},
	"Affected":      {},
	"GetNodeDetail": {},
	"Explore":       {},
	"GetPermalink":  {},
}
```
Add `"GetHealth": {}` as an 11th entry, following the exact comment precedent already in the file ("UPDATED at plan 03-05: GetPermalink...") — add a parallel "UPDATED at [this plan]: GetHealth..." comment.

**Count literal** (`internal/uiserver/readonly_test.go:61`, verbatim):
```go
if len(got) != 10 {
	t.Fatalf("uiv1connect.UIServiceHandler has %d methods, want exactly 10: %v", len(got), got)
}
```
Change `10` → `11` in both the literal and the format string.

**mutatingVerbs guard — do NOT touch** (`internal/uiserver/readonly_test.go:89-93`, cited in RESEARCH Pitfall 1 and CONTEXT D-01's corrected block):
```go
var mutatingVerbs = []string{
	"Create", "Update", "Delete", "Remove", "Set", "Put", "Post",
	"Write", "Add", "Insert", "Mutate", "Patch", "Modify", "Sync",
	"Reindex", "Index", "Clear", "Reset", "Save",
}
```
`"Index"` is in this list. The rpc MUST be named `GetHealth` (not `GetIndexHealth`) to avoid a `strings.Contains` false-positive here. Do not add an allowlist exception to this list.

---

### `internal/query/files.go` (glob swap, D-14)

**Analog:** itself — two existing call sites, both must change.

**Sanity-check call site** (`internal/query/files.go:146-149`, verbatim):
```go
if opts.Pattern != "" {
    if _, err := filepath.Match(opts.Pattern, "sanity-check"); err != nil {
        return FilesResult{}, fmt.Errorf("query: invalid pattern %q: %w", opts.Pattern, err)
    }
}
```

**Real match call site** (`internal/query/files.go:167-177`, verbatim):
```go
if opts.Pattern != "" {
    matched, err := filepath.Match(opts.Pattern, p)   // p is already filepath.ToSlash(f.Path) — line 157
    if err != nil {
        return FilesResult{}, fmt.Errorf("query: invalid pattern %q: %w", opts.Pattern, err)
    }
    if !matched {
        continue
    }
}
```
Replace `filepath.Match` with `doublestar.Match` in both call sites — import `github.com/bmatcuk/doublestar/v4`. Use `Match`, not `PathMatch`, because `p` is already forward-slashed via `filepath.ToSlash` before matching (`files.go:157`), and `doublestar.Match` "always uses '/' as the path separator" per its own doc comment — this matches what the call site already produces. Error type/wrapping (`fmt.Errorf("query: invalid pattern %q: %w", ...)`) is unchanged — `doublestar.Match`'s error type is compatible with the existing wrap.

---

### `web/src/lib/workbench-url.ts` (D-10, D-11, D-12)

**Analog:** `web/src/lib/browse-url.ts` — imports its shape-validation primitives, but the reader for the repeated key MUST diverge (D-11).

**Known-key set + shape-integer primitives to import, NOT redefine** (`web/src/lib/browse-url.ts:22-24,42-67`, verbatim):
```ts
export const BROWSE_PARAM_KEYS = ['symbol', 'file', 'line', 'depth', 'limit', 'q'] as const;

const KNOWN_KEYS: ReadonlySet<string> = new Set(BROWSE_PARAM_KEYS);
```
```ts
const INTEGER_SHAPE = /^-?\d+$/;

export function isShapeInteger(raw: string): boolean {
	return INTEGER_SHAPE.test(raw) && Number.isSafeInteger(Number(raw));
}

function parseShapeInteger(raw: string | null): number | undefined {
	if (raw === null || raw === '') return undefined;
	if (!isShapeInteger(raw)) return undefined;
	return Number(raw);
}
```
`workbench-url.ts` imports `isShapeInteger`/`parseShapeInteger` from `browse-url.ts` (D-10) rather than redefining them, applying the same shape-only-not-range discipline to `depth`/`limit`.

**Known-key single-value reader — DO NOT COPY for the `file` key** (`web/src/lib/browse-url.ts:81-89`, single-value-drops-repeats reader, verbatim, shown here specifically so the planner sees what NOT to copy):
```ts
return {
	symbol: params.get('symbol') ?? undefined,
	file: params.get('file') ?? undefined,
	...
```
`params.get('file')` silently keeps only the first occurrence. Per D-11, `workbench-url.ts`'s reader for its repeated `file=` key MUST use `params.getAll('file')` instead — this is a deliberate, required divergence from the analog, not an oversight if the planner sees the two differ.

**Unknown-parameter ordered-multimap preservation pattern** (`web/src/lib/browse-url.ts:70-77`, verbatim) — reuse the same technique (an ordered array of pairs, never a plain object) if `workbench-url.ts` needs to preserve unrecognized params:
```ts
export function parseBrowseParams(params: URLSearchParams): BrowseParams {
	const unknown: Array<[string, string]> = [];
	for (const [key, value] of params.entries()) {
		if (!KNOWN_KEYS.has(key)) {
			unknown.push([key, value]);
		}
	}
	...
```

---

### `web/src/lib/components/workbench/DataTable.svelte` (D-06)

**Analog:** `web/src/lib/components/browse/NeighborsPanel.svelte` — a results-list component taking typed row data + a navigate callback as props, no self-navigation.

**Props shape and click-to-navigate delegation pattern** (`web/src/lib/components/browse/NeighborsPanel.svelte:16-52`, verbatim):
```svelte
<script lang="ts">
	import type { Node, Location } from '$lib/gen/ui_pb';
	import type { BlastRadiusState } from '$lib/browse-state';
	import { NAV_INTENT, type BrowseNavDelta, type NavIntent } from '$lib/browse-nav';
	import { isShapeInteger } from '$lib/browse-url';

	let {
		calls,
		calledBy,
		blastRadius,
		depth,
		onNavigate
	}: {
		calls: Node[];
		calledBy: Node[];
		blastRadius: BlastRadiusState;
		depth?: number;
		onNavigate: (delta: BrowseNavDelta, intent: NavIntent) => void;
	} = $props();

	function entryKey(entry: Node | Location): string {
		return `${entry.filePath}:${entry.startLine}:${entry.name}`;
	}

	function openEntry(entry: Node | Location): void {
		onNavigate(
			{ symbol: entry.name, file: entry.filePath, line: entry.startLine },
			NAV_INTENT.NAVIGATE
		);
	}
```
`DataTable.svelte` should follow this shape: rows and columns as typed props, an `onNavigate`/click-through callback rather than internal navigation, and the same `${filePath}:${startLine}:${name}` composite-key convention (`entryKey`) for stable row identity — TanStack's `getRowId` option is the place to plug this same key.

**shadcn-svelte table + TanStack composition** (source: `github.com/TanStack/table` main branch `examples/svelte/lib-shadcn/src/App.svelte`, quoted in RESEARCH Pattern 3 — no in-tree analog exists, this is the external reference the planner should use verbatim rather than inventing a variant):
```svelte
<Table.Root>
  <Table.Header>
    {#each table.getHeaderGroups() as headerGroup (headerGroup.id)}
      <Table.Row>
        {#each headerGroup.headers as header (header.id)}
          <Table.Head colspan={header.colSpan}>
            {#if !header.isPlaceholder}
              {#if header.column.getCanSort()}
                <Button variant="ghost" size="sm" onclick={header.column.getToggleSortingHandler()}>
                  <FlexRender {header} />
                </Button>
              {:else}
                <FlexRender {header} />
              {/if}
            {/if}
          </Table.Head>
        {/each}
      </Table.Row>
    {/each}
  </Table.Header>
  <Table.Body>
    {#each table.getRowModel().rows as row (row.id)}
      <Table.Row>
        {#each row.getAllCells() as cell (cell.id)}
          <Table.Cell><FlexRender {cell} /></Table.Cell>
        {/each}
      </Table.Row>
    {:else}
      <Table.Row><Table.Cell colspan={columns.length} class="h-24 text-center">No results.</Table.Cell></Table.Row>
    {/each}
  </Table.Body>
</Table.Root>
```

---

### `web/src/lib/components/workbench/*-columns.ts` (4 files, D-06)

**No in-tree analog.** These are the first `ColumnDef<Location>[]` arrays in this codebase. Use `Location`'s wire shape (`ui.proto:100-105`: `name`, `kind`, `file_path`, `start_line`) as the column set for all four — `impact-columns.ts`, `affected-columns.ts`, `callers-columns.ts`, `callees-columns.ts` are near-identical `ColumnDef[]` literals over the same row type, differing only in which analysis-specific header summary (node/edge counts for Impact, echoed files for Affected) renders above the table, per D-06.

---

### `web/src/lib/components/workbench/FilePicker.svelte` (D-15)

**Analog:** `web/src/lib/search.ts`'s controller pattern (debounce + AbortController + monotonic request-identity) and `web/src/lib/components/browse/SearchPanel.svelte`'s `Command` usage.

**Controller contract to reuse, not reimplement** (`web/src/lib/search.ts:1-46`, verbatim, doc comment + constants):
```ts
// D-16: ~150ms debounce (the low end of the general 150-300ms guidance —
// this server is on loopback, not crossing a network) and a 2-character
// minimum (symbol search is a names case). Cancellation is independent of
// the delay and non-negotiable: debouncing alone does not fix out-of-order
// responses, since a slow response to a shorter prefix can still land
// after the response to a longer one. Every dispatch gets its own
// AbortController passed into the Connect call's own `signal` option, the
// previous one is aborted before a new one starts, AND a monotonically
// increasing request identity backs that up...
export const SEARCH_DEBOUNCE_MS = 150;
export const SEARCH_MIN_CHARS = 2;
```
`FilePicker.svelte` should call the existing `Files` RPC (once D-14's glob fix lands) through this same debounce/cancel/identity-guard shape — reuse `search.ts`'s controller function(s) directly against `Files` rather than writing a second debounce implementation. `rpc-errors.ts`'s `classifyRpcError` is the one error classifier — reused here too, per `search.ts`'s own comment ("Rejections are mapped through the ONE shared classifier (D-04, rpc-errors.ts) — no second error vocabulary here").

---

### `web/src/routes/health/+page.svelte` (filled, D-01..D-04)

**Analog:** `web/src/lib/components/StatusBanner.svelte` for the verdict-rendering idiom, `web/src/lib/status.ts` for the verdict source (`classifyStatus`, reused unchanged per D-04).

**Verdict-branch rendering idiom** (`web/src/lib/components/StatusBanner.svelte`, verbatim):
```svelte
<script lang="ts">
	import type { IndexStatus } from '$lib/status';
	let { status }: { status: IndexStatus } = $props();
</script>

{#if status.verdict === 'stale'}
	<div class="border-b border-amber-300 bg-amber-50 px-4 py-2 text-sm text-amber-900" role="status" data-testid="status-banner-stale">
		The index is stale — it may not reflect recent changes. Run <code>codegraph index</code> to refresh it.
	</div>
{:else if status.verdict === 'no-index'}
	...
{:else if status.verdict === 'indexing'}
	...
{/if}
```
The health page's own trust-verdict banner (HLT-02, "above the raw numbers") should render `classifyStatus`'s five-member `StatusVerdict` using this exact `{#if}/{:else if}` branch-per-verdict idiom and `role="status"`/`data-testid` convention — do not build a second verdict function (D-04 forbids it explicitly).

**`classifyStatus` — reused unchanged, not reimplemented** (`web/src/lib/status.ts:45-62`, verbatim):
```ts
export function classifyStatus(response: GetStatusResponse): IndexStatus {
	const commit: CommitKnowledge = response.commitSha ? 'known' : 'unknown';

	if (response.initialized) {
		return { verdict: response.stale ? 'stale' : 'ok', commit };
	}
	if (!response.storeExists) {
		return { verdict: 'no-index', commit };
	}
	if (response.indexingInProgress) {
		return { verdict: 'indexing', commit };
	}
	return { verdict: 'unknown', commit };
}
```

---

## Shared Patterns

### Handler wrapping: `withEngine`

**Source:** `internal/uiserver/handlers.go:50` (declaration), used by every handler except `GetStatus`.
**Apply to:** `GetHealth` handler.
```go
func withEngine(ctx context.Context, repoPath string, fn func(*query.Engine) error) error {
```
One open, one deferred close, one error-translation site via `mapEngineError`. `GetHealth` must use this — it is not `GetStatus`'s documented, single, deliberate exception.

### Location row mapper convention

**Source:** `internal/uiserver/handlers.go:235-253` (`locationToProto`/`locationsToProto`).
**Apply to:** No new code needed here — `Callers`/`Callees`/`Impact`/`Affected` already emit `[]*uiv1.Location` through this mapper; the four `*-columns.ts` files on the frontend are the consumers of this shape, not new producers.

### Error classification

**Source:** `web/src/lib/rpc-errors.ts` (`classifyRpcError`), explicitly designated by Phase 3 for reuse by Phases 4/5/6, reused already by `search.ts`.
**Apply to:** `FilePicker.svelte`, and the workbench tabs' per-analysis RPC error states (WRK-04 criterion 3 — composition is Claude's discretion, but the classifier itself is not reinvented).

### Drift-guard shape: regenerate-and-byte-compare (proto:drift), NOT hash-and-compare (web:drift)

**Source:** `Taskfile.yml:279-390` (`proto:drift` full task).
**Apply to:** `task web:components:drift` (D-16).

Report-count-before-comparing precondition, matching rule `84d1gfpywd` (`Taskfile.yml:339-348`, verbatim):
```bash
files=$(git ls-files -- 'internal/schema/*.pb.go' 'internal/uiproto/uiv1/*.pb.go' 'internal/uiproto/uiv1/uiv1connect/*.connect.go' 'web/src/lib/gen/*.ts')
nfiles=0
if [ -n "${files}" ]; then
  nfiles=$(printf '%s\n' "${files}" | wc -l | tr -d ' ')
fi
echo "proto:drift: compared ${nfiles} generated files"
if [ "${nfiles}" -lt 4 ]; then
  echo "::error::proto:drift: enumerated only ${nfiles} committed generated files ..."
  exit 1
fi
```
Regenerate-into-scratch-then-`cmp -s`-byte-compare shape (`Taskfile.yml:361-386`, structure to mirror):
```bash
mkdir -p "${scratch}/bin" "${scratch}/gen"
# ... build/install the pinned tool into scratch/bin ...
# ... regenerate into ${scratch}/gen, never in place ...
drift_found=0
while IFS= read -r f; do
  [ -z "${f}" ] && continue
  fresh="${scratch}/gen/${f}"
  if [ ! -f "${fresh}" ]; then
    echo "::error::...: regeneration produced no output for ${f} at ${fresh}"
    drift_found=1
    continue
  fi
  if ! cmp -s "${f}" "${fresh}"; then
    echo "::error::...: ${f} differs from the pinned toolchain's regeneration"
    drift_found=1
  fi
done <<< "${files}"
if [ "${drift_found}" -ne 0 ]; then
  exit 1
fi
```
`web:components:drift` should: (1) count and report committed files under `web/src/lib/components/ui/{table,tabs}/` before comparing (floor = 14: 9 table + 5 tabs), (2) re-run `pnpm dlx shadcn-svelte@1.5.1 add table tabs -y -o` (pinned version, not `@latest`) into a scratch directory, (3) byte-compare (`cmp -s`) each committed file against its scratch-regenerated counterpart. Do **not** follow `web:drift`'s (`Taskfile.yml:1074-1181`) hash-and-compare-against-a-committed-manifest shape — components have no `.build-manifest`-equivalent marker file to compare against; RESEARCH's Code Examples section states this explicitly.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `web/src/lib/components/ui/table/*.svelte` (9 files) | component | render | Vendored via `shadcn-svelte add table` — first table primitive in this repo; use the external reference in RESEARCH Pattern 3 (`github.com/TanStack/table` `examples/svelte/lib-shadcn/`), not an in-tree analog |
| `web/src/lib/components/ui/tabs/*.svelte` (5 files) | component | render | Vendored via `shadcn-svelte add tabs` — first tabs primitive in this repo (D-13 notes it's "a Bits UI primitive from the same family already vendored in 03-06," i.e. same *toolchain*, not the same *component*) |
| `web/src/lib/components/workbench/*-columns.ts` | config | transform | First `ColumnDef<Location>[]` literals in this codebase — no prior TanStack column-definition file exists to copy from; derive directly from `Location`'s proto fields (`ui.proto:100-105`) |

## Metadata

**Analog search scope:** `internal/uiserver/`, `internal/query/`, `internal/gitmeta/`, `internal/uiproto/uiv1/`, `web/src/lib/`, `web/src/lib/components/`, `web/src/routes/`, `web/tests/`, `Taskfile.yml`
**Files scanned:** ~20 read this session (handlers.go, readonly_test.go, status.go, validate.go, files.go, detect.go, ui.proto excerpts, browse-url.ts, status.ts, search.ts, NeighborsPanel.svelte, StatusBanner.svelte, browse-url.test.ts, status.test.ts, Taskfile.yml's proto:drift/web:drift targets)
**Pattern extraction date:** 2026-08-29
