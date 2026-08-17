# Phase 5: Process, CI & In-Tree Sweep — Discussion Log

**Gathered:** 2026-08-15
**Mode:** autonomous `--interactive` (discuss inline, user answers all questions)

---

## Phase Boundary (from ROADMAP, amended)

Sweep the contributor-facing and in-tree surfaces so a contributor and an agent meet a project described on its own terms — with TypeScript-the-indexed-language intact. Requirements: PROC-01/02/03, CODE-01, CODE-03 (amended — drop migrate).

## Areas selected for discussion

All three gray areas (user-selected all):

1. CODE-01 in-tree sweep line
2. CODE-03 migrate reframing → **became a scope ruling**
3. PROC-03 workflow sweep scope

## Decisions by area

### 1. CODE-01 in-tree sweep line → framing = comparison baseline; truth = capability names + history
**Options:** framing=baseline/truth=names+history / broader (strip most history) / narrower (only active claims)
**Selected:** framing = references to the prior implementation as a comparison baseline; truth = capability names + history.
Remove "matches TS", "parity with the original", "drop-in", "vs the TS version", "behavioral parity". Keep `tsextract`, the language registry, the capability matrix, TypeScript-as-indexed-language, and past-tense history. Term-by-term with recorded reasons, never regex. Includes the `syntheticParitySrc` residual and the matrix `golden-parity` code comments.

### 2. CODE-03 migrate → **AMENDED: drop migrate entirely (maintainer ruling)**
The user initially framed this as reframing, then ruled **"let's drop the migrate capability entirely."** After I named the tension (CODE-03 says migrate stays; the milestone's "never capability" rule), the user reaffirmed: **drop migrate entirely, amend CODE-03.** The user further clarified that the "core value" text ("uninstall TS CodeGraph... migrate their indexes... works the same or better") is itself competitive framing — the parity story — not a binding constraint.

**Amended scope:** remove the command, the `internal/migrate` package, its fixture, and the `modernc.org/sqlite` sole-use dependency. Nothing left referencing it. A TS user installs the Go binary and re-indexes from source. Recorded deliberately in REQUIREMENTS.md CODE-03 and ROADMAP.md (commit `89fe57f`), and in engram (`4j8kjck5gd`, `myywc0y9vm`).

### 3. PROC-03 workflow sweep scope → sweep framing prose in contributor templates + workflows
Sweep the remaining framing prose ("ports observable behavior from another implementation", "parity decisions") in the issue templates, PR template + variants, and the workflow files (ci, release, bench, post-release-verify, linux-cross-canary, require-issue-link, auto-close-unsolicited-prs, corpora). Phase 4 already re-pointed the FLAG-PARITY links; this phase sweeps the framing prose. `bench.yml`'s own de-coupling is Phase 6's; only its framing is swept here.

## Claude's Discretion (recorded)
- Exact rewording of each contributor-facing template
- The new name for `syntheticParitySrc`
- Order of the internal//tools//test/ comment sweep

## Deferred
- `bench.yml` de-coupling + `docs/BENCHMARKS.md` → Phase 6
- Memory sweep (MEM-01/02) → Phase 6
- Whether to restore migrate later → future decision (git history retains it)

---

*All decisions captured in 05-CONTEXT.md (D-01 through D-06). CODE-03 amended via maintainer ruling.*
