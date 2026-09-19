# Phase 6 Plan 6: Live-Session Evidence — Does Claude Code Match, Deliver and Space the PreToolUse Nudge?

**Status:** Task 1 (scaffold) done 2026-09-19 at HEAD `d3201ba9`. Sessions A and B (Tasks 2–3)
have not run yet; every verdict line below is still open.

## Method

Per D-18 (06-CONTEXT.md), with the pass bar locked there before any session runs, the live
evidence comes from the **orchestrator**, never from an executor subagent. A backgrounded
subagent cannot reliably drive another pane's TTY (the 05-06 precedent, D-09). The orchestrator
starts one fresh Claude Code session per protocol in a sibling Herdr pane:

```bash
herdr agent start <name> --kind claude --pane <id> -- --debug-file /tmp/06-live/debug/<name>.log
herdr agent prompt <name> "<text>" --wait
herdr agent read <name> --source recent-unwrapped > /tmp/06-live/transcripts/<name>.txt
```

Each session's JSONL transcript (`~/.claude/projects/<cwd-slug>/<session-id>.jsonl`, with any
subagent sidechain alongside) is copied to `/tmp/06-live/transcripts/` before it is searched.
Excerpts pasted here are ≤ 40 verbatim lines per criterion.

**Fire counting.** A fire is one occurrence of the pinned nudge text's distinctive substring

```
returns the matching symbols' source and call paths
```

in the session JSONL(s) and the debug log. Each fire is attributed to a key by where it appears
(main thread before `/clear`, subagent sidechain, main thread after `/clear`) and is recorded
with its timestamp in the Fire log table. Same-key gaps are computed from those timestamps.

**Negative-space rule (STATE standing rule, research pitfall 12).** A transcript grep is a
claim about the grep. Every absence claim (C2's no-fire window, C6's zero fires, C7's zero hook
errors) is backed by a search first shown to FIND the thing when present: the pinned substring
must first be found at session A's prompt-1 fire, and the C7 search must first find a known
marker (e.g. the guard's command string in the debug log) before a zero is recorded.

**Attribution (T-06-29).** The maintainer's global hooks also run in these sessions and may
deny or rewrite commands. A prompt, deny or block counts against C7 only when the debug log
attributes it to the `pretooluse-nudge.sh` command.

## Pre-flight

Recorded 2026-09-19 (read-only; nothing under `$HOME` written):

```
$ claude --version
2.1.278 (Claude Code)

$ shasum -a 256 ~/.claude/settings.json
be3ad316a7c4d5d921569f0398839638824fc4254d660378075e65ff69a339ef  /Users/sean/.claude/settings.json

$ test ! -e ~/.claude/hooks/pretooluse-nudge.sh; echo "exit: $?"
exit: 0

$ echo "$TMPDIR"
/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/

$ id -u
501
```

The guard's sentinel directory for these sessions is therefore
`/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/codegraph-nudge-501/` (64-hex names, zero
bytes). The `settings.json` checksum above is the value Task 3 re-checks.

## Hooks reference (NUDGE-03)

Fetched 2026-09-19 with `curl -fsSL https://code.claude.com/docs/en/hooks.md` (markdown form of
https://code.claude.com/docs/en/hooks; 3848 lines, sha256
`e0a14dcffd8299c22401f2ba18e577470f4ae25cb3dd6609a68160f1f2fbb390`, saved as
`/tmp/06-live/hooks-reference.md`). Quoted verbatim; the local `claude --version` the sessions use
is `2.1.278 (Claude Code)`.

**PreToolUse `hookSpecificOutput.additionalContext`:**

> | `additionalContext`        | String added to Claude's context alongside the tool result. Ignored when `permissionDecision` is `"defer"`. See [Add context for Claude](#add-context-for-claude) |

> The `additionalContext` field passes a string from your hook into Claude's context window. Claude Code wraps the string in a system reminder and inserts it into the conversation at the point where the hook fired. Claude reads the reminder on the next model request, but it doesn't appear as a chat message in the interface.

> Return `additionalContext` inside `hookSpecificOutput` alongside the event name:

> * [PreToolUse](#pretooluse), [PostToolUse](#posttooluse), [PostToolUseFailure](#posttoolusefailure), and [PostToolBatch](#posttoolbatch): next to the tool result

> When several hooks return `additionalContext` for the same event, Claude receives all of the values.

> A hook's `additionalContext`, `systemMessage`, and `initialUserMessage` strings, and its plain stdout, are capped at 10,000 characters:

**The handler `if` field:**

> | `if`            | no       | Permission rule syntax to filter when this hook runs, such as `"Bash(git *)"` or `"Edit(*.ts)"`. The hook command only runs if the tool call matches the pattern. See the [Bash matching table](#bash-if-matching) below for how Bash patterns evaluate against subcommands, `$()`, and backticks. Only evaluated on tool events: `PreToolUse`, `PostToolUse`, `PostToolUseFailure`, `PermissionRequest`, and `PermissionDenied`. On other events, a hook with `if` set never runs. Uses the same syntax as [permission rules](/docs/en/permissions)

> The `if` field holds exactly one permission rule. There is no `&&`, `||`, or list syntax for combining rules; to apply multiple conditions, define a separate hook handler for each.

> | `Bash(git *)`      | `npm test && git push`      | yes        | each subcommand is checked; `git push` matches                                                                            |

> When Claude Code can't determine which commands the Bash input runs, it runs your hook regardless of the pattern. Because the `if` filter is best-effort, use the [permission system](/docs/en/permissions) rather than a hook to enforce a hard allow or deny.

**Matcher grammar (exact names vs regex):**

> | Only letters, digits, `_`, `-`, spaces, `,`, and `\|` | Exact string, or list of exact strings separated by `\|` or `,` with optional surrounding whitespace | `Bash` matches only the Bash tool; `Edit\|Write` and `Edit, Write` each match either tool exactly; `code-reviewer` matches only that agent type |
> | Contains any other character                          | JavaScript regular expression, unanchored                                                            | `^Notebook` matches any tool whose name starts with `Notebook`; `mcp__memory__.*` matches every tool from the `memory` server                   |

> A matcher on the regular-expression path is tested with JavaScript's `RegExp.prototype.test`, which succeeds on a match anywhere in the value. `Edit.*` matches both `Edit` and `NotebookEdit`; wrap the pattern in `^` and `$`, as in `^Edit$`, when you need a whole-string match.

**Exit code 2 blocking, and the non-zero "hook error":**

> Exit 2 means a blocking error. On [events that can block](#exit-code-2-behavior-per-event), exit 2 blocks whether or not you print JSON: even a JSON `permissionDecision` of `"allow"` can't override it. Claude Code still reads any valid [JSON output](#json-output) on stdout. On `Elicitation` and `ElicitationResult`, an exit-2 hook's `hookSpecificOutput` is ignored.

> | `PreToolUse`          | Yes        | Blocks the tool call |

> * With stdout that Claude Code [treats as plain text](#exit-code-0), or with empty stdout, it's a non-blocking error for most hook events: the action proceeds, and the transcript shows a `<hook name> hook error` notice followed by the first line of stderr, prefixed with `Failed with non-blocking status code:`. To capture the full stderr, enable [debug logging](#debug-hooks).

> For events that use the standard decision model, when Claude Code tries to parse your stdout as JSON and can't, it reports a non-blocking error on every exit code other than 2. The transcript shows a `<hook name> hook error` notice with the parse message. On the events that add plain-text stdout as context, Claude Code doesn't add the text. Before v2.1.248, Claude Code treated that stdout as plain text.

**Default timeout:**

> | `timeout`       | no       | Seconds before canceling. Claude Code doesn't enforce it on a command hook you run with [`async: true`](#run-hooks-in-the-background). Defaults: 600 for `command`, `http`, and `mcp_tool`; 30 for `prompt`; 60 for `agent`. […]

> * A timed-out `command`, `http`, or `mcp_tool` hook doesn't block the tool call. The call continues through the normal [permission flow](/docs/en/permissions), so don't count on a stalled hook to act as a gate.

**Also relevant to the five open points (quoted, not interpreted):**

> All matching hooks run in parallel. If you define the same handler in more than one settings file, it runs once. A plugin's or skill's copy of the same handler stays separate.

> Hooks from settings files, managed policy settings, and plugins also run inside [subagents](/docs/en/sub-agents). When a subagent calls a tool, tool events such as `PreToolUse` and `PostToolUse` fire the same configured hooks as in the main conversation, and the input carries the `agent_id` and `agent_type` [common input fields](#common-input-fields) that identify the subagent.

> Hook execution details are written to the debug log file. Start Claude Code with `claude --debug-file <path>` to write the log to a known location, or run `claude --debug` and read the log at `~/.claude/debug/<session-id>.txt`. The `--debug` flag doesn't print to the terminal.

The reference says nothing about deduplicating same-command handlers *within one* settings file
that differ only in `if`; that is the dedup point the Bash `rg`/`find` prompts settle live.

## Scaffold

Built at HEAD `d3201ba9`, everything under `/tmp/06-live/` (nothing under `$HOME` written):

```bash
rm -rf /tmp/06-live && mkdir -p /tmp/06-live/debug /tmp/06-live/transcripts
GOTOOLCHAIN=go1.26.6 go build -o /tmp/06-live/codegraph ./cmd/codegraph
for n in indexed unindexed; do
  mkdir -p /tmp/06-live/$n && cp -R internal/indexer/testdata/gofixture/. /tmp/06-live/$n/ && git -C /tmp/06-live/$n init -q
done
CODEGRAPH_NO_WATCH=1 /tmp/06-live/codegraph init /tmp/06-live/indexed
#   Files=4 nodes=20 edges=22 duration=50ms
(cd /tmp/06-live/indexed   && /tmp/06-live/codegraph install --target claude --location local --pretool-nudge)
(cd /tmp/06-live/unindexed && /tmp/06-live/codegraph install --target claude --location local --pretool-nudge)
#   both: exit 0; created .mcp.json, .claude/CLAUDE.md, .claude/skills/codegraph/SKILL.md,
#   .claude/hooks/session-nudge.sh, .claude/settings.json, .claude/hooks/pretooluse-nudge.sh,
#   .claude/skills/codegraph/.codegraph-manifest.json
```

Positive assertions (identical for both repos except `.codegraph`):

```
== indexed
-rwxr-xr-x  1187  .claude/hooks/pretooluse-nudge.sh
-rwxr-xr-x   771  .claude/hooks/session-nudge.sh
19:codegraph_bin='/tmp/06-live/codegraph'
.hooks.PreToolUse | length  ->  6
{"matcher":"Bash","hooks":[{"type":"command","if":"Bash(grep *)","command":"${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh"}]}
{"matcher":"Bash","hooks":[{"type":"command","if":"Bash(rg *)","command":"${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh"}]}
{"matcher":"Bash","hooks":[{"type":"command","if":"Bash(find *)","command":"${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh"}]}
{"matcher":"Grep","hooks":[{"type":"command","if":null,"command":"${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh"}]}
{"matcher":"Glob","hooks":[{"type":"command","if":null,"command":"${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh"}]}
{"matcher":"Read","hooks":[{"type":"command","if":null,"command":"${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh"}]}
.codegraph: dir
== unindexed
(same guard listing, same rendered codegraph_bin line, same six PreToolUse blocks)
.codegraph: absent
```

`.mcp.json` in both repos registers `codegraph` as `/tmp/06-live/codegraph serve --mcp`
(stdio), and `.claude/settings.json` also carries the SessionStart `session-nudge.sh` entries
(`startup`, `resume`). A project-scope `.mcp.json` may raise Claude Code's own MCP-server
approval prompt at session start; that prompt is not from the PreToolUse hook and does not
count against C7.

Smoke test outside Claude Code (a synthetic Grep stdin with `session_id` `scaffold-smoke-0000`,
`CLAUDE_PROJECT_DIR` set): the indexed guard printed the pinned JSON on the first run and
nothing on the second (cooldown), both exit 0; the un-indexed guard printed nothing, exit 0.
The one sentinel it wrote was deleted afterwards and the sentinel directory removed, so the
sessions start with no sentinels. This shows the pinned substring is present in the guard's
output; it does not replace the live positive control in session A.

```
{"hookSpecificOutput":{"hookEventName":"PreToolUse","additionalContext":"This repo has a codegraph index: codegraph_explore (CLI: `codegraph explore`) returns the matching symbols' source and call paths for where-is-X and how-does-Y questions."}}
[run 1 exit 0]
[run 2 exit 0]
[unindexed exit 0]
```

## Session A protocol (indexed; Task 2)

cwd `/tmp/06-live/indexed`. Clear stale state first: `rm -rf "${TMPDIR}/codegraph-nudge-$(id -u)"`.
Start: `herdr agent start live-a --kind claude --pane <id> -- --debug-file /tmp/06-live/debug/live-a.log`.
Prompts, in order, each with `herdr agent prompt live-a "<text>" --wait`, noting wall-clock times:

1. `Use the Grep tool to find where the function Alpha is defined in this repository, and tell me the file and line.` → expect exactly one fire (C1).
2. Immediately (well inside 60 s of prompt 1's fire): `Use the Grep tool to find every call site of Alpha, then use the Glob tool to list the .go files under pkgb, then Read pkgb/pkgb.go.` → several matched calls, expect 0 fires (C2).
3. Wait at least 65 s after prompt 1's fire (`sleep 65` in the orchestrator's own shell), then: `Use the Grep tool to find where Beta is defined.` → expect exactly one fire (C3).
4. Within 60 s of prompt 3's fire: `Use the Task tool to start one general-purpose subagent. Tell it to use the Grep tool to find where Alpha is defined and report the line. Do not search yourself.` → expect exactly one fire inside the subagent, none in the main thread (C4).
5. Wait at least 65 s, then: `Run exactly this with the Bash tool: rg -n Alpha .` → record `Bash rg path fired: yes|no` and, from the debug log, how many times the guard ran for that one call (dedup point).
5b. Wait at least 65 s, then: `Run exactly this with the Bash tool: find . -name '*.go' -newer go.mod` → record whether it fired and, from the debug log, how many times the guard ran. The three Bash rules are three single-handler blocks with the SAME command, so fold this `find` result into the dedup Point line beside the `rg` result: if either `rg` or `find` never fires because Claude Code deduplicated same-command handlers across blocks, that is an ESCALATE.
6. Wait at least 65 s, then: `Run exactly this with the Bash tool: git log --oneline | rg -c .` → expect 0 fires: `FP check (git log | rg): <n> fires`.
7. `Run exactly this with the Bash tool: echo "$CLAUDE_CODE_SESSION_ID"` → CLAUDE_CODE_SESSION_ID export point (compare with the transcript's session id).
8. `/clear`, then: `Use the Grep tool to find where Alpha is defined.` → expect exactly one fire (C5).

Save `herdr agent read live-a --source recent-unwrapped > /tmp/06-live/transcripts/live-a.txt`
and copy the session JSONL(s) (main and subagent). Positive control first: show the pinned
substring search finding prompt 1's fire, then reuse the same search for every window.

## Session B protocol (un-indexed; Task 3)

cwd `/tmp/06-live/unindexed`. Start:
`herdr agent start live-b --kind claude --pane <id> -- --debug-file /tmp/06-live/debug/live-b.log`.
Prompts 1 and 5 from session A:

1. `Use the Grep tool to find where the function Alpha is defined in this repository, and tell me the file and line.`
5. `Run exactly this with the Bash tool: rg -n Alpha .`

Save the transcript and count fires with the SAME pinned-substring search already
positive-controlled on session A. Confirm from live-b's debug log that the guard ran and exited 0.
C7 is then assessed over both sessions' debug logs and transcripts, and the global config is
re-checked against the pre-flight checksum.

## Fire log

| Key | Fire timestamps | Same-key gaps |
|-----|-----------------|---------------|
| main (before `/clear`) | (orchestrator fills from session A) | |
| subagent | (orchestrator fills from session A) | |
| main (after `/clear`) | (orchestrator fills from session A) | |
| un-indexed main | (orchestrator fills from session B) | |

## Verdicts (D-18 pass bar, locked in 06-CONTEXT before any session)

C1 first matched call fires once: PENDING

C2 no same-key fire within 60 s: PENDING

C3 fires again after a >= 60 s gap: PENDING

C4 subagent first matched call fires once: PENDING

C5 fires after /clear: PENDING

C6 un-indexed control: PENDING

C7 no hook error, prompt, deny or block from the hook: PENDING

D-18 verdict: PENDING

## Recorded, not gated

Matched calls: PENDING

Fires: PENDING

Fire rate: PENDING

True-positive fires: PENDING

Uptake: PENDING

Hook wall time: PENDING

Bash rg path fired: PENDING

FP check (git log | rg): PENDING

## The five points the docs do not confirm

Point additionalContext without a decision: PENDING

Point subagent session_id: PENDING

Point CLAUDE_CODE_SESSION_ID export: PENDING

Point same-command handlers with different if: PENDING

Point stdin key order: PENDING

## Global configuration

Global config unchanged: PENDING
