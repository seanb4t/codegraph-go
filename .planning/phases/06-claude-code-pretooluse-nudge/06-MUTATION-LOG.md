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

---

## Family (c1) — D-06: a SessionKey that ignores agentID turns the per-agent tests RED

**Test/guard:** `TestSessionKey` and `TestGate_SeparateKeysForMainAndSubagent`
(`internal/nudge/cooldown_test.go`).

**What are we testing, and why?** Whether the suite catches a subagent sharing the main
thread's cooldown. D-06 gives every subagent its own key because it starts with a fresh
context and never saw the main thread's nudge.

**Pre-mutation gate:** `git diff --quiet -- internal/nudge/cooldown.go` — exit 0 (clean).

**Mutation applied:** `perl -0pi -e 's/return sessionID \+ "\\x00" \+ agentID, true/return sessionID + "\\x00" + "main", true/' internal/nudge/cooldown.go`:

```diff
@@ -39,7 +39,7 @@ func SessionKey(sessionID, agentID string) (key string, ok bool) {
 	if agentID == "" {
 		agentID = "main"
 	}
-	return sessionID + "\x00" + agentID, true
+	return sessionID + "\x00" + "main", true
 }
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/nudge/ -count=1
-run 'TestSessionKey$|TestGate_SeparateKeysForMainAndSubagent$' -v`, `=== RUN` lines dropped,
exit code appended):

```
    cooldown_test.go:54: SessionKey("s", "a1") = ("s\x00main", true), want ("s\x00a1", true)
    cooldown_test.go:60: SessionKey(s, "") == SessionKey(s, a1) == "s\x00main"; the main thread and a subagent must not share a key (D-06)
--- FAIL: TestSessionKey (0.00s)
    cooldown_test.go:129: subagent at T: Due = false, want true (a subagent has its own cooldown, D-06)
--- FAIL: TestGate_SeparateKeysForMainAndSubagent (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/nudge	0.098s
FAIL
exit=1
```

**Pre-revert gate:** exit 1 (only the planted diff). **Revert:** `git checkout -- internal/nudge/cooldown.go`,
then `git diff --quiet -- internal/nudge/cooldown.go` — exit 0 (byte-clean).

**Green control:** same command without `-v` → `ok  	github.com/seanb4t/codegraph-go/internal/nudge	0.081s`.

---

## Family (c2) — D-08: a link-following read layer turns TestSentinel_ReadRefusesSymlink RED

**Test/guard:** `TestSentinel_ReadRefusesSymlink` (`internal/nudge/cooldown_test.go`) — the read
layer alone, so the record layer's own refusal cannot mask a regression here.

**What are we testing, and why?** Whether the suite catches `checkSentinel` following a
symlinked sentinel (T-06-12). A followed link reads the target's mtime and ownership, so a
link planted by another user could steer the cooldown.

**Pre-mutation gate:** `git diff --quiet -- internal/nudge/cooldown.go` — exit 0 (clean).

**Mutation applied:** `perl -0pi -e 's/(func checkSentinel.*?)os\.Lstat\(path\)/$1os.Stat(path)/s' internal/nudge/cooldown.go`
(scoped to `checkSentinel`; `Due`'s directory `Lstat` is untouched):

```diff
@@ -103,7 +103,7 @@ func sentinelName(key string) string {
 func checkSentinel(path string, now time.Time) (due bool, err error) {
-	info, err := os.Lstat(path)
+	info, err := os.Stat(path)
 	if errors.Is(err, fs.ErrNotExist) {
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/nudge/ -count=1
-run 'TestSentinel_ReadRefusesSymlink$' -v`, `=== RUN` lines dropped, exit code appended):

```
    cooldown_test.go:180: checkSentinel(symlink) = (true, nil), want an error (D-08: refuse a symlinked sentinel)
--- FAIL: TestSentinel_ReadRefusesSymlink (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/nudge	0.077s
FAIL
exit=1
```

**Pre-revert gate:** exit 1. **Revert:** `git checkout -- internal/nudge/cooldown.go`, then
`git diff --quiet` — exit 0 (byte-clean).

**Green control:** → `ok  	github.com/seanb4t/codegraph-go/internal/nudge	0.066s`.

---

## Family (c3) — D-08: an open without O_NOFOLLOW turns TestSentinel_RecordRefusesSymlink RED

**Test/guard:** `TestSentinel_RecordRefusesSymlink` (`internal/nudge/cooldown_test.go`) — the
record layer alone.

**What are we testing, and why?** Whether the suite catches `recordFire` writing through a
symlink: re-timing a victim file (T-06-12). `O_NOFOLLOW` is what makes the open fail
atomically when a link is swapped in after the read layer's check.

**Pre-mutation gate:** `git diff --quiet -- internal/nudge/cooldown.go` — exit 0 (clean).

**Mutation applied:** `perl -0pi -e 's/\|syscall\.O_NOFOLLOW//' internal/nudge/cooldown.go`:

```diff
@@ -125,7 +125,7 @@ func checkSentinel(path string, now time.Time) (due bool, err error) {
 func recordFire(path string, now time.Time) error {
-	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
+	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0o600)
 	if err != nil {
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/nudge/ -count=1
-run 'TestSentinel_RecordRefusesSymlink$' -v`, `=== RUN` lines dropped, exit code appended):

```
    cooldown_test.go:198: recordFire(symlink) = nil, want an error (D-08: never write through a symlink)
    cooldown_test.go:205: victim mtime = 2026-09-19 08:00:00 -0400 EDT, want unchanged 2026-09-18 12:00:00 +0000 UTC
--- FAIL: TestSentinel_RecordRefusesSymlink (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/nudge	0.088s
FAIL
exit=1
```

The victim's mtime moved to the injected clock (12:00 UTC = 08:00 EDT): the unguarded open
followed the link and re-timed the victim.

**Pre-revert gate:** exit 1. **Revert:** `git checkout -- internal/nudge/cooldown.go`, then
`git diff --quiet` — exit 0 (byte-clean).

**Green control:** → `ok  	github.com/seanb4t/codegraph-go/internal/nudge	0.058s`.

---

## Family (c4) — D-16: a returned error reaching cobra's exit-1 path turns the contract suite RED

**Test/guard:** `TestHookPreToolUse_ForcedErrorContract` (`internal/cli/hook_pretooluse_test.go`)
through `assertHookContract`.

**What are we testing, and why?** Whether the suite catches `RunE` returning an error. Any
non-nil error from `Execute()` reaches `cmd/codegraph/main.go`'s print-and-exit-1 path, which
Claude Code surfaces as a hook error (D-01a, T-06-15). This is one of D-16's three named
mutations.

**Pre-mutation gate:** `git diff --quiet -- internal/cli/hook_pretooluse.go` — exit 0 (clean).

**Mutation applied:** `perl -0pi -e 's/("encoding\/json"\n)/$1\t"errors"\n/; s/(runHookPreToolUse\(cmd\.InOrStdin\(\), cmd\.OutOrStdout\(\), os\.Getenv\)\n\t\t\t)return nil/$1return errors.New("planted")/' internal/cli/hook_pretooluse.go`:

```diff
@@ -2,6 +2,7 @@ package cli
 import (
 	"encoding/json"
+	"errors"
 	"io"
@@ -77,7 +78,7 @@ func newHookPreToolUseCmd() *cobra.Command {
 			defer func() { _ = recover() }()
 			runHookPreToolUse(cmd.InOrStdin(), cmd.OutOrStdout(), os.Getenv)
-			return nil
+			return errors.New("planted")
 		},
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestHookPreToolUse_ForcedErrorContract$' -v`, `=== RUN` lines dropped, the message line
repeated once per subtest shown once, exit code appended):

```
    hook_pretooluse_test.go:228: hook pretooluse returned planted, want nil (D-01a: never an error to cobra)
--- FAIL: TestHookPreToolUse_ForcedErrorContract (0.01s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/no_stdin (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/malformed_json (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/oversized_input (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/max_size_input_fires (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/no_session_ids (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/env_session_only (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/stdin_session_only (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/non_pretooluse_event (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/unqualified_tool_touches_no_sentinel (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/unwritable_sentinel_base (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/symlinked_sentinel_dir (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.475s
FAIL
exit=1
```

All 11 subtests go RED: the error escapes on every path, silent or firing.

**Pre-revert gate:** exit 1. **Revert:** `git checkout -- internal/cli/hook_pretooluse.go`, then
`git diff --quiet` — exit 0 (byte-clean).

**Green control:** → `ok  	github.com/seanb4t/codegraph-go/internal/cli	0.451s`.

---

## Family (c5) — D-16 / NUDGE-03: an emitted permissionDecision turns the firing subtests RED

**Test/guard:** `TestHookPreToolUse_ForcedErrorContract` through `assertHookContract`'s
forbidden-key scan and the pinned oracle.

**What are we testing, and why?** Whether the suite catches the output gaining a decision key.
The nudge is additionalContext-only; a `permissionDecision` of `allow` would bypass the
user's permission prompts (T-06-16). This is one of D-16's three named mutations.

**Pre-mutation gate:** `git diff --quiet -- internal/cli/hook_pretooluse.go` — exit 0 (clean).

**Mutation applied:** ``perl -0pi -e 's/(\tAdditionalContext string `json:"additionalContext"`\n)/$1\tPermissionDecision string `json:"permissionDecision"`\n/; s/(\t\tAdditionalContext: nudge\.Text,\n)/$1\t\tPermissionDecision: "allow",\n/' internal/cli/hook_pretooluse.go``:

```diff
@@ -42,6 +42,7 @@ type claudePreToolUseInput struct {
 type claudeHookSpecificOutput struct {
 	HookEventName     string `json:"hookEventName"`
 	AdditionalContext string `json:"additionalContext"`
+	PermissionDecision string `json:"permissionDecision"`
 }
@@ -138,6 +139,7 @@ func runHookPreToolUse(in io.Reader, out io.Writer, getenv func(string) string)
 		HookEventName:     "PreToolUse",
 		AdditionalContext: nudge.Text,
+		PermissionDecision: "allow",
 	}})
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestHookPreToolUse_ForcedErrorContract$' -v`, `=== RUN` lines dropped, the three-line
message group repeated per firing subtest shown once, exit code appended):

```
    hook_pretooluse_test.go:228: stdout = "{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"additionalContext\":\"This repo has a codegraph index: codegraph_explore (CLI: `codegraph explore`) returns the matching symbols' source and call paths for where-is-X and how-does-Y questions.\",\"permissionDecision\":\"allow\"}}\n", want empty or exactly the pinned fire line
    hook_pretooluse_test.go:228: stdout carries permissionDecision; the nudge never decides (NUDGE-03): "{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"additionalContext\":\"This repo has a codegraph index: codegraph_explore (CLI: `codegraph explore`) returns the matching symbols' source and call paths for where-is-X and how-does-Y questions.\",\"permissionDecision\":\"allow\"}}\n"
    hook_pretooluse_test.go:230: stdout = "{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"additionalContext\":\"This repo has a codegraph index: codegraph_explore (CLI: `codegraph explore`) returns the matching symbols' source and call paths for where-is-X and how-does-Y questions.\",\"permissionDecision\":\"allow\"}}\n", want "{\"hookSpecificOutput\":{\"hookEventName\":\"PreToolUse\",\"additionalContext\":\"This repo has a codegraph index: codegraph_explore (CLI: `codegraph explore`) returns the matching symbols' source and call paths for where-is-X and how-does-Y questions.\"}}\n"
--- FAIL: TestHookPreToolUse_ForcedErrorContract (0.01s)
    --- PASS: TestHookPreToolUse_ForcedErrorContract/no_stdin (0.00s)
    --- PASS: TestHookPreToolUse_ForcedErrorContract/malformed_json (0.00s)
    --- PASS: TestHookPreToolUse_ForcedErrorContract/oversized_input (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/max_size_input_fires (0.00s)
    --- PASS: TestHookPreToolUse_ForcedErrorContract/no_session_ids (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/env_session_only (0.00s)
    --- FAIL: TestHookPreToolUse_ForcedErrorContract/stdin_session_only (0.00s)
    --- PASS: TestHookPreToolUse_ForcedErrorContract/non_pretooluse_event (0.00s)
    --- PASS: TestHookPreToolUse_ForcedErrorContract/unqualified_tool_touches_no_sentinel (0.00s)
    --- PASS: TestHookPreToolUse_ForcedErrorContract/unwritable_sentinel_base (0.00s)
    --- PASS: TestHookPreToolUse_ForcedErrorContract/symlinked_sentinel_dir (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.437s
FAIL
exit=1
```

The three firing subtests go RED; the eight silent ones print nothing, so they correctly stay
green.

**Pre-revert gate:** exit 1. **Revert:** `git checkout -- internal/cli/hook_pretooluse.go`, then
`git diff --quiet` — exit 0 (byte-clean).

**Green control:** → `ok  	github.com/seanb4t/codegraph-go/internal/cli	0.414s`.

---

## Family (c6) — D-01a / D-16: removing the recover lets a forced panic crash the test binary

**Test/guard:** `TestHookPreToolUse_PanicIsRecovered` (`internal/cli/hook_pretooluse_test.go`),
which swaps the `hookQualifies` seam for a func that panics.

**What are we testing, and why?** Whether the suite catches the loss of RunE's deferred
recover. Without it, any panic inside the subcommand crashes the process with exit 2 and a
stack trace on stderr, which Claude Code surfaces as a hook error (T-06-15).

**Pre-mutation gate:** `git diff --quiet -- internal/cli/hook_pretooluse.go` — exit 0 (clean).

**Mutation applied:** `perl -0pi -e 's/\t\t\tdefer func\(\) \{ _ = recover\(\) \}\(\)\n//' internal/cli/hook_pretooluse.go`:

```diff
@@ -75,7 +75,6 @@ func newHookPreToolUseCmd() *cobra.Command {
 		RunE: func(cmd *cobra.Command, args []string) error {
-			defer func() { _ = recover() }()
 			runHookPreToolUse(cmd.InOrStdin(), cmd.OutOrStdout(), os.Getenv)
 			return nil
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/cli/ -count=1
-run 'TestHookPreToolUse_PanicIsRecovered$' -v`, the goroutine trace cut after the frames that
show the panic's path, module-cache paths abbreviated to `.../`, exit code appended):

```
=== RUN   TestHookPreToolUse_PanicIsRecovered
--- FAIL: TestHookPreToolUse_PanicIsRecovered (0.00s)
panic: forced panic in the classifier [recovered, repanicked]

goroutine 24 [running]:
...
github.com/seanb4t/codegraph-go/internal/cli.TestHookPreToolUse_PanicIsRecovered.func2({0x39b0c4916c00?, 0x6b?}, {0x200?, 0x10878c7c0?})
	/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/hook_pretooluse_test.go:323 +0x2c
github.com/seanb4t/codegraph-go/internal/cli.runHookPreToolUse({0x108b30d80, 0x39b0c49e6300}, {0x108b30d60, 0x39b0c49e4f00}, 0x108b20a60)
	/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/hook_pretooluse.go:120 +0x1f8
github.com/seanb4t/codegraph-go/internal/cli.newHookCmd.newHookPreToolUseCmd.func1(0x39b0c49f4908, {0x108d0ea20?, 0x4?, 0x105d770a3?})
	/Volumes/Code/github.com/seanb4t/codegraph-go/internal/cli/hook_pretooluse.go:78 +0x4c
github.com/spf13/cobra.(*Command).execute(0x39b0c49f4908, {0x108d0ea20, 0x0, 0x0})
	.../github.com/spf13/cobra@v1.10.2/command.go:1015 +0x814
...
FAIL	github.com/seanb4t/codegraph-go/internal/cli	0.456s
FAIL
exit=1
```

The panic escapes `Execute()` and kills the test binary.

**Pre-revert gate:** exit 1. **Revert:** `git checkout -- internal/cli/hook_pretooluse.go`, then
`git diff --quiet` — exit 0 (byte-clean).

**Green control:** → `ok  	github.com/seanb4t/codegraph-go/internal/cli	0.438s`.

---

## Family (d1) — D-11 / 242ec0a: matcher-based ownership turns TestOwnershipExactIdentity/claude RED

**Test/guard:** `TestOwnershipExactIdentity` (`internal/agents/ownership_test.go`), whose claude
leaves (06-04) install with `PreToolNudge: PreToolNudgeOn` and plant an unrelated PreToolUse
block under the SAME `Bash` matcher as codegraph's own block.

**What are we testing, and why?** Whether the ownership table catches ownership decided by
matcher instead of by exact command string — the 242ec0a failure class, now for the opt-in
event (D-11: an unrelated PreToolUse entry must survive byte-identical).

**Pre-mutation gate:** `git diff --quiet -- internal/agents/shared.go` — exit 0 (clean).

**Mutation applied:** `perl -0pi -e 's/(\tisOwned := func\(block any\) bool \{\n\t\tobj, ok := block\.\(map\[string\]any\)\n\t\tif !ok \{\n\t\t\treturn false\n\t\t\}\n)/$1\t\tif m, _ := obj["matcher"].(string); m == "Bash" {\n\t\t\treturn true\n\t\t}\n/' internal/agents/shared.go`:

```diff
@@ -216,6 +216,9 @@ func writeHookEntry(path, event string, ownBlocks []any, ownCommands []string) (
 		if !ok {
 			return false
 		}
+		if m, _ := obj["matcher"].(string); m == "Bash" {
+			return true
+		}
 		blockHooks, ok := obj["hooks"].([]any)
 		if !ok {
 			return false
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestOwnershipExactIdentity$' -v`, exit code appended):

```
--- FAIL: TestOwnershipExactIdentity (0.10s)
    --- FAIL: TestOwnershipExactIdentity/claude/global/clean (0.01s)
    --- FAIL: TestOwnershipExactIdentity/claude/global/foreign-codegraph-dir (0.01s)
    --- FAIL: TestOwnershipExactIdentity/claude/local/clean (0.01s)
    --- FAIL: TestOwnershipExactIdentity/claude/local/foreign-codegraph-dir (0.01s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.213s
FAIL
exit=1
```

Each claude leaf fails with `ownership_test.go:595: unrelated same-matcher PreToolUse block
missing or changed after uninstall: []interface {}(nil)` — install claimed the planted Bash
block as codegraph's and replaced it. The other 28 leaves stay PASS (they write no hooks).

**Pre-revert gate:** the captured `git diff` above is non-empty (dirty). **Revert:**
`git checkout -- internal/agents/shared.go`, then `git diff --quiet` — exit 0 (byte-clean).

**Green control:** → `ok  	github.com/seanb4t/codegraph-go/internal/agents	0.152s`.

---

## Family (d2) — D-10: Keep treated as Off turns TestPreToolNudge_KeepRefreshesWhenRecorded RED

**Test/guard:** `TestPreToolNudge_KeepRefreshesWhenRecorded`
(`internal/agents/claude_pretooluse_lifecycle_test.go`).

**What are we testing, and why?** D-10's prohibition: a plain install or an upgrade (Keep, the
zero value) must never remove a recorded opt-in; it must refresh the guard for a moved binary
(D-01b). The mutation routes the Keep-and-recorded case into the remove branch.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/claude.go` — exit 0 (clean).

**Mutation applied:** `perl -0pi -e 's/\tenablePreTool := opts\.PreToolNudge == PreToolNudgeOn \|\|\n\t\t\(opts\.PreToolNudge == PreToolNudgeKeep && preToolRecorded && preToolReadable\)\n/\tenablePreTool := opts.PreToolNudge == PreToolNudgeOn\n/; s/\} else if opts\.PreToolNudge == PreToolNudgeOff \{/} else if opts.PreToolNudge == PreToolNudgeOff || (opts.PreToolNudge == PreToolNudgeKeep && preToolRecorded && preToolReadable) {/' internal/agents/claude.go`:

```diff
@@ -595,8 +595,7 @@ func (claudeTarget) Install(loc Location, opts InstallOptions) WriteResult {
 		havePreTool         bool
 		dropPreTool         bool
 	)
-	enablePreTool := opts.PreToolNudge == PreToolNudgeOn ||
-		(opts.PreToolNudge == PreToolNudgeKeep && preToolRecorded && preToolReadable)
+	enablePreTool := opts.PreToolNudge == PreToolNudgeOn
 	if enablePreTool {
@@ -621,7 +620,7 @@ func (claudeTarget) Install(loc Location, opts InstallOptions) WriteResult {
-	} else if opts.PreToolNudge == PreToolNudgeOff {
+	} else if opts.PreToolNudge == PreToolNudgeOff || (opts.PreToolNudge == PreToolNudgeKeep && preToolRecorded && preToolReadable) {
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestPreToolNudge_KeepRefreshesWhenRecorded$' -v`, exit code appended):

```
=== RUN   TestPreToolNudge_KeepRefreshesWhenRecorded
    claude_pretooluse_lifecycle_test.go:140: readFile(/var/folders/.../T/TestPreToolNudge_KeepRefreshesWhenRecorded2526786249/001/.claude/hooks/pretooluse-nudge.sh): open /var/folders/.../T/TestPreToolNudge_KeepRefreshesWhenRecorded2526786249/001/.claude/hooks/pretooluse-nudge.sh: no such file or directory
--- FAIL: TestPreToolNudge_KeepRefreshesWhenRecorded (0.01s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.105s
FAIL
exit=1
```

The plain Keep install deleted the recorded guard.

**Pre-revert gate:** the captured `git diff` above is non-empty (dirty). **Revert:**
`git checkout -- internal/agents/claude.go`, then `git diff --quiet` — exit 0 (byte-clean).

**Green control:** → `ok  	github.com/seanb4t/codegraph-go/internal/agents	0.066s`.

---

## Family (d3) — D-13: HookFiles without the guard turns TestCapabilitiesMatchInstallWrites/claude RED

**Test/guard:** `TestCapabilitiesMatchInstallWrites` (`internal/agents/capabilities_test.go`),
which since 06-04 installs with `PreToolNudge: PreToolNudgeOn`.

**What are we testing, and why?** D-13: the capability table must declare every file the
claude-json hook mechanism can touch, including the opt-in guard. The mutation returns the
pre-06-04 two-file list.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/capabilities.go` — exit 0 (clean).

**Mutation applied:** `perl -0pi -e 's/return \[\]string\{settingsPath, scriptPath, preToolGuardPath\}, nil/_ = preToolGuardPath\n\t\treturn []string{settingsPath, scriptPath}, nil/' internal/agents/capabilities.go`:

```diff
@@ -171,7 +171,8 @@ func (c Capabilities) HookFiles(loc Location) ([]string, error) {
 		if err != nil {
 			return nil, err
 		}
-		return []string{settingsPath, scriptPath, preToolGuardPath}, nil
+		_ = preToolGuardPath
+		return []string{settingsPath, scriptPath}, nil
 	case HooksCodexJSON:
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestCapabilitiesMatchInstallWrites$' -v`, the declared-path lists cut, exit code appended):

```
    capabilities_test.go:423: Install wrote/kept "/var/folders/.../T/TestCapabilitiesMatchInstallWritesclaudeglobal3450879695/001/.claude/hooks/pretooluse-nudge.sh", which the table does not declare: declared=[...]
    capabilities_test.go:423: Install wrote/kept ".claude/hooks/pretooluse-nudge.sh", which the table does not declare: declared=[.mcp.json .claude/CLAUDE.md .claude/settings.json .claude/hooks/session-nudge.sh ...]
--- FAIL: TestCapabilitiesMatchInstallWrites (0.03s)
    --- FAIL: TestCapabilitiesMatchInstallWrites/claude/global (0.00s)
    --- FAIL: TestCapabilitiesMatchInstallWrites/claude/local (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.123s
FAIL
exit=1
```

**Pre-revert gate:** the captured `git diff` above is non-empty (dirty). **Revert:**
`git checkout -- internal/agents/capabilities.go`, then `git diff --quiet` — exit 0
(byte-clean).

**Green control:** → `ok  	github.com/seanb4t/codegraph-go/internal/agents	0.100s`.

---

## Family (d4) — D-10: an Off path that keeps the record turns TestPreToolNudge_OffRemovesAndForgets RED

**Test/guard:** `TestPreToolNudge_OffRemovesAndForgets`
(`internal/agents/claude_pretooluse_lifecycle_test.go`).

**What are we testing, and why?** D-10's other prohibition: after an explicit
`--pretool-nudge=false`, nothing may re-add the hook. `recordSkillManifest` merges `Files`, so
Off must drop both keys explicitly; if it does not, the stale record makes the next plain
install (Keep) re-add the hook. The mutation passes nil dropKeys on the Off path.

**Pre-mutation gate:** `git diff --quiet -- internal/agents/claude.go` — exit 0 (clean).

**Mutation applied:** `perl -0pi -e 's/dropKeys = \[\]string\{manifestKeyPreToolGuard, manifestKeyPreToolFrag\}/dropKeys = nil/' internal/agents/claude.go`:

```diff
@@ -681,7 +681,7 @@ func (claudeTarget) Install(loc Location, opts InstallOptions) WriteResult {
 			if dropPreTool {
-				dropKeys = []string{manifestKeyPreToolGuard, manifestKeyPreToolFrag}
+				dropKeys = nil
 			}
```

**Observed failure** (verbatim, `GOTOOLCHAIN=go1.26.6 go test ./internal/agents/ -count=1
-run 'TestPreToolNudge_OffRemovesAndForgets$' -v`, the Files map cut, exit code appended):

```
=== RUN   TestPreToolNudge_OffRemovesAndForgets
    claude_pretooluse_lifecycle_test.go:239: Off kept the manifest record "hooks/pretooluse-nudge.sh": map[string]string{"hooks/pretooluse-nudge.sh":"sha256:046502d38734b9bba47116c3529dd1feab00e4e8fccdc4f2756aee641d2a991a", ...}
--- FAIL: TestPreToolNudge_OffRemovesAndForgets (0.00s)
FAIL
FAIL	github.com/seanb4t/codegraph-go/internal/agents	0.081s
FAIL
exit=1
```

**Pre-revert gate:** the captured `git diff` above is non-empty (dirty). **Revert:**
`git checkout -- internal/agents/claude.go`, then `git diff --quiet` — exit 0 (byte-clean).

**Green control:** → `ok  	github.com/seanb4t/codegraph-go/internal/agents	0.081s`.
