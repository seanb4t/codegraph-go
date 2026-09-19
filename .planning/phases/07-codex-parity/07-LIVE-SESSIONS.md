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

### Protocol C evidence (orchestrator, 2026-09-19)

**C1: global install through the HEAD binary** (`transcripts/C1-install.txt`, `C1-config-before.toml`)

```
$ (cd $S/bare && HOME=$S/home CODEX_HOME=$S/home/.codex XDG_CONFIG_HOME=$S/home/.config $S/codegraph install --target codex --location global --yes)
Codex CLI: configured
  updated: /private/tmp/07-live/home/.codex/config.toml
  created: /private/tmp/07-live/home/.codex/AGENTS.md
trust/hook-state lines before: 5   after: 5
$ diff C1-config-before.toml $S/home/.codex/config.toml
10a11,14
>
> [mcp_servers.codegraph]
> command = "/private/tmp/07-live/codegraph"
> args = ["serve", "--mcp"]
```

The fixed splice from 07-01 appended only our table. Codex's own `[projects."…"]` and `[hooks.state."…"]` tables were untouched.

**C2: L1 global** (`C2-{bare,trusted,untrusted}-mcp.json`)

```
bare      [{"name":"codegraph","command":"/private/tmp/07-live/codegraph"}]            <- global layer
trusted   [{"name":"codegraph","command":"/private/tmp/07-live/codegraph-project"}]    <- project layer wins
untrusted [{"name":"codegraph","command":"/private/tmp/07-live/codegraph"}]            <- project layer still not loaded
```

**C3: A1 global quoted form** (`C3-hook-review-before.txt`, `C3-hook-review-after.txt`). The probe script was placed under a directory whose name contains a space (`.../hooks/space dir/a1-probe-global.sh`), so the single-quoted absolute command form is exercised against whitespace (the AR-06-08 case). The TUI was restarted (`/quit`, then `codex`). The Codex update offer (0.155.0 to 0.155.1) was declined with "Skip", so the version under test stayed 0.155.0.

```
  Hooks need review
  1 hook is new or changed.                      <- only the new global hook; the trusted project hook was not re-flagged
  [!] Hook 1 · new
  [x] Hook 2
  Source    User config - ~/.codex/hooks.json
  Command   '/private/tmp/07-live/home/.codex/hooks/space dir/a1-probe-global.sh'
  Trust     New hook - review required   -> (t) ->   Trust     Trusted
$ cat probes/a1-global.log                        # after prompt "Run this exact shell command and nothing else: echo a1-global"
a1-global 2026-09-19T17:11:22Z /private/tmp/07-live/trusted
$ tail -1 probes/a1-local.log                     # the project hook fired on the same turn
a1-local 2026-09-19T17:11:22Z /private/tmp/07-live/trusted
stdin: {"tool_name":"Bash","command":"echo a1-global","cwd":"/private/tmp/07-live/trusted","has_agent_id":false}
```

**C4: L7** (`transcripts/preflight-orchestrator.sha`, `transcripts/postflight.sha`)

```
same b4077b8a786230575935c44cfa85c17da7ca2bf3f1386675c8c6b22d322dbb17   ~/.codex/config.toml
same e0ad2381a6b1bf7933438d7946914737be14ca398ae41090298183c21fb49a99   ~/.codex/hooks.json
same 54e268bda66adfb9c0d17a0eb73235a452f390b68d83ba1eea1dceada5379982   ~/.codex/AGENTS.md
same e711379d68094bffbd5d997cccd172bb43ec3c8fa9276e06c67a59b4ce647644   ~/.agents/skills/codegraph/SKILL.md
$S/home/.codex/auth.json -> /Users/sean/.codex/auth.json   (symlink, never read or printed)
```

### CODEX-01 verdicts

L1 project config loads when trusted: PASS
L1 global entry shown (global install): PASS
L2 untrusted project layer not loaded: PASS
L3 prompt-input lists skill and AGENTS.md block: PASS
L4 uninstalled repo shows no codegraph surface: PASS
L7 real HOME unchanged (CODEX-01): PASS
Untrusted warning: none observed (mcp list / debug prompt-input / exec stderr carry no warning; the only trust messaging is the TUI prompt quoted in B4)
Trust override (-c projects trust_level) grants trust: no (A3: `[]` under the override; the same key written by the real TUI trust loads the layer, B6)
A1 local command form shell-expanded: yes (B5: a1-local 2026-09-19T17:02:54Z /private/tmp/07-live/trusted)
A1 global quoted command form runs: yes (C3: a1-global 2026-09-19T17:11:22Z /private/tmp/07-live/trusted, single-quoted absolute path containing a space)
A2 AGENTS.md trust-gated in a real session: no (B8: "## CodeGraph" injected in the untrusted read-only exec session, 2 vs 2)
D-16 project .agents/skills trust-gated: no (B8: "- codegraph: Use when" listed in the untrusted session, 2 vs 2)
D-15 .codex/skills read: yes (B7: r0 = trusted/.codex/skills lists cgprobe-dotcodex; also listed untrusted, B8)
D-15 CODEX_HOME/skills read: yes (B7: r1 = home/.codex/skills lists cgprobe-codexhome)
D-17 skill description as listed: "Use when asked where X is defined, how Y works, what calls X, or what changing X breaks in a .codegraph/ repo." (untruncated, byte-identical to SKILL.md frontmatter; B7)
Hooks.json runs behind features.hooks: yes (B5: marker 1->2 with hooks on, 1->1 with -c features.hooks=false; `hooks stable true` in features list)
PreToolUse stdin fields (main thread): cwd, hook_event_name, model, permission_mode, session_id, tool_input, tool_name, tool_use_id, transcript_path, turn_id; tool_name "Bash"; tool_input.command is a string; no agent_id/agent_type on the main thread (B5)
CODEX-01 verdict: PASS

## CODEX-05 and CODEX-06 (after the scope flip and the nudge)

### Method

Per D-02/D-04, this evidence is gathered by the **orchestrator**, never by an executor
subagent — a backgrounded subagent cannot reliably drive another pane's TTY (the 05-06/06-06
precedent, carried into CODEX-01 above). Task 1 (this section's scaffold) is executor work: it
builds the scratch, runs the real HEAD-binary installs, plants the A4/D-23 probe, and writes the
16-line PENDING skeleton below — nothing here is a live-session claim yet. Tasks 2 and 3 are
ORCHESTRATOR work in a sibling Herdr pane against a real Codex TUI plus scripted `codex exec
--json -C <repo>` runs, all under the scratch environment `HOME=$S2/home CODEX_HOME=$S2/home/.codex`
(plus `XDG_CONFIG_HOME=$S2/home/.config` for codegraph runs).

Evidence is the session JSONL under `$S2/home/.codex/sessions/YYYY/MM/DD/*.jsonl` — never the
model's own paraphrase of what it saw. Fire counting for L6 searches each session's JSONL for the
pinned nudge substring `returns the matching symbols' source and call paths`
(`additionalContext` delivery), with a positive control (the first expected fire) always shown
before any zero-count claim. A4's evidence is the probe's captured **raw PreToolUse stdin**
(`$S2/probes/a4-*.json`), not the model's report of it. Any hook error or block found in a
session's JSONL or the TUI output is attributed to a specific hook by its exact `command` string
in that repo's `hooks.json` — never assumed to be codegraph's guard without that match.

**Negative-space rule (STATE standing rule, carried from CODEX-01).** A transcript or listing grep
is a claim about the grep. Every absence claim (0 fires, no hook error, untrusted-hook-skipped)
must be backed by a search first shown to FIND the thing when present in a positive-control case.

**Never patch on a FAIL.** Per this plan's prohibitions, a FAIL or an A4 `no`/`not observed` is
recorded and escalated as `Maintainer decision: …`, never silently worked around.

### Pre-flight

Recorded 2026-09-19 (read-only; nothing under the real `$HOME` written — the only real-HOME
operations are the four `shasum -a 256` lines below):

```
$ codex --version
codex-cli 0.155.0

$ shasum -a 256 ~/.codex/config.toml
b4077b8a786230575935c44cfa85c17da7ca2bf3f1386675c8c6b22d322dbb17  /Users/sean/.codex/config.toml

$ shasum -a 256 ~/.codex/hooks.json
e0ad2381a6b1bf7933438d7946914737be14ca398ae41090298183c21fb49a99  /Users/sean/.codex/hooks.json

$ shasum -a 256 ~/.codex/AGENTS.md
54e268bda66adfb9c0d17a0eb73235a452f390b68d83ba1eea1dceada5379982  /Users/sean/.codex/AGENTS.md

$ shasum -a 256 ~/.agents/skills/codegraph/SKILL.md
e711379d68094bffbd5d997cccd172bb43ec3c8fa9276e06c67a59b4ce647644  /Users/sean/.agents/skills/codegraph/SKILL.md
```

All four values are identical to the CODEX-01 pre-flight record above — nothing under the real
`$HOME` changed between plans.

```
$ echo "$TMPDIR"
/var/folders/_b/3hyf5qvs62q0wh2vyh856z580000gn/T/

$ id -u
501
```

tmux: not installed — local tmux evidence skipped by maintainer decision 2026-09-19 (#75). The
`Picker tmux re-run after flip` line below is left `PENDING` for the orchestrator to fill with the
not-run decision; follow-up tracked as GitHub issue #75.

### Scaffold

Built under `$S2=$(cd /tmp && pwd -P)/07-live2` (the real, non-symlinked `/private/tmp/07-live2`
on this macOS host). Nothing under the real `$HOME` was written.

```bash
S2=$(cd /tmp && pwd -P)/07-live2
rm -rf "$S2"; mkdir -p "$S2/home/.codex" "$S2/probes" "$S2/transcripts"
GOTOOLCHAIN=go1.26.6 go build -o "$S2/codegraph" ./cmd/codegraph
ln -s "$HOME/.codex/auth.json" "$S2/home/.codex/auth.json"   # symlink, never a copy (D-02)

for n in indexed unindexed bare; do
  mkdir -p "$S2/$n"
  cp -R internal/indexer/testdata/gofixture/. "$S2/$n/"
  git -C "$S2/$n" init -q
done
CODEGRAPH_NO_WATCH=1 "$S2/codegraph" init "$S2/indexed"   # files=4 nodes=20 edges=22

for n in indexed unindexed; do
  (cd "$S2/$n" && HOME="$S2/home" CODEX_HOME="$S2/home/.codex" XDG_CONFIG_HOME="$S2/home/.config" \
    "$S2/codegraph" install --target codex --location local --pretool-nudge --yes)
  # created: .codex/config.toml, AGENTS.md, .agents/skills/codegraph/SKILL.md,
  #          .agents/skills/codegraph/.codegraph-manifest.json,
  #          .codex/hooks/codegraph-pretooluse.sh, .codex/hooks.json
  # note: Codex loads this project's MCP server only once trusted …
  # note: Codex skips a new or changed hook until you trust it — open /hooks …
done

(cd "$S2/bare" && HOME="$S2/home" CODEX_HOME="$S2/home/.codex" XDG_CONFIG_HOME="$S2/home/.config" \
  "$S2/codegraph" install --target codex --location global --yes)
# created: $S2/home/.codex/config.toml, $S2/home/.codex/AGENTS.md,
#          $S2/home/.agents/skills/codegraph/SKILL.md, .codegraph-manifest.json
# (no nudge requested globally — no $S2/home/.codex/hooks.json written)

# A4/D-23 probe, appended LAST via jq, after codegraph's own group:
cat > "$S2/probes/a4-probe.sh" <<'SH'
#!/bin/sh
cat > "$S2/probes/a4-$(date +%s)-$$.json"
exit 0
SH
chmod 0755 "$S2/probes/a4-probe.sh"
for n in indexed unindexed; do
  jq --arg cmd "'$S2/probes/a4-probe.sh'" \
    '.hooks.PreToolUse += [{"matcher":"^Bash$","hooks":[{"type":"command","command":$cmd,"timeout":5}]}]' \
    "$S2/$n/.codex/hooks.json" > "$S2/$n/.codex/hooks.json.tmp" && mv "$S2/$n/.codex/hooks.json.tmp" "$S2/$n/.codex/hooks.json"
done
```

Positive assertions (verified by this plan's own `<verify>` gate, re-stated here):

```
-rwxr-xr-x  $S2/codegraph
lrwxr-xr-x  $S2/home/.codex/auth.json -> /Users/sean/.codex/auth.json
$S2/indexed/.codegraph:    dir (indexed, files=4 nodes=20 edges=22)
$S2/unindexed/.codegraph:  absent
$S2/bare/.codegraph:       absent
$S2/bare/.codex:           absent
$S2/indexed/.codex/hooks/codegraph-pretooluse.sh    present, mode 0755, codegraph_bin='/private/tmp/07-live2/codegraph'
$S2/unindexed/.codex/hooks/codegraph-pretooluse.sh  present, mode 0755, codegraph_bin='/private/tmp/07-live2/codegraph'
$S2/indexed/.codex/hooks.json    .hooks.PreToolUse[0] = codegraph group (command == "$(git rev-parse --show-toplevel)/.codex/hooks/codegraph-pretooluse.sh")
$S2/indexed/.codex/hooks.json    .hooks.PreToolUse[1] = probe group (command matches a4-probe\.sh), length == 2
$S2/unindexed/.codex/hooks.json  same two-group shape (codegraph first, probe last)
$S2/home/.codex/config.toml      exactly 1 [mcp_servers.codegraph] table
$S2/home/.codex/hooks.json       absent (no global nudge requested)
$S2/indexed/AGENTS.md             contains <!-- CODEGRAPH_START --> (1 match)
$S2/indexed/.agents/skills/codegraph/SKILL.md   present
```

### Protocol L1/L5 (Task 2) — ORCHESTRATOR

1. Clear stale sentinels: `rm -rf "${TMPDIR}/codegraph-nudge-$(id -u)"`.
2. Start the Codex TUI in a sibling Herdr pane in `$S2/indexed`; accept the folder trust prompt;
   DO NOT open `/hooks` yet; quit. Same for `$S2/unindexed`.
3. Untrusted-hook check: `codex exec --json -C "$S2/indexed" 'Run exactly this shell command and
   nothing else: rg -n Alpha .'` → record whether the pinned nudge substring appears in that
   session's JSONL and whether any `$S2/probes/a4-*.json` was written for the turn (the probe is
   the positive control that would show a hook ran even though it isn't trusted yet).
4. Reopen the TUI in each repo, run `/hooks`, trust codegraph's group and the probe group.
5. L1 post-flip: `codex mcp list --json` in `$S2/bare` (global layer only) and in `$S2/indexed`
   (project layer, trusted); confirm `[mcp_servers.codegraph]` in both `$S2/home/.codex/config.toml`
   and `$S2/indexed/.codex/config.toml`.
6. L5: `codex exec --json -C "$S2/indexed" 'Where is the function Alpha defined in this
   repository, and what calls it? Give file:line references.'` (never names codegraph). From the
   session JSONL: confirm the injected skill list includes `codegraph` and the agent made a call
   to the codegraph MCP server's tool or ran `codegraph explore`.
7. Record negative space (every grep/rg/find the L5 session ran) and uptake (what the agent did
   first, whether a nudge fire preceded the codegraph call).

### Protocol L6/A4/D-23/uninstall/tmux (Task 3) — ORCHESTRATOR

1. L6 in ONE session (the Herdr TUI in `$S2/indexed`): `rg -n Alpha .` (expect one fire),
   immediately `grep -rn Beta .` (expect none), after at least 65 s `find . -name '*.go'` (expect
   one fire). Fill the Fire log table. Un-indexed control in `$S2/unindexed` (`rg -n Alpha .`):
   expect 0 fires while a new `$S2/probes/a4-*.json` proves the hooks ran. Search both JSONLs and
   the TUI output for a hook error or block attributable to `codegraph-pretooluse.sh`.
2. A4: `Use a subagent to run exactly this shell command and report its output: rg -n Gamma .` in
   the indexed TUI; inspect the probe's raw stdin files for the subagent's call.
3. D-23: `codegraph install --target codex --location local --pretool-nudge=false --yes` in
   `$S2/indexed` (the probe group becomes first); reopen `/hooks` and record whether later foreign
   hooks are re-flagged.
4. Uninstall: `codegraph uninstall --target codex --location local --yes` in `$S2/indexed` and
   `--location global`; confirm the scratch repo is left clean (probe group intact).
5. `GOTOOLCHAIN=go1.26.6 task test:tmux` at HEAD — **SKIPPED by maintainer decision 2026-09-19
   (#75):** tmux is retired on this machine (replaced by herdr); local tmux evidence for FIX-03 is
   deliberately not gathered here. The orchestrator records the not-run decision on the `Picker
   tmux re-run after flip` line rather than a PASS/FAIL, with follow-up tracked as GitHub issue
   #75.
6. L7: re-run the four `shasum -a 256` lines against the real `$HOME`.
7. Write `CODEX-05 live verdict` and `CODEX-06 verdict` per the locked pass bar (D-06), or FAIL
   plus `Maintainer decision: …`.

### Fire log

| key | fire timestamps | gaps |
|-----|------------------|------|
| _(empty — filled by Task 2/3)_ | | |

### Task 2 evidence (orchestrator, 2026-09-19, codex-cli 0.155.0)

All runs used `HOME=/private/tmp/07-live2/home CODEX_HOME=/private/tmp/07-live2/home/.codex`. Transcripts are in `/private/tmp/07-live2/transcripts/`. Stale sentinels were cleared first (`rm -rf "${TMPDIR}codegraph-nudge-501"`). Two Codex "Update available 0.155.0 -> 0.155.1" prompts were declined with "Skip" (one accidental "Update now" was interrupted with Ctrl-C before the cask swapped; `codex --version` stayed 0.155.0 throughout).

**B1: folder trust, hooks left untrusted.** In both `indexed` and `unindexed` the TUI trust prompt was accepted and the hook prompt answered "3. Continue without trusting (hooks won't run)":

```
  Hooks need review
  2 hooks are new or changed.
  Hooks can run outside the sandbox after you trust them.
› 1. Review hooks
  2. Trust all and continue
  3. Continue without trusting (hooks won't run)
[projects."/private/tmp/07-live2/indexed"]     trust_level = "trusted"
[projects."/private/tmp/07-live2/unindexed"]   trust_level = "trusted"
(no [hooks.state] table yet)
```

**B2: untrusted-hook check** (`B2-untrusted-hooks.jsonl`, rollout `01a0bb82-161e-…`)

```
$ codex exec --json -C $S2/indexed 'Run exactly this shell command and nothing else: rg -n Alpha .' < /dev/null
/bin/zsh -lc 'rg -n Alpha .' -> exit 0
pinned nudge substring in the rollout: 0
$S2/probes before: a4-probe.sh    after: a4-probe.sh      (no a4-*.json written: the probe hook did not run either)
stderr: (nothing about hooks)
```

Codex reported nothing about the skipped hooks in `exec`. The only statement is the TUI's "hooks won't run" option text. The positive control for the probe's absence is B5 below (two `a4-*.json` files per Bash call once trusted).

**B3: `/hooks` trust** (`B3-indexed-hooks-{before,after}.txt`)

```
  PreToolUse            2           0           2           Before a tool executes
  [!] Hook 1 · new   Command   "$(git rev-parse --show-toplevel)/.codex/hooks/codegraph-pretooluse.sh"   Trust  New hook - review required
  [!] Hook 2 · new   Command   '/private/tmp/07-live2/probes/a4-probe.sh'                                Trust  New hook - review required
  -> (t, t) ->  [x] Hook 1   [x] Hook 2   Trust  Trusted
[hooks.state."/private/tmp/07-live2/indexed/.codex/hooks.json:pre_tool_use:0:0"]   trusted_hash = "sha256:2234081d…"
[hooks.state."/private/tmp/07-live2/indexed/.codex/hooks.json:pre_tool_use:1:0"]   trusted_hash = "sha256:07c31625…"
[hooks.state."/private/tmp/07-live2/unindexed/.codex/hooks.json:pre_tool_use:0:0"] / :1:0   (same, for unindexed)
```

**B4: L1 post-flip** (`B4-{bare,indexed}-mcp.json`)

```
bare     [{"name":"codegraph","enabled":true,"command":"/private/tmp/07-live2/codegraph","args":["serve","--mcp"]}]
indexed  [{"name":"codegraph","enabled":true,"command":"/private/tmp/07-live2/codegraph","args":["serve","--mcp"]}]
$S2/home/.codex/config.toml:      [mcp_servers.codegraph] command = "/private/tmp/07-live2/codegraph" args = ["serve", "--mcp"]
$S2/indexed/.codex/config.toml:   [mcp_servers.codegraph] command = "/private/tmp/07-live2/codegraph" args = ["serve", "--mcp"]
```

Both layers were written by the HEAD binary from the same ExecPath, so the merged entry is identical from both cwds. That the project layer is loaded when trusted was shown with a distinct binary path in CODEX-01 (B6). "Both scopes" is evidenced by the two config files plus `mcp list` from two cwds, never by a scope field (adjacency backstop).

**B5: L5 uptake** (`B5-L5.jsonl`, rollout `01a0bb83-dec3-…`). Prompt: `Where is the function Alpha defined in this repository, and what calls it? Give file:line references.`

```
injected context: "- codegraph: Use when" x2 (skill listed); "codegraph_explore" x4 (the MCP tool definition loaded)
agent_message: I'm using the CodeGraph skill because this repository is indexed and the question asks for a definit…
command_execution: /bin/zsh -lc 'cat /private/tmp/07-live2/indexed/.agents/skills/codegraph/SKILL.md && codegraph explore "Where is function Alpha defined, and what directly or transitively calls it? Include file and line references."'
command_execution: /bin/zsh -lc 'codegraph callers Alpha && codegraph callers Run'
agent_message: `Alpha` is defined at pkga/pkga.go:13 …
mcp_tool_call items: none (the agent used the CLI path, not the MCP tool)
grep/rg/find first-word commands: 0
pinned nudge substring in the rollout: 0 (no Bash call qualified: neither command starts with grep/rg/find)
probe files written this session: a4-1789852382-61975.json, a4-1789852386-62280.json (hooks ran on both calls; tool_name "Bash", no agent_id)
```

Positive control for the searches: the skill-listing search found `- codegraph: Use when` x2 in this rollout and x2 in the B2 rollout; the pinned-substring search finds the string in `internal/nudge/text.go` and in the 06-06 fire logs.

### CODEX-05/06 verdicts

L1 post-flip both scopes (codegraph-installed): PASS
L5 fresh session reaches for codegraph unprompted: PASS
L6 nudge fires once then cools down; un-indexed 0 fires: PENDING
L7 real HOME unchanged (CODEX-05/06): PENDING
Untrusted hook skipped before /hooks trust: yes (B2: 0 pinned substrings and no a4-*.json for the turn; B5 wrote two a4-*.json per session once trusted)
A4 PreToolUse stdin carries a subagent id: PENDING
tool_input.command type: PENDING
D-23 removing our group re-flags later foreign hooks: PENDING
Negative space (grep/rg/find runs in the L5 session): 0 — the only commands were `cat …SKILL.md && codegraph explore "…"` and `codegraph callers Alpha && codegraph callers Run`
Matched Bash search calls: PENDING
Fires: PENDING
Uptake: the agent opened with "I'm using the CodeGraph skill because this repository is indexed", read SKILL.md and ran `codegraph explore` in its FIRST Bash call, then `codegraph callers`; no nudge fire preceded the codegraph call (0 fires: no qualifying search was ever run); the MCP tool was loaded but the CLI path was chosen
Uninstall leaves the scratch repo clean: PENDING
Picker tmux re-run after flip: PENDING
CODEX-05 live verdict: PENDING
CODEX-06 verdict: PENDING
