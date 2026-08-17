package bench

import (
	"encoding/json"
	"os"
	"testing"
)

// committedBaselinePath is the on-disk path (relative to this package) to
// the committed regression baseline that ships in the repo and that
// CheckRegression compares every CI run against. ROADMAP success
// criterion 2 names this exact file as the mechanism BENCH-02 must prove
// still fires — reading it here, rather than constructing an in-memory
// stand-in, is what makes that proof non-vacuous (cross-AI review HIGH
// 4). Follows the same relative-path idiom internal/upgrade's workflow
// guards already use for reading committed files off disk.
const committedBaselinePath = "../../tools/bench/baseline.json"

// loadCommittedBaseline reads and unmarshals committedBaselinePath into a
// Metrics, failing the test loudly — never skipping — if the file is
// missing or malformed. A skip here would make BENCH-02's whole
// committed-baseline proof silently disappear the day the path moves.
func loadCommittedBaseline(t *testing.T) Metrics {
	t.Helper()
	data, err := os.ReadFile(committedBaselinePath)
	if err != nil {
		t.Fatalf("loadCommittedBaseline: reading %s: %v", committedBaselinePath, err)
	}
	var m Metrics
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("loadCommittedBaseline: unmarshaling %s: %v", committedBaselinePath, err)
	}
	return m
}

// TestCheckRegressionAgainstCommittedBaseline drives CheckRegression with
// the COMMITTED tools/bench/baseline.json as the baseline argument,
// rather than an in-memory Metrics literal that merely shares its shape
// — the mechanism ROADMAP success criterion 2 names explicitly. Before
// this test, no Go test in the repository loaded that file; the only
// existing references wrote their own baseline into a temp directory.
//
// The two failing cases' percentages deliberately mirror
// TestCheckRegression's "throughput 11% slower" and "peak RSS 16%
// larger" table cases, so a mutation that reddens one reddens both
// (Plan 06-03 Task 3's rehearsal). No absolute figure from the committed
// file is asserted as a literal here: the percentages are derived from
// whatever the file holds, so a legitimate future re-bless changes the
// numbers without breaking this test.
func TestCheckRegressionAgainstCommittedBaseline(t *testing.T) {
	baseline := loadCommittedBaseline(t)

	// Load positively first. A silently-empty unmarshal (e.g. the file
	// decoding to a zero-value Metrics) would otherwise "pass" every
	// case below for the wrong reason.
	if baseline.Repo == "" {
		t.Fatalf("loadCommittedBaseline(%s): Repo is empty, want non-empty", committedBaselinePath)
	}
	if baseline.FilesPerSec <= 0 {
		t.Fatalf("loadCommittedBaseline(%s): FilesPerSec = %v, want > 0", committedBaselinePath, baseline.FilesPerSec)
	}
	if baseline.PeakRSSBytes <= 0 {
		t.Fatalf("loadCommittedBaseline(%s): PeakRSSBytes = %v, want > 0", committedBaselinePath, baseline.PeakRSSBytes)
	}
	if baseline.GOOS == "" {
		t.Fatalf("loadCommittedBaseline(%s): GOOS is empty, want non-empty", committedBaselinePath)
	}
	if baseline.GOARCH == "" {
		t.Fatalf("loadCommittedBaseline(%s): GOARCH is empty, want non-empty", committedBaselinePath)
	}

	// ceilingBytes is 0 in every case below: a zero ceiling disables the
	// absolute bounded-memory branch (regression.go's ceilingBytes > 0
	// guard), so the peak-RSS case below can only fail through the
	// RELATIVE tolerance band. INDX-06's absolute ceiling is covered by
	// the pre-existing table in regression_test.go and is not this
	// file's subject.
	const ceilingBytes = int64(0)

	t.Run("committed baseline in frame: no regression passes", func(t *testing.T) {
		// Same frame, identical numbers: proves the four category
		// guards (platform, runner, scratch-fs, degenerate-baseline)
		// are quiet, which is what makes the two failing cases below
		// meaningful rather than accidental.
		current := baseline
		err := CheckRegression(baseline, current, ceilingBytes)
		if err != nil {
			t.Fatalf("CheckRegression(committed baseline) = %v, want nil", err)
		}
	})

	t.Run("committed baseline throughput 11% slower: exceeds band fails", func(t *testing.T) {
		// Copies the frame fields verbatim so the category guards stay
		// quiet; only FilesPerSec moves, 11% below the loaded baseline
		// (past the 10% band).
		current := Metrics{
			Repo:         baseline.Repo,
			Subject:      baseline.Subject,
			GOOS:         baseline.GOOS,
			GOARCH:       baseline.GOARCH,
			Runner:       baseline.Runner,
			ScratchFS:    baseline.ScratchFS,
			FilesPerSec:  baseline.FilesPerSec * 0.89,
			PeakRSSBytes: baseline.PeakRSSBytes,
		}
		err := CheckRegression(baseline, current, ceilingBytes)
		if err == nil {
			t.Fatalf("CheckRegression(committed baseline) = nil, want error")
		}
		if !containsFold(err.Error(), "throughput") {
			t.Errorf("error %q does not mention expected hint %q", err.Error(), "throughput")
		}
	})

	t.Run("committed baseline peak RSS 16% larger: exceeds band fails", func(t *testing.T) {
		// Copies the frame fields verbatim so the category guards stay
		// quiet; only PeakRSSBytes moves, 16% above the loaded baseline
		// (past the 15% band).
		current := Metrics{
			Repo:         baseline.Repo,
			Subject:      baseline.Subject,
			GOOS:         baseline.GOOS,
			GOARCH:       baseline.GOARCH,
			Runner:       baseline.Runner,
			ScratchFS:    baseline.ScratchFS,
			FilesPerSec:  baseline.FilesPerSec,
			PeakRSSBytes: int64(float64(baseline.PeakRSSBytes) * 1.16),
		}
		err := CheckRegression(baseline, current, ceilingBytes)
		if err == nil {
			t.Fatalf("CheckRegression(committed baseline) = nil, want error")
		}
		if !containsFold(err.Error(), "RSS") {
			t.Errorf("error %q does not mention expected hint %q", err.Error(), "RSS")
		}
	})
}
