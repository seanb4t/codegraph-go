---
phase: 09-source-view-follow-through-breadcrumb-editor-handoff
reviewed: 2026-09-12T19:49:59Z
depth: deep
files_reviewed: 27
files_reviewed_list:
  - corpora/breadcrumb-check.json
  - internal/cli/editordiscovery_test.go
  - internal/cli/editordiscovery.go
  - internal/cli/editorurl_test.go
  - internal/cli/editorurl.go
  - internal/cli/ui_test.go
  - internal/cli/ui.go
  - internal/uiproto/uiv1/ui.proto
  - internal/uiserver/editorlink_test.go
  - internal/uiserver/editorlink.go
  - internal/uiserver/editorpresets.go
  - internal/uiserver/handlers.go
  - internal/uiserver/readonly_test.go
  - internal/uiserver/server_test.go
  - internal/uiserver/server.go
  - web/scripts/breadcrumb-check.mjs
  - web/src/lib/breadcrumb.ts
  - web/src/lib/components/browse/EditorLinkPicker.svelte
  - web/src/lib/components/browse/SourcePane.svelte
  - web/src/lib/editor-prefs.ts
  - web/src/lib/source-lines.ts
  - web/tests/breadcrumb.test.ts
  - web/tests/editor-link-picker.test.ts
  - web/tests/editor-prefs.test.ts
  - web/tests/setup.ts
  - web/tests/source-lines.test.ts
  - web/tests/source-pane-breadcrumb.test.ts
  - web/tests/source-pane-editor-link.test.ts
findings:
  critical: 1
  warning: 4
  info: 0
  total: 5
status: issues_found
---

# Phase 09: Code Review Report

**Reviewed:** 2026-09-12T19:49:59Z
**Depth:** deep
**Files Reviewed:** 27
**Status:** issues_found

## Summary

This phase adds the sticky source-view breadcrumb (BRW-10) and the editor
handoff (BRW-11/BRW-12): a new `GetEditorLink` RPC, its Go-side template
validator/allowlist/discovery machinery, and the SPA's probe/gutter/picker
UI. The Go side is careful and well-tested: the scheme allowlist is
positive-only and asserted at exactly 15 members, path confinement runs
before any substitution, `{path}` is built via `filepath.Join`
(never `EvalSymlinks`-resolved), percent-encoding is scoped per template
position with a dedicated boundary test, discovery is proven probe-only
by DI-based fakes, and `go build`/`go vet`/the targeted Go test subset all
pass cleanly. The frontend correctly isolates the one `{@html}` render
site behind `splitHighlightedLines`'s balanced-tag scan, guards every
`localStorage` call, and never assembles a URL client-side.

The one real functional defect found is in the SPA's editor-link probe
cold-start path: a persisted **preset** override (as opposed to a
**custom** one) is silently dropped on the very first `GetEditorLink`
call of a page load, because the template lookup depends on a preset
list that has not arrived yet. The test suite exercises the pre-seeded
**custom**-override path but never the pre-seeded **preset**-override
path, which is exactly the one that is broken — see CR-01.

## Critical Issues

### CR-01: A persisted preset editor-link override is silently ignored on the first probe of every page load

**File:** `web/src/lib/components/browse/SourcePane.svelte:386, 411-444`
(also `web/src/lib/editor-prefs.ts:86-93`)

**Issue:**

`templateForRequest` resolves a `{kind: 'preset', id}` override by looking
its id up in a `presets` array supplied by the caller:

```ts
// web/src/lib/editor-prefs.ts:86-93
export function templateForRequest(
	override: EditorOverride | null,
	presets: readonly EditorPreset[]
): string | undefined {
	if (!override) return undefined;
	if (override.kind === 'custom') return override.template;
	return presets.find((p) => p.id === override.id)?.template;
}
```

`SourcePane.svelte` supplies that list from `lastPresets`, a `$state`
that starts at `[]` and is only populated *after* the first
`GetEditorLink` response arrives:

```ts
// web/src/lib/components/browse/SourcePane.svelte:386
let lastPresets = $state<EditorPreset[]>([]);
...
// :427 (inside the probe $effect)
const template = templateForRequest(override, untrack(() => lastPresets));
activeClient.getEditorLink({ path, line, col, template }, ...).then((response) => {
	lastPresets = response.presets;
	editorLinkState = { kind: 'loaded', response };
	...
});
```

`editorOverride` is itself seeded from `localStorage` at mount
(`readEditorOverride()`, line 373), i.e. from a *previous* session. On
the very first probe of a fresh page load, if the persisted override is
`{kind: 'preset', id: 'cursor'}` (the common case — the picker's most
prominent affordance is its three preset buttons, not the custom-template
field), `templateForRequest(override, [])` returns `undefined` because
`[].find(...)` cannot match anything. The very first `GetEditorLink`
request is therefore sent with **no** template override at all, so the
server answers using its own effective default (flag/env/discovery/none)
instead of the user's saved preference — silently, with no error and no
visible indication that the override was dropped.

Concretely: a user who saved "Cursor" as their editor and reloads the
page will see the header "Open in editor" link resolve to whatever the
*server's* default happens to be (e.g. a discovered VS Code install, or
no link at all) on that first load, not their saved Cursor preference.
Only the *second* `GetEditorLink` call in the same page session (any
subsequent file navigation) is correct, because by then `lastPresets` has
been populated by the first (wrongly-templated) response.

This is precisely the "assumption-delta invariant" the file's own header
comment states `templateForRequest` exists to prevent
(`editor-prefs.ts:1-12`: "both the load-time probe and a gutter click
call it so they always resolve identically for the same override") — it
holds for `gutterClick` vs. subsequent probes, but is violated by the
very first probe of a session.

The gap is visible in test coverage: `source-pane-editor-link.test.ts`
has a test "sends a pre-seeded custom override as the probe template"
(line 170) but no equivalent test for a pre-seeded **preset** override,
which is exactly the code path that is broken. A `kind: 'custom'`
override is unaffected because `templateForRequest` never needs a
presets list for that branch.

**Fix:** Do not let the very first probe silently drop a pending preset
override. One option: if the resolved `template` differs from what a
non-empty `lastPresets` would have produced once the first response's
`presets` arrive, immediately re-issue the probe with the corrected
template before settling `editorLinkState`:

```ts
activeClient
	.getEditorLink({ path, line, col, template }, { signal: controller.signal })
	.then(async (response) => {
		lastPresets = response.presets;
		if (template === undefined && override?.kind === 'preset') {
			const resolved = templateForRequest(override, response.presets);
			if (resolved !== undefined) {
				const corrected = await activeClient.getEditorLink(
					{ path, line, col, template: resolved },
					{ signal: controller.signal }
				);
				editorLinkState = { kind: 'loaded', response: corrected };
				return;
			}
		}
		editorLinkState = { kind: 'loaded', response };
		...
	})
```

A simpler alternative that avoids a double round-trip: have the server
also report enough about "what is buildable" that the client can pick a
preset id without first knowing its wire template — but that reopens a
design question (D-18's "never hold a second template copy") this plan
already settled, so the corrective re-probe above is the smaller change.
At minimum, add a regression test seeding a **preset** (not custom)
override before mount and asserting the first `getEditorLink` call
carries that preset's template.

## Warnings

### WR-01: `buildEditorURL` assumes a pre-validated template with no defensive guard

**File:** `internal/uiserver/editorlink.go:207-231`

**Issue:** `buildEditorURL` locates a placeholder's closing brace with
`end := strings.IndexByte(template[i:], '}')` and immediately slices
`template[i+1 : i+end]` with no check for `end == -1`. Every current call
site (`editorLinkAnswer`) validates the template via
`ValidateEditorTemplate` first, which does reject unbalanced braces, so
this is not reachable today — but the function's own doc comment only
says the input "is assumed already validated," and nothing in the type
system enforces that. A future call site (or a refactor that reorders
validation) that skips `ValidateEditorTemplate` would panic
(slice bounds out of range) inside a request handler instead of failing
gracefully with `TEMPLATE_INVALID`.

**Fix:** Add a defensive `if end == -1 { break }` (writing the remainder
verbatim, mirroring `source-lines.ts`'s own malformed-input fallback) or
have `buildEditorURL` return `(string, error)` so a validation-invariant
violation degrades to `TEMPLATE_INVALID` rather than panicking:

```go
end := strings.IndexByte(template[i:], '}')
if end == -1 {
	b.WriteString(template[i:])
	break
}
```

### WR-02: No source-level guard that `editordiscovery.go` never calls a process-spawning API

**File:** `internal/cli/editordiscovery_test.go` (whole file); `internal/cli/editordiscovery.go`

**Issue:** `TestDiscoverEditorNeverExecutes` proves `discoverEditorWith`
(the dependency-injected function) only calls the supplied `lookPath`/
`stat` fakes — it does not, and cannot, prove that `editordiscovery.go`
itself contains no `exec.Command`/`os.StartProcess`/`syscall.Exec` call
anywhere in the file (e.g., in a future helper function never routed
through `editorProbes`). A `grep` today shows zero such calls, but unlike
this package's other SRV-03 guards (`TestUIServiceMethodSetIsExactlyTheReadSet`,
`TestUIServiceDeclaresNoMutatingMethod` in `readonly_test.go`, which
reflect over the generated interface — a structural, not merely
behavioral, check), there is no equivalent static assertion here. This
is exactly the class of guard the review scope for this phase called out
by name ("verify the source guard test actually asserts this"), and it
does not yet exist as a source-level check — only a behavioral,
DI-based one.

**Fix:** Add a lightweight source-scan test (e.g. `go/parser` over the
file, or a literal `strings.Contains` scan of the file's bytes) asserting
`internal/cli/editordiscovery.go` contains none of `exec.Command(`,
`os.StartProcess(`, `syscall.Exec(`, matching the "positive, non-vacuous,
fails in both directions" discipline this codebase already applies
elsewhere (rule 84d1gfpywd, cited throughout `readonly_test.go`).

### WR-03: `EditorLinkPicker` self-opens as `role="dialog"` with no focus management

**File:** `web/src/lib/components/browse/EditorLinkPicker.svelte:80-84`; `web/src/lib/components/browse/SourcePane.svelte:388-401, 433-436`

**Issue:** The picker can open itself with no direct user gesture — on
the first `NO_TEMPLATE` (non-disabled) or `TEMPLATE_INVALID` answer to
the load-time probe, `pickerOpenedOnce`/`pickerOpen` flip true
automatically (`SourcePane.svelte:433-436`). The rendered panel declares
`role="dialog" aria-label="Editor link"` but has no `aria-modal`, no
programmatic focus movement into the panel when it opens, and no focus
restoration when it closes (Escape only reassigns `pickerOpen`, it never
calls `.focus()` on the toggle button). A screen-reader user has no
signal that a dialog just appeared, and a keyboard user's focus stays
wherever it was (likely nowhere, since this is a probe-driven auto-open,
not a click).

**Fix:** Either drop `role="dialog"` in favor of a role that matches an
inline, non-modal disclosure panel (e.g. `role="region"`), or — if
`dialog` semantics are intended — move focus into the panel's first
interactive element on open and back to the `editor-link-picker-toggle`
button on close, and add `aria-modal="false"` to make the non-modal
nature explicit to assistive tech.

### WR-04: `CODEGRAPH_NO_EDITOR_URL` parsing runs even when `--no-editor-url` already settled the outcome

**File:** `internal/cli/editorurl.go:92-100`

**Issue:** `resolveEditorLink` always calls `parseBoolEnv` on
`CODEGRAPH_NO_EDITOR_URL` before checking `in.noEditorURL`:

```go
disabled, err := parseBoolEnv(in.getenv(noEditorURLEnvVar))
if err != nil {
	return uiserver.EditorLinkOptions{}, fmt.Errorf("%s: %s", noEditorURLEnvVar, err)
}
if in.noEditorURL || disabled {
	return uiserver.EditorLinkOptions{Source: uiserver.EditorTemplateDisabled}, nil
}
```

If an operator passes the correct, explicit `--no-editor-url` flag but
also has a stray, malformed `CODEGRAPH_NO_EDITOR_URL` value left over in
their shell environment (e.g. `CODEGRAPH_NO_EDITOR_URL=disabled` instead
of `true`), `codegraph ui` refuses to start at all, even though the
operator's explicit, unambiguous flag already fully determines the
outcome. This is a defensible "fail loud on ambiguous config" choice,
but it is not the same choice D-17 documents for the flag/env template
values (where the flag winning is allowed to short-circuit discovery,
just not validation) — worth a one-line comment explaining the choice,
or reordering so `in.noEditorURL` short-circuits the env parse the same
way the malformed-template case is allowed to be pre-empted by which
value actually wins.

**Fix:** Either document why this ordering is intentional (env is
"validated regardless of who wins," mirroring D-17's flag/env template
rule), or swap the order so `if in.noEditorURL { return Disabled }` is
checked before parsing the env var, so an operator's own explicit,
correct flag can never be defeated by an unrelated stray environment
variable.

---

_Reviewed: 2026-09-12T19:49:59Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
