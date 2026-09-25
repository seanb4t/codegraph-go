---
id: SEED-005
status: dormant
planted: 2026-09-25
planted_during: between milestones (post v0.14.0) / gsd-explore changie
trigger_when: after codegraph-go's first changie-cut release verifies end to end (App-authored tag, release.yml fired, body == .changes/vX.md, post-release-verify green)
scope: cross-repo (router-hosts, engram, fovea, fzymgc-house-skills)
---

# SEED-005: changie release management as the template for the other repos

## Why This Matters

Every one of Sean's active repos runs release-please under squash-merge, and the
title-is-the-only-input trap has already fired twice: codegraph-go PR #21 (`q1hg0dewy6`) and
router-hosts PR #381 (`m71wm4hsar`, 107 commits proposed as 0.10.14). codegraph-go's changie
design (`.planning/notes/changie-release-management.md`) is the first repo to fix it
structurally; once proven, the same `.changie.yaml`, release workflow, and fragment gate port
with small edits.

## When to Surface

- After the trigger above, and once the GSD phase-close fragment hook question is resolved
  (research/questions.md), because the other repos also ship through GSD.
- Repo-specific deltas to expect: engram's release PR also bumps `charts/engram/Chart.yaml`
  and `plugin.json` (use changie `replacements` — they run on `merge`); router-hosts relies on
  `squash_merge_commit_message=PR_BODY` carrying a `gate_status:` trailer — unaffected, but
  verify the fragment gate's exemption marker does not collide.
