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
