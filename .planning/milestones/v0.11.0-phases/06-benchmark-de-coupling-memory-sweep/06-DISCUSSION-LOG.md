# Phase 6: Benchmark De-coupling & Memory Sweep - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-16
**Phase:** 6-Benchmark De-coupling & Memory Sweep
**Areas discussed:** Historical artifacts, bench.yml job shape, Corpus manifest, Memory sweep

---

## Area selection

| Option | Description | Selected |
|--------|-------------|----------|
| Historical artifacts | Disposition of the 6 headtohead JSONs, BENCHMARKS.md's comparison tables, BASELINE.md's narrative | ✓ |
| bench.yml job shape | The headtohead job is the only absolute-numbers publisher and it npm-installs the TS binary | ✓ |
| Corpus manifest | realcorpus pins colbymchenry-codegraph and pebble; internal/corpora holds the locked set | ✓ |
| Memory sweep | What counts as retired framing, and what instrument proves the sweep complete | ✓ |

**User's choice:** All four areas.

---

## Historical artifacts

### Q1 — Disposition of the 6 committed `tools/bench/headtohead-*.json` captures

| Option | Description | Selected |
|--------|-------------|----------|
| Retain, move to archive dir | Move to tools/bench/archive/ with a README marking them dated parity-era measurements; preserves real history in the spirit of MEM-01's supersede-don't-delete | |
| Delete them | Remove entirely; git history still holds them | ✓ |
| Retain in place, unchanged | Leave as data files; nothing publishes them once BENCHMARKS.md drops the multipliers | |

**User's choice:** Delete them.
**Notes:** Mirrors the Phase 5 `codegraph migrate` precedent — remove outright rather than reframe. Git history is the archive.

### Q2 — How to rework `docs/BENCHMARKS.md`

| Option | Description | Selected |
|--------|-------------|----------|
| Rewrite from scratch | New document on the project's own terms; largest diff; risks losing methodology prose | ✓ |
| Surgical section removal | Delete comparison sections and retitle, rest byte-untouched; smallest diff; risks residual woven framing | |
| Rewrite, mine the old for methodology | New structure, methodology sections explicitly carried forward re-authored | |

**User's choice:** Rewrite from scratch.
**Notes:** Claude flagged a consequence at the time: the document's absolute figures currently live *inside* the comparison tables, so a from-scratch rewrite creates a dependency on a fresh absolute-only measurement run. Recorded in CONTEXT.md under D-02 as a plan-level task dependency.

### Q3 — Is `tools/bench/BASELINE.md` in scope?

| Option | Description | Selected |
|--------|-------------|----------|
| In scope — sweep the framing | Live engineering documentation; sweep comparison references, keep the guard rationale intact | ✓ |
| Out of scope — historical record | A dated investigation log; rewriting falsifies the record | |
| In scope, and split it | Promote the load-bearing guard rationale, archive/delete the dated narrative | |

**User's choice:** In scope — sweep the framing.
**Notes:** Its guard rationale explains why `CheckRegression`'s platform / runner-class / `scratch_fs` refusals exist, which is load-bearing for BENCH-02.

### Q4 — How the in-tree sweep is proven complete

| Option | Description | Selected |
|--------|-------------|----------|
| Positive-controlled multiline census | `rg -U`, bounded patterns, planted-phrase positive control before trusting any zero, `rg -o \| wc -l` | ✓ |
| Bare backstop census, no exclusions | Unfiltered whole-tree census, every hit classified by hand — the technique that found "variant four" in Phase 5 | |
| Both — targeted gate plus backstop | Durable gate plus a one-time bare backstop | |

**User's choice:** Positive-controlled multiline census.
**Notes:** Directly applies memory `hakm8j156e` (line-based `rg` missed 34 comment-wrapped occurrences) and rule `84d1gfpywd` (guards need a positive assertion).

---

## bench.yml job shape

### Q1 — Restructuring the `headtohead` job

| Option | Description | Selected |
|--------|-------------|----------|
| Rename + strip the 3 lines | Rename the job, delete Node setup / npm install / -ts-binary; preserves runner pinning and D-13 by construction | |
| Delete job, build fresh publisher | Remove entirely, author a new absolute-numbers job; no residual framing, but must re-derive hard-won properties | ✓ |
| Fold into the regression path | Have the gated regression invocation publish; couples publish to a gate on a different runner profile | |

**User's choice:** Delete job, build fresh publisher.
**Notes:** Claude flagged that several load-bearing properties exist only in the deleted job's body and comments, which motivated the follow-up question below.

### Q2 — Properties the fresh publisher must carry forward (multi-select)

| Option | Description | Selected |
|--------|-------------|----------|
| Runner pinning + env contract | namespace-profile-linux-amd64-4x8 + CODEGRAPH_BENCH_RUNNER; CheckRegression refuses cross-runner comparison | ✓ |
| D-13 no-Taskfile exception | Inline `go run` keeps `-rebless` unreachable by tab-completion; a wrong-platform baseline already produced a fictitious 10.6% regression | ✓ |
| Non-blocking / publish-not-gate | A slower number must never fail CI | ✓ |
| Artifact upload + job summary | `if-no-files-found: error` positive assertion plus GITHUB_STEP_SUMMARY | ✓ |

**User's choice:** All four.
**Notes:** Converts "delete and rebuild" from an open-ended rewrite into a checkable carry-forward list.

### Q3 — Replacing the `-mode headtohead` subcommand

| Option | Description | Selected |
|--------|-------------|----------|
| New `publish` mode, delete headtohead | Single-subject absolute Metrics JSON; delete the case, the two-subject loop, and -ts-binary | ✓ |
| Keep the mode, make TS optional | Smallest code change, but a two-subject architecture surviving under another name | |
| You decide | Planner chooses shape under a locked no-two-subject constraint | |

**User's choice:** New `publish` mode, delete headtohead.

### Q4 — Proving `CheckRegression` still fires (BENCH-02)

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse the FIXT-07 protocol | Phase 3's exact discipline: pre-mutation diff gate, confirm applied, observe RED, EXIT-trap restore, byte-clean revert, committed log | ✓ |
| Reuse, plus a corrupt-baseline case | Also mutate the platform/runner/scratch_fs refusal guards; broader coverage, more cost | |
| You decide | Planner selects the mutation set under locked constraints | |

**User's choice:** Reuse the FIXT-07 protocol.
**Notes:** Consistency with a protocol already proven in this repo; its failure modes are known.

---

## Corpus manifest

**Framing note:** Claude surfaced during this area that the "two manifests need reconciling"
concern from memory `0x27xn3wvr` was already resolved by Phase 1 — `internal/corpora/manifest.go:6-15`
records the non-merge as deliberate with four reasons in `01-04-PLAN.md`. This narrowed the
area from an architecture question to a framing-and-entry question.

### Q1 — The `colbymchenry-codegraph` entry

| Option | Description | Selected |
|--------|-------------|----------|
| Drop it, keep the other two | Remove the entry and its SiblingDir; weft-go + pebble remain | ✓ |
| Drop it, add a replacement | Pin a neutral third repo to keep measurement breadth | |
| Replace corpus with the locked set | Repoint at hugo/guava/serilog/requests; drops pebble and re-opens the licence-bar question | |

**User's choice:** Drop it, keep the other two.

### Q2 — Reach of the prose rewrite

| Option | Description | Selected |
|--------|-------------|----------|
| Rewrite realcorpus + update both cross-refs | Re-author realcorpus prose AND update internal/corpora's package doc and Manifest.Note | ✓ |
| Rewrite realcorpus only | Leave internal/corpora; its comments describe a still-true architectural decision | |
| You decide | Planner determines reach under locked constraints | |

**User's choice:** Rewrite realcorpus + update both cross-refs.
**Notes:** `Manifest.Note` is a committed data field, so a stale pointer would ship inside `corpora/manifest.json`, not just in source comments.

### Q3 — The pebble / licence-bar difference

| Option | Description | Selected |
|--------|-------------|----------|
| Keep pebble, re-justify the difference | Redistribution risk for a fetch-only never-vendored corpus is a different question from FIXT-01's fixture bar | ✓ |
| Drop pebble, align the bars | Converges the policies but leaves a single-entry benchmark corpus | |
| You decide | Planner resolves under a locked no-head-to-head-justification constraint | |

**User's choice:** Keep pebble, re-justify the difference.

---

## Memory sweep

**Environment note:** the engram MCP endpoint was unreachable during this discussion
(`getaddrinfo ENOTFOUND mcp-gw.fzymgc.house`). Recorded in CONTEXT.md as D-16, a fail-loud
plan-time precondition.

### Q1 — Does the sweep extend beyond the engram spine?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — spine plus agent-facing files | MEM-02's stated outcome is defeated by CLAUDE.md's Core Value line regardless of the spine's state | ✓ |
| Spine only — files were Phases 4/5 | Literal engram-scoped reading; keeps phase boundaries clean but leaves MEM-02 unmet | |
| Spine plus a gap report | Sweep the spine, file the file findings as an explicit gap against Phases 4/5 | |

**User's choice:** Yes — spine plus agent-facing files.
**Notes:** Executes memory `myywc0y9vm`'s recorded maintainer correction. `.planning/STATE.md`'s core-value line additionally names `migrate`, a capability removed in Phase 5.

### Q2 — The supersede classification test

| Option | Description | Selected |
|--------|-------------|----------|
| Present-tense standing-fact test | Supersede only current standing-fact assertions; matches the ROADMAP note exactly | |
| Any present-tense OR forward-looking | Also supersede future-obligation records ("Phase N must maintain parity") since those steer future sessions | ✓ |
| You decide | Planner defines the test under locked constraints | |

**User's choice:** Any present-tense OR forward-looking.

### Q3 — What makes completeness checkable

| Option | Description | Selected |
|--------|-------------|----------|
| Committed enumeration + re-query evidence | Per-record verdicts plus post-sweep re-query output a reviewer can replay | |
| Full enumeration, no re-query | Verdicts for every record; the "did it land" step stays asserted | ✓ |
| Enumeration + planted-control probe | Adds a positive control that the search instrument surfaces known framing-bearing records | |

**User's choice:** Full enumeration, no re-query.
**Notes:** Claude stated the concern once — without re-query evidence the supersede landing is asserted rather than demonstrated, the vacuous-pass shape rule `84d1gfpywd` targets — and the user's selection stands. Recorded in CONTEXT.md D-15 as an accepted trade-off with the residual risk named.

### Q4 — Which engram scopes are covered

| Option | Description | Selected |
|--------|-------------|----------|
| Spine + rules + overlays | Every scope a fresh session recalls; rules are injected as MUST-follow ground truth | ✓ |
| Spine + rules only | Skip work-local overlays that get promoted or discarded when their branch lands | |
| Spine only | Literal MEM-01 reading; the 3 current rules carry no parity framing | |

**User's choice:** Spine + rules + overlays.

---

## Claude's Discretion

None. Every question presented was answered with an explicit selection; no "You decide"
option was chosen in any area.

## Deferred Ideas

- **Replacement third benchmark corpus entry** — dropping colbymchenry-codegraph narrows the
  corpus from three repos to two. A measurement-quality question, not a de-coupling question.
- **Post-sweep re-query verification** — explicitly declined (Memory sweep Q3); recorded so a
  future audit knows it was considered rather than overlooked.
- **Renaming the `realcorpus` package** — its prose is rewritten but the name stays; a rename
  would ripple through five importing files and is not required by BENCH-02.
