# Phase 3: Browse, Inspect & Navigation - Pattern Map

**Mapped:** 2026-08-28
**Files analyzed:** 13 (7 Go, 6 TS/Svelte) + 1 conditional (D-01 source endpoint)
**Analogs found:** 11 / 13 (2 have NO IN-REPO ANALOG — Svelte modules with no prior precedent)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `internal/uiproto/uiv1/ui.proto` (add `GetPermalink` rpc+messages) | config/schema | request-response | same file, existing `GetNodeDetail`/`SourceBlob` messages | exact (additive edit to itself) |
| `internal/uiserver/permalink.go` (new, D-06 handler) | controller/handler | request-response | `internal/uiserver/degrade.go` (one-concern-per-file, doc-comment-with-decision-IDs) | role-match |
| `internal/gitmeta/permalink.go` or extension to `worktree.go` (D-06 remote-URL + pushed-check) | service | event-driven (git subprocess) | `internal/gitmeta/worktree.go` + `internal/indexer/commit.go` | exact |
| `internal/uiserver/confinement_test.go` (new, D-02 RPC-boundary regression test) | test | request-response | `internal/query/errors_test.go` (assertion shape); `internal/indexer/capability/matrix_test.go` (positive-control/no-vacuous-pass discipline) | role-match |
| `internal/uiserver/permalink_test.go` (new) | test | request-response | `internal/uiserver/degrade_test.go`, `internal/uiserver/handlers_test.go` | role-match |
| `web/src/lib/status.ts` (new, D-04 status gate) | store/provider | request-response (fetch-on-load/nav) | `web/src/routes/+page.svelte` (only existing `GetStatus` call site) | role-match (component→module extraction) |
| `web/src/lib/rpc-errors.ts` (new, D-04 error mapper) | utility | transform | NO IN-REPO ANALOG (closest: `internal/uiserver/handlers.go`'s `mapEngineError`, Go-side, cross-language) | no in-repo TS analog |
| `web/src/lib/browse-url.ts` (new, D-13 URL parse/serialize) | utility | transform | NO IN-REPO ANALOG | no in-repo TS analog |
| `web/src/lib/highlight.ts` (new, D-19 highlighter) | utility | transform | NO IN-REPO ANALOG | no in-repo TS analog |
| `web/src/routes/browse/+page.svelte` (fills D-18 placeholder) | route/component | request-response | `web/src/routes/+page.svelte` (the only existing RPC-calling Svelte page) | exact |
| `web/src/lib/components/ui/command/*`, `.../popover/*` (shadcn-svelte-added) | component | request-response | NO IN-REPO ANALOG (`web/src/lib/` today holds only `client.ts`, `gen/`, `utils.ts`, `assets/` per CONTEXT D-22) | no in-repo analog — vendored via CLI |

## Pattern Assignments

### `internal/uiproto/uiv1/ui.proto` (config, additive edit)

**Analog:** the file's own existing service block and message shapes.

**Additive-only discipline** (`internal/uiproto/uiv1/ui.proto:6-8`):
```protobuf
// Additive-only discipline (D-02a, inherited from internal/schema):
// within this package, a field number is never renumbered and never
// reused. When a field is retired, its number is added to a `reserved`
// clause instead of being deleted or repurposed.
```

**Service method registration pattern** (`ui.proto:44-53`):
```protobuf
service UIService {
  rpc GetStatus(GetStatusRequest) returns (GetStatusResponse);
  rpc Search(SearchRequest) returns (SearchResponse);
  ...
  rpc GetNodeDetail(GetNodeDetailRequest) returns (GetNodeDetailResponse);
  rpc Explore(ExploreRequest) returns (ExploreResponse);
}
```
`GetPermalink(GetPermalinkRequest) returns (GetPermalinkResponse)` is added as a tenth line inside this block, following the one-rpc-per-message-pair shape already used by all nine existing methods. Message field numbering starts at 1 for a brand-new message (no `reserved` band collision risk, unlike editing `GetNodeDetailResponse`/`NodeDefinition`/`ExploreGroup` which each carry pre-reserved numbers per the file's header comment, lines 24-30).

---

### `internal/uiserver/permalink.go` (controller/handler, request-response)

**Analog:** `internal/uiserver/degrade.go` (file-scoped single-concern pattern) + `internal/uiserver/handlers.go`'s `GetNodeDetail` (handler shape).

**One-concern-per-file-with-doc-comment-naming-decision-IDs** (`internal/uiserver/degrade.go:1-20`):
```go
package uiserver

import (
	"errors"

	"connectrpc.com/connect"

	"github.com/seanb4t/codegraph-go/internal/graphstore"
	"github.com/seanb4t/codegraph-go/internal/query"
	uiv1 "github.com/seanb4t/codegraph-go/internal/uiproto/uiv1"
)

// degradeKind classifies an openEngine failure into one of the states
// SRV-04's degrade path must render distinctly (D-14, D-16): ...
```
This is CONTEXT.md's own named precedent ("Established Patterns" — "One concern per file with a doc comment naming its decision IDs — `degrade.go`, `truncate.go`, `originguard.go`, `spa.go`. New server code (permalink derivation) should follow it."). `permalink.go` should open with a package-doc-style comment citing D-06/D-07/D-08/D-09 the same way `originguard.go:1-6` cites SRV-02.

**Handler shape to copy** (`internal/uiserver/handlers.go:728-751`, `GetNodeDetail`):
```go
func (s *uiService) GetNodeDetail(ctx context.Context, req *connect.Request[uiv1.GetNodeDetailRequest]) (*connect.Response[uiv1.GetNodeDetailResponse], error) {
	var resp *uiv1.GetNodeDetailResponse
	err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
		...
		r, err := nodeDetailToProto(eng, d)
		if err != nil {
			return err
		}
		resp = r
		return nil
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}
```
> **CORRECTION (2026-08-28, cycle-1 cross-AI review finding L3 — this paragraph is STALE).**
> The sentence below is wrong and `03-05-PLAN.md` Task 3 step (d) is right: `GetPermalink`
> **DOES** need `withEngine`. It needs the Engine for two things settled after this
> paragraph was written — the **indexed** commit SHA read out of the index metadata (the
> whole point of BRW-09 is pinning to the indexed commit, not to `HEAD`), and
> `(*query.Engine).ValidateRepoRelativePath`, the SRV-05 confinement wrapper. Store-degrade
> therefore follows the standard non-`GetStatus` path. Follow the plan, not this paragraph.
> The rest of the paragraph — copy the `connect.Request[...]`/`connect.Response[...]`
> signature shape and the "map to proto, return" structure — still holds, and the
> degrade-never-error guidance in the next paragraph is unaffected and correct.

~~`GetPermalink` will NOT need `withEngine` (it does not touch the graph store — D-06 says the server shells to git via `internal/gitmeta`, not `query.Engine`), so this is a partial-pattern match: copy the `connect.Request[...]`/`connect.Response[...]` signature shape and the "map to proto, return" structure, but the body calls a new `internal/gitmeta` function instead of `withEngine`.~~

**Error handling / degrade-never-error pattern to copy** (`internal/uiserver/degrade.go:105-113`, `errIndexingInProgress`, and `internal/indexer/commit.go:41-49`'s doc comment): D-06/D-07 mandate "unknown → empty/no-link, never an error" — this is the SAME shape `resolveHeadCommitSHA` already uses (see gitmeta section below), not `mapEngineError`'s classify-and-wire-as-Connect-error shape. `GetPermalink` should never return a Connect error for "commit unpushed" or "non-GitHub remote" — those are successful responses with `availability = NO_LINK` / `LINKABLE_UNVERIFIED`, mirroring `Explore`'s `empty=true` contract (CONTEXT D-14) and `degradedStatus`'s answer-not-error shape (`degrade.go:136+`).

---

### `internal/gitmeta/` extension (service, event-driven via git subprocess)

**Analog:** `internal/gitmeta/worktree.go:1-52` (`WorktreeRoot`) + `internal/indexer/commit.go:14-80` (`resolveHeadCommitSHA`).

**Package doc / degrade contract** (`internal/gitmeta/worktree.go:1-18`):
```go
// Package gitmeta is the stdlib-only, best-effort git introspection layer
// ...
// Every function here degrades to a safe zero value on ANY failure: missing
// git, a non-repo path, a timeout, or a transient error all report "no
// signal" rather than an error. A read query must never fail or block on
// git being unavailable, slow, or absent (WORK-03).
package gitmeta

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gitTimeout = 5 * time.Second

func WorktreeRoot(ctx context.Context, dir string) string {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	cmd.Stdin = nil // git must never be able to block on an interactive prompt
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	...
}
```

**Test-only seam pattern** (`internal/indexer/commit.go:14-20`):
```go
var gitExecLookPath = exec.LookPath
```
This is the exact seam shape (`internal/uiserver/handlers.go:34`'s `var openEngine = query.OpenAt` is the SAME shape at a different call site) to copy for any git-availability check the permalink derivation needs.

**D-07's own code example** (already synthesized in `03-RESEARCH.md` Code Examples section, following this exact pattern):
```go
cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "-r", "--contains", sha)
out, err := cmd.Output()
if err != nil {
    return availabilityNoLink, "could not verify remote branches"
}
if len(bytes.TrimSpace(out)) == 0 {
    return availabilityLinkableUnverified, "commit not found on any remote-tracking branch — link may 404"
}
return availabilityLinkable, ""
```

**Remote-URL derivation** — NO existing gitmeta function reads `git remote get-url origin` or normalizes `insteadOf` rewrites; `worktree.go`'s `WorktreeRoot`/`CommonDir` are the closest analog for "shell out, timeout-bound, degrade to empty string" but a new function is needed. Follow the identical `exec.CommandContext` + `cmd.Stdin = nil` + `gitTimeout` shape.

---

### `internal/uiserver/confinement_test.go` (test, request-response — SRV-05/D-02's RPC-boundary regression test)

**Analog:** `internal/query/errors_test.go:45,139` (the existing one-layer-down assertions to mirror at the RPC boundary) + `internal/indexer/capability/matrix_test.go:33-77` (positive-control / no-missing-no-extra discipline, rule `84d1gfpywd`).

**Existing confinement assertion this test mirrors, one layer down** (`internal/query/errors_test.go:45,139`):
```go
{"SourceFor absolute path", sourceForErr(t, e, "/etc/passwd"), `query: absolute path "/etc/passwd" is not allowed`},
{"SourceFor escapes repo root", sourceForErr(t, e, "../outside.txt"), `query: path "../outside.txt" escapes the repo root`},
```
(from `TestClassifiedErrorsPreserveTheirMessages`, `internal/query/errors_test.go:45`) and
```go
{"SourceFor empty path", sourceForErr(t, e, ""), ErrInvalidArgument, "query: empty file path"},
{"SourceFor absolute path", sourceForErr(t, e, "/etc/passwd"), ErrInvalidArgument, `query: absolute path "/etc/passwd" is not allowed`},
```
(from `TestEveryReachableErrorIsClassified`, `internal/query/errors_test.go:139`).

**Positive-control / discriminating-guard discipline** (`internal/indexer/capability/matrix_test.go:33-41`):
```go
// TestMatrix_CoversRegisteredLanguages proves the D-11 descriptor covers
// EXACTLY the languages registered in the LanguageSpec registry — no
// missing (a registered language with no matrix entry would silently
// overclaim nothing about it), no extra (a matrix entry for an
// unregistered language would be describing something that doesn't exist).
func TestMatrix_CoversRegisteredLanguages(t *testing.T) {
	registered := indexer.RegisteredLanguageIDs()
	...
	var missing, extra []string
	for _, id := range registered {
		if !descriptorSet[id] {
			missing = append(missing, id)
		}
	}
```
CONTEXT.md D-02 requires the SAME shape applied to Phase 3's new test: a refusal-only assertion "passes vacuously if the RPC starts refusing everything" (per rule `84d1gfpywd`), so the new test MUST pair each `../`/absolute-path/symlink-escape refusal case with a legitimate in-repo path in the SAME test that returns real source — the "positive control" `matrix_test.go` demonstrates via missing/extra set-equality, adapted here to refusal/success pairing rather than set membership.

**What is verified absent (searched, not assumed):** `internal/uiserver/*_test.go` currently has zero occurrences of `confinement` or `escapes the repo root` (CONTEXT D-02, re-verifiable via `rg -n "confinement|escapes the repo root" internal/uiserver/`) — this is the gap the new test file closes.

---

### `web/src/routes/browse/+page.svelte` (route/component, request-response)

**Analog:** `web/src/routes/+page.svelte` (the only existing Svelte page that calls a UIService RPC).

**onMount + typed state-union + uiClient call pattern** (`web/src/routes/+page.svelte:1-29`):
```svelte
<script lang="ts">
	import { onMount } from 'svelte';
	import { uiClient } from '$lib/client';
	import type { GetStatusResponse } from '$lib/gen/ui_pb';

	type LoadState =
		| { kind: 'loading' }
		| { kind: 'error'; message: string }
		| { kind: 'loaded'; status: GetStatusResponse };

	let state = $state<LoadState>({ kind: 'loading' });

	onMount(() => {
		uiClient
			.getStatus({})
			.then((status) => {
				state = { kind: 'loaded', status };
			})
			.catch((err: unknown) => {
				state = { kind: 'error', message: err instanceof Error ? err.message : String(err) };
			});
	});
</script>
```
This discriminated-union `LoadState` shape is the direct precedent for `browse/+page.svelte`'s own state modeling (search results / node detail / picker / error states), and for D-04's shared `rpc-errors.ts` — the `.catch((err: unknown) => ...)` site here is exactly the ad-hoc per-view try/catch CONTEXT.md's D-04 explicitly says to extract into one shared module instead of repeating.

**Server field access convention:** `status.initialized`, `status.indexingInProgress` (camelCase, generated from the proto's `initialized`/`indexing_in_progress` — confirms Connect-ES's proto→TS field-name transform for all new `GetPermalinkResponse` fields the browse page will read, e.g. `response.availability`, `response.url`).

**Nav placeholder this phase replaces** (`web/src/layout` nav array, `web/src/routes/+layout.svelte:16-21`):
```typescript
const navEntries = [
	{ href: '/browse', label: 'Browse' },
	{ href: '/workbench', label: 'Workbench' },
	{ href: '/graph', label: 'Graph' },
	{ href: '/health', label: 'Health' }
];
```
`/browse` is already a live route; no layout change needed for Phase 3.

**Placeholder being filled** (`web/src/routes/browse/+page.svelte`, current full content):
```svelte
<!--
  Placeholder slot for Phase 3 (Browse, Inspect & Navigation): ...
-->
<h1 class="text-lg font-semibold">Browse</h1>
<p class="mt-1 text-sm text-muted-foreground">
	Phase 3: find any symbol or file, read its verbatim source with callers, callees and blast
	radius.
</p>
```

---

### `web/src/lib/status.ts`, `web/src/lib/rpc-errors.ts`, `web/src/lib/browse-url.ts`, `web/src/lib/highlight.ts` (utility/store, D-04/D-13/D-19)

**NO IN-REPO ANALOG.** `web/src/lib/` today holds only `client.ts`, `gen/ui_pb.ts`, `utils.ts`, `assets/` (verified: `ls web/src/lib` — confirmed by CONTEXT.md D-22's own citation of this same fact, independently re-confirmed this session). There is no existing SvelteKit store module, no existing error-mapping module, no existing URL-param module, and no existing highlight/syntax module anywhere in `web/src/`.

**Closest adjacent pattern for `status.ts` and `rpc-errors.ts`:** `web/src/routes/+page.svelte`'s inline `onMount`/`uiClient.getStatus({})`/`.catch()` block (cited above) is the only precedent for "call a UIService RPC and branch on success/failure" — `status.ts` and `rpc-errors.ts` are new client.ts-adjacent modules extracting that inline shape into reusable form, following `client.ts`'s own module-level `export const` pattern (`web/src/lib/client.ts:26-30`):
```typescript
export const transport = createConnectTransport({
	baseUrl: "/",
	useBinaryFormat: false,
});

export const uiClient = createClient(UIService, transport);
```

**Cross-language analog for `rpc-errors.ts` (informational only, not a TS precedent):** `internal/uiserver/handlers.go:101-129`'s `mapEngineError` is the SERVER-side error-classification counterpart this TS module mirrors on the wire's OTHER end — same three-way split (`CodeNotFound`/`CodeInvalidArgument`/`CodeUnavailable`+`IndexingInProgress` detail) the client must recognize, per CONTEXT D-04's table. Cited for the vocabulary (Connect codes + typed detail), not as a TS code pattern.

**No analog for `browse-url.ts` and `highlight.ts` at all** — these are net-new problem domains (URL query-param parse/serialize; syntax highlighter registration) this repo has never implemented in either language. Follow `03-RESEARCH.md`'s Pattern 1/2 (`goto()`-driven URL state) and Pattern 3 (highlight.js selective registration) code examples directly — those are RESEARCH's synthesized examples, not in-repo analogs, and should be labeled as such in the plan.

---

### `web/src/lib/components/ui/command/`, `web/src/lib/components/ui/popover/` (shadcn-svelte-vendored components)

**NO IN-REPO ANALOG.** `web/src/lib/` contains zero `components/ui/*` subdirectories today (confirmed: `ls web/src/lib` lists only `client.ts`, `gen/`, `utils.ts`, `assets/` — no `components/` directory exists in the tree at all). Phase 3 is, per CONTEXT.md D-22's own text, "the first phase that makes the [vendored-component supply-chain] gap live." There is no in-repo shadcn-svelte component to pattern-match against; these are added fresh via `pnpm dlx shadcn-svelte@latest add command popover` (per `03-RESEARCH.md`'s Installation section) and reviewed as committed code at add-time, not modeled on an existing file.

## Shared Patterns

### Path Confinement (SRV-05's mandated reuse target)
**Source:** `internal/query/node.go:29-90` (`resolveSourcePath`, `readSourceFile`)
**Apply to:** Any Phase 3 code touching a client-steerable file path — currently only `GetNodeDetailRequest.file` (per D-01's enumeration; the phase adds NO new source-reading endpoint per CONTEXT's resolution of the D-01 tension, only a regression test at the existing RPC boundary).
```go
func (e *Engine) resolveSourcePath(relPath string) (string, error) {
	if e.repoRoot == "" { ... }
	if relPath == "" {
		return "", invalidArgumentf("query: empty file path")
	}
	if filepath.IsAbs(relPath) {
		return "", invalidArgumentf("query: absolute path %q is not allowed", relPath)
	}
	cleaned := filepath.Clean(relPath)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", invalidArgumentf("query: path %q escapes the repo root", relPath)
	}
	...
	// WR-03: re-verify confinement after resolving symlinks
	resolvedRoot, err := filepath.EvalSymlinks(root)
	...
}
```
This is the SINGLE implementation both MCP and uiserver already share via `internal/query.Engine.SourceFor` (`internal/query/detail.go:258`, documented as "a wrapper over the existing confinement gate, not a second read path"). SRV-05 is satisfied structurally already — Phase 3 adds no new call into this function, only a test asserting the existing wiring.

### Degrade-to-empty-never-error (git absence, unpushed commits)
**Source:** `internal/indexer/commit.go:41-80` (`resolveHeadCommitSHA`)
**Apply to:** `internal/gitmeta`'s new remote-URL and pushed-check functions (D-06/D-07); `GetPermalink`'s handler must never return a Connect error for "git absent" or "commit not pushed" — only `availability = NO_LINK`/`LINKABLE_UNVERIFIED`.
```go
func resolveHeadCommitSHA(repoPath string) string {
	if _, err := gitExecLookPath("git"); err != nil {
		return ""
	}
	...
	if err != nil {
		return ""
	}
	sha := strings.TrimSpace(string(out))
	if !isLowercaseHexCommitSHA(sha) {
		return ""
	}
	return sha
}
```

### Connect Error Classification (server side, for the client's `rpc-errors.ts` mapper to mirror)
**Source:** `internal/uiserver/handlers.go:101-129` (`mapEngineError`)
**Apply to:** `internal/uiserver/permalink.go` if it needs to surface a genuine error (it should not, per D-06/D-07's degrade contract — this is cited for the VOCABULARY the TS mapper must recognize, not as a pattern `permalink.go` itself copies).
```go
func mapEngineError(err error) error {
	if err == nil { return nil }
	switch {
	case errors.Is(err, query.ErrNotFound), errors.Is(err, graphstore.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, query.ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, graphstore.ErrStoreLocked):
		return errIndexingInProgress()
	default:
		writeDiagLine("internal error: %v", err)
		return connect.NewError(connect.CodeInternal, errInternal)
	}
}
```

### One-Concern-Per-File With Decision-ID Doc Comments
**Source:** `internal/uiserver/originguard.go:1-9`, `internal/uiserver/degrade.go:1-20`, `internal/uiserver/truncate.go:1-13`
**Apply to:** `internal/uiserver/permalink.go` — open with a package/file doc comment citing D-06/D-07/D-08/D-09 by name, mirroring:
```go
// Package uiserver implements codegraph ui's local, loopback-only Connect
// RPC server. originguard.go is the first piece to exist in this package
// (SRV-02): the exact-match Origin/Host control that must run before any
// RPC handler is reachable at all — ...
```

### Test-Only Control Seam (package-level var, no exported setter)
**Source:** `internal/uiserver/handlers.go:34` (`var openEngine = query.OpenAt`), `internal/indexer/commit.go:20` (`var gitExecLookPath = exec.LookPath`)
**Apply to:** any new git-availability or engine-open seam `permalink.go`/`gitmeta` additions need for deterministic testing.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `web/src/lib/rpc-errors.ts` | utility | transform | No TS error-mapping module exists anywhere in `web/src/`; closest is the Go-side `mapEngineError` (cross-language, cited for vocabulary only) |
| `web/src/lib/browse-url.ts` | utility | transform | No URL parse/serialize module exists; net-new problem domain for this repo |
| `web/src/lib/highlight.ts` | utility | transform | No syntax-highlighting code exists anywhere in the tree; net-new dependency and pattern |
| `web/src/lib/components/ui/command/`, `.../popover/` | component | request-response | `web/src/lib/` has zero `components/ui/*` today (confirmed via directory listing); Phase 3 is the first phase to add vendored shadcn-svelte components per CONTEXT D-22 |
| `internal/gitmeta/`'s new remote-URL-derivation function | service | event-driven | `worktree.go` has no existing `git remote get-url`/`insteadOf`-normalization function; `WorktreeRoot`/`CommonDir` are the closest shape-analog (exec.CommandContext + timeout + degrade-to-empty), not a literal precedent |

## Metadata

**Analog search scope:** `internal/uiserver/`, `internal/query/`, `internal/gitmeta/`, `internal/indexer/`, `internal/indexer/capability/`, `internal/uiproto/uiv1/`, `internal/schema/`, `web/src/routes/`, `web/src/lib/`
**Files read this session:** `internal/uiserver/handlers.go` (partial, targeted), `internal/uiserver/degrade.go` (full), `internal/uiserver/originguard.go` (head), `internal/uiserver/truncate.go` (head), `internal/uiserver/spa.go` (lines 190-230), `internal/query/node.go` (full), `internal/query/errors_test.go` (lines 1-150), `internal/gitmeta/worktree.go` (lines 1-80), `internal/indexer/commit.go` (lines 1-80), `internal/indexer/capability/matrix_test.go` (lines 1-60), `internal/uiproto/uiv1/ui.proto` (lines 1-60), `web/src/routes/+page.svelte` (full), `web/src/routes/+layout.svelte` (full), `web/src/routes/+layout.ts` (full), `web/src/routes/browse/+page.svelte` (full), `web/src/routes/workbench/+page.svelte` (full), `web/src/routes/health/+page.svelte` (full), `web/src/lib/client.ts` (full), `web/package.json` (full)
**Pattern extraction date:** 2026-08-28
