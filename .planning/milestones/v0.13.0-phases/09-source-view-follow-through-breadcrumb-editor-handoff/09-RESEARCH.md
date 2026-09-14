# Phase 9: Source View Follow-Through — Breadcrumb & Editor Handoff - Research

**Researched:** 2026-09-12
**Domain:** Svelte 5 client-side rendering (breadcrumb), ConnectRPC wire-surface design (new read-only rpc), and third-party editor URI-scheme handoff (external-protocol navigation)
**Confidence:** MEDIUM — the in-repo mechanics (proto discipline, path confinement, Playwright gate pattern) are HIGH confidence, directly read from source at line-level this session. The three editor URI-scheme syntaxes are MEDIUM/LOW confidence — only VS Code's is confirmed against an official page; Cursor's and JetBrains' are cross-referenced community sources, not official specs, and are flagged for empirical confirmation during execution.

## Summary

This phase bolts two independent, additive features onto one existing mount point,
`SourcePane.svelte`, plus the threat model covering both. BRW-10 (breadcrumb) is pure
frontend work: `FileSymbols` (UIService's 13th rpc, already shipped) already returns every
symbol's `start_line`/`end_line` for a file; the breadcrumb is a derived-state computation
over data the client already fetches (the graph page's `fileSymbolsCache` pattern is the
precedent to reuse) plus a scroll listener. No proto, Engine, or CLI change.

BRW-11/12 add `GetEditorLink`, UIService's 15th rpc, whose contract mirrors `GetPermalink`
(03-05, the 10th rpc) almost exactly: an `Availability` closed enum standing in for
`PermalinkAvailability`, `(*query.Engine).ValidateRepoRelativePath` as the single confinement
gate, `CodeInvalidArgument` naming only the caller's own repo-relative path (never the
absolute host path) for the one true error case, and the same "configuration state is an
answer, never an error" discipline. The template resolution precedence (flag → env →
discovered default → unconfigured) and the malformed-value-is-a-startup-failure rule (D-17)
are new *policy*, but the code shape — a `--no-open`-style boolean flag, a `CODEGRAPH_*` env
var, and startup-time (never per-request) discovery via `exec.LookPath` — all have direct
precedent elsewhere in this codebase.

The riskiest unknowns are external, not architectural: the exact URI syntax each of VS Code,
Cursor, and JetBrains actually accept for file+line+column, and whether an async
click-handler → RPC-round-trip → external-protocol-navigation sequence survives each
browser's transient-user-activation window (Safari is measurably stricter than
Chromium/Firefox here — see Common Pitfalls). Both require empirical confirmation with a real
browser during execution, not just documentation reading.

**Primary recommendation:** Ship `GetEditorLink` as `GetPermalink`'s structural twin (closed
enum, single confinement gate, additive proto, same reason-string discipline), reuse the
`fileSymbolsCache`-per-path fetch pattern for the breadcrumb's data source, resolve the header
"open in editor" link via the load-time probe (D-12) so its `<a href>` is a plain synchronous
browser navigation (sidestepping the transient-activation risk entirely for that path), and
budget one Playwright script (`web/scripts/breadcrumb-check.mjs`, following the
`graph-collapse-affordance-check.mjs` precedent exactly) as the sanctioned, re-runnable
live-browser gate for BRW-10's criterion 1.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Breadcrumb symbol-containment computation | Browser / Client | — | Pure derived state over `FileSymbols` data already on the client (D-01/D-02); no server round trip per scroll event, ever |
| Editor-link template resolution & validation | API / Backend (`internal/uiserver`) | — | D-05/D-07: repo-root confinement and the scheme allowlist MUST be enforced server-side so a browser client can never weaken it; this is the phase's one hard security requirement |
| Editor-link template *source of truth* (flag/env/discovery) | API / Backend (`internal/cli`, `internal/uiserver.Options`) | — | Startup-time only (D-15); never re-probed per request, following the existing "environment facts frozen at Listen" pattern (publisher, engine) |
| Per-browser override storage | Browser / Client (`localStorage`) | — | D-18: server is stateless/read-only by construction (SRV-03); nothing is ever persisted server-side |
| Editor-link URL construction (`{path}` substitution) | API / Backend | — | D-08: absolute path is joined server-side, after confinement passes — never client-side, so the client can never smuggle an unconfined path into the template |
| External protocol dispatch (actually launching the editor) | Browser / OS | — | Out of scope by construction (SRV-03); the browser's own URI-handler prompt is the sole mechanism, never a server-side shell-out |

## Standard Stack

### Core

No new runtime dependency is required. Everything BRW-10/11/12/13 need is already a
dependency of this repo:

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@bufbuild/protobuf` / `@connectrpc/connect-web` | 2.14.0 / 2.1.2 (pinned, `web/package.json` [VERIFIED: web/package.json:36-37]) | Generated TS client for the new `GetEditorLink` rpc | Already the sole RPC transport for every other UIService method; no second client library is introduced |
| `connectrpc.com/connect` (Go) | already a repo dependency | Server-side rpc handler, `connect.CodeInvalidArgument` | Same reasoning; `GetPermalink`'s handler is the direct template |
| Svelte 5 runes (`$state`, `$effect`, `$props`) | `svelte@^5.56.1` [VERIFIED: web/package.json:33] | Breadcrumb reactive state, per-browser override popover | Already the sole component model in `web/src` |
| `@playwright/test` | `1.62.1` [VERIFIED: web/package.json:22, exact-pinned] | Live-browser gate for BRW-10 criterion 1 | Already the sanctioned, committed-script live-browser mechanism (`web/scripts/graph-*.mjs`); `agent-browser` is explicitly rejected in this repo for anything that must be a re-runnable gate (05-04-PLAN.md, quoted in Architecture Patterns below) — reserved for one-off UAT only |

### Supporting

No new supporting library. `highlight.js` (already a dependency, used by `highlightSource`)
is untouched — the per-line DOM restructuring (D-04's consequence) wraps its output, it does
not replace it.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| A new rpc (`GetEditorLink`) | Extending `GetPermalink` with editor fields | Rejected explicitly in CONTEXT.md D-05: `PermalinkAvailability` encodes remote-trust semantics (git remote, commit push status) that have no analog for a local editor link, which has no remote and no commit at all. Reusing the enum would force editor-link callers to interpret git-shaped values that never apply to them. |
| `localStorage` for the per-browser override | A server-side cookie/session | Rejected by SRV-03 (server is stateless, read-only by construction — no Set/Put/Save rpc may exist). `localStorage` is also the ONLY way to scope a preference "per browser" as D-18 specifies, since a cookie sent with every RPC would need a Set-Cookie response the server is not allowed to emit. |
| A committed Playwright script for BRW-10's live-browser proof | Ad hoc `agent-browser` session | `agent-browser` is fine for a one-off UAT pass but was explicitly rejected in 05-04-PLAN.md for anything that must be "re-runnable identically" as a gate — it is not in the lockfile and sits outside the supply chain. BRW-10's criterion 1 demands exactly the kind of re-runnable, RED-then-GREEN-demonstrable proof only a committed script gives. |

**Installation:** none — no `pnpm add` / `go get` is required for this phase's stack. If a
package IS needed later (e.g. a URL-template parsing helper), it must clear the Package
Legitimacy Gate below before use; none is currently planned.

## Package Legitimacy Audit

No external packages are introduced by this phase's `Standard Stack` above — every library
used is already a dependency, already legitimacy-cleared in a prior phase (`@playwright/test`
in 05-04, `@connectrpc/connect-web` in Phase 2/3). This section is included per the
instructions' "required whenever a phase installs external packages" rule; the honest
disposition here is **N/A — no new packages**.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| *(none introduced)* | — | — | — | — | — | N/A |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

## Architecture Patterns

### System Architecture Diagram

```
Browser (SPA)                                    Server (uiserver)
──────────────                                    ─────────────────
 SourcePane.svelte
   │
   ├─ on file target load ─────────────────────▶  FileSymbols(path)   [existing rpc,
   │  (existing rpc, already shipped)              no change]
   │      │
   │      ▼
   │  fileSymbolsCache (per-path, client-side)
   │      │
   │      ▼
   │  scroll listener → firstFullyVisibleLine()
   │      │
   │      ▼
   │  innermostContaining(fileSymbolsCache, line)   ← PURE CLIENT COMPUTATION,
   │      │                                            zero new rpcs (BRW-10)
   │      ▼
   │  sticky breadcrumb bar (renders symbol name,
   │  or dimmed file-path-only "empty" state)
   │
   ├─ on pane load ─────────────────────────────▶  GetEditorLink(path, startLine)
   │  (probe, once per opened target — D-12)          │
   │      │                                            ▼
   │      │                                    ValidateRepoRelativePath(path)
   │      │                                       (SAME gate GetPermalink/
   │      │                                        GetNodeDetail/FileSymbols use)
   │      │                                            │
   │      │                              reject ◀──── fails: traversal / absolute /
   │      │                          CodeInvalidArgument   symlink-escape / empty
   │      │                          (names caller's own                 │
   │      │                           rel path only)                  passes
   │      │                                                              │
   │      │                                                              ▼
   │      │                                          resolve effective template:
   │      │                                          per-request override (if any)
   │      │                                            → else server default
   │      │                                              (flag → env → discovered
   │      │                                               → NO_TEMPLATE)
   │      │                                                              │
   │      │                                                              ▼
   │      │                                          validate template against
   │      │                                          scheme allowlist (D-13)
   │      │                                                              │
   │      │                                            invalid: TEMPLATE_INVALID
   │      │                                            valid: fill {path}/{line}/{col}
   │      │                                            (path = RepoPath + confined rel,
   │      │                                             percent-encoded per placeholder)
   │      ◀────────────────────────────────────────  EditorLinkAvailability +
   │      │                                            url (if BUILDABLE) + reason
   │      ▼
   │  header "Open in editor" <a href> becomes
   │  real/clickable; gutter line numbers become
   │  clickable links
   │
   ├─ on line-number click ────────────────────▶  GetEditorLink(path, line, col=1,
   │  (per click, on demand — D-12)                 override?)
   │      │                                            (same validate/resolve/build
   │      ◀────────────────────────────────────────    pipeline as above)
   │      ▼
   │  navigate to returned url
   │  (browser's own external-protocol prompt
   │   fires here — the ONE security boundary,
   │   BRW-13 — never CSP)
   │
   └─ per-browser override popover (gear icon)
      reads/writes localStorage only; server
      never stores it (SRV-03)
```

### Recommended Project Structure

No new top-level directories. Changes land inside the existing tree:

```
internal/uiproto/uiv1/ui.proto          # + GetEditorLink rpc, request/response, enum (15th rpc)
internal/uiserver/
  editorlink.go                          # new handler file, mirrors permalink.go's shape
  editorlink_test.go                     # confinement + scheme-allowlist + reason-string tests
  server.go                              # Options gains the resolved template / disabled state
  readonly_test.go                       # wantUIServiceMethods +1 (15), mutatingVerbs re-run
internal/cli/
  ui.go                                  # --editor-url / --no-editor-url flags, env fallback,
                                          # startup discovery, fail-fast on malformed value (D-17)
  editordiscovery.go                     # new: exec.LookPath + platform app-dir probing (D-15)
web/src/lib/
  components/browse/SourcePane.svelte    # sticky breadcrumb bar, per-line DOM + gutter,
                                          # header "Open in editor" link + popover
  browse-state.ts                        # file target state may carry FileSymbols if needed
  editor-prefs.ts                        # new: localStorage read/write for the per-browser
                                          # override (D-18), try/catch-wrapped
web/scripts/
  breadcrumb-check.mjs                   # new: Playwright live-browser gate for BRW-10 crit 1
.planning/phases/09-.../
  09-SECURITY.md                         # BRW-13
  09-MUTATION-LOG.md                     # RED demonstrations (stale/empty breadcrumb; rejected path)
```

### Pattern 1: Additive read-only rpc mirroring an existing sibling

**What:** `GetEditorLink` is built as a structural copy of `GetPermalink`'s shape: same
confinement gate, same closed-enum-for-configuration-state discipline, same
"one true error case is caller path input" rule, same reason-string-never-shared-across-causes
convention.
**When to use:** Any time a new UIService rpc answers a question closely related to an
existing one but with genuinely different semantics (here: no remote, no commit — D-05).
**Example (structural template, from the actual `GetPermalink` handler this session read):**
```go
// Source: internal/uiserver/permalink.go (read this session, lines 118-186)
func (s *uiService) GetPermalink(ctx context.Context, req *connect.Request[uiv1.GetPermalinkRequest]) (*connect.Response[uiv1.GetPermalinkResponse], error) {
    // ... request-shape validation with no Engine dependency, before withEngine ...
    var classifiedErr *connect.Error
    err := withEngine(ctx, s.repoPath, func(eng *query.Engine) error {
        path := req.Msg.GetPath()
        if verr := eng.ValidateRepoRelativePath(path); verr != nil {
            // ... one reclassified case (fs.ErrNotExist) -> CodeInvalidArgument naming only
            //     the caller's own repo-relative path; everything else falls through
            return verr
        }
        // ... build the answer; configuration states are ALWAYS a successful response ...
        return nil
    })
    // ...
}
```
`GetEditorLink` follows this exact shape: `ValidateRepoRelativePath` first (SRV-05), a closed
`EditorLinkAvailability` enum for `BUILDABLE`/`NO_TEMPLATE`/`TEMPLATE_INVALID`, and
`CodeInvalidArgument` reserved for the one true caller-input error (a rejected path), never for
"no template configured."

### Pattern 2: Per-path fetch-and-cache for a file-scoped rpc

**What:** The graph page's `fileSymbolsCache`/`pendingFileSymbols` pattern: a `Map<string,
FileSymbolsResponse>` keyed by path, a `Set<string>` of in-flight paths to prevent duplicate
requests, populated once per path and reused.
**When to use:** BRW-10's breadcrumb needs the SAME `FileSymbols` data the graph page already
fetches, but in `SourcePane` rather than `+page.svelte`'s route component — the caching
mechanism (not the cache instance) is the reusable part.
**Example:**
```ts
// Source: web/src/routes/graph/+page.svelte (read this session, lines 59-82, 419-494)
let fileSymbolsCache = $state<Map<string, FileSymbolsResponse>>(new Map());
let pendingFileSymbols = new Set<string>();
// ... on demand: check cache first, then pendingFileSymbols, then issue exactly one
// uiClient.fileSymbols({ path: id }) call, writing the result into a NEW Map (never
// mutating the existing one in place, for Svelte 5 reactivity) ...
```

### Pattern 3: Startup-time environment discovery, never per-request

**What:** `uiserver.Listen`'s live-push publisher and the SPA build filesystem are both
constructed once, at `Listen`, and never re-probed per RPC. Editor discovery (D-15) follows
the identical discipline: `exec.LookPath` and platform app-directory `stat` calls run exactly
once at CLI startup (before `uiserver.Listen`, so a malformed explicit value can fail fast —
D-17), never inside `GetEditorLink`'s handler.
**When to use:** Any environment fact that does not change during the process's lifetime.
**Example (the precedent, not editor-specific):**
```go
// Source: internal/uiserver/server.go (read this session, lines 143-152)
// "The publisher is constructed against its OWN context ... construction fails ONLY if
// fsnotify itself cannot be created ... Discovery runs once at Listen, never per request."
```

### Anti-Patterns to Avoid

- **Client-side path confinement as a "second" check:** D-08/SRV-05 already establish that
  confinement is server-side ONLY; a client-side filter on the `{path}` value would be a
  second implementation that can silently drift from the real one (03-SECURITY.md's T-03-17
  names this exact anti-pattern and accepts it as closed precisely because no client-side copy
  exists).
- **A denylist for the scheme allowlist (D-13):** rule `84d1gfpywd`-adjacent — a denylist
  (`javascript:`, `data:`, `blob:`, `file:` explicitly listed as forbidden) is a negative-only
  guard that passes vacuously the moment a new dangerous scheme is invented. D-13 mandates a
  **positive** allowlist check on the parsed scheme.
- **Reprobing editor discovery per request:** would violate the "environment facts frozen at
  Listen" discipline this codebase already enforces elsewhere, and would turn a read-only rpc
  into one with `exec.LookPath` filesystem side effects on every call.
- **Async RPC-then-`window.open()` for the line-number gutter click:** risks losing the
  browser's transient user-activation window (see Common Pitfalls) — verify against a real
  browser rather than assuming the Chromium-dev-loop behavior generalizes to Safari/WebKit.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Path confinement for `{path}` | A second `filepath.Clean`/`Rel`/`EvalSymlinks` check inside `editorlink.go` | `(*query.Engine).ValidateRepoRelativePath` | Already the ONE gate `GetNodeDetail`, `GetPermalink`, and `FileSymbols` all share [VERIFIED: internal/query/node.go:81-94, quoted above]; a second implementation is exactly the drift risk 03-SECURITY.md's T-03-17/T-03-01b entries exist to prevent |
| URL percent-encoding for the `{path}` placeholder | A hand-rolled string-replace/escape routine | `net/url.PathEscape` per path segment (Go stdlib), the SAME pattern `permalink.go`'s `percentEncodeRepoPath` already implements | `GetPermalink`'s own history (IN-03) shows what goes wrong without per-segment escaping: a literal `#`/`?` in a filename previously produced a URL a browser parses as a fragment/query break |
| Per-browser preference persistence | A server-side session/cookie mechanism | `localStorage`, wrapped in try/catch (D-18) | The server is stateless and read-only by construction (SRV-03) — no Set/Put/Save rpc may exist, so there is nowhere server-side to persist it even if desired |
| A live-browser proof of BRW-10's criterion 1 | An ad hoc `agent-browser` session recorded only in prose | A committed `@playwright/test` script under `web/scripts/`, following `graph-collapse-affordance-check.mjs`'s exact structure (real Chromium, polling a published DOM/global seam, writing a diagnostic JSON record) | This repo has already litigated this exact choice (05-04-PLAN.md) and rejected `agent-browser` for anything that must be a re-runnable, RED-then-GREEN-demonstrable gate |

**Key insight:** every "don't hand-roll" item in this phase already has a shipped precedent
in this exact codebase — the discipline is reuse-the-existing-gate, not invent-a-new-one. The
phase's own CONTEXT.md D-08/D-09 sections name the precedent file for each case explicitly.

## Runtime State Inventory

Not applicable — this phase is purely additive (new rpc, new UI surface, new flags). It is not
a rename, refactor, or migration phase. No existing stored data, live service config,
OS-registered state, secret/env-var name, or build artifact is renamed or restructured.

**Nothing found in any category** — verified by reading CONTEXT.md's `<domain>` section
(purely additive scope) and confirming no existing `CODEGRAPH_EDITOR_URL`-shaped or
`--editor-url`-shaped configuration exists anywhere in the repo today
[VERIFIED: `rg -n "editor.url\|EDITOR_URL" internal/ web/` returns zero hits — checked this
session].

## Common Pitfalls

### Pitfall 1: Transient user-activation window lost between click and external-protocol navigation

**What goes wrong:** A click handler that calls an async `GetEditorLink` rpc and THEN
navigates (`window.location.href = url` or `window.open(url)`) risks the navigation being
silently blocked as if it were an unsolicited popup, because the browser's "this came from a
real user gesture" flag can expire or be consumed before the async work completes.
**Why it happens:** Per MDN and WebKit's own engineering blog [CITED:
developer.mozilla.org/en-US/docs/Web/Security/Defenses/User_activation,
webkit.org/blog/13862], "transient activation" is a short-lived window starting at a genuine
user gesture (click, keypress); the browser does NOT pause this timer during async work — by
the time a `Promise` resolves, especially in Safari/WebKit, the activation may already have
expired. Community writeups [CITED: sitepoint.com, mai1015.com — cross-referenced, consistent
with the above] confirm Safari on iOS is measurably stricter than Chromium/Firefox, where "any
small delay... is enough to miss this window."
**How to avoid:**
- For the header "Open in editor" link (D-12's per-target probe), this risk is ALREADY
  avoided by design: the rpc runs once when the pane loads (not on click), so by the time the
  user clicks, `<a href="...">` is a plain, already-resolved anchor — ordinary browser-native
  navigation, not JS-triggered, with zero activation risk.
- For the line-number gutter (per-click, on-demand rpc — D-12), the round trip is local
  loopback (typically single-digit milliseconds), which is very likely to stay inside
  Chromium/Firefox's activation window, but Safari's stricter timing is a real, unresolved
  risk the phase's own D-12 flags as needing research confirmation. Recommend: (a) prefer
  `location.href = url` over `window.open(url)` for the actual navigation (no new-tab/popup
  semantics to trip a popup blocker), and (b) verify empirically with a live-browser Playwright
  check (at minimum against the default CI/dev browser, Chromium) that the external-protocol
  behavior actually fires after the round trip, before treating the click-driven gutter path as
  equivalent in reliability to the header link.
**Warning signs:** A line-number click that silently does nothing in one browser but works in
another; no console error (browsers do not always report a blocked external-protocol
navigation as an error).

### Pitfall 2: Editor URI scheme details are not authoritatively documented for two of the three presets

**What goes wrong:** Shipping a JetBrains or Cursor preset template built from
community-sourced syntax that turns out to be wrong for a real installed IDE version, or wrong
in some edge case (missing `project=` parameter, wrong placeholder order).
**Why it happens:** VS Code's `vscode://file/{full path}:{line}:{column}` syntax is confirmed
against an OFFICIAL page [CITED: code.visualstudio.com/docs/editor/command-line —
`vscode://file/c:/myProject/package.json:5:10` quoted verbatim in the docs]. Cursor's
`cursor://file/{path}:{line}` syntax is sourced ONLY from a community forum post
[CITED: forum.cursor.com/t/does-cursor-have-a-unique-open-scheme/3659 — not an official Cursor
doc] and multiple GitHub issues note Cursor's OWN command-line handling of `file:line:col`
syntax has known gaps as of mid-2026 [CITED: github.com/cursor/cursor/issues/3258,
github.com/cursor/cursor/issues/1858] — whether the *URL-handler* path (as opposed to the CLI
argument path) reliably honors line/column is unconfirmed. JetBrains' `jetbrains://<ide>/
navigate/reference?project=<name>&path=<file>:<line>:<col>` syntax is corroborated across
several independent community sources [CITED: medium.com/@alanhe421, github.com/alanhe421/
jetbrains-url-schemes] but a JetBrains YouTrack ticket exists specifically because "comprehensive
official documentation" for this URL scheme is missing [CITED: youtrack.jetbrains.com/projects/
TBX/issues/TBX-3965]. Additionally, JetBrains' documented examples use a **project-relative**
`path` (e.g. `path=lib/hello.rb`), which may be in tension with D-08's design of always filling
`{path}` with the server's **absolute** filesystem path — this is a genuine open design
question, not settled by this research.
**How to avoid:** Treat Cursor's and JetBrains' exact preset template strings as `[ASSUMED]`
(see Assumptions Log) requiring a `checkpoint:human-verify` or a live manual test against a
real installed Cursor/JetBrains IDE before the preset ships as "known-good" — do not present
either as equivalent-confidence to the VS Code preset in any user-facing copy or code comment.
**Warning signs:** A preset that opens the editor but lands on the wrong line, or opens the
wrong project window, or (for JetBrains) prompts for a project selection instead of navigating
directly.

### Pitfall 3: Single-blob `<pre>` rendering blocks BOTH the breadcrumb's line mapping and the gutter

**What goes wrong:** Attempting to ship BRW-10's breadcrumb OR BRW-11/12's line-number gutter
independently, as if they were unrelated rendering changes.
**Why it happens:** `SourcePane.svelte` currently renders the entire highlighted source as one
`{@html highlightSource(text, language)}` blob inside a single `<pre><code>` element — there is
no per-line DOM today [VERIFIED: web/src/lib/components/browse/SourcePane.svelte — read this
session; both the `file` branch (`<pre class="overflow-x-auto rounded border p-4
text-sm"><code>{@html highlightSource(...)}</code></pre>`) and the `single-def` branch use this
exact one-blob shape]. CONTEXT.md's own "Consequence (for the planner)" note under D-04
already identifies this: both features need per-line elements, so they share ONE rendering
change.
**How to avoid:** Plan the per-line DOM restructuring (`<span data-line>` rows, or a two-column
grid — Claude's discretion per CONTEXT.md) as a SINGLE task/wave that both BRW-10 and BRW-11/12
depend on, not as two independent, possibly-conflicting edits to the same render block.
**Warning signs:** Two plans/waves both rewriting `SourcePane.svelte`'s `<pre>` block
independently, producing a merge conflict or a silently-reverted change.

### Pitfall 4: `wantUIServiceMethods` / `mutatingVerbs` fixtures going stale

**What goes wrong:** Adding `GetEditorLink` to `ui.proto` without updating
`internal/uiserver/readonly_test.go`'s hardcoded method count and set, causing
`TestUIServiceMethodSetIsExactlyTheReadSet` to fail for the WRONG reason (fixture staleness,
not an actual regression) — or worse, silently passing because both sides were edited
sloppily to match each other rather than being independently re-derived.
**Why it happens:** This exact mistake happened once already in this codebase's history: 03-05's
plan text said "change nothing else" but the test's hardcoded method-count literal (9→10) HAD
to change for `GetPermalink` to be addable at all [VERIFIED: STATE.md decisions log, Phase 03
entry: "TestUIServiceMethodSetIsExactlyTheReadSet's hardcoded method-count literal (9->10) was
updated alongside wantUIServiceMethods' new GetPermalink entry"].
**How to avoid:** `wantUIServiceMethods` count goes from 14 to 15
[VERIFIED: internal/uiserver/readonly_test.go:69-83 — literal map has exactly 14 entries as of
this session's read; `if len(got) != 14` at line ~108]; add `"GetEditorLink": {}` to the map and
update every literal length check (`14`, and the doc-comment prose above it) in the SAME
commit as the proto change.
**Warning signs:** `TestUIServiceMethodSetIsExactlyTheReadSet` failing with "observed method
set size 15 != fixture size 14."

## Code Examples

### The confinement gate `GetEditorLink` must call, verbatim

```go
// Source: internal/query/node.go, lines 81-94 (read this session)
// ValidateRepoRelativePath exposes resolveSourcePath's confinement gate to
// callers outside this package, for validation only — it returns just the
// error, discarding the resolved absolute path resolveSourcePath computes
// as a byproduct.
func (e *Engine) ValidateRepoRelativePath(relPath string) error {
	_, err := e.resolveSourcePath(relPath)
	return err
}
```

The underlying gate this delegates to performs, in order (all confirmed this session,
`internal/query/node.go:17-73`): empty-path rejection, `filepath.IsAbs` rejection,
`filepath.Clean`+prefix-check escape rejection, `filepath.Rel`-against-root re-check, THEN
`filepath.EvalSymlinks` on both root and candidate with a re-verified `filepath.Rel` — this is
the two-layer (string-level + symlink-resolved) check D-08 explicitly reuses for the absolute
path it joins onto `Options.RepoPath` after confinement passes.

### The `FileSymbols` shape the breadcrumb consumes

```go
// Source: internal/query/filesymbols.go (read this session, lines 1-95, full file)
const MaxFileSymbols = 2000

type FileSymbolsResult struct {
	Symbols   []*schema.Node
	Total     int64
	Truncated bool
}
```
Each `*schema.Node` carries `StartLine`/`EndLine` (int, 1-based) — the breadcrumb's
"innermost containing range" computation (D-02: smallest span among ranges containing the
current line) operates directly on these two fields, sorted by `StartLine` (ties broken by
`Name`) as `FileSymbols` itself already returns them
[VERIFIED: internal/query/filesymbols.go:80-85 — `sort.SliceStable` block, quoted:
`if matches[i].StartLine != matches[j].StartLine { return matches[i].StartLine <
matches[j].StartLine } return matches[i].Name < matches[j].Name`].

### The per-path fetch/cache pattern to reuse for the breadcrumb's data source

```ts
// Source: web/src/routes/graph/+page.svelte (read this session, lines 59, 82, 466-494)
let fileSymbolsCache = $state<Map<string, FileSymbolsResponse>>(new Map());
let pendingFileSymbols = new Set<string>();
// ... on demand, checking cache then pendingFileSymbols before issuing:
pendingFileSymbols.add(id);
uiClient
	.fileSymbols({ path: id })
	.then((response) => {
		pendingFileSymbols.delete(id);
		const cache = new Map(fileSymbolsCache);
		cache.set(id, response);
		fileSymbolsCache = cache; // new Map instance — Svelte 5 reactivity requires this
	});
```

### The Playwright live-browser gate shape to follow for BRW-10 criterion 1

```js
// Source: web/scripts/graph-collapse-affordance-check.mjs (read this session, lines 1-80)
import { chromium } from '@playwright/test';
// ... launches real Chromium against the built binary's own URL, polls a published
// window.__codegraph* global or DOM data-testid until a condition holds, drives REAL
// mouse/scroll input (never dispatched synthetic events), and writes a diagnostic JSON
// record to corpora/ alongside the script.
```
For BRW-10, the analogous script scrolls the source pane to a known line (via real
`page.mouse.wheel` or `element.scrollIntoView` + `page.waitForTimeout` settle, following
`pollUntil`'s poll-don't-sleep convention already in this file) and asserts the breadcrumb's
`data-testid` element's text content equals the expected symbol name — then, for the RED
demonstration, the SAME script is run against a git-stashed pre-fix checkout to record the
stale/empty failure output the mutation-log convention requires.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `SourcePane` renders one `{@html}` blob with no per-line addressability | Per-line DOM (this phase) | Phase 9 | Both the breadcrumb's line lookup and the gutter's per-line click targets become possible; every prior phase's source rendering assumed line-blindness was fine (permalink anchors were computed from `BrowseTargetState`, never from DOM position) |
| No editor integration exists anywhere in this UI | `GetEditorLink` + presets + discovery | Phase 9 | First external-application handoff surface in the product; first rpc whose response is consumed as a browser-native `href` rather than rendered as text/JSON |

**Deprecated/outdated:** none — this is new capability, not a replacement of an existing one.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Cursor's URI scheme for opening a file at a line is `cursor://file/{path}:{line}` (column support unconfirmed) | Standard Stack / Common Pitfalls (Pitfall 2) | The Cursor preset silently opens the file at line 1 (or fails to open at all) rather than at the intended line; user-visible but not a security issue |
| A2 | JetBrains' URI scheme is `jetbrains://<ide>/navigate/reference?project=<name>&path=<file>:<line>:<col>`, and `path` may need to be project-relative rather than absolute | Standard Stack / Common Pitfalls (Pitfall 2) | The JetBrains preset may prompt for project selection or fail to navigate if the server always fills `{path}` with an absolute path (D-08); this is a genuine, unresolved tension between D-08's design and JetBrains' documented example shape, not just an unverified syntax detail |
| A3 | An async click → `GetEditorLink` rpc → navigate sequence for the line-number gutter stays inside Chromium/Firefox's transient-activation window given local-loopback rpc latency, but is at meaningfully higher risk in Safari/WebKit | Common Pitfalls (Pitfall 1) | If wrong even for Chromium/Firefox, gutter line clicks could silently fail to trigger the external-protocol prompt in the primary dev/CI browser, not just in Safari |
| A4 | Zed has no stable file+line-openable URL scheme as of this research date (2026-09-12), corroborating CONTEXT.md's "Zed nowhere" decision | Common Pitfalls / Standard Stack framing | Low risk — this only affects whether "Zed nowhere" needs re-litigating; multiple independent, dated GitHub issues/discussions confirm the gap, and the decision is already locked in CONTEXT.md regardless |

**If this table is empty:** N/A — see rows above. Every editor-URI-scheme claim beyond VS
Code's officially-documented syntax is `[ASSUMED]`/community-sourced and should be spot-checked
against a real installed IDE during execution, ideally behind a `checkpoint:human-verify` per
the Package-Legitimacy-Gate-adjacent discipline this project applies to unverified external
claims generally.

## Open Questions

1. **Does JetBrains' `path` parameter want an absolute or project-relative path, and does it
   require a `project=` value the server cannot know?**
   - What we know: JetBrains' own community-sourced examples use a path relative to a named
     project (`path=lib/hello.rb`, `project=hello-world`); D-08 mandates the server always fill
     `{path}` with the absolute joined path for every preset uniformly.
   - What's unclear: whether JetBrains' navigate/reference handler tolerates an absolute path
     in `path` (many URI-scheme handlers do accept absolute paths even when their own docs
     example relative ones), and what value (if any) a `project=` parameter should carry when
     the server has no concept of a JetBrains "project name" distinct from the repo directory.
   - Recommendation: treat this as an execution-time empirical spike (install a JetBrains IDE
     or ask the maintainer to test) rather than guessing the exact template string; ship the
     JetBrains preset behind the honest understanding that its template may need a follow-up
     fix, and record the actual working syntax in `09-SUMMARY.md` once confirmed live, per this
     project's own "found live, not assumed" discipline (see multiple STATE.md precedents:
     03-06's Command primitive, 03-07's search-URL bug — both found only by live verification).

2. **Should the line-number gutter's click-to-navigate use `location.href =` or
   `window.open()`, and does it need a synchronous-open-then-redirect trick to survive Safari's
   activation window?**
   - What we know: `location.href =` avoids popup-blocker semantics entirely (it is a
     navigation, not a new window); the "open blank window synchronously, redirect it once the
     async result resolves" pattern is a documented workaround for genuine `window.open()`
     popups but is unnecessary if `location.href` is used instead, and would in any case be an
     awkward UX for a same-tab custom-protocol handoff (a flash of `about:blank`).
   - What's unclear: whether ANY approach reliably survives Safari's stricter timing for a
     genuinely external (non-http) scheme navigation, since research found no authoritative
     source measuring this specific case (custom-scheme navigation, not `window.open` to an
     http URL).
   - Recommendation: use `location.href =` (simplest, no popup semantics to fight), and treat
     "confirmed working in Chromium" as sufficient for v1 given this project's other UI phases
     also gate live-browser proof on a single real browser (Playwright's `chromium`); explicitly
     accept cross-browser (Safari/Firefox) unverified risk in `09-SECURITY.md` rather than
     silently assuming parity.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `@playwright/test` (Chromium) | BRW-10's live-browser gate | ✓ [VERIFIED: web/package.json:22 — exact-pinned `1.62.1`, already installed as of Phase 5/6] | 1.62.1 | — |
| A real VS Code / Cursor / JetBrains IDE installation on the dev/CI machine | Manually confirming preset templates work (Open Question 1, Assumptions A1/A2) | Not probed this session — machine-dependent | — | If unavailable, ship presets as documented-but-unverified and flag via `checkpoint:human-verify`, per the Package Legitimacy Gate's own precedent for unverified external claims |
| `go`/`task build:release` toolchain | Building the real binary the Playwright script drives (`./codegraph ui --no-open`) | ✓ (already required by every prior UI phase's live-browser verification — 05-03-SUMMARY.md, 06-*-SUMMARY.md) | — | — |

**Missing dependencies with no fallback:** none identified.
**Missing dependencies with fallback:** real-editor installations for preset verification (see
row above) — fallback is explicit `[ASSUMED]` tagging plus a human-verify checkpoint, not a
blocked plan.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Go framework | `go test` (stdlib), `task test:unit` [VERIFIED: Taskfile.yml:117 target exists] |
| Frontend unit framework | `vitest` 4.1.11 + `@testing-library/svelte` 5.4.2, jsdom 30.0.1 [VERIFIED: web/package.json:26-31] |
| Frontend live-browser framework | `@playwright/test` 1.62.1, driven via committed `web/scripts/*.mjs` (no `playwright.config.ts` in this repo — scripts launch `chromium` directly) [VERIFIED: `web/scripts/graph-collapse-affordance-check.mjs` read this session; no `playwright.config.*` found in `web/`] |
| Config file | none for Playwright; `web/vite.config.ts` for vitest (per Phase 2's `resolve.conditions:['browser']` fix, guarded on `process.env.VITEST`) |
| Quick run command | `cd web && pnpm test` (vitest, unit-level, seconds); `go test ./internal/uiserver/... ./internal/query/...` (targeted) |
| Full suite command | `task test:unit` (Go) + `task web:test` (frontend unit) + `node web/scripts/breadcrumb-check.mjs` (new, live-browser) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| BRW-10 | Breadcrumb names the innermost containing symbol while scrolling; empty state when in no symbol | unit (jsdom, derivation logic) + live-browser (real scroll, real DOM) | `pnpm test -- source-pane` (unit) + `node web/scripts/breadcrumb-check.mjs` (live) | ❌ Wave 0 — both new |
| BRW-11 | `GetEditorLink` rejects a traversal/absolute/symlink-escaping path with `CodeInvalidArgument` naming only the caller's own relative path | unit (Go, mirrors `TestGetPermalinkPathConfinementAtRPCBoundary`) | `go test ./internal/uiserver/... -run TestGetEditorLink` | ❌ Wave 0 — new `editorlink_test.go` |
| BRW-12 | `--editor-url` sets the default; per-browser override replaces it for that browser only; presets exist for VS Code/Cursor/JetBrains; Zed absent from every preset AND every UI string | unit (Go flag/env/discovery; TS preset-count + Zed-absence check) | `go test ./internal/cli/...` + `pnpm test -- editor-prefs` | ❌ Wave 0 — new |
| BRW-13 | Every named threat carries a test or recorded verdict | manual-only (SECURITY.md authoring), backed by the unit/live tests above | N/A (document, not a runnable check) | ❌ Wave 0 — `09-SECURITY.md` |

### Sampling Rate

- **Per task commit:** `cd web && pnpm test` + targeted `go test ./internal/uiserver/...
  ./internal/cli/... ./internal/query/...`
- **Per wave merge:** `task test:unit` + `task web:test` + `node
  web/scripts/breadcrumb-check.mjs`
- **Phase gate:** Full suite green before `/gsd-verify-work`, PLUS the live-browser Playwright
  run (BRW-10's criterion 1 explicitly requires "verified in a live browser against a real
  index, not only jsdom")

### Wave 0 Gaps

- [ ] `internal/uiserver/editorlink_test.go` — covers BRW-11 (confinement, scheme allowlist,
      reason-string distinctness, closed-enum coverage)
- [ ] `internal/cli/editordiscovery_test.go` — covers BRW-12's precedence (flag → env →
      discovered → unconfigured) and D-17's fail-fast-on-malformed-explicit-value behavior
- [ ] `web/tests/source-pane-breadcrumb.test.ts` (or extend `source-pane.test.ts`) — covers
      BRW-10's derivation logic at the unit level (jsdom)
- [ ] `web/scripts/breadcrumb-check.mjs` — new live-browser Playwright script, following
      `graph-collapse-affordance-check.mjs`'s exact structure; this is the sanctioned mechanism
      for BRW-10's "demonstrated wrong against the pre-fix build" RED requirement too (run
      against a `git stash`'d pre-fix checkout, capture the stale/empty failure, record in
      `09-MUTATION-LOG.md`)
- [ ] Framework install: none — vitest, Playwright, and Go's stdlib testing are all already
      present and already used for the exact same shapes of test in prior phases.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Single-user local tool, no auth surface (unchanged from prior phases) |
| V3 Session Management | no | No session state introduced; `localStorage` override is a client preference, not a session |
| V4 Access Control | no | No access-control boundary changes; loopback-only server, `originHostGuard` already covers the mux |
| V5 Input Validation | **yes** | `(*query.Engine).ValidateRepoRelativePath` (path); a positive scheme allowlist (D-13) for the template value; `{line}`/`{col}` numeric bounds (mirror `GetPermalink`'s `line >= 1` check, `permalink.go:141-142` read this session) |
| V6 Cryptography | no | No cryptographic operation introduced |
| V14 Configuration | **yes** | D-17's fail-fast-on-malformed-startup-value is itself a configuration-security control: a silently-ignored bad `--editor-url` would be a worse failure mode than refusing to start |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Path traversal / symlink escape into `{path}` | Tampering / Information Disclosure | `ValidateRepoRelativePath` — same gate as `GetNodeDetail`/`GetPermalink`/`FileSymbols`; see 03-SECURITY.md T-03-01/T-03-01b as the direct precedent to extend, not reinvent |
| Scheme injection via flag, env, or per-request override (`javascript:`, `data:`, `blob:`, `file:`) | Tampering / Elevation of Privilege | D-13's **positive** scheme allowlist — `javascript:`/`data:`/`blob:`/`file:` refused by construction (not enumerated as a denylist); a template must contain `{path}` or it is `TEMPLATE_INVALID` |
| Absolute host path leaking into an error message | Information Disclosure | Mirror `permalink.go`'s reclassification discipline: the rejected-path error names ONLY the caller's own repo-relative path string, never the resolved absolute path — verified this session at `permalink.go:139-146` |
| The override text field as a self-XSS surface | Tampering | The override is a plain string passed through the SAME server-side validator as the flag value (D-06); it is never rendered as HTML client-side (Svelte's text interpolation escapes by construction, the same property T-03-26 already relies on) |
| Startup-time discovery accidentally launching an editor rather than merely probing for it | Elevation of Privilege | D-15 is explicit: discovery is `stat`/`exec.LookPath` ONLY, never a launch (SRV-03); a probe-vs-launch verdict (not just a test) belongs in `09-SECURITY.md` per D's own BRW-13 note |
| DNS rebinding against the new `GetEditorLink` mux entry | Spoofing | Inherited for free: `GetEditorLink` registers on the SAME Connect handler `originHostGuard` already wraps (`server.go`, `Listen`) — no new mux entry, same accepted-risk disposition as T-03-13/T-03-23 |
| The browser's own external-protocol prompt bypassed or spoofed | Spoofing / Tampering | Out of this phase's control by design — BRW-13 explicitly names this prompt (not CSP) as the actual boundary; the phase's job is to never construct a URL that reaches an unintended scheme, not to control browser chrome |
| Async click→rpc→navigate losing user-activation, silently no-op | (not a security threat, but adjacent) DoS-shaped UX failure | See Common Pitfalls Pitfall 1 — `location.href=`, verified live at minimum in Chromium |

## Sources

### Primary (HIGH confidence — read directly this session)
- `internal/uiproto/uiv1/ui.proto` — full service definition, all 14 existing rpcs' doc comments, `Node`/`Location` shared messages
- `internal/uiserver/permalink.go` — full file — the structural template for `GetEditorLink`
- `internal/query/node.go` (lines 17-105) — `resolveSourcePath`/`ValidateRepoRelativePath`
- `internal/query/filesymbols.go` — full file — `MaxFileSymbols`, `FileSymbolsResult` shape, sort order
- `internal/uiserver/server.go` — full file — `Options`, `Listen`, startup-time-only discipline
- `internal/cli/ui.go` — full file — flag/env precedent (`--no-open`), `shouldOpenBrowser`
- `internal/uiserver/readonly_test.go` — `wantUIServiceMethods`, `mutatingVerbs`, the set-equality fixture pattern
- `web/src/lib/components/browse/SourcePane.svelte` — full file — the single-`{@html}`-blob rendering, `PermalinkClient` interface pattern, `permalinkSurface` snippet
- `web/src/routes/graph/+page.svelte` (relevant sections) — `fileSymbolsCache`/`pendingFileSymbols` pattern
- `web/scripts/graph-collapse-affordance-check.mjs` (partial) — the sanctioned live-browser Playwright gate structure
- `web/package.json` — exact dependency versions
- `.planning/milestones/v0.12.0-phases/03-browse-inspect-navigation/03-SECURITY.md` and `.planning/phases/08-tmux-real-pty-harness/08-SECURITY.md` — SECURITY.md house shape
- `.planning/REQUIREMENTS.md`, `.planning/STATE.md`, `09-CONTEXT.md` — phase scope, locked decisions, project history

### Secondary (MEDIUM confidence — official or cross-checked web sources)
- code.visualstudio.com/docs/editor/command-line — official VS Code doc, `vscode://file/{path}:{line}:{col}` syntax confirmed verbatim
- developer.mozilla.org/en-US/docs/Web/Security/Defenses/User_activation, webkit.org/blog/13862 — official/semi-official transient-activation mechanics
- Multiple cross-referenced GitHub issues confirming Zed has no stable file+line URL scheme as of this research date (zed-industries/zed#8482, discussion #6551)

### Tertiary (LOW confidence — community-sourced, marked for validation)
- forum.cursor.com/t/does-cursor-have-a-unique-open-scheme/3659 — Cursor's `cursor://file/{path}:{line}` syntax, unofficial
- medium.com/@alanhe421 (Understanding JetBrains URL Scheme), github.com/alanhe421/jetbrains-url-schemes — JetBrains `jetbrains://<ide>/navigate/reference?project=&path=` syntax, cross-referenced across two independent community sources but explicitly NOT officially documented per JetBrains' own open YouTrack ticket (TBX-3965)
- sitepoint.com, mai1015.com — community discussion of the async-navigation-loses-activation pattern, consistent with but not a substitute for the MDN/WebKit primary sources

## Metadata

**Confidence breakdown:**
- Standard stack / in-repo mechanics: HIGH — every claim traced to a specific file and line
  range read this session
- Editor URI-scheme syntax (VS Code): MEDIUM-HIGH — official doc, but not independently
  tested against a real VS Code install this session
- Editor URI-scheme syntax (Cursor, JetBrains): LOW — community-sourced only, explicitly
  flagged for empirical confirmation (Assumptions A1/A2, Open Question 1)
- Transient-activation risk: MEDIUM — primary sources (MDN, WebKit) confirm the general
  mechanism; no source specifically measures a local-loopback-RPC-then-custom-scheme-navigation
  sequence, so the specific risk-to-this-phase magnitude is inferred, not measured
- Pitfalls: HIGH for the in-repo pitfalls (single-blob rendering, stale fixtures — both
  verified against actual file content and actual project history), MEDIUM for the
  browser-behavior pitfalls (well-sourced general mechanism, untested specific case)

**Research date:** 2026-09-12
**Valid until:** 30 days for the in-repo mechanics (stable); editor URI-scheme facts should be
re-verified at execution time regardless of elapsed time, since they were never HIGH confidence
to begin with
