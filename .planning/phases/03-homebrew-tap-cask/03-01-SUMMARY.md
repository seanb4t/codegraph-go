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
  - "internal/cli/man.go — the hidden `codegraph man <dir>` command, directory-creating (Task 3) with a five-behavior test contract in internal/cli/man_test.go"
  - "internal/cli/root.go registration of newManCmd()"
  - "A minimal, production-shaped `homebrew_casks:` block in .goreleaser.yaml"
  - "Taskfile.yml `release:rehearse-cask` — the maintainer-only local cask-install rehearsal target, RED-then-GREEN demonstrated end to end, including a real Gatekeeper-cleared `brew install --cask`"
  - "A measured, confirmed real-world finding: Homebrew Cask unconditionally quarantines every download, so this rehearsal (and by extension the shipped cask's post-install hook) categorically requires a genuinely Developer-ID-signed and notarized binary — not merely a config concern, a Gatekeeper enforcement fact. This finding is load-bearing for plan 03-02's cask expansion and any future local cask rehearsal in this repository."
  - "docs/FLAG-PARITY.md — the `man` Go-only divergence entry, recording Hidden: true as a deliberate difference from githooks (D-02)"
affects: [03-02, 03-03, 03-04, 03-05]

# Actuals (#2632)
actuals:
  tokens: 11194
  tasks: 2
  commits: 4

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
  - "Measured man page count: 30 (not CONTEXT.md's ~27 estimate, not RESEARCH.md's traced 35). Root cause identified with certainty: doc.GenManTree(newRootCmd(), ...) never calls Execute() on the constructed tree, so Cobra's InitDefaultCompletionCmd (called only from command.go:1113, inside Execute()'s preparation path) never fires — the completion command and its 4 shell subcommands (5 pages) are absent from the generated tree, accounting for the 35-vs-30 gap exactly. Re-measured after Task 3's directory-creation change: still 30 — the fix only adds os.MkdirAll before generation, it does not touch the tree walked."
  - "Task 3 TDD measurement, recorded as observed rather than as the plan predicted: of the five behaviors in the test contract, only directory-creation (`TestManCmd_CreatesMissingDirectory_AndWritesFullTree`) was genuinely RED against Task 2's committed implementation. The unwritable-path-error behavior (`TestManCmd_UnwritablePath_ReturnsNonNilErrorNamingPath`) passed immediately — Task 2's RunE already wrapped doc.GenManTree's error with the target directory via `fmt.Errorf(\"generate man pages into %s: %w\", dir, err)`, which already satisfies \"a non-nil error whose message contains the offending path.\" The plan's action text predicted both would require implementation changes; only one did. Recorded as a measured correction to the plan's prediction, not silently absorbed."
  - "requirements-completed scoped to [BREW-04] only, not BREW-05, even though the plan's own frontmatter lists both as this plan's requirements. BREW-04 (\"The cask installs man pages\") is proven end-to-end by the orchestrator's op-run rehearsal (CASK-REHEARSE-EVIDENCE outcome=pass — a real brew install --cask executing the installed binary's man subcommand, codegraph.1 landing on disk) and constrained by Task 3's five-behavior test contract. BREW-05 (\"The cask carries a test: block exercising a real command\") is NOT delivered by this plan — no test: block exists in the homebrew_casks: block this plan wrote — and 03-02-PLAN.md's own text names BREW-05's install-gate mechanism as its own deliverable (\"Turn the proven slice into the real cask: the install gate that BREW-05 actually...\")."

patterns-established:
  - "Maintainer-only local rehearsal targets that install real artifacts requiring Gatekeeper-cleared binaries MUST declare the same MACOS_SIGN_P12/MACOS_SIGN_PASSWORD/MACOS_NOTARY_ISSUER_ID/MACOS_NOTARY_KEY_ID/MACOS_NOTARY_KEY preconditions release:rehearse-notarize already uses — Homebrew Cask's unconditional quarantine (extend/os/mac/cask/quarantine.rb) makes this load-bearing, not optional, for ANY target that executes an installed cask binary."

requirements-completed: [BREW-04]

coverage:
  - id: D1
    description: "Hidden `codegraph man <dir>` Cobra command generating the full man-page tree, registered on newRootCmd(), creating its own target directory (including missing parents), and surfacing a non-nil error naming the offending path on write failure"
    requirement: "BREW-04"
    verification:
      - kind: unit
        ref: "go build ./... "
        status: pass
      - kind: integration
        ref: "go run ./cmd/codegraph man <dir> (manual, dir pre-created) — measured 30 files"
        status: pass
      - kind: unit
        ref: "go test ./internal/cli/... -run TestManCmd -v — 5/5 behaviors pass (directory creation incl. missing parents, unwritable-path error naming the path, full command-tree write incl. codegraph.1, Hidden field read from newRootCmd().Commands(), exact-one-arg)"
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
        ref: "op run --env-file=.env -- env CASK_REHEARSE=1 task release:rehearse-cask (orchestrator run, real Apple Developer ID credentials loaded from a gitignored .env of op:// references)"
        status: pass
    human_judgment: false
    rationale: "Executed and observed PASS. Verbatim evidence line: `CASK-REHEARSE-EVIDENCE schema=1 snapshot_version=unknown cask_sha256=2a9ddeca59a8d318a70d547671449e8028c3b6431db3846ca080ac465f4a7199 url_mechanism=file man_page_count=30 reported_version=codegraph v0.7.0 (commit 20b90fc98891ecb97aaccc920315c7ec37881b56, built 2026-08-10T14:37:44Z) go1.26.5 darwin/arm64 outcome=pass` / `release:rehearse-cask: PASS`. Observed in order: both darwin binaries genuinely signed AND notarized; cask rendered to dist/homebrew/Casks/codegraph.rb; the rehearsal diff was exactly one line (12c12, the release URL replaced by a file:// URL); brew tap of the throwaway local repo; brew install --cask succeeded; \"Linking Binary 'codegraph' to '/opt/homebrew/bin/codegraph'\"; the post-install hook executed the installed binary; 30 man pages landed; brew uninstall --cask and brew untap both clean. A1 CONFIRMED (binary sits at the zip root — linking succeeded with the binaries: block unchanged, no rename handling needed). A3 CONFIRMED (#{HOMEBREW_PREFIX}/bin/codegraph is a live symlink by the time the post-install hook runs). D-10 CONFIRMED (the install genuinely is the gate — the hook executes as part of brew install). Two caveats carried forward, not glossed over: (1) `snapshot_version=unknown` in the evidence line — that field did not populate, though the version is present in `reported_version`; recorded as a known gap in the evidence schema, not silently fixed (Taskfile.yml is owned by the concurrent 03-03 executor in this session and was not touched). (2) Homebrew printed \"Warning: You are using macOS 27. We do not provide support for this pre-release version.\" — this evidence comes from a pre-release macOS host, stated plainly."

# Metrics
duration: "~70min (original halted session) + ~20min (resumption session: blocker-clearing evidence review, Task 3 TDD, SUMMARY/WINDOWS closeout)"
completed: 2026-08-10
status: complete
---

# Phase 3 Plan 1: Homebrew Man-Page Tracer Summary

**`codegraph man`, a minimal `homebrew_casks:` block, and the man command's five-behavior test contract are landed, and the tracer's own end-to-end proof — a real `brew install --cask` executing the installed binary via a genuinely signed-and-notarized build — has been run and observed PASS (`CASK-REHEARSE-EVIDENCE ... outcome=pass`). Both A1 and A3 are CONFIRMED. The whole Homebrew delivery path is proven; plan 03-02 builds on a proven slice, not a designed one.**

## Performance

- **Duration:** ~70 min (original session) + ~20 min (resumption session)
- **Completed:** 2026-08-10T14:33:11Z (Task 2, original session); resumption session completed same day
- **Tasks:** 3 of 3 committed (Task 1 checkpoint resolved pre-dispatch; Task 2 implemented and, per the orchestrator's credentialed rehearsal run, its tracer `<verify>` fully passed; Task 3 executed this resumption session per the cleared tracer feedback gate)
- **Files modified:** 8 total across both sessions (1 created, 7 modified: internal/cli/root.go, .goreleaser.yaml, Taskfile.yml, go.mod, go.sum, internal/cli/man.go, docs/FLAG-PARITY.md) plus internal/cli/man_test.go created this session

## Accomplishments

- `internal/cli/man.go` — the hidden `codegraph man <dir>` command (`Hidden: true`, `Args: cobra.ExactArgs(1)`, `doc.GenManTree` against a fresh `newRootCmd()`), registered in `newRootCmd()`'s `AddCommand` list and documented in both the package doc comment and `newRootCmd`'s own comment.
- `go-md2man/v2` and `blackfriday/v2` promoted from module-graph-only entries to compiled dependencies (`go build -mod=mod ./...`, since `go mod tidy` fails module-wide on an unrelated pre-existing `internal/parser/cgo` issue). Source-mode `govulncheck ./...` over the main module: **CLEAN, 0 symbol-reachable vulnerabilities** (1 vulnerability exists in an imported-but-not-called package, 1 in a required-but-not-imported module — neither reachable).
- A minimal, production-shaped `homebrew_casks:` block in `.goreleaser.yaml`: `ids: [zip]` (avoids `ErrMultipleArchivesSameOS`), no `url:` key (Pattern 2), `repository:` wired to `seanb4t/homebrew-tap` with `HOMEBREW_TAP_TOKEN`, and `hooks.post.install` invoking `codegraph man`. The `archives:` block is byte-unchanged (confirmed via diff against the pre-plan commit — additions only, zero deletions).
- `Taskfile.yml`'s new `release:rehearse-cask` target: demonstrated **RED** before any implementation existed (failed by naming `dist/homebrew/Casks/codegraph.rb does not exist or is empty` — never a crash), then demonstrated **partial GREEN** through the render, the exactly-one-line URL rewrite (verified via diff assertion), and a real `brew install --cask` of a throwaway local-tap-wrapped copy — the last step (a modern-Homebrew requirement neither RESEARCH.md nor CONTEXT.md anticipated) that itself required discovering and implementing a local-git-tap wrapper this session.
- `task release:dry-run` and `task release:dry-run-signed` both still exit 0 with the new `homebrew_casks:` block present, and the "homebrew cask" pipe now runs inside both (confirmed in their own log output).
- Measured man page count: **30** (neither CONTEXT.md's ~27 estimate nor RESEARCH.md's traced 35). Root-caused with certainty: `doc.GenManTree(newRootCmd(), ...)` never calls `.Execute()` on the tree it walks, so Cobra's `InitDefaultCompletionCmd` (fired only from `command.go:1113`, inside `Execute()`'s own preparation path) never runs — the `completion` command and its 4 shell subcommands (5 pages) are absent, which is exactly the 35-vs-30 gap. Re-measured after Task 3's directory-creation change: still 30.
- **The tracer's own `<verify>` reached PASS.** The orchestrator ran `op run --env-file=.env -- env CASK_REHEARSE=1 task release:rehearse-cask` — the five `MACOS_*` credentials this plan's original session found totally absent from the environment were, in fact, one `op run` away via `op://` references in a gitignored `.env`. Verbatim evidence line and the full observed sequence are recorded in the `D3` coverage entry above. A1 and A3 are both CONFIRMED, not merely no-longer-blocked.
- `internal/cli/man_test.go` — the five-behavior test contract (Task 3): directory creation including missing parents, a non-nil error naming the offending path on an unwritable write, the full command-tree write (not just the root page), the `Hidden` field read from the command object, and exact-one-argument validation. `internal/cli/man.go` gained `os.MkdirAll(dir, 0o755)` before `doc.GenManTree` — the one implementation change the test contract required (see key-decisions for the measured correction to the plan's prediction that a second behavior also needed a fix).
- `docs/FLAG-PARITY.md` gained the `man` Go-only section, its closing-summary entry, and the sentence naming `man`'s deliberate `Hidden: true` divergence from the `githooks` precedent (D-02) — keeping the document's own closing claim ("every command registered in `newRootCmd()` has a section above") true.

## Task Commits

1. **Task 1: Confirm the tap name and cask token** — resolved pre-dispatch by the orchestrator (option `confirm-d14`); not re-asked, no commit of its own (decision already recorded in CONTEXT.md D-14).
2. **Task 2: One path, end to end** — `8646f19` (feat) — man.go, root.go, .goreleaser.yaml, Taskfile.yml, go.mod, go.sum. Committed as real, correct, tested progress; the task's own `<verify>` (the tracer feedback gate) is now confirmed PASS by the orchestrator's credentialed rehearsal run (see D3 coverage above).
3. **Task 3: The man command's behavior contract** — three commits, this resumption session, per the plan's own RED→GREEN TDD contract:
   - `bbab820` (test) — `internal/cli/man_test.go`, all five behaviors; genuine RED on `TestManCmd_CreatesMissingDirectory_AndWritesFullTree` (quoted in Issues Encountered), the other four passed immediately against Task 2's committed implementation.
   - `3d07cf9` (feat) — `internal/cli/man.go`: `os.MkdirAll(dir, 0o755)` before `doc.GenManTree`, turning the one RED test GREEN.
   - `208b385` (docs) — `docs/FLAG-PARITY.md`: the `man` Go-only section, divergence summary entry, and Hidden-vs-githooks sentence.

**Plan metadata:** (this commit, SUMMARY.md + WINDOWS.md)

## Files Created/Modified

- `internal/cli/man.go` — hidden `codegraph man <dir>` command, now directory-creating (Task 3)
- `internal/cli/man_test.go` — five-behavior test contract (Task 3, new file)
- `internal/cli/root.go` — registers `newManCmd()`, updates two doc comments
- `.goreleaser.yaml` — new `homebrew_casks:` top-level block (additions only, `archives:` byte-unchanged)
- `Taskfile.yml` — new `release:rehearse-cask` target (~215 lines, including the 5 Apple-credential preconditions added mid-session once their necessity was measured)
- `go.mod` / `go.sum` — `go-md2man/v2 v2.0.7`, `blackfriday/v2 v2.1.0` promoted to compiled/full-hash entries
- `docs/FLAG-PARITY.md` — new `man` Go-only divergence section (Task 3)

## Decisions Made

- **Task 1 resolution recorded, not re-asked.** Decision id D-14, option `confirm-d14`: tap `seanb4t/homebrew-tap`, cask token `codegraph`, contract `brew tap seanb4t/tap && brew install codegraph`. Answered by the maintainer at orchestration time, pre-dispatch.
- **`go mod tidy` bypassed in favor of `go build -mod=mod ./...`.** The former walks every package's full transitive (including test) dependency graph and fails on `internal/parser/cgo`'s unrelated `tree-sitter-swift` binding resolution issue — a pre-existing condition, out of this plan's scope per the deviation-rule scope boundary. `-mod=mod` scopes resolution to only what `go build ./...` actually needs, producing the identical, minimal two-line `go.mod` diff.
- **`release:rehearse-cask` wraps the cask in a throwaway local git-repo tap.** Modern Homebrew (6.0.16, auto-updated mid-session by `brew install --cask` itself before `HOMEBREW_NO_AUTO_UPDATE=1` was added) refuses `brew install --cask <path>.rb` with "Homebrew requires casks to be in a tap" — a real, measured environment finding, not present in RESEARCH.md's trace (RESEARCH.md's source reads predate whatever Homebrew release introduced this restriction). The rehearsal creates a throwaway git repo under a run-scoped temp dir, `git init`s it, taps it via `brew tap <name> file://<path>`, installs by tap-qualified name, and untaps + uninstalls in the cleanup trap on both the pass and fail path.
- **Apple credential preconditions added to `release:rehearse-cask` mid-session, once discovered load-bearing.** Homebrew Cask's `Cask::Quarantine.cask!` (`extend/os/mac/cask/quarantine.rb`) unconditionally quarantines every downloaded artifact via a direct LaunchServices FFI call — regardless of URL scheme, so `file://` buys no exemption. There is no `--no-quarantine` install flag in this Homebrew version, and `spctl --add` (which could otherwise locally allowlist one binary's cdhash) was removed as of macOS 15.0 ("operations that modify the rule database... will no longer be supported", confirmed via `man spctl` on this host, running Darwin 27.0.0). Consequently, ANY target that installs a real cask and executes the installed binary — not just `release:rehearse-notarize`'s literal notarization submission — requires real Developer ID signing + notarization to avoid Gatekeeper SIGKILL. The target's precondition set now names the same five `MACOS_*` variables `release:rehearse-notarize` requires, so a future run without them fails fast and by name (measured: now fails in under a second at the precondition step) instead of after a ~15s build followed by a confusing `terminated by uncaught signal KILL`. **This finding remains true and load-bearing after the blocker below cleared** — it is the reason the credentialed rehearsal run had to use real Apple credentials rather than any workaround, and it is inherited context plan 03-02 depends on for its own cask work.
- **The blocker was resolved by the orchestrator, not worked around.** The original session's diagnosis (immediately above and in the resolution record below) was correct about the mechanism and incomplete about the environment: the five `MACOS_*` variables were not directly in the shell, but were one `op run --env-file=.env` away via `op://` references in a gitignored `.env` this session's executor had no visibility into. The orchestrator ran `op run --env-file=.env -- env CASK_REHEARSE=1 task release:rehearse-cask` and it reached `CASK-REHEARSE-EVIDENCE ... outcome=pass` (full evidence in the D3 coverage entry above and the resolution record below).
- **Task 3's TDD cycle measured one implementation gap, not two.** The plan's action text predicted both directory-creation and unwritable-path-error-naming would require new implementation. Only directory-creation did — Task 2's existing `fmt.Errorf("generate man pages into %s: %w", dir, err)` already satisfied the unwritable-path behavior. Recorded as a measured correction rather than silently treated as "the plan was right."

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

**3. [Rule 1/TDD GREEN] `man <dir>` did not create its target directory — the one behavior Task 3's test contract found genuinely RED**
- **Found during:** Task 3, first `go test ./internal/cli/... -run TestManCmd` run, before any implementation change
- **Issue:** `doc.GenManTree` does not create its own destination; `codegraph man <missing-dir>` failed with `no such file or directory` — this is the cask hook's actual dependency, since Homebrew's man1 directory is absent on a prefix where nothing has yet installed a man page
- **Fix:** `os.MkdirAll(dir, 0o755)` before `doc.GenManTree` in `internal/cli/man.go`
- **Files modified:** internal/cli/man.go
- **Verification:** `go test ./internal/cli/... -run TestManCmd -v` — all 5 behaviors GREEN; re-measured man page count unchanged at 30
- **Committed in:** `3d07cf9` (GREEN), preceded by `bbab820` (RED, the full 5-test contract)

### Resolved — Genuine External Blocker (cleared by the orchestrator, not worked around)

**`task release:rehearse-cask` could not reach PASS in the original session without real Apple Developer ID credentials — RESOLVED**

The original session's finding stands as true and load-bearing: Homebrew Cask's `Cask::Quarantine.cask!` (`extend/os/mac/cask/quarantine.rb`) unconditionally quarantines every downloaded artifact via a direct LaunchServices FFI call, regardless of URL scheme; there is no `--no-quarantine` install flag; and `spctl --add` was removed as of macOS 15.0. Consequently ANY target that installs a real cask and executes the installed binary — the shipped cask's post-install hook included — categorically requires a genuinely Developer-ID-signed and notarized binary. This mechanism, and the manual outside-Homebrew reproduction that confirmed it (ad-hoc-signed + quarantined → exit 137/SIGKILL), remain accurate and are inherited context for plan 03-02.

**What was incomplete, not wrong, in the original diagnosis:** the session correctly found zero `MACOS_*` credentials directly in the shell environment (`env | rg` returned no matches) and correctly concluded the executor could not supply them. What that session had no visibility into was that the maintainer's actual credentials lived one `op run --env-file=.env` away — `op://` references inside a gitignored `.env` never read into that session's environment.

**Resolution:** the orchestrator ran, on its own machine, with the real `.env` present:
```
op run --env-file=.env -- env CASK_REHEARSE=1 task release:rehearse-cask
```
This reached `CASK-REHEARSE-EVIDENCE schema=1 snapshot_version=unknown cask_sha256=2a9ddeca59a8d318a70d547671449e8028c3b6431db3846ca080ac465f4a7199 url_mechanism=file man_page_count=30 reported_version=codegraph v0.7.0 (commit 20b90fc98891ecb97aaccc920315c7ec37881b56, built 2026-08-10T14:37:44Z) go1.26.5 darwin/arm64 outcome=pass` and `release:rehearse-cask: PASS`. Full observed sequence (signing/notarization, cask render, one-line diff, tap, install, hook execution, man-page count, clean uninstall/untap) is recorded verbatim in the `D3` coverage entry above, along with the two caveats carried forward (`snapshot_version=unknown` gap in the evidence schema; the pre-release macOS 27 host warning).

- **A1 (binary at zip root, no rename handling needed): CONFIRMED** — linking succeeded with `binaries: [codegraph]` unchanged.
- **A3 (`#{HOMEBREW_PREFIX}/bin/codegraph` live at postflight time): CONFIRMED** — the post-install hook's invocation of the installed binary succeeded without needing `#{staged_path}`.
- **D-10 (the install is the gate): CONFIRMED** — the hook executed as part of `brew install`, and the rehearsal's uninstall-on-failure trap was exercised correctly on the earlier failing attempts this session did not need to repeat.
- **`.planning/WINDOWS.md` entry 1** (the `unrun-verify` this blocker produced) is closed via `gsd-tools windows fixed 1` — the ledger's `fixed` verb does not accept a reason parameter (only `waive` does), so the evidence trail lives here and in the D3 coverage rationale rather than in the ledger row itself.

---

**Total deviations:** 3 auto-fixed (2 from the original session — both Rule 3, blocking, necessary to make progress on files this plan already touches; 1 from this session — Rule 1/TDD GREEN, the directory-creation fix) + 1 genuine external blocker, now resolved (not worked around, not silently absorbed — the resolution path and evidence are recorded above).
**Impact on plan:** the tracer's own purpose — proving the whole Homebrew delivery path end-to-end before expansion work is built on it — has been achieved. Every acceptance criterion in Task 2's `<verify>` block that depended on Apple credentials now has a real, observed PASS. Task 3's behavior contract closes the gap between "the command exists" and "the command's behavior is tested," and the two genuinely new-implementation-requiring behaviors the plan called for turned out, on measurement, to be one.

## Issues Encountered

- **Homebrew auto-updated itself (6.0.15 → 6.0.16) mid-session**, triggered by the first `brew install --cask` attempt before `HOMEBREW_NO_AUTO_UPDATE=1`/`HOMEBREW_NO_ENV_HINTS=1` were added to the rehearsal target. This is an environment side effect outside this plan's file scope; not reverted (Homebrew has no supported downgrade path, and the version bump is not expected to affect anything this phase depends on).
- **`man <dir>` required the destination directory to already exist, RESOLVED this session.** Task 3's TDD RED, quoted verbatim: `man /var/folders/.../several/missing/parents/man1: unexpected error: generate man pages into /var/folders/.../several/missing/parents/man1: open /var/folders/.../several/missing/parents/man1/codegraph-affected.1: no such file or directory` (`TestManCmd_CreatesMissingDirectory_AndWritesFullTree`, first run, before `os.MkdirAll` was added). GREEN after `3d07cf9`.

## Known Stubs

None. `internal/cli/man.go` is a complete, real implementation, now including directory creation and unwritable-path error surfacing (Task 3) alongside Task 2's core generation behavior.

## Threat Flags

None beyond what the plan's own `<threat_model>` already covers. No new network endpoints, auth paths, or trust-boundary schema changes were introduced beyond what T-03-01/T-03-02/T-03-03/T-03-SC already anticipate.

## User Setup Required

None remaining for this plan. The five `MACOS_*` credentials (`MACOS_SIGN_P12`, `MACOS_SIGN_PASSWORD`, `MACOS_NOTARY_ISSUER_ID`, `MACOS_NOTARY_KEY_ID`, `MACOS_NOTARY_KEY`) that the original session found absent from its environment were supplied by the orchestrator via `op run --env-file=.env` against a gitignored `.env` of `op://` references, and `release:rehearse-cask` reached `outcome=pass`. See `docs/RELEASE.md` for how each credential is obtained, and the `release:rehearse-notarize`/`release:rehearse-cask` targets in `Taskfile.yml` for how a future maintainer re-runs either rehearsal locally.

## Next Phase Readiness

- **Ready for plan 03-02 to build on this plan's cask mechanism as proven.** The tracer's own purpose — proving the whole Homebrew delivery path end-to-end before expansion work is built on it — has been achieved. A1 (binary at zip root, no rename handling needed) and A3 (`#{HOMEBREW_PREFIX}/bin/codegraph` live at postflight time) are both **CONFIRMED**, observed directly during the credentialed rehearsal run (D3 coverage entry above), not merely traced from source.
- **What IS proven and safe to build on:** the `codegraph man` command (compiles, registered, hidden, generates 30 real man pages, now creates its own target directory including missing parents, and surfaces a non-nil error naming the offending path on write failure — all five behaviors under test); the `homebrew_casks:` block's shape (validates under `task check:goreleaser`, coexists cleanly with `release:dry-run`/`release:dry-run-signed`, `archives:` byte-unchanged); the render mechanism (`dist/homebrew/Casks/codegraph.rb` renders correctly from a real `goreleaser release --snapshot` run); the URL-rewrite mechanism (exactly one line, diff-verified); the local-tap-wrapping mechanism required for ANY future local cask rehearsal in this repository; and — the mechanism plan 03-02 most directly inherits — **any cask that executes the installed binary via a post-install hook requires a genuinely Developer-ID-signed and notarized build**, because Homebrew Cask quarantines unconditionally and macOS 15.0 removed `spctl --add`. This is not a config nicety; it is a hard Gatekeeper enforcement fact that governs every future local rehearsal target in this repository, not just this one.
- **Known gap, carried forward, not silently dropped:** the rehearsal evidence's `snapshot_version` field reports `unknown` rather than the resolved version (the version is present separately in `reported_version`). This is a gap in the evidence schema, not in what was proven — recorded here for whoever next touches `Taskfile.yml`'s evidence-emission logic (owned by the concurrent 03-03 executor in this session, not modified here).
- **`.planning/WINDOWS.md` entry 1** is now `status: fixed` (`gsd-tools windows fixed 1`) — 0 open entries remain from this plan.

---
*Phase: 03-homebrew-tap-cask*
*Completed: 2026-08-10*

## Self-Check: PASSED

- FOUND: `internal/cli/man.go`
- FOUND: `internal/cli/man_test.go`
- FOUND: `release:rehearse-cask` target in `Taskfile.yml`
- FOUND: `homebrew_casks:` block in `.goreleaser.yaml`
- FOUND: `man` section in `docs/FLAG-PARITY.md`
- FOUND: commit `8646f19` in `git log --oneline --all`
- FOUND: commit `bbab820` in `git log --oneline --all`
- FOUND: commit `3d07cf9` in `git log --oneline --all`
- FOUND: commit `208b385` in `git log --oneline --all`
