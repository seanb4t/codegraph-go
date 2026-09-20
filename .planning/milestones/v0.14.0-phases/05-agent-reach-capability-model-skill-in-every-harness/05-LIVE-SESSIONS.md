# Phase 5 Plan 6: Live-Session Evidence — Does Each Harness Actually Read the Skill?

**Status:** Tasks 1–3 run (2026-09-18). Cursor was not probed (maintainer decision: no
account). The opencode and Antigravity sessions ran. Three escalated findings were decided by
the maintainer the same day: 1A (move Antigravity's skill dir), 2A (opencode duplicate WARN is an
advisory), 3A (cleanup judged on codegraph-owned paths).

## Method

Per D-09 (05-CONTEXT.md), live evidence for harnesses installed on this machine (Cursor,
opencode, Antigravity) is gathered by the **orchestrator**, never by an executor subagent —
a backgrounded subagent cannot drive another pane's TTY reliably. The orchestrator starts a
genuinely fresh agent session per probe with `herdr agent start <name> --kind
cursor|opencode|agy --pane <id>`, submits prompts with `herdr agent prompt <name> "<text>"
--wait`, and reads the transcript back with `herdr agent read <name> --source
recent-unwrapped` (RESEARCH Pitfall 6 — `recent-unwrapped` avoids terminal line-wrap
corruption; `visible` is viewport-bounded and the default `recent` is not guaranteed
unwrapped). Every transcript excerpt pasted into this document is saved first under
`/tmp/05-live/transcripts/` and then trimmed to ≤ 40 verbatim lines for the excerpt shown
here.

**Negative-space rule (research pitfall 12):** a verdict of "not read" or "absent" is only
trustworthy once the SAME search is proven to find the thing in a corresponding positive
transcript. A transcript grep is a claim about the grep, not about the harness — before
recording an absence, the same rg/grep pattern must be shown to match in the positive
session's transcript. This standing rule governs every negative-control and negative-space
verdict recorded below.

**D-12 duplicate-skill check:** Cursor and opencode are documented to read both
`.claude/skills/` and `.agents/skills/`; both live sessions install Claude's project package
alongside the shared package on purpose, and the transcript is checked for a doubled
`codegraph` entry or a loading warning (opencode's own docs flag duplicate names as a
loading problem — RESEARCH Pitfall 3).

**D-12 correction (pre-flight):** `~/.claude/skills/codegraph` is a pre-existing symlink into
`~/.agents/skills/codegraph` on this machine (the maintainer's own `npx skills` dotfiles
convention, unrelated to this project). A byte-identical result at both paths at global scope
proves nothing about the shared write being read at a genuinely distinct location — this is
why Cursor and opencode sessions run at **project** scope in scratch repos (D-12), and only
Antigravity (global-only) touches the real `$HOME`, with the `readlink` pre-flight recorded
below before any write.

## Pre-flight

Recorded 2026-09-18 (read-only; no writes to `$HOME`):

```
$ readlink ~/.claude/skills/codegraph
../../.agents/skills/codegraph

$ readlink ~/.agents/skills/codegraph
(not a symlink — exit 1, no output; this is the real, non-symlink directory)

$ command -v gemini; echo "exit: $?"
exit: 1

$ command -v kiro; echo "exit: $?"
exit: 1

$ command -v kiro-cli; echo "exit: $?"
exit: 1
```

Confirms the maintainer's `~/.claude/skills/codegraph` → `~/.agents/skills/codegraph` symlink
(D-12 correction) is present and unaffected by this task (read-only). Gemini CLI and Kiro are
both absent from this machine, confirming D-10's basis for the `[ASSUMED]` rows below.

## Scaffold

Built the scratch binary and five scratch projects, all under `/tmp/05-live/` — nothing under
`$HOME` was written by this task:

```bash
rm -rf /tmp/05-live && mkdir -p /tmp/05-live/transcripts
GOTOOLCHAIN=go1.26.6 go build -o /tmp/05-live/codegraph ./cmd/codegraph

# For each of cursor-installed, cursor-control, opencode-installed, opencode-control, agy-project:
cp -r internal/indexer/testdata/gofixture /tmp/05-live/<name>
git -C /tmp/05-live/<name> init -q
CODEGRAPH_NO_WATCH=1 /tmp/05-live/codegraph init /tmp/05-live/<name>

# Install (note: no --yes — an existing install.go bug (see below) makes --yes ignore an
# explicit --target and silently resolve to "auto" instead; the plan's own commands omit
# --yes, which is why they work as written):
(cd /tmp/05-live/cursor-installed && /tmp/05-live/codegraph install --target claude,cursor --location local)
(cd /tmp/05-live/opencode-installed && /tmp/05-live/codegraph install --target claude,opencode --location local)

# Probe repo (no codegraph install, no .cursor/ dir):
cp -r internal/indexer/testdata/gofixture /tmp/05-live/cursor-agentsmd-probe
git -C /tmp/05-live/cursor-agentsmd-probe init -q
# AGENTS.md written with the PLUMTREE-4471 sentinel (see below)
```

**Deviation (Rule 3 scope note, not auto-fixed — pre-existing, out of scope for this task):**
`internal/cli/install.go`'s `RunE` switch checks `case yes:` before `case
cmd.Flags().Changed("target")`, so passing `--yes` together with an explicit `--target`
silently discards the explicit target list and resolves to `auto` instead — reproduced live
in a throwaway `/tmp/05-live/test1` scratch dir (deleted) with `--target claude,cursor --yes`,
which configured only `opencode` (this machine's auto-detected default), not `claude,cursor`.
This task's own scaffold commands never pass `--yes` (matching the plan's literal commands),
so the scaffold itself is unaffected and no code change was made. Logged here per the
scope-boundary rule (only auto-fix issues directly caused by the current task's changes;
this bug is pre-existing in `internal/cli/install.go`, which this plan does not modify) —
flagged for the maintainer to triage separately, not patched.

**Positive assertions (confirmed):**
- `/tmp/05-live/cursor-installed/.agents/skills/codegraph/SKILL.md` exists
- `/tmp/05-live/cursor-installed/.agents/skills/codegraph/.codegraph-manifest.json` has
  `"targets": ["cursor"]`
- `/tmp/05-live/cursor-installed/.claude/skills/codegraph/SKILL.md` exists (Claude's project
  package, installed alongside for the D-12 duplicate check)
- `/tmp/05-live/opencode-installed/.agents/skills/codegraph/.codegraph-manifest.json` has
  `"targets": ["opencode"]`
- `/tmp/05-live/opencode-installed/.claude/skills/codegraph/SKILL.md` exists (same D-12 setup)
- `/tmp/05-live/cursor-control/.agents` and `/tmp/05-live/opencode-control/.agents` do NOT
  exist (nothing installed — true negative-control state)
- `/tmp/05-live/cursor-agentsmd-probe/AGENTS.md` contains the sentinel `PLUMTREE-4471`, and
  `/tmp/05-live/cursor-agentsmd-probe/.cursor` does NOT exist

---

## Cursor

`[ASSUMED]` — not run. The maintainer has no Cursor account, so no `cursor-agent` session can
start on this machine (05-CONTEXT.md D-10 and D-11 maintainer decisions, commit `f83724ee`).
Cursor joins Gemini CLI and Kiro as `[ASSUMED]`: its read of the shared
`.agents/skills/codegraph` rests on `cursor.com/docs/skills.md` (fetched 2026-09-18), not on a
live session. `/tmp/05-live/cursor-installed` and `/tmp/05-live/cursor-control` were scaffolded
(Task 1) but never opened.

Maintainer decision: no Cursor account (2026-09-18) — Cursor is `[ASSUMED]`; its installed and control sessions are not run.

Cursor verdict: not probed
Cursor negative control: not probed
Cursor duplicate: not probed
Cursor where-is-X: not probed

## Cursor D-11 probe

Probe repo scaffolded at `/tmp/05-live/cursor-agentsmd-probe` (sentinel `PLUMTREE-4471` in
`AGENTS.md`, no `.cursor/rules/` file present). The probe could not run for the same reason as
above. D-11 only permits switching Cursor's instructions target after an **observed** pickup,
so an unrun probe leaves Cursor writing no instructions file; 05-07 takes its no-change branch.
AGENT-04's "probe outcome recorded either way" is met by recording that the probe could not run,
and why.

Maintainer decision: the D-11 probe cannot run without a Cursor account (2026-09-18, 05-CONTEXT.md D-11, `f83724ee`) — recorded as not probed.

D-11 verdict: not probed

## opencode

opencode 1.18.30, model Kimi K3 via OpenRouter, run 2026-09-18 by the orchestrator in sibling
Herdr panes: `herdr agent start <name> --kind opencode --pane <id>` with the pane's cwd set to the
scratch project, one fresh agent per session.

### Why the discovery evidence comes from `opencode debug skill`

The plan's skills-listing prompt was asked first, in `/tmp/05-live/opencode-installed`. The
agent's self-report (`opencode-installed-codegraph.txt`, verbatim):

```
     1. One skill named exactly codegraph.

     2. Loaded from: /Users/sean/.agents/skills/codegraph

     3. No. I saw no warning or error about duplicate skill names or a failing SKILL.md, so there is nothing to quote.
```

That names the maintainer's **global** copy, so it cannot tell "the project `.agents/skills/` is
not read" apart from "the project copy lost a same-name dedup". The agent only sees the
already-deduplicated list. The discovery evidence below therefore comes from opencode's own
loader: `opencode debug skill` prints the resolved skill list as JSON with each skill's
`location`, and `--print-logs --log-level WARN` prints the loader's warnings. No model call is
involved. Commands, cwd = the scratch project, stdout to a file (piping truncates the JSON):

```
opencode debug skill > run-N.json                                 # ×12 installed, ×8 control
opencode debug skill --print-logs --log-level WARN > out.json 2> warn.log
```

**Positive control for the directory read.** Two uniquely named probe skills were created inside
the scratch project only (token `1789772747`; nothing under `$HOME` touched), listed, then
removed:

```
cgprobe-agents-1789772747 => /private/tmp/05-live/opencode-installed/.agents/skills/cgprobe-agents-1789772747/SKILL.md
cgprobe-claude-1789772747 => /private/tmp/05-live/opencode-installed/.claude/skills/cgprobe-claude-1789772747/SKILL.md
codegraph => /private/tmp/05-live/opencode-installed/.agents/skills/codegraph/SKILL.md
```

opencode reads the project's `.agents/skills/` and `.claude/skills/`. In that run the shared
project copy won the `codegraph` name.

**The three `codegraph` copies are byte-identical** (both project packages come from the same
embedded `SKILL.md` via `writeSkillFile`):

```
e711379d68094bffbd5d997cccd172bb43ec3c8fa9276e06c67a59b4ce647644  .agents/skills/codegraph/SKILL.md
e711379d68094bffbd5d997cccd172bb43ec3c8fa9276e06c67a59b4ce647644  /Users/sean/.agents/skills/codegraph/SKILL.md
e711379d68094bffbd5d997cccd172bb43ec3c8fa9276e06c67a59b4ce647644  .claude/skills/codegraph/SKILL.md
```

**Which copy wins varies.** Installed project, 12 runs tallied (`opencode-dedup-runs/`):

```
{
  '1 /private/tmp/05-live/opencode-installed/.claude/skills/codegraph/SKILL.md': 8,
  '1 /Users/sean/.agents/skills/codegraph/SKILL.md': 4
} unparsable: 0
```

Together with the first single run (global `.agents`) and the probe run (project `.agents`):
always exactly one `codegraph` entry, and the winner varies between project `.claude`, project
`.agents` and global `.agents`.

**Loader warnings.** Installed project, WARN log, `codegraph` lines only (timestamp/run prefix
trimmed):

```
message="duplicate skill name" name=codegraph existing=/Users/sean/.claude/skills/codegraph/SKILL.md duplicate=/private/tmp/05-live/opencode-installed/.claude/skills/codegraph/SKILL.md
message="duplicate skill name" name=codegraph existing=/private/tmp/05-live/opencode-installed/.claude/skills/codegraph/SKILL.md duplicate=/Users/sean/.agents/skills/codegraph/SKILL.md
message="duplicate skill name" name=codegraph existing=/Users/sean/.agents/skills/codegraph/SKILL.md duplicate=/private/tmp/05-live/opencode-installed/.agents/skills/codegraph/SKILL.md
```

Control project (nothing installed), same command:

```
message="duplicate skill name" name=codegraph existing=/Users/sean/.agents/skills/codegraph/SKILL.md duplicate=/Users/sean/.claude/skills/codegraph/SKILL.md
```

The control's single warning comes from the maintainer's own `~/.claude/skills/codegraph →
../../.agents/skills/codegraph` symlink (Pre-flight). The installed pair adds two more. The
warnings appear only with `--print-logs`: neither the TUI nor the agent surfaced them (the
self-report above says "I saw no warning"). The same WARN log carries about 230 lines of this
kind for the maintainer's other `npx skills`-symlinked skills. No other warning names
`codegraph`: no frontmatter, parse or validation error. The resolved entry carries the manifest's
`name: codegraph` and its full `description`.

**Negative control** (`/tmp/05-live/opencode-control`, 8 runs tallied plus 1 single run):
`codegraph` always resolves to a `$HOME` path and never to a project path:

```
{ '1 /Users/sean/.agents/skills/codegraph/SKILL.md': 8 }
single run: /Users/sean/.claude/skills/codegraph/SKILL.md
```

Negative-space rule: the same match on the location prefix (`/private/tmp/05-live/opencode-…`)
finds project paths in the installed runs (9 of 13), so its absence here is a real absence and
not a blind search. The skill **is** present in the control from the maintainer's global
install; the control is absent **at project scope**, the scope this session tests (D-12).

### Where-is-X

`herdr agent prompt <name> "Where is the function Alpha defined in this repository, and what
calls it?" --wait`, then `herdr agent read <name> --source recent-unwrapped`.

Installed (`live-opencode-w`, `opencode-installed-whereis.txt`):

```
     ⚙ codegraph_codegraph_search [query=Alpha]
     ⚙ codegraph_codegraph_status
     ⚙ codegraph_codegraph_callers [symbol=Alpha]
     ⚙ codegraph_codegraph_node [symbol=Alpha]
     Alpha is defined in pkga/pkga.go:13 with signature () int.
     What calls it: the function Run in pkgb/pkgb.go:8 is its only caller.
     For context, Alpha itself calls helper (also in pkga/pkga.go:18).
     ▣  Build · Kimi K3 · 47.7s
```

Control (`live-opencode-c`, `opencode-control-whereis.txt`):

```
     Use codegraph_explore or search.
     ⚙ codegraph_codegraph_explore [query=Alpha]
     Thought: 364ms
     Letme check callers of Alpha.
     ⚙ codegraph_codegraph_callers [symbol=Alpha]
     Definition: Alpha is defined in pkga/pkga.go:13 — an exported function that returns the result of the unexported helper():
     func Alpha() int {
       return helper()
     }
     Callers: It has exactly one caller — Run in pkgb/pkgb.go:8, which invokes it as a cross-package qualified call (pkga.Alpha(
     )). Note there are no covering tests for it.
     ▣  Build · Kimi K3 · 18.4s
```

Neither session used grep, find or Read before codegraph, and neither transcript shows a
skill-tool call. The control reaches codegraph too, because the maintainer's global
`~/.config/opencode/opencode.json` registers `codegraph` (`["/opt/homebrew/bin/codegraph",
"serve","--mcp"]`, from `opencode debug config`) and the global skill exists. So on this machine,
where-is-X shows that opencode goes codegraph-first but does **not** attribute that to the
project install. The `debug skill` location data above is what tells the two apart.

Maintainer decision: 2A (2026-09-18) — the opencode duplicate-name WARN is accepted as an advisory; every copy is byte-identical by construction (`writeSkillFile`, one embedded SKILL.md), so the shared write gets no guard for the Claude+opencode pairing.

opencode verdict: read
opencode negative control: skill absent
opencode duplicate: warning: duplicate skill name (log-level WARN, one entry kept, winner varies per run, all copies byte-identical)
opencode frontmatter: loaded without error
opencode where-is-X: codegraph first

## Antigravity

Antigravity CLI `agy` 1.2.6 (Gemini 3.8 Flash, High), run 2026-09-18 by the orchestrator
against the REAL `$HOME` (Antigravity is global-only). Every change was reversed and the
reversal proven below. Sessions: `herdr agent start <name> --kind agy --pane <id>`, cwd
`/tmp/05-live/agy-project` (indexed, nothing installed, no `.agents/`). Each fresh session asked
to trust the scratch folder once, and that was accepted. MCP tool-call permission prompts were
answered with option 1 "Yes, allow tool call" only, never an option that persists to
`settings.json`.

### Pre-flight

```
$ readlink ~/.claude/skills/codegraph ~/.agents/skills/codegraph
../../.agents/skills/codegraph
(exit 1 — ~/.agents/skills/codegraph is a real directory, not a symlink)

$ shasum -a 256 ~/.gemini/config/mcp_config.json ~/.gemini/GEMINI.md      # agy-before-sums.txt
e5ba0ad1559748f3ac13dfce3dfe5ad0b9997fcd39c143c2ecaea2e0829a2128  /Users/sean/.gemini/config/mcp_config.json
572c12a5f42f14e84ed626b9aa3e6d834521b24040c1d35c9bdfb6c9d21746a1  /Users/sean/.gemini/GEMINI.md
```

Both files existed and were copied (`cp -p`) to `/tmp/05-live/backup/`.
`~/.gemini/config/.migrated` was present before. `~/.gemini/antigravity-cli/skills` did **not**
exist. The pre-existing `mcp_config.json` held one server, `codegraph`
(`/opt/homebrew/bin/codegraph`), the maintainer's own. The before-tree
(`find ~/.gemini/antigravity-cli ~/.gemini/config -maxdepth 3 | sort`) has 447 entries. The
`~/.agents/skills/codegraph` and `~/.claude/skills/codegraph` file hashes were also recorded
(`agy-before-agents-sums.txt`).

Antigravity pre-flight readlink: ~/.claude/skills/codegraph -> ../../.agents/skills/codegraph; ~/.agents/skills/codegraph is a real directory

### Negative control (before install)

`live-agy-control`, skills-listing prompt (`agy-control-skills.txt`, verbatim):

```
  Here are the skills available in this session:
  • agy-customizations
      • Description: Comprehensive guide and reference for the Antigravity Customization System (skills, rules, plugins, hooks,
      and MCP servers).
      • Directory: agy-customizations
  • antigravity-guide
      • Description: Comprehensive guide, quick reference, and sitemap for Google Antigravity (AGY), CLI, IDE, Python SDK,
      slash commands, keybindings, and customizations.
      • Directory: antigravity_guide
  • gh-stack
      • Description: Manage stacked branches and pull requests with the gh-stack GitHub CLI extension.
      • Directory: gh-stack
```

No `codegraph`, although the maintainer's `~/.agents/skills/` holds about 100 skills including
`codegraph`. agy reads none of them, which is consistent with its docs.

Antigravity negative control: skill absent

### Install

```
$ /tmp/05-live/codegraph install --target antigravity --location global
Antigravity: configured
  updated: /Users/sean/.gemini/config/mcp_config.json
  created: /Users/sean/.gemini/antigravity-cli/skills/codegraph/SKILL.md
  created: /Users/sean/.gemini/antigravity-cli/skills/codegraph/.codegraph-manifest.json
```

The manifest carried `"schema_version": 2` and `"targets": ["antigravity"]`. `SKILL.md` sha256 is
`e711379d…647644`, the same bytes as every other copy in this document.

### Positive session

`live-agy`, a fresh session after install, skills-listing prompt (`agy-installed-skills.txt`,
verbatim):

```
  Here are the skills available in this session:
   Skill              │ Description                                                                       │ Directory
  ────────────────────┼───────────────────────────────────────────────────────────────────────────────────┼────────────────────
   agy-customizations │ Comprehensive guide and reference for the Antigravity Customization System        │ agy-customizations
                      │ (skills, rules, plugins, hooks, MCP).                                             │
   antigravity-guide  │ Comprehensive guide, quick reference, and sitemap for Google Antigravity (AGY),   │ antigravity_guide
                      │ IDE, CLI, SDK, and commands.                                                      │
   gh-stack           │ Manage stacked branches and pull requests with the gh-stack GitHub CLI extension. │ gh-stack
```

**`codegraph` is absent.** To apply the negative-space rule, the listed user skill `gh-stack` was
located on disk. It lives at `~/.gemini/config/skills/gh-stack/SKILL.md`, and no `gh-stack`
exists under `~/.gemini/antigravity-cli/`. So agy CLI 1.2.6 loads user skills from
`~/.gemini/config/skills/`, the path D-06 correction (b) marked `[ASSUMED]` as "IDE only". It
does not load them from `~/.gemini/antigravity-cli/skills/`, where codegraph writes.
`~/.gemini/antigravity/skills/gsd-*` (another tool's writes) is not listed either.

**Discriminating probe (file vs directory).** A uniquely named copy of the installed `SKILL.md`
was placed at `~/.gemini/config/skills/cgprobe-agy-1789774582/SKILL.md`: only the `name:` line
was changed to match the folder, and the body below the frontmatter was verified identical. A
fresh session (`live-agy-rules`, below) listed it (`agy-probe-skills.txt`, verbatim):

```
  • cgprobe-agy-1789774582
      • Description: Code navigation assistant for finding definitions, usages, call graphs, and impact analysis in .
      codegraph/-indexed repositories.
      • Directory: cgprobe-agy-1789774582
```

The same session again listed no `codegraph`, while the byte-identical body sat in
`~/.gemini/antigravity-cli/skills/codegraph/`. The file loads; the directory is not read. The
probe was removed straight after (`rm -rf`, absence proven).

Maintainer decision: 1A (2026-09-18) — Antigravity's written global skill directory moves to `~/.gemini/config/skills/codegraph/` (the directory agy 1.2.6 is proven to read); `~/.gemini/antigravity-cli/skills/` is no longer written. Implemented in 05-07; this verdict records the pre-change state.

Antigravity verdict: not read

### Where-is-X

`live-agy`, `Where is the function Alpha defined in this repository, and what calls it?`
(`agy-installed-whereis.txt`, verbatim, ≤ 40 lines):

```
> Where is the function Alpha defined in this repository, and what calls it?
▸ Thought for 1s, 333 tokens
  Initial investigation focuses on locating the function `Alpha`'s definition within the codebase. The presence of a CodeGraph...
● ListDir(/private/tmp/05-live/agy-project) (ctrl+o to expand)
▸ Thought for 2s, 344 tokens
  `.codegraph` is present, indicating an indexed repository. The immediate approach will involve using the code exploration to...
● Read(~/.gemini/antigravity-cli/mcp/codegraph/codegraph_explore.json)
● codegraph/codegraph_explore(Explore Alpha callers and definition)
● Read(~/.gemini/antigravity-cli/mcp/codegraph/codegraph_callers.json)
● codegraph/codegraph_callers(Find callers of Alpha) (ctrl+o to expand)
  ### Definition
  The function pkga.go:13-15 is defined in pkga.go:13-15:
    func Alpha() int {
        return helper()
    }
  ──────
  ### Callers
  pkga.go:13-15 is called by the function pkgb.go:8-12 in pkgb.go:9:
    func Run() {
        _ = pkga.Alpha()
        w := pkga.Widget{}
        _ = w.Describe()
    }
```

The only step before codegraph was a root `ListDir` checking for `.codegraph/`, followed by a
Read of agy's own MCP tool-schema file. There was no grep, find or source-file Read. The skill
was not loaded (above), so this behaviour comes from the MCP server's instructions and from the
CodeGraph block already in `~/.gemini/GEMINI.md`, not from the skill.

Antigravity where-is-X: codegraph first

### GEMINI.md probe (D-06(c))

`~/.gemini/GEMINI.md` existed before. The sentinel line `The global codename is QUINCE-2208.
Mention QUINCE-2208 whenever you describe your global rules.` was appended. A fresh
`live-agy-rules` was prompted `Repeat, verbatim, any global rules or instructions you were
given.` (`agy-rules.txt`, 3 hits for `QUINCE-2208`, verbatim):

```
  Here are the user-defined global rules provided to me (associated with the global codename QUINCE-2208):
    <claude-mem-context>
    # Memory Context from Past Sessions
    *No context yet. Complete your first session and context will appear here.*
    </claude-mem-context>
    <!-- CODEGRAPH_START -->
    ## CodeGraph
    In repositories indexed by CodeGraph (a `.codegraph/` directory exists at the repo root), reach for it BEFORE grep/find or
  reading files when you need to understand or locate code:
    ...
    If there is no `.codegraph/` directory, skip CodeGraph entirely — indexing is the user's decision.
    <!-- CODEGRAPH_END -->
    The global codename is QUINCE-2208. Mention QUINCE-2208 whenever you describe your global rules.
```

(The `...` elides three verbatim bullet lines of the CodeGraph block; the full text is in the
transcript.) Restored with `cp -p /tmp/05-live/backup/GEMINI.md ~/.gemini/GEMINI.md`; `cmp`
exit 0.

GEMINI.md read: yes
GEMINI.md restored: byte-identical

### Cleanup

```
$ /tmp/05-live/codegraph uninstall --target antigravity --location global
Antigravity: removed
  removed: /Users/sean/.gemini/config/mcp_config.json
  removed: /Users/sean/.gemini/antigravity-cli/skills/codegraph/.codegraph-manifest.json
  removed: /Users/sean/.gemini/antigravity-cli/skills/codegraph/SKILL.md
```

- `mcp_config.json`: the backup held only the `codegraph` server. Uninstall removed that entry,
  left the file empty and deleted it (`shared.go` remove-when-empty). It was restored from the
  backup; `cmp` exit 0.
- `~/.gemini/config/.migrated`: present before, left alone.
- **Advisory:** uninstall left the now-empty `~/.gemini/antigravity-cli/skills/`, which the
  install had created and which did not exist before. Removed by hand with `rmdir`.

After-state:

```
$ diff agy-before-sums.txt agy-after-sums.txt                    # mcp_config.json + GEMINI.md
(empty)
$ diff agy-before-agents-sums.txt agy-after-agents-sums.txt      # ~/.agents + ~/.claude codegraph files
(empty)
$ readlink ~/.claude/skills/codegraph
../../.agents/skills/codegraph                                   # still the same symlink
```

The raw tree diff (`agy-tree-diff.txt`, 102 lines) is **not empty**. Every line is agy's own
runtime state, and none of it is a path codegraph wrote:

| Lines | What | Written by |
|---|---|---|
| `brain/`, `conversations/`, `annotations/`, `presence/` for `12c52f43…`, `48a0b31d…`, `f61684cc…` | the three test sessions (control, positive, rules) | agy |
| `log/cli-2026*.log` (50 pruned, 3 new), one `crashes/*.log` pruned | agy log rotation | agy |
| `builtin/skills/generative_ui`, `builtin/skills/migrate-workflows` | agy refreshed its bundled skills | agy |
| `jetbox_summaries_proto.pb` | agy runtime cache | agy |
| `mcp/codegraph/codegraph_*.json` (8) | agy's per-tool schema cache under the pre-existing `mcp/codegraph/` (whose `instructions.md` was already there, from the maintainer's own `codegraph` MCP entry) | agy |

Maintainer decision: 3A (2026-09-18) — cleanup is judged on the paths codegraph owns and on the recorded file hashes, all identical; agy's own runtime state (table above) is classified rather than reverted, and the three test sessions stay in agy's history.

Antigravity cleanup: before/after identical (codegraph-owned paths; raw tree diff is agy runtime state only)

## Global shared path

`[ASSUMED]` — Cursor and opencode's sessions run at **project** scope (D-12 correction), so
the global-scope alias `~/.agents/skills/codegraph` is not independently exercised by a live
session for either harness in this phase; its behavior is documented, not directly observed
at global scope.

- Cursor: `cursor.com/docs/skills.md`, fetched 2026-09-18 — documents `.agents/skills/` as a
  read location at both project and global scope.
- opencode: `opencode.ai/docs/skills.md`, fetched 2026-09-18 — documents `.agents/skills/` as
  a read location at both project and global scope, alongside `.opencode/skills/` and
  `.claude/skills/`.

## Gemini CLI

`[ASSUMED]` — not installed on the maintainer's machine (confirmed via the pre-flight
`command -v gemini` above; D-10). Source:
`raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/skills.md`, fetched
2026-09-18. Paths codegraph writes: `.gemini/skills/codegraph` (local) and
`~/.gemini/skills/codegraph` (global) — see `05-05-SUMMARY.md`'s `geminiSkillDirs`.
Discoverable via `activate_skill` per that doc. Per D-06 correction (a) / RESEARCH Critical
Finding #1: within the same tier (user or workspace), the `.agents/skills/` alias
**outranks** `.gemini/skills/` on a name collision — so the shared write (when present
alongside Gemini's harness-specific write) wins any tie, not the harness-specific one.

## Kiro

`[ASSUMED]` — not installed on the maintainer's machine (confirmed via the pre-flight
`command -v kiro`/`command -v kiro-cli` above; D-10). Sources: `kiro.dev/docs/skills/` and
`kiro.dev/docs/steering/`, both fetched 2026-09-18. Paths codegraph writes:
`.kiro/skills/codegraph` (local) and `~/.kiro/skills/codegraph` (global) — see
`05-05-SUMMARY.md`'s `kiroSkillDirs`. `AGENTS.md` is read as a steering source by Kiro itself
(`~/.kiro/steering/AGENTS.md` global, `./AGENTS.md` local) — no codegraph write; this is an
advisory (possible doubled instructions when another target writes those paths), not a
change (D-06 correction (d)).

---

## Summary table

| Harness | Scope | Path | Verdict |
|---|---|---|---|
| Cursor | project | `.agents/skills/codegraph` (shared) | `[ASSUMED]`, not probed (no Cursor account) |
| Cursor (D-11 AGENTS.md) | project | `AGENTS.md` (repo root) | not probed, so the no-change branch |
| opencode | project | `.agents/skills/codegraph` (shared) | read; duplicate-name WARN (log-only, one entry, winner varies, bytes identical) |
| Antigravity | global | `~/.gemini/antigravity-cli/skills/codegraph` | **not read** (agy 1.2.6 reads `~/.gemini/config/skills/`, proven by probe), so the dir moves there (1A, 05-07); GEMINI.md read |
| Global shared path (`~/.agents/skills/codegraph`) | global | `~/.agents/skills/codegraph` | `[ASSUMED]` — sessions ran at project scope |
| Gemini CLI | local + global | `.gemini/skills/codegraph`, `~/.gemini/skills/codegraph` | `[ASSUMED]` — not installed (D-10) |
| Kiro | local + global | `.kiro/skills/codegraph`, `~/.kiro/skills/codegraph` | `[ASSUMED]` — not installed (D-10) |
