---
phase: 01-cross-compile-spike-goreleaser-release-migration
plan: 05
subsystem: infra
tags: [github-actions, release-verification, cosign, attestation, self-upgrade, taskfile]

# Dependency graph
requires:
  - phase: 01-01
    provides: "release:dry-run Taskfile target, REL05-EVIDENCE convention, dist/artifacts.json extra.Format==\"binary\" filtering idiom"
  - phase: 01-02
    provides: "activated archives:/checksum:/binary_signs:/sboms:/release: blocks in .goreleaser.yaml, including the corrected per-platform-distinct binary_signs.signature template this plan's verify:release-assets exercises"
  - phase: 01-04
    provides: "gh attestation verify as the post-migration attestation command, docs/RELEASE.md § a/§ b's verbatim verification instructions this plan's verify:release-assets reuses character-for-character"
  - phase: 01-06
    provides: "empirical proof (real CI run 31282287965) that binary_signs.signature/sboms.documents resolve to four distinct published names — the prerequisite this plan's Task 2 checkpoint cites as satisfied"
provides:
  - ".github/workflows/post-release-verify.yml — permanent, workflow_run-triggered (never release: [published]) verification workflow with a resolve-tag job (tag validation + TAG-EVIDENCE), a verify-supply-chain job, and a self-upgrade matrix job, all sharing the event-aware conclusion guard"
  - "Taskfile.yml verify:release-assets — REQUIRED/ALLOWED/UNCLASSIFIED asset classification, bounded asset-visibility retry, checksum coverage assertion, cosign verify-blob against docs/RELEASE.md § a's verbatim regex, gh attestation verify"
  - "Taskfile.yml verify:self-upgrade — semver-predecessor prior-release resolution (drafts dropped, stable-vs-prerelease policy stated), explicit codegraph upgrade \"$TAG\", byte-identity assertion against the re-downloaded published asset"
affects: ["01-05 Task 2 and Task 3 (this plan's own remaining tasks, explicitly NOT executed in this dispatch)"]

# Actuals (#2632)
actuals:
  tokens: 9500
  tasks: 1
  commits: 1

tech-stack:
  added: []
  patterns:
    - "workflow_run + event-aware if: guard (github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success') as the only safe way to pair a release-completion trigger with a retained workflow_dispatch re-verification path — the bare conclusion-only guard silently skips (reports green) under workflow_dispatch"
    - "A dedicated resolve-tag job that validates and republishes the verified value as a job output, rather than trusting workflow_run.head_branch directly in downstream jobs — the same 'resolve once, everyone consumes the output' shape as this phase's earlier goreleaser-shape tests"
    - "jq-based semver-predecessor resolution (drop drafts -> parse -> filter strictly-less-than -> stated stable-vs-prerelease policy -> select max) in place of gh release list's chronology-ordered default, when a Taskfile target needs a real prior-version selection rather than 'the previous list entry'"

key-files:
  created:
    - .github/workflows/post-release-verify.yml
  modified:
    - Taskfile.yml
    - CONTRIBUTING.md

key-decisions:
  - "binary_signs.signature's known Sigstore-bundle shape is verified via cosign verify-blob's --bundle flag against a re-downloaded .sigstore.json, using docs/RELEASE.md § a's --certificate-identity-regexp copied character-for-character rather than re-derived, so a future edit to either the doc or the workflow that silently diverges is at least byte-comparable."
  - "verify:self-upgrade's semver comparison for two prereleases sharing the identical major.minor.patch as $TAG falls back to a lexical string comparison of the prerelease identifier, a documented, accepted simplification: no full semver-precedence library exists in this shell/jq environment, and no release in this project's history has yet exercised that specific edge. Flagged in-line as a code comment, not silently assumed."
  - "gh and jq availability on the Namespace runner classes (namespace-profile-linux-amd64-4x8, namespace-profile-macos-6x14-tahoe) is ASSUMED, matching GitHub-hosted runner image parity, and asserted via an explicit precondition/early-exit check rather than trusted silently — this assumption is UNVERIFIED until the workflow's first real dispatch or workflow_run firing, which cannot happen from this worktree (see Known Limitations)."

patterns-established:
  - "Any future Taskfile target that needs 'the prior semver version below X, excluding drafts' resolves it via the drop-drafts -> parse -> filter-strictly-less-than -> stated-policy -> select-max jq pipeline this plan establishes in verify:self-upgrade, rather than gh release list's publication-chronology order."

requirements-completed: [REL-06, REL-07, REL-08]
# Closed against the REAL published v0.5.1, not a fixture. REL-06/REL-07 by
# the asset-set + single-checksums-writer criteria; REL-08 by cosign
# verify-blob, gh attestation verify, and a genuinely shipped v0.4.0 binary
# self-upgrading byte-identically on BOTH darwin/arm64 and linux/amd64.
# Evidence in "Task 3 — REL-08 Proven Against the Published Release".

duration: ~70min (Task 1) + release incident and recovery
completed: 2026-08-09
status: complete
---

# Phase 1 Plan 5: Automated Post-Release Verification Workflow Summary

**PLAN COMPLETE — all three tasks.** Task 1 built `post-release-verify.yml` plus the
`verify:release-assets` and `verify:self-upgrade` Taskfile targets, which re-prove
REL-06/REL-07/REL-08 against a re-downloaded PUBLISHED release on every future release. Task 2
(the one-way merge) was performed by the maintainer: release-please cut **v0.5.0**, whose pipeline
run **failed and published zero assets** on a Taskfile version-assertion bug, and then — after a
patch-forward fix — **v0.5.1**, which succeeded with all 17 expected assets. Task 3 proved every
REL-08 claim against v0.5.1's published assets: `cosign verify-blob` **Verified OK** under the
unchanged cert identity, `gh attestation verify` green with a firing negative control, and a
genuinely shipped v0.4.0 binary self-upgrading **byte-identically on both darwin/arm64 and
linux/amd64**. This plan's own verifiers then failed on that good release; three false negatives
were found and fixed, and the workflow re-dispatched green.

## Performance

- **Duration:** ~70 min for Task 1, plus the v0.5.0 release incident and its patch-forward recovery
- **Started:** 2026-08-08 (continuation-style dispatch; exact spawn time not captured)
- **Completed:** 2026-08-09
- **Tasks:** 3 of 3 completed (Task 2's blocking checkpoint resolved by the maintainer's merge)
- **Files modified:** 3 by the Task 1 dispatch (1 created, 2 modified), plus `Taskfile.yml` again in PRs #39 and #41 during recovery
- **Releases produced:** v0.5.0 (FAILED — 0 assets, left as a permanent prerelease per D-07) and v0.5.1 (SUCCESS — 17 assets)

## Accomplishments

- `.github/workflows/post-release-verify.yml` — new permanent workflow. Triggers on `workflow_run` (`workflows: ["release"]`, `types: [completed]`) plus `workflow_dispatch` with a `tag` input — never `release: [published]`, which fires before `release.yml` uploads any asset (review HIGH-1). Every job (`resolve-tag`, `verify-supply-chain`, `self-upgrade`) carries the identical event-aware guard `github.event_name != 'workflow_run' || github.event.workflow_run.conclusion == 'success'` (cycle-2 review MEDIUM/pi) — confirmed exactly 3 occurrences in the file, matching the 3 jobs.
- `resolve-tag` job resolves and VALIDATES the tag under verification (never trusts `head_branch` directly): requires `v[0-9]*` shape, verifies `refs/tags/$TAG` exists via the GitHub API, cross-checks the tag's peeled commit against `head_sha` on the `workflow_run` path, with an exactly-one-matching-tag-by-commit fallback when `head_branch` is empty or non-tag-shaped (cycle-2 review MEDIUM/Codex). Emits one `TAG-EVIDENCE tag=... tag_commit=... upstream_run=... upstream_head_sha=... source=...` line and publishes the validated value as `outputs.tag`, consumed by both downstream jobs via `needs.resolve-tag.outputs.tag` — `head_branch` appears nowhere outside `resolve-tag`.
- `GH_TOKEN: ${{ github.token }}` is set on every step that invokes a `task verify:*` target (2 of 2), and both new Taskfile targets declare a named, actionable `GH_TOKEN`-non-empty precondition (review HIGH-5).
- `Taskfile.yml`'s `verify:release-assets`: classifies published assets into REQUIRED (exact set equality, both directions — 4 raw binaries, 4 zips, 4 `.sigstore.json`, 4 `.spdx.json`, 1 checksums file per release) / ALLOWED (explicit, currently empty allowlist) / UNCLASSIFIED (hard failure, named) — never a fixed total count (review MEDIUM, Codex Plan-05: confirmed no literal `17` anywhere in the target). Polls `gh release view --json assets` with a bounded 10-attempt/6-second backoff before asserting anything (CDN/API visibility lag tolerance, never a substitute for the `workflow_run` ordering guarantee). Downloads all 8 payloads plus the checksums file, asserts exactly 8 lines and both-directions set equality against the downloads, excludes sidecars/self-reference from the checksums (D-12), then runs `cosign verify-blob` with `docs/RELEASE.md` § a's `--certificate-identity-regexp` copied character-for-character, and `gh attestation verify` per § b.
- `Taskfile.yml`'s `verify:self-upgrade`: resolves the PRIOR release by SEMANTIC-VERSION PREDECESSOR ORDER (never `gh`'s publication-chronology list order — cycle-2 review MEDIUM/Codex): drops drafts (named, not silently skipped), parses semver via `jq capture()` (unparseable tags named and skipped, never silently ignored), filters to strictly-less-than `$TAG`, applies a STATED prerelease policy (prefer the greatest stable predecessor, fall back to prerelease only if none exists — comment-documented as a decision), selects the maximum, and logs `PRIOR-RELEASE tag=... policy=... candidates=...`. Downloads that prior release's real binary for `$GOOS`/`$GOARCH`, runs `./codegraph upgrade "$TAG"` EXPLICITLY (never a bare `upgrade`, which would resolve GitHub's timing-sensitive "latest" endpoint — review MEDIUM, Codex Plan-05), and asserts the upgraded binary's sha256 equals the NEW release's independently re-downloaded published asset — byte identity, not exit-code success — plus that `codegraph version --json`'s `.version` reports `$TAG`.
- `CONTRIBUTING.md` documents both new targets and the new workflow next to the existing `release:dry-run`/`release:dry-run-signed` bullets.
- All of Task 1's automated `<verify>` steps pass locally: `task lint:actions` (actionlint, clean after one shellcheck-false-positive fix — see Deviations), `go test ./internal/upgrade/...` (0 failures), `task --list` lists both new targets with descriptions.
- **Prerequisite work completed per this dispatch's explicit instruction:** `task release:dry-run` re-run on this darwin host, post-01-04 landing. **Result: PASS.** `dist/artifacts.json` shows exactly 4 released `Binary` entries (`extra.Format=="binary"`), 4 `Archive` entries, 1 `codegraph_v0.4.0_checksums.txt`, and 4 `SBOM` entries — matching Task 2's merge prerequisite 3 exactly (four binaries, eight archives — 4 raw + 4 zip counted together as the 8 downloadable payloads —, one checksums file, and the sidecars). `goreleaser release` reported `release succeeded after 18s`; all four binaries independently confirmed correctly-typed via `file -b` (2 ELF x86-64/aarch64, 2 Mach-O x86_64/arm64). Toolchain: go 1.26.5 darwin/arm64, zig 0.16.0, syft 1.50.0, cosign v3.1.3 — matching the orchestrator's recorded local toolchain notes.

## Task Commits

1. **Task 1: Automated post-release verification — the claims re-prove themselves on every release (D-08)** — `30d145e` (feat)

**Task 2: Publish — merge the `feat(release):` PR (one-way)** — NOT EXECUTED. Returned as a blocking checkpoint below; see "Task 2 Checkpoint" for the prerequisite assessment.

**Task 3: Prove REL-08 against the real published release and record the evidence** — NOT EXECUTED. Blocked on Task 2.

## Files Created/Modified

- `.github/workflows/post-release-verify.yml` — new permanent workflow (280 lines): header comment naming `release-please.yml:57-92`/`release.yml:47-50` and the T-01-31/T-01-38/T-01-39/T-01-40 threat classes; `resolve-tag`, `verify-supply-chain`, `self-upgrade` jobs.
- `Taskfile.yml` — new `verify:release-assets` and `verify:self-upgrade` targets (~353 lines), inserted immediately after `check:goreleaser`.
- `CONTRIBUTING.md` — one new bullet documenting both targets and the new workflow.

## Decisions Made

See `key-decisions` in frontmatter. In addition:

- Chose `gh api repos/<repo>/commits/<ref>` (not manual tag-object peeling) to resolve a tag's underlying commit sha in `resolve-tag` — this endpoint resolves both lightweight and annotated tags to their target commit uniformly, avoiding a second API round-trip to peel an annotated tag object.
- `verify:self-upgrade` computes sha256 via a `sha256_of()` helper that prefers `sha256sum` (Linux) and falls back to `shasum -a 256` (darwin, which ships no `sha256sum` by default) — necessary because this target's matrix runs on both `namespace-profile-linux-amd64-4x8` and `namespace-profile-macos-6x14-tahoe`, unlike `verify:release-assets`, which only runs on the linux leg and can use `sha256sum -c` directly.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `task lint:actions` shellcheck false positive on a jq `--arg` variable inside a single-quoted filter**
- **Found during:** Task 1, first `task lint:actions` run after writing `post-release-verify.yml`
- **Issue:** `resolve-tag`'s fallback branch calls `gh api ... --jq --arg sha "${HEAD_SHA}" '[.[] | select(.commit.sha==$sha) | ...]'`. shellcheck's embedded-script scan flagged `SC2016` ("Expressions don't expand in single quotes") on the jq filter, because `$sha` inside single quotes looks like an un-expanded shell variable — it is actually a jq `--arg`-bound variable, correctly protected from shell expansion by the single quotes.
- **Fix:** Added a `# shellcheck disable=SC2016` directive (on its own line, per shellcheck's directive-parsing requirements — an earlier attempt appending explanatory prose to the same directive line broke shellcheck's own parser, visible as a secondary SC1072/SC1073 cascade) immediately above the flagged line, with a preceding comment explaining why the suppression is correct rather than papering over a real issue.
- **Files modified:** `.github/workflows/post-release-verify.yml`
- **Verification:** `task lint:actions` exits 0.
- **Committed in:** `30d145e` (Task 1 commit)

**2. [Rule 1 - Bug] Header comment and a Taskfile code comment reproduced literal grep-target substrings, contaminating their own acceptance checks**
- **Found during:** Task 1, while running the plan's own `rg`-based acceptance-criteria checks after the first draft
- **Issue:** (a) The workflow header comment originally spelled out the event-aware guard's exact literal text as an example, which meant `rg -o "<guard>" post-release-verify.yml | wc -l` returned 4 instead of 3 (the acceptance criterion requires the count to equal the number of jobs). (b) `verify:self-upgrade`'s code comment originally said "never a bare `codegraph upgrade`," which literally contains the bare-invocation substring the criterion "a bare upgrade with no version argument appears nowhere in the block" warns against, even though it appeared inside explanatory prose, not an actual invocation.
- **Fix:** (a) Reworded the header prose to describe the guard's shape without reproducing it character-for-character, pointing the reader to each job's own `if:` line instead. (b) Reworded the self-upgrade comment to describe the risk ("an argument-less invocation would instead resolve GitHub's...") without containing the bare phrase.
- **Files modified:** `.github/workflows/post-release-verify.yml`, `Taskfile.yml`
- **Verification:** Re-ran both `rg` checks from the plan's acceptance criteria; both now match the stated expectations exactly (guard count == 3; no bare-invocation substring in the self-upgrade block).
- **Committed in:** `30d145e` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — mechanical corrections to satisfy the plan's own stated acceptance criteria and CI lint gate, no scope creep).

## Known Limitations / Unverified Claims (recorded, not glossed over)

Several of Task 1's acceptance criteria are explicitly **behavioral** and require a real firing of `post-release-verify.yml` — which cannot happen from this worktree, because the workflow does not yet exist on the repository's default branch (the same `workflow_dispatch`-registration limitation plans 01-01 and 01-06 documented for their own canaries) and because Task 2's merge — the only thing that would land it on `main` and cause a real `release.yml` completion to fire it — was explicitly out of scope for this dispatch. These are recorded here as **NOT YET DEMONSTRATED**, not silently assumed passing:

- A `workflow_dispatch` run observed actually EXECUTING (not skipping) the event-aware guard fix.
- `resolve-tag` FAILING (not skipping, not defaulting to "latest") on an empty/branch-shaped `head_branch` or a nonexistent tag — the logic is written per the plan's spec and reasoned through by hand above, but not exercised against a real GitHub Actions event payload.
- The four self-upgrade prior-release-selection behavioral criteria (stable-after-RC, historical dispatch, drafts excluded, no-predecessor) — the jq-based algorithm is designed to satisfy all four (see the Accomplishments section's algorithm description), but none has been run against this repository's real release history from within the workflow.
- `gh` and `jq` availability on the Namespace runner classes this workflow targets — assumed by parity with GitHub-hosted runner images, asserted via an explicit precondition rather than trusted silently, but genuinely unverified until first real dispatch.
- The `verify:self-upgrade` prerelease lexical-comparison simplification (two prereleases sharing the identical `major.minor.patch` as `$TAG`) has no exercised edge case in this project's release history to validate against.

None of these gaps block Task 1's own `<verify>` (which is fully automated and passed — see Accomplishments) or its `<done>` criterion (a permanent, least-privilege workflow exists, committed, statically correct by every `rg`-based acceptance criterion checked above). They are gaps in **behavioral** proof that only a real release publication (Task 3, after Task 2) can close — which is exactly the ordering this plan's own task sequence encodes.

## Task 2 Checkpoint: Prerequisite Assessment

Task 2 (`checkpoint:decision`, `gate="blocking"`) was **NOT executed** per this dispatch's explicit scope. No `gh pr merge`, no `git tag`, no tag push, nothing published. Below is the prerequisite assessment the plan requires before a human selects `merge`.

| # | Prerequisite | Status | Evidence |
|---|---|---|---|
| 1 | Plan 01-01's canary recorded PASS (REL-05 on real Linux, both arches) | **MET** | 01-01-SUMMARY.md — canary run [31273571889](https://github.com/seanb4t/codegraph-go/actions/runs/31273571889), V1 first dispatch, both linux legs on real non-emulated hardware, `files=430 symbols=4281` both legs. |
| 2 | `go test ./...`, `task lint:actions`, `task check:goreleaser` green on the branch | **PARTIALLY MET — one flake to note** | `task lint:actions` and `task check:goreleaser` confirmed green locally (this dispatch, this commit). PR #35's most recent `test` check (run [31273571863](https://github.com/seanb4t/codegraph-go/actions/runs/31273571863), against current PR HEAD `0461333`) shows `FAILURE`: `test/wireoracle`'s `TestFrozenTranscriptsMatch/error-malformed-args` — `stderr must contain exactly one "codegraph: mcp-session" line, found 0`. This matches the SAME documented, pre-existing wire-oracle flake class prior plans in this phase have already logged as out-of-scope and load-dependent (01-02-SUMMARY.md's "Issues Encountered"; STATE.md's "Pending Todos" — "Wire oracle ... response ordering flake ... latent on main, re-run of the identical commit passed"). **I reproduced `go test ./test/wireoracle/...` in isolation on this darwin host and it passed clean** (19.3s, 0 failures), consistent with a load-dependent flake rather than a real regression from this phase's work — none of this plan's changes touch `test/wireoracle`, `internal/mcp`, or session-line code. **This is a flag for the maintainer, not a verified blocker**: PR #35's CI should be re-run (or the specific job re-dispatched) before merge to confirm green, since the literal instruction says "confirm before merging," not "assume it will pass." |
| 3 | `task release:dry-run` run on a darwin host since 01-04 landed, producing 4 binaries, 8 archives, 1 checksums file, sidecars | **MET (this dispatch)** | See Accomplishments — re-run on this darwin host just now: 4 `Binary`, 4 `Archive`, 1 checksums file, 4 `SBOM` entries; all four binaries `file -b`-confirmed correctly-typed. |
| 4 | PR title starts with `feat(release):` | **MET** | `gh pr view 35 --json title` returns `"feat(release): migrate to single-runner goreleaser release with zig cross-compilation"`. |
| 5 | Plan 01-06's sign-enabled snapshot leg recorded green, four distinct `.sigstore.json` names confirmed | **MET** | 01-06-SUMMARY.md + independently re-confirmed this dispatch: `gh run view 31282287965` shows `conclusion: "success"` for all 4 jobs including `sign-snapshot`, with `SIGN-EVIDENCE count=4 distinct=4` and `SBOM-EVIDENCE count=4 distinct=4`. |
| 6 | `.goreleaser.yaml` carries a live `release:` block with `replace_existing_artifacts: true` | **MET** | Confirmed by reading the file (this dispatch): `.goreleaser.yaml:296` — `replace_existing_artifacts: true`, non-commented, live. |

**Additional context for the maintainer, not a formal prerequisite:** PR #35 remains a DRAFT with a red `Issue link required` check (no v0.5.0 tracking issue exists yet) — per the orchestrator's own notes, this is being handled separately and is not this plan's concern, but it is also a blocker on the literal ability to merge via `gh pr merge` (which additionally is globally denied to agents regardless).

**My assessment:** every prerequisite this plan names is MET except prerequisite 2, which shows a single wire-oracle flake that matches a pre-existing, documented, load-dependent pattern unrelated to this phase's changes, and which I independently reproduced as passing in isolation. I am not selecting `merge` — that decision, and the judgment call on whether the flake needs a CI re-run first, belongs to the maintainer per the plan's own design (`gh pr merge` is globally denied to agents; a human performs the merge by hand).

## Issues Encountered

None beyond the two deviations documented above and the PR #35 `test` check flake noted in the Task 2 assessment table (not caused by this plan's changes; not fixed here since it is outside this plan's file scope and — per the executor scope-boundary rule — pre-existing, unrelated flakiness in `test/wireoracle` is out of scope for a plan that touches only `.github/workflows/post-release-verify.yml`, `Taskfile.yml`, and `CONTRIBUTING.md`).

## User Setup Required

None for Task 1. Task 2 requires the maintainer to review the prerequisite table above and perform the PR #35 merge by hand (agents are globally denied `gh pr merge`).

## Task 2 — Publish: EXECUTED (maintainer merge, 2026-08-08/09)

The maintainer merged PR #35. release-please parsed the `feat(release):` PR TITLE (squash-only,
`squash_merge_commit_title=PR_TITLE`), computed the MINOR bump v0.4.0 → v0.5.0, opened release PR
#38, and on its merge created the tag and Release. **D-06R held end to end: no human ran
`git tag`, no `Release-As:` footer forced a version.**

### The v0.5.0 incident — first execution of the `release:` pipe FAILED

`release.yml` run 31284774340 failed and **published zero assets**. This plan's own text named
the `release:` publish pipe as the single "UNEXERCISED, and unexercisable" surface; what actually
broke was one layer out — the Taskfile guard wrapping the invocation, on the same unrehearsable
path.

**Root cause (a gate that could never PASS).** `task release:goreleaser` aborted on its own
MAINT-03 assertion before `goreleaser release` ran:

```
resolved goreleaser reports version '2.17.1',
want 'github.com/goreleaser/goreleaser/v2 v2.17.1'
```

Task renders every `cmds:` string through Go `text/template` **before the shell sees it**, so the
literal `{{.Version}}` in `go list -f '{{.Version}}'` was consumed by *Task*, which has no
`.Version` variable. It rendered empty → `go list -f ''` → go's default module format. The
comparison could never be equal. Invisible outside Task (the same command in a plain shell prints
`v2.17.1`), and this target runs only from `release.yml` on a tag push.

**Blast radius contained:** the assertion precedes the release invocation, so nothing was built,
signed, or uploaded. No partial or corrupt artifacts — just an empty Release.

**Mitigation and recovery, D-07 respected throughout:**
- `v0.5.0` marked **prerelease** (a flag edit, not a deletion) so `/releases/latest` fell back to
  v0.4.0 — verified on BOTH paths `codegraph upgrade` uses: the API endpoint (→ v0.4.0, 14 assets)
  and the `releases/latest` HTML redirect (→ `302 .../tag/v0.4.0`). `codegraph upgrade` never
  stayed broken.
- Fixed forward in PR #39 (`fix(release):`) — `go list -m` + `awk '{print $2}'`, no braces for
  Task to eat. A **second latent bug** was caught on the other side of the comparison:
  goreleaser-action's prebuilt binary reports `2.17.1` while the from-source fallback reports
  `v2.17.1`, so normalizing only the pin would have gone green in CI and failed on every
  contributor machine. Both sides are now v-stripped; a stale `2.16.0` is still rejected, proven
  by negative control.
- The tag and release were **never deleted or re-pushed**. v0.5.0 remains an empty prerelease
  permanently, exactly as D-07 requires.

### v0.5.1 — the migrated pipeline succeeded

`release.yml` run 31285232766: **17 assets** — 4 raw binaries, 4 `.zip`, 4 `.sigstore.json`,
4 `.spdx.json`, 1 `codegraph_v0.5.1_checksums.txt`. The sign/SBOM name-collision fix held under
**real keyless OIDC**, not just plan 01-06's throwaway-key rehearsal.

## Task 3 — REL-08 Proven Against the Published Release

All claims verified against **re-downloaded published assets**, never a local `dist/` copy.
Verified twice and independently: once by hand by the orchestrator, and once by the automated
`post-release-verify` workflow (run 31285981504, all four jobs green).

**REL-08 (a) — cosign, under the UNCHANGED certificate identity:**

```
cosign verify-blob \
  --bundle codegraph_v0.5.1_linux_amd64.sigstore.json \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp '^https://github\.com/seanb4t/codegraph-go/\.github/workflows/release\.ya?ml@refs/tags/v[0-9][^[:space:]]*$' \
  codegraph_v0.5.1_linux_amd64
→ Verified OK
```

The regex is byte-identical to the pre-migration one. **The collapse from three jobs to one did
not move the cosign SAN (D-11a).**

**REL-08 (b) — provenance via the native attestor:**

`gh attestation verify codegraph_v0.5.1_linux_amd64 --repo seanb4t/codegraph-go` → exit 0,
`predicateType: https://slsa.dev/provenance/v1`. Negative control (a file with no attestation)
correctly exits 1 with HTTP 404, so the check demonstrably fires.

Note: verifying the **checksums file** 404s by design — `subject-checksums:` makes the binaries
listed *inside* it the attestation subjects; the file itself is never a subject. Plan 01-04's doc
rewrite records this correctly, closing the drift tracked in issue #14.

**REL-08 (c) — a genuinely shipped prior binary self-upgrades, both arches:**

```
SELF-UPGRADE-EVIDENCE goos=darwin goarch=arm64 prior_tag=v0.4.0
  prior_sha256=c05378829d9493716ee7c7fb460e05d3b34815d3061fb833ac40806f068a1c09
  upgraded_sha256=69325c30b8b5768dd226472ed31da8e30c1289b877a447fd59f808a6c997764d
  published_sha256=69325c30b8b5768dd226472ed31da8e30c1289b877a447fd59f808a6c997764d

SELF-UPGRADE-EVIDENCE goos=linux goarch=amd64 prior_tag=v0.4.0
  prior_sha256=655ab31c91ac8d416dba540da4a683ad66e6e689e5ebe8f0e8757b54111c9d2a
  upgraded_sha256=baafd6dcdeb9f9be76d3228d6160c7e39808f00939e5761300e40ddda36867fa
  published_sha256=baafd6dcdeb9f9be76d3228d6160c7e39808f00939e5761300e40ddda36867fa
```

A v0.4.0 binary built by the **old** pipeline upgraded itself to v0.5.1 built by the **new** one,
byte-identically. The three-part runtime contract (asset name, sidecar name, cert SAN) survives
the migration boundary intact.

**REL-06 / REL-07 — asset-set and single checksum writer:**

| Property | Observed |
|---|---|
| Payloads covered by checksums | 8 (4 raw + 4 `.zip`) |
| Sidecars in checksums | 0 |
| Self-reference | none |
| Duplicate entries | none |
| Published payloads uncovered | none |
| Total published | 17 = 8 payloads + 8 sidecars + 1 checksums |

**Tag binding:** `TAG-EVIDENCE tag=v0.5.1 tag_commit=4fe18c9a8b5bcb655fd384d4201ee04c96472af9
source=dispatch` — resolved and validated, never assumed.

### Three false negatives in this plan's own verifiers, found by their first real run

`post-release-verify` FAILED on v0.5.1 — a release whose claims all verify by hand. Fixed in
PR #41 (`ci(release):`, typed so it cuts no release):

1. **`verify:release-assets` never downloaded the cosign bundle.** It fetched checksums, raw and
   `.zip` payloads, then handed cosign a `.sigstore.json` path that was never fetched. Fixed by
   downloading it into a **separate** temp dir — `DL_DIR`'s listing is itself the set-equality
   assertion, so adding a sidecar there would have traded one failure for another.
2. **`verify:self-upgrade` judged "stable" by semver suffix alone.** `v0.5.0` has no suffix, so
   `select(.pre == "")` read it as stable and chose it as v0.5.1's predecessor — despite being
   API-flagged prerelease *because* it shipped empty. Now requires no suffix AND
   `.prerelease == false`.
3. **`verify:self-upgrade` did not skip zero-asset releases.** It already dropped drafts as
   "no downloadable asset"; the identical reasoning was never extended to empty releases.

Defects 2 and 3 were only reachable *because* v0.5.0 exists as an empty release — which D-07
guarantees is permanent, so they had to be handled rather than waited out.

After the fix, `post-release-verify` was **dispatched** against v0.5.1 (run 31285981504) and all
four jobs passed. That dispatch also exercised the event-aware guard cycle-2 review flagged: under
`workflow_dispatch`, `github.event.workflow_run` is null, so an unconditional conclusion test
would have made every job SKIP silently. The jobs ran — the guard is correct.

## Next Phase Readiness

**This plan is COMPLETE.** All three tasks executed; REL-06, REL-07 and REL-08 closed against a
real published release.

**Carry into Phase 2 — the SIGN-03 baseline moved:**
- Phase 2's un-notarized-darwin RED baseline must come from **v0.5.1**, NOT v0.5.0. v0.5.0 has
  zero assets and cannot serve as a baseline for anything. `codegraph_v0.5.1_darwin_arm64` and
  `codegraph_v0.5.1_darwin_amd64` are the deliberately un-notarized artifacts; do not replace them.
- `v0.5.0` remains a permanent empty prerelease. Any tooling that enumerates releases must
  tolerate it — `verify:self-upgrade` now does, and future release-walking code should follow that
  precedent rather than rediscovering it.

---
*Phase: 01-cross-compile-spike-goreleaser-release-migration*
*Completed: 2026-08-09 (all 3 tasks; REL-06/REL-07/REL-08 proven against published v0.5.1)*

## Self-Check: PASSED

- FOUND: `.github/workflows/post-release-verify.yml`
- FOUND: `Taskfile.yml` (contains `verify:release-assets:` and `verify:self-upgrade:` targets)
- FOUND: `CONTRIBUTING.md` (references both new targets)
- FOUND: `.planning/phases/01-cross-compile-spike-goreleaser-release-migration/01-05-SUMMARY.md`
- FOUND commit: `30d145e` (`git log --oneline` confirms)
