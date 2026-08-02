# Phase 10: Local Build Tooling & CONTRIBUTING - Pattern Map

**Mapped:** 2026-08-01
**Files analyzed:** 10 (5 new, 5 modified)
**Analogs found:** 5 external / 5 in-repo-is-its-own-analog

This is an infrastructure/build-tooling phase. The repo deliberately has no
`Makefile`/`Taskfile`/`scripts/build` today (`scripts/` holds only
`pr_template_policy.py`), so the five **new** files have no in-repo analog —
their pattern source is the external sibling repo `github.com/holomush/holomush`
(same maintainer, same pattern in production; already fetched live during
RESEARCH.md). The five **modified** workflow files ARE their own analogs: the
planner's job is to preserve their job names/step names/comments and swap only
`run:` bodies for `task <target>` calls.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `Taskfile.yml` (new, repo root) | config/build-orchestration | batch | `github.com/holomush/holomush` `Taskfile.yaml` (EXTERNAL) | pattern-match, scope differs (holomush wraps ~1500 lines of unrelated web/proto/docker tasks — do not port those) |
| `go.tool.mod` + `go.tool.sum` (new) | config/tool-bootstrap | batch | `github.com/holomush/holomush` `go.tool.mod` (EXTERNAL) | exact pattern match (isolated tool modfile + header comment explaining the split) |
| `go.tool-lint.mod` + `go.tool-lint.sum` (new) | config/tool-bootstrap | batch | `github.com/holomush/holomush` `go.tool.mod` (EXTERNAL, same shape, second file) | exact pattern match — MVS-conflict rationale must be in the header comment |
| `.github/actions/install-task/action.yml` (new) | CI composite action | request-response (build-and-PATH) | `github.com/holomush/holomush` `.github/actions/install-task/action.yml` (EXTERNAL) | exact pattern match |
| `.github/workflows/ci.yml` (modified) | CI workflow | event-driven | itself (self-analog) — 5 jobs, `test` job has 8 steps | self — preserve `name:` fields and comments verbatim, only `run:` bodies + runner/cache steps change |
| `.github/workflows/bench.yml` (modified) | CI workflow | event-driven | itself (self-analog) | self — `rebless` job moves to Namespace FIRST per D-09 ordering; no local task target exists for `-rebless` (D-13) |
| `.github/workflows/release-please.yml` (modified) | CI workflow | event-driven | itself (self-analog); sweep body also matches `Taskfile.yml`'s new `check:cross` target 1:1 | self — `pretag-gate` job's inline sweep becomes `task check:cross`; needs `install-task` wired in even though NOT in D-06's Namespace-runner scope |
| `.github/workflows/release.yml` (modified) | CI workflow (release pipeline) | event-driven | itself (self-analog) | self — runners move to Namespace except `provenance` (D-07 hard exception); `macos-latest` darwin leg stays native (D-08) |
| `CONTRIBUTING.md` (modified, pointer only) | doc | — | itself, `## Building` section (lines 57-75) | self — append pointer paragraph only, do not rewrite (D-00) |
| `.planning/REQUIREMENTS.md` (modified) | doc | — | existing requirement entries in that file | self — add `DEV-01` per D-16, pending maintainer confirmation |

## Pattern Assignments

### `Taskfile.yml` (new, config/build-orchestration)

**Analog (EXTERNAL):** `github.com/holomush/holomush` `Taskfile.yaml`

**GO_TOOL variable indirection pattern:**
```yaml
vars:
  GO_TOOL: GOWORK=off go tool -modfile=go.tool.mod
  GO_TOOL_LINT: GOWORK=off go tool -modfile=go.tool-lint.mod

tasks:
  lint:actions:
    desc: Lint GitHub Actions workflows
    cmds:
      - "{{.GO_TOOL_LINT}} actionlint .github/workflows/*"

  check:goreleaser:
    desc: Validate goreleaser config
    cmds:
      - "{{.GO_TOOL}} goreleaser check"
```

**Cross-toolchain gating pattern — `preconditions:` with actionable `msg:` (D-11), NEVER `status:` or `platforms:`:**
```yaml
tasks:
  vet:windows:
    desc: "Typecheck windows PPID watchdog (GOOS=windows go vet internal/daemon) — needs mingw-w64"
    preconditions:
      - sh: command -v x86_64-w64-mingw32-gcc
        msg: "mingw-w64 not found. Install with: apt-get install gcc-mingw-w64-x86-64 (or your platform's equivalent). See CONTRIBUTING.md#building."
    env:
      CGO_ENABLED: "1"
      CC: x86_64-w64-mingw32-gcc
    cmds:
      - GOOS=windows GOARCH=amd64 go vet ./internal/daemon/

  build:cross:
    desc: "Cross-compile to a non-host GOOS/GOARCH via zig — needs zig on PATH"
    preconditions:
      - sh: command -v zig
        msg: "zig not found. Install per CONTRIBUTING.md's cross-build prerequisite section."
    cmds:
      - "{{.GO_TOOL}} goreleaser build --single-target --clean"
```

**Anti-pattern (confirmed by Context7 docs, cited in RESEARCH.md):** `platforms:` silently
*skips* a task on a non-matching host instead of failing — the same GOLDEN-01
failure class `status:` was already rejected for by D-11. Do not use it for
`vet:windows`/`build:cross`-style targets; use `preconditions:` only.

**Fine-grained targets CI calls directly must map 1:1 onto ci.yml's named steps
(D-02) — build the target names from the exact `run:` bodies extracted below,
not a redesign.** Suggested namespacing (Claude's discretion per CONTEXT.md):
`test:unit`, `test:golden`, `test:integration`, `test:daemon`, `test:race`,
`vet`, `vet:windows`, `vet:daemon-windows`, `lint:actions`,
`check:goreleaser`, `check:cross`, `bench:headtohead`, `bench:regression`.
Coarse wrapper `task test` = `deps: [test:unit, test:golden, test:integration, test:daemon, test:race]` (D-10, host-only legs).

**No local `-rebless` target** (D-13) — `bench.yml`'s rebless job stays the sole
invocation site; do not add a `task bench:rebless`.

**`task bench:regression` must surface the platform guard's refusal legibly (D-14)** — wrap `tools/bench/runner`'s exit, don't let a bare non-zero exit be the only signal on macOS dev machines.

---

### `go.tool.mod` / `go.tool.sum` and `go.tool-lint.mod` / `go.tool-lint.sum` (new, config/tool-bootstrap)

**Analog (EXTERNAL):** `github.com/holomush/holomush` `go.tool.mod` header comment

**Why-the-split header comment pattern (MUST be present verbatim-in-spirit, not
omitted — CONTEXT.md specifics section: "Without it the two files look like an
accident and someone will merge them"):**
```
// This module is isolated from the repo's main go.mod (and from
// go.tool-lint.mod) because a single shared tool module does NOT compile:
// MVS resolves one version per module across the whole graph, and
// actionlint's expected YAML API loses the version bid to whatever
// task/goreleaser drag in transitively (verified:
// action_metadata.go:273:22: te.Errors[0].Error undefined).
//   go.tool.mod       -> task, goreleaser   (237 net-new modules if merged into root go.mod)
//   go.tool-lint.mod  -> actionlint          (16 modules alone; incompatible if co-located above)
// NOT covered by Renovate/Dependabot (both gomod managers target go.mod only) —
// version bumps here are manual. See CONTEXT.md D-03/D-05.
```

**Installation commands (from RESEARCH.md, verified live against this repo's Go 1.26.5 toolchain):**
```bash
GOWORK=off go get -tool -modfile=go.tool.mod github.com/go-task/task/v3/cmd/task@v3.52.0
GOWORK=off go get -tool -modfile=go.tool.mod github.com/goreleaser/goreleaser/v2@v2.17.1
GOWORK=off go get -tool -modfile=go.tool-lint.mod github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
```
`go get -tool` auto-populates the `require` block and the matching `.sum` file
(`go.tool.sum`, NOT `go.tool.mod.sum` — matches Go's own `-modfile` convention).

**goreleaser version reconciliation (flag for planner, do not silently pick):**
`ci.yml`'s current `goreleaser-check` job pins `v2.17.1`
(`.github/workflows/ci.yml:341`); `release.yml`'s build matrix pins
`GORELEASER_VERSION: v2.17.0` (`.github/workflows/release.yml:51`). Pre-existing
mismatch, not introduced by this phase. RESEARCH.md recommends pinning
`go.tool.mod` to `v2.17.1` (matching the job this phase replaces) and filing the
`v2.17.0` discrepancy as a separate follow-up — do not silently "fix" a
release-pipeline version choice that was Phase-9-audited.

---

### `.github/actions/install-task/action.yml` (new, CI composite action)

**Analog (EXTERNAL):** `github.com/holomush/holomush` `.github/actions/install-task/action.yml`

```yaml
# Source: github.com/holomush/holomush .github/actions/install-task/action.yml
# (public repo, fetched live 2026-08-01)
name: Install Task
description: Build the pinned task binary from go.tool.mod and add it to PATH.
runs:
  using: composite
  steps:
    - name: Build Task from go.tool.mod
      shell: bash
      run: |
        set -euo pipefail
        mkdir -p "$HOME/.local/bin"
        # GOWORK=off: go.work enables workspace mode, which is incompatible
        # with -modfile.
        GOWORK=off go build -modfile=go.tool.mod \
          -o "$HOME/.local/bin/task" \
          github.com/go-task/task/v3/cmd/task
        echo "$HOME/.local/bin" >> "$GITHUB_PATH"
    - name: Verify Task
      shell: bash
      run: task --version
```

**CDN-flake rationale must be an in-file comment** (D-04 evidence: `arduino/setup-task`
and `taskfile.dev/install.sh` both broke on CDN flake in holomush production,
incidents `holomush-06vz`/`holomush-ar31b`) — otherwise a future contributor
"simplifies" this back to a marketplace action.

**Consumers:** every job that later calls `task <target>` needs this action as
its first tool-dependent step: `ci.yml`'s five jobs, `bench.yml`'s two jobs,
`release-please.yml`'s `pretag-gate`, `release.yml`'s `build`/`assemble` jobs.

---

### `.github/workflows/ci.yml` (modified — self-analog)

**Six required-status-check job `name:` fields — MUST stay byte-identical
(ruleset `20157557`; GitHub matches on rendered job name, not workflow key or
step name):**
```
test
govulncheck (DIST-03, blocking)
reproducibility (double-build hash-diff, DIST-04)
perf regression gate (PERF-02, INDX-06)
actionlint (workflow static analysis)
goreleaser check (config validation, DIST-01)
```
(Source: `.github/workflows/ci.yml` lines 47, 149, 166, 254, 307, 326.)

**`test` job's 8 steps (lines 46-146) — every `name:`, comment block, and
`run:` body must be preserved; only the body becomes `task <target>`:**

| Step name (line) | Current `run:` body | Suggested target |
|---|---|---|
| Build (60) | `go build ./...` | `task build` |
| Test (excluding internal/daemon) (69-73) | materialized `go list` + filtered `go test` — see exact script below | `task test:unit` |
| Test golden parity suite (83) | `go test ./testdata/golden/...` | `task test:golden` |
| Test subprocess integration harness (91) | `go test ./test/integration/...` | `task test:integration` |
| Test internal/daemon (isolated) (100) | `go test ./internal/daemon/ -count=1` | `task test:daemon` |
| Test concurrency packages under -race (114) | `go test -race -count=1 -p 1 ./internal/daemon/... ./internal/watch/... ./internal/cli/...` | `task test:race` |
| Typecheck windows lock classifier (124) | `GOOS=windows GOARCH=amd64 go vet ./internal/graphstore/` | `task vet:windows` |
| Install mingw-w64 (140) | `sudo apt-get update && sudo apt-get install -y gcc-mingw-w64-x86-64` | keep as its own CI-only step (not a local task — installs a system package) |
| Typecheck windows PPID watchdog (146) | `GOOS=windows GOARCH=amd64 go vet ./internal/daemon/` (env `CGO_ENABLED=1`, `CC=x86_64-w64-mingw32-gcc`) | `task vet:daemon-windows` |

Exact "Test (excluding internal/daemon)" body (lines 69-73, the GOLDEN-01/IN-08
materialize-before-set-e-check pattern — port verbatim into the task body, do
not "clean up"):
```bash
set -euo pipefail
pkgs_raw=$(go list ./...)
mapfile -t pkgs < <(printf '%s\n' "$pkgs_raw" | grep -v '/internal/daemon$')
go test "${pkgs[@]}"
```

**`govulncheck` job (148-163):** uses `golang/govulncheck-action`, not a
runnable command — RESEARCH.md confirms **no task-target equivalent exists**;
deliberately left un-task-ified (out of scope per Deferred Ideas).

**`reproducibility` job (165-251):** two long inline scripts (linux/amd64
blocking double-build, lines 192-218; linux/arm64 reported-only, lines 220-251).
Both are candidates for `task check:reproducibility` / `task check:reproducibility:arm64`
— port verbatim, these are security-sensitive byte-for-byte scripts (SOURCE_DATE_EPOCH
pinning, sha256 comparison) that must not be "improved" during the move.

**`perf-regression` job (253-295):**
```bash
go run ./tools/bench/runner -mode regression \
  -baseline tools/bench/baseline.json \
  -trials 3 \
  -ceiling-bytes 4294967296
```
→ `task bench:regression`. Note: `tools/bench/runner` is invoked from the repo
root as a normal Go package under the main module (confirmed: NOT under a
nested `go.mod` — the landmine CONTEXT.md/RESEARCH.md flag would silently
excise it from `go list ./...`; verified no `tools/go.mod` exists).

**`actionlint` job (306-323):**
```bash
go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
"$(go env GOPATH)/bin/actionlint" .github/workflows/*.yml
```
→ `{{.GO_TOOL_LINT}} actionlint .github/workflows/*.yml` (`task lint:actions`) — the
`go install @pin` mechanism this step used is exactly what D-03's isolated
modfile replaces.

**`goreleaser-check` job (325-341):**
```bash
go install github.com/goreleaser/goreleaser/v2@v2.17.1
"$(go env GOPATH)/bin/goreleaser" check
```
→ `{{.GO_TOOL}} goreleaser check` (`task check:goreleaser`).

**Runner/cache migration (D-06), applied per job except `govulncheck`'s action
step is unaffected:**
```yaml
steps:
  - uses: actions/checkout@<sha> # vX
  - name: Set up Go
    uses: actions/setup-go@<sha> # vX
    with:
      go-version-file: go.mod
      cache: false
  - name: Cache Go modules and build
    uses: namespacelabs/nscloud-cache-action@c5f8dab7560444c4bf8dbc64f1b203431873c547 # v1.6.1
    with:
      cache: go
  - name: Install Task
    uses: ./.github/actions/install-task
```
`runs-on: namespace-profile-linux-amd64-{2x4,4x8}` — sizing is Claude's
discretion (holomush: `2x4` for lint/fast jobs, `4x8` for test jobs).

---

### `.github/workflows/bench.yml` (modified — self-analog)

**Two jobs, `name:` fields to preserve:**
```
head-to-head publish (PERF-01, non-blocking)
record candidate perf baseline (PERF-02, manual only)
```

**`rebless` job's `go run` invocations (lines 101, 158, 229) — 3 total across
the file, all from repo root:**
```bash
# headtohead (line 101)
go run ./tools/bench/runner -mode headtohead \
  -go-binary "$(pwd)/codegraph-bench" \
  -ts-binary "$(command -v codegraph)"

# rebless record (line 158)
go run ./tools/bench/runner -mode regression -rebless \
  -baseline tools/bench/baseline.json \
  -seed 42 -count 120000 \
  -trials "$TRIALS"

# rebless verify (line 229)
go run ./tools/bench/runner -mode regression \
  -baseline tools/bench/baseline.json \
  -seed 42 -count 120000 \
  -trials "$TRIALS" \
  -ceiling-bytes 4294967296
```
**D-13 constraint: none of these three become a `task <target>` reachable
locally** — `-rebless` (the second) is the dangerous one; wrapping even the
other two in a shared task risks a contributor discovering the rebless flag by
adjacency. RESEARCH.md's Pattern/Anti-Pattern guidance: these stay CI-only
invocations, described in comments, not task-ified.

**D-09 ordering constraint (sequencing, not code):** move `rebless` job to
Namespace and commit a new Namespace-recorded `baseline.json` FIRST; only then
move `ci.yml`'s `perf-regression` job to Namespace. `CheckRegression`
(`tools/bench/runner/main.go`) compares only `{goos, goarch}` — structurally
blind to a runner-class-only change — so the ordering is the only safeguard.

---

### `.github/workflows/release-please.yml` (modified — self-analog, D-15)

**Job name to preserve:** `pre-tag 6-target go list sanity sweep` (line 35).

**Inline sweep being replaced (lines 46-54), verbatim body for `task check:cross`:**
```bash
set -euo pipefail
for pair in linux/amd64 linux/arm64 windows/amd64 windows/arm64 darwin/amd64 darwin/arm64; do
  GOOS="${pair%/*}" GOARCH="${pair#*/}" go list -mod=readonly ./... > /dev/null || {
    echo "::error::go list -mod=readonly ./... failed for GOOS=${pair%/*} GOARCH=${pair#*/}"
    exit 1
  }
done
```
Port this loop body verbatim into `Taskfile.yml`'s `check:cross` target — do
not reimplement "cleaner" inside Taskfile syntax (Don't-Hand-Roll table,
RESEARCH.md: risk of silently changing the target list).

**Bootstrap dependency this job newly acquires:** `pretag-gate` needs
`install-task` wired in as a step even though **this workflow is NOT in D-06's
Namespace-runner migration scope** — it stays on `ubuntu-latest`. The planner
must ensure the tool build (go.tool.mod → `task`) is present in this job before
`task check:cross` can run; no `nscloud-cache-action` needed here, but the
`install-task` composite step is still required.

---

### `.github/workflows/release.yml` (modified — self-analog, D-06/D-07/D-08)

**Structural constants that MUST NOT change (from the file's own locked-contract header, lines 1-36):**
- Workflow filename/path, `on: push: tags: v[0-9]*` trigger — anchors
  `internal/upgrade/verify.go`'s `releaseWorkflowRefPattern`.
- `provenance` job (line 332-358) — HARD EXCEPTION, cannot move runners (D-07):
  it's a reusable workflow (`slsa-framework/slsa-github-generator/.github/workflows/generator_generic_slsa3.yml@v2.1.0`)
  that declares its own `runs-on`; a caller cannot override it.

**`build` job matrix (lines 61-100) — D-08 constraint:** the `macos-latest` leg
(darwin/amd64 + darwin/arm64, native Xcode clang, `needs_zig: false`) must stay
a **real macOS runner** if migrated to Namespace — never substituted with a
zig cross-build. The other 4 legs (`ubuntu-latest`: linux/amd64 native,
linux/arm64 + windows/amd64 + windows/arm64 via zig) are Namespace-eligible.

**`assemble` job (203-323) and its cosign/syft/gh-release steps** are unaffected
by the task-ification (they're already precise, security-audited inline
scripts per the file's own header) — only their runner class moves per D-06,
not their `run:` bodies.

**goreleaser version in this file:** `env.GORELEASER_VERSION: "v2.17.0"` (line
51) — see the reconciliation note under `go.tool.mod` above.

---

### `CONTRIBUTING.md` (modified — pointer only, D-00)

**Analog:** itself, `## Building` section (lines 57-75) — **already correct
content**, do NOT rewrite. Append a pointer paragraph directly after line 73
("Workflow changes should pass `actionlint` locally before you push.") pointing
at the new `task` targets, e.g. referencing `task test`, `task lint`, and the
cross-toolchain `preconditions:` targets. Keep `## Never do these` (lines
112-128) unchanged — its `-rebless` prohibition (lines 119-123) is exactly what
D-13 preserves at the tooling layer.

---

## Shared Patterns

### GO_TOOL / GO_TOOL_LINT variable indirection
**Source (EXTERNAL):** `github.com/holomush/holomush` `Taskfile.yaml`
**Apply to:** every task in `Taskfile.yml` that invokes `task`, `goreleaser`, or `actionlint`.
```yaml
vars:
  GO_TOOL: GOWORK=off go tool -modfile=go.tool.mod
  GO_TOOL_LINT: GOWORK=off go tool -modfile=go.tool-lint.mod
```

### `preconditions:` with actionable `msg:` for cross-toolchain gating
**Source (EXTERNAL):** taskfile.dev docs + holomush live pattern
**Apply to:** `vet:windows`, `vet:daemon-windows`, `build:cross` — any target
D-10 does NOT include in the host-only `task test` wrapper.
**Never use `status:` or `platforms:`** for this — both silently skip (GOLDEN-01
failure class), confirmed for `platforms:` specifically as a new RESEARCH.md finding.

### Required-status-check job name preservation
**Source:** `.github/workflows/ci.yml` (self) + GitHub ruleset `20157557` (confirmed live via `gh api`)
**Apply to:** all six `ci.yml` job `name:` fields listed above — GitHub matches
on rendered job name, not workflow key or step name. This is the single highest-risk
regression in this phase; every plan touching `ci.yml` must diff-check job names.

### SHA-pin every third-party Action, resolved tag in trailing comment
**Source:** repo-wide convention, e.g. `.github/workflows/ci.yml:51` (`actions/checkout@df4cb...# v6.0.3`)
**Apply to:** `namespacelabs/nscloud-cache-action` and any new Namespace-related action.
Use the fully-qualified resolved tag (`# v1.6.1`), NOT holomush's abbreviated `# v1` comment style.

### `env:` indirection instead of direct `${{ }}` interpolation in `run:`
**Source:** `.github/workflows/release.yml:141-144` (`TAG`, `GOOS`, `GOARCH` passed via `env:`)
**Apply to:** any new/modified `run:` block touching workflow context values —
never interpolate `${{ github.* }}` directly into shell.

## No Analog Found (in-repo)

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `Taskfile.yml` | config | batch | repo has no Makefile/Taskfile/justfile today — pattern sourced externally from `github.com/holomush/holomush` |
| `go.tool.mod` / `go.tool-lint.mod` | config | batch | no isolated tool-modfile precedent in this repo — external analog only |
| `.github/actions/install-task/action.yml` | CI composite action | request-response | repo has zero composite actions today (`ls .github/actions/` is empty) — external analog only |

## Metadata

**Analog search scope:** `.github/workflows/*.yml`, `CONTRIBUTING.md`, `go.mod`,
`tools/bench/runner/main.go`, `.planning/phases/10-*/10-{CONTEXT,RESEARCH}.md`;
external reference `github.com/holomush/holomush` (already fetched live during
RESEARCH.md, cited here rather than re-fetched).
**Files scanned:** 10 in-repo (all workflow files + CONTRIBUTING.md + go.mod +
tools/bench/runner/main.go), plus RESEARCH.md's already-captured excerpts of 3
external holomush files.
**Pattern extraction date:** 2026-08-01
