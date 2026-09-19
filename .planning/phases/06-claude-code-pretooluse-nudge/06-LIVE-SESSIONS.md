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

## Run notes (2026-09-19, orchestrator — Claude Code 2.1.278, Opus 5, auto mode)

- Both sessions: the workspace-trust prompt was answered "Yes, I trust this folder" (records a trust entry in `~/.claude.json`, not `~/.claude/settings.json`) and the project MCP prompt "Use this MCP server" (so `codegraph_explore` was available and uptake could be observed).
- **The maintainer's global config exposes no Grep tool** — Session A's agent said "This session has no dedicated Grep tool, so I ran rg (ripgrep) from the shell instead". Every search therefore went through **Bash `rg`**, i.e. the `Bash(rg *)` handler; the `Grep`/`Glob` blocks were never exercised live (their registration is pinned by `TestPreToolUseRegistrationShape`).
- Timing deviations from the scripted protocol, all recorded, none weakening a criterion: prompt 2 ran 55–73 s after prompt 1's fire (so it straddles the cooldown boundary — better C2 evidence); prompt 3 ran ~40 s after the previous fire (inside the window → more C2 evidence, not C3); C4 was re-run so the subagent searched while the main key was still inside its window (the first attempt could not discriminate); C5 was re-run with explicit `rg` because after the first `/clear` the agent answered through `codegraph_explore` without searching.
- Fires are counted as `hook_additional_context` attachments carrying the pinned substring `returns the matching symbols' source and call paths` in the session JSONL(s) (main + `subagents/agent-*.jsonl`); positive control: that search finds prompt 1's fire (below) before any zero-count claim. Event list: `/tmp/06-live/transcripts/session-a-events.txt`, `session-b-events.txt`; debug logs `/tmp/06-live/debug/live-{a,b}.log`; timeline `/tmp/06-live/transcripts/timeline-final.txt`.

Positive control (session A JSONL, verbatim, truncated):

```
{"parentUuid":"ab5b1aff-…","isSidechain":false,"attachment":{"type":"hook_additional_context","content":["This repo has a codegraph index: codegraph_explore (CLI: `codegraph explore`) returns the matching symbols' source and call paths for where-is-X and how-does-Y quest…
2026-09-19T10:42:45.872Z [DEBUG] Hook PreToolUse (${CLAUDE_PROJECT_DIR}/.claude/hooks/pretooluse-nudge.sh) provided additionalContext (170 chars)
```

## Fire log

| Key | Fire timestamps (UTC) | Same-key gaps |
|-----|-----------------|---------------|
| main (session e8c8b2c2, before `/clear`) | 10:42:46.4, 10:43:46.8, 10:46:10.0, 10:47:32.1, 10:49:27.2 | 60.4 s, 143.2 s, 82.0 s, 115.1 s — silent same-key calls at 59.0 s and 59.6 s (10:43:44.9, 10:43:45.5), at 40.9/44.3/48.9 s (10:44:27.9, :31.3, :35.9) |
| subagent a6956955 (session e8c8b2c2) | 10:44:55.4 | its second call 5.0 s later (10:45:00.5) silent |
| subagent afc4f666 (session e8c8b2c2) | 10:46:19.4 — 9.4 s after the main key's 10:46:10.0 fire | — |
| main after 1st `/clear` (session 0bae455c) | 10:51:00.7 | follow-ups 10:51:05.2, 10:51:09.1 silent (4.5 s, 8.4 s) |
| main after 2nd `/clear` (session 5801bf99) | 10:51:37.3 — 36.5 s after the previous session's fire | — |
| un-indexed main (session B) | none (2 matched calls: 10:53:31.4, 10:53:52.8) | — |

Session A event excerpt (verbatim from `session-a-events.txt`):

```
2026-09-19T10:42:45.824Z  MAIN  TOOL Bash rg -n --no-heading '\bAlpha\b' /private/tmp/06-live/indexed
2026-09-19T10:42:46.412Z  MAIN  FIRE
2026-09-19T10:43:44.898Z  MAIN  TOOL Bash rg -n --no-heading '\bAlpha\(' /private/tmp/06-live/indexed
2026-09-19T10:43:45.465Z  MAIN  TOOL Bash rg --files -g '*.go' /private/tmp/06-live/indexed/pkgb
2026-09-19T10:43:46.215Z  MAIN  TOOL Read /private/tmp/06-live/indexed/pkgb/pkgb.go
2026-09-19T10:43:46.808Z  MAIN  FIRE
2026-09-19T10:46:09.440Z  MAIN  TOOL Bash rg -n --no-heading '\bWidget\b' /private/tmp/06-live/indexed
2026-09-19T10:46:10.022Z  MAIN  FIRE
2026-09-19T10:46:18.827Z  SUB:agent-afc4f666[side]  TOOL Bash rg -n --no-ignore --hidden -g '!.git' '^func\s+(\([^)]*\)\s*)?helper
2026-09-19T10:46:19.407Z  SUB:agent-afc4f666[side]  FIRE
2026-09-19T10:47:31.479Z  MAIN  TOOL Bash find . -name '*.go' -newer go.mod
2026-09-19T10:47:32.059Z  MAIN  FIRE
2026-09-19T10:48:53.269Z  MAIN  TOOL Bash git log --oneline | rg -c .
2026-09-19T10:51:36.695Z  MAIN  TOOL Bash rg -n Widget .
2026-09-19T10:51:37.276Z  MAIN  FIRE
```

Session B event excerpt (verbatim): `2026-09-19T10:53:31.350Z  MAIN  TOOL Bash ls -a && rg -n --no-ignore --hidden -g '!.git' '\bAlpha\b'` · `2026-09-19T10:53:52.799Z  MAIN  TOOL Bash rg -n Alpha .` — no FIRE line; pinned-substring hits across session B's JSONL: 0.

Sentinel directory after both sessions: `…/T/codegraph-nudge-501/` mode `drwx------`, 5 empty `-rw-------` files (hashed keys: 3 main sessions + 2 subagents; none from session B — the un-indexed guard never starts the binary).

## Verdicts (D-18 pass bar, locked in 06-CONTEXT before any session)

C1 first matched call fires once: PASS

The session's first matched call (`rg -n … '\bAlpha\b'`, 10:42:45.8) produced exactly one fire (10:42:46.4) — delivered as `hook_additional_context` and logged "provided additionalContext (170 chars)".

C2 no same-key fire within 60 s: PASS

Every same-key gap between fires is ≥ 60 s (60.4, 143.2, 82.0, 115.1 s); matched calls at 59.0 s and 59.6 s after a fire were silent and the call at 60.3 s fired; calls at 40.9–48.9 s were silent; follow-ups after `/clear` at 4.5 s and 8.4 s were silent; the first subagent's second call at 5.0 s was silent.

C3 fires again after a >= 60 s gap: PASS

After 143.2 s the main key fired again on `rg … '\bWidget\b'` (10:46:10.0); also at 82.0 s (find) and 115.1 s (rg Gamma).

C4 subagent first matched call fires once: PASS

Discriminating run: subagent afc4f666's first matched call fired at 10:46:19.4, 9.4 s after the main key's own fire (main still inside its window) — its own key. (First run: subagent a6956955 fired once, its second call 5.0 s later silent.) Both subagent transcripts carry the parent's `sessionId` and their own `agentId`.

C5 fires after /clear: PASS

Discriminating run: after the second `/clear` (new session 5801bf99) the first matched call fired at 10:51:37.3, 36.5 s after the previous session's fire at 10:51:00.7 — a new session id is a new key. (After the first `/clear` the new session's first matched call also fired, 10:51:00.7.)

C6 un-indexed control: PASS (0 fires)

Session B (un-indexed, same registration) made 2 matched calls; the same pinned-substring search that found every session-A fire finds 0. Debug log: the `rg` handler's `if` matched (grep/find handlers skipped) and no error was logged — the guard ran and stayed silent (its directory check exits before the binary; timing below).

C7 no hook error, prompt, deny or block from the hook: PASS

Positive control: the guard's command string appears 9 times in live-a.log (one "provided additionalContext" per fire). In BOTH debug logs: 0 lines matching `hook error|non-blocking error|blocking error`, 0 error/deny/block lines naming `pretooluse-nudge`; 0 `hook_*error` attachments in any transcript. The `permissionDecision: allow` / "Boost auto-rewrite" entries in live-a.log come from the maintainer's own global Boost hook (attributed by hook output, not by symptom), never from `pretooluse-nudge.sh`.

D-18 verdict: PASS

## Recorded, not gated

Matched calls: 18

Fires: 9

Fire rate: 9/18

True-positive fires: 7/9

Uptake: Session A's first session kept searching with `rg` after each fire (it named `codegraph explore` in its own commentary but did not call it); after `/clear`, with the SessionStart nudge and the codegraph skill in the fresh context, the agent answered the where-is-X prompt by calling `mcp__codegraph__codegraph_explore` directly, with no search at all.

Hook wall time: 2.8–9.0 ms (median 3.2) un-indexed guard, directory check only; 11.4–13.0 ms (median 12.2) indexed guard running the binary and firing — 20 runs each, measured directly outside Claude Code (the in-session tool_use→fire gap of ~0.6 s includes the maintainer's other hooks running in parallel)

Bash rg path fired: yes

FP check (git log | rg): 0 fires

(81.2 s after the previous fire, i.e. outside the cooldown: Claude Code's `Bash(rg *)` `if` matched the pipe tail, the guard ran, and the D-02 first-word rule dropped it. Bash `find` path also confirmed: `find . -name '*.go' -newer go.mod` fired at 10:47:32.1.)

## The five points the docs do not confirm

Point additionalContext without a decision: confirmed — a PreToolUse hook that returns only `hookSpecificOutput.additionalContext` (no `permissionDecision`) is delivered to the model as a `hook_additional_context` attachment and does not change the tool call.

Point subagent session_id: confirmed — subagent hook calls carry the PARENT's `session_id` (subagent JSONLs record `sessionId` e8c8b2c2… with their own `agentId`); the per-(session, agent) key is what separates them (D-06).

Point CLAUDE_CODE_SESSION_ID export: confirmed in 2.1.278 — `echo "$CLAUDE_CODE_SESSION_ID"` in the session printed e8c8b2c2-4169-405a-9c5a-b1a5d118c8e7, equal to the transcript's session id.

Point same-command handlers with different if: NOT deduplicated — the three single-handler Bash blocks (same command, different `if`) are evaluated independently: the debug log shows `Skipping hook due to if condition "Bash(grep *)" not matching` / `"Bash(find *)" not matching` while the `rg` handler ran, and for the `find` call the grep/rg handlers were skipped and the find handler ran and fired.

Point stdin key order: not relied on — the subcommand parses stdin with `encoding/json`, so key order is irrelevant; the live sessions exercised parsing successfully on every fire.

## Global configuration

Global config unchanged: yes — sha256 matches pre-flight (be3ad316a7c4d5d921569f0398839638824fc4254d660378075e65ff69a339ef before and after); `~/.claude/hooks/pretooluse-nudge.sh` absent before and after.
