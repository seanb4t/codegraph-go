---
phase: 03-browse-inspect-navigation
fixed_at: 2026-08-29T13:45:00Z
review_path: .planning/phases/03-browse-inspect-navigation/03-REVIEW.md
iteration: 1
findings_in_scope: 23
fixed: 22
skipped: 0
status: all_fixed
---

# Phase 3: Code Review Fix Report

**Fixed at:** 2026-08-29
**Source review:** .planning/phases/03-browse-inspect-navigation/03-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope (per orchestrator's already-fixed exclusions): 23 (WR-01 through WR-09, IN-01 through IN-03, IN-05 through IN-15)
- Fixed: 22
- Recorded as `no_change_needed` (documented, non-defect): 1 (IN-09)
- Skipped: 0

Two findings were verified as already fixed by a maintainer-authorized gap-closure
pass before this run and were **not touched**: CR-01 (commit `6ff79d4a`) and IN-04
(commit `77b75129`). Both were re-verified present and correct; see below.

One finding, **WR-04**, was reproduced as OVERSTATED by the orchestrator's own
pre-run investigation (a test-coverage gap, not a live security hole) and was fixed
accordingly — as a coverage addition, not an escaping-logic rewrite.

One finding, **WR-07**, had its review-suggested fix partially investigated and
REJECTED: passing `--` to `git branch --contains` was empirically verified to be
both unnecessary (git already treats the next argv token as `--contains`'s value
regardless of a leading `-`) and actively harmful (it breaks the command outright,
since git then tries to resolve the literal `--` as an object). This is documented
in the WR-07 commit rather than applied blindly.

**Verification baseline preserved** (observed numbers, not exit statuses):
- `task web:test` → **142 of 142** (was 128/128; +14 new tests across the fix set)
- `GOTOOLCHAIN=go1.26.5 task test:unit` → **50 ok, 0 FAIL** (unchanged from baseline)
- `GOTOOLCHAIN=go1.26.5 task lint:go` → **0 issues**
- `task web:drift` → PASS, 73 source files / 27 output files, both digest halves match
- `task proto:drift` → compared 4 generated files, all byte-identical
- `task lint:actions`, `GOTOOLCHAIN=go1.26.5 go vet ./...`, `cd web && pnpm check` → all clean (0 errors/warnings)
- `GOTOOLCHAIN=go1.26.5 go test ./internal/upgrade/ -count=1 -v` → **121 named PASS, 0 FAIL** (was 118; +3 new tests)
- Wire oracle → **42 named PASS, 0 FAIL** (unchanged; 42 frozen transcripts on disk at `testdata/wireoracle/transcripts/`)
- `git status --porcelain` → clean except the pre-existing untracked `.planning/milestone.lock`

**Rule 84d1gfpywd compliance:** every fix below that repairs or adds a guard was
proven to discriminate — a violation was planted, the guard was observed to go RED,
the violation was reverted, and the guard was observed to return to GREEN. Where the
guard is a runtime test this was done by editing source and re-running vitest/go
test; where the guard is a compile-time type constraint (IN-02) this was done by
attempting the prohibited assignment and observing `go build` fail. Every such
RED/GREEN pair is recorded in the corresponding commit message and summarized below.

## Already Fixed (verified, not touched)

### CR-01: keystroke-driven view teardown
**Commit:** `6ff79d4a` (pre-existing, maintainer-authorized)
**Verified:** `web/src/routes/browse/+page.svelte` derives `targetKey` via
`JSON.stringify` of the five target fields and reads `params` through `untrack()`
(lines 73-97). `web/tests/browse-page.test.ts` exists and backs this. Not altered.

### IN-04: lint-go not gating merge
**Commit:** `77b75129` (pre-existing, maintainer-authorized)
**Verified:** `task lint:go` is folded into the required `test` job at
`.github/workflows/ci.yml:90` ("Lint (golangci-lint via task lint:go)" step). No
standalone `lint-go` job exists. `requiredCheckNames` in
`internal/upgrade/taskfile_shape_test.go` was NOT edited (mirrors an out-of-repo
GitHub ruleset). Not altered.

## Fixed Issues

### WR-01: `decorateCallTargets` teardown resurrects stale text

**Files modified:** `web/src/lib/call-targets.ts`, `web/tests/call-targets.test.ts`, `web/build/`
**Commit:** `74c8f5e7`
**Applied fix:** The teardown fallback (`parent.appendChild(originalText)` when the
recorded anchor was no longer a child of `parent`) injected a previous render's
source text into a subtree Svelte's `{@html}` had already replaced. Changed the
fallback to drop the stale nodes silently instead of re-inserting into DOM the
decorator no longer owns.
**Discrimination proof:** Added a test simulating the owner-replaced-subtree case
directly. Reverting the fix → RED (`'Baz and QuxFoo and Bar'` vs expected
`'Baz and Qux'`). Restoring → GREEN.

### WR-02: `EXTENSION_LANGUAGE` hand-enumerated population, no disk binding

**Files modified:** `internal/indexer/languages.go`, `web/src/lib/highlight.ts`,
`web/src/lib/components/browse/SourcePane.svelte`,
`web/highlight_extension_coverage_test.go` (new), `web/build/`
**Commit:** `a36ece6b`
**Applied fix:** Exported `RegisteredLanguageExtensions()` from `internal/indexer`
(the disk-backed source of truth). Moved `EXTENSION_LANGUAGE` into `highlight.ts` as
a text-parseable literal declaration. Added a Go guard parsing it as text (Go cannot
import TypeScript) and asserting set equality — key AND value — against the
registry, with a planted-fixture discriminates test.
**Discrimination proof:** `TestExtensionLanguageComparisonDiscriminates` proves the
comparison can fail in all three directions (missing/extra/mismatched) against a
planted fixture; `TestExtensionLanguageCoversRegisteredExtensions` passes against
the real tree (23/23 extensions both sides).

### WR-03: `TestCaptureArrivalLedgerPreservesWireOrder` guards a copy of the code

**Files modified:** `test/wireoracle/capture.go`, `test/wireoracle/capture_test.go`
**Commit:** `f60e0143`
**Applied fix:** Extracted `scanTimestamped(r io.Reader, emit func(ArrivalLine)) error`
as the ONE scan-and-timestamp primitive. `scanArrivalLines` now wraps it. `Capture`'s
stdout goroutine now calls `scanTimestamped` directly instead of an inline copy.
**Discrimination proof (two-step, unusual for this finding since the defect was
"tested code ≠ running code"):** (1) Reverted to pre-fix code, sabotaged Capture's
inline goroutine to buffer-then-reverse lines — `TestCaptureArrivalLedgerPreservesWireOrder`
still PASSED, reproducing the exact vacuity WR-03 describes (the test never touches
Capture's own code path). (2) Restored the fix, applied the identical sabotage to
`scanTimestamped` (now the shared primitive) — the same test went RED, naming the
exact reordering. Reverted sabotage → GREEN. Full package including
`TestFrozenTranscriptsMatch`'s 42 real-subprocess scenarios still passes.

### WR-04: `{@html}` XSS guard — TEST COVERAGE GAP, not a live hole (orchestrator-corrected)

**Files modified:** `web/tests/browse-tracer.test.ts`
**Commit:** `4f37c89e`
**Applied fix:** Per the orchestrator's own pre-run investigation, this was
overstated in the review — `browse-tracer.test.ts:127-129` already positively
asserts escaping for the unregistered-language fallback, and hljs escapes correctly
on the registered-language path too (verified empirically). Added a test feeding
hostile input through the REGISTERED `'go'` path with both a positive
(`toContain('&lt;script&gt;')`) and a non-vacuous negative
(`not.toContain('<script>')`) assertion. No production code changed — the existing
escaping was already correct.
**Discrimination proof:** Temporarily patched `highlightSource` to strip hljs's
escaping on the registered path → RED. Reverted → GREEN.

### WR-05: depth writer/parser asymmetry

**Files modified:** `web/src/lib/browse-url.ts`,
`web/src/lib/components/browse/NeighborsPanel.svelte`,
`web/tests/neighbors-panel.test.ts`, `web/build/`
**Commit:** `1bff2370`
**Applied fix:** Exported `isShapeInteger()` from `browse-url.ts` (one grammar, not
two that can drift). `handleDepthChange` now validates against it before writing to
the URL; a non-conforming value (`2.5`, `1e3`) is never written; an empty value
explicitly clears the param.
**Discrimination proof:** Reverting the fix → RED (`onNavigate` called with
`{ depth: 2.5 }`). Restoring → GREEN.

### WR-06: user-less scp-like GitHub remote misclassified

**Files modified:** `internal/gitmeta/permalink.go`, `internal/gitmeta/permalink_test.go`
**Commit:** `4ef1a225`
**Applied fix:** `parseGitHubRemote`'s no-scheme branch now strips a leading `user@`
if present, then applies the existing colon-before-slash rule to what remains —
making the `@` optional per git's own documented grammar
(`[user@]host.xz:path/to/repo.git/`).
**Discrimination proof:** Added `TestRemoteGitHubRepo_SCPLikeSyntaxNoUser`. Reverting
the fix → RED (`Reason: "origin remote is not a recognized URL"`). Restoring →
GREEN, full package 19/19.

### WR-07: unvalidated commit SHA reaches git CLI + rendered URL

**Files modified:** `internal/schema/meta.go`, `internal/indexer/commit.go`,
`internal/uiserver/permalink.go`, `internal/uiserver/permalink_test.go`
**Commit:** `ed9a0eaf`
**Applied fix:** Promoted `isLowercaseHexCommitSHA` to `schema.IsCommitSHA` (the one
validator applied at write time). `GetPermalink` now validates the stored SHA
immediately after reading it, degrading to `NO_LINK` before it can reach
`buildGitHubBlobURL` or `gitmeta.CommitOnRemoteTrackingBranch`.
**Rejected sub-fix (documented, not applied):** the review's suggestion to add `--`
to `git branch --contains sha` was investigated and found to be both unnecessary
(git already treats the next argv token as `--contains`'s value regardless of a
leading `-` — empirically verified with git 2.54.0: `--contains -o`, `--contains
--help`, and `--contains --` all fail with "malformed object name `<token>`", never
a flag reinterpretation) and actively harmful (adding `--` makes git resolve the
literal `--` itself as the object name, breaking the command). This is documented
inline in `internal/gitmeta/permalink.go` and in the commit message.
**Discrimination proof:** `overwriteCommitSHA` test helper bypasses the indexer's
write-time validation to simulate a foreign-written store; a hostile SHA
(`"../../attacker/attacker-repo/blob/main"`) is fed through. Reverting the
read-boundary check → RED (`availability = LINKABLE_UNVERIFIED`, hostile value
would reach the URL). Restoring → GREEN.

### WR-08: `createStatusGate` no cancellation/response-identity guard

**Files modified:** `web/src/lib/status.ts`, `web/tests/status.test.ts`, `web/build/`
**Commit:** `daf360f5`
**Applied fix:** Minted a monotonic request id per fetch; drops any settled response
(success or rejection) whose id no longer matches the most recent fetch.
**Discrimination proof:** Test resolves the SECOND (fresher) fetch first, then the
FIRST (stale) fetch. Reverting the fix → RED (final verdict reverts to `'stale'`).
Restoring → GREEN, 13/13.

### WR-09: `CopyAction` swallows clipboard failures

**Files modified:** `web/src/lib/components/browse/CopyAction.svelte`,
`web/tests/source-pane.test.ts`, `web/build/`
**Commit:** `20d8efe5`
**Applied fix:** Added `copyState` (`'idle' | 'copied' | 'failed'`) surfaced on the
button text, with try/catch around `navigator.clipboard.writeText`, an explicit
check for `navigator.clipboard` being undefined, and auto-reset after 1.5s.
**Discrimination proof:** Reverting the fix reproduced the ORIGINAL bug exactly — 3
tests fail, including two unhandled-rejection errors vitest surfaces explicitly.
Restoring → GREEN, 10/10.

### IN-01: `//nolint:staticcheck` disables all staticcheck classes, not just ST1005

**Files modified:** `internal/uiserver/degrade.go`
**Commit:** `0793f942`
**Applied fix:** Replaced `errors.New(indexingInProgressMessage)` (with a blanket
nolint) with `indexingInProgressError{}`, a named error type whose `Error()` method
returns the message — ST1005 only inspects string literals passed to
`errors.New`/`fmt.Errorf`, so this sidesteps the check by construction, removing the
nolint directive entirely. 0 nolint directives remain in the file.
**Discrimination proof:** Temporarily reverted the call site back to
`errors.New(indexingInProgressMessage)` (keeping the nolint removed) → `task lint:go`
fires ST1005 again, proving the directive was doing real suppression work. Restored
→ 0 lint issues.

### IN-02: `classifiedErr` is a second, unaudited error path

**Files modified:** `internal/uiserver/permalink.go`
**Commit:** `96483ac0`
**Applied fix:** Narrowed `classifiedErr`'s type from `error` to `*connect.Error`.
`connect.NewError` already returns `*connect.Error`, so the one legitimate
assignment site is unaffected; any future assignment of a raw `error` now fails to
compile.
**Discrimination proof:** Temporarily added `classifiedErr = verr` (a raw engine
error) → `go build` failed with "cannot use verr (variable of interface type error)
as *connect.Error value in assignment: need type assertion" — exactly the
compile-time backstop intended. Reverted → builds and tests clean.

### IN-03: `TestToolModfilesPopulationMatchesDisk` is count-only

**Files modified:** `internal/upgrade/taskfile_shape_test.go`
**Commit:** `6c5b971f`
**Applied fix:** Sort both sides and compare with `slices.Equal` instead of a bare
`len() != len()` count.
**Discrimination proof:** Swapped one real path in `isolatedModfilePaths` for a
same-cardinality but WRONG bogus path — the OLD count-only logic stayed GREEN
(proving the exact gap IN-03 describes); the NEW `slices.Equal` logic went RED
immediately, naming the mismatch. Reverted the temporary swap; real fix committed
clean.

### IN-05: `inScopeJobs` hand-enumerated, no disk binding

**Files modified:** `internal/upgrade/taskfile_shape_test.go`
**Commit:** `064613c5`
**Applied fix:** `TestInScopeJobsPopulationMatchesDisk` globs every job declared in
the same three workflow files `inScopeJobs` already covers and asserts set equality,
modulo a new `usesOnlyJobExceptions` carve-out (mirroring `runBodyException`'s own
shape) for the two jobs that are legitimately `uses:`-only with no `run:` body
(`govulncheck`, `release-please`) — each exception itself verified to still exist
and still carry zero `run:` steps. Per the maintainer's explicit instruction,
`requiredCheckNames` was NOT touched (mirrors an out-of-repo GitHub ruleset).
**Discrimination proof:** Injected a synthetic job into a scratch copy of `ci.yml` →
FAIL, naming `"ci.yml/fake-new-job"` exactly. Restored the real file → PASS.

### IN-06: `IDENTIFIER_PATTERN` exported with mutable shared `g`-flagged state

**Files modified:** `web/src/lib/call-targets.ts`, `web/tests/call-targets.test.ts`, `web/build/`
**Commit:** `ea463900`
**Applied fix:** Changed the exported const to a factory `identifierPattern()`
returning a fresh regex on every call.
**Discrimination proof:** Reverted to a single shared instance behind the same
function signature → new test RED AND cross-contaminated unrelated tests in the
same file (a live demonstration of the exact failure mode). Restored → 15/15 GREEN.

### IN-07: `CanonicalizeResponseOrder` misclassifies requests as responses

**Files modified:** `test/wireoracle/normalize.go`, `test/wireoracle/normalize_test.go`
**Commit:** `9fb52c93`
**Applied fix:** `isResponseLine(raw)` requires id present AND method absent
(`frameMethod` already existed for exactly this distinction).
`CanonicalizeResponseOrder` now classifies with `isResponseLine` instead of a bare
`responseID` check.
**Discrimination proof:** Test with a request frame between two genuine responses.
Reverting `isResponseLine`'s call site back to bare `responseID` → RED (the request
frame gets swapped with the id:2 response). Restoring → GREEN. Full package
including 42 real-subprocess scenarios unaffected.

### IN-08: `RemotePresence` `default` arm asserts "not observed"

**Files modified:** `internal/uiserver/permalink.go`, `internal/uiserver/permalink_test.go`
**Commit:** `9a38838c`
**Applied fix:** `RemotePresenceNotObserved` is now the explicit case; `default`
falls back to `checkUnknownReason`'s honest "could not verify" wording. Extracted
`remotePresenceResponse(blobURL, presence)` so the default arm is independently
unit-testable with a synthetic out-of-range value.
**Discrimination proof:** Reverted the default arm's Reason back to
`notObservedReason` → 2 tests RED, one naming exactly the forbidden outcome.
Restored → GREEN.

### IN-09: tri-state `RemotePresence` collapses to two wire values — `no_change_needed`

**Recorded as:** documented, non-defect design.
**Evidence:** The review's own text states: "This is the recorded and frozen design
(the enum shape was fixed at 03-05's blocking-human checkpoint, and the code
documents the choice at :38-47), so it is not a defect." No code change made.
Verified `internal/uiserver/permalink.go:38-47`'s doc comment still documents this
choice, unchanged by any fix in this pass.

### IN-10: `GetPermalink` no validation of `line`/`end_line`

**Files modified:** `internal/uiserver/permalink.go`, `internal/uiserver/permalink_test.go`
**Commit:** `0a7cff85`
**Applied fix:** Validates `line >= 1` and `end_line >= line` (plus `end_line` set
without `line`) BEFORE `withEngine`, returning `connect.CodeInvalidArgument`
directly — sidestepping the `classifiedErr` reclassification hazard IN-02 already
documents, since this validation has no Engine dependency.
**Discrimination proof:** Three tests (line=-1, end_line=0 with line=42, end_line
set with line unset). Reverting the validation → all 3 RED ("expected a non-nil
error, got nil"). Restoring → GREEN.

### IN-11: duplicate `data-testid` across NeighborsPanel regions

**Files modified:** `web/src/lib/components/browse/NeighborsPanel.svelte`,
`web/tests/neighbors-panel.test.ts`, `web/build/`
**Commit:** `9a094dc6`
**Applied fix:** Prefixed each region's testid with its own name
(`neighbor-caller-`, `neighbor-callee-`, `neighbor-blast-`) instead of the shared
`neighbor-entry-`.
**Discrimination proof:** Test with a node appearing as both a callee and a
blast-radius entry with the identical key. Reverting to the shared prefix → 3 tests
RED, including "Unable to fire a click event" from `querySelector` no longer
resolving uniquely. Restoring → GREEN, 8/8.

### IN-12: SearchPanel's "seed once" effect re-runs on every `q` change

**Files modified:** `web/src/lib/components/browse/SearchPanel.svelte`,
`web/tests/search-panel.test.ts`, `web/build/`
**Commit:** `0bba508a`
**Applied fix:** Read `initialQuery` through `untrack()` so the seed effect has zero
tracked dependencies and runs exactly once, at mount.
**Discrimination proof:** Test advances to just under the debounce window, rerenders
with a different `initialQuery`, then advances only the remaining time of the
ORIGINAL window. Reverting the fix → RED (search hasn't fired even at the point it
should have — proving the debounce timer was reset by the re-triggered effect).
Restoring → GREEN, 11/11.

### IN-13: `createSearchController` has no disposal

**Files modified:** `web/src/lib/search.ts`,
`web/src/lib/components/browse/SearchPanel.svelte`, `web/tests/search.test.ts`,
`web/tests/search-panel.test.ts`, `web/build/`
**Commit:** `c9c4c462`
**Applied fix:** Added `dispose()` to the controller (clears pending debounce timer,
aborts both live and explore `AbortController`s). `SearchPanel` calls it from a
cleanup-only `$effect` on unmount.
**Discrimination proof (two layers):** (1) Removing `dispose()` from `search.ts` →
3 tests RED (`controller.dispose is not a function`). (2) Removing the `$effect`
wiring from `SearchPanel` (with `dispose()` still present) → component-level test
RED (RPC fired after unmount). Restoring both → 25/25 GREEN combined.

### IN-14: errcheck discards have no inline reasoning — `docs`, not a defect fix

**Files modified:** `internal/mcp/tools.go`, `internal/agents/opencode.go`
**Commit:** `e57268d9`
**Applied fix:** Added a one-line comment at each `defer func() { _ = close() }()`
call site pointing at `openEngine`'s own doc comment, which now explains once,
authoritatively, why that closer's error is non-actionable. Added a similar
one-line comment at the opencode.go stale-file sweep site. No behavior change.

### IN-15: `.golangci.yml`'s enabled-linters header has no durable guard

**Files modified:** `internal/upgrade/golangci_shape_test.go` (new)
**Commit:** `dbad7ae1`
**Applied fix:** `TestGolangciEnabledLintersHeaderMatchesConfig` parses both the `#
enabled-linters: N` header comment and the real `linters.enable` +
`formatters.enable` lists with the real YAML decoder, and asserts equality.
**Discrimination proof:** Planted-fixture test covering both directions (under/over
count). Also proved against the REAL file: temporarily added a sixth linter to
`.golangci.yml` without touching the header → RED, naming the exact drift ("header
comment claims 5 ... = 6"). Reverted the scratch edit (clean diff) → GREEN.

## Skipped Issues

None — all 23 in-scope findings were either fixed (22) or recorded as
`no_change_needed` with disproving evidence (1: IN-09).

---

_Fixed: 2026-08-29_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
