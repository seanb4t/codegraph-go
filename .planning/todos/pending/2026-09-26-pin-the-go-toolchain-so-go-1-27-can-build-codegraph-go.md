---
created: 2026-09-26T17:27:46.717Z
title: Pin the Go toolchain so Go 1.27 can build codegraph-go
area: build
severity: minor
files:
  - go.mod:3
  - Taskfile.yml:240
  - .github/workflows/ci.yml:54-56
---

## Problem

On a machine whose default Go is 1.27.x, `go build ./...` and `go test ./...` fail. This was observed with go1.27.1 on darwin/arm64 on 2026-09-26, during v0.15.0 Phase 2's post-merge gate. The failure is in `github.com/cockroachdb/swiss`, which pebble pulls in:

```
swiss@v0.0.0-20251224182025-b0f6560f979b/map.go:286:7: undefined: hashFn
map.go:337:14: undefined: getRuntimeHasher
map.go:338:22: undefined: fastrand64
```

swiss links into Go runtime internals that 1.27 changed. `go.mod` declares `go 1.26.6` with no `toolchain` line, and `GOTOOLCHAIN=auto` only ever *upgrades*, so the build runs on the local 1.27.1 instead of the declared 1.26.6. With `GOTOOLCHAIN=go1.26.6` the build passes and 53 packages pass `go test ./...`; `internal/daemon` is run on its own, as `Taskfile.yml` does.

CI is unaffected: `ci.yml` uses `setup-go` with `go-version-file: go.mod`, so it builds on 1.26.6. `test:tmux` in `Taskfile.yml` already pins `GOTOOLCHAIN=go1.26.6`, so the repo has hit this before. Contributors and agents on a newer local Go get a confusing compile error in a transitive dependency.

## Solution

Pick one, then verify on Go 1.27.1 with plain `go build ./...`:

- Add `toolchain go1.26.6` to `go.mod`, which makes `GOTOOLCHAIN=auto` select 1.26.6. Check how this interacts with the isolated `go.tool*.mod` modfiles and the `GOWORK=off` invocations. `TestToolModfilesRemainIsolated` and its siblings may assert on modfile shape.
- Or set `GOTOOLCHAIN` centrally in the Taskfile (`env:`) and document it in CONTRIBUTING. This only helps `task …` invocations, not bare `go`.
- Or bump pebble or swiss to a version that works with Go 1.27, if upstream has one. This is the real fix, but it touches the storage dependency; run the full storage and integration suites.
