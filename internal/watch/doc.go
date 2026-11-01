// Package watch implements SYNC-01: a native, debounced filesystem watcher
// with no polling default. On macOS it uses one recursive FSEvents stream
// per repository root (watcher_darwin.go), because kqueue holds one file
// descriptor for every watched file and directory. Elsewhere it uses
// github.com/fsnotify/fsnotify (watcher_fsnotify.go, D-04). A burst of
// events is coalesced by a Debouncer (see debounce.go) into one flush call
// over the union of changed paths.
//
// internal/watch depends only on internal/indexer's exported ShouldSkipDir
// predicate — never on internal/graphstore or pebble directly (D-04a
// archtest boundary; this package has no storage concerns of its own).
package watch
