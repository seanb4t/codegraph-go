# Phase 6: First Release Through the Chain - Context

**Gathered:** 2026-10-17
**Status:** Ready for planning

<domain>
## Phase Boundary

A real release ships through the chain Phase 5 built, with no new mechanism. The milestone PR (#88) merges. `release-pr.yml` opens the `chore(main): release vX.Y.Z` PR from the one unreleased fragment, and the maintainer merges it. `release-tag.yml` pushes the tag with the App token. The locked `release.yml` publishes with `.changes/vX.Y.Z.md` as the body. An installed v0.14.0 upgrades to the new release with verification on darwin/arm64 and linux/amd64. SEED-005 is annotated, and the capability tag is recorded (SHIP-01…04).

In scope:
- Pre-merge readiness of #88: review-finding fixes, the pre-merge milestone audit, the issue link, and marking #88 ready.
- The live run and its evidence.
- The final audit and milestone close, landed in one planning-only follow-up PR.

Out of scope:
- Any new workflow, guard, test or infrastructure built for this phase.
- The Homebrew upgrade path.
- Porting to other repositories (PORT-01, v2).

</domain>

<decisions>
## Implementation Decisions

### Governing principle
- **D-01:** **Just make it work. No space shuttle.** The maintainer said this during this discussion, repeating Phase 5 D-01. Phase 6 runs the chain as built and records what happened. The planner adds no new workflow, guard, test, harness or verification infrastructure. Each step uses the plainest tool that answers it: `gh`, `curl`, `docker`, `cmp`. Plans stay few and short.

### Sequencing and landing
- **D-02:** **Plan first; CONTEXT and PLAN ride #88.** `/gsd-plan-phase 6` runs on `gsd/v0.15.0-milestone` before the merge, so this phase's plan is reviewed inside the milestone PR.
- **D-03:** **The #88 merge is a precondition, outside `/gsd-execute-phase 6`.** Pre-merge work runs on the milestone branch, in this order:
  1. The four `/gsd-quick` fix clusters (D-07).
  2. The pre-merge audit (D-05), plus a fix for anything it finds.
  3. The milestone issue and `Closes #N` (D-08).
  4. `gh pr ready 88`, never `/gsd-ship`'s create step (STATE).
  5. The maintainer squash-merges #88.

  Execution then starts on a **fresh follow-up branch cut from `main`**. Executor commits never land on the squash-merged, dead milestone branch. Skip execute-phase's `handle_branching`, as the STATE carry-over already says.
- **D-04:** **One planning-only follow-up PR carries everything after the merge.** That is Phase 6's SUMMARY, VERIFICATION and UAT, the final milestone audit, and the milestone close/archive. `fragment-required` passes a `.planning/`-only diff with no marker, and `release-pr.yml` no-ops because no fragments remain after the release.
- **D-05:** **The milestone audit runs twice.** A pre-merge pass audits Phases 1–5, with Phase 6 showing as pending, so cross-phase gaps get fixed inside #88 before it starts the chain. A final pass runs after Phase 6 verifies, on the follow-up branch, covering all 29 requirements.

### #88 merge-readiness
- **D-06:** **v0.15.0's notes are the one existing `Breaking` fragment** (the `query`/`unlock` removal). No fragment is added. The capability's no-user-visible-change verdicts for Phases 4 and 5 stand.
- **D-07:** **These findings are fixed inside #88 before it merges, as four `/gsd-quick` tasks:**
  1. **WR-01 (2026-10-14 review):** in the zero-fragment path, `release-pr` closes any open `changie/release` PR. The close goes through a Taskfile target, so `TestWorkflowRunBodiesInvokeTask` holds.
  2. **Runbook WR-02/03/04 (2026-10-16 review), in `docs/RELEASE-PROCEDURES.md`:**
     - §7/§9: drop the "disposable live proof" framing (wording set by D-09).
     - §12: replace the tap-failure remedy that re-runs the full `goreleaser release`.
     - §4 step 5: correct the trigger claim. The milestone merge also changes `CHANGELOG.md` and fires `release-tag.yml`, which skips with success.
  3. **Hygiene WR-02 + WR-03 (2026-10-14 review):**
     - `require-issue-link.yml` and `pr_template_policy.is_housekeeping` list files through the paginated REST call, cross-check the count against `changed_files`, and fail closed on a mismatch.
     - The exemption reason is escaped before it reaches stderr, or a multi-line reason is refused, as `fragment_policy` already does.
  4. **IN-05:** the tag job checks out with `persist-credentials: false`, and `release:tag` gets the App token only at push time, through `env:`. This intentionally changes `validateTagJobCredential` and the `release-tag.yml` comment. Re-point them RED-first (rule `84d1gfpywd`, Go RED evidence per `x1cjy9vyhq`).

  Each fix is recorded as `fixed` with its commit in `05-REVIEW-DISPOSITION.md`. That includes the seven 2026-10-14 findings the ID reuse dropped (`jf9tgc0bb5`), restored through the tool's disposition step, never by hand. Findings this list does not name, including Phase 4's open ones, are whatever the pre-merge audit (D-05) makes of them.
- **D-08:** **Open one milestone issue and put `Closes #N` in #88's body,** following the #76 precedent (`Closes #17`, `Closes #13`). Creating the issue is outward-facing, so the maintainer authorizes it at that step. #88's body becomes the squash commit message on `main` (`squash_merge_commit_message: PR_BODY`), so it must not contain `[skip ci]` (rule `f18zrdsgx5`).

### Failure and evidence policy
- **D-09:** **v0.15.0 is a real, stable release from the moment the tag is pushed. Recovery is patch-forward only.** Any defect ships as the next patch through a `Fixes` fragment (if shipped code changes) and the chain. Nothing is torn down, deleted or re-tagged. The runbook's §7 teardown applies to no published `vX.Y.Z`. — **Reversibility:** one-way — a pushed `vX.Y.Z` tag and its published Release are resolved as latest by every installed `codegraph upgrade` at once, and §7 forbids reusing a stable tag name.
- **D-10:** **Re-runs: infrastructure failures once; everything else is fixed.** A failed chain job may be re-run once if its log shows an infrastructure cause, with the cause recorded. Examples: runner loss, network, a GitHub 5xx, an Apple notary timeout. A failure that would fail the same way again is never re-run: config, code, a guard, a wrong key. It is fixed forward through a PR. This reconciles the ROADMAP's "no re-run to make a check green" with Phase 5 D-06's "a failed tag job is re-run like any other job".
- **D-11:** **SHIP-02's body check is content-equal, not byte-for-byte.** Normalize both sides the same way: CRLF→LF, strip trailing whitespace on each line, and strip leading and trailing blank lines. After that the Release body (`gh api repos/seanb4t/codegraph-go/releases/tags/vX.Y.Z --jq .body`) must equal `.changes/vX.Y.Z.md`. Any other difference is a defect, handled under D-09. The planner amends SHIP-02 in `REQUIREMENTS.md` and ROADMAP Phase 6 success criterion 2 to this wording, in their existing shapes. Phase 5 D-06 is the precedent.
- **D-12:** **The agent verifies; the maintainer signs off once.**
  - The maintainer reviews the rendered notes and version and merges the release PR (SC1). That merge is outward-facing.
  - The agent gathers and checks everything else, then presents one evidence summary for a single blocking sign-off. That sign-off is SC2 and SC3's human checkpoint, so the ROADMAP needs no amendment.
  - App authorship is shown by who pushed the tag, because `release:tag` makes a lightweight tag with no tagger. The evidence is the `release-tag` run that logged `pushed vX.Y.Z at <sha>`, plus the `release.yml` tag-push run whose `triggering_actor` is the App's `[bot]` login, not `seanb4t` and not `github-actions[bot]`.
  - The other checks are `changie latest` on `main`, a green `post-release-verify`, and the upgrade runs (D-13).

### Upgrade proof (SHIP-03)
- **D-13:** **The upgrade starts from v0.14.0, as SHIP-03 says, so no amendment is needed.** Both runs are on this Mac, use the published v0.14.0 tarball, and never touch the Homebrew install. `upgrade.Run` refuses a Homebrew-managed target before any network call (`internal/upgrade/brew.go`).
  - **darwin/arm64:** extract into a scratch dir, run `./codegraph upgrade`, then confirm `./codegraph --version` reports the new release.
  - **linux/amd64:** run the same steps in `docker run --platform linux/amd64` (OrbStack, Rosetta). If emulation gets in the way, fall back to a Namespace `devbox`. Do not build anything new for it.
  - Verification runs inside the binary (`sigstore-go` against the unchanged cosign identity) and needs only network access.

### SHIP-04
- **D-14:** Annotate SEED-005 with the verified trigger (this release's version and Release URL). Fill values only in shapes the seed file already has, using the tool's verb if one exists (generated-files rule). Record the capability tag this release ran with. `CONTRIBUTING.md` names `gsd-capability-changie.git#v0.1.2`, and the executor confirms the installed one rather than assuming.

### Claude's Discretion
- The follow-up branch name, and whether it is cut right after the #88 merge or after the release-PR merge. Either way it is before the first executor commit.
- Grouping of the four quick tasks, and the order of quick tasks vs. the pre-merge audit, within D-03's sequence.
- The milestone issue's title and body.
- The evidence summary's format, and where the transcripts live (a phase artifact such as `06-VERIFICATION.md`).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements and roadmap
- `.planning/REQUIREMENTS.md`: SHIP-01…04 (SHIP-02 amended per D-11); CAP-04 (capability install tag)
- `.planning/ROADMAP.md` §"Phase 6: First Release Through the Chain": success criteria and the notes on sequencing, no human `git tag`, and fix-forward
- `.planning/ROADMAP.md` §"v0.15.0 — Changie Release Management": locked contracts that bind every phase
- `.planning/seeds/SEED-005-changie-template-for-other-repos.md`: the trigger SHIP-04 annotates

### The chain being exercised
- `.github/workflows/release-pr.yml`: batch and open/update the release PR with the App token (WR-01 fix lands here)
- `.github/workflows/release-tag.yml`: `CHANGELOG.md` path trigger, App-token tag push (IN-05 fix lands here)
- `.github/workflows/release.yml`: locked; publishes on tag push
- `.github/workflows/post-release-verify.yml`: SC3's green check
- `Taskfile.yml`: `release:batch`, `release:tag`, `release:goreleaser`
- `internal/upgrade/upgrade.go`, `internal/upgrade/brew.go`, `internal/upgrade/verify.go`: the upgrade flow, the Homebrew refusal, and the cosign identity

### Findings fixed pre-merge (D-07)
- `git show f05db688:.planning/phases/05-release-chain-release-please-retirement/05-REVIEW.md`: WR-01, WR-02, WR-03 and IN-05 of the 2026-10-14 review (dropped from the ledger by ID reuse)
- `.planning/phases/05-release-chain-release-please-retirement/05-REVIEW.md`: runbook WR-02, WR-03 and WR-04 of the 2026-10-16 review
- `.planning/phases/05-release-chain-release-please-retirement/05-REVIEW-DISPOSITION.md`: the ledger the fixes are recorded in
- `scripts/pr_template_policy.py`, `.github/workflows/require-issue-link.yml`, `scripts/fragment_policy.py` (the reference implementation for pagination and `escape_data`)
- `docs/RELEASE-PROCEDURES.md` §4, §6, §7, §9, §12

### Prior phase decisions
- `.planning/phases/05-release-chain-release-please-retirement/05-CONTEXT.md`: D-01 (documented pattern, no space shuttle), D-03 (App token for the PR), D-04 (ready, never draft), D-06 (path trigger, skip-if-exists), D-14 (we don't test GoReleaser's rendering)
- `.planning/phases/05-release-chain-release-please-retirement/05-UAT.md` §Deferred Follow-Ups: tests 3 and 4 are this phase's live observations

### Project rules
- engram rules `84d1gfpywd` (RED-first, positive assertion), `eyva2svrcz` (never test third-party behavior), `x1cjy9vyhq` (Go RED evidence), `f18zrdsgx5` (never `[ci skip]`), `xmz3xknbj0` (commit signing)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `scripts/fragment_policy.py`: `fetch_changed_files` (paginated, count cross-checked, fails closed) and `escape_data` are the pattern the hygiene fixes copy.
- `release:tag`: `changie latest`, then `git ls-remote --exit-code` (exit 2 means absent, so tag; exit 0 means skip), then a lightweight `git tag` and a single-ref push.

### Established Patterns
- Every CI `run:` body is a `task` invocation (`TestWorkflowRunBodiesInvokeTask`).
- The two hygiene allowlists are a mirrored pair with a parity test. Edit both together.
- `protect-main` (ruleset 20157557) is squash-only, with 9 required contexts and 0 approvals. `Issue link required` is not required.

### Integration Points
- The live state at discussion time:
  - #88 is a draft and 231 commits ahead of `main`, with all 9 required checks green and `Issue link required` red.
  - No release-please PR is open, so Phase 5's orphan-PR note is moot.
  - The latest release is v0.14.1.
  - `.changes/unreleased/` holds one `Breaking` fragment.
- The maintainer's own `codegraph` is the Homebrew v0.14.1 install, so it is not usable for SHIP-03.
- Local tools: OrbStack Docker (`linux/aarch64`, amd64 via `--platform`) and the `devbox` CLI.

</code_context>

<specifics>
## Specific Ideas

- "let's stop building a space shuttle here. just .. work.": the maintainer, at the upgrade-proof question. This governs the whole phase (D-01). Simplest path, no amendments where the requirement already fits, no new infrastructure.

</specifics>

<deferred>
## Deferred Ideas

- Upgrading through Homebrew (`brew upgrade` after the tap update) is not required by SHIP-03 and is not done in this phase.

### Reviewed Todos (not folded)
- "Pin the Go toolchain so Go 1.27 can build codegraph-go" (build, score 0.9): local-build ergonomics. CI and the release build use `go.mod`'s 1.26.9, so this does not affect the cut.
- "tools/bench/runner pinnedAt() validates a checkout by git rev-parse HEAD alone" (bench, 0.6): unrelated.
- "Reply on GH #85 / #82 / #80 with the reshaped contract" (server-mode, 0.6): unrelated.

</deferred>

---

*Phase: 06-first-release-through-the-chain*
*Context gathered: 2026-10-17*
