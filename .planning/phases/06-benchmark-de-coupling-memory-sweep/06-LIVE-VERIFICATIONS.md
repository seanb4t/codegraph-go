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

1. **The job summary shows an absolute per-corpus table.** The `Publish results to job summary`
   step emits `publish-results.json` VERBATIM into `$GITHUB_STEP_SUMMARY` (a `cat` inside a fenced
   block), so the summary's content is byte-identical to the uploaded artifact. That artifact was
   downloaded and read: two records, `weft-go` and `cockroachdb-pebble`, each with
   `"subject": "go"`, absolute `files_per_sec` / `bytes_per_sec` / `query_latency_median_ms` /
   `peak_rss_bytes` / `cold_start_ms`, all positive, on runner
   `namespace-profile-linux-amd64-4x8`. No comparison column, no second subject, no ratio.
2. **The raw-results artifact uploaded without tripping `if-no-files-found: error`.** The job log
   confirms `if-no-files-found: error` is configured, and the artifact `publish-results`
   (id 9270550732) finalized at 454 bytes. A configured-but-untripped fail-loud setting plus a
   real finalized artifact is the positive assertion here — not merely the absence of an error.
3. **No step installed or invoked any implementation other than the freshly-built Go binary.** The
   job's complete step list is: Set up job, Set up runner, Checkout, Set up Go, Cache Go modules
   and build, Build Go codegraph binary, Run benchmark, Publish results to job summary, Upload raw
   results artifact, plus post-cleanup. A bounded grep of the step names for
   `node|npm|npx|pnpm|yarn|codegraph@|colbymchenry` returns zero. There is no Node/npm setup step
   because `tools/bench/publishcheck` is pure Go — 06-01 built it precisely so this job needs no
   Node toolchain.

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

`MEM02_FILES_STATUS=pending-fresh-session-not-performed`

**Reason:** the closest direct test of MEM-02's file half is starting a genuinely fresh session in
this repository and reading the startup context it assembles (`.claude/CLAUDE.md` plus whatever
the harness injects from `.planning/`) — this plan's own `<verify><human-check>` names it
explicitly and states it "cannot be run by a test." This executor runs inside the single session
that made the edits; it cannot spawn an independent fresh session of itself to observe what a new
session's own context assembly would show, so the read was not performed. Per this plan's own
reviewed design (06-05:215 — `workflow.human_verify_mode: end-of-phase` in this repo's config
means `checkpoint:human-verify` tasks are not emitted here), the status token is recorded as
explicitly pending rather than left silently unrecorded, so MEM-02's file half is never claimed as
verified on silence.

**What was performed, mechanically, in place of the fresh-session read:** every occurrence of the
bounded framing-word pattern set was enumerated and swept from `.claude/CLAUDE.md`,
`.planning/PROJECT.md` and `.planning/STATE.md` (see `06-MEMORY-SWEEP.md`), and
`CLAUDE_MD_FRAMING_TOTAL=0` / `STATE_MD_RETIRED_CORE_VALUE=0` / `CORE_VALUE_EQUAL=true` are all
proven this task. A fresh session reading these three post-edit files would not encounter the
retired framing this task exists to remove — but that is an inference from the census and diff,
not the live observation the human-check calls for.

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
