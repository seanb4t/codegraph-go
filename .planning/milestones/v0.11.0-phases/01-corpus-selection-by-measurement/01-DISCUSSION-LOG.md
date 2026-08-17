# Phase 1: Corpus Selection by Measurement - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-13
**Phase:** 1-Corpus Selection by Measurement
**Areas discussed:** Measurement instrument, Measurement record shape, Corpus scope & fetch, Coverage bar & fallback

---

## Measurement instrument

### Q1 — What builds the per-edge-kind counts?

Scouting found that nothing in the tree produces them: `internal/cli/status.go` renders "Nodes by Kind" and there is no edge aggregation anywhere in the CLI.

| Option | Description | Selected |
|--------|-------------|----------|
| Spike tool in `tools/` | New `tools/corpusmeasure` opening the Pebble store, emitting JSON. Mirrors `tools/bench` and `testdata/golden/gocapture` precedent; no product surface touched. Was flagged as recommended for smallest blast radius. | |
| Extend `codegraph status` | Add "Edges by Kind" alongside "Nodes by Kind". Most discoverable and reusable — but a user-facing capability change in a phase scoped to measurement + fetch, putting a CLI transcript at risk right before a re-freeze phase. | ✓ |
| Both — tool now, status later | Spike tool for Phase 1; file the `status` extension as a deferred idea. | |

**User's choice:** Extend `codegraph status`
**Notes:** The scope-creep tension was raised in the option description and the user chose product surface anyway. Treated as decided; the follow-up questions then managed the resulting blast radius rather than relitigating.

### Q2 — How wide does the new section go?

| Option | Description | Selected |
|--------|-------------|----------|
| All three surfaces | Human text + `--json` + MCP resource. Measurement record then comes from `status --json` with no bespoke parsing. Cost: re-freeze `resources-read-status.golden` inside Phase 1, which the roadmap sequenced Phase 2/3 to own. | ✓ |
| Human + `--json` only | Leave the MCP status resource untouched — no wireoracle transcript churn in Phase 1. Cost: MCP and CLI status views diverge. | |
| `--json` only | Smallest diff, zero human-transcript churn. Cost: a section nobody can see without `-j`. | |

**User's choice:** All three surfaces
**Notes:** Phase 1 now explicitly owns a wire-oracle transcript re-freeze that the roadmap's phase ordering did not anticipate. Recorded in CONTEXT.md as D-02 so planning treats it as known scope rather than a surprise.

### Q3 — What to do about `StatusResult.FilesByLanguage`?

Discovered mid-discussion: the field already exists at `internal/query/status.go:57` but carries `json:"-"`, suppressing it to match an output shape retired by this very milestone on 2026-08-13.

| Option | Description | Selected |
|--------|-------------|----------|
| Un-suppress it, same diff | Drop `json:"-"` alongside adding `edgesByKind`; both halves of FIXT-01's data come from one call. On-thesis — the constraint justifying the suppression was formally retired. | ✓ |
| Un-suppress and drop `languages` | Also remove the now-redundant `languages []string`. Cleanest end state, but a breaking JSON removal. | |
| Leave suppressed | Spike reads language counts from the store directly. Keeps the JSON contract frozen, at the cost of two code paths for one record. | |

**User's choice:** Un-suppress it, same diff
**Notes:** `languages []string` is retained as redundant-but-harmless. Its removal was moved to deferred ideas.

### Q4 — Dense or sparse `edgesByKind`?

Framed against rule `84d1gfpywd`: if a kind can be omitted, "absent" and "measured zero" become indistinguishable — which is exactly what FIXT-01 criterion 3 turns on.

| Option | Description | Selected |
|--------|-------------|----------|
| Dense, derived from `RANK_EDGES` | Always all 9 keys including zeros, key set derived from the constant, key-set equality test. "Absent" impossible by construction. | ✓ (modified) |
| Dense, hand-listed | Always 9 keys, enumerated in the aggregator. Introduces a second list that can drift from `RANK_EDGES`. | |
| Sparse — non-zero kinds only | Smallest JSON, consistent with `nodesByKind`. Cost: the exact ambiguity the coverage criterion must resolve. | |

**User's choice:** Free text — *"1 - but opt in, perhaps? only at a verbose, or 'include all edges' option so general use which doesn't need the absolute clarity isn't killed with a bunch of zeros?"*
**Notes:** Synthesis of options 1 and 3 rather than a straight pick. Sparse becomes the default for readability; dense (derived key set, explicit zeros, key-set equality test) sits behind an opt-in flag, and the measurement run uses the flag. Rigor available on demand, not imposed on the default reading experience.

### Q5 — What does the argument-less MCP status resource emit?

| Option | Description | Selected |
|--------|-------------|----------|
| Sparse — match the default | Resource mirrors flagless CLI behavior; density stays a deliberate CLI-side act. Keeps the transcript re-freeze small. | ✓ |
| Dense — agents get the unambiguous view | An agent cannot pass a flag, and absent-vs-zero is worse for a machine reader. Cost: CLI-default and MCP diverge and need documenting. | |
| Sparse, and skip `edgesByKind` on MCP entirely | No transcript re-freeze at all; revisits the Q2 answer. | |

**User's choice:** Sparse — match the default

---

## Measurement record shape

### Q1 — What form does the committed record take?

| Option | Description | Selected |
|--------|-------------|----------|
| JSON data + generated prose | Machine-readable measurements keyed `repo@SHA` are source of truth; markdown is generated from them. Prose cannot drift from numbers. | ✓ |
| Hand-written markdown only | Simplest and most readable, matching prior spike-result records. Cost: numbers become prose on transcription; nothing downstream can consume them. | |
| JSON only, no prose doc | Zero drift surface. Cost: the *reasoning* has nowhere to live but commit messages. | |

**User's choice:** JSON data + generated prose

### Q2 — Frozen once, or re-derivable?

| Option | Description | Selected |
|--------|-------------|----------|
| Re-runnable target + drift guard | Target regenerates the JSON; a test asserts the coverage claim still holds. A SHA bump that kills `type_of` coverage fails loudly. Positive-assertion posture per rule `84d1gfpywd`. | ✓ |
| Re-runnable target, no guard | Reproducible but checked by review. Leans on discipline — the same bet the milestone made on VOCAB-01. | |
| Frozen once — spike output only | Honest snapshot. Cost: nothing detects a coverage regression later. | |

**User's choice:** Re-runnable target + drift guard

### Q3 — When does the guard re-measure vs check the record?

| Option | Description | Selected |
|--------|-------------|----------|
| Cheap always + re-measure on pin change | Every CI run asserts the committed JSON satisfies coverage (no indexing); a path-filtered job re-indexes only when pins change. | ✓ |
| Re-measure on every CI run | Truly live, and doubles as proof corpora were fetched. Cost: indexing `apache/arrow` on every push. | |
| Cheap always + manual re-measure | Lowest CI cost. Cost: a pin bump with stale numbers passes CI — the exact silent-narrowing failure Phase 1 exists to prevent. | |

**User's choice:** Cheap always + re-measure on pin change

### Q4 — Where do the pinned SHAs live?

| Option | Description | Selected |
|--------|-------------|----------|
| One manifest, record references it | Single manifest (repo, SHA, license, `locked`) as sole pin authority, read by Taskfile target, cache key and guard. Rejected candidates stay marked unlocked — measured, not fetched. | ✓ |
| Record IS the manifest | Fewest files, zero cross-file drift. Cost: an evidence file becomes hand-edited, and the fetch path depends on it. | |
| Pins in Taskfile vars | Conventional for a task-driven repo. Cost: two copies of each SHA, so the guard must assert they agree. | |

**User's choice:** One manifest, record references it

---

## Corpus scope & fetch

### Q1 — Whole repository or pinned subtree?

| Option | Description | Selected |
|--------|-------------|----------|
| Subtree-capable, whole-repo default | Optional `subtree` lets a monorepo contribute just its language slice; bounds fetch/index time. Cost: `subtree` joins the reproducibility claim. | |
| Whole repo always | `repo@SHA` is the entire input — nothing to get wrong. Cost: indexing all of `apache/arrow` for C#; GB-scale cache entries; may force dropping arrow. | ✓ |
| Subtree required for every entry | Maximally deliberate. Cost: more ceremony; a vanished subtree becomes a fetch failure. | |

**User's choice:** Whole repo always
**Notes:** Consequence flagged immediately and accepted — this likely rules `apache/arrow` out on size, making the search for a dedicated C# corpus part of Phase 1's job.

### Q2 — How does the fetch work?

Prefaced with the observation that codegraph-go gained git/worktree awareness in v1.0 (`internal/gitmeta`, `StatusResult.WorktreeMismatch`), so whether a corpus carries a `.git` directory can change indexing behavior — and therefore the goldens Phase 2 freezes.

| Option | Description | Selected |
|--------|-------------|----------|
| Shallow git fetch at SHA | `git init` + `fetch --depth 1 origin <sha>` + checkout. Tree carries a real `.git`, so corpora exercise the git-aware path as real users do. | ✓ |
| Tarball from codeload | Fastest, smallest, no git dependency, content-addressed. Cost: no `.git` — the git-aware path goes unexercised and the goldens inherit the gap. | |
| Shallow clone, then strip `.git` | Git SHA guarantee with a pure-source indexed input. Cost: most moving parts; the strip step joins the reproducibility check. | |

**User's choice:** Shallow git fetch at SHA

### Q3 — Where do fetched corpora land?

| Option | Description | Selected |
|--------|-------------|----------|
| Gitignored path in-tree | Tests resolve by relative path, `git status` stays clean, one `rm -rf` resets. Cost: a multi-GB gitignored directory inside the working tree. | |
| `$HOME` cache dir | Shared across worktrees; keeps the repo small. Cost: tests need a resolver; invisible from inside the repo. | |
| Env var, default in-tree | Override via `CODEGRAPH_CORPUS_DIR`, echoing the existing `CODEGRAPH_{JAVA,PYTHON,TSJS}_CORPUS` convention. | ✓ (modified) |

**User's choice:** Free text — *"3, default to XDG_CACHE_DIR relative"*
**Notes:** Took the env-var-override structure but overrode the default: `${XDG_CACHE_HOME:-$HOME/.cache}/…` rather than in-tree. Keeps the working tree small and shares one fetch across worktrees.

### Q4 — How is the CI cache keyed?

Prefaced with the constraint that GitHub gives a repo 10 GB of total cache with LRU eviction, so granularity decides whether one pin bump evicts everything else. No workflow in this repo uses `actions/cache` today.

| Option | Description | Selected |
|--------|-------------|----------|
| One entry per corpus, keyed `repo@sha` | Bumping one pin refetches only that corpus. Content-addressed, so no `restore-keys` fallback is needed — a prefix match on a SHA key would be wrong by definition. | ✓ |
| Single entry keyed on manifest hash | Simplest YAML, atomic. Cost: any pin bump refetches everything; one blob likelier to hit the 10 GB ceiling. | |
| Per corpus, with `restore-keys` prefix | Faster incremental fetches. Cost: a partial restore may hold the *wrong* SHA's tree — a real correctness burden. | |

**User's choice:** One entry per corpus, keyed `repo@sha`

---

## Coverage bar & fallback

### Q1 — Is "non-zero" really the bar?

| Option | Description | Selected |
|--------|-------------|----------|
| Non-zero, but record the count | Keep the roadmap's bar literally; surface actual numbers so thin coverage is visible rather than hidden behind a boolean. | |
| Threshold per kind | A kind needs N+ edges from a named repo. Defends against a corpus emitting one freak `overrides` edge. Cost: N is invented; tightens a criterion the roadmap fixed. | ✓ |
| Non-zero, plus a distinct-file bar | Non-zero edges spanning 2+ source files — targets the real failure without inventing a magnitude. Cost: a second dimension to compute and assert. | |

**User's choice:** Threshold per kind
**Notes:** A deliberate tightening. A suite clearing a threshold also clears non-zero, so FIXT-01's stated criteria remain satisfied.

### Q2 — How is the threshold set?

| Option | Description | Selected |
|--------|-------------|----------|
| Set N from measured data, then freeze | Measure first, choose N against real distributions, freeze into the guard. Cost: N is chosen by the pass it judges, so the rationale must be written down. | ✓ |
| Uniform N = 10, fixed up front | One number, maximally simple. Cost: a legitimately-rare kind landing at 6 blocks the phase on an invented number. | |
| Tiered: strict for the risky two | Threshold for `overrides`/`type_of`, non-zero for the rest. Strictness exactly where the documented failure mode is. Cost: two rules. | |

**User's choice:** Set N from measured data, then freeze

### Q3 — What if a kind still misses across every candidate?

Framed against the documented trap: v1.0 Phase 1 found this repo's own idiomatic Go produces zero `overrides` and `type_of` edges.

| Option | Description | Selected |
|--------|-------------|----------|
| Extend the behavioral corpus deliberately | Add a targeted case to `corpus/behavioral/` and record the kind as synthetically covered. Guaranteed to close the gap, reuses a kept asset. Cost: weaker evidence than real-world coverage. | |
| Keep searching for a real corpus | Strongest evidence. Cost: unbounded search on a blocking phase, with Phases 2 and 3 stalled behind it. | |
| Real corpus first, behavioral as recorded fallback | Bounded additional search; fall back with the shortfall recorded. Cost: "bounded" needs a stated limit. | ✓ |

**User's choice:** Real corpus first, behavioral as recorded fallback

### Q4 — What bounds the search?

| Option | Description | Selected |
|--------|-------------|----------|
| Candidate count | A fixed number of additional candidates get measured; the record names every repo tried and what it scored, so failures become evidence. | ✓ |
| One extra round, then fall back | Two measurement passes total — easy to plan and review as two diffs. | |
| Fall back immediately on the first miss | Fastest resolution of a blocking phase. Cost: gives up on real-world coverage the moment it gets hard. | |

**User's choice:** Candidate count
**Notes:** The specific count was not fixed and is left to planning.

---

## Todo cross-reference

Four pending todos matched Phase 1. Presented with the recommendation to fold none.

**User's choice:** None — record as reviewed

| Todo | Score | Area | Disposition |
|------|-------|------|-------------|
| Add golangci-lint with gofmt and idiomatic Go linters | 0.9 | ci | Reviewed, not folded — keyword match, not subject match |
| `release:dry-run-signed` additions-only diff guard passes vacuously | 0.6 | release | Reviewed, not folded — same defect class as rule `84d1gfpywd`, wrong subsystem |
| brew trust instructions recommend the broader `--tap` grant | 0.4 | docs | Reviewed, not folded — unrelated |
| `post-release-verify.yml` conclusion guard has no test asserting it | 0.3 | ci | Reviewed, not folded — vacuous-guard class, wrong subsystem |

---

## Claude's Discretion

No area was deferred wholesale. Left open by an explicit "next area" and therefore open to the planner:

- The dense-mode flag's name; whether `nodesByKind` gains the same opt-in density.
- Manifest format (YAML vs JSON) and path; where the generated prose lands; whether the prose is regenerated-and-diffed in CI.
- What "same tree twice" is verified by; whether the fetch target needs `--force`.
- The exact extra-candidate count, and the exact N (data-driven by construction).
- Whether the 5 priority-4 languages get a file-count threshold or stay at non-zero.
- Whether a behavioral-corpus fallback covers a *language* miss as well as an edge-kind miss.

## Deferred Ideas

- **Dense `nodesByKind`** — symmetry with `edgesByKind`; raised and deliberately left.
- **Dropping `StatusResult.Languages`** — offered and not chosen; a breaking JSON removal deserving its own scope.
- **A file-count threshold for the 5 priority-4 languages** — the threshold decision covered edge kinds only.
