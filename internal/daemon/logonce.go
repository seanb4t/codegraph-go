package daemon

import (
	"log"
	"sync"
)

// logf is log.Printf indirected behind a package-level func var, the same
// test-seam convention as startTimeOf, so tests can capture what this
// package logs without replacing the global logger's output. It writes to
// stderr: serve --mcp reaches this package, and its stdout carries the MCP
// protocol.
var logf = log.Printf

// loggedKeys holds every key logOnce has logged in this process.
var (
	loggedMu   sync.Mutex
	loggedKeys = map[string]struct{}{}
)

// logOnce logs through logf the first time this process sees key, and does
// nothing for the same key after that. A failure that repeats on every
// registry listing, such as a file that can never be removed, is reported
// once instead of on each listing.
func logOnce(key, format string, args ...any) {
	loggedMu.Lock()
	_, seen := loggedKeys[key]
	loggedKeys[key] = struct{}{}
	loggedMu.Unlock()
	if !seen {
		logf(format, args...)
	}
}
