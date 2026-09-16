---
phase: 02-guards-ci-wiring-docs-burn-down
reviewed: 2026-09-16T00:00:00Z
depth: deep
files_reviewed: 30
files_reviewed_list:
  - .github/required-status-checks.txt
  - .github/workflows/ci.yml
  - SECURITY.md
  - docs/RELEASE.md
  - go.mod
  - go.sum
  - internal/upgrade/taskfile_shape_test.go
  - scripts/check-ruleset-drift.sh
  - web/src/lib/components/ui/button/button.svelte
  - web/src/lib/components/ui/command/command-group.svelte
  - web/src/lib/components/ui/command/command-input.svelte
  - web/src/lib/components/ui/command/command-item.svelte
  - web/src/lib/components/ui/command/command-link-item.svelte
  - web/src/lib/components/ui/command/command-separator.svelte
  - web/src/lib/components/ui/command/command-shortcut.svelte
  - web/src/lib/components/ui/command/command.svelte
  - web/src/lib/components/ui/dialog/dialog-content.svelte
  - web/src/lib/components/ui/dialog/dialog-description.svelte
  - web/src/lib/components/ui/dialog/dialog-overlay.svelte
  - web/src/lib/components/ui/input-group/input-group-addon.svelte
  - web/src/lib/components/ui/input-group/input-group-button.svelte
  - web/src/lib/components/ui/input-group/input-group-text.svelte
  - web/src/lib/components/ui/input-group/input-group.svelte
  - web/src/lib/components/ui/input/input.svelte
  - web/src/lib/components/ui/table/table-caption.svelte
  - web/src/lib/components/ui/table/table-footer.svelte
  - web/src/lib/components/ui/table/table-head.svelte
  - web/src/lib/components/ui/table/table-row.svelte
  - web/src/lib/components/ui/tabs/tabs-list.svelte
  - web/src/lib/components/ui/tabs/tabs-trigger.svelte
  - web/src/lib/components/ui/tabs/tabs.svelte
  - web/src/lib/components/ui/textarea/textarea.svelte
findings:
  critical: 0
  warning: 1
  info: 2
  total: 3
status: issues_found
---

# Phase 2: Code Review Report

**Reviewed:** 2026-09-16T00:00:00Z
**Depth:** deep
**Files Reviewed:** 30
**Status:** issues_found

## Summary

Reviewed the full Phase 2 file set at deep depth: the new `scripts/check-ruleset-drift.sh` guard and its `ci.yml`/`taskfile_shape_test.go` wiring, the `go.mod`/`go.sum` grpc/x-net/x-crypto/x-text bump, the `docs/RELEASE.md` and `SECURITY.md` doc edits, and the 24 regenerated shadcn-svelte UI components.

**Authored guard surface** (`scripts/check-ruleset-drift.sh`, the `taskfile_shape_test.go` diff, the three new `ci.yml` steps): sound. Verified independently, not just read:
- `shellcheck scripts/check-ruleset-drift.sh` — clean, zero findings.
- `actionlint .github/workflows/ci.yml` — clean, zero findings.
- Traced every failure path by hand (missing/empty fixture, non-200/empty body, unparseable JSON via `jq -e`, wrong ruleset name, non-active enforcement, zero live contexts, real mismatch) — each is a named `::error::` and hard `exit 1`, in the correct order (fixture emptiness is checked *before* the network call, per the script's own stated invariant). `set -euo pipefail` is in effect and the two places that need to defeat it (`curl ... || true`, `diff ... || true`) do so deliberately and only where needed; the `jq | sort` pipelines correctly propagate a `jq` failure to the caller under `pipefail` (verified this isn't a false green — `sort` alone would otherwise mask a `jq` error).
- The built-in positive control (plant an extra context, assert the comparator reports drift) runs *after* the zero-live-contexts guard, so it can't itself pass vacuously against an empty live set.
- `.github/required-status-checks.txt`'s 8 lines match `ci.yml`/`pr-title.yml` job `name:` strings byte-for-byte (checked with `rg`), and the new `readRequiredCheckNames` loader/duplicate-detection/exception-list changes in `taskfile_shape_test.go` are covered by dedicated new tests (missing file, blank file, CRLF trimming, duplicate detection). The new `runBodyExceptions` entry is a single step in a single job with a real, matched reason, exactly as required.
- `go.mod`/`go.sum`: confirmed the diff is scoped to exactly `golang.org/x/crypto`, `x/net`, `x/text`, `google.golang.org/genproto/googleapis/rpc`, and `google.golang.org/grpc` (the GO-2026-6348 bump chain) — no unrelated requires moved. `GOTOOLCHAIN=go1.26.6 go build ./...` passes clean.
- `docs/RELEASE.md`: no raw dependency counts remain (grepped for the old `134`/`107`/`27 direct`/`13 direct` figures — gone), and every package named in the rewritten prose (`modelcontextprotocol/go-sdk`, `gonum.org/v1/gonum`, the `charm.land/*` TUI stack, `google.golang.org/protobuf`) is confirmed present as a direct `require` in `go.mod`. `SECURITY.md` gained exactly the one described sentence, names no specific CVE/exposure.
- The 24 `web/src/lib/components/ui/**` files: diffed each against `diff_base`. All 24 are explainable as Tailwind v4 class-string reordering/arbitrary-value spacing normalization (`calc(100%-1px)` → `calc(100%_-_1px)`, `group-data-[orientation=horizontal]` → `group-data-horizontal`) or removal of unused internal `cn-*` marker classes (confirmed zero references anywhere else in `web/`, including CSS/tests). The three changes 02-05-SUMMARY.md names as behavioral (`button.svelte` hover color-mix, `command-link-item.svelte` selected-state scheme, `table-row.svelte`'s additive `has-aria-expanded:bg-muted/50`) are the only ones with rendered effect, and none has any live caller that would be broken by it (`table-row` addition is dormant — no caller sets `aria-expanded` on a row; `command-link-item` is exported but has no current caller in `web/src` outside its own registry index).

Two minor, non-blocking issues found in the authored guard script and the dependency bump; details below.

## Warnings

### WR-01: `check-ruleset-drift.sh`'s live-ruleset fetch has no timeout

**File:** `scripts/check-ruleset-drift.sh:91-96`
**Issue:** The `curl` invocation that fetches the live ruleset carries `-sS` and a `-w` format but no `--max-time`/`--connect-timeout`. A network stall (e.g., a GitHub API hang or a routing black-hole from the runner) blocks this step for however long the job-level timeout allows, rather than failing fast the way every other failure path in this script is designed to (every other branch is a named `::error::` within milliseconds). This is a robustness gap in a script whose entire stated design goal is "never a skip, never a silent pass, hard-fail loud" — an indefinite hang is a softer failure mode than the ones this script otherwise refuses to allow, and it burns CI minutes on the shared `test` job rather than the isolated `Ruleset drift check` step alone.
**Fix:**
```bash
CURL_ARGS=(-sS --connect-timeout 10 --max-time 30 -w '\n%{http_code}' -H 'Accept: application/vnd.github+json' -H 'X-GitHub-Api-Version: 2022-11-28')
```

## Info

### IN-01: Orphaned go.sum checksums left over from the targeted grpc bump

**File:** `go.sum:491-559`
**Issue:** The bump replaced `golang.org/x/crypto v0.54.0`, `x/net v0.57.0`, `x/text v0.40.0`, `google.golang.org/grpc v1.82.1`, and the old-dated `google.golang.org/genproto/googleapis/rpc` pseudo-version with their new versions in `go.mod`, but `go.sum` still carries both the old and new `h1:`/`go.mod h1:` pairs for all five modules (10 leftover lines no longer referenced by any `require` line). `go build`/`go mod verify` succeed either way, so this isn't a correctness defect, but it's a sign `go mod tidy` wasn't run after the targeted `go get` bump — a future contributor diffing `go.sum` may wonder why the pre-bump versions still verify.
**Fix:** Run `go mod tidy` (network access to the module proxy required — this could not be verified in this sandbox because an unrelated pre-existing `tree-sitter-swift` test-only import path fails to resolve via `go mod tidy` here) and re-commit the pruned `go.sum`.

### IN-02: `command-link-item.svelte` selected-state styling now depends on an attribute with no confirmed caller

**File:** `web/src/lib/components/ui/command/command-link-item.svelte:16`
**Issue:** Already disclosed and functionally verified in 02-05-SUMMARY.md, noted here only for completeness: this file's selected-state classes were switched from `aria-selected:*` to the identical `data-selected:*` block `command-item.svelte` uses. `CommandLinkItem` has no caller anywhere in `web/src` outside its own registry `index.ts` export today, so this behavioral change is currently dormant in the shipped app — it will only matter the first time something actually renders a `<Command.LinkItem>`. Not a blocker; flagging so the dormant coupling is visible if/when a caller is added.
**Fix:** None required now. When `CommandLinkItem` gains its first real caller, re-verify the selected-state highlight renders (bits-ui's `CommandPrimitive.LinkItem` should set the same `data-selected` attribute `CommandPrimitive.Item` does, but this repo has no rendered proof of that yet).

---

_Reviewed: 2026-09-16T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
