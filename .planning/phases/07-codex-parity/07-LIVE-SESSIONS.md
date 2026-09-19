# Phase 7: Live-Session Evidence — Codex parity

**Status:** Task 1 scaffold complete (2026-09-19, executor). Tasks 2 and 3 are ORCHESTRATOR
work (Herdr pane, real Codex TUI, real model sessions) and are not performed by this executor.

## CODEX-01 (before any codex.go change)

### Method

Per 07-CONTEXT.md D-02..D-05, this evidence is gathered by the **orchestrator**, never by an
executor subagent — a backgrounded subagent cannot reliably drive another pane's TTY (the
05-06/06-06 precedent, D-09/D-18). The orchestrator drives one interactive Codex TUI session in
a sibling Herdr pane plus scripted `codex exec --json -C <repo>` / `codex mcp list --json` /
`codex debug prompt-input` / `codex features list` runs, all under the scratch environment
`HOME=$S/home CODEX_HOME=$S/home/.codex` (plus `XDG_CONFIG_HOME=$S/home/.config` for codegraph
runs), where `$S` is the real (non-symlinked) path recorded below.

Evidence tiers run **model-free first** (D-03): `codex features list`, `codex mcp list --json`,
`codex debug prompt-input`. The only model turns are the A1 hook probes (Task 2 step 5, Task 3
step 3) and the A2/D-15/D-16 real-session checks (Task 2 step 8), whose evidence is the session
JSONL's *injected context* — never the model's own paraphrase of what it saw.

**Negative-space rule (STATE standing rule).** A transcript or listing grep is a claim about the
grep. Every absence claim (L2, L4, an untrusted "no") must be backed by a search first shown to
FIND the thing when present in the trusted/positive-control case, before a zero or an absence is
recorded.

**Never trust the untrusted project.** `$S/untrusted` is a negative control (D-05) and must never
be granted trust by any means other than the one explicitly recorded
`Trust override (-c projects trust_level) grants trust:` line — and even that override is
recorded, never acted on to make `untrusted` a normal working repo for any other check.

**Distinct project-layer binary.** The project layer's `.codex/config.toml` in `trusted` and
`untrusted` points at `$S/codegraph-project` — a byte-identical but distinct copy of the scratch
`$S/codegraph` binary used for the global install — so `codex mcp list --json`'s merged,
scope-less output can be attributed to the project layer vs. the global layer by *which binary
path* appears, since the JSON carries no scope column (07-RESEARCH anti-pattern 3).

### Pass bar (07-CONTEXT D-06, locked 2026-09-19)

| Check | Passes when |
|---|---|
| L1 | In a trusted local scratch repo, `codex mcp list --json` shows `codegraph` from the project `.codex/config.toml`; with a global install it shows at global scope too (CODEX-06 "both scopes") |
| L2 | In the same repo left untrusted, the project-layer entry is NOT loaded; any warning is recorded |
| L3 | `codex debug prompt-input` lists the codegraph skill (from the shared `.agents/skills`) and the codegraph `AGENTS.md` block |
| L4 | An uninstalled scratch repo in the same scratch HOME shows no codegraph surface at all |
| L5 (CODEX-06) | A fresh session, from a prompt that never names codegraph, lists the skill **and** makes at least one `mcp__codegraph__codegraph_explore` call or `codegraph explore` run |
| L6 (CODEX-05) | With the nudge opted in and the hook trusted, the first matched Bash search call (grep/rg/find first word) produces exactly one `additionalContext` delivery; the 60 s per-key cooldown holds; an un-indexed repo gets 0 fires; no hook error or block is attributable to our hook |
| L7 | The real-HOME checksums are unchanged |

CODEX-01 in this document settles L1 (both scopes), L2, L3, L4 and L7. L5 (CODEX-06) and L6
(CODEX-05) are settled in their own later plans' live sessions.

`CODEX-01 verdict: PASS` requires L1 (project and global), L2, L3, L4 and L7 all PASS.

## Pre-flight

Recorded 2026-09-19 (read-only; nothing under the real `$HOME` written — the only real-HOME
operations are the four `shasum -a 256` lines below, `test -e ~/.codex/auth.json`, `readlink`,
and `codex --version`):

```
$ codex --version
codex-cli 0.155.0

$ test -e "$HOME/.codex/auth.json"; echo "exit: $?"
exit: 0

$ shasum -a 256 ~/.codex/config.toml
b4077b8a786230575935c44cfa85c17da7ca2bf3f1386675c8c6b22d322dbb17  /Users/sean/.codex/config.toml

$ shasum -a 256 ~/.codex/hooks.json
e0ad2381a6b1bf7933438d7946914737be14ca398ae41090298183c21fb49a99  /Users/sean/.codex/hooks.json

$ shasum -a 256 ~/.codex/AGENTS.md
54e268bda66adfb9c0d17a0eb73235a452f390b68d83ba1eea1dceada5379982  /Users/sean/.codex/AGENTS.md

$ shasum -a 256 ~/.agents/skills/codegraph/SKILL.md
e711379d68094bffbd5d997cccd172bb43ec3c8fa9276e06c67a59b4ce647644  /Users/sean/.agents/skills/codegraph/SKILL.md
```

(No file listed above was ever missing, so no `absent` case applies this run. Contents of these
four files are never printed here or anywhere else in this document — D-02/D-08.)

```
$ readlink "$S/home/.codex/auth.json"
/Users/sean/.codex/auth.json
```

`HOME="$S/home" CODEX_HOME="$S/home/.codex" codex features list` (scratch, model-free) — the
relevant line:

```
hooks                                    stable             true
```

(The full listing has ~150 rows; only the `hooks` row is evidentiary here. Task 3 step 3 reuses
this same probe with `-c features.hooks=false` for the optional off-switch check.)

## Codex reference (dated citations for CODEX-01)

Fetched 2026-09-19 with `curl -fsSL <url>.md` (Markdown form of the doc page, per each page's own
"append `.md` to the page URL" note), saved under `$S/transcripts/docs/`. Local `codex --version`
for these sessions: `codex-cli 0.155.0`.

**(a) Project config loads only for trusted projects; the `projects.<path>.trust_level` key.**

From `https://developers.openai.com/codex/local-config.md` (fetched 2026-09-19):

> Codex reads configuration details from more than one location. Your personal defaults live in
> `~/.codex/config.toml`, and you can add project overrides with `.codex/config.toml` files. For
> security, Codex loads project `.codex/` layers only when you trust the project.

> 2. Project config files: `.codex/config.toml`, ordered from the project root down to your
> current working directory (closest wins; trusted projects only)

> If you mark a project as untrusted, Codex skips project-scoped `.codex/` layers, including
> project-local config, hooks, and rules. User and system config still load, including
> user/global hooks and rules.

From `https://learn.chatgpt.com/docs/agent-approvals-security.md` (fetched 2026-09-19), the
`trust_level` key itself:

> To preserve the stricter command-approval rule, omit an explicit `approval_policy` and add a
> project entry to your user-level `~/.codex/config.toml`:
>
> ```toml
> [projects."/path/to/project"]
> trust_level = "untrusted"
> ```
>
> Commands then require approval unless an execution-policy rule allows them. This also disables
> project-local configuration.

Neither page's Markdown documents a `trust_level = "trusted"` example explicitly, nor whether a
`-c 'projects."<path>".trust_level="trusted"'` CLI override grants trust the same way the real
TUI prompt does — that is exactly the live gap Task 2 step 3 settles (`Trust override
(-c projects trust_level) grants trust:` line below).

**(b) Skill locations — `.agents/skills`, `$HOME/.agents/skills`, and any `.codex/skills` /
`$CODEX_HOME/skills` mention.**

From `https://developers.openai.com/codex/skills.md` (fetched 2026-09-19):

> Codex reads skills from repository, user, admin, and system locations. For repositories, Codex
> scans `.agents/skills` in every directory from your current working directory up to the
> repository root. If two skills share the same `name`, Codex doesn't merge them; both can appear
> in skill selectors.

> | `REPO` | `$CWD/.agents/skills` — Current working directory: where you launch Codex. |
> | `REPO` | `$CWD/../.agents/skills` — A folder above CWD when you launch Codex inside a Git repository. |
> | `REPO` | `$REPO_ROOT/.agents/skills` — The topmost root folder when you launch Codex inside a Git repository. |
> | `USER` | `$HOME/.agents/skills` — Any skills checked into the user's personal folder. |
> | `ADMIN` | `/etc/codex/skills` — Any skills checked into the machine or container in a shared, system location. |
> | `SYSTEM` | Bundled with Codex by OpenAI. |

This documented table names no `.codex/skills` or `$CODEX_HOME/skills` location at all — matching
07-CONTEXT.md's D-15 framing that those two roots are "read-only `SkillDirs[1:]` entries only if
the live check shows Codex reading them." The `cgprobe-dotcodex` and `cgprobe-codexhome` SKILL.md
probes scaffolded below exist to settle that live, since the docs are silent on it.

**(c) Hooks — locations, `/hooks` trust review, PreToolUse input fields,
`hookSpecificOutput.additionalContext`.**

From `https://learn.chatgpt.com/docs/hooks.md` (fetched 2026-09-19):

Locations:

> - `~/.codex/hooks.json`
> - `~/.codex/config.toml`
> - `<repo>/.codex/hooks.json`
> - `<repo>/.codex/config.toml`

> Project-local hooks load only when the project `.codex/` layer is trusted. In untrusted
> projects, Codex still loads user and system hooks from their own active config layers.

Trust review:

> Codex lists configured hooks before deciding which ones can run. Before a non-managed hook can
> run, Codex requires you to review and trust the exact hook definition. Codex records trust
> against the hook's current hash, so new or changed hooks are marked for review and skipped
> until trusted.
>
> Use `/hooks` in the CLI to inspect hook sources, review new or changed hooks, trust hooks, or
> disable individual non-managed hooks. If hooks need review at startup, Codex prints a warning
> that tells you to open `/hooks`.

The documented example command form for a repo-local hook (confirming D-20's chosen quoting is
exactly what the docs themselves show, not an invention):

> ```toml
> [[hooks.PreToolUse]]
> matcher = "^Bash$"
>
> [[hooks.PreToolUse.hooks]]
> type = "command"
> command = '/usr/bin/python3 "$(git rev-parse --show-toplevel)/.codex/hooks/pre_tool_use_policy.py"'
> timeout = 30
> ```

> - For repo-local hooks, prefer resolving from the git root instead of using a relative path
> such as `.codex/hooks/...`. Codex may be started from a subdirectory, and a git-root-based path
> keeps the hook location stable.

PreToolUse input fields (this is the documented set — no `agent_id` field is listed for
PreToolUse; that field is documented only for `SubagentStart`/`SubagentStop`, matching
07-CONTEXT.md's "Not confirmed" flag on D-21's `agent_id` assumption for PreToolUse specifically):

> Fields in addition to Common input fields:
>
> | `turn_id` | `string` | Codex-specific extension. Active Codex turn id |
> | `tool_name` | `string` | Canonical hook tool name, such as `Bash`, `apply_patch`, or an MCP name like `mcp__fs__read` |
> | `tool_use_id` | `string` | Tool-call id for this invocation |
> | `tool_input` | `JSON value` | Tool-specific input. `Bash` and `apply_patch` use `tool_input.command`. MCP and other local function tools send their arguments. |

Common input fields (shared across all hook events, including PreToolUse):

> | `session_id` | `string` | Current Codex session id. Subagent hooks use the parent session id. |
> | `cwd` | `string` | Working directory for the session |
> | `hook_event_name` | `string` | Current hook event name |

Default timeout: "If `timeout` is omitted, Codex uses `600` seconds for most hooks." Matcher
grammar: "The `matcher` field is a regex string that filters when hooks fire."

`hookSpecificOutput.additionalContext` (deny/allow is a separate, unrelated field on the same
object; this session never uses `permissionDecision`):

> JSON on `stdout` can use `systemMessage`. To deny a supported tool call, return this
> hook-specific shape: `{"hookSpecificOutput": {"hookEventName": "PreToolUse",
> "permissionDecision": "deny", ...}}`

(the probes scaffolded below never return this shape — they print nothing and exit 0, per D-19's
"never tell users to use `--dangerously-bypass-hook-trust`" and this plan's own prohibition on
patching `codex.go`.)

If a future re-fetch of any of these three pages fails, fall back to Context7 `/openai/codex`
with its own fetch date, noted inline at that point — not silently substituted here.

## Scaffold

Built under `$S=$(cd /tmp && pwd -P)/07-live` (the real, non-symlinked `/private/tmp/07-live` on
this macOS host, so trust keys never differ by the `/tmp` symlink). Nothing under the real
`$HOME` was written.

```bash
S=$(cd /tmp && pwd -P)/07-live   # /private/tmp/07-live
rm -rf "$S"; mkdir -p "$S/home/.codex" "$S/probes" "$S/transcripts"
GOTOOLCHAIN=go1.26.6 go build -o "$S/codegraph" ./cmd/codegraph
cp "$S/codegraph" "$S/codegraph-project"
ln -s "$HOME/.codex/auth.json" "$S/home/.codex/auth.json"   # symlink, never a copy (D-02)

for n in trusted untrusted bare; do
  mkdir -p "$S/$n"
  cp -R internal/indexer/testdata/gofixture/. "$S/$n/"
  git -C "$S/$n" init -q
done
CODEGRAPH_NO_WATCH=1 "$S/codegraph" init "$S/trusted"     # files=4 nodes=20 edges=22
CODEGRAPH_NO_WATCH=1 "$S/codegraph" init "$S/untrusted"   # files=4 nodes=20 edges=22

for n in trusted untrusted; do
  (cd "$S/$n" && HOME="$S/home" CODEX_HOME="$S/home/.codex" XDG_CONFIG_HOME="$S/home/.config" \
    "$S/codegraph" install --target opencode --location local --yes)
  # created: opencode.jsonc, AGENTS.md, .agents/skills/codegraph/SKILL.md,
  #          .agents/skills/codegraph/.codegraph-manifest.json
  mkdir -p "$S/$n/.codex"
  cat > "$S/$n/.codex/config.toml" <<EOF
[mcp_servers.codegraph]
command = "$S/codegraph-project"
args = ["serve", "--mcp"]
EOF
done
```

This project-layer `.codex/config.toml` reproduces exactly the bytes `codexTableBody` (in
`internal/agents/codex.go`) renders for `execPath = "$S/codegraph-project"` — hand-planted
because before any `codex.go` change (D-01), codegraph itself cannot write a project-local
`.codex/config.toml` at all (`Install` early-returns for `loc != LocationGlobal`).

D-15 probes:

```bash
for n in trusted untrusted; do
  mkdir -p "$S/$n/.codex/skills/cgprobe-dotcodex"
  # SKILL.md: name: cgprobe-dotcodex, description: D-15 probe for the project .codex/skills root
done
mkdir -p "$S/home/.codex/skills/cgprobe-codexhome"
# SKILL.md: name: cgprobe-codexhome, description: D-15 probe for the CODEX_HOME/skills root
```

A1 local probe (`trusted` only) — `.codex/hooks/a1-probe.sh` (mode 0755, POSIX `sh`):

```sh
#!/bin/sh
printf 'a1-local %s %s\n' "$(date -u +%FT%TZ)" "$PWD" >> "$S/probes/a1-local.log"
cat > "$S/probes/a1-local-stdin-$(date +%s).json"
exit 0
```

and `.codex/hooks.json` with one `PreToolUse` group, `matcher: "^Bash$"`, one handler, `type:
"command"`, `timeout: 5`, and `command` equal to the exact D-20 local form (the outer double
quotes are literal characters inside the string, matching the docs' own example above):

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "^Bash$",
        "hooks": [
          {
            "type": "command",
            "command": "\"$(git rev-parse --show-toplevel)/.codex/hooks/a1-probe.sh\"",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
```

`$S/untrusted` deliberately carries no `.codex/hooks.json` and no A1 probe — the negative
control (D-05) never gets a hook to trust in the first place.

Positive assertions (verified by this plan's own `<verify>` gate, re-stated here):

```
-rwxr-xr-x  $S/codegraph
-rwxr-xr-x  $S/codegraph-project
lrwxr-xr-x  $S/home/.codex/auth.json -> /Users/sean/.codex/auth.json
$S/trusted/.codegraph:    dir (indexed)
$S/untrusted/.codegraph:  dir (indexed)
$S/bare/.codegraph:       absent
$S/bare/AGENTS.md:        absent
$S/bare/.codex:           absent
$S/trusted/.codex/config.toml    command = "$S/codegraph-project" (1 match)
$S/untrusted/.codex/config.toml  command = "$S/codegraph-project" (1 match)
$S/trusted/AGENTS.md              contains <!-- CODEGRAPH_START -->
$S/trusted/.agents/skills/codegraph/SKILL.md   present
$S/trusted/.codex/hooks/a1-probe.sh            present, mode 0755
$S/trusted/.codex/hooks.json .hooks.PreToolUse[0].hooks[0].command == "$(git rev-parse --show-toplevel)/.codex/hooks/a1-probe.sh"
$S/untrusted/.codex/hooks.json    absent
$S/home/.codex/skills/cgprobe-codexhome/SKILL.md  present
$S/home/.codex/config.toml        absent (no global install yet — Task 3's job)
```

## Protocol A (model-free, before any trust exists) — ORCHESTRATOR, Task 2

1. `$S/bare`: `codex mcp list --json` and `codex debug prompt-input`. L4 is recorded only after
   step 7's positive control exists (the same searches must first find the codegraph surface in
   `trusted` before their absence in `bare` counts as PASS).
2. `$S/untrusted`: `codex mcp list --json` (stdout and stderr) and `codex debug prompt-input`.
   `L2 untrusted project layer not loaded: PASS` when no `codegraph` entry appears (true today
   since no global install exists yet either); record any warning verbatim on the
   `Untrusted warning:` line, or `none observed`.
3. Trust override: `codex -c 'projects."'"$S"'/untrusted".trust_level="trusted"' mcp list --json`
   in `$S/untrusted` → `Trust override (-c projects trust_level) grants trust: yes|no (<excerpt>)`.
   `untrusted` is never trusted any other way, ever, for the rest of this plan.

## Protocol B (one interactive TUI + minimal model turns) — ORCHESTRATOR, Task 2

4. Start the Codex TUI in a sibling Herdr pane, scratch env, cwd `$S/trusted`; accept the trust
   prompt; show only the `[projects."…"]` lines Codex wrote to the scratch config.
5. `/hooks`, trust the a1-probe hook, prompt `Run this exact shell command and nothing else: echo
   a1-probe`. `A1 local command form shell-expanded: yes` only if `$S/probes/a1-local.log` gained
   a line for this turn; record the stdin field list and whether `tool_input.command` is a string
   or an array on `PreToolUse stdin fields (main thread):`. `Hooks.json runs behind
   features.hooks: yes|no` from the same marker.
6. L1 project: `codex mcp list --json` in `$S/trusted` → PASS when `codegraph` appears with
   `$S/codegraph-project` in its transport.
7. L3/D-15: `codex debug prompt-input` in `$S/trusted` → this is the positive control for step
   1's absence searches. `D-15 .codex/skills read`, `D-15 CODEX_HOME/skills read`, `D-17 skill
   description as listed`.
8. A2/D-16 in REAL sessions: `codex exec --json -C "$S/untrusted" '...'` and the TUI session in
   `$S/trusted`; search each session's JSONL for the codegraph skill listing and `## CodeGraph`,
   with the trusted JSONL as the positive control.

## Protocol C (global install) — ORCHESTRATOR, Task 3

1. `(cd "$S/bare" && HOME="$S/home" CODEX_HOME="$S/home/.codex" XDG_CONFIG_HOME="$S/home/.config"
   "$S/codegraph" install --target codex --location global --yes)`; the HEAD binary's own fixed
   TOML splice (07-01) writes `$S/home/.codex/config.toml` for the first time in this scratch —
   record whether the scratch config's `[projects."…"]` trust lines from Protocol B survive.
2. L1 global: `codex mcp list --json` in `$S/bare` shows `$S/codegraph` (global layer); in
   `$S/trusted` still shows `$S/codegraph-project` (project layer wins).
3. A1 global: quoted absolute path to `$S/home/.codex/hooks/a1-probe-global.sh`; trust it in
   `/hooks`; prompt `echo a1-global`.
4. L7: re-run the four `shasum -a 256` commands against the real `$HOME`; PASS only when every
   value equals the pre-flight record above.
5. `CODEX-01 verdict: PASS` only if L1 (project), L1 (global), L2, L3, L4 and L7 are all PASS.

### Protocol A/B evidence (orchestrator, 2026-09-19, codex-cli 0.155.0)

All runs used `HOME=/private/tmp/07-live/home CODEX_HOME=/private/tmp/07-live/home/.codex`. Transcripts are in `/private/tmp/07-live/transcripts/`. The orchestrator re-took the pre-flight sha256 independently (`transcripts/preflight-orchestrator.sha`), and all four values equal the scaffold's record.

**A1: bare, model-free** (`A1-bare-mcp.json`, `A1-bare-prompt.json`)

```
$ codex mcp list --json          # in $S/bare
[]
$ codex debug prompt-input       # 14068 bytes; stderr empty
searches:  codegraph=0  CODEGRAPH_START=0  "## CodeGraph"=0  cgprobe-dotcodex=0  cgprobe-codexhome=1
```

**A2: untrusted, model-free** (`A2-untrusted-mcp.json`, `A2-untrusted-prompt.json`). The project `.codex/config.toml` is identical in shape to `trusted`'s (`command = "/private/tmp/07-live/codegraph-project"`).

```
$ codex mcp list --json          # in $S/untrusted; exit 0; stdout:
[]
# stderr: (empty)                # codex debug prompt-input stderr: (empty)
prompt-input searches:  codegraph=2  CODEGRAPH_START=1  "## CodeGraph"=1  cgprobe-dotcodex=1  cgprobe-codexhome=1
```

**A3: trust override** (`A3-override-mcp.json`)

```
$ codex -c 'projects."/private/tmp/07-live/untrusted".trust_level="trusted"' mcp list --json   # in $S/untrusted
[]
# exit 0; stderr empty; $S/home/.codex/config.toml still absent afterwards (the override wrote nothing)
```

Positive control for the override key: the real TUI trust (B4) wrote `[projects."/private/tmp/07-live/trusted"]` / `trust_level = "trusted"`, the same key shape the override used, and that trust makes the identical project config load (B6). The `-c` override does not grant trust.

**B4: real TUI trust prompt** (Herdr pane `w1H:pA`, `codex` in `$S/trusted`)

```
> You are in /private/tmp/07-live/trusted
  Do you trust the contents of this directory? Working with untrusted contents comes with higher risk
  of prompt injection. Trusting the directory allows project-local config, hooks, and exec policies to
  load.
› 1. Yes, continue
  2. No, quit
```

After "Yes, continue", the scratch config held exactly these trust lines:

```
[projects."/private/tmp/07-live/trusted"]
trust_level = "trusted"
```

**B5: hook review, A1 local and the stdin capture** (`B5-hook-review-before.txt`, `B5-hook-review-after.txt`)

```
  Hooks need review
  1 hook is new or changed.
  Hooks can run outside the sandbox after you trust them.
...
  [!] Hook 1 · new
  Event     PreToolUse
  Matcher   ^Bash$
  Source    Project config - /private/tmp/07-live/trusted/.codex/hooks.json
  Command   "$(git rev-parse --show-toplevel)/.codex/hooks/a1-probe.sh"
  Mode      Sync
  Timeout   5s
  Trust     New hook - review required
```

After pressing `t`, the hook showed `Trust     Trusted`, and the scratch config gained a position-keyed hash entry:

```
[hooks.state."/private/tmp/07-live/trusted/.codex/hooks.json:pre_tool_use:0:0"]
trusted_hash = "sha256:2c48671e342b9f4f695100bc3aa4a178502291526dfa0f2b6254f279e5f6d3e3"
```

The trust state key is file path, event, group index and handler index. This matters for D-23: our group must be appended last.

Probe turn (TUI prompt `Run this exact shell command and nothing else: echo a1-probe`). `$S/probes` was empty before the turn. Afterwards:

```
$ cat /private/tmp/07-live/probes/a1-local.log
a1-local 2026-09-19T17:02:54Z /private/tmp/07-live/trusted
$ jq -c keys probes/a1-local-stdin-1789837374.json
["cwd","hook_event_name","model","permission_mode","session_id","tool_input","tool_name","tool_use_id","transcript_path","turn_id"]
$ jq -c '{tool_name, command_type: (.tool_input.command|type), tool_input_keys: (.tool_input|keys), has_agent_id: has("agent_id")}'
{"tool_name":"Bash","command_type":"string","tool_input_keys":["command"],"has_agent_id":false}
tool_input = {"command":"echo a1-probe"}; session_id = 01a0ba9e-524d-7331-a576-91cf8d26ae06 (= the TUI rollout id)
```

`features.hooks` discrimination: two `codex exec --json -C $S/trusted` turns, each run with `< /dev/null`. The first attempt without it blocked on "Reading additional input from stdin..." and timed out (exit 124). It was discarded as inconclusive, not counted.

```
codex -c features.hooks=false exec ... 'echo a1-off'   -> /bin/zsh -lc 'echo a1-off' -> exit 0 ; marker lines 1 -> 1 (silent)
codex exec ... 'echo a1-on'                             -> /bin/zsh -lc 'echo a1-on'  -> exit 0 ; marker lines 1 -> 2
  a1-local 2026-09-19T17:07:08Z /private/tmp/07-live/trusted
```

**B6: L1 project** (`B6-trusted-mcp.json`)

```
$ codex mcp list --json          # in $S/trusted
{"name":"codegraph","enabled":true,"transport":{"type":"stdio","command":"/private/tmp/07-live/codegraph-project","args":["serve","--mcp"],"env":null,"env_vars":[],"cwd":null}}
```

**B7: L3, D-15 and D-17** (`B7-trusted-prompt.json`). This is the positive control for A1's absence searches.

```
searches (trusted vs bare):  codegraph 2/0  CODEGRAPH_START 1/0  "## CodeGraph" 1/0  cgprobe-dotcodex 1/0  cgprobe-codexhome 1/1
### Skill roots
- `r0` = `/private/tmp/07-live/trusted/.codex/skills`
- `r1` = `/private/tmp/07-live/home/.codex/skills`
- `r2` = `/private/tmp/07-live/home/.codex/skills/.system`
- `r3` = `/private/tmp/07-live/trusted/.agents/skills`
- cgprobe-dotcodex: D-15 probe for the project .codex/skills root (file: r0/cgprobe-dotcodex/SKILL.md)
- codegraph: Use when asked where X is defined, how Y works, what calls X, or what changing X breaks in a .codegraph/ repo. (file: r3/codegraph/SKILL.md)
- cgprobe-codexhome: D-15 probe for the CODEX_HOME/skills root (file: r1/cgprobe-codexhome/SKILL.md)
# AGENTS.md instructions for /private/tmp/07-live/trusted
  ...
  ## CodeGraph
```

SKILL.md frontmatter: `description: Use when asked where X is defined, how Y works, what calls X, or what changing X breaks in a .codegraph/ repo.` The listing renders it byte-identical, untruncated.

**B8: A2 and D-16 in real sessions.** Untrusted: `codex exec --json -C $S/untrusted 'Reply with the single word ok.' < /dev/null` (rollout `01a0baa3-068c-…`). Trusted: the TUI session (rollout `01a0ba9e-524d-…`). The untrusted repo was never trusted: the scratch config holds no `$S/untrusted` key.

```
injected-context search          trusted-TUI  untrusted-exec
"- codegraph: Use when"          2            2
"## CodeGraph"                   2            2
"CODEGRAPH_START"                2            2
"AGENTS.md instructions for"     1            1
"cgprobe-dotcodex"               2            2
turn_context: trusted   approval=on-request sandbox=workspace-write
              untrusted approval=never      sandbox=read-only      (so Codex did treat it as untrusted)
```

In a real untrusted session, Codex gates the project config (MCP servers: A2 and A3 `[]`) and hooks, but not `AGENTS.md`, project `.agents/skills` or project `.codex/skills`.

### CODEX-01 verdicts

L1 project config loads when trusted: PASS
L1 global entry shown (global install): PENDING
L2 untrusted project layer not loaded: PASS
L3 prompt-input lists skill and AGENTS.md block: PASS
L4 uninstalled repo shows no codegraph surface: PASS
L7 real HOME unchanged (CODEX-01): PENDING
Untrusted warning: none observed (mcp list / debug prompt-input / exec stderr carry no warning; the only trust messaging is the TUI prompt quoted in B4)
Trust override (-c projects trust_level) grants trust: no (A3: `[]` under the override; the same key written by the real TUI trust loads the layer, B6)
A1 local command form shell-expanded: yes (B5: a1-local 2026-09-19T17:02:54Z /private/tmp/07-live/trusted)
A1 global quoted command form runs: PENDING
A2 AGENTS.md trust-gated in a real session: no (B8: "## CodeGraph" injected in the untrusted read-only exec session, 2 vs 2)
D-16 project .agents/skills trust-gated: no (B8: "- codegraph: Use when" listed in the untrusted session, 2 vs 2)
D-15 .codex/skills read: yes (B7: r0 = trusted/.codex/skills lists cgprobe-dotcodex; also listed untrusted, B8)
D-15 CODEX_HOME/skills read: yes (B7: r1 = home/.codex/skills lists cgprobe-codexhome)
D-17 skill description as listed: "Use when asked where X is defined, how Y works, what calls X, or what changing X breaks in a .codegraph/ repo." (untruncated, byte-identical to SKILL.md frontmatter; B7)
Hooks.json runs behind features.hooks: yes (B5: marker 1->2 with hooks on, 1->1 with -c features.hooks=false; `hooks stable true` in features list)
PreToolUse stdin fields (main thread): cwd, hook_event_name, model, permission_mode, session_id, tool_input, tool_name, tool_use_id, transcript_path, turn_id; tool_name "Bash"; tool_input.command is a string; no agent_id/agent_type on the main thread (B5)
CODEX-01 verdict: PENDING
