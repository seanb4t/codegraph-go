---
phase: 05
review: 05-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: open
    title: "`release-pr` leaves a stale open release PR when the fragment count drops to zero"
  - id: WR-02
    severity: warning
    disposition: open
    title: "Path-based hygiene exemption fails open on PRs with more than 100 files"
  - id: WR-03
    severity: warning
    disposition: open
    title: "`pr_template_policy.emit` writes the attacker-controlled exemption reason to stderr unescaped"
  - id: IN-01
    severity: info
    disposition: open
    title: "`CONTRIBUTING.md` and `.changie.yaml` give a fragment command that blocks without a TTY"
  - id: IN-02
    severity: info
    disposition: open
    title: "`check:changie` leg 14 silently depends on fragments left behind by legs 4-13"
  - id: IN-03
    severity: info
    disposition: open
    title: "`docs/RELEASE-PROCEDURES.md` §2 still lists a hand-pushed release candidate as a normal tag shape"
  - id: IN-04
    severity: info
    disposition: open
    title: "The \"linear-time\" claim for the exemption markers is not accurate"
  - id: IN-05
    severity: info
    disposition: open
    title: "The tag job keeps the App token in `.git/config` while `go tool` runs repository-pinned third-party code"
open: 8
total: 8
recorded: 2026-10-14T22:48:46.098Z
---

# Phase 05: Code Review Disposition

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

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
