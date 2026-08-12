---
created: 2026-08-10T22:39:46.876Z
title: Add golangci-lint with gofmt and idiomatic Go linters
area: ci
severity: minor
files:
  - go.tool-lint.mod:24
  - Taskfile.yml:10
  - Taskfile.yml:3163
  - Taskfile.yml:3521
  - internal/query/files_status_test.go
---

## Problem

**This repository has no formatting gate anywhere.** Verified 2026-08-10 across
all three surfaces during `/gsd-audit-uat`:

- `task lint` (Taskfile.yml:3521) is `vet` + `lint:actions` only
- no `.golangci.yml` exists at the repo root
- no workflow under `.github/workflows/` references `gofmt`

`task vet` runs `go vet ./...`, which does not check formatting. So `gofmt`
violations land and persist until someone notices by hand.

That is not hypothetical — it has produced three separate findings:

1. **`internal/query/files_status_test.go` is unformatted right now.** A WR-01
   test block is indented one level too deep. Committed and clean in the working
   tree; introduced by `ea2b889` (a phase-08 commit) and never caught.
2. `internal/indexer/resolve.go` carried a `gofmt -l` violation in
   `retryConformanceCalls` that was deferred at v0.1/05-12 as out-of-plan-scope,
   sat unfixed across milestones, and was eventually fixed by hand rather than by
   a gate (confirmed resolved 2026-08-10).
3. During the same session an orchestrator wrote `gofmt -l <file> && echo "clean"`
   as its own check — which is itself vacuous, since `gofmt -l` exits 0 whether or
   not it lists files. It printed the filename **and** "clean".

Beyond formatting, the repo currently has no idiomatic-Go linting at all
(`errcheck`, `ineffassign`, `staticcheck`, `unused`, `revive`, …). `go vet`
covers a deliberately narrow set of correctness checks and is not a substitute.

## Solution

Add `golangci-lint` and enable `gofmt`/`gofumpt` plus a sensible idiomatic set.

**Pin it in `go.tool-lint.mod`, not `go.tool.mod`.** This repo keeps two tool
modfiles deliberately, and `TestToolModfilesRemainIsolated` enforces the split:

- `go.tool.mod` → build/release tooling (`task`, `goreleaser`, `govulncheck`),
  invoked via `GO_TOOL` (`Taskfile.yml:9`)
- `go.tool-lint.mod` → linters. Currently pins only
  `github.com/rhysd/actionlint/cmd/actionlint` (go.tool-lint.mod:24), invoked via
  `GO_TOOL_LINT` (`Taskfile.yml:10`)

golangci-lint belongs alongside actionlint in the second one.

Sketch:

1. `tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint` in `go.tool-lint.mod`
2. A `lint:go` target mirroring `lint:actions` (Taskfile.yml:3163):
   `{{.GO_TOOL_LINT}} golangci-lint run ./...`
3. Add `lint:go` to the `lint` wrapper (Taskfile.yml:3521), and wire it into CI
   the same way `lint:actions` is
4. A `.golangci.yml` enabling at minimum `gofmt` (or `gofumpt`), plus the
   idiomatic set the team wants

**Two constraints worth respecting when implementing:**

- **Fix, don't suppress.** `task vet`'s own description records the precedent:
  "Findings surfaced on first run are fixed, not suppressed." Expect a first-run
  backlog; budget for it rather than blanket-`//nolint`-ing.
- **Make the gate assert positively.** Per repo rule `84d1gfpywd`, do not write
  `gofmt -l X && echo ok` — that passes vacuously. `golangci-lint run` exits
  non-zero on findings, so the exit code is the verdict; if any wrapper script
  captures output instead, test it for emptiness explicitly rather than relying
  on an exit status that is always 0.

Naming note: the target is `lint:actions`, never `lint:actionlint` — copy the
target name from Taskfile.yml, not the CI check's display name.

Open question for whoever picks this up: whether to enable `gofumpt` (stricter
superset) or plain `gofmt`, and how aggressive the idiomatic set should be.
