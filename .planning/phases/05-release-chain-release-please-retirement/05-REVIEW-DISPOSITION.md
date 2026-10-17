---
phase: 05
review: 05-REVIEW.md
titles: json
findings:
  - id: WR-01
    severity: warning
    disposition: open
    title: "§10(a) describes a pipeline that no longer exists and cites a step §4 does not contain"
  - id: WR-02
    severity: warning
    disposition: open
    title: "§7 forbids reusing a stable tag name, but its own rollback re-creates exactly that"
  - id: WR-03
    severity: warning
    disposition: open
    title: "§12 tap-failure remedy re-runs the full `goreleaser release`, which replaces live signed and notarized assets, while claiming \"nothing about the release needs to be touched\""
  - id: WR-04
    severity: warning
    disposition: open
    title: "§4 step 5 says only a release-PR merge fires `release-tag.yml`, but the doc elsewhere and the workflow say otherwise"
  - id: WR-05
    severity: warning
    disposition: open
    title: "Decision IDs collide within the same edit and across sections, so cross-references are ambiguous"
  - id: IN-01
    severity: info
    disposition: open
    title: "`rm -P` is BSD/macOS-only and does not securely erase on APFS or SSD"
  - id: IN-02
    severity: info
    disposition: open
    title: "Stale integration-branch name"
  - id: IN-03
    severity: info
    disposition: open
    title: "Dangling and opaque references"
  - id: IN-04
    severity: info
    disposition: open
    title: "`gh` commands omit the repository, so they target whatever checkout the maintainer is in"
  - id: IN-05
    severity: info
    disposition: open
    title: "The tag job keeps the App token in `.git/config` while `go tool` runs repository-pinned third-party code"
open: 10
total: 10
recorded: 2026-10-16T20:33:45.304Z
---

# Phase 05: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| WR-01 | warning | open | - |
| WR-02 | warning | open | - |
| WR-03 | warning | open | - |
| WR-04 | warning | open | - |
| WR-05 | warning | open | - |
| IN-01 | info | open | - |
| IN-02 | info | open | - |
| IN-03 | info | open | - |
| IN-04 | info | open | - |
| IN-05 | info | open | - (not in the current review) |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved. A `|` in the reason is kept as prose and escaped on the next run.
Re-running the gate keeps every row it can. A row the current review no longer reports is kept and its Source cell flagged, so a finding does not leave this record silently. ONE exception: when a finding id is REUSED by a different finding, the earlier decision cannot keep a row — the id is taken — and it is dropped. A RECORDED decision (anything but `open`) is named on the console when that happens; a row still at `open` is replaced silently, because `open` records no decision to lose.
