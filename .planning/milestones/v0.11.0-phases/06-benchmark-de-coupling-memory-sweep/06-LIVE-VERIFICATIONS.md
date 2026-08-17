# Phase 6 Live Verification Ledger

This ledger records every live (non-static) verification this phase depends on, each with a
machine-checkable token. Created by 06-04 (BENCH-03); appended by 06-05 (MEM-02 file half) and
06-06 (MEM-02 store half).

## BENCH-03

`BENCH03_STATUS=closed-by-ci-run https://github.com/seanb4t/codegraph-go/actions/runs/31973967130`

Verbatim `gh run view 31973967130 --json conclusion,event,headBranch,url`:

```json
{"conclusion":"success","event":"workflow_dispatch","headBranch":"gsd/v0.11.0-standalone-project-identity","url":"https://github.com/seanb4t/codegraph-go/actions/runs/31973967130"}
```

Dispatched 2026-08-16 after the maintainer pushed the branch, with
`gh workflow run bench.yml --ref gsd/v0.11.0-standalone-project-identity -f job=publish`.
Job `benchmark publish (BENCH-01/BENCH-03, non-blocking)` (id 95230550626) concluded `success`;
every other job in the workflow was correctly `skipped`.

**The three inspection conditions, each checked rather than inferred from the green tick:**

1. **The job summary shows the absolute per-corpus figures.** The `Publish results to job summary`
   step emits `publish-results.json` VERBATIM into `$GITHUB_STEP_SUMMARY` (a `cat` inside a fenced
   JSON block — not a rendered markdown table; see the wording note below), so the summary's
   content is byte-identical to the uploaded artifact. That artifact was downloaded and read: two
   records, `weft-go` and `cockroachdb-pebble`, each with `"subject": "go"`, absolute
   `files_per_sec` / `bytes_per_sec` / `query_latency_median_ms` / `peak_rss_bytes` /
   `cold_start_ms`, all positive, on runner `namespace-profile-linux-amd64-4x8`. No comparison
   column, no second subject, no ratio.
2. **The raw-results artifact uploaded without tripping `if-no-files-found: error`.** The workflow
   confirms `if-no-files-found: error` is configured (`.github/workflows/bench.yml:154`), and the
   artifact `publish-results` finalized and downloads successfully. A configured-but-untripped
   fail-loud setting plus a real downloadable artifact is the positive assertion here — not merely
   the absence of an error.
3. **No step installed or invoked any implementation other than the freshly-built Go binary.** The
   job's complete step list is: Set up job, Set up runner, Checkout, Set up Go, Cache Go modules
   and build, Build Go codegraph binary, Run benchmark, Publish results to job summary, Upload raw
   results artifact, plus post-cleanup. A scan of the full run log for
   `npm|npx|pnpm|yarn|node_modules` returns zero against a positive-controlled instrument (the same
   log yields 49 `codegraph` hits, so the zero is not vacuous). There is no Node/npm setup step
   because `tools/bench/publishcheck` is pure Go — 06-01 built it precisely so this job needs no
   Node toolchain.

**Wording note (recorded rather than smoothed over).** Condition 1 was originally phrased as "an
absolute per-corpus **table**". The summary step publishes a fenced JSON block, not a rendered
markdown table. The substance the requirement asks for — absolute per-corpus figures for both
corpora with no second implementation — is published; the word "table" was verification-checklist
phrasing, and neither BENCH-03's requirement text (`bench.yml` runs and publishes the absolute
numbers without invoking another implementation) nor Phase 6's Success Criterion 3 uses it.

**History of this token.** It previously carried the `pending-no-ci-run` value (written without its
key here so the acceptance gate's token count stays at one), resolved that way
by the maintainer at 06-04 Task 3's blocking `checkpoint:decision`: at that time the branch was
unpushed (`git ls-remote --exit-code --heads origin <branch>` non-zero, verified twice) and
`gh run list --workflow=bench.yml` returned only runs predating this phase, the newest of which
(`31366949399`, `schedule`, `main`, 2026-08-10) executed the OLD `headtohead` job that 06-02
deleted. No truthful run URL existed, so the `closed-by-ci-run` option's `<evidence-required>`
clause could not be satisfied and the requirement was recorded pending rather than closed by
implication (review HIGH 5's fix). The maintainer subsequently pushed the branch and authorised the
dispatch, which supplied the missing evidence. Exactly one status token is carried, per the
acceptance gate — this entry REPLACES the pending one rather than sitting alongside it.

BENCH-01 was never dependent on this status: the local publish-mode measurement committed in
06-04 Task 1 (`06-PUBLISH-RESULTS.json`) supplies BENCH-01's numbers independently of whether
`bench.yml` has run. The CI run corroborates them on different hardware rather than replacing them.

## MEM-02 — file half

`MEM02_FILES_STATUS=verified-fresh-session`

**Evidence:** the live observation this token requires was performed during the phase's UAT
(`06-UAT.md` test 1, result `pass`). That testing session was itself a genuinely fresh post-`/clear`
session: it assembled its own startup context (`.claude/CLAUDE.md` plus the harness-injected
`.planning/` content) and read it, which is exactly what 06-05 Task 3's `<human-check>` asks for and
exactly what the executor session could not do for itself. The observation recorded: `.claude/CLAUDE.md`
states the Compatibility constraint is retired (v0.11.0, 2026-08-13) and that behavior is defined by
codegraph-go's own requirements and frozen goldens; Core Value is a standalone claim; no present-tense
port/parity/drop-in framing appears, and the surviving references to another implementation are
past-tense historical or explicitly retired.

**Independently re-derived at verification time**, rather than transcribed from the UAT record:
`CLAUDE_MD_FRAMING_TOTAL=0` and `STATE_MD_RETIRED_CORE_VALUE=0` re-run verbatim from 06-05 Task 3's
gate, with the instrument positive-controlled first against a planted line-wrapped fixture (it
returns 2 there, so the zero is a real absence and not a dead pattern). The three residual
`reimplement` hits in `.claude/CLAUDE.md` were read in context and are not project framing: line 17
is the past-tense `NOTICE` attribution clause (D-13 KEEP class) and lines 83/85 are the
parser-runtime sense of the word in the tree-sitter Option C discussion.

**History of this token.** It previously carried the `pending-fresh-session-not-performed` value
(written without its key here so exactly one status token is carried), because the 06-05 executor
ran inside the single session that made the edits and could not spawn an independent fresh session
of itself; the census and diff it did run were an inference, not the live read. The UAT session
supplied the missing live observation. This entry REPLACES the pending one rather than sitting
alongside it.

## MEM-02 — store half

`MEM02_STORE_STATUS=accepted-by-d15-evidence-standard`

The fresh-session recall check was not performed — this orchestrator session cannot spawn an
independently fresh session of itself, and a worktree-spawned executor carries no `mcp__engram__*`
tools (engram is installed at user scope, not in this repo's `.mcp.json`), which is what halted
06-06's first execution attempt.

The sweep is therefore accepted on the D-15 evidence standard rather than claimed as
fresh-session-demonstrated: the four corrections were applied by `supersede_memory`, and both the
`superseded_by` stamp and the cleanliness of recall were confirmed by RE-QUERY after the writes
(`get_memory` on a superseded target; `search_memory` phrased in the superseded records' own terms
returning only the correcting records). See `## Supersedes executed (MEM-01)` in
`06-MEMORY-SWEEP.md`.

Marked by the standard actually met, not described as demonstrated.

**UAT outcome, recorded as corroboration and explicitly NOT as an upgrade (maintainer ruling).**
`06-UAT.md` test 2 is recorded `skipped`, not `pass`: the maintainer cannot query the engram store
directly, so the human judgment that checkpoint asks for was never rendered. An agent-run
adversarial probe during that session — a `search_memory` phrased in the SUPERSEDED records' own
vocabulary returning all four correcting records (`gxwkk3necn`, `xj1stbrsw6`, `b9wjge7375`,
`mw5z9s9bft`) and none of the four superseded originals, plus `get_memory("3ekc84hbqt")` showing the
original intact with `superseded_by` stamped — re-confirms the same D-15 re-query evidence this
phase already accepted. It establishes nothing stronger, and this token is NOT promoted to
`verified-fresh-session` on the strength of it.
