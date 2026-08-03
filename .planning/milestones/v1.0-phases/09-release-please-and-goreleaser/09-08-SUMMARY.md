---
phase: 09-release-please-and-goreleaser
plan: 08
status: COMPLETE
completed: 2026-08-01
subsystem: infra
tags: [release-please, cosign, slsa, sigstore, rel-02, first-live-run]

requires:
  - phase: 09-release-please-and-goreleaser
    provides: "09-01..09-06's release-please + GoReleaser pipeline; 09-07 SKIPPED, so this was the pipeline's first live run"
provides:
  - "v0.2.0 published — signed, SBOM'd, SLSA-attested, cut entirely by release-please with no human running git tag"
  - "REL-02 satisfied on BOTH halves, the second proven against a genuinely shipped v0.1.0 binary"
  - "Empirical confirmation that releaseWorkflowRefPattern is actor-independent — the argument used to justify skipping 09-07"
affects: []

tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []
---

# 09-08 — COMPLETE. v0.2.0 published and verified.

`v0.2.0` is live: signed, SBOM'd, SLSA-attested, and cut end to end by
release-please from Conventional Commits. **No human ran `git tag`** — the tag
and Release were created by `fzy-release-please[bot]`, which is REL-02's first
half stated literally.

## Deviation from the plan's task structure, stated not buried

The work happened, but not through the task sequence as written:

- **Task 1's decision gate** was not presented by an executor. The maintainer
  was shown the computed version, the changelog composition, and the full
  live evidence set *before* merging, then merged PR #2 by hand. The gate's
  substance — a human sees the number before it becomes permanent — was
  honoured; its mechanic was not.
- **Task 2's merge** was performed by the maintainer, not an executor. A
  standing global deny rule (`Bash(gh pr merge *)`) blocks agent merges, and it
  was not worked around via the equivalent `gh api .../merge -X PUT`. That deny
  is why the merge method got scrutinised at all — see "What the deny caught".
- **Task 3's verification** was run by the orchestrator, in full, and is
  recorded below.

## Acceptance criteria — evidence

| Criterion | Evidence |
|---|---|
| Version computed, not chosen | `0.2.0`, minor bump from `0.1.0`; manifest `"." : "0.1.0"` → `"0.2.0"` |
| No human `git tag` | Tag + Release author: `fzy-release-please[bot]` |
| No version forcing | `git log v0.1.0..main --format='%B' \| rg '^Release-As:'` → empty, **and** the same pattern matches a synthetic `Release-As: 9.9.9` line, proving the check non-vacuous |
| No sticky config key | `release-please-config.json` carries no `release-as` |
| Not a prerelease | `prerelease=false` |
| Body is release-please's changelog | `## [0.2.0](compare/v0.1.0...v0.2.0)` with `### Features` / `### Bug Fixes`; zero planning-only `docs(...)` entries |
| `release.yml` fired on a tag push | run `30675077940`, `event=push ref=v0.2.0` |
| All jobs green | 6 build legs + `sign, SBOM, and publish` + 4 SLSA jobs = **11/11 success** |
| Publish took the UPLOAD branch (D-04) | Release object created `00:14:36Z`; `release.yml` started `00:15:21Z` — the Release predated the run by **45s**, so `gh release create` could not have run and `upload --clobber` necessarily did |
| Asset contract | **20 assets** — 6 binaries, 6 `.sigstore.json`, 6 SBOMs, checksums, provenance |
| `releases/latest` | → `v0.2.0` |

## Task 3 verification — all five, run against real artifacts

1. **Asset contract** — 20 assets, all six platforms with sidecars.
2. **cosign** — `docs/RELEASE.md` §6(a) run verbatim with the full anchored
   identity regex: **`Verified OK`**.
3. **`TestVerifyReleaseE2E`** — **RAN, did not skip, PASSED** both subtests
   (`accepts_production_identity`, `rejects_wrong_identity`). This was its first
   execution against any real artifact, ever — the empirical cosign check that
   09-07's skip deferred to this plan.
4. **SLSA** — **PASSED**, builder pinned
   `generator_generic_slsa3.yml@refs/tags/v2.1.0`, at commit `cce95f3` (main's
   release commit).
5. **The real upgrade path** — a genuinely shipped `v0.1.0` binary (built
   2026-07-14, commit `803b4c9`, sha `773223fd…`), downloaded fresh from its own
   release into a scratch directory, ran its own `codegraph upgrade`:

   ```
   upgraded to v0.2.0
   ```

   It then reported `v0.2.0 (commit cce95f3, built 2026-08-01)`.

### The chain closes exactly

| | sha256 |
|---|---|
| Binary the upgrade installed | `a64c1549f012b065d077b89e63b683629e7a897a7a016b9e03d4ae8dea19c00b` |
| SLSA-attested subject | `a64c1549f012b065d077b89e63b683629e7a897a7a016b9e03d4ae8dea19c00b` |
| cosign `Verified OK` on | `a64c1549f012b065d077b89e63b683629e7a897a7a016b9e03d4ae8dea19c00b` |
| `TestVerifyReleaseE2E` passed against | `a64c1549f012b065d077b89e63b683629e7a897a7a016b9e03d4ae8dea19c00b` |

**The artifact a user receives is byte-identical to the artifact that was
attested.** That is the whole supply-chain claim, closed by measurement rather
than asserted.

## The upgrade test initially failed, and why that was not a defect

On the first attempt `codegraph upgrade` returned
`could not resolve the latest version from GitHub (API returned 404)`.

Cause: `internal/upgrade` sends no `Authorization` header anywhere —
`newLatestRedirectClient()` is a bare `http.Client`, and neither
`resolveLatestVersion` nor its API fallback attaches credentials. Correct design
for a tool distributed publicly; fatal against a private repository. The repo
was private at the time.

REL-02's second half was therefore **unprovable, not unmet**. The phase was held
open rather than closed on the four passing checks. When the repository was made
public the unauthenticated redirect went `404 → 302`, the API returned `v0.2.0`,
and the upgrade succeeded on the first attempt.

Worth keeping: a green result on four of five checks would have looked like a
satisfied requirement. The fifth is the only one that tests what a *user*
experiences.

## This retroactively vindicates the 09-07 skip

09-07 was skipped partly on the argument that `releaseWorkflowRefPattern` binds
repo slug + workflow path + tag ref and contains **no actor component**, so an
App-token-triggered signature must satisfy a binary that predates the App. That
was read off a regex and argued structurally.

It is now empirical. The `v0.1.0` binary hard-codes the identity constants from
*before* the pipeline was rewired to release-please, and it verified a signature
produced by an App-triggered run. The disposable rehearsal would have proven
exactly this and nothing more — at the cost of a permanent double-signing of the
`v0.2.0` tag name in the public Sigstore transparency log.

The risk accepted at the skip — that this became the pipeline's first live run —
did not materialise. Recorded as an outcome, not a vindication of the reasoning
in general: Phase 8's first live release *did* catch two bugs green CI missed.

## Defects found while verifying

- **Both verification guides were wrong.** `docs/RELEASE.md` §1(b) and
  `RELEASE-PROCEDURES.md` §6(b) instructed verifying
  `codegraph_<tag>_checksums.txt` against
  `codegraph_<tag>_checksums.txt.intoto.jsonl`. No such file is published and
  the checksums file is not an attested subject — provenance covers the six
  binaries. A user following the docs would get
  `FAILED: artifact hash does not match provenance subject` on a sound release
  and reasonably conclude the supply chain was compromised. Fixed in PR #6.
- **`release.yml`'s own comment was the source.** It described provenance as
  running "over this job's checksums file"; the file is the *transport* for the
  subject list. Corrected in PR #7 along with naming the bundle
  `codegraph_<tag>.intoto.jsonl` — releases through v0.2.0 keep
  `multiple.intoto.jsonl`.

## What the deny rule caught

`Bash(gh pr merge *)` halted the executor mid-sequence. In that pause the
executor's proposed `--merge` was caught: `main` had **zero merge commits**
across ~940, and a merge commit would have been the first. The plan's own
`blocking-human` gate had already been answered "proceed", and that single
approval would have carried through every downstream step.

Broad approvals at the start of an irreversible sequence age badly. A second ask
at the moment of consequence caught what the first could not.

## Requirement status

**REL-02 — SATISFIED, both halves.**

- Releases are cut by release-please from Conventional Commits, no human tagging.
- A real prior binary upgrades to the result through the unchanged identity.
