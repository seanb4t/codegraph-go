#!/bin/sh
# .claude/hooks/session-nudge.sh — NUDGE-01 / NUDGE-02
#
# Claude Code's SessionStart hook dispatcher pipes a JSON event payload to
# this script's stdin; no stdin parsing is needed here because the
# settings.json matcher entries already filter dispatch down to the
# "startup" and "resume" events (D-07) before this script ever runs.
#
# The only filesystem operation this script performs is the single
# directory-existence check below. It never invokes the codegraph binary,
# never opens or reads the index, and never parses its own stdin.
if [ -d "${CLAUDE_PROJECT_DIR:-.}/.codegraph" ]; then
  printf '%s\n' 'This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y questions.'
fi
exit 0
