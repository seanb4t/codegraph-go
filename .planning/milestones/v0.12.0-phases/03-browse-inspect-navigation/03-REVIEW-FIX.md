---
phase: 03-browse-inspect-navigation
fixed_at: 2026-08-29T18:00:00Z
review_path: .planning/phases/03-browse-inspect-navigation/03-REVIEW.md
iteration: 3
findings_in_scope: 14
fixed: 14
skipped: 0
status: all_fixed
---

# Phase 3: Code Review Fix Report

**Fixed at:** 2026-08-29
**Source review:** .planning/phases/03-browse-inspect-navigation/03-REVIEW.md
**Iteration:** 3 (final — `--auto` loop's last allowed pass)

**Summary:**
- Findings in scope (Critical + Warning + Info, `--all`): 14 (WR-01 through WR-05, IN-01 through IN-09)
- Fixed: 14
- Skipped: 0

This is a re-review of a prior fix pass. Every one of the five Warnings was
caused by that prior pass rather than surviving from the original review, and
two of them (WR-03, WR-05) reproduced the exact defect class the prior pass
was itself fixing — a guard bound to a hardcoded population, and a
discriminator that exercises a copy of the code instead of the guard's own
code path. Both received close attention to make sure this pass did not
repeat the pattern a third time: WR-03 binds the workflow-file population to
a real directory glob (not a second hardcoded list), and WR-05 extracts one
shared comparison function that both the real guard and its discriminator
call, verified by sabotaging the shared function and observing BOTH tests go
red together.

**Verification baseline preserved** (observed numbers, not exit statuses):
- `task web:test` → **149 of 149** (was 142 at HEAD before this pass; +7 net)
- `GOTOOLCHAIN=go1.26.5 task test:unit` → **50 ok, 0 FAIL** (unchanged)
- `GOTOOLCHAIN=go1.26.5 task lint:go` → **0 issues**
- `task web:drift` → PASS, 73 source / 27 output files, both digest halves match
- `task proto:drift` → compared 4 generated files, all byte-identical
- `task lint:actions`, `GOTOOLCHAIN=go1.26.5 go vet ./...`, `cd web && pnpm check` → all clean (0 errors/warnings)
- `GOTOOLCHAIN=go1.26.5 go test ./internal/upgrade/ -count=1 -v` → **122 named PASS, 0 FAIL** (was 121; +1, WR-03's new guard)
- Wire oracle → **42 on disk, 42 named subtests PASS** (unchanged)
- `git status --porcelain` → clean except the pre-existing untracked `.planning/milestone.lock`

**Environment:** `workflow.use_worktrees=false` for this project — all edits and
commits were made directly in the main checkout on `gsd/v0.12.0-local-graph-ui`
(no worktree was created; the cleanup tail is a no-op in this mode).

**Rule 84d1gfpywd compliance:** every fix that touches or adds a guard was
proven to discriminate — a violation was planted (either the finding's own
concrete repro, or a hand-crafted one), the guard was observed to go RED, the
violation was reverted, and the guard was observed to return to GREEN. One
discrimination attempt (IN-09) initially relied on vitest's global
unhandled-rejection detection and did NOT discriminate reliably (it failed
the overall run's exit code without failing the specific test); it was
replaced with a direct, environment-independent proof (a fake thenable
recording whether `.catch()` was invoked) before being accepted. Running the
full web suite (not just the touched test file) for IN-09 also caught a real,
independent latent bug: a test mock's `goto` stand-in did not return a
Promise, unlike the real function its type claims to satisfy — fixed
alongside the finding itself and noted in that entry below.

## Fixed Issues

### WR-01: SearchPanel's search box stopped resyncing on non-typing URL changes

**Files modified:** `web/src/lib/components/browse/SearchPanel.svelte`, `web/tests/search-panel.test.ts`, `web/build/`
**Commit:** `1e864095`
**Applied fix:** IN-12's `untrack()` fix (previous pass) made the seed effect
run exactly once at mount, so a `q` change reaching the panel from anything
OTHER than its own typing (concretely: browser Back/Forward replaying an
earlier history entry) could no longer resync the box — an over-correction.
Tracked `initialQuery` again, but skip the reseed only when it already
equals the controller's CURRENT query (`searchState.query`), which
`handleInputChange` already sets synchronously on every keystroke — making
typing a no-op here exactly as IN-12 intended, without blinding the effect
to non-typing changes.
**Discrimination proof:** Updated IN-12's own pinned test (which asserted a
prop change must NEVER re-seed, without distinguishing typing from URL
navigation) to actually simulate typing before the prop change, and added a
new test for the back/forward case. Reverting the fix → RED on the new
back/forward test (`expected 'beta' to be 'alpha'` — the exact
address-bar/rendered-view disagreement WR-01 describes). Restoring → GREEN,
13/13 in the file.

### WR-02: the malformed-SHA NO_LINK branch reused a false reason string

**Files modified:** `internal/uiserver/permalink.go`, `internal/uiserver/permalink_test.go`
**Commit:** `2b688f64`
**Applied fix:** WR-07's malformed-SHA check (previous pass) reused
`noCommitSHAReason`, which is false for that branch (the field IS present;
the store is not a pre-upgrade graph) and collapsed two distinct causes into
one wire string — the exact conflation `notObservedReason`/
`checkUnknownReason` exist three lines up to prevent. Added
`malformedCommitSHAReason` and pointed the branch at it.
**Discrimination proof:** Changed the test to assert the exact
`malformedCommitSHAReason` string AND that it differs from
`noCommitSHAReason`. Reverting the fix → RED (`reason = "this index has no
recorded commit SHA..."`, the wrong string). Restoring → GREEN, both
`TestGetPermalink_NoLink*` tests pass.

### WR-03: the workflow-file population was itself a hardcoded 3-of-14 list

**Files modified:** `internal/upgrade/taskfile_shape_test.go`
**Commit:** `75f5668f`
**Applied fix:** `inScopeWorkflowFiles` named only 3 of 14 files under
`.github/workflows/`, with no disk binding — the file-population version of
the exact defect IN-05 fixed one level down (job population) in the prior
pass. Added `workflowFileExceptions` (`workflowFileException{Workflow,
Reason}`), one entry per file not in `inScopeWorkflowFiles`, each with an
honest, specific reason: bot/PR automation with no Taskfile-target-
duplicating `run:` body, a native build matrix (D-08), or — for the two CI
canary workflows (`darwin-toolchain-canary.yml`, `linux-cross-canary.yml`) —
an explicit, undisguised statement that they are not yet brought under
per-job D-01 enforcement in this pass, deliberately scoped this way to avoid
a blind audit of two substantial, previously-unaudited matrix workflows
under a final review iteration. `validateWorkflowFileExceptions` checks
every entry still exists on disk. `TestWorkflowFilePopulationMatchesDisk`
requires every `*.yml` file on disk to appear in exactly one of
`inScopeWorkflowFiles` or `workflowFileExceptions`. `requiredCheckNames` was
NOT touched — it mirrors an out-of-repo GitHub ruleset and stays
hand-written per existing project decision (binding constraint).
**Discrimination proof:** Added a scratch `security-scan.yml` — WR-03's own
concrete repro, a raw `run: go test ./... -tags=security` job — → RED,
naming it exactly (`workflow file(s) on disk named in NEITHER
inScopeWorkflowFiles nor workflowFileExceptions: [security-scan.yml]`).
Removed the scratch file → GREEN. Separately verified a stale exception
entry (renamed to a nonexistent file) fails loudly via
`validateWorkflowFileExceptions`.

### WR-04: the status gate fired one GetStatus per keystroke

**Files modified:** `web/src/lib/status.ts`, `web/tests/status.test.ts`, `web/build/`
**Commit:** `f3cda54e`
**Applied fix:** `navigationIdentity` included every query param, so typing
in Browse's search box (`q`, written on every keystroke, undebounced, per
D-11) minted a fresh navigation identity per character and fired one extra
`GetStatus` RPC per keystroke — each re-entering `withEngine`/`openEngine`
and contending with the store lock during an active re-index. WR-08's
`requestId` guard (previous pass) hid the visible symptom (a superseded
response reverting the banner) but left the request storm itself in place.
Excluded `q` via a new `VIEW_LOCAL_PARAMS` list, mirroring the same
"`q` is view-local" reasoning CR-01 already applied to the Browse load
effect.
**Discrimination proof:** Added a paired test: varying only `q` across five
values produces the same identity and fires no extra `GetStatus`, while
varying a target field still does. Reverting the fix → RED on both the new
`navigationIdentity` unit test and the `createStatusGate`-level test
(`expected "vi.fn()" to be called 1 times, but got 5 times` — the exact
per-keystroke storm). Restoring → GREEN, 16/16 in the file.

### WR-05: the enabled-linters discriminator exercised a copy of the guard's code

**Files modified:** `internal/upgrade/golangci_shape_test.go`
**Commit:** `015405fc`
**Applied fix:** `TestGolangciEnabledLintersHeaderDiscriminates` (IN-15,
previous pass) re-implemented the regex match, `strconv.Atoi`,
`yaml.Unmarshal`, the count comparison, and the failure message locally
against synthetic fixtures instead of calling
`TestGolangciEnabledLintersHeaderMatchesConfig`'s own code — proving a
re-implementation could fail, not that the guard could. This is the same
defect shape WR-03 (of the previous pass) fixed for
`test/wireoracle/capture.go`. Extracted `compareEnabledLintersHeader`
(the comparison) and `enabledLintersHeaderMismatch` (the message) so both
tests call the SAME functions.
**Discrimination proof:** Sabotaged `compareEnabledLintersHeader` itself
(dropped the `+ len(cfg.Formatters.Enable)` term) → BOTH
`TestGolangciEnabledLintersHeaderMatchesConfig` AND
`TestGolangciEnabledLintersHeaderDiscriminates` went RED together — proving
the coupling holds, unlike the pre-fix shape where only the real guard
would have caught it. Restoring → GREEN, full `internal/upgrade` package
122 named PASS.

### IN-01: `isResponseLine` absorbed `CanonicalizeResponseOrder`'s R2 doc comment

**Files modified:** `test/wireoracle/normalize.go`
**Commit:** `d0535e11`
**Applied fix:** IN-07 (previous pass) inserted `isResponseLine` between the
R2-rationale paragraph and `CanonicalizeResponseOrder`'s own doc block, so
godoc attributed the R2 explanation to `isResponseLine` instead. Moved
`isResponseLine` (with only its own paragraph) above the R2 paragraph,
which now sits contiguous with `CanonicalizeResponseOrder`'s existing doc
block, directly above the func it explains.
**Verification:** Doc-only change (no logic touched). `go doc
./test/wireoracle CanonicalizeResponseOrder` now shows the R2 paragraph as
that func's own doc. Full `test/wireoracle` package test run unaffected.

### IN-02: `Capture`'s scanner error was discarded with no justification

**Files modified:** `test/wireoracle/capture.go`
**Commit:** `927efa7e`
**Applied fix:** `Capture`'s stdout goroutine discarded
`scanTimestamped`'s return with `_ =` and no inline reason — IN-14
(previous pass) established that every such discard in this phase carries a
one-line justification, and this one had none. If the 10 MiB
`scanner.Buffer` cap is exceeded, the goroutine closed `lines` silently and
looked exactly like a clean EOF. Recorded the error in `scanErr` (written
once by the goroutine, strictly before its deferred `close(lines)` — a
channel close happens-after every preceding statement in the same
goroutine) and folded it into `drainUntil`'s existing "stdout closed" error
text when present.
**Verification:** No new guard/discrimination proof applicable (behavior
change, not a new test coupling). Verified with `go test -race` on the
capture-driving tests — no data race — and the full `test/wireoracle`
package test run.

### IN-03: `buildGitHubBlobURL` did not escape `owner`/`repo`

**Files modified:** `internal/uiserver/permalink.go`, `internal/uiserver/permalink_test.go`
**Commit:** `dc94a6f0`
**Applied fix:** WR-07's own comment (previous pass) claimed "every OTHER
component of that URL is escaped", but `owner` and `repo` were written
verbatim. A remote of `github.com:owner/re#po.git` yields `repo = "re#po"`,
rendering a URL a browser resolves with everything after `#` read as a
fragment — a correctness defect (not a vulnerability; `RemoteGitHubRepo`'s
exact-equality host check prevents any origin crossing). Applied
`url.PathEscape` to both and corrected the doc comment.
**Discrimination proof:** Added `TestBuildGitHubBlobURL_PercentEncodesOwnerAndRepo`
driving the function directly. Reverting the fix → RED (`re#po` present raw
in the URL). Restoring → GREEN, full `internal/uiserver` package passes.

### IN-04: the scp-like remote parser misparsed a bare local path as a host

**Files modified:** `internal/gitmeta/permalink.go`, `internal/gitmeta/permalink_test.go`
**Commit:** `282e4cf2`
**Applied fix:** WR-06 (previous pass) made the scp-like syntax's `user@`
prefix optional, which also removed the implicit "`@` must be present" gate
that had kept bare filesystem paths out of that branch. A Windows-style
local path with a colon before its first path separator (a drive letter,
e.g. `D:\src\myrepo`) parsed as `host="D"`, echoing a fragment of a local
path into `RemoteGitHubRepo`'s Reason text. Rejected a candidate host that
is a single character or contains a path separator, before returning it.
**Discrimination proof:** Added a negative test for three local-path shapes
(`D:\src\myrepo`, `C:/src/myrepo`, `c:\Users\me\repo`) and a positive
control proving a genuinely short (2-char) valid scp-like host still
parses. Reverting the fix → RED on all three negative cases (`host="D"`,
`host="C"`, `host="c"`). Restoring → GREEN, full `internal/gitmeta` package
passes.

### IN-05: the git-SHA hex-length constants existed in two packages

**Files modified:** `internal/schema/meta.go`, `internal/indexer/commit.go`, `internal/indexer/commit_test.go`
**Commit:** `28945d46`
**Applied fix:** `internal/indexer/commit.go` kept its own
`gitSHA1HexLen`/`gitSHA256HexLen` copies alongside `schema`'s promoted
versions (WR-07, previous pass), documented only as "numerically identical
to schema's" with nothing asserting that stayed true. Exported `schema`'s
constants (`SHA1HexLen`, `SHA256HexLen`) and deleted the indexer package's
copies entirely; its tests now reference `schema.SHA1HexLen`/
`schema.SHA256HexLen` directly.
**Verification:** Not a new guard (a duplicate-definition removal, which
makes drift definitionally impossible rather than merely detectable).
`go build ./...` and `go vet ./...` clean; `internal/indexer` and
`internal/schema` package test runs both pass.

### IN-06: `GetStatus` put an unvalidated stored commit SHA on the wire

**Files modified:** `internal/uiserver/handlers.go`, `internal/uiserver/handlers_test.go`
**Commit:** `1c76754c`
**Applied fix:** `schema.IsCommitSHA`'s own doc comment (WR-07, previous
pass) states the rule as "callers that read a commit SHA out of a Meta
record ... should call this first", but `GetStatus` read it via
`schema.IndexedCommitSHA` and never validated. No reachable sink exists
today (the SPA renders it as escaped text and never builds a URL from it),
so this closes a latent hazard rather than a live one. Degrades an
unvalidated value to `""` (== unknown, per D-05), the same fallback
`GetPermalink`'s malformed-SHA branch uses.
**Discrimination proof:** Added a subtest reusing `overwriteCommitSHA`
(permalink_test.go, same package) with the same hostile value
`GetPermalink`'s own malformed-SHA test drives. Reverting the fix → RED
(`GetStatus commit_sha = "../../attacker/attacker-repo/blob/main"`, the raw
hostile value on the wire). Restoring → GREEN, full `internal/uiserver`
package passes.

### IN-07: `CopyAction`'s reset timer was never cleared on destroy

**Files modified:** `web/src/lib/components/browse/CopyAction.svelte`, `web/tests/source-pane.test.ts`, `web/build/`
**Commit:** `669e04d2`
**Applied fix:** WR-09's fix (previous pass) scheduled a 1.5s `setTimeout`
after every copy attempt and only cleared it on the NEXT click. Unmounting
within that window left the timer pending. Added a cleanup-only `$effect`
clearing `resetTimer` on unmount.
**Discrimination proof:** Test proves the timer is actually gone via
`vi.getTimerCount()` (not merely that nothing throws — Svelte 5 makes the
post-destroy assignment harmless at runtime). Reverting the fix → RED
(`expected 1 to be +0`, the pending timer still scheduled after unmount).
Restoring → GREEN, 11/11 in the file.

### IN-08: a non-conforming committed depth value had no visible feedback

**Files modified:** `web/src/lib/components/browse/NeighborsPanel.svelte`, `web/tests/neighbors-panel.test.ts`, `web/build/`
**Commit:** `b96135fc`
**Applied fix:** WR-05's grammar-check fix (previous pass) silently
discarded a non-conforming committed value with no feedback at all. Added
`depthInvalid` state, cleared on any successful commit (including the
explicit empty-clears-depth case) and set on a non-conforming commit,
surfaced via `aria-invalid` on the input. Left the empty-clears-depth
behavior itself untouched — that was WR-05's own prior, already-tested
addition, and reverting it is a separate judgment call this fix does not
relitigate (the finding offered both options; this is the narrower,
lower-risk one).
**Discrimination proof:** Reverting the fix → RED (`aria-invalid`
attribute absent entirely). Restoring → GREEN, 9/9 in the file.

### IN-09: `createBrowseNavigator`'s `navigate()` ignored the promise `goto` returns

**Files modified:** `web/src/lib/browse-nav.ts`, `web/tests/browse-nav.test.ts`, `web/tests/browse-page.test.ts`, `web/build/`
**Commit:** `d11ef770`
**Applied fix:** `gotoFn(...)` returns `Promise<void>`, neither awaited nor
`.catch()`ed. Added `void gotoFn(...).catch(() => {})` — fire-and-forget by
design.
**Discrimination proof:** An initial attempt relying on vitest's global
unhandled-rejection detection did NOT discriminate reliably (verified: it
failed the overall test run's exit code without failing the specific test
— non-deterministic timing/attribution in this environment). Replaced with
a direct proof: a fake thenable recording whether its own `.catch()` was
invoked. Reverting the fix → RED (`catchCalled` stayed `false`). Restoring
→ GREEN.
**Independent bug caught by full-suite verification:** Running the FULL web
suite (not just the touched test file) after this fix crashed with an
uncaught exception in `browse-page.test.ts`: its `$app/navigation` mock's
`goto` stand-in returned `undefined` instead of a `Promise`, unlike the
real `goto` its `GotoFn` type claims to satisfy — invisible before this fix
because nothing previously called a Promise method on the return value.
Fixed the mock to `return Promise.resolve()`, matching the real contract.
This is exactly why the fixer's verification step runs the full suite, not
only the file touched by a given finding.

## Skipped Issues

None — all 14 in-scope findings were fixed.

---

_Fixed: 2026-08-29_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 3_
