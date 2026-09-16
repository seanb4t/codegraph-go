---
phase: 02-guards-ci-wiring-docs-burn-down
reviewed: 2026-09-16T12:00:00Z
depth: deep
files_reviewed: 8
files_reviewed_list:
  - scripts/check-ruleset-drift.sh
  - .github/workflows/ci.yml
  - .github/required-status-checks.txt
  - internal/upgrade/taskfile_shape_test.go
  - go.mod
  - go.sum
  - docs/RELEASE.md
  - SECURITY.md
findings:
  critical: 0
  warning: 0
  info: 2
  total: 2
status: clean
---

# Phase 2: Code Review Report

**Reviewed:** 2026-09-16T12:00:00Z
**Depth:** deep
**Files Reviewed:** 8
**Status:** clean

## Summary

This is iteration 2 of `--auto` re-review. Per orchestrator instructions, the 24 regenerated `web/src/lib/components/ui/**` files reviewed in iteration 1 (`02-REVIEW.iter2.md`, preserved) were not touched by the intervening commit and are not re-reviewed here; their prior verdict (clean, IN-02 carried forward) stands. Only `97bb6a13` landed since iteration 1, touching exactly `scripts/check-ruleset-drift.sh` (1 line changed). All 8 files listed in this iteration's config were re-examined; the 7 files other than the script are byte-identical to iteration 1's review and their findings are carried forward unchanged.

**WR-01 fix verified, independently, not just read:**
- Source diff confirms the fix is exactly what the fix report claims: `CURL_ARGS` on `scripts/check-ruleset-drift.sh:91` gained `--connect-timeout 10 --max-time 30`, with no other line touched.
- `shellcheck scripts/check-ruleset-drift.sh` — reran it myself: exit 0, zero findings.
- Re-ran the RED case myself (not just trusting the fix report): `RULESET_URL_BASE=http://10.255.255.1 bash scripts/check-ruleset-drift.sh` under `time` — failed at `curl: (28) Failed to connect ... after 10073 ms: Timeout was reached`, printed the script's own `::error::...GET rulesets/20157557 returned HTTP 000 (or an empty body)...` line, and exited 1 in 10.099s wall time. Matches the fix report's claimed RED exactly.
- Traced the control flow by hand for the failure-mode the reviewer instructions specifically asked to check: `RESPONSE="$(curl "${CURL_ARGS[@]}" ... || true)"` is a command substitution assigned to a variable, not a pipeline — `pipefail` (which only affects pipeline exit-status propagation) does not apply to it, and the explicit `|| true` independently prevents `set -e` from firing on a non-zero `curl` exit status inside the substitution. On a hard connection failure (as reproduced above), `curl` writes nothing to stdout, so `RESPONSE` is empty, `HTTP_CODE` (`tail -n1`) is empty, and `BODY` (`sed '$d'`) is empty — both arms of `[ "${HTTP_CODE}" != "200" ] || [ -z "${BODY}" ]` are satisfied and the script takes its named hard-fail branch, never falling through to treat an empty/timed-out response as a valid (and vacuously matching or non-matching) body. On a mid-transfer `--max-time` abort, `curl` exits nonzero before writing its `-w` suffix, so `RESPONSE` ends up as a body fragment with no trailing `200` line; `HTTP_CODE` then equals the last (partial) body line rather than `200`, so the `!= "200"` arm alone still routes it to the same hard-fail branch. No path silently reads a stalled/incomplete fetch as pass or as a vacuous set match.
- `GREEN` (no test double needed — a live GitHub API call is legitimate here since this is an unauthenticated read of a public repo's ruleset, no secret involved): `GOTOOLCHAIN=go1.26.6 go build ./...` passes, and `go test ./internal/upgrade/...` (which reads `ci.yml`, not the script) is green, confirming the fix didn't regress anything the taskfile-shape tests cover.
- No new issue introduced by the fix: the two-character flag addition doesn't change argument order in a way that affects `-w`'s placement or output, doesn't touch the `GITHUB_TOKEN` conditional append, and doesn't affect the trap/self-check/comparator logic below it.

WR-01 is resolved. No new Critical or Warning findings surfaced in this iteration, in the script or the other 7 unchanged files (re-verified: `go build`, `.github/required-status-checks.txt` still lists 8 contexts matching `ci.yml`'s job names, `check-ruleset-drift.sh` is still wired into `ci.yml:243` as a CI-only step with no Taskfile wrapper).

The two Info items from iteration 1 are unresolved (both explicitly out of the `critical_warning` fix scope per the fix report) and are carried forward below.

## Info

### IN-01: Orphaned go.sum checksums left over from the targeted grpc bump

**File:** `go.sum:489-558`
**Issue:** Re-confirmed present and unchanged: `go.sum` still carries `h1:`/`go.mod h1:` pairs for the pre-bump versions (`golang.org/x/crypto v0.54.0`, `x/net v0.57.0`, `x/text v0.40.0`, `google.golang.org/grpc v1.82.1`, and the older-dated `google.golang.org/genproto/googleapis/rpc v0.0.0-20260523011958-...` pseudo-version) alongside the new versions actually required by `go.mod` (`v0.55.0`, `v0.58.0`, `v0.41.0`, `v1.83.2`, `v0.0.0-20260526163538-...`). `go build`/`go mod verify` succeed either way — not a correctness defect — but it indicates `go mod tidy` wasn't run after the targeted `go get` bump.
**Fix:** Run `go mod tidy` (requires module-proxy network access) and re-commit the pruned `go.sum`.

### IN-02: `command-link-item.svelte` selected-state styling now depends on an attribute with no confirmed caller

**File:** `web/src/lib/components/ui/command/command-link-item.svelte:16`
**Issue:** Unchanged since iteration 1 (file not touched by the intervening commit); carried forward for completeness only. Already disclosed in 02-05-SUMMARY.md: the selected-state classes were switched from `aria-selected:*` to `data-selected:*`. `CommandLinkItem` has no caller anywhere in `web/src` outside its own registry `index.ts` export today, so the change is currently dormant.
**Fix:** None required now. Re-verify the selected-state highlight renders correctly the first time `<Command.LinkItem>` gains a real caller.

---

_Reviewed: 2026-09-16T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
