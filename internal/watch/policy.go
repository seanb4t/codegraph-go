package watch

import (
	"errors"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

// ErrWatchDisabled is the sentinel error internal/daemon wraps when
// WatchDisabledReason reports a non-empty reason, and internal/cli errors.Is
// against to print the human-visible disabled message (D-09/D-11). It lives
// in internal/watch (not internal/daemon) so both internal/daemon (wraps it)
// and internal/cli (errors.Is it) can import it without an import cycle.
var ErrWatchDisabled = errors.New("daemon: watching is disabled by policy")

// DisabledError carries the human-readable disabled reason alongside the
// ErrWatchDisabled sentinel (03-REVIEW.md IN-05): daemon.Run returns this
// type so CLI consumers extract the exact reason Run's own policy gate saw
// via errors.As, instead of re-deriving it with fresh WatchDisabledReason
// calls that can silently desynchronize (different root normalization,
// different Probe inputs). errors.Is(err, ErrWatchDisabled) keeps working
// everywhere via the Is method below.
type DisabledError struct{ Reason string }

// Error preserves the exact string the previous fmt.Errorf("%w: %s", ...)
// wrap produced, so any consumer of the rendered message is unaffected.
func (e *DisabledError) Error() string {
	return ErrWatchDisabled.Error() + ": " + e.Reason
}

// Is makes errors.Is(err, ErrWatchDisabled) match a *DisabledError, keeping
// every existing sentinel check working unchanged.
func (e *DisabledError) Is(target error) bool { return target == ErrWatchDisabled }

// Probe carries WatchDisabledReason's inputs — env lookup, WSL detection,
// and the two CLI flags — as explicit values/functions rather than reading
// process state directly (D-05: the watcher is in-process, so this
// package never mutates the process environment; CODEGRAPH_NO_WATCH is
// read directly via the Env probe). Nil Env/IsWSL default to
// os.Getenv/DetectWSL inside WatchDisabledReason, so callers only need to
// set them explicitly in tests.
type Probe struct {
	// Env looks up an environment variable by name. Defaults to os.Getenv.
	Env func(string) string
	// IsWSL reports whether the process is running under WSL2. Defaults to
	// the cached DetectWSL.
	IsWSL func() bool
	// NoWatch is the --no-watch flag (D-01/D-02): opt-out, always wins.
	NoWatch bool
	// ForceWatch is the --watch flag, repurposed as the explicit force-on
	// escape hatch (D-03): overrides the WSL2 auto-off, but a plain
	// default-on run does not override anything.
	ForceWatch bool
}

// isWindowsDriveMountRe matches a WSL2 /mnt/<single-letter> drive mount —
// deliberately single-letter only (D-10), so /mnt/wsl/... (WSL's own
// virtual filesystem, not a slow Windows drive) stays on the fast path.
var isWindowsDriveMountRe = regexp.MustCompile(`(?i)^/mnt/[a-z](/|$)`)

// isWindowsDriveMount reports whether projectRoot is on a WSL2 /mnt/<drive>
// mount. An unconditional backslash-to-forward-slash replace is required —
// filepath.ToSlash is NOT equivalent here, since on Linux (the only GOOS
// DetectWSL ever reports true for) filepath.Separator is already '/',
// making ToSlash a no-op that would leave WSL2 paths reported with
// Windows-style backslashes unnormalized.
func isWindowsDriveMount(projectRoot string) bool {
	return isWindowsDriveMountRe.MatchString(strings.ReplaceAll(projectRoot, `\`, "/"))
}

// WatchDisabledReason returns "" when the watcher should run, or a short
// human-readable reason (the documented disabled-reason strings, D-12/D-13)
// when it should not. Precedence, first match wins (D-04):
//
//  1. NoWatch flag OR Env("CODEGRAPH_NO_WATCH")=="1" -> off
//  2. ForceWatch flag OR Env("CODEGRAPH_FORCE_WATCH")=="1" -> on (beats auto-detect)
//  3. WSL2 + /mnt/[a-z] drive -> off
//  4. default -> on
//
// Env comparisons are strict `== "1"` (D-10) — never strconv.ParseBool or
// non-empty truthiness — so unrelated env noise ("true", "yes", "0", padded
// values) can never silently flip watch state (T-03-01).
//
// The --no-watch flag and the CODEGRAPH_NO_WATCH env var are two inputs
// to the same tier-1 check, both producing the same reason string, so the
// disabled message is identical whichever one triggered it.
func WatchDisabledReason(projectRoot string, p Probe) string {
	if p.Env == nil {
		p.Env = os.Getenv
	}
	if p.IsWSL == nil {
		p.IsWSL = DetectWSL
	}

	if p.NoWatch || p.Env("CODEGRAPH_NO_WATCH") == "1" {
		return "CODEGRAPH_NO_WATCH=1 is set"
	}
	if p.ForceWatch || p.Env("CODEGRAPH_FORCE_WATCH") == "1" {
		return ""
	}
	// D-13: this message says "file watching" rather than naming fs.watch,
	// a Node-specific API, since this is a Go binary (documented choice;
	// this string is human-facing stderr, never parsed).
	if p.IsWSL() && isWindowsDriveMount(projectRoot) {
		return "project is on a WSL2 /mnt/ drive, where recursive file watching is too slow to be reliable"
	}
	return ""
}

var (
	wslOnce  sync.Once
	wslCache bool
)

// DetectWSL reports whether the process is running under WSL2, cached after
// the first call (sync.Once-guarded, D-10): non-linux GOOS is
// unconditionally false (no I/O); WSL_DISTRO_NAME
// or WSL_INTEROP env presence is true (no I/O); otherwise /proc/version is
// read and lowercased, true if it contains "microsoft" or "wsl"; any read
// failure degrades to false (never panics — V5/V12: a hostile or missing
// /proc must not crash or hang the process).
func DetectWSL() bool {
	wslOnce.Do(func() {
		wslCache = detectWSLUncached()
	})
	return wslCache
}

func detectWSLUncached() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
		return true
	}
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	version := strings.ToLower(string(data))
	return strings.Contains(version, "microsoft") || strings.Contains(version, "wsl")
}

// resetWSLCacheForTests clears DetectWSL's cached result so tests can force
// a fresh evaluation. Unexported, no exported setter — mirrors
// internal/daemon.Daemon's onSyncStart test-only control seam convention.
// Production code never calls this.
func resetWSLCacheForTests() {
	wslOnce = sync.Once{}
	wslCache = false
}
