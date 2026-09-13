# Phase 9: Source View Follow-Through — Breadcrumb & Editor Handoff - Context

**Gathered:** 2026-09-12
**Status:** Ready for planning

<domain>
## Phase Boundary

Two additions to the existing `SourcePane.svelte` mount point, plus their threat model:

1. **BRW-10** — a client-side "which symbol am I inside" breadcrumb for a long file's source, computed from the already-shipped `FileSymbols` rpc's line ranges. No proto edit, no Engine change, no CLI flag.
2. **BRW-11 / BRW-12** — an "open in editor" handoff: a `{path}`/`{line}`/`{col}` URI template resolved **server-side** so repo-root confinement is enforced at the RPC boundary; the template's default comes from `codegraph ui --editor-url`, an env var, or startup-time editor discovery, and a per-browser override replaces it for that browser only. Presets for VS Code, Cursor and JetBrains. Zed is neither a preset nor a claimed target.
3. **BRW-13** — `09-SECURITY.md` naming the browser's external-protocol prompt (not CSP) as the security boundary, with every named threat carrying a test or a recorded verdict.

Out of scope by construction (REQUIREMENTS.md → Out of Scope): server-side shell-out to launch an editor (breaks SRV-03); nested scope-stack breadcrumb (BRW-14, v2); a codegraph config file as a template source (deferred below).

</domain>

<decisions>
## Implementation Decisions

### Breadcrumb (BRW-10)

- **D-01:** The breadcrumb is a **single-line sticky bar pinned above the `<pre>`** inside the source pane — it sticks to the viewport top as the *page* scrolls (vertical scroll is the page today, not the pane; the `<pre>` is `overflow-x-auto` only). It is one `data-testid` element. Not the pane header row (scrolls away), not a floating overlay (covers code).
- **D-02:** The "current line" is the **first fully visible source line under the sticky bar**. The crumb names the innermost `FileSymbols` range containing that line (smallest `start_line..end_line` span among those that contain it). Deterministic and assertable in a live browser by scrolling to a known line.
- **D-03:** When the current line is inside **no** symbol (package clause, imports, gaps between functions) the bar **still renders**, showing the **file path only, dimmed**, with an empty symbol segment. Layout never shifts as symbols appear/disappear, and "empty" is a distinguishable, testable state — which success criterion 1's RED demonstration (stale or empty against the pre-fix build) needs. Never the nearest preceding symbol (a lie about containment).
- **D-04:** **File view only** (`target.kind === 'file'`). Single-def view already shows exactly one symbol's source; a crumb there is deferred (see Deferred Ideas).
- **Consequence (for the planner):** the `<pre>` currently renders source as one `{@html highlightSource()}` blob with no per-line DOM. Both the breadcrumb's line→position mapping and D-09's line-number gutter need per-line elements, so the two features share **one** change to how `SourcePane` renders source lines. `MaxFileSymbols = 2000` caps the ranges the crumb can see; how (or whether) to surface that cap is Claude's discretion.

### Editor-link wire shape (BRW-11)

- **D-05:** A **new rpc, `GetEditorLink`** — UIService's 15th method — not an extension of `GetPermalink`. `GetPermalink` runs git introspection with a timeout and its `PermalinkAvailability` enum encodes *remote trust*; an editor link has no remote and no commit. Read-named so it clears `mutatingVerbs`; `wantUIServiceMethods` is updated by set-equality with the method count asserted from both sides (the Phase 3 `GetPermalink` / HLT-06 precedent). Additive proto, no schema bump. — **Reversibility:** costly — the UI proto is under the repo's additive-only field discipline; a shipped rpc and its field numbers are deprecated, never renamed or reused.
- **D-06:** The per-browser override **rides on every request** as an optional template field (`GetEditorLinkRequest.template`, name at Claude's discretion). Absent ⇒ the server's effective default. The server stays **stateless and read-only by construction** — no Set/Put/Save rpc exists or may exist (`mutatingVerbs`), so nothing is persisted server-side and nothing needs resetting. The override is validated on every call exactly like the flag value.
- **D-07:** The response carries a **closed enum** — `EditorLinkAvailability { UNSPECIFIED, BUILDABLE, NO_TEMPLATE, TEMPLATE_INVALID }` (names at Claude's discretion, closed set is not) — plus `url` (populated only when BUILDABLE) and `reason` (populated whenever not BUILDABLE). Configuration states are **answers, never errors**, mirroring `GetPermalink` (D-07 of Phase 3). The **one** error case is a rejected **path**: traversal-shaped, absolute, empty, symlink-escaping or otherwise out-of-root paths return Connect `CodeInvalidArgument` via `(*query.Engine).ValidateRepoRelativePath` (SRV-05), naming **only the caller's own repo-relative path, never the absolute host path** — the `permalink.go` shape. Success criterion 2's test names the rejected input and fails if the request succeeds. The response also carries the **effective default's provenance** (flag / env / discovered-`<preset>` / none / disabled) so the picker (D-18) can show it; exact field shape is Claude's discretion. — **Reversibility:** costly — closed enum on a shipped wire surface; values can be added, never renamed.
- **D-08:** `{path}` is filled with **`filepath.Join(Options.RepoPath, rel)`** — the configured root plus the confined relative path — **after** `ValidateRepoRelativePath` (which already applies both the string-level and the `EvalSymlinks` gate) has accepted it. Not the symlink-resolved real path: editors key on the workspace folder the user opened (`/tmp/x` and `/private/tmp/x` are different workspaces to VS Code), and the resolved form was only ever needed to *prove* confinement. Each placeholder value is percent-encoded per its position so a path cannot smuggle `?`/`#`/`:` into the template (encoding rules: research).

### Handoff affordance

- **D-09:** Two click targets, both in `SourcePane`: an **"Open in editor" link in the pane header** (next to the existing permalink surface; file view opens at line 1, single-def at `node.startLine`/`startCol`) **and a line-number gutter** where clicking a number opens the editor at that line (`{col}` = 1). The gutter's per-line DOM is the same change D-04's consequence names — one rendering change serves both features. This delivers the phase goal's "any node or line".
- **D-10:** The gutter is **always rendered**; line numbers are plain text when the link is not buildable and become links when it is. Layout never depends on editor configuration.
- **D-11:** Editor links appear in **SourcePane only** (file and single-def views — which are exactly BRW-11's "node detail and source views"). **Not** on `NeighborsPanel` rows (N rows ⇒ N rpcs or a batched rpc) and **not** on graph-view nodes (Phase 11 territory).
- **D-12:** Resolution is **per click, on demand**: a click on the header link or a line number issues one `GetEditorLink` for that `(path, line, col, override?)` and then navigates to the returned `url`. **Plus one probe per opened target**: when the pane loads a file/symbol it calls `GetEditorLink` once for `(path, start line)` and keeps the availability + reason — that result makes the header link a real `<a href>` (hover/copy/right-click work), turns the gutter clickable (D-10), and drives the picker's "current state" display. Zero rpcs while reading; line clicks pay one rpc each. — Research must confirm the async rpc→navigate sequence stays inside the browser's transient user-activation window so the external-protocol prompt is not suppressed.

### Template policy, defaults, and the picker (BRW-12)

- **D-13:** The server accepts templates whose scheme is in a **committed allowlist** — the three presets' schemes (`vscode`, `cursor`, the JetBrains family — see research), their siblings (`vscode-insiders`, `vscodium`), and `http`/`https` (web IDEs; JetBrains' localhost REST endpoint). Membership is a **positive** check on the parsed scheme; anything else is `TEMPLATE_INVALID` naming the scheme. `javascript:`, `data:`, `blob:`, `file:` are refused **by construction**, not by a denylist (rule `84d1gfpywd`: a denylist is a negative-only guard). A template must contain `{path}`; `{line}`/`{col}` are optional; unknown placeholders are `TEMPLATE_INVALID`.
- **D-14:** Precedence for the server's effective default: **`--editor-url` flag → `CODEGRAPH_EDITOR_URL` env → discovered default (D-15) → unconfigured** (`NO_TEMPLATE`; the SPA presents the picker on first use). A config *file* is not a source in this phase (deferred).
- **D-15:** **Startup-time editor discovery**, in popularity order: **VS Code (`code`) → Cursor (`cursor`) → JetBrains launchers** (`idea`, `goland`, `webstorm`, `pycharm`, `rider`, `clion`, `phpstorm`, `rubymine` — exact list and order at research time). Probe `PATH` via `exec.LookPath` **and platform app locations** (`/Applications/*.app` on macOS, `%LOCALAPPDATA%\Programs` on Windows; Linux equivalents at research time). First hit selects that editor's preset as the default. Discovery is `stat`/`LookPath` only — it **never launches** anything (SRV-03) — and its failure is **never fatal**: the server starts and reports `NO_TEMPLATE`. Discovery runs once at `Listen`, never per request.
- **D-16:** Off switch: **`--no-editor-url`** (a boolean flag mirroring the existing `--no-open`) or **`CODEGRAPH_NO_EDITOR_URL=true|1|yes`**. When set: discovery is skipped, `GetEditorLink` answers `NO_TEMPLATE` with reason "disabled by operator", the header link is hidden and the gutter stays plain. No magic `none` value.
- **D-17:** A **malformed explicit value** (`--editor-url` or `CODEGRAPH_EDITOR_URL` failing D-13) is a **startup failure**: `codegraph ui` exits non-zero naming the reason before binding. An operator who typed a template meant it; a typo must not silently fall through to "whatever editor we discovered". The same validator is reused per request for override templates — one code path, exercised twice.
- **D-18:** The per-browser override lives in a **popover on the header "Open in editor" link** (a small gear/chevron beside it): the **three presets** (VS Code, Cursor, JetBrains), a **custom template text field** (validated server-side on first use via D-06), and **"Use server default"** (clears the override). It shows which choice is active and the server default's provenance from D-07. Stored in **`localStorage`** — the SPA's first use of it (no existing storage helper; wrap reads/writes in try/catch and render correctly without it). Nothing is added to the global nav. Zed appears nowhere: a check reports the preset count and asserts "Zed"/"zed" is absent from the preset list and from every string the UI ships (success criterion 3).

### BRW-13 — SECURITY.md shape

- `09-SECURITY.md` follows `.planning/milestones/v0.12.0-phases/03-browse-inspect-navigation/03-SECURITY.md`'s shape. It must state that **the browser's external-protocol prompt is the boundary and CSP is not** — CSP does not govern `href` navigation, so the only thing between a malicious template and in-origin script execution is D-13's positive scheme check. Threats to cover at minimum (IDs at Claude's discretion): path traversal / symlink escape into `{path}` (test, per D-07); scheme injection via flag, env or override (test, per D-13); placeholder encoding / template smuggling (test, per D-08); the override text field as a self-XSS surface (verdict); discovery as a read-only probe, never a launch (verdict); the external-protocol prompt as consent boundary (verdict). Every threat carries a test or a recorded verdict — never prose alone.

### Claude's Discretion

- Exact rpc/message/field/enum names, field numbers (additive), and the provenance field's shape (D-07).
- Exact JetBrains preset template (Toolbox `jetbrains://<ide>/navigate/reference?path=…` vs IDE-side `idea://open?file=…&line=…` vs `http://localhost:63342/api/file/…`) and which launcher names discovery probes — settle at research time against current JetBrains docs.
- Percent-encoding rules per placeholder; whether `{col}` defaults to 1 when a template omits it.
- Per-line DOM shape (e.g. `<span data-line>` rows vs a two-column grid) and how line numbers coexist with `highlightSource()`'s HTML output.
- How the breadcrumb tracks scroll (IntersectionObserver vs `scroll` listener + `getBoundingClientRect`), debounce, and whether the crumb is itself a link that scrolls to the symbol's start.
- Whether/how to surface the `MaxFileSymbols` cap in the crumb.
- Probe locations for Linux in D-15; whether `code-insiders`/`codium` are discovery targets (they are not presets).
- Test names, error wording, commit granularity, and which assertions are RED-demonstrated in `09-MUTATION-LOG.md` (at minimum: success criterion 1's stale/empty breadcrumb, and the rejected-path test).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope and requirements
- `.planning/ROADMAP.md` § "Phase 9: Source View Follow-Through — Breadcrumb & Editor Handoff" — goal, four success criteria, the BRW-10-before-BRW-11 ordering, the `GetPermalink`-vs-new-message open question (settled here as D-05), the Zed and shell-out exclusions
- `.planning/REQUIREMENTS.md` — BRW-10…BRW-13 text, the "server flag with a per-browser UI override" scope decision, BRW-14 (v2), and the Out of Scope table (server-side shell-out)
- `.planning/STATE.md` § Accumulated Context → Decisions — Phase 03-05 rulings on `GetPermalink`: closed enum not open string; deleted-file reclassified to `CodeInvalidArgument` naming only the caller's relative path; `wantUIServiceMethods` count literal updated with the fixture

### Wire surface and server
- `internal/uiproto/uiv1/ui.proto` — UIService (14 rpcs today; `GetEditorLink` is the 15th), `GetPermalinkRequest/Response` + `PermalinkAvailability` as the closed-enum precedent, `FileSymbolsRequest/Response`, `Node` line/col fields
- `internal/uiserver/permalink.go` — "an answer, never an error" discipline; `ValidateRepoRelativePath` as the single path gate; reason-string conventions (distinct causes never share a string)
- `internal/uiserver/readonly_test.go` — `wantUIServiceMethods` set-equality fixture and `mutatingVerbs` (forbids Set/Put/Save/Reset/Index…); the positive-count pattern every new guard follows
- `internal/uiserver/server.go` — `Options` struct (`RepoPath`, `Addr`); where the resolved editor template / discovery result and the off switch land
- `internal/uiserver/handlers.go` — rpc registration point
- `internal/query/node.go` — `resolveSourcePath` / `ValidateRepoRelativePath`: string-level Clean/Rel gate then `EvalSymlinks` re-check; the gate D-07 and D-08 reuse
- `internal/query/filesymbols.go` — `MaxFileSymbols = 2000`
- `internal/cli/ui.go` — `codegraph ui` flags (`--path`, `--no-open`) and `uiserver.Listen(uiserver.Options{...})` wiring; `--editor-url` / `--no-editor-url` are added here

### SPA
- `web/src/lib/components/browse/SourcePane.svelte` — the mount point: header row (`CopyAction` + `permalinkSurface`), the single-blob `<pre><code>{@html …}</code></pre>` that gains per-line DOM, the injected `PermalinkClient` interface pattern (mirror it for the editor client so tests can stub it), the documented `$state`-name collision pitfall
- `web/src/lib/browse-state.ts` — `BrowseTargetState` union (`file` / `single-def` / …)
- `web/src/routes/graph/+page.svelte` — `fileSymbolsCache` / `pendingFileSymbols` pattern for calling `uiClient.fileSymbols({ path })` once per path
- `web/tests/source-pane.test.ts` — existing jsdom coverage of the pane; success criterion 1 additionally requires a **live browser** run (Playwright 1.62.1 is already a `web/package.json` dev dependency)

### Guard and security precedents
- `.planning/milestones/v0.12.0-phases/03-browse-inspect-navigation/03-SECURITY.md` — SECURITY.md shape for a browse/inspect phase with a path-confinement threat
- `.planning/phases/08-tmux-real-pty-harness/08-SECURITY.md` — most recent SECURITY.md; verdict-vs-test recording convention
- `.planning/phases/07-guards-that-cannot-fire/07-MUTATION-LOG.md` and `.planning/phases/08-tmux-real-pty-harness/08-MUTATION-LOG.md` — RED-demonstration shape (cleanliness gate, mutation, pasted failing output, byte-clean revert)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `FileSymbols` rpc (shared `Node` message, `start_line`/`end_line`, `MaxFileSymbols=2000`, index-miss ⇒ empty result) — the breadcrumb's only data source; the graph page's per-path cache shows the call pattern.
- `(*query.Engine).ValidateRepoRelativePath` — the one confinement gate; `GetEditorLink` calls it exactly as `GetPermalink` does, then joins the relative path onto `Options.RepoPath` (D-08).
- `SourcePane`'s `PermalinkClient` interface + `permalinkSurface` snippet — the shape for an `EditorLinkClient` prop and an `editorSurface` snippet; keeps the pane testable with a stub client.
- `CopyAction` component in the header row — the editor link and its picker sit alongside it.
- `--no-open` on `codegraph ui` — the boolean-flag precedent `--no-editor-url` mirrors; `CODEGRAPH_*` env vars exist in `internal/watch/debounce.go` and `internal/corpora/manifest.go`.
- `TestUIServiceMethodSetIsExactlyTheReadSet` / `TestUIServiceDeclaresNoMutatingMethod` — the fixture and count to update when the 15th rpc lands.

### Established Patterns
- **Answers, not errors, for configuration states**; errors only for invalid caller input (`permalink.go`). Distinct causes never share a reason string.
- **Closed enums** on the wire, never booleans or open strings, for any tri-state (Phase 3 D-07).
- **Additive-only proto evolution**; `wantUIServiceMethods` updated by set-equality with the count asserted from both sides.
- **Read-only by construction** (SRV-03): no mutating verb in any rpc name; nothing in the server writes state. Editor discovery is a probe, not a launch.
- **Positive assertions in every guard** (rule `84d1gfpywd`): the preset-count check reports a count; the scheme check asserts allowlist membership; RED demonstrations are recorded in a mutation log.
- Svelte 5 runes; `data-testid` on every user-visible state; never name a local `state` in a component (collides with `$state`).
- Startup-time environment facts are frozen at `Listen` (publisher, engine), not re-probed per request — discovery follows.

### Integration Points
- `internal/uiproto/uiv1/ui.proto` → regenerate (`BLD-04` codegen drift guard covers it) → `internal/uiserver/handlers.go` registers `GetEditorLink`.
- `internal/uiserver/server.go` `Options` gains the editor template / disabled state; `internal/cli/ui.go` resolves flag → env → discovery before `Listen` and fails fast on D-17.
- `web/src/lib/components/browse/SourcePane.svelte` — header row (link + picker popover), sticky breadcrumb bar above `<pre>`, per-line rendering with gutter; `web/src/lib/browse-state.ts` if the file target needs to carry `FileSymbols`.
- Generated TS client under `web/src/lib/gen/` picks up the new rpc.
- `09-SECURITY.md` and `09-MUTATION-LOG.md` under this phase directory.

</code_context>

<specifics>
## Specific Ideas

- "Make this easy to start up": the editor link must work out of the box with **no flag** — discover the user's editor at startup, and if nothing is found still start and let the browser pick.
- Precedence stated verbatim: **flag → env → built-in/discovered default**.
- Off switch stated verbatim: **`--no-editor-url` or `CODEGRAPH_NO_EDITOR_URL=true|1|yes`**.
- "A bad/invalid value is a failure, yes" — explicit malformed templates refuse to start.
- The breadcrumb behaves like an editor's sticky scroll / IntelliJ breadcrumb: first fully visible line, innermost symbol.

</specifics>

<deferred>
## Deferred Ideas

- **A codegraph config file as a template source** — no config file exists in the repo today; introducing one is a new capability. Flag → env → discovery covers this phase. Note for the roadmap backlog if a second setting ever needs a file.
- **Editor links on `NeighborsPanel` rows and graph-view nodes** — would need per-row rpc fan-out or a batched rpc; graph-page work belongs with Phase 11.
- **Breadcrumb in single-def view** (offset from `node.startLine` so a long type body names the method you are inside) — natural follow-on to BRW-10, alongside BRW-14's nested scope stack.

### Reviewed Todos (not folded)
- `2026-08-10-brew-trust-instructions-recommend-the-broader-tap-grant-with-no-security-framing.md` — keyword match only; this is DOCS-07, owned by Phase 12.
- `2026-09-08-graphstore-archtest-ignores-per-package-load-errors.md` — keyword match only; a guard-class item with no relation to the source view.

</deferred>

---

*Phase: 09-source-view-follow-through-breadcrumb-editor-handoff*
*Context gathered: 2026-09-12*
