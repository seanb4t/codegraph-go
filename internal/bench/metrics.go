package bench

// Metrics is a plain json-tagged data holder for one benchmark run's
// results — no methods with side effects, no network, no shelling out
// (mirrors internal/version.VersionInfo's discipline). Consumed by the
// committed-baseline regression gate and the absolute-numbers publisher
// (bench.yml's publish job).
type Metrics struct {
	Subject string `json:"subject"` // provenance label, e.g. "go" — not part of a measurement's identity
	Repo    string `json:"repo"`
	GOOS    string `json:"goos"`
	GOARCH  string `json:"goarch"`

	// Runner records the CI runner identity this Metrics was measured on
	// (e.g. a Namespace runs-on profile label such as
	// "namespace-profile-linux-amd64-4x8"), captured verbatim from the
	// workflow that ran it. It sits alongside GOOS/GOARCH because a
	// runner-class change (e.g. GitHub-hosted ubuntu-latest ->
	// Namespace) can hold GOOS/GOARCH constant while still changing the
	// measurement environment — namespace-profile-linux-amd64-4x8 IS
	// linux/amd64, so the GOOS/GOARCH guard alone is structurally blind
	// to that migration. Empty means "recorded before this field
	// existed" (e.g. any baseline.json predating this change), not an
	// error — measurement must not require it, since it is only
	// meaningful in CI. Runner does NOT yet participate in
	// CheckRegression's comparison; that lands separately, once a
	// Runner-labelled baseline exists to compare against.
	Runner string `json:"runner"`

	// ScratchFS records which filesystem class the regression corpus +
	// Pebble store were materialized on for THIS measurement: "tmpfs" or
	// "disk" (see tools/bench/runner's scratchFSTmpfs/scratchFSDisk).
	// Same philosophy as Runner: the storage frame is exactly as
	// load-bearing to a throughput comparison as the runner class is —
	// PERF-02's synthetic corpus is ~18MB of raw content but ~770MB of
	// actual on-disk footprint once small-file block overhead is
	// accounted for (120k files at ~4KB/block minimum each), and at that
	// file count the workload is IOPS/metadata-bound, not
	// bandwidth-bound. Comparing a tmpfs-recorded baseline against a
	// disk-recorded gate run (or vice versa) would be exactly the kind
	// of measurement-frame mismatch that produced the original
	// fictitious 10.6% "regression" — a different location for the same
	// failure class. Empty means "recorded before this field existed"
	// or "the caller supplied an explicit -scratch-dir and its
	// filesystem class wasn't characterized" — not an error. ScratchFS
	// does NOT yet participate in CheckRegression's comparison; wiring
	// that in, like Runner's, is a deliberately separate, later change.
	ScratchFS string `json:"scratch_fs"`

	// MedianOfTrials records how many INDEPENDENT measurement sessions
	// these numbers are the median of — where a session is a full
	// corpus-materialize + init + measure cycle, not a repeat of the
	// measured command inside one session (that inner repetition is
	// D-05's fixed median-of-5 and is not represented here).
	//
	// This is provenance, deliberately not a gate. Unlike GOOS/GOARCH, a
	// differing trial count is not a category error: median-of-3 and
	// median-of-5 estimate the same population median, just with
	// different precision, so refusing to compare them would be enforcing
	// a rule that isn't a validity rule. It is recorded and printed so
	// that a human reviewing a candidate baseline can see the procedure
	// that produced it — the thing nobody could see when a darwin
	// baseline was gating a linux runner.
	MedianOfTrials int `json:"median_of_trials"`

	FilesPerSec          float64 `json:"files_per_sec"`
	BytesPerSec          float64 `json:"bytes_per_sec"`
	QueryLatencyMedianMS float64 `json:"query_latency_median_ms"`
	PeakRSSBytes         int64   `json:"peak_rss_bytes"`
	ColdStartMS          float64 `json:"cold_start_ms"`
}
