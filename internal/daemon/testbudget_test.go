package daemon

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

// This file holds the two shared test-only helpers that close MAINT-01
// (issue #13, the -race failure on the getppid seam) and MAINT-02 (issue
// #17, the daemon package's tests failing under full-suite load) at their
// shared root cause: t.Fatalf's runtime.Goexit() unwinds a test's stack and
// runs its t.Cleanup/defer callbacks WITHOUT ever waiting for a bare
// `go func(){ d.Run(ctx) }()` spawn to return, whenever the test's own
// fixed wall-clock deadline fires before Run's internal shutdown join
// completes. Under full-suite parallel load those deadlines fire far more
// often than on an idle machine, and an orphaned goroutine that outlives
// its test can then read a package-level test seam (getppid, registryDir)
// that a LATER, unrelated test is concurrently writing back via its own
// restore — a data race whose victim depends purely on scheduling.
//
// testBudget scales every wall-clock deadline in this package's tests
// together, from one knob, so a contended machine or a resource-
// constrained CI runner can buy headroom without touching N independent
// literals. joinDaemonRun is the actual fix: registered as a t.Cleanup
// (which — unlike a plain defer positioned after the failing statement —
// always runs on the Goexit unwind path too) immediately after every spawn
// site, it cancels the context and blocks until the spawned goroutine's
// result actually arrives, so no goroutine outlives its test regardless of
// which statement inside the test triggered the failure.

const (
	// testBudgetDefaultScale is applied when testBudgetScaleEnvVar is
	// unset or unparsable. It is NOT 1x: 04-02-SUMMARY.md's Task 1/Task 3
	// measurements (this package's own instrument) found that a 16-core
	// dev machine running the full unfiltered `go test ./...` (~51
	// packages) can delay a single debounced fsnotify->Sync round-trip
	// well past the base 5s/10s literals under contention — raising the
	// DEFAULT, not only the env-var ceiling, is what gives every
	// contended machine real headroom without editing source. A large
	// default costs nothing on the common/fast path (a healthy Run/Sync
	// round-trip completes in milliseconds regardless of how generous the
	// ceiling is) — the only place this duration is ever spent is a test
	// that is already slow or stuck, which is exactly the case that needs
	// the headroom.
	//
	// 04-02-SUMMARY.md records the honest limit of this knob: on THIS
	// specific measurement session's machine, `ps aux` during Task 3
	// showed several concurrent, unrelated Claude Code agent sessions
	// plus a live desktop GUI competing for the same cores — a load this
	// package's tests do not control and CI's isolated runner does not
	// have. Even at this default (and the max clamp below), that specific
	// external contention could still occasionally blow the budget on
	// this machine at this moment; no finite, bounded scale can guarantee
	// otherwise against unbounded external load. What this default DOES
	// reliably close, measured over repeated runs: (1) the data race is
	// gone in every case, contended or not — join-before-restore removes
	// the concurrent access structurally, it does not depend on winning a
	// timing race against a bigger number; (2) isolated package runs and
	// GOMAXPROCS=4 runs (the CI runner class this phase's scope is
	// actually calibrated against, per 04-CONTEXT.md's RESOLVED
	// assumption) are clean.
	testBudgetDefaultScale = 25.0

	// testBudgetMaxScale bounds how far testBudgetScaleEnvVar can stretch
	// every deadline in this package's tests, leaving headroom above the
	// default for a machine even more contended than the one 04-02's own
	// measurements were taken on. This clamp exists so raising the scale
	// can never quietly convert a genuine hang (Run/RunWithRetry never
	// returning at all — the SYNC-06/INDX-05 join-before-lock-release
	// invariant broken) into a very slow PASS: 40x the package's largest
	// base deadline (joinTimeoutBase, 10s) is 400s, well inside a single
	// CI step's timeout, so a real hang still fails loudly rather than
	// being silently absorbed by an unbounded knob. Scaling a deadline is
	// NOT the same as isolating a test away from the load that exposes
	// it — the assertions and the load are unchanged, only the wall-clock
	// budget the assertion is given adapts to the environment it actually
	// runs in. The proof that this did not become an escape hatch is
	// 04-02-SUMMARY.md's Task 3 measured rate, taken under the SAME
	// unfiltered full-suite load as the baseline, not a relaxed one.
	testBudgetMaxScale = 40.0

	// testBudgetScaleEnvVar lets a contended dev machine or a resource-
	// constrained CI runner (e.g. GOMAXPROCS=4) buy headroom without
	// editing source. Unset in normal CI/dev use today — this is
	// defensive/contributor-facing (04-CONTEXT.md's RESOLVED assumption:
	// ci.yml's existing test:daemon/test:race isolation shows no occurrence
	// of these failures on the real runner across the last 52 runs), not a
	// knob CI is expected to need to turn.
	testBudgetScaleEnvVar = "CODEGRAPH_DAEMON_TEST_BUDGET_SCALE"

	// joinTimeoutBase is joinDaemonRun's own bounded-wait input to
	// testBudget: long enough to cover the package's largest legitimate
	// teardown path (watchdogInterval's 1s ticker plus a several-episode
	// flush-lock requeue chain) with real headroom, scaled like every
	// other deadline in this package's tests rather than living as its own
	// independent literal.
	joinTimeoutBase = 10 * time.Second
)

var (
	testBudgetScaleOnce sync.Once
	testBudgetScaleVal  float64
)

// testBudgetScale resolves the shared scale factor exactly once per test
// binary run (sync.Once — not per-call), so every deadline in the package
// scales together from a single resolved value instead of each call site
// re-reading (and potentially re-parsing inconsistently) the environment.
func testBudgetScale() float64 {
	testBudgetScaleOnce.Do(func() {
		testBudgetScaleVal = testBudgetDefaultScale
		raw := os.Getenv(testBudgetScaleEnvVar)
		if raw == "" {
			return
		}
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil || v <= 0 {
			return
		}
		if v > testBudgetMaxScale {
			v = testBudgetMaxScale
		}
		testBudgetScaleVal = v
	})
	return testBudgetScaleVal
}

// testBudget (MAINT-01, MAINT-02) scales base by the package's single
// shared budget factor. Every fixed wall-clock deadline in this package's
// test files — every timer-based select arm, every context.WithTimeout,
// every polling helper's own bound — is routed through this call so the
// package has exactly one knob, not N independently-editable literals. The
// base duration itself is never changed by this — only the scaling is new.
func testBudget(base time.Duration) time.Duration {
	return time.Duration(float64(base) * testBudgetScale())
}

// joinDaemonRun (MAINT-01, MAINT-02) registers a t.Cleanup that cancels ctx
// and then blocks — bounded by testBudget(joinTimeoutBase) — until the
// spawned Daemon.Run/RunWithRetry goroutine's result actually arrives on
// runErr. This is the actual fix: a t.Cleanup (unlike the defer-based
// restores this same task converts to t.Cleanup) always runs on the
// runtime.Goexit() unwind path a failing t.Fatalf takes, so the spawned
// goroutine is guaranteed to have returned — and stopped reading any
// package-level test seam — before this test's own cleanups finish,
// regardless of which statement inside the test body triggered the
// failure. If the goroutine's result does not arrive within the budget,
// that is itself reported as a test failure (via t.Errorf, not silently
// swallowed) rather than the join hanging the whole run or masking a
// genuine deadlock: silence here would recreate exactly the bug this task
// closes.
//
// runErr must be a channel this test's own spawn goroutine sends its
// single result to AND closes immediately after sending (see call sites):
// closing after the send makes a second, later receive — this cleanup,
// running after the test body may have already consumed the one buffered
// value on its own happy path — return immediately with the zero value
// instead of blocking for the full budget on every passing test.
//
// Ordering contract (t.Cleanup runs LIFO — last registered, first run):
// joinDaemonRun MUST be called AFTER any t.Cleanup that restores a
// package-level test seam (getppid, registryDir) that this test mutated,
// so this join's cleanup sits later on the stack and therefore runs
// FIRST — the spawned goroutine has genuinely stopped reading the seam
// before the seam is restored to its prior value.
func joinDaemonRun(t *testing.T, cancel context.CancelFunc, runErr <-chan error) {
	t.Helper()
	t.Cleanup(func() {
		cancel()
		budget := testBudget(joinTimeoutBase)
		select {
		case <-runErr:
		case <-time.After(budget):
			t.Errorf("joinDaemonRun: spawned goroutine did not return within %s of ctx cancellation — goroutine leak (MAINT-01/MAINT-02): it may still be reading a package-level test seam this or a later test restores", budget)
		}
	})
}
