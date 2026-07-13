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
// committed baseline and fails the gate (PERF-02) when either:
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
