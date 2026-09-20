---
phase: 02-guards-ci-wiring-docs-burn-down
fixed_at: 2026-09-16T11:47:47Z
review_path: .planning/phases/02-guards-ci-wiring-docs-burn-down/02-REVIEW.md
iteration: 1
findings_in_scope: 1
fixed: 1
skipped: 0
status: all_fixed
---

# Phase 2: Code Review Fix Report

**Fixed at:** 2026-09-16T11:47:47Z
**Source review:** .planning/phases/02-guards-ci-wiring-docs-burn-down/02-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 1 (WR-01 — critical_warning scope; IN-01 and IN-02 are Info, out of scope, see below)
- Fixed: 1
- Skipped: 0

**Verification environment:** main checkout (`gsd/v0.14.0-milestone`), not an isolated worktree — `workflow.use_worktrees` is `false` in `.planning/config.json`, so per the fixer's honored opt-out, editing/committing/verifying happened directly in the working tree, no worktree was created, and no worktree cleanup tail applies.

## Fixed Issues

### WR-01: `check-ruleset-drift.sh`'s live-ruleset fetch has no timeout

**Files modified:** `scripts/check-ruleset-drift.sh`
**Commit:** `97bb6a13`
**Applied fix:** Added `--connect-timeout 10 --max-time 30` to the script's `CURL_ARGS` array (the fetch of the live GitHub ruleset), so a network stall now fails loud within ~10-30s instead of hanging for however long the job-level CI timeout allows. No other behavior of the script changed — the fixture-emptiness precondition, the non-200/empty-body/unparseable-JSON/non-active/zero-live-contexts hard-fails, both printed counts, the exact-set-equality comparator, and the positive control (planted-context self-check run before the real comparison) are all byte-identical to before.

**Verification (4-part, rule 84d1gfpywd):**

1. **shellcheck** — `shellcheck scripts/check-ruleset-drift.sh` produced zero output (clean, no findings).

2. **RED** — ran the script against a black-hole address to prove the timeout actually bounds the hang:
   ```
   $ RULESET_URL_BASE="http://10.255.255.1" timeout 60 bash scripts/check-ruleset-drift.sh
   ruleset-drift: fixture .github/required-status-checks.txt lists 8 contexts
   curl: (28) Failed to connect to 10.255.255.1 port 80 after 10070 ms: Timeout was reached
   ::error::ruleset-drift: GET rulesets/20157557 returned HTTP 000 (or an empty body) — hard failure, never a skip
   EXIT: 1
   ( ... 10.101 total wall time)
   ```
   Exits non-zero in ~10.1s (bounded by `--connect-timeout 10`), naming the fetch failure via a proper `::error::` line — not a hang. `RULESET_URL_BASE` is a script-supported override (used directly, no wrapper needed).

3. **GREEN** — ran the script unmodified against the live GitHub API:
   ```
   $ bash scripts/check-ruleset-drift.sh
   ruleset-drift: fixture .github/required-status-checks.txt lists 8 contexts
   ruleset-drift: live has 8 contexts, fixture has 8 contexts
   ruleset-drift: self-check PASS — planted context detected
   ruleset-drift: PASS — 8 contexts identical
   EXIT: 0
   ( ... 0.203s total wall time)
   ```
   Still prints `live has 8 contexts, fixture has 8 contexts` and `ruleset-drift: PASS` with exit 0 — the timeout addition did not regress the normal path.

4. **Go tests** — `GOTOOLCHAIN=go1.26.6 go test ./internal/upgrade/ -count=1`:
   ```
   ok  	github.com/seanb4t/codegraph-go/internal/upgrade	0.421s
   ```
   Confirms the taskfile-shape tests (which read `ci.yml`, not the script) are unaffected.

## Out of Scope (not fixed — Info severity, per fix_scope: critical_warning)

- **IN-01** (`go.sum:491-559` orphaned checksums from the grpc bump) — Info severity, excluded by `fix_scope`. Left untouched; the review's own fix note flags that `go mod tidy` requires network access to the module proxy and could not be verified in a sandboxed environment due to an unrelated pre-existing `tree-sitter-swift` import-path resolution issue.
- **IN-02** (`command-link-item.svelte` dormant `data-selected` coupling) — Info severity, excluded by `fix_scope`. The review's own fix note says no action is required now; re-verify only when `CommandLinkItem` gains its first real caller.

## Skipped Issues

None — the one in-scope finding (WR-01) was fixed cleanly.

---

_Fixed: 2026-09-16T11:47:47Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
