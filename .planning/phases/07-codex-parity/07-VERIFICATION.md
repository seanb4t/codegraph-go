---
phase: 07-codex-parity
verified: 2026-09-19T00:00:00Z
status: passed
score: 6/6 must-haves verified (5 verified + 1 accepted override)
covered_files: [".codex/hooks/codegraph-pretooluse-global.sh", ".codex/hooks/codegraph-pretooluse-local.sh", ".codex/hooks/hooks.json", ".planning/WINDOWS.md", ".planning/phases/07-codex-parity/07-01-PLAN.md", ".planning/phases/07-codex-parity/07-01-SUMMARY.md", ".planning/phases/07-codex-parity/07-02-PLAN.md", ".planning/phases/07-codex-parity/07-02-SUMMARY.md", ".planning/phases/07-codex-parity/07-03-PLAN.md", ".planning/phases/07-codex-parity/07-03-SUMMARY.md", ".planning/phases/07-codex-parity/07-04-PLAN.md", ".planning/phases/07-codex-parity/07-04-SUMMARY.md", ".planning/phases/07-codex-parity/07-05-PLAN.md", ".planning/phases/07-codex-parity/07-05-SUMMARY.md", ".planning/phases/07-codex-parity/07-06-PLAN.md", ".planning/phases/07-codex-parity/07-06-SUMMARY.md", ".planning/phases/07-codex-parity/07-07-PLAN.md", ".planning/phases/07-codex-parity/07-07-SUMMARY.md", ".planning/phases/07-codex-parity/07-08-PLAN.md", ".planning/phases/07-codex-parity/07-08-SUMMARY.md", ".planning/phases/07-codex-parity/07-09-PLAN.md", ".planning/phases/07-codex-parity/07-09-SUMMARY.md", ".planning/phases/07-codex-parity/07-10-PLAN.md", ".planning/phases/07-codex-parity/07-10-SUMMARY.md", ".planning/phases/07-codex-parity/07-11-PLAN.md", ".planning/phases/07-codex-parity/07-11-SUMMARY.md", ".planning/phases/07-codex-parity/07-CONTEXT.md", ".planning/phases/07-codex-parity/07-LIVE-SESSIONS.md", ".planning/phases/07-codex-parity/07-MUTATION-LOG.md", ".planning/phases/07-codex-parity/07-PATTERNS.md", ".planning/phases/07-codex-parity/07-RESEARCH.md", ".planning/phases/07-codex-parity/07-REVIEW-FIX.md", ".planning/phases/07-codex-parity/07-REVIEW.md", ".planning/todos/completed/2026-09-18-install-yes-discards-explicit-target.md", "README.md", "codexassets.go", "docs/AGENT-CAPABILITIES.md", "docs/CLI-REFERENCE.md", "internal/agents/capabilities.go", "internal/agents/capabilities_test.go", "internal/agents/capability_doc_test.go", "internal/agents/codex.go", "internal/agents/codex_pretooluse.go", "internal/agents/codex_pretooluse_test.go", "internal/agents/codex_test.go", "internal/agents/instructions.go", "internal/agents/opencode.go", "internal/agents/ownership_test.go", "internal/agents/registry_test.go", "internal/agents/shared.go", "internal/agents/shared_test.go", "internal/agents/testdata/toml/codex-bom.installed.toml", "internal/agents/testdata/toml/codex-bom.toml", "internal/agents/testdata/toml/codex-bom.uninstalled.toml", "internal/agents/testdata/toml/codex-indented-layout.installed.toml", "internal/agents/testdata/toml/codex-indented-layout.toml", "internal/agents/testdata/toml/codex-indented-layout.uninstalled.toml", "internal/agents/toml.go", "internal/agents/toml_test.go", "internal/cli/hook_pretooluse.go", "internal/cli/hook_pretooluse_codex_test.go", "internal/cli/install.go", "internal/cli/install_test.go", "internal/cli/printconfigstyle_test.go", "internal/cli/testdata/plain/print-config-style-local.golden", "internal/cli/testdata/plain/print-config-style.golden", "internal/cli/tui/agentpicker.go", "internal/cli/tui/agentpicker_test.go", "internal/cli/tui/daemonpicker.go", "internal/cli/tui/daemonpicker_test.go", "internal/cli/uninstall.go", "internal/mcp/instructions_contract_test.go", "internal/mcp/server.go", "test/tmux/install_cancel_test.go"]
covered_digest: "v1:sha256:b01c231d78e25e0fa6b386d58f5b30f906b77dcd15b7d17f37e26c6c38c18b3d"
behavior_unverified: 1
overrides_applied: 1
overrides:
  - must_have: "FIX-03's picker-footer fix is asserted by the tmux real-PTY harness, re-run after CODEX-02's scope flip (ROADMAP SC-5, REQUIREMENTS.md FIX-03 literal text)."
    reason: "tmux was retired from the maintainer's machines in favour of herdr, so the local real-PTY run was waived on 2026-09-19 (GitHub issue #75 tracks whether the harness moves to herdr, a pure-Go PTY, stays CI-only, or is retired). The underlying defect is fixed and positive-controlled at the model level: lipgloss.Height(View()) <= 30 at 100x30 with all 8 targets, RED on the pre-fix delegate and GREEN at HEAD (07-MUTATION-LOG.md Families (c1)/(c2), re-run by the verifier). The re-anchored TTY-05 assertion compiles under `go vet -tags tmux`; CI's tmux-e2e job runs it when the branch is pushed. The maintainer accepted the model-level guard as sufficient for this phase rather than publishing the branch early to satisfy the literal wording."
    accepted_by: "sean"
    accepted_at: "2026-09-19T00:00:00Z"
behavior_unverified_items:
  - truth: "FIX-03's picker-footer fix is asserted by the tmux real-PTY harness, re-run after CODEX-02's scope flip (ROADMAP SC-5, REQUIREMENTS.md FIX-03 literal text)."
    test: "Run `task test:tmux` (or the CI `tmux-e2e` job) at this HEAD, which spawns the release binary inside a genuine 100x30 tmux pane, drives the install/uninstall picker, and asserts TTY-05's re-anchored `capture-pane` output contains the footer text `space: toggle` plus all 8 target display names, with `executed=6 skipped=0`."
    expected: "TTY-05 passes with `executed=6`, confirming the real-PTY overflow (which the model-level test cannot observe — actual terminal escape handling, bubbletea's real render loop, pagination under a live TTY) is genuinely fixed post-flip, not just fixed at the bubbletea-model level."
    why_human: "The maintainer explicitly decided (2026-09-19, recorded in 07-MUTATION-LOG.md Family (c3) and STATE.md) to skip local tmux evidence because tmux was retired from this machine in favor of herdr (GitHub issue #75 tracks the harness's future). The branch is unpushed (`git log origin/gsd/v0.14.0-milestone` fails — no remote-tracking ref), so the CI `tmux-e2e` job that is supposed to be the fallback real-PTY confirmation has never run against this code either. `go vet -tags tmux ./test/tmux/...` confirms the re-anchored assertion compiles, but no execution evidence — local or CI — exists at this HEAD. This can only be resolved by a human pushing the branch and observing the CI job, or by accepting the model-level guard (Families c1/c2, which I independently re-ran and confirmed GREEN) as sufficient evidence in place of the literal tmux re-run."
coincidental_reliance_items: []
---


**Re-verified at HEAD 2026-09-20 (v0.14.0 milestone audit).** This report went `stale` because later phases legitimately modified files in its `covered_files` — Phase 7's own fixes touched `internal/cli/install.go`, and Phases 5-7 touched `internal/agents/*`. The milestone audit re-exercised these claims against HEAD rather than re-stamping blind: the full suite (54 packages plus `internal/daemon` run separately) is green, an independent integration check verified 7/7 cross-phase seams with a real-binary end-to-end pass, and every phase holds a security audit with 0 threats open at or above the `high` gate. The digest below is recomputed over the same covered set at that HEAD. See `.planning/v0.14.0-MILESTONE-AUDIT.md`.
# Phase 7: Codex Parity Verification Report

**Phase Goal:** Codex receives everything Claude Code does — project-local MCP config through the existing TOML splice, the skill package at the path(s) Codex actually reads, a repo-root `AGENTS.md` block, and an opt-in PreToolUse nudge carrying Phase 6's contract — with every mechanism verified in a live scratch project before a line of `codex.go` changes, the picker footer verified after the target count settles, the published capability table honest about what ships, and a genuinely fresh Codex session reaching for codegraph unprompted.
**Verified:** 2026-09-19
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth (mapped to ROADMAP Success Criterion) | Status | Evidence |
|---|---|---|---|
| 1 | **SC-1 (CODEX-01):** Before any `codex.go` change, a live verification in a scratch trusted project records — with dated citations — whether Codex loads a project-scoped `.codex/config.toml`, which skill path(s) it reads, and whether `hooks.json` works behind `features.hooks`; the stale doc comment is corrected in the same commit. | ✓ VERIFIED | `07-LIVE-SESSIONS.md` records `CODEX-01 verdict: PASS` (L1 project/global, L2, L3, L4, L7 all PASS, line 603) with dated Codex CLI 0.155.0 citations and verbatim excerpts. `internal/agents/codex.go:13-29` now reads "Verified live 2026-09-19 against codex-cli 0.155.0 (…CODEX-01)…" — the prior "no per-project config" claim is gone (confirmed by direct read). Commit order confirmed: 07-04 (live evidence, CODEX-01) precedes 07-05 (first `codex.go` change) per plan `depends_on` chain and git log. |
| 2 | **SC-2 (CODEX-02/03/04):** `SupportsLocation(local)` is true; local install writes `.codex/config.toml` via the TOML splice preserving existing tables/inline tables/comments/CRLF, tells the user to trust the project; skill installed at both scopes idempotently/byte-invariant; marker block in repo-root `AGENTS.md`; uninstall removes it leaving the rest byte-identical. | ✓ VERIFIED | Independently ran the real binary (not trusting the SUMMARY): a maintainer-shaped fixture (2-space-indented `[mcp_servers.codegraph]`, sibling multi-line-array/inline-table entries, a UTF-8 BOM, a far-away `[memories]` header) round-trips through `install`/`uninstall --target codex --location global` byte-for-byte outside codegraph's own table, BOM included. A fresh local install in a real git repo wrote `.codex/config.toml`, `AGENTS.md` (marker block), `.agents/skills/codegraph/{SKILL.md,.codegraph-manifest.json}`, and printed the D-10 trust note verbatim. `go test ./internal/agents/... ./internal/cli/...` passes (11.965s/26.108s, re-run this session). |
| 3 | **SC-3 (CODEX-05):** A Codex PreToolUse nudge carries the additionalContext-only, first-fires-then-60s-cooldown-per-session-and-subagent contract; opt-in only via `--pretool-nudge`; skipped with a message when `[features] hooks = false`; docs state the `/hooks` trust requirement. | ✓ VERIFIED | Live evidence: `07-LIVE-SESSIONS.md` line 941 `CODEX-05 live verdict: PASS` (L6 one fire + 60s+ cooldown + 0 fires un-indexed, L7). Re-ran the real binary: `install --target codex --location local --pretool-nudge --yes` wrote a quoted-from-day-one `hooks.json` (`"$(git rev-parse --show-toplevel)/.codex/hooks/codegraph-pretooluse.sh"`, matcher `^Bash$`, one handler, `timeout: 5`) and printed the D-19 `/hooks` trust note verbatim. `internal/agents/codex_pretooluse_test.go` and `internal/cli/hook_pretooluse_codex_test.go` pass. |
| 4 | **SC-4 (CODEX-06):** A genuinely fresh Codex session in an indexed repo reaches for codegraph unprompted (skill listed, MCP tool called or `codegraph explore` run); `codex mcp list` shows the entry at both scopes. | ✓ VERIFIED | `07-LIVE-SESSIONS.md` line 942 `CODEX-06 verdict: PASS` (L1 post-flip both scopes, L5 fresh session reaches for codegraph unprompted, L7, all PASS with verbatim excerpts and negative-space logging per D-06). |
| 5 | **SC-5a (AGENT-14):** The published per-harness capability table matches what ships (`[ASSUMED]` where unverifiable), kept honest by a drift test; `instructions.go`'s "4 of 8" comment and the MCP `instructions` skill sentence are updated to match what ships. | ✓ VERIFIED | `docs/AGENT-CAPABILITIES.md` exists (92 lines, 16 rows), linked from `README.md:129`. `go test ./internal/agents/ -run TestCapabilityDoc` passes (`TestCapabilityDoc_MirrorsCapabilities`, `TestCapabilityDoc_VerificationColumn`, re-run this session). `internal/mcp/server.go`'s `instructions` const carries the harness-neutral sentence "codegraph install also adds the codegraph skill for every agent it configures except Hermes" — independently measured at char offset 207 (well inside the 512-byte window), whole const 582 bytes, matching the SUMMARY's claim. |
| 6 | **SC-5b (FIX-03):** The install/uninstall agent picker renders its help footer within a 100x30 pane with all 8 targets listed, asserted by the tmux harness *after* CODEX-02's scope flip. | ✅ PASSED (override, accepted by sean 2026-09-19) | The underlying defect is fixed and proven at the model level: `go test ./internal/cli/tui/... -run 'TestAgentPickerFootprint\|TestDaemonPickerFootprint'` passes all subtests (height boundaries 1/2/29/30/31, empty roster), re-run this session. `07-MUTATION-LOG.md` Family (c1)/(c2) show the guard RED on the pre-fix delegate. **However**, the requirement's own literal text (REQUIREMENTS.md FIX-03: "asserted by the tmux harness, re-run after the milestone's last target-count change") calls for a real-PTY execution, and that execution has not happened anywhere: the maintainer explicitly waived the local run (07-MUTATION-LOG.md Family (c3), STATE.md line 434/486), and the branch is unpushed (`git log origin/gsd/v0.14.0-milestone` errors — no remote tracking ref exists), so the CI `tmux-e2e` fallback job has never run against this code either. `go vet -tags tmux ./test/tmux/...` confirms the re-anchored assertion at least compiles. This is a state/behavior claim (real terminal rendering) that a model-level unit test cannot stand in for — see `behavior_unverified_items`. |

**Score:** 5/6 truths verified (1 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `internal/agents/toml.go` | any-indent, string/array-aware TOML scanner + conflict refusal + BOM handling | ✓ VERIFIED | `findTOMLTableRange`, `tomlTableConflict`, `tomlLineEnding`, BOM pseudo-line handling all present; real-binary round trip confirmed independently (maintainer-shaped + BOM-only fixtures) |
| `internal/agents/codex.go` | dual-scope `Capabilities()`, table-driven Install/Uninstall, corrected doc comment | ✓ VERIFIED | `Capabilities()` declares `Scopes: {global, local}`, `Hooks: HooksCodexJSON`, `MCPConfig/Instructions/SkillDirs` as PathFuncs; doc comment cites 07-LIVE-SESSIONS.md with a date |
| `internal/agents/codex_pretooluse.go`, `.codex/hooks/*` | Codex PreToolUse guard templates + lifecycle (On/Keep/Off) | ✓ VERIFIED | Files exist on disk (`.codex/hooks/hooks.json`, `*-local.sh`, `*-global.sh`, mode 755); real local install wrote the quoted D-20 command form and D-19 trust note |
| `internal/cli/hook_pretooluse.go` | `--harness codex` envelope | ✓ VERIFIED | `internal/cli/hook_pretooluse_codex_test.go` passes; wired into `internal/cli/tui`/install paths |
| `docs/AGENT-CAPABILITIES.md` | drift-tested 16-row capability table | ✓ VERIFIED | Exists, drift test passes, README links it |
| `internal/mcp/server.go` instructions const | harness-neutral skill sentence within 512 bytes | ✓ VERIFIED | Independently measured; wire-oracle suite green (57.966s, re-run this session) |
| `07-MUTATION-LOG.md` | 31 positive-controlled mutation families (a1-j3) | ✓ VERIFIED | All 31 family headers present (`rg -n "^## Family"` confirms a1-a3, b1-b2, c1-c3, d1-d4, e1-e3, f1-f5, g1-g4, h1-h2, i1-i2, j1-j3) |
| `.planning/WINDOWS.md` row (D-08) | released-binary data-loss window recorded | ✓ VERIFIED | id 38, kind `deviation`, status `open` (correctly still open per maintainer decision B2 — closes only when v0.14.0 ships) |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `internal/agents/codex.go` Install/Uninstall | `internal/agents/toml.go` splice/strip/conflict | table-driven path resolution | ✓ WIRED | Confirmed by real-binary tracer: conflict refusal, BOM preservation, and byte-identical sibling tables all observed live |
| `internal/agents/codex.go` Install | `internal/agents/shared.go instructionsRequestedElsewhere` | shared `AGENTS.md` ownership gate | ✓ WIRED | `TestSharedAgentsMD_UninstallOrders`/`TestOwnershipSharedInstructions` pass; live install/uninstall sequence not independently re-run this session but unit-tested and unchanged since 07-06 |
| `.codex/hooks.json` (installed) | `.codex/hooks/codegraph-pretooluse.sh` → `codegraph hook pretooluse --harness codex` | registered command → guard → CLI envelope | ✓ WIRED | Live-verified in 07-04/07-09 (A1: shell-expanded, fires); re-confirmed structurally this session (hooks.json contents match D-20 exactly) |
| `docs/AGENT-CAPABILITIES.md` | `internal/agents/capabilities.go Capabilities()` | drift test | ✓ WIRED | `TestCapabilityDoc_MirrorsCapabilities` passes |
| `test/tmux/install_cancel_test.go` TTY-05 | real-PTY tmux pane | `task test:tmux` / CI `tmux-e2e` | ⚠️ UNEXECUTED | Compiles (`go vet -tags tmux`); no local or CI run exists at this HEAD (see truth #6) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Build succeeds | `GOTOOLCHAIN=go1.26.6 go build ./...` | exit 0 | ✓ PASS |
| Full agents/cli/mcp/wireoracle suite | `go test ./internal/agents/... ./internal/cli/... ./internal/mcp/... ./test/wireoracle/... -count=1` | all `ok` (11.9s/26.1s/15.1s/58.0s) | ✓ PASS |
| Maintainer-shaped TOML round trip (real binary) | `install`/`uninstall --target codex --location global` on a synthetic indented+BOM fixture | siblings byte-identical, BOM preserved, codegraph table cleanly removed | ✓ PASS |
| BOM-only residual keep-clean (WR-03) | `install` then `uninstall` on a BOM-only config.toml | file removed entirely (not left as `﻿\n`) | ✓ PASS |
| Local install with `--pretool-nudge` (real binary) | `install --target codex --location local --pretool-nudge --yes` in a fresh git repo | writes quoted D-20 hooks.json, D-10/D-19 notes, AGENTS.md, shared skill | ✓ PASS |
| Picker footer footprint (model-level) | `go test ./internal/cli/tui/... -run 'TestAgentPickerFootprint\|TestDaemonPickerFootprint' -v` | all subtests PASS | ✓ PASS |
| Capability-doc drift guard | `go test ./internal/agents/ -run TestCapabilityDoc -v` | both PASS | ✓ PASS |
| tmux real-PTY TTY-05 re-run post-flip | `task test:tmux` / CI `tmux-e2e` | not executed (no local tmux, branch unpushed) | ? SKIP → routed to human verification |

### Requirements Coverage

| Requirement | Source Plan(s) | Status | Evidence |
|---|---|---|---|
| CODEX-01 | 07-04 (declares), 07-05 (consumes) | ✓ SATISFIED | Live verdict PASS; doc comment corrected in codex.go |
| CODEX-02 | 07-01, 07-02, 07-05 | ✓ SATISFIED | TOML fix, `--yes`/`--target` order fix, scope flip all confirmed live and by test |
| CODEX-03 | 07-05 | ✓ SATISFIED | Shared skill package install confirmed at local scope by real binary; `TestCodex_SharedSkillPackage_LastRequester` passes |
| CODEX-04 | 07-06 | ✓ SATISFIED | Shared `AGENTS.md` ownership tests pass; D-11/D-12 mechanisms confirmed present |
| CODEX-05 | 07-07, 07-08, 07-09 | ✓ SATISFIED | Live verdict PASS; real-binary local install confirms hooks.json/guard/notes |
| CODEX-06 | 07-09 | ✓ SATISFIED | Live verdict PASS |
| FIX-03 | 07-03, 07-09 | ✅ SATISFIED (override) | Model-level fix proven RED→GREEN with mutation-log positive control; the requirement's own literal "asserted by the tmux harness" clause has no execution evidence anywhere at this HEAD (see truth #6) — **not silently passed** |
| AGENT-14 | 07-10, 07-11 | ✓ SATISFIED | Drift-tested capability doc published and linked; instructions const and stale comments corrected |

No orphaned requirements: all 8 IDs in the phase's declared scope (CODEX-01…06, FIX-03, AGENT-14) appear in at least one plan's `requirements:` frontmatter, matching `REQUIREMENTS.md`'s Phase 7 assignment.

### Anti-Patterns Found

None blocking. Notable, already-documented, accepted deviations (not defects):
- `internal/agents/codex.go`'s doc comment still contains the sentence "Hooks stay HooksNone this plan — 07-07 adds codex-json" — a stale comment fragment; the actual `Hooks: HooksCodexJSON` field is correctly set (confirmed by direct read of the struct literal). Cosmetic only — ℹ️ Info, not a functional gap.
- `test/tmux/poll_contract_test.go` gofmt drift — pre-existing, out of scope for this phase (07-03-SUMMARY.md), unrelated to Phase 7's changes.
- `CODEX_HOME` not honored for global paths — documented advisory (07-CONTEXT.md, D-flagged), not a defect.
- WINDOWS ledger row (D-08) correctly still `open` — this is expected, not a gap; it closes only when the v0.14.0 release ships per maintainer decision B2.

### Human Verification Required

#### 1. FIX-03's post-flip real-PTY tmux assertion

**RESOLVED 2026-09-19 — override accepted by the maintainer (sean):** the model-level footprint guard is accepted as sufficient evidence for this phase; CI's `tmux-e2e` job remains the outstanding real-PTY confirmation and runs on push/PR. Recorded in the frontmatter `overrides:` block, in STATE.md, and in GitHub issue #75. No further action blocks Phase 7.

**Test:** Push the branch (or otherwise get CI running) and observe the `tmux-e2e` GitHub Actions job, or manually run `task test:tmux` on a machine with tmux installed, at this HEAD.
**Expected:** TTY-05 (re-anchored on `space: toggle` + all 8 target display names) reports `executed=6 skipped=0` with no failures, confirming the picker footer overflow fix holds under a genuine terminal (escape-sequence handling, bubbletea's real render loop, pagination) — not just the bubbletea *model* the unit tests exercise.
**Why human:** This is a real-PTY behavioral claim that cannot be verified by static analysis or by running the model-level Go tests (which I already re-ran and confirmed pass). The maintainer's own 2026-09-19 decision explicitly deferred this evidence to CI (GitHub issue #75), and the branch has not yet been pushed, so CI has never run against this code. A human must either (a) push the branch and let CI settle this, or (b) explicitly decide the model-level guard (Families c1/c2, GREEN) is sufficient evidence to accept FIX-03 as fully met without the literal tmux re-run — in which case an override should be recorded in this file's frontmatter.

### Gaps Summary

No BLOCKER-level gaps. Every must-have artifact exists, is substantive, and is wired; the full regression suite (agents/cli/mcp/wireoracle) passes; the real binary was independently exercised for the TOML splice fix (including the BOM edge case from the code review's CR-01/WR-03 fixes), the local Codex install with the opt-in nudge, and the capability-doc drift guard — all behaving exactly as documented, not merely as claimed.

The one open item is FIX-03's literal "asserted by the tmux harness" clause, which is present-and-compiling but has no execution evidence anywhere (not locally — maintainer-waived — and not in CI — branch unpushed). This is honestly reported here rather than silently passed, per the orchestrator's explicit instruction. It routes to `human_needed` rather than `gaps_found` because: the underlying functionality is proven via strong alternate evidence (a positive-controlled model-level test suite plus a real-binary difference check), the deviation was an explicit, documented maintainer decision (not an oversight), and a CI job exists and is ready to run the moment the branch is pushed.

---

_Verified: 2026-09-19_
_Verifier: Claude (gsd-verifier)_
