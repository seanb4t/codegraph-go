# Phase 2: Guards, CI Wiring & Docs Burn-down - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-15
**Phase:** 02-guards-ci-wiring-docs-burn-down
**Areas discussed:** web:drift paired assertion (GRD-09), Ruleset drift test + CI wiring (GRD-11/12), button.svelte drift outcome (GRD-10), Doc claims that rot (DOCS-08/09/10)

---

## web:drift paired assertion (GRD-09)

The first question set (guard location / RED replay mechanism / set strictness) was **withdrawn** when the maintainer asked "what are we testing and why?" and then "why are we testing how copy works?". The orchestrator verified: CI runs `web:drift` on a clean checkout after `web:build:verify` builds into a scratch dir, so `98cd41dd`'s 4-file tree hashes to a different output digest than its 13-file manifest and fails in CI; `98cd41dd` never reached CI (`fad3b39c` fixed it first); the local `find` enumeration is deliberate for `//go:embed all:build`. The paired assertion would test git staging hygiene.

| Option | Description | Selected |
|--------|-------------|----------|
| Prove CI's guard, change nothing | Scratch-clone replay of 98cd41dd fails `task web:drift`; mutation-log entry; WINDOWS #29 closed with cause; requirement reworded | ✓ |
| Local refusal on a dirty web/build | `git status --porcelain web/build` non-empty → exit non-zero; a pre-push honesty check, not a gate property | |
| Paired set assertion as roadmapped | on-disk == git-tracked both directions with floor, per the ledger and pitfall 13 | |

**User's choice:** Prove CI's guard, change nothing (Recommended)
**Notes:** Maintainer reset the requirement's premise; recorded as D-01…D-03. The "local refusal" option is in Deferred Ideas.

---

## Ruleset drift test + CI wiring (GRD-11/12)

Research note presented: `GET /repos/{owner}/{repo}/rulesets/{id}` needs Metadata:read and works unauthenticated on public repos. Live fact presented: fixture (7) already diverges from the ruleset (6) — `goreleaser check (config validation, DIST-01)` is not required live.

| Option | Description | Selected |
|--------|-------------|----------|
| Steps inside the existing `test` job | Blocking immediately, no ruleset/fixture change | ✓ |
| A new dedicated job | Needs ruleset + fixture additions before it blocks | |

**User's choice:** Steps inside the existing `test` job (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Exact set equality | fixture == live, both directions | ✓ |
| Fixture ⊆ live only | tolerates extra live contexts; hides under-assertion | |

**User's choice:** Exact set equality (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Taskfile target + CI step, FAILS offline naming the cause | `check:ruleset-drift` locally and in CI; hard fail on API error | |
| Go test with t.Skip when offline | the declined GRD-07 shape; skip is a vacuous pass | |
| CI-only step, no local target | shell step in ci.yml only | ✓ |

**User's choice:** CI-only step, no local target
**Notes:** The hard-fail-on-API-error property from the first option is carried into D-06 regardless; the shared data file (D-07) is Claude's discretion.

| Option | Description | Selected |
|--------|-------------|----------|
| Add goreleaser check + tmux e2e to the ruleset now | maintainer repo-settings action; fixture grows to 8; tmux e2e required on every PR | ✓ |
| Reconcile the fixture to what is live today | drop goreleaser check from the fixture; leave tmux e2e out | |
| Add goreleaser check only; tmux e2e stays unrequired | fixture stays at 7 | |

**User's choice:** Add goreleaser check + tmux e2e to the ruleset now

---

## button.svelte drift outcome (GRD-10)

Live fact presented: `components-drift.yml` run 34820878640 (2026-09-14, scheduled) ran under Corepack pnpm v11.23.0 and reported button.svelte, command-group.svelte and command-input.svelte differing — the toolchain hypothesis in WINDOWS #31 is refuted.

| Option | Description | Selected |
|--------|-------------|----------|
| Re-vendor, with a reviewed diff | regenerate, review, run gates, commit as human-approved re-vendor | ✓ |
| Waive with cause, keep the frozen copies | monitor stays red every Monday | |
| Re-vendor only non-visual diffs; waive visual ones | per-file classification | |

**User's choice:** Re-vendor, with a reviewed diff (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| All eight families to one registry snapshot | one upstream point in time, one diff, one commit | ✓ |
| Only the families that currently drift | minimal diff, two registry generations in tree | |

**User's choice:** All eight families to one registry snapshot (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Show me the diff first (blocking-human) | checkpoint before commit | |
| Executor reviews; commit if all gates green | diff summary in SUMMARY; end-of-phase review | ✓ |

**User's choice:** Executor reviews; commit if all gates green

---

## Doc claims that rot (DOCS-08/09/10)

Live facts presented: `docs/RELEASE.md` states 27 direct / 134 indirect and credits `mark3labs/mcp-go`; go.mod has ~34 direct-ish / 116 indirect and `modelcontextprotocol/go-sdk v1.7.0`; the `tool-vuln` job is advisory (reports, never fails) and root `SECURITY.md` does not mention it; GH #14's cited line numbers are stale.

| Option | Description | Selected |
|--------|-------------|----------|
| Keep the shape, drop the raw numbers | describe wide-but-shallow, name the direct requires, point at `go list -m all`; no test | ✓ |
| Fresh counts with an 'as of <commit>' qualifier, no guard | correct today, stale on the next bump | |
| Fresh counts + a drift test against go.mod | a guard for a sentence | |

**User's choice:** Keep the shape, drop the raw numbers (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Whole-repo census, then fix every hit | positive-controlled `rg` across README/docs/workflows/SECURITY/internal/upgrade | ✓ |
| Only the three sites GH #14 names | stale line numbers; a fourth site would survive | |

**User's choice:** Whole-repo census, then fix every hit (Recommended)

| Option | Description | Selected |
|--------|-------------|----------|
| Main module blocks; tool modfiles advisory; name the accepted exposure | also names GO-2026-5932 | |
| Add the advisory line only, no exposure named | one sentence | ✓ |
| Advisory line + a drift assertion (the original GRD-08 shape) | a guard for a doc sentence | |

**User's choice:** Add the advisory line only, no exposure named

---

## Claude's Discretion

- GRD-14 / DOCS-11 bookkeeping mechanics (no "record-only" verb in `gsd-tools windows`; report upstream; rows stay `open`)
- Data-file location/format for the shared required-contexts list; mutation-log file name
- Exact wording for the reworded requirement, the RELEASE.md paragraph, the census pattern, the SECURITY.md sentence
- Whether the ruleset change is applied via UI or the executor-prepared `gh api` call

## Deferred Ideas

- Local pre-push `web/build` cleanliness refusal
- Pinning the shadcn-svelte registry snapshot the drift monitor compares against
- Gating the `window.__codegraphFileGraph*` debug globals behind a build flag (Phase 1 R-01-14)
