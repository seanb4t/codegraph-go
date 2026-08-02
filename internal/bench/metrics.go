package bench

// Metrics is a plain json-tagged data holder for one benchmark run's
// results — no methods with side effects, no network, no shelling out
// (mirrors internal/version.VersionInfo's discipline). Consumed by the
// committed-baseline regression gate (Plan 08-06) and the head-to-head
// Go-vs-TS runner (Plan 08-07).
type Metrics struct {
	Subject string `json:"subject"` // e.g. "go" or "ts"
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
