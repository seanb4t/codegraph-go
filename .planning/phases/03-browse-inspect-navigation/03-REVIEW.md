---
phase: 03-browse-inspect-navigation
reviewed: 2026-08-29T00:00:00Z
depth: deep
files_reviewed: 41
files_reviewed_list:
  - internal/gitmeta/permalink.go
  - internal/gitmeta/permalink_test.go
  - internal/uiserver/permalink.go
  - internal/uiserver/permalink_test.go
  - internal/uiserver/confinement_test.go
  - internal/uiserver/degrade.go
  - internal/uiserver/readonly_test.go
  - internal/uiserver/handlers.go
  - internal/query/node.go
  - internal/mcp/tools.go
  - internal/agents/opencode.go
  - internal/corpora/coverage.go
  - internal/graphstore/export.go
  - internal/graphstore/pebble_store.go
  - internal/indexer/languages_python.go
  - internal/cli/upgrade.go
  - internal/upgrade/taskfile_shape_test.go
  - tools/corpora/prose.go
  - tools/transcriptfreeze/classify.go
  - test/wireoracle/capture.go
  - test/wireoracle/capture_test.go
  - test/wireoracle/normalize.go
  - test/wireoracle/scenarios.go
  - test/wireoracle/oracle_test.go
  - web/highlight_coverage_test.go
  - web/src/lib/browse-url.ts
  - web/src/lib/browse-state.ts
  - web/src/lib/browse-nav.ts
  - web/src/lib/call-targets.ts
  - web/src/lib/highlight.ts
  - web/src/lib/rpc-errors.ts
  - web/src/lib/search.ts
  - web/src/lib/status.ts
  - web/src/lib/components/StatusBanner.svelte
  - web/src/lib/components/browse/CopyAction.svelte
  - web/src/lib/components/browse/DefinitionPicker.svelte
  - web/src/lib/components/browse/NeighborsPanel.svelte
  - web/src/lib/components/browse/SearchPanel.svelte
  - web/src/lib/components/browse/SourcePane.svelte
  - web/src/routes/browse/+page.svelte
  - web/src/routes/+layout.svelte
  - Taskfile.yml
  - .github/workflows/ci.yml
  - .golangci.yml
findings:
  critical: 1
  warning: 9
  info: 14
  total: 24
status: issues_found
---

# Phase 3: Code Review Report

**Reviewed:** 2026-08-29
**Depth:** deep (cross-file: import graph, call chains, TS↔Go boundary, wire contract)
**Files Reviewed:** 41 in-scope source files (generated proto/connect output, `web/build/**`, and the 36 vendored `components/ui/**` files excluded per scope)
**Status:** issues_found

## Summary

Ten plans, ~14.7k inserted lines. The Go server side is in good shape: `GetPermalink`
reuses the one confinement gate rather than adding a second (`internal/query/node.go:91`
delegates to `resolveSourcePath` unchanged), the `fs.ErrNotExist` reclassification in
`internal/uiserver/permalink.go:90-95` builds its message only from the caller's own
repo-relative path and does **not** weaken `mapEngineError`'s default scrub, and
`percentEncodeRepoPath` escapes per-segment correctly. I traced every changed code path
that can reach the wire and found **no absolute-host-path disclosure**. The two
confinement guards' `strings.Contains(msg, dir)` negative checks were suspect (macOS
`t.TempDir()` returns the unresolved `/var/...` form while `EvalSymlinks` errors carry
`/private/var/...`); I verified empirically that `/private/var/folders/…` *does* contain
`/var/folders/…` as a substring, so those guards are sound. `web/highlight_coverage_test.go`
is the strongest new guard in the phase — real set equality in both directions, a reverse
binding, and a planted positive control.

The client side is where the defects are. One is a live, user-visible correctness bug on
the phase's own primary interaction path (CR-01, traced across four files). One more is a
proven DOM-corruption bug in a shared exported module that today's single caller happens
to mask (WR-01, reproduced under vitest). Three findings are rule-`84d1gfpywd`
instances — a guard that tests a copy of the code it claims to guard (WR-03), a negative
containment check anchored to an input that cannot contain the forbidden string (WR-04),
and a hand-enumerated population with no binding to its source of truth (WR-02, the
memory-`v4zqxrz6b3` shape).

**Verification caveat:** the Go test suite could not be executed on this host. `go build ./...`
fails inside `github.com/cockroachdb/swiss@v0.0.0-20251224182025` (`undefined: hashFn`,
`undefined: fastrand64`) because the local toolchain is go1.27.0 while `go.mod` declares
go 1.26.5 and CI pins via `go-version-file`. That is a pre-existing transitive-dependency /
toolchain mismatch, unrelated to this phase's diff, and no finding below depends on running
Go tests. The vitest suite does run; WR-01 was reproduced with it.

---

## Critical Issues

### CR-01: Typing in the search box tears down and re-issues the entire open-node view on every keystroke

**Files:**
- `web/src/lib/components/browse/SearchPanel.svelte:148-152`
- `web/src/routes/browse/+page.svelte:37`, `:62-89`, `:120-122`
- `web/src/routes/+layout.svelte:47-49`
- `web/src/lib/components/browse/SourcePane.svelte:149-175`

**Issue:** `handleInputChange` calls `onQueryChange?.(value)` on **every** `oninput` event,
undebounced (`SearchPanel.svelte:151`). `+page.svelte:120-122` routes that straight into
`navigator.navigate(page.url, { q: query || undefined }, NAV_INTENT.REFINE)`, which calls
`goto()` and therefore changes `page.url`. `params` is
`$derived(parseBrowseParams(page.url.searchParams))` (`+page.svelte:37`) and returns a
**fresh object literal** every time, so the derived signal is invalidated on each URL write.
The load effect at `+page.svelte:62-89` reads `params`, so it re-runs — and its first act
for any non-idle target is `targetState = { kind: 'loading' }` (`:73`), which makes
`SourcePane` render the `browse-loading` branch (`SourcePane.svelte:210-211`).

Concrete scenario. Open `/browse?symbol=resolveSourcePath&file=internal/query/node.go&line=33`
(the exact URL the 03-07 manual UAT used). Type the 7 characters `Callees` into the search box.
Result, per character:

1. `q` is rewritten in the URL → `params` identity changes → the load effect re-runs;
2. the previous `AbortController` is aborted and a new `GetNodeDetail` **and** `Impact` are
   dispatched for the *same, unchanged* symbol target;
3. `targetState` is set to `loading`, so the syntax-highlighted source, the callers/callees
   lists and the blast radius **all disappear** and are replaced by `Loading…` until the
   round trip completes;
4. `SourcePane`'s permalink effect (`:149-175`) also depends on `target`, so `GetPermalink`
   is re-issued too;
5. `navigationIdentity(url)` in `+layout.svelte:48` includes the sorted query string, so
   `?q=Ca` and `?q=Cal` are distinct identities — `statusGate.notifyNavigated` fires and
   `GetStatus` is re-fetched as well.

That is **4 RPCs per keystroke** (28 for a 7-character query) and a source pane that flickers
to `Loading…` on every character. This directly contradicts the phase goal's own wording
("keep clicking outward without losing their place") and D-05's "no timer, no polling loop" —
typing is now a de-facto polling loop keyed to keystrokes. No test covers it: every
`browse-*.test.ts` exercises the loaders and the navigator in isolation, never the
route-level `q`-write → `params`-change → reload cycle.

**Fix:** the load effect must depend only on the *target* params, not on the whole
`BrowseParams` object. Derive a narrow target key and gate on it:

```ts
// +page.svelte
let params = $derived(parseBrowseParams(page.url.searchParams));
// The load effect's real dependencies — q is view-local, not a target field.
let targetKey = $derived(
    `${params.symbol ?? ''}\0${params.file ?? ''}\0${params.line ?? ''}\0${params.depth ?? ''}\0${params.limit ?? ''}`
);

$effect(() => {
    targetKey;                        // the only tracked read
    const currentParams = untrack(() => params);
    ...
});
```

`targetKey` is a string, so an unchanged target produces an equal value and Svelte does not
invalidate dependents — typing no longer restarts the load. Independently,
`navigationIdentity` (`web/src/lib/status.ts:70-75`) should exclude view-local params such
as `q`, or `+layout.svelte:47-49` should compare only `page.url.pathname` plus the target
params, so `GetStatus` is not re-fetched per character either. Add a route-level test that
mounts the browse page with a fixed `symbol`/`file`, fires three `input` events, and asserts
the `getNodeDetail` stub was called exactly once.

---

## Warnings

### WR-01: `decorateCallTargets`' teardown re-attaches the previous node's source text into the live `<code>` element

**File:** `web/src/lib/call-targets.ts:216-234` (specifically `:227-229`), reached via
`web/src/lib/call-targets.ts:253-263` and `web/src/lib/components/browse/SourcePane.svelte:260-263`

**Issue:** The teardown closure restores each decorated span by inserting the recorded
`originalText` node back before `insertedNodes[0]`. When that anchor is no longer a child of
the recorded `parent`, it falls back to `parent.appendChild(originalText)` (`:228`). If
`parent` is the `<code>` element itself — which it is for every top-level, unhighlighted text
node hljs emits, i.e. the common case — and Svelte's `{@html}` has *already* replaced that
element's children, the fallback appends **stale text from the previous node's source** onto
the freshly rendered source. The user then reads two different functions' bodies concatenated
with no separator, presented as verbatim repository source.

Reproduced directly against the shipped component (vitest, jsdom, `@testing-library/svelte`):

```
render(SourcePane, { state: singleDef('alpha Target beta\n', calls: [Target]) })
rerender({ state: singleDef('gamma Other delta\n', calls: [Other]) })

code.textContent === "gamma Other delta\nalpha Target beta\n"   // observed
code.innerHTML   === 'gamma <button data-call-target="true">Other</button> delta\nalpha Target beta\n'
```

It also reproduces across three hops. It does **not** reproduce when the previous render had
no decorated target (`calls: []`), nor when the same state is re-rendered, nor when a
`{ kind: 'loading' }` render is interposed — and that last case is exactly what
`+page.svelte:73` currently does, which is why this is masked in production today rather than
live. The masking is incidental (it depends on Svelte flushing the `loading` DOM before the
RPC promise resolves) and is pinned by no test; removing the loading flash, or Phase 6's
live-push replacing content in place, exposes it immediately. The existing teardown test
(`web/tests/call-targets.test.ts:163-183`) only exercises the happy path where nothing else
mutated the subtree, so the `:228` fallback branch has zero coverage.

**Fix:** never re-insert into a parent the decorator no longer owns.

```ts
for (const { parent, originalText, insertedNodes } of decorations) {
    const anchor = insertedNodes[0];
    if (anchor && anchor.parentNode === parent) {
        parent.insertBefore(originalText, anchor);
        for (const inserted of insertedNodes) inserted.parentNode?.removeChild(inserted);
    }
    // else: the subtree was replaced by its owner (Svelte's {@html}); the
    // inserted nodes are already detached. Restoring here would inject stale
    // content into DOM this decorator no longer owns — drop it silently.
}
```

Add a regression test asserting `code.textContent` equals the new source exactly after a
direct single-def → single-def rerender.

### WR-02: `EXTENSION_LANGUAGE` is a hand-enumerated population with no binding to the indexer's extension registry

**File:** `web/src/lib/components/browse/SourcePane.svelte:74-105`

**Issue:** The extension→language map is transcribed by hand from
`internal/indexer/languages_*.go`. I verified it is *currently* correct — both sides carry
exactly the same 23 extensions (`.c .cc .cjs .cpp .cs .cxx .go .h .hh .hpp .java .js .jsx
.kt .kts .mjs .php .py .rb .rs .swift .ts .tsx`). But nothing binds it: `rg
'EXTENSION_LANGUAGE|languageForPath' web/` returns only `SourcePane.svelte` itself — no TS
test, no Go guard — and `internal/indexer` exports `RegisteredLanguageIDs()` but no
equivalent for extensions.

This is the memory-`v4zqxrz6b3` shape: the population narrows silently because a new subject
passes by being absent. Concrete: add `".hxx"` to `languages_cpp.go`'s `Extensions` list.
`RegisteredLanguageIDs()` is unchanged (still `cpp`), so
`TestHighlightRegistrationCoversRegisteredLanguages` — the phase's own coverage guard — stays
green. `/browse?file=foo.hxx` then renders `languageForPath` → `''` →
`highlightSource(text, '')` → `console.warn` + escaped plaintext. BRW-06's "syntax-highlighted
verbatim source" silently fails for that file type, with no failing test anywhere. The same
holds for any 15th language's extensions.

**Fix:** export the extension registry from the indexer and extend the existing Go guard,
which already parses `highlight.ts` as text and already sits in `web_test`:

```go
// internal/indexer/languages.go
func RegisteredLanguageExtensions() map[string]string { /* ext -> LanguageSpec.ID */ }
```

Then move `EXTENSION_LANGUAGE` out of the `.svelte` file into a single-purpose, literal-only
declaration (`web/src/lib/highlight.ts`, alongside `HIGHLIGHT_COVERAGE`, whose declaration
shape the Go guard already parses) and add a set-equality assertion in
`web/highlight_coverage_test.go` with a planted-fixture control matching
`TestHighlightCoverageComparisonDiscriminates`.

### WR-03: `TestCaptureArrivalLedgerPreservesWireOrder` guards a copy of the code, not the code

**Files:** `test/wireoracle/capture_test.go:26-75`, `test/wireoracle/capture.go:191-211` and
`:366-385`

**Issue:** The test drives `scanArrivalLines` (`capture.go:203`), which the file's own doc
comment describes as "the identical scan-and-timestamp primitive Capture's own
stdout-reading goroutine uses below … extracted here as a standalone, independently testable
function". `Capture` does **not** call it. The production path is the inline goroutine at
`capture.go:369-385`, which duplicates the scanner construction, the buffer size and the
copy-then-timestamp shape. `scanArrivalLines` has exactly one caller in the repository, and
it is this test.

The consequence is the failure mode rule `84d1gfpywd` names: the assertion passes while doing
nothing about the property it claims to protect. Concrete: change the inline goroutine at
`capture.go:369-385` to fan out across two scanner goroutines, or to buffer and sort lines
before sending, and `TestCaptureArrivalLedgerPreservesWireOrder` still passes green — the
"capture preserves wire order" claim that 03-03-EVIDENCE.md's whole VERDICT rests on becomes
unbacked, silently. This matters more than usual because the R2 resolution
(`CanonicalizeResponseOrder`) deliberately removed the oracle's own ability to detect
response reordering, leaving this test as the only remaining evidence that reordering is not
introduced downstream.

**Fix:** make `Capture` call the primitive. Replace the inline goroutine body with a channel
adapter over the same function, or extract a `scanTimestamped(r io.Reader, emit func(ArrivalLine))`
used by both, so the tested code and the running code are the same code:

```go
go func() {
    defer close(lines)
    _ = scanTimestamped(stdout, func(al ArrivalLine) {
        lines <- scannedLine{raw: al.Raw, arrived: al.Arrived}
    })
}()
```

### WR-04: the `{@html}` XSS guard's negative assertion is vacuous for the registered-language path

**File:** `web/tests/browse-tracer.test.ts:121-130`

**Issue:**

```ts
const highlighted = highlightSource('func main() {}', 'go');
expect(highlighted).toContain('class="hljs-');
expect(highlighted).not.toContain('<script>');       // <-- vacuous
```

The input `'func main() {}'` contains no `<`, so `not.toContain('<script>')` is trivially true
regardless of what `highlightSource` does. The paired positive
(`toContain('class="hljs-')`) proves the *highlighting* ran; it proves nothing about
*escaping*. The only genuine escaping assertion in the suite
(`:127-129`) covers the **unregistered**-language fallback (`escapeHtml`), which is the branch
`highlight.ts:110` itself says "should be unreachable in a correct tree".

`SourcePane.svelte:238` and `:262` are the repository's only `{@html}` sites, and the string
they render is produced by `hljs.highlight()` for a **registered** language — the path with no
escaping coverage at all. Concrete: change `highlightSource` to
`hljs.highlight(code, { language }).value` plus any HTML-passthrough plugin, or to a
hand-built `<span class="...">${code}</span>` concatenation, and both assertions still pass
while a Go source file containing `// <script>alert(1)</script>` becomes executable markup in
the browse view.

**Fix:** anchor the negative to an input that *can* contain the forbidden string, on the
registered-language path:

```ts
const hostile = highlightSource('func main() { /* <script>alert(1)</script> */ }', 'go');
expect(hostile).toContain('&lt;script&gt;');   // positive: escaping happened
expect(hostile).not.toContain('<script>');     // negative: now non-vacuous
```

### WR-05: the depth control writes a URL value that this app's own parser then silently discards

**Files:** `web/src/lib/components/browse/NeighborsPanel.svelte:56-62`,
`web/src/lib/browse-url.ts:46-52`

**Issue:** `handleDepthChange` accepts any value for which `Number.isFinite(Number(raw))` is
true, then writes it into the URL via `onNavigate({ depth: parsed }, REFINE)`.
`serializeBrowseParams` emits `String(p.depth)`. But `parseShapeInteger` accepts only
`/^-?\d+$/`, so a non-integer round-trips to `undefined`.

Concrete: type `2.5` into the Depth input (an `<input type="number">` accepts it; `min="0"`
does not constrain the decimal). The URL becomes `?symbol=Foo&…&depth=2.5`. On the next
`params` read, `depth` is `undefined`, so `loadBlastRadius` sends `depth: 0`
(`browse-state.ts:212`) and the server uses its own default. The address bar claims depth 2.5,
the rendered blast radius is the default depth, the input still shows `2.5`, and no error
surfaces anywhere. Sharing that URL reproduces the same silent disagreement for the recipient —
and NAV-01's whole premise is that the URL *is* the state. `1e3` behaves similarly (it
serializes as `1000`, silently changing the value the user typed).

The finding is specifically about the writer/parser asymmetry, not about range validation:
D-12 correctly leaves range to the server, and `depth=-1` (which *is* a valid integer literal)
correctly reaches the server and returns `CodeInvalidArgument`.

**Fix:** make the writer speak the same grammar as the reader.

```ts
function handleDepthChange(e: Event): void {
    const raw = (e.currentTarget as HTMLInputElement).value;
    if (raw === '') { onNavigate({ depth: undefined }, NAV_INTENT.REFINE); return; }
    if (!/^-?\d+$/.test(raw)) return;   // same shape check browse-url.ts applies
    onNavigate({ depth: Number(raw), }, NAV_INTENT.REFINE);
}
```

Better still, export the shape predicate from `browse-url.ts` so there is one definition
rather than two that can drift.

### WR-06: a valid GitHub origin in git's user-less scp-like form is classified as `NO_LINK`

**File:** `internal/gitmeta/permalink.go:135-151`

**Issue:** `parseGitHubRemote`'s no-scheme branch recognises the scp-like form **only** when an
`@` is present (`:139`). Git's own documented syntax makes the user optional:
`[user@]host.xz:path/to/repo.git/`. A repository configured with
`git remote add origin github.com:seanb4t/codegraph-go.git` is a perfectly ordinary,
git-accepted remote, but `strings.Index(raw, "@")` returns `-1`, the function returns
`("", "", "")`, and `RemoteGitHubRepo` (`:106-109`) reports
`Reason: "origin remote is not a recognized URL"`.

Concrete effect: BRW-09's permalink surface renders
`No permalink available: origin remote is not a recognized URL`
(`SourcePane.svelte:198-201`) for a repository that *does* have a GitHub origin — and D-20's
truncated-file escape hatch disappears with it. The unit tests only cover
`https://github.com/owner/repo.git` and `https://gitlab.com/...`
(`internal/uiserver/permalink_test.go:61,86,110,134,166`), so the shape is untested in both
directions.

**Fix:** treat the `@` as optional, keying the scp-like decision on the colon-before-slash
rule the code already implements:

```go
if idx := strings.Index(raw, "://"); idx == -1 {
    rest := raw
    if at := strings.Index(raw, "@"); at >= 0 {
        rest = raw[at+1:]
    }
    if colon := strings.Index(rest, ":"); colon >= 0 {
        slash := strings.Index(rest, "/")
        if slash == -1 || colon < slash {
            host = rest[:colon]
            owner, repo = splitOwnerRepo(rest[colon+1:])
            return host, owner, repo
        }
    }
    return "", "", ""
}
```

Add table cases for `github.com:owner/repo.git` and `git@github.com:owner/repo.git` to
`internal/gitmeta/permalink_test.go`.

### WR-07: the indexed commit SHA reaches a git CLI argument and a rendered URL with no read-side validation

**Files:** `internal/uiserver/permalink.go:104`, `:122`, `:124`;
`internal/gitmeta/permalink.go:224`, `internal/uiserver/permalink.go:162-180`

**Issue:** `schema.IndexedCommitSHA` (`internal/schema/meta.go:41-50`) returns the stored
string with no format check. `internal/indexer/commit.go:75` validates the SHA with
`isLowercaseHexCommitSHA` at **write** time — the read side skips that validator entirely.
The unvalidated value is then used in two places that both assume it is opaque-safe:

1. `exec.CommandContext(ctx, "git", "branch", "-r", "--contains", sha)`
   (`gitmeta/permalink.go:224`) — the argument is not preceded by `--`, so a value beginning
   with `-` is parsed by git as an option rather than a rev.
2. `b.WriteString(sha)` spliced raw into the blob URL (`uiserver/permalink.go:169`), between
   `/blob/` and the percent-encoded path. Every other component of that URL is escaped; this
   one is not.

Concrete input: a graph store whose `Meta.commit_sha` is
`../../attacker/attacker-repo/blob/main`. `GetPermalink` then returns
`https://github.com/owner/repo/blob/../../attacker/attacker-repo/blob/main/pkga/pkga.go`,
which the browser normalises to
`https://github.com/attacker/attacker-repo/blob/main/pkga/pkga.go` and which
`SourcePane.svelte:182-190` renders as a `View on GitHub` link with
`availability = LINKABLE`. The user believes they are following a permalink into their own
repository.

Honest caveat on reachability: no shipped command imports a store today —
`graphstore.Import` (`internal/graphstore/export.go:154`) has no CLI caller, only tests. The
attacker-supplied-index precondition is therefore not reachable through the current binary.
It is, however, exactly the shape `.claude/CLAUDE.md` names as the milestone-2 architecture
("CI-distributed indexes"), and both hardening steps are one-liners against a validator that
already exists.

**Fix:** validate once at the read boundary and pass `--` to git.

```go
// internal/schema/meta.go — reuse the same predicate the indexer already applies at write.
sha, ok := schema.IndexedCommitSHA(meta)
if !ok || !schema.IsCommitSHA(sha) {   // promote isLowercaseHexCommitSHA to schema
    resp = &uiv1.GetPermalinkResponse{
        Availability: uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_NO_LINK,
        Reason:       noCommitSHAReason,
    }
    return nil
}
```

```go
// internal/gitmeta/permalink.go
cmd := exec.CommandContext(ctx, "git", "branch", "-r", "--contains", "--", sha)
```

### WR-08: `createStatusGate` has no cancellation or response-identity guard; a stale verdict can overwrite a fresh one

**File:** `web/src/lib/status.ts:125-137`, `:147-151`

**Issue:** `fetchStatus()` fires `client.getStatus({})` with no `AbortSignal` and no
monotonic request id, and whichever promise settles last wins via `emit`. Every other
async surface built in this phase guards this explicitly —
`search.ts:133,176,188` (abort + `liveRequestId`), `browse-state.ts:239-261`
(`NavigationGeneration`), `+page.svelte:77` — and `status.ts`'s own doc comment
(`:97-100`) reasons carefully about *fetch count* while never addressing *response order*.

Concrete: the user runs `codegraph index` in another terminal. They click Browse (fetch A
starts, server still reports `stale: true`), then click Health ~50 ms later (fetch B starts,
indexing has finished, server reports `stale: false`). If A's response lands after B's — an
ordinary loopback interleaving, and one made far more likely by CR-01, which fires a
`GetStatus` per keystroke — the banner reverts to
`The index is stale — it may not reflect recent changes` and stays there until the next
navigation. That is precisely the case D-05's "sees the change on their next click" contract
promises to handle.

`web/tests/status.test.ts` asserts fetch *counts* (1 → 1 → 2) but never lands two responses
out of order, so this is untested.

**Fix:** mint an id per fetch and drop superseded responses, mirroring `search.ts`:

```ts
let statusRequestId = 0;
function fetchStatus(): void {
    const requestId = ++statusRequestId;
    client.getStatus({})
        .then((r) => { if (requestId === statusRequestId) emit(classifyStatus(r)); })
        .catch(() => { if (requestId === statusRequestId) emit(UNKNOWN_STATUS); });
}
```

Add a test that resolves two `getStatus` promises out of order and asserts the final verdict
is the later request's.

### WR-09: `CopyAction` swallows every clipboard failure — no feedback, no handling, unhandled rejection

**File:** `web/src/lib/components/browse/CopyAction.svelte:20-22`

**Issue:**

```ts
async function handleCopy(): Promise<void> {
    await navigator.clipboard.writeText(value);
}
```

There is no `try`/`catch` and no success/failure surface. `onclick={handleCopy}` discards the
returned promise, so every rejection becomes an unhandled rejection in the console and a
no-op on screen.

Two concrete failures, both ordinary:

1. `navigator.clipboard` is `undefined` outside a secure context. `codegraph ui` binds
   loopback today so `http://127.0.0.1` is secure — but Phase 1's D-08 shipped the bind
   address as an unwired field explicitly intended to be wired later; the first time it is
   pointed at a LAN address, `handleCopy` throws `TypeError: Cannot read properties of
   undefined (reading 'writeText')` and the button becomes silently inert.
2. Even on loopback, `writeText` rejects with `NotAllowedError` when the document is not
   focused (e.g. the click is dispatched while a devtools panel holds focus). Nothing is
   copied; the user is given no signal and will paste whatever was on the clipboard before.

This is exactly the pattern `~/.claude/CLAUDE.md` forbids ("MUST surface problems clearly,
never hide them"), and the component's own doc comment argues *against* silent inertness
("a present-but-inert affordance promises an action it cannot perform").

**Fix:** handle both branches and give the user a signal.

```ts
let copyState = $state<'idle' | 'copied' | 'failed'>('idle');

async function handleCopy(): Promise<void> {
    try {
        if (!navigator.clipboard) throw new Error('clipboard API unavailable in this context');
        await navigator.clipboard.writeText(value);
        copyState = 'copied';
    } catch {
        copyState = 'failed';
    }
    setTimeout(() => (copyState = 'idle'), 1500);
}
```

and render `Copied` / `Copy failed` from `copyState` with a `data-testid` a test can assert.

---

## Info

### IN-01: `//nolint:staticcheck` suppresses the whole linter on that line, not just ST1005

**File:** `internal/uiserver/degrade.go:106-111`

The justification is sound on its merits — `indexingInProgressMessage` is user-facing prose
crossing to a browser, never a wrapped internal error, so ST1005 genuinely does not apply, and
the reasoning is stated inline as the project's rules require. The narrow issue is
granularity: `//nolint:staticcheck` disables *all* staticcheck classes on that line (SA
correctness checks, S simplifications, QF quickfixes), not only ST1005. A future real
`SA`-class finding on `connect.NewError(...)` would be silently suppressed.

A directive-free alternative removes the trade-off entirely, because ST1005 only inspects
string literals passed to `errors.New`/`fmt.Errorf`:

```go
type indexingInProgressError struct{}
func (indexingInProgressError) Error() string { return indexingInProgressMessage }
// ...
connErr := connect.NewError(connect.CodeUnavailable, indexingInProgressError{})
```

### IN-02: the `classifiedErr` outer variable is a second, unaudited error path out of `GetPermalink`

**File:** `internal/uiserver/permalink.go:76-98`, `:145-150`

The pattern is correct today and its rationale (avoiding `withEngine`'s unconditional
`mapEngineError` re-wrap) is documented well. But it establishes a return path that bypasses
the one information-disclosure scrub the package relies on. Nothing prevents a future edit
inside the closure from writing `classifiedErr = err` for a raw engine error carrying an
absolute host path, and `TestGetPermalinkRefusesSinceDeletedFile` only exercises the one
existing branch. Consider constraining the variable's type to make misuse hard, e.g.
`var classified *connect.Error` with a small helper that only accepts a code plus a
caller-supplied message.

### IN-03: `TestToolModfilesPopulationMatchesDisk` is a count check whose set-equality property is only complete alongside its sibling

**File:** `internal/upgrade/taskfile_shape_test.go:990-999`

`len(matches) != len(isolatedModfilePaths)` is genuinely set-complete *in conjunction with*
`TestToolModfilesRemainIsolated`, which `os.Stat`s every registered path (so slice ⊆ disk) and
pairwise-`SameFile`s them (so the slice has no duplicates). Equal cardinality plus containment
plus distinctness does imply equality. The dependency is implicit, though: run
`go test -run '^TestToolModfilesPopulationMatchesDisk$'` alone and it is a bare count. Making
it self-contained is three lines:

```go
sort.Strings(matches)
want := append([]string(nil), isolatedModfilePaths...)
sort.Strings(want)
if !slices.Equal(matches, want) { t.Fatalf(...) }
```

### IN-04: the new `lint-go` job is not in `requiredCheckNames`, so it does not block merge

**Files:** `.github/workflows/ci.yml:453-481`, `internal/upgrade/taskfile_shape_test.go:76-84`

`lint-go` was added to `inScopeJobs` (`:154`) but not to `requiredCheckNames`, the fixture of
GitHub ruleset 20157557's required status-check contexts. `actionlint (workflow static
analysis)` and `test` are in that list; `lint-go` is not. A PR with a red `lint-go` leg is
therefore mergeable. 03-CONTEXT.md's Folded Todo #1 asked for "CI invokes it" and this
satisfies the letter of that, but if the intent was a blocking gate, the repository ruleset
needs the new context added and the fixture updated alongside it (re-verified with
`gh api repos/seanb4t/codegraph-go/rulesets/20157557`, per the fixture's own instructions).

### IN-05: `inScopeJobs` and `requiredCheckNames` remain hand-enumerated with no disk binding

**File:** `internal/upgrade/taskfile_shape_test.go:76-84`, `:151-165`

Plan 03-10 correctly closed this class for tool modfiles
(`TestToolModfilesPopulationMatchesDisk`) but left the same shape untouched two declarations
above it, in the file it was already editing. `inScopeJobs` has only a non-empty guard
(`:1429-1431`); nothing asserts that every job in `ci.yml` appears in it. A new CI job added
without a matching entry is bound by nothing and the suite stays green — the identical defect,
one fixture over. The modfile guard's own doc comment even points at `inScopeJobs` as the
precedent it is imitating.

### IN-06: `IDENTIFIER_PATTERN` is exported with the `g` flag, sharing mutable `lastIndex`

**File:** `web/src/lib/call-targets.ts:55`

`String.prototype.matchAll` clones the regex, so the current single call site
(`:125`) is safe. But the symbol is exported and any future `.test()` or `.exec()` call on it
advances `lastIndex` on the shared instance, after which `matchAll` (which seeds the clone
from the original's `lastIndex`) silently skips the beginning of subsequent inputs —
identifiers at the start of a text node stop being clickable, with no error. Prefer exporting
a factory (`export const identifierPattern = () => /.../gu`) or a non-global source string.

### IN-07: `CanonicalizeResponseOrder` classifies "any line with a numeric id" as a response

**File:** `test/wireoracle/normalize.go:230-243`, using `responseID`
(`test/wireoracle/capture.go:605-613`)

`responseID` returns ok for any JSON line carrying a numeric top-level `id` — which, per
JSON-RPC 2.0, includes *requests*, not only responses. Every server→client request
(`sampling/createMessage`, `roots/list`, `elicitation/create`) carries both `method` and `id`.
No current scenario emits one, so this is latent. When one is added, that request frame joins
the sorted-by-id set and can be swapped with an unrelated response line, masking or
fabricating an ordering discrepancy in the frozen transcript. Narrow the predicate to
`id present AND method absent` — `frameMethod` (`capture.go:619`) already exists next door.

### IN-08: the `RemotePresence` switch's `default` arm silently asserts "not observed"

**File:** `internal/uiserver/permalink.go:124-142`

`default: // gitmeta.RemotePresenceNotObserved` means any future `RemotePresence` member is
reported to the user as `notObservedReason` — "this commit is not observed on any
remote-tracking branch; it may be unpushed" — a positive claim about the commit that a new
"could not determine" style member would make false. Given D-07's insistence that "could not
check" must never be stated as fact, prefer an explicit
`case gitmeta.RemotePresenceNotObserved:` with `default:` falling back to the *Unknown*
wording, which is the safe direction.

### IN-09: the tri-state `RemotePresence` collapses to two wire values, with the distinction surviving only as free-text prose

**File:** `internal/uiserver/permalink.go:130-141`

`RemotePresenceUnknown` and `RemotePresenceNotObserved` both map to
`PERMALINK_AVAILABILITY_LINKABLE_UNVERIFIED`, differing only in `reason` — a human-readable
string, not a machine-readable field. This is the recorded and frozen design (the enum shape
was fixed at 03-05's blocking-human checkpoint, and the code documents the choice at
`:38-47`), so it is not a defect. It is worth recording that the wire is now additive-only:
a future client that needs to distinguish "checked, not there" from "could not check"
programmatically cannot, and adding a fourth availability member later would change the
meaning of `LINKABLE_UNVERIFIED` for existing clients.

### IN-10: `GetPermalink` performs no validation of `line` / `end_line`

**File:** `internal/uiserver/permalink.go:162-180`

`buildGitHubBlobURL` dereferences `line` and `endLine` without checking `*line >= 1` or
`*endLine >= *line`. A client may send `line: -1` (`#L-1`) or `endLine: 0` with
`line: 42` (`#L42-L0`), producing an anchor GitHub cannot resolve. I could not construct a
reachable path from the shipped client: `SourcePane.permalinkParamsFor` (`:129-140`) sources
both values from `Node.start_line`/`Node.end_line`, every extractor populates both
(`internal/indexer/*/…extract.go`), and the one node kind with neither — the package
pseudo-node built at `internal/indexer/resolve.go:206` — carries no `FilePath`, so
`ValidateRepoRelativePath("")` refuses the request before any URL is built. Recorded as Info
for that reason. Since the RPC is now a frozen public method callable by anything, a
`connect.CodeInvalidArgument` for `line < 1` or `end_line < line` would be cheap and matches
the service's own validation posture elsewhere (`validateLimit`, `validateFilesDepth`).

### IN-11: `NeighborsPanel` emits duplicate `data-testid` values across its three sections

**File:** `web/src/lib/components/browse/NeighborsPanel.svelte:78`, `:102`, `:145`

All three regions build `neighbor-entry-${entryKey(entry)}` from `filePath:startLine:name`.
A node that appears both as a callee and in the blast radius — the common case for a
self-referential impact set, which 03-07's own manual UAT observed ("one blast-radius entry
(itself)") — yields two elements with the same test id, and
`screen.getByTestId(...)` throws `Found multiple elements`. Prefix the id with the region
(`neighbor-callee-…`, `neighbor-blast-…`).

### IN-12: SearchPanel's "seed the query once on mount" effect actually re-runs on every `q` change

**File:** `web/src/lib/components/browse/SearchPanel.svelte:101-107`, with
`web/src/routes/browse/+page.svelte:129`

```ts
// Seed the query once on mount ...
$effect(() => {
    if (initialQuery) controller.setQuery(initialQuery);
});
```

`initialQuery` is `params.q ?? ''`, which changes on every keystroke once CR-01's URL write
is in play, so the effect re-runs and calls `setQuery` a second time per character. That
resets the 150 ms debounce timer an extra time on each keystroke, delaying the live search by
one URL-settle round trip. Behaviour still converges, but the comment describes a
mount-once seed the code does not implement. Guard it (`untrack`, or a `seeded` flag) so the
stated contract and the code agree.

### IN-13: `createSearchController` exposes no disposal; a pending debounce fires after unmount

**File:** `web/src/lib/search.ts:115-261`

`debounceTimer` is never cleared on teardown and the controller has no `destroy()`.
`SearchPanel` unsubscribes from the store (`:91-96`) but cannot stop the timer, so unmounting
the panel within 150 ms of the last keystroke still dispatches `Search` + `Files` for a
component that no longer exists. Harmless today (the results are discarded), but it is the
one async surface in this phase without a lifecycle exit. Add
`dispose() { clearTimeout(debounceTimer); liveAbort?.abort(); exploreAbort?.abort(); }` and
call it from the panel's `$effect` cleanup.

### IN-14: several golangci-lint `errcheck` fixes discard the error rather than handle it

**Files:** `internal/agents/opencode.go:245`, `internal/mcp/tools.go:210,416,432,449,466,483,507,524`

`removeOpencodeEntry(stalePath)` → `_, _ = removeOpencodeEntry(stalePath)` and
`defer close()` → `defer func() { _ = close() }()` satisfy the linter by making the discard
explicit without changing behaviour. Both are defensible (a best-effort stale-file sweep; an
engine closer on a read-only path), and neither is a regression — the errors were already
being dropped. Recording it because the project's own rule is "MUST attempt proper fix first
(not ignore directives)", and `_ =` is an ignore directive expressed in the type system. At
minimum the nine `defer func() { _ = close() }()` sites would benefit from a one-line comment
stating why a closer error is non-actionable there, in the style the rest of this codebase
uses.

### IN-15: `.golangci.yml`'s `# enabled-linters: 5` header has no durable guard

**File:** `.golangci.yml:1-12`

The count is described as "machine-read by 03-10-PLAN.md Task 2's `<verify>` block" — a
one-shot plan-time check, not a repository test. Nothing in `internal/upgrade/*_test.go`
parses it. Enabling a sixth linter without updating the comment produces no failure anywhere,
so the header will drift from the config it annotates. Either bind it (a small test parsing
the comment and counting `linters.enable` + `formatters.enable`, mirroring
`TestUIProtoFieldNumbersAreStableAndUnique`'s pinned-length pattern) or delete the count and
keep only the prose rationale.

---

_Reviewed: 2026-08-29_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
