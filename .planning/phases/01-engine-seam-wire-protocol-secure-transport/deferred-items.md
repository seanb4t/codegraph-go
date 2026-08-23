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

## `buf generate`/`buf build` misidentify the controlling workspace inside a linked git worktree nested under its own main checkout — sandbox-only, does not affect CI

**Found during:** 01-07 Task 1, first invocation of `task proto:gen` /
the new `task proto:drift` from inside this execution's Claude Code
worktree (`/Volumes/Code/.../.claude/worktrees/agent-<id>/`, a git linked
worktree physically nested inside the main checkout's own working tree).

**Issue:** `buf build`/`buf generate` invoked with the implicit `"."`
input (both `task proto:gen`'s and, before this fix, the drafted
`task proto:drift`'s default) fail with:

```
internal/schema/graph.proto:23:9:`Node` declared multiple times
internal/uiproto/uiv1/ui.proto:22:9:`UIService` declared multiple times
```

`buf ls-files` with the same implicit input shows why: it enumerates the
same `.proto` files from BOTH this worktree's own tree AND the outer main
checkout's tree at the same relative paths (and, when a sibling agent's
worktree happens to exist alongside this one under `.claude/worktrees/`,
from that sibling too) — five file paths for what should be two logical
files. `buf`'s own debug log (`--debug`) shows its "controlling workspace"
termination search resolving to the OUTER main checkout's absolute path
rather than this worktree's own directory, even though this worktree has
its own `buf.yaml` at its own root. The most likely cause: this worktree's
`.git` is a gitlink FILE (`gitdir: .../.git/worktrees/agent-<id>`), not a
directory, and buf's git-awareness appears to walk upward past it looking
for a literal `.git` DIRECTORY, landing on the outer checkout's real
`.git/` and treating that outer tree as the buf module root — which then
recurses into every nested worktree directory (`.claude/worktrees/*`)
sitting inside it.

**Verified workaround (applied in `proto:drift`'s own cmds:, not in
`proto:gen`):** passing buf an EXPLICIT absolute path as its input
(`buf generate -o "${scratch}/gen" "${repo_root}"` where `repo_root=$(pwd)`)
rather than relying on the implicit `"."` avoids the upward workspace
search entirely and regenerates only this worktree's own two proto files,
byte-identical to the committed output — confirmed both via `buf build`
returning cleanly and via a full `proto:drift` run reporting `compared 3
generated files` with zero drift.

**Not fixed in `proto:gen` here:** `01-07`'s `files_modified` does not
include `proto:gen`'s own task body (`Taskfile.yml`'s `proto:gen` task is
01-01's deliverable; this plan's own artifact contract says explicitly
"reuses rather than duplicates" it and "must not re-author" it). `proto:gen`
therefore still uses the implicit `"."` input and will exhibit the same
duplicate-declaration failure if ever run from inside a similarly-nested
linked worktree. This is understood to be a LOCAL SANDBOX artifact only —
this repository's CI runners (`namespace-profile-linux-amd64-4x8`, per
`ci.yml`) check out a normal, non-nested working tree with a real `.git`
directory, so `task proto:gen` in CI is unaffected. Any future plan
touching `proto:gen` should consider applying the same explicit-absolute-
path input fix for consistency and to protect future contributors who
also develop from inside a nested worktree layout.
