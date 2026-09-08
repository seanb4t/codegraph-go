---
phase: 06-live-push
reviewed: 2026-09-07T21:17:00Z
depth: deep
files_reviewed: 9
files_reviewed_list:
  - web/src/routes/health/+page.svelte
  - web/tests/live-route-refetch.test.ts
  - internal/uiserver/server.go
  - internal/uiserver/server_test.go
  - web/src/lib/components/graph/GraphCanvas.svelte
  - internal/uiproto/uiv1/ui.proto
  - internal/uiserver/livehandler.go
  - internal/uiserver/livepublish.go
  - web/src/lib/status.ts
findings:
  critical: 0
  warning: 1
  info: 1
  total: 2
status: issues_found
---

# Phase 6: Fix-Pass Re-Review

**Reviewed:** 2026-09-07T21:17:00Z
**Depth:** deep (re-review of fix commits `5fc33c57`..`14c0455d` against `06-REVIEW.md`'s six findings)
**Files Reviewed:** 9 (the fix-pass diff surface; `web/build/*` bundle artifacts excluded as generated output)
**Status:** issues_found (both new findings are minor; no fix in this pass is vacuous or wrong)

## Summary

Six findings from `06-REVIEW.md` (CR-01, WR-01, WR-03, WR-04, IN-01, IN-02) were re-audited
against **this project's specific failure mode**: a fix that satisfies its own literal check
while the symptom persists, or a vacuous guard created inside a fix for a vacuity finding. I did
not trust the fixer's self-report or the orchestrator's pre-verification; for the two
test-bearing fixes (CR-01, WR-01) I **reverted the fix hunk in place, re-ran the exact named
test, confirmed it goes genuinely RED with a message matching the described symptom, then
restored the file and confirmed a clean `git diff`.** For the two comment-only fixes (WR-04's
doc-comment removal, IN-02) I diffed line-by-line and counted removed vs. added occurrences of
the leak pattern to confirm zero functional change. For the no-test fix (WR-03) I read the
`renderer.add`/`renderer.removeByIds` implementations directly to independently judge whether
the recorded no-test rationale actually holds, rather than accepting the rationale on its own
say-so — and found it holds for the fix's own scope, but exposed a knock-on comment-staleness
defect the fix itself introduced in a sibling effect it didn't touch.

**Both fixes I could mechanically discriminate (CR-01, WR-01) are genuine, non-vacuous, and
correctly ordered:**

- **CR-01** (`web/src/routes/health/+page.svelte`): reverted all four `id !== requestId ||`
  guards back to bare `controller.signal.aborted` checks (the pre-fix shape) and re-ran
  `pnpm vitest run tests/live-route-refetch.test.ts -t "CR-01 regression"`. The test failed with
  `Expected: bbbb...  Received: aaaa...` — exactly the stale-mount-overwrites-live-response
  symptom the finding describes, not a generic assertion failure. Restored; `git diff` on the
  file was empty afterward. **1 assertion examined in the new test's core check
  (`toHaveTextContent('b'.repeat(40))`, invoked twice — once after the live response settles,
  once after the stale mount response settles): both are positive-value assertions with a
  positive control (proven to fail pre-fix), not zero-count or upper-bound assertions.**
- **WR-01** (`internal/uiserver/server.go`): removed the single `s.stopPublisher()` line from
  the `errCh` branch and re-ran `go test ./internal/uiserver/... -run
  TestServeStopsPublisherOnAbnormalExit -v`. It failed with `Subscribe's channel yielded a value
  instead of being closed — the publisher's registry was not stopped` — the exact failure
  message the test's own author wrote for this exact revert. Restored; `git diff` on the file
  was empty afterward. **1 assertion examined (a three-way `select` distinguishing
  closed-channel/open-channel-with-value/timeout): a genuine behavioral assertion, not a
  zero-count or "nothing bad happened" check** — it requires the registry to be observably
  stopped, not merely absence of a crash.
- The **ctx.Done() branch's publisher-before-Shutdown ordering is untouched** by this fix (confirmed
  by reading `server.go:240-260` directly) — `stopPublisher()` still runs before
  `Shutdown(shutdownCtx)`, so the 5s Ctrl-C exit-code risk this ordering exists to prevent is not
  reintroduced.
- **`stopPublisher` is `sync.Once`-guarded** (`server.go:294-301`), so the errCh branch's new call
  cannot double-stop the publisher if a caller also calls `Close()` afterward.

**WR-04's regeneration is doc-comment-only, confirmed three ways:** `git show a6bc08c1`'s diff on
`ui.pb.go`/`ui.proto`/`uiv1connect/ui.connect.go` touches only comment text (no field numbers,
no message/rpc signatures); `task proto:drift` re-ran clean against the current tree
(byte-identical to the pinned toolchain's regeneration); `task web:drift` re-ran clean (110
source files hashed, 32 output files, both digests matched the committed bundle). The claimed
"30 remaining" leaks were independently recounted against the exact 32-file review scope using
`rg -o | wc -l` (never `-c`, per this repo's own durable-gate discipline): **30, confirmed exactly.**

**IN-01**'s removed guard (`if (id !== requestId) return;` in `status.ts`'s `applyLiveEvent`) was
confirmed dead by re-reading the surrounding code: no `await` exists between `const id =
++requestId` (now `requestId += 1`) and the guard's removal point, so no code could have run
between mint and check that would supersede `id`. The kept `requestId += 1` line is not dead: it
is read by `fetchStatus`'s own two `id !== requestId` guards (lines 233, 242), which is the real
supersession mechanism for an in-flight `fetchStatus` call. Confirmed this reasoning by reading
both call sites together, not by trusting the commit message's restatement of it.

**IN-02** is a comment-only addition (`internal/uiserver/livehandler.go:43-50`); `git show
d8c7b4b4` touches no non-comment line.

**One new defect was found, introduced by WR-03's own fix, that none of CR-01/WR-01/WR-04/IN-01/IN-02's
verification would have caught because it lives one effect over from what WR-03 touched** — see
WR-2-01 below. It does not reopen CR-01/WR-01/WR-03/WR-04/IN-01/IN-02 and does not block ship, but
it is a comment now actively misleading a future reader, introduced squarely by the commit under
review (`d2780a2f`), so it belongs in this fix-pass's own record rather than deferred to a future
sweep that has no reason to look at this line.

## Warnings

### WR-2-01: WR-03's fix left the FIFTH effect's comment claiming parity with the FOURTH effect's now-different reasoning

**File:** `web/src/lib/components/graph/GraphCanvas.svelte:949-952`
**Issue:** Before `d2780a2f`, the fourth effect (`addedElements`) and fifth effect
(`removedElementIds`) shared one "no guard needed" rationale, and the fifth effect's comment
said so explicitly: `// Same reasoning as the fourth effect above.` (confirmed via `git show
033ee9cb:web/src/lib/components/graph/GraphCanvas.svelte`, lines 918-928 pre-fix). `d2780a2f`
rewrote the fourth effect's own comment and code to the **opposite** conclusion — "no guard
needed" became "identity guard IS needed, because this shape was found at guava scale to
double-fire" (`GraphCanvas.svelte:932-941`) — but left the fifth effect's comment
(`GraphCanvas.svelte:949-951`) untouched, still reading `// Same reasoning as the fourth effect
above.` That sentence is now false: it no longer describes what is above it. A future reader
who reads effect five's comment in isolation is told effect five shares effect four's reasoning,
and will reasonably (and now incorrectly) conclude effect five was audited and found to need — or
not need — the identical guard, when in fact effect five was never examined during this
finding's resolution at all.

I independently checked whether effect five is actually exposed to the same hazard by reading
`removeByIds`'s implementation (`GraphCanvas.svelte:489-506`): each call does
`opts.cy.getElementById(id)` and guards on `el.length > 0` before calling `.remove()`, and
rebuilds `liveAddedElements` via a `.filter()` that is naturally idempotent against re-removing
already-removed ids. Unlike `add()` (effect four's sink, which the review correctly flagged as
throwing on a duplicate id or double-appending to `liveAddedElements`), a **second, spurious
invocation of `removeByIds` with the identical id list is a true no-op** — nothing throws,
nothing double-appends, only a harmless redundant `computeGeometry`/`onGeometry` republish
occurs. So effect five does NOT need the identity guard on correctness grounds — but that
conclusion is mine, arrived at by reading `removeByIds`'s body, not anything the current comment
states or that WR-03's fix pass recorded anywhere.

**Fix:** Replace the stale cross-reference with the actual, effect-five-specific reasoning —
e.g.: `// Unlike the fourth effect above, no identity guard is needed here: removeByIds() is` `//
naturally idempotent against a repeated invocation with the same id list (getElementById +` `//
an el.length guard before remove(), and a filter() that no-ops on an already-absent id) — a` `//
double-fire republishes geometry redundantly but corrupts nothing.` This is a one-line
diff-surface fix; no test is warranted for it (it changes no behavior), but it should not wait
for the next full review pass to be reconciled, since it is a `06-REVIEW.md`-cited fix commit
that produced it.

## Info

### IN-2-01: WR-04's own commit message and `06-REVIEW.md` resolution note undercount the fix's own diff by 3

**File:** `.planning/phases/06-live-push/06-REVIEW.md:339-347` (resolution note); commit message
of `a6bc08c1`
**Issue:** Both the commit message and the `06-REVIEW.md` resolution note state "20 occurrences
removed" and "53 -> 30 total remaining." Arithmetically, 53 - 20 = 33, not 30 — the note's own
two numbers don't agree with each other. I recounted the actual diff directly: `git show
a6bc08c1` touching only the in-scope, non-generated files (`ui.proto`, `livehandler.go`,
`livepublish.go`, `status.ts`, the four route/component files) contains **23** removed-line
occurrences of the `06-0[0-9]` pattern and **0** added occurrences — not 20. I also
independently recounted the current remaining total across the review's exact 32-file scope
using `rg -o "06-0[0-9]" <file> | wc -l` per-file (never `-c`, this project's own durable-gate
discipline) and got exactly **30**, matching the note's claimed remaining count. So the
before/after totals (53, 30) are both independently verified correct; the "20 removed" figure in
between them is the one wrong number, undercounting the actual work by 3. This is the *opposite*
direction of the failure mode this review is hunting for — the fix did **more** than its own
paperwork claims, not less — so it carries no functional or regression risk. It is flagged
because this project's own standard for this review ("counts as evidence your checks ran") makes
an internally-inconsistent count in a committed resolution note a real (if low-severity) defect
in the record, and it is cheap to fix.
**Fix:** Correct "20 occurrences removed" to "23 occurrences removed" in `06-REVIEW.md`'s WR-04
resolution note (`06-REVIEW.md:339`); the commit message itself is immutable history and does
not need (and per this project's git discipline should not get) a rewrite.

## Verification Performed (raw evidence)

- `go build ./...` — clean.
- `go vet ./internal/uiserver/...` — clean. `gofmt -l` on the package — no output (already
  formatted).
- `go test ./internal/uiserver/... -run 'TestServeStopsPublisherOnAbnormalExit|TestServeReturnsNilOnContextCancellation' -v` — both PASS.
- `pnpm vitest run` (full suite) — **464/464 passed**, 42/42 files.
- `pnpm check` — 0 errors, 0 warnings, 1159 files.
- `task proto:drift` — PASS, 4 generated files byte-identical to pinned-toolchain regeneration.
- `task web:drift` — PASS, 110 source files / 32 output files, both digests matched the committed
  `web/build/`.
- `task test:unit` — all Go packages `ok`.
- `git status --porcelain` before and after every scratch revert — empty both times (no stray
  edits left in the tree).
- Revert-and-rerun on `internal/uiserver/server.go` (WR-01) and
  `web/src/routes/health/+page.svelte` (CR-01) — both produced genuine, symptom-matching RED;
  both restored cleanly.

## Verdict on the Six Original Findings

| Finding | Verdict |
|---|---|
| CR-01 | **Genuine fix, discriminating test proven by revert.** No issue. |
| WR-01 | **Genuine fix, discriminating test proven by revert. Ordering with ctx.Done() branch unchanged and verified correct.** No issue. |
| WR-03 | **Genuine fix on its own stated scope; no-test rationale independently checked and found sound** (the underlying hazard is real for `add()`, confirmed by re-reading the sink). **But the fix left a stale, now-misleading comment on the untouched sibling (fifth) effect — see WR-2-01.** |
| WR-04 | **Genuine, doc-comment-only change; zero field/schema drift, confirmed three independent ways.** Own count is internally inconsistent by 3 — see IN-2-01 (informational only). |
| IN-01 | **Genuine dead-code removal; the surviving `requestId` bump is real and is consumed by `fetchStatus`'s own guards** — confirmed by reading both together. No issue. |
| IN-02 | **Comment-only, confirmed via diff.** No issue. |

No instance of "the fix satisfies its own literal check while the symptom persists" or "a
vacuous guard created inside a fix for a vacuity finding" was found in this fix pass. The one
new defect (WR-2-01) is a documentation-staleness knock-on effect, not either of those shapes,
and is scoped narrowly enough that it should not block the phase.

---

_Reviewed: 2026-09-07T21:17:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
