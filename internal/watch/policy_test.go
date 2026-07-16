package watch

import (
	"runtime"
	"testing"
)

// TestWatchDisabledReason is the RED contract for WATCH-03 (D-04/D-09/D-10):
// a table-driven precedence test against the not-yet-implemented Probe /
// WatchDisabledReason API. Until policy.go lands (Task 2), this file fails
// to build — undefined: Probe / WatchDisabledReason.
func TestWatchDisabledReason(t *testing.T) {
	cases := []struct {
		name        string
		projectRoot string
		probe       Probe
		want        string
	}{
		{
			name:        "default: no flags, no env, not WSL -> watch runs",
			projectRoot: "/repo",
			probe: Probe{
				Env:   func(string) string { return "" },
				IsWSL: func() bool { return false },
			},
			want: "",
		},
		{
			name:        "NoWatch flag true -> disabled with verbatim reason",
			projectRoot: "/repo",
			probe: Probe{
				Env:     func(string) string { return "" },
				IsWSL:   func() bool { return false },
				NoWatch: true,
			},
			want: "CODEGRAPH_NO_WATCH=1 is set",
		},
		{
			name:        "CODEGRAPH_NO_WATCH=1 env (flag false) -> SAME verbatim reason as the flag",
			projectRoot: "/repo",
			probe: Probe{
				Env: func(k string) string {
					if k == "CODEGRAPH_NO_WATCH" {
						return "1"
					}
					return ""
				},
				IsWSL: func() bool { return false },
			},
			want: "CODEGRAPH_NO_WATCH=1 is set",
		},
		{
			name:        "NoWatch beats ForceWatch: both true -> disabled (opt-out always wins, tier 1 before tier 2)",
			projectRoot: "/repo",
			probe: Probe{
				Env:        func(string) string { return "" },
				IsWSL:      func() bool { return false },
				NoWatch:    true,
				ForceWatch: true,
			},
			want: "CODEGRAPH_NO_WATCH=1 is set",
		},
		{
			name:        "ForceWatch flag beats WSL auto-detect",
			projectRoot: "/mnt/c/repo",
			probe: Probe{
				Env:        func(string) string { return "" },
				IsWSL:      func() bool { return true },
				ForceWatch: true,
			},
			want: "",
		},
		{
			name:        "CODEGRAPH_FORCE_WATCH=1 env behaves identically to the ForceWatch flag",
			projectRoot: "/mnt/c/repo",
			probe: Probe{
				Env: func(k string) string {
					if k == "CODEGRAPH_FORCE_WATCH" {
						return "1"
					}
					return ""
				},
				IsWSL: func() bool { return true },
			},
			want: "",
		},
		{
			name:        "WSL + /mnt/c drive, no flags/force -> disabled with verbatim WSL reason",
			projectRoot: "/mnt/c/repo",
			probe: Probe{
				Env:   func(string) string { return "" },
				IsWSL: func() bool { return true },
			},
			want: "project is on a WSL2 /mnt/ drive, where recursive file watching is too slow to be reliable",
		},
		{
			name:        "WSL + /mnt/wsl/anything -> watch runs (single-letter regex excludes /mnt/wsl)",
			projectRoot: "/mnt/wsl/anything",
			probe: Probe{
				Env:   func(string) string { return "" },
				IsWSL: func() bool { return true },
			},
			want: "",
		},
		{
			name:        "WSL + backslash-normalized project root still matches the /mnt/ mount",
			projectRoot: `\mnt\c\repo`,
			probe: Probe{
				Env:   func(string) string { return "" },
				IsWSL: func() bool { return true },
			},
			want: "project is on a WSL2 /mnt/ drive, where recursive file watching is too slow to be reliable",
		},
		{
			name:        "not WSL + /mnt/c drive -> watch runs (mount check irrelevant when not WSL)",
			projectRoot: "/mnt/c/repo",
			probe: Probe{
				Env:   func(string) string { return "" },
				IsWSL: func() bool { return false },
			},
			want: "",
		},
		{
			name:        `strict env: CODEGRAPH_NO_WATCH="true" does NOT disable`,
			projectRoot: "/repo",
			probe: Probe{
				Env: func(k string) string {
					if k == "CODEGRAPH_NO_WATCH" {
						return "true"
					}
					return ""
				},
				IsWSL: func() bool { return false },
			},
			want: "",
		},
		{
			name:        `strict env: CODEGRAPH_NO_WATCH="yes" does NOT disable`,
			projectRoot: "/repo",
			probe: Probe{
				Env: func(k string) string {
					if k == "CODEGRAPH_NO_WATCH" {
						return "yes"
					}
					return ""
				},
				IsWSL: func() bool { return false },
			},
			want: "",
		},
		{
			name:        `strict env: CODEGRAPH_NO_WATCH="0" does NOT disable`,
			projectRoot: "/repo",
			probe: Probe{
				Env: func(k string) string {
					if k == "CODEGRAPH_NO_WATCH" {
						return "0"
					}
					return ""
				},
				IsWSL: func() bool { return false },
			},
			want: "",
		},
		{
			name:        `strict env: CODEGRAPH_NO_WATCH=" 1" (padded) does NOT disable`,
			projectRoot: "/repo",
			probe: Probe{
				Env: func(k string) string {
					if k == "CODEGRAPH_NO_WATCH" {
						return " 1"
					}
					return ""
				},
				IsWSL: func() bool { return false },
			},
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := WatchDisabledReason(tc.projectRoot, tc.probe)
			if got != tc.want {
				t.Fatalf("WatchDisabledReason(%q, ...) = %q, want %q", tc.projectRoot, got, tc.want)
			}
		})
	}
}

// TestDetectWSL exercises the reset hook and the caching property only —
// deliberately does not assert a specific true/false result, since the real
// os.Getenv/GOOS/proc-version probe depends on the host running the test.
func TestDetectWSL(t *testing.T) {
	if runtime.GOOS != "linux" {
		// D-10: DetectWSL is unconditionally false off-linux. This is the
		// one host-independent assertion we CAN make safely on a non-linux
		// CI/dev machine.
		resetWSLCacheForTests()
		if got := DetectWSL(); got != false {
			t.Fatalf("DetectWSL() on GOOS=%s = %v, want false", runtime.GOOS, got)
		}
		return
	}

	resetWSLCacheForTests()
	first := DetectWSL()
	second := DetectWSL()
	if first != second {
		t.Fatalf("DetectWSL() not cached: first call = %v, second call = %v", first, second)
	}

	// After an explicit reset, DetectWSL re-evaluates (may agree or
	// disagree with the prior cached value on a real host — we only assert
	// that the reset hook triggers a fresh evaluation that is itself stable
	// across the next two calls).
	resetWSLCacheForTests()
	third := DetectWSL()
	fourth := DetectWSL()
	if third != fourth {
		t.Fatalf("DetectWSL() not cached after reset: third call = %v, fourth call = %v", third, fourth)
	}
}
