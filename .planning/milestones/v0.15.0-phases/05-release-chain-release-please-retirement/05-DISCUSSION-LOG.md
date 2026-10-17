# Phase 5: Release Chain & release-please Retirement - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-13
**Phase:** 5-Release Chain & release-please Retirement
**Areas discussed:** Tag trigger & idempotency, Release PR identity & body (cut short), then collapsed to the documented changie pattern

---

## Tag trigger & idempotency

| Option | Description | Selected |
|--------|-------------|----------|
| State-based | Every push to main: tag `changie latest` if untagged | ✓ |
| Diff-based | Tag only when the push added a `.changes/vX.Y.Z.md` | |
| Release-PR-based | Look up the pushed commit's PR via the API | |

**User's choice:** State-based. Later superseded by D-06 (documented `paths: [CHANGELOG.md]` trigger plus skip-if-tag-exists).

| Option | Description | Selected |
|--------|-------------|----------|
| Commit that added the file | `git log --diff-filter=A` on `.changes/vX.Y.Z.md` | ✓ |
| The pushed HEAD | `github.sha` | |

**User's choice:** Commit that added the file. Later superseded by D-06 — with the path trigger the pushed commit is the release merge.

| Option | Description | Selected |
|--------|-------------|----------|
| Both paths | pretag-gate gates the release-PR job and the tag job | ✓ |
| Release-PR path only | Literal REL-10 | |

**User's choice:** Both paths (D-07, kept).

---

## Release PR identity & body

| Option | Description | Selected |
|--------|-------------|----------|
| By path | `.changes/` prefix in both hygiene allowlists | |
| By App author | Exempt the App's `[bot]` login | |
| By body markers | Workflow-written exemption markers | |

**User's choice (free text):** "why is this so complicated?" — then: "you've asked a bazillion questions about how to do something that I'm pretty sure is really well doc'd? isn't there an action for this and a general approach? Why are we making a space shuttle?"
**Notes:** Claude fetched changie's docs (Context7 `/miniscruff/changie`: release trigger, GoReleaser integration) and changie-action's `gen-release-pr.yml`, and proposed adopting that documented pattern wholesale (`create-pull-request` with the App token; `paths: [CHANGELOG.md]` tag trigger with skip-if-tag-exists), with every remaining choice at Claude's discretion. The user chose "Lock it, write CONTEXT". Path-based recognition (D-05) is Claude's call under that lock.

---

## Claude's Discretion

- Release PR lifecycle (fixed branch regenerated each run, body = rendered notes, opened non-draft), workflow file layout and name, `create-pull-request` inputs, Taskfile target names.
- Census mechanics (plain grep, positive control, `.changes/v*.md` counted as CHANGELOG history, test rename) and where the 45 RELEASE-PROCEDURES verdicts are recorded.
- REL-13 rehearsal input.

## Deferred Ideas

- Closing an orphaned release-please PR at Phase 6 entry, if one is open when the milestone merges.
