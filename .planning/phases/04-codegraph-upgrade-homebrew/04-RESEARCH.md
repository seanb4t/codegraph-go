# Phase 4: `codegraph upgrade` × Homebrew - Research

**Researched:** 2026-08-10
**Domain:** Go CLI self-upgrade detection of a Homebrew-managed install (macOS + Linux), Homebrew Cask internals
**Confidence:** MEDIUM — mechanism (Go stdlib symlink resolution, `internal/upgrade` seams) is HIGH; the two research targets CONTEXT.md flagged are answered, but one answer (D-04) **falsifies the locked decision it was meant to confirm** and needs a maintainer call before planning proceeds on the current wording

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Detection: what the criteria should say (UPGR-02)**

- **D-01:** ROADMAP criteria 1–3 and REQUIREMENTS UPGR-02 are amended from `Cellar` to `Caskroom`, and the detector recognizes **both** trees. Phase 3 shipped a `homebrew_casks:` block; a cask stages into `$HOMEBREW_PREFIX/Caskroom/<token>/<version>/` with `$HOMEBREW_PREFIX/bin/<name>` symlinked into it — `Cellar/` is the *formula* tree and has never existed for this project. Cellar/formula detection is kept even though this project ships no formula, so a future formula path or a hand-built brew install is not silently invisible.
- **D-02:** Detection is structural only, and the Phase-3 sentinel is REMOVED rather than left as dead weight. Removal touches four places: `.goreleaser.yaml:570-588` (post.install sentinel write), `.goreleaser.yaml:623-624` (post.uninstall `FileUtils.rm_f`), `Taskfile.yml:1996-2028`-ish (`release:rehearse-cask` Step 5b assertions), and any shape-test assertion pinning those blocks. BREW-05's install gate (man-page + version assertions) is untouched.
- **D-03:** Detection = **path-shape match AND a Homebrew-authored install receipt.** Resolve `os.Executable()` through symlinks, require a `Caskroom/<token>/<version>/` or `Cellar/<name>/<version>/` segment shape under *any* prefix, **and** require the Homebrew-written `INSTALL_RECEIPT.json` at the matching ancestor (cask: `<prefix>/Caskroom/<token>/.metadata/INSTALL_RECEIPT.json`; formula: `<prefix>/Cellar/<name>/<version>/INSTALL_RECEIPT.json`). Existence is sufficient; parsing it is optional.
- **D-04:** Linuxbrew stays in criterion 3, scoped to the Cellar shape only, with an inline note recording why. **Research must confirm before this is locked** that Homebrew on Linux genuinely has no cask support — asserted from general knowledge, not measured, and this milestone has falsified five scoping assumptions taken exactly this way.

**Refusal semantics (UPGR-01)**

- **D-05:** Bare `codegraph upgrade` under a brew-managed install returns an error and exits non-zero.
- **D-06:** `--force` is powerless against the refusal. No escape hatch — `brew uninstall --cask codegraph` is the honest path.
- **D-07:** The refusal message carries the symlink-resolved install path plus the exact command, e.g. `codegraph is managed by Homebrew (/opt/homebrew/Caskroom/codegraph/0.8.0). Upgrade with: brew upgrade codegraph`.
- **D-08:** Criterion 1's proof is amended from a sha256/mtime comparison to a seam-based assertion: the refusal fires, names `brew upgrade codegraph`, and neither `Options.download` nor `Options.swap` is ever invoked, driven through the injectable func-vars at `upgrade.go:38-43`. The real-tap acceptance run only has to observe the refusal.

**Read-only path (UPGR-03)**

- **D-09:** `--check` under a brew-managed install does not check — it steps aside with the same pointer rather than resolving a version. This is a criterion amendment, not a requirement change (UPGR-03 already says "reports how to upgrade", never "reports the available version").
- **D-10:** `--check` exits zero. Bare `upgrade` stays non-zero. Both exit behaviors must be documented in `--help` and `docs/`.

**Placement and ordering**

- **D-11:** Detection fires first in `Run()`, before `resolveLatest` — zero network, works offline, keeps criterion 4 trivially satisfied.
- **D-12:** Detection and the refusal live in `internal/upgrade`, as a new `brew.go` / `brew_test.go` pair.

**Standing constraint on all folded work**

- **D-13:** Only test, fix, or address things we own. Before proposing any verification: who owns the code this exercises; could we fix it or only notice it; and would the next natural run notice it anyway.

### Claude's Discretion

- The exact construction of the test fixtures — `t.TempDir()` layout, receipt file contents, how the four prefixes (`/opt/homebrew`, `/usr/local`, custom, linuxbrew) are simulated. Constraint: the false-positive case must be an **executing test**, not a comment.
- The exported/unexported shape of the detection API in `brew.go` and whether it returns a struct (path, token, version) or a bool plus path.
- Exact wording of the `--check` step-aside line, subject to D-07's resolved-path rule and UPGR-03's "reports how to upgrade".
- Whether the receipt check reads any field out of `INSTALL_RECEIPT.json` or only asserts its existence.
- How `docs/` and `--help` record the two exit behaviors (D-10).

### Deferred Ideas (OUT OF SCOPE)

- A `codegraph doctor` / install-provenance command.
- Detecting other managed install channels (a future formula, a distro package, `go install`).
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| UPGR-01 | `codegraph upgrade` detects a Homebrew-managed install and refuses, pointing at `brew upgrade codegraph`, and never modifies the Cellar | `checkWritable`'s early-refusal precedent (`upgrade.go:119`) and the func-var seams (`upgrade.go:38-43`) give the exact mechanism for D-05/D-08; see Architecture Patterns and Code Examples |
| UPGR-02 | Brew detection resolves symlinks to the real install path rather than matching a hardcoded prefix, so it is correct on Apple Silicon `/opt/homebrew`, Intel `/usr/local`, a custom prefix, and linuxbrew | `os.Executable()`'s documented symlink caveat (confirmed against pkg.go.dev this session) plus D-03's receipt-existence check answer UPGR-02's mechanism; **D-04's linuxbrew-cask premise is falsified by this session's research — see the Linuxbrew Cask Finding below, this changes what UPGR-02's linuxbrew test must cover** |
| UPGR-03 | `codegraph upgrade --check` still works under a brew-managed install — read-only, no mutation — and reports how to upgrade | `Options.Check`'s existing early-return branch (`upgrade.go:98-105`) is already structurally read-only; D-11 only requires moving brew detection above it |
</phase_requirements>

## Summary

The mechanism this phase needs is small and already has a precedent inside `internal/upgrade`: `checkWritable` (`swap.go:18-28`) is a working example of an early, error-returning refusal called from the head of `Run()`, and the four func-var seams (`resolveLatest`/`download`/`verify`/`swap`, `upgrade.go:38-43`) are exactly what D-08's seam-based proof needs — no new test infrastructure, one fake that records whether `download`/`swap` were called. The new code is one file (`internal/upgrade/brew.go`): resolve the target path through `filepath.EvalSymlinks` (Go's own docs recommend this over trusting `os.Executable()`'s raw return — confirmed against pkg.go.dev this session, HIGH confidence), pattern-match a `Caskroom/<token>/<version>` or `Cellar/<name>/<version>` path-segment shape, and `os.Stat` the Homebrew-authored `INSTALL_RECEIPT.json` at the corresponding ancestor (`<Caskroom>/<token>/.metadata/INSTALL_RECEIPT.json` for casks, `<Cellar>/<name>/<version>/INSTALL_RECEIPT.json` for formulae — both confirmed against Homebrew's own source this session, see below).

**The one finding that changes scope:** D-04 asked this phase to confirm "Homebrew on Linux does not support casks" before locking the linuxbrew test to the Cellar shape only. **That premise is false for the current Homebrew release line.** Homebrew merged Linux support for `binary`/`zap`-only casks in PR #19121 (merged 2025) and Homebrew 6.0.0 (June 2026, the same major version this project's own dev machine and Phase 3's rehearsal both ran) shipped further Linux-cask work: explicit Linux cask requirements (PR #21909), Caskroom using the user's primary group on Linux (PR #22202), and emitting Linux checksum variations for casks (PR #22632). codegraph's cask (`.goreleaser.yaml:495-523`) is a `binary`-only cask with no `app`/`pkg`/`zap` stanzas — precisely the shape PR #19121 made Linux-installable. **This is the sixth falsified scoping assumption in this milestone, same class as the prior five** — a maintainer decision is needed on whether criterion 3's linuxbrew leg must now cover the Caskroom shape too, not merely Cellar. See "Linuxbrew Cask Finding" below and the Assumptions Log.

D-03's receipt claim is independently confirmed at the source level: `AbstractTab::FILENAME = "INSTALL_RECEIPT.json"` (`Library/Homebrew/tab.rb`) is shared by both `Tab` (formula) and `Cask::Tab`; `Cask::Tab`'s `tabfile` is built as `cask.metadata_main_container_path/FILENAME`, and `metadata_main_container_path` is `caskroom_path.join(".metadata")` (`Library/Homebrew/cask/metadata.rb`, `METADATA_SUBDIR = ".metadata"`). One caveat worth carrying into planning: cask install receipts are a **2024 addition** (PR #17554, merged 2024-07-13) — a cask installed by a pre-4.3.7-era Homebrew would lack the file. Given D-03's detector requires shape-AND-receipt (not shape-OR-receipt), the only failure mode is a false negative on an ancient Homebrew (upgrades normally instead of refusing) — never a false positive, so this doesn't threaten criterion 3's false-positive leg, only widens the (already-acceptable, per D-11's ownership-test framing) surface of "very old Homebrew installs aren't detected."

**Primary recommendation:** Implement `brew.go` as path-shape-match AND `os.Stat`-only receipt check (no JSON parsing), called at the top of `Run()` before `resolveLatest`, using `filepath.EvalSymlinks` on the caller-supplied `targetPath`. Take D-04's linuxbrew finding back to the maintainer before the planner locks criterion 3's linuxbrew test scope.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Brew-managed path/receipt detection | `internal/upgrade` (business-logic package) | OS/filesystem (via `os`, `path/filepath`) | D-12 places this in `internal/upgrade`, the package that already owns install-shape knowledge (`checkWritable`, `atomicSwap`, `releaseAssetName`); it is a pure filesystem predicate, no network |
| Refusal branch + exit-code semantics | `internal/upgrade.Run()` | `internal/cli/upgrade.go` (renders the error to stderr via cobra's standard error path) | `Run()` is the single orchestration point every seam-based test already exercises (D-08); the CLI layer stays a thin wrapper that resolves `os.Executable()` and delegates, unchanged in shape from today |
| `--check` step-aside reporting | `internal/upgrade.Run()` | — | Same function, same early branch, per D-11 — no new component |
| Exit-code documentation (D-10) | `docs/` + `--help` (cobra `Long`/`Example` strings) | — | Static text, no runtime component |
| Sentinel removal | Release pipeline (`.goreleaser.yaml` cask hooks) + CI verification (`Taskfile.yml` `release:rehearse-cask`) | — | Entirely outside `internal/upgrade` — this is undoing Phase-3 release-time instrumentation, not application logic |
| Homebrew's own version-awareness (`brew outdated`) | External (Homebrew, out of process) | — | D-09: codegraph deliberately does not duplicate this; `--check` under brew points at Homebrew rather than answering itself |

## Standard Stack

### Core

No new external dependency is needed for this phase. Detection is pure Go standard library.

| Package | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os` | stdlib (Go 1.26.5 confirmed installed on this machine) | `os.Executable()` (already called in `internal/cli/upgrade.go:45`), `os.Stat` for receipt existence | Already imported in this package; `os.Executable()`'s own doc explicitly names the symlink caveat this detector must handle |
| `path/filepath` | stdlib | `filepath.EvalSymlinks`, `filepath.Dir`, path-segment matching | `os.Executable()`'s own godoc: "If a stable result is needed, `path/filepath.EvalSymlinks` might help." [VERIFIED: pkg.go.dev/os#Executable] — this is the stdlib's own documented answer to D-03/D-12's resolve-through-symlinks requirement |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| — | — | — | No supporting libraries needed; `encoding/json` (stdlib) only if the planner's discretion chooses to parse `INSTALL_RECEIPT.json` fields rather than just check existence — CONTEXT.md's Claude's Discretion section notes existence-only is sufficient and avoids depending on a schema Homebrew owns and does not treat as a public contract (see Pitfall 2 below) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `filepath.EvalSymlinks` | `os.Readlink` in a manual loop | `EvalSymlinks` already handles the multi-hop/relative-symlink walk correctly and is the function the stdlib's own `os.Executable` doc recommends; hand-rolling the loop reintroduces exactly the kind of bug class this project's own "don't hand-roll" instinct exists to avoid |
| Path-shape regex/string match | `filepath.Match` glob | A `regexp` (or manual segment split via `filepath.SplitList`/iterating `filepath.Dir`) is more legible for "does this path contain a `Caskroom/<token>/<version>` triple of segments" than a glob pattern, and errors out predictably on malformed input rather than glob's silent non-match on edge cases; either is fine — Claude's Discretion, no locked decision |

**Installation:** none — no `go get` required for this phase.

**Version verification:** N/A (stdlib only). Go toolchain confirmed on this machine: `go version go1.26.5 darwin/arm64` [VERIFIED: `go version` on this machine].

## Package Legitimacy Audit

**Not applicable.** This phase installs no external packages — detection is implemented entirely with Go standard library (`os`, `path/filepath`, optionally `encoding/json`). No `go.mod` changes are expected. If the planner's task breakdown surfaces an unexpected need for a third-party package, re-run the Package Legitimacy Gate protocol before including it.

## Linuxbrew Cask Finding (the load-bearing research result)

D-04 asked this phase to confirm, before locking criterion 3's linuxbrew scope to "Cellar shape only", that "Homebrew on Linux does not support casks." **This is false for the Homebrew release line this project is currently on.**

**What was true (and is the likely source of the assumption):** as recently as a 2022 Homebrew maintainer discussion, "Casks are only for macOS" [CITED: github.com/orgs/Homebrew/discussions/3999, maintainer reply, 2022-11-29] and `brew cask` on Linuxbrew returned "Casks are not supported on Linux."

**What changed:** Homebrew PR #19121, "feat: allow linux binaries in casks", enables casks that contain **only `binary` or `zap` stanzas** to be installed on Linux, with the explicit caveat (author's own words) that this initially requires `--cask` explicitly rather than going through Homebrew's API-driven auto-detection [CITED: github.com/Homebrew/brew/pull/19121]. Homebrew's own 6.0.0 release notes (June 2026 — the same major version installed on this machine and the one Phase 3 rehearsed against, `Homebrew 6.0.16-2-g007333f` [VERIFIED: `brew --version` on this machine]) list four further Linux-cask items shipped since: AppImage support for Linux, a Linux freedesktop trash implementation for casks, "makes Linux cask requirements explicit" (PR #21909), "`caskroom` use the user's primary group on Linux" (PR #22202), and "emits Linux variations for casks with Linux checksums" (PR #22632) [CITED: brew.sh/2026/06/11/homebrew-6.0.0].

**Why this matters for codegraph specifically:** codegraph's own cask (`.goreleaser.yaml:495-523`) declares `binaries: [codegraph]` and `generate_completions_from_executable` — no `app`, `pkg`, or `zap` stanza. This is exactly the shape PR #19121 made installable on Linux. Whether `brew install --cask` for this specific cask (with its `hooks.post.install`/`post.uninstall` Ruby blocks) actually succeeds end-to-end on a real Linuxbrew host was **not measured this session** (out of this phase's scope per D-13 — that would be re-testing Homebrew/Phase 3's publication, not this phase's detection code) — but the premise that the Caskroom shape is categorically unreachable on Linux, which D-04 uses to justify testing only the Cellar shape there, no longer holds for a binary-only cask on current Homebrew.

**Consequence for planning:** the detector itself is unaffected — it is prefix-agnostic by construction (D-03) and does not special-case OS. What needs a maintainer decision is criterion 3's **test scope**: whether the linuxbrew constructed-tree test should now cover both Cellar and Caskroom shapes (matching what current Homebrew can actually produce on Linux for this project's own cask), not Cellar alone. This is cheap either way — the constructed-tree fixture work is Claude's Discretion and prefix-agnostic detection means adding a second linuxbrew case is a few more lines, not new detection logic — but D-04's own inline rationale ("the Caskroom leg is unreachable there by construction") is the part that needs correcting before it's written into ROADMAP/REQUIREMENTS wording, or a future reader will trust a citation that no longer matches reality.

## Architecture Patterns

### System Architecture Diagram

```
codegraph upgrade [--check] [--force] [version]
        │
        ▼
internal/cli/upgrade.go: newUpgradeCmd().RunE
        │  os.Executable() → targetPath (raw, may be a symlink)
        ▼
internal/upgrade.Run(currentVersion, targetPath, opts)
        │
        ▼
  ┌─────────────────────────────────────────────┐
  │ NEW: detectBrewManaged(targetPath)           │  ← D-11: fires FIRST,
  │   1. filepath.EvalSymlinks(targetPath)       │     before any network call
  │   2. match Caskroom/<token>/<ver> OR         │
  │      Cellar/<name>/<ver> segment shape       │
  │   3. os.Stat(<ancestor>/INSTALL_RECEIPT.json)│
  │      (Caskroom: .../.metadata/...; Cellar:   │
  │       .../<name>/<ver>/INSTALL_RECEIPT.json) │
  └─────────────────────┬─────────────────────┬─┘
           detected     │                     │  not detected
                        ▼                     ▼
        ┌───────────────────────┐   resolveLatest(releaseRepoSlug)  (existing)
        │ Check?                │             │
        │  yes → step-aside msg,│             ▼
        │        exit 0 (D-09/  │   Check? ──yes──▶ report, exit 0 (existing,
        │        D-10)          │        │           unchanged)
        │  no  → refusal error, │        no
        │        exit != 0      │        ▼
        │        (D-05..D-08)   │   same-version guard → checkWritable →
        └───────────────────────┘   download → verify → swap  (existing,
                                     unchanged, never reached under brew)
```

A reader can trace both the existing (non-brew) path and the new brew-refusal path from `codegraph upgrade`'s entry point down to either an exit or the pre-existing resolve→download→verify→swap sequence, without needing to open the file — the new branch is a single insertion point at the head of `Run()`.

### Recommended Project Structure

```
internal/upgrade/
├── upgrade.go        # existing — Run() gains one early call to detectBrewManaged (D-11)
├── swap.go           # existing — checkWritable/atomicSwap, the refusal-shape precedent
├── verify.go          # existing — unrelated to this phase, untouched
├── brew.go            # NEW (D-12) — detection: path-shape match + receipt stat
├── brew_test.go        # NEW (D-12) — constructed-tree tests, incl. the false-positive case (criterion 3)
└── upgrade_test.go     # existing — gains D-08's seam-based refusal assertion, same idiom as
                         #            TestUpgradeRun_TamperedDownloadNeverSwaps (fakes recording
                         #            download/swap calls)
```

### Pattern 1: Early error-returning refusal inside `Run()` (the `checkWritable` precedent)

**What:** A function called at the head of an orchestration function that returns a descriptive error and causes the caller to `return` immediately, before any subsequent step (network, download, mutation) runs.
**When to use:** Exactly D-05/D-11's shape — a precondition that must gate everything downstream, offline, before any resource is touched.
**Example (existing code, the shape to follow):**
```go
// Source: internal/upgrade/upgrade.go:117-121 (read this session)
// D-13: refuse a non-writable target BEFORE downloading anything — no
// wasted download for an upgrade that can't be installed anyway.
if err := checkWritable(targetPath); err != nil {
    return err
}
```
The new brew-detection branch follows the identical shape, placed even earlier (before `resolveLatest`, per D-11), and branches on `opts.Check` to choose between the step-aside report (exit 0) and the hard refusal (non-zero) rather than always returning an error.

### Pattern 2: Seam-based proof via existing func-var fakes (D-08)

**What:** Assert control flow ("did our code call X") by injecting a fake for the func-var seam and recording whether it was invoked, rather than asserting an indirect filesystem side effect (sha256/mtime).
**When to use:** Exactly D-08's replacement for the rejected sha256/mtime proposal — this is already the idiom `upgrade_test.go` uses for `TestUpgradeRun_TamperedDownloadNeverSwaps` and `TestUpgradeRun_CheckReportsAvailabilityWithoutDownloading`.
**Example (existing code, read this session):**
```go
// Source: internal/upgrade/upgrade_test.go:15-48 (TestUpgradeRun_CheckReportsAvailabilityWithoutDownloading)
var downloadCalled, verifyCalled, swapCalled bool
opts := Options{
    Check: true,
    resolveLatest: func(repoSlug string) (string, error) { return "v1.2.3", nil },
    download: func(v string) ([]byte, []byte, error) { downloadCalled = true; return nil, nil, nil },
    verify:   func(binary, bundleJSON []byte) error { verifyCalled = true; return nil },
    swap:     func(targetPath string, newBinary []byte) error { swapCalled = true; return nil },
}
if err := Run("v1.0.0", "/does/not/matter", opts); err != nil { /* ... */ }
if downloadCalled || verifyCalled || swapCalled { /* fail */ }
```
D-08's new test for the brew-refusal path is this same shape: seed `targetPath` as a constructed Caskroom-or-Cellar tree with a real `INSTALL_RECEIPT.json`, assert `Run` returns a non-nil error naming `brew upgrade codegraph`, and assert `downloadCalled`/`swapCalled` are both still false.

### Pattern 3: `filepath.EvalSymlinks` before any path-shape comparison

**What:** Resolve the executable path through symlinks before doing any string/segment matching against it.
**When to use:** Any time `os.Executable()`'s return value is compared structurally (as opposed to just used as an opaque "file to replace" — which is all `checkWritable`/`atomicSwap` need it for today).
**Example:**
```go
// Pattern, not existing code — this is what brew.go needs to do that
// upgrade.go currently does NOT do with targetPath.
resolved, err := filepath.EvalSymlinks(targetPath)
if err != nil {
    // A path that can't be resolved (e.g. already removed) is not a
    // brew-managed install signal — fail open to "not detected", not to
    // an error that blocks the whole upgrade command (D-11's offline
    // guarantee shouldn't turn into an upgrade-blocking bug on a
    // filesystem race).
    resolved = targetPath
}
```
Source rationale: `os.Executable()`'s own godoc [VERIFIED: pkg.go.dev/os#Executable] states the return value may or may not already be resolved depending on OS — the Ruby cask hook this phase's detection must match (`Pathname.new(binary).realpath.dirname`, `.goreleaser.yaml:608-609`) already does full resolution, so the Go side must too or the two won't agree on directories at the boundary Homebrew's `bin/codegraph` symlink creates.

### Anti-Patterns to Avoid

- **Path-prefix guessing** (e.g. `strings.HasPrefix(path, "/opt/homebrew")`): explicitly rejected by D-03/D-12 and REQUIREMENTS UPGR-02 itself — fails on Intel `/usr/local`, custom prefixes, and linuxbrew, and is exactly the "vacuous gate" shape repo rule `84d1gfpywd` targets (a detector that would never fire on this project's own measured install, per D-01's finding about the stale `Cellar` wording).
- **Shelling out to `brew --prefix` / `brew list`**: rejected by D-03's "Rejected" list — breaks criterion 4 (brew-absent machine must still upgrade normally) if the shell-out itself errors ungracefully, adds a subprocess to every invocation of `codegraph upgrade`, and tests Homebrew's CLI rather than this project's own filesystem-shape wiring.
- **Sentinel-file detection** (D-08's rejected "sentinel primary" and "structural OR sentinel"): explicitly superseded by D-02's removal — do not resurrect the `.codegraph-brew-install` file or any successor sentinel as part of this phase's detection mechanism.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Resolving a path through symlinks reliably cross-platform | A manual `os.Readlink` loop with a hop-count guard | `filepath.EvalSymlinks` (stdlib) | Stdlib already handles multi-hop, relative-symlink, and platform differences; this is precisely what `os.Executable()`'s own doc recommends for a "stable result" |
| Detecting "is Homebrew present on this machine" | A `PATH`-scanning heuristic or shelling out to `brew --version` on every upgrade | Nothing — D-03's detector never needs to know whether `brew` is on `PATH` at all; it only inspects the resolved install path itself, which is why criterion 4 (brew absent) is automatically satisfied by construction (D-11) | The detector's job is "is *this specific binary* Homebrew-managed", not "is Homebrew installed anywhere" — conflating the two would add exactly the brew-CLI dependency D-03 explicitly rejected |

**Key insight:** every hand-rolling temptation in this phase (manual symlink walking, shelling out to `brew`, reviving the sentinel) has already been explicitly rejected in CONTEXT.md's Decisions with a named reason — this phase's "don't hand-roll" risk is not a novel one, it's re-introducing something already tried and rejected in the same milestone (the sentinel) or the prior phase (Cellar-prefix guessing, in D-01's own retelling of why `Cellar` was wrong).

## Runtime State Inventory

This phase removes a runtime artifact (the Phase-3 sentinel) that a prior phase's release pipeline writes onto real users' machines, so the rename/refactor trigger applies to the **removal** half of this phase (D-02), even though the phase's headline feature (detection) is greenfield.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | The sentinel file `.codegraph-brew-install` (schema=1, 6 `key=value` lines), written by `hooks.post.install` at `Pathname.new(binary).realpath.dirname` — measured this milestone to resolve to `<prefix>/Caskroom/codegraph/<version>/` (`03-02-SUMMARY.md:128`, confirmed again in `03-05-SUMMARY.md`) | **Code edit only, no data migration.** The sentinel lives *inside* the Caskroom versioned directory Homebrew purges wholesale on both `brew uninstall` and a failed-install rollback (`03-05-SUMMARY.md`'s directly-observed finding: "Homebrew's own install-failure rollback... purges the Caskroom's versioned directory"). Nothing on an already-installed user's machine needs cleanup — the next `brew upgrade`/`brew uninstall` naturally removes the old sentinel along with everything else in that directory. Removing the write in `.goreleaser.yaml` is sufficient; no migration script is needed. |
| Live service config | None — the sentinel is not registered with any external service, only written to the local filesystem by the cask's own hook. | None. |
| OS-registered state | None — no launchd/systemd/Task Scheduler registration involved anywhere in this phase. | None. |
| Secrets/env vars | None — no secret or env-var name changes in this phase. | None. |
| Build artifacts | `.goreleaser.yaml`'s cask block and `Taskfile.yml`'s `release:rehearse-cask` target both currently assert the sentinel's presence/schema (lines identified below) — these are **build/CI-time artifacts of Phase 3**, not something shipped to end users, but they must be edited in the same change or CI will fail asserting a file the hook no longer writes. | **Code edit, four locations** (verified by `rg` this session, exact line numbers current as of this research): `.goreleaser.yaml:451` (comment referencing "D-08's sentinel", now stale), `.goreleaser.yaml:570-588` (the write itself, inside `hooks.post.install`), `.goreleaser.yaml:623-624` (the `FileUtils.rm_f` inside `hooks.post.uninstall`), `Taskfile.yml:2006-2030` (Step 5b's positive sentinel assertions — schema-line check + six-key loop), `Taskfile.yml:2072` (idempotency check references `SENTINEL_PATH`), `Taskfile.yml:2091-2095` (post-uninstall symmetry check references `SENTINEL_PATH`), `Taskfile.yml:2211-2215` (the `CASK-REHEARSE-EVIDENCE schema=2` line interpolates `${SENTINEL_VERDICT}`). No `goreleaser_shape_test.go`/`taskfile_shape_test.go` assertion currently pins the sentinel (confirmed by `rg 'sentinel|codegraph-brew-install'` against both files this session, zero matches) — so no shape-test edit is needed, only the `.goreleaser.yaml`/`Taskfile.yml` bodies themselves. |

**Nothing found in category:** Live service config, OS-registered state, and secrets/env vars are all explicitly "None" — verified by reading the sentinel's write/remove sites and confirming they touch only the local filesystem inside a directory Homebrew itself owns and purges.

## Common Pitfalls

### Pitfall 1: Trusting `os.Executable()`'s raw return value for shape matching

**What goes wrong:** `os.Executable()` may return the symlink path or the resolved target depending on OS — comparing the raw value against a `Caskroom/.../` segment shape can silently fail to detect a real brew install (or, less likely, match a coincidental symlink name).
**Why it happens:** The function's contract is explicitly "no guarantee" per its own doc; it's easy to assume Go always resolves symlinks the way some other languages' equivalents do.
**How to avoid:** Always call `filepath.EvalSymlinks` on the value before any shape comparison — mirrors the Ruby hook's own `Pathname#realpath` call this Go code must agree with.
**Warning signs:** A constructed-tree test that stages the executable directly (no symlink) would pass even with this bug present — the false-negative would only show up when the fixture actually models Homebrew's real `bin/<name>` → `Caskroom/.../` symlink structure, which is why the fixture's realism matters (see Claude's Discretion above).

### Pitfall 2: Depending on `INSTALL_RECEIPT.json`'s internal schema, not just its existence

**What goes wrong:** Parsing specific JSON fields out of the receipt couples this project's detector to a file Homebrew's own documentation marks as **private API** — `AbstractTab` (which both `Tab` and `Cask::Tab` subclass) is documented at `rubydoc.brew.sh` as "part of a private API... may only be used in the Homebrew/brew repository" [CITED: rubydoc.brew.sh/AbstractTab.html, search-derived, not independently fetched and quoted verbatim this session — treat as MEDIUM confidence pending a direct fetch]. A schema change in a future Homebrew release could silently break field-level parsing with no deprecation notice, since it's not a contract Homebrew commits to for third parties.
**Why it happens:** The receipt's *existence and location* (`.metadata/INSTALL_RECEIPT.json` under the Caskroom token directory, or directly under `Cellar/<name>/<version>/`) is old, stable-shaped, source-confirmed behavior; its *contents* are not documented as a public contract at all.
**How to avoid:** This is exactly what CONTEXT.md's Claude's Discretion note already anticipates — "Existence is sufficient for D-03; parsing it is optional and adds a dependency on a format Homebrew owns." Recommend the planner lock this to existence-only (`os.Stat`, check `err == nil`), not `encoding/json` field parsing, unless a specific field becomes load-bearing for the refusal message (D-07 only needs the resolved *path*, which the detector already has independent of the receipt's contents).
**Warning signs:** Any code path that does `json.Unmarshal` on the receipt and then branches on a specific key is over-coupling to an internal format.

### Pitfall 3: Assuming Cellar/formula receipts and Caskroom/cask receipts share identical age/stability guarantees

**What goes wrong:** Formula `INSTALL_RECEIPT.json` is old (present since early Homebrew versions, part of the long-standing Formula-Cookbook-documented Cellar layout). Cask `INSTALL_RECEIPT.json` is comparatively new — merged in PR #17554 on 2024-07-13 [CITED: github.com/Homebrew/brew/pull/17554]. A cask installed by a Homebrew version older than that PR's release would have no receipt, and D-03's shape-AND-receipt detector would fail open (treat it as "not brew-managed") rather than refuse.
**Why it happens:** Both receipts share a filename and a conceptual role, which can lead to treating them as equally battle-tested.
**How to avoid:** This is a **false-negative-only** risk (an ancient-Homebrew user's `codegraph upgrade` behaves as if brew-unmanaged, i.e. upgrades normally over a brew-installed binary) — it does not threaten criterion 3's false-positive leg. Acceptable per D-11's ownership-test framing (a Homebrew old enough to predate this 2024 feature is itself an edge case Homebrew's own ecosystem is moving away from — `brew update` self-upgrades Homebrew itself on every invocation in the common case). Document this as a known, accepted limitation rather than building version-detection logic to compensate — that would be re-implementing Homebrew's own concerns (D-13's ownership test again).
**Warning signs:** A bug report from a user with a several-years-stale, never-`brew update`d Homebrew install where `codegraph upgrade` overwrote a Caskroom binary — expected and documented, not a regression to chase.

### Pitfall 4: `--force`'s meaning silently expanding

**What goes wrong:** A future edit accidentally routes `--force` through the new brew-detection branch (e.g. `if opts.Force { skip detection }`), quietly reintroducing the escape hatch D-06 explicitly rejected.
**Why it happens:** `Force` already bypasses one guard (`TestUpgradeRun_ForceReinstallsSameVersion`) in the exact same function, so the two guards sit close together in `Run()` and a careless edit can conflate them.
**How to avoid:** Place the brew-detection branch (D-11) structurally before the `Force`-sensitive same-version check, and never read `opts.Force` inside the new branch at all — its absence from that code path is the enforcement mechanism, not a runtime check.
**Warning signs:** Any new test titled something like `TestUpgradeRun_ForceOverridesBrewDetection` existing and passing would itself be the regression — D-06 says this must not be buildable.

## Code Examples

### Existing early-refusal precedent (verbatim, read this session)

```go
// Source: internal/upgrade/swap.go:18-28 (checkWritable, read this session)
func checkWritable(targetPath string) error {
	dir := filepath.Dir(targetPath)
	f, err := os.CreateTemp(dir, ".codegraph-upgrade-writable-check-*")
	if err != nil {
		return fmt.Errorf("upgrade: %s is not writable by the current user; refusing to upgrade (no changes made): %w", targetPath, err)
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return nil
}
```

### Existing `Run()` head, showing exactly where D-11's new branch inserts (verbatim, read this session)

```go
// Source: internal/upgrade/upgrade.go:82-105 (Run, read this session)
func Run(currentVersion, targetPath string, opts Options) error {
	out := opts.Out
	if out == nil {
		out = io.Discard
	}

	resolveLatest := opts.resolveLatest
	if resolveLatest == nil {
		resolveLatest = defaultResolveLatest
	}

	latest, err := resolveLatest(releaseRepoSlug)
	if err != nil {
		return fmt.Errorf("upgrade: %w", err)
	}

	if opts.Check {
		if latest == currentVersion {
			fmt.Fprintf(out, "codegraph %s is up to date\n", currentVersion)
		} else {
			fmt.Fprintf(out, "codegraph %s is available (current: %s)\n", latest, currentVersion)
		}
		return nil
	}
	// ...
```
D-11 requires the new brew-detection call to land **above** the `resolveLatest` call shown here (not just above the `opts.Check` branch) — both the refusal and the `--check` step-aside must be reachable with zero network calls.

### Go stdlib symlink-resolution contract (source-verified this session)

```
func Executable() (string, error)

Executable returns the path name for the executable that started the
current process. There is no guarantee that the path is still pointing
to the correct executable. If a symlink was used to start the process,
depending on the operating system, the result might be the symlink or
the path it pointed to. If a stable result is needed,
path/filepath.EvalSymlinks might help.
```
Source: pkg.go.dev/os#Executable [VERIFIED: fetched this session]. This is the stdlib's own confirmation that D-12's "resolve `os.Executable()` through symlinks" instruction is necessary, not defensive over-engineering.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| "Casks are macOS-only" (accurate as of the 2022 Homebrew maintainer statement this milestone's D-04 likely drew on) | Homebrew now supports installing `binary`/`zap`-only casks on Linux, with ongoing Linux-cask infrastructure work (explicit Linux requirements, Caskroom group handling, Linux checksum variants) | PR #19121 merged 2025; Homebrew 6.0.0 (June 2026) shipped four further Linux-cask items | codegraph's own cask is exactly the `binary`-only shape this now covers — D-04's "Caskroom leg unreachable on Linux by construction" rationale needs a maintainer re-check before criterion 3's linuxbrew scope is finalized (see Linuxbrew Cask Finding) |
| Cask installs had no install receipt at all (pre-2024) | `Cask::Tab`/`INSTALL_RECEIPT.json` for casks, mirroring the long-standing formula receipt | PR #17554 merged 2024-07-13 | This is what makes D-03's detector possible at all for casks — a Homebrew old enough to predate this has no receipt to check (Pitfall 3), but any Homebrew a user is likely running today has it |

**Deprecated/outdated:** The Phase-3 sentinel mechanism itself (D-02) — superseded within the same milestone by structural detection, one phase after it was introduced. Not an external deprecation, but worth naming so a future reader doesn't wonder why a freshly-shipped mechanism is being removed one phase later: Phase 3's own D-08 explicitly deferred this choice to Phase 4, and this is Phase 4 making it.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `AbstractTab`'s "private API" framing is accurately summarized (search-derived from rubydoc.brew.sh, not independently re-fetched and quoted verbatim in this session after the first fetch) | Pitfall 2 | If Homebrew in fact treats this file's location as a stable, semi-public contract (as its 2024 addition and Cellar-parity design suggest), the "don't parse fields" caution is more conservative than strictly necessary — low-risk direction to err in, since existence-only detection is CONTEXT.md's own preferred discretion choice regardless |
| A2 | codegraph's specific cask (with `hooks.post.install`/`post.uninstall` Ruby blocks and `generate_completions_from_executable`, beyond the bare `binary`/`zap` stanzas PR #19121 names) is actually end-to-end installable via `brew install --cask` on a real Linuxbrew host today | Linuxbrew Cask Finding, Summary | This was NOT measured this session (out of scope per D-13 — it is Phase 3's publication surface, not Phase 4's detection code) — if hooks or completions generation turn out to be macOS-gated even though the `binary` stanza itself is Linux-installable, the practical urgency of widening criterion 3's linuxbrew test is lower than the finding suggests, though the *detector* should still be correct for whatever shape Homebrew does produce |
| A3 | No `goreleaser_shape_test.go`/`taskfile_shape_test.go` assertion pins the sentinel (confirmed via `rg` returning zero matches against both files this session) — this is a point-in-time grep result, not a guarantee that no assertion is added between now and when the planner's tasks execute | Runtime State Inventory | Low risk — re-run the same `rg` at plan-execution time before deleting anything, per the general "search before editing" instruction CONTEXT.md's canonical refs already carry forward |

**If this table is empty:** N/A — three assumptions recorded above, all LOW-to-MEDIUM risk and none blocking; the one HIGH-value item (D-04's premise) is not in this table because it was fully confirmed via two independent official sources (a merged Homebrew PR and Homebrew's own dated release-notes blog post), not left as an assumption.

## Open Questions

1. **Does criterion 3's linuxbrew test need to cover the Caskroom shape, not just Cellar?**
   - What we know: D-04's stated rationale ("Caskroom leg unreachable on Linux by construction") is factually outdated for the current Homebrew release line, and codegraph's own cask is exactly the shape Homebrew's Linux-cask work targets.
   - What's unclear: Whether the maintainer wants to (a) widen the linuxbrew constructed-tree test to include a synthetic Caskroom fixture, matching what real Homebrew could now produce, or (b) keep D-04's scope as-is with a corrected rationale comment (still valid as "cheap to test even though unreachable" reasoning, just not for the reason originally given), or (c) explicitly measure real installability on Linuxbrew before deciding (which would be new scope, and arguably brushes against D-13's "don't re-test Homebrew" boundary since the detector's own correctness doesn't depend on the answer).
   - Recommendation: Surface this finding to the maintainer before the plan locks criterion 3's wording; the detector code itself needs no change either way (it is already prefix- and OS-agnostic), so this only affects which fixtures the test suite constructs and how the ROADMAP/REQUIREMENTS amendment (D-01's mandate to fix stale wording) describes linuxbrew's scope.

2. **Should the refusal/step-aside message differentiate cask vs. formula installs?**
   - What we know: D-07 locks the message shape to "resolved path + exact command", and D-03's detector recognizes both shapes but this project ships no formula today.
   - What's unclear: Whether `brew upgrade codegraph` is the correct command name regardless of whether detection matched via Caskroom or Cellar (it is — `brew upgrade` handles both formulae and casks transparently), so this is likely a non-issue, but worth the planner confirming the message never needs a cask-vs-formula branch.
   - Recommendation: No branch needed — `brew upgrade codegraph` is Homebrew's own unified command for both install types; treat this as resolved, listed here only so the planner doesn't second-guess it.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Homebrew (`brew`) | Real-tap acceptance run (criteria 1/2 in CONTEXT.md's domain section — "the phase is not done until it has been run against the real tap") | ✓ (this research machine) | `Homebrew 6.0.16-2-g007333f` [VERIFIED: `brew --version` this session] | N/A for unit tests — `internal/upgrade`'s constructed-tree tests never shell out to `brew`, so CI does not need Homebrew installed at all; only the manual/human real-tap acceptance step does |
| Go toolchain | All implementation | ✓ | `go1.26.5 darwin/arm64` [VERIFIED: `go version` this session] | — |
| `seanb4t/homebrew-tap` (real published tap) | Acceptance run only, not unit tests | Not verified this session (external GitHub-hosted tap, Phase 3's output) | — | The constructed-Cellar/Caskroom-shaped fixtures in `t.TempDir()` fully substitute for CI/unit-test purposes per the phase description's own framing ("Implementation can proceed in parallel... against a constructed Cellar-shaped symlink tree") |

**Missing dependencies with no fallback:** None — the real tap is required only for the final human acceptance step (explicitly out of CI's critical path per the phase description), and every other dependency is already present.

**Missing dependencies with fallback:** None currently missing.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go's built-in `testing` package (`go test`) — confirmed by every existing file in `internal/upgrade` (`upgrade_test.go`, `swap_test.go`, `verify_test.go`, `goreleaser_shape_test.go`, `taskfile_shape_test.go`) |
| Config file | none — no `go.mod`-level test framework config; `Taskfile.yml` defines the `task test` invocation |
| Quick run command | `go test ./internal/upgrade/...` |
| Full suite command | `task test` (repo convention — `Taskfile.yml` is the single definition of every CI job body per this repo's own established pattern) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| UPGR-01 | Bare `codegraph upgrade` under a constructed brew-managed tree refuses, names `brew upgrade codegraph`, never calls `download`/`swap` | unit (seam-based, D-08) | `go test ./internal/upgrade/... -run TestUpgradeRun_RefusesBrewManaged -v` | ❌ Wave 0 — new test in `brew_test.go` |
| UPGR-01 | Real-tap acceptance: genuine `brew install`, `codegraph upgrade` refuses, Cellar/Caskroom binary sha256+mtime unchanged | manual/acceptance (per phase description, requires the real Phase-3 tap) | none — human-run against a real machine | N/A — explicitly not CI-automatable this phase |
| UPGR-02 | Detection fires on Caskroom/Cellar shape under `/opt/homebrew`, `/usr/local`, custom prefix, linuxbrew; does NOT fire on a non-brew path merely containing the string `Cellar`/`Caskroom` | unit (constructed-tree, table-driven) | `go test ./internal/upgrade/... -run TestDetectBrewManaged -v` | ❌ Wave 0 — new test in `brew_test.go`; the false-positive case is one table row, not a separate test, but MUST be present as an executing assertion per CONTEXT.md's explicit constraint |
| UPGR-02 | Detection does not fire, and `brew` absent from `PATH` does not affect upgrade at all | unit | same `TestDetectBrewManaged` table, plus reuse of existing `TestUpgradeRun_ValidPathVerifiesBeforeSwap`-style non-brew-path fixture | ❌ Wave 0 |
| UPGR-03 | `--check` under a constructed brew-managed tree reports how to upgrade, exits 0, calls none of download/verify/swap | unit (seam-based) | `go test ./internal/upgrade/... -run TestUpgradeRun_CheckBrewManaged -v` | ❌ Wave 0 |
| UPGR-03 | Real-tap acceptance: `--check` under genuine install mutates nothing, `brew upgrade codegraph` still succeeds afterward | manual/acceptance | none | N/A |

### Sampling Rate

- **Per task commit:** `go test ./internal/upgrade/... -v`
- **Per wave merge:** `task test`
- **Phase gate:** Full suite green before `/gsd-verify-work`, plus the real-tap acceptance run recorded separately (it is not part of the automated gate, per the phase description's own framing of "not done until run against the real tap")

### Wave 0 Gaps

- [ ] `internal/upgrade/brew.go` — the detection function itself (`detectBrewManaged` or similar name, per Claude's Discretion on shape)
- [ ] `internal/upgrade/brew_test.go` — constructed-tree table tests covering all four prefixes, the false-positive case, and the receipt-absent case
- [ ] `internal/upgrade/upgrade_test.go` — additions for D-08's seam-based refusal proof and the `--check` step-aside proof, in the same idiom as the existing tests in this file (no new file needed for these two, they belong beside the existing `Run()`-level tests)

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Not applicable — no auth surface in this phase |
| V3 Session Management | no | Not applicable |
| V4 Access Control | no | Not applicable — this is a local filesystem predicate, no privilege boundary crossed |
| V5 Input Validation | yes | `targetPath` is not user-supplied network input — it originates from `os.Executable()` — but `filepath.EvalSymlinks`'s error must be handled fail-open-to-"not detected" (not a panic or an uncontrolled error bubbling up and blocking a legitimate non-brew upgrade), per Pattern 3 above |
| V6 Cryptography | no | Not applicable — this phase adds no cryptographic surface; `internal/upgrade/verify.go`'s existing Sigstore verification is untouched and out of scope |

### Known Threat Patterns for this phase's stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| A crafted directory structure on disk (e.g. an attacker-controlled path literally named `.../Caskroom/foo/1.0/`) tricking detection into a false positive, causing `codegraph upgrade` to silently no-op instead of upgrading a genuinely vulnerable binary | Tampering / Denial of availability of the security-relevant self-upgrade path | D-03's receipt-existence requirement is exactly this mitigation — a path-shape match alone (rejected by D-03/D-12) would be spoofable by anyone able to create a matching directory name; requiring `os.Stat` to also succeed on a real `INSTALL_RECEIPT.json` at the correct ancestor raises the bar to "attacker can also write a plausible Homebrew receipt file", which is a much weaker practical threat than a bare directory-name string match and is the same class of defense repo rule `84d1gfpywd` (positive assertion, not absence-of-bad-shape) already generalizes |
| A malicious actor with local write access replacing the `Cellar`/`Caskroom` binary and its receipt to make an attacker binary masquerade as brew-managed, causing `codegraph upgrade` to refuse to self-heal a compromised install | Spoofing | Out of this phase's threat model — anyone with write access to the install directory can already replace the binary directly; `codegraph upgrade`'s refusal changes only which *command* a user runs next (`brew upgrade` vs. self-swap), not whether the machine is compromised. This mirrors D-06's own reasoning for why `--force` gets no override: the honest path (`brew uninstall`) is always available regardless of detection's outcome |

## Sources

### Primary (HIGH confidence)

- pkg.go.dev/os#Executable — fetched directly this session, quoted verbatim; confirms the symlink-resolution caveat D-12 depends on
- `internal/upgrade/upgrade.go`, `internal/upgrade/swap.go`, `internal/upgrade/verify.go`, `internal/upgrade/upgrade_test.go`, `internal/cli/upgrade.go` — all read directly this session, quoted verbatim where cited
- `.goreleaser.yaml` lines 400-460, 520-635 — read directly this session, quoted/paraphrased with line citations
- `Taskfile.yml` lines 1985-2100, 2195-2219 — read directly this session, quoted/paraphrased with line citations
- `Homebrew 6.0.16-2-g007333f` and `go1.26.5 darwin/arm64` — confirmed by direct command execution on this machine this session

### Secondary (MEDIUM confidence)

- github.com/Homebrew/brew/pull/19121 ("feat: allow linux binaries in casks") — fetched via WebFetch this session, official Homebrew/brew repository, merged PR
- brew.sh/2026/06/11/homebrew-6.0.0 — fetched via WebFetch this session, official Homebrew release-notes blog
- github.com/Homebrew/brew/pull/17554 ("Add cask install receipts") — fetched via WebFetch this session, official Homebrew/brew repository, merged 2024-07-13
- github.com/Homebrew/brew/blob/master/Library/Homebrew/tab.rb and Library/Homebrew/cask/metadata.rb — fetched via WebFetch this session against the current `master` branch; `AbstractTab::FILENAME = "INSTALL_RECEIPT.json"`, `METADATA_SUBDIR = ".metadata"`, and the `cask.metadata_main_container_path/FILENAME` construction confirmed
- `.planning/phases/03-homebrew-tap-cask/03-02-SUMMARY.md:128`, `03-05-SUMMARY.md`, `03-EVIDENCE.md` (lines ~100-180, ~250-290, ~790-835) — this project's own prior-phase measured evidence, read this session

### Tertiary (LOW confidence)

- github.com/orgs/Homebrew/discussions/3999 — a 2022 community discussion (maintainer reply quoted), used only to establish the historical baseline the "casks are macOS-only" assumption likely traces to, not as current-state evidence
- rubydoc.brew.sh/AbstractTab.html — summarized via WebSearch snippet, not independently re-fetched and quoted verbatim; flagged as Assumption A1 above pending direct confirmation if the planner wants HIGH confidence on the "private API" framing specifically

## Metadata

**Confidence breakdown:**
- Standard stack (stdlib-only mechanism): HIGH — every claim verified against either this repo's own source or pkg.go.dev's official docs, fetched directly
- Architecture (detection placement, seam reuse): HIGH — directly derived from reading `upgrade.go`/`swap.go`/`upgrade_test.go` this session, not inferred
- D-03 (receipt location/stability): MEDIUM-HIGH — path construction confirmed at the Homebrew source-code level (`tab.rb`, `cask/metadata.rb`) via direct fetch; the "private API, no stability contract" framing is MEDIUM (one search-summarized source, not re-verified verbatim)
- D-04 (linuxbrew cask support): **the assumption CONTEXT.md asked this phase to confirm is FALSE** — confirmed via two independently-fetched official Homebrew sources (a merged PR and a dated release-notes post). This is not a confidence gap; it is a scope-changing finding that needs a maintainer decision before the ROADMAP/REQUIREMENTS wording locks
- Pitfalls: HIGH for Pitfalls 1/3/4 (directly derived from source read this session); MEDIUM for Pitfall 2 (the "private API" framing is a search-derived summary, see Assumption A1)

**Research date:** 2026-08-10
**Valid until:** Homebrew's Linux-cask work is moving fast (multiple PRs merged within the last ~18 months as of this research) — treat the Linuxbrew Cask Finding specifically as valid for ~14 days before re-checking if planning stalls; the stdlib and this-repo-code findings are stable for the life of this phase (30 days, standard).
