# Deferred Items — Phase 01

Out-of-scope discoveries logged per the executor's scope-boundary rule
(fix only what the current task's changes directly caused).

## `go mod tidy` fails on the main module — pre-existing, unrelated to 01-01

**Found during:** 01-01 Task 2, attempting to run `go mod tidy` to drop
the now-stale `// indirect` markers on `connectrpc.com/connect` and
`github.com/pkg/browser` after they became directly imported.

**Issue:** `go mod tidy` (no `-modfile`, the main `go.mod`) fails with:

```
go: finding module for package github.com/tree-sitter/tree-sitter-swift/bindings/go
go: github.com/seanb4t/codegraph-go/internal/parser/cgo imports
	github.com/alex-pinkus/tree-sitter-swift/bindings/go tested by
	github.com/alex-pinkus/tree-sitter-swift/bindings/go.test imports
	github.com/tree-sitter/tree-sitter-swift/bindings/go: module github.com/tree-sitter/tree-sitter-swift@latest found (v0.0.0-20220113184755-db675450dcc1), but does not contain package github.com/tree-sitter/tree-sitter-swift/bindings/go
```

This is a test-only import inside `github.com/alex-pinkus/tree-sitter-swift`'s
own upstream test file resolving to a renamed module
(`alex-pinkus/tree-sitter-swift` -> `tree-sitter/tree-sitter-swift`) whose
new location does not (yet) publish the same package path. It reproduces
identically without any of 01-01's changes present (verified by
inspecting `internal/parser/cgo`, which 01-01 never touches) and is
unrelated to the proto/uiserver work this plan performs.

**Consequence:** `connectrpc.com/connect` and `github.com/pkg/browser`
remain marked `// indirect` in `go.mod` even though `connectrpc.com/connect`
is now directly imported by `internal/uiserver` — cosmetic only; `go
build ./...` and `go vet ./...` are unaffected, since the `// indirect`
comment carries no build-time meaning to the Go toolchain.

**Not fixed here:** fixing the upstream `tree-sitter-swift` module
reference is out of scope for a UI-transport phase and touches a
different subsystem (`internal/parser/cgo`) this plan's `files_modified`
does not include.
