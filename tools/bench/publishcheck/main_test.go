package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/seanb4t/codegraph-go/internal/bench"
)

// TestPublishCheck is the ONLY top-level test symbol in this package
// (cycle-3 HIGH 1 — the test-surface delta assertion in 06-01-PLAN.md's
// Task 1 enumerates added top-level names literally, so a second
// top-level Test* here would invalidate that literal list). All seven
// cases below run as t.Run subtests.
//
// The verifier under test is itself under test here because a
// verification instrument that has never been shown to reject a bad
// input is not much evidence that it can reject anything (the same
// positive-control discipline D-04 imposes on the phase's own census).
func TestPublishCheck(t *testing.T) {
	t.Run("well-formed two-record fixture passes and prints the expected line", func(t *testing.T) {
		file := writeFixture(t, twoGoodRecords())
		out, err := runCapture(t, "-file", file, "-want-records", "2", "-want-repos", "cockroachdb-pebble,weft-go")
		if err != nil {
			t.Fatalf("run: %v", err)
		}
		if !strings.Contains(out, "PUBLISH_RECORDS=2 REPOS=cockroachdb-pebble,weft-go ALL_SUBJECT_GO=true ALL_METRICS_POSITIVE=true SHA256=") {
			t.Errorf("unexpected output: %s", out)
		}
	})

	t.Run("short record count fails the want-records check", func(t *testing.T) {
		records := twoGoodRecords()[:1]
		file := writeFixture(t, records)
		_, err := runCapture(t, "-file", file, "-want-records", "2", "-want-repos", "weft-go")
		if err == nil {
			t.Fatal("want error for a one-record fixture checked against -want-records 2")
		}
	})

	t.Run("a non-go subject fails", func(t *testing.T) {
		// Deliberately "other", not "ts": tools/bench sits inside 06-04's
		// Phase-6 census surface, whose bounded pattern set would count a
		// quoted "ts" as retired subject framing (cycle-3 new trap).
		records := twoGoodRecords()
		records[1].Subject = "other"
		file := writeFixture(t, records)
		_, err := runCapture(t, "-file", file, "-want-records", "2", "-want-repos", "cockroachdb-pebble,weft-go")
		if err == nil {
			t.Fatal("want error for a record with subject \"other\"")
		}
	})

	t.Run("a non-positive metric fails, naming the record and field", func(t *testing.T) {
		records := twoGoodRecords()
		records[0].FilesPerSec = 0
		file := writeFixture(t, records)
		_, err := runCapture(t, "-file", file, "-want-records", "2", "-want-repos", "cockroachdb-pebble,weft-go")
		if err == nil {
			t.Fatal("want error for a record with files_per_sec = 0")
		}
		if !strings.Contains(err.Error(), "files_per_sec") {
			t.Errorf("error %q should name the offending field files_per_sec", err)
		}
	})

	t.Run("a mismatched repo set fails", func(t *testing.T) {
		records := twoGoodRecords()
		file := writeFixture(t, records)
		_, err := runCapture(t, "-file", file, "-want-records", "2", "-want-repos", "some-other-repo,weft-go")
		if err == nil {
			t.Fatal("want error for a repo set that does not match -want-repos")
		}
	})

	t.Run("an empty JSON array fails rather than passing vacuously", func(t *testing.T) {
		file := writeFixture(t, []bench.Metrics{})
		_, err := runCapture(t, "-file", file, "-want-records", "2", "-want-repos", "cockroachdb-pebble,weft-go")
		if err == nil {
			t.Fatal("want error for an empty JSON array (rule 84d1gfpywd)")
		}
	})

	t.Run("the printed SHA256 equals the input file's own SHA-256", func(t *testing.T) {
		records := twoGoodRecords()
		file := writeFixture(t, records)

		wantBytes, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		wantSum := sha256.Sum256(wantBytes)
		want := hex.EncodeToString(wantSum[:])

		out, err := runCapture(t, "-file", file, "-want-records", "2", "-want-repos", "cockroachdb-pebble,weft-go")
		if err != nil {
			t.Fatalf("run: %v", err)
		}
		got := extractSHA256(t, out)
		if got != want {
			t.Errorf("SHA256 = %q, want %q (the input file's own SHA-256)", got, want)
		}
	})
}

// --- fixtures and helpers ---

func twoGoodRecords() []bench.Metrics {
	return []bench.Metrics{
		{
			Subject:              "go",
			Repo:                 "weft-go",
			GOOS:                 "darwin",
			GOARCH:               "arm64",
			MedianOfTrials:       1,
			FilesPerSec:          100.5,
			BytesPerSec:          20000.0,
			QueryLatencyMedianMS: 5.5,
			PeakRSSBytes:         123456,
			ColdStartMS:          3.2,
		},
		{
			Subject:              "go",
			Repo:                 "cockroachdb-pebble",
			GOOS:                 "darwin",
			GOARCH:               "arm64",
			MedianOfTrials:       1,
			FilesPerSec:          200.5,
			BytesPerSec:          30000.0,
			QueryLatencyMedianMS: 6.5,
			PeakRSSBytes:         654321,
			ColdStartMS:          4.2,
		},
	}
}

func writeFixture(t *testing.T, records []bench.Metrics) string {
	t.Helper()
	data, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "publish.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

// runCapture invokes run with args, redirecting os.Stdout so the
// printed token line can be asserted against directly rather than only
// inferred from a nil error.
func runCapture(t *testing.T, args ...string) (string, error) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w

	runErr := run(args)

	os.Stdout = orig
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	return buf.String(), runErr
}

func extractSHA256(t *testing.T, out string) string {
	t.Helper()
	const marker = "SHA256="
	idx := strings.Index(out, marker)
	if idx < 0 {
		t.Fatalf("output missing %s: %s", marker, out)
	}
	return strings.TrimSpace(out[idx+len(marker):])
}
