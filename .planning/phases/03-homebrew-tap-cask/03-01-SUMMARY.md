---
phase: 03-homebrew-tap-cask
plan: 01
subsystem: infra
tags: [goreleaser, homebrew, cask, cobra, man-pages, gatekeeper, codesign]

# Dependency graph
requires:
  - phase: 02-signed-macos-binaries
    provides: ".goreleaser.yaml notarize:/signs:/sboms:/release: blocks, the release:rehearse-notarize maintainer-only rehearsal precedent"
provides:
  - "internal/cli/man.go — the hidden `codegraph man <dir>` command (not yet directory-creating; Task 3's scope)"
  - "internal/cli/root.go registration of newManCmd()"
  - "A minimal, production-shaped `homebrew_casks:` block in .goreleaser.yaml"
  - "Taskfile.yml `release:rehearse-cask` — the maintainer-only local cask-install rehearsal target, RED-then-partial-GREEN demonstrated"
  - "A measured, confirmed real-world finding: Homebrew Cask unconditionally quarantines every download, so this rehearsal (and by extension the shipped cask's post-install hook) categorically requires a genuinely Developer-ID-signed and notarized binary — not merely a config concern, a Gatekeeper enforcement fact"
affects: [03-02, 03-03, 03-04, 03-05]

# Actuals (#2632)
actuals:
  tokens: 9050
  tasks: 1
  commits: 1

# Tech tracking
tech-stack:
  added: ["github.com/spf13/cobra/doc (compiled dep as of this plan)", "github.com/cpuguy83/go-md2man/v2 v2.0.7 (newly compiled)", "github.com/russross/blackfriday/v2 v2.1.0 (newly compiled)"]
  patterns: ["Taskfile maintainer-only rehearsal target wrapping a rendered artifact in a throwaway local git-repo Homebrew tap, since modern Homebrew (6.0.16) rejects installing a loose cask .rb path directly"]

key-files:
  created: ["internal/cli/man.go"]
  modified: ["internal/cli/root.go", ".goreleaser.yaml", "Taskfile.yml", "go.mod", "go.sum"]

key-decisions:
  - "D-14 confirmed as recorded (pre-resolved checkpoint, not re-asked): tap seanb4t/homebrew-tap, cask token codegraph, contract `brew tap seanb4t/tap && brew install codegraph`."
  - "go-md2man/v2 and blackfriday/v2 promoted from module-graph-only (/go.mod hash) to compiled deps via `go build -mod=mod ./...` — `go mod tidy` could not be used because it fails module-wide on a pre-existing, unrelated internal/parser/cgo test-dependency issue (tree-sitter-swift); -mod=mod scopes the resolution to only what's actually imported."
  - "The homebrew_casks: block is deliberately minimal for this task: name/directory/ids/repository/homepage/description/binaries/hooks.post.install only. Completions, hooks.post.uninstall, D-11's second assertion, and D-08's sentinel are explicitly out of scope, deferred to plan 03-02."
  - "release:rehearse-cask wraps the rewritten cask in a throwaway local git-repo tap (tapped via `brew tap <name> file://<local-repo>`) — a real-environment finding not anticipated in RESEARCH.md/CONTEXT.md: modern Homebrew (6.0.16, measured this session) refuses `brew install --cask <path>.rb` directly with \"Homebrew requires casks to be in a tap\"."
  - "Measured man page count: 30 (not CONTEXT.md's ~27 estimate, not RESEARCH.md's traced 35). Root cause identified with certainty: doc.GenManTree(newRootCmd(), ...) never calls Execute() on the constructed tree, so Cobra's InitDefaultCompletionCmd (called only from command.go:1113, inside Execute()'s preparation path) never fires — the completion command and its 4 shell subcommands (5 pages) are absent from the generated tree, accounting for the 35-vs-30 gap exactly."

patterns-established:
  - "Maintainer-only local rehearsal targets that install real artifacts requiring Gatekeeper-cleared binaries MUST declare the same MACOS_SIGN_P12/MACOS_SIGN_PASSWORD/MACOS_NOTARY_ISSUER_ID/MACOS_NOTARY_KEY_ID/MACOS_NOTARY_KEY preconditions release:rehearse-notarize already uses — Homebrew Cask's unconditional quarantine (extend/os/mac/cask/quarantine.rb) makes this load-bearing, not optional, for ANY target that executes an installed cask binary."

requirements-completed: []  # BREW-04/BREW-05 NOT completed — Task 2's tracer verify did not pass; see Deviations/Issues below.

coverage:
  - id: D1
    description: "Hidden `codegraph man <dir>` Cobra command generating the full man-page tree, registered on newRootCmd()"
    requirement: "BREW-04"
    verification:
      - kind: unit
        ref: "go build ./... "
        status: pass
      - kind: integration
        ref: "go run ./cmd/codegraph man <dir> (manual, dir pre-created) — measured 30 files"
        status: pass
    human_judgment: false
  - id: D2
    description: "Minimal homebrew_casks: block in .goreleaser.yaml (ids: [zip], no url:, hooks.post.install invoking codegraph man)"
    requirement: "BREW-04"
    verification:
      - kind: unit
        ref: "task check:goreleaser"
        status: pass
      - kind: integration
        ref: "task release:dry-run, task release:dry-run-signed (both exercise the new homebrew cask pipe)"
        status: pass
    human_judgment: false
  - id: D3
    description: "release:rehearse-cask end-to-end proof: a real brew install --cask of the GoReleaser-rendered cask, with the post-install hook executing the installed binary and codegraph.1 landing on disk"
    requirement: "BREW-04"
    verification:
      - kind: manual_procedural
        ref: "task release:rehearse-cask (maintainer must supply real Apple Developer ID credentials)"
        status: fail
    human_judgment: true
    rationale: "Blocked in this session by the total absence of MACOS_SIGN_P12/MACOS_SIGN_PASSWORD/MACOS_NOTARY_ISSUER_ID/MACOS_NOTARY_KEY_ID/MACOS_NOTARY_KEY. Homebrew Cask unconditionally quarantines every download (measured, not assumed — see Deviations); an ad-hoc-signed, non-notarized binary is SIGKILLed by Gatekeeper the instant the post-install hook executes it. A1 (binary at zip root) and A3 (HOMEBREW_PREFIX/bin/codegraph live at postflight time) both remain UNCONFIRMED — the process was killed before either could be observed. Requires the maintainer to re-run this target locally with real credentials loaded."

# Metrics
duration: ~70min
completed: 2026-08-10
status: halted
---

# Phase 3 Plan 1: Homebrew Man-Page Tracer Summary

**`codegraph man` and a minimal `homebrew_casks:` block landed and validated everywhere reachable without Apple credentials; the final end-to-end proof (`task release:rehearse-cask`) is blocked on Gatekeeper's unconditional quarantine of Homebrew Cask downloads, measured directly this session, not assumed.**

## Performance

- **Duration:** ~70 min
- **Completed:** 2026-08-10T14:33:11Z
- **Tasks:** 1 of 3 committed (Task 1 checkpoint resolved pre-dispatch; Task 2 implemented but its tracer `<verify>` did not fully pass; Task 3 deferred per the tracer feedback gate)
- **Files modified:** 6 (1 created, 5 modified)

## Accomplishments

- `internal/cli/man.go` — the hidden `codegraph man <dir>` command (`Hidden: true`, `Args: cobra.ExactArgs(1)`, `doc.GenManTree` against a fresh `newRootCmd()`), registered in `newRootCmd()`'s `AddCommand` list and documented in both the package doc comment and `newRootCmd`'s own comment.
- `go-md2man/v2` and `blackfriday/v2` promoted from module-graph-only entries to compiled dependencies (`go build -mod=mod ./...`, since `go mod tidy` fails module-wide on an unrelated pre-existing `internal/parser/cgo` issue). Source-mode `govulncheck ./...` over the main module: **CLEAN, 0 symbol-reachable vulnerabilities** (1 vulnerability exists in an imported-but-not-called package, 1 in a required-but-not-imported module — neither reachable).
- A minimal, production-shaped `homebrew_casks:` block in `.goreleaser.yaml`: `ids: [zip]` (avoids `ErrMultipleArchivesSameOS`), no `url:` key (Pattern 2), `repository:` wired to `seanb4t/homebrew-tap` with `HOMEBREW_TAP_TOKEN`, and `hooks.post.install` invoking `codegraph man`. The `archives:` block is byte-unchanged (confirmed via diff against the pre-plan commit — additions only, zero deletions).
- `Taskfile.yml`'s new `release:rehearse-cask` target: demonstrated **RED** before any implementation existed (failed by naming `dist/homebrew/Casks/codegraph.rb does not exist or is empty` — never a crash), then demonstrated **partial GREEN** through the render, the exactly-one-line URL rewrite (verified via diff assertion), and a real `brew install --cask` of a throwaway local-tap-wrapped copy — the last step (a modern-Homebrew requirement neither RESEARCH.md nor CONTEXT.md anticipated) that itself required discovering and implementing a local-git-tap wrapper this session.
- `task release:dry-run` and `task release:dry-run-signed` both still exit 0 with the new `homebrew_casks:` block present, and the "homebrew cask" pipe now runs inside both (confirmed in their own log output).
- Measured man page count: **30** (neither CONTEXT.md's ~27 estimate nor RESEARCH.md's traced 35). Root-caused with certainty: `doc.GenManTree(newRootCmd(), ...)` never calls `.Execute()` on the tree it walks, so Cobra's `InitDefaultCompletionCmd` (fired only from `command.go:1113`, inside `Execute()`'s own preparation path) never runs — the `completion` command and its 4 shell subcommands (5 pages) are absent, which is exactly the 35-vs-30 gap.

## Task Commits

1. **Task 1: Confirm the tap name and cask token** — resolved pre-dispatch by the orchestrator (option `confirm-d14`); not re-asked, no commit of its own (decision already recorded in CONTEXT.md D-14).
2. **Task 2: One path, end to end** — `8646f19` (feat) — man.go, root.go, .goreleaser.yaml, Taskfile.yml, go.mod, go.sum. **Committed as real, correct, tested progress, but the task's own `<verify>` did not fully pass** (see Issues Encountered) — the commit message states this explicitly.
3. **Task 3: The man command's behavior contract** — **not started.** Per the tracer feedback gate (auto mode active, `workflow.auto_advance: true`), a failing tracer `<verify>` halts before any expansion task.

**Plan metadata:** (this commit, SUMMARY.md + WINDOWS.md)

## Files Created/Modified

- `internal/cli/man.go` — hidden `codegraph man <dir>` command
- `internal/cli/root.go` — registers `newManCmd()`, updates two doc comments
- `.goreleaser.yaml` — new `homebrew_casks:` top-level block (additions only, `archives:` byte-unchanged)
- `Taskfile.yml` — new `release:rehearse-cask` target (~215 lines, including the 5 Apple-credential preconditions added mid-session once their necessity was measured)
- `go.mod` / `go.sum` — `go-md2man/v2 v2.0.7`, `blackfriday/v2 v2.1.0` promoted to compiled/full-hash entries

## Decisions Made

- **Task 1 resolution recorded, not re-asked.** Decision id D-14, option `confirm-d14`: tap `seanb4t/homebrew-tap`, cask token `codegraph`, contract `brew tap seanb4t/tap && brew install codegraph`. Answered by the maintainer at orchestration time, pre-dispatch.
- **`go mod tidy` bypassed in favor of `go build -mod=mod ./...`.** The former walks every package's full transitive (including test) dependency graph and fails on `internal/parser/cgo`'s unrelated `tree-sitter-swift` binding resolution issue — a pre-existing condition, out of this plan's scope per the deviation-rule scope boundary. `-mod=mod` scopes resolution to only what `go build ./...` actually needs, producing the identical, minimal two-line `go.mod` diff.
- **`release:rehearse-cask` wraps the cask in a throwaway local git-repo tap.** Modern Homebrew (6.0.16, auto-updated mid-session by `brew install --cask` itself before `HOMEBREW_NO_AUTO_UPDATE=1` was added) refuses `brew install --cask <path>.rb` with "Homebrew requires casks to be in a tap" — a real, measured environment finding, not present in RESEARCH.md's trace (RESEARCH.md's source reads predate whatever Homebrew release introduced this restriction). The rehearsal creates a throwaway git repo under a run-scoped temp dir, `git init`s it, taps it via `brew tap <name> file://<path>`, installs by tap-qualified name, and untaps + uninstalls in the cleanup trap on both the pass and fail path.
- **Apple credential preconditions added to `release:rehearse-cask` mid-session, once discovered load-bearing.** Homebrew Cask's `Cask::Quarantine.cask!` (`extend/os/mac/cask/quarantine.rb`) unconditionally quarantines every downloaded artifact via a direct LaunchServices FFI call — regardless of URL scheme, so `file://` buys no exemption. There is no `--no-quarantine` install flag in this Homebrew version, and `spctl --add` (which could otherwise locally allowlist one binary's cdhash) was removed as of macOS 15.0 ("operations that modify the rule database... will no longer be supported", confirmed via `man spctl` on this host, running Darwin 27.0.0). Consequently, ANY target that installs a real cask and executes the installed binary — not just `release:rehearse-notarize`'s literal notarization submission — requires real Developer ID signing + notarization to avoid Gatekeeper SIGKILL. The target's precondition set now names the same five `MACOS_*` variables `release:rehearse-notarize` requires, so a future run without them fails fast and by name (measured: now fails in under a second at the precondition step) instead of after a ~15s build followed by a confusing `terminated by uncaught signal KILL`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `go mod tidy` fails module-wide; used `go build -mod=mod ./...` instead**
- **Found during:** Task 2, after adding the `github.com/spf13/cobra/doc` import
- **Issue:** `go mod tidy` attempts to resolve `internal/parser/cgo`'s test dependency on `github.com/alex-pinkus/tree-sitter-swift`, which in turn cannot resolve `github.com/tree-sitter/tree-sitter-swift/bindings/go` — a pre-existing condition unrelated to this plan's files
- **Fix:** Used `go build -mod=mod ./...` instead, which only resolves what `go build ./...` actually imports (the two new man-page dependencies), producing the identical minimal `go.mod`/`go.sum` diff `go mod tidy` would have produced for this scope
- **Files modified:** go.mod, go.sum
- **Verification:** `go build ./...` (without `-mod=mod`) exits 0 afterward; `git diff go.mod` shows exactly the two expected new `// indirect` lines
- **Committed in:** `8646f19`

**2. [Rule 3 - Blocking] Modern Homebrew rejects loose-path cask installs — wrapped the rehearsal cask in a throwaway local tap**
- **Found during:** Task 2, first `task release:rehearse-cask` GREEN attempt
- **Issue:** `brew install --cask <path>/codegraph.rb` failed: "Error: Homebrew requires casks to be in a tap, rejecting: ..." — RESEARCH.md's own source-level trace did not surface this restriction
- **Fix:** The target now creates a throwaway `git init`-ed local repo under a run-scoped temp dir, `brew tap`s it via a `file://` URL, and installs/uninstalls by the tap-qualified cask name; cleaned up (untapped) on both the pass and fail path
- **Files modified:** Taskfile.yml
- **Verification:** `brew tap` + `brew install --cask <tap>/codegraph` progressed past the prior rejection to the actual Cask installation step
- **Committed in:** `8646f19`

### Not Auto-fixed — Genuine External Blocker (documented, not worked around)

**[Not a Rule 1-3 case — no code fix exists] `task release:rehearse-cask` cannot reach PASS without real Apple Developer ID credentials**
- **Found during:** Task 2, second `task release:rehearse-cask` GREEN attempt (after the tap fix above)
- **Issue:** `brew install --cask` failed: `Error: Failure while executing; ".../codegraph man .../man1" was terminated by uncaught signal KILL.` — Gatekeeper killing the post-install hook's invocation of the freshly-installed, ad-hoc-signed-only binary
- **Root cause, measured directly (not inferred):**
  1. `Cask::Quarantine.cask!` (`extend/os/mac/cask/quarantine.rb`) unconditionally quarantines every cask download via a direct LaunchServices FFI call, called from `cask/download.rb` regardless of URL scheme.
  2. Manually reproduced outside Homebrew: copying the built binary, ad-hoc `codesign --force -s -`, then `xattr -w com.apple.quarantine ...` on it, then executing it — **exit 137 (SIGKILL)**, confirming quarantine + ad-hoc-only signing is sufficient to trigger the kill on its own, independent of anything cask-specific.
  3. `spctl --add` (which could otherwise locally allowlist a specific binary's designated requirement, entirely offline) is confirmed REMOVED as of macOS 15.0 per `man spctl` on this host: "operations that modify the rule database or the global state of the assessment subsystem will no longer be supported."
  4. No environment variable in this session (`MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_ISSUER_ID`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_KEY`) is set — confirmed via a scoped `env | rg` check, zero matches.
- **Why this was NOT auto-fixed:** there is no code-level or configuration-level remedy available to the executor. Real Apple Developer ID signing and App Store Connect API notarization credentials are held only by the maintainer and cannot be substituted, generated, or bypassed by an agent (analogous to an OAuth login gate — see `<authentication_gates>`). Disabling Gatekeeper system-wide (`spctl --master-disable` / `csrutil`-adjacent settings) would be an unauthorized, destructive, machine-wide security change outside this agent's authority and was not attempted.
- **Action taken instead:** (a) added the same five `MACOS_*` preconditions `release:rehearse-notarize` already requires to `release:rehearse-cask`, so a future credential-less run fails fast and by name rather than after a confusing SIGKILL; (b) documented the mechanism in the target's `desc:` block with the measured evidence inline; (c) recorded an `unrun-verify` entry in `.planning/WINDOWS.md` (open); (d) halted before Task 3 per the tracer feedback gate; (e) this Summary's `coverage:` block marks D3 `human_judgment: true` with the full rationale above, rather than a false `pass`.
- **What unblocks it:** the maintainer running, on their own Mac, with real credentials loaded:
  ```
  MACOS_SIGN_P12=... MACOS_SIGN_PASSWORD=... \
  MACOS_NOTARY_ISSUER_ID=... MACOS_NOTARY_KEY_ID=... MACOS_NOTARY_KEY=... \
  CASK_REHEARSE=1 task release:rehearse-cask
  ```
  A subsequent run should reach the `CASK-REHEARSE-EVIDENCE` line with `outcome=pass`, at which point A1 and A3 can be marked CONFIRMED and Task 3 can proceed.

---

**Total deviations:** 2 auto-fixed (both Rule 3 — blocking, both necessary to make progress on files this plan already touches) + 1 genuine external blocker (documented, not worked around, not silently absorbed).
**Impact on plan:** Task 2's implementation is real and correct everywhere it could be exercised without Apple credentials (compiles, `task check:goreleaser` passes, `task release:dry-run`/`release:dry-run-signed` both pass with the new pipe running inside them, the RED demonstration is genuine, the tap-wrapping and URL-rewrite mechanisms are proven). The remaining ~30% of the tracer's own `<verify>` — the actual Gatekeeper-cleared execution of the installed binary — is gated on credentials this session does not have. Task 3 was correctly NOT started per the tracer feedback gate.

## Issues Encountered

- **Homebrew auto-updated itself (6.0.15 → 6.0.16) mid-session**, triggered by the first `brew install --cask` attempt before `HOMEBREW_NO_AUTO_UPDATE=1`/`HOMEBREW_NO_ENV_HINTS=1` were added to the rehearsal target. This is an environment side effect outside this plan's file scope; not reverted (Homebrew has no supported downgrade path, and the version bump is not expected to affect anything this phase depends on).
- **`man <dir>` currently requires the destination directory to already exist** (Task 3's explicit scope — `doc.GenManTree` does not create its own destination, and Task 2's rehearsal, per the plan's own design, was expected to run against a Homebrew prefix where `man1` already exists). Verified directly: `codegraph man /nonexistent-path` fails with `no such file or directory`. This is NOT a bug in Task 2's implementation — it is intentionally out of scope, deferred to Task 3's TDD RED/GREEN cycle, which did not run this session.

## Known Stubs

None. `internal/cli/man.go` is a complete, real implementation of its Task-2-scoped behavior (no directory creation, no unwritable-path wrapping — those are Task 3's explicitly-scoped additions, not stubs standing in for missing functionality).

## Threat Flags

None beyond what the plan's own `<threat_model>` already covers. No new network endpoints, auth paths, or trust-boundary schema changes were introduced beyond what T-03-01/T-03-02/T-03-03/T-03-SC already anticipate.

## User Setup Required

**External credentials require manual configuration to complete this plan.** The maintainer must supply, in their own local shell (never committed, never logged):
- `MACOS_SIGN_P12` — base64-encoded Developer ID Application `.p12` certificate (or a filesystem path)
- `MACOS_SIGN_PASSWORD` — the password chosen when exporting the `.p12`
- `MACOS_NOTARY_ISSUER_ID` — App Store Connect API key Issuer ID
- `MACOS_NOTARY_KEY_ID` — App Store Connect API key Key ID
- `MACOS_NOTARY_KEY` — the App Store Connect API key itself (base64-encoded `.p8`, or a filesystem path)

See `docs/RELEASE.md` for how each is obtained (same five variables `release:rehearse-notarize` already documents). Then re-run:
```
CASK_REHEARSE=1 task release:rehearse-cask
```

## Next Phase Readiness

- **Not ready for plan 03-02 to build on this plan's cask mechanism as "proven."** The tracer's own purpose — proving the whole Homebrew delivery path end-to-end before expansion work is built on it — has NOT been achieved. A1 (binary at zip root, no rename handling needed) and A3 (`#{HOMEBREW_PREFIX}/bin/codegraph` live at postflight time) both remain **UNCONFIRMED**, not falsified — the process was killed by Gatekeeper before either could be observed one way or the other.
- **What IS proven and safe to build on:** the `codegraph man` command itself (compiles, registered, hidden, generates 30 real man pages against a pre-existing directory); the `homebrew_casks:` block's shape (validates under `task check:goreleaser`, coexists cleanly with `release:dry-run`/`release:dry-run-signed`, `archives:` byte-unchanged); the render mechanism (`dist/homebrew/Casks/codegraph.rb` renders correctly from a real `goreleaser release --snapshot` run); the URL-rewrite mechanism (exactly one line, diff-verified); and the local-tap-wrapping mechanism now required for ANY future local cask rehearsal in this repository (a pattern plan 03-02 and later plans should reuse rather than rediscover).
- **Blocker for full closure:** maintainer must re-run `release:rehearse-cask` locally with real Apple credentials (see User Setup Required above). Once that run reaches `CASK-REHEARSE-EVIDENCE ... outcome=pass`, Task 3 (man_test.go TDD, docs/FLAG-PARITY.md entry) should be executed as a follow-up, and this plan's status should be updated from `halted` to `complete`.
- **`.planning/WINDOWS.md`** now carries one open `unrun-verify` entry for this blocker — it must be resolved (not silently dropped) before this milestone ships.

---
*Phase: 03-homebrew-tap-cask*
*Completed: 2026-08-10*

## Self-Check: PASSED

- FOUND: `internal/cli/man.go`
- FOUND: `release:rehearse-cask` target in `Taskfile.yml`
- FOUND: `homebrew_casks:` block in `.goreleaser.yaml`
- FOUND: commit `8646f19` in `git log --oneline --all`
