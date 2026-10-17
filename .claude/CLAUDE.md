## Project

**codegraph-go** gives coding agents (Claude Code, Codex, Cursor, Gemini and others) a pre-indexed code knowledge graph: fast symbol, call-path and impact queries from one static Go binary with a CLI, an MCP server, a local graph UI and auto-sync. There is no bundled runtime to install or manage.

The constraints that bind every change are in `openspec/config.yaml` under `context`. The short version: one static binary per platform, CGo only for tree-sitter, signed and reproducible releases, macOS and Linux (Windows only through WSL2), and CLI and MCP output shapes are a compatibility surface.

## Workflow

Planning runs on [OpenSpec](https://github.com/Fission-AI/OpenSpec). The backlog lives in GitHub issues.

- **Issue first.** Every change starts from a GitHub issue. A feature or enhancement needs `approved-feature` or `approved-enhancement` before a PR. `spec-needed` means a merged OpenSpec proposal comes before code.
- **Behavior changes go through an OpenSpec change:** `/opsx:propose`, then `/opsx:apply`, then `/opsx:archive`. The line after the title of `proposal.md` is `Issue: #<n>`. Specs live in `openspec/specs/`, active changes in `openspec/changes/`.
- **Fixes and chores** that change no specified behavior go straight to a PR.
- **Pull requests.** A proposal-only PR says `Refs #<n>`. The implementation PR says `Closes #<n>` and carries the code, the changelog fragment and the `openspec archive` result.
- **Generated files.** `openspec update` owns the `/opsx:*` commands (`.claude/commands/opsx/`) and the `openspec-*` skills (`.claude/skills/`, `.agents/skills/`). Customize OpenSpec through `openspec/config.yaml`, never by editing those files.
- **History.** The planning record from before OpenSpec is archived at the `planning-archive-v0.15.0` tag; read a file with `git show planning-archive-v0.15.0:.planning/<path>`. Requirement and decision IDs in comments (`SYNC-02`, `D-06R`) resolve there, and `docs/2026-10-17-gsd-era-decisions.md` holds the old decision log.

## Technology stack

| Area | Choice |
|---|---|
| Language | Go (version in `go.mod`), one static binary per platform |
| Parsing | `tree-sitter/go-tree-sitter` plus per-language grammar modules, the one CGo dependency (`PARSER-DECISION.md`) |
| Storage | `cockroachdb/pebble/v2` |
| MCP | `modelcontextprotocol/go-sdk`, stdio transport |
| CLI and terminal UI | `spf13/cobra`; `charm.land` bubbletea and lipgloss v2 |
| File watching | `fsnotify/fsnotify` |
| Graph UI | SvelteKit 2 and Svelte 5 in `web/` (pnpm), embedded by `web/embed.go`, talking Connect RPC (protobuf via buf) |
| Release | GoReleaser v2, cosign keyless signing, SLSA provenance, SBOMs, Apple notarization, Homebrew tap, changie for the changelog and versions |
| Tooling | `Taskfile.yml`; `task`, `goreleaser`, `actionlint` and `changie` build from the isolated `go.tool*.mod` files |

## Working here

- `task test` runs the suite, `task lint` the linters. `task --list` shows the rest.
- **Changelog fragments.** A PR that changes `cmd/**` or `internal/**` with a user-visible effect adds `task changie -- new -k <Kind> -b "<one sentence>"` (kinds: Breaking, Features, Fixes, Performance, Dependencies). Otherwise the PR body carries `<!-- changelog-exempt: <reason> -->`. The required `fragment-required` check enforces this.
- **Embedded sources.** `.claude/hooks/{hooks.json,session-nudge.sh,pretooluse-nudge.sh}`, `.claude/skills/codegraph/SKILL.md` and `.codex/hooks/*` are `//go:embed` sources that ship in the binary. Edit them as product code.
- **Required checks are pinned by job name** in the `protect-main` ruleset. Renaming a CI job blocks every PR until the ruleset changes too.
- **Tests cover only code this repo owns.** Never test third-party behavior (git, GitHub, changie, GoReleaser, cobra and so on).
- `LICENSE` stays verbatim MIT text. Upstream attribution lives only in `NOTICE`.
