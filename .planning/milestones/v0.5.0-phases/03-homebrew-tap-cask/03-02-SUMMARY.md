---
phase: 03-homebrew-tap-cask
plan: 02
subsystem: infra
tags: [goreleaser, homebrew, cask, cobra, ruby, gatekeeper, notarization]

# Dependency graph
requires:
  - phase: 03-homebrew-tap-cask
    provides: "03-01's proven tracer — a minimal homebrew_casks: block, codegraph man, and a real, credentialed release:rehearse-cask PASS"
provides:
  - "The completed BREW-05 install gate: hooks.post.install carries two positive assertions (man-page presence, version equality) that raise and roll back a real brew install on failure"
  - "The Phase-4 sentinel (.codegraph-brew-install) written beside the real, symlink-resolved installed binary — the cross-phase contract internal/upgrade reads in Phase 4"
  - "hooks.post.uninstall — symmetric removal of the man pages and sentinel this cask writes, using staged_path (not a live symlink) since Homebrew's own artifact uninstall order removes the binary symlink first"
  - "generate_completions_from_executable (BREW-03): bash/zsh/fish completions generated from the installed binary at install time, with every discretionary cask metadata field (platform scope, livecheck, auto_updates, depends_on) decided and recorded"
  - "release:rehearse-cask extended to schema=2: sentinel/completions/post-uninstall verdicts, reinstall idempotency, a real version-mismatch perturbation, a real completions-bad-executable perturbation, and a rehearsal-fidelity fix (the declared version line is now rewritten to the LOCAL build's real reported version, since --snapshot mode's synthetic pseudo-version would otherwise permanently mismatch the ldflags-injected binary version)"
  - "7 new shape tests in internal/upgrade/goreleaser_shape_test.go (5 TestHomebrewCask* property tests + 2 non-vacuity companions), each demonstrated RED against a real, confirmed-applied, byte-cleanly-reverted mutation"
affects: [03-03, 03-04, 03-05, 04-upgrade-brew-detection]

# Actuals (#2632)
actuals:
  tokens: 11285
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns: ["Rehearsal-copy line rewriting for a snapshot-mode-only fidelity gap (declared version, alongside the pre-existing download-URL rewrite) — same technique, extended", "OK=0/continue assertion accumulation in a single Taskfile rehearsal run (mirrors release:dry-run-signed's own idiom), so one invocation surfaces every new-behavior failure instead of stopping at the first"]

key-files:
  created: []
  modified: [".goreleaser.yaml", "Taskfile.yml", "internal/upgrade/goreleaser_shape_test.go"]

key-decisions:
  - "SUPERSEDES this file's own prior halted state. The predecessor session correctly found .env absent in its worktree (git worktrees do not contain gitignored files) but incorrectly cited `op whoami` failure as evidence credentials were unreachable — `op whoami` tests interactive account sign-in, not `op run`'s non-interactive op:// resolution, which the orchestrator measured working on this host before dispatching this session. This session ran on the main checkout (not a worktree) specifically so `.env` was reachable, and confirmed `op run --env-file=.env -- ...` resolves real Apple/notary credentials before starting any task work."
  - "Task 1's RED-before-GREEN sequence was run for real: Taskfile.yml's assertions were extended FIRST while .goreleaser.yaml still carried 03-01's minimal hook, a full credentialed op-run rehearsal was executed and observed failing on exactly the four new behaviors (sentinel absent, idempotency reinstall missing sentinel, post-uninstall man pages not removed, version-mismatch perturbation installing successfully when it should not), then .goreleaser.yaml was completed and the SAME rehearsal re-run to GREEN."
  - "Deviation (Rule 1/3, not Rule 4): the version-equality assertion, once real, immediately fired against release:rehearse-cask's OWN snapshot-mode render (declared '0.7.0-SNAPSHOT-91b6df6' vs binary-reported '0.7.0') — not a hook bug, a rehearsal-fidelity gap the new assertion's correctness surfaced. --snapshot mode gives GoReleaser's rendered cask version (ctx.Version, a SNAPSHOT-suffixed pseudo-version) and the binary's ldflags-injected version (set from GoReleaser's Tag field, the nearest real tag) two different values; in a REAL tagged release these are the same string. Fixed by rewriting the rehearsal copy's declared version line to the LOCAL build's actual reported version (queried directly from the pre-archive build artifact via dist/artifacts.json, no brew install needed to learn it) — mirroring the existing download-URL rewrite. The gate's behavior is unchanged; only the rehearsal's fidelity to what a real tagged release renders."
  - "Sentinel location: written at post-install via Pathname.new(binary).realpath.dirname (literal symlink resolution, matching the plan's explicit 'resolved through symlinks, never a path-prefix guess' requirement) but removed at post-uninstall via staged_path — NOT a re-resolved symlink. Confirmed from GoReleaser's own cask.rb.tmpl and Homebrew's Cask::Installer#uninstall_artifacts source: stanzas render in declared order and artifacts uninstall in that same order, so the `binary \"codegraph\"` Symlinked artifact (declared before hooks: in the template) has already removed the #{HOMEBREW_PREFIX}/bin/codegraph symlink by the time hooks.post.uninstall runs — realpath-resolving it there would raise Errno::ENOENT. staged_path is Homebrew's own authoritative real-install-directory value (token+version, not a hand-matched prefix) and, per A1, is byte-identical to the directory the install-time realpath resolved into."
  - "Platform scope / livecheck / auto_updates / depends_on (03-CONTEXT.md discretion items) resolved by reading GoReleaser v2.17.1's actual pinned source rather than guessing: HomebrewCask has no os:/platform: field at all (the rendered on_linux block is a structural side effect of ids:[zip] covering all 4 platforms, not a deliberate cross-platform claim — no verification in this phase ever ran on Linux); cask.rb.tmpl unconditionally emits the livecheck skip stanza with no config lever; there is no auto_updates field or template output for it in this pinned version; HomebrewCaskDependency carries a literal `// TODO: support macos, arch` comment, so depends_on macos is genuinely unavailable, not declined."
  - "generate_completions_from_executable.executable is left UNSET deliberately — cask.go:78-79 auto-defaults it to binaries[0] (\"codegraph\") whenever the block is otherwise configured, avoiding a second, potentially-drifting spelling of the binary name already named in the binaries: list."

patterns-established:
  - "A single rehearsal invocation accumulates every new-behavior assertion failure via OK=0/continue rather than exiting on the first (mirrors release:dry-run-signed's existing idiom) — lets one RED run surface all four new Task-1 behavior failures simultaneously instead of requiring four separate runs."
  - "Rehearsal-copy line rewriting for a rehearsal-only fidelity gap: when a real, correct assertion fires against an artifact of the REHEARSAL mechanism itself (not the thing under test), the fix is to extend the established rewrite-and-diff-assert pattern (already used for the download URL) rather than weaken the assertion."

requirements-completed: [BREW-03, BREW-04, BREW-05]

coverage:
  - id: D1
    description: "hooks.post.install carries two positive assertions (man-page count > 1, version equality via v-stripped exact match) that raise and roll back a real brew install on failure — BREW-05's amended gate mechanism (D-09/D-10/D-11)"
    requirement: "BREW-05"
    verification:
      - kind: integration
        ref: "op run --env-file=.env -- env CASK_REHEARSE=1 task release:rehearse-cask (real signed+notarized darwin binaries, real brew install --cask against a throwaway local tap)"
        status: pass
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestHomebrewCaskHooksHaveStructuralProperties"
        status: pass
    human_judgment: false
  - id: D2
    description: "A deliberate version-mismatch perturbation proves the version-equality assertion actually fires: brew install exits non-zero with a message quoting both the real reported version and the mutated declared version; restored and re-ran clean afterward"
    requirement: "BREW-05"
    verification:
      - kind: manual_procedural
        ref: "release:rehearse-cask Step 7 — mutated 'version \"0.0.0-cask-rehearsal-mutated-version\"' against a real installed binary reporting 0.7.0; observed exit 1, message 'installed binary reports version \"0.7.0\", cask declares \"0.0.0-cask-rehearsal-mutated-version\"'; restore-and-reinstall observed exit 0"
        status: pass
    human_judgment: false
  - id: D3
    description: "The Phase-4 sentinel (.codegraph-brew-install) is written beside the real, symlink-resolved installed binary on install, and removed on uninstall (D-08/D-07)"
    requirement: "BREW-04"
    verification:
      - kind: integration
        ref: "release:rehearse-cask — sentinel present with all 6 keys after install (schema=1, cask_token, cask_version, homebrew_prefix, man_dir, man_page_count, installed_at), absent after uninstall"
        status: pass
    human_judgment: false
  - id: D4
    description: "Shell completions (bash/zsh/fish) generated from the installed binary via generate_completions_from_executable — BREW-03"
    requirement: "BREW-03"
    verification:
      - kind: integration
        ref: "release:rehearse-cask — all 3 completion files present and non-empty after install (completion_count=3); codegraph completion bash confirmed against the installed binary"
        status: pass
      - kind: unit
        ref: "internal/upgrade/goreleaser_shape_test.go#TestHomebrewCaskGeneratedCompletionsShellsIsExactSet"
        status: pass
    human_judgment: false
  - id: D5
    description: "Completions cannot substitute for the install gate — measured directly: a deliberately wrong completions executable name produces exit 0 with a warning (RESEARCH.md Pitfall 1)"
    requirement: "BREW-05"
    verification:
      - kind: manual_procedural
        ref: "release:rehearse-cask Step 7b — mutated generate_completions_from_executable \"codegraph-does-not-exist\"; observed exit 0, output: 'Failed to generate bash/zsh/fish completions from codegraph-does-not-exist ... exited with 127'"
        status: pass
    human_judgment: false
  - id: D6
    description: "7 new shape tests hold the cask block's non-obvious fields as properties, each demonstrated RED against a real, confirmed-applied, byte-cleanly-reverted mutation"
    requirement: "BREW-03, BREW-04, BREW-05"
    verification:
      - kind: unit
        ref: "go test ./internal/upgrade/... -run 'TestHomebrewCask|TestParseGoreleaserCask|TestParseGoreleaserArchivesRaw' -v"
        status: pass
      - kind: unit
        ref: "task test:unit"
        status: pass
    human_judgment: false

# Metrics
duration: "~2h (context loading + 4 full credentialed rehearsal cycles + 6 mutation-RED demonstrations)"
completed: 2026-08-10
status: complete
---

# Phase 3 Plan 2: Cask Completion Summary

**BREW-05's install gate is real: two positive assertions (man-page presence, version equality) that raise and roll back a genuine `brew install --cask`, both watched fire against deliberate mutations; the Phase-4 sentinel and symmetric uninstall cleanup are written and removed correctly; shell completions generate from the installed binary in exactly three shells; and every one of these is proven by a real, credentialed `release:rehearse-cask` PASS plus 7 new shape tests each demonstrated RED against a confirmed-applied, byte-cleanly-reverted mutation of `.goreleaser.yaml`.**

## Performance

- **Duration:** ~2h (deep context read across 03-01/03-03 SUMMARYs, CONTEXT.md, RESEARCH.md, the pinned GoReleaser v2.17.1 module source, and Homebrew 6.0.16's own Ruby cask internals on this host; 4 full credentialed `goreleaser release` cycles including real Apple notarization; 6 mutation-RED-then-revert cycles for the shape tests)
- **Completed:** 2026-08-10T18:04Z
- **Tasks:** 3 of 3 committed
- **Files modified:** 3 (`.goreleaser.yaml`, `Taskfile.yml`, `internal/upgrade/goreleaser_shape_test.go`)

## Accomplishments

- **`hooks.post.install` completed** with D-11's two positive assertions, in order: (1) the man directory contains more than one `codegraph*.1` page (no pinned count — the tree grows with new subcommands), raising and naming the directory/count if not; (2) the installed binary's `version --json` output equals the cask's own declared `version`, both normalized by stripping a single leading `v` and compared by equality (never containment). Both raise, and `Cask::Installer#install_artifacts` does not rescue that raise — it rolls back already-installed artifacts and re-raises, so `brew install` genuinely exits non-zero.
- **The Phase-4 sentinel** (`.codegraph-brew-install`, schema 1, six `key=value` lines) is written at `Pathname.new(binary).realpath.dirname` — a literal symlink resolution, never a path-prefix guess — confirmed on this machine to resolve to `/opt/homebrew/Caskroom/codegraph/<version>`.
- **`hooks.post.uninstall` added**, removing the man pages (`codegraph*.1` glob) and the sentinel via `FileUtils.rm_f` (force-style, explicit statements, no `&&` chain). The sentinel's removal location is `staged_path`, not a re-resolved symlink — confirmed from both GoReleaser's `cask.rb.tmpl` (stanzas render in declared order: `binary` before `hooks:`) and Homebrew's own `Cask::Installer#uninstall_artifacts` (artifacts uninstall in that same declared order) that the `#{HOMEBREW_PREFIX}/bin/codegraph` symlink is already gone by the time this hook runs.
- **`generate_completions_from_executable`** added: `shells: [bash, zsh, fish]`, `shell_parameter_format: cobra`, `executable:` deliberately unset (auto-defaults to `binaries[0]`). Confirmed `codegraph completion bash` works against the actual installed binary, not a locally built one.
- **Every discretionary metadata decision recorded with a source-traced reason**: platform scope (darwin-only claim; `HomebrewCask` has no os:/platform: field in this pinned version — the rendered `on_linux` block is a structural side effect of the shared `zip` archive filter, not a deliberate cross-platform claim), livecheck (GoReleaser's template unconditionally emits the skip stanza — nothing to configure), auto_updates (no such field exists in this pinned version), depends_on macos (`HomebrewCaskDependency` carries a literal `// TODO: support macos, arch` — genuinely unavailable, not declined).
- **`release:rehearse-cask` extended to schema=2**: sentinel presence + 6-key parse, completions count (hardened to a hard assertion in Task 2), reinstall idempotency, post-uninstall symmetry, a real version-mismatch perturbation, and a real completions-bad-executable perturbation — all observed PASS in the same single credentialed rehearsal invocation, reusing one built-and-notarized zip across every install/uninstall cycle (no rebuild, no re-notarization per perturbation).
- **Rehearsal-fidelity fix, discovered by the new assertion working correctly**: `--snapshot` mode's synthetic cask version (`0.7.0-SNAPSHOT-91b6df6`) permanently mismatched the ldflags-injected binary version (`0.7.0`) — a gap in the rehearsal target's own fidelity to a real tagged release, not a hook bug. Fixed by rewriting the rehearsal copy's declared `version "..."` line to the LOCAL build's actual reported version (read directly from the pre-archive build artifact, no install needed to learn it), extending the existing download-URL-rewrite pattern from one line to two.
- **7 new shape tests** in `internal/upgrade/goreleaser_shape_test.go`, each demonstrated RED against a real mutation, then restored byte-clean and re-verified GREEN (see Deviations/mutation log below).

## Task Commits

1. **Task 1: The gate — two positive assertions, the sentinel, and the removal that matches** — `47d85ac` (feat) — `.goreleaser.yaml`, `Taskfile.yml`.
2. **Task 2: Completions generated from the binary, and the cask metadata a real cask carries** — `db7fc99` (feat) — `.goreleaser.yaml`, `Taskfile.yml`.
3. **Task 3: Shape tests that hold the block's non-obvious fields as properties** — `02f60b3` (test) — `internal/upgrade/goreleaser_shape_test.go`.

**Plan metadata:** this commit (SUMMARY.md, superseding the halted record)

## Files Created/Modified

- `.goreleaser.yaml` — `homebrew_casks[0].hooks.post.install` completed (two assertions + sentinel), `hooks.post.uninstall` added, `generate_completions_from_executable` added, discretionary-metadata comments added
- `Taskfile.yml` — `release:rehearse-cask` extended to schema=2 (sentinel, completions, idempotency, post-uninstall, two perturbations); the rehearsal-copy rewrite extended from one line (URL) to two (URL + declared version)
- `internal/upgrade/goreleaser_shape_test.go` — 7 new test functions + 2 new raw-map parsers (`parseGoreleaserHomebrewCasks`, `parseGoreleaserArchivesRaw`) + a `countNonCommentRaiseStatements` helper

## Decisions Made

See `key-decisions` in frontmatter for the full, source-traced record. Summarized:

- **The halt this file previously recorded is SUPERSEDED, not silently erased.** See "Superseded Halt" below for the full record of what the predecessor session got right and wrong.
- **Sentinel write vs. removal use two different, both-correct location mechanisms** (symlink realpath at install time; `staged_path` at uninstall time) — deliberate, not inconsistent, because the symlink is confirmed gone by uninstall time (Homebrew's own artifact-uninstall ordering) while `staged_path` is Homebrew's authoritative value for the same directory at any time.
- **The version-equality assertion's very first real fire was against the rehearsal's own snapshot-mode artifact**, not a deliberate mutation — recorded as a Rule 1/3 deviation below, not smoothed over.
- **Every "Claude's Discretion" metadata item was resolved by reading GoReleaser's pinned source**, not by convention or guess — three of the four discretionary items (livecheck, auto_updates, depends_on macos) turned out to have NO config lever in this pinned version at all, which the summary states plainly rather than inventing a value to set.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1/3 — rehearsal-fidelity bug, not a hook bug] `--snapshot` mode's synthetic cask version permanently mismatched the ldflags-injected binary version**

- **Found during:** Task 1, the first GREEN attempt after completing `hooks.post.install`.
- **Issue:** The version-equality assertion — working exactly as designed — raised `codegraph cask post-install: installed binary reports version "0.7.0", cask declares "0.7.0-SNAPSHOT-91b6df6"` on the rehearsal's own normal install path. Root cause, confirmed by inspecting a real snapshot render: GoReleaser's `ctx.Version` (used for the cask's `version "..."` line) is a SNAPSHOT-suffixed pseudo-version under `--snapshot`, while the binary's ldflags (`-X .../version.Version={{ .Tag }}`) use the nearest real tag — two different values that are the SAME string only in a real, tagged (non-snapshot) release.
- **Fix:** Extended the rehearsal's existing "rewrite exactly the download URL line, diff-assert it" pattern to also rewrite the declared `version "..."` line to the value the LOCAL (pre-archive) build artifact actually reports — queried directly via `dist/artifacts.json`, no install needed. The diff-line-count assertion was updated from 2 (one line changed) to 4 (two lines changed) in both the primary and HTTP-fallback code paths.
- **Files modified:** `Taskfile.yml`
- **Verification:** Re-ran the full credentialed rehearsal; observed `REHEARSAL-DIFF` showing exactly the version and URL lines changed, then a clean install with `Cask codegraph (0.7.0)` matching the binary's `0.7.0`.
- **Committed in:** `47d85ac`

### None Beyond the Above

Every other behavior matched the plan's design exactly. No architectural changes, no scope additions beyond what BREW-03/04/05 name.

---

**Total deviations:** 1 auto-fixed (Rule 1/3, rehearsal-fidelity — the gate's own behavior is unchanged).
**Impact on plan:** None beyond the fix itself; the deviation is entirely contained to the rehearsal harness, not the shipped cask config.

## Superseded Halt (preserved, not erased)

This SUMMARY previously recorded a `status: halted` outcome from an earlier session. That record is preserved here rather than silently deleted, per the orchestrator's instruction:

**What the halted session got RIGHT:** its worktree genuinely had no `.env` — git worktrees do not contain gitignored, untracked files from the main checkout, so no worktree-scoped executor could ever reach the credentials that way. `task release:rehearse-cask` (run with no `op run` wrapper) genuinely exits non-zero immediately at its own `MACOS_SIGN_P12` precondition in that environment — a real, correctly-observed fact.

**What it got WRONG:** it additionally cited `op whoami` reporting "account is not signed in" as corroborating evidence that credentials were unreachable in general. `op whoami` tests *interactive account sign-in*, not `op run`'s non-interactive `op://` reference resolution — the two are unrelated capabilities. The orchestrator measured, on this same host, that `op run --env-file=.env -- ...` resolves real credential values successfully even while `op whoami` reports not signed in. This session re-verified that measurement independently before starting any task work (`op run --env-file=.env -- env CASK_REHEARSE=1 sh -c 'echo len=${#MACOS_SIGN_P12}; echo len2=${#MACOS_NOTARY_KEY}'` → `len=4352`, `len2=344`, no value ever printed) and ran on the main checkout (not a worktree) specifically so `.env` was reachable on disk.

**Resolution:** this session ran the full plan to completion using `op run --env-file=.env -- env CASK_REHEARSE=1 task release:rehearse-cask`, exactly the invocation the orchestrator had already proven working. `.planning/WINDOWS.md` entry 3 (the `unrun-verify` this halt filed) is now `status: fixed`. Entry 2 (03-03's unrelated browser-authentication blocker) remains open — this plan does not depend on it and does not resolve it.

## Issues Encountered

- One transient Apple notarization API timeout (`context deadline exceeded` waiting on a submission status poll) during an early rehearsal attempt — not a code or config issue; the very next invocation of the same command succeeded normally. Not a deviation, just recorded for anyone re-running this rehearsal and hitting the same transient network condition.

## Known Stubs

None. Every mechanism in this plan (the gate, the sentinel, the uninstall symmetry, the completions) is a real, exercised implementation, not a placeholder.

## Threat Flags

None beyond what the plan's own `<threat_model>` already covers (T-03-04, T-03-05, T-03-06 — all three explicitly mitigated by this plan's own design, and now also verified by a real credentialed rehearsal rather than only by static config review).

## User Setup Required

None new. The same five `MACOS_*` credentials 03-01 already documented (`docs/RELEASE.md`) are what this plan's rehearsals consumed, via the same `op run --env-file=.env` mechanism.

## Next Phase Readiness

- **Ready for 03-04/03-05 to build on a fully proven cask.** BREW-03, BREW-04, and BREW-05 are all complete and verified by a real, credentialed, end-to-end `brew install --cask` / `brew uninstall --cask` cycle plus 7 new shape tests.
- **Plan 03-03 remains halted** on its own, unrelated blocker (no authenticated browser session for GitHub App creation) — `.planning/WINDOWS.md` entry 2 stays open. This plan's completion does not unblock it; 03-03's tap-push credential work is independent of this plan's install-gate/completions/sentinel work.
- **Phase 4's consumer is ready to be written against:** the sentinel's location (real, symlink-resolved binary directory), format (schema=1, six `key=value` lines, fixed order), and cross-phase contract are all now real, tested artifacts rather than a plan-time design.

---
*Phase: 03-homebrew-tap-cask*
*Completed: 2026-08-10*

## Self-Check: PASSED

- FOUND: `hooks.post.install` two-assertion block in `.goreleaser.yaml` (`git show 47d85ac -- .goreleaser.yaml`)
- FOUND: `hooks.post.uninstall` block in `.goreleaser.yaml`
- FOUND: `generate_completions_from_executable` block in `.goreleaser.yaml` (`git show db7fc99 -- .goreleaser.yaml`)
- FOUND: 7 new test functions in `internal/upgrade/goreleaser_shape_test.go` (`git show 02f60b3 --stat`)
- FOUND: commit `47d85ac` in `git log --oneline --all`
- FOUND: commit `db7fc99` in `git log --oneline --all`
- FOUND: commit `02f60b3` in `git log --oneline --all`
- CONFIRMED: `task check:goreleaser` exits 0 against the final committed state
- CONFIRMED: `op run --env-file=.env -- env CASK_REHEARSE=1 task release:rehearse-cask` exits 0 against the final committed state (commit `02f60b3`), evidence line `CASK-REHEARSE-EVIDENCE schema=2 ... sentinel=present completion_count=3 ... outcome=pass`
- CONFIRMED: `go test ./internal/upgrade/... -run 'TestHomebrewCask|TestParseGoreleaserCask|TestParseGoreleaserArchivesRaw' -v` exits 0
- CONFIRMED: `task test:unit` exits 0
- CONFIRMED: no residual brew-managed `codegraph` cask or stray tap left on this machine
