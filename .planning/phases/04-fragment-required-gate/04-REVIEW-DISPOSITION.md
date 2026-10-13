---
phase: 04
review: 04-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: open
    title: "`CHANGELOG_EXEMPT_MARKER` backtracks cubically; a crafted PR body hangs the gate"
  - id: WR-02
    severity: warning
    disposition: open
    title: "The metadata-fetch and file-list-shape fail-closed branches have no test"
  - id: WR-03
    severity: warning
    disposition: open
    title: "Workflow-command escaping of the exemption reason (attacker-controlled PR body) is untested"
  - id: IN-01
    severity: info
    disposition: open
    title: "Marker semantics can surprise: the first marker decides, and quoted examples exempt"
  - id: IN-02
    severity: info
    disposition: open
    title: "Any added file under `.changes/unreleased/` counts as a fragment"
  - id: IN-03
    severity: info
    disposition: open
    title: "`LISTED_PATH_LIMIT` is defeated by the trailing per-path loop"
  - id: IN-04
    severity: info
    disposition: open
    title: "`fetch_changed_files` claims \"Never raises\", but `_run_gh` can raise"
  - id: IN-05
    severity: info
    disposition: open
    title: "Stale scaffolding and formatting leftovers"
  - id: IN-06
    severity: info
    disposition: open
    title: "The `OK` check in `check:fragment-policy` is not anchored to the result line"
open: 9
total: 9
recorded: 2026-10-13T16:34:50.698Z
---

# Phase 04: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | - |
| WR-02 | warning | open | - |
| WR-03 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |
| IN-05 | info | open | - |
| IN-06 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
