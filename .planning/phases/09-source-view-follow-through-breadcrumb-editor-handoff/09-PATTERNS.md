# Phase 9: Source View Follow-Through — Pattern Map

**Mapped:** 2026-09-12
**Files analyzed:** 12
**Analogs found:** 12 / 12

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/uiproto/uiv1/ui.proto` (+ `GetEditorLink` rpc/messages/enum) | proto/config | request-response | same file — `GetPermalinkRequest`/`GetPermalinkResponse`/`PermalinkAvailability` (lines 599-663) | exact |
| `internal/uiserver/editorlink.go` (new) | controller/service (RPC handler) | request-response | `internal/uiserver/permalink.go` (full file, 288 lines) | exact |
| `internal/uiserver/editorlink_test.go` (new) | test | request-response | `internal/uiserver/permalink_test.go::TestGetPermalinkPathConfinementAtRPCBoundary` (lines 432-472) | exact |
| `internal/uiserver/readonly_test.go` (modify) | test/fixture | — | same file — `wantUIServiceMethods` (lines 15-77), `TestUIServiceMethodSetIsExactlyTheReadSet` (79-101), `mutatingVerbs` (128-134) | exact |
| `internal/uiserver/server.go` (`Options` +field(s)) | config/model | — | same file — `Options` struct (lines 61-73) | exact |
| `internal/cli/ui.go` (+`--editor-url`/`--no-editor-url` flags) | CLI/controller | request-response (startup config) | same file — `newUiCmd`, `shouldOpenBrowser` (full 113-line file) | exact |
| `internal/cli/editordiscovery.go` (new) | utility | file-I/O (stat/LookPath probe) | no direct in-repo analog for probing; closest shape is `shouldOpenBrowser`'s "compute a policy decision once, at startup, from environment facts" pattern in `internal/cli/ui.go` (95-113) | role-match |
| `web/src/lib/components/browse/SourcePane.svelte` (modify: per-line DOM, breadcrumb, header link) | component | request-response + client-derived state | itself (full file, 254 lines) — `PermalinkClient` interface (20-30), `$effect`-driven fetch (109-142), `permalinkSurface` snippet (145-167) | exact (self) |
| `web/src/routes/graph/+page.svelte` (fetch/cache pattern reference only) | route/provider | CRUD-ish fetch-cache | same file — `fileSymbolsCache`/`pendingFileSymbols` (lines 59-82, 460-495) | exact |
| `web/src/lib/editor-prefs.ts` (new) | utility/store | event-driven (localStorage read/write) | no existing localStorage-backed pref in `web/src/lib/` — none found (`rg -n "localStorage" web/src` = 0 hits) | no analog |
| `web/tests/*.test.ts` (new: breadcrumb + editor-link unit tests) | test | request-response / DOM-derived | `web/tests/source-pane.test.ts` (full file, 293 lines) — fixture builders `node()`/`fileState()`/`sourceRender()` (14-56) | exact |
| `web/scripts/breadcrumb-check.mjs` (new) | test (live-browser gate) | event-driven (browser automation) | `web/scripts/graph-collapse-affordance-check.mjs` (full file, 332 lines) | exact |

## Pattern Assignments

### `internal/uiproto/uiv1/ui.proto` (+`GetEditorLink`)

**Analog:** same file, `GetPermalinkRequest`/`Response`/`PermalinkAvailability` (`internal/uiproto/uiv1/ui.proto:599-663`)

**Rpc registration** (`ui.proto:50-67`):
```protobuf
service UIService {
  rpc GetStatus(GetStatusRequest) returns (GetStatusResponse);
  ...
  rpc GetPermalink(GetPermalinkRequest) returns (GetPermalinkResponse);
```
`GetEditorLink` is added as the 15th line in this block, additive only (never renumber existing rpcs).

**Request message shape to mirror** (`ui.proto:599-614`):
```protobuf
message GetPermalinkRequest {
  string path = 1;
  optional int32 line = 2;
  optional int32 end_line = 3;
}
```
`GetEditorLinkRequest` mirrors this (`path`, optional `line`, optional `col`, plus D-06's optional `template` override field) — same `optional` proto3-presence discipline so "unset" is distinguishable from "0".

**Closed enum shape to mirror** (`ui.proto:616-644`):
```protobuf
enum PermalinkAvailability {
  PERMALINK_AVAILABILITY_UNSPECIFIED = 0;
  PERMALINK_AVAILABILITY_LINKABLE = 1;
  PERMALINK_AVAILABILITY_LINKABLE_UNVERIFIED = 2;
  PERMALINK_AVAILABILITY_NO_LINK = 3;
}
```
`EditorLinkAvailability` follows the same shape: `_UNSPECIFIED = 0` (required zero value, never produced), then `BUILDABLE`, `NO_TEMPLATE`, `TEMPLATE_INVALID` (D-07) — same "closed set, additive-only" comment discipline.

**Response message shape to mirror** (`ui.proto:646-663`):
```protobuf
message GetPermalinkResponse {
  string url = 1;
  PermalinkAvailability availability = 2;
  string reason = 3;
}
```
`GetEditorLinkResponse` mirrors `url`/`availability`/`reason`, plus D-07's provenance field (new, e.g. `string template_source = 4;`).

---

### `internal/uiserver/editorlink.go` (new)

**Analog:** `internal/uiserver/permalink.go` (full file)

**File-header discipline to copy** (`permalink.go:1-13`):
```go
// permalink.go implements GetPermalink (D-06/D-07/D-08/D-09): ...
// Everything this handler produces from git introspection is an ANSWER,
// never an error: ... The one case that IS an error is an invalid path
// argument, handled by (*query.Engine).ValidateRepoRelativePath (SRV-05)
// below.
package uiserver
```

**Reason-string constants — distinct causes never share a string** (`permalink.go:33-52`):
```go
const noCommitSHAReason = "this index has no recorded commit SHA (a pre-upgrade graph) — re-index to enable permalinks"
const malformedCommitSHAReason = "..."
const (
	notObservedReason  = "this commit is not observed on any remote-tracking branch; ..."
	checkUnknownReason = "could not verify whether this commit is on a remote-tracking branch (...); the link may 404"
)
```
`editorlink.go` needs equivalent distinct reason constants per D-07/D-13/D-16 cause: e.g. `noTemplateConfiguredReason`, `disabledByOperatorReason` ("disabled by operator" — D-16 verbatim), `templateInvalidSchemeReason`, `templateMissingPathPlaceholderReason`.

**Core validation + confinement + classifiedErr pattern** (`permalink.go:118-186`, request validation lines ~100-112 not shown here but present just above `GetPermalink`'s body):
```go
var classifiedErr *connect.Error
err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
    path := req.Msg.GetPath()
    if verr := eng.ValidateRepoRelativePath(path); verr != nil {
        if errors.Is(verr, fs.ErrNotExist) {
            classifiedErr = connect.NewError(connect.CodeInvalidArgument, fmt.Errorf(
                "path %q does not exist in the working tree — ...", path))
            return nil
        }
        return verr
    }
    // ... build the answer; configuration states are ALWAYS a successful response ...
    return nil
})
if err != nil {
    return nil, err
}
if classifiedErr != nil {
    return nil, classifiedErr
}
return connect.NewResponse(resp), nil
```
`GetEditorLink` follows this exact shape: `ValidateRepoRelativePath` first (SRV-05), never leak the absolute host path in the error message — name only the caller's own repo-relative path (per D-07/`permalink.go` reclassification discipline). Note: `editorlink.go` has NO git dependency (D-05), so `withEngine`'s closure is simpler — no `gitmeta`/`schema.IndexedCommitSHA` calls, no `IndexMeta()`.

**Line validation to reuse verbatim** (`permalink.go:100-111`, read this session):
```go
if line := req.Msg.Line; line != nil && *line < 1 {
    return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("line %d must be >= 1", *line))
}
```
`GetEditorLink` applies the same `line >= 1` check (and `col >= 1` if `col` is present) before calling `withEngine`.

**Percent-encoding helper to reuse (not reinvent)** (`permalink.go:275-288`):
```go
func percentEncodeRepoPath(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}
```
D-08 requires `{path}` to be percent-encoded per placeholder position; reuse `url.PathEscape`, and reuse this exact per-segment splitting approach if `{path}` is filled with a path-shaped placeholder (not a single opaque string) — this is `permalink.go`'s IN-03 fix already-solved.

**Closed-enum switch-with-safe-default pattern** (`permalink.go:210-238`):
```go
func remotePresenceResponse(blobURL string, presence gitmeta.RemotePresence) *uiv1.GetPermalinkResponse {
	switch presence {
	case gitmeta.RemotePresenceObserved:
		return &uiv1.GetPermalinkResponse{Url: blobURL, Availability: uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_LINKABLE}
	case gitmeta.RemotePresenceNotObserved:
		return &uiv1.GetPermalinkResponse{Url: blobURL, Availability: ..._LINKABLE_UNVERIFIED, Reason: notObservedReason}
	default: // any future member — degrade to the SAFE/uncertain direction, never the positive claim
		return &uiv1.GetPermalinkResponse{Url: blobURL, Availability: ..._LINKABLE_UNVERIFIED, Reason: checkUnknownReason}
	}
}
```
Extract an equivalent small helper in `editorlink.go` (e.g. `templateAvailabilityResponse`) so the "unknown/future value degrades to the safe answer" behavior is independently unit-testable, mirroring this exact extraction rationale.

---

### `internal/uiserver/editorlink_test.go` (new)

**Analog:** `internal/uiserver/permalink_test.go::TestGetPermalinkPathConfinementAtRPCBoundary` (lines 432-472)

**Table-driven confinement-at-RPC-boundary shape to copy verbatim structure:**
```go
func TestGetPermalinkPathConfinementAtRPCBoundary(t *testing.T) {
	...
	srv := startedServer(t, dir)
	client := uiv1connect.NewUIServiceClient(http.DefaultClient, srv.URL())

	t.Run("in-repo control", func(t *testing.T) {
		resp, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{Path: "pkga/pkga.go"}))
		if err != nil { t.Fatalf(...) }
		if resp.Msg.GetUrl() == "" { t.Fatalf(...) }
	})

	cases := []struct{ name, path, wantFrag string }{
		{name: "escape", path: "../outside.txt", wantFrag: "escapes the repo root"},
		{name: "absolute", path: "/etc/passwd"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := client.GetPermalink(context.Background(), connect.NewRequest(&uiv1.GetPermalinkRequest{Path: c.path}))
			if err == nil { t.Fatalf("... succeeded, want a refusal") }
			if code := connect.CodeOf(err); code != connect.CodeInvalidArgument { t.Fatalf(...) }
			if c.wantFrag != "" && !strings.Contains(err.Error(), c.wantFrag) { t.Fatalf(...) }
		})
	}
}
```
`TestGetEditorLinkPathConfinementAtRPCBoundary` copies this exact positive-control-first + table shape (BRW-11's success criterion 2). Add scheme-allowlist cases (`TEMPLATE_INVALID` for `javascript:`, `data:`) and reason-string-distinctness assertions as additional subtests or a second test function, following `readonly_test.go`'s "positive count before applying the check" (rule `84d1gfpywd`) discipline for any allowlist-membership test.

---

### `internal/uiserver/readonly_test.go` (modify)

**Analog:** same file — `wantUIServiceMethods` fixture (lines 15-77) and its update history comments

**Fixture update pattern (exact precedent for this phase's edit):**
```go
// UPDATED at plan 06-01: WatchGraph (RPC-04) is the fourteenth read-only
// rpc, added additively to internal/uiproto/uiv1/ui.proto's `service
// UIService` block. ...
var wantUIServiceMethods = map[string]struct{}{
	"GetStatus": {}, "Search": {}, "Files": {}, "Callers": {}, "Callees": {},
	"Impact": {}, "Affected": {}, "GetNodeDetail": {}, "Explore": {},
	"GetPermalink": {}, "GetHealth": {}, "FileGraph": {}, "FileSymbols": {},
	"WatchGraph": {},
}
```
Add a new `// UPDATED at plan 09-XX:` comment block naming `GetEditorLink` as the 15th read-only rpc, add `"GetEditorLink": {}` to the map, and update every hardcoded literal count (`14` → `15`) at both `if len(got) != 14` (~line 108) and the doc-comment prose above `wantUIServiceMethods` and above `TestUIServiceMethodSetIsExactlyTheReadSet`. `mutatingVerbs` (lines 128-134) needs no change — `GetEditorLink` contains no verb from that list — but `TestUIServiceDeclaresNoMutatingMethod` will re-run against it automatically.

---

### `internal/uiserver/server.go` (`Options` +field)

**Analog:** same file, `Options` struct (lines 61-73)

```go
type Options struct {
	RepoPath string
	// Addr is the bind address net.Listen receives, defaulting to
	// DefaultAddr when empty. Nothing outside this package writes this
	// field in v1 — no flag, no environment variable, no configuration
	// file reads it (D-08). ...
	Addr string
}
```
Add the resolved editor template / disabled-state field(s) here (e.g. `EditorURLTemplate string`, `EditorURLDisabled bool` or a small resolved struct), following the exact doc-comment convention of naming which flag/env writes the field and citing the decision ID, mirroring `Addr`'s comment style.

---

### `internal/cli/ui.go` (+ flags)

**Analog:** same file (full 113-line file) — `newUiCmd`, `shouldOpenBrowser`

**Existing boolean-flag + fail-fast precedent to extend:**
```go
var openBrowser = browser.OpenURL
...
func newUiCmd() *cobra.Command {
	var path string
	var noOpen bool
	cmd := &cobra.Command{
		...
		RunE: func(cmd *cobra.Command, args []string) error {
			start, err := resolveStartPath(path)
			if err != nil { return err }
			srv, err := uiserver.Listen(uiserver.Options{RepoPath: start})
			if err != nil { return err }
			...
		},
	}
	cmd.Flags().StringVarP(&path, "path", "p", "", "repo path (default: cwd)")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "do not open a browser automatically")
	return cmd
}
```
`--editor-url`/`--no-editor-url` are added as new `cmd.Flags()` calls in this same block; D-17's fail-fast-on-malformed-value happens BEFORE `uiserver.Listen(...)` is called — i.e., resolve+validate the flag/env/discovery value first (call the shared validator editorlink.go will also use), return an error from `RunE` immediately (same pattern as `resolveStartPath`'s `if err != nil { return err }`) rather than letting a bad value flow into `Options` silently.

**Policy-decision-from-environment-facts precedent (`shouldOpenBrowser`, lines 95-113):**
```go
func shouldOpenBrowser(noOpen bool, cmd *cobra.Command) bool {
	if noOpen { return false }
	if os.Getenv("CI") != "" { return false }
	f, ok := cmd.OutOrStdout().(*os.File)
	if !ok { return false }
	fi, err := f.Stat()
	if err != nil { return false }
	return fi.Mode()&os.ModeCharDevice != 0
}
```
A `resolveEditorURLTemplate(flagValue string, noEditorURL bool) (string, error)`-shaped function (in `editordiscovery.go` or `ui.go`) should follow this exact "several independent short-circuit checks, each returning early" shape for the flag → env → discovery → unconfigured precedence (D-14), with `os.Getenv("CODEGRAPH_EDITOR_URL")` and `os.Getenv("CODEGRAPH_NO_EDITOR_URL")` read the same way `os.Getenv("CI")` is read here — no viper, no new config layer.

---

### `internal/cli/editordiscovery.go` (new)

**Analog:** no direct probing precedent in-repo; role-matched to `shouldOpenBrowser`'s "compute once from environment, never re-probe" shape (`internal/cli/ui.go:95-113`, see above). No existing `exec.LookPath` usage pattern was found in `internal/cli/` — this is genuinely new capability; follow stdlib `exec.LookPath` directly, no wrapper library, consistent with the repo's "no new dependency needed" stance in RESEARCH.md's Standard Stack section.

---

### `web/src/lib/components/browse/SourcePane.svelte` (modify)

**Analog:** itself — full file (254 lines)

**Client-prop interface pattern to mirror for the editor-link client** (lines 20-30):
```ts
interface PermalinkClient {
	getPermalink(
		request: MessageInitShape<typeof GetPermalinkRequestSchema>,
		options?: { signal?: AbortSignal }
	): Promise<GetPermalinkResponse>;
}
```
Add `EditorLinkClient` with the same shape (`getEditorLink(request, options?)`), passed as an optional prop exactly like `client?: PermalinkClient` (line ~40), never imported as a module-level singleton — keeps the pane stub-testable.

**`$effect`-driven fetch-on-target-change pattern to mirror** (lines 109-142):
```ts
let permalinkState = $state<PermalinkState>({ kind: 'idle' });
$effect(() => {
	const params = permalinkParamsFor(target);
	if (!params || !client) {
		permalinkState = { kind: 'idle' };
		return;
	}
	permalinkState = { kind: 'loading' };
	const controller = new AbortController();
	client
		.getPermalink({ path: params.path, line: params.line, endLine: params.endLine }, { signal: controller.signal })
		.then((response) => { permalinkState = { kind: 'loaded', response }; })
		.catch(() => { permalinkState = { kind: 'idle' }; });
	return () => controller.abort();
});
```
The D-12 "probe once per opened target" `GetEditorLink` call follows this exact shape: a `editorLinkState = $state<...>({kind:'idle'})`, an `$effect` keyed on `target`, `AbortController` cleanup, and a `.catch()` that degrades to idle rather than throwing — never surfacing an rpc failure as the whole pane's failure.

**Snippet-based conditional render pattern to mirror** (lines 145-167, `permalinkSurface`):
```svelte
{#snippet permalinkSurface()}
	{#if permalinkState.kind === 'loaded'}
		{@const p = permalinkState.response}
		{#if p.availability === PermalinkAvailability.LINKABLE}
			<a href={p.url} ... data-testid="permalink-linkable">View on GitHub</a>
		{:else if ...}
		{/if}
	{/if}
{/snippet}
```
The header "Open in editor" link (D-09) is a new `{#snippet editorLinkSurface()}` following this identical `{#if kind==='loaded'}{@const}{#if availability === X}` cascade, rendered inside the SAME header row div as `{@render permalinkSurface()}` (lines 208-211, 226-230).

**Existing single-blob render block that MUST become per-line DOM** (lines 217-221, 244-249) — this is the one change both BRW-10 and BRW-11/12 share (Pitfall 3):
```svelte
<pre class="overflow-x-auto rounded border p-4 text-sm"><code>{@html highlightSource(
		text,
		language
	)}</code></pre>
```
Plan this as a single task: wrap `highlightSource`'s output per-line (e.g. split rendered HTML by `\n` into `<span data-line={n}>` rows, or a two-column `<div class="grid grid-cols-[auto_1fr]">` gutter+code grid — Claude's discretion per CONTEXT.md) — both the `file` branch and the `single-def` branch need the identical restructuring, not two independent edits.

---

### `web/src/routes/graph/+page.svelte` (fetch/cache pattern — reference only, not modified this phase unless breadcrumb needs `FileSymbols` fetched from `SourcePane` itself)

**Analog:** same file (lines 59-82, 460-495)

```ts
let fileSymbolsCache = $state<Map<string, FileSymbolsResponse>>(new Map());
let pendingFileSymbols = new Set<string>();
// on demand:
pendingFileSymbols.add(id);
uiClient
	.fileSymbols({ path: id })
	.then((response) => {
		pendingFileSymbols.delete(id);
		const cache = new Map(fileSymbolsCache);
		cache.set(id, response);
		fileSymbolsCache = cache; // NEW Map instance — Svelte 5 reactivity requires this
	});
```
`SourcePane`'s breadcrumb needs this exact pattern reproduced locally (not imported — it's route-scoped state in `+page.svelte`, and `SourcePane` is a component that may need its own `fileSymbolsCache`/`pendingFileSymbols` pair, or receives `FileSymbols` via a prop/client call routed the same way `PermalinkClient` is). The load-bearing detail: always reassign to a NEW `Map` (never mutate in place) for Svelte 5 reactivity to fire.

---

### `web/src/lib/editor-prefs.ts` (new)

**Analog:** none found — `rg -n "localStorage" web/src` returns zero hits. This is genuinely new capability in this codebase.

**Guidance (no in-repo precedent to copy; use general Svelte 5 + try/catch discipline already established elsewhere in this file tree):**
- Follow `SourcePane.svelte`'s "never throw for an optional/degraded integration" discipline (its `.catch()` blocks degrade to `idle` rather than propagating) — wrap every `localStorage.getItem`/`setItem` call in try/catch and fail silently to "no override" (D-18 says "render correctly without it").
- Follow the graph page's "never mutate in place, reassign" reactivity discipline if the override is held in `$state`.
- No existing store/module in `web/src/lib/` for reference; this file has no in-repo analog and should be written directly against D-18's spec (three presets, custom template, "use server default", provenance display).

---

### `web/tests/*.test.ts` (new breadcrumb + editor-link unit tests)

**Analog:** `web/tests/source-pane.test.ts` (full file, 293 lines)

**Fixture-builder pattern to copy verbatim structure** (lines 14-56):
```ts
function node(name: string, overrides: Partial<Node> = {}): Node {
	return { id: name, kind: 'func', name, qualifiedName: `pkg.${name}`, filePath: `${name}.go`,
		language: 'go', startLine: 10, endLine: 20, startCol: 0, endCol: 0, signature: '',
		docstring: '', visibility: '', isExported: true, returnType: '', ...overrides
	} as unknown as Node;
}
function fileState(overrides: Partial<BrowseTargetState & { kind: 'file' }> = {}): BrowseTargetState {
	return { kind: 'file', path: 'internal/query/node.go', source: new TextEncoder().encode('package query\n'),
		truncated: false, totalLines: 1, returnedLines: 1, ...overrides } as BrowseTargetState;
}
```
New breadcrumb/gutter tests build equivalent `fileSymbolsResponse(overrides)` and multi-symbol fixtures the same way — small factory functions with sane defaults, `...overrides` spread last. Import style: `import { render, screen, waitFor, fireEvent } from '@testing-library/svelte';` (line 4) and render `SourcePane` directly with stub clients, matching `PermalinkClient`'s stub-testability goal.

---

### `web/scripts/breadcrumb-check.mjs` (new)

**Analog:** `web/scripts/graph-collapse-affordance-check.mjs` (full file, 332 lines)

**Structural shape to copy exactly:**
```js
#!/usr/bin/env node
import { chromium } from '@playwright/test';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as url from 'node:url';

function repoRoot() { /* path.resolve(here, '..', '..') */ }

async function pollUntil(pg, check, timeoutMs, pollMs) {
	const deadlineAt = Date.now() + timeoutMs;
	for (;;) {
		const value = await check();
		if (value) return value;
		if (Date.now() >= deadlineAt) throw new Error(`pollUntil: condition did not become true within ${timeoutMs}ms`);
		await pg.waitForTimeout(pollMs);
	}
}

async function main() {
	// launch real chromium, navigate to the built `codegraph ui --no-open` URL,
	// drive REAL mouse/scroll input, poll a published DOM data-testid or
	// window.__codegraph* global, assert, write a diagnostic JSON record,
	// throw (non-zero exit) on failure.
}

const isMain = process.argv[1] && url.pathToFileURL(process.argv[1]).href === import.meta.url;
if (isMain) {
	main().catch((err) => {
		console.error(`breadcrumb-check: fatal — ${err instanceof Error ? err.message : String(err)}`);
		if (err instanceof Error && err.stack) console.error(err.stack);
		process.exit(1);
	});
}
```
**Diagnostic-record + exit-code convention to copy** (lines 288-317 of the analog):
```js
const record = { success: ..., /* named fields for each phase's outcome */, pageErrorCount: pageErrors.length, pageErrors, browserIdentity: { name: 'chromium', version: browserVersion } };
await fs.promises.mkdir(path.dirname(outPath), { recursive: true });
fs.writeFileSync(outPath, JSON.stringify(record, null, 2) + '\n');
process.stdout.write(`...: wrote ${outPath} — success=${record.success}\n`);
if (!record.success) { throw new Error(`...: FAILED — ...`); }
```
For BRW-10 criterion 1: scroll to a known line via `page.mouse.wheel` or `element.scrollIntoView` + `pollUntil`-style settle (never a bare `sleep`), assert the sticky breadcrumb's `data-testid` element's `textContent` equals the expected symbol name, and — per the phase's RED-demonstration requirement — the SAME script is run once against a `git stash`'d pre-fix checkout to capture the stale/empty failure for `09-MUTATION-LOG.md`.

## Shared Patterns

### Answers-not-errors for configuration state
**Source:** `internal/uiserver/permalink.go` (file header comment, lines 1-13; `remotePresenceResponse`, lines 210-238)
**Apply to:** `editorlink.go` — `NO_TEMPLATE`/`TEMPLATE_INVALID` are successful responses; only a rejected caller path is a Connect error.

### Distinct reason strings per cause, never shared
**Source:** `internal/uiserver/permalink.go:33-52` (`noCommitSHAReason`, `malformedCommitSHAReason`, `notObservedReason`, `checkUnknownReason`)
**Apply to:** `editorlink.go`'s reason constants for `NO_TEMPLATE` (unconfigured vs. disabled-by-operator — D-16's exact wording) and `TEMPLATE_INVALID` (bad scheme vs. missing `{path}` vs. unknown placeholder).

### Single confinement gate, never a second implementation
**Source:** `internal/query/node.go:17-94` (`resolveSourcePath`/`ValidateRepoRelativePath`)
**Apply to:** `editorlink.go` calls `eng.ValidateRepoRelativePath(path)` — the exact same call `permalink.go` makes — never a client-side or second server-side path check (Anti-Pattern flagged in RESEARCH.md).

### Positive set-equality / positive-count guards, never vacuous negatives
**Source:** `internal/uiserver/readonly_test.go` — `wantUIServiceMethods` set-equality (15-101) and `mutatingVerbs`'s "assert inspected > 0 before applying the check" (128-160)
**Apply to:** any new allowlist/count-based test in `editorlink_test.go` or the scheme-allowlist unit test (rule `84d1gfpywd`).

### `$effect` + `AbortController` + degrade-to-idle-on-failure for optional RPC integrations
**Source:** `web/src/lib/components/browse/SourcePane.svelte:109-142`
**Apply to:** the new editor-link probe effect and any breadcrumb `FileSymbols` fetch inside `SourcePane`.

### Reassign-to-new-Map/Set for Svelte 5 reactivity, never mutate in place
**Source:** `web/src/routes/graph/+page.svelte:59-82, 466-494`
**Apply to:** any cache (`fileSymbolsCache`-shaped) the breadcrumb or editor-prefs code introduces inside `$state`.

### Startup-time-only environment facts, never re-probed per request
**Source:** `internal/uiserver/server.go` (`Options`, `Listen` doc comments) and `internal/cli/ui.go`'s `shouldOpenBrowser` (95-113)
**Apply to:** `internal/cli/editordiscovery.go` — discovery runs once at CLI startup before `uiserver.Listen`, never inside `GetEditorLink`'s handler.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `web/src/lib/editor-prefs.ts` | utility/store | event-driven (localStorage) | No existing localStorage-backed preference exists anywhere in `web/src/lib/` (`rg -n "localStorage" web/src` = 0 hits) — this is the first such module; plan directly from D-18's spec and the general try/catch + Svelte 5 `$state` reactivity discipline used elsewhere in this codebase. |
| `internal/cli/editordiscovery.go` | utility | file-I/O (PATH/app-dir probe) | No existing `exec.LookPath`/platform-app-dir probing code in `internal/cli/`; role-matched only to `shouldOpenBrowser`'s "compute once from environment" shape, not a structural analog. |

## Metadata

**Analog search scope:** `internal/uiserver/`, `internal/cli/`, `internal/query/`, `internal/uiproto/uiv1/`, `web/src/lib/`, `web/src/routes/graph/`, `web/tests/`, `web/scripts/`
**Files scanned:** `permalink.go`, `permalink_test.go`, `readonly_test.go`, `server.go`, `ui.go`, `node.go`, `ui.proto`, `SourcePane.svelte`, `graph/+page.svelte`, `source-pane.test.ts`, `graph-collapse-affordance-check.mjs`, plus an `rg` sweep for `localStorage` (zero hits) and `editor.url`/`EDITOR_URL` (zero hits, confirming greenfield config surface)
**Pattern extraction date:** 2026-09-12
