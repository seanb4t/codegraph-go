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
		{
			// Regression test for the perf-gate-throughput-regress debug
			// session: a baseline recorded on one GOOS/GOARCH must never be
			// silently compared against a current run on a different one -
			// this harness's own headtohead data shows 6.7x-148x throughput
			// deltas between darwin/arm64 and linux/amd64 on identical code,
			// so a cross-platform comparison is not noise, it's invalid.
			// FilesPerSec/PeakRSSBytes are set identically on both sides so
			// this case fails ONLY on the platform check, isolating it from
			// the numeric-tolerance checks above.
			name: "GOOS/GOARCH mismatch between baseline and current fails even when metrics are identical",
			baseline: Metrics{
				GOOS:         "darwin",
				GOARCH:       "arm64",
				FilesPerSec:  100.0,
				PeakRSSBytes: 500_000_000,
			},
			current: Metrics{
				GOOS:         "linux",
				GOARCH:       "amd64",
				FilesPerSec:  100.0,
				PeakRSSBytes: 500_000_000,
			},
			ceiling: ceiling,
			wantErr: true,
			errHint: "platform",
		},
		{
			// Boundary neighbor: GOARCH agrees, only GOOS differs. The
			// debug session's own blind-spot note flagged that the
			// fully-mismatched pair above does not prove the check reads
			// BOTH fields — a guard comparing only GOARCH would pass that
			// case and this one would catch it. darwin/arm64 vs
			// linux/arm64 is also the exact pair the same-platform control
			// measured ~3x apart on one physical machine.
			name: "GOOS differs while GOARCH matches fails",
			baseline: Metrics{
				GOOS:         "darwin",
				GOARCH:       "arm64",
				FilesPerSec:  100.0,
				PeakRSSBytes: 500_000_000,
			},
			current: Metrics{
				GOOS:         "linux",
				GOARCH:       "arm64",
				FilesPerSec:  100.0,
				PeakRSSBytes: 500_000_000,
			},
			ceiling: ceiling,
			wantErr: true,
			errHint: "platform",
		},
		{
			// The mirror boundary neighbor: GOOS agrees, only GOARCH
			// differs. Catches a guard that compares only GOOS.
			name: "GOARCH differs while GOOS matches fails",
			baseline: Metrics{
				GOOS:         "linux",
				GOARCH:       "amd64",
				FilesPerSec:  100.0,
				PeakRSSBytes: 500_000_000,
			},
			current: Metrics{
				GOOS:         "linux",
				GOARCH:       "arm64",
				FilesPerSec:  100.0,
				PeakRSSBytes: 500_000_000,
			},
			ceiling: ceiling,
			wantErr: true,
			errHint: "platform",
		},
		{
			// The "no more" side of the guard: a genuinely matching
			// platform pair must still pass. Without this, a guard that
			// rejected EVERY comparison would satisfy all the failure
			// cases above and permanently red the gate.
			name: "matching GOOS/GOARCH on both sides passes",
			baseline: Metrics{
				GOOS:         "linux",
				GOARCH:       "amd64",
				FilesPerSec:  100.0,
				PeakRSSBytes: 500_000_000,
			},
			current: Metrics{
				GOOS:         "linux",
				GOARCH:       "amd64",
				FilesPerSec:  98.0,
				PeakRSSBytes: 510_000_000,
			},
			ceiling: ceiling,
			wantErr: false,
		},
		{
			// Unattributed on BOTH sides matches and is allowed: callers
			// that construct Metrics without platform fields (the eight
			// cases above, and any pre-attribution baseline) must not be
			// broken by the guard. Asserted explicitly rather than left
			// as an incidental property of the other cases.
			name: "unattributed platform on both sides passes",
			current: Metrics{
				FilesPerSec:  100.0,
				PeakRSSBytes: 500_000_000,
			},
			ceiling: ceiling,
			wantErr: false,
		},
		{
			// Half-attributed is a mismatch, deliberately. A baseline
			// with no recorded platform cannot be validated against an
			// attributed run — "unknown" is not a wildcard. Failing loud
			// here is the whole point of this fix: the alternative is
			// gating on a number whose provenance nobody can check.
			name: "attributed current against unattributed baseline fails",
			baseline: Metrics{
				FilesPerSec:  100.0,
				PeakRSSBytes: 500_000_000,
			},
			current: Metrics{
				GOOS:         "linux",
				GOARCH:       "amd64",
				FilesPerSec:  100.0,
				PeakRSSBytes: 500_000_000,
			},
			ceiling: ceiling,
			wantErr: true,
			errHint: "platform",
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
