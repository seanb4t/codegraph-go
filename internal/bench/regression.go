package bench

import "fmt"

// DefaultThroughputTolerance is the maximum allowed relative regression in
// indexing throughput (files/sec) before the gate fails: 10% slower than
// the committed baseline. This is a D-05 starting point, deliberately
// tune-able by callers that need a different budget.
const DefaultThroughputTolerance = 0.10

// DefaultRSSTolerance is the maximum allowed relative growth in peak RSS
// before the gate fails: 15% larger than the committed baseline. This is a
// D-05 starting point, deliberately tune-able by callers that need a
// different budget.
const DefaultRSSTolerance = 0.15

// CheckRegression compares a candidate run's Metrics (current) against a
// committed baseline and fails the gate (PERF-02) when any of:
//   - baseline and current were measured on different platforms
//     (GOOS/GOARCH), which makes every numeric comparison below
//     meaningless — see the platform-mismatch check for why, or
//   - indexing throughput regresses beyond DefaultThroughputTolerance, or
//   - peak RSS grows beyond DefaultRSSTolerance relative to the baseline, or
//   - peak RSS exceeds ceilingBytes, an absolute bounded-memory budget
//     (INDX-06) enforced independently of the relative RSS delta — a
//     baseline that itself already used a lot of memory must not let
//     further growth hide behind a large denominator.
//
// CheckRegression NEVER mutates baseline or current, and it never panics:
// a degenerate baseline (zero or negative FilesPerSec) returns a plain
// error instead of dividing by zero. Re-blessing the baseline (updating
// baseline.json to accept a new normal) is a separate, explicit action —
// an operator-invoked `-rebless` flag on the runner (Plan 08-07) — and is
// never a side effect of running this check. That separation is D-05's
// point: an accidental auto-rewrite here would silently defeat the gate.
func CheckRegression(baseline, current Metrics, ceilingBytes int64) error {
	// Platform validity precedes numeric validity: comparing across
	// GOOS/GOARCH is not a tolerance question, it is a category error.
	// tools/bench/BASELINE.md's recorded investigation shows a stable,
	// reproducible, entirely fictitious ~10.6% "regression" on the
	// synthetic PERF-02/INDX-06 corpus that survived three rounds of
	// triage before a platform mismatch was found to be the actual
	// cause — a cross-platform delta on IDENTICAL code carries no
	// information about the code at all. Refusing loudly is strictly
	// better than gating on a number that cannot mean anything — the
	// alternative (what this function did before) silently reported
	// that same kind of fictitious regression. An unset GOOS/GOARCH on
	// BOTH sides still matches and is allowed, so callers that construct
	// Metrics without platform attribution (unit tests, pre-attribution
	// baselines) are unaffected.
	if baseline.GOOS != current.GOOS || baseline.GOARCH != current.GOARCH {
		return fmt.Errorf(
			"bench: platform mismatch: baseline was measured on %s but this run is %s; "+
				"indexing throughput from this harness is not comparable across platforms, "+
				"so this comparison would be meaningless. Record a baseline on this platform "+
				"(runner -mode regression -rebless) instead of comparing across them",
			platformString(baseline), platformString(current),
		)
	}

	// Runner identity is a category error, not a tolerance question, for
	// the same reason GOOS/GOARCH is: namespace-profile-linux-amd64-4x8
	// IS linux/amd64, so the platform guard above is structurally blind to
	// a runner-class change that holds GOOS/GOARCH constant while changing
	// everything about the actual measurement environment (a stable,
	// reproducible, entirely fictitious ~10.6% "regression" survived three
	// rounds of triage before this was understood — see
	// tools/bench/BASELINE.md). An empty Runner means "never recorded",
	// not a wildcard: it is refused against a non-empty value on either
	// side, exactly like the platform guard's unattributed-vs-attributed
	// case. Both empty still matches, so callers that construct Metrics
	// without runner attribution (unit tests, pre-attribution baselines)
	// are unaffected.
	if baseline.Runner != current.Runner {
		return fmt.Errorf(
			"bench: runner mismatch: baseline was measured on runner %s but this run is %s; "+
				"a runner-class change can hold GOOS/GOARCH constant while changing the actual "+
				"measurement environment, so this comparison would be meaningless. An empty "+
				"runner value means it predates runner recording, which is not a wildcard match "+
				"against a recorded one — re-bless the baseline on this runner class "+
				"(bench.yml's rebless job) instead of comparing across them",
			runnerString(baseline.Runner), runnerString(current.Runner),
		)
	}

	// ScratchFS gets the identical treatment, for the identical reason:
	// the regression corpus is IOPS/metadata-bound at 120k files, and
	// 10-04-PLAN measured that switching scratch filesystem class alone
	// (tmpfs vs disk) moves session-to-session variance by more than the
	// runner class does. A scratch_fs mismatch is a measurement-frame
	// change of the same kind as GOOS/GOARCH or Runner, not a tolerance
	// question, and an empty value means "never recorded" rather than a
	// wildcard, mirroring Runner's own empty-side handling exactly.
	if baseline.ScratchFS != current.ScratchFS {
		return fmt.Errorf(
			"bench: scratch filesystem mismatch: baseline was measured on scratch_fs %s but "+
				"this run is %s; the scratch filesystem class is as load-bearing to a throughput "+
				"comparison as the runner class or GOOS/GOARCH are (tools/bench/BASELINE.md), so "+
				"this comparison would be meaningless. An empty scratch_fs value means it "+
				"predates scratch_fs recording, which is not a wildcard match against a recorded "+
				"one — re-bless the baseline on this scratch_fs class instead of comparing across "+
				"them",
			scratchFSString(baseline.ScratchFS), scratchFSString(current.ScratchFS),
		)
	}

	if baseline.FilesPerSec <= 0 {
		return fmt.Errorf("bench: invalid baseline: FilesPerSec must be positive, got %.4f", baseline.FilesPerSec)
	}
	if baseline.PeakRSSBytes <= 0 {
		return fmt.Errorf("bench: invalid baseline: PeakRSSBytes must be positive, got %d", baseline.PeakRSSBytes)
	}

	throughputDelta := (baseline.FilesPerSec - current.FilesPerSec) / baseline.FilesPerSec
	if throughputDelta > DefaultThroughputTolerance {
		return fmt.Errorf(
			"bench: throughput regressed %.1f%% (budget: %.1f%%): baseline=%.2f files/s current=%.2f files/s",
			throughputDelta*100, DefaultThroughputTolerance*100, baseline.FilesPerSec, current.FilesPerSec,
		)
	}

	rssDelta := float64(current.PeakRSSBytes-baseline.PeakRSSBytes) / float64(baseline.PeakRSSBytes)
	if rssDelta > DefaultRSSTolerance {
		return fmt.Errorf(
			"bench: peak RSS grew %.1f%% (budget: %.1f%%): baseline=%d bytes current=%d bytes",
			rssDelta*100, DefaultRSSTolerance*100, baseline.PeakRSSBytes, current.PeakRSSBytes,
		)
	}

	if ceilingBytes > 0 && current.PeakRSSBytes > ceilingBytes {
		return fmt.Errorf(
			"bench: peak RSS %d bytes exceeds absolute ceiling %d bytes (INDX-06 bounded-memory budget)",
			current.PeakRSSBytes, ceilingBytes,
		)
	}

	return nil
}

// platformString renders m's GOOS/GOARCH as "goos/goarch" for error
// messages, degrading to "unknown" for an unattributed component rather
// than emitting a confusing bare slash. Never used for comparison — the
// mismatch check compares the raw fields directly, so an empty GOOS is
// never conflated with a populated one.
func platformString(m Metrics) string {
	goos, goarch := m.GOOS, m.GOARCH
	if goos == "" {
		goos = "unknown"
	}
	if goarch == "" {
		goarch = "unknown"
	}
	return goos + "/" + goarch
}

// runnerString renders a Runner value for error messages, degrading an
// empty value to "(not recorded)" rather than an empty pair of quotes.
// Never used for comparison — the mismatch check compares the raw field
// directly, so an empty Runner is never conflated with a populated one.
func runnerString(runner string) string {
	if runner == "" {
		return "(not recorded)"
	}
	return runner
}

// scratchFSString is runnerString's twin for ScratchFS, same rationale.
func scratchFSString(scratchFS string) string {
	if scratchFS == "" {
		return "(not recorded)"
	}
	return scratchFS
}
