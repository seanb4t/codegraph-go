# Phase 2: Phase-Close Fragment Capability - Discussion Log

> **Audit trail only.** Do not use this file as input to planning, research or execution agents.
> Decisions are in CONTEXT.md. This log keeps the alternatives that were considered.

**Date:** 2026-09-25
**Phase:** 02-phase-close-fragment-capability
**Mode:** `--auto`. Every area was auto-selected, and each question was answered with the recommended option.
**Areas discussed:** Capability identity & manifest, PR number source, Changie command resolution, Fragment classification, Idempotency across verify:post call sites, Commit convention, Proof harness, codegraph-go install & docs

---

## Capability identity & manifest

| Option | Description | Selected |
|--------|-------------|----------|
| id `changie`, skill `changie-fragments` (invoked as `gsd-changie-fragments`) | Short id, not a reserved prefix, and matches the memory `f7m4ye3rkc` design | ✓ |
| id `changie-fragments` | Duplicates the skill name | |
| id `gsd-capability-changie` | Reserved `gsd-` prefix; rejected by the validator | |

[auto] Capability identity: Q "id and skill naming?" → Selected "id `changie`, skill `changie-fragments`" (recommended default)

## PR number source

| Option | Description | Selected |
|--------|-------------|----------|
| Look up the open PR for the current branch via `gh`; skip if none; `--pr` override; draft milestone PR opened at a maintainer checkpoint | Real number, never a placeholder | ✓ |
| Config value holding the PR number | Stale across milestones; wrong on other branches | |
| Placeholder number (e.g. `PR=1`) | Invented evidence; links the wrong PR | |
| Make `PR` optional | Contradicts CHG-04 and Phase 1 D-11 | |

[auto] PR number: Q "where does the skill get <n>?" → Selected "gh lookup for the current branch + draft milestone PR" (recommended default)

## Changie command resolution

| Option | Description | Selected |
|--------|-------------|----------|
| Federated key `workflow.changie_command` (default `changie`); codegraph-go sets `task changie --` | Portable; works with this repo's modfile-pinned changie | ✓ |
| Literal `PATH` lookup only | Always skips in codegraph-go (changie is not on `PATH`), so CAP-05 could never pass | |
| Require contributors to install changie on `PATH` | Contradicts Phase 1 D-04/D-05's single install path | |

[auto] Changie command: Q "how does the skill find changie?" → Selected "configurable command key" (recommended default)

## Fragment classification

| Option | Description | Selected |
|--------|-------------|----------|
| Model reads the SUMMARYs with a user-visible rubric; conventional-commit types as hints; kinds parsed from the host `.changie.yaml` | Follows CAP-03's SUMMARY input; adds no new structure | ✓ |
| New SUMMARY frontmatter field `changelog:` | Invents structure in a tool-owned file (forbidden) | |
| Commit-type mapping only | Misses context; many phases use `docs`/`chore` for user-visible work | |

[auto] Classification: Q "how are user-visible changes and kinds decided?" → Selected "SUMMARY reading + rubric + commit hints" (recommended default)

## Idempotency across verify:post call sites

| Option | Description | Selected |
|--------|-------------|----------|
| Git trailers `Changie-Phase` / `Changie-Summaries`; process only uncovered SUMMARYs | Handles double dispatch and later gap-closure plans | ✓ |
| Marker file in the phase directory | Adds structure to the tool-owned `.planning/` tree | |
| `Phase` custom field in `.changie.yaml` | Changes the Phase 1 locked vocabulary | |
| No idempotency | execute-phase + verify-work + autonomous would write duplicates | |

[auto] Idempotency: Q "how to avoid duplicate fragments?" → Selected "commit trailers" (recommended default)

## Commit convention

| Option | Description | Selected |
|--------|-------------|----------|
| `docs(NN): add changelog fragments` via plain `git commit` with an explicit pathspec and trailers | Trailers supported; only this run's files staged | ✓ |
| `gsd_run query commit` | Adds no trailers (`gv8ppm3cce`) | |

[auto] Commit: Q "how are fragments committed?" → Selected "plain git commit, explicit pathspec" (recommended default)

## Proof harness

| Option | Description | Selected |
|--------|-------------|----------|
| Deterministic script + committed `test/run.sh` against a scratch project with a stub `gh`; one real skill run for the judgment half; evidence pasted into codegraph-go artifacts | Every skip case testable without a model; count-asserted | ✓ |
| Manual transcript only | Not repeatable; weak against rule `84d1gfpywd` | |

[auto] Proofs: Q "how are CAP-01..03 proven?" → Selected "script + test/run.sh" (recommended default)

## codegraph-go install & docs

| Option | Description | Selected |
|--------|-------------|----------|
| Install first, then `config-set` both keys; CONTRIBUTING subsection under "What `.planning/` is" | Tool verbs only; keeps DOCS-12's §Pull requests rewrite in Phase 5 | ✓ |
| Hand-edit `.planning/config.json` | Forbidden (tool-owned) | |
| Document under §Pull requests | Collides with DOCS-12 | |

[auto] Install: Q "install order and doc placement?" → Selected "install → config-set; subsection under What .planning/ is" (recommended default)

## Claude's Discretion

- File layout inside the capability repository, SKILL.md wording, optional CI, skip-reason wording, and how the phase directory is discovered.

## Deferred Ideas

- PORT-01 / PORT-02 (v2), a `Phase` custom field, and making the capability public.
- Reviewed but not folded: the bench `pinnedAt` todo and the GH #85 reply todo (keyword-only matches).
