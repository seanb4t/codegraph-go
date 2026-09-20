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

## Re-review (pass 2)

**Reviewed:** 2026-09-19
**Scope:** CR-01 (`internal/agents/toml.go`, `splitTOMLLines` and its three consumers) and
WR-01 (`internal/agents/shared.go`, `writeHookEntry`) — commits `12d4c0cb`, `01e8306d`,
`c1b17930`, `2ac15d05` — plus any regression they could introduce in their callers.
**Verdict:** issues found (1 new Warning). Both fixes are correct and safe as far as the
originally-scoped defects go; one incomplete-fix gap in CR-01's Uninstall path was found and
verified by direct execution. WR-01 holds with no findings.

### CR-01 — verified correct for header/splice recognition; one incomplete-fix gap found in the Uninstall "file became empty" path (see WR-03)

`splitTOMLLines`'s synthetic zero-th BOM pseudo-line is byte-offset-safe: the pseudo-line's
own `offset` is 0 and every other line's offset is computed exactly as before (the BOM is
skipped only as a starting cursor position, `content` itself is never mutated), so
`content[start:end]` slicing in `spliceTOMLTable`/`stripTOMLTable`, `tomlTableConflict`'s
reported line numbers (via `tomlConflictError`'s offset-based lookup), and `tomlBoolSetting`'s
table/key matching all remain correct. Traced and confirmed by direct execution or inspection
for every case in scope:

- BOM-only file (3 bytes, no trailing newline): `splitTOMLLines` returns just the pseudo-line;
  no header found, no panic.
- BOM + immediate EOF / BOM + blank line / BOM + comment / BOM + `[[array]]`: all produce a
  correctly-offset first real line at byte 3; `isTOMLHeaderLine`/`isTOMLBlankOrCommentLine`
  behave identically to the no-BOM case.
- BOM + CRLF: `tomlLineEnding`'s `strings.Contains(content, "\r\n")` check is BOM-agnostic and
  correct; the header line's `tomlLine.text` carries its trailing `\r` exactly as the existing
  CRLF handling expects, and `TrimSpace`'s `\r`-is-whitespace behavior makes header comparison
  work unchanged.
- File whose first content line is codegraph's own table (the originally reported case) vs.
  one where it isn't (e.g. `model = "o1"` before the header): both resolve the header at the
  correct absolute offset.
- A second, mid-file BOM: correctly NOT treated as a header or as strippable leading
  whitespace (pinned by `TestFindTOMLTableRange_MidFileBOMNotStrippedAsWhitespace`) — this is
  an explicit, intentional, already-reviewed design boundary (BOM special-casing is
  absolute-leading-only), not a new defect.
- A lone `\xef` or a truncated 1-2-byte BOM prefix: `strings.HasPrefix(content, tomlBOM)`
  requires an exact 3-byte match, so these fall through as ordinary first-line content,
  identical to pre-fix behavior for this input shape (no regression, no crash).
- A BOM "inside" a multi-line string: structurally impossible — the special-cased BOM can only
  ever be content's absolute first 3 bytes, and a multi-line string cannot open before byte 0.

No off-by-three, panic, or incorrect byte range was found in any of the above. The synthetic
pseudo-line is never eligible to be a header, never counted as blank/comment content for the
back-off scan (`isTOMLBlankOrCommentLine` is never invoked on it since the back-off loop only
walks from `endIdx`/`len(lines)` backward toward `headerIdx+1`, never as low as index 0 when a
real header line exists after it), and is never duplicated or deleted by `spliceTOMLTable` (it
sits before `start` in every case tested, so `content[:start]` always includes it verbatim).

### WR-03 (new, pass 2): CR-01's fix leaves a stray `BOM + "\n"` file behind on `codexTarget.Uninstall` when the file's only content was the BOM and codegraph's own table, while reporting the action as `ActionRemoved`

**File:** `internal/agents/toml.go:63-90` (`stripTOMLTable`), consumed by
`internal/agents/codex.go:296-322` (`codexTarget.Uninstall`, specifically the `updated == ""`
branch at `:309`)

**Issue:** `codex.go`'s `Uninstall` decides whether to delete the config file outright
(`os.Remove`, reported `ActionRemoved`) versus rewrite it (`atomicWriteFile`, also reported
`ActionRemoved`) by testing `updated == ""` — the "file became completely empty" precedent the
function's own doc comment calls out explicitly ("a strip that empties config.toml entirely
removes the file rather than leaving an empty one"). For a BOM-prefixed config.toml whose only
content is the BOM and codegraph's own table — the exact realistic shape CR-01 itself used to
justify severity (07-LIVE-SESSIONS.md: a project-local `.codex/config.toml` holding only the
`[mcp_servers.*]` tables) — `stripTOMLTable` now correctly finds and removes the table, but
because the BOM is preserved byte-for-byte per CR-01's own "round-trips byte-for-byte" contract,
the residual content is `"\ufeff\n"`, not `""`. This is pinned intentionally by CR-01's own
fixture test (`TestStripTOMLTable_UTF8BOM_Fixture` asserts the result equals
`codex-bom.uninstalled.toml`, whose bytes are exactly `ef bb bf 0a` — confirmed by direct
inspection) and independently corroborated by 07-MUTATION-LOG.md Family (j1)'s own transcript
(`want="\ufeff\n"`).

`updated == ""` is therefore false, so `Uninstall` takes the `default` branch: it rewrites the
file to contain just the BOM and a trailing newline via `atomicWriteFile`, rather than removing
it via `os.Remove` — yet still reports the action as `ActionRemoved`. The user is told the file
was removed; in fact a 4-byte stub file lingers on disk indefinitely (every subsequent
`Uninstall` will report `ActionNotFound` for it, since `stripTOMLTable` on `"\ufeff\n"` is
already a no-op — table not found — so `updated == existing`).

Verified by direct execution (temporary in-package test, removed after verification, no source
modified):

```go
content := "\xef\xbb\xbf[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n"
updated := stripTOMLTable(content, "mcp_servers.codegraph")
// updated = "\ufeff\n" (4 bytes) -- not "", not equal to content
// codex.go's Uninstall: updated != "" -> atomicWriteFile(path, updated); report ActionRemoved
// -> config.toml still exists on disk afterward, containing only a BOM + newline
```

Impact is bounded — no data loss (nothing user-authored is destroyed), no corruption (a
BOM-plus-blank-line file is valid, harmless TOML), and no crash — but the reported action is
factually wrong (`ActionRemoved` while the file persists), which is the same "silent, both
wrong" reporting pattern CR-01 itself was raised against, just at a smaller blast radius. This
is an incomplete fix, not a new defect independent of CR-01: it was introduced by CR-01's own
choice to preserve the BOM byte-for-byte, which is correct in isolation but wasn't threaded
through to the "is this file now effectively empty" check one layer up in `codex.go`.

**Fix:** Treat a BOM-only residual the same as a fully-empty residual for the delete-vs-rewrite
decision in `codex.go`'s `Uninstall` (and any other caller relying on `stripTOMLTable`'s `""`
sentinel for keep-clean removal, if one exists) — e.g.:

```go
switch {
case updated == existing:
    result.Files = append(result.Files, FileResult{Path: configPath, Action: ActionNotFound})
case updated == "" || strings.TrimRight(strings.TrimPrefix(updated, tomlBOM), "\r\n") == "":
    if err := os.Remove(configPath); err != nil {
        ...
    }
    // file removed entirely -- the BOM is discarded along with it, which is
    // acceptable since nothing else in the file depended on its presence
```

or, equivalently, have `stripTOMLTable` itself return `""` (dropping the BOM) whenever the only
non-empty residual is the BOM, and let a *fresh* BOM be re-synthesized only if some other future
write path needs one — since a file being deleted has no downstream reader left to care whether
its BOM survived. Add an integration-level test through `codexTarget{}.Uninstall` (not just the
`stripTOMLTable` unit level) asserting the file is actually absent from disk afterward for this
exact "BOM + only-table" fixture, since the current `TestStripTOMLTable_UTF8BOM_Fixture` only
exercises the string function and never catches this call-site gap.

### WR-01 — verified correct; no findings

`writeHookEntry`'s `ownedInsertAt` tracking is correct for every shape checked:

- **Interleaved (own at 0 and 2, foreign at 1):** traced by hand — `ownedInsertAt` locks to the
  first owned block's position among unowned blocks (0) on the first owned block seen and never
  moves; the foreign block lands at `unowned[:0]` (empty) + `normalizedOwn` + `unowned[0:]`
  (foreign), landing at the same index it started at whenever `normalizedOwn` has exactly one
  element (codegraph's only production shape, confirmed via the PreToolUse/Claude registration
  call sites). Multiple pre-existing owned blocks collapsing into one `normalizedOwn` on
  self-heal is expected behavior, not a regression.
- **Owned last (foreign at 0, own at 1):** `ownedInsertAt` becomes `len(unowned)` = 1 at the
  point the owned block is seen; result correctly reassembles as `[foreign, normalizedOwn...]`
  — unchanged relative order.
- **Multiple pre-existing owned blocks:** `ownedInsertAt` is set only once, on the *first*
  owned block encountered (guarded by `if ownedInsertAt == -1`), so later owned blocks never
  perturb the insertion point — correct.
- **Empty/absent array:** `hooks[event].([]any)` type-asserts to `nil` when absent; the loop
  runs zero times, `ownedInsertAt` stays `-1`, and the pre-existing first-install append-last
  path is taken, matching `TestWriteHookEntry_FirstInstallAppendsAfterForeignBlocks`.
- **Slice-bounds safety:** `unowned[:ownedInsertAt]`/`unowned[ownedInsertAt:]` can never go out
  of bounds — `ownedInsertAt` is assigned exactly `len(unowned)` at the moment it is set, and
  `unowned` only grows (never shrinks) afterward, so `ownedInsertAt <= len(unowned)` holds at
  the point of use unconditionally. No new panic path.
- **No drop/duplicate/reorder of unowned blocks:** `unowned`'s own relative order and full
  membership is preserved intact in both the `ownedInsertAt == -1` and `else` branches; only
  the insertion point of the (already fully-formed) `normalizedOwn` slice varies.
- **Determinism:** `writeJSONFile` serializes via `json.MarshalIndent` on a `map[string]any`,
  whose key ordering `encoding/json` already sorts independently of this change — untouched by
  the fix, still deterministic.
- **Claude settings.json / six-PreToolUse-block shape:** confirmed via the fixer's own
  verification that `TestPreToolUseRegistrationShape` and all 32 `TestOwnershipExactIdentity`
  leaves stayed green; the fix only changes insertion position within an event's array, never
  block identity, matcher, or handler shape.

### Mutation log families j1/j2 — claim shape verified (not re-run)

Both `git show`n diffs match the log's transcripts exactly:

- **(j1)** short-circuits `splitTOMLLines`' BOM branch with `if false && ...` — correctly turns
  off only the new code path, and the logged RED transcript's `got=`/`want=` values match what
  hand-tracing the unmutated `isTOMLHeaderLine`/`findTOMLTableRange` behavior on that exact
  BOM'd input predicts (header search fails because `isTOMLHeaderLine` never strips the BOM,
  precisely CR-01's original defect reproduced). The log's own `want="\ufeff\n"` line for the
  `TestStripTOMLTable_UTF8BOM_Fixture` case independently corroborates WR-03 above: the intended
  GREEN result for that fixture was never `""`.
- **(j2)** collapses the `if ownedInsertAt == -1 {...} else {...}` branch back to the pre-fix
  unconditional `append(unowned..., normalizedOwn...)` — the logged failure (foreign block
  observed at index 0 instead of 1) is exactly the predicted WR-01 regression shape, and the
  companion first-install control staying green under the same mutation is consistent with that
  code path being identical before and after the fix.

Both mutations' diffs are minimal, single-purpose, and target exactly the code the fix
introduced — no unrelated changes riding along that would inflate or deflate the claimed RED.

### Summary

- CR-01: fix verified correct at the `toml.go` layer for every case in scope. One new
  incomplete-fix Warning (WR-03) found one layer up, in `codex.go`'s Uninstall keep-clean
  decision, which CR-01 did not touch and which its own fixture pins the triggering shape for.
- WR-01: fix verified correct with no findings.
- Mutation log families j1/j2: claim shape verified against the actual diffs and transcripts;
  consistent and trustworthy.

**Findings this pass:** 0 Critical, 1 Warning (WR-03), 0 Info.

---

_Re-reviewed: 2026-09-19_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep (narrow scope: CR-01/WR-01 fixes and their callers)_

## Fix iteration 2

**Fixed at:** 2026-09-19T00:00:00Z
**Scope:** WR-03 only (re-review pass 2's single new finding).

**Summary:**
- Findings in scope: 1 (WR-03)
- Fixed: 1
- Skipped: 0

**Verification environment:** main checkout at
`/Volumes/Code/github.com/seanb4t/codegraph-go`, branch `gsd/v0.14.0-milestone` — no worktree
used (`workflow.use_worktrees: false` in `.planning/config.json`).

### WR-03: BOM-only residual defeats keep-clean uninstall, and the reported action is wrong

**Files modified:** `internal/agents/toml.go`, `internal/agents/toml_test.go`,
`internal/agents/codex_test.go`, `internal/agents/testdata/toml/codex-bom.uninstalled.toml`
**Commits:**
- `e639799c` — `test(07-12): RED — BOM-only residual defeats keep-clean uninstall (WR-03)`
- `55c25b81` — `fix(07-12): drop a BOM-only residual entirely in stripTOMLTable (WR-03)`

**Chosen fix (cause-level, per the review's "prefer one definition of effectively empty"
guidance):** `stripTOMLTable` (`internal/agents/toml.go:63-90`) itself now treats a
before-side residual of `""` OR exactly the leading BOM (`before == tomlBOM`), combined with an
empty after-side, as "effectively empty" and returns `""` — dropping the BOM along with the
rest. This was chosen over patching `codex.go`'s `Uninstall` switch because it keeps
`stripTOMLTable`'s `""` return the single "file is now empty" sentinel every caller keys its
keep-clean removal off of; `codex.go`'s `updated == ""` check needed no change at all. The only
other caller of `stripTOMLTable` is `codex.go:305` itself (confirmed via
`rg -n "stripTOMLTable\("` — no other target or call site exists), so no other caller needed
threading through. CR-01 is not weakened: a BOM present alongside real content on either side
of the stripped table still round-trips byte-for-byte
(`TestStripTOMLTable_UTF8BOM_PreservedWhenOtherContentSurvives`).

**RED proof** (`GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestStripTOMLTable_UTF8BOM_TableOnlyResidualDropsEntirely$|TestStripTOMLTable_UTF8BOM_Fixture$|TestCodex_Uninstall_BOMOnlyEmptiedConfigIsRemoved$' -v`):
```
    codex_test.go:227: config.toml should have been removed entirely after uninstall (BOM-only residual), got: "\ufeff\n"
--- FAIL: TestCodex_Uninstall_BOMOnlyEmptiedConfigIsRemoved (0.00s)
    toml_test.go:230: stripTOMLTable must drop a BOM-only residual entirely when codegraph's table was the file's only content (WR-03), got: "\ufeff\n"
--- FAIL: TestStripTOMLTable_UTF8BOM_TableOnlyResidualDropsEntirely (0.00s)
    toml_test.go:296: stripTOMLTable BOM-fixture mismatch:
        got="\ufeff\n"
        want=""
--- FAIL: TestStripTOMLTable_UTF8BOM_Fixture (0.00s)
FAIL
```
(`TestStripTOMLTable_UTF8BOM_PreservedWhenOtherContentSurvives` — the new positive control —
passed against unfixed code, as expected, since it exercises the unaffected `default` branch.)

**Test changes and why (explicit, per the fix instructions):**
- **`internal/agents/testdata/toml/codex-bom.uninstalled.toml`** (a frozen fixture from CR-01)
  changed from 4 bytes (`ef bb bf 0a`, i.e. `"\ufeff\n"`) to 0 bytes (`""`). This is a genuine
  contract change, not a silent edit: WR-03's whole premise is that the old fixture bytes
  encoded the bug, corroborated independently by 07-MUTATION-LOG.md Family (j1)'s own
  transcript (`want="\ufeff\n"`) predating this fix.
- **`TestStripTOMLTable_UTF8BOM_RemovesEntirelyPreservingBOM`** (CR-01's original unit test for
  this exact scenario, not called out by name in the review but discovered during
  investigation to assert the opposite of the corrected contract) renamed to
  **`TestStripTOMLTable_UTF8BOM_TableOnlyResidualDropsEntirely`** and its assertion flipped
  from "BOM survives" to "result is exactly `\"\"`". The rename means 07-MUTATION-LOG.md
  Family (j1)'s `-run` regex (which names the old test) will no longer match it going
  forward — that family's logged transcript remains an accurate historical record of what ran
  at the time, unaffected by a later rename.
- **`TestStripTOMLTable_UTF8BOM_PreservedWhenOtherContentSurvives`** (new) is the companion
  positive control: BOM + codegraph's table + a following unrelated table still preserves the
  BOM byte-for-byte after stripping, proving the fix is scoped to the BOM-only-residual case
  and does not weaken CR-01's general guarantee.
- **`TestCodex_Uninstall_BOMOnlyEmptiedConfigIsRemoved`** (new, `internal/agents/codex_test.go`)
  is the integration-level test the review asked for: through `codexTarget{}.Uninstall` itself
  (install, prepend a BOM, uninstall), asserting the file is actually absent from disk
  afterward and the reported action is `ActionRemoved` — the real-binary scenario
  07-REVIEW-FIX.md's re-review pinned, run against an isolated `t.TempDir()` fake `HOME`
  (`fakeHome`), never the real developer machine.

Verified by:
- The 3 new/changed tests above, now green, plus the unaffected fixture pair
  (`TestSpliceTOMLTable_UTF8BOM_Fixture`) and the `TestFindTOMLTableRange_UTF8BOM` /
  `TestSpliceTOMLTable_UTF8BOM_UpdatesInPlaceNeverDuplicates` /
  `TestFindTOMLTableRange_MidFileBOMNotStrippedAsWhitespace` guards, all still green.
- `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ ./internal/cli/... ./internal/mcp/ -count=1`
  — all packages `ok`.
- Positive-control mutation: 07-MUTATION-LOG.md Family (j3) — narrowing
  `beforeIsEmptyOrBOMOnly` back to "before must be truly empty" reproduces the exact RED
  transcript above (all three guards fail, the BOM-preserved positive control stays green);
  reverted byte-clean (`git diff --quiet` clean before, dirty before revert, clean after),
  re-confirmed green, full-package re-check `ok` after the revert.

## Skipped Issues (iteration 2)

None — WR-03 was fixed.

## Mutation log (iteration 2)

07-MUTATION-LOG.md gained one new family for this fix:
- Family (j3) — WR-03's BOM-only-residual drop in `stripTOMLTable`.

---

_Fixed: 2026-09-19T00:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
