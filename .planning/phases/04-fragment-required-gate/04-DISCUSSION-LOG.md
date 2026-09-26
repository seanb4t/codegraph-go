# Phase 4: Fragment-Required Gate - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-26
**Phase:** 04-fragment-required-gate
**Areas discussed:** Trigger & trust model, Landing sequence, Path-rule home, What counts as a fragment (+ Ruleset guards, raised by the maintainer)

---

## Trigger & trust model

| Option | Description | Selected |
|--------|-------------|----------|
| pull_request_target + base checkout | Policy always from main (pr-template-format.yml model); workflow must be on main before it runs anywhere | ✓ |
| pull_request | Runs on #88 immediately; a PR can edit the gate that judges it | |

**User's choice:** pull_request_target + base checkout.

---

## Landing sequence

First attempt (rejected for clarification): two PRs (gate, then a "fixture" PR carrying the required-check list line) vs one PR with an admin-bypass merge vs merging the milestone so far to main.

**Maintainer clarification 1:** "What fixture? What are you testing? It had best not be how git, github, gh workflows, or changie _work_." Explained that the "fixture" is `.github/required-status-checks.txt` and that the gate's tests cover only our verdict function over literal inputs. Recorded as CONTEXT D-01. The framing "PR B is the first live proof the gate reports" was dropped because it would be testing GitHub.

**Maintainer clarification 2:** "why are we watching/checking rulesets or required checks? … Why are we running continual tests for that?" Explained the history: T-10-01-05 (v1.0 Phase 10) rests on a false premise, since a required check that never reports blocks PRs loudly; GRD-12's drift script only catches the maintainer's own settings edits. That led to a new question:

| Option | Description | Selected |
|--------|-------------|----------|
| Remove both in Phase 4 | Delete TestRequiredCheckNamesPreserved + helpers, check-ruleset-drift.sh + ci.yml step, required-status-checks.txt; amend GATE-05 | ✓ |
| Keep, add the one line | GATE-05 as written | |
| Keep for now, backlog removal | | |

Then:

| Option | Description | Selected |
|--------|-------------|----------|
| One PR off main, then flip | Gate + guard removal to main as ci:/test: commits; maintainer adds the required check after merge | ✓ |
| Wait for the milestone merge | Phase 4 can't close on a working gate | |

| Option | Description | Selected |
|--------|-------------|----------|
| Milestone branch, cherry-pick | Author on gsd/v0.15.0-milestone, cherry-pick code-only commits to a branch off main, merge main back | ✓ |
| Worktree off main | Commit trail split across two branches | |

---

## Path-rule home

| Option | Description | Selected |
|--------|-------------|----------|
| Go helper under tools/ | Go table tests; needs setup-go | |
| Python beside pr_template_policy.py | python3 on the runner; needs a Python test | |
| Shell in the Taskfile | Awkward to test | |

**User's choice (free text):** "we already have python to run this policy? Why wouldn't we write a test for it in python?" So: Python, with a stdlib unittest.

| Option | Description | Selected |
|--------|-------------|----------|
| New scripts/fragment_policy.py | Own file + test_fragment_policy.py; pr_template_policy.py untouched | ✓ |
| Extend pr_template_policy.py | Roadmap's "one file, two edits" | |

---

## What counts as a fragment

| Option | Description | Selected |
|--------|-------------|----------|
| An ADDED file, not .gitkeep | changeType ADDED under .changes/unreleased/ | ✓ |
| Any touched file there | Modified/renamed counts too | |

| Option | Description | Selected |
|--------|-------------|----------|
| Check output only | ::error:: guidance, no PR comment, read-only permissions | ✓ |
| Sticky comment like siblings | issues: write + github-script | |

---

## Claude's Discretion

- Event types, how the changed-file list reaches the script (gh in-script vs per-run delimiter output), Taskfile target names, exact `::error::` wording, and where the new workflow sits in the shape-test populations.

## Deferred Ideas

- A unittest suite for `scripts/pr_template_policy.py` (backlog candidate).
