---
phase: 03-browse-inspect-navigation
reviewed: 2026-08-29T13:59:52Z
head: 76d7e417
iteration: 2
depth: deep
files_reviewed: 88
files_reviewed_list:
  - .github/workflows/ci.yml
  - .golangci.yml
  - Taskfile.yml
  - go.tool-golangci.mod
  - internal/agents/antigravity_test.go
  - internal/agents/opencode.go
  - internal/cli/present/sanitize_test.go
  - internal/cli/upgrade.go
  - internal/cli/upgrade_test.go
  - internal/corpora/coverage.go
  - internal/corpora/coverage_test.go
  - internal/daemon/daemon_test.go
  - internal/gitmeta/permalink.go
  - internal/gitmeta/permalink_test.go
  - internal/graphstore/export.go
  - internal/graphstore/pebble_store.go
  - internal/graphstore/store_test.go
  - internal/indexer/commit.go
  - internal/indexer/languages.go
  - internal/indexer/languages_python.go
  - internal/mcp/skill_claims_drift_test.go
  - internal/mcp/tools.go
  - internal/query/node.go
  - internal/query/render_status_test.go
  - internal/schema/meta.go
  - internal/uiproto/uiv1/ui.proto
  - internal/uiserver/confinement_test.go
  - internal/uiserver/degrade.go
  - internal/uiserver/handlers.go
  - internal/uiserver/handlers_test.go
  - internal/uiserver/permalink.go
  - internal/uiserver/permalink_test.go
  - internal/uiserver/readonly_test.go
  - internal/uiserver/server_test.go
  - internal/uiserver/spa_test.go
  - internal/upgrade/golangci_shape_test.go
  - internal/upgrade/release_workflow_shape_test.go
  - internal/upgrade/swap_test.go
  - internal/upgrade/taskfile_shape_test.go
  - internal/upgrade/upgrade_test.go
  - test/wireoracle/capture.go
  - test/wireoracle/capture_test.go
  - test/wireoracle/normalize.go
  - test/wireoracle/normalize_test.go
  - test/wireoracle/oracle_test.go
  - test/wireoracle/scenarios.go
  - tools/corpora/measure_test.go
  - tools/corpora/prose.go
  - tools/corpora/prose_test.go
  - tools/transcriptfreeze/classify.go
  - web/highlight_coverage_test.go
  - web/highlight_extension_coverage_test.go
  - web/package.json
  - web/src/ambient-node.d.ts
  - web/src/lib/browse-nav.ts
  - web/src/lib/browse-state.ts
  - web/src/lib/browse-url.ts
  - web/src/lib/call-targets.ts
  - web/src/lib/components/StatusBanner.svelte
  - web/src/lib/components/browse/CopyAction.svelte
  - web/src/lib/components/browse/DefinitionPicker.svelte
  - web/src/lib/components/browse/NeighborsPanel.svelte
  - web/src/lib/components/browse/SearchPanel.svelte
  - web/src/lib/components/browse/SourcePane.svelte
  - web/src/lib/highlight.ts
  - web/src/lib/rpc-errors.ts
  - web/src/lib/search.ts
  - web/src/lib/status.ts
  - web/src/routes/+layout.svelte
  - web/src/routes/browse/+page.svelte
  - web/tests/browse-history.test.ts
  - web/tests/browse-nav.test.ts
  - web/tests/browse-page.test.ts
  - web/tests/browse-state.test.ts
  - web/tests/browse-tracer.test.ts
  - web/tests/browse-url.test.ts
  - web/tests/call-targets.test.ts
  - web/tests/definition-picker.test.ts
  - web/tests/degrade-states.test.ts
  - web/tests/fixtures/Harness.svelte
  - web/tests/harness.test.ts
  - web/tests/neighbors-panel.test.ts
  - web/tests/rpc-errors.test.ts
  - web/tests/search-panel.test.ts
  - web/tests/search.test.ts
  - web/tests/setup.ts
  - web/tests/source-pane.test.ts
  - web/tests/status.test.ts
  - web/tests/support/browse-page-state.svelte.ts
  - web/vite.config.ts
findings:
  critical: 0
  warning: 5
  info: 9
  total: 14
status: issues_found
---

# Phase 3: Code Review Report (iteration 2 — post-fix re-review)

**Reviewed:** 2026-08-29T13:59:52Z
**HEAD:** 76d7e417
**Depth:** deep
**Diff base:** f2c0bcda (the phase's full change set); fix pass is 74c8f5e7..e57268d9 plus 6ff79d4a / 77b75129
**Status:** issues_found

## Summary

This pass audits the 22 applied fixes plus the two pre-applied maintainer fixes, not
the original findings. Twenty of the twenty-two do exactly what the fix report claims
and introduce nothing new; I re-derived the load-bearing ones directly rather than
taking the report's word for them.

Verified good, in the reviewer's own hands:

- **WR-02** — ran `GOTOOLCHAIN=go1.26.5 go test ./web/ -run 'TestExtensionLanguage' -v`.
  Both sides are 23 entries and set-equal key-and-value; the discriminator produces
  exactly `missing=[.rs] extra=[.zig] mismatched=[.py: indexer="python" ts="ruby"]`.
  The binding is transitive (EXTENSION_LANGUAGE values -> indexer IDs via this guard;
  indexer IDs -> hljs registration via `highlight_coverage_test.go`), and neither
  guard can pass over an empty parse: both fail loudly on a zero-entry parse and both
  carry a positive floor (`< 20`, `< 14`).
- **WR-03** — `Capture`'s stdout goroutine now genuinely calls `scanTimestamped`, the
  same function `TestCaptureArrivalLedgerPreservesWireOrder` drives. The tested code
  is the running code.
- **WR-07** — `schema.IsCommitSHA` is applied at `internal/uiserver/permalink.go:164`,
  before `buildGitHubBlobURL` and before `gitmeta.CommitOnRemoteTrackingBranch`. The
  rejection of the `git branch --contains -- sha` suggestion is documented in place at
  `internal/gitmeta/permalink.go:230-242` and I did not reproduce a defect it would
  fix; I accept the rejection.
- **Wire shape (criterion 3)** — `git diff 74c8f5e7~1..HEAD -- internal/uiproto web/src/lib/gen`
  is empty. `PermalinkAvailability` still has its four members
  (`ui.proto:586-589`), `RemotePresence` still has its three states
  (`internal/gitmeta/permalink.go:196-209`), and IN-08 made `RemotePresenceNotObserved`
  an explicit case rather than removing anything. Nothing was renumbered or collapsed.
- **Disclosure scrub (criterion 4)** — `mapEngineError`'s default arm
  (`internal/uiserver/handlers.go:112-132`) is untouched. `GetPermalink`'s one
  reclassified arm still matches only `errors.Is(verr, fs.ErrNotExist)` and builds its
  message from the caller's own repo-relative `path`, never from the `*fs.PathError`
  text. IN-10's new pre-`withEngine` returns are `connect.NewError` values built from
  request fields only. No new path for an absolute host path to cross the wire.
- **Ignore directives (criterion 5)** — IN-01 removed the blanket
  `//nolint:staticcheck` by construction; `rg nolint --type go` returns four remaining
  directives, all pre-existing and all carrying inline reasons. IN-14's comments cover
  8 of 8 `defer func() { _ = close() }()` sites in `internal/mcp/tools.go` (the ninth
  regex hit is the doc comment itself at line 66).

Five fixes are less clean than the report claims, and the five WARNINGs below are all
consequences of the fix pass rather than survivors of the original review. Two of them
(**WR-01**, **WR-04**) are behavior the fix pass changed or masked beyond its own
finding, which is the class this re-review exists to catch. Nothing here is Critical:
I could not construct a security, data-loss, or crash scenario from any of it.

## Warnings

### WR-01: IN-12's `untrack()` broke URL-to-search-box resynchronization on back/forward

**File:** `web/src/lib/components/browse/SearchPanel.svelte:126-129`
(`web/src/lib/components/browse/SearchPanel.svelte:84`, `web/src/routes/browse/+page.svelte:165-170`)

**Issue:** The seed effect now reads `initialQuery` through `untrack()`, so it has zero
tracked dependencies and runs exactly once at mount. That is what the comment asked
for, but `initialQuery` is `params.q ?? ''` and `SearchPanel` is **not** keyed or
remounted on navigation (`+page.svelte:165` renders it unconditionally, no `{#key}`).
The search box's displayed text is `searchState.query` (line 191), which only ever
changes via `controller.setQuery` — called at line 128 (once, at mount) and line 172
(typing). So a URL change to `q` that did not originate from typing can no longer
reach the panel at all.

Concrete reproduction (all pushState/replaceState behavior is this phase's own, D-11):

1. Open `/browse`. Type `alpha` — REFINE replaces the entry, URL is `/browse?q=alpha`.
2. Click a search result — `handleSearchSelect` (`+page.svelte:135-147`) NAVIGATEs,
   pushing `/browse?q=alpha&file=x.go` (`applyDelta` carries `q` forward untouched,
   `browse-nav.ts:99-116`).
3. Type `beta` — REFINE replaces entry 2, URL is `/browse?q=beta&file=x.go`.
   History is now `[/browse?q=alpha, /browse?q=beta&file=x.go]`.
4. Press the browser Back button. URL returns to `/browse?q=alpha`; `initialQuery`
   becomes `"alpha"`.
5. The search input still reads `beta` and still lists beta's results. The address bar
   and the rendered view disagree — the exact failure D-11's "correct and shareable at
   every instant" contract exists to prevent. Before this fix the effect re-ran and
   resynced.

The fix's own new test pins the regression rather than catching it:
`web/tests/search-panel.test.ts:340-360` rerenders with `initialQuery: 'FooBar'` and
asserts the search still fires with `term: 'Foo'`.

**Fix:** Keep the debounce-reset fix but make the effect react to a URL-originated
change only. Track `initialQuery` and skip re-seeding when it already equals the
controller's current query:

```svelte
$effect(() => {
	const incoming = initialQuery; // tracked on purpose
	if (incoming === untrack(() => searchState.query)) return; // typing echo: no-op
	controller.setQuery(incoming);
});
```

That is a no-op for the per-keystroke echo IN-12 was fixing (the values are equal, so
no `setQuery` and no debounce reset) and still resyncs on back/forward, where they
differ. Add a test that rerenders with a `initialQuery` differing from the current
`searchState.query` and asserts the input value follows the URL.

---

### WR-02: WR-07's malformed-SHA arm reuses `noCommitSHAReason`, putting a false statement on the wire and collapsing two distinct causes

**File:** `internal/uiserver/permalink.go:164-170` (reason constant at `:36`)

**Issue:** When `schema.IsCommitSHA(sha)` rejects a stored value, the handler answers
with the reason string `"this index has no recorded commit SHA (a pre-upgrade graph)
- re-index to enable permalinks"`. For this branch that sentence is factually wrong on
both clauses: the index *does* carry a recorded commit SHA, and the store is not a
pre-upgrade graph. An operator whose store was written by a foreign indexer (the
milestone-2 CI-distributed-index shape WR-07's own commit message invokes) is told the
field is absent, which is the one diagnosis that hides the real signal — that something
wrote an unvalidated value into `Meta.commit_sha`.

This also contradicts the module's own stated discipline three lines up: the
`notObservedReason` / `checkUnknownReason` split at `permalink.go:44-47` exists
precisely so two different causes never collapse into the same wire string ("the text
differs, so an operator reading `reason` can still tell ... without conflating them
into the SAME string"). WR-07 reintroduced the collapse it argues against.

Concrete: run `overwriteCommitSHA(t, dir, "not-a-sha")` against a store whose
`Meta.commit_sha` was previously the real 40-hex HEAD, then call `GetPermalink`. The
response is byte-identical to the response for a genuine pre-upgrade graph with an
empty `commit_sha`; no client or operator can tell the two apart.

The regression test does not distinguish them either:
`internal/uiserver/permalink_test.go:362` asserts only
`strings.Contains(resp.Msg.GetReason(), "re-index")`, which both strings satisfy.

**Fix:** Give the branch its own constant and assert on it:

```go
const malformedCommitSHAReason = "this index's recorded commit SHA is not a well-formed git object id; the store may have been written by a different tool - re-index to repair it"

if !schema.IsCommitSHA(sha) {
	resp = &uiv1.GetPermalinkResponse{
		Availability: uiv1.PermalinkAvailability_PERMALINK_AVAILABILITY_NO_LINK,
		Reason:       malformedCommitSHAReason,
	}
	return nil
}
```

and change `permalink_test.go:362` to compare against `malformedCommitSHAReason`
exactly, so the two NO_LINK causes stay distinguishable by construction.

---

### WR-03: IN-05's new population guard is itself bound to a hand-enumerated file set — 9 of 14 workflow files are covered by neither the guard nor a declared exception

**File:** `internal/upgrade/taskfile_shape_test.go:1509-1512` (`inScopeWorkflowFiles`),
consumed at `:1560-1571`

**Issue:** IN-05 correctly closed the gap for *jobs inside three files*, and its
`usesOnlyJobExceptions` carve-out is properly validated against disk
(`validateUsesOnlyJobExceptions`, `:1499-1533`) — a stale exception fails loudly. But
the file population it iterates is itself a bare hardcoded list:

```go
var inScopeWorkflowFiles = []string{"ci.yml", "release-please.yml", "corpora.yml"}
```

with no disk binding and no validated exception record. Its doc comment states that
"bench.yml and release.yml carry their own documented D-01 exceptions ... and are
deliberately excluded from this population check too", which reads as though the
directory holds five files. It holds fourteen
(`ls .github/workflows/*.yml | wc -l` -> `14`). Nine are named nowhere:
`auto-close-unsolicited-prs.yml`, `auto-label-issues.yml`, `close-draft-prs.yml`,
`darwin-toolchain-canary.yml`, `linux-cross-canary.yml`, `post-release-verify.yml`,
`pr-template-format.yml`, `pr-title.yml`, `require-issue-link.yml`.

This is criterion 2's shape moved one level up: the new subject passes *because it is
absent*. Concrete: add `.github/workflows/security-scan.yml` with a job whose only
step is `run: go test ./... -tags=security` (a raw invocation duplicating a Taskfile
definition — the exact D-01 single-definition violation `TestWorkflowRunBodiesInvokeTask`
exists to catch). `TestInScopeJobsPopulationMatchesDisk` never opens the file because
it is not in `inScopeWorkflowFiles`; `TestWorkflowRunBodiesInvokeTask` iterates
`inScopeJobs`, which does not name it. Both stay GREEN and the whole suite stays
GREEN.

That the invariant is genuinely not universal today is checkable:
`.github/workflows/post-release-verify.yml:122` is a multi-line raw shell `run: |`
body, not a `task <target>` call.

**Fix:** Glob the directory instead of enumerating it, and demote the two D-01
exclusions to a validated exception record with the same shape
`usesOnlyJobExceptions` already uses:

```go
type workflowFileException struct{ Workflow, Reason string }

var workflowFileExceptions = []workflowFileException{
	{"bench.yml", "rebless/publish/diagnostic jobs run `go run ./tools/bench/runner` inline, commented in-file above the rebless job"},
	{"release.yml", "native build matrix, D-08"},
	// ... one entry per remaining out-of-scope workflow, each with a reason
}

files, err := filepath.Glob(filepath.Join(workflowsDir, "*.yml"))
// fail if a workflowFileExceptions entry names a file that no longer exists,
// then require every non-excepted globbed file's jobs to appear in inScopeJobs.
```

`requiredCheckNames` stays untouched — it mirrors an out-of-repo GitHub ruleset and is
correctly hand-written.

---

### WR-04: the status gate fires one GetStatus per keystroke; WR-08's fix hides the visible symptom without restoring the documented trigger contract

**File:** `web/src/routes/+layout.svelte:47-49`, `web/src/lib/status.ts:70-75` and
`:163-167`

**Issue:** `navigationIdentity(url)` is `pathname + sorted query string`, so it changes
whenever *any* query param changes — including `q`. The layout effect reads `page.url`
reactively and calls `statusGate.notifyNavigated(...)` on every change;
`notifyNavigated` fires a fresh `fetchStatus()` whenever the identity differs from the
last one. Typing in the Browse search box writes `q` to the URL on every keystroke by
design (D-11 REFINE, `+page.svelte:158-160` -> `browse-nav.ts:128`), which updates
`page.url`, which mints a new identity, which fires a `GetStatus` RPC.

Concrete: with the Browse view open, type `hello` into the search box. Five additional
`GetStatus` calls are issued, one per character, on top of the one the navigation
contract allows. Each one enters `withEngine` -> `openEngine` server-side and opens the
Pebble store; during an active re-index this repeatedly contends with the store lock
that `graphstore.Open`'s bounded retry budget guards, so the banner can flap into the
degrade path purely because a human is typing.

`status.ts`'s own module doc states the gate is "the ONE fetch trigger this app has for
'on load, on navigation, nothing else' ... deliberately a small seam, never a polling
subsystem." Per-keystroke firing violates that as written. WR-08's `requestId` guard is
correct in itself and I verified it works, but its effect here is to make the *visible*
consequence (a superseded response reverting the banner) invisible while leaving the
storm in place — a fix that quiets a symptom rather than the cause.

No guard binds this: `web/tests/status.test.ts:152-167` and `:169-210` only ever vary
the identity by `symbol` (`/browse?symbol=Foo` vs `/browse?symbol=Bar`), never by `q`.

**Fix:** Make the gate's identity the *target* identity, not the full query string —
`q` is explicitly view-local (`+page.svelte:64-66` says so). Either narrow
`navigationIdentity` to the params that select a view, or drop the view-local ones:

```ts
const VIEW_LOCAL_PARAMS = ['q'];

export function navigationIdentity(url: URL): string {
	const params = new URLSearchParams(url.searchParams);
	for (const k of VIEW_LOCAL_PARAMS) params.delete(k);
	params.sort();
	const qs = params.toString();
	return qs ? `${url.pathname}?${qs}` : url.pathname;
}
```

Add a test asserting `notifyNavigated('/browse?q=a')` after
`notifyNavigated('/browse?q=ab')` produces **one** fetch total, and keep the existing
`symbol`-varying test so the guard still discriminates in the direction that must
still fire.

---

### WR-05: IN-15's "planted positive control" exercises a copy of the comparison, not the function under test — WR-03's own defect shape, reintroduced in the same pass

**File:** `internal/upgrade/golangci_shape_test.go:90-148` (subject under test at
`:54-84`)

**Issue:** `TestGolangciEnabledLintersHeaderDiscriminates` declares itself "the planted
positive control (rule 84d1gfpywd): proves the comparison above can actually fail". It
does not exercise the comparison above. Lines 121-142 re-implement the regex match, the
`strconv.Atoi`, the YAML decode, the `len(Linters.Enable) + len(Formatters.Enable)` sum,
the `headerCount != actual` predicate, and even the failure message, against local
fixtures. `TestGolangciEnabledLintersHeaderMatchesConfig` is never called and there is
no extracted function both tests could share.

This is exactly what WR-03 fixed for `test/wireoracle/capture.go` earlier in this same
pass ("the test guards a copy of the code"), landed here at commit `dbad7ae1`.

Concrete: change `golangci_shape_test.go:78-83` from `t.Errorf(...)` to `t.Logf(...)`,
or change line 77 to `actual := len(cfg.Linters.Enable)` while bumping the header to 4.
The real guard stops discriminating; `TestGolangciEnabledLintersHeaderDiscriminates`
stays GREEN and still claims in its own doc comment to have proven otherwise.

**Fix:** Extract the comparison so both tests drive one implementation:

```go
func compareEnabledLintersHeader(src []byte) (headerCount, actual int, err error) { ... }

func TestGolangciEnabledLintersHeaderMatchesConfig(t *testing.T) {
	data, _ := os.ReadFile(golangciConfigPath)
	h, a, err := compareEnabledLintersHeader(data)
	// ... assert err == nil && h == a
}

func TestGolangciEnabledLintersHeaderDiscriminates(t *testing.T) {
	for _, c := range cases {
		h, a, err := compareEnabledLintersHeader([]byte(c.src)) // SAME function
		// ... assert (h != a) == c.wantMismatch
	}
}
```

## Info

### IN-01: IN-07 inserted `isResponseLine` inside `CanonicalizeResponseOrder`'s doc comment

**File:** `test/wireoracle/normalize.go:194-216`

**Issue:** The paragraph beginning `// CanonicalizeResponseOrder is 03-03-PLAN.md Task
3's R2 resolution ...` (lines 194-205) is now the doc comment attached to
`func isResponseLine` at line 217. `CanonicalizeResponseOrder`'s own remaining comment
(line 225 onward) no longer carries the R2 rationale that explains why the function
exists. godoc will attribute the go-sdk `jsonrpc2.Async` explanation to the wrong
symbol.

**Fix:** Move the `isResponseLine` block (lines 206-216) plus its function above line
194 so the R2 paragraph is reunited with `CanonicalizeResponseOrder`.

---

### IN-02: WR-03's `_ = scanTimestamped(...)` discards the scanner error with no justification, while IN-14 in the same pass required one everywhere else

**File:** `test/wireoracle/capture.go:396-398`

**Issue:** Behavior is unchanged from the pre-fix inline loop (which also never checked
`scanner.Err()`), so this is not a regression. But it is now an explicit `_ =` discard,
and IN-14 (`e57268d9`) established in this same pass that every such discard carries a
one-line reason pointing at the authoritative explanation. This one does not. If the
10 MiB `scanner.Buffer` cap is exceeded the goroutine closes `lines` silently and the
capture looks like a clean EOF, which surfaces downstream as a confusing `drainUntil`
timeout rather than "line too long".

**Fix:** Either add the one-line reason, or surface it — e.g. record the error on the
`Capture` result so a truncated stream fails as a truncated stream.

---

### IN-03: the WR-07 comment claims "every OTHER component of that URL is escaped"; `owner` and `repo` are written raw

**File:** `internal/uiserver/permalink.go:156-157`, sinks at `:243-245`

**Issue:** `buildGitHubBlobURL` percent-encodes the path per segment
(`percentEncodeRepoPath`) and now validates `sha`, but writes `owner` and `repo`
verbatim. A remote of `github.com:owner/re#po.git` yields `repo = "re#po"` and a
rendered URL of `https://github.com/owner/re#po/blob/<sha>/<path>`, which a browser
resolves as `https://github.com/owner/re` with everything after `#` as a fragment. The
host is pinned to `github.com` by `RemoteGitHubRepo`'s exact-equality check, so this
cannot cross an origin — it is a correctness wart and an inaccurate comment, not a
vulnerability.

**Fix:** Run `owner` and `repo` through `url.PathEscape` in `buildGitHubBlobURL`, or
correct the comment to say which components are escaped.

---

### IN-04: WR-06 broadened `parseGitHubRemote` so a bare local path with a colon before any slash is now parsed as a host

**File:** `internal/gitmeta/permalink.go:144-155`

**Issue:** Making the `user@` optional was the right fix, but it also removed the `@`
requirement that previously kept bare filesystem paths out of the scp-like branch.
`D:\src\myrepo` (or any local path with a colon before its first `/`) now yields
`host = "D"`, so `RemoteGitHubRepo` answers
`origin remote host "D" is not github.com` instead of the previous, more accurate
`origin remote is not a recognized URL`. Both are NO_LINK; only the operator-facing
sentence changes, and it can echo a fragment of a local path.

**Fix:** Reject a candidate host that contains a path separator or is a single
character, before returning it — or require the host to contain a `.` (git's own
scp-like grammar documents `host.xz`).

---

### IN-05: `gitSHA1HexLen` / `gitSHA256HexLen` now exist in two packages, bound only by a comment

**File:** `internal/indexer/commit.go:35-38` and `internal/schema/meta.go:58-61`

**Issue:** WR-07 promoted the *predicate* to `schema.IsCommitSHA` (good), but left a
second copy of the two length constants in `internal/indexer`, documented as
"numerically identical to schema's" (`commit.go:92-94`). Nothing asserts that. The
indexer's tests build fixtures from the local constants, so a change to schema's values
would be caught only indirectly.

**Fix:** Delete the indexer copies and reference `schema`'s (exporting them if the
indexer tests need them), or add a one-line compile-time assertion binding the pairs.

---

### IN-06: `GetStatus` still puts the unvalidated stored `commit_sha` on the wire

**File:** `internal/uiserver/handlers.go:324`

**Issue:** WR-07's own doc comment (`internal/schema/meta.go:63-76`) states the rule as
"callers that read a commit SHA out of a Meta record and pass it to either of those
[sinks] should call this first". `GetStatus` reads it with
`commitSHA, _ := schema.IndexedCommitSHA(meta)` and does not validate. That is
defensible today — the SPA renders it as escaped text (`web/src/routes/+page.svelte:91`)
and never builds a URL from it (`web/src/lib/status.ts:50` only tests truthiness) — so
there is no reachable sink. But the rule is applied at one of two read sites with
nothing binding it, so a future consumer of `GetStatusResponse.commit_sha` inherits the
hazard silently.

**Fix:** Validate in `GetStatus` too and emit `""` on failure (which already means
"unknown" per D-05), so the invariant "a commit SHA that leaves this server is
well-formed" holds at every exit.

---

### IN-07: `CopyAction`'s 1.5 s reset timer is never cleared on destroy

**File:** `web/src/lib/components/browse/CopyAction.svelte:44-46`

**Issue:** WR-09's fix schedules `setTimeout(..., 1500)` after every copy attempt and
clears it only on the *next* click. Unmounting within the window leaves the timer
pending; it later assigns `copyState` on a destroyed component. Harmless in Svelte 5 at
runtime, but under vitest's fake timers a later `advanceTimersByTime` in another test
can run it.

**Fix:** Add `$effect(() => () => clearTimeout(resetTimer));`.

---

### IN-08: WR-05's fix added an empty-input clear that was not part of its finding, and silently ignores non-conforming input

**File:** `web/src/lib/components/browse/NeighborsPanel.svelte:69-77`

**Issue:** WR-05's finding was the writer/parser grammar asymmetry, and
`isShapeInteger` fixes it correctly. The same edit also added a new behavior — an empty
input now writes `{ depth: undefined }`, clearing the URL param, where the previous
code did nothing. The input is `onchange` (line 137), not `oninput`, so this only fires
on commit and the blast radius is limited; but committing an empty value now drops
`depth` from `targetKey` (`+page.svelte:87-95`) and tears the whole node view down to
`loading` for a reload at the default depth. Separately, a committed non-conforming
value (`2.5` in a `type="number"` input) is discarded with no feedback, leaving the
control showing one depth and the URL another.

**Fix:** Keep the grammar check; either revert the empty branch to a no-op or make the
clear explicit in the UI (a "reset" affordance). Give the ignored-input case visible
feedback (`aria-invalid`, or reset the input to the URL's value on blur).

---

### IN-09: `createBrowseNavigator` ignores the promise `goto` returns

**File:** `web/src/lib/browse-nav.ts:128-132`

**Issue:** `gotoFn(...)` returns a `Promise<void>` that is neither awaited nor
`.catch`ed. A rejection (SvelteKit throws for `goto` during SSR, and navigation can be
aborted) becomes an unhandled rejection with no diagnostic, in the module documented as
"the ONE URL writer".

**Fix:** `void gotoFn(url, {...}).catch(() => {});` with a comment, or propagate the
promise out of `navigate` so callers can decide.

---

_Reviewed: 2026-08-29T13:59:52Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
_Iteration: 2 (post-fix re-review)_
