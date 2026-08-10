---
phase: 03-homebrew-tap-cask
plan: 04
subsystem: infra
tags: [homebrew, cask, github-app, gatekeeper, documentation, requirements-amendment]

# Dependency graph
requires:
  - phase: 03-homebrew-tap-cask
    provides: "03-02's completed hooks.post.install gate (two positive assertions) and 03-03's tap-scoped GitHub App, its installation scope, and its credential-minting mechanism"
provides:
  - "03-EVIDENCE.md — two BREW-05 install-gate RED observations (a truncated/SIGKILLed binary; a real signed+notarized binary reporting a version the cask does not declare), each confirmed-applied and byte-clean-reverted, plus a BREW-02 credential-boundary refusal (403 'Resource not accessible by integration' against seanb4t/codegraph-go, with a positive control proving the same token wrote seanb4t/homebrew-tap first)"
  - "REQUIREMENTS.md BREW-05 and ROADMAP.md Phase 3 criterion 3, both amended after the proof to name hooks.post.install's two assertions instead of a nonexistent cask test: block"
  - "docs/RELEASE.md §4 — the shipped Homebrew guarantee, the measured man-path caveat with a working prefix-qualified man invocation, and three explicitly pending claims for plan 03-05 to close"
  - "docs/RELEASE-PROCEDURES.md §12 — the maintainer runbook for the tap: ownership, the tap App's key rotation, and failed-push recovery (marked as resting on a pipeline-ordering argument, not an observed failure)"
  - "README.md's single canonical Homebrew install line"
affects: [03-05]

# Actuals (#2632)
actuals:
  tokens: 9914
  tasks: 3
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "One-time, hand-driven mutation proof outside the Taskfile: since Taskfile.yml/.goreleaser.yaml were not in this plan's files_modified, both BREW-05 mutations reproduced release:rehearse-cask's own render+rewrite steps by hand against a real, credentialed goreleaser release, rather than extending the Taskfile target."
    - "A temporary, local-only, never-pushed git tag (created, used to drive one real signed+notarized build, then deleted) as the mechanism for getting a genuinely different reported version out of a second real build, when the mutation must be driven from the binary side rather than the cask side."
    - "Direct GitHub App JWT minting via openssl RS256 signing (no SDK/gh-extension dependency) to reproduce, outside a workflow run, the exact two-step App-JWT-to-installation-token exchange release.yml's create-github-app-token step performs in CI."

key-files:
  created:
    - .planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md
  modified:
    - .planning/REQUIREMENTS.md
    - .planning/ROADMAP.md
    - docs/RELEASE.md
    - docs/RELEASE-PROCEDURES.md
    - README.md

key-decisions:
  - "Mutation 1 (a binary that cannot execute) needed no real Apple credentials: the corrupted (truncated-to-1024-bytes) entry retains enough Mach-O header to be quarantined and SIGKILLed by Gatekeeper regardless of the ORIGINAL binary's real signing status, so the observed failure (uncaught SIGKILL inside assertion one's system_command call) is genuine and did not require a fresh signed build — Run A's already-real zip was corrupted directly."
  - "Mutation 2 (a binary that runs and reports the wrong version) required a SECOND real, credentialed goreleaser release (Run B) under a temporary local git tag, because the mutation must be driven from the binary side (03-02 already drove it from the cask side) and a genuinely-signed binary is the only way to survive Homebrew Cask's unconditional quarantine long enough to reach assertion two. The temp tag was deleted immediately after the build captured what it needed, confirmed via both an absence check and git describe resolving back to v0.7.0."
  - "The App-JWT-to-installation-token exchange for the BREW-02 credential probe was reproduced directly against GitHub's REST API (openssl RS256 signing, no external JWT library) rather than by adding a scratch workflow run, since the plan explicitly forbids modifying release.yml/Taskfile.yml in this plan's scope and the exchange itself needed no CI context to demonstrate."
  - "Task 1 and Task 2's evidence content were drafted into a single Write call and therefore landed in one commit rather than two, since both were being composed in the same editing pass before either was staged. Task 3's docs changes remained a fully separate, later commit as planned. Recorded here as a process deviation from strict one-commit-per-task granularity, not a content or correctness gap — both tasks' acceptance criteria are independently satisfied by the committed content."
  - "REQUIREMENTS.md's 'Three scoping assumptions were falsified... before these requirements were written' framing paragraph was deliberately left untouched: that tally counts falsifications discovered BEFORE the requirements were authored (see PROJECT.md), a different and fixed-count record from SIGN-02's and now BREW-05's post-hoc 'Amended <date> (plan/D-ID)' inline pattern, which is the correct analog and the one this plan's amendment follows. No live-amendment counter exists elsewhere in the file to reconcile; none was invented."

patterns-established:
  - "For a plan whose files_modified excludes the Taskfile/GoReleaser config a rehearsal target lives in, reproduce that target's own render-and-rewrite steps by hand in a scratch directory rather than temporarily editing the excluded files and reverting — keeps the mutation entirely outside any file this plan is not authorized to touch."

requirements-completed: [BREW-05]

coverage:
  - id: D1
    description: "BREW-05's install gate demonstrated RED for a binary that cannot execute at all (truncated Mach-O, real quarantine, real SIGKILL) — confirmed-applied via sha256 pairs, non-zero exit, and a post-failure bin-path check showing no codegraph remained"
    requirement: "BREW-05"
    verification:
      - kind: manual_procedural
        ref: "brew install --cask local-rehearsal/cask-mutation1/codegraph against a hand-corrupted real signed+notarized zip; exit 1, 'terminated by uncaught signal KILL' inside assertion one's system_command call"
        status: pass
    human_judgment: false
  - id: D2
    description: "BREW-05's install gate demonstrated RED for a binary that runs and reports the wrong version, driven from the binary side (a second real signed+notarized build under a temporary local tag) rather than the cask side 03-02 already exercised"
    requirement: "BREW-05"
    verification:
      - kind: manual_procedural
        ref: "brew install --cask local-rehearsal/cask-mutation2/codegraph; exit 1, message quoting both 'installed binary reports version \"0.0.0-cask-mutation2-binary-version\"' and 'cask declares \"0.7.0\"'"
        status: pass
    human_judgment: false
  - id: D3
    description: "BREW-02's credential boundary demonstrated with a positive control (a real, reverted write to seanb4t/homebrew-tap) and a negative proof (the same token refused a write to seanb4t/codegraph-go with a resource-scope 403, not a credential-validity 401)"
    requirement: "BREW-02"
    verification:
      - kind: manual_procedural
        ref: "POST .../homebrew-tap/git/refs -> 201, DELETE -> 204, repo confirmed back to LICENSE+README only; POST .../codegraph-go/git/refs -> 403 'Resource not accessible by integration'"
        status: pass
    human_judgment: false
  - id: D4
    description: "REQUIREMENTS.md BREW-05 and ROADMAP.md Phase 3 criterion 3 amended after the proof, naming hooks.post.install's two assertions instead of a cask test: block Homebrew Casks do not have"
    requirement: "BREW-05"
    verification:
      - kind: unit
        ref: "git diff .planning/ROADMAP.md shows only criterion 3's line changed, criterion 4 byte-unchanged, no heading added/removed/altered"
        status: pass
      - kind: manual_procedural
        ref: "getMilestonePhaseFilter() output identical before and after the edit: [01-cross-compile-spike-goreleaser-release-migration, 02-apple-signing-notarization, 03-homebrew-tap-cask]"
        status: pass
    human_judgment: false

# Metrics
duration: "~2h (two real credentialed goreleaser release cycles including Apple notarization, real GitHub App JWT/token minting and boundary probing, three documentation-file additions)"
completed: 2026-08-10
status: complete
---

# Phase 3 Plan 4: Prove the Gate, Correct the Record Summary

**The install gate has been watched fail for both of the reasons D-11 built it to catch — a binary that cannot execute, and a binary that executes and reports the wrong version — each against a real, credentialed `brew install --cask`, confirmed-applied and byte-clean-reverted. The tap credential's write scope has been watched refuse a write to `seanb4t/codegraph-go` with a resource-scope `403`, after the same token first succeeded (and was reverted) against `seanb4t/homebrew-tap`. Both planning artifacts that named a cask `test:` block — a stanza Homebrew Casks do not have — now name the mechanism that exists, amended after and citing the proof.**

## Performance

- **Duration:** ~2h (two real, credentialed `goreleaser release` cycles including genuine Apple notarization; a real GitHub App JWT mint and installation-token exchange plus a real write/revert/refuse probe against two live repositories; three documentation files extended)
- **Completed:** 2026-08-10
- **Tasks:** 3 of 3 complete
- **Files modified:** 6 total (1 created — `03-EVIDENCE.md`; 5 modified — `REQUIREMENTS.md`, `ROADMAP.md`, `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`, `README.md`)

## Accomplishments

- **Mutation 1 (a binary that cannot execute).** A real, Developer-ID-signed and Apple-notarized darwin/arm64 zip (`Run A`, real `goreleaser release --snapshot --skip=publish,sign --clean` with real credentials) had its `codegraph` entry replaced with the first 1024 bytes of itself — a truncated, non-loadable file that `file` still reports as `Mach-O 64-bit executable arm64` (the header survives; the loadable content does not). Rezipped, checksum updated so Homebrew's own download verification is satisfied and the failure is genuinely the post-install hook, then installed via a throwaway local git tap: `brew install --cask` exited `1`, with `Error: Failure while executing; ... man ... was terminated by uncaught signal KILL` — the raise originates at assertion one's own `system_command` call site, before assertion two is ever reached. Post-failure: no `codegraph` on the bin path, no Caskroom directory, `brew list --cask codegraph` exits `1`.
- **Mutation 2 (a binary that runs and is the wrong artifact), driven from the binary side.** Plan 03-02 already perturbed this relationship from the cask side (editing the declared version against a real installed binary); this mutation is its mirror. A temporary, local-only git tag (`v0.0.0-cask-mutation2-binary-version`) was created at `HEAD`, a second real signed+notarized `goreleaser release` run (`Run B`) built a genuinely working darwin/arm64 binary reporting that bogus version, and the tag was deleted immediately after (confirmed absent, `git describe` resolved back to `v0.7.0`). A cask declaring the CORRECT version (`0.7.0`, from Run A) was repointed only at Run B's zip. Install exited `1` with `Error: codegraph cask post-install: installed binary reports version "0.0.0-cask-mutation2-binary-version", cask declares "0.7.0"` — quoting both values, exactly D-11's assertion two. Post-failure: no `codegraph` on the bin path.
- **BREW-02's credential boundary, with a positive control.** Minted the tap-scoped GitHub App's installation token directly against GitHub's REST API (App id `4549710`, installation id `152719025`, same App-JWT-then-installation-token exchange `release.yml` performs in CI). A throwaway branch ref created on `seanb4t/homebrew-tap` succeeded (`201`), was deleted (`204`), and the tap repository was confirmed to still contain only what plan 03-03 seeded (`LICENSE`, `README.md`, one branch). The same token's write attempt against `seanb4t/codegraph-go` was refused with `403 "Resource not accessible by integration"` — a resource-scope refusal, unambiguously distinct from a `401` credential-validity failure, so no second disambiguating call was needed. The App's private key was fetched from 1Password to a path outside the repo, used only there, and shredded (`rm -P`) immediately after minting the token; no key or token material was ever printed to any transcript or file that survives.
- **REQUIREMENTS.md BREW-05 and ROADMAP.md criterion 3 amended, after and citing the proof.** Both previously named a cask `test:` block. Measured, not assumed, that this is unachievable: `Cask::DSL.instance_methods(false)` on Homebrew 6.0.16 has no `test` method, `brew test --help` states it operates on `installed_formula` only, and the pinned GoReleaser v2.17.1 `HomebrewCask` struct (`pkg/config/config.go`) has no such field. Both artifacts now name `hooks.post.install`'s two positive assertions instead; the claim itself — a broken cask fails before a user hits it — is unchanged. `git diff .planning/ROADMAP.md` shows only criterion 3's line touched, criterion 4 byte-identical, no heading added/removed/altered anywhere in the file. The milestone's phase filter was captured before and after the edit and is identical: `["01-cross-compile-spike-goreleaser-release-migration","02-apple-signing-notarization","03-homebrew-tap-cask"]`.
- **`docs/RELEASE.md` §4 — the Homebrew install path, published.** States what the cask installs (binary, completions, man pages — the latter two generated from the installed binary at install time, not shipped as files), what the install gate does on the user's behalf, and the man-path dependency as a measured fact: on this real Apple Silicon Mac, `/etc/manpaths.d/` is empty and `path_helper -s`'s own `MANPATH` output omits `/opt/homebrew/share/man` entirely — genuinely absent from the system-level man path configuration, not merely sometimes missing. `man codegraph` in a shell with no Homebrew environment sourced returns `No manual entry for codegraph` even with the page correctly on disk (reproduced with `env -i` on this machine); the prefix-qualified escape hatch `man "$(brew --prefix)/share/man/man1/codegraph.1"` was run on this same machine and correctly rendered the page, bypassing man-path resolution entirely. States plainly that `codegraph upgrade`'s brew-detection is Phase 4's work and is not yet shipped — use `brew upgrade codegraph` today. Exactly three claims are marked pending in the document's existing status-note voice: the tap resolving, a cold install succeeding, and completion working in all three shells through a real brew-installed binary — none asserted as already true.
- **`docs/RELEASE-PROCEDURES.md` §12 — the maintainer runbook.** Names the tap (`seanb4t/homebrew-tap`) and that GoReleaser's `homebrew_casks:` pipe owns `Casks/codegraph.rb` end to end — never hand-edited. Names the publishing App (`seanb4t homebrew tap publishing`, id `4549710`) as deliberately distinct from the release-please App, and why that separation is the entire content of BREW-02/criterion 5. Gives the key-rotation procedure (generate, reseed, confirm via `GET /app`, delete the old key, shred local copies) mirroring §9's existing shape for the release-please App. Gives failed-tap-push recovery in terms of what is already true: `cask.Pipe{}` is registered last in GoReleaser's publish pipeline, so a failed tap push happens strictly after the release itself (raw binaries, `.zip`s, checksums, signs, SBOMs, attestation) is already complete and independently re-verifiable — remedy is fix-the-credential-and-rerun, never delete or re-push a release or tag. States its own basis explicitly: this rests on the pipeline-ordering argument, not an observed failure, and points to plan 03-05's evidence (BREW-06) for the argument in full.
- **`README.md`** carries the single canonical `brew tap seanb4t/tap && brew install codegraph` line, confirmed the only occurrence of that exact string in any user-facing document in the repository (`rg` over the working tree, excluding `graphify-out/`'s generated cache and `.planning/research/`'s planning artifact).

## Task Commits

1. **Task 1 + Task 2: watch the install gate fail twice; refuse the tap credential once** — `556ca14` (docs) — `.planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md`, `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`. Both tasks' evidence content was drafted in a single `Write` call before either was staged, so both landed in one commit rather than the two the plan's `<files>` blocks implied — see Deviations below.
2. **Task 3: publish the install path** — `5777eeb` (docs) — `README.md`, `docs/RELEASE.md`, `docs/RELEASE-PROCEDURES.md`.

**Plan metadata:** this commit (SUMMARY.md)

## Files Created/Modified

- `.planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md` — new; both BREW-05 RED observations, the BREW-02 refusal record with its positive control, and both accepted-limitation statements
- `.planning/REQUIREMENTS.md` — BREW-05 amended (mechanism corrected, dated note added, claim strength unchanged)
- `.planning/ROADMAP.md` — Phase 3 criterion 3 amended identically; criterion 4 and every heading byte-unchanged
- `docs/RELEASE.md` — new §4 "Installing via Homebrew (macOS)"
- `docs/RELEASE-PROCEDURES.md` — new §12 "The Homebrew tap (maintainer side)"
- `README.md` — new "Homebrew (macOS)" subsection under Install

## Decisions Made

See `key-decisions` in the frontmatter for the full record. Summarized:

- Mutation 1 needed no fresh Apple credentials (a truncated real zip is sufficient); mutation 2 needed a second real signed+notarized build under a temporary, immediately-deleted local git tag, because it had to be driven from the binary side.
- The BREW-02 credential exchange was reproduced directly against GitHub's REST API with hand-rolled RS256 JWT signing (openssl), not via a scratch CI workflow, since `release.yml`/`Taskfile.yml` were out of this plan's `files_modified` scope.
- Task 1 and Task 2's evidence landed in a single commit rather than two — a process deviation from strict one-task-one-commit granularity, not a correctness gap; both tasks' acceptance criteria are independently satisfied in the committed content.
- REQUIREMENTS.md's pre-writing "Three scoping assumptions" tally was deliberately left untouched — it is a different, fixed-count record from the post-hoc "Amended <date>" pattern SIGN-02 established and BREW-05 now follows.

## Deviations from Plan

### Process deviation (not a Rule 1-4 fix — recorded for commit-granularity honesty)

**1. Task 1 and Task 2's evidence content committed together, not separately**
- **Found during:** preparing to stage Task 1's commit
- **Issue:** the plan's `<files>` blocks list Task 1 as touching `03-EVIDENCE.md`/`REQUIREMENTS.md`/`ROADMAP.md` and Task 2 as appending to `03-EVIDENCE.md` alone, implying two commits. Both tasks' evidence sections were drafted into the same `Write` call before either was staged, so they landed in one commit (`556ca14`).
- **Impact:** none on correctness or on any acceptance criterion — both tasks' required content is present, correctly attributed to its own section of `03-EVIDENCE.md`, and independently verifiable. Recorded here so the commit history is read accurately rather than assumed to match the plan's task boundaries 1:1.
- **Files affected:** `.planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md`, `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`
- **Committed in:** `556ca14`

### None Beyond the Above

No Rule 1 (bug), Rule 2 (missing critical functionality), or Rule 3 (blocking-issue) auto-fixes were needed. No Rule 4 (architectural) decision was required — the plan's own design was followed exactly, including its explicit prohibition on inventing structure in `ROADMAP.md`/`REQUIREMENTS.md`.

**Total deviations:** 1 (process/commit-granularity only, no content or correctness impact).
**Impact on plan:** none — every task's acceptance criteria are independently satisfied by the committed content regardless of which commit boundary they fell on.

## Issues Encountered

None beyond the recorded process deviation above. Both real `goreleaser release` cycles, both real `brew install --cask` mutation attempts, and the real GitHub App credential probe all completed on the first attempt with no retries needed.

## Known Stubs

None. Every mechanism this plan touches — the two mutation proofs, the credential probe, the three documentation additions — reflects a real, executed observation or a value edit to existing text, not a placeholder.

## Threat Flags

None beyond what the plan's own `<threat_model>` already covers (T-03-11, T-03-12 — both explicitly mitigated: the amendment was written after the proof and cites it, with a before/after diff confirming no heading changed and criterion 4 stayed byte-identical; the credential refusal was recorded only after a positive control proved the same token could write the tap).

## User Setup Required

None new. The five `MACOS_*` signing/notarization credentials and the `HOMEBREW_TAP_APP_ID`/`HOMEBREW_TAP_APP_PRIVATE_KEY` secrets this plan's proofs depended on were already seeded by plans 03-01/03-02/03-03; this plan consumed them via `op run --env-file=.env` and a direct 1Password document fetch (immediately shredded), adding no new maintainer setup.

## Next Phase Readiness

- **Ready for plan 03-05.** BREW-05's install gate and BREW-02's credential boundary are now both recorded observations, not arguments from source-reading alone. `docs/RELEASE.md`'s three pending markers (tap resolves, cold install succeeds, completion works in all three shells) are enumerated precisely for plan 03-05 to close with real, executed evidence against a real published release. `docs/RELEASE-PROCEDURES.md`'s failed-push recovery text explicitly names plan 03-05's evidence (BREW-06) as where its pipeline-ordering argument gets whatever executed evidence that plan adds.
- `.planning/WINDOWS.md` remains at `open_count: 0` — this plan introduced no stubs, skipped tests, or unrun verifications requiring a ledger entry.
- `task test:unit` and both plan-specified verify commands (`go test ./internal/upgrade/... -run TestHomebrewCask`, `go test ./internal/cli/... -run TestFlagParityDocCoversRegisteredFlags`) all exit 0 on the current `HEAD`.

---
*Phase: 03-homebrew-tap-cask*
*Completed: 2026-08-10*

## Self-Check: PASSED

- FOUND: `.planning/phases/03-homebrew-tap-cask/03-EVIDENCE.md`
- FOUND: BREW-05 amendment in `.planning/REQUIREMENTS.md` (`git show 556ca14 -- .planning/REQUIREMENTS.md`)
- FOUND: criterion 3 amendment in `.planning/ROADMAP.md` (`git show 556ca14 -- .planning/ROADMAP.md`)
- FOUND: "Homebrew (macOS)" section in `README.md` (`git show 5777eeb -- README.md`)
- FOUND: "## 4. Installing via Homebrew (macOS)" in `docs/RELEASE.md` (`git show 5777eeb -- docs/RELEASE.md`)
- FOUND: "## 12. The Homebrew tap (maintainer side)" in `docs/RELEASE-PROCEDURES.md` (`git show 5777eeb -- docs/RELEASE-PROCEDURES.md`)
- FOUND: commit `556ca14` in `git log --oneline -5`
- FOUND: commit `5777eeb` in `git log --oneline -5`
- CONFIRMED: `go test ./internal/upgrade/... -run TestHomebrewCask -v` exits 0
- CONFIRMED: `go test ./internal/cli/... -run TestFlagParityDocCoversRegisteredFlags` exits 0
- CONFIRMED: `task test:unit` exits 0
- CONFIRMED: no residual brew-managed `codegraph` cask or stray `local-rehearsal` tap on this machine
- CONFIRMED: no leftover mutation git tags (`git tag -l | grep -c mutation` → 0)
