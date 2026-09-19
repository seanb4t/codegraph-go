# 06-MUTATION-LOG — Claude Code PreToolUse Nudge

**Phase:** 06-claude-code-pretooluse-nudge
**Date:** 2026-09-19
**Scope:** every guard this phase adds is positive-controlled here (D-00, D-16, rule
`84d1gfpywd`). Families, one per owning plan:

- Family (a) — plan 06-01: the PreToolUse guard script and its install-time rendering.
  (a1) the guard's final `exit 0` turned into `exit 2` turns
  `TestPreToolUseGuard/binary_exits_nonzero` RED; (a2) `shellSingleQuote` reduced to naive
  wrapping turns `TestRenderPreToolGuard/quote_and_space_in_path` RED.
- Family (b) — plan 06-02: the classifier corpora and the nudge-text drift guards.
- Family (c) — plan 06-03: the cooldown and the subcommand-level D-16 contract suite
  (including (c4)/(c5), a returned error reaching cobra's exit-1 path and an emitted
  `permissionDecision`).
- Family (d) — plan 06-04: the sticky opt-in, uninstall and ownership.
- Family (e) — plan 06-05: the dogfooded registration and the CLI opt-in.

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

## Family (a1) — D-16: an `exit 2` guard tail turns TestPreToolUseGuard RED

**Test/guard:** `TestPreToolUseGuard` (`internal/agents/claude_pretooluse_test.go`) — the
guard-level D-16 exec suite over the REAL embedded template (`claudeassets.PreToolUseGuardTemplate`,
rendered with stub binaries), asserting exit 0 on every path.

**What are we testing, and why?** Whether the suite actually catches a guard that lets a
non-zero status escape. The guard's unconditional final `exit 0` is what keeps a failing or
crashing binary from blocking the user's tool call or surfacing a "hook error" (NUDGE-03,
T-06-02); the suite must be shown able to see its loss.

**Pre-mutation gate:** `git diff --quiet -- .claude/hooks/pretooluse-nudge.sh` — exit 0 (clean).

**Mutation applied:** `perl -0pi -e 's/\nexit 0\n\z/\nexit 2\n/' .claude/hooks/pretooluse-nudge.sh`
(the `//go:embed` directive picks the change up when the test binary builds):

```diff
--- a/.claude/hooks/pretooluse-nudge.sh
+++ b/.claude/hooks/pretooluse-nudge.sh
@@ -24,4 +24,4 @@ if [ ! -f "$codegraph_bin" ] || [ ! -x "$codegraph_bin" ]; then
   exit 0
 fi
 "$codegraph_bin" hook pretooluse
-exit 0
+exit 2
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestPreToolUseGuard$' -v`, `=== RUN` lines dropped, exit code appended):

```
    claude_pretooluse_test.go:361: guard exit = 2, want 0 (stdout "STUB-OUT\n", stderr "")
    claude_pretooluse_test.go:361: guard exit = 2, want 0 (stdout "", stderr "")
    claude_pretooluse_test.go:361: guard exit = 2, want 0 (stdout "", stderr "/var/folders/.../T/TestPreToolUseGuardbinary_crashes3585332460/003/pretooluse-nudge.sh: line 26: 50763 Segmentation fault: 11  \"$codegraph_bin\" hook pretooluse\n")
    claude_pretooluse_test.go:361: guard exit = 2, want 0 (stdout "STUB-OUT\n", stderr "")
--- FAIL: TestPreToolUseGuard (0.42s)
    --- FAIL: TestPreToolUseGuard/indexed_binary_ok (0.05s)
    --- PASS: TestPreToolUseGuard/not_indexed (0.03s)
    --- PASS: TestPreToolUseGuard/codegraph_is_file (0.04s)
    --- PASS: TestPreToolUseGuard/binary_missing (0.04s)
    --- PASS: TestPreToolUseGuard/binary_not_executable (0.04s)
    --- FAIL: TestPreToolUseGuard/binary_exits_nonzero (0.06s)
    --- FAIL: TestPreToolUseGuard/binary_crashes (0.06s)
    --- FAIL: TestPreToolUseGuard/project_dir_unset_indexed (0.06s)
    --- PASS: TestPreToolUseGuard/project_dir_unset_not_indexed (0.04s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.499s
FAIL
exit=1
```

The four subtests that reach the tail go RED, including the named target
`--- FAIL: TestPreToolUseGuard/binary_exits_nonzero`; the five that exit early (un-indexed,
missing or non-executable binary) stay green, as they should — they never reach the final
line.

**Pre-revert gate:** `git diff --quiet -- .claude/hooks/pretooluse-nudge.sh` — exit 1 (only the
planted diff).

**Revert:** `git checkout -- .claude/hooks/pretooluse-nudge.sh`, then
`git diff --quiet -- .claude/hooks/pretooluse-nudge.sh` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestPreToolUseGuard$'`
→ `ok  	github.com/seanb4t/codegraph-go/internal/agents	0.467s`.

---

## Family (a2) — T-06-01: naive ExecPath quoting turns TestRenderPreToolGuard RED

**Test/guard:** `TestRenderPreToolGuard/quote_and_space_in_path`
(`internal/agents/claude_pretooluse_test.go`) — renders the real template for a stub binary at
`<tmp>/it's a dir/codegraph`, runs `sh -n` over the result, then executes it in an indexed
project and asserts the stub started.

**What are we testing, and why?** Whether the render test catches an ExecPath that is not
POSIX-quoted. `shellSingleQuote` is the only thing standing between an install path and shell
interpretation of its bytes in an executable script (T-06-01).

**Pre-mutation gate:** `git diff --quiet -- internal/agents/claude_pretooluse.go` — exit 0 (clean).

**Mutation applied:** a `perl -0pi` substitution on `shellSingleQuote`'s body, dropping the
escaping of embedded single quotes:

```diff
--- a/internal/agents/claude_pretooluse.go
+++ b/internal/agents/claude_pretooluse.go
@@ -67,7 +67,7 @@ func claudePreToolUseBlocks(loc Location) ([]any, []string, error) {
 // embedded single quote is emitted as close-quote, backslash-quote,
 // reopen-quote, so no byte of s is ever interpreted by the shell (T-06-01).
 func shellSingleQuote(s string) string {
-	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
+	return "'" + s + "'"
 }
 
 // renderPreToolGuard renders the embedded guard template for execPath,
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestRenderPreToolGuard$' -v`, `=== RUN` lines dropped, exit code appended):

```
    claude_pretooluse_test.go:417: sh -n /var/folders/.../T/TestRenderPreToolGuardquote_and_space_in_path782972962/003/pretooluse-nudge.sh: exit status 2
        /var/folders/.../T/TestRenderPreToolGuardquote_and_space_in_path782972962/003/pretooluse-nudge.sh: line 19: unexpected EOF while looking for matching `''
        /var/folders/.../T/TestRenderPreToolGuardquote_and_space_in_path782972962/003/pretooluse-nudge.sh: line 28: syntax error: unexpected end of file
--- FAIL: TestRenderPreToolGuard (0.01s)
    --- PASS: TestRenderPreToolGuard/plain_path (0.00s)
    --- FAIL: TestRenderPreToolGuard/quote_and_space_in_path (0.00s)
    --- PASS: TestRenderPreToolGuard/relative_path_rejected (0.00s)
    --- PASS: TestRenderPreToolGuard/empty_path_rejected (0.00s)
    --- PASS: TestRenderPreToolGuard/template_token_exactly_once (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.085s
FAIL
exit=1
```

Only the quote-bearing path goes RED (`--- FAIL: TestRenderPreToolGuard/quote_and_space_in_path`);
`plain_path` stays green because a path without a single quote is quoted correctly either way.

**Pre-revert gate:** `git diff --quiet -- internal/agents/claude_pretooluse.go` — exit 1 (only
the planted diff).

**Revert:** `git checkout -- internal/agents/claude_pretooluse.go`, then
`git diff --quiet -- internal/agents/claude_pretooluse.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1 -run 'TestRenderPreToolGuard$'`
→ `ok  	github.com/seanb4t/codegraph-go/internal/agents	0.129s`.
