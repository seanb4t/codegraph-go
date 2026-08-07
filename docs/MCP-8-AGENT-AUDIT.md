# 8-agent MCP protocol negotiation audit (VRFY-05)

**Measured on:** 2026-08-05

**Method:** proxying capture shim (`tools/mcpaudit`, D-09). For each roster
client already installed on this machine, the client's own MCP configuration
was rewritten, one client at a time, to invoke `tools/mcpaudit -real
<absolute path to the real codegraph binary> -log <observation log path>`
in place of the real `codegraph serve --mcp` command. The shim proxies every
byte through unchanged in both directions while observing the client's
`initialize` request and the server's matching `initialize` response — the
`protocolVersion offered` column below is read from the client's own
request, and the `protocolVersion negotiated` column is read from the
**server's response**, never inferred from what the client offered. Each
client was then driven through exactly one non-interactive prompt (recorded
verbatim per row) asking it to call the `codegraph_status` tool, bounded by
a 120-second timeout, and the resulting observation log line was
transcribed into the row below. Every measured row's config file was backed
up and SHA-256-verified before any edit, restored immediately after
measurement, and the restoration proven by a `sha256-before`/`sha256-after`
pair — see the `Config restore` column.

This document never pastes raw observation-log frames (T-02-01) — only the
declared columns below, transcribed by hand from each session's completed
`Observation` record.

## Roster (fixed order, per ROADMAP.md success criterion 5)

| Client | Shipped version | Status | protocolVersion offered | protocolVersion negotiated | Declared capabilities | Probe behavior | Measured on | Config restore | Unverified (doc-sourced) |
|---|---|---|---|---|---|---|---|---|---|
| **Claude Code** | `2.1.222` [VERIFIED: `claude --version`, run 2026-08-05] | **MEASURED** | `2025-11-25` | `2025-11-25` | `{"roots":{"listChanged":true},"elicitation":{}}` | neither | 2026-08-05 | `~/.claude.json` (global `mcpServers.codegraph`): sha256-before=`4a3b4d8072a9f02321ac5d73f5df5da5c0296e94727a92c0dfee2e038820eb09` sha256-after=`4a3b4d8072a9f02321ac5d73f5df5da5c0296e94727a92c0dfee2e038820eb09` match=yes | — |
| **Cursor** | — | **UNMEASURED** — not installed: no `cursor` CLI on PATH and no `Cursor.app` in `/Applications` [VERIFIED: `command -v cursor`, `ls /Applications`, run 2026-08-05] | | | | | | n/a | — |
| **Codex CLI** | `codex-cli 0.146.0` [VERIFIED: `codex --version`, run 2026-08-05] | **MEASURED** | `2025-06-18` | `2025-06-18` | `{"elicitation":{"form":{},"url":{}}}` | second process spawn (two `codegraph serve --mcp` processes were started for this one session; the first exited without ever sending a client frame, the second carried the full `initialize` exchange) | 2026-08-05 | `~/.codex/config.toml` (`[mcp_servers.codegraph]`): sha256-before=`eb2a2a28ae61ea4ef2168acdbf5aa446a57d52b880b350e7b519ad397b52f54c` sha256-after=`eb2a2a28ae61ea4ef2168acdbf5aa446a57d52b880b350e7b519ad397b52f54c` match=yes | — |
| **opencode** | `1.18.10` [VERIFIED: `opencode --version`, run 2026-08-05] | **MEASURED** | `2025-11-25` | `2025-11-25` | `{"roots":{}}` | neither | 2026-08-05 | `~/.config/opencode/opencode.json` (`.mcp.codegraph`): sha256-before=`cec1f0ef18c2eff19304daf603626e2e8ba27cf62b99b4896f7ef629a2f97000` sha256-after=`cec1f0ef18c2eff19304daf603626e2e8ba27cf62b99b4896f7ef629a2f97000` match=yes | — |
| **Gemini CLI** | — | **UNMEASURED** — not installed: no `gemini` CLI on PATH; only an unrelated desktop app ("Gemini 2.app") present in `/Applications` [VERIFIED: `command -v gemini`, `ls /Applications`, run 2026-08-05] | | | | | | n/a | — |
| **Hermes** | — | **UNMEASURED** — not installed: no `hermes` binary on PATH and no matching app in `/Applications` [VERIFIED: `command -v hermes`, `ls /Applications`, run 2026-08-05] | | | | | | n/a | — |
| **Antigravity** | — | **UNMEASURED** — GUI-only IDE, not confirmed non-interactively scriptable within this audit's bounded spike: `Antigravity.app`/`Antigravity IDE.app` are present in `/Applications` but no `antigravity` CLI exists on PATH or inside the app bundle [VERIFIED: `command -v antigravity`, `ls /Applications/Antigravity.app/Contents/MacOS`, run 2026-08-05]. The spike confirmed `~/.gemini/config/mcp_config.json` exists and already carries a `codegraph` entry (shared with Gemini CLI per FEATURES.md), so editing it is mechanically possible — but launching/relaunching the GUI app and driving it through one bounded, non-interactive handshake could not be verified without an interactive session, so it was not attempted rather than risk an unbounded or unrestorable edit against a live desktop app. | | | | | | n/a | — |
| **Kiro** | — | **UNMEASURED** — not installed: no `kiro` binary on PATH and no matching app in `/Applications` [VERIFIED: `command -v kiro`, `ls /Applications`, run 2026-08-05] | | | | | | n/a | — |

## A caveat on spot-checking `~/.claude.json`'s digest

Unlike `~/.codex/config.toml` and `~/.config/opencode/opencode.json` — which
are only touched by explicit user edits — `~/.claude.json` is Claude Code's
own actively-written session state file (usage stats, per-project
bookkeeping, timestamps). It is written continuously by ordinary Claude Code
activity, including any other concurrently running session. The
`sha256-after` published for Claude Code above was captured at the instant
of restoration and is an accurate, verified proof for that instant — but a
spot-check run any time afterward may legitimately show a *different*
current digest simply because the file kept changing for unrelated reasons
after the audit completed, not because the restore failed. **What matters,
and what was re-verified after writing this document, is that the
`mcpServers.codegraph` entry itself is byte-identical to its pre-audit
value** (`"command": "/Volumes/Code/github.com/seanb4t/codegraph-go/bin/codegraph"`,
`"args": ["serve", "--mcp"]`) — i.e. it no longer names the shim. A reviewer
spot-checking this row should compare the `codegraph` entry's own JSON, not
the whole file's digest, unless no other Claude Code session has touched
`~/.claude.json` since the audit ran.

## Verbatim commands run

- **Claude Code:** `claude -p "Call the codegraph_status tool from the codegraph MCP server and report the raw result." --allowedTools mcp__codegraph__codegraph_status --output-format text`
- **Codex CLI:** `codex exec "Call the codegraph_status tool from the codegraph MCP server and report the raw result." --skip-git-repo-check`
- **opencode:** `opencode run "Call the codegraph_status tool from the codegraph MCP server and report the raw result." --auto`

All three completed within the 120-second timeout and exited 0. None of the
three clients actually invoked `codegraph_status` in its raw MCP form — each
independently reported that the connected server exposes only
`codegraph_explore` (the default companion-tool allowlist behavior;
`CODEGRAPH_MCP_TOOLS` was not set for this audit, matching how each client's
real production config runs today) and declined to fabricate a result. This
does not affect the audit: the `initialize` exchange — the only exchange
VRFY-05 measures — completed and was fully observed in all three sessions
before the tool-call prompt was ever answered.

## What the probe column implies for Phase 3

Two of the three measured clients (Claude Code, opencode) show `neither` —
their first, and only, client-to-server frame is `initialize` itself, with
no pre-initialize probe request and no second process spawn. Codex CLI is
the one exception on this small sample: it starts the `codegraph` process
twice for a single logical session, with the first process exiting having
sent nothing at all. This is exactly the shape PITFALLS Pitfall 8 predicted
(`server/discover` is described in the `2026-07-28` changelog as usable "as
a backward-compatibility probe on STDIO" and clients may already probe with
a second spawn today, ahead of the spec formalizing it). With only one of
three measured clients exhibiting this behavior, and even that one not
sending a distinguishable *probe request* (just a bare extra process
lifecycle), this sample does not show `server/discover` as urgently
latency-critical for Phase 3 — but it does confirm that **a second process
spawn is a real, already-observed client behavior**, not a hypothetical.
Phase 3 should treat surviving a redundant process spawn (already true
today, since the server has no problem being started twice) as a baseline
it must not regress, rather than as new work `server/discover` creates.

## Reproducing this audit

Each restored config's `command` points at
`/Volumes/Code/github.com/seanb4t/codegraph-go/bin/codegraph` — the
developer's real, pre-audit local install location. That path is **not**
populated by this repository's checked-in build tooling: `/bin/` is
gitignored (`.gitignore`), `task build` is a compile-check only
(`go build ./...`, no output binary retained), and `task build:release`
writes `./codegraph` at the repo root, not `bin/codegraph`. It is a
developer-managed local install path outside this repo's build graph.

Consequence for anyone re-running this audit or spot-checking a restored
config: a connection attempt against a freshly cloned checkout will fail
with "binary not found" until `go build -o bin/codegraph ./cmd/codegraph` is
run by hand. That failure is expected and orthogonal to config
restoration — it does **not** indicate a failed restore or a broken audit;
the configs were verified byte-identical to their pre-audit values (see the
`Config restore` column above) regardless of whether `bin/codegraph`
currently exists on disk.

## UNMEASURED rows: blocking reasons on this machine

Five of the eight roster rows are UNMEASURED, all for the same underlying
reason class: the client is not installed on this audit machine, or (in
Antigravity's single partial-installation case) has no confirmed
non-interactive entrypoint. Per D-10, every UNMEASURED row's measurement
columns are left empty rather than filled from any vendor documentation —
no `protocolVersion` value is claimed for any of these five clients
anywhere in this document.
