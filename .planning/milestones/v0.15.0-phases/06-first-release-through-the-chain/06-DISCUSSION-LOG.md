# Phase 6: First Release Through the Chain - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-10-17
**Phase:** 06-first-release-through-the-chain
**Areas discussed:** Sequencing & landing, #88 merge-readiness, Failure & evidence policy, Upgrade proof setup

---

## Sequencing & landing

**When do Phase 6's CONTEXT and PLAN land relative to the #88 merge?**

| Option | Description | Selected |
|--------|-------------|----------|
| Plan first, ride #88 | Plan on the milestone branch so CONTEXT and PLAN merge with #88. Only execution artifacts need the follow-up PR. | ✓ |
| Merge #88 now, plan after | All Phase 6 work on a fresh branch from main. The release PR waits while planning runs. | |
| You decide | Claude picks. | |

**Where do the execution artifacts and the milestone close go?**

| Option | Description | Selected |
|--------|-------------|----------|
| One planning-only PR | Phase 6 SUMMARY, VERIFICATION and UAT, the audit and the close/archive in one PR from main. | ✓ |
| Two PRs (v0.5.0 precedent) | A phase-complete PR (#42/#53 style), then an archive PR (#59 style). | |
| You decide | Claude picks. | |

**When does the milestone audit run?**

| Option | Description | Selected |
|--------|-------------|----------|
| After the release | One audit on the follow-up branch, covering all 29 requirements. | |
| Pre-merge pass, then final | Audit Phases 1-5 before #88 merges and fix gaps in #88, then a final audit after the release. | ✓ |
| You decide | Claude picks. | |

**Where does the #88 merge sit relative to execute-phase?**

| Option | Description | Selected |
|--------|-------------|----------|
| Precondition, outside execute | Audit, fixes, ready and merge happen first. Execution starts on a fresh branch from main with an entry check. | ✓ |
| First plan task (human checkpoint) | A blocking-human merge task, which means switching branches mid-execution. | |
| You decide | Claude picks. | |

---

## #88 merge-readiness

**Which findings get fixed inside #88 before it merges?** (multi-select)

| Option | Description | Selected |
|--------|-------------|----------|
| WR-01 stale release PR | Close changie/release in the zero-fragment path, via a Taskfile target. | ✓ |
| Runbook WR-02/03/04 | §7/§9 teardown framing, §12 tap remedy, §4 step 5 trigger claim. | ✓ |
| Hygiene WR-02 + WR-03 | Paginated, count-checked file listing (fail closed), plus stderr escaping. | ✓ |
| IN-05 token persistence | persist-credentials: false plus a push-time credential; re-points validateTagJobCredential. | ✓ |

**What vehicle carries the fixes?**

| Option | Description | Selected |
|--------|-------------|----------|
| /gsd-quick per cluster | Four quick tasks with atomic commits and RED-first evidence where a guard moves. Phase 5 stays closed. | ✓ |
| Phase 5 gap plan (05-09) | Reopen Phase 5. Its VERIFICATION and UAT go stale. | |
| Phase 6 pre-merge wave | A 06-01 hardening plan on the milestone branch. Contradicts the precondition decision. | |

**#88's 'Issue link required' check is red. What do we do?**

| Option | Description | Selected |
|--------|-------------|----------|
| Open + Close an issue | A milestone issue plus Closes #N (the #76 precedent). | ✓ |
| Exemption marker | A marker with a reason, which lands in the squash commit message. | |
| Leave it red | Not a required context. | |

**v0.15.0's notes are one Breaking bullet. Right content?**

| Option | Description | Selected |
|--------|-------------|----------|
| One bullet is right | Fragments describe binary-user-visible changes. The Phase 4/5 verdicts stand. | ✓ |
| Add a 'Changed' fragment | A hand-written note about the release-notes format change. | |
| Decide at the release PR | A fragment PR before the release-PR merge regenerates it. | |

---

## Failure & evidence policy

**Is v0.15.0 a real release or a disposable live proof?**

| Option | Description | Selected |
|--------|-------------|----------|
| Real, patch forward only | A stable release from tag push. Defects ship as v0.15.1 through the chain. | ✓ |
| Disposable until verified | Tear down and re-cut the same name. Needs a §7 carve-out and a human tag action. | |
| You decide | Claude picks. | |

**What may be re-run when a chain job fails?**

| Option | Description | Selected |
|--------|-------------|----------|
| Infra re-run once, else fix | An infrastructure cause visible in the log may be re-run once. Anything that would fail the same way again is fixed forward. | ✓ |
| Never re-run anything | Every red step is fixed through a PR. | |
| Re-run per job semantics | Idempotent jobs may be re-run freely. release.yml never. | |

**SHIP-02 byte-for-byte: verdict on a whitespace-only difference?**

| Option | Description | Selected |
|--------|-------------|----------|
| Strict; record any diff | Raw cmp. Stop for a ruling on any difference. | |
| Trailing-newline tolerant | Pre-decide that a trailing-newline difference passes. | |
| Content-equal is enough | Normalize whitespace on both sides and compare. | ✓ |

**Notes:** Claude defined the normalization in CONTEXT D-11: CRLF→LF, trailing whitespace stripped per line, and leading and trailing blank lines stripped. SHIP-02's wording is amended to match.

**How much of SC1-3 is the maintainer's?**

| Option | Description | Selected |
|--------|-------------|----------|
| Agent verifies, you sign off | The maintainer merges the release PR. The agent gathers the evidence. One blocking sign-off. | ✓ |
| Three hands-on checkpoints | As the ROADMAP words it. | |
| Merge only; no sign-off | Amend SC2/SC3's human labels. | |

---

## Upgrade proof setup

**Which installed version does the SHIP-03 upgrade start from?**

| Option | Description | Selected |
|--------|-------------|----------|
| v0.14.0, as written | Matches SHIP-03, so no amendment. | |
| v0.14.1 (current latest) | Amend SHIP-03. | |
| Both | Four runs. Amend SHIP-03. | |

**User's choice:** free text: "let's stop building a space shuttle here. just .. work."
**Notes:** Claude stopped asking and took the simplest path: v0.14.0 as written; a darwin/arm64 scratch-dir tarball and linux/amd64 via `docker --platform` on this Mac; no Homebrew path, no new infrastructure. The remark was recorded as the phase's governing principle (CONTEXT D-01). The remaining upgrade questions (environment, Homebrew path) and the final "explore more gray areas" prompt were not asked.

---

## Claude's Discretion

- The follow-up branch name and when it is cut.
- Grouping and order of the four quick tasks and the pre-merge audit.
- The milestone issue's title and body.
- The evidence summary's format and location.
- SHIP-03's environments and the content-equal normalization rule (under the "just work" directive).

## Deferred Ideas

- Upgrading through Homebrew (`brew upgrade`): not required by SHIP-03, so not done.
- Todos reviewed and not folded (the todo question came back empty): Go 1.27 toolchain pin, bench pinnedAt(), and the GH #85/#82/#80 reply.
