#!/bin/sh
# .claude/hooks/session-nudge.sh — NUDGE-01 / NUDGE-02
#
# Claude Code's SessionStart hook dispatcher pipes a JSON event payload to
# this script's stdin; no stdin parsing is needed here because the
# settings.json matcher entries already filter dispatch down to the
# "startup" and "resume" events (D-07) before this script ever runs.
#
# The script never invokes the codegraph binary, never opens or reads the
# index, and never parses its own stdin. It checks the project directory
# for .codegraph/. Only when that is missing and the project directory is
# a linked worktree root (its .git is a file) does it run one git command,
# to find the main checkout and check for .codegraph/ there (#139).
project_dir=${CLAUDE_PROJECT_DIR:-.}

nudge() {
  printf '%s\n' 'This repo has a codegraph index — prefer codegraph_explore / `codegraph explore` over grep for where-is-X / how-does-Y questions.'
}

if [ -d "$project_dir/.codegraph" ]; then
  nudge
elif [ -f "$project_dir/.git" ]; then
  # The first record of the porcelain list is the main worktree. A record
  # with a "bare" line means the repository has no main worktree. The home
  # directory is never an index root, as given or with symlinks resolved.
  git -C "$project_dir" worktree list --porcelain 2>/dev/null | {
    main=
    while IFS= read -r line; do
      case $line in
      '') break ;;
      bare) main=; break ;;
      'worktree '*) main=${line#worktree } ;;
      esac
    done
    home=
    [ -n "$HOME" ] && home=$(cd "$HOME" 2>/dev/null && pwd -P)
    if [ -n "$main" ] && [ "$main" != "$HOME" ] && [ "$main" != "$home" ] && [ -d "$main/.codegraph" ]; then
      nudge
    fi
  }
fi
exit 0
