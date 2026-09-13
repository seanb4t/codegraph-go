# Phase 9: Source View Follow-Through — Breadcrumb & Editor Handoff - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-12
**Phase:** 9-source-view-follow-through-breadcrumb-editor-handoff
**Areas discussed:** Breadcrumb placement & current-line rule, Editor-link wire shape, Handoff affordance: link vs per-line, Template policy & unconfigured state

---

## Breadcrumb placement & current-line rule

| Option | Description | Selected |
|--------|-------------|----------|
| Sticky bar pinned above the `<pre>` | Sticks to viewport top as the page scrolls; one `data-testid` element | ✓ |
| In the pane's existing header row | Scrolls away on a long file unless the row is made sticky | |
| Floating overlay in the `<pre>`'s corner | Covers code, fights horizontal scroll | |

**User's choice:** Sticky bar pinned above the `<pre>` (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| First fully visible line under the sticky bar | Matches editor sticky-scroll / IDE breadcrumbs; deterministic; Playwright-assertable | ✓ |
| Line at a fixed fraction (~1/3 down) | Harder to explain and reproduce across viewports | |
| Line under the mouse / last clicked line | Does nothing on keyboard/trackpad scroll | |

**User's choice:** First fully visible line under the sticky bar (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| File path only, dimmed | Bar always renders; stable layout; distinguishable empty state for the RED demo | ✓ |
| Hide the bar entirely | Pop-in shifts code; absent is hard to tell from broken | |
| Show the nearest preceding symbol | A lie about containment | |

**User's choice:** File path only, dimmed (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| File view only | BRW-10 is about long files; single-def already shows one symbol | ✓ |
| Both views | Extra FileSymbols call per opened symbol; line-offset math against a truncated slice | |

**User's choice:** File view only (recommended)
**Notes:** User moved to the next area after four questions.

---

## Editor-link wire shape

| Option | Description | Selected |
|--------|-------------|----------|
| New rpc, e.g. `GetEditorLink` | 15th UIService rpc; independent of git introspection; own closed enum; read-named | ✓ |
| Extend `GetPermalink` with editor fields | Couples to the git-introspection timeout path; stretches a remote-trust enum | |
| Client-side build from a GetStatus-exposed root | Violates BRW-11's server-side resolution requirement | |

**User's choice:** New rpc (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Optional template field on every request | Server stateless and read-only; validated per call like the flag | ✓ |
| Client sends a preset ID, server maps it | Custom overrides have no path to the server | |

**User's choice:** Optional template field on every request (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Closed enum + reason; bad path is a Connect error | Mirrors GetPermalink's answer-not-error discipline; `CodeInvalidArgument` via `ValidateRepoRelativePath` | ✓ |
| url-or-empty plus reason string, no enum | Loses the closed-set property; SPA branches on strings | |
| Everything is an error | Collapses operator config into caller failure | |

**User's choice:** Closed enum + reason; bad path is a Connect error (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Configured root + relative, after confinement passes | Matches the workspace the editor has open; confinement already proven on the resolved form | ✓ |
| EvalSymlinks-resolved real path | `/tmp/x` vs `/private/tmp/x` are different workspaces to VS Code | |

**User's choice:** Configured root + relative (recommended)
**Notes:** User moved to the next area after four questions.

---

## Handoff affordance: link vs per-line

| Option | Description | Selected |
|--------|-------------|----------|
| Header link + click-a-line-number gutter | Per-line DOM shared with the breadcrumb; delivers "any node or line" | ✓ |
| Header link only | Smallest change; "any line" unmet | |
| Header link + breadcrumb-follows-scroll line | Link target silently changes while scrolling | |

**User's choice:** Header link + gutter (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Always rendered; clickable only when buildable | Layout never depends on config | ✓ |
| Only when a template is configured | Source width changes with config | |

**User's choice:** Always rendered (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| SourcePane only — file and single-def views | Exactly BRW-11's "node detail and source views"; no per-row fan-out | ✓ |
| Also on every NeighborsPanel row | N rows ⇒ N rpcs or a batched variant | |
| Also in the graph view's node click | Phase 11 territory | |

**User's choice:** SourcePane only (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Per click, on demand | Zero calls while reading; user-activation window caveat | ✓ |
| Once per view for the header, per click for lines | One extra rpc per opened target; header is a real href | |
| Eagerly for header AND all lines | Client-side template filling — ruled out by BRW-11 | |

**User's choice:** Per click, on demand (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| One probe GetEditorLink per opened file/symbol | Learns buildability up front; header becomes a real href; gutter turns clickable | ✓ |
| Expose `editor_template_configured` on GetStatus | GetStatus grows a per-feature field; override still validated only on click | |
| Drop "only when buildable"; gutter always clickable | A click that opens a settings panel is a surprise | |

**User's choice:** One probe per opened target (recommended)
**Notes:** The probe resolves the conflict between per-click resolution and "clickable only when buildable"; header link ends up a real `<a href>` from the probe result.

---

## Template policy & unconfigured state

| Option | Description | Selected |
|--------|-------------|----------|
| Allowlist of known editor schemes + http(s) | Positive membership check; javascript:/data: refused by construction | ✓ |
| Any scheme except a denylist | Negative-only guard (rule 84d1gfpywd) | |
| Any scheme at all | XSS via the flag or override box | |

**User's choice:** Allowlist (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| At startup — refuse to listen on a bad template | Fail fast; one validator reused per request | |
| Lazily, per request | Typo invisible until first click | |

**User's choice (free text):** "we need to make this easy to startup, so we should do some defaulting if there's no editor-url passed in, or check config"
**Follow-up (free text):** "1 - check path for editors in order of popularity, when none are found startup anyway and present a picker. 2: flag -> env -> built in/discovered default"
**Notes:** No codegraph config file exists today; a config-file source was recorded as a deferred idea. Precedence and discovery captured as D-14/D-15.

| Option | Description | Selected |
|--------|-------------|----------|
| `code` → `cursor` → JetBrains launchers, PATH only | `exec.LookPath` only; installed-but-not-on-PATH gets the picker | |
| Same order, but also probe platform app locations | `/Applications` on macOS, `%LOCALAPPDATA%\Programs` on Windows | ✓ |
| Include code-insiders / codium siblings | Discovery targets only, not presets | |

**User's choice:** Same order, also probe platform app locations

| Option | Description | Selected |
|--------|-------------|----------|
| Popover on the link: 3 presets + custom template + "use server default" | Lives where the feature is; localStorage; shows server default's provenance | ✓ |
| Global settings entry in the layout header | First settings surface for one option | |
| Presets only, no custom text field | Non-preset editors have no per-browser path | |

**User's choice:** Popover on the link (recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| `--editor-url none` disables; bad explicit value refuses to start | Magic literal | |
| No off switch; bad value warns and falls through | Typo hides itself | |
| Off switch, but bad values warn and fall through | Same typo issue | |

**User's choice (free text):** "use '--no-editor-url' or CODEGRAPH_NO_EDITOR_URL=true|1|yes"
**Follow-up (free text):** "a bad/invalid value is a failure, yes"
**Notes:** Boolean off switch mirrors the existing `--no-open` flag; malformed explicit templates fail startup (D-16/D-17).

---

## Claude's Discretion

- Rpc/message/field/enum names and numbers; provenance field shape
- Exact JetBrains preset template and launcher list; Linux probe locations; whether `code-insiders`/`codium` are discovery targets
- Percent-encoding per placeholder; `{col}` default
- Per-line DOM shape and scroll-tracking mechanism; breadcrumb click behaviour; surfacing the `MaxFileSymbols` cap
- Test names, error wording, commit granularity, which assertions are RED-demonstrated

## Deferred Ideas

- A codegraph config file as a template source
- Editor links on NeighborsPanel rows and graph-view nodes
- Breadcrumb in single-def view
