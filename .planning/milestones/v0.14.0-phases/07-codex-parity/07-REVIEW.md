---
phase: 07-codex-parity
reviewed: 2026-09-19T00:00:00Z
depth: deep
files_reviewed: 33
files_reviewed_list:
  - internal/agents/toml.go
  - internal/agents/codex.go
  - internal/agents/codex_pretooluse.go
  - internal/agents/capabilities.go
  - internal/agents/shared.go
  - internal/agents/opencode.go
  - internal/agents/instructions.go
  - internal/cli/install.go
  - internal/cli/uninstall.go
  - internal/cli/hook_pretooluse.go
  - internal/cli/tui/agentpicker.go
  - internal/cli/tui/daemonpicker.go
  - internal/mcp/server.go
  - codexassets.go
  - .codex/hooks/codegraph-pretooluse-local.sh
  - .codex/hooks/codegraph-pretooluse-global.sh
  - .codex/hooks/hooks.json
  - internal/agents/toml_test.go
  - internal/agents/codex_test.go
  - internal/agents/codex_pretooluse_test.go
  - internal/agents/capabilities_test.go
  - internal/agents/capability_doc_test.go
  - internal/agents/ownership_test.go
  - internal/agents/registry_test.go
  - internal/agents/shared_test.go
  - internal/cli/install_test.go
  - internal/cli/printconfigstyle_test.go
  - internal/cli/hook_pretooluse_codex_test.go
  - internal/cli/tui/agentpicker_test.go
  - internal/cli/tui/daemonpicker_test.go
  - internal/mcp/instructions_contract_test.go
  - test/tmux/install_cancel_test.go
  - README.md
findings:
  critical: 1
  warning: 2
  info: 1
  total: 4
status: fixed
fix_status:
  fixed: 3
  deferred: 1
  fixed_at: 2026-09-19T00:00:00Z
  fixed_findings: [CR-01, WR-01, WR-03]
  deferred_findings: [IN-01]
---

# Phase 7: Code Review Report

**Reviewed:** 2026-09-19
**Depth:** deep
**Files Reviewed:** 33 (of 40 in scope; docs/AGENT-CAPABILITIES.md, docs/CLI-REFERENCE.md, the TOML/plain-golden fixtures were cross-checked via the plans' SUMMARY/mutation-log evidence rather than re-read byte-for-byte)
**Status:** issues_found

## Summary

This review focused on the phase's own stated highest-risk surfaces: the rewritten TOML
scanner (`toml.go`), the Codex PreToolUse hook adapter and guard scripts, and the
ownership/shared-instructions machinery. The `07-MUTATION-LOG.md` families are real,
methodologically sound positive controls, and cross-checking them against the current source
confirms every one of the 07-01..07-11 guards still holds at HEAD. `07-CONTEXT.md`'s locked
decisions (D-00..D-30) were treated as ground truth throughout; no finding below contradicts
one, and the six items in the prompt's "known and accepted" list were confirmed present but not
re-reported.

Adversarial testing of the TOML scanner beyond what the phase's own fixtures cover surfaced one
**reproducible, unaddressed defect**: a UTF-8 BOM at the start of a Codex `config.toml` — a
realistic condition for a Windows-authored local per-project file, which per the phase's own
live evidence (07-LIVE-SESSIONS.md scaffold) commonly contains *only* the `[mcp_servers.*]`
tables with no preceding `model =`/`approval_policy =` prelude — defeats `isTOMLHeaderLine`'s
leading-whitespace trim, so `findTOMLTableRange` never recognizes codegraph's own existing table.
This was verified by direct execution (not just reasoning): `install` produces a genuine
duplicate `[mcp_servers.codegraph]` header (the corrupted "refused, not duplicated" invariant
this whole phase's Family (a) exists to guarantee), and `uninstall` becomes permanently unable to
remove the entry (`stripTOMLTable` returns the input byte-for-byte unchanged). See CR-01.

Two further findings are bounded but real: `writeHookEntry`'s owned-block always-append-last
behavior repositions a foreign hook block on any *update* (not just removal) to codegraph's own
Codex PreToolUse registration, which the phase's own live evidence (07-LIVE-SESSIONS.md T3) shows
causes Codex's position-keyed hook trust to spuriously re-flag an untouched foreign hook for
review — a case D-23's live check exercised only for removal, not update (WR-01). And Codex's
argv-tolerant `tool_input.command` decoding takes the *last* array element, which is only correct
if Codex ever wraps a shell invocation as `[shell, flag, full-command-string]`; if it ever sends
a literal `[prog, arg1, arg2, ...]` split, the nudge would silently misclassify and never fire —
noted as informational since this path has never been exercised live and the failure mode is a
missed nudge, never a block (IN-01).

Every other area in the phase's stated risk profile (the hook adapter's never-block contract,
the shell guards' quoting/whitespace safety and cheap-check-first structure, exact-command
ownership identity, `instructionsRequestedElsewhere`'s registry-derived correctness across every
uninstall order, and the FIX-03 delegate fix) held up under adversarial tracing and is not
re-litigated here.

## Critical Issues

### CR-01: A UTF-8 BOM at the start of a Codex config.toml defeats header recognition, producing a duplicate `[mcp_servers.codegraph]` table on install and a permanently un-removable entry on uninstall

**Fix status: fixed.** `splitTOMLLines` (`internal/agents/toml.go`) now emits content's
absolute leading UTF-8 BOM as its own synthetic, non-header, non-splicable pseudo-line
(offset 0, 3 bytes) instead of folding it into the header line that follows — the single
point of entry `findTOMLTableRange`, `tomlTableConflict` and `tomlBoolSetting` all funnel
through, so all three inherit the fix. The BOM itself is never rewritten (round-trips
byte-for-byte through splice/strip), and a BOM appearing on any later line is never treated
as leading whitespace to strip (pinned by `TestFindTOMLTableRange_MidFileBOMNotStrippedAsWhitespace`).
RED/GREEN: `test(07-12)` commit `12d4c0cb`, `fix(07-12)` commit `01e8306d`. Positive control:
07-MUTATION-LOG.md Family (j1). New tests: `TestFindTOMLTableRange_UTF8BOM`,
`TestSpliceTOMLTable_UTF8BOM_UpdatesInPlaceNeverDuplicates`,
`TestStripTOMLTable_UTF8BOM_RemovesEntirelyPreservingBOM`,
`TestFindTOMLTableRange_MidFileBOMNotStrippedAsWhitespace`,
`TestSpliceTOMLTable_UTF8BOM_Fixture`, `TestStripTOMLTable_UTF8BOM_Fixture` (fixtures:
`internal/agents/testdata/toml/codex-bom{,.installed,.uninstalled}.toml`).

**File:** `internal/agents/toml.go:194-196` (`isTOMLHeaderLine`), consumed by `findTOMLTableRange` (`:107-126`), `spliceTOMLTable` (`:28-55`) and `stripTOMLTable` (`:63-90`)

**Issue:** `isTOMLHeaderLine` recognizes a header only via `strings.HasPrefix(strings.TrimLeft(line, " \t"), "[")`. `TrimLeft(line, " \t")` strips only ASCII space and tab — it does not strip a leading UTF-8 BOM (`U+FEFF`, bytes `EF BB BF`). `splitTOMLLines` does no BOM stripping either. Consequently, if content's very first three bytes are a BOM and the file's first line is itself a table header — the common shape for a project-local `.codex/config.toml`, which per this phase's own live scaffold (07-LIVE-SESSIONS.md) is written as nothing but the `[mcp_servers.codegraph]` table with no preceding key — that header line is never recognized as a header at all.

Verified by direct execution against the current source (not just reasoning):

```go
content := "\xef\xbb\xbf[mcp_servers.codegraph]\ncommand = \"/old/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n"
_, _, found := findTOMLTableRange(content, "mcp_servers.codegraph")
// found = false

updated := spliceTOMLTable(content, "mcp_servers.codegraph", codexTableBody("/new/codegraph"))
// updated = "﻿[mcp_servers.codegraph]\ncommand = \"/old/codegraph\"\nargs = [...]\n" +
//           "\n[mcp_servers.codegraph]\ncommand = \"/new/codegraph\"\nargs = [...]\n"
//   -> a genuine SECOND [mcp_servers.codegraph] header appended, TWO definitions of the
//      same table now exist in the file (invalid/duplicate-key TOML) -- exactly the
//      "refused, not duplicated" invariant Family (a) was built to guarantee, silently
//      violated for this one input shape.

stripped := stripTOMLTable(content, "mcp_servers.codegraph")
// stripped == content byte-for-byte -- codegraph's own table is never found, so
// `codegraph uninstall --target codex` reports the removal as having happened
// (falls through the "not found" branch of Uninstall, since updated == existing)
// while the file is left completely untouched, forever.
```

`Install` reports this as `configured`/`created` with no error (the write genuinely succeeds; it just corrupts the file's structure), and `Uninstall` reports `not-found`/`not-configured` with no error either — both silent, both wrong. A downstream TOML parser (including Codex's own config loader) that rejects duplicate table definitions would then fail to load the *entire* config.toml, not just codegraph's entry, taking every other MCP server in that file down with it — the same blast radius the phase's headline released-binary bug had, for a different input class it did not anticipate.

**Fix:** Strip a leading BOM once, at the single point of entry all of `findTOMLTableRange`, `tomlTableConflict`, and `tomlBoolSetting` funnel through (`splitTOMLLines`), and re-emit it on write so round-tripping stays byte-faithful for files that already carry one:

```go
const tomlBOM = "\xef\xbb\xbf"

func splitTOMLLines(content string) []tomlLine {
	content = strings.TrimPrefix(content, tomlBOM) // caller must offset by len(tomlBOM) if reproducing bytes
	...
}
```
More precisely: since `findTOMLTableRange`/`spliceTOMLTable`/`stripTOMLTable` all operate on absolute byte offsets into the original `content`, the safest fix is to detect and skip the BOM as a special-cased zero-th "line" (offset 0, length 3, never itself eligible to be a header or to be spliced over) inside `splitTOMLLines`, rather than trimming it — preserving every offset the rest of the file already relies on. Add a fixture mirroring this exact scenario (BOM + a file whose first content line is `[mcp_servers.codegraph]`) to `toml_test.go`, since none of the existing CRLF/multi-line-string/conflict fixtures exercise a BOM.

## Warnings

### WR-01: `writeHookEntry` repositions a foreign Codex hook block on any *update* to codegraph's own PreToolUse registration, not just on removal — spuriously invalidating Codex's position-keyed hook trust for an untouched hook

**Fix status: fixed.** `writeHookEntry` (`internal/agents/shared.go`) now tracks the index the
first owned block originally occupied among the unowned blocks during its single partition
pass, and splices the new normalized owned blocks back in at that same index instead of
always appending them last. A genuine first install (no owned block existed before the call)
is unaffected and still appends codegraph's group after every existing block, preserving
D-23. RED/GREEN: `test(07-12)` commit `c1b17930`, `fix(07-12)` commit `2ac15d05`. Positive
control: 07-MUTATION-LOG.md Family (j2). New tests:
`TestWriteHookEntry_UpdatePreservesForeignBlockPosition`,
`TestWriteHookEntry_FirstInstallAppendsAfterForeignBlocks` (`internal/agents/shared_test.go`).
Confirmed still green: `TestPreToolUseRegistrationShape` and all 32
`TestOwnershipExactIdentity` leaves.

**File:** `internal/agents/shared.go:261-280` (`writeHookEntry`), reached from `internal/agents/codex_pretooluse.go:193` (`installCodexPreToolNudge`)

**Issue:** `writeHookEntry` always rebuilds the hooks array as `unowned` (original relative order) followed by `normalizedOwn` (codegraph's own blocks), regardless of where codegraph's block sat before the write:

```go
newEvents := append(append([]any{}, unowned...), normalizedOwn...)
```

D-23's live check (07-LIVE-SESSIONS.md T3) proved this is safe for *removal*, because codegraph's group is always appended last on install, so removing it never shifts anything after it. But the same reasoning does not extend to an *update in place* — e.g. `codegraph install --target codex --location local --pretool-nudge` re-run after the binary's `ExecPath` changes, or any future guard-content bump. If a user (or another tool) appended a foreign hook group *after* codegraph's own — which, given codegraph's own convention of always appending last, is exactly where a subsequently-added foreign hook would land — an update to codegraph's own block will silently leapfrog it back to the end, displacing the foreign block one index earlier.

Verified by direct execution:

```go
// initial hooks.json: PreToolUse[0]=own(OWN_V1), PreToolUse[1]=foreign(FOREIGN)
writeHookEntry(path, "PreToolUse", []any{newOwnBlock(OWN_V2)}, []string{"OWN_V1", "OWN_V2"})
// after: PreToolUse[0]=foreign(FOREIGN), PreToolUse[1]=own(OWN_V2)
```

Codex's hook trust is position-keyed (`hooks.state."<path>:pre_tool_use:<groupIdx>:<handlerIdx>"`, confirmed live in 07-LIVE-SESSIONS.md B5/T3). The foreign hook's trust-state key silently changes from `:1:0` to `:0:0` even though its own bytes never changed, so Codex re-flags it as `Modified since last trusted — review required` on the user's next session — a spurious trust prompt for a hook the user did nothing to.

**Fix:** When an owned block already exists at some index and is merely being *updated* (not removed), preserve its original position among the array rather than always moving it to the end. For example, track the index of the first owned block found during the partition pass and splice `normalizedOwn` back in at that index instead of unconditionally appending after `unowned`. D-23's "appended last" guarantee should then be scoped explicitly to first-install only, and the live check in a follow-up plan should additionally exercise "foreign hook added after ours, then codegraph's own content changes" — the case D-23's T3 evidence did not cover.

### WR-02 — folded into IN-01 below (see Info)

## Info

### IN-01: `codexShellCommand`'s argv-form decoding takes the last array element, which is only correct for a `[shell, flag, full-command]` wrapper shape — an untested assumption

**Fix status: deferred (in scope by explicit fix-scope decision, not an oversight).** Per the
finding's own Fix guidance, no code change is warranted before Codex's argv shape is ever
observed live — the "any doubt, silent" contract already makes the failure mode safe (a
missed nudge, never a block or crash). Left for a natural follow-up alongside CODEX-05/06's
next live session, at which point a positive-control fixture pinning the observed shape
should be added and the doc comment corrected from "argv (the last element)" to the
confirmed shape.

**File:** `internal/cli/hook_pretooluse.go:72-95` (`codexShellCommand`)

**Issue:** D-21 documents Codex's `tool_input.command` as "a string or an argv array (the last element when an argv)". Every live session recorded in 07-LIVE-SESSIONS.md observed only the string form (`"command": "rg -n Alpha ."`); the argv branch has never been exercised against a real Codex payload. Taking the *last* element is correct only if Codex's argv form is a `["/bin/sh", "-c", "actual command string"]`-style wrapper (a common convention for "run via shell" tool schemas) — if Codex instead ever sends a literal split argv like `["rg", "-n", "Alpha", "."]`, the last element (`"."`) would be classified instead of the actual command, and `hookQualifies` would almost certainly reject it, silently suppressing a nudge that should have fired.

**Fix:** No code change is required before this path is ever observed live — the contract's "any doubt, silent" posture already makes the failure mode safe (a missed nudge, never a block or crash). When Codex's argv shape is confirmed live (a natural follow-up to CODEX-05/06), add a positive-control fixture pinning whichever shape is actually observed, and update the doc comment to state the confirmed shape rather than "argv (the last element)" as a standing assumption.

---

_Reviewed: 2026-09-19_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: deep_
