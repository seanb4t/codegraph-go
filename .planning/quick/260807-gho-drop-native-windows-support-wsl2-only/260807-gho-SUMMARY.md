---
status: complete
quick_id: 260807-gho
slug: drop-native-windows-support-wsl2-only
date: 2026-08-07
commits: [085b7a3, dd474bb]
---

# Summary — Drop native Windows support (WSL2 only)

one_liner: "Native Windows is gone from the build, the release matrix, CI, and the runtime surface — Linux and macOS are the supported platforms and WSL2 is the Windows path — with four in-repo set-equality gates catching every piece that had to move in lockstep."

## What changed

**Removed** (code whose only purpose was running codegraph *on* Windows):

| Layer | Removed |
|-------|---------|
| Build | `.goreleaser.yaml`'s `codegraph-windows-{amd64,arm64}` zig-cross entries; the `.exe` conditional in `archives.name_template` |
| Release CI | `release.yml`'s two windows matrix legs, the `$EXT` branch, the `codegraph.exe` find fallback; zig step narrowed to linux/arm64 |
| CI gates | `ci.yml`'s `vet:windows` and `vet:daemon-windows` jobs + the mingw-w64 apt step; the two matching `Taskfile.yml` targets |
| Source | `internal/daemon/{stop,watchdog}_windows.go`; `internal/graphstore/locked_windows{,_test}.go` |
| Upgrade | `swapWindows()`'s rename-self-aside dance; `releaseAssetName()`'s `.exe` suffix |

**Kept**, per the plan's boundary rule — Windows-adjacent code serving
supported platforms: `internal/watch/policy.go`'s WSL2 `/mnt/<drive>`
detection (that *is* the supported Windows path), `internal/migrate`'s
drive-letter handling (migrating a Windows-authored TS index from
WSL/Linux), and `internal/bench`'s `normalizeMaxrss("windows")` case
(asserts an unsupported OS errors — more true now, not less).

## The gates did the work

Four in-repo assertions caught pieces that had to move together. Every one
is a **set-equality or exhaustiveness** check rather than an exit-status
check, which is exactly why a *removal* tripped them — an `if X then fail`
gate goes quiet when X disappears, but a set comparison notices:

1. `TestReleaseAssetNameMatchesGoReleaser` — pinned 6 os/arch pairs; left
   at 6 it would have kept asserting windows assets exist. → 4
2. `TestCheckCrossMatchesGoreleaserTargets` — holds Taskfile `check:cross`
   set-equal to the goreleaser build targets. Fired with a verbatim
   set-difference message.
3. `TestWorkflowRunStepsInvokeTaskTargets` — fails any `runBodyExceptions`
   entry matching no real step. The mingw-w64 exception went stale the
   moment the apt step was deleted. This is the subtle one: allowlists
   normally rot silently, and asserting the *reverse* direction turns a
   stale exemption into a red build.
4. `TestContributingReferencesRealTaskTargets` — caught CONTRIBUTING.md
   still naming `vet:daemon-windows`.

## Verification

| Check | Result |
|-------|--------|
| `go build ./...` | pass |
| `go vet ./...` | pass |
| `go test ./internal/upgrade/... -count=1` | pass |
| `go test ./internal/graphstore/... -count=1` | pass |
| `task test:daemon` (the isolated gate CI runs) | pass, 64.3s |
| `task test:golden` | pass, 13.7s |
| `task test:wireoracle` | pass, 19.1s |
| `task check:goreleaser` | pass — 1 config validated |
| `task lint:actions` (actionlint) | pass |
| `GOOS=windows go build ./...` | fails — **intended outcome**, not a regression |

## Known unknowns / not fixed

- **`TestRunWatchdogCancelsRunOnSimulatedReparent` failed once** during an
  ad-hoc full-parallel `go test ./... -count=1` (250s, a timeout), and
  passed 5/5 in isolation at ~1.04s each. Not caused by this change: the
  deleted `watchdog_windows.go` carries `//go:build windows` and is
  excluded from a darwin build by its own tag, so removing it cannot alter
  what compiles or runs here. This is the repo's documented
  full-suite-contention flake class (CI isolates `internal/daemon` into its
  own `-count=1` step for exactly this reason; v0.3.0 carries the daemon
  extreme-load timeout tail as an accepted limitation under MAINT-02). The
  gate CI actually runs — `task test:daemon` — is green.
- **Pre-existing `gofmt` violation** in
  `internal/query/files_status_test.go`, confirmed present on `origin/main`
  by checking out that ref and re-running `gofmt -l`. Untouched, out of
  scope.
- **`GO-2026-5932`** and the daemon timeout tail carry forward from v0.3.0
  unchanged — unrelated to this task.

## Deferred / upstream gap

The UAT format cannot express "waived — no longer applicable." Its
`result:` vocabulary is exactly `pass | skipped | blocked | pending`
(`gsd-core/bin/lib/uat.cjs`: `categorizeItem`, plus the collection filter
treating `pending|skipped|blocked` as outstanding). Per the
planning-artifacts rule no value was invented, so Phase 07 Test 3 keeps
`result: skipped` with a `reason:` recording permanent closure, and will
keep surfacing in `/gsd-audit-uat` as one explained item.

A reason worded to trip `categorizeItem`'s `device|physical` branch would
have reclassified it as `device_needed` and hidden it — deliberately not
done, since that games the classifier instead of recording the truth.

**Recommend reporting upstream to GSD:** a terminal `result:` value for
items closed as no-longer-applicable (distinct from `pass`, which asserts
something was tested).

## Follow-ups for the human

- Open the PR from `chore/drop-windows-support`. The repo gates PR titles
  (`pr-title.yml`) and requires an issue link (`require-issue-link.yml`),
  so an issue may need to exist first.
- The lead commit is `feat(platform)!` with a `BREAKING CHANGE:` footer —
  release-please will read that when picking the next version. Dropping
  published release assets is a deliberate breaking change.
