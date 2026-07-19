# Flag parity: TS CodeGraph 1.3.1 vs codegraph-go (SURF-05)

This document is the systematic per-command audit of `codegraph-go`'s CLI
flag surface against the live, pinned `@colbymchenry/codegraph@1.3.1` TS CLI
(`npm install @colbymchenry/codegraph@1.3.1 --no-save` into a throwaway
prefix, per `.planning/phases/08-surface-reconciliation-signed-v1-0-0-release/08-RESEARCH.md`).
It is the single artifact REL-04's drop-in gate reads to declare surface
parity: every TS 1.3.1 flag name + default is either **present** in Go, a
recorded **divergence** (with a reason), or documented as a **Go-only**
extension.

> **Status note:** this audit is a snapshot after SURF-01..05 landed
> (impact depth default, `files --dir`, short-flag aliases, `affected`
> scripting flags). It does not change any flag, default, or command
> behavior itself — SURF-05 is audit + doc + drift-guard only. Behavioral
> divergences recorded below (`install --auto-allow`, `files --format`
> default, `node` missing file-mode, etc.) are deliberate and are **not**
> resolved by this document.

## How this doc is enforced

`internal/cli/flag_parity_test.go` walks the real `newRootCmd()` command
tree (every command and subcommand) and, for each registered long flag
name, asserts the name appears as a literal substring somewhere in this
file's text. A registered flag with no corresponding row here fails the
build (`go test ./internal/cli/... -run FlagParity -count=1`) — this doc
cannot silently drift from the actual cobra flag surface.

Columns below: **TS flag** (name/short/default, `[VERIFIED: TS 1.3.1 --help
+ bin/codegraph.js]`) → **Go flag** (name/short/default, current
`internal/cli/*.go` source) → **Status** (`present` / `divergence(reason)`
/ `Go-only`).

## `init`

| TS flag | Go flag | Status |
|---|---|---|
| `-i`/`--index` (deprecated) | — | `divergence(TS itself marks this deprecated; not ported)` |
| `-f`/`--force` (home/root safety guard) | — | `divergence(no Go equivalent exists; a new safety-guard behavior is out of SURF's flag-name/default scope, not a mechanical gap — see index below for the analogous case)` |
| `-v`/`--verbose` | `-v`/`--verbose` | `present` |
| — | `-q`/`--quiet` | `Go-only` |
| — | `--workers` (no short) | `Go-only` |

## `index`

| TS flag | Go flag | Status |
|---|---|---|
| `-f`/`--force` (home/root safety guard) | `-f`/`--force` (rebuild-without-prompting) | `divergence(same letter, different semantic — Go's -f skips the rebuild confirmation prompt; TS's -f bypasses a home/filesystem-root safety check. Accepted: not a mechanical flag-name gap)` |
| `-q`/`--quiet` | `-q`/`--quiet` | `present` |
| `-v`/`--verbose` | `-v`/`--verbose` | `present` |
| — | `--workers` (no short) | `Go-only` |

## `uninit`

| TS flag | Go flag | Status |
|---|---|---|
| `-f`/`--force` (skip confirm) | `-f`/`--force` (skip confirm) | `present` (same letter, same semantic) |

## `query`

| TS flag | Go flag | Status |
|---|---|---|
| `-p` | `-p`/`--path` | `present` |
| `-k`/`--kind` | `-k`/`--kind` | `present` |
| `-l`/`--limit` (default 10) | `-l`/`--limit` (default `0`) | `divergence(Go's 0 means "uncapped", validated by validateLimit and never defaulted to a specific number downstream — internal/query/search.go's Query(); TS defaults to 10. Accepted: a behavioral default gap pre-dating this phase's SURF-03 short-flag work, not resolved here)` |
| `-j`/`--json` | `-j`/`--json` | `present` |

## `search` (Go-only, no TS command)

| TS flag | Go flag | Status |
|---|---|---|
| — | `-p`/`--path` | `Go-only` |
| — | `--kind` (no short) | `Go-only` |
| — | `--limit` (no short) | `Go-only` |
| — | `--json` (no short) | `Go-only` |

`search` itself has no TS 1.3.1 counterpart at all — the whole command is
a documented Go-only extension (CONTEXT D-06).

## `callers`

| TS flag | Go flag | Status |
|---|---|---|
| `-p` | `-p`/`--path` | `present` |
| `-l`/`--limit` (default 20) | `-l`/`--limit` (default `0`, uncapped) | `divergence(same 0-means-uncapped gap as query's --limit; TS defaults to 20)` |
| `-j`/`--json` | `-j`/`--json` | `present` |

## `callees`

| TS flag | Go flag | Status |
|---|---|---|
| `-p` | `-p`/`--path` | `present` |
| `-l`/`--limit` (default 20) | `-l`/`--limit` (default `0`, uncapped) | `divergence(same 0-means-uncapped gap as query's --limit; TS defaults to 20)` |
| `-j`/`--json` | `-j`/`--json` | `present` |

## `impact`

| TS flag | Go flag | Status |
|---|---|---|
| `-p` | `-p`/`--path` | `present` |
| `-d`/`--depth` (default 2, clamp `[1,10]`) | `-d`/`--depth` (default `0`→engine `defaultDepth=2`, clamp `[1,50]`) | `present` (default now matches TS per SURF-01/D-02); `divergence(max clamp intentionally stays 50, not TS's 10 — D-02 explicit, not a gap)` |
| `-j`/`--json` | `-j`/`--json` | `present` |

## `affected`

| TS flag | Go flag | Status |
|---|---|---|
| `-p` | `-p`/`--path` | `present` |
| `--stdin` | `--stdin` (no short, matches TS) | `present` |
| `-d`/`--depth` (default 5) | `-d`/`--depth` (default `0`→engine `defaultAffectedDepth=5`) | `present` |
| `-f`/`--filter <glob>` | `-f`/`--filter <glob>` (`filepath.Match` semantics) | `present` |
| `-q`/`--quiet` | `-q`/`--quiet` | `present` |
| `-j`/`--json` | `-j`/`--json` | `present` |

Full parity achieved by SURF-04/SURF-03 (08-04/08-05): every TS `affected`
flag name, short, and effective default is present. `--filter`'s exact
glob engine differs (`filepath.Match`, which does not cross a `/` with
`*`) from TS's hand-rolled `globToRegex` — both are glob-shaped, narrow
divergence in `**`-style patterns only, not a missing flag.

## `files`

| TS flag | Go flag | Status |
|---|---|---|
| `-p` | `-p`/`--path` | `present` |
| `--filter <dir>` (plain path-prefix `startsWith` match) | `--dir <prefix>` (new, `strings.HasPrefix`) | `divergence(D-03 locked "keep ours + add TS's": TS's directory filter is spelled --filter, but that name is already ours for language — the new flag takes the distinct name --dir and implements TS's exact prefix-match semantics)` |
| — (TS has no separate language filter) | `--filter` (language, kept) | `Go-only` (pre-existing, retained per D-03) |
| `--pattern <glob>` | `--pattern <glob>` | `present` |
| `--format` (default `"tree"`) | `--format` (default `""`→flat) | `divergence(default value mismatch — Go defaults to flat, TS to tree; accepted, not changed in this phase to avoid surprising existing Go users)` |
| `--max-depth` | `--depth` (directory-nesting cap, default 0=unlimited) | `divergence(naming only — Go keeps its established --depth name rather than a breaking rename to --max-depth; same semantic: directory-nesting cap)` |
| `--no-metadata` | — | `divergence(TS-only flag, not ported to Go; recorded, not implemented — out of this phase's flag/default reconciliation scope)` |
| `-j`/`--json` | `-j`/`--json` | `present` |

## `status`

| TS flag | Go flag | Status |
|---|---|---|
| — | `-p`/`--path` | `Go-only` (needed since Go's status resolves an explicit repo root; harmless addition) |
| `-j`/`--json` | `-j`/`--json` | `present` |

## `node`

| TS flag | Go flag | Status |
|---|---|---|
| `-p` | `-p`/`--path` | `present` |
| `-f`/`--file` | `-f`/`--file` | `present` |
| `--offset` (file-mode only) | — | `divergence(TS's whole "file mode" — reading a file by line/offset with no symbol name — has no Go equivalent at all; see below)` |
| `--limit` (file-mode only) | — | `divergence(same file-mode gap)` |
| `--symbols-only` (file-mode only) | — | `divergence(same file-mode gap)` |
| — | `-l`/`--line` (NODE-03) | `Go-only` |

TS's `node [name]` supports a second mode entirely: when `name` is
omitted, it reads a FILE with line numbers plus dependents, governed by
`--offset`/`--limit`/`--symbols-only`. Go's `node.go` has no code path
where an empty symbol triggers a file read — `--file` only disambiguates a
symbol match today. This is a genuine net-new capability gap (comparable
in scope to a small feature), not a flag/default reconciliation item, and
is recorded here as an accepted divergence rather than implemented in this
phase (same treatment as `search`/`migrate`'s Go-only/accepted status,
per CONTEXT's own precedent — see 08-RESEARCH.md Pitfall 5).

## `explore`

| TS flag | Go flag | Status |
|---|---|---|
| `-p` | `-p`/`--path` | `present` |
| `--max-files` (no short) | `--max-files` (no short, default `0`→5) | `present` |

## `serve` (hidden in both)

| TS flag | Go flag | Status |
|---|---|---|
| `-p` | `-p`/`--path` | `present` |
| `--mcp` | `--mcp` | `present` |
| `--no-watch` | `--no-watch` | `present` |
| — | `--watch` (force-on, overrides WSL2/slow-fs auto-off) | `Go-only` (Phase 3) |

## `sync`

| TS flag | Go flag | Status |
|---|---|---|
| `-q`/`--quiet` | `-q`/`--quiet` | `present` |
| — | `-v`/`--verbose` | `Go-only` |
| — | `--workers` (no short) | `Go-only` |

## `daemon` / `daemons`

| TS flag | Go flag | Status |
|---|---|---|
| (none — TS's `daemon` is an interactive picker only, no flags) | `-p`/`--path` (bare/list) | `Go-only` |
| | `daemon start`: `-p`, `-q`/`--quiet`, `-v`/`--verbose`, `--workers` | `Go-only` |
| | `daemon stop`: `-p`, `--all` | `Go-only` |

TS's `daemon` command is flag-less (picker-only). Go's richer
`start`/`stop` subcommand surface (DMON-01..04, shipped in Phase 7) is a
documented Go-only extension — there is no TS flag to reconcile against.

## `unlock`

| TS flag | Go flag | Status |
|---|---|---|
| (no flags) | (no flags) | `present` (flag-less command, matches) |

## `version`

| TS flag | Go flag | Status |
|---|---|---|
| (no flags; `-v`/`--version` at root) | `--json` (no short) | `Go-only` |

## `telemetry`

| TS flag | Go flag | Status |
|---|---|---|
| (no flags) | (no flags) | `present` (flag-less command, matches) |

## `upgrade`

| TS flag | Go flag | Status |
|---|---|---|
| `--check` (no short) | `--check` (no short) | `present` |
| `-f`/`--force` | `-f`/`--force` | `present` (added by SURF-03/08-03 — previously absent entirely in Go, now matches TS's name, short, and semantic: reinstall even if already on the latest version, without weakening `verify()`) |

## `install`

| TS flag | Go flag | Status |
|---|---|---|
| `-t`/`--target` | `-t`/`--target` | `present` |
| `-l`/`--location` | `-l`/`--location` | `present` |
| `-y`/`--yes` | `-y`/`--yes` | `present` |
| `--no-permissions` | — | `divergence(TS-only flag; Go instead requires an explicit opt-in — see --auto-allow below. Not ported as a separate suppression flag)` |
| `--print-config <id>` | — | `divergence(TS-only flag; not ported)` |
| — (permissions written by default in TS, suppressed only via --no-permissions) | `--auto-allow` (no short, default `false`) | `divergence(genuine BEHAVIORAL divergence, not just naming: TS writes mcp__codegraph__* permissions by default; Go requires explicit --auto-allow opt-in. Deliberate, security-conservative Go default from Phase 6/7 — NOT flipped in this mechanical reconciliation phase per D-04/RESEARCH Pitfall 4; accepted)` |

## `uninstall`

| TS flag | Go flag | Status |
|---|---|---|
| `-t`/`--target` | `-t`/`--target` | `present` |
| `-l`/`--location` | `-l`/`--location` | `present` |
| `-y`/`--yes` | `-y`/`--yes` | `present` |

Full parity — every TS `uninstall` flag name, short, and semantic is
present (added by SURF-03/08-03).

## `migrate` (Go-only, no TS command)

| TS flag | Go flag | Status |
|---|---|---|
| — | `--from` (no short) | `Go-only` |
| — | `--to` (no short) | `Go-only` |
| — | `-f`/`--force` | `Go-only` |
| — | `--drop-dangling` (no short) | `Go-only` |

`migrate` (the one-step TS-SQLite → new-format Pebble store conversion)
has no TS 1.3.1 counterpart at all — accepted divergence, documented per
CONTEXT D-06.

## `githooks` (Go-only, no TS command)

| TS flag | Go flag | Status |
|---|---|---|
| — | `githooks install [path]`, `githooks remove [path]`, `githooks status [path]` — no flags on any subcommand | `Go-only` |

`githooks` (Phase 5) has no TS equivalent — documented Go-only extension,
no flags to reconcile.

## Summary of every recorded divergence

- **Dual `files --filter`(language, kept)/`--dir`(directory, new)** — D-03
  locked "keep ours + add TS's" divergence.
- **Short-flag divergences (accepted, not remapped):** `init -f`/`index
  -f` (TS home/root guard vs Go's skip-confirm semantic).
- **`install --auto-allow` default-off** — deliberate security-conservative
  behavioral divergence from TS's default-on permissions write.
- **`files --format` default** (`flat` in Go vs `tree` in TS).
- **`files --depth` vs TS `--max-depth`** naming divergence (same
  semantic, Go's established name kept).
- **`node` missing TS file-mode** (`--offset`/`--limit`/`--symbols-only`)
  — a net-new capability gap, not a flag reconciliation item.
- **`query`/`callers`/`callees --limit` default** — Go's `0` means
  uncapped; TS defaults to 10/20 respectively.
- **`install` missing `--no-permissions`/`--print-config <id>`** — TS-only
  flags, not ported.
- **`init` missing `-f`/`--force` (home/root guard) and deprecated `-i`**
  — TS-only, not ported (TS itself deprecates `-i`).
- **`files` missing `--no-metadata`** — TS-only flag, not ported.
- **`search`** — Go-only extension, no TS command.
- **`migrate`** — Go-only extension/accepted divergence, no TS command.
- **`githooks`** — Go-only extension, no TS command.
- **`daemon`/`daemons` subcommand surface, `serve --watch`, `version
  --json`, `sync -v`/`--workers`, `init -q`/`--workers`, `index
  --workers`, `node -l`/`--line`, `status -p`, `search`'s whole flag set**
  — additive Go-only extensions; no TS name was taken or altered.

Every command registered in `newRootCmd()` (`internal/cli/root.go`) has a
section above, including the three flag-less commands (`unlock`,
`telemetry`, and every `githooks` subcommand).
