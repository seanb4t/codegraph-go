---
phase: 03-browse-inspect-navigation
reviewed: 2026-08-29T14:46:52Z
depth: deep
reviewed_at_head: 03240916
diff_base_full_phase: f2c0bcda
diff_base_fix_pass_2: 27f6fff5
files_reviewed: 27
files_reviewed_list:
  - internal/gitmeta/permalink.go
  - internal/gitmeta/permalink_test.go
  - internal/indexer/commit.go
  - internal/indexer/commit_test.go
  - internal/schema/meta.go
  - internal/uiserver/handlers.go
  - internal/uiserver/handlers_test.go
  - internal/uiserver/permalink.go
  - internal/uiserver/permalink_test.go
  - internal/upgrade/golangci_shape_test.go
  - internal/upgrade/taskfile_shape_test.go
  - test/wireoracle/capture.go
  - test/wireoracle/normalize.go
  - web/src/lib/browse-nav.ts
  - web/src/lib/components/browse/CopyAction.svelte
  - web/src/lib/components/browse/NeighborsPanel.svelte
  - web/src/lib/components/browse/SearchPanel.svelte
  - web/src/lib/status.ts
  - web/tests/browse-nav.test.ts
  - web/tests/browse-page.test.ts
  - web/tests/neighbors-panel.test.ts
  - web/tests/search-panel.test.ts
  - web/tests/source-pane.test.ts
  - web/tests/status.test.ts
  - web/src/routes/browse/+page.svelte
  - web/src/routes/+layout.svelte
  - web/src/lib/search.ts
findings:
  critical: 0
  warning: 3
  info: 5
  total: 8
status: issues_found
---

# Phase 3: Code Review Report (final verification re-review)

**Reviewed:** 2026-08-29T14:46:52Z
**Depth:** deep
**HEAD:** `03240916`
**Files Reviewed:** 27 (24 changed by fix pass 2 excluding `web/build/` and `.planning/`, plus 3 integration call sites traced across module boundaries)
**Status:** issues_found

## Summary

This is the terminal review for the phase. No fix pass follows, so every finding
below is stated as "what ships".

**Verdict: no Critical findings survive.** The five fix-pass-1 regressions
(WR-01..WR-05) are genuinely closed, not moved. I re-derived each mechanically
rather than trusting `03-REVIEW-FIX.md`:

- **WR-01** — `SearchPanel.svelte:133-137` tracks `initialQuery` again and skips
  only when it already equals `untrack(() => searchState.query)`. Both directions
  verified against real wiring: `handleInputChange` (`:178-182`) calls
  `controller.setQuery` **before** `onQueryChange`, and `search.ts:207-208`
  updates the store synchronously, so a typing echo is a genuine no-op; the input
  is `value={searchState.query}` (`:201`), so an externally-driven reseed does
  reach the DOM. The regression-pinning test was **corrected, not deleted or
  weakened** — `web/tests/search-panel.test.ts:323-379` now fires a real input
  event before the rerender and pins the debounce deadline at t=250 (a re-seed
  would push it to t=270), and a second test pins the Back/Forward direction. I
  also traced `applyDelta` (`browse-nav.ts:98-114`): it spreads the current
  params, so `q` survives every NAVIGATE/REFINE delta — the fix therefore cannot
  clear the search box on a result click, which was the most plausible
  regression shape here.
- **WR-02** — `permalink.go:38-46,178` introduces `malformedCommitSHAReason`, a
  distinct constant. Distinguishable on the wire and the tests tell them apart
  non-tautologically: `permalink_test.go:293` now asserts the absent-field case
  equals `noCommitSHAReason` exactly, and `:405-411` asserts the malformed case
  equals `malformedCommitSHAReason` **and** does not equal `noCommitSHAReason`.
  Collapsing the two would fail both.
- **WR-03** — coupling accepted per orchestrator. I confirmed the guard runs and
  passes (`TestWorkflowFilePopulationMatchesDisk` PASS at HEAD) and that
  `requiredCheckNames` was untouched. See WR-02 below for the residual gap.
- **WR-04** — `status.ts:72,83` excludes `q` via `VIEW_LOCAL_PARAMS`. The fix does
  not stop both directions: `navigationIdentity` is consumed at exactly two sites
  (`+layout.svelte:21,48`) and nothing else in `web/src`, and the paired tests
  (`status.test.ts:111-127,185-201`) assert the `q` case collapses **and** that a
  `symbol` change still mints a new identity and still fires a second
  `GetStatus`.
- **WR-05** — coupling accepted per orchestrator; both `TestGolangciEnabledLinters
  HeaderMatchesConfig` and `...Discriminates` PASS at HEAD and both route through
  `compareEnabledLintersHeader`.

**Wire shape is intact.** `git diff --name-only 27f6fff5..HEAD` touches no
`.proto`, no `*.pb.go`, no `*.connect.go`, no `web/src/lib/gen/**`. The four
`PermalinkAvailability` members and three `RemotePresence` states are unchanged;
WR-02 was expressed entirely in the `reason` string, which is the additive-safe
lever D-07 already established.

**No new information-disclosure path.** `internal/query/**` is untouched by both
fix passes. The only new wire strings are two package-level constants with no
interpolation. IN-04's change to `internal/gitmeta/permalink.go` strictly
*reduces* disclosure: a Windows-style local remote that previously surfaced as
`origin remote host "D" is not github.com` now surfaces as `origin remote is not
a recognized URL`, with no fragment of the local path echoed. I checked the other
`remote.Reason` producers and the scheme branch (`u.Hostname()` excludes userinfo
and port), and found no path that puts a host filesystem path on the wire.

**The committed bundle is current** — I positive-controlled it rather than
trusting `web:drift`: the built layout chunk contains the WR-04 `q` deletion
(`nodes/0.ggKEIXcw.js`: `var A=[\`q\`]; ... for(let e of A)t.delete(e)`) and the
browse chunk contains IN-09's `.catch(()=>{})`.

**What I found instead** are three Warnings, all introduced or left in place by
fix pass 2. One of them is a *third* instance of this phase's recurring pattern:
a guard that asserts the mechanism the fixer added rather than the symptom the
finding named. None is a correctness or security defect, and none causes wrong
data to be computed, stored, or transmitted.

## Warnings

### WR-01: IN-08's "visible feedback" is invisible — `aria-invalid` on an unstyled bare input, with a guard that asserts the attribute instead of the symptom

**Files:**
`web/src/lib/components/browse/NeighborsPanel.svelte:146-155`,
`web/tests/neighbors-panel.test.ts:181-195`

**Issue:** IN-08's finding was, verbatim, "a non-conforming committed depth value
had no visible feedback". The fix adds `aria-invalid={depthInvalid}` to the depth
control. That control is a **raw `<input type="number">` with no `class`
attribute at all** (`NeighborsPanel.svelte:147-155`) — it is not the vendored
`Input` component and it carries no `data-slot`.

Every compiled `aria-invalid` selector in the shipped stylesheet requires one of
those two things. Verified directly against the committed output
(`web/build/_app/immutable/assets/0.Z5wzVqLj.css`):

```
.aria-invalid\:border-destructive[aria-invalid=true]
.aria-invalid\:ring-3[aria-invalid=true]
.has-\[\[data-slot\]\[aria-invalid\=true\]\]\:border-destructive:has([data-slot][aria-invalid=true])
```

All of them are class-gated. The bare input matches none. There is also no global
`[aria-invalid]` rule anywhere outside `web/src/lib/components/ui/**` (confirmed
by search across `web/src`).

**Concrete failure scenario:** open `/browse?symbol=Foo&depth=2`. Type `2.5` into
the Depth box and tab away. `handleDepthChange` (`:79-91`) sees `raw="2.5"`,
`isShapeInteger` returns false, sets `depthInvalid = true`, and returns without
navigating. Result for a sighted user: no border change, no ring, no message, the
box still displays `2.5`, and the URL still says `depth=2`. That is *exactly* the
pre-fix behaviour the finding described. Only assistive technology is informed.

**Why this matters beyond the a11y wart:** `neighbors-panel.test.ts:189` asserts
`expect(input).toHaveAttribute('aria-invalid', 'true')`. That guard passes while
the user-facing symptom is unchanged — the same decorative-guard shape that
produced WR-03 and WR-05 in the previous pass, one level further out.

**Fix:**

```svelte
<input
  type="number"
  min="0"
  class="aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20"
  value={depth ?? ''}
  onchange={handleDepthChange}
  data-testid="neighbors-depth-input"
  aria-invalid={depthInvalid}
/>
{#if depthInvalid}
  <p class="text-sm text-destructive" data-testid="neighbors-depth-invalid">
    Depth must be a whole number — the previous depth is still in effect.
  </p>
{/if}
```

and assert the rendered message (the symptom), not only the attribute (the
mechanism), in the test.

### WR-02: `workflowFileExceptions` reasons state machine-checkable facts that nothing checks — weaker than the validator it claims to mirror

**Files:**
`internal/upgrade/taskfile_shape_test.go:1528-1601` (the exception list),
`:1603-1620` (`validateWorkflowFileExceptions`)

**Issue:** `validateWorkflowFileExceptions` checks exactly two things: the reason
is non-blank, and the named file still exists on disk. Its own doc comment claims
it is "mirroring `validateRunBodyExceptions`/`validateUsesOnlyJobExceptions`'s
own ... discipline". That claim is false for the property that matters:
`validateUsesOnlyJobExceptions` (`:1686-1712`) actually **parses the workflow and
fails if the excepted job has gained a `run:` step**, with an explicit message
telling the author to move it into `inScopeJobs`. The new file-level validator
does no such thing.

Three entries state a property that is trivially checkable with the
`workflowFileYAML` parser already in this file:

- `auto-close-unsolicited-prs.yml` — "its only step is `uses: actions/github-script`, no run: body at all"
- `auto-label-issues.yml` — same
- `close-draft-prs.yml` — same

I confirmed all three are true today (each file contains exactly one `uses:` line
and zero `run:` lines), so **nothing violating ships**. I also enumerated the two
canary exceptions and found `darwin-toolchain-canary.yml` has one `run:` step and
`linux-cross-canary.yml` has five, and every one of the six is a bare
`task <target>` call — so those exceptions are conservative, not concealing a
live D-01 violation either.

**Concrete failure scenario (future drift, which is the entire purpose of this
guard):** a maintainer adds `- name: Install deps / run: go build ./...` to
`auto-label-issues.yml`. `TestWorkflowFilePopulationMatchesDisk` still passes —
the file exists and the reason string is non-empty. `TestWorkflowRunBodiesInvoke
Task` never sees it, because the file is not in `inScopeWorkflowFiles`. The
exception's recorded justification is now false, and the D-01 raw-invocation the
whole file exists to catch ships green. This is the same "a hand-enumerated
population narrows silently because a new subject passes by being absent" shape
WR-03 was raised for, displaced from the population to the population's
carve-outs.

**Fix:** give the file-level validator the same teeth as its sibling — add a
machine-checkable discriminant to the exception record rather than relying on
prose:

```go
type workflowFileException struct {
	Workflow string
	Reason   string
	// UsesOnly asserts the file declares ZERO run: steps. When true,
	// validateWorkflowFileExceptions parses the workflow and fails if
	// any job has gained one, exactly as validateUsesOnlyJobExceptions
	// does one level down.
	UsesOnly bool
}
```

set `UsesOnly: true` on the three github-script entries, and in
`validateWorkflowFileExceptions` parse the file and return an error naming the
offending step when `exc.UsesOnly` and any `strings.TrimSpace(step.Run) != ""`.

### WR-03: the ONE URL writer swallows every navigation rejection with no diagnostic, justified by a comment citing a module that does not exist and a precedent that does the opposite

**File:** `web/src/lib/browse-nav.ts:128-142`

**Issue:** IN-09's fix is `void gotoFn(url, {...}).catch(() => {})`. The catch
handler is empty: a rejected navigation is discarded with no log, no state
change, and no dev-mode signal, in the module its own header calls "the ONE URL
writer for the Browse view and every phase after it".

The justification comment ends: *"it is exactly the same 'expected, ignorable
failure' shape as browse-tracer.ts's own hljs fallback."* Both halves of that
sentence are wrong, and I verified both:

1. **`browse-tracer.ts` does not exist.** `web/src/lib/` contains
   `browse-nav.ts`, `browse-state.ts`, `browse-url.ts`, `call-targets.ts`,
   `client.ts`, `highlight.ts`, `rpc-errors.ts`, `search.ts`, `status.ts`,
   `utils.ts` — and no `browse-tracer.ts`. Only `web/tests/browse-tracer.test.ts`
   exists, and it is a test of `browse-state.ts` + `highlight.ts`.
2. **The actual hljs fallback does the opposite of what the comment claims.**
   `highlight.ts:168-176` does not swallow anything — it emits
   `console.warn('highlightSource: unregistered language identifier ...')`, and
   its own doc comment at `:161` says the branch "should be unreachable in a
   correct tree and **loud (a console warning)** when it is reached."

So the one precedent invoked to license silence is a precedent for noise. Per
`.claude/CLAUDE.md`'s lint-and-errors rule ("MUST surface problems clearly, never
hide them"), this is the wrong disposition and the wrong citation.

**Concrete failure scenario (latent, not currently reachable):** `goto` rejects
when invoked during SSR and for a URL SvelteKit's router will not handle. If a
later phase mounts a navigator on a server-rendered path, or a delta ever
produces a URL the router rejects, the user clicks a caller and *nothing happens*
— no navigation, no error state, no console entry, and no test can catch it
because the rejection is consumed. I could not construct a reachable rejection at
HEAD (`navigate()` is only called from client-side event handlers in
`+page.svelte:135-160`), which is why this is a Warning and not a Blocker.

**Note on the guard:** `browse-nav.test.ts:177-201` proves `.catch()` is *called*
via a fake thenable. That is honest coupling for the mechanism, and I confirmed
it discriminates. It does not, and cannot, assert that a real rejection is
surfaced — because nothing surfaces it.

**Fix:**

```ts
void gotoFn(url, {
	replaceState: intent === NAV_INTENT.REFINE,
	noScroll: true,
	keepFocus: true
}).catch((err: unknown) => {
	// Match highlight.ts's own discipline for an expected-but-not-silent
	// failure: this branch should be unreachable in a correct tree, so make
	// it loud rather than invisible.
	console.warn('browse-nav: navigation to', url.href, 'was rejected:', err);
});
```

and correct the comment — delete the `browse-tracer.ts` citation or repoint it at
`highlight.ts:168-176`, whose behaviour the fix would then actually match.

## Info

### IN-01: the workflow-file population guard is blind to `.yaml` workflows

**File:** `internal/upgrade/taskfile_shape_test.go:1633-1640`

`TestWorkflowFilePopulationMatchesDisk` filters with
`strings.HasSuffix(e.Name(), ".yml")`. GitHub Actions accepts `.yaml` equally. A
workflow added as `security-scan.yaml` would be invisible to the new guard for
the same "absent, therefore passing" reason WR-03 named. Currently benign: all 14
files on disk are `.yml` and `ls .github/workflows/*.yaml` matches zero. This
also matches the pre-existing filter at `:756`, so it is a consistency issue
rather than a regression — fixing one without the other would be worse.

**Fix:** `if e.IsDir() || (!strings.HasSuffix(e.Name(), ".yml") && !strings.HasSuffix(e.Name(), ".yaml"))`, applied at both `:756` and `:1635`.

### IN-02: IN-04's path-separator clause is unreachable, and no test exercises it

**Files:** `internal/gitmeta/permalink.go:169`,
`internal/gitmeta/permalink_test.go:63-85`

The new guard is `len(candidateHost) > 1 && !strings.ContainsAny(candidateHost, "\\/")`.
The enclosing condition is `slash == -1 || colon < slash`, which already
guarantees `rest[:colon]` contains no `/` — so the `/` half of `ContainsAny` can
never fire. Separately, all three negative fixtures (`D:\src\myrepo`,
`C:/src/myrepo`, `c:\Users\me\repo`) are rejected by the **length** check alone,
because each candidate host is a single character. The `ContainsAny` clause
therefore has no discriminating test at all: deleting it entirely leaves the
suite green.

**Fix:** either drop the `/` from the character set and add a fixture with a
multi-character backslash-bearing candidate (e.g. `sub\dir:repo`), or drop the
`ContainsAny` clause and document that the colon-before-slash rule already
excludes `/`.

### IN-03: `depthInvalid` is never cleared when the `depth` prop changes externally

**File:** `web/src/lib/components/browse/NeighborsPanel.svelte:62,79-91`

`depthInvalid` is only ever written by `handleDepthChange`. It has no reset tied
to the `depth` prop. Scenario: type `2.5` into Depth and blur (`depthInvalid`
becomes true), then press browser Back to an entry with a different depth. The
`depth` prop changes, so `value={depth ?? ''}` resets the box to a conforming
value, but `aria-invalid` stays `"true"` — a field announced as invalid while
displaying a value the component itself accepts. Low impact today because the
attribute has no visual effect (see WR-01), but the two findings compound if
WR-01 is fixed without this one.

**Fix:** derive rather than store, or clear it alongside the prop:
`$effect(() => { depth; depthInvalid = false; });`

### IN-04: WR-01's typing-echo suppression is coupled to a SvelteKit invariant it does not state

**File:** `web/src/lib/components/browse/SearchPanel.svelte:133-137`

The effect compares `initialQuery` against the controller's **current** query
rather than against the last value this effect itself seeded. That is correct
only while SvelteKit never delivers a *superseded* `q` to the prop. It does not
today — the client router guards each navigation with a token and a cancelled
navigation never assigns `page.url` — so I could not construct a failing input,
which is why this is Info and not a Warning. If that invariant ever changed (or a
future caller drives `initialQuery` from something other than `page.url`), a
late-arriving stale `q` would call `setQuery` with an older value and visibly
revert characters the user had already typed.

**Fix (defence in depth, no behaviour change today):** record the last seeded
value and compare against both:

```ts
let lastSeeded = $state<string | undefined>(undefined);
$effect(() => {
	const incoming = initialQuery;
	if (incoming === lastSeeded) return;
	if (incoming === untrack(() => searchState.query)) { lastSeeded = incoming; return; }
	lastSeeded = incoming;
	controller.setQuery(incoming);
});
```

### IN-05: IN-04's host rule degrades the diagnostic for a legitimate one-character SSH host alias

**File:** `internal/gitmeta/permalink.go:169`

`len(candidateHost) > 1` also rejects a valid single-character `~/.ssh/config`
`Host` alias (e.g. a remote of `g:owner/repo.git`). Such a remote now reports
`origin remote is not a recognized URL` instead of the more accurate
`origin remote host "g" is not github.com`. No link is produced either way — the
host is not `github.com` — so the only cost is diagnostic precision for a rare
configuration. The positive control at `permalink_test.go:76-85` pins the
two-character case, so the boundary is deliberate and tested; recording it here
only so the trade-off is visible rather than accidental.

## Verification I performed independently

Recorded so the orchestrator can re-derive:

- `git diff --name-only 27f6fff5..HEAD` → 37 paths; 24 outside `web/build/` and
  `.planning/`; **zero** matching `proto|gen/`.
- `GOTOOLCHAIN=go1.26.5 go test ./internal/upgrade/ -run 'TestWorkflowFilePopulationMatchesDisk|TestGolangciEnabledLintersHeader' -count=1 -v`
  → three named `--- PASS` lines (`...MatchesConfig`, `...Discriminates`,
  `TestWorkflowFilePopulationMatchesDisk`), package `ok`.
- `git ls-files testdata/wireoracle/transcripts | wc -l` → `42` (positive control
  for the path); no transcript contains either NO_LINK reason string, so WR-02
  needed no refreeze.
- `.github/workflows/` → 14 `.yml`, 0 `.yaml`; the 3 github-script exceptions each
  have exactly one `uses:` and zero `run:`; both canaries' six `run:` steps are
  all `task <target>`.
- Committed bundle contains both WR-04 (`var A=[\`q\`]` + `t.delete(e)` in
  `nodes/0.ggKEIXcw.js`) and IN-09 (`.catch(()=>{})` in `nodes/3.CrBFS6Mo.js`).
- All compiled `aria-invalid` CSS selectors in `assets/0.Z5wzVqLj.css` are
  class-gated or `data-slot`-gated; the depth input has neither.
- `web/src/lib/` contains no `browse-tracer.ts`; `highlight.ts:168-176` warns
  rather than swallowing.
- All three `goto` stand-ins in `web/tests` (`browse-page.test.ts:66`,
  `browse-nav.test.ts:12`, `browse-history.test.ts:27`) now return a thenable, so
  the IN-09 mock fix left no divergent sibling.
- `git status --porcelain` → clean except the pre-existing untracked
  `.planning/milestone.lock`.

## What ships

**Ships as a defect:** WR-01 only. A user who types a non-integer depth gets no
visible feedback, which is the condition IN-08 was filed to remove. It is a
usability/a11y defect, not a correctness or security one — no wrong value is
computed, stored, navigated to, or transmitted.

**Ships as latent or diagnostic-only:** WR-02 (future workflow drift can slip past
a carve-out; nothing violating today), WR-03 (a swallowed rejection with no
reachable trigger at HEAD, plus a false justification comment), IN-01 through
IN-05.

**Ships clean:** the wire contract, the confinement gate, the commit-SHA
validation on both read sites, the permalink URL encoding, the scp-like remote
parser's local-path rejection, the status gate's request identity, the wireoracle
scanner-error handoff (its happens-before argument is sound: the write precedes
`close(lines)` in the same goroutine, and the sole read at `capture.go:450` occurs
only after a receive observes the close), and the CopyAction unmount cleanup.

---

_Reviewed: 2026-08-29T14:46:52Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
