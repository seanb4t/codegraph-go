---
phase: 01-defect-flake-burn-down
reviewed: 2026-09-15T19:15:22Z
depth: deep
files_reviewed: 22
files_reviewed_list:
  - .github/workflows/pr-template-format.yml
  - .github/workflows/require-issue-link.yml
  - Taskfile.yml
  - corpora/graph-console-check.json
  - internal/bench/regression.go
  - internal/bench/regression_test.go
  - internal/cli/index.go
  - internal/cli/index_lock_test.go
  - internal/daemon/daemon.go
  - internal/daemon/daemon_test.go
  - internal/daemon/watchdog.go
  - internal/daemon/watchdog_posix.go
  - internal/daemon/watchdog_shape_test.go
  - internal/daemon/watchdog_test.go
  - scripts/check-workflow-output-delimiter.sh
  - tools/bench/BASELINE.md
  - web/scripts/graph-console-check.mjs
  - web/src/app.d.ts
  - web/src/lib/components/graph/GraphCanvas.svelte
  - web/src/lib/components/graph/graph-style.ts
  - web/src/routes/+layout.svelte
  - web/static/codegraph-mark.svg
  - web/static/favicon.svg
findings:
  critical: 1
  warning: 0
  info: 1
  total: 2
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-09-15T19:15:22Z
**Depth:** deep
**Files Reviewed:** 22
**Status:** issues_found

## Summary

This phase's Go-side changes (the daemon watchdog's per-instance `getppid`/tick
seam, the `internal/cli/index.go` store-lock refusal, `internal/bench/regression.go`'s
`Repo` frame guard, and the two `pull_request_target` workflows' per-run heredoc
delimiter) are all narrowly scoped, well-tested, and fix the defects they target
at the cause — no widened timeouts, no added sleeps in production code, no
allowlisted warnings. The new `check:workflow-output-delimiter` and
`check:graph-console` Taskfile targets extract and exercise the actual shipped
workflow/script bodies rather than hand-copied duplicates, which is the right
shape for a regression gate. `tools/bench/BASELINE.md`'s investigation record is
unusually disciplined about not asserting an unmeasured cause.

The one substantive finding is in `GraphCanvas.svelte`'s overlapping-layout
guard, which is exactly the area the orchestrator asked this pass to scrutinize
closely. Tracing the guard against the installed `cytoscape@3.34.2` emitter
source shows the generation-token check does not actually close the race it
documents itself as closing: a *single* `layoutstop` event that bubbles from
one Cytoscape layout instance up to `cy` invokes *every* currently-registered
`cy.one('layoutstop', ...)` listener in one synchronous pass, not just the
listener belonging to the layout instance that emitted it. That means an
*older*, superseded ELK layout's real completion can satisfy the *newest*
generation's callback before the newest generation's own ELK computation has
actually finished — clearing `layoutInFlight` early and letting `cy.destroy()`
run while a real layout promise is still outstanding, which is the exact crash
class (`endBatch()` dispatching into a destroyed renderer) FIX-04 was written
to prevent. The existing test suite (`web/tests/graph-live-update.test.ts`)
exercises the "older callback fires, then the newer one is invoked separately"
ordering but never the "one bubble satisfies both" case that real Cytoscape's
emitter actually produces, so this gap has no regression coverage.

## Critical Issues

### CR-01: The overlapping-layout generation guard does not scope a `layoutstop` bubble to the layout instance that emitted it, so a stale, superseded layout's completion can be misread as the current generation's own settle

**File:** `web/src/lib/components/graph/GraphCanvas.svelte:373-433` (the `opts.cy.one('layoutstop', ...)` registration inside `runLayout`, plus `layoutInFlight`'s clear at line 381 and the mount-effect teardown gate at lines 1149-1176)

**Issue:**

`runLayout` registers its completion handler on `cy` itself:

```js
opts.cy.one('layoutstop', () => {
    if (myGeneration !== layoutGeneration) return;
    layoutInFlight = false;
    ...
});
...
activeLayoutRun = opts.cy.layout({ ...LAYOUT_OPTIONS, fit }).run();
```

The component's own comment (lines 274-291) correctly identifies that
Cytoscape's layout events **bubble from the layout instance up to `cy`**
(confirmed independently against the installed `cytoscape@3.34.2` source —
`node_modules/.pnpm/cytoscape@3.34.2/.../dist/cytoscape.esm.mjs:35450-35460`
sets `bubble: function() { return true; }` and `parent: function(layout) {
return getCy(layout); }` on every layout's emitter options), and concludes
that the generation-token check (`myGeneration !== layoutGeneration`) is what
prevents a superseded run's callback from acting on the wrong state.

That reasoning covers only half the hazard. Reading the emitter's own `emit`
implementation (`cytoscape.esm.mjs:12502-12547`) shows that when a layout's
`emit('layoutstop')` bubbles to `cy`, it becomes a **single** call to `cy`'s
own `emit()`, which iterates `cy`'s *entire* current listener array and
invokes **every** matching listener in that one synchronous pass — regardless
of which underlying layout instance actually triggered the bubble, and with
no argument-based way for a listener to tell which instance fired it (the
registered callbacks here take no event parameter at all).

Concretely: if `runLayout` is called twice in quick succession (a live update
arriving while a user's expand/collapse layout is still running — exactly the
scenario the code's own comment describes, and a scenario `cytoscape-elk`'s
own `stop()`/`destroy()` are documented no-ops for, so the older run's async
ELK promise keeps running to completion regardless), **two** `cy.one`
listeners are pending: the OLDER call's (captured `myGeneration = N`) and the
NEWER call's (captured `myGeneration = N+1`, matching the live
`layoutGeneration`). Whichever of the two ELK computations finishes *first*
fires exactly one `layoutstop` bubble to `cy`. That single bubble invokes
**both** pending listeners in the same pass:

- The stale (`N`) listener's token check fails and it no-ops, as intended.
- The current (`N+1`) listener's token check **trivially passes** — it has
  no way to know this event actually originated from the *older*, still-
  finishing-late layout rather than its own — so it runs the full settle
  path (`layoutInFlight = false`, edge reveal, survivor-position write-back,
  resize, metrics/geometry publish) using whatever positions the *older*
  layout happened to leave behind, even though the *newer* generation's own
  ELK computation may still be genuinely in flight.

Because `.one()` self-unregisters after firing, the newer generation's
listener has now been consumed. When that newer layout's own ELK promise
later resolves for real and calls `emit('layoutstop')`, there is no listener
left on `cy` to catch it — the "real" settle for generation `N+1` never
separately runs.

The direct consequence for FIX-04 (the mechanism this same file documents at
length): the mount-effect's teardown (lines 1149-1176) decides whether to
defer `cy.destroy()` by asking `rendererAtTeardown.isLayoutInFlight()`. If
teardown is requested in the window between the premature (`N`-triggered)
clearing of `layoutInFlight` and the genuine completion of generation `N+1`'s
own ELK work, `isLayoutInFlight()` incorrectly reports `false`, teardown skips
the deferred-destroy path, and `cy.destroy()` runs while an ELK promise for
generation `N+1` is still outstanding — reproducing the exact
`endBatch()`-dispatches-to-a-destroyed-renderer crash FIX-04 exists to
prevent (per this file's own comment at lines 1082-1098, `endBatch()` has no
destroyed-instance guard). It also means the FIX-05 edge-reveal can fire
against a layout whose real position computation has not actually finished,
and that `survivorPositions` write-back (D-06's "exact half" of the
stability guarantee) can apply the wrong generation's captured positions.

The generation-token check is real, necessary defense against the *other*
half of this hazard (an old listener acting on stale data when it fires on
its own), but it cannot detect "this event came from an instance that isn't
mine" — that requires per-instance scoping, which is missing.

**Test coverage gap:** `web/tests/graph-live-update.test.ts`'s "layout
generation guard (serialization)" suite (`makeGuardFakeCy`, lines 401-639)
models each pending `cy.one('layoutstop', ...)` registration as an
independently-invocable function (`cy.__handlerAt(i)()`) that the test calls
in whatever order it chooses. That is a reasonable model for "the stale
listener eventually fires on its own" but cannot reproduce "one real emit()
call synchronously invokes two listeners in the same pass," because the fake
never invokes more than one handler per simulated event. No test in this
file (or `graph-expand.test.ts` / `graph-expansion.test.ts`) exercises the
one-bubble-satisfies-both-listeners shape, so this gap ships with zero
regression coverage.

**Fix:**

Scope the completion handler to the specific layout instance that
`runLayout` just started, instead of relying solely on the generation
counter. Cytoscape's own layout objects are separate emitters (each created
via `layoutProto.createEmitter()`), so registering directly on the layout
instance — rather than on `cy` — means only that instance's own `emit()`
call (before it bubbles anywhere) can invoke it:

```js
layoutInFlight = true;
const thisRun = opts.cy.layout({ ...LAYOUT_OPTIONS, fit });
activeLayoutRun = thisRun.run();
thisRun.one('layoutstop', () => {
    if (myGeneration !== layoutGeneration) return; // still worth keeping as defense in depth
    layoutInFlight = false;
    ...
});
```

(Move the `opts.cy.one(...)` registration above to be `thisRun.one(...)`,
registered against the same layout object `.run()` is called on, rather than
against `opts.cy`.) This makes "only my own layout's completion can invoke
my callback" a structural guarantee instead of an assumption the generation
token cannot actually enforce, and it should be paired with a new test that
drives the fake core (or a real headless instance with two real ELK layouts)
through the "older instance's real completion fires while the newer
instance's own work is still pending" shape directly, rather than only
testing independently-ordered handler invocations.

## Info

### IN-01: `tools/bench/BASELINE.md`'s in-flight discriminator link points to a specific CI run that will eventually be pruned by GitHub's retention policy

**File:** `tools/bench/BASELINE.md:439,477`
**Issue:** The "GH #20 follow-up decisions" section cites `run 34980422924`
as the source of the `16569.160272289788` files/s figure, with `[gh20-run]`
linking directly to that run's Actions page. GitHub Actions run artifacts and
logs are not retained indefinitely (typically 90 days by default), so the
underlying evidence behind this specific number may become unreachable while
the markdown file — which is explicitly written as a permanent investigation
record ("kept complete on purpose") — persists indefinitely.
**Fix:** No action required for this review (this is a documentation
durability note, not a functional defect); if the artifact/log is still
needed as evidence after the retention window, consider mirroring the
`baseline-candidate` artifact's raw numbers into the repo (e.g. alongside
`tools/bench/baseline.json`'s own history) rather than relying solely on the
external run link.

---

_Reviewed: 2026-09-15T19:15:22Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
