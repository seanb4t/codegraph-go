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

	FilesPerSec          float64 `json:"files_per_sec"`
	BytesPerSec          float64 `json:"bytes_per_sec"`
	QueryLatencyMedianMS float64 `json:"query_latency_median_ms"`
	PeakRSSBytes         int64   `json:"peak_rss_bytes"`
	ColdStartMS          float64 `json:"cold_start_ms"`
}
