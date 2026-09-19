# 07-MUTATION-LOG — Codex Parity

**Phase:** 07-codex-parity
**Date:** 2026-09-19
**Scope:** every guard this phase adds is positive-controlled here (D-00, rule
`84d1gfpywd`). Families, one per owning plan:

- Family (a) — plan 07-01: the Codex TOML table splice (`findTOMLTableRange`,
  `tomlTableConflict`, `tomlLineEnding`).
- Family (b) — plan 07-02: `install --yes`/`--target` resolution order.
- Family (c) — plan 07-03: the agent/daemon picker footer overflow.
- Family (d) — plan 07-05: the Codex global/local scope flip.
- Family (e) — plan 07-06: the shared repo-root `AGENTS.md`.
- Family (f) — plan 07-07: the Codex PreToolUse nudge runtime (hooks.json
  fragment, guard script, adapter).
- Family (g) — plan 07-08: the Codex nudge lifecycle and CLI opt-in.
- Family (h) — plan 07-10: the published capability-table doc drift guard.
- Family (i) — plan 07-11: the MCP `instructions` wire-string byte budget.

## Pre-mutation cleanliness gate — the convention this log follows

Before every tracked-file mutation, AND before every revert, `git diff --quiet -- <file>`
is asserted. Before the plant it must exit clean (0), proving no pre-existing tracked edit
is overwritten by the mutation; before the revert it must exit dirty (1), proving the
planted diff is the only change the revert discards, so the revert is never a blind checkout
of someone else's in-flight work. After the revert it must exit clean again. Every family
entry below records the gate's result at each point.

Every Go command runs with `GOTOOLCHAIN=go1.26.6`. Temp paths in the transcripts are
abbreviated to `/var/folders/.../T/`.

---

## Family (a1) — D-07: the column-0-only end scan turns TestSpliceTOMLTable_MaintainerIndentedLayout RED

**Test/guard:** `TestSpliceTOMLTable_MaintainerIndentedLayout` (`internal/agents/toml_test.go`)
— splices `codexBody()` into `codex-indented-layout.toml` (the maintainer-shaped fixture) and
asserts the result equals `codex-indented-layout.installed.toml` byte for byte.

**What are we testing, and why?** Whether the guard catches the exact released-binary
defect (07-CONTEXT.md's blocking finding): `isTOMLHeaderLine` recognizing a table header only
at column 0, the shape that let an indented `[mcp_servers.codegraph]` swallow every following
sibling table on `install`/`uninstall --target codex`.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/toml.go` — exit 0 (clean).

**Mutation applied:** `perl -0pi -e 's/strings\.TrimLeft\(line, " \\t"\)/line/' internal/agents/toml.go`
(the pinned Family a1 site — `isTOMLHeaderLine`'s only use of `strings.TrimLeft(line, " \t")`):

```diff
--- a/internal/agents/toml.go
+++ b/internal/agents/toml.go
@@ -192,7 +192,7 @@ func splitTOMLLines(content string) []tomlLine {
 // "[x]" or "[[x]]" table header at any indentation (D-07). This function
 // is the file's only caller of the leading-whitespace trim it uses here.
 func isTOMLHeaderLine(line string) bool {
-	return strings.HasPrefix(strings.TrimLeft(line, " \t"), "[")
+	return strings.HasPrefix(line, "[")
 }
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestSpliceTOMLTable_MaintainerIndentedLayout$' -v`, `=== RUN` line dropped, exit code
appended):

```
    toml_test.go:133: spliceTOMLTable maintainer-layout mismatch:
        got="model = \"gpt-test\"\napproval_policy = \"on-request\"\n\n[features]\nhooks = true\n\n  [mcp_servers.alpha]\n  command = \"/usr/bin/alpha\"\n  args = [\"a\"]\n\n  [mcp_servers.codegraph]\n  command = \"/old/codegraph\"\n  args = [\"serve\", \"--mcp\"]\n\n  [mcp_servers.context7]\n  command = \"/usr/bin/context7\"\n  startup_timeout_ms = 5000\n  args = [\n    \"--flag-one\",\n    \"--flag-two\",\n    \"--flag-three\",\n    \"--flag-four\",\n  ]\n\n  # engram: local memory\n  [mcp_servers.engram]\n  command = \"/usr/bin/engram\"\n  startup_timeout_ms = 3000\n  env = { MEMORY_DIR = \"/tmp/m\" }\n\n# memories\n[memories]\nenabled = true\n\n[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n"
        want="model = \"gpt-test\"\napproval_policy = \"on-request\"\n\n[features]\nhooks = true\n\n  [mcp_servers.alpha]\n  command = \"/usr/bin/alpha\"\n  args = [\"a\"]\n\n[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n\n  [mcp_servers.context7]\n  command = \"/usr/bin/context7\"\n  startup_timeout_ms = 5000\n  args = [\n    \"--flag-one\",\n    \"--flag-two\",\n    \"--flag-three\",\n    \"--flag-four\",\n  ]\n\n  # engram: local memory\n  [mcp_servers.engram]\n  command = \"/usr/bin/engram\"\n  startup_timeout_ms = 3000\n  env = { MEMORY_DIR = \"/tmp/m\" }\n\n# memories\n[memories]\nenabled = true\n"
--- FAIL: TestSpliceTOMLTable_MaintainerIndentedLayout (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.127s
FAIL
exit=1
```

**Note on the observed shape (accuracy over prediction):** with this mutation, `isTOMLHeaderLine`
no longer recognizes ANY indented header — including codegraph's own `  [mcp_servers.codegraph]`
header line at the start of the scan. `findTOMLTableRange` therefore reports `found = false`
for the indented table entirely, so `spliceTOMLTable` takes its "not found" (append) path: the
original indented block is left completely untouched and a second, column-0
`[mcp_servers.codegraph]` block is appended after `[memories]` (visible in `got` above). This is
a stricter regression than the released bug (which only broke the *end* scan, not header
recognition itself), but it exercises the same pinned mutation site the plan specifies and
proves the guard is not vacuous — the fixture still goes RED, and the failure is not the
prediction ("context7/engram missing") but a genuine, verbatim-recorded appended duplicate.

**Pre-revert gate:** `git diff --quiet -- internal/agents/toml.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/agents/toml.go`, then
`git diff --quiet -- internal/agents/toml.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestSpliceTOMLTable_MaintainerIndentedLayout$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/agents	0.099s`.

---

## Family (a2) — D-07: a disabled conflict check turns TestTOMLTableConflict and TestSpliceTOMLTable_ConflictLeavesContentUnchanged RED

**Test/guard:** `TestTOMLTableConflict` and `TestSpliceTOMLTable_ConflictLeavesContentUnchanged`
(`internal/agents/toml_test.go`) — twelve conflict-shape subtests (inline table, dotted key at
root/under parent, quoted/spaced header, array-of-tables, duplicate header, detached own
subtable, plus four non-conflicting controls) and the matching "splice leaves the input
byte-identical" check for the eight conflicting shapes.

**What are we testing, and why?** Whether the guard catches `tomlTableConflict` silently
answering "no conflict" for every one of these forms. D-07 requires an inline `codegraph = {…}`,
a dotted `mcp_servers.codegraph.*` key, a quoted or spaced header variant, an
`[[mcp_servers.codegraph]]` array-of-tables header, a duplicate header, or a detached own
subtable to be **refused**, never silently duplicated.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/toml.go` — exit 0 (clean).

**Mutation applied:** an early `return nil` inserted immediately after `tomlTableConflict`'s
signature line (the pinned Family a2 site):

```diff
--- a/internal/agents/toml.go
+++ b/internal/agents/toml.go
@@ -300,6 +300,7 @@ var errTOMLTableConflict = errors.New("codegraph's TOML table conflicts with an
 // 1-based line; spliceTOMLTable/stripTOMLTable both leave content
 // completely unchanged in that case, never producing a duplicate key.
 func tomlTableConflict(content, tableName string) error {
+	return nil
 	subtablePrefix := tableName + "."
 	exactHeader := "[" + tableName + "]"
 	lines := splitTOMLLines(content)
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestTOMLTableConflict$|TestSpliceTOMLTable_ConflictLeavesContentUnchanged$' -v`,
`=== RUN`/`--- PASS` lines for the two tests' own headers kept, per-subtest `--- PASS` lines for
the passing controls kept, exit code appended):

```
    toml_test.go:427: tomlTableConflict("[mcp_servers]\ncodegraph = { command = \"/x\" }\n") = nil, want a conflict error
    toml_test.go:427: tomlTableConflict("mcp_servers.codegraph.command = \"/x\"\n") = nil, want a conflict error
    toml_test.go:427: tomlTableConflict("[mcp_servers]\ncodegraph.command = \"/x\"\n") = nil, want a conflict error
    toml_test.go:427: tomlTableConflict("[\"mcp_servers\".\"codegraph\"]\ncommand = \"/x\"\n") = nil, want a conflict error
    toml_test.go:427: tomlTableConflict("[mcp_servers . codegraph]\ncommand = \"/x\"\n") = nil, want a conflict error
    toml_test.go:427: tomlTableConflict("[[mcp_servers.codegraph]]\ncommand = \"/x\"\n") = nil, want a conflict error
    toml_test.go:427: tomlTableConflict("[mcp_servers.codegraph]\ncommand = \"/a\"\n\n[mcp_servers.codegraph]\ncommand = \"/b\"\n") = nil, want a conflict error
    toml_test.go:427: tomlTableConflict("[mcp_servers.codegraph.env]\nFOO = \"bar\"\n\n[some_other_table]\nkey = \"x\"\n\n[mcp_servers.codegraph]\ncommand = \"/x\"\n") = nil, want a conflict error
--- FAIL: TestTOMLTableConflict (0.00s)
    --- FAIL: TestTOMLTableConflict/inline_table_under_parent (0.00s)
    --- FAIL: TestTOMLTableConflict/dotted_key_at_root (0.00s)
    --- FAIL: TestTOMLTableConflict/dotted_key_under_parent (0.00s)
    --- FAIL: TestTOMLTableConflict/quoted_header (0.00s)
    --- FAIL: TestTOMLTableConflict/spaced_header (0.00s)
    --- FAIL: TestTOMLTableConflict/array_of_tables (0.00s)
    --- FAIL: TestTOMLTableConflict/duplicate_header (0.00s)
    --- FAIL: TestTOMLTableConflict/detached_own_subtable (0.00s)
    --- PASS: TestTOMLTableConflict/clean_own_table (0.00s)
    --- PASS: TestTOMLTableConflict/prefix_lookalike_table (0.00s)
    --- PASS: TestTOMLTableConflict/unrelated_key_named_codegraph (0.00s)
    --- PASS: TestTOMLTableConflict/quoted_project_trust_table (0.00s)
    toml_test.go:447: spliceTOMLTable on a conflicting input must leave content unchanged:
        got="[mcp_servers]\ncodegraph = { command = \"/x\" }\n\n[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n"
        want="[mcp_servers]\ncodegraph = { command = \"/x\" }\n"
    toml_test.go:447: spliceTOMLTable on a conflicting input must leave content unchanged:
        got="mcp_servers.codegraph.command = \"/x\"\n\n[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n"
        want="mcp_servers.codegraph.command = \"/x\"\n"
    toml_test.go:447: spliceTOMLTable on a conflicting input must leave content unchanged:
        got="[mcp_servers]\ncodegraph.command = \"/x\"\n\n[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n"
        want="[mcp_servers]\ncodegraph.command = \"/x\"\n"
    toml_test.go:447: spliceTOMLTable on a conflicting input must leave content unchanged:
        got="[\"mcp_servers\".\"codegraph\"]\ncommand = \"/x\"\n\n[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n"
        want="[\"mcp_servers\".\"codegraph\"]\ncommand = \"/x\"\n"
    toml_test.go:447: spliceTOMLTable on a conflicting input must leave content unchanged:
        got="[mcp_servers . codegraph]\ncommand = \"/x\"\n\n[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n"
        want="[mcp_servers . codegraph]\ncommand = \"/x\"\n"
    toml_test.go:447: spliceTOMLTable on a conflicting input must leave content unchanged:
        got="[[mcp_servers.codegraph]]\ncommand = \"/x\"\n\n[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n"
        want="[[mcp_servers.codegraph]]\ncommand = \"/x\"\n"
    toml_test.go:447: spliceTOMLTable on a conflicting input must leave content unchanged:
        got="[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n"
        want="[mcp_servers.codegraph]\ncommand = \"/a\"\n\n[mcp_servers.codegraph]\ncommand = \"/b\"\n"
    toml_test.go:447: spliceTOMLTable on a conflicting input must leave content unchanged:
        got="[mcp_servers.codegraph.env]\nFOO = \"bar\"\n\n[some_other_table]\nkey = \"x\"\n\n[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n"
        want="[mcp_servers.codegraph.env]\nFOO = \"bar\"\n\n[some_other_table]\nkey = \"x\"\n\n[mcp_servers.codegraph]\ncommand = \"/x\"\n"
--- FAIL: TestSpliceTOMLTable_ConflictLeavesContentUnchanged (0.00s)
    --- FAIL: TestSpliceTOMLTable_ConflictLeavesContentUnchanged/inline_table_under_parent (0.00s)
    --- FAIL: TestSpliceTOMLTable_ConflictLeavesContentUnchanged/dotted_key_at_root (0.00s)
    --- FAIL: TestSpliceTOMLTable_ConflictLeavesContentUnchanged/dotted_key_under_parent (0.00s)
    --- FAIL: TestSpliceTOMLTable_ConflictLeavesContentUnchanged/quoted_header (0.00s)
    --- FAIL: TestSpliceTOMLTable_ConflictLeavesContentUnchanged/spaced_header (0.00s)
    --- FAIL: TestSpliceTOMLTable_ConflictLeavesContentUnchanged/array_of_tables (0.00s)
    --- FAIL: TestSpliceTOMLTable_ConflictLeavesContentUnchanged/duplicate_header (0.00s)
    --- FAIL: TestSpliceTOMLTable_ConflictLeavesContentUnchanged/detached_own_subtable (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.104s
FAIL
exit=1
```

Every one of the eight conflicting subtests goes RED in both tests; the four non-conflicting
controls (`clean_own_table`, `prefix_lookalike_table`, `unrelated_key_named_codegraph`,
`quoted_project_trust_table`) correctly stay green, since a stub returning `nil` for every
input still matches "no conflict" for inputs that legitimately have none.

**Pre-revert gate:** `git diff --quiet -- internal/agents/toml.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/agents/toml.go`, then
`git diff --quiet -- internal/agents/toml.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestTOMLTableConflict$|TestSpliceTOMLTable_ConflictLeavesContentUnchanged$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/agents	0.063s`.

---

## Family (a3) — D-07: an LF-only tomlLineEnding turns TestSpliceTOMLTable_CRLFPreserved RED

**Test/guard:** `TestSpliceTOMLTable_CRLFPreserved` (`internal/agents/toml_test.go`) — subtests
`append` (splicing into a CRLF file with no existing codegraph table) and `replace` (splicing
into a CRLF file with an existing codegraph table), each asserting the result contains no LF
byte not immediately preceded by a CR.

**What are we testing, and why?** Whether the guard catches `spliceTOMLTable` mixing bare `\n`
into an otherwise CRLF file — D-07's "CRLF input keeps CRLF" requirement.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/toml.go` — exit 0 (clean).

**Mutation applied:** an early `return "\n"` inserted immediately after `tomlLineEnding`'s
signature line (the pinned Family a3 site):

```diff
--- a/internal/agents/toml.go
+++ b/internal/agents/toml.go
@@ -278,6 +278,7 @@ func tomlHeaderPath(line string) string {
 // line ending, else "\n" (D-07: a CRLF file must round-trip byte for
 // byte through splice/strip).
 func tomlLineEnding(content string) string {
+	return "\n"
 	if strings.Contains(content, "\r\n") {
 		return "\r\n"
 	}
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestSpliceTOMLTable_CRLFPreserved$' -v`, `=== RUN` line dropped, exit code appended):

```
    toml_test.go:315: found LF not preceded by CR at byte 54:
        "[some_other_table]\r\nkey = \"value\"\r\nnested = [\"a\", \"b\"]\n\n[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n"
    toml_test.go:328: found LF not preceded by CR at byte 23:
        "[mcp_servers.codegraph]\ncommand = \"/usr/local/bin/codegraph\"\nargs = [\"serve\", \"--mcp\"]\n\r\n[some_other_table]\r\nkey = \"value\"\r\nnested = [\"a\", \"b\"]\r\n"
--- FAIL: TestSpliceTOMLTable_CRLFPreserved (0.00s)
    --- FAIL: TestSpliceTOMLTable_CRLFPreserved/append (0.00s)
    --- FAIL: TestSpliceTOMLTable_CRLFPreserved/replace (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.073s
FAIL
exit=1
```

Both subtests go RED: the appended/replaced codegraph block is written with bare `\n` into a
CRLF file in both the append and replace paths.

**Pre-revert gate:** `git diff --quiet -- internal/agents/toml.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/agents/toml.go`, then
`git diff --quiet -- internal/agents/toml.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestSpliceTOMLTable_CRLFPreserved$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/agents	0.049s`.

Full-package re-check after the revert: `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1`
→ `ok  	github.com/seanb4t/codegraph-go/internal/agents	5.875s`.
