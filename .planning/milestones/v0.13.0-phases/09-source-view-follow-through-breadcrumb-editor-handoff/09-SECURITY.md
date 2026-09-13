---
phase: "09"
slug: "source-view-follow-through-breadcrumb-editor-handoff"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-09-12"
---

# Phase 9 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

**Register origin:** `register_authored_at_plan_time: true` — all five phase plans (09-01
through 09-05) carry a `<threat_model>` block. The register below is the union of every
plan's rows, deduplicated by id. Verification depth is ASVS L1 (grep/execution-level
mitigation presence), which the workflow's short-circuit rule declares sufficient for
`threats_open: 0` at L1.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| browser → `GetEditorLink` | Unauthenticated, client-steerable `path`/`line`/`col`/`template` override on a loopback rpc | repo-relative path, integers, an editor URI template string |
| operator shell → `internal/cli` flags/env | `--editor-url`/`--no-editor-url`/`CODEGRAPH_EDITOR_URL`/`CODEGRAPH_NO_EDITOR_URL` resolved once at startup, before the port binds | an editor URI template string, or an off-switch boolean |
| `GetEditorLinkResponse.url` → browser → OS URI handler | The server-built URL is handed to the browser as a plain `href` or a `location.assign` target; the OS's URI-scheme registry decides what actually launches | a fully-substituted editor URI |
| repository bytes → `highlight.js` → per-line splitter → `{@html}` | Occasionally adversarial third-party source is tokenized by highlight.js, re-split into per-line rows by `splitHighlightedLines`, and rendered as markup | file contents |
| `localStorage` (per-browser) → `GetEditorLinkRequest.template` | A user's own typed or chosen preference is read back and sent as the per-request override on every subsequent probe/click | an editor URI template string |
| `codegraph ui` startup → host filesystem / `PATH` (discovery) | `discoverEditor` probes `exec.LookPath` and platform application directories once, at startup, to find an installed editor | which launcher, if any, is present on the host |

**The boundary is the browser's external-protocol prompt, not CSP.**

CSP does not govern href navigation, so the only thing between a malicious template and an unintended scheme is D-13's positive scheme check on the server.

The browser's external-protocol prompt is the consent boundary; this phase's job is never to construct a URL that reaches an unintended scheme, not to control browser chrome.

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-09-01 | Tampering / Information Disclosure | `GetEditorLink` `{path}` (traversal, absolute, empty, symlink escape) | high | mitigate | `eng.ValidateRepoRelativePath(path)` runs inside `withEngine`, before any template substitution; `TestGetEditorLinkPathConfinementAtRPCBoundary` (`internal/uiserver/editorlink_test.go`) names `../outside.txt`, `/etc/passwd`, `""`, `escape-link/secret.txt` and fails if any request succeeds; watched fail live against the real handler in `09-MUTATION-LOG.md` family (c) | closed |
| T-09-02 | Tampering / Elevation of Privilege | scheme injection via flag, env, per-request override, or a discovered launcher name | high | mitigate | D-13's positive allowlist `editorURLSchemes` (15 members) checked on the parsed, lower-cased scheme; `TestEditorTemplateSchemeAllowlist` asserts the exact member count and preset membership; the same `ValidateEditorTemplate` runs at CLI startup (D-17), per request, and for launcher-composed templates (09-02); watched fail live in `09-MUTATION-LOG.md` family (d) | closed |
| T-09-03 | Information Disclosure | refusal/reason strings leaking the absolute host path | medium | mitigate | every `CodeInvalidArgument` is built from the caller's own relative path (permalink.go's reclassification discipline); `TestGetEditorLinkPathConfinementAtRPCBoundary` (editorlink_test.go:118,139) asserts the error text contains neither `filepath.Abs(dir)` nor its `EvalSymlinks` form | closed |
| T-09-04 | Tampering | placeholder smuggling — a filename carrying a space, `#`, `?`, `:`, or `&` into the template | medium | mitigate | per-position percent-encoding in `encodeEditorPathSegments`; `TestEditorLinkPathEncodingPerPosition` | closed |
| T-09-05 | Tampering (XSS) | `splitHighlightedLines` feeding `{@html}` | high | mitigate | the splitter scans only the literal `<span`/`</span>` tokens highlight.js emits and never un-escapes; `web/tests/source-lines.test.ts#never un-escapes text` asserts `&lt;script&gt;` survives verbatim and every line stays balanced | closed |
| T-09-06 | Tampering (self-XSS) | the custom-template input rendered back into the picker | low | mitigate | rendered only through Svelte text interpolation (escaped by construction, the same property T-03-26 relies on); `{@html` stays confined to `SourcePane.svelte` and is never fed the override; `web/tests/editor-link-picker.test.ts#shows the custom template in the input with no preset pressed, for a custom override` proves the value round-trips as text, never markup; the value is also validated server-side on first use (D-06) | closed |
| T-09-07 | Elevation of Privilege | `discoverEditor` launching an editor rather than merely probing for it | high | mitigate | `exec.LookPath`/`os.Stat` only; a source guard asserts zero `exec.Command`/`exec.CommandContext`/`os.StartProcess` in `editordiscovery.go` with a positive `exec.LookPath` count; `TestDiscoverEditorNeverExecutes` runs discovery against a recording fake and asserts only lookPath/stat calls were made | closed |
| T-09-08 | Spoofing | DNS rebinding against the new `GetEditorLink` mux entry | low | accept | Verdict: inherited — `GetEditorLink` registers on the SAME Connect handler `originHostGuard` already wraps; no new mux entry (03-SECURITY.md T-03-13 disposition) | closed (accepted) |
| T-09-09 | Spoofing / Tampering | the browser's external-protocol prompt bypassed or spoofed | medium | transfer | Verdict: out of this server's control by design — the header link is a native `<a href>` so the browser's own prompt governs it, and the gutter navigates with `location.assign` after one loopback rpc, Chromium-observed issuing that rpc in the live gate (`corpora/breadcrumb-check.json`, `gutterClickIssuedRpc:true`); Safari/WebKit is unverified — see Notes | closed |
| T-09-10 | Denial of Service | oversized or pathological template text | low | mitigate | `EditorTemplateMaxBytes = 2048` and a linear placeholder scan with no regex backtracking; `TestEditorTemplateSchemeAllowlist/too_long` exercises the 2049-byte case | closed |
| T-09-11 | Repudiation | the pasted RED transcripts in `09-MUTATION-LOG.md` | medium | mitigate | Verdict: each family names the exact command and pastes verbatim output; a non-failing demonstration is reported as an observed negative result with its root cause, never rewritten until it does (08-MUTATION-LOG.md family (d) precedent) — exercised for real in this phase's own family (d), where a `t.Fatalf` short-circuit was recorded honestly rather than papered over | closed |
| T-09-13 | Information Disclosure | the override persisting in `localStorage` | low | accept | Verdict: it holds a template string the user typed or chose, never a repository path, and never leaves the loopback origin except as the request's own template field | closed (accepted) |
| T-09-14 | Tampering | a malformed operator template silently replaced by a discovered default | medium | mitigate | D-17: `resolveEditorLink` validates both explicit values before consulting any other source; `TestUICommandRefusesMalformedEditorURLBeforeBinding` asserts no URL is printed and the process exits non-zero before `uiserver.Listen` binds | closed |
| T-09-15 | Tampering | a launcher name flowing into a template scheme without passing the allowlist | medium | mitigate | `templateForLauncher` only emits schemes from the committed ten-launcher table, all members of `editorURLSchemes`; `TestTemplateForLauncherEmitsAllowlistedTemplates` validates every emitted template and reports `inspected == 10` | closed |
| T-09-16 | Information Disclosure | discovery revealing which editors a host has installed | low | accept | Verdict: the only consumer is the loopback SPA on the same machine; the response names a launcher, never a filesystem path; the same-origin guard is unchanged | closed (accepted) |
| T-09-17 | Denial of Service | a very long file producing thousands of rows and a per-scroll computation | low | mitigate | Verdict: the current line is O(1) geometry (`firstFullyVisibleLine`), throttled to one requestAnimationFrame per burst (`SourcePane.svelte:288,314`, T-09-17 named in the source comment), no per-row rect reads; source is already truncated server-side (`SourceBlob.truncated`) | closed |
| T-09-18 | Repudiation | the live-gate record and the mutation-log RED transcript | medium | mitigate | Verdict: `breadcrumb-check.mjs` writes its diagnostic record on every exit path via try/finally; every family in `09-MUTATION-LOG.md` pastes the failing record or test transcript verbatim and names the pre-fix commit or the mutated file | closed |
| T-09-19 | Tampering | `web/build/` drifting from `web/src` after an SPA change | medium | mitigate | Verdict: `task web:build` then `task web:drift` compares independently-computed source-tree and output-tree digests; re-run in this plan's own Task 3 phase-close gate | closed |
| T-09-20 | Denial of Service (UX) | a click burst issuing many rpcs | low | mitigate | a single in-flight guard (`gutterClickInFlight`) shared across every gutter cell — one rpc per click, never a burst; `web/tests/source-pane-editor-link.test.ts#BUILDABLE gutter cells are buttons; a click issues one rpc then navigates via location.assign, a burst click is ignored` | closed |
| T-09-21 | Tampering | a mutation surviving into HEAD (`breadcrumb.ts`, `web/build`, `editorlink.go`) | high | mitigate | pre- and post-gates per family, shared-file mutations strictly one at a time, `09-MUTATION-LOG.md`'s Closing section re-checks `git diff --quiet` on all three paths and greps the restored `ValidateRepoRelativePath` call and the absent `"javascript"` key at HEAD; ongoing regression protection is inherited from the SAME shipped tests those mutations targeted — `TestGetEditorLinkPathConfinementAtRPCBoundary`, `TestEditorTemplateSchemeAllowlist`, and `web/tests/breadcrumb.test.ts` would fail on any future recurrence | closed |
| T-09-22 | Repudiation | a SECURITY.md register citing tests that do not exist | medium | mitigate | Verdict: this document's own authoring pass resolved every cited `Test…` name to a real `func Test…(` definition via `rg`, checked at HEAD, before being written down | closed |
| T-09-SC | Tampering | npm/pip/cargo installs | low | accept | Verdict: no package installed by this phase (09-RESEARCH.md Package Legitimacy Audit: N/A); the gate is not triggered | closed (accepted) |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above `high` count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-09-01 | T-09-08 | `originHostGuard` already wraps the whole mux; `GetEditorLink` adds no new mux entry, so no new DNS-rebinding surface | plan 09-01 | 2026-09-12 |
| R-09-02 | T-09-09 | Out of the server's control by design — BRW-13 names the browser's external-protocol prompt, not CSP, as the boundary; the phase's job is never to construct a URL for an unintended scheme, not to control browser chrome | plan 09-01 / 09-04 | 2026-09-12 |
| R-09-03 | T-09-13 | The override holds a template string the user typed or chose, never a repository path, and is scoped to the loopback origin | plan 09-04 | 2026-09-12 |
| R-09-04 | T-09-16 | Discovery's only consumer is the loopback SPA on the same machine; a launcher name is not sensitive information | plan 09-02 | 2026-09-12 |
| R-09-05 | T-09-SC | No package is installed by this phase (09-RESEARCH.md Package Legitimacy Audit: N/A) | plans 09-01..09-05 | 2026-09-12 |

---

## Notes

**1. Assumptions carried — Cursor and JetBrains preset templates (09-RESEARCH.md A1/A2).**
Both remain `[ASSUMED]`: **not tested — remains [ASSUMED]**. No JetBrains or Cursor
installation was available to click through and visually confirm correct navigation (right
file, right line, no project-selection prompt) during this plan's execution session — this
repo's own `[ASSUMED]` tag in `editorpresets.go` and the picker's visible unverified note
(`EditorLinkPicker.svelte`'s `needsAssumedNote`) are therefore still accurate, and
`.planning/WINDOWS.md` entry 35 stays **open** rather than being closed on an unperformed
check. (A JetBrains Toolbox install (`idea`, `goland`, `pycharm` scripts) is present on this
host's `PATH`, but its presence only means `discoverEditor` would select it — it does not
constitute the visual click-through confirmation the human-check requires.) The `idea://open?
file={path}&line={line}` form (IDE-side, not the Toolbox `navigate/reference?project=…` form)
was chosen specifically because JetBrains' Toolbox form wants a project-relative path
GetEditorLink does not have, while the IDE-side absolute-path form is corroborated by
spatie/ignition's own shipped, widely-used editor-link table, which uses the same
absolute-path `{path}`/`{line}` shape D-08 produces.

**2. Transient user activation (09-RESEARCH.md Pitfall 1, Assumption A3, Open Question 2).**
The header "Open in editor" link is a plain, already-resolved `<a href>` (resolved at
load-time via the probe, D-12) — ordinary browser-native navigation with zero
transient-activation risk. The line-number gutter's click-then-rpc-then-navigate sequence was
observed, in a real Chromium session against this repository's own index, to issue the
`GetEditorLink` rpc and then call `location.assign` within the click's activation window
(`corpora/breadcrumb-check.json`: `gutterClickIssuedRpc: true`). **Safari/WebKit and Firefox
are UNVERIFIED** for this specific sequence — recorded honestly, not silently assumed at
parity with the tested Chromium behavior, per research's own Pitfall 1 finding that Safari is
measurably stricter about consuming the activation window during async work.

**3. Server-side shell-out is out of scope by construction (SRV-03).** Restated as a verdict,
not merely a plan-time decision: nothing in `internal/uiserver` or `internal/cli`'s editor
discovery/resolution path ever spawns, execs, or otherwise launches an external process to
open an editor. `discoverEditor` is `exec.LookPath`/`os.Stat` only (T-09-07); the actual
launch is delegated entirely to the browser's own external-protocol handling (the Trust
Boundaries table's own framing of this phase's one true security boundary).

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-09-12 | 22 | 22 | 0 | plan 09-05 (ASVS L1 inline; auditor short-circuited per `threats_open:0` + `register_authored_at_plan_time:true` + `asvs_level:1`) |

**Audit note — what was checked by execution vs. by reading.** All five `high`-severity rows
(T-09-01, T-09-02, T-09-05, T-09-07, T-09-21) were checked by **execution**: their cited Go
tests were re-run at HEAD during this plan (`TestGetEditorLinkPathConfinementAtRPCBoundary`
and `TestEditorTemplateSchemeAllowlist` both green after Task 1's reverts;
`TestDiscoverEditorNeverExecutes` and `TestEditorLinkPathEncodingPerPosition` confirmed
present and passing in the existing suite). Every `Test…`-cited name in this register (6
distinct Go test functions) was independently resolved to a real `func Test…(` definition via
`rg` against `internal/` and `web/` at HEAD — not trusted from any plan's prose. Every
`.test.ts`-cited vitest file was confirmed present under `web/tests/`. The `medium`/`low`
rows disposed `accept`/`transfer` (T-09-08, T-09-09, T-09-13, T-09-16, T-09-SC) were checked
by **reading** the cited source/design property, consistent with 08-SECURITY.md's own
audit-note precedent for accepted risks that inherit an existing, already-verified guard.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-09-12

**Outstanding, not security-blocking:**
- `.planning/WINDOWS.md` entry 35 — Cursor/JetBrains preset templates remain `[ASSUMED]`,
  pending a real human click-through against an installed IDE (Notes item 1 above).
- Safari/WebKit transient-activation behavior for the gutter's click→rpc→navigate sequence is
  unverified (Notes item 2 above) — Chromium-only live-gate coverage is the current state.
