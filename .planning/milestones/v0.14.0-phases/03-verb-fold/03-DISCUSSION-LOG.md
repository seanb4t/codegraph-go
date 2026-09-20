# Phase 3: Verb Fold - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-16
**Phase:** 03-verb-fold
**Areas discussed:** search --full human output (VERB-01), Rename stubs' contract (VERB-03/04/08), Golden re-freeze review gate (VERB-07), feat!: commit shape (VERB-08)

---

## search --full human output (VERB-01)

Facts presented: `Query`/`Search` share `matchNodes`; today's `query` human line equals `search`'s (`Name (Kind) File:Line`), so signature/qualified-name output is new.

| Option | Description | Selected |
|--------|-------------|----------|
| Two lines: default line + indented signature | line 1 byte-identical to default `search`; line 2 `    QualifiedName  Signature`; `--full` stays a superset | ✓ |
| One wide line | `QualifiedName (Kind) Signature File:Line` | |
| Replace name with qualified name, append signature | not a superset of the default line | |

**User's choice:** Two lines: default line + indented signature (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — same notice, same position | WORK-02 notice at the top of the human branch, as in `search`/`query` | ✓ |
| No notice under --full | diverges from every other read command | |

**User's choice:** Yes — same notice, same position (Recommended)

---

## Rename stubs' contract (VERB-03/04/08)

Facts presented: every command sets `SilenceUsage`/`SilenceErrors`, `main.go` exits 1 on any error; `man` is the only hidden command; `lock.go:208` and `index.go:128` tell users to run `codegraph unlock`.

| Option | Description | Selected |
|--------|-------------|----------|
| 1 — the tree's only error exit | non-nil error → exit 1; no new convention | ✓ |
| 2 (usage error) | distinct code; needs new main.go path + test | |

**User's choice:** 1 — the tree's only error exit (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Rename + the full replacement invocation | verbatim clause + `run: codegraph …` + lifetime line | ✓ |
| Verbatim requirement text only | minimal | |

**User's choice:** Rename + the full replacement invocation (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| ROADMAP backlog entry + BREAKING CHANGE footer | `999.x` row via the roadmap verb + CHANGELOG via release-please; CLI-REFERENCE allowlist repeats the version | ✓ |
| BREAKING CHANGE footer + CLI-REFERENCE reason only | nothing in `.planning/` | |

**User's choice:** ROADMAP backlog entry + BREAKING CHANGE footer (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| `codegraph daemon unlock` — updated in the same commit as the move | census finds zero old-verb refs; tests updated RED-first | ✓ |
| Leave them naming the stub for one release | would need a census exception | |

**User's choice:** `codegraph daemon unlock` — updated in the same commit as the move (Recommended)

---

## Golden re-freeze review gate (VERB-07)

Facts presented: goldens naming `query` are mostly data fixtures/JSON keys; the real generated surface is `docs/CLI-REFERENCE.md`, completions, man pages, `query_cli_test.go`, lock-message assertions; wire-oracle transcripts mention `query` only inside tool output text.

| Option | Description | Selected |
|--------|-------------|----------|
| Executor reviews; you review at end of phase | regenerate, RED against a reintroduced visible `query`, mutation log, commit; diff summary in SUMMARY | ✓ |
| Show me the diff first (blocking-human) | halt before the re-freeze commit | |

**User's choice:** Executor reviews; you review at end of phase (Recommended)

---

## feat!: commit shape (VERB-08)

| Option | Description | Selected |
|--------|-------------|----------|
| One `feat!:` commit for the surface change; ordinary commits around it | single BREAKING CHANGES entry | ✓ |
| Every fold commit is `feat!:` | louder CHANGELOG | |
| Squash the whole phase into one `feat!:` commit | loses atomic history | |

**User's choice:** One `feat!:` commit for the surface change; ordinary commits around it (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Rename target + removal minor | allowlist reason doubles as the documented lifetime | ✓ |
| Rename target only | removal minor lives only in CHANGELOG + backlog | |

**User's choice:** Rename target + removal minor (Recommended)

---

## Claude's Discretion

- Stub file layout; flag-parse test location; census script placement and exact `rg` invocation; completions/man regeneration path (follow the existing generators); `999.x` row wording; commit ordering around the single `feat!:` commit

## Deferred Ideas

- MCP `codegraph_search` full-record flag; a distinct "renamed" exit code; the stubs' removal itself (v0.15.0, backlog row)
- Reviewed, not folded: the bench `pinnedAt` HEAD-only todo (keyword match only)
