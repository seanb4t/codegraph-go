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

---

## Family (b1) — D-13: re-shadowing install.go's explicit --target behind --yes turns TestInstall_YesWithExplicitTarget_HonoursTarget RED

**Test/guard:** `TestInstall_YesWithExplicitTarget_HonoursTarget` (`internal/cli/install_test.go`)
— asserts `install --target codex -y --location global` configures exactly Codex, never
widening to Claude via the `auto` default.

**What are we testing, and why?** Whether the guard catches the exact D-13 regression shape:
`--yes` shadowing an explicit `--target` again, silently reintroducing the released-binary
bug where `install --target codex --yes` configured whatever `auto` resolved to instead of
Codex.

**Pre-mutation gate:** `git diff --quiet -- internal/cli/install.go` — exit 0 (clean).

**Mutation applied:** `perl -pi -e 's/case cmd\.Flags\(\)\.Changed\("target"\):/case !yes \&\&
cmd.Flags().Changed("target"):/' internal/cli/install.go` (the pinned Family b1 site — the
only `case cmd.Flags().Changed("target"):` line in the file):

```diff
--- a/internal/cli/install.go
+++ b/internal/cli/install.go
@@ -109,7 +109,7 @@ func newInstallCmd() *cobra.Command {
 
 			var targets []agents.AgentTarget
 			switch {
-			case cmd.Flags().Changed("target"):
+			case !yes && cmd.Flags().Changed("target"):
 				// D-13: an explicit --target wins over --yes — --yes only
 				// supplies the non-interactive default when no target was
 				// named.
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestInstall_YesWithExplicitTarget_HonoursTarget$' -v`, `=== RUN` line kept, exit code
appended):

```
=== RUN   TestInstall_YesWithExplicitTarget_HonoursTarget
    install_test.go:882: expected explicit --target codex to configure Codex, got:
        Claude Code: configured
          created: /var/folders/.../T/TestInstall_YesWithExplicitTarget_HonoursTarget1544763511/001/.claude.json
          created: /var/folders/.../T/TestInstall_YesWithExplicitTarget_HonoursTarget1544763511/001/.claude/CLAUDE.md
          created: /var/folders/.../T/TestInstall_YesWithExplicitTarget_HonoursTarget1544763511/001/.claude/skills/codegraph/SKILL.md
          created: /var/folders/.../T/TestInstall_YesWithExplicitTarget_HonoursTarget1544763511/001/.claude/hooks/session-nudge.sh
          created: /var/folders/.../T/TestInstall_YesWithExplicitTarget_HonoursTarget1544763511/001/.claude/settings.json
          created: /var/folders/.../T/TestInstall_YesWithExplicitTarget_HonoursTarget1544763511/001/.claude/skills/codegraph/.codegraph-manifest.json
--- FAIL: TestInstall_YesWithExplicitTarget_HonoursTarget (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.509s
FAIL
exit=1
```

With `--yes` re-shadowing the explicit target, `--target codex -y` re-resolves to `auto`,
which falls back to Claude in a fresh fake home — exactly the released-binary behavior D-13
fixed.

**Pre-revert gate:** `git diff --quiet -- internal/cli/install.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/cli/install.go`, then
`git diff --quiet -- internal/cli/install.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestInstall_YesWithExplicitTarget_HonoursTarget$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/cli	0.509s`.

---

## Family (b2) — D-13: re-shadowing uninstall.go's explicit --target behind --yes turns TestUninstall_YesWithExplicitTarget_HonoursTarget RED

**Test/guard:** `TestUninstall_YesWithExplicitTarget_HonoursTarget` (`internal/cli/install_test.go`)
— asserts `uninstall --target codex --yes --location global` removes exactly Codex's
configuration, leaving Claude's `mcpServers.codegraph` entry untouched.

**What are we testing, and why?** The uninstall sibling of Family (b1): whether the guard
catches `--yes` shadowing an explicit `--target` in `uninstall.go`, which would resolve to
`all` and remove every installed agent's configuration instead of just the one named.

**Pre-mutation gate:** `git diff --quiet -- internal/cli/uninstall.go` — exit 0 (clean).

**Mutation applied:** `perl -pi -e 's/case cmd\.Flags\(\)\.Changed\("target"\):/case !yes \&\&
cmd.Flags().Changed("target"):/' internal/cli/uninstall.go` (the pinned Family b2 site — the
only `case cmd.Flags().Changed("target"):` line in the file):

```diff
--- a/internal/cli/uninstall.go
+++ b/internal/cli/uninstall.go
@@ -48,7 +48,7 @@ func newUninstallCmd() *cobra.Command {
 
 			var targets []agents.AgentTarget
 			switch {
-			case cmd.Flags().Changed("target"):
+			case !yes && cmd.Flags().Changed("target"):
 				// D-13: an explicit --target wins over --yes — --yes only
 				// supplies the non-interactive default when no target was
 				// named.
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestUninstall_YesWithExplicitTarget_HonoursTarget$' -v`, `=== RUN` line kept, exit code
appended):

```
=== RUN   TestUninstall_YesWithExplicitTarget_HonoursTarget
    install_test.go:920: expected --yes NOT to widen an explicit --target codex to Claude, got:
        Antigravity: not-configured
          not-found: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.gemini/config/mcp_config.json
          not-found: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.gemini/config/skills/codegraph/.codegraph-manifest.json
          not-found: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.gemini/config/skills/codegraph/SKILL.md
        Claude Code: removed
          removed: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.claude.json
          removed: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.claude/CLAUDE.md
          not-found: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.claude/settings.json
          removed: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.claude/skills/codegraph/.codegraph-manifest.json
          removed: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.claude/skills/codegraph/SKILL.md
          removed: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.claude/hooks/session-nudge.sh
          removed: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.claude/settings.json
          not-found: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.claude/hooks/pretooluse-nudge.sh
          not-found: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.claude/settings.json
        Codex CLI: removed
          removed: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.codex/config.toml
          removed: /var/folders/.../T/TestUninstall_YesWithExplicitTarget_HonoursTarget3948374326/001/.codex/AGENTS.md
        Cursor: not-configured
        Gemini CLI: not-configured
        Hermes Agent: not-configured
        Kiro: not-configured
        opencode: not-configured
--- FAIL: TestUninstall_YesWithExplicitTarget_HonoursTarget (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.449s
FAIL
exit=1
```

With `--yes` re-shadowing the explicit target, `--target codex --yes` re-resolves to `all`,
so Claude's (and every other registered target's) configuration is removed alongside Codex's
— the exact regression shape D-13 fixed on the uninstall side.

**Pre-revert gate:** `git diff --quiet -- internal/cli/uninstall.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/cli/uninstall.go`, then
`git diff --quiet -- internal/cli/uninstall.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestUninstall_YesWithExplicitTarget_HonoursTarget$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/cli	0.449s`.

Full-package re-check after both reverts: `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1`
→ `ok  	github.com/seanb4t/codegraph-go/internal/cli	17.205s`.

---

## Family (c1) — D-24: re-planting the trailing newline in checkboxDelegate.Render turns TestAgentPickerFootprintFitsDefaultPane RED

**Test/guard:** `TestAgentPickerFootprintFitsDefaultPane` (`internal/cli/tui/agentpicker_test.go`)
— asserts the agent picker's rendered view fits within 30 lines at a 100x30 window with the
real registry's 8 targets, and shows the help footer text `space: toggle`.

**What are we testing, and why?** Whether the guard catches the exact pre-fix defect FIX-03
regressed to: `checkboxDelegate.Render` writing its own trailing newline on top of
bubbles/v2/list's `populatedView` separator, costing each row 2 lines instead of 1 and
overflowing the 100x30 pane so the help footer never renders (D-24).

**Pre-mutation gate:** `git diff --quiet -- internal/cli/tui/agentpicker.go` — exit 0 (clean).

**Mutation applied:** `perl -pi -e 's/Fprintf\(w, "%s%s %s", cursor, box/Fprintf(w, "%s%s
%s\\n", cursor, box/' internal/cli/tui/agentpicker.go` (the plan's pinned Family c1 site — the
one `Fprintf(w, "%s%s %s", cursor, box` call, re-planting the byte-for-byte pre-fix newline):

```diff
--- a/internal/cli/tui/agentpicker.go
+++ b/internal/cli/tui/agentpicker.go
@@ -64,7 +64,7 @@ func (d *checkboxDelegate) Render(w io.Writer, m list.Model, index int, item lis
 	if index == m.Index() {
 		cursor = "> "
 	}
-	fmt.Fprintf(w, "%s%s %s", cursor, box, ai.target.DisplayName())
+	fmt.Fprintf(w, "%s%s %s\n", cursor, box, ai.target.DisplayName())
 }
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/tui/ -count=1
-run 'TestAgentPickerFootprintFitsDefaultPane$' -v`, `=== RUN` line dropped, exit code
appended):

```
    agentpicker_test.go:212: lipgloss.Height(view.Content) = 37, want <= 30 at 100x30 with 8 targets:
           Select agents to configure   
                                        
        > [ ] Antigravity               
                                        
          [ ] Claude Code               
                                        
          [ ] Codex CLI                 
                                        
          [ ] Cursor                    
                                        
          [ ] Gemini CLI                
                                        
          [ ] Hermes Agent              
                                        
          [ ] Kiro                      
                                        
          [ ] opencode                  
                                        
                                        
                                        
                                        
                                        
                                        
                                        
                                        
                                        
                                        
                                        
                                        
                                        
                                        
                                        
                                        
                                        
                                        
        space: toggle  enter: confirm  q/esc: cancel
--- FAIL: TestAgentPickerFootprintFitsDefaultPane (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli/tui	0.348s
FAIL
exit=1
```

The re-planted newline reproduces the exact pre-fix height (37, matching Task 1's own RED
transcript) — the footer line never appears in the printed capture (it scrolls past the 30-line
window this transcript's `head`-equivalent view still shows in full, but the `lipgloss.Height`
assertion catches the overflow directly).

**Pre-revert gate:** `git diff --quiet -- internal/cli/tui/agentpicker.go` — exit 1 (only the
planted diff).

**Revert:** `git checkout -- internal/cli/tui/agentpicker.go`, then
`git diff --quiet -- internal/cli/tui/agentpicker.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/tui/ -count=1
-run 'TestAgentPickerFootprintFitsDefaultPane$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/cli/tui	0.359s`.

---

## Family (c2) — D-24: re-planting the trailing newline in daemonDelegate.Render turns TestDaemonPickerFootprintFitsDefaultPane RED

**Test/guard:** `TestDaemonPickerFootprintFitsDefaultPane` (`internal/cli/tui/daemonpicker_test.go`)
— asserts the daemon picker's rendered view fits within 30 lines at a 100x30 window with 8
records, and shows the help footer text `enter: stop selected`.

**What are we testing, and why?** The daemon-picker sibling of Family (c1): whether the guard
catches `daemonDelegate.Render`'s identical trailing-newline defect (D-24).

**Pre-mutation gate:** `git diff --quiet -- internal/cli/tui/daemonpicker.go` — exit 0 (clean).

**Mutation applied:** `perl -pi -e 's/Fprintf\(w, "%s%s \(pid %d, up %s\)"/Fprintf(w, "%s%s
(pid %d, up %s)\\n"/' internal/cli/tui/daemonpicker.go` (the plan's pinned Family c2 site — the
one `Fprintf(w, "%s%s (pid %d, up %s)"` call, re-planting the byte-for-byte pre-fix newline):

```diff
--- a/internal/cli/tui/daemonpicker.go
+++ b/internal/cli/tui/daemonpicker.go
@@ -61,7 +61,7 @@ func (d daemonDelegate) Render(w io.Writer, m list.Model, index int, item list.I
 		cursor = "> "
 	}
 	age := time.Since(di.record.StartedAt).Round(time.Second)
-	fmt.Fprintf(w, "%s%s (pid %d, up %s)", cursor, filepath.Base(di.record.RepoRoot), di.record.PID, age)
+	fmt.Fprintf(w, "%s%s (pid %d, up %s)\n", cursor, filepath.Base(di.record.RepoRoot), di.record.PID, age)
 }
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/tui/ -count=1
-run 'TestDaemonPickerFootprintFitsDefaultPane$' -v`, `=== RUN` line dropped, exit code
appended):

```
    daemonpicker_test.go:384: lipgloss.Height(view.Content) = 37, want <= 30 at 100x30 with 8 records:
           Running daemons    
                              
        > r0 (pid 1000, up 0s)
                              
          r1 (pid 1001, up 0s)
                              
          r2 (pid 1002, up 0s)
                              
          r3 (pid 1003, up 0s)
                              
          r4 (pid 1004, up 0s)
                              
          r5 (pid 1005, up 0s)
                              
          r6 (pid 1006, up 0s)
                              
          r7 (pid 1007, up 0s)
                              
                              
                              
                              
                              
                              
                              
                              
                              
                              
                              
                              
                              
                              
                              
                              
                              
                              
        enter: stop selected  a: stop all  q/esc: cancel
--- FAIL: TestDaemonPickerFootprintFitsDefaultPane (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli/tui	0.457s
FAIL
exit=1
```

**Pre-revert gate:** `git diff --quiet -- internal/cli/tui/daemonpicker.go` — exit 1 (only the
planted diff).

**Revert:** `git checkout -- internal/cli/tui/daemonpicker.go`, then
`git diff --quiet -- internal/cli/tui/daemonpicker.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/tui/ -count=1
-run 'TestDaemonPickerFootprintFitsDefaultPane$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/cli/tui	0.329s`.

Full-package re-check after both reverts: `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/tui/
-count=1` → `ok  	github.com/seanb4t/codegraph-go/internal/cli/tui	0.318s`.

---

## Family (c3)

**Not run: the maintainer decided to skip it (2026-09-19).** tmux has been retired from the maintainer's machines and replaced by herdr. The maintainer chose to skip the local real-PTY RED/GREEN evidence rather than install tmux: "skip the tmux evidence, open a GH issue to consider a move from tmux to herdr, or if the e2e that we use tmux for is still needed (or if there are other options)".

- No tmux was installed, and `task test:tmux` was not run locally. There is therefore no tmux version, no GREEN `executed=6` line and no RED excerpt to record.
- The local guard for FIX-03 is the model-level footprint test (Families c1 and c2 above), which is RED on the pre-fix delegate and GREEN at HEAD.
- The re-anchored TTY-05 assertion (`space: toggle` plus all 8 display names) compiles under `go vet -tags tmux ./test/tmux/`. Its real-PTY run is left to the CI `tmux-e2e` job (ubuntu-latest, tmux 3.4).
- 07-09's post-scope-flip tmux re-run is skipped under the same decision.
- Follow-up: https://github.com/seanb4t/codegraph-go/issues/75 (move the harness to herdr, a pure-Go PTY or teatest; keep it CI-only; or retire it).

Maintainer decision: skip local tmux evidence for FIX-03 (c3 and the 07-09 re-run); CI tmux-e2e is the only real-PTY run; see #75.
Family (c3) verdict: not run (maintainer decision 2026-09-19); model-level guard c1/c2 RED→GREEN stands.

---

## Family (d1) — D-09: reverting the Scopes literal to global-only turns TestCapabilitiesDeclared/codex and TestCodex_SupportsLocation_GlobalAndLocal RED

**Test/guard:** `TestCapabilitiesDeclared` (codex row, `internal/agents/capabilities_test.go`)
and `TestCodex_SupportsLocation_GlobalAndLocal` (`internal/agents/codex_test.go`).

**What are we testing, and why?** Whether the guards catch the scope flip being silently
reverted — Codex's `Capabilities().Scopes` collapsing back to `{LocationGlobal}` — since every
other Codex behavior this plan added (local install, the trust Note, the shared skill, D-15's
read-only dirs) is reachable only once `SupportsLocation(LocationLocal)` is true.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/codex.go` — exit 0 (clean).

**Mutation applied:** `perl -pi -e 's/Scopes:       \[\]Location\{LocationGlobal, LocationLocal\},/Scopes:       []Location{LocationGlobal},/' internal/agents/codex.go`:

```diff
--- a/internal/agents/codex.go
+++ b/internal/agents/codex.go
@@ -47,7 +47,7 @@ func (t codexTarget) SupportsLocation(loc Location) bool {
  // hooks yet (07-07 adds codex-json).
  func (codexTarget) Capabilities() Capabilities {
  	return Capabilities{
-		Scopes:       []Location{LocationGlobal, LocationLocal},
+		Scopes:       []Location{LocationGlobal},
  		ConfigFormat: ConfigFormatTOML,
  		Hooks:        HooksNone,
  		MCPConfig:    codexConfigPath,
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestCodex_SupportsLocation_GlobalAndLocal$|TestCapabilitiesDeclared$' -v`, exit code
appended):

```
=== RUN   TestCapabilitiesDeclared
=== RUN   TestCapabilitiesDeclared/antigravity
=== RUN   TestCapabilitiesDeclared/claude
=== RUN   TestCapabilitiesDeclared/codex
    capabilities_test.go:186: Scopes = [global], want [global local]
=== RUN   TestCapabilitiesDeclared/cursor
=== RUN   TestCapabilitiesDeclared/gemini
=== RUN   TestCapabilitiesDeclared/hermes
=== RUN   TestCapabilitiesDeclared/kiro
=== RUN   TestCapabilitiesDeclared/opencode
--- FAIL: TestCapabilitiesDeclared (0.00s)
    --- PASS: TestCapabilitiesDeclared/antigravity (0.00s)
    --- PASS: TestCapabilitiesDeclared/claude (0.00s)
    --- FAIL: TestCapabilitiesDeclared/codex (0.00s)
    --- PASS: TestCapabilitiesDeclared/cursor (0.00s)
    --- PASS: TestCapabilitiesDeclared/gemini (0.00s)
    --- PASS: TestCapabilitiesDeclared/hermes (0.00s)
    --- PASS: TestCapabilitiesDeclared/kiro (0.00s)
    --- PASS: TestCapabilitiesDeclared/opencode (0.00s)
=== RUN   TestCodex_SupportsLocation_GlobalAndLocal
    codex_test.go:25: codex should support local (D-09 scope flip)
--- FAIL: TestCodex_SupportsLocation_GlobalAndLocal (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.121s
FAIL
exit=1
```

**Pre-revert gate:** `git diff --quiet -- internal/agents/codex.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/agents/codex.go`, then
`git diff --quiet -- internal/agents/codex.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1` →
`ok  	github.com/seanb4t/codegraph-go/internal/agents	2.674s`.

---

## Family (d2) — D-09/D-14: deleting the `installDeclaredSkill` call turns TestCapabilitiesMatchInstallWrites/codex and TestCodex_Install_Local_WritesConfigInstructionsAndSkill RED

**Test/guard:** `TestCapabilitiesMatchInstallWrites` (`codex/global` and `codex/local` leaves,
`internal/agents/capabilities_test.go`) and `TestCodex_Install_Local_WritesConfigInstructionsAndSkill`
(`internal/agents/codex_test.go`).

**What are we testing, and why?** Whether the guards catch Codex's `Install` silently dropping
the shared skill-package write — the exact regression shape the plan's own precondition names
(`installDeclaredSkill(&result, t, loc)` deleted from `Install`), leaving config.toml and
AGENTS.md written but the declared skill directory (`DescribePaths`) never populated.

**Note on the plan's own precondition check (accuracy over prediction):** the plan's `<verify>`
names a precondition of `rg -c -F 'installDeclaredSkill(&result, t, loc)' internal/agents/codex.go`
`= "1"` before planting. Run literally, this returns **2**, not 1 — `uninstallDeclaredSkill(&result,
t, loc)` on a different line also matches the literal substring `installDeclaredSkill(&result, t,
loc)`, since "uninstall" contains "install" as a substring (`un` + `install...`). This is a bug in
the plan's own grep pattern (a substring collision), not a real second call site — confirmed with a
corrected pattern: `rg -c -P '(?<!un)installDeclaredSkill\(&result, t, loc\)' internal/agents/codex.go`
→ `1`. The actual perl mutation below is unaffected by this: its pattern requires
`\n\s*installDeclaredSkill\(&result, t, loc\)\n` (whitespace only before the call), which cannot
match `\tuninstallDeclaredSkill(...)` since `un` is not whitespace — confirmed by the diff below
touching exactly the one call site in `Install`, `uninstallDeclaredSkill` in `Uninstall` untouched.
Recorded per plan-authored-verify-assertion guidance: verify the substance a different way and
record the deviation, rather than editing the check to force a match it does not have.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/codex.go` — exit 0 (clean).

**Mutation applied:** `perl -0pi -e 's/\n\s*installDeclaredSkill\(&result, t, loc\)\n/\n/'
internal/agents/codex.go`:

```diff
--- a/internal/agents/codex.go
+++ b/internal/agents/codex.go
@@ -215,8 +215,6 @@ func (t codexTarget) Install(loc Location, opts InstallOptions) WriteResult {
  		recordFile(&result, instrPath, fr, err)
  	}
 
-	installDeclaredSkill(&result, t, loc)
-
  	if note, err := codexTrustNote(loc); err != nil {
  		result.Errors = append(result.Errors, fmt.Errorf("resolve codex trust note: %w", err))
  	} else if note != "" {
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestCodex_Install_Local_WritesConfigInstructionsAndSkill$|TestCapabilitiesMatchInstallWrites$'
-v`, `=== RUN` lines for unaffected leaves dropped, exit code appended):

```
=== RUN   TestCapabilitiesMatchInstallWrites/codex/global
    capabilities_test.go:413: declared path ".../.agents/skills/codegraph/SKILL.md" was not among Install's created/updated/unchanged files: [{.../.codex/config.toml created} {.../.codex/AGENTS.md created}]
    capabilities_test.go:413: declared path ".../.agents/skills/codegraph/.codegraph-manifest.json" was not among Install's created/updated/unchanged files: [{.../.codex/config.toml created} {.../.codex/AGENTS.md created}]
=== RUN   TestCapabilitiesMatchInstallWrites/codex/local
    capabilities_test.go:413: declared path ".agents/skills/codegraph/SKILL.md" was not among Install's created/updated/unchanged files: [{.codex/config.toml created} {AGENTS.md created}]
    capabilities_test.go:413: declared path ".agents/skills/codegraph/.codegraph-manifest.json" was not among Install's created/updated/unchanged files: [{.codex/config.toml created} {AGENTS.md created}]
--- FAIL: TestCapabilitiesMatchInstallWrites (0.03s)
    --- FAIL: TestCapabilitiesMatchInstallWrites/codex/global (0.00s)
    --- FAIL: TestCapabilitiesMatchInstallWrites/codex/local (0.00s)
=== RUN   TestCodex_Install_Local_WritesConfigInstructionsAndSkill
    codex_test.go:60: expected shared skill SKILL.md at .../.agents/skills/codegraph/SKILL.md
--- FAIL: TestCodex_Install_Local_WritesConfigInstructionsAndSkill (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.108s
FAIL
exit=1
```

**Pre-revert gate:** `git diff --quiet -- internal/agents/codex.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/agents/codex.go`, then
`git diff --quiet -- internal/agents/codex.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1` →
`ok  	github.com/seanb4t/codegraph-go/internal/agents	2.674s`.

---

## Family (d3) — D-07: disabling Install's `tomlTableConflict` check turns TestCodex_Install_RefusesConflictingCodegraphTable RED

**Test/guard:** `TestCodex_Install_RefusesConflictingCodegraphTable` (`internal/agents/codex_test.go`).

**What are we testing, and why?** Whether the guard catches `Install` silently skipping the D-07
conflict check it added on top of `spliceTOMLTable`'s own internal (silent, no-error) conflict
handling — without the explicit check, an existing conflicting `codegraph` definition produces
`ActionUnchanged` with no error, instead of a named, visible failure.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/codex.go` — exit 0 (clean).

**Mutation applied:** the `if cerr := tomlTableConflict(existing, codexTOMLTable); cerr != nil {`
guard's condition replaced with an unreachable `if false {` (preserving `cerr`'s declaration so the
block still compiles), forcing every install through the splice branch regardless of a conflict:

```diff
--- a/internal/agents/codex.go
+++ b/internal/agents/codex.go
@@ -190,7 +190,8 @@ func (t codexTarget) Install(loc Location, opts InstallOptions) WriteResult {
  	} else {
  		existed := fileExists(configPath)
  		existing := readFileOrEmpty(configPath)
-		if cerr := tomlTableConflict(existing, codexTOMLTable); cerr != nil {
+		if false {
+			var cerr error
  			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", configPath, cerr))
  		} else {
  			updated := spliceTOMLTable(existing, codexTOMLTable, codexTableBody(opts.ExecPath))
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestCodex_Install_RefusesConflictingCodegraphTable$' -v`, exit code appended):

```
=== RUN   TestCodex_Install_RefusesConflictingCodegraphTable
    codex_test.go:133: expected a conflict error, got none: {Files:[{Path:.codex/config.toml Action:unchanged} {Path:AGENTS.md Action:created} {Path:.agents/skills/codegraph/SKILL.md Action:created} {Path:.agents/skills/codegraph/.codegraph-manifest.json Action:created}] Notes:[Codex loads this project's MCP server (.../TestCodex_Install_RefusesConflictingCodegraphTable2678842244/001) only once the project is trusted — accept Codex's trust prompt, or add `trust_level = "trusted"` under `[projects."..."]` in ~/.codex/config.toml (codegraph never writes this entry itself). The codegraph skill and the AGENTS.md block are read regardless of trust.] Errors:[]}
--- FAIL: TestCodex_Install_RefusesConflictingCodegraphTable (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.130s
FAIL
exit=1
```

Note the still-`ActionUnchanged` config.toml: `spliceTOMLTable`'s own internal
`tomlTableConflict` check independently refuses to WRITE a conflicting table (it returns
`content` unchanged rather than corrupting it), but with Install's explicit check gone, that
silent no-op is reported as an ordinary successful "unchanged" file rather than a named error —
exactly the D-07 "refused, not silently absorbed" property this guard protects.

**Pre-revert gate:** `git diff --quiet -- internal/agents/codex.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/agents/codex.go`, then
`git diff --quiet -- internal/agents/codex.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1` →
`ok  	github.com/seanb4t/codegraph-go/internal/agents	2.674s`.

---

## Family (d4) — D-10: dropping the trust Note append turns TestCodex_Install_Local_TrustNote RED

**Test/guard:** `TestCodex_Install_Local_TrustNote` (`internal/agents/codex_test.go`).

**What are we testing, and why?** Whether the guard catches `Install` silently dropping the D-10
trust advisory — the one place a user learns that Codex will not load the just-written project
MCP server until the project is trusted.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/codex.go` — exit 0 (clean).

**Mutation applied:** the `codexTrustNote(loc)` call-and-append block removed from `Install`
entirely (the `codexTrustNote` function itself is left in place, merely unused — Go does not
error on an unused top-level function):

```diff
--- a/internal/agents/codex.go
+++ b/internal/agents/codex.go
@@ -217,12 +217,6 @@ func (t codexTarget) Install(loc Location, opts InstallOptions) WriteResult {
 
  	installDeclaredSkill(&result, t, loc)
 
-	if note, err := codexTrustNote(loc); err != nil {
-		result.Errors = append(result.Errors, fmt.Errorf("resolve codex trust note: %w", err))
-	} else if note != "" {
-		result.Notes = append(result.Notes, note)
-	}
-
  	return result
  }
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestCodex_Install_Local_TrustNote$' -v`, exit code appended):

```
    codex_test.go:97: expected exactly one trust note, got 0: []
--- FAIL: TestCodex_Install_Local_TrustNote (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.095s
FAIL
exit=1
```

**Pre-revert gate:** `git diff --quiet -- internal/agents/codex.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/agents/codex.go`, then
`git diff --quiet -- internal/agents/codex.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1` →
`ok  	github.com/seanb4t/codegraph-go/internal/agents	2.674s`.

Family (d) verdict: all four scope-flip guards (D-09's Scopes literal, D-14's shared-skill write,
D-07's conflict refusal, D-10's trust Note) demonstrated RED against a real planted mutation and
reverted byte-clean; internal/agents is GREEN after every revert.

---

## Family (e1) — D-11: an early `return nil` in `instructionsRequestedElsewhere` turns TestSharedAgentsMD_KeptWhileOtherConfigured RED

**Test/guard:** `TestSharedAgentsMD_KeptWhileOtherConfigured` (`internal/agents/shared_test.go`).

**What are we testing, and why?** Whether the guard catches the D-11 requester-detection helper
itself being neutered — `instructionsRequestedElsewhere` always reporting "no one else needs
this file," which would make BOTH `codexTarget.Uninstall` and `opencodeTarget.Uninstall`
unconditionally strip the shared `AGENTS.md` block the moment either agent uninstalls, even
while the other is still fully configured.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/shared.go` — exit 0 (clean).

**Mutation applied:** an early `return nil` inserted immediately after
`instructionsRequestedElsewhere`'s signature line (the pinned Family e1 site):

```diff
--- a/internal/agents/shared.go
+++ b/internal/agents/shared.go
@@ -779,6 +779,7 @@ func upsertInstructionsEntry(filePath, startMarker, endMarker, content string) (
 // other Capabilities-derived helper in this package.
 func instructionsRequestedElsewhere(path string, loc Location, self TargetID) []TargetID {
+	return nil
 	absPath, err := filepath.Abs(filepath.Clean(path))
 	if err != nil {
 		return nil
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestSharedAgentsMD_KeptWhileOtherConfigured$' -v`, exit code appended):

```
    shared_test.go:586: expected AGENTS.md kept (ActionKept) in result, got [{Path:.codex/config.toml Action:removed} {Path:AGENTS.md Action:removed} {Path:.agents/skills/codegraph/.codegraph-manifest.json Action:updated} {Path:.agents/skills/codegraph/SKILL.md Action:kept}]
--- FAIL: TestSharedAgentsMD_KeptWhileOtherConfigured (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.091s
FAIL
exit=1
```

With the helper stubbed to always report zero requesters, codex's uninstall strips the shared
`AGENTS.md` block outright — the exact D-11 regression shape this guard exists to catch.

**Pre-revert gate:** `git diff --quiet -- internal/agents/shared.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/agents/shared.go`, then
`git diff --quiet -- internal/agents/shared.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestSharedAgentsMD_KeptWhileOtherConfigured$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/agents	0.127s`.

---

## Family (e2) — D-11: removing opencode.go's gate turns TestSharedAgentsMD_UninstallOrders/opencode_then_codex RED

**Test/guard:** `TestSharedAgentsMD_UninstallOrders/opencode_then_codex/{preexisting,absent}`
(`internal/agents/shared_test.go`).

**What are we testing, and why?** Whether the guard catches `opencodeTarget.Uninstall` skipping
the D-11 gate specifically (as opposed to Family (e1)'s package-wide helper failure) — the
regression shape where codex's own gate keeps working but opencode's own call site is deleted or
disabled, so opencode alone always strips the shared block regardless of who else still needs it.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/opencode.go` — exit 0 (clean).

**Mutation applied:** the `len(others) > 0` gate condition short-circuited to always-false via
`false &&` (keeping `others` referenced so the file still compiles) — the pinned Family e2 site,
the one `else if others := instructionsRequestedElsewhere(instrPath, loc, t.ID()); len(others) > 0 {`
line in `opencodeTarget.Uninstall`:

```diff
--- a/internal/agents/opencode.go
+++ b/internal/agents/opencode.go
@@ -334,7 +334,7 @@ func (t opencodeTarget) Uninstall(loc Location) WriteResult {
 
 	if instrPath, err := opencodeInstructionsPath(loc); err != nil {
 		result.Errors = append(result.Errors, fmt.Errorf("resolve opencode instructions path: %w", err))
-	} else if others := instructionsRequestedElsewhere(instrPath, loc, t.ID()); len(others) > 0 {
+	} else if others := instructionsRequestedElsewhere(instrPath, loc, t.ID()); false && len(others) > 0 {
 		// D-11: the repo-root AGENTS.md is shared with codex at local
 		// scope — leave the marker block in place while another
 		// registered target still declares this same file and reports
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestSharedAgentsMD_UninstallOrders$|TestOwnershipSharedInstructions$' -v`, unaffected
subtests' `--- PASS` lines kept, exit code appended):

```
--- PASS: TestOwnershipSharedInstructions (0.11s)
    --- PASS: TestOwnershipSharedInstructions/codex_then_opencode/clean (0.05s)
    --- PASS: TestOwnershipSharedInstructions/codex_then_opencode/foreign-codegraph-dir (0.01s)
    --- PASS: TestOwnershipSharedInstructions/opencode_then_codex/clean (0.01s)
    --- PASS: TestOwnershipSharedInstructions/opencode_then_codex/foreign-codegraph-dir (0.02s)
    --- PASS: TestOwnershipSharedInstructions/target_all/clean (0.01s)
    --- PASS: TestOwnershipSharedInstructions/target_all/foreign-codegraph-dir (0.01s)
    shared_test.go:530: expected AGENTS.md kept (ActionKept) in result, got [{Path:opencode.jsonc Action:removed} {Path:AGENTS.md Action:removed} {Path:.agents/skills/codegraph/.codegraph-manifest.json Action:updated} {Path:.agents/skills/codegraph/SKILL.md Action:kept}]
    shared_test.go:530: expected AGENTS.md kept (ActionKept) in result, got [{Path:opencode.jsonc Action:removed} {Path:AGENTS.md Action:removed} {Path:.agents/skills/codegraph/.codegraph-manifest.json Action:updated} {Path:.agents/skills/codegraph/SKILL.md Action:kept}]
--- FAIL: TestSharedAgentsMD_UninstallOrders (0.28s)
    --- PASS: TestSharedAgentsMD_UninstallOrders/codex_then_opencode/preexisting (0.00s)
    --- PASS: TestSharedAgentsMD_UninstallOrders/codex_then_opencode/absent (0.01s)
    --- FAIL: TestSharedAgentsMD_UninstallOrders/opencode_then_codex/preexisting (0.00s)
    --- FAIL: TestSharedAgentsMD_UninstallOrders/opencode_then_codex/absent (0.01s)
    --- PASS: TestSharedAgentsMD_UninstallOrders/target_all/preexisting (0.01s)
    --- PASS: TestSharedAgentsMD_UninstallOrders/target_all/absent (0.25s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.513s
FAIL
exit=1
```

**Note on the observed shape (accuracy over prediction, per the established convention — Family
(a1) and the 07-05-SUMMARY.md grep-bug precedent):** the plan text predicted this mutation would
also turn `TestOwnershipSharedInstructions/opencode_then_codex` RED. It does not, and the reason
is structural, not a bug in either test: `TestOwnershipSharedInstructions` (Task 2) asserts only
END-of-sequence state (every foreign byte identical, every codegraph entry gone) — it never
inspects the intermediate `FileResult` from opencode's OWN first uninstall call. With this
mutation, opencode strips the shared block on its own turn (the bug), but by the time codex's
(still-correct) gate runs second, opencode's own MCP entry is already gone, so
`instructionsRequestedElsewhere` correctly reports no remaining requester and codex's own
`removeMarkedSection` call is a harmless no-op against an already-empty span. The end state is
therefore identical whether or not opencode's own gate fired — the defect is real (a user relying
on Codex's block staying up while opencode is uninstalled first, in the general case where codex
uninstalls LATER, would lose it prematurely) but is only OBSERVABLE at the intermediate step,
which is exactly what `TestSharedAgentsMD_UninstallOrders`'s per-step `assertAgentsMDKept` check
was designed to catch — and it does, RED on both `preexisting` and `absent` subtests. This is
sufficient positive control for the D-11 gate as specified; `TestOwnershipSharedInstructions`'s
scope (end-state-only, matching its own Task 2 behavior spec) is unaffected by this finding.

**Pre-revert gate:** `git diff --quiet -- internal/agents/opencode.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/agents/opencode.go`, then
`git diff --quiet -- internal/agents/opencode.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestSharedAgentsMD_UninstallOrders$|TestOwnershipSharedInstructions$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/agents	0.304s`.

---

## Family (e3) — D-12: dropping the override Note append turns TestCodex_Install_OverrideNote RED

**Test/guard:** `TestCodex_Install_OverrideNote/{local,global}` (`internal/agents/codex_test.go`).

**What are we testing, and why?** Whether the guard catches `Install` silently dropping the D-12
`AGENTS.override.md`-shadow advisory — the one place a user learns that Codex will not see the
codegraph block codegraph just wrote to `AGENTS.md` because an override file shadows it.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/codex.go` — exit 0 (clean).

**Mutation applied:** the entire D-12 override-detection-and-Note block removed from `Install`
(the instructions step's `recordFile` call is untouched — only the override check that follows it
is deleted):

```diff
--- a/internal/agents/codex.go
+++ b/internal/agents/codex.go
@@ -214,18 +214,6 @@ func (t codexTarget) Install(loc Location, opts InstallOptions) WriteResult {
 		fr, err := upsertInstructionsEntry(instrPath, codegraphSectionStart, codegraphSectionEnd, instructionsBody())
 		recordFile(&result, instrPath, fr, err)
 
-		// D-12: an AGENTS.override.md beside the instructions file Codex
-		// reads shadows AGENTS.md for Codex — the block is still written
-		// above (a later install of the override's content could still
-		// pull it in), but the user should know Codex will not see it
-		// until then. codegraph never writes AGENTS.override.md itself.
-		overridePath := filepath.Join(filepath.Dir(instrPath), "AGENTS.override.md")
-		if fileExists(overridePath) {
-			result.Notes = append(result.Notes, fmt.Sprintf(
-				"%s shadows %s for Codex — Codex will not see the codegraph block there until the override includes it (codegraph never writes AGENTS.override.md itself)",
-				overridePath, instrPath,
-			))
-		}
 	}
 
 	installDeclaredSkill(&result, t, loc)
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestCodex_Install_OverrideNote$' -v`, exit code appended):

```
    codex_test.go:508: expected exactly one Note naming AGENTS.override.md, got 0: [Codex loads this project's MCP server (/var/folders/.../TestCodex_Install_OverrideNotelocal718478490/002) only once the project is trusted — accept Codex's trust prompt, or add `trust_level = "trusted"` under `[projects."/var/folders/.../TestCodex_Install_OverrideNotelocal718478490/002"]` in ~/.codex/config.toml (codegraph never writes this entry itself). The codegraph skill and the AGENTS.md block are read regardless of trust.]
    codex_test.go:540: expected exactly one Note naming AGENTS.override.md, got 0: []
--- FAIL: TestCodex_Install_OverrideNote (0.01s)
    --- FAIL: TestCodex_Install_OverrideNote/local (0.00s)
    --- FAIL: TestCodex_Install_OverrideNote/global (0.01s)
    --- PASS: TestCodex_Install_OverrideNote/no_override_present (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.101s
FAIL
exit=1
```

The `no_override_present` subtest correctly stays green throughout — it asserts the ABSENCE of an
override Note, which a stub that never adds one still satisfies trivially.

**Pre-revert gate:** `git diff --quiet -- internal/agents/codex.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/agents/codex.go`, then
`git diff --quiet -- internal/agents/codex.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestCodex_Install_OverrideNote$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/agents	0.132s`.

Full-package re-check after all three reverts: `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/
-count=1` → `ok  	github.com/seanb4t/codegraph-go/internal/agents	26.659s`.

Family (e) verdict: the D-11 package-wide requester-detection helper (e1), the D-11 opencode-side
gate call site specifically (e2), and the D-12 override Note (e3) all demonstrated RED against a
real planted mutation and reverted byte-clean; internal/agents is GREEN after every revert. e2's
finding that `TestOwnershipSharedInstructions` (an end-state-only guard, per its own Task 2
behavior spec) does not itself go RED for that mutation is recorded above as a scope observation,
not a defect — `TestSharedAgentsMD_UninstallOrders`'s per-step assertion is the guard that
demonstrably catches this exact regression shape.

---

## Family (f1) — D-19: the local guard's trailing `exit 0` turned into `exit 2` turns TestCodexPreToolUseGuard/local/binary_exits_nonzero RED

**Test/guard:** `TestCodexPreToolUseGuard/local/*` (`internal/agents/codex_pretooluse_test.go`).

**What are we testing, and why?** Whether the guard catches the local guard's own final "this hook
never blocks, never reports an error" contract (D-16) being violated — the script's LAST line,
reached after the child binary runs regardless of that binary's own exit code, is what makes the
guard's own exit status always 0 for Codex's hook engine.

**Pre-mutation gate:** `git diff --quiet -- .codex/hooks/codegraph-pretooluse-local.sh` — exit 0
(clean).

**Mutation applied:** `perl -0pi -e 's/\nexit 0\n\z/\nexit 2\n/' .codex/hooks/codegraph-pretooluse-local.sh`
(the pinned Family f1 site — the script's own final line):

```diff
--- a/.codex/hooks/codegraph-pretooluse-local.sh
+++ b/.codex/hooks/codegraph-pretooluse-local.sh
@@ -33,4 +33,4 @@ if [ ! -f "$codegraph_bin" ] || [ ! -x "$codegraph_bin" ]; then
   exit 0
 fi
 "$codegraph_bin" hook pretooluse --harness codex
-exit 0
+exit 2
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestCodexPreToolUseGuard$' -v`, `=== RUN` lines dropped, exit code appended):

```
    codex_pretooluse_test.go:418: guard exit = 2, want 0 (stdout "STUB-OUT\n", stderr "")
    codex_pretooluse_test.go:418: guard exit = 2, want 0 (stdout "", stderr "")
    codex_pretooluse_test.go:418: guard exit = 2, want 0 (stdout "", stderr ".../codegraph-pretooluse.sh: line 35: NNNNN Segmentation fault: 11  \"$codegraph_bin\" hook pretooluse --harness codex\n")
    codex_pretooluse_test.go:418: guard exit = 2, want 0 (stdout "STUB-OUT\n", stderr ".../codegraph: line 2: cat: No such file or directory\n")
--- FAIL: TestCodexPreToolUseGuard (0.70s)
    --- FAIL: TestCodexPreToolUseGuard/local/indexed_binary_ok (0.09s)
    --- PASS: TestCodexPreToolUseGuard/local/not_indexed (0.04s)
    --- PASS: TestCodexPreToolUseGuard/local/codegraph_is_file (0.05s)
    --- PASS: TestCodexPreToolUseGuard/local/binary_missing (0.05s)
    --- PASS: TestCodexPreToolUseGuard/local/binary_not_executable (0.04s)
    --- FAIL: TestCodexPreToolUseGuard/local/binary_exits_nonzero (0.08s)
    --- FAIL: TestCodexPreToolUseGuard/local/binary_crashes (0.11s)
    --- FAIL: TestCodexPreToolUseGuard/local/root_from_own_path_with_empty_path (0.08s)
    --- PASS: TestCodexPreToolUseGuard/global/indexed_pwd (0.06s)
    --- PASS: TestCodexPreToolUseGuard/global/not_indexed_pwd (0.04s)
    --- PASS: TestCodexPreToolUseGuard/global/binary_exits_nonzero (0.06s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.792s
FAIL
exit=1
```

Every LOCAL subtest that actually reaches the final line (the binary starts, however it behaves)
turns RED, not only `binary_exits_nonzero` — the mutation is on the script's shared final line,
so any case that gets that far is affected. The GLOBAL subtests are untouched, since the global
template is a separate file this mutation never touches.

**Pre-revert gate:** `git diff --quiet -- .codex/hooks/codegraph-pretooluse-local.sh` — exit 1
(only the planted diff).

**Revert:** `git checkout -- .codex/hooks/codegraph-pretooluse-local.sh`, then
`git diff --quiet -- .codex/hooks/codegraph-pretooluse-local.sh` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestCodexPreToolUseGuard$'` → `ok  	github.com/seanb4t/codegraph-go/internal/agents	0.567s`.

---

## Family (f2) — D-22: the local guard's `$0`-derived root replaced by `${PWD:-.}` turns TestCodexPreToolUseGuard/local/indexed_binary_ok RED

**Test/guard:** `TestCodexPreToolUseGuard/local/*` (`internal/agents/codex_pretooluse_test.go`).

**What are we testing, and why?** Whether the guard catches the local guard silently switching
from deriving its repo root from its own invocation path (`$0`, D-22 — Codex has no
`CLAUDE_PROJECT_DIR` equivalent) to reading `$PWD`/cwd instead, the shape the GLOBAL guard
legitimately uses. `TestCodexPreToolUseGuard`'s own negative control (`bogusCwd`, added in
commit `602e30a3` after an earlier, ineffective attempt at faking `$PWD` alone was empirically
found to be silently self-healed by the shell — see that commit's message) sets every LOCAL
subtest's actual OS-level working directory to an unrelated, un-indexed directory, so only a
correct `$0`-based derivation can still find the real project's `.codegraph`.

**Pre-mutation gate:** `git diff --quiet -- .codex/hooks/codegraph-pretooluse-local.sh` — exit 0
(clean).

**Mutation applied:** the three-line `$0`-strip root derivation collapsed into a single `${PWD:-.}`
read (the pinned Family f2 site):

```diff
--- a/.codex/hooks/codegraph-pretooluse-local.sh
+++ b/.codex/hooks/codegraph-pretooluse-local.sh
@@ -24,9 +24,7 @@
 # 4. The binary runs as a child with this script's stdin inherited,
 #    through the Codex envelope (--harness codex, D-21); this hook never
 #    blocks, denies, or reports a hook error — it exits 0 on every path.
-hooks_dir=${0%/*}
-codex_dir=${hooks_dir%/*}
-root=${codex_dir%/*}
+root=${PWD:-.}
 [ -d "$root/.codegraph" ] || exit 0
 codegraph_bin='@codegraph-exec-path@'
 if [ ! -f "$codegraph_bin" ] || [ ! -x "$codegraph_bin" ]; then
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestCodexPreToolUseGuard$' -v`, `=== RUN` lines dropped, exit code appended):

```
    codex_pretooluse_test.go:423: binary started = false, want true
    codex_pretooluse_test.go:423: binary started = false, want true
    codex_pretooluse_test.go:423: binary started = false, want true
    codex_pretooluse_test.go:423: binary started = false, want true
--- FAIL: TestCodexPreToolUseGuard (0.44s)
    --- FAIL: TestCodexPreToolUseGuard/local/indexed_binary_ok (0.05s)
    --- PASS: TestCodexPreToolUseGuard/local/not_indexed (0.03s)
    --- PASS: TestCodexPreToolUseGuard/local/codegraph_is_file (0.03s)
    --- PASS: TestCodexPreToolUseGuard/local/binary_missing (0.03s)
    --- PASS: TestCodexPreToolUseGuard/local/binary_not_executable (0.04s)
    --- FAIL: TestCodexPreToolUseGuard/local/binary_exits_nonzero (0.03s)
    --- FAIL: TestCodexPreToolUseGuard/local/binary_crashes (0.04s)
    --- FAIL: TestCodexPreToolUseGuard/local/root_from_own_path_with_empty_path (0.04s)
    --- PASS: TestCodexPreToolUseGuard/global/indexed_pwd (0.06s)
    --- PASS: TestCodexPreToolUseGuard/global/not_indexed_pwd (0.03s)
    --- PASS: TestCodexPreToolUseGuard/global/binary_exits_nonzero (0.05s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.529s
FAIL
exit=1
```

`local/indexed_binary_ok` (named explicitly in this plan's own Task 3 `<verify>` OR-clause)
is among the four RED subtests, confirming the negative control works: with the real cwd wrong
and no correct `$0`-derived fallback, the mutated guard incorrectly treats the indexed project
as un-indexed.

**Pre-revert gate:** `git diff --quiet -- .codex/hooks/codegraph-pretooluse-local.sh` — exit 1
(only the planted diff).

**Revert:** `git checkout -- .codex/hooks/codegraph-pretooluse-local.sh`, then
`git diff --quiet -- .codex/hooks/codegraph-pretooluse-local.sh` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestCodexPreToolUseGuard$'` → `ok  	github.com/seanb4t/codegraph-go/internal/agents	0.602s`.

---

## Family (f3) — D-21: falling back to `CLAUDE_CODE_SESSION_ID` turns TestHookPreToolUseCodex_ForcedErrorContract/claude_env_session_ignored RED

**Test/guard:** `TestHookPreToolUseCodex_ForcedErrorContract/claude_env_session_ignored`
(`internal/cli/hook_pretooluse_codex_test.go`).

**What are we testing, and why?** Whether the guard catches the Codex envelope silently adopting
Claude's own env-var session fallback (`runHookPreToolUse` reads `CLAUDE_CODE_SESSION_ID` when
stdin's `session_id` is empty) — Codex's live PreToolUse stdin never carries any such variable
(07-LIVE-SESSIONS.md B5), so a Codex session with no stdin `session_id` must stay silent, not
adopt an unrelated Claude environment variable that happens to be set.

**Pre-mutation gate:** `git diff --quiet -- internal/cli/hook_pretooluse.go` — exit 0 (clean).

**Mutation applied:** `runHookPreToolUseCodex`'s direct `nudge.SessionKey(event.SessionID, ...)`
call replaced with the same env-fallback shape `runHookPreToolUse` already uses for Claude (the
pinned Family f3 site — the only occurrence of this exact call in the file):

```diff
--- a/internal/cli/hook_pretooluse.go
+++ b/internal/cli/hook_pretooluse.go
@@ -254,7 +254,11 @@ func runHookPreToolUseCodex(in io.Reader, out io.Writer) {
 		return
 	}
 
-	key, ok := nudge.SessionKey(event.SessionID, event.AgentID)
+	session := os.Getenv("CLAUDE_CODE_SESSION_ID")
+	if session == "" {
+		session = event.SessionID
+	}
+	key, ok := nudge.SessionKey(session, event.AgentID)
 	if !ok {
 		return
 	}
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestHookPreToolUseCodex_ForcedErrorContract$' -v`, `=== RUN` lines dropped, exit code
appended):

```
    hook_pretooluse_codex_test.go:186: stdout = "{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"additionalContext\":\"This repo has a codegraph index: codegraph_explore (CLI: `codegraph explore`) returns the matching symbols' source and call paths for where-is-X and how-does-Y questions.\"}}\n", want ""
--- FAIL: TestHookPreToolUseCodex_ForcedErrorContract (0.04s)
    --- PASS: TestHookPreToolUseCodex_ForcedErrorContract/no_stdin (0.00s)
    --- PASS: TestHookPreToolUseCodex_ForcedErrorContract/malformed_json (0.00s)
    --- PASS: TestHookPreToolUseCodex_ForcedErrorContract/oversized_input (0.00s)
    --- PASS: TestHookPreToolUseCodex_ForcedErrorContract/max_size_input_fires (0.00s)
    --- PASS: TestHookPreToolUseCodex_ForcedErrorContract/no_session_id (0.00s)
    --- FAIL: TestHookPreToolUseCodex_ForcedErrorContract/claude_env_session_ignored (0.00s)
    --- PASS: TestHookPreToolUseCodex_ForcedErrorContract/non_pretooluse_event (0.00s)
    [... 18 further PASS lines, all unaffected ...]
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.510s
FAIL
exit=1
```

Only the one targeted subtest turns RED — every other subtest in the 24-case table is unaffected,
confirming the mutation is isolated to the env-fallback behavior.

**Pre-revert gate:** `git diff --quiet -- internal/cli/hook_pretooluse.go` — exit 1 (only the
planted diff).

**Revert:** `git checkout -- internal/cli/hook_pretooluse.go`, then
`git diff --quiet -- internal/cli/hook_pretooluse.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestHookPreToolUseCodex_ForcedErrorContract$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/cli	0.466s`.

---

## Family (f4) — D-21: `codexShellCommand` reading the argv's first element turns TestHookPreToolUseCodex_ForcedErrorContract/command_argv_last_element_fires RED

**Test/guard:** `TestHookPreToolUseCodex_ForcedErrorContract/command_argv_last_element_fires`
(`internal/cli/hook_pretooluse_codex_test.go`).

**What are we testing, and why?** Whether the guard catches `codexShellCommand` silently reading
the WRONG end of an argv-form `tool_input.command` — D-21 specifies the LAST element (mirroring a
shell invocation like `["bash", "-lc", "<command>"]`, where the actual command is the final
argument), not the first (which would usually be an interpreter name like `bash`, never itself a
search command).

**Pre-mutation gate:** `git diff --quiet -- internal/cli/hook_pretooluse.go` — exit 0 (clean).

**Mutation applied:** the last-element index into the decoded argv replaced with the first-element
index (the pinned Family f4 site):

```diff
--- a/internal/cli/hook_pretooluse.go
+++ b/internal/cli/hook_pretooluse.go
@@ -86,7 +86,7 @@ func codexShellCommand(raw json.RawMessage) (string, bool) {
 			return "", false
 		}
 		var last string
-		if err := json.Unmarshal(arr[len(arr)-1], &last); err != nil || last == "" {
+		if err := json.Unmarshal(arr[0], &last); err != nil || last == "" {
 			return "", false
 		}
 		return last, true
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestHookPreToolUseCodex_ForcedErrorContract$' -v`, `=== RUN` lines dropped, exit code
appended):

```
--- FAIL: TestHookPreToolUseCodex_ForcedErrorContract (0.01s)
    --- PASS: TestHookPreToolUseCodex_ForcedErrorContract/no_stdin (0.00s)
    [... 16 further PASS lines, including command_single_element_argv_fires (single-element
    array: first == last, so this subtest is unaffected) ...]
    --- FAIL: TestHookPreToolUseCodex_ForcedErrorContract/command_argv_last_element_fires (0.00s)
    [... 6 further PASS lines ...]
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.478s
FAIL
exit=1
```

Only the multi-element argv case (`["bash", "-lc", "rg -n Alpha ."]`) turns RED — reading `arr[0]`
("bash") instead of `arr[len(arr)-1]` ("rg -n Alpha .") makes `hookQualifies` reject a non-search
first word, so the nudge silently fails to fire. The single-element-argv subtest is unaffected
precisely because its first and last elements coincide.

**Pre-revert gate:** `git diff --quiet -- internal/cli/hook_pretooluse.go` — exit 1 (only the
planted diff).

**Revert:** `git checkout -- internal/cli/hook_pretooluse.go`, then
`git diff --quiet -- internal/cli/hook_pretooluse.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestHookPreToolUseCodex_ForcedErrorContract$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/cli	0.437s`.

---

## Family (f5) — D-22: dropping the Go core's own `cwd`/`.codegraph` re-check turns TestHookPreToolUseCodex_ForcedErrorContract/cwd_not_indexed RED

**Test/guard:** `TestHookPreToolUseCodex_ForcedErrorContract/{cwd_missing,cwd_relative,cwd_not_indexed,cwd_codegraph_is_file}`
(`internal/cli/hook_pretooluse_codex_test.go`).

**What are we testing, and why?** Whether the guard catches `runHookPreToolUseCodex` silently
losing its OWN independent re-check that stdin's `cwd` is absolute and holds a `.codegraph`
directory (D-22) — this Go-level check exists precisely because the shell guard is not treated as
a security boundary the adapter can rely on alone; without it, a qualifying Bash event fires
regardless of whether the named `cwd` is even indexed.

**Pre-mutation gate:** `git diff --quiet -- internal/cli/hook_pretooluse.go` — exit 0 (clean).

**Mutation applied:** both `cwd` gating `if` blocks replaced with no-op reads of the same
functions, keeping `path/filepath` and `os` imports referenced so the mutation isolates the
LOGIC change from an unrelated build failure (the pinned Family f5 site):

```diff
--- a/internal/cli/hook_pretooluse.go
+++ b/internal/cli/hook_pretooluse.go
@@ -247,12 +247,8 @@ func runHookPreToolUseCodex(in io.Reader, out io.Writer) {
 	if !hookQualifies(nudge.ToolShell, command) {
 		return
 	}
-	if event.Cwd == "" || !filepath.IsAbs(event.Cwd) {
-		return
-	}
-	if info, statErr := os.Stat(filepath.Join(event.Cwd, ".codegraph")); statErr != nil || !info.IsDir() {
-		return
-	}
+	_ = filepath.IsAbs(event.Cwd) // Family (f5): cwd re-check removed
+	_ = os.Stat
 
 	key, ok := nudge.SessionKey(event.SessionID, event.AgentID)
 	if !ok {
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestHookPreToolUseCodex_ForcedErrorContract$' -v`, `=== RUN` lines dropped, exit code
appended):

```
--- FAIL: TestHookPreToolUseCodex_ForcedErrorContract (0.02s)
    [... 18 PASS lines ...]
    --- FAIL: TestHookPreToolUseCodex_ForcedErrorContract/cwd_missing (0.00s)
    --- FAIL: TestHookPreToolUseCodex_ForcedErrorContract/cwd_relative (0.00s)
    --- FAIL: TestHookPreToolUseCodex_ForcedErrorContract/cwd_not_indexed (0.00s)
    --- FAIL: TestHookPreToolUseCodex_ForcedErrorContract/cwd_codegraph_is_file (0.00s)
    [... 2 further PASS lines ...]
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.491s
FAIL
exit=1
```

All four `cwd_*` edge cases turn RED together — the mutation removes the whole re-check, so every
subtest exercising it fails, `cwd_not_indexed` (named explicitly in this plan's own Task 3
`<verify>`) among them.

**Pre-revert gate:** `git diff --quiet -- internal/cli/hook_pretooluse.go` — exit 1 (only the
planted diff).

**Revert:** `git checkout -- internal/cli/hook_pretooluse.go`, then
`git diff --quiet -- internal/cli/hook_pretooluse.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestHookPreToolUseCodex_ForcedErrorContract$'` →
`ok  	github.com/seanb4t/codegraph-go/internal/cli	0.452s`.

Full-package re-check after all five reverts: `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/
./internal/cli/ -count=1` → both `ok`.

Family (f) verdict: the local guard's exit-0 contract (f1), the local guard's `$0`-vs-`$PWD` root
derivation (f2, with a corrected negative control after an earlier attempt was found ineffective —
see commit `602e30a3`), the adapter's Claude-env-fallback exclusion (f3), the argv last-element
extraction (f4), and the Go core's own independent `cwd`/`.codegraph` re-check (f5) all
demonstrated RED against a real planted mutation and reverted byte-clean; both packages are GREEN
after every revert.
