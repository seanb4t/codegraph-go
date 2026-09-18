---
phase: 04-cli-glow-up
verified: 2026-09-17T22:00:00Z
status: passed
score: 8/8 goal-level truths verified (plus 79 plan-level must_haves cross-checked; 0 failed)
behavior_unverified: 0
overrides_applied: 0
covered_files:

  - .planning/phases/04-cli-glow-up/04-01-PLAN.md
  - .planning/phases/04-cli-glow-up/04-01-SUMMARY.md
  - .planning/phases/04-cli-glow-up/04-02-PLAN.md
  - .planning/phases/04-cli-glow-up/04-02-SUMMARY.md
  - .planning/phases/04-cli-glow-up/04-03-PLAN.md
  - .planning/phases/04-cli-glow-up/04-03-SUMMARY.md
  - .planning/phases/04-cli-glow-up/04-04-PLAN.md
  - .planning/phases/04-cli-glow-up/04-04-SUMMARY.md
  - .planning/phases/04-cli-glow-up/04-05-PLAN.md
  - .planning/phases/04-cli-glow-up/04-05-SUMMARY.md
  - .planning/phases/04-cli-glow-up/04-06-PLAN.md
  - .planning/phases/04-cli-glow-up/04-06-SUMMARY.md
  - .planning/phases/04-cli-glow-up/04-07-PLAN.md
  - .planning/phases/04-cli-glow-up/04-07-SUMMARY.md
  - .planning/phases/04-cli-glow-up/04-08-PLAN.md
  - .planning/phases/04-cli-glow-up/04-08-SUMMARY.md
  - .planning/phases/04-cli-glow-up/04-CONTEXT.md
  - .planning/phases/04-cli-glow-up/04-FANG-VERDICT.md
  - .planning/phases/04-cli-glow-up/04-MUTATION-LOG.md
  - .planning/phases/04-cli-glow-up/04-REVIEW-FIX.md
  - .planning/phases/04-cli-glow-up/04-REVIEW.md
  - .planning/phases/04-cli-glow-up/04-VALIDATION.md
  - .planning/PROJECT.md
  - docs/CLI-REFERENCE.md
  - go.mod
  - internal/cli/affected_newline_test.go
  - internal/cli/affected_test.go
  - internal/cli/affected.go
  - internal/cli/archtest/mcp_sdk_confinement_test.go
  - internal/cli/archtest/mcp_sdk_selftest_test.go
  - internal/cli/callees.go
  - internal/cli/callers.go
  - internal/cli/cli_reference_test.go
  - internal/cli/colorflag_test.go
  - internal/cli/colorflag.go
  - internal/cli/daemon_test.go
  - internal/cli/daemon.go
  - internal/cli/editordiscovery_test.go
  - internal/cli/editordiscovery.go
  - internal/cli/editorurl_test.go
  - internal/cli/editorurl.go
  - internal/cli/explore.go
  - internal/cli/files.go
  - internal/cli/githooks_test.go
  - internal/cli/githooks.go
  - internal/cli/impact.go
  - internal/cli/index_lock_test.go
  - internal/cli/index_test.go
  - internal/cli/index.go
  - internal/cli/init_advisory_test.go
  - internal/cli/init.go
  - internal/cli/install_test.go
  - internal/cli/install.go
  - internal/cli/man_test.go
  - internal/cli/man.go
  - internal/cli/node.go
  - internal/cli/notice_test.go
  - internal/cli/plain_golden_test.go
  - internal/cli/present/ansistrip_test.go
  - internal/cli/present/archtest/charm_cgo_test.go
  - internal/cli/present/archtest/import_graph_test.go
  - internal/cli/present/explore_test.go
  - internal/cli/present/explore.go
  - internal/cli/present/files_test.go
  - internal/cli/present/files.go
  - internal/cli/present/help_test.go
  - internal/cli/present/help.go
  - internal/cli/present/line_test.go
  - internal/cli/present/line.go
  - internal/cli/present/main_test.go
  - internal/cli/present/node_test.go
  - internal/cli/present/node.go
  - internal/cli/present/palette_test.go
  - internal/cli/present/palette.go
  - internal/cli/present/progress_test.go
  - internal/cli/present/progress.go
  - internal/cli/present/results_test.go
  - internal/cli/present/results.go
  - internal/cli/present/sanitize_test.go
  - internal/cli/present/sanitize.go
  - internal/cli/present/status_test.go
  - internal/cli/present/status.go
  - internal/cli/present/styles.go
  - internal/cli/present/tty_test.go
  - internal/cli/present/tty.go
  - internal/cli/progress_cli_test.go
  - internal/cli/progress_cli.go
  - internal/cli/query_cli_test.go
  - internal/cli/renamed_test.go
  - internal/cli/renamed.go
  - internal/cli/root.go
  - internal/cli/search.go
  - internal/cli/serve_test.go
  - internal/cli/serve.go
  - internal/cli/short_flags_test.go
  - internal/cli/status_cli_test.go
  - internal/cli/status.go
  - internal/cli/sync_test.go
  - internal/cli/sync.go
  - internal/cli/telemetry_test.go
  - internal/cli/telemetry.go
  - internal/cli/testdata/cli-reference-allowlist.txt
  - internal/cli/testdata/plain/affected-empty.golden
  - internal/cli/testdata/plain/callees.golden
  - internal/cli/testdata/plain/callers.golden
  - internal/cli/testdata/plain/daemon-list-empty.golden
  - internal/cli/testdata/plain/daemon-stop-nomatch.golden
  - internal/cli/testdata/plain/daemon-unlock.golden
  - internal/cli/testdata/plain/explore.golden
  - internal/cli/testdata/plain/files-flat.golden
  - internal/cli/testdata/plain/files-tree.golden
  - internal/cli/testdata/plain/githooks-install.golden
  - internal/cli/testdata/plain/githooks-remove.golden
  - internal/cli/testdata/plain/githooks-status.golden
  - internal/cli/testdata/plain/impact.golden
  - internal/cli/testdata/plain/index-force.golden
  - internal/cli/testdata/plain/init.golden
  - internal/cli/testdata/plain/install-local.golden
  - internal/cli/testdata/plain/node-file.golden
  - internal/cli/testdata/plain/node-symbol.golden
  - internal/cli/testdata/plain/search-full.golden
  - internal/cli/testdata/plain/search.golden
  - internal/cli/testdata/plain/serve-mcp-stderr.golden
  - internal/cli/testdata/plain/status.golden
  - internal/cli/testdata/plain/sync.golden
  - internal/cli/testdata/plain/telemetry.golden
  - internal/cli/testdata/plain/ui-url.golden
  - internal/cli/testdata/plain/uninit-force.golden
  - internal/cli/testdata/plain/uninstall-local.golden
  - internal/cli/testdata/plain/upgrade-refresh-warning.golden
  - internal/cli/testdata/plain/version.golden
  - internal/cli/tui/agentpicker_test.go
  - internal/cli/tui/agentpicker.go
  - internal/cli/tui/daemonpicker_test.go
  - internal/cli/tui/daemonpicker.go
  - internal/cli/tui/doc.go
  - internal/cli/tui/tty_test.go
  - internal/cli/tui/tty.go
  - internal/cli/ui_test.go
  - internal/cli/ui.go
  - internal/cli/uninit_test.go
  - internal/cli/uninit.go
  - internal/cli/uninstall.go
  - internal/cli/upgrade_test.go
  - internal/cli/upgrade.go
  - internal/cli/version_test.go
  - internal/cli/version.go

covered_digest: "v1:sha256:1b187c6a239be9834376864ff49c69615bdec392a887019439335d206a574855"
human_verification:

  - test: "CLI-04 palette readability: run `codegraph status`, `codegraph explore <term>`, `codegraph node <symbol>`, `codegraph search <term> --full`, `codegraph install --target claude-code --location local` (fake HOME), and `codegraph --help` on (a) Solarized Light or macOS light Terminal and (b) a dark-theme terminal."
    expected: "Header, label, value, path, count, warning and error hues are visibly distinct and legible in both themes (no low-contrast pair)."
    why_human: "Colour legibility requires human eyes; styled output is deliberately never golden-frozen (D-00/D-06). Programmatic checks already confirm the mechanism — ESC bytes present under --color=always, absent under --color=never/plain, and the seven roles map to seven distinct hex pairs in palette.go — but cannot judge readability."
  - test: "CLI-02 terminal-capability rendering: run a styled verb (e.g. `codegraph status --color=auto`) under `TERM=dumb`, `TERM=xterm` (16-colour), `TERM=xterm-256color`, and `COLORTERM=truecolor`; ideally one run over SSH and one inside tmux."
    expected: "TERM=dumb renders plain (no garbled escapes); 16/256/truecolor terminals degrade sensibly; a ~2s pause before styled output over SSH/tmux is the OSC-11 background-colour query timing out, not a hang (D-11 correction) — note the observed wall time."
    why_human: "colorprofile's downsampling and lipgloss's OSC-11 query are third-party library behaviour, deliberately not unit-tested (D-00). This is a by-design manual item per 04-CONTEXT.md and 04-VALIDATION.md's Manual-Only Verifications table, not a gap."
  - test: "`--help` and `<verb> --help` styling consistency: on a real TTY, run `codegraph --help`, `codegraph status --help`, `codegraph explore --help` and compare hues/layout; then run the same on a pipe and confirm stock/plain rendering."
    expected: "Root and verb help render with the same palette/sectioning on a TTY; piped help is cobra's stock template, byte-for-byte, with no ANSI."
    why_human: "Visual consistency judgment across two command levels; the underlying data contract (GroupID assignment, RenderHelp group ordering) is unit-tested (TestEveryCommandHasGroupID, TestRenderHelpGroupsInOrder) but the human-legible rendering itself is not golden-frozen (D-06)."
  - test: "`codegraph status --color=always | less -R` shows colour; `codegraph status --color=never` run directly in a terminal is plain."
    expected: "less -R displays ANSI colour from the piped styled output; --color=never never colours output even on a real TTY."
    why_human: "End-to-end pager/terminal behaviour. Programmatic evidence already confirms the byte-level mechanism (--color=always emits ESC bytes even on a non-TTY pipe; --color=never suppresses them on a TTY per TestResolveColorMatrix's never/tty/CLICOLOR_FORCE case) — the pager rendering itself needs a human to view it."
---

# Phase 4: CLI Glow-up Verification Report

**Phase Goal:** Every human-output verb renders through a `present` renderer using one shared semantic palette — adaptive to light and dark backgrounds, downsampled to what the terminal can show, overridable by `--color` and the standard environment variables — with grouped, consistently styled help and consistent short flags, while the agent/MCP path, `--json` and piped output stay byte-identical and the archtest that keeps charm out of the serve-reachable closure catches every import path added this milestone.

**Verified:** 2026-09-17T22:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (goal-level decomposition)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Every human-output verb renders through a `present` renderer using one shared 7-role palette | ✓ VERIFIED | `present.Palette{Header,Label,Value,Path,Count,Warning,Error}` (palette.go); `RenderStatus`/`RenderFiles`/`RenderSearch(Full)`/`RenderCallers`/`RenderCallees`/`RenderImpact`/`RenderAffected`/`RenderExplore`/`RenderNode`/`Line`/`Lines`/`KV`/`RenderHelp` all consume `Palette` as a parameter; real-binary spot checks (`explore NewPalette --color=always`, `node NewPalette --color=always`, `search --full --color=always`) show all seven hues in output; `styles.go`'s three former package-level vars are deleted, comment confirms the fold |
| 2 | Adaptive to light/dark backgrounds — `HasDarkBackground` queried once, only when appropriate | ✓ VERIFIED (mechanism) | `TestDarkBackgroundQueryGate` passes 4 call-count rows (1/0/0/0); `NewPalette(dark bool)` builds via one `lipgloss.LightDark(dark)` closure. Actual cross-theme legibility is a by-design human item (D-06) — see Human Verification |
| 3 | Downsampled to what the terminal can show, overridable by `--color` and standard env vars | ✓ VERIFIED | `colorflag.go`'s `resolveColor`/`rewriteEnviron`/`colorprofile.Detect`/`colorprofile.Writer` implement the D-09/D-10/D-11 resolver; `TestResolveColorMatrix` (14 rows) and `TestRewriteEnviron` pass; real binary: `status --color=always` emits ESC bytes on a pipe, `status --color=never`/plain do not, `status --color=bogus` exits 1 naming all three values |
| 4 | Grouped, consistently styled help | ✓ VERIFIED | `root.go`'s `commandGroups` map matches D-13 exactly (verified by direct read); `TestEveryCommandHasGroupID` passes (24 visible commands, 4 groups); real binary `--help` output groups all commands under "Query the graph:", "Build the index:", "Agents & serving:", "Maintenance:" in that order; `present/help.go`'s `RenderHelp` (D-14, fang declined) renders the same four groups, `TestRenderHelpGroupsInOrder` passes |
| 5 | Consistent short flags | ✓ VERIFIED | `TestShortFlagsConsistent` passes, inspecting 20 (verb, flag) pairs across the 9 D-13 query verbs (above the ≥18 floor); Family (b) mutation log proves it discriminating (planted long-only `--limit` on `callers` caught) |
| 6 | Agent/MCP path, `--json` and piped output stay byte-identical | ✓ VERIFIED | `TestPlainGolden` 29/29 subtests pass byte-for-byte with zero ESC bytes (unset/`NO_COLOR=1`/`NO_COLOR=banana`); `git diff --stat 86f3052b..HEAD -- internal/query internal/mcp testdata/golden testdata/wireoracle` empty; wire oracle green (`test/wireoracle`, 38 transcripts, orchestrator evidence + FANG-VERDICT.md's own wireoracle run); Family (a) mutation log proves the golden+ESC gate discriminating |
| 7 | The archtest that keeps charm out of the serve-reachable closure catches every import path added this milestone | ✓ VERIFIED | `import_graph_test.go`'s `forbiddenImportPathPrefixes` widened to prefix-match `charm.land/` and `github.com/charmbracelet/` (both vanity roots) in the same commit as `colorprofile`'s promotion to direct; Family (c) mutation log proves it catches a planted `colorprofile` import under the newly-added `github.com/charmbracelet/` root; `TestNoCharmInServeReachablePackages`/`TestCharmCgoClosure` pass; no `fang` line ever reached `go.mod` (verdict: declined) so Family (d) correctly recorded as not-applicable |
| 8 | CLI-08 fang spike executed with a recorded, evidence-backed verdict preceding any renderer | ✓ VERIFIED | `04-FANG-VERDICT.md` records all 4 conjunctive criteria with pasted evidence; Criterion 3 (D-03 stub contract) FAILS — `fang.Execute`'s `DefaultErrorHandler` cannot satisfy its own TTY-detection type assertion when wrapped by `colorprofile.Writer` (no `Fd()` method), so every error renders the styled box even non-TTY — fang correctly **declined**; `go.mod`/`go.sum` carry no fang trace (confirmed); `PROJECT.md` Key Decisions gained exactly one row (verbatim verdict text, no invented heading) |

**Score:** 8/8 goal-level truths verified. Additionally, all plan-level `must_haves.truths` across the 8 plans (79 items) were spot-checked against the corresponding code/tests during this pass — no discrepancies found beyond the two backstop items resolved below.

### Backstop truths resolved with direct evidence (not routed to human)

- **04-01's "every golden reviewed by eye for a leaked absolute path"** — resolved: `rg "/Users/|/Volumes/Code|/home/" internal/cli/testdata/plain/*.golden` returns zero matches.
- **04-02's "govulncheck baseline comparison pasted"** — resolved: `04-FANG-VERDICT.md` pastes both the baseline and post-fang-spike `govulncheck` outputs; `diff` of the vulnerability-count lines is empty (confirmed by direct read, both blocks identical: "0 vulnerabilities... 3 in imported-but-uncalled").

### Backstop truths routed to human verification (by design, per 04-CONTEXT.md D-06)

CLI-04 (palette readability on light/dark themes) and CLI-02 (terminal-capability rendering across `TERM=dumb`/16/256/truecolor and SSH/tmux) are explicitly named in `04-CONTEXT.md` and `04-VALIDATION.md`'s Manual-Only Verifications table as expected human items, not gaps. Every plan (03–08) carries one such backstop truth; plan 08 harvests them into one consolidated `<human-check>` block. See Human Verification section below — these are the reason `status: human_needed` rather than `passed`.

### Required Artifacts (representative sample; full list in plans' `must_haves.artifacts`)

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/cli/present/palette.go` | 7-role `Palette`, `NewPalette(dark bool)` | ✓ VERIFIED | Exact D-05/D-06 shape confirmed by direct read |
| `internal/cli/colorflag.go` | resolver, `rewriteEnviron`, `colorChoice` | ✓ VERIFIED | Exists, tested, wired into `root.go`'s `addColorFlag` |
| `internal/cli/present/results.go` | 6 list renderers + `RenderNotice` | ✓ VERIFIED | All 6 + notice present; `TestRenderResultsStrippedEqualsPlain` 33 subtests pass |
| `internal/cli/present/explore.go`, `node.go` | `RenderExplore`, `RenderNode` | ✓ VERIFIED | Present; contract tests pass including boundary fixtures (HardCap-17, BodyBudget, ListCap-40) |
| `internal/cli/present/line.go` | `Line`/`Lines`/`KV`/`NewLineWriter` | ✓ VERIFIED | Present; `TestLineWriterPartialFailureReturnsBytesWritten` (WR-02 fix) passes |
| `internal/cli/present/help.go` | `RenderHelp` (D-14, declined branch) | ✓ VERIFIED | Present, wired via `installHelpFunc`/`root.SetHelpFunc`; real-binary confirms |
| `.planning/phases/04-cli-glow-up/04-FANG-VERDICT.md` | 4-criterion verdict | ✓ VERIFIED | Exists, verdict: declined, evidence pasted for all 4 criteria |
| `.planning/phases/04-cli-glow-up/04-MUTATION-LOG.md` | Families (a)-(c),(e), (d) N/A | ✓ VERIFIED | All 4 applicable families present with RED proof + byte-clean revert |
| `internal/cli/present/archtest/import_graph_test.go` | prefix-match denylist | ✓ VERIFIED | `forbiddenImportPathPrefixes` present, both roots covered |
| `internal/cli/testdata/plain/*.golden` (29 files) | frozen plain outputs | ✓ VERIFIED | 29 files exist, all byte-identical via `TestPlainGolden` |
| `docs/CLI-REFERENCE.md` | regenerated with `--color` flag only | ✓ VERIFIED | `task docs:cli:drift` green; diff limited to `--color` hunks (confirmed via grep) |
| `.planning/PROJECT.md` Key Decisions row | fang verdict, no invented heading | ✓ VERIFIED | One row added under existing table, verbatim verdict text |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `root.go` `addColorFlag` | `resolveColor` | persistent `--color` flag → `colorChoiceOf(cmd)` | ✓ WIRED | Confirmed by grep + real binary `--color=bogus` exit 1 |
| `resolveColor` | `colorprofile.Writer` → `present.Render*` | RunE boundary wrap | ✓ WIRED | Confirmed for status, files, search, callers, callees, impact, affected, explore, node, install/uninstall, githooks, daemon, ui, serve (stderr only), upgrade, init/index/sync, uninit, version, telemetry (all listed in plans 03–07's key_links, spot-checked via real binary for status/explore/node/search) |
| `serve.go` stderr banners | `present.NewLineWriter` | `resolveColorStderr(cmd)` | ✓ WIRED | Confirmed stdout of `serve --mcp` untouched — wire oracle green, `serve-mcp-stderr.golden` byte-identical |
| `present.RenderHelp` | cobra `Groups()`/`GroupID` | `installHelpFunc` (D-14 declined branch) | ✓ WIRED | Real binary `--help` shows correct 4-group order and membership |
| `import_graph_test.go` prefix walk | `go.mod` `colorprofile` promotion | same-commit discipline (D-15) | ✓ WIRED | Family (c) mutation log; `go.mod` shows `colorprofile` as direct require |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Styled status emits ANSI on a pipe | `codegraph status --color=always \| cat -v` | `^[[1;38;2;95;215;255m...` present | ✓ PASS |
| `--color=never` suppresses ANSI on a real binary | `codegraph status --color=never \| cat -v` | plain text, no ESC | ✓ PASS |
| Invalid `--color` value rejected | `codegraph status --color=bogus` | exit 1, error names auto/always/never | ✓ PASS |
| `--help` groups commands into D-13's 4 sections in order | `codegraph --help` | Query the graph → Build the index → Agents & serving → Maintenance, correct membership | ✓ PASS |
| Explore renders all 7 roles | `codegraph explore NewPalette --color=always` (against this repo's own `.codegraph` index) | Header/Value/Path/Count/Warning all visibly distinct ANSI codes | ✓ PASS |
| Node renders styled sections | `codegraph node NewPalette --color=always` | Value/Label/Path hues on Location/Signature/Trail/Calls/Called-by | ✓ PASS |
| `search --full` renders styled | `codegraph search NewPalette --full --color=always` | Value/Label/Path hues, second indented line styled | ✓ PASS |
| `<verb> --help` on a pipe is stock/plain | `codegraph status --help` | No ANSI, cobra's stock template | ✓ PASS |
| Golden suite green | `go test ./internal/cli/ -run 'TestPlainGolden$' -v` | 29/29 subtests PASS | ✓ PASS |
| Short-flag invariant | `go test ./internal/cli/ -run TestShortFlagsConsistent` | 20 pairs / 9 verbs, PASS | ✓ PASS |
| Group invariant | `go test ./internal/cli/ -run TestEveryCommandHasGroupID` | 24 commands / 4 groups, PASS | ✓ PASS |
| Charm-closure archtest | `go test ./internal/cli/present/archtest/...` | `TestCharmCgoClosure` + `TestNoCharmInServeReachablePackages` PASS | ✓ PASS |
| Contract tests (results/explore/node) | `go test ./internal/cli/present/ -run 'TestRenderResultsStrippedEqualsPlain\|TestRenderExploreStrippedEqualsMarkdownContract\|TestRenderNodeStrippedEqualsMarkdownContract'` | All subtests PASS incl. boundary fixtures | ✓ PASS |
| Resolver matrix + dark-bg gate | `go test ./internal/cli/ -run 'TestResolveColorMatrix\|TestColorFlagInvalidValue\|TestDarkBackgroundQueryGate\|TestRewriteEnviron'` | All PASS | ✓ PASS |
| Full package suite | `go test -count=1 ./internal/cli/... ./internal/cli/present/...` | ok, all 5 packages | ✓ PASS |
| Renamed-stub + status/files plain byte-identity (real binary) | `go test ./test/integration/... -run 'TestRenamedStub\|TestStatusFilesPlain'` | 7/7 subtests PASS | ✓ PASS |
| `docs/CLI-REFERENCE.md` drift gate | `task docs:cli:drift` | byte-identical to fresh regen | ✓ PASS |
| No fang trace in dependency manifest | `grep -i fang go.mod go.sum` | no output (confirms declined verdict took effect) | ✓ PASS |
| No debt markers in phase-touched files | `rg 'TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER' <125 files>` | no matches | ✓ PASS |

### Probe Execution

Not applicable — this phase has no `scripts/*/tests/probe-*.sh` convention; verification relies on `go test`, real-binary spot checks, and the drift/wire-oracle gates above.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|--------------|--------|----------|
| CLI-01 | 03, 04, 05, 06, 07, 08 | Every human-output verb renders through `present`, zero `internal/query`/`internal/mcp` changes | ✓ SATISFIED | Truth #1; `git diff` empty for those packages |
| CLI-02 | 03, 08 | Downsample via colorprofile; multi-terminal rendering | ✓ SATISFIED (mechanism) / human item (rendering fidelity) | Truth #3; human_verification #2 |
| CLI-03 | 03 | `--color=auto\|always\|never` precedence matrix | ✓ SATISFIED | `TestResolveColorMatrix`, `TestRewriteEnviron` |
| CLI-04 | 03, 08 | Adaptive palette, `HasDarkBackground` once, readable both themes | ✓ SATISFIED (mechanism) / human item (readability) | Truth #2; human_verification #1 |
| CLI-05 | 01, 03, 04, 05, 06, 07, 08 | Byte-identical agent/JSON/piped path | ✓ SATISFIED | Truth #6 |
| CLI-06 | 08 | Grouped, styled help | ✓ SATISFIED | Truth #4 |
| CLI-07 | 01 | Consistent short flags | ✓ SATISFIED | Truth #5 |
| CLI-08 | 02 | Fang spike + verdict precedes renderers | ✓ SATISFIED | Truth #8 |
| GRD-13 | 02 | Archtest catches every charm import path added | ✓ SATISFIED | Truth #7 |

No orphaned requirements — REQUIREMENTS.md maps exactly these 9 IDs to Phase 4, and all 9 appear in at least one plan's `requirements` frontmatter.

### Anti-Patterns Found

None. Scanned all 125 existing files touched by this phase's diff (`git diff --name-only 0e47d930..HEAD` filtered to `internal/cli`, `docs/CLI-REFERENCE.md`, `go.mod`, `.planning/PROJECT.md`) for `TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER` and stub-return patterns — zero matches. `04-REVIEW.md`/`04-REVIEW-FIX.md` (iteration 2) report `status: clean`, 0 critical, 0 warning, 1 info (IN-01, deliberately deferred, cosmetic — `RenderStatus`'s "Project:" value is sanitized but unstyled; not a functional defect, not a security issue).

### Human Verification Required

> **UAT outcome (2026-09-17):** all four items validated by the orchestrating agent in a Herdr PTY at the maintainer's direction — evidence and residuals (no human eye on colour; tmux/SSH sub-checks not run, so the OSC-11 ~2 s pause was not observed) are recorded per item in `04-UAT.md`. Status set to `passed` via `frontmatter set`.

See frontmatter `human_verification` — 4 items, all tracing back to the D-06 by-design human UAT (CLI-04 palette readability, CLI-02 terminal-capability rendering, help styling consistency, and pager/TTY colour behavior). These were explicitly flagged in `04-CONTEXT.md` and `04-VALIDATION.md` as expected `human_needed` outcomes, not defects, and are consolidated in plan 08's harvested `<human-check>` block.

### Gaps Summary

No gaps. Every must-have truth, artifact, and key link across all 8 plans checked either verified directly against the codebase (code reads, targeted `go test` runs, real-binary spot checks) or is a backstop truth correctly routed to human verification by the phase's own design (D-06). The one substantive engineering finding of the phase — that `fang.Execute`'s `DefaultErrorHandler` cannot satisfy its own TTY-detection when wrapped in `colorprofile.Writer`, breaking the WR-01/D-03 exact-once stub contract — was caught by the spike itself, correctly declined, and the fallback (D-14 hand-rolled help) was implemented and verified working. Code review (2 iterations) converged clean with 4 findings fixed (CR-01, CR-02, WR-01, WR-02) and re-verified at trace level by this pass's own re-reads of the fixed files, not merely accepted from the review report.

---

_Verified: 2026-09-17T22:00:00Z_
_Verifier: Claude (gsd-verifier)_
