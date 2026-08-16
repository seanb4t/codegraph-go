// Command publishcheck is a pure-Go verifier for tools/bench/runner's
// `-mode publish` JSON output. It exists so publish mode's end-to-end
// acceptance criteria can be enforced without shelling out to a
// second-language runtime (closes review finding 06-01:246), and 06-04
// reuses this same binary with -emit-rows to generate the published
// Markdown data rows rather than hand-typing them.
//
// publishcheck unmarshals -file into []internal/bench.Metrics (reusing
// the shared struct rather than redeclaring the shape, so a renamed or
// retagged field is caught by the compiler instead of silently going
// unchecked) and asserts:
//   - the file parses as a JSON array;
//   - its length equals -want-records;
//   - the sorted set of `repo` values equals the sorted -want-repos set;
//   - every record's `subject` equals -want-subject;
//   - every record's `median_of_trials` equals -want-median-of-trials;
//   - every one of files_per_sec, bytes_per_sec, query_latency_median_ms,
//     peak_rss_bytes, cold_start_ms is strictly positive in every record.
//
// Any violation exits non-zero naming the offending record and field.
// On success it prints one line naming the verified totals plus the
// SHA-256 of the input file's bytes, so a caller (06-04) can prove byte
// identity between a stdout capture and a committed artifact without
// depending on a platform-specific checksum tool.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/seanb4t/codegraph-go/internal/bench"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "publishcheck: %v\n", err)
		os.Exit(1)
	}
}

// config holds every flag this tool accepts.
type config struct {
	file             string
	wantRecords      int
	wantRepos        string
	wantSubject      string
	wantMedianTrials int
	emitRows         bool
}

func parseFlags(args []string) (config, error) {
	fs := flag.NewFlagSet("publishcheck", flag.ContinueOnError)
	var cfg config
	fs.StringVar(&cfg.file, "file", "", "path to a publish-mode JSON capture (required)")
	fs.IntVar(&cfg.wantRecords, "want-records", 0, "expected number of records (required, must be > 0)")
	fs.StringVar(&cfg.wantRepos, "want-repos", "", "comma-separated set of expected repo values (required)")
	fs.StringVar(&cfg.wantSubject, "want-subject", "go", "expected subject value for every record")
	fs.IntVar(&cfg.wantMedianTrials, "want-median-of-trials", 1, "expected median_of_trials value for every record")
	fs.BoolVar(&cfg.emitRows, "emit-rows", false, "also print one pipe-delimited Markdown data row per record, in sorted repo order (06-04 reuses this to generate the published table)")

	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if cfg.file == "" {
		return config{}, fmt.Errorf("-file is required")
	}
	if cfg.wantRecords <= 0 {
		return config{}, fmt.Errorf("-want-records must be > 0, got %d", cfg.wantRecords)
	}
	if cfg.wantRepos == "" {
		return config{}, fmt.Errorf("-want-repos is required")
	}
	return cfg, nil
}

func run(args []string) error {
	cfg, err := parseFlags(args)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(cfg.file)
	if err != nil {
		return fmt.Errorf("read %s: %w", cfg.file, err)
	}

	var records []bench.Metrics
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("parse %s as a JSON array of bench.Metrics: %w", cfg.file, err)
	}

	if err := checkRecords(records, cfg); err != nil {
		return err
	}

	repos := sortedRepos(records)
	allSubjectGo := true
	for _, r := range records {
		if r.Subject != "go" {
			allSubjectGo = false
			break
		}
	}
	sum := sha256.Sum256(data)

	fmt.Printf("PUBLISH_RECORDS=%d REPOS=%s ALL_SUBJECT_GO=%t ALL_METRICS_POSITIVE=true SHA256=%s\n",
		len(records), strings.Join(repos, ","), allSubjectGo, hex.EncodeToString(sum[:]))

	if cfg.emitRows {
		emitRows(records)
	}

	return nil
}

// checkRecords asserts every acceptance property this verifier exists
// to enforce. Any violation returns a non-nil error naming the
// offending record (by index and repo) and field — an instrument that
// cannot say WHY it rejected an input is not much better than one that
// cannot reject at all.
func checkRecords(records []bench.Metrics, cfg config) error {
	// An empty JSON array unmarshals successfully into a zero-length
	// slice; the record-count check below is what makes that fail
	// rather than pass vacuously (rule 84d1gfpywd), since -want-records
	// is required to be > 0.
	if len(records) != cfg.wantRecords {
		return fmt.Errorf("record count = %d, want %d", len(records), cfg.wantRecords)
	}

	wantRepos := strings.Split(cfg.wantRepos, ",")
	sort.Strings(wantRepos)
	gotRepos := sortedRepos(records)
	if !equalStringSlices(gotRepos, wantRepos) {
		return fmt.Errorf("repo set = %v, want %v", gotRepos, wantRepos)
	}

	for i, r := range records {
		if r.Subject != cfg.wantSubject {
			return fmt.Errorf("record %d (repo %q): subject = %q, want %q", i, r.Repo, r.Subject, cfg.wantSubject)
		}
		if r.MedianOfTrials != cfg.wantMedianTrials {
			return fmt.Errorf("record %d (repo %q): median_of_trials = %d, want %d", i, r.Repo, r.MedianOfTrials, cfg.wantMedianTrials)
		}
		if r.FilesPerSec <= 0 {
			return fmt.Errorf("record %d (repo %q): files_per_sec = %v, want > 0", i, r.Repo, r.FilesPerSec)
		}
		if r.BytesPerSec <= 0 {
			return fmt.Errorf("record %d (repo %q): bytes_per_sec = %v, want > 0", i, r.Repo, r.BytesPerSec)
		}
		if r.QueryLatencyMedianMS <= 0 {
			return fmt.Errorf("record %d (repo %q): query_latency_median_ms = %v, want > 0", i, r.Repo, r.QueryLatencyMedianMS)
		}
		if r.PeakRSSBytes <= 0 {
			return fmt.Errorf("record %d (repo %q): peak_rss_bytes = %v, want > 0", i, r.Repo, r.PeakRSSBytes)
		}
		if r.ColdStartMS <= 0 {
			return fmt.Errorf("record %d (repo %q): cold_start_ms = %v, want > 0", i, r.Repo, r.ColdStartMS)
		}
	}

	return nil
}

// sortedRepos returns records' repo values, sorted, for both the
// repo-set comparison and the printed REPOS= token.
func sortedRepos(records []bench.Metrics) []string {
	repos := make([]string, 0, len(records))
	for _, r := range records {
		repos = append(repos, r.Repo)
	}
	sort.Strings(repos)
	return repos
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// emitRows prints one pipe-delimited Markdown data row per record, in
// sorted repo order, with the rounding the published document uses.
// This is 06-04's single source of the document's numbers — it never
// hand-types a figure this binary already measured.
func emitRows(records []bench.Metrics) {
	sorted := make([]bench.Metrics, len(records))
	copy(sorted, records)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Repo < sorted[j].Repo })

	for _, r := range sorted {
		fmt.Printf("| %s | %.1f | %.1f | %.3f | %d | %.3f |\n",
			r.Repo, r.FilesPerSec, r.BytesPerSec, r.QueryLatencyMedianMS, r.PeakRSSBytes, r.ColdStartMS)
	}
}
