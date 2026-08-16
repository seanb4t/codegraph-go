# Phase 6 Live Verification Ledger

This ledger records every live (non-static) verification this phase depends on, each with a
machine-checkable token. Created by 06-04 (BENCH-03); appended by 06-05 (MEM-02 file half) and
06-06 (MEM-02 store half).

## BENCH-03

`BENCH03_STATUS=pending-no-ci-run` — no successful `bench.yml` `publish`-job run exists for this
branch; recorded pending rather than closed by implication (review HIGH 5's fix — a local
measurement satisfies BENCH-01's need for numbers but is not evidence that `bench.yml` ran).

Decided at Task 3's blocking `checkpoint:decision`, resolved by the maintainer as
`pending-no-ci-run` (option `pending-no-ci-run`, not `dispatch-now` or `closed-by-ci-run`).

**Reason, verified before and independently of the checkpoint decision:**

- The branch this plan executes on (`gsd/v0.11.0-standalone-project-identity`) is **not pushed to
  origin**: `git ls-remote --exit-code --heads origin gsd/v0.11.0-standalone-project-identity`
  returns non-zero (verified twice — once by the orchestrator before the decision was put to the
  maintainer, once independently by this plan's executor at Task 4).
- `gh run list --workflow=bench.yml --limit 5 --json databaseId,conclusion,event,headBranch,createdAt`
  returns only runs that predate this phase:

  ```json
  [{"conclusion":"success","createdAt":"2026-08-10T07:42:21Z","databaseId":31366949399,"event":"schedule","headBranch":"main"},{"conclusion":"success","createdAt":"2026-08-03T09:46:49Z","databaseId":30802859875,"event":"schedule","headBranch":"main"},{"conclusion":"success","createdAt":"2026-08-02T17:29:56Z","databaseId":30758999046,"event":"workflow_dispatch","headBranch":"gsd/v1.0-drop-in-parity-human-ux"},{"conclusion":"success","createdAt":"2026-08-02T16:06:24Z","databaseId":30755875993,"event":"workflow_dispatch","headBranch":"gsd/v1.0-drop-in-parity-human-ux"},{"conclusion":"success","createdAt":"2026-08-02T15:57:48Z","databaseId":30755536282,"event":"workflow_dispatch","headBranch":"gsd/v1.0-drop-in-parity-human-ux"}]
  ```

  The newest run (`31366949399`) executed the OLD `headtohead` job that 06-02 deleted — it is not
  evidence for BENCH-03, and no run's `headBranch` matches this phase's branch.
- Because the branch is unpushed, no `https://github.com/<owner>/<repo>/actions/runs/<id>` URL
  exists that could truthfully be recorded, so the `closed-by-ci-run` option's
  `<evidence-required>` clause cannot be satisfied.
- The maintainer explicitly declined to push the branch or dispatch a run at this time
  (`dispatch-now` was not selected).

**What remains to close BENCH-03:** push this branch (or land it), dispatch `bench.yml` with
`job=publish`, inspect the resulting run's job summary (absolute per-corpus table present),
confirm the `publish-results` artifact uploaded (`if-no-files-found: error` did not trip), confirm
no step in the job installed or invoked any implementation other than the freshly-built Go binary,
then append a `closed-by-ci-run` entry here with the run URL and the verbatim
`gh run view <id> --json conclusion,event,headBranch,url` output showing `conclusion` = `success`.

BENCH-01 is unaffected by this status: the local publish-mode measurement committed in this plan's
Task 1 (`.planning/phases/06-benchmark-de-coupling-memory-sweep/06-PUBLISH-RESULTS.json`) supplies
BENCH-01's numbers independently of whether `bench.yml` has run.

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
