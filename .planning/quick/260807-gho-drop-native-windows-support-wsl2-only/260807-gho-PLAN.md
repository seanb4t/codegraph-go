---
quick_id: 260807-gho
slug: drop-native-windows-support-wsl2-only
created: 2026-08-07
mode: quick
---

# Drop native Windows support — WSL2 only

## Objective

Make "no native Windows support" an explicit, coherent project position:
Linux and macOS (amd64/arm64) are supported natively; Windows users run the
linux binary under WSL2. Nothing unsupported is built, shipped, or gated.

## Origin

`/gsd-audit-uat` surfaced Phase 07 (v1.0) Test 3 — "Windows daemon stop
termination + PPID watchdog on a real Windows host" — as the only
prerequisite-blocked item in the ledger, unverifiable since 2026-07-19 for
want of a Windows host. Rather than carry it indefinitely, the platform
itself is dropped and the test becomes moot.

## Decisions (user, 2026-08-07)

- **D-1 — Cut depth: full removal.** Not docs-only, not stop-shipping-but-
  keep-the-code. Nothing unsupported is built, shipped, or gated.
- **D-2 — Timing: immediate.** The next release simply has no windows
  assets; no deprecation cycle. v0.3.0 and earlier keep theirs (GitHub
  Releases are immutable).

## Boundary rule

Remove code that exists **to run codegraph on Windows**. Keep code that
handles **Windows-authored data** or **WSL2**, both of which serve supported
platforms:

| Keep | Why |
|------|-----|
| `internal/watch/policy.go` WSL2 `/mnt/<drive>` detection | This *is* the supported Windows path |
| `internal/migrate/` drive-letter path handling | Migrating a TS index authored on Windows, from WSL/Linux, is a supported flow |
| `internal/bench` `normalizeMaxrss("windows")` case | Asserts an *unsupported* OS errors — more correct after this change, not less |

## Tasks

### Task 1 — Remove the Windows build, gate, and runtime surface

- `.goreleaser.yaml` — drop `codegraph-windows-{amd64,arm64}` build entries
  and the `.exe` conditional in `archives.name_template`
- `.github/workflows/release.yml` — drop both windows matrix legs, the
  `$EXT` branch in the asset-rename step, and the `codegraph.exe` find
  fallback; narrow the zig step to linux/arm64
- `.github/workflows/ci.yml` + `Taskfile.yml` — drop `vet:windows`,
  `vet:daemon-windows`, and the mingw-w64 apt step
- Delete `internal/daemon/{stop,watchdog}_windows.go` and
  `internal/graphstore/locked_windows{,_test}.go`
- `internal/upgrade/swap.go` — drop `swapWindows()` (the rename-self-aside
  dance for a running `.exe`)
- `internal/upgrade/upgrade.go` — drop `releaseAssetName()`'s `.exe` suffix
- `.github/ISSUE_TEMPLATE/bug_report.yml` — platform placeholder → WSL2

**Gates that must move in lockstep** (each is a set-equality or
exhaustiveness assertion that fails on a stale entry — that is why they
exist):

- `TestReleaseAssetNameMatchesGoReleaser` — 6 pinned os/arch pairs → 4
- `TestCheckCrossMatchesGoreleaserTargets` — Taskfile `check:cross` pair
  list must stay set-equal to the goreleaser build targets
- `TestWorkflowRunStepsInvokeTaskTargets` — fails any `runBodyExceptions`
  entry matching no real step; the mingw-w64 exception goes stale
- `TestContributingReferencesRealTaskTargets` — CONTRIBUTING.md must not
  name `vet:daemon-windows`

verify: `go build ./...`, `go vet ./...`, `task test:daemon`,
`go test ./internal/upgrade/... ./internal/graphstore/... -count=1`,
`task check:goreleaser`, `task lint:actions`
done: all green; `GOOS=windows go build ./...` now fails (intended)

### Task 2 — Documentation

- `README.md` — Supported platforms table + WSL2 instructions, including
  the `/mnt/<drive>` watcher caveat verified against `policy.go`
- `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md` — six targets → four;
  asset shape loses `[.exe]`
- `CONTRIBUTING.md` — drop the mingw-w64 prerequisite, state the platform
  position for contributors

done: no doc claims a windows asset ships; historical statements about
past releases stay factually intact rather than being renumbered

### Task 3 — Close the originating UAT item

`.planning/milestones/v1.0-phases/07-.../07-UAT.md` Test 3 → recorded as
permanently closed.

**Tooling constraint:** the UAT format's `result:` vocabulary is exactly
`pass | skipped | blocked | pending` (`gsd-core/bin/lib/uat.cjs`
`categorizeItem`, and the collection filter that treats
`pending|skipped|blocked` as outstanding). There is no `waived`/`n-a`
value. Per the planning-artifacts rule, no new value is invented: the item
keeps `result: skipped` and the `reason:` records permanent closure. It
will keep appearing in `/gsd-audit-uat` as one explained item — a tooling
expressiveness gap to report upstream, not outstanding work.

done: reason states the code under test is deleted and the scenario cannot
occur; no invented frontmatter shapes

## Must not

- Invent a `result:` value the UAT format does not emit
- Write a reason engineered to trip `categorizeItem`'s `device|physical`
  keyword branch — that games the classifier rather than recording truth
- Touch `CHANGELOG.md` (release-please owns it) or edit historical
  statements about what past releases shipped
- Fix the pre-existing `gofmt` violation in
  `internal/query/files_status_test.go` — present on `origin/main`, out of
  scope
