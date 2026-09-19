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

---

## Family (b1) — D-02: substring matching turns the false-positive corpus RED

**Test/guard:** `TestQualifiesCorpora` (`internal/nudge/classify_test.go`) over
`internal/nudge/testdata/false-positives.json` — every D-15 row as its own subtest.

**What are we testing, and why?** Whether the corpora catch a classifier that fires when a
search word merely appears anywhere in the command. D-02's first-word rule is what keeps pipe
tails (`git log | grep fix`), which Claude's `if` pre-filter also matches, from firing the
nudge (T-06-09).

**Pre-mutation gate:** `git diff --quiet -- internal/nudge/classify.go` — exit 0 (clean).

**Mutation applied:** a `perl -0pi` substitution replacing the first-field membership test
with a `strings.Contains` loop:

```diff
--- a/internal/nudge/classify.go
+++ b/internal/nudge/classify.go
@@ -77,5 +77,10 @@ func shellQualifies(command string) bool {
 	if i == len(fields) {
 		return false
 	}
-	return searchCommands[fields[i]]
+	for w := range searchCommands {
+		if strings.Contains(command, w) {
+			return true
+		}
+	}
+	return false
 }
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/nudge/ -count=1
-run 'TestQualifiesCorpora$' -v`, `=== RUN` lines dropped, exit code appended):

```
    classify_test.go:56: Qualifies("shell", "git grep -n \"func Parse\"") = true, want false (note: "accepted miss — first word is git; D-02 checks only the first word and Claude's Bash(grep *) rule never matches git grep")
    classify_test.go:56: Qualifies("shell", "cd internal && grep -rn Alpha .") = true, want false (note: "accepted miss — compound command, first word cd; D-02 parse-doubt rule")
    classify_test.go:56: Qualifies("shell", "git log --oneline | grep fix") = true, want false (note: "")
    classify_test.go:56: Qualifies("shell", "ps aux | grep codegraph") = true, want false (note: "")
    classify_test.go:56: Qualifies("shell", "cat build.log | grep -i error") = true, want false (note: "")
    classify_test.go:56: Qualifies("shell", "kubectl get pods -A | grep Running") = true, want false (note: "")
    classify_test.go:56: Qualifies("shell", "sudo find / -name core") = true, want false (note: "")
    classify_test.go:56: Qualifies("shell", "\"grep\" -rn x .") = true, want false (note: "")
    classify_test.go:56: Qualifies("shell", "$(which grep) -rn x .") = true, want false (note: "")
--- FAIL: TestQualifiesCorpora (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/0 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/1 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/2 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/3 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/4 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/5 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/6 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/7 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/8 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/9 (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/10 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/11 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/12 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/13 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/14 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/15 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/16 (0.00s)
    --- PASS: TestQualifiesCorpora/true-positives/17 (0.00s)
    --- FAIL: TestQualifiesCorpora/false-positives/0 (0.00s)
    --- FAIL: TestQualifiesCorpora/false-positives/1 (0.00s)
    --- FAIL: TestQualifiesCorpora/false-positives/2 (0.00s)
    --- FAIL: TestQualifiesCorpora/false-positives/3 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/4 (0.00s)
    --- FAIL: TestQualifiesCorpora/false-positives/5 (0.00s)
    --- FAIL: TestQualifiesCorpora/false-positives/6 (0.00s)
    --- FAIL: TestQualifiesCorpora/false-positives/7 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/8 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/9 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/10 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/11 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/12 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/13 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/14 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/15 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/16 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/17 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/18 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/19 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/20 (0.00s)
    --- PASS: TestQualifiesCorpora/false-positives/21 (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/nudge	0.071s
FAIL
exit=1
```

The named target goes RED: `--- FAIL: TestQualifiesCorpora/false-positives/0` is the
`git log --oneline | grep fix` row, joined by the other pipe-tail, `sudo find`, quoted and
`$(which grep)` rows. Two true-positive rows go RED as well — the accepted misses
(`git grep`, `cd … && grep`) whose pinned `want` is false; the mutation flips them too.
The `FOO="a b" grep` row stays green because the assignment parse-doubt check runs before the
mutated line.

**Pre-revert gate:** `git diff --quiet -- internal/nudge/classify.go` — exit 1 (only the
planted diff).

**Revert:** `git checkout -- internal/nudge/classify.go`, then
`git diff --quiet -- internal/nudge/classify.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/nudge/ -count=1 -run 'TestQualifiesCorpora$'`
→ `ok  	github.com/seanb4t/codegraph-go/internal/nudge` (green).

---

## Family (b2) — D-02: dropping the assignment skip turns the `LC_ALL=C grep` row RED

**Test/guard:** `TestQualifiesCorpora` over `internal/nudge/testdata/true-positives.json`.

**What are we testing, and why?** Whether the corpora see the loss of the leading
`NAME=value` skip that D-02 requires; without it `LC_ALL=C grep …` is judged by its
assignment word and a genuine where-is-X search goes silent.

**Pre-mutation gate:** `git diff --quiet -- internal/nudge/classify.go` — exit 0 (clean).

**Mutation applied:** a `perl -0pi` deletion of the assignment-skip loop, so the first field
is always the one tested:

```diff
--- a/internal/nudge/classify.go
+++ b/internal/nudge/classify.go
@@ -69,11 +69,6 @@ func Qualifies(tool Tool, input string) bool {
 func shellQualifies(command string) bool {
 	fields := strings.Fields(command)
 	i := 0
-	for ; i < len(fields) && assignmentRe.MatchString(fields[i]); i++ {
-		if strings.ContainsAny(fields[i], "'\"`$\\") {
-			return false
-		}
-	}
 	if i == len(fields) {
 		return false
 	}
```

**Observed failure** (verbatim, same command, `=== RUN` and `--- PASS` lines dropped, exit
code appended):

```
    classify_test.go:56: Qualifies("shell", "LC_ALL=C grep -rn handleRequest internal") = false, want true (note: "")
--- FAIL: TestQualifiesCorpora (0.00s)
    --- FAIL: TestQualifiesCorpora/true-positives/5 (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/nudge	0.074s
FAIL
exit=1
```

Exactly one row goes RED, the named target: `--- FAIL: TestQualifiesCorpora/true-positives/5`
(`LC_ALL=C grep -rn handleRequest internal`). The false-positive rows that carry assignments
(`FOO="a b" grep …`) stay green because a quoted first field is not a search word either way.

**Pre-revert gate:** `git diff --quiet -- internal/nudge/classify.go` — exit 1 (only the
planted diff).

**Revert:** `git checkout -- internal/nudge/classify.go`, then
`git diff --quiet -- internal/nudge/classify.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/nudge/ -count=1 -run 'TestQualifiesCorpora$'`
→ `ok  	github.com/seanb4t/codegraph-go/internal/nudge` (green).

---

## Family (b3) — D-14: a renamed tool in `nudge.Text` turns TestPreToolUseNudgeTextNamesOnlyRealTools RED

**Test/guard:** `TestPreToolUseNudgeTextNamesOnlyRealTools`
(`internal/mcp/skill_claims_drift_test.go`) — every `codegraph_<name>` token in the constant
`nudge.Text` must be in `allToolNames()`.

**What are we testing, and why?** Whether the drift guard catches the nudge naming a tool that
does not exist — the wire-claim-drift bug class GUARD-01 exists for, now over a Go constant
rather than a file (RESEARCH Pitfall 4; T-06-10).

**Pre-mutation gate:** `git diff --quiet -- internal/nudge/text.go` — exit 0 (clean).

**Mutation applied:** `perl -pi -e 's/codegraph_explore \(/codegraph_explorer (/' internal/nudge/text.go`:

```diff
--- a/internal/nudge/text.go
+++ b/internal/nudge/text.go
@@ -5,4 +5,4 @@ package nudge
 // `codegraph explore` CLI fallback. Its bytes are pinned — by hand-typed
 // oracles in the adapter tests here, and by the text drift guards 06-02
 // adds — so any edit is a deliberate, test-visible change.
-const Text = "This repo has a codegraph index: codegraph_explore (CLI: `codegraph explore`) returns the matching symbols' source and call paths for where-is-X and how-does-Y questions."
+const Text = "This repo has a codegraph index: codegraph_explorer (CLI: `codegraph explore`) returns the matching symbols' source and call paths for where-is-X and how-does-Y questions."
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/mcp/ -count=1
-run 'TestPreToolUseNudgeTextNamesOnlyRealTools$' -v`, `=== RUN` lines dropped, exit code
appended):

```
    skill_claims_drift_test.go:885: nudge.Text names codegraph_explorer, which is not a member of allToolNames() — a renamed or removed tool left behind in the nudge text
--- FAIL: TestPreToolUseNudgeTextNamesOnlyRealTools (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/mcp	0.333s
FAIL
exit=1
```

**Pre-revert gate:** `git diff --quiet -- internal/nudge/text.go` — exit 1 (only the planted
diff).

**Revert:** `git checkout -- internal/nudge/text.go`, then
`git diff --quiet -- internal/nudge/text.go` — exit 0 (byte-clean).

**Green control:** `GOTOOLCHAIN=go1.26.6 go test ./internal/mcp/ -count=1 -run 'TestPreToolUseNudgeTextNamesOnlyRealTools$'`
→ `ok  	github.com/seanb4t/codegraph-go/internal/mcp` (green).
