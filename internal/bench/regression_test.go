package bench

import "testing"

// TestCheckRegression drives the committed-baseline + tolerance-band
// regression gate (PERF-02) plus the independent absolute peak-RSS ceiling
// (INDX-06). See regression.go for the policy this proves.
func TestCheckRegression(t *testing.T) {
	const ceiling = int64(1_000_000_000) // 1GB absolute bounded-memory budget

	defaultBaseline := Metrics{
		FilesPerSec:  100.0,
		PeakRSSBytes: 500_000_000,
	}

	tests := []struct {
		name     string
		baseline Metrics // zero value means "use defaultBaseline"
		current  Metrics
		ceiling  int64
		wantErr  bool
		errHint  string // substring the error message must contain
	}{
		{
			name: "clean run: faster and smaller, under ceiling passes",
			current: Metrics{
				FilesPerSec:  110.0,
				PeakRSSBytes: 400_000_000,
			},
			ceiling: ceiling,
			wantErr: false,
		},
		{
			name: "throughput 9% slower: within band passes",
			current: Metrics{
				FilesPerSec:  91.0, // 9% slower than 100
				PeakRSSBytes: 500_000_000,
			},
			ceiling: ceiling,
			wantErr: false,
		},
		{
			name: "throughput 11% slower: exceeds band fails",
			current: Metrics{
				FilesPerSec:  89.0, // 11% slower than 100
				PeakRSSBytes: 500_000_000,
			},
			ceiling: ceiling,
			wantErr: true,
			errHint: "throughput",
		},
		{
			name: "peak RSS 14% larger: within band passes",
			current: Metrics{
				FilesPerSec:  100.0,
				PeakRSSBytes: 570_000_000, // +14%
			},
			ceiling: ceiling,
			wantErr: false,
		},
		{
			name: "peak RSS 16% larger: exceeds band fails",
			current: Metrics{
				FilesPerSec:  100.0,
				PeakRSSBytes: 580_000_000, // +16%
			},
			ceiling: ceiling,
			wantErr: true,
			errHint: "RSS",
		},
		{
			// A baseline that already sits close to the ceiling: growing
			// only +10.5% (well within the 15% RSS tolerance band) still
			// pushes current over the absolute ceiling. Proves the ceiling
			// fires independently of the relative-delta check.
			name: "above absolute ceiling fails even when relative RSS delta is in-band",
			baseline: Metrics{
				FilesPerSec:  100.0,
				PeakRSSBytes: 950_000_000,
			},
			current: Metrics{
				FilesPerSec:  100.0,
				PeakRSSBytes: 1_050_000_000, // +10.5% vs baseline, but > 1GB ceiling
			},
			ceiling: ceiling,
			wantErr: true,
			errHint: "ceiling",
		},
		{
			name: "below absolute ceiling passes",
			baseline: Metrics{
				FilesPerSec:  100.0,
				PeakRSSBytes: 950_000_000,
			},
			current: Metrics{
				FilesPerSec:  100.0,
				PeakRSSBytes: 980_000_000, // +3.2% vs baseline, under 1GB ceiling
			},
			ceiling: ceiling,
			wantErr: false,
		},
		{
			name: "zero baseline throughput yields a clear error, not a panic",
			baseline: Metrics{
				FilesPerSec:  0,
				PeakRSSBytes: 500_000_000,
			},
			current: Metrics{
				FilesPerSec:  50.0,
				PeakRSSBytes: 500_000_000,
			},
			ceiling: ceiling,
			wantErr: true,
			errHint: "baseline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.baseline
			if b == (Metrics{}) {
				b = defaultBaseline
			}

			err := CheckRegression(b, tt.current, tt.ceiling)
			if tt.wantErr && err == nil {
				t.Fatalf("CheckRegression() = nil, want error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("CheckRegression() = %v, want nil", err)
			}
			if tt.wantErr && tt.errHint != "" {
				if !containsFold(err.Error(), tt.errHint) {
					t.Errorf("error %q does not mention expected hint %q", err.Error(), tt.errHint)
				}
			}
		})
	}
}

// containsFold is a tiny case-insensitive substring check so the test
// doesn't need to import strings twice with different casing assumptions
// about the error message wording.
func containsFold(s, substr string) bool {
	sl := []rune(s)
	subl := []rune(substr)
	for i := 0; i+len(subl) <= len(sl); i++ {
		match := true
		for j := range subl {
			a, b := sl[i+j], subl[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
