---
phase: 07-codex-parity
fixed_at: 2026-09-19T00:00:00Z
review_path: .planning/phases/07-codex-parity/07-REVIEW.md
iteration: 1
findings_in_scope: 2
fixed: 2
skipped: 0
status: all_fixed
---

# Phase 07: Code Review Fix Report

**Fixed at:** 2026-09-19T00:00:00Z
**Source review:** .planning/phases/07-codex-parity/07-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 2 (CR-01, WR-01 — per explicit fix-scope instruction; IN-01 excluded from scope, recorded deferred below, not attempted)
- Fixed: 2
- Skipped: 0

**Verification environment:** main checkout at `/Volumes/Code/github.com/seanb4t/codegraph-go`, branch `gsd/v0.14.0-milestone` — no worktree used (`workflow.use_worktrees: false` in `.planning/config.json`).

## Fixed Issues

### CR-01: A UTF-8 BOM at the start of a Codex config.toml defeats header recognition, producing a duplicate `[mcp_servers.codegraph]` table on install and a permanently un-removable entry on uninstall

**Files modified:** `internal/agents/toml.go`, `internal/agents/toml_test.go`, plus new fixtures `internal/agents/testdata/toml/codex-bom{,.installed,.uninstalled}.toml`
**Commits:**
- `12d4c0cb` — `test(07-12): RED — UTF-8 BOM defeats TOML header recognition (CR-01)`
- `01e8306d` — `fix(07-12): stop a leading UTF-8 BOM from defeating TOML header recognition (CR-01)`

**RED proof** (`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestFindTOMLTableRange_UTF8BOM$|TestSpliceTOMLTable_UTF8BOM_UpdatesInPlaceNeverDuplicates$|TestStripTOMLTable_UTF8BOM_RemovesEntirelyPreservingBOM$|TestSpliceTOMLTable_UTF8BOM_Fixture$|TestStripTOMLTable_UTF8BOM_Fixture$' -v`):
```
toml_test.go:188: findTOMLTableRange: found = false, want true (a leading BOM must not defeat header recognition)
--- FAIL: TestFindTOMLTableRange_UTF8BOM (0.00s)
    toml_test.go:200: spliceTOMLTable on a BOM'd file produced 2 [mcp_servers.codegraph] headers, want exactly 1 (CR-01 duplicate-table regression)
--- FAIL: TestSpliceTOMLTable_UTF8BOM_UpdatesInPlaceNeverDuplicates (0.00s)
    toml_test.go:218: stripTOMLTable must remove codegraph's own table from a BOM'd file
--- FAIL: TestStripTOMLTable_UTF8BOM_RemovesEntirelyPreservingBOM (0.00s)
    toml_test.go:252: spliceTOMLTable BOM-fixture mismatch (got 2 headers)
--- FAIL: TestSpliceTOMLTable_UTF8BOM_Fixture (0.00s)
    toml_test.go:262: stripTOMLTable BOM-fixture mismatch (got the un-stripped table)
--- FAIL: TestStripTOMLTable_UTF8BOM_Fixture (0.00s)
FAIL
```
(`TestFindTOMLTableRange_MidFileBOMNotStrippedAsWhitespace` — the negative-space control — passed against unfixed code, as expected.)

**Applied fix:** Fixed at the cause identified by the review: `isTOMLHeaderLine`'s
`strings.TrimLeft(line, " \t")` never stripped a leading UTF-8 BOM (`\xef\xbb\xbf`), so a
config.toml opening with one — the common shape for a project-local `.codex/config.toml`
whose first content line is codegraph's own table — was never recognized as a header.
Rather than trim the BOM (which would require offset-adjusting every downstream byte-range
calculation), `splitTOMLLines` — the single point of entry `findTOMLTableRange`,
`tomlTableConflict` and `tomlBoolSetting` all funnel through — now emits content's absolute
leading BOM as its own synthetic zero-th pseudo-line (offset 0, the 3 BOM bytes, never itself
eligible to be a header or spliced over) when present, leaving every other line's byte offset
unaffected. The BOM is never rewritten — it round-trips byte-for-byte through
splice/strip/append, satisfying "a BOM present on input must still be present on output."
Only content's true, absolute leading BOM is special-cased: a BOM byte sequence appearing on
any later line is ordinary line content, never treated as leading whitespace to strip before
recognizing a header (pinned by `TestFindTOMLTableRange_MidFileBOMNotStrippedAsWhitespace`).

Verified by:
- The 6 new tests above, now green.
- Fixture-driven regression (`codex-bom.toml` → `codex-bom.installed.toml` →
  `codex-bom.uninstalled.toml`), pinning the round trip byte-exact rather than only via
  substring checks.
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/... ./internal/mcp/ -count=1`
  — all packages `ok` (including every pre-existing D-07 TOML test, e.g.
  `TestSpliceTOMLTable_MaintainerIndentedLayout`, `TestTOMLTableConflict`,
  `TestSpliceTOMLTable_CRLFPreserved`).
- Positive-control mutation: 07-MUTATION-LOG.md Family (j1) — disabling the BOM
  special-case reproduces the exact RED transcript above; reverted byte-clean, re-confirmed
  green.

### WR-01: `writeHookEntry` repositions a foreign Codex hook block on any *update* to codegraph's own PreToolUse registration, not just on removal — spuriously invalidating Codex's position-keyed hook trust for an untouched hook

**Files modified:** `internal/agents/shared.go`, `internal/agents/shared_test.go`
**Commits:**
- `c1b17930` — `test(07-12): RED — writeHookEntry repositions a foreign hook on update (WR-01)`
- `2ac15d05` — `fix(07-12): preserve foreign hook block position on writeHookEntry update (WR-01)`

**RED proof** (`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestWriteHookEntry_UpdatePreservesForeignBlockPosition$' -v`):
```
shared_test.go:763: expected codegraph's own (updated) block to stay at index 0, got command "/foreign/hook.sh": [...]
--- FAIL: TestWriteHookEntry_UpdatePreservesForeignBlockPosition (0.00s)
FAIL
```
(`TestWriteHookEntry_FirstInstallAppendsAfterForeignBlocks` — the D-23 control — passed
against unfixed code, as expected, since the first-install path was never the defect.)

**Applied fix:** Per the review's Fix guidance exactly: `writeHookEntry` now tracks, during
its single partition pass over the existing event array, the index the first owned block
originally occupied among the *unowned* blocks (`ownedInsertAt`). When rebuilding the array,
if an owned block existed before this call, the new normalized owned blocks are spliced back
in at that same index — preserving the position of any foreign block that already followed
codegraph's group — instead of unconditionally appending them after every unowned block. A
genuine first install (no owned block existed before the call, `ownedInsertAt == -1`) is
unaffected and still appends codegraph's group after every existing block, preserving D-23's
"appended last on first install" guarantee.

Verified by:
- The 2 new tests above, now green.
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/... ./internal/mcp/ -count=1`
  — all packages `ok`.
- Explicitly re-confirmed green as named in the fix instructions: `TestPreToolUseRegistrationShape`
  (both `settings.json` and `hooks.json` leaves) and all 32 `TestOwnershipExactIdentity` leaves.
- Positive-control mutation: 07-MUTATION-LOG.md Family (j2) — reverting to the pre-fix
  unconditional append-last reproduces the exact RED transcript above; reverted byte-clean,
  re-confirmed green.

## Deferred Issues (excluded from fix scope by explicit instruction)

### IN-01: `codexShellCommand`'s argv-form decoding takes the last array element, which is only correct for a `[shell, flag, full-command]` wrapper shape — an untested assumption

**File:** `internal/cli/hook_pretooluse.go:72-95` (`codexShellCommand`)
**Status:** deferred, not fixed — explicitly out of this fix's scope (`CR-01` and `WR-01`
only). This also matches the finding's own Fix guidance: no code change is warranted before
Codex's argv shape is ever observed live, since the "any doubt, silent" contract already
makes the failure mode safe (a missed nudge, never a block or crash). Left for a natural
follow-up alongside CODEX-05/06's next live session, at which point a positive-control
fixture pinning the observed shape should be added and the doc comment corrected from "argv
(the last element)" to the confirmed shape.

## Skipped Issues

None — both in-scope findings (CR-01, WR-01) were fixed.

## Mutation log

07-MUTATION-LOG.md gained two new families for this fix:
- Family (j1) — CR-01's UTF-8 BOM handling in `splitTOMLLines`.
- Family (j2) — WR-01's owned-block position preservation in `writeHookEntry`.

---

_Fixed: 2026-09-19T00:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
