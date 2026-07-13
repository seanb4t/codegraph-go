package bench

import (
	"encoding/json"
	"testing"
)

func TestRSSNormalize(t *testing.T) {
	tests := []struct {
		name      string
		goos      string
		rawMaxrss int64
		wantBytes int64
		wantErr   bool
	}{
		{
			name:      "linux KB to bytes",
			goos:      "linux",
			rawMaxrss: 2048,
			wantBytes: 2097152,
			wantErr:   false,
		},
		{
			name:      "darwin bytes identity",
			goos:      "darwin",
			rawMaxrss: 2097152,
			wantBytes: 2097152,
			wantErr:   false,
		},
		{
			name:      "unsupported OS errors",
			goos:      "windows",
			rawMaxrss: 2048,
			wantBytes: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeMaxrss(tt.goos, tt.rawMaxrss)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeMaxrss(%q, %d) expected error, got nil", tt.goos, tt.rawMaxrss)
				}
				if got != 0 {
					t.Fatalf("normalizeMaxrss(%q, %d) expected zero value on error, got %d", tt.goos, tt.rawMaxrss, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeMaxrss(%q, %d) unexpected error: %v", tt.goos, tt.rawMaxrss, err)
			}
			if got != tt.wantBytes {
				t.Fatalf("normalizeMaxrss(%q, %d) = %d, want %d", tt.goos, tt.rawMaxrss, got, tt.wantBytes)
			}
		})
	}
}

func TestSmoke(t *testing.T) {
	want := Metrics{
		Subject:              "go",
		Repo:                 "codegraph-go",
		GOOS:                 "darwin",
		GOARCH:               "arm64",
		FilesPerSec:          1234.5,
		BytesPerSec:          987654.3,
		QueryLatencyMedianMS: 42.7,
		PeakRSSBytes:         2097152,
		ColdStartMS:          15.3,
	}

	data, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal(Metrics) unexpected error: %v", err)
	}

	var got Metrics
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal(Metrics) unexpected error: %v", err)
	}

	if got != want {
		t.Fatalf("round-tripped Metrics = %+v, want %+v", got, want)
	}
}
