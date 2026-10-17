---
phase: 02-phase-close-fragment-capability
verified: 2026-09-26T18:30:00Z
status: passed
score: 4/4 must-haves verified
covered_files: [".planning/phases/02-phase-close-fragment-capability/02-01-PLAN.md", ".planning/phases/02-phase-close-fragment-capability/02-01-SUMMARY.md", ".planning/phases/02-phase-close-fragment-capability/02-02-PLAN.md", ".planning/phases/02-phase-close-fragment-capability/02-02-SUMMARY.md", ".planning/phases/02-phase-close-fragment-capability/02-03-PLAN.md", ".planning/phases/02-phase-close-fragment-capability/02-03-SUMMARY.md", ".planning/phases/02-phase-close-fragment-capability/02-04-PLAN.md", ".planning/phases/02-phase-close-fragment-capability/02-04-SUMMARY.md", ".planning/phases/02-phase-close-fragment-capability/02-05-PLAN.md", ".planning/phases/02-phase-close-fragment-capability/02-05-SUMMARY.md", ".planning/phases/02-phase-close-fragment-capability/02-06-PLAN.md", ".planning/phases/02-phase-close-fragment-capability/02-06-SUMMARY.md", ".planning/phases/02-phase-close-fragment-capability/02-CONTEXT.md", ".planning/phases/02-phase-close-fragment-capability/02-RESEARCH.md", ".planning/phases/02-phase-close-fragment-capability/02-VALIDATION.md", ".planning/phases/02-phase-close-fragment-capability/02-SECURITY.md", ".planning/phases/02-phase-close-fragment-capability/02-REVIEW.md", ".planning/phases/02-phase-close-fragment-capability/02-CAPABILITY-LOG.md", ".planning/phases/02-phase-close-fragment-capability/02-MUTATION-LOG.md", ".planning/phases/02-phase-close-fragment-capability/02-PATTERNS.md", ".planning/phases/02-phase-close-fragment-capability/02-DISCUSSION-LOG.md", ".planning/REQUIREMENTS.md", ".planning/ROADMAP.md", ".planning/STATE.md", ".changie.yaml", "Taskfile.yml", "internal/upgrade/changie_shape_test.go", ".gitignore", "CONTRIBUTING.md"]
covered_digest: "v1:sha256:6336bd6718c4b488b547a1bca4249c0badfd8bee11f26cc5acfbbd8006390db6"
behavior_unverified: 0
overrides_applied: 0
---

# Phase 2: Phase-Close Fragment Capability Verification Report

**Phase Goal:** Closing a GSD phase in this repository writes that phase's changelog fragments without a human remembering to: a private `role: "feature"` capability, `seanb4t/gsd-capability-changie`, owns a `changie-fragments` skill dispatched at `verify:post`, and codegraph-go installs it at project scope. The capability repository is separate and private — its commits happen outside this roadmap's commit trail — so what this phase delivers *here* is the capability installed, configured and documented; its first real firing is proven at Phase 3's close (`CAP-05`).
**Verified:** 2026-09-26
**Status:** passed
**Re-verification:** No — initial verification

All evidence below was independently reproduced in this verification session (live `gh`/`git` calls against the real GitHub repository, a live run of the capability's own `test/run.sh`, a live `task check:changie`, a live `render-hooks` query, live byte comparisons of installed artifacts against the published tag) — not copied from SUMMARY.md or REVIEW.md narration.

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth (Success Criterion) | Status | Evidence |
|---|---|---|---|
| 1 (CAP-01) | `seanb4t/gsd-capability-changie` exists as a private repo with README, LICENSE, tagged release; `capability.json` (`role: "feature"`, non-reserved id, `engines.gsd` pinned) installs cleanly on a scratch project | ✓ VERIFIED | `gh repo view seanb4t/gsd-capability-changie --json visibility,isPrivate` → `{"isPrivate":true,"visibility":"PRIVATE"}`. `git ls-remote --tags` shows `v0.1.0` peeling to `5700e60f...` and `v0.1.1` peeling to `35b674eb...`, matching the local checkout's `git tag -v` output exactly. `capability.json` on disk: `id: changie` (no `gsd-`/`gsd-core-`/`anthropic-` prefix), `role: feature`, `engines.gsd: ^1.14.0`. Live `bash test/run.sh` in the published checkout: `run.sh: 32 of 32 legs passed against a scratch project` (includes the install-on-scratch-project leg). `README.md` and `LICENSE` (MIT) present on disk. |
| 2 (CAP-02) | Manifest registers exactly one `verify:post` step (`ref.skill`, `onError: skip`) gated on `workflow.changie_fragments` (boolean, default true); dispatched true/absent, never false | ✓ VERIFIED | Live `gsd-tools loop render-hooks verify:post --raw` in codegraph-go: `activeHooks` contains exactly one entry with `capId: changie`, `ref.skill: changie-fragments`, `when: workflow.changie_fragments`, `onError: skip`. `capability.json`'s `config` block declares `workflow.changie_fragments` (boolean, default true) and `workflow.changie_command` (string, default "changie"). The capability's own 32-leg suite includes the true/false/absent dispatch-gate proof (`[write]`/gate legs in the same run that passed 32/32). |
| 3 (CAP-03) | Given a phase number, the skill writes one fragment per user-visible change through `CI=true changie new`, using only the host's declared kinds, commits under phase conventions; each of three skip cases (no changie, no `.changie.yaml`, no user-visible change) prints a reason and returns without prompting/blocking | ✓ VERIFIED | Live `bash test/run.sh`: 32/32 legs including `[write]`, `[skip:no-user-visible-change]`, `[undeclared-kind]`, `[argv-literal]` (a body with `$(...)`, backticks, quotes, `;`, `*` reaches the fragment literally and nothing runs), `[commit-scope]`, `[rollback]`, `[idempotent]`, `[write-no-pr]`, `[no-pr-required-host]`. A **real** `Skill(skill="gsd-changie-fragments", args="1 --pr 4242 --repo <R>")` dispatch (not a by-hand SKILL.md reading) is transcribed in 02-05-SUMMARY.md with the exact commits (`9ff51d0f`, `0d2410f1`) confirmed present in `git log`; the dispatch wrote one `Features` fragment for the user-visible change, wrote nothing for the planning-only change, and the commit's trailers were read back live: `Changie-Phase: 1`, `Changie-Summaries: 01-01,01-02`. D-02's reinterpretation of the "changie not on PATH" skip as "configured `workflow.changie_command` fails `--version`" is implemented exactly as specified — confirmed by reading `write-fragments.sh` lines 176-190 (`skip changie-unavailable "configured command '...' failed to run --version"`). |
| 4 (CAP-04) | codegraph-go has the capability installed at project scope from the HTTPS tag; `workflow.changie_fragments` set in `.planning/config.json`; `.gsd/` gitignored; `CONTRIBUTING.md` documents the per-clone install | ✓ VERIFIED | `.gsd/capabilities/changie/capability.json` is byte-identical (`diff` exit 0) to `git -C <capability-checkout> show v0.1.1:capability.json`. `.gsd-capabilities.json` records `"source": "https://github.com/seanb4t/gsd-capability-changie.git#v0.1.1"`. `.planning/config.json` has `"changie_fragments": true` and `"changie_command": "task changie --"`. `git status --porcelain` is clean (no untracked install output). `git check-ignore -v` attributes `.gsd-capabilities.json` to its own `.gitignore:51` line and `.gsd/capabilities/changie/capability.json` to the separate `.gitignore:46` line — no adjacency bug. `CONTRIBUTING.md` line 206 has `### Changelog fragments at phase close`, naming the one-time per-clone install command, the per-machine global-install step, the manual fallback, the `fragment-required` backstop, and the manual re-run syntax. The global materialization is also present and verified: `~/.claude/skills/gsd-changie-fragments/SKILL.md` is byte-identical to `git show v0.1.1:skills/changie-fragments/SKILL.md`, with a `.gsd-capability-skill` marker naming `changie`. |

**Score:** 4/4 truths verified (0 present-but-behavior-unverified)

### Requirement CAP-05 (Correctly Out of Scope)

CAP-05 ("the step is observed firing at a real phase close inside this milestone") is explicitly deferred to Phase 3's close per this phase's own scope boundary and REQUIREMENTS.md (`CAP-05 | Phase 3 | Pending`). This phase's own close produced a first, expected **skip-path** observation, not a firing: `.changes/unreleased/` is empty (only `.gitkeep`) and `git log` shows no new fragment commit for Phase 2's own close, consistent with 02-05-PLAN.md's stated expectation that Phase 2's own SUMMARYs (planning, contributor tooling, a private capability repo) are not user-visible per D-06, so `skip: no-user-visible-change` is the correct outcome here. This is recorded as a first observation of the skip path, not as CAP-05 evidence, exactly as 02-05-PLAN.md instructed.

### CHG-01 / CHG-04 Amendment (D-13) — Regression Check

Plan 02-06 amended Phase 1's `CHG-01`/`CHG-04` requirements in place (PR field made optional). Verified independently, not just via REQUIREMENTS.md's checkbox:
- `.changie.yaml` line 49-52: `key: PR`, `type: int`, `minInt: 1`, `optional: true`.
- `GOWORK=off go test ./internal/upgrade/ -run '^TestChangie'` — all 8 `TestChangie*` tests PASS, including `TestChangieConfigShape` (which pins the exact custom-key set including `optional`).
- Live `task check:changie` → `12 of 12 checks passed against a scratch copy (source tree byte-unchanged)`, including leg 5 ("changie new with no PR accepted"), legs 7-8 (PR=0 / PR=abc still refused), and leg 12 (`batch auto --dry-run` renders no-PR fragment without a link).
- `.planning/REQUIREMENTS.md` CHG-01/CHG-04 text matches the current `.changie.yaml` shape (optional PR, `minInt: 1` still enforced).
- No regression against Phase 1's original guarantees: `PR=0` and non-integer `PR` are still refused; only the "PR omitted entirely" case changed from refused to accepted, which is the amendment's explicit intent.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `gsd-capability-changie` (GitHub repo) | Private, README, LICENSE, tags | ✓ VERIFIED | Live `gh repo view` + `git ls-remote --tags`, both confirmed above |
| `capability.json` | `role: feature`, id `changie`, two config keys, one `verify:post` step | ✓ VERIFIED | Read directly from the installed project-scope copy, byte-identical to the tag |
| `skills/changie-fragments/SKILL.md` + `scripts/write-fragments.sh` | Judgment/deterministic split | ✓ VERIFIED | Present in the checkout; materialized globally, byte-identical to tag |
| `.gitignore` (`.gsd/`, `.gsd-capabilities.json`) | Both install-output paths ignored, non-overlapping | ✓ VERIFIED | `git check-ignore -v` attributes each path to its own distinct line |
| `.planning/config.json` | `workflow.changie_fragments`, `workflow.changie_command` | ✓ VERIFIED | Both present with the values the plans specify |
| `CONTRIBUTING.md` §Changelog fragments at phase close | Install, fallback, backstop, manual re-run | ✓ VERIFIED | Section present, content matches must-haves; `TestContributingReferencesRealTaskTargets` PASSES |
| `.changie.yaml` (PR optional) | `type: int`, `minInt: 1`, `optional: true` | ✓ VERIFIED | Read directly, matches |
| `internal/upgrade/changie_shape_test.go` | Re-pointed shape guard | ✓ VERIFIED | `TestChangieConfigShape` PASS live |
| `02-CAPABILITY-LOG.md` / `02-MUTATION-LOG.md` | RED/GREEN mutation transcripts (D-09) | ✓ VERIFIED | 1010 + 319 lines; both contain multiple RED and GREEN transcript sections |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| codegraph-go `.gsd/capabilities/changie/` | `gsd-capability-changie` tag v0.1.1 | HTTPS install spec | ✓ WIRED | `capability.json` byte-identical; `.gsd-capabilities.json` records the exact `#v0.1.1` source URL |
| `~/.claude/skills/gsd-changie-fragments/SKILL.md` | `verify:post` dispatch `Skill(gsd-changie-fragments)` | Global materialization | ✓ WIRED | File present, byte-identical to tag; render-hooks confirms the dispatch step exists and points at this skill name |
| `.planning/config.json` `workflow.changie_command` | `Taskfile.yml` `changie:` task | `"task changie --"` | ✓ WIRED | Value confirmed in config.json; `task changie -- --version` style invocation is what `write-fragments.sh`'s `changie-unavailable` probe runs, confirmed by reading the script |
| codegraph-go draft PR #88 | `write-fragments.sh`'s `gh pr list --head <branch>` lookup | Open-PR resolution | ✓ WIRED | `gh pr view 88` → `{"state":"OPEN","isDraft":true,"headRefName":"gsd/v0.15.0-milestone","baseRefName":"main"}`, live-confirmed still open |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Capability's own scratch-project proof suite | `bash test/run.sh` (in the published checkout) | `run.sh: 32 of 32 legs passed against a scratch project` | ✓ PASS |
| codegraph-go's changie guard | `task check:changie` | `check:changie: 12 of 12 checks passed against a scratch copy (source tree byte-unchanged)` | ✓ PASS |
| Go shape tests | `go test ./internal/upgrade/ -run '^TestChangie\|^TestContributingReferencesRealTaskTargets$'` | All PASS | ✓ PASS |
| Full build | `GOTOOLCHAIN=go1.26.6 go build ./...` | Exit 0 | ✓ PASS |
| `verify:post` hook registration | `gsd-tools loop render-hooks verify:post --raw` | Exactly one `changie` step, gated correctly | ✓ PASS |
| Draft milestone PR still open | `gh pr view 88 --json state,isDraft` | `OPEN`, draft | ✓ PASS |

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX` debt markers found in any file modified by this phase (codegraph-go side or the capability repository's committed source). The one `XXXXXX` match in `Taskfile.yml` is a pre-existing `mktemp` template pattern, not a debt marker — verified false positive by reading context.

**WR-01 (carried from code review, independently reproduced here):** `write-fragments.sh`'s `--pr` validation (`grep -Eq '^[1-9][0-9]*$'` without `-z`) anchors per line, not per whole string. Reproduced live: `printf '12\nDANGEROUS' | grep -Eq '^[1-9][0-9]*$'` prints `MATCHED`. This means an operator-typed, multi-line `--pr` argument (e.g. `--pr $'12\nDANGEROUS'`) is not rejected by this check. Scope is narrow: it only affects an explicit, hand-typed `--pr` value (the automatic `gh`-lookup path strips whitespace first, and changie's own `type: int` parser would still reject a genuinely malformed value at write time). This does not affect the automated `verify:post` dispatch path, which never passes a hand-typed `--pr`. **Classification: WARNING, non-blocking for CAP-01..04.** It is a real, reproduced gap in a defense-in-depth check, not a failure of any must-have truth or prohibition tested by this phase's plans — the phase's `prohibitions` about PR-number integrity are scoped to "no placeholder PR number" and "no raw gh output reaching a fragment," both of which hold. Fix requires capability `v0.1.2`, not yet approved; tracked as a follow-up, consistent with the code review's own disposition.

**CR-01 (carried from code review):** Independently not re-litigated here — the code review's own orchestrator addendum already reproduced this against the real `task` 3.52.0 binary with 8 injection payloads (0/8 executed) and downgraded it to INFO (test-coverage gap: the capability's `[argv-literal]` leg uses a `fake-task` stand-in that doesn't reproduce Task's real `CLI_ARGS`-through-a-shell quoting, but Task's own `shellQuote` behavior on `{{.CLI_ARGS}}` means the production path is safe). This verification's own read of the `Taskfile.yml` `changie:` task confirms it uses `{{.CLI_ARGS}}` as documented, consistent with that finding. Not re-blocking.

**IN-01 (carried from code review):** Cosmetic — stale `# Leg N/11` comments in `Taskfile.yml` vs. the current 12-leg reality. Confirmed present, does not affect behavior. INFO only.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| CAP-01 | 02-01, 02-02, 02-03 | Private repo, capability manifest, scratch install | ✓ SATISFIED | See Truth 1 |
| CAP-02 | 02-01 | Exactly one gated `verify:post` step | ✓ SATISFIED | See Truth 2 |
| CAP-03 | 02-01, 02-02, 02-05, 02-06 | Fragment-writing skill, skip cases, commit conventions | ✓ SATISFIED | See Truth 3 |
| CAP-04 | 02-03, 02-04, 02-05, 02-06 | codegraph-go project-scope install, config, docs | ✓ SATISFIED | See Truth 4 |
| CHG-01 (amended) | 02-06 (D-13) | `PR` custom field now optional | ✓ SATISFIED | See amendment regression check |
| CHG-04 (amended) | 02-06 (D-13) | No-PR fragment accepted; PR=0 still refused | ✓ SATISFIED | See amendment regression check |
| CAP-05 | (Phase 3) | First real firing at a real phase close | Correctly deferred | Not in this phase's scope; see "CAP-05 (Correctly Out of Scope)" above |

No orphaned requirements found: REQUIREMENTS.md's Phase 2 row maps exactly CAP-01..CAP-04, all claimed by the phase's plans (`requirements:` frontmatter across 02-01..02-06), plus the D-13 amendment to CHG-01/CHG-04 which is explicitly documented in 02-CONTEXT.md and 02-06-PLAN.md's `requirements:` list.

### Human Verification Required

None. All must-haves resolved to VERIFIED via live, independently-reproduced evidence (GitHub API calls, live test-suite runs, live byte comparisons, live Go test runs). No visual, real-time, or subjective-quality behavior is in scope for this phase.

### Gaps Summary

No gaps. All four in-scope Success Criteria (CAP-01..CAP-04) are independently verified against the live system, not just against SUMMARY.md narration. One non-blocking WARNING (WR-01) is carried forward from code review, independently reproduced, and correctly scoped as a follow-up for a future capability release (v0.1.2) rather than a phase-goal blocker — it does not touch the automated `verify:post` path this phase's goal is about. CAP-05 is correctly out of scope for this phase and is not claimed as complete anywhere in the phase's artifacts.

---

_Verified: 2026-09-26_
_Verifier: Claude (gsd-verifier)_
